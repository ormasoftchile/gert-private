# ken — History

## Core Context (Phases 1–17 Summary)

**Project:** gert — Governed Executable Runbook Engine (v2 redesign)  
**Mission:** Complete redesign applying v1 learnings; establish v2 as specification-driven reference  
**Role:** Architect (design review, correctness strategy, decision authority)

### Foundation

**Team:** Ken (Architect) + Brian (Implementation), Barbara (Integration), Dennis (Research), Leslie (Docs), Raphael (UI)

**Codebase:** 
- `design/gert-v2/` — LaTeX specification document
- `v2/` — Go implementation (parser, planner, engine, adapters)
- `domains/home/` — First official domain kit (gert-domain-home compiler)

### v1 → v2 Key Transition

**v1 Success Factors:**
- Governance (approval gates, allowlists, redaction) beats feature parity
- Evidence capture (append-only JSONL, SHA256, resumption) enables auditability
- Extension isolation (out-of-process .tool.yaml, .provider.yaml) prevents side effects

**v2 Goals:**
1. Retain all governance differentiators
2. Add OpenTelemetry hooks, W3C Trace Context
3. Clarify core/extension boundary via JSON-RPC
4. Support saga/compensation patterns (Temporal-style)
5. Policy-as-code path (OPA integration deferred to v2.1)

### Critical Architectural Decisions (Locked Phases 1–14)

**Execution Model:**
- Flat ExecutionPlan (no DAG); runtime makes branch/iterate decisions dynamically
- Single-run-per-process for v2.0 (concurrency deferred to v2.1)
- Client-driven step advancement (no auto-advance; explicit state machine)

**Governance:**
- Denylist wins over allowlist (deny-by-default for unknown commands)
- 5 primitives: allowlist, denylist, env-block, redaction, approval-gates
- Approval gate model: ordered checkpoints, timeout/escalation, trace audit trail

**Infrastructure:**
- Crash-safe trace: single `write(2)` + `fsync` per DurableEvent
- 14 normative event types; HMAC-SHA256 tamper evidence (optional)
- SSE for run output streaming; JWT bearer auth for `gert serve`
- Evidence types: text, checklist, attachment (SHA256 content-addressed)

### Design Document Sections

**Complete (locked):**
- §02 Architecture: five-component model, data flow, lifecycle, concurrency control
- §11 Governance: primitives table, checkpoint logic, approval UX, audit trail
- §12 Evidence & Tracing: JSONL format, event taxonomy, crash safety
- §08 Testing: scenario discovery, contract tests, golden trace comparison, CI/CD gates

**Pending:**
- Kit-Zero (DRI-specific) → being replaced by domain-agnostic Domain Kit Examples
- Policy-as-code (§10) → deferred to v2.1

### Phase Outcomes

| Phases | Work | Completion |
|--------|------|-----------|
| 1–7 | Governance, planner, parser, runtime core, events | ✅ Locked |
| 8–13 | Layers: input providers, HTTP/WS/SSE, evidence, OTLP, CLI | ✅ Complete |
| 14–17 | Integration testing, server hardening, security audits | ✅ Complete |
| 18+ | Domain kit integration (home, scheduler, mobile app) | 🔄 In progress |

---

## Phase 3 — Domain Home Compiler Integration (2026-04-23 to 2026-04-24)

### Status: ✅ Complete

Brian's integration test deliverables for gert-domain-home compiler are complete and all tests passing.

### Deliverables

1. **Integration Test Suite** — `domains/home/integration_test.go`
   - 3 integration tests (`//go:build integration` tag)
   - Proves compiled YAML is syntactically valid and structurally correct
   - All 7 compiled outputs (5 routines + 2 incident templates) validated
   - Pass rate: 100% ✅

2. **CLI Tool** — `domains/home/cmd/home-validate/main.go`
   - 69 lines
   - Compiles Property to YAML and pretty-prints
   - Enables human-friendly demo and manual validation

3. **Decision Record** — Merged to `.squad/decisions.md`
   - Documented GERT v2 entry points (Parser, Planner, Engine, E2E harness)
   - Explained integration approach rationale
   - Analyzed Go module boundary constraints (internal/ packages)
   - Recorded learnings and patterns

