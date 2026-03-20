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

## Governance

- All meaningful changes require team consensus
- Document architectural decisions here
- Keep history focused on work, decisions focused on direction
