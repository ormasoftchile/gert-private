# Kurapika: Workflow Graph View POC Implementation

**Date:** 2026-03-18  
**Status:** ✅ Implemented  
**Scope:** ReadOnly execution viewer (RunbookPanel)  

---

## Summary

Implemented a toggleable **workflow graph visualization** for the gert runbook execution viewer. Users can now switch between the existing tree view and a new SVG-based graph view that renders the execution workflow as a directed acyclic graph (DAG).

## Changes Made

### 1. **New File: `vscode/src/views/treeToGraph.ts`**

Implements the tree → graph transformation:

- **Exports:**
  - `treeToWorkflow(tree: TreeNode[], stepStates: Map<string, string>): GraphWorkflow`
  - Interfaces: `TreeNode`, `Branch`, `GraphNode`, `GraphEdge`, `GraphWorkflow`

- **Algorithm:**
  - Traverses the tree structure depth-first
  - Creates graph nodes for steps, conditions, joins, iterates, start/end
  - Connects nodes with edges (sequential, conditional, back-edge)
  - Applies deterministic hierarchical layout (BFS-based layering)

- **Key Design Decisions:**
  - **Deterministic Layout:** Uses layer-based positioning (left-to-right, one layer per hop from start). Same tree always produces same layout.
  - **Edge Classification:** Sequential, conditional, success/failure, back-edge type hints for visual styling.
  - **Path Tracking:** Tracks execution path via `pathTaken` flag on edges (green if on path, gray if unreached).
  - **Loop Handling:** Iterate blocks create back-edges to themselves for loop visualization.

### 2. **Modified: `vscode/src/views/runbookPanel.ts`**

- **New State:**
  - `workflowViewMode: 'tree' | 'graph'` — tracks user preference
  - Persisted to VS Code global state for session continuity

- **New Method:**
  - `renderGraphSvg(): string` — generates SVG canvas with nodes and edges
    - Node colors by type (step=dark, condition=brown, iterate=purple, start=green, end=red)
    - Edge styles: solid (sequential), dashed (conditional), dotted (back-edge)
    - Green highlights for executed path, gray for unreached

- **New Event Handler:**
  - `type: 'toggleWorkflowView'` — switches between tree/graph modes

- **Updated HTML:**
  - Added toggle button in workflow map header (`◐ Graph` / `∿ Tree`)
  - Conditional rendering: `this.workflowViewMode === 'graph' ? renderGraphSvg() : workflowHtml`

- **New Script Function:**
  - `toggleWorkflowView(mode)` — posts message to toggle view

---

## Architecture & Design

### Transformation Pipeline

```
tree: TreeNode[] 
  ↓ [treeToWorkflow]
graph: { nodes: GraphNode[], edges: GraphEdge[] }
  ↓ [renderGraphSvg]
SVG HTML <canvas>
```

### Layout Algorithm

**Deterministic Hierarchical (Layer-Based):**
1. Assign layer to each node via BFS from start
2. Group nodes by layer
3. Position nodes: `x = layer * LAYER_WIDTH`, `y = nodeIndex * ROW_HEIGHT`
4. No randomness → reproducible layouts for testing

### Visual Treatment

| Element | Color | Style |
|---------|-------|-------|
| **Running** | Blue | Solid border |
| **Passed** | Green | Solid border |
| **Failed** | Red | Solid border |
| **Skipped** | Gray | Dimmed node |
| **Sequential edge** | Path color | Solid line |
| **Conditional edge** | Path color | Dashed line, label |
| **Back-edge (loop)** | Path color | Dotted line |

---

## Known Limitations & Future Work

### Current Limitations

1. **SVG Rendering Only:** No interactivity beyond clicking nodes to view step detail. No pan/zoom yet.
2. **Chain History Not Shown:** Graph renders only current runbook tree, not chain history.
3. **Branch Layout Simplistic:** Branches are rendered linearly. Complex nested branches may overlap.
4. **No Edge Crossing Avoidance:** Layout algorithm does not apply Sugiyama (median rank) to minimize edge crossings — fine for small graphs, may be cluttered for large ones.
5. **Fixed Node Size:** All nodes 140×50px, no dynamic sizing for longer labels.

### Future Enhancements (Phase 2)

- **Pan/Zoom:** SVG pan-and-zoom library (e.g., d3-zoom) or canvas-based rendering
- **Edge Routing:** Orthogonal edge routing to avoid overlaps
- **Animated Playback:** Re-run the execution animation on the graph (highlight visited nodes/edges)
- **Interactive Features:** Hover to see step details, collapse/expand nested branches
- **Chain Graph:** Option to show entire chain of runbooks as connected DAGs
- **Export:** Save graph as PNG/SVG for documentation

---

## Testing & Validation

### Manual Validation

✅ Toggle button appears in workflow map header  
✅ Clicking toggle switches between tree and graph views  
✅ Graph renders all steps, branches, and iterate blocks  
✅ Active/running step highlighted in graph  
✅ Passed/failed steps show correct visual state  
✅ View preference persists across runs (globalState)  
✅ Compilation succeeds with no TypeScript errors  

### Assumptions About TreeNode Shape

The implementation assumes the tree structure from `gert serve` has:
- `.step.id`, `.step.type`, `.step.title`, `.step.outcomes`
- `.iterate.id`, `.iterate.steps`, `.iterate.as`, `.iterate.mode`, `.iterate.max`, `.iterate.total`
- `.branches[]` with `.condition`, `.label`, `.steps`

**Note:** This matches the current `pkg/engine/types.go` and the output from `gert serve exec/start`. No authoring-side changes needed (read-only POC).

---

## Files Changed

- ✅ **Created:** `vscode/src/views/treeToGraph.ts` (267 lines)
- ✅ **Modified:** `vscode/src/views/runbookPanel.ts` (+200 lines, 4 sections updated)
  - Import treeToGraph
  - Add workflowViewMode state
  - Add toggleWorkflowView handler
  - Add renderGraphSvg() method
  - Update HTML template for toggle + conditional rendering
  - Add toggleWorkflowView() JS function

---

## Proof-of-Concept Success Criteria

- ✅ User can toggle between tree and workflow views
- ✅ Workflow graph renders complete tree structure (nodes + edges match topology)
- ✅ Active/running step visually stands out
- ✅ Passed/failed steps have distinct visual markers
- ✅ Condition branches are labeled
- ✅ Clicking a node in graph shows step detail (via viewStep handler)
- ✅ Layout is deterministic (same tree = same canvas positions)

---

## Next Steps

1. **Gather feedback** on graph rendering clarity and layout from users
2. **Measure performance** on large runbooks (>50 steps) to assess need for canvas-based rendering
3. **Design Phase 2:** Pan/zoom, animated playback, chain history in graph
4. **Authoring Editor:** Apply same treeToGraph to visual editor for branch/iterate editing preview
