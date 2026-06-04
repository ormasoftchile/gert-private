# Barbara — Project History

## Learnings

- **2026-06-03:** The GERT runtime's governance enforcement happens INSIDE the Go binary, before subprocess fork. This means any hosting option that runs the unmodified binary preserves governance automatically — the web platform cannot bypass it unless it circumvents the JSON-RPC contract entirely.
- **2026-06-03:** GERT has four adapter surfaces: JSON-RPC exec contract (`exec/v1`), WebSocket event stream, tool invocation contract, and extension handshake. The web platform is just another adapter — it must consume these contracts, not invent new execution paths.
- **2026-06-03:** The event system is one-way (runtime → subscribers) and the trace file is the authoritative record (at-least-once to file, best-effort to subscribers). This means the web layer can tolerate event streaming failures as long as the trace file is persisted reliably.
- **2026-06-03:** Key architectural decision: WHERE the Go binary runs determines isolation, blast radius, and governance preservation. Produced 5-option architecture menu. Recommended Thin Relay (Container Apps Jobs) for MVP with Sidecar Agent as enterprise endgame.
- **2026-06-03:** Approval gates mid-execution are the hardest seam for the web platform. Jobs-based topologies need state serialization + resume; pool/sidecar topologies handle them naturally but at higher cost/complexity.
- **2026-06-04:** Brainstorm output merged to decisions.md. Architectural recommendation (Thin Relay MVP → Sidecar endgame) recorded. Four related agent outputs (John, Don, David) synchronized cross-agent. Orchestration log created.
- **2026-06-03:** Go-only constraint removed. User open to C# native runtime. Re-evaluated all 5 original topologies + 4 new C# options. Key insight: C# eliminates the binary lifecycle problem AND makes approval gates trivial (async/await vs checkpoint/resume). New MVP recommendation: B1 (App Service + BackgroundService) — zero cold start, trivial approval gates, single deployment, $55/mo. New enterprise endgame: B4 (App Service Per-Tenant) — same isolation as A4 Sidecar but without container orchestration complexity. Retired A2 (superseded by B2 Functions Isolated). Critical constraint: C# Runtime must be `internal sealed` to prevent governance bypass in shared-process topology. Don must validate governance parity. Decision output merged to decisions.md (2026-06-03T20:36:12). Orchestration log created.
- **2026-06-03:** Produced comprehensive A6 architecture document (`design/web-platform/a6-architecture.md`). Key design decisions: (1) App Service P1v3 for always-on execution — no cold start, (2) IApprovalGate interface stubbed from day one as A6→A8 migration seam — polling in A6, Durable Functions external events in A8, (3) JSONL trace in Blob Storage is authoritative record (not Cosmos DB) — if Blob unavailable, steps BLOCK, (4) Service Bus sessions for at-most-once run dispatch, (5) `internal sealed` runtime classes to prevent governance bypass. Explicit scope: A6 handles single-approver gates <24h; multi-approver, escalation, and long-running approvals deferred to A8. Estimated MVP cost: ~$113/month. Created decision document at `.squad/decisions/inbox/barbara-a6-architecture.md`.
- **2026-06-03:** Defined Interactive Waiting Patterns — distinguished Approval Steps (long wait, external approver, polling-based) from Choice Steps (short wait, portal session user, TaskCompletionSource + SignalR). Introduced `IChoiceGate` interface as the dual of `IApprovalGate`. Key design: choice gates use TCS + SignalR push for sub-second resume (NOT polling like approvals). Both interfaces are A6→A8 migration seams — swap to `WaitForExternalEvent<T>()` behind DI. Added `choices` Cosmos DB container, new API endpoints (`/choices/{id}/select`, `/choices/pending`), and SignalR `ChoiceRequired` event. Insight: choice steps are cheap in A6 (user responds quickly); approval steps are expensive (thread held hours). Migration pressure to A8 comes from approval-heavy workloads, not choice-heavy ones.
- **2026-06-03:** CORRECTION — Generalized `IChoiceGate` → `IUserInputGate`. Germán's directive: every user input is blocking, not just choices. The gate now handles Choice, Text, Confirmation, FileUpload, and Form via `UserInputKind` enum. Interface takes `UserInputRequest` (describes what's needed) and returns `UserInputResponse` (carries the answer). Cosmos DB container renamed `choices` → `user_inputs`. Endpoint unified to `POST /runs/{id}/steps/{stepId}/input` accepting `UserInputResponse` payload. Migration seam unchanged in shape: `IUserInputGate` (A6: TCS + SignalR) → `IUserInputGate` (A8: `WaitForExternalEvent<UserInputResponse>`). Decision note: `.squad/decisions/inbox/barbara-user-input-gate-correction.md`.
- **2026-06-03:** Declaration-before-process scenario taxonomy produced (`.squad/decisions/inbox/barbara-declaration-scenarios.md`). 22 concrete scenarios across healthcare, insurance, finance, real estate, employment, government, education. **Five archetype patterns** extracted: (A) Single-Party Informed Consent, (B) Two-Party Witnessed Declaration, (C) Proxy/Representative Declaration, (D) Conditional Eligibility-Then-Declaration, (E) Multi-Step Sectioned Disclosure. **Integration seam rule:** GERT owns form presentation, acknowledgment capture, signature token recording, witness approval gate, and JSONL trace emission. GERT does NOT own eligibility determination, KYC/AML screening, biometric validation, e-signature legal authority, payment, scheduling, or translation. **Six data model gaps flagged for Don:** `principal_id` (proxy flows), `display_language`+`disclosure_version_hash` (translation audit), `SignatureToken` type with `assurance_level`, `declaration_id` for cross-run revocation linking, `form_interaction_log` events (21 CFR Part 11), parent/child run relationship (A8). **Uncomfortable questions raised:** minors/capacity, language validity, e-sig jurisdiction (ESIGN/eIDAS/Chilean Law 19.799), revocation-after-execution compensation pattern, dual-blind multi-party flows.
- **2026-06-04T12:28:39.293-04:00:** Contract-acceptance campaigns need a distinct campaign layer above A6 runtime: tenant config, campaign definitions, immutable contract artifact versions, audience members, invitations, runs, and notification subscriptions. The white-label portal should remain a shared Static Web Apps deployment fronted by Front Door, with per-tenant branding and domains loaded by hostname; per-tenant deployments are an exception for isolation/networking/regulatory requirements.
- **2026-06-04T12:28:39.293-04:00:** A magic link is invitation-possession proof, not identity proof. Recommended baseline is two-stage invite redemption (to survive email scanner prefetch), followed by server-to-server customer identity validation before the contract runbook opens; webhook delivery must be paired with a pull reconciliation API because customer endpoints will fail at the worst possible time.

