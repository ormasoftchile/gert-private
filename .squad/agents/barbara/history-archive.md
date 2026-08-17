# barbara — History Archive

**Archived:** 2026-08-15T16:27:17.3244895-07:00
**Size:** 43653 bytes

---

# Barbara — Project History (Summarized)

## Current Status Summary (2026-06-07)

**Phase 1 COMPLETE.** GXL/GIS/GCP grammars (75 KB EBNF), spec sections (§03a/03b/03c/03d LaTeX), conformance corpus (211 vectors, 0 TBD).

**Phase 2 blocking decisions finalized:**
- **Runbook v1 schema canonical home:** `gert-private/design/gert/schemas/runbook.v1.schema.json` (hand-authored, language-neutral, Edith authored, 23/23 examples validate)
- **Schema arbitration queue:** Edith filed 5 spec questions (SQ-001 to SQ-005) for Barbara ratification
- **Conformance schema naming:** Tess recommends rename `schema.json` → `vector.schema.json` (awaiting Barbara ratification)
- **Parse-gate proposal:** Deferred to separate cycle before Day 9 runtime kickoff

**Current role:** Spec authority, architecture arbitration, Phase 2 decision arbiter

**Immediate next steps:** (1) Arbitrate 5 Edith spec questions, (2) Ratify Tess schema rename, (3) Draft parse-gate proposal

---

## Consolidated Phase 1 Record

**2026-06-04 to 2026-06-07:** Delivered and arbitrated:
- Stream A/B/F: GXL/GIS/GCP EBNF grammars + LaTeX specs + parser gate spec (9 PLAN-* codes)
- OQ resolutions: OI-GIS-01 (`\${` escape), OI-GCP-02 (bare-root captures), OQ1/OQ2 (short-circuit, capture.default scalars-only), PJVM (RFC 8259 types)
- Stream D: 21 runbooks migrated (all P1–P5 exit criteria passed)
- TESS-AMBIG-3..6: 4 ambiguities resolved (boolean ordering, array equality, field access on non-object/null)
- Optional chaining: `?.` / `?.[N]` proposed and EBNF applied (GIS-only scope)
- Runtime migration plan: Ratified 5 OQ-M decisions; Design repo = spec authority, Gert repo = implementation
- GIS miss semantics: Ken's engine correct; hard error on mandatory-miss confirmed
- GDP/keyword arbitration: `str`/`list`/`regex` dual-role confirmed; parser already correct

**Web Platform & Architecture:**
- A6 (MVP): App Service P1v3 $113/mo, Service Bus, Cosmos user_inputs, IUserInputGate (Choice/Text/Confirmation/FileUpload/Form)
- A8 (Enterprise): Durable Functions with WaitForExternalEvent<T>() for approval gates
- Campaign layer: tenant → campaign → audience → invitation → contract artifact → run
- White-label portal: Magic-link auth, per-tenant custom domains via Static Web Apps
- Declaration/consent: 22 scenarios mapped to 5 archetype patterns, 10 gaps identified

**Key Learnings:**
- Language-neutral canonical artifacts (schemas, vectors) must live in design repo, not runtime repos
- First non-Go port reveals hidden language dependencies in generated contracts
- Spec authority ≠ implementation ownership: design repo ratifies, runtime repos execute

## Key Milestones (Phase 1)

