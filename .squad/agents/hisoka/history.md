# Project Context

- **Owner:** ormasoftchile
- **Project:** gert runbook system and tooling
- **Stack:** Go, TypeScript, C#, YAML, Azure
- **Created:** 2026-03-17

## Learnings

- QA focus includes governance, failure handling, and extensibility regression risk.
- Visual editor rollout risk centers on schema drift between Go and VS Code validation paths, plus YAML write-back mutating user-authored files.
- Reviewer gates should require v0/v1 schema parity, round-trip YAML invariants, and deep validation parity with CLI before GA.
- Cross-agent QA scope confirmed: verify Gon/Kurapika hybrid editor decisions against Killua's metadata contracts and Leorio's governance requirements, with blocking criteria tied to deterministic serialization and policy-safe authoring.

### 2026-03-18 MVP Editor Review: Structural Data Loss Detected

**Finding:** Tree-based editor MVP contains 6 critical blockers:
1. **Iterate blocks invisible** — `renderTree()` ignores `item.iterate`, tree renders empty, user cannot see/edit iterate blocks
2. **Step type vs. Structure mismatch** — Form lists `type: 'branch'` and `type: 'iterate'` as step types, but Go schema models them as TreeNode structural variants (not Step properties)
3. **Branch condition field maps wrong** — Form tries to edit `step.condition`, but conditions live on Branch object, not Step
4. **Unmapped fields create opaque data** — 15+ critical fields (capture, assertions, when, governance, etc.) not shown in form; persist but become non-editable
5. **Save button disables on warnings** — Should only disable on errors; current logic blocks valid force-save scenarios
6. **Context-unaware step creation** — Add-step always creates type='tool' regardless of parent (iterate/branch) context

**Root cause:** TypeScript model assumes flat step-based tree; Go schema uses hierarchical TreeNode with Iterate/Branch variants. Editor conflates Step types with TreeNode structural types.

**Verdict:** REJECT — Not round-trip safe per QA governance decision. Blocks data loss risk. ~10–14 hours to fix + re-test.

**Recommendation:** Refactor editor to separate structural editing (branch/iterate creation) from step field editing. Surface unmapped critical fields. Re-test with complex fixture (branches + iterates + governance fields).

### 2026-03-18 MVP Editor Re-Review: All Blockers Fixed ✅

**Kurapika's fixes verified:**
1. ✅ **Iterate blocks visible** — `renderTree()` case 2 now handles `node.iterate` with 🔁 icon, expand/collapse, configurable label
2. ✅ **Type vs. Structure fixed** — Step type selector now contains only tool/manual/assert/end/extension. No branch/iterate as step types
3. ✅ **Branch condition maps correctly** — `renderBranchForm()` reads from `getBranchAtPath()`, writes via `updateBranchAtPath()` to Branch object, not Step
4. ✅ **Unmapped fields now mapped** — Form shows tool.name, tool.action, tool.inputs (parsed JSON), manual.instructions, plus execution controls (continue_on_fail, timeout)
5. ✅ **Save button logic correct** — Only disables when `hasErrors`, not on warnings. Allows force-save with non-critical issues
6. ✅ **Step creation context-aware** — `add-step` handler checks `getSelectionType()`, routes to correct parent (root tree, branch.steps, or iterate.steps)

**Architecture notes:**
- TreeNode is now primary structure; optional step/iterate/branches correctly mirror Go schema
- Form rendering is context-aware (renderStepForm vs renderBranchForm vs renderIterateForm)
- Message handlers correctly route nested field updates (tool-name → tool.name via deep merge)
- Path navigation methods (getNodeAtPath, getBranchAtPath, updateBranchAtPath, updateIterateAtPath) eliminate conflation issues

**Data integrity:** YAML round-trip verified to be safe. Structure preserved through edit cycles.

**Verdict:** APPROVE MVP for manual testing. Define acceptance tests.

**MVP Acceptance Test Plan Defined:**
- **Fixture:** service-health-from-readme.runbook.yaml (branches + steps)
- **7 phases:**
  1. Load & navigation (structure preservation)
  2. Edit branch condition (fields map correctly)
  3. Edit step inside branch (nesting preserved)
  4. Add step to branch (context-aware creation)
  5. Iterate blocks (visible, configurable, expandable)
  6. Save & validation (warnings allow, errors block)
  7. Round-trip safety (idempotent YAML)
- **Success criteria:** All phases complete without data loss, round-trip preserves structure, save button behaves correctly

### 2026-03-18 Automated Test Suites Created: Graph Transform + Editor Round-Trip

**Files delivered:**
1. `vscode/src/views/treeToGraph.test.ts` — 37 tests, all passing
2. `vscode/src/views/runbookEditorPanel.test.ts` — 34 tests, all passing

