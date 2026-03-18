# Orchestration Entry: Kurapika

### 2026-03-18 — Drag-to-reorder steps in editor graph + host-adaptive visualization abstraction (vizAdapter.ts)

| Field | Value |
|-------|-------|
| **Agent routed** | Kurapika (Frontend Engineer) |
| **Why chosen** | Frontend graph rendering owner — drag-to-reorder is a graph interaction feature; vizAdapter abstraction is the host-adaptive visualization layer Kurapika owns |
| **Mode** | `background` |
| **Why this mode** | No hard data dependencies on other agents; graph interaction and adapter refinement are self-contained frontend tasks |
| **Files authorized to read** | `vscode/src/views/treeToGraph.ts`, `vscode/src/views/runbookPanel.ts`, `vscode/src/views/runbookEditorPanel.ts`, `vscode/src/views/vizAdapter.ts` |
| **File(s) agent must produce** | `vscode/src/views/runbookEditorPanel.ts` (drag-to-reorder step interaction in editor graph), `vscode/src/views/vizAdapter.ts` (host-adaptive visualization abstraction refinement) |
| **Outcome** | Pending |
