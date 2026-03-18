# Decision: Visual Runbook Editor — UX Design Direction

**Date:** 2026-03-17  
**Author:** Gon (Lead Architect)  
**Status:** In Review  
**Revision:** v1 — Initial Design Direction  

---

## Problem Statement

Kurapika's Phase 1 MVP correctly chose a form-first approach, but the linear step-list model **flattens branch/conditional structures**, causing data loss when editing non-trivial runbooks. The root issue: UX was designed _after_ implementation began, not before. We need a coherent visual design that preserves runbook structure while remaining form-first for business users.

---

## Proposed Design Direction

### Overall Layout: Form-First + Synchronized Tree Map

The editor uses a **dual-panel hybrid model**:

**Left Panel (60%):** Form-based authoring
- **Runbook Home:** Name, kind, description, lifecycle controls, governance summary badge
- **Step Editor:** Active step form with type selector, parameters, error state
- **Step Navigator:** Scrollable list of step cards (linear view of current level)
- **Add Step Button:** Quick action to append step to current level

**Right Panel (40%):** Synchronized read-only Flow Map
- **Branch Tree Visualization:** Hierarchical tree showing branches, conditionals, and iterate blocks as indented nodes
- **Active Step Highlight:** Visual feedback when user selects a step in the form
- **Expand/Collapse:** Collapse branches to see overall topology; expand to review structure
- **Click-to-Navigate:** Click a branch/conditional node → jump to its step form on the left

**Top Bar:** Floating toolbar
- Validation/Governance badge (red=errors, yellow=policy warnings, green=safe)
- Save button (enabled only if valid)
- Preview Diff button (shows what will write to YAML)
- Cancel/Revert button

### Runbook Home Screen

Appears when no step is selected or user clicks "Runbook Overview":
- **Name field:** Text input
- **Kind selector:** Dropdown (mitigation, monitor, triage)
- **Description field:** Textarea for prose documentation
- **Governance Summary:** Compact badge showing:
  - Tool count + risk level (number of tools requiring approval)
  - Provider integrations count
  - Redaction/deny fields present (yes/no)
  - Compliance status (schema version, v0 vs v1)
- **Meta/Imports Section:** Read-only list of discovered tools and providers (sourced from schema introspection)
- **Quick Links:** "View YAML" (open raw editor), "Open in Terminal" (CLI preview)

### Step Builder Section

**Step List Card** (left side, scrollable):
- Each step shows: `[icon] Step Type | Step Title | [↑↓ move] [✕ delete]`
- Visual indentation for nested steps (within branches/iterate blocks)
- Color-coded type badges (action=blue, branch=purple, iterate=orange, condition=green)
- Click to select → populates form on right side

**Step Detail Form** (active step, left side):
- **Header:** Step type badge, step ID (read-only), step title (editable)
- **Action/Type Selector:** Dropdown + conditional parameters based on type
  - Action step: tool selector, action ID, parameters panel
  - Branch step: routing key selector, branch conditions (inline condition builder)
  - Iterate step: collection selector, loop var name
  - Conditional: condition expression builder
- **Parameters Panel:** Schema-driven fields populated from tool/provider definitions
  - Required fields marked (*), optional unmarked
  - Inline validation hints (red underline on error)
  - Contextual help from tool annotations
- **Governance & Risk Panel** (collapsible):
  - Approval required? (yes/no badge)
  - Redaction (shows which params will be redacted)
  - Deny list (env vars blocked from this step)
  - Annotation notes from schema
- **Delete Step Button:** Bottom of form

**Add Step Actions** (+ button):
- "Add sequential step" (appends after current)
- "Add branch" (converts next steps into branches)
- "Add iterate" (wraps steps in loop)
- "Add conditional" (wraps steps in if/else)

### Flow Map: Tree Visualization (Right Panel)

**Hierarchical Tree Renderer:**
```
Runbook: my-mitigation
├─ Step 1: Gather Evidence [diagnostic action] ✓
├─ Branch: env=prod? [routing]
│  ├─ Step 2a: API Health Check [diagnostic] ⚠ (approval required)
│  ├─ Step 2b: Scale Out [action] ✓
│  └─ Iterate: For each region
│     ├─ Step 2c: Restart Service [action] ✓
│     └─ Step 2d: Validate [diagnostic] ✓
├─ Step 3: Recommend [outcome] ✓
```

**Node Types & Styling:**
- **Step nodes:** `[icon] type | title` + status indicator (✓ valid, ✗ error, ⚠ warning)
- **Branch nodes:** `[fork icon] "condition"` with child count
- **Iterate nodes:** `[loop icon] "collection"` with variable name
- **Conditional nodes:** `[if icon] "expression"`
- **Indentation:** 2 levels per nesting depth
- **Active highlight:** Current form-selected step has light blue background
- **Hover preview:** Tooltip shows step ID, type, and first validation error (if any)

**Interactions:**
- Click any node → select it in the form (left panel scrolls to show its details)
- Hover branch node → highlight all child steps
- Expand/collapse chevron on branch/iterate/conditional nodes
- Right-click node → context menu (delete, duplicate, move up/down, extract as sub-runbook)

### Validation & Governance Indicators

**Inline Validation (in forms):**
- Red text or red underline on fields with schema errors
- Tooltip with error detail on hover
- Invalid fields block save button

**Governance Badge (top bar):**
- **Red:** Schema errors (always block save)
- **Yellow:** Governance warnings (policy violations, approval blocks)
  - Tooltip shows which steps need approval, redaction issues, etc.
  - Allow "force save" with confirmation? (TBD by governance team)
- **Green:** Valid and compliant

**Risk Summary Panel** (collapsible, right side):
- List of all governance-flagged steps with category (e.g., "Step 2a requires approval", "Step 2b has redacted param")
- Each item clickable → jump to step form

### Save & Preview Flow

**1. Save Button (enabled only if valid)**
- User clicks "Save"
- Extension serializes form back to YAML object
- Runs full `validateRunbook()` pass
- If valid: writes to file, shows toast "Saved"
- If invalid: shows error detail, allows edit and re-save

**2. Preview Diff Button**
- Shows unified diff: current YAML file vs. what the form will produce
- Highlights additions (+), deletions (-), changes (modified)
- Allows user to review before committing
- "Looks good" → saves; "Cancel" → close diff, continue editing

**3. Cancel/Revert**
- Discards unsaved changes
- Reloads from file
- Confirms if there are edits

**4. YAML Sync Strategy**
- Form always generates canonical YAML from structured data
- Preserves YAML formatting/comments where feasible (via `roundtrip` YAML parser)
- Non-destructive: only rewrites sections touched by form; unknown fields preserved
- If user edits raw YAML while form is open: form reloads from file to stay in sync

---

## Key Design Decisions (Rationale)

### 1. Form-First vs. Tree-First?

**Decision:** Form-first with synchronized tree visualization.

**Why:** Business users need guided, low-friction step-by-step form editing. Engineers need full structure visibility. The tree map (right panel) serves as a "check" on form operations, making it clear what dependencies and nesting exist without forcing tree-drag interactions that can be error-prone.

**Alternative considered:** Pure tree editor (drag/drop nodes). Rejected: higher friction for common case (adding/editing a sequential step), risk of accidental nesting via drag mishaps.

---

### 2. Flat List vs. Hierarchical Tree?

**Decision:** Hierarchical tree with indentation, but step form presents one level at a time.

**Why:** Preserves branch/conditional/iterate structures that YAML round-trip requires. The "one level at a time" form UI keeps the form simple (no nested forms), while the tree map gives full topology visibility.

**MVP trade-off:** Phase 1 only allows sequential steps + branches + iterate + conditionals at the top level. Post-MVP can support arbitrary nesting depth in tree editor. (Current Kurapika impl flattens; this design allows nesting.)

---

### 3. Edit-in-Place vs. Modal/Dialog?

**Decision:** Edit-in-place form (right side of panel) with no modal.

**Why:** Non-modal keeps editing context visible (tree map, step navigator) and avoids cognitive overload. User can see relationships between steps while editing.

**Alternative:** Context menu → modal form. Rejected: modal hides structure, breaks form-first mental model.

---

### 4. Read-Only vs. Editable Governance Fields?

**Decision:** Read-only badges + collapsible governance panel in Phase 1 MVP. Governance fields (approval, redaction, deny) are visible and explained but not user-editable in the MVP.

**Why:** Governance policies are enforced _at runtime and at policy save time_. The editor shouldn't allow users to override policy; instead, it should warn clearly and block unsafe saves. Policy edits happen in separate admin workflows.

**Post-MVP:** If a "governance editor" becomes a feature, separate tool handles policy definitions; the runbook editor just _displays_ them.

---

## MVP Scope Boundaries

### In MVP (Phase 1)

✅ **Form-based runbook authoring:**
- Name, kind, description fields
- Add/edit/delete sequential steps
- Step type selector (action, branch, iterate, conditional)
- Basic parameter form from tool schema (text, select, toggle inputs)
- Inline validation (schema errors, missing required fields)

✅ **Tree visualization (read-only + navigation):**
- Display branch/iterate/conditional structure
- Click-to-navigate to step form
- Expand/collapse branches
- Hover tooltips

✅ **Governance visibility:**
- Risk/approval badges on step cards
- Governance summary on Runbook Home
- Validation error list
- Color-coded governance warning badge

✅ **Save & serialization:**
- Form → YAML serialization (non-destructive)
- Validation before save (block invalid saves)
- Preview diff
- Cancel/revert

### NOT in MVP (Phase 2+)

❌ **Tree-based branch/step editing** (drag/drop, add branches via tree)  
❌ **Governance policy editing** (governance is read-only, admin-managed)  
❌ **Tool/provider catalog browser** (users type tool IDs; can't browse available tools visually)  
❌ **Advanced parameter editors** (complex object/array type params)  
❌ **Undo/transaction history**  
❌ **Collaborative editing / multi-user sync**  
❌ **Step templates / reusable sub-runbooks**  

---

## UX Principles for Kurapika

1. **Structure visibility over simplicity:** Always show the tree structure (right panel), even if it complicates the form. Business users need to see nesting, branches, and loop boundaries—these affect execution logic. Never flatten.

2. **Form as progress, not endpoint:** The form guides step-by-step editing, but governance and risk are always visible and always checked before save. The form is a tool for entry, not a safety gate; validation is the gate.

3. **Respect YAML as truth:** The form output must be valid, round-trippable YAML that preserves user intent and passes the same validation as hand-written entries. Never force user data into a different shape to fit the form.

---

## Next Steps

1. **Kurapika:** Review this direction; align on form layout, tree rendering tech (SVG/canvas?), and interaction details.
2. **Killua:** Confirm schema introspection APIs are sufficient for form generation (tool selector, parameter types).
3. **Leorio:** Confirm governance metadata is accessible and displayable in the UI (what fields, what format?).
4. **Hisoka:** Define round-trip test harness: form → YAML → form → YAML (must be idempotent).
5. **Prototype:** Build lo-fi mockup (HTML + mock data) to validate layout and interactions before investing in full webview rebuild.

---
