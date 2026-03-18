# UI Specification & Interaction Design: Runbook Visual Editor

**Author:** Kurapika (Frontend Engineer)  
**Date:** 2026-03-18  
**Status:** In Review  
**Scope:** MVP Phase 1 — Form-first authoring with synchronized tree visualization  

---

## 1. Layout & Component Structure

### Page Structure: 2-Column + Header

The editor adopts a **2-column adaptive grid layout** inside the VS Code webview panel:

```
┌─────────────────────────────────────────────────────────────────┐
│  🔗 [Runbook] | 🟢 Valid | [Save] [Preview] [Cancel] [Revert]   │  ← Header Toolbar
├─────────────────────────┬─────────────────────────────────────────┤
│                         │                                          │
│  LEFT PANEL (60%)       │    RIGHT PANEL (40%)                     │
│  ─────────────────      │    ─────────────────                     │
│                         │                                          │
│  [Runbook Home]         │  📊 Flow Map: Tree                       │
│  ─── or ────            │                                          │
│  [Step Navigator]       │  Runbook: my-mitigation                  │
│  • Step 1: Gather       │  ├─ Step 1: Gather... ✓                 │
│  • Step 2: Check        │  ├─ Branch: env=prod?                   │
│  • Step 3: Recommend    │  │  ├─ Step 2a: API Check ⚠             │
│                         │  │  └─ Step 2b: Scale                   │
│  [Step Detail Form]     │  └─ Step 3: Recommend ✓                 │
│  Type: [Action]         │                                          │
│  Title: [________]      │  [Governance Summary]                    │
│  [Params...]            │  • 2 steps need approval                │
│  [Governance Panel]     │  • 1 redaction present                   │
│  [Delete]               │                                          │
│                         │                                          │
└─────────────────────────┴─────────────────────────────────────────┘
```

**Grid Layout (CSS Grid or Flex):**
- **Header:** Fixed height, sticky top, dark background matching VS Code palette
- **Left Panel:** 60% width, scrollable, form container
- **Right Panel:** 40% width, scrollable, tree and summary
- **Breakpoint:** On narrow screens (<1200px), convert to stacked layout (tree below form)

### Component Hierarchy

