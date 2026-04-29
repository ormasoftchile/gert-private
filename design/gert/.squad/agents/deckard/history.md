# Deckard — Architectural Lead, gert-tui

**Role**: Architecture, scope decisions, Go design patterns, component boundaries, code review.

**Project**: gert-tui standalone TUI binary

---

## 2026-04-29: Roy's Scaffold Review — Internal Package Blocker

**Session**: gert-tui Module Integration Blocker Analysis  
**Status**: ✅ DECISION APPROVED  

### Problem Summary

Roy completed initial Go module scaffold for gert-tui with two blockers:

1. **Blocker 1 (go:embed)**: `//go:embed ../../runbooks/*.yaml` violates Go embed rules (no path traversal)
2. **Blocker 2 (hard)**: gert-tui cannot import `internal/parser`, `internal/planner`, `internal/engine` — Go modules enforce internal/ access only within same module

### Audit Findings

**gert Public API:**
- ✅ `pkg/parser` — Parser interface, ParsedRunbook type
- ✅ `pkg/planner` — Planner interface, Config type, error definitions
- ✅ `pkg/engine` — Engine interface, RunHandle, EngineConfig, ExecutionPlan, RunOptions
- ✅ `pkg/input` — Input provider interfaces
- ✅ `pkg/schema` — Runbook/tool schema types

**gert Internal Wiring:**
- ❌ `internal/adapter/wire.go:BuildEngineConfig` — Wires parser + planner + engine (not public)
- ❌ `internal/parser` — Parser implementation (not accessible from external modules)
- ❌ `internal/planner` — Planner implementation (not accessible from external modules)
- ❌ `internal/engine` — Engine implementation (not accessible from external modules)

**Key Finding**: gert exposes the interfaces and types, but NOT the wiring logic or constructors needed to build end-to-end flows.

### Decision: Option A — Add pkg/run to gert

**Chosen Resolution**: Create public `pkg/run` package in gert that:
1. Accepts runbook path + options
2. Internally uses `internal/parser`, `internal/planner`, `internal/engine`
3. Returns unified `Runner` interface or simplified `Run(path, opts) RunHandle` function

**gert-tui integration path**:
- Import only `pkg/run` (+ pkg/input, pkg/engine for types)
- Calls `pkg/run.Run(runbookPath, opts)` 
- Wires RunHandle to TUI event loop

**Rationale**:
- ✅ Spec principle: "grow gert, not gert-tui"
- ✅ gert-tui remains separate, independently versioned binary
- ✅ Cleanest architectural boundary: external consumers don't touch internals
- ✅ Extensible: batch/dry-run/replay modes all live in pkg/run

**Alternatives rejected**:
- ❌ Option B (move gert-tui to cmd/gert-tui): violates separate binary story
- ❌ Option C (promote internal/ to public): exposes unstable API surface

### Blocker 1 Resolution: go:embed Path

**Decision**: Move runbooks from `/gert-tui/runbooks/` to `/gert-tui/cmd/gert-tui/runbooks/`

**Rationale**: Runbooks are TUI-specific assets; embedding from cmd package is idiomatic.

### Action Items

**Phase 1 (Roy, immediate)**:
- [ ] Move runbooks directory under cmd/gert-tui/
- [ ] Update embed.go to use local embed path

**Phase 2 (gert team, sequential)**:
- [ ] Design + implement pkg/run in gert
- [ ] Add tests and documentation

**Phase 3 (Roy, after Phase 2)**:
- [ ] Wire TUI to pkg/run.Run()
- [ ] Implement engine event loop in TUI app

### Architectural Principle Reinforced

This decision reinforces the spec principle: **dependencies flow from TUI → gert, not gert → TUI**. gert grows new public APIs to serve consumers; consumers do not patch into internals.

---

## Knowledge Base

### gert Module Structure

```
github.com/ormasoftchile/gert/
├── pkg/                           — PUBLIC API
│   ├── engine/       interface Engine, RunHandle, ExecutionPlan, EngineConfig, RunOptions
│   ├── parser/       interface Parser, ParsedRunbook
│   ├── planner/      interface Planner, Config
│   ├── input/        interface PromptProvider, InputProvider
│   ├── schema/       Runbook, Tool, Step schema types
│   └── trace/        Event types, TraceWriter interface
├── internal/         — PRIVATE IMPLEMENTATION
│   ├── adapter/      BuildEngineConfig (wiring)
│   ├── parser/       Parser implementation
│   ├── planner/      Planner implementation
│   ├── engine/       Engine implementation
│   └── ...
└── cmd/
    └── gert/         — CLI (uses BuildEngineConfig)
```

### gert-tui Module Structure

```
github.com/ormasoftchile/gert-tui/
├── cmd/gert-tui/
│   ├── main.go
│   └── runbooks/             ← MOVE HERE (from ../../runbooks)
├── internal/
│   ├── tui/         — TUIApp (3-panel bubbletea model)
│   └── embed/       — embed.go (will use pkg/run)
└── go.mod           — requires gert
```

### Design Coherence

- **Separation**: TUI does not couple to gert internals; consumes only public interfaces
- **Extensibility**: pkg/run becomes the standard way to execute runbooks (batch, web, mobile can use it)
- **Versioning**: gert-tui can release independently; depends on stable pkg/run contract
