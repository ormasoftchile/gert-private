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

## Project Context

GERT is a governed, executable, traceable runbook engine (Go binary, local-first). It has a VSCode extension, TUI runner, and mobile SDKs. The team focus is building a web execution platform on Azure — white-labeled customer portals where end-users run runbooks as part of business processes (applications, authorizations, compliance flows). Azure infra only (no k8s). Key concerns: queue-driven execution, webhook delivery, tenant isolation, audit traceability.

**User:** ormasoftchile
**Session start:** 2026-06-03
