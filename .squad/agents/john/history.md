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

### 2026-06-03 — C# Runtime Re-Evaluation: Revised Azure Compute Stack

**Trigger:** Runtime language changed from Go to C# (.NET 8 Isolated Worker / ASP.NET Core). Full compute re-evaluation required.

**New compute findings:**

- **App Service B2/B3:** 3-instance scale-out ceiling — hard stop for production multi-tenant. Dev/staging only. B2 ~$61/mo, B3 ~$123/mo.
- **App Service P1v3:** 2 cores, 8 GB, ~$124/mo. 30-instance scale-out. Slots on P1v3+. VNet on Standard+/Premium. Viable for MVP when hosting API + IHostedService worker in single plan (zero extra compute cost for worker).
- **Functions Premium EP1:** 1 vCPU, 3.5 GB, ~$88–173/mo baseline. Unlimited timeout. 100-instance scale-out. VNet ✅. Durable ✅. Slots ✅. Strong production option.
- **Functions Flex Consumption:** NEW TIER. Unlimited timeout (`functionTimeout: "00:00:00"`). 1,000-instance scale-out. Pay-per-use (no baseline cost). VNet ✅. Durable ✅. Deployment slots: verify GA status. Primary recommendation for GERT C# production.
- **Functions Consumption Y1:** Still RED LINE. 10-min hard cap unchanged.
- **Durable Functions (whole-run):** VIABLE and recommended. `WaitForExternalEvent` provides zero-cost approval gate hibernation — orchestrator checkpoints to Storage, deallocates compute, waits days/indefinitely. Must NOT be implemented per-step (Red Line violation per Don's governance doc). History size: watch for runbooks with >50 steps and large payloads — blob externalization required. No hard history record count limit; practical degradation beyond thousands of events.
- **Container Apps (C# re-score):** .NET 8 alpine image ~80MB vs Go ~10MB — immaterial at runtime. Same cost ($0.000024/vCPU-sec). Cold start slightly slower (~2–3s vs ~0.1s Go). Still best for ephemeral per-run isolation (Thin Relay pattern). Not the default for persistent workers with C# in play.
- **ACI:** Still PASS. No queue trigger integration, no self-healing. Container Apps Jobs is strictly better for the same isolation goal.

**Approval gate — key finding:**
- Durable Functions `WaitForExternalEvent` is the only compute option where approval gate = zero infrastructure work + zero compute cost during wait. Alternatives (Service Bus deferral, checkpoint-serialize-terminate) work but require more code.

**Multi-tenancy model:**
- Shared plan: Flex Consumption shared task hub + named task hub per tenant in shared Storage (~20,000 tx/sec limit on Standard Storage). Cheapest model.
- Enterprise isolation: Dedicated EP1 plan per tenant or dedicated Storage Account per tenant on Flex Consumption.

**VNet:** B2/B3 no VNet. Standard+, Premium, EP1+, Flex Consumption, Container Apps (dedicated env) all support VNet + private endpoints.

**Deployment:**
- App Service / Functions Premium: slots (instant swap, zero-downtime).
- Container Apps: revision-based traffic split (canary).
- Flex Consumption: zip deploy; slots in limited preview — verify before relying on.

**Revised recommended stack:**
- Dev: App Service P1v3 (shared) + IHostedService worker
- Prod: Flex Consumption + Durable Functions (whole-run orchestrator)
- Enterprise: Functions Premium EP1 per tenant + dedicated Storage
- Per-run isolation: Container Apps Jobs (Thin Relay)

**Non-compute unchanged:** Service Bus Standard, Entra External ID, Static Web Apps Standard, Event Grid. Entra External ID + `Microsoft.Identity.Web` middleware is actually simpler in C# ASP.NET Core.

**2026-06-03 Update:** Topology analysis synchronized with Barbara's B1/B4 recommendations. Decision output merged to decisions.md (2026-06-03T20:36:12). Orchestration log created. Barbara and Don must confirm final recommendation on B1 (MVP) vs alternatives before infrastructure provisioning.

---

## Project Context

GERT is a governed, executable, traceable runbook engine (Go binary, local-first). It has a VSCode extension, TUI runner, and mobile SDKs. The team focus is building a web execution platform on Azure — white-labeled customer portals where end-users run runbooks as part of business processes (applications, authorizations, compliance flows). Azure infra only (no k8s). Key concerns: queue-driven execution, webhook delivery, tenant isolation, audit traceability.

**User:** ormasoftchile
**Session start:** 2026-06-03
