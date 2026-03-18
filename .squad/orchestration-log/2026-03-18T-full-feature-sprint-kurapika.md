# Orchestration Entry: Kurapika

### 2026-03-18 — Live event wiring (branchResolved/iteratePassEnd into graph) + pan/zoom for execution viewer and editor

| Field | Value |
|-------|-------|
| **Agent routed** | Kurapika (Frontend Engineer) |
| **Why chosen** | Frontend graph rendering owner — event wiring for `branchResolved`/`iteratePassEnd` into SVG graph and pan/zoom are visual layer concerns in execution viewer and editor panels |
| **Mode** | `background` |
| **Why this mode** | No hard data dependencies on other agents; graph event wiring and pan/zoom are self-contained frontend tasks |
| **Files authorized to read** | `vscode/src/views/treeToGraph.ts`, `vscode/src/views/runbookPanel.ts`, `vscode/src/views/runbookEditorPanel.ts`, `vscode/src/serve/*.ts` |
| **File(s) agent must produce** | `vscode/src/views/runbookPanel.ts` (event listeners for branchResolved/iteratePassEnd + pan/zoom), `vscode/src/views/runbookEditorPanel.ts` (pan/zoom), `vscode/src/views/treeToGraph.ts` (event-driven node state updates) |
| **Outcome** | Pending |
