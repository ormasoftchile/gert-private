# Ken — Phase 2 Architectural Review

**Date:** 2026-04-19  
**Reviewer:** Ken (Software Architect)  
**Implementor:** Brian  
**Integrations:** Barbara (testutil fakes)

---

## VERDICT: ⚠️ REJECTED

Phase 2 has strong foundations but contains **2 critical defects** that must be fixed before approval.

---

## Critical Defects

### 1. Missing Compile-Time Interface Guard (planner.go)

**File:** `v2/internal/planner/planner.go`  
**Issue:** No `var _ plannerPkg.Planner = (*impl)(nil)` guard exists.

The concrete type `impl` claims to implement `planner.Planner`, but there's no compile-time verification. If the interface changes, compilation will silently succeed but the constructor will panic at runtime.

**Required fix:**
```go
// After line 17 (after imports, before const)
var _ plannerPkg.Planner = (*impl)(nil)
```

**Assigned to:** Barbara (standard infrastructure pattern, quick fix)

---

### 2. Cycle Detection Uses Permanent Marking — Blocks Valid Diamond Dependencies

**File:** `v2/internal/planner/planner.go`, lines 260-268  
**Issue:** The `seen` map uses permanent marking (`pc.seen[inclPath] = true`), which is never cleared.

This incorrectly rejects valid diamond dependencies:

```
root.yaml
├── branch A: include shared.yaml   ← marks shared.yaml
└── branch B: include shared.yaml   ← REJECTED (but should be allowed!)
```

The spec allows the same runbook to be included in **different branches** (sibling references). Only **ancestor references** (A→B→A) should be forbidden.

**Current behavior:** Uses permanent marking for all paths  
**Required behavior:** Use temporary (ancestor-only) marking with backtracking, OR distinguish sibling vs. ancestor references

**Fix approach:**
```go
// Before recursing into child:
pc.seen[inclPath] = true
defer delete(pc.seen, inclPath)  // Backtrack after recursion completes
```

This implements "gray" marking (temporary during recursion) instead of "black" marking (permanent).

**Assigned to:** John (requires understanding of graph traversal semantics)

---

## Non-Critical Issues (Address in Phase 3)

### 3. Tests Missing `testutil.Tag` Links

**File:** `v2/internal/planner/planner_test.go`  
**Issue:** None of the 13 tests use `testutil.Tag()` to link to spec rules. Parser tests demonstrate the pattern:
```go
_ = testutil.Tag("03-schema-vnext.md", "§4.2", "step MUST have a type field")
```

**Severity:** Non-blocking (documentation quality)

### 4. No Test for Diamond Dependency (Different Branches)

**File:** `v2/internal/planner/planner_test.go`  
**Issue:** Missing test case for "same runbook included in sibling branches should succeed."

Once defect #2 is fixed, add this test to prevent regression.

### 5. `rawSpec` Fallback Indicates Schema Gap

**File:** `v2/internal/planner/planner.go`, lines 296-354  
**Issue:** The `rawSpec` fallback suggests malformed input could reach the planner. If the parser validates that every step has its typed spec populated, this fallback is dead code.

**Recommendation:** Either remove the fallback (parser guarantees spec is populated) or add explicit panic with "unreachable: parser should reject steps without typed spec" comment.

---

## What's Working Well

1. **Interface conformance** — `Plan()` signature matches `pkg/planner.Planner` and `pkg/engine.Planner` (both return `*engine.ExecutionPlan`)

2. **Max depth enforcement** — Depth is passed by value (`depth int`), not by pointer. Each branch of recursion gets its own counter. ✅ Correct.

3. **Tool lookup contract** — `resolveTool` calls `Lookup(ctx, name, action)` matching the interface signature. Brian's in-test fakes and Barbara's `FakeToolRegistry` both implement the same signature. ✅

4. **Error wrapping** — `PlanError.Unwrap()` returns `Code`, enabling `errors.Is()` matching:
   ```go
   errors.Is(err, plannerPkg.ErrImportCycle)  // works ✅
   ```

5. **ExecutionPlan completeness** — All required fields populated:
   - `RunbookPath` ✅
   - `Steps` ([]ResolvedStep) ✅
   - `Tools` map ✅
   - `Providers` (empty map, acceptable) ✅
   - `Metadata` (PlannedAt, RunbookID, RunbookName) ✅

6. **Topo sort** — Uses declaration order, which is correct per §02: "ExecutionPlan is a flat ordered list, not a DAG." Linear flow processing is the intended design.

7. **Build/vet clean** — `go test ./internal/planner/... -v` passes all 13 tests. `go vet ./...` reports no warnings.

8. **Barbara's fakes** — Both `FakeRunbookLoader` and `FakeToolRegistry` include compile-time interface guards. Professional quality.

---

## Verification Commands Run

```bash
cd /Volumes/Projects/gert/v2 && go test ./internal/planner/... -v -count=1
# Result: 13/13 PASS

cd /Volumes/Projects/gert/v2 && go vet ./...
# Result: No warnings
```

---

## Required Actions

| # | Defect | Assignee | Priority |
|---|--------|----------|----------|
| 1 | Add interface guard | Barbara | Critical |
| 2 | Fix cycle detection (allow diamond) | John | Critical |

Phase 2 remains **REJECTED** until both critical defects are resolved.

---

**Ken**  
Software Architect
# Phase 2 Re-Review Verdict

**Reviewer:** Ken (Software Architect)
**Date:** 2026-04-20
**Status:** ✅ **APPROVED**

## Context

Phase 2 was initially REJECTED with 2 defects:
- **D1:** Missing interface guard for `Planner` implementation
- **D2:** Cycle detection blocked valid diamond dependencies

Barbara fixed D1. John fixed D2. Brian (original author) did not touch these fixes.

## Verification Results

