# Decision: Iterate Block Visual Language Split

**Date:** 2026-06-26  
**By:** Kurapika (Frontend Engineer)  
**Status:** Implemented

## Decision

Sequential and parallel `iterate` blocks are now rendered with distinct visual languages.

### Sequential iterate (no `concurrency` or `concurrency: 1`)
- **Visual:** n8n-style container rect wrapping body steps
- **Node type:** still `'iterate'`, distinguished by `data._iterateType = 'sequential'`
- **Header label:** `↻ for each {as} in {over}` — clearly communicates the loop variable
- **No back-edge** — the container rect IS the visual loop indicator
- **Container geometry** stored as `data._containerHeight` / `data._containerWidth` for the renderer

### Parallel iterate (`concurrency > 1`)
- **Visual:** Fork diamond `◇ ×N` → body column → Join diamond `◇ join`
- **Node types:** `'fork-diamond'` and `'join-diamond'` (new types added to `GraphNode.type` union)
- **No container rect** — the fork/join chrome frames the body
- **No back-edge**

## Why

The old rendering (dashed-border header node + back-edge bezier) gave no visual signal about whether iteration was sequential or parallel. The two paradigms have fundamentally different execution semantics (one item at a time vs N concurrent), so the static graph should communicate this immediately.

## Type changes

Added to `TreeNode.iterate`: `over?: string`, `concurrency?: number`, `collect?: Record<string, string>`  
Added to `GraphNode.type`: `'fork-diamond'`, `'join-diamond'`

## Backward compatibility

- `expandedIterate` tree nodes still use the original `layoutExpandedIterate()` path (unchanged)
- The `iteratePassStripH` runtime expansion in `renderExecutionGraph` still works (targets `type === 'iterate'` nodes which sequential containers are)
- Pre-existing iterate tests updated: back-edge assertion replaced with container/no-back-edge assertion; parallel test added
