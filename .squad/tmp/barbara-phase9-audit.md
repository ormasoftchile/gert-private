# Phase 9 Spec Audit — gert serve

**Auditor:** Barbara (Integrations Specialist)  
**Date:** 2026-04-18  
**Requested by:** Cristian

---

## 1. Spec Summary — What Phase 9 Delivers

Phase 9 implements `gert serve`, the HTTP/WebSocket/JSON-RPC server that powers:

1. **VS Code extension** (stdio JSON-RPC 2.0)
2. **Web clients** (HTTP + WebSocket)
3. **External integrations** (CI/CD, observability, approval bots)

### Architectural Position

`gert serve` is an **adapter** in the API/Adapter Layer. It does NOT own execution logic; it:

- Accepts JSON-RPC 2.0 requests over stdio or HTTP POST `/rpc`
- Calls `Runtime.Start()` → receives `RunHandle`
- Stores handles in a session map keyed by `run_id`
- Drains `handle.Events()` channel and forwards events to clients
- Dispatches control calls (`exec/next`, `exec/cancel`, `exec/approve`) to the handle

### Contracts Exposed

Per Section 13 (Adapter Contracts):

**exec/v2 (JSON-RPC Execution Contract):**
- Transport: stdio JSON-RPC 2.0 OR HTTP POST `/rpc`
- Methods:
  - `exec/start` — start new run
  - `exec/next` — advance by one step
  - `exec/cancel` — cancel run
  - `exec/status` — get current status
  - `run/list` — list all runs
  - `run/get` — get run details

**events/v2 (WebSocket Event Stream Contract):**
- Transport: WebSocket at `/ws`
- Client sends `subscribe` message with `runId`
- Server streams events in order
- Supports `sinceSequence` for reconnection

**Webhook transport for wait_for_event:**
- Endpoint: `POST /events/{run-id}/{event-id}`
- Auth: HMAC-SHA256
- Used by `wait_for_event` step type (gert serve mode only)

**Health endpoint:**
- `GET /health` (Section 15)

---

## 2. Gap Analysis

### 2.1 Missing Packages

**Status:** None exist yet.

1. **`v2/cmd/serve/`** — MISSING
   - Entry point: `main.go` with CLI flag parsing
   - Flags: `--stdio`, `--http`, `--port`, `--bind`, `--trace-dir`

2. **`v2/internal/serve/`** — MISSING
   - `server.go` — Main server struct
   - `jsonrpc.go` — JSON-RPC 2.0 handler (stdio or HTTP POST)
   - `websocket.go` — WebSocket upgrade and event streaming
   - `webhook.go` — POST /events handler for wait_for_event
   - `session.go` — Session map: `run_id -> RunHandle`

3. **`v2/pkg/serve/`** — DECISION NEEDED
   - Should serve expose a public API, or is it purely internal to cmd/serve?
   - Recommendation: Keep serve as `internal/serve` only. External consumers use the JSON-RPC contract, not Go APIs.

### 2.2 Engine Surface

**What exists:**

```go
// v2/pkg/engine/engine.go
type Engine interface {
    Start(ctx context.Context, plan *ExecutionPlan, opts RunOptions) (RunHandle, error)
    Resume(ctx context.Context, runID string, opts RunOptions) (RunHandle, error)
}

type RunHandle interface {
    Next(ctx context.Context) (*StepResult, error)
    Approve(ctx context.Context, decision ApprovalDecision) error
    SubmitEvidence(ctx context.Context, stepID string, ev map[string]*EvidenceValue) error
    Cancel(ctx context.Context, reason string) error
    State() RunState
    Events() <-chan Event
}
```

**Conclusion:** ✅ Engine surface is complete for Phase 9.

The serve layer only needs:
- `Engine.Start()` → `RunHandle`
- `RunHandle.Next()` → step-by-step execution
- `RunHandle.Events()` → event stream for forwarding
- `RunHandle.Cancel()` → cancellation

All methods exist.

### 2.3 EventBus Implementation

