# Phase 6 Design: Tool Runtime

**Author:** Ken (Software Architect)  
**Date:** 2026-04-20  
**Status:** PROPOSED  
**Priority:** CRITICAL (blocking Phase 6 implementation)  
**Requested by:** Cristian (Coordinator)

---

## 1. Architecture Overview

Phase 6 transforms the `ToolExecutor` stub (Phase 5) into a complete tool runtime
that spawns external processes, negotiates transport protocols, and captures
structured output. The architecture introduces four new concerns:

```
┌─────────────────────────────────────────────────────────┐
│  pkg/engine          (interfaces only)                  │
│   ExecutorRegistry ← ToolExecutor registered as "tool"  │
└──────────────┬──────────────────────────────────────────┘
               │ depends on
┌──────────────▼──────────────────────────────────────────┐
│  pkg/tool            (leaf package, no gert imports)    │
│   ToolRuntime        — invoke interface                 │
│   ToolResult         — invocation result                │
│   Transport          — enum: stdio, jsonrpc, mcp        │
│   ToolRegistry       — discovery + lookup               │
│   ToolDef (re-export)— references pkg/schema.ToolDef    │
└──────────────┬──────────────────────────────────────────┘
               │ implemented by
┌──────────────▼──────────────────────────────────────────┐
│  internal/tool/      (transport implementations)        │
│   stdio.go           — spawn-per-invocation             │
│   jsonrpc.go         — persistent JSON-RPC over stdio   │
│   mcp.go             — MCP handshake + tools/call       │
│   registry.go        — multi-source tool catalog        │
│   process.go         — shared process management        │
└──────────────┬──────────────────────────────────────────┘
               │ consumed by
┌──────────────▼──────────────────────────────────────────┐
│  internal/executor/tool.go  (replaces Phase 5 stub)     │
│   ToolExecutor.Execute → ToolRuntime.Invoke             │
└─────────────────────────────────────────────────────────┘
               │ tested with
┌──────────────▼──────────────────────────────────────────┐
│  cmd/tools/          (reference tool binaries)          │
│   echo/              — stdio echo                       │
│   fail/              — always fails                     │
│   slow/              — configurable delay               │
│   json-emitter/      — structured JSON output           │
│   jsonrpc-server/    — JSON-RPC 2.0 test server         │
│   mcp-server/        — MCP test server                  │
│   stub/              — generic builtin stub             │
└─────────────────────────────────────────────────────────┘
```

### Transport Abstraction

All three transports (stdio, JSON-RPC, MCP) implement a single `ToolTransport`
interface. The `ToolRuntime` selects the appropriate transport based on the
`ToolDef.Transport.Type` field from the tool definition.

### Tool Registry

The tool registry is a multi-source catalog. Sources are scanned in discovery
order (§05 spec): builtin → project `tools/` → `requires:` packages → config
paths → extension contributions → MCP dynamic discovery. First match wins.

### ToolExecutor Wiring

The `ToolExecutor` replaces its Phase 5 stub entirely. It receives a
`ToolRuntime` at construction. On `Execute`, it:

1. Resolves template arguments from `vars`
2. Calls `ToolRuntime.Invoke(ctx, toolName, action, resolvedArgs)`
3. Maps `ToolResult` to `engine.StepResult` with captured output
4. Applies governance redaction before returning

---

## 2. Decisions

### D1: Three Transport Implementations for v2.0

**What:** Ship three transports: `stdio`, `stdio-jsonrpc`, `mcp`. No gRPC.

**Why:** The spec (§05) defines these three. gRPC is listed as an open question
(Q8 in §09) and has no runbook usage. Adding gRPC would require protobuf
compilation and a code-generation dependency — unacceptable for a tool runtime
that must work with zero external tooling.

**Constraint:** No gRPC transport in v2.0. The `ToolTransport` interface allows
future gRPC addition without breaking changes.

---

### D2: `ToolTransport` Interface in `pkg/tool`

**What:** Define the `ToolTransport` interface in `pkg/tool/` as a public,
gert-internal contract. Transport implementations live in `internal/tool/`.

**Why:** `pkg/tool` is a leaf package (no gert imports). The interface must be
importable by both `internal/executor/tool.go` and `internal/tool/*.go` without
cycles. Placing it in `pkg/tool` keeps the dependency graph acyclic.

**Constraint:** `pkg/tool` imports only stdlib and `pkg/schema`. No other gert
packages allowed.

---

### D3: stdio Transport Spawns Per-Invocation

**What:** Each `stdio` tool invocation spawns a new subprocess. The process
exits after producing output.

**Why:** §05 spec: "The binary is spawned once per invocation." No process
pooling or reuse for stdio. This is the simplest possible model and matches the
spec exactly. Persistent processes are handled by `jsonrpc` and `mcp` transports.

**Constraint:** stdio processes are ephemeral. No process caching for stdio
transport.

---

### D4: JSON-RPC and MCP Transports Use Persistent Processes

**What:** A single subprocess per tool definition, spawned on first use, kept
alive for the duration of the run. Graceful shutdown at run completion.

