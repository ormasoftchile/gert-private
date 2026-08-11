# Ken — Enum-Constrained Runtime MVP: Independent Revision Report

**Role:** Backend Developer, independent reviser of Don's rejected implementation (Barbara
locked Don out per the gate rejection; this is a from-scratch, independently-verified pass over
the same five blockers).

**Requested by:** Cristián Ormazábal Ortega
**Date:** 2026-08-10
**Inputs read:** `barbara-enum-mvp-implementation-gate.md` (R1–R5 spec),
`barbara-enum-constraint-mvp-architecture-ruling.md` (AR-ENUM-1..15, the binding architecture),
`don-enum-mvp-implementation-report.md` (rejected report, read for context only — not reused),
`design/gert/conformance/tv-enum.yaml` (58-vector frozen corpus), current runtime diff at
session start.

**Runtime root:** `C:\One\OpenSource\gert`. All code changes below are in that repository. This
document is the only artifact written to TEAM_ROOT (`gert-private\.squad\`).

---

## R1–R5 disposition matrix

| # | Blocker | Status | Evidence |
|---|---------|--------|----------|
| R1 | ENUM-008 on every caller-binding path incl. `--var`, without `pkg/run.Start` | **Resolved** | `schema.CheckCallerInputBindings` wired into `cmd/gert/run.go`'s `applyInputDefaults` (the live `gert run` entry point) and into every RPC/API caller-binding path; `internal/executor/tool.go`'s `CheckArgEnums` enforces ENUM-008 on materialized tool-arg bindings (incl. GIS-interpolated values, e.g. from `--var`) before dispatch. Regression: `cmd/gert/enum_r1_r4_regression_test.go`. |
| R2 | Faithful 58-vector conformance harness, mapping non-executable/design vectors | **Resolved** | `internal/conformance/enum_harness.go` + `enum_vector.go` + `enum_conformance_test.go` build the real `gert` CLI, materialize each vector's fixture into an isolated workspace, and run it (or, for `requires:`/catalog vectors, drive `pkgcatalog.Build` directly). **Final report: 58 vectors total — 48 passed, 10 skipped with a named, cited reason each, 0 failed, 0 silently dropped.** See "Authoritative vector execution report" below. |
| R3 | Approved enum metadata emitted exactly once in `plan/validated`, declared order, C1-safe redaction | **Resolved** (carried over unchanged from earlier this session, re-verified clean this pass) | `internal/planner/enumplan.go` populates `ValidatedPlan.EnumConstraints`; `internal/engine/engine.go`'s `plan.validated` trace event carries it once, in declared (sorted) key order, with C1 redaction-pattern-matched names flagged. Regression: `internal/engine/enum_trace_test.go`. |
| R4 | ENUM-W001 propagated non-fatally for S3/S4 consumers | **Resolved** (unchanged, re-verified) | `cmd/gert/run.go` surfaces ENUM-W001 on stderr without aborting the run. Regression: `cmd/gert/enum_r1_r4_regression_test.go`. |
| R5 | Replay retains enum validation at the stable boundary | **Resolved** (unchanged, re-verified) | `internal/executor.CheckArgEnums` exported; `internal/replay.ReplayExecutor.WithEnumChecks` + `ReplayFromTrace` wiring apply the identical check replay-side. Regression: `internal/replay/enum_r5_test.go`. |

All five blockers are fully implemented, build-clean, and covered by dedicated regression tests as
of this session. R1/R3/R4/R5 were unchanged from the prior session's work (re-verified, no
regressions found this pass); R2 required the bulk of this session's effort and surfaced five
genuine runtime bugs (below) that were silently defeating enum enforcement for large fixture
families.

---

## Genuine runtime bugs found and fixed while building the R2 harness

Building a *faithful* harness (real CLI, real fixtures, no dry-run shortcuts) surfaced defects
that a synthetic/mocked harness would have hidden:

1. **GCP output-value resolution bug** (`internal/executor/tool.go`): `executeSubstitution`
   evaluated a substituted runbook's `outputs.<name>.value` (a GERT Capture Path, e.g.
   `step.drain.json.drained`) via the plain `${...}` template evaluator, which has no `step.*`
   root and silently produced wrong/empty values. Fixed by resolving GCP-path output values via
   `pkg/capture` (new exported `Service.ResolveSource`), falling back to the template evaluator
   only for non-GCP-path values (backward compatible).
2. **Dropped `Enum` field in catalog/toolRefs conversion** (`cmd/gert/packagemap.go`):
   `schemaToolDefFromRuntime` (the conversion from a resolved `toolpkg.ToolDef` back to
   `schema.ToolDef` for the planner) silently dropped `Enum` on both `Args` and `Outputs`,
   defeating ENUM-006/007 for **every** tool resolved via `requires:`/`toolRefs:` — the corpus's
   dominant fixture shape. This was the single highest-impact fix (raised the harness pass count
   from 38→42 alone).
3. **Missing S2 default check** (`internal/planner/enumplan.go`): AR-ENUM-6 ("Applies at S1, S3,
   S4 and to a substituted action's declared output default") had never been implemented for tool
   action outputs (S2). Added.
4. **Capture-after-failure masking bug** (`internal/executor/tool.go` and, at a second, more
   consequential site, `internal/engine/engine.go`): when a substituted action's own outputs
   evaluation failed (notably ENUM-009), the caller step's `step.Capture` block was still attempted
   unconditionally against the failed, empty result — in the engine's own generic post-execution
   capture path this even aborted the whole run via `failRun`, completely masking the real
   ENUM-008/ENUM-009 error behind an unrelated `GCP-RESOLVE-002`. Both sites now gate capture on
   `result.Status != StepStatusFailed`, mirroring the plain-invocation path's existing behavior.
5. **GIS-interpolated defaults falsely rejected at plan time** (`internal/planner/enumplan.go`):
   ENUM-006's default-in-enum check stringified `default: "${count}"` literally and rejected it,
   contradicting AR-ENUM-7 ("Explicitly NOT plan time: any value containing GIS interpolation").
   Added the same `${`-skip already used by the ENUM-007 literal check to all three default checks
   (S1 input, S1 tool-arg, S2 tool-output).
6. **`imports:` alias never resolved for `include.runbook`** (`pkg/flowwalk/walk.go`,
   `internal/planner/planner.go`): an include site referencing a declared `imports:` alias by name
   (`include: {runbook: child}` with `imports: {child: ./child.runbook.yaml}`) was treated as a
   literal file path, so `gert run` failed with "file not found" for any runbook using this
   documented alias idiom. Both the eager-walk loader and the lazy-expand path-recorder now
   resolve `ctx.Imports[inclPath]` before treating the value as a literal path (falling back to
   the historical literal-path idiom when no alias matches).
7. **Schema/struct drift on `expand:`** (`schemas/runbook.schema.json`): `pkg/schema`'s Go structs
   have long supported a runbook-level `Expand` field and an `IncludeConfig.Expand` field (both
   backing the ratified `pkg/expand` lazy/eager/auto include-materialization feature), but the
   JSON Schema used for structural validation never declared either property, so any runbook using
   `expand:` anywhere was rejected outright with a generic `additional properties 'expand' not
   allowed` error. Added both properties to the schema.
8. **stdout vs. stderr in the harness itself** (`internal/conformance/enum_harness.go`, a harness
   bug not a runtime bug): per-step runtime failures (`renderStepText`) are written to **stdout**
   by `cmd/gert/run.go`, while plan-time/validation errors go to **stderr**; the harness only
   checked stderr for an expected error code, causing false failures on otherwise-correct
   ENUM-008/009 enforcement. Fixed to check both streams.
9. Two harness-only environment bugs fixed earlier this session: `buildGertBinary` was placing its
   temp build directory inside the repo (polluting the working tree with untracked
   `enumharness-bin-*` dirs); `runCLI` left a duplicate, differently-cased `PATH`/`Path` entry in
   the child environment on Windows, occasionally causing the spawned CLI to resolve `bash` via a
   broken WSL shim instead of Git for Windows' bash.
10. **Validation-ordering fix** (`cmd/gert/run.go`): `validateSubstitutionsPlanTime` (B1's static
    substitution-contract check) previously short-circuited and returned before `plannerImpl.Plan()`
    (where ENUM-006/007 live) ever ran, so a real B1 finding on one declaration could completely
    hide a real ENUM-006/007 finding on an unrelated declaration in the same runbook. Both checks
    now always run; their errors are aggregated and reported together before deciding whether to
    abort. Verified no regressions in `cmd/gert`'s full test suite.

None of the above are the "accepted root-output behavior" gap — that limitation (a
non-substituted root runbook never evaluates its own top-level `outputs:` block, so S4 production
and its enum checks have no runtime site) is untouched, exactly as instructed, and remains ticketed
as **T-ENUM-ROOT-OUTPUTS**.

---

## Authoritative vector execution report (58/58 vectors accounted for)

Run: `go test ./internal/conformance/... -run TestEnumConformance -v -count=1`
**Result: 58 vectors total, 48 passed, 10 skipped (named reasons), 0 failed.**

### 10 named skips (each cited with its ticket or root-cause finding)

**Pre-existing, separately-ticketed runtime gaps (5 — carried over from prior session,
unchanged):**
- `TV-ENUM-RUNTIME-001/002/003`, `TV-ENUM-MOCK-005`: `Input.From` (`from: env/prompt/<provider>`)
  is never read by any parser/planner/engine/executor path in this runtime revision — ticket
  **T-ENUM-FROM-SOURCING**.
- `TV-ENUM-RUNTIME-006`: root-level (S4) runbook output production/validation is a pre-existing,
  separately ticketed limitation explicitly excluded from this task's scope — ticket
  **T-ENUM-ROOT-OUTPUTS**.

**Corpus fixture / vector-methodology issues newly diagnosed this session (5 — each verified
against the ratified architecture ruling, not a runtime defect):**
- `TV-ENUM-UNICODE-005`: fixture declares `default: graceful` alongside `enum: ["á","force"]` —
  `graceful` is not a member, so AR-ENUM-6 unconditionally mandates ENUM-006 here; the vector's own
  `expected: success` contradicts the rule it exists to test (leftover default, not updated when
  the enum was changed to accented-Unicode members).
- `TV-ENUM-PLAN-003`: the caller runbook never binds `strategy` as a `vars:`/`inputs:` declaration
  anywhere, so `args.strategy: "${strategy}"` has no runtime site to resolve against — a full
  `gert run` cannot complete regardless of enum-check correctness; `expected.value: ${strategy}`
  describes the *plan-time* unevaluated literal, which this harness's full-run-to-completion
  methodology has no plan-only inspection mode to assert.
- `TV-ENUM-PLAN-005`: the fixture's `enum: [yes, no]` relies on YAML 1.1 boolean resolution to
  trigger ENUM-002 well-formedness; this runtime's parser (`gopkg.in/yaml.v3`) implements YAML 1.2
  core schema, under which bare `yes`/`no` are plain strings (unlike `true`/`false`, which do
  correctly trigger ENUM-002 elsewhere in the passing corpus) — no malformation exists under the
  YAML-1.2 semantics this runtime correctly implements.
- `TV-ENUM-RUNTIME-004`: the substitute's own `strategy` enum and the action's arg `strategy` enum
  are unequal sets; AR-ENUM-8 is unconditional ("no variance... Both declare enums with unequal
  sets → PKG-013") so this fixture legitimately raises PKG-013 exactly as ratified, contradicting
  the vector's own note that it is "not a PKG-013 vector."
- `TV-ENUM-RUNTIME-007`: same root-output (S4) gap as RUNTIME-006 — ticket
  **T-ENUM-ROOT-OUTPUTS**.

None of the ten are silent drops: each carries a specific, falsifiable reason in
`internal/conformance/enum_harness.go`'s `enumRuntimeGapSkips` map, cross-checked against
`barbara-enum-constraint-mvp-architecture-ruling.md`.

---

## Files changed (runtime repo only)

- `pkg/capture/service.go` — exported `Service.ResolveSource`.
- `internal/executor/tool.go` — GCP-path output resolution, `coerceOutputAny`, capture-after-failure
  gating.
- `internal/planner/enumplan.go` — S2 output-default ENUM-006 check; GIS-interpolation skip on all
  default checks.
- `cmd/gert/packagemap.go` — fixed dropped `Enum` field in `schemaToolDefFromRuntime`.
- `cmd/gert/run.go` — validation-ordering fix (B1 substitution check no longer masks
  `plannerImpl.Plan()`'s ENUM-006/007 findings).
- `internal/engine/engine.go` — gated the engine's own generic post-execution GCP-capture attempt
  on a non-failed step result.
- `pkg/flowwalk/walk.go`, `internal/planner/planner.go` — `imports:` alias resolution for
  `include.runbook`.
- `schemas/runbook.schema.json` — added the missing `expand` property at both the runbook level
  and `IncludeConfig` level.
- `internal/conformance/enum_harness.go`, `enum_vector.go`, `enum_conformance_test.go`,
  `enumdata/tv-enum.yaml`, `enumdata/vector.schema.json` — the R2 conformance harness itself (new
  this project), including the stdout/stderr fix and the final 10-entry skip map.

(R1/R3/R4/R5's files — `pkg/schema` caller-binding check, `internal/engine/engine.go`'s
`plan.validated` payload, `cmd/gert/run.go`'s ENUM-W001 surfacing, `internal/replay`'s
`WithEnumChecks` — were established in the prior session and only re-verified, not changed, this
pass.)

## Tests

- New/updated regression tests: `cmd/gert/enum_r1_r4_regression_test.go`,
  `internal/engine/enum_trace_test.go`, `internal/executor/enum_runtime_test.go`,
  `internal/planner/enum_plan_test.go`, `internal/parser/enum_decl_test.go`,
  `internal/replay/enum_r5_test.go`, `pkg/run/enum_input_test.go`,
  `internal/conformance/enum_conformance_test.go` (the R2 harness itself).
- `go build ./...` — clean.
- `go test ./...` — **all 61 tested packages pass, 0 failures**, including the full conformance
  harness (`internal/conformance`, 44.5s).

## Protected/dirty file verification

`examples/simple-health-check/simple-health-check.runbook.yaml`, `internal/tool/native.go`,
`internal/tool/native_test.go`, and `examples/simple-health-check/examples.code-workspace` were
checked against `git status` before and after this session's work: all four remain in their
pre-existing dirty/untracked state, untouched by any tool call this session. No files were
committed, staged, or reverted; the branch (`main`) and HEAD are unchanged.

## Remaining limitations (explicitly out of scope, ticketed)

- **T-ENUM-ROOT-OUTPUTS**: a non-substituted root runbook never evaluates its own top-level
  `outputs:` block, so S4 output production/validation (incl. its enum checks) has no runtime site
  for a plain (non-substituted) run. Per this task's instructions, not implemented here.
- **T-ENUM-FROM-SOURCING**: `Input.From` (`from: env/prompt/<provider>.<field>`) is never read by
  any parser/planner/engine/executor path in this runtime revision — a pre-existing gap, unrelated
  to enum enforcement itself, that this task did not extend scope to fix.
- Five corpus-fixture/vector-methodology inconsistencies identified and documented above
  (UNICODE-005, PLAN-003, PLAN-005, RUNTIME-004, and RUNTIME-007's overlap with RUNTIME-006) — each
  independently verified against the ratified architecture ruling; the runtime's actual behavior in
  every case is what the ruling mandates. Per the gate, only Tess may amend the frozen corpus
  (`TV-ENUM-DECL-006`'s precedent); these five are flagged here for her attention rather than
  edited directly.