**Status:** ✅ Exists at `v2/pkg/eventbus/`

- `bus.go` — EventBus interface + impl
- `dispatcher.go` — EventDispatcher interface (for wait_for_event)

**Note:** Section 02 specifies EventDispatcher lives in `gert serve` only (not in `gert run`).

### 2.4 Runbook Fixture Status

**Last r-number used:** r19 (r19-input-provider)

**Created:** r20-serve-rpc at:
`/Volumes/Projects/gert/design/gert-v2/testdata/runbooks/r20-serve-rpc/schema.yaml`

---

## 3. Event Types Defined in Spec

Per Section 06 (Runtime Events), the complete event catalog:

### 3.1 Mandatory Events (MUST emit)

**Run lifecycle:**
- `run/started` — First event in every run
- `run/completed` — Terminal state (success/failure/cancelled)
- `run/cancelled` — When cancelled before natural completion

**Step lifecycle:**
- `step/started` — Before step execution
- `step/completed` — After successful step (outcome: success | skipped)
- `step/failed` — On step error
- `step/skipped` — Not executed due to control flow

### 3.2 Conditional Events (MAY emit)

**Governance:**
- `governance/command_checked` — Allowlist/denylist verdict
- `governance/approval_requested` — Approval gate entered
- `governance/approval_received` — Each approval decision
- `governance/redaction_applied` — Redaction occurred

**Step retry:**
- `step/retrying` — Before retry attempt (v2.1 feature)

**Extensions:**
- `extension/loaded` — Extension handshake complete
- `tool/invoked` — Tool call started
- `tool/completed` — Tool call finished

### 3.3 Event Envelope

All events carry:
```json
{
  "event_id": "<UUID v4>",
  "run_id": "<UUID>",
  "runbook_id": "<string>",
  "timestamp": "<RFC3339 with μs precision>",
  "kind": "<event type>",
  "sequence": "<int64, monotonic per run>",
  "payload": { ... }
}
```

**Defined in:** `v2/pkg/engine/run.go` lines 207–216 (Event struct exists ✅)

---

## 4. r20 Fixture Description

**Path:** `/Volumes/Projects/gert/design/gert-v2/testdata/runbooks/r20-serve-rpc/schema.yaml`

**Purpose:** Minimal runbook for exercising `gert serve` step-by-step execution.

**Flow:**
1. **capture_env** (cli) — Echo environment variable, capture as `environment`
2. **branch_by_env** (branch) — Three paths:
   - `environment == "prod"` → prod_check + prod_confirm
   - `environment == "staging"` → staging_deploy
   - `true` (default) → dev_deploy
3. **complete** (cli) — Final echo with environment name

**Key features:**
- Uses infix condition syntax: `'environment == "prod"'`
- Variable capture from stdout
- Template expansion: `{{ .env | default "dev" }}`
- Short steps (echo only, no sleeps)
- Suitable for RPC-driven execution:
  - Client calls `exec/start` with `{"env": "prod"}`
  - Client calls `exec/next` 3–4 times to drive steps
  - Client subscribes to event stream via `/ws`

**Expected events (prod path):**
1. `run/started`
2. `step/started` (capture_env)
3. `step/completed` (capture_env)
4. `step/started` (branch_by_env)
5. `step/started` (prod_check)
6. `step/completed` (prod_check)
7. `step/started` (prod_confirm)
8. `step/completed` (prod_confirm)
9. `step/completed` (branch_by_env)
10. `step/started` (complete)
11. `step/completed` (complete)
12. `run/completed`

---

## 5. Recommendations for Brian

### 5.1 Implementation Order

1. **Phase 9.1:** Stdio JSON-RPC server only
   - `cmd/serve/main.go` with `--stdio` flag
   - `internal/serve/jsonrpc.go` implementing exec/v2 methods
   - Session map: `run_id -> RunHandle`
   - Event forwarding as JSON-RPC notifications