**Why:** §05 spec: "A single process is spawned at first use and kept alive for
the duration of the run." This is efficient for tools invoked multiple times
(e.g., `aws` in r05 with 10+ invocations). The process manager tracks all
active processes and shuts them down on run cancellation.

**Constraint:** Persistent processes are scoped to a single run. No cross-run
process reuse.

---

### D5: Builtin Tools Are Embedded Stubs in v2.0

**What:** The 8 builtin tools referenced by r01 and r05 (slack-notify,
pagerduty-notify, alertmanager-notify, aws, okta, palo-alto, splunk, email-notify)
are implemented as `.tool.yaml` definitions pointing to a single `gert-test-stub`
binary that echoes input and exits 0.

**Why:** Real integrations require API keys, network access, and per-vendor
SDKs. This bloats the build and makes tests non-hermetic. Stubs satisfy tool
resolution, transport testing, and runbook execution without external
dependencies. Real implementations are a v2.1 deliverable.

**Constraint:** No real API calls from builtin tools in v2.0. Stubs return
deterministic JSON output.

---

### D6: Reference Tools Are Go Binaries

**What:** All 7 reference tool binaries are implemented in Go under `v2/cmd/tools/`.

**Why:** Cross-platform (Windows Tier 2 support). Shell scripts require bash
and complicate Windows CI. Go binaries are self-contained, fast to compile, and
consistent across platforms. Barbara's gap analysis recommends Go binaries.

**Constraint:** Reference tools have zero external dependencies (stdlib only).

---

### D7: MCP Transport Implements Minimal Compliance

**What:** MCP transport supports: `initialize`, `initialized`, `tools/list`,
`tools/call`, `tools/cancel`. No MCP resources, prompts, or sampling.

**Why:** Gert uses MCP exclusively for tool discovery and invocation. The full
MCP spec includes capabilities gert doesn't consume. Minimal compliance reduces
implementation surface and test burden while covering the critical path.

**Constraint:** MCP transport does not support `resources/*`, `prompts/*`, or
`sampling/*` methods.

---

### D8: `pkg/tool.ToolRegistry` Replaces `pkg/planner.ToolRegistry`

**What:** The planner's `ToolRegistry` (name+action lookup) is insufficient for
the tool runtime, which needs full catalog access, transport config, and
lifecycle management. Phase 6 introduces `pkg/tool.ToolRegistry` as the
canonical registry interface. The planner's interface is adapted via a thin
wrapper.

**Why:** The planner only needs "does this tool+action exist?" for validation.
The runtime needs "give me the full ToolDef so I can spawn the right transport."
A single registry serves both consumers: planner calls `Lookup(name, action)`
(validated), runtime calls `Get(name)` (full definition).

**Constraint:** `pkg/planner.ToolRegistry` remains stable. `pkg/tool.ToolRegistry`
is additive. No breaking change to Phase 2 code.

---

## 3. Package Map

```
v2/
├── pkg/
│   ├── tool/                    ← Public interfaces (leaf package)
│   │   ├── doc.go               ← Package documentation
│   │   ├── transport.go         ← ToolTransport interface, TransportConfig
│   │   ├── runtime.go           ← ToolRuntime interface, ToolResult
│   │   └── registry.go          ← ToolRegistry interface (NEW)
│   ├── schema/
│   │   └── tool.go              ← ToolDef, ToolAction, ArgDef (existing)
│   └── engine/
│       ├── executor.go          ← StepExecutor interface (existing)
│       └── engine.go            ← EngineConfig (add ToolRuntime field)
│
├── internal/
│   ├── tool/                    ← Transport implementations (NEW)
│   │   ├── doc.go               ← Package documentation
│   │   ├── stdio.go             ← StdioTransport
│   │   ├── jsonrpc.go           ← JSONRPCTransport
│   │   ├── mcp.go               ← MCPTransport
│   │   ├── process.go           ← ProcessManager (lifecycle)
│   │   ├── registry.go          ← MultiSourceRegistry
│   │   ├── runtime.go           ← ConcreteToolRuntime
│   │   ├── stdio_test.go
│   │   ├── jsonrpc_test.go
│   │   ├── mcp_test.go
│   │   ├── process_test.go
│   │   ├── registry_test.go
│   │   └── runtime_test.go
│   └── executor/
│       └── tool.go              ← ToolExecutor (REPLACE stub)
│
├── cmd/tools/                   ← Reference tool binaries (NEW)
│   ├── echo/main.go             ← stdio echo
│   ├── fail/main.go             ← always fails
│   ├── slow/main.go             ← configurable delay
│   ├── json-emitter/main.go     ← structured JSON output
│   ├── jsonrpc-server/main.go   ← JSON-RPC 2.0 test server
│   ├── mcp-server/main.go       ← MCP test server
│   └── stub/main.go             ← generic builtin stub
│
└── testdata/tools/              ← Tool definitions for tests (NEW)
    ├── echo.tool.yaml
    ├── fail.tool.yaml
    ├── slow.tool.yaml
    ├── json-emitter.tool.yaml
    ├── jsonrpc-test.tool.yaml
    ├── mcp-test.tool.yaml       ← MCP server declaration
    └── builtin/                 ← Builtin stubs
        ├── slack-notify.tool.yaml
        ├── pagerduty-notify.tool.yaml
        ├── alertmanager-notify.tool.yaml
        ├── aws.tool.yaml
        ├── okta.tool.yaml
        ├── palo-alto.tool.yaml
        ├── splunk.tool.yaml
        └── email-notify.tool.yaml
```

