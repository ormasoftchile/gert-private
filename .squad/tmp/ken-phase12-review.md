# Ken — Phase 12 Architectural Review: OpenTelemetry Integration

**Date:** 2026-07-20  
**Reviewer:** Ken (Software Architect)  
**Implementor:** Brian  
**Phase:** 12 — OpenTelemetry Span Integration, RunStore Tests, Atomic Writes  
**Depends on:** Phase 11 (Evidence & Replay, sealed at review approval)

---

## VERDICT: APPROVED WITH NON-BLOCKING ITEMS [8.5/10]

Phase 12 delivers a clean, zero-dependency OTel integration that precisely matches my design
intent. Brian's custom interfaces (TracerProvider, Tracer, Span) are OTel API-compatible but
avoid the ~30MB SDK dependency — a pragmatic tradeoff I endorse. The span hierarchy is correct
(gert.run → gert.step.{kind} → gert.branch.{label}), context propagation is working, and
all Phase 11 non-blocking items are addressed. Two deviations require acknowledgment but are
acceptable for v2.0.

---

## Validation Gate

### Build

```
go build ./...   → ✅ BUILD OK
```

All Phase 12 packages compile cleanly:
- `pkg/otel` ✅ (new package)
- `pkg/engine` ✅ (TracerProvider field added)
- `internal/engine` ✅ (span instrumentation)
- `internal/adapter` ✅ (buildTracerProvider, stdoutTracerProvider)
- `internal/runstore` ✅ (unchanged, tests added)
- `internal/evidence` ✅ (atomic writes, warn logging)
- `cmd/gert` ✅ (CLI flags)

### Tests

```
go test ./... -race -count=1   → ✅ ALL PASS (35 packages)
```

**Phase 12 test functions verified:**

| Package | Tests | Coverage |
|---------|-------|----------|
| `pkg/otel` | 11 | NoopTracer, RecordingProvider, SpanHierarchy, Status, Error, Attributes, Context, ResolveProvider |
| `internal/runstore` | 9 | SaveLoadState, Latest, SkipsTmpFiles, NotFound, WriteTrace, RegisterPlan, TracePath, Concurrent (10×100 goroutines), AtomicPartialWrite |
| `internal/engine` | 4 (OTel) | OTelSpans_Noop, OTelSpans_Hierarchy, OTelSpans_Error, OTelSpans_Parallel |

---

## Phase 11 Non-Blocking Items: Status

| Item | Spec | Status |
|------|------|--------|
| NBI-1: Add RunStore unit tests | 6 test cases spec'd | ✅ **9 tests implemented** — exceeds spec |
| NBI-2: Atomic attachment writes | temp → sync → rename | ✅ **Implemented** — attachment.go:57-80 |
| NBI-3: Warn logging for attachment errors | log at warn level | ✅ **Implemented** — collector.go:88,100 `[WARN]` |

All three Phase 11 housekeeping items are complete. Brian addressed them before beginning
Part B (OTel integration).

---

## Five-Axis Review

### 1. Correctness — Does it match the spec? ✅ (9/10)

**Fully compliant areas:**

- **D-12-01 (Pluggable interface with noop default):** ✅
  - `pkg/otel/tracer.go` defines `TracerProvider`, `Tracer`, `Span` interfaces
  - `noopTracerProvider`, `noopTracer`, `noopSpan` provide zero-allocation defaults
  - `ResolveProvider(nil)` returns noop — engine never panics
  - No `go.opentelemetry.io/otel` dependency (verified via `go.mod`)

- **D-12-02 (Span hierarchy):** ✅
  - `gert.run` span started at engine.go:93 with run-level attributes
  - `gert.step.{kind}` spans started at engine.go:358, correctly inherits from run span via context
  - `gert.branch.{label}` spans started at engine.go:646, correctly inherits from parallel step span
  - Parent-child verified by `TestEngine_OTelSpans_Hierarchy` and `TestEngine_OTelSpans_Parallel`

- **D-12-03 (Context propagation):** ✅
  - `traceCtx` carried in `runHandle` (engine.go:111, 289)
  - `spanCtx` passed to executors (engine.go:485)
  - `otel.ContextWithSpan()` / `otel.SpanFromContext()` helpers work correctly (tracer.go:118-129)

- **D-12-04 (Attributes with gert.* prefix):** ✅
  - `attributes.go` defines 12 constants: `AttrRunID`, `AttrRunbookPath`, `AttrRunMode`,
    `AttrActor`, `AttrStepID`, `AttrStepKind`, `AttrStepIndex`, `AttrStepDuration`,
    `AttrToolName`, `AttrToolAction`, `AttrToolTransport`, `AttrInputType`, `AttrInputProvider`
  - All use `gert.*` prefix per spec

