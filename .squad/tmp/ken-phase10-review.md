# Phase 10 — Adapters: Architecture Review

**Reviewer:** Ken (Software Architect)  
**Date:** 2026-04-24  
**Commit range:** Phase 10 implementation (22 new packages)  
**Validation gate:** `go build ./... && go vet ./... && go test ./... -race -count=1` — ALL PASS

---

## Verdict: ✅ APPROVE

**Average score: 7.5 / 10** — threshold met (≥ 7.0)  
**Minimum score: 6 / 10** — no dimension below 5

---

## Dimension Scores

| # | Dimension | Score | Notes |
|---|-----------|-------|-------|
| 1 | Stub replacement | **9** | All 5 Phase 9 stubs replaced. Serve test confirms real Dispatcher + JSONLWriter. |
| 2 | Wiring harness | **7** | Functional but deviates: no `io.Closer` return, no `plat` parameter, MultiWriter not used to tee into EventBus. |
| 3 | JSONL trace writer | **8** | Thread-safe, append-only, O_APPEND, Close flushes via Sync. HMAC signing deferred (acceptable). |
| 4 | EventBus | **9** | Clean RWMutex fan-out, KindPrefix+RunID filtering, non-blocking publish, ctx-cancel unsubscribe. Map-keyed-by-channel approach is elegant. |
| 5 | Dispatcher | **8** | First-waiter-wins ✓, buffered pending ✓, filter matching ✓, ctx cancel ✓, timeout ✓. Missing: bounded pending buffer (spec says 1000/channel). |
| 6 | gert run CLI | **6** | Missing `--mode` and `--as` flags from spec §7.2. Exit code mapping differs (cancelled→1 not 2). `--output json` is a welcome addition. |
| 7 | gert dry-run | **7** | DryRunExecutorRegistry wrapping correct. `[DRY-RUN]` labels present. Missing: phase-level output (parse/plan/governance lines from §8.4). Assert steps wrapped uniformly (spec says run real assert). |
| 8 | Test coverage | **7** | ~25 tests, good concurrency tests (ThreadSafe, ConcurrentPublish). Missing spec'd tests: WaitTimeout, BufferedEventConsumed, DropOnFull, DryRun governance report, webhook delivery. |
| 9 | Deviations | **7** | Webhook HMAC (`GERT_WEBHOOK_KEY`) is welcome security addition. Nested engine for substeps is heavier than spec'd callback approach but ensures correctness. Both acceptable. |
| 10 | Code quality | **7** | Idiomatic Go, no goroutine leaks, clean boundaries. `runSubStepsViaEngine` 15-param signature is a smell. `fileRunbookLoader`/`plannerToolRegistry` duplicated across cmd packages. |

---

## Blocking Issues

None. All dimensions ≥ 5 and average ≥ 7.0.

---

## Non-Blocking Issues (address in Phase 11 or follow-up)

### NB-1: Missing `io.Closer` from `BuildEngineConfig` (Wiring Harness)

**Severity:** Medium  
**Spec ref:** §3.2 — "Returns an `io.Closer` that closes the trace writer, stops the webhook listener, and drains the event bus."

The implementation returns `(EngineConfig, error)` instead of `(EngineConfig, io.Closer, error)`. This means:
- JSONLWriter is never explicitly closed/flushed by callers
- Webhook listener goroutine has no explicit shutdown path beyond ctx cancellation
- Resource cleanup relies entirely on context cancellation + process exit

**Recommendation:** Add a `Closer` that wraps `traceWriter.Close()` + `webhookListener.Stop()`. Both `cmd/serve` and `cmd/gert` should `defer closer.Close()`.

### NB-2: MultiWriter not integrated into wiring

**Severity:** Low  
**Spec ref:** §5.2 — "Used in BuildEngineConfig to combine JSONLWriter + EventBusForwarder"

`BuildEngineConfig` creates both a `JSONLWriter` and an `EventBus` but doesn't tee them via `MultiWriter`. The `EventBus` only receives events if callers publish explicitly. Trace events don't automatically flow to EventBus subscribers.

**Recommendation:** Wire `MultiWriter(jsonlWriter, eventBusForwarder)` as the `TraceWriter` in `EngineConfig`.

### NB-3: `runSubStepsViaEngine` has 15 parameters

**Severity:** Low  
**File:** `internal/adapter/wire.go`

This function signature is hard to maintain. Every new engine dependency requires adding another parameter.

**Recommendation:** Accept an `EngineConfig` or a purpose-built `SubStepConfig` struct.

### NB-4: Duplicated types across cmd packages

**Severity:** Low  
**Files:** `cmd/gert/run.go` and `cmd/serve/main.go` both define `fileRunbookLoader` and `plannerToolRegistry`.

