# Merge Decisions: shared/renderer/graph/treeToGraph.ts

**Author:** Kurapika  
**Date:** 2026-06-26  
**Task:** Phase 1 Task 1 — Unified treeToGraph + treeOps

---

## Context

Two implementations of `treeToGraph.ts` existed in the codebase:
- `vscode/src/views/treeToGraph.ts` — canonical, full feature set (expandedIterate, invokeBody, branch centering)
- `web/src/shared/treeToGraph.ts` — web port with three critical bug fixes

The merge target is `shared/renderer/graph/treeToGraph.ts`.

---

## Decision A: branchCondition field on GraphNode

**What:** Added `branchCondition?: string` to `GraphNode` in `shared/renderer/types.ts` (was already there from Phase 0) and ported the computation from web's `layoutStep()`.

**Logic:**
- If the branch decision was made AND a branch was taken: `"<condition> → true"`
- If the branch decision was made but no branch first-step is taken (else path): `"<condition> → false"`
- If the step has branches but no decision yet: raw condition string, no suffix

**Why:** Gives every branch-step node a readable human subtitle showing which condition fired and in what direction. VS Code's version had none; web added this and it's clearly correct behavior.

---

## Decision B: Passthrough nodes must NOT be in nodeStepMap (CRITICAL)

**What:** Removed `nodeStepMap.set(passId, id)` from the passthrough node creation block.

**VS Code had:** `nodeStepMap.set(passId, id)` — maps the passthrough graph-node-id → parent step id.

**Web's fix:** Deliberately omits this mapping with the comment: "must not resolve to parent branch-step, otherwise isTaken would return true and edges into this passthrough would render green."

**Why this matters:**  
`nodeStepId(passId)` is used when computing `taken` for edges. If `passId` maps to the parent step's id, then `isTaken(nodeStepId(passId))` returns `true` whenever the parent ran — even if that branch was never taken. This causes untaken branch paths to render green in the execution graph. The passthrough node represents a collapsed/skipped branch and must stay grey/skipped.

**Decision:** Web is correct. passthrough nodes are never added to `nodeStepMap`.

---

## Decision C: End-node incoming edges must use isTaken() not hardcoded true

**What:** Changed end-node incoming edge `taken` from `taken: true` to:
- `taken: isTaken(nodeStepId(bid))` for multi-branch reconvergence
- `taken: isTaken(nodeStepId(lastId))` for the single-bottom path

**VS Code had:** Both hardcoded as `taken: true`.

**Web's fix (previously applied as standalone commit):** Uses the same `isTaken(nodeStepId(...))` pattern that every other edge in the function uses.

**Why:** Hardcoded `true` means the edge from the last node to `end-0` always renders green, even when the workflow hasn't completed or took a different branch that leads to a terminal step. Causes false visual feedback.

**Decision:** Web is correct. This fix was already recorded in Kurapika's history (session 2026-06-26 "Fix web graph edge color").

---

## Structural decision: No type re-exports in shared/renderer/graph/ files

`shared/renderer/graph/treeToGraph.ts` previously had:
```typescript
export type { TreeNode, Branch, GraphNode, GraphEdge, GraphWorkflow } from '@gert/renderer';
```
And `treeOps.ts` had:
```typescript
export type { InvokeChildData } from '@gert/renderer';
```

These were removed because:
1. `@gert/renderer` IS `shared/renderer` — re-exporting from yourself is circular at the barrel level
2. `shared/renderer/index.ts` now includes `export * from './graph'`, which would cause duplicate named exports
3. The types are already available via the barrel — consumers don't need them re-exported from the graph module

---

## What was NOT changed

- All VS Code-exclusive features are preserved: `layoutExpandedIterate`, `invokeBody` compound layout, branch centering (taken branch centered, collapsed branches distributed), terminal branch detection, `options.omitEnd`
- `treeOps.ts`: all functions preserved, `prefixChildTree` made public (added `export`)
- No VS Code or web importers were modified — this is purely additive
