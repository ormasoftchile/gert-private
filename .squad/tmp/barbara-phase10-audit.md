# Phase 10 Spec Audit: Adapters & `gert run` CLI

**Auditor:** Barbara (Integrations Specialist)  
**Date:** 2026-04-20  
**Phase:** Phase 10 — Adapters  
**Phase 9 Sealed At:** Commit 545e4b6 (`gert serve`)  
**Spec Files Audited:**
- `/Volumes/Projects/gert/design/gert-v2/sections/02-architecture.tex` (Architecture)
- `/Volumes/Projects/gert/design/gert-v2/sections/13-adapter-contracts.tex` (Adapter Contracts)
- `/Volumes/Projects/gert/design/gert-v2/spec/02-architecture.md` (Spec markdown)
- `/Volumes/Projects/gert/design/gert-v2/spec/13-adapter-contracts.md` (Spec markdown)

---

## Executive Summary

Phase 10 implements **adapters** — the wiring layer between core runtime components and entry points. This includes:

1. **`gert run` CLI** — standalone runbook execution (no server mode)
2. **`gert serve` adapter improvements** — replacing stubs with real implementations
3. **EventDispatcher integration** — webhook/channel transport for `wait_for_event`
4. **TraceWriter integration** — JSONL trace output with HMAC signatures
5. **Real tool runtime wiring** — stdio/jsonrpc/mcp transports

### Critical Finding

⚠️ **The spec does NOT define a `gert run` CLI command with flags, arguments, or behavior.**

The spec mentions `gert run` only in **constraint contexts**:
- "gert run MUST reject `wait_for_event` steps" (§02-architecture.tex:524, 1079)
- "gert run mode" vs. "gert serve mode" distinction (§02-architecture.tex:553, 1840)
- Observability examples showing `gert run deploy.yaml --otel-endpoint=...` (§15)

**No formal CLI specification exists for:**
- Flags (e.g., `--var`, `--input`, `--trace`, `--dry-run`)
- Output format (stdout, stderr, exit codes)
- Error handling and reporting
- Progress indication
- Resume/checkpoint behavior in standalone mode

### Last r-number Used

**r20** — `r20-serve-rpc` (tests `gert serve` JSON-RPC execution contract)

Next available: **r21**

---

## Audit Question Answers

### 1. What does the spec say about the `gert run` CLI?

**Answer:** The spec does NOT define a `gert run` command specification.

**References Found:**

| Location | Context | Quote |
|----------|---------|-------|
| §02-architecture.tex:524-525 | wait_for_event constraint | "`gert serve` mode; `gert run` MUST reject the step with error: `'step type 'wait_for_event' requires gert serve mode'`" |
| §02-architecture.tex:553 | Runtime mode table | "`gert serve` mode — rejected at runtime in `gert run` (local CLI) mode." |
| §02-architecture.tex:1075-1080 | wait_for_event serve-only | "If a runbook containing a `wait_for_event` step is executed via `gert run` (local CLI, non-server mode), the Runtime Core MUST fail at dispatch time with: `step type 'wait_for_event' requires gert serve mode`" |
| §02-architecture.tex:1840 | EventDispatcher scope | "Lives exclusively inside `gert serve` process. It is not instantiated by `gert run`." |
| §02-architecture.tex:1904 | Dispatcher handle | "Runtime Core receives a `DispatcherHandle` at construction time (nil in `gert run` mode)" |
| §15-observability-diagnostics.tex | Examples only | `gert run deploy.yaml --otel-endpoint=http://jaeger:4317`<br>`gert run deploy.yaml --log-level=debug`<br>`gert run list [--limit=N] [--json]` |

**Inferred Behavior from Examples (§15):**

```bash
# Observability flags (inferred, not specified):
gert run deploy.yaml --otel-endpoint=http://jaeger:4317
gert run deploy.yaml --otel-console
gert run deploy.yaml --log-level=debug
gert run deploy.yaml --log-output=file:/var/log/gert/gert.log
gert run deploy.yaml --log-output=syslog://localhost:514
gert run deploy.yaml --output=json

# Run management (inferred):
gert run list [--limit=N] [--json]
```

**Gap:** No formal specification exists for:
- Positional arguments (runbook path)
- `--var KEY=VALUE` (variable overrides)
- `--input KEY=VALUE` (input bindings)
- `--trace PATH` (trace file output location)
- `--resume RUN_ID` (checkpoint restoration)
- `--dry-run` (see Q2)
- Exit codes (success = 0, failure = ?, validation error = ?)
- Stdout/stderr separation (step output vs. gert diagnostics)

