# Phase 10 — Adapters

**Author:** Ken (Software Architect)  
**Date:** 2026-04-23  
**Requested by:** Cristian  
**Depends on:** Phase 9 (`gert serve`, commit 545e4b6)

---

## 1 Overview

Phase 10 closes every integration gap between the `gert serve` HTTP layer (Phase 9) and the real engine subsystems built in Phases 1–8. After Phase 9 shipped, Brian logged five placeholder components in `cmd/serve/main.go`:

| # | Placeholder | Real Implementation |
|---|-------------|---------------------|
| 1 | `TerminalInputProvider` on serve stdin | Input provider chain (env → vault → RPC) |
| 2 | `noopToolRegistry` | `internal/tool.MapRegistry` + project tool scan |
| 3 | `discardTraceWriter` | `internal/trace.JSONLWriter` (append-only JSONL) |
| 4 | `testutil.FakeEventDispatcher` | `internal/eventbus.Dispatcher` (webhook/in-proc) |
| 5 | Minimal `runSubSteps` without event emission | Engine-native substep runner via `RunHandle.Next` loop |

Phase 10 also delivers:

- **`gert run`** — standalone CLI adapter for non-served execution
- **`gert dry-run`** — wiring for dry-run mode (parse + plan + validate, skip real execution)
- **Adapter wiring harness** — shared `internal/adapter` package for building `EngineConfig` from common flags

After Phase 10, every `cmd/*` entry point constructs a fully-wired `EngineConfig` with production implementations — no fakes, no noop stubs, no discard writers.

---

## 2 Package Layout

### New files

```
v2/
├── internal/
│   ├── adapter/
│   │   ├── doc.go                # Package docs
│   │   ├── wire.go               # BuildEngineConfig(), BuildPlanner(), shared wiring
│   │   ├── wire_test.go          # Unit tests for wiring
│   │   ├── options.go            # WireOptions struct (all CLI flag values)
│   │   └── options_test.go       # Options validation tests
│   ├── eventbus/
│   │   ├── doc.go                # Package docs
│   │   ├── bus.go                # In-process EventBus implementation
│   │   ├── bus_test.go           # EventBus tests
│   │   ├── dispatcher.go         # EventDispatcher implementation (webhook + in-proc)
│   │   └── dispatcher_test.go    # EventDispatcher tests
│   └── trace/
│       ├── doc.go                # Package docs
│       ├── jsonl_writer.go       # JSONL TraceWriter implementation
│       ├── jsonl_writer_test.go  # TraceWriter tests
│       ├── multi_writer.go       # Tee writer: file + bus forwarding
│       └── multi_writer_test.go  # Multi-writer tests
├── cmd/
│   ├── serve/
│   │   └── main.go              # MODIFIED: use adapter.BuildEngineConfig
│   └── gert/
│       ├── main.go              # MODIFIED: add "run" and "dry-run" subcommands
│       ├── run.go               # gert run implementation
│       ├── dryrun.go            # gert dry-run implementation
│       └── doc.go               # MODIFIED: update usage docs
```

### Modified files

| File | Change |
|------|--------|
| `cmd/serve/main.go` | Replace all 5 placeholders with `adapter.BuildEngineConfig` |
| `cmd/gert/main.go` | Add `run` and `dry-run` subcommand dispatch |
| `pkg/eventbus/dispatcher.go` | No changes (interface is stable) |
| `pkg/trace/writer.go` | No changes (interface is stable) |

### No changes to `pkg/*`

All `pkg/*` interfaces remain untouched. Phase 10 is purely `internal/*` and `cmd/*` work — implementations wired to existing interfaces.

---

## 3 Adapter Wiring

### 3.1 The Problem

Today `cmd/serve/main.go` constructs `EngineConfig` manually with 140 lines of imperative wiring, including 5 placeholder implementations. `cmd/gert/main.go` has no execution support at all. When we add `gert run`, we'd duplicate the entire wiring block.

### 3.2 Solution: `internal/adapter.BuildEngineConfig`

A new `internal/adapter` package provides a single `BuildEngineConfig` function that constructs a fully-wired `EngineConfig` from declarative options:

