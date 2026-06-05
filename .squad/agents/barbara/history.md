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

## 2026-06-04T20:14:36-07:00 — Stream E Removed (Scribe notification)

**Status Update:** User directive executed. Stream E (migrator tooling) removed entirely per scope decision. Rationale: GERT has no production runbooks, so migration solves a non-problem.

**Impact on Phase 1 Plan:**
- Stream E is completely removed from the plan
- Phase 1 closes with Streams A/B/C/D/F only
- C# runtime can begin once spec is frozen (no migration dependency)

**Cleanup Note:** Grammar files (gis.ebnf, gcp.ebnf) retained migration-tool comments as "do-not-touch" per directive. Team should decide whether to remove these in a grammar-owned follow-up.

**Decisions:** Merged to `.squad/decisions.md` (timestamp 2026-06-04T20:14:36-07:00)