**Recommendation:** Phase 10 should implement the **minimal viable CLI**:

```bash
gert run <runbook-path> [flags]

Flags:
  --var KEY=VALUE       Variable override (repeatable)
  --input KEY=VALUE     Input binding override (repeatable)
  --trace PATH          Trace file output (default: ./trace.jsonl)
  --output FORMAT       Output format: text (default), json, quiet
  --help                Show help
```

Advanced flags (resume, observability, logging) can be deferred to Phase 11+ as they depend on those subsystems being complete.

---

### 2. What does the spec say about `gert dry-run`?

**Answer:** The spec mentions `--dry-run` only in the context of **`gert migrate`**, NOT as a general execution mode.

**References Found:**

| Location | Context | Quote |
|----------|---------|-------|
| §03-schema-vnext.tex:163 | migrate example | `gert migrate --dry-run runbook.yaml` |
| §06-runtime-events.tex:725 | replay note | "To achieve full deterministic replay (for testing and debugging), use dry-run mode (§8.2) or snapshot the external environment." |
| §10-migration-compatibility.tex:179 | migrate command | `gert migrate --to-v2 --dry-run runbook.yaml` |
| §10-migration-compatibility.tex:201 | migrate flag table | `--dry-run` \| bool \| Preview changes without writing |
| §10-migration-compatibility.tex:461 | migration workflow | "Run `gert migrate --to-v2 --dry-run --check` to identify migration scope." |

**Gap Analysis:**

1. **No `gert run --dry-run` specification exists.**
2. Reference to "dry-run mode (§8.2)" in §06 appears to be a dangling reference — §8.2 does not define dry-run mode.
3. The concept of a "dry-run" execution (parse → plan → validate but do NOT execute) is useful but **unspecified**.

**Inferred Semantics:**

A `gert run --dry-run` flag would likely:
- Parse the runbook (validate YAML schema)
- Resolve imports and tools (full planning phase)
- Validate all step preconditions
- NOT execute any steps
- Report plan summary or validation errors

**Recommendation:**

Phase 10 should **NOT implement `--dry-run`** for `gert run`. This feature is mentioned only for `gert migrate` and lacks a specification. If Brian encounters a clear use case during implementation, escalate to Ken for spec clarification.

**Defer to:** Phase 13 (Testing/Acceptance) or Phase 14 (Polish) if dry-run execution becomes a validated requirement.

---

### 3. What does the spec say about EventDispatcher / wait_for_event webhook transport?

**Answer:** Fully specified in §02-architecture.tex (§"Wait-for-Event Executor Contract" and §"Event Dispatcher").

**Interface Specification (§02-architecture.tex:1874-1897):**

```go
type EventDispatcher interface {
    // Register creates a listener for an inbound event on a waiting run.
    // Returns a one-time HMAC token (non-empty for webhook source only).
    Register(ctx context.Context, reg ListenerRegistration) (token string, err error)

    // Send posts an event to a named channel (in-process transport only).
    // Returns ErrNotRegistered if no listener exists for the eventID.
    Send(ctx context.Context, eventID string, payload map[string]any) error

    // Unregister removes a listener (called on timeout or run cancellation).
    Unregister(ctx context.Context, runID, eventID string) error
}

type ListenerRegistration struct {
    RunID         string
    EventID       string
    Source        string              // "webhook" | "channel" | "broker"
    Filter        map[string]string   // key-value pairs that must match
    PayloadSchema *jsonschema.Schema  // validation schema for inbound payload
    Timeout       time.Duration
}
```

**Transport Implementations (§02-architecture.tex:1848-1870):**

1. **Webhook Transport** (HTTP POST endpoint)
   - Path: `POST /events/{run-id}/{event-id}`
   - Auth: `HMAC-SHA256` signature (one-time token from `Register()`)
   - Responses:
     - `202 Accepted` — event dispatched successfully
     - `404 Not Found` — no listener registered
     - `401 Unauthorized` — signature verification failed
     - `410 Gone` — run no longer in WAITING state
   - Body: JSON payload validated against `ListenerRegistration.PayloadSchema`

