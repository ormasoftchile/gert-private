# Phase 1 Slices 3–7 Gate Review

**Date:** 2026-08-17  
**Reviewer:** Barbara (Lead / Architect)  
**Verdict:** APPROVED-WITH-CONDITIONS  
**Scope:** Slices 3 (profile schema), 4 (ProfileApprovalGate + attendance), 5 (Tier 0 preflight), 7 (fixture migration + governance vectors), classification validation (GOV-014)

---

## Verdict: APPROVED-WITH-CONDITIONS

Two conditions, both non-blocking for merge but required before we report to the counterparty:

1. **[REQUIRED BEFORE REPORTING]** GOV-009/GOV-010 vectors do not actually catch silent routing coercion (see §A below). Add one behavioral test (not conformance vector) that exercises the ProfileEvaluator with `ToolApprovalTriState=&false + ToolClassification="destructive"` and asserts the classification field on StepInfo is unchanged after evaluation. This pins the "never assign or coerce" guarantee at the behavioral level. **Owner: Ken** (he owns the ProfileEvaluator; Don is locked out as original Classification author).

2. **[REQUIRED BEFORE PHASE 2]** David's `gert plan` command must emit the `config/no-binding-for-profile` message format we committed to — including the "note: this runbook runs correctly in profiles: X, Y" line. That line is the single highest-value operator output in the preflight design. It is computable from plan.Tools + discovered profile files. Verify it is present when the `gert plan` slice ships. **Owner: David.**

---

## Counterparty Validation Points — Status

| Point | Status | Evidence |
|-------|--------|---------|
| 1. GovernanceEvaluator wired in production | **DEMONSTRATED** (from Slice 2, reconfirmed: wire.go:161, run.go ×2, sub-engine ×1, per-run in Start/Resume) |
| 2. Policy composes runbook + per-tool | **DEMONSTRATED** (BuildEvaluator from GovernanceSource + ToolApprovalTriState on StepInfo + OR in evaluator) |
| 3. Approval applies to all transport paths | **DEMONSTRATED** (pre-dispatch chokepoint in executeStep, before executor) |
| 4. Approval never changes classification/retry | **DEMONSTRATED structurally** (different types, different fields, no derivation code) — but see §A for the test gap |

---

## §A. GOV-009/GOV-010 Weakness (Condition 1)

These vectors exercise `requires-approval: false + classification: destructive` (GOV-009) and `+ mutating` (GOV-010). They prove the COMBINATION is schema-valid and the runbook executes successfully. Their note claims "This vector would FAIL if anyone adds code that derives classification from requires-approval."

**That claim is only partially true.** The vectors would catch a REJECTION (code that errors on the combination). They would NOT catch a silent routing coercion inside the ProfileEvaluator (e.g., `if tristate == &false { classification = "read-only" }`), because:
- The conformance harness runs without a profile (NoOpApprovalGate)
- Routing changes in the ProfileEvaluator are invisible without a profile
- The runbook would still complete successfully

The real protection today is structural: `ToolClassification` is populated from `action.Classification` in engine.go:536, never derived from RequiresApproval. The ProfileEvaluator's legacy opt-out path (`if !*step.ToolApprovalTriState { result.RequiresApproval = false; return }`) only suppresses approval — it never touches classification. This is sound, but a regression test at the evaluator level would make it fail-safe.

---

## §B. Nil Profile Preservation

Verified. Three layers all nil-guard:
- `profileEvaluator.Evaluate()`: `if pe.profile == nil { return result, nil }` — transparent
- `checkToolEnvironmentPreflight()`: `if profile == nil { return nil }` — no checks
- `checkAttendancePreflight()`: `if profile == nil { return nil }` — no checks

No `--profile` flag → `runtimeProfile == nil` → all new code paths are inert. Existing behavior unchanged.

---

## §C. Unspecified Matrix — Verified Correct

The ProfileEvaluator's `default` branch (covering nil classification AND any unrecognized value) implements:
- Test context → auto-approve ✅
- Attended → gate fires (NOT warn-and-proceed) ✅
- Unattended + legacy_allow → suppress prompt only ✅
- Unattended + prompt (default) → deny ✅

The comment explicitly states the governing principle: "absence — or garbage — must never grant additional execution rights." Commit 31c0aa0 adds defense-in-depth tests for unrecognized values.

---

## §D. Legacy Unspecified Policy — Verified Correct

`legacy_unspecified_policy: allow` suppresses the approval prompt ONLY. The evaluator sets `result.RequiresApproval = false` and returns. It does NOT set `result.Allowed = true` with a different classification, does NOT modify ToolClassification, does NOT affect any retry or late-result field (none exist on EvaluationResult). The comment in the code says: "Suppresses the migration prompt only. Does NOT relax retry, idempotency, or late-result behavior."

---

## §E. CI Hang Hazard

The hazard is genuinely closed for profile-using paths. `attendedFromProfile()` extracts attendance → `WireOptions.Attended` → `buildApprovalGate()` selects NoOpApprovalGate when `Attended == &false`. A CI profile with `attendance: unattended` will never block on stdin.

