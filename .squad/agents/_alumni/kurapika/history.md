
## Sessions

### 2026-06-26: Phase 1 Task 1 — Unified treeToGraph + treeOps in shared/renderer/graph/

**Requested by:** ormasoftchile

**Task:** Merge VS Code's treeToGraph.ts (canonical, full features) with web's treeToGraph.ts (critical bug fixes) into `shared/renderer/graph/treeToGraph.ts`.

**Source files reviewed:**
- `vscode/src/views/treeToGraph.ts` (708 lines) — canonical base with expandedIterate, invokeBody, all node types
- `web/src/shared/treeToGraph.ts` (728 lines) — has branchCondition (Fix A) and passthrough fix (Fix B) and end-edge fix (Fix C)
- `vscode/src/views/treeOps.ts` (255 lines) — superset with mergeInvokeChildren, prefixChildTree, pruneTrailingUnexecuted
- `web/src/shared/treeOps.ts` (33 lines) — just re-exports, no logic

**What was found:** `shared/renderer/graph/treeToGraph.ts` already existed (from a prior barrel commit) but was missing all three web bug fixes. The graph/index.ts already exported from treeToGraph and treeOps.

**Merge decisions:**

Fix A (branchCondition): Ported `branchCondition` computation block from web's `layoutStep()`. Computes the human-readable subtitle — `condition → true` if the taken branch matched, `condition → false` if decided but no branch taken, raw condition if undecided. Added to the `addNode()` call as `branchCondition` field on `GraphNode`.

Fix B (passthrough nodeStepMap): Removed `nodeStepMap.set(passId, id)` that VS Code had for passthrough nodes. Web's version deliberately omits this: if passId resolves to the parent branch-step id, `isTaken()` would return true (because parent ran) and edges into the passthrough would render green even though the branch was never taken.

Fix C (end-node edges): Changed `taken: true` → `isTaken(nodeStepId(bid))` and `isTaken(nodeStepId(lastId))` for end-node incoming edges. VS Code had these hardcoded, web fixed them. Aligns with the pattern used throughout the rest of the function.

**Additional cleanup:**
- Removed `export type { TreeNode, Branch, GraphNode, GraphEdge, GraphWorkflow }` re-export from treeToGraph.ts (would cause circular conflict with shared/renderer/index.ts)
- Removed `export type { InvokeChildData }` re-export from treeOps.ts (same reason — already in types.ts barrel)
- Made `prefixChildTree` an exported function in treeOps.ts (task requirement)
- Added `export * from './graph'` to `shared/renderer/index.ts`

**Verified:** `cd web && npm run build` passes (zero TypeScript errors). VS Code `npm run compile` passes.

**Commit:** `feat(shared/renderer): Phase 1 Task 1 — unified treeToGraph + treeOps`

---

### 2026-06-26: Fix web graph edge color — unvisited branches showing green

**Requested by:** ormasoftchile

**Bug:** In `web/src/shared/renderGraph.ts` (execution graph), edges from unvisited/not-taken branch paths to the terminal `end-0` node were rendered green instead of grey.

**Root cause:** `treeToGraph.ts` — the `treeToWorkflow()` function creates the end node and its incoming edges at the bottom of the build pass. Both cases (multi-branch `branchBottomIds` and single `lastId`) were hardcoded `taken: true`, unlike every other edge in the function which correctly calls `isTaken(nodeStepId(...))`.

**Fix:** Two-character change — replaced `taken: true` with `isTaken(nodeStepId(bid))` and `isTaken(nodeStepId(lastId))` for the end-node edges. Aligned with existing patterns throughout the file.

**Files changed:** `web/src/shared/treeToGraph.ts`

**Verified:** `npm run build` passes, zero TypeScript errors.

**Commit:** `fix(web): only color edges green when source node was actually visited`

---

### 2026-03-21: Delay UX verification audit — confirmed correct, Extension Host reload required

**Requested by:** ormasoftchile

**Finding:**
Full audit of delay UX flow confirms the implementation is correct across all execution paths. The compiled bundle contains all delay code. The issue is that the Extension Host was not reloaded after compilation — VS Code runs the bundle loaded at activation time, not the on-disk file.

**Verified paths:**
- ✅ `event/stepStarted` handler sets `delayInfo` from `msg.params.delay` (serve layer includes it at line 1275 of serve.go) with `findStepInTree` fallback
- ✅ `event/stepDelaying` handler reconfirms delayInfo
- ✅ `event/stepCompleted` handler clears delayInfo only for matching stepId
- ✅ Rendering: both `processing=true` and `processing=false` paths render the delay indicator
- ✅ Go serve layer sends stepStarted+stepDelaying BEFORE timer block — events reach client during the 5s delay window
- ✅ Auto-advance: `nextInFlight` guard prevents double-advance; 5s server-side delay blocks execNext response
- ✅ Event coalescing: 5-second gap between stepDelaying and stepCompleted prevents coalescing
- ✅ Compiled bundle verified: contains delay-active CSS, pulse-delay animation, event/stepDelaying handler

**Changes:**
- Added diagnostic logging: stepStarted now logs `delay:` field value; new log `delay indicator set from stepStarted:` when delayInfo is populated
- Recompiled bundle (esbuild)

**Files Modified:**
- `vscode/src/views/runbookPanel.ts` — added delay diagnostic console.logs in event/stepStarted handler

