# brian — History

## Project Context

**Project:** gert — Governed Executable Runbook Engine
**Owner:** ormasoftchile
**Mission:** Redesign gert from scratch as v2, applying learnings from v1.
**Stack:** Go, YAML, JSON Schema (Draft 2020-12), LaTeX (design docs), TypeScript (VS Code extension)

## Current Focus

The team is working on the **v2 design document** located at `design/gert-v2/`.
This is a LaTeX document using the MastersThesis class. Sections are in `design/gert-v2/sections/`.

### v2 Goals
1. Extensive research: runbooks, workflows, governance, traceability
2. Apply research + project learnings to define the new version
3. Document the new design as a usable reference to build v2

### v1 Key Features (for reference)
- Validate/execute/debug runbooks in YAML
- Governance: approval gates, allowlists, output redaction
- Evidence capture with SHA256, append-only JSONL traces
- VS Code extension + TUI + JSON-RPC server
- Tool definitions (.tool.yaml) and input providers (.provider.yaml)
- Step types: cli, manual, tool, invoke, branch, iterate

## Key Files
- `design/gert-v2/main.tex` — root LaTeX document
- `design/gert-v2/sections/` — all section .tex files
- `design/gert-v2/MastersThesis.cls` — document class
- `ext/` — Go source (core engine)
- `vscode/` — VS Code extension (TypeScript)

## Learnings

## Cross-Agent Notes from Dennis's Research Brief (2026-04-18)

**For Brian (Go Runtime):** Research prioritizes saga/compensation and context propagation as key implementation priorities.

- **Saga/Compensation Pattern**: Allow steps to register compensating actions; execute in reverse on failure. Industry standard in Temporal, Argo, Prefect. MVP priority (high impact, moderate effort).
- **Context Propagation**: OpenTelemetry trace context must propagate across invoked runbooks and tools. Baggage for runbook-specific metadata. Correlation ID in all log entries.
- **Timeout/Escalation State Machine**: Human step SLA enforcement with timeout and escalation paths. On timeout: escalate to alternate approvers, send notifications, or fail. MVP priority (high impact, low-moderate effort).
- **Idempotency Guarantees**: Step design requirements must be enforced and documented. Enables retry with backoff without side effects.

Full research brief: `.squad/tmp/dennis-research-brief.md`
Decision inbox: `.squad/decisions/inbox/dennis-research-foundations.md` → merged to `.squad/decisions.md`

## Learnings — 2026-04-18

### §12 Written From Scratch

Wrote the complete `12-evidence-tracing-resumption.tex` section. Key decisions embedded:
- JSONL trace lives at `.runbook/runs/<run-id>/trace.jsonl`; 14 normative event types defined
- `DurableEvent` envelope with `seq`, `type`, `timestamp`, `run_id`, `path`, `data` fields
- Crash safety via `O_APPEND` + single `write(2)` + `fsync` per event
- Optional HMAC-SHA256 tamper evidence on each event (`GERT_TRACE_KEY` env)
- Evidence: text, checklist, attachment (SHA256 content-addressed in `attachments/`)
- Snapshot format already exists in v1 (`RunState` JSON) — preserved in v2 with `version` field added
- Partial step resumption: idempotent tools re-issued; non-idempotent tools treated as failed
- Replay mode uses scenario YAML; v1 scenario format is forward-compatible
- OTel is opt-in (no-op tracer when `OTEL_EXPORTER_OTLP_ENDPOINT` unset); zero overhead default

### §08 Expanded

Expanded `08-testing-and-acceptance.tex` from 4 bullets to a full testing strategy covering:
- Test framework: Go `testing` + testify; table-driven tests as canonical pattern
- `gert test`: scenario discovery convention, test.yaml assertions, v1 scenario compatibility
- Contract tests for each interface boundary (runtime, schema, extension, tool, event bus)
- Integration test strategy with fixture corpus covering all step types
- Regression: v1 corpus at 95% pass rate target; golden trace comparison
- Performance targets: step startup <10ms p50, trace write <5ms p50
- Extension test harness via `pkg/exttest` package
- CI/CD: PR gate (unit+schema+contract+lint+vet), merge gate (integration+regression+golden), nightly (bench+race)

### Go Implementability Brief

Wrote `/Volumes/Projects/gert/.squad/tmp/brian-implementability-notes.md` covering:
- Top 5 concerns: event bus blocking, context propagation, saga compensation, parallel iterate safety, serve JSON-RPC compatibility
- Recommended package structure rooted at `pkg/` with clean `ext/` boundary
- Key interfaces: Runtime, Planner, EventBus, TraceWriter, RunStore, CommandExecutor, EvidenceCollector, GovernanceEngine
- Channels over mutexes for event bus; sequential engine loop (no goroutine-per-step)
- v1 patterns to preserve and anti-patterns to eliminate documented

### Cross-Agent Flags

- **For Ken**: §02 needs interface definitions matching the interfaces in the implementability brief.
- **For John**: `DurableEvent.data` is `json.RawMessage` — needs discriminated union JSON Schema per event type.
- **For Leslie**: §08 uses `\begin{tabular}` and `[label=\arabic*.]` — confirm `enumitem` is in the preamble.
