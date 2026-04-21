# Phase 7 Design: Extension Host

**Author:** Ken (Software Architect)  
**Date:** 2026-04-20  
**Status:** PROPOSED  
**Priority:** HIGH (blocking Phase 7 implementation)  
**Requested by:** Cristian (Coordinator)

---

## 1. Architecture Overview

The Extension Host manages out-of-process plugins that contribute tools,
providers, and governance policy rules to the gert v2 runtime. Extensions
communicate with the host via JSON-RPC 2.0 over stdio and are governed by
an explicit capability-grant model defined in §04 of the design spec.

### Extension Lifecycle

```
discovery → verification → launch → handshake → contribution → ready → [invocations] → shutdown
```

State machine (from §04):

```
discovered → verified → starting → initializing → ready → draining → stopped
                                                      └─→ crashed
```

### Data Flow Diagram

```
┌───────────────────────────────────────────────────────────────┐
│  EngineConfig                                                 │
│                                                               │
│  ┌──────────────────┐    contributes tools    ┌────────────┐ │
│  │  ExtensionHost   │ ──────────────────────→ │ToolRegistry│ │
│  │  (pkg/extension) │                         │(pkg/tool)  │ │
│  │                  │    contributes rules     ├────────────┤ │
│  │  Interface only  │ ──────────────────────→ │PolicyEval  │ │
│  │                  │                         │(governance)│ │
│  │                  │    contributes provs    ├────────────┤ │
│  │                  │ ──────────────────────→ │ProviderReg │ │
│  └────────┬─────────┘                         │(Phase 8)   │ │
│           │                                   └────────────┘ │
└───────────┼───────────────────────────────────────────────────┘
            │ implemented by
┌───────────▼──────────────────────────────────────────┐
│  internal/extension/                                  │
│                                                       │
│  ┌─────────────┐   ┌───────────────┐   ┌──────────┐ │
│  │ host.go     │   │ process.go    │   │ proto.go │ │
│  │ (concrete   │   │ (subprocess   │   │ (JSON-RPC│ │
│  │  host impl) │   │  manager)     │   │  codec)  │ │
│  └─────────────┘   └───────────────┘   └──────────┘ │
│                                                       │
│  ┌─────────────┐   ┌───────────────┐   ┌──────────┐ │
│  │ manifest.go │   │ health.go     │   │ grants.go│ │
│  │ (YAML parse │   │ (ping/crash   │   │ (capabil-│ │
│  │  + validate)│   │  recovery)    │   │  ity eval│ │
│  └─────────────┘   └───────────────┘   └──────────┘ │
└───────────────────────────────────────────────────────┘
```

### How Contributed Tools Flow into Phase 6 ToolRegistry

1. Extension returns `contributions/list` with `tools` array
2. Host validates each tool against granted capabilities (`capability/tool-registration` required)
3. Host converts wire-format tool definitions into `*schema.ToolDef` objects
4. Host registers them into the `ToolRegistry` (same interface Phase 6 uses)
5. Extension-contributed tools are resolved BEFORE static tools (per §05 name resolution order)

### How Contributed Policy Rules Flow into Governance

1. Extension returns `contributions/list` with `policyContributions` array
2. Host validates `capability/policy-contribution` is granted
3. Host converts wire-format rules into `governance.PolicyRule` structs
4. Rules are additive-only: they cannot override or disable built-in rules
5. Host-defined rules take precedence on conflict (§11 governance contract)

### How Contributed Providers Flow (Phase 8 Preview)

1. Extension returns `contributions/list` with `providers` array
2. Host validates `capability/provider-registration` is granted
3. Host registers provider with its declared `prefixes` (e.g., `["pd."]`)
4. At resolution time, `from: pd.incident_id` routes to the extension's provider
5. Full provider framework is Phase 8; extension host prepares the registration hooks

### ExtensionHost Placement in EngineConfig

```go
// EngineConfig (engine.go) — add field:
ExtensionHost extension.ExtensionHost // Optional: if nil, no extensions loaded
```

The engine calls `ExtensionHost.Load()` during run initialization (after config
validation, before step execution begins). This matches the pattern used by
`GovernanceEvaluator` and `ToolRuntime` — optional injectable dependency.

---

## 2. Decisions

### D1: Extension Communication Protocol — JSON-RPC 2.0 over stdio

**What:** Extensions communicate with the host exclusively via JSON-RPC 2.0
over stdin/stdout pipes. No gRPC. No HTTP. No Unix sockets.

**Why:** §04 spec mandates: "They communicate with the host over JSON-RPC 2.0."
The manifest field `transport: stdio-jsonrpc` is the primary transport. The spec
also mentions `grpc` and `mcp` as manifest options, but for v2.0:

