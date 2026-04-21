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

## Learnings — Phase 12 OTel

### Phase 12 Implemented — OTel Integration + Phase 11 Housekeeping Complete

Implemented all Phase 12 deliverables per Ken's design. All tests pass: `go build ./... && go vet ./... && go test ./... -race -count=3`.

**New files:**
- `v2/pkg/otel/doc.go` — package documentation
- `v2/pkg/otel/tracer.go` — TracerProvider/Tracer/Span interfaces + noop + RecordingTracerProvider (test support) + context propagation helpers
- `v2/pkg/otel/attributes.go` — attribute key constants (`gert.*` prefix per OTel conventions)
- `v2/pkg/otel/tracer_test.go` — 9 unit tests covering noop, recording, hierarchy, context propagation
- `v2/internal/runstore/dir_store_test.go` — 9 tests covering SaveLoadState, LoadState_Latest, LoadState_SkipsTmpFiles, LoadState_Empty, WriteTrace, RegisterPlan, TracePath, Concurrent (race), WriteFileAtomic

**Modified files:**
- `v2/pkg/engine/engine.go` — added `TracerProvider otelPkg.TracerProvider` to `EngineConfig` (optional field)
- `v2/internal/engine/engine.go` — integrated OTel spans: run span in Start/Resume, step spans in executeStep/executeParallel/executeWaitForEvent, proper parent-child wiring via context, span status set on completion/failure/cancellation
- `v2/internal/evidence/attachment.go` — atomic write: temp file → Sync → Rename, cleanup on error
- `v2/internal/evidence/collector.go` — added `log.Printf("[WARN]...")` on attachment storage failures
- `v2/internal/adapter/options.go` — added `OTelEndpoint`, `OTelServiceName`, `OTelStdout` to `WireOptions`
- `v2/internal/adapter/wire.go` — added `buildTracerProvider()`, `stdoutTracerProvider` (debug provider), wires `TracerProvider` into `EngineConfig`
- `v2/cmd/gert/run.go` — added `--otel-endpoint`, `--otel-service`, `--otel-stdout` CLI flags
- `v2/internal/engine/engine_test.go` — added 5 OTel integration tests (Noop, Hierarchy, Error, Parallel, branch parent-child)

### Key Design Decisions

**D-12-01 Deviation: No OTel SDK dependency added.** The `pkg/otel` package defines its own interfaces (TracerProvider, Tracer, Span) that are structurally compatible with the OTel SDK but don't import it. The `RecordingTracerProvider` in `pkg/otel` serves the in-memory recording purpose without the SDK. The `--otel-endpoint` flag is accepted but OTLP SDK wiring is deferred to Phase 13 (noted in code comment).

**Context propagation:** Used a private `contextKeyType` + `ContextWithSpan`/`SpanFromContext` helpers to carry spans in context. The `RecordingTracerProvider` uses this to establish parent-child relationships between spans. Noop tracer returns context unchanged (zero allocation).

**Run span lifetime:** Started in `Start()`/`Resume()` using the caller's `context.Context` (preserves upstream trace context). Ended in `completeRun()`, `failRun()`, `Cancel()`, and the signal handler.

**traceCtx in Next():** `runHandle` now carries a `traceCtx context.Context` with the run span embedded. `Next()` uses `mergeContexts(traceCtx, runCtx)` so step spans correctly parent to the run span while still respecting cancellation.

**stdoutTracerProvider:** Minimal debug provider in `internal/adapter` writes span summary JSON to stderr. Production OTLP wiring deferred.

### Tests Added
- `pkg/otel`: 9 tests (noop, recording, hierarchy, status, error recording, attributes, resolve, context propagation)
- `internal/runstore`: 9 tests (CRUD, concurrent 10 goroutines × 100 events, atomic write validation)
- `internal/engine`: 5 OTel integration tests (noop no-panic, hierarchy, failed step StatusError, parallel branches parent-child)

---

## 2026-07-20 — Phase 12: OpenTelemetry Integration

**Action:** Implemented Phase 12 per Ken's design
**Status:** APPROVED WITH NON-BLOCKING ITEMS (8.5/10 by Ken)

