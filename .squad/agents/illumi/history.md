# Illumi — History & Learnings

## Project Context

- **Project:** gert — YAML-driven runbook orchestration engine
- **Owner:** ormasoftchile
- **Stack:** Go (backend/engine), TypeScript (VS Code extension + React WebViews), C# (Azure integrations), YAML schemas
- **Description:** Runbook execution engine with a VS Code extension that provides a visual editor, runbook runner, and tool catalog — all rendered via WebView panels backed by React/TypeScript.
- **My role:** Port and extend the VS Code UX to a standalone web application so the team can use Playwright for automated end-to-end testing without human involvement.

## Learnings

<!-- Illumi appends learnings here during work sessions -->

### 2025-01-23: Runbook Runner Web Implementation (Phase 2)

**Task:** Port RunbookPanel from VS Code extension to `web/src/views/runbookRunner.ts` with live execution, output streaming, and manual choice prompts.

**What I Learned:**

1. **State machines are 100% portable:** The `snapshotStateMachine.ts` module (287 lines of pure reducer logic) copied to `web/src/shared/` with ZERO modifications and worked immediately. No VS Code imports, no side effects, no framework coupling. This is the gold standard for portable code — pure functions that transform state based on events. Every module should aspire to this level of purity.

2. **WebSocket API is simpler than stdio:** Browser WebSocket is cleaner than Node's `child_process.spawn()` + readline streams. No process lifecycle management, no stderr vs stdout pipe juggling, no `readyState` polling. Just `onopen`, `onmessage`, `onerror`, `onclose` callbacks. The server-sent event model (one-way push) maps perfectly to reactive UI updates.

3. **Test IDs as first-class contracts:** Added `data-testid` attributes to every interactive element and output component. This creates a stable contract between frontend and QA automation that survives CSS refactoring. Test IDs should be treated as part of the component API, not an afterthought. Playwright tests will be trivial to write.

4. **Simplified graph beats complex DAG for MVP:** The VS Code RunbookPanel uses an 889-line `graphRenderer.ts` with SVG layout, themes, minimap, zoom/pan, and annotation badges. I implemented a vertical step list in 50 lines that provides 90% of the value (step state visibility) with 5% of the complexity. The hard part of runbook execution is state management, not visualization. Premature graph sophistication is technical debt.

5. **Event-driven rendering scales well:** The `handleEvent()` → `applyEvent()` → `render()` cycle is declarative and predictable. Every WebSocket event triggers a state transition, which triggers a full UI rebuild. This brute-force approach works because the DOM is small (< 100 steps) and Vite's dev server hot-reloads instantly. No need for React-style virtual DOM diffing.

6. **CSS variables enable instant theming:** All styles use `--vscode-*` custom properties from the parent HTML. This means light mode will "just work" when we add a theme toggle — just swap the `:root` variable values. The web app and VS Code extension share the same visual language despite zero shared CSS files.

7. **Auto-scroll requires explicit DOM manipulation:** Appending output lines to a scrollable div doesn't auto-scroll to bottom. Must set `scrollTop = scrollHeight` after each DOM update. Tried `scroll-behavior: smooth` but it's janky when lines arrive fast. Instant jump is better UX for log streaming.

8. **Modal overlays need backdrop click prevention:** The choice prompt modal uses `position: fixed` with `rgba(0,0,0,0.7)` backdrop. Initially, clicking the backdrop closed the modal (bad — user might click accidentally mid-decision). Removed backdrop click handler so only choice buttons dismiss the modal. Manual steps should be explicit, not accidental.

9. **Disabled state needs visual feedback:** When a run is active, the "Run" button and path input are disabled (`disabled` attribute). Added `opacity: 0.5` and `cursor: not-allowed` in CSS. Without visual feedback, users spam-click the button thinking it's broken. Disabled UI elements must look disabled.

10. **Error states deserve first-class treatment:** Created dedicated error banner (red border, error icon, help text) instead of just `alert()`. When server is offline, show actionable help: "Make sure the gert server is running: `gert serve --http --port 7777`". Error messages should teach, not just complain.

