# Squad Decisions

## Active Decisions

### 2026-03-17T00:00:00Z: Visual runbook editor direction
**By:** Gon
**What:** Prefer a hybrid editor architecture in the VS Code extension: structured visual model + synchronized YAML source, with schema/tool/provider-driven extension points.
**Why:** Fastest path to value using existing extension and serve infrastructure, while preserving compatibility with emergent tools/providers and existing YAML-first workflows.

### 2026-03-17T00:00:00Z: Scope boundary for MVP
**By:** Gon
**What:** Keep YAML as the canonical persisted format in MVP; avoid introducing a new persisted runbook representation.
**Why:** Minimizes migration risk, keeps CLI/engine unchanged, and allows gradual rollout with reversible UX changes.

### 2026-03-17T00:00:00Z: Visual runbook editor UX direction
**By:** Kurapika
**What:** Ship a hybrid visual authoring experience in the existing VS Code extension: guided form-first editing with a synchronized read-only/limited-edit tree graph for branches and iterate blocks; persist runbook data as schema-backed JSON object and render YAML only as output/export.
**Why:** Business users need low-friction authoring and guardrails, while engineers still need transparent structure and compatibility with existing runbook engine, schema validation, and execution workflows.

### 2026-03-17T00:00:00Z: Extensibility and safety defaults
**By:** Kurapika
**What:** Use schema + tool-definition introspection to render dynamic fields for unknown future tools/providers, and enforce governance by default with visible policy panels, deny/approval hints, and preflight validation gates before save.
**Why:** Gert supports evolving tool/provider ecosystems and strict governance requirements; static form layouts would become brittle and unsafe.

### 2026-03-17T03:05:32Z: Backend contracts for visual editor MVP
**By:** Killua
**What:** Add backend read APIs for editor metadata rather than changing runtime semantics: expose canonical schema bundle (runbook/tool/project), effective tool/provider catalogs, and validation diagnostics via serve/CLI.
**Why:** UI can be generated from backend-owned contracts, preserving YAML compatibility and avoiding hardcoded UI logic tied to specific providers/transports.

### 2026-03-17T03:05:32Z: Compatibility policy for visual editing
**By:** Killua
**What:** Keep YAML as canonical, preserve v0/v1 plus shorthand parsing, and apply non-destructive normalization when saving from editor (retain unknown ordering/comments where feasible, never rewrite unrelated sections).
**Why:** Existing hand-written runbooks and tool packs must keep working without migrations or forced format shifts.

### 2026-03-17T03:05:32Z: Minimal Go MVP changes
**By:** Killua
**What:** Implement small additive APIs only: schema export expansion (tool/project), serve endpoints for runbook/tool/provider metadata and diagnostics, and index/list helpers in schema/tools for discoverability.
**Why:** Enables a robust editor MVP quickly with low regression risk and no engine behavior changes.

### 2026-03-17T00:00:00Z: Azure integration model should use existing tool/provider abstractions
**By:** Leorio
**What:** For visual runbooks that mix CLI and Azure operations, model Azure actions as governed tool actions (JSON-RPC or MCP spawn transport) and provider-resolved inputs, not as a new core step type. Keep governance in shared runbook/tool policy fields (rules, redaction, deny env vars, requires approval) and expose them in editor UX as explicit risk/approval/redaction controls.
**Why:** Existing contracts already separate execution from transport and include approval/redaction/deny logic in runtime; extending via tool/provider schemas minimizes coupling and keeps enforcement centralized.

### 2026-03-17T00:00:00Z: Visual runbook editor quality strategy
**By:** Hisoka (QA Reviewer)
**What:**
- Treat schema parity and YAML round-trip fidelity as release blockers for the visual editor MVP.
- Require compatibility coverage across `runbook/v0` and `runbook/v1` in both Go validation and VS Code extension validation.
- Enforce reviewer gates for deterministic serialization, deep-reference integrity, and governance-safe editing before GA.
**Why:**
- The editor path introduces a second schema-validation and serialization surface. Without explicit gates, regressions can bypass CLI/runtime safety guarantees.

