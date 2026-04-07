# Decision: Graph rendering parity — edge color + branch subtitle

**Date:** 2025-07-17  
**Author:** kurapika  
**Commit:** `fix(web): edge color blue + branch subtitle → true/false suffix`

---

## Context

The web graph renderer (`web/src/shared/renderGraph.ts`) renders taken/active edges in
green (`#4caf50`). VS Code's `stepNodeRenderer.ts` hardcodes `#4da6ff` (blue) for taken
edges. Additionally, branch-step node subtitles in the web showed raw conditions without
indicating which direction was taken; VS Code appends ` → true` / ` → false`.

---

## Decisions

### 1. Taken-edge color: `#4da6ff` (blue)

**File:** `web/src/shared/themes/graphTheme.ts`

Changed `edges.taken.color`, `edges.taken.markerColor`, `edges.backEdge.color`, and
`edges.backEdge.markerColor` from `#4caf50` → `#4da6ff`.

**Why `#4da6ff` specifically:** VS Code's `renderEdges` in `stepNodeRenderer.ts:64`
hardcodes this value. The web theme is the canonical source — updating here fixes both
the edge stroke and the SVG arrowhead marker in one change.

**What was NOT changed:** `states.passed.stroke` (`#4caf50`) and
`filters['glow-passed'].color` — these control node borders and glow for completed
nodes, which remain green and are semantically distinct from the edge path color.

---

### 2. Branch subtitle suffix: `→ true` / `→ false`

**File:** `web/src/shared/treeToGraph.ts`, `layoutStep` function

Updated `branchCondition` computation to append suffixes based on execution state:

| Condition | Suffix |
|-----------|--------|
| Taken branch found (`isTaken` returns true for branch's first step) | ` → true` |
| Step is decided (passed/running/failed) but no taken branch (else path) | ` → false` |
| Step not yet decided | no suffix (raw condition) |

**Rationale:** Matches `vscode/src/views/stepNodeRenderer.ts:255`. The web needed the
same visual signal. The three-state distinction (true / false / pending) is more
informative than VS Code which only shows `→ true` — the `→ false` case helps when
the else/default path was taken.
