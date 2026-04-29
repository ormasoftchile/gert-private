# Decision: pkg/run Public API Surface

**Date**: 2026-04-28  
**Author**: Brian (Go Implementer)  
**Status**: Implemented & Merged  
**Commit**: 800d52c

## Problem

gert-tui (external Go module `github.com/ormasoftchile/gert-tui`) needs to run gert runbooks but cannot import gert's `internal/` packages (parser, planner, engine constructors). Go's visibility rules prevent cross-module internal package access.

## Decision

Add `pkg/run` public package with a high-level `Start()` API that internally wires parser → planner → engine → RunHandle.

## API Surface

### Main Entry Point

```go
func Start(ctx context.Context, cfg Config) (engine.RunHandle, error)
```

- **Input**: `Config` struct with runbook source, variables, prompt provider, client ID
- **Output**: `engine.RunHandle` ready for `Next()` loop
- **Errors**: Parse, plan, or engine initialization errors

### Config Structure

```go
type Config struct {
    // Runbook source: EITHER RunbookPath OR (RunbookFS + RunbookName)
    RunbookPath string
    RunbookFS   fs.FS
    RunbookName string

    // Interactive prompts (nil = auto-approve defaults)
    PromptProvider input.PromptProvider

    // Client identifier (e.g., "tui", "cli", "batch")
    Client string

    // Initial variable bindings
    Variables map[string]string

    // Optional: domain kit for tool loading
    KitFS fs.FS

    // Optional: trace event writer (nil = no-op)
    TraceWriter trace.TraceWriter

    // Optional: runtime event callback
    OnEvent func(engine.Event)
}
```

## Wiring Pattern

Internal wiring sequence (hidden from caller):

1. **Load**: Read runbook bytes from RunbookPath or RunbookFS
2. **Parse**: `internal/parser.New(platform) → ParseBytes()`
3. **Tool Registry**: Build internal MapRegistry + planner adapter
4. **Plan**: `internal/planner.New(config) → Plan()`
5. **Engine Config**: Build EngineConfig (executors, dispatcher, trace, platform)
6. **Start**: `internal/engine.New(config) → Start() → RunHandle`

## Implementation Notes

### Adapters

Created 4 internal adapter types to bridge internal/pkg interfaces:

1. **toolRegistryAdapter**: Adapts `internal/tool.MapRegistry` to `planner.ToolRegistry`
   - Converts `pkg/tool.ToolDef` → `schema.ToolDef` on lookup
   
2. **fsRunbookLoader**: Loads runbooks from `fs.FS` for include support
   - Falls back to filesystem if RunbookFS is nil
   
3. **noopTraceWriter**: Discards trace events when TraceWriter is nil
   - Implements `trace.TraceWriter.Append()` as no-op
   
4. **noopPromptProvider**: Auto-selects defaults when PromptProvider is nil
   - Auto-approve first option/route, auto-fill form defaults

### Defaults

When optional Config fields are nil:
- **PromptProvider**: noopPromptProvider (auto-approve)
- **TraceWriter**: noopTraceWriter (discard)
- **KitFS**: nil (built-in tools only)
- **Variables**: empty map
- **OnEvent**: nil (no callbacks)

## Files

- `pkg/run/doc.go` — Package documentation
- `pkg/run/run.go` — Start() + adapters (320 lines)
- `pkg/run/run_test.go` — Test suite (3 tests)

## Testing

✅ **TestStart_MinimalRunbook**: Full wiring with echo step  
✅ **TestStart_MissingRunbook**: Error handling for missing runbook  
✅ **TestStart_InvalidRunbook**: Error handling for malformed YAML

## Impact

### External Clients (gert-tui)

**Before**:
```go
// ❌ Cannot import internal packages
import "github.com/ormasoftchile/gert/internal/parser"  // error
import "github.com/ormasoftchile/gert/internal/planner" // error
```

**After**:
```go
// ✅ Clean public API
import "github.com/ormasoftchile/gert/pkg/run"
import "github.com/ormasoftchile/gert/pkg/engine"

handle, err := run.Start(ctx, run.Config{
    RunbookPath: "runbook.yaml",
    Client:      "tui",
})
for {
    result, err := handle.Next(ctx)
    if err == io.EOF { break }
    // process result
}
```

### Internal CLI

No changes required. CLI continues using `internal/adapter.BuildEngineConfig()` for full control (trace files, OTLP, resume, etc.).

## Trade-offs

### Accepted

1. **Simplified Config**: External clients get subset of EngineConfig options (no governance evaluator, no extension host, no checkpoint/resume). Sufficient for TUI use case.

2. **No Substeps**: SubStepRunner is nil. External clients cannot use nested substep execution. Acceptable — substeps are CLI-only feature.

3. **Minimal Tool Loading**: KitFS is stub (built-ins only). Future: scan KitFS for `.tool.yaml` files.

### Not Accepted

- Exposing full EngineConfig to external clients: Too complex, breaks encapsulation
- Making internal packages public: Violates Go module visibility conventions

## Status

**Implemented**: 2026-04-28  
**Merged**: main branch (800d52c)  
**Ready for Use**: gert-tui can wire against `pkg/run.Start()` immediately

## Next Steps

1. Roy (gert-tui) wires TUI to `pkg/run.Start()`
2. Test end-to-end runbook execution in TUI
3. Future: Add KitFS scanning for external tool definitions
