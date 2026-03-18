# Orchestration Entry: Hisoka

### 2026-03-18 — Webview integration tests + GitHub Actions CI pipeline + Go test verification

| Field | Value |
|-------|-------|
| **Agent routed** | Hisoka (QA Reviewer) |
| **Why chosen** | QA owns test strategy, CI pipeline setup, and cross-stack verification; webview integration tests extend prior contract test suite |
| **Mode** | `background` |
| **Why this mode** | Test infrastructure and CI pipeline are independent of feature work; additive and non-blocking |
| **Files authorized to read** | `vscode/src/views/*.ts`, `vscode/src/views/__tests__/*`, `vscode/jest.config.js`, `vscode/package.json`, `.github/workflows/*`, `pkg/**/*_test.go`, `ext/serve/pkg/**/*_test.go` |
| **File(s) agent must produce** | Webview integration tests in `vscode/src/views/__tests__/`, `.github/workflows/vscode.yml`, `.github/workflows/go.yml`, Go test verification report |
| **Outcome** | Pending |
