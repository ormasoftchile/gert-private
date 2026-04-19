# Brian — Go Implementability Notes for gert v2
**Date:** 2026-04-18
**Author:** Brian (Go Programmer)
**Audience:** Full design team — especially Ken (architecture), Leslie (spec), Dennis (research)

---

## Top 5 Go Implementation Concerns

### 1. Event Bus Blocking the Execution Loop

The v2 design says adapters (TUI, Web, VS Code) subscribe to events from the runtime. The key danger in Go: if the event delivery channel blocks (slow consumer), the engine's execution loop stalls. v1 sidesteps this by writing events only to the trace file and having the serve layer tail it — clean, but adds polling latency.

**Recommendation:** The event bus must deliver to each subscriber on a bounded buffered channel (capacity 64–256 events). If a subscriber's channel is full, the event is dropped or an overflow policy is applied. The engine MUST NOT block on event delivery. A `NonBlockingPublish` pattern:

```go
func (b *EventBus) Publish(ev DurableEvent) {
    for _, sub := range b.subscribers {
        select {
        case sub.ch <- ev:
        default:
            sub.dropped.Add(1) // metrics; never block engine
        }
    }
}
```

This is the critical safety valve for the concurrency model.

### 2. Context Propagation Across Nested Invokes

When a runbook invokes a child runbook (`invoke` step), the child engine shares the parent's `context.Context` for cancellation but must have its own `RunStore`, `TraceWriter`, and `RunState`. In v1, `ResumeForServe` shows this pattern, but the boundary between parent and child context is implicit.

**Recommendation:** Define an `InvokeContext` struct that carries the parent run ID, the invoke frame for the stack, and a child-scoped context with its own cancellation:

```go
type InvokeContext struct {
    ParentRunID string
    ChildRunID  string
    Frame       InvokeFrame
    Ctx         context.Context    // child-scoped; cancels with parent
    Cancel      context.CancelFunc // child-only cancellation
}
```

The child engine's `context.Context` must be a child of the parent's context so that `gert exec --cancel <run-id>` propagates into all active children.

### 3. Saga/Compensation Without a Framework

Dennis's research recommends the saga pattern. Go has no native saga runtime (Temporal provides one, but that's a heavy dependency). Implementing saga compensation in idiomatic Go requires:

- Each step optionally declares a `compensate:` block (a cli/tool step to reverse it).
- The engine maintains a compensation stack (`[]CompensationEntry`) in `RunState`.
- On failure, the engine walks the stack in reverse, executing each compensation step.
- Compensation steps must be exempt from governance blocking (they are already approved by the parent step).

**Concern:** If a compensation step itself fails, what happens? The design must decide: halt, skip, or continue. This decision needs to be in the spec before implementation, or Brian will invent it inconsistently.

### 4. Parallel Iterate Goroutine Safety

v1 already implements parallel iterate via goroutines (`concurrency` field on `IterateBlock`). The v1 pattern uses a scoped variable copy per slot. The concern for v2 is that the event bus, trace writer, and snapshot writer must be safe for concurrent use when multiple goroutines are executing iterate slots.

**Recommendation:**
- `TraceWriter.Write` must serialize writes with a mutex or channel (v1 does this correctly via single-goroutine ownership — the engine loop calls it, not goroutines directly).
- For parallel iterate, each goroutine should collect its results into a local buffer, then the coordinator goroutine merges and writes to the trace after each slot completes.
- Do NOT have goroutines call `TraceWriter.Write` directly — route through a results channel.

```go
type slotResult struct {
    slotIndex int
    events    []DurableEvent
    state     SlotState
}
resultsCh := make(chan slotResult, concurrency)
// goroutines send to resultsCh; engine loop drains and writes
```

### 5. The `gert serve` JSON-RPC Contract Must Not Break