```go
package adapter

// WireOptions holds all values needed to construct a production EngineConfig.
// All fields map directly to CLI flags or environment variables.
type WireOptions struct {
    // Mode is the execution mode: "real", "dry-run", "replay".
    Mode string

    // TraceDir is the directory for JSONL trace files.
    // Default: "./traces" relative to working directory.
    TraceDir string

    // ToolDirs are directories to scan for .tool.yaml files.
    // Default: ["./tools", "."].
    ToolDirs []string

    // ProviderDir is the directory for .provider.yaml files.
    // Default: ".".
    ProviderDir string

    // Actor is the identity string (from --as flag).
    Actor string

    // Interactive controls whether terminal prompts are enabled.
    // False for gert serve (RPC-driven), true for gert run (TTY).
    Interactive bool

    // Stdin/Stdout/Stderr for interactive input (nil = os defaults).
    Stdin  io.Reader
    Stdout io.Writer
    Stderr io.Writer

    // WebhookAddr is the listen address for inbound webhook events.
    // Empty string disables the webhook listener.
    // Used by EventDispatcher for wait_for_event delivery.
    WebhookAddr string

    // EventBusBufferSize is the per-subscriber channel buffer.
    // Default: 256.
    EventBusBufferSize int

    // OnEvent is an optional synchronous callback for each engine event.
    OnEvent func(engine.Event)
}

// BuildEngineConfig constructs a production-ready EngineConfig.
// The caller is responsible for calling Close() on the returned Closer
// when the engine is no longer needed (flushes traces, stops webhook listener).
func BuildEngineConfig(ctx context.Context, plat platform.Platform, opts WireOptions) (engine.EngineConfig, io.Closer, error)
```

The function:

1. Creates the **JSONL TraceWriter** — opens `{TraceDir}/{runID}.jsonl`, wraps it in a `JSONLWriter`
2. Creates the **EventBus** — in-process fan-out bus with configurable buffer
3. Creates the **MultiWriter** — tees trace events to both JSONL file and EventBus
4. Creates the **EventDispatcher** — in-process + optional webhook listener
5. Creates the **ToolRegistry** — `MapRegistry` populated by scanning `ToolDirs` for `.tool.yaml`
6. Creates the **ToolRuntime** — `DefaultToolRuntime` backed by the registry
7. Creates the **InputProvider chain** — env → vault → prompt (if interactive) or env → vault → error (if non-interactive)
8. Creates the **PromptProvider** — `TerminalInputProvider` if interactive, nil otherwise
9. Creates the **GovernanceEvaluator** — loaded from plan governance policy
10. Creates the **ApprovalGate** — terminal prompt if interactive, noop otherwise
11. Creates the **Evaluator** and **ConditionEvaluator** — template + infix expression evaluators
12. Creates the **ExecutorRegistry** — `MapRegistry` with all Phase 5 executors, using a substep runner that emits engine events

Returns an `io.Closer` that closes the trace writer, stops the webhook listener, and drains the event bus.

### 3.3 `cmd/serve/main.go` After Phase 10

```go
func main() {
    addr := flag.String("addr", ":7778", "listen address")
    traceDir := flag.String("trace-dir", "./traces", "trace output directory")
    webhookAddr := flag.String("webhook-addr", "", "webhook listener for wait_for_event")
    flag.Parse()

    plat := platform.Real()
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    ecfg, closer, err := adapter.BuildEngineConfig(ctx, plat, adapter.WireOptions{
        Mode:        "real",
        TraceDir:    *traceDir,
        Interactive: false,  // serve is RPC-driven, not TTY
        WebhookAddr: *webhookAddr,
    })
    if err != nil {
        log.Fatalf("gert serve: %v", err)
    }
    defer closer.Close()

    parserImpl, _ := internalparser.New(plat)
    plannerImpl := adapter.BuildPlanner(plat, parserImpl)

    cfg := servepkg.ServerConfig{
        Addr:         *addr,
        EngineConfig: ecfg,
        EngineFactory: func(cfg engine.EngineConfig) engine.Engine {
            return internalengine.New(cfg)
        },
        Parser:  parserImpl,
        Planner: plannerImpl,
    }

    srv, err := serve.NewServer(cfg)
    if err != nil {
        log.Fatalf("gert serve: %v", err)
    }
    fmt.Fprintf(os.Stderr, "gert serve listening on %s\n", cfg.Addr)
    if err := srv.Start(ctx); err != nil {
        log.Fatalf("gert serve: %v", err)
    }
}
```

~40 lines. No placeholder implementations.

### 3.4 `cmd/gert run` After Phase 10

