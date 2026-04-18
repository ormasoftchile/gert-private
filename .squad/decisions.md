# Squad Decisions

## Active Decisions

### 2026-03-17T00:00:00Z: Visual runbook editor direction
**By:** Gon
**What:** Prefer a hybrid editor architecture in the VS Code extension: structured visual model + synchronized YAML source, with schema/tool/provider-driven extension points.
**Why:** Fastest path to value using existing extension and serve infrastructure, while preserving compatibility with emergent tools/providers and existing YAML-first workflows.

### 2026-03-17T00:00:00Z: Scope boundary for MVP
**By:** Gon
**What:** Keep YAML as the canonical persisted format in MVP; avoid introducing a new persisted runbook representation.
**Why:** Minimizes migration risk, keeps CLI/engine unchanged, and allows gradual rollout with reversible UX changes.

### 2026-03-17T00:00:00Z: Visual runbook editor UX direction
**By:** Kurapika
**What:** Ship a hybrid visual authoring experience in the existing VS Code extension: guided form-first editing with a synchronized read-only/limited-edit tree graph for branches and iterate blocks; persist runbook data as schema-backed JSON object and render YAML only as output/export.
**Why:** Business users need low-friction authoring and guardrails, while engineers still need transparent structure and compatibility with existing runbook engine, schema validation, and execution workflows.

### 2026-03-17T00:00:00Z: Extensibility and safety defaults
**By:** Kurapika
**What:** Use schema + tool-definition introspection to render dynamic fields for unknown future tools/providers, and enforce governance by default with visible policy panels, deny/approval hints, and preflight validation gates before save.
**Why:** Gert supports evolving tool/provider ecosystems and strict governance requirements; static form layouts would become brittle and unsafe.

### 2026-03-17T03:05:32Z: Backend contracts for visual editor MVP
**By:** Killua
**What:** Add backend read APIs for editor metadata rather than changing runtime semantics: expose canonical schema bundle (runbook/tool/project), effective tool/provider catalogs, and validation diagnostics via serve/CLI.
**Why:** UI can be generated from backend-owned contracts, preserving YAML compatibility and avoiding hardcoded UI logic tied to specific providers/transports.

### 2026-03-17T03:05:32Z: Compatibility policy for visual editing
**By:** Killua
**What:** Keep YAML as canonical, preserve v0/v1 plus shorthand parsing, and apply non-destructive normalization when saving from editor (retain unknown ordering/comments where feasible, never rewrite unrelated sections).
**Why:** Existing hand-written runbooks and tool packs must keep working without migrations or forced format shifts.

### 2026-03-17T03:05:32Z: Minimal Go MVP changes
**By:** Killua
**What:** Implement small additive APIs only: schema export expansion (tool/project), serve endpoints for runbook/tool/provider metadata and diagnostics, and index/list helpers in schema/tools for discoverability.
**Why:** Enables a robust editor MVP quickly with low regression risk and no engine behavior changes.

### 2026-03-17T00:00:00Z: Azure integration model should use existing tool/provider abstractions
**By:** Leorio
**What:** For visual runbooks that mix CLI and Azure operations, model Azure actions as governed tool actions (JSON-RPC or MCP spawn transport) and provider-resolved inputs, not as a new core step type. Keep governance in shared runbook/tool policy fields (rules, redaction, deny env vars, requires approval) and expose them in editor UX as explicit risk/approval/redaction controls.
**Why:** Existing contracts already separate execution from transport and include approval/redaction/deny logic in runtime; extending via tool/provider schemas minimizes coupling and keeps enforcement centralized.

### 2026-03-17T00:00:00Z: Visual runbook editor quality strategy
**By:** Hisoka (QA Reviewer)
**What:**
- Treat schema parity and YAML round-trip fidelity as release blockers for the visual editor MVP.
- Require compatibility coverage across `runbook/v0` and `runbook/v1` in both Go validation and VS Code extension validation.
- Enforce reviewer gates for deterministic serialization, deep-reference integrity, and governance-safe editing before GA.
**Why:**
- The editor path introduces a second schema-validation and serialization surface. Without explicit gates, regressions can bypass CLI/runtime safety guarantees.

### 2026-03-18: Visual paradigm references — n8n and Logic Apps Designer
**By:** ormasoftchile (via Copilot)
**What:** Preferred design references for the visual editor are n8n and Azure Logic Apps Designer.
**Why:** User confirmed affinity after reviewing workflow UI references. These two guide visual language, interaction patterns, and UX decisions for the workflow view.

### 2026-03-18: Workflow graph as optional alternative view
**By:** ormasoftchile (via Copilot)
**What:** The execution viewer (RunbookPanel) supports both the current tree view AND a new workflow graph view. Users toggle between them; neither replaces the other.
**Why:** Preserves existing behavior while introducing the workflow paradigm as a discoverable alternative. Reduces adoption friction.

### 2026-03-18: Workflow graph direction confirmed
**By:** ormasoftchile (via Copilot)
**What:** Workflow graph view for runbook execution is confirmed as the right direction. Current SVG prototype is directionally correct — polish and iteration should follow.
**Why:** User explicitly confirmed: "this is the way."

### 2026-03-18: Workflow view is the default for both authoring and execution
**By:** ormasoftchile (via Copilot)
**What:** Both the authoring experience (RunbookEditorPanel) and the execution experience (RunbookPanel) should be workflow-graph-first. A tree view adds little value over editing YAML directly.
**Why:** The visual tool's value proposition is showing the workflow — conditions, branching paths, sequence — not the document structure.

### 2026-03-18: Form-first + synchronized tree map dual-panel design
**By:** Gon (Lead Architect)
**What:** Editor uses a dual-panel hybrid model: left panel (60%) form-based authoring, right panel (40%) synchronized read-only flow map. Top bar with validation/governance badge, save, preview diff, cancel.
**Why:** Kurapika's Phase 1 MVP flattened branch/conditional structures; this design preserves runbook structure while remaining form-first for business users.

### 2026-03-18: Workflow-first visual paradigm architecture
**By:** Gon (Lead Architect)
**What:** Tree data already carries all topology info needed for graph derivation (client-side transform). Two additive backend events needed for execution annotation: `event/branchResolved` and `event/iteratePassEnd`.
**Why:** No new data shape required for static topology. Execution path annotation needs minimal additive Go events, zero-cost if UI not listening.

### 2026-03-18: Safety guards for non-linear runbook editing
**By:** Killua (Backend Engineer)
**What:** Block add/remove/reorder operations when branches or iterates are present. Allow metadata-only editing when tree has complex structures. Accept YAML reformatting as MVP limitation.
**Why:** Non-linear structures are incompatible with linear step extraction/rebuild pattern; safety guards prevent data loss.

### 2026-03-18: TreeNode structure authority (Go schema is canonical)
**By:** Killua (Backend Engineer)
**What:** Go schema is the authority. TreeNode = {step?, iterate?, branches?}. Branches are structural containers, NOT a step type. Step.Branches (legacy) should not be used.
**Why:** TypeScript form was treating branch as a step type, causing schema mismatch. Clarification unblocks correct form design.

### 2026-03-18: Phase 1 visual runbook editor MVP
**By:** Kurapika (Frontend Engineer)
**What:** Implemented Phase 1 MVP with YAML as single source of truth, simplified flat tree model, form-first UI, client-side message passing. Limitation: does not preserve branch/routing structure.
**Why:** Fastest path to value for linear runbooks; branch preservation deferred to Phase 2.

### 2026-03-18: Tree-based runbook editor MVP (Phase 2)
**By:** Kurapika (Frontend Engineer)
**What:** Rebuilt runbookEditorPanel.ts with tree-based architecture preserving branches, iterates, conditionals. 2-column dual-panel layout with path-based navigation and round-trip safe serialization.
**Why:** Phase 1 flattened structures; tree-based model aligns with Go schema and preserves all runbook structure.

### 2026-03-18: UI specification for runbook visual editor
**By:** Kurapika (Frontend Engineer)
**What:** Defined 2-column adaptive grid layout (60% form / 40% tree), header toolbar with validation badge, governance summary, step navigator, and responsive breakpoint behavior.
**Why:** Provides concrete interaction design spec for the editor implementation.

### 2026-03-18: Add-step handler bugs fixed
**By:** Kurapika (Frontend Engineer)
**What:** Fixed 3 critical bugs: iterate node detection in getSelectionType(), double-appended 'steps' in branch path construction, missing iterate path prefix in add-step handler.
**Why:** Bugs prevented step insertion into branch and iterate nodes.

### 2026-03-18: Schema alignment fix applied — all Hisoka blockers resolved
**By:** Kurapika (Frontend Engineer)
**What:** All 6 Hisoka blockers fixed. Data model corrected (TreeNode with optional step/iterate/branches), form rendering is context-aware, selectedStepPath renamed to selectedNodePath.
**Why:** Resolves structural data loss vectors identified in QA review.

### 2026-03-18: Workflow graph view POC implemented
**By:** Kurapika (Frontend Engineer)
**What:** Implemented toggleable SVG workflow graph visualization in RunbookPanel. New treeToGraph.ts module with deterministic hierarchical layout. Nodes typed (step, condition, join, iterate, start, end), edges classified (sequential, conditional, back-edge), execution path highlighted green.
**Why:** Delivers the confirmed workflow-first visual paradigm for the execution viewer.

### 2026-03-18T16:32:00Z: MVP editor rejection — 6 critical blockers (SUPERSEDED)
**By:** Hisoka (QA Reviewer)
**What:** Rejected MVP due to 6 blockers: invisible iterate blocks, step types include structural types, branch condition maps to wrong object, unmapped fields, unsafe save, missing validation.
**Why:** Structural data loss vectors in TypeScript UI model vs Go schema.
**Status:** SUPERSEDED by Hisoka's subsequent approval after all blockers were fixed.

### 2026-03-18T21:15:00Z: MVP editor approved — all blockers fixed
**By:** Hisoka (QA Reviewer)
**What:** Approved MVP for manual testing. All 6 blockers verified fixed: iterate blocks visible, step types correct, branch conditions mapped to Branch object, form rendering context-aware, YAML round-trip safe.
**Why:** Editor data model now correctly aligns with Go schema.

### 2026-03-18: Visualization adapts to host environment

**By:** Cristián Ormazábal Ortega (via Copilot)  
**Status:** Directive  

**What:** Run visualizations (tree view, workflow graph) must be host-environment-dependent. VS Code and web can support full graph/SVG rendering with multiple view modes. A TUI version must restrict to tree-based layout only. The visualization layer should be designed with an adapter pattern — not hardcoded to a single rendering surface.

**Why:** gert targets multiple surfaces (VS Code extension, web, TUI). The rendering engine needs a clean abstraction boundary so each host picks the best visualization it can support.

---

### 2026-03-18: Authoring editor graph integration (Phase B)

**By:** Gon (Lead Architect)  
**Status:** Implemented  

**What:**
1. Graph as view, tree as truth — SVG graph derived from `TreeNode[]` via `treeToWorkflow()`. All mutations operate on tree model.
2. stepId-based click mapping via `buildStepIdToPathMap()`, decoupled from treeToGraph internals.
3. Selection type determined by node content, not path shape.
4. Structural coloring (step-type-based) in authoring context, not execution coloring.

**Why:** Branches and iterate blocks must survive edit→save→reload. Graph and form track same `selectedNodePath`. Kurapika can enhance `treeToGraph.ts` independently.

**Risks:** Shallow branch traversal may show incomplete graphs for complex runbooks. Iterate nodes without `id` fields won't be clickable (acceptable for MVP).