---

## 4. Interface Contracts

### 4.1 ToolTransport Interface

```go
// pkg/tool/transport.go

package tool

import (
    "context"
    "github.com/ormasoftchile/gert/v2/pkg/schema"
)

// ToolTransport abstracts the communication protocol with a tool process.
// Each transport implementation handles process lifecycle and message framing.
type ToolTransport interface {
    // Invoke sends a tool action request and returns the result.
    // For stdio: spawns process, captures stdout/stderr, returns on exit.
    // For jsonrpc: sends JSON-RPC request to persistent process, reads response.
    // For mcp: sends tools/call to MCP server, reads response.
    Invoke(ctx context.Context, def *schema.ToolDef, action string, args map[string]any, ictx InvocationContext) (*ToolResult, error)

    // Close shuts down any persistent processes managed by this transport.
    // For stdio: no-op (processes are per-invocation).
    // For jsonrpc: sends shutdown method, then SIGTERM.
    // For mcp: sends shutdown, then SIGTERM.
    Close(ctx context.Context) error
}

// InvocationContext carries run-level metadata for tool invocations.
// Injected as GERT_* env vars (stdio) or context object (jsonrpc/mcp).
type InvocationContext struct {
    RunID    string
    StepID   string
    TraceID  string
    UserID   string
    Mode     string // "real" | "dry-run" | "replay"
    Attempt  int
}
```

### 4.2 ToolRegistry Interface

```go
// pkg/tool/registry.go (NEW FILE)

package tool

import (
    "context"
    "github.com/ormasoftchile/gert/v2/pkg/schema"
)

// ToolRegistry provides discovery and lookup of tool definitions.
// Implementations scan multiple sources in priority order.
type ToolRegistry interface {
    // Get returns the full tool definition by name.
    // Returns ErrToolNotFound if no tool matches.
    Get(ctx context.Context, name string) (*schema.ToolDef, error)

    // Lookup returns the tool definition if the named action exists.
    // Returns ErrToolNotFound if no tool matches.
    // Returns ErrActionNotFound if the tool exists but lacks the action.
    Lookup(ctx context.Context, name string, action string) (*schema.ToolDef, error)

    // List returns all registered tool names.
    List(ctx context.Context) []string

    // Register adds a tool definition to the registry.
    // If a tool with the same name already exists, the first registration wins
    // and a warning is logged.
    Register(def *schema.ToolDef) error
}

// Sentinel errors for tool registry.
var (
    ErrToolNotFound   = errors.New("tool: not found")
    ErrActionNotFound = errors.New("tool: action not found")
)
```

### 4.3 ToolRuntime Interface (Updated)

```go
// pkg/tool/runtime.go (UPDATED — add InvocationContext)

package tool

import "context"

// ToolRuntime invokes tool actions and returns their results.
// The runtime selects the appropriate transport based on the tool definition.
type ToolRuntime interface {
    // Invoke calls the named tool action with the given arguments.
    // ctx carries deadline and cancellation.
    // Returns ToolResult on success (including non-zero exit for stdio).
    // Returns error only for infrastructure failures.
    Invoke(ctx context.Context, toolName string, action string, args map[string]any, ictx InvocationContext) (*ToolResult, error)

    // Close shuts down all persistent tool processes.
    Close(ctx context.Context) error
}

// ToolResult is the outcome of a tool invocation. (existing — unchanged)
type ToolResult struct {
    Stdout     string
    Stderr     string
    ExitCode   int
    JSONOutput []byte
    DurationMs int64
}
```

### 4.4 ToolExecutor.Execute (Updated)

