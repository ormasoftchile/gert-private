# C# Governance Parity Design — GERT Web Runtime

**Author:** Don (Backend Dev)  
**Date:** 2026-06-03T21:50:51.668-04:00  
**Status:** Draft — internal review required before implementation

---

## Purpose

The C# web runtime (A6: App Service + BackgroundService) must be **governance-equivalent** to the Go
reference implementation. Governance-equivalent means: identical pipeline stages, identical contract
guarantees, identical audit-trail output, and identical rejection behaviour for any policy input.

This document specifies the C# runtime design in enough detail that an implementer can build it
without reading Go source — while still being bound to the Go reference for all behavioural questions.

**Non-goals.** This document does not cover Azure infra provisioning (John's domain), frontend
consumption (Leslie's domain), or webhook routing (David's domain).

---

## 1. Pipeline Mapping

The Go pipeline is the normative reference. Every stage must be reproduced in C# with equivalent
contracts. No stage may be skipped, reordered, or merged.

### 1.1 Go Pipeline (Reference)

```
Parser.Parse(ctx, r, opts) → *ParsedRunbook
  ↓
Planner.Plan(ctx, rb, opts) → *ExecutionPlan
  ↓
Runtime.Start(ctx, plan, opts) → RunHandle
  ↓
RunHandle.Next(ctx) → *StepResult   ← governance fires here, every step
  ↓  (repeat until io.EOF)
RunHandle.Next returns io.EOF       ← run terminal
```

`RunHandle.Next()` is the **governance boundary**. Before every step executes:

1. Command allowlist/denylist evaluated (cli + tool steps)
2. Environment variable blocking applied
3. Interactive gate checked; if approval or user input is required, execution suspends inside `NextAsync()` until external input is durably recorded
4. JSONL trace line written **synchronously** to the trace file
5. Step dispatched only after trace write completes

### 1.2 C# Pipeline — Class and Method Mapping

| Go | C# Class | C# Method Signature |
|---|---|---|
| `Parser` interface | `IRunbookParser` | `Task<ParsedRunbook> ParseAsync(Stream yaml, ParseOptions opts, CancellationToken ct)` |
| `Planner` interface | `IRunbookPlanner` | `Task<ExecutionPlan> PlanAsync(ParsedRunbook rb, PlanOptions opts, CancellationToken ct)` |
| `Runtime` interface | `IGertRuntime` | `Task<IRunHandle> StartAsync(ExecutionPlan plan, RunOptions opts, CancellationToken ct)` |
| `Runtime.Resume` | `IGertRuntime` | `Task<IRunHandle> ResumeAsync(string runId, RunOptions opts, CancellationToken ct)` |
| `RunHandle` interface | `IRunHandle` | `Task<StepResult?> NextAsync(CancellationToken ct)` (null = EOF) |
| `RunHandle.Approve` | `IRunHandle` | `Task ApproveAsync(ApprovalDecision decision, CancellationToken ct)` |
| `RunHandle.SubmitEvidence` | `IRunHandle` | `Task SubmitEvidenceAsync(string stepId, IReadOnlyDictionary<string, EvidenceValue> evidence, CancellationToken ct)` |
| `RunHandle.Cancel` | `IRunHandle` | `Task CancelAsync(string reason, CancellationToken ct)` |
| `RunHandle.State` | `IRunHandle` | `RunState State { get; }` |
| `RunHandle.Events` | `IRunHandle` | `IAsyncEnumerable<GertEvent> Events { get; }` |

### 1.3 C# Dependency Direction

```
IRunbookParser
    ↓
IRunbookPlanner
    ↓
IGertRuntime
    ↓
IRunHandle          ← governance engine lives here; no component above this imports it
    ↓
IGovernanceLayer    ← internal to runtime package; not injectable from adapters
IStepRunner         ← internal sealed; not visible outside runtime assembly
IRunStore           ← trace writer; called synchronously before NextAsync returns
```

The rule from the Go architecture applies: **dependencies flow inward toward the Core Domain.**
No adapter (`BackgroundService`, API controller) may import `IStepRunner` or any concrete
step-runner type. They interact only via `IRunHandle`.

---

## 2. IRunHandle Interface Definition

```csharp
/// <summary>
/// Represents a live execution handle for a single GERT runbook run.
/// The governance engine fires inside NextAsync() before every step dispatch.
/// </summary>
/// <remarks>
/// CONTRACT: NextAsync() is not thread-safe. Callers must not invoke it
/// concurrently for the same run. Multiple sequential callers are permitted
/// (re-entrant per run, never concurrent per run — mirrors Go RunHandle invariant).
///
/// NextAsync() returns null when the run has no more steps (equivalent to
/// Go's io.EOF return). After null is returned, no further calls are valid.
/// </remarks>
public interface IRunHandle : IAsyncDisposable
{
    /// <summary>
    /// Advances the run by one step. Blocks until the step completes or
    /// is suspended waiting for user input, evidence, or approval.
    /// </summary>
    /// <returns>
    /// A <see cref="StepResult"/> for the step that just executed, or
    /// <see langword="null"/> when the run has no more steps.
    /// </returns>
    /// <remarks>
    /// Governance pre-flight fires inside this method before every dispatch:
    /// <list type="number">
    ///   <item>Command allowlist/denylist check</item>
    ///   <item>Environment variable blocking</item>
    ///   <item>Interactive gate evaluation (user input or approval; suspends if required)</item>
    ///   <item>JSONL trace write (synchronous — must complete before step dispatches)</item>
    ///   <item>Step dispatch</item>
    /// </list>
    /// Any governance violation throws <see cref="GovernanceBlockedException"/>.
    /// </remarks>
    Task<StepResult?> NextAsync(CancellationToken ct = default);

    /// <summary>
    /// Records an approval decision for the step currently suspended at an
    /// approval gate. Must only be called while NextAsync is suspended.
    /// </summary>
    Task ApproveAsync(ApprovalDecision decision, CancellationToken ct = default);

    /// <summary>
    /// Records evidence for interactive steps that collect structured payloads.
    /// For decision: route name.
    /// For collector: multi-field form data and artifact attachments.
    /// User-input and approval waits flow through IUserInputGate / IApprovalGate instead.
    /// </summary>
    Task SubmitEvidenceAsync(
        string stepId,
        IReadOnlyDictionary<string, EvidenceValue> evidence,
        CancellationToken ct = default);

    /// <summary>
    /// Requests a graceful stop of the run. Emits run/cancelled followed by
    /// run/completed with outcome "cancelled" before returning.
    /// </summary>
    Task CancelAsync(string reason, CancellationToken ct = default);

    /// <summary>
    /// Returns a read-only snapshot of the current mutable run state.
    /// Safe to call from any thread (snapshot is immutable).
    /// </summary>
    RunState State { get; }

    /// <summary>
    /// Structured events emitted during execution, in total order per run.
    /// Events are best-effort to the caller; the JSONL trace is authoritative.
    /// </summary>
    IAsyncEnumerable<GertEvent> Events { get; }
}
```

### 2.1 Supporting Types

```csharp
public sealed record StepResult(
    string StepId,
    StepOutcome Outcome,           // Success | Skipped | Failed | Suspended
    TimeSpan Duration,
    int? ExitCode,                 // null for non-cli/tool steps
    IReadOnlyDictionary<string, string> CapturedVars);

public enum StepOutcome { Success, Skipped, Failed, Suspended }
```

---

## User Input Steps and Approval Steps — Runtime Contract

This section closes the interactive-wait gap for the C# runtime. Both approval waits (third-party
actor) and user-input waits (portal subject) are **governance events inside `IRunHandle.NextAsync()`**,
not adapter-side callbacks and not direct `IStepRunner` concerns. Choice selection, free-form text,
confirmation, file upload, and multi-field forms are the same primitive: suspend, await user input,
resume.

