# brian — History

## Core Context (Phases 1–17 Summary)

**Project:** gert — Governed Executable Runbook Engine v2 redesign  
**Mission:** Complete v2 from specification, applying v1 learnings  
**Role:** Implementation Lead (Go runtime engineer)

### v1 → v2 Transition

**v1 Features (for reference):**
- YAML runbook execution with real-time governance
- Approval gates, allowlists, command redaction, evidence capture (append-only JSONL)
- VS Code ext, TUI, JSON-RPC server, .tool.yaml and .provider.yaml extensions

**v2 Goals:**
1. Retain governance differentiator (approval gates, denylist-wins, audit trail)
2. Add OpenTelemetry hooks, W3C Trace Context propagation
3. Clarify core/extension boundary via JSON-RPC
4. Support saga/compensation patterns for complex workflows
5. Policy-as-code path (OPA integration deferred to v2.1)

### Critical Architecture Decisions (Locked)

| Decision | Rationale |
|----------|-----------|
| Flat ExecutionPlan (no DAG) | Runtime makes branch/iterate decisions dynamically |
| Single-run-per-process | Simplifies correctness; concurrency deferred to v2.1 |
| Client-driven step advancement | No auto-advance; explicit state machine control |
| Denylist wins over allowlist | Deny-by-default for unknown commands (safety-first) |
| 5 governance primitives | allowlist, denylist, env-block, redaction, approval-gates |
| Crash-safe trace | Single `write(2)` + `fsync` per DurableEvent; HMAC-SHA256 tamper evidence optional |
| SSE for run output streaming | Real-time events; JWT bearer auth for `gert serve` |

### v2 Stack

- **Parser:** JSON Schema structural + semantic validation (two-phase)
- **Planner:** DAG → flat ExecutionPlan (resolves includes, tool lookup)
- **Engine:** Step loop, trace writer, event bus, checkpoint/approval gates
- **Trace:** JSONL format, 14 normative event types, DurableEvent envelope
- **Evidence:** text, checklist, attachment (SHA256 content-addressed)

### Team

- **Ken** (Architect): Design review, correctness strategy
- **Brian** (Implementation): Phase design/exec, spec compliance, code review
- **Barbara** (Integration): E2E test design, cross-phase coordination
- **Dennis** (Research): Industry patterns (saga, OTLP, governance-as-code)
- **Leslie** (Documentation): Design authorship, domain kit guides
- **Raphael** (UI): VS Code extension, visual correctness

### Phase Outcomes

| Phase | Focus | Status |
|-------|-------|--------|
| 1–2 | Governance & planner architecture | ✅ Locked |
| 3–7 | Parser, runtime core, event model | ✅ Locked |
| 8–13 | Layers (input providers, HTTP/WS/SSE, evidence, OTLP, CLI) | ✅ Complete |
| 14–17 | Integration testing, server hardening, security fixes | ✅ Complete |
| 18+ | Domain kit integration, scheduler, mobile app | 🔄 In progress |

---

## Phase 3 — Domain Home Compiler Integration (2026-04-23 to 2026-04-24)

### Deliverables

1. **Integration Test Suite** — `domains/home/integration_test.go`
   - 3 integration tests with `//go:build integration` tag
   - Parse-time validation strategy (no full engine execution)
   - All tests passing ✅

2. **CLI Validation Tool** — `domains/home/cmd/home-validate/main.go`
   - 69 lines
   - Compiles and outputs YAML for human inspection
   - Demo-friendly manual validation

3. **Decision Record** — `.squad/decisions.md`
   - GERT v2 entry points documented (Parser, Planner, Engine)
   - Integration approach rationale
   - Go module boundary constraints analysis
   - Implementation patterns and learnings

### GERT v2 Entry Points Discovered

**Parser (`v2/pkg/parser`):**
- `Parser.Parse(ctx, path) (*ParsedRunbook, error)`
- `Parser.ParseBytes(ctx, []byte) (*ParsedRunbook, error)`
- Two-phase validation: JSON Schema + semantic rules
- Output: `*ParsedRunbook` wrapping `*schema.Runbook`

