# brian — History

## Project Context

**Project:** gert — Governed Executable Runbook Engine
**Owner:** ormasoftchile
**Mission:** Redesign gert from scratch as v2, applying learnings from v1.
**Stack:** Go, YAML, JSON Schema (Draft 2020-12), LaTeX (design docs), TypeScript (VS Code extension)

## Current Focus

The team is working on the **v2 design document** located at `design/gert-v2/`.
This is a LaTeX document using the MastersThesis class. Sections are in `design/gert-v2/sections/`.

### v2 Goals
1. Extensive research: runbooks, workflows, governance, traceability
2. Apply research + project learnings to define the new version
3. Document the new design as a usable reference to build v2

### v1 Key Features (for reference)
- Validate/execute/debug runbooks in YAML
- Governance: approval gates, allowlists, output redaction
- Evidence capture with SHA256, append-only JSONL traces
- VS Code extension + TUI + JSON-RPC server
- Tool definitions (.tool.yaml) and input providers (.provider.yaml)
- Step types: cli, manual, tool, invoke, branch, iterate

## Key Files
- `design/gert-v2/main.tex` — root LaTeX document
- `design/gert-v2/sections/` — all section .tex files
- `design/gert-v2/MastersThesis.cls` — document class
- `ext/` — Go source (core engine)
- `vscode/` — VS Code extension (TypeScript)

## Learnings

## Cross-Agent Notes from Dennis's Research Brief (2026-04-18)

**For Brian (Go Runtime):** Research prioritizes saga/compensation and context propagation as key implementation priorities.

- **Saga/Compensation Pattern**: Allow steps to register compensating actions; execute in reverse on failure. Industry standard in Temporal, Argo, Prefect. MVP priority (high impact, moderate effort).
- **Context Propagation**: OpenTelemetry trace context must propagate across invoked runbooks and tools. Baggage for runbook-specific metadata. Correlation ID in all log entries.
- **Timeout/Escalation State Machine**: Human step SLA enforcement with timeout and escalation paths. On timeout: escalate to alternate approvers, send notifications, or fail. MVP priority (high impact, low-moderate effort).
- **Idempotency Guarantees**: Step design requirements must be enforced and documented. Enables retry with backoff without side effects.

Full research brief: `.squad/tmp/dennis-research-brief.md`
Decision inbox: `.squad/decisions/inbox/dennis-research-foundations.md` → merged to `.squad/decisions.md`

## Learnings — 2026-04-18

### §12 Written From Scratch

Wrote the complete `12-evidence-tracing-resumption.tex` section. Key decisions embedded:
- JSONL trace lives at `.runbook/runs/<run-id>/trace.jsonl`; 14 normative event types defined
- `DurableEvent` envelope with `seq`, `type`, `timestamp`, `run_id`, `path`, `data` fields
- Crash safety via `O_APPEND` + single `write(2)` + `fsync` per event
- Optional HMAC-SHA256 tamper evidence on each event (`GERT_TRACE_KEY` env)
- Evidence: text, checklist, attachment (SHA256 content-addressed in `attachments/`)
- Snapshot format already exists in v1 (`RunState` JSON) — preserved in v2 with `version` field added
- Partial step resumption: idempotent tools re-issued; non-idempotent tools treated as failed
- Replay mode uses scenario YAML; v1 scenario format is forward-compatible
- OTel is opt-in (no-op tracer when `OTEL_EXPORTER_OTLP_ENDPOINT` unset); zero overhead default

### §08 Expanded

Expanded `08-testing-and-acceptance.tex` from 4 bullets to a full testing strategy covering:
- Test framework: Go `testing` + testify; table-driven tests as canonical pattern
- `gert test`: scenario discovery convention, test.yaml assertions, v1 scenario compatibility
- Contract tests for each interface boundary (runtime, schema, extension, tool, event bus)
- Integration test strategy with fixture corpus covering all step types
- Regression: v1 corpus at 95% pass rate target; golden trace comparison
- Performance targets: step startup <10ms p50, trace write <5ms p50
- Extension test harness via `pkg/exttest` package
- CI/CD: PR gate (unit+schema+contract+lint+vet), merge gate (integration+regression+golden), nightly (bench+race)

### Go Implementability Brief

Wrote `/Volumes/Projects/gert/.squad/tmp/brian-implementability-notes.md` covering:
- Top 5 concerns: event bus blocking, context propagation, saga compensation, parallel iterate safety, serve JSON-RPC compatibility
- Recommended package structure rooted at `pkg/` with clean `ext/` boundary
- Key interfaces: Runtime, Planner, EventBus, TraceWriter, RunStore, CommandExecutor, EvidenceCollector, GovernanceEngine
- Channels over mutexes for event bus; sequential engine loop (no goroutine-per-step)
- v1 patterns to preserve and anti-patterns to eliminate documented