```go
func runRun(args []string) {
    fs := flag.NewFlagSet("run", flag.ExitOnError)
    traceDir := fs.String("trace-dir", "./traces", "trace output directory")
    actor := fs.String("as", "", "actor identity")
    mode := fs.String("mode", "real", "execution mode: real, dry-run")
    fs.Parse(args)

    if fs.NArg() < 1 {
        fmt.Fprintln(os.Stderr, "usage: gert run [flags] <runbook.yaml>")
        os.Exit(1)
    }
    runbookPath := fs.Arg(0)

    plat := platform.Real()
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    ecfg, closer, err := adapter.BuildEngineConfig(ctx, plat, adapter.WireOptions{
        Mode:        *mode,
        TraceDir:    *traceDir,
        Actor:       *actor,
        Interactive: true,  // CLI is TTY-driven
        Stdin:       os.Stdin,
        Stdout:      os.Stdout,
        Stderr:      os.Stderr,
    })
    if err != nil {
        log.Fatalf("gert run: %v", err)
    }
    defer closer.Close()

    parserImpl, _ := internalparser.New(plat)
    plannerImpl := adapter.BuildPlanner(plat, parserImpl)

    // Parse → Plan → Engine → Run loop
    parsed, err := parserImpl.Parse(ctx, runbookPath)
    if err != nil { log.Fatalf("parse: %v", err) }

    plan, err := plannerImpl.Plan(ctx, parsed)
    if err != nil { log.Fatalf("plan: %v", err) }

    eng := internalengine.New(ecfg)
    handle, err := eng.Start(ctx, plan, engine.RunOptions{
        Mode:  engine.RunMode(*mode),
        Actor: *actor,
    })
    if err != nil { log.Fatalf("start: %v", err) }

    // Drive execution: call Next() in a loop until EOF.
    for {
        result, err := handle.Next(ctx)
        if err == io.EOF {
            break
        }
        if err != nil {
            fmt.Fprintf(os.Stderr, "error: %v\n", err)
            os.Exit(1)
        }
        renderStepResult(os.Stdout, result)
    }

    state := handle.State()
    os.Exit(exitCodeForStatus(state.Status))
}
```

### 3.5 Planner Wiring Helper

```go
// BuildPlanner constructs a Planner with real RunbookLoader and ToolRegistry.
func BuildPlanner(plat platform.Platform, p parser.Parser) planner.Planner {
    loader := &fileRunbookLoader{parser: p}
    toolDirs := []string{"./tools", "."}
    registry := tool.ScanRegistry(toolDirs)
    return internalplanner.New(planner.Config{
        Loader: loader,
        Tools:  registry,
    })
}
```

---

## 4 EventDispatcher Implementation

### 4.1 The Contract (from `pkg/eventbus/dispatcher.go`)

```go
type EventDispatcher interface {
    Dispatch(ev InboundEvent) error
    Wait(ctx context.Context, stepID string, filter EventFilter, timeout time.Duration) (*InboundEvent, error)
    Cancel(stepID string, reason string)
}
```

### 4.2 Design: `internal/eventbus.Dispatcher`

The production `EventDispatcher` is a goroutine-safe correlator that matches inbound external events with steps blocked in `Wait()`.

```go
package eventbus

type Dispatcher struct {
    mu       sync.Mutex
    waiters  map[string][]waiterEntry    // channel → waiters
    pending  map[string][]InboundEvent   // channel → buffered events
    webhook  *WebhookListener            // nil if webhook disabled
}

type waiterEntry struct {
    stepID   string
    filter   eventbus.EventFilter
    delivery chan *eventbus.InboundEvent
}
```

**Two inbound event paths:**

1. **In-process dispatch** — another goroutine calls `Dispatch(ev)` directly (used in tests, invoke-child runs, programmatic event injection).

2. **Webhook HTTP** — external systems POST events to `http://{webhookAddr}/events/inbound`. The webhook handler parses the JSON body, constructs an `InboundEvent`, and calls `Dispatch()`.

### 4.3 Webhook Listener

The `WebhookListener` is an internal HTTP server started by `BuildEngineConfig` when `WireOptions.WebhookAddr != ""`:

```go
type WebhookListener struct {
    server     *http.Server
    dispatcher *Dispatcher
}

func NewWebhookListener(addr string, dispatcher *Dispatcher) *WebhookListener

// Start begins listening. Call Stop() to shut down.
func (w *WebhookListener) Start(ctx context.Context) error
func (w *WebhookListener) Stop(ctx context.Context) error
```

**Webhook request format:**

```
POST /events/inbound HTTP/1.1
Content-Type: application/json

{
    "event_id": "evt-abc123",
    "source": "github",
    "channel": "deployment",
    "payload": {
        "status": "success",
        "sha": "abc123"
    }
}
```

**Webhook response:**

- `200 OK` — event matched a waiter or buffered successfully
- `400 Bad Request` — malformed JSON
- `503 Service Unavailable` — server shutting down

### 4.4 Consume Semantics

Per locked decision #5: **first-waiter-wins dispatch**. When `Dispatch(ev)` is called:

1. Scan all waiters on the event's channel for a filter match
2. If a waiter matches, deliver the event and remove the waiter (consume)
3. If no waiter matches, buffer the event in `pending[channel]`
4. When a step calls `Wait()`, first scan `pending` for an already-buffered match, then register as a waiter

Buffer is bounded: `pending` per channel is capped at 1000 events. Oldest events are evicted if the buffer is full (log warning).

