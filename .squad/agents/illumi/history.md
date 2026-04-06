# Illumi — History & Learnings

## Project Context

- **Project:** gert — YAML-driven runbook orchestration engine
- **Owner:** ormasoftchile
- **Stack:** Go (backend/engine), TypeScript (VS Code extension + React WebViews), C# (Azure integrations), YAML schemas
- **Description:** Runbook execution engine with a VS Code extension that provides a visual editor, runbook runner, and tool catalog — all rendered via WebView panels backed by React/TypeScript.
- **My role:** Port and extend the VS Code UX to a standalone web application so the team can use Playwright for automated end-to-end testing without human involvement.

## Learnings

<!-- Illumi appends learnings here during work sessions -->

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