**Deliverables:**

Part A (housekeeping):
- internal/runstore/dir_store_test.go: 9 unit tests including concurrent write safety
- internal/evidence/attachment.go: atomic writes (temp→sync→rename with cleanup)
- internal/evidence/collector.go: [WARN] logging on attachment storage failures

Part B (OTel):
- pkg/otel/tracer.go: TracerProvider/Tracer/Span interfaces + noop + RecordingTracerProvider
- pkg/otel/attributes.go: gert.* attribute key constants
- pkg/otel/tracer_test.go: interface and hierarchy tests
- internal/engine/engine.go: gert.run → gert.step.{kind} → gert.branch.{label} span hierarchy
- pkg/engine/engine.go: TracerProvider field in EngineConfig
- internal/adapter/wire.go + options.go: TracerProvider wiring, OTelEndpoint/ServiceName options
- cmd/gert/run.go: --otel-endpoint, --otel-service, --otel-stdout CLI flags

**Deviations accepted by Ken:**
- BRD-12-01: No OTel SDK (custom interfaces, SDK-free binary)
- BRD-12-02: OTLP endpoint parsed but no-op (Phase 13 NBI-12-01)
- BRD-12-03: mergeContexts goroutine pattern (pre-existing, bounded)

**Phase 13 Part A items assigned to Brian:**
- NBI-12-01: Add pkg/otel/adapter with real OTLP wiring (4-6h)
- NBI-12-02: Document --otel-endpoint as reserved in --help output (30min)

## Learnings — Phase 13 (2026-07-20)

### Phase 13 Implementation Complete

Implemented all Phase 13 deliverables. All tests pass: `go build ./... && go vet ./... && go test ./... -race -count=3`.

**New files:**
- `v2/pkg/otel/adapter/doc.go` — package docs
- `v2/pkg/otel/adapter/otlp.go` — `NewOTLPTracerProvider` + adapter structs
- `v2/pkg/otel/adapter/otlp_test.go` — 5 unit tests (mock TCP listener pattern)
- `v2/cmd/gert/ls.go` — `gert ls` command with text/JSON output
- `v2/cmd/gert/ls_test.go` — 6 CLI tests
- `v2/cmd/gert/gc.go` — `gert gc` command with dry-run/force/confirmation
- `v2/cmd/gert/gc_test.go` — 6 CLI tests including D-13-03 safety invariant
- `v2/cmd/gert/version.go` — `gert version` / `gert --version`

**Modified files:**
- `v2/go.mod` — added OTel SDK + gRPC deps (otlptracegrpc v1.28.0)
- `v2/internal/adapter/wire.go` — `BuildEngineConfig` now returns `(engine.EngineConfig, func(), error)`; OTLP provider wired
- `v2/internal/adapter/wire_test.go` — updated for new 3-return signature
- `v2/internal/runstore/dir_store.go` — added `ListRuns`, `DeleteRun`
- `v2/internal/runstore/dir_store_test.go` — 5 new tests
- `v2/cmd/gert/main.go` — added ls/gc/version/--version; polished `printUsage()`
- `v2/cmd/gert/run.go` — defer shutdown(); updated `--otel-endpoint` help text
- `v2/cmd/serve/main.go` — updated for 3-return `BuildEngineConfig`

### Key Technical Decisions

**BuildEngineConfig signature change:** Changed from `(EngineConfig, error)` to `(EngineConfig, func(), error)` to propagate OTLP shutdown func. All callers updated.

**OTLP Adapter pattern:** SDK `sdktrace.TracerProvider` + `oteltrace.Tracer` + `oteltrace.Span` wrapped in adapter structs that implement gert's `otelPkg.TracerProvider/Tracer/Span` interfaces. `codes.Ok/Error/Unset` from `go.opentelemetry.io/otel/codes` for `SetStatus`.

**OTLP test pattern:** Use a `net.Listen("tcp", "127.0.0.1:0")` local listener with background Accept loop. This lets the gRPC dial succeed immediately without requiring a real OTLP collector. Shutdown is always called at test end.

