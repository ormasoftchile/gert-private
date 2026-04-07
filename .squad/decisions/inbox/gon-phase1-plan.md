# Decision: Shared Renderer Phase 1 — Full Feature Parity Architecture

**Date:** 2026-04-07
**By:** Gon (Lead Architect)
**Status:** PROPOSED

## What

Phase 1 of the shared renderer refactor: port ALL 20 VS Code graph renderer features into `shared/renderer/graph/` as a single implementation consumed by both VS Code and web. No feature remains platform-specific except IPC wiring.

## Key Architecture Decisions

### 1. `GraphRenderOptions` replaces `GraphRenderContext`

VS Code's `GraphRenderContext` is a class with methods bound to `RunbookPanel`. The shared version uses a plain options bag with optional callback injection:

```typescript
export interface GraphRenderOptions {
  // All features are optional — progressive enablement
  invokeChildren?: Map<string, InvokeChildData>;
  branchResolutions?: Map<string, Map<number, boolean>>;
  selectedIteratePass?: Map<string, number>;
  chainHistory?: ChainEntry[];
  viewingChainIndex?: number | null;
  displayConfig?: DisplayConfig;
  // Platform callbacks
  findChildChainIndex?: (invokeStepId: string) => number | null;
  getIteratePassDetail?: (stepId: string) => any | undefined;
}
```

**Why:** Options bag is serializable, testable without mocking, and allows each platform to provide only what it has. Callbacks are the escape hatch for platform-specific lookups.

### 2. Web's `branchCondition` fix becomes canonical

Web computes `branchCondition` with `→ true`/`→ false` at layout time. VS Code computes it dynamically during render. Shared version adopts web's approach.

**Why:** Layout-time computation is simpler, self-contained, and doesn't require passing `branchResolutions` to the subtitle renderer. The renderer can still use `branchResolutions` for richer subtitles when available, but `branchCondition` provides a guaranteed baseline.

### 3. Web's passthrough `nodeStepMap` fix becomes canonical

VS Code's `nodeStepMap.set(passId, id)` on passthrough nodes maps them to the parent branch step, causing `isTaken()` to incorrectly return true. Web correctly omits this mapping.

**Why:** Untaken branch edges rendering as green (taken) is a visual bug. Web discovered and fixed it; shared version inherits the fix.

### 4. Shared renderer owns ALL SVG generation

Both `renderExecutionGraph()` and `renderEditorGraph()` live in `shared/renderer/graph/renderGraph.ts`. Platform adapters are thin wrappers (~50 lines) that read platform settings and call the shared function.

**Why:** Single source of truth eliminates divergence. Testing one function verifies both platforms.

### 5. 10 tasks across 2 engineers, 7-9 day estimate

Critical path: shared treeToGraph → core renderer → platform adapters → web UI.
Maximum parallelism: Tasks 1, 3, 5, 6, 10 can all run concurrently.

## Risks

1. **CSS variable handling** — shared SVG uses `var(--vscode-*)` which web must map to `var(--gert-*)`. Mitigated by Phase 0 theme extraction.
2. **Feature interaction complexity** — iterate pass selection + expanded iterate + prune interact non-trivially. Task 4 (iterate state mapping) is the hardest single task.
3. **VS Code golden test breakage** — any SVG output change breaks existing goldens. Mitigated by adapter approach (VS Code adapter produces identical output).

## Impact

- Web gains 20 features it was missing
- VS Code loses zero functionality
- Both platforms share a single tested renderer
- Future graph features are implemented once