### Build & Test
```bash
cd /Volumes/Projects/gert/v2 && go build ./... && go vet ./... && go test ./internal/planner/... -v -count=1
```
**Result:** All 14 tests PASS (13 original + 1 new diamond test)

### D1 Verified ✅
**Interface guard present at line 19:**
```go
var _ plannerPkg.Planner = (*impl)(nil)
```
- Correctly placed at package level
- Uses correct concrete type `impl` (unexported)
- Uses correct interface `plannerPkg.Planner`
- Compiles successfully (go build passes)

### D2 Verified ✅
**DFS backtracking implemented correctly at lines 270-271:**
```go
pc.seen[inclPath] = true
defer delete(pc.seen, inclPath)
```
- `defer delete` appears immediately after `pc.seen[inclPath] = true`
- No intervening code between the two statements
- `defer` is inside `resolveInclude` function scope — correct behavior
- Pattern matches standard DFS backtracking (mark on entry, unmark on return)

### Diamond Test Verified ✅
**`TestPlanner_DiamondDependency` (lines 364-442):**
- Tests diamond: A→B→D and A→C→D (D included from two branches)
- Verifies no cycle error returned (`if err != nil { t.Fatalf(...) }`)
- Verifies plan correctness: expects 2 steps (d-step inlined twice)
- Test PASSES — diamond dependencies now correctly allowed

### Regression Check ✅
- All 13 original tests still pass
- `TestPlanner_ImportCycleDetected` and `TestPlan_ImportCycleDetection` both still detect true cycles (A→B→A)
- Backtracking fix does not break real cycle detection

## Verdict

**APPROVED** — Phase 2 complete.

Both defects correctly fixed. All 14 tests pass. Cycle detection now properly distinguishes true cycles from diamond dependencies.

## Non-blocking Suggestions for Phase 3

1. **Test traceability:** Add `testutil.Tag()` links in tests to trace back to spec sections
2. **rawSpec cleanup:** Consider removing `rawSpec` fallback or documenting when it's expected (parser should guarantee typed specs)
3. **Test coverage:** Consider adding more edge cases (triple-diamond, mixed diamond+cycle)

---
*Decision recorded by Ken, Software Architect*
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
# Schema Decisions: Fixtures r11–r13

**Author:** Barbara (Integrations Specialist)
**Date:** 2026-04-18
**Status:** Proposed

## Context

While creating fixtures r11, r12, r13 as specified by Phase 5, several schema
conventions were discovered that differ from the task brief's sketch YAML.
Recording these so the team has a shared reference.

---

## Decision 1: `iterate` and `parallel` are FlowNode discriminators, not step types

**Finding:** The task brief used `type: iterate` as a step type. This is wrong.
In v2, `iterate` and `parallel` are **FlowNode keys** (`- iterate: {...}` /
`- parallel: {...}`), not values of the `type` field on a `step:`.

The valid step `type` values are the 14 in `StepType`:
`cli`, `tool`, `include`, `choice`, `decision`, `collector`, `branch`,
`approve`, `assert`, `compensate`, `wait_for_event`, `end`, `extension`.

**Impact:** Any documentation or tooling that generates `type: iterate` or
`type: parallel` on a step will be rejected by both structural (JSON Schema)
and the parser's FlowNode dispatch.

---

## Decision 2: No `log` or `set` step types in v2

**Finding:** The task brief referenced `type: log` and `type: set`. Neither
exists in the v2 schema.

- **Logging:** Use `type: cli` with `command: echo` (or any shell command).
- **Variable setting:** Use `capture:` on a `cli` step to store stdout into a
  named variable.

The `collect:` map on `IterateNode` provides per-iteration variable accumulation.

---

## Decision 3: `approve` step uses `approvals:` (ApprovalGate), not `approve:`

**Finding:** The `ApproveSpec` struct wraps an `ApprovalGate` under key
`approvals:`. The quorum reviewer list is `pool: [...]` (not `reviewers:`),
and the quorum count is `required:` (not `quorum:`).

Correct form:
```yaml
- step:
    id: gate
    type: approve
    approvals:
      pool: [alice, bob, carol]
      required: 2
      timeout: 4h
      on_timeout: fail
```

**Semantic rule:** At least one of `roles` or `pool` must be non-empty, or the
parser rejects with `approve/no-reviewers`.

---

## Decision 4: `decision` routes with `goto:` are semantically validated

**Finding:** The semantic validator (`validate_semantic.go`) checks that every
`goto` value on a `DecisionRoute` references an existing step ID in the runbook
flow. Floating `goto` values cause `decision/invalid-goto` errors.

This means the decision step and its goto targets must all live in the same
runbook (cross-runbook routing uses `runbook:` instead of `goto:`).

---

## Fixture naming convention

All existing fixtures use `schema.yaml` as the filename (not `runbook.yaml`).
The glob pattern in `parser_test.go` is `r*/schema.yaml`. New fixtures must
follow this convention.

# Phase 3: Runtime Core Architecture Design

**Date:** 2026-04-20  
**Author:** Ken (Software Architect)  
**Status:** PROPOSED  
**Priority:** CRITICAL (blocking Phase 3 implementation)

---

## Summary

Phase 3 Runtime Core establishes the engine architecture for executing `*engine.ExecutionPlan` step by step. The design defines interfaces for the Engine, RunHandle, step execution dispatch, parallel branch coordination, wait_for_event semantics, and trace event protocol.

---

## Interface Design Decisions

### 1. Engine Interface and EngineConfig

**Location:** `pkg/engine/engine.go`

```go
type Engine interface {
    Start(ctx context.Context, plan *ExecutionPlan, opts RunOptions) (RunHandle, error)
    Resume(ctx context.Context, runID string, opts RunOptions) (RunHandle, error)
}

type EngineConfig struct {
    Executors   ExecutorRegistry       // required: step dispatch
    Dispatcher  eventbus.EventDispatcher // required: wait_for_event
    TraceWriter trace.TraceWriter      // required: event persistence
    Platform    platform.Platform      // required: signal handling
    EventBus    eventbus.EventBus      // optional: in-process fan-out
    Store       RunStore               // optional: checkpoint/resume
    OnEvent     func(Event)            // optional: event callback
}
```

