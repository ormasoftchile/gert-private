# David — Project History

## Learnings

**2026-06-03 — Webhook Delivery Patterns Review**
- Analyzed 4 dimensions of event delivery (webhook models, subscription types, payload designs, security)
- Key insight: retry + exponential backoff is the sweet spot for v1 — simple, survives transient failures, unblocks shipping
- Defensive design: always plan for "customer endpoint is down" before writing code; DLQ is non-negotiable in v1
- Per-execution webhooks scale well initially; tenant-level webhooks reduce blast radius for v1.1; Service Bus moves responsibility to Azure in v2
- HMAC-SHA256 + HTTPS is table-stakes; mTLS and Managed Identity defer to v2
- Summary events (not full stream) in v1 keeps payload small and integrations simple; full audit trace available on-demand via polling endpoint
- Baseline: 5 retries over 24h with exponential backoff; customer admin can replay from DLQ indefinitely

**2026-06-04:** Brainstorm output merged to decisions.md. Webhook & event delivery baseline documented with v1–v2 roadmap. Rollout plan recorded. Security checklist included. Orchestration log created.

**2026-06-04T12:28:39.293-04:00 — Campaign Identity + Completion Integration Design**
- Identity validation at magic-link click must be fail-closed, fast, and tenant-scoped: 8s absolute timeout, per-endpoint circuit breaker, no run start unless the customer endpoint returns a valid approved contract.
- One-time magic links should be opaque tokens with Cosmos DB conditional state transitions (`issued -> redeeming -> redeemed|denied`) so concurrent clicks cannot double-start runs while transient failures can still recover safely.
- Customer notifications should stay summary-first and at-least-once: Service Bus-backed delivery, stable idempotency key per semantic event, scheduled retries, DLQ + replay for prolonged endpoint outages.
- Main integration risks in this scenario are contract drift at the customer identity API, webhook auth/config breakage, and ambiguous retry windows; these surfaced ratification items for timeout, retry schedule, webhook scope, and Cosmos storage shape.

---

## Project Context

GERT is a governed, executable, traceable runbook engine (Go binary, local-first). It has a VSCode extension, TUI runner, and mobile SDKs. The team focus is building a web execution platform on Azure — white-labeled customer portals where end-users run runbooks as part of business processes (applications, authorizations, compliance flows). Azure infra only (no k8s). Key concerns: queue-driven execution, webhook delivery, tenant isolation, audit traceability.

**User:** ormasoftchile
**Session start:** 2026-06-03

---

## Session: 2026-06-04T02:50:37Z — Interactive Waiting Patterns & User Input Gate

**Scribe consolidated 8 inbox items.**

**Key outcomes for David:**
- **New webhook events:** `user_input_requested`, `user_input_received`, `user_input_timed_out`
- **Approval events:** `approval.requested`, `approval.approved`, `approval.rejected`, `approval.timed_out`
- **Event payload:** Summary events only (no full trace changes)
- **Scope:** Same retry/backoff model (5 attempts over 24h)
- **Status:** Event schema updated in decisions.md for integration planning


## Learnings from Declaration & Consent Integration Analysis (2026-06-03)

### Receiver Landscape: Five Industry Archetypes

Declarations are not generic events; they are legal commitments with distinct downstream receivers:

1. **Healthcare:** EHR, surgical scheduling, insurance pre-auth, medical archive. Require signed PDFs, witness verification, 7-year retention, immutable records. Cannot revoke post-procedure start.

2. **Insurance:** Policy admin, underwriting, billing, agent portal. Require evidence of intent, strict ordering (no charge before consent), settlement finality. Billing system is particularly sensitive (recurring charge linkage).

3. **Government:** Registry of acts, archival system, citizen portal. Strictest: registry capacity limits, 30-year archival, citizen notification within 5 min, reproducible audit trails.

4. **Finance/KYC:** AML system, account opening, compliance archive. Compliance-first: silent drops = regulatory fines. Every KYC declaration version traceable. 10-year hold.

5. **Employment:** HRIS, payroll, benefits enrollment. Tight batch windows (payroll deadline is immovable). Tax withholding election cannot be silent-dropped.

**Implication:** Cannot use one-size-fits-all event schema. Must support evidence package depth (trace excerpt, hash chain, audit receipt) that allows receivers to independently verify immutability.

### Evidence Package Shape

Declarations require layered evidence beyond what transient events carry:

- **Signed Document (PDF):** Time-limited SAS URL (7 days default) before seal, permanent after seal. Hash included for immutability proof.
- **Trace Excerpt (NDJSON):** Not full execution trace; only form input + signature steps. Reduces payload; receiver can fetch full trace via declaration_id if audited.
- **Hash Chain (Merkle Proof):** Daily batch Merkle tree proving declaration is in immutable archive (no post-hoc edits by system operator).
- **Audit Receipt (PDF + QR):** Human-readable receipt + permanent QR link for declarant to print/file as proof.

