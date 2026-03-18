# Orchestration Entry: Gon

### 2026-03-18 — Editor click→form fix + host-adaptive viz + YAML comment preservation

| Field | Value |
|-------|-------|
| **Agent routed** | Gon (Lead Architect) |
| **Why chosen** | Architect owns editor panel, viz abstraction architecture, and YAML fidelity concerns |
| **Mode** | `background` |
| **Why this mode** | Independent of backend and frontend graph work; operates on editor panel and architecture-level abstractions |
| **Files authorized to read** | `vscode/src/views/runbookEditorPanel.ts`, `vscode/src/views/treeToGraph.ts`, `vscode/src/schema/*.ts`, `pkg/schema/*.go` |
| **File(s) agent must produce** | `vscode/src/views/runbookEditorPanel.ts` (click→form fix), viz adapter abstraction files, YAML round-trip comment handling |
| **Outcome** | Pending |