```go
// internal/executor/tool.go (REPLACES Phase 5 stub)

package executor

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/ormasoftchile/gert/v2/pkg/engine"
    "github.com/ormasoftchile/gert/v2/pkg/expr"
    "github.com/ormasoftchile/gert/v2/pkg/schema"
    "github.com/ormasoftchile/gert/v2/pkg/tool"
)

type ToolExecutor struct {
    evaluator expr.Evaluator
    runtime   tool.ToolRuntime
}

func NewToolExecutor(eval expr.Evaluator, runtime tool.ToolRuntime) *ToolExecutor {
    return &ToolExecutor{evaluator: eval, runtime: runtime}
}

func (e *ToolExecutor) Execute(ctx context.Context, step engine.ResolvedStep, vars map[string]any) (*engine.StepResult, error) {
    spec, ok := step.Spec.(*schema.ToolCallSpec)
    if !ok || spec == nil {
        return nil, fmt.Errorf("tool executor: invalid spec for step %s", step.ID)
    }

    // 1. Resolve template arguments.
    args := make(map[string]any)
    for k, v := range spec.Tool.Args {
        if s, ok := v.(string); ok {
            resolved, err := resolveTemplate(e.evaluator, s, vars)
            if err != nil {
                return nil, err
            }
            args[k] = resolved
            continue
        }
        args[k] = v
    }

    // 2. Build invocation context from run vars.
    ictx := tool.InvocationContext{
        RunID:   stringFromVars(vars, "gert.run_id"),
        StepID:  step.ID,
        TraceID: stringFromVars(vars, "gert.trace_id"),
        UserID:  stringFromVars(vars, "gert.actor"),
        Mode:    stringFromVars(vars, "gert.mode"),
        Attempt: 1,
    }

    // 3. Invoke the tool.
    start := time.Now()
    result, err := e.runtime.Invoke(ctx, spec.Tool.Name, spec.Tool.Action, args, ictx)
    elapsed := time.Since(start)

    if err != nil {
        r := newResult(step, engine.StepStatusFailed)
        r.Error = fmt.Errorf("tool %s/%s: %w", spec.Tool.Name, spec.Tool.Action, err)
        r.DurationMs = elapsed.Milliseconds()
        return r, nil // step failure, not infrastructure error
    }

    // 4. Map ToolResult to StepResult.
    status := engine.StepStatusCompleted
    if result.ExitCode != 0 {
        status = engine.StepStatusFailed
    }

    sr := newResult(step, status)
    sr.DurationMs = elapsed.Milliseconds()
    sr.Output["stdout"] = result.Stdout
    sr.Output["stderr"] = result.Stderr
    sr.Output["exit_code"] = result.ExitCode
    sr.Output["duration_ms"] = result.DurationMs

    // 5. Parse JSON output if available.
    if len(result.JSONOutput) > 0 {
        var parsed map[string]any
        if json.Unmarshal(result.JSONOutput, &parsed) == nil {
            sr.Output["json"] = parsed
        }
    }

    // 6. Apply capture mappings.
    if step.Capture != nil {
        for varName, source := range step.Capture {
            switch source {
            case "stdout":
                sr.Vars[varName] = result.Stdout
            case "stderr":
                sr.Vars[varName] = result.Stderr
            case "exitCode":
                sr.Vars[varName] = result.ExitCode
            default:
                // Dot-path into JSON output: "stdout.incident.id"
                if val := extractJSONPath(result.JSONOutput, source); val != nil {
                    sr.Vars[varName] = val
                }
            }
        }
    }

    return sr, nil
}
```

### 4.5 ProcessManager (internal)

```go
// internal/tool/process.go

package tool

import (
    "context"
    "os/exec"
    "sync"
)

// ProcessManager tracks persistent tool processes (jsonrpc, mcp).
// Scoped to a single run. Handles startup, readiness, and shutdown.
type ProcessManager struct {
    mu        sync.Mutex
    processes map[string]*managedProcess // keyed by tool name
}

type managedProcess struct {
    cmd    *exec.Cmd
    stdin  io.WriteCloser
    stdout io.ReadCloser
    stderr io.ReadCloser
    done   chan struct{}
}

// GetOrStart returns an existing process for the tool, or starts a new one.
// Blocks until the ready-pattern is matched on stdout/stderr.
func (pm *ProcessManager) GetOrStart(ctx context.Context, def *schema.ToolDef) (*managedProcess, error)

// Shutdown sends the shutdown method (if configured), then SIGTERM,
// then SIGKILL after 5s grace period. Called at run end.
func (pm *ProcessManager) Shutdown(ctx context.Context) error
```

---

## 5. Reference Tools Spec

### 5.1 `echo` — Stdio Echo

| Field | Value |
|-------|-------|
| **Binary** | `v2/cmd/tools/echo/main.go` |
| **Transport** | stdio |
| **Input** | `--message <string>` via argv |
| **Output** | Echoes message to stdout, exits 0 |
| **Error** | Never fails (exit 0 always) |
| **Lines** | ~30 |

**Behavior:** `echo --message "hello"` → stdout: `hello\n`, exit 0.

---

### 5.2 `fail` — Always Fails

| Field | Value |
|-------|-------|
| **Binary** | `v2/cmd/tools/fail/main.go` |
| **Transport** | stdio |
| **Actions** | `fail` (exit code N), `timeout` (sleep forever) |
| **Input** | `--exit-code <int>` or `--sleep <seconds>` |
| **Output** | stderr: "error: forced failure", exit N |
| **Error** | Always non-zero exit |
| **Lines** | ~80 |

**Behavior:**
- `fail --exit-code 1` → stderr: `error: forced failure\n`, exit 1
- `fail --sleep 9999` → blocks indefinitely (tests timeout/cancellation)

---

### 5.3 `slow` — Configurable Delay