### Cross-Agent Flags

- **For Ken**: §02 needs interface definitions matching the interfaces in the implementability brief.
- **For John**: `DurableEvent.data` is `json.RawMessage` — needs discriminated union JSON Schema per event type.
- **For Leslie**: §08 uses `\begin{tabular}` and `[label=\arabic*.]` — confirm `enumitem` is in the preamble.

## Learnings

### TikZ Diagram Replacement (2026-04)

- Replaced verbatim ASCII art dependency graph in sections/02-architecture.tex with a proper TikZ figure using stepbox and gertarrow styles.
- The edit tool failed to match verbatim blocks due to whitespace; used Python string replacement instead.
- build/ is in .gitignore; commit only main.pdf at repo root, not build/main.pdf.
- Bidirectional Runtime Core <-> Extension Host peer relationship shown with two bent arrows (to[bend right=12]) in each direction.

## Learnings — Phase 1 (2026-04-19)

### Phase 1 Parser Implemented at `v2/internal/parser/`

Implemented the full Phase 1 parser for gert v2 runbooks. All 26 tests pass; all 10 fixtures (r01-r10) parse successfully.

**New files:**
- `v2/internal/parser/parser.go` — implements `pkg/parser.Parser` interface
- `v2/internal/parser/unmarshal.go` — YAML to schema types with two-phase type dispatch
- `v2/internal/parser/validate_structural.go` — JSON Schema (Draft 2020-12) validation
- `v2/internal/parser/validate_semantic.go` — cross-field semantic rules
- `v2/internal/parser/errors.go` — ValidationError / ValidationErrors types
- `v2/internal/parser/parser_test.go` — 26 tagged tests (all passing)
- `v2/schemas/schema.go` — go:embed wrapper for runbook.schema.json

**Modified files (schema type extensions):**
- `v2/pkg/schema/runbook.go` — GovernanceConfig with GovernanceRule/RedactRule types
- `v2/pkg/schema/steps.go` — ToolInvocation.Args changed to map[string]any
- `v2/pkg/schema/step.go` — added StepTypeExtension = "extension"
- `v2/schemas/runbook.schema.json` — relaxed additionalProperties on multiple types to match fixtures

### Key Technical Finding: yaml.v3 duplicate-key panic

`schema.Step` has overlapping YAML field names across multiple inline spec structs (e.g. "prompt" in ChoiceSpec/CollectorSpec/DecisionSpec). yaml.v3 panics when encountering `schema.Step` transitively via `FlowNode.Step` during any Decode() call. The parser uses raw `*yaml.Node` extraction with type-dispatched spec decoding to avoid this entirely. This should be documented as a schema design constraint for future phases — inline specs with overlapping keys cannot coexist in a yaml.v3-decoded struct.

### Libraries
- YAML: gopkg.in/yaml.v3 v3.0.1
- JSON Schema: github.com/santhosh-tekuri/jsonschema/v6 v6.0.2

### Two-Phase Validation
1. Structural: JSON Schema Draft 2020-12
2. Semantic: duplicate step IDs, nested parallel detection (parallel/nested-forbidden), signal allow-list, path normalization, branch/approve/collector constraints

### Fixture Notes
- All r01-r10 pass
- r10 had a YAML quoting bug (shell escaping in fixture) — fixed in fixture file
- r02 uses `over: null` for convergence-mode iterate — schema accepts string-or-null
- Fixtures use `type: extension` (v1 retained step type) — added to StepType enum

### Cross-Agent Notes
- For Barbara/Ken: schema.Step needs a JoinSpec field for parallel steps — current design loses join: data
- For John: ToolInvocation.args relaxed to map[string]any — spec update needed
## Learnings - Phase 1 Correctness Fixes (2026-04-19)

Fixed 5 parser correctness gaps (S1-S5) flagged by Ken Phase 1 review.

S1: Updated stale comment in v2/pkg/parser/parser.go (semantic validation now in parser, not Planner).
S2: walkFlowNodes now emits parallel/nested-forbidden for flow-level ParallelNode when inParallel==true. New test: TestParser_NestedParallelNodeForbidden.
S3: Added explicit StepTypeExtension case with comment in unmarshal.go.
S4: collectStepIDs now counts fn.Iterate.ID and fn.Parallel.ID. New tests: TestParser_IterateNodeDuplicateID, TestParser_ParallelNodeDuplicateID.
S5: Added assertErrorCode helper; 9 tests now assert specific error codes.
Build: go build ./... go vet ./... go test ./internal/parser/... -> 23 PASS.

---

