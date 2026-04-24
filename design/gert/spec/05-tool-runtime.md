# Tool Runtime

The tool runtime resolves, validates, and invokes tool definitions at runbook execution time. Tools are the primary mechanism by which gert performs side-effecting work: executing binaries, calling JSON-RPC services, and delegating to MCP-hosted capabilities. Every invocation is subject to capability gate checking, governance policy, and output redaction before any captured value enters the run state or trace.

## Tool Discovery

The host resolves tool references in three phases:

### Phase 1: Static Discovery

`.tool.yaml` files located by scanning the following directories in order:
1. The built-in tool registry compiled into the host binary.
2. `<workspace-root>/tools/` — conventional location for project-level tool definitions (`tools/<name>.tool.yaml`).
3. Directories listed in the workspace config (`.gert/config.yaml`, field `tool-paths`).
4. Required packages declared in the runbook's `requires:` field (each package's `tools/` subdirectory is scanned).

### Phase 2: Dynamic Registration

Extensions with `capability/tool-registration` may register additional tool definitions during the `contributions/list` phase. Dynamically registered tools are merged into the resolved catalog before execution begins.

### Phase 3: MCP Discovery

When the runbook declares an MCP server under `tools.mcp-servers`, the host performs a `tools/list` call to that server at startup and registers every tool reported as a dynamically discovered tool.

### Name Resolution Order

When a runbook step references a tool by name:
1. Exact match in dynamically registered tools (MCP and extension, in registration order).
2. Exact match in project-level `tools/` directory.
3. Exact match in required-package tool directories (in `requires:` declaration order).
4. Exact match in the built-in registry.

If two tools share the same name, the first match wins and a warning is emitted. Tool authors SHOULD use reverse-DNS namespace prefixes to avoid collisions (e.g. `acme.pd-lookup`).

## Tool Definition Schema (v2)

```text
# tools/my-service.tool.yaml
apiVersion: tool/v2

meta:
  name: my-service              # Unique identifier within the workspace
  version: "2.0.0"             # Tool definition version (semver)
  description: "Calls the MyService REST API"
  binary: my-service-cli        # Executable name (looked up in PATH)

transport:
  mode: stdio-jsonrpc           # stdio | stdio-jsonrpc | grpc | mcp

  # For stdio-jsonrpc and mcp: startup behaviour
  startup:
    argv: ["--server-mode"]     # Extra args passed to the binary
    ready-pattern: "ready"      # Regex matched against stdout/stderr to
                                # confirm the server is accepting requests
    timeout: "10s"
    shutdown-method: "shutdown" # JSON-RPC method to call on graceful stop

governance:
  requires-capabilities: []     # Host capabilities the tool needs, e.g.
                                # ["capability/network"]
  allowed-environments: ["real", "dry-run"]
  requires-approval: false      # Set true to gate invocation on operator OK

actions:
  lookup:
    description: "Look up an entity by ID"
    argv: ["lookup", "--id", "{{ .id }}"]  # Only for stdio transport
    timeout: "30s"
    args:
      id:
        type: string
        required: true
        description: "Entity identifier"
        redact: false
    output:
      format: json              # text | json | jsonl
    capture:
      stdout:
        format: json
```

### Transport-Specific Fields

**`stdio`** — Binary spawned once per invocation. The `argv` array in the action definition is rendered as a Go template with resolved argument values, then appended to the binary name to form the complete command. stdout is captured as the tool output.

**`stdio-jsonrpc`** — Binary spawned once per run (persistent process). A single process kept alive across all action calls. Each invocation sends a JSON-RPC request to the process's stdin and reads the response from stdout. `startup.ready-pattern` confirms the server is accepting connections.

**`mcp`** — Host spawns the binary and performs the MCP initialization handshake. Actions map to MCP `tools/call` requests.

## Transport Layer

### stdio Transport

**Process establishment:** `exec.Command(binary, resolvedArgv...)` with a controlled environment and per-invocation timeout context.

**Invocation:** Encoded entirely in process arguments and environment; no envelope.

**Response:** stdout captured in full after process exit. Non-zero exit code → tool error.

**Error envelope:**

```json
{
  "error": {
    "code": "TOOL_NONZERO_EXIT",
    "message": "tool exited with code 1",
    "details": {
      "exitCode": 1,
      "stderr": "connection refused\n",
      "durationMs": 320
    }
  }
}
```

### stdio-jsonrpc Transport

**Process establishment:** Single process spawned at first use and kept alive for the duration of the run. Host waits for `startup.ready-pattern` on stdout/stderr before sending the first request.

**Invocation envelope:**

```json
{
  "jsonrpc": "2.0",
  "id": "<invocationId>",
  "method": "<actionName>",
  "params": {
    "args": { "id": "INC-12345" },
    "context": {
      "runId": "run-abc123",
      "stepId": "lookup_incident",
      "traceId": "trace-xyz",
      "userId": "ops@acme.example",
      "attempt": 1
    }
  }
}
```

**Response envelope:**

```json
{
  "jsonrpc": "2.0",
  "id": "<invocationId>",
  "result": {
    "output": { ... },
    "telemetry": {
      "startedAt": "2026-04-18T10:00:00Z",
      "finishedAt": "2026-04-18T10:00:00.320Z",
      "durationMs": 320
    }
  }
}
```

**Error envelope:**

```json
{
  "jsonrpc": "2.0",
  "id": "<invocationId>",
  "error": {
    "code": -32050,
    "message": "entity not found",
    "data": {
      "category": "runtime",
      "retryable": false,
      "details": { "entityId": "INC-12345" }
    }
  }
}
```

**Error code ranges:**

