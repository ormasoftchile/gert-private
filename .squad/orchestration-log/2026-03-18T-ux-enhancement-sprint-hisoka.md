# Orchestration Entry: Hisoka

### 2026-03-18 — Query syntax highlighting (TextMate grammars + visual editor inline highlighting)

| Field | Value |
|-------|-------|
| **Agent routed** | Hisoka (QA / Reviewer) |
| **Why chosen** | QA/reviewer owns TextMate grammar correctness and syntax highlighting verification — query highlighting requires grammar injection rules and validation |
| **Mode** | `background` |
| **Why this mode** | Independent of engine and editor panel work; operates on grammar JSON files and editor highlighting internals |
| **Files authorized to read** | `vscode/syntaxes/runbook-injection.json`, `vscode/syntaxes/*.json`, `vscode/package.json`, `vscode/src/views/runbookEditorPanel.ts` |
| **File(s) agent must produce** | `vscode/syntaxes/runbook-injection.json` (query language grammar injections), `vscode/src/views/runbookEditorPanel.ts` (inline query highlighting in visual editor) |
| **Outcome** | Pending |
