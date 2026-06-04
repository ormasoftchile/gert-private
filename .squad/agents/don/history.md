# Don — Project History

## Learnings

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

## Project Context

GERT is a governed, executable, traceable runbook engine (Go binary, local-first). It has a VSCode extension, TUI runner, and mobile SDKs. The team focus is building a web execution platform on Azure — white-labeled customer portals where end-users run runbooks as part of business processes (applications, authorizations, compliance flows). Azure infra only (no k8s). Key concerns: queue-driven execution, webhook delivery, tenant isolation, audit traceability.

**User:** ormasoftchile
**Session start:** 2026-06-03
