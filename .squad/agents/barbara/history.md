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
