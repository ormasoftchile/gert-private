# Squad Decisions

## Active Decisions

### Architecture Options: GERT Web Execution Platform on Azure
**Author:** Barbara (Lead / Architect)  
**Date:** 2026-06-03  
**Status:** Proposal — awaiting team decision

Five execution topology options evaluated:
1. **Thin Relay** (Functions + Container Apps Job): Per-run ephemeral containers, strong isolation
2. **Embedded Worker** (Functions with custom Go handler): Cost-efficient, short runbooks, shared host process
3. **Persistent Agent Pool** (Container Apps with KEDA): Always-warm pool, high concurrency, multi-run blast radius risk
4. **Sidecar Agent** (Container Apps with sidecar): Per-tenant dedicated compute, maximum governance enforcement
5. **Hybrid Dispatch** (Functions + Container Apps, tiered): Route short/simple to Functions, long/complex to Jobs

**Recommendation:** Start with **Option 1 (Thin Relay)** for MVP. Plan migration to **Option 4 (Sidecar Agent)** for enterprise.

**Key reasoning:**
- Governance preservation is non-negotiable; Thin Relay makes bypass structurally impossible
- Blast radius must be per-run (one run failure ≠ multi-tenant impact)
- 3–8s cold start acceptable for v1
- Sidecar is endgame for enterprise customers willing to pay for dedicated compute

**Explicitly rejected for v1:**
- Option 3 (Persistent Pool): Multi-run blast radius incompatible with governance-first value proposition
- Option 5 (Hybrid Dispatch): Premature optimization; build one lane well first

**Open questions:**
1. Approval gate handling in Thin Relay: Job state serialization or keep-alive?
2. Trace storage: Blob (append-friendly) vs Cosmos DB (queryable)?
3. Event schema: Raw GERT events or web-specific transformation?
4. Runbook storage: Blob per-tenant or Git repo with webhook sync?

---

### Azure Service Stack: Hard Constraints & Recommended Services
**Author:** John (Azure Platform Engineer)  
**Date:** 2026-06-03  
**Status:** Inbox — pending team review

**Compute recommendation:** Container Apps (no time limit, scale-to-zero, KEDA queue trigger)  
- ⚠️ Functions Consumption ruled out: 10-minute hard limit incompatible with runbooks
- Functions Premium viable for short runs but higher cost

**Queue recommendation:** Service Bus Standard (DLQ critical, sessions for FIFO, 14-day retention)  
- 256KB message size limit (use Blob pointer pattern for large payloads)
- 1,000 msg/sec throughput adequate for MVP

**Auth recommendations:**
- Customer auth: Entra External Identities (50K MAU free tier)
- Service-to-service: Managed Identity / Workload Identity (zero secrets, zero per-call cost)

**Frontend recommendation:** Static Web Apps Standard ($9/mo, unlimited custom domains for white-label)

**Webhook recommendation:** Event Grid (primary, 24h retry) + Service Bus fallback (for SLA-sensitive delivery)

**Open questions:**
1. Runbook payload size: Ever exceed 256KB?
2. Execution time P99: Typical runbook duration?
3. MAU projection: End-users per customer?
4. Webhook SLA: Acceptable failure window?
5. Frontend architecture: Static SPA or SSR needed?

---

### Governance Red Lines: Execution Adapter Patterns
**Author:** Don (Backend Dev)  
**Date:** 2026-06-03  
**Status:** Proposed — requires team review before adoption

**Red Line 1: No pattern may bypass** `Runtime.Start → RunHandle.Next`  
→ Eliminates direct tool invocation, per-step Durable Functions, step result spoofing

**Red Line 2: Trace MUST be written only by Runtime Core**  
→ Eliminates external process trace writes, synthetic event injection, middleware appends

**Red Line 3: Exactly one live Runtime Core per run_id at a time**  
→ Requires at-most-once queue semantics (Service Bus sessions) + distributed lock on run record

**Red Line 4:** `client: "web"` MUST be set in RunOptions for all web-triggered runs  
→ Audit trail integrity: admins must isolate web vs CLI vs TUI runs

**Red Line 5: Approval gates require pause/resume protocol**  
→ Worker must checkpoint via RunStore if it can't stay alive through approval window

**Patterns evaluation:**
| Pattern | Safe? | Notes |
|---|---|---|
| Queue-triggered worker | ✅ YES | Recommended baseline |
| HTTP Container App | ✅ YES | Viable; timeout constraints |
| Durable Functions (whole run) | ✅ YES | Viable but complex |
| Durable Functions (per-step) | 🚨 RED | Violates Red Lines 1, 2, 3 |
| Sidecar/agent | ✅ YES | Viable; IPC design needed |
| Direct tool invocation | 🚨 RED | Violates Red Lines 1, 2 |

**Decision required:** Team ratification of red lines before adapter implementation begins.

---

### Webhook & Event Delivery Baseline
**Author:** David (Integration Engineer)  
**Date:** 2026-06-03  
**Status:** Proposal

**Baseline v1.0:**
- **Delivery model:** Retry + exponential backoff (5 attempts over 24h: 1m, 5m, 30m, 2h, 6h)
- **Subscription model:** Per-execution webhook (URL provided at run submission)
- **Event payload:** Summary events only (`execution.started`, `step.completed`, `execution.finished`, `execution.failed`)
- **Security:** HMAC-SHA256 signature via X-GERT-Signature header

**Future upgrades:**
- v1.1: Tenant-level webhook registration
- v2.0: Azure Service Bus topic subscription (at-least-once, customer-managed consumer)
- v2.0: Azure Event Grid integration

**Transient customer endpoint down (<24h):**  
Events queue in memory, retry loop, final events → DLQ (Azure Storage), customer can replay

**Extended outage (>24h):**  
Events persist in DLQ indefinitely, customer replays when endpoint recovers

**Rollout plan:**
1. Sprint 1: Per-execution webhook + summary events + HMAC
2. Sprint 2: Retry/DLQ logic + admin replay endpoint
3. Sprint 3: Tenant-level webhook registration
4. Sprint 4: Service Bus topic subscription (v2.0 alpha)
5. Sprint 5: Event Grid integration (v2.0 beta)

**Open questions:**
1. Acceptable event latency? (Recommend: <5s)
2. Per-event filtering at registration? (Recommend: no in v1)
3. Auto-purge DLQ events? (Recommend: keep indefinitely, manual purge)
4. Support HMAC algorithms other than SHA256? (Recommend: no in v1)

---

## Governance

- All meaningful changes require team consensus
- Document architectural decisions here
- Keep history focused on work, decisions focused on direction