**Implication:** Evidence package is not inline in webhook. Webhook carries URLs; receiver pulls evidence on demand. Scales to large audiences without massive payloads.

### Delivery Guarantee: At-Least-Once + Immutability

V1.0 retry model is necessary but not sufficient. Declarations demand idempotency at the receiver level:

- Receiver deduplicates on `idempotency_key` (not just event_id). Safe to retry declaration.completed for same declaration_id multiple times.
- Replay endpoint (`GET /api/v1/declarations/{declaration_id}/replay?since=T`) for reconstruction if receiver crashes mid-processing.
- Sealed archive notification (`declaration.sealed` event) signals: "Declaration is now in permanent immutable archive; all SAS URLs permanent; hash chain finalized."
- Long-term retrieval SLA: Healthcare 7yr, Finance 10yr, Government 10–30yr. No expiration after seal.

**Implication:** Webhook infrastructure must support replay + permanent SAS (or fallback retrieval mechanism) for receivers to comply with legal holds.

### Edge Cases: Nine Declaration Event Types

Declarations are not binary (completed or failed). Each path requires an event:

1. **submitted:** Form filled, signature pending. Receiver audits; optional queue.
2. **verified:** KYC/ID verification passed. Continue; optional status update.
3. **signed:** Declarant signature captured (witness may still be pending).
4. **completed:** FINAL; all signatures, all verification done. Trigger downstream: schedule surgery, bind policy, create compliance record.
5. **declined:** Declarant actively rejected (clicked "Decline"). Cancel downstream; notify participant.
6. **abandoned:** User didn't finish (inactivity, closed tab). Audit-log; may retry.
7. **expired_unwitnessed:** Witness window passed. Retry or escalate.
8. **sealed:** N days post-completion; immutable archive. Archive-ready; receiver downloads evidence package to long-term storage.
9. **revoked:** Declarant/authorized party revoked consent post-completion. **CRITICAL:** Cancel downstream, reverse charges, halt procedure.

**Implication:** V1.0 event family (execution lifecycle + user input) is too coarse. Declaration workflow needs a second event family with finer granularity.

### Revocation Semantics: The Hardest Edge Case

Post-hoc revocation is non-trivial. Declarant signed surgical consent, was placed on pre-op protocol, then calls to withdraw consent hours before surgery.

- Receiver gets `declaration.revoked` event.
- Receiver must be idempotent: revoking twice = safe no-op.
- Receiver must cascade: cancel surgery, notify anesthesia team, halt pre-op protocol.
- Audit trail critical: compliance must know who revoked, when, what cascade impact.

**Implication:** Revocation is not handled at webhook registration time; it's a post-completion async action requiring playbook execution at receiver.

### Retention SLA Complexity

Industries have different retention requirements:
- Healthcare: 7 years minimum (varies by jurisdiction).
- Insurance: 5–10 years (product-dependent).
- Government: 10–30 years (registry-dependent).
- Finance/KYC: 10 years (regulatory).
- Employment: 7 years (tax, labor law).

**Implication:** Cannot have one DLQ retention policy. Declarations must be tagged with industry + retention tier. Compliance dashboard should show: "Healthcare declarations from 2020 still held; can purge finance declarations from 2019."

### Open Questions for Next Sprint

1. Idempotency key scope: `declaration_id` only, or `declaration_id + version`? (If revocation bumps version, idempotency_key changes?)
2. Cascade notifications: When revocation happens, does GERT notify declarant/witness/surgeon directly, or only via webhook?
3. Audit receipt: PDF with QR only, or support text QR body in email?
4. SAS URL expiration before seal: 7 days? Configurable?
5. Merkle tree scope: Daily batch per-tenant, or global daily batch?
6. Revocation window: Can revoke indefinitely, or only within N days?

### Next Session: Implementation Roadmap

- **Phase 1 (MVP):** 4 core events (submitted, signed, completed, declined); evidence package URLs; idempotency implementation.
- **Phase 2 (Robustness):** 5 edge case events; DLQ compliance-critical alerting; replay endpoint.
---

## Team Integration — White-Label Campaign Platform (2026-06-04)

**Cross-agent coordination:**
- **Barbara:** Integration contracts ratify webhook delivery model (signed webhook + pull reconciliation) and identity validation requirements (server-to-server only); campaign layer defines notification subscription entities
- **John:** Aspire local dev supports testing integration contracts in hybrid dev loop
- **Leslie:** Portal UX determines when webhooks are triggered (completion, abandonment, identity denial); affects event schema design
- **Scribe:** 5 integration contract ratification items merged to decisions.md for team review

**Key decisions:** Identity endpoint timeout 8s absolute; circuit breaker opens after 5 failures (5 min); campaign links stored as separate Cosmos docs; webhook registration at tenant/campaign scope; 5 retries over 24h; default events (campaign.closed, user.started, user.completed, user.failed).

---

## Team Directive — 2026-06-04T17:15:45-07:00

**Leslie uses he/him pronouns.** All team members and the coordinator must refer to Leslie with he/him going forward.

