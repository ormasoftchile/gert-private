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

### 2026-06-04T12:28:39.293-04:00 — Aspire Local Dev Reality Check for A6

**Key findings:**
- Aspire is a good fit for A6 local orchestration of **Service Bus**, **Blob Storage/Azurite**, and optionally **Cosmos DB**, but it does **not** fully localize the whole Azure stack.
- **Service Bus** support is solid in Aspire (`Aspire.Hosting.Azure.ServiceBus`, `Aspire.Azure.Messaging.ServiceBus`) and is good for functional local work, but emulator behavior should not be treated as proof of production-grade session semantics under concurrency.
- **Blob Storage** support is solid (`Aspire.Hosting.Azure.Storage`, `Aspire.Azure.Storage.Blobs`) and is the right local path for JSONL/append-blob trace work.
- **Cosmos DB** support exists (`Aspire.Hosting.Azure.CosmosDB`, `Aspire.Microsoft.Azure.Cosmos`), but the emulator is not serverless, and on Mac the safer local path is often `RunAsPreviewEmulator()` or a cheap real dev Cosmos account.
- **Azure SignalR** is supported in Aspire (`Aspire.Hosting.Azure.SignalR`), but the emulator only helps in **Serverless** mode. For current A6 MVP, the correct local dev story is still **in-process ASP.NET Core SignalR**, not Azure SignalR emulation.
- There is no public official Aspire package/emulator story for **Event Grid**, **Static Web Apps**, or **Entra External ID/B2C**. These remain cloud-backed or separately tooled in local development.
- Recommended team posture: **hybrid local dev** — Aspire for app + good emulators, real Azure dev resources for auth/eventing/cloud-only behaviors, and explicit dev auth bypass when not working on identity.

## Project Context

GERT is a governed, executable, traceable runbook engine (Go binary, local-first). It has a VSCode extension, TUI runner, and mobile SDKs. The team focus is building a web execution platform on Azure — white-labeled customer portals where end-users run runbooks as part of business processes (applications, authorizations, compliance flows). Azure infra only (no k8s). Key concerns: queue-driven execution, webhook delivery, tenant isolation, audit traceability.

**User:** ormasoftchile
**Session start:** 2026-06-03

---

## Session: 2026-06-04T02:50:37Z — Interactive Waiting Patterns & User Input Gate

**Scribe consolidated 8 inbox items.**

**Key outcomes for John:**
- **Cosmos DB addition:** New `user_inputs` container (partition key `/runId`)
- **Compute confirmed:** App Service P1v3 for A6; cost baseline $113/mo
- **SignalR note:** In-process Hub for MVP; evaluate SignalR Service at scale
- **Database strategy:** Serverless Cosmos for mutable state (run, approvals, user inputs); Blob for immutable traces
- **Status:** Azure service selections finalized in decisions.md

## Team Integration — White-Label Campaign Platform (2026-06-04)

**Cross-agent coordination:**
- **Barbara:** A6 architecture (App Service P1v3) is the compute foundation for campaign execution; user input gates (IUserInputGate) determine what John needs to orchestrate locally
- **David:** Integration contracts define external dependency requirements (identity endpoints, webhook delivery) that affect dev environment setup
- **Leslie:** Portal UX (magic-link auth, shared SWA) determines what local dev must simulate vs what can be stubbed in fast inner loop
- **Scribe:** 5 Aspire local dev recommendations merged to decisions.md for team review

**Key decisions:** Hybrid local dev with Aspire orchestration (API/worker + Service Bus emulator + Blob/Azurite); real Cosmos emulator optional (allow real dev account); real Entra ID/Event Grid; in-process SignalR for MVP; two dev modes (fast inner loop + cloud integration loop); skip 100% Aspire parity ambition.

---

## Expression Language Alignment — GXL/GIS/GCP Approved (2026-06-05)

**Action for John:** C# runtime implementation must conform to new portable expression grammar. Go-style evaluation (text/template, expr-lang/expr) is DEPRECATED. All runbooks must migrate to GXL (boolean) and GIS (string interpolation) before Phase 1 close.

**What changes:**
- Boolean `when`, `condition`, `until` fields: Migrate from Go `expr` syntax to GXL (GERT Expression Language)
- String interpolation in titles, args, tool argv, display content: Migrate from Go `text/template` to GIS (GERT Interpolation Syntax, dollar-brace: `${...}`)
- No Go template semantics in C# runtime — grammar is owned by GERT, not Go
- Parser/plan-time validation enforces migration before any runbook reaches execution

**C# runtime implications:**
- Implement GXL lexer/parser/evaluator (owned by GERT spec, not dependent on Go `expr` library)
- Implement GIS interpolation engine (portable dotted paths, camelCase function names: `str.startsWith`, `list.indexOf`)
- Support `capture.default:` for optional API field handling (replaces text/template silent-empty fallback)
- Conform to portable JSON value model for subtree captures (null, bool, number, string, array, object)
- No C#-specific expression extensions; cross-runtime parity is the gate

**References:** `.squad/decisions/decisions.md` (GXL/GIS/GCP open questions resolved + architecture proposal)

---

## Team Directive — 2026-06-04T17:15:45-07:00

**Leslie uses he/him pronouns.** All team members and the coordinator must refer to Leslie with he/him going forward.

---

## GitHub Actions Cost Control — Workflow Disable Pattern (2026-06-07T19:28:42-07:00)

**Directive:** User requested disabling all GitHub Actions across GERT-family repos to control spending. Re-enabling is case-by-case future work.

**Operations executed:**
- **List active workflows:** `gh workflow list --repo ormasoftchile/{repo} --all --json id,name,state,path`
  - `--all` flag shows both active and disabled workflows; without it, only active workflows appear
  - Returns JSON with numeric `id` (preferred for disable/enable) and `path` (human-readable for commit mapping)
- **Disable workflows:** `gh workflow disable <id> --repo ormasoftchile/{repo}`
  - Accepts both numeric `id` and `path`; numeric `id` is safer (path can contain special chars)
  - No confirmation prompt; idempotent (re-running on already-disabled workflow is safe)
  - Applies immediately; new commits won't trigger disabled workflows
- **Cancel in-flight runs:** `gh run list --repo ormasoftchile/{repo} --status in_progress|queued --json databaseId,name,headBranch --limit 50`
  - Then: `gh run cancel <id> --repo ormasoftchile/{repo}`
  - Stops active compute spend within seconds

**Security carve-out applied:**
- **Kept Dependabot active** in `gert` repo. Rationale: Dependabot security scanning (vulnerability tracking) has higher ROI than cost savings. Security posture > cost for automated dependency audits.
- Did not find Dependabot, CodeQL, or other security workflows in `gert-private`, `gert-vscode`, or `gert-tui`.

**Outcome:** 8 workflows disabled across 4 repos (0 runs cancelled; no in-flight jobs at time of action). Documented in `.squad/decisions/inbox/john-actions-disabled-2026-06-07.md` with per-repo inventory and re-enable runbook.

**Key learnings:**
1. `gh workflow list --all` is mandatory to see disabled workflows for verification; without `--all`, verification appears to show zero workflows if all are disabled.
2. Workflow `id` (numeric) is the stable identifier for disable/enable operations; `path` is human-readable but may cause issues with special characters.
3. Security workflows (Dependabot, CodeQL, SAST/DAST) should be exempted from blanket cost-control directives — the cost of a vulnerability in production vastly exceeds automation spend.
4. No in-flight runs existed at disable time, but if they had, `gh run cancel` would have stopped compute within seconds (faster than workflow disable, which only prevents future runs).

