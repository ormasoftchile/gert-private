# Decisions Log

Merged from `.squad/decisions/inbox/` by Scribe.

---

## D-001: Editor UX Directives (User)

**Date:** 2026-03-18  
**By:** Cristián Ormazábal Ortega (via Copilot)  
**Status:** Directive — pending implementation  

### Layout Flexibility
The editor panel layout (form vs graph, left vs right) should be configurable via a VS Code setting. Switching between YAML source and visual workflow should be a toggle in the UI (not a command palette action). A play button to run the runbook should be available from both views.

### Tool Name Validation
Tool names in step definitions should not be free text. There needs to be a registration system (gert.yaml packages, tool packs, project-level catalog) to restrict tool selection to registered tools only.

### Query Syntax Highlighting
Steps containing query expressions (KQL, SQL, or similar) should have syntax highlighting in both the YAML source view and the visual editor form fields.

### n8n / Logic Apps Research
Research n8n and Azure Logic Apps Designer for features applicable to gert's visual editor. These are confirmed design references.

---

## D-002: Chain Visualization Model for Invoke Steps

**Date:** 2026-03-18  
**By:** Gon (Lead Architect)  
**Status:** Implemented  

Chain navigation is view-only state (`viewingChainIndex`) that selects which chain entry's tree/states to render in the graph — it never affects execution flow. The graph renderer accepts the selected chain entry's data and renders it identically to the current entry.

**Rationale:** Keeping chain navigation purely in the view layer avoids interfering with the execution engine or client-server protocol. Chain history already captures enough data (tree, stepStates, stepDetails) for full graph rendering.

---

## D-003: Deep-Merge YAML Serialization for Comment Preservation

**Date:** 2026-03-18  
**By:** Gon (Lead Architect)  
**Status:** Implemented  

Replace top-level `doc.set(key, doc.createNode(value))` with recursive `deepMergeNode()` that walks the YAML AST in parallel with the edited JS object, preserving comments and formatting for unchanged values.

**Tradeoffs:** Positional sequence merge is correct for step arrays (add/remove at end) but imperfect if steps are reordered mid-array. Falls back to full stringify with user-visible warning on failure.

---

## D-004: n8n and Logic Apps Feature Research

**Date:** 2026-03-18  
**By:** Gon (Lead Architect)  
**Status:** Research — informs roadmap  

Evaluated 17 features from n8n and Azure Logic Apps Designer. 7 recommended HIGH priority:
1. **Tool catalog panel** — searchable list from `tools/list`, click-to-insert (HIGH)
2. **Expression autocomplete** — template variable autocomplete in all text fields (HIGH)
3. **Per-step retry with backoff** — `retry: {max, interval, backoff}` in step schema (HIGH)
4. **Error branches** — `_error` condition type for post-retry failure routing (HIGH)
5. **Per-step I/O detail panel** — richer trace/input/output display (HIGH)
6. **Credential references in tool defs** — document required env vars (MEDIUM)
7. **Parallel iterate (concurrency)** — concurrent step execution in iterate blocks (MEDIUM)

Key insight: Gert's differentiators (YAML as truth, governance-first, tool/provider abstraction, VS Code-native) mean features need adaptation, not direct transplant.

---

## D-005: Drag-to-Reorder is Same-Parent Only

**Date:** 2026-03-18  
**By:** Kurapika (Frontend Engineer)  
**Status:** Implemented  

Step drag-to-reorder in the editor Flow Map is restricted to siblings within the same parent steps array. Cross-branch and cross-iterate dragging is blocked — only nodes with matching `data-parent-path` are valid drop targets.

**Rationale:** Moving steps between structural containers would require semantic validation. Same-parent reorder is safe, lossless, and covers the most common use case.

---

## D-006: HTTP Transport for Web Client Support

**Date:** 2026-04-05  
**By:** Killua (Backend Engineer)  
**Status:** Implemented  

### Context

The gert system currently uses JSON-RPC over stdio for VS Code extension communication. To support a standalone web application, HTTP + WebSocket transport exposes the same JSON-RPC API allowing browsers to connect.

### Decision

Added `gert serve --http --port N` to provide HTTP/WebSocket transport alongside existing stdio mode.

### Implementation Details

**Files:**
- New: `ext/serve/pkg/serve/serve_http.go` (265 lines)
- Modified: `cmd/gert/main.go`, `go.mod`

**Architecture:**
- `POST /rpc` — JSON-RPC requests (synchronous responses)
- `GET /ws` — WebSocket for real-time event streaming
- `GET /health` — Health check
- Events broadcast to all WebSocket clients
- CORS auto-allows localhost origins

**Key Design Choices:**
- No business logic duplication (reuses `Server.dispatch()`)
- Stdio mode unchanged (backward compatible)
- WebSocket for events (not HTTP responses)
- Standard library `net/http` + `gorilla/websocket`
- Graceful shutdown with SIGINT/SIGTERM

### Impact

- VS Code extension: No impact (stdio unchanged)
- Web client: Can connect via HTTP from localhost
- Dependencies: Added `gorilla/websocket`
- Build: Backward compatible

### Testing

✓ Build passes  
✓ Server starts, responds to `/health`  
✓ JSON-RPC test: POST /rpc works  
✓ Flags visible in `gert serve --help`  

---

## D-007: Web Frontend Scaffold — Phase 1 ToolCatalog MVP

**Date:** 2026-03-18  
**By:** Illumi (Web Frontend Engineer)  
**Status:** Implemented

### Context

Team needs web-based gert UI for automated E2E testing with Playwright. Web version swaps JSON-RPC transport from stdio to HTTP/WebSocket while maintaining same API contract.