### 1. IUserInputGate interface (C#)

```csharp
public interface IUserInputGate
{
    Task<UserInputResponse> AwaitUserInputAsync(
        string runId,
        string stepId,
        UserInputRequest request,
        CancellationToken cancellationToken);
}

public record UserInputRequest(
    string StepLabel,
    UserInputKind Kind,
    string? Prompt = null,
    IReadOnlyList<ChoiceOption>? Options = null,
    JsonDocument? Schema = null);

public enum UserInputKind { Choice, Text, Confirmation, FileUpload, Form }

public record UserInputResponse(
    string StepId,
    UserInputKind Kind,
    string? SelectedKey = null,
    string? TextValue = null,
    Uri? FileReference = null,
    JsonDocument? FormData = null);

public record ChoiceOption(string Key, string Label, string? Description = null);
```

**A6 implementation shape (App Service + `BackgroundService`):**
- Authoritative state lives in the run database, not in SignalR.
- `RunHandleImpl.NextAsync()` writes a pending interaction row keyed by `(run_id, step_id)` with `kind`, `prompt`, timeout, and the request payload needed for portal rendering (`options` for `Choice`, JSON schema for `Form`, upload metadata for `FileUpload`).
- The portal API `POST /runs/{runId}/steps/{stepId}/input` validates that the caller is the run subject and that the submitted payload matches the requested `UserInputKind`, then writes the normalized response payload (`selected_key`, `text_value`, `file_reference`, or `form_data`), `submitted_at`, and `status = 'submitted'` to that row.
- `DatabaseUserInputGate` in the worker short-polls that row (with backoff) or consumes a lightweight wake-up message after the API write. The database row is the source of truth; any queue, in-memory `TaskCompletionSource`, or SignalR notification is only an accelerator.
- SignalR may be used to push the request to the browser and reflect that input was accepted, but SignalR is UX-only and must not be required for correctness. The worker resumes on any valid `UserInputResponse`, not only a choice selection.

