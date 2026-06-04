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