2. **Channel Transport** (in-process)
   - Maintained as `map[string]chan Payload` keyed by `event.id`
   - Any component in the same `gert serve` process can call `EventDispatcher.Send(eventID, payload)`
   - Used for test fixtures and internal coordination (e.g., extension → runtime signaling)

3. **Message Broker Transport** (stub in v2.0)
   - Pluggable `MessageBroker` interface (implementation-defined)
   - Ships as no-op stub by default
   - Deferred to v2.1+

**Executor Behavior (§02-architecture.tex:947-1082):**

When a `wait_for_event` step executes:

1. Runtime Core persists run state to run store (BoltDB/SQLite)
2. Calls `EventDispatcher.Register(runID, eventID, source, filter, payloadSchema, timeout)`
3. Sets `run.state = WAITING`
4. Writes `step/waiting` trace event
5. Suspends execution (does NOT block a thread)
6. Returns control to the serve loop

When an event arrives:

1. Dispatcher validates payload against schema
2. Checks filter conditions
3. If match succeeds:
   - Writes `event/received` trace event
   - Applies `capture` mappings from step spec
   - Sets `run.state = RUNNING`
   - Resumes execution at next step

**Timeout Handling:**

If timeout elapses before event arrival, executor applies `on_timeout` policy:
- `on_timeout: fail` → set `run.state = FAILED`, emit `step/failed` with reason `"wait_for_event timeout"`
- `on_timeout: continue` → set `run.state = RUNNING`, proceed to next step

**Serve-Only Constraint:**

`wait_for_event` is ONLY valid in `gert serve` mode. If executed in `gert run` mode:
- Runtime Core MUST fail at dispatch time
- Error message: `"step type 'wait_for_event' requires gert serve mode"`

**Implementation Status (Phase 9):**

Phase 9 (`gert serve`, commit 545e4b6) used a **fake EventDispatcher**:

```go
// cmd/serve/main.go:115
Dispatcher:  testutil.NewFakeEventDispatcher(),
```

**Phase 10 Requirements:**

1. Implement **real EventDispatcher** in `pkg/eventbus` (DONE — interface exists at `/Volumes/Projects/gert/v2/pkg/eventbus/dispatcher.go`)
2. Wire real dispatcher into `cmd/serve/main.go` (replaces fake)
3. Ensure `cmd/gert/main.go` passes `nil` dispatcher to runtime (enforces serve-only constraint)
4. Implement webhook HTTP handler in serve package
5. Add integration test with r21 fixture (see below)

**Spec Compliance Checklist:**

- [ ] EventDispatcher interface matches spec (Register, Send, Unregister)
- [ ] Webhook transport returns correct HTTP status codes
- [ ] HMAC-SHA256 one-time token generation and validation
- [ ] Filter matching (payload key-value pairs)
- [ ] JSON Schema validation of inbound payloads
- [ ] Timeout handling (fail vs. continue policies)
- [ ] `gert run` rejects `wait_for_event` with specified error message
- [ ] Trace events: `step/waiting`, `event/received`, `step/failed` (on timeout)

---

### 4. Are there any adapter contracts in the spec that Phase 9 partially or fully missed?

**Answer:** Phase 9 implemented the **JSON-RPC execution contract (exec/v2)** successfully. No critical gaps identified in adapter contracts themselves. However, **wiring gaps** exist in how `gert serve` integrates real components.

**Phase 9 Accomplishments:**

✅ JSON-RPC execution contract (§13-adapter-contracts.tex:53-100)
- Methods: `exec/start`, `exec/next`, `exec/cancel`, `exec/status`, `run/list`, `run/get`
- Transport: stdio JSON-RPC 2.0 and HTTP POST to `/rpc`
- Request/response envelopes match spec

✅ WebSocket event stream (§13-adapter-contracts.tex:102-150)
- Real-time event streaming via `/events` WebSocket endpoint
- Event envelope: `{event_id, run_id, kind, timestamp, sequence, payload}`

**Wiring Gaps in Phase 9 (Stubs Used):**

These are **not spec violations** — Phase 9 correctly used stubs to isolate the serve infrastructure. Phase 10 must replace them with real implementations:

| Stub | Location | Real Implementation | Status |
|------|----------|---------------------|--------|
| `testutil.NewFakeEventDispatcher()` | `cmd/serve/main.go:115` | `eventbus.NewDispatcher()` | Package exists at `pkg/eventbus/` |
| `discardTraceWriter{}` | `cmd/serve/main.go:116` | `trace.NewJSONLWriter()` | Package exists at `pkg/trace/` |
| `noopToolRegistry{}` | `cmd/serve/main.go:41` | `tool.NewBuiltinRegistry()` | Exists at `internal/tool/registry.go` |