**A8 implementation shape (Durable Functions):**
- `DurableUserInputGate` writes the same pending interaction record for portal read models, then waits with `context.WaitForExternalEvent<UserInputResponse>($"user-input:{stepId}")`.
- The portal API validates and persists the response payload, then raises the Durable external event for the orchestration instance.
- The orchestration resumes on any valid `UserInputResponse`; no `IRunHandle` or governance code changes are required.

**How the worker knows user input was submitted:**
- The portal API POST handler is the writer of record.
- In A6, the worker observes the database state change (polling or queue wake-up followed by re-read).
- In A8, the orchestrator observes the matching external event after the same durable write completes.

### 2. IApprovalGate interface (formalized)

```csharp
public interface IApprovalGate
{
    Task<ApprovalDecision> AwaitApprovalAsync(
        string runId,
        string stepId,
        ApprovalRequest request,
        CancellationToken cancellationToken);
}

public enum ApprovalDecision { Approved, Rejected, TimedOut }

public sealed record ApprovalRequest(
    string RunId,
    string StepId,
    IReadOnlyList<string> ApproverRoles,
    int MinApprovals,
    TimeSpan? Timeout,
    string Message);
```

**A6 implementation shape:**
- `DatabaseApprovalGate` or `ServiceBusApprovalGate` may keep the worker alive, but the durable approval decision must still be recorded in the run database before `AwaitApprovalAsync()` completes.
- The approver-facing API callback writes the approval state (`approved`, `rejected`, or timeout metadata) and the worker re-reads until a terminal decision is visible.
- If Service Bus is used, it should be a wake-up mechanism or correlation channel, not the only durable record.

**A8 implementation shape:**
- `DurableApprovalGate` maps directly to `WaitForExternalEvent<ApprovalDecision>($"approval:{stepId}")`.
- The approver callback persists the decision, then signals the orchestration instance.
- On timeout, the orchestrator returns `ApprovalDecision.TimedOut` without changing the `IRunHandle` contract.

### 3. Governance pipeline integration

Both interactive gates fire **inside `IRunHandle.NextAsync()`**. The JSONL trace must be written
**before** the runtime waits, and again **after** the response is received. Adapters must not emit
synthetic gate events on behalf of the runtime. All `user_input_requested` and `user_input_received`
payloads must include `kind` so auditors can reconstruct whether the runtime asked for a choice, free
text, confirmation, file upload, or form submission.

**User-input step sequence:**
1. Governance fires (authorization, policy, audit pre-check)
2. JSONL trace written: `{ "event": "user_input_requested", "stepId": "...", "kind": "Choice|Text|Confirmation|FileUpload|Form", ... }`
3. `IUserInputGate.AwaitUserInputAsync(...)` suspends
4. Portal user submits input via portal API
5. JSONL trace written: `{ "event": "user_input_received", "stepId": "...", "kind": "Choice|Text|Confirmation|FileUpload|Form", ... }`
6. `NextAsync()` resumes after any valid `UserInputResponse` and execution continues

**Approval step sequence:**
1. Governance fires (authorization, policy, audit pre-check)
2. JSONL trace written: `{ "event": "approval_requested", "stepId": "...", ... }`
3. `IApprovalGate.AwaitApprovalAsync(...)` suspends
4. Third party approves or rejects via callback/API
5. JSONL trace written: `{ "event": "approval_received", "stepId": "...", "decision": "Approved|Rejected|TimedOut" }`
6. `NextAsync()` returns and execution continues or terminates per policy

**Runtime rule:** `IRunHandle.NextAsync()` owns the full interactive wait lifecycle. `BackgroundService`,
controllers, orchestrators, and portal hubs may publish/read state, but they must not decide that a
step has completed without the gate interface returning to `RunHandleImpl`.

### 4. RED LINE enforcement

- `internal sealed IStepRunner` must **not** bypass `IUserInputGate` or `IApprovalGate`.
- Interactive waits belong in `RunHandleImpl.NextAsync()` / governance flow, never in adapter code and never inside concrete step runners.
- The existing Roslyn analyzers (`GERT0001`, `GERT0002`) enforce `IStepRunner` inaccessibility and RE2 usage, which is necessary but not sufficient.
- **Current gap:** the analyzers do **not yet** prove that any user-input resumption path is forced through `IUserInputGate` / `IApprovalGate` and back into `IRunHandle.NextAsync()` rather than from some parallel adapter path. That requires either a new analyzer (forbidden gate references and forbidden resume helpers outside `RunHandleImpl` / governance layer) or an architecture test that fails the build.

