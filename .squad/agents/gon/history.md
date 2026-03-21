# Project Context

- **Owner:** ormasoftchile
- **Project:** gert runbook system and tooling
- **Stack:** Go, TypeScript, C#, YAML, Azure
- **Created:** 2026-03-17

## Learnings

- Team seeded to explore a visual runbook editor for business users.
- (2026-03-19) Deliverable #9: Refactored `TextVizAdapter.renderWorkflow()` to delegate to `renderTreeAsText()` + `extractConnectors()` when `VizOptions.tree` is provided. Graph-based BFS fallback retained for backward compat. SVG adapter untouched.
- Existing VS Code extension already has a mature run webview surface and JSON-RPC client, making a side-by-side visual editor in the same extension the fastest MVP path.
- Extensibility pressure points are dynamic tool/provider resolution and project-aware package references, so the editor model should be AST + schema metadata, not static forms.
- Recommendation drafted: hybrid visual editor with YAML source of truth, schema-driven inspector, and plugin-like step cards resolved from tool/provider definitions.
- Cross-agent alignment: MVP should be phased as contracts-first (Killua), form-first UX (Kurapika), governance-visible Azure/tool abstraction (Leorio), and QA release gates on round-trip + v0/v1 parity (Hisoka).

## 2026-03-17 Design Direction Session

### Problem Identified
- Kurapika's Phase 1 MVP flattens branch/conditional/iterate structures in tree model, causing data loss when editing non-trivial runbooks.
- Root cause: UX designed after implementation started, not before.
- Critical insight: form-first is correct, but **form must synchronize with a visible hierarchical tree model**, not flatten it.

### Design Solution: Hybrid Form-First + Tree Map
- **Left panel:** Step-by-step form editor (Runbook Home, Step Detail, Step Navigator)
- **Right panel:** Synchronized read-only/navigation tree showing full structure (branches, iterates, conditionals with indentation)
- **Interactions:** Click tree node → jump to form; form edits reflect in tree in real time
- **Key win:** Preserves data fidelity while keeping form-first UX for non-technical users

### MVP Scope Clarified
- **In:** Form authoring (seq steps, branches, iterate), tree visualization, governance visibility, valid YAML serialization
- **Out:** Tree-based drag/drop editing, governance policy editing, tool catalog browser, undo/history, templates
- **Rationale:** Business users need structure visibility; engineers need YAML transparency. Tree map is the "check" on form operations.

### UX Principles Established
1. Structure visibility over simplicity — always show tree, never flatten
2. Form as progress, not endpoint — governance/validation at save gate
3. Respect YAML as truth — form output must be round-trippable and pass existing validation

### Key Design Decisions Rationalized
- **Form-first vs. tree-first:** Form-first + synchronized tree (rejects pure tree editor as error-prone for common case)
- **Flat vs. hierarchical:** Hierarchical with indentation; form presents one level at a time (solves Phase 1 flattening bug)
- **Edit-in-place vs. modal:** Non-modal form on right side keeps structure visible
- **Read-only vs. editable governance:** Read-only badges + governance panel Phase 1 (policy is runtime/admin concern, not editor concern)

### Next Work
- Kurapika: align on layout, tree rendering tech (SVG/canvas), interaction details
- Killua: confirm schema introspection for form generation
- Leorio: confirm governance metadata accessibility for UI display
- Hisoka: define round-trip test harness (form → YAML → form → YAML idempotent)
- Gon (next): prototype lo-fi mockup to validate layout before webview rebuild

## Learnings

### 2026-03-18 Workflow-First Visual Paradigm Assessment

- **The tree IS the graph.** `TreeNode { step, branches, iterate }` maps to graph topology without a new data model. Edges are sequential (array order), conditional (branches), and loop (iterate back-edge). A separate edge list is not needed and would create YAML round-trip risk.

- **Tree→graph transformation belongs client-side.** The full topology is already sent to the extension in `exec/start → tree`. The `pkg/diagram/diagram.go` Mermaid generator is the reference implementation to port to `treeToGraph.ts`. Server-side graph API would couple view topology to the serve release cycle.

- **Go serve changes are additive and small.** Two events: `event/branchResolved` (which branch was taken) and `event/iteratePassEnd` (loop pass state). Both are ~10-line additions to `ext/serve/pkg/serve/serve.go`. Not blockers for the execution viewer PoC.

- **`rebuildTree()` in RunbookEditorPanel is a P0 bug** before any graph authoring. It flattens branches and iterate blocks, causing data loss. Fix precedes Phase B work.