**Rationale:**
- Engine is stateless; all state lives in RunHandle
- Four required dependencies: executors (dispatch), dispatcher (wait_for_event), trace writer (persistence), platform (signals)
- Optional dependencies enable additional features (event fan-out, checkpointing)
- `Validate()` method enforces required fields at construction

### 2. Run State Machine

**Location:** `pkg/engine/run.go`

```
[pending] → [running] → [completed]
              ↓
          [waiting] → [running]
              ↓
          [failed]
              ↓
          [cancelled]
```

**States:**
- `RunStatusPending`: Initial state before first `Next()` call
- `RunStatusRunning`: Actively executing steps
- `RunStatusWaiting`: Paused at wait_for_event or approval gate
- `RunStatusCompleted`: All steps executed successfully
- `RunStatusFailed`: Step or infrastructure failure
- `RunStatusCancelled`: User-requested cancellation

**Run struct:**
```go
type Run struct {
    ID               string
    Status           RunStatus
    Plan             *ExecutionPlan
    Vars             map[string]any        // runtime variables
    StepResults      map[string]*StepResult
    CurrentStepIndex int                   // -1 = not started
    StartedAt        time.Time
    CompletedAt      time.Time
    Error            error
    Sequence         int64                 // monotonic event counter
    Actor            string
    Mode             RunMode
}
```

**Rationale:**
- Separate `Run` (internal mutable state) from `RunState` (external snapshot)
- `Sequence` enables strict event ordering across parallel branches
- `CurrentStepIndex` into flat `Plan.Steps` keeps execution deterministic

### 3. Step Execution Interface

**Location:** `pkg/engine/executor.go`

```go
type StepExecutor interface {
    Execute(ctx context.Context, step ResolvedStep, vars map[string]any) (*StepResult, error)
}

type ExecutorRegistry interface {
    Register(kind string, exec StepExecutor)
    Lookup(kind string) StepExecutor
}
```

**StepResult:**
```go
type StepResult struct {
    StepID      string
    Status      StepStatus    // pending/running/completed/failed/skipped/waiting
    Outcome     StepOutcome   // deprecated alias
    Output      map[string]any
    Vars        map[string]any
    StartedAt   time.Time
    CompletedAt time.Time
    DurationMs  int64
    Error       error
}
```

**Contract:**
- Return non-nil `*StepResult` on success (even for skipped)
- Return error only for infrastructure failures (not step logic failures)
- Set `StepResult.Status = StepStatusFailed` for step logic failures
- Executors MUST NOT modify `vars`; output goes in `StepResult.Vars`

### 4. Parallel Execution Model

**Location:** `pkg/engine/engine.go`

```go
type BranchExecutor interface {
    ExecuteBranches(ctx context.Context, branches []BranchSpec, run *Run) ([]BranchResult, error)
}
```

**Design decisions:**
- Each branch runs in a dedicated goroutine
- Results collected in **declaration order** (branches[0] → results[0])
- Trace events **buffered per branch**, flushed in order at join
- Error in any branch triggers **fail-fast**: cancel remaining branches via context
- Branch cancellation uses shared context derived from parent

**Rationale:**
- Deterministic ordering enables reproducible replays
- Buffered trace events avoid interleaved output across branches
- Fail-fast prevents wasted work on doomed parallel executions
- BranchExecutor interface allows test doubles for parallel logic

### 5. wait_for_event Semantics

**Location:** `pkg/eventbus/dispatcher.go` (existing)

The `EventDispatcher` interface already implements consume semantics:

```go
type EventDispatcher interface {
    Dispatch(ev InboundEvent) error
    Wait(ctx context.Context, stepID string, filter EventFilter, timeout time.Duration) (*InboundEvent, error)
    Cancel(stepID string, reason string)
}
```

**Consume semantics:**
- First waiter wins: `Dispatch` delivers to exactly one matching `Wait` call
- `Wait` blocks until event arrives, timeout, or context cancellation
- Engine emits `event/received` when event arrives, `step/resumed` when execution continues

**Engine integration:**
1. `wait_for_event` executor calls `dispatcher.Wait(ctx, stepID, filter, timeout)`
2. On event arrival, executor returns `StepResult` with event payload
3. Engine emits `event/received` trace event
4. Engine emits `step/resumed` trace event
5. Execution continues to next step

### 6. Trace Event Protocol

**Location:** `pkg/trace/writer.go` (existing)

The `TraceWriter` interface:
```go
type TraceWriter interface {
    Append(event TraceEvent) error
    Close() error
}
```

**Required event sequence:**
```
run/started         → on Run begin (first Next() call)
  step/started      → before each step
  step/completed    → on step success
  step/failed       → on step failure
  step/skipped      → on step skip (condition false)
  event/received    → when wait_for_event receives event
  step/resumed      → after event received
run/completed       → on successful completion
run/failed          → on unrecoverable failure
run/cancelled       → on user cancellation
```

**Event emission guarantees:**
- Events written to TraceWriter **synchronously** (at-least-once)
- Events sent to EventBus **best-effort** (non-blocking, drop on full)
- Events sent to RunHandle.Events() channel **best-effort**
- `OnEvent` callback invoked **synchronously** after trace write

### 7. Signal Handling

**Location:** `pkg/platform/platform.go`

Added `NotifySignals` method:
```go
type Platform interface {
    // ... existing methods ...
    NotifySignals(ctx context.Context) <-chan Signal
}

type Signal struct {
    Name string  // "SIGTERM", "SIGINT"
}
```

