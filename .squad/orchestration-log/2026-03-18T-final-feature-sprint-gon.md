# Orchestration Entry: Gon

### 2026-03-18 — Chain visualization for invoke steps + YAML comments/formatting preservation on save

| Field | Value |
|-------|-------|
| **Agent routed** | Gon (Lead Architect) |
| **Why chosen** | Architect owns editor panel design, data flow, and serialization strategy — invoke chain visualization requires architectural alignment with step resolution; YAML comment preservation extends Gon's prior comment-preserving save work |
| **Mode** | `background` |
| **Why this mode** | Independent of graph interaction and adapter work; operates on editor rendering and serialization internals |
| **Files authorized to read** | `vscode/src/views/runbookEditorPanel.ts`, `vscode/src/views/treeToGraph.ts`, `vscode/src/schema/*.ts`, `pkg/schema/*.go` |
| **File(s) agent must produce** | `vscode/src/views/runbookEditorPanel.ts` (invoke step chain visualization), `vscode/src/views/treeToGraph.ts` (invoke chain graph nodes/edges) |
| **Outcome** | Pending |
