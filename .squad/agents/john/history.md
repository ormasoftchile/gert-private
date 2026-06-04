# John — Project History

## Learnings

### 2026-06-03 — Azure Service Options Survey for GERT Platform

**Compute:**
- Azure Functions Consumption plan: hard 10-min execution cap (230s on HTTP triggers). No good for long runbooks.
- Functions Premium: 60-min cap (unlimited with Durable), no cold start, ~$173/mo minimum for EP1.
- Container Apps: best fit — scale-to-zero, no execution time limit, built-in KEDA, DAPR optional. ~$0.000024/vCPU-sec.
- Container Instances: fine for one-off burst, but no queue trigger integration; requires orchestration wrapper.
- App Service: always-on cost ($55+/mo Basic) unjustified for runbook workloads; no scale-to-zero.

**Queues:**
- Service Bus Standard: 256KB max message, 14-day retention, 1,000 msg/sec/queue, built-in DLQ, sessions for ordering. Best fit.
- Service Bus Premium: 100MB message size, dedicated capacity, ~$668/mo per messaging unit.
- Storage Queues: 64KB native (64MB with pointer trick), 7-day retention, no DLQ, no ordering. Too primitive.
- Event Grid: push-only, no polling, 1MB max event, 24h retry window. Good for webhook fanout, not queue backbone.
- Event Hubs: streaming, no per-message DLQ, partitioned ordering only. Overkill for runbook queue.

**Auth:**
- Entra External Identities: 50,000 MAU free tier, good for white-label customer portals. OIDC/SAML support.
- Workload Identity: right tool for service-to-service (Function → Service Bus, Function → Storage). No per-call cost.
- Custom auth with Entra IdP: most flexible but highest dev burden; worth it only if External ID is too constraining.

**Frontend:**
- Static Web Apps Free: 2 custom domains, 100GB bandwidth, 0.5GB storage. Good enough for white-label MVP.
- Static Web Apps Standard: $9/mo, unlimited custom domains, 100GB, Azure Functions integration.
- App Service for SSR: $55+/mo Basic; only justified if SSR framework (Next.js) is a hard requirement.

**Webhook delivery:**
- Event Grid: 10M events/mo free, push with retry (24h window, exponential backoff), 1MB max. Best for low-volume webhook fanout.
- Service Bus customer subscription: reliable, ordered, DLQ for failed delivery. Better when customers poll or have unreliable endpoints.
- Custom Function dispatcher: most control, most maintenance. Justified only if Event Grid retry semantics are insufficient.

**2026-06-04:** Brainstorm output merged to decisions.md. Hard service constraints documented and recommended stack ratified. Orchestration log created. Open questions recorded for team discussion.

---

## Project Context

GERT is a governed, executable, traceable runbook engine (Go binary, local-first). It has a VSCode extension, TUI runner, and mobile SDKs. The team focus is building a web execution platform on Azure — white-labeled customer portals where end-users run runbooks as part of business processes (applications, authorizations, compliance flows). Azure infra only (no k8s). Key concerns: queue-driven execution, webhook delivery, tenant isolation, audit traceability.

**User:** ormasoftchile
**Session start:** 2026-06-03
