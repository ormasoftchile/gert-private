# Orchestration Entry: Kurapika

### 2026-03-18 — Live event wiring + pan/zoom for both panels

| Field | Value |
|-------|-------|
| **Agent routed** | Kurapika (Frontend Engineer) |
| **Why chosen** | Frontend graph rendering owner — event wiring into SVG graph and pan/zoom are both visual layer concerns |
| **Mode** | `background` |
| **Why this mode** | No hard data dependencies on other agents; graph rendering is self-contained |
| **Files authorized to read** | `vscode/src/views/treeToGraph.ts`, `vscode/src/views/runbookPanel.ts`, `vscode/src/views/runbookEditorPanel.ts`, `vscode/src/serve/*.ts` |
| **File(s) agent must produce** | `vscode/src/views/runbookPanel.ts` (event listeners + pan/zoom), `vscode/src/views/runbookEditorPanel.ts` (pan/zoom), `vscode/src/views/treeToGraph.ts` (event-driven node updates) |
| **Outcome** | Pending |