---

### 2026-03-18: Test suite strategy for graph + editor contracts

**By:** Hisoka (QA Reviewer)  
**Status:** Implemented  

**What:** Two automated test suites (71 total tests, all passing): `treeToWorkflow()` transformation contract (37 tests) and editor YAML round-trip fidelity contract (34 tests). Implementation-independent — tests inputs/outputs, not internals.

**Why:** Kurapika and Gon both modifying graph/editor code simultaneously. Contract tests prevent invisible regressions.

**Constraints:** Tests must not break on treeToGraph or editor panel internal refactors. Jest with ts-jest.

**Risks accepted:** No webview integration tests this sprint. Inline fixtures only. Empty branch arm orphaned path flagged for Kurapika.

---

### 2026-03-18: Execution path event design

**By:** Killua (Backend Engineer)  
**Status:** Implemented  

**What:** Two new additive JSON-RPC events:
1. `event/branchResolved` — per branch condition evaluation: `{parentStepId, branchIndex, condition, taken}`.
2. `event/iteratePassEnd` — per iterate pass completion: `{iterateStepId, pass, max, converged}` with synthetic `iterate-<stepIdx>` IDs.

**Why:** Frontend needs per-branch and per-pass granularity for graph annotation. Synthetic IDs avoid schema changes. Zero-cost when no UI listener active. Purely additive — no engine behavior changes.

---

### 2026-03-18: Execution viewer graph — production layout & rendering

**By:** Kurapika (Frontend Engineer)  
**Status:** Implemented  

**What:** Production-quality top-to-bottom recursive bounding-box layout replacing POC left-to-right BFS. Themed SVG rendering with VS Code CSS custom properties, distinct node shapes per type, zero runtime dependencies.

**Why:** Users running incident response runbooks need at-a-glance flow understanding. Full rewrite of `treeToGraph.ts` layout and `runbookPanel.ts` `renderGraphSvg()`.

---

### 2026-03-18: Peek YAML, Tool Catalog, and Scenario Browser UX

**By:** Gon (Lead Architect)
**What:** Three editor UX decisions: (1) Peek YAML uses a webview-side context menu + modal overlay, not VS Code native UI, keeping context inside the graph editor. (2) Tool catalog is a separate webview panel (`gert.showToolCatalog`) with dual data source — JSON-RPC `tools/list` first, fallback to scanning `*.tool.yaml` from disk. (3) Scenario browser is embedded in the Runbook Home section of the editor, not a separate panel.
**Why:** Peek YAML keeps focus in the editor. Dual-source tool catalog works offline-first without `gert serve`. Inline scenario list avoids panel proliferation since scenarios are tightly coupled to the runbook being edited.

---

### 2026-03-19: Graph redesign — 4-layer architecture and golden snapshot testing

**By:** Gon (Lead Architect)
**What:** Redesigned the graph/tree rendering into a 4-layer pipeline: (1) Tree Operations (`treeOps.ts` — merge invoke children, prune trailing unexecuted), (2) Connectors (`treeConnectors.ts` — extract branch arrows, iterate badges, markers), (3a) Text Renderers (`textTreeRenderer.ts`, `textGraphRenderer.ts`), (3b) Semantic Graph + SVG/Mermaid renderers. Tree is source of truth; graph is a derived view. State machine (`snapshotStateMachine.ts`) is a pure reducer with no VS Code dependency. Golden snapshot testing replays `.jsonl` recordings through the state machine + renderers, producing `.golden` files with three assertion modes (FULL, DIFF, INVARIANTS). CI enforces golden parity — any visual output change breaks the build until explicitly approved.
**Why:** Prior merge/prune logic was embedded in the graph renderer (~80 lines), untestable without a webview. The new architecture enables deterministic offline testing of both tree and graph views, catches visual regressions automatically, and ensures tree/graph parity through shared data pipeline.

---

### 2026-03-18: Invoke visualization — inline expansion with breadcrumb fallback

**By:** Gon (Lead Architect)
**What:** After researching 7 platforms (n8n, Azure Logic Apps, Temporal, Prefect, Dagster, GitHub Actions, Airflow), adopted a hybrid of Airflow TaskGroups + Dagster breadcrumbs for invoke/child runbook visualization. Default: invoke node expands inline as a container showing child steps with live state updates. Overflow (>8 steps): collapses to summary bar with progress indicator. Large children (>20 steps): forces drill-down with breadcrumbs. Two new additive serve events: `event/invokeStart` (with child tree) and `event/invokeEnd`. Implementation in 3 phases: (1) summary bar, (2) inline expansion with container node layout, (3) auto-select hybrid.
**Why:** Current abrupt context-switch on invoke was explicitly rejected by users. Black-box and separate-page models (n8n, Logic Apps, Temporal) provide no live child visibility. Inline expansion gives the core ask — seeing child progress from the parent — while breadcrumb fallback handles complex children.

---

### 2026-03-18: Query syntax highlighting architecture

**By:** Hisoka (QA Reviewer)
**What:** Query fields (`query`, `command`, `expression`, `filter`) receive SQL/KQL syntax highlighting via two mechanisms: (1) YAML source view uses TextMate injection grammar with repository rules in `runbook-injection.json`. (2) Visual editor uses transparent-textarea + overlay-div pattern with a JS tokenizer, rendering query fields as individually highlighted textareas separate from the JSON inputs blob.
**Why:** Query expressions are the most common complex content in tool args. Two-layer approach ensures highlighting in both YAML and form editing. The overlay pattern is cosmetic only — preserves plain text values, avoiding model corruption.

---

### 2026-03-18: Per-step retry, parallel iterate, and error branches

**By:** Killua (Backend Engineer)
**What:** Three engine enhancements: (1) Steps can declare `retry: {max, interval, backoff}` with linear/exponential backoff, emitting `event/stepRetrying` for UI. (2) IterateBlock accepts `concurrency` field for parallel list-mode execution via goroutines with scoped variable copies (convergence mode stays sequential, approval steps force concurrency=1). (3) `condition: "_error"` is a reserved branch condition that activates on parent step failure (after retries), with error message available as `{{ .error }}`.
**Why:** Retry with backoff is standard for transient failures. Parallel iterate benefits multi-region rollouts. Error branches enable structured recovery flows (escalation, fallback, cleanup) without requiring full stop or blind continue.

---

### 2026-03-19: RunbookPanel uses applyEvent() as single source of truth

**By:** Kurapika (Frontend Engineer)
**What:** `RunbookPanel` no longer maintains parallel state logic. All event handlers delegate to `applyEvent()` from `snapshotStateMachine.ts`. The panel stores the returned `SnapshotState` and exposes `stepStates`, `invokeChildren`, and `branchResolutions` as getters. Future event types must be handled in `snapshotStateMachine.ts` first.
**Why:** The panel had divergent state logic (manual `.set()` calls, `runtimeToGraphIds` cross-pollination) causing bugs not covered by golden snapshot tests. Using `applyEvent()` ensures the exact code path tested in golden files runs in production.

---

### 2026-03-19: Graph end-node placement uses edge-walk depth

**By:** Kurapika (Frontend Engineer)
**Status:** Implemented
**What:** End node and trailing unexecuted node placement determined by edge-walk depth from start (BFS traversal distance), not pixel Y-position. Walks taken path only — follows edges through executed and structural passthrough nodes, stops at pending step nodes. Three-phase post-layout: (1) `omitEnd: true` during layout, (2) dead-end join removal, (3) edge-walk to find last executed step, prune trailing, place fresh end node.
**Why:** Y-position fails on runbooks with uneven branch depths or multiple nesting levels. Edge-walk depth is topologically correct regardless of runbook shape.

---

### 2026-03-19: Fully-qualified prefix resolution for nested invoke states

**By:** Kurapika (Frontend Engineer)
**What:** Snapshot state machine resolves `parentStepId` to its fully-qualified prefix before storing prefixed step states. New `resolveInvokePrefix()` helper searches existing `stepStates` for keys ending with `::parentStepId` that have running/passed state. `invokeChildren` map stores entries under both raw and FQ keys for dual-lookup compatibility.
**Why:** Nested invokes produce 1-level-deep runtime IDs but the merged tree uses fully-qualified prefixed IDs. Without resolution, `pruneTrailingUnexecuted` fails exact-match lookups and prunes executed children. Old golden test masked this via CRLF parsing bug (vacuous pass).

---

### 2026-03-18: YAML↔Visual dual-mode editor with expression autocomplete

**By:** Kurapika (Frontend Engineer)
**What:** Runbook editor supports two modes toggled via tab bar: Visual (form + graph, default) and YAML (raw source with line numbers). Expression autocomplete triggers on `{{ .` pattern showing available variables and Sprig functions. Graph edges have "+" insert points for quick step insertion (1 click vs. 3).
**Why:** YAML mode gives engineers direct source access without leaving the editor panel. Model re-parsed on tab switch keeps views synchronized (YAML canonical). Expression autocomplete reduces template expression errors.

---

### 2026-03-18: User directive — no temp files outside workspace

**By:** Cristián Ormazábal Ortega (via Copilot)  
**Status:** Directive  

**What:** Never create temporary files outside the working directory. All temp/scratch files must stay within the project workspace folder.

**Why:** User directive — keeps the filesystem clean and all artifacts discoverable within the repo.

---

### 2026-03-18: Editor backend APIs are stateless read-only endpoints

**By:** Killua (Backend Engineer)  
**Status:** Implemented  

**What:** Added six JSON-RPC endpoints (`tools/list`, `tools/get`, `exec/dryRun`, `schema/stepFields`, `schema/toolArgs`) to the serve layer. All are stateless and read-only — they do not create engines, modify server state, or touch execution sessions.

**Why:** The visual editor needs tool catalog data, argument schemas, and dry-run validation to drive dynamic form rendering and preflight checks. These must come from the Go backend (source of truth for schema, tool parsing, and governance rules) rather than being hardcoded in TypeScript.

**Key constraints:**
- Tool discovery uses existing `schema.DiscoverProject` + `project.ResolveToolRef` — no new resolution logic.
- `exec/dryRun` validates with the full 3-phase pipeline but never instantiates an engine or executor.
- `schema/stepFields` returns step-type field definitions derived from Go struct tags — if schema structs change, this endpoint must be updated.
- All endpoints accept optional `cwd` to support multi-root workspace scenarios.

**Impact:** Kurapika can replace hardcoded field lists with `schema/stepFields` and `schema/toolArgs` calls. Gon's editor architecture can use `tools/list` for the tool picker and `exec/dryRun` for save-time validation. Hisoka can gate on `exec/dryRun` governance warnings for reviewer checks.

---

### 2026-03-18: Graph rendering — visual readability defaults

**By:** Kurapika (Frontend Engineer)  
**Status:** Applied  

**What:** Node width 240px (was 180) with 28-char truncation. Join nodes invisible (4px, 0.4 opacity). Minimum untaken branch opacity 0.55. Bezier convergence adds `dx * 0.3` to control points.

**Why:** Live test on `service-health-branching` revealed overlapping labels, confusing join nodes, unreadable dim branches, and crossing edges.

---

### 2026-03-18: Editor graph click fix, vizAdapter abstraction, comment-preserving save

**By:** Gon (Lead Architect)  
**Status:** Implemented  

**What:**
1. Fixed iterate graph nodes not responding to clicks — missing `.id` on in-editor iterate blocks broke stepId→path mapping. Fix assigns IDs on creation and backfills synthetic IDs in `buildStepIdToPathMap()`.
2. Created `vizAdapter.ts` with `VizAdapter` interface, `VizHost` type, and `createVizAdapter()` factory. Graph adapter wraps `treeToWorkflow` + SVG. TUI adapter renders indented text tree.
3. Save handler now uses `YAML.parseDocument()` for comment-preserving AST round-trips via `doc.set()`.

