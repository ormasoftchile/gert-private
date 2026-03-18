### 2026-03-18: Drag-to-reorder is same-parent only

**By:** Kurapika (Frontend Engineer)
**Status:** Implemented

**What:** Step drag-to-reorder in the editor Flow Map is restricted to siblings within the same parent steps array. Cross-branch and cross-iterate dragging is blocked by design — only nodes with matching `data-parent-path` are considered valid drop targets.

**Why:** Moving steps between structural containers (branches, iterates) would require semantic validation (e.g., does the step make sense outside its branch condition?). Same-parent reorder is a safe, lossless operation that covers the most common use case: adjusting execution order within a sequence.

**Risks:** Users may expect to drag steps between branches. A future enhancement could add cross-container moves with confirmation dialogs.
