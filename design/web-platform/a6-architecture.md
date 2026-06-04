# A6 Architecture: GERT Web Platform MVP

**Author:** Barbara (Lead / Architect)  
**Date:** 2026-06-03  
**Status:** Proposal  
**Topology:** App Service + BackgroundService (C#)

---

## Executive Summary

A6 is the MVP architecture for the GERT web execution platform. It uses a single **Azure App Service** running a C# **BackgroundService** worker pattern. This topology eliminates cold starts, simplifies approval gate handling (native async/await), and deploys as a single unit. The C# runtime provides governance parity with the reference Go implementation while enabling trivial async pause/resume for approval gates.

**Key constraints preserved:**
- Pipeline: `Parser.Parse → Planner.Plan → Runtime.Start → RunHandle.Next()`
- Governance fires in `NextAsync()` before every action
- JSONL trace written synchronously before step proceeds
- RED LINE: No per-step Durable Functions (bypasses governance)

---

## 1. Component Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              GERT Web Platform (A6)                         │
└─────────────────────────────────────────────────────────────────────────────┘

                          ┌─────────────────────────┐
                          │   Static Web Apps       │
                          │   (Customer Portals)    │
                          │   • White-label UI      │
                          │   • Per-tenant domains  │
                          └───────────┬─────────────┘
                                      │ HTTPS
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Entra External Identities                          │
│                          (Customer B2C Auth)                                │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │ JWT
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              App Service (B1/P1v3)                          │
│  ┌────────────────────────┐    ┌────────────────────────────────────────┐  │
│  │     ASP.NET Core       │    │      BackgroundService Worker          │  │
│  │     Web API            │    │  ┌──────────────────────────────────┐  │  │
│  │  • POST /runs          │    │  │     GERT C# Runtime              │  │  │
│  │  • GET  /runs/{id}     │◄───┤  │  • Parser.Parse()               │  │  │
│  │  • POST /runs/{id}/    │    │  │  • Planner.Plan()               │  │  │
│  │         approve        │    │  │  • Runtime.Start()              │  │  │
│  │  • WebSocket /events   │    │  │  • RunHandle.NextAsync()  ◄──────┤──┤──┤── GOVERNANCE SEAM
│  │                        │    │  │  • IApprovalGate.Check()        │  │  │
│  └────────────┬───────────┘    │  └──────────────────────────────────┘  │  │
│               │                │                  │                      │  │
│               │                └──────────────────┼──────────────────────┘  │
│               │                                   │                         │
└───────────────┼───────────────────────────────────┼─────────────────────────┘
                │                                   │
       ┌────────┴────────┐              ┌──────────┴──────────┐
       ▼                 ▼              ▼                     ▼
┌─────────────┐   ┌─────────────┐ ┌─────────────┐      ┌─────────────┐
│  Service    │   │  Cosmos DB  │ │   Blob      │      │  Event Grid │
│  Bus Std    │   │  (Run State)│ │  Storage    │      │  (Webhooks) │
│             │   │             │ │  (Traces)   │      │             │
│ • runs      │   │ • Runs      │ │ • JSONL     │      │ • Customer  │
│ • webhooks  │   │ • Runbooks  │ │   traces    │      │   endpoints │
│ • dlq       │   │ • Tenants   │ │ • Artifacts │      │             │
└─────────────┘   └─────────────┘ └─────────────┘      └─────────────┘
```

---

## 2. Component Inventory

| Component | Azure Service | Purpose | Owns |
|-----------|--------------|---------|------|
| **Customer Portal** | Static Web Apps Standard | White-labeled SPA per customer | UI, custom domains, CDN |
| **Auth Provider** | Entra External Identities | Customer end-user authentication | User sessions, MFA, JWT issuance |
| **API Layer** | App Service (P1v3) | REST API, WebSocket, request routing | HTTP endpoints, auth validation |
| **Execution Worker** | BackgroundService (in App Service) | Queue-triggered runbook execution | GERT runtime lifecycle |
| **GERT C# Runtime** | In-process library (`internal sealed`) | Parser, Planner, Runtime, Governance | Execution semantics, trace writes |
| **Run Queue** | Service Bus Standard | Async run dispatch, at-most-once delivery | Message ordering, DLQ |
| **Webhook Queue** | Service Bus Standard (separate queue) | Async webhook delivery with retry | Outbound event delivery |
| **Run State Store** | Cosmos DB (serverless) | Run records, approval state, tenant config | Durable run state |
| **Trace Storage** | Blob Storage (Hot tier) | JSONL trace files, artifacts | Immutable audit trail |
| **Webhook Delivery** | Event Grid + Service Bus | Customer endpoint notification | Retry logic, DLQ |

---

## 3. Data Flow: Happy Path

**Scenario:** Customer end-user triggers a runbook through a white-labeled portal.

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ 1. END USER ACTION                                                           │
│    User clicks "Start Process" in customer portal                            │
│    → Static Web App → POST /runs with JWT                                    │
└──────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│ 2. API VALIDATION                                                            │
│    App Service API validates JWT (Entra External ID)                         │
│    → Extract tenant_id, user_id from claims                                  │
│    → Validate user has permission to run this runbook                        │
│    → Create Run record in Cosmos DB (status: "queued")                       │
│    → Enqueue message to Service Bus "runs" queue                             │
│    → Return 202 Accepted with run_id                                         │
└──────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│ 3. QUEUE PICKUP                                                              │
│    BackgroundService dequeues message (Service Bus session lock)             │
│    → At-most-once guarantee: session lock prevents duplicate processing      │
│    → Load runbook from Cosmos DB or Blob (tenant-scoped)                     │
│    → Update Run status: "running"                                            │
└──────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│ 4. GERT RUNTIME EXECUTION                                                    │
│    var parsed = await Parser.ParseAsync(runbookYaml);                        │
│    var plan = await Planner.PlanAsync(parsed, options);                      │
│    using var handle = await Runtime.StartAsync(plan, runOptions);            │
│                                                                              │
│    while (true) {                                                            │
│        var result = await handle.NextAsync();  // ◄── GOVERNANCE SEAM        │
│        if (result.IsCompleted) break;                                        │
│                                                                              │
│        // JSONL trace written BEFORE this point returns                      │
│        // Events emitted to subscribers                                      │
│        // Approval gates handled via IApprovalGate                           │
│    }                                                                         │
└──────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│ 5. COMPLETION                                                                │
│    → Update Run status: "completed" | "failed"                               │
│    → Flush JSONL trace to Blob Storage                                       │
│    → Enqueue webhook message to Service Bus "webhooks" queue                 │
│    → Complete Service Bus message (removes from queue)                       │
└──────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│ 6. WEBHOOK DELIVERY                                                          │
│    Webhook worker dequeues from "webhooks" queue                             │
│    → POST to customer endpoint with HMAC-SHA256 signature                    │
│    → Retry with exponential backoff (5 attempts over 24h)                    │
│    → On persistent failure: move to DLQ for manual replay                    │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Governance Seam

The governance seam is the critical enforcement point where authorization, policy, and audit fire. In A6, this happens **inside `IRunHandle.NextAsync()`**.

### 4.1 Where Governance Fires

```csharp
public sealed class RunHandle : IRunHandle
{
    private readonly ITraceWriter _traceWriter;
    private readonly IGovernanceEngine _governance;
    private readonly IApprovalGate _approvalGate;
    
    public async Task<StepResult> NextAsync(CancellationToken ct = default)
    {
        var step = GetNextStep();
        
        // 1. PRE-FLIGHT GOVERNANCE CHECK
        var verdict = await _governance.EvaluateAsync(step, _state);
        if (verdict.Denied)
        {
            await WriteTraceEvent(new GovernanceBlockedEvent(step, verdict));
            throw new GovernanceDeniedException(verdict);
        }
        
        // 2. APPROVAL GATE (if required by step policy)
        if (step.RequiresApproval)
        {
            await WriteTraceEvent(new ApprovalRequestedEvent(step));
            var approval = await _approvalGate.WaitForApprovalAsync(step, ct);
            await WriteTraceEvent(new ApprovalReceivedEvent(step, approval));
            
            if (approval.Rejected)
            {
                await WriteTraceEvent(new RunCancelledEvent("approval_rejected"));
                throw new ApprovalRejectedException(approval);
            }
        }
        
        // 3. TRACE WRITE (synchronous, before action)
        await WriteTraceEvent(new StepStartedEvent(step));
        
        // 4. EXECUTE STEP
        var result = await ExecuteStepAsync(step, ct);
        
        // 5. TRACE WRITE (synchronous, after action)
        await WriteTraceEvent(new StepCompletedEvent(step, result));
        
        return result;
    }
    
    private async Task WriteTraceEvent(ITraceEvent evt)
    {
        // CRITICAL: Synchronous write to durable storage
        // The trace is the authoritative record
        await _traceWriter.AppendAsync(evt);
    }
}
```

### 4.2 JSONL Trace Preservation

| Requirement | Implementation |
|-------------|----------------|
| **Trace is authoritative** | Blob Storage (append blob) with immutable policy |
| **Sync write before proceed** | `WriteTraceEvent()` awaits Blob append completion |
| **No event loss** | Trace write failure = step failure (fail-safe) |
| **Replay determinism** | Trace contains all inputs, outputs, decisions |

### 4.3 What Can Fail Without Losing Audit Trail

| Failure | Audit Trail Impact | Recovery |
|---------|-------------------|----------|
| App Service crash mid-step | ✅ Preserved (last event in trace) | Resume from checkpoint |
| Blob Storage unavailable | ⚠️ Step blocked until available | Retry with backoff |
| Cosmos DB unavailable | ⚠️ Status update delayed | Trace is source of truth |
| Webhook delivery fails | ✅ Preserved (DLQ) | Manual replay |
| Service Bus unavailable | ✅ Run already in progress | Complete; webhook queued later |

---

## 5. IApprovalGate Interface (A6→A8 Migration Seam)

This interface is **stubbed from day one** so that A6→A8 migration is surgical. In A6, approvals are handled via in-memory polling or WebSocket push. In A8, the same interface delegates to Durable Functions orchestrator.

```csharp
/// <summary>
/// Approval gate abstraction for governance-required human approvals.
/// A6 (MVP): In-memory polling with Cosmos DB state.
/// A8 (Future): Durable Functions external event.
/// </summary>
public interface IApprovalGate
{
    /// <summary>
    /// Block until approval is granted, rejected, or times out.
    /// </summary>
    /// <param name="step">Step awaiting approval</param>
    /// <param name="ct">Cancellation token (respects timeout)</param>
    /// <returns>Approval decision with approver identity and timestamp</returns>
    Task<ApprovalDecision> WaitForApprovalAsync(
        ResolvedStep step, 
        CancellationToken ct = default);
    
    /// <summary>
    /// Submit an approval decision (called by API when approver acts).
    /// </summary>
    /// <param name="runId">Run awaiting approval</param>
    /// <param name="stepId">Step awaiting approval</param>
    /// <param name="decision">Approval or rejection with justification</param>
    Task SubmitDecisionAsync(
        string runId, 
        string stepId, 
        ApprovalDecision decision);
    
    /// <summary>
    /// Query pending approvals (for admin dashboard).
    /// </summary>
    Task<IReadOnlyList<PendingApproval>> GetPendingApprovalsAsync(
        string? tenantId = null);
}

public sealed record ApprovalDecision(
    bool Approved,
    string ApproverIdentity,
    string? Role,
    string? Comment,
    DateTimeOffset Timestamp);

public sealed record PendingApproval(
    string RunId,
    string StepId,
    string TenantId,
    IReadOnlyList<string> RequiredRoles,
    int MinApprovals,
    int CurrentApprovals,
    DateTimeOffset RequestedAt,
    TimeSpan? Timeout);
```

### 5.1 A6 Implementation (MVP)

```csharp
public sealed class PollingApprovalGate : IApprovalGate
{
    private readonly CosmosContainer _approvals;
    private readonly TimeSpan _pollInterval = TimeSpan.FromSeconds(5);
    
    public async Task<ApprovalDecision> WaitForApprovalAsync(
        ResolvedStep step, 
        CancellationToken ct)
    {
        // Store pending approval in Cosmos DB
        await StorePendingApprovalAsync(step);
        
        // Poll until decision or timeout
        while (!ct.IsCancellationRequested)
        {
            var decision = await CheckForDecisionAsync(step.Id);
            if (decision != null)
                return decision;
            
            await Task.Delay(_pollInterval, ct);
        }
        
        throw new OperationCanceledException("Approval timeout");
    }
}
```

### 5.2 A8 Implementation (Future)

```csharp
public sealed class DurableFunctionsApprovalGate : IApprovalGate
{
    private readonly IDurableOrchestrationContext _context;
    
    public async Task<ApprovalDecision> WaitForApprovalAsync(
        ResolvedStep step, 
        CancellationToken ct)
    {
        // Durable Functions external event (survives orchestrator replay)
        return await _context.WaitForExternalEvent<ApprovalDecision>(
            $"approval:{step.Id}",
            step.ApprovalTimeout ?? TimeSpan.FromDays(7));
    }
}
```

---

## Interactive Waiting Patterns

A6 runbooks encounter two distinct types of execution suspension: **approval steps** (external human gate, long wait) and **user input steps** (portal user interaction, short wait). Both suspend execution waiting for external input, but the input channel, expected latency, and notification path differ fundamentally.

---

### Approval Steps

#### Mechanics (A6)

The BackgroundService thread executing a run hits `IApprovalGate.WaitForApprovalAsync()` and enters a polling loop against Cosmos DB. The thread is held alive (via `Task.Delay` with a `CancellationToken`) for the duration of the wait — potentially hours.

1. Runtime encounters a step with `RequiresApproval = true`
2. Writes `ApprovalRequestedEvent` to JSONL trace
3. Creates a pending approval record in Cosmos DB `approvals` container
4. Enters poll loop: check Cosmos DB every 5 seconds for a decision
5. When the approver submits via `POST /runs/{id}/steps/{stepId}/approve`, the API writes the decision to Cosmos DB
6. Next poll iteration picks up the decision and returns

#### IApprovalGate Seam (reference)

Defined in [Section 5](#5-iapprovalgate-interface-a6a8-migration-seam) above. The key method:

```csharp
Task<ApprovalDecision> WaitForApprovalAsync(ResolvedStep step, CancellationToken ct);
```

#### Portal UX for the Approver

1. When the approval record is created, a notification is dispatched (email, push, or in-app depending on tenant config)
2. The notification contains a deep link: `https://{tenant}.portal/approve/{runId}/{stepId}?token={one-time-token}`
3. The approver clicks the link → portal loads approval context (step description, run metadata, requester identity)
4. Approver clicks **Approve** or **Reject** (with optional comment)
5. Portal calls `POST /runs/{runId}/steps/{stepId}/approve` with the decision
6. API writes decision to Cosmos DB → next poll cycle resumes execution

#### Database State

| Container | Document | Key Fields |
|-----------|----------|------------|
| `approvals` | Pending approval | `id`, `runId`, `stepId`, `tenantId`, `requestedAt`, `timeout`, `requiredRoles[]`, `status` ("pending" / "approved" / "rejected" / "timed_out") |
| `approvals` | Decision | `id`, `runId`, `stepId`, `approverIdentity`, `decision`, `comment`, `decidedAt` |
| `runs` | Run record | `status` = "waiting_approval", `currentStepId`, `waitingSince` |

Partition key for `approvals` is `/runId` — all approval records for a run are co-located.

#### Timeout Behavior

1. `CancellationToken` is configured with the step's `approvalTimeout` (default: 24 hours)
2. When the token fires, `WaitForApprovalAsync` throws `OperationCanceledException`
3. The runtime writes `ApprovalTimedOutEvent` to the trace
4. Run status is set to `"failed"` with reason `"approval_timeout"`
5. Webhook fires with `execution.failed` event
6. The approval record in Cosmos DB is updated to `status: "timed_out"`

**A6 ceiling:** A held thread for 24 hours is wasteful but tolerable at MVP scale (1-3 tenants, <50 approval-gated runs/day). At scale this becomes untenable — A8 eliminates this cost entirely.

---

### User Input Steps

#### What Is a User Input Step?

A user input step suspends execution until the **portal session user** (not a third-party approver) provides required input. The input may be any form of user-provided data — not only a selection from a list. Examples:

- **Choice:** "Which plan do you want to apply for?" → [Basic, Premium, Enterprise]
- **Free-form text:** "Enter your tax ID number"
- **Confirmation / consent:** "I agree to these terms and conditions"
- **File upload:** "Attach a photo of your government-issued ID"
- **Form (multi-field):** "Fill in your contact details" → { name, email, phone, address }

This is fundamentally different from approval:
- The **actor** is the run subject (end-user), not an external approver
- The **latency** is seconds to minutes, not hours to days
- The **channel** is the live portal session, not a notification link
- No **escalation** or **delegation** logic applies
- The **input kind** varies — selections, text, files, confirmations, multi-field forms

#### IUserInputGate Interface

```csharp
/// <summary>
/// User input gate abstraction for all interactive input during execution.
/// The runbook suspends; the portal user must provide input; execution resumes.
/// A6 (MVP): In-memory TaskCompletionSource with Cosmos DB state for resilience.
/// A8 (Future): Durable Functions WaitForExternalEvent&lt;UserInputResponse&gt;.
/// </summary>
public interface IUserInputGate
{
    /// <summary>
    /// Suspend execution until the portal user provides the requested input.
    /// </summary>
    Task<UserInputResponse> AwaitUserInputAsync(
        string runId,
        string stepId,
        UserInputRequest request,
        CancellationToken cancellationToken);

    /// <summary>
    /// Submit the user's response (called by portal API when user completes input).
    /// </summary>
    Task SubmitResponseAsync(
        string runId,
        string stepId,
        UserInputResponse response);

    /// <summary>
    /// Query the pending input request for a run (portal reads this to render UI).
    /// </summary>
    Task<PendingUserInput?> GetPendingInputAsync(string runId);
}

/// <summary>Describes what input the step needs from the user.</summary>
public sealed record UserInputRequest(
    string StepLabel,
    UserInputKind Kind,
    string? Prompt = null,
    IReadOnlyList<ChoiceOption>? Options = null,   // only when Kind = Choice
    JsonDocument? Schema = null);                   // only when Kind = Form

/// <summary>The kind of input the step is waiting for.</summary>
public enum UserInputKind { Choice, Text, Confirmation, FileUpload, Form }

/// <summary>The user's response — exactly one field populated per Kind.</summary>
public sealed record UserInputResponse(
    string StepId,
    UserInputKind Kind,
    string? SelectedKey = null,      // Kind = Choice
    string? TextValue = null,        // Kind = Text or Confirmation ("yes"/"no")
    Uri? FileReference = null,       // Kind = FileUpload (blob URI after upload)
    JsonDocument? FormData = null);  // Kind = Form (matches Schema)

public sealed record ChoiceOption(
    string Key,
    string Label,
    string? Description,
    IReadOnlyDictionary<string, string>? Metadata);

public sealed record PendingUserInput(
    string RunId,
    string StepId,
    UserInputRequest Request,
    DateTimeOffset RequestedAt,
    TimeSpan Timeout,
    string Status);  // "pending" | "completed" | "timed_out"
```

#### A6 Implementation: TaskCompletionSource + SignalR

Unlike approval (polling every 5s), user input steps use **`TaskCompletionSource`** with **SignalR** push. Rationale: the portal user is actively in-session — we want sub-second response, not 5-second poll latency.

```csharp
public sealed class SignalRUserInputGate : IUserInputGate
{
    private readonly CosmosContainer _userInputs;
    private readonly ConcurrentDictionary<string, TaskCompletionSource<UserInputResponse>> _pending = new();

    public async Task<UserInputResponse> AwaitUserInputAsync(
        string runId,
        string stepId,
        UserInputRequest request,
        CancellationToken cancellationToken)
    {
        // 1. Persist pending input request to Cosmos DB (resilience)
        var pendingInput = new PendingUserInput(
            RunId: runId,
            StepId: stepId,
            Request: request,
            RequestedAt: DateTimeOffset.UtcNow,
            Timeout: TimeSpan.FromMinutes(15),
            Status: "pending");

        await _userInputs.UpsertItemAsync(pendingInput);

        // 2. Create in-memory TCS for instant resume
        var tcs = new TaskCompletionSource<UserInputResponse>(
            TaskCreationOptions.RunContinuationsAsynchronously);
        _pending[CompoundKey(runId, stepId)] = tcs;

        // 3. Push "input_required" event via SignalR to portal session
        await _hubContext.Clients
            .Group($"run:{runId}")
            .SendAsync("UserInputRequired", pendingInput, cancellationToken);

        // 4. Await response or timeout
        cancellationToken.Register(() => tcs.TrySetCanceled(cancellationToken));
        return await tcs.Task;
    }

    public async Task SubmitResponseAsync(
        string runId, string stepId, UserInputResponse response)
    {
        // Write response to Cosmos DB
        await _userInputs.PatchItemAsync<PendingUserInput>(
            stepId,
            new PartitionKey(runId),
            new[] { PatchOperation.Set("/response", response),
                    PatchOperation.Set("/status", "completed") });

        // Resume the waiting thread instantly
        var key = CompoundKey(runId, stepId);
        if (_pending.TryRemove(key, out var tcs))
            tcs.TrySetResult(response);
    }

    public async Task<PendingUserInput?> GetPendingInputAsync(string runId)
    {
        // Portal reads this on reconnect to render input UI
        var query = new QueryDefinition(
            "SELECT * FROM c WHERE c.runId = @runId AND c.status = 'pending'")
            .WithParameter("@runId", runId);

        var iterator = _userInputs.GetItemQueryIterator<PendingUserInput>(query);
        // ... return first or null
    }
}
```

#### How the BackgroundService Resumes

1. Portal user completes the input in the UI (selects an option, types text, uploads a file, etc.)
2. Portal calls `POST /runs/{runId}/steps/{stepId}/input` with a `UserInputResponse` payload
3. API controller calls `IUserInputGate.SubmitResponseAsync()`
4. `SubmitResponseAsync` sets the `TaskCompletionSource` result → the awaiting thread resumes **immediately** (no polling delay)
5. If the App Service restarted between request and response, the worker re-reads pending input from Cosmos DB and re-registers the TCS on recovery

#### Database State

| Container | Document | Key Fields |
|-----------|----------|------------|
| `user_inputs` | Pending input | `id` (stepId), `runId`, `stepId`, `request` (UserInputRequest), `requestedAt`, `timeout`, `status` ("pending" / "completed" / "timed_out"), `response` (UserInputResponse) |
| `runs` | Run record | `status` = "waiting_user_input", `currentStepId`, `waitingSince` |

Partition key for `user_inputs` is `/runId`.

#### Portal Read Flow

The portal needs to know what to render:
1. Portal subscribes to SignalR group `run:{runId}` on session start
2. When `UserInputRequired` event arrives, portal renders the appropriate input UI based on `request.Kind`
3. If portal reconnects mid-wait, it calls `GET /runs/{runId}/input/pending` → API reads Cosmos DB via `GetPendingInputAsync()`
4. Portal renders the input UI from the `PendingUserInput.Request` (options for Choice, text field for Text, upload widget for FileUpload, form schema for Form, etc.)

#### Timeout Behavior

- Default timeout: **15 minutes** (user abandoned session)
- On timeout: `OperationCanceledException` → trace writes `UserInputTimedOutEvent` → run fails with reason `"user_input_timeout"`
- Much shorter than approval timeout (24h) because the user is expected to be actively in-session

---

### A6 vs A8 Comparison: Waiting Patterns

| Dimension | A6: Approval | A6: User Input | A8: Both |
|-----------|-------------|----------------|----------|
| **Wait mechanism** | Polling loop (5s interval) | TaskCompletionSource + SignalR push | `WaitForExternalEvent<T>()` |
| **Thread cost during wait** | 1 thread held (expensive for long waits) | 1 thread held (cheap — short wait) | Zero (orchestrator replays on event) |
| **Expected latency** | Hours to days | Seconds to minutes | N/A (event-driven) |
| **Resume trigger** | Cosmos DB poll detects decision | TCS set by API call (instant) | External event raised to orchestration |
| **Input kinds** | Approve / Reject (binary) | Choice, Text, Confirmation, FileUpload, Form | Any (generic payload) |
| **Resilience on restart** | Re-poll from Cosmos DB | Re-register TCS, re-read pending input | Durable (survives replay natively) |
| **Compute cost of waiting** | ~$0.002/hr per waiting run (thread slot) | Negligible (short duration) | $0 (zero compute) |
| **Practical ceiling** | 50 concurrent approval waits | 100+ concurrent user input waits (short-lived) | Unlimited (event-driven) |
| **Killer argument for A8** | Approval waits of days = wasted compute | User input + approval on same model = simplicity | Zero-cost wait + infinite scale |

**Key insight:** User input steps are cheap in A6 (user responds quickly, TCS is instant). Approval steps are expensive in A6 (thread held for hours). The migration pressure to A8 comes primarily from approval-heavy workloads, not user-input-heavy ones.

---

### Migration Seam: IApprovalGate + IUserInputGate

Both interfaces are designed so that A6 implementations (polling/TCS) and A8 implementations (Durable Functions `WaitForExternalEvent`) are swappable behind DI registration.

#### Interface Summary (C#)

```csharp
// ──── APPROVAL GATE ────
public interface IApprovalGate
{
    Task<ApprovalDecision> WaitForApprovalAsync(ResolvedStep step, CancellationToken ct = default);
    Task SubmitDecisionAsync(string runId, string stepId, ApprovalDecision decision);
    Task<IReadOnlyList<PendingApproval>> GetPendingApprovalsAsync(string? tenantId = null);
}

// ──── USER INPUT GATE ────
public interface IUserInputGate
{
    Task<UserInputResponse> AwaitUserInputAsync(string runId, string stepId, UserInputRequest request, CancellationToken cancellationToken);
    Task SubmitResponseAsync(string runId, string stepId, UserInputResponse response);
    Task<PendingUserInput?> GetPendingInputAsync(string runId);
}
```

#### DI Registration (A6 vs A8)

```csharp
// A6 (MVP) — registered at startup
services.AddSingleton<IApprovalGate, PollingApprovalGate>();
services.AddSingleton<IUserInputGate, SignalRUserInputGate>();

// A8 (Future) — swap implementations, zero runtime code changes
services.AddScoped<IApprovalGate, DurableFunctionsApprovalGate>();
services.AddScoped<IUserInputGate, DurableFunctionsUserInputGate>();
```

#### A8 User Input Gate Implementation (Future)

```csharp
public sealed class DurableFunctionsUserInputGate : IUserInputGate
{
    private readonly IDurableOrchestrationContext _context;

    public async Task<UserInputResponse> AwaitUserInputAsync(
        string runId,
        string stepId,
        UserInputRequest request,
        CancellationToken cancellationToken)
    {
        // Zero compute during wait — orchestrator suspends entirely
        return await _context.WaitForExternalEvent<UserInputResponse>(
            $"user_input:{stepId}",
            TimeSpan.FromMinutes(15));
    }

    public Task SubmitResponseAsync(string runId, string stepId, UserInputResponse response)
    {
        // Raise event to the durable orchestration
        return _client.RaiseEventAsync(runId, $"user_input:{stepId}", response);
    }

    public async Task<PendingUserInput?> GetPendingInputAsync(string runId)
    {
        // Read from orchestration custom status
        var status = await _client.GetStatusAsync(runId);
        return status?.CustomStatus?.ToObject<PendingUserInput>();
    }
}
```

#### Governance Integration

Both gates fire **inside** `RunHandle.NextAsync()`, preserving the governance seam:

```csharp
public async Task<StepResult> NextAsync(CancellationToken ct = default)
{
    var step = GetNextStep();

    // Governance check...
    // Approval gate (if required)...

    // USER INPUT GATE (if step requires user input)
    if (step.RequiresUserInput)
    {
        var request = new UserInputRequest(
            step.Label, step.InputKind, step.Prompt, step.Options, step.FormSchema);
        await WriteTraceEvent(new UserInputRequestedEvent(step, request));
        var response = await _userInputGate.AwaitUserInputAsync(
            _currentRunId, step.Id, request, ct);
        await WriteTraceEvent(new UserInputReceivedEvent(step, response));

        // Inject response into step context (downstream actions use it)
        _state.SetVariable(step.OutputVariable, response);
    }

    // Trace write, execute step...
}
```

---

## 6. Failure Modes

### 6.1 Service Bus Unavailable

| Phase | Impact | Mitigation |
|-------|--------|------------|
| Run submission | ❌ 503 Service Unavailable | Client retry with backoff |
| Run in progress | ✅ No impact (already dequeued) | N/A |
| Webhook delivery | ⚠️ Delayed | Queue on reconnect |

**Blast radius:** New runs cannot be queued. Running runs unaffected. Webhooks delayed.

### 6.2 App Service Restart Mid-Run

| Scenario | Data Loss | Recovery |
|----------|-----------|----------|
| Before step starts | None | Re-process from queue (message still locked) |
| During step execution | Current step output | Retry step (idempotency required) |
| After step, before complete | None | Resume from checkpoint |

**Recovery procedure:**
1. Service Bus message lock expires (5 minutes default)
2. Message becomes visible again
3. New worker instance picks up message
4. Load run state from Cosmos DB
5. Resume from last completed step

### 6.3 Runbook Step Throws Exception

```csharp
try
{
    var result = await ExecuteStepAsync(step, ct);
    await WriteTraceEvent(new StepCompletedEvent(step, result));
}
catch (Exception ex) when (ex is not GovernanceDeniedException)
{
    // CRITICAL: Write failure event BEFORE propagating
    await WriteTraceEvent(new StepFailedEvent(step, ex));
    
    // Update run status
    await _runStore.UpdateStatusAsync(_runId, RunStatus.Failed);
    
    // Complete Service Bus message (don't retry failed runs)
    await _messageReceiver.CompleteAsync(message);
    
    throw;
}
```

### 6.4 Cosmos DB Unavailable

| Operation | Impact | Mitigation |
|-----------|--------|------------|
| Run creation | ❌ 503 | Client retry |
| Status update | ⚠️ Delayed | Trace is source of truth; reconcile later |
| Approval lookup | ⚠️ Approval blocked | Retry with backoff |

**Blast radius:** Run submission blocked. Status queries stale. Approvals delayed.

### 6.5 Blob Storage Unavailable

| Operation | Impact | Mitigation |
|-----------|--------|------------|
| Trace write | ❌ Step blocked | Retry with backoff; fail run if persistent |
| Artifact upload | ⚠️ Step blocked | Retry with backoff |

**Blast radius:** All running steps blocked. This is intentional—**trace is authoritative**.

---

## 7. Blast Radius Analysis

### 7.1 Single Points of Failure

| Component | Failure Impact | SPOF? | Mitigation |
|-----------|---------------|-------|------------|
| App Service | All runs halt | **YES** | Scale out to 2+ instances |
| Cosmos DB | Run state unavailable | **YES** | Multi-region (future) |
| Blob Storage | Traces unavailable | **YES** | RA-GRS replication |
| Service Bus | Queue unavailable | **YES** | Premium tier (future) |
| Entra External ID | Auth unavailable | No (Azure SLA) | N/A |

### 7.2 Failure Isolation

| Failure Scope | What's Affected | What's NOT Affected |
|---------------|-----------------|---------------------|
| One run fails | That run only | Other runs, other tenants |
| One tenant's runbook bad | That tenant's runs | Other tenants |
| Worker crashes | In-flight runs (1-10) | Queued runs (recovered) |
| App Service down | All runs for all tenants | Trace history (Blob) |

### 7.3 Worst Case Scenario

**Scenario:** App Service crashes with 10 concurrent runs mid-execution.

**Impact:**
- 10 runs interrupted
- 10 Service Bus messages return to queue after lock timeout
- Up to 10 steps may need re-execution (idempotency critical)
- Traces preserved up to last completed step

**Recovery time:** 5 minutes (message lock timeout) + restart time

**Data loss:** Zero (traces durable, messages durable)

---

## 8. Deferred to A8

The following capabilities are **explicitly NOT in A6**:

| Capability | Why Deferred | A8 Solution |
|------------|--------------|-------------|
| Multi-approver workflows | Requires durable state across days | Durable Functions orchestrator |
| Approval escalation | Complex timeout/fallback logic | Durable timer + sub-orchestration |
| Approval delegation | Org hierarchy integration | External event + role resolution |
| Approval audit dashboard | Low priority for MVP | Cosmos DB change feed + Power BI |
| Conditional approval routing | Step-by-step orchestration | Activity function chains |
| Long-running approval (>24h) | Worker keep-alive impractical | Durable external events |

### 8.1 What A6 DOES Support

| Capability | A6 Implementation |
|------------|-------------------|
| Single-approver gates | `IApprovalGate.WaitForApprovalAsync()` |
| Approval with timeout | `CancellationToken` with timer |
| Approval rejection | `ApprovalDecision.Approved = false` |
| Approval audit trail | JSONL trace events |
| Approval via API | `POST /runs/{id}/steps/{stepId}/approve` |
| Approval via WebSocket | Real-time push to portal |

---

## 9. Cost Estimate (MVP Scale)

**Assumptions:**
- 1-3 customers (tenants)
- ~100 runs/day total
- ~5 steps/run average
- ~10 webhooks/run
- ~50 active users/month

### 9.1 Monthly Cost Breakdown

| Service | SKU | Monthly Cost | Notes |
|---------|-----|--------------|-------|
| App Service | P1v3 (1 instance) | $81 | 2 vCPU, 8GB RAM, always-on |
| Cosmos DB | Serverless | ~$10 | 100 runs × 30 days × 5 ops = 15K RU |
| Blob Storage | Hot tier | ~$2 | 100 runs × 30 × 10KB trace = 30MB |
| Service Bus | Standard | $10 | Base + ~3K messages/month |
| Static Web Apps | Standard | $9 | Unlimited custom domains |
| Entra External ID | Free tier | $0 | <50K MAU free |
| Event Grid | Per-event | ~$1 | ~30K events/month |

### 9.2 Total Monthly Cost

| Scenario | Cost |
|----------|------|
| **MVP (1 instance)** | **~$113/month** |
| **Growth (2 instances)** | ~$194/month |
| **HA (3 instances + Premium SB)** | ~$350/month |

### 9.3 Cost Scaling Triggers

| Metric | Threshold | Action |
|--------|-----------|--------|
| CPU > 70% sustained | 1 hour | Scale to 2 instances |
| Runs > 500/day | Sustained | Evaluate A8 |
| Approvals > 50/day | Sustained | Evaluate A8 |
| Tenants > 10 | Any | Evaluate per-tenant isolation |

---

## Appendix A: Service Configuration

### A.1 App Service

```json
{
  "plan": "P1v3",
  "os": "Linux",
  "runtime": "dotnet|8.0",
  "alwaysOn": true,
  "healthCheck": "/health",
  "instances": {
    "min": 1,
    "max": 3,
    "default": 1
  }
}
```

### A.2 Service Bus

```json
{
  "tier": "Standard",
  "queues": [
    {
      "name": "runs",
      "maxDeliveryCount": 3,
      "lockDuration": "PT5M",
      "requiresSession": true
    },
    {
      "name": "webhooks",
      "maxDeliveryCount": 10,
      "lockDuration": "PT1M",
      "requiresSession": false
    }
  ]
}
```

### A.3 Cosmos DB

```json
{
  "kind": "GlobalDocumentDB",
  "capacityMode": "Serverless",
  "containers": [
    { "name": "runs", "partitionKey": "/tenantId" },
    { "name": "runbooks", "partitionKey": "/tenantId" },
    { "name": "tenants", "partitionKey": "/id" },
    { "name": "approvals", "partitionKey": "/runId" },
    { "name": "user_inputs", "partitionKey": "/runId" }
  ]
}
```

---

## Appendix B: API Contract Summary

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/runs` | POST | Submit new run |
| `/runs/{id}` | GET | Get run status and trace |
| `/runs/{id}/steps/{stepId}/approve` | POST | Submit approval decision |
| `/runs/{id}/steps/{stepId}/input` | POST | Submit user input response (UserInputResponse payload) |
| `/runs/{id}/input/pending` | GET | Get pending user input request for a run |
| `/runs/{id}/cancel` | POST | Cancel running run |
| `/tenants/{id}/runbooks` | GET | List available runbooks |
| `/events` | WebSocket | Real-time event stream |
| `/health` | GET | Health check |

---

## Appendix C: Decision Log

| Decision | Rationale |
|----------|-----------|
| App Service over Container Apps | Zero cold start, simpler deployment, adequate for MVP scale |
| P1v3 over B1 | B1 has 60-min warm-up; P1v3 always-on required for queue processing |
| Service Bus sessions | At-most-once delivery critical for governance integrity |
| Cosmos DB serverless | Pay-per-request; scales to zero; adequate for MVP |
| Blob append for traces | Immutable, append-only, cost-effective |
| `internal sealed` runtime | Prevents governance bypass via subclassing in shared process |

---

*Document version: 1.1.0*  
*Last updated: 2026-06-03T22:48:47-04:00*