```
RunbookEditorPanel (root container)
├── Header (sticky)
│   ├── Breadcrumb: [Gert] → [Runbooks] → [runbook-name.runbook.yaml]
│   ├── StatusBadge (🟢 Valid / 🟡 Warning / 🔴 Error)
│   ├── ToolBar
│   │   ├── SaveButton (disabled if invalid)
│   │   ├── PreviewDiffButton
│   │   └── CancelButton
│   └── InfoMenu (?) "Show Tips", "View Schema"
│
├── MainContent (2-column grid)
│   │
│   ├── LeftPanel
│   │   ├── RunbookHomeSection (conditional, shown when no step selected)
│   │   │   ├── NameField (text input)
│   │   │   ├── KindSelector (dropdown)
│   │   │   ├── DescriptionField (textarea)
│   │   │   ├── GovernanceSummaryBadge
│   │   │   └── MetaImportsSection (read-only list of tools/providers)
│   │   │
│   │   ├── StepNavigator (scrollable list)
│   │   │   ├── AddStepButton ("+ Step")
│   │   │   └── StepCard[] (repeatable)
│   │   │       ├── StepIcon (type indicator)
│   │   │       ├── StepTitle (click to select)
│   │   │       ├── MoveUpButton (↑)
│   │   │       └── MoveDownButton (↓)
│   │   │
│   │   ├── StepDetailForm (shown when step selected)
│   │   │   ├── StepHeader
│   │   │   │   ├── TypeBadge (color-coded icon + label)
│   │   │   │   ├── StepIdDisplay (read-only)
│   │   │   │   └── StepTitleField (editable)
│   │   │   │
│   │   │   ├── StepTypeSelector (action/branch/iterate/conditional/end)
│   │   │   │
│   │   │   ├── ParametersPanel (schema-driven, type-specific)
│   │   │   │   ├── FieldGroup[] (logical grouping by schema section)
│   │   │   │   │   ├── SchemaField (input, select, textarea, checkbox)
│   │   │   │   │   ├── ValidationHint (red/orange if error/warning)
│   │   │   │   │   └── TooltipHelp (from schema annotations)
│   │   │   │   │
│   │   │   │   ├── BranchConditionBuilder (if type==branch)
│   │   │   │   │   ├── RoutingKeySelector (dropdown)
│   │   │   │   │   └── ConditionExpressionField (text/structured)
│   │   │   │   │
│   │   │   │   └── IterateBuilder (if type==iterate)
│   │   │   │       ├── CollectionSelector (schema field reference)
│   │   │   │       └── VarNameField (loop variable)
│   │   │   │
│   │   │   ├── GovernanceDetailPanel (collapsible)
│   │   │   │   ├── ApprovalRequiredBadge (yes/no)
│   │   │   │   ├── RedactionList (param names, read-only)
│   │   │   │   ├── DenyListDisplay (env var names, read-only)
│   │   │   │   └── AnnotationNotes (from schema policy)
│   │   │   │
│   │   │   └── DeleteStepButton (destructive action, bottom)
│   │   │
│   │   └── StepActionsPanel (+ button actions)
│   │       ├── AddSequentialStepAction
│   │       ├── AddBranchAction
│   │       ├── AddIterateAction
│   │       └── AddConditionalAction
│   │
│   └── RightPanel
│       ├── FlowMapSection
│       │   ├── TreeRenderer (hierarchical list, indented)
│       │   │   ├── TreeNode[] (recursive)
│       │   │   │   ├── NodeIcon (step/branch/iterate/conditional)
│       │   │   │   ├── NodeLabel (title or condition expression)
│       │   │   │   ├── NodeStatusBadge (✓/✗/⚠)
│       │   │   │   ├── ExpandCollapseChevron (if has children)
│       │   │   │   └── ChildNodeList[] (indented, conditional render if expanded)
│       │   │   │
│       │   │   └── ActiveNodeHighlight (light blue bg on current selection)
│       │   │
│       │   └── TreeContextMenu (right-click node)
│       │       ├── DeleteNodeAction
│       │       ├── DuplicateNodeAction
│       │       ├── MoveUpNodeAction
│       │       ├── MoveDownNodeAction
│       │       └── ExtractAsSubAction
│       │
│       └── GovernanceSummaryDrawer (collapsible)
│           ├── FlaggedStepsList (click to jump to form)
│           ├── RiskCategoryFilter (approval / redaction / deny)
│           └── CountBadges (X approval, Y redaction, Z deny)
│
└── PreviewDiffModal (conditional)
    ├── DiffViewer (unified diff format)
    │   ├── OriginalYAML (left, read-only)
    │   └── GeneratedYAML (right, read-only)
    ├── LineNumbering
    ├── SyntaxHighlight (YAML)
    └── LooksGoodButton / CancelButton
```

### HTML-Level Layout (Vanilla + CSS Grid)

```html
<div class="runbook-editor">
  <div class="editor-header">
    <!-- Breadcrumb, status badge, toolbar -->
  </div>
  
  <div class="editor-main">
    <div class="left-panel">
      <div class="runbook-home-section" id="homeSection" hidden>
        <!-- Metadata form -->
      </div>
      
      <div class="step-navigator-section">
        <button class="add-step-btn">+ Add Step</button>
        <div class="step-cards-list">
          <!-- StepCard elements -->
        </div>
      </div>
      
      <div class="step-detail-form" id="detailForm" hidden>
        <!-- Active step form, schema-driven fields -->
      </div>
    </div>
    
    <div class="right-panel">
      <div class="flow-map-section">
        <div class="tree-renderer">
          <!-- Hierarchical tree nodes -->
        </div>
      </div>
      
      <div class="governance-summary-drawer">
        <!-- Flagged steps, risk categories -->
      </div>
    </div>
  </div>
</div>
```

---

## 2. Information Architecture

### Runbook Home Screen

**When shown:** On initial load, or when user clicks "Back to Overview" / "Runbook Home" breadcrumb.

**Fields (in order):**
1. **Name** (required, text input)
   - Label: "Runbook Name"
   - Placeholder: "e.g., api-outage-mitigation"
   - Validation: non-empty, alpha-numeric + hyphen

2. **Kind** (required, dropdown)
   - Label: "Type"
   - Options: mitigation, monitor, triage, (extensible)
   - Default: mitigation
   - Helps tag runbook purpose for filtering/discovery

3. **Description** (optional, textarea)
   - Label: "Description"
   - Placeholder: "Notes on when/how to use this runbook"
   - Rows: 4-6, expandable

4. **Visual Hierarchy Separator** (divider line)

### Governance Summary Badge (Runbook Home)

Compact 2-row summary:
```
🛡️ Governance Status (click to expand)
├─ ⚠️ 2 steps require approval → {Tool B}, {Tool C}
├─ 👁️ 1 parameter redacted → {Step 1.param-X}
├─ 🚫 2 deny env vars → {PROD_KEY}, {SERVICE_PASS}
└─ ✓ Schema: runbook/v1 (compliant)
```

Clicking expands a drawer with full detail: which steps, which params, which vars. Clicking each item jumps to that step's form.

5. **Meta/Imports Section** (read-only, collapsible)
   - Label: "Tools & Providers Used"
   - List format:
     ```
     Tools:
     • azure-cli (v2.40)
     • nslookup (builtin)
     
     Providers:
     • Azure ServiceManagement
     • Local Shell
     
     Schemas:
     • runbook/v1
     • tool-v0
     ```
   - Sourced from schema introspection on load; displays what the editor discovered.

6. **Meta/Links Section** (quick actions)
   - Button: "View YAML" (opens raw editor in a side panel or new tab)
   - Button: "CLI Preview" (shows CLI command that would execute this runbook)

### Step Navigator (Left Panel, Always Visible)

**Step Card Layout** (per step):
```
┌─ [icon] Action | Gather Evidence        [↑] [↓]
│  ─────────────────────────────────────────────────
│  Type: tool/azure-cli, Status: ✓
│
├─ [icon] Branch | env == "prod" ?
│  ─────────────────────────────────────────────────
│  Status: ⚠ (2 children need approval)
│
├─ [icon] Iterate | for region in regions
│  ─────────────────────────────────────────────────
│  Status: ✓
│
└─ [icon] Conditional | on_error == true
   ─────────────────────────────────────────────────
   Status: ✓ (1 child)

[+ Add Sequential Step]
```

**Card Elements:**
- **Icon** (colored, type-specific): ⚙️ action, 🔀 branch, 🔁 iterate, ✓ conditional, ⏹️ end
- **Title/Label** (clickable → selects step, populates form)
- **Metadata Line** (type, status)
- **Move Buttons** (↑ move up, ↓ move down; disabled if already at boundary)
- **Visual Indentation** (2x standard indent per nesting level for branch/iterate children)

**Scrollable:** Navigator scrolls independently; selected card stays in viewport.

### Step Detail Form (Left Panel, Conditional)

**Header:**
- Type icon + label (color-coded)
- Step ID (read-only, gray text, e.g., "step-id-001")
- Title field (editable text input)

**Type Selector:**
- Dropdown: "Step Type"
- Options: 
  - **Action** → rendered fields depend on next selector (tool/provider)
  - **Branch** → routing key + conditions
  - **Iterate** → collection + loop var
  - **Conditional** → expression field
  - **End** → outcome/result field

**Dynamic Parameters Panel (Schema-Driven):**

For **Action steps:**
- Tool selector: dropdown of available tools from catalog (or text input for MVP)
- Action ID: dropdown of available actions for selected tool (from tool schema)
- Action Parameters: rendered from tool schema
  - Text inputs, selects, checkboxes, textareas
  - Required fields marked with `*`
  - Inline validation hint under field: red text if error, orange if warning
  - Contextual help in tooltip: hover for annotation from schema

For **Branch steps:**
- Routing Key: text input or dropdown of available keys from runbook context
- Condition Expression: textarea or structured builder (MVP: raw text input)
- Branches UI (preview):
  - List of branches this condition creates (read-only in MVP)
  - Example: "Condition will route to branches: [true → steps 2a, 2b], [false → step 3]"

For **Iterate steps:**
- Collection: dropdown of available collections (from runbook vars or schema)
- Loop Variable: text input (e.g., "region")
- Iteration bound: UI shows "Loop vars: region (from {source})"

For **Conditional steps:**
- Expression: textarea with syntax hint
- Validation: "Parse OK" or error detail