**Why:** Click→form is the primary navigation contract. Host-adaptive viz fulfills the user directive for multi-surface rendering. Comment preservation fulfills Killua's compatibility decision (2026-03-17) for non-destructive normalization.

---

### 2026-03-18: Governance display contract via JSON-RPC

**By:** Leorio (Integrations Engineer)  
**Status:** Implemented  

**What:** `governance/evaluate` read-only JSON-RPC endpoint in `ext/serve`. Takes runbook path, evaluates governance policy per step without executing, returns structured assessment. TypeScript interfaces in `vscode/src/serve/governance.ts`. Client method `governanceEvaluate()` wired in `client.ts`.

**Why:** Visual editor needs inline governance info (risk badges, approval gates, redaction indicators, policy violations) without running the runbook. Decoupled from execution per "backend read APIs for editor metadata" decision (Killua, 2026-03-17).

**Key choices:** Walk-all-nodes (badges every visible node), merged governance sources (single call), no execution side effects (safe on every open/save), arrays default to `[]` not `null`.

---

### 2026-03-18: Branch condition builder expression patterns

**By:** Gon (Lead Architect)
**Status:** Proposed

**What:** The branch condition builder generates Go template expressions using a fixed set of patterns: `{{ contains .var "value" }}`, `{{ eq .var "value" }}`, `{{ regexMatch "pattern" .var }}`, `{{ gt .var value }}` / `{{ lt .var value }}`, and their negations. Unrecognized patterns fall through to raw expression editing (Advanced mode).

**Why:** These patterns cover the most common branch conditions observed in example runbooks. The Go template function names match the Sprig library used by the gert engine. Power users can always toggle to Advanced mode for arbitrary expressions.

**Impact:** If the engine's template function set changes, the builder patterns must be updated. UX may want visual indicators when a condition can't be parsed. Round-trip test coverage should include builder-generated → save → reload → parse cycle.

---

### 2026-03-18: Missing testdata fixtures block 28 Go tests

**By:** Hisoka (QA Reviewer)
**Status:** Flagged

**What:** Go test run across `pkg/...` reveals 28 test failures in 5 packages caused by missing `testdata/` directory. Additionally, 11 iterate tests in `pkg/engine` fail with `unknown step type: ""` due to fixtures missing the `type` field.

**Why:** CI pipeline (`go test ./pkg/...`) will report failures on every run. Cannot distinguish new regressions from known fixture gaps.

**Recommendation:** Add `testdata/` fixtures or skip dependent tests with `t.Skip()`. Update engine iterate test fixtures to include a valid `type` field. Consider splitting CI into core tests (passing) and integration tests (fixture-dependent).

---

### 2026-03-18: Schema bundle endpoint and tool catalog array format (v2)

**By:** Killua (Backend Engineer)
**Status:** Implemented

**What:** `schema/bundle` generates all three JSON schemas from Go structs at runtime (no static files). `tools/list` returns actions as an array `[{name, description, args}]` instead of a map. `exec/dryRun` returns a `steps` array with `{id, type, title, dependencies}` plus a separate `warnings` array.

**Why:** Runtime schema generation guarantees schemas match Go types (no drift). Array format for actions is natural for frontend rendering. Steps + dependencies array enables graph rendering without tree parsing. Splitting warnings from errors lets UI differentiate non-blocking issues.

**Impact:** Frontend consumers should use array-format actions from `tools/list`. `tools/detail` is an alias for `tools/get`. `schema/bundle` is stateless and callable before any execution session.

---

### 2026-03-18: Zoom range and default behavior

**By:** Kurapika (Frontend Engineer)
**Status:** Applied

**What:** Graph pan/zoom uses 0.3–2.0 range (not 3.0), default zoom fits SVG to container width, reset returns to fit-to-width.

**Why:** Max 2.0 keeps text legible at extreme zoom. Fit-to-container means large runbooks (15+ nodes) are immediately visible without manual zoom-out. Fixed 1.0 default cut off wide graphs requiring immediate user interaction.

---

### 2026-04-06T02:47:54Z: Web application feasibility and phased delivery

**By:** Gon (Lead Architect)  
**Status:** Proposed

**What:** Add a standalone web application to gert in 4 phased deliverables: (1) HTTP transport + RunbookRunner MVP, (2) Playwright test infrastructure, (3) RunbookEditor + ToolCatalog views, (4) optional shared component library. Estimated 6-8 weeks to full feature parity.

**Why:** The existing JSON-RPC API, mature TypeScript view components, and host-adaptive visualization layer (`vizAdapter.ts`) make web porting highly feasible. Enables AI agents to autonomously verify UI changes via Playwright E2E (critical business goal). Phased approach de-risks and validates architecture early before committing full build.

**Key decisions:**
- Fork view components to `web/` directory (avoid tight coupling); share only pure business logic (state machine, graph rendering, helpers)
- HTTP transport + WebSocket for event streaming (mirrors stdio semantics)
- Playwright for E2E verification with agent-friendly structured JSON output
- Team assignments: Illumi (web shell + views), Killua (HTTP transport), Knov (Playwright), Hisoka (QA strategy)
- Risks accepted: code duplication until Phase 4, HTTP attack surface (localhost-only mitigates), WebSocket complexity

**Impact:** Unblocks web-native use cases. Enables broader runbook adoption. Establishes autonomous agent verification loop. Does not replace VS Code extension (additive). Web app runs localhost-only in Phase 1 (multi-tenant deployment out of scope).

---

### 2026-04-06T02:47:54Z: VS Code extension component portability assessment

**By:** Illumi (Web Frontend Engineer)  
**Status:** Analysis Complete

**What:** Inventoried all 3 WebView panels (~4,800 lines TypeScript). Assessed VS Code API dependencies, identified reusable core, and estimated effort per component. ToolCatalogPanel is 95% portable; RunbookPanel 70%; RunbookEditorPanel 40%.

**Why:** Determines feasibility and effort for web port. Identifies critical blockers (file I/O, workspace management) requiring new server APIs. Confirms core business logic (state machine, graph rendering, helpers) is 100% reusable.

**Key blockers for web:**
- File operations → Server API endpoints needed (`GET /api/files`, `POST /api/files`)
- Workspace management → Server API (`GET /api/workspace`)
- Dialogs/input → Replace with HTML forms
- Command execution → Direct API calls

**Reusable modules (100% portable):**
- `snapshotStateMachine.ts` (287 lines)
- `treeToGraph.ts` (708 lines)
- `graphRenderer.ts` (889 lines)

**Tech stack recommendation:** Vite + TypeScript + Web Components (no React initially, leverage existing HTML generation pattern). HTTP client via `fetch()` + WebSocket.

**Impact:** Unblocks Phase 1 porting. Requires server extension work (Killua). Component extraction and refactoring work is straightforward (8-12 days per major view).

---

### 2026-04-06T02:47:54Z: Playwright testing strategy for autonomous agent verification

**By:** Knov (Playwright Specialist)  
**Status:** Proposed

**What:** Comprehensive Playwright architecture with 15 core test scenarios, Page Object Model pattern, fast persistent test fixtures (sub-5-second startup), and structured JSON output format enabling AI agents to autonomously verify UI changes without human intervention.

**Why:** Enables primary business goal (agent self-verification). Fast feedback loop (<5 min) unblocks rapid iteration. Flakiness mitigation (explicit state waiters, streaming indicators, graph-ready signals) ensures reliable CI gating.

**Key design principles:**
- Agent-first output (structured JSON with pass/fail verdict)
- Page Object Model for maintainability
- `data-testid` discipline for stable selectors
- Explicit async waiters (no `waitForTimeout()`)
- Deterministic test data (fixture-based)

**15 core scenarios:**
- Runner: linear execution, tool invocation, branching, iterates, graph rendering, error handling (R1-R10)
- Editor: YAML round-trip, real-time sync, metadata editing (E1-E3)
- Catalog: tool discovery, detail drill-down (C1-C2)

**Autonomous dev loop:** Agent makes change → `npm run test:e2e --grep "..."` → read `test-results/agent-report.json` → interpret PASS/FAIL.

**CI integration:** GitHub Actions workflow runs after go-build-test succeeds (parallel to vscode-build-test). Headless Chrome, artifact capture on failure.

**Implementation timeline:** 4 weeks (foundation, runner tests, editor/catalog tests, CI hardening).

**Impact:** Agents can verify UI changes autonomously. Establishes fast feedback loop. Test suite becomes living documentation of expected behavior. Enables confident refactoring and feature additions.

---

### 2026-04-06T00:00:00Z: Tab re-entrancy guard for WebSocket stability
**By:** Killua
**What:** Always guard tab navigation with currentTab state tracking to prevent duplicate load/dispose cycles. Pattern: check `if (this.currentTab === tabName) return;` before reloading.
**Why:** Playwright click() can trigger event handlers multiple times; without guards, UI recreation loops cause WebSocket disconnect loops and other state corruption. This pattern should be used in all tab-based UIs.
**Status:** APPROVED

### 2026-04-06T00:00:00Z: Use data-testid for stable test selectors
**By:** Illumi
**What:** Always use `data-testid` attributes for test targeting instead of text-based selectors like `:has-text()`. Selectors must be specific and deterministic.
**Why:** Text selectors are fragile, non-deterministic, and match unintended elements. data-testid provides stable, explicit test contracts that don't break with UI text changes.
**Status:** APPROVED

## Governance

- All meaningful changes require team consensus
- Document architectural decisions here
- Keep history focused on work, decisions focused on direction

---

## Recent Decisions (2026-04-07)

### 2026-04-07: User directive — web feature parity is non-negotiable

**By:** ormasoftchile (via Copilot)
**Date:** 2026-04-07

**What:** Chain views, prune, iterate pass selection, annotation badges, and minimap are REQUIRED features for the web version — not optional. The shared renderer must include ALL VS Code graph features. Missing features must be implemented, not left as web-only gaps.

**Why:** User requirement — web must be functionally equivalent to VS Code. Gaps are bugs, not deferred scope.

**Impact on plan:** Phase 1 of the shared renderer refactor must implement missing graph features DIRECTLY in the shared package (not in web first and then refactor). Implement once, both platforms consume.

---

### 2026-04-07: Shared Renderer Phase 1 — Full Feature Parity Architecture

**By:** Gon (Lead Architect)
**Date:** 2026-04-07
**Status:** PROPOSED

**What:** Phase 1 of the shared renderer refactor: port ALL 20 VS Code graph renderer features into `shared/renderer/graph/` as a single implementation consumed by both VS Code and web. No feature remains platform-specific except IPC wiring.

**Key Architecture Decisions:**

1. `GraphRenderOptions` replaces `GraphRenderContext`
   - VS Code's `GraphRenderContext` is a class with methods bound to `RunbookPanel`
   - Shared version uses a plain options bag with optional callback injection
   - Options bag is serializable, testable without mocking, and allows each platform to provide only what it has
   - Callbacks are the escape hatch for platform-specific lookups

2. Web's `branchCondition` fix becomes canonical
   - Web computes `branchCondition` with `→ true`/`→ false` at layout time
   - VS Code computes it dynamically during render
   - Shared version adopts web's approach for simplicity and self-containment