### 2026-03-18: Visual paradigm references — n8n and Logic Apps Designer
**By:** ormasoftchile (via Copilot)
**What:** Preferred design references for the visual editor are n8n and Azure Logic Apps Designer.
**Why:** User confirmed affinity after reviewing workflow UI references. These two guide visual language, interaction patterns, and UX decisions for the workflow view.

### 2026-03-18: Workflow graph as optional alternative view
**By:** ormasoftchile (via Copilot)
**What:** The execution viewer (RunbookPanel) supports both the current tree view AND a new workflow graph view. Users toggle between them; neither replaces the other.
**Why:** Preserves existing behavior while introducing the workflow paradigm as a discoverable alternative. Reduces adoption friction.

### 2026-03-18: Workflow graph direction confirmed
**By:** ormasoftchile (via Copilot)
**What:** Workflow graph view for runbook execution is confirmed as the right direction. Current SVG prototype is directionally correct — polish and iteration should follow.
**Why:** User explicitly confirmed: "this is the way."

### 2026-03-18: Workflow view is the default for both authoring and execution
**By:** ormasoftchile (via Copilot)
**What:** Both the authoring experience (RunbookEditorPanel) and the execution experience (RunbookPanel) should be workflow-graph-first. A tree view adds little value over editing YAML directly.
**Why:** The visual tool's value proposition is showing the workflow — conditions, branching paths, sequence — not the document structure.

### 2026-03-18: Form-first + synchronized tree map dual-panel design
**By:** Gon (Lead Architect)
**What:** Editor uses a dual-panel hybrid model: left panel (60%) form-based authoring, right panel (40%) synchronized read-only flow map. Top bar with validation/governance badge, save, preview diff, cancel.
**Why:** Kurapika's Phase 1 MVP flattened branch/conditional structures; this design preserves runbook structure while remaining form-first for business users.

### 2026-03-18: Workflow-first visual paradigm architecture
**By:** Gon (Lead Architect)
**What:** Tree data already carries all topology info needed for graph derivation (client-side transform). Two additive backend events needed for execution annotation: `event/branchResolved` and `event/iteratePassEnd`.
**Why:** No new data shape required for static topology. Execution path annotation needs minimal additive Go events, zero-cost if UI not listening.

### 2026-03-18: Safety guards for non-linear runbook editing
**By:** Killua (Backend Engineer)
**What:** Block add/remove/reorder operations when branches or iterates are present. Allow metadata-only editing when tree has complex structures. Accept YAML reformatting as MVP limitation.
**Why:** Non-linear structures are incompatible with linear step extraction/rebuild pattern; safety guards prevent data loss.

### 2026-03-18: TreeNode structure authority (Go schema is canonical)
**By:** Killua (Backend Engineer)
**What:** Go schema is the authority. TreeNode = {step?, iterate?, branches?}. Branches are structural containers, NOT a step type. Step.Branches (legacy) should not be used.
**Why:** TypeScript form was treating branch as a step type, causing schema mismatch. Clarification unblocks correct form design.

### 2026-03-18: Phase 1 visual runbook editor MVP
**By:** Kurapika (Frontend Engineer)
**What:** Implemented Phase 1 MVP with YAML as single source of truth, simplified flat tree model, form-first UI, client-side message passing. Limitation: does not preserve branch/routing structure.
**Why:** Fastest path to value for linear runbooks; branch preservation deferred to Phase 2.

### 2026-03-18: Tree-based runbook editor MVP (Phase 2)
**By:** Kurapika (Frontend Engineer)
**What:** Phase 2 will extend the form editor to support branches and iterate nesting (tree-based editing). New TreeNode structure for internal representation.
**Why:** Phase 1 was flat/linear only. Phase 2 adds full tree editing capability, preserving branch/conditional data.

---

## 2026-04-19T22:34:59Z: Phase 1 Parser Implementation Approved