**Recommendation:** Move to `internal/adapter` alongside `BuildEngineConfig`.

### NB-5: Missing CLI flags from spec

**Severity:** Low  
**Spec ref:** §7.2

| Flag | Status |
|------|--------|
| `--trace` (file) | ✓ (replaces `--trace-dir`; acceptable alternative) |
| `--output` | ✓ (enhancement over spec) |
| `--var` | ✓ |
| `--mode` | ✗ Missing — `gert run` cannot switch to dry-run without separate `gert dry-run` command |
| `--as` | ✗ Missing — actor identity not settable from CLI |
| `--tool-dir` | ✗ Missing — hardcoded to "." |

**Recommendation:** Add `--mode` and `--as` before Phase 11 (replay mode needs `--mode=replay`).

### NB-6: Exit code mapping

**Severity:** Low  
**Spec ref:** §7.4

Spec: `cancelled → exit 2`. Impl: `cancelled → exit 1` (same as failed), `validation → exit 2`.

This is a defensible alternative mapping (validation errors are a distinct failure class). Document the chosen mapping.

### NB-7: Unbounded pending buffer in Dispatcher

**Severity:** Low  
**Spec ref:** §4.4 — "per channel is capped at 1000 events"

`Dispatcher.pending` grows without bound. In production with a chatty webhook source, this could accumulate memory.

**Recommendation:** Add eviction in `Dispatch()` when `len(pending[channel]) > 1000`.

### NB-8: Webhook `verifySignature` fail-open

**Severity:** Low  
**File:** `internal/eventbus/webhook.go`

When `GERT_WEBHOOK_KEY` is empty, all requests are accepted. This is documented and consistent with the design's "Phase 10 does not add auth" stance, but the fail-open default should be logged clearly at startup.

### NB-9: Missing spec'd tests

**Severity:** Low

Tests not present from the test plan (§11):
- `TestDispatcher_WaitTimeout`
- `TestDispatcher_BufferedEventConsumed`
- `TestBus_DropOnFull`
- `TestGertDryRun_NoSideEffects`
- `TestGertDryRun_GovernanceReport`
- `TestServe_TraceFileWritten`
- `TestServe_EventDispatcher_WebhookDelivery`

The existing 25 tests provide adequate coverage. These can be added incrementally.

---

## Positive Observations

1. **DryRunExecutorRegistry pattern** — clean decorator, same engine pipeline, no conditional branches scattered through the codebase. Exactly as designed.

2. **EventBus subscriber map keyed by `<-chan`** — more efficient than the linear slice + ID counter in the design. Unsubscribe is O(1) instead of O(n).

3. **Webhook HMAC** — Brian added `GERT_WEBHOOK_KEY` HMAC verification proactively. The design explicitly said auth was out of scope, but this is a welcome security improvement with minimal complexity.

4. **`--output json`** — enables machine consumption of `gert run` output (CI/CD pipelines). Not in the spec but valuable.

5. **`splitRunArgs`** — allows `gert run <path> --flags` ordering, which is more ergonomic than requiring flags before the positional argument.

6. **Test patterns** — concurrency tests use `sync.WaitGroup` barriers properly, cleanup via `t.Cleanup`, temp dirs via `os.MkdirTemp(".", ...)` (not system temp).

---

## Substep Runner Decision Assessment

Brian chose **nested engine instantiation** instead of the spec'd **OnStepEvent callback approach** (D5). Assessment:

| Aspect | Callback (spec'd) | Nested Engine (implemented) |
|--------|-------------------|----------------------------|
| Event emission | Via callback injection | Full engine lifecycle |
| Governance pre-flight | Manual check | Automatic (engine handles it) |
| Variable merging | Manual | Engine's standard var flow |
| Performance | Lighter (no engine alloc) | Heavier (full engine per substep block) |
| Correctness surface | Partial (must wire manually) | Full (engine handles everything) |
| Parameter sprawl | Lower | Higher (15 params) |

**Verdict:** Acceptable. The nested engine approach trades allocation cost for correctness. For Phase 10's goal of "no fakes, no stubs," this is a pragmatic choice. The 15-parameter signature should be refactored (NB-3) before Phase 11 adds more state.

---

## Phase 11 Readiness

Phase 10 establishes the foundation for Phase 11 (Evidence & Replay):
- ✅ Trace writer is real and durable
- ✅ RunHandle.Next() loop pattern works for replay
- ⚠️ `--mode=replay` will need the `--mode` flag added to `gert run` (NB-5)
- ⚠️ Trace path convention (`{runID}.jsonl`) not enforced — replay will need to locate trace files
- ✅ RunStore interface exists in `pkg/engine/run.go` ready for implementation

---

*Ken, Software Architect*
