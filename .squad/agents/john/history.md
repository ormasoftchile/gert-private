# john — History

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

**For John (Schema):** §03 Schema vNext has critical gaps. Needs concrete starting point with:

1. **Field inventory** — What are the core language fields in v2? How do they differ from v1? (retained, removed, breaking-changed, new)
2. **Namespace convention** — For extension fields, what is the exact convention? (just "namespaced" is not enough)
3. **Versioning strategy** — What is `apiVersion: runbook/v2`? What breaks from v1?
4. **Migration path** — Must specify how `gert migrate` handles `runbook/v0` and `runbook/v1`
5. **Validation split** — Define what "structural" vs. "semantic" validation means concretely
6. **Step types** — Do cli, tool, manual, invoke, branch, iterate survive unchanged? Any new types? Removed types?
7. **Expression language** — Go templates currently — staying in v2?
8. **Tool schema vNext** — Does tool definition format change in v2?
9. **Provider schema** — How are providers defined in v2?
10. **Extension registration** — How are namespaced schema extensions registered and validated?

Ken's recommendation: Treat §03 as "needs expansion 3–5×". This is a blocker for Brian's parser implementation and validation engine. See full gap analysis at `.squad/tmp/ken-gap-analysis.md` (§03 section, lines 70–90). Decisions documented at `.squad/decisions.md` (search "Ken's Gap Analysis Findings").