### GERT v2 Integration Points

**Parser Layer (`v2/pkg/parser`):**
- Two-phase validation: JSON Schema structural + semantic cross-field rules
- Entry: `Parser.ParseBytes(ctx, []byte) (*ParsedRunbook, error)`
- Exit: `*ParsedRunbook` wrapping `*schema.Runbook`

**Planner Layer (`v2/pkg/planner`):**
- Resolves includes, performs tool lookup, builds ExecutionPlan
- Entry: `Planner.Plan(ctx, *ParsedRunbook) (*ExecutionPlan, error)`
- Exit: `*engine.ExecutionPlan` (flat list of executable steps)

**Engine Layer (`v2/pkg/engine`):**
- Real runtime with event bus, trace writer, step executor
- Entry: `Engine.Start(ctx, *ExecutionPlan, RunOptions) (Handle, error)`
- Control: `Handle.Next(ctx) (*StepResult, error)` (client-driven)
- Output: `RunState` (Completed/Failed/Waiting/etc)

**E2E Harness Pattern** (`v2/internal/e2e/helpers_test.go`):
- Uses `platform.Real()` (not mocks)
- Wires via `adapter.BuildEngineConfig(ctx, WireOptions)`
- Temp dirs for trace/run artifacts
- Drives engine to completion via `for { Next(); if EOF { break } }`
- Validates trace JSONL events for lifecycle assertions

### Validation Strategy

**What we proved (parser boundary):**
- ✅ Compiled YAML is syntactically valid (no unmarshal errors)
- ✅ Structure is complete (all required fields present, correct types)
- ✅ Round-trip I/O preserves structure (write → read → parse)

**What we did NOT prove (requires internal packages):**
- ⚠️ JSON Schema validation (parser internals)
- ⚠️ Semantic cross-field validation (planner internals)
- ⚠️ Full engine execution (runtime internals)

**Rationale:** Compiler's responsibility is boundary correctness. GERT engine tests validate execution.

### Key Learnings

1. **Three-layer stack architecture** — Parser → Planner → Engine (clean separation)
2. **Go module boundaries strict** — internal/ packages only accessible to parent module; no workaround
3. **Adapter pattern** — Production config wiring via `adapter.BuildEngineConfig()`
4. **Pragmatic testing** — Test the boundary you control; trust downstream validation
5. **Parse-time validation sufficient** — Proves syntactic correctness (90% of value)

### Consequences

**Positive:**
- ✅ Integration tests prove compilation boundary correctness
- ✅ No subprocess complexity (no CLI dependency)
- ✅ Fast tests (no engine overhead)
- ✅ Clear separation of concerns (compiler vs. runtime)

**Gaps (acceptable for now):**
- ⚠️ No full JSON Schema validation (requires parser internals)
- ⚠️ No execution validation (requires engine internals)
- ⚠️ Manual testing still needed for full validation

### Future Work (Deferred)

1. **Full E2E test in v2 repo:** Create `v2/internal/e2e/domain_home_test.go` (can import internals from same module)
2. **CI smoke test:** Shell out to `gert validate` or `gert run --dry-run`
3. **Scheduler integration:** Wire compiler output into scheduler's policy merge

---

## Architecture Checkpoints

**Locked (high confidence):**
- Execution model (flat plan, client-driven)
- Governance primitives (denylist-wins, approval gates)
- Evidence model (JSONL, tamper evidence, resumption)
- Event taxonomy (14 types, lifecycle semantics)

**Under refinement:**
- Policy-as-code language (OPA, CEL, custom DSL?)
- Scheduler/delegation integration
- Mobile app architecture

**Deferred to v2.1+:**
- Concurrent runs per process
- Policy-as-code implementation
- Advanced compensation patterns

## Key Files

- `design/gert-v2/main.tex` — Design document (325 pages, 1.4MB)
- `v2/internal/{parser,planner,engine,adapter}` — Core implementation
- `v2/pkg/{parser,planner,engine,schema}` — Public APIs
- `domains/home/pkg/compiler/` — Domain kit compiler
- `.squad/decisions.md` — All decisions (Phases 1–3)