**By:** Ken (Software Architect)  
**Status:** APPROVED  
**Priority:** CRITICAL (unblocks Phase 2)

**Decision:** Phase 1 parser implementation (`v2/internal/parser/`) is production-ready and approved to proceed to Phase 2.

**Review Summary:**
- All 10 evaluation criteria passed (two-phase validation, nested parallel, signal allow-list, path normalization, error types, step dispatch, ParsedRunbook completeness, test quality, package boundaries, vet clean)
- 14/14 tests passing
- `go vet ./...` clean (no warnings)
- All 10 runbook fixtures (r01–r10) parse successfully

**Evidence:**
- Read 6 parser files: `parser.go`, `unmarshal.go`, `validate_structural.go`, `validate_semantic.go`, `errors.go`, `parser_test.go`
- Ran `go test ./internal/parser/... -v -count=1` → **14/14 PASS**
- Ran `go vet ./...` → **clean**

**Non-Blocking Suggestions for Phase 2:**

| # | Location | Suggestion |
|---|----------|------------|
| S1 | `pkg/parser/parser.go:33` | Update interface comment: semantic validation now lives in parser |
| S2 | `validate_semantic.go:walkFlowNodes` | Detect nested flow-level `ParallelNode` |
| S3 | `unmarshal.go` switch | Add comment for `StepTypeExtension` pass-through |
| S4 | `validate_semantic.go:collectStepIDs` | Add `IterateNode.ID` and `ParallelNode.ID` to uniqueness check |
| S5 | `parser_test.go` structural tests | Assert specific error codes (not just `err != nil`) |
| S6 | `ParsedRunbook.Warnings` | Populate deprecated-field warnings in Phase 2+ |

**Rationale:**
- Two-phase ordering is unambiguous: structural errors short-circuit before semantic validation
- Locked decisions (nested parallel forbidden, signal allow-list, path normalization) correctly implemented
- Error types are structured (ValidationError with code, field, message) and caller-friendly
- Tests are spec-linked via testutil.Tag; 14/14 passing
- Package boundaries clean; internal/parser unexported, only Parser/ParsedRunbook/ParseWarning exported

**Impact:**
- Parser ready for integration into serve APIs (backend for CLI and editor)
- Unblocks Phase 2: Planner architecture design, engine integration, serve endpoint for validation diagnostics
- Unblocks Phase 2 parallel work: Brian (S2/S4/S5 fixes), Barbara (testutil schema types)

### 2026-04-15: No cross-repo dependencies in design/gert-v2
**By:** Gon (Lead Architect)
**What:** `/design/gert-v2` repo will NOT depend on `/gert/v2` or other cross-repo paths during build. All references are doc-only.
**Why:** Design repo is independent, artifact-focused; avoids tool lock-in and merge/deployment complexity during rapid iteration.

### 2026-04-18: Bug fix: r10 YAML shell quoting
**By:** John (Schema)
**What:** r10 fixture had shell-escaped quoting (`'"'"'` instead of `'`) in approvers list; fixed to plain single quotes.
**Why:** YAML parser was failing on shell syntax; real runbooks will not use shell quoting.

### 2026-04-19T21:32:56Z: Enhancement to r02 (Use Iterate Node)

**By:** Barbara (Integrations Specialist)  
**Status:** PROPOSED  
**Priority:** LOW (defer to post-Phase 5)

**Problem:** r02 prose describes 10-minute monitoring loop with 60 iterations but uses 17 sequential `cli` steps (workaround). True `iterate` step would align schema with prose.

**Recommendation:** Defer to post-Phase 5 (can implement after r11 written and iterate executor ready)

**Rationale:**
- Tests iterate semantics in real-world canary deployment scenario
- Reduces r02 verbosity (17 steps → 1 iterate with 3 nested)
- Aligns schema with prose intent

**Effort:** ~30 lines YAML refactor  
**Owner:** John (Schema) to refactor r02  
**Timing:** After r11 is written and iterate executor implemented (Phase 5)

---

## 2026-04-19T22:27:31Z: Phase 1 Parser Implementation Complete

**By:** Brian (Go Programmer)  
**Status:** APPROVED  
**Priority:** CRITICAL (blocking Phase 1 complete)

**Decision:** Phase 1 parser implementation (v2/internal/parser/) is production-ready.

**Deliverables:**
- 6 files: parser.go, unmarshal.go, validate_structural.go, validate_semantic.go, errors.go, parser_test.go
- 14/14 tests passing
- go build clean (no errors/warnings)
- Two-phase YAML decode strategy for schema.Step inline spec conflicts
- All 10 runbook fixtures (r01–r10) parse successfully
- JSON Schema validation via github.com/santhosh-tekuri/jsonschema/v6
- Semantic validation for runbook integrity

**Key Design:**
- Raw `yaml.Node` capture enables selective field decoding (avoids "duplicated key" panic)
- Never calls `yaml.Node.Decode()` on types transitively containing `schema.Step` or `schema.FlowNode`
- Type-discriminator-based spec unmarshaling (e.g., switch on step.type, unmarshal only matching spec)

**Schema Updates:**
- runbook.schema.json: 11 relaxations for fixture compatibility (GovernanceConfig, ToolRef, CollectorFieldType, Assertion, etc.)
- Go schema: 4 type updates (GovernanceRule, RedactRule, ToolRef.Source/Actions, ToolInvocation.Args any)
- Added StepTypeExtension = "extension"

**Rationale:**
- Two-phase decode prevents the yaml.v3 "duplicated key" panic that blocked v1 design
- Fixture compatibility ensures parser handles real runbooks reliably
- Clear error context aids debugging and UX

**Impact:**
- Parser can now be integrated into serve APIs (serves backend for CLI and editor)
- Unblocks Phase 2 (engine integration, serve endpoint for validation diagnostics)

---

## 2026-04-19T22:27:31Z: StepSpec Interface Ready

**By:** Ken (Architect)  
**Status:** APPROVED  
**Priority:** CRITICAL (blocking Phase 1 complete)

**Decision:** StepSpec interface and ResolvedStep.Spec type-safety integration is complete.

**Deliverables:**
- StepSpec interface with StepKind() method (v2/pkg/engine/stepspec.go)
- ResolvedStep.Spec: any → StepSpec (v2/pkg/engine/run.go)
- StepKind() implemented on all 14 step types (v2/pkg/schema/steps.go)

**Step Types Implementing StepKind():**
- CLISpec, ToolCallSpec, IncludeSpec, ChoiceSpec, DecisionSpec, CollectorSpec
- BranchSpec, IterateNode, ParallelNode
- ApproveSpec, AssertSpec, CompensateSpec, WaitForEventSpec, EndSpec

**Rationale:**
- Compile-time verification ensures all step types implement StepSpec
- Engine dispatch code now type-safe: `step.Spec.StepKind()` vs. reflection
- Adding new step types will fail at compile time if StepKind() missing

**Build Status:**
- `go build ./...` ✓
- `go vet ./...` ✓

**Impact:**
- Engine can now use StepSpec for type-safe dispatch logic
- Unblocks Phase 2 (ResolvedStep serialization and executor implementation)

---

## 2026-04-19: Parser Phase 1 Correctness Fixes (S1-S5)

**By:** Brian (Go Programmer)
**Status:** IMPLEMENTED — pending Ken review
**Scope:** `v2/internal/parser/`, `v2/pkg/parser/parser.go`

---

### S1 — Stale comment (cosmetic)

`v2/pkg/parser/parser.go` interface comment no longer claims semantic validation is "the Planner's job". The parser performs both structural (JSON Schema) and semantic validation.

---

### S2 — Flow-level ParallelNode nested in parallel branch

**Problem:** `walkFlowNodes` only checked `inParallel` for `schema.Step{Type:"parallel"}`. A flow-level `ParallelNode` (`fn.Parallel != nil`) inside a parallel branch was not caught.