- `stdio-jsonrpc` is the only REQUIRED implementation
- `mcp` is OPTIONAL (extensions may speak MCP protocol over the same stdio pipe —
  the host adapts MCP framing to JSON-RPC internally)
- `grpc` is DEFERRED (open question Q8 from §09; no implementation in v2.0)

**Constraint:** All extension protocol messages use JSON-RPC 2.0 envelope format.
The host is always the initiator; extensions never send unsolicited requests.

---

### D2: Extension Discovery — Multi-Source, Manifest-Based

**What:** Extensions are discovered via manifest files (`gert-extension.yaml`)
found through four sources in precedence order:

1. Built-in extensions (compiled into host binary)
2. Workspace config (`.gert/extensions.yaml`)
3. Runbook declaration (`extensions:` field in runbook)
4. CLI override (`--extension <path>`)

Later sources take precedence. Deduplication by `meta.name`.

**Why:** §04 §ext-discovery defines this exact resolution order. The existing
`ExtensionDecl` stub in `pkg/extension/host.go` has `Path` and `Name` fields
that map directly to this model.

**Constraint:** Discovery is synchronous and completes before any extension
process is started. All manifests are validated before launching any subprocess.

---

### D3: Contribution Registration — Eager, At Load Time

**What:** All contributions (tools, providers, policy rules) are collected
immediately after handshake via `contributions/list`. No lazy loading.

**Why:** §04 lifecycle: "The extension registers its contributions (tools, schema
fields, policies, providers)" happens at step 5, before "the host dispatches
invocations" at step 6. The spec requires all contributions to be known before
execution begins.

This also matches Phase 6 ToolRegistry's design: tool catalog is frozen before
run start. The planner validates tool references against the registry — if tools
were lazily registered, planner validation would be incomplete.

**Constraint:** All extension contributions are available in the registry before
the first step executes. No mid-run registration.

---

### D4: Extension Isolation — One Process Per Extension

**What:** Each extension runs as exactly one child process. No pooling. No
sharing of processes between extensions.

**Why:** §04 §ext-sandbox: "Extensions always run as child processes of the gert
host. They are never loaded as shared libraries or plug-ins executed in-process."

One-process-per-extension provides:
- Clean isolation (an extension crash cannot affect another extension)
- Simple capability enforcement (granted capabilities are per-process)
- Deterministic shutdown (SIGTERM per process)
- Clear stderr attribution (each process's stderr maps to one extension)

**Constraint:** Extension count is bounded by available OS processes. For v2.0
this is acceptable (typical usage: 1-5 extensions per run). Process pooling
is a v2.1 optimization if needed.

---

### D5: Grants/Permissions — Spec-Driven Capability Model

**What:** The `Grants []string` field in `ExtensionDecl` declares which
capabilities from §04 Table 1 the HOST IS WILLING TO GRANT to this extension.
This is the host-side policy decision.

The flow:
1. Extension manifest declares REQUESTED capabilities (`capabilities:` field)
2. Project manifest (`ExtensionDecl.Grants`) declares OFFERED capabilities
3. Host computes `grantedCapabilities = intersection(requested, offered)`
4. Host sends `grantedCapabilities` in `extension/initialize`
5. Extension acknowledges (may reject if a required capability is missing)

**Why:** §04: "Capabilities are requested in the manifest and granted (or denied)
by host policy before the process is started." The two-phase model (request +
grant) provides defense-in-depth: an extension cannot gain capabilities the
host policy hasn't explicitly permitted.

**Capability taxonomy (v2.0):**

| Capability | Allows |
|---|---|
| `capability/tool-registration` | Register new tool definitions |
| `capability/schema-extension` | Contribute namespaced schema fields |
| `capability/event-subscription` | Subscribe to host runtime events |
| `capability/policy-contribution` | Register governance policy rules |
| `capability/provider-registration` | Register input providers |
| `capability/file-read` | Read files within declared paths |
| `capability/file-write` | Write files within declared paths |
| `capability/network` | Outbound network to declared hosts |
| `capability/env-read` | Read declared environment variables |
| `capability/run-state-read` | Read current run state |

**Constraint:** `Grants []string` in `ExtensionDecl` uses the exact
`capability/*` strings from the taxonomy. Unknown capability strings are
silently ignored (forward compatibility).

---

### D6: Extension Health — Ping + Crash Detection, No Auto-Restart

**What:** The host performs periodic `extension/ping` (default 30s interval,
5s timeout). On ping failure OR unexpected process exit:

