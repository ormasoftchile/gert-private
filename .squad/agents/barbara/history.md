# Barbara — Project History (Summarized)

## Overview
Barbara is the spec editor and architecture specialist for the GERT web platform and GXL expression language. Work spans governance enforcement, Azure infrastructure, declaration/consent workflows, and language design.

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
