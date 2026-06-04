# Don — Project History

## Learnings

### 2026-06-03 — C# Governance Parity Design

**Deliverable produced:** `design/web-platform/c-sharp-governance-parity.md`

**Key design decisions made:**

- **Pipeline mapping:** Full Go-to-C# interface mapping documented. `IRunHandle.NextAsync()` is the governance boundary — identical to Go's `RunHandle.Next()`. Every stage (Parser → Planner → Runtime → RunHandle) has a named C# equivalent with method signatures.
- **IApprovalGate stubbed for A6/A8 swap:** Designed so `ServiceBusApprovalGate` (A6: held thread, Service Bus reply queue) can be swapped for `DurableApprovalGate` (A8: `WaitForExternalEvent`) without changing `IRunHandle` or the governance layer. The runbook contract sees only `IApprovalGate`.
- **IStepRunner enforcement via `internal sealed`:** `IStepRunner` and all concrete runners (`CliStepRunner`, `ToolStepRunner`, etc.) are `internal` to `Gert.Runtime.Core`. No `InternalsVisibleTo` grants to adapter assemblies. Bypass is a compile error, not a code review finding. Two Roslyn analyzers (`GERT0001`, `GERT0002`) enforce this and the RE2 namespace restriction at build time.
- **RE2 via `Google.Re2` NuGet:** All redaction evaluation uses `Google.Re2.Regex`, never `System.Text.RegularExpressions`. Patterns containing lookahead/lookbehind/possessive quantifiers will throw at startup (fail-fast). Named group syntax difference (`(?P<name>...)`) flagged as open question OQ-5.
- **Template engine gap:** Go `text/template` has no C# equivalent. Option A (minimal port) is recommended for MVP. Option B (WASM compiled Go evaluator) is the fallback. 10 canonical test vectors defined (TV-TMPL-001..010).
- **Trace synchronous write:** `IRunStore.WriteTraceAsync()` must complete before step dispatch. Call order in `RunHandleImpl.NextAsync()` is: governance check → emit event → **synchronous trace write** → step dispatch. Fire-and-forget is explicitly forbidden.
- **Validation gate:** 10 gates (G-01..G-10) defined. Go `gert replay` on C#-generated traces (G-06) is the strongest parity check. No production promotion with any gate failing. Roslyn analyzers enforce G-08 and G-09 at build time.
- **60 minimum test vectors** across 12 categories (allowlist, denylist, env blocking, redaction, approval gates, template evaluation).

**Open questions filed:** OQ-1 (template strategy), OQ-2 (approval reply queue topology), OQ-3 (canonical Go test harness), OQ-4 (sequence counter strategy), OQ-5 (RE2 named group syntax).

---

### 2026-06-03 — Execution Adapter Pattern Review

**Patterns reviewed:** Queue-triggered worker, HTTP-triggered Container App, Durable execution (Azure Durable Functions), Sidecar/agent model.

**Key governance concerns identified:**

- GERT's governance layer (allowlist/denylist, env var filtering, contract risk assessment, approval gates) fires entirely inside the **Runtime Core** during the `RunHandle.Next()` loop. It cannot be preserved by any pattern that bypasses `Runtime.Start → RunHandle.Next`.
- The JSONL trace file is written **synchronously inside the Runtime Core before each action proceeds**. Any pattern that writes trace events from outside the Go binary (e.g., a Function orchestrator writing its own log) would produce a trace that is NOT the authoritative GERT trace and would fail replay determinism.
- The `RunHandle` interface is **stateful and sequential** (`Next()` must not be called concurrently, re-entrant only per run). It cannot be split across multiple processes or Function invocations.
- Splitting execution into per-step Azure Durable Function activities is a **governance red line**: each activity invocation re-enters the binary cold, losing the in-process governance engine state and potentially allowing steps to execute without the full pre-flight chain.
- Approval gates (`RunHandle.Approve()`) and evidence submission (`RunHandle.SubmitEvidence()`) require a **live, running Go process** that holds the `RunHandle`. Any pattern that doesn't keep the worker alive through the pause window cannot support interactive runbooks without an explicit resume mechanism (checkpoint → reload from RunStore).
- The `client: "web"` field in `run/started` must be set correctly. Forgetting to identify the client means audit trail queries filtering by client will miss web-triggered runs.
- Queue-triggered worker is the strongest match: decoupled submission, worker owns the full Runtime Core, governance fires completely, trace goes to Blob Storage, approval gates handled via reply messages or a dedicated interaction queue.