**Additional Adapter Contracts (Not Yet Implemented):**

These are **Phase 11+ deliverables**, NOT Phase 10:

- **Tool Invocation Contract** (§13-adapter-contracts.tex:152-200)
  - stdio, jsonrpc, mcp transports
  - Phase 6 delivered internal implementation
  - Phase 10 wires it into serve
  
- **Extension Handshake Contract** (§13-adapter-contracts.tex:202-250)
  - Ed25519 trust verification
  - Manifest lifecycle
  - Phase 7 delivered, Phase 10 wires it

**Phase 10 Scope:**

Replace stubs in `cmd/serve/main.go` and `cmd/gert/main.go` with real components. No new adapter contracts need implementing.

---

### 5. What is the last r-number used?

**Answer:** `r20` — `r20-serve-rpc`

**All Existing Fixtures:**

```
r01-k8s-incident            (Phase 1-2 acceptance)
r02-canary-deploy           (Phase 1-2 acceptance)
r03-employee-onboarding     (Phase 1-2 acceptance)
r04-soc2-evidence           (Phase 1-2 acceptance)
r05-security-breach         (Phase 1-2 acceptance)
r06-db-migration            (Phase 1-2 acceptance)
r07-financial-approval      (Phase 1-2 acceptance)
r08-fda-release             (Phase 1-2 acceptance)
r09-oncall-escalation       (Phase 1-2 acceptance)
r10-gdpr-deletion           (Phase 1-2 acceptance)
r11-iterate-loop            (Phase 3 runtime)
r12-approval-quorum         (Phase 4 governance)
r13-decision-routing        (Phase 5 step types)
r14-assert-compensate       (Phase 5 step types)
r15-branch-collector        (Phase 5 step types)
r16-end-step                (Phase 5 step types)
r17-tool-transport          (Phase 6 tool runtime)
r18-extension-host          (Phase 7 extensions)
r19-input-provider          (Phase 8 input providers)
r20-serve-rpc               (Phase 9 gert serve)
```

**Next Available:** `r21`

---

### 6. What engine/tool/trace packages currently exist that Phase 10 will wire together?

**Current Package Inventory:**

#### Engine & Runtime

| Package | Path | Status | Purpose |
|---------|------|--------|---------|
| Engine | `v2/internal/engine/` | ✅ Complete | Runtime Core execution loop |
| Executor Registry | `v2/internal/executor/` | ✅ Complete | Step type executor registry |
| Parser | `v2/internal/parser/` | ✅ Complete | YAML → ParsedRunbook |
| Planner | `v2/internal/planner/` | ✅ Complete | ParsedRunbook → ExecutionPlan |
| Expression Evaluator | `v2/internal/expr/` | ✅ Complete | Go template + infix conditions |

#### Subsystems (Real Implementations Exist)

| Package | Path | Key Types | Status |
|---------|------|-----------|--------|
| Trace | `pkg/trace/` | `TraceWriter`, `TraceEvent` | ✅ Interface + types defined |
| Trace Writer | `pkg/trace/writer.go` | JSONL append-only writer | ⚠️ Needs implementation |
| EventBus | `pkg/eventbus/` | `EventDispatcher`, `InboundEvent`, `EventFilter` | ✅ Interface defined, needs implementation |
| Tool Runtime | `internal/tool/` | `ToolRegistry`, `ToolRuntime`, stdio/jsonrpc/mcp transports | ✅ Complete (Phase 6) |
| Input Provider | `internal/input/` | `ChainProvider`, `EnvProvider`, `VaultProvider`, `PromptProvider` | ✅ Complete (Phase 8) |
| Governance | `internal/governance/` | `ApprovalGate` | ✅ Complete (Phase 4) |

#### Stubs in cmd/serve/main.go (Lines to Replace)

**Line 41-42: noopToolRegistry**

```go
// Current (Phase 9 stub):
type noopToolRegistry struct{}
func (n *noopToolRegistry) Lookup(_ context.Context, _ string, _ string) (*schema.ToolDef, error) {
    return nil, planner.ErrToolNotFound
}

// Phase 10 replacement:
toolRegistry := internaltool.NewBuiltinRegistry()
```