For **End steps:**
- Outcome field: dropdown of outcomes (success/failure/escalate)
- Notes: optional textarea

**Governance & Risk Panel (Collapsible, Titled "Policy & Safety"):**
- **Approval Required:** Yes/No badge (red if yes)
  - If yes, shows which service/role requires approval
  - Tooltip: "Step blocked at runtime until approved via {approval-url}"
- **Redaction:** 
  - If present, shows redacted param names in a gray list
  - Tooltip: "These values will be hidden in logs and audit trails"
- **Deny List:**
  - Shows environment variable names that will NOT be passed to this step
  - Tooltip: "Blocked for security; these secrets not allowed in this tool"
- **Annotations:** 
  - Author notes from policy (e.g., "Critical path; requires enterprise SLA")

**Delete Step Button:**
- Destructive action, red/warning styling, bottom of form
- Click → confirmation dialog: "Delete this step? This cannot be undone."
- On confirm → removes from list, updates tree, clears form

### Flow Map (Right Panel, Always Visible)

**Hierarchical Tree Renderer:**

Rendered as an indented list, not a DOM tree. Each node is a row:

```
Runbook: my-mitigation
├─ 📍 Step 1: Gather Evidence
│  Status: ✓ (valid, no warnings)
│  
├── 🔀 Branch: env == "prod"?
│  Status: ⚠ (2 children flagged)
│  ├─ 📍 Step 2a: API Health Check
│  │  Status: ⚠ (approval required)
│  │  
│  ├─ 📍 Step 2b: Scale Out
│  │  Status: ✓
│  │  
│  └─ 🔁 Iterate: for region in regions
│     Status: ✓
│     ├─ 📍 Step 2c: Restart Service
│     │  Status: ✓
│     │  
│     └─ ✓ Step 2d: Validate
│        Status: ✓
│
└─ 📍 Step 3: Recommend
   Status: ✓
```

**Node Rendering:**
- **Indentation:** 2 pixels per nesting level (or use flexbox column-gap + padding)
- **Tree connectors:** CSS Unicode box-drawing (├, ├─, │, └─, etc.) or SVG lines (lightweight)
- **Icon:** Emoji or inline SVG (⚙️ action, 🔀 branch, 🔁 iterate, ✓ conditional, ⏹️ end)
- **Label:** Step title or condition expression (truncate if >50 chars, show tooltip on hover)
- **Status badge:** Right-aligned
  - ✓ Green (valid, no warnings)
  - ⚠️ Yellow (warnings, e.g., approval required)
  - ✗ Red (errors)
- **Expand/Collapse chevron:** Left of icon (if node has children); click to toggle visibility
- **Active highlight:** Light blue background on the node corresponding to currently selected step in form
- **Hover effect:** Light gray background, node ID tooltip

**Interactions on Tree Nodes:**
- **Click:** Select node → jump to step form on left (panel auto-scrolls to that form)
- **Right-click:** Context menu
  - Delete node
  - Duplicate node
  - Move up / Move down
  - Extract as sub-runbook (future)
- **Expand/Collapse:** Click chevron to toggle children visibility (state persisted per session)
- **Hover any node:** Show tooltip with full node details (ID, type, first error if any)

### Governance Summary Drawer (Right Panel, Collapsible)

**Title:** "Policy & Safety" (matches governance panel title on form)

**Content:**
- **Tab 1: Approval Queue**
  - List: Each line is clickable, shows
    - Step title + step ID
    - Service/role requiring approval
    - (Severity badge if available)
  - Click any line → jump to step form on left
  - Empty state: "All steps approved ✓"

- **Tab 2: Redaction**
  - List: Step title, redacted param names
  - Click → jump to step form
  - Empty state: "No redacted parameters"

- **Tab 3: Deny List**
  - List: Step title, env var names denied
  - Click → jump to step form
  - Empty state: "No deny rules"

**Count badges** (at drawer top):
- 5 Approvals | 2 Redacted | 3 Deny (red/yellow/neutral color)

---

## 3. Interactions

### User Adds a Step

**Scenario:** User clicks "+ Add Step" button

**Flow:**
1. Modal/Inline dialog appears: "What type of step?"
   - Buttons: [Action] [Branch] [Iterate] [Conditional] [End]