Residual risk: profileless runs in CI where a tool declares `requires-approval: true`. This is the pre-existing `TTYOutput: true` hardcode bug, not introduced by this work. The Tier 0 attendance check (PLAN-011) catches the case where someone uses `--profile ci` with `attendance: attended` — it fails before execution. The profileless case remains a known limitation pending isatty() / `--unattended` flag (documented in preflight.go comments).

---

## §F. Test-Context Transport Rules — Verified Correct

- `native` → always allowed ✅
- `mcp` → blocked unless `transport.allow_subprocess_in_test: true` ✅
- `mcp-http` → NEVER allowed, no override ✅
- All error messages describe the subprocess opt-in as "an auditable author assertion, NOT a sandbox" ✅
- preflight.go header comment explicitly states "subprocess inherits the full parent environment (exec.Command + mergeEnv uses os.Environ())" ✅

---

## Ratification Decisions

### 1. Error class: PKG-010 for classification validation — RATIFIED

`PKG-010` already covers "package export references a file that fails any validation check." Classification validation is such a check. The error message includes the full actionable detail (invalid value, allowed values list). Inventing a new `TOOL-PARSE` class would fragment the error namespace without functional benefit. The conformance vector (GOV-014) correctly uses `PackageExpected` shape with `error.code: PKG-010`.

**Discoverability note:** The nested message ("action 'ping': classification 'invalid-class' is not valid") is clear to an external consumer. The `PKG-010` wrapper says WHERE in the system the error was caught (package catalog loading); the nested message says WHAT failed. This pattern is consistent with other PKG-0xx errors. Acceptable.

### 2. Tess Q1 — TV-CONFORM-GOV-004: recast, do not bypass

**Decision:** The harness MUST NOT gain an approval-bypass mechanism. That would create a test path where approval is not enforced — a safety invariant violation.

**Required action:** Recast TV-CONFORM-GOV-004 to expect the specific error that occurs when approval is required but no interactive channel is available. The harness runs with `TTYOutput: false` → `NoOpApprovalGate` → auto-approves → vector would actually PASS. Wait — re-reading: the harness uses NoOpApprovalGate which auto-approves. So the tool WOULD execute successfully. The SKIP reason says "the CLI harness has no mechanism to supply approval" but NoOp auto-approves...

**Correction:** If the harness auto-approves, the vector should just run and PASS (the runbook completes). Tess should investigate whether the vector actually fails or was pre-emptively skipped. If it passes under NoOp, un-skip it. If the ProfileEvaluator denies it (because the test has a profile that doesn't allow the action), adjust the fixture. **Owner: Tess.** Un-skip or recast by end of Phase 1.

### 3. Tess Q3 — CONFORM-CROSSCUT category: DEFER

Current category is fine. Revisit when governance vectors exceed 20. No action now.

### 4. Edith OQ-1 — Profile ID format: DECIDED

`[a-z0-9][a-z0-9-]*`, max 64 characters. Required for shell quoting safety, tab-completion, and filename-stem use (profiles stored as `<id>.yaml`). Leading digit allowed. No uppercase, no spaces, no special characters beyond hyphen. Edith patches §17.1.

### 5. Edith OQ-2 — Per-tool endpoint + auth inheritance: DECIDED

Per-tool overrides inherit profile-level auth by default. If a per-tool override needs distinct auth, add an `auth:` block inside `tools.<tool-id>:` in the profile. Phase 1 implements endpoint-only overrides with inherited auth. Per-tool auth override is Phase 3 scope. Edith patches §17.4 with this note.

### 6. Edith OQ-3 — `gert plan --show-profiles` zero matches: DECIDED

Exit 0 with empty output + a single-line stderr message: "no profile in <search-paths> provides a complete binding for all toolRefs in <runbook>." Empty output is machine-parseable (no false positives for CI scripts). Non-zero exit is wrong here because "zero matches" is a valid, informative answer — not a failure. Edith patches §17.6.

---

## Systemic Process Recommendation

Two instances this session of "checks wired to nothing from the CLI entry point":
1. `AllowedEnvironments` / `RequiresCapabilities` were declared but unenforced (the ghost fields that started this project).
2. David's Tier 0 preflight was implemented in the planner but unreachable from `gert run` until d7b77c1 wired `Profile` into the planner config.

**Recommended convention (adopt for Phase 2+):** Every enforcement function MUST have a companion integration test that exercises it via the CLI entry point (`cmd/gert/run.go` or the equivalent wire path). Unit tests against the planner or evaluator prove correctness; an integration test proves reachability. Name them `TestCLI_<Feature>_Reachable`. If the function is unreachable, the test fails immediately — not after months of silent dormancy.

This is the gap class the whole project exists to eliminate. It recurred inside the project itself. A convention is cheaper than catching it in review.

---

## Summary

All negotiated contract points are satisfied. The implementation is structurally sound and matches the six-round negotiated design. Two conditions required before external reporting (one behavioral test for the coercion invariant, one `gert plan` output format requirement). Six ratification decisions issued. One process recommendation for the team.

---

*Barbara — 2026-08-17*

---

