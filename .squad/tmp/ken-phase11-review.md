# Ken — Phase 11 Architectural Review: Evidence & Replay

**Date:** 2026-07-17  
**Reviewer:** Ken (Software Architect)  
**Implementor:** Brian  
**Phase:** 11 — Evidence Collection, Trace Replay, Run Resumption  
**Depends on:** Phase 10 (Adapters, sealed at commit c570a9c)

---

## VERDICT: APPROVED [8/10]

Phase 11 is a strong implementation that faithfully translates my spec into working Go code.
The four capabilities — evidence collection, trace replay, run resumption, and evidence querying —
are all functional, well-tested, and architecturally sound. Brian's deviations from spec are
pragmatic and acceptable for the v2 initial release. Three non-blocking items should be
addressed before Phase 12 begins.

---

## Validation Gate

### Build

```
go build ./...   → 3 FAIL (pre-existing, not Phase 11)
```

The failures are in `pkg/testutil/fake_input_provider.go` (missing `Name()` method on
`FakeInputProvider` after Phase 8 added `Name()` to `InputProvider` interface). This cascades
to `internal/executor` and `internal/parser` via transitive import. **Not a Phase 11 issue.**

All Phase 11 packages compile cleanly:
- `pkg/evidence` ✅
- `pkg/trace` ✅
- `internal/evidence` ✅
- `internal/replay` ✅
- `internal/resume` ✅
- `internal/runstore` ✅
- `internal/trace` ✅
- `internal/engine` ✅
- `internal/adapter` ✅
- `internal/serve` ✅
- `cmd/gert` ✅

### Tests

```
go test ./pkg/evidence/... ./internal/evidence/... ./internal/replay/... \
       ./internal/resume/... ./internal/trace/... ./internal/serve/... \
       ./internal/engine/... ./internal/adapter/... ./cmd/gert/... \
       -race -count=1
→ ALL PASS
```

**40 Phase 11 test functions** across 10 test files:

| Package | Tests | Coverage |
|---------|-------|----------|
| `pkg/evidence` | 5 | Model + hashing |
| `internal/evidence` (collector) | 5 | CLI/tool/attachment/checklist/nil |
| `internal/evidence` (hook) | 1 | Integration with engine |
| `internal/evidence` (attachment) | 3 | Copy+hash, dedup, missing file |
| `internal/replay` (engine) | 2 | E2E, scenario override |
| `internal/replay` (scenario) | 2 | Valid + invalid YAML |
| `internal/replay` (executor) | 6 | CLI match/nomatch/allow, tool, manual, registry |
| `internal/resume` (resume) | 3 | Skip completed, checkpoint, already complete |
| `internal/resume` (scanner) | 4 | Checkpoint, orphaned tool, no checkpoint, corrupted |
| `internal/trace` (reader) | 7 | ReadAll, ReadSince, malformed, kind filter, stepID filter, empty, cancelled |
| `internal/serve` (rpc) | 2 | run.evidence, run.resume |