1. Mark extension as `crashed`
2. Emit `extension.crashed` trace event
3. Cancel all in-flight invocations with error code `-32099`
4. Surface error to operator — DO NOT auto-restart

**Why:** §04 §ext-crash: "extension crashes never crash the host." The spec
describes crash handling but does NOT mandate auto-restart. Auto-restart is
dangerous because:
- A crash during contribution registration leaves partial state
- Restarted extensions may produce different contributions (non-determinism)
- Circuit-breaking without operator consent violates governance auditability

For v2.0, the operator decides whether to abort or continue with degraded tools.
v2.1 may add configurable restart policy (`restart: never | once | always`).

**Constraint:** No automatic restart in v2.0. Failed extensions stay in
`crashed` state for the remainder of the run.

---

### D7: ExtensionHost Integration Point — Before Run Start

**What:** The engine calls `ExtensionHost.Load()` during run initialization,
AFTER `EngineConfig.Validate()` succeeds, BEFORE the first step executes.

Sequence:
```
engine.Start(ctx, plan, opts)
  → config.Validate()
  → ExtensionHost.Load(ctx, manifest)  // discover + launch + handshake + contributions
  → register contributed tools into ToolRegistry
  → register contributed policy rules into PolicyEvaluator
  → begin step execution loop
```

On `engine.Shutdown()` or run completion:
```
  → ExtensionHost.Shutdown(ctx)  // graceful stop all extensions
```

**Why:** Extensions must be fully loaded before step execution because:
1. Planner validation needs the complete tool catalog
2. Governance evaluation needs all policy rules before first pre-flight check
3. Provider resolution needs registered prefixes before first `from:` binding

**Constraint:** `ExtensionHost.Load()` is synchronous and blocking. If any
extension fails to initialize, the error is surfaced but does NOT abort the run
(the extension is skipped with a warning, per §04 versioning rules).

---

### D8: Schema for Project Manifest — Workspace `.gert/extensions.yaml`

**What:** Extension declarations live in `.gert/extensions.yaml` (workspace
level) and optionally in the runbook's `extensions:` field. There is no
dedicated `gert.yaml` project manifest.

The `ProjectManifest` struct in `pkg/extension/host.go` is the in-memory
representation populated from these sources. It is NOT written as a separate
file — it's assembled by the discovery process from multiple sources.

**Why:** §04 §ext-discovery defines exactly these locations. The existing
`ProjectManifest` struct serves as the aggregated in-memory model. Adding a
separate `gert.yaml` would conflict with the spec and add an unnecessary
configuration file.

**Constraint:** The `ProjectManifest` struct name stays but represents the
MERGED discovery result, not a single file. Rename to `DiscoveredExtensions`
is acceptable if clearer during implementation.

---

## 3. Package Map

```
v2/
├── pkg/
│   └── extension/                   ← Public interfaces (near-leaf package)
│       ├── doc.go                   ← Package documentation (existing)
│       ├── host.go                  ← ExtensionHost interface (extend existing)
│       ├── manifest.go              ← ExtensionManifest, Compatibility types
│       ├── contribution.go          ← Contribution, ContributedTool, ContributedProvider
│       ├── capability.go            ← Capability constants, CapabilitySet type
│       └── state.go                 ← ExtensionState enum, lifecycle states
│
├── internal/
│   └── extension/                   ← Concrete implementation (NEW)
│       ├── doc.go                   ← Package documentation
│       ├── host.go                  ← concreteHost (implements ExtensionHost)
│       ├── process.go               ← ExtensionProcess, subprocess management
│       ├── proto.go                 ← JSON-RPC codec, message types
│       ├── handshake.go             ← Initialize/contributions protocol
│       ├── manifest.go              ← YAML parser for gert-extension.yaml
│       ├── discovery.go             ← Multi-source discovery logic
│       ├── health.go                ← Ping loop, crash detection
│       ├── grants.go                ← Capability intersection, enforcement
│       ├── host_test.go
│       ├── process_test.go
│       ├── proto_test.go
│       ├── handshake_test.go
│       ├── manifest_test.go
│       ├── discovery_test.go
│       ├── health_test.go
│       └── grants_test.go
│
├── pkg/testutil/
│   └── fakeexthost.go               ← FakeExtensionHost for engine tests
│
├── cmd/extensions/                   ← Reference extension binaries (NEW)
│   └── hello-ext/
│       ├── main.go                  ← Entry point
│       └── gert-extension.yaml      ← Extension manifest
│
└── testdata/extensions/              ← Test extension manifests (NEW)
    ├── valid-minimal.yaml
    ├── valid-full.yaml
    ├── invalid-missing-name.yaml
    ├── invalid-bad-capability.yaml
    └── incompatible-version.yaml
```