### 5. Timeout and cancellation

- **User-input timeout:** if a user-input step timeout elapses, `AwaitUserInputAsync()` ends via the supplied `CancellationToken`. `RunHandleImpl` must write a terminal interactive trace event (for example `user_input_timed_out`) and transition the runbook to `TimedOut` before returning control.
- **Approval timeout:** `AwaitApprovalAsync()` returns `ApprovalDecision.TimedOut`, then `RunHandleImpl` writes the post-wait trace event and transitions the run according to policy.
- **Portal session drop / browser navigation:** the wait persists because the pending interaction is stored server-side. Losing the browser session must not cancel the wait.
- **Persistence window for user input:** user-input waits remain pending until the earlier of (a) the step-level timeout, (b) the run-level cancellation token, or (c) an explicit operator/admin cancellation. For short-lived interactive steps, the recommended default if the runbook omits a timeout is 15 minutes.
- **Reconnect semantics:** on reload, the portal queries the run state, sees the pending interaction row, and re-renders the same request payload. SignalR disconnects do not change runtime state.

---

## 4. IStepRunner Enforcement (The RED LINE)

**Red line:** No per-step execution that bypasses `IRunHandle.NextAsync()`. If a step runner is
called directly (e.g., from a Durable Activity without going through `IRunHandle`), governance
does not fire and the audit trail is not produced. This is a structural security violation.

### 4.1 Enforcement via `internal sealed`

```csharp
// In assembly: Gert.Runtime.Core

// INTERNAL — not visible outside the assembly.
// No adapter, controller, or Activity may inject or call this directly.
internal interface IStepRunner
{
    Task<StepResult> RunAsync(ResolvedStep step, RunContext ctx, CancellationToken ct);
}

// Concrete implementations are also internal sealed:
internal sealed class CliStepRunner   : IStepRunner { ... }
internal sealed class ToolStepRunner  : IStepRunner { ... }
internal sealed class ManualStepRunner: IStepRunner { ... }
internal sealed class InvokeStepRunner: IStepRunner { ... }
```

`IStepRunner` and all concrete runners are in assembly `Gert.Runtime.Core` with no
`[assembly: InternalsVisibleTo(...)]` grants to adapter assemblies. The only public surface
is `IRunHandle`. Governance fires in `RunHandleImpl.NextAsync()`, which is the sole caller
of `IStepRunner.RunAsync()`.

### 4.2 Structural Guarantee

```
Gert.Runtime.Core (assembly boundary)
├── public interface IRunHandle         ← the only door into execution
├── public interface IGertRuntime       ← factory for IRunHandle
├── internal interface IStepRunner      ← step dispatch, invisible outside
├── internal sealed CliStepRunner
├── internal sealed ToolStepRunner
└── internal sealed RunHandleImpl       ← calls GovernanceLayer then IStepRunner
                                           in that order, every time

Gert.Worker (BackgroundService assembly)
└── depends only on: IRunHandle, IGertRuntime, IRunbookParser, IRunbookPlanner
    CANNOT import: IStepRunner, CliStepRunner, ToolStepRunner
```

### 4.3 Durable Functions: Correct vs. Incorrect Pattern

**CORRECT (A8, if adopted):**
```csharp
// Orchestrator calls IRunHandle.NextAsync() — governance fires inside
[FunctionName("RunbookOrchestrator")]
public async Task RunOrchestrator(IDurableOrchestrationContext context)
{
    // IRunHandle is reconstructed from durable entity state
    // NextAsync() fires governance pre-flight, writes trace, dispatches step
    var result = await context.CallActivityAsync<StepResult>("AdvanceStep", input);
}

// Activity wraps IRunHandle.NextAsync() — not the step runner directly
[FunctionName("AdvanceStep")]
public async Task<StepResult> AdvanceStep([ActivityTrigger] AdvanceInput input)
{
    var handle = await _runtime.ResumeAsync(input.RunId, ...);
    return await handle.NextAsync();  // ✅ governance fires here
}
```

**FORBIDDEN (RED LINE):**
```csharp
// Activity calls IStepRunner directly — governance does NOT fire
[FunctionName("RunStep")]  // 🚨 DO NOT DO THIS
public async Task<StepResult> RunStep([ActivityTrigger] StepInput input)
{
    return await _stepRunner.RunAsync(input.Step, ...);  // 🚨 bypasses governance
}
```

The `internal` access modifier makes the forbidden pattern a **compile error**, not a code review
finding.

---

## 5. RE2 Compliance

### 5.1 The Risk

Go's `regexp` package implements RE2 semantics (no lookahead, no backreferences, guaranteed linear
time). The GERT governance layer uses RE2 patterns for:

