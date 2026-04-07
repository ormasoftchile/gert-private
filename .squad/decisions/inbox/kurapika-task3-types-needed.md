# Type gaps found in Phase 1 Task 3 — stepNodeRenderer port

**Filed by:** Kurapika (2026-06-26)
**Context:** Porting `vscode/src/views/stepNodeRenderer.ts` to `shared/renderer/graph/stepNodeRenderer.ts`

## Missing: `GraphRenderContext`

The full `GraphRenderContext` interface lives in `vscode/src/views/graphRenderer.ts` and has not been promoted to `shared/renderer/types.ts`. The ported `stepNodeRenderer.ts` only needs a subset — a minimal local interface was defined for now.

### Fields used by stepNodeRenderer (minimum needed in shared):

```typescript
export interface GraphRenderContext {
  invokeChildren: Map<string, InvokeChildData>;
  branchResolutions: Map<string, Map<number, boolean>>;
  getIteratePassDetail: (stepId: string) => any | undefined;
  findChildChainIndex: (invokeStepId: string) => number | null;
}
```

### Full interface in vscode (for reference):

The full `GraphRenderContext` also includes: `annotations`, `viewingChainIndex`, `chainHistory`, `tree`, `stepStates`, `stepDetails`, `selectedIteratePass`, `snapshotState`, `iterateChildDetailsByPass`, `displayConfig`, `delayInfo`, `outcomeResult`, `runCompleted`, `iteratePassInfoMap`, `currentStepDetail`, `debugMode`, `autoScreenshot`, `savedGraphTransform`, `recording`, `recordingLog`.

These in turn reference `Annotation`, `ChainEntry`, `DisplayConfig`, and `SnapshotState` (the last one IS already in shared types).

## Recommendation

When `graphRenderer.ts` itself is ported (likely a later Phase 1 task), promote the full `GraphRenderContext` to `shared/renderer/types.ts` and remove the local minimal interface from `stepNodeRenderer.ts`.