### Import Rules

```
pkg/extension → pkg/schema, pkg/governance, pkg/tool  (ALLOWED: near-leaf)
pkg/extension → internal/*                             (FORBIDDEN: direction violation)
pkg/engine    → pkg/extension                          (ALLOWED: interface only)
internal/extension → pkg/extension                     (ALLOWED: implements interface)
internal/extension → pkg/tool                          (ALLOWED: registers tools)
internal/extension → pkg/governance                    (ALLOWED: registers policy rules)
internal/extension → pkg/schema                        (ALLOWED: uses ToolDef, ProviderDef)
internal/extension → internal/tool                     (FORBIDDEN: cross-internal)
```

---

## 4. Interface Contracts (Go Pseudocode)

### Extended `ExtensionHost` Interface

The existing stub interface is sufficient for Phase 7 with one addition — a
`Status` method for observability:

```go
package extension

// ExtensionHost manages the lifecycle of out-of-process extensions.
type ExtensionHost interface {
    // Load discovers and launches all extensions declared in the manifest.
    // Blocks until all extensions complete handshake and register contributions.
    // Individual extension failures are non-fatal (logged, extension skipped).
    Load(ctx context.Context, manifest *ProjectManifest) error

    // Shutdown gracefully stops all running extensions.
    // Sends extension/shutdown to each, waits up to shutdownTimeout, then SIGTERM.
    Shutdown(ctx context.Context) error

    // ContributedTools returns tools from all loaded extensions.
    ContributedTools() []*schema.ToolDef

    // ContributedProviders returns providers from all loaded extensions.
    ContributedProviders() []*schema.ProviderDef

    // ContributedPolicyRules returns governance rules from all loaded extensions.
    ContributedPolicyRules() []governance.PolicyRule

    // Status returns the current state of each loaded extension.
    Status() []ExtensionStatus
}

// ExtensionStatus reports the state of a single loaded extension.
type ExtensionStatus struct {
    Name    string
    Version string
    State   ExtensionState
    Error   error  // non-nil if State == StateCrashed
}
```

### ExtensionState Lifecycle Enum

```go
package extension

type ExtensionState int

const (
    StateDiscovered   ExtensionState = iota
    StateVerified
    StateStarting
    StateInitializing
    StateReady
    StateDraining
    StateStopped
    StateCrashed
)
```

### Capability Constants

```go
package extension

const (
    CapToolRegistration     = "capability/tool-registration"
    CapSchemaExtension      = "capability/schema-extension"
    CapEventSubscription    = "capability/event-subscription"
    CapPolicyContribution   = "capability/policy-contribution"
    CapProviderRegistration = "capability/provider-registration"
    CapFileRead             = "capability/file-read"
    CapFileWrite            = "capability/file-write"
    CapNetwork              = "capability/network"
    CapEnvRead              = "capability/env-read"
    CapRunStateRead         = "capability/run-state-read"
)

// CapabilitySet is a set of granted capabilities.
type CapabilitySet map[string]bool

// Has reports whether the capability is in the set.
func (cs CapabilitySet) Has(cap string) bool { return cs[cap] }

// Intersect returns capabilities present in both sets.
func (cs CapabilitySet) Intersect(requested []string) CapabilitySet { ... }
```

### ExtensionManifest (Parsed from `gert-extension.yaml`)

```go
package extension

// ExtensionManifest is the parsed gert-extension.yaml.
type ExtensionManifest struct {
    APIVersion    string            `yaml:"apiVersion"`
    Meta          ManifestMeta      `yaml:"meta"`
    Compatibility Compatibility     `yaml:"compatibility"`
    EntryPoint    EntryPoint        `yaml:"entry-point"`
    Transport     string            `yaml:"transport"` // "stdio-jsonrpc" | "mcp" | "grpc"
    Capabilities  []string          `yaml:"capabilities"`
    FileReadPaths []string          `yaml:"file-read-paths,omitempty"`
    FileWritePaths[]string          `yaml:"file-write-paths,omitempty"`
    NetworkHosts  []string          `yaml:"network-hosts,omitempty"`
    EnvReadPats   []string          `yaml:"env-read-patterns,omitempty"`
    Platform      *PlatformConstraint `yaml:"platform,omitempty"`
}

type ManifestMeta struct {
    Name        string `yaml:"name"`
    Version     string `yaml:"version"`
    Description string `yaml:"description,omitempty"`
    Author      string `yaml:"author,omitempty"`
}

type Compatibility struct {
    APIVersion      string `yaml:"api-version"`      // semver range
    ProtocolVersion string `yaml:"protocol-version"` // "2.0"
}

type EntryPoint struct {
    Executable string   `yaml:"executable"`
    Args       []string `yaml:"args,omitempty"`
}

type PlatformConstraint struct {
    OS   []string `yaml:"os,omitempty"`
    Arch []string `yaml:"arch,omitempty"`
}
```