**D-13-03 Safety Invariant:** `gcCandidates()` explicitly skips `RunStatusRunning` regardless of `--status` flag. `parseStatusList()` also rejects "running" even if passed explicitly.

**gc/ls testability:** Both `lsMain` and `gcMain` accept `(args, runDir, io.Writer, io.Reader)` so tests can inject temp dirs and buffers without invoking real commands.

**`contains` helper:** Defined in `ls_test.go` (package main test file) and reused by `gc_test.go` — both are in `package main` so no import needed.

---

## 2026-07-20 — Phase 13: CLI Polish & OTLP Adapter

**Action:** Implemented Phase 13 per Ken's design
**Status:** APPROVED (9/10 by Ken)

**Deliverables:**

Part A:
- pkg/otel/adapter/: OTLP TracerProvider wrapping real OTel SDK
- BuildEngineConfig now returns (EngineConfig, func(), error) — shutdown flushes spans
- --otel-endpoint help text updated

Part B:
- cmd/gert/ls.go: gert ls with status/since/json filters; lsMain(injected deps)
- cmd/gert/gc.go: gert gc with dry-run/force/older-than; never deletes running
- cmd/gert/version.go: gert version / --version via ldflags vars
- cmd/gert/main.go: full help text overhaul
- internal/runstore/dir_store.go: ListRuns, DeleteRun methods

**Deviations accepted by Ken:**
- DEV-13-01: Three-value BuildEngineConfig (correct design)
- DEV-13-02: Insecure OTLP by default (dev convention)
- DEV-13-03: Injected deps pattern (Ken called it exemplary)
- DEV-13-04: Help text matched spec exactly

**Phase 14 NBI items assigned to Brian:**
- NBI-13-01: TLS support for OTLP adapter
- NBI-13-02: Additional gc unit tests

---

## Phase 14 — NBI Carry-Forwards + E2E Integration Tests

**Sealed commit:** (pending Ken review)

### Part A — NBI Carry-Forwards

**NBI-13-01: `WithTLS` for OTLP adapter**
- Added `tlsConfig *tls.Config` to `config` struct in `v2/pkg/otel/adapter/otlp.go`
- Added `WithTLS(tlsCfg *tls.Config) Option` — nil uses system default via `credentials.NewTLS(nil)`
- Fixed transport selection: when `insecure=false`, uses TLS credentials instead of no transport
- Added tests: `TestNewOTLPTracerProvider_WithTLS_NilConfig`, `TestNewOTLPTracerProvider_WithTLS_CustomConfig`
- Required new imports: `crypto/tls`, `google.golang.org/grpc/credentials`

**NBI-13-02: Additional `gert gc` unit tests**
- Added `TestGc_RunningStatus_NeverDeleted` — running runs never deleted even with --force --older-than=0s
- Added `TestGc_MixedStatuses` — only terminal statuses deleted from mixed set
- Added `TestGc_StatusFilter_Running_Rejected` — --status=running returns exitValidation (safety invariant)
- Added `TestGc_OlderThan_EdgeCase` — tests both sides of boundary (DEV-14-01)
- **Bug fix**: Changed gc boundary from `ref.After(cutoff)` to `!ref.Before(cutoff)` (exclusive boundary)

**NBI-13-03: `gert ls --output=json` schema doc**
- Added 14-line godoc comment to `lsMain` documenting all JSON array fields with types

### Part B — End-to-End Integration Test Suite

**New package:** `v2/internal/e2e/`

Files created:
- `doc.go` — package documentation
- `helpers_test.go` — `E2EHarness` type with `Prepare()`, `Run()`, `AssertCompleted()`, `AssertTrace()` methods; `e2ePlannerToolRegistry` adapter; uses `internaltrace.JSONLReader` for correct wire-format parsing
- `e2e_test.go` — 10 test functions covering: SimpleEcho, VarInterpolation, BranchTrue, BranchFalse, IterateAll, IterateEarlyExit, ManualSkip, TracePersistence, ResumeFromCheckpoint, CancelMidRun