- Output redaction (`redact[*].pattern`)
- Environment variable blocking (`deny_env_vars[*]` — glob-style, but evaluated via RE2 anchoring)

C# `System.Text.RegularExpressions` is a superset of RE2. It supports lookahead, lookbehind, and
backreferences — constructs that are **undefined or different** in RE2. A redaction rule authored
in Go may silently produce different results in C# if the C# engine takes a different match path.

**This is a governance security risk:** a pattern that redacts `SECRET=abc123` in Go may fail to
redact it in C# if the engines interpret the pattern differently.

### 5.2 Resolution: Google.RE2 NuGet

**All redaction pattern evaluation in the C# runtime MUST use `Google.RE2` (NuGet package:
`Google.Re2`), not `System.Text.RegularExpressions`.**

```xml
<!-- Gert.Runtime.Core.csproj -->
<PackageReference Include="Google.Re2" Version="1.*" />
```

```csharp
using Google.Re2;

internal sealed class RedactionEngine
{
    private readonly IReadOnlyList<CompiledRule> _rules;

    internal sealed record CompiledRule(Regex Pattern, string Replacement);

    public RedactionEngine(IReadOnlyList<RedactRule> rules)
    {
        // RE2 constructor throws on invalid patterns — fail fast at startup.
        _rules = rules
            .Select(r => new CompiledRule(new Regex(r.Pattern), r.Replace))
            .ToList();
    }

    /// <summary>
    /// Applies all redaction rules in declaration order.
    /// Uses RE2 engine — guaranteed linear time, RE2 semantics only.
    /// </summary>
    public string Apply(string input)
    {
        var result = input;
        foreach (var rule in _rules)
        {
            result = rule.Pattern.Replace(result, rule.Replacement);
        }
        return result;
    }
}
```

### 5.3 Patterns at Risk

The following pattern features are **valid in C# regex but undefined or rejected by RE2**.
Any redaction rule containing these must be caught at startup (RE2 constructor will throw):

| C# Feature | RE2 Behaviour | Risk Level |
|---|---|---|
| Lookahead `(?=...)` | Rejected — RE2 does not support | HIGH — runtime error |
| Lookbehind `(?<=...)` | Rejected | HIGH — runtime error |
| Named backreferences `\k<name>` | Rejected | HIGH — runtime error |
| Possessive quantifiers `*+` | Rejected | HIGH — runtime error |
| `\b` word boundary | Supported by RE2 | LOW |
| Named groups `(?P<name>...)` | RE2 syntax (Go) vs `(?<name>...)` C# | MEDIUM — test required |

**Named group syntax difference:** Go RE2 uses `(?P<name>...)` syntax; `Google.Re2` NuGet uses the
same syntax as the underlying RE2 library. Validate named group patterns against canonical test
vectors (see Section 7).

### 5.4 Integration Points

`RedactionEngine` is called in exactly two places inside `RunHandleImpl`:

1. After a `cli` or `tool` step completes: applied to captured stdout/stderr before variable store write
2. Before `IRunStore.WriteTraceAsync()`: applied to any field in the step result that is tagged for redaction

Both call sites are inside `Gert.Runtime.Core` (internal). No adapter touches redaction output.

---

## 6. Template Engine Gap

### 6.1 The Problem

Go `text/template` is GERT's expression language for:

- Variable interpolation in step parameters (`{{.vars.deployment_id}}`)
- Branch predicates (`{{eq .vars.env "prod"}}`)
- Step output captures
- Approval gate message formatting

**There is no C# equivalent of Go `text/template`.** Any C# port must be validated against
canonical test vectors. Advanced template features (pipelines, custom functions, `range`, `with`)
may not be trivially portable.

### 6.2 Strategy

**Option A — Port the evaluator (recommended for A6 MVP):**
Implement a minimal subset of `text/template` sufficient for runbook use. The subset must cover:

| Feature | Required for MVP | Notes |
|---|---|---|
| `{{.vars.name}}` dot access | YES | Simple property traversal |
| `{{eq .vars.x "value"}}` comparisons | YES | Used in branch predicates |
| `{{and}} {{or}} {{not}}` | YES | Logical combinators |
| `{{if}} {{else}} {{end}}` | YES | Branch control |
| `{{range}} {{end}}` | NO (v2) | Iterate step expansion |
| `{{with}} {{end}}` | NO (v2) | Scoped binding |
| Custom functions (sprig-style) | NO (v2) | Future extension |
| Pipelines `{{.x \| funcname}}` | LIMITED | Only pipeline to registered funcs |

A lightweight recursive evaluator over a parsed AST is the recommended approach.
**Do not use a general-purpose template library** (Handlebars, Scriban, Fluid) as the
first choice — their semantics differ from `text/template` in edge cases that matter for
branch predicates.

