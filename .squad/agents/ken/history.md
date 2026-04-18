# ken — History

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

### 2026-04-18 — Initial design review (sections 00–09)

**What the current design covers well:**
- Correct identification of component categories (core domain, extension kernel, tool runtime, API layer, adapters)
- Sound dependency direction principle (inward toward core)
- Right instinct on extension isolation (out-of-process, JSON-RPC over stdio, deny-by-default trust)
- Correct framing of adapters as thin renderers over events
- Identifies the right open questions (gRPC transport, extension signing, policy execution location)

**What's critically missing:**
- **Governance layer** — gert's core differentiator (allowlists, redaction, approval gates) is entirely absent from all nine sections
- **Migration and compatibility** — no v1→v2 story; all existing runbooks have unknown fate
- **Data flow / architecture detail** — §02 names five components but defines zero interfaces
- **Event schema** — §06 names five categories but defines zero events; meanwhile the decisions log already has implemented events (`event/branchResolved`, `event/iteratePassEnd`) that the design hasn't caught up to
- **Evidence, tracing, and resumption** — append-only JSONL, SHA256, run resumption are load-bearing v1 features with no v2 equivalent
- **Input provider framework** — `.provider.yaml` has no v2 equivalent
- **Extension capability taxonomy** — §04 and §07 both reference capabilities but neither defines what they are
- **Dangling spec reference** — `specs/002-extension-runtime-v0/spec.md` is cited in §04 but does not exist

**Key architectural concerns:**
- The document is currently a table-of-contents with intent statements, not a buildable spec
- Every section is 1–2× too thin; most need 5–10× more content before Brian can write Go
- The decisions log has outpaced the design document — multiple already-implemented decisions aren't reflected
- The three open questions in §09 are the *least* urgent open questions; the deeper ones (concurrency model, trace format, provider resolution, `gert serve` RPC contract) aren't listed

## Cross-Agent Notes from Dennis's Research Brief (2026-04-18)

**For Ken (Architect):** Research confirms governance, OTel, and saga/compensation as core priorities for v2.

- **Saga/Compensation Pattern**: Industry standard in Temporal, Argo, Prefect. Should be MVP priority for transaction reliability and rollback on failure.
- **OpenTelemetry Integration**: W3C Trace Context and OpenTelemetry are industry standards for distributed systems. Design hooks now for later implementation.
- **Governance Model**: Policy-as-code (OPA) is CNCF standard; more scalable than hardcoded allowlists. Recommend integrating evaluation hooks (pre-execution, pre-step, post-step) into architecture.
- **Human-in-Loop SLA**: Timeout/escalation are essential, not optional. ITIL and enterprise tools (ServiceNow, Jira, AWS Step Functions) all implement this.

Full research brief: `.squad/tmp/dennis-research-brief.md`
Decision inbox: `.squad/decisions/inbox/dennis-research-foundations.md` → merged to `.squad/decisions.md`
