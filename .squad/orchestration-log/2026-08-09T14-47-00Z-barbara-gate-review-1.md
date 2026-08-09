# Orchestration Log — barbara-gate-review-1 (Architecture Gate #1)

**Timestamp:** 2026-08-09T14:47-07:00  
**Agent:** Barbara (Lead/Architect)  
**Session:** GERT Tool Packages MVP — Gate Review #1 (Edith's specification against AR-TP-1..10)  
**Task Type:** Architecture review (sync, gate decision)

## Session Summary

Comprehensive gate review of Edith's implementation of the ratified GERT Tool Packages architecture ruling (AR-TP-1..10). Fifteen blocking issues identified across wiring, contradiction, and under-specification, primarily in the substitution and path-safety contracts. The architecture direction is sound; defects are locally fixable.

**Verdict:** REJECTED — required revisions before advancement

## Issues Identified

### Blocking (Highest Severity — Semantic Contradictions/Under-specification)

| # | Issue | Classification |
|---|-------|---|
| R1 | Tool action shape self-contradictory (map vs. list, `args/output` vs. `inputs/outputs`) | Semantic contradiction |
| R2 | Per-action `impl:` collides with top-level mobile `impl:` | Naming collision |
| R3 | Substituted-action outputs have no defined capture path | Under-specification |
| R4 | Reference example does not produce its declared output via valid GCP | Specification error |
| R5 | Path containment base directories unstated; reference example appears to escape | Under-specification + example error |

### Blocking (Medium Severity — Wiring Failures)

| # | Issue | Classification |
|---|-------|---|
| R6 | PKG-020/PKG-021 unreachable behind closed schemas | Schema wiring error |
| R7 | `dependencies:` schema contradicts PKG-016 reachability | Schema contradiction |
| R8 | `apiVersion` inconsistency between spec and schema | Spec/schema mismatch |
| R9 | Tier-3 qualification/pinning mechanism schema-incompatible | Schema compatibility error |
| R10 | §05 collision text contradicts uniform rule | Spec contradiction |
| R11 | Replay semantics for substitution undefined | Under-specification |
| R12 | Step-level `tool.version` unreconciled with constraint sites | Spec ambiguity |
| R13 | Lock-file `root` pattern invariant not structurally enforced | Schema enforcement gap |
| R14 | Missing error codes for mutually-exclusive bindings | Error catalog gap |
| R15 | No concrete unchanged-runbook real/mock acceptance scenario | Specification gap |

## Work Completed

- Full normative artifact review against ratified AR-TP-1..10 ruling
- Schema file validation and cross-reference checks
- Fixture example verification (reference tools, r23 corpus entry)
- Error catalog completeness audit
- Spec text coherence analysis

## Recommendations

1. **R1:** Establish one canonical `tool/v1` action shape; reconcile `args/output` with `inputs/outputs` vocabulary
2. **R2:** Disambiguate per-action key (suggest renaming `impl` → `execute` at action level)
3. **R3:** Define `outputs.<name>` capture namespace and wire into GCP grammar
4. **R4:** Rewrite reference substitute to produce declared output via valid GCP paths
5. **R5:** Create explicit "Resolution base per path kind" table; verify no escapes
6. **R6:** Add mandated pre-schema raw-document scan for `toolPackages:` and `alias:`
7. **R7:** Remove schema `maxItems: 0` constraint that blocks PKG-016
8. **R8:** Normalize all `apiVersion` values across examples and schemas
9. **R9:** Make tier-3 addressing schema-compatible (qualified `name` only)
10. **R10:** Split collision paragraph into cross-source (precedence) vs. same-source (hard error) cases
11. **R11:** Define substitution replay behavior explicitly (relates to non-fatal `replay/packageDrift`)
12. **R12:** Reconcile step-level version as deprecated/intersected plan-time constraint
13. **R13:** Tighten lock-root pattern to structurally forbid raw absolute paths
14. **R14:** Assign error codes for unspecified conditions
15. **R15:** Provide concrete unchanged-runbook scenario with mode variations

## Decision Status

**Status:** Revision required before advancement  
**Blocker:** All 15 issues must be resolved; R1–R5 especially critical (semantic safety)  
**Follow-up:** Don (Backend Dev) assigned as independent reviser; Edith locked out per directive

## Notes

- Evaluation scope: conformance wiring, schema consistency, and specification completeness
- Architecture ruling (AR-TP-1..10) remains authoritative; no re-litigation of ruled decisions
- Defects are not architectural departures but locally-fixable wiring failures and omissions
- Examples and schemas are the highest-risk areas for user-facing breaks; thorough re-review of all 15 will be needed at gate #2