11. **Minimal shared modules reduce coupling:** Copied only `snapshotStateMachine.ts`, `helpers.ts` (escaping/icons), and `treeOps.ts` (type definitions). Skipped `graphRenderer.ts` (889 lines + theme system), `treeToGraph.ts` (708 lines + layout engine), and `stepNodeRenderer.ts` (SVG templating). Total shared code: ~400 lines. The less you share, the less you break when refactoring.

12. **TypeScript strict mode catches real bugs:** Unused variable warnings (`i` in `.map((entry, i) =>`) exposed dead code. Changed to `.map((entry) =>)` to satisfy the linter. These warnings aren't noise — they're code smell detectors. If a variable is declared but never read, it's either a bug or dead code.

13. **Build time is a developer happiness metric:** Vite builds the entire web app in 42ms. TypeScript compilation + bundling + minification + gzip = 42ms. This enables instant feedback loops during development. If builds took 5+ seconds, I'd be context-switching to Slack between changes. Fast builds = flow state preservation.

14. **Bundle size matters for internal tools:** 71.86 KB total (22.93 KB gzipped) for the entire web app. ToolCatalog + RunbookRunner + client library + state machine + helpers. No bloat. Internal tools often ignore bundle size because "it's just us," but slow load times erode trust. If it feels sluggish, people stop using it.

15. **WebSocket reconnection is a Phase 3 problem:** Current implementation fails hard on disconnect (error banner, execution stops). Reconnection logic would require: (a) persist `runId` across disconnects, (b) resume event stream from last acknowledged event, (c) merge partial state, (d) handle idempotency. Too complex for MVP. Better to fail visibly than silently lose state.

**Key Decisions Made:**

- **No full DAG rendering:** Vertical step list instead of graph. Can add later if users request it.
- **No syntax highlighting:** Plain text for SQL/KQL queries. Can add highlight.js later (~30 KB).
- **No source mapping:** Can't jump to YAML source line (VS Code extension feature, needs file I/O).
- **No annotation support:** Step notes/tags not implemented (future enhancement).
- **Single concurrent run:** UI blocks "Run" button during execution. No runId collision handling needed.

**Portability Insights:**

- RunbookPanel was ~2,000 lines across 14 files. Web version is 440 lines in 1 file (78% reduction).
- Main complexity: State machine (100% reused), WebSocket events (90% reused), HTML templates (100% new).
- Biggest blocker: `graphRenderer.ts` dependencies (VS Code themes, annotation system, file I/O for prose).
- Workaround: Skip graph, render list. Users can still see step states and output, which is 90% of value.

**Files Created/Modified:**

```
web/src/
  views/
    runbookRunner.ts          — Main view class (440 lines, new)
  shared/
    snapshotStateMachine.ts   — State reducer (copied, unchanged)
    helpers.ts                — Escaping/icons (copied, highlight.js removed)
    treeOps.ts                — Type definitions (minimal extraction)
  main.ts                     — Wired RunbookRunner into tab navigation (modified)
web/
  index.html                  — Added RunbookRunner CSS (~400 lines, modified)
.squad/decisions/inbox/
  illumi-runbook-runner.md    — Decision document (new)
```

**Build Validation:**

✅ `npm run build` — Zero TypeScript errors  
✅ Bundle size: 71.86 KB (22.93 KB gzipped)  
✅ Build time: 42ms  
✅ Test IDs: 14 test attributes for Playwright automation  

**Next Steps:**

1. **Integration test:** Wait for Killua to deploy HTTP server with WebSocket support
2. **Manual validation:** Test against real runbook execution (mitigation, RCA, etc.)
3. **Knov handoff:** Provide test ID list for Playwright E2E tests
4. **Phase 3 backlog:** DAG visualization, syntax highlighting, WebSocket reconnection

---

### 2025-01-23: Web Scaffold and ToolCatalog MVP Implementation

**Task:** Create `web/` directory scaffold with Vite + TypeScript and port ToolCatalogPanel to browser-native implementation.

**What I Learned:**

1. **Vite scaffolding approach:** Used manual `npm init` instead of `npm create vite` due to interactive prompts in non-interactive mode. Direct npm package installation and manual config files proved more reliable for automated workflows.

2. **TypeScript strict mode catches unused vars:** Had to remove unused imports (`ToolArgInfo`) and unused instance variables (`currentTab`) to pass `tsc` compilation. The strict linting rules enforce clean code — no dead weight.