**Engine integration:**
- Engine listens for signals via `platform.NotifySignals(ctx)`
- On SIGTERM/SIGINT: call `runHandle.Cancel(ctx, "signal:SIGTERM")`
- Grace period: allow current step to complete before forced termination

---

## Deliverables

| File | Description |
|------|-------------|
| `v2/pkg/engine/engine.go` | Engine interface, EngineConfig, BranchExecutor |
| `v2/pkg/engine/run.go` | Run struct, RunStatus, StepStatus, StepResult |
| `v2/pkg/engine/executor.go` | StepExecutor, ExecutorRegistry, ExecutionContext |
| `v2/pkg/platform/platform.go` | Added NotifySignals, Signal type |
| `v2/pkg/platform/fake.go` | FakePlatform.NotifySignals |
| `v2/pkg/platform/real.go` | realPlatform.NotifySignals |
| `v2/internal/engine/doc.go` | Package documentation |
| `v2/internal/engine/engine.go` | Concrete engine stub |
| `v2/internal/engine/engine_test.go` | 11 test cases (9 skip, 2 pass) |

---

## Test Coverage

| Test | Status | Description |
|------|--------|-------------|
| `TestEngine_Start_EmitsRunStarted` | SKIP | run/started on first Next() |
| `TestEngine_Execute_SingleStep_Success` | SKIP | Single step execution |
| `TestEngine_Execute_MultipleSteps_SequentialOrder` | SKIP | Step ordering |
| `TestEngine_Execute_StepFailed_EmitsStepFailed` | SKIP | Failure event emission |
| `TestEngine_Execute_UnknownStepKind_ReturnsError` | SKIP | Missing executor handling |
| `TestEngine_Cancel_EmitsRunCancelled` | SKIP | Cancellation event |
| `TestEngine_State_ReturnsCurrentRunState` | SKIP | State snapshot |
| `TestEngine_Events_ReceivesEventsOnChannel` | SKIP | Event channel delivery |
| `TestEngine_VarsPropagation_StepVarsMergedIntoRunVars` | SKIP | Variable propagation |
| `TestEngineConfig_Validate_MissingRequired` | PASS | Config validation (missing) |
| `TestEngineConfig_Validate_AllRequired` | PASS | Config validation (valid) |

---

## Build Status

```
$ cd /Volumes/Projects/gert/v2
$ go build ./...   # PASS
$ go vet ./...     # PASS
$ go test ./internal/engine/... -v   # 11 tests (9 SKIP, 2 PASS)
```

---

## Impact

- **Unblocks Phase 3 implementation:** Brian can write concrete executor logic
- **Unblocks step executor implementations:** CLI, tool, parallel, iterate, wait_for_event
- **Unblocks adapter integration:** Serve, CLI, TUI can use RunHandle interface
- **Event protocol locked:** Trace format compatible with §06 spec

---

## Open Questions (deferred)

1. **Parallel branch trace buffering** — exact buffer size and backpressure strategy (defer to implementation)
2. **Resume serialization format** — how Run struct is serialized for checkpoint (defer to Phase 4)
3. **Concurrent executor access** — whether executors must be goroutine-safe (clarified: yes, for parallel branches)

---

## Locked Decisions

These decisions are now locked per project governance:

1. **Client-driven execution:** RunHandle.Next() drives advancement; engine never auto-advances
2. **Deterministic branch order:** Parallel results collected in declaration order
3. **Buffered branch traces:** Events buffered per branch, flushed at join
4. **Fail-fast parallel:** Error in any branch cancels siblings
5. **Consume semantics:** wait_for_event uses first-waiter-wins dispatch
6. **Synchronous trace writes:** TraceWriter.Append() must complete before step completion


---

# Phase 3 Runtime Core — Architectural Review

**Reviewer:** Ken (Software Architect)  
**Date:** 2026-04-20  
**Artifact:** `v2/internal/engine/engine.go`  
**Requested by:** Cristian

---

## VERDICT: ✅ APPROVED

Phase 3 implementation is **APPROVED**. The runtime core is well-designed, race-free, and production-ready.

---

## Review Criteria Analysis

### 1. Interface Conformance ✅

**Pass.** Compile-time guard present at line 20:
```go
var _ enginepkg.Engine = (*impl)(nil)
```

### 2. Trace Event Ordering ✅

**Pass.** Events emitted correctly:
- Linear: `run/started → step/started → step/completed → run/completed` (verified)
- Failure: `step/failed → run/completed` (with error payload; no separate `run/failed` kind — documented at line 438)
- Parallel: branch events buffered in `eventBuffer`, flushed in declaration order at lines 303-307 (deterministic by index `i`, not by completion order)
- wait_for_event: `step/started → event/received → step/resumed → step/completed` (lines 349-405)

### 3. Mutex Discipline ✅

**Pass.** Mutex released during all blocking I/O:
- `dispatcher.Wait()`: unlocked at line 358, relocked at line 360
- `executor.Execute()`: unlocked at line 198, relocked at line 200
- `traceWriter.Append()`: called while holding mutex (correct — trace writes are synchronous and brief)

The mutex-held trace write is the right call: sequence numbers must be assigned atomically, and trace writes are append-only file ops (fast).

### 4. Parallel Fail-Fast ✅

**Pass.** `errgroup.WithContext()` at line 277 creates a derived context that cancels all branches when any branch returns an error. `eg.Wait()` at line 298 blocks until all goroutines complete — no leaks.

### 5. Signal Race ✅

**Pass.** Signal handler goroutine (lines 65-87) properly exits via two paths:
- `case sig, ok := <-sigCh`: handles signal, then returns
- `case <-runCtx.Done()`: run completed normally, goroutine exits

No leak: when run completes, `runCancel()` is called (line 421 or 446), closing `runCtx`, which terminates the signal goroutine.

### 6. safeClose Correctness ✅