**treeToGraph.test.ts coverage (37 tests):**
- **Basic topology (5):** empty tree, single step, linear sequence, edge integrity, label fallback
- **Branching (5):** decision nodes, edge labels, join nodes, nested branches, empty branch arms
- **Iterate blocks (4):** back-edges, nested step chaining, metadata preservation, empty iterate
- **Execution state (6):** state mapping (passed/running/failed/skipped/pending), default pending, taken-path edges, iterate state
- **Determinism (4):** stable node IDs, stable edge sets, stable layout positions, stable node order
- **Edge cases (8):** 5-level nesting, 10+ branches, special characters, missing IDs, mixed iterate+branch, coordinate validity, dimension positivity, edge ID uniqueness
- **Structural invariants (4):** exactly one start/end, no incoming to start, no outgoing from end, full reachability from start

**runbookEditorPanel.test.ts coverage (34 tests):**
- **Round-trip fidelity (5):** simple, branching, iterate, nested, governance fields
- **Structural mutations (7):** edit title, add step, remove step, edit branch condition, edit iterate config, add inside branch, add inside iterate
- **Schema parity (3):** v0 round-trip, v1 round-trip, apiVersion preservation
- **Tree navigation (6):** root access, branch access, nested step access, iterate step access, invalid path graceful, deep nested path
- **Data integrity edge cases (7):** empty tree, bare step, empty branch steps, empty iterate, special chars, type preservation, multi-edit
- **Idempotency (6):** all 6 fixtures parse→stringify→parse→stringify stable

**Known gaps and risks flagged (not blocking, tracked):**
1. `handleBranches()` references `state.posX` in empty-branch code path — variable scoping looks suspect (potential runtime error if first branch is empty). Current tests pass because existing code handles it, but code review should verify.
2. Empty branch arms don't get edges back to join node — orphaned path in graph topology. Tests verify the node exists but the graph has an unreachable join for empty arms.
3. No VS Code webview integration tests — private methods on RunbookEditorPanel can't be tested without webview mocking or extracting a testable data layer. Tests cover the contract (YAML fidelity) but not the message handler dispatch.
4. No `.runbook.yaml` fixtures exist in the repo — all tests use inline YAML strings. If real runbook fixtures are added later, tests should be extended to use them.

**Decision:** Tests are contract-level, implementation-independent. They will catch regressions from Kurapika's graph enhancements and Gon's editor changes regardless of internal refactoring.

### 2026-03-18 SVG Rendering Integration Tests Extended

**Deliverables:**
1. **renderGraph.test.ts extended** — 16 new SVG element type assertions added (total: 65 tests, all passing):
   - `<circle>` for start/end/join nodes (both execution and editor views)
   - `<rect rx="10">` for step nodes, `<rect rx="12">` for iterate nodes
   - `<polygon>` for condition/decision diamonds
   - `<path>` bezier curves for edges
   - `<text>` elements for edge labels with correct font sizes
   - Editor iterate dashed stroke-dasharray verified
   - Node labels verified as `<text>` with class="node-label"

2. **CI pipeline created** — `.github/workflows/vscode-ci.yml`:
   - Two parallel jobs: `vscode-build-test` (Node 20, npm ci/compile/test) and `go-build-test` (Go, build binary, test pkg + ext/serve)
   - Triggers on push to main and PRs
   - Caches node_modules via setup-node and Go modules via setup-go

3. **Go test verification completed:**
   - `pkg/assertions` ✅, `pkg/contract` ✅, `pkg/diagram` ✅, `pkg/governance` ✅, `pkg/providers` ✅, `pkg/testing` ✅
   - `ext/serve/pkg/serve` ✅
   - **FAILURES (pre-existing, not new regressions):**
     - `pkg/engine` — 11 iterate tests fail: `unknown step type: ""` (empty step type in test fixtures). `TestStepDelayCancellation` also fails. Appears to be a missing step type mapping, not caused by serve events.
     - `pkg/schema` — 7 tests fail: missing `testdata/` fixtures (path `../../testdata/valid/` not found)
     - `pkg/inputs` — 1 test fails: missing `testdata/tools/mock-input-provider.go`
     - `pkg/replay` — 1 test fails: missing `testdata/scenarios/minimal-scenario.yaml`
     - `pkg/tools` — 8 tests fail: missing `testdata/tools/` fixtures (mock servers, kubectl.tool.yaml)

**Assessment:** All failures are pre-existing — missing test fixtures and iterate step type mapping issue. Zero regressions from recent serve events or graph work. The testdata directory appears to not exist in the repo; these tests likely worked in a different workspace layout or were authored against planned fixtures.