---

## Project Context

GERT is a governed, executable, traceable runbook engine (Go binary, local-first). It has a VSCode extension, TUI runner, and mobile SDKs. The team focus is building a web execution platform on Azure — white-labeled customer portals where end-users run runbooks as part of business processes (applications, authorizations, compliance flows). Azure infra only (no k8s). Key concerns: queue-driven execution, webhook delivery, tenant isolation, audit traceability.

**User:** ormasoftchile
**Session start:** 2026-06-03

---

## Session: 2026-06-04T02:50:37Z — Interactive Waiting Patterns & User Input Gate

**Scribe consolidated 8 inbox items.**

**Key outcomes for Barbara:**
- **IUserInputGate adoption:** Generalized `IChoiceGate` to cover Choice, Text, Confirmation, FileUpload, Form
- **A6 finalized:** P1v3 App Service confirmed; cost baseline $113/mo
- **Interface design:** TCS + SignalR for sub-second UX; 15-min timeout for in-session interaction
- **Cosmos container:** `user_inputs` (partition key `/runId`)
- **API endpoints:** `POST /runs/{id}/steps/{stepId}/input` unified
- **Status:** Decisions ratified and merged to decisions.md

## Team Integration — White-Label Campaign Platform (2026-06-04)

**Cross-agent coordination:**
- **John:** Aspire local dev supports A6 architecture in multi-mode dev environment (fast loop + cloud integration loop)
- **David:** Integration contracts align campaign layer with webhook delivery guarantees; server-to-server identity validation requirement cascades from Barbara's decision
- **Leslie:** Portal UX decisions (shared SWA, per-tenant custom domains, magic-link auth) are the frontend implementation of campaign layer constraints
- **All:** 8 campaign architecture ratification items + 5 integration contract items + 7 portal UX items merged to decisions.md for team review

**Architectural implication:** Campaign layer (tenant, campaign, audience, invitation, contract artifact, run, notification) becomes first-class platform surface; white-label portal is the referent frontend; A6 runtime remains isolated execution engine.

