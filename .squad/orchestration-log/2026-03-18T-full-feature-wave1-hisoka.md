# Orchestration Entry: Hisoka

### 2026-03-18 — Webview integration tests + CI pipeline (GitHub Actions)

| Field | Value |
|-------|-------|
| **Agent routed** | Hisoka (QA Reviewer) |
| **Why chosen** | QA owns test strategy and CI pipeline setup; webview integration tests extend prior test suite work |
| **Mode** | `background` |
| **Why this mode** | Test infrastructure is independent of feature work; CI pipeline is additive |
| **Files authorized to read** | `vscode/src/views/*.ts`, `vscode/src/views/__tests__/*`, `vscode/jest.config.js`, `vscode/package.json`, `.github/workflows/*` |
| **File(s) agent must produce** | Webview integration tests in `vscode/src/views/__tests__/`, `.github/workflows/vscode.yml`, `.github/workflows/go.yml` |
| **Outcome** | Pending |
