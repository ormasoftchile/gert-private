# Orchestration Entry: Kurapika

### 2026-03-18 — Layout flexibility + expression autocomplete + per-step I/O detail panel + insert points on graph edges

| Field | Value |
|-------|-------|
| **Agent routed** | Kurapika (Frontend Engineer) |
| **Why chosen** | Frontend engineer owns editor panel layout, interactive graph features, and form-based UX — all four tasks are editor surface work |
| **Mode** | `background` |
| **Why this mode** | Independent of backend changes; operates on editor webview internals |
| **Files authorized to read** | `vscode/src/views/runbookEditorPanel.ts`, `vscode/src/views/runbookPanel.ts`, `vscode/src/views/vizAdapter.ts`, `vscode/src/views/treeToGraph.ts`, `vscode/src/extension.ts` |
| **File(s) agent must produce** | `vscode/src/views/runbookEditorPanel.ts` (layout setting + YAML↔Visual toggle + play button + insert points), `vscode/src/views/runbookPanel.ts` (per-step I/O detail panel) |
| **Outcome** | Pending |
