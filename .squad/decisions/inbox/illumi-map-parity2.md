# Decision: Workflow Map Visual Parity — Batch 2

**Date:** 2026-04-06  
**Author:** Illumi  
**Commit:** ce01021  
**Requested by:** ormasoftchile

---

## Summary

Fixed 2 remaining visual gaps between web and VS Code workflow map, based on screenshot comparison.

---

## Fix 1 — Outcome Banner: Subtitle Line + Left Accent Border

### Problem
Web outcome banner only showed one line (headline) and no visual indicator of state category.

### Solution
- Added `<div class="outcome-label">` subtitle below `.outcome-state`, rendering `outcomeResult.state` raw string (e.g. "resolved", "escalated")
- Added left accent border (3px solid) to `.map-outcome-banner` base class
- Per-state border colors:
  - `resolved` → `#4caf50` (green)
  - `escalated` → `#f44336` (red)
  - `needs_rca` → `#569cd6` (blue)
  - `no_action` → `#888` (grey)
- Label CSS: `font-size: 11px; opacity: 0.7; margin-top: 2px`

---

## Fix 2 — Toolbar: Header Row Position + Prune/Auto Buttons

### Problem
- Toolbar was a separate row below the workflow-header, not in the header like VS Code
- Missing Prune and Auto toggle buttons

### Solution
- Moved `.map-toolbar` inside `.workflow-header` with `margin-left:auto` for right-alignment
- Removed standalone toolbar border/background; toolbar inherits header style
- Added buttons:
  - **Separator** `|` between Fit and Prune
  - **Prune** toggle: hides unvisited (pending) nodes. Adds `.pruned` class to `.map-content`, uses `syncPruneClasses()` to add `node-unvisited` to pending step `<g>` elements. CSS: `.pruned .node-unvisited { display: none; }`
  - **Auto** toggle: triggers `fitGraphToWidth` + `applyGraphTransform` on every step state change when enabled
- Button active state: `.map-toolbar button.active { background: rgba(0,120,212,0.3); border-color: #0078d4; }`

### Design decisions
- `node-unvisited` class is maintained on SVG `<g>` elements (not shape children) for reliable CSS targeting
- `syncPruneClasses()` is a full pass over all nodes — called once on Prune toggle to handle already-rendered nodes
- `syncGraphStepState()` incrementally updates `node-unvisited` class on each step transition — no extra overhead
- Edge pruning (`.edge-grey`) CSS rule is included but edges don't yet carry this class; reserved for future enhancement

---

## Files Changed
- `web/src/views/runbookRunner.ts` only (per task constraint)

## Build
- `tsc` — 0 errors
- `vite build` — ✓ built in 49ms
