# Decision: treeToGraph test runner for shared package

**Author:** Kurapika  
**Date:** 2026-06-26  
**Task:** Phase 1 Task 10

## Choice: Option C — Extend vscode/jest to include shared

### What was chosen

Extended `vscode/jest.config.js` `roots` to include `<rootDir>/../shared`, placing the test file at `shared/renderer/graph/treeToGraph.test.ts`.

### Why not Option A (vitest in shared/)

Would require a new `package.json`, `vitest.config.ts`, and `tsconfig.json` inside `shared/renderer/`. The `@gert/renderer` path alias needs to be resolvable by the test runner, which means duplicating tsconfig `paths` config. More infrastructure for the same outcome.

### Why not Option B (web test suite)

The `web/` package has no unit test runner — only Playwright e2e. Adding jest/vitest to `web/` would be more invasive than extending the already-configured vscode jest.

### Why Option C

- `vscode/jest.config.js` + `ts-jest` is already configured and working
- `vscode/tsconfig.json` already includes `../shared/renderer/**/*.ts` and maps `@gert/renderer` to the shared types
- Single-line change to `roots` was sufficient — zero new infrastructure
- Tests run in the same environment that already validates the shared code

### Trade-offs

- Tests live in `shared/` but execute via `vscode/` tooling. Future: if `shared/` gets its own package.json (e.g. for npm publishing), the tests should migrate to a standalone vitest config there.
