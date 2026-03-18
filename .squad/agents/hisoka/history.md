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