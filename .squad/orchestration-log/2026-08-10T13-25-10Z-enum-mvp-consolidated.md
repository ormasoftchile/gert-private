# Orchestration Log — Enum-Constrained Tool and Runbook Outputs MVP Session

**Session:** 2026-08-10 (started 2026-08-10T13:25:10-07:00, completed 2026-08-10T17:45-07:00)
**Requestor:** Cristián Ormazábal Ortega
**Team Root:** `C:\One\OpenSource\gert-private`
**Session Owner:** Barbara (Lead Architect) — coordinating and final gating

---

## Session Arc: Architecture → Spec → Corpus → Implementation → Verification

### Phase 1: Architecture Ruling (Barbara) — 2026-08-10T13:25:10

**Agent:** Barbara (Architect)
**Input:** Cristián's enum-feature request
**Output:** `.squad/decisions/inbox/barbara-enum-constraint-mvp-architecture-ruling.md` (AR-ENUM-1..15, C1/C2/C3)
**Result:** ✓ RATIFIED — binding scope ruling with three corrections

- Four declaration sites (S1/S2/S3/S4): tool action `args`/`outputs`, runbook `inputs`/`outputs`
- String-only constraint
- ENUM-001..009 error codes; ENUM-W001 warning (case-only-distinct)
- Parse-time checks (declarations, defaults); runtime checks (bindings)
- Asymmetric Unicode normalization (declared must be NFC, candidates NFC'd before comparison)
- Corrections: C1 (forbidden on `type: secret`), C2 (mock enum is conformance-only), C3 (no new schema)

**Downstream:** Edith, Tess, Don

---

### Phase 2: Specification Analysis (Edith) — 2026-08-10T13:25:00

**Agent:** Edith (Spec Editor)
**Input:** AR-ENUM-1..15 architecture ruling
**Output:** `.squad/decisions/inbox/edith-string-enum-args-io.md` (four questions for Barbara)
**Status:** OPEN — waiting for Barbara's Q1–Q4 approval before authoring normative sections
**Questions:** Q1 (parse-time + runtime checking), Q2 (Output schema vs prose), Q3 (error codes: SEM-0xx + PKG-030), Q4 (fix pattern/example fields same PR)

**Note:** No artifacts changed; analysis only. Edith correctly deferred pending architecture approval.

**Blocked:** Awaiting Barbara's Q1–Q4 approval; owns parallel prose fix in `03-schema-vnext.tex` §2a post-approval.

---

### Phase 3: Conformance Corpus (Tess) — 2026-08-10T13:25:00

**Agent:** Tess (Conformance Tester)
**Input:** AR-ENUM-1..15; corpus authoring per conformance mandate
**Output:** 
  - `design/gert/conformance/tv-enum.yaml` (58 vectors, frozen)
  - `.squad/decisions/inbox/tess-enum-corpus-notes.md` (schema gap finding)
  - `.squad/decisions/inbox/tess-enum-decl-006-correction.md` (YAML 1.2 fact check)
**Result:** ✓ COMPLETE

- 58-vector corpus finalized; one schema gap found (Output lacks `default` property, documented as untestable-in-corpus)
- TV-ENUM-DECL-006 corrected: bare `yes`/`no` resolve to strings under YAML 1.2 core schema, not booleans (1.1 assumption was wrong)
  - Fixture: `enum: [yes, no]` → `enum: [true, false]` (canonical bool-resolving scalars in 1.2)
  - Vector intent preserved (YAML-1.2-core-schema keyword collision detection); count stays 58
- All AR-ENUM-1..15 rules mapped cleanly; no invented fields
- Owns four further corpus amendments (UNICODE-005/PLAN-005/RUNTIME-004/PLAN-003) diagnosed during Ken's R2 harness

**Blocked on:** Ken completing R2 harness (needed to identify which four vectors need amendment)

---

### Phase 4: Initial Implementation (Don) — 2026-08-10 onwards (C:\One\OpenSource\gert repo)

**Agent:** Don (Backend Developer)
**Input:** AR-ENUM-1..15, Tess's 58-vector corpus, gate criteria R1–R5
**Output:** `.squad/decisions/inbox/don-enum-mvp-implementation-report.md` (runtime implementation + 2 findings)
**Status:** ✓ REPORTED; SUPERSEDED by Ken's independent revision
**Implementation scope:** AR-ENUM-1..15 end-to-end in `gert` (runtime repo)
  - ENUM-001..009 error codes
  - ENUM-W001 warning (case-only-distinct)
  - PKG-013 extended for substitution enum-set equality
  - C1 redaction (enum member lists redacted on sensitive declarations)
  - ValidatedPlan metadata carriage

**Findings escalated to Barbara:**
1. TV-ENUM-DECL-006 conflicts with this repo's YAML 1.2 resolver (not Don's error; vector/library mismatch) — **Tess handled** via corrected fixture
2. No root-runbook output-materialization path in engine (S4 site unreachable) — **Ticketed T-ENUM-ROOT-OUTPUTS** (pre-existing, out of MVP scope)