**Fix:** Added `inParallel` guard in `walkFlowNodes` before delegating to `validateParallelNode`. Emits `parallel/nested-forbidden` for the outer ParallelNode when `inParallel == true`.

**New test:** `TestParser_NestedParallelNodeForbidden`

---

### S3 — StepTypeExtension silent pass-through (cosmetic)

Added explicit `case schema.StepTypeExtension:` with comment: "Extension steps are validated structurally only; their body is opaque to the parser." No behavior change.

---

### S4 — IterateNode.ID and ParallelNode.ID missing from uniqueness check

**Problem:** `collectStepIDs` recursed into `fn.Iterate.Steps` and `fn.Parallel.Branches` but never incremented `ids[fn.Iterate.ID]` or `ids[fn.Parallel.ID]`. `stepIDExists` already handled these IDs correctly — this was a gap only in the uniqueness collector.

**Fix:** Added `ids[fn.Iterate.ID]++` and `ids[fn.Parallel.ID]++` before recursing into children.

**New tests:** `TestParser_IterateNodeDuplicateID`, `TestParser_ParallelNodeDuplicateID`

---

### S5 — Structural error tests now assert specific codes

**Problem:** 8 structural tests checked only `err != nil`. `TestParser_BranchRequiresAtLeastOneArm` also lacked a code assertion.

**Fix:** Added `assertErrorCode(t, err, code)` helper that uses `errors.As` to unwrap `parser.ValidationErrors` and checks for a specific code. Applied to all 9 tests:
- 8 structural tests check for `schema/structural`
- `TestParser_BranchRequiresAtLeastOneArm` checks for `branch/no-arms`

---

### Build result

```
go build ./...   PASS
go vet ./...     PASS
go test ./internal/parser/... -v -count=1   23 tests PASS (was 20 before new tests)
```

**Impact:** Parser correctness improved. All specified gaps (S1–S5) resolved. New tests validate fixes. Ready for integration with Phase 2 Planner.

---

## 2026-04-19: testutil Real Types

**By:** Barbara (Integrations Specialist)  
**Date:** 2026-04-20  
**Status:** APPROVED (implemented, build clean)

## Context

Phase 0 created `v2/pkg/testutil/` with placeholder stubs because Brian's schema/engine types did not yet exist. Phase 1 delivered `pkg/engine`, `pkg/eventbus`, and `pkg/trace`. This decision records how the testutil package was updated to match the real interfaces.

## Decisions

### 1. FakeStepExecutor implements engine.StepExecutor

`engine.StepExecutor.Execute` takes `engine.ResolvedStep` (value, not pointer) and returns `*engine.StepResult`. The fake matches this exactly. Stub `Step` and `StepResult` types were removed.

A compile-time guard was added:
```go
var _ engine.StepExecutor = (*FakeStepExecutor)(nil)
```

### 2. FakeEventDispatcher implements eventbus.EventDispatcher

The real `eventbus.EventDispatcher` interface requires:
- `Dispatch(ev InboundEvent) error`
- `Wait(ctx, stepID, EventFilter, timeout) (*InboundEvent, error)`
- `Cancel(stepID, reason string)`

The stub `Event` and `EventFilter func(Event) bool` types were removed. All three methods now use real types. `Cancel` was added as a new method (was missing from Phase 0).

**Filter semantics:** `eventbus.EventFilter` is a struct (Source, ID, Payload map), not a predicate function. A `matchesFilter(ev, f)` helper was added that checks each non-empty field against the event.

**Cancellation signal:** `DrainAll` and `Cancel` send a nil `*eventbus.InboundEvent` on the delivery channel. Waiters check `ev == nil` to distinguish cancellation from a real event.

**Extra methods retained:** `WaitOnChannel`, `WaitersCount`, `DrainAll` are testutil-specific helpers not in the interface. They were kept and updated to use real types.

### 3. ConcurrentEventCollector collects trace.TraceEvent