3. Web's passthrough `nodeStepMap` fix becomes canonical
   - VS Code's `nodeStepMap.set(passId, id)` on passthrough nodes maps them to the parent branch step
   - This causes `isTaken()` to incorrectly return true
   - Web correctly omits this mapping — shared version inherits the fix

4. Shared renderer owns ALL SVG generation
   - Both `renderExecutionGraph()` and `renderEditorGraph()` live in `shared/renderer/graph/renderGraph.ts`
   - Platform adapters are thin wrappers (~50 lines) that read platform settings and call the shared function
   - Single source of truth eliminates divergence

5. 10 tasks across 2 engineers, 7-9 day estimate
   - Critical path: shared treeToGraph → core renderer → platform adapters → web UI
   - Maximum parallelism: Tasks 1, 3, 5, 6, 10 can all run concurrently

**Risks:**

1. CSS variable handling — shared SVG uses `var(--vscode-*)` which web must map to `var(--gert-*)`. Mitigated by Phase 0 theme extraction.
2. Feature interaction complexity — iterate pass selection + expanded iterate + prune interact non-trivially. Task 4 (iterate state mapping) is the hardest single task.
3. VS Code golden test breakage — any SVG output change breaks existing goldens. Mitigated by adapter approach.

**Impact:**

- Web gains 20 features it was missing
- VS Code loses zero functionality
- Both platforms share a single tested renderer
- Future graph features are implemented once

---

### 2026-04-07: Phase 0 — Shared Renderer Extraction Complete

**Agent:** Kurapika (Frontend Engineer)
**Date:** 2026-04-07
**Status:** ✅ COMPLETE

**What:** Created `shared/renderer/` — a zero-dependency package containing shared rendering logic extracted from both `vscode/` and `web/`. Both builds passing.

**Deliverables:**

- `shared/renderer/types.ts` — unified type exports
- `shared/renderer/helpers.ts` — shared utility functions with optional `highlightCode` callback
- `shared/renderer/theme/` — canonical theme implementations (graphTheme, highContrast, light)
- Updated 20 files across vscode/ and web/ to consume `@gert/renderer`

**Build Status:**

- web: `npm run build` ✅ (tsc + vite, 94 modules, zero errors)
- vscode: `npm run compile` ✅ (esbuild, 1.1mb bundle)
- vscode tsc type check: 5 pre-existing errors in `extension.ts` (`.label` on `string`) — unrelated

**Notes:**

- `GraphNode.branchCondition` was added to shared types (web had it, vscode didn't — merged)
- `highlightQuery` callback pattern ready for any renderer that wants syntax highlighting
- Ready for Phase 1 feature porting

---

## 2026-04-18T20:18:56Z: Architectural Decisions — Ken's Gap Analysis Findings

**By:** Ken (Software Architect)  
**Date:** 2026-04-18  
**Source:** Gap analysis of gert v2 design sections 00–09

These are decisions that the current design *implies* but never explicitly states. Each must be formally decided and recorded before core contract freeze.

### Decision: v2 must execute v1 runbooks in compatibility mode
**Status:** Implied, not stated  
**Implication from:** §03 (schema vNext as breaking redesign), §01 (no mention of preserving v1 behavior)  
**Options:**
1. v2 engine runs v1 YAML natively via a compatibility adapter in the parser
2. `gert migrate` auto-upgrades v1 to v2 schema before execution
3. v2 provides a `--compat` flag that engages v1 parsing semantics
4. v2 drops v1 compatibility entirely (breaking change for all existing users)

**Why this must be decided:** Without this decision, the schema section cannot define breaking changes, the migration section cannot be written, and Brian cannot implement the parser.

### Decision: Governance is host-enforced, not extension-contributed
**Status:** Implied by §07's "host enforces" framing, not stated  
**Implication from:** §07 (runtime hardening), §04 (capability-based trust)  
**What must be stated:** Does the governance layer (allowlists, denylists, redaction, approval gates) live in the core runtime, or can extensions contribute policy rules? If extensions can contribute policy, are they evaluated in-process or out-of-process? This is a security-critical decision.

### Decision: The event model is the single source of truth for adapter rendering
**Status:** Implied by §06 ("adapters render from events"), not formally decided  
**Implication from:** §06 (adapters must not implement private execution logic), §02 (adapters are thin)  
**What must be stated:** Is there *any* direct API call from adapter to core, or is every adapter interaction mediated by the event stream? The decisions log shows VS Code using JSON-RPC RPC calls (e.g., `runbook/diagram`, serve endpoints) — this contradicts a pure event model.

### Decision: JSON-RPC stdio transport is the v2 extension wire protocol
**Status:** Implied by §04 ("Transport: JSON-RPC over stdio"), partially stated  
**What must be stated:** Is this the *only* transport for extensions (i.e., gRPC is deferred per §09 open question)? If so, the spec must define the JSON-RPC method namespace, versioning, and error codes. The open question in §09 about gRPC must be resolved before the extension manifest format is frozen.

### Decision: Core runtime has no direct dependency on `gert serve`
**Status:** Implied by §01 (no UI coupling), not stated  
**Implication from:** v1 architecture where `pkg/serve` knows about runtime internals  
**What must be stated:** In v2, does `gert serve` become a thin adapter over the event stream and API layer, or does it retain privileged access to runtime internals? This directly affects the package boundary design in §02.

### Decision: Tool definitions remain file-convention-based in v2
**Status:** Implied by §05, not stated  
**Implication from:** v1 convention `tools/<name>.tool.yaml`  
**What must be stated:** Does v2 retain the `.tool.yaml` file convention, or does tool registration become exclusively extension-mediated? If both, what is the priority/override order?

### Decision: SHA256 evidence hashing and append-only JSONL trace format are preserved in v2
**Status:** Not stated anywhere in v2 design  
**Implication from:** v1 README, operational requirements for tamper-evident audit  
**What must be stated:** Are the trace format, evidence model, and resumption contract preserved as-is, redesigned, or removed? This is a first-class operational feature in v1 that is entirely absent from the v2 design.

### Decision: The "planner" produces a flat ordered step list with resolved branches
**Status:** Implied by §02 mentioning "planner", never defined  
**What must be stated:** What does the planner output? A DAG? A flat sequence? A tree? This is foundational for the runtime state machine and the event model.

### Decision: Extension capability taxonomy (initial v0 set)
**Status:** Referenced in §04/§07, never defined  
**Proposed initial capability set (to be ratified):**
- `schema.register` — contribute schema extensions
- `tool.register` — register tool definitions
- `event.subscribe` — receive runtime events
- `governance.policy` — contribute policy rules
- `evidence.capture` — write evidence artifacts
- `run.execute` — trigger step execution (high-privilege)
- `fs.read(path)` — read files within declared scope
- `fs.write(path)` — write files within declared scope
- `network.outbound(host)` — make outbound network calls

**Why this must be decided:** Every other extension-related design decision depends on this list being agreed.

---

# Decision: Research-Backed Design Principles for Gert v2

**Date:** 2026-04-18  
**By:** Dennis (CS Researcher)  
**Status:** Proposed  
**Type:** Architectural Direction

## Context

Comprehensive research into runbook systems, workflow orchestration, governance models, and traceability patterns reveals industry best practices and academic foundations that should inform the gert v2 design.

Research covered:
- 15+ industry systems (AWS SSM, PagerDuty, Rundeck, StackStorm, Temporal, Prefect, Argo, Airflow, etc.)
- 5 academic papers and technical publications
- Multiple standards (ITIL, W3C, OpenTelemetry, SOC2, ISO27001, HIPAA)
- Policy frameworks (OPA, Cedar)

Full research brief: `.squad/tmp/dennis-research-brief.md`

## Proposed Principles

### 1. Workflow Sophistication: Learn from Orchestration Engines
**Principle:** Adopt proven workflow patterns from Temporal, Argo, and Prefect.
- Saga/Compensation Pattern: Allow steps to register compensating actions; execute in reverse on failure
- Parallel Execution: Support fan-out/fan-in for independent steps
- Retry with Backoff: Enhanced step-level retry configuration (exponential backoff, jitter)
- Idempotency Guarantees: Document and enforce idempotent step design

### 2. Policy-as-Code: Dynamic Governance at Scale
**Principle:** Integrate external policy engines (OPA) rather than hardcoding all governance logic.
- Policy evaluation hooks: pre-execution, pre-step, post-step
- Centralized policy repository (Git), versioned and tested
- Alternative Cedar rejected for multi-cloud context

### 3. RBAC for Execution: Who Can Run What
**Principle:** Role-based access control for runbook execution, not just command allowlists.
- Roles define execution permissions (e.g., `incident-responder` can run `kind: mitigation`)
- Delegated administration (project leads grant permissions within scope)
- Integration with IdPs (LDAP, SAML, OIDC)

### 4. Distributed Traceability: OpenTelemetry Integration
**Principle:** Emit OpenTelemetry spans for observability and correlation with existing monitoring infrastructure.
- Each step = one span (start, end, attributes, status)
- Trace context propagation across invoked runbooks and tools
- Correlation ID in all log entries and JSONL events

### 5. Human-in-Loop with SLA Enforcement
**Principle:** Timeout and escalation are essential for operational reliability, not optional features.
- Configurable SLA timers on manual and approval steps
- On timeout: escalate to alternate approvers, send notifications, or fail
- Multi-level approval chains (parallel and serial)

### 6. Schema Evolution: Self-Describing and Versioned
**Principle:** Schemas must be self-describing, versioned, and migratable.
- Add `$schema` field to all runbook/tool/provider YAML files
- Semantic versioning with clear compatibility guarantees
- Automated compatibility testing (old runbooks on new runtime)

### 7. Evidence Integrity: Cryptographic Assurance
**Principle:** For high-assurance environments, evidence must be cryptographically signed.
- Optional cryptographic signing of JSONL trace files
- Timestamped attestations (RFC 3161 timestamp authority)
- Chain-of-custody metadata in evidence artifacts

## Prioritization Recommendation

**v2.0 MVP:**
1. Saga/compensation pattern
2. Timeout/escalation for human steps
3. Self-describing schemas ($schema field)

**v2.1+:**
- RBAC for execution
- Policy-as-code (OPA hooks)
- Parallel fan-out/fan-in
- OpenTelemetry spans

**Future (post-v2):**
- Cryptographic log signing
- W3C PROV provenance graphs

## Next Steps

1. Ken (Architect): Review principles, incorporate into architecture chapter
2. John (Schema): Assess schema impacts of saga/compensation and timeout/escalation
3. Barbara (Integrations): Identify OPA and OpenTelemetry integration points
4. Brian (Go Runtime): Plan saga executor and timeout/escalation state machine

**Decision Requested:** Accept these principles as guiding constraints for v2 architecture.
# Integration Decisions — Barbara (Integrations Specialist)
**Date:** 2026-04-18
**Author:** Barbara
**Sections:** §04 Extension Runtime, §05 Tool Runtime, §14 Input Provider Framework

---

## Decision 1: Extension spec reference is valid — do not remove it

**What:** `specs/002-extension-runtime-v0/spec.md` exists in the repository at the path referenced. The design section now explicitly cites it as the normative specification while the design chapter provides the design-level elaboration.

**Why:** Removing a valid spec reference would orphan an existing normative document. The right fix was to correct the relationship: spec = normative/implementable, design chapter = design rationale + elaboration.

---

## Decision 2: Capability taxonomy is a named, versioned list

**What:** Extension capabilities follow the naming convention `capability/<name>` (e.g., `capability/tool-registration`, `capability/network`). The initial set of 10 capabilities is defined in §04 Table 1.

**Why:** Unnamed capability strings cannot be statically analysed or enumerated by tooling. A named list enables IDE completion, policy file validation, and extension manifest linting. Future additions are backward-compatible minor bumps to the host API version.

**Capabilities defined:**
- `capability/tool-registration`
- `capability/schema-extension`
- `capability/event-subscription`
- `capability/policy-contribution`
- `capability/provider-registration`
- `capability/file-read`
- `capability/file-write`
- `capability/network`
- `capability/env-read`
- `capability/run-state-read`

---

## Decision 3: Extension manifest format is YAML, not JSON

**What:** The `gert-extension.yaml` manifest uses YAML (not JSON as in the v0 spec). `apiVersion: extension/v2`.

**Why:** gert is a YAML-first system. All other definition files (.tool.yaml, .provider.yaml, runbooks) use YAML. Adopting JSON for manifests would require operators to maintain two formats. YAML is a superset of JSON so existing JSON manifests remain parseable.

---

## Decision 4: Extension discovery uses 4 sources with explicit precedence

**What:** Discovery order: (1) built-in, (2) workspace `.gert/extensions.yaml`, (3) runbook `extensions:` field, (4) CLI `--extension` flag. Later sources take precedence; duplicates by name are deduplicated with a warning.

**Why:** Allows workspace-level defaults while letting individual runbooks or operators override for testing. This is the same layering pattern used by tool-path resolution.

---

## Decision 5: JSON-RPC method namespace is `extension/` for lifecycle, `contributions/` for registration

**What:**
- Lifecycle methods: `extension/initialize`, `extension/ping`, `extension/shutdown`
- Registration: `contributions/list`
- Tool invocation: `tools/invoke`, `tools/cancel`

**Why:** Namespaced methods prevent collisions when extensions implement their own custom methods. The `extension/` prefix signals host-initiated lifecycle control; `contributions/` signals the registration phase.

---

## Decision 6: Tool resolution uses 4-source order, first-match wins with name collision warning

**What:** Tool name resolution: (1) dynamically registered (MCP + extensions), (2) project `tools/`, (3) required-package `tools/`, (4) built-in registry. Collision emits a warning; no error.

**Why:** Allows workspace tools to override package defaults (intentional shadowing) while surfacing unintentional collisions as warnings. Matches common package manager semantics.

---

## Decision 7: MCP tools are registered under `<server-name>/<tool-name>` composite key

**What:** When an MCP server is declared as `name: acme-mcp`, its tools are registered as `acme-mcp/lookup`, `acme-mcp/resolve`, etc.

**Why:** MCP servers can expose many tools; without namespacing, two MCP servers could collide. The composite key preserves origin traceability and enables unambiguous runbook references.

---

## Decision 8: Provider definitions use `apiVersion: provider/v2`, distinct from tools

**What:** Provider manifests use `apiVersion: provider/v2`, not `tool/v2`. The v1 `provider/v0` format is preserved for backward compatibility via automatic conversion.

**Why:** Providers have a distinct resolution contract from tools (different JSON-RPC method, caching semantics, prefix-dispatch model). Keeping them as a distinct apiVersion prevents accidental misuse and allows the provider schema to evolve independently.

---

## Decision 9: Provider composition chains are declared inline as YAML lists

**What:**
```yaml
inputs:
  api_token:
    from:
      - env.ACME_API_TOKEN
      - vault.secret/acme/api-token
      - prompt
```
Left-to-right, first successful resolution wins.

**Why:** Common enterprise pattern: try env var first (CI/CD injects it), fall back to secret manager for interactive use, fall back to prompt as last resort. The inline list is the least surprising syntax given that `from: prompt` is already a string.

---

## Decision 10: Providers are persistent processes (long-lived) for stdio-jsonrpc and mcp

**What:** Providers using `stdio-jsonrpc` or `mcp` transport are started once at pre-flight and kept alive for the full run. Providers using `stdio` are per-resolution (spawned once per call).

**Why:** External providers (Vault, PagerDuty) may need to establish authenticated sessions or maintain connection pools. Spawning once amortises that cost. `stdio` providers are stateless by definition (binary/exit model), so per-resolution is the right default.

---

## Open Questions (for Ken / team resolution)

1. **gRPC transport for extensions**: The v0 spec left gRPC as an open question. §04 declares `grpc` as a valid transport value but does not specify the proto contract. This needs resolution before the manifest format is frozen.
2. **Extension signature verification**: The v0 spec made signatures optional. Should v2 require signatures for production deployments? Suggest making them required in `allowed-environments: [real]` policy.
3. **Provider prefix length-based disambiguation**: When two providers share a common prefix root (e.g., `pd.` and `pd.legacy.`), the longest-match rule is specified. Brian should confirm the implementation handles this correctly in the prefix map.
4. **Resolution caching across run resumptions**: Currently cache scope is in-memory only. If a run is resumed, all provider resolutions are re-run. This may surprise operators. Consider a `scope: resume` option that persists resolved values in the run snapshot.

# Brian — Implementation Decisions Inbox
**Date:** 2026-04-18
**Author:** Brian (Go Programmer)
**Status:** Pending merge to `.squad/decisions.md`

---

## Decision: JSONL Trace Event Format (v2)

**What:** The v2 trace format uses a `DurableEvent` envelope with fields `seq` (int64), `type` (string), `timestamp` (RFC3339), `run_id` (string), `path` (TreePath), and `data` (json.RawMessage). Fourteen normative event types are defined in §12. Every event is followed by `fsync` for durability. The `seq` field provides total order within a run.

**Why:** v1's `TraceEvent` wraps only `StepResult` — it cannot represent governance events, tool invocations, or partial step lifecycle. The new format is additive (unknown fields tolerated) and type-discriminated, enabling typed consumers. The `seq` field enables `ReadTraceSince` for incremental polling by the serve layer.

**Impact:** John must define JSON Schema for each of the 14 event types. The serve layer must be updated to publish `DurableEvent`s to the VS Code extension (see compatibility decision below).

---

## Decision: Event Bus Uses Bounded Buffered Channels

**What:** The `EventBus` delivers to each subscriber via a buffered channel (capacity 64–256 events). If a subscriber's channel is full, the event is dropped (counted in a metric) and the engine's execution loop is NOT blocked. The `Publish` method uses a non-blocking `select { case ch <- ev: default: }` pattern.

**Why:** Blocking the execution loop on a slow adapter consumer (e.g. a TUI renderer lagging behind) would make human-in-the-loop runbooks unpredictably slow. The engine's correctness must be independent of adapter throughput. Dropped events are acceptable for display adapters (they can request catch-up via `ReadTraceSince`); they are not used for durability (that is the TraceWriter's job).