**Validation:** `go build ./...` clean, `go test ./...` (61 packages pass)

**Blocker for Barbara's gate:** Five genuine runtime defects (not caught by hand-written tests) hide enum enforcement in large vector families — discovered by Ken's R2 harness

---

### Phase 5: Implementation Gate Review 1 (Barbara) — 2026-08-10

**Agent:** Barbara (Architect, gate)
**Input:** Don's complete working tree (C:\One\OpenSource\gert), AR-ENUM-1..15, Tess's corpus, R1–R5 spec
**Output:** `.squad/decisions/inbox/barbara-enum-mvp-implementation-gate.md` (R1–R5 blockers)
**Result:** ✗ REJECTED

**Five blockers identified (R1–R5):**
1. **R1 — ENUM-008 caller-binding:** Missing `--var` and RPC input paths; not all live sites covered
2. **R2 — Conformance harness:** No faithful harness; design-only vectors not mechanized; no real CLI execution
3. **R3 — Enum metadata:** Not carried in `ValidatedPlan.EnumConstraints`; missing trace payload
4. **R4 — ENUM-W001 warning:** Not surfaced to end-user; missing stderr output
5. **R5 — Replay validation:** Replay path does not validate enum membership; generic registry used

**Disposition:** All five are genuine, non-negotiable blockers. **Revision owner: Ken (Backend Developer, independent).** Don and Ken locked out per gate-rejection protocol.

---

### Phase 6: Independent Revision (Ken) — 2026-08-10 (C:\One\OpenSource\gert repo)

**Agent:** Ken (Backend Developer, independent reviser)
**Input:** 
  - Barbara's R1–R5 blocker spec
  - AR-ENUM-1..15 binding architecture
  - Tess's 58-vector frozen corpus (with TV-ENUM-DECL-006 corrected)
  - Don's working tree (read for context, not reused)
**Output:** `.squad/decisions/inbox/ken-enum-mvp-implementation-revision.md` (R1–R5 resolution + bugs found + 58-vector harness)
**Result:** ✓ COMPLETE

**R1–R5 resolution matrix:**

| Blocker | Resolution |
|---|---|
| R1 — ENUM-008 caller-binding | Wired `schema.CheckCallerInputBindings` into `cmd/gert/run.go` entry point and all RPC/API paths; `internal/executor/tool.go` CheckArgEnums on materialized tool args (incl. `--var`-sourced). Regression: `enum_r1_r4_regression_test.go`. ✓ |
| R2 — Conformance harness | Built faithful R2 harness: `internal/conformance/enum_harness.go` + `enum_vector.go` + `enum_conformance_test.go` build real `gert` CLI, materialize each of 58 vectors into isolated workspace, run end-to-end (or drive `pkgcatalog.Build` for catalog vectors). Final result: **48 pass, 10 skip (named), 0 fail**. ✓ |
| R3 — Enum metadata | `internal/planner/enumplan.go` populates `ValidatedPlan.EnumConstraints`; `internal/engine/engine.go` carries once in `plan.validated` trace event, declared order, C1-safe redaction. Regression: `enum_trace_test.go`. ✓ |
| R4 — ENUM-W001 warning | `cmd/gert/run.go` surfaces ENUM-W001 to stderr without aborting run. Regression: `enum_r1_r4_regression_test.go`. ✓ |
| R5 — Replay validation | `internal/executor.CheckArgEnums` exported; `internal/replay.ReplayExecutor.WithEnumChecks` + `ReplayFromTrace` wiring apply identical check at replay boundary. Regression: `enum_r5_test.go`. ✓ |