2. User clicks a type
3. Form appears with that type pre-selected
   - All required fields are empty or show placeholders
   - Optional fields are hidden or collapsible
4. User fills in required fields (type selector, tool/provider if Action)
5. Step is added to end of step list (step navigator)
6. Tree view updates to show new node (expanded parent if within a branch)
7. Step auto-selected; form shows its details
8. User can continue editing or click another step

**Alternative (faster):** "Add sequential step" is the default action of the "+ Step" button; "Add branch/iterate" are menu items (or always available via dropdown).

### User Clicks a Step in Tree

**Scenario:** User clicks a node in the Flow Map (right panel)

**Flow:**
1. Node is highlighted (light blue background)
2. Left panel automatically scrolls to show the step form
3. Step detail form populates with that step's current data
4. All validators/hints run
5. If step has governance flags, the governance panel shows them

### User Edits a Step Parameter

**Scenario:** User fills in a "Tool Action Parameters" field in the form

**Flow:**
1. User begins typing or selecting
2. Real-time validation runs:
   - Schema type checking (text vs. number vs. enum)
   - Required field check
   - Regex/pattern validation if present
3. **Invalid:** Field shows red underline + error text below
4. **Valid:** No error text; optional check-mark icon
5. Save button remains disabled if any field has an error
6. On every keystroke, form updates internal state (not yet saved to YAML)

### User Edits a Branch Condition

**Scenario:** User clicks on a Branch step in the form

**Flow:**
1. Form shows:
   - Type selector (pre-selected: "Branch")
   - Routing Key field (dropdown or text)
   - Condition Expression field (textarea in MVP)
   - Branch preview list (read-only; shows which branches this condition creates)
2. User updates condition text
3. Inline hint shows: "Syntax OK" or "Error: invalid expression (expected 'key op value')"
4. Saves same way as any other step (only when user clicks "Save" button at top)

### User Saves the Runbook

**Scenario:** User clicks "Save" button at top

**Flow:**
1. Button is **disabled** if any step has validation errors
2. If enabled, user clicks "Save"
3. **Option A (Immediate Save):** 
   - Extension serializes form to YAML object
   - Runs `validateRunbook()` pass
   - If valid: writes to file, shows toast "✓ Saved"
   - If invalid: error detail shown, form stays open for edits
4. **Option B (Preview First):**
   - User clicks "Preview" button instead
   - Diff modal opens showing unified diff: current YAML vs. generated YAML
   - User reviews changes
   - If looks good: clicks "Save" in diff modal → writes file
   - If wants to edit: clicks "Cancel" in diff modal → closes diff, resumes form editing

### User Opens a Complex Runbook

**Scenario:** User opens a runbook with branches, iterate, conditionals

**Flow:**
1. File loads; extension parses YAML
2. Tree is extracted (multi-level hierarchy preserved)
3. Left panel shows Runbook Home by default
4. Right panel shows full tree (all nodes visible, indented, most collapsed by default)
5. All branches/iterate/conditional nodes have expand/collapse chevrons
6. User clicks chevron to expand a branch → children become visible
7. User clicks a step node in the tree → form jumps to that step

**UI Hint:** "Runbook has branches (3 subtree(s) found). Expand tree at right to review structure."

### User Validates

**Scenario:** At any time, user can see current validation status

**Flow:**
1. Header badge shows:
   - 🟢 Green: "Valid & Safe" if no errors and no governance warnings
   - 🟡 Yellow: "{N} warnings" (e.g., "2 approvals required", click to show list)
   - 🔴 Red: "{N} errors" (e.g., "2 fields invalid"), Save button disabled
2. Governance Summary drawer (right panel) updates in real-time as user edits
3. Individual field errors are shown under the field in red text
4. Clicking an error in Governance drawer jumps to that step/field

---

## 4. Visual Language

### Color Palette (VSCode Theme Tokens)

All colors use `var(--vscode-*)` CSS variables for automatic light/dark mode support:

