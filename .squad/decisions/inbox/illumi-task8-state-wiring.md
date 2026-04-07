# Task 8 — Web Runner State Wiring: Current vs. Needs Task 9

**Date:** 2026-04-07  
**Author:** Illumi  
**Context:** After wiring `renderExecutionGraph` from `@gert/renderer/graph` into the web runner

---

## GraphRenderOptions fields currently populated by web runner

These fields are NOW passed at the call site in `renderWorkflowMap()`:

| Field | Source in `RunState` | What it enables |
|-------|---------------------|-----------------|
| `delayInfo` | `this.state.delayInfo` | Delay timer badge on active step node |
| `theme` | `defaultGraphTheme` | Node/edge color scheme |
| `outcomeLabel` | `this.state.outcomeResult?.state` | Outcome string in final node |
| `stepDetails` | `this.state.stepDetails` | Step detail display in nodes; invoke boundary |
| `currentStepId` | `this.state.currentStepDetail?.stepId` | Active glow/arrow indicator |
| `snapshotState` | `this.state.snapshotState` | Full snapshot (iteratePasses, history, finished flag) |
| `branchResolutions` | `this.state.snapshotState.branchResolutions` | Resolved branch highlight (taken vs. not-taken) |
| `invokeChildren` | `this.state.snapshotState.invokeChildren` | Inline invoke merge + parent minimap |
| `runCompleted` | `this.state.runCompleted` | Prune: hide unvisited nodes after run ends |
| `outcomeResult` | `this.state.outcomeResult` | Outcome banner in workflow map SVG |

---

## GraphRenderOptions fields NOT yet populated (need Task 9 or later)

These exist on `GraphRenderOptions` but the web runner has no corresponding state yet:

| Field | Why missing | Task needed |
|-------|------------|-------------|
| `annotationCounts` | Web runner has no annotation data source; no annotation RPC yet | Task 9: wire annotation feed |
| `annotations` | Same — no annotation fetch/WS event from backend for web | Task 9 |
| `chainHistory` | Web runner has no chain (invoke-chain) history tracking; single-level execution only | Task 9: implement chain tracking in RunState |
| `viewingChainIndex` | Requires chain navigation UI (prev/next chain) — not implemented in web | Task 9 |
| `selectedIteratePass` | Web has no iterate-pass selection UI | Task 9: add pass selector widget |
| `iteratePassInfoMap` | Populated from WS events not yet emitted/handled in web | Task 9 |
| `iterateChildDetailsByPass` | Deep iterate pass detail tracking — not tracked in RunState | Task 9 |
| `displayConfig` | No user-configurable display preferences in web UI yet | Future task (P3 priority) |
| `debugMode` | No debug toggle in web toolbar | Future task |
| `recording` / `recordingLog` | No recording feature in web | Future task |
| `autoScreenshot` | No screenshot feature in web | Future task |
| `savedGraphTransform` | Web has no persisted graph transform between page loads | Future task |
| `findChildChainIndex` | Callback requires chain navigation state — see chainHistory | Task 9 |
| `getIteratePassDetail` | Callback requires iterateChildDetailsByPass — see above | Task 9 |

---

## Summary

**Task 8 achieved:** Basic graph features (step states, branch coloring, active indicator, invoke merge, outcome banner, prune) are now fully wired. The shared renderer's Phase 1 features are available as soon as the web runner collects the corresponding state.

**Task 9 scope:** Chain navigation + annotation counts + iterate pass selection. These require new WS event handling, new RunState fields, and new UI controls (chain breadcrumb, pass selector, annotation badge click).
