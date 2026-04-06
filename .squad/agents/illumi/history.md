# Illumi — Activity History

## Learnings

### 2025-01-20: Three-Panel Layout Port from VS Code

**Task:** Port VS Code runbook panel's three-panel layout to web runner

**What I learned:**
- VS Code runbook panel uses a sophisticated three-panel layout: Prose (left), Workflow Map (center), Active Step (right)
- Prose panel renders step instructions as narrative content from runbook YAML
- Workflow map shows execution graph with visual state tracking (pending → running → passed/failed/skipped)
- Active step panel displays current step details, output, captures, and manual action controls
- Resizable splitters between panels persist user preferences across sessions
- VS Code uses vscode-* CSS variables; adapted to standard CSS custom properties for web
- VS Code uses `acquireVsCodeApi()` for messaging; replaced with GertWebClient HTTP+WS
- Step state tracking is critical for visual feedback — prose panel highlights active step, workflow map shows node states
- data-testid attributes on ALL interactive elements are essential for Playwright tests

**Key files read:**
- `/Volumes/Projects/gert/vscode/src/views/runbookPanel/html.css.ts` — THREE-PANEL CSS layout
- `/Volumes/Projects/gert/vscode/src/views/runbookPanel/html.ts` — HTML structure
- `/Volumes/Projects/gert/vscode/src/views/runbookPanel/prose.ts` — Prose rendering
- `/Volumes/Projects/gert/vscode/src/views/graphRenderer.ts` — Graph rendering (SVG)

**Implementation approach:**
1. Replaced shallow 2-panel layout (step list + output) with 3-panel (prose, map, active step)
2. Ported CSS from `html.css.ts` — prose/map/active-step-panel classes, splitter styles
3. Implemented resizable splitters with mouse drag handlers (persist widths in component state)
4. Prose panel: renders steps as narrative sections (h3 + instructions + query)
5. Workflow map: simplified vertical node list (VS Code uses full SVG DAG, we use simpler layout for MVP)
6. Active step panel: shows current step type, title, state, instructions, output, captures
7. Maintained all existing data-testid attributes for backward compatibility with Playwright tests
8. Added new testids: `prose-panel`, `workflow-map`, `active-step-panel`, `splitter-left`, `splitter-right`

**Challenges & Solutions:**
- **Challenge:** VS Code graph renderer has complex DAG layout logic with iterate/branch handling
- **Solution:** Simplified to vertical node list for MVP — maintains visual state tracking without full graph complexity
- **Challenge:** Splitter drag handling needs to prevent text selection during drag
- **Solution:** Set `document.body.style.userSelect = 'none'` during drag, restore on mouseup
- **Challenge:** Panel widths need to persist across re-renders
- **Solution:** Store proseWidth/mapWidth in component state, apply inline styles

**What works well:**
- Three-panel layout matches VS Code UX
- Resizable splitters feel native and responsive
- Prose panel highlights active step with visual indicator (left border + background)
- Workflow map shows step states with color coding (pending/running/passed/failed/skipped)
- Active step panel displays real-time output and captures
- Build passes with no TypeScript errors
- All existing data-testid attributes preserved

**Next steps for future work:**
- Enhance workflow map with full SVG DAG rendering (copy graphRenderer.ts logic)
- Add prose content from runbook's `prose` field (background, prerequisites, mitigation, escalation, references)
- Implement step navigation (click workflow node → jump to step)
- Add outcome banner for completion states (resolved/escalated/needs_rca)
- Support iterate/branch visualization in workflow map

**Status:** COMPLETE — Build passes, three-panel layout functional

---

### 2025-01-20: Web Frontend RPC Audit

**Task:** Comprehensive audit of web frontend vs VS Code extension for RPC/event mismatches

**Findings:**
- ✅ All RPC methods correctly match VS Code client (previous fixes were complete)
- ✅ All WebSocket event handlers align with server emissions
- ✅ Build passes clean with no errors
- ✅ TypeScript compiles with 0 errors
- ✅ Full data-testid coverage for all interactive elements in active views
- ✅ No code changes needed — frontend is production-ready

**Key Observations:**
1. **RPC method naming convention:** All execution-related methods use `exec/*` prefix (exec/start, exec/next, exec/submitChoice, etc.)
2. **Event naming convention:** Step lifecycle events use `step/*` prefix, custom events use `event/*` prefix, run-level events use `run/*` prefix
3. **Defensive event handling:** Frontend handles some events (run/started, step/output, run/error) that aren't yet emitted by server — this is safe and future-proof
4. **Graceful degradation:** Editor view checks for workspace/* API support and shows "unsupported" state rather than failing — excellent pattern
5. **Test coverage:** Every interactive element has `data-testid` for E2E testing — follows charter requirement

**What worked well:**
- Parallel file reading with view tool
- Using grep to search server event emissions across large codebase
- Build + TypeScript validation in parallel
- Creating structured markdown report for team visibility

**What to improve:**
- Could have used codetopo MCP tools for symbol search (per custom instructions)
- Should store memory about RPC naming conventions for future tasks

**Deliverables:**
- Comprehensive audit report in `.squad/decisions/inbox/illumi-frontend-audit.md`
- This history entry

**Status:** COMPLETE — No fixes needed, frontend is fully aligned