**Line 95: ToolRuntime**

```go
// Current (Phase 9):
toolRegistry := internaltool.NewBuiltinRegistry()
toolRuntime := internaltool.NewDefaultToolRuntime(toolRegistry)

// Phase 10: Already wired correctly! ✅
```

**Line 115: EventDispatcher**

```go
// Current (Phase 9 stub):
Dispatcher:  testutil.NewFakeEventDispatcher(),

// Phase 10 replacement:
realDispatcher := eventbus.NewDispatcher()
// ...
Dispatcher: realDispatcher,
```

**Line 116: TraceWriter**

```go
// Current (Phase 9 stub):
type discardTraceWriter struct{}
func (discardTraceWriter) Append(_ trace.TraceEvent) error { return nil }
func (discardTraceWriter) Close() error                    { return nil }

// Phase 10 replacement:
traceWriter, err := trace.NewJSONLWriter("trace.jsonl")
if err != nil {
    log.Fatalf("gert serve: %v", err)
}
defer traceWriter.Close()
// ...
TraceWriter: traceWriter,
```

---

## Gap Analysis: What's Missing for Phase 10

### Critical Gaps (Blockers)

1. **`gert run` CLI Specification**
   - **Impact:** HIGH — Phase 10 cannot implement the primary deliverable without a spec
   - **Blocker:** No flags, arguments, output format, or error handling defined
   - **Resolution Path:** Ken must approve minimal CLI spec (see Recommendation below)

2. **TraceWriter Implementation**
   - **Impact:** MEDIUM — trace.jsonl output is foundational
   - **Status:** Interface exists (`pkg/trace/writer.go`), implementation missing
   - **Resolution Path:** Brian implements JSONL writer with HMAC chaining (Phase 11 feature, but basic writer needed now)

3. **EventDispatcher Implementation**
   - **Impact:** MEDIUM — `wait_for_event` support is a Phase 10 deliverable
   - **Status:** Interface exists (`pkg/eventbus/dispatcher.go`), implementation missing
   - **Resolution Path:** Brian implements in-memory dispatcher with webhook HTTP handler

### Non-Critical Gaps (Can Work Around)

4. **Dry-Run Mode**
   - **Impact:** LOW — mentioned in spec but not specified for `gert run`
   - **Resolution Path:** Defer to Phase 13+ or clarify with Ken if needed

5. **Resume/Checkpoint in Standalone Mode**
   - **Impact:** LOW — checkpointing is Phase 11 feature
   - **Resolution Path:** `gert run` does NOT support `--resume` in Phase 10

6. **Observability Flags**
   - **Impact:** LOW — OTel integration is Phase 12 feature
   - **Resolution Path:** Examples in spec suggest `--otel-endpoint`, but defer to Phase 12

---

## r21 Fixture: `gert run` Integration Test

**Created At:** `/Volumes/Projects/gert/design/gert-v2/testdata/runbooks/r21-gert-run/schema.yaml`

**Purpose:**

Exercise the standalone `gert run` CLI adapter with:
- Variable overrides (`--var`)
- Infix condition evaluation
- Tool invocation (echo)
- Environment capture
- Scenario validation (non-serve mode constraints)

**Key Features:**

1. ✅ Uses real tools (echo, shell commands) — no wait_for_event (serve-only)
2. ✅ Tests infix condition syntax (`environment == "prod"`)
3. ✅ Variable capture and flow through steps
4. ✅ Suitable for standalone execution (no EventDispatcher required)
5. ✅ Clear expected trace event sequence documented in comments
6. ✅ Demonstrates reject behavior (commented-out wait_for_event scenario)

**Test Scenarios:**

| Scenario | CLI Invocation | Expected Outcome |
|----------|----------------|------------------|
| Default (dev) | `gert run r21-gert-run/schema.yaml` | Deploys to "development", exit 0 |
| Production | `gert run r21-gert-run/schema.yaml --var environment=prod` | Deploys to "prod", exit 0 |
| Staging | `gert run r21-gert-run/schema.yaml --var environment=staging` | Deploys to "staging", exit 0 |

**Not Tested (Serve-Only):**

The fixture includes a **commented-out scenario** showing `wait_for_event` rejection:

```yaml
# SCENARIO 2 (serve-only): wait_for_event rejection
# Uncomment to test serve-only constraint validation.
# Expected: gert run MUST reject with error:
#   "step type 'wait_for_event' requires gert serve mode"
```