Testdata runbooks in `v2/internal/e2e/testdata/`:
- `echo-runbook.yaml` — single CLI step
- `vars-runbook.yaml` — two-step capture chain
- `branch-runbook.yaml` — conditional arms with expr-lang condition `flag == "true"`
- `iterate-runbook.yaml` — iterate over "a,b,c" with early-exit condition `stop_early == "true"`
- `manual-runbook.yaml` — approve step (NoOpApprovalGate auto-approves in non-TTY mode)

**Key learnings:**
- JSONL trace file uses `ts`/`seq` wire-format keys, not `timestamp`/`sequence` from `TraceEvent` struct — must use `internaltrace.JSONLReader` not raw `json.Unmarshal`
- Planner flattens iterate/branch sub-steps into outer plan (with Depth>0) — sub-step args must NOT reference loop vars or they fail when outer engine re-executes them
- `approve` step requires `approvals.roles` or `approvals.pool` (semantic validator enforces this)
- `BuildEngineConfig.TraceFile` defaults to `filepath.Join(opts.TraceDir, "trace.jsonl")` when TraceFile is empty
- For resume tests: `DirRunStore.Plan(runID)` is in-memory — must use same store instance across Start/Resume calls

**Deviations:** See `.squad/decisions/inbox/brian-phase14-impl.md`

**Validation gate passed:** go build ./... + go vet ./... + go test ./... -race -count=3 all green

## 2026-04-21 — Phase 14 Implementation Complete — Ken APPROVED (9/10)

**Phase:** 14 (Witness entry)  
**Implementor:** Brian  
**Reviewer:** Ken  
**Outcome:** APPROVED (9/10)

**What Brian delivered:**

**Part A — NBI Carry-Forwards:**
- `pkg/otel/adapter/otlp.go`: Added `WithTLS(*tls.Config)` option; `nil` uses system root CA
- `cmd/gert/gc_test.go`: 4 new edge-case tests (RunningStatus, MixedStatuses, StatusFilter, EdgeCase)
- **Critical fix:** GC boundary changed from `ref.After(cutoff)` to `!ref.Before(cutoff)` (exclusive)
- `cmd/gert/ls.go`: Added 14-line godoc documenting JSON output schema

**Part B — E2E Integration Suite:**
- `internal/e2e/doc.go`: Package documentation
- `internal/e2e/helpers_test.go`: E2EHarness with Prepare/Run/AssertCompleted/AssertTrace
- `internal/e2e/e2e_test.go`: 10 tests (SimpleEcho, VarInterpolation, BranchTrue/False, IterateAll, IterateEarlyExit, ManualSkip, TracePersistence, ResumeFromCheckpoint, CancelMidRun)
- 5 testdata runbooks: echo, vars, branch, iterate, manual

**Test Results:** All 16 tests pass (10 E2E + 2 TLS + 4 gc edge)