**Option B — Shared evaluator via WASM (fallback):**
Compile the Go template evaluator to WASM, call from C# via Wasmtime. Guarantees parity but
adds ~10ms cold-start and WASM runtime dependency. Use only if Option A fails parity tests.

### 6.3 Required Canonical Test Vectors

The following template expressions must be validated before production use. Each vector is defined
in the JSON test vector contract (Section 7).

```
TV-TMPL-001: {{.vars.name}} with string value
TV-TMPL-002: {{.vars.count}} with integer value
TV-TMPL-003: {{eq .vars.env "prod"}} → true
TV-TMPL-004: {{eq .vars.env "prod"}} → false
TV-TMPL-005: {{and (eq .vars.env "prod") (eq .vars.region "us-east-1")}}
TV-TMPL-006: {{if eq .vars.level "high"}}HIGH{{else}}LOW{{end}}
TV-TMPL-007: Nested dot: {{.vars.config.timeout}}
TV-TMPL-008: Missing variable → empty string (Go text/template default)
TV-TMPL-009: RE2 named group in template context: {{.vars.match}}
TV-TMPL-010: Integer comparison: {{lt .vars.count 10}}
```

---

## 7. Governance Test Vector Contract

### 7.1 Purpose

Language-neutral JSON test vectors define the **expected governance outcome** for a given input.
The C# runtime must produce identical outcomes to the Go runtime for every vector before any
production use.

Vectors are stored in `test/vectors/governance/` and run against both runtimes in CI.

### 7.2 JSON Schema

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://gert.ormasoft.com/schemas/governance-test-vector/v1",
  "title": "GertGovernanceTestVector",
  "type": "object",
  "required": ["id", "description", "input", "expected"],
  "properties": {
    "id": {
      "type": "string",
      "pattern": "^TV-[A-Z]+-[0-9]{3}$",
      "description": "Stable vector ID. Never reuse a retired ID."
    },
    "description": {
      "type": "string",
      "description": "Human-readable explanation of what this vector tests."
    },
    "input": {
      "type": "object",
      "required": ["runbook_fragment", "governance_policy", "step_context"],
      "properties": {
        "runbook_fragment": {
          "type": "string",
          "description": "Minimal YAML snippet containing the step(s) under test."
        },
        "governance_policy": {
          "type": "object",
          "description": "Governance policy block as it would appear in the runbook meta.",
          "properties": {
            "allowed_commands": { "type": "array", "items": { "type": "string" } },
            "denied_commands":  { "type": "array", "items": { "type": "string" } },
            "deny_env_vars":    { "type": "array", "items": { "type": "string" } },
            "redact": {
              "type": "array",
              "items": {
                "type": "object",
                "required": ["pattern", "replace"],
                "properties": {
                  "pattern": { "type": "string" },
                  "replace": { "type": "string" }
                }
              }
            },
            "approval_gates": {
              "type": "array",
              "items": {
                "type": "object",
                "required": ["step_id", "roles", "min_approvals"],
                "properties": {
                  "step_id":       { "type": "string" },
                  "roles":         { "type": "array", "items": { "type": "string" } },
                  "min_approvals": { "type": "integer", "minimum": 1 },
                  "timeout_seconds": { "type": "integer" }
                }
              }
            }
          }
        },
        "step_context": {
          "type": "object",
          "description": "Runtime context for the step under test.",
          "properties": {
            "step_id":      { "type": "string" },
            "command":      { "type": "string" },
            "env_vars":     { "type": "object", "additionalProperties": { "type": "string" } },
            "captured_stdout": { "type": "string" },
            "vars":         { "type": "object", "additionalProperties": true }
          }
        }
      }
    },
    "expected": {
      "type": "object",
      "required": ["verdict"],
      "properties": {
        "verdict": {
          "type": "string",
          "enum": ["allowed", "blocked", "approval_required", "approval_granted", "approval_rejected"],
          "description": "Governance verdict for the step."
        },
        "block_reason": {
          "type": "string",
          "enum": ["not_in_allowlist", "in_denylist", "approval_timeout"],
          "description": "Present only when verdict is 'blocked'."
        },
        "redacted_stdout": {
          "type": "string",
          "description": "Expected stdout after redaction rules applied. Omit if no redaction."
        },
        "env_vars_blocked": {
          "type": "array",
          "items": { "type": "string" },
          "description": "Names of environment variables expected to be stripped."
        },
        "trace_events_emitted": {
          "type": "array",
          "items": { "type": "string" },
          "description": "Ordered list of event kinds expected in the JSONL trace for this step."
        },
        "template_output": {
          "type": "string",
          "description": "Expected output of template evaluation (for TV-TMPL-* vectors)."
        }
      }
    },
    "tags": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Labels: allowlist, denylist, redaction, approval, template, env_blocking."
    }
  }
}
```

### 7.3 Required Vector Coverage

The following vector categories must have complete coverage before production:

| Category | Vector IDs | Minimum Count |
|---|---|---|
| Command allowlist — allowed | TV-ALLOW-001..010 | 5 |
| Command allowlist — blocked | TV-ALLOW-011..020 | 5 |
| Command denylist | TV-DENY-001..010 | 5 |
| Env var blocking — glob patterns | TV-ENV-001..010 | 8 |
| Redaction — basic pattern | TV-REDACT-001..010 | 8 |
| Redaction — named groups | TV-REDACT-011..020 | 4 |
| Redaction — multiple rules | TV-REDACT-021..030 | 4 |
| Approval — gate entered | TV-APPROV-001..005 | 3 |
| Approval — granted | TV-APPROV-006..010 | 3 |
| Approval — rejected | TV-APPROV-011..015 | 3 |
| Approval — timeout | TV-APPROV-016..020 | 2 |
| Template evaluation | TV-TMPL-001..020 | 10 |
| **Total minimum** | | **60** |

---

## 8. JSONL Trace Contract

### 8.1 Trace Schema

Every JSONL line in the trace file is a serialized `GertEvent` with a mandatory envelope.
The C# runtime must produce output conforming to the normative schema defined in §07-runtime-events.tex.

**Mandatory envelope fields (every event):**

```json
{
  "event_id":    "<UUID v4>",
  "run_id":      "<UUID>",
  "runbook_id":  "<string>",
  "timestamp":   "<RFC3339 with microseconds — e.g., 2026-06-03T21:50:51.668000Z>",
  "kind":        "<string>",
  "sequence":    <int64>,
  "payload":     { }
}
```

### 8.2 C# Trace Writer Contract

```csharp
/// <summary>
/// Authoritative JSONL trace writer. Every call is synchronous with respect
/// to the caller — WriteTraceAsync must complete before NextAsync dispatches
/// the step. Fire-and-forget is forbidden.
/// </summary>
public interface IRunStore
{
    /// <summary>
    /// Appends a single event line to the JSONL trace for the given run.
    /// MUST complete (flush to durable storage) before returning.
    /// Implementations MUST NOT buffer without flushing (no fire-and-forget).
    /// </summary>
    /// <exception cref="TraceWriteException">
    /// Thrown if the write fails. RunHandleImpl treats this as a fatal run error
    /// and calls CancelAsync before propagating.
    /// </exception>
    Task WriteTraceAsync(string runId, GertEvent evt, CancellationToken ct = default);