### Expression Language & Parser Gate (Streams A/B/F — COMPLETE)
- Delivered: GXL/GIS/GCP EBNF grammars (75 KB), normative LaTeX specs (§03a/03b/03c/03d), parser gate spec with 9 PLAN-* error codes
- Critical decisions ratified: OI-GIS-01 (\${ canonical escape), OI-GCP-02 (bare-root GDP captures), Portable JSON Value Model (PJVM), now() zero-arg stdlib function
- Grammar discrepancies resolved (scientific notation, len namespace, missing stdlib, GDP member access) — grammar is authoritative
- Parser gate: single-checkpoint, no-bypass, type-enforced contract (Phase 2 open: in-flight grammar upgrade strategy — OPQ-GATE-01)

### Stream D — Fixture Migration (Germán + Don-1)
- All P1–P5 exit criteria passed: 21 runbooks migrated (0 and/or remain, 0 ! prefix, 0 infix contains, 0 jq paths)
- Deferred-001 ({{ now }} in r04): Resolved by now() stdlib addition
- Stream D final status: ✅ COMPLETE

### Web Platform Architecture (A6 MVP → A8 Enterprise)
- **A6 (MVP):** App Service P1v3, $113/mo, IUserInputGate (TCS + SignalR), IApprovalGate (polling via Service Bus), Cosmos DB user_inputs container
- **A8 (Endgame):** Durable Functions with WaitForExternalEvent<T>() for approval/input gates
- Key insight: approval steps (long wait) drive A8 migration pressure; choice steps (short wait) are cheap in A6
- Campaign layer architecture: tenant → campaign → audience → invitation → contract artifact → run. White-label SWA portal, per-tenant domains, magic-link auth.

### Declaration/Consent Workflows
- 22 concrete scenarios across healthcare, insurance, finance, real estate, employment, government, education
- 5 archetype patterns: (A) Single-Party Informed Consent, (B) Two-Party Witnessed, (C) Proxy/Representative, (D) Eligibility-Then-Declaration, (E) Multi-Step Sectioned
- 10 gaps identified: signature capture, identity proofing, witness flow, document versioning, locale provenance, on-behalf-of, validity expiry, revocation, contextual PII, QTSP integration
- Proposed: DeclarationCollectedEvent, WitnessAttestedEvent, DeclarationRevokedEvent, IWitnessGate interface, UserInputKind.Signature
- Non-goals confirmed: GERT ≠ TSP, doesn't store biometrics, doesn't render legal text, doesn't enforce jurisdiction rules

### Team Directive
- Leslie uses he/him pronouns (as of 2026-06-04T17:15:45-07:00)

## Active Open Questions
- OQ-GATE-01: In-flight grammar version upgrade strategy (strict vs. sticky) — blocks Phase 2 plan storage
- Tess's ambiguities (TESS-AMBIG-3 through 6): boolean ordering, array equality, scalar field access, null field access — pending Barbara decision

## Session: 2026-06-04T02:50:37Z
Consolidated 8 inbox items. Ratified IUserInputGate (Choice/Text/Confirmation/FileUpload/Form), A6 architecture, campaign layer, white-label portal UX.

## 2026-06-05T07:28:56-07:00 — GIS Optional-Chaining Proposal

**Status:** PROPOSED — awaiting ormasoftchile review.

**Deliverables:**
- `design/gert/proposals/gis-optional-chaining.md` — full design covering grammar extension, semantics, conformance vectors, scope clarification, parse-time implications
- `.squad/decisions/inbox/barbara-gis-optional-chaining.md` — decision summary with headline picks

**Key Design Choices:**
- `?.` as opt-in optional path separator; `?.[N]` for optional bracket indexing (recommended IN)
- Missing path → empty string `""` (locked by user; identity element in concatenation)
- Full-tail short-circuit matching JS/TS semantics (no invented rules)
- GIS-only scope — GXL boolean and GCP captures have separate mechanisms
- `??` nullish-coalescing explicitly OUT (deferred)
- No migration of existing fixtures — opt-in only

**Open Questions Surfaced:**
- OQ-OC-1: Confirm `?.[N]` in scope
- OQ-OC-2: Confirm `${?.root}` illegal
- OQ-OC-3: Info-level audit trace on optional-miss?
- OQ-OC-4: Confirm `?.` doesn't propagate through stdlib calls

**Grammar Impact:** Additive — extends GDP production with `PathSegment` alternatives. No breaking changes.

## 2026-06-05T07:28:56.273-07:00 — GIS Optional-Chaining EBNF Applied

## Cross-Agent Coordination — Phase 2 Day 1 (2026-06-05T09:25:14.584-07:00)

### Incoming Notification: Parse-Gate Proposal Backlog Item

**From Coordinator (Phase 2 Day 1 Kickoff):**

**Q4 Decision:** Parse-gate OPQs (in-flight grammar upgrades, statically-reachable steps, warning trace, plan persistence, PLAN-* corpus) **Deferred** — you will write a separate parse-gate proposal **before Day 9** (Don's Day 9 runtime plan entry).

**Rationale:** 
- Parse-gate is governance-layer territory deserving a dedicated proposal cycle (similar to your GIS optional-chaining flow)
- Should not be buried in Day-1 runtime plan
- Day 9 is ~1 week out at current pace; timeline permits separate proposal cycle

**Scope of Parse-Gate Proposal (when you start):**

From Open Questions in Phase 2 Go Runtime Plan:

| Gate | Item | Decision Needed |
|------|------|-----------------|
| **OPQ-GATE-01** (BLOCKING) | In-flight grammar version upgrade | Option A (strict, disruptive) vs. Option B (sticky, requires version registry) |
| **OPQ-GATE-02** (blocking Day 9) | Statically reachable step set definition | Formal definition for cross-step capture validation (PLAN-009) |
| **OPQ-GATE-05** (blocking plan store) | Plan storage and re-validation policy | ValidatedPlan persistence model (where stored, eviction, forced re-validation) |
| **OPQ-GATE-03, 04, 06, 07** (lower priority) | Other parse-gate concerns | See `design/gert/sections/03d-parse-time-enforcement.tex` §8 |

**Timeline:**
- Backlog item: can start anytime before Day 9
- Don's Day 9 entry becomes: "Spec drafted in separate proposal cycle before implementation"
- Similar to your GIS optional-chaining proposal cycle flow

**Reference:**
- Phase 2 Go Runtime Plan: `design/gert/phase2-go-runtime-plan.md` § Open questions before/near Day 2, item 4
- Parser gate spec: `design/gert/sections/03d-parse-time-enforcement.tex` § Open Issues Phase 2
- Decision ratified in: `.squad/decisions.md` § Phase 2 Day 1 — Don's open questions resolved, Q4


**Status:** IMPLEMENTED — ratified EBNF delta applied to `design/gert/grammar/gis.ebnf` under GIS-only scope.

**Changes:**
- Added GIS-only GDP override with `GISPathSegment`, `OptionalDotAccess`, and `OptionalBracketAccess` productions.
- Added `OPTIONAL_DOT = '?.'` and `OPTIONAL_BRACKET_OPEN = '?.['` with longest-match lookahead and equal-precedence left-to-right path evaluation.
- Documented hard-error default vs. opt-in tolerance, full-tail short-circuit to empty string `""`, and optional bracket indexing.
- Added `${?.root}` to explicit rejections; root `IDENT` remains mandatory to prevent typo masking.
- Dropped implementation summary to `.squad/decisions/inbox/barbara-gis-grammar-ebnf-applied.md`.

**Scope Guard:** Did not edit `gxl.ebnf`, `gcp.ebnf`, `design/gert/sections/03b-interpolation-syntax.tex`, or `design/gert/conformance/`.

---

## 2026-06-04T20:14:36-07:00 — Stream E Removed (Scribe notification)

**Status Update:** User directive executed. Stream E (migrator tooling) removed entirely per scope decision. Rationale: GERT has no production runbooks, so migration solves a non-problem.

**Impact on Phase 1 Plan:**
- Stream E is completely removed from the plan
- Phase 1 closes with Streams A/B/C/D/F only
- C# runtime can begin once spec is frozen (no migration dependency)

**Cleanup Note:** Grammar files (gis.ebnf, gcp.ebnf) retained migration-tool comments as "do-not-touch" per directive. Team should decide whether to remove these in a grammar-owned follow-up.

**Decisions:** Merged to `.squad/decisions.md` (timestamp 2026-06-04T20:14:36-07:00)

## 2026-06-05T18:33:34-07:00 — Runtime Migration Plan

**Status:** PROPOSED — awaiting ormasoftchile review.

**Deliverables:**
- `design/gert/proposals/runtime-migration-plan.md` — full 8-section plan covering current state survey, gap analysis, phased migration strategy, risk register, cross-repo coordination, effort estimates, non-goals, and open questions
- `.squad/decisions/inbox/barbara-runtime-migration-plan.md` — decision summary with headline picks

**Key Design Choices:**
- Parallel-package strategy (`internal/eval/` alongside `internal/expr/`) with interface adapters
- 8 phases (A-H) ordered by dependency; critical path ~12-18 days serial
- Build-tag feature flag for zero-overhead compile-time switch
- Cutover requires: 264/264 vectors green, 22 fixtures execute, perf ≤2x, 1-week soak
- Recommend cherry-picking 4-day sketch (commits 97ce48b..5c550c0) as starting material
- Git submodule for conformance vector distribution (pending visibility decision)

**Blast Radius Documented:**
- 12 executor types + 2 wiring points + engine config struct
- ~58 test cases that depend on expr-lang or text/template semantics
- Current capture mechanism is keyword-enum, not path-based (full replacement needed)

**Open Questions Surfaced:**
- OQ-M1: Conformance vector distribution mechanism
- OQ-M2: Sketch cherry-pick confirmation
- OQ-M3: Build tag vs. runtime flag
- OQ-M4: Breaking syntax change communication
- OQ-M5: Team size (1 vs. 2 engineers)

**Risks Highlighted:**
- Semantic divergence (expr-lang behaviors GXL forbids)
- Unresolved spec ambiguities blocking implementation
- Capture path upgrade requires structural output wrapping

## 2026-06-05T22:09:06-04:00 — Runtime Migration Plan RATIFIED

Ratified all 5 OQ-M decisions encoded by ormasoftchile. Updated proposal status to `RATIFIED — 2026-06-05`; replaced "Open Questions" section with "Ratified Decisions" in `design/gert/proposals/runtime-migration-plan.md`. Identified DRIFT-DETECTION-001 as a new Phase A scope item (sync script + CI `make verify-vectors` target to prevent vendored vector drift). Memo dropped to `.squad/decisions/inbox/barbara-runtime-migration-ratified.md`. This design repo's role is now spec authority + ambiguity arbiter; active implementation hands off to `ormasoftchile/gert` runtime squad.

## 2026-06-05T22:31:50-04:00 — OQ-M5 Wording Correction: Single Squad Directive

Corrected OQ-M5 language in `design/gert/proposals/runtime-migration-plan.md` (line 491) per single-squad directive (commit 541d194). Changed: "second backend agent's assignment happens in the `ormasoftchile/gert` runtime squad" → "second backend agent joins **this squad** (`gert-private`), both operate in gert repo during Phases A–H, then return to this squad for next assignment." Memo dropped to `.squad/decisions/inbox/barbara-m5-single-squad-correction.md` with exact before/after and explicit instruction for Scribe to update the dated decisions section.

## 2026-06-05T22:31:50-04:00 — TESS-AMBIG-3..6 Arbitrated; Phase 1 Corpus Fully Pinned

Arbitrated all four open ambiguities from Tess's Stream C Day 2 memo. Introduced two new error codes (`GXL-TYPE-005`, `GXL-PATH-004`). Closed all 4 TBD conformance vectors (TV-GXL-EVAL-033, TV-GXL-EVAL-086, TV-GXL-PATH-022, TV-GXL-PATH-023); added 3 companion vectors (TV-GXL-EVAL-091, TV-GXL-EVAL-092, TV-GXL-PATH-036). Patched `gxl.ebnf` (§5.2, §5.3, §6.2, §6.3) and `03a-expression-language.tex` (§Type System, §Path Access, §Error Catalog). Phase 1 GXL conformance corpus now has zero TBD entries. Memo dropped to `.squad/decisions/inbox/barbara-tess-ambig-3456-arbitration.md`. Flagged `GXL-PATH-004` and `GXL-TYPE-005` for Phase A awareness (Ken/Don evaluator dispatch split).

## 2026-06-05 TESS-AMBIG-3..6 Arbitrated

Arbitration complete. Four TBD vectors pinned with two new error codes (GXL-TYPE-005, GXL-PATH-004) and one extended code (GXL-TYPE-001):
- **GXL-TYPE-005:** Boolean ordered comparison forbidden
- **GXL-PATH-004:** Field access on non-object/null
- **GXL-TYPE-001:** Extended to forbid list/object equality

Spec sections updated: `gxl.ebnf` §5.2/5.3/6.2/6.3; `03a-expression-language.tex` Type System, Path Access, Error Catalog. Corpus final: 211 vectors, 0 TBD.

Phase 1 exit gate ✅ COMPLETE.

## 2026-06-05T23:12:22-04:00 — GDP/Keyword Dual-Role Arbitration (commit `fdd14db`)

**Status:** Spec arbitration complete. Resolved TV-GXL-PARSE-028 ambiguity via Option A.

**Decision:** Namespace keywords `str`, `list`, `regex` have dual role: (1) NamespaceCall heads when followed by `.`, (2) valid GDP root identifiers when NOT followed by `.`. Carve-out scoped to these three keywords only.

**Verification:** All gate vectors pass. Regressions: zero.

**Spec updated:** `design/gert/grammar/gxl.ebnf` (§2/§3 notes) and `design/gert/sections/03a-expression-language.tex` (keywords table, identifiers, GDP, NamespaceCall sections).

**Parser impact:** Don's PR #10 already implements correct behavior — no code change required.

## 2026-06-06T01:40:00-04:00 — Phase G Arbitration: GIS Miss Semantics (commit e819895)

**Status:** RATIFIED — Option A (mandatory-miss-as-error).

**Decision:** Ken's GIS engine is correct as shipped. Legacy `missingkey=zero` was an under-spec'd implementation leak, not a contract. The spec already mandates hard errors for mandatory path misses.

**Spec Authority:** `gis.ebnf` §4.3 + `03b-interpolation-syntax.tex` §Unresolved Variables both forbid silent empty-string substitution by default. All 15/15 GIS-PATH conformance vectors already pass and expect hard error on mandatory-miss.

**Phase H Impact:** Zero. Proceed with planned hard cutover (delete legacy engine, no compat shim, no CLI flag, no parser change).

**Migration:** Authors who relied on missing-key-as-empty-string use optional chaining (`?.`) or capture defaults for soft-miss tolerance.

## 2026-06-07T19:14:39-07:00 — Runbook v1 Schema Canonical Source Decision

**Status:** DECIDED — Option A (design repo is canonical home).

**Decision:** `design/gert/schemas/runbook.v1.schema.json` is the single source of truth for runbook structural validation. Hand-authored, language-neutral, consumed by all surfaces (IDE, CI, portals, future ports).

**Key trade-offs considered:**
- Go structs (Option B) are an implementation, not the source — they'd force C#/TS ports to reverse-engineer Go-specific output
- Extension-local schema (Option C) guarantees drift across three+ consumers
- LaTeX→JSON Schema generation doesn't exist; Go struct→schema generators produce Go-biased output — hand-authoring wins for precision
- Design repo already hosts all normative artifacts; DESIGN ONLY directive explicitly supports this class of artifact

**Boundary established:** Schema = structural gatekeeper (types, required fields, enums, patterns). Runtime = semantic gatekeeper (expression evaluation, path resolution, cross-step integrity, governance). Schema-valid ≠ executable.

## Learnings

**Pattern: "Single normative grammar + tool-generated/hand-authored artifacts for every consumer surface"**

When a contract serves multiple language ecosystems (Go, C#, TS) and multiple consumer surfaces (IDE, CLI, CI, web portal), the canonical artifact MUST live in a language-neutral design repo. Implementation repos consume; they never own the contract. This pattern applies to:
- Conformance vectors (already established via DRIFT-DETECTION-001)
- Runbook JSON Schema (this decision)
- GXL stdlib manifest (future — same pattern should apply)

**Anti-pattern avoided:** "Runtime structs generate the schema" sounds DRY but creates a hidden language dependency. The first non-Go port would either duplicate the generation or consume a Go-biased artifact with Go type names leaking through. Paying the cost of hand-authoring once saves every future port from fighting a translation layer.

## 2026-06-07T19:28:42-07:00 — Runbook Schema Arbitration & Spec Question Gate

**Status:** DECISION RATIFIED + ARBITRATION NEEDED

**Runbook v1 Schema Home — FINALIZED (2026-06-07T19:14:39-07:00)**
- Canonical location: `gert-private/design/gert/schemas/runbook.v1.schema.json` (hand-authored, language-neutral)
- Ownership: Barbara (architecture), Edith (day-to-day), review gate required for all schema changes
- Consumer pattern: All surfaces (gert-vscode, gert-tui, runtime, future C#/TS) reference canonical location
- Status: Unblocks gert-vscode Phase 2 (YAML schema, IDE integration)

**Runbook Schema Authoring Complete (Edith)**
- PR #7: Initial schema authored, validates all 23 example runbooks
- 5 spec questions raised (SQ-001 to SQ-005), now in queue for Barbara arbitration
- No blockers — schema structure sound; questions are design clarity only

**Spec Questions Awaiting Arbitration:**
| # | Question | Blocks |
|---|----------|--------|
| SQ-001 | Is `name:` field on Step a legacy alias for `title:`, deprecated, or missing? | Nothing immediately; (a) safe default chosen |
| SQ-002 | What fields does `extension` step type carry? Payload fixed or open? | gert-vscode hover/completion for extension steps |
| SQ-003 | Is `list` a normative type alias for `array` in input declarations? | Normative enum definition in spec §Input Declarations |
| SQ-004 | Should runbook `id` pattern differ from step `id` pattern? | ID validation strictness for gert-vscode |
| SQ-005 | Are `iterate` and `parallel` valid step `type` values? | Would affect schema if embedded inside step wrappers |

**Recommendation:** Arbitrate all 5 in single session before Edith updates schema/spec sections.

**Conformance Schema Naming Clarity (Tess)**
- Evaluation complete: collision risk between test-vector schema and runbook schema
- Recommendation: Rename `design/gert/conformance/schema.json` → `design/gert/conformance/vector.schema.json`
- Blast radius: 13-16 files (1 code change, 12 documentation/comment updates)
- Status: Awaiting Barbara ratification; atomic PR scope documented
- Timeline: 1-2 hours to execute once approved

## 2026-08-09T14:09:01-07:00 — Tool Packages Architecture Ruling (AR-TP-1..10)
Issued binding architecture ruling package for the GERT Tool Packages MVP:
`.squad/decisions/inbox/barbara-tool-packages-architecture-ruling.md`.
Final calls: `requires:` is canonical (toolPackages rejected, PKG-020); toolRefs binds/narrows only;
two-phase catalog (freeze then bind) with 5 tiers, same-tier collision = hard error, undeclared
cross-tier shadowing = PKG-022; uniform higher-tier-wins across §05/§06 (both current texts were
wrong and inconsistent); explicit exports (no implicit package scanning); SemVer 2.0.0 with a
caret/tilde/relational conjunctive-only constraint subset (no ||, no wildcards, no partials);
runbook-backed action substitution declared tool-relative with exact I/O signature match and
governance composition that can only narrow (OR approval, union denies, intersect allows);
POSIX-only package-internal paths with realpath containment and reparse-point/symlink escape as a
hard error; sha256sum-style order-free package + catalog digests; resume refuses on digest drift
unless explicitly overridden and audited; lexical tool-name scoping across includes with one global
package set per run; 28 PKG-* codes + PLAN-010 + 3 warnings; ≥46 conformance vectors in 7 new
categories. Deferred as explicit non-goals: remote catalogs, publishing, transitive deps, aliases,
action allowlists, non-tool assets, multi-version coexistence.
Next: Edith authors; I re-review §5 substitution and §7 digest text before the phase gate.

## 2026-08-09T15:45:00-07:00 — Tool Packages Gate Review #3 (final, narrow): APPROVED
Third and final architecture gate on the Tool Packages MVP. Entry:
`.squad/decisions/inbox/barbara-tool-packages-gate-review-3.md`.
Reviewed Ken's independent S1/S2 sweep (Edith and Don locked out for this cycle).
S1 confirmed resolved: §03 `toolRefs` no longer teaches `alias:` (now PKG-021) or the deleted
`tools/<name>.tool.yaml` default-discovery rule; it states the negation and cross-references the
ratified frozen-catalog/tier model in §06. S2 confirmed resolved: all three mapping-shaped
`actions:` listings converted to the list form, `sensitive_inputs` preserved in §08; corpus-wide
scan shows zero mapping-shaped `actions:` blocks remain. Ken's four optional cleanups verified
text-only: no schema, error code, grammar production, or runtime semantic changed; AR-TP-1..10 and
R1-R15 stay closed. Re-ran validation myself rather than trusting the matrix: verify_corpus 280/280,
5 JSON Schemas valid, gert-package.yaml and drain-node.yaml 0 errors, label/ref closure 293/0

## Learnings

### 2026-08-15T12:58:20-07:00 — Dynamic Include Architecture Contract

**Delivered:** `.squad/decisions/inbox/barbara-dynamic-include-contract.md` — full 12-section contract gating four parallel implementation streams for runtime-resolved (dynamic) runbook includes.

**Key architecture decisions:**
1. **Frozen catalog is inviolable.** A dynamically included child MUST NOT pull in new packages at execution time. All dependencies must already be in the Phase C frozen catalog. This preserves the `catalog/frozen` digest as the supply-chain integrity anchor.
2. **Dry-run does NOT attempt resolution.** Dynamic includes remain unresolved leaves in dry-run/preview, identical to each other. The `runbook_ref` template contains variables only populated at execution time; attempting resolution would produce misleading results.
3. **`DINC-*` error code family** (13 codes + 1 warning) introduced for dynamic include failures, following `errkit` conventions.
4. **`--allow-package-drift` reused** (no new flag) for resume drift on dynamic include pins, since dynamic include resolution drift is a sub-case of package drift.
5. **Variable scope: inherit full parent scope** (same as static includes). Stricter isolation deferred to a future `scope: isolated` opt-in.

**Existing seams discovered:**
- `flowwalk.Walker` is the sole include traversal engine; planner and preview both delegate to it. Dynamic includes get `BeforeInclude → load=false` (opaque at plan time).
- `SubStepParent` carries `NestDepth` (visual) but has no structural include chain — the contract adds `IncludeChain []string` for runtime cycle/depth checks.
- `IncludeExecutor` has exactly two paths today: eager (`spec.ResolvedSteps`) and lazy (`spec.LazyRunbookPath` via `LazyRunbookLoader.LoadFlow`). Dynamic includes add a third path via `DynamicIncludeResolver.Resolve`.
- `RegistryConfig` wires all executor dependencies; `DynamicIncludeResolver` follows the same optional-dependency pattern as `LazyRunbookLoader` and `SubstitutionParser`.
- `PlanMetadata` already carries `CatalogDigest` and `PackageDigests`; dynamic include pins extend it with `DynamicIncludes []schema.LockedDynamicInclude`.
- `schemas/runbook.schema.json` `IncludeConfig` has `"required": ["runbook"], "additionalProperties": false` — must change to a `oneOf` for static vs dynamic.

**Critical file paths:**
- `pkg/schema/steps.go` — `IncludeConfig` (WIP complete, adopted)
- `pkg/pkgcatalog/catalog.go` — `RunbookEntry`, `RunbookByQualified`, `RunbookByBare`
- `pkg/trace/event.go` — existing event kinds + new `include/resolved`, `include/notFound`
- `pkg/errkit/errors.go` — `DINC-*` codes to add
- `internal/executor/include.go` — primary executor modification site
- `internal/executor/registry.go` — `RegistryConfig` wiring
- `internal/adapter/lazyloader.go` — template for `dynamicloader.go`
- `pkg/engine/run.go` — `PlanMetadata` extension
- `pkg/schema/packagelock.go` — `LockedDynamicInclude` to add
- `schemas/runbook.schema.json` — `oneOf` schema change
unresolved; r23's lone error is the known corpus-wide `apiVersion: runbook/v2` issue.
Approved with no reviser designated. Spec is ready for Cristian's review.
Remaining work is deferred Tess-owned conformance (`tv-pkg-resolve.yaml`, >=46 vectors, now fully
unblocked) plus four non-blocking pre-existing tickets and one typographic nit (a `\S\ref` leaking
verbatim inside a `minted` comment at `03-schema-vnext.tex:184`).
Learning: three narrowing gates worked. Gate 1 (15 blockers) was design; gate 2 (2 blockers) caught
chapters whose meaning this workstream changed but never edited — the highest-yield question at a
gate is not "is the new text right" but "which untouched text did the new text just falsify."

## 2026-08-09 — Tool Packages MVP runtime, final implementation gate (2nd pass): APPROVED
Reviewed David's independent revision of the runtime working tree against my own rejecting first
pass. Verified all six blockers in the code that runs, not the summary: plan-time substitution
validation now runs unconditionally in the shared `runWithMode` before Plan/Start, so dry-run and
unreachable-by-`when:` steps are validated and cycles/depth are decided by static DFS frames;
digest closure covers `execute.path` substitutes and package-internal includes; tier-1 catalog
entries carry the package digest; the resume manifest is authoritative so a removed package is
PKG-001 by name; `outputs.<name>` is implemented end-to-end with the step-context check at plan
validation; the include closure fails closed with typed PKG-017 instead of resolving dynamically.
All nine lower-severity fixes verified. Protected files byte-identical, no design file touched,
tree left dirty and uncommitted, build clean, tests green apart from one unrelated `internal/serve`
timing flake that passes 5/5 on re-run. Approved with no reviser designated; ready for Cristian.
Learning: the highest-yield thing I did this pass was not re-reading the diff — it was noticing
which claims the test suite could not actually prove on this platform. `maxLinkHops` was "fixed"
and its two tests skipped on Windows, so the only evidence for §3.7 was the code reading right. I
built a real 10-link chain and drove it through the exported `pkgpath.Resolve` from a scratch
module outside the product tree: PKG-008 at 10 hops, clean at 8. A skipped test is an unverified
claim wearing a passing suite's colours; find those first and verify them by hand.

## 2026-08-10T13:25:10-07:00 — `enum` String Constraint MVP: Architecture Ruling (AR-ENUM-1..15)
Issued the binding ruling for the proposed `enum` MVP:
`.squad/decisions/inbox/barbara-enum-constraint-mvp-architecture-ruling.md`. APPROVED with three
binding corrections: (C1) `enum` forbidden on `type: secret` and the member list redacted on any
`redact: true` / `sensitive_inputs` declaration — an enum on a redacted field is a value oracle and
"expected one of [...]" in an error message defeats the redaction sitting next to it; (C2) "exact
enum match across package mocks" discharged as a required conformance vector class, not a runtime
check, because only one binding is loaded per run and inventing lock fields to fake in-run
detectability is scope creep; (C3) no `tool.v1.schema.json` — §06 stays the sole normative source
for `.tool.yaml`.
Resolved: four declaration sites only (tool `args`/`outputs`, runbook `inputs`/`outputs`);
string-only, enum is a constraint not a type; YAML 1.2 core-schema string resolution with no
coercion (`enum: [yes, no]` fails loudly); no empty/whitespace/edge-whitespace members; NFC
uniqueness; asymmetric normalisation (authors MUST be NFC, external candidates ARE normalised);
codepoint equality, no case folding, case-only-distinct legal with ENUM-W001; two orderings stated
separately (declared = presentation/plan/UI, canonical codepoint = comparison) with SET equality
for contract identity so reordering is not a break; enum constrains values not presence; plan-time
covers well-formedness + default + literal binds + PKG-013 signature, runtime covers everything
interpolated with no partial evaluation; single-cause reporting (type error beats enum error);
replay does not bypass; enums declaration-local across includes with no intersection semantics;
metadata carried once in the ValidatedPlan/validation record and never in per-step event payloads;
one new ENUM-001..009 family plus ENUM-W001, with substitution mismatch deliberately kept inside
PKG-013 rather than a new code; grammars and schema apiVersions untouched; 12 ratified non-goals.
Found four adjacent drifts while reading the objects Edith will edit; ruled D1 (stale
`from: captures.*` output prose contradicting the schema's `value:` GCP form) fixed in the same PR
because the enum output rules reference that exact object, D2/D3 (schema `Input` missing
`pattern`/`example`, `from` enum contradicting §03) as separate tickets so they do not ride along.
Next: Edith authors in dependency order (03d error catalog first); Tess writes `tv-enum.yaml`,
>=54 vectors in 8 categories, blocked on the code numbers; I re-review §8, §10 and the catalog text
at the gate.
Learning: the sharpest question on a "just add a keyword" proposal is where the keyword becomes
*visible*. Enum looked like a pure validation feature until I traced it into traces, JSON output and
error messages — and on a redacted field the constraint metadata leaks exactly what the value
redaction was protecting. Constraints are data about data; ask who gets to read them before asking
whether they are enforced correctly.

## 2026-08-10T14:50-07:00 — enum spec final gate (Gate 3): APPROVED

Read Don's B6 fix under author lockout (Edith and David both out). One line at
`03-schema-vnext.tex:586`: the canonical `outputs:` shape block's `value:` placeholder went from
the GIS template `"${captured_value}"` to a bare GCP path `"captures.<captureName>"`, comment
preserved. Exactly the remedy specified, and only that — `03-schema-vnext.tex` is the only design
file touched since the re-gate, `captured_value` is now a zero-hit grep tree-wide, `tv-enum.yaml`
byte-identical.
Re-ran the sweep *structurally* rather than by grepping the literal — walked every `outputs:`
mapping in every YAML/JSON/TeX file under `design/gert/` and enumerated the `value:` keys inside
them. 11 sites, all bare GCP or literal-free; the other 208 `value:` occurrences in the tree are
`expected.value`, `evidence[].value` and collector `options[].value`, correctly untouched. One
shape for one field, tree-wide: D1/B1/B6 closed.
Integrity: corpus `OK 423/423`; all five JSON Schemas parse; `$defs.Input`/`$defs.Output` match
their shape blocks element-for-element and in order; `$defs.Output` genuinely has no `default`,
which is what makes the B3 text true rather than merely asserted; 58 vectors, no duplicate ids,
`ENUM-MOCK` present; zero duplicate labels, zero dangling refs, zero inline `\paragraph{`.
Authorised runtime implementation in `C:\One\OpenSource\gert`, bounded by the ratified error
family, the four sites, single-cause reporting, C1 redaction and frozen apiVersions, with
`tv-enum.yaml` as the acceptance gate — run, not edited. D2–D5 stay open tickets; D2 explicitly
NOT folded in despite being the same *smell* as B6, because it is a different class and a
different cause.
Learning: the check that missed B6 was the right check for the wrong half of the line. A grep for
the stale *key* (`from: captures`) can never see a wrong *value* under the correct key. When a
ruling says "one shape for this field", verify it by enumerating the field's occurrences
structurally — enter the parent block and list what is declared inside it — not by searching for
the shape you already know is wrong. You only find the drift you can already name; the sweep has
to be over the field, not over the defect.

📌 Enum Client Parity Session (2026-08-11T09:45:19-07:00 → 2026-08-11T11:42-07:00):
- Issued AR-CE-1..10 architecture ruling (schema unity, DTO carriage, selector/fallback pattern, validation authority, error code mapping, redaction, trace, parity tests)
- Gate 1: Identified B-1..B-6 blockers (DTO delivery, renderer, GUI form, TUI wiring, warnings, parity matrix); locked out Don/Ken; assigned David independent revision
- Gate 2 (Final): Verified all F-1..F-11 fixes via real binaries (graphjson DTO, GUI form, TUI selector, error codes, warning visibility); approved READY FOR PRODUCTION; no architecture reopened; T-TUI-WARN-VIEWTEST mandatory follow-up


📌 Dynamic Include Rulings Session (2026-08-15T12:58:20-07:00):
- Arbitrated all 14 of Tess's open questions (B-1..B-14) for the dynamic include executor stream
- Key rulings: on_not_found:continue covers ONLY DINC-002 (not-found), never DINC-003 (ambiguity) or DINC-013 (file-missing); governance allow_commands uses intersection semantics with formal non-widening proof; deny lists are set-union (deduplicated); child tier-4 toolRefs shadow tier-1 via normal BindFile precedence (no new error); expand: field is schema-rejected (not silently ignored) when runbook_ref present; capture: missing source key is silent omission (consistent with static includes); B-1/B-2 homoglyph/NFC already handled by ValidateRenderedRef safe-id profile; B-3/B-4 are GIS/GXL grammar questions resolved by existing parser errors
- New sentinel required: DINC-013 (resolved file missing from disk) — Ken must add
- Learning: the seam between "identity not found" and "identity found but infrastructure broken" is load-bearing for on_not_found semantics. Collapsing them under one flag would let environment corruption masquerade as a missing publication. Always decompose "not found" into its causal categories before exposing a user-facing skip/continue control.


📌 DINC-007 Arbitration Session (2026-08-15T14:13:00-07:00):
- Ruled B-15: DINC-007 (extra with-key not declared in child inputs) is a WARNING, reclassified to DINC-W007 with class "DINC-W"
- Shipped errkit sentinel was wrong (class "DINC" = fatal); executor was already correct (non-fatal trace emit)
- Contract §11 code block was the error source; prose table was correct all along
- Learning: when a contract has both a prose table and a code block expressing the same fact, they MUST be cross-checked at authoring time. The code block feels authoritative to implementers ("it's code, it must be right"), so any inconsistency will be resolved in favour of the code block downstream. Always proof-read the code block against the prose table before ratifying.


📌 Final Dynamic Include Arbitration (2026-08-15T15:07:00-07:00):
- B-16: DINC-009 is unreachable dead code (evaluator stringifies all values). Retained as defensive; TV-DYN-TMPL-005 permanently skipped. Coercion validation is a future enhancement.
- B-17: Capabilities field ruled out of scope — GovernanceConfig doesn't carry it today, and adding a new governance primitive requires its own design cycle. §6 amended. Widening is correctly a warning (DINC-W001) per existing §6.4.
- B-18: Static include governance gap is a pre-existing security defect. Ruled out of scope — fixing it as a side-effect of dynamic includes risks production regressions. Recorded as separate defect for tracked follow-up.
- Learning: when a spec names a concept that doesn't exist in the schema yet, that's a forward-reference, not an implementation requirement. Specs must distinguish "this is how it WILL work" from "this must ship NOW." Missing that distinction creates phantom blockers where implementers can't land a vector because the platform substrate isn't there. Always tag forward-references explicitly in the contract so vectors can be written as "pending platform support" rather than "failing."


U+1F4CC DEF-004 / when-field Arbitration (2026-08-15T15:31:00-07:00):
- B-19: when: field is parsed, validated, stored, and silently ignored at runtime — platform-wide, not dynamic-include-specific. Ruled out of scope. Blast radius too large to fix as a side-effect.
- Correct authoring pattern for conditional dynamic includes: use ranch steps with condition: arms, not when: on individual steps. BranchExecutor + ConditionEvaluator is the only working conditional mechanism.
- Tess's passing ICM e2e tests already use branch/condition (not when:), confirming the pattern works.
- Contract §9 amended to recommend ranch/condition: and note the when: gap.
- Learning: a validated-but-unevaluated field is strictly worse than a rejected one. When the parser accepts syntax and the runtime ignores it, the author gets positive feedback that their intent is well-formed while it silently does nothing. This is the same class of defect as B-14's xpand: on dynamic includes — but B-14 was caught at design time and schema-rejected, while when: slipped through because it predates this feature. The general principle: every field the parser accepts must have a corresponding runtime consumer, or the parser must reject it. "Parse but ignore" is never acceptable for control-flow fields.


U+1F4CC B-20 Amendment to B-19 (2026-08-15T15:42:00-07:00):
- Amended B-19 after David's verified analysis revealed three distinct when: fields (not one), inconsistent evaluation (collector works, step/include don't), and an official example teaching the broken pattern.
- Code deferral upheld — blast radius argument strengthened by evidence that stale conditions exist in shipped examples.
- NEW requirement: documentation/schema remediation before ship. Schema gets description fields noting the limitation; example gets a warning comment. No code changes, no execution path risk.
- Owner: David (wrote the tracking record, has full context, is free).
- Learning: deferring a code fix is a legitimate scope decision, but it creates a secondary obligation: every artifact that advertises the deferred capability must be marked. The defect tracking record is necessary but not sufficient — it catches engineers who read the tracking file, while the schema and examples catch engineers who never will. Remediation must meet users where they encounter the broken surface, not where we file the bug.
- Learning: "the field is unimplemented" and "the field works in one place and silently doesn't in two others" are categorically different defects. The latter creates a trust transfer — "I saw it work on collectors, so it must work on steps" — that the former never could. When triaging a validated-but-inert field, always enumerate which consumers exist and which don't, because partial implementation is more dangerous than no implementation.


U+1F4CC Feature Review Gate — Dynamic Includes (2026-08-15T16:14:00-07:00):
- Ruled B-21: require_approval auto-approve in non-TTY is acceptable as designed (no human to prompt = auto-approve is only sane default). Requires schema description only. Distinct from when: (which is validated-then-ignored regardless of context).
- Issued APPROVE-WITH-CONDITIONS: three non-blocking conditions (B-15 errkit reclassification, B-20 schema descriptions, B-21 require_approval description). All documentation/classification; no execution-path changes.
- All 11 user requirements verified MET. Frozen-catalog invariant structurally guaranteed (no mutation API, no fetch machinery reachable from resolver). Runtime cycle/depth enforcement compensates for planner's inability to see dynamic targets.
- Learning: "inapplicable because the precondition is absent" (require_approval in non-TTY) is categorically different from "validated then ignored" (when:). The distinction is whether the feature's semantic contract can possibly be honoured in the context. When no human exists to approve, the field's promise is vacuously true — not violated. When a condition expression exists and the runtime simply doesn't evaluate it, the promise is actively broken. This distinction governs whether documentation-only or code-fix is the appropriate remediation.
- Learning: a multi-stream feature review should verify *structural guarantees* (what the types make impossible) over *behavioral assertions* (what the code happens not to do). The frozen-catalog invariant holds because DynamicIncludeResolver has no s.FS and no HTTP client — not because there's a check that says "don't fetch." Structural guarantees survive refactoring; behavioral assertions rot.

---

## Session: Dynamic Runbook Includes — Phase 4 Wrap-up (2026-08-15)

**Agents:** Barbara (arch), Tess (test), Ken (stream1), Don (streams2-3), David (stream4)

**Outcome:** Feature complete, approved, ready for merge (APPROVE WITH CONDITIONS, all conditions satisfied)

**Key achievements:**
- 5 defects found and fixed (DEF-001..006 via Tess real-CLI exercise + implementation streams)
- 21 architecture rulings (B-1..B-21) issued and enforced
- 3 deferred items formally captured (B-18, B-19/B-20, B-21)
- 26/29 conformance vectors pass; 3 skips permanent with ruling
- Cross-repo coordination: design repo (spec authority) ↔ runtime repo (implementation)

**Coordination patterns that worked:**
- Conformance corpus authored before implementation as specification
- Real CLI exercise (not hand-reading) revealed every gap
- Plausible descriptions can be materially wrong; rely on code review
- Formal deferral of gaps better than hidden work

**Status:** This session's work fully merged into .squad/decisions.md. Five orchestration logs recorded. Session log captures defining lesson: real verification beats narrative.


# barbara

Current role: Implementation engineer.

Session: Dynamic Runbook Includes (2026-08-15)
- Feature complete and approved
- Ready for merge

Detailed history: .squad/agents/barbara/history-archive.md


U+1F4CC MCP HTTP Transport Contract Session (2026-08-16T00:55:39Z):
- Authored binding contract for Streamable HTTP MCP support (mcp-http transport mode).
- Key architecture finding: the transport seam ALREADY EXISTS. pkg/tool.ToolTransport interface + DefaultToolRuntime.Invoke switch dispatch = the integration point. No refactor of existing stdio MCP needed. New MCPHTTPTransport implements the same interface independently.
- Issued B-22..B-26: HTTPS-only (B-22), protocol version pin at 2025-03-26 (B-23), credential redaction (B-24), no URL allow-list (B-25), governance is transport-agnostic (B-26).
- Error class: new MCP class with 9 codes (MCP-001 through MCP-009) covering validation, protocol, session, transport, and auth failures.
- Shared response types extracted to mcp_types.go so stdio and HTTP transports parse identically — this is the structural guarantee of Requirement 4 (identical ToolResult regardless of transport).
- Learning: when a feature request says "add X that works exactly like Y," the first question is whether there's a seam above Y that X can plug into. If there is, the feature is an implementation of an existing interface (low risk, no refactor). If there isn't, the feature requires creating the seam first (high risk, refactor). Always verify the seam before writing the contract — it determines the entire stream topology.
- Learning: the security posture question "does this need an allow-list?" depends on whether there's a trust boundary between the author and the runtime-resolved value. Static authored URLs (same trust as command:) → no allow-list. Runtime-resolved identities (dynamic includes) → catalog as allow-list. The trust model determines the control, not the perceived "dangerousness" of the operation.

Learning (B-32, 2026-08-15T18:06:00-07:00): Audience scoping on bearer tokens constrains which service will ACCEPT the token, not who may PRESENT it. A token sent to a malicious host can be replayed against the legitimate audience. "The attacker can't use the token at their service" is irrelevant when the attack is forwarding it to YOUR service. Bearer = possession is authorization. When reasoning about credential-exposure risk, model the attacker's ability to REPLAY, not their ability to CONSUME. This invalidated B-25/B-27's core premise and required the host-restriction narrowing in B-32.

📌 MCP HTTP Final Review Gate (2026-08-15T19:18:27-07:00):
- Issued APPROVE (unconditional) for Streamable HTTP MCP transport. All 5 requirements MET.
- B-32 fully realised: allowed_hosts required, MCP-010/011 static, MCP-012 runtime fatal, MCP-013 redirects blocked. One definition for host matching — no drift between static and runtime checks.
- Minor Req4 divergence noted but non-blocking: HTTP returns ToolResult{ExitCode:1} on IsError while stdio returns nil. Error message is identical; callers check err first. Observable behavior from runbook author's perspective is indistinguishable.
- Cross-feature (dynamic-include + MCP HTTP) verified safe: frozen catalog is the real trust boundary; allowed_hosts is defense-in-depth against partial compromise. A fully compromised package author can bypass allowed_hosts — but can also issue commands directly against the real endpoint, which is a strictly more powerful attack.
- Learning: when reviewing a token-attachment control, trace the full attack chain to its logical conclusion. If the attacker who bypasses your control can do something strictly worse via a path you can't control (declaring legitimate-looking commands against the real endpoint), the control's ceiling is "defense against carelessness and partial compromise." That is still worth building — most real incidents are partial, not total — but the review should state the ceiling explicitly rather than implying the control prevents all exploitation.
- Learning: stale test-file header comments (DEF-011/012/013 noted as open) vs. user's verified claim (all passing, defects fixed) — trust verified state over file comments. Test headers are snapshot documentation; they rot within hours during active multi-stream development.

## 2026-08-16T02:33:36Z — MCP HTTP Transport Feature — Team Session Orchestration

**Feature:** Native Streamable HTTP MCP transport (transport.mode: mcp-http)
**Outcome:** ✅ COMPLETE — All requirements met, final review gate APPROVED (unconditional)

- Completed assigned stream work
- Collaborated on binding architecture contract
- Participated in team orchestration
- All deliverables verified and tested
- Ready for production merge

## 2026-08-16 — Team Orchestration Session: Runtime Portability Evaluation

**Session:** Scribe coordination session with barbara, don, david  
**Task:** Evaluate and document SQL Live-Site Operations "Runtime Portability for Gert Runbooks" implementation request

**Key outputs from cross-agent session:**
- Barbara: Architectural evaluation published to decisions.md (ACCEPT-WITH-MODIFICATIONS verdict)
- Don: Ground-truth verification against source — all claims accurate except run-gert.ps1
- David: Integration critique identifying three blockers + one phase-ordering risk

**Cross-finding consensus:**
- All three agents independently flagged schema non-goal contradiction as critical
- All three flagged phase estimation (Phase 2 estimate dependent on unanswered OQ-2) as unreliable
- David flagged phase-ordering risk (Phase 3 semantics required but Phase 1 ships HTTP with timeout hazard)

**Deliverables merged to decisions.md:**
- 3 inbox files archived
- 3 orchestration logs written
- Session log written to .squad/log/

**Team notation:** Decision-making requires Cristiano's input on schema non-goal contradiction and OQ-2 resolution. All three architectural evaluations recommend identical remedy: accept additive schema changes, answer OQ-2 in Phase 0, implement tiered-preflight with Tier 0 static mandatory.

## Learnings

📌 Runtime Portability Evaluation (2026-08-16T16:00:00-07:00):
- Evaluated inbound implementation request from SQL Live-Site Operations for runtime binding resolver, managed identity, workload identity, host bridge, and preflight validation.
- Verdict: Accept-with-modifications. Core abstraction (profile → resolver → transport + auth) aligns with existing `ToolTransport` interface seam.
- Key finding: "no tool contract schema changes" is contradicted by the ask's own requirements (capability declaration for preflight, action classification for approval policy). Additive schema changes are unavoidable.
- Learning: when an ask declares "no schema changes" as a non-goal but requires new declarative metadata (contexts, classifications), the ask has an internal contradiction. Call it out immediately — the contradiction won't resolve itself and will block implementation if deferred.
- Learning: preflight validation must separate STATIC bindability (does a binding EXIST for this context?) from DYNAMIC reachability (can we reach the endpoint / acquire a token RIGHT NOW?). These are different failure classes deserving different error codes and different operator messages. Conflating them produces confusing errors.
- Learning: when a phase estimate depends on an unanswered architectural question (library vs. subprocess), the estimate is fiction. Require the question to be answered in a prior phase before accepting the estimate.
- Decision written to `.squad/decisions/inbox/barbara-runtime-portability-ask-evaluation.md`.
- Formal reply authored incorporating Don's ground truth verification (7/8 claims true, run-gert.ps1 FALSE, ghost fields AllowedEnvironments/RequiresCapabilities discovered, --package-map existing mechanism identified) and David's integration/protocol critique (tiered preflight, corrected error taxonomy, late-result safety defect, host bridge protocol gaps, Phase 1/3 ordering risk).
- Reply written to `.squad/decisions/inbox/barbara-runtime-portability-reply-draft.md`.
- Gate closure ruling written to `.squad/decisions/inbox/barbara-runtime-portability-gate-closure.md`. Gate CLOSED — all 6 blocking conditions satisfied. Three counter-positions accepted with rulings. One open item remains: context vocabulary for AllowedEnvironments (blocks early win only, not Phase 0).
- Learning: when reviewing an external team's implementation request, the most valuable findings are often what the request MISSED about the existing codebase (dead schema fields, existing CLI flags, existing enforcement patterns) — not disagreements about what they proposed. Ground truth verification against the actual source is the highest-ROI review activity.
- Learning: when a counterparty proposes a SAFER default than yours (unspecified vs. read-only for absent classification), accept it immediately and work through the consequences rather than defending your original position. The consequences (four-value enum, explicit handling in every code path, legacy precedence rules) are engineering work, not design disagreements.
- Learning: "fail-closed by denial" (Phase 1: deny all unattended mutating) vs. "fail-closed by requiring evidence" (Phase 3: allow with audit trail) is a clean way to stage approval policy across phases without creating an unsafe window.
- Gate closure REVISED (2026-08-16T17:10:00-07:00): incorporated Don's source findings. Key corrections: (1) `allowed-environments: ["real"]` is a RunMode discriminator, not a deployment context — two orthogonal axes on one field; added AllowedModes field + fixture migration. (2) ApprovalGate enforcement exists ONLY on substitution path; direct-invocation path (stdio/mcp/mcp-http) bypasses gate entirely — disclosed to consumer, added as Phase 1 fix. (3) classification is per-action pointer-typed, not per-tool. (4) `gert plan` framed as "profile compatibility report" with explicit limits.
- Learning: when you rule "free-form, no collision" on a field that already has populated values, verify what those values MEAN before ruling. `"real"` looked like a test namespace until you trace it to `engine.RunModeReal` — then it's a completely different semantic axis. Free-form fields are cheap to declare and expensive to disentangle once two meanings share one field.

📌 Runtime Portability — Final Design Agreement (2026-08-17T06:15:00-07:00):
- GATE CLOSED after 3 rounds of exchange with SQL Live-Site Operations team.
- Final acknowledgment: `.squad/decisions/inbox/barbara-runtime-portability-final-acknowledgment.md`
- Key principle adopted from counterparty: "Classification is an opt-in to reduced friction. Absence must never grant additional execution rights."
- Three late corrections from counterparty (all accepted): (1) interactive unspecified must gate, not warn-and-proceed; (2) attendance (human presence) is orthogonal to context (execution host) — must be a declared profile property, not inferred from TTY; (3) test context must be restricted to native-only bindings at Tier 0.
- New Phase 1 items: declared attendance replacing TTY inference, test-context-non-native-binding rule, attendance-mismatch Tier 0 check.
- OQ2 closed: subprocess retained (extension already uses subprocess), versioned IPC, capability advertisement.
- Learning: "human present" ≠ "human attentive." During live-site incidents, passive warnings are missed. Any behavior that relies on a human noticing something must use an active gate (confirmation prompt), not a passive signal (warning message). This is especially true for `unspecified` classification — the whole point of unspecified is that nobody has verified what the action does.
- Learning: when a counterparty corrects you multiple times and is right each time, say so plainly. Trust compounds and the design improves faster when you acknowledge it without ceremony.

📌 Final Acknowledgment Rev 2 (2026-08-17T06:45:00-07:00):
- Revised §3 test-context rule: subprocess allowed with explicit opt-in (hermetic fake MCP servers are legitimate), mcp-http NEVER allowed. Original blanket ban was too strict.
- §1 disclosure: "interactive unspecified → gate fires" is a breaking regression for existing runbooks (0 prompts → N prompts). Mitigation: explicit `requires-approval: false` (detectable via non-nil *ToolGovernance pointer) grandfathered as read-only equivalent; new `legacy_unspecified_policy: allow|prompt` profile field as auditable escape hatch; PKG-W plan-time warning for migration pressure.
- §2 disclosure: TTYOutput is hardcoded true in run.go — no isatty() detection exists. CI pipelines get TerminalApprovalGate and block on stdin. Live latent bug that declared-attendance fixes. Also: TTYOutput conflates approval attendance with physical IO availability — profile `attendance` replaces only the approval dimension.
- Learning: before publishing "this rule applies to all X," cost it against the existing corpus. 75 tool actions with no classification × "gate fires on unspecified" = 75 new prompts across the ecosystem. The principle can be right while the rollout is wrong. Always compute the blast radius of a universal rule against the actual population.

📌 Final Acknowledgment Rev 3 (2026-08-17T06:37:00-07:00):
- Fifth correction from SQL Live-Site accepted: *ToolGovernance pointer does NOT distinguish explicit `requires-approval: false` from absent — only proves some governance property was authored. Fix: RequiresApproval becomes *bool (tri-state). Blast radius: 1 declaration + 1 read site, 0.5 days.
- Declined runbook-level tri-state extension (different semantic, 15+ read sites, zero design benefit).
- Corrected test-context subprocess wording: Gert does zero process sandboxing; the opt-in is an author assertion, not a technical sandbox. mcp-http prohibition IS enforced (Tier 0 block).
- Exchange CLOSED after 5 rounds. Design agreed. Implementation authorized.
- Learning: "non-nil pointer with field == zero value" does NOT prove the field was authored — it proves the STRUCT was authored. For tri-state detection on individual fields, you need a pointer on THE FIELD, not on the containing struct. This is a Go-specific trap: omitempty on bool suppresses false, making explicit-false unrepresentable on output; *bool+omitempty gives nil/&false/&true which is the correct tri-state.

📌 Final Acknowledgment Rev 4 — EXCHANGE CLOSED (2026-08-17T06:56:00-07:00):
- Sixth correction accepted: `requires-approval: false` must NOT coerce `classification: read-only`. Approval routing and side-effect classification are orthogonal. Coercing &false→read-only would grant retry eligibility to potentially destructive actions — a safety regression inside a compat shim. Withdrew "equivalent to read-only" language.
- Condition accepted: runbook-level requires-approval: false cannot override per-action semantics. Verified: substitution path uses monotone OR (structurally impossible to suppress); plain-step path has NO gate today (GovernanceEvaluator is nil in all prod wiring — corrected our own earlier inaccurate disclosure). Phase 1 constraint: BuildPolicy must compose both runbook-level and per-tool governance.
- No landmine confirmed: zero coupling between RequiresApproval and retry logic anywhere in codebase.
- Design FINAL. Six rounds, six corrections accepted. Implementation authorized.
- Learning: a "compatibility shim that relaxes safety" is a contradiction in terms. Compat shims preserve EXISTING behavior; they must never grant NEW rights. When you write "legacy X → treat as safe-equivalent Y," check whether Y grants capabilities X never had (retry eligibility, non-halting timeout handling). If it does, the shim is not a compat mechanism — it is a silent privilege escalation wearing a compat label.

📌 Slice 1 + Slice 2 Review Gate (2026-08-17T07:25:00-07:00):
- APPROVED. Commits a2e7db0 (Don, schema) + c810b96 (Ken, enforcement).
- All 8 requirements verified against source: GovernanceEvaluator wired, policy composes both levels (monotone OR), chokepoint pre-dispatch covers all transports, orthogonality structurally enforced, tri-state integrity proven by test case (b), binding constraint satisfied.
- CI hang hazard assessed: not a regression (no existing artifact triggers the gate); documented as known limitation pending declared-attendance work.
- Expected gap: unspecified-fires-gate behavior deferred to ProfileApprovalGate slice.
- Review written to `.squad/decisions/inbox/barbara-slice1-slice2-review.md`.


---


---