### Contribution Types

```go
package extension

// Contribution is the wire-format result of contributions/list.
type Contribution struct {
    Tools            []ContributedTool     `json:"tools,omitempty"`
    Providers        []ContributedProvider `json:"providers,omitempty"`
    SchemaExtensions []SchemaExtension     `json:"schemaExtensions,omitempty"`
    PolicyRules      []ContributedPolicy   `json:"policyContributions,omitempty"`
}

// ContributedTool is a tool declared by an extension.
type ContributedTool struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description,omitempty"`
    Transport   string                 `json:"transport,omitempty"` // defaults to parent extension's transport
    Actions     map[string]*ToolAction `json:"actions,omitempty"`
}

// ContributedProvider is a provider declared by an extension.
type ContributedProvider struct {
    Name     string   `json:"name"`
    Prefixes []string `json:"prefixes"`
}

// ContributedPolicy is a governance rule declared by an extension.
type ContributedPolicy struct {
    ID          string   `json:"id"`
    Description string   `json:"description,omitempty"`
    Allow       []string `json:"allow,omitempty"`
    Deny        []string `json:"deny,omitempty"`
    DenyEnvVars []string `json:"denyEnvVars,omitempty"`
}

// SchemaExtension is a namespaced field contributed by an extension.
type SchemaExtension struct {
    Namespace string         `json:"namespace"` // e.g., "x-acme"
    Fields    map[string]any `json:"fields"`
}
```

### Internal: ExtensionProcess

```go
package extension // internal/extension

// ExtensionProcess manages a single extension subprocess.
type ExtensionProcess struct {
    manifest  *ExtensionManifest
    granted   CapabilitySet
    state     ExtensionState
    cmd       *exec.Cmd
    stdin     io.WriteCloser
    stdout    io.ReadCloser
    stderr    io.ReadCloser
    codec     *jsonrpcCodec
    contribs  *Contribution
    mu        sync.Mutex
}

func (p *ExtensionProcess) Start(ctx context.Context) error { ... }
func (p *ExtensionProcess) Initialize(ctx context.Context, hostInfo HostInfo) error { ... }
func (p *ExtensionProcess) ListContributions(ctx context.Context) (*Contribution, error) { ... }
func (p *ExtensionProcess) Ping(ctx context.Context) error { ... }
func (p *ExtensionProcess) Shutdown(ctx context.Context) error { ... }
func (p *ExtensionProcess) Kill() { ... }
func (p *ExtensionProcess) State() ExtensionState { ... }
```

### Contribution Flow Pseudocode

```go
// ExtensionHost → ToolRegistry integration (at engine init)
func wireExtensions(host extension.ExtensionHost, reg tool.ToolRegistry) {
    for _, td := range host.ContributedTools() {
        reg.Register(td) // ToolRegistry.Register added in Phase 7
    }
}

// ExtensionHost → PolicyEvaluator integration
func wirePolicies(host extension.ExtensionHost, eval governance.PolicyEvaluator) {
    for _, pr := range host.ContributedPolicyRules() {
        eval.AddRule(pr) // PolicyEvaluator.AddRule added in Phase 7
    }
}
```

**Note:** `ToolRegistry.Register()` and `PolicyEvaluator.AddRule()` are new
methods that Phase 7 implementation must add to the existing interfaces (or use
a mutable variant). The read-only `ToolRegistry` from Phase 6 needs a
`MutableToolRegistry` extension:

```go
// pkg/tool — additive interface (does not break Phase 6)
type MutableToolRegistry interface {
    ToolRegistry
    Register(def *schema.ToolDef) error
}
```

---

## 5. Extension Communication Protocol

### Wire Format

All messages use JSON-RPC 2.0 over newline-delimited JSON on stdin/stdout:

```
← host writes to extension's stdin
→ extension writes to extension's stdout (host reads)
```

Each message is a single JSON object terminated by `\n`.

### Handshake Sequence

```
Host                              Extension
  │                                    │
  │── extension/initialize ──────────→ │
  │                                    │
  │←── result (acknowledged caps) ─────│
  │                                    │
  │── contributions/list ────────────→ │
  │                                    │
  │←── result (tools, providers, ...) ─│
  │                                    │
  │        [extension is now READY]    │
  │                                    │