    /// <summary>
    /// Returns the next sequence number for a run. Monotonically increasing,
    /// scoped to run_id, starting from 0. Thread-safe.
    /// </summary>
    Task<long> NextSequenceAsync(string runId, CancellationToken ct = default);

    /// <summary>
    /// Checkpoints the run state after each step completion.
    /// Enables pause/resume across worker restarts.
    /// </summary>
    Task SaveRunStateAsync(string runId, RunState state, CancellationToken ct = default);

    /// <summary>
    /// Loads a previously saved run state for resume.
    /// Returns null if no checkpoint exists.
    /// </summary>
    Task<RunState?> LoadRunStateAsync(string runId, CancellationToken ct = default);
}
```

### 8.3 Synchronous Write Guarantee

The synchronous write invariant is enforced by the call order in `RunHandleImpl.NextAsync()`:

```csharp
// Inside RunHandleImpl.NextAsync() — simplified
private async Task<StepResult?> NextAsync(CancellationToken ct)
{
    var step = _plan.NextStep();
    if (step is null) return null;

    // 1. Governance pre-flight
    await _governance.CheckAsync(step, _state, ct);

    // 2. Emit step/started event
    var startedEvent = BuildStepStartedEvent(step);

    // 3. Write trace SYNCHRONOUSLY — must complete before dispatch
    await _runStore.WriteTraceAsync(_runId, startedEvent, ct);
    // ↑ If this throws, the step does NOT execute. Run is cancelled.

    // 4. Dispatch step only after trace write confirms
    var result = await _stepRunner.RunAsync(step, _context, ct);

    // 5. Write step/completed or step/failed trace line
    await _runStore.WriteTraceAsync(_runId, BuildStepResultEvent(step, result), ct);

    return result;
}
```

**No buffering, no background tasks, no `_ = WriteTraceAsync(...)` fire-and-forget.** Any
implementation that defers the trace write to a background queue violates the contract and
invalidates the audit trail.

### 8.4 Sequence Number Integrity

- Sequence numbers are assigned by `IRunStore.NextSequenceAsync()` immediately before trace write
- They are monotonically increasing within a run, starting from 0
- In BackgroundService (A6): backed by an atomic counter in the run state checkpoint
- In Durable Entity (A8): tracked in entity state, incremented only via entity operations
- **Never increment sequence numbers inline in the orchestrator** (Durable replay would double-increment)

### 8.5 Trace Verification in CI

The following checks run against every JSONL trace produced by the C# runtime in CI:

1. **Schema conformance:** Every line parses against the `GertEvent` JSON Schema
2. **Sequence continuity:** Sequence numbers are gapless (0, 1, 2, ... N)
3. **First event:** `kind == "run/started"` at `sequence == 0`
4. **Terminal event:** Last event is `kind == "run/completed"` or `kind == "run/cancelled"`
5. **Redaction applied:** No pattern from `governance.redact` appears in any trace field
6. **Go replay compatibility:** The Go `gert replay` command successfully replays the C# trace

Check 6 is the strongest parity guarantee: if Go can replay a C# trace, the trace is structurally
and semantically correct.

---

## 9. Validation Gate

The C# runtime must pass all of the following checks before it may handle production traffic.

### 9.1 Required Gates

| Gate | Criterion | Enforced By |
|---|---|---|
| **G-01** Test vector coverage | All 60 minimum test vectors pass | CI (xUnit) |
| **G-02** RE2 parity | All TV-REDACT-* vectors produce identical output in Go and C# | CI (go test + xUnit compared) |
| **G-03** Template parity | All TV-TMPL-* vectors produce identical output in Go and C# | CI |
| **G-04** Trace schema conformance | 100% of C#-generated JSONL lines pass JSON Schema validation | CI |
| **G-05** Sequence continuity | Zero gaps in sequence numbers across 1,000 synthetic runs | CI |
| **G-06** Go replay compatibility | Go `gert replay` succeeds on 100% of C#-generated traces | CI |
| **G-07** Synchronous write proof | Trace write completes before step dispatch in 100% of test runs | Integration test |
| **G-08** IStepRunner inaccessibility | `IStepRunner` is not in any public API surface (Roslyn analyzer check) | CI (build) |
| **G-09** RE2 package enforced | No `System.Text.RegularExpressions` usage in `Gert.Runtime.Core` | CI (Roslyn analyzer) |
| **G-10** Interactive gates registered | `IApprovalGate` and `IUserInputGate` are registered in DI container before runtime start | Integration test |

### 9.2 Roslyn Analyzer Rules

Two custom Roslyn analyzers enforce G-08 and G-09 at compile time:

**GERT0001 — StepRunnerAccessViolation:**
```
Error: IStepRunner or its implementations must not be referenced outside Gert.Runtime.Core.
```

**GERT0002 — ForbiddenRegexNamespace:**
```
Error: System.Text.RegularExpressions must not be used in Gert.Runtime.Core.
       Use Google.Re2.Regex instead.