**Constraint:** The `TraceWriter.Write` path is separate from the `EventBus.Publish` path. Every event is written to the trace file AND published to the bus — these are independent operations. A slow bus subscriber never affects trace durability.

---

## Decision: Sequential Engine Loop (No Goroutine-Per-Step)

**What:** The core execution loop is single-goroutine and sequential. One goroutine owns the `RunState`, the `TraceWriter`, and the `EventBus.Publish` calls. Parallel iterate (`concurrency` field) uses a bounded worker pool; workers communicate results back to the coordinator goroutine via a results channel, not directly to the TraceWriter.

**Why:** Concurrent access to `RunState` would require pervasive locking with no correctness benefit for the common case (sequential runbook). The governance engine, snapshot writer, and trace writer are all simpler without synchronization. Parallel iterate is a special case with well-defined fan-out/fan-in semantics.

---

## Decision: Snapshot Format Must Be Versioned

**What:** All `RunState` JSON snapshots in v2 must include `"schema_version": 2`. The `LoadLatestCheckpoint` function must check this field and return an error (not silently corrupt) if it encounters a version mismatch.

**Why:** v1 snapshots have no version field. When v2 changes `RunState` (e.g. adding `InvokeStack`, `ParallelSlots`), unversioned snapshots from older runs will silently deserialize with zero values, causing incorrect resumption. Versioning enables explicit migration or rejection.

---

## Decision: OTel is Opt-In via Environment Variables Only

**What:** OpenTelemetry tracing is enabled when `OTEL_EXPORTER_OTLP_ENDPOINT` is set in the environment. When unset, the OTel SDK is initialized with a no-op tracer (`trace.NewNoopTracerProvider()`). There is no `--otel` CLI flag. Configuration is 100% via the standard OTEL environment variable convention.

**Why:** Adding a CLI flag creates a discoverability and documentation burden. The OTEL environment variable convention is the industry standard and is already understood by operators who use OTel. No-op tracer incurs zero overhead (no allocations) so always-initializing the SDK is safe.

---

## Decision: Non-Idempotent In-Flight Tool Calls Are Treated as Failed on Resumption

**What:** If a `tool/invoked` event exists in the trace with no matching `tool/responded` event (orphaned correlation ID), the behaviour on resumption is determined by the tool's `idempotent` flag. If `idempotent: true`, the tool call is re-issued. If `idempotent: false` (the default), the step is treated as failed and the normal failure path is taken.

**Why:** Re-invoking a non-idempotent tool (e.g. `send-notification`, `create-database`) on resumption would cause duplicate side effects. The safe default is to fail, forcing the operator to either declare idempotency or handle the failure branch. This matches the behaviour of Temporal's at-least-once guarantee with idempotency keys.

---

## Decision: `pkg/` Is the Core Boundary; `ext/` Cannot Be Imported by `pkg/`

**What:** The Go module enforces a strict import boundary: packages under `pkg/` MUST NOT import packages under `ext/`. The `ext/` packages (serve, tui, mcp, debug) MAY import from `pkg/`. This is enforced by a linter rule (`depguard` or `gomodguard`).

**Why:** v1 already follows this convention informally. Formalizing it in the linter ensures that future contributors cannot accidentally add an `ext/ → pkg/` import that introduces coupling between the adapter layer and the core. This is the architectural property that makes the core testable without spinning up a serve layer.

---

## Decision: v1 Scenario Files Are Forward-Compatible With v2 `gert test`

**What:** v1 scenario YAML files (with `commands` and `evidence` keys) are valid input to the v2 `gert test` runner without modification. v2 adds an optional `assertions` block; if absent, the test runner only checks that the run completes without error.

**Why:** Operators have existing scenario test suites. Requiring them to migrate scenario files before v2 adoption raises the barrier to migration. Forward compatibility for read-only scenario input (no write-back) is free — the parser just ignores the missing `assertions` block.

# Dennis — v2 Framing Decisions
**Author:** Dennis (CS Researcher)  
**Date:** 2026-04-18  
**Status:** Proposed — for team review

---

## Framing Decisions Made in §00, §01, §09

### 1. Problem Statement Framing: Three Properties, Not One

I framed the gert v2 problem as requiring three co-present properties: **executable**, **governed**, and
**traceable**. A tool that is only executable is a script runner (Bash). A tool that is only governed is
a policy engine (OPA). Only the intersection of all three defines the gert design space. This framing
should be used consistently across sections that reference gert's purpose.

**Citation:** IBM Redbook on Operational Runbooks; IJSRET Vol.10 Issue 6.

---

### 2. Success Definition is Feature-Gated, Not Process-Gated

