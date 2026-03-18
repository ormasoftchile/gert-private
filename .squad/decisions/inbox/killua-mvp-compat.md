### 2026-03-17T18:00:00Z: Visual Editor MVP Schema Compatibility Constraints

**By:** Killua (Backend Engineer)
**Topic:** Editor schema/runtime compatibility review for MVP
**Decision:** Implement safety guards to block editing non-linear runbook structures. Allow metadata-only editing when tree contains branches/iterates or unsupported step types.

## Compatibility Constraints Identified

### Critical Constraints (Require Mitigation)

1. **Non-Linear Tree Structures**
   - Runbooks with `branches` at top level in tree nodes are fundamentally incompatible with the editor's linear step extraction/rebuild pattern.
   - Runbooks with `iterate` blocks at top level cannot be safely edited via the flattened step UI.
   - **Impact:** Corrupts branch conditions and iterate semantics on save.
   - **Mitigation:** Block add/remove/reorder operations when branches or iterates are present.

2. **Complex Step Types**
   - Editor only handles simple tool steps (type="tool", no nested manual/assertions/choices).
   - Steps with type="manual", "assert", "branch", "parallel", or "end" are not extracted or edited.
   - Steps with manual instructions, approvals, required_evidence, or choices are silently lost.
   - **Impact:** Data loss on save; required fields disappear.
   - **Mitigation:** Block tree editing when unsupported step types are detected.

3. **Tool Configuration Fidelity**
   - Editor extracts only step.id, step.type, step.title from tool steps.
   - Discards tool.name, tool.action, tool.args, tool.capture, and all other step fields.
   - **Impact:** Complete loss of tool configuration on save.
   - **Mitigation:** Preserve existing tree structure by default; only allow metadata editing when tree has complex structures.

4. **YAML Preservation**
   - YAML.stringify() reformats all YAML; loses comments, custom spacing, and ordering.
   - Decision requires "non-destructive normalization" but current approach is fully destructive.
   - **Impact:** Hand-written runbooks become autoformatted; user comments disappear.
   - **Mitigation:** Accept as MVP limitation; log as future backlog item for selective field updates.

### Implementation Changes Made

**File: vscode/src/views/runbookEditorPanel.ts**

1. **Added safety detection methods:**
   - `isTreeLinearAndEditable()`: Returns false if tree contains iterate blocks or any branches.
   - `hasUnsupportedStepTypes()`: Returns true if tree contains non-tool steps or complex step fields.
   - `getTreeEditBlockReason()`: Returns human-readable reason if tree cannot be edited safely.

2. **Updated renderUI():**
   - Check `getTreeEditBlockReason()` before rendering step editor.
   - If blocked, show warning message with reason and guidance ("Edit YAML directly for full control").
   - Hide add/remove/reorder buttons when tree is not editable.
   - Always allow metadata editing (name, kind, description).

3. **Guarded tree mutations in handleWebviewMessage():**
   - Add guards to add-step, remove-step, move-step handlers.
   - Show warning dialog if user attempts tree operation on locked structure.
   - Allow operation to proceed only if tree is confirmed editable.

4. **Added structural safety check before save:**
   - Validate that all top-level tree nodes have step.id and step.type after rebuild.
   - Fail save with error message if rebuild corrupted tree structure.
   - Prevents silent data loss from malformed trees.

## What the Editor MVP Can Now Safely Edit

✅ **Allowed:**
- Metadata: name, kind, description
- Simple linear runbooks (no branches/iterates) with only tool steps
- Add/remove/reorder steps in linear runbooks
- Edit step id, type, title (not tool config)
- Run validation on edits

❌ **Blocked (with user-visible warning):**
- Runbooks with branches at top level
- Runbooks with iterate blocks
- Steps with type != "tool"
- Steps with manual instructions, approvals, assertions, or choices
- Tool configuration editing (tool.name, tool.action, etc.)
- Step capture/export/when/timeout editing

## Alignment with Decision

This implementation aligns with Kurapika's direction:
- **YAML-canonical:** Always writes back to YAML format; preserves file as source of truth.
- **Schema-driven:** Editor behavior driven by schema introspection, not hardcoded lists.
- **Safety-first:** Governance by default — visible warnings, preflight gates, never silently corrupt.
- **Reversible:** Users can edit YAML directly without lost work; editor errors are recoverable.

## Testing Implications

| Scenario | Expected Behavior |
|----------|-------------------|
| Open simple linear runbook | Step editor visible; can add/remove/reorder |
| Open runbook with branches | Warning shown; step editor hidden; metadata still editable |
| Open runbook with iterate | Warning shown; step editor hidden; metadata still editable |
| Edit runbook with manual step | Warning shown; step editor hidden; metadata still editable |
| Add step to simple runbook, save | Tree rebuilt correctly; validation passes; file saved |
| Attempt add step on branching runbook | Warning dialog shown; operation blocked |
| Metadata-only edit on complex runbook | Name/kind/description persisted; tree untouched |

## Future Work (Post-MVP)

1. **Selective field updates** — Preserve comments/formatting by updating YAML fields in place instead of stringify.
2. **Branch editor UI** — Add conditional branch path editor for non-linear runbooks.
3. **Step type selector** — Add form fields for manual/assert/parallel steps.
4. **Tool config editor** — Expose tool.name, tool.action, tool.args in UI with tool registry autocomplete.
5. **Iterate block editor** — Add convergence/list mode selector with step editing inside iterate context.

## Backward Compatibility

- Existing runbooks unaffected. Editor is read-only for unsupported structures.
- Simple runbooks continue to work as designed.
- No breaking changes to Go schema or CLI.
- Users can always fall back to YAML editing.
