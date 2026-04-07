# Decision: Shared Renderer Refactor Scope

**Date:** 2026-03-22  
**By:** Gon (Lead Architect)  
**Status:** PROPOSED — awaiting user decision

## What
Extract a shared rendering library at `shared/renderer/` (monorepo internal package) consumed by both VS Code extension and web app. Start with types + helpers + theme (zero-risk), then phase through graph rendering, prose, step panels, state machine, and input forms.

## Why
~4,053 lines in VS Code and ~2,777 in web duplicate rendering logic at ~75% overlap. A change in one panel renderer requires manual duplication to the other. The graph theme system is already 95% portable — the architecture supports sharing with minimal adaptation.

## Key Architecture Decisions
1. **Monorepo internal package** (not npm publish) — TypeScript path aliases in both tsconfig files
2. **Platform adapters via callbacks** — `renderMarkdown`, `resolveImagePath`, `highlightCode` injected by each platform, not conditional imports
3. **CSS variable indirection** — shared CSS uses `--gert-*` vars; VS Code maps to `--vscode-*`, web maps to hardcoded values
4. **ESLint import guard** — shared package cannot import from `vscode` or `web/src/api`

## First PR
Extract types + helpers + theme into `shared/renderer/`. Import path rewiring only, zero logic changes. Proves build integration works with zero risk.

## Risks
- CSS regression from VS Code theme variable handling (mitigated by screenshot tests)
- Event name fragmentation in snapshotStateMachine (mitigated by explicit mapping layer)
- Feature drift between platforms (mitigated by feature flags in options bags)