v1's VS Code extension communicates over a stable JSON-RPC stdio contract. Ken notes that the v2 design doesn't address this at all. If v2 changes the event schema (which it will — the new `DurableEvent` format differs from v1's `TraceEvent`), the VS Code extension will break silently.

**Recommendation:** The serve layer must version its events. Minimally, emit a `{"version":"2"}` header on connection so the client can negotiate. More robustly, define a compatibility shim in the serve layer that translates v2 `DurableEvent` types to the v1 JSON-RPC event names the extension currently expects. This shim lives in `ext/serve`, not in `pkg/engine`.

---

## Recommended Go Package Structure

Based on the five components Ken named in §02 and the v1 package structure:

```
gert/
├── cmd/
│   └── gert/           # CLI entrypoint; thin wiring only
│
├── pkg/
│   ├── schema/         # Component 1: Schema & Validation
│   │   ├── parse.go        # YAML → Runbook struct
│   │   ├── validate.go     # structural validation
│   │   ├── semantic.go     # semantic validation (capture refs, etc.)
│   │   └── migrate.go      # v1 → v2 migration
│   │
│   ├── planner/        # Component 2: Planner
│   │   ├── planner.go      # Runbook → ExecutionPlan
│   │   ├── plan.go         # ExecutionPlan, StepNode types
│   │   └── resolver.go     # input binding resolution
│   │
│   ├── engine/         # Component 3: Core Runtime
│   │   ├── engine.go       # execution loop
│   │   ├── types.go        # RunState, DurableEvent, etc.
│   │   ├── runstore.go     # RunStore interface + DirRunStore
│   │   ├── resume.go       # ResumeEngine, ResumeForServe
│   │   ├── snapshot.go     # SaveCheckpoint, LoadLatestCheckpoint
│   │   ├── trace.go        # TraceWriter
│   │   └── parallel.go     # parallel iterate coordination
│   │
│   ├── governance/     # Component 4: Governance Engine (part of runtime)
│   │   ├── engine.go       # GovernanceEngine
│   │   ├── allowlist.go
│   │   ├── denylist.go
│   │   ├── redact.go
│   │   └── approval.go
│   │
│   ├── events/         # Event Bus
│   │   ├── bus.go          # EventBus interface + ChannelBus impl
│   │   └── types.go        # event type constants
│   │
│   ├── tools/          # Tool Runtime
│   │   ├── manager.go
│   │   ├── stdio.go
│   │   ├── jsonrpc.go
│   │   └── mcp.go
│   │
│   ├── evidence/       # Evidence capture
│   │   ├── evidence.go     # EvidenceValue, HashFile
│   │   └── collector.go    # EvidenceCollector interface
│   │
│   ├── replay/         # Replay/scenario executor
│   │   ├── scenario.go
│   │   └── executor.go
│   │
│   ├── testing/        # gert test runner
│   │   ├── runner.go
│   │   └── assertions.go
│   │
│   └── testutil/       # Shared test helpers (not test binary)
│       ├── trace.go        # ReadTrace, AssertEvent helpers
│       ├── harness.go      # Integration test harness
│       └── exttest.go      # Extension test harness
│
└── ext/                # Adapter layer (outside pkg/ boundary)
    ├── serve/          # JSON-RPC serve layer
    ├── tui/            # TUI adapter
    ├── mcp/            # MCP server
    └── debug/          # Debugger
```

**Key boundary rule (mirrors v1):** `pkg/engine` must not import anything from `ext/`. The `ext/` packages are allowed to import `pkg/engine` but not vice versa.

---

## Key Go Interfaces

```go
// Runtime executes a planned runbook and publishes events to the bus.
// Implementations: Engine (production), DryRunEngine, ReplayEngine.
type Runtime interface {
    Execute(ctx context.Context, plan *planner.ExecutionPlan,
        bus events.EventBus) error
}

// Planner transforms a parsed runbook into an execution plan.
// Pure function; no I/O.
type Planner interface {
    Plan(ctx context.Context, rb *schema.Runbook,
        inputs map[string]string) (*ExecutionPlan, error)
}

// EventBus distributes runtime events to registered subscribers.
// Must not block the caller on slow consumers.
type EventBus interface {
    Publish(ev engine.DurableEvent)
    Subscribe(id string) (<-chan engine.DurableEvent, func())
    Close()
}

// TraceWriter durably persists trace events for a single run.
type TraceWriter interface {
    Write(ev engine.DurableEvent) error
    Close() error
}

// RunStore manages the filesystem artefacts of a single run.
type RunStore interface {
    RunID() string
    WriteTrace(ev engine.DurableEvent) error
    ReadTrace() ([]engine.DurableEvent, error)
    ReadTraceSince(afterSeq int64) ([]engine.DurableEvent, error)
    SaveCheckpoint(state *engine.RunState) (string, error)
    LoadLatestCheckpoint() (*engine.RunState, int64, error)
    AcquireLock() error
    ReleaseLock() error
    WriteManifest(m *engine.RunManifest) error
}

// CommandExecutor runs a shell command and returns its result.
// Implementations: RealExecutor, ReplayExecutor, DryRunExecutor.
type CommandExecutor interface {
    Execute(ctx context.Context, argv []string,
        env map[string]string) (*CommandResult, error)
}

// EvidenceCollector gathers evidence values from an operator.
// Implementations: InteractiveCollector, ReplayCollector.
type EvidenceCollector interface {
    Collect(ctx context.Context, stepID string,
        prompts []Prompt) (map[string]*EvidenceValue, error)
}

// GovernanceEngine evaluates policy rules against proposed actions.
type GovernanceEngine interface {
    CheckCommand(cmd []string) error         // allowlist / denylist
    CheckEnv(vars map[string]string) error   // env block
    RequiresApproval(stepID string) bool     // approval gates
    Approve(ctx context.Context, stepID string,
        approver string) error
}
```

---

## Concurrency Patterns

### Event Bus: Channels over Mutexes

Use buffered channels for subscriber delivery, not a mutex-protected slice. Rationale:

- **Fan-out without blocking**: `select { case ch <- ev: default: }` is safe, composable, and testable.
- **Backpressure visibility**: dropped event counters expose slow consumers without stalling the engine.
- **Context cancellation**: subscriber goroutines can `select` on both the event channel and `ctx.Done()`.

A mutex-protected notification pattern (`sync.Mutex` + condition variable) is harder to test and harder to reason about under timeout scenarios (approval gates, SLA enforcement).

### Context Propagation

Every function that may block on I/O, a human, or a tool call takes `context.Context` as its first argument. The engine passes a single root context into `Execute`; `time.AfterFunc` or `context.WithTimeout` is used for per-step timeouts. The serve layer can cancel a run by cancelling the root context.

```go
// Step-level timeout example:
stepCtx, cancel := context.WithTimeout(ctx, step.Timeout)
defer cancel()
result, err := executor.Execute(stepCtx, step.Argv, step.Env)
```

**No goroutine-per-step by default.** The engine is a sequential loop with one goroutine. Parallel iterate uses a worker pool (bounded by `concurrency`), not an unbounded goroutine per item. This is critical for governance and tracing correctness.

### Snapshot Writes

Snapshots are written synchronously after each step. They are the bottleneck (fsync on every checkpoint). If this becomes a performance issue, snapshots can be written to a goroutine-owned write queue with a max-depth of 1 (drop older if newer comes in). However, this optimisation should only be applied after profiling shows it is needed.

---

## v1 Patterns Worth Preserving

| Pattern | Location | Why keep it |
|---|---|---|
| `DirRunStore` interface separation | `pkg/engine/runstore.go` | Clean separation enables `MemRunStore` for tests |
| Atomic snapshot write (tmp + rename) | `pkg/engine/runstore.go:SaveCheckpoint` | Prevents corrupt snapshots on crash |
| PID lock file for concurrent run detection | `pkg/engine/runstore.go:AcquireLock` | Simple, works on all POSIX systems |
| `AddSecrets` / `redactString` pipeline | `pkg/engine/trace.go` | Prevents secrets leaking to trace; must be preserved |
| `context.Context` through all executor calls | `pkg/engine/engine.go` | Cancellation propagation; do not add `error`-only APIs |
| `RunStore` interface (not a concrete type) | `pkg/engine/runstore.go` | Enables `MemRunStore` for fast unit tests |

## v1 Anti-Patterns to Eliminate

| Anti-pattern | Location | Problem | v2 Fix |
|---|---|---|---|
| `TraceEvent` wraps `StepResult` (only one event type) | `pkg/engine/trace.go` | Cannot represent partial steps, governance events, or tool events | Replace with typed `DurableEvent` catalogue (§12) |
| Engine struct carries both state and I/O | `pkg/engine/engine.go` | Hard to unit-test without real files | Inject `RunStore`, `EventBus`, and `CommandExecutor` as interfaces |
| `serve` layer directly accesses engine fields | `ext/serve/pkg/serve/session.go` | Leaks internals; breaks encapsulation | Engine exposes events only; serve subscribes to bus |
| Snapshot format is not versioned | `pkg/engine/snapshot.go` | Cannot evolve `RunState` schema without breaking existing runs | Add `"version": 2` field to snapshot JSON |
| `History []*StepResult` grows unbounded | `pkg/engine/types.go` | Memory pressure on long-running runbooks | Cap in-memory history; load from trace for resume |
| No context on evidence collector | `pkg/providers/manual.go` | Cannot respect timeouts or cancellation for manual prompts | Add `ctx context.Context` parameter |
