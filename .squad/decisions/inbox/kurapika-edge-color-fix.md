# Decision: Fix edge color for unvisited branch paths to end node

**Date:** 2026-06-26
**By:** Kurapika
**Status:** Implemented

## What

In `web/src/shared/treeToGraph.ts`, edges from branch leaf nodes to the synthetic `end-0` node were hardcoded as `taken: true`. This caused all paths into the end node to render green, even when the source branch was never executed.

## Root Cause

`treeToWorkflow()` creates the end node and its incoming edges at the bottom of the file (lines ~689–707). Unlike every other edge in the function — which correctly derives `taken` from `isTaken(nodeStepId(...))` — these end-node edges were unconditionally set to `taken: true`.

This affected two cases:
1. `branchBottomIds` — multiple branches reconverging into end (the branching runbook scenario)
2. Single `lastId` — the linear tail of the workflow connecting to end

## Fix

Changed both cases to use `isTaken(nodeStepId(bid))` and `isTaken(nodeStepId(lastId))` respectively, consistent with how all other edges determine their taken state.

## Why This Approach

Keeps the fix minimal and aligned with existing patterns in the same file. No new state or logic needed — the `isTaken`/`nodeStepId` helpers already exist and are used everywhere else.

## Acceptance

- Edges from unvisited nodes → end: grey ✅
- Edges from visited nodes → end: green ✅
- End node circle: green when run completed ✅ (node state is separate from edge state)
- Build: passes with zero TypeScript errors ✅