3. **WebSocket connection lifecycle:** The browser WebSocket API is simpler than Node's `ws` library. No need for explicit `readyState` polling — just `onopen`, `onmessage`, `onerror`, `onclose` callbacks. Promise-based connection initialization makes async/await usage clean in the client constructor.

4. **Fetch vs stdio transport:** Replacing child process stdio with `fetch()` POST requests was straightforward — both are async promise-based. The key difference is error handling: HTTP failures throw on `response.ok === false`, while stdio process errors come through stderr. Web version needs explicit HTTP status checks.

5. **Clipboard API requires HTTPS or localhost:** `navigator.clipboard.writeText()` works on `localhost` but would fail on `http://` in production. Future deployment needs HTTPS or must fall back to `document.execCommand('copy')` for older browsers. Added error handling to show "❌ Copy failed" if clipboard write is blocked.

6. **CSS custom properties for VS Code theming:** Ported VS Code's CSS variable names (`--vscode-foreground`, `--vscode-panel-border`, etc.) directly to the web app. This gives visual parity with the extension AND allows future theme switching by just swapping the `:root` variable values. Dark theme is hardcoded for MVP; light theme could be added with a class toggle.

7. **HTML escaping is critical for XSS prevention:** Since we're generating HTML strings (not React), must manually escape all user data. Created `escapeHtml()` and `escapeAttr()` utility functions. Tool names, descriptions, and YAML snippets all flow through escaping before insertion into DOM. This prevents injection attacks if a malicious `.tool.yaml` file contains `<script>` tags.

8. **Test IDs as first-class citizens:** Added `data-testid` attributes proactively for Knov's Playwright tests. This is better than relying on class names or element structure, which might change during UI refactoring. Test IDs are a contract between frontend and QA.

9. **Graceful degradation pattern:** The app shows a helpful error message when the server is offline instead of a cryptic network error. This reduces support burden — users know exactly what to do (start `gert serve --http --port 7777`). Error states are part of the UX, not an afterthought.

10. **Vite proxy config is dev-only:** The proxy in `vite.config.ts` only works during `npm run dev`. For production builds, the static files in `dist/` must be served by a reverse proxy (nginx, caddy) or the gert server itself. This means Killua's HTTP server should eventually serve the static frontend files at `/` in addition to `/rpc` and `/ws`.

**Key Decisions Made:**

- **No React framework:** Stayed consistent with VS Code extension pattern (string-based HTML generation)
- **TypeScript strict mode:** Enforced for type safety and cleaner code
- **Vite over webpack:** Faster builds, simpler config, better DX
- **Manual HTML escaping:** Required due to string concatenation approach
- **Proxy-based development:** Vite dev server proxies `/rpc` and `/ws` to avoid CORS issues during local dev

**Portability Insights:**

- ToolCatalogPanel was indeed 95% portable as estimated
- Main changes: Removed `vscode.*` API calls, replaced webview HTML injection with direct DOM manipulation
- Shared types (`ToolInfo`, `ToolActionInfo`, etc.) copied verbatim from VS Code client
- YAML parsing with same `yaml` npm package — zero translation needed

**Files Created:**

```
web/
  index.html                — App shell with tab nav
  vite.config.ts            — Vite config with proxy
  tsconfig.json             — TypeScript config (strict mode)
  package.json              — Dependencies and scripts
  README.md                 — Updated with web app guide
  src/
    main.ts                 — Entry point, tab routing
    api/client.ts           — GertWebClient (HTTP + WebSocket)
    views/toolCatalog.ts    — ToolCatalog view (ported)
.squad/decisions/inbox/illumi-web-scaffold.md — Decision document
```

**Build Validation:**

✅ `npm install` — All deps installed (Vite, TypeScript, YAML)  
✅ `npm run build` — TypeScript + Vite build succeeded with zero errors  
✅ Output: `dist/` with static files (3.85 KB HTML, 58.46 KB JS bundle)  

**Next Steps:**

1. **Integration test:** Wait for Killua to implement HTTP server endpoints, then test live connection
2. **Error handling refinement:** Test what happens when server returns malformed JSON, empty tool list, network timeout
3. **Phase 2 prep:** Study RunbookPanel for porting (execution viewer with live state updates)
4. **Shared logic extraction:** Identify pure modules to copy from VS Code extension to `src/shared/`

---

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
