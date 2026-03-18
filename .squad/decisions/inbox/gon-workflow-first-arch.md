# Architectural Assessment: Workflow-First Visual Paradigm

**Date:** 2026-03-18  
**Author:** Gon (Lead Architect)  
**Status:** Proposed  
**Requested by:** ormasoftchile  

---

## Background

The team has shipped a tree-view editor (RunbookEditorPanel) and a tree-view execution viewer (RunbookPanel). The user feedback is that a visual tool should show _workflow_ — nodes and directed edges with condition labels and execution path annotation — not a document structure rendered as a collapsible list. The reference visual is n8n / Azure Logic Apps Designer.

This assessment covers what needs to change architecturally, what the minimum proof-of-concept is, and how to sequence the work.

---

## 1. Execution Viewer (RunbookPanel) — What Needs to Change

### Current state

`exec/start` returns `{ tree: [] }` via `resolveTreeForDisplay` in `ext/serve/pkg/serve/serve.go` (line ~2264). This already includes:
- Step `id`, `type`, `title`, `when` (guard condition), `outcomes`
- Iterate blocks with `mode`, `over`, `as`, `total`, `max`, `until`
- Branch arrays with `condition`, `label`, and nested `steps`

The extension holds this in `RunbookPanel.tree: any[]` and renders it with `renderTree()` in `runbookPanel.ts` (line ~2342) — a recursive HTML renderer that produces **indented list items**, not a graph.

Live state tracking works: `event/stepStarted`, `event/stepCompleted`, `event/stepSkipped` carry `stepId` + `status`, which populates `RunbookPanel.stepStates: Map<string, string>`. The tree nodes are colored by looking up the stepId in that map.

### What is missing for a workflow graph

1. **Topology (nodes + edges).** The tree is hierarchical but edges are implicit — "B follows A because B is next in the array." A graph renderer needs explicit `{ from, to, label }` edges.