```

Both analyzers ship in `Gert.Analyzers` and are referenced by `Gert.Runtime.Core.csproj` as
analyzer-only package references.

### 9.3 Gate Status Tracking

Gate status is tracked per release candidate in `design/web-platform/validation-gate-status.md`
(to be created when the first RC is cut). No RC may be promoted to production with any gate
in a non-passing state.

---

## 10. Open Questions

| # | Question | Owner | Target |
|---|---|---|---|
| OQ-1 | Template engine: Option A (port) vs Option B (WASM)? Needs spike to assess effort. | Don | Before A6 implementation starts |
| OQ-2 | Service Bus approval reply queue: one queue per run or one tenant-wide queue with correlation? | Don + John | Before approval gate implementation |
| OQ-3 | TV-TMPL vector suite: who authors the canonical Go outputs? Need a Go test harness that emits expected values. | Don | Sprint 1 |
| OQ-4 | Sequence counter in A6: in-memory atomic + checkpoint, or always read from RunStore? Failure recovery implication. | Don | Sprint 1 |
| OQ-5 | Named RE2 group syntax: `Google.Re2` NuGet uses RE2 C library conventions — confirm `(?P<name>...)` syntax works. | Don | Before redaction implementation |
| OQ-6 | File upload inputs need hard limits and storage rules: max size per file, allowed content types, blob container layout, and retention/quarantine strategy before `UserInputKind.FileUpload` ships. | Don + John | Before file-upload input implementation |

---

## References

- `design/gert/sections/02-architecture.tex` — Go pipeline, interface definitions, key invariants
- `design/gert/sections/07-runtime-events.tex` — Event envelope schema, event catalog, JSONL trace contract
- `design/gert/sections/12-governance-policy.tex` — Governance primitives (allowlist, denylist, env blocking, redaction, approval gates)
- `.squad/agents/don/history.md` — Prior analysis: execution adapter patterns, C# runtime option analysis
- `.squad/decisions.md` — Governance red lines (Active Decisions)
- `Google.Re2` NuGet: https://www.nuget.org/packages/Google.Re2
