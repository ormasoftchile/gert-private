# barbara — History

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


## Cross-Agent Notes from Ken's Architectural Review (2026-04-18)

**For Barbara (Integrations):** §05 Tool Runtime section has important gaps. Needs detailed specifications for:

1. **Tool discovery algorithm** — How are tool definitions found in v2? Still by convention `tools/<name>.tool.yaml`? Or registry-based?
2. **Transport layer specification** — stdio/jsonrpc/mcp exist in v1. Do all three survive in v2? Any new transports?
3. **Invocation envelope** — What fields are in the "run context and telemetry" mentioned? Full spec needed.
4. **Error envelope schema** — "Structured and machine-readable" is not a spec. Define the exact error shape.
5. **Capability gates** — What checks happen during tool invocation? Who enforces them?
6. **Tool versioning** — Can a runbook pin a tool version? How?
7. **Timeout/cancellation** — What is the contract for timeouts? How does cancellation propagate to tools?
8. **Output capture** — How does stdout/stderr map to captured variables in v2?
9. **Built-in vs. extension-contributed tools** — What's the difference in §05's treatment?
10. **MCP tool integration** — MCP tool discovery is dynamic in v1. How does this interact with static governance policies in v2?

Ken's assessment: §05 is "Important" severity with 10 major gaps. This is a direct blocker for Barbara's tool/provider contract design and for Brian's tool runtime implementation. See full gap analysis at `.squad/tmp/ken-gap-analysis.md` (§05 section, lines 112–130). Decisions documented at `.squad/decisions.md` (search "Ken's Gap Analysis Findings").

## Cross-Agent Notes from Dennis's Research Brief (2026-04-18)

**For Barbara (Integrations):** Research identifies OPA integration and structured error envelopes as key integration findings.

- **OPA Integration**: OPA is CNCF standard, cloud-agnostic, and enables centralized governance policy across tools. Integration hooks: pre-execution, pre-step, post-step. OPA server mode for policy evaluation.
- **Structured Error Envelopes**: Industry tools (AWS, Temporal, Argo) use structured error formats with error codes, context, and metadata. Enables consistent error handling across tool/provider integrations.
- **OpenTelemetry Integration Points**: Emit spans per step (start, end, attributes, status). Trace context propagation across invoked runbooks and tools. Correlation IDs in all log entries.
- **Policy Examples**: "Commands including kubectl delete require 2 approvals", "Production runbooks require change-manager role"

Full research brief: `.squad/tmp/dennis-research-brief.md`
Decision inbox: `.squad/decisions/inbox/dennis-research-foundations.md` → merged to `.squad/decisions.md`