2. **Branch-taken annotation.** During execution, when a step has `branches`, the engine evaluates each condition and queues matching branch steps. Currently no event announces which branch was taken. The UI can _infer_ this (if a branch's first step fires `stepStarted`, that branch was taken), but an explicit event is cleaner for multi-branch and nested cases.

3. **Iterate pass state.** `event/iterateStarted` is emitted (line ~419 in runbookPanel.ts references it) but the graph needs to visually distinguish "this loop node is on pass 2/5" vs "this loop is done / converged / failed."

### Is this additive or a new data shape?

**Additive.** The tree data already carries all topology information needed to derive a graph. No new data shape is required for static topology — the transformation is purely client-side.

For _execution path annotation_, two small additive events to `ext/serve/pkg/serve/serve.go` are sufficient:

```
event/branchResolved   { parentStepId: string, branchIndex: number, condition: string, taken: boolean }
event/iteratePassEnd   { iterateStepId: string, pass: number, max: number, converged: boolean }
```

`event/branchResolved` fires once per branch condition evaluation in `handleTreeNext`. `event/iteratePassEnd` fires at the end of each iterate watchpoint pop (already has the data, just needs serialization). Both are zero-cost if the UI is not listening.

### Minimum change to `gert serve`

Two additions to `ext/serve/pkg/serve/serve.go`:
1. Emit `event/branchResolved` when branching is evaluated in `handleTreeNext`  
2. Emit `event/iteratePassEnd` from the `iterateWatchpoint` pop handler  

No changes to `exec/start` result shape, no changes to the engine, no new methods. These are 10–20 line additions.

### Client changes for execution viewer graph

**New file: `vscode/src/views/treeToGraph.ts`**
```ts
interface GraphNode { id: string; type: string; title: string; isIterate?: boolean; isBranch?: boolean }
interface GraphEdge { from: string; to: string; label?: string; edgeType: 'seq' | 'branch' | 'iterate-body' | 'iterate-back' }
function treeToGraph(tree: any[]): { nodes: GraphNode[]; edges: GraphEdge[] }
```

Pure function. No dependencies on the serve layer. Shared by both panels.

**`vscode/src/views/runbookPanel.ts`**
- Replace `renderTree()` HTML renderer with a graph canvas (recommendation: Cytoscape.js or a lightweight SVG layout using dagre-d3)
- `stepStates` map already works — node color = look up state by id
- Listen for `event/branchResolved` and mark edge `{ from, to }` as "taken" or "not taken"
- No change to `ChainEntry.tree: any[]` — it keeps working as before

---

## 2. Authoring Editor (RunbookEditorPanel) — What Needs to Change

### Current state

`RunbookEditorPanel` (file: `vscode/src/views/runbookEditorPanel.ts`) parses YAML → `runbook.tree: TreeNode[]` and renders a right-side tree using `renderTree()` (line ~920). A left-side form panel edits individual nodes.

**Critical existing flaw:** `rebuildTree()` (referenced in Kurapika's MVP doc) serializes back to a _flattened_ linear tree. Branches and iterate blocks are discarded. This is a known architectural debt from Phase 1 that is a release blocker regardless of the workflow-first direction.

### Mapping graph → YAML

A workflow graph can express exactly what the YAML tree expresses:
- Sequential edges → linear `tree` array order
- Conditional edges → `branches` array on a parent step node
- Loop edges → `iterate` block (back-edge = "run again if condition not met")

The internal data model does **not need to change**. The `TreeNode[]` structure is the graph — it just needs to be _rendered as one_.

```
YAML tree → (parse) → TreeNode[] ← (treeToGraph) → { nodes, edges }
                     ↑                                      ↓
                     (rebuild from graph edits) ← (user edits in canvas)
```

Round-trip fidelity survives _if and only if_ the graph editor maintains the `TreeNode[]` as the source of truth and does not introduce a separate edge list. Graph rendering is purely a view — mutations must write back to the existing tree model.

### Does YAML round-trip fidelity survive?

Yes, with this constraint: **the graph authoring surface must not have a separate edge data model**. Edges are derived from tree position. "Connect A to B" = "place B after A in the tree array." "Add branch from A" = "add a `Branch` to `A.branches`." These are tree mutations, not edge list mutations.

This means standard graph editor frameworks that store explicit edge lists (React Flow, etc.) need an adapter that converts their edge definition back to tree mutations on every save. This is feasible but requires careful validation that no edge topology is representable in the graph UI that cannot be expressed in the YAML tree model (e.g., diamond merges, cycles outside iterate — both are unsupported and must be blocked in the UI).

### Minimum viable graph authoring surface

1. **Fix `rebuildTree()`** to preserve branches and iterate blocks — this is a pre-requisite for _anything_ beyond Phase 1 and is unrelated to graph rendering.
2. Replace the right-side `renderTree()` HTML with an SVG graph using `treeToGraph(tree)` output.  
3. Click on a graph node → populate the left-side form (same as clicking a tree node today — same `selectedNodePath` mechanism works).
4. "Add step after selected" / "Add branch from selected" = existing form operations, just navigated from the graph.
5. **Defer**: drag-to-connect, arbitrary topology editing, undo/redo. These are Phase 3.

---

## 3. Shared Concerns

### Tree→Graph: server-side vs client-side

**Decision: client-side.**

Rationale:
- The tree is already transmitted in `exec/start`. No additional bytes need to flow.
- The `pkg/diagram` package (`pkg/diagram/diagram.go`) already fully implements tree→Mermaid graph generation in Go, proving the transformation is well-understood and bounded. The TypeScript mirror of this logic is ~100 lines.
- Server-side adds a new `runbook/graph` RPC method, a new Go struct for edges, schema versioning risk, and a serve coupling between UI topology needs and the engine.
- The serve layer should emit _execution path events_ (additive, minimal), not graph topology — that is a view concern.

**Risk surface for client-side:**
- The `treeToGraph` function must stay in sync with schema changes to `TreeNode`, `Branch`, `IterateBlock`. Risk is low — schema is stable and changes go through validation.
- Mitigation: `treeToGraph` is a pure function with narrowly typed inputs. A unit test suite covering branch/iterate/nested cases keeps it correct.

**Risk surface for server-side:**
- Schema evolution in Go types propagates to a graph API — creates a second surface to maintain.
- Serve layer becomes a view-coupling point; changes to how the graph looks require a Go rebuild and release cycle.

### Shared file between panels

`vscode/src/views/treeToGraph.ts` — referenced by both `runbookPanel.ts` and `runbookEditorPanel.ts`. This is the natural factoring point. It should be pure, no VSCode API dependencies.

---

## 4. Sequencing Recommendation

### Phase A: Execution Viewer first (RunbookPanel)

**Why first:**
- Read-only display change — zero YAML round-trip risk
- Much tighter feedback loop: run a real runbook, see nodes light up in the correct execution order
- Validates that `treeToGraph` correctly models the schema, before using it for authoring mutations
- The `mapWidth` / left-tab / `renderTree` code in RunbookPanel is already isolated and well-understood

**Minimum proof-of-concept (2–3 days of work):**

1. Implement `treeToGraph(tree: any[]): { nodes, edges }` in `treeToGraph.ts` (the diagram.go port).
2. Embed a lightweight graph renderer in the RunbookPanel webview. Cytoscape.js is already available as a CDN include. Dagre layout gives automatic top-to-bottom flow placement.
3. Wire `stepStates` map to node color (pending = gray, running = blue, passed = green, failed = red, skipped = faint).
4. This replaces _only_ `renderTree()` in `runbookPanel.ts` — nothing else in the panel changes.

**Go changes (optional for PoC, useful for Phase A GA):**
- Add `event/branchResolved` emission in `handleTreeNext` in `ext/serve/pkg/serve/serve.go` — annotates which branch edges are "active" vs "not taken."

### Phase B: Authoring Editor (RunbookEditorPanel)

Preconditions from Phase A:
- `treeToGraph.ts` is proven correct against real runbooks
- Graph library choice is settled

**Work items:**
1. Fix `rebuildTree()` — must preserve branches and iterate. This is blocking regardless.
2. Swap `renderTree()` in RunbookEditorPanel for the same graph canvas.
3. Wire graph node click to `selectedNodePath` (existing mechanism).
4. Validate that "add branch" / "add step" operations write to `TreeNode[]` correctly, and that `treeToGraph` reflects the mutation immediately.
5. Test round-trip: edit via graph → save → re-open → graph matches YAML.

---

## 5. Specific Files and Interfaces Summary

| File | Change | Phase |
|---|---|---|
| `vscode/src/views/treeToGraph.ts` | **New** — `treeToGraph(tree) → { nodes, edges }` | A |
| `vscode/src/views/runbookPanel.ts` | Replace `renderTree()` with graph canvas in webview HTML | A |
| `vscode/src/serve/client.ts` | Add `event/branchResolved` and `event/iteratePassEnd` to handled event types | A |
| `ext/serve/pkg/serve/serve.go` | Emit `event/branchResolved` / `event/iteratePassEnd` in tree cursor handlers | A (optional) |
| `vscode/src/views/runbookEditorPanel.ts` | Fix `rebuildTree()`, replace `renderTree()` with graph canvas | B |
| `pkg/diagram/diagram.go` | No change — reference implementation for `treeToGraph.ts` | — |
| `pkg/schema/schema.go` | No change — `TreeNode`, `Branch`, `IterateBlock` are stable | — |

---

## 6. Risk Register

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Graph library layout misbehaves on complex nested branches/iterates | Medium | Medium | Prototype with longest real runbook before committing library choice |
| `treeToGraph` drifts from schema changes | Low | High | Unit tests covering all node types; runs in CI |
| `rebuildTree()` bug causes data loss in editor | High (known) | Critical | Fix before exposing graph authoring; Hisoka round-trip gate applies |
| Branch editor creates topologies not representable in YAML tree | Low | High | Restrict authoring operations to: add-sequential, add-branch, add-iterate; no free-connect |
| Graph canvas performance on large runbooks (50+ nodes) | Low | Low | Virtualize if needed; most runbooks are <20 steps |

---

## Decisions Proposed

1. **Tree→graph transformation is client-side** — `treeToGraph.ts` in the extension, not a new serve API.
2. **Execution viewer is tackled first** — lower risk, faster validation of graph rendering.
3. **`rebuildTree()` fix is a pre-requisite for Phase B** — not optional, treat as P0 before any graph authoring work.
4. **Graph authoring does not introduce an edge list data model** — all mutations write back to `TreeNode[]`; edges are derived view-only.
5. **Two additive serve events** (`event/branchResolved`, `event/iteratePassEnd`) improve execution path annotation but are not blockers for the execution viewer PoC.