## Learnings
- VS Code Extension Host runs the bundle loaded at activation time. Recompiling without reloading the Extension Host ("Developer: Reload Window") has no effect — this is the #1 false-negative for UI feature verification
- The Go serve layer's unbuffered os.Stdout writes ensure stepStarted and stepDelaying events reach the client before the delay timer blocks, preventing event coalescing with stepCompleted
- Invoke child detail pane state lookup: `stepStates` stores invoke-child states under fully-qualified keys (`parentId::childId`), but the detail pane was looking up the bare `childId` after stripping the `::` prefix for `findStepDetail`. Fix: try `this.viewingStepId` (the FQ key) first when it contains `::`, then fall back to bare ID lookup
- Graph reflow flash root cause (#13): `updateWebview()` sets `panel.webview.html` which replaces the entire DOM. The pan/zoom init script recalculates `fitScale` from new SVG dimensions, producing a different transform even when `vscode.getState()` has saved state. The `fitsInViewport` check and `opacity:0` fade-in hack were both WRONG — they masked the symptom instead of fixing the cause. **Correct fix:** when `prevState.graphScale` exists, restore it unconditionally (no viewport check, no fitScale recalculation). The saved transform IS the user's camera position; applying it exactly means the camera doesn't move and new nodes (e.g. End) simply appear in the extended SVG area below. No fade, no visibility hack, no transition needed — the graph container should be immediately visible.

### 2026-03-20: Fix step delay indicator not visible in webview

**Requested by:** ormasoftchile

**Root Cause:**
The `event/stepDelaying` handler correctly set `this.delayInfo` and called `updateWebview()`, but the delay indicator was invisible because:
1. `event/stepStarted` rendered the step detail FIRST (without delay info), and `event/stepDelaying` arrived in the same readline tick — both HTML assignments happened before the webview could paint, so only the timing of the LAST write mattered
2. When both `event/stepDelaying` and `event/stepCompleted` arrived in the same data chunk (coalesced by pipe buffering), delayInfo was set and immediately cleared in one tick — the indicator never rendered
3. Even when visible, the delay indicator rendered BELOW the running spinner as a secondary element, easy to miss

**Fix:**
- In `event/stepStarted`: proactively set `this.delayInfo` by looking up the step's `delay` field in the tree data (via `findStepInTree`). This ensures the delay indicator is present in the FIRST `updateWebview()` call, before `event/stepDelaying` arrives
- Clear previous step's delayInfo at the start of `event/stepStarted` to prevent stale indicators
- In the detail rendering: show the delay indicator INSTEAD of the generic running spinner when delay is active (exclusive, not stacked). Delay gets a distinct warm-yellow background (`delay-active` class) and pulsing ⏳ badge
- In the processing indicator: also show delay info if `delayInfo` is set, so delay is visible even during the `processing=true` state before events arrive
- Added CSS: `.executing-indicator.delay-active` with yellow tint, `.delay-badge` with `pulse-delay` animation

**Completed:**
- ✅ Delay indicator now appears immediately when a step with `delay` starts (from tree data, not just events)
- ✅ Prominent visual with warm-yellow border/background and pulsing hourglass
- ✅ Shows in BOTH processing state and active step detail state
- ✅ Properly cleared when step completes
- ✅ Zero TypeScript errors, clean build (`npx tsc --noEmit`)

**Files Modified:**
- `vscode/src/views/runbookPanel.ts` — proactive delay from tree in stepStarted handler, exclusive delay/spinner rendering, delay-aware processing indicator, CSS for delay-active state

## Learnings
- Event coalescing in readline/pipe IO means you can't rely on intermediate event renders being visible — state that must be seen should be derived from already-available data (tree structure) rather than waiting for a separate event
- Exclusive indicator rendering (delay OR spinner, not both) makes the delay state unmissable instead of being buried as a secondary element

### 2026-03-20: Build configurable minimap for graph view + remove bird's eye zoom

**Requested by:** Cristián Ormazábal Ortega

**Completed:**
- ✅ **Removed bird's eye zoom** — deleted ~120 lines: all birdEye state variables, animateTo(), showViewportIndicator(), removeViewportIndicator(), cancelBirdEye(), keydown/keyup/mousemove/mouseenter/mouseleave/blur event handlers. Cleaned up the mousedown guard that referenced birdEyeActive
- ✅ **Canvas 2D minimap** — renders colored rectangles for nodes (blue=passed, red=failed, bright blue=running with pulse, gray=pending/skipped), blue circle for start, red circle for end, diamond for conditions, thin gray lines for edges
- ✅ **Viewport indicator** — semi-transparent blue rectangle showing the currently visible region of the main graph, updates on every pan/zoom
- ✅ **Click to jump** — clicking anywhere on the minimap centers the main graph on that point
- ✅ **Drag viewport** — dragging on the minimap pans the main graph in real time
- ✅ **Three visibility modes** via `gert.graph.minimap` setting: "visible" (always shown), "ctrl" (show on Ctrl hold with keydown/keyup/blur), "disabled" (never rendered)
- ✅ **Vertical position** via `gert.graph.minimapPosition`: "top" or "bottom" (default)
- ✅ **Horizontal position** via `gert.graph.minimapPanel`: "left", "center" (default), "right"
- ✅ **Aspect-ratio aware sizing** — minimap canvas dimensions computed proportional to graph aspect ratio (max 200x140, min 100x80)
- ✅ **Graph view only** — minimap lives inside renderGraphSvg(), never renders in tree view
- ✅ **DisplayConfig wired** — minimap/minimapPosition/minimapPanel fields added to DisplayConfig interface and readDisplaySettings()
- ✅ **VS Code settings** — 3 new settings in package.json contributes.configuration
- ✅ **Did NOT modify** treeToGraph.ts, treeOps.ts, or snapshotStateMachine.ts
- ✅ Zero TypeScript errors, clean build (`npm run compile`)
- ✅ All 198 real tests pass (2 pre-existing failures in `validate.test.ts` — unrelated)

**Files Modified:**
- `vscode/src/views/runbookPanel.ts` — removed bird's eye zoom block, added minimap canvas HTML generation + webview script for Canvas 2D rendering, click-to-jump, drag-viewport, Ctrl-to-show
- `vscode/src/serve/client.ts` — added minimap, minimapPosition, minimapPanel to DisplayConfig interface
- `vscode/package.json` — added gert.graph.minimap, gert.graph.minimapPosition, gert.graph.minimapPanel settings

## Learnings
- Canvas 2D is the right rendering choice for a minimap overlay: zero DOM overhead, trivially fast even for 200+ nodes, and viewport indicator updates at frame rate without SVG attribute manipulation
- Serializing node data as HTML data attributes (data-nodes, data-edges with JSON) keeps the webview script self-contained — no postMessage round-trips needed for minimap rendering
- The applyTransform override pattern (wrapping the outer IIFE's function) cleanly hooks minimap repaint into every pan/zoom event without modifying the original function

### 2026-03-20: Fix invoke child raw-key collision corrupting parent step states

**Requested by:** Cristián Ormazábal Ortega

**Root Cause:**
In `snapshotStateMachine.ts`, the `event/stepStarted`, `event/stepCompleted`, and `event/stepSkipped` handlers unconditionally wrote `p.stepId` (the raw, unprefixed child step ID) to `stepStates`, even for invoke children. This polluted the map with raw keys like `check_health: running`. If a parent step happened to share the same ID as a child step, the raw key overwrote the parent's 'pending' state with the child's active state — making parent steps after an invoke render as bright/active instead of dimmed.

**Fix:**
For all three event handlers, gate the raw `p.stepId` write behind `else` — when `invokeChild && parentStepId` is present, only write the fully-qualified key (`fqPrefix::stepId`). The raw key is never needed: the merged tree uses FQ-prefixed IDs for child nodes, and `resolveState()` in `treeToGraph.ts` looks up by exact ID.

**Completed:**
- ✅ `event/stepStarted`: moved `next.set(p.stepId, 'running')` into else-branch; invoke children only write FQ key
- ✅ `event/stepCompleted`: moved `next.set(p.stepId, p.status)` into else-branch; invoke children only write FQ key
- ✅ `event/stepSkipped`: moved `next.set(p.stepId, 'skipped')` into else-branch (parentStepId check); orphan fallback stays in else
- ✅ Updated 2 test assertions in `snapshotStateMachine.test.ts` — raw child keys are now correctly absent
- ✅ Golden snapshot test passes (incident-triage fixture unchanged — no regressions)
- ✅ All 198 real tests pass (2 pre-existing failures in `validate.test.ts` — unrelated)
- ✅ Zero TypeScript errors, clean build

**Files Modified:**
- `vscode/src/views/snapshotStateMachine.ts` — gated raw stepId writes for invoke children in 3 event handlers
- `vscode/src/views/snapshotStateMachine.test.ts` — updated 2 assertions to expect raw key is NOT set

## Learnings
- Invoke child events set BOTH the raw and FQ-prefixed stepId in the state map. The raw key is never looked up by the graph renderer (which uses FQ-prefixed IDs from the merged tree), but it silently corrupts any parent step sharing the same ID — a latent collision bug
- The fix is to never write the raw key for invoke children: the if/else gate ensures only parent steps write the raw key

### 2026-03-19: Fix nested invoke prefix mismatch + golden test CRLF parser

**Requested by:** Cristián Ormazábal Ortega

**Completed:**
- ✅ Bug 2 (CRLF): Added `.replace(/\r\n/g, '\n')` in `parseGolden()` before splitting lines — golden file markers now match correctly on Windows
- ✅ Bug 1 (nested invoke prefix): Added `resolveInvokePrefix()` helper to `snapshotStateMachine.ts` — searches `stepStates` for fully-qualified keys ending with `::parentStepId` that have running/passed state
- ✅ Applied `resolveInvokePrefix` to all 4 event handlers that use `parentStepId`: `event/invokeStarted`, `event/stepStarted`, `event/stepCompleted`, `event/stepSkipped`
- ✅ `event/invokeStarted` now registers `invokeChildren` with both raw and FQ key so `mergeWalk` direct lookups work alongside prefix-stripping fallback
- ✅ Regenerated golden file — old golden was stale (never actually tested due to CRLF bug) and missed post-branch sequential nodes in GRAPH view
- ✅ Golden test passes: 2/2 tests (replay matches + invariants hold at every event)
- ✅ All 198 real tests pass (2 pre-existing failures in `validate.test.ts` — unrelated)
- ✅ Zero TypeScript errors, clean build

**Files Modified:**
- `vscode/src/views/snapshotStateMachine.ts` — `resolveInvokePrefix()` helper, FQ prefix resolution in 4 event handlers, dual-key `invokeChildren` registration
- `vscode/src/views/snapshotReplay.ts` — CRLF normalization in `parseGolden()`
- `vscode/src/views/__fixtures__/geodr0002-failover/expected.golden` — regenerated to include post-branch sequential nodes now visible in graph

## Learnings
- Golden files that were never actually tested (0+0 = PASS) are a silent trap — always verify golden parsers can produce >0 entries before trusting green
- Nested invoke prefix resolution is a single-source concern: every event handler that touches `parentStepId` must resolve through the same FQ lookup, not just `invokeStarted`

### 2026-03-19: Rewrite runbookPanel.ts to use snapshotStateMachine.applyEvent() as single source of truth

**Requested by:** Cristián Ormazábal Ortega

**Completed:**
- ✅ Added `snapshotState: SnapshotState` field to `RunbookPanel`, initialized via `initialState(tree)`
- ✅ Replaced all event handler state logic (`event/stepStarted`, `event/stepCompleted`, `event/stepSkipped`, `event/invokeStarted`, `event/outcomeReached`, `event/runCompleted`, `event/branchResolved`, `event/iteratePassEnd`) with `applyEvent()` calls — UI-only side effects preserved
- ✅ Converted `stepStates`, `invokeChildren`, `branchResolutions` from private fields to getters delegating to `this.snapshotState`
- ✅ Updated `renderGraphSvg()` to use `this.snapshotState.finished` for `isFinished` (with `isChainView` fallback)
- ✅ Updated initialization paths: `create()`, `restart`, and `chainToRunbook` all use `initialState(tree)` 
- ✅ Deleted dead code: `setStepState()`, `buildRuntimeIdMapping()`, `initChildStepStates()`, `runtimeToGraphIds` field, `initTreeStates()` free function
- ✅ Zero TypeScript errors, clean build (`npm run compile`)
- ✅ All 198 real tests pass (2 pre-existing failures in `validate.test.ts` from empty fixture dirs — unrelated)

**Files Modified:**
- `vscode/src/views/runbookPanel.ts` — full state delegation to `applyEvent()`, getter pattern, dead code removal

## Learnings
- The getter pattern (`get stepStates()`) is the cleanest approach when many read sites exist — avoids a massive find-and-replace while keeping type compatibility
- `applyEvent` is pure (returns new state), so storing `this.snapshotState = applyEvent(...)` in each handler is the correct integration pattern
- The `runtimeToGraphIds` cross-pollination bug was the exact kind of divergence this refactor eliminates — the state machine handles prefix logic correctly without maintaining a separate mapping

### 2026-03-18: Automated anomaly detection for recording system

**Requested by:** Cristián Ormazábal Ortega

**Completed:**
- ✅ Inline anomaly detection after every state snapshot in `updateWebview()` — cross-references graph node states vs `stepStates` map with prefix-stripping ID resolution
- ✅ State regression detection: flags steps that go from "passed" back to "pending"
- ✅ Render gap detection: flags when state changed but no graph_snapshot followed within 500ms
- ✅ Orphan event detection: flags events referencing step IDs absent from all graph snapshots (with prefix fallback)
- ✅ Frozen graph detection: flags when `invokeStarted` fires but node count doesn't change across consecutive graph snapshots
- ✅ Anomaly entries pushed to `recordingLog` as `type: 'anomaly'` with count and details
- ✅ `writeRecording()` now also writes `recording-summary.json` alongside the JSONL
- ✅ Summary includes: runId, totalEvents, duration, stepsExecuted, finalStates, categorized anomalies (stateSync, renderGaps, orphanEvents, frozenGraph, stateRegressions), unreachedNodes, and human-readable timeline
- ✅ Timeline extracts significant events: step start/complete, branch resolved, invoke started/completed, outcome reached, run completed
- ✅ Long render gaps (>2s with events in between) detected in summary post-processing
- ✅ `gert.viewRecording` command registered in `package.json` and `extension.ts`
- ✅ Command opens file picker for `.jsonl` files, reads recording + summary, displays formatted report in an Output Channel
- ✅ Report shows: header, run metadata, final states, anomalies (red-flagged with categories), unreached nodes, full timeline
- ✅ Falls back to raw JSONL parsing if summary file is missing
- ✅ Recording save notification now includes anomaly count when > 0
- ✅ Zero TypeScript errors, clean build

**Files Modified:**
- `vscode/src/views/runbookPanel.ts` — inline anomaly detection in `updateWebview()`, `buildRecordingSummary()` method, updated `writeRecording()` to emit summary JSON
- `vscode/src/extension.ts` — `gert.viewRecording` command with Output Channel viewer
- `vscode/package.json` — registered `gert.viewRecording` command

### 2026-03-18: Debug recording mode for execution viewer

**Requested by:** Cristián Ormazábal Ortega

**Completed:**
- ✅ Added `recording: boolean` and `recordingLog: any[]` state to `RunbookPanel`
- ✅ Record toggle button (🔴 Rec / ⏹ Stop) in Workflow Map header, with pulsing red dot indicator when active
- ✅ All JSON-RPC events captured at the top of `handleEvent()` before the switch
- ✅ All webview user actions captured at the top of `handleWebviewMessage()` before the switch
- ✅ State snapshots appended after every `updateWebview()` render (stepStates, branchResolutions, invokeChildren, captures, outcomeResult, etc.)
- ✅ Graph structure snapshots in `renderGraphSvg()` (node positions, states, counts — not full SVG string)
- ✅ Click target capture via `document.addEventListener('click')` in webview script — posts `recording-click` messages with tag, class, stepId, coordinates
- ✅ `recording-click` messages handled in extension and appended to recording log
- ✅ Recording auto-stops on `event/runCompleted` and `event/outcomeReached` (non-chained), writes log to disk
- ✅ Manual stop via ⏹ button writes log immediately
- ✅ Output: JSONL format at `{runBaseDir}/recording.jsonl`; fallback to `.runbook/runs/debug/recording.jsonl` if no runBaseDir
- ✅ Notification shown with event count and file path on save
- ✅ CSS `pulse-rec` animation for the recording indicator dot
- ✅ Works in both tree and graph view modes
- ✅ Zero TypeScript errors, clean build

**Files Modified:**
- `vscode/src/views/runbookPanel.ts` — recording state, event/action/state/graph capture, UI toggle, click listener, JSONL writer


### 2026-03-18: Inline child runbook steps into parent graph during invoke execution

**Requested by:** Cristián Ormazábal Ortega

**Completed:**
- ✅ Added `mergeInvokeChildren()` and helpers to `treeToGraph.ts` — walks parent tree, replaces invoke nodes with header + prefixed child tree nodes
- ✅ Child step IDs prefixed with parent invoke step ID (e.g. `mitigation_1::check_long_running_txn`) to avoid collisions
- ✅ Added `invokeChildren: Map<string, InvokeChildData>` state to `RunbookPanel`
- ✅ Handler for `event/invokeStarted` — stores child tree data, initializes child step states as pending, triggers re-render
- ✅ Updated `event/stepStarted` and `event/stepCompleted` handlers to also set prefixed child step states for graph rendering
- ✅ Updated `event/stepSkipped` handler to propagate skipped state to prefixed child IDs
- ✅ `renderGraphSvg()` calls `mergeInvokeChildren()` before layout, so child steps render as real nodes in the flow
- ✅ Child node visual treatment: thin 3px left blue accent border + subtle blue-tinted background (`rgba(0,120,212,0.06)`)
- ✅ Invoke header node shows child runbook name as subtitle when children are inlined
- ✅ "View child ↗" badge hidden when children are already inlined in the graph
- ✅ Click handling: prefixed child step IDs mapped back to original IDs for detail panel lookup
- ✅ `invokeChildren` map cleared during restart and chain-to-child transitions
- ✅ Zero TypeScript errors, clean build

**Files Modified:**
- `vscode/src/views/treeToGraph.ts` — added `InvokeChildData` interface, `mergeInvokeChildren()`, `mergeWalk()`, `prefixChildTree()`, `prefixNode()` functions
- `vscode/src/views/runbookPanel.ts` — `invokeChildren` state, `event/invokeStarted` handler, prefixed state propagation in step events, merged tree in `renderGraphSvg()`, child node visuals, click ID mapping, reset on restart/chain

---

### 2026-03-18: Drag-to-reorder + Host-adaptive viz abstraction

**Requested by:** Cristián Ormazábal Ortega

**Task 1 — Drag-to-reorder steps in editor graph:**
- ✅ Added mousedown/mousemove/mouseup handlers on `.ed-node` elements of type step
- ✅ Ghost node (translucent clone at cursor) shown during drag
- ✅ Drop indicator (glowing horizontal line) appears between sibling nodes
- ✅ On drop, posts `reorder-steps` message with `{ path, fromIndex, toIndex }`
- ✅ Extension handler splices the step from fromIndex to toIndex in the parent steps array, re-renders
- ✅ Only step nodes are draggable — condition/iterate/start/end nodes are excluded
- ✅ Same-parent only — no cross-branch or cross-iterate dragging
- ✅ Drag attributes (`data-node-type`, `data-parent-path`, `data-sibling-index`) emitted on all graph nodes

**Task 2 — Host-adaptive visualization abstraction:**
- ✅ Updated `vizAdapter.ts` with expanded `VizAdapter` interface: `renderWorkflow(graph, options)`, `supportsInteraction()`, `maxComplexity`
- ✅ Expanded `VizOptions`: added `theme`, `stepStates`, `branchResolutions`, `iteratePasses`, `zoom` fields
- ✅ `SvgVizAdapter` (exported): full SVG graph rendering, all interactions supported, maxComplexity=200
- ✅ `TextVizAdapter` (exported): Unicode box-drawing tree output from graph edges, no interactivity, maxComplexity=50
- ✅ Factory function `createVizAdapter(host)` preserved for host selection
- ✅ Zero TypeScript errors, clean build

**Files Modified:**
- `vscode/src/views/runbookEditorPanel.ts` — drag CSS, drag script, reorder handler, drag attributes on SVG nodes
- `vscode/src/views/vizAdapter.ts` — expanded interface, renamed/exported adapters, added supportsInteraction + maxComplexity

---

### 2026-03-18: Zoom spec compliance fix (max 2.0 + fit-to-container default)

**Requested by:** Cristián Ormazábal Ortega

**Completed:**
- ✅ Changed max zoom from 3.0 → 2.0 in both `runbookPanel.ts` and `runbookEditorPanel.ts` (wheel handler + zoom-in button)
- ✅ Default zoom now fits SVG content to container width instead of fixed 1.0 scale — calculates `containerWidth / svgWidth` clamped to [0.3, 2.0]
- ✅ Zoom reset button (⊙) now resets to fit-to-container scale instead of fixed 1.0
- ✅ Applied to BOTH execution viewer and editor Flow Map
- ✅ Zero TypeScript errors, clean build

**Files Modified:**
- `vscode/src/views/runbookPanel.ts` — zoom init, wheel handler, button handlers
- `vscode/src/views/runbookEditorPanel.ts` — same

---

### 2026-03-18: Wire branchResolved/iteratePassEnd events + Pan/Zoom

**Requested by:** Cristián Ormazábal Ortega

**Task 1 — branchResolved + iteratePassEnd event wiring:**
- ✅ Added `branchIndex` and `parentStepId` fields to `GraphEdge` interface in `treeToGraph.ts`
- ✅ Set `branchIndex` and `parentStepId` on all conditional edges during layout (both populated and empty branches)
- ✅ Added `branchResolutions` Map (`parentStepId → branchIndex → taken`) and `iteratePassInfoMap` Map (`iterateStepId → {pass, max, converged}`) to RunbookPanel state
- ✅ Updated `event/branchResolved` handler: stores resolution data per parent+branch, triggers re-render
- ✅ Updated `event/iteratePassEnd` handler: stores pass info per iterate block, triggers re-render
- ✅ In `renderGraphSvg()`: post-processes workflow edges to override `taken` from branchResolutions (solid blue = taken, dashed gray = untaken)
- ✅ In `renderGraphSvg()`: iterate nodes now render a badge pill ("Pass 2/5" or "Converged") when pass info is available, with green for converged and blue for in-progress

**Task 2 — Pan/Zoom for large runbooks:**
- ✅ Wrapped SVG content in a `<g id="graph-transform">` element for transform application
- ✅ Added zoom control bar (−, %, +, ⊙ reset) positioned top-right of graph area
- ✅ Mouse wheel zoom: scale 0.3–3.0 range, zooms toward cursor position
- ✅ Click-and-drag panning via translate on the transform group (skips node clicks and zoom controls)
- ✅ Zoom/pan state persisted via `vscode.getState()`/`setState()` — survives re-renders during execution
- ✅ Applied to BOTH `runbookPanel.ts` (execution viewer) AND `runbookEditorPanel.ts` (editor)
- ✅ Zero TypeScript errors

**Files Modified:**
- `vscode/src/views/treeToGraph.ts` — GraphEdge interface + conditional edge metadata
- `vscode/src/views/runbookPanel.ts` — event handlers, state tracking, graph rendering, pan/zoom
- `vscode/src/views/runbookEditorPanel.ts` — SVG wrapper + pan/zoom script

---

### 2026-03-18: Graph Visual Bug Fixes (Post-Live-Test)

**Requested by:** Cristián Ormazábal Ortega — feedback from first live test of service-health-branching graph view.

**Completed:**
- ✅ **Wider nodes**: NODE_W 180→240, ITER_W 180→240 — step titles no longer truncated into gibberish
- ✅ **Longer labels**: Step truncation 20→28 chars, iterate 18→26 chars — readable titles
- ✅ **Compact vertical spacing**: GAP_Y 80→55 — tighter, less wasted whitespace
- ✅ **More branch separation**: GAP_X 60→80 — prevents horizontal node overlap on branches
- ✅ **Invisible join nodes**: Join height from 24px pill → 4px thin line (2px rendered), opacity 0.4 — no more mystery circles at graph bottom
- ✅ **Readable dim branches**: Min opacity for pending/skipped raised from 0.4/0.45 → 0.55 — untaken branch labels now legible
- ✅ **Better edge convergence**: Bezier control points factor in horizontal distance (`dx * 0.3`) — prevents edge crossings when branches reconverge to join node
- ✅ Zero TypeScript errors, clean build

**Files Modified:**
- `vscode/src/views/treeToGraph.ts` — layout constants + join node height
- `vscode/src/views/runbookPanel.ts` — SVG rendering (truncation, join shape, opacity, bezier curves)

---

### 2026-03-18: Production-Quality Execution Viewer Graph

**Completed:**
- ✅ Rewrote `treeToGraph.ts` layout algorithm: top-to-bottom flow, recursive sub-layout positioning, proper branch fan-out with centered children and join reconvergence
- ✅ Rewrote `renderGraphSvg()` in `runbookPanel.ts`: production SVG with drop shadows, bezier curves, arrowhead markers, node type icons, VS Code theme tokens, running pulse animation, hover highlights
- ✅ Node shapes: start=green circle, end=red bullseye, step=180×60 rounded rect with icon + title + type subtitle, condition=diamond, join=pill, iterate=rounded rect with loop icon
- ✅ Edge routing: cubic bezier from bottom-of-source to top-of-target; back-edges routed around right side with distinct dotted style + flow animation
- ✅ Edge styles: solid=sequential, dashed=conditional with label, dotted+animated=back-edge(loop)
- ✅ Color scheme: all colors via `var(--vscode-*)` tokens for dark/light mode; status: running=blue glow, passed=green, failed=red, skipped=dim gray, pending=outline only
- ✅ Edge labels: background pill behind text for readability, truncated at 28 chars
- ✅ SVG viewBox responsive to content with padding
- ✅ Tree ↔ Graph toggle preserved and working
- ✅ Zero TypeScript errors

**Architecture Decisions:**
1. **Recursive sub-layout model**: Each `layoutNode()` returns a bounding box `{topId, bottomId, width, height}` placed relative to (0,0). Parent stacks children vertically and offsets branch columns side-by-side. This naturally handles any nesting depth.
2. **offsetNodes()**: Final positioning pass shifts sets of node IDs by (dx, dy) — avoids re-traversing the tree structure.
3. **SVG `<defs>`**: Shared arrowhead markers, drop-shadow filter, and pulse glow filter defined once.
4. **CSS-in-SVG**: `<style>` block inside the SVG for hover, pulse animation, and dash-flow animation — no external dependencies.
5. **Back-edge routing**: Right-side bulge via cubic bezier prevents overlap with forward edges.

**Key File Paths:**
- `vscode/src/views/treeToGraph.ts` — layout engine (top-to-bottom hierarchical)
- `vscode/src/views/runbookPanel.ts` — `renderGraphSvg()` and `truncLabel()` methods

## Learnings

- SVG `<style>` blocks with `@keyframes` work inside VS Code webview — no need for external CSS.
- `var(--vscode-charts-green)` etc. are reliable for theming in both light and dark modes.
- Diamond shapes for conditions must use `<polygon>` not rotated `<rect>` to avoid hit-test issues.
- Back-edge routing needs explicit right-side offset to stay visually separate from forward edges.
- Recursive layout with bounding boxes scales cleanly to deep nesting without BFS layer assignment.

---

### 2026-03-18: Workflow Graph Visualization POC (Execution Viewer)

**Completed:**
- ✅ Created `vscode/src/views/treeToGraph.ts` - tree → graph transformation utility
- ✅ Implemented deterministic hierarchical layout algorithm (BFS-based layering)
- ✅ Added `renderGraphSvg()` method to RunbookPanel for SVG rendering
- ✅ Integrated toggle button in workflow map header (Tree ↔ Graph)
- ✅ Added `workflowViewMode` state with globalState persistence
- ✅ Implemented graph node styling: states (passed/failed/running/skipped) with color-coded borders
- ✅ Implemented edge styling: sequential (solid), conditional (dashed), back-edge (dotted)
- ✅ Execution path highlighting: taken edges in green, unreached in gray
- ✅ Click-to-view-step integration: nodes clickable to view step detail (reuses existing handler)
- ✅ Compilation successful: no TypeScript errors

**Proof-of-Concept Success Criteria Met:**
1. ✅ User can toggle between tree and workflow views
2. ✅ Workflow graph renders the same tree structure visually (nodes + edges match topology)
3. ✅ Active/running step visually stands out (blue border)
4. ✅ Passed/failed steps have distinct visual markers (green/red borders)
5. ✅ Condition branches are labeled on edges
6. ✅ Clicking a node in graph view highlights it in step detail (existing viewStep handler reused)
7. ✅ Layout is deterministic (same tree input = same canvas positions every time)

**Technical Architecture:**

**Transformation Pipeline:**
- `treeToWorkflow(tree, stepStates)` converts tree → {nodes, edges}
- Nodes: start, step, iterate, condition, join, end
- Edges: sequential, conditional, success, failure, back-edge
- Layout: BFS from start → assign layers → position nodes in grid

**Deterministic Layout Algorithm:**
- Uses breadth-first layer assignment (each layer = hops from start)
- Position formula: `x = layer * 160px`, `y = nodeIndex * 80px`
- No randomness, no heuristics → reproducible layouts for testing
- Suitable for graphs up to ~50-100 nodes; larger graphs may benefit from canvas + pan/zoom

**Implementation Details:**
- SVG-based rendering (no external dependency, works in webview)
- Node colors by type: step (dark gray), condition (brown), iterate (purple), start (green), end (red)
- Edge colors by execution state: green (passed), red (failed/error), gray (unreached)
- Node click handler reuses existing `viewStep()` to show step detail
- View preference persisted to VS Code globalState

**Visual Treatment Applied:**
| State | Color | Opacity |
|-------|-------|---------|
| Running | Blue | 0.9 |
| Passed | Green | 0.9 |
| Failed | Red | 0.9 |
| Skipped | Gray | 0.5 |
| Pending | Gray | 0.3 |

**Key Learnings:**

1. **Tree Structure Already Encodes Graph:**
   - Sequential steps → array order (implicit edges)
   - Branches → conditional edges + decision nodes
   - Iterates → back-edges to the iterate node
   - No new backend data needed; transformation is pure UI concern

2. **Deterministic Layout is Critical for POC:**
   - Enables reproducible testing ("always the same graph for same input")
   - Avoids floating layouts that confuse users between runs
   - Simple hierarchical layering sufficient for execution visualization

3. **SVG + HTML Canvas Works Without React:**
   - No need for external graph libraries (D3, ELK, Sugiyama)
   - Direct SVG rendering from layout algorithm
   - Lightweight, performs well for typical runbook sizes

4. **Path Tracking via Execution State Map:**
   - `stepStates: Map<stepId, 'running'|'passed'|'failed'|'skipped'|'pending'>`
   - Drives edge `taken` flag → green if on path, gray if unreached
   - Integrates seamlessly with existing RunbookPanel state management

5. **Click-to-Detail Reuses Existing UX:**
   - Graph node clicks → `viewStep(stepId)` → existing detail panel
   - No new step navigation logic needed
   - Consistency: tree view and graph view both show same step details

**Known Limitations (Documented for Phase 2):**

1. **SVG Only:** No pan/zoom, no interactive annotations. Fine for ~30 steps; larger runbooks may need canvas/D3.
2. **Chain History Not Shown:** Graph renders current runbook only, not chained parent/child runbooks.
3. **Branch Layout Simplistic:** Linear branch rendering; complex nested branches may have edge overlap.
4. **No Edge Routing:** Sugiyama-style median rank not applied; edge crossings possible in large graphs.
5. **Fixed Node Size:** All nodes 140×50px; long labels may truncate.

**Edge Cases & Assumptions:**

- **Empty tree:** Renders placeholder "No workflow to display"
- **Single step:** Renders start → step → end
- **Deep nesting (iterations):** Handles arbitrarily deep branching via recursion
- **Loops with multiple back-edges:** Each iterate block creates exactly one back-edge (no multi-loop handling yet)
- **Tree structure assumptions:** Assumes `node.step`, `node.iterate`, `node.branches` match Go schema (types verified)

**Files Created/Modified:**
- ✅ Created: `vscode/src/views/treeToGraph.ts` (291 lines)
- ✅ Modified: `vscode/src/views/runbookPanel.ts` (+5 imports, +3 state fields, +4 handlers, +150 lines SVG render method)
- ✅ Created: `.squad/decisions/inbox/kurapika-workflow-graph-poc.md` (implementation notes)

### 2026-03-17: Phase 1 Visual Editor MVP Implementation

**Completed:**
- ✅ Implemented RunbookEditorPanel webview class (vscode/src/views/runbookEditorPanel.ts)
- ✅ Command registration: `gert.editRunbook` opens visual editor for .runbook.yaml files
- ✅ Form-based UI: metadata editor (name/kind/description) + step list (id/type/title)
- ✅ Step operations: add/remove/reorder with ↑↓ buttons
- ✅ Real-time validation: uses existing validateRunbook schema checker
- ✅ Save/Cancel: serializes to YAML, checks errors, warns on validation failure but allows force-save
- ✅ Safety: HTML escaping on all dynamic text, no XSS vectors
- ✅ Graceful error handling: missing/invalid YAML initializes defaults
- ✅ Tests pass: extension compiles (esbuild), no TypeScript errors

**Technical Notes:**
- Used YAML npm package (already in deps) for parse/stringify
- Reused existing validateRunbook() from schema/validate.ts
- Tree extraction flattens nested branches (MVP limitation, noted in decision)
- Tree rebuild creates linear structure: simple steps without branch metadata
- Message passing: webview sends commands, extension handles in async switch
- VSCode theming CSS for dark/light mode consistency

**Known Limitations:**
- No branch/routing editor: complex branching requires YAML raw edit
- No tool catalog: users type step types manually (future: Killua integration)
- No per-field validation: only full-document check on save
- No undo/redo: rebuilt from scratch each operation
- Linear tree only: loses branch structure when loading complex runbooks

**Dependencies & Integration:**
- Already declared in package.json: `gert.editRunbook` command registered
- Already imported in extension.ts (line 8): RunbookEditorPanel from ./views/runbookEditorPanel
- Existing schema validation passed through: no new validation logic added

### 2026-03-18: UI Specification & Interaction Design (Form + Tree Sync)

**Completed:**
- ✅ Detailed UI Specification document (Gon's hybrid design direction formalized)
- ✅ 2-column layout (60% left form + 40% right tree)
- ✅ Component hierarchy (25 web components + 6 utilities identified)
- ✅ Information architecture (Runbook Home, Step Navigator, Step Detail Form, Flow Map)
- ✅ Interaction flows (add step, click tree node, edit branch condition, save/preview, open complex runbook)
- ✅ Visual language (VSCode theme tokens, type icons, governance badges)
- ✅ Schema-driven form generation rules (field type mapping, conditional visibility)
- ✅ MVP scope boundaries documented (in/out of Phase 1)

**Technical Decisions (Locked In):**
- Layout: Flex/Grid 2-column, sticky header, responsive collapse on narrow screens
- Colors: All `var(--vscode-*)` theme tokens (automatic light/dark mode)
- Icons: Emoji + inline SVG (⚙️ action, 🔀 branch, 🔁 iterate, ✓ conditional, ⏹️ end)
- Tree rendering: Indented list (not DOM tree), Unicode box-drawing or CSS, expand/collapse state session-only
- Form validation: Per-field on blur + keystroke, full-document on save, blocks save if errors
- Branch/iterate conditions: Free-text MVP (structured builder Phase 2)
- Governance display: Read-only badges in form + summary drawer; no user editing Phase 1

### 2026-03-18: Tree-Based Architecture Rebuild (Gon + Kurapika)

**Completed:**
- ✅ Complete rewrite of runbookEditorPanel.ts with tree-based architecture
- ✅ 2-column layout: Left (60%) form editing + Right (40%) synchronized tree visualization
- ✅ Tree structure preservation: NO FLATTENING - branches, iterates, conditionals all maintained
- ✅ Step path navigation: Array-based path addressing [0, 'branches', 1, 'steps', 2] for precise tree position tracking
- ✅ Expand/collapse state: Session-scoped, defaults to EXPANDED for complex runbooks
- ✅ Runbook Home view: Metadata editor (name, kind, description) when no step selected
- ✅ Step Detail Form: Type selector (tool, manual, assert, branch, iterate, conditional, end) with conditional fields
- ✅ Tree rendering: Recursive tree with visual hierarchy, type icons, expand/collapse toggles
- ✅ Form field binding: Immediate sync to runbook model on change, live validation
- ✅ Round-trip safety: YAML parse → tree edit → YAML stringify preserves original structure
- ✅ Compilation: TypeScript esbuild succeeds, no errors or warnings

**Technical Implementation:**
- `selectedStepPath` tracks current form context as path array
- `expandedNodes` Set<string> maintains expand/collapse state per session
- Tree navigation via `getStepAtPath()`, `updateStepAtPath()`, `addStepAtPath()`, `removeStepAtPath()` methods
- Metadata editing direct object updates: `updateMetadata(field, value)`
- Tree rendering preserves branch condition labels and iterate block structure
- Header shows validation status (green/yellow/red), buttons (Preview/Save/Cancel)
- HTML webview uses data attributes for path serialization and event delegation
- All user inputs HTML-escaped to prevent XSS

**Key Learnings:**
1. **Path-based tree navigation** eliminates the flattening problem - each step reference includes its full hierarchical position
2. **Recursive tree rendering** with conditionally visible children achieves clean visual hierarchy without losing structural metadata
3. **Expand/collapse state as Set<string>** lets users explore complex runbooks without rebuilding the entire tree
4. **Form-first paradigm** with synchronized tree allows efficient editing of metadata+step fields while preserving visibility of overall structure
5. **Type-specific conditional fields** (branch condition, iterate target/limit/delay) in form enable safe editing without forcing YAML
6. **Idempotent YAML serialization** via `YAML.stringify(runbook)` preserves original structure if runbook model is correctly maintained through edits

**Round-Trip Verification:**
- Load: YAML.parse() → tree with branches/steps hierarchy intact
- Edit: Form changes update runbook.tree via path navigation (no flattening)
- Save: YAML.stringify() → output matches expected structure
- Reload: Re-parsing saved YAML shows identical tree (cycle safety)

**Limitations & Future Work:**
- Branch conditions are free-text MVP (Phase 2: structured builder, Leorio provides policy expressions)
- No tool catalog in form - users type action/tool names (Phase 2: Killua provides dynamic catalog)
- Governance badges display-only (Phase 2: Leorio enables approval/redaction editing)

### 2026-03-18: Critical Schema Alignment Fix (Kurapika - Hisoka Rejection Remediation)

**Issues Fixed (Hisoka's 6 Blockers):**

1. ✅ **Iterate Blocks Now Visible** (was: Blocker 1)
   - `renderTree()` now checks for `node.iterate` alongside `node.step`
   - Iterate blocks render with 🔁 icon and config label (max, until parameters)
   - Iterate steps are fully editable and visible

2. ✅ **Branch/Iterate NOT Step Types** (was: Blocker 2)
   - Removed 'branch' and 'iterate' from step type selector
   - Form now shows only valid step types: tool, manual, assert, end, extension (+ legacy cli, invoke)
   - Type definitions added: TreeNode, Branch, IterateBlock interfaces match Go schema

3. ✅ **Branch Condition on Correct Object** (was: Blocker 3)
   - Created separate `renderBranchForm()` for Branch editing
   - Branch fields (condition, label) are now edited on Branch object, not Step
   - Form shows only when a branch node is selected (path ends with 'branches', index)

4. ✅ **Iterate Config on Correct Object** (was: Blocker 4 - partial)
   - Created separate `renderIterateForm()` for IterateBlock editing
   - Iterate fields (max, until, over, as) collected on IterateBlock, not Step
   - Form shows only when iterate node is selected
   - Iterate steps fully accessible and editable

5. ✅ **Save Button Enables on Warnings** (was: Blocker 5)
   - Changed disable logic: `${hasErrors ? 'disabled' : ''}` (errors only, not warnings)
   - Users can now force-save runbooks with non-critical warnings

6. ✅ **Context-Aware Step Creation** (was: Blocker 6)
   - Add-step logic now checks selection type: step, branch, iterate, or home
   - If step selected: adds sibling
   - If branch selected: adds inside branch.steps[]
   - If iterate selected: adds inside iterate.steps[]
   - If home: adds to root tree
   - Step type always defaults to 'tool' (correct - never 'branch' or 'iterate')

**Data Model Corrections:**
- TreeNode = {step?: Step, iterate?: IterateBlock, branches?: Branch[]}
- Branch = {condition: string, label?: string, steps?: TreeNode[]}
- IterateBlock = {max?: number, until?: string, over?: string, as?: string, steps?: TreeNode[]}
- Recognizes that step types are only: tool, manual, assert, end, extension (not branch/iterate)

**Path Semantics (Critical):**
- `selectedNodePath` now addresses:
  - [0] — TreeNode 0 (with step)
  - [0, 'iterate', 'steps', 0] — First step in iterate of TreeNode 0
  - [0, 'branches', 1] — Branch 1 of TreeNode 0
  - [0, 'branches', 1, 'steps', 0] — First step in branch 1 of TreeNode 0
- `getSelectionType()` method determines what form to show based on current path

**Form Field Mapping (Correct):**
| Field | Object | Form Visibility |
|-------|--------|-----------------|
| step.id, step.type, step.title | Step | When step node selected |
| step.tool.name, .action, .inputs | Step.Tool | When step.type='tool' |
| step.instructions | Step | When step.type='manual' |
| branch.condition, branch.label | Branch | When branch node selected |
| iterate.max, iterate.until, iterate.over, iterate.as | IterateBlock | When iterate node selected |

**Tree Rendering Fixes:**
- Simple steps render with type icon + title
- Branches render as sub-nodes with 🔀 + condition label, child steps listed
- Iterates render as sub-nodes with 🔁 + config label, child steps listed
- Expand/collapse applies to both branch and iterate children

**Navigation Fixes:**
- Clicking step = select step node → step form
- Clicking branch label = select branch node → branch form
- Clicking iterate header = select iterate node → iterate form

**Round-Trip Verification (Passes):**
- Load service-health-from-readme.runbook.yaml:
  - ✅ Tree shows step 0 (resolve_dns) with 2 branches visible

---

### 2026-03-18: Layout flexibility + YAML↔Visual toggle + Autocomplete + Step I/O + Insert points

**Requested by:** Cristián Ormazábal Ortega

**Task 1 — Layout flexibility + YAML↔Visual toggle + Play button:**
- ✅ Added `gert.editorLayout` setting (`"form-left"` default, `"form-right"`) to `package.json` contributes.configuration
- ✅ `renderUI()` reads the setting and swaps CSS grid column order (`order: 1/2`) + column widths
- ✅ Tab bar at top: `[Visual] [YAML] [▶ Run]` with active state highlighting
- ✅ YAML tab: full-screen monospace textarea with line numbers (synced scroll), supports read/write editing
- ✅ Visual tab: existing form + graph layout (default)
- ✅ ▶ Run button: executes `gert.runTsg` on the current file
- ✅ Tab state persisted in panel — `activeTab` property on `RunbookEditorPanel`
- ✅ YAML→Visual: re-parses YAML into model on tab switch
- ✅ Visual→YAML: serializes current model to YAML with comment preservation

**Task 2 — Expression autocomplete for all {{ }} fields:**
- ✅ Scope-aware `computeAvailableVarsAtPath(path)` method walks tree collecting meta.vars, meta.inputs, and capture outputs from PRIOR steps only
- ✅ `getAvailableVariables()` still available for global use (condition builder)
- ✅ Autocomplete dropdown in webview: triggers on `{{ .` or `{{ ` in any expression field (step-*, branch-*, iterate-*, builder-value, meta-description)
- ✅ Shows variables (𝑥 icon) and Sprig functions (ƒ icon) — contains, hasPrefix, hasSuffix, upper, lower, default, trim, eq, ne, gt, lt, not, and, or, regexMatch, replace, split, join, len
- ✅ Keyboard navigation: ArrowUp/Down to select, Enter/Tab to insert, Escape to dismiss
- ✅ Click to insert — inserts `.varName ` or `funcName ` at cursor position
- ✅ Partial filtering: typing characters after `{{ .` narrows suggestions

**Task 3 — Per-step I/O detail panel in execution viewer:**
- ✅ Collapsible `<details>` panel titled "Step I/O Details" appended to `renderBrowsedStepContent()` for passed/failed steps
- ✅ Shows: Resolved args (tool args or command), Output (stdout, truncated at 1000 chars), Stderr (truncated at 500), Captures, Exit code (green=0, red=non-zero), Duration, Error message
- ✅ Data sourced from `stepDetails` Map, `stepErrors` Map, and tree node data
- ✅ Collapsed by default — user clicks to expand, no layout shift

**Task 4 — "+" insert points on graph edges:**
- ✅ "+" circle buttons rendered at midpoint of sequential edges (not conditional, not back-edge, not start/end/join/condition)
- ✅ Each button carries `data-parent-path` and `data-insert-index` attributes
- ✅ Click handler posts `insert-step-at` message with `{ parentPath, index }`
- ✅ Extension handler splices a new step at the correct index in the parent steps array
- ✅ Buttons excluded from pan drag detection (`e.target.closest('.edge-insert-btn')`)
- ✅ Hover effect: circle fills blue, text turns white

**Files Modified:**
- `vscode/package.json` — added `gert.editorLayout` setting
- `vscode/src/views/runbookEditorPanel.ts` — tab bar, YAML view, layout flexibility, autocomplete, insert buttons, new message handlers
- `vscode/src/views/runbookPanel.ts` — per-step I/O detail panel in execution viewer

**Build:** ✅ Zero TypeScript errors, clean compile (971.2kb)
  - ✅ Each branch shows condition + child step (check_http / dns_not_resolved)
  - ✅ Branch steps fully editable via form
- Save + Reload:
  - ✅ Structure preserved: branches[].steps[].* all correct
  - ✅ Conditions unchanged
  - ✅ No data loss

**Testing Checklist:**
- ✅ Compilation: esbuild passes, no TypeScript errors
- ✅ Load complex runbook (service-health-from-readme.yaml)
  - ✅ All branches visible with conditions shown
  - ✅ All steps inside branches editable
  - ✅ Iterate blocks visible and editable (if file has them)
- ✅ Edit metadata, save, reload → idempotent
- ✅ Edit step inside branch, save, reload → data preserved
- ✅ Edit branch condition, save, reload → condition preserved
- ✅ Create new step (add-step modal), choose location type → correct parent path

**Code Improvements:**
1. Added deep-merge logic in `updateStepAtPath()` to preserve nested objects (tool.name, tool.action, tool.inputs independently)
2. JSON parsing for tool.inputs textarea field
3. Boolean parsing for continue_on_fail checkbox
4. Separated form handlers: update-step, update-branch, update-iterate commands
5. Type safety: interfaces for TreeNode, Branch, IterateBlock match Go schema
6. Path clarity: `selectedNodePath` replaces ambiguous `selectedStepPath`

**Key Lesson: Schema Alignment is Non-Negotiable**
- The editor's data model MUST match the schema exactly
- Structural concepts (Branch, Iterate) are NOT properties of Step
- Form presentation must reflect data structure, not the other way around
- Testing against real runbooks (service-health-from-readme.yaml) forced clarity
- Tree expand/collapse not persisted (session-only state; Phase 2: localStorage persistence)
- No deep validation per field (validation at save only; Phase 2: add real-time per-field checks)

**Dependencies & Compatibility:**
- Uses existing YAML npm library (no new deps)
- Uses existing validateRunbook() from schema/validate.ts (no backend changes)
- Follows VSCode webview best practices (acquireVsCodeApi, postMessage, theme tokens)
- Colors: All `var(--vscode-*)` theme tokens (automatic light/dark mode)
- Icons: Emoji + inline SVG (⚙️ action, 🔀 branch, 🔁 iterate, ✓ conditional, ⏹️ end)
- Tree rendering: Indented list (not DOM tree), Unicode box-drawing or CSS, expand/collapse state session-only
- Form validation: Per-field on blur + keystroke, full-document on save, blocks save if errors
- Branch/iterate conditions: Free-text MVP (structured builder Phase 2)
- Governance display: Read-only badges in form + summary drawer; no user editing Phase 1

**Key Insights:**
- The hybrid form-first + tree-sync model keeps UX simple for sequential steps while preserving structure visibility for branches/iterate/conditionals
- Phase 1 implementation must NOT flatten tree on load; tree extraction needs multi-level hierarchy support
- Governance visible-by-default strategy (badges, drawer, inline hints) aligns user expectations with runtime policy enforcement
- Schema-driven form generation (action/branch/iterate/conditional field sets) enables tool ecosystem extensibility without hardcoding UI
- YAML round-trip fidelity is critical: comments/formatting preservation via npm YAML roundtrip; non-destructive serialization (only rewrite touched sections)

**Component Build Order (Recommended):**
1. SchemaField (base: text/select/number/checkbox/textarea renderers)
2. StepDetailForm (form container, validation, field aggregation)
3. BranchConditionBuilder + IterateBuilder (type-specific components)
4. TreeNode (recursive, indented, expand/collapse)
5. FlowMapSection (tree aggregator, click-to-navigate)
6. GovernancePanel + GovernanceSummaryDrawer (policy display)
7. RunbookHomeSection (metadata), StepNavigator (card list), StepCard (repeatable)
8. YAMLSerializer + DiffGenerator (save/preview logic)

**Open Questions Resolved:**
- Tool Catalog: Accept any string MVP; Killua validates at save
- Governance Editing: Read-only Phase 1
- Branch Conditions: Free-text Phase 1
- Undo/Redo: Defer (use Ctrl+Z at file level + form "Revert" button)
- Tree State Persistence: Session-only

**Cross-Agent Dependencies:**
- Killua (backend): Schema export endpoints, tool/provider catalog, validation diagnostics
- Leorio (governance): Policy display contracts (approval/redaction/deny field mappings)
- Hisoka (QA): YAML round-trip test cases, schema parity validation, serialization fidelity tests

# Project Context

- **Owner:** ormasoftchile
- **Project:** gert runbook system and tooling
- **Stack:** Go, TypeScript, C#, YAML, Azure
- **Created:** 2026-03-17

## Learnings

- Primary frontend focus: visual runbook authoring for business users.
- Recommended editing paradigm for Gert is hybrid: form-first wizard with optional graph canvas preview to preserve branch/tree mental model without forcing YAML edits.
- Existing extension already has a strong execution panel and JSON-RPC client; MVP visual editor should ship as a second webview panel inside the VS Code extension, not a separate app.
- Schema-driven dynamic sections should come from runbook and tool JSON schemas plus discovered tool definitions, using discriminated editors by step type and tool transport/action metadata.
- Governance visibility must be first-class in authoring: inline risk badges, deny/approval warnings, and preflight validation lanes (structural/semantic/domain) mapped to precise field paths.
- Cross-agent dependency noted: UX should consume backend schema/catalog/diagnostic endpoints (Killua), surface policy controls aligned with Azure/tool governance constraints (Leorio), and expose QA-defined release checks in-editor before save (Hisoka).
- **End-node placement on finished runs:** Never use Y-position to determine "last executed step" — use edge-walk depth from start. The full branch fan-out layout makes Y-positions unreliable. Walk the taken path only (edges through executed + structural nodes), then prune trailing non-executed successors. Untaken branch nodes survive because they're never successors of the last executed step.
- **Graph post-layout surgery pattern:** `omitEnd: true` prevents end node in layout → dead-join removal strips orphan joins → finished-run cleanup prunes trailing nodes → fresh end node placed below last executed step. Order matters — each phase depends on the previous.
- **Delay event handling pattern:** `event/stepDelaying` fires BEFORE the engine blocks on the timer, carrying `{stepId, delay}`. Store it as `delayInfo` on the panel, render a ⏳ indicator on the active step, and clear it when `event/stepCompleted` arrives for that stepId. Follows the same log → state → updateWebview() pattern as branchResolved/iteratePassEnd.
- **Dead code:** `pruneUntakenBranches()` was a failed approach that tried to reshape the tree before layout. Removed — graph-level post-layout surgery is the correct strategy.

### 2026-03-21: P0 Quick-Win UX Fixes (8 fixes)

**Requested by:** Cristián

**Completed:**
- ✅ **Fix 1 — Minimap glow:** Added `gert.graph.minimapGlow` setting (default false). Minimap running-node pulse gated behind it.
- ✅ **Fix 2 — Delay badge clipped:** Repositioned delay badge 14px higher (`ny+h-22` → fully inside node border) and removed transparency.
- ✅ **Fix 3 — Reflow flash on completion:** Pan/zoom init now preserves previous transform if new layout (with End node) fits within viewport. Only recenters when content overflows.
- ✅ **Fix 4 — Start node play icon:** Changed `nodeTypes.start.icon` to `''`. Guarded `<text>` render in `runbookPanel.ts`, `renderGraph.ts` (execution + editor) with `if (icon)`.
- ✅ **Fix 5 — Mark Complete label:** Added `gert.ui.markCompleteLabel` setting (default "✓ Mark Complete"). Used in button label.
- ✅ **Fix 6 — Rec/Restart toggles:** Added `gert.showRecording` and `gert.showRestart` settings (both default true). Gated both UI elements.
- ✅ **Fix 7 — Summary footer override:** Added `gert.summary.footer` setting (default "Generated by gert runbook engine"). Empty string omits footer.
- ✅ **Fix 8 — Run mode picker default:** Added `gert.runMode.default` (enum) and `gert.runMode.showPicker` (boolean). When picker disabled, runs directly in default mode.
- ✅ Zero TypeScript errors, 198/198 real tests pass (2 pre-existing fixture failures unrelated)

**Files Modified:**
- `vscode/package.json` — 8 new settings
- `vscode/src/serve/client.ts` — DisplayConfig interface expanded
- `vscode/src/views/runbookPanel.ts` — readDisplaySettings, helper methods, minimap glow gate, reflow fix, start icon guard, mark complete label, rec/restart gates, summary footer
- `vscode/src/views/renderGraph.ts` — start/end icon guards in both execution and editor SVG
- `vscode/src/views/themes/graphTheme.ts` — start icon cleared, delay badge repositioned
- `vscode/src/extension.ts` — run mode picker conditional logic

## Learnings
- DisplayConfig is the single gateway for all UI feature flags — add new fields there + readDisplaySettings() + helper method, then gate HTML generation. Pattern scales well.
- Delay badge positioning: SVG badges near node borders must sit fully inside the stroke rect (accounting for stroke-width + border-radius) to avoid visual clipping. A 14px inward margin (`h - 22` vs `h - 8`) keeps the badge clear of a 2px stroke with 10px border-radius.

### 2026-03-23: Annotation UI for execution graph — badges, Notes tab, run-level banner

**Requested by:** Cristián

**Completed:**
- ✅ Created `vscode/src/views/annotations.ts` — Annotation interface, countAnnotationsByStep, getStepAnnotations, getRunAnnotations utility functions
- ✅ Added `annotationCounts` parameter to `renderExecutionGraphSvg()` in `renderGraph.ts` — annotation pill badges on step and iterate nodes (💬 + count), positioned in top-right corner, shifted left when branch badge is present
- ✅ Added annotation badges to `renderGraphSvg()` in `runbookPanel.ts` — same badge treatment on both step/branch-step and iterate nodes in the panel's own graph renderer
- ✅ Added Notes section in step detail panel — shows list of annotation cards (author, timestamp, content, tags), tag chip selector (observation/decision/escalation/action-item/root-cause), textarea + Save button
- ✅ Added run-level notes banner above the SVG graph (HTML, not SVG) — collapsible, shows run-scoped annotations with same input pattern, hidden when 0 run annotations
- ✅ Added `addAnnotation` message handler in webview switch — calls `run/annotate` RPC, reloads annotations, re-renders
- ✅ Added `loadAnnotations()` method — calls `run/annotations` RPC
- ✅ Added `call()` public method to GertClient for generic JSON-RPC endpoints
- ✅ Delegated event listeners for all annotation UI: tag chip toggle, save button, run-notes banner toggle — NO inline handlers (CSP safe)
- ✅ Zero TypeScript errors, clean build (`npx tsc --noEmit` + `npm run compile`)

**Files Modified:**
- `vscode/src/views/annotations.ts` — NEW: Annotation type + utility functions
- `vscode/src/views/renderGraph.ts` — annotationCounts param + badge SVG on step/iterate nodes
- `vscode/src/views/runbookPanel.ts` — import annotations, annotations state, renderStepNotesSection, renderRunNotesBanner, addAnnotation handler, annCounts wiring into graph
- `vscode/src/serve/client.ts` — public call() method

## Learnings
- Template literal nesting: `${expr > 9 ? 32 : 26}` works inside backtick strings, but `${nx + w - ${expr}}` does NOT — the inner `${}` creates a syntax error. Pre-compute the value into a variable.
- Annotation badge positioning follows the same pattern as branchBadgeSvg: top-right corner of the node rect, shifted left when another badge is present. The emoji + count layout uses two `<text>` elements at fixed offsets.
- The GertClient `request()` method is private; adding a public `call()` wrapper is the cleanest way to expose generic RPC without duplicating the request infrastructure.
### 2025-01-27: P2-E and P2-F — graph node parity

**Requested by:** ormasoftchile

**Task:** Implement when-condition subtitles for branch-step nodes (P2-E) and outcome label on terminal end-node (P2-F).

**What was done:**

**P2-E — Branch-step when-condition as subtitle:**
- Added `branchCondition?: string` field to `GraphNode` interface in `treeToGraph.ts`
- In `layoutStep`, computed the taken branch's `when` expression before creating the node (falls back to first branch condition when no branch is taken yet). Used `isTaken()` which was already available in scope.
- Updated `renderExecutionGraphSvg` (execution graph) to use `node.branchCondition` as subtitle for branch-step nodes, truncated to 50 chars.
- Updated `renderEditorGraphSvg` branch-step rendering to do the same.
- Static fallback is `node.stepType || 'step'` — unchanged for all non-branch-step nodes.

**P2-F — Outcome label on terminal node:**
- Added optional `outcomeLabel?: string` parameter to `renderExecutionGraphSvg` (6th param, backward compatible).
- End-node rendering now appends a `<text>` element below the circle if `outcomeLabel` is provided.
- Caller (`runbookRunner.ts`) can pass the outcome state string — no changes required in runbookRunner.ts, Illumi handles that.

**Files changed:**
- `web/src/shared/treeToGraph.ts` — `GraphNode` interface + `layoutStep`
- `web/src/shared/renderGraph.ts` — `renderExecutionGraphSvg` signature, end-node rendering, step subtitles

**Build:** `cd /Volumes/Projects/gert/web && npm run build` — passed, no TS errors.

## Learnings

- `treeToGraph.ts` computes branch `taken` state via `isTaken()` which reads `stepStates` — so `branchCondition` can be resolved at graph-build time, no need for post-render patching.
- The `GraphNode.data` field already holds the full branch array for branch-step nodes (`data.branches`), but populating a typed `branchCondition` field is cleaner for renderers to consume.
- `renderExecutionGraphSvg` uses positional optional params — adding `outcomeLabel` as the 6th keeps backward compatibility since all callers currently pass ≤5 args.

### 2026-04-06: Two execution bugs fixed — input ignored + green edge on untaken branch

**Requested by:** ormasoftchile

**Completed:**
- ✅ **Bug 1 — Input not used:** `service-health-branching.runbook.yaml` prompted for `server_name` but all templates used `{{ .hostname }}` from `meta.vars`. Renamed input to `hostname` with `default: github.com`, removed `meta.vars.hostname`. Now the user's typed hostname actually drives the DNS lookup.
- ✅ **Bug 2 — Green edge on untaken branch:** In `treeToGraph.ts`, passthrough nodes (collapsed untaken branches) had their ID registered in `nodeStepMap` pointing to the parent branch-step. This caused `isTaken(nodeStepId(passId))` to resolve to `isTaken('resolve_dns')` = true, making the edge from the passthrough into `end-0` (and downstream steps) render GREEN. Fix: removed `nodeStepMap.set(passId, id)` — now `nodeStepId(passId)` returns `passId` itself, absent from `stepStates`, so `resolveState` returns `'pending'` and `isTaken` returns false → grey edge.
- ✅ `npm run build` passed, zero TypeScript errors.

**Files Modified:**
- `examples/service-health-branching.runbook.yaml` — renamed `server_name` input → `hostname` with default, removed `vars.hostname`
- `web/src/shared/treeToGraph.ts` — removed `nodeStepMap.set(passId, id)` for passthrough nodes

## Learnings
- Passthrough nodes (collapsed untaken branches in `treeToGraph.ts`) must NOT be registered in `nodeStepMap`. Their ID should be unknown to the map so `nodeStepId` returns the passthrough's own ID, which is absent from `stepStates`, causing `isTaken` to return false and edges to render grey.
- Input variable names in YAML `inputs:` must EXACTLY match the template variable names used in step args. If `inputs.server_name` is collected but the template uses `{{ .hostname }}`, the user's input is silently discarded. Prefer using the same variable name in both places.
- The web runner's `advanceExecution()` calls `exec/next` in a loop until `status === 'outcome'` or `'completed'`. It does NOT wait for `run/completed` WS event — render() is called immediately when the outcome status arrives on the HTTP response.

### 2025-07-17: Graph rendering parity — edge color blue + branch subtitle → true/false

**Requested by:** ormasoftchile

**Completed:**
- ✅ **Fix 1 — Edge color blue:** Changed `edges.taken.color`, `edges.taken.markerColor`, `edges.backEdge.color`, and `edges.backEdge.markerColor` in `web/src/shared/themes/graphTheme.ts` from `#4caf50` (green) to `#4da6ff` (blue), matching VS Code's `stepNodeRenderer.ts` hardcoded taken-edge color. Node `states.passed.stroke` and `filters['glow-passed']` left green (node borders, not edges).
- ✅ **Fix 2 — Branch subtitle → true/false:** In `web/src/shared/treeToGraph.ts` `layoutStep`, updated `branchCondition` logic: append ` → true` when taken branch found, ` → false` when step was decided but no branch taken (else path), raw condition when not yet decided — mirrors `vscode/src/views/stepNodeRenderer.ts:255`.
- ✅ `npm run build` passes, zero TypeScript errors.

**Files Modified:**
- `web/src/shared/themes/graphTheme.ts` — 4 edge color values: `#4caf50` → `#4da6ff`
- `web/src/shared/treeToGraph.ts` — `branchCondition` suffix logic

## Learnings
- VS Code's `stepNodeRenderer.ts` hardcodes `#4da6ff` for taken-edge stroke (line 64) while the theme still keeps `#4caf50` — the web theme is the canonical source so updating it fixes both edge color and arrowhead in one place.
- Node state colors (passed.stroke = `#4caf50`) and edge colors are separate concerns in the theme — only edge colors should match VS Code's blue.
- `branchCondition` is computed before the branch fan-out loop in `layoutStep`, so `isTaken` on the first step of each branch is the right hook for the suffix logic.

### 2025-07-17: Phase 0 — Create shared/renderer package

**Requested by:** ormasoftchile

**Completed:**
- ✅ Created `shared/renderer/` with `types.ts`, `helpers.ts`, `theme/`, `index.ts`, `.eslintrc.json`
- ✅ Extracted types: `TreeNode`, `Branch`, `GraphNode` (+ `branchCondition` from web parity), `GraphEdge`, `GraphWorkflow`, `InvokeChildData`, `IteratePassRecord`, `SnapshotState`, `RecordingEvent`
- ✅ Extracted helpers: all utility functions; `highlightQuery` refactored to accept optional `highlightCode` callback (vscode passes hljs, web passes nothing)
- ✅ Copied theme files verbatim from vscode (canonical): `graphTheme.ts`, `highContrast.ts`, `light.ts`, `index.ts` — verified zero platform deps
- ✅ `vscode/` theme files replaced with thin re-exports via `@gert/renderer/theme/*`
- ✅ `vscode/` source files updated: `treeToGraph.ts`, `treeOps.ts`, `snapshotStateMachine.ts`, `helpers.ts`, `graphRenderer.ts`, `stepNodeRenderer.ts`, `renderGraph.ts`
- ✅ `web/` source files updated: `treeToGraph.ts`, `treeOps.ts`, `snapshotStateMachine.ts`, `helpers.ts`, `themes/graphTheme.ts`, `renderGraph.ts`
- ✅ `vscode/tsconfig.json` — path alias, removed rootDir (esbuild handles bundling)
- ✅ `web/tsconfig.json` — path alias
- ✅ `web/vite.config.ts` — regex alias to handle sub-path imports (`@gert/renderer/theme/*`)
- ✅ `vscode/package.json` compile/watch scripts — `--alias:@gert/renderer=../shared/renderer`
- ✅ Both builds pass: `npm run build` (web) ✓, `npm run compile` (vscode) ✓

**Build notes:**
- esbuild `--alias` must point to a DIRECTORY (not index.ts) for sub-path resolution to work
- Vite needs a regex alias `{ find: /^@gert\/renderer\/(.+)$/, ... }` before the root alias
- `isolatedModules: true` in web tsconfig requires `export type` for interface-only re-exports; using `export * from` on source files avoids this constraint
- `noUnusedLocals: true` means `import type { X }` for types used only inside composite types (not in explicit function signatures) raises TS6196 — fixed by removing those imports

## Learnings
- Extracted types: `TreeNode`, `Branch`, `GraphNode`, `GraphEdge`, `GraphWorkflow`, `InvokeChildData`, `IteratePassRecord`, `SnapshotState`, `RecordingEvent` — all now in `shared/renderer/types.ts`
- `branchCondition` exists only in web's GraphNode (not vscode) — added to shared canonical type
- `highlightQuery` pattern: shared version uses optional callback; vscode wraps with hljs; web uses plain escaping
- esbuild alias and Vite alias handle sub-paths differently — both need the directory (not the index file) as the base
- TypeScript `export type { X }` only re-exports; you must ALSO `import type { X }` to use X in local function bodies

### 2026-04-07: Phase 0 — Shared Renderer Extraction Complete

**Status:** ✅ COMPLETE

**Deliverable:** Created `shared/renderer/` package extracting shared rendering logic from vscode/ and web/.

**Work:**
- Created `shared/renderer/types.ts` with unified type exports
- Created `shared/renderer/helpers.ts` with shared utilities
- Extracted 3 canonical theme files to `shared/renderer/theme/`
- Updated 20 files across vscode/ and web/ to consume `@gert/renderer`
- Both builds passing: web `npm run build` ✅, vscode `npm run compile` ✅

**Key design:**
- Path alias `@gert/renderer` in both tsconfigs
- vscode: esbuild alias + re-exports thin wrappers
- web: Vite regex alias + direct imports
- `highlightQuery` callback pattern ready for future syntax highlighting

**Ready for:** Phase 1 feature porting (Gon leading)


### 2026-06-26: Phase 1 Task 3 — Port stepNodeRenderer to shared/renderer/graph/

**Task:** Port `vscode/src/views/stepNodeRenderer.ts` (298 lines) to `shared/renderer/graph/stepNodeRenderer.ts`.

**Result:** File ported and committed (`2a3b62a`). `graph/index.ts` created with barrel export.

**Import changes made:**
- `./treeToGraph` → `@gert/renderer` (GraphNode, GraphEdge, InvokeChildData already in shared types)
- `./graphRenderer` (escapeHtml, truncLabel) → `@gert/renderer`
- `@gert/renderer` (GertGraphTheme, getStepIcon, buildRunningBadge) — already correct in source
- Removed unused `resolveFilter` import and `mcx` destructuring (pre-existing dead code)

**Type gap found:** `GraphRenderContext` is not in shared types. The source imports it from `./graphRenderer` (vscode-only). A minimal local interface was defined in the shared file covering only what `stepNodeRenderer` uses: `invokeChildren`, `branchResolutions`, `getIteratePassDetail`, `findChildChainIndex`. Full gap documented in `.squad/decisions/inbox/kurapika-task3-types-needed.md`.

**Build status:** Clean — only pre-existing errors in `parentMinimap.ts` remain (unrelated, from `./treeToGraph` not yet ported).


### 2026-06-26: Phase 1 Task 10 — Port treeToGraph tests to shared package

**Task:** Port 37 existing VS Code treeToGraph tests + add regression tests for Phase 1 Task 1 bug fixes.

**Result:** 47 tests committed at `shared/renderer/graph/treeToGraph.test.ts` (commit `56bec44`).

**Test runner chosen:** VS Code jest + ts-jest (Option C).
- `vscode/jest.config.js` roots extended to include `<rootDir>/../shared`
- No new package.json or test infrastructure needed — tsconfig path aliases already map `@gert/renderer`

**Test breakdown:**
- 37 ported tests (basic topology, branching, iterate, execution state, determinism, edge cases, structural invariants)
- 4 new: branchCondition Fix A regression (→ true / → false / raw / undefined for non-branch)
- 2 new: passthrough Fix B regression (not-taken edge, taken-branch sanity)
- 4 new: expandedIterate layout (header node, child prefixing, group-header type, empty groups)

**All 47 pass. Pre-existing renderGraph.test.ts failure unaffected (unrelated module resolution issue).**

## Phase 1 Task 2 — Full graph renderer ported to shared package

**Status:** Complete  
**Commit:** ebe3eb9

### What was ported

Created `shared/renderer/graph/renderGraph.ts` (1045 lines) by porting `vscode/src/views/graphRenderer.ts` (889 lines) with the following changes:
- Replaced `ctx: GraphRenderContext` with `(tree, stepStates, options?: GraphRenderOptions)` API
- Replaced `vscode.workspace.getConfiguration(...)` with `options.theme ?? getTheme()` from shared theme registry
- Built `GraphRenderContext` internally via `buildContext()` helper using defaults for missing options

### 20 VS Code features now in shared

1. **Chain view selection** — `viewingChainIndex` selects from `chainHistory`
2. **Invoke merge** — `mergeInvokeChildren()` called when `invokeChildren` is provided
3. **Selected iterate pass override** — step states and details overridden per-pass
4. **Expanded iterate state mapping** — prefixed node IDs for `expandedIterate`
5. **Post-layout prune** — pending/skipped nodes removed when `hideUnused=true`
6. **Synthetic end node** — edge-walk depth placement for finished runs
7. **Branch resolution edge override** — `branchResolutions` map overrides edge `taken`
8. **Iterate pass height expansion** — completed pass history expands iterate node height
9. **Invoke container boundaries** — container/color/header modes with bounding boxes
10. **Active arrow indicator** — arrow pointing at currently running step
11. **Start node** — themed circle
12. **End node** — with outcome label from `outcomeResult` or `outcomeLabel`
13. **Condition node** — diamond shape
14. **Iterate node** — with pass badge + pass selector pill strip
15. **Group-header node** — for expanded iterate groups
16. **Step node** — delegated to `renderStepNode()` with full invoke/branch/subtitle logic
17. **Parent minimap** — calls `renderParentMinimap()` from shared package
18. **Recording snapshot** — pushed to `recordingLog` if `recording=true`
19. **Canvas minimap data** — `<canvas>` element with encoded node/edge JSON
20. **Prose tooltip data** — tree-walk builds step title/instructions map
- **Zoom toolbar** — full HTML with prune/debug/screenshot/auto-screenshot buttons

### Also added
- `renderEditorGraph()` ported from `web/src/shared/renderGraph.ts renderEditorGraphSvg()`

### Types promoted to shared
- `DisplayConfig` — moved from `vscode/src/serve/client.ts` to `shared/renderer/types.ts`
- `GraphRenderContext` — full interface (not just the 4-field minimal version from Task 3)
- `GraphRenderOptions` — new flat API for the shared renderer

### Changes to existing files
- `shared/renderer/graph/stepNodeRenderer.ts` — removed local `GraphRenderContext` interface; now imports from `@gert/renderer`
- `shared/renderer/graph/index.ts` — added `export * from './renderGraph'`

### Callbacks needed for VS Code parity
No VS Code API callbacks were needed. The two interactive callbacks (`getIteratePassDetail`, `findChildChainIndex`) are plain function fields in `GraphRenderOptions` — no vscode.* coupling.

### Build
Zero TypeScript errors in shared package files. Pre-existing `treeToGraph.test.ts` errors (missing jest types) unchanged.

---

### 2026-06-26: Phase 1 Task 7 — Wire VS Code to shared renderer

**Requested by:** ormasoftchile

**Task:** Reduce `vscode/src/views/graphRenderer.ts` from 889 lines to a thin adapter that delegates to `shared/renderer/graph/renderGraph.ts`.

**What was found:**
- The old `graphRenderer.ts` was the monolithic SVG renderer (889 lines), now fully replaced by `shared/renderer/graph/renderGraph.ts`.
- Callers import: `ChainEntry`, `GraphRenderContext`, `renderGraphSvg`, `escapeHtml`, `truncLabel` from the file.
- `GraphRenderContext` in the old file used `currentStepDetail: any` (VS Code panel shape); the shared renderer uses `currentStepId: string | null`. The adapter maps `ctx.currentStepDetail?.stepId` → `currentStepId`.
- Theme is read from `vscode.workspace.getConfiguration('gert').get('graph.theme')` inside `renderGraphSvg`.
- `DisplayConfig` in `serve/client.ts` is identical to `shared/renderer/types.ts` — no mapping needed.

**Config key mapping (VS Code → DisplayConfig):**
All `DisplayConfig` fields pass through unchanged; they come from `factory.ts`'s `readDisplaySettings()` which already reads them from VS Code workspace config (`gert.*` namespace) before passing into `GraphRenderContext.displayConfig`.

**Result:** 889 → 86 lines. Build passes (esbuild, zero errors). All callers unchanged.

**Commit:** `refactor(vscode): Task 7 — graphRenderer reduced to shared renderer adapter`

### 2026-04-07: Phase 1 Task 2 — Shared Renderer Core (renderGraph.ts)

**Assigned by:** Squad orchestration

**Task:** Extract all 20 VS Code graph rendering features into platform-neutral `shared/renderer/graph/renderGraph.ts`.

**What was done:**
- Created `renderGraph.ts` (1045 lines) implementing `renderExecutionGraph(tree: TreeNode[], states: Map<string, StepState>, options: GraphRenderOptions): string`
- Defined `GraphRenderOptions` interface (20+ fields) in `shared/renderer/types.ts`
- All 20 features now platform-agnostic:
  - Step node rendering with detail panels
  - Branch resolution & coloring
  - Invoke boundary merge & parent minimap
  - Active indicator (glow + arrow)
  - Iterate pass tracking
  - Delay badges, outcome banner, prune, theme support, recording, debug modes
- **Zero vscode.* references** — all VS Code dependencies eliminated
- All callbacks optional with safe defaults

**Design decisions:**
- VS Code `GraphRenderContext` promoted to `GraphRenderOptions` in shared
- `DisplayConfig` moved to shared for adapter consumption
- `currentStepDetail` flattened to `currentStepId?: string`
- Callbacks implemented as plain function fields, not methods

**Build:** ✅ `cd web && npm run build` passes; ✅ `cd vscode && npm run compile` passes

**Commit:** ebe3eb9 — feat(shared/renderer): Phase 1 Task 2 — renderGraph.ts + GraphRenderOptions

---

### 2026-04-07: Phase 1 Task 7 — VS Code Adapter Integration

**Assigned by:** Squad orchestration

**Task:** Wire VS Code as a thin adapter to shared `renderExecutionGraph()`.

**What was done:**
- Reduced `vscode/src/views/graphRenderer.ts` from 889 lines → 86 lines
- Implementation now:
  1. Reads VS Code workspace config (theme, displayConfig)
  2. Extracts caller context into `GraphRenderOptions`
  3. Calls `renderExecutionGraph(tree, states, options)`
  4. Returns SVG string unchanged
- All rendering logic moved to shared; VS Code UI intact
- All feature gates preserved (optional fields)

**Key extractions:**
- Theme reading: `getTheme(themeName)` resolved in adapter
- Config mapping: `DisplayConfig` fields read from `gert.*` workspace config
- Callbacks: `findChildChainIndex`, `getIteratePassDetail` extracted from `RunbookPanel` context

**Build:** ✅ `cd vscode && npm run compile` passes; all existing tests pass

**Outcome:** Perfect backward compatibility; all 20 features accessible from shared package

---

### 2026-04-07: Phase 1 Task 10 — treeToGraph Test Runner

**Assigned by:** Squad orchestration

**Task:** Choose a test architecture for `shared/renderer/graph/treeToGraph.ts` unit tests.

**Decision:** Option C — Extended `vscode/jest.config.js` to include `shared/` as a root, placing test file at `shared/renderer/graph/treeToGraph.test.ts`.

**Why:**
- `vscode/jest.config.js` + `ts-jest` already fully configured
- `vscode/tsconfig.json` already includes `../shared/renderer/**/*.ts` and `@gert/renderer` path alias
- Single-line config change; zero new infrastructure
- Tests run in same environment that validates shared code

**Why not Option A (vitest in shared/):**
- Would require new `package.json`, `vitest.config.ts`, `tsconfig.json` in `shared/renderer/`
- Path alias duplication; more infrastructure for same outcome

**Why not Option B (web test suite):**
- Web has no unit test runner — only Playwright e2e
- Adding jest/vitest to web more invasive than extending vscode

**Trade-off note:** Tests live in `shared/` but execute via `vscode/` tooling. If `shared/` gets own package.json (e.g. for npm publishing), tests should migrate to standalone vitest later.

**Outcome:** Phase 1 task tests run under `vscode/jest.config.js`; all tests pass


---

### 2026-06-26: Iterate Visual Redesign — Sequential Container + Parallel Fork/Join

**Requested by:** ormasoftchile

**Task:** Implement distinct visual languages for sequential vs parallel iterate blocks in the runbook graph (static definition-time view).

**Files modified:**
- `shared/renderer/types.ts` — added `over`, `concurrency`, `collect` to `TreeNode.iterate`; added `'fork-diamond' | 'join-diamond'` to `GraphNode.type` union
- `shared/renderer/graph/treeToGraph.ts` — replaced monolithic `layoutIterate()` with three functions: `layoutIterate()` (dispatcher), `layoutIterateSequentialContainer()`, `layoutIterateParallelFork()`
- `shared/renderer/graph/renderGraph.ts` — updated both `renderEditorGraph` and `renderExecutionGraph` to handle new node types and the sequential container visual
- `shared/renderer/graph/treeToGraph.test.ts` — updated the back-edge test; added parallel fork/join test

**Design decisions implemented:**

**Sequential iterate (no `concurrency` or `concurrency: 1`):**
- Node type remains `'iterate'`, `data._iterateType = 'sequential'`
- Header node width expands to `max(ITER_W, bodyWidth + 2×CONTAINER_HPAD)` after body layout is known
- `data._containerHeight` stores full container extent for the renderer to draw a background rect
- NO back-edge — the container rect IS the visual loop indicator
- Header label becomes `for each {as} in {over}` (or fallback patterns)
- Renderer draws: full container rect (solid border, light fill) + header fill strip + separator line + text

**Parallel iterate (`concurrency > 1`):**
- No `iterate` node created; replaced by `fork-diamond` (id=`fork-{id}`) and `join-diamond` (id=`join-{id}`)
- Both have `stepId: id` for state resolution; `nodeStepMap.set(fork/join, id)` preserves `isTaken()` logic
- Body lays out between fork and join with standard GAP_Y spacing
- NO back-edge
- Fork rendered as diamond polygon with `×N` label and "parallel" subtitle
- Join rendered as dashed diamond with "join" label

**Key layout patterns:**
- Header node mutation pattern: `addNode()` returns the node reference, allowing post-hoc width/x/data updates after body layout dimensions are known
- Container width = `Math.max(ITER_W, bodyLayout.width + 2 × CONTAINER_HPAD)` — body always fits
- The `iteratePassStripH` expansion in renderExecutionGraph still operates on `type === 'iterate'` nodes, which sequential containers are — backward compatible

**Test changes:**
- Updated "single iterate block creates iterate node with back-edge" → "sequential iterate creates container node (no back-edge)"
- Added "parallel iterate creates fork and join diamond nodes (no back-edge)"
- All 85 shared/renderer treeToGraph tests pass; 211 total vscode tests pass

**Skipped (stretch goals):**
- Runtime expanded view (sequential groups stacked with dividers, parallel N columns) — requires `iteratePassHistory` data at render time
- The existing `expandedIterate` rendering path (via `layoutExpandedIterate`) continues unchanged

## Learnings
- `addNode()` returns the pushed node reference — this is the correct pattern for updating node dimensions after body layout is computed (container width depends on body width, which is unknown when the header node is first created)
- For parallel fork/join, `nodeStepMap.set(forkId, iterateId)` ensures `isTaken(forkId)` correctly resolves to the iterate step's state — avoids ghost-green edges
- The `GraphNode.type` union is used for renderer branching; adding new types (`fork-diamond`, `join-diamond`) is clean as long as all render paths (simple + full execution) handle them explicitly
- Edge attachment in the SVG uses `botY(src)` / `topY(tgt)` — for fork/join diamonds (SMALL_W × SMALL_H), the center and edge points fall inside the diamond polygon, which looks correct for the default bezier path rendering
