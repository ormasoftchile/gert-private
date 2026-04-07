# kurapika-task2-renderer — Decision Record

**Filed by:** Kurapika  
**Date:** Phase 1 Task 2  
**Commit:** ebe3eb9

## GraphRenderOptions final shape

```typescript
export interface GraphRenderOptions {
  // Core
  delayInfo?: { stepId: string; delay: string; delayMs?: number; delayStartMs?: number } | null;
  theme?: GertGraphTheme;
  outcomeLabel?: string;
  annotationCounts?: Map<string, number>;
  annotations?: Annotation[];
  outcomeResult?: any;

  // VS Code parity
  invokeChildren?: Map<string, InvokeChildData>;
  branchResolutions?: Map<string, Map<number, boolean>>;
  selectedIteratePass?: Map<string, number>;
  snapshotState?: SnapshotState;
  iteratePassInfoMap?: Map<string, any>;
  iterateChildDetailsByPass?: Map<string, Map<string, any>[]>;
  stepDetails?: Map<string, any>;
  currentStepId?: string;
  displayConfig?: DisplayConfig;

  // Chain
  chainHistory?: ChainEntry[];
  viewingChainIndex?: number | null;

  // Flags
  debugMode?: boolean;
  recording?: boolean;
  recordingLog?: any[];
  autoScreenshot?: boolean;
  runCompleted?: boolean;
  savedGraphTransform?: { tx: number; ty: number; scale: number } | null;

  // Callbacks
  findChildChainIndex?: (invokeStepId: string) => number | null;
  getIteratePassDetail?: (stepId: string) => any | undefined;
}
```

## Design decisions

### currentStepId vs currentStepDetail
VS Code's `GraphRenderContext` has `currentStepDetail: any` and uses `ctx.currentStepDetail?.stepId`. The shared renderer flattens this to `currentStepId?: string` — callers extract the stepId before passing to `renderExecutionGraph`. This removes `any` indirection.

### No vscode.* callbacks needed
All 20 VS Code features were extractable without requiring VS Code API callbacks. The two interactive methods (`getIteratePassDetail`, `findChildChainIndex`) that were methods on `RunbookPanel` are now plain optional function fields. Default implementations (returning `undefined` / `null`) make them safe to omit.

### DisplayConfig moved to shared
Previously only in `vscode/src/serve/client.ts`. Now in `shared/renderer/types.ts`. VS Code's `client.ts` can re-export it from there in Task 7.

### GraphRenderContext promotion
The minimal 4-field `GraphRenderContext` in `stepNodeRenderer.ts` was replaced with a full interface (20+ fields) in `shared/renderer/types.ts`. The full interface is constructed internally by `buildContext()` — consumers only see `GraphRenderOptions`.

## VS Code features that needed design decisions

| Feature | VS Code | Shared |
|---------|---------|--------|
| Theme selection | `vscode.workspace.getConfiguration('gert').get('graph.theme')` | `options.theme ?? getTheme()` |
| Current step | `ctx.currentStepDetail?.stepId` (any) | `options.currentStepId?: string` |
| Recording push | direct mutation of `ctx.recordingLog` | same — caller passes mutable array |
| SnapshotState | required in ctx | optional with empty default |