### Decision

Created `web/` directory with Vite + TypeScript scaffold; ported ToolCatalogPanel as Phase 1 MVP.

### Tech Stack

- **Vite** — Fast build tool and dev server
- **TypeScript** — Type-safe development
- **Vanilla HTML** — String-based templates (no React)
- **Fetch API** — HTTP transport
- **WebSocket API** — Real-time events

### Architecture

```
web/
  index.html
  src/
    main.ts
    api/client.ts (GertWebClient)
    views/toolCatalog.ts (ToolCatalogPanel)
  vite.config.ts (proxy to :7777)
```

### API Transport

- **Extension:** stdio → `GertClient` spawns `gert serve`
- **Web:** HTTP → `GertWebClient` connects to `gert serve --http --port 7777`
- **Same API surface**, different transport layer

### ToolCatalog Features

- Searchable list of all available tools
- Expandable action cards (arguments, types)
- One-click YAML snippet copy
- Visual parity with VS Code extension
- Graceful error state when server offline
- All test IDs for Playwright automation

### Development Workflow

1. Start gert: `./gert serve --http --port 7777`
2. Start Vite: `cd web && npm run dev`
3. Open: http://localhost:5173
4. Vite proxies `/rpc` and `/ws` to localhost:7777

### Build Output

- `npm run build` outputs to `dist/` (58KB bundle)
- Zero TypeScript errors

### Why This Decision

1. Consistency with VS Code extension HTML generation pattern
2. Portable (minimal VS Code API surface)
3. Enables Playwright testing
4. Lightweight, no framework overhead
5. Incremental phase delivery

### Validation

✓ npm install succeeds  
✓ npm run build succeeds (0 errors)  
✓ Vite config proxies `/rpc` and `/ws`  
✓ Test IDs added for Playwright  

---

## D-008: Playwright E2E Infrastructure Setup

**Date:** 2026-04-05  
**By:** Knov (Playwright & E2E Testing Specialist)  
**Status:** Implemented

### Summary

Created complete Playwright test infrastructure at `web/tests/` enabling autonomous agent verification of UI changes through structured E2E testing.

### Infrastructure Components

#### Custom Test Fixture (`fixtures/base.ts`)

Starts gert server + Vite dev server once per test worker (parallel startup, ~5-8s).

- `beforeAll`: Starts both servers in parallel
  - gert: `gert serve --http --port 7778`
  - Vite: `npm run dev` (port 5173)
- `afterAll`: Cleanly kills both processes
- Exposed fixtures: `gertClient` (HTTP client), `testRunbook` (test data path)

#### Page Object Model

**ToolCatalogPage** (full implementation):
- `goto()` — Navigate to catalog
- `waitForLoad()` — Wait for tool list + loading indicator
- `getToolCount()` — Count tools
- `clickTool(name)` / `clickFirstTool()` — Interact
- `getDetailTitle()` — Read detail panel
- `search(query)` — Search
- `isServerErrorVisible()` / `getServerErrorText()` — Error states

**RunbookRunnerPage** & **RunbookEditorPage** (stubs for Phase 2)

#### Agent Reporter (`helpers/agent-reporter.ts`)

Structured JSON output (`test-results/agent-report.json`) for autonomous agent verification:

```json
{
  "verdict": "PASS" | "FAIL",
  "timestamp": "ISO-8601",
  "summary": "3/3 passed",
  "tests": [
    {
      "name": "test name",
      "status": "passed" | "failed" | "skipped",
      "duration": 1234,
      "error": null | "error message"
    }
  ],
  "stats": {
    "total": 3,
    "passed": 3,
    "failed": 0,
    "skipped": 0
  }
}
```

#### Test Specs (`tool-catalog.spec.ts`)

4 tests:
1. ✅ Loads tool catalog and shows tool list
2. ✅ Clicking a tool shows its detail
3. ⏭️ Shows friendly error when server not running (Phase 2)
4. ⏭️ Search filters tool list (Phase 2)

#### CI Integration (`.github/workflows/e2e.yml`)

Runs on push/PR to main. Steps: checkout, Go setup, Node setup, build gert, install dependencies, run tests, upload artifacts (7-day retention).

### Autonomous Dev Loop

1. Make UI change
2. Run: `cd web && npm run test:e2e`
3. Check: `test-results/agent-report.json`
   - `verdict === "PASS"` → proceed
   - `verdict === "FAIL"` → read errors, view screenshots
4. Debug if needed: `npm run test:e2e:headed`
5. Commit when green

### Playwright Configuration

- Test dir: `./tests/specs`
- Base URL: `http://localhost:5173` (Vite)
- Timeouts: 30s/test, 10s/assertion
- Reporters: HTML, JSON, agent reporter, list
- Screenshots/traces on failure
- Chrome only (Phase 1)

### Phase 1 Scope

**Ready now:**
✓ Complete test infrastructure  
✓ Custom fixture with server management  
✓ Page Object Model architecture  
✓ Agent reporter  
✓ CI workflow  
✓ 2 passing tests for tool catalog  
✓ Sample test runbooks

**Deferred to Phase 2:**
- Runbook runner tests
- Graph view tests
- Editor tests
- Server lifecycle control
- Multi-browser testing

### Dependencies

**NPM packages added:**
- `@playwright/test: ^1.42.0` (devDependency)

**Existing dependencies leveraged:**
- `vite` (dev server)
- `typescript` (type checking)

### Validation

✓ Playwright 1.59.1 installed  
✓ Chromium browser installed  
✓ Ready for web app implementation  

---

**Merged from `.squad/decisions/inbox/` on 2026-04-06T03:05:29Z**