The stub `CollectedEvent` struct was removed. The collector now stores `[]trace.TraceEvent` directly. `EventsForStep` extracts `step_id` from the `json.RawMessage` payload using a small unmarshal helper (`stepIDFromPayload`).

### 4. golden.go uses trace.TraceEvent

The stub `TraceEvent` type was replaced with `trace.TraceEvent`. Normalization field names updated: `.At` → `.Timestamp`, `.Seq` → `.Sequence`. The hand-rolled JSONL reader was replaced with `bytes.NewReader`.

## Impact

- Any test code referencing `testutil.Event`, `testutil.EventFilter` (func), `testutil.Step`, `testutil.StepResult`, `testutil.TraceEvent`, or `testutil.CollectedEvent` must be updated to use the real package types.
- No existing tests outside `pkg/testutil` were found to depend on these stub types.
- `go build ./...` and `go vet ./...` pass clean.

---

## 2026-04-19: Phase 2 Planner Design

**Date:** 2026-04-20
**Author:** Ken (Software Architect)
**Status:** APPROVED
**Priority:** CRITICAL (blocking Phase 2)

## Decision Summary

Phase 2 Planner architecture establishes interfaces for transforming `*parser.ParsedRunbook` into `*engine.ExecutionPlan`. The design introduces three key interfaces and a concrete stub implementation.

---

## Interface Design Decisions

### 1. Planner Interface Location

**Decision:** Keep `Planner` interface in `pkg/engine/planner.go`; place supporting interfaces (`RunbookLoader`, `ToolRegistry`, `Config`) in `pkg/planner/planner.go`.

**Rationale:**
- `engine.ExecutionPlan` is already in `pkg/engine/run.go`
- Avoids import cycles: planner depends on engine (for ExecutionPlan), not vice versa
- `engine.Planner` is the API the engine consumes; `planner.Config` is how you build one
- Implementation lives in `internal/planner/` — clean separation of interface from impl

### 2. Planner Interface Signature

**Before (old signature in engine/planner.go):**
```go
Plan(ctx context.Context, rb *parser.ParsedRunbook, opts PlanOptions) (*ExecutionPlan, error)
```

**After (simplified signature):**
```go
Plan(ctx context.Context, rb *parser.ParsedRunbook) (*ExecutionPlan, error)
```

**Rationale:**
- Configuration moved to `planner.Config` struct at construction time
- Planner is constructed once with Loader, Tools, BaseDir, MaxIncludeDepth
- Simplifies the call site — runtime just calls `p.Plan(ctx, rb)`
- Options that vary per-call (e.g., vars) can be added later without interface change

### 3. RunbookLoader Interface

```go
type RunbookLoader interface {
    Load(ctx context.Context, path string) (*parser.ParsedRunbook, error)
}
```

**Design decisions:**
- Returns `*parser.ParsedRunbook`, not raw bytes — loader wraps parser internally
- Path resolution is relative to loader's configured base directory
- Test doubles can return in-memory runbooks without filesystem
- Error types: `ErrRunbookNotFound` for missing files

### 4. ToolRegistry Interface

```go
type ToolRegistry interface {
    Lookup(ctx context.Context, name string, action string) (*schema.ToolDef, error)
}
```

**Design decisions:**
- Two-key lookup (name + action) rather than single composite key
- Returns full `*schema.ToolDef` so planner can validate args
- Error types: `ErrToolNotFound`, `ErrActionNotFound` (separate cases)
- Registry is pre-populated (scanned at construction, not lazy)

### 5. Config Struct

```go
type Config struct {
    Loader          RunbookLoader  // required
    Tools           ToolRegistry   // required
    BaseDir         string         // optional, defaults to runbook directory
    MaxIncludeDepth int            // optional, defaults to 10
}
```

**Rationale:**
- Required fields panic if nil — fail fast, don't return cryptic errors later
- Optional fields have sensible defaults
- No ProviderDir — provider resolution is tool-runtime concern, not planner concern

---

## ExecutionPlan Structure

The existing `engine.ExecutionPlan` structure is sufficient for Phase 2:

