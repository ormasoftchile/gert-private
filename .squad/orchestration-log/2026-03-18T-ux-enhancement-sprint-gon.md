# Orchestration Entry: Gon

### 2026-03-18 — Peek YAML + searchable tool catalog panel + scenario browser in editor

| Field | Value |
|-------|-------|
| **Agent routed** | Gon (Lead Architect) |
| **Why chosen** | Architect owns editor panel design, data flow, tool catalog API integration, and scenario/replay data structures — all three tasks require architectural coordination |
| **Mode** | `background` |
| **Why this mode** | Independent of frontend layout work; operates on editor data layer and new panel views |
| **Files authorized to read** | `vscode/src/views/runbookEditorPanel.ts`, `vscode/src/serve/client.ts`, `pkg/replay/scenario.go`, `pkg/replay/replay.go`, `pkg/schema/tool.go`, `vscode/src/extension.ts` |
| **File(s) agent must produce** | `vscode/src/views/runbookEditorPanel.ts` (peek YAML panel, tool catalog panel, scenario browser) |
| **Outcome** | Pending |
