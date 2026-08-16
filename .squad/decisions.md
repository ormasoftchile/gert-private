# Squad Decisions

**Last Updated:** 2026-08-15T19:31:53Z
**Archive Trigger:** decisions.md >= 51200 bytes threshold; entries older than 7 days archived to decisions-archive.md

---

## 2026-08-09 — GERT Tool Packages MVP — Spec Authored (Barbara ruling actioned)

**By:** Edith (Spec Editor)
**Ruling:** `.squad/decisions/archive/barbara-tool-packages-architecture-ruling.md` (Barbara, ratified 2026-08-09, AR-TP-1..10) — moved from `inbox/` to `archive/` as fully actioned; no clauses reopened, no contradictions found.

**What shipped:** The complete Tool Packages MVP is now normative spec text, not a proposal:

- **Bindings (AR-TP-1):** `requires:` is canonical; `toolPackages:` is rejected (`PKG-020`, no alias, no dual-read); `toolRefs:` binds/narrows only; `alias` removed from the `ToolRef` schema; `source`/`actions` retained as deprecated (D-001/D-002), non-enforcing, warning-only (`PKG-W002`/`PKG-W001`).
- **Resolution (AR-TP-2):** Two-phase (catalog freeze, then per-file bind) discovery, 5 deterministic tiers, same-tier collision is `PKG-006`, undeclared cross-tier shadowing is `PKG-022` (the one intentional breaking change vs. prior "later wins" text), enumeration fully sorted for byte-identical catalog digests.
- **Schemas (AR-TP-3):** New `tool-package/v1`, `config/v1`, `package-lock/v1` schemas; `runbook.v1.schema.json` gained `requires`/`PackageRequirement` additively.
- **Versioning (AR-TP-4):** Strict SemVer 2.0.0, small conjunctive constraint grammar, validation only — no solving.
- **Substitution (AR-TP-5):** Declared in `.tool.yaml`, resolved relative to the declaring file, exact I/O signature match, governance composed by the ruled invariant table (never widened), max depth 4, traced as nested spans.
- **Paths (AR-TP-6):** POSIX-only authored paths, realpath containment, junction/reparse handling, escape is `PKG-007` hard fail.
- **Digests/trace/resume (AR-TP-7):** Raw-byte SHA-256, order-independent sha256sum-style digests; 5 new trace events; resume hard-refuses on drift (`PKG-009`, overridable), replay is non-fatal on drift.
- **Scoping (AR-TP-8):** Tool-name binding is lexically scoped per file (asymmetric with shared variable scope); package set is global per run; lazy includes are still package-analysed at plan time.
- **Conformance (AR-TP-9/9b):** Full `PKG-001..028` + `PKG-W001..003` error catalog registered in `03d-parse-time-enforcement.tex`; 7 new categories + `PackageExpected` shape registered in `conformance/vector.schema.json` for Tess to author `tv-pkg-resolve.yaml` against (not authored by Edith — file ownership per ruling).
- **No shims (AR-TP-10):** Pre-1.0, no compatibility shims; the cross-tier shadowing behaviour change is documented as intentionally breaking.

**Files changed:** see `.squad/agents/edith/history.md` entry "2026-08-09T14:44:49-07:00 — GERT Tool Packages MVP — Full Spec Authoring" for the full file list.

**Validation:** `verify_corpus.py` 280/280 vectors validate (no regressions); full `tectonic` LaTeX build succeeds; all schemas Draft 2020-12 valid; example package + r23 fixture validate against their schemas.

**Deferred to Tess:** `conformance/tv-pkg-resolve.yaml` and its conformance vectors — explicitly out of Edith's scope per the ruling's file-ownership split.

**Status:** Ratified ruling fully actioned into spec. Ready for review.

---

## 2026-08-09 — GERT Tool Packages MVP — Gate Review #1 (REJECTED)

**By:** Barbara (Lead / Architect)  
**Date:** 2026-08-09T14:47-07:00  
**Requested by:** Cristián Ormazábal Ortega  
**Reviews:** Edith's Tool Packages MVP spec/schemas/fixtures against `.squad/decisions/archive/barbara-tool-packages-architecture-ruling.md` (AR-TP-1..10, ratified)  
**Verdict:** **REJECTED** — 15 required corrections (R1–R15); architecture direction sound, defects are locally fixable wiring and under-specification.  
**Reviser designated:** **Don** (Backend Dev / runtime semantics). **Edith is locked out** per reviewer lockout protocol.

**Full entry:** `.squad/decisions/inbox/barbara-tool-packages-gate-review.md`

### R1–R15 Blocking Issues (Summary)

| # | Category | Issue |
|---|----------|-------|
| R1–R5 | Semantic contradictions / under-specification | Action shape (map vs. list), `impl:` collision, undefined output capture, invalid reference example, unstated path bases |
| R6–R10 | Wiring failures | PKG-020/021 unreachable, `dependencies` schema contradiction, `apiVersion` mismatch, tier-3 incompatible, collision rule contradiction |
| R11–R15 | Wiring failures (medium severity) | Undefined replay semantics, unreconciled `tool.version`, unenforced lock invariant, missing error codes, no real/mock acceptance scenario |

### Requirements

1. **R1:** Establish one canonical `tool/v1` action shape; reconcile `args/output` with `inputs/outputs`
2. **R2:** Disambiguate per-action `impl:` key (collides with top-level mobile `impl:`)
3. **R3:** Define `outputs.<name>` capture namespace for substituted actions
4. **R4:** Rewrite reference substitute so it produces declared output via valid GCP
5. **R5:** Create explicit "Resolution base per path kind" table; verify no escapes
6. **R6:** Add mandated pre-schema raw-document scan for `toolPackages:` and `alias:`
7. **R7:** Fix `dependencies` schema/PKG-016 contradiction
8. **R8:** Normalize `apiVersion` values across examples and schemas
9. **R9:** Make tier-3 addressing schema-compatible (qualified `name` only)
10. **R10:** Split collision paragraph into cross-source vs. same-source cases
11. **R11:** Define substitution replay behavior (relates to non-fatal `replay/packageDrift`)
12. **R12:** Reconcile step-level `tool.version` as deprecated/intersected constraint
13. **R13:** Tighten lock-root pattern to structurally forbid raw absolute paths
14. **R14:** Assign error codes for unspecified conditions
15. **R15:** Provide concrete unchanged-runbook scenario with mode variations

### Deferred Tess Work

Tess's `conformance/tv-pkg-resolve.yaml` (≥46 vectors) blocked pending R1–R9 resolution. Vector target list specified; do not author until R1–R9 land.

### Gate Disposition

**REJECTED.** Fifteen required corrections identified. Architecture ruling remains authoritative; no re-litigation. R1–R5 are highest-risk semantic areas (substitution, path safety); R6–R15 are wiring failures that make ruled error codes unreachable or ruled mechanisms unusable. All locally fixable.

**Reviser:** Don (independent, Edith locked out). Re-submit for second gate pass.

---

## 2026-08-09 — GERT Tool Packages MVP — Rejection Revision (R1–R15), by Don (independent)

**By:** Don (Backend Developer), acting independently as revision owner. Edith (original
author) was locked out per Cristian's directive and did not advise or contribute to this
revision.

**Subject:** Barbara rejected Edith's Tool Packages MVP spec/schemas/fixtures
(`.squad/decisions/inbox/barbara-tool-packages-gate-review.md`, 15 blocking items
R1–R15). This entry records Don's resolution of all 15, plus the optional R16
(`warnings:` on `PackageExpected`), without reopening the ratified architecture ruling
(`.squad/decisions/archive/barbara-tool-packages-architecture-ruling.md`, AR-TP-1..10).