| Field | Value |
|-------|-------|
| **Binary** | `v2/cmd/tools/slow/main.go` |
| **Transport** | stdio |
| **Input** | `--delay <seconds>` |
| **Output** | stdout: `{"delayed_seconds": N}`, exit 0 |
| **Error** | Respects context cancellation (exits on SIGTERM) |
| **Lines** | ~50 |

**Behavior:** `slow --delay 3` → sleeps 3s, stdout: `{"delayed_seconds":3}\n`, exit 0.

---

### 5.4 `json-emitter` — Structured JSON Output

| Field | Value |
|-------|-------|
| **Binary** | `v2/cmd/tools/json-emitter/main.go` |
| **Transport** | stdio |
| **Input** | `--key <string> --value <string>` |
| **Output** | stdout: `{"<key>": "<value>"}`, exit 0 |
| **Error** | Missing args → exit 1 with usage to stderr |
| **Lines** | ~60 |

**Behavior:** `json-emitter --key status --value running` → `{"status":"running"}\n`, exit 0.

---

### 5.5 `jsonrpc-server` — JSON-RPC 2.0 Test Server

| Field | Value |
|-------|-------|
| **Binary** | `v2/cmd/tools/jsonrpc-server/main.go` |
| **Transport** | stdio-jsonrpc |
| **Startup** | `--server-mode`, prints `ready\n` to stderr |
| **Actions** | `ping`, `add`, `error`, `slow` |
| **Shutdown** | `shutdown` method → clean exit |
| **Lines** | ~250 |

**Methods:**
- `ping` → `{"result": {"message": "pong"}}`
- `add(a, b)` → `{"result": {"sum": a+b}}`
- `error(code)` → `{"error": {"code": code, "message": "forced error"}}`
- `slow(seconds)` → delays, then returns `{"result": {"delayed": seconds}}`

**Startup sequence:** Reads `--server-mode` flag, prints `ready\n` to stderr,
enters JSON-RPC read loop on stdin, writes responses to stdout. One JSON object
per line (newline-delimited JSON).

---

### 5.6 `mcp-server` — MCP Test Server

| Field | Value |
|-------|-------|
| **Binary** | `v2/cmd/tools/mcp-server/main.go` |
| **Transport** | mcp (over stdio) |
| **Startup** | `--stdio`, MCP `initialize` handshake |
| **Tools** | `mcp-ping`, `mcp-lookup`, `mcp-fail` |
| **Lines** | ~350 |

**MCP Handshake:**
1. Receives `initialize` → responds with server info + capabilities
2. Receives `initialized` notification → ready for requests

**tools/list response:**
```json
{
  "tools": [
    {"name": "mcp-ping", "description": "Returns pong", "inputSchema": {}},
    {"name": "mcp-lookup", "description": "Lookup by ID", "inputSchema": {"type":"object","properties":{"id":{"type":"string"}}}},
    {"name": "mcp-fail", "description": "Always fails", "inputSchema": {}}
  ]
}
```

**tools/call behavior:**
- `mcp-ping` → `{"content": [{"type": "text", "text": "pong"}]}`
- `mcp-lookup(id)` → `{"content": [{"type": "text", "text": "{\"id\":\"<id>\",\"status\":\"found\"}"}]}`
- `mcp-fail` → `{"isError": true, "content": [{"type": "text", "text": "forced failure"}]}`

---

### 5.7 `stub` — Generic Builtin Stub

| Field | Value |
|-------|-------|
| **Binary** | `v2/cmd/tools/stub/main.go` |
| **Transport** | stdio |
| **Input** | `--action <name> [--key value ...]` |
| **Output** | JSON: `{"tool":"<binary>","action":"<name>","args":{...},"status":"ok"}` |
| **Error** | Never fails (always exit 0) |
| **Lines** | ~80 |

**Behavior:** Accepts any arguments, echoes them as structured JSON. Used as the
binary for all 8 builtin tool stubs. Deterministic output for golden trace testing.

---

## 6. Built-in Tool Stubs

All builtin stubs use the `gert-test-stub` binary. Each `.tool.yaml` declares
the specific actions referenced by r01 and r05 runbooks.

### 6.1 slack-notify

| Field | Value |
|-------|-------|
| **Referenced by** | r01, r05 |
| **Transport** | stdio |
| **Actions** | `send-message` |
| **Stub output** | `{"tool":"slack-notify","action":"send-message","channel":"<channel>","message":"<msg>","status":"ok"}` |

### 6.2 pagerduty-notify

| Field | Value |
|-------|-------|
| **Referenced by** | r01, r05 |
| **Transport** | stdio |
| **Actions** | `create-incident`, `resolve-incident` |
| **Stub output** | `{"tool":"pagerduty-notify","action":"<action>","incident_id":"PD-STUB-001","status":"ok"}` |

### 6.3 alertmanager-notify

| Field | Value |
|-------|-------|
| **Referenced by** | r01 |
| **Transport** | stdio |
| **Actions** | `resolve` |
| **Stub output** | `{"tool":"alertmanager-notify","action":"resolve","alert_id":"<id>","status":"ok"}` |