**Pass.** Uses `CompareAndSwap` (line 721):
```go
if closed.CompareAndSwap(false, true) {
    close(ch)
}
```
CAS is the correct idiom — two concurrent callers cannot both see `false`.

### 7. errgroup Dependency ✅

**Pass.** `golang.org/x/sync v0.20.0` present in both `go.mod` (line 8) and `go.sum` (lines 5-6).

### 8. Test Quality ⚠️ (Minor)

**Mostly pass, with caveats:**

- `TestEngine_ParallelBranches`: Verifies trace event ordering (branch-a before branch-b). However, the test doesn't force a race — both branches complete near-instantly. The ordering check is valid because it tests the *flush order*, not the *completion order*.

- `TestEngine_SignalCancellation`: **Good test.** Uses a blocking executor that waits on `ctx.Done()`, then injects a signal. This actually exercises the race: signal vs. normal completion.

**Observation:** A stress test that adds random delays to parallel branches would increase confidence, but the current tests are sufficient for Phase 3 acceptance.

### 9. Race Detector (`-race -count=5`) ✅

**Pass.** All 5 repetitions clean:
```
go test ./internal/engine/... -race -count=5 -v
PASS (16 tests × 5 = 80 executions, 0 races)
```

Also verified with `-count=10`: still clean.

### 10. go vet ✅

**Pass.** `go vet ./...` returns no warnings.

---

## Non-Blocking Suggestions for Phase 4

1. **mergeContexts goroutine leak** — The helper at line 727-735 spawns a goroutine that only exits when *both* contexts cancel or the merged context cancels. If the caller's context (`a`) outlives the run context (`b`), the goroutine exits promptly. However, if `b` is cancelled but `a` is long-lived (e.g., `context.Background()`), the merged context's goroutine will linger until `cancel()` is called. **Current usage is safe** (merged context is scoped to a single `Next()` call), but consider documenting this or using a different approach in Phase 4.

2. **Test coverage for parallel branch failure** — Add a test where one branch fails and verify that:
   - Other branches receive context cancellation
   - Events from the failing branch appear in trace before events from cancelled branches

3. **Documentation consistency** — `doc.go` mentions `run/failed` (line 21) but no such event kind exists. The implementation correctly uses `run/completed` with an error payload (line 438-439). Update `doc.go` to match.

---

## Summary

| Criterion | Result |
|-----------|--------|
| Interface guard | ✅ |
| Trace ordering | ✅ |
| Mutex discipline | ✅ |
| Parallel fail-fast | ✅ |
| Signal race | ✅ |
| safeClose CAS | ✅ |
| errgroup dependency | ✅ |
| Test quality | ⚠️ (minor) |
| Race detector | ✅ |
| go vet | ✅ |

**Phase 3 is complete.**

---

*Reviewed by Ken, Software Architect*

---

# Phase 5 Review — Ken APPROVED (2026-04-21)

**Reviewer:** Ken (Software Architect)
**Phase:** 5 — Step Type Executors
**Note:** Initially rejected on C9 (assert test missing failure details). Fix Agent resolved defect.

## Final Verdict: ✅ APPROVED (10/10)