**Resolution summary (see `.squad/decisions/inbox/don-tool-packages-revision-r1-r15.md`
for the full R1–R15 matrix, file list, validation outcome, and Tess follow-up notes):**
- R1: `actions:` is always a list; unified `args:` vocabulary; `output:` vs `outputs:` kept deliberately distinct.
- R2: per-action `impl:` renamed `execute:` to stop colliding with top-level mobile `impl:`.
- R3: new `outputs.<name>` GCP capture root for substituted-action outputs (`grammar/gcp.ebnf`).
- R4: reference substitute (`drain-node.yaml`) reworked to actually produce its declared output via valid GCP, not a literal.
- R5: new "Resolution base per path kind" table in `06-tool-runtime.tex` naming containment roots explicitly.
- R6: PKG-020/PKG-021 made reachable via a mandated pre-schema raw-document scan.
- R7: `dependencies` schema no longer contradicts PKG-016 reachability.
- R8: `apiVersion` values corrected across manifest examples.
- R9: tier-3 addressed only via qualified `name`, never `package` (schema-compatible with MCP single-label names).
- R10: §05 collision paragraph split into cross-source (precedence) vs. same-source (hard error).
- R11: substitution replays like `invoke` (executes against the caller's scenario file); tied explicitly to non-fatal `replay/packageDrift`.
- R12: `ToolInvocation.version` redefined as plan-time-evaluated, intersected, deprecated in favor of `toolRefs[].version`.
- R13: lock-file `root` pattern structurally forbids raw absolute paths.
- R14: new `PKG-029` (package+path mutual exclusivity) and `PKG-030` (unresolved name) error codes.
- R15: r23 fixture now documents an unchanged-runbook real/dry-run/replay acceptance scenario with a worked scenario-file override map.
- R16: `PackageExpected.warnings` added to `conformance/vector.schema.json`.

**Validation:** All touched JSON schemas valid; all touched YAML fixtures parse;
`verify_corpus.py` 280/280 vectors validate (no regression from the `vector.schema.json`
change); full `tectonic` LaTeX build of `main.tex` succeeds with no new errors or
undefined references (the two pre-existing undefined refs are in untouched files, out of
scope).

**Deviations flagged for Barbara's second gate pass (not fixed, out of scope):** a
pre-existing terminology overload between §07's trace-event replay and §13's
execution-mode replay (both called "replay"); the corpus-wide `apiVersion: runbook/v2`
issue (Barbara's non-blocking note #5, separately ticketed).

**Deferred to Tess (unchanged from the ruling):** the ≥46-vector
`conformance/tv-pkg-resolve.yaml` corpus. Not authored in this revision per explicit
task scope — Tess should author it against the now-resolved contracts once this revision
clears gate review.

**Status:** Revision complete; returned to Barbara for second gate pass.

---

## 2026-08-09 — GERT Tool Packages MVP — Gate Review #2 (REJECTED, narrow scope)

**By:** Barbara (Lead / Architect)  
**Date:** 2026-08-09T15:40-07:00  
**Requested by:** Cristián Ormazábal Ortega  
**Reviews:** Don's R1–R15 revision against `.squad/decisions/archive/barbara-tool-packages-architecture-ruling.md` (AR-TP-1..10) and `.squad/decisions/inbox/barbara-tool-packages-gate-review.md`  
**Revision under review:** `.squad/decisions/inbox/don-tool-packages-revision-r1-r15.md` (Don, independent)  
**Verdict:** **REJECTED** — narrow scope: S1 (documentation) and S2 (documentation) sweep required. R1–R15 verified resolved; architecture locked.  
**Reviser designated:** **Ken** (Backend Dev). **Edith and Don are both locked out** — they authored the original and the revision now under review.

**Full entry:** `.squad/decisions/inbox/barbara-tool-packages-gate-review-2.md`

### R1–R15: Verified Resolved (by manual re-derivation)

All 15 items re-derived from actual artifacts (not taken on trust) and verified resolved:
- R1–R5: Semantic contradictions fixed (action shape, `impl:` collision, output capture, reference example, path bases)
- R6–R15: Wiring failures fixed (PKG-020/021 reachable, schema contradictions, apiVersion, tier-3, collision, replay, version, lock invariant, error codes, scenarios)
- R16 (optional): `warnings:` added to `PackageExpected`

**Validation re-run:** `verify_corpus.py` 280/280, `jsonschema` 0 errors on all fixtures, `\label`/`\ref` closure 0 unresolved.

### S1 — Documentation Issue: §03 `\subsection{toolRefs}` teaches deleted rules

**Location:** `03-schema-vnext.tex` lines ~170–188

**Problem:** Schema chapter (first place readers look for document shape) still instructs authors to use `alias:` (now `PKG-021` hard error) and default discovery path `tools/<name>.tool.yaml` (deleted by AR-TP-2 ratified tier model).

**Fix Required:** Rewrite subsection to drop `alias`, drop default-discovery sentence, cross-reference §06 ratified tier model instead of restating it locally.

### S2 — Documentation Issue: Three `.tool.yaml` listings use forbidden mapping-shaped `actions:`

**Locations:** 
1. `03-schema-vnext.tex:3175` (`actions: {get-pods: ...}`)
2. `03-schema-vnext.tex:3296` (`actions: {check: ...}`)
3. `08-security-and-trust.tex:320` (`actions: {deploy: ... sensitive_inputs ...}`)

**Problem:** §06 now mandates `actions:` is always a list, never a mapping. Three listings contradict this. §08 case contains the only normative statement of `sensitive_inputs` placement.

**Fix Required:** Convert all three to list form (`- name: ...`), preserving all fields including `sensitive_inputs`.

### Optional Cleanups Approved

Four mechanical, non-blocking cleanups authorized:
1. Stale `inbox/` ruling citations → `archive/` (15 occurrences)
2. Package name drift `com.acme.` → `acme.incident-tools` (30 occurrences)
3. `LocalStructured` → `LocalOutputs` §3.4a mis-citation
4. r23 dry-run narrative accuracy

### Gate Disposition

**REJECTED**, narrowly. All R1–R15 verified genuine resolved and locked. Two new blockers (S1, S2) identified — pure documentation sweep with no design decisions needed. The correct answers are already written in §06 and §08; §03 and §08 just need to be brought into alignment. Spec goes to user next; reader opening schema chapter first must not be told to author keys that parse gate rejects.

**Reviser:** Ken (independent, Edith and Don locked out). Scope strictly limited to S1/S2. Re-submit for third gate pass; will be diff-only check + validation re-run.

---

## 2026-08-09 — GERT Tool Packages MVP — S1/S2 Documentation Sweep (Ken, independent)

**By:** Ken (Backend Developer), acting independently as revision owner. Edith (original author) and Don (first revision author) were locked out per Cristián's directive.

**Subject:** Barbara's second gate identified two documentation-only issues (S1, S2) requiring mechanical sweep. Scope explicitly limited to §03 and §08; AR-TP-1..10 and R1–R15 remain closed and were not reopened.

**Resolution summary (see `.squad/decisions/inbox/ken-tool-packages-revision-s1-s2.md` for full matrix and validation):**

- **S1 resolved:** `03-schema-vnext.tex` §03 `\subsection{toolRefs}` rewritten: removed `alias:` example (replaced with one-line `PKG-021` note), removed default-discovery-path sentence, added cross-references to §06 ratified tier model. All cross-references verified.
- **S2 resolved:** All three mapping-shaped `actions:` listings converted to canonical list form (`- name: ...`); `sensitive_inputs` preserved under §08's converted entry. Corpus-wide scan: zero mapping-shaped `actions:` blocks remain.
- **Optional cleanups completed:** Stale `inbox/` ruling citations swept to `archive/` (0 remaining); `com.acme.` → `acme.incident-tools` normalized (30 occurrences consistent); `LocalStructured` → `LocalOutputs` §3.4a mis-citation corrected; r23 dry-run narrative made accurate; one clarifying sentence on `StepCapture` form scope.

**Validation:** `verify_corpus.py` 280/280, all 5 schemas valid JSON, `jsonschema` 0 errors, `tectonic` build 0 new errors, `\label`/`\ref` closure 0 unresolved, full diff review confirms only S1/S2 changes + optional cleanups (no collateral edits).

**No semantic changes:** No schema modifications, no error codes changed, no grammar productions altered, no runtime semantics modified. AR-TP-1..10 and R1–R15 remain byte-compatible with Gate 2 acceptance.

**Status:** Revision complete; returned to Barbara for third (final) gate pass.

---

## 2026-08-09 — GERT Tool Packages MVP — Gate Review #3 (final): APPROVED

**By:** Barbara (Lead / Architect), at Cristian Ormazabal Ortega's request.
**Reviews:** Ken's S1/S2 sweep (`.squad/decisions/inbox/ken-tool-packages-revision-s1-s2.md`),
authored independently with Edith and Don locked out.
**Full entry:** `.squad/decisions/inbox/barbara-tool-packages-gate-review-3.md`

**Verdict: APPROVED. The specification is ready for Cristian's review.**

- **S1 resolved.** `03-schema-vnext.tex` `\subsection{toolRefs}` no longer teaches `alias:` (now
  `PKG-021`, hard parse-gate error) and no longer teaches the deleted `tools/<name>.tool.yaml`
  default-discovery rule; it now states the negation and cross-references the ratified
  frozen-catalog/tier model in §06 instead of restating it. All three cross-references resolve.
- **S2 resolved.** All three mapping-shaped `actions:` listings converted to the canonical list
  form; `sensitive_inputs` preserved under §08's converted `deploy` entry. Corpus-wide scan of
  every `actions:` block (`.tex`/`.yaml`/`.md`): zero mapping-shaped occurrences remain.
- **No stale alias / default-discovery guidance survives in any normative example.** Remaining
  `alias` hits are prohibitions, the unrelated `imports:` alias map, or GXL/GCP function aliases.
- **Ken's optional cleanups altered no accepted semantics.** Stale `inbox/` ruling citations swept
  to `archive/` (0 remaining); `com.acme.` -> `acme.incident-tools` normalized consistently and
  still schema-valid; `LocalStructured` -> `LocalOutputs` §3.4a mis-citation corrected; r23's
  dry-run narrative made accurate; one clarifying sentence added that `StepCapture` gains no
  `step.{id}.outputs.{name}` form (MVP scope choice). No schema, error code, grammar production,
  or runtime semantic changed. AR-TP-1..10 and R1-R15 remain closed.
- **Validation re-run independently:** `verify_corpus.py` 280/280; all 5 JSON Schemas valid;
  `gert-package.yaml` and `drain-node.yaml` validate with 0 errors; `\label`/`\ref` closure 293
  labels / 0 unresolved. r23's single `apiVersion: runbook/v2` error is the known corpus-wide
  pre-existing issue.

**Deferred, Tess-owned (unblocked, not blocking):** `conformance/tv-pkg-resolve.yaml`, >=46
vectors, against the now-stable surfaces (`PKG-*` enum, `TV-PKG-*` ids, `PackageExpected` incl.
`warnings:`, `outputs.<name>`, `execute:`, `PKG-029`/`PKG-030`, `tab:tool-path-bases`,
replay-as-`invoke`). Gate-2 §3 target list stands.

**Non-blocking pre-existing issues (own tickets):** corpus-wide `apiVersion: runbook/v2`; the
§07/§13 `replay` terminology overload; the spec-wide under-specification of downstream behaviour
after an upstream mode-skip; the optional r23 `assessment.md` companion. One editorial nit: a
`\S\ref` macro leaks verbatim inside a `minted` comment at `03-schema-vnext.tex:184` — typographic
only, fold into the next §03 touch.

**No reviser designated. No further gate pass required.**

---

## 2026-08-09T15:59:36-07:00 — Tool Packages MVP: Runtime Implementation (Don)

**By:** Don (Backend Developer), at Cristián Ormazábal Ortega's request.
**Repo:** `C:\One\OpenSource\gert` (runtime repo). **Status: not committed** (working-tree only,
per task instructions) — this is a status/decision entry, not a merge record.
**Full entry:** `.squad/agents/don/history.md` (2026-08-09 section) has the complete file/scope
inventory.

**Decision: ship a realistic, fully-tested core subset now rather than a shallow full-surface
attempt.** Implemented and unit-tested: PKG-001..030/PKG-W001..003 error codes; strict SemVer +
MVP constraint grammar; secure path resolution with symlink containment; `tool-package/v1`,
`config/v1`, `package-lock/v1` schema types; the PKG-020/021 pre-schema forbidden-key scan; the
full two-phase (`Build`/`BindFile`) five-tier catalog freeze and resolution engine with all
ratified collision rules (PKG-006/011/022/029/030) and digest algorithm; a new
`internal/adapter.BuildPackageCatalog`/`ResolveToolRefsViaCatalog` API pair for Ken's future CLI
wiring, added alongside (not replacing) the existing path-only `ResolveToolRefs`.

**Explicitly deferred, reported not hidden:** substitution (`execute.kind: runbook`) end-to-end;
evidence/trace event emission and resume/replay drift behavior (`PKG-009`,
`--allow-package-drift`, `replay/packageDrift`); real MCP/extension tier-3 discovery (only an
extension point exists); `.gert/config.yaml` loading into `pkg/run/run.go`; a fully faithful
two-resolution-base implementation for `requires[].path` (project-scope vs runbook-scope) —
currently a workspace-then-runbook-dir fallback heuristic. Also flagged, not fixed: the existing
runtime's `ToolDef.Actions` is a `map[string]*ToolAction`, diverging from the ratified spec's
"actions MUST be an array" rule — converting it was judged out of scope (large, unrelated,
sweeping breaking change) for this task.

**Bug fixed during implementation:** `toolRefs[].version` was originally a resolution no-op
(the runtime `ToolDef` type carries no version field); fixed by threading
`schema.ToolDef.Version` through to a new `pkgcatalog.Entry.Version` field and enforcing
`semver.Constraint.Satisfies` in the package-pin and tier-4 path binding, per the ratified
"checked against the resolved tool definition's meta.version" rule. One pre-existing test fixture
had been silently passing under the old no-op and was corrected.

**Validation:** `go build ./...` and full `go test ./...` are green across the entire repo with
zero regressions (including all 22 pre-existing runbook parser fixtures). The mandated dirty-tree
constraint files (`native.go`/`native_test.go`/the example runbook/the untracked `.code-workspace`)
were inspected once, never modified, and verified unchanged at session end.

**Deferred, still-owned-by-others work is unaffected:** Tess's `conformance/tv-pkg-resolve.yaml`
remains unblocked and untouched; the `PKG-*`/tier surfaces it will test against are now
implemented and unit-tested (not conformance-vector-tested) in `pkg/pkgcatalog`.

**No reviser designated for this entry. Next owner (Ken for CLI wiring, or whoever picks up
substitution/evidence/resume) should treat `pkg/pkgcatalog`/`pkg/semver`/`pkg/pkgpath` as stable,
tested building blocks, not scaffolding to be redesigned.**

---

## 2026-08-09T17:58-07:00 — Tool Packages MVP: Implementation Gate Review (Barbara) — REJECTED

**By:** Barbara (Lead/Architect), at Cristian Ormazabal Ortega's request.
**Under review:** the complete uncommitted working tree of `C:\One\OpenSource\gert`
(Don: runtime core + final integration pass; Ken: CLI wiring), against AR-TP-1..10, the
TV-PKG-PATH-002 binding ruling, the gate-3-approved spec, `gcp.ebnf` 3.4a, and Tess's
85-vector corpus. Production wiring read directly; agent summaries used only to locate code.
**Full entry:** `.squad/decisions/inbox/barbara-tool-packages-implementation-gate-review.md`.

**Verdict: REJECTED. Revision owner: David (Integration Engineer)** — Don and Ken are locked
out as authors of the code under review; the residual work is integration/wiring, which is
David's competence. Edith and Tess remain locked out of runtime code by role.

**Verified correct (not re-litigable):** SemVer + constraint grammar; secure path resolution
and exact conformance to my TV-PKG-PATH-002 workspace-escape ruling; two-phase Build/BindFile
with no post-freeze mutation; five tiers with tier-3 bare-name stripping; PKG-006 hard
collision and genuine PKG-022 enforcement; the full binding contract incl. the Phase-0
raw-YAML PKG-020/PKG-021 scan; non-fatal warning discipline; `.gert/config.yaml` genuinely
loaded on the production path; `--package-map` partial override proven with a byte-identical
runbook through the real CLI; substitution declaration/scope isolation/signature exactness;
exact governance composition arithmetic incl. a true `allow_commands` intersection; the three
trace events emitted at the real Phase-C boundary with a shared run_id; dry-run side-effect
avoidance; resume PKG-009 refusal and `governance/packageDriftAccepted` with both digests and
operator. Protected user edits byte-identical; dirty tree preserved; zero regressions.

**Blockers:** (B1) `pkgsubst.Plan` has one call site — inside the executor — so PKG-013/014/
015/026/027/028 fire only when a step executes; unreachable steps and all of dry-run are
unvalidated, contradicting 5.4/5.5/5.6 verbatim. (B2) package digest closure omits
`execute.path` substitutes and package-internal includes, so a substitute runbook can be
rewritten without changing any digest — defeating 7.5 resume integrity. (B3) catalog digest
uses the export file digest where 7.3 requires the package digest for tier-1. (B4) resume
drift iterates only currently-present packages, so a removed package is never detected and
PKG-001 is unreachable. (B5) the ratified `outputs.<name>` capture root (`gcp.ebnf` 3.4a,
`LocalOutputs`) is absent from the runtime GCP parser. Plus a sixth, found in review: the
include closure is never traversed — child `requires:`/`toolRefs:` are ignored, so resolution
is dynamically scoped, the exact model 8.1 rejected; implement it or fail closed. Nine
lower-severity required fixes are listed in section 3 of the full entry.

**Don's reported gaps, classified:** deep governance deny/allow enforcement — ACCEPTABLE
NON-GOAL (the evaluator is orphaned repo-wide, so substitution grants no relative escalation;
conditional on B1 landing, ticketed). Capture of substitution outputs — BLOCKER (B5): the
grammar was ratified and simply not implemented. In-memory resume plan — ACCEPTABLE NON-GOAL,
genuinely pre-existing; Don's refusal to fake a passing CLI test was the right call; ticket
plan persistence and document drift-checking as in-process-only. Empty `ConstraintSources` —
REQUIRED FIX, not a non-goal: the provenance already exists in `mergeRequirements`, and a
permanently-empty field in an evidence record is worse than an absent one. Absent replay mode
— ACCEPTABLE NON-GOAL, confirmed no replay entry point exists anywhere.

**A second gate pass is required, scoped to B1-B5, the include-closure item, and section 3.**

---

## 2026-08-09 — Tool Packages MVP (runtime): FINAL IMPLEMENTATION GATE — APPROVED

**Barbara (Lead / Architect), second gate pass.** Full entry:
`.squad/decisions/inbox/barbara-tool-packages-final-gate-decision.md`. Reviewed David's
independent revision (Don and Ken locked out, and they made no contribution) against my
rejecting first pass, AR-TP-1..10, the TV-PKG-PATH-002 ruling, and the gate-3 spec.

**Verdict: APPROVED. Ready for Cristián.** No third gate pass required.

All six blockers verified resolved in the code that actually runs: (B1) plan-time
substitution validation via a structural `flowwalk` visitor invoked before `Plan`/`Start`
in the shared `runWithMode`, so dry-run and unreachable-by-`when:` steps are validated,
with cycles/depth decided statically by DFS frames; (B2) digest closure now covers
`execute.path` substitutes and package-internal includes, cycle-guarded and sorted;
(B3) tier-1 entries carry the package digest, so `CatalogDigest()` proves what §7.3 says;
(B4) the manifest's `PackageDigests` is authoritative — a removed package is PKG-001 by
name, an added one PKG-009; (B5) the ratified `outputs.<name>` capture root is implemented
end-to-end with the step-context check at plan validation, plus a genuine latent-bug find
(`schemaToolDefFromRuntime` dropping `Execute`/`Outputs`, which would have silently
defeated B1 and B5); (§5) the include closure fails closed with a typed PKG-017 rather
than resolving dynamically — the sanctioned fallback (b), with option (a) ticketed.

All nine §3 fixes verified, including real `ConstraintSources`, PKG-002 provenance, PKG-003
on build metadata, deterministic PKG-006 ordering, typed PLAN-010 that still unwraps to
`ErrToolNotFound`, live PKG-018 normalisation, failing (not silent) output coercion, and
`origin` in the `package/resolved` payload. `maxLinkHops` I verified **myself on Windows**
against real symlinks (10 hops → PKG-008; 8 hops → clean), since its tests skip there.

Protected files byte-identical, no `design/` file touched during the revision window, tree
dirty and uncommitted as instructed, `go build ./...` clean, `go test ./...` green apart
from one unrelated timing flake in `internal/serve` that passes 5/5 on re-run.

**Accepted non-goals:** deep governance enforcement inside substitute bodies (orphaned
repo-wide, no relative escalation); cross-process resume plan persistence (pre-existing —
Cristián must be told `--allow-package-drift`/PKG-009 are in-process-only today); replay
mode; §5 option (a). **Follow-ups ticketed:** lexical `BindFile` binding, documenting the
single-file `requires:`/`toolRefs:` restriction, `GovernanceEvaluator` wiring,
`ExecutionPlan` persistence, Windows link-hop tests, hop counting across intermediate
components, and the pre-existing corpus/terminology items.

---

## 2026-08-10 — Enum-Constrained Tool and Runbook Outputs MVP — RATIFIED ARCHITECTURE

**By:** Barbara (Lead / Architect), at Cristián Ormazábal Ortega's request.
**Date:** 2026-08-10T13:25:10-07:00
**Full entry:** `.squad/decisions/archive/barbara-enum-constraint-mvp-architecture-ruling-archived.md`

**Verdict: RATIFIED.** Architecture ruling (AR-ENUM-1..15) with three binding scope corrections (C1/C2/C3):
- C1: `enum` forbidden on `type: secret`; member lists redacted on sensitive declarations (audit-trail safety)
- C2: Package mock enum equality is conformance-only, not a runtime check (no in-run comparand)
- C3: No `tool.v1.schema.json` in this MVP; tool-action `enum` lives in `06-tool-runtime.tex` prose

Four declaration sites: tool action `args`/`outputs` (S1/S2), runbook `inputs`/`outputs` (S3/S4).
String-only constraint; type-restricted; checked at parse time (declarations, defaults) and runtime (bindings).
ENUM-001..009 error codes; ENUM-W001 warning (case-only-distinct); PKG-013 extended for substitution enum-set equality.
Asymmetric Unicode normalization (declared members must be NFC; candidate values are NFC'd before comparison).
No integer/identifier/label-value unification with collectors in this MVP. No enum on secrets. No enum identity in package digests.

---

## 2026-08-10 — Enum MVP: Specification Work (Edith) — OPEN / IN PROGRESS

**By:** Edith (Spec Editor)
**Date:** 2026-08-10T13:25-07:00
**Status:** Analysis complete, awaiting Barbara's ruling on scope questions before authoring.
**Full entry:** `.squad/decisions/inbox/edith-string-enum-args-io.md`

Four schema/prose questions for Barbara's sign-off before Edith authors the normative sections:
- Q1: Enum enforcement at parse time (literals) and runtime (bindings) — recommendation is both (mirrors GCP-TYPE-001 pattern)
- Q2: Runbook `Output` schema vs prose conflict (schema newer, pre-enum); treating schema as canonical and rewriting L403-420 prose
- Q3: Error-code family — recommend `SEM-0xx` for runbook inputs/outputs, `PKG-030` for tool-action violations
- Q4: `pattern:`/`example:` on `Input` are prose-only (dead fields, schema doesn't have them); fix in same PR as adding `enum:` to avoid third generation of drift

**Owner:** Edith. **Blockers:** Barbara's Q1-Q4 approval.

---

## 2026-08-10 — Enum MVP: Corpus Work (Tess) — COMPLETE

**By:** Tess (Conformance Tester)
**Date:** 2026-08-10
**Status:** 58-vector corpus finalized; one schema gap found and documented.
**Full entry:** `.squad/decisions/archive/tess-enum-corpus-notes-archived.md` + `.squad/decisions/archive/tess-enum-decl-006-correction-archived.md`

**TV-ENUM-DECL-006 corrected:** YAML 1.2 core schema fact. Bare `yes`/`no` resolve to `!!str`, not `!!bool` (1.1 was the boolean resolver).
Vector's fixture amended: `enum: [yes, no]` → `enum: [true, false]` (the canonical YAML-1.2-core-schema booleans, which DO trigger ENUM-002).
Vector id/category/expectation intent preserved; count stays 58.

**Finding: `$defs.Output` in `runbook.v1.schema.json` lacks `default` property** (unlike Input).
AR-ENUM-6 rule 3 includes S4 (runbook output defaults) in the "default must be a member" sites, but the schema has no `default` key at S4.
Documented as untestable-in-corpus, not a vector defect — a schema gap for later resolution (add `default` to Output, or explicit ruling that S4 defaults are schema-free).
No other ambiguities found; all AR-ENUM-1..15 rules mapped cleanly.

**Owner:** Tess (final). **Tess owns four further corpus amendments** (UNICODE-005/PLAN-005/RUNTIME-004/PLAN-003) to fix defects diagnosed during Ken's R2 harness run;
she does not edit the frozen corpus otherwise.

---

## 2026-08-10 — Enum MVP: Initial Runtime Implementation (Don) — COMPLETE (SUPERSEDED)

**By:** Don (Backend Developer)
**Date:** 2026-08-10
**Status:** Reported; implementation superseded by Ken's independent revision.
**Full entry:** `.squad/decisions/archive/don-enum-mvp-implementation-report-archived.md`

Implemented AR-ENUM-1..15 end-to-end in runtime (`gert` repo). Two findings escalated to Barbara:
- Finding 1: `TV-ENUM-DECL-006` vector conflicts with this repo's YAML 1.2 resolver (not a code error, a vector/library conflict); requested Barbara's ruling.
- Finding 2: No root-runbook output-materialization path exists in the engine (pre-existing gap, S4 enforcement unreachable, ticketed T-ENUM-ROOT-OUTPUTS).

Full validation: `go build ./...` clean, `go test ./...` all 61 packages pass.

**NOTE:** Don reported the genuine DECL-006 conflict and the root-output gap correctly. However, his implementation work included five runtime defects that silently defeated enum enforcement for large fixture families.
Ken's independent R2 harness (built later) surfaced all five (GCP output resolution, dropped Enum field in catalog conversion, missing S2 default check, capture-after-failure masking, GIS-interpolation false rejection).
Per Barbara's gate-rejection process, Don is locked out; Ken revises independently.

---

## 2026-08-10 — Enum MVP (runtime): Implementation Gate Review 1 (Barbara) — REJECTED

**By:** Barbara (Lead / Architect)
**Date:** 2026-08-10
**Status:** Gate rejected; revision owner designated.
**Full entry:** `.squad/decisions/archive/barbara-enum-mvp-implementation-gate-archived.md`

Reviewed Don's complete uncommitted working tree (`C:\One\OpenSource\gert`) against AR-ENUM-1..15, Tess's 58-vector corpus, and §R1-R5 gate criteria.

**Five blockers identified (R1–R5):**
- R1: ENUM-008 caller-binding enforcement incomplete; missing `--var` path
- R2: No faithful conformance harness; design-only vectors not mechanized
- R3: Enum metadata not carried in `ValidatedPlan`
- R4: ENUM-W001 warning not surfaced to end-user
- R5: Replay path not validated

All five are genuine, non-negotiable blockers. **Revision owner: Ken (Backend Developer).** Don and Ken locked out as authors; Ken revises independently from scratch per the gate-rejection protocol.

---

## 2026-08-10 — Enum MVP (runtime): Independent Revision (Ken) — COMPLETE + R1–R5 VERIFIED

**By:** Ken (Backend Developer), independent reviser.
**Date:** 2026-08-10
**Status:** Revision complete; all R1–R5 blockers resolved; bugs found and fixed; harness green.
**Full entry:** `.squad/decisions/archive/ken-enum-mvp-implementation-revision-archived.md`

Revised R1–R5 from scratch against ratified architecture (AR-ENUM-1..15), 58-vector frozen corpus, and gate-rejection spec.

**Final disposition matrix:**

| Blocker | Status | Evidence |
|---|---|---|
| R1 — ENUM-008 caller-binding | **Resolved** | `schema.CheckCallerInputBindings` wired at `cmd/gert/run.go` entry and all RPC/API paths; `internal/executor/tool.go` CheckArgEnums on materialized tool args (incl. `--var`-sourced values). Regression test: `enum_r1_r4_regression_test.go`. |
| R2 — 58-vector conformance harness | **Resolved** | `internal/conformance/enum_harness.go` + `enum_vector.go` + `enum_conformance_test.go` build real `gert` CLI, materialize each vector into workspace, run it (or drive `pkgcatalog.Build` for catalog vectors). **Final report: 48 passed, 10 skipped (named), 0 failed.** |
| R3 — Enum metadata in ValidatedPlan | **Resolved** | `internal/planner/enumplan.go` populates `ValidatedPlan.EnumConstraints`; `internal/engine/engine.go` carries it once in `plan.validated` trace, declared order, C1-safe redaction. Regression test: `enum_trace_test.go`. |
| R4 — ENUM-W001 surfaced | **Resolved** | `cmd/gert/run.go` surfaces ENUM-W001 to stderr without aborting. Regression test: `enum_r1_r4_regression_test.go`. |
| R5 — Replay enum validation | **Resolved** | `internal/executor.CheckArgEnums` exported; `internal/replay.ReplayExecutor.WithEnumChecks` + `ReplayFromTrace` wiring apply identical check at replay boundary. Regression test: `enum_r5_test.go`. |

**Ten genuine runtime bugs found and fixed** (not in Ken's charter, but revealed by building a faithful R2 harness):
1. GCP output-value resolution (executeSubstitution was not resolving GCP paths in output values)
2. Dropped `Enum` field in catalog/toolRefs conversion (schemaToolDefFromRuntime lost Enum on both Args and Outputs)
3. Missing S2 default check (AR-ENUM-6 tool-output-default case never implemented)
4. Capture-after-failure masking (failed step captures attempted anyway, hiding real ENUM-008/009)
5. GIS-interpolated defaults falsely rejected at plan time (ENUM-006 stringified `"${count}"` literally)
6. Missing `imports:` alias resolution for `include.runbook` (alias-by-name includes failed as literal file paths)
7. Schema/struct drift on `expand:` property (runbook/include Expand field existed in Go but not in JSON schema)
8. Stdout/stderr split in harness (plan-time errors go stderr, runtime failures go stdout; harness only checked stderr)
9. Temp build directory polluting repo (enumharness-bin-* dirs in working tree)
10. Validation-ordering fix (B1 substitution checks short-circuited before planner.Plan ran, masking ENUM-006/007)

**Authoritative 58-vector execution report:**
- 48 vectors pass ✓
- 10 vectors skip with named, audited reasons (5 pre-existing ticketed gaps, 5 corpus/methodology defects)
- 0 vectors fail
- 0 vectors silently dropped

All blockers substantively resolved. No architecture reopened. Dirty tree preserved (no commits, no stage, protected files byte-identical).

---

## 2026-08-10 — Enum MVP: Final Implementation Gate (Barbara) — APPROVED

**By:** Barbara (Lead / Architect)
**Date:** 2026-08-10T17:45-07:00
**Status:** Final gate passed; implementation complete and approved.
**Full entry:** `.squad/decisions/archive/barbara-enum-mvp-final-gate-approval-archived.md`

Reviewed Ken's independent revision against R1–R5 spec, AR-ENUM-1..15, Tess's corrected 58-vector corpus, and Edith's AR-ENUM-3(3) prose fix.
Verified directly: read current runtime diff (39 modified, 43 new paths), built and tested cleanly.
Ran R2 harness myself independently: 58 vectors, **48 pass, 10 skip, 0 fail.** Executed four live CLI probes against scratch runbooks.
No product artifact modified in either repository.

**VERDICT: APPROVED.** R1–R5 all substantively resolved. No blocker remains.

All five genuine bugs Ken found during harness construction are correct fixes, ratified in-scope consequences of the R2 requirement.
The 10 skip reasons independently verified: 5 pre-existing ticketed gaps (T-ENUM-FROM-SOURCING, T-ENUM-ROOT-OUTPUTS), 5 corpus defects (each adjudicated against AR-ENUM-1..15, none are evasions).

**Remaining limitations, all ticketed, none blocking:**
- **T-ENUM-ROOT-OUTPUTS** — non-substituted root runbook never evaluates own `outputs:` (S4 site, no engine hook exists)
- **T-ENUM-FROM-SOURCING** — `Input.From` (from: env/prompt/provider) never read by any runtime path
- **T-ENUM-SENSITIVE-DECL** — no first-class sensitivity marker; EnumMeta.Redacted best-effort name-vs-governance.redact proxy, not a guarantee
- **T-ENUM-REPLAY-WIRE** (new) — adapter.go replay branch unreachable today (ScenarioFile never assigned), documented at site

Ken is released. His revision found and fixed five genuine runtime defects that hand-written tests could not surface; conformance harness is now the cheap gate wanted.

**Tess** owns four further corpus amendments (UNICODE-005/PLAN-005/RUNTIME-004/PLAN-003), each keeping intent, count, and expectation; re-run harness after.
**Edith** owns parallel §2a prose fix in `03-schema-vnext.tex` (strike YAML-1.1 aside from AR-ENUM-3 rule 3, add 1.2-core-schema-accurate trap example).

No further gates required. Ready for Cristián.

---

## 2026-08-15 — Dynamic Runbook Includes Feature — COMPLETE (Phase 4 Sessions)

**Status:** Feature implemented, tested, reviewed, approved. Ready for merge.

**Summary:** Runtime-resolved (dynamic) runbook includes in the `gert` Go repo. Catalog-based dynamic include form (`include: {runbook_ref: "${...}", resolve_from: catalog, with: {...}}`) resolving an identity against approved package-export catalogs at execution time — never an arbitrary filesystem path. Driving use case: an ICM orchestrator that takes an incident ID, applies deterministic rules to suggest a TSG, checks whether that TSG exists as a gert runbook, asks the operator to confirm, then dynamically includes it.

**Spawn Manifest — Sessions in order:**

1. **Barbara** (architect, claude-opus-4.6) — authored the binding architecture contract that gated all implementation streams. Issued rulings B-1 through B-21 across multiple arbitration rounds. Served as the code-review gate. Outcome: **APPROVE WITH CONDITIONS** — all 11 user requirements MET, three documentation-only conditions.

2. **Tess** (tester, claude-sonnet-4.6) — authored conformance vectors before implementation existed. Drove real CLI end-to-end. Found 5 defects by exercising the actual binary. Closed corpus at 26 pass / 3 skip / 0 fail. Outcome: corpus closed, all skips permanent with closing rulings.

3. **Ken** (backend, claude-sonnet-4.6) — Stream 1: schema, parser, errkit error taxonomy (DINC-001..013). Fixed DEF-003 and DEF-005. Locked out after DINC-007 reclassification, unable to apply Barbara B-15 himself.

4. **Don** (backend, claude-sonnet-4.6) — Streams 2 & 3: planner, preview/dry-run rendering, executor (`executeDynamic`). Fixed DEF-001 (two visitors bug), applied B-15 on Ken's behalf during lockout, fixed DEF-006 (entry governance never seeded).

5. **David** (backend, claude-sonnet-4.6) — Stream 4: pin recording, replay re-binding, resume-drift detection. Filed two deferred defects (B-18, B-19/B-20). Corrected coordinator's description three times.

**Architecture Rulings — B-1 through B-21 (Complete List):**

All rulings below are from Barbara's binding contract and rulings document, incorporated with full authority:

**B-1** (DINC-001): Rendered Reference Validation — Empty, path-like, or non-ASCII refs rejected before catalog lookup.
**B-2** (DINC-002): Reference Scope — Bare IDs and package-qualified IDs only; no paths, no URIs, no relative components.
**B-3** (DINC-003): Catalog-Only Constraint — Structurally enforced: no filesystem access, no exec.Command, no http.Client on resolution path.
**B-4** (DINC-004): Include Cycle Detection — Tracked via call stack in context; cycle detected at runtime before execution.
**B-5** (DINC-005): Include Depth Limit — Max 8 levels deep; exceeded depth raises DINC-005.
**B-6** (DINC-006): Required Input Validation — Child inputs checked for required fields; missing required raises DINC-006.
**B-7** (DINC-W007): Extra Input Key Warning — Child runbook receives unexpected input keys; emitted as warning, non-fatal.
**B-8** (DINC-008): Input Enum Validation — Input values matched against declared enums; mismatch raises DINC-008.
**B-9** (DINC-009): Output Schema Mismatch — Child output structure validated against child schema; mismatch raises DINC-009.
**B-10** (DINC-010): Child Package Missing — Child's `requires:` entry not in frozen catalog; raises DINC-010.
**B-11** (DINC-011): Child Tool Unresolvable — Child's `toolRefs:` entry not in frozen catalog; raises DINC-011.
**B-12** (DINC-W001): Governance Widening Warning — Parent governance composed with child governance; any widening emits DINC-W001.
**B-13** (DINC-W002): Deprecated Fields Warning — Deprecated field values emit DINC-W002.
**B-14** (DINC-W003): Include Resolution Warning — Reserved for informational includes-resolution warnings.
**B-15** (DINC-W007 reclassification): Errkit sentinel must classify as `DINC-W007`/class `"DINC-W"`, not `DINC-007`/class `"DINC"`. Required pre-ship.
**B-16** (Frozen Catalog Invariant): Resolved package set is locked at plan time. Include execution cannot trigger new package downloads.
**B-17** (On-Not-Found Behavior): `on_not_found: continue` allows runbook to not exist; sets `result.Vars["runbook_found"] = false`; step completes (not failed).
**B-18** (DEFERRED): Static Include Governance Gap — Non-dynamic branch does not enforce composed governance. Deferred due to production risk.
**B-19** (DEFERRED): when: Field Inert — `CollectorField.When` works; `Step.When` and `IncludeConfig.When` are inert. Code fix deferred; documentation applied.
**B-20** (Documentation): when: Remediation — Schema `description` fields added to three inert `when:` entries. Example warning comment added.
**B-21** (Documentation): require_approval Scope — Schema `description` added noting TTY-only enforcement; auto-approves non-interactive.

**Conformance Corpus Status:**
- **Total vectors:** 29 (designed corpus)
- **Pass:** 26
- **Skip:** 3 (permanent, documented)
- **Fail:** 0

**Deferred Defect Records (Open Work):**

**B-18 — Static Include Governance Gap** (filed by David, 2026-08-15)
Non-dynamic branch of `IncludeExecutor.Execute` does not enforce composed governance. Both eager-static and lazy-static affected. Dynamic includes (fixed) unaffected. Deferred due to production risk; requires separate migration with deprecation/flag plan. Full analysis in `.squad/decisions/inbox/defect-static-include-governance-gap.md`.

**B-19 / B-20 — when: Field Inert in Two of Three Definitions** (filed by David)
`CollectorField.When` works; `Step.When` and `IncludeConfig.When` are schema-accepted but inert at runtime. Code fix deferred; documentation remediation applied. Open work tracked separately.

**B-21 — require_approval TTY-Only Enforcement** (ruled by Barbara)
Enforced only in TTY/interactive mode; auto-approves non-interactive runs. Ruled acceptable as designed for v1 MVP. Documented in schema and examples.

**Key Findings from Implementation:**
- DEF-001: CLI preflight crash on every dynamic-include runbook (fixed by Don: two visitors bug)
- DEF-003: Governance composed but never enforced in dynamic path (fixed by Ken: added enforcement at execution boundary)
- DEF-006: Entry runbook governance never seeded (fixed by Don: seed governance at plan time)

**Architecture Review:** Barbara's review gate approved with three documentation-only conditions, all satisfied. All 11 user requirements verified met.

**Inbox Merged:** 13 files totaling ~200KB (barbara-dynamic-include-contract, barbara-dynamic-include-rulings, barbara-dynamic-include-review, tess-dynamic-include-vectors, tess-dynamic-include-open-questions, tess-dynamic-include-e2e, ken-dynamic-include-stream1, ken-dynamic-include-governance, don-dynamic-include-stream2, don-dynamic-include-stream3, david-dynamic-include-stream4, defect-static-include-governance-gap, defect-when-field-not-evaluated)

## Inbox Merged

Files merged: 9

# Architecture Contract: Streamable HTTP MCP Transport

**Author:** Barbara (Lead / Architect)
**Date:** 2026-08-16T00:55:39Z
**Status:** RATIFIED — all implementation streams build against this contract
**Continues from:** B-21 (dynamic-include rulings). New rulings start at B-22.

---

## 0. Preamble — Existing Architecture Confirmed

Before specifying the new work, I confirm the following about the existing codebase (verified by reading the source directly):

**The transport seam already exists.** `pkg/tool.ToolTransport` is:
```go
type ToolTransport interface {
    Invoke(ctx context.Context, def ToolDef, action string, args map[string]any) (*ToolResult, error)
    Close() error
}
```

`DefaultToolRuntime.Invoke` (`internal/tool/runtime.go`) dispatches on `def.Transport` via a switch statement. Persistent transports (JSONRPC, MCP) are pooled in `r.persistent[toolName]`. This is the integration point for the new HTTP transport.

**The stdio MCP transport is self-contained.** `MCPTransport` (`internal/tool/mcp.go`) owns:
- Process lifecycle (`StartProcess`, `ensureStarted`)
- Content-Length framing (`writeMCPMessage`, `readMCPMessage`, `readContentLength`)
- MCP lifecycle (`initialize` → read response → `initialized` notification)
- `tools/call` and `tools/list` dispatch
- Response parsing (`mcpResponse`, `mcpCallResult`, `mcpContent`)

The JSON-RPC logic is welded to stdio pipes. **No refactor of `MCPTransport` is required.** The new HTTP transport implements the same `ToolTransport` interface independently. The two share response-parsing types (extracted to a shared file) but not I/O logic.

**Schema transport config** (`pkg/schema/tool.go`):
```go
type TransportConfig struct {
    Type    Transport         // legacy enum
    Mode    string            // canonical (overrides Type)
    Command string
    Args    []string
    Env     map[string]string
}
```

Currently has no URL or auth fields. These must be added.

---

## 1. Schema Extension

### 1.1 TransportConfig additions

```go
// pkg/schema/tool.go — additions to TransportConfig
type TransportConfig struct {
    // ... existing fields ...
    URL  string      `yaml:"url,omitempty"  json:"url,omitempty"`
    Auth *AuthConfig `yaml:"auth,omitempty" json:"auth,omitempty"`
}

type AuthConfig struct {
    Provider     string   `yaml:"provider"             json:"provider"`
    Scope        string   `yaml:"scope,omitempty"       json:"scope,omitempty"`
    AllowedHosts []string `yaml:"allowed_hosts"         json:"allowed_hosts"`
}
```

### 1.2 New transport constant

```go
// pkg/tool/tool.go
const TransportMCPHTTP TransportType = "mcp-http"
```

Also add to `pkg/schema/tool.go`:
```go
const TransportMCPHTTP Transport = "mcp-http"
```

### 1.3 Validation rules

| Condition | Error |
|---|---|
| `mode: mcp-http` without `url` | Schema validation error |
| `mode: mcp-http` with `command` or `args` | Schema validation error (mutually exclusive with url) |
| `url` does not start with `https://` | `MCP-001` (HTTPS required for remote endpoints) |
| `auth` configured without `allowed_hosts` | `MCP-010` (fatal: allowed_hosts required when auth present) |
| `url` host not in `allowed_hosts` | `MCP-011` (fatal: host mismatch caught at validation) |
| `auth.provider` is not a recognized provider name | `MCP-002` (unknown auth provider) |

**B-22 Ruling — HTTPS only:** Remote MCP endpoints MUST use HTTPS. Plain HTTP is rejected at validation time. No `--allow-insecure` escape hatch in this iteration. Rationale: MCP tool calls carry operator credentials and can mutate production systems (IcM ticket actions). Allowing HTTP would let a network-position attacker intercept bearer tokens. Localhost exceptions are not needed because a local MCP server would use `mode: mcp` (stdio).

### 1.4 Target YAML shape (matches user requirement exactly)

```yaml
transport:
  mode: mcp-http
  url: https://icm-mcp-prod.azure-api.net/v1/
  auth:
    provider: azure-cli
    scope: api://icmmcpapi-prod/mcp.tools
    allowed_hosts:
      - icm-mcp-prod.azure-api.net
```

---

## 2. HTTP MCP Transport Implementation

### 2.1 Type and location

```go
// internal/tool/mcp_http.go
type MCPHTTPTransport struct {
    mu          sync.Mutex
    url         string
    auth        AuthProvider   // §4
    sessionID   string         // from Mcp-Session-Id response header
    initialized bool
    httpClient  *http.Client
}
```

Implements `pkg/tool.ToolTransport`.

### 2.2 Lifecycle — initialize → tools/call

On first `Invoke` call (lazy, same pattern as stdio):

1. **Send `initialize` request** — HTTP POST to `url` with:
   - Body: JSON-RPC 2.0 `initialize` message (same params as stdio: `protocolVersion`, `capabilities`, `clientInfo`)
   - Header: `Content-Type: application/json`
   - Header: `Accept: application/json, text/event-stream`
   - Header: `MCP-Protocol-Version: 2025-03-26`
   - Header: `Authorization: Bearer <token>` (from auth provider, §4)

2. **Parse response** — detect Content-Type:
   - `application/json` → parse body directly as JSON-RPC response
   - `text/event-stream` → parse SSE stream, extract `data:` lines, assemble JSON-RPC response (§3)

3. **Capture `Mcp-Session-Id`** from response headers. Store in `t.sessionID`.

4. **Send `notifications/initialized`** — HTTP POST to `url` with:
   - Body: JSON-RPC 2.0 notification (no `id` field)
   - Header: `Mcp-Session-Id: <captured value>` (if server provided one)
   - No response body expected (HTTP 204 or 202 accepted)

5. Mark `t.initialized = true`.

Subsequent `Invoke` / `ListTools` calls:
- Send `tools/call` or `tools/list` as HTTP POST
- Include `Mcp-Session-Id` header if present
- Include `Authorization` header (refreshed if expired, §4.3)
- Include `MCP-Protocol-Version: 2025-03-26`
- Parse response (JSON or SSE, §3)
- Convert to `*ToolResult` exactly as stdio does

### 2.3 Protocol version

**B-23 Ruling — Version pinning:** gert advertises `MCP-Protocol-Version: 2025-03-26` (the current Streamable HTTP spec). If the server returns an HTTP 4xx with a body indicating version mismatch, emit `MCP-003` (fatal). If the server simply ignores the header and responds successfully, proceed — interoperability with older servers that don't enforce version headers is acceptable.

### 2.4 Session ID semantics

| Scenario | Behavior |
|---|---|
| Server returns `Mcp-Session-Id` on initialize | Store; send on all subsequent requests |
| Server omits `Mcp-Session-Id` | Proceed without it; do not fail |
| Server returns HTTP 404 on a request with session ID | Session expired — re-initialize (once) then retry the failed request. If re-init also fails, emit `MCP-004` (fatal). |

The session ID lives on the `MCPHTTPTransport` instance, which is pooled per tool name in `DefaultToolRuntime.persistent`. This means the session survives across multiple tool calls within a single run — correct behavior.

### 2.5 Close

`Close()` sends a JSON-RPC `shutdown` notification (best-effort, no response expected) if a session is active, then drops the HTTP client. No process to kill.

---

## 3. Transport Framing — SSE Handling

### 3.1 Content-Type detection

On every HTTP response:
- If `Content-Type` starts with `application/json` → read full body, parse as single JSON-RPC response.
- If `Content-Type` starts with `text/event-stream` → parse as SSE stream (§3.2).
- Otherwise → `MCP-005` (unexpected content type).

### 3.2 SSE parsing

SSE events consist of lines. The parser handles:
- `data: <json>` — append to current event's data buffer (with `\n` between multi-line data)
- Empty line — event boundary; dispatch accumulated data
- `event:` — ignored (MCP uses only the default event type)
- `id:` — ignored (MCP does not use SSE last-event-id)
- Lines starting with `:` — comments, ignored

For a given JSON-RPC request with `id: N`, the SSE stream may contain:
- Zero or more JSON-RPC notifications (no `id` field) — these are progress/log messages; log them but do not treat as the response.
- Exactly one JSON-RPC response with `id: N` — this is the terminal response. Once received, stop reading the stream.

If the stream closes (HTTP connection ends) before a response with matching `id` is received → `MCP-006` (incomplete SSE stream).

### 3.3 Timeout

HTTP requests use the context deadline from `ctx`. If the context has no deadline, a default 120-second timeout is applied. This is configurable via a future `timeout:` field on `TransportConfig` (not in this iteration — hardcode 120s).

---

## 4. Authentication Provider

### 4.1 Interface

```go
// internal/tool/auth.go
type AuthProvider interface {
    // Token returns a valid bearer token. Implementations handle
    // acquisition, caching, and refresh.
    Token(ctx context.Context) (string, error)
}
```

### 4.2 `azure-cli` provider

```go
// internal/tool/auth_azurecli.go
type AzureCLIAuthProvider struct {
    scope string
    // cached token + expiry
    mu       sync.Mutex
    token    string
    expiry   time.Time
}
```

Acquires a token by executing:
```
az account get-access-token --scope <scope> --query accessToken -o tsv
```

**B-24 Ruling — No credential in YAML, trace, log, or error:**
- The `scope` field is the only auth-related value that appears in YAML. It is a resource identifier, not a credential.
- The acquired bearer token MUST NOT appear in: YAML files, trace events, log output, error messages, step output, `ToolResult.Stdout`, or diagnostic dumps.
- If auth fails, the error message reports the failure reason (e.g., "az CLI not authenticated") but NEVER includes the token value.
- The `Authorization` header value is never logged by gert's HTTP client (use a non-logging transport or strip auth headers before any debug logging).

### 4.3 Token caching and refresh

- On first `Token()` call, run `az account get-access-token`.
- Cache the token and parse its expiry from the JWT `exp` claim (or from `az`'s `expiresOn` field).
- On subsequent calls, return cached token if `now + 5min < expiry`.
- If within 5min of expiry, re-acquire proactively.
- If `az` command fails, emit `MCP-007` (auth failure).

### 4.4 Future providers

The `AuthConfig.Provider` field is a string, not an enum. Recognized values in this iteration:
- `"azure-cli"` — described above

Future values (NOT in scope, listed for schema stability):
- `"env"` — read token from an env var named in a `variable:` field
- `"managed-identity"` — Azure managed identity (no CLI dependency)
- `"device-code"` — interactive device-code flow

The schema shape (`provider` + `scope` + future fields like `variable`) is designed to accommodate these without breaking changes.

### 4.5 No auth configured

If `transport.auth` is nil/omitted, no `Authorization` header is sent. This supports MCP servers that use other auth mechanisms (API keys in custom headers via `env`, network-level auth, etc.) — but those mechanisms are NOT part of this feature. Omitting auth simply means no bearer token.

---

## 5. Error Taxonomy

New error class: `"MCP"` (fatal).

| Code | Condition | Fatal? | Message template |
|---|---|---|---|
| `MCP-001` | URL is not HTTPS | Fatal | `mcp-http: url %q must use https://` |
| `MCP-002` | Unknown auth provider | Fatal | `mcp-http: unknown auth provider %q` |
| `MCP-003` | Protocol version mismatch / rejected | Fatal | `mcp-http: server rejected protocol version (HTTP %d)` |
| `MCP-004` | Session expired and re-initialize failed | Fatal | `mcp-http: session expired; re-initialization failed: %v` |
| `MCP-005` | Unexpected response content-type | Fatal | `mcp-http: unexpected content-type %q (expected application/json or text/event-stream)` |
| `MCP-006` | SSE stream closed before response received | Fatal | `mcp-http: SSE stream closed without response for request id %d` |
| `MCP-007` | Auth token acquisition failed | Fatal | `mcp-http: failed to acquire auth token: %v` |
| `MCP-008` | HTTP transport error (connection refused, TLS failure, timeout) | Fatal | `mcp-http: transport error: %v` |
| `MCP-009` | JSON-RPC error response from server | Fatal | `mcp-http: server error %d: %s` |
| `MCP-010` | `auth` configured without `allowed_hosts` | Fatal | `mcp-http: auth.allowed_hosts is required when auth is configured` |
| `MCP-011` | `url` host not in `auth.allowed_hosts` | Fatal | `mcp-http: url host %q is not in auth.allowed_hosts` |
| `MCP-W001` | Token not attached (host not in allowed_hosts — runtime defensive) | Warning | `mcp-http: auth token not attached — host %q is not in allowed_hosts` |

**Requirement 4 guarantee:** `MCP-009` (JSON-RPC error from server) wraps the server's error code and message. A tool-level error (i.e., `result.isError == true` in the MCP response) is returned as a failed `ToolResult` with non-zero exit code — **exactly as stdio MCP does today** (see `mcp.go` line 89: `"mcp tool error: %s"`). The `ToolResult` shape is identical regardless of transport. The runbook author cannot distinguish stdio from HTTP tool errors.

---

## 6. Security Posture

### 6.1 URL allow-listing

**B-25 Ruling — No URL allow-list required.** Unlike dynamic includes (where the catalog is the trust boundary), tool definitions are authored by the same team that authors the runbook. A `.tool.yaml` declaring `url: https://evil.com` is the same trust level as one declaring `command: /usr/bin/evil`. Both are authored artifacts subject to code review and package governance. There is no runtime-resolved URL — the URL is static in the YAML. Adding an allow-list would be security theater without a trust boundary to enforce.

**Contrast with dynamic includes:** Dynamic includes resolve an *identity* at runtime against a frozen catalog — the catalog IS the allow-list. Tool definitions are statically declared — the .tool.yaml IS the authored source. Different trust models, different controls.

### 6.2 TLS verification

Standard Go `http.DefaultTransport` TLS verification applies. No `InsecureSkipVerify`. No custom CA configuration in this iteration (can be added later via `tls:` config on `TransportConfig`).

### 6.3 Token redaction

Per B-24: bearer tokens are never written to any persistent or observable surface. The HTTP client used by `MCPHTTPTransport` must NOT be wrapped in a logging/tracing transport that captures request headers. If gert adds HTTP debug logging in the future, the `Authorization` header must be redacted.

---

## 7. Governance

**B-26 Ruling — Remote tool calls ARE subject to governance.** The governance evaluator receives the tool definition and step context regardless of transport. `deny_commands` does not apply (there is no "command" for HTTP — `deny_commands` gates CLI shell invocations). However:

- `require_approval` applies normally (the approval gate fires before any tool invocation, per `internal/executor/tool.go`).
- `ToolGovernance.RequiresCapabilities` and `AllowedEnvironments` apply normally (checked by the governance evaluator before the executor dispatches).
- Future `deny_tools` or tool-level allow-list governance would apply here. Not in scope for this feature.

The key insight: governance gates at the executor level (before `ToolRuntime.Invoke` is called), not at the transport level. Adding a new transport does not bypass governance because governance fires upstream.

---

## 8. Runtime Wiring

### 8.1 `DefaultToolRuntime.Invoke` extension

Add a case to the switch:
```go
case toolpkg.TransportMCPHTTP:
    return r.invokePersistent(ctx, toolName, *def, action, args, func() toolpkg.ToolTransport {
        return NewMCPHTTPTransport(def.URL, def.Auth)
    })
```

### 8.2 `ToolDef` extension

`pkg/tool.ToolDef` gains:
```go
URL  string         // from TransportConfig.URL
Auth *schema.AuthConfig // from TransportConfig.Auth
```

The existing `internal/tool.RuntimeToolDef` conversion function (which builds `pkg/tool.ToolDef` from `schema.ToolDef`) must copy these fields.

### 8.3 `MCPHTTPTransport.ListTools`

Same interface as `MCPTransport.ListTools`:
```go
func (t *MCPHTTPTransport) ListTools(ctx context.Context, def ToolDef) ([]map[string]any, error)
```

Called by the tool registry's discovery mechanism when a tool is declared with `mode: mcp-http`. The response shape is identical to stdio MCP's `tools/list` result.

---

## 9. Shared Response Types

Extract from `internal/tool/mcp.go` into `internal/tool/mcp_types.go`:
```go
type mcpContent struct { ... }
type mcpCallResult struct { ... }
type mcpResponse struct { ... }
```

Both `MCPTransport` (stdio) and `MCPHTTPTransport` (HTTP) import and use these types for response parsing. This ensures Requirement 4 — identical output shape regardless of transport.

---

## 10. Scope Boundary — What Is NOT In Scope

| Excluded | Rationale |
|---|---|
| Refactoring `MCPTransport` (stdio) | Works as-is; HTTP is a new parallel implementation |
| OAuth device-code flow | Future auth provider |
| Managed identity auth | Future auth provider |
| Custom CA/TLS config | Future extension |
| WebSocket transport | MCP spec supports it; not needed for IcM |
| Tool discovery from remote `tools/list` at catalog-build time | Tools are statically declared in .tool.yaml; remote list is a registry feature |
| `deny_tools` governance | Future governance extension |
| Streaming tool output (SSE progress → live TUI) | Future UX enhancement; responses are buffered |
| `timeout:` field on TransportConfig | Hardcode 120s; add field later |
| Connection pooling / HTTP/2 multiplexing | Go's `http.Client` handles this by default |
| Retry on transient HTTP errors (5xx) | Fail on first error; retry logic is a future enhancement |

---

## 11. Proposed Stream Breakdown

| Stream | Owner | Scope |
|---|---|---|
| **Stream A — Schema + Validation** | Ken | `TransportConfig` extensions, `AuthConfig` type, `TransportMCPHTTP` constant, schema validation (MCP-001, MCP-002), `ClassForCode` extension |
| **Stream B — HTTP Transport + SSE** | Don | `MCPHTTPTransport`, SSE parser, lifecycle (§2), shared types (§9), wiring into `DefaultToolRuntime` (§8) |
| **Stream C — Auth Provider** | David | `AuthProvider` interface, `AzureCLIAuthProvider`, token caching, `MCP-007`, redaction guarantees (B-24) |
| **Stream D — Integration + Tests** | Tess | End-to-end wiring test, mock MCP HTTP server, test vectors for init/session/auth/tool-call/SSE/errors |

Dependencies: B depends on A (needs schema types) and C (needs AuthProvider interface). D depends on all.

---

## Appendix — MCP Protocol Reference (2025-03-26 Streamable HTTP)

For implementers. The MCP Streamable HTTP transport (replacing the deprecated HTTP+SSE transport):

- Client sends JSON-RPC messages as HTTP POST to the server's endpoint URL.
- Server responds with either `application/json` (single response) or `text/event-stream` (SSE stream containing the response plus optional notifications).
- `Mcp-Session-Id` header: server MAY issue on any response; client MUST echo on subsequent requests.
- `MCP-Protocol-Version` header: client MUST send; server validates.
- Notifications (no `id`) sent via POST; server replies with 202/204.
- Server MAY issue HTTP 404 to indicate session expired; client should re-initialize.


# MCP HTTP Transport — Final Architecture Review

**Reviewer:** Barbara (Lead / Architect)
**Date:** 2026-08-15T19:18:27-07:00
**Verdict:** ✅ **APPROVE**

---

## Five Requirements — Verdict

| # | Requirement | Verdict | Evidence |
|---|---|---|---|
| 1 | Add `transport.mode: mcp-http` with a `url` field | **MET** | `TransportMCPHTTP = "mcp-http"` in both `pkg/tool/tool.go:17` and `pkg/schema/tool.go:392`. `TransportConfig.URL` added at `pkg/schema/tool.go:291`. `ValidateTransportConfig` enforces url-required + HTTPS-only + mutual exclusion with command/args. |
| 2 | MCP HTTP lifecycle: initialize, Mcp-Session-Id, notifications/initialized, MCP-Protocol-Version, SSE | **MET** | `mcp_http.go:ensureInitialized` → `initialize()`: sends initialize with `protocolVersion: "2025-03-26"`, inspects response for errors (unlike stdio — B-30 not replicated), captures `Mcp-Session-Id` from headers, sends `notifications/initialized`. `MCP-Protocol-Version` header set on every request. SSE parser (`parseSSEResponse`) handles `data:` lines, multi-line events, comment heartbeats, and ID correlation. Session-expired 404 → re-initialize → retry (§2.4). |
| 3 | Auth provider: Azure CLI bearer token, credentials never in YAML | **MET** | `AzureCLIAuthProvider` in `auth_azurecli.go`: runs `az account get-access-token --scope <scope> -o json`, parses response, caches with expiry, proactive refresh at 5min buffer, `Invalidate()` on 401. Token never appears in errors (classified error messages reference "az CLI" failure reasons, never token value). `AuthConfig.Scope` is the only auth-related value in YAML. B-24 enforced. |
| 4 | Dispatch tools/list and tools/call exactly as stdio, preserving typed outputs | **MET** | Both transports share `mcp_types.go` (`mcpResponse`, `mcpCallResult`, `mcpContent`, `callResult()`). Decode path in `toolResult()`: `content[0].Text` → `Stdout`; JSON parse → `Output` map. Same as stdio (mcp.go lines 69-78). Error message format: `"mcp tool error: %s"` — identical. **One minor note:** HTTP returns a `ToolResult{ExitCode:1, Stderr:text}` alongside the error on tool-level `IsError`; stdio returns `nil, error`. The error is identical; the ToolResult difference is invisible to callers (who check `err != nil` first). Not a requirement failure — the observable behavior from a runbook author's perspective is indistinguishable. |
| 5 | Schema/runtime wiring, validation, and tests | **MET** | Schema: `TransportConfig` extended with `URL` and `Auth *AuthConfig` (with `AllowedHosts`). Runtime: `DefaultToolRuntime.Invoke` has `case TransportMCPHTTP` dispatching to `invokePersistent` → `NewMCPHTTPTransport`. Validation: `ValidateTransportConfig` checks MCP-001/002/010/011 at scan time. Tests: `mcp_http_test.go` + `mcp_http_tess_test.go` + `validate_transport_test.go` — 27/27 passing per verified report. |

---

## B-32 Realisation — FULLY BUILT AS RULED

| Control | Implementation | Verified |
|---|---|---|
| `auth:` requires `allowed_hosts` (MCP-010) | `ValidateTransportConfig`: if `auth != nil && len(AllowedHosts) == 0` → MCP-010 fatal | ✅ |
| URL host in allowed_hosts static check (MCP-011) | `ValidateTransportConfig`: parses URL, extracts `u.Hostname()`, lowercase exact match against list | ✅ |
| Runtime host mismatch fatal (MCP-012) | `TokenGate.AttachToken`: `hostAllowed()` → `errkit.New("MCP-012", ...)` | ✅ Confirmed by source read |
| Redirects refused on authenticated requests (MCP-013) | `NewMCPHTTPTransport`: `client.CheckRedirect = func(...) { return http.ErrUseLastResponse }` when gate != nil. `send()` detects 3xx → `errkit.New("MCP-013", ...)` naming the Location. | ✅ |
| Host matching: `u.Hostname()`, lowercase, exact, no wildcard | Both `ValidateAuthConfig` and `TokenGate.hostAllowed` use identical logic: `strings.ToLower(u.Hostname())` vs `strings.ToLower(entry)` | ✅ One definition, no drift |

---

## B-27: mcp/authAttached Trace Event — NOT DEAD CODE

`TokenGate.AttachToken` emits via `trace.EmitterFromContext(ctx)` on every successful token attachment. The emitter is the same one wired through `engine.go:569` → TraceWriter.Append — the same path used by all other trace events (include/resolved, step/started, etc.). It fires on every authenticated request, which means every `tools/call` and `tools/list` invocation on an authenticated MCP HTTP tool. This is not dead code — it is executed on the hot path of the feature's primary use case.

---

## B-24: Token Escape Surface Audit

| Surface | Safe? | Evidence |
|---|---|---|
| Error messages | ✅ | `classifyAzError` never includes token; MCP-007/012 errors reference scope, host, stderr — never token |
| Trace events | ✅ | `mcp/authAttached` payload: `{url_host, scope}` only |
| ToolResult.Stdout/Stderr/Output | ✅ | Token never enters ToolResult — it exists only in the `Authorization` header |
| HTTP debug logging | ✅ | No HTTP debug/trace transport installed; `http.Client{}` has no `Transport` override that would log headers |
| Panic stack | ⚠️ Acceptable | If `AttachToken` panics after `provider.Token()` returns, the token is on the stack. This is inherent to any in-memory secret and is not a design flaw — panic stacks are not a normal observability surface. |

**Verdict: B-24 is satisfied.** No credential material reaches any persistent or normal-observability surface.

---

## B-31: Documentation Adequacy

Go doc comments on `AuthConfig` (`pkg/schema/tool.go:298-365`) are extensive — they explain the replay threat, the `allowed_hosts` mechanism, and exactly what happens on mismatch. The wildcard non-support is documented in `TokenGate.AttachToken`'s doc comment. An author who writes `*.azure-api.net` gets MCP-012 with a message naming the host and referencing `allowed_hosts`.

Missing: an example `.tool.yaml` file in `examples/`. This is a **nice-to-have**, not a blocker — the doc comments and the schema types are sufficient for an engineer reading code or IDE hover-docs. I recommend adding one before broader adoption but do not condition approval on it.

---

## B-30: Stdio ensureStarted Defect — STILL CORRECT TO DEFER

The HTTP path deliberately does NOT replicate it (confirmed: `initialize()` checks `initResp.Error` before marking `t.initialized = true`). The stdio defect is tracked. The two paths are now asymmetric in a good way — the new code is correct, the old code has a known defect that will be fixed in a dedicated cleanup pass. Deferral remains the right call.

---

## Cross-Feature Question: Dynamic Include + MCP HTTP + Token Replay

A runtime-selected child runbook can declare an `mcp-http` tool with `auth:`. B-32's `allowed_hosts` is the control. Is it sufficient?

**Yes.** The chain of controls is:

1. **Frozen catalog** — the child runbook must come from an approved package. An external attacker cannot inject a tool definition.
2. **MCP-011 (static, at tool scan time)** — the declared URL's host must be in `allowed_hosts`. A package author who sets `url: https://evil.com` must ALSO set `allowed_hosts: [evil.com]` — the mismatch is caught structurally.
3. **The residual threat** — a compromised package author sets BOTH `url: https://evil.com` AND `allowed_hosts: [evil.com]`. B-32 cannot prevent this because the attacker controls the authored artifact. But at this point, the attacker can also set `url: https://icm-mcp-prod.azure-api.net` and issue malicious commands directly — a strictly more powerful attack that no host allow-list prevents. The allow-list stops *accidental* token leakage and *partial* compromise (attacker can modify URL but not allowed_hosts, or vice versa); it cannot stop a fully compromised author.

**The frozen catalog is the real trust boundary.** `allowed_hosts` is defense-in-depth against partial compromise or carelessness. Together they are sufficient.

---

## Deferred Items Confirmed

| Item | Status | Still correct to defer? |
|---|---|---|
| B-18 (static-include governance gap) | Tracked | ✅ |
| B-19/B-20 (`when:` inert) | Tracked, schema remediation required | ✅ |
| B-30 (stdio ensureStarted) | Tracked | ✅ |
| DEF-002 (DINC-009 unreachable) | B-16, permanent | ✅ |

---

## Final Verdict

**✅ APPROVED.** No conditions. The implementation is complete, correct, and faithful to the contract and all rulings through B-33. The security posture (B-32 host restriction, B-33 fatal enforcement + no redirects, B-24 token redaction) is structurally sound. The five requirements are met. The cross-feature interaction with dynamic includes is adequately controlled by the frozen catalog + allowed_hosts defense-in-depth.


# MCP HTTP — Binding Rulings B-22 through B-26

**Author:** Barbara (Lead / Architect)
**Date:** 2026-08-16T00:55:39Z
**Status:** RATIFIED — implementers build directly against these rulings

---

## B-22 — HTTPS only for remote MCP endpoints

**Date:** 2026-08-16T00:55:39Z

**Ruling:** Remote MCP endpoints MUST use HTTPS. Plain HTTP URLs are rejected at validation time with `MCP-001`. No `--allow-insecure` flag in this iteration.

**Rationale:** MCP tool calls carry bearer tokens and can mutate production systems (e.g., IcM ticket remediation actions). Allowing HTTP would let a network-position attacker intercept credentials. Localhost/stdio servers use `mode: mcp` (subprocess) and are unaffected.

---

## B-23 — Protocol version pinning at 2025-03-26

**Date:** 2026-08-16T00:55:39Z

**Ruling:** gert advertises `MCP-Protocol-Version: 2025-03-26` on all HTTP MCP requests. If the server responds with an HTTP error indicating version rejection, emit `MCP-003` (fatal). If the server ignores the header and responds successfully, proceed without error.

**Rationale:** The 2025-03-26 spec defines Streamable HTTP (replacing the deprecated SSE transport). Pinning to this version is forward-looking — it's the current stable spec. Graceful degradation for servers that ignore the header ensures interop with older implementations.

---

## B-24 — No credential material in YAML, traces, logs, or errors

**Date:** 2026-08-16T00:55:39Z

**Ruling:** Bearer tokens acquired by any auth provider MUST NOT appear in: YAML files, trace events, log output, error messages, `ToolResult` fields, step output, or any diagnostic dump. The `scope` field is a resource identifier (not a credential) and may appear in diagnostics.

**Rationale:** Tokens are short-lived secrets. Any persistence or observability surface that captures them creates a credential leakage vector. The only place a token may exist is in-memory within the `AuthProvider` and in the outgoing HTTP `Authorization` header (which must not be captured by debug logging).

---

## B-25 — No URL allow-list required

**Date:** 2026-08-16T00:55:39Z

**Ruling:** Any HTTPS URL is acceptable for `transport.url`. No allow-list, registry, or pre-approval mechanism is required.

**Rationale:** Tool definitions are authored artifacts (.tool.yaml files) subject to the same code-review and package-governance controls as any other authored configuration. The URL is static — not runtime-resolved. Contrast with dynamic includes, where the resolved identity is runtime-variable and the catalog serves as the allow-list. Different trust models require different controls. Adding an allow-list here would be security theater: the same author who can set a malicious URL can also set a malicious `command:`.

---

## B-26 — Remote tool calls subject to governance

**Date:** 2026-08-16T00:55:39Z

**Ruling:** Governance fires at the executor level (before `ToolRuntime.Invoke`), not at the transport level. Adding a new transport does not bypass governance. Specifically: `require_approval`, `RequiresCapabilities`, and `AllowedEnvironments` apply to MCP HTTP tools identically to stdio tools. `deny_commands` does not apply (there is no shell command to deny).

**Rationale:** The tool executor's governance pre-flight (`internal/executor/tool.go`) checks all policies before invoking the tool runtime. This is transport-agnostic by design — the executor doesn't know or care which transport will be used. The new transport plugs in below this gate.

---

## B-27 — B-25 narrowed: audience-scoped tokens are sufficient; no host allow-list needed

**Date:** 2026-08-15T18:03:00-07:00

**Amends:** B-25

**Challenge:** A dynamically-included child runbook (resolved at runtime from the frozen catalog) can declare an MCP HTTP tool pointing to any HTTPS host. David's auth provider attaches a live Azure bearer token. The operator running the parent never reviewed the child's URL. The URL is "authored" but not "reviewed by the person accepting the risk." This is a credential-forwarding-to-attacker-chosen-host risk — the gap is not the URL itself but the **combination of URL + attached credential**.

**Revised Ruling:** B-25 is **narrowed** as follows:

**Unauthenticated MCP HTTP requests:** B-25 stands unchanged. Any HTTPS URL is acceptable. No allow-list. The request carries no credential — SSRF risk is minimal.

**Authenticated MCP HTTP requests (where `transport.auth` is configured):** No host allow-list is required, because the existing mitigations are structural and sufficient:

1. **Frozen catalog** — child runbooks come only from approved packages. An arbitrary third party cannot inject a tool URL.
2. **Audience-scoped tokens** — a token acquired with `scope: api://icmmcpapi-prod/mcp.tools` carries that audience in its claims. Any properly configured Azure AD resource server that is NOT the intended audience rejects the token server-side. An attacker-chosen host receives a token it cannot validate (wrong audience) and gains nothing useful.
3. **HTTPS** — prevents interception in transit.
4. **Short-lived** — Azure CLI tokens expire in ~1 hour.

The threat model is: a compromised or careless package author places a malicious URL in a child runbook's tool definition. The frozen catalog prevents injection by outsiders. Against an insider (compromised package author), the audience-scoped token provides defense-in-depth — the token is useless at hosts that aren't the declared resource.

**One mandatory safeguard:** The auth provider MUST emit a trace event `mcp/authAttached` with payload `{url_host: <host-portion-only>, scope: <scope>}` (NO token value) on every authenticated request. This enables audit detection of scope-vs-host mismatch without adding a runtime gate.

**Implementation note for David:** Emit this trace event inside `MCPHTTPTransport` immediately before sending an authenticated request. The event is informational (non-gating, non-fatal). If someone declares `scope: api://icmmcpapi-prod/mcp.tools` but `url: https://evil.com`, the trace shows the discrepancy for post-hoc audit.

**Why no allow-list:** An allow-list would require a new configuration surface (where is the list? who maintains it? how does it interact with package catalogs?). The existing mitigations (catalog trust + audience-scoped tokens) close the gap structurally without adding operational burden. If a future threat model shows audience validation is insufficient (e.g., a server that accepts any audience), we can add a host allow-list then — but that would indicate a broken resource server, not a broken gert design.

---

## B-28 — Stdio env-var credentials are a pre-existing surface; B-24 does not retroactively cover them

**Date:** 2026-08-15T18:03:00-07:00

**Question:** Does B-24's redaction mandate extend to stdio MCP's `transport.env` map, which today can carry secrets (e.g., `API_KEY=xxx`) passed to the subprocess via `mergeEnv`?

**Ruling:** **No. B-24 applies exclusively to the new HTTP MCP auth provider.** The stdio env-var path is a pre-existing surface with different characteristics:

1. `transport.env` values are placed in YAML by the operator — they are authored, not runtime-acquired. B-24 exists specifically because the HTTP auth provider acquires a credential at runtime that the operator never typed into any file.

2. Env values are passed to a local subprocess and never traverse a network from gert itself. Whether the subprocess uses them over a network is outside gert's control.

3. These values are NOT currently in traces or logs (only in YAML and process memory). That is acceptable status-quo behavior.

**Action:** Documentation-only. Note that `transport.env` values in `.tool.yaml` should be treated as sensitive by operators (use CI vault injection or env-var indirection rather than literal secrets in committed YAML). No code change. Not a defect — a usage guidance gap.

**Not in scope:** Secret-reference resolution in `transport.env` (e.g., vault references). That is a future platform feature.

---

## B-29 — Protocol version divergence: stdio stays at 2024-11-05; HTTP uses 2025-03-26; this is correct

**Date:** 2026-08-15T18:04:00-07:00

**Question:** Stdio MCP advertises `protocolVersion: "2024-11-05"`. HTTP MCP advertises `MCP-Protocol-Version: 2025-03-26`. Should they match?

**Ruling:** They MUST NOT match. They are different protocol revisions for different transports.

- `2024-11-05` is the MCP version that defines the stdio/Content-Length framing model. The existing `MCPTransport` was built against it and interoperates with stdio MCP servers implementing that version. Changing it risks breaking compatibility with existing servers that validate the version field.
- `2025-03-26` is the MCP version that defines Streamable HTTP (POST + SSE responses, `Mcp-Session-Id` header). HTTP MCP servers expect this version in the `MCP-Protocol-Version` header.

These are not two implementations of the same spec at different versions — they are two different transport specifications. The version field means "I speak this protocol," and the two transports speak different protocols.

**Documentation:** No operator-facing documentation is needed beyond the tool schema itself (`mode: mcp` vs `mode: mcp-http`). An operator never chooses between transports for the same server — a server is either stdio or HTTP. The version is an implementation detail that follows from the transport choice.

---

## B-30 — Stdio `ensureStarted` defect: file and defer

**Date:** 2026-08-15T18:04:00-07:00

**Question:** Stdio `ensureStarted` sets `initialized = true` without checking whether the `initialize` response is an error. Fix now or defer?

**Ruling:** **Defer.** File as a defect. Do NOT fix in this feature.

**Rationale:**
1. This is a pre-existing defect in the stdio path. It has existed since `MCPTransport` was written and affects only stdio MCP tools, which are working in production (presumably with servers that don't reject initialization).
2. Fixing it means modifying `internal/tool/mcp.go` — the existing stdio transport — as part of a feature that is supposed to ADD a new transport without touching the old one. That violates the scope boundary.
3. Don's HTTP transport MUST NOT replicate it (already communicated — his `initialize` path inspects the response and emits MCP-003 on failure).

**Accumulation acknowledgment:** This is the third deferred pre-existing defect alongside B-18 (static-include governance gap) and B-19/B-20 (inert `when:`). I accept this accumulation deliberately. Each has the same shape: a real defect, adjacent to the stream, with regression risk if fixed as a side-effect. They are tracked, not forgotten. When this feature ships, a cleanup pass addressing all three should be prioritized.

**File to:** `.squad/decisions/inbox/defect-stdio-mcp-initialize-unchecked.md` with location `internal/tool/mcp.go:ensureStarted`, the fact that `initialized = true` is set unconditionally, and that the response body is discarded without error checking.

---

## B-31 — Go-only validation is acceptable; author-facing documentation lives in Go doc comments and the tool spec section

**Date:** 2026-08-15T18:04:00-07:00

**Question:** No JSON Schema governs `.tool.yaml`. B-20's schema-description remediation pattern cannot apply here. Is Go-only validation acceptable, and where does documentation live?

**Ruling:** **Go-only validation is acceptable.** This is the existing pattern for ALL tool validation and changing it is out of scope.

The absence of a `.tool.yaml` JSON Schema is a pre-existing architectural choice (noted by Ken: "No JSON Schema governs .tool.yaml" — C3 in the existing code comments). Tool definitions are validated by `internal/tool.ParseToolFile` and the schema types in `pkg/schema/tool.go`. This works and is well-tested.

**Author-facing documentation for `mcp-http` and `auth:` lives in:**
1. **Go doc comments on `TransportConfig`, `AuthConfig`** — these are the source of truth for any future generated documentation.
2. **`design/gert/sections/06-tool-runtime.tex`** — the existing tool runtime spec section. Ken or the implementer of Stream A should add a subsection for `mode: mcp-http` with the transport fields and auth provider shape.
3. **A `.tool.yaml` example** in `examples/` demonstrating the MCP HTTP configuration.

B-20's pattern (schema `description` fields) was appropriate because a JSON Schema existed and IDE completions were actively misleading users. Here, no schema exists, so no misleading completions are generated. The documentation surface is the spec section and examples — standard for this codebase.

---

## B-32 — B-25/B-27 revised: token attachment restricted to declared hosts (replay risk accepted as real)

**Date:** 2026-08-15T18:06:00-07:00

**Supersedes:** B-27's conclusion that "audience-scoped tokens are structurally limited" is **withdrawn as technically incorrect.** The revised ruling below replaces B-25/B-27's reasoning on authenticated requests.

**The corrected threat model:** A bearer token sent to an attacker-controlled host can be **replayed** against the legitimate audience (the real IcM endpoint) for the token's lifetime. Audience restriction constrains which server will *accept* the token, not who may *present* it. Possession is authorization. Token exfiltration to a malicious host IS a compromise regardless of audience scoping — this is the confused-deputy / token-replay pattern.

**Revised Ruling:** Token attachment is restricted to explicitly declared hosts.

### Mechanism

`AuthConfig` gains an `allowed_hosts` field:

```yaml
transport:
  mode: mcp-http
  url: https://icm-mcp-prod.azure-api.net/v1/
  auth:
    provider: azure-cli
    scope: api://icmmcpapi-prod/mcp.tools
    allowed_hosts:
      - icm-mcp-prod.azure-api.net
```

**Enforcement rule:** Before attaching an `Authorization` header, the auth provider checks that the request URL's host (scheme + hostname + port) matches an entry in `auth.allowed_hosts`. If no match, the request is sent **without** the `Authorization` header and a `MCP-W001` warning is emitted (new warning-class code): `"mcp-http: auth token not attached — host %q is not in allowed_hosts"`.

**Validation rules:**
- If `auth` is configured and `allowed_hosts` is empty or omitted → `MCP-010` (fatal): `"mcp-http: auth.allowed_hosts is required when auth is configured"`. Rationale: forcing explicit host declaration is the entire point of this control.
- If `url`'s host is not in `allowed_hosts` at validation time → `MCP-011` (fatal): `"mcp-http: url host %q is not in auth.allowed_hosts"`. Catches the misconfiguration statically rather than at runtime.

### Why this is the right narrowing

1. **Closes the replay vector.** A compromised package author who declares `url: https://evil.com` cannot also make the token go there, because `allowed_hosts` is validated against `url` at parse time (MCP-011). To exfiltrate the token, they would need to control both the URL and the allowed_hosts declaration — but if they can do that, they can also set `url` to the legitimate host and issue malicious tool calls directly, which is a strictly more powerful attack that no allow-list prevents.

2. **Does not burden the unauthenticated case.** `allowed_hosts` is only required when `auth` is configured. An MCP HTTP tool without auth has no token to protect and no restriction.

3. **Small implementation surface.** A host-match check in the HTTP transport before header attachment. David already has a policy hook routed for this.

4. **Prevention, not just detection.** B-27's `mcp/authAttached` trace event remains (for audit), but this is the gate that stops the send.

### Updated target YAML (canonical)

```yaml
transport:
  mode: mcp-http
  url: https://icm-mcp-prod.azure-api.net/v1/
  auth:
    provider: azure-cli
    scope: api://icmmcpapi-prod/mcp.tools
    allowed_hosts:
      - icm-mcp-prod.azure-api.net
```

### Error codes added

| Code | Class | Condition | Fatal? |
|---|---|---|---|
| `MCP-010` | `MCP` | `auth` configured without `allowed_hosts` | Fatal |
| `MCP-011` | `MCP` | `url` host not in `allowed_hosts` | Fatal |
| `MCP-W001` | `MCP-W` | Token not attached because host not in `allowed_hosts` (runtime, if URL is somehow different from validated url — defensive) | Warning |

### For the record

Token replay from a malicious host was **considered and accepted as a real threat**. The original B-25/B-27 reasoning that audience scoping structurally prevents exploitation was incorrect — audience scoping prevents the attacker from *consuming* the token at their own service, but does not prevent *replaying* it against the legitimate service. The `allowed_hosts` restriction is the prevention mechanism. The frozen catalog remains the primary trust boundary (only approved package authors can declare tool URLs), and `allowed_hosts` is defense-in-depth against a compromised author.

---

## B-33 — Runtime host mismatch is FATAL; redirects disabled on authenticated requests

**Date:** 2026-08-15T18:25:00-07:00

### Part 1: Fatal, not warning

**Ruling:** A runtime host mismatch MUST be fatal (`MCP-012`, class `MCP`). The request is NOT sent. The token is NOT attached. Execution stops with a clear error naming the mismatched host and pointing at `allowed_hosts`.

**Rationale:** B-32 exists to prevent token exfiltration. A warning that allows the request to proceed — with or without the token — defeats the control entirely:
- With token attached: the control is decorative (the token reaches the unapproved host).
- Without token attached: the request silently downgrades to unauthenticated, producing a confusing 401 from the far end that no operator will diagnose as a security gate firing.

Fatal is the only semantics that actually prevents the thing B-32 was designed to prevent.

**Ken must reclassify `ErrMCPW001`:** Change from `class: "MCP-W"` / warning to `code: "MCP-012"` / `class: "MCP"` / fatal. Remove `ErrMCPW001` entirely (it was never shipped externally — the frozen-catalog invariant on error codes applies only post-ship). Replace with:

```go
ErrMCP012 = &Error{code: "MCP-012", class: "MCP"}
```

Message: `"mcp-http: request to host %q blocked — not in auth.allowed_hosts %v"`

### Part 2: Redirects disabled on authenticated requests

**Ruling:** The `http.Client` used by `MCPHTTPTransport` for authenticated requests MUST have `CheckRedirect` set to reject ALL redirects (return `http.ErrUseLastResponse`). The request fails with `MCP-013`:

```go
ErrMCP013 = &Error{code: "MCP-013", class: "MCP"}
```

Message: `"mcp-http: redirect from %q to %q rejected — redirects are disabled on authenticated requests"`

**Rationale:** Per-hop host checking is more permissive but complex to get right (must strip the `Authorization` header before the redirect if the target host differs, race between check and send, interaction with HTTP/2 push). Disabling redirects entirely is simpler, correct, and matches the security posture of every major OAuth client library (which strip `Authorization` on cross-origin redirects by default). A legitimate MCP server that requires redirects can be addressed by the operator updating `url:` to point at the final endpoint.

For **unauthenticated** requests (no `auth:` configured), default redirect-following behavior is acceptable — there is no token to protect.

### Part 3: Host-matching semantics confirmed as binding

**Ruling:** The following is the single canonical definition for host matching, used by both MCP-011 (static validation) and MCP-012 (runtime check):

1. Parse `url` with `net/url.Parse`
2. Extract hostname via `u.Hostname()` (strips port)
3. Lowercase both the extracted hostname and each `allowed_hosts` entry
4. **Exact string equality** — no prefix matching, no suffix matching, no wildcards, no regex, no glob

This is the binding definition. If the static and runtime checks use different comparison logic, that is a bug.


# Stream C — Auth Provider (David)

**Status:** Implementation complete. `go build ./...` exit 0. All packages outside `internal/tool` pass. Note: `mcp_http_tess_test.go` has a pre-existing syntax error in `TestMCPHTTPTransport_SSE_CorrectIDSelected` (DEF-012 test missing server setup — untracked file, not caused by Stream C changes); `internal/tool` tests require Tess to fix that before `go test ./internal/tool/...` can pass.

---

## Update — emitter wiring audit + Don collision resolved (2026-08-15T19:12)

**Don's `emit_ctx.go` has not landed** — `internal/tool/emit_ctx.go` does not exist in the repo. My earlier solution (`pkg/trace/emitter.go`) already provides the same functionality and is what `auth_gate.go` currently imports. There is no import cycle.

**B-27 `mcp/authAttached` reaches the real trace stream — provably.**

Full production chain traced:
1. `internal/engine/engine.go:569` — `executor.WithEventEmitter(spanCtx, fn)` installs the emitter; `fn` calls `h.emitEventLocked(spanCtx, kind, payload)`.
2. `executor.WithEventEmitter` delegates to `trace.WithEventEmitter(ctx, e)` (same context key as `trace.EmitterFromContext`).
3. The enriched context is passed to `exec.Execute(emitterCtx, ...)` → `runtime.Invoke(ctx, ...)` → `invokePersistent(ctx, ...)` → `transport.Invoke(ctx, ...)`.
4. `MCPHTTPTransport.send(ctx, ...)` creates `reqCtx` as a child of `ctx` (adds deadline only), passes to `buildHTTPRequest(reqCtx, ...)` → `gate.AttachToken(reqCtx, req)`.
5. `AttachToken` calls `trace.EmitterFromContext(reqCtx)` — same key → receives the engine emitter.
6. `emitEventLocked` writes `trace.TraceEvent{Kind: "mcp/authAttached", Payload: {url_host, scope}}` to `h.engine.cfg.TraceWriter.Append` (the persistent trace file) AND broadcasts in-process.

The emitter is not merely called in tests — it is the engine's production emitter, and it writes to the trace file.

**MCP-013 redirect ownership:** I own it. `CheckRedirect` is configured in `NewMCPHTTPTransport` (my `mcp_http.go` changes); the policy decision (authenticated only, `http.ErrUseLastResponse`) is entirely within my stream. Don's transport receives the configured client.

---

Ken edited `auth_gate.go` as part of the MCP-012 reclassification. On inspection the file was already in the correct state for both changes he described:
- `AttachToken` already returned `errkit.New("MCP-012", ...)` as a fatal error (applied in my previous B-32 pass).
- `ValidateAuthConfig` already used `u.Hostname()` — not `u.Host` — so port-stripping was already correct.

The one genuine divergence was the **MCP-012 error message text**. My message read:
```
mcp-http: host %q is not in auth.allowed_hosts — token not attached; add %q to allowed_hosts or update url:
```
Ken's registered canonical form is:
```
mcp-http: request host %q is not in auth.allowed_hosts — token not attached (update allowed_hosts or url: to match)
```
Updated `auth_gate.go` to match Ken's exact wording. `go build ./...` exit 0; all packages outside `internal/tool` pass.

**Zero `MCP-W` references** anywhere in the codebase — confirmed by grep across all `.go` files.

---

Barbara's final ruling: `CheckRedirect` must return `http.ErrUseLastResponse` (not a custom error) so the redirect response is surfaced to `send()` rather than becoming a transport error wrapped in MCP-008.

**Changes in this pass:**
- `NewMCPHTTPTransport` (`mcp_http.go`): `CheckRedirect` now returns `http.ErrUseLastResponse` for authenticated transports. Unauthenticated transports (gate == nil) retain nil CheckRedirect and follow redirects normally.
- `send()` (`mcp_http.go`): Added 3xx detection block — when gate is non-nil and status is 3xx, emit `errkit.New("MCP-013", "...redirected to <Location>...— update url: to <Location>")` with the `Location` header value. This gives the operator an actionable message naming the destination.
- `TestMCPHTTPTransport_CheckRedirectInstalledOnAuthenticated` (`mcp_http_test.go`): Updated to assert `errors.Is(err, http.ErrUseLastResponse)` instead of `errkit.ErrMCP013`.
- `TestMCPHTTPTransport_TokenNeverReachesRedirectTarget` (new test): Security proof — redirecting server (in `allowed_hosts`) 302s to evil server; asserts evil server never receives any Authorization header AND error is MCP-013.

**errkit.Error.Is() semantics confirmed:** Code-based comparison (`e.code == t.code`), so `errors.Is(errkit.New("MCP-013", msg), errkit.ErrMCP013)` returns true. Don's `TestMCPHTTPTransport_RedirectBlocked_Authenticated` continues to pass without modification.

**Redirect policy stated:**
- Authenticated: redirect → hard stop (`http.ErrUseLastResponse` → `send()` detects 3xx → MCP-013 with Location). Token NEVER forwarded.
- Unauthenticated: follow redirects normally (no restriction).

---

---

## Update — B-32 revised: host-mismatch is now fatal (2026-08-15T18:47)

User instruction superseded the B-32 written text on the non-fatal/warning behavior.
New behavior: host not in `allowed_hosts` → hard MCP-012 error, request not sent.

Changes:
- `TokenGate.AttachToken`: returns `MCP-012` (fatal) on host mismatch instead of nil + mcp/authSkipped event
- `EventKindMCPAuthSkipped` removed from `pkg/trace/event.go` — no longer emitted
- `NewMCPHTTPTransport`: sets `CheckRedirect` to block redirects on authenticated transport (MCP-013); Don had already implemented this with `errkit.ErrMCP013`
- `auth_gate_test.go`: updated `TestTokenGate_RejectsDisallowedHost` (was non-fatal, now asserts MCP-012); added `TestTokenGate_SuffixConfusionRejected`; added `TestTokenGate_MCP012ErrorIsActionable`
- All 38 gate/auth tests pass; `go build ./...` exit 0

**Host-matching semantics (stated explicitly):**
- Match on `req.URL.Hostname()` (port-stripped) — exact case-insensitive
- No wildcards (a `*` entry is treated as a literal hostname and will never match)
- No IDN/punycode normalization
- Port specs in allowed_hosts entries are matched literally (not stripped) — use plain hostnames

**Redirect decision:** authenticated transports refuse all redirects (MCP-013). An operator who needs a redirect path must update `url:` to point to the final endpoint.

---

B-32 superseded B-25/B-27 on token attachment policy. New requirements implemented:

- `schema.AuthConfig.AllowedHosts []string` — required when auth is configured (Ken landed this in `pkg/schema/tool.go`)
- `TokenGate` struct in `internal/tool/auth_gate.go` — the single policy decision point for all token attachment
- `ValidateAuthConfig(rawURL, auth)` — static validation: MCP-010 (no allowed_hosts) and MCP-011 (url host not in list)
- `AttachToken(ctx, req)` — runtime enforcement + B-27 trace event emission
- `EventKindMCPAuthAttached` and `EventKindMCPAuthSkipped` added to `pkg/trace/event.go`
- `trace.EventEmitter` + `trace.WithEventEmitter` + `trace.EmitterFromContext` added to `pkg/trace/emitter.go` (cycle-free emitter sharing — `internal/executor/events.go` updated to delegate there)
- `MCPHTTPTransport` updated: `auth AuthProvider` → `gate *TokenGate`; added HTTP 401 invalidate-and-retry path
- `runtime.go`: validates auth config before construction; constructs `TokenGate` with `AllowedHosts`
- 11 gate tests in `auth_gate_test.go` including B-24 event-payload redaction proof

**Note on B-32 runtime behavior:** Host-mismatch at runtime (after static validation) is **non-fatal** per B-32. The request proceeds unauthenticated; `mcp/authSkipped` event is emitted with `MCP-W001` code. The static MCP-011 check catches the common misconfiguration at parse time.

---

## AuthProvider Interface (for Don — Stream B)

```go
// Package: github.com/ormasoftchile/gert/internal/tool

// AuthProvider acquires bearer tokens for HTTP transports that require authentication.
// Implementations handle acquisition, caching, and refresh internally.
//
// The token returned by Token is an in-memory secret. It must not be written
// to any persistent or observable surface (traces, logs, error messages, step
// output). See B-24.
type AuthProvider interface {
    // Token returns a valid bearer token, refreshing proactively when within
    // five minutes of expiry.
    Token(ctx context.Context) (string, error)

    // Invalidate clears any cached token, forcing re-acquisition on the next
    // Token call. Call after receiving HTTP 401 to handle mid-run token expiry.
    Invalidate()
}

// NewAuthProvider(provider, scope string) (AuthProvider, error)
// Recognized provider values: "azure-cli"
// Returns MCP-002 error for unknown provider.
```

**Don's usage pattern for mid-run 401:**
```go
// On HTTP 401 response:
auth.Invalidate()
token, err = auth.Token(ctx)  // forces re-acquisition
// retry the request with the new token
```

---

## Files Owned

- `internal/tool/auth.go` — `AuthProvider` interface + `NewAuthProvider` factory
- `internal/tool/auth_azurecli.go` — `AzureCLIAuthProvider` implementation
- `internal/tool/auth_azurecli_test.go` — 16 tests including B-24 redaction proof

---

## Caching and Refresh Strategy

- **Proactive refresh:** Token is refreshed when `now + 5min >= expiry`. The old (still-valid) token is returned on probe failure — graceful degradation, not error.
- **Hard expiry:** At the hard expiry point, the cache is invalid and re-acquisition is mandatory.
- **Mid-run 401 path:** `Don.MCPHTTPTransport` should call `Invalidate()` on receiving HTTP 401, then `Token()` again. `AzureCLIAuthProvider` handles the mutex and re-acquisition internally.
- **Expiry parsing:** Prefers `expires_on` (Unix timestamp, TZ-safe) from `az`'s JSON output; falls back to `expiresOn` ("2006-01-02 15:04:05.000000" in local time); falls back to `now + 50min` if both are absent/malformed.
- **Concurrency:** `sync.Mutex` guards the cache; concurrent calls during refresh block rather than stampede.

---

## Four Failure Modes (B-24 / actionable messages)

| Scenario | Error code | Message (summarised) |
|----------|------------|----------------------|
| `az` not on PATH | MCP-007 | `az CLI is not installed or not on PATH — install from https://aka.ms/installazurecli and retry` |
| Not logged in | MCP-007 | `not authenticated — run 'az login' and retry` |
| No scope consent | MCP-007 | `no consent for scope "<scope>" — grant application consent or run 'az login' with the required scope` |
| Malformed JSON output | MCP-007 | `az CLI returned malformed output: <json decode error>` or `missing accessToken field` |

Classification uses `errors.As(err, &exec.Error{})` to detect the not-installed case, then stderr heuristics (`AADSTS` → consent, `Please run 'az login'` → auth) for the others.

---

## B-24 Redaction Proof

Test: `TestAzureCLIAuthProvider_TokenNeverLeaksIntoDiagnostics` in `auth_azurecli_test.go`.

The test seeds the cache with a known token value, then forces all four failure paths and the generic error path. It asserts the known token string is absent from every error message returned. It also asserts that a second cache load doesn't return a stale token after `Invalidate()`.

Redaction guarantees by design:
- `classifyAzError` is only called on command failure — no token has been produced at that point.
- `azTokenResponse` is decoded in-memory and the raw `stdout` bytes are discarded.
- The `token` field on `AzureCLIAuthProvider` is mutex-protected and never serialized, traced, or logged.
- Error messages contain only: stderr text from `az`, the scope string (not a secret), and static operator guidance.

---

## Dependencies Not Yet Complete (Ken — Stream A)

- `errkit.ClassForCode("MCP-007")` returns `""` — Ken has not yet registered the `"MCP"` class in `errkit/errors.go`. Errors work; they just have empty class in the structured error.
- `schema.AuthConfig` type — Ken owns this. When he lands it, `runtime.go` already consumes it at the wire point (`def.Auth.Provider`, `def.Auth.Scope`).

---

## Cross-Stream Seam

Don's `MCPHTTPTransport` (Stream B) receives an `AuthProvider` from `runtime.go`'s switch case for `TransportMCPHTTP`. That wiring is already in place in `internal/tool/runtime.go` — it calls `NewAuthProvider(def.Auth.Provider, def.Auth.Scope)` and passes the result to `NewMCPHTTPTransport(def.URL, auth)`.

Don should call `auth.Token(ctx)` to get the bearer token and set it as `Authorization: Bearer <token>` on each HTTP request. For mid-run 401 handling, call `auth.Invalidate()` then `auth.Token(ctx)` again before the retry.

---

## Test Results

```
go test -count=1 ./internal/tool/... -run "TestAzureCLI|TestNewAuth|TestParseAz"
ok  github.com/ormasoftchile/gert/internal/tool  (all 16 PASS)
```

Full suite: `internal/tool` has pre-existing failures unrelated to Stream C:
- `TestStdioTransport_*` — missing `.testtools` binaries (pre-existing)
- `TestMCPTransport_*` — same
- `TestMCPHTTPTransport_SSEMultiLineData` — Don's in-flight SSE test (not mine)


# Don — Stream B: Streamable HTTP MCP Transport

**Date:** 2026-08-16
**Status:** COMPLETE
**Author:** Don (Backend Dev)
**Feature:** `mode: mcp-http` transport for GERT tool runtime

---

## What Was Built

### New files

| File | Purpose |
|---|---|
| `internal/tool/mcp_http.go` | `MCPHTTPTransport` implementing `pkg/tool.ToolTransport` |
| `internal/tool/mcp_types.go` | Shared `mcpContent`, `mcpCallResult`, `mcpResponse` types (extracted from `mcp.go`); `mcpResponse.Result` is `json.RawMessage` to support both `tools/call` and `tools/list` shapes |
| `internal/tool/mcp_http_test.go` | 10 tests: JSON/SSE response paths, multi-line SSE, session handling, MCP-005, MCP-006 (implied by session expiry), runtime wiring proof |

### Modified files

| File | Change |
|---|---|
| `internal/tool/mcp.go` | Removed duplicate types (now in `mcp_types.go`); updated `Invoke` to use `resp.callResult()` |
| `internal/tool/runtime.go` | Replaced Ken's placeholder `TransportMCPHTTP` case with real `NewMCPHTTPTransport` dispatch |
| `pkg/schema/tool.go` | Removed my accidental duplicate `AuthConfig` type (Ken had already added it); removed duplicate `TransportMCPHTTP` constant |
| `pkg/tool/tool.go` | Removed duplicate `TransportMCPHTTP` constant (Ken had already added it); `URL`/`Auth` fields I added were net-new |

### Streams A and C: already landed

By the time I started implementing, **Ken (Stream A)** had already landed:
- `TransportMCPHTTP` constant in both schema and runtime packages
- `URL`/`Auth` fields on `schema.TransportConfig`  
- `AuthConfig` type in `pkg/schema/tool.go`
- `mapTransport` case for `schema.TransportMCPHTTP`
- `URL`/`Auth` copy in `RuntimeToolDef`
- MCP-001…MCP-009 sentinels in `pkg/errkit/errors.go`
- `validate_transport.go` (MCP-001, MCP-002, MCP-010, MCP-011 validation)

**David (Stream C)** had also already landed:
- `AuthProvider` interface in `internal/tool/auth.go`
- `AzureCLIAuthProvider` in `internal/tool/auth_azurecli.go`

---

## Design Decisions Made

### `json.RawMessage` for `mcpResponse.Result`

The original stdio `MCPTransport` had `Result *mcpCallResult` in `mcpResponse`, which only works for `tools/call` responses. For `tools/list`, the result shape is `{"tools": [...]}`. Changing to `json.RawMessage` lets callers decode into the appropriate shape. Added `callResult()` helper so stdio code doesn't regress.

### No `http404Error` interface assertion in tests

Session-expiry detection uses a private sentinel type `*http404Error` rather than the HTTP 404 status code directly. This avoids accidentally triggering re-init on a 404 that happens for other reasons (e.g. wrong URL path from day one). The re-init-once guard ensures `MCP-004` fires on double failure.

### Token redaction (B-24)

`buildHTTPRequest` sets `Authorization: Bearer <token>` but the token is never returned from any function, never stored in `ToolResult`, and is not included in any error message. `AzureCLIAuthProvider.classifyAzError` explicitly does not include the token (which would not exist at failure time anyway).

---

## Test Coverage

| Test | Proves |
|---|---|
| `TestMCPHTTPTransport_JSONResponse` | Happy path, session ID capture |
| `TestMCPHTTPTransport_SSEResponse` | SSE framing, notification skip |
| `TestMCPHTTPTransport_SSEMultiLineData` | Multi-line SSE event + heartbeat comment skip |
| `TestMCPHTTPTransport_ToolError` | `isError:true` → error, matching stdio shape |
| `TestMCPHTTPTransport_SessionID` | Server omitting session ID — no error |
| `TestMCPHTTPTransport_UnexpectedContentType` | MCP-005 |
| `TestMCPHTTPTransport_ProtocolVersionHeader` | `MCP-Protocol-Version: 2025-03-26` on all requests |
| `TestMCPHTTPTransport_RuntimeWiring` | **WIRING PROOF** — `DefaultToolRuntime` dispatches `mcp-http` to `MCPHTTPTransport` |
| `TestMCPHTTPTransport_SessionExpiry` | 404 → re-init → retry; `initCount == 2` asserted |
| `TestMCPHTTPTransport_ListTools` | `tools/list` response parsing |

---

## Corrections Applied from Ken's Recon (Ken MCP-HTTP Recon)

Ken's recon landed after initial implementation and identified four items. All addressed:

### Correction 1 — ID correlation under SSE (important)

Ken's finding: the stdio MCPTransport has no real JSON-RPC id correlation — it assumes the next message off the pipe is the response. This is survivable on a synchronous pipe but **wrong for SSE**, where a server can interleave notifications between the terminal response.

**Fix applied:** `parseSSEResponse` now takes `expectedID int` and explicitly skips:
- Events with no `id` field (notifications/progress messages)
- Events whose `id` != `expectedID` (stale responses)

Only returns when it finds `id == expectedID`. The `parseJSONResponse` path also verifies the ID. The post-hoc ID check in `doRequest` was removed since correlation now happens at read time. Note: calls are serialized behind `t.mu`, so there's only one outstanding request at a time — but the explicit ID check makes this invariant visible and protects against future concurrency changes.

### Correction 2 — Protocol version divergence (noted, not resolved)

Stdio advertises `2024-11-05`; HTTP transport pins to `2025-03-26` per B-23. This divergence is real and documented. B-23's value is used; I have not resolved the divergence in stdio (out of scope per Ken's recon). Awaiting Barbara's ruling if she wants to address it.

### Correction 3 — Result decoding matches stdio exactly

`toolResult()` uses `callResult()` (decodes `json.RawMessage` → `mcpCallResult`) and follows the identical path: `content[0].text` → `Stdout`; if text parses as JSON → `Output`. `isError:true` → non-zero ExitCode + error. Protocol error → `MCP-009`. This is proven by `TestMCPHTTPTransport_ToolError`.

### Correction 4 — Initialize response inspection (stdio defect NOT replicated)

Ken's finding: stdio `ensureStarted` sets `initialized = true` before checking the initialize response — so a JSON-RPC error on init is silently treated as success.

**Fix applied:** `initialize()` now calls `doRequest()` and explicitly checks `initResp.Error != nil` before marking `t.initialized = true`. A server that returns a JSON-RPC error on initialize causes `Invoke` to fail with "mcp-http: initialize failed: server error N: <message>". Proven by `TestMCPHTTPTransport_InitializeError`.

### Import cycle fix (Ken's auth_gate.go)

Ken's `auth_gate.go` imported `internal/executor` for `executor.EmitterFromContext` (which didn't exist yet), creating a build-breaking import cycle: `internal/tool` → `internal/executor` → `internal/tool`.

**Fix:** Created `internal/tool/emit_ctx.go` defining `EmitFunc`, `WithToolEmitter`, and `ToolEmitterFromContext` in `internal/tool/` itself. Updated `auth_gate.go` to use `ToolEmitterFromContext` (no executor import). The executor can seed the context via `WithToolEmitter` without creating a cycle. This also unblocked `pkg/pjvm` and `internal/conformance` which were failing due to the cycle.

## Test Results (post-corrections)

- `go build ./...` ✅
- `internal/tool`: 11/11 `TestMCPHTTP*` pass; overall pass except `TestStdioTransport_*` (pre-existing known failure)
- `internal/conformance`: ✅ (was failing due to import cycle — now fixed)
- `pkg/pjvm`: ✅ (same fix)
- All other packages: ✅


---

## Session expiry and no-auth paths

- **Server omits `Mcp-Session-Id`**: proceed normally; no `Mcp-Session-Id` header sent on subsequent requests.
- **Server returns HTTP 404 mid-run**: session is cleared, `initialize` is retried once. On double failure: `MCP-004`.
- **No auth configured** (`transport.auth` absent): no `Authorization` header. Compliant with B-24 and §4.5.

---

## Follow-up Round (2026-08-16 — post-corrections)

### Item 1 — emit_ctx.go was dead code

I verified: `auth_gate.go` already imports `pkg/trace` and calls `trace.EmitterFromContext(ctx)` directly (Ken landed that with MCP-012). The `emit_ctx.go` file I created during the import-cycle fix defined a parallel `toolEmitKey{}` context key that nothing in production ever seeded. `WithToolEmitter` had zero callers outside tests. Deleted `emit_ctx.go`. Production audit-trail path is now correct: `engine.go` seeds `trace.WithEventEmitter`, `auth_gate.go` reads it with `trace.EmitterFromContext`.

### Item 2 — End-to-end wiring: two-test proof

`TestMCPHTTPTransport_RuntimeWiring` (already existed) proves `DefaultToolRuntime.Invoke` dispatches through the `TransportMCPHTTP` switch to `invokePersistent` → `NewMCPHTTPTransport`. The gap it leaves: it constructs `toolpkg.TransportMCPHTTP` directly in Go code, not via the `schema → mapTransport` conversion that `ScanDir`/`ParseToolFile` uses.

Added `TestMCPHTTPTransport_SchemaRuntimeWiring`: calls `RuntimeToolDef` on a `schema.ToolDef` with `Transport.Type = schema.TransportMCPHTTP`, asserts the result has `Transport = toolpkg.TransportMCPHTTP` and `URL` correctly propagated, then invokes through `DefaultToolRuntime`. This proves the `schema.TransportMCPHTTP → mapTransport → toolpkg.TransportMCPHTTP → MCPHTTPTransport` path that a `.tool.yaml` file with `mode: mcp-http` takes.

The two tests together prove the full chain from schema constant through runtime wiring.

### Item 3 — MCP-013 redirect blocking (B-30)

Implemented in `NewMCPHTTPTransport`: when `gate != nil`, `httpClient.CheckRedirect` is set to return `fmt.Errorf("%w: ...", errkit.ErrMCP013)`. Unauthenticated transports (gate == nil) have no `CheckRedirect` installed and follow redirects normally.

Note: `CheckRedirect` fires BEFORE `AttachToken` is called for the redirect request, so a redirect from an allowed host to a different host is blocked before the token can be forwarded.

Three tests added:
- `TestMCPHTTPTransport_RedirectBlocked_Authenticated`: live 307 redirect + gate with correct allowed_hosts → `errors.Is(err, errkit.ErrMCP013)` ✓
- `TestMCPHTTPTransport_NoCheckRedirectOnUnauthenticated`: structural check, `gate == nil → httpClient.CheckRedirect == nil` ✓
- `TestMCPHTTPTransport_CheckRedirectInstalledOnAuthenticated`: structural + functional check, installed `CheckRedirect` function unwraps to `ErrMCP013` ✓

Tess's `TestMCPHTTPTransport_AuthenticatedRedirect_Blocked_MCP013` has a hard `t.Skip("BLOCKED-DEF-011: ...")`. The blocking condition is now resolved; Tess should remove the skip.

### Item 4 — Ken's auth_gate.go edits (reconciliation)

Current `auth_gate.go` already has:
- `u.Hostname()` fix (not `u.Host`) in both `AttachToken` and `ValidateAuthConfig` ✓
- `trace.EmitterFromContext` (not `emit_ctx.go` key) ✓
- **MCP-012 fatal** behavior in `AttachToken` for disallowed host ✓ (Ken landed this; it was not yet visible in my earlier read)

My earlier diagnostic that MCP-012 was "not yet landed" was wrong — the file was in intermediate state when I read it. Current state: fully reconciled, no conflicts.

### Test results (follow-up complete)

- `go build ./...` ✅
- `go test ./... -count=1`: all packages pass; no new failures vs. known pre-existing `TestStdioTransport_*` and `internal/serve TestSSE_FilterByRunID`
- `go vet ./...`: only pre-existing two `internal/serve` warnings


# Ken — MCP-HTTP Transport Recon

**Date:** 2026-08-16T00:55:39Z
**Author:** Ken (Backend Dev)
**Purpose:** Ground-truth survey for Barbara's gating architecture contract. No design, no production code.

---

## Q2 FIRST — The Critical Seam

**The `ToolTransport` interface exists and is the clean seam. No extraction needed.**

`pkg/tool/tool.go`:
```go
type ToolTransport interface {
    Invoke(ctx context.Context, def ToolDef, action string, args map[string]any) (*ToolResult, error)
    Close() error
}
```

All four existing transports (`StdioTransport`, `JSONRPCTransport`, `MCPTransport`, `NativeCLITransport`) implement it. A new `MCPHTTPTransport` would implement the same two methods without touching any existing type.

**What is NOT separable:** The `writeMCPMessage` / `readMCPMessage` / `readContentLength` helpers in `mcp.go` are Content-Length framed stdio I/O — they're coupled to `*bufio.Reader`, `*bufio.Writer`, and `*ProcessHandle`. They are NOT reusable for HTTP. The HTTP transport rewrites the transport layer entirely but inherits the same interface contract and the same result-decoding logic.

**What can be shared:** The response shape types (`mcpContent`, `mcpCallResult`, `mcpResponse`, `jsonrpcError`) are small and could be moved to a new `internal/tool/mcptypes.go` for sharing, or simply redeclared in the HTTP transport file. They contain no I/O logic.

**ID correlation note:** The current `MCPTransport` does NOT implement request-ID correlation. It serializes all calls behind `t.mu` (sync.Mutex) and reads the next message, assuming it is the response to the pending request. For Streamable HTTP MCP, this assumption does NOT hold if the server can deliver SSE notifications between response frames. The HTTP transport needs real ID correlation (a `map[int]chan jsonrpcResponse` or equivalent). This is a new implementation requirement, not a port of existing code.

---

## 1. The Stdio MCP Transport as It Exists

**File:** `internal/tool/mcp.go`

**Struct:**
```go
type MCPTransport struct {
    mu          sync.Mutex
    proc        *ProcessHandle
    reader      *bufio.Reader
    writer      *bufio.Writer
    nextID      int
    initialized bool
}
```

**Entry points:**
- `Invoke(ctx, def, action, args) (*ToolResult, error)` — holds `t.mu` for the entire call
- `ListTools(ctx, def) ([]map[string]any, error)` — same pattern
- `Close() error` — sends `shutdown` request, calls `proc.Kill()`
- `ensureStarted(ctx, def) error` — private; called at start of Invoke/ListTools

**Lifecycle (ensureStarted → Invoke → Close):**

1. `ensureStarted`: calls `StartProcess(ctx, def.Command, def.Args, def.Env)` → `*ProcessHandle` (process is live); creates `bufio.Reader`/`Writer` wrapping process stdio; sends `initialize` request with `protocolVersion: "2024-11-05"`, reads ONE response (no ID check on result); sends `notifications/initialized` notification (no ID, no response expected); sets `t.initialized = true`.

2. `Invoke` / `ListTools`: write request via `writeMCPMessage` (Content-Length framed), flush, call `readMCPMessage` (reads Content-Length header + body), unmarshal response, decode result.

3. `Close`: sends `shutdown` request, flushes, kills process.

**Wire framing:** LSP-style Content-Length headers — `Content-Length: N\r\n\r\n{body}`. Implemented by:
- `writeMCPMessage(w *bufio.Writer, payload any)` — marshals to JSON, writes header + body
- `readMCPMessage(ctx, r *bufio.Reader, proc *ProcessHandle) ([]byte, error)` — reads header via `readContentLength`, then `io.ReadFull` for body; on `ctx.Done()` kills the process
- `readContentLength(r *bufio.Reader) (int, error)` — reads lines until blank line, parses Content-Length

**`initialize` handling:** Done in `ensureStarted`. The response is read and discarded (no field inspection). `notifications/initialized` is sent immediately after (no response read because it's a notification). Neither server-advertised capabilities nor server info are inspected.

**Defect noted (not fixing):** `ensureStarted` sets `t.initialized = true` before checking whether the initialize response is an error object. A server that replies with `{"jsonrpc":"2.0","id":0,"error":{"code":-32603,"message":"..."}}` to `initialize` will be silently treated as initialized.

---

## 3. `tools/list` and `tools/call` Dispatch

**Dispatch path:**

1. `internal/executor/tool.go` (not surveyed; calls `ToolRuntime.Invoke`)
2. `DefaultToolRuntime.Invoke` (`internal/tool/runtime.go`) looks up `def.Transport`:
   ```go
   case toolpkg.TransportMCP:
       return r.invokePersistent(ctx, toolName, *def, action, args, func() toolpkg.ToolTransport {
           return &MCPTransport{}
       })
   ```
3. `invokePersistent` caches the transport by tool name in `r.persistent map[string]toolpkg.ToolTransport` (created once, reused for all calls to the same tool).

**Result decoding** (`mcp.go::Invoke`):
```go
text := resp.Result.Content[0].Text
result := &toolpkg.ToolResult{ExitCode: 0, Stdout: text}
var parsed map[string]any
if json.Unmarshal([]byte(text), &parsed) == nil {
    result.Output = parsed
}
```
- Text content → `ToolResult.Stdout`
- If text parses as JSON object → also populates `ToolResult.Output`
- `resp.Result.IsError == true` → `fmt.Errorf("mcp tool error: %s", content[0].Text)` — returned as Go error
- `resp.Error != nil` (JSON-RPC protocol error) → `fmt.Errorf("mcp error %d: %s", code, message)`

**`ListTools`** returns `[]map[string]any` (raw tool descriptors from `result.tools`). No type mapping.

**Typed output preservation:** Only `Output map[string]any` on `ToolResult`. Works when the server emits JSON in the content text. No typed schema enforcement at the transport layer.

---

## 4. Transport Schema and Validation

**Schema type:** `pkg/schema/tool.go`:
```go
type TransportConfig struct {
    Type    Transport         `yaml:"type,omitempty"    json:"type"`
    Mode    string            `yaml:"mode,omitempty"    json:"mode,omitempty"`
    Command string            `yaml:"command,omitempty" json:"command,omitempty"`
    Args    []string          `yaml:"args,omitempty"    json:"args,omitempty"`
    Env     map[string]string `yaml:"env,omitempty"     json:"env,omitempty"`
}

type Transport string

const (
    TransportStdio   Transport = "stdio"
    TransportJSONRPC Transport = "jsonrpc"   // note: runtime constant is "stdio-jsonrpc"
    TransportMCP     Transport = "mcp"
    TransportNative  Transport = "native"
)
```

`UnmarshalYAML` on `TransportConfig` copies `Mode` into `Type` when `Mode != ""`, so `transport.mode: mcp` is equivalent to `transport.type: mcp` after parsing. The `Mode` field is retained verbatim.

**No JSON Schema governs tool files.** From `scan.go` comment: "No JSON Schema governs .tool.yaml (C3, no tool.v1.schema.json)". There is exactly one JSON Schema file in `schemas/`: `runbook.schema.json`. There is no `tool.schema.json`. Validation is entirely Go-side.

**No `oneOf` discrimination on mode.** All mode-specific fields (`Command`, `Args`, `Env`) are in a flat struct with no guards. Adding `url` and `auth` fields follows the same flat pattern — there is no JSON Schema to update for a oneOf constraint.

**Validation point at runtime:** `internal/tool/scan.go::mapTransport`:
```go
func mapTransport(t schema.Transport) (toolpkg.TransportType, error) {
    switch t {
    case schema.TransportStdio:   return toolpkg.TransportStdio, nil
    case schema.TransportJSONRPC: return toolpkg.TransportJSONRPC, nil
    case schema.TransportMCP:     return toolpkg.TransportMCP, nil
    case schema.TransportNative:  return toolpkg.TransportNative, nil
    default:
        return "", fmt.Errorf("unsupported transport %q", t)
    }
}
```
A `.tool.yaml` with `mode: mcp-http` today fails here with `unsupported transport "mcp-http"`.

**Second validation point:** `DefaultToolRuntime.Invoke` in `runtime.go` has a parallel switch on `def.Transport` with `default: return nil, fmt.Errorf("tool runtime: unsupported transport %q", ...)`. Both must be extended for `mcp-http`.

**`ToolDef` in `pkg/tool/tool.go`** carries only `Command string`, `Args []string`, `Env map[string]string` from the transport config. There are NO `URL` or `Auth` fields anywhere in `ToolDef`. These must be added to both `schema.TransportConfig` and `toolpkg.ToolDef`, and threaded through `RuntimeToolDef` in `scan.go`.

---

## 5. Existing HTTP and Auth Machinery

**HTTP server (not client):** `internal/serve/` is entirely server-side. `net/http.Server`, `http.Handler`, `http.ServeMux`. Not reusable for an outbound HTTP MCP client.

**SSE in `internal/serve/sse.go`:** Server-side SSE writer:
- `writeSSE(w http.ResponseWriter, ev RunEvent)` — writes `event:/id:/data:` lines
- `writeSSEHeartbeat(w http.ResponseWriter)` — writes `: hb\n\n` comment frame

This is **server-side only**. There is **no SSE client reader** anywhere in the codebase. The HTTP transport must implement SSE parsing from scratch: read `data:` prefixed lines, strip prefix, JSON-unmarshal the payload.

**HTTP client code:** There is NO reusable outbound HTTP client wrapper anywhere. The only HTTP client usage is in tests (`internal/serve/*_test.go`) via `httptest.NewServer` + `http.Get`/`http.Post`. Production code has no `http.Client`, no retry policy, no timeout helper.

**Webhook receiver** (`internal/eventbus/webhook.go`): HTTP *server* receiving inbound POSTs with HMAC-SHA256 signature verification. Not reusable for client.

**Auth / credentials / token machinery:** There is NO bearer token handling, no Azure CLI (`az`) invocation, no credential provider interface for HTTP, no `Authorization` header construction anywhere in the codebase. The input provider chain (`internal/input/`) handles env vars and Vault for runbook vars — it does not serve as an HTTP auth mechanism. All auth for `mcp-http` must be built from scratch.

**Azure CLI specifically:** Zero occurrences of `az`, `azure`, `AzureCLI`, or `azure-cli` anywhere in the repo. An `azure-cli` auth provider would exec `az account get-access-token --scope <scope>` and parse the JSON output. That is new infrastructure.

**Redaction helpers:** `pkg/governance/` has `RedactionPattern` for output scrubbing. There is no HTTP header redaction. Bearer tokens in `Authorization` headers sent in trace/logs must be explicitly sanitized — no existing helper does this.

---

## 6. Error Taxonomy

**Conformance error classes in `pkg/errkit/errors.go`:**
- `GXL-PARSE`, `GXL-TYPE`, `GXL-PATH`, `GXL-EVAL` — expression language
- `GIS-PARSE`, `GIS-PATH`, `GIS-TYPE`, `GIS-EVAL` — string interpolation
- `GCP-PARSE`, `GCP-RESOLVE`, `GCP-DEFAULT`, `GCP-TYPE`, `GCP-EVAL` — container platform
- `PKG`, `PKG-W` — package system
- `PLAN` — planner
- `ENUM`, `ENUM-W` — enum validation
- `DINC`, `DINC-W` — dynamic includes

**Tool / MCP transport errors: none exist.** All errors from `MCPTransport`, `StdioTransport`, etc. are plain `fmt.Errorf` strings with no conformance code or class. There is no `TOOL-*` or `MCP-*` family in the errkit registry.

**Proposed class for new errors:** `MCPH` (MCP over HTTP) would be consistent with the existing naming pattern — short, subsystem-prefixed, all-caps. `ClassForCode` in `errors.go` would need a new case for `strings.HasPrefix(code, "MCPH-")`. Barbara should confirm the name; I'm reporting the gap.

**`IsWarning` pattern:** The generalized `strings.HasSuffix(class, "-W")` convention (from my DINC work) means a `MCPH-W` class would be treated as a warning automatically without changing `IsWarning`.

---

## 7. Tool Registry and Wiring

**End-to-end path for a new `mcp-http` tool:**

1. `ScanDir` → `ParseToolFile` → `yaml.Unmarshal` into `schema.ToolDef` with `TransportConfig.Mode = "mcp-http"` → `TransportConfig.UnmarshalYAML` copies `Mode` → `Type = "mcp-http"`
2. `RuntimeToolDef` → calls `mapTransport("mcp-http")` → **FAILS** today with "unsupported transport"
3. If `mapTransport` is extended: returns new `toolpkg.TransportMCPHTTP` constant; `RuntimeToolDef` must also copy `URL` and `Auth` from `schema.TransportConfig` to `toolpkg.ToolDef`
4. `buildToolRegistry` in `adapter/wire.go` → wraps in `OverlayRegistry` — no changes needed here
5. `NewDefaultToolRuntime(registry)` — no changes needed
6. Tool step executor → `DefaultToolRuntime.Invoke` → switch on `def.Transport` → **FALLS TO DEFAULT** today; must add `case toolpkg.TransportMCPHTTP:` returning `r.invokePersistent(..., func() ToolTransport { return &MCPHTTPTransport{url: def.URL} })`

**Dead-code risk:** Two hard-error defaults guard the dispatch path. Missing either case (`mapTransport` or `Invoke` switch) produces a runtime error at first use — same failure mode as a typo in the transport name. Not a silent failure, but still a gap that must be caught by tests.

**`ToolDef` in `pkg/tool/tool.go` needs new fields:**
```go
// NEW — needed for mcp-http
URL  string       // transport.url
Auth *AuthConfig  // transport.auth (new type TBD)
```
These must be populated in `RuntimeToolDef` (scan.go) and threaded into `MCPHTTPTransport`.

**`ListTools` is NOT part of `ToolTransport` interface.** Only `MCPTransport` and the new HTTP transport need it. `DefaultToolRuntime` does not expose it. If Barbara's design needs dynamic tool discovery at runtime, a second interface or a separate method needs to be defined. Currently `ListTools` is called from nowhere in the production dispatch path — it exists only in `mcp_test.go` and the `MCPTransport` struct.

---

## 8. Test Infrastructure

**Existing infrastructure for stdio MCP:**
- `TestMain` in `internal/tool/test_main_test.go`: builds real Go binaries from `cmd/tools/` into a temp `.testtools/` dir, sets `PATH` and `GERT_TOOLS_DIR`, runs the test suite, cleans up.
- `cmd/tools/mcp-server/main.go`: a complete content-length-framed stdio MCP server fixture. Handles `initialize`, `notifications/initialized`, `tools/list`, `tools/call` (echo/fail/slow), `shutdown`.
- Tests in `internal/tool/mcp_test.go`: 4 tests, all using the compiled `mcp-server` binary.

**What is NOT there for HTTP testing:**
- No `httptest.NewServer` MCP fixture. One needs to be written.
- No SSE client reader (needed by both the production transport and any response-reading test helper).
- No HTTP server fixture in `cmd/tools/` directory.

**What an HTTP MCP test server needs:**
1. An `httptest.NewServer` that accepts POST at a fixed path
2. Handles `initialize`: reads JSON-RPC body, returns `{"result":{"protocolVersion":"...","serverInfo":{...}}}` with `Mcp-Session-Id: <uuid>` response header
3. Handles `notifications/initialized`: returns 202 Accepted (no body required per spec)
4. Handles `tools/list` and `tools/call`: may return either a direct JSON response or an SSE stream starting with `data: {jsonrpc response}\n\n`
5. Session state: validates that subsequent requests carry the `Mcp-Session-Id` header from step 2

**Recommended test structure:**
- `internal/tool/mcp_http_test.go` using `httptest.NewServer`
- A `fakeHTTPMCPServer` struct (analogous to `cmd/tools/mcp-server/main.go` but as an `http.Handler`)
- No separate binary needed — in-process httptest is sufficient and faster

**SSE parsing for production:**
The production transport must parse SSE lines from the response body. The spec format:
```
data: {"jsonrpc":"2.0","id":1,"result":{...}}\n\n
```
This is a net-new implementation. `bufio.NewScanner` on the response body, scan for `data:` prefix, strip prefix, JSON-unmarshal. Handle `: ` comment lines (heartbeats) by skipping. No existing helper.

---

## Summary of Gaps Barbara Must Design Against

| Gap | Severity | Location |
|-----|----------|----------|
| `TransportConfig` has no `url` or `auth` fields | Blocking | `pkg/schema/tool.go` |
| `toolpkg.ToolDef` has no `URL` or `Auth` fields | Blocking | `pkg/tool/tool.go` |
| `mapTransport` doesn't handle `mcp-http` | Blocking | `internal/tool/scan.go` |
| `DefaultToolRuntime.Invoke` doesn't handle `mcp-http` | Blocking | `internal/tool/runtime.go` |
| No HTTP client wrapper (no retry, no timeout) | Must build | new |
| No SSE client reader | Must build | new |
| No Azure CLI token provider | Must build | new |
| No bearer-token redaction for traces/logs | Must build | new (or governance extension) |
| No HTTP MCP test server fixture | Must build | `internal/tool/` tests |
| No errkit class for HTTP MCP errors | Barbara decides | `pkg/errkit/errors.go` |
| No ID correlation in MCPTransport (in-scope for HTTP) | Must implement | new MCPHTTPTransport only |
| `ListTools` not in `ToolTransport` interface; needed for mcp-http? | Barbara decides | `pkg/tool/tool.go` |


# Ken — Stream A: Schema + Validation Report

**Date:** 2026-08-16  
**Feature:** Native Streamable HTTP MCP transport (`mcp-http`)  
**Status:** COMPLETE — all 17 tests pass, `go build ./...` exit 0

---

## AuthConfig shape and extensibility

```go
// pkg/schema/tool.go
type AuthConfig struct {
    Provider string `yaml:"provider"`
    Scope    string `yaml:"scope,omitempty"`
}
```

**Extensibility story:** `Provider` is a plain string — no Go enum, no `oneOf` in YAML. Adding a second provider (e.g., `managed-identity`) requires:
1. Add its name to `knownAuthProviders` in `internal/tool/validate_transport.go` — one line.
2. Implement the `AuthProvider` interface in `internal/tool/auth_<provider>.go`.
3. Add a `case` in `NewAuthProvider` (David's file).

The `AuthConfig` struct itself is unchanged — no breaking change to `.tool.yaml` syntax or any schema type.

B-24 compliance: `auth:` names a *provider* and a *scope* only. There is no field that can hold a credential, token, or secret. Structurally impossible to inline credential material.

---

## Validation rules and error messages

All rules enforced in `internal/tool/validate_transport.go::ValidateTransportConfig`, called from `ParseToolFile` at scan time. Errors are `%w`-wrapped with the file path, so `errors.Is(err, errkit.ErrMCP001)` works through the chain.

| Rule | Condition | Sentinel | Error message (operator-actionable) |
|------|-----------|----------|--------------------------------------|
| mcp-http requires url | `url` absent | ErrMCP001 | `mcp-http: transport.url is required` |
| url must be HTTPS | `url` doesn't start with `https://` | ErrMCP001 | `mcp-http: url "<url>" must use https://` |
| command rejected on mcp-http | `command` non-empty on mcp-http | plain | `mcp-http: transport.command is not valid for mode mcp-http (mode mcp-http uses url:, not command:)` |
| args rejected on mcp-http | `args` non-empty on mcp-http | plain | `mcp-http: transport.args is not valid for mode mcp-http (mode mcp-http uses url:, not command/args)` |
| auth.provider must be known | provider not in `knownAuthProviders` | ErrMCP002 | `mcp-http: unknown auth provider "<name>" (recognized providers: azure-cli)` |
| mcp requires command | `command` empty on mcp | plain | `mcp: transport.command is required for mode mcp` |
| url rejected on mcp | `url` non-empty on mcp | plain | `mcp: transport.url is not valid for mode mcp (mode mcp uses command:, not url:)` |
| auth rejected on mcp | `auth` non-nil on mcp | plain | `mcp: transport.auth is not valid for mode mcp (auth is only used by mode mcp-http)` |

Following Barbara's B-14 principle: cross-arm fields are **rejected, not silently ignored**. A tool author who writes `url:` under `mode: mcp` gets an error telling them exactly what's wrong and what to use instead.

---

## Switch/discrimination points on transport mode

All enumerated to confirm no dead-code miss (recon finding: hard-error defaults).

| File | Location | `mcp-http` handled? |
|------|----------|---------------------|
| `internal/tool/scan.go` | `mapTransport` switch | ✅ `case schema.TransportMCPHTTP: return toolpkg.TransportMCPHTTP, nil` |
| `internal/tool/runtime.go` | `DefaultToolRuntime.Invoke` switch | ✅ stub returning "not yet wired — Stream B pending" |
| `internal/tool/validate_transport.go` | `ValidateTransportConfig` switch | ✅ primary enforcement |

The stub in `runtime.go` surfaces loudly at call time (returns an error), not silently. Don replaces it with `NewMCPHTTPTransport` when Stream B lands.

---

## B-32 Addendum: allowed_hosts

**Date:** 2026-08-15 (B-32 ruling incorporated)

Barbara reversed B-25/B-27. Audience-scoped tokens prevent a malicious host from consuming a token, but not from replaying it against the legitimate endpoint. `allowed_hosts` is the prevention mechanism.

### AuthConfig shape (revised)

```go
type AuthConfig struct {
    Provider     string   `yaml:"provider"`
    Scope        string   `yaml:"scope,omitempty"`
    AllowedHosts []string `yaml:"allowed_hosts,omitempty"`
}
```

`AllowedHosts` is required whenever `auth:` is present. Entries are bare hostnames (no scheme, no port). Matching is exact, case-insensitive, parsed host — no substring or wildcard. This prevents `icm.evil.com` from matching `icm.com` and `icm-mcp.azure-api.net.evil.com` from matching `icm-mcp.azure-api.net`. **David must use identical semantics** in the runtime host check before header attachment.

### New validation rules

| Rule | Condition | Sentinel | Error message |
|------|-----------|----------|----------------|
| allowed_hosts required when auth configured | `auth != nil && len(AllowedHosts) == 0` | ErrMCP010 | `mcp-http: auth.allowed_hosts is required when auth is configured (B-32: …)` |
| url host must be in allowed_hosts | url host not in AllowedHosts (static check) | ErrMCP011 | `mcp-http: url host "<host>" is not in auth.allowed_hosts [...]` |

No `allowed_hosts` requirement when `auth:` is absent — unauthenticated transports are unconstrained (B-32 explicitly preserves this).

### MCP-012 reclassification (B-32 revised ruling)

**Date:** 2026-08-15

`ErrMCPW001` / `MCP-W001` removed. Reclassified as `ErrMCP012` / `MCP-012`, class `MCP`, fatal.

- `MCP-W` class removed entirely from errkit (`ClassForCode`, `Classes()`, `classSentinels`, `codeOrder`).
- `ErrMCP013` added: redirect blocked on authenticated request — `http.ErrUseLastResponse`; David raises this in the HTTP transport.
- `auth_gate.go`: `AttachToken` now returns `errkit.New("MCP-012", ...)` on host mismatch instead of emitting a warning and proceeding. `u.Host` → `u.Hostname()` fixed in both `AttachToken` and `ValidateAuthConfig` to match binding semantics.
- `pkg/schema/tool.go` doc comment updated.
- Repo-wide sweep confirmed: no remaining references to `MCP-W001`, `ErrMCPW001`, or `MCP-W` class.

**MCP-012 message:** `mcp-http: request host "<host>" is not in auth.allowed_hosts — token not attached (update allowed_hosts or url: to match)`

All 22 tests pass. `go build ./...` and `pkg/errkit` tests clean.


### New/updated tests (5 added, 2 updated)

- `TestValidateTransport_MCPHTTP_ValidWithAuth` — updated with `AllowedHosts`
- `TestValidateTransport_MCPHTTP_MissingAllowedHosts_MCP010` — NEW
- `TestValidateTransport_MCPHTTP_HostNotInAllowedHosts_MCP011` — NEW
- `TestValidateTransport_MCPHTTP_SubstringHostDoesNotMatch_MCP011` — NEW (replay-attack guard)
- `TestValidateTransport_MCPHTTP_NoAuth_AllowedHostsNotRequired` — NEW
- `TestParseToolFile_MCPHTTP_ValidWithAuth` — updated with `allowed_hosts:` in YAML
- `TestParseToolFile_MCPHTTP_UnknownProvider_Fails` — updated with `allowed_hosts:` to isolate MCP-002

All 22 tests pass. `go build ./...` and `pkg/errkit` tests clean.

### Host-matching semantics (for David to match)

1. Parse `cfg.URL` with `net/url.Parse` — use `u.Hostname()` (strips port if present)
2. Lowercase both the parsed hostname and each `AllowedHosts` entry
3. Exact string equality only — no `strings.HasPrefix`, no `strings.Contains`, no wildcard expansion
4. Port in `AllowedHosts` entries is not supported in this iteration; entries must be bare hostnames

If a future ruling adds wildcard support (e.g. `*.azure-api.net`), only `hostInList` in `validate_transport.go` and the parallel runtime check need updating — the schema and `AuthConfig` struct are unchanged.


- **`pkg/schema/tool.go`** — `TransportMCPHTTP` constant; `URL string` + `Auth *AuthConfig` on `TransportConfig`; `AuthConfig` struct
- **`pkg/tool/tool.go`** — `TransportMCPHTTP TransportType = "mcp-http"` constant
- **`pkg/errkit/errors.go`** — `MCP` class; `ErrMCP001`..`ErrMCP009` sentinels; `ClassForCode`, `codeOrder`, `classSentinels`, `sentinels`, `Classes()` all updated
- **`internal/tool/scan.go`** — `mapTransport` case; `URL`/`Auth` threading in `RuntimeToolDef`; `ValidateTransportConfig` call in `ParseToolFile`
- **`internal/tool/runtime.go`** — stub `case toolpkg.TransportMCPHTTP`
- **`internal/tool/validate_transport.go`** (new) — full validation logic
- **`internal/tool/validate_transport_test.go`** (new) — 17 tests (11 unit + 5 integration through `ParseToolFile` + 1 `toolFileTemplate` const)

---

## Test results

```
=== 17 new tests: ALL PASS ===
TestValidateTransport_MCPHTTP_ValidMinimal          PASS
TestValidateTransport_MCPHTTP_ValidWithAuth         PASS
TestValidateTransport_MCPHTTP_MissingURL_MCP001     PASS
TestValidateTransport_MCPHTTP_HTTPNotHTTPS_MCP001   PASS
TestValidateTransport_MCPHTTP_CommandRejected       PASS
TestValidateTransport_MCPHTTP_ArgsRejected          PASS
TestValidateTransport_MCPHTTP_UnknownProvider_MCP002 PASS
TestValidateTransport_MCPStdio_ValidMinimal         PASS
TestValidateTransport_MCPStdio_MissingCommand       PASS
TestValidateTransport_MCPStdio_URLRejected          PASS
TestValidateTransport_MCPStdio_AuthRejected         PASS
TestParseToolFile_MCPHTTP_ValidMinimal              PASS
TestParseToolFile_MCPHTTP_ValidWithAuth             PASS
TestParseToolFile_MCPHTTP_MissingURL_Fails          PASS
TestParseToolFile_MCPHTTP_HTTPScheme_Fails          PASS
TestParseToolFile_MCPHTTP_UnknownProvider_Fails     PASS
```

Pre-existing failures (not mine):
- `internal/tool TestStdioTransport_*` — missing `.testtools/internal-tool/json-emitter.exe` (build artifact)
- No `internal/serve` failures (green)
- Two `go vet` warnings in `internal/serve` (pre-existing, not mine)

`go build ./...` exit 0.


# Tess — MCP HTTP Stream D Final Decision Report
**Date:** 2026-08-16T02:29:31Z  
**Requested by:** Cristián  
**Context:** Stream C (David's AzureCLIAuthProvider) — final adversarial pass  
**Status:** CLOSED

---

## Team-relevant finding: David's redaction sentinel is weak

`auth_azurecli_test.go` defines `knownToken = "******"`. Six asterisks can appear in truncated Go error formatting (e.g., `"...got ******: ..."`) or in log output using redaction markers. A test asserting `!strings.Contains(msg, "******")` can pass even when the real token is present, as long as no six-asterisk run appears in the message.

**This is a gap in David's redaction test, not a gap in production code.** The production redaction behavior is correct — `AzureCLIAuthProvider` never passes the raw token to any error constructor, only the classified failure reason. The weakness is that the test would not catch a future regression where a developer accidentally logs the token, because the sentinel would not collide with normal output in the same way a high-entropy value would.

**Recommendation:** Change `knownToken` in `auth_azurecli_test.go` to a high-entropy sentinel like `TESS_SENTINEL_TOKEN_D42E9B1C`. Tess's `TestAzureCLIProvider_SentinelNeverInAnyErrorOutput` already uses this value and covers all five error paths independently. The two test files are complementary, but David's should be strengthened.

**Action:** Barbara or Cristián to decide whether to ask David to update `knownToken` before the next commit, or accept the complementary coverage as-is.

---

## AUTH-003 tightening — confirmed safe

`TestMCPHTTPTransport_SuffixConfusion_TokenNotAttached` now asserts `err != nil` unconditionally (was conditional). The test passes after tightening because DEF-013 is genuinely fixed: `AttachToken` returns fatal `MCP-012` on host mismatch. If someone regresses it back to warn-and-continue, the test will now fail immediately.

---

## Final counts

- **Stream D vectors:** 27 PASS / 0 SKIP / 0 FAIL  
- **Group I (AzureCLI):** 4 PASS / 0 SKIP / 0 FAIL  
- **Total Tess vectors:** 31 PASS / 0 SKIP / 0 FAIL  
- **Full suite:** 62 packages, `go test ./... -count=1` exit 0  


# Tess — MCP HTTP Stream D Report
**Date:** 2026-08-16 (updated after stale-defect correction pass)  
**Requested by:** Cristián  
**Feature:** Native Streamable HTTP MCP transport (`mode: mcp-http`)  
**Test file:** `internal/tool/mcp_http_tess_test.go`

---

## Lead verdicts (required)

### Redirect-with-token (TV-MCP-AUTH-002) — **PASS. DEF-011 was stale — B-33 Part 2 IS implemented.**

`NewMCPHTTPTransport` installs `CheckRedirect` when `gate != nil`, returning `ErrMCP013`. A 302 from an allowed host is refused before the redirect fires. `TestMCPHTTPTransport_AuthenticatedRedirect_Blocked_MCP013` **PASSES**: `Invoke` returns an MCP-013 error, and the evil server confirms it received zero requests — the `Authorization` header never reached the redirect destination. The initial defect report (DEF-011) was filed against mid-flight code; the fix had landed before tests ran.

### Interleaved-notification SSE (TV-MCP-SSE-001) — **PASS. Both notification skip and ID correlation correct.**

`parseSSEResponse` skips nil-id events (notifications) AND checks `resp.ID == expectedID` — both guards are present. `TestMCPHTTPTransport_SSE_NotificationBeforeResponse` confirms notification skip. `TestMCPHTTPTransport_SSE_CorrectIDSelected` confirms that a stale response (id=99) before the matching response is skipped and the correct answer is returned. Both **PASS**. The initial DEF-012 report was also stale.

---

## End-to-end token-leak sweep (B-24 / B-27)

Scope: all observable surfaces through a complete authenticated `Invoke` call — error messages, `ToolResult` fields, and trace event payloads. Sentinel: `TESS_SENTINEL_TOKEN_D42E9B1C`.

Methodology in `TestMCPHTTPTransport_TokenRedaction_NotInAnyOutput`:
1. A real `TokenGate` with a `mockAuthProvider` returning the sentinel token
2. The server echoes the `Authorization` header back as the tool result text (to verify the token IS reaching the server — baseline confirmation)
3. A `trace.WithEventEmitter` captures all emitted events for the call
4. Sweep: error messages, `result.Stderr`, non-stdout `result.Output` fields, all trace event payloads (JSON-serialized)

**Result: PASS. No leak found.**
- `result.Stderr` is empty — not leaked.
- No non-stdout `result.Output` field contains the sentinel.
- `mcp/authAttached` event was emitted (B-27 confirmed) with payload `{url_host, scope}` — no token in the serialized payload.
- Auth provider failure errors (`TestMCPHTTPTransport_AuthFailure_ErrorHasNoToken`) contain only the MCP-007 failure reason, not the sentinel. **PASS.**

---

## B-27 emitter wiring verdict (requested)

`mcp/authAttached` **IS wired in production.** Evidence:

`internal/engine/engine.go` calls `executor.WithEventEmitter(spanCtx, func(...) { h.emitEventLocked(...) })` before every `exec.Execute` call. This `emitterCtx` is passed as `ctx` to the executor, which calls `runtime.Invoke(ctx, ...)`, which flows through `MCPHTTPTransport.Invoke(ctx, ...)` → `buildHTTPRequest(ctx, ...)` → `gate.AttachToken(ctx, req)`. `AttachToken` calls `trace.EmitterFromContext(ctx)` which finds the engine-installed emitter. The event is then routed through `emitEventLocked` → `trace.TraceEvent` → persisted in the run store.

`TestMCPHTTPTransport_TokenRedaction_NotInAnyOutput` proves this end-to-end: when a real emitter is installed in ctx, `mcp/authAttached` fires and is captured. Without an emitter in ctx, `EmitterFromContext` returns nil (safe no-op per the emitter contract). There is no gap — the engine always provides the emitter.

---

## Defects found (summary — all resolved or stale)

| DEF | What | Status |
|-----|------|--------|
| DEF-007 | Import cycle `auth_gate.go` → `internal/executor` | RESOLVED before tests ran |
| DEF-008 / DEF-009 / DEF-010 | Compile failures (`t.auth`, arg count, type mismatch) | RESOLVED by Don before tests ran |
| DEF-011 | No `CheckRedirect` on `httpClient` | STALE — fix was already in HEAD |
| DEF-012 | `parseSSEResponse` no `id == expectedID` check | STALE — fix was already in HEAD |
| DEF-013 | `AttachToken` returns nil on host mismatch | STALE — `MCP-012` return was already in HEAD |

No open security defects. All three reported security issues (DEF-011/DEF-012/DEF-013) were filed against mid-flight code and were already fixed by the time tests ran against stable HEAD.

---

## Vector results

| # | Test | Status |
|---|------|--------|
| TV-MCP-INIT-001 | `TestMCPHTTPTransport_InitRejected_NotMarkedInitialized` | **PASS** |
| TV-MCP-INIT-002 | `TestMCPHTTPTransport_B30_NotReplicatedFromStdio` | **PASS** |
| TV-MCP-INIT-003 | `TestMCPHTTPTransport_ProtocolVersionHeader_2025` | **PASS** |
| TV-MCP-SESS-001 | `TestMCPHTTPTransport_NoSessionID_Accepted` | **PASS** |
| TV-MCP-SESS-002 | `TestMCPHTTPTransport_SessionIDSentOnSubsequentCalls` | **PASS** |
| TV-MCP-SESS-003 | `TestMCPHTTPTransport_SessionIDUpdatedOnLaterResponse` | **PASS** |
| TV-MCP-SESS-004 | `TestMCPHTTPTransport_SessionExpiry_ReinitFails_MCP004` | **PASS** |
| TV-MCP-AUTH-001 | `TestMCPHTTPTransport_AllowedHost_TokenAttached` | **PASS** |
| TV-MCP-AUTH-002 | `TestMCPHTTPTransport_AuthenticatedRedirect_Blocked_MCP013` | **PASS** |
| TV-MCP-AUTH-003 | `TestMCPHTTPTransport_SuffixConfusion_TokenNotAttached` | **PASS** |
| TV-MCP-AUTH-004 | `TestMCPHTTPTransport_RuntimeHostMismatch_Fatal_MCP012` | **PASS** |
| TV-MCP-SSE-001 | `TestMCPHTTPTransport_SSE_NotificationBeforeResponse` | **PASS** |
| TV-MCP-SSE-002 | `TestMCPHTTPTransport_SSE_CorrectIDSelected` | **PASS** |
| TV-MCP-SSE-003 | `TestMCPHTTPTransport_SSE_StreamEndsWithoutResponse_MCP006` | **PASS** |
| TV-MCP-SSE-004 | `TestMCPHTTPTransport_SSE_MultiLineData_HeartbeatIgnored` | **PASS** |
| TV-MCP-FAIL-001 | `TestMCPHTTPTransport_ConnectionRefused_MCP008` | **PASS** |
| TV-MCP-FAIL-002 | `TestMCPHTTPTransport_ServerError5xx` | **PASS** |
| TV-MCP-FAIL-003 | `TestMCPHTTPTransport_MalformedJSON` | **PASS** |
| TV-MCP-FAIL-004 | `TestMCPHTTPTransport_ContextCancelled_Error` | **PASS** |
| TV-MCP-FAIL-005 | `TestMCPHTTPTransport_JSONRPCError_MCP009` | **PASS** |
| TV-MCP-FAIL-006 | `TestMCPHTTPTransport_UnexpectedContentType_MCP005` | **PASS** |
| TV-MCP-PARITY-001 | `TestMCPHTTPTransport_JSONOutput_Stdout_And_OutputMap` | **PASS** |
| TV-MCP-PARITY-002 | `TestMCPHTTPTransport_PlainTextOutput_StdoutOnly` | **PASS** |
| TV-MCP-PARITY-003 | `TestMCPHTTPTransport_IsError_ExitCode1_MCPToolError` | **PASS** |
| TV-MCP-PARITY-004 | `TestMCPHTTPTransport_SSEResult_SameShapeAsJSON` | **PASS** |
| TV-MCP-REDACT-001 | `TestMCPHTTPTransport_TokenRedaction_NotInAnyOutput` | **PASS** |
| TV-MCP-REDACT-002 | `TestMCPHTTPTransport_AuthFailure_ErrorHasNoToken` | **PASS** |

**Final summary: 27 PASS / 0 SKIP / 0 FAIL** (of 27 Stream D vectors)

---

## Stream C extension (David's AzureCLIAuthProvider) — 2026-08-16T02:29:31Z

**Updated after final pass. 4 Group I adversarial vectors added. Total: 31 PASS / 0 SKIP / 0 FAIL.**

### AUTH-003 tightening
`TestMCPHTTPTransport_SuffixConfusion_TokenNotAttached` had a conditional `if err != nil { check MCP-012 }` that would silently pass on a regression to nil-return. Tightened to `if err == nil { t.Fatal(...) }` — error is now required unconditionally, matching the shape of TV-MCP-AUTH-004. **HOLDS after tightening.** DEF-013 is fixed, `AttachToken` returns fatal MCP-012, Invoke returns non-nil error.

### Comment cleanup
File header, AUTH-003, AUTH-004 stale `// BLOCKED: DEF-013` annotations rewritten. DEF-011/012/013 now marked RESOLVED in header. No false-defect commentary survives.

### Group I — AzureCLI adversarial vectors

| # | Test | Status |
|---|------|--------|
| TV-AZ-001 | `TestAzureCLIProvider_SentinelNeverInAnyErrorOutput` | **PASS** |
| TV-AZ-002 | `TestAzureCLIProvider_NoRetryLoopOnFailure` | **PASS** |
| TV-AZ-003 | `TestAzureCLIProvider_InvalidateThenFailDoesNotReturnStale` | **PASS** |
| TV-AZ-004 | `TestAzureCLIProvider_InvalidateCalledOnTransport401` | **PASS** |

### Sentinel sweep against David's provider

Sentinel: `TESS_SENTINEL_TOKEN_D42E9B1C`. Swept across: all 5 error message paths (az not installed, not logged in, no scope consent, malformed output, generic failure), `Invalidate()`+re-acquire failure, and 401-triggered re-acquire failure. **CLEAN — sentinel not found in any error message, no panic output.** 

David's own redaction test uses `knownToken = "******"` — asterisks can appear in truncated Go error output, making that sentinel collision-prone. Our sentinel is high-entropy and collision-proof. The two tests are complementary; no contradiction.

### Expiry cache behavior verified
`AzureCLIAuthProvider` caches by expiry. `Invalidate()` clears both `token` and `expiry` fields. After `Invalidate()`, a subsequent `Token()` call with a failing runner returns an error (stale token not returned). Cache hit is confirmed: second `Token()` call with a fixed-expiry success runner does not spawn a second shell process. 

### No retry loop
A single failure returns one error. No spin/retry loop exists in `Token()`.

### 401-driven Invalidate
`TestAzureCLIProvider_InvalidateCalledOnTransport401` proves `Invalidate()` is called when the transport receives HTTP 401, so a stale token is discarded before the re-auth attempt rather than being retried forever.

### Build and test status after final pass

```
go build ./...            → exit 0
go test ./internal/tool   → 31 Tess vectors PASS, 0 SKIP, 0 FAIL  
go test ./... -count=1    → exit 0 (62 packages, all green)
```

---

## Key findings summary

1. **B-30 verified**: HTTP transport checks `initResp.Error != nil` before setting `initialized = true`. The stdio defect was NOT replicated. ✓
2. **Protocol version verified**: HTTP transport sends `MCP-Protocol-Version: 2025-03-26`. Does not use stdio's `2024-11-05`. ✓
3. **Requirement 4 (transport parity) verified**: `toolResult()` logic is identical between JSON and SSE paths. Same `ToolResult` shape from both transports. ✓
4. **SSE interleaved notification**: correctly skipped (nil id). ✓
5. **SSE ID correlation**: correct — stale responses with wrong id are skipped. ✓
6. **B-33 Part 1 (host mismatch fatal)**: `AttachToken` returns `MCP-012`, request not sent. ✓
7. **B-33 Part 2 (redirect blocking)**: `CheckRedirect` returns `MCP-013` on authenticated transports. ✓
8. **B-27 (mcp/authAttached wiring)**: event fires and is routed through engine's emitEventLocked in production. ✓
9. **B-24 (token redaction)**: transport layer clean — sentinel not found in errors, result fields, or trace event payloads. ✓

---

## Build and test status

```
go build ./...   → exit 0
go test ./...    → exit 0 (62 packages, all green)
```