**2026-06-04:** Brainstorm output merged to decisions.md. Governance red lines formalized and documented. Pattern evaluation matrix recorded. Requires team ratification before implementation. Orchestration log created.

---

### 2026-06-03 — C# Runtime Option Analysis

**Trigger:** User confirmed they are NOT attached to the Go binary for web execution. Open to a C# reimplementation of GERT governance for the Azure backend.

**What this unlocks:**
- Durable Functions Orchestrators can now legitimately implement `RunHandle.Next()` semantics — the orchestrator IS the governance engine, not a bypass of it. This was a red line when Go binary was fixed because each activity invocation would cold-start the binary, losing governance state. In C#, governance fires inside the orchestrator before each activity, which is correct.
- `ASP.NET Core BackgroundService` is the simplest C# equivalent of the queue-triggered worker — no new architectural complexity, same governance model, cleaner async/await ergonomics.
- `Durable Entities` can model `IRunHandle` state directly — entity operations are sequential by design, sequence numbers are trivially monotonic, and approval gates become entity signals.

**Red lines revised:**
- Per-step Durable Function Activities remain a RED LINE even in C# — if the activity calls a step runner directly (bypassing `IRunHandle.NextAsync()`), governance does not fire. Enforce by making `IStepRunner` internal, not injectable from outside `IRunHandle`.
- Durable Orchestrator replay is safe IFF: all governance checks with external side effects are wrapped in activities (cached on replay, not re-executed). Never put non-deterministic code inline in the orchestrator.

**Critical implementation risks identified:**
1. **Template evaluation drift (HIGH):** Go `text/template` has no C# equivalent. Any porting of template evaluation must be validated against a canonical test vector suite. Advanced template features may be impossible to port faithfully.
2. **Regex engine divergence in redaction (HIGH — governance security risk):** C# .NET regex is a superset of RE2. A redaction rule that matches in Go may behave differently in C#. **Resolution: use the `Google.RE2` NuGet package for all redaction evaluation — non-negotiable.**
3. **Governance logic drift (HIGH):** The Go governance layer is the reference implementation. C# port must be validated against a language-neutral test vector suite (JSON spec vectors) before production.
4. **JSONL trace contract drift (MEDIUM):** C#-generated traces must be validated against the JSON Schema artifact. Go replay engine must be run against C#-generated traces in CI.
5. **Durable replay non-determinism (MEDIUM):** Use `context.CurrentUtcDateTime`, `context.NewGuid()` — never `DateTime.UtcNow` or `Guid.NewGuid()` in orchestrator code.
6. **Feature parity gap over time (LOW but compounding):** Two runtimes = two maintenance surfaces. Spec-first governance feature development is required.

**Recommended C# runtime pattern hierarchy:**
1. BackgroundService + Container App (simplest, recommended MVP)
2. Container App Job (stronger per-run isolation)
3. Durable Orchestrator (approval-gate-heavy runbooks)
4. Durable Entity (interactive wizard-style runbooks only)

**Structural design:**
- `IRunHandle.NextAsync()` is the governance boundary — governance pre-flight fires before every step dispatch, same as Go's `Next()`.
- `IGovernanceLayer` mirrors the Go governance engine: allowlist, denylist, env var filtering, redaction, approval gates.
- `IRunStore.WriteTraceAsync()` is the authoritative JSONL writer — called synchronously before `NextAsync()` returns.
- Sequence numbers are scoped per run_id, starting at 0. In BackgroundService: simple `int` counter. In Durable: tracked in orchestrator state, incremented only via activity results (never inline).

**Decision filed:** `.squad/decisions/inbox/don-csharp-runtime-options.md` → merged to decisions.md (2026-06-03T20:36:12). Orchestration log created. **Action Item: Validate C# governance parity with Go binary before implementation.**