```

### Method: `extension/initialize`

Request (host → extension):
```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "extension/initialize",
  "params": {
    "host": { "name": "gert", "version": "2.0.0", "apiVersion": "2.0" },
    "protocolVersion": "2.0",
    "grantedCapabilities": ["capability/tool-registration", "capability/network"],
    "runContext": { "runId": "run-abc123", "mode": "real" }
  }
}
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "extension": { "name": "acme.incident-pack", "version": "1.2.0" },
    "protocolVersion": "2.0",
    "acknowledgedCapabilities": ["capability/tool-registration", "capability/network"]
  }
}
```

Error codes: `-32001` (protocol unsupported), `-32002` (required cap missing),
`-32003` (API version out of range).

### Method: `contributions/list`

Request:
```json
{ "jsonrpc": "2.0", "id": "2", "method": "contributions/list", "params": {} }
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "2",
  "result": {
    "tools": [{ "name": "acme.pd-lookup", "actions": {"lookup": {...}} }],
    "providers": [{ "name": "pd", "prefixes": ["pd."] }],
    "schemaExtensions": [],
    "policyContributions": [{ "id": "acme-allow-pd", "allow": ["pd-*"] }]
  }
}
```

### Method: `extension/ping`

Request:
```json
{ "jsonrpc": "2.0", "id": "42", "method": "extension/ping", "params": {} }
```

Response (must arrive within 5 seconds):
```json
{ "jsonrpc": "2.0", "id": "42", "result": { "status": "ok" } }
```

### Method: `extension/shutdown`

Request:
```json
{ "jsonrpc": "2.0", "id": "99", "method": "extension/shutdown", "params": {} }
```

Response (best-effort):
```json
{ "jsonrpc": "2.0", "id": "99", "result": { "status": "ok" } }
```

If no response within 10s: SIGTERM. If no exit within 5s after SIGTERM: SIGKILL.

### Method: `tools/invoke` (host → extension, during execution)

Request:
```json
{
  "jsonrpc": "2.0",
  "id": "500",
  "method": "tools/invoke",
  "params": {
    "tool": "acme.pd-lookup",
    "action": "lookup",
    "args": { "incident_id": "INC-42" },
    "timeoutMs": 30000
  }
}
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "500",
  "result": {
    "exitCode": 0,
    "output": { "status": "resolved", "title": "DB overload" }
  }
}
```

### Health Ping Mechanism

- Ping interval: 30 seconds (configurable via host config)
- Ping timeout: 5 seconds (per §04 spec)
- On timeout: transition to `crashed` state, cancel in-flight invocations
- Ping starts AFTER extension reaches `ready` state
- No ping during handshake (handshake has its own timeout: 30s default)

---

## 6. Test Plan (28 Tests)

### Group A: Host Lifecycle Tests (7 tests)

| # | Test | Verifies |
|---|------|----------|
| A1 | `TestLoad_SingleExtension_Success` | Load with one valid extension reaches ready state |
| A2 | `TestLoad_MultipleExtensions_AllReady` | Load with 3 extensions, all reach ready |
| A3 | `TestLoad_ExtensionFailsHandshake_Skipped` | One extension fails init, others still load |
| A4 | `TestLoad_EmptyManifest_NoError` | Empty ProjectManifest is valid (no-op) |
| A5 | `TestShutdown_GracefulStop` | Shutdown sends extension/shutdown, waits for exit |
| A6 | `TestShutdown_TimeoutForceKill` | Extension ignores shutdown → SIGTERM → SIGKILL |
| A7 | `TestLoad_ContextCancelled_AbortsDiscovery` | Context cancellation during load aborts cleanly |

### Group B: Handshake Tests (6 tests)

| # | Test | Verifies |
|---|------|----------|
| B1 | `TestHandshake_ProtocolVersionMismatch_Error` | Extension returns -32001, host marks as crashed |
| B2 | `TestHandshake_RequiredCapabilityDenied_Error` | Extension returns -32002, host skips extension |
| B3 | `TestHandshake_APIVersionIncompatible_Rejected` | Manifest compatibility check rejects before launch |
| B4 | `TestHandshake_Success_CapabilitiesAcknowledged` | Normal flow: init → contributions → ready |
| B5 | `TestHandshake_Timeout_MarkedCrashed` | Extension doesn't respond to init within timeout |
| B6 | `TestHandshake_PartialCapabilities_Accepted` | Extension gets subset of requested capabilities |

### Group C: Contribution Registration Tests (7 tests)

| # | Test | Verifies |
|---|------|----------|
| C1 | `TestContributions_ToolRegistration` | Contributed tools appear in ContributedTools() |
| C2 | `TestContributions_PolicyRuleRegistration` | Contributed rules appear in ContributedPolicyRules() |
| C3 | `TestContributions_ProviderRegistration` | Contributed providers appear in ContributedProviders() |
| C4 | `TestContributions_CapabilityEnforcement_ToolBlocked` | Tool registration without capability → rejected |
| C5 | `TestContributions_CapabilityEnforcement_PolicyBlocked` | Policy contribution without capability → rejected |
| C6 | `TestContributions_DuplicateToolName_Warning` | Duplicate tool name logs warning, first wins |
| C7 | `TestContributions_EmptyContributions_Valid` | Extension with no contributions is valid |

### Group D: Health and Crash Tests (4 tests)

| # | Test | Verifies |
|---|------|----------|
| D1 | `TestPing_Success_ExtensionStaysReady` | Successful ping keeps extension in ready state |
| D2 | `TestPing_Timeout_MarkedCrashed` | No ping response → crashed state |
| D3 | `TestCrash_ProcessExits_MarkedCrashed` | Extension process exits unexpectedly → crashed |
| D4 | `TestCrash_InFlightInvocation_CancelledWithError` | Crash during tool invocation returns -32099 |

### Group E: Engine Integration Tests (4 tests)

| # | Test | Verifies |
|---|------|----------|
| E1 | `TestEngine_ExtensionToolInvocation_EndToEnd` | Engine invokes extension-contributed tool |
| E2 | `TestEngine_ExtensionPolicyRule_Evaluated` | Extension-contributed rule blocks a command |
| E3 | `TestEngine_NoExtensionHost_NilSafe` | Engine runs with nil ExtensionHost (no crash) |
| E4 | `TestEngine_ExtensionCrashMidRun_StepErrors` | Extension crashes → affected steps error |

---

## 7. Import Constraint Analysis

### Dependency Graph (Allowed)

```
pkg/engine ──→ pkg/extension (interface)
pkg/engine ──→ pkg/tool (interface)
pkg/engine ──→ pkg/governance (interface)

