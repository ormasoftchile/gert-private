# Decision: Chain visualization model for invoke steps

**Date:** 2026-03-18
**By:** Gon (Lead Architect)
**Status:** Implemented

## What

Chain navigation is view-only state (`viewingChainIndex`) that selects which chain entry's tree/states to render in the graph — it never affects execution flow. The graph renderer (`renderGraphSvg`) accepts the selected chain entry's data and renders it identically to the current entry.

## Why

Invoke chains create a parent-child runbook relationship. Users need to:
1. See where they are in the chain (breadcrumb)
2. Navigate between parent and child graphs (click breadcrumb or invoke badge)
3. Maintain spatial context (parent minimap)

Making chain navigation purely view-layer keeps it safe — no risk of interfering with the execution engine or client-server protocol.

## Implications

- No new server events or protocol changes needed
- Chain history already captures enough data (tree, stepStates, stepDetails) for full graph rendering
- Minimap reuses `treeToWorkflow` at reduced scale — no new layout algorithm