## 2026-04-19: Phase 2 Kickoff

**Status:** Approved and orchestrated

Parser Phase 1 fixes complete. Ready for Phase 2 planning logic.

**Deliverables:**
- validate_semantic.go, unmarshal.go, parser_test.go, pkg/parser/parser.go modified
- 23 tests pass (+3 new)
- All 5 correctness gaps (S1–S5) resolved

**Next:** Implement concrete Planner logic using Ken's interface design. Barbara ready with testutil real types. Build clean, ready for integration.

## Learnings — Phase 2 Planner (2026-04-19)

### Phase 2 Planner Implemented at `v2/internal/planner/`

Implemented the full Phase 2 planner. All 13 tests pass; `go build ./... && go vet ./...` clean.

**Modified files:**
- `v2/internal/planner/planner.go` — replaced stub with full implementation
- `v2/internal/planner/planner_test.go` — removed all t.Skip, fixed max-depth test path building, added 7 canonical test names + TestPlanner_TopoSort

**Key design decisions:**
- `planCtx` struct accumulates mutable state (tools map, seen map) across recursive calls — avoids threading extra params through every function
- Include steps inline child runbook steps directly into the flat step list (`ResolvedStep` has no sub-plan field)
- Declaration order IS the topological order — no BFS needed since schema has no `next` field
- Cycle detection uses permanent path marking (as specified), known diamond-dep limitation noted in decisions
- `specForStep` returns `rawSpec{kind}` fallback for nil spec pointers — prevents panics on malformed input
- Errors use existing `pkg/planner.PlanError{Code: plannerPkg.ErrXxx}` — test assertions use pkg sentinels, no duplicate internal errors file needed

**Test coverage (13 tests):**
- TestPlanner_MinimalRunbook, TestPlanner_ImportResolution, TestPlanner_ImportCycleDetected
- TestPlanner_MaxDepthExceeded, TestPlanner_ToolDiscovery, TestPlanner_UnknownTool, TestPlanner_TopoSort
- TestPlan_BasicRunbook, TestPlan_IncludeResolution, TestPlan_ToolResolution
- TestPlan_ImportCycleDetection, TestPlan_ToolNotFound, TestPlan_MaxDepthExceeded

**Cross-agent notes:**
- For Barbara: inline fakes in planner_test.go; replace with pkg/testutil when ready
- For Ken: no engine type changes needed; all schema specs already implement StepKind()

## 2026-04-19: Phase 2 Approval Finalized

**Status:** ✅ PHASE 2 APPROVED

Defects D1 (Barbara's interface guard) and D2 (John's backtracking fix) both verified correct. All 14 tests pass including new diamond-dependency test. Phase 2 implementation complete. Ready to transition to Phase 3.

## 2026-04-19: Phase 3 — Runtime Core Implementation

**Status:** ✅ COMPLETE — 16 tests pass, race-free (`-race`)

### What was built

`v2/internal/engine/engine.go` — concrete `pkg/engine.Engine` implementation.

The core loop uses the existing `Start() + RunHandle.Next()` client-driven model. Key additions over Ken's stub:

1. **Signal handling** — `Start()` creates `runCtx/runCancel`. A goroutine listens on `Platform.NotifySignals(runCtx)`. On signal: acquires mutex, emits `run/cancelled`, closes events channel, calls `runCancel()`. `mergeContexts()` helper propagates cancellation into step context so executors are interrupted.

2. **Parallel branches** — Engine type-asserts `step.Spec` to `parallelBranchProvider` interface (`GetBranches() []engine.BranchSpec`). Branches run in `errgroup` goroutines with isolated `eventBuffer` per branch. On join, buffers flushed in declaration order (deterministic trace ordering). `golang.org/x/sync v0.20.0` added to go.mod.

3. **wait_for_event** — Engine type-asserts to `waitEventProvider` interface (`EventFilter() (eventbus.EventFilter, time.Duration)`). Releases mutex while calling `dispatcher.Wait()`, emits `event/received` + `step/resumed`.

4. **Mutex discipline** — `executeStep()` and `executeWaitForEvent()` release `h.mu` before calling external I/O. `safeClose()` with `eventsClosed atomic.Bool` prevents double-close panics.

5. **No `run/failed` event kind** — `trace` package has none; infrastructure failures emit `run/completed` with error payload.

### Test coverage (16 tests)

Ken's 9 `t.Skip`-gated tests all enabled + 5 new:
- `TestEngine_EmitsRunEvents` — sequence monotonicity + event order
- `TestEngine_StepFailure` — step/failed present, step-2 not started
- `TestEngine_ParallelBranches` — 2 concurrent branches, events flushed in declaration order
- `TestEngine_WaitForEvent` — dispatcher unblocks step, event/received + step/resumed
- `TestEngine_SignalCancellation` — SIGINT while executor running → run/cancelled