The five success criteria in §00 are all verifiable functional outcomes, not process outcomes ("team
agreed", "document reviewed"). This was a deliberate choice to give Brian and the build team a clear
done state. If any of the five criteria are adjusted, the change should be recorded here.

**Current five criteria:**
1. v1 runbook migrates and executes via `gert migrate` without manual fixup
2. TUI + VS Code + a third-party adapter all build against published contracts without importing internals
3. Governance enforced across all execution modes including extension-invoked runs
4. Every execution produces a valid v2 trace (append-only, crash-safe, schema-valid)
5. `gert test` passes a scenario suite covering all step types, governance rules, and evidence capture

---

### 3. Non-Goals Are Feature-Scoped

All five non-goals in §01 are feature-scoped ("gert v2 does not provide X"), not process-scoped
("we will not do Y in this sprint"). This was Ken's explicit request and is the right pattern for
design documents intended as build references. If the team re-opens a non-goal, it should be promoted
to a goal with a rationale.

---

### 4. Open Questions Are Structured as Decision Records

Each question in §09 now has three parts: (a) why it matters, (b) options, (c) information needed.
This structure is intentional: it means any team member can pick up a question, gather the information
listed, and bring a proposal to the decisions log without needing to re-read the full design. Questions
that depend on each other are cross-referenced (Q2↔Q12, Q4↔Q7).

---

### 5. v1 Preservation Goals Are Explicit and Enumerated

Goal G3 explicitly lists every v1 feature that must be preserved by name: evidence capture, SHA256
hashing, scenario replay, Bubble Tea TUI, VS Code extension, `gert test`, and its assertion engine.
This prevents any of these features from being dropped silently during the v2 build. If the team
decides to drop or defer one of these, a decision record is required.

---

### 6. OTel Integration Is a Binding Goal, Not a Stretch Goal

I escalated OpenTelemetry integration from "recommendation" (research brief §7.2) to a binding goal
(G6) in §01. The reasoning: W3C Trace Context and OTel spans are now the industry standard for
operational visibility in any system that touches infrastructure. For gert to be useful in modern SRE
environments, OTel is not optional. If the team disagrees, this should be downgraded to a non-binding
recommendation and the rationale recorded here.

---

### 7. Saga / Compensation Is a Binding Goal

Similarly, saga/compensation (G5) was elevated from recommendation to goal. The research brief §2.2
is clear: without compensation, partially-executed runbooks leave infrastructure in undefined states.
This is a correctness issue, not a feature request.

---

## Questions for the Team

- **Q1 (v1 compatibility):** Should we survey existing users before the concurrency model decision?
  If so, who owns that survey?
- **Q6 (gert serve contract):** Brian should produce a JSON-RPC method inventory before contract freeze.
  Is this assigned?
- **Q11 (extension schema namespacing):** John (Schema) should validate whether JSON Schema Draft 2020-12
  supports dynamic `$ref` resolution before we commit to option (a).

# John — Schema vNext Key Decisions
**Date:** 2026-04-18
**Author:** John (YAML/Schema Specialist)
**Section:** §03 Schema vNext

---

## Decision 1: apiVersion Values for v2

**What:** v2 document kinds use `runbook/v2`, `tool/v2`, and `provider/v2` as their `apiVersion` strings.

**Why:** The `<kind>/<major>` format is consistent with v1 (`runbook/v1`, `tool/v0`) and follows Kubernetes convention. Minor/patch changes within a major do not alter the `apiVersion` string — they are tracked by the schema artifact semver.

**Breaking change definition:** A major increment (e.g. v1 → v2) constitutes a breaking change. Adding optional fields or loosening constraints does not.

---

## Decision 2: $schema Field is Recommended, Not Required

**What:** All v2 runbook, tool, and provider documents *should* include a `$schema` field pointing to `https://schemas.gert.dev/<kind>/<version>.json`. The field is optional at runtime (the runtime dispatches on `apiVersion`) but strongly recommended for IDE tooling.

**Why:** Dennis's research confirmed this is industry standard (GitHub Actions, Argo, Kubernetes CRDs). It enables zero-config IDE auto-completion and offline validation with any Draft 2020-12 validator.

---

## Decision 3: v1 Compatibility Mode in v2 Parser

**What:** The v2 parser accepts `runbook/v1` documents in compatibility mode — all v1 field paths are normalized to v2 equivalents before validation. `gert migrate` permanently rewrites to v2 canonical form. `runbook/v0` is not supported in v2 (two-stage migration required).

**Why:** Existing v1 runbooks must not break on upgrade to v2 runtime. The compatibility adapter is in the parser layer (Brian's domain); it does not leak into the runtime.

---

## Decision 4: meta.* Fields Promoted to Top-Level (Breaking Change)

**What:** The `meta:` wrapper is removed in v2. Fields `name`, `kind`, `description`, `vars`, `inputs`, `governance`, `prose`, `defaults` become top-level runbook fields. A new required top-level field `id` (machine slug) is introduced.

**Why:** The `meta:` wrapper was a historical artifact that created unnecessary nesting. Promoting fields to top-level improves document readability, reduces YAML depth, and aligns with the schema self-description pattern (top-level `$schema` and `apiVersion` are already top-level).

**Migration:** `gert migrate` handles promotion automatically.

---

## Decision 5: toolRefs Replaces Flat tools Array

**What:** The v1 `tools: [name, ...]` string array is replaced by `toolRefs: [{name, path?, alias?}]` in v2. Default discovery path remains `tools/<name>.tool.yaml`.

**Why:** The flat array forced tool names to match filenames exactly and provided no way to alias or override paths. `toolRefs` supports both conventions and enables aliasing for runbooks that reference the same tool under different names.

---

## Decision 6: step.type Inventory for v2

**What:**
- Retained: `tool`, `cli`, `invoke`, `choice`, `assert`, `parallel`, `extension`, `end`
- Renamed: `collector` → `manual`, `router` → `end` (merged)
- New: `compensate` (saga/rollback pattern)
- Removed: `noop` (use `when:` guard instead)
- `branch` is both a step type (inline) and a TreeNode sibling

**Why:**
- `manual` is a clearer name for human interaction steps than `collector`.
- `compensate` addresses the saga pattern gap identified in Dennis's research brief.
- `noop` steps can always be replaced by a `when:` guard on any step; eliminating the type reduces surface area.

---

## Decision 7: Go Templates Retained in v2 (With Function Library)

**What:** Go `text/template` remains the expression language for all interpolated fields. A standard function library is added: `contains`, `hasPrefix`, `hasSuffix`, `trim`, `toLower`, `toUpper`, `split`, `join`, `env`, `now`, `runID`, `not`.

**Why:** The v1 template engine is battle-tested and broadly used in existing runbooks. Switching to a different expression language would be a major migration burden with no clear benefit for operational use cases. The function library addresses the expressiveness gap identified in v1 (e.g. string manipulation required pipeline gymnastics).

**Template error handling change:** In v2, a template evaluation error is a hard step failure (not a soft skip). This is a behavior change from v1.

---

## Decision 8: Extension Namespace Convention — x-<namespace> Prefix

**What:** Extension fields in all v2 documents must use the `x-<namespace>` prefix pattern (e.g. `x-myorg-cost-center`). Unregistered `x-*` fields produce a WARNING; registered fields are validated against an extension schema fragment.

**Why:** Follows OpenAPI and GitHub Actions convention. Allows `additionalProperties: false` in the core JSON Schema while still permitting extension fields. The `x-` prefix is immediately recognizable as non-core.

**Hard error:** Any field that does not match a core field name AND does not start with `x-` is a structural validation error.

---

## Decision 9: Two-Phase Validation (Structural + Semantic)

**What:** `gert validate` runs in two phases:
1. **Structural** — JSON Schema Draft 2020-12 validation (field types, required fields, enums, patterns). Stateless.
2. **Semantic** — 10 domain rules (step ID uniqueness, invoke target resolution, branch condition variable scope, tool reference resolution, provider binding, circular invoke detection, capture write-before-read, governance pre-check, output completeness, compensate ordering).

**Why:** JSON Schema cannot express domain rules that require cross-referencing parts of the document or loading external files. Separating the phases gives better error messages and allows the structural pass to run without any file system access.

---

## Decision 10: tool/v2 and provider/v2 Schema Changes

**What:**
- `tool/v2` adds: `capture.stdout.format` (json|text|lines), per-action `contract`, per-action `governance`, `meta.version`.
- `provider/v2` adds: typed `fields` map with schema declarations, `meta.kind` (input-provider | enrichment-provider), `capabilities` array.

**Why:** `capture.format` enables the `json.<path>` capture syntax in runbook steps. Per-action contracts allow the governance engine to evaluate effects without step-level contract overrides. The `fields` map in provider definitions enables the semantic validator to check `from: <provider>.<field>` bindings at validate time.

---

## Decision 11: Approval Gate Gains Timeout and Escalation (v2)

**What:** The `approvals` block on `manual` steps gains three new fields: `timeout` (duration), `on_timeout` (escalate|fail|skip), `escalate_to` (array of role/email strings).

**Why:** Dennis's research identified SLA enforcement for human steps as a gap in v1. Without timeouts, a runbook can stall indefinitely waiting for human approval. The escalation chain aligns with ITIL change management practice.

---

## Decision 12: parallel Step Type with Join Semantics

**What:** A new `type: parallel` step executes multiple branch groups concurrently. The `join.wait_for` field specifies `all | any | majority`. Captures from parallel branches are merged (last-writer-wins with a warning on conflict).

**Why:** Parallel fan-out/fan-in is a standard workflow pattern identified in Dennis's research brief. It reduces total execution time for independent checks (e.g. DNS + HTTP probes running simultaneously).

---

## Compatibility Matrix

| Feature | v0 | v1 | v2 |
|---------|----|----|-----|
| apiVersion field | runbook/v0 | runbook/v1 | runbook/v2 |
| meta wrapper | ✓ | ✓ | ✗ (promoted) |
| $schema field | ✗ | ✗ | ✓ (recommended) |
| tools: string array | ✓ | ✓ | ✗ (toolRefs) |
| type: collector | ✗ | ✓ | ✗ (→ manual) |
| type: compensate | ✗ | ✗ | ✓ (new) |
| type: parallel | ✗ | ✓ | ✓ |
| Input type field | ✗ | ✗ | ✓ |
| Output declarations | ✗ | ✗ | ✓ |
| Approval timeout | ✗ | ✗ | ✓ |
| json.<path> capture | ✗ | ✗ | ✓ |
| Extension x- prefix | ✗ | ✗ | ✓ |

# Ken — Architectural Decisions
**Date:** 2026-04-18
**Author:** Ken (Software Architect)
**Status:** Pending team review

These decisions were made in the process of authoring §02 (Architecture) and §11 (Governance and Policy)
for the gert v2 design document.

---

## AD-01: ExecutionPlan is a flat ordered list, not a DAG

**Decision:** The `ExecutionPlan` produced by the Planner is a flat, ordered list of resolved steps.
Branch and iterate control flow are runtime decisions made by the Runtime Core against evaluated
expressions. The Planner does not compute which branches will be taken.

**Rationale:** Branch predicates and iterate counts depend on captured variable values that are only
known at runtime. Attempting to pre-compute a DAG would require symbolic execution or over-
approximation. A flat list with runtime control flow evaluation is simpler, faster to plan, and
consistent with how v1 works today.

**Implications:** The Planner's output has no branching structure. The Runtime Core is the sole
authority over which steps actually execute. Adapters receive a `branchResolved` event when a
branch predicate is evaluated, providing visibility without requiring the adapter to understand
the plan structure.

---

## AD-02: Step advancement is always client-driven

**Decision:** The Runtime Core never auto-advances steps. The adapter (CLI, TUI, gert serve client,
test harness) is always responsible for calling `RunHandle.Next()`. The `--auto` flag in the CLI
adapter produces the appearance of auto-advance by calling `Next()` in a loop, but the runtime
contract is unchanged.

**Rationale:** Client-driven advancement is what makes interactive step-through, approval gates,
and evidence collection possible without special cases in the runtime. A single `Next()` contract
covers all modes (interactive, automated, debugger). This is the v1 model and it has proven sound.

**Implications:** Every adapter must implement a step loop. This is a small price for a clean
runtime contract. The `gert serve` adapter's loop is driven by `exec/next` RPC calls from the
VS Code extension.

---

## AD-03: Single-run-per-process for v2.0

**Decision:** The v2.0 runtime hosts at most one run per process. `gert serve` may host multiple
sequential runs within a session but not concurrent runs.

**Rationale:** Governance enforcement (file-level trace writes, subprocess environment filtering,
approval gate state) is dramatically simpler with one run per process. Process isolation provides
a natural crash boundary. The demand for concurrent runs has not been demonstrated in v1 usage.

**Implications:** Operators who need to run multiple runbooks simultaneously must start multiple
`gert exec` processes. Concurrent runs within a single `gert serve` session are an explicit v2.1
concern listed in §09 Open Questions.

---

## AD-04: Denylist takes precedence over allowlist

**Decision:** When evaluating command governance, the denylist is checked before the allowlist.
A command in both lists is denied. The allowlist is only consulted if the denylist check passes.

**Rationale:** Security-by-default: a denylist entry should always be a hard block regardless of
what the allowlist says. If an operator adds `rm` to the denylist, they mean it unconditionally.
Requiring operators to remove `rm` from the allowlist as well would be an easy mistake to miss.

**Implications:** Operators who want to test whether a command is permitted must check the denylist
first. The `GovernanceEngine.CheckCommand()` implementation already encodes this order in v1;
v2 preserves and formalizes it.

---

## AD-05: OPA integration deferred to v2.1; ship structured YAML policy first

**Decision:** v2.0 uses structured YAML `meta.governance` blocks as the sole policy mechanism.
Open Policy Agent (OPA) integration, via a `PolicyEngine` interface with `EvaluatePreStep` and
`EvaluatePreRun` hooks, is targeted for v2.1.

**Rationale:** The v2.0 governance vocabulary is small (six primitives) and closed. A full
policy DSL (Rego) would be over-engineered for this scope. Structured YAML blocks are
self-contained, JSON Schema validatable at authoring time, and backward-compatible with v1.
The `PolicyEngine` interface is designed now so that v2.1 OPA support is a drop-in addition
without breaking the runtime contract.

**Implications:** The `PolicyEngine` interface must be defined as a first-class Go interface
in the Runtime Core package (not just in the design document) before v2.0 ships, even if the
only implementation is the built-in structured evaluator. This ensures v2.1 OPA work has a
clean seam to attach to.

---

## AD-06: Identity is self-asserted in v2.0 via `--as`

**Decision:** Actor identity in v2.0 is the string passed via `--as <identity>`. It is not
authenticated or verified by gert. The `--as` value is recorded in the trace as the actor
identity for all governance events.

**Rationale:** Authentication is the responsibility of the surrounding infrastructure
(CI/CD pipeline credentials, VS Code GitHub Copilot session, MCP server auth). gert is a
governance engine, not an identity provider. Introducing authentication in v2.0 would require
picking an identity protocol (OIDC? SSH cert?) with significant integration cost and no clear
winner across all deployment contexts.

**Implications:** Governed environments must enforce authenticated identity at the adapter layer
(e.g., the `gert serve` process is launched with a verified identity token that populates `--as`).
An `IdentityProvider` interface is defined for v2.1 to allow adapters to supply verified claims.

---

## AD-07: Saga/compensation deferred to v2.1

**Decision:** v2.0 does not implement saga/compensation handlers. Cancellation stops the
current step (SIGTERM + SIGKILL grace period) and writes `run/cancelled` to the trace.
No compensation steps are invoked.

**Rationale:** Compensation requires: (a) a way to declare compensation handlers in the schema,
(b) a mechanism to track which compensation handlers have been "registered" as steps complete,
(c) execution of the compensation chain in reverse order. This is substantial schema and runtime
work. Dennis's research identifies it as an important pattern (Temporal, Argo) but not a v2.0
blocker. The most common v1 workflows have manual cleanup steps, not automated compensation.

**Implications:** The schema and lifecycle model must be designed with compensation in mind
(i.e., the step schema should have an optional `on_cancel` or `compensate` field that is
parsed but not executed in v2.0). This avoids a breaking schema change when v2.1 adds support.

---

## AD-08: `gert serve` is an adapter in the API/Adapter Layer

**Decision:** `gert serve` is formally classified as an adapter implementing the `Adapter` interface.
It is not a special mode of the runtime or a peer component; it is a consumer of `RunHandle` and
the event channel, like TUI and any future web adapter.

**Rationale:** In v1, the coupling between `ext/serve` and `pkg/engine` was a major pain point
(identified in gap analysis). The serve package imported internal engine types directly. Classifying
`gert serve` as an adapter enforces the same boundary rules as TUI: no execution logic, only event
rendering and input forwarding.

**Implications:** The `serve` package must not import `pkg/engine` internal types. It may only
import the public `RunHandle`, `Event`, `StepResult`, and `ApprovalDecision` types defined in the
API/Adapter Layer boundary package. Brian (Go) should enforce this with a `go/analysis` import
restriction check in CI.

---

## AD-09: Trace events have monotonic sequence numbers

**Decision:** Every trace event in the append-only JSONL trace file has a `seq` field containing
a monotonically increasing integer, starting from 1 for each run.

**Rationale:** Sequence numbers enable consumers to detect gaps (indicating corruption or tampering),
establish total ordering without relying on wall-clock timestamps, and support efficient incremental
consumption (a reader can resume from a known sequence number). This is a minor addition to the
trace format with significant auditability benefits.

**Implications:** The `TraceWriter` must maintain an internal sequence counter protected by a mutex.
Sequence numbers must not be reused, even across crash-recovery resume scenarios (resume continues
from last known sequence).

# Decision: Iterate Visual Redesign — Test Contract

**Date:** 2026-06-XX  
**By:** Knov (Playwright & E2E Testing Specialist)  
**Task:** Write Playwright tests for iterate block visual redesign

---

## Decision: CSS Class Names as the Test Contract

Tests assert on `.wf-iterate-container`, `.wf-fork-diamond`, `.wf-join-diamond` in the SVG DOM. These are the classes Kurapika must add to the graph renderer. The tests **intentionally fail until those classes are shipped** — they are the acceptance criteria, not post-hoc coverage.

**Why this matters:** Any implementation that passes these tests is correct from the testing perspective. If Kurapika uses different class names, the tests should be updated at the same time.

---

## Decision: Secondary Runtime Attributes Use Soft Assertions

`data-iterate-pass` and `data-iterate-lane` (runtime expansion attributes) are checked with `console.warn`, not `expect()`. Reason: runtime expansion is a separate milestone and blocking the static tests on unimplemented runtime features would cause false failures in CI before the feature lands.

Once Kurapika ships runtime expansion, the soft checks should be promoted to hard `expect()` assertions.

---

## Decision: No Scenario Injection in Runtime Tests

The web app does not expose `scenarioDir` to the web UI (the `exec/start` call in `runbookRunner.ts` hardcodes `mode: 'real'`). Runtime tests therefore run against the real gert engine with real tool invocations.

**Recommendation:** Kurapika or Killua should add scenario/replay support to the web runner to enable deterministic runtime testing. Until then, runtime tests depend on tool availability and may flake in CI environments without the `curl` gert tool registered.

**Tracking:** Runtime tests have `test.setTimeout(120000)` and graceful fallbacks for failed invocations (iterate steps have `continue_on_fail: true`).

---

## Files Delivered

| File | Tests | Purpose |
|------|-------|---------|
| `web/tests/specs/iterate-sequential.spec.ts` | 4 | Sequential iterate: container rect, no back-edge, header text, runtime passes |
| `web/tests/specs/iterate-parallel.spec.ts` | 5 | Parallel iterate: fork diamond, join diamond, symmetric count, no wrong class, runtime columns |

**Screenshots produced:**
- `web/tests/screenshots/iterate-sequential-static.png`
- `web/tests/screenshots/iterate-sequential-runtime.png`
- `web/tests/screenshots/iterate-parallel-static.png`
- `web/tests/screenshots/iterate-parallel-runtime.png`

# Phase 1 Shared Renderer Verification — Knov Report

**Date:** 2026-04-07  
**Verified by:** Knov (Playwright & E2E Testing Specialist)  
**Commit:** `60ae641`

---

## Overall Verdict: ✅ PASS (with one bug found and fixed)

---

## Step-by-Step Results

### Step 1: Build State

| Target | Result |
|--------|--------|
| `web` build (`tsc && vite build`) | ✅ PASS (after tsconfig fix) |
| `vscode` build (`esbuild`) | ✅ PASS — 1.1MB clean bundle |

**Bug Found:** `web/tsconfig.json` included `../shared/renderer` but not excluding `*.test.ts` files. The `treeToGraph.test.ts` (which uses Jest globals) was being type-checked by the web's tsconfig, failing with `Cannot find name 'describe'` etc.

**Fix Applied:** Added to `web/tsconfig.json`:
```json
"exclude": ["../shared/renderer/**/*.test.ts", "../shared/renderer/**/*.spec.ts"]
```

---

### Step 2: Existing Playwright Tests

**Result: 13 passed → 17 passed (4 new tests added) — no regressions**

Same 4 pre-existing failures remain (confirmed pre-existing by checking before/after):
- `knov-edge-color-verify.spec.ts` — DNS test (requires live DNS infrastructure)
- `knov-screenshot.spec.ts` — screenshot capture (pre-existing)
- `tool-catalog.spec.ts` (×2) — tool list timeout (pre-existing gert server config issue)

---

### Step 3: Verification Spec

**Result: 4/4 PASS** — `web/tests/specs/knov-phase1-verify.spec.ts`

| Test | Result |
|------|--------|
| graph renders SVG nodes from shared renderer | ✅ PASS |
| prune toggle button exists in toolbar | ✅ PASS |
| deleted files confirmed gone from web/src/shared/ | ✅ PASS |
| shared/renderer has zero vscode imports | ✅ PASS |

**Screenshot:** `web/tests/screenshots/phase1-graph.png`

---

### Step 5: treeToGraph Unit Tests

**Result: 84 tests PASS** (37 from `vscode/src/views/treeToGraph.test.ts` + 47 from `shared/renderer/graph/treeToGraph.test.ts`)

Command: `cd vscode && npx jest --testPathPattern="treeToGraph" --no-coverage`

---

### Step 6: Duplicate File / Platform Import Checks

| Check | Result |
|-------|--------|
| `web/src/shared/renderGraph.ts` absent | ✅ PASS — file is gone |
| `web/src/shared/treeToGraph.ts` absent | ✅ PASS — file is gone |
| `web/src/shared/` only has: `helpers.ts`, `snapshotStateMachine.ts`, `themes/`, `treeOps.ts` | ✅ PASS |
| `grep -r "from 'vscode'" shared/renderer/` | ✅ PASS — CLEAN |
| `grep -r "from '.*web/" shared/` | ✅ PASS — no web imports |

---

## Summary

Phase 1 is verified. The shared renderer:
- Builds cleanly in both web and vscode
- Renders the graph correctly in the web app (SVG with `wf-node`/`ed-node` classes)
- Exposes the prune toggle in the toolbar
- Has zero platform-specific imports
- Has no duplicates left in `web/src/shared/`
- 84 unit tests all pass

**One tsconfig bug was found and fixed** as part of this verification. The fix is clean and precise: excluding test files from the web TypeScript compilation.

# Decision: Iterate Block Visual Language Split

**Date:** 2026-06-26  
**By:** Kurapika (Frontend Engineer)  
**Status:** Implemented

## Decision

Sequential and parallel `iterate` blocks are now rendered with distinct visual languages.

### Sequential iterate (no `concurrency` or `concurrency: 1`)
- **Visual:** n8n-style container rect wrapping body steps
- **Node type:** still `'iterate'`, distinguished by `data._iterateType = 'sequential'`
- **Header label:** `↻ for each {as} in {over}` — clearly communicates the loop variable
- **No back-edge** — the container rect IS the visual loop indicator
- **Container geometry** stored as `data._containerHeight` / `data._containerWidth` for the renderer

### Parallel iterate (`concurrency > 1`)
- **Visual:** Fork diamond `◇ ×N` → body column → Join diamond `◇ join`
- **Node types:** `'fork-diamond'` and `'join-diamond'` (new types added to `GraphNode.type` union)
- **No container rect** — the fork/join chrome frames the body
- **No back-edge**

## Why

The old rendering (dashed-border header node + back-edge bezier) gave no visual signal about whether iteration was sequential or parallel. The two paradigms have fundamentally different execution semantics (one item at a time vs N concurrent), so the static graph should communicate this immediately.

## Type changes

Added to `TreeNode.iterate`: `over?: string`, `concurrency?: number`, `collect?: Record<string, string>`  
Added to `GraphNode.type`: `'fork-diamond'`, `'join-diamond'`

## Backward compatibility

- `expandedIterate` tree nodes still use the original `layoutExpandedIterate()` path (unchanged)
- The `iteratePassStripH` runtime expansion in `renderExecutionGraph` still works (targets `type === 'iterate'` nodes which sequential containers are)
- Pre-existing iterate tests updated: back-edge assertion replaced with container/no-back-edge assertion; parallel test added

# Decision: Bibliography System for Gert v2 Design Document

**Date:** 2026-04-18  
**Author:** Leslie (LaTeX Specialist)  
**Status:** Implemented  

## Context

The gert v2 design document had NO bibliography infrastructure. All references to academic papers, industry systems, standards, and specifications were mentioned inline as text (e.g., "IBM Redbook", "Temporal", "OpenTelemetry Specification") with no proper citations or bibliography chapter.

This was a significant documentation gap:
- No way to look up full citations
- No DOIs or URLs for referenced papers
- Difficult to audit which claims are backed by research
- Fails academic/professional documentation standards

## Decision

Implemented a complete bibliography system using **biblatex + biber** (not natbib).

### Why biblatex + biber?

**Chosen:** biblatex with biber backend, numeric citation style, sorted by name/year/title

**Rejected:** natbib + bibtex (older, less flexible)

**Rationale:**
- biblatex is the modern LaTeX bibliography system (actively maintained)
- Biber backend provides better Unicode support, better sorting, more flexibility
- Numeric citation style fits technical documentation style
- The MastersThesis class is compatible with biblatex
- pdflatex + biber workflow is standard on modern TeX installations

### Implementation

1. **Created references.bib** with 40+ entries covering:
   - Academic papers (IBM Redbook, IJSRET, DBSec 2024)
   - Standards (OpenTelemetry, ISO 27001, SOC 2, NIST SP 800-53, RFC 3339)
   - Specifications (JSON Schema, OpenAPI, W3C Trace Context, ITIL v4)
   - Workflow systems (Temporal, Argo, Prefect, Airflow, Step Functions)
   - Runbook tools (AWS SSM, PagerDuty, Rundeck, StackStorm)
   - Policy frameworks (OPA, Cedar, Sentinel)
   - Cloud-native tools (Kubernetes, Gatekeeper, Jaeger, Zipkin)
   - Development tools (Testify, Bubble Tea, JSON-RPC, VS Code API, MCP)

2. **Updated main.tex:**
   - Added `\usepackage[backend=biber,style=numeric,sorting=nyt]{biblatex}`
   - Added `\addbibresource{references.bib}`
   - Added `\backmatter` and `\printbibliography[heading=bibintoc,title={References}]` before `\end{document}`

3. **Added \cite{} commands** throughout sections:
   - §00 Overview: 9 citations
   - §01 Goals and Non-Goals: 12 citations
   - §03 Schema: 3 citations
   - §05 Tool Runtime: 2 citations
   - §08 Testing: 1 citation
   - §09 Open Questions: 4 citations
   - Total: 26 unique citations (out of 40+ available entries)

4. **Compilation workflow:**
   ```
   pdflatex main.tex    # First pass
   biber main           # Process bibliography
   pdflatex main.tex    # Second pass (resolve citations)
   pdflatex main.tex    # Final pass (resolve cross-refs)
   ```

## Outcomes

✅ **All 26 citations resolved successfully** (no undefined citation warnings)  
✅ **Bibliography chapter added** with proper formatting  
✅ **Page count increased from 147 to 155** (8 pages for References)  
✅ **References appear in table of contents**  
✅ **Clean compilation** (no critical errors)  

## Citation Style Examples

- Single: `\cite{temporal-docs}`
- Multiple: `\cite{soc2-aicpa,iso27001,hipaa-security-rule}`
- In text: "...industry standard~\cite{otel-spec}..."

## Future Work

### Additional citations to add:
- §02 Architecture (currently no citations)
- §04 Extension Runtime (currently no citations)
- §11 Governance and Policy (policy-as-code examples)
- §12 Evidence/Tracing (event sourcing, PROV model)
- §14 Input Provider Framework (provider patterns)

### BibTeX entries available but not yet cited:
- google-sre-book, google-sre-workbook (SRE best practices)
- w3c-prov (provenance model)
- dbsec2024-playbook-generation (playbook optimization)
- runops-engineering, runbooks-as-code-tutorial
- airflow-docs, rundeck-docs, stackstorm-docs
- sentinel-docs

These can be added as sections are expanded with more context.

## Files Modified

- `/Volumes/Projects/gert/design/gert-v2/references.bib` (created)
- `/Volumes/Projects/gert/design/gert-v2/main.tex` (updated)
- `/Volumes/Projects/gert/design/gert-v2/sections/00-overview.tex` (citations added)
- `/Volumes/Projects/gert/design/gert-v2/sections/01-goals-and-nongoals.tex` (citations added)
- `/Volumes/Projects/gert/design/gert-v2/sections/03-schema-vnext.tex` (citations added)
- `/Volumes/Projects/gert/design/gert-v2/sections/05-tool-runtime.tex` (citations added)
- `/Volumes/Projects/gert/design/gert-v2/sections/08-testing-and-acceptance.tex` (citations added)
- `/Volumes/Projects/gert/design/gert-v2/sections/09-open-questions.tex` (citations added)

## Recommendation

This bibliography system is **production-ready** and should be maintained as the design document evolves:

1. **When adding new references:** Add BibTeX entry to references.bib with proper metadata
2. **When citing:** Use `\cite{key}` not inline text descriptions
3. **Cite keys:** Use descriptive names (e.g., `temporal-docs`, `iso27001`, `otel-spec`)
4. **Compile:** Always run full pdflatex + biber + pdflatex + pdflatex cycle
5. **Verify:** Check main.log for undefined citations before committing

## Cross-references

- Dennis's research brief: `/Volumes/Projects/gert/.squad/tmp/dennis-research-brief.md` (primary source for references)
- Bibliography file: `design/gert-v2/references.bib`
- Leslie's history: `.squad/agents/leslie/history.md`

# Decision Record: Document Structure Preparation for v2 Design Spec

**Date:** 2026-04-18  
**Author:** Leslie (LaTeX Specialist)  
**Audience:** gert Team, specifically Ken (Architecture), Dennis (Research), Brian (Go Implementation)

---

## Context

Ken's gap analysis identified **six entirely missing sections** critical to a buildable v2 design specification. The current document (9 sections) is a "table of contents with design intent" but lacks buildable specifications for migration, governance, tracing, adapter contracts, input providers, and observability.

Ken's recommendations: These six sections must be added before any implementation begins, as they are either launch blockers (migration) or core to the design (governance, events, adapters).

---

## Decision

**Created six new sections (§10–§15) with stub structure.**

### Sections Created

| § | Title | Rationale |
|---|-------|-----------|
| 10 | Migration and Compatibility | Launch blocker: operators need to know what breaks, what migrates automatically, what requires manual work |
| 11 | Governance and Policy | Governance is gert's core differentiator (allowlists, denylists, env blocking, redaction, approval gates); v1 design entirely absent from v2 |
| 12 | Evidence, Tracing, and Resumption | Append-only trace format, state snapshots, SHA256 capture, run resumption; load-bearing operational feature missing from current design |
| 13 | Adapter Contracts | Explicit interface (event subscription, handshake, render lifecycle) required before TUI, Web, VS Code adapter work begins |
| 14 | Input Provider Framework | Input resolution framework; carries forward v1 `.provider.yaml` model (or explicitly redesigns it) |
| 15 | Observability and Diagnostics | Structured logging, metrics, OpenTelemetry; makes §07 hardening claims implementable |

### Stub Structure

Each section follows a consistent template:
- `\chapter{Title}` — matching Ken's proposal title exactly
- **2-3 sentence description** — drawn from Ken's section proposal, emphasizing scope and criticality
- `\section{TODO}` with placeholder text `This section is under active authoring.`

This provides clear visual markers for authoring while maintaining compilability and table-of-contents visibility.

### Compilation and Audits

- **main.tex updated:** Added `\input{sections/10-migration-compatibility}` through `\input{sections/15-observability-diagnostics}` after §09
- **Compilation verified:** pdflatex succeeded with 18-page output (172 KB), no critical LaTeX errors
- **Existing sections audited:** All §00–§09 confirmed LaTeX-clean (no unclosed environments, no escaping errors)

---

## Rationale

### Why six sections, not more/fewer?

Ken's gap analysis identified **10 entirely missing topics** but prioritized only 6 for immediate authoring (the other 4 map to subsections within existing chapters). These 6 address the most critical gaps that block implementation and are most likely to require dedicated sections for clarity.

### Why stubs, not blank files?

Stubs serve three purposes:
1. **Compilability:** Document can compile and be distributed; sections are visible in TOC
2. **Visibility:** Authors know exactly where to write (clear sections to fill) and what topics are in-scope (from description)
3. **Reviewability:** Ken and ormasoftchile can review authoring assignments and priorities before writing begins

### Why integrate into main.tex now vs. later?

Integrating now ensures:
- Document compiles cleanly as a whole from this point forward
- TOC shows full structure immediately (helps with editorial prioritization)
- No risk of sections being forgotten in the final integration phase
- Adapters (Brian's Go code, John's schema) can reference section numbers stable

---

## Next Steps (For Team)

1. **Assign authoring:** Each section should be assigned to a single author (likely Dennis for research-heavy §10, §12, §15; Ken for architecture-driven §13, §14; coordinated team effort for §11)
2. **Define author order:** Suggest authoring order: §10 (migration), §11 (governance), §13 (adapter contracts) first, as they unblock other work
3. **Link to design decisions:** As authors write, maintain cross-references to `.squad/decisions.md` 
4. **Expand existing sections:** Remember Ken's critical gaps in §00–§09 also require 3–5× expansion

---

## Decision Metadata

- **Status:** Accepted
- **Impact:** Document structure now ready for full writing sprint
- **Risk:** None (stubs are non-breaking)
- **Reversibility:** High (stubs can be removed or reorganized without affecting compilation)