2. **Phase 9.2:** HTTP + WebSocket
   - Add `--http --port 8080` flags
   - HTTP handler: POST `/rpc` → JSON-RPC over HTTP
   - WebSocket handler: `/ws` → event stream
   - Health endpoint: GET `/health`

3. **Phase 9.3:** wait_for_event webhook transport
   - EventDispatcher implementation (already interfaced in pkg/eventbus)
   - Webhook handler: POST `/events/{run-id}/{event-id}`
   - HMAC-SHA256 signature verification
   - Persistence for WAITING runs across restart

### 5.2 Testing Strategy

**Unit tests:**
- `internal/serve/jsonrpc_test.go` — Test each RPC method
- `internal/serve/session_test.go` — Session map lifecycle
- `internal/serve/websocket_test.go` — Event streaming

**Integration tests:**
- Use r20-serve-rpc fixture
- Spin up serve in --stdio mode
- Send exec/start, exec/next via stdin
- Verify events on stdout

**Acceptance test:**
- Full HTTP server with WebSocket
- Browser client connects to `/ws`
- Verify real-time event delivery

### 5.3 Dependencies

**Required packages already exist:**
- `pkg/engine` — Engine and RunHandle interfaces ✅
- `pkg/eventbus` — EventBus and EventDispatcher interfaces ✅
- `pkg/planner` — For plan creation ✅
- `pkg/trace` — TraceWriter for JSONL ✅

**New dependencies needed:**
- JSON-RPC 2.0 library (or hand-roll — spec is simple)
  - Recommendation: `github.com/sourcegraph/jsonrpc2` (stdio transport)
- WebSocket library: `github.com/gorilla/websocket` (stdlib WS is low-level)

### 5.4 Error Codes

Section 13 references "canonical error code registry" (§13.1.7).

**Action:** Define error code constants in `internal/serve/errors.go`:
```go
const (
    ErrCodeRunNotFound      = 1001
    ErrCodeInvalidParams    = 1002
    ErrCodeInternalError    = 1003
    ErrCodeRunbookNotFound  = 1004
    ErrCodePlanningFailed   = 1005
    ErrCodeEngineStartFailed = 1006
    // ...
)
```

### 5.5 Crash Recovery

Per Section 02.7.2 (Crash Recovery in gert serve):

- Expose `exec/list` and `exec/resume` methods
- On restart, scan trace dir for runs in WAITING state
- Call `Engine.Resume(runID)` to restore RunHandle
- Re-register EventDispatcher listeners from checkpoint

**Implementation note:** RunStore must persist:
- Run state (RunState struct)
- EventDispatcher listener registrations

---

## 6. Open Questions for Cristian

1. **serve package location:**
   - Keep as `internal/serve` (my recommendation), OR
   - Expose public API at `pkg/serve` for embedding in other Go programs?

2. **JSON-RPC library choice:**
   - Hand-roll (spec is simple, ~200 LOC)
   - Use `github.com/sourcegraph/jsonrpc2`
   - Use `github.com/powerman/rpc-codec/jsonrpc2`

3. **HTTP server framework:**
   - stdlib `net/http` (my recommendation for minimal deps)
   - `github.com/gorilla/mux` (if complex routing needed)

4. **Authentication/RBAC:**
   - Section 07.7 mentions RBAC for `gert serve`
   - Is this Phase 9 scope, or deferred to Phase 10?

---

## Summary

**Ready to implement:** Engine surface is complete, eventbus exists, r20 fixture created.

**Missing:** All serve packages (`cmd/serve`, `internal/serve`).

**Spec clarity:** Excellent. Section 02.7, 06, 13 provide complete contracts.

**Risk:** Low. serve is a thin adapter over existing engine APIs.

**Estimated scope:** ~1500 LOC for stdio + HTTP + WebSocket + basic error handling.

---

**Next steps:**
1. Brian creates `cmd/serve/main.go` skeleton
2. Implement exec/v2 methods against Engine interface
3. Test with r20 fixture via stdio transport
4. Add HTTP + WebSocket in second iteration