### 6.4 aws

| Field | Value |
|-------|-------|
| **Referenced by** | r05 |
| **Transport** | stdio (stub; real would be stdio-jsonrpc) |
| **Actions** | `iam-revoke`, `s3-block`, `cloudtrail-query`, `guardduty-findings` |
| **Stub output** | `{"tool":"aws","action":"<action>","args":{...},"status":"ok","request_id":"AWS-STUB-001"}` |

### 6.5 okta

| Field | Value |
|-------|-------|
| **Referenced by** | r05 |
| **Transport** | stdio (stub) |
| **Actions** | `disable-user`, `revoke-sessions`, `list-sessions` |
| **Stub output** | `{"tool":"okta","action":"<action>","user_id":"<id>","status":"ok"}` |

### 6.6 palo-alto

| Field | Value |
|-------|-------|
| **Referenced by** | r05 |
| **Transport** | stdio (stub) |
| **Actions** | `block-ip`, `create-rule` |
| **Stub output** | `{"tool":"palo-alto","action":"<action>","ip":"<ip>","rule_id":"PA-STUB-001","status":"ok"}` |

### 6.7 splunk

| Field | Value |
|-------|-------|
| **Referenced by** | r05 |
| **Transport** | stdio (stub) |
| **Actions** | `query`, `export-results` |
| **Stub output** | `{"tool":"splunk","action":"<action>","query":"<q>","results":[],"status":"ok"}` |

### 6.8 email-notify

| Field | Value |
|-------|-------|
| **Referenced by** | r05 |
| **Transport** | stdio |
| **Actions** | `send` |
| **Stub output** | `{"tool":"email-notify","action":"send","to":"<addr>","subject":"<subj>","status":"ok"}` |

---

## 7. Test Plan (32 tests)

### 7.1 Transport Tests (12 tests)

**stdio transport (`internal/tool/stdio_test.go`):**

| # | Test | Description |
|---|------|-------------|
| T01 | `TestStdio_Echo` | Invokes echo tool, verifies stdout capture |
| T02 | `TestStdio_NonZeroExit` | Invokes fail tool, verifies ExitCode and stderr |
| T03 | `TestStdio_Timeout` | Invokes slow tool with tight timeout, verifies context cancellation |
| T04 | `TestStdio_JSONOutput` | Invokes json-emitter, verifies JSONOutput populated |
| T05 | `TestStdio_EnvInjection` | Verifies GERT_* env vars passed to subprocess |
| T06 | `TestStdio_TemplateArgs` | Verifies argv template rendering with vars |

**JSON-RPC transport (`internal/tool/jsonrpc_test.go`):**

| # | Test | Description |
|---|------|-------------|
| T07 | `TestJSONRPC_Ping` | Starts jsonrpc-server, sends ping, verifies pong |
| T08 | `TestJSONRPC_Add` | Sends add(2,3), verifies sum=5 |
| T09 | `TestJSONRPC_Error` | Sends error action, verifies JSON-RPC error envelope |
| T10 | `TestJSONRPC_Shutdown` | Verifies graceful shutdown via shutdown method |

**MCP transport (`internal/tool/mcp_test.go`):**

| # | Test | Description |
|---|------|-------------|
| T11 | `TestMCP_DiscoverTools` | Handshake + tools/list, verifies 3 tools discovered |
| T12 | `TestMCP_Invoke` | tools/call mcp-ping, verifies text content "pong" |

---

### 7.2 Registry Tests (6 tests)

**`internal/tool/registry_test.go`:**

| # | Test | Description |
|---|------|-------------|
| T13 | `TestRegistry_BuiltinLookup` | Registers builtin, verifies Lookup finds it |
| T14 | `TestRegistry_ProjectOverridesBuiltin` | Project tool shadows builtin with same name |
| T15 | `TestRegistry_ActionNotFound` | Tool exists but action missing → ErrActionNotFound |
| T16 | `TestRegistry_ToolNotFound` | Unknown tool → ErrToolNotFound |
| T17 | `TestRegistry_MCPDynamicDiscovery` | MCP server tools registered via tools/list |
| T18 | `TestRegistry_ListAll` | List() returns all registered tool names |

---

### 7.3 Executor Integration Tests (8 tests)

**`internal/executor/tool_test.go`:**

| # | Test | Description |
|---|------|-------------|
| T19 | `TestToolExecutor_StdioSuccess` | Full execute: resolve args → invoke echo → capture stdout |
| T20 | `TestToolExecutor_StdioFailure` | Invoke fail → StepStatusFailed, error in output |
| T21 | `TestToolExecutor_JSONCapture` | Invoke json-emitter → JSON parsed, dot-path capture |
| T22 | `TestToolExecutor_Timeout` | Invoke slow with 1s timeout → context deadline exceeded |
| T23 | `TestToolExecutor_InvalidSpec` | Non-ToolCallSpec → error |
| T24 | `TestToolExecutor_TemplateResolution` | Args with `{{ .var }}` resolved from vars |
| T25 | `TestToolExecutor_CaptureMapping` | stdout/stderr/exitCode/json-path capture |
| T26 | `TestToolExecutor_DryRunMode` | dry-run mode passed in InvocationContext |

