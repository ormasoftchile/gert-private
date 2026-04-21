# Phase 9 Code Review — `gert serve`

**Reviewer:** Ken (Architect)
**Date:** 2026-04-23
**Implementor:** Brian
**Verdict:** **APPROVED**

---

## Scores

| # | Dimension              | Score | Notes |
|---|------------------------|-------|-------|
| 1 | Correctness            | 8     | JSON-RPC 2.0 error codes match spec; all 5 methods correct; extra-JSON guard is a nice touch |
| 2 | Architecture           | 9     | Dependency rules fully respected: `internal/serve` has zero imports from `internal/*`; `pkg/serve` is a leaf |
| 3 | Concurrency            | 8     | RunRegistry uses RW mutex correctly; EventBridge fan-out is race-safe; `pumpEvents` goroutine tracked via `sync.WaitGroup` |
| 4 | Test coverage          | 7     | 24/24 design-specified tests present and meaningful; no stubs; `events_test.go` file is absent but EventBridge is exercised indirectly through WS/SSE integration tests |
| 5 | Engine integration     | 8     | `run.start` → `Engine.Start` → `RunHandle` → registry + pump; `run.next` → `handle.Next` → `io.EOF` mapping to `-32012`; all correct |
| 6 | Event delivery         | 8     | WS: JSON text frames; SSE: `event:/id:/data:` format correct; subscription filtering by `runID`; SSE `Last-Event-ID` replay works; slow-consumer drop confirmed by test |
| 7 | Server lifecycle       | 8     | Graceful shutdown: cancel active runs → close bridge → `http.Server.Shutdown` → `wg.Wait`; GC goroutine respects context; `stopOnce` prevents double-close |
| 8 | Middleware             | 8     | Request ID (UUID), structured JSON logging, CORS, panic recovery all present; `statusRecorder` implements `Flusher`+`Hijacker`+`Unwrap` for WS/SSE compatibility |
| 9 | Deviations             | 7     | See §Deviations below — acceptable given Phase 10 scope |
| 10 | Code quality           | 8     | Clean idiomatic Go; no dead code; good use of `json.RawMessage` for RPC ID passthrough; `normalizeID` handles null IDs per spec |

**Overall Score: 7.9 / 10 — APPROVED**

---

## Key Findings

### Blocking Issues

**None.** No dimension scores below 5. The implementation is solid and merge-ready.

### Non-Blocking Issues (address in Phase 10 or follow-up)

#### NB-1: Missing `events_test.go` (Test Coverage)

The design specifies `events_test.go` as a separate file for EventBridge unit tests. Brian chose to exercise EventBridge indirectly through WS and SSE integration tests, which do validate Broadcast/Subscribe/Unsubscribe/Replay. However, a dedicated unit test file would improve isolation and make EventBridge easier to refactor independently.

**Files:** Missing `internal/serve/events_test.go`
**Severity:** Non-blocking (EventBridge IS tested, just not in isolation)

#### NB-2: EventBridge.Broadcast acquires lock twice (events.go:88-116)

`Broadcast` does a read-lock check for `closed`, releases, then acquires a write-lock for history, releases, then acquires a read-lock for fan-out. This is three separate lock acquisitions. While correct (the closed check is a fast-path exit), a single write-lock for history + read-lock for fan-out would be cleaner:

```go
// Current: RLock(closed check) → Unlock → Lock(history) → Unlock → RLock(fan-out) → Unlock
// Suggested: Lock(closed check + history) → Unlock → RLock(fan-out) → Unlock
```

**File:** `internal/serve/events.go:88-116`
**Severity:** Non-blocking (race-safe as-is, minor inefficiency)

#### NB-3: `mapRunbookError` uses string matching for parse/validation errors (rpc.go:455-463)

The fallback error mapping checks for substrings like `"yaml:"`, `"["`, and `"validation"` in error messages. This is fragile — if error messages change upstream, the classification breaks. Consider introducing typed errors (e.g., `parser.ParseError`, `parser.ValidationError`) in Phase 10 to replace string matching.

**File:** `internal/serve/rpc.go:455-463`
**Severity:** Non-blocking (works today; typed errors are a cleaner long-term solution)

#### NB-4: `handleRunCancel` ignores `Handle.Cancel` error (rpc.go:308)

```go
_ = entry.Handle.Cancel(r.Context(), params.Reason)
```

If `Cancel` returns an error (e.g., the run transitioned between the state check and the cancel call), the client still gets a success response. In practice this is unlikely to cause issues since the registry state is updated to cancelled regardless, but logging the error would improve debuggability.

**File:** `internal/serve/rpc.go:308`
**Severity:** Non-blocking (defensive; log the error)

#### NB-5: Recovery middleware swallows panic details (middleware.go:116-118)

The recovery middleware catches panics but doesn't log the recovered value or stack trace. The design doc specifies "catches panics, logs stack trace, returns 500." Consider adding:

```go
log.Printf("panic recovered: %v\n%s", rec, debug.Stack())
```

**File:** `internal/serve/middleware.go:113-121`
**Severity:** Non-blocking (functional but loses diagnostic info)

#### NB-6: `cmd/serve/main.go` uses `TerminalInputProvider` for server context (main.go:88)

A network server using `os.Stdin` for interactive prompts is inappropriate — there may be no TTY. This is acceptable for Phase 9's noop/dev mode, but Phase 10 must replace this with an RPC-backed input provider that forwards prompts to the connected client.

**File:** `cmd/serve/main.go:88`
**Severity:** Non-blocking (known; Phase 10 Adapters scope)