- **D-12-05 (Pluggable export):** ⚠️ Partial
  - `--otel-stdout` flag ✅ — prints JSON span summaries to stderr
  - `--otel-endpoint` flag ✅ parsed — but wiring deferred (see BRD-12-02)
  - `--otel-service` flag ✅ — passed to stdoutTracerProvider

- **D-12-06 (Precise span open/close):** ✅
  - Run span opened at `eng.Start()`, closed at `runHandle.close()` / error / completion
  - Step spans opened at step execution start, `defer stepSpan.End()` ensures close
  - Branch spans follow same pattern with proper defer

- **D-12-07 (OTel spans complement NDJSON trace):** ✅
  - Trace events still emitted via `TraceWriter.Append()` (unchanged from Phase 11)
  - OTel spans are orthogonal — can be enabled/disabled independently

**RunStore test coverage:**

The 9 tests comprehensively cover:
1. `TestDirRunStore_SaveLoadState` — round-trip serialization ✅
2. `TestDirRunStore_LoadState_Latest` — descending index selection ✅
3. `TestDirRunStore_LoadState_SkipsTmpFiles` — `.tmp` file ignored ✅
4. `TestDirRunStore_LoadState_Empty` — returns `os.ErrNotExist` ✅
5. `TestDirRunStore_WriteTrace` — JSONL event serialization ✅
6. `TestDirRunStore_RegisterPlan` — plan caching ✅
7. `TestDirRunStore_TracePath` — path construction ✅
8. `TestDirRunStore_Concurrent` — 10 goroutines × 100 events (race-safe) ✅
9. `TestWriteFileAtomic_PartialWrite` — no `.tmp` files remain ✅

**Attachment atomic write verified:**

```go
// attachment.go:57-80
tmpPath := destPath + ".tmp"
dst, err := os.Create(tmpPath)
// ... io.Copy, Sync, Close with cleanup on each error ...
if err := os.Rename(tmpPath, destPath); err != nil {
    _ = os.Remove(tmpPath)
    return nil, fmt.Errorf(...)
}
```

Pattern is correct: temp → sync → rename with cleanup on all error paths. ✅

**Attachment warn logging verified:**

```go
// collector.go:88
log.Printf("[WARN] evidence: failed to store attachment %s: %v", path, err)
```

Both `[]any` and `[]string` attachment formats have the same logging. ✅

### 2. Readability — Clear naming, consistent patterns? ✅ (9/10)

- **Package structure:** `pkg/otel` is a leaf package — no internal imports. Clean.
  
- **Interface design:** Matches OTel trace API naming conventions (`TracerProvider`, `Tracer`,
  `Span`, `StatusCode`). Go developers familiar with OTel will recognize the patterns.

- **doc.go:** Excellent package documentation explaining the zero-dependency design and
  bridging strategy.

- **Noop naming:** `noopTracerProvider`, `noopTracer`, `noopSpan` — lowercase private types,
  exposed via `NoopTracerProvider()` function. Standard Go pattern.

- **Recording provider:** `RecordingTracerProvider` with `Spans()` method returning snapshot.
  Thread-safe, test-friendly.

- **Engine integration:** `tracer()` helper method (engine.go:39-40) abstracts noop fallback.
  Clean injection point.

- **Minor nit:** `spanCounter` and `spanMu` (tracer.go:235-236) are package-level globals for
  test span ID generation. This is fine for tests but would be problematic if the recording
  provider were used outside tests. Document intent in comment.

### 3. Architecture — Clean boundaries, correct patterns? ✅ (9/10)

**Package dependency graph (verified, no cycles):**

```
pkg/otel              → stdlib only (leaf ✅)
pkg/engine            → pkg/otel (TracerProvider field)
internal/engine       → pkg/otel (span instrumentation)
internal/adapter      → pkg/otel (wire.go buildTracerProvider)
cmd/gert              → internal/adapter (CLI flags)
```

**Pattern consistency:**

- `stdoutTracerProvider` in `wire.go` follows the same pattern as noop: implements interface,
  returns custom span type. No SDK dependency.

- Recording provider parallels `fakeTraceWriter` in engine_test.go — both are test doubles
  that capture and expose data for assertions.

- CLI flag wiring follows established pattern: parse in main.go, pass to WireOptions, use in
  buildTracerProvider.