This can be tested manually or in an integration test that expects failure.

---

## Recommendations for Brian's Implementation Order

### Phase 10 Implementation Sequence

**Pre-Implementation (Ken Approval Required):**

1. ✅ **Escalate `gert run` CLI spec gap to Ken**
   - Propose minimal viable CLI (see below)
   - Get approval before writing cmd/gert/run.go

**Recommended Minimal CLI Spec (for Ken's review):**

```bash
gert run <runbook-path> [flags]

Arguments:
  runbook-path         Path to runbook YAML file (required)

Flags:
  --var KEY=VALUE      Variable override (repeatable)
  --input KEY=VALUE    Input binding override (repeatable)
  --trace PATH         Trace file output (default: trace.jsonl)
  --output FORMAT      Output format: text, json, quiet (default: text)
  --help               Show this help

Exit Codes:
  0  Success (run completed)
  1  Execution failure (step failed)
  2  Validation error (parse/plan failed)
  3  Runtime error (crash, panic)

Output:
  - Stdout: Step output (when --output=text)
  - Stderr: gert diagnostics, progress, errors
  - Trace: trace.jsonl (append-only JSONL event log)
```

**Week 1: Core Wiring (No New Features)**

1. **Implement TraceWriter** (`pkg/trace/writer.go`)
   - JSONL append-only file writer
   - Event serialization (JSON marshaling of `TraceEvent`)
   - Signature placeholder (HMAC chain deferred to Phase 11)
   - Tests: write events, verify JSONL format, close behavior

2. **Implement EventDispatcher** (`pkg/eventbus/dispatcher.go`)
   - In-memory registration map (`map[string]*ListenerRegistration`)
   - `Register()`, `Send()`, `Unregister()` methods
   - Timeout handling (goroutine per listener with timer)
   - Filter matching (payload key-value equality)
   - Tests: register, send, timeout, unregister

3. **Wire Real Components into `cmd/serve/main.go`**
   - Replace `testutil.NewFakeEventDispatcher()` → `eventbus.NewDispatcher()`
   - Replace `discardTraceWriter{}` → `trace.NewJSONLWriter("trace.jsonl")`
   - Replace `noopToolRegistry{}` → wiring already correct (Phase 9 left it in place)
   - Verify: `gert serve` starts without crashes

**Week 2: `gert run` CLI**

4. **Implement `cmd/gert/run.go`** (after Ken approves spec)
   - Parse CLI flags (`--var`, `--input`, `--trace`, `--output`)
   - Wire Parser → Planner → Runtime Core pipeline
   - Pass `nil` EventDispatcher (enforces serve-only constraint for wait_for_event)
   - Output formatting (text, json, quiet)
   - Exit code handling (0/1/2/3)

5. **Integration Test with r21 Fixture**
   - Run: `gert run testdata/runbooks/r21-gert-run/schema.yaml`
   - Run: `gert run testdata/runbooks/r21-gert-run/schema.yaml --var environment=prod`
   - Verify: trace.jsonl written with correct event sequence
   - Verify: exit code 0 on success

**Week 3: Webhook Transport + Testing**

6. **Webhook HTTP Handler** (`internal/serve/events.go`)
   - Route: `POST /events/{run-id}/{event-id}`
   - HMAC-SHA256 signature verification
   - Payload JSON Schema validation
   - Dispatcher integration (call `dispatcher.Send()`)
   - HTTP status codes: 202, 401, 404, 410

7. **Integration Test: wait_for_event with Webhook**
   - Start `gert serve` with r21 fixture (modified to include wait_for_event)
   - POST event to webhook endpoint
   - Verify: step resumes, trace event `event/received` written

8. **Test: `gert run` Rejects wait_for_event**
   - Run: `gert run testdata/runbooks/r21-gert-run/scenario-wait-for-event.yaml`
   - Verify: exits with code 2 (validation error)
   - Verify: error message matches spec: `"step type 'wait_for_event' requires gert serve mode"`

---

## Integration Concerns & Risks

### Risk 1: TraceWriter HMAC Signature Complexity

**Issue:** Spec requires HMAC-SHA256 signature over canonical JSON of trace events. Phase 11 feature, but TraceWriter interface expects `Signature` field.

**Mitigation:**
- Phase 10: Leave `Signature` field empty (`""`)
- Phase 11: Implement HMAC chaining with secret key from `GERT_TRACE_KEY` env var
- Tests: Verify JSONL output is valid even with empty signature

**Status:** LOW RISK — can be stubbed safely

---

### Risk 2: EventDispatcher Concurrency

**Issue:** Dispatcher manages goroutines-per-listener with timeouts. Incorrect synchronization could cause deadlocks or race conditions.

**Mitigation:**
- Use `sync.Mutex` to guard registration map
- Each listener gets its own goroutine with `time.AfterFunc` for timeout
- Unregister cancels the timeout timer (`timer.Stop()`)
- Tests: Run under `go test -race`

**Status:** MEDIUM RISK — requires careful concurrency review by Ken

---

### Risk 3: `gert run` CLI Spec Divergence

**Issue:** If Brian implements `gert run` without Ken's approval, behavior may not match team expectations or future spec iterations.

**Mitigation:**
- ✅ **BLOCKER: Do NOT implement `cmd/gert/run.go` until Ken approves CLI spec**
- Propose minimal spec (see above)
- Get written approval in `.squad/decisions/`

**Status:** HIGH RISK — mitigated by requiring Ken approval

---

### Risk 4: Webhook HMAC Token Reuse

**Issue:** Spec requires "one-time" HMAC tokens for webhook authentication. If tokens are reusable, replay attacks are possible.

**Mitigation:**
- `EventDispatcher.Register()` generates fresh HMAC token (UUID + secret)
- Token is invalidated after first successful use OR timeout
- `Unregister()` deletes token from token map
- Tests: Verify second POST with same token returns 401

**Status:** MEDIUM RISK — requires security review by Ken

---

## Deliverables Checklist

### Code Deliverables

- [ ] `pkg/trace/writer.go` — JSONL trace writer implementation
- [ ] `pkg/trace/writer_test.go` — TraceWriter unit tests
- [ ] `pkg/eventbus/dispatcher.go` — EventDispatcher implementation
- [ ] `pkg/eventbus/dispatcher_test.go` — EventDispatcher unit tests
- [ ] `cmd/gert/run.go` — `gert run` CLI command (after Ken approval)
- [ ] `cmd/serve/main.go` — Replace stubs with real TraceWriter, EventDispatcher
- [ ] `internal/serve/events.go` — Webhook HTTP handler for `/events/{run-id}/{event-id}`

### Test Deliverables

- [ ] `r21-gert-run/schema.yaml` — Integration test fixture (DONE — see below)
- [ ] `r21-gert-run/README.md` — Fixture documentation
- [ ] Integration test: `gert run r21-gert-run/schema.yaml` succeeds
- [ ] Integration test: `gert run` with `--var` overrides works
- [ ] Integration test: `gert run` rejects `wait_for_event` with correct error
- [ ] Integration test: `gert serve` with webhook POST triggers event dispatch

### Documentation Deliverables

- [ ] `.squad/decisions/inbox/barbara-phase10-audit.md` — Architectural decisions (see below)
- [ ] `.squad/tmp/barbara-phase10-audit.md` — This audit document (DONE)
- [ ] Update `PLAN.md` if Phase 10 scope changes based on Ken's CLI spec approval

---

## Next Steps

1. **Barbara → Ken:** Escalate `gert run` CLI spec gap with proposed minimal spec
2. **Ken → Brian:** Approve or revise CLI spec, document in `.squad/decisions/`
3. **Brian:** Implement Phase 10 in recommended sequence (Week 1 → 2 → 3)
4. **Brian → Ken:** Submit Phase 10 for review with integration tests passing
5. **Ken → Team:** Approve Phase 10, proceed to Phase 11 (Evidence & Replay)

---

## Appendix: Existing Package Interfaces

### pkg/trace/writer.go

```go
type TraceWriter interface {
    Append(event TraceEvent) error
    Close() error
}
```

### pkg/eventbus/dispatcher.go

```go
type EventDispatcher interface {
    Dispatch(ev InboundEvent) error
    Wait(ctx context.Context, stepID string, filter EventFilter, timeout time.Duration) (*InboundEvent, error)
    Cancel(stepID string, reason string)
}
```

**Note:** This interface differs slightly from the spec's `Register/Send/Unregister` pattern. The implementation may need adjustment to match spec exactly. Flagging for Ken's review.

---

**End of Audit**