**Ten genuine runtime bugs discovered (not in charter, revealed by faithful R2 harness):**
1. GCP output-value resolution (executeSubstitution not resolving GCP paths in output.value)
2. Dropped `Enum` field in catalog/toolRefs conversion (highest-impact fix; affected 50%+ of corpus)
3. Missing S2 default check (AR-ENUM-6 tool-output-default case never implemented)
4. Capture-after-failure masking (failed step captures attempted anyway, hiding real errors)
5. GIS-interpolated defaults falsely rejected at plan time (ENUM-006 stringified `"${count}"` literally)
6. Missing `imports:` alias resolution for `include.runbook` (alias-by-name includes failed)
7. Schema/struct drift on `expand:` property (Go field existed, JSON schema didn't declare it)
8. Stdout/stderr split in harness (plan-time errors go stderr, runtime failures go stdout)
9. Temp build directory pollution (enumharness-bin-* dirs in working tree)
10. Validation-ordering fix (B1 substitution checks masked planner.Plan's ENUM-006/007 findings)

**58-vector execution report:**
- 48 vectors: ✓ PASS
- 10 vectors: ⊘ SKIP (named reasons)
  - 5 pre-existing ticketed gaps (T-ENUM-FROM-SOURCING ×3, T-ENUM-ROOT-OUTPUTS ×2)
  - 5 corpus/methodology defects (UNICODE-005/PLAN-003/PLAN-005/RUNTIME-004/RUNTIME-007)
- 0 vectors: ✗ FAIL
- 0 vectors: silent drop

All blockers substantively resolved. No architecture reopened. Dirty tree preserved; protected files byte-identical.

---

### Phase 7: Final Implementation Gate (Barbara) — 2026-08-10T17:45-07:00

**Agent:** Barbara (Architect, final gate)
**Input:** 
  - Ken's independent revision (uncommitted working tree, `C:\One\OpenSource\gert`)
  - R1–R5 verification scope
  - AR-ENUM-1..15, Tess's corrected corpus, Edith's AR-ENUM-3(3) prose fix
**Output:** `.squad/decisions/inbox/barbara-enum-mvp-final-gate-approval.md` (R1–R5 verified + verdict)
**Result:** ✓ APPROVED — production ready

**Verification method (not report-based):**
- Read current runtime diff (39 modified, 43 new paths)
- Built: `go build ./...` (clean)
- Tested: `go test ./...` (61 packages, 0 FAIL)
- Ran R2 harness independently: 58 vectors, **48 pass, 10 skip, 0 fail** (reproduced)
- Executed four live CLI probes against scratch runbooks (deleted after)
- No product artifact modified in either repository; `git status` byte-identical before/after

**R1–R5 final dispositions:**
- R1 ✓ ENUM-008 on every caller-binding path: proven by execution (CLI probe passed, rejected non-member)
- R2 ✓ Faithful 58-vector conformance harness: 48 passed, 10 audited skips, 0 failed, 0 silent drops
- R3 ✓ Enum metadata in ValidatedPlan: emitted once per run, declared order, C1-safe redaction in trace
- R4 ✓ ENUM-W001 warning: reached end-user via stderr, run continued (non-fatal)
- R5 ✓ Replay validation: CheckArgEnums called before consulting recorded fixture

**Ten bugs fixed by Ken:**
- All five runtime defects Ken found and fixed in R2 harness are genuine, ratified in-scope consequences of the R2 requirement
- None are scope creep; all are necessary to run the corpus faithfully
- Collateral fixes (imports alias resolution, expand schema property, validation ordering) required for end-to-end execution

**10 skip reasons adjudicated:**
- 5 pre-existing, separately ticketed (T-ENUM-FROM-SOURCING ×3, T-ENUM-ROOT-OUTPUTS ×2)
- 5 corpus defects (none are evasions of MVP behavior; each verified against AR-ENUM-1..15)

**Remaining limitations (ticketed, none blocking):**
- T-ENUM-ROOT-OUTPUTS (non-substituted root runbook never evaluates own outputs; S4 site unreachable)
- T-ENUM-FROM-SOURCING (Input.From never read; pre-existing)
- T-ENUM-SENSITIVE-DECL (no first-class sensitivity marker; EnumMeta.Redacted is best-effort proxy)
- T-ENUM-REPLAY-WIRE (adapter.go replay branch unreachable today, documented at site)

**VERDICT: APPROVED.** Architecture stands. No further gates required.

---

## Decision Consolidation

### Consolidated Entry in `.squad/decisions.md`

**Single consolidated decision block merges:**
1. Barbara's architecture ruling (AR-ENUM-1..15, C1/C2/C3) — RATIFIED
2. Edith's specification questions — OPEN / IN PROGRESS (awaiting Barbara's Q1–Q4 approval)
3. Tess's corpus work (58 vectors, DECL-006 correction, schema gap finding) — COMPLETE
4. Don's implementation report (findings 1 & 2) — COMPLETE / SUPERSEDED
5. Barbara's gate-1 rejection (R1–R5 blockers) — REJECTED
6. Ken's independent revision (R1–R5 resolution, 10 bugs, 48/58 vectors passing) — COMPLETE
7. Barbara's final gate approval (R1–R5 verified, ready for production) — APPROVED

**Rejection/lockout provenance preserved:**
- Don's implementation work flagged in Ken's context as superseded by independent revision
- Ken's independence from Don's code explicitly stated (read for context, not reused)
- Final gate verified independently (not report-based)

**Inbox archival:** All 13 enum-related inbox files moved to archive with `-archived` suffix:
```
barbara-enum-constraint-mvp-architecture-ruling-archived.md
barbara-enum-mvp-final-gate-approval-archived.md
barbara-enum-mvp-implementation-gate-archived.md
barbara-enum-spec-authoring-gate-review-archived.md
barbara-enum-spec-final-gate-approval-archived.md
barbara-enum-spec-regate-review-archived.md
david-enum-spec-revision-archived.md
don-enum-mvp-implementation-report-archived.md
don-enum-spec-b6-fix-archived.md
edith-string-enum-args-io-archived.md
ken-enum-mvp-implementation-revision-archived.md
tess-enum-corpus-notes-archived.md
tess-enum-decl-006-correction-archived.md
```

---

## Session Outcomes

### Deliverables
- ✓ Architecture ruling (AR-ENUM-1..15, C1/C2/C3) — binding, no reopens
- ✓ Conformance corpus (58 vectors) — frozen, DECL-006 corrected, 5 amendments pending Tess
- ✓ Runtime implementation (all R1–R5) — verified at final gate
- ⊘ Specification prose (Edith) — awaiting Q1–Q4 approval; parallel AR-ENUM-3(3) fix pending

### Outstanding Work
1. **Edith:** Q1–Q4 approval from Barbara, then author normative sections + parallel §2a prose fix
2. **Tess:** Four corpus amendments (UNICODE-005/PLAN-005/RUNTIME-004/PLAN-003) keeping intent/count; re-run harness post-amendment

### Ticketed Non-Goals
- T-ENUM-ROOT-OUTPUTS (root-output materialization)
- T-ENUM-FROM-SOURCING (Input.From sourcing)
- T-ENUM-SENSITIVE-DECL (first-class sensitivity marker)
- T-ENUM-REPLAY-WIRE (adapter replay wiring)

### No Further Gates Required
Ready for Cristián.