**Architecture decisions:**

- **BRD-12-01 (No OTel SDK):** The custom interfaces are a **good** architectural choice.
  They isolate gert from the OTel SDK's transitive dependencies (~30MB) while remaining
  wire-compatible. A thin adapter in Phase 13 can bridge to the real SDK when needed.
  This is the right call for v2.0.

- **Context merging pattern:** `mergeContexts()` (engine.go:1404-1414) creates a derived
  context cancelled when either parent is cancelled. This is correct for combining trace
  context with run/step cancellation. The goroutine-per-call concern (BRD-12-03) is valid
  but bounded by run/step lifecycle — see Performance section.

### 4. Security — Safe handling? ✅ (9/10)

- **Attribute values:** Attributes use `any` type. Values flow from engine (runID, runbookPath,
  stepID, stepKind) which are all controlled strings. No user-controlled data enters attributes
  that could cause injection issues in downstream OTel backends.

- **stdout provider:** Writes to stderr (not stdout) to avoid polluting command output. Uses
  JSON encoding which escapes special characters. ✅

- **No network exposure:** The OTLP endpoint flag is parsed but not wired — no network calls
  in v2.0. When Phase 13 adds OTLP export, standard TLS/auth patterns should be followed.

- **Span data lifetime:** `SpanData` structs in RecordingTracerProvider hold references to
  attribute values. For test use, this is fine. Production providers (Phase 13) should copy
  values to avoid retaining references to executor data.

### 5. Performance — Bounded overhead? ✅ (8/10)

**Noop path (default):**
- `noopTracer.Start()` returns `ctx, noopSpan{}` — zero allocation
- `noopSpan` methods are empty — no overhead
- ✅ Correct implementation of "zero overhead when disabled" requirement

**Recording path (tests):**
- One `SpanData` allocation per span
- Mutex-protected append to slice
- Sequential span IDs via `newSpanID()` — mutex-protected counter
- Acceptable for test use; not designed for production tracing

**stdout provider:**
- One `fmt.Fprintf` per span end — minimal overhead
- No buffering or batching (acceptable for debug use)

**mergeContexts goroutine concern (BRD-12-03):**

Brian correctly flagged this in his handoff. Analysis:

```go
func mergeContexts(a, b context.Context) context.Context {
    merged, cancel := context.WithCancel(a)
    go func() {
        defer cancel()
        select {
        case <-b.Done():
        case <-merged.Done():
        }
    }()
    return merged
}
```

- Called 3× per step: lines 309, 311 (in Next), and once more in parallel branches
- Goroutine exits when either context is done
- For a 100-step run: ~300 goroutines created, all exit by run completion
- For a 1000-step run: ~3000 goroutines — still bounded and transient

**Verdict:** Acceptable for v2.0. The goroutines are short-lived (step duration) and bounded
(3 × step count). If profiling shows this as a bottleneck in v2.1, consider using
`context.AfterFunc` (Go 1.21+) or a channel-based approach without goroutines.

---

## Deviation Assessment

| # | Deviation | Spec Reference | Impact | Verdict |
|---|-----------|---------------|--------|---------|
| BRD-12-01 | No `go.opentelemetry.io/otel` SDK | D-12-01 (pluggable interface) | Custom interfaces instead of real SDK | ✅ **Accepted** — pragmatic for v2.0, Phase 13 adapter |
| BRD-12-02 | OTLP endpoint no-op | D-12-05 (pluggable export) | Flag parsed but not wired | ⚠️ **Accepted** — document as v2.1, stdout is sufficient for v2.0 |
| BRD-12-03 | mergeContexts goroutine-per-call | N/A (impl detail) | 3 goroutines per step | ✅ **Accepted** — bounded, transient, profile in v2.1 |

### BRD-12-01: No Real OTel SDK

**Rationale:** Brian reports ~30MB transitive dependencies from `go.opentelemetry.io/otel`.
This would significantly impact binary size and build times.

**My assessment:** The custom interfaces (`TracerProvider`, `Tracer`, `Span`) are a subset of
the OTel trace API. They're designed for wire-compatibility — a thin adapter can implement
gert's `TracerProvider` by wrapping the real `trace.TracerProvider`. This is the right
architecture: gert defines what it needs, adapters bridge to implementations.

**Verdict:** ✅ Accepted. Phase 13 should deliver a `otel/adapter` package that:
1. Imports `go.opentelemetry.io/otel`
2. Wraps `trace.TracerProvider` to implement `otelPkg.TracerProvider`
3. Users opt-in via build tag or separate binary