| Element | CSS Variable | Fallback (light/dark) | Usage |
|---------|--------------|----------------------|-------|
| **Text (primary)** | `--vscode-editor-foreground` | #333/#e0e0e0 | Labels, titles, body text |
| **Text (secondary)** | `--vscode-descriptionForeground` | #666/#999 | Hints, secondary labels, metadata |
| **Background** | `--vscode-editor-background` | #fff/#1e1e1e | Main editor area |
| **Background (panel)** | `--vscode-panel-background` | #f5f5f5/#252526 | Left/right panels |
| **Border** | `--vscode-panel-border` | #ddd/#3e3e42 | Section separators, panel edges |
| **Input field** | `--vscode-input-background` | #fff/#3c3c3c | Text inputs, textareas |
| **Input border (focus)** | `--vscode-inputOption-activeBorder` | #007acc (blue) | Focus ring on inputs |
| **Success (✓)** | `--vscode-testing-iconPassed` | #107c10 (green) | Valid status, checkmarks |
| **Warning (⚠️)** | `--vscode-inputValidation-warningBackground` | #fff3a8/#464414 | Warnings, yellow badge |
| **Error (✗)** | `--vscode-inputValidation-errorBackground` | #f48771/#5f2c2c (red) | Errors, validation failures |
| **Highlight (active)** | `--vscode-editor-selectionBackground` | #add6ff/#264f78 | Active step highlight in tree |
| **Button (primary)** | `--vscode-button-background` | #007acc (blue) | Save, preview buttons |
| **Button (secondary)** | `--vscode-button-secondaryBackground` | #3e3e42 (gray) | Cancel, revert buttons |
| **Link** | `--vscode-textLink-foreground` | #007acc (blue) | Clickable labels, icons |

### Typography

- **Headers (section titles):** Font-weight: 600, size: 16px
- **Form labels:** Font-weight: 500, size: 13px
- **Body/helper text:** Font-weight: 400, size: 12px
- **Monospace (IDs, code):** Font-family: monospace, size: 12px
- **Line height:** 1.5x standard

### Icons (Emoji + Inline SVG)

**Step types:**
- ⚙️ **Action** (tool/command) — blue background
- 🔀 **Branch** (conditional routing) — purple background
- 🔁 **Iterate** (loop) — orange background
- ✓ **Conditional** (if/else) — green background
- ⏹️ **End** (exit/outcome) — gray background

**Validation Status:**
- ✓ **Valid** — green checkmark
- ⚠️ **Warning** — yellow exclamation
- ✗ **Error** — red X or stop sign

**Governance:**
- 🔐 **Approval required** — lock/shield icon
- 👁️ **Redaction** — eye-slash or redaction bar
- 🚫 **Deny** — prohibition sign
- ℹ️ **Info** — info circle (for help tooltips)

**Actions:**
- ↑ **Move Up** — up arrow
- ↓ **Move Down** — down arrow
- ✕ **Delete/Close** — X symbol
- 📋 **Copy** — clipboard icon
- + **Add** — plus sign

### Spacing & Layout

- **Panel gutter:** 16px (between left/right panels)
- **Form field spacing:** 12px (vertical gap between fields)
- **Section padding:** 16px (inside left/right panels)
- **Tree node vertical gap:** 4px (between indented rows)
- **Button size:** 36px height, 12px padding left/right (VSCode standard)

### Animations (Subtle, Optional for MVP)

- **Expand/collapse tree node:** 150ms ease-out (chevron rotates, children fade in)
- **Form transition:** 100ms fade-in when switching between steps
- **Save toast:** Fade in 100ms, hold 2s, fade out 200ms
- **Hover states:** 50ms background color transition on interactive elements

---

## 5. Form Generation (Schema-Driven Fields)

### Field Rendering Logic

For each step type, schema defines what fields appear and how they're rendered.

**Input:** Step type (action/branch/iterate/conditional) → lookup schema for that type → render fields

**Schema source (MVP):**
- Hardcoded schema struct in Go (generated from `schema/runbook.go`)
- Executor fetches schema on startup via `/api/schema` endpoint (future: Killua backend)
- VS Code extension caches schema, regenerates forms from cached schema

**Field Type Mapping:**