### 4.5 Timeout and Cancellation

- `Wait()` blocks until: (a) matching event arrives, (b) `timeout` elapses, or (c) `ctx` is cancelled
- On timeout: returns `eventbus.ErrEventTimeout`
- On cancellation: returns `ctx.Err()`
- `Cancel(stepID, reason)` unblocks a specific waiter with a nil event (engine handles nil → step failure)

---

## 5 JSONL Trace Writer

### 5.1 `internal/trace.JSONLWriter`

```go
package trace

type JSONLWriter struct {
    mu   sync.Mutex
    file *os.File
    enc  *json.Encoder
}

func NewJSONLWriter(path string) (*JSONLWriter, error)
func (w *JSONLWriter) Append(event trace.TraceEvent) error
func (w *JSONLWriter) Close() error
```

**Invariants:**

- Each `Append()` writes exactly one JSON line terminated by `\n`
- Writes are serialized by mutex (per locked decision #6: synchronous trace writes)
- File is opened with `O_APPEND|O_CREATE|O_WRONLY` — no read, no truncate
- `Close()` calls `file.Sync()` then `file.Close()` — flush to disk

### 5.2 `internal/trace.MultiWriter`

Tees every trace event to multiple sinks:

```go
type MultiWriter struct {
    writers []trace.TraceWriter
}

func NewMultiWriter(writers ...trace.TraceWriter) *MultiWriter
func (m *MultiWriter) Append(event trace.TraceEvent) error   // writes to all, returns first error
func (m *MultiWriter) Close() error                           // closes all, returns first error
```

Used in `BuildEngineConfig` to combine:
- `JSONLWriter` → durable audit log
- `EventBusForwarder` → in-process event fan-out (converts `TraceEvent` → `eventbus.Event` → `EventBus.Publish`)

### 5.3 Trace File Path Convention

```
{TraceDir}/{runID}.jsonl
```

The `JSONLWriter` creates the trace directory if it doesn't exist (`os.MkdirAll`).

### 5.4 HMAC Signing

When `GERT_TRACE_KEY` is set in the environment, the `JSONLWriter` computes HMAC-SHA256 over the canonical JSON of the event (all fields except `sig`) and sets `event.Signature` before writing. This matches the `TraceEvent.Signature` field in `pkg/trace/event.go`.

---

## 6 In-Process EventBus

### 6.1 `internal/eventbus.Bus`

```go
package eventbus

type Bus struct {
    mu          sync.RWMutex
    subscribers []subscriber
    nextID      uint64
}

type subscriber struct {
    id         uint64
    ch         chan eventbus.Event
    kindPrefix string
    runID      string
}
```

Implements `pkg/eventbus.EventBus`:

- `Publish(ev)` — iterates all subscribers, sends event if filter matches, drops (non-blocking) if channel full
- `Subscribe(ctx, opts)` — creates a buffered channel, returns it. Cancels subscription when `ctx` is done.
- `Unsubscribe(ch)` — removes subscriber, closes channel

### 6.2 Integration with Serve

The `internal/serve.EventBridge` already consumes `engine.Event` from `RunHandle.Events()` and fans out to SSE/WS clients. Phase 10 adds a second fan-out path: the `EventBus` receives events via the `MultiWriter` → `EventBusForwarder` chain, enabling any in-process subscriber to observe events without coupling to the serve layer.

---

## 7 `gert run` CLI Adapter

### 7.1 Entry Point

`cmd/gert/run.go` — a new file in the existing `cmd/gert` package.

### 7.2 Flag Set

| Flag | Default | Description |
|------|---------|-------------|
| `--trace-dir` | `./traces` | JSONL trace output directory |
| `--as` | `""` | Actor identity |
| `--mode` | `real` | Execution mode: `real` or `dry-run` |
| `--var KEY=VALUE` | — | Input variable (repeatable) |
| `--tool-dir` | `./tools` | Tool definition directory (repeatable) |
| `--quiet` | `false` | Suppress step output (only show final status) |

### 7.3 Execution Loop

```
parse(runbookPath)  →  plan(parsed)  →  engine.Start(plan, opts)  →  loop { handle.Next() } until EOF
```

The run loop renders each `StepResult` to stdout:

```
✓ step-1 (cli) — 320ms
✓ step-2 (tool: slack-notify) — 1.2s
✗ step-3 (assert) — FAILED: expected "ok", got "error"
```

### 7.4 Exit Codes

| Status | Exit Code |
|--------|-----------|
| `completed` | 0 |
| `failed` | 1 |
| `cancelled` | 2 |
| `dry-run completed` | 0 |

### 7.5 Interactive Input

When `--mode=real` and stdin is a TTY:
- `TerminalInputProvider` is wired for choice/decision/collector prompts
- `PromptProvider` is wired for variable input prompts
- `ApprovalGate` uses terminal prompt ("Approve step X? [y/N]")

When stdin is not a TTY (piped input):
- `Interactive` is set to `false`
- Input comes from `--var` flags and environment only
- Approval gates fail with "no interactive approval available"

---

## 8 `gert dry-run` Mode

### 8.1 What Dry-Run Does

Dry-run validates the entire execution pipeline without performing side effects:

1. **Parse** — full schema validation (same as real)
2. **Plan** — full import resolution, tool discovery, binding validation (same as real)
3. **Governance pre-flight** — evaluates all governance rules against the plan (same as real)
4. **Step simulation** — for each step:
   - CLI steps: log the command that *would* execute, skip subprocess spawn
   - Tool steps: log the tool invocation that *would* happen, skip transport call
   - Choice/decision/collector: skip prompt, use default values
   - Approval gates: auto-approve (log that approval would be required)
   - Assert steps: evaluate the assertion expression, report result
   - Include steps: resolved at plan time (no runtime behavior)
   - Parallel/iterate: simulate the structure, skip real execution

### 8.2 Implementation Strategy

Dry-run is **not** a separate code path. It uses the same engine with a `DryRunExecutorRegistry` that wraps every real executor:

```go
// DryRunExecutor wraps a real executor and skips side effects.
type DryRunExecutor struct {
    inner engine.Executor
    kind  string
}

func (d *DryRunExecutor) Execute(ctx context.Context, step engine.ResolvedStep, vars map[string]any) (*engine.StepResult, error) {
    // Emit a "dry-run" result without calling inner.Execute()
    return &engine.StepResult{
        StepID:      step.ID,
        Status:      engine.StepStatusCompleted,
        Outcome:     engine.StepOutcomeSuccess,
        Output:      map[string]any{"dry_run": true, "would_execute": d.kind},
        StartedAt:   time.Now(),
        CompletedAt: time.Now(),
    }, nil
}
```

The `DryRunExecutorRegistry` is constructed by `BuildEngineConfig` when `Mode == "dry-run"`:

```go
type DryRunExecutorRegistry struct {
    inner engine.ExecutorRegistry
}

func (r *DryRunExecutorRegistry) Lookup(kind string) engine.Executor {
    exec := r.inner.Lookup(kind)
    if exec == nil {
        return nil
    }
    return &DryRunExecutor{inner: exec, kind: kind}
}
```

### 8.3 Dry-Run Trace

Dry-run writes a trace file just like real execution. The `run/started` event has `"mode": "dry-run"`. Each `step/completed` event includes `"dry_run": true` in the payload. This allows tools to distinguish dry-run traces from real traces.

### 8.4 Dry-Run CLI Output

```
$ gert dry-run deploy.yaml
[dry-run] parse: OK
[dry-run] plan: 5 steps, 2 tools
[dry-run] governance: 3 rules evaluated, 0 denied
[dry-run] step-1 (cli): would execute `kubectl apply -f deploy.yaml`
[dry-run] step-2 (tool: slack-notify): would invoke slack-notify.send
[dry-run] step-3 (approve): would require approval (auto-approved in dry-run)
[dry-run] step-4 (assert): expression evaluated: true
[dry-run] step-5 (end): run would complete

dry-run complete: 5 steps validated, 0 issues
```

### 8.5 `gert dry-run` is an Alias

`gert dry-run <runbook>` is sugar for `gert run --mode=dry-run <runbook>`. Both paths converge in `cmd/gert/run.go` with mode detection.

---

## 9 SubStep Runner Fix

### 9.1 The Problem

`cmd/serve/main.go` defines a local `runSubSteps` function that:
- Creates a step executor loop outside the engine
- Does NOT emit trace events (step/started, step/completed)
- Does NOT evaluate governance pre-flight
- Does NOT merge variables correctly

This is used as the `SubStepRunner` callback for iterate/parallel executors.

### 9.2 The Fix

Replace the manual `runSubSteps` with the engine's own step execution. The iterate and parallel executors receive a `SubStepRunner` that uses the engine's `executeStep` method (which already emits events, evaluates governance, and merges vars).

In `BuildEngineConfig`, the `SubStepRunner` is constructed as a closure that delegates to the engine's internal step runner:

```go
// The substep runner is injected into the executor registry.
// It calls the engine's step execution path, ensuring all
// trace events and governance checks are applied to substeps.
subRunner := func(ctx context.Context, steps []schema.FlowNode, vars map[string]any) ([]*engine.StepResult, error) {
    return runSubStepsViaEngine(ctx, registry, steps, vars)
}
```

The key change: `runSubStepsViaEngine` calls `registry.Lookup(kind).Execute()` just like `runSubSteps` does today, but the `RegistryConfig` now includes the trace writer and event callback. The executor emits events via the `OnStepEvent` callback provided in the config.

**Decision D5 elaborates on the approach chosen.**

---

## 10 Integration with Existing Packages

### 10.1 Changes to `internal/*`

| Package | Change |
|---------|--------|
| `internal/adapter/` | **NEW** — wiring harness |
| `internal/eventbus/` | **NEW** — `Bus` and `Dispatcher` implementations |
| `internal/trace/` | **NEW** — `JSONLWriter` and `MultiWriter` |
| `internal/tool/registry.go` | Add `ScanDir(dir string) ([]ToolDef, error)` to load `.tool.yaml` files from disk |
| `internal/executor/registry.go` | Add `OnStepEvent` callback to `RegistryConfig` for substep trace emission |

### 10.2 Changes to `cmd/*`

| File | Change |
|------|--------|
| `cmd/serve/main.go` | Replace entire `buildEngineConfig` with `adapter.BuildEngineConfig`; remove `noopToolRegistry`, `discardTraceWriter`, `runSubSteps` |
| `cmd/gert/main.go` | Add `run` and `dry-run` subcommand dispatch |
| `cmd/gert/run.go` | **NEW** — `gert run` implementation |
| `cmd/gert/dryrun.go` | **NEW** — `gert dry-run` implementation (thin wrapper) |
| `cmd/gert/render.go` | **NEW** — `renderStepResult` and exit code helpers |
| `cmd/gert/doc.go` | Update usage text |

### 10.3 No Changes to `pkg/*`

All `pkg/*` packages remain leaf packages. Phase 10 creates only `internal/*` implementations and `cmd/*` wiring. This preserves the dependency direction invariant: `cmd → internal → pkg`.

---

## 11 Test Plan

### 11.1 `internal/trace` (5 tests)

| # | Test Name | Validates |
|---|-----------|-----------|
| 1 | `TestJSONLWriter_AppendWritesOneLine` | Each Append() produces exactly one JSON line with trailing newline |
| 2 | `TestJSONLWriter_ConcurrentAppend` | 100 goroutines appending concurrently produce 100 valid JSON lines |
| 3 | `TestJSONLWriter_CloseFlushes` | Close() syncs and closes file; subsequent Append() returns error |
| 4 | `TestJSONLWriter_HMACSigning` | When GERT_TRACE_KEY is set, `sig` field is populated with valid HMAC-SHA256 |
| 5 | `TestMultiWriter_TeesAllWriters` | MultiWriter with 3 writers delivers event to all 3; error from one doesn't stop others |

### 11.2 `internal/eventbus` (7 tests)

| # | Test Name | Validates |
|---|-----------|-----------|
| 6 | `TestBus_PublishSubscribe` | Subscriber receives published events matching its filter |
| 7 | `TestBus_KindPrefixFilter` | Subscriber with `KindPrefix: "step/"` receives step/* events, not run/* |
| 8 | `TestBus_DropOnFull` | Publishing to a full subscriber channel does not block; event is dropped |
| 9 | `TestBus_UnsubscribeClosesChannel` | After Unsubscribe, the channel is closed and no more events arrive |
| 10 | `TestDispatcher_WaitDispatchMatch` | Wait() + Dispatch() — matching event is delivered to waiting step |
| 11 | `TestDispatcher_WaitTimeout` | Wait() with 10ms timeout returns ErrEventTimeout when no event arrives |
| 12 | `TestDispatcher_BufferedEventConsumed` | Dispatch() before Wait() — event is buffered; subsequent Wait() consumes it |

### 11.3 `internal/adapter` (5 tests)

| # | Test Name | Validates |
|---|-----------|-----------|
| 13 | `TestBuildEngineConfig_AllFieldsSet` | BuildEngineConfig returns config where Validate() passes (all required fields non-nil) |
| 14 | `TestBuildEngineConfig_DryRunWrapsExecutors` | When Mode="dry-run", returned Executors wraps each executor with DryRunExecutor |
| 15 | `TestBuildEngineConfig_InteractiveFalse_NoPrompt` | When Interactive=false, PromptProvider is nil and InputProvider has no prompt fallback |
| 16 | `TestBuildEngineConfig_TraceDir_CreatesDirectory` | TraceDir that doesn't exist is created by BuildEngineConfig |
| 17 | `TestBuildEngineConfig_CloserCleansUp` | Calling Close() on the returned Closer closes trace writer, stops webhook, drains bus |

### 11.4 `cmd/gert` integration (4 tests)

| # | Test Name | Validates |
|---|-----------|-----------|
| 18 | `TestGertRun_SimpleRunbook` | `gert run` with a 2-step CLI runbook completes with exit code 0 |
| 19 | `TestGertRun_FailedStep_ExitCode1` | `gert run` with a failing step exits with code 1 |
| 20 | `TestGertDryRun_NoSideEffects` | `gert dry-run` completes without spawning subprocesses; trace file has `dry_run: true` |
| 21 | `TestGertDryRun_GovernanceReport` | `gert dry-run` with governance policy reports which steps would be denied |

### 11.5 `cmd/serve` integration (3 tests)

| # | Test Name | Validates |
|---|-----------|-----------|
| 22 | `TestServe_RealEngineConfig` | Server starts with real EngineConfig (no fakes); health endpoint returns 200 |
| 23 | `TestServe_TraceFileWritten` | After exec/start + run.next, a JSONL trace file exists in trace-dir |
| 24 | `TestServe_EventDispatcher_WebhookDelivery` | POST to webhook endpoint delivers event to a step blocked in wait_for_event |

### 11.6 Test Patterns

All tests follow established conventions:
- `testutil.Tag()` links to spec rules
- Table-driven where applicable
- `-race -count=3` clean
- No external network calls in unit tests (webhook tests use `httptest.Server`)

---

## 12 Architectural Decisions

### D1: Shared Wiring Harness (`internal/adapter`)

**Context:** `cmd/serve` and `cmd/gert` both need to construct `EngineConfig`. Without sharing, the wiring code is duplicated and diverges over time.

**Decision:** Create `internal/adapter` as a shared wiring package. Both `cmd/serve` and `cmd/gert` call `adapter.BuildEngineConfig()`.

**Alternatives considered:**
- Wire in each `cmd/` independently — leads to divergence, harder to test
- Put wiring in `pkg/` — violates leaf-package invariant (wiring imports `internal/*`)

**Consequences:**
- Single source of truth for production wiring
- Easy to test in isolation
- `cmd/*` files become thin (~40 lines each)
- Minor: `internal/adapter` imports many `internal/*` packages (acceptable for a wiring package)

---

### D2: JSONL TraceWriter in `internal/trace`

**Context:** Phase 9 used `discardTraceWriter`. We need a real implementation that writes append-only JSONL.

**Decision:** Implement `internal/trace.JSONLWriter` with mutex-serialized writes to an `os.File` opened with `O_APPEND|O_CREATE|O_WRONLY`.

**Alternatives considered:**
- Buffered writer (bufio.Writer) — risks data loss on crash; audit log must be durable
- WAL-style writer — over-engineered for append-only JSONL

**Consequences:**
- Every `Append()` is a single `write()` syscall (JSON line fits in one write)
- Mutex cost is acceptable (per decision #6: synchronous trace writes)
- HMAC signing when `GERT_TRACE_KEY` is set

---

### D3: EventDispatcher with Optional Webhook

**Context:** `wait_for_event` steps need external events. Phase 9 used `testutil.FakeEventDispatcher`. Production needs a real dispatcher with an inbound event path.

**Decision:** Implement `internal/eventbus.Dispatcher` with two inbound paths: (1) in-process `Dispatch()` and (2) optional HTTP webhook listener.

**Alternatives considered:**
- Polling-based (step polls an external API) — violates event-driven design
- Message queue (NATS, Redis Streams) — external dependency, out of scope for v2.0
- WebSocket-only inbound — harder to integrate with CI/CD webhooks (GitHub, GitLab use HTTP POST)

**Consequences:**
- Webhook listener is opt-in (only starts when `--webhook-addr` is provided)
- In-process path covers all test scenarios and child-run events
- Future phases can add message queue paths by implementing the same `Dispatch()` call
- Webhook endpoint must be protected in production (Phase 10 does not add auth; documented as out of scope, same as Phase 9 D6)

---

### D4: Dry-Run as Executor Wrapper, Not Separate Code Path

**Context:** Dry-run must validate the full pipeline without side effects. Two approaches: (a) add `if mode == dry-run` checks throughout the engine, or (b) wrap executors.

**Decision:** Wrap each `Executor` with `DryRunExecutor` that returns a synthetic `StepResult` without calling the inner executor. The engine, governance, trace writing, and event emission all run normally.

**Alternatives considered:**
- Conditional checks in engine (`if dryRun { skip }`) — spreads dry-run logic across the codebase
- Separate lightweight engine — duplicates lifecycle management

**Consequences:**
- Dry-run exercises the full pipeline: parse → plan → governance → trace → events
- Only the executor step is skipped (no subprocess spawn, no tool invocation)
- Dry-run traces are complete and auditable
- Governance denials are reported accurately (the governance evaluator runs on real policy)
- Minor: assert/end steps may benefit from running their real executor even in dry-run (Phase 10 makes assert run for real, since it has no side effects)

---

### D5: SubStep Runner Delegates to Executor Registry (Not Engine Internals)

**Context:** The `SubStepRunner` callback used by iterate/parallel executors currently bypasses the engine's event emission and governance checks. It needs to emit events.

**Decision:** Add an `OnStepEvent` callback to `executor.RegistryConfig`. The substep runner calls `executor.Execute()` as before, but the executor emits events via the callback. The engine sets this callback to its own `emitEvent` function.

**Alternatives considered:**
- Have substeps call `RunHandle.Next()` — breaks the execution model (Next advances the plan index, substeps don't live in the plan)
- Pass the entire engine into the executor — creates import cycle between `internal/engine` and `internal/executor`
- Make substep runner part of the engine — already is, but the event callback was missing

**Consequences:**
- Substeps emit `step/started` and `step/completed` events
- Governance pre-flight is evaluated for substeps (if evaluator is configured)
- No import cycle: `internal/executor` calls a `func(Event)` callback, not an engine method
- Trace file includes substep events (previously missing)

---

### D6: Input Provider Chain Order

**Context:** `gert run` (interactive) and `gert serve` (non-interactive) need different input resolution strategies.

**Decision:**
- **Interactive** (`gert run` with TTY): env → vault → static (from `--var` flags) → terminal prompt
- **Non-interactive** (`gert serve`, piped stdin): env → vault → static → error

The chain is constructed by `BuildEngineConfig` based on `WireOptions.Interactive`.

**Alternatives considered:**
- Always include terminal prompt (fails when stdin is a pipe)
- RPC-driven input provider for serve (deferred to Phase 11; serve currently uses `input/prompted` + `input/received` events)

**Consequences:**
- `gert run` in a TTY provides full interactive experience
- `gert run` in CI (piped) fails fast on unresolved inputs (no hung prompt)
- `gert serve` never blocks on stdin
- Future: RPC-driven input provider for serve (adapter sends `input/prompted` event, client responds via `input.submit` RPC method)

---

### D7: Tool Registry Population via Directory Scan

**Context:** Phase 9 used `noopToolRegistry` (always returns `ErrToolNotFound`). Production needs tools loaded from `.tool.yaml` files on disk.

**Decision:** Add `internal/tool.ScanDir(dir string) ([]ToolDef, error)` that walks a directory for `*.tool.yaml` files, parses them, and returns `ToolDef` values. `BuildEngineConfig` calls `ScanDir` for each directory in `WireOptions.ToolDirs` and merges results into a `MapRegistry`.

**Alternatives considered:**
- Embedded tool definitions — too rigid, can't add project-level tools
- Runtime discovery (scan on first Lookup) — non-deterministic, harder to test

**Consequences:**
- Tools are discovered at startup, not at execution time
- Missing tool directory is a warning, not an error (empty registry is valid)
- Duplicate tool names across directories: first-seen-wins (match v1 behavior)
- `NewBuiltinRegistry()` stubs are still registered as fallbacks

---

### D8: No New External Dependencies

**Context:** Phase 10 adds significant functionality. Do we need any new libraries?

**Decision:** No new external dependencies. All implementations use the standard library + existing dependencies (`github.com/google/uuid`, `golang.org/x/sync`, `github.com/coder/websocket`).

**Alternatives considered:**
- `go.uber.org/zap` for structured logging — deferred to observability phase
- `github.com/fsnotify/fsnotify` for tool file watching — over-engineered for v2.0

**Consequences:**
- `go.mod` unchanged
- No supply chain risk
- Slightly more boilerplate for JSON encoding (acceptable)

---

## 13 Dependency Graph After Phase 10

```
cmd/serve          cmd/gert
    │                  │
    └──────┬───────────┘
           │
    internal/adapter         (wiring harness)
           │
    ┌──────┼──────────────────────────┐
    │      │                          │
internal/engine  internal/executor  internal/planner
    │      │          │               │
    │  internal/eventbus  internal/trace  internal/tool  internal/input
    │      │          │               │               │
    └──────┴──────────┴───────────────┴───────────────┘
                          │
                     pkg/* (interfaces only)
```

All arrows flow downward. No cycles. `pkg/*` remains the leaf layer.

---

## 14 Acceptance Criteria

Phase 10 is complete when:

1. `cmd/serve/main.go` contains zero placeholder implementations
2. `cmd/gert run testdata/basic.yaml` executes a runbook end-to-end with trace file output
3. `cmd/gert dry-run testdata/basic.yaml` completes without subprocess execution
4. All 24 tests pass with `-race -count=3`
5. `go vet ./...` clean
6. No new entries in `go.mod` (no external dependencies added)
7. Trace files are valid JSONL (each line parses as `trace.TraceEvent`)

---

*Ken, Software Architect*
