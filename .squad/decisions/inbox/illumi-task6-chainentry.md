# Decision: ChainEntry.tree typed as TreeNode[] not any[]

**Task:** Phase 1 Task 6 — parentMinimap extraction  
**Date:** 2026-04-07  
**Author:** illumi

## Context

`ChainEntry` in `vscode/src/views/graphRenderer.ts` defines `tree: any[]`.  
The shared type `TreeNode` already exists in `shared/renderer/types.ts` and matches exactly what the tree field contains.

## Decision

Used `tree: TreeNode[]` in the shared `ChainEntry` definition instead of `any[]`.

## Rationale

- `TreeNode` is the canonical type for tree data in this system
- `any[]` provides zero type safety; `TreeNode[]` enables proper downstream typing
- `renderParentMinimap` passes `tree` directly to `treeToWorkflow(tree, ...)` which expects `TreeNode[]`
- No breakage — the VS Code definition was loosely typed, not intentionally `any`

## Impact

Any consumer of the shared `ChainEntry` that passes tree data must ensure it matches `TreeNode`. This is the correct constraint.