pkg/extension ──→ pkg/schema (ToolDef, ProviderDef)
pkg/extension ──→ pkg/governance (PolicyRule)
pkg/extension ──→ pkg/tool (ToolRegistry — for contribution type alignment)

internal/extension ──→ pkg/extension (implements ExtensionHost)
internal/extension ──→ pkg/schema (converts wire → ToolDef)
internal/extension ──→ pkg/governance (converts wire → PolicyRule)
internal/extension ──→ pkg/tool (uses ToolDef shape for validation)
```

### Forbidden Imports (Cycle Prevention)

```
pkg/extension ──✗──→ internal/*         (public pkg cannot import internal)
pkg/extension ──✗──→ pkg/engine         (would create cycle: engine→extension→engine)
internal/extension ──✗──→ internal/tool  (cross-internal: use pkg/tool interface instead)
internal/extension ──✗──→ pkg/engine     (would require engine types in extension)
```

### Strategy for Contribution Wiring

The engine (or a wiring layer in `internal/engine/`) is responsible for:
1. Calling `ExtensionHost.ContributedTools()` → `ToolRegistry.Register()`
2. Calling `ExtensionHost.ContributedPolicyRules()` → `PolicyEvaluator.AddRule()`

This keeps `pkg/extension` free of registry/evaluator imports at the concrete
level — it only imports the contribution TYPES (ToolDef, PolicyRule), not the
consumers.

---

## 8. Reference Extension: `hello-ext`

### Purpose

A minimal extension binary for integration testing. Demonstrates:
- Proper JSON-RPC 2.0 handshake
- Contributing one tool and one policy rule
- Responding to pings
- Graceful shutdown

### Location

```
v2/cmd/extensions/hello-ext/
├── main.go                  ← Extension binary
└── gert-extension.yaml      ← Extension manifest
```

### Manifest (`gert-extension.yaml`)

```yaml
apiVersion: extension/v2

meta:
  name: gert.hello-ext
  version: "1.0.0"
  description: "Reference extension for testing"
  author: "gert team"

compatibility:
  api-version: ">=2.0.0 <3.0.0"
  protocol-version: "2.0"

entry-point:
  executable: "./hello-ext"

transport: stdio-jsonrpc

capabilities:
  - capability/tool-registration
  - capability/policy-contribution
```

### Contributed Tool

```json
{
  "name": "gert.hello",
  "description": "Says hello — reference tool for testing",
  "actions": {
    "greet": {
      "description": "Returns a greeting",
      "args": { "name": { "type": "string", "required": true } },
      "returns": "string"
    }
  }
}
```

### Contributed Policy Rule

```json
{
  "id": "hello-allow",
  "description": "Allow hello-related commands",
  "allow": ["echo", "printf"]
}
```

### Behavior

1. On `extension/initialize`: validates protocol version, acknowledges capabilities
2. On `contributions/list`: returns one tool (`gert.hello`) and one policy rule
3. On `tools/invoke` (tool=`gert.hello`, action=`greet`): returns `{"output": {"greeting": "Hello, <name>!"}}`
4. On `extension/ping`: returns `{"status": "ok"}`
5. On `extension/shutdown`: exits cleanly with code 0

### Implementation Sketch (`main.go`)

```go
package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "os"
)

func main() {
    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        var req jsonrpcRequest
        if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
            continue
        }
        resp := handleRequest(req)
        data, _ := json.Marshal(resp)
        fmt.Fprintf(os.Stdout, "%s\n", data)
        if req.Method == "extension/shutdown" {
            os.Exit(0)
        }
    }
}

func handleRequest(req jsonrpcRequest) jsonrpcResponse {
    switch req.Method {
    case "extension/initialize":
        return initializeResponse(req)
    case "contributions/list":
        return contributionsResponse(req)
    case "tools/invoke":
        return invokeResponse(req)
    case "extension/ping":
        return pingResponse(req)
    case "extension/shutdown":
        return shutdownResponse(req)
    default:
        return errorResponse(req.ID, -32601, "method not found")
    }
}
```

---

## 9. Open Questions

### Q1: Should `ToolRegistry` become `MutableToolRegistry`?

Phase 6's `ToolRegistry` is read-only (`Lookup`, `All`). Extensions need to
`Register` tools. Options:
- (a) Add `Register` to existing `ToolRegistry` (breaking change)
- (b) New `MutableToolRegistry` interface that embeds `ToolRegistry`
- (c) Registry constructor accepts initial set + extension contributions at build time

**Recommendation:** Option (b) — additive, no Phase 6 breakage.

### Q2: Should extension manifest validation use JSON Schema or Go code?

§04 implies structural validation. Options:
- JSON Schema for gert-extension.yaml (reuses validation infra from §03)
- Pure Go struct validation (simpler, no schema files to ship)

**Recommendation:** Go struct validation for v2.0. JSON Schema for v2.1 if
community extensions emerge and need self-service validation tooling.

### Q3: How should extension-contributed tool invocations be routed?

When a step invokes `acme.pd-lookup`, the ToolRuntime needs to route the call
BACK to the extension process (not spawn a new subprocess). Options:
- (a) Extension-contributed tools get a special transport type (`extension://`)
- (b) ToolDef.Source field encodes the extension name (`ext://acme.incident-pack/pd-lookup`)
- (c) ToolRuntime delegates to ExtensionHost for extension-sourced tools

**Recommendation:** Option (c) — the engine or ToolRuntime checks if the tool
was contributed by an extension and delegates `tools/invoke` to the extension
process via the host. This keeps ToolTransport agnostic to extensions.

### Q4: What happens when two extensions contribute conflicting policy rules?

§11 says host rules take precedence. But what about extension-vs-extension?
- First-registered wins? (discovery order dependent)
- Both apply? (deny from either blocks)

**Recommendation:** Both apply. All deny rules from all extensions are merged
(deny union). All allow rules are intersected only within the same extension.
Cross-extension: deny-wins from ANY source.

### Q5: Should extensions be loadable in test mode without subprocess?

For unit testing, spawning real processes is slow. Should `pkg/extension` provide
a `FakeExtensionHost` or should tests use the existing interface with mocks?

**Recommendation:** `pkg/testutil.FakeExtensionHost` (same pattern as
`FakeTraceWriter`, `FakePlatform`). Returns configurable contributions without
spawning any processes.

---

## Summary

Phase 7 is a **protocol implementation phase**. The interfaces already exist in
stub form; the design spec (§04) defines the wire protocol completely. The
implementation work is:

1. Parse `gert-extension.yaml` manifests (internal/extension/manifest.go)
2. Discover extensions from 4 sources (internal/extension/discovery.go)
3. Manage subprocesses (internal/extension/process.go)
4. Implement JSON-RPC 2.0 handshake (internal/extension/handshake.go, proto.go)
5. Wire contributions into ToolRegistry and PolicyEvaluator
6. Health monitoring (internal/extension/health.go)
7. Build `hello-ext` reference binary (cmd/extensions/hello-ext/)
8. FakeExtensionHost for engine tests (pkg/testutil/fakeexthost.go)

Estimated effort: **Medium-High** (JSON-RPC codec + process management are non-trivial).
Critical path dependency: Phase 6 ToolRegistry must support `Register()` (Q1 resolution).