### BRD-12-02: OTLP Endpoint No-Op

**Rationale:** Without the real SDK, there's no OTLP exporter to wire.

**My assessment:** The flag is parsed and stored in `WireOptions.OTelEndpoint`. The wiring
point in `buildTracerProvider` returns nil when set (currently). This is the correct seam
for Phase 13 to add OTLP support.

**Verdict:** ⚠️ Accepted for v2.0. Document in release notes that `--otel-endpoint` is
reserved but not yet functional. Phase 13 should wire OTLP via the real SDK adapter.

### BRD-12-03: mergeContexts Goroutine Pattern

**Rationale:** Each call spawns a goroutine to watch the secondary context.

**My assessment:** This is a standard pattern for multi-parent context cancellation in Go.
The alternatives are:
- `context.AfterFunc` (Go 1.21+) — requires minimum version bump
- Channel-based without goroutines — more complex, similar performance

For v2.0 with typical runs of 10-100 steps, this is negligible overhead.

**Verdict:** ✅ Accepted. Add to "v2.1 performance review" list if profiling shows concern.

---

## Non-Blocking Items for Phase 13 Part A

### NBI-12-01: Add OTLP Adapter Package

Create `pkg/otel/adapter` (or separate module) that:
- Imports `go.opentelemetry.io/otel`
- Provides `NewOTLPTracerProvider(endpoint string, opts ...Option) otelPkg.TracerProvider`
- Wraps real `trace.TracerProvider` to implement gert's interface
- Wire in `buildTracerProvider` when `OTelEndpoint != ""`

**Priority:** Medium (enables production tracing)  
**Estimate:** 4-6 hours  
**Owner:** Brian

### NBI-12-02: Document OTLP Flag Status

Update CLI help and documentation:
- Mark `--otel-endpoint` as "reserved for future use" in `--help` output
- Add to release notes: "OTLP export coming in v2.1"

**Priority:** Low (user clarity)  
**Estimate:** 30 minutes  
**Owner:** Brian

### NBI-12-03: Consider context.AfterFunc for Go 1.21+

If gert's minimum Go version is raised to 1.21, replace `mergeContexts` goroutine with
`context.AfterFunc` for lower overhead context merging.

**Priority:** Low (performance optimization)  
**Estimate:** 1 hour  
**Owner:** Future optimization pass

---

## Test Quality Assessment

**Strengths:**
- OTel noop path tested (doesn't panic with nil provider)
- Span hierarchy verified via RecordingTracerProvider
- Status and error propagation tested
- Parallel branch spans tested
- RunStore concurrent write safety tested with race detector
- All tests pass with `-race -count=1`

**Gaps (non-blocking):**
- No benchmark comparing noop vs recording overhead
- No integration test with real OTel SDK (requires BRD-12-01 resolution)
- No test for stdoutTracerProvider output format

---

## Summary Scorecard

| Axis | Score | Notes |
|------|-------|-------|
| Correctness | 9/10 | Spec implemented correctly, hierarchy verified |
| Readability | 9/10 | Clean interfaces, good documentation |
| Architecture | 9/10 | Zero-dep design is correct, clean seams for Phase 13 |
| Security | 9/10 | Safe attribute handling, no network exposure yet |
| Performance | 8/10 | Noop path is zero-overhead, goroutine pattern acceptable |
| **Overall** | **8.5/10** | |

---

## Final Verdict

```
VERDICT: APPROVED WITH NON-BLOCKING ITEMS [8.5/10]
```

**Ships as-is.** Phase 12 delivers:
1. ✅ All Phase 11 housekeeping complete (RunStore tests, atomic writes, warn logging)
2. ✅ Functional OTel span integration with correct hierarchy
3. ✅ Zero-overhead noop default
4. ✅ stdout debug provider for immediate use
5. ⚠️ OTLP wiring deferred (documented)

**Non-blocking items for Phase 13 Part A:**
1. NBI-12-01: Add OTLP adapter package (4-6h)
2. NBI-12-02: Document OTLP flag status (30min)
3. NBI-12-03: Consider context.AfterFunc for Go 1.21+ (future)

Brian: excellent execution. The decision to build OTel-compatible interfaces without the SDK
dependency is architecturally sound — it keeps gert lean while preserving the option for full
OTel integration. The span hierarchy is correct, the tests verify the critical paths, and
all Phase 11 housekeeping is addressed. Ship it.

---

*Ken, Software Architect*  
*gert v2 Project*
