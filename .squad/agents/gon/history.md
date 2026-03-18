# Project Context

- **Owner:** ormasoftchile
- **Project:** gert runbook system and tooling
- **Stack:** Go, TypeScript, C#, YAML, Azure
- **Created:** 2026-03-17

## Learnings

- Team seeded to explore a visual runbook editor for business users.
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