# Extension Runtime

The gert v2 extension runtime defines how external packages contribute tools, schema fields, governance policies, and input providers to a running gert host without requiring changes to the host binary. Extensions run as child processes and communicate with the host over JSON-RPC 2.0, governed by an explicit capability grant model. Every operation an extension may perform is gated by a named capability that the host evaluates before launching any process.

## Architecture Overview

[DIAGRAM: Extension lifecycle — host discovers manifests → evaluates compatibility → launches child process → performs versioned handshake → extension registers contributions → host dispatches invocations → graceful shutdown]

1. The gert host discovers extension manifests on startup.
2. The host evaluates compatibility and capability policy before launching any process.
3. Each accepted extension is started as a child process; its executable communicates with the host over its declared transport.
4. The host and extension perform a versioned handshake. The host sends the set of capabilities it is willing to grant; the extension confirms receipt.
5. The extension registers its contributions (tools, schema fields, policies, providers).
6. For the remainder of the run the host dispatches invocations to the extension and enforces all capability constraints.
7. At run completion the host sends a graceful shutdown signal.

## Capability Taxonomy

| Capability name | Allows | Excludes |
|----------------|--------|---------|
| `capability/tool-registration` | Register new tool definitions at runtime via `contributions/list`. | Does not allow invoking other tools or modifying existing tool definitions. |
| `capability/schema-extension` | Contribute namespaced additional fields to the runbook schema (e.g. `x-acme.*` annotations). | Cannot modify or remove existing schema fields; cannot alter core validation rules. |
| `capability/event-subscription` | Subscribe to host runtime events (step-started, step-completed, run-ended, etc.) via the `events/subscribe` method. | Cannot emit synthetic events into the host event bus. |
| `capability/policy-contribution` | Register additional governance policy rules evaluated during pre-flight and approval checks. | Cannot override or disable built-in governance rules; cannot grant capabilities to other extensions. |
| `capability/provider-registration` | Register new input providers that handle `from:` prefix bindings. | Cannot intercept resolution requests destined for another registered provider. |
| `capability/file-read` | Read files within the path scope declared in the manifest (`file-read-paths`). | Cannot read files outside declared paths; cannot write or delete files. |
| `capability/file-write` | Write or create files within the path scope declared in the manifest (`file-write-paths`). | Cannot write outside declared paths; does not imply read access. |
| `capability/network` | Make outbound network calls to the hosts declared in the manifest (`network-hosts`). | Cannot connect to undeclared hosts; cannot listen for inbound connections. |
| `capability/env-read` | Read environment variables whose names match the declared `env-read-patterns` list. | Cannot read undeclared variables; cannot write or unset environment variables. |
| `capability/run-state-read` | Read current run state (step status, captured variables, trace events). | Cannot modify run state; does not imply the ability to emit output. |

The capability string format is `capability/<name>`. Future versions may introduce scoped variants using path notation, e.g. `capability/file-read:logs/**`.

## Extension Manifest Format

Each extension ships a `gert-extension.yaml` file alongside its executable.

```yaml
# gert-extension.yaml — Extension Manifest
apiVersion: extension/v2

meta:
  name: acme.incident-pack          # Reverse-DNS namespaced identifier
  version: "1.2.0"                  # Semver
  description: "Acme incident resolution tools and providers"
  author: "Acme Platform Team <platform@acme.example>"

compatibility:
  api-version: ">=2.0.0 <3.0.0"    # Host API version range (semver)
  protocol-version: "2.0"           # Extension protocol version

entry-point:
  executable: "./bin/acme-extension"
  args: ["--config", "acme.yaml"]

transport: stdio-jsonrpc             # stdio-jsonrpc | grpc | mcp

capabilities:
  - capability/tool-registration
  - capability/provider-registration
  - capability/schema-extension
  - capability/network
  - capability/file-read

# Required when capability/file-read is declared:
file-read-paths:
  - "./runbooks/**"
  - "./data/**"

# Required when capability/network is declared:
network-hosts:
  - "api.pagerduty.com"
  - "*.acme.example"

# Required when capability/env-read is declared:
env-read-patterns:
  - "ACME_API_*"
  - "PD_TOKEN"

platform:                            # Optional OS/arch constraints
  os: ["linux", "darwin", "windows"]
  arch: ["amd64", "arm64"]
```

Manifest validation is performed by the host before the process is started. An extension with an unsatisfied `compatibility.api-version` range is rejected before any subprocess is created.

## Extension Discovery

The host discovers extensions using the following resolution order (later entries take precedence):

1. **Built-in extensions**: compiled into the host binary; always available; no manifest required.
2. **Workspace config**: extensions declared in `.gert/extensions.yaml` at the workspace root.
3. **Runbook declaration**: extensions listed in the runbook's top-level `extensions:` field.
4. **CLI override**: `--extension <path>` flags provided at invocation time.

**Workspace config format:**

```yaml
# .gert/extensions.yaml
extensions:
  - path: "./extensions/acme-pack"   # directory containing gert-extension.yaml
  - path: "/opt/gert-extensions/audit-pack"
```

**Runbook `extensions:` field:**

```yaml
apiVersion: runbook/v2
meta:
  name: incident-triage
extensions:
  - path: "./extensions/acme-pack"
```

When the same extension appears in multiple discovery sources, the host deduplicates by `meta.name`. If two distinct extensions share the same name, the higher-precedence one wins; the lower one is logged as a warning and skipped.

## Handshake and Lifecycle Protocol

All messages use JSON-RPC 2.0. The host is always the initiator; the extension never sends unsolicited requests.

### extension/initialize (host → extension)

