# Decision: Web/VS Code Parity Batch 1

**Date:** 2025-01-20
**Author:** Illumi (Web Frontend Engineer)
**Status:** Implemented

## Summary

Implemented 12 visual/functional parity items in `web/src/views/runbookRunner.ts` to align the web app with the VS Code extension.

## Items Implemented

### P1-A — Prose panel header: runbook name + kind badge
- Added `runbookKind: string` to RunState
- `executeRun()` now reads `result.kind` and stores it (lowercased)
- Added `deriveRunbookTitle()` — basename stripping `.runbook.yaml`, dashes→spaces, title-case
- Prose header changed from static `panel-header` to `workflow-header` with title + kind badge
- Badge CSS added per kind: guide, rca, mitigation, reference, composable

### P1-C — Prose arrow indicator on active step
- `renderStepAsProse()` now adds `indicator-arrow` class when step is active
- Always renders `<span class="prose-arrow-marker">&#9654;</span>` inside each section
- `syncProseActiveStep()` updated to toggle `indicator-arrow` alongside `active`
- CSS: `.prose-arrow-marker { display: none; position: absolute; left: 6px; top: 6px; }` / `.indicator-arrow .prose-arrow-marker { display: block; }`

### P2-A — Workflow map header: kind badge
- Changed map header from `panel-header` to `workflow-header` class
- Added same kind badge as prose header

### P2-B — Workflow map: outcome banner
- Added `mapOutcomeBanner` computed in `renderThreePanelLayout()`
- Renders after map header when `outcomeResult` is set
- CSS: `.map-outcome-banner` with state-based color variants

### P2-C — Workflow map: Restart button
- Added restart button to map header when `runCompleted || outcomeResult`
- `window.restart` resets all run state fields and calls `render()`

### P2-D — Workflow map: zoom toolbar
- Added `.map-toolbar` between map header and `.map-content`
- Buttons: `−`, `100%` display, `+`, `Fit`
- `window.zoomIn`, `window.zoomOut`, `window.fitGraph` implemented
- `applyGraphTransform()` now updates `#zoom-pct` text after any zoom change

### P3-A — Right panel: contextual header label
- Added `<div class="workflow-header"><span>${label}</span></div>` at top of active-step-panel
- Label: OUTCOME / COMPLETE / STEP DETAIL / ACTIVE STEP based on state
- Both `renderThreePanelLayout()` and `renderActiveStepPanel()` now include this header

### P3-B — Inputs: uppercase label
- Changed `Inputs (N)` to `INPUTS (N)`
- Added `text-transform: uppercase; letter-spacing: 0.5px` to `.inputs-toggle` CSS

### P3-C — Outcome: RESULT badge instead of emoji
- Removed ✅/❌ emoji outcome banner structure
- Now uses `active-step-header` + `type-badge outcome` + `step-name` structure
- Badge text: `RESULT` if `outcomeIsConclusion(state)`, else `RECOMMENDATION`
- `outcomeIsConclusion` imported from `../shared/helpers` (already existed)

### P3-D — Outcome: recommendation text in `outcome-recommendation-full`
- Changed class from `instructions` to `outcome-recommendation-full` in outcome branch
- CSS: `font-size: 13px; white-space: pre-wrap; line-height: 1.5; margin-top: 12px`

### P3-E — Outcome: Copy Summary button
- Added `<button onclick="copySummary()">📋 Copy Summary</button>` in `.actions`
- `window.copySummary` reads `#summaryText` textarea and writes to clipboard
- `buildSummaryText()` private helper: title + Outcome: headline + recommendation

### P3-F — Outcome: Save for Replay button
- Added `<button onclick="saveForReplay()">💾 Save for Replay</button>` in same `.actions`
- `window.saveForReplay` reads `#replayData` textarea, creates Blob, triggers download as `replay.json`
- Hidden `#replayData` textarea holds `{ runbook, vars }` JSON

## Build Status

✅ Build passes — `npm run build` exits 0, no TypeScript errors.

## Decisions Made

- **Restart resets `runId` to null** — shows "Enter runbook path" state, keeps `runbookPath` pre-filled so user can re-run with one click
- **`buildSummaryText()` is a lightweight inline helper** rather than importing from VS Code's `summary.ts` (which has VS Code-specific dependencies)
- **`replayData` textarea alongside `summaryText`** — avoids re-serializing state in the click handler (consistent with VS Code pattern)
- **`syncProseActiveStep()` updated** — keeps DOM-level sync consistent with `renderStepAsProse()` so both paths set `indicator-arrow`

## Files Modified

- `/Volumes/Projects/gert/web/src/views/runbookRunner.ts`
