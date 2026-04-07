# Phase 1 Shared Renderer Verification — Knov Report

**Date:** 2026-04-07  
**Verified by:** Knov (Playwright & E2E Testing Specialist)  
**Commit:** `60ae641`

---

## Overall Verdict: ✅ PASS (with one bug found and fixed)

---

## Step-by-Step Results

### Step 1: Build State

| Target | Result |
|--------|--------|
| `web` build (`tsc && vite build`) | ✅ PASS (after tsconfig fix) |
| `vscode` build (`esbuild`) | ✅ PASS — 1.1MB clean bundle |

**Bug Found:** `web/tsconfig.json` included `../shared/renderer` but not excluding `*.test.ts` files. The `treeToGraph.test.ts` (which uses Jest globals) was being type-checked by the web's tsconfig, failing with `Cannot find name 'describe'` etc.

**Fix Applied:** Added to `web/tsconfig.json`:
```json
"exclude": ["../shared/renderer/**/*.test.ts", "../shared/renderer/**/*.spec.ts"]
```

---

### Step 2: Existing Playwright Tests

**Result: 13 passed → 17 passed (4 new tests added) — no regressions**

Same 4 pre-existing failures remain (confirmed pre-existing by checking before/after):
- `knov-edge-color-verify.spec.ts` — DNS test (requires live DNS infrastructure)
- `knov-screenshot.spec.ts` — screenshot capture (pre-existing)
- `tool-catalog.spec.ts` (×2) — tool list timeout (pre-existing gert server config issue)

---

### Step 3: Verification Spec

**Result: 4/4 PASS** — `web/tests/specs/knov-phase1-verify.spec.ts`

| Test | Result |
|------|--------|
| graph renders SVG nodes from shared renderer | ✅ PASS |
| prune toggle button exists in toolbar | ✅ PASS |
| deleted files confirmed gone from web/src/shared/ | ✅ PASS |
| shared/renderer has zero vscode imports | ✅ PASS |

**Screenshot:** `web/tests/screenshots/phase1-graph.png`

---

### Step 5: treeToGraph Unit Tests

**Result: 84 tests PASS** (37 from `vscode/src/views/treeToGraph.test.ts` + 47 from `shared/renderer/graph/treeToGraph.test.ts`)

Command: `cd vscode && npx jest --testPathPattern="treeToGraph" --no-coverage`

---

### Step 6: Duplicate File / Platform Import Checks

| Check | Result |
|-------|--------|
| `web/src/shared/renderGraph.ts` absent | ✅ PASS — file is gone |
| `web/src/shared/treeToGraph.ts` absent | ✅ PASS — file is gone |
| `web/src/shared/` only has: `helpers.ts`, `snapshotStateMachine.ts`, `themes/`, `treeOps.ts` | ✅ PASS |
| `grep -r "from 'vscode'" shared/renderer/` | ✅ PASS — CLEAN |
| `grep -r "from '.*web/" shared/` | ✅ PASS — no web imports |

---

## Summary

Phase 1 is verified. The shared renderer:
- Builds cleanly in both web and vscode
- Renders the graph correctly in the web app (SVG with `wf-node`/`ed-node` classes)
- Exposes the prune toggle in the toolbar
- Has zero platform-specific imports
- Has no duplicates left in `web/src/shared/`
- 84 unit tests all pass

**One tsconfig bug was found and fixed** as part of this verification. The fix is clean and precise: excluding test files from the web TypeScript compilation.
