# Revised Topology Menu: C# Runtime Unlocked

**Author:** Barbara (Lead Architect)  
**Date:** 2026-06-03  
**Status:** Proposal — supersedes original 5-option menu  
**Trigger:** User removed Go-only constraint. C# native runtime is now in scope.

---

## Executive Summary

A C# runtime fundamentally changes the topology calculus. The "binary lifecycle" problem (starting/stopping a Go process, IPC overhead, cold-start binary pull) disappears. In its place we get:

- **In-process execution** (same CLR as the web API)
- **Native async/await** for approval gates (no serialization needed if process stays alive)
- **Familiar Azure hosting** without container image management
- **Simpler debugging, profiling, and observability** (one runtime, one telemetry pipeline)

**New risk introduced:** In-process execution couples the API's blast radius to the runtime's blast radius unless we design explicit isolation boundaries.

---

## Critical Architectural Questions Answered

### Q1: In-process with the web API, or separate worker service?

**Answer: Separate worker service, same deployment unit.**

A `BackgroundService` (IHostedService) runs in the same process as the API but on its own thread pool. This gives:
- Shared memory for fast communication (no IPC/RPC overhead)
- Independent failure domains at the logical level (a crashed run doesn't crash the API if we isolate correctly)
- Single deployment artifact (one App Service / one container)

However, for **enterprise tenants**, the worker MUST be a separate process/container to achieve true blast-radius isolation.

### Q2: Can App Service host API + BackgroundService in one process?

**Yes.** ASP.NET Core's `IHostedService` pattern is designed for this. The API handles HTTP, the BackgroundService pulls from Service Bus and executes runs. Both share the same DI container, same process, same App Service Plan.

### Q3: Does C# in-process eliminate the "Job" pattern entirely?

**For MVP: Yes.** No need for Container Apps Jobs if the runtime is in-process.  
**For enterprise: No.** Per-tenant isolation still requires separate compute units (Jobs, separate App Services, or Container Apps with sidecars).

### Q4: Blast radius of in-process vs. isolated execution?

| Mode | Blast Radius | Failure Scenario |
|------|-------------|------------------|
| In-process BackgroundService | **All runs on that instance** | Unhandled exception, memory leak, or CPU spin in one run affects all concurrent runs on same host |
| Separate worker process (same host) | **All runs on that host** | Process crash is isolated, but host resource exhaustion still shared |
| Container Apps Job (per-run) | **Single run** | One run = one container = perfect isolation |
| Dedicated per-tenant service | **Single tenant** | Tenant's runs are isolated from other tenants entirely |

### Q5: Approval gates (hours-long pauses)?

| Topology | Gate Strategy | Complexity |
|----------|--------------|-----------|
| In-process BackgroundService | **async/await + CancellationToken** — run stays alive, awaits approval signal via Service Bus message or SignalR | Low (native C# async) |
| App Service (Always On) | Same as above — process never recycles if Always On is enabled | Low |
| Container Apps Job | **Checkpoint + Resume** — serialize state, terminate job, new job on approval | High |
| Azure Functions | **Durable Functions** — native orchestrator pause/resume | Medium (Durable overhead) |

**Winner for approval gates: App Service with BackgroundService + Always On.** The run simply `await`s a `TaskCompletionSource` that fires when the approval arrives. No serialization, no state reconstruction, no cold restart.

### Q6: Tenant isolation with C# in-process vs. Go binary?

| Aspect | Go Binary (per-run process) | C# In-Process | C# Isolated Worker |
|--------|---------------------------|---------------|-------------------|
| Memory isolation | ✅ OS-level | ❌ Shared heap | ✅ Separate process |
| CPU isolation | ✅ OS scheduler | ⚠️ Thread pool contention | ✅ OS scheduler |
| Governance bypass risk | None (binary enforces) | Low (if Runtime class is sealed/internal) | None (same as Go) |
| Noisy neighbor | Impossible | Possible (must mitigate) | Impossible |

---

## Revised Option Scoring

### Original Options Re-Scored (with C# available)

#### A1: Thin Relay (Functions → Container Apps Job)
| Attribute | Score | Notes |
|-----------|-------|-------|
| Runtime | Go or C# | C# image is lighter (~80MB vs ~250MB with Go toolchain) |
| Blast radius | ⭐⭐⭐⭐⭐ | Per-run isolation, unchanged |
| Cold start | ⭐⭐⭐ | 3–8s (Go) or 2–5s (C# trimmed) |
| Isolation | ⭐⭐⭐⭐⭐ | Container-per-run |
| Cost | ⭐⭐⭐ | Per-run billing, zero when idle |
| Approval gates | ⭐⭐ | Still needs checkpoint/resume |
| **C# impact** | **Marginally better** | Smaller image, faster start, but the Job pattern adds complexity that C# could avoid entirely |

**Verdict:** Still valid but **no longer the recommended MVP** — C# unlocks simpler paths.

#### A2: Embedded Worker (Functions with Go handler)
| Attribute | Score | Notes |
|-----------|-------|-------|
| Runtime | Go only | C# makes this option irrelevant — use native Functions instead |
| **C# impact** | **Superseded** | Replace with B2 (Functions Isolated Worker) below |

**Verdict:** ❌ Retired. B2 is strictly superior with C#.

#### A3: Persistent Agent Pool (Container Apps + KEDA)
| Attribute | Score | Notes |
|-----------|-------|-------|
| Runtime | Go or C# | C# simplifies the pool worker |
| Blast radius | ⭐⭐ | Multi-run, multi-tenant on same instance |
| Isolation | ⭐⭐ | Thread-level only |
| Cold start | ⭐⭐⭐⭐⭐ | Always warm |
| Cost | ⭐⭐ | Always-on cost |
| Approval gates | ⭐⭐⭐⭐ | Long-lived process, natural await |
| **C# impact** | **Slightly better** | Simpler code, but blast radius concern unchanged |

**Verdict:** Still rejected for MVP. Blast radius unacceptable for governance-first product.

#### A4: Sidecar Agent (Container Apps per-tenant)
| Attribute | Score | Notes |
|-----------|-------|-------|
| Runtime | Go or C# | C# sidecar is lighter, faster to build/deploy |
| Blast radius | ⭐⭐⭐⭐⭐ | Per-tenant |
| Isolation | ⭐⭐⭐⭐⭐ | Dedicated compute per customer |
| Cold start | ⭐⭐⭐⭐ | Warm for active tenants |
| Cost | ⭐⭐ | Per-tenant always-on cost |
| Approval gates | ⭐⭐⭐⭐⭐ | Long-lived, natural await |
| **C# impact** | **Better** | No container image management if using App Service per-tenant instead |

**Verdict:** Still the enterprise endgame. C# makes it cheaper/simpler to implement.

#### A5: Hybrid Dispatch (Functions + Container Apps tiered)
| Attribute | Score | Notes |
|-----------|-------|-------|
| Runtime | Go or C# | C# simplifies both lanes |
| Blast radius | ⭐⭐⭐⭐ | Per-run on both lanes |
| Isolation | ⭐⭐⭐⭐ | Good |
| Cold start | ⭐⭐⭐⭐ | Warm lane for premium |
| Cost | ⭐⭐⭐ | Optimized |
| Approval gates | ⭐⭐⭐ | Complex (two patterns) |
| **C# impact** | **Better but still premature** | Easier to build both lanes, but why build two when one suffices? |

**Verdict:** Still premature for v1. Revisit when usage patterns emerge.

---

### NEW Options Unlocked by C# Runtime

#### B1: App Service + BackgroundService (⭐ NEW MVP RECOMMENDATION)

**Architecture:**
```
[Service Bus Queue] → [App Service (Always On)]
                         ├── ASP.NET Core Web API (HTTP)
                         └── BackgroundService (queue consumer + GERT C# runtime)
```

| Attribute | Score | Notes |
|-----------|-------|-------|
| Runtime | C# only | Native .NET 8 |
| Blast radius | ⭐⭐⭐ | All runs on same instance share process. Mitigated by: per-run exception isolation, memory limits per run, circuit breakers |
| Isolation | ⭐⭐⭐ | Logical isolation (separate async contexts), not OS-level |
| Cold start | ⭐⭐⭐⭐⭐ | **Zero.** Always On = always warm |
| Cost | ⭐⭐⭐⭐⭐ | B1 App Service Plan (~$55/mo) handles API + worker. No per-run billing |
| Approval gates | ⭐⭐⭐⭐⭐ | `await approvalSignal` — trivial. No serialization |
| Complexity | ⭐⭐⭐⭐⭐ | Single deployment, single codebase, standard ASP.NET patterns |
| Governance | ⭐⭐⭐⭐ | C# Runtime class is `internal sealed`, same enforcement as Go binary |
| Observability | ⭐⭐⭐⭐⭐ | Single Application Insights instance, correlated traces |
| Scaling | ⭐⭐⭐⭐ | App Service auto-scale (CPU/queue depth). Multiple instances = parallel runs |

**Blast radius mitigation:**
1. Each run executes in its own `Task` with isolated `CancellationTokenSource`
2. Unhandled exceptions in one run do not propagate (wrapped in try/catch at the run boundary)
3. Memory: Set per-run allocation ceiling; if exceeded, run is terminated (not the process)
4. CPU: Use `SemaphoreSlim` to cap concurrent runs per instance
5. Health check endpoint: if process is unhealthy, App Service restarts it (all in-flight runs retry from queue)

**Why this is the new MVP recommendation:**
- Zero cold start solves the UX problem
- Approval gates are trivial (no checkpoint/resume complexity)
- Single deployment unit = fastest path to production
- Cost-effective for early stage ($55/mo vs per-run billing)
- Governance preserved: C# Runtime is sealed, internal, same enforcement guarantees
- Blast radius is **acceptable for MVP** (shared process) with clear upgrade path to B4/A4

---

#### B2: Azure Functions Isolated Worker (.NET 8)

**Architecture:**
```
[Service Bus Queue] → [Azure Functions (Isolated Worker, .NET 8)]
                         └── GERT C# Runtime (in-process with Function)
```

| Attribute | Score | Notes |
|-----------|-------|-------|
| Runtime | C# only | .NET 8 Isolated Worker |
| Blast radius | ⭐⭐⭐⭐ | Per-function-instance (multiple runs may share an instance) |
| Isolation | ⭐⭐⭐⭐ | Better than App Service (Functions infra manages instance lifecycle) |
| Cold start | ⭐⭐⭐ | 1–3s on Premium plan; Consumption plan: 5–10s |
| Cost | ⭐⭐⭐⭐ | Consumption: pay-per-execution. Premium: ~$100/mo always-warm |
| Approval gates | ⭐⭐ | **Problem:** Function timeout (5min Consumption, 30min Premium, unlimited on Dedicated) |
| Governance | ⭐⭐⭐⭐ | Same as B1 |
| Scaling | ⭐⭐⭐⭐⭐ | Automatic, event-driven |

**Limitations:**
- Approval gates requiring hours-long waits are incompatible with Consumption plan timeout
- Premium plan removes timeout but costs more than App Service
- No advantage over B1 for this workload unless you need massive burst scaling

**Verdict:** Valid for short-running runbooks without approval gates. Not recommended as primary topology due to gate limitations.

---

#### B3: Durable Functions Orchestrator

**Architecture:**
```
[Service Bus Queue] → [Durable Function Orchestrator]
                         └── Calls Activity: "ExecuteFullRun" (GERT C# Runtime)
                         └── WaitForExternalEvent("approval") ← for gates
```

| Attribute | Score | Notes |
|-----------|-------|-------|
| Runtime | C# only | Durable Functions SDK |
| Blast radius | ⭐⭐⭐⭐ | Framework-managed isolation |
| Isolation | ⭐⭐⭐ | Shared function app host |
| Cold start | ⭐⭐⭐ | Same as B2 |
| Cost | ⭐⭐⭐ | Storage transactions add cost |
| Approval gates | ⭐⭐⭐⭐⭐ | **Native.** `WaitForExternalEvent` is purpose-built for this |
| Governance | ⭐⭐⭐⭐ | Safe IF the orchestrator calls the full run as ONE activity (not per-step) |
| Complexity | ⭐⭐⭐ | Durable Functions learning curve, replay semantics |

**CRITICAL CONSTRAINT (Don's Red Line 1):**
The orchestrator MUST call the entire GERT run as a single activity. It must NOT decompose the run into per-step activities. Per-step orchestration violates governance Red Lines 1, 2, and 3.

**Valid pattern:**
```csharp
[Function("RunOrchestrator")]
public async Task Run([OrchestrationTrigger] TaskOrchestrationContext ctx)
{
    var runId = ctx.GetInput<string>();
    // Execute entire run as one activity (governance preserved)
    var result = await ctx.CallActivityAsync<RunResult>("ExecuteRun", runId);
    
    if (result.Status == RunStatus.AwaitingApproval)
    {
        // Native approval gate — can wait hours/days
        var approval = await ctx.WaitForExternalEvent<ApprovalDecision>("approval");
        // Resume run with approval result
        result = await ctx.CallActivityAsync<RunResult>("ResumeRun", (runId, approval));
    }
}
```

**Verdict:** Interesting for approval-heavy workflows. But adds complexity (Durable SDK, storage accounts, replay semantics) that B1 avoids with simple `await`. Consider as upgrade path if approval gates become the dominant pattern.

---

#### B4: App Service Per-Tenant (Dedicated Worker)

**Architecture:**
```
[Service Bus Queue (per-tenant session)] → [App Service (per-tenant, Always On)]
                                              └── Dedicated BackgroundService for one tenant
```

| Attribute | Score | Notes |
|-----------|-------|-------|
| Runtime | C# only | Same as B1, one instance per tenant |
| Blast radius | ⭐⭐⭐⭐⭐ | Per-tenant. One tenant's failure cannot affect another |
| Isolation | ⭐⭐⭐⭐⭐ | OS-level process isolation between tenants |
| Cold start | ⭐⭐⭐⭐⭐ | Always warm (Always On) |
| Cost | ⭐⭐ | ~$55/mo per tenant (B1 plan). Expensive at scale |
| Approval gates | ⭐⭐⭐⭐⭐ | Same as B1 — trivial await |
| Governance | ⭐⭐⭐⭐⭐ | Per-tenant process = same isolation as Go binary per-run |
| Scaling | ⭐⭐⭐ | Vertical only per tenant. Add instances for more tenants |

**Verdict:** Enterprise endgame (C# equivalent of A4). Simpler than Container Apps sidecar — no container orchestration needed. Use App Service Environments (ASE) for network isolation if required.

---

## Revised Recommendations

### MVP Path: B1 (App Service + BackgroundService)

**Why B1 over A1 (previous recommendation):**

| Dimension | A1 (Thin Relay / Jobs) | B1 (App Service + BGService) |
|-----------|----------------------|------------------------------|
| Cold start | 3–8s | **0s** |
| Approval gates | Hard (checkpoint/resume) | **Trivial (await)** |
| Deployment complexity | Functions + Container Apps + ACR | **Single App Service** |
| Cost (MVP) | ~$0–30/mo (usage) | **~$55/mo (fixed)** |
| Blast radius | ⭐⭐⭐⭐⭐ (per-run) | ⭐⭐⭐ (per-instance) |
| Time to production | 4–6 weeks | **2–3 weeks** |

**The tradeoff:** B1 trades per-run blast radius isolation for dramatically simpler approval gates, zero cold start, and faster time-to-market. For an MVP with <100 concurrent runs, the blast radius risk is acceptable with proper mitigation (per-run error isolation, concurrency caps, health checks).

### Enterprise Path: B4 (App Service Per-Tenant)

**Why B4 over A4 (previous recommendation):**
- Same isolation guarantees (per-tenant dedicated compute)
- No container image management, no ACR, no Container Apps complexity
- Standard Azure App Service operations (familiar to most .NET teams)
- Slot deployments for zero-downtime updates per tenant
- If network isolation needed: App Service Environment (ASE)

### Migration Path

```
Phase 1 (MVP):     B1 — Single App Service, shared process, all tenants
Phase 2 (Growth):  B1 scaled — Multiple instances (auto-scale), still shared
Phase 3 (Premium): B4 — Premium tenants get dedicated App Service
Phase 4 (Enterprise): B4 + ASE — Network-isolated, dedicated compute
```

---

## Risk Register

| Risk | Severity | Mitigation |
|------|----------|-----------|
| B1: Noisy neighbor (one run starves others) | Medium | SemaphoreSlim concurrency cap, per-run memory ceiling, health check restart |
| B1: Process crash loses all in-flight runs | Medium | Runs re-queue from Service Bus (PeekLock, not auto-complete). At-most-once via distributed lock on run_id |
| B1: Governance bypass (C# runtime in shared process) | Low | Runtime class is `internal sealed`, no public API surface. Code review gate on Runtime namespace |
| B4: Cost at scale (many tenants) | Medium | Tier tenants: free/shared on B1, premium on B4. Only enterprise customers get dedicated |
| C# runtime parity with Go | High | Must port ALL governance enforcement. Don must validate. No shortcuts |

---

## Decision Required

1. **Approve B1 as new MVP topology** (replaces A1 Thin Relay)
2. **Approve B4 as enterprise endgame** (replaces A4 Sidecar)
3. **Retire A2** (superseded by B2)
4. **Confirm Don will validate** C# runtime governance parity with Go binary
5. **Confirm approval gate strategy:** simple `await` in BackgroundService (B1) vs Durable Functions (B3)

---

## Appendix: Full Option Matrix

| Option | Runtime | Blast Radius | Cold Start | Isolation | Cost | Approval Gates | MVP? | Enterprise? |
|--------|---------|-------------|-----------|-----------|------|---------------|------|-------------|
| A1: Thin Relay (Jobs) | Go/C# | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ | ~~Yes~~ → No | No |
| A2: Embedded Worker | Go | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐ | ❌ Retired | No |
| A3: Persistent Pool | Go/C# | ⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐ | No | No |
| A4: Sidecar Agent | Go/C# | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐⭐ | No | ~~Yes~~ → B4 |
| A5: Hybrid Dispatch | Go/C# | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | No | No |
| **B1: App Service + BGService** | **C#** | **⭐⭐⭐** | **⭐⭐⭐⭐⭐** | **⭐⭐⭐** | **⭐⭐⭐⭐⭐** | **⭐⭐⭐⭐⭐** | **✅ YES** | No |
| B2: Functions Isolated | C# | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐ | No | No |
| B3: Durable Functions | C# | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | No | No |
| **B4: App Service Per-Tenant** | **C#** | **⭐⭐⭐⭐⭐** | **⭐⭐⭐⭐⭐** | **⭐⭐⭐⭐⭐** | **⭐⭐** | **⭐⭐⭐⭐⭐** | No | **✅ YES** |
# Revised Execution Adapter Patterns: Go Binary vs. C# Reimplementation

**Author:** Don (Backend Dev)  
**Date:** 2026-06-03  
**Status:** Proposal — requires team review before adoption  
**Trigger:** User confirmed they are NOT attached to the Go binary for web execution. Open to writing a C# runtime that reimplements GERT governance natively.

---

## Background & Constraint Change

Previous analysis assumed the Go binary was fixed and must be hosted as-is. All patterns were evaluated through that lens. The Go binary constraint is now lifted for the web backend. The C# option means the governance engine itself is rewritten — it is not a wrapper calling the Go binary.

This is a significant unlock. Re-evaluated all patterns below.

---

## What Changes with a C# Runtime

When C# **is** the governance engine (not a wrapper):

- `RunHandle.Next()` is a C# method — the governance loop is native C# code
- The JSONL trace is written by a C# `IRunStore` implementation
- Approval gates are native `Task` continuations or `WaitForExternalEvent` in Durable
- Tool subprocess invocation is native `System.Diagnostics.Process`
- Durable Functions orchestrators can legitimately implement `Next()` semantics because the orchestrator IS the governance engine, not a bypass of it

What does **not** change:

- Red Line 1 still applies in spirit: `IRunHandle.NextAsync()` must fire governance before each step dispatch, regardless of language
- Red Line 2 still applies: the JSONL trace must be written only by `IRunStore.WriteTraceAsync()`, not by any external process or orchestration layer
- Red Line 3 still applies: exactly one live `IRunHandle` per `run_id` at a time
- Red Lines 4 & 5 still apply: `client: "web"` in every run, approval gates require pause/resume protocol

---

## C# Runtime Structural Design

Before evaluating patterns, here is what a C# GERT runtime looks like structurally, mirroring the Go interfaces exactly:

```csharp
// Mirrors Go Parser interface
public interface IParser
{
    Task<ParsedRunbook> ParseAsync(Stream yaml, ParseOptions opts, CancellationToken ct = default);
}

// Mirrors Go Planner interface
public interface IPlanner
{
    Task<ExecutionPlan> PlanAsync(ParsedRunbook rb, PlanOptions opts, CancellationToken ct = default);
}

// Mirrors Go Runtime interface
public interface IGertRuntime
{
    Task<IRunHandle> StartAsync(ExecutionPlan plan, RunOptions opts, CancellationToken ct = default);
    Task<IRunHandle> ResumeAsync(string runId, RunOptions opts, CancellationToken ct = default);
}

// THE governance boundary — mirrors Go RunHandle exactly
public interface IRunHandle
{
    string RunId { get; }
    Task<StepResult?> NextAsync(CancellationToken ct = default); // null == io.EOF
    Task ApproveAsync(ApprovalDecision decision, CancellationToken ct = default);
    Task SubmitEvidenceAsync(string stepId, Dictionary<string, EvidenceValue> ev, CancellationToken ct = default);
    Task CancelAsync(string reason, CancellationToken ct = default);
    RunState State { get; }
    IAsyncEnumerable<GertEvent> Events { get; }
}

// Governance layer — called inside NextAsync before step dispatch
public interface IGovernanceLayer
{
    Task<CommandCheckResult> CheckCommandAsync(string command, RunContext ctx, CancellationToken ct = default);
    Task<EnvVarCheckResult> FilterEnvVarsAsync(IReadOnlyDictionary<string, string> env, RunContext ctx, CancellationToken ct = default);
    Task<string> RedactOutputAsync(string output, RunContext ctx, CancellationToken ct = default);
    Task<ApprovalGateResult> EvaluateApprovalGateAsync(ResolvedStep step, RunContext ctx, CancellationToken ct = default);
}

// RunStore — owns the JSONL trace. WriteTraceAsync is the authoritative record.
public interface IRunStore
{
    Task WriteTraceAsync(GertEvent ev, CancellationToken ct = default); // must flush before returning
    Task SaveCheckpointAsync(string runId, int stepIndex, RunState state, CancellationToken ct = default);
    Task<RunState?> LoadCheckpointAsync(string runId, CancellationToken ct = default);
}
```

The `NextAsync()` execution loop in C#:

```csharp
// Pseudocode — the governance loop, same semantics as Go
public async Task<StepResult?> NextAsync(CancellationToken ct)
{
    if (_stepIndex >= _plan.Steps.Count) return null; // EOF

    var step = _plan.Steps[_stepIndex];

    // 1. Emit step/started
    await _store.WriteTraceAsync(new GertEvent { Kind = "step/started", Sequence = NextSeq(), ... }, ct);

    // 2. Governance pre-flight (MUST happen before any side effect)
    if (step.Kind is StepKind.Cli or StepKind.Tool)
    {
        var cmdCheck = await _governance.CheckCommandAsync(step.Command, _ctx, ct);
        if (!cmdCheck.Allowed)
        {
            await _store.WriteTraceAsync(new GertEvent { Kind = "governance/commandBlocked", ... }, ct);
            throw new GovernanceException(cmdCheck.Reason);
        }
        // env var filtering, redaction setup...
    }

    // 3. Approval gate (blocks until approved — hours/days acceptable)
    if (step.RequiresApproval)
    {
        await _store.WriteTraceAsync(new GertEvent { Kind = "gov/awaiting_approval", ... }, ct);
        await _approvalGate.WaitForApprovalAsync(step.Id, ct); // awaits DB or message signal
        await _store.WriteTraceAsync(new GertEvent { Kind = "gov/approved", ... }, ct);
    }

    // 4. Execute step, capture output
    var result = await _stepRunner.RunAsync(step, _varStore, ct);

    // 5. Redact output BEFORE storing in variables or trace
    result = await _governance.RedactAsync(result, _ctx, ct);

    // 6. Emit step/completed
    await _store.WriteTraceAsync(new GertEvent { Kind = "step/completed", ... }, ct);

    // 7. Checkpoint
    await _store.SaveCheckpointAsync(RunId, _stepIndex, State, ct);

    _stepIndex++;
    return result;
}
```

Does C# async/await translate cleanly to this governance model? **Yes.** The sequential `await` chain preserves the total ordering invariant. The governance pre-flight fires before any side effect. `IAsyncEnumerable<GertEvent>` replaces the Go event channel. The only structural difference is that C# exceptions are used where Go returns `(result, error)` tuples.

---

## Pattern Evaluation Matrix

### Pattern A: Queue-triggered Worker — Container App Job

**Description:** Service Bus queue triggers a worker. Worker owns the full runtime loop. One job per run.

| Dimension | Go Binary | C# Runtime |
|---|---|---|
| Governance preservation | ✅ Complete — Go binary is the governance engine | ✅ Complete — C# IRunHandle is the governance engine |
| Hosting | Container App Job, ephemeral per-run | Container App Job OR BackgroundService in Container App |
| Approval gates | Checkpoint → reload from RunStore on resume signal | Same pattern. `await _approvalGate.WaitForApprovalAsync()` blocks naturally in C# |
| Trace writer | Go `DirRunStore.WriteTrace` (fsync) | C# `BlobRunStore.WriteTraceAsync` (flush-on-write to Azure Blob append block) |
| Cold start | 3–8s (container pull if not warm) | Same — JIT adds ~100–300ms on first invocation |
| Blast radius | Per-run (one container per run) | Per-run — same |
| Verdict | ✅ **Recommended baseline (Go)** | ✅ **Recommended baseline (C#)** |

**Notes (C#):** `BackgroundService` can host the runtime loop inside a Container App (non-ephemeral), processing runs from a queue via `ServiceBusProcessor`. This eliminates cold start at the cost of a persistent process. Scale via KEDA queue-length trigger.

---

### Pattern B: HTTP-triggered Container App (Synchronous)

**Description:** HTTP POST → Container App → run synchronously → return result.

| Dimension | Go Binary | C# Runtime |
|---|---|---|
| Governance preservation | ✅ Complete | ✅ Complete |
| Timeout constraint | Azure Load Balancer: 230s hard limit | Same — ASP.NET Core's `HttpContext.RequestAborted` gives cleaner cancellation |
| Approval gates | INCOMPATIBLE without long-poll extension | INCOMPATIBLE for same reason — HTTP cannot await hours |
| Streaming output | SSE/chunked transfer | ASP.NET Core SSE (`IAsyncEnumerable` → `text/event-stream`) is first-class |
| Verdict | ⚠️ Short-running runbooks only | ⚠️ Short-running runbooks only; C# has marginally better SSE ergonomics |

---

### Pattern C: Durable Functions Orchestrator — Whole Run

**Description:** Service Bus trigger → Durable orchestrator → activities execute steps → approval gates as external events.

| Dimension | Go Binary | C# Runtime |
|---|---|---|
| **Governance preservation** | 🚨 HIGH RISK — orchestrator calls Go binary per activity. Governance pre-flight must happen inside the binary, but the orchestrator controls execution order. Replay semantics mean the binary could be called with wrong inputs on re-execution. | ✅ **NOW VIABLE** — orchestrator IS the IRunHandle. Governance fires in the orchestrator before each activity. |
| **Durable replay safety** | N/A | ⚠️ Orchestrator code is replayed on every wake-up. All governance checks that run inline (not in activities) must be **deterministic and produce the same result on replay**. Governance checks that have external side effects (DB lookups, policy API calls) must be wrapped in activities. |
| Approval gates | N/A | ✅ Native: `context.WaitForExternalEvent<ApprovalEvent>("ApprovalReceived")`. Pauses are free — the orchestrator sleeps until the event arrives. No worker needs to be alive. |
| JSONL trace ordering | N/A | ⚠️ WriteTrace must be called in activities (not inline orchestrator code) to ensure it is NOT re-executed on replay. Activity results are cached in Durable history, so trace is written exactly once per event. |
| Sequence numbers | N/A | Sequence counter must live in orchestrator state (custom status or output object) and be incremented only via activity results — never inline. |
| Fan-out / parallel steps | N/A | ⚠️ GERT steps are sequential by design. Do NOT use `Task.WhenAll` on steps — it breaks total ordering invariant and trace sequence. |
| Cost | N/A | Higher than queue worker: Durable history storage + orchestration overhead. Acceptable for long-running runbooks with approval gates. |
| Verdict | 🚨 RED — governance bypass risk | ✅ **VIABLE for long-running runbooks with approval gates.** Most natural fit for the pause/resume model. |

**Critical implementation constraint for C# Durable pattern:**

```csharp
// WRONG: governance inline in orchestrator — replay re-evaluates this
var cmdCheck = _governance.CheckCommand(step.Command, ctx); // determinism NOT guaranteed on replay

// CORRECT: governance as activity — result cached, not re-executed on replay
var cmdCheck = await context.CallActivityAsync<CommandCheckResult>("CheckCommand", step);
```

The governance layer itself (allowlist eval, denylist eval, env var filtering, redaction) must be implemented as deterministic, idempotent activities if Durable is used.

---

### Pattern D: Durable Entity as RunHandle State — New Option (C# Only)

**Description:** A Durable Entity holds the mutable run state (step index, variable store, governance state). An orchestrator (or direct entity client) drives it via operations.

| Dimension | C# Runtime |
|---|---|
| Concept | Entity = `IRunHandle` state. Each entity operation = one `Next()` call. |
| Governance | Governance layer is called within entity operations — sequential by design (entities are single-threaded). |
| Approval gates | Entity waits for a `Signal` from the approval API. Entity can sleep indefinitely. |
| Trace ordering | ✅ Entity operations are serialized — sequence numbers are trivially monotonic. |
| State durability | ✅ Entity state is persisted by Durable runtime after each operation. Crash safety built-in. |
| Complexity | HIGH — entity + orchestrator interaction is more complex than a simple queue worker. |
| When to use | Approval-heavy runbooks where pause duration is days. Interactive wizard-style runbooks. |
| Verdict | ✅ **Viable for interactive/approval-heavy runbooks. Not recommended as default pattern.** |

```csharp
[FunctionName("RunHandleEntity")]
public static async Task RunHandleEntity([EntityTrigger] IDurableEntityContext ctx)
{
    switch (ctx.OperationName)
    {
        case "Next":
            var run = ctx.GetState<RunEntityState>();
            // governance pre-flight, step execution, trace write — all sequential
            ctx.SetState(run);
            ctx.Return(result);
            break;
        case "Approve":
            var run = ctx.GetState<RunEntityState>();
            run.PendingApproval = false;
            ctx.SetState(run);
            break;
    }
}
```

---

### Pattern E: ASP.NET Core BackgroundService — Container App

**Description:** `IHostedService` + `BackgroundService` implementation hosts the GERT runtime loop in a Container App. Processes runs from a Service Bus queue. No Durable Functions involved.

| Dimension | Go Binary | C# Runtime |
|---|---|---|
| Concept | Similar to Persistent Agent Pool (Go) | BackgroundService with ServiceBusProcessor, one run at a time per replica |
| Governance | ✅ Identical to Pattern A — IRunHandle owns the loop | ✅ Same |
| Approval gates | Checkpoint → resume on queue message | ✅ Native: `await approvalChannel.WaitAsync(ct)` — BackgroundService stays alive |
| Multi-run concurrency | One binary per run (Container Job) | Configurable: `SemaphoreSlim` limits concurrent runs per replica |
| Blast radius | Per-run (Go: separate binary) | ⚠️ Per-replica if concurrent runs enabled — one exception can crash all in-flight runs on that replica. Mitigation: one run per replica, KEDA scale-out. |
| Cold start | Per-container start | JIT warm after first run — subsequent runs on same replica are cold-start-free |
| Cost | Ephemeral per-run (Container Job) | Persistent replica — lower per-run cost if volume is high |
| Verdict | ✅ Viable (= Pattern A) | ✅ **Strong option for high-throughput scenarios. Simplest C# pattern.** |

```csharp
public class GertWorkerService(
    ServiceBusClient bus,
    IGertRuntime runtime,
    IRunStore store) : BackgroundService
{
    protected override async Task ExecuteAsync(CancellationToken ct)
    {
        var processor = bus.CreateProcessor("run-requests");
        processor.ProcessMessageAsync += async args =>
        {
            var request = args.Message.Body.ToObjectFromJson<RunRequest>();
            var plan = await BuildPlanAsync(request, ct);
            var handle = await runtime.StartAsync(plan, new RunOptions { Client = "web" }, ct);
            while (await handle.NextAsync(ct) is not null) { }
            await args.CompleteMessageAsync(args.Message, ct);
        };
        await processor.StartProcessingAsync(ct);
        await Task.Delay(Timeout.Infinite, ct);
    }
}
```

---

### Pattern F: Azure Functions Isolated Worker (.NET 8) — Service Bus Trigger

**Description:** Service Bus trigger fires a Function. Function hosts one run. Function exits when run completes.

| Dimension | Go Binary | C# Runtime |
|---|---|---|
| Governance | ✅ Complete (binary owns it) | ✅ Complete (IRunHandle owns it) |
| Time limit | 10-min consumption plan hard limit | Same — Premium plan extends to 60 min |
| Approval gates | ❌ Process exits before approval | ❌ Function exits before approval — requires external coordination to resume (a new Function invocation with checkpoint restore) |
| Short runbooks | ✅ Works well | ✅ Works well — lowest cost for short, frequent runs |
| Long runbooks | ❌ Incompatible | ❌ Incompatible without Durable |
| Verdict | ⚠️ Short runs only | ⚠️ Short runs only. Consider Pattern C (Durable) for approval gate support. |

---

### Pattern G: Per-Step Durable Function Activities — Explicitly Evaluated

**Description:** Orchestrator fires one Durable Activity per step. Each activity independently invokes the Go binary (or C# step runner).

| Dimension | Go Binary | C# Runtime |
|---|---|---|
| Governance preservation | 🚨 **RED LINE** — each activity cold-starts the binary, losing in-process governance state. Governance pre-flight is not guaranteed to fire before each step. Trace is split across activity invocations. | 🚨 **RED LINE** — even in C#, if the governance layer is NOT present in the activity (i.e., activity is a pure step runner without governance), this pattern bypasses IGovernanceLayer entirely. |
| Why it's tempting | Fan-out retry, activity-level observability | Same |
| Why it's wrong | Activity = step = governance bypass. Not fixable without embedding the full runtime in each activity (at which point you have Pattern C, not G). | Same — unless each activity contains the full governance loop, which defeats the purpose. |
| Verdict | 🚨 **RED LINE — prohibited** | 🚨 **RED LINE — prohibited** |

Note: The C# version of this red line is more subtle. A developer might implement the step activity to call `IStepRunner.RunAsync(step)` directly, skipping `IRunHandle.NextAsync()`. This must be prevented by design — `IStepRunner` must not be a public interface accessible outside `IRunHandle`. Enforce this with `internal` visibility.

---

## Summary: Recommended Pattern Hierarchy

| Priority | Pattern | Language | When to Use |
|---|---|---|---|
| 1 | Queue Worker (BackgroundService) | C# | Default path. High volume, Container App + KEDA. Simple, testable, governance-complete. |
| 2 | Queue Worker (Container App Job) | Go or C# | Strong isolation per run. Slightly higher cold-start cost. Use when blast radius must be zero. |
| 3 | Durable Functions Orchestrator | C# | Long-running runbooks with approval gates. Most natural pause/resume model. Higher complexity. |
| 4 | Durable Entity as RunHandle | C# | Interactive wizard-style runbooks with complex pause semantics. Only when Pattern 3 is insufficient. |
| 5 | HTTP Container App | Go or C# | Short, synchronous runbooks where caller can block. Dev/test or internal tools only. |
| 6 | Functions Isolated Worker | C# | Short-running, frequent, time-bounded runbooks. Budget-sensitive workloads. |

---

## Risks of C# Reimplementation

### Risk 1: Template Evaluation Semantic Drift — HIGH

GERT uses Go `text/template` for step spec evaluation. C# has no exact equivalent. Any C# template engine (Scriban, Liquid, Handlebars.NET) will have different edge-case behavior. The risk: a runbook that produces output `X` in Go produces output `Y` in C#, causing trace divergence.

**Mitigation:** Define and test a canonical set of template test vectors. Run both runtimes against them. If vectors don't match, the C# runtime is not production-safe. Consider: only allow a safe subset of Go template syntax (variable substitution, basic conditionals) and explicitly reject advanced features that are hard to port.

### Risk 2: Regex Engine Divergence in Redaction — HIGH (Governance Security Risk)

Go uses RE2. C# .NET uses a backtracking regex engine that is a strict superset of RE2. A pattern that compiles and matches in .NET may not be a valid RE2 pattern (and vice versa). Critically: a redaction rule that blocks a secret in Go may fail to compile in C# .NET, allowing the secret to appear in the trace.

**Mitigation:** Use the `RE2` NuGet package (Google.RE2) in C# to enforce RE2 semantics. Do not use `System.Text.RegularExpressions` for governance redaction. This is non-negotiable.

### Risk 3: Governance Logic Drift — HIGH

The Go governance layer has been developed and tested against the GERT governance spec. A C# port introduces a fresh surface for bugs. A missed edge case in allowlist evaluation, denylist precedence, or env var glob matching is a security defect, not a functional bug.

**Mitigation:** Extract the governance specification (allowlist semantics, denylist precedence, env var glob matching rules) into a language-neutral test suite (JSON test vectors). Run both runtimes against it. The Go runtime is the reference implementation — if C# differs, C# is wrong.

### Risk 4: JSONL Trace Contract Drift — MEDIUM

The trace schema is defined in the GERT spec. Any field name difference, timestamp format difference, sequence numbering difference, or missing mandatory event breaks cross-runtime trace compatibility. Replay mode reading a C#-generated trace in the Go TUI (or vice versa) will fail silently.

**Mitigation:** Define the JSONL event schema as a JSON Schema artifact (same as the Go runtime uses for the runbook schema). Validate C# trace output against it in integration tests. Run the Go replay engine against C#-generated traces in CI.

### Risk 5: Durable Functions Replay Non-Determinism — MEDIUM

If Pattern C (Durable Orchestrator) is used, non-deterministic code in the orchestrator (DateTime.Now, Guid.NewGuid(), non-deterministic awaits) will cause replay failures. The Durable runtime will detect replays and throw `NonDeterministicOrchestrationException`.

**Mitigation:** Follow Durable best practices strictly. Use `context.CurrentUtcDateTime` instead of `DateTime.UtcNow`. Use `context.NewGuid()` instead of `Guid.NewGuid()`. Never call async operations (HttpClient, DB) directly in the orchestrator — wrap in activities.

### Risk 6: Feature Parity Gap Over Time — LOW (but compounding)

Any new governance feature added to the Go runtime must be simultaneously ported to C#, or the runtimes diverge. Two runtimes = two maintenance surfaces.

**Mitigation:** Adopt a spec-first development process. New governance features are specified first (in the design docs), then implemented in both runtimes simultaneously. Consider whether the C# runtime is a long-term commitment or a temporary web-only path that will be retired when the Go runtime gains a proper web hosting mode.

---

## Recommendation

The C# option is credible and unlocks patterns that were red lines under the Go constraint (specifically: Durable Orchestrators for approval-gate-heavy runbooks). However, the implementation risks — especially template evaluation drift, regex engine divergence, and governance logic drift — require a rigorous test vector strategy before the C# runtime can be trusted with production governance.

**Recommended sequence:**
1. Build Pattern E (BackgroundService + Container App) as the MVP path. It is the simplest translation of the existing queue-worker pattern, with no Durable complexity.
2. Define governance test vectors and JSONL schema validation before writing any C# governance code.
3. Adopt RE2 NuGet for all redaction regex evaluation — non-negotiable.
4. Consider Pattern C (Durable Orchestrator) only after the BackgroundService version is stable and governance tests pass.
5. Revisit Pattern D (Durable Entity) only if approval gates are a core product feature with documented SLA requirements (e.g., runbooks that pause for days with guaranteed resume).

**Open questions for team:**
1. Is the C# runtime a permanent commitment or a stepping stone until the Go runtime supports native web hosting?
2. Can the governance test vector suite be owned by the design doc team (Barbara?) as a language-neutral artifact?
3. If we ship C# and Go runtimes simultaneously, who owns the parity gate — integration tests or manual sign-off?
4. Does the RE2 NuGet package (if used) have an acceptable license for production use in this project?
# Revised Azure Service Stack — C# Runtime Edition

**Author:** John (Azure Platform Engineer)  
**Date:** 2026-06-03  
**Status:** Inbox — supersedes previous compute recommendation  
**Trigger:** Runtime language changed from Go to C# (.NET 8 Isolated / ASP.NET Core)

---

## Headline Change

The Go-binary assumption previously made Container Apps the only sensible compute host. C# opens four credible alternatives. The **approval gate problem** (hours-long pause waiting for human input) is now the primary discriminator between options — it is the one constraint that makes Durable Functions structurally superior to every other tier.

**New primary recommendation:** Durable Functions on Functions Flex Consumption (unbounded timeout, pay-per-use, approval gate hibernation native, 1,000-instance scale).  
**Fallback if Flex Consumption not GA/stable:** Durable Functions on Functions Premium EP1.

Container Apps remains the right answer for the Thin Relay (ephemeral per-run job) topology, but is no longer the default for persistent worker scenarios.

---

## 1. Compute Options — Full Re-Evaluation

### 1A. App Service (B2 / B3 / P1v3 / P2v3)

| Tier | vCPU | RAM | Cost/mo | Scale-out | Slots | VNet | Notes |
|------|------|-----|---------|-----------|-------|------|-------|
| B2 | 2 | 3.5 GB | ~$61 | **3 max** | ❌ | ❌ | Dev only — 3-instance ceiling is a hard stop |
| B3 | 4 | 7 GB | ~$123 | **3 max** | ❌ | ❌ | Dev only — same ceiling |
| P1v3 | 2 | 8 GB | ~$124 | 30 | ✅ (20) | ✅ | Production baseline |
| P2v3 | 4 | 16 GB | ~$248 | 30 | ✅ (20) | ✅ | Enterprise / memory-heavy runbooks |
| P3v3 | 8 | 32 GB | ~$496 | 30 | ✅ (20) | ✅ | Only if runbook parallelism is memory-bound |

**Execution time limit:** None. App Service does not time out long-running threads.  
**Background worker pattern:** `IHostedService` + `BackgroundService` in the same ASP.NET Core host → zero additional compute cost. Worker pulls from Service Bus naturally load-balanced across scale-out instances.

**The case FOR App Service:**
- Single plan hosts both the HTTP API and the queue consumer — no second compute resource needed.
- Deployment slots (P1v3+) enable blue/green with instant traffic swap.
- Predictable flat cost — easy to budget per-tenant on shared plans.
- `IHostedService` is idiomatic C# — no framework lock-in.

**The case AGAINST App Service:**
- B2/B3: 3-instance ceiling is unacceptable for production multi-tenant load.
- No scale-to-zero — P1v3 costs $124/mo whether idle or not.
- Approval gate requires explicit checkpoint (Service Bus deferred message + database row). Not free.
- Scale-out to 30 is a hard ceiling; Container Apps and Flex Consumption blow past it.
- `IHostedService` copies run on every instance — requires Service Bus pull (naturally serialized) or distributed locking on run_id.

**Verdict:** Best for **MVP / dev environments** (shared P1v3 for API + worker, single plan, simple billing). Not the endgame for multi-tenant production.

**Per-tenant cost model:**
- Shared plan (all tenants): ~$124/mo total. Cheapest. Shared blast radius.
- Per-tenant plan: $124+/tenant/mo. Clean isolation. Only viable at enterprise pricing tier.

---

### 1B. Functions Premium (EP1 / EP2 / EP3)

| SKU | vCPU | RAM | Min cost/mo | Scale-out | VNet | Durable |
|-----|------|-----|------------|-----------|------|---------|
| EP1 | 1 | 3.5 GB | ~$88–173 | 100 | ✅ | ✅ |
| EP2 | 2 | 7 GB | ~$174+ | 100 | ✅ | ✅ |
| EP3 | 4 | 14 GB | ~$348+ | 100 | ✅ | ✅ |

**Execution time limit:** Configurable. Default 30 min; set `functionTimeout: "00:00:00"` in host.json = **unlimited**.  
**Cold start:** None — always-warm pre-provisioned instances.  
**Elastic scale:** Adds instances dynamically above the baseline; billed per-second for added instances.

**The case FOR Premium:**
- Mature, battle-tested. GA since 2019.
- Durable Functions runs natively — approval gate hibernation via `WaitForExternalEvent`.
- KEDA-compatible Service Bus trigger out of the box.
- Slots available for blue/green.
- VNet integration fully supported (for private Service Bus, private storage).

**The case AGAINST Premium:**
- Always-on baseline cost (~$88/mo minimum for EP1) even with zero traffic.
- More expensive than Flex Consumption at low-to-medium volume.
- Scale-out ceiling of 100 instances (vs 1,000 on Flex Consumption).

**Verdict:** **Strong fallback** if Flex Consumption has stability concerns. Best current option for Durable Functions if you need battle-tested infrastructure with known quotas.

---

### 1C. Functions Flex Consumption ⭐ NEW TIER

| Attribute | Value |
|-----------|-------|
| Execution timeout | **Unlimited** (`functionTimeout: "00:00:00"`) |
| Default timeout | 30 minutes |
| Scale-out | **Up to 1,000 instances** (5× Premium, 33× App Service) |
| Cold start | Improved (pre-warmed pool; not zero, but faster than standard Consumption) |
| Billing | **Per-second** (like Consumption) — no always-on baseline cost |
| VNet integration | ✅ Yes (newer, verify regional availability) |
| Durable Functions | ✅ Yes |
| Deployment slots | ⚠️ Limited/evolving — verify GA status |
| .NET 8 Isolated | ✅ Native |

**What changed from Consumption (Y1):** The 10-min hard cap is **gone**. Flex Consumption allows unlimited execution time — this removes the only red line that existed.

**The case FOR Flex Consumption:**
- No baseline cost — pays only for actual execution. Zero cost at zero load.
- Unlimited timeout eliminates the runbook duration concern entirely.
- 1,000-instance scale-out is headroom no other tier matches.
- Durable Functions + `WaitForExternalEvent` for approval gates — no infrastructure cost during wait (orchestrator hibernates, nothing runs, nothing is billed).
- C# .NET 8 is the primary supported language — first-class citizen.

**The case AGAINST / Watch items:**
- Newer tier — watch for feature gaps vs Premium (slots, Durable configuration options).
- VNet integration maturity should be validated for private Service Bus.
- Pre-warmed instances reduce but don't eliminate cold start for very bursty workloads.
- Durable Functions history storage is still Azure Storage Tables — same constraints apply (see §1D).

**Quota ask before committing:**
- Confirm 1,000-instance limit is per-app or per-subscription.
- Confirm VNet integration is available in target region.
- Confirm Durable Functions task hub isolation is supported.

**Verdict:** **Primary compute recommendation for C# GERT**. Flex Consumption + Durable Functions is the cleanest architecture: pay-per-use, no timeout, approval gate hibernation native, 1,000-instance headroom.

---

### 1D. Durable Functions (Compute-agnostic, runs on EP1 or Flex Consumption)

Durable Functions is not a compute tier — it is a programming model that runs ON top of Premium or Flex Consumption. Evaluated here because it changes the architecture significantly.

**Governance alignment (Don Trujillo's red lines):**
- ✅ **Whole-run orchestrator**: Single Orchestrator function wraps the entire GERT runtime execution. The C# GERT runtime runs as a single Activity. Red Lines 1, 2, 3 are all preserved — runtime controls its own lifecycle.
- 🚨 **Per-step orchestrator**: NOT permitted per Red Line 1 (bypasses `Runtime.Start → RunHandle.Next`). Do not implement per-step Durable.

**Approval gate — the killer feature:**
```csharp
// Orchestrator hibernates here. Zero compute. Zero cost. Wakes on external event.
var approval = await context.WaitForExternalEvent<ApprovalResult>("runbook-approval", timeout: TimeSpan.FromDays(30));
```
The orchestrator checkpoints its state to Azure Storage, deallocates all compute, and waits. No keepalive loop needed. No redis lock. No polling. This is the most elegant solution to GERT's approval gate problem.

**History size quotas:**
- Azure Storage Table entity: 1 MB max per row
- Orchestration history: one row per event (step completion, external event, timer)
- For a 50-step runbook with modest outputs: safe
- For runbooks with large step outputs (>100KB each): **blob externalization required**
  - Configure `extendedSessionsEnabled` + store large payloads in Blob, keep reference in history
  - Durable v2 supports this natively via `IMessageSerializerSettingsFactory`
- No published hard limit on history record count, but practical degradation starts around thousands of events

**Orchestration duration:** No hard limit. Orchestrations can run for days, months, indefinitely.

**Task hub isolation for multi-tenancy:**
- Each tenant can get a named task hub (separate Storage queues/tables)
- Provides logical isolation without separate infrastructure
- Shared storage account = shared throughput limit (20,000 transactions/sec on Standard)
- For high-volume tenants: separate Storage accounts per tenant

**Verdict:** ✅ VIABLE and recommended for GERT. Approval gate hibernation alone justifies the choice. Must run on EP1 or Flex Consumption (not Consumption Y1, which has the 10-min activity cap). Implement as whole-run orchestrator only.

---

### 1E. Container Apps — Re-Scored for C#

| Attribute | Go | C# | Delta |
|-----------|----|----|-------|
| Image size | ~10 MB (static binary) | ~80 MB (.NET 8 alpine) | +70 MB — negligible at runtime |
| Cold start | ~0.1s process start | ~2–3s .NET runtime init | C# slower cold start |
| Startup cost | Minimal | Higher | Go wins for pure cold start |
| Azure SDK | Manual HTTP/gRPC | First-class SDK | C# wins for integration |
| Cost (vCPU-sec) | $0.000024 | $0.000024 | Same |
| 24/7 equivalent | ~$63/mo (1 vCPU) | ~$63/mo (1 vCPU) | Same |

**Free grants:** 180,000 vCPU-seconds and 360,000 GiB-seconds per month per subscription.

**C# does not meaningfully change the Container Apps cost model.** The runtime overhead of .NET 8 is ~80-120ms of additional initialization vs a Go binary, which is immaterial for queue-triggered workloads.

**Container Apps Jobs (Barbara's Thin Relay — ephemeral per-run isolation):**
- Still the best option for per-run isolation with zero residual blast radius
- Each job is a fresh container, exits when run completes
- Approval gate: **checkpoint + terminate** required (no native hibernation)
  - Worker serializes run state to Blob or Cosmos, exits
  - On approval: new Job spins up, deserializes, resumes
  - More complex than Durable `WaitForExternalEvent` but achieves the same isolation guarantee

**When Container Apps still wins over Durable Functions:**
- When per-run process isolation is non-negotiable (true blast radius zero)
- When the runbook runtime must remain a standalone binary process (if Go binary is reintroduced alongside C# wrapper)
- When governance requires provable process-level separation

**Verdict:** Container Apps (Jobs) remains the **Thin Relay architecture choice**. For the persistent worker pattern with C#, Durable Functions is now superior. Container Apps doesn't lose ground — it just no longer needs to be the default.

---

### 1F. Azure Container Instances (ACI)

**Per-second pricing:** ~$0.0000135/vCPU-sec, ~$0.0000015/GB-sec (Linux).  
**1 vCPU × 1 GB running 24/7:** ~$42/mo.

**Re-evaluation with C#:**
- C# containers are fully supported on ACI.
- Problem is not language — it's operational: no queue trigger integration, no self-healing, no autoscale.
- ACI requires an external orchestrator to start a container per run, which adds latency and complexity.
- If you already have Container Apps, ACI adds nothing Container Apps Jobs doesn't do better.

**One valid use case:** Burst isolation for compute-heavy runbooks where Container Apps cold start is too slow (rare scenario). ACI can pre-warm a container group.

**Verdict:** **PASS**. ACI remains impractical as the primary compute. Container Apps Jobs provides the same per-run isolation with better lifecycle management.

---

## 2. Compute Decision Matrix

| Option | Timeout | Approval Gate | Scale-out | Cost Model | VNet | Deployment | GERT Score |
|--------|---------|--------------|-----------|------------|------|-----------|------------|
| App Service P1v3 + IHostedService | ♾️ None | Checkpoint required | 30 | Flat $124/mo | ✅ | Slots | ⭐⭐⭐ MVP |
| Functions Premium EP1 + Durable | ♾️ None | Native (WaitForEvent) | 100 | $88+ baseline | ✅ | Slots | ⭐⭐⭐⭐ |
| **Flex Consumption + Durable** | **♾️ None** | **Native (WaitForEvent)** | **1,000** | **Pay-per-use** | ✅ | Limited | **⭐⭐⭐⭐⭐ Primary** |
| Container Apps Jobs | ♾️ None | Checkpoint required | 300 | Per-vCPU-sec | ✅ | Revision | ⭐⭐⭐⭐ Thin Relay |
| Functions Consumption Y1 | 🚨 10 min | N/A | 200 | Pay-per-use | ❌ | Zip | 🚨 RED LINE |
| ACI | ♾️ None | Checkpoint required | Manual | Per-sec | ✅ | Manual | ⭐ Pass |

---

## 3. Multi-Tenancy Cost Model

### Shared Plan (All Tenants)

**App Service shared P1v3:**
- $124/mo total, all tenants share compute
- Blast radius: one tenant's CPU spike affects others
- Mitigated by: Service Bus message TTL, run timeout enforcement in C# runtime
- Suitable for: MVP, SMB customers, < 10 concurrent tenant runs

**Flex Consumption shared task hub:**
- $0/mo at zero load; pay per execution second
- Logical tenant isolation via Service Bus sessions (per-tenant session ID)
- Billing naturally per-tenant (track execution seconds per session tag)
- Shared Storage Account throughput: watch at high volume

**Durable Functions multi-tenant task hubs:**
- Named task hub per tenant (separate queues/tables in shared Storage)
- Storage account limit: 20,000 transactions/sec (Standard) — shared pool
- At high concurrency: separate Storage accounts per enterprise tenant

### Per-Tenant Plan (Full Isolation)

| Tier | Cost/Tenant/Mo | Scale | Notes |
|------|---------------|-------|-------|
| App Service P1v3 | $124 | 30 instances | Clean isolation, high cost |
| Functions EP1 | ~$88–173 | 100 instances | Better scale, higher floor |
| Container Apps (min 1 replica) | ~$63 | 300 instances | Pay-per-use with warmth |

**Per-tenant plan is only economically viable at enterprise pricing tiers** (customer paying > $500/mo platform fee). For standard SaaS, shared-plan with logical isolation is the only viable model.

**Recommendation:** 
- Standard tenants: Shared Flex Consumption + Service Bus session routing + task hub per tenant
- Enterprise tenants: Dedicated EP1 plan + isolated task hub + private Storage account

---

## 4. Approval Gate Analysis

This is the highest-value discriminator in the C# stack selection.

### Option A — Durable Functions `WaitForExternalEvent` (RECOMMENDED)

```csharp
[FunctionName("RunbookOrchestrator")]
public async Task RunOrchestrator([OrchestrationTrigger] IDurableOrchestrationContext ctx)
{
    var run = ctx.GetInput<RunRequest>();
    
    // Execute GERT runtime as an Activity (respects Red Lines 1, 2, 3)
    var result = await ctx.CallActivityAsync<RunResult>("ExecuteRunbook", run);
    
    if (result.RequiresApproval)
    {
        // Orchestrator hibernates. Zero compute. Zero cost.
        var approval = await ctx.WaitForExternalEvent<ApprovalResult>(
            "approval", 
            timeout: TimeSpan.FromDays(30));
        
        // Resume after human clicks approve
        await ctx.CallActivityAsync("ResumeRunbook", (run, approval));
    }
}
```

- Zero infrastructure work to implement pause/resume
- Compute cost during approval wait: **$0.00**
- Maximum wait time: configurable, days to indefinite
- Wake-up mechanism: HTTP POST to `/{functionName}/instances/{instanceId}/raiseEvent/approval`
- Governance: runtime still controls its own lifecycle (Activity boundary is per-run, not per-step)

### Option B — Service Bus Message Deferral (NEXT BEST)

- Worker defers the Service Bus message when approval is required
- Stores deferred sequence number in RunStore (DB)
- On approval event: worker reacquires deferred message, resumes processing
- Works with ANY compute host (App Service, Container Apps, Flex Consumption without Durable)
- More code to write than Option A, but no framework dependency

### Option C — Checkpoint Serialize + Terminate (Container Apps Jobs)

- Job serializes GERT run state to Blob Storage when approval gate hit
- Job exits (container terminates — zero blast radius)
- On approval: new Job starts, deserializes state, resumes from checkpoint
- Requires GERT C# runtime to support state serialization/deserialization
- Clean blast radius, but adds development complexity

**Decision:** Approval gate implementation should drive compute tier selection. If approval gates are frequent and multi-day waits are expected → **Durable Functions is the only tier with zero-cost hibernation**. App Service and Container Apps require active keep-alive or checkpoint infrastructure.

---

## 5. VNet Integration by Tier

| Tier | VNet Integration | Private Endpoints | Static Outbound IP | Cost |
|------|-----------------|------------------|-------------------|------|
| App Service B2/B3 | ❌ | ❌ | ❌ | — |
| App Service Standard+ | ✅ | ✅ | ✅ | Included in plan |
| App Service Premium v3 | ✅ | ✅ | ✅ | Included in plan |
| Functions EP1+ | ✅ | ✅ | ✅ | Included in plan |
| Functions Flex Consumption | ✅ | ✅ (verify region) | ✅ | Included |
| Functions Consumption Y1 | ❌ | ❌ | ❌ | — |
| Container Apps | ✅ (internal env) | ✅ | ✅ | +~$35/mo for dedicated env |

**For GERT:** VNet integration is required to reach private Service Bus endpoints and private Storage in enterprise deployments. B2/B3 are immediately disqualified for enterprise. All recommended tiers (P1v3, EP1, Flex Consumption, Container Apps with dedicated env) support VNet.

---

## 6. Deployment Model Comparison

| Approach | Blue/Green | Rollback | Zero-Downtime | Notes |
|----------|-----------|----------|--------------|-------|
| App Service slots | ✅ Instant swap | ✅ Swap back | ✅ | Slots on Standard+/Premium |
| Functions Premium slots | ✅ | ✅ | ✅ | Same slot mechanism |
| Flex Consumption | ⚠️ Zip deploy only | Re-deploy | Limited | Slots feature: verify GA |
| Container Apps revisions | ✅ Traffic split | ✅ | ✅ | Canary % traffic to new revision |
| ACI | ❌ | Manual | ❌ | Requires external LB |

**Note on Flex Consumption deployment slots:** As of late 2025, deployment slots were in limited preview for Flex Consumption. Verify availability before committing to this tier if zero-downtime deploy is non-negotiable. Fallback: use revision-based Container Apps or Premium plan with slots.

---

## 7. Revised Recommended Stack

### Compute (Tiered)

**Dev / MVP:**
- App Service P1v3 (shared)
- Single plan: ASP.NET Core API + IHostedService worker
- Cost: ~$124/mo flat
- Approval gate: Service Bus message deferral
- Deploy: slots (P1v3 includes 20 slots)

**Production (Recommended):**
- **Functions Flex Consumption + Durable Functions**
- Queue trigger: Service Bus with KEDA (Durable Extension handles this natively)
- Approval gate: `WaitForExternalEvent` (zero-cost hibernation)
- Cost: pay-per-use, ~$0/mo at zero load
- Scale: 1,000 instances
- Multi-tenant: named task hub per tenant, shared Storage Account (upgrade to per-tenant storage at enterprise tier)

**Enterprise / Full Isolation:**
- Functions Premium EP1 per tenant (or per-tenant task hub with dedicated Storage)
- Dedicated VNet, private Service Bus, private Storage
- Durable Functions: same programming model as Flex, better slot support
- Cost: ~$88-173/mo baseline per tenant plan

**Per-Run Isolation (Thin Relay topology, Barbara's Option 1):**
- Container Apps Jobs (C# container)
- Triggered by Service Bus message (ACA Jobs queue trigger)
- Approval gate: checkpoint serialize to Blob + terminate + resume on new Job
- Cost: ~$0.000024/vCPU-sec per run
- Zero residual blast radius between runs

---

### Non-Compute — Unchanged

| Service | Tier | Reason to Keep |
|---------|------|----------------|
| Service Bus | Standard | DLQ critical, sessions for FIFO, 14-day retention, 256KB message. C# has first-class SDK. No change. |
| Entra External ID | Free (50K MAU) | Customer auth, OIDC/SAML. C# ASP.NET Core has native Entra middleware. Better, not worse. |
| Static Web Apps Standard | $9/mo | SPA hosting, unlimited custom domains, white-label. No change. |
| Event Grid | Pay-per-event | Webhook fanout, 24h retry. No change. |

**One enhancement with C#:** Entra External ID + `Microsoft.Identity.Web` (MSAL) middleware in ASP.NET Core is a drop-in. Authentication story is actually simpler than with a custom Go HTTP server.

---

## 8. Bicep Anchors (IaC Implications)

Key Bicep modules needed for Flex Consumption + Durable:

```bicep
// Functions Flex Consumption Plan
resource flexPlan 'Microsoft.Web/serverfarms@2023-12-01' = {
  name: 'gert-flex-plan'
  sku: { name: 'FC1'; tier: 'FlexConsumption' }
  properties: { reserved: true }  // Linux
}

// Durable Functions Storage Account (per tenant for enterprise)
resource durableStorage 'Microsoft.Storage/storageAccounts@2023-01-01' = {
  name: 'gertdurable${tenantId}'
  sku: { name: 'Standard_LRS' }
  kind: 'StorageV2'
}

// Managed Identity — zero secrets
resource funcIdentity 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' = {
  name: 'gert-worker-identity'
}

// Service Bus role assignment — no connection strings
resource sbRole 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(serviceBus.id, funcIdentity.id, 'Azure Service Bus Data Receiver')
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '4f6d3b9b-027b-4f4c-9142-0e5a2a2247e0')
    principalId: funcIdentity.properties.principalId
  }
}
```

Managed Identity everywhere. Zero connection strings in app settings. Service Bus Data Receiver role on the namespace.

---

## 9. Open Questions (Updated)

1. **Flex Consumption GA status:** Are deployment slots available? Is VNet integration stable in target region?
2. **Durable history size:** What is the expected step count and output payload size for a GERT runbook? Do we need blob externalization from day one?
3. **Task hub isolation:** How many tenants share one Storage account before throughput becomes a concern? At what tenant count do we need per-tenant storage?
4. **Approval gate latency SLA:** Is the Durable `WaitForExternalEvent` wake-up time acceptable? (Typically < 5s from HTTP POST to orchestrator resume.)
5. **Cold start budget:** Can Flex Consumption's improved-but-not-zero cold start be tolerated for interactive runbook submissions? Or do we need a min-instance > 0 (which adds baseline cost)?
6. **Slots on Flex Consumption:** If not available, is revision-based Container Apps deployment acceptable for the API layer?
7. **Service Bus Standard throughput:** Confirm 1,000 msg/sec/queue is sufficient for worst-case concurrent tenant volume.
# White-Label Web Portal Options Menu

**Prepared by:** Leslie (Frontend Dev)  
**Date:** 2026-06-03  
**Status:** For team review and consensus

---

## Overview

GERT needs to deliver white-labeled web portals where end-users (applicants, patients, citizens) run governance runbooks as part of business workflows — applying for a service, granting consent, completing onboarding. The end user **never knows GERT exists**. They're filling out an application, not "running a runbook."

This options menu spans four critical dimensions:

1. **Deployment model** — how the web app reaches the end user
2. **Authentication UX** — how they sign in
3. **Runbook execution UX** — how they experience the step-by-step process
4. **Progress & delivery** — how they learn their submission was completed

For each option, I've focused on **what the end user actually sees**, not the engineering complexity.

---

## DIMENSION 1: White-Label Deployment Model

> **Principle:** The app is branded for the customer, the end-user never knows they're on a platform.

### Option 1A: Shared Multi-Tenant SaaS (Branded per Config)

**How it works:**
- Single React/TypeScript web app deployed to Azure Static Web Apps
- Tenant ID inferred from subdomain: `customer-a.platform.com`, `customer-b.platform.com`
- Branding (logo, colors, text) fetched from config service on page load
- Routes, permissions, runbook catalog configured per tenant in backend

**What the end-user sees:**
- Visits `mybank.platform.com` → sees your bank's logo, colors, text
- Feels like a native bank portal (because it's branded to your bank)
- Works on mobile, desktop, across browsers

**Branding:**
- Logo, primary/secondary colors: config
- Company name, legal links: config
- Custom domain: via CNAME to platform domain
- Language/localization: config + i18n system

**Custom domain support:**
- CNAME to `mybank.platform.com` — no HTTPS cert issues
- Platform provides SSL automatically (*.platform.com)
- Customer sees their own domain in address bar

**Operational overhead per new tenant:**
- Create config entry (tenant ID, branding, runbook catalog)
- Set up CNAME on customer's DNS
- Trigger backend user provisioning
- **No code deploy, no app rebuild** — tenants are live immediately

**Tradeoffs:**
- ✅ Lowest ops overhead; scale to 100+ tenants with one deployment
- ✅ Consistent, tested UX across all customers
- ✅ Tenant isolation via Azure role-based access
- ❌ Customers can't customize UI/flows beyond branding
- ❌ Shared infrastructure means one tenant's issue can affect others (mitigated via rate limits)

---

### Option 1B: Separate Static Deployments Per Customer

**How it works:**
- Separate React app instance deployed per customer
- Customer has own Azure Static Web Apps resource
- Deployed to `customer-a.com`, `customer-b.com`, etc.
- Customers can host on their own domain, CDN, region of choice

**What the end-user sees:**
- Visits `apply.bankname.com` → sees bank's branding, feels fully native
- App loads from bank's infrastructure (faster if they're geographically close)

**Branding:**
- Full repository fork per customer — all branding hardcoded
- Or: parameterized build that ingests config files at build time
- Complete control; customers can modify any UI

**Custom domain support:**
- Native — app is already on customer's own domain
- Can point to customer's existing CDN/WAF
- Customer owns the HTTPS cert

**Operational overhead per new tenant:**
- Fork/template the repository
- Customize branding, update build config
- Deploy to customer's Azure subscription (or platform's, with separate resource)
- **Requires a deployment pipeline per customer** — ~15 min per onboarding
- Updates to shared features require deploying N times

**Tradeoffs:**
- ✅ Customers have complete control; can customize beyond branding
- ✅ Tenant isolation is strong (separate infrastructure)
- ✅ Compliance teams like it (customer owns the deployment)
- ❌ High per-tenant ops cost; scale to 20+ tenants gets expensive
- ❌ Bug fixes and feature updates require N deployments
- ❌ Each customer runs their own infrastructure (dup costs)

---

### Option 1C: Iframe or Web Component Embedding

**How it works:**
- Platform provides a self-contained web component (e.g., `<gert-portal>`)
- Customer embeds it in their existing website: `<gert-portal tenant-id="X" token="JWT"></gert-portal>`
- Web component is versioned, deployed to CDN
- Component handles auth, UI, execution internally

**What the end-user sees:**
- Visits `bankname.com/apply` (customer's existing website)
- See your bank's header, nav, footer
- Embedded runbook portal (styled to match the bank's design)
- Feels seamless — never left the bank's site

**Branding:**
- CSS variable injection (`--primary-color`, `--font-family`, etc.)
- Component respects customer's design tokens
- Not full control — component's UI structure is fixed

**Custom domain support:**
- Native — customer's own domain, no platform domain involved
- Works with customer's existing WAF, auth, logging

**Operational overhead per new tenant:**
- Share component embed code + config (tenant ID, branding tokens)
- Customer adds one line of HTML and sets env vars
- No platform deployment needed
- **Lightest ops burden** — tenant can be live in minutes

**Tradeoffs:**
- ✅ Customers can embed in existing portals — no domain redirect needed
- ✅ Almost no platform ops; customers self-serve onboarding
- ✅ Works with customer's existing design system
- ❌ Auth token exchange is complex (cross-origin; requires backend coordination)
- ❌ Harder to inject custom fonts, global styles
- ❌ Component sizing/responsiveness can be tricky in customer's layout

---

### Option 1D: Customer-Hosted Shell + Component Library

**How it works:**
- Platform provides a design system / component library (npm package)
- Customer hosts the web app shell on their infrastructure
- Customer imports runbook components, authentication, session management from platform library
- Customer's developers build the portal UI using platform components

**What the end-user sees:**
- Visits customer's domain
- Portal looks & feels like customer's other products (fully customized)
- Works on any infrastructure (customer's cloud, on-prem, etc.)

**Branding:**
- Customer has complete control — they own the CSS and layout
- Platform library provides unstyled or themed components
- Customers can use Tailwind, Material, or their own design system

**Custom domain support:**
- Native — customer's own domain and infrastructure

**Operational overhead per new tenant:**
- Provide npm package, SDKs, documentation
- Customer's dev team integrates library into their shell
- Customer deploys to their infra
- **High support overhead** — many ways to integrate poorly
- Each customer's version may drift; hard to push updates

**Tradeoffs:**
- ✅ Full control for technically sophisticated customers
- ✅ No platform infrastructure lock-in
- ✅ Works for on-prem or hybrid deployments
- ✅ Customers can integrate with existing identity, logging, monitoring
- ❌ Requires developer time from customer; not good for non-technical customers
- ❌ Library versioning / integration support becomes complex
- ❌ Hard to push security fixes or feature updates
- ❌ Risk of customers building non-compliant integrations

---

## DIMENSION 2: Authentication UX

> **Principle:** The end user should use credentials they already have. We hide the technical plumbing (Entra, OIDC, etc.).

### Option 2A: Entra External Identities (Platform-Managed)

**What the end-user sees:**
- Clicks "Sign in" on the portal
- Redirected to a login page branded for the company (e.g., "Sign in with BankName")
- Can sign in with:
  - Email + password (Entra manages credentials)
  - Microsoft account (if enabled)
  - Social login (Google, Facebook, if enabled)
- Redirected back to portal, logged in

**Setup from customer's perspective:**
- Platform manages the Entra tenant + branding
- Customer provides logo, colors, company name
- **Dependency:** Azure Entra External ID service

**Mobile experience:**
- App Store / OAuth flow works smoothly
- Native app can use system browser for login (best practice)
- Return to app after login

**Compliance & audit:**
- Platform owns the credential store
- Audit logs in Azure Entra
- Good for GDPR (platform is the processor)

**Tradeoffs:**
- ✅ Lowest friction for customer setup; we handle identity
- ✅ Branded login page reduces "is this a phishing site?" concern
- ✅ Entra integration is mature, compliant (SOC2, FedRAMP available)
- ✅ Supports MFA out of the box
- ❌ Vendor lock-in to Azure Entra
- ❌ Customers with strict IdP requirements (ADFS, Okta) can't use this
- ❌ If Entra goes down, all portals can't authenticate

---

### Option 2B: Customer's Own IdP via OIDC Federation

**What the end-user sees:**
- Clicks "Sign in"
- Redirected to **your company's login page** (Okta, Azure AD, Auth0, etc.)
- Signs in with company credentials (likely SSO if they're an employee or contractor)
- Redirected back to portal, logged in

**Setup from customer's perspective:**
- Customer provides OIDC discovery URL + client ID/secret
- We validate OIDC metadata and configure federation
- Portal redirects to customer's login endpoint
- Customer's IdP returns identity token; we validate it

**Mobile experience:**
- Custom app can handle OAuth flow
- If customer has a corporate app, may already be SSO'd
- Works with customer's session management

**Compliance & audit:**
- Customer owns credentials → complies with strict IdP policies
- Good for enterprises with ADFS, RACF, or proprietary identity systems
- Audit logs on customer's IdP

**Tradeoffs:**
- ✅ Customers with existing IdP (Okta, Azure AD) don't duplicate credentials
- ✅ No lock-in to platform's identity provider
- ✅ Regulatory requirement for many enterprises: "identity must stay on-premises"
- ✅ Supports SSO if customer has already authenticated elsewhere
- ❌ Requires upfront integration work per customer
- ❌ We can't control login UX/branding (customer's IdP controls it)
- ❌ If customer's IdP integration breaks, we can't easily troubleshoot

---

### Option 2C: Social Login (Google, Microsoft, etc.) via Entra

**What the end-user sees:**
- Clicks "Sign in with Google" (or Microsoft, Facebook, etc.)
- Signs in with personal Google account (or Microsoft account)
- Redirected back, logged in
- **No password to remember**

**Setup from customer's perspective:**
- Entra External ID supports social login via federation
- We enable Google/Microsoft/etc. as an identity provider in Entra

**Mobile experience:**
- Extremely smooth; most users already on Google/Microsoft
- Single tap on mobile if already authenticated to device

**Compliance & audit:**
- Platform stores user profiles (email, name) + link to social provider
- Social provider doesn't store interaction logs
- Good for consumer-facing processes (not B2B)

**Tradeoffs:**
- ✅ Lowest friction for one-time users (no account signup)
- ✅ Works great for mobile (pre-authenticated in most cases)
- ✅ No password management burden
- ❌ Not appropriate for regulated/professional processes (healthcare, finance often forbid this)
- ❌ Some customers don't allow employee use of personal social accounts
- ❌ Requires explicit consent from customer

---

### Option 2D: Magic Link / Email OTP (No Password)

**What the end-user sees:**
- Enters their email address (or phone)
- Clicks "Send me a link" or "Send me a code"
- Checks email/SMS
- Clicks link or enters 6-digit code
- Logged in — **no password ever entered**

**Setup from customer's perspective:**
- Email/SMS provider configured (SendGrid, Twilio, or Entra Email OTP)
- Very low complexity

**Mobile experience:**
- Link can open the app directly (deep linking)
- User never leaves the app
- Or: code can be auto-filled on iOS 16+
- **Excellent for mobile**

**Compliance & audit:**
- No password = no credential storage
- Very clean for one-time consent flows
- Audit: "User proved email ownership, acted via link"

**Use cases:**
- "I need to grant consent to this medical record release" — one-time link, one-time action
- Multi-day process where user returns? Add a "Continue your application" email button

**Tradeoffs:**
- ✅ **Best for one-time or short-lived processes**
- ✅ Excellent mobile UX
- ✅ No password means lower phishing risk
- ✅ Works for users without existing credentials (external users, contractors)
- ✅ Audit trail is explicit: "user proved email ownership"
- ❌ Not appropriate for long-lived sessions (user has to re-authenticate frequently)
- ❌ Depends on email delivery (can be slow in spam filter)
- ❌ Users find it unfamiliar (not yet mainstream)

---

## DIMENSION 3: Runbook Execution UX

> **Principle:** The end user is filling out an application, not "executing a runbook." We hide the step-by-step details and show a process they understand.

### Option 3A: Wizard (Step-by-Step, Linear)

**What the end-user sees:**
- Landing page: "Let's get your application started"
- **Step 1 of 5:** Personal information (text fields appear, tooltip explains each)
- Clicks "Next" → validates → shows **Step 2 of 5:** Income & assets
- Progress bar shows where they are (Step 2 of 5, 40% done)
- **Can't skip steps.** "Back" button available to edit prior steps
- Final step: review & confirm → submit
- After submit: "Your application is being processed" + status updates

**Mobile experience:**
- Full-screen form per step
- Large touch targets
- No horizontal scrolling
- Mobile-friendly progress indicator at top

**Use cases:**
- **Complex, multi-day processes** where you want to guide the user
- **Processes with dependencies** — "If you answered 'yes' to Q7, you must answer Q12"
- **Non-technical end-users** — step-by-step feels safe and clear

**Branching logic:**
- Support conditional steps: "If divorced, provide ex-spouse info"
- Don't show disabled steps; reveal them based on answers

**Tradeoffs:**
- ✅ **Most accessible for complex forms**
- ✅ Progress bar reduces "how much more?" anxiety
- ✅ Validation per step prevents late-stage data errors
- ✅ Can't accidentally skip required fields
- ✅ Good for processes that take 10+ minutes (users feel progress)
- ❌ More clicks than form-first (5 steps = 5 "Next" clicks)
- ❌ Hard to see "big picture" of the form
- ❌ Slower for users who know what they're doing

---

### Option 3B: Form-First (All Inputs Visible, Then Execute)

**What the end-user sees:**
- Landing page: full application form (all fields visible)
- Scroll through the entire form, fill in what they know
- Fields they don't understand have inline help text / tooltips
- Sections expand/collapse (optional details, complex sections)
- **One large "Submit Application" button** at the bottom
- Validation runs on submit; errors highlight in red
- After submit: real-time progress (see steps executing, or polling status)

**Mobile experience:**
- Scrollable form; not as clean as wizard
- But: user can see all fields before committing
- Search/Ctrl+F works (find fields by name)

**Use cases:**
- **Quick processes** (5 min, 5–10 fields)
- **Users who want to see what they're signing** — compliance, legal review
- **Technical users** who prefer not to be hand-held

**Tradeoffs:**
- ✅ **Fastest for simple forms** (no "Next" clicks)
- ✅ Users see the full scope upfront
- ✅ Faster for returning users or copy-paste workflows
- ✅ Better for accessibility tools (screen readers can see full form)
- ❌ Overwhelming for complex forms (50+ fields)
- ❌ Validation errors all at once can be frustrating
- ❌ Mobile experience is scroll-heavy

---

### Option 3C: Chat-Like (Conversational, Step Inputs Feel Like Dialogue)

**What the end-user sees:**
- Message-like interface: "Hello! Let's get your application started. What's your full name?"
- User types or fills in name field at bottom
- Message appears in chat above (styled as "user message")
- System responds: "Nice to meet you, Alex! Now, what's your email?"
- User enters email
- System: "Thanks! Do you currently have health insurance?"
- [Yes] [No] [Not sure]
- User clicks "Yes"
- System: "Great. Which provider?"
- [Dropdown with providers]
- After all inputs collected: "Ready to submit? [Review] [Edit]"

**Mobile experience:**
- **Excellent** — feels like texting
- Chat scrolls naturally
- Native mobile patterns
- Large touch targets for buttons

**Use cases:**
- **Very new/non-technical users** (elderly, low-literacy populations)
- **Mobile-first processes** (consent, quick surveys)
- **One-time processes** (not long sessions)
- **Processes meant to feel friendly/casual** (e.g., customer feedback, onboarding)

**Branching logic:**
- Easy to show/hide questions based on answers
- Flow feels natural ("Why are you asking that?" → "Because you said X")

**Tradeoffs:**
- ✅ **Most friendly/approachable for non-technical users**
- ✅ Excellent mobile UX (native chat patterns)
- ✅ Natural for conditional logic / Q&A
- ✅ Reduced cognitive load (one question at a time)
- ❌ Hard to see full context or review answers (scroll up to see past responses)
- ❌ Takes longer to complete (can't skim/scan like a form)
- ❌ Desktop UX feels gimmicky (not expected on web)
- ❌ Difficult for accessibility (no visible form labels)

---

### Option 3D: Status Dashboard (Submit + Return Later)

**What the end-user sees:**
- Dashboard page: list of "Your Applications"
- **[New Application]** button
- Existing applications show status: "Pending Review", "Approved", "Rejected", "Awaiting Your Input"
- Clicks "New Application" → presented with form (wizard or form-first)
- Fills in what they can, hits "Save & Continue Later"
- Dashboard appears: "Application saved. You'll receive an email when we need more info."
- Later, email arrives: "Your application needs one more piece — [Complete Application]"
- Clicks link → portal opens, form is pre-filled, shows only pending fields
- Completes, hits "Submit"

**Mobile experience:**
- Dashboard lists applications, easy to see status at a glance
- Works great for apps that take hours/days
- User can take breaks

**Use cases:**
- **Multi-day processes** (background checks, reviews)
- **Processes with approval gates** (human review, payment processing)
- **Mobile users** who might not complete in one session
- **Regulated processes** (healthcare, finance) where audit trail of "who accessed when" matters

**Email integration:**
- **Critical:** Email is often the primary notification
- "Your application is ready for review" → not a surprise
- "More information needed" → actionable link in email

**Tradeoffs:**
- ✅ **Best for processes spanning hours/days**
- ✅ Users don't feel rushed
- ✅ Workflow feels like a real process (not a quiz)
- ✅ Email integration gives users control (can process on their schedule)
- ✅ Audit trail is clear: "user accessed at 3pm, submitted at 4pm"
- ❌ More infrastructure (state management, session recovery)
- ❌ Users might abandon if too much time passes
- ❌ Mobile: "Continue" link expiration needs to be long enough

---

## DIMENSION 4: Progress & Result Delivery to End User

> **Principle:** The user submitted something; now they need to know when it's done. Email is often the primary notification channel.

### Option 4A: Real-Time Updates via Server-Sent Events (SSE)

**What the end-user sees:**
- Submit application
- Browser connects to server; page shows live updates:
  - ✓ "Received your submission"
  - ⏳ "Validating personal information..." → ✓ Done
  - ⏳ "Running background check..." → (stays for 10 min) → ✓ Complete
  - ⏳ "Sending confirmation email..." → ✓ Done
- After completion: "Application approved! You'll receive an email shortly."
- **User sees execution as it happens**

**Technical notes:**
- SSE connection over HTTPS (long-lived)
- Backend publishes step completion events
- Browser displays in real-time

**Mobile experience:**
- Works on iOS/Android browsers
- **But:** if user closes browser, updates stop
- If user navigates away, need recovery flow

**Tradeoffs:**
- ✅ **Premium UX; user feels engaged**
- ✅ Reduces "did it work?" anxiety
- ✅ Can show errors in real-time
- ✅ Great for processes that complete in <5 min
- ❌ Requires persistent connection (not ideal on mobile with poor connectivity)
- ❌ If user closes app/browser, misses updates
- ❌ SSE is still a bit exotic; not all firewalls/proxies support it well
- ❌ Scaling SSE requires careful connection management

---

### Option 4B: Polling (Regular Status Checks, Simple)

**What the end-user sees:**
- Submit application
- Page shows: "Your application is being processed..."
- Every 2-5 seconds, browser checks backend: "Is it done yet?"
- Backend responds: `{ status: "validating", step_number: 2 }`
- UI updates progress bar
- After completion: `{ status: "complete", result: "approved" }`
- Page shows final result

**Mobile experience:**
- Works everywhere (no persistent connection needed)
- Doesn't drain battery as much as SSE
- Can be slower if polling interval is large

**Tradeoffs:**
- ✅ **Simplest to implement and maintain**
- ✅ Works everywhere (no connectivity issues)
- ✅ Tolerates mobile app backgrounding well
- ✅ Easy to add retry logic if backend is slow
- ❌ Slightly laggy (2-5 sec delay between update and display)
- ❌ Creates more backend load (more requests)
- ❌ Users might refresh page or navigate away

---

### Option 4C: Email Notification on Completion

**What the end-user sees:**
- Submits application
- Sees: "Your application has been received. We'll email you when it's complete."
- User navigates away, closes browser
- 2 hours later: email arrives: "Your application has been processed. [View Result]"
- Clicks link → portal shows result + next steps

**Email content:**
- Plain-language summary (not technical jargon)
- "Your application was approved" (not "workflow 47 completed with status OK")
- Clear next steps: "What happens now?"

**Mobile experience:**
- Email is primary interface
- User gets notification on phone lock screen
- Clicks link, loads portal in browser

**Tradeoffs:**
- ✅ **Most reliable way to reach users** (email is always-on)
- ✅ Users don't have to stay on page
- ✅ Works for multi-hour/multi-day processes
- ✅ Email record is audit trail
- ✅ User can forward email to others ("Here's the approval")
- ❌ Email delivery can be slow (spam filters)
- ❌ Can't show real-time progress
- ❌ Requires valid email address (more data collection)

---

### Option 4D: In-App Notification + Async Result Page

**What the end-user sees:**
- Submits application
- Sees: "Your submission was received. You can check the status anytime."
- Navigates to "My Applications" dashboard
- Sees application card with status: "Processing..."
- Refreshes page (or page auto-updates every 10 sec)
- Status changes to "Approved"
- Card now shows result details

**Plus email notification (recommended):**
- "Your application was processed" → user clicks email link → result page loads with status

**Mobile experience:**
- Dashboard is primary interface
- User bookmarks or adds to home screen
- Checks back periodically
- Notification arrives via email

**Tradeoffs:**
- ✅ **Flexible; works for any timeline**
- ✅ User can check status whenever they want (no "am I notified?" anxiety)
- ✅ Dashboard is dashboard (not a one-off result page)
- ✅ Combines in-app UX with email notification
- ✅ Works well with status dashboard execution model
- ❌ Requires user to actively check (not as discoverable as email)
- ❌ If email is the only notification, user might not see result for days
- ❌ **Best combined with email, not standalone**

---

## Recommended Combinations

> Not all combinations make sense. Here are sensible defaults based on use case.

### **Use Case 1: Quick Consent (5 min, one-time, employee approval)**

**Recommended:**
- **Deployment:** Shared multi-tenant SaaS (1A) — low ops, consistent UX
- **Auth:** Magic link / email OTP (2D) — fastest, best for one-time
- **Execution:** Form-first (3B) — all fields visible, user can review before signing
- **Delivery:** Email notification (4C) — email confirms completion, provides audit trail

**Why:** User sees everything upfront, consents, receives confirmation email. No account creation needed. Audit trail is clean.

---

### **Use Case 2: Complex Multi-Step Application (30 min, days in process)**

**Recommended:**
- **Deployment:** Shared multi-tenant SaaS (1A) OR separate deployments per customer (1B) if customer has strict compliance requirements
- **Auth:** Entra External ID (2A) for speed, or customer's own IdP (2B) for enterprises
- **Execution:** Wizard (3A) — guides user through complex flow, progress bar reduces anxiety
- **Delivery:** Polling + email (4B + 4C) — user sees progress if they stay, email notifies when done

**Why:** Customers appreciate step-by-step guidance. Email + in-app status gives multiple ways to stay informed.

---

### **Use Case 3: Mobile-First, Non-Technical User (Health form, patient consent)**

**Recommended:**
- **Deployment:** Shared multi-tenant SaaS (1A)
- **Auth:** Magic link / email OTP (2D) — no password confusion, mobile-friendly
- **Execution:** Chat-like (3C) — feels familiar, one question at a time, accessible
- **Delivery:** Email confirmation + in-app notification (4D with 4C) — email is primary, app shows status

**Why:** Chat is friendly. Magic link is mobile-native. Email is how patients expect to be notified.

---

### **Use Case 4: Enterprise / Regulated Process (Healthcare, Finance, with approval gates)**

**Recommended:**
- **Deployment:** Customer's own IdP (1B) or customer-hosted shell (1D) — compliance teams want no platform lock-in
- **Auth:** Customer's own IdP via OIDC (2B) — credentials stay on-prem, audit trail on customer's systems
- **Execution:** Status dashboard (3D) — users can save & return, multiple approval gates are clear
- **Delivery:** Email notification on each gate + in-app dashboard (4C + 4D) — formal process workflow

**Why:** Customers own auth, branding, audit trail. Users see formal workflow (not a "form"). Email provides formal notification of approval/rejection.

---

### **Use Case 5: Real-Time Execution Visibility (Infrastructure provisioning, immediate feedback needed)**

**Recommended:**
- **Deployment:** Shared multi-tenant SaaS (1A)
- **Auth:** Entra External ID (2A)
- **Execution:** Wizard (3A) with real-time progress display
- **Delivery:** Real-time SSE (4A) — user watches steps execute — plus email confirmation (4C)

**Why:** Process completes in <5 min. SSE shows real-time progress. Email confirms completion.

---

## Decision Framework

**Choose deployment (1):**
- 20+ customers, low per-tenant customization → **Shared SaaS (1A)**
- <10 customers, high customization needs → **Separate deployments (1B)**
- Customer wants to embed in existing site → **Iframe (1C)**
- Customer has strict compliance / wants to host → **Customer shell (1D)**

**Choose auth (2):**
- One-time process, non-technical users → **Magic link (2D)**
- Enterprise with existing IdP → **Customer's OIDC (2B)**
- Consumer users, low friction → **Social login (2C)**
- Default for everything → **Entra External ID (2A)**

**Choose execution (3):**
- Complex, multi-step, users need guidance → **Wizard (3A)**
- Simple form, users want to see all fields → **Form-first (3B)**
- Non-technical, mobile-first users → **Chat-like (3C)**
- Multi-day process with approval gates → **Status dashboard (3D)**

**Choose delivery (4):**
- Process <5 min, user stays on page → **Real-time SSE (4A)**
- Any process, simple implementation → **Polling (4B)**
- Process takes hours+, email is primary → **Email notification (4C)**
- Multi-day process, dashboard-centric → **In-app + async (4D)**
- **Best practice:** Combine email + in-app (4C + 4D) for most cases

---

## Next Steps

1. **Team feedback:** Which combinations resonate for our first customers?
2. **Proof of concept:** Build one end-to-end example (e.g., quick consent flow: 1A + 2D + 3B + 4C)
3. **Customer interviews:** Validate against real use cases
4. **Architecture:** Define config schema for theming, runbook catalog, auth providers

---

## 2026-06-05 — Phase B speculative kickoff (Don PR #10) + spec hygiene

**Authors:** Don (Backend Dev), Tess (Test Engineer), Barbara (Lead Architect)  
**Status:** Phase B lexer+parser delivered via PR #10 (draft, stacked on PR #9). Spec ambiguities resolved.

### Phase B Status: GXL Lexer + Parser (Don PR #10)

**Deliverable:** GXL lexer and recursive-descent parser, all 83 GXL-PARSE conformance vectors PASS on first run.

**Implementation path:** `internal/eval/gxl/` (Go package with `//go:build gxl` tag)
- `lexer.go` — tokenization per gxl.ebnf §2
- `parser.go` — recursive-descent per gxl.ebnf §3
- `ast.go` — Position, Node interface, LiteralNode, UnaryNode, BinaryNode, PathNode, CallNode
- `errors.go` — ParseError type + GXL-PARSE-* and GXL-TYPE-004 error codes
- Unit tests: 13 lexer tests + 11 parser tests; all pass

**Test results:**
| Corpus | Total | PASS | FAIL | SKIP |
|--------|-------|------|------|------|
| GXL-PARSE | 83 | **83** | 0 | 0 |
| GXL-EVAL | 92 | 0 | 0 | 92 (Phase C) |
| GXL-PATH | 36 | 0 | 0 | 36 (Phase D) |
| GIS-PATH | 15 | 0 | 0 | 15 (Phase E/Ken) |
| GCP-PATH | 41 | 0 | 0 | 41 (Phase F/Ken) |
| **TOTAL** | **267** | **83** | **0** | **184** |

**PR Details:** `ormasoftchile/gert` PR #10 (draft, base: `phase-a-pjvm` = PR #9 base). Branch: `phase-b-lexer-parser`.

### Tess Fix: YAML Escape Encoding (commit `424a334`)

**Issue:** Conformance vectors TV-GXL-PARSE-062 and TV-GXL-PARSE-063 in `design/gert/conformance/tv-gxl-parse.yaml` used YAML single-quoted strings with doubled backslashes (`\\q`, `\\xFF`). In YAML single-quoted scalars, backslash has no escape semantics (only `''` is special), so `\\q` produced two literal backslashes + `q`, not the intended `\q` invalid-escape sequence.

**Impact:** The GXL lexer saw valid escapes (`\\` → `\`) + regular char, yielding `parse_ok` instead of the intended `GXL-PARSE-004` error.

**Fix applied:** `design/gert/conformance/tv-gxl-parse.yaml`
| Vector | Before | After |
|--------|--------|-------|
| TV-GXL-PARSE-062 | `'"hello \\q world"'` | `'"hello \q world"'` |
| TV-GXL-PARSE-063 | `'"hex \\xFF"'` | `'"hex \xFF"'` |

**Verification:** YAML `repr` check confirms both vectors now present single-backslash sequences to GXL parser. Schema validation passed; 3 spot-check vectors unchanged.

**Timing:** Landed on `gert-private/main` (commit `424a334`) before Ken's Phase A sync runs, protecting the 83/83 pass rate from regression to 81/83 upon PR merge.

### Barbara Arbitration: GDP/Keyword Dual-Role (commit `fdd14db`)

**Issue:** TV-GXL-PARSE-028 requires `list[10]` to parse_ok, but `list` is keyword KW_LIST. Grammar defines GDP = IDENT (excluding keywords), so strictly `list[10]` should error as GXL-PARSE-010 (keyword-as-identifier).

**Resolution (Option A — Legitimize pragmatic parser behavior):** Namespace prefix keywords `str`, `list`, and `regex` serve a **dual role**:

1. **When immediately followed by `.`:** Committed to NamespaceCall production only.
   - If NamespaceCall fails (method but no `(`), all Primary alternatives fail → `GXL-PARSE-001`.
   - Example: `str.foo` (no parens) → `GXL-PARSE-001` ✓

2. **When NOT immediately followed by `.`:** Treated as valid GDP root identifier.
   - NamespaceCall fails immediately (requires DOT), falls through to GDP → parse proceeds.
   - Example: `list[10]` → `list` not followed by DOT → valid GDP root → parse_ok ✓

**Scope:** Carve-out applies **only** to `str`, `list`, `regex`. Does NOT extend to `math`, `len`, `now`, `and`, `or`, `not`, `true`, `false`, `null`.

**Verification:** Gate vectors all pass under Option A:
- TV-GXL-PARSE-028: `list[10]` → parse_ok ✓ (NOT followed by DOT → GDP carve-out)
- TV-GXL-PARSE-048: `str.contains(message, "error")` → parse_ok ✓ (NamespaceCall success)
- TV-GXL-PARSE-083: `str.foo` → GXL-PARSE-001 ✓ (DOT-committed, NamespaceCall fails)

Surrounding namespace vectors (TV-GXL-PARSE-049..053) and keyword vectors (TV-GXL-PARSE-081, 082) all unchanged — no regressions.

**Spec updated:** `design/gert/grammar/gxl.ebnf` (§2 keyword comment, §2 IDENT exception, §3 NamespaceCall/GDP notes); `design/gert/sections/03a-expression-language.tex` (Keywords table dual-role annotation, Identifiers subsection GXL-PARSE-010 exception, GDP subsection dual-role paragraph, NamespaceCall `str.foo` note).

**Parser impact:** Don's PR #10 parser already implements the correct behavior — **no code change required**. TV-GXL-PARSE-028 remains `parse_ok` as committed.

### Phase B Exit-Criteria Readiness

Phase B merge order (pending Germán PR landing):
1. PR #8: Phase A sync infra (Ken)
2. PR #9: Phase A PJVM/Clock/Harness (Don)
3. PR #10: Phase B lexer+parser (Don)

Once PR #10 merges, Don can start **Phase C (GXL evaluator + stdlib, 92 tv-gxl-eval vectors)**.

---

## Decision: Remove /probe-token — Kills ICM MCP Session

**Date:** 2026-08-19  
**Author:** Ken (Backend Dev)  
**Status:** Recommended — awaiting coordinator/Cristián approval to merge  
**Severity:** High — currently in production extension; each invocation permanently disables ICM MCP for the VS Code session

### Problem Statement

After running `@gert /probe-token`, the ICM MCP server (`icm-mcp`) stops and cannot be restarted until a VS Code reload or machine reboot. The user correctly suspected the probe is doing something seriously wrong.

### Root Cause (Evidence-Based)

#### 1. Four unconditional `invokeTool` calls against a live remote MCP server

`runProbe()` in `src/probeToken.ts` lines 305–315 executes T1, T2, T3, and T4 with **no early exit on failure**:

```typescript
// T1 — synchronous
results.push(await runAttempt('T1-synchronous', invokeToolFn, token, ...));
// T2 — after microtask
await Promise.resolve();
results.push(await runAttempt('T2-microtask', invokeToolFn, token, ...));
// T3 — after 250ms setTimeout
await new Promise<void>((r) => setTimeout(r, 250));
results.push(await runAttempt('T3-macrotask', invokeToolFn, token, ...));
// T4 — after loopback HTTP round-trip
await loopbackRoundTrip();
results.push(await runAttempt('T4-loopback', invokeToolFn, token, ...));
```

Each `runAttempt` calls `invokeToolFn` which calls `vscode.lm.invokeTool(toolName, ..., cancellation)`.

#### 2. `icm-mcp` is a remote HTTP server — VS Code owns its session lifecycle

`$APPDATA\Code\User\mcp.json`:
```json
"icm-mcp": {
  "type": "http",
  "url": "https://icm-mcp-prod.azure-api.net/v1/"
}
```

This is not a local stdio process. VS Code's MCP client maintains an in-memory HTTP session for it. Every `invokeTool` failure that is a transport/session failure causes VS Code to attempt a reconnect. After repeated reconnect failures (4 in rapid succession from the probe), VS Code's MCP state machine for `icm-mcp` enters a permanently-disabled state for the current VS Code session.

#### 3. The chat `_token` is passed as `cancellation` to every invokeTool call

`extension.ts` lines 128–143:
```typescript
await handleProbeToken(
  ...,
  (name, options, cancellation) =>
    vscode.lm.invokeTool(name, { ... }, cancellation as vscode.CancellationToken),
  _token,  // <-- chat request CancellationToken
  ...
);
```

`handleProbeToken` captures this as `cancellation` and passes it to all 4 `invokeToolFn` calls (`probeToken.ts:413`). When the chat handler eventually returns, VS Code fires `_token` cancellation, potentially triggering additional MCP teardown.

#### 4. The probe is marked THROWAWAY and its measurement is complete

`src/probeToken.ts` line 1:
```typescript
// probeToken.ts — Throwaway diagnostic: does toolInvocationToken survive an await?
```

Per `history.md` (Probe Schema Diagnostics section): T1 fails identically to T2/T3/T4 (~5-7s). The measurement is done. The probe has no remaining diagnostic value.

### Hypotheses (not VS Code-source-verified)

- VS Code applies a crash-count gate or exponential back-off after N consecutive MCP failures and stops attempting reconnects for the session lifetime. The probe reliably triggers this by producing exactly 4 failures in rapid succession.
- The `_token` cancellation callback may also signal VS Code to tear down any in-flight MCP state established during the handler.

### Safe Immediate Recovery (no reboot required)

**`Developer: Reload Window`** (`Ctrl+Shift+P` → type "Reload Window").

This resets the VS Code extension host and all in-memory MCP client state. The ICM MCP server should reconnect on the next `invokeTool` call. Machine reboot is not required and should not be the recommended recovery.

**Caveat:** If VS Code's MCP session failure also caused server-side auth token invalidation at `icm-mcp-prod.azure-api.net`, a fresh sign-in prompt may appear after reload. This is the designed behavior for authentication failure — not a sign that the reload failed.

### Recommended Code Change

Remove `/probe-token` entirely. It is complete, throwaway, and actively dangerous.

**Files to change:**

1. **`package.json`** — remove `probe-token` from `chatParticipants[0].commands` array. This makes the command inert without reinstalling the extension.

2. **`src/extension.ts`** — remove the `probe-token` branch (lines ~125–144):
   ```typescript
   if (request.command === 'probe-token') { ... }
   ```
   Also remove the `handleProbeToken` import.

3. **Delete `src/probeToken.ts`** — entire file. The `buildSchemaDump`/`schemaVerdict` helpers are unused outside this file.

4. **Delete `test/probeToken.test.js`** — remove the associated test file.

**Test count impact:** Will reduce test count by the number of probe-token tests. Run `npm test` to confirm clean pass after deletion.

### Future Diagnostic Probes — Binding Rule

Any future diagnostic that calls `vscode.lm.invokeTool` against a live MCP endpoint must:
- Call at most once per invocation, OR
- Stop on first failure (no unconditional multi-attempt loops), OR
- Target a mock/stub, never a live MCP endpoint registered in `mcp.json`.

Calling `invokeTool` N times unconditionally against a real remote MCP server in a production extension is not safe.