---

### 7.4 Reference Tool Tests (6 tests)

**`cmd/tools/*_test.go` (binary-level tests):**

| # | Test | Description |
|---|------|-------------|
| T27 | `TestEchoBinary` | Build and run echo, verify output |
| T28 | `TestFailBinary` | Build and run fail --exit-code 42, verify exit code |
| T29 | `TestSlowBinary` | Build and run slow --delay 1, verify timing |
| T30 | `TestJSONEmitterBinary` | Build and run json-emitter, verify JSON output |
| T31 | `TestJSONRPCServerBinary` | Build, start, send ping, verify pong, shutdown |
| T32 | `TestStubBinary` | Build and run stub --action test, verify JSON echo |

---

## 8. Import Constraint Analysis

### Dependency Graph

```
pkg/schema       ← leaf (no gert imports)
    ↑
pkg/tool         ← imports pkg/schema only
    ↑
pkg/engine       ← imports pkg/tool, pkg/schema, pkg/eventbus, etc.
    ↑
internal/tool    ← imports pkg/tool, pkg/schema
    ↑
internal/executor ← imports pkg/engine, pkg/expr, pkg/schema, pkg/tool
    ↑
internal/engine  ← imports internal/executor, pkg/engine, pkg/tool
```

### Cycle Prevention Rules

1. **`pkg/tool` → `pkg/schema`**: ToolDef, ToolAction, ArgDef types referenced.
   `pkg/tool` MUST NOT import `pkg/engine` or `pkg/planner` or any `internal/` package.

2. **`pkg/engine` → `pkg/tool`**: EngineConfig gains `ToolRuntime tool.ToolRuntime`
   field. This is a one-way dependency. `pkg/tool` does NOT import `pkg/engine`.

3. **`internal/executor` → `pkg/tool`**: ToolExecutor uses `tool.ToolRuntime`.
   Already imports `pkg/engine` and `pkg/schema`. Adding `pkg/tool` is safe
   (no cycle: `internal/executor` → `pkg/tool` → `pkg/schema`).

4. **`internal/tool` → `pkg/tool`**: Implements `ToolTransport` and `ToolRegistry`.
   Also imports `pkg/schema` for `ToolDef`. Does NOT import `pkg/engine` or
   `internal/executor`.

5. **`internal/engine` → `pkg/tool`**: Passes `ToolRuntime` from EngineConfig
   to `NewToolExecutor`. This is the wiring point.

### Potential Cycle: `pkg/planner` ↔ `pkg/tool`

Both packages define tool-related interfaces. `pkg/planner.ToolRegistry` has
`Lookup(ctx, name, action) (*schema.ToolDef, error)`. `pkg/tool.ToolRegistry`
also has `Lookup`. Neither imports the other — they are independent contracts.
`internal/tool.MultiSourceRegistry` implements both via embedding or adaptation.
No cycle.

### `EngineConfig` Changes

```go
// pkg/engine/engine.go — add one field:

type EngineConfig struct {
    // ... existing fields ...

    // ToolRuntime invokes external tools via stdio/jsonrpc/mcp transports.
    // Required when tool steps are in the execution plan.
    // Optional: if nil, tool steps fail with ErrNotImplemented.
    ToolRuntime tool.ToolRuntime
}
```

This field is **optional** (not in `Validate()` required list) because not all
runbooks use tool steps. The `NewDefaultRegistry` constructor in
`internal/executor/registry.go` is updated to accept `tool.ToolRuntime` and pass
it to `NewToolExecutor`.

### `RegistryConfig` Changes

```go
// internal/executor/registry.go — add ToolRuntime field:

type RegistryConfig struct {
    // ... existing fields ...
    ToolRuntime tool.ToolRuntime
}

func NewDefaultRegistry(cfg RegistryConfig) *MapRegistry {
    r := NewMapRegistry()
    // ... existing registrations ...
    r.Register("tool", NewToolExecutor(cfg.Evaluator, cfg.ToolRuntime))
    // ...
    return r
}
```

---

## 9. Open Questions

### Q1: Should `pkg/tool.ToolRegistry` replace `pkg/planner.ToolRegistry`?

**Recommendation:** No. Keep both. The planner interface is minimal (Lookup only)
and used during plan time. The tool runtime registry is richer (Get, List,
Register) and used during execution. `internal/tool.MultiSourceRegistry`
implements both. This avoids changing the planner's public interface.

**Requires:** No user input. Decided.

### Q2: Should MCP transport support HTTP/SSE in v2.0?

**Recommendation:** No. MCP over stdio only. HTTP/SSE transport is a v2.1
feature. The spec (§05) mentions MCP over stdio; HTTP/SSE is not referenced in
any runbook. This reduces scope significantly.

**Requires:** Confirmation from Cristian. Low risk to defer.