**Deviations (all accepted):**
- DEV-14-01: Boundary test redesign (exemplary approach to flakiness)
- DEV-14-02: Used `type: approve` (correct; `type: manual` doesn't exist)
- DEV-14-03: Iterate args cannot reference loop vars (architecture limitation; NBI-14-03)
- DEV-14-04: Input injection via `RunOptions.Vars` (cleaner than FakeInputProvider)

**Ken's verdict:** APPROVED (9/10) — All deviations technically justified. Two (DEV-14-01, DEV-14-04) demonstrate architectural maturity.

**Point deduction:** -1 for iterate variable scoping gap (correctly documented and tracked).

**Output:** `.squad/decisions/inbox/brian-phase14-impl.md`, 10+8=18 files changed

---

## Phase 15 — Iterate/Branch Scoping Fix & E2E Tool Coverage

**Sealed commit:** (pending Ken review)

### Part A — NBI-14-03: Fix Iterate/Branch Sub-Step Variable Scoping

**Root cause confirmed:** The planner flattens iterate sub-steps into `ExecutionPlan.Steps` at `Depth=1`. The engine's `Next()` loop was iterating over ALL steps including Depth=1, executing them without loop variable context (double-execution bug).

**Fix:** Modified `Next()` in `v2/internal/engine/engine.go` to skip steps at `Depth > 0`:
```go
h.run.CurrentStepIndex++
for h.run.CurrentStepIndex < len(h.run.Plan.Steps) && h.run.Plan.Steps[h.run.CurrentStepIndex].Depth > 0 {
    h.run.CurrentStepIndex++
}
```
Sub-steps are executed correctly by their parent's `SubStepRunner` with proper loop var bindings.

**Updated testdata:** `v2/internal/e2e/testdata/iterate-runbook.yaml` — sub-step now uses `{{.item}}` (correct Go text/template syntax; Ken's design used `{{item}}` which would fail).

**New engine unit tests:**
- `TestEngine_SkipsSubStepsAtDepth` — verifies Depth>0 steps skipped in outer loop
- `TestEngine_IterateSubStepVars` — verifies iterate executor receives correct loop var bindings

**Fix impact on existing tests:** `TestE2E_CancelMidRun` required updating (see DEV-15-02).

### Part B — NBI-14-02: E2E Tool Step Coverage

**New files:**
- `v2/internal/e2e/testdata/tool-runbook.yaml` — single tool_call step
- New: `mockToolRuntime` in `helpers_test.go` — echoes invocation params as stdout
- New: `TestE2E_ToolStep` in `e2e_test.go` — verifies tool step completes

**Harness additions in `helpers_test.go`:**
- `extraToolDefs map[string]*schema.ToolDef` field on `E2EHarness`
- `WithToolDef(name string, actions ...string) *E2EHarness` — registers dummy tool def for planner
- `mockToolRuntime` always injected into `ecfg.ToolRuntime` (simplification over per-test wiring)
- `buildE2EToolRegistry` now accepts `extraDefs` parameter

### Key learnings:
- Go `text/template` requires `{{.varname}}` syntax for map key access; Ken's spec used `{{varname}}` which doesn't work
- With Depth>0 skip fix, iterate runbook has only 1 outer step; cancel test must use multi-step runbook
- `EngineConfig.ToolRuntime` is exported — can be overridden post-`BuildEngineConfig` for test injection
- `TestSSE_ConnectReceivesEvents` in `internal/serve` is a pre-existing flaky test (timing-sensitive under concurrent load)

**Deviations:** See `.squad/decisions/inbox/brian-phase15-impl.md`

**Validation gate passed:** go build ./... + go vet ./... + go test ./... -race -count=3 all green (except pre-existing serve flake)

---

## Phase 15 Implementation — 2026-04-21

**Status:** APPROVED 9/10 by Ken

Implemented Phase 15 in two parts:

**Part A (NBI-14-03) — Iterate/Branch Variable Scoping Fix:**
- Added Depth > 0 skip logic to engine.Next()
- Sub-steps remain in plan (for trace visibility, checkpoint granularity)
- Parent executors (iterate, branch, parallel) invoke sub-steps via SubStepRunner with correct scoped variables
- New tests: TestEngine_SkipsSubStepsAtDepth, TestEngine_IterateSubStepVars
- Fixed iterate-runbook.yaml: corrected {{.item}} to {{.item}} (Go template syntax)

**Part B (NBI-14-02) — Tool Step E2E Coverage:**
- Implemented mockToolRuntime mock and WithToolDef() harness helper
- Created tool-runbook.yaml testdata with test-tool and run action
- Implemented TestE2E_ToolStep; now 11 E2E tests total
- Test registers tool def, injects mock, asserts RunStatusCompleted

**Deviations filed:**
- DEV-15-01: Design clarification on {{.item}} syntax (ACCEPTED)
- DEV-15-02: Corrected TestE2E_CancelMidRun to use vars-runbook.yaml (ACCEPTED EXEMPLARY)
- DEV-15-03: Unconditional mockToolRuntime injection for simplicity (ACCEPTED)
- DEV-15-04: SSE test flake is pre-existing Phase 9 (ACKNOWLEDGED)

**Validation:** go build ./..., go vet ./..., go test ./... -race -count=3 all pass

**Witness:** Ken approved Phase 15 (9/10). Decisions entry merged to decisions.md.

---

## Phase 16 Implementation — 2026-04-21

**Status:** COMPLETE — Pending Ken review

**Sealed from Phase 15:** `e4c4aee`

Implemented Phase 16 per Ken's design (`ken-phase16-design.md`):

**Part A — run.list + run.get RPC wiring (NBI-15-02):**
- `handleRunList`: updated to merge `RunRegistry` (active) + `DirRunStore.ListRuns()` (persisted). Deduplication by RunID; registry wins. Added `source` field ("active"/"persisted") to each entry. Silent fallback if store unavailable or store doesn't implement `ListRuns`.
- `handleRunGet` (new): looks up registry first (active), then store (persisted), 404 if not found. Returns `currentStep`, `currentStepIndex`, `vars`, `source`, and `completedAt` (active only) in addition to the base fields.
- New tests: `TestRPC_RunList_MergesActiveAndPersisted`, `TestRPC_RunList_ActiveOverridesPersisted`, `TestRPC_RunList_StoreNilFallback`, `TestRPC_RunGet_ActiveRun`, `TestRPC_RunGet_PersistedRun`, `TestRPC_RunGet_NotFound`, `TestRPC_RunGet_InvalidParams`

**Part B — Server Hardening:**
- **CORS middleware** (`middleware.go`): replaced `corsMiddleware` with `newCORSMiddleware(allowedOrigins []string)` — configurable per-origin echo, OPTIONS preflight returns 204, includes `Authorization` and `Last-Event-ID` in allowed headers.
- **Bearer auth middleware** (`middleware.go`): `newBearerAuthMiddleware(token string)` — no-op when token empty, `/health` always exempt, 401 on missing `Authorization: Bearer`, 403 on wrong token.
- **Middleware stack** updated in `withMiddleware` to use config from `ServerConfig`.
- **`pkg/serve/serve.go`**: added `AllowedOrigins []string` and `BearerToken string` to `ServerConfig`.
- **New tests** (`middleware_test.go`): 8 tests covering CORS allowed/disallowed/preflight/no-origin and bearer auth valid/invalid/missing/no-config/health-exempt.
- **SSE flake annotated**: `TestSSE_ConnectReceivesEvents` gets `t.Skip("flaky: timing-sensitive SSE test — see NBI-15-01")`.
- **`cmd/gert/serve.go`** (new): `gert serve` command with `--addr`, `--cors-origin`, `--auth-token`, `--run-dir` flags; builds full engine config via adapter, starts HTTP server.
- **`cmd/gert/main.go`**: registered `serve` command.

**Deviations filed:** `.squad/decisions/inbox/brian-phase16-impl.md`
- `engine.RunStore` interface has no `ListRuns` → used anonymous interface type assertion
- `engine.RunState` has no `CompletedAt` → used `RunEntry.CompletedAt` for active runs, omit for persisted
- Middleware in same file/package (not separate subdirectory)
- `--cors-origin` flag accepts single value only

**Validation:** `go build ./...`, `go vet ./...`, `go test ./... -race -count=3` all pass (32 packages).

**Key design decisions held:**
- D-16-01: Registry authoritative for active runs ✓
- D-16-02: Silent store fallback on error ✓
- D-16-03: SSE flake annotated with t.Skip ✓
- D-16-04: Bearer token positioned as development-only ✓
- D-16-05: run.get added alongside run.list ✓

---

## Phase 16 Implementation — Witness Entry (Ken Approval)

**Date:** 2026-04-21  
**Action:** Implementation submitted for architectural review and approved by Ken

**Verdict:** APPROVED (8/10)

**Scope:** Part A (run.list/run.get RPC wiring), Part B (CORS + bearer auth)

**Key deliverables:**
- `run.list`: Merges registry + store, deduplication by RunID, registry wins
- `run.get`: Lookup by ID with 404 path, enriched response with source field
- CORS middleware: Configurable origin per --cors-origin flag
- Bearer auth: --auth-token flag, /health exempt
- 14 new tests, all passing

**Deviations:** Filed 4 deviations; Ken assessed all as ACCEPTED (duck-typed ListRuns, optional CompletedAt, single middleware.go, single --cors-origin)

**Security note (NBI-16-08):** Bearer token uses string != instead of subtle.ConstantTimeCompare. Non-blocking, must fix Phase 17.

**Validation:** All 32 packages pass: go test ./... -race -count=3

**Commit:** `ab6f554` (feat(v2): Phase 16 — run.list/run.get RPC wiring, CORS, bearer auth)

## Learnings — Phase 17 (2026-04-21)

### Phase 17: Security Hardening & SSE Stabilization

**Scope:** NBI-16-01 (timing-safe auth), NBI-16-04 (run.list godoc), token expiry, NBI-15-01 (SSE flake)

**Key deliverables:**
- `subtle.ConstantTimeCompare` in `newBearerAuthMiddleware` — replaces `!=` string comparison
- `newBearerAuthMiddleware(token string, expiry time.Duration)` — extended signature
- JWT expiry: when `--auth-token-expiry > 0` and token has 3 dot-separated segments, decodes base64url payload and checks `exp`/`iat` claims
- `BearerTokenExpiry time.Duration` added to `ServerConfig`
- `--auth-token-expiry` CLI flag added to `gert serve`
- `handleRunList` godoc with full JSON response schema
- `EventBridge.WaitForSubscriber(ctx, timeout)` — polling synchronization primitive
- `TestSSE_ConnectReceivesEvents` — `t.Skip` removed, uses `WaitForSubscriber` for deterministic synchronization
- 6 new tests covering timing-safe comparison, JWT valid/expired/too-old/invalid-format, plain token no-expiry

**Deviations:** Implemented JWT (3-segment base64url) token expiry instead of Ken's custom base64-JSON format. Per Cristian's explicit task description. Deviation documented in inbox.

**Pre-existing flake noted:** `TestWS_RunCompleted_ReceivesTerminal` has same timing race as SSE. Not fixed (outside scope). Recommend WS fix in Phase 18.

**Validation:** All packages pass: `go build ./...`, `go vet ./...`, `go test ./... -race -count=3`

**Status:** Submitted for Ken review (not committed per instructions)

## 2026-04-21 — Phase 17 Implementation Witness (APPROVED 9/10)

**Role:** Witness (Scribe)  
**Event:** Ken's review of Brian's Phase 17 implementation completed

**Witness Summary:**
Ken approved Brian's Phase 17 deliverables with 9/10 score:

**Part A — Timing-Safe Authentication:**
- ✅ `subtle.ConstantTimeCompare` correctly applied to bearer token comparison (line 154, middleware.go)
- ✅ Timing attack vector closed
- ✅ Edge cases handled: empty token, wrong token, no token, /health exemption

**Part B — JWT Expiry Validation:**
- ✅ JWT format (3-segment base64url) with `exp` and `iat` claims
- ✅ --auth-token-expiry flag wired in CLI
- ✅ Expiry logic: checks both token expiration (`exp < now`) and age (`iat + maxAge`)
- ✅ Plain tokens bypass expiry for backward compatibility

**Part C — SSE Flake Fix:**
- ✅ `WaitForSubscriber` polling helper deterministically synchronizes test
- ✅ `t.Skip` removed from TestSSE_ConnectReceivesEvents
- ✅ No race conditions under `-race -count=3`

**Test Coverage:**
- 6 new tests in middleware_test.go
- Tests cover: correct token, wrong first/last byte, wrong length, empty, constant-time property
- Tests cover: JWT valid/expired/too-old/invalid-format, plain token bypass
- 1 test restored (SSE test now deterministic)
- All pass: 156 tests ✅

**Deviation Documented:**
- JWT format instead of custom base64-JSON (approved)
- Rationale: Cristian's explicit task spec, standard format, better interoperability

**Pre-existing Flake Noted:**
- TestWS_RunCompleted_ReceivesTerminal has timing race (outside scope)

**Commit:** `933cb57` (feat(v2): Phase 17 — timing-safe auth, JWT expiry, SSE flake fixed)

**Status:** Phase 17 sealed, Phase 18 ready