| Code | Meaning |
|------|---------|
| `-32700` to `-32600` | JSON-RPC parse and request errors |
| `-32050` | Application-level tool error (retryable per `data.retryable`) |
| `-32051` | Capability not granted |
| `-32052` | Input validation failed |
| `-32053` | Output schema validation failed |
| `-32054` | Timeout |
| `-32055` | Cancelled by host |
| `-32099` | Tool process crashed |

### MCP Transport

See MCP Tool Integration section below.

## Invocation Context

Every tool invocation — regardless of transport — receives a structured context object. For `stdio` tools, relevant fields are injected as environment variables prefixed `GERT_`; for `stdio-jsonrpc` and `mcp` tools, the context is transmitted in the invocation envelope.

```json
{
  "context": {
    "runId":       "run-abc123",        // Globally unique run identifier
    "stepId":      "lookup_incident",   // Step ID within the runbook
    "traceId":     "trace-xyz789",      // Distributed trace correlation ID
    "userId":      "ops@acme.example",  // Identity executing the run
    "mode":        "real",              // real | dry-run | replay
    "attempt":     1,                   // Retry attempt number (1-based)
    "capabilities": [                   // Effective host capability set
      "capability/network",
      "capability/file-read"
    ]
  }
}
```

## Capability Gate

Before dispatching any tool invocation the host checks two conditions:

1. The tool's `governance.requires-capabilities` list is a subset of the current run's granted capability set. If not, the step fails immediately with a `CAPABILITY_DENIED` error; the run does not attempt to start the tool process.
2. The tool's `governance.allowed-environments` list includes the current execution mode. A tool not listed for `dry-run` will be skipped (with a warning) when the run mode is dry-run.

## Output Capture

Tool output is mapped to named step captures using the `capture:` declaration in the action definition.

```yaml
capture:
  dns_output: stdout          # Capture stdout as variable dns_output
  error_log:  stderr          # Capture stderr as variable error_log
  exit_code:  exitCode        # Capture numeric exit code
```

For `json` output format, capture paths may use dot notation to extract nested fields:

```yaml
capture:
  incident_id: "stdout.incident.id"
  severity:    "stdout.incident.severity"
```

**Redaction:** Before any captured value is stored in the run state or emitted to the trace, the governance redaction pipeline is applied. Redaction rules are declared in the runbook's `meta.governance.redact` block and in the tool action's per-argument `redact: true` flag. Redacted values are replaced with `<redacted>` in all stored and displayed output.

## MCP Tool Integration

MCP (Model Context Protocol) servers are long-lived processes that expose a dynamic catalog of tools. Gert treats an MCP server as a tool source, not as a single tool.

### MCP Server Declaration

```yaml
# In runbook meta or .gert/config.yaml
tools:
  mcp-servers:
    - name: acme-mcp
      binary: acme-mcp-server
      args: ["--port", "stdio"]
      transport: mcp
      startup:
        timeout: "15s"
```

### MCP Discovery

On run startup the host:
1. Spawns the MCP server binary.
2. Sends the MCP `initialize` request (per MCP specification).
3. After receiving `initialized`, sends `tools/list`.
4. Registers each returned tool under the name `<server-name>/<tool-name>` in the dynamic tool catalog.

### MCP Tool Invocation

```json
{
  "jsonrpc": "2.0",
  "id": "5",
  "method": "tools/call",
  "params": {
    "name": "lookup_incident",
    "arguments": { "id": "INC-12345" },
    "_gert_context": {
      "runId": "run-abc123",
      "stepId": "triage_step",
      "traceId": "trace-xyz"
    }
  }
}
```

The `_gert_context` field is a gert extension to the MCP envelope. MCP servers that do not recognise it MUST ignore it. The response follows the standard MCP `tools/call` response format.

### MCP vs stdio-jsonrpc Differences

| Feature | MCP | stdio-jsonrpc |
|---------|-----|--------------|
| Tool multiplicity | Multiple tools per process | One tool per process |
| Initialization | `initialize`/`initialized` handshake | `startup.ready-pattern` |
| Tool names | Discovered dynamically at runtime | Declared statically in `.tool.yaml` |
| Response format | `content` array (flattened to string by gert) | Direct result |

## Cancellation and Timeout

**Per-tool timeout:** Declared in the action definition under `timeout:` (e.g. `30s`). Default is the runbook's `meta.defaults.timeout`. Host wraps each invocation in a `context.WithTimeout`; when deadline elapses the host sends a `tools/cancel` request (JSON-RPC transport) or sends SIGTERM (stdio transport), then forcibly terminates the process after a 5-second grace period.

**Cancellation propagation:** When a run is cancelled (via Ctrl-C, the debugger `quit` command, or an API cancellation request), the host:
1. Sets the run-level cancel context.
2. Sends `tools/cancel` to all in-flight JSON-RPC and MCP invocations with `reason: "run-cancelled"`.
3. After a 5-second drain window, forcibly terminates any unresponsive tool processes.
4. Emits a `run.cancelled` trace event recording which steps were interrupted.

**`tools/cancel` request envelope:**

```json
{
  "jsonrpc": "2.0",
  "id": "cancel-5",
  "method": "tools/cancel",
  "params": {
    "invocationId": "5",
    "reason": "run-cancelled"
  }
}
```

## Tool Versioning

Tool definitions carry a `meta.version` field. When multiple definitions for the same tool name are present, the host selects by resolution order (not by version number). Explicit version pinning is supported via the runbook's `requires:` block:

```yaml
requires:
  - package: github.com/acme/gert-tools
    version: ">=1.2.0 <2.0.0"
```