---

## Deviations Assessment (Dimension 9)

Brian noted that `ServerConfig` includes `Engine`/`EngineFactory`/`Parser`/`Planner` fields, and `cmd/serve` uses real implementations with noop stubs for `EventDispatcher` and `TraceWriter`.

**Assessment: Acceptable.**

1. **`Engine`/`EngineFactory` dual path** — Good design. Allows tests to inject a fake engine directly while production uses the factory. This is a standard Go pattern (accept interface or constructor).

2. **`Parser`/`Planner` on ServerConfig** — Correct. The server needs these to implement the `run.start` flow (parse → plan → start). The design doc's `§6.2` explicitly shows this pipeline.

3. **Noop `EventDispatcher`/`TraceWriter` in `cmd/serve`** — Acceptable for Phase 9. The real implementations will be wired in Phase 10 (Adapters). Brian correctly used `testutil.NewFakeEventDispatcher()` and a `discardTraceWriter{}` as explicit placeholders rather than passing nil (which would fail `EngineConfig.Validate()`).

4. **`noopToolRegistry` and `fileRunbookLoader`** in `cmd/serve` — Reasonable local types. The tool registry returns `ErrToolNotFound` for all lookups, which is correct for a server that doesn't yet load tool definitions. These will be replaced when the real ToolRegistry adapter lands.

**None of these deviations are blocking.** They are explicit, documented, and scoped for Phase 10 resolution.

---

## Positive Observations

1. **`ensureNoExtraJSON`** (rpc.go:412-421) — Rejects payloads with trailing garbage after the JSON object. This is a subtle JSON-RPC 2.0 compliance detail that many implementations miss. Well done.

2. **`normalizeID`** (rpc.go:405-410) — Correctly handles the case where the client omits the `id` field by returning `"null"` instead of an empty/zero value. Per JSON-RPC 2.0 spec, error responses for parse errors MUST have `id: null`.

3. **Copy semantics in `RunRegistry.Get`** (registry.go:46-56) — Returns a value copy of `RunEntry`, preventing callers from mutating shared state without holding the lock. The `WithEntry` method provides the locked-mutation path. This is clean concurrency hygiene.

4. **`statusRecorder` implements `http.Flusher`, `http.Hijacker`, `http.Pusher`, `Unwrap`** (middleware.go:44-77) — This ensures the logging middleware doesn't break SSE flushing or WebSocket upgrades. Many middleware implementations forget to proxy these interfaces.

5. **EventBridge replay buffer** (events.go) — Bounded history with `since(seq)` query supports SSE `Last-Event-ID` reconnection cleanly. The ring-buffer trim in `add()` prevents unbounded memory growth.

6. **Test harness design** (test_helpers_test.go) — Clean separation of fakes with injectable behaviors (`startFunc`, `results` slice, `nextErr`). The `testServerHarness` struct makes it easy to configure different scenarios.

---

## Test Inventory (24/24)

| # | Test Name | File | Status |
|---|-----------|------|--------|
| 1 | TestRPC_RunStart_Success | rpc_test.go | ✅ |
| 2 | TestRPC_RunStart_InvalidParams | rpc_test.go | ✅ |
| 3 | TestRPC_RunStart_RunbookNotFound | rpc_test.go | ✅ |
| 4 | TestRPC_RunNext_Advances | rpc_test.go | ✅ |
| 5 | TestRPC_RunNext_EOF | rpc_test.go | ✅ |
| 6 | TestRPC_RunNext_RunNotFound | rpc_test.go | ✅ |
| 7 | TestRPC_RunCancel_Stops | rpc_test.go | ✅ |
| 8 | TestRPC_RunStatus_ReflectsState | rpc_test.go | ✅ |
| 9 | TestRPC_RunList_ShowsActiveRuns | rpc_test.go | ✅ |
| 10 | TestRPC_UnknownMethod | rpc_test.go | ✅ |
| 11 | TestRPC_MalformedJSON | rpc_test.go | ✅ |
| 12 | TestRPC_MissingJsonrpcField | rpc_test.go | ✅ |
| 13 | TestWS_ConnectReceivesEvents | ws_test.go | ✅ |
| 14 | TestWS_FilterByRunID | ws_test.go | ✅ |
| 15 | TestWS_RunCompleted_ReceivesTerminal | ws_test.go | ✅ |
| 16 | TestWS_SlowConsumer_DropsEvents | ws_test.go | ✅ |
| 17 | TestSSE_ConnectReceivesEvents | sse_test.go | ✅ |
| 18 | TestSSE_LastEventID_Replay | sse_test.go | ✅ |
| 19 | TestSSE_FilterByRunID | sse_test.go | ✅ |
| 20 | TestHealth_Returns200 | health_test.go | ✅ |
| 21 | TestRegistry_ConcurrentAddGet | registry_test.go | ✅ |
| 22 | TestRegistry_GC_RemovesOldRuns | registry_test.go | ✅ |
| 23 | TestServer_StartStop_Graceful | server_test.go | ✅ |
| 24 | TestServer_CancelActiveRuns_OnStop | server_test.go | ✅ |

---

## Verdict

**APPROVED — Phase 10 unlocked.**

The implementation faithfully follows the design doc, respects all dependency rules, passes all 24 required tests with `-race`, and correctly integrates with the engine interfaces. The 6 non-blocking items are noted for follow-up but do not warrant rejection. This is a clean, idiomatic adapter layer that improves gert v2's overall code health.