### Cross-agent notes

- **Ken**: `parallelBranchProvider` and `waitEventProvider` are internal engine interfaces. The planner should provide a wrapper around `schema.ParallelNode` that implements `parallelBranchProvider` so the engine can dispatch without importing schema directly.
- **Barbara**: test fakes are inlined in `engine_test.go`; candidates for promotion to `pkg/testutil` if reuse grows.



## 2026-04-20: Phase 3 Approval

**Status:** ✅ PHASE 3 APPROVED

Ken's architectural review complete. Runtime core implementation approved — all 10 criteria pass. Implementation is race-free (-race -count=10), well-designed with proper parallel execution, wait_for_event, and signal handling. 16 tests pass. Cross-agent note: planner should wrap `schema.ParallelNode` to implement `parallelBranchProvider` interface so engine can dispatch without importing schema directly.

## Learnings — Phase 4 Governance Engine (2026-04-20)

### Phase 4 Implemented — Governance Engine Complete

Implemented the full Phase 4 governance engine according to Ken's design. All 26 tests pass with race detection.

**New files:**
- pkg/governance/evaluator.go, evidence.go, approval.go — public contracts
- internal/governance/builder.go, evaluator.go, approval.go, redaction.go, evidence.go — implementations
- internal/governance/*_test.go — 26 tests

**Modified files:**
- pkg/engine/run.go — StepStatusDenied, StepOutcomeDenied
- pkg/engine/engine.go — GovernanceEvaluator, ApprovalGate in EngineConfig
- internal/engine/engine.go — pre-flight checks, post-execution redaction

### Key Decisions

D1: StepInfo projection breaks import cycle (pkg/governance must not import pkg/engine)
D2: evaluator.requireApproval extracted via type assertion at construction
D3: Deny-wins enforced by evaluation order (deny first, short-circuit)
D4: Redaction is post-execution pass (before trace emission)
D5: Mutex released during ApprovalGate.RequestApproval (blocking call)
D6: path.Match for glob patterns (*, ?, [range])
D7: Evidence is a value type (no pointers, fully JSON-serializable)
D8: Step-level governance deferred to Phase 5

### Tests (26 passing, -race -count=5 clean)

Evaluator: 14 tests (allow, deny, deny-wins, permissive, glob, env-vars, approval)
Builder: 6 tests (merge semantics, allow-replace, deny-union, approval-OR)
Redaction: 4 tests (simple, multiple, recursive, no-match)
Approval: 2 tests (NoOp success, context cancellation)

Build & validation: go build ./... && go vet ./... && go test ./... -race -count=3 → all PASS

Phase 4 complete. Ready for Phase 5.

## Learnings — Phase 5 Executors (2026-04-22)

Implemented Phase 5 step executors with new expr/input packages, platform Exec support, and executor registry wiring. Added compensation execution + end-step terminal handling in the engine, plus 25+ executor/expr tests with race coverage. Validation gate: go build/vet/test all pass with -race.

## Learnings — Phase 6 Tool Runtime (2026-04-23)

Implemented Phase 6 tool runtime: pkg/tool interfaces, internal transports (stdio/jsonrpc/mcp), process management, registry/runtime, and ToolExecutor wiring. Added reference tool binaries + tests, built tool binaries in tests under repo-local .testtools (no /tmp), and stub .tool.yaml definitions. Full build/vet/tests including -race -count=3 pass.

## Learnings — Phase 7 (2026-04-24)

- Implemented extension host runtime with JSON-RPC handshake, health checks, discovery, and tool registry registration.
- Added dynamic tool registry Register() support plus test coverage for MapRegistry.
- Extended runbook schema to accept extensions entries (name, path, grants) so the r18 extension host fixture parses.
- Test extension binaries now build under repo-local .testextensions to avoid /tmp writes.

## Learnings — Phase 8 (2026-04-24)

- Added the input provider framework (pkg/input value providers, internal/input providers + registry) with unit tests.
- Renamed interactive prompt interface to PromptProvider and updated executors/test fakes to preserve choice/decision/collector flows.
- Engine now passes extension manifests to ExtensionHost.Load, shuts down the host on run completion, and schema accepts inputs.fallback/outcome.summary.

## Learnings — 2026-04-20

- Replaced condition evaluation in v2 with expr-lang/expr infix syntax while keeping TemplateEvaluator for string interpolation.
- Updated condition tests and executor iterate condition test to infix expressions, including boolean enforcement.
- Migrated runbook fixture condition/when clauses in design/gert-v2/testdata to infix syntax (plus assessment notes) and added expr dependency.