**Planner (`v2/pkg/planner`):**
- `Planner.Plan(ctx, *ParsedRunbook) (*ExecutionPlan, error)`
- Config: `RunbookLoader` (include resolution), `ToolRegistry`
- Output: `*engine.ExecutionPlan` with resolved steps

**Engine (`v2/pkg/engine`):**
- `Engine.Start(ctx, *ExecutionPlan, RunOptions) (Handle, error)`
- `Handle.Next(ctx) (*StepResult, error)`
- Wired via `adapter.BuildEngineConfig()` with real platform
- Output: `RunState.Status` → Completed/Failed/etc

### Problem: Internal Package Boundary

The integration test cannot import `v2/internal/*` from outside module. Go language constraint.

**Solution: Parse-Time Validation**
- ✅ Compiled YAML is syntactically valid (unmarshals cleanly)
- ✅ Has expected structure (all required fields present)
- ✅ Round-trips through file I/O
- ⚠️ Does NOT validate JSON Schema rules (requires parser internals)
- ⚠️ Does NOT validate semantic cross-field rules
- ⚠️ Does NOT execute engine

This validates the compiler's responsibility (boundary correctness), not GERT's engine.

### Test Results

```
cd domains/home && go test -tags integration -v ./
✅ TestIntegration_CompileAndParsePoolCleanRoutine
✅ TestIntegration_ParseAllCompiledRoutines (7 routines/incidents)
✅ TestIntegration_WriteCompiledRunbooksToFile (round-trip validation)
PASS ok  0.200s
```

### Learnings

1. **Parser + Planner + Engine** form v2's execution stack
2. **E2E harness pattern** exists in `v2/internal/e2e/helpers_test.go`
3. **Module boundaries** are strict; internal packages only accessible to parent module
4. **Parse-time validation** sufficient for compiler boundary proof
5. **Full E2E test** requires living inside v2 module (`v2/internal/e2e/domain_home_test.go`)

### Next Steps (Deferred)

- Shell out to `gert validate` for schema validation (external process)
- Full E2E test in v2 repo (requires v2 module scope)
- CI pipeline smoke test: `gert run <compiled-yaml>`

**Phase 3 Status:** ✅ Complete

---

## Phase 4 — Three Test Suites (2026-04-24)

### Deliverables

1. **Seasonal Cadence Tests** — `domains/home/pkg/compiler/seasonal_test.go`
   - 6 test functions covering seasonal interval selection
   - Added `CompileRoutineAt(r, now)` method for time-injected compilation
   - Tests all four seasons (summer, winter, spring, autumn)
   - Tests fallback to `every:` when no `seasonal:` block
   - Tests edge case of ONLY `seasonal:` with no `every:`
   - All tests passing ✅

2. **Delegation Runtime Tests** — `domains/home/pkg/delegation/policy_test.go`
   - 14 test functions for delegation policy runtime behavior
   - `IsAwayModeActive()`: 6 tests (within/before/after window, boundaries, nil)
   - `RoutineIsAssignedToDelegate()`: 4 tests (found/not found/empty/nil)
   - `GetDelegateForRoutine()`: 4 tests (found/not assigned/not active/nil)
   - All tests passing ✅

3. **CLI Integration Test** — `domains/home/cmd/home-validate/main_test.go`
   - 1 integration test with `//go:build integration` tag
   - Builds and runs `home-validate` binary with `casa-santiago.home.yaml`
   - Validates YAML structure (not golden file comparison)
   - Parses each YAML document to ensure validity
   - Asserts basic sanity (length > 100 bytes, contains id field)
   - All tests passing ✅

### Implementation Details

**Time Injection for Seasonal Tests:**
- Refactored `CompileRoutine()` to call `CompileRoutineAt(prop, r, time.Now())`
- Added `CompileRoutineAt(prop, r, now)` for deterministic seasonal testing
- Created `resolveIntervalAt(cadence, now)` to support time injection
- Existing `resolveInterval()` now delegates to `resolveIntervalAt(cadence, time.Now())`
- This preserves backward compatibility while enabling testability

**Delegation Test Design:**
- Tests use fixed dates (e.g., 2025-04-25 to 2025-05-02) for reproducibility
- Boundary tests validate inclusive [from, to] window semantics
- Validates that `model.Delegation.IsAwayModeActive(now)` is used correctly
- Tests confirm nil-safety across all functions