### Q3: Where do `.tool.yaml` definitions for builtin stubs get embedded?

**Recommendation:** Use `//go:embed` in `internal/tool/registry.go` to embed
`testdata/tools/builtin/*.tool.yaml` at compile time. The `MultiSourceRegistry`
loads embedded definitions as the lowest-priority source.

**Requires:** Brian's implementation decision. `//go:embed` is stdlib, no deps.

### Q4: Should `ToolExecutor` apply governance redaction before returning?

**Recommendation:** Yes, but as a post-processing step in the engine loop, not
in the executor. The engine already applies governance checks pre-step; post-step
redaction should follow the same pattern. The executor returns raw output; the
engine applies `governance.Redactor.Redact()` before trace write.

**Requires:** Alignment with Phase 4 governance evaluator. Non-blocking for
Phase 6 — executor returns raw, engine redacts.

### Q5: Concurrency model for JSON-RPC requests to the same persistent process

**Recommendation:** Serial. One in-flight JSON-RPC request per process at a time.
The JSON-RPC spec supports concurrent requests via ID correlation, but
implementing a request multiplexer adds complexity. Serial invocation is
sufficient for gert's step-at-a-time execution model. If parallel branches both
invoke the same JSON-RPC tool, each branch gets its own process instance.

**Requires:** No user input. Decided.

### Q6: Tool definition for r05's non-builtin tool references

r05 references 5 tools with `source: tool://auth-service`,
`tool://internal-api`, `tool://credential-revoker`, `tool://log-exporter`,
`tool://incident-tracker`. These are project-level tools (not builtins).

**Recommendation:** Create minimal `.tool.yaml` stubs in `testdata/tools/project/`
for these 5 tools. They use the same `gert-test-stub` binary. This is a Barbara
(Integrations) deliverable for test fixture completeness.

**Requires:** Barbara to create fixture tool definitions.

---

## Appendix A: Transport Protocol Details

### A.1 stdio Wire Format

```
Host:    exec.Command(binary, ...resolvedArgv)
         env: GERT_RUN_ID=xxx, GERT_STEP_ID=xxx, GERT_TRACE_ID=xxx, GERT_MODE=real
Process: writes to stdout (captured), writes to stderr (captured)
         exits with code N
Host:    ExitCode=N, Stdout=captured, Stderr=captured
```

### A.2 JSON-RPC Wire Format

```
Host:    exec.Command(binary, ...startup.argv)
         waits for ready-pattern on stderr
Host → stdin:
         {"jsonrpc":"2.0","id":"1","method":"<action>","params":{"args":{...},"context":{...}}}
stdin → Host:
         {"jsonrpc":"2.0","id":"1","result":{"output":{...},"telemetry":{...}}}
Host → stdin (shutdown):
         {"jsonrpc":"2.0","id":"shutdown","method":"shutdown","params":{}}
```

### A.3 MCP Wire Format

```
Host:    exec.Command(binary, "--stdio")
Host → stdin:
         {"jsonrpc":"2.0","id":"1","method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"gert","version":"2.0.0"}}}
stdin → Host:
         {"jsonrpc":"2.0","id":"1","result":{"protocolVersion":"2025-03-26","capabilities":{"tools":{}},"serverInfo":{...}}}
Host → stdin:
         {"jsonrpc":"2.0","method":"notifications/initialized"}
Host → stdin:
         {"jsonrpc":"2.0","id":"2","method":"tools/list","params":{}}
stdin → Host:
         {"jsonrpc":"2.0","id":"2","result":{"tools":[...]}}
Host → stdin (invocation):
         {"jsonrpc":"2.0","id":"3","method":"tools/call","params":{"name":"<tool>","arguments":{...},"_gert_context":{...}}}
stdin → Host:
         {"jsonrpc":"2.0","id":"3","result":{"content":[{"type":"text","text":"..."}]}}
```

---

## Appendix B: Deliverable Summary

| # | Deliverable | Owner | Files |
|---|-------------|-------|-------|
| 1 | `pkg/tool/registry.go` | Brian | 1 new |
| 2 | `pkg/tool/runtime.go` update | Brian | 1 modified |
| 3 | `pkg/tool/transport.go` update | Brian | 1 modified |
| 4 | `pkg/engine/engine.go` update | Brian | 1 modified |
| 5 | `internal/tool/` package | Brian | 7 new |
| 6 | `internal/executor/tool.go` replace | Brian | 1 modified |
| 7 | `internal/executor/registry.go` update | Brian | 1 modified |
| 8 | `cmd/tools/` (7 binaries) | Brian | 7 new |
| 9 | `testdata/tools/` (14 definitions) | Barbara | 14 new |
| 10 | Transport tests | Brian | 3 new test files |
| 11 | Registry tests | Brian | 1 new test file |
| 12 | Executor integration tests | Brian | 1 modified test file |
| 13 | Reference tool tests | Brian | 6 new test files |

**Total:** ~7 new packages, ~35 new files, ~2500 lines of Go code, ~200 lines of YAML.

---

*Ken — Software Architect*
