# Illumi — History & Learnings

## Project Context

- **Project:** gert — YAML-driven runbook orchestration engine
- **Owner:** ormasoftchile
- **Stack:** Go (backend/engine), TypeScript (VS Code extension + React WebViews), C# (Azure integrations), YAML schemas
- **Description:** Runbook execution engine with a VS Code extension that provides a visual editor, runbook runner, and tool catalog — all rendered via WebView panels backed by React/TypeScript.
- **My role:** Port and extend the VS Code UX to a standalone web application so the team can use Playwright for automated end-to-end testing without human involvement.

## Learnings

<!-- Illumi appends learnings here during work sessions -->

### 2025-01-23: Initial Web Portability Analysis

**Task:** Analyze VS Code extension frontend for web portability and recommend tech stack.

**What I Learned:**

1. **The codebase uses NO React** — All WebViews generate HTML via string concatenation (`getHtml()` methods). The project description says "React WebViews," but the actual implementation is pure TypeScript → HTML strings. This is a critical discovery that changes the porting strategy.

2. **State management is exemplary** — The `snapshotStateMachine.ts` module is a pure reducer with zero side effects and no framework dependencies. It's 100% portable and represents excellent functional architecture.

3. **API contract is already JSON-RPC** — All communication between extension and Go server uses a well-defined JSON-RPC protocol. This means the web version is a transport swap (stdio → HTTP/WebSocket), not an API redesign.

4. **File I/O is the biggest challenge** — The RunbookEditorPanel does direct filesystem reads/writes. In the web version, this requires server-mediated file operations with conflict detection (optimistic locking or OT).

5. **Graph rendering is SVG-based** — The workflow visualization uses SVG generated server-side with D3-like layout algorithms. This is browser-native and highly portable. No canvas or framework-specific charting library.

6. **VS Code API surface is localized** — Only ~100 VS Code API calls across all views, and they fall into 3 categories: clipboard (trivial), file I/O (needs server API), and user dialogs (needs HTML modals). No deep coupling.

**Key Decisions Made:**

- **Tech stack:** Vite + TypeScript + string-based HTML (port existing pattern, not React)
- **Source organization:** Fork to `web/` directory, share pure logic (`snapshotStateMachine.ts`, `treeToGraph.ts`, `graphRenderer.ts`)
- **API transport:** Extend Go server with HTTP + WebSocket (maintain stdio mode for VS Code extension)

**Portability Scores:**
- ToolCatalogPanel: 95% portable (1-2 days)
- RunbookPanel: 70% portable (4-6 days)
- RunbookEditorPanel: 40% portable (8-12 days)

**Effort to MVP:** 6-8 weeks (1 engineer, all 3 views with server extensions).

**Files Created:**
- `.squad/decisions/inbox/illumi-web-portability-analysis.md` — Full 20KB analysis with recommendations, file matrices, API inventory, and risk assessment.

**Next Steps:**
- Wait for team decision on React vs. string-based HTML rendering
- Prototype HTTP server extension (chi router + WebSocket)
- Define HTTP API contract (OpenAPI spec)
- Scaffold `web/` directory with Vite
