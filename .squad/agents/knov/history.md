# Knov — History & Learnings

## Project Context

- **Project:** gert — YAML-driven runbook orchestration engine
- **Owner:** ormasoftchile
- **Stack:** Go (backend/engine), TypeScript (VS Code extension + React WebViews), C# (Azure integrations), YAML schemas
- **Description:** Runbook execution engine with a VS Code extension that provides a visual editor, runbook runner, and tool catalog — all rendered via WebView panels backed by React/TypeScript.
- **My role:** Build a Playwright E2E test suite for the gert web application, enabling agents to autonomously verify UI changes without requiring human testing in VS Code. This closes the dev loop and removes the human bottleneck.

## Learnings

### 2026-04-05: Initial Playwright Strategy Design

**Context:** Analyzed the gert codebase to design a comprehensive Playwright testing strategy for the proposed web application. The goal is to enable AI agents to autonomously verify their UI changes.

**Key Findings:**

1. **Current VS Code Architecture:**
   - Three main views: RunbookPanel (runner), RunbookEditorPanel (visual editor), ToolCatalogPanel
   - Complex state management: step execution states, branch routing, iterate loops, invoke chains
   - Real-time WebSocket communication with Go backend (`gert serve`)
   - SVG-based workflow graph rendering with pan/zoom (graphRenderer.ts)
   - Existing Jest tests for business logic (treeToGraph, snapshotStateMachine) but NO E2E tests

2. **Testing Challenges Identified:**
   - **Async execution states:** Steps transition pending → running → completed asynchronously via WebSocket events
   - **Streaming output:** Tool stdout/stderr arrives incrementally; tests must wait for completion signal
   - **Graph rendering:** SVG nodes/edges render after async layout calculation
   - **Complex workflows:** Branches, iterates, invoke chains create non-linear execution paths
   - **Manual steps:** User choice pickers require interaction simulation

3. **Architecture Decisions:**
   - **Persistent test servers:** Start `gert serve` + web dev server ONCE per test run (3-5s startup vs. per-test startup)
   - **Page Object Model:** Isolate UI structure from test logic (RunbookRunnerPage, RunbookEditorPage, ToolCatalogPage)
   - **data-testid discipline:** All selectors use stable `data-testid` attributes, immune to CSS changes
   - **Agent-first reporting:** Structured JSON output with verdict/assertions/screenshots for autonomous agent evaluation
   - **Custom waiters:** Explicit state polling via `page.waitForFunction()` for async transitions (NEVER fixed timeouts)

4. **Critical Test Scenarios (15 total):**
   - **Runner (10):** Linear execution, branching, iterates, invoke chains, streaming output, graph rendering, outcome selection, error handling
   - **Editor (3):** YAML parsing, form editing, round-trip fidelity
   - **Catalog (2):** Tool discovery, detail drill-down

5. **Flakiness Mitigation Strategy:**
   - **State transitions:** Wait for `data-state` attribute changes on step elements
   - **Output streaming:** Add `data-output-complete="true"` flag set by backend event handler
   - **Graph rendering:** Add `data-graph-ready="true"` flag set after all nodes/edges mounted
   - **Deterministic fixtures:** Use fixed runbook YAMLs from `examples/` directory

6. **CI Integration:**
   - New workflow: `.github/workflows/web-e2e.yml`
   - Runs in parallel with existing `vscode-ci.yml` (independent concerns)
   - Uploads agent-report.json artifact for automated consumption

**Deliverables:**
- Comprehensive strategy document: `.squad/decisions/inbox/knov-playwright-strategy.md`
- Test architecture, 15 critical scenarios, autonomous dev loop workflow, data-testid requirements, CI integration plan, flakiness mitigation playbook

**Next Steps:**
- Wait for web app foundation from Illumi (frontend engineer)
- Coordinate on data-testid placement during component implementation
- Implement Phase 1 (foundation + test fixtures) once web scaffold exists

**References:**
- Existing test harness: `vscode/test-harness/harness-run8.js` (CDP-based VS Code automation — inspiration for flakiness handling)
- Example runbooks: `examples/incident-triage.runbook.yaml`, `examples/simple-health-check.runbook.yaml`
- View implementations: `vscode/src/views/runbookPanel/`, `vscode/src/views/runbookEditorPanel.ts`, `vscode/src/views/toolCatalogPanel.ts`
- Graph rendering: `vscode/src/views/graphRenderer.ts`, `vscode/src/views/treeToGraph.ts`

<!-- Knov appends learnings here during work sessions -->
