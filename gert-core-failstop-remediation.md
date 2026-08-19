# Gert Core — Fail-Stop Defect Remediation

**Status:** Implemented, merged, pushed
**Commit on `main`:** `ba9d6d4` (merge of `squad/failure-stop`)
**Date:** 2026-08-18
**Responds to:** *Gert Core Request — Unhandled Step Failure Must Stop the Run*

---

## 1. Verdict

Accepted as a severity-1 fail-open defect. The report was accurate, and the
reproduction was correct in every particular.

The investigation found the defect was **broader than reported**. The behaviour
you observed at the top level of the flow existed in the same form inside
`include`, `branch`, and `parallel` containers. Had only the reported symptom
been fixed, a nested failure would still have produced a successful terminal
outcome.

---

## 2. Root cause

Two independent findings, both in `internal/engine/engine.go`.

**2.1 — The default was `continue`, by deliberate deferral.**

```go
// Default: continue (backward compatible behavior)
// In v3, this will change to "stop"
onError = "continue"
```

Any step failing without an explicit `on_error` was tolerated. This is the
direct cause of the run you traced: `get_icm_incident` failed, the default
swallowed it, and the flow proceeded into the fallback branch and reached
`icm-tsg-routing-complete`.

**2.2 — `continue_on_fail` was read in exactly one place.**

`grep ContinueOnFail internal/engine/engine.go` returned a single hit. Nested
container execution never consulted it, and propagated any child failure
upward unconditionally. This was inert only because the top-level default
swallowed whatever it propagated.

These two interact, and that interaction is why the earlier attempt failed.
Flipping the default alone converts every previously-tolerated nested failure
into a fatal one — which is precisely why it broke
`TestIterationEvents_CollectHealth` despite an explicit `continue_on_fail: true`.
That test was not collateral damage; it was correctly detecting an incomplete
fix.

---

## 3. The fix

**3.1 — One shared precedence helper.** `resolveOnError()`
(`internal/engine/engine.go`) is the single definition of the rule:

| Precedence | Source |
|---|---|
| 1 | `on_error` |
| 2 | `continue_on_fail` |
| 3 | default — now `stop` |

Deliberately one function. Two copies would drift and silently reintroduce the
defect.

**3.2 — Unhandled failure is terminal.** On resolution `stop`, the run is
atomically marked `RunStatusFailed`, `run/failed` is emitted, the run is marked
done, spans are closed and extensions shut down. The next `Next()` returns
`io.EOF`. No subsequent sibling, branch, display or `end` step starts.

**3.3 — Containers obey the same rule.** A terminal sub-run failure is
signalled with the `ErrSubRunFailed` sentinel (wrapped, `errors.Is`-matchable)
and converted by the `include` and `branch` executors into a **failed
`StepResult`** — not an infrastructure error. The parent engine then applies
`resolveOnError` to the container step exactly as it would to a leaf step.

This last point is the substance of the fix. It preserves your requirement 3 at
container level: `on_error: continue`, `continue_on_fail: true`, and
`on_error: goto:<step>` all work on `include`, `branch`, and `parallel` steps.
Genuine infrastructure faults remain distinguishable from step failures and
still hard-fail via `failRun`.

**3.4 — CLI.** `gert run` now emits its JSON summary before exiting on failure
rather than returning immediately. The non-zero exit is unchanged.

**3.5 — HTTP.** `POST /runs` auto-advance inherits the terminal behaviour, and
is exercised through the real server rather than the internal advance helper.

---

## 4. Requirements mapping

| # | Requirement | Evidence |
|---|---|---|
| 1 | Unhandled failure → terminal failed, `run/failed`, error to caller | `TestFailStop_FirstToolFail_StopsRun` |
| 2 | No later sibling/branch/display/`end` starts | same, asserts no further `step/started` |
| 3 | `on_error: continue` intact | `TestFailStop_ContinueOnFail_Continues`, `TestFailStop_IncludeOnErrorContinue`, `TestFailStop_BranchOnErrorContinue` |
| 3 | `continue_on_fail: true` intact | `TestIterationEvents_CollectHealth` (pre-existing, passes unmodified) |
| 3 | `on_error: goto:<step>` intact | `TestFailStop_IncludeOnErrorGoto` |
| 4 | Parallel branch failure stops later siblings | `TestFailStop_ParallelBranchFail_StopsLaterSiblings`, `TestFailStop_ParallelBranch_ContinueOnFail` |
| 5 | Real HTTP `POST /runs`, not the internal helper | `TestFailStop_HTTP_PostRuns_FailedStepTerminatesRun` |

Additional coverage beyond the request: `TestFailStop_IncludeChildFail_StopsRun`,
`TestFailStop_BranchChildFail_StopsRun`, `TestFailStop_InfraError_HardFails`,
`TestRunSubStepsViaEngine_FailedChild_ReturnsError`,
`TestRunSubStepsViaEngine_ContinueOnFail_ReturnsNil`,
`TestIncludeExecutor_InfraError_Propagates`.

---

## 5. Mutation controls (measured, not reasoned)

Each was applied to the source, the suite re-run, and the failure observed.

| Mutation | Test that failed |
|---|---|
| Default back to `continue` | `TestFailStop_FirstToolFail_StopsRun` — *expected io.EOF, got nil* |
| Nested propagation made unconditional | `TestFailStop_ParallelBranch_ContinueOnFail` |
| Parent ignores failed container | `TestFailStop_ParallelBranchFail_StopsLaterSiblings` |
| Remove sub-run terminal-state check | `TestRunSubStepsViaEngine_FailedChild_ReturnsError` |
| Remove sentinel distinction | `TestFailStop_InfraError_HardFails` |

---

## 6. Rejected attempt

The prior attempt you rejected was not used. It was preserved unmodified for
reference at `analysis/rejected-failstop/` and then discarded from the working
tree; it never entered any commit. Your three stated grounds for rejection were
each independently confirmed during this work.

---

## 7. Verification

All figures below are from a **detached worktree** containing only tracked
files — never from a working tree. The reproducibility gate agreed in Phase 1B
applies here.

At `ba9d6d4` (merged `main`):

```
git status --porcelain   → (empty)
go build ./...           → exit 0
go test ./...            → exit 0
                           63 ok · 23 no test files · 0 FAIL
```

Identical to the pre-change baseline at `d1f4352` — no package lost coverage
and no test was deleted, skipped or weakened to accommodate the change.

No existing test had `continue_on_fail` added to make it pass. Where a
pre-existing test changed behaviour, it was ruled on individually:

| Test | Ruling |
|---|---|
| `TestEngine_Execute_StepFailed_EmitsStepFailed` | Encoded the old default; updated to assert terminal run |
| `TestE2E_ToolStep` | Tool was never resolvable; the failure had been silently swallowed. Now asserts `RunStatusFailed` |
| `TestIncludeExecutor_FailedChildPropagates` | Retained; updated to the sentinel contract, still asserts a failed result |
| `TestRun_Substitution_Enum009_OutputViolation`, `TestEnumConformance` | Unchanged; passed once the CLI emitted its summary before exiting |
| `TestIterationEvents_CollectHealth` | **Unmodified.** Passes as-is — the regression guard you named |

---

## 8. Limitations

- `internal/adapter` had no test coverage before this change; the new
  `failstop_substep_test.go` is its first.
- Retry semantics for read-only idempotent actions on transport loss remain
  future work, as before. `INDETERMINATE` handling for mutating, destructive
  and unspecified actions is unchanged by this fix.
- CI evidence for this commit has still not been observed by us. The figures
  above are local clean-checkout results.
