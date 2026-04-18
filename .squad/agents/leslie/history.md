# leslie — History

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

### Bibliography Infrastructure Added (2026-04-18)

Successfully implemented complete bibliography/references system for the gert v2 design document.

**Implementation details:**
- Created `references.bib` with 40+ BibTeX entries (academic papers, standards, workflow systems, runbook tools, policy frameworks, cloud-native tools, development tools)
- Updated `main.tex` with biblatex + biber backend (numeric citation style, sorted by name/year/title)
- Added 29 `\cite{}` commands across §00, §01, §03, §05, §08, §09
- Full compilation cycle: pdflatex → biber → pdflatex → pdflatex
- All citations resolved successfully, no undefined warnings
- Document expanded from 147 to 155 pages (8 pages for References chapter)

**Decision:** Chose biblatex + biber (modern LaTeX standard) over natbib + bibtex (older, less flexible). Biber provides better Unicode support, sorting flexibility, and active maintenance.

**System choice:** Documented in `.squad/decisions.md` entry "Decision: Bibliography System for Gert v2 Design Document"

### Document Structure Preparation (2026-04-18)

Created six new stub sections (§10–§15) based on Ken's gap analysis "New Section Proposals":
- §10 Migration and Compatibility — launch blocker for adoption; covers schema migration from v1 to v2
- §11 Governance and Policy — first-class treatment of gert's core differentiator (allowlists, denylists, env blocking, redaction, approval gates)
- §12 Evidence, Tracing, and Resumption — append-only trace format, state snapshots, SHA256 evidence, run resumption
- §13 Adapter Contracts — explicit interface for TUI, Web, VS Code adapters; event subscription API, handshake, render lifecycle
- §14 Input Provider Framework — v2 input provider model; `from:` resolution, JSON-RPC protocol, v1 `.provider.yaml` compatibility
- §15 Observability and Diagnostics — structured logging, metrics, OpenTelemetry integration, diagnostics for extension/tool interactions

Each stub contains 2-3 sentence description and \section{TODO} placeholder. Updated main.tex to \input all six sections after §09. Compilation verified (18 pages, 172KB PDF, no critical errors). All existing sections (§00–§09) audited and confirmed LaTeX-clean (no unclosed environments, no escaping issues).

## Cross-Agent Notes from Ken's Architectural Review (2026-04-18)

**Priority finding:** Design is a skeleton placeholder, not a buildable specification. Every section needs 3–5× expansion before implementation can begin.

**For Leslie:** Treat every section as "needs author pass." Critical gaps: §00 needs problem statement and v1 relationship, §01 needs measurable goals and v1 pain points, §02 needs component interfaces and data flow, §03 needs field inventory and migration path, §04 needs protocol definition, §06 needs event schema, §07 needs threat model and governance specification.

**Entirely missing:** Migration strategy, governance layer design, evidence capture spec, adapter contracts, input provider design, runbook lifecycle, observability model, deployment packaging, concurrency model, and gert serve / RPC contract.

**Recommended:** Leslie should prioritize writing the 7 items listed as "Priority Recommendations" in Ken's gap analysis at `.squad/tmp/ken-gap-analysis.md`. Decisions documented at `.squad/decisions.md` (search "Ken's Gap Analysis Findings").

### Bibliography Infrastructure Added (2026-04-18)

Successfully implemented complete bibliography/references system for the gert v2 design document.

**What was done:**
1. Created references.bib with 40+ BibTeX entries
2. Updated main.tex with biblatex + biber backend
3. Added cite commands throughout sections
4. Successfully compiled document

**Statistics:**
- Total BibTeX entries: 40+
- Actually cited: 26
- Final page count: 155 (up from 147)

**Entry types:** Academic papers, standards (OTEL, ISO27001, SOC2, NIST), workflow systems (Temporal, Argo, Prefect), policy frameworks (OPA, Cedar), cloud-native tools (Kubernetes, Gatekeeper), development tools.

**Compilation:** Using biblatex + biber, numeric style, all citations resolved successfully.