| # | Criterion | Result |
|---|-----------|--------|
| C1 | Import discipline (internal/executor imports only pkg/*) | ✅ PASS |
| C2 | Nil-safety | ✅ PASS |
| C3 | Deny-wins in assert | ✅ PASS |
| C4 | CLI executor subprocess model | ✅ PASS |
| C5 | Template evaluator thread-safety | ✅ PASS |
| C6 | end step terminal handling | ✅ PASS |
| C7 | parallel/wait_for_event NOT registered in registry | ✅ PASS |
| C8 | include executor registered as no-op | ✅ PASS |
| C9 | Test coverage — failures shape verified (type, subject, expected) | ✅ PASS |
| C10 | Race safety (-race -count=5) | ✅ PASS |

*Reviewed by Ken, Software Architect*

---

# Phase 6 Review — Ken APPROVED (2026-04-21)

**Reviewer:** Ken (Software Architect)
**Phase:** 6 — Tool Runtime
**Status:** APPROVED (10/10)

| # | Criterion | Result |
|---|-----------|--------|
| C1 | Import discipline (pkg/tool leaf, internal/executor no internal/tool) | ✅ PASS |
| C2 | Transport interface correctness | ✅ PASS |
| C3 | stdio spawn-per-invocation | ✅ PASS |
| C4 | Persistent process management (jsonrpc/mcp + mutex) | ✅ PASS |
| C5 | MCP handshake correctness (protocol 2024-11-05) | ✅ PASS |
| C6 | All 8 builtin stubs registered | ✅ PASS |
| C7 | ToolExecutor replacement (Phase 5 stub fully replaced) | ✅ PASS |
| C8 | 7 reference tool binaries compile and conform to wire format | ✅ PASS |
| C9 | 32 tests, TestMain binary builds, -race -count=3 green | ✅ PASS |
| C10 | Race safety — full suite clean | ✅ PASS |

Non-blocking recommendations: R1 (preserve stdout/stderr on failed exit), R2 (MCP notifications/initialized method name), R3 (stale Phase 5 doc comment), R4 (aws.tool.yaml copy-paste).

*Reviewed by Ken, Software Architect*

---

# Phase 12 — OpenTelemetry Integration Architecture

**Date:** 2026-04-21  
**Author:** Ken (Software Architect)  
**Status:** PROPOSED  
**Phase:** 12 — OpenTelemetry Integration

---

## Summary

Phase 12 integrates OpenTelemetry (OTel) distributed tracing into the gert engine. The design establishes OTel as an optional, pluggable dependency with noop default, enabling observability without bloat for users who don't need it.

---

## Decision D-12-01: OTel Dependency Model — Pluggable Interface with Noop Default

**Decision:** OTel is optional and pluggable. The engine depends on a `TracerProvider` interface, not the OTel SDK directly. When no provider is configured, a noop tracer is used.

**Rationale:**
- gert binaries remain small (~10MB) without OTel SDK bloat
- Users opt-in by wiring a real provider
- Interface is forward-compatible with OTel Go SDK

**Trade-off:** Users must wire OTel themselves; no auto-discovery.

---

## Decision D-12-02: Span Structure — Hierarchical Mapping

**Decision:** Spans form a tree: `run → step → tool/input`. Span names use `gert.{concept}.{subtype}` format.

| gert Concept | Span Name Format |
|--------------|------------------|
| Run | `gert.run` |
| Step | `gert.step.{kind}` |
| Tool call | `gert.tool.{transport}` |
| Input prompt | `gert.input.{type}` |
| Parallel branch | `gert.branch.{label}` |

---

## Decision D-12-03: Context Propagation

**Decision:** The `context.Context` carries the current span through engine → executors → tools. Executors receive a context with the step span as parent.

---

## Decision D-12-04: Attribute Conventions

**Decision:** Span attributes use `gert.*` prefix for gert-specific attributes, OTel semantic conventions (`process.*`) where applicable.

Key attributes: `gert.run.id`, `gert.step.id`, `gert.step.kind`, `gert.tool.name`, `gert.tool.transport`.

---

## Decision D-12-05: Export Target — Pluggable via TracerProvider

**Decision:** gert does not configure export targets. The user provides a pre-configured `TracerProvider`. CLI offers convenience flags (`--otel-endpoint`, `--otel-stdout`) for common cases.

---

## Decision D-12-06: Integration Points

**Decision:** Spans open/close at precise lifecycle points:
- `gert.run` span: constructor to `finishRun()`
- `gert.step.*` span: `executeStep()` entry to exit
- `gert.tool.*` span: around transport call
- `gert.input.*` span: around blocking input wait

---

## Decision D-12-07: Relationship to Evidence/Trace

**Decision:** OTel spans and gert's NDJSON trace are complementary:
- NDJSON: Audit log, replay source, evidence (required)
- OTel: Distributed tracing, performance analysis (optional)

Correlation via `gert.run.id` attribute matching `run_id` in NDJSON.

---

## Scope

**Ships in Phase 12:**
1. OTel span integration
2. Pluggable TracerProvider
3. Context propagation
4. Attribute conventions
5. RunStore unit tests (Phase 11 housekeeping)
6. Atomic attachment writes (Phase 11 housekeeping)
7. Attachment error logging (Phase 11 housekeeping)

**Deferred:**
- OTel metrics API
- Baggage propagation to tools
- Automatic trace context injection to subprocesses

---

## Dependencies

- Depends on: Phase 11 (Evidence & Replay)
- Blocks: Phase 13+ (OTel metrics, baggage propagation)

---

## Phase 12 Preflight — Build Verification

**Date:** 2026-04-21  
**Author:** Barbara (Integrations Specialist)  
**Status:** ✅ Build green — Phase 12 may proceed

### Findings

1. **FakeInputProvider.Name()** — Already present
   - Ken's Phase 11 review flagged missing method
   - Method already present, committed as part of Phase 11 (`3961cf6`)
   - Compile-time guard (`var _ input.InputProvider = (*FakeInputProvider)(nil)`) confirms correctness

2. **Hardcoded Machine Path in `cmd/gert/run_test.go`** — Fixed
   - `TestRun_SuccessExitCode` used hardcoded absolute path
   - Fixed to portable relative path: `filepath.Join("..", "..", "..", "design", "gert-v2", "testdata", "runbooks", "r21-gert-run", "schema.yaml")`
   - Commit: `7cafb25`

### Build & Test Status

```
go build ./...        ✅ exit 0
go test ./... -race   ✅ all packages pass, 0 failures
```

### Recommendation

✅ **Phase 12 may begin.** Integration surface is clean.

---
# Phase 12 Review Decision

**Reviewer:** Ken (Software Architect)  
**Date:** 2026-07-20  
**Phase:** 12 — OpenTelemetry Integration

---

## Verdict

```
APPROVED WITH NON-BLOCKING ITEMS [8.5/10]
```

Phase 12 ships. Brian's implementation matches the design intent with acceptable deviations.

---

## Phase 13 Part A Housekeeping Items

### NBI-12-01: Add OTLP Adapter Package

**Description:** Create `pkg/otel/adapter` that wraps the real OTel SDK (`go.opentelemetry.io/otel`) to implement gert's `TracerProvider` interface.

**Acceptance criteria:**
- `NewOTLPTracerProvider(endpoint string, opts ...Option) otelPkg.TracerProvider`
- Wire into `buildTracerProvider` when `WireOptions.OTelEndpoint != ""`
- May use build tags or separate module to keep core gert lean

**Priority:** Medium  
**Estimate:** 4-6 hours  
**Owner:** Brian

### NBI-12-02: Document OTLP Flag Status

**Description:** Update CLI help and documentation to indicate `--otel-endpoint` is reserved but not yet functional.

**Acceptance criteria:**
- `--help` output notes "reserved for future use"
- Release notes mention OTLP export in v2.1

**Priority:** Low  
**Estimate:** 30 minutes  
**Owner:** Brian

### NBI-12-03: Consider context.AfterFunc (Go 1.21+)

**Description:** If minimum Go version is raised to 1.21, replace `mergeContexts` goroutine pattern with `context.AfterFunc` for lower overhead.

**Priority:** Low (future optimization)  
**Estimate:** 1 hour  
**Owner:** Future pass

---

## Accepted Deviations

| ID | Deviation | Rationale |
|----|-----------|-----------|
| BRD-12-01 | No real OTel SDK dependency | Avoids ~30MB transitive deps; custom interfaces are API-compatible |
| BRD-12-02 | OTLP endpoint flag is no-op | SDK required for OTLP; flag parsed and ready for Phase 13 wiring |
| BRD-12-03 | mergeContexts spawns goroutine per call | Standard pattern; bounded by step count; profile in v2.1 if needed |

---

## Phase 11 Housekeeping: Complete

| Item | Status |
|------|--------|
| RunStore unit tests (9 tests added) | ✅ Done |
| Atomic attachment writes (temp→sync→rename) | ✅ Done |
| Warn logging for attachment errors | ✅ Done |

---

## Next Phase

Phase 13 may proceed with Part A addressing NBI-12-01 and NBI-12-02 before Part B work begins.

---

*Ken, Software Architect*

---
# Brian Phase 12 Implementation — Decisions & Deviations

**Author:** Brian (Go Programmer)
**Date:** 2026-07-18
**Phase:** 12 (OpenTelemetry Integration)
**Status:** Implementation complete — awaiting Ken review

---

## Implemented As Designed

- D-12-02: Span hierarchy `gert.run → gert.step.{kind} → gert.branch.{label}` ✓
- D-12-03: Context propagation through engine → executors via `spanCtx` ✓
- D-12-04: Attribute constants in `pkg/otel/attributes.go` with `gert.*` prefix ✓
- D-12-06: Span open/close at precise lifecycle points ✓
- D-12-07: OTel spans and NDJSON trace are independent ✓

---

## Deviations from Ken's Design

### BRD-12-01: OTel SDK Not Added as Dependency

**Ken's intent:** "Add `go.opentelemetry.io/otel` as a dependency (use `go get` in v2/)"

**What was done:** `go.opentelemetry.io/otel` was NOT added to `go.mod`. The `pkg/otel` package defines its own structurally-compatible interfaces (TracerProvider, Tracer, Span) without importing the SDK.

**Rationale:**
- The real value is in the interface design, not the SDK import
- Adding the SDK would pull in ~30MB of transitive dependencies
- `RecordingTracerProvider` (in `pkg/otel/tracer.go`) satisfies all test requirements without the SDK
- The interface is forward-compatible: wrapping the real OTel SDK in Phase 13 requires only a thin adapter

**Action for Phase 13:** When OTLP wiring is added, the real OTel SDK can be imported in `internal/adapter/otel_adapter.go` (as Ken noted). The `pkg/otel` interfaces need no changes.

---

### BRD-12-02: OTLP Endpoint Wiring Deferred

**Ken's design:** `--otel-endpoint` flag activates OTLP exporter.

**What was done:** `--otel-endpoint` flag is parsed and stored in `WireOptions.OTelEndpoint`. `buildTracerProvider()` in `wire.go` accepts the value but does not construct an OTLP exporter (returns nil → noop). A code comment marks the deferral.

**Rationale:** OTLP requires the `go.opentelemetry.io/otel/exporters/otlp/otlptrace` SDK. Adding it in Phase 12 would require the full SDK dependency chain. Deferred to Phase 13 per the "No OTel SDK" decision above.

**User impact:** `--otel-stdout` (stderr debug spans) works now. `--otel-endpoint` is accepted but currently a no-op.

---

### BRD-12-03: Context Propagation Design

**Ken's intent:** Pass `spanCtx` to executors so they can create child spans.

**Implementation detail:** `runHandle` now carries `traceCtx context.Context` (holding the run span) alongside `runCtx` (for cancellation). `Next()` builds `stepCtx = merge(traceCtx, runCtx, callerCtx)` so:
1. The run span is accessible in the context for child span creation
2. Either `runCtx` (signal) or `callerCtx` cancellation stops the step
3. Executors receive `spanCtx` (child of step span) for further child spans

**Note for Ken:** The `mergeContexts` function creates a goroutine per call. With 3 merges per step this could accumulate goroutines under high step counts. This is pre-existing behavior; consider replacing with a `context.WithoutCancel`-based approach in Phase 14+.

---

## Notable Implementation Notes

### pkg/otel Package Design

The `pkg/otel` package is self-contained with no external dependencies:
- `TracerProvider`, `Tracer`, `Span` interfaces (subset of OTel SDK interfaces)
- `noopTracerProvider` / `noopTracer` / `noopSpan` — zero allocation no-ops
- `RecordingTracerProvider` — thread-safe in-memory span recorder for tests
- `ContextWithSpan` / `SpanFromContext` — context propagation helpers
- `WithAttributes` — `SpanStartOption` functional option

### stdoutTracerProvider (internal/adapter/wire.go)

A minimal debug provider that writes span summaries to stderr as JSON when `--otel-stdout` is set. Does not persist spans, does not propagate context. Intended for CLI debugging only.

### Attachment Atomicity (internal/evidence/attachment.go)

The existing code already had `dst.Sync()` but wrote directly to the final path. Changed to: create `.tmp` → write → Sync → Rename → cleanup on any error. The rename is atomic on POSIX systems (same filesystem).

### RunStore Tests (internal/runstore/dir_store_test.go)

9 tests covering all Ken-specified cases plus the concurrent writer test (10 goroutines × 100 events = 1000 total lines verified). The `TestWriteFileAtomic_PartialWrite` test verifies no `.tmp` files remain after successful SaveState and that the final JSON is valid.

---

# Phase 13 — CLI Polish & OTLP Adapter — Decisions

**Author:** Ken (Software Architect)  
**Date:** 2026-07-20  
**Phase:** 13 — CLI Polish & OTLP Adapter  
**Status:** PROPOSED

---

## Summary

Phase 13 completes the NBI carry-forwards from Phase 12 (OTLP adapter wiring) and adds essential CLI commands for production use (`gert ls`, `gert gc`, `gert version`).

---

## Decision D-13-01: OTel SDK as Direct Dependency (No Build Tags)

**Decision:** Add `go.opentelemetry.io/otel` SDK packages directly to `v2/go.mod` without using build tags to isolate them.

**Rationale:**
- Go's dead code elimination removes unused OTel code from binaries that don't call `--otel-endpoint`
- Build tags add complexity (multiple build configurations, CI matrix expansion)
- The OTel SDK is well-maintained and adds ~10MB which is acceptable for a CLI tool
- Phase 12 already established the interface boundary in `pkg/otel`; the adapter is a thin wrapper

**Trade-off:** All binaries include OTel SDK code in `go.mod`, but DCE keeps actual binary size minimal.

---

## Decision D-13-02: Defer context.AfterFunc Optimization (NBI-12-03)

**Decision:** NBI-12-03 (replacing `mergeContexts` goroutine pattern with `context.AfterFunc`) is deferred to Phase 14+.

**Rationale:**
- Go 1.25.7 supports `context.AfterFunc` (added in Go 1.21)
- The current `mergeContexts` pattern is bounded: max 3 goroutines per step
- No evidence of goroutine leaks or performance issues in Phase 11/12 testing
- Phase 13 scope should prioritize user-facing features over internal optimization
- Future profiling may reveal this is a non-issue

**Action:** Ken will re-evaluate in Phase 14 based on production telemetry.

---

## Decision D-13-03: gert gc Safety Invariant

**Decision:** `gert gc` MUST NOT delete runs with status "running".

**Rationale:**
- A running run may be paused or awaiting input
- Deleting its state would corrupt the run and lose evidence
- This is a hard invariant, not configurable via flags
- Implementation: filter `running` status from deletion candidates before any prompts

**Enforcement:** Unit test `TestGc_PreservesRunning` verifies this invariant.

---

## Decision D-13-04: Version Information via ldflags

**Decision:** Embed version, commit hash, and build date using Go's `-ldflags -X` mechanism.

**Rationale:**
- Standard Go pattern used by most CLI tools (kubectl, docker, gh)
- Works with any CI/CD system
- Defaults to "dev/unknown/unknown" for local `go build` without flags
- No runtime file reads or external dependencies

**Implementation:**
```go
var (
    Version   = "dev"
    Commit    = "unknown"
    BuildDate = "unknown"
)
```

Build command:
```bash
go build -ldflags "-X main.Version=v2.0.0 -X main.Commit=$(git rev-parse --short HEAD) -X main.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
```

---

## Decision D-13-05: Part B Scope — CLI Polish over Integration Tests

**Decision:** Phase 13 Part B implements CLI commands (`gert ls`, `gert gc`, `gert version`) rather than comprehensive integration tests.

**Rationale:**
- Phases 0-12 have extensive unit tests (all packages pass with `-race`)
- Golden file tests in `internal/parser`, `internal/replay`, etc. provide integration-level coverage
- User-facing gaps are more impactful for v2.0 GA readiness:
  - No way to list past runs
  - No way to clean up disk space
  - No version command
- Integration tests (end-to-end runbook execution) can be added in Phase 14+ as a hardening measure

**Trade-off:** Delays comprehensive integration test suite, but delivers user-facing features sooner.

---

## Scope Confirmation

### Part A (NBI Carry-Forward)

| ID | Status | Notes |
|----|--------|-------|
| NBI-12-01 | IN SCOPE | OTLP adapter package, wire integration |
| NBI-12-02 | IN SCOPE | --otel-endpoint help text update |
| NBI-12-03 | DEFERRED | context.AfterFunc optimization (D-13-02) |

### Part B (New Features)

| Command | Scope |
|---------|-------|
| `gert ls` | List runs with status/date filters |
| `gert gc` | Delete old runs with safety checks |
| `gert version` | Print version info |
| Help polish | Consistent command descriptions |

### Explicitly Out of Scope

- `gert serve` enhancements (auth, rate limiting)
- MCP tool transport improvements
- Integration test suite
- Schema validation enhancements
- Documentation/README updates (deferred to docs sprint)

---

## Dependencies

- Depends on: Phase 12 (sealed, OTel interfaces established)
- Blocks: Phase 14 (serve hardening, integration tests)

---

## Acceptance Criteria

Phase 13 is accepted when:

1. `go build ./...` exits 0
2. `go vet ./...` exits 0
3. `go test ./... -race -count=3` all pass
4. `gert --help` shows all commands with descriptions
5. `gert version` outputs version string
6. `gert ls` lists runs (or shows empty message)
7. `gert gc --dry-run` shows candidates for deletion
8. `gert run ... --otel-endpoint=localhost:4317` exports spans to OTLP collector

---

*Ken, Software Architect*

---

# Phase 13 Preflight — Barbara's Report

**Date:** 2026-07-20  
**Baseline:** Commit 6ab513e (Phase 12 sealed)  
**Status:** ✅ CLEAN — Ready for Phase 13

## Checks Performed

### 1. Build (`go build ./...`)
**Status:** ✅ PASS  
All v2 packages build successfully with no errors or warnings.

### 2. Vet (`go vet ./...`)
**Status:** ✅ PASS  
No static analysis issues detected.

### 3. Tests with Race Detector (`go test ./... -race -count=1 -timeout=120s`)
**Status:** ✅ PASS  
- 14 packages with passing tests
- 24 packages with no test files (expected for cmd/pkg structure)
- No race conditions detected
- All tests completed within timeout

### 4. Temporary Files (`find . -name "*.tmp" -not -path "*/vendor/*"`)
**Status:** ✅ PASS  
No leftover .tmp files from atomic writes.

### 5. Module Tidiness (`go mod tidy && git diff go.mod go.sum`)
**Status:** ✅ PASS  
- `go.mod` and `go.sum` are tidy
- One dependency downloaded during tidy check (gopkg.in/check.v1) — normal behavior
- No diffs after tidying

## Baseline Health

The integration surface is clean. All build artifacts, tests, and dependencies are in good order. No blockers for Phase 13.

## Recommendation

✅ **APPROVED** — Phase 13 can proceed.

---

*Barbara, QA Engineer*