- **Sequence: execution viewer first.** Read-only → no round-trip risk → validates `treeToGraph` against real runbooks → proves graph library choice. Phase B (authoring) inherits the proven graph renderer.

- **Key constraint for graph authoring:** restrict editing operations to add-sequential / add-branch / add-iterate. No free-connect. Any topology not expressible in the YAML tree model must be blocked in the UI.

- **`pkg/diagram/diagram.go` is a gold-standard reference** — already generates correct Mermaid from the same tree structure, including branches, iterate, and conditions. `treeToGraph.ts` should mirror its traversal logic.

## 2026-03-18 Phase B — Authoring Editor Graph Integration

### Bugs Fixed
- **`getSelectionType()` misidentified nested steps.** The `path.includes('iterate')` check caused steps inside iterate blocks to be typed as 'iterate', breaking add-step and form rendering for nested structures. Replaced with direct node inspection: `node.iterate && !node.step` for iterate, `node.step` for step.
- **`getIterateAtPath()` used fragile path-walking.** Required path to end with `['steps', N]` and walked up — failed for direct iterate TreeNode selection at path `[0]`. Simplified to `getNodeAtPath(path)?.iterate`.
- **`updateIterateAtPath()` same fragile pattern.** Simplified to match.
- **`add-step` handler computed wrong parent path for nested steps.** For a step at `[0, 'branches', 1, 'steps', 0]`, it appended 'steps' to an already-correct steps array path, yielding `[..., 'steps', 'steps']`. Fixed to use `getNodeAtPath` on the sliced parent path directly.