| Schema Type | HTML Element | User Input | Validation |
|-------------|--------------|------------|-----------|
| `string` | `<input type="text">` | Any text | Length check, regex if present |
| `enum` (predefined list) | `<select>` | Dropdown choice | Enum membership check |
| `number` | `<input type="number">` | Numeric input | Range check (min/max) |
| `boolean` | `<input type="checkbox">` | On/off toggle | (no validation needed) |
| `text` (multiline) | `<textarea>` | Free text, multiline | Length check |
| `object` (structured) | Nested fieldset | Expands to sub-fields | Recursive validation |
| `array` | Repeatable field group | Add/remove rows | Item type validation |

**Example: Action Step ("Tool Action")**

Schema for action step:
```json
{
  "type": "action",
  "fields": {
    "tool": { "type": "enum", "options": ["azure-cli", "nslookup", "curl"], "required": true },
    "action": { "type": "enum", "required": true, "options": "depend on tool selected" },
    "params": { "type": "object", "properties": { ... }, "required": false }
  }
}
```

Form rendering:
```
[Step Type] Action (select box)
│
├─ [Tool] [azure-cli ▼]  (required, dropdown)
│
├─ [Action] [scale-out ▼]  (required, depends on tool)
│   (Hint: "Available actions for azure-cli")
│
├─ Parameters
│  ├─ [Resource Group] [_________]  (text, required)
│  ├─ [VM Count] [___]  (number, optional)
│  ├─ [Timeout (sec)] [___]  (number, optional, default 300)
│  └─ [Force?] ☐  (checkbox, optional)
│
└─ [Governance & Risk] ▼ (collapsible)
```

### Conditional Field Visibility

**Rules:**
- If a field has `"conditions": [{"field": "tool", "equals": "azure-cli"}]`, it ONLY shows if that condition is true.
- Dependent fields update in real-time when parent field changes (e.g., changing tool selector re-renders the action dropdown).

**Example:** Tool selector is "azure-cli" → Action dropdown shows azure-specific actions. User changes tool to "nslookup" → Action dropdown updates.

### Validation Rules (Per-Field)

Schema includes `"validate"` rules:
```json
{
  "name": "Resource Group",
  "type": "string",
  "required": true,
  "validate": {
    "pattern": "^[a-zA-Z0-9-]{3,64}$",
    "message": "Resource group name must be 3-64 characters, alphanumeric and hyphens only"
  }
}
```

**Renderer:** When user exits field (blur event) or after keystroke:
1. Check required: show error if empty and required
2. Check pattern: show error if regex fails
3. Show inline error text under field (red, small font)
4. Disable Save button if any field error

---

## 6. MVP Scope Boundaries

### What's IN Scope

✅ **Step CRUD:** Add, edit, delete, reorder steps (linear list)  
✅ **Branch/Iterate/Conditional Support:** Limited to top-level and one level of nesting  
✅ **Runbook Metadata:** Name, kind, description editing  
✅ **Schema-Driven Forms:** Fields render from schema for action/branch/iterate/conditional steps  
✅ **Real-Time Validation:** Per-field + full-document validation before save  
✅ **Governance Visibility:** Approval/redaction/deny badges (read-only display, no editing)  
✅ **Tree Visualization:** Read-only hierarchical tree with expand/collapse  
✅ **Save/Preview:** Serialize to YAML, preview diff, save to file  
✅ **YAML Round-Trip Fidelity:** Comments and formatting preserved where feasible  

### What's OUT of Scope (Phase 2+)

❌ **Drag/Drop Tree Editing:** Complex, high-friction, high-error risk (defer)  
❌ **Tool Catalog Picker UI:** Users type tool names manually in MVP (Killua integration later)  
❌ **Undo/Redo:** Requires transaction model we don't have yet  
❌ **Comments/Annotations:** YAML library doesn't support in-line comment preservation  
❌ **Policy Editing:** Policy fields are display-only; editing happens in separate governance admin tool  
❌ **Multi-Branch Collapse/Expand State Persistence:** Expand/collapse state is session-only  
❌ **Arbitrarily Deep Nesting:** MVP limits to top-level + 1–2 levels deep  
❌ **Diff/Merge:** No version control inside editor; work with git externally  

---

## Component Checklist (What Needs Building)

### Web Components / React Components