Sent immediately after the extension process starts.

Request params:

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "extension/initialize",
  "params": {
    "host": {
      "name": "gert",
      "version": "2.0.0",
      "apiVersion": "2.0"
    },
    "protocolVersion": "2.0",
    "grantedCapabilities": [
      "capability/tool-registration",
      "capability/network"
    ],
    "runContext": {
      "runId": "run-abc123",
      "mode": "real"
    }
  }
}
```

Success response:

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "extension": {
      "name": "acme.incident-pack",
      "version": "1.2.0"
    },
    "protocolVersion": "2.0",
    "acknowledgedCapabilities": [
      "capability/tool-registration",
      "capability/network"
    ]
  }
}
```

Error codes:
- `-32001`: Protocol version not supported by extension.
- `-32002`: Required capability not granted (extension considers the capability non-negotiable).
- `-32003`: Host API version outside extension compatibility range.

### contributions/list (host → extension)

Sent after a successful `extension/initialized` notification. The extension returns its contribution manifest filtered to the granted capability set.

```json
{
  "jsonrpc": "2.0",
  "id": "2",
  "method": "contributions/list",
  "params": {}
}
```

Response:

```json
{
  "jsonrpc": "2.0",
  "id": "2",
  "result": {
    "tools": [ { "name": "acme.pd-lookup", ... } ],
    "providers": [ { "name": "pd", "prefixes": ["pd."] } ],
    "schemaExtensions": [],
    "policyContributions": []
  }
}
```

### extension/ping (host → extension)

Periodic health check. The extension MUST respond within the host-configured ping timeout (default 5 seconds). A non-response triggers a crash-recovery cycle.

```json
{ "jsonrpc": "2.0", "id": "42", "method": "extension/ping", "params": {} }
```

Response:

```json
{ "jsonrpc": "2.0", "id": "42", "result": { "status": "ok" } }
```

### extension/shutdown (host → extension)

Graceful termination. The extension SHOULD drain in-flight requests and release resources within the shutdown timeout (default 10 seconds).

```json
{ "jsonrpc": "2.0", "id": "99", "method": "extension/shutdown", "params": {} }
```

Response (best-effort):

```json
{ "jsonrpc": "2.0", "id": "99", "result": { "status": "ok" } }
```

If the extension does not exit within the shutdown timeout, the host sends SIGTERM followed by SIGKILL after a grace period.

### Lifecycle State Machine

1. **discovered**: manifest found and parsed; no process yet.
2. **verified**: manifest validated; capabilities evaluated against policy.
3. **starting**: executable launched; stdin/stdout pipes attached.
4. **initializing**: `extension/initialize` sent; awaiting response.
5. **ready**: `contributions/list` completed; accepting invocations.
6. **draining**: `extension/shutdown` sent; waiting for in-flight completions.
7. **stopped**: process has exited cleanly.
8. **crashed**: process exited unexpectedly.

## Versioning and Compatibility

The host maintains a monotonically increasing API version string (semver). The extension manifest declares a semver range in `compatibility.api-version`. Evaluated before any process is started.

- If the host API version falls outside the declared range, the extension is rejected with a structured error logged to the run trace. The run continues with remaining extensions; a warning is surfaced to the operator.
- During `extension/initialize`, if the extension does not support the sent `protocolVersion` it MUST return error code `-32001`.
- If an extension requests a capability that the host version does not implement, the capability is silently absent from `grantedCapabilities`. The extension SHOULD treat absent non-required capabilities as optional degradation rather than a fatal error.

Host API versions are tracked in `specs/api-versions.md`. Each new capability addition = minor version bump; breaking changes to existing method contracts = major version bump.

## Sandboxing and Process Isolation

### Process Isolation

Extensions always run as child processes. Never loaded as shared libraries or executed in-process. The host:
- Sets a restricted environment: only env vars matching `env-read-patterns` from the manifest are forwarded.
- Attaches the extension's stderr to a structured log collector; raw stderr lines stored in the run trace under `extension.<name>.stderr`.
- Does not forward the host process's file descriptor table beyond stdin/stdout/stderr.

### Capability Enforcement

Every RPC method the extension may call is guarded by a capability check. The host maintains a granted-capability set per extension process. Before dispatching any extension-initiated request, the host verifies the required capability is in the granted set. Violations return:

```json
{
  "jsonrpc": "2.0",
  "id": "...",
  "error": {
    "code": -32010,
    "message": "capability not granted",
    "data": {
      "required": "capability/file-write",
      "granted": ["capability/file-read", "capability/network"]
    }
  }
}
```

File-path and network-host constraints are enforced at the system call boundary using OS-level mechanisms where available (pledge/unveil on OpenBSD, seccomp on Linux) and by host-side validation of all path and URL arguments on other platforms.

### Timeout and Crash Handling

- **Per-invocation timeout**: each `tools/invoke` call carries a `timeoutMs` field. If the extension does not respond within that deadline, the host sends a `tools/cancel` request and, if still unresponsive after a grace period, terminates the process.
- **Ping timeout**: if an extension fails to respond to `extension/ping` within 5 seconds it is considered unresponsive. The host logs a structured warning and terminates the process.
- **Unexpected exit**: if the extension process exits with a non-zero status or is killed by a signal, the host marks the extension as **crashed**, emits an `extension.crashed` trace event, and cancels all in-flight invocations with a structured `-32099` (extension crashed) error.
- **Host isolation**: extension crashes NEVER crash the host. A run that loses an extension mid-execution surfaces an error on the affected step and allows the operator to decide whether to abort or continue with a degraded tool set.
