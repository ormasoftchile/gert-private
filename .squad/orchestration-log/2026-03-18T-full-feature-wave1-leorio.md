# Orchestration Entry: Leorio

### 2026-03-18 — Governance display contracts — governance/evaluate endpoint + TypeScript types

| Field | Value |
|-------|-------|
| **Agent routed** | Leorio (Integrations Engineer) |
| **Why chosen** | Governance domain owner — endpoint design and type contract generation are integration concerns |
| **Mode** | `background` |
| **Why this mode** | Independent of graph rendering and editor work; governance layer has its own data model |
| **Files authorized to read** | `pkg/governance/*.go`, `ext/serve/pkg/serve/serve.go`, `vscode/src/serve/*.ts` |
| **File(s) agent must produce** | `ext/serve/pkg/serve/serve.go` (governance/evaluate RPC endpoint), `vscode/src/serve/governanceTypes.ts` or equivalent TypeScript types |
| **Outcome** | Pending |