### Graph Visualization (replaces tree renderer)
- Imported `treeToWorkflow` from `./treeToGraph` (Kurapika's module, read-only for me).
- `renderGraphSVG()` replaces `renderTree()` in the right panel. Uses `treeToWorkflow()` for layout, swaps x/y to get top-to-bottom flow, renders inline SVG.
- Node shapes: ellipse (start/end), diamond (condition/branch), dashed rect (iterate), rounded rect (steps). Structural coloring by step type — no execution state.
- `buildStepIdToPathMap()` maps step IDs back to tree paths for click→form wiring. Uses recursive tree walk, decoupled from treeToWorkflow's internal ID generation.
- Graph node clicks → `select-node` message → same `selectedNodePath` mechanism as before. Click SVG background → home.

### Context-Aware Operations
- Step form: added "Add Branch" button (`add-branch` command).
- Home form: added "Add Step" and "Add Iterate Block" buttons.
- `add-branch` handler: calls existing `addBranchAtPath()` on selected step.
- `add-iterate` handler: creates iterate TreeNode as sibling in parent steps array.
- Action button handler simplified: passes `data-action` directly as command (no hardcoded if/else chain).

### Key Decisions
- **Graph is view-only, mutations go through TreeNode[].** Matches my proposal constraint — no separate edge list, no round-trip risk.
- **stepId-based path mapping over object reference or counter sync.** Robust against treeToGraph internal changes. Falls back gracefully for nodes without IDs.
- **Condition node click → parent step.** Branch condition nodes in the graph map to the parent step that owns the branches, not a separate branch entity.

### Files Modified
- `vscode/src/views/runbookEditorPanel.ts` — all Phase B changes

## 2026-03-18 Phase C — Editor Polish & Architecture

### Task 1: Graph Node Click → Form Navigation
- **Root cause found:** Iterate blocks created in-editor had no `.id`, so `buildStepIdToPathMap()` couldn't correlate graph nodes to tree paths. Clicks on iterate nodes silently failed.
- **Fix:** `addIterateAtPath()` and `add-iterate` handler now assign `id: iterate_{timestamp}` on creation. `buildStepIdToPathMap()` assigns stable synthetic IDs to any iterate block lacking one (using path-based key), ensuring all graph nodes are clickable.
- **Verified existing wiring:** The `<script>` block already queries `[data-graph-path]` elements and posts `select-node` messages. The `handleWebviewMessage` handler routes `select-node` → `selectedNodePath` → `renderUI()` → correct form. No changes needed to the message handler or script block.
- **Click behavior:** step node → Step Detail form, branch condition → parent step form, iterate → Iterate Detail form, start/end → Runbook Home.

### Task 2: Host-Adaptive Visualization Architecture
- Created `vscode/src/views/vizAdapter.ts` with:
  - `VizAdapter` interface: `renderWorkflow(tree, state, options) → string`
  - `VizOptions`: mode (authoring/execution), interactive flag, selectedPath
  - `VizHost` type: 'vscode' | 'web' | 'tui'
  - `createVizAdapter(host)` factory function
- `GraphVizAdapter` (vscode/web): wraps `treeToWorkflow()` + SVG rendering. Self-contained implementation that mirrors the editor panel's rendering style without coupling to panel state.
- `TuiVizAdapter`: renders indented text tree with Unicode box-drawing characters. Placeholder — ready for TUI surface implementation.
- `runbookEditorPanel.ts` NOT refactored to use adapter — this is a forward-looking abstraction per spec. Panel still uses its own `renderGraphSVG()` with richer interactive features (selection highlighting, panel-specific path resolution).

### Task 3: YAML Comment/Formatting Preservation on Save
- **Problem:** `YAML.stringify(this.runbook)` discards all comments, reorders keys, and reformats the document.
- **Solution:** Store original YAML source on load (`originalSource`). On save, use `YAML.parseDocument(originalSource)` to get a comment-preserving AST, then apply edited fields with `doc.set(key, doc.createNode(value))`. Keys removed from the runbook object are also removed from the document.
- **Behavior:** Comments in untouched sections are preserved. Field ordering from the original file is maintained for keys that existed. New keys are appended. Falls back to plain `YAML.stringify()` only if no original source is available.
- **Preview diff** also uses the same comment-preserving path.

### Key Decisions
- **Stable iterate IDs over transient counters.** Using `iterate_{timestamp}` ensures IDs survive save/reload cycles and are unique across sessions.
- **vizAdapter is deliberately decoupled from panel state.** The adapter returns bare SVG/text — interactive wiring (click handlers, selection highlights) belongs to the host surface. This keeps the adapter portable.
- **Document-level YAML editing over field-level patching.** `doc.set()` replaces the entire value subtree for changed keys, which is correct because the editor replaces whole sections (tree, meta). Surgical field-level patching would be fragile against structural changes like adding/removing branches.

### Files Modified
- `vscode/src/views/runbookEditorPanel.ts` — iterate ID assignment, buildStepIdToPathMap robustness, comment-preserving save, originalSource tracking
- `vscode/src/views/vizAdapter.ts` — new file, host-adaptive visualization abstraction

## 2026-03-18 Phase D — Chain Visualization & YAML Comment Preservation

### Task 1: Chain Visualization for Invoke Steps (runbookPanel.ts)

**Problem:** The execution viewer's graph only rendered the active chain entry's tree. When runbooks used `invoke` to call child runbooks, there was no visual chain navigation.

**Solution — three additions:**

1. **Chain breadcrumb** above the graph: `parent-tsg › child-tsg › [current]`. Clicking a crumb switches the graph to that chain entry. Renders only when `chainHistory.length > 0`. New state variable `viewingChainIndex` (null = current, number = chain history index).

2. **Enhanced invoke nodes** in the graph: invoke step nodes now show the target runbook name (`↗ dns-check`), use a distinctive blue stroke, and display a "View child ↗" badge when the child has been executed. Badge click navigates to the child's chain entry graph.

3. **Parent minimap** in bottom-left when viewing a child: faded thumbnail SVG of the parent graph with invoke steps highlighted in blue. Click navigates to the parent. Renders via `renderParentMinimap()` using `treeToWorkflow` at ~30% scale.

**Key design decisions:**
- `viewingChainIndex` resets to null on restart and chain-to-runbook transitions
- `renderGraphSvg()` selects tree/states/stepDetails from chain history or current based on `viewingChainIndex`
- `findChildChainIndex()` maps invoke step ID → next chain entry index
- Branch resolutions only applied when viewing current (not chain history)

### Task 2: YAML Comments/Formatting Preservation (runbookEditorPanel.ts)

**Problem:** The editor's save flow used `doc.set(key, doc.createNode(value))` for each top-level key, which created fresh AST nodes and lost all inline comments and formatting.

**Solution — deep merge instead of full replacement:**

1. **Persistent YAML document:** Store `yamlDoc: YAML.Document` on load via `parseDocument()`. Re-parse after each save.

2. **`serializeWithComments()`**: Entry point for both save and preview-diff. Parses `originalSource` into a fresh document, deep-merges the edited JS object into it, handles key additions/removals.

3. **`deepMergeNode(doc, existing, value)`**: Recursive merge that:
   - **Scalars:** If value unchanged, returns original node (preserves comment). If changed, creates new scalar and copies comment.
   - **Maps:** Recurses per key. Adds new keys, removes deleted keys. Preserves ordering for unchanged keys.
   - **Sequences:** Positional merge up to shorter length, splices excess, appends new items.
   - **Type mismatch:** Falls back to `createNode()`.

4. **Fallback:** If round-trip fails, shows warning and falls back to `YAML.stringify()`.

### Learnings

- **Deep merge > full replacement for YAML preservation.** The previous `doc.set(key, doc.createNode(value))` approach loses all inline comments because `createNode()` builds from plain JS objects. Walking the AST in parallel with the JS object preserves everything unchanged.
- **Chain navigation is view-only state.** `viewingChainIndex` doesn't affect execution — it only selects which tree/states to render. This keeps it safe to add without touching the execution flow.
- **Minimap rendering reuses `treeToWorkflow`.** No separate layout pass needed — same function at reduced scale gives a coherent thumbnail.

### Files Modified
- `vscode/src/views/runbookPanel.ts` — chain breadcrumb, chain navigation, invoke node enhancement, parent minimap
- `vscode/src/views/runbookEditorPanel.ts` — deep merge YAML serialization, persistent yamlDoc

## 2026-03-18 Phase D — Graph Click Navigation Fix & Condition Builder

### Task 1: Graph Node Click → Form Navigation Fix

#### Bug Found & Fixed
- **Iterate ID mismatch on first render.** `renderGraphSVG()` called `treeToWorkflow()` before `buildStepIdToPathMap()`. Since `treeToWorkflow` generates ephemeral IDs for iterate nodes without IDs (`iterate-_gN`), and `buildStepIdToPathMap` assigns persistent IDs (`iterate_path_N`), the path map lookup failed for iterate nodes on the first render. Fix: swap call order so `buildStepIdToPathMap` runs first (assigns stable IDs), then `treeToWorkflow` reads them.

#### Click Handler Improvements
- Replaced per-element `querySelectorAll('[data-graph-path]').forEach(addEventListener)` with **event delegation** on `#graph-container`. Single listener uses `e.target.closest('.ed-node[data-graph-path]')` to find the clicked node.
- Background click detection improved: any click not on `.ed-node` or zoom controls navigates to home. Previously relied on `e.target === svgCanvas`, which missed clicks on the `<g id="graph-transform">` wrapper.
- Start/End nodes correctly navigate to Runbook Home (path `[]`).

### Task 2: Structured Branch Condition Builder

#### Implementation
- **Variable dropdown** populated from `getAvailableVariables()` — collects `meta.vars` keys, `meta.inputs` keys, and all `step.capture` keys from the tree.
- **Operator dropdown** with 7 operators: contains, not_contains, equals, not_equals, matches (regex), gt, lt.
- **Value input** for comparison value.
- **Live preview** shows the generated Go template expression.
- **Advanced toggle** switches between visual builder and raw textarea for power users.
- **Condition parsing** on load: regex-based best-effort parse of common Go template patterns (`contains`, `not`, `eq`, `ne`, `regexMatch`, `gt`, `lt`). Handles both `.varname` and `.captures.varname` patterns.
- Unparseable conditions default to Advanced mode with raw textarea visible.
- Toggling back from Advanced mode attempts to re-parse the expression into builder fields.

#### Expression generation patterns
- `contains` → `{{ contains .var "value" }}`
- `not_contains` → `{{ not (contains .var "value") }}`
- `equals` → `{{ eq .var "value" }}`
- `not_equals` → `{{ ne .var "value" }}`
- `matches` → `{{ regexMatch "pattern" .var }}`
- `gt` → `{{ gt .var value }}`
- `lt` → `{{ lt .var value }}`

### Learnings
- **Call order matters for side-effecting ID assignment.** `buildStepIdToPathMap` mutates tree nodes (adds iterate IDs) — it must run before any read-only consumer like `treeToWorkflow`.
- **Event delegation is strictly better for SVG click handling in webviews.** Eliminates issues with dynamic DOM, deeply nested SVG groups, and `pointer-events:none` children.
- **Variable collection as a flat set is sufficient for builder MVP.** Strict ordering (only variables captured before the branch point) is a Phase 2 refinement.

### Files Modified
- `vscode/src/views/runbookEditorPanel.ts` — graph click fix, event delegation, `getAvailableVariables()`, structured branch condition builder

## 2026-03-18 Phase E — Peek YAML, Tool Catalog, Scenario Browser

### Task 1: Peek YAML (Context Menu on Graph Nodes)
- Right-click any graph node in the editor → context menu with "Peek YAML".
- Click → modal overlay shows the YAML fragment for just that node, stringified with `YAML.stringify()`.
- Copy button with clipboard API. Dismiss via click-outside or Escape.
- Implementation: `contextmenu` event handler on `#graph-container` with event delegation. Custom positioned menu div. `peek-yaml` message → host extracts node at path → `peek-yaml-result` message back to webview.

### Task 2: Searchable Tool Catalog Panel
- New file: `vscode/src/views/toolCatalogPanel.ts`.
- Command: `gert.showToolCatalog` registered in `package.json` and `extension.ts`.
- Data source: tries `GertClient.toolsList()` via JSON-RPC first (5s timeout), falls back to scanning `tools/*.tool.yaml` files.
- Search bar filters tools by name/description in real time.
- Each tool rendered as a card: name, version, description, action count.
- Click action header → expands to show args table (name, type, description, required indicator).
- "Insert Step" button generates a YAML snippet for the action and copies to clipboard.

### Task 3: Scenario Browser in Editor
- `discoverScenarios()` scans `scenarios/{runbook-name}/{scenario}/inputs.yaml`, also checks `gert.yaml` `paths.scenarios`.
- `renderScenarioSection()` renders a list of discovered scenarios with name, input summary, and Run button.
- "New Scenario" button prompts for kebab-case name, creates directory + empty `inputs.yaml`, opens it in a side editor.

### Key Decisions
- **Peek YAML uses webview-side modal, not VS Code native quickpick.** Allows syntax-highlighted display and copy button without losing webview context.
- **Tool catalog tries JSON-RPC first with timeout, not conditional.** Avoids needing to detect whether gert serve is running — just try and fall back.
- **Scenario browser is part of Runbook Home, not a separate tab.** Keeps the UI lightweight per spec.

### Files Modified
- `vscode/src/views/runbookEditorPanel.ts` — peek YAML context menu + handler, scenario browser
- `vscode/src/views/toolCatalogPanel.ts` — new file, searchable tool catalog webview panel
- `vscode/src/extension.ts` — registered `gert.showToolCatalog` command
- `vscode/package.json` — added `gert.showToolCatalog` command entry

## 2026-03-20 Bug Fixes — Invoke Boundary Labels & Click→Prose Navigation

### Bug 1: Boundary Container Labels Wrong/Confusing

**Root cause:** The invoke group computation in `renderGraphSvg()` iterated ALL graph nodes with `::` in their IDs. This incorrectly included internal graph artifacts:
- **Condition diamond nodes** — IDs like `cond-invoke_net::check_dns-_g5` (the condition ID inherits `::` from the prefixed step ID it belongs to)
- **Pass-through nodes** — IDs like `pass-cond-invoke_net::...-1-_g6`

These artifact IDs were split on `::` and treated as invoke child groups, creating spurious containers with labels like `cond-invoke_net` or `pass-cond-invoke_net`.

**Fix:**
1. Added `if (/^(cond|pass)-/.test(node.id)) continue;` filter to skip internal graph artifacts from the invoke group loop.
2. Added fallback loop that progressively strips `::` prefixes when looking up `invokeChildren` for the label (mirrors `mergeWalk` in treeOps.ts).

### Bug 2: Click→Prose Navigation Broken

**Root cause (two-part):**

**(a) Prefix stripping discarded navigation context.** The `viewStep` handler stripped `::` prefixes from invoke child IDs: `invoke_net::check_dns` → `check_dns`. The prose tree (from the UNMERGED tree) only contains parent runbook steps. No section matched the stripped child ID → no `section.active` → auto-scroll found nothing → prose stays at the top.

**(b) Auto-scroll timing.** The IIFE ran synchronously before layout was guaranteed in VS Code webviews after a full `html` setter replacement.

**Fix:**
1. `viewStep` handler stores the full raw ID (no `::` stripping).
2. `renderStepAsProse()` checks: exact match, unprefixed base match, AND parent invoke prefix match.
3. `getHtml()` computes `viewingBaseId` (stripped) for right-panel step detail lookup.
4. Chain history lookup also tries stripped IDs.
5. Auto-scroll wrapped in `requestAnimationFrame`.

### Verification
- `npm run compile` — clean build
- 198/198 views tests pass

### Files Modified
- `vscode/src/views/runbookPanel.ts` — all fixes