# Decision: MVP Editor Rejection — Structural Data Loss Vectors

**Date:** 2026-03-18T16:32:00Z  
**Author:** Hisoka (QA Reviewer)  
**Status:** BLOCKED — 6 Critical Issues

---

## Summary

**❌ REJECT** — The tree-based runbook editor MVP contains critical architectural misalignments between the TypeScript UI model and the Go schema that create data loss vectors and structure corruption. The implementation is not ready for user testing.

---

## Technical Findings

### Blocker 1: Iterate Blocks Completely Invisible
- **Code Location:** `runbookEditorPanel.ts`, `renderTree()` lines 826–877
- **Issue:** Tree renderer only handles `item.branches`; completely ignores `item.iterate`
- **Impact:** Any runbook with iterate blocks:
  - Renders as empty tree
  - User cannot see/edit iterate configuration
  - On save: iterate block silently untouched in memory
  - **Verdict:** STRUCTURE LOSS
- **Required Fix:** Add iterate rendering with expand/collapse + config editor

### Blocker 2: Step Types Include Structural Types (Model Mismatch)
- **Code Location:** Form step-type selector (lines 654–662)
- **Issue:** Form presents `type: 'branch'` and `type: 'iterate'` as step types
- **Schema Reality:** Go schema has:
  - `TreeNode.Iterate *IterateBlock` — separate from Step
  - `TreeNode.Branches []Branch` — separate from Step
  - Steps can have `branches` field, but iterate is at TreeNode level
- **Impact:** Form-data mismatch; user edits non-existent fields
- **Required Fix:** Separate UI for structural editing vs. step field editing

### Blocker 3: Branch Condition Field Maps to Wrong Object
- **Code Location:** Form branch condition field (line 700)
- **Issue:**
  ```typescript
  ${step.type === 'branch' ? `
    <textarea id="step-condition"...>${this.escapeHtml(step.condition || '')}</textarea>
  ` : ''}
  ```
- **Schema Reality:** `step` has NO `.condition` field. Condition lives on `Branch` object (`branches[j].condition`).
- **Impact:** Form writes to non-existent property; branch conditions always empty or lost
- **Data Loss:** YES — branch conditions cannot be edited

### Blocker 4: Unmapped Fields Create Silent Opaque Data
- **Code Location:** Form field selection
- **Missing from Form:**
  - `step.when` — conditional execution
  - `step.capture` / `step.assertions` — output/validation rules
  - `step.continue_on_fail`, `timeout`, `delay` — execution controls
  - `step.export`, `step.scope` — governance
  - `step.approvals`, `step.required_evidence` — manual step policies
  - `tool.params`, `tool.schema` — tool configuration
  - All step-level contract/gate/policy overrides
- **Impact:** Fields persist (shallow merge safe), but become **non-editable**, creating opaque data worse than loss
- **Verdict:** UI cannot manage full runbook complexity

### Blocker 5: Save Button Disables on Warnings (Should Allow Force-Save)
- **Code Location:** Save button render (line 521)
- **Issue:** `${!this.validationResult.valid ? 'disabled' : ''}` disables on both errors AND warnings
- **Schema Decision:** Errors block; warnings allow force-save
- **Impact:** UX blocker — users cannot save runbooks with non-critical warnings
- **Required Fix:** `${this.validationResult.errors?.length > 0 ? 'disabled' : ''}`

### Blocker 6: Context-Unaware Step Creation
- **Code Location:** "Add Step" handler (lines 968–977)
- **Issue:** Always creates `type: 'tool'` regardless of parent context
- **Impact:** Step added inside iterate block loses iterate context
- **Data Loss:** YES — structure corruption

---

## Model Mismatch Analysis

**TypeScript Model (Current)**
- Step is primary unit
- Step types: tool, manual, assert, branch, iterate, etc.
- Form edits step fields directly
- Tree is flat list of steps

**Go Schema (Canonical)**
```
TreeNode {
  Step     Step          // optional
  Iterate  *IterateBlock // optional, mutually exclusive with Step
  Branches []Branch      // at TreeNode level
}
```

**Mismatch:** TypeScript treats `branch` and `iterate` as step types; Go schema treats them as **structural variants of TreeNode**, not Step properties.

---

## Data Integrity Impact

| Aspect | Finding |
|--------|---------|
| Structure preservation | ❌ Fails — iterates invisible, branches confuse form |
| Field completeness | ⚠️ Partial — unmapped fields frozen, not lost but non-editable |
| Form safety | ❌ Fails — writes to non-existent fields (`step.condition`) |
| Validation gates | ❌ Fails — save button blocks valid saves with warnings |
| Navigation | ❌ Incomplete — iterate blocks skipped in tree |

**Verdict:** This is NOT round-trip safe.

---

## Recommendation

### Do NOT Merge Without:

1. Refactor tree renderer to handle iterate blocks (visible + editable)
2. Separate structural editing (branch/iterate creation) from step field editing
3. Fix branch condition field mapping (read from Branch object, not Step)
4. Surface at least the most-critical unmapped fields (capture, assertions, when)
5. Fix save button logic to distinguish errors from warnings
6. Add context-aware step creation

### Effort Estimate
~10–14 hours of development + 4 hours QA re-testing.

### Fallback
If time-boxed, revert to simpler read-only tree view + "Edit YAML" button for MVP. Less ambitious, but avoids data loss risk.

---

## Previous Context

- Previous MVP rejected for data loss (branch flattening)
- Hisoka's governance decision: "Treat schema parity and YAML round-trip fidelity as release blockers"
- This MVP still violates that standard

---

## Next Steps

1. Kurapika addresses blockers 1–6
2. Hisoka re-reviews with complex fixture (branches + iterates + governance)
3. Only after PASS → QA sign-off for manual testing in VS Code