```go
type ExecutionPlan struct {
    RunID       string
    RunbookPath string
    Steps       []ResolvedStep
    Tools       map[string]*schema.ToolDef
    Providers   map[string]*schema.ProviderDef
    Governance  governance.GovernancePolicy
    Metadata    PlanMetadata
}
```

**Notes:**
- `Steps` is a flat ordered list (per §02 decision: ExecutionPlan is flat, not DAG)
- `Tools` map keyed by tool name — planner populates this during tool discovery
- `Providers` left empty for Phase 2 — provider resolution deferred to runtime
- `Governance` populated from runbook's governance config during planning

### ResolvedStep Structure

```go
type ResolvedStep struct {
    ID     string
    Kind   string
    Spec   StepSpec   // typed: *CLISpec, *ToolCallSpec, *IncludeSpec, etc.
    Depth  int        // nesting depth (0 = root runbook)
    Origin string     // source runbook path
}
```

**No changes needed** — the existing structure supports:
- Typed specs via StepSpec interface (Phase 1 deliverable)
- Include tracking via Depth and Origin fields
- Tool resolution via Spec (ToolCallSpec carries invocation details)

---

## Error Model

Structured errors with codes for programmatic handling:

```go
var (
    ErrNotImplemented   = errors.New("planner: not implemented")
    ErrToolNotFound     = errors.New("planner: tool not found")
    ErrActionNotFound   = errors.New("planner: action not found")
    ErrRunbookNotFound  = errors.New("planner: runbook not found")
    ErrImportCycle      = errors.New("planner: import cycle detected")
    ErrMaxDepthExceeded = errors.New("planner: max include depth exceeded")
)

type PlanError struct {
    Code   error   // one of the Err* sentinels
    StepID string  // optional: step that caused the error
    Path   string  // optional: runbook path involved
    Detail string  // human-readable context
}
```

**Rationale:**
- Sentinel errors enable `errors.Is(err, planner.ErrToolNotFound)`
- Wrapped errors preserve context chain
- StepID enables pointing users to the exact step that failed

---

## Deliverables

| File | Description |
|------|-------------|
| `v2/pkg/planner/doc.go` | Package documentation |
| `v2/pkg/planner/planner.go` | Planner, RunbookLoader, ToolRegistry interfaces; Config; error types |
| `v2/pkg/engine/planner.go` | Updated Planner interface (simplified signature) |
| `v2/internal/planner/planner.go` | Concrete stub implementation returning ErrNotImplemented |
| `v2/internal/planner/planner_test.go` | 6 skeleton test cases (all t.Skip) |

---

## Test Coverage (Skeleton)

| Test | Description |
|------|-------------|
| `TestPlan_BasicRunbook` | Single CLI step → single ResolvedStep |
| `TestPlan_IncludeResolution` | Parent includes child → child steps inlined |
| `TestPlan_ToolResolution` | Tool call → tool registered in plan.Tools |
| `TestPlan_ImportCycleDetection` | A includes B includes A → ErrImportCycle |
| `TestPlan_ToolNotFound` | Unknown tool → ErrToolNotFound |
| `TestPlan_MaxDepthExceeded` | 12-deep include chain → ErrMaxDepthExceeded |

---

## Build Verification

```
$ cd /Volumes/Projects/gert/v2 && go build ./...
# (clean exit)

$ go vet ./...
# (clean exit)

$ go test ./internal/planner/... -v
# 6 tests SKIP (Phase 2: not yet implemented)
```

---

## Impact

- Unblocks Phase 2 implementation (Brian can write concrete planner logic)
- Unblocks tool loading infrastructure (needs ToolRegistry implementation)
- Unblocks include resolution logic
- Runtime can now depend on engine.Planner interface

---

## Open Questions (deferred)

1. **Provider resolution** — deferred to runtime, not planner
2. **Governance policy merging** — how to merge parent + child runbook policies (deferred)
3. **Parallel planning** — should nested includes be resolved concurrently? (deferred to v2.1)