**CLI Test Design:**
- Uses `os/exec` to run compiled binary (subprocess pattern)
- Separates stdout (YAML) from stderr (summary) for validation
- Structural validation instead of golden file comparison (less brittle)
- Validates YAML syntax with `gopkg.in/yaml.v3`
- Uses `extractYAMLDocuments()` helper to split multi-doc output

### Test Results

```
cd domains/home && go test ./...
✅ pkg/compiler (12 tests: 6 new + 6 existing)
✅ pkg/delegation (14 tests: all new)

cd domains/home && go test -tags integration ./...
✅ domains/home (3 integration tests: existing)
✅ cmd/home-validate (1 integration test: new)
✅ pkg/compiler (12 tests)
✅ pkg/delegation (14 tests)
```

**Total new test count: 21 tests**
- Seasonal cadence: 6 tests
- Delegation policy: 14 tests
- CLI end-to-end: 1 test

### Learnings

1. **Time injection is essential for seasonal logic testing** — Without it, tests are non-deterministic and tied to current date
2. **Delegation policy boundary semantics** — Window is inclusive on both ends (from ≤ now ≤ to)
3. **Integration test patterns** — Subprocess execution with `os/exec` is idiomatic for CLI testing
4. **YAML validation strategies** — Structural validation (parse + assert fields) > golden files for compiler output
5. **Test organization** — Domain tests at package level, integration tests with build tags

**Phase 4 Status:** ✅ Complete

---

## Open Questions & Future Work

1. Should full E2E tests live in v2 repo?
2. CI pipeline: Shell out to `gert validate` or add full E2E?
3. Scheduler integration: How to wire gert-domain-home with GERT runtime?
4. Mobile app: Should compile YAML or call Scheduler HTTP API?

## Key Files

- `design/gert-v2/main.tex` — Design document (LaTeX)
- `v2/internal/{parser,planner,engine}` — Core v2 stack
- `v2/cmd/gert` — CLI tool
- `domains/home/` — gert-domain-home compiler + tests
- `.squad/decisions.md` — All decisions (Phase 1–3)

---

## Phase 4: Domains/Home Test Coverage (2026-04-24)

**Status:** ✅ Complete — 21 tests green

**Mission:** Write comprehensive test suites for domains/home compiler to validate correctness.

### Deliverables

1. **Seasonal Cadence Tests (6 tests)**
   - Added time-injection variant `CompileRoutineAt(prop, routine, now)` to eliminate time-dependent flakiness
   - Tests: Summer (7d), Winter (21d), Spring (14d), Autumn (7d), boundary conditions
   - Pattern established for future time-dependent testing

2. **Delegation Policy Tests (14 tests)**
   - `IsAwayModeActive()` — away window validation with time-injection
   - `RoutineIsAssignedToDelegate()` — routine-to-delegate mapping
   - `GetDelegateForRoutine()` — integration of both checks
   - Coverage: happy path, boundary conditions, edge cases

3. **CLI Integration Test (1 test)**
   - End-to-end: `home-validate casa-santiago.home.yaml → GERT YAML`
   - Structural YAML validation (parse + assert properties)
   - Not golden-file based (more resilient to compiler improvements)

### Decisions Generated

1. **Time Injection Pattern for Seasonal Cadence Testing** — Deterministic, idiomatic Go approach
2. **Structural YAML Validation over Golden Files** — Maintainable, clearer test failures
3. **Delegation Policy Function Signatures (Observation)** — Asymmetry is intentional; documented future enhancement path

### Technical Patterns

- Time injection method signature: `func (c *Compiler) CompileRoutineAt(prop, routine, now)`
- Structural validation: parse YAML → check field presence → validate structure
- Integration tests use `//go:build integration` tag (not run by default)

### Quality

- All 21 tests green ✅
- Follows existing test patterns in codebase
- Zero breaking changes to public API

### Next Phase

- Ken: v2 schema roundtrip validation (with documented constraint)
- John: Minimal example property
- Team: Merge Phase 4 decisions, address schema gaps in v1 enhancement

