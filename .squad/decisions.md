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

### Topology Segmentation Decision: A6 MVP + A8 Future

**Author:** Germán (via Copilot)  
**Date:** 2026-06-03  
**Status:** Adopted

**Decision:** Start with A6 (App Service + BackgroundService, C#) as MVP topology. When approval-heavy runbook customers gain traction, segment them into dedicated tenants running A8 (Durable Functions Orchestrator). A6 and A8 coexist — different customer tiers, not a full migration.

**Rationale:** A6 ships faster and is simpler to operate. A8 reserved for customers with government/compliance flows that have natural human approval pause points. Segmentation by tenant preserves isolation.

---

### A6 Architecture Decisions

**Author:** Barbara (Lead / Architect)  
**Date:** 2026-06-03T21:50:51-04:00  
**Status:** Adopted  
**Full document:** `design/web-platform/a6-architecture.md`

#### 1. App Service P1v3 as Compute Host

**Decision:** Use Azure App Service P1v3 (not B1, not Container Apps) for the execution worker.

**Rationale:**
- B1 has 60-minute warm-up time — unacceptable for queue processing
- Container Apps has cold start latency (3-8s) — acceptable but unnecessary
- P1v3 is always-on, zero cold start, simplest deployment model
- Single deployment unit: API + BackgroundService in one App Service

**Trade-offs:** Higher base cost ($81/mo vs ~$20/mo for consumption models), but eliminates cold start UX issues.

#### 2. IApprovalGate Interface as A6→A8 Migration Seam

**Decision:** Define `IApprovalGate` interface from day one with two implementations planned:
- A6 MVP: `PollingApprovalGate` (Cosmos DB polling)
- A8 Future: `DurableFunctionsApprovalGate` (external events)

**Rationale:**
- Approval gates are the hardest migration point
- Interface allows surgical swap without runtime changes
- A6 polling is adequate for single-approver, <24h scenarios
- A8 required when approval workflows span days or need escalation

#### 3. JSONL Trace as Authoritative Record (Not Cosmos DB)

**Decision:** Blob Storage append blob is the source of truth for execution audit trail, not Cosmos DB.

**Rationale:**
- Append-only semantics match trace append-only invariant
- Immutable blob policy prevents tampering
- Cosmos DB is for mutable state (run status, approvals), not audit
- Trace survives Cosmos DB unavailability

**Implication:** If Blob Storage is unavailable, steps BLOCK. This is intentional — audit trail integrity trumps availability.

#### 4. Service Bus Sessions for At-Most-Once Processing

**Decision:** Use Service Bus sessions (not standard queues) for run dispatch.

**Rationale:**
- At-most-once critical for governance integrity (Red Line 3)
- Session lock prevents duplicate run_id processing across workers
- Lock timeout (5 min) enables recovery on worker crash
- Alternative (distributed lock) requires additional infra

#### 5. `internal sealed` Runtime Classes

**Decision:** The C# GERT runtime classes MUST be `internal sealed`.

**Rationale:**
- Prevents governance bypass via subclassing
- Shared-process topology (App Service) means customer code could theoretically access runtime
- `internal` prevents cross-assembly access
- `sealed` prevents inheritance within assembly

**Red Line:** Any PR that makes runtime classes public or unsealed is rejected.

#### 6. Explicit Scope: What A6 Does NOT Support

**Decision:** The following are explicitly deferred to A8:

| Capability | A6 Status | A8 Enabler |
|------------|-----------|------------|
| Multi-approver workflows | ❌ Deferred | Durable Functions |
| Approval escalation | ❌ Deferred | Durable timers |
| Approvals > 24h | ❌ Deferred | External events |
| Conditional approval routing | ❌ Deferred | Activity chains |

**Rationale:** A6 is MVP. Building approval complexity without Durable Functions creates tech debt. Better to have clear scope than half-implemented features.

#### Cost Baseline

| Service | SKU | Monthly |
|---------|-----|---------|
| App Service | P1v3 | $81 |
| Cosmos DB | Serverless | ~$10 |
| Blob Storage | Hot | ~$2 |
| Service Bus | Standard | $10 |
| Static Web Apps | Standard | $9 |
| Event Grid | Per-event | ~$1 |
| **Total** | | **~$113/mo** |

#### Migration Triggers (A6 → A8)

| Metric | Threshold | Action |
|--------|-----------|--------|
| Approval workflows > 1 approver | Any | Evaluate A8 |
| Approval duration > 24h | Common | Migrate to A8 |
| Runs > 500/day sustained | 1 week | Evaluate scale-out or A8 |
| Tenants > 10 | Any | Evaluate per-tenant isolation |

#### Open Questions for A6 Team

1. **Idempotency keys:** Do runbooks need idempotency tokens for safe retry after crash?
2. **Trace retention:** How long must JSONL traces be kept? (Compliance requirement)
3. **Blob immutability:** Enable WORM policy from day one or defer?
4. **Multi-region:** Is geo-redundancy required for MVP or can wait?

---

### Interactive Waiting Patterns: User Input Gate

**Author:** Barbara (Lead / Architect); correction by Germán  
**Date:** 2026-06-03T22:50:37-04:00  
**Status:** Adopted  
**Full document:** `design/web-platform/a6-architecture.md` (Interactive Waiting Patterns section)

#### Context

GERT runbooks have two distinct types of execution suspension:

1. **Approval steps** — external human (admin, manager) must approve/reject. Long wait (hours/days). Notification-driven.
2. **User input steps** — the portal session user provides input (choice, text, file, confirmation, form). Short wait (seconds/minutes). Session-driven.

Both require runtime suspension inside `IRunHandle.NextAsync()`, but have different actors, timeouts, notification channels, and resume mechanics.

#### Decision: IUserInputGate Interface

Rename `IChoiceGate` (too narrow) to `IUserInputGate` (general primitive):

```csharp
public interface IUserInputGate
{
    Task<UserInputResponse> AwaitUserInputAsync(
        string runId, string stepId, UserInputRequest request, CancellationToken ct = default);
    Task SubmitUserInputAsync(string runId, string stepId, UserInputResponse response);
    Task<PendingUserInput?> GetPendingInputAsync(string runId);
}
```

**Supported input kinds:**
- `Choice` — user selects from predefined options
- `Text` — free-form text entry
- `Confirmation` — yes/no decision
- `FileUpload` — file upload
- `Form` — multi-field structured input

#### A6 Implementation: TaskCompletionSource + SignalR

- **Approval gates** use polling (5s interval against Cosmos DB) — acceptable for long waits
- **User input gates** use `TaskCompletionSource` with SignalR push — sub-second resume for in-session interaction

Rationale: Portal user is actively in-session. 5-second poll latency is poor UX. TCS + SignalR gives instant feedback.

#### Storage: Cosmos DB Container `user_inputs`

Partition key: `/runId`. Stores pending user input for resilience (App Service restart recovery) and portal reconnect scenarios.

#### Timeout: 15 Minutes (vs 24h for Approvals)

User input steps timeout at 15 minutes. If the user abandons the session, the run fails with `user_input_timeout`. Rationale: user is actively in-session; timeout = fail fast.

#### API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/runs/{id}/steps/{stepId}/input` | POST | Portal submits user input response |
| `/runs/{id}/steps/{stepId}/input/pending` | GET | Portal reads pending input request |

#### A8 Migration Seam

Both `IApprovalGate` and `IUserInputGate` swap to Durable Functions behind DI registration:
- A6: TCS + SignalR; A8: `WaitForExternalEvent<UserInputResponse>()`
- Zero runtime code changes required

#### JSONL Audit Trail

All user input interactions generate pre/post-wait events:
- `user_input_requested` — logged before the wait
- `user_input_received` — logged after response observed
- Event includes `kind` (Choice, Text, etc.) for audit

#### Impact on Other Agents

- **Don (Backend):** Implement `IUserInputGate` + `IApprovalGate` in runtime. Both fire inside `NextAsync()`.
- **David (Integration):** New webhook events: `user_input_requested`, `user_input_received`, `user_input_timed_out`.
- **John (Azure):** New Cosmos DB container `user_inputs` with partition key `/runId`. SignalR Service may be needed at scale (in-process for MVP).

---

### C# Governance Parity Design for GERT Web Runtime

**Author:** Don (Backend Dev)  
**Date:** 2026-06-03T21:50:51-04:00  
**Status:** Adopted  
**Full document:** `design/web-platform/c-sharp-governance-parity.md`

The C# web runtime (A6: App Service + BackgroundService) must be governance-equivalent to the Go reference implementation.

#### D-1: IRunHandle.NextAsync() is the governance boundary — non-negotiable

The C# governance pipeline mirrors Go exactly:
`ParseAsync → PlanAsync → StartAsync → NextAsync (loop until null)`

`IRunHandle.NextAsync()` fires governance pre-flight (allowlist, denylist, env blocking, user input gate, approval gate, synchronous trace write) before every step dispatch. This is not optional and cannot be restructured without invalidating the audit trail.

#### D-2: IStepRunner is `internal sealed` — enforced at compile time

`IStepRunner` and all concrete step runners are `internal` to `Gert.Runtime.Core`. No adapter assembly may reference them. Bypass is a compile error.

Two Roslyn analyzers ship in `Gert.Analyzers`:
- `GERT0001` — blocks any reference to `IStepRunner` outside `Gert.Runtime.Core`
- `GERT0002` — blocks any use of `System.Text.RegularExpressions` in `Gert.Runtime.Core`

**Team action required:** Approve Roslyn analyzer approach. Fallback: namespace enforcement via architecture tests (ArchUnitNET).

#### D-3: IApprovalGate + IUserInputGate stubbed with A6/A8 swap design

Both gates have two implementations:
- A6: `CosmosDbApprovalGate`, `CosmosDbUserInputGate` (polling + durable writes)
- A8: `DurableApprovalGate`, `DurableUserInputGate` (WaitForExternalEvent)

Neither implementation touches `IRunHandle` or the governance layer. The swap is a DI registration change only.

**Team action required:** Confirm approval reply queue topology (one queue per run vs. one tenant-wide queue with correlation filter).

#### D-4: `Google.Re2` NuGet for all redaction evaluation — non-negotiable

`System.Text.RegularExpressions` is forbidden in `Gert.Runtime.Core`. `Google.Re2` (NuGet) is the only permitted regex engine. This matches Go's RE2 semantics.

Patterns using lookahead, lookbehind, possessive quantifiers, or named backreferences will throw at startup (fail-fast). Confirm `(?P<name>...)` syntax is accepted by the library.

#### D-5: Trace write is synchronous — fire-and-forget is forbidden

`IRunStore.WriteTraceAsync()` must complete (flush to durable storage) before `NextAsync()` dispatches the step. If trace write fails, the step does NOT execute and the run is cancelled.

This is a hard invariant. Any implementation that defers the write to a background queue violates the audit trail contract.

#### Validation gates — 10 gates, all must pass before production

| Gate | Criterion |
|---|---|
| G-01 | 60 minimum test vectors pass |
| G-02 | RE2 redaction parity (Go == C# for all TV-REDACT-*) |
| G-03 | Template parity (Go == C# for all TV-TMPL-*) |
| G-04 | 100% JSONL trace schema conformance |
| G-05 | Zero sequence number gaps across 1,000 synthetic runs |
| G-06 | Go `gert replay` succeeds on 100% of C#-generated traces |
| G-07 | Trace write completes before step dispatch (integration test) |
| G-08 | `IStepRunner` not in public API (Roslyn analyzer) |
| G-09 | No `System.Text.RegularExpressions` in `Gert.Runtime.Core` (Roslyn analyzer) |
| G-10 | `IApprovalGate` and `IUserInputGate` registered in DI before runtime start |

#### Open Questions Requiring Team Input

| OQ | Question | Owner |
|---|---|---|
| OQ-1 | Template engine: port Go `text/template` subset vs WASM compiled Go evaluator? | Don |
| OQ-2 | Approval reply queue: one per run or tenant-wide with correlation filter? | Don + John |
| OQ-3 | Canonical Go test harness author for TV-TMPL-* outputs? | Don |
| OQ-4 | Sequence counter: in-memory atomic + checkpoint or always read from RunStore? | Don |
| OQ-5 | `Google.Re2` NuGet named group syntax: confirm `(?P<name>...)` support | Don |

---

### Azure Service Stack: Hard Constraints & Recommended Services
**Author:** John (Azure Platform Engineer)  
**Date:** 2026-06-03  
**Status:** Adopted

**Compute recommendation:** App Service P1v3 (selected for A6) + Container Apps for A8  
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

**Database recommendations:**
- Cosmos DB Serverless for mutable state (run status, approvals, user inputs)
- Blob Storage (append blobs) for immutable audit trail
- New container `user_inputs` alongside existing containers

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
**Status:** Adopted

**Red Line 1: No pattern may bypass** `Runtime.Start → RunHandle.Next`  
→ Eliminates direct tool invocation, per-step Durable Functions, step result spoofing

**Red Line 2: Trace MUST be written only by Runtime Core**  
→ Eliminates external process trace writes, synthetic event injection, middleware appends

**Red Line 3: Exactly one live Runtime Core per run_id at a time**  
→ Requires at-most-once queue semantics (Service Bus sessions) + distributed lock on run record

**Red Line 4:** `client: "web"` MUST be set in RunOptions for all web-triggered runs  
→ Audit trail integrity: admins must isolate web vs CLI vs TUI runs

**Red Line 5: Approval gates + User input gates require pause/resume protocol**  
→ Worker must checkpoint via RunStore if it can't stay alive through gate window

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
**Status:** Adopted

**Baseline v1.0:**
- **Delivery model:** Retry + exponential backoff (5 attempts over 24h: 1m, 5m, 30m, 2h, 6h)
- **Subscription model:** Per-execution webhook (URL provided at run submission)
- **Event types:** 
  - Execution lifecycle: `execution.started`, `execution.finished`, `execution.failed`
  - Step events: `step.started`, `step.completed`, `step.failed`
  - Interaction events: `user_input_requested`, `user_input_received`, `user_input_timed_out`
  - Approval events: `approval.requested`, `approval.approved`, `approval.rejected`, `approval.timed_out`
- **Event payload:** Summary events only (not full trace)
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

---

## Proposals — Awaiting Team Ratification

These entries are output from the 2026-06-03 declaration-before-process brainstorm session (agents: Barbara, Don, David, Leslie). No decisions finalized until team consensus.

---

## Campaign Architecture & White-Label Platform Decisions

### 1. Shared White-Label Frontend as Default

**Author:** Barbara (Lead / Architect)  
**Date:** 2026-06-04T12:28:39.293-04:00  
**Status:** Needs team ratification

**Decision request:** Default to one shared Azure Static Web Apps frontend, fronted by Azure Front Door, with tenant branding/domain resolved by hostname. Do **not** provision one frontend deployment per tenant unless a tenant has explicit isolation, networking, or regulatory requirements.

**Why it needs ratification:** This affects Leslie's frontend architecture, John's DNS/network plan, and cost/isolation expectations in sales conversations.

### 2. Campaign Layer is a First-Class Platform Surface

**Decision request:** Introduce explicit campaign entities and APIs: `Tenant`, `IdentityIntegration`, `Campaign`, `AudienceMember`, `Invitation`, `ContractArtifact`, `Run`, and `NotificationSubscription`.

**Why it needs ratification:** Without this, each adapter will invent its own partial state model. This is an architecture seam, not an implementation detail.

### 3. Invitation Token is Not Sufficient Identity Assurance

**Decision request:** Require customer identity validation before contract review/acceptance steps open. Invitation link proves possession of email access; it does not replace identity verification.

**Why it needs ratification:** This is a product/scope decision with major UX and legal consequences. If the team wants "email click only," it must be an explicit low-assurance mode, not an accidental default.

### 4. Two-Stage Invitation Redemption

**Decision request:** Implement invitation redemption as GET landing page + explicit POST redeem action, so email security scanners do not consume one-time links.

**Why it needs ratification:** This changes portal behavior, token semantics, and support expectations. It is also the cleanest mitigation for enterprise mail gateway link prefetch.

### 5. Customer Identity Validation Must Be Server-to-Server

**Decision request:** The portal submits challenge answers to GERT API; App Service calls the customer identity endpoint server-to-server. Browser-direct calls into customer infrastructure are disallowed.

**Why it needs ratification:** This drives CORS posture, secret handling, network topology, and the private connectivity story for enterprise customers.

### 6. Notification Delivery = Signed Webhook + Pull Reconciliation

**Decision request:** Customer completion notification must include both a signed webhook path and a pull reconciliation API. Webhook-only is insufficient.

**Why it needs ratification:** David's outbound delivery design and customer integration contract depend on this. Operationally, this is the difference between "we sent it" and "they can recover truth."

### 7. Pinned Contract Artifact Hash and Runbook Version Per Campaign

**Decision request:** Every campaign launch must pin an immutable contract artifact version/hash and immutable runbook version. "Latest" pointers are forbidden for active campaigns.

**Why it needs ratification:** This is required for auditability and legal defensibility. It also affects campaign authoring UX and deployment pipelines.

### 8. Dedicated Networking/Infra Tier for Private Customer APIs

**Decision request:** If a customer's identity endpoint is private, support it through a premium isolation/networking tier (for example App Service VNet integration + NAT/VPN/private routing), not as an ad hoc exception inside the shared baseline.

**Why it needs ratification:** John needs a stable network product offering, and sales needs a clear answer on what is "standard" versus "premium."

---

## Integration Contracts & Event Delivery Decisions

### 1. Identity Check Timeout + Circuit Breaker Thresholds

**Author:** David (Integration Engineer)  
**Date:** 2026-06-04T12:28:39.293-04:00  
**Status:** Proposed

**Proposal:** identity endpoint call budget is 8 seconds absolute; per-endpoint circuit breaker opens after 5 consecutive transient failures and stays open for 5 minutes.

**Why it needs ratification:** this changes customer UX during link redemption and sets the operational boundary for partial outage handling.

**Trade-off:** shorter timeout protects portal UX; longer timeout may improve success for slow customer APIs but increases abandoned sessions.

### 2. Campaign Link Storage Shape in Cosmos DB

**Proposal:** keep campaign links in the existing `campaigns` container as separate `campaign_link` documents, not embedded arrays inside a campaign root document.

**Why it needs ratification:** it affects partitioning, hot-document risk, and future migration to a dedicated container.

**Trade-off:** fastest path now, but high-scale campaigns may later need a dedicated `campaign_links` container.

### 3. Webhook Registration Scope for Campaign Integrations

**Proposal:** customer onboarding registers webhook endpoints at tenant/campaign scope even though the current adopted baseline says per-execution webhook in v1.0.

**Why it needs ratification:** this is a product-surface decision, not just an implementation detail.

**Trade-off:** tenant/campaign registration is operationally cleaner for recurring campaigns; per-execution registration is simpler for initial platform scope.

### 4. Webhook Retry Schedule Interpretation

**Proposal:** keep the adopted policy of 5 retries over 24 hours, implemented as immediate attempt plus retries at approximately +1m, +5m, +30m, +4h, and +18h with jitter.

**Why it needs ratification:** existing notes say "5 attempts over 24h" but the sample intervals recorded elsewhere do not actually span 24 hours.

**Trade-off:** longer tail protects against customer outages; shorter tail moves events to DLQ faster for manual replay.

### 5. Default External Event Set

**Proposal:** default subscriptions include `campaign.closed`, `user.started`, `user.completed`, and `user.failed`; `user.abandoned`, `user.identity_denied`, `user.identity_error`, and milestone events remain opt-in.

**Why it needs ratification:** this determines the stable public contract and webhook volume customers must handle.

**Trade-off:** smaller default set is easier for customers; larger default set gives better operational visibility.

---

## Local Development & Aspire Integration Decisions

### 1. Adopt Hybrid Local Development for A6

**Author:** John  
**Date:** 2026-06-04T12:28:39.293-04:00  
**Status:** Proposed

**Decision:** Use Aspire for local orchestration of API/worker + Service Bus emulator + Blob/Azurite. Use Cosmos emulator only where it is stable on the developer machine; otherwise allow a real dev Cosmos account. Use real Azure dev resources for Entra External ID/B2C and Event Grid.

### 2. Keep A6 Local SignalR as In-Process ASP.NET Core SignalR

**Decision:** Do not force Azure SignalR into the local loop for MVP. Revisit only if cloud topology explicitly requires Azure SignalR beyond the current in-process hub model.

### 3. Treat Aspire App Service Integration as Deployment Plumbing

**Decision:** Local A6 development should run the ASP.NET Core app and BackgroundService directly. App Service modeling belongs to publish/deployment concerns.

### 4. Define Two Supported Dev Modes

**Decision:** Support two development modes:
- **Fast inner loop:** local app + emulators + dev auth bypass.
- **Cloud integration loop:** local app or deployed app + real External ID/Event Grid (+ optional real Cosmos/SignalR).

### 5. Do Not Block on "100% Local Aspire Parity"

**Decision:** Team should explicitly accept that some Azure services are cloud-only and that this is not a failure of the architecture.

---

## Portal Frontend & UX Design Decisions

### 1. MVP Frontend Topology = Shared SWA

**Author:** Leslie (Frontend Dev)  
**Date:** 2026-06-04T12:28:39.293-04:00  
**Status:** Needs team ratification

**Proposal:** Use one Azure Static Web Apps Standard frontend for the shared portal shell, with tenant resolution by hostname and runtime config. Dedicated SWA only for enterprise/regulatory isolation needs.

**Why:** Lowest operational overhead, one release train, branding without redeploys.

### 2. Domain Strategy = Explicit Per-Tenant Custom Domains

**Proposal:** Bind each tenant hostname explicitly in SWA (`acme.myservice.com`, `portal.customer.com`).

**Why:** Cleaner onboarding/offboarding, clearer certificate ownership, less DNS ambiguity.

**Open question:** Do we need wildcard support later for high-volume self-serve onboarding?

### 3. CDN/WAF Strategy = SWA Edge First, Front Door Later

**Proposal:** Rely on Static Web Apps built-in edge delivery for MVP. Escalate to Front Door when: WAF, multi-region routing, canary releases across SWAs, or centralized advanced domain strategy becomes necessary.

### 4. Theme Storage = Cosmos DB Metadata + Blob Asset Storage

**Proposal:** Store mutable portal config in Cosmos DB and logos/fonts/legal assets in Blob Storage.

**Why:** Runtime rebranding without redeploys; matches platform split of mutable state vs large artifacts.

### 5. Auth UX = Magic-Link-First

**Proposal:** The invitation flow should not require end users to create an account or enter a password. After link validation, create a secure session transparently; if stronger proofing is needed, show a short secure verification interstitial.

**Open question:** Should an optional authenticated records portal be part of MVP or explicitly Phase 2?

### 6. Resume/Concurrency Model = Resumable Progress, Single Active Writer

**Proposal:** Submitted progress persists server-side; unsaved drafts persist locally on the same device; only one device can actively submit at a time.

**Why:** Safer than allowing concurrent submissions and easier for users to understand.

### 7. Audit Branding Boundary

**Ratification needed:** Must `Powered by GERT` or another audit-visible footer remain present in every white-label portal/completion/receipt experience?

**Frontend recommendation:** If required, keep it subtle and consistent in the footer/receipt rather than in the primary action area.

---

# Declaration-Before-Process Scenario Taxonomy

**Author:** Barbara (Lead / Architect)  
**Date:** 2026-06-03T23:53:24-04:00  
**Status:** Brainstorm — input to runbook template design  
**Requested by:** Germán (ormasoftchile)

---

## Purpose

Before building runbook templates for "declaration before process" flows, we need to understand the full scenario universe, classify what varies, identify recurring archetypes, and draw explicit lines between what GERT owns vs. what it integrates with. This document is that foundation.

---

## 1. Scenario Universe — 22 Concrete Scenarios

| # | Scenario | Industry | Form Name | Gated Process |
|---|----------|----------|-----------|---------------|
| S-01 | Patient signs informed consent for elective surgery | Healthcare | Surgical Informed Consent | OR scheduling + procedure execution |
| S-02 | Patient signs separate anesthesia consent with different risk profile | Healthcare | Anesthesia Risk Declaration | Anesthesia administration |
| S-03 | Jehovah's Witness patient refuses blood transfusion — documented refusal | Healthcare | Informed Refusal of Treatment | Blood transfusion order bypass |
| S-04 | Parent consents to non-emergency procedure for minor child | Healthcare | Parental/Guardian Consent | Pediatric procedure scheduling |
| S-05 | Patient enrolls in clinical trial (IRB-governed dual-witness) | Healthcare | Clinical Trial Informed Consent | Trial enrollment + randomization |
| S-06 | Patient consent to release records to third party (insurer, employer) | Healthcare | HIPAA Authorization | Records release job |
| S-07 | Pre-insurance health declaration before life/health policy issuance | Insurance | Underwriting Health Declaration | Policy issuance + premium calculation |
| S-08 | Claims fraud declaration — sworn statement before claim processing | Insurance | Fraud Attestation | Claims adjudication |
| S-09 | Client acknowledges crypto investment risk before account activation | Finance | Crypto Risk Acknowledgment | Trading account unlock |
| S-10 | Suitability assessment for complex investment product (MiFID II) | Finance | Suitability Declaration | Product purchase authorization |
| S-11 | Mortgage applicant signs RESPA/TILA loan disclosure before loan origination | Finance | Loan Disclosure Acknowledgment | Loan processing trigger |
| S-12 | New bank account KYC/AML self-certification | Finance | KYC Declaration | Account opening |
| S-13 | Seller discloses known property defects before real estate closing | Real Estate | Property Condition Disclosure | Escrow/closing process |
| S-14 | Tenant signs move-in condition declaration before receiving keys | Real Estate | Move-In Condition Report | Key handover |
| S-15 | New hire authorizes background check before hiring decision | Employment | Background Check Authorization | Background screening trigger |
| S-16 | Employee acknowledges OSHA safety training completion before site access | Employment | Safety Training Acknowledgment | Site/equipment access grant |
| S-17 | Employee signs NDA before accessing confidential materials | Employment | Non-Disclosure Agreement | Asset/system access provisioning |
| S-18 | Benefits applicant self-certifies income/eligibility before disbursement | Government | Eligibility Self-Certification | Benefits payment processing |
| S-19 | Citizen declares immigration/customs information before entry processing | Government | Customs/Immigration Declaration | Entry clearance |
| S-20 | Student acknowledges loan terms before loan disbursement (MPN) | Education | Master Promissory Note | Student loan disbursement |
| S-21 | Research subject signs IRB consent before study participation | Education | Human Subjects Research Consent | Subject enrollment |
| S-22 | Event participant signs liability waiver before extreme sports activity | Recreation/Legal | Activity Liability Waiver | Activity entry / equipment issue |

---

## 2. Classification Along GERT-Relevant Axes

### 2a. Who Declares? Who Witnesses/Co-Signs?

| # | Who Declares | Witness / Co-Sign | Notes |
|---|-------------|-------------------|-------|
| S-01 | Patient (adult, competent) | Doctor countersign | Mutual signature required |
| S-02 | Patient (adult, competent) | Anesthesiologist | Separate signer from surgeon |
| S-03 | Patient (adult, competent) + legal rep | Doctor + second witness | Two independent witnesses; patient must demonstrate capacity |
| S-04 | Parent or legal guardian | Doctor | Guardian identity must be verified against patient record |
| S-05 | Patient + PI (Principal Investigator) | IRB-designated witness | Dual-party signature; each party's signature timestamped independently |
| S-06 | Patient | None or notary (jurisdiction-dependent) | |
| S-07 | Applicant | None (self-declaration) | Fraud risk — fraud attestation clause within the form |
| S-08 | Claimant | None (sworn under penalty of perjury) | |
| S-09 | Account holder | None | |
| S-10 | Client + Financial Advisor | Compliance Officer (some jurisdictions) | Advisor attests suitability assessment was conducted |
| S-11 | Borrower (co-borrower if applicable) | Loan officer | Each borrower signs independently |
| S-12 | Account applicant | None (KYC officer internal review, not co-sign) | KYC officer action is approval, not co-sign |
| S-13 | Seller | Buyer acknowledgment receipt | Seller declares; buyer acknowledges receipt — different acts |
| S-14 | Tenant | Property manager countersign | Disputed condition items require both parties |
| S-15 | Applicant | None | |
| S-16 | Employee | Supervisor countersign | Supervisor attests training was observed |
| S-17 | Employee | HR representative | Some jurisdictions require notarization for enforceability |
| S-18 | Applicant | None (caseworker review is approval step, not co-sign) | |
| S-19 | Traveler | None | |
| S-20 | Student (co-signer if minors/parent) | None | |
| S-21 | Research subject (guardian if minor) | Investigator | |
| S-22 | Participant | Event staff witness (often optional) | |

### 2b. Pre-Conditions Before Form Is Presented

| # | Pre-Conditions Required | Who/What Checks Them |
|---|------------------------|----------------------|
| S-01 | Patient identity verified; diagnosis confirmed; surgical plan approved by surgeon; patient has capacity | EHR system + clinical staff |
| S-02 | S-01 consent completed; pre-op labs within range | EHR + clinical system |
| S-03 | Patient capacity assessed (mental competency evaluation); attending physician confirms | Clinical assessment tool |
| S-04 | Guardian relationship verified (birth cert / legal document); patient age confirmed | Identity document check |
| S-05 | Eligibility criteria met (inclusion/exclusion criteria screened); IRB approval active | Trial management system |
| S-06 | Patient identity confirmed | Identity verification |
| S-07 | Applicant identity verified (KYC); previous policy history checked | Underwriting system + KYC |
| S-08 | Claim number exists; claimant identity matched to policy | Claims system |
| S-09 | Account exists; AML screening passed; country-of-residence eligible | AML engine + geo-check |
| S-10 | Client onboarded (KYC complete); suitability questionnaire completed | CRM + compliance system |
| S-11 | Loan application accepted; property appraisal complete; title search clear | Loan origination system |
| S-12 | Identity document verified; sanctions/PEP screening passed | KYC/AML engine |
| S-13 | Escrow opened; purchase agreement executed | Real estate transaction system |
| S-14 | Lease signed; first payment received | Property management system |
| S-15 | Conditional offer made by hiring manager | HRIS / ATS |
| S-16 | Training session completed (attendance recorded) | LMS |
| S-17 | Onboarding step N-1 complete; access request submitted | HR onboarding system |
| S-18 | Application received; identity verified; income documentation submitted | Benefits management system |
| S-19 | Boarding pass / travel document scanned | Customs system |
| S-20 | Enrollment confirmed; loan application approved | Financial aid system |
| S-21 | Screening eligibility confirmed; protocol approved | IRB system |
| S-22 | Age verified (if applicable); equipment briefing completed | On-site check |

### 2c. What Downstream Process Is Gated?

| # | Gated Process | Trigger Mechanism |
|---|--------------|-------------------|
| S-01 | Surgery scheduling | Scheduling system API call post-consent |
| S-02 | Anesthesia administration order | EHR order entry |
| S-03 | Treatment plan modification in EHR | EHR write operation |
| S-04 | Procedure order creation | EHR system |
| S-05 | Trial enrollment database entry | CTMS (Clinical Trial Management) |
| S-06 | Records export job | Document management system |
| S-07 | Policy issuance + premium calculation | Underwriting engine |
| S-08 | Claims adjudication workflow | Claims processing system |
| S-09 | Trading account activation | Brokerage platform |
| S-10 | Product purchase authorization | Order management system |
| S-11 | Loan file submission to underwriting | LOS (Loan Origination System) |
| S-12 | Account creation in core banking | Core banking system |
| S-13 | Title transfer / closing | Escrow + title company |
| S-14 | Key issuance | Property management system |
| S-15 | Background check vendor API call | Screening vendor (e.g., Sterling, Checkr) |
| S-16 | Site access badge activation | Physical access system |
| S-17 | System/asset access provisioning | IAM/ITSM system |
| S-18 | Benefits payment initiation | Payment processor |
| S-19 | Entry clearance / boarding | Border management system |
| S-20 | Loan funds disbursement | Loan servicer |
| S-21 | Subject randomization / kit assignment | CTMS |
| S-22 | Equipment checkout / activity entry | On-site system or manual |

### 2d. Identity Proofing Level Required

| Level | Description | Scenarios |
|-------|-------------|-----------|
| **L1 — Basic Auth** | User authenticated to the portal only | S-09, S-22 (low-stakes) |
| **L2 — Verified Account** | Email-verified account + session consistency | S-16, S-17, S-20 |
| **L3 — Document-Backed** | Government ID verified (OCR + liveness) | S-07, S-12, S-13, S-15, S-18, S-19 |
| **L4 — KYC-Grade** | Full KYC: document + AML/PEP screening | S-10, S-11, S-12 |
| **L5 — Clinical/Legal** | In-person capacity assessment + clinician attestation | S-01–S-06, S-08, S-21 |
| **L5+ — Biometric** | Biometric signature capture (not just e-sig) | S-05, clinical trial enrollment in some jurisdictions |

### 2e. Retention / Audit Requirements

| # | Retention Period | Regulator / Standard | Immutability Requirement |
|---|-----------------|----------------------|--------------------------|
| S-01 | 10–25 years (varies by state/country) | Joint Commission, CMS, state medical boards | WORM — medical record |
| S-02 | Same as S-01 | Same | Same |
| S-03 | Same + incident file | Same + ethics committee | Same |
| S-04 | Until patient age 18 + 10 years | State law | WORM |
| S-05 | Life of trial + 15 years post-approval | 21 CFR Part 11, ICH E6 | Electronic records rule (21 CFR 11) — full audit trail |
| S-06 | 6 years | HIPAA | WORM |
| S-07 | Policy life + 7–10 years | State insurance commissioner | WORM |
| S-08 | 5–10 years | State, NAIC | WORM |
| S-09 | 5 years | FINRA, SEC, MiFID II | WORM (if applicable) |
| S-10 | 5 years | MiFID II, SEC | WORM |
| S-11 | 3 years post-payoff | CFPB, RESPA/TILA | Immutable |
| S-12 | 5 years post-closure | FinCEN, BSA | WORM |
| S-13 | 5–7 years | State real estate board | Immutable |
| S-14 | Lease life + 3–7 years | State landlord/tenant law | Standard |
| S-15 | 5 years or duration of employment + 3 | FCRA, EEOC | Immutable |
| S-16 | Employment duration + 5 years | OSHA | Standard |
| S-17 | Duration of NDA term + 3–7 years | Contract law | Standard |
| S-18 | 3–7 years | SSA, state welfare agencies | Immutable |
| S-19 | 5–7 years | CBP, DHS | WORM |
| S-20 | Life of loan + 10 years | Dept. of Education, FERPA | Immutable |
| S-21 | Life of study + 15 years | FDA, NIH, 21 CFR Part 11 | WORM + audit trail |
| S-22 | 3–7 years (statute of limitations) | Tort law | Standard |

### 2f. Revocability

| Pattern | Revocable? | What Happens to Gated Process |
|---------|-----------|-------------------------------|
| **Ongoing treatment consent** (S-01, S-02) | Yes, at any time pre-procedure | Scheduled procedure must be cancelled/paused; EHR flagged |
| **Refusal declaration** (S-03) | Yes, patient can revoke refusal | Treatment can proceed; new consent may be required |
| **Enrollment consent** (S-05, S-21) | Yes, withdrawal is a right (Belmont Report) | Subject withdrawn from trial; already-collected data handling per protocol |
| **Records release** (S-06) | Yes, until disclosure occurs | If revoked before execution, release job aborted |
| **Insurance declaration** (S-07) | No, post-issuance | Policy voidable only for material misrepresentation (insurer-initiated) |
| **Financial risk ack** (S-09, S-10) | No, per-transaction | Does not undo executed trades |
| **KYC declaration** (S-12) | No | Account closure is separate process |
| **Property disclosure** (S-13) | No, once delivered | Buyer may have rescission rights under state law |
| **NDA** (S-17) | No | Contract law governs; breach remedies |
| **Benefits certification** (S-18) | No, but can file correction | Overpayment recovery process |

---

## 3. Archetype Patterns

After classifying 22 scenarios, 5 archetypes emerge. These map directly to GERT runbook template categories.

### Archetype A — Single-Party Informed Consent

**Definition:** One principal reads disclosures and attests they understood. No co-signer from the same party, though a professional may countersign.

**Core flow:**
```
[PRE-CHECK] Identity verified → [DISCLOSE] Present disclosure content → 
[ACKNOWLEDGE] User confirms read → [DECLARE] User signs/attests → 
[COUNTERSIGN?] Optional professional countersign → [EMIT] Audit event → 
[TRIGGER] Downstream process
```

**Scenarios:** S-01 (minus dual-sign), S-06, S-07, S-08, S-09, S-11, S-13, S-16, S-18, S-19, S-20, S-22

**GERT runbook steps needed:** `form` (IUserInputGate), `approval` (IApprovalGate for countersign), trace emit, webhook trigger

---

### Archetype B — Two-Party Witnessed Declaration

**Definition:** Principal signs; a second independent party (witness, co-signer, professional) must separately countersign. Both signatures are independently timestamped and immutable.

**Core flow:**
```
[PRE-CHECK] Both parties' identities verified → [PRESENT] Disclosure to declarant → 
[DECLARANT SIGN] First signature captured → [AUDIT-1] Emit declarant_signed event →
[NOTIFY WITNESS] Witness notified → [WITNESS REVIEW] Witness reads content → 
[WITNESS COUNTERSIGN] Second signature captured → [AUDIT-2] Emit witness_countersigned event →
[TRIGGER] Downstream process
```

**Scenarios:** S-01 (surgeon countersign), S-02, S-05, S-10, S-14, S-16 (supervisor version)

**Key distinction from A:** The witness/co-signer is a separate actor with a separate approval step. The runbook suspends between declarant sign and witness countersign — this is an `IApprovalGate` scenario.

---

### Archetype C — Proxy / Representative Declaration

**Definition:** A third party declares on behalf of a principal who cannot (minor, incapacitated adult, deceased estate). The relationship between declarant and principal must be verified before the form is presented.

**Core flow:**
```
[PRE-CHECK] Principal identity → [VERIFY RELATIONSHIP] Guardian/POA document check → 
[CAPACITY ASSESSMENT?] If incapacity claim, clinical/legal assessment step → 
[PRESENT FORM] Proxy-specific disclosure (includes statement of authority) → 
[PROXY SIGN] Representative signature → [PROFESSIONAL COUNTERSIGN] Clinician/lawyer attests →
[AUDIT] Emit with principal_id + proxy_id + relationship_type → [TRIGGER] Downstream
```

**Scenarios:** S-03, S-04, S-21 (guardian version)

**GERT runbook steps needed:** `form` (relationship verification), `approval` (capacity/POA document upload + clinician approval), dual audit events

**Hardest seam:** Verifying the proxy relationship is almost always outside GERT (external document management, legal records, EHR). GERT's job is to collect the claim and record it — the verification step is an approval gate that asks a clinician or legal officer to confirm the document.

---

### Archetype D — Conditional Eligibility → Declaration

**Definition:** The declaration is only presented after an external system confirms the principal is eligible. The eligibility check is a prerequisite — its output may pre-fill fields in the form.

**Core flow:**
```
[IDENTITY-PROOF] → [ELIGIBILITY QUERY] External system API → 
[GATE] If not eligible: terminate with reason → 
[PRE-FILL] Populate form fields from eligibility response → 
[DISCLOSURE] Present eligibility-specific disclosure (shows what was confirmed) → 
[DECLARE] User signs → [AUDIT] Emit with eligibility_reference_id → 
[TRIGGER] Downstream (eligibility reference passed as correlation key)
```

**Scenarios:** S-05 (inclusion criteria), S-07 (prior policy check), S-10 (suitability), S-15 (offer status), S-20 (loan approval), S-21

**Critical seam:** The eligibility query is OUTSIDE GERT. GERT receives the result, records it in the trace, and gates on it — but does not own the eligibility logic. This is a `step` that calls an external API and fails/branches on the response.

---

### Archetype E — Multi-Step Disclosure-Acknowledge-Sign

**Definition:** Long-form disclosure broken into sections, each requiring explicit acknowledgment before advancing. Final signature is a meta-attestation that all sections were read. Common in heavily regulated contexts (clinical trials, financial products).

**Core flow:**
```
[PRE-CHECK] Identity + eligibility → 
[SECTION 1] Present → [ACK-1] Acknowledge → [AUDIT-1] Emit section_acknowledged →
[SECTION 2] Present → [ACK-2] Acknowledge → [AUDIT-2] Emit section_acknowledged →
... (N sections) ...
[SUMMARY] Show all acknowledged sections → 
[FINAL SIGN] Signature on aggregated declaration → [AUDIT-FINAL] Emit final_signed →
[COUNTER-SIGN?] Professional/IRB witness → [TRIGGER] Downstream
```

**Scenarios:** S-05 (ICH E6 clinical trials — mandated section acknowledgment), S-10 (complex investment product prospectus), S-11 (multi-page TILA mortgage disclosure), S-20 (student loan MPN)

**GERT runbook steps needed:** Repeated `form` steps with `confirmation` kind per section. The runbook template is parameterizable — customer configures number of sections and content per section. Each section acknowledgment emits a discrete audit event.

**Why this matters for GERT product design:** Template customers need to author N-section disclosures without writing N individual steps in YAML. There needs to be a "disclosure block" runbook primitive or a loop construct.

---

## 4. Integration Seams — What GERT Owns vs. What It Integrates With

### What GERT Owns (Runbook Steps)

| Capability | GERT Mechanism | Notes |
|-----------|----------------|-------|
| Form presentation (any section of disclosure) | `IUserInputGate` — `Form` kind | Field schema from runbook YAML |
| Section-by-section acknowledgment | `IUserInputGate` — `Confirmation` kind | Per-section event emitted to trace |
| Identity capture (collect ID claim) | `IUserInputGate` — `Form` kind | GERT collects; external system verifies |
| File upload (document collection) | `IUserInputGate` — `FileUpload` kind | Stored in Blob; reference in trace |
| Witness / co-signer approval step | `IApprovalGate` | Notified via approval webhook |
| Signature capture (e-signature) | `IUserInputGate` — `Form` kind with `signature` field type | Needs new field type definition |
| Audit emission per step | JSONL trace (automatic, synchronous) | Non-negotiable per Red Line 2 |
| Conditional branching on eligibility result | Runbook step result routing | Eligibility API call is the step; GERT routes on result |
| Timeout enforcement | `IUserInputGate` 15-min / `IApprovalGate` 24-h | Config per step in runbook YAML |
| Revocation capture | New: `form` step triggered by separate "revoke" runbook | Revocation is its own auditable process |

### What Is OUTSIDE GERT (Integration Points)

| Capability | Who Owns It | GERT Interface |
|-----------|------------|----------------|
| Identity proofing / document verification | KYC vendor (Onfido, Jumio, etc.) | Step calls external API; result in trace |
| AML / PEP / sanctions screening | Compliance engine | Step calls external API; result in trace |
| Eligibility determination | Customer's domain system (EHR, LOS, HRIS, etc.) | Step calls customer webhook; result gates progression |
| Biometric signature validation | Biometric vendor | Outside; GERT captures the signature image/token |
| E-signature legal validity enforcement | SignNow, DocuSign, or equivalent | GERT can integrate as a step, or own a lightweight capture |
| Payment processing | Payment processor | Completely outside GERT |
| Scheduling (surgery, meeting, etc.) | Customer system | Triggered by webhook POST-run |
| Translation / localization of form content | i18n layer in portal frontend | GERT stores form schema; portal renders in user language |
| Capacity / mental competency assessment | Clinical system or legal officer | Inputs to GERT as an approval decision |
| Policy issuance | Insurance core system | Triggered by webhook POST-run |
| Downstream process execution | Customer system | Webhook delivery (David's integration layer) |

### The Seam Diagram (text)

```
External Systems          GERT Boundary              Customer's Downstream
─────────────────        ──────────────────────       ──────────────────────
KYC Vendor      ──API→   [Step: verify_identity]
Eligibility Sys ──API→   [Step: check_eligibility]  
                         [Step: form (disclosure)]
                         [Step: confirmation (ack)]
                         [Step: form (signature)]
                         [Step: approval (witness)]  
                         [Trace: JSONL immutable]
                         [Webhook: declaration.signed] ──→  Surgery Scheduler
                                                       ──→  Policy Engine
                                                       ──→  Account System
```

---

## 5. Uncomfortable Questions — Surface Now, Not Later

### 5.1 Minors and Legal Capacity

- **Who is the principal in the GERT data model when a proxy acts?** The trace must record both `principal_id` (the minor/incapacitated person) and `declarant_id` (the proxy). This is a schema decision for the audit event — currently `IUserInputGate` only carries `runId` and `stepId`, not a `principal_id` distinct from the session user.
- **Age of majority varies by jurisdiction and by act.** A 16-year-old can consent to STI treatment in some US states but not surgery. GERT cannot enforce this — but GERT must capture the DOB, relationship claim, and the jurisdiction in the trace so a regulator can verify the correct rule was applied.
- **Emancipated minors.** Legal status that removes need for guardian consent. Verification is outside GERT; GERT must record the emancipation claim in the trace.

### 5.2 Mental Capacity and Competency

- S-03 (refusal) is the hardest case: a patient refusing life-saving treatment. If a clinician asserts incapacity, the patient's declaration may be legally void — but the GERT trace still exists as evidence. GERT should NOT make a capacity determination. The trace must record the clinician's assertion and the outcome of any capacity hearing.
- **Dementia / progressive conditions.** Consent given today may be legally questioned retroactively. Time-of-signature metadata + video/biometric record may be required. GERT owns the timestamp and form capture; biometric is outside GERT.

### 5.3 Language and Translation

- Informed consent requires that the declarant understood the content. Presenting a form in English to a Spanish speaker may invalidate the consent.
- **GERT does not own translation.** But GERT must record which language version of the disclosure was presented — this is an audit field on the form step event. If a customer presents a form without recording the language, the audit trail is incomplete.
- **Recommendation:** Add `display_language` and `disclosure_version_hash` as mandatory fields on every `form`/`confirmation` step event. The portal frontend controls rendering; GERT records which version was shown.

### 5.4 E-Signature Legal Validity

- **ESIGN Act (US):** Electronic signatures are valid for most consumer contracts. Exceptions: wills, certain family law matters, court orders.
- **eIDAS (EU):** Three tiers — Simple Electronic Signature (SES), Advanced (AES), Qualified (QES). Clinical trial consent in the EU under 21 CFR Part 11 equivalent requires at minimum AES.
- **Chile:** Law 19.799 recognizes electronic signatures; qualified electronic signature (FEA) required for acts that traditionally require notarization (including some real estate and POA documents).
- **GERT's position:** GERT should capture a signature token. For high-assurance scenarios, that token must come from a qualified trust service provider (e.g., eID integrations, DocuSign QES, FirmaVirtual in Chile). GERT is the orchestrator and trace emitter — not the signature authority. **Recommendation:** Define a `SignatureToken` type in `IUserInputGate` Form responses, with fields: `provider`, `token`, `assurance_level` (SES/AES/QES), `timestamp`, `certificate_reference`. The provider does the legal heavy lifting; GERT records what was produced.

### 5.5 Withdrawn Consent + Already-Executed Process

- **If consent is revoked after the downstream process starts:** GERT's job was to gate the process — once triggered, GERT's runbook is complete. The revocation must be a separate, independently audited runbook (a "revocation flow" template).
- **Key design question:** Should GERT emit a `consent.revoked` event that customer systems can subscribe to for compensation logic (cancel surgery, pause claim, etc.)? YES — this should be a webhook event on the downstream integration layer. GERT doesn't execute the compensation; it records the revocation and notifies.
- **Audit implication:** The original declaration trace and the revocation trace must be linkable by a common `declaration_id` or `consent_reference`. This is a new field — currently runs are only linked by `runId`. A parent/child run relationship or an external correlation ID field is needed.

### 5.6 Multi-Jurisdiction Compliance in a Single Platform

- A white-labeled GERT portal used by a healthcare provider in Chile, the EU, and the US must handle three different consent validity regimes simultaneously.
- **GERT's position:** GERT should record jurisdiction as a mandatory trace field. Customers configure jurisdiction in the runbook template. GERT does NOT validate that the correct legal standard was met — that is a customer legal responsibility. But the trace provides the evidence for that validation.

### 5.7 The "Form" Input Kind Is Not Enough for Some Scenarios

- S-05 (clinical trial) under 21 CFR Part 11 requires: audit trail of who presented the form, who accessed it, how many times it was accessed before signing, and that the signer was not coerced (witnessed in-person for some protocols).
- **Current `IUserInputGate` Form kind** captures the submission. It does not capture form-open events, scroll depth, time-on-page, or in-person witness presence.
- **Recommendation for high-assurance scenarios:** Add a `form_interaction_log` event type (form_opened, section_scrolled, form_abandoned, form_submitted) as trace events. This is not about UX analytics — it is about demonstrating non-coercion to a regulator.

### 5.8 Dual-Blind Scenarios (When Both Parties Cannot See Each Other's Response)

- Clinical trial IRB consent: the investigator must confirm they answered all questions to the subject's satisfaction, but must not be able to see the subject's answers before co-signing (conflict of interest concern).
- This requires parallel, independent runbook flows that join at a countersign step. Current GERT model is sequential. Multi-party parallel execution is not in scope for A6 — raise as A8 scenario.

---

## 6. Summary: Archetype → Template Design

| Archetype | Template Name | Min GERT Steps | Key Gates | Key Audit Events |
|-----------|--------------|----------------|-----------|-----------------|
| A — Single-Party Consent | `single-consent` | 3 (disclose, acknowledge, sign) | IUserInputGate | `consent_disclosed`, `consent_acknowledged`, `consent_signed` |
| B — Two-Party Witnessed | `witnessed-declaration` | 5 (+ notify witness, await countersign) | IUserInputGate + IApprovalGate | + `witness_notified`, `witness_countersigned` |
| C — Proxy Declaration | `proxy-consent` | 6 (+ verify relationship, capacity gate) | IUserInputGate + IApprovalGate (capacity) | + `proxy_declared`, `relationship_verified`, `capacity_confirmed` |
| D — Conditional Eligibility | `eligibility-then-consent` | 4 (+ eligibility step, branching) | External API result gate | + `eligibility_confirmed`, `eligibility_reference_id` in all events |
| E — Multi-Step Disclosure | `sectioned-disclosure` | 3 + N×2 (N sections, each: present + ack) | IUserInputGate per section | + `section_N_acknowledged` per section, `disclosure_version_hash` |

---

## 7. Recommended Additions to GERT Data Model (Flagged for Don)

Based on this analysis, the following fields/types are missing from current design and must be addressed before implementing declaration scenarios:

| Gap | Impact | Priority |
|-----|--------|----------|
| `principal_id` distinct from `session_user_id` on trace events | Proxy/representative flows impossible without this | HIGH |
| `display_language` + `disclosure_version_hash` on form step events | Language/translation audit gap | HIGH |
| `SignatureToken` type with `provider`, `assurance_level`, `certificate_reference` | E-sig legal validity not capturable | HIGH |
| `declaration_id` as cross-run correlation field | Revocation cannot be linked to original consent | MEDIUM |
| `form_interaction_log` events (form_opened, section_scrolled) | 21 CFR Part 11 high-assurance gap | MEDIUM (defer to post-A6) |
| Parent/child run relationship | Multi-party parallel flows (Archetype B variant, dual-blind) | LOW (A8 scope) |

---

*This document is input for runbook template design sprint. Next: prioritize which 2–3 archetypes ship in v1 runbook library, and confirm data model gaps with Don before schema is finalized.*

---

# Declaration/Consent Scenario Family — Runtime Gap Analysis

**Author:** Don (Backend Dev)  
**Date:** 2026-06-03T23:53:24-04:00  
**Status:** Gap analysis — not yet a design decision  
**Requested by:** Germán (ormasoftchile)  

---

## Scope

This document maps the **declaration-before-process scenario family** against GERT's current runtime primitives. It identifies what already fits, what is missing, and what new interfaces or trace event types would be needed to support production use.

**Scenario family (seeds from Germán):**
- Surgical consent (patient signs before procedure)
- Exemption-of-responsibility forms (liability waiver before activity)
- Pre-insurance declarations (health/risk disclosure before policy issuance)
- Any form requiring a registered declaration before a downstream process can proceed

These share a common pattern: **collect a legally significant statement from an identified person → register it immutably → gate the downstream process on its existence**.

---

## 1. What We Have — Existing Primitives Mapped to Scenario Steps

### 1.1 IUserInputGate Kinds

| Input Kind | Scenario Step That Maps | Fits? | Notes |
|---|---|---|---|
| `Choice` | "Do you consent to general anaesthesia? [Yes / No / Partial]" | ✅ Fits | Binary or multi-option consent selection |
| `Text` | "State any known drug allergies in your own words" | ✅ Fits | Free-text declarations, beneficiary names, exclusion clauses |
| `Confirmation` | "I confirm I have read and understood the above terms" (tick-box / yes/no) | ✅ Fits — **this is the closest current primitive to a declaration acknowledgment** | Maps to legal acknowledgment but lacks signature semantics |
| `FileUpload` | "Upload a copy of your government-issued ID" / "Attach the signed consent PDF from your physician" | ✅ Fits for file evidence collection | Does not capture *who* signed, does not verify content |
| `Form` | Multi-field pre-insurance health declaration (BMI, smoking status, pre-conditions) | ✅ Fits | Schema-driven; suitable for structured declarations |

**Verdict:** The five existing kinds cover the *data collection* mechanics of a declaration. What they do not cover is **identity binding, signature semantics, legal validity markers, or provenance of the presented document**. A declarant clicking "Confirmation" leaves no cryptographic or legal trail beyond the trace event timestamp.

---

### 1.2 IApprovalGate

| Scenario Step | Maps? | Notes |
|---|---|---|
| Witnessing physician counter-signs consent (hospital scenario) | ✅ Partial fit | A second human approver exists, which maps to `IApprovalGate`. But the "approver" here is a *witness*, not an authorizer — semantics differ. IApprovalGate models a binary approve/reject decision; a witness attestation may need to carry their own signature artifact. |
| Notary validation of a liability waiver | ✅ Partial fit | Same witness-vs-approver ambiguity. A notary is attesting presence and identity, not approving content. |
| Insurance underwriter reviews declaration before policy issuance | ✅ Good fit | This is a genuine approval (underwrite / decline) — maps cleanly to `IApprovalGate`. |
| Legal guardian counter-signs on behalf of minor | ❌ Does not fit | Requires two distinct identity bindings (subject + declarant) on the *same* trace event. IApprovalGate has one actor. |

**Verdict:** IApprovalGate works where there is a downstream authorizer (underwriter, administrator). It does not work for co-signers, witnesses, or guardian-on-behalf-of flows because those need multi-actor identity on a single declaration event.

---

### 1.3 JSONL Trace as Authoritative Record

This is a **strong existing fit** for the "must be registered" requirement.

| Requirement | GERT Capability | Status |
|---|---|---|
| Declaration must be registered before downstream process proceeds | Trace write is **synchronous before step dispatch** — if write fails, process does not advance | ✅ Already enforced |
| Record is immutable after creation | Append-only blob with optional WORM policy (ImmutableStorage) | ✅ Structural fit (WORM policy to be enabled per A6 open question) |
| Record is append-only (cannot revise history) | JSONL append semantics; no update or delete operations on trace | ✅ Structural fit |
| Record carries timestamp of declaration | Every JSONL event carries `timestamp` (UTC, RFC3339) | ✅ Already present |
| Record survives process failure | Written to Blob Storage before worker proceeds; survives worker crash | ✅ Already enforced |
| Record is queryable for compliance review | Cosmos DB `runs` container carries run_id → trace blob pointer | ✅ Present |

**Verdict:** The JSONL trace is the right substrate for declaration registration. Every declaration event should be a first-class trace event (`declaration_collected`, not just `user_input_received`). The trace alone satisfies "registered" for simple audit use cases. It does not satisfy qualified electronic signature requirements without additional primitives.

---

### 1.4 Service Bus Sessions / At-Most-Once

| Scenario Step | Relevance |
|---|---|
| Declaration runbook triggered by a hospital scheduling system | Service Bus session ensures exactly-once execution — no duplicate declaration forms dispatched to the same patient |
| Insurance pre-declaration form submitted via customer portal | Session lock prevents same form being processed twice on worker restart during submission |
| Guardian consent triggered as part of surgical intake workflow | At-most-once ensures the consent collection step does not repeat for the same `run_id` |

**Verdict:** At-most-once delivery is **directly relevant** — a declaration collected twice could confuse legal systems (which signature is authoritative?). Service Bus sessions already prevent this at the queue boundary.

---

### 1.5 RE2 Redaction (Google.Re2)

| Field in a Declaration Form | Redaction Pattern Needed | Status |
|---|---|---|
| RUT (Chilean national ID, format `12.345.678-9`) | `\d{1,2}\.\d{3}\.\d{3}-[\dkK]` | ✅ RE2-expressible |
| SSN (US Social Security Number) | `\d{3}-\d{2}-\d{4}` | ✅ RE2-expressible |
| Medical record numbers (varies by institution) | Customer-defined pattern | ✅ RE2-expressible |
| Passport number (alphanumeric, varies) | Customer-defined per country | ✅ RE2-expressible |
| IBAN / bank account | `[A-Z]{2}\d{2}[A-Z0-9]+` | ✅ RE2-expressible |
| Full name in free-text field | ❌ Contextual PII — not RE2-expressible | ❌ Gap — no NLP/entity detection |

**Verdict:** RE2 redaction handles structured PII (IDs, numbers, account codes) well. Contextual PII in free-text fields (names embedded in narrative, addresses, diagnoses spelled out) cannot be reliably redacted with RE2. This is an existing limitation — not new to declaration scenarios, but more acute because declaration forms have high free-text PII density.

---

### 1.6 `client="web"` RunOptions Tagging

| Use Case | Value |
|---|---|
| Compliance officer filters audit trail by channel | `client="web"` lets them isolate portal-initiated declarations from CLI/API-initiated runs |
| Per-channel statistics (how many surgical consents collected via portal vs. batch?) | Possible with `client` tag |
| Multi-tenant audit: hospital A's runs vs. hospital B's | `tenantId` in RunOptions (already in design) |

**Verdict:** Already a fit. Declaration scenarios need no extension here beyond what is already designed.

---

## 2. What's Missing — Gaps That Block Real Customer Use

| # | Gap | Blocked Scenario | Severity |
|---|---|---|---|
| G-1 | **Signature capture primitive** | Any legally significant declaration | 🔴 Critical |
| G-2 | **Identity proofing / step-up auth at declaration moment** | Surgical consent, legal waivers | 🔴 Critical |
| G-3 | **Witness / co-signer flow (multi-actor single event)** | Witnessed consent, guardian signature | 🔴 Critical |
| G-4 | **Presented-document version hash** | Any declaration where form content may change | 🔴 Critical |
| G-5 | **Multi-language / translation provenance** | Consents in regulated bilingual jurisdictions | 🟠 High |
| G-6 | **Capacity / on-behalf-of flow (subject ≠ declarant)** | Minor consent, guardian/ward, power of attorney | 🟠 High |
| G-7 | **Time-bounded validity (expiry)** | 30-day pre-surgical consent, annual policy declaration | 🟠 High |
| G-8 | **Revocation (inverse event, no rewrite)** | Patient withdraws consent; policyholder rescinds declaration | 🟠 High |
| G-9 | **Contextual PII redaction in free-text** | Free-text declaration fields containing names, diagnoses | 🟡 Medium |
| G-10 | **Qualified electronic signature integration (eIDAS / ESIGN / Ley 19.799)** | Legal-weight e-signature requirement | 🟡 Medium (integration point, not core) |

---

### G-1: Signature Capture Primitive

**Problem:** `UserInputKind.Confirmation` captures a click, not a signature. Under Chilean Ley 19.799, the US ESIGN Act, and EU eIDAS Regulation, a "simple electronic signature" requires at minimum: identity binding + intent to sign + timestamp. A click-through confirmation satisfies none of these legally.

**Options — not yet a decision:**

| Option | Description | Complexity | Legal Weight |
|---|---|---|---|
| Typed-name attestation | User types their full name to acknowledge intent | Low | Minimal (ESIGN "simple") |
| Drawn signature (SVG/canvas blob) | User draws signature, stored as FileUpload artifact | Medium | Weak (no identity binding) |
| Cryptographic token-bound signature | Platform signs a hash of `{run_id, step_id, document_hash, timestamp}` with the user's Entra External ID credential | High | Strong (ESIGN compliant; eIDAS SES level) |
| Qualified Trust Service Provider (QTSP) integration | External TSP issues QES; GERT records the TSP's response token in trace | High | eIDAS QES / Ley 19.799 Advanced ES |

**Runtime implication:** A new `UserInputKind.Signature` is needed, OR `FileUpload` is extended with `SignatureMetadata` (typed name, canvas blob URI, TSP token). The trace event must record:
- `declared_by` (user identity from Entra token)
- `signature_method` (typed / drawn / cryptographic / QTSP)
- `document_hash` (SHA-256 of the exact document shown)
- `signature_artifact_uri` (Blob URI if drawn/QTSP)

---

### G-2: Identity Proofing / Step-Up Authentication

**Problem:** Entra External ID issues a JWT on login. By the time a consent step executes, the JWT may be 30+ minutes old. A "strong" declaration requires that the user's identity was **freshly verified at the moment of declaration**. Especially relevant for:
- High-risk surgical consent (re-prompt MFA)
- Government-regulated declarations (step-up to government ID)
- Financial declarations (re-prompt password + MFA)

**Current state:** GERT trusts the existing session JWT. No step-up mechanism exists.

**Runtime implication:** `IUserInputGate.AwaitUserInputAsync()` needs an `IdentityProofingLevel` parameter (None / SessionJWT / FreshMFA / GovernmentID). The gate implementation enforces it — e.g., for `FreshMFA`, the portal must complete an Entra Conditional Access challenge before the `SubmitResponseAsync()` call is accepted. The trace event records `identity_proofing_level` and `identity_proof_token` (opaque Entra claim or OIDC token fragment).

---

### G-3: Witness / Co-Signer Flow

**Problem:** A surgical consent often requires a witness (usually a nurse or second clinician) to attest they observed the patient signing. A guardian consent requires the guardian to sign **alongside** the child (two different Entra identities). 

`IApprovalGate` models a downstream authorizer — semantics are approve/reject. A witness is not approving or rejecting; they are **attesting presence and observing**. A co-signer is adding their own declaration alongside the primary.

**Current state:** No multi-actor primitive on a single trace event. `IApprovalGate` can be sequenced after `IUserInputGate`, which creates two separate trace events — but they are not cryptographically linked as a compound declaration.

**Runtime implication:** A new `IWitnessGate` (or an extension to `IApprovalGate`) is needed. Sketch:

```csharp
// Two actors, same logical declaration event
public interface IWitnessGate
{
    Task<WitnessAttestation> AwaitWitnessAsync(
        string runId,
        string declarationStepId,  // references the primary declaration
        WitnessRequest request,
        CancellationToken ct = default);
}

public sealed record WitnessRequest(
    string WitnessRole,          // "physician", "notary", "guardian"
    string? RequiredWitnessId,   // null = any valid identity; non-null = specific person
    TimeSpan Timeout);

public sealed record WitnessAttestation(
    string WitnessIdentity,      // Entra user id
    string WitnessDisplayName,
    DateTimeOffset AttestedAt,
    string? SignatureArtifactUri,
    string RelatedDeclarationStepId);
```

The JSONL trace must carry both the primary `declaration_collected` event and the `witness_attested` event with a `related_step_id` reference, so they are queryable together.

---

### G-4: Presented-Document Version Hash

**Problem:** Declaration forms change. A consent signed in January may reference v1.2 of the hospital's consent form. The trace must capture **the exact bytes of the document shown to the declarant**. Without this, it is impossible to prove in litigation what the declarant actually read and agreed to.

**Current state:** The trace records that a `user_input_received` event occurred. It does not capture which version of the form template was rendered or a hash of it.

**Runtime implication:** A new trace event field `document_hash` (SHA-256 hex, string) and `document_version` (customer-provided string) must be mandatory on declaration events. The customer must supply the pre-rendered content hash via a `DeclarationContext` parameter on the runbook step — GERT does not render the document but MUST record what the customer claims was shown.

---

### G-5: Multi-Language / Translation Provenance

**Problem:** A hospital in Chile serving Spanish-speaking patients must show consents in Spanish. If the same runbook has English and Spanish variants, the trace must record which language the declarant actually saw — not just which language the runbook was authored in.

**Current state:** No language/locale field on `UserInputRequest` or on trace events.

**Runtime implication:** Add `Locale` (BCP-47 string, e.g., `"es-CL"`) to `UserInputRequest` and propagate it to the `user_input_requested` / `declaration_collected` trace events. The `document_hash` (G-4) inherently captures this — a Spanish document and an English document will have different hashes — but the locale must also be explicit for query purposes.

---

### G-6: Capacity / On-Behalf-Of Flow (Subject ≠ Declarant)

**Problem:** When a parent signs a consent for a minor child, or a legal guardian signs for a ward, the declaration has two distinct identities:
- **Subject:** the person the declaration concerns (the child, the ward)
- **Declarant:** the person signing (the parent, the guardian)

These are different Entra identities (or one may be an identity-less minor). The relationship must be declared and traced.

**Current state:** GERT traces only the authenticated session user. There is no concept of subject vs. declarant, relationship type, or evidence of authority (e.g., power-of-attorney document).

**Runtime implication:** A `DeclarationPrincipal` construct:

```csharp
public sealed record DeclarationPrincipal(
    string DeclarantIdentity,          // Entra user id of the person signing
    string? SubjectIdentity,           // null = self; non-null = on-behalf-of
    string? SubjectDisplayName,        // e.g., "María González (minor)"
    OnBehalfOfRelationship? Relationship,  // Parent, Guardian, AttorneyInFact, etc.
    string? AuthorityEvidenceStepId);  // step_id where authority document was uploaded (FileUpload)

public enum OnBehalfOfRelationship { Self, Parent, Guardian, AttorneyInFact, CorporateAuthorized, Other }
```

This would be a field on `UserInputRequest` (and on declaration-specific step types), not a top-level runtime primitive. The trace event must include it.

---

### G-7: Time-Bounded Validity (Expiry)

**Problem:** A pre-surgical consent is often valid only for a defined period (e.g., 30 days). Once expired, the downstream surgical scheduling process must reject it and re-collect consent. The runbook engine needs to express validity windows.

**Current state:** GERT traces have timestamps but no notion of expiry. There is no primitive to say "this trace event is valid until T" nor any runtime check that prevents a downstream step from consuming an expired declaration.

**Runtime implication:** Two things are needed:

1. **Trace event field:** `valid_until` (optional, ISO8601 UTC). Written into the `declaration_collected` event. Immutable after write.
2. **Runtime check primitive:** A `CheckDeclarationValidityAsync(string runId, string declarationStepId)` method — or a runbook-level step type — that reads the trace event, checks `valid_until` against `DateTimeOffset.UtcNow`, and blocks execution if expired. This is NOT a gate (it does not wait) — it is a guard that throws `DeclarationExpiredException` if the declaration has lapsed.

The expiry check must itself be traced (`declaration_validity_checked`, outcome: valid | expired).

---

### G-8: Revocation (Inverse Event, No History Rewrite)

**Problem:** A patient may withdraw consent before the procedure. An insurance policyholder may rescind a pre-declaration. Under the immutability invariant, the original declaration event cannot be deleted or modified. But the runtime must model the withdrawal in a way that downstream processes (and compliance officers) can determine the effective state of a declaration.

**Current state:** No revocation primitive. The append-only trace has no concept of logical reversal of a prior event.

**Runtime implication:** A `declaration_revoked` trace event type that:
- References the original `declaration_collected` event by `step_id` (not by rewriting it)
- Carries `revoked_by` (identity), `revoked_at` (timestamp), `revocation_reason` (text)
- Is itself immutable after write

The runtime must provide a query helper `GetEffectiveDeclarationStateAsync(string runId, string declarationStepId)` that returns `Active | Revoked | Expired` by reading the trace (not Cosmos DB) — so the audit trail remains the single source of truth.

There is also a **process-level implication**: if a downstream process has already started consuming the declaration (e.g., surgical scheduling began), the revocation event must trigger a webhook `declaration.revoked` so dependent systems can halt. This is David's domain (webhook delivery) but the trace event definition is Don's.

---

### G-9: Contextual PII Redaction in Free-Text

**Problem:** RE2 handles structured PII patterns well (RUT, SSN, IBAN). In declaration forms, free-text fields often contain names, medical diagnoses spelled out, addresses, and other contextual PII that cannot be matched with a regex.

**Current state:** Google.Re2 is the only redaction engine. No NLP or entity detection layer exists.

**Runtime implication:** This is a hard gap with no simple fix — adding NLP entity detection to the runtime is a large surface area. **The correct short-term answer is a governance policy**: declare certain step types (like declaration forms) as PII-bearing, and prohibit their trace events from being included in any external export, log stream, or webhook payload without customer-controlled masking. The redaction engine stays RE2; the policy layer gets a `PiiZone` flag on step types.

---

### G-10: Qualified Electronic Signature Integration

**Problem:** For advanced use cases (notarized documents, regulated financial declarations, Chilean Firma Electrónica Avanzada under Ley 19.799), a Qualified Trust Service Provider (QTSP) must issue the signature. GERT is not a QTSP and must not try to be one.

**Current state:** No integration point for QTSP responses. No trace field for TSP-issued signature tokens.

**Runtime implication:** GERT's role is to:
1. Redirect the user to the QTSP flow (as an external step or as a specialized `UserInputKind.QualifiedSignature` that carries a TSP redirect URI)
2. Receive the TSP callback (signed token, certificate reference, timestamp authority token)
3. Write the TSP callback data immutably into the trace as a `qualified_signature_received` event
4. Gate the downstream process on existence of this event

GERT must NOT: issue certificates, manage private keys, interact with PKI directly, or store biometrics (see Non-Goals).

---

## 3. New Interface Sketches (Runtime-Level Decisions Raised)

### 3.1 New Trace Event Types

```csharp
// Replaces or supplements "user_input_received" for legally significant inputs
public sealed record DeclarationCollectedEvent(
    string RunId,
    string StepId,
    string EventType,              // "declaration_collected"
    DateTimeOffset Timestamp,
    int SequenceNumber,
    string DeclarantIdentity,      // Entra user id
    string DeclarationKind,        // "consent" | "exemption" | "pre_declaration" | "acknowledgment"
    string DocumentHash,           // SHA-256 of rendered document bytes (customer-supplied)
    string DocumentVersion,        // customer-supplied version string
    string Locale,                 // BCP-47 (e.g., "es-CL")
    string SignatureMethod,        // "typed_name" | "drawn" | "cryptographic" | "qtsp"
    string? SignatureArtifactUri,  // Blob URI if applicable
    string? TypedName,             // if SignatureMethod = "typed_name"
    string IdentityProofingLevel,  // "session_jwt" | "fresh_mfa" | "government_id"
    string? IdentityProofToken,    // opaque Entra/OIDC claim fragment
    DateTimeOffset? ValidUntil,    // optional expiry
    DeclarationPrincipal Principal) // subject / declarant / on-behalf-of

public sealed record WitnessAttestedEvent(
    string RunId,
    string StepId,
    string EventType,              // "witness_attested"
    DateTimeOffset Timestamp,
    int SequenceNumber,
    string RelatedDeclarationStepId,
    string WitnessIdentity,
    string WitnessRole,
    string? SignatureArtifactUri)

public sealed record DeclarationRevokedEvent(
    string RunId,
    string StepId,                 // new step id for this revocation
    string EventType,              // "declaration_revoked"
    DateTimeOffset Timestamp,
    int SequenceNumber,
    string OriginalDeclarationStepId, // immutable reference back
    string RevokedByIdentity,
    string RevocationReason)

public sealed record DeclarationValidityCheckedEvent(
    string RunId,
    string StepId,
    string EventType,              // "declaration_validity_checked"
    DateTimeOffset Timestamp,
    int SequenceNumber,
    string CheckedDeclarationStepId,
    string Outcome,                // "valid" | "expired"
    DateTimeOffset? ValidUntil)

public sealed record QualifiedSignatureReceivedEvent(
    string RunId,
    string StepId,
    string EventType,              // "qualified_signature_received"
    DateTimeOffset Timestamp,
    int SequenceNumber,
    string TspProvider,            // e.g., "eSign", "CertSuperior", "DocuSign"
    string TspSignatureToken,      // opaque TSP-issued token (not parsed by GERT)
    string TspCertificateReference,
    string DocumentHash,
    DateTimeOffset SignedAt)
```

---

### 3.2 Extended UserInputKind

```csharp
public enum UserInputKind
{
    Choice,
    Text,
    Confirmation,
    FileUpload,
    Form,
    Signature,          // NEW — typed name, drawn, or cryptographic binding
    QualifiedSignature  // NEW — TSP redirect + callback; integrates with external QTSP
}
```

---

### 3.3 New IWitnessGate Interface

```csharp
/// <summary>
/// Suspends a run until a witness (distinct from the primary declarant) attests
/// the declaration event. Used for witnessed consents and co-signer flows.
/// Fires inside IRunHandle.NextAsync() — same governance boundary as IApprovalGate.
/// </summary>
public interface IWitnessGate
{
    Task<WitnessAttestation> AwaitWitnessAsync(
        string runId,
        string declarationStepId,
        WitnessRequest request,
        CancellationToken ct = default);

    Task SubmitAttestationAsync(
        string runId,
        string declarationStepId,
        WitnessAttestation attestation);

    Task<PendingWitness?> GetPendingWitnessAsync(string runId);
}

public sealed record WitnessRequest(
    string WitnessRole,
    string? RequiredWitnessIdentity,  // null = any valid auth'd identity
    TimeSpan Timeout,
    bool RequireSignatureArtifact);

public sealed record WitnessAttestation(
    string WitnessIdentity,
    string WitnessDisplayName,
    DateTimeOffset AttestedAt,
    string? SignatureArtifactUri,
    string RelatedDeclarationStepId);

public sealed record PendingWitness(
    string RunId,
    string DeclarationStepId,
    WitnessRequest Request,
    DateTimeOffset RequestedAt,
    string Status);  // "pending" | "attested" | "timed_out"
```

---

### 3.4 DeclarationContext on UserInputRequest

```csharp
// Extension to the existing UserInputRequest record
// Populated by the runbook definition for declaration-type steps
public sealed record DeclarationContext(
    string DeclarationKind,             // "consent" | "exemption" | "pre_declaration" | "acknowledgment"
    string DocumentHash,                // SHA-256 hex — customer must supply; GERT records, does not compute
    string DocumentVersion,
    string Locale,                      // BCP-47
    DateTimeOffset? ValidUntil,
    DeclarationPrincipal Principal,
    IdentityProofingLevel RequiredProofingLevel,
    bool RequireWitness,
    string? WitnessRole);

public sealed record DeclarationPrincipal(
    string DeclarantIdentity,
    string? SubjectIdentity,
    string? SubjectDisplayName,
    OnBehalfOfRelationship Relationship,
    string? AuthorityEvidenceStepId);

public enum OnBehalfOfRelationship { Self, Parent, Guardian, AttorneyInFact, CorporateAuthorized, Other }
public enum IdentityProofingLevel { SessionJwt, FreshMfa, GovernmentId }
```

---

### 3.5 Runtime Query Helper

```csharp
// On IRunHandle or a dedicated IDeclarationRegistry:
public interface IDeclarationRegistry
{
    Task<DeclarationState> GetEffectiveStateAsync(
        string runId,
        string declarationStepId,
        CancellationToken ct = default);
}

public sealed record DeclarationState(
    string RunId,
    string DeclarationStepId,
    DeclarationStatus Status,          // Active | Expired | Revoked
    DateTimeOffset CollectedAt,
    DateTimeOffset? ValidUntil,
    DateTimeOffset? RevokedAt,
    string? RevocationReason);

public enum DeclarationStatus { Active, Expired, Revoked }
```

---

## 4. Explicit Non-Goals — What GERT Must NOT Do Here

| Non-Goal | Rationale |
|---|---|
| **Be a Trust Service Provider (TSP) or issue qualified certificates** | We integrate with QTSPs (DocuSign, CertSuperior, eSign, etc.). GERT records their token. We are NOT a PKI. |
| **Store biometrics (fingerprints, facial recognition data)** | No biometric data enters the trace or any GERT store. If a QTSP uses biometrics for proofing, that is internal to the TSP — GERT receives only a signed token. |
| **Render legal text or declaration form content** | The customer provides the form (HTML, PDF). GERT records the hash of what the customer claims was shown. We do not render, host, or validate the form content. |
| **Perform legal interpretation** | Whether a collected declaration constitutes valid consent under applicable law is the customer's and their legal counsel's concern. GERT records facts; it does not adjudicate. |
| **Validate medical capacity of the declarant** | Whether a patient has capacity to consent is a clinical determination. GERT can gate on `IdentityProofingLevel` but cannot assess cognitive or legal capacity. |
| **Manage consent revocation workflows beyond the trace event** | GERT writes `declaration_revoked`. Downstream processes (EHR, scheduling systems) consuming that event are the customer's responsibility. GERT fires the webhook; it does not orchestrate the downstream response. |
| **Implement per-jurisdiction compliance rules** | Whether Chilean Ley 19.799, US ESIGN, EU eIDAS, or local health regulations are satisfied is outside GERT's scope. GERT provides the primitives; the customer implements the compliant runbook. |
| **Be the sole system of record for declarations** | GERT's trace is a strong audit trail. Regulated industries (healthcare, insurance) will need to export trace events to their own records management systems. GERT is a producer, not the final archive. |

---

## 5. Gap Priority Summary

| Gap | New Primitive | Priority | Blocks Production? |
|---|---|---|---|
| G-1: Signature capture | `UserInputKind.Signature`, `DeclarationCollectedEvent` | P0 | ✅ Yes |
| G-2: Identity proofing | `IdentityProofingLevel` on `UserInputRequest` | P0 | ✅ Yes |
| G-3: Witness/co-signer | `IWitnessGate`, `WitnessAttestedEvent` | P0 | ✅ Yes |
| G-4: Presented-document hash | `document_hash` field on `DeclarationCollectedEvent` | P0 | ✅ Yes |
| G-5: Multi-language provenance | `Locale` field on `DeclarationCollectedEvent` | P1 | For regulated bilingual jurisdictions |
| G-6: On-behalf-of / capacity | `DeclarationPrincipal` | P1 | For minor/guardian scenarios |
| G-7: Time-bounded validity | `valid_until` field + `CheckDeclarationValidityAsync` | P1 | For time-sensitive consents |
| G-8: Revocation | `DeclarationRevokedEvent` + `IDeclarationRegistry` | P1 | For withdrawal scenarios |
| G-9: Contextual PII redaction | Policy flag (`PiiZone`) — no new engine | P2 | No (governance workaround exists) |
| G-10: QTSP integration | `UserInputKind.QualifiedSignature`, `QualifiedSignatureReceivedEvent` | P2 | Only for advanced e-signature tier |

---

*Next step: review with Barbara for architecture ratification; review with Germán for scenario validation against Chilean legal requirements (Ley 19.799 alignment, RUT handling, healthcare regulation).*

---

# Declaration & Consent Integration Patterns — Analysis & Event Catalog

**Author:** David (Integration Engineer)  
**Date:** 2026-06-03T23:53:24-04:00  
**Status:** Analysis — Ready for architectural review  
**Audience:** Team; especially backend (Don), platform (John), product (Germán)

---

## Executive Summary

Declaration/consent flows are legal record producers. Unlike transient workflow events, a "declaration_completed" webhook represents a durable, auditable commitment that must survive to downstream systems without loss. This analysis maps the integration surface: downstream receivers, evidence requirements, delivery guarantees, and the new webhook event types this scenario family demands.

**Key findings:**
1. **Five upstream receiver categories** span healthcare, insurance, government, finance/KYC, and employment — each with different SLA and evidence expectations.
2. **Evidence package** must bundle signed document URL, trace excerpt (decision path), hash chain (immutability proof), and audit receipt.
3. **Delivery must guarantee at-least-once + immutability**, not just retry; declarations cannot be silently lost or altered mid-flight.
4. **Nine new webhook event types** beyond v1.0 baseline to cover declaration lifecycle: submitted, verified, signed, completed, declined, abandoned, expired_unwitnessed, revoked, archived.
5. **Post-completion events** ("sealed", "revoked") mean declarations are never truly "final" — they have ongoing lifecycle.

---

## 1. WHO ARE THE RECEIVERS? — Downstream Systems by Industry

### Healthcare

| Receiver System | Event Trigger | Expectations | SLA |
|---|---|---|---|
| Hospital EHR | declaration_completed (surgical consent) | Signed PDF + date/time, declarant identity, witness identity; immutable archive | 5 min acceptance; 7-year retention |
| Surgical Scheduling | declaration_completed | ORD code, surgical site marking, consent scope; cannot retry on schedule already booked | <1 min; no-revoke-post-surgery |
| Insurance Pre-Auth | declaration_completed | Declarant insurance ID, pre-auth code, coverage scope; immutable record | 15 min; 5-year legal hold |
| Medical Record Archive | declaration_completed | Full audit trail (declarant consent journey), digital signature; tamper-evident seal | <30 min batch; 10-year retention |

**Summary:** Healthcare receivers demand immutability, signed PDFs, legal holds, and witness verification. Post-completion modification is forbidden.

### Insurance

| Receiver System | Event Trigger | Expectations | SLA |
|---|---|---|---|
| Policy Admin System | declaration_completed (underwriting approval) | Declarant name, DOB, answers to underwriting questions, timestamp | <5 min; policy cannot bind without it |
| Underwriting | declaration_completed (medical questionnaire) | Full form responses, risk assessment answers; audit trail of changes | 1h batch; must block coverage until received |
| Billing | declaration_completed (payment terms consent) | Declarant signed payment authorization, frequency, amount; recurring charge linkage | <10 min; initiates recurring charge |
| Agent Portal | declaration_completed | Visible to agent as "consent received" status; commission calc depends on it | <1 min display; agent SLA tracked |

**Summary:** Insurance receivers need evidence of intent (signed), strict ordering (can't charge until consent), and settlement finality.

### Government

| Receiver System | Event Trigger | Expectations | SLA |
|---|---|---|---|
| Registry of Acts | declaration_completed (deed, license, permit application) | Declarant legal name, identity proof reference, declaration text, notary signature if required; immutable record | <24h; registry must accept within 30 days or declare void |
| Archival System | declaration_completed | Full audit trail (every keystroke, every revision, final signature), tamper chain; must be reproducible | <48h; 30-year legal hold for some documents |
| Citizen Portal Notification | declaration_completed | Notification to citizen: "Your declaration of X was registered on [date]"; clickable link to sealed receipt | <5 min; immediate user feedback |

**Summary:** Government receivers are strictest: registries have capacity limits, archival has replay requirements, citizen portals need instant notification.

### Finance / KYC

| Receiver System | Event Trigger | Expectations | SLA |
|---|---|---|---|
| AML System | declaration_completed (KYC questionnaire, beneficial owner disclosure) | Declarant answers, timestamp, identity verification status, PEP/sanctions check integration | <5 min; blocks account opening until received |
| Account Opening Pipeline | declaration_completed | Declarant PII consent, terms acceptance, AML clearance status; triggers next step in onboarding | <10 min; no silent drops (regulatory risk) |
| Compliance Archive | declaration_completed | Immutable record of every KYC declaration version; hash chain proving no tampering | <1h batch; 10-year regulatory hold |

**Summary:** Finance is compliance-first: silent drops = fines. Immutability is non-negotiable. Every version must be traceable.

### Employment

| Receiver System | Event Trigger | Expectations | SLA |
|---|---|---|---|
| HRIS | declaration_completed (beneficiary designation, tax elections) | Employee name, declaration text, timestamp, signature; non-repudiation | <1h batch; cannot process payroll without tax election consent |
| Payroll | declaration_completed (withholding, garnishment consent) | Declarant consent to deductions, frequency, amount; immutable record | <24h; next payroll cycle must have it |
| Benefits Enrollment | declaration_completed (plan election) | Employee plan choices, effective date, dependent consent if needed | <4h; open enrollment windows are tight |

**Summary:** Employment receivers have tight batch windows and strict payroll deadlines. Missing one declaration can cascade to wrong withholding.

---

## 2. EVENT PAYLOADS — Declaration Completed Webhook Schema

### Core Event Structure (All Declaration Events)

```json
{
  "event_type": "declaration.completed",
  "event_id": "evt_1a2b3c4d5e",
  "timestamp": "2026-06-03T23:53:24Z",
  "run_id": "run_12345",
  "tenant_id": "tenant_9876",
  "declaration_id": "decl_abcdefg",
  "idempotency_key": "decl_abcdefg_completed_v1",
  "signature": "sha256=...",
  "payload": { ... }
}
```

**Idempotency:** Receiver MUST deduplicate on `idempotency_key` (not just `event_id`). A retry of "declaration.completed" for the same `declaration_id` must be safe.

### Summary Event (In Webhook)

```json
{
  "payload": {
    "declaration_type": "surgical_consent",
    "declarant": {
      "name": "Jane Doe",
      "identity_verified": true,
      "identity_verification_timestamp": "2026-06-03T23:51:00Z"
    },
    "declaration_status": "completed",
    "declaration_content_hash": "sha256:abc123...",
    "signed_document_url": "https://gert.example.com/api/v1/declarations/decl_abcdefg/document?token=sas_xyz",
    "signed_document_url_expires_at": "2026-06-10T23:53:24Z",
    "signature_timestamp": "2026-06-03T23:52:30Z",
    "witness": {
      "name": "Dr. Smith",
      "role": "surgeon",
      "witnessed_at": "2026-06-03T23:52:00Z"
    },
    "evidence_package_url": "https://gert.example.com/api/v1/declarations/decl_abcdefg/evidence",
    "trace_excerpt_url": "https://gert.example.com/api/v1/declarations/decl_abcdefg/trace-excerpt",
    "metadata": {
      "run_version": "1.2.3",
      "form_fields_count": 12,
      "approval_steps_count": 2,
      "total_completion_duration_seconds": 420
    }
  }
}
```

**Design rationale:**
- **Signed document URL** is time-limited SAS (not full document embedded) to avoid massive payloads.
- **Hash** allows receiver to verify immutability later (after fetching full document).
- **Witness fields** required for medical/notarial declarations.
- **Metadata** gives receiver enough context to route without fetching full evidence.
- **Trace excerpt URL** points to JSONL subset (decision path only, not full execution trace).

### Evidence Package Endpoint Response

When receiver calls the `evidence_package_url`, they receive:

```json
{
  "declaration_id": "decl_abcdefg",
  "status": "completed",
  "created_at": "2026-06-03T23:50:00Z",
  "completed_at": "2026-06-03T23:52:30Z",
  "evidence": {
    "signed_document": {
      "url": "https://gert.example.com/...",
      "expires_at": "2026-06-10T23:53:24Z",
      "media_type": "application/pdf",
      "size_bytes": 245678,
      "hash": "sha256:abc123..."
    },
    "trace_excerpt": {
      "url": "https://gert.example.com/api/v1/declarations/decl_abcdefg/trace-excerpt",
      "format": "application/x-ndjson",
      "line_count": 18,
      "hash": "sha256:def456..."
    },
    "hash_chain": {
      "format": "application/json",
      "merkle_root": "sha256:root789...",
      "inclusion_proof": [
        { "step": 1, "hash": "sha256:h1..." },
        { "step": 5, "hash": "sha256:h5..." },
        { "step": 12, "hash": "sha256:h12..." }
      ],
      "archive_timestamp": "2026-06-03T23:52:31Z"
    },
    "audit_receipt": {
      "format": "application/pdf",
      "url": "https://gert.example.com/...",
      "qr_code_data": "https://gert.example.com/receipts/rec_xyz123",
      "receipt_number": "REC-2026-0001234567"
    }
  }
}
```

**Evidence package design:**
- **Merkle hash chain** proves the declaration is part of the immutable archive (no post-hoc edits).
- **Audit receipt** is a PDF with human-readable receipt number + QR code for non-technical users.
- **Trace excerpt** is JSONL (not full trace) with only decision steps + signature events.

---

## 3. EVIDENCE PACKAGE — What Receivers Can Pull Back

When receiver gets "declaration.completed", they have access to:

### 3.1 Signed Document (PDF)

- **What:** The filled declaration form, signed by declarant + witnesses, rendered as immutable PDF.
- **Access:** Time-limited SAS URL (valid for 7 days default).
- **Hash:** SHA256 checksum to verify no tampering.
- **Retention:** Stored in Blob with immutable policy (WORM) after N days. Receiver should download immediately if long-term compliance hold required.

### 3.2 JSONL Trace Excerpt

- **What:** Subset of execution trace: only form input steps + signature event.
- **Format:** NDJSON (one event per line).
- **Example:**
  ```json
  {"seq": 1, "timestamp": "2026-06-03T23:50:15Z", "type": "step_started", "step_id": "form_input_declarant", "label": "Surgical Site & Procedure"}
  {"seq": 2, "timestamp": "2026-06-03T23:51:00Z", "type": "user_input_received", "step_id": "form_input_declarant", "field_keys": ["surgical_site", "procedure_code"], "user_id": "user_123"}
  {"seq": 5, "timestamp": "2026-06-03T23:52:00Z", "type": "step_started", "step_id": "witness_signature", "label": "Witness Acknowledgment"}
  {"seq": 6, "timestamp": "2026-06-03T23:52:20Z", "type": "approval_approved", "approver": "witness_456", "approver_role": "surgeon"}
  {"seq": 8, "timestamp": "2026-06-03T23:52:30Z", "type": "declaration_signed", "declarant_signature": "sig_abc", "witness_signature": "sig_def"}
  ```
- **Why not full trace?** Reduces payload size; receiver has declaration_id to fetch full trace later if audited.
- **Immutability:** Trace is append-only in Blob Storage; receiver can pin specific JSONL hash at download time.

### 3.3 Hash Chain / Merkle Inclusion Proof

- **What:** Cryptographic proof that this declaration is part of the sealed archive.
- **Structure:** Merkle tree with leaf = declaration, branches = other concurrent declarations, root = daily archive seal.
- **Receiver's use:** Independently verify "this declaration was not modified post-hoc by the system operator".
- **Implementation:** Daily batch job creates Merkle tree, publishes root + inclusion proof to receiver on demand.

### 3.4 Audit Receipt (PDF + QR)

- **What:** Human-readable receipt showing:
  - Receipt number (REC-YYYY-SEQ)
  - Declaration ID
  - Declarant name (PII, so printed copy only, not API)
  - Timestamp (when completed)
  - Witness (if applicable)
  - QR code linking to sealed receipt on portal
- **Use:** Declarant can print + file; shows proof of declaration to third parties.
- **Expiry:** Receipt URL valid for 30 days; QR code is permanent link to sealed receipt.

---

## 4. DELIVERY GUARANTEES — Legal Record SLAs

Declarations are not transient events. Receivers need guarantees beyond "try 5 times then give up".

### 4.1 At-Least-Once With Idempotency

**Guarantee:** Receiver will receive "declaration.completed" at least once within 24h. Retries use exponential backoff (1m, 5m, 30m, 2h, 6h).

**Receiver's obligation:** Deduplicate on `idempotency_key`, not just `event_id`. If receiver processes idempotent key twice, it must be safe (e.g., idempotent update in database, or no-op if already exists).

**Example:** AML system receives declaration.completed for KYC_5555, processes it (creates compliance record). Retry arrives (network flake). AML sees same idempotency_key, skips reprocessing, returns 200 OK.

### 4.2 Replay Endpoint for Reconstruction

**Guarantee:** If receiver misses "declaration.completed" or crashes mid-processing, they can request a full replay.

**Endpoint:** `POST /api/v1/declarations/{declaration_id}/replay?since=2026-06-03T00:00:00Z`

**Response:** Replays all events for this declaration since `since` timestamp:
```json
{
  "events": [
    { "event_type": "declaration.submitted", "timestamp": "2026-06-03T23:50:00Z", ... },
    { "event_type": "declaration.verified", "timestamp": "2026-06-03T23:51:00Z", ... },
    { "event_type": "declaration.signed", "timestamp": "2026-06-03T23:52:30Z", ... },
    { "event_type": "declaration.completed", "timestamp": "2026-06-03T23:52:31Z", ... }
  ]
}
```

**Use case:** Receiver's webhook endpoint was down for 2 hours. They call replay endpoint, catches up, processes events in order.

### 4.3 Sealed Archive Notification

**Guarantee:** After N days (configurable, default 30), a "declaration.sealed" event is published. This means:
- No further changes to the declaration are possible (immutable policy enforced on Blob).
- Receiver can rely on permanent access (no expiration of SAS URLs).
- Hash chain is finalized (included in permanent Merkle root).

**Event:**
```json
{
  "event_type": "declaration.sealed",
  "declaration_id": "decl_abcdefg",
  "sealed_at": "2026-07-03T23:52:30Z",
  "merkle_root_hash": "sha256:root789...",
  "is_sealed_permanently": true
}
```

**Receiver's use:** After seal, download signed PDF and evidence package to long-term archive (no SAS expiration worry).

### 4.4 Long-Term Retrieval SLA

| Industry | Retention | Retrieval Endpoint |
|---|---|---|
| Healthcare | 7 years minimum (varies by jurisdiction) | Permanent (no expiration after seal) |
| Insurance | 5–10 years (varies by product) | Permanent after seal |
| Government | 10–30 years (varies by registry) | Permanent after seal |
| Finance/KYC | 10 years (AML/regulatory hold) | Permanent after seal |
| Employment | 7 years (tax, labor law) | Permanent after seal |

**Implication:** Sealed declarations cannot expire. Receivers must have permanent access to retrieve 7+ years later.

---

## 5. EDGE CASES — Declarations That Don't Complete

Not all declarations flow smoothly to completion. Each edge case requires a distinct event type that receivers must handle.

### 5.1 Declarant Abandons Mid-Form

**Scenario:** User enters surgical consent form, fills 3 of 8 fields, closes browser tab.

**Event:** `declaration.abandoned`

```json
{
  "event_type": "declaration.abandoned",
  "declaration_id": "decl_abcdefg",
  "abandoned_at": "2026-06-03T23:50:45Z",
  "abandonment_reason": "user_inactivity",
  "inactivity_duration_seconds": 1800,
  "last_completed_step": 3,
  "total_steps": 8,
  "fields_completed": ["surgical_site", "procedure_code", "medical_history_allergies"],
  "idempotency_key": "decl_abcdefg_abandoned_v1"
}
```

**Receiver's obligation:** 
- For healthcare: if EHR was tentatively blocked waiting for consent, unblock now or set timeout.
- For insurance: if policy binding was halted, mark as "consent incomplete, retry" or expire the underwriting request.
- Silent drop: NO. Receiver must audit-log this so compliance team knows consent was never obtained.

**Retention:** Abandoned declarations are retained in archive indefinitely (may be needed for audit trail "why was surgery not scheduled?").

### 5.2 Declarant Declines / Actively Rejects

**Scenario:** User reads surgical consent form and clicks "Decline" button (e.g., "I do not consent to general anesthesia, request local only").

**Event:** `declaration.declined`

```json
{
  "event_type": "declaration.declined",
  "declaration_id": "decl_abcdefg",
  "declined_at": "2026-06-03T23:51:30Z",
  "decline_reason": "user_explicit_rejection",
  "declarant_feedback": "Prefer local anesthesia per my prior experience",
  "steps_completed": 5,
  "total_steps": 8,
  "idempotency_key": "decl_abcdefg_declined_v1"
}
```

**Receiver's obligation:**
- For healthcare: surgical schedule must be cancelled or modified; patient communication required.
- For insurance: underwriting request fails; applicant must reapply with different choices.
- For employment: benefits election fails; employee must re-elect.
- **Non-negotiable:** Receiver must take action. Silent drop = compliance violation.

### 5.3 Witness Fails to Co-Sign Within Window

**Scenario:** Surgical consent requires witness signature. Declarant signed, witness received notification, but didn't sign within 24h window.

**Event:** `declaration.expired_unwitnessed`

```json
{
  "event_type": "declaration.expired_unwitnessed",
  "declaration_id": "decl_abcdefg",
  "expired_at": "2026-06-04T23:51:30Z",
  "witness_window_seconds": 86400,
  "declarant_signed_at": "2026-06-03T23:52:00Z",
  "witness_name": "Dr. Smith",
  "witness_status": "unsigned",
  "idempotency_key": "decl_abcdefg_expired_unwitnessed_v1"
}
```

**Receiver's obligation:**
- For healthcare: surgical appointment must be rescheduled; new consent required.
- For notarial declarations: registry cannot accept unsigned declaration.
- Trigger retry workflow: send witness reminder, extend window, or escalate to supervisor.

### 5.4 Post-Hoc Revocation (Consent Withdrawn After Process Started)

**Scenario:** Patient signed surgical consent, was placed on pre-op protocol, then calls to withdraw consent hours before surgery.

**Event:** `declaration.revoked`

```json
{
  "event_type": "declaration.revoked",
  "declaration_id": "decl_abcdefg",
  "revoked_at": "2026-06-04T08:30:00Z",
  "originally_completed_at": "2026-06-03T23:52:30Z",
  "time_to_revocation_seconds": 32400,
  "revocation_reason": "patient_requested",
  "revocation_authority": "patient_self_service",
  "cascade_impact": "requires_cancellation_of_scheduled_surgery",
  "idempotency_key": "decl_abcdefg_revoked_v1"
}
```

**Receiver's obligation — CRITICAL:**
- For healthcare: surgery MUST be cancelled or rescheduled; anesthesia team must be notified.
- For insurance: if claim processing already started, must be halted.
- For financial: if charge already authorized, must be reversed (or flagged for chargeback dispute resolution).
- **Non-negotiable:** Revocation is binding. Receiver must have escalation playbook in place.

**Implementation note:** Revocation is a new step in runbook (not a workflow step, but a post-completion action). Requires async job to monitor for revocation commands.

---

## 6. WEBHOOK EVENT CATALOG — New Event Types Required

The v1.0 Webhook baseline covers execution lifecycle (started, finished, failed) and user interaction (input_requested, input_received). Declaration flows require a **new family** of events.

### Declaration Event Family (New in This Scenario)

| Event Type | Trigger | Receiver Action | Idempotency |
|---|---|---|---|
| `declaration.submitted` | Declarant submits form (all fields filled, signature not yet captured) | Audit-log; optional internal queue | idempotency_key: `decl_X_submitted_v1` |
| `declaration.verified` | System completes verification step (e.g., ID check, KYC lookup) | Continue processing; may update status display | `decl_X_verified_v1` |
| `declaration.signed` | Declarant signature captured in form (witness not yet signed) | If single-party declaration, may trigger downstream. If multi-party, wait for all witnesses. | `decl_X_signed_v1` |
| `declaration.completed` | **Final state**: All required signatures captured, all verification passed, declaration is immutable. | Trigger downstream process: schedule surgery, bind policy, create compliance record, etc. | `decl_X_completed_v1` |
| `declaration.declined` | Declarant explicitly rejected (clicked "Decline"). | Cancel downstream process; notify participant; may trigger alternative flow. | `decl_X_declined_v1` |
| `declaration.abandoned` | Declarant did not finish (inactivity timeout, session closed). | Audit-log; may retry or escalate. | `decl_X_abandoned_v1` |
| `declaration.expired_unwitnessed` | Witness window passed without signature. | Retry or escalate witness flow; cancel downstream if SLA-critical. | `decl_X_expired_unwitnessed_v1` |
| `declaration.sealed` | N days post-completion: declaration locked in immutable archive, all SAS URLs become permanent. | Archive-ready notification; receiver should download evidence package to long-term storage. | `decl_X_sealed_v1` |
| `declaration.revoked` | Declarant (or authorized party) revoked consent post-completion. | **CRITICAL:** Cancel downstream process, reverse charges, halt procedure, notify all participants. | `decl_X_revoked_v1` |

### New Supporting Events (Related to Declaration Lifecycle)

| Event Type | Trigger | Receiver Action |
|---|---|---|
| `declaration.verification_failed` | KYC lookup failed; ID not found. | Declarant must retry verification; optional re-submission. |
| `declaration.witness_notification_sent` | Witness was notified to co-sign. | Optional tracking (witness system may integrate). |
| `declaration.witness_reminder_sent` | Reminder sent to unsigned witness. | Optional tracking. |

---

## 7. IMPLEMENTATION ARCHITECTURE

### 7.1 Event Routing & DLQ Strategy

**Current v1.0 model (per-execution webhook):**
- Customer provides webhook URL at run submission.
- Events queue in memory, retry with exponential backoff, DLQ on final failure.

**For declarations, we extend this:**

1. **Per-execution webhook** (current): Still used for all declaration events.
2. **Tenant-level webhooks** (v1.1 future): Allow customer to register a webhook for all declarations (not just per-execution).
3. **Event subscriptions** (v2.0 future): "Send me only `declaration.completed` and `declaration.revoked`" (filtering at registration time).

**DLQ strategy:**
- Declarations in DLQ are treated as **compliance-critical**.
- Admin replay endpoint shows DLQ items with highest priority.
- Default retention: indefinite (never auto-purge declaration events from DLQ).
- Alert: If any `declaration.completed` event sits in DLQ > 1h, page on-call.

### 7.2 Evidence Package Storage

**Signed document (PDF):**
- Blob Storage with WORM policy enabled after seal date.
- Before seal: time-limited SAS URL (7 days default).
- After seal: permanent URL (no expiration).

**Trace excerpt (JSONL):**
- Blob Storage, append blob for immutability.
- Checksum verified at download time.

**Hash chain (Merkle proof):**
- Computed daily in batch job.
- Stored as JSON document in Cosmos DB + Blob for archival.

**Audit receipt (PDF + QR):**
- Generated on-demand via endpoint.
- QR code points to sealed receipt on portal (permanent URL).

### 7.3 Revocation Handling

Revocation is the hardest edge case. Design:

1. **Revocation request API:** `POST /api/v1/declarations/{declaration_id}/revoke`
   - Requires auth (declarant or authorized party).
   - Idempotent: calling twice returns same result.

2. **Revocation event generation:**
   - `declaration.revoked` event created immediately.
   - Webhook dispatched (same retry/DLQ logic as completion events).

3. **Receiver's revocation handler:**
   - Must be idempotent (revoking twice = safe no-op).
   - Must notify downstream (cancel surgery, reverse charge).
   - Must log audit trail: "Revocation received at [time], cascade impact = [action]".

---

## 8. DELIVERY TIMELINE & DEPENDENCIES

### Phase 1: MVP (Sprint 1–2)

- [x] Webhook v1.0 baseline established (done; in decisions.md)
- [ ] Add 4 core declaration events: submitted, signed, completed, declined
- [ ] Evidence package structure defined (signed document + trace excerpt URLs)
- [ ] Idempotency key implementation (already in v1.0)

### Phase 2: Robustness (Sprint 3–4)

- [ ] Add 5 edge case events: abandoned, expired_unwitnessed, sealed, revoked, verification_failed
- [ ] DLQ strategy for declarations (compliance-critical alerting)
- [ ] Replay endpoint for declaration reconstruction

### Phase 3: Scale (Sprint 5+)

- [ ] Tenant-level webhook registration (v1.1)
- [ ] Event filtering at subscription time
- [ ] Hash chain / Merkle proof generation (batch job)
- [ ] Long-term SLA compliance (7+ year retrieval)

---

## 9. OPEN QUESTIONS FOR TEAM

1. **SAS URL expiration:** 7 days default? Configurable per declaration type?
2. **Merkle tree scope:** Daily batch (all declarations in a day) or per-tenant per-day?
3. **Revocation window:** Can revoke indefinitely, or only within N days of completion?
4. **Witness timeout:** 24h hardcoded, or configurable in runbook?
5. **DLQ retention:** Indefinite for declaration events, or cap at N days?
6. **Audit receipt printing:** PDF only, or support QR+text in email body?
7. **Cascade notifications:** When revocation happens, do we send email to patient + witness + surgeon, or only webhook to EHR?
8. **Historical API:** Can receiver request full event history for a declaration before webhooks existed (backdated query)?

---

## 10. REFERENCE: Related Decisions

- **Webhook & Event Delivery Baseline** — .squad/decisions.md, "Webhook & Event Delivery Baseline" section
- **Interactive Waiting Patterns** — .squad/decisions.md, approval gates + user input gates
- **C# Governance Parity** — Trace write synchronicity; JSONL immutability
- **Azure Service Stack** — Blob Storage WORM, Event Grid delivery, Cosmos DB for event metadata

---

**END OF ANALYSIS**  
**Next step:** Team review → approval → priority backlog assignment to Don (backend), John (Azure infra).

---

# Declaration/Consent UX Patterns — Analysis & Recommendations
**Leslie (Frontend Dev)** | 2026-06-04

## Executive Summary

This document analyzes seven critical UX pattern areas for declaration-before-process flows (surgical consents, insurance disclosures, exemption documents, etc.). Drawing on GERT's existing IUserInputGate interface, current timeout model, white-label constraints, and resumability requirements, I propose concrete flow designs and identify gaps requiring new primitives or team decisions.

**Key Findings:**
- **Presentation enforcement** requires frontend-only tracking (not gate-driven); current system lacks decision on disclosure minimums
- **Signature patterns** need new IUserInputGate kind to model typed, drawn, certificate-based, or OTP variants
- **Witness/co-sign flows** require architectural choice: IApprovalGate (email-link, different-session) vs new witness-specific primitive
- **Attachments** should use existing FileUpload gate but need camera vs picker UX decision
- **Resumability** should extend timeout to 60+ minutes for declarations (vs 15-min user input default); save-draft optional
- **White-label** audio/visual themeing is feasible; "Powered by GERT" watermark visibility requires governance decision
- **Receipt/confirmation** should issue QR-code-verified PDF with optional email delivery

---

## 1. Presentation Patterns: Enforcing Active Reading

### Current State
- IUserInputGate.Form kind supports multi-field presentation
- 15-minute user input timeout applies (fail-fast for active sessions)
- No frontend tracking for "scrolled to bottom," "time on page," "required expansion" enforcement

### Proposed Pattern
```
Declaration Presentation Flow:
┌─ Load runbook.Steps[N] (declaration gate)
├─ Render full document (scrolled to top)
├─ Disable "Accept" button until:
│  ├─ Document scrolled to bottom (frontend tracking)
│  ├─ 45 seconds elapsed (time minimum)
│  ├─ All accordion sections expanded (tracked client-side)
│  └─ (Optional) TTS read-aloud completion OR user clicks "I have read this"
├─ User clicks "Accept" → fires IUserInputGate.NextAsync(confirmed: true)
└─ Server processes confirmation; proceeds to next step
```

### Design Decisions Needed
1. **Scroll-to-bottom enforcement**: Hard requirement? Or optional UX nudge?
   - Recommendation: Hard requirement for legal declarations, optional for informational notices
2. **Time minimums**: 45 sec? 60 sec? Per-section or global?
   - Recommendation: 45 sec global minimum; configurable per customer via runbook metadata
3. **Read-aloud (TTS)**: Browser-native (Synthesis API) or pre-recorded audio?
   - Recommendation: Browser-native TTS for accessibility + scalability; pre-recorded optional for high-stakes legal

### New Frontend Primitives Required
- **DeclarationPresentation component** (wraps Form gate, tracks scroll/time/expansion state)
- **ScrollTracker hook** (detects scroll-to-bottom with sticky footer UX)
- **AccessibilityLayer** (implements ARIA labels, TTS control, keyboard nav for all accordion states)

### Fits Existing System?
**Partial.** Form gate can carry the declaration content, but presentation enforcement is 100% frontend. No server-side gate evaluation needed; gate fires only when presentation requirements are met client-side.

---

## 2. Signature Patterns: Legal Authority & Verification

### Current State
- IUserInputGate lacks signature-specific kind
- FileUpload gate exists for document capture
- No sub-types for typed, drawn, certificate-based, or OTP confirmation

### Proposed Pattern
```
Multi-Variant Signature Flow:

Option A: Typed Signature (Fastest)
┌─ Text gate: "Enter your full name as signature"
├─ User types: "John Q. Public"
├─ Render preview: signature in calligraphy font
└─ Confirm → IUserInputGate.NextAsync(signature: typed)

Option B: Drawn Signature (Legal Standard)
┌─ New "Signature" gate kind (pressure-sensitive canvas)
├─ User draws on iPad/trackpad/touch
├─ Capture: SVG path data + pressure curve + timestamp
├─ Render preview + "clear & retry" button
└─ Confirm → IUserInputGate.NextAsync(signature: drawn, pressureData: [...])

Option C: Certificate-Based (Enterprise)
┌─ Redirect to enterprise signing service (DigiCert, Adobe Sign)
├─ Return signed envelope token
└─ IUserInputGate.NextAsync(signature: cert, token: ABC123...)

Option D: OTP Confirmation (SMS/Email fallback)
┌─ Text gate: "6-digit code sent to +1-***-5555"
├─ User enters OTP
└─ IUserInputGate.NextAsync(signature: otp_confirmed)
```

### Design Decisions Needed
1. **Legal signature validity**: Drawn canvas only? Or accept typed?
   - Recommendation: Drawn (Option B) for medical/legal; typed acceptable for informational
2. **Pressure data retention**: Store in gate response? In audit log only?
   - Recommendation: Store in audit log; gate response includes signature_id reference
3. **OTP delivery**: SMS, email, or both?
   - Recommendation: Email default; SMS as premium/high-security option

### New Frontend Primitives Required
- **SignatureCanvas component** (pressure-sensitive drawing, undo/clear, preview)
- **SignatureGateKind** enum extending IUserInputGate (typed | drawn | cert | otp)
- **PressureCapture service** (hooks into PointerEvents API for pressure, tilt, speed)
- **SignaturePreview component** (renders SVG or image with verification badge)

### Fits Existing System?
**No.** Requires new IUserInputGate.Kind = "Signature" with sub-type discrimination. Drawn signature requires frontend canvas + pressure data capture. Gate response schema extends to include signature metadata (type, pressure, timestamp, verification_token).

---

## 3. Witness/Co-Sign Flows: Multi-Party Consent

### Current State
- IUserInputGate designed for single-user input (no witness scenario)
- IApprovalGate exists for email-link-driven approvals (different-session)
- No decision on same-session witness handoff (e.g., "pass device to next person")

### Proposed Pattern

#### Pattern A: Same-Session Witness (Quick & Simple)
```
Surgical Consent + Witness
┌─ User 1 (patient) completes declaration steps
├─ Reaches "Witness Signature" gate
├─ QR code + verbal code displayed: "Show witness your screen or say: #5739"
├─ UI enters "waiting for witness" state (shows countdown timer, "witness not ready" message)
├─ User 2 (witness) scans QR or enters #5739 on own device
├─ Both users see side-by-side confirmation ("Patient: Jane Doe", "Witness: Dr. Smith")
├─ User 2 enters witness signature (drawn or typed)
├─ Both signatures captured + witness timestamp logged
└─ Process continues
```

#### Pattern B: Different-Session Witness (Email Link)
```
Surgical Consent + Remote Witness
┌─ User 1 completes declaration, enters witness email: "dr.smith@hospital.com"
├─ Server creates witness token + email link
├─ Witness receives email: "You have been asked to witness consent form for [Patient Name]"
├─ Witness clicks link → IApprovalGate flow (existing gate kind)
├─ Witness sees read-only declaration + signs
├─ Process notifies User 1 of witness confirmation
└─ Process proceeds
```

#### Pattern C: Batch Witness (Forms Management)
```
Multi-Patient Exemption Batch
┌─ Witness (admin) receives email: "3 exemption forms awaiting your signature"
├─ Portal shows list of unsigned declarations
├─ Witness clicks each → IApprovalGate review/signature flow
├─ Witness can bulk-sign or reject with comments
└─ System notifies each patient of witness status
```

### Design Decisions Needed
1. **Same-session linking**: QR code, verbal code, or session ID + PIN?
   - Recommendation: QR (scannable) + verbal code (5-digit, memorable) as fallback
2. **Witness auth**: Require witness login? Or frictionless link?
   - Recommendation: Link-based for medical (trust email domain); login required for financial/legal
3. **Witness visibility**: Can witness see prior signatures? Full document or summary only?
   - Recommendation: Full document (legal requirement); redact sensitive data per runbook config

### New Frontend Primitives Required
- **WitnessLinkingGate** component (QR + verbal code display + device linking)
- **WaitingForWitness** state UI (spinner, countdown, retry messaging)
- **WitnessSideBySideConfirmation** component (dual signature preview)
- **WitnessPortal** (list view for batch witnessing)
- **WitnessTokenResolver** service (validates token + retrieves consent document state)

### Fits Existing System?
**Partial.** IApprovalGate handles email-witness pattern (Pattern B) with existing primitives. Same-session witness (Pattern A) requires new "WitnessLinking" gate kind or compound gate (IUserInputGate + sub-step for witness). Batch witness (Pattern C) uses existing IApprovalGate + frontend batch UI (new).

**Recommendation:** Implement Pattern B first (IApprovalGate); deferred Patterns A & C pending customer demand.

---

## 4. Attachments: Capture, Preview, Validation

### Current State
- IUserInputGate.FileUpload kind exists
- No guidance on camera vs file picker UX
- No server-side validation (OCR, format, content-based checks)
- No recovery flow if upload fails post-signature

### Proposed Pattern
```
Medical Document Attachment Flow
┌─ Declaration step complete; next: "Upload Insurance Card"
├─ Render FileUpload gate with camera + picker options
│  ├─ "📷 Take Photo" → launches device camera (iOS/Android native)
│  │  ├─ User frames document (auto-detect edges)
│  │  ├─ System warps perspective (document flatten)
│  │  └─ User previews + retakes if needed
│  └─ "📁 Choose File" → native file picker (accepts JPG/PNG/PDF)
├─ After selection: Client preview rendered (thumbnail + size check)
├─ User confirms upload → IUserInputGate.NextAsync(fileRef: ABC123)
├─ Server receives file; runs OCR validation:
│  ├─ Extract text + validate against required fields (expiration date, name match)
│  ├─ If OCR fails: return validation error → client shows "Unclear photo, please retake"
│  └─ If OCR passes: store file reference + proceed
└─ If process fails at approval stage, file remains available for re-attachment (resumable)
```

### Design Decisions Needed
1. **Camera vs picker default**: Which UX should be primary?
   - Recommendation: Camera-first on mobile (iOS/Android); picker-first on desktop
2. **Client-side validation**: Should we block upload before server validation?
   - Recommendation: Block on file size (>10MB) and format; let server OCR validation be authoritative
3. **Post-signature failure recovery**: If process fails after signature but before final approval, can user re-upload?
   - Recommendation: Yes, create "supplemental attachment" step; track which step attachment relates to
4. **PII redaction**: Should we allow OCR results in frontend logs?
   - Recommendation: No; log only success/failure codes, not OCR text

### New Frontend Primitives Required
- **CameraCapture component** (native camera, auto-edge-detection, perspective warp)
- **FileUploadGate component** (dual-mode: camera + picker, client-side preview)
- **DocumentPreview component** (thumbnail + metadata like size, capture date)
- **UploadProgressIndicator** (chunked upload, retry logic, timeout handling)
- **AttachmentValidationError** UI (server OCR failure message + "retake" CTA)

### Fits Existing System?
**Yes, with extension.** FileUpload gate carries attachment metadata; client-side preview is frontend-only. Server-side OCR validation is runbook-step logic (not gate logic). Post-signature recovery requires new "supplemental attachment" gate kind or metadata flag (attachment_phase: "initial" | "recovery").

**Recommendation:** Use existing FileUpload gate; add server-side OCR validation as optional runbook metadata; implement "supplemental attachment" flow for recovery scenarios.

---

## 5. Resumability: Long Documents & Network Resilience

### Current State
- IUserInputGate has 15-minute user input timeout (fail-fast for active sessions)
- Runbook execution uses TCS+SignalR (A6) or Durable Functions WaitForExternalEvent (A8)
- No save-draft pattern implemented

### Proposed Pattern
```
Long Declaration Resumability Flow (Surgical Consent)

Scenario: User starts consent form, gets interrupted after 8 min (page 2 of 3)

Timeline:
T+0:00   User loads declaration step → IUserInputGate.Form fires
T+8:30   User reads page 2, clicks "Continue Later" → Client saves form state locally + posts resumable_checkpoint
T+8:35   Server receives checkpoint: saves form_state blob + issues resume_token (JWT with expiry: 24h)
T+8:36   Client displays: "Your progress is saved. Resume link: [link]. Expires: tomorrow 11:53 PM"
T+24h    User clicks resume link → /runbook/{id}/resume?token=ABC123
         Server validates token + retrieves form_state
         Client hydrates form with prior responses (page 1 pre-filled, page 2 ready to edit, page 3 blank)
         User completes + signature → proceeds
T+24:05h User submits completed form → Server logs: "resumed from checkpoint @ T+8:30, completed @ T+24:05"
```

### Design Decisions Needed
1. **Timeout for declarations**: Keep 15 min (fail-fast)? Or extend to 60+ min?
   - Recommendation: **60-minute timeout for declarations** (vs 15-min user input default)
     - Rationale: Declarations are inherently slow reads; 15 min is too aggressive
     - Exception: Fields requiring live data (appointment details) stay 15-min timeout
2. **Save-draft button**: Always show? Or hide until >5 min in?
   - Recommendation: Show after 5 minutes; disable first 5 min to encourage continuous flow
3. **Resume token expiry**: 24h? 7 days? Configurable?
   - Recommendation: 24h default; configurable per customer (regulatory requirement may vary)
4. **Form state storage**: Client localStorage vs server DB?
   - Recommendation: Server DB (compliance/audit trail); optional client localStorage for redundancy
5. **Signature timeout**: Does signature expire if form is resumed?
   - Recommendation: Yes, signature must be re-entered if form resumed >5 min after initial completion

### New Frontend Primitives Required
- **CheckpointManager** hook (tracks form state, emits resumable_checkpoint events)
- **ContinueLaterButton** component (saves state + displays resume link with expiry)
- **ResumeFormHydration** hook (rehydrates saved form state from server)
- **ResumableStateBlob** (JSON serialization of form responses, field-by-field)
- **ResumeLinkExpiry** UI component (shows countdown to expiry)

### Fits Existing System?
**Partial.** 15-minute user input timeout is hardcoded; extending to 60+ minutes requires gate-level decision or declarative override (runbook metadata: `userInputTimeoutSec: 3600`). Save-draft pattern requires new "checkpoint" event type (separate from gate.NextAsync()).

**Recommendation:** 
- Extend IUserInputGate to include optional `timeoutSeconds` override (gate-level, not global)
- Add "CheckpointRequest" gate event (user-initiated save, not a gate firing)
- Use existing form state serialization; no new gate kind needed

---

## 6. White-Label Considerations: Themeing & Audit Integrity

### Current State
- White-label via Static Web Apps Standard (logo, color, font, text)
- "Powered by GERT" requirement for audit integrity unknown
- Declaration flows must not expose GERT internals (Leslie's charter)

### Proposed Pattern
```
Declaration UI Theming Matrix

┌─ Header/Footer
│  ├─ Logo: Themeable (customer brand)
│  ├─ Organization name: Themeable ("Acme Hospital" vs "GERT Medical")
│  ├─ "Powered by GERT": ??? (audit requirement TBD)
│  └─ Favicon: Themeable
│
├─ Typography
│  ├─ Declaration title: Themeable (color, size, weight)
│  ├─ Section headings: Themeable
│  ├─ Body text: Themeable (font family, size)
│  └─ Signature font: Themeable (calligraphy variant per brand)
│
├─ Color Scheme
│  ├─ Primary button (Accept/Sign): Themeable
│  ├─ Secondary button (Continue Later): Themeable
│  ├─ Accent (scroll indicators, checkboxes): Themeable
│  └─ Error states: Consistent red (non-themeable for safety)
│
├─ Legal/Compliance Text
│  ├─ Signature attestation: Themeable (customer's legal counsel wording)
│  ├─ Witness disclaimer: Themeable
│  ├─ Data privacy notice: Mix of themeable (org name) + non-themeable (GERT data practices)
│  └─ "Digital signature" footer: Non-themeable (audit/regulatory)
│
└─ Accessibility
   ├─ WCAG 2.1 AA color contrast: Non-negotiable (safety override)
   ├─ Font size: Responsive per browser zoom level (non-themeable floor: 12px)
   └─ TTS voice: Themeable (English, Spanish, etc. per customer market)
```

### Design Decisions Needed
1. **"Powered by GERT" visibility**: Required in all white-label customers?
   - Current assumption: Needed for audit/compliance (tbd)
   - Recommendation: Move to footer as small text (5pt) + always present for regulatory chain-of-custody
   - Alternative: Expose in metadata only (not visual UI); let customer decide visibility
2. **Signature attestation language**: Fixed GERT template or customer-supplied?
   - Recommendation: Customer-supplied via runbook metadata; GERT provides template; customer can override
3. **Camera UI themeing**: Should device camera interface reflect brand colors?
   - Recommendation: No; use native device camera (non-themeable for consistency)
4. **Audit watermark persistence**: Watermark visible if customer tries to screenshot/export PDF?
   - Recommendation: Implement server-side watermark on PDF export; client-side watermark on screenshot (CSS print media)

### New Frontend Primitives Required
- **ThemeProvider** component (centralizes color, font, spacing tokens)
- **ThemeContract** TypeScript interface (enforces required vs optional theme tokens)
- **AccessibilityThemeValidator** utility (ensures contrast ratios meet WCAG AA)
- **WatermarkLayer** component (overlay on screenshot + hidden print media watermark)
- **LocalizationProvider** (text + TTS language selection)

### Fits Existing System?
**Yes.** Static Web Apps Standard already supports themeing via config. No new gate kinds needed. Watermark logic is UI-layer concern (not gate-related).

**Recommendation:** Standardize on ThemeProvider + ThemeContract; implement watermark as PrintMediaLayer + ScreenshotDetection (passive monitoring, no active protection).

---

## 7. Receipt/Confirmation: Proof of Signature

### Current State
- No confirmation/receipt pattern defined
- No QR-code-based verification
- PDF download not yet modeled

### Proposed Pattern
```
Declaration Receipt Flow

After signature completion:
┌─ Server generates confirmation_id: "DECL-2026-06-04-A7K9M"
├─ Creates receipt blob:
│  ├─ Declaration title + signatory names
│  ├─ Signature timestamps (user signed @ 2026-06-04T15:23:00Z, witness @ 15:24:30Z)
│  ├─ Hash of full document (SHA256)
│  ├─ QR code (encodes: confirmation_id + document_hash + issuer_id)
│  └─ Audit metadata (IP, browser, device model)
├─ Renders on-screen confirmation:
│  ├─ "Your declaration has been signed and recorded"
│  ├─ Confirmation ID (copyable): DECL-2026-06-04-A7K9M
│  ├─ QR code (scannable for verification)
│  ├─ "Download as PDF" button → generates + signs PDF (include watermark)
│  ├─ "Email me a copy" button → sends receipt + PDF to registered email
│  └─ "Print" button → renders print-friendly HTML
├─ PDF generation (server-side):
│  ├─ Embed original declaration text + signature image
│  ├─ Add embedded QR code (links to: /verify?qr=ABC123)
│  ├─ Digital signature (X.509 cert sign the PDF)
│  ├─ Watermark: "Digitally Signed - Original on file at [institution]"
│  └─ Include audit metadata in PDF metadata fields
└─ QR verification endpoint (/verify?qr=ABC123):
   ├─ Decode QR
   ├─ Lookup confirmation by confirmation_id + hash
   ├─ Return public view: "Signed by [Name] on [Date/Time]" (no sensitive data)
   └─ Optional: institutions can verify authenticity via API endpoint
```

### Design Decisions Needed
1. **QR code lifespan**: Permanent? Or expire after 7 days?
   - Recommendation: Permanent; but add expiration flag for future non-repudiation scenarios
2. **Email delivery**: Always offer? Or only if runbook config allows?
   - Recommendation: Offer by default; respect customer's email retention policy (delete after 90 days if config says so)
3. **PDF digital signature**: Required? Or optional/premium?
   - Recommendation: Optional for now; implement for enterprise tier only (requires X.509 cert provisioning)
4. **Public verification**: Can anyone scan QR + see who signed?
   - Recommendation: Public basic info (name, date) only; no PII or medical details; verification requires auth for full details

### New Frontend Primitives Required
- **ConfirmationScreen** component (confirmation ID, QR, CTA buttons)
- **ReceiptQRCode** component (encodes confirmation + hash, renders QR)
- **PDFGenerator** service (async, runs on server, returns download URL)
- **EmailReceiptForm** component (email + consent for delivery)
- **VerificationPortal** (public QR scan → confirmation page)

### Fits Existing System?
**Yes.** Receipt is post-gate; no new gate kinds needed. QR encoding + PDF generation are server-side utilities (not gate-related). Frontend components (ConfirmationScreen, ReceiptQRCode) are standard React components, no gate extensions required.

**Recommendation:** Implement immediately; can use existing infrastructure. QR verification portal is low-risk, high-value for customer audit trails.

---

## Summary: New Primitives & System Gaps

### New IUserInputGate Kinds Needed
1. **"Signature"** (replaces implicit Form gate for signatures)
   - Sub-types: `typed | drawn | cert | otp`
   - Response includes: `signature_data, signature_type, pressure_metadata (if drawn), timestamp`
   - 15-min timeout should extend to 60 min for drawn signatures (UX: drawing pressure capture takes time)

2. **"WitnessLinking"** (optional, deferred; use IApprovalGate for now)
   - Enables same-session witness handoff via QR + verbal code
   - Response includes: `witness_linking_token, witness_device_id, linked_at_timestamp`

### New Frontend Components (No Gate Changes)
- DeclarationPresentation (wraps Form, enforces scroll/time/expansion)
- SignatureCanvas (pressure-sensitive drawing)
- CameraCapture (auto-edge-detection, perspective warp)
- WaitingForWitness (QR + countdown)
- CheckpointManager (save-draft state tracking)
- ThemeProvider (white-label centralization)
- ConfirmationScreen (receipt + QR + download)

### System Extensions (Not New Primitives)
- IUserInputGate.timeoutSeconds override (gate-level, not global)
- CheckpointRequest event type (user-initiated save, not gate firing)
- Server-side OCR validation (runbook metadata flag)
- PDF digital signature (enterprise feature, server-side)
- QR verification endpoint (public portal, not gate-related)

### Design Decisions Requiring Team Ratification
1. **"Powered by GERT" watermark visibility** in white-labeled declarations (audit vs aesthetic)
2. **Witness topology**: IApprovalGate (email-link) vs new WitnessLinking gate (same-session QR)
3. **Declaration timeout**: Extend to 60 min (vs 15-min user input default)?
4. **Signature variants**: Typed acceptable for non-medical? Or drawn-only requirement?
5. **Save-draft pattern**: Implement immediately? Or phase 2 (deferred pending customer demand)?

---

## Recommendations Going Forward

### Immediate (Next Sprint)
- [ ] Implement DeclarationPresentation component (scroll-to-bottom + time-minimum UX)
- [ ] Create Signature gate kind (typed + drawn canvas support)
- [ ] Extend IUserInputGate.timeoutSeconds for declaration-specific overrides
- [ ] Build ConfirmationScreen + QR receipt flow

### Near-Term (Sprint+1/+2)
- [ ] Integrate server-side OCR validation with FileUpload gate
- [ ] Implement CameraCapture component (auto-edge-detection)
- [ ] Add CheckpointManager for save-draft (optional client-side localStorage)
- [ ] Build WitnessLinking gate (QR + verbal code handoff) as experiment; measure UX friction

### Medium-Term (Pending Customer Demand)
- [ ] IApprovalGate witness email-link flow (low effort, high value)
- [ ] PDF digital signature integration (enterprise tier)
- [ ] Batch witnessing portal (admin bulk-sign)
- [ ] MultiLanguage TTS for declarations (accessibility tier)

### Deferred (Architectural Review Required)
- [ ] Non-repudiation signatures (cert-based, time-stamped)
- [ ] Regulatory compliance hooks (HIPAA/GDPR audit metadata fields)
- [ ] Biometric signature capture (fingerprint-based on mobile)

---

## Appendix: Sample Runbook Metadata for Declarations

```yaml
# .squad/runbooks/surgical-consent.yml
steps:
  - id: read-consent
    kind: UserInput
    gateKind: Form
    config:
      userInputTimeoutSeconds: 3600  # 60 min vs default 15
      presentationEnforcement:
        scrollToBottom: true
        timeMinimumSeconds: 45
        expandAllAccordions: true
        readAloudTTS: true
      description: "Surgical Consent Form"
      fields:
        - id: consent-checkbox
          type: Confirmation
          label: "I have read and understood the above consent"

  - id: patient-signature
    kind: UserInput
    gateKind: Signature  # NEW
    config:
      signatureType: drawn  # or: typed | cert | otp
      timeoutSeconds: 3600
      description: "Sign below"

  - id: insurance-attachment
    kind: UserInput
    gateKind: FileUpload
    config:
      userInputTimeoutSeconds: 900  # 15 min for attachment capture
      cameraFirst: true
      ocrValidation: true
      ocrFields:
        - patientName
        - policyExpirationDate
      description: "Upload your insurance card"

  - id: witness-approval
    kind: Approval  # Existing IApprovalGate
    config:
      approverEmails:
        - "witness@hospital.com"
      description: "Awaiting witness signature"
      timeoutSeconds: 86400  # 24h for witness to respond

  - id: confirmation
    kind: UserInput
    gateKind: Confirmation
    config:
      description: "Receipt & Confirmation"
      includeQRCode: true
      includePDFDownload: true
      includeEmailOption: true
```

---

**End of Document**

Prepared by: Leslie (Frontend Dev)
Date: 2026-06-04
Status: Ready for team review & design ratification

---

## Leslie — White-Label Frontend Foundations Decisions

**Author:** Leslie (Frontend Dev)  
**Date:** 2026-06-04T17:14:51-07:00  
**Status:** Team inbox for ratification

### Proposed Decisions

1. **Correct SWA custom-domain assumption**  
   Treat Azure Static Web Apps Standard as having a small per-app custom-domain quota, not unlimited domains. Current Microsoft docs disagree between 5 and 6, so design to 5 until John verifies the live subscription limit.

2. **Keep explicit hostname onboarding**  
   Each tenant hostname is explicitly validated, bound, mapped to a tenant record, and issued a managed certificate. Do not depend on wildcard SWA routing for MVP.

3. **Use Front Door as the domain/WAF scale layer**  
   Direct SWA custom domains are acceptable for low-count MVP tenants. Add Azure Front Door Standard/Premium when WAF, bot/rate controls, custom certificate policy, many hostnames, wildcard strategy, or multiple SWA origins are required.

4. **Tier tenant isolation**  
   Default to shared SWA + shared A6 data plane with strict `tenantId` enforcement. Move tenants to shared Front Door controls or dedicated stacks when regulation, private networking, release cadence, region, WAF, or volume demands it.

5. **Keep SWA as static shell only**  
   SWA should host the React shell and static assets. App Service remains authoritative for identity validation, run state, tenant data access, and audit progression. Move the frontend to App Service or Container Apps only for SSR, custom edge/server logic, BYO cert/TLS control, region constraints, or artifact/domain scale beyond SWA.

### Open Questions for Team

- John: what custom-domain quota is active in our target Azure subscription and region?
- Barbara/John: which tenant tier gets Front Door from day one?
- Product: do enterprise customers require BYO certificates or are Azure-managed certificates acceptable?
- Barbara/Don: what traffic/security threshold triggers dedicated tenant stack migration?

---

## Team Directive: Pronoun Usage

**Date:** 2026-06-04T17:15:45-07:00  
**By:** ormasoftchile (via Copilot)  

**Directive:** Leslie uses he/him pronouns. All agents and the coordinator must refer to Leslie with he/him going forward.
