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

### 2026-04-05: Phase 1 Playwright Infrastructure Implementation

**Context:** Built the complete Playwright E2E test infrastructure for the gert web application. This is Phase 1 — the foundation that enables autonomous agent verification of UI changes.

**What Was Built:**

1. **Test Directory Structure (`web/tests/`):**
   - `playwright.config.ts` — Main config (Chrome only, agent reporter, screenshots/traces on failure)
   - `fixtures/base.ts` — Custom fixture that starts gert server (port 7778) + Vite dev server (port 5173)
   - `fixtures/runbooks/` — Sample runbooks: `simple.runbook.yaml` (linear) and `branching.runbook.yaml` (conditional paths)
   - `pages/` — Page Objects: `ToolCatalogPage` (full), `RunbookRunnerPage` (stub), `RunbookEditorPage` (stub)
   - `helpers/agent-reporter.ts` — Custom Playwright reporter → `test-results/agent-report.json`
   - `specs/tool-catalog.spec.ts` — 4 tests (2 active, 2 skipped for Phase 2)
   - `specs/runbook-runner.spec.ts` — 5 placeholder tests (all skipped, Phase 2)

2. **Custom Fixture Architecture:**
   - **beforeAll:** Starts gert + Vite servers in parallel (target <8s startup)
   - **afterAll:** Gracefully kills both processes (SIGTERM → SIGKILL fallback)
   - **Exposed fixtures:**
     - `gertClient`: HTTP client for direct API calls to gert server
     - `testRunbook`: Path to `simple.runbook.yaml` test data
   - **Reliability:** Smart server polling, detailed error messages, process output capture

3. **Agent Reporter (Key Innovation):**
   - Outputs structured JSON to `test-results/agent-report.json`
   - Format: `{ verdict: "PASS"|"FAIL", timestamp, summary, tests: [...], screenshots: {...}, stats: {...} }`
   - This is the **single source of truth** for agent self-verification
   - Agents read this file to know if their UI change is good (no human review needed)

4. **Page Object Model:**
   - **ToolCatalogPage:** Full implementation with methods for navigation, waiting, interaction, and error checking
   - Assumes `data-testid` attributes: `tool-catalog-container`, `tool-list`, `tool-item`, `tool-detail`, `search-input`, `loading-indicator`, `server-error`
   - **Runner/Editor Pages:** Stubs with outlined Phase 2 methods (state machine waits, output streaming, graph rendering)

5. **Test Specs:**
   - **tool-catalog.spec.ts:** 4 tests
     - ✅ "loads tool catalog and shows tool list" (active)
     - ✅ "clicking a tool shows its detail" (active)
     - ⏭️ "shows friendly error when server is not running" (skipped — Phase 2 needs server lifecycle control)
     - ⏭️ "search filters tool list" (skipped — Phase 1, search not yet implemented)
   - **runbook-runner.spec.ts:** 5 placeholder tests (all skipped, Phase 2)

6. **CI Integration (`.github/workflows/e2e.yml`):**
   - Triggers: push/PR to main
   - Steps: checkout → setup Go/Node → build gert → install deps → install Playwright → run tests
   - Artifacts: test results, **agent-report.json**, screenshots (on failure), traces (on failure)
   - Retention: 7 days

7. **Package.json Updates:**
   - Scripts: `test:e2e`, `test:e2e:ui`, `test:e2e:headed`
   - DevDependency: `@playwright/test: ^1.42.0`

**Key Learnings:**

1. **Custom Fixtures for Server Management:**
   - Starting servers once per worker (not per test) is critical for performance
   - Parallel server startup saves ~3-5s (vs sequential)
   - Smart polling with fallback timeouts handles flaky startup
   - Graceful shutdown (SIGTERM first) prevents zombie processes

2. **Agent-First Reporting:**
   - Structured JSON output is the key to autonomous dev loops
   - Agents need: verdict, test names, status, errors, screenshot paths
   - Human-readable reports (HTML) are secondary; machine-readable JSON is primary
   - Single source of truth eliminates parsing ambiguity

3. **data-testid Discipline:**
   - Stable selectors are non-negotiable for reliable E2E tests
   - `data-testid` attributes immune to CSS/DOM structure changes
   - Coordinated with Illumi during implementation (shared contract)
   - Additional semantic attributes (`data-state`, `data-output-complete`) needed for async states (Phase 2)

4. **Flakiness Mitigation from Day 1:**
   - Explicit `waitFor()` with state predicates (NEVER fixed timeouts)
   - Loading indicator handling (wait for hidden, fallback if fast load)
   - Server readiness polling before tests
   - Screenshots + traces on failure for debugging

5. **Phased Implementation Strategy:**
   - Phase 1: Infrastructure + tool catalog tests (web app being built in parallel)
   - Phase 2: Runner tests (execution states, streaming, graph rendering)
   - Phase 3: Editor tests (YAML round-trip, validation)
   - This unblocks parallel work (Illumi builds app, Knov readies tests)

6. **Test Runbook Design:**
   - Keep test runbooks minimal and deterministic
   - `simple.runbook.yaml`: 1 tool step (fast, no external dependencies)
   - `branching.runbook.yaml`: Manual choice + 2 branches (tests non-linear paths)
   - Using `echo` tool (assumed always available) for reliability

**Coordination Points:**

- **Illumi (Frontend Engineer):** Web app must have data-testid attributes on all interactive elements
- **Killua (Backend Engineer):** `gert serve --http --port 7778` must be implemented for tests to run
- **Phase 2 triggers:** RunbookPanel and RunbookEditorPanel implementations complete

**Verification:**

```bash
cd web && npm install && npx playwright install chromium
✅ Playwright 1.59.1 installed
✅ Chromium browser installed
✅ All infrastructure files created
✅ CI workflow ready
```

**Next Actions:**

1. Wait for Illumi's tool catalog implementation with data-testid attributes
2. Wait for Killua's `gert serve --http` implementation
3. Run first real tests once both dependencies land
4. Phase 2: Implement runner tests after RunbookPanel is built

**Autonomous Dev Loop Workflow (How Agents Use This):**

1. Agent makes UI change
2. Agent runs: `cd web && npm run test:e2e`
3. Agent reads: `test-results/agent-report.json`
4. If `verdict === "PASS"` → proceed to commit
5. If `verdict === "FAIL"` → read errors, view screenshots, fix, repeat

**No human review required for E2E validation.** This is the key to closing the dev loop.

**References:**
- Decision document: `.squad/decisions/inbox/knov-playwright-infra.md`
- Playwright config: `web/playwright.config.ts`
- Custom fixture: `web/tests/fixtures/base.ts`
- Agent reporter: `web/tests/helpers/agent-reporter.ts`
- Tool catalog tests: `web/tests/specs/tool-catalog.spec.ts`

<!-- Knov appends learnings here during work sessions -->