---

### 2026-06-03 — Choice Gate Runtime Contract

**Deliverables produced:** `design/web-platform/c-sharp-governance-parity.md`, `.squad/decisions/inbox/don-choice-gate-contract.md`

**Key learnings:**

- **Choice waits are first-class gates:** Choice steps are not `SubmitEvidenceAsync()` payload collection; they suspend inside `IRunHandle.NextAsync()` exactly like approval waits, but the actor is the portal subject rather than a third-party approver.
- **A6 durable source of truth:** For both choices and approvals, the portal/API callback must write the durable interaction row first. Worker polling, queue wake-up messages, or SignalR are accelerators only; they are never the authoritative contract.
- **A8 parity point:** `WaitForExternalEvent` is the correct Durable mapping for both `IChoiceGate` and `IApprovalGate`, provided the same pre/post JSONL trace contract is preserved around the wait.
- **Trace ordering matters for interactive steps too:** `choice_requested` / `approval_requested` must be written before blocking, and `choice_received` / `approval_received` after the response is durably observed.
- **Analyzer gap identified:** Existing Roslyn analyzers protect `IStepRunner` visibility and RE2 usage, but they do not yet prove gate calls can only originate from `IRunHandle.NextAsync()`. That needs a new analyzer or architecture test before interactive steps ship.
- **Timeout semantics split cleanly:** Choice waits should time out via cancellation and transition the run to `TimedOut`; approval waits may return `ApprovalDecision.TimedOut` while preserving the same `IRunHandle` contract.

---

## Project Context

GERT is a governed, executable, traceable runbook engine (Go binary, local-first). It has a VSCode extension, TUI runner, and mobile SDKs. The team focus is building a web execution platform on Azure — white-labeled customer portals where end-users run runbooks as part of business processes (applications, authorizations, compliance flows). Azure infra only (no k8s). Key concerns: queue-driven execution, webhook delivery, tenant isolation, audit traceability.

**User:** ormasoftchile
**Session start:** 2026-06-03


---

### 2026-06-03 — User Input Gate Correction

**Deliverables produced:** `design/web-platform/c-sharp-governance-parity.md`, `.squad/decisions/inbox/don-user-input-gate-correction.md`

**Key corrections applied:**

- **Generalized the runtime primitive:** Replaced `IChoiceGate` with `IUserInputGate` so any blocking portal interaction (choice, text, confirmation, file upload, form) uses the same suspend/await/resume contract inside `IRunHandle.NextAsync()`.
- **Trace contract broadened:** Interactive JSONL events are now `user_input_requested` / `user_input_received` and must carry `kind` so auditors can reconstruct the exact prompt category.
- **A6/A8 resume parity clarified:** App Service workers and Durable orchestrations both resume on any valid `UserInputResponse`; SignalR, queue wake-ups, and in-memory wait handles remain accelerators only.
- **Analyzer gap sharpened:** Follow-up enforcement now targets any resume path that bypasses `IUserInputGate`, not only choice-specific flows.
- **Open question updated:** Replaced the old choice-specific enforcement question with a broader file-upload sizing/blob-storage question for `UserInputKind.FileUpload`.

---

## Session: 2026-06-04T02:50:37Z — Interactive Waiting Patterns & User Input Gate

**Scribe consolidated 8 inbox items.**

**Key outcomes for Don:**
- **IUserInputGate contract:** `AwaitUserInputAsync(runId, stepId, UserInputRequest, CT) → UserInputResponse`
- **Gate execution:** Both `IUserInputGate` and `IApprovalGate` fire inside `IRunHandle.NextAsync()` only
- **C# governance parity:** 10 validation gates (G-01 through G-10) adopted
- **Analyzers:** `GERT0001` (block IStepRunner refs), `GERT0002` (block System.Text.RegularExpressions)
- **RE2 requirement:** Google.Re2 NuGet mandatory; confirm `(?P<name>...)` syntax support
- **OQ-5:** Sequence counter behavior (in-memory atomic + checkpoint vs RunStore read)
- **Status:** Decisions ratified and merged to decisions.md; implementation can proceed with OQ resolutions

