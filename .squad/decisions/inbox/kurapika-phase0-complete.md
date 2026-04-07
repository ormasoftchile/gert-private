# Phase 0 Complete — shared/renderer package

**Agent:** Kurapika (Frontend Engineer)
**Date:** 2025-07-17
**Status:** ✅ Both builds passing

## What Was Done

Created `shared/renderer/` — a zero-dependency package containing shared rendering logic extracted from both `vscode/` and `web/`.

## New Files Created

| File | Contents |
|------|----------|
| `shared/renderer/types.ts` | `TreeNode`, `Branch`, `GraphNode`, `GraphEdge`, `GraphWorkflow`, `InvokeChildData`, `IteratePassRecord`, `SnapshotState`, `RecordingEvent` |
| `shared/renderer/helpers.ts` | All utility functions; `highlightQuery` accepts optional `highlightCode` callback |
| `shared/renderer/theme/graphTheme.ts` | Copied verbatim from vscode (canonical) |
| `shared/renderer/theme/highContrast.ts` | Copied verbatim from vscode |
| `shared/renderer/theme/light.ts` | Copied verbatim from vscode |
| `shared/renderer/theme/index.ts` | Theme registry with `getTheme()` + `export *` |
| `shared/renderer/index.ts` | Barrel export |
| `shared/renderer/.eslintrc.json` | `no-restricted-imports` guard: cannot import from vscode/ or web/ |

## Files Modified

### vscode/
- `tsconfig.json` — path alias `@gert/renderer`, removed `rootDir`
- `package.json` — esbuild `--alias:@gert/renderer=../shared/renderer`
- `src/views/treeToGraph.ts` — types replaced with `import/export type from @gert/renderer`
- `src/views/treeOps.ts` — `InvokeChildData` now from `@gert/renderer`
- `src/views/snapshotStateMachine.ts` — types now from `@gert/renderer`
- `src/views/helpers.ts` — re-exports from `@gert/renderer`; local hljs `highlightQuery`
- `src/views/graphRenderer.ts` — theme imports → `@gert/renderer`
- `src/views/stepNodeRenderer.ts` — theme imports → `@gert/renderer`
- `src/views/renderGraph.ts` — theme imports → `@gert/renderer`
- `src/views/themes/graphTheme.ts` — thin re-export from `@gert/renderer/theme/graphTheme`
- `src/views/themes/highContrast.ts` — thin re-export
- `src/views/themes/light.ts` — thin re-export
- `src/views/themes/index.ts` — thin re-export

### web/
- `tsconfig.json` — path alias `@gert/renderer`
- `vite.config.ts` — regex alias for sub-paths + root alias
- `src/shared/treeToGraph.ts` — types replaced with `import/export type from @gert/renderer`
- `src/shared/treeOps.ts` — re-exports `TreeNode`, `InvokeChildData` from `@gert/renderer`
- `src/shared/snapshotStateMachine.ts` — types now from `@gert/renderer`
- `src/shared/helpers.ts` — re-exports from `@gert/renderer`
- `src/shared/themes/graphTheme.ts` — thin re-export from `@gert/renderer/theme/graphTheme`
- `src/shared/renderGraph.ts` — theme imports → `@gert/renderer`

## Build Status

- **web:** `npm run build` ✅ (tsc + vite, 94 modules, zero errors)
- **vscode:** `npm run compile` ✅ (esbuild, 1.1mb bundle)
- **vscode tsc type check:** 5 pre-existing errors in `extension.ts` (`.label` on `string`) — unrelated to this PR

## Notes for Phase 1

- `GraphNode.branchCondition` was added to shared types (web had it, vscode didn't — merged)
- `highlightQuery` callback pattern ready for any renderer that wants syntax highlighting
- The 5 pre-existing TS errors in `extension.ts` should be fixed separately
