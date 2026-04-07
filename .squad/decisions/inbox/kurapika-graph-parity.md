# Decision: Graph node parity — when-conditions and outcome label

**Date:** 2025-01-27
**By:** Kurapika (Frontend Engineer)
**Implements:** P2-E, P2-F

---

## P2-E: branchCondition resolved at graph-build time

**Decision:** Populate `GraphNode.branchCondition` inside `treeToGraph.ts > layoutStep` rather than computing it in the renderer.

**Why:** `layoutStep` already has `isTaken()` in scope and knows which branch is resolved. Keeping this logic in the graph model layer means both renderers (execution + editor) get the value for free. Renderers stay thin.

**Tradeoff:** The fallback (first branch condition shown before any branch is taken) is an editorial choice — it gives the editor graph a hint of what the condition is even with no execution state. Alternative was showing `stepType` — we prefer the condition since it communicates intent.

---

## P2-F: outcomeLabel as an optional 6th parameter

**Decision:** Add `outcomeLabel?: string` as a 6th optional parameter to `renderExecutionGraphSvg` rather than a separate options object or post-render DOM manipulation.

**Why:** Simplest backward-compatible extension. All existing call sites pass ≤5 args and continue to work. The outcome label is purely visual and belongs at render time.

**Integration note:** `runbookRunner.ts` (Illumi's domain) should pass `outcomeResult?.state` as the 6th argument when calling `renderExecutionGraphSvg`. The renderer is ready — no coordination change needed on this side.

---

## No changes to renderEditorGraphSvg end node

**Decision:** The editor graph end node was not updated to show outcomeLabel.

**Why:** The editor graph has no execution state — an outcomeLabel would always be empty. The existing end-node appearance is sufficient for editing context.