**Note:** Spec predicted ~42 tests; actual is 40. Two missing are runstore tests (see
Non-Blocking Item #1 below). All claimed functionality is tested.

---

## Five-Axis Review

### 1. Correctness — Does it match the spec? ✅ (9/10)

**Fully compliant areas:**

- **pkg/evidence (leaf package):** All 4 EvidenceKind constants present (text, checklist,
  attachment, snapshot). `EvidenceRecord` struct matches spec exactly — 8 fields with correct
  JSON tags and `omitempty` on optional fields. `HashFile`/`HashBytes` use SHA256 correctly.
  Zero imports from `internal/*` — confirmed leaf package. ✅

- **pkg/trace (reader.go, filter.go):** `TraceReader` interface has all 3 spec methods
  (`ReadAll`, `ReadSince`, `ReadFiltered`). `TraceFilter` struct has 4 fields (Kinds, StepID,
  AfterSeq, BeforeSeq) matching spec. `Matches()` method correctly delegates StepID filtering
  to implementation (requires payload inspection). ✅

- **internal/trace/jsonl_reader.go:** Implements `TraceReader` correctly. Malformed lines
  silently skipped (crash safety contract). Buffer sized at 1MB max line. Context cancellation
  respected per-line. StepID filtering done via payload unmarshal. ✅

- **internal/evidence/collector.go:** Extracts all 4 evidence kinds from `StepResult.Output`.
  Handles both `[]any` and `[]string` attachment formats (defensive). Checklist skips empty
  items. ✅

- **internal/evidence/attachment.go:** SHA256 content-addressed dedup works correctly.
  `filepath.Join(s.dir, sha+ext)` — destination controlled, no path traversal risk.
  Write sequence: `Create → Copy → Sync → Close` with proper error handling at each stage. ✅

- **internal/replay/executor.go:** `ReplayExecutorRegistry` correctly mirrors Phase 10's
  `DryRunExecutorRegistry` pattern. `Lookup()` wraps via `NewReplayExecutor(kind, scenario)`.
  CLI/tool/manual dispatch with first-match-wins scenario matching. ✅

- **internal/replay/engine.go:** `ReplayFromTrace()` follows spec: load trace → extract
  runbook path → parse → plan → build/load scenario → wrap executors → start engine with
  `RunModeReplay`. Evidence pre-feeding via `replayEvidenceHook` with fallback to original
  hook. ✅

- **internal/resume/scanner.go:** Correctly identifies checkpoint events, completed step IDs,
  orphaned tool calls (invoked without completed), and maximum sequence number. All payload
  extraction handles malformed JSON safely. ✅

- **Engine integration:** Evidence hook call site is correctly placed — after executor, after
  redaction, before trace event emission (engine.go:464-472). Only fires on
  `StepStatusCompleted`. Records attached to `result.Evidence`. ✅

- **RPC methods:** `run.evidence` and `run.resume` added to dispatch switch per spec.
  `run.evidence` reads filtered trace, extracts evidence from `step/completed` events.
  `run.resume` calls `Engine.Resume()`, registers in ActiveRuns, pumps events. ✅

- **CLI:** `--resume` flag wired correctly in `cmd/gert/run.go`. Calls `eng.Resume()` when
  set, falls back to `eng.Start()` otherwise. ✅

**Accepted deviations:**

1. **DefaultCollector caches AttachmentStore per runDir** (spec has one store per collector).
   Brian's deviation is sound: the collector is instantiated once in `wire.go` via
   `NewDefaultCollector()` and shared across the engine lifecycle. Per-runDir caching
   correctly handles the case where `runDir` varies per run. The cache is mutex-protected.
   **Acceptable.** The cache grows linearly with unique runDirs seen by this process —
   bounded by single-run-per-process constraint (D-10-05).

2. **Resume uses `CurrentStepIndex` (flat index) instead of `TreePath`** (spec §8.2 shows
   TreePath as authoritative). Brian's deviation is pragmatic: the engine's execution loop
   already uses `CurrentStepIndex` for step advancement. `TreePath` is a v2.1 concern when
   nested execution (invoke-in-progress resume) ships. For v2.0 with flat sequential plans,
   `findStepIndex(plan, state.CurrentStep)` is correct. **Acceptable for v2.0.**

3. **Replay emits `run/replayed` instead of `run/completed`** (engine.go:711). This is a
   **good** deviation — it distinguishes replay runs from real runs in the trace, enabling
   downstream tooling to filter. Spec didn't define this event but it's additive and
   informative. **Approved — add to event catalogue in §06.**

**Missing from spec but not blocking:**

- Orphaned tool call handling (spec §8.4): Brian's scanner *detects* orphaned calls
  (`OrphanedToolCalls` in `ResumeContext`) but the resume path doesn't act on them yet
  (no idempotency check, no forced-fail of non-idempotent tools). This is logged as a
  known open item. **Acceptable for v2.0 — document in release notes.**

- PID lock protocol (spec §8.5): Not implemented. `DirRunStore` has no inter-process
  locking. **Acceptable for v2.0** given single-run-per-process constraint. Must be
  addressed before concurrent-runs in v2.1.

### 2. Readability — Clear naming, consistent patterns? ✅ (8/10)

- **Package naming:** `pkg/evidence`, `internal/evidence`, `internal/replay`,
  `internal/resume`, `internal/runstore` — all clear and self-documenting.

- **Type naming:** `EvidenceRecord`, `EvidenceKind`, `EvidenceSet`, `DefaultCollector`,
  `AttachmentStore`, `ReplayExecutor`, `ReplayExecutorRegistry`, `TraceScanner`,
  `ResumeContext`, `DirRunStore` — all follow Go conventions.

- **Import aliases:** Consistent use of `evidencepkg` for `pkg/evidence` and `tracepkg`
  for `pkg/trace` across all internal packages. ✅

- **Code style:** Clean, minimal comments where needed. Defensive nil checks at function
  entry. Type assertions always use two-value form (`v, ok := x.(T)`). ✅

- **Nit:** `_ = ctx` and `_ = step` in `DefaultCollector.Collect()` (collector.go:44-45)
  are explicit unused-param markers — acceptable but consider using them in future (e.g.,
  step.Kind for kind-specific collection).

- **Nit:** `_ = state` in `ScanTrace()` (scanner.go:52) — the `state` parameter is passed
  but unused. It was designed for future use (validating state.CurrentStep against trace).
  Acceptable but document the intent.

### 3. Architecture — Clean boundaries, correct patterns? ✅ (9/10)

**Package dependency graph (verified, no cycles):**

```
pkg/evidence           → stdlib only (leaf ✅)
pkg/trace              → stdlib only (leaf ✅)
pkg/engine             → pkg/evidence (safe: leaf import)
internal/trace         → pkg/trace
internal/evidence      → pkg/engine, pkg/evidence
internal/replay        → internal/engine, internal/trace, pkg/*
internal/resume        → internal/trace, pkg/engine, pkg/trace
internal/runstore      → internal/trace, pkg/engine, pkg/trace
internal/serve         → all above (adapter layer)
internal/adapter       → all above (wire layer)
cmd/gert               → internal/adapter (entry point)
```

**No import cycles.** Dependency direction is strictly inward toward `pkg/*` leaf packages.

**Pattern consistency:**

- `ReplayExecutorRegistry` ↔ `DryRunExecutorRegistry` (Phase 10): Same wrapping pattern.
  Both implement `ExecutorRegistry`, both check `inner.Lookup()` before wrapping. ✅

- `EvidenceHook` as optional post-step callback: Clean. Nil-safe at call site
  (engine.go:465 checks `!= nil`). Does not modify executor implementations. ✅

- `DirRunStore` follows the adapter pattern: wraps filesystem I/O behind an interface.
  Atomic writes via `writeFileAtomic()` (temp → fsync → rename). ✅

- `ReplayEngine` reuses the real engine pipeline (parse → plan → execute) with swapped
  executors. This is architecturally superior to a separate replay loop — governance,
  conditions, and events all behave identically. ✅

**Architecture concern (non-blocking):**

The `resume.ScanTrace()` function uses a type assertion to discover `TracePath()` on the
store (scanner.go:56-58):
```go
pathProvider, ok := store.(interface{ TracePath(string) string })
```
This is a reasonable duck-typing approach but creates an implicit contract. The store
interface should eventually expose `TracePath()` formally. **Not blocking.**

### 4. Security — Safe file handling, no path traversal? ✅ (8/10)

- **Attachment storage:** Destination path is `filepath.Join(s.dir, sha+ext)` where `sha`
  is a hex-encoded SHA256 digest and `ext` comes from `filepath.Ext(sourcePath)`. The
  digest is safe (hex chars only). The extension is safe because `filepath.Ext()` returns
  at most one dot-prefixed segment. `filepath.Join()` canonicalizes. **No path traversal
  risk.** ✅

- **HashFile:** Accepts arbitrary path. This is fine — the caller (DefaultCollector) only
  passes paths from `StepResult.Output["attachments"]`, which are set by executors. An
  executor could theoretically place `/etc/shadow` in the output, but that's an executor
  trust issue, not an evidence bug. Executors are already trusted code. ✅

- **JSONL reader:** Malformed lines silently skipped (jsonl_reader.go:69-71). Buffer capped
  at 1MB (line 48). Context cancellation checked per-line (line 51). **Crash-safe against
  corrupted trace files.** ✅

- **Scenario loader:** Reads YAML via `os.ReadFile()` — no path validation, but scenario
  files are user-specified via `--scenario` flag (trusted input). ✅

- **Attachment write pattern:** `os.Create → io.Copy → Sync → Close`. No temp-file-then-
  rename pattern. If `io.Copy` fails mid-stream, a partial file remains at `destPath`.
  On re-run, the `os.Stat()` check (attachment.go dedup) would find the partial file and
  skip re-copy — potentially serving corrupt content.

  **Mitigation:** The SHA256 is computed from the *source* file before copy begins. A
  consumer verifying the attachment hash against the `EvidenceRecord.SHA256` field would
  detect corruption. This is acceptable for v2.0 but a temp→rename pattern would be
  more robust.

  **Consider:** Using atomic write pattern for attachments (temp → sync → rename). Low
  priority — the dedup check path returns the correct hash regardless, and consumers
  SHOULD verify hashes.

### 5. Performance — Caching, bounded memory? ✅ (7.5/10)

- **DefaultCollector cache:** `map[string]*AttachmentStore` keyed by `runDir`, protected
  by `sync.Mutex`. Given single-run-per-process (D-10-05), this cache holds exactly 1
  entry during normal operation. For `gert serve` with sequential runs, entries accumulate
  but the store is lightweight (just a `dir` string). **Acceptable for v2.0.** If
  concurrent-runs ship in v2.1, add cache eviction or scope collector per-run.

- **JSONL reader:** Reads entire file into memory (`ReadAll`). For large traces (100K+
  events), this could be significant. The 1MB line buffer is good. For v2.0 with typical
  runs of 10-100 steps, memory usage is negligible. **Acceptable.**

- **TraceScanner:** Single pass over all events to build `ResumeContext`. O(n) time, O(k)
  memory where k = completed steps + orphaned tool calls. ✅

- **ReplayEngine `buildScenarioFromTrace()`:** Iterates events once, builds scenario
  fixtures. O(n) time, O(m) memory where m = number of steps. ✅

- **DirRunStore writer cache:** `map[string]*JSONLWriter` — same pattern as collector
  cache. One writer per active run. Bounded by process lifetime. ✅

---

## Deviation Assessment

| # | Deviation | Spec Reference | Verdict |
|---|-----------|---------------|---------|
| D1 | DefaultCollector caches AttachmentStore per runDir | §4.3 (single store) | ✅ Accepted — bounded by single-run-per-process |
| D2 | Resume uses CurrentStepIndex, not TreePath | §8.2 (TreePath authoritative) | ✅ Accepted for v2.0 — flat plans only |
| D3 | Replay emits run/replayed event | Not in spec | ✅ Approved — good addition, add to §06 |
| D4 | Orphaned tool call detection without action | §8.4 (mark failed/re-execute) | ⚠️ Accepted — detection is implemented, action deferred |
| D5 | No PID lock protocol | §8.5 | ⚠️ Accepted — single-run-per-process covers v2.0 |

---

## Non-Blocking Items

### Nit 1: RunStore has zero tests

`internal/runstore/dir_store.go` is 225 lines of non-trivial code (atomic writes, snapshot
parsing, writer caching) with **no test file**. This is the only Phase 11 package without
tests. It works (verified via integration through resume and engine tests) but should have
dedicated unit tests covering:

- `SaveState → LoadState` round-trip
- Latest snapshot selection (descending index)
- `.tmp` file skipping in `LoadState`
- `WriteTrace` event serialization
- `writeFileAtomic` partial-write recovery
- Empty/missing snapshot directory

**Assign to:** Brian. Estimated: 2-3 hours. Not blocking merge but should be done before
Phase 12.

### Nit 2: Attachment write is not atomic

`AttachmentStore.Store()` writes directly to the final path. If the process crashes during
`io.Copy`, a partial file persists. The dedup check (`os.Stat`) would then skip re-copy,
potentially serving truncated content.

**Recommendation:** Use temp → sync → rename pattern:
```go
tmp := destPath + ".tmp"
// write to tmp, sync, close
os.Rename(tmp, destPath)
```

This matches the pattern already used in `DirRunStore.writeFileAtomic()`.

**Severity:** Low — consumers SHOULD verify SHA256 from EvidenceRecord. But defense-in-depth
is worth the 3 extra lines.

### Nit 3: Silent attachment errors

`DefaultCollector.Collect()` silently swallows attachment storage errors (collector.go:86-87):
```go
if err != nil {
    continue  // best-effort: log but don't fail the step
}
```

The comment says "log" but there's no logging. Consider emitting a trace event or logging
at warn level so operators notice systematic failures (e.g., disk full, permissions).

### Consider: Add `run/replayed` to event catalogue

Brian's `run/replayed` event (engine.go:711) is a good idea. It should be formally added
to §06's event catalogue and §12's event kind list. This enables downstream tooling to
distinguish replay runs from real runs in the trace.

### FYI: `replay_auto` attestation string

`ReplayExecutor.executeManual()` uses `"replay_auto"` as a fallback attestation for manual
steps without scenario evidence (executor.go:144). This magic string is not defined in any
constant. Consider adding it to `pkg/engine/constants.go` for discoverability.

### FYI: Pre-existing build failures

3 packages fail to build due to `FakeInputProvider` missing the `Name()` method added to
`InputProvider` in Phase 8. This is a known pre-existing issue that should be fixed
(1-line addition: `func (f *FakeInputProvider) Name() string { return "fake" }`). Not
Phase 11's responsibility.

---

## Test Quality Assessment

**Strengths:**
- Malformed JSONL handling tested (`TestScanTrace_CorruptedLineSkipped`,
  `TestJSONLReader_SkipMalformed`)
- Orphaned tool call detection tested (`TestScanTrace_OrphanedToolCall`)
- Context cancellation tested (`TestJSONLReader_CancelledContext`)
- SHA256 dedup tested (`TestAttachmentStore_Dedup`)
- Full replay E2E tested (`TestReplay_EndToEnd` — verifies real executor NOT called)
- Resume skip-completed tested (`TestResume_SkipsCompletedSteps`)
- All tests pass with `-race`

**Gaps (non-blocking):**
- No RunStore tests (Nit #1 above)
- No concurrent attachment access test
- No test for attachment partial-write recovery
- No test for replay with non-zero exit codes

---

## Summary Scorecard

| Axis | Score | Notes |
|------|-------|-------|
| Correctness | 9/10 | Matches spec with accepted deviations |
| Readability | 8/10 | Clean, consistent, minor nits |
| Architecture | 9/10 | Clean boundaries, correct patterns, no cycles |
| Security | 8/10 | Safe file handling, crash-resilient reader |
| Performance | 7.5/10 | Bounded for v2.0, needs attention for v2.1 concurrent runs |
| **Overall** | **8/10** | |

---

## Final Verdict

```
VERDICT: APPROVED [8/10]
```

**Non-blocking items before Phase 12:**
1. Add RunStore unit tests (Brian, 2-3h)
2. Consider atomic attachment writes (Brian, 30min)
3. Add warn logging for attachment errors (Brian, 15min)

**Documentation updates:**
- Add `run/replayed` to §06 event catalogue
- Document orphaned tool call handling as v2.1 item in release notes
- Document PID lock protocol as v2.1 item

Brian: solid work. The replay engine architecture — reusing the real pipeline with swapped
executors — is exactly right. Evidence hook placement (post-redaction, pre-trace) is correct.
The deviations are pragmatic. Ship it.
