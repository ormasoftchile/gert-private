# Agent: Brian — Session History

## 2026-04-28: pkg/run Public API Implementation

**Session**: gert v2 pkg/run — public runner API for external clients  
**Role**: Implementation Engineer

### Context

Received task from Cristian: gert-tui (separate Go module) cannot import gert's internal packages to wire parser + planner + engine. Architectural decision (Deckard): add `pkg/run` public package exposing a high-level runner API.

### Contributions

1. **pkg/run/run.go** — Public Start() API
   - `Config` struct: RunbookPath/RunbookFS, PromptProvider, Variables, Client, KitFS, TraceWriter, OnEvent
   - `Start(ctx, Config) (RunHandle, error)` — wires parser → planner → engine → RunHandle
   - Handles both filesystem and embedded FS loading (RunbookPath or RunbookFS+RunbookName)
   - Internal wiring: builds parser, planner, tool registry, executor registry, engine config
   - Returns ready-to-use RunHandle for `Next()` loop

2. **Adapters** — Bridge internal/pkg interfaces
   - `toolRegistryAdapter`: adapts internal MapRegistry to planner.ToolRegistry
   - `fsRunbookLoader`: loads runbooks from fs.FS for include support
   - `noopTraceWriter`: discards trace events when nil TraceWriter
   - `noopPromptProvider`: auto-selects defaults for non-interactive mode

3. **pkg/run/run_test.go** — Basic test coverage
   - TestStart_MinimalRunbook: validates full wiring with echo step
   - TestStart_MissingRunbook: error handling for missing runbook
   - TestStart_InvalidRunbook: error handling for malformed YAML

### Implementation Details

**Files Created**: 3
- `pkg/run/doc.go` — Package documentation
- `pkg/run/run.go` — Start() implementation + adapters (320 lines)
- `pkg/run/run_test.go` — Test suite

**Wiring Pattern**:
1. Load runbook bytes (os.ReadFile or fs.ReadFile)
2. Parse with internal/parser.New(platform)
3. Build tool registries (internal MapRegistry + planner adapter)
4. Plan with internal/planner.New(config)
5. Build EngineConfig (executors, dispatcher, trace, platform, etc.)
6. Create engine with internal/engine.New(config)
7. Start run → return RunHandle

**Build**: ✅ Passes (`go build ./...`)  
**Tests**: ✅ All pass (`go test ./pkg/run/... -v`)  
**Vet**: ✅ No issues (`go vet ./...`)

### Commits

- **800d52c**: feat: add pkg/run public runner API for external clients
  - Implements full Start() wiring
  - Exposes Config struct for external clients
  - Handles both filesystem and embedded FS loading
  - No internal package access required by callers

### Key Achievements

✅ **Public API Surface**: Clean Start(ctx, Config) → RunHandle contract  
✅ **No Internal Imports**: External clients use only pkg/ packages  
✅ **Full Engine Wiring**: Parser + planner + engine in one call  
✅ **FS Support**: RunbookPath OR RunbookFS+RunbookName loading  
✅ **Tested**: 3 tests cover happy path + error cases  
✅ **Ready for gert-tui**: Roy can wire against this immediately

### Status

🎯 **pkg/run API COMPLETE**

Implementation complete and merged to main. gert-tui can now import `github.com/ormasoftchile/gert/pkg/run` and call `run.Start()` to wire the full engine.

---

## 2026-04-28: TUI Interface Gaps Implementation

**Session**: Gert v2 Go implementation — 3 critical TUI interface gaps  
**Role**: Implementation Engineer

### Context

Received architectural decision from Ken: add EventKindStepOutput, EventKindRunFailed, and RunState.Plan to unblock gert-tui panel integration. Architecture validated; scope clear; constraints understood.

### Contributions

1. **EventKindStepOutput** (pkg/trace/event.go)
   - New EventKind constant for real-time step output streaming
   - Engine emits this kind as step accumulates stdout/stderr
   - Enables TUI output panel to display live progress

2. **EventKindRunFailed** (pkg/trace/event.go)
   - New EventKind constant for explicit run failure
   - Engine emits on error exit vs. success exit
   - Enables TUI to distinguish failed runs visually

3. **RunState.Plan** (pkg/engine/run.go)
   - New field: `Plan *ExecutionPlan` on RunState struct
   - Populated during run initialization from input ExecutionPlan
   - Enables TUI step-list panel to render step names/descriptions/order

### Implementation Details

**Files Modified**: 3
- `pkg/trace/event.go` — 2 EventKind constants
- `pkg/engine/run.go` — 1 field addition + initialization
- `internal/engine/engine.go` — Event emission logic + Plan capture

**Build**: ✅ Passes (`go build ./...`)  
**Tests**: ✅ All pass; no regressions  
**Quality**: Code reviewed; approved for merge

### Commits

- **f960d2f**: feat: add step/output, run/failed event kinds and Plan to RunState
  - Implements all 3 gaps
  - Follows Ken's architecture decision (Option B: event-based + field addition)
  - ~30 lines net addition

- **824d70f**: docs: document gert-tui interface gaps implementation
  - Adds this history entry
  - Documents decision rationale and implementation scope

### Key Achievements

✅ **EventKindStepOutput**: Step output now streams as discrete events  
✅ **EventKindRunFailed**: Run failure now distinguished from completion  
✅ **RunState.Plan**: ExecutionPlan now exposed to TUI consumers  
✅ **Zero Breakage**: All existing tests pass; backward compatible  
✅ **Ready for Integration**: TUI panels can now wire to these data sources

### Status

🎯 **TUI INTERFACE GAPS COMPLETE**

Implementation complete and merged to main. Ready for TUI panel integration in next phase.

---

## Architectural Notes

**Decision**: Option B (Event-Based + Field Addition) vs. Option A (Polling-Based State Machine)

**Why Option B**:
- **Efficient**: No polling overhead; changes drive events
- **Simple**: Events are domain facts; no complex state tracking
- **Aligned**: Matches gert v2 event-stream architecture
- **Extensible**: New event kinds added incrementally
- **Proven**: Used throughout gert v2 codebase already

**Trade-offs Accepted**:
- TUI must register event listeners (vs. polling); standard bubbletea pattern
- RunState.Plan is reference; caller must not mutate (Go convention)

---

## Learnings

- RunGraph wired into TUIApp at app.go (parallel to flat state — flat state retained for rendering)
- Harness accessors (GetStepStatus, GetStepOutputStr, GetSteps, GetStepCount) now delegate to RunGraph
- Harness-first rule: TUI rendering still uses flat state; RunGraph integration validated via tests