- [ ] **RunbookEditorPanel** (root container, Webview lifecycle)
- [ ] **EditorHeader** (breadcrumb, status badge, toolbar, save/cancel/preview buttons)
- [ ] **StatusBadge** (🟢/🟡/🔴, click to show detail)
- [ ] **LeftPanel** (container for all left-side content)
  - [ ] **RunbookHomeSection** (name, kind, description fields, governance badge)
  - [ ] **StepNavigatorSection** (scrollable step cards, + button for add step)
  - [ ] **StepCard** (repeatable, type icon, title, move/delete buttons)
  - [ ] **StepDetailForm** (dynamic schema-driven fields, type selector)
    - [ ] **SchemaField** (generic input renderer: text, select, number, checkbox, textarea)
    - [ ] **FieldGroup** (logical grouping of fields)
    - [ ] **FieldError** (inline validation hint)
    - [ ] **BranchConditionBuilder** (for branch step type)
    - [ ] **IterateBuilder** (for iterate step type)
    - [ ] **GovernancePanel** (approval/redaction/deny display)
  - [ ] **DeleteStepButton** (destructive, confirmation dialog)
- [ ] **RightPanel** (container for tree + governance drawer)
  - [ ] **FlowMapSection** (tree renderer)
    - [ ] **TreeNode** (recursive, indented, expand/collapse chevron)
    - [ ] **TreeContextMenu** (right-click, delete/duplicate/move)
    - [ ] **NodeHighlight** (light blue bg for active node)
  - [ ] **GovernanceSummaryDrawer** (approval/redaction/deny tabs)
- [ ] **PreviewDiffModal** (modal + diff viewer)
  - [ ] **DiffViewer** (unified diff, line numbers, syntax highlight)
- [ ] **AddStepModal** (type selector dialog)
- [ ] **ConfirmDeleteDialog** (confirmation before destructive action)

### Utilities / Helpers

- [ ] **SchemaManager** (load schema, lookup field type, resolve dependencies)
- [ ] **FormStateManager** (track current step, form values, validation state)
- [ ] **YAMLSerializer** (convert form state back to YAML object)
- [ ] **ValidationEngine** (run per-field and full-document validation)
- [ ] **TreeBuilder** (parse YAML into hierarchical tree structure)
- [ ] **DiffGenerator** (create unified diff for preview)

### Styling

- [ ] **CSS Grid layout** (2-column responsive)
- [ ] **VSCode theme tokens** (color palette)
- [ ] **Form styling** (inputs, buttons, sections)
- [ ] **Tree styling** (indentation, icons, hover/active states)
- [ ] **Animations** (optional: expand/collapse, fade-in transitions)

---

## Open Questions

1. **Tool Catalog Provider:** In MVP, users type tool names manually. Should we hard-code a list of known tools, or accept any string?
   - **Decision:** Accept any string; Killua will validate at save time.

2. **Governance Editing:** Can users edit approval/redaction/deny fields, or are they locked?
   - **Decision:** Read-only display in MVP. Policy editing is out-of-scope.

3. **Branch Condition Syntax:** Should conditions be free-text expressions, or structured (form-builder style)?
   - **Decision:** Free-text (MVP faster). Structured condition builder in Phase 2.

4. **Undo/Redo:** Needed for MVP, or acceptable to defer?
   - **Decision:** Defer. Users can Ctrl+Z at file level; unsaved form changes can be reverted with "Revert" button.

5. **Tree Expand/Collapse Persistence:** Save state to file, or session-only?
   - **Decision:** Session-only (simpler, no YAML mutation).

---

## Summary for Developer

**Layout:** 2-column (60/40 split), left form + right tree, sticky header with save/cancel.  

**Key Interactions:**
- Click step in tree → form jumps to step
- Edit form field → real-time validation, disable Save if error
- Click Save → validate, diff preview, serialize to YAML
- Expand tree nodes → reveal nested branches/iterate/conditionals

**Component Count:** ~25 web components + 6 utility modules.

**Governance UX:** Red/yellow/green badges; read-only policy display; click to jump to flagged steps.

**Schema Integration:** Fields render from JSON schema; tool/branch/iterate/conditional have unique field sets.

**MVP Constraints:** No drag/drop, no tool catalog UI, no undo/redo, no policy editing, limited nesting depth.

---
**NEXT STEP:** Kurapika to build SchemaField + StepDetailForm components; Hisoka to define test cases for YAML round-trip fidelity.
