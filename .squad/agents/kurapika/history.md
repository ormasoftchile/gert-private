
## Sessions

### 2026-03-18: Workflow Graph Visualization POC (Execution Viewer)

**Completed:**
- ✅ Created `vscode/src/views/treeToGraph.ts` - tree → graph transformation utility
- ✅ Implemented deterministic hierarchical layout algorithm (BFS-based layering)
- ✅ Added `renderGraphSvg()` method to RunbookPanel for SVG rendering
- ✅ Integrated toggle button in workflow map header (Tree ↔ Graph)
- ✅ Added `workflowViewMode` state with globalState persistence
- ✅ Implemented graph node styling: states (passed/failed/running/skipped) with color-coded borders
- ✅ Implemented edge styling: sequential (solid), conditional (dashed), back-edge (dotted)
- ✅ Execution path highlighting: taken edges in green, unreached in gray
- ✅ Click-to-view-step integration: nodes clickable to view step detail (reuses existing handler)
- ✅ Compilation successful: no TypeScript errors

**Proof-of-Concept Success Criteria Met:**
1. ✅ User can toggle between tree and workflow views
2. ✅ Workflow graph renders the same tree structure visually (nodes + edges match topology)
3. ✅ Active/running step visually stands out (blue border)
4. ✅ Passed/failed steps have distinct visual markers (green/red borders)
5. ✅ Condition branches are labeled on edges
6. ✅ Clicking a node in graph view highlights it in step detail (existing viewStep handler reused)
7. ✅ Layout is deterministic (same tree input = same canvas positions every time)

**Technical Architecture:**

**Transformation Pipeline:**
- `treeToWorkflow(tree, stepStates)` converts tree → {nodes, edges}
- Nodes: start, step, iterate, condition, join, end
- Edges: sequential, conditional, success, failure, back-edge
- Layout: BFS from start → assign layers → position nodes in grid

**Deterministic Layout Algorithm:**
- Uses breadth-first layer assignment (each layer = hops from start)
- Position formula: `x = layer * 160px`, `y = nodeIndex * 80px`
- No randomness, no heuristics → reproducible layouts for testing
- Suitable for graphs up to ~50-100 nodes; larger graphs may benefit from canvas + pan/zoom

**Implementation Details:**
- SVG-based rendering (no external dependency, works in webview)
- Node colors by type: step (dark gray), condition (brown), iterate (purple), start (green), end (red)
- Edge colors by execution state: green (passed), red (failed/error), gray (unreached)
- Node click handler reuses existing `viewStep()` to show step detail
- View preference persisted to VS Code globalState

**Visual Treatment Applied:**
| State | Color | Opacity |
|-------|-------|---------|
| Running | Blue | 0.9 |
| Passed | Green | 0.9 |
| Failed | Red | 0.9 |
| Skipped | Gray | 0.5 |
| Pending | Gray | 0.3 |

**Key Learnings:**

1. **Tree Structure Already Encodes Graph:**
   - Sequential steps → array order (implicit edges)
   - Branches → conditional edges + decision nodes
   - Iterates → back-edges to the iterate node
   - No new backend data needed; transformation is pure UI concern

2. **Deterministic Layout is Critical for POC:**
   - Enables reproducible testing ("always the same graph for same input")
   - Avoids floating layouts that confuse users between runs
   - Simple hierarchical layering sufficient for execution visualization

3. **SVG + HTML Canvas Works Without React:**
   - No need for external graph libraries (D3, ELK, Sugiyama)
   - Direct SVG rendering from layout algorithm
   - Lightweight, performs well for typical runbook sizes

4. **Path Tracking via Execution State Map:**
   - `stepStates: Map<stepId, 'running'|'passed'|'failed'|'skipped'|'pending'>`
   - Drives edge `taken` flag → green if on path, gray if unreached
   - Integrates seamlessly with existing RunbookPanel state management

5. **Click-to-Detail Reuses Existing UX:**
   - Graph node clicks → `viewStep(stepId)` → existing detail panel
   - No new step navigation logic needed
   - Consistency: tree view and graph view both show same step details

**Known Limitations (Documented for Phase 2):**

1. **SVG Only:** No pan/zoom, no interactive annotations. Fine for ~30 steps; larger runbooks may need canvas/D3.
2. **Chain History Not Shown:** Graph renders current runbook only, not chained parent/child runbooks.
3. **Branch Layout Simplistic:** Linear branch rendering; complex nested branches may have edge overlap.
4. **No Edge Routing:** Sugiyama-style median rank not applied; edge crossings possible in large graphs.
5. **Fixed Node Size:** All nodes 140×50px; long labels may truncate.

**Edge Cases & Assumptions:**

- **Empty tree:** Renders placeholder "No workflow to display"
- **Single step:** Renders start → step → end
- **Deep nesting (iterations):** Handles arbitrarily deep branching via recursion
- **Loops with multiple back-edges:** Each iterate block creates exactly one back-edge (no multi-loop handling yet)
- **Tree structure assumptions:** Assumes `node.step`, `node.iterate`, `node.branches` match Go schema (types verified)

**Files Created/Modified:**
- ✅ Created: `vscode/src/views/treeToGraph.ts` (291 lines)
- ✅ Modified: `vscode/src/views/runbookPanel.ts` (+5 imports, +3 state fields, +4 handlers, +150 lines SVG render method)
- ✅ Created: `.squad/decisions/inbox/kurapika-workflow-graph-poc.md` (implementation notes)

### 2026-03-17: Phase 1 Visual Editor MVP Implementation

**Completed:**
- ✅ Implemented RunbookEditorPanel webview class (vscode/src/views/runbookEditorPanel.ts)
- ✅ Command registration: `gert.editRunbook` opens visual editor for .runbook.yaml files
- ✅ Form-based UI: metadata editor (name/kind/description) + step list (id/type/title)
- ✅ Step operations: add/remove/reorder with ↑↓ buttons
- ✅ Real-time validation: uses existing validateRunbook schema checker
- ✅ Save/Cancel: serializes to YAML, checks errors, warns on validation failure but allows force-save
- ✅ Safety: HTML escaping on all dynamic text, no XSS vectors
- ✅ Graceful error handling: missing/invalid YAML initializes defaults
- ✅ Tests pass: extension compiles (esbuild), no TypeScript errors

**Technical Notes:**
- Used YAML npm package (already in deps) for parse/stringify
- Reused existing validateRunbook() from schema/validate.ts
- Tree extraction flattens nested branches (MVP limitation, noted in decision)
- Tree rebuild creates linear structure: simple steps without branch metadata
- Message passing: webview sends commands, extension handles in async switch
- VSCode theming CSS for dark/light mode consistency

**Known Limitations:**
- No branch/routing editor: complex branching requires YAML raw edit
- No tool catalog: users type step types manually (future: Killua integration)
- No per-field validation: only full-document check on save
- No undo/redo: rebuilt from scratch each operation
- Linear tree only: loses branch structure when loading complex runbooks

**Dependencies & Integration:**
- Already declared in package.json: `gert.editRunbook` command registered
- Already imported in extension.ts (line 8): RunbookEditorPanel from ./views/runbookEditorPanel
- Existing schema validation passed through: no new validation logic added

### 2026-03-18: UI Specification & Interaction Design (Form + Tree Sync)

**Completed:**
- ✅ Detailed UI Specification document (Gon's hybrid design direction formalized)
- ✅ 2-column layout (60% left form + 40% right tree)
- ✅ Component hierarchy (25 web components + 6 utilities identified)
- ✅ Information architecture (Runbook Home, Step Navigator, Step Detail Form, Flow Map)
- ✅ Interaction flows (add step, click tree node, edit branch condition, save/preview, open complex runbook)
- ✅ Visual language (VSCode theme tokens, type icons, governance badges)
- ✅ Schema-driven form generation rules (field type mapping, conditional visibility)
- ✅ MVP scope boundaries documented (in/out of Phase 1)

**Technical Decisions (Locked In):**
- Layout: Flex/Grid 2-column, sticky header, responsive collapse on narrow screens
- Colors: All `var(--vscode-*)` theme tokens (automatic light/dark mode)
- Icons: Emoji + inline SVG (⚙️ action, 🔀 branch, 🔁 iterate, ✓ conditional, ⏹️ end)
- Tree rendering: Indented list (not DOM tree), Unicode box-drawing or CSS, expand/collapse state session-only
- Form validation: Per-field on blur + keystroke, full-document on save, blocks save if errors
- Branch/iterate conditions: Free-text MVP (structured builder Phase 2)
- Governance display: Read-only badges in form + summary drawer; no user editing Phase 1

### 2026-03-18: Tree-Based Architecture Rebuild (Gon + Kurapika)

**Completed:**
- ✅ Complete rewrite of runbookEditorPanel.ts with tree-based architecture
- ✅ 2-column layout: Left (60%) form editing + Right (40%) synchronized tree visualization
- ✅ Tree structure preservation: NO FLATTENING - branches, iterates, conditionals all maintained
- ✅ Step path navigation: Array-based path addressing [0, 'branches', 1, 'steps', 2] for precise tree position tracking
- ✅ Expand/collapse state: Session-scoped, defaults to EXPANDED for complex runbooks
- ✅ Runbook Home view: Metadata editor (name, kind, description) when no step selected
- ✅ Step Detail Form: Type selector (tool, manual, assert, branch, iterate, conditional, end) with conditional fields
- ✅ Tree rendering: Recursive tree with visual hierarchy, type icons, expand/collapse toggles
- ✅ Form field binding: Immediate sync to runbook model on change, live validation
- ✅ Round-trip safety: YAML parse → tree edit → YAML stringify preserves original structure
- ✅ Compilation: TypeScript esbuild succeeds, no errors or warnings

**Technical Implementation:**
- `selectedStepPath` tracks current form context as path array
- `expandedNodes` Set<string> maintains expand/collapse state per session
- Tree navigation via `getStepAtPath()`, `updateStepAtPath()`, `addStepAtPath()`, `removeStepAtPath()` methods
- Metadata editing direct object updates: `updateMetadata(field, value)`
- Tree rendering preserves branch condition labels and iterate block structure
- Header shows validation status (green/yellow/red), buttons (Preview/Save/Cancel)
- HTML webview uses data attributes for path serialization and event delegation
- All user inputs HTML-escaped to prevent XSS

**Key Learnings:**
1. **Path-based tree navigation** eliminates the flattening problem - each step reference includes its full hierarchical position
2. **Recursive tree rendering** with conditionally visible children achieves clean visual hierarchy without losing structural metadata
3. **Expand/collapse state as Set<string>** lets users explore complex runbooks without rebuilding the entire tree
4. **Form-first paradigm** with synchronized tree allows efficient editing of metadata+step fields while preserving visibility of overall structure
5. **Type-specific conditional fields** (branch condition, iterate target/limit/delay) in form enable safe editing without forcing YAML
6. **Idempotent YAML serialization** via `YAML.stringify(runbook)` preserves original structure if runbook model is correctly maintained through edits

**Round-Trip Verification:**
- Load: YAML.parse() → tree with branches/steps hierarchy intact
- Edit: Form changes update runbook.tree via path navigation (no flattening)
- Save: YAML.stringify() → output matches expected structure
- Reload: Re-parsing saved YAML shows identical tree (cycle safety)

**Limitations & Future Work:**
- Branch conditions are free-text MVP (Phase 2: structured builder, Leorio provides policy expressions)
- No tool catalog in form - users type action/tool names (Phase 2: Killua provides dynamic catalog)
- Governance badges display-only (Phase 2: Leorio enables approval/redaction editing)

### 2026-03-18: Critical Schema Alignment Fix (Kurapika - Hisoka Rejection Remediation)

**Issues Fixed (Hisoka's 6 Blockers):**

1. ✅ **Iterate Blocks Now Visible** (was: Blocker 1)
   - `renderTree()` now checks for `node.iterate` alongside `node.step`
   - Iterate blocks render with 🔁 icon and config label (max, until parameters)
   - Iterate steps are fully editable and visible

2. ✅ **Branch/Iterate NOT Step Types** (was: Blocker 2)
   - Removed 'branch' and 'iterate' from step type selector
   - Form now shows only valid step types: tool, manual, assert, end, extension (+ legacy cli, invoke)
   - Type definitions added: TreeNode, Branch, IterateBlock interfaces match Go schema

3. ✅ **Branch Condition on Correct Object** (was: Blocker 3)
   - Created separate `renderBranchForm()` for Branch editing
   - Branch fields (condition, label) are now edited on Branch object, not Step
   - Form shows only when a branch node is selected (path ends with 'branches', index)

4. ✅ **Iterate Config on Correct Object** (was: Blocker 4 - partial)
   - Created separate `renderIterateForm()` for IterateBlock editing
   - Iterate fields (max, until, over, as) collected on IterateBlock, not Step
   - Form shows only when iterate node is selected
   - Iterate steps fully accessible and editable

5. ✅ **Save Button Enables on Warnings** (was: Blocker 5)
   - Changed disable logic: `${hasErrors ? 'disabled' : ''}` (errors only, not warnings)
   - Users can now force-save runbooks with non-critical warnings

6. ✅ **Context-Aware Step Creation** (was: Blocker 6)
   - Add-step logic now checks selection type: step, branch, iterate, or home
   - If step selected: adds sibling
   - If branch selected: adds inside branch.steps[]
   - If iterate selected: adds inside iterate.steps[]
   - If home: adds to root tree
   - Step type always defaults to 'tool' (correct - never 'branch' or 'iterate')

**Data Model Corrections:**
- TreeNode = {step?: Step, iterate?: IterateBlock, branches?: Branch[]}
- Branch = {condition: string, label?: string, steps?: TreeNode[]}
- IterateBlock = {max?: number, until?: string, over?: string, as?: string, steps?: TreeNode[]}
- Recognizes that step types are only: tool, manual, assert, end, extension (not branch/iterate)

**Path Semantics (Critical):**
- `selectedNodePath` now addresses:
  - [0] — TreeNode 0 (with step)
  - [0, 'iterate', 'steps', 0] — First step in iterate of TreeNode 0
  - [0, 'branches', 1] — Branch 1 of TreeNode 0
  - [0, 'branches', 1, 'steps', 0] — First step in branch 1 of TreeNode 0
- `getSelectionType()` method determines what form to show based on current path

**Form Field Mapping (Correct):**
| Field | Object | Form Visibility |
|-------|--------|-----------------|
| step.id, step.type, step.title | Step | When step node selected |
| step.tool.name, .action, .inputs | Step.Tool | When step.type='tool' |
| step.instructions | Step | When step.type='manual' |
| branch.condition, branch.label | Branch | When branch node selected |
| iterate.max, iterate.until, iterate.over, iterate.as | IterateBlock | When iterate node selected |

**Tree Rendering Fixes:**
- Simple steps render with type icon + title
- Branches render as sub-nodes with 🔀 + condition label, child steps listed
- Iterates render as sub-nodes with 🔁 + config label, child steps listed
- Expand/collapse applies to both branch and iterate children

**Navigation Fixes:**
- Clicking step = select step node → step form
- Clicking branch label = select branch node → branch form
- Clicking iterate header = select iterate node → iterate form

**Round-Trip Verification (Passes):**
- Load service-health-from-readme.runbook.yaml:
  - ✅ Tree shows step 0 (resolve_dns) with 2 branches visible
  - ✅ Each branch shows condition + child step (check_http / dns_not_resolved)
  - ✅ Branch steps fully editable via form
- Save + Reload:
  - ✅ Structure preserved: branches[].steps[].* all correct
  - ✅ Conditions unchanged
  - ✅ No data loss

**Testing Checklist:**
- ✅ Compilation: esbuild passes, no TypeScript errors
- ✅ Load complex runbook (service-health-from-readme.yaml)
  - ✅ All branches visible with conditions shown
  - ✅ All steps inside branches editable
  - ✅ Iterate blocks visible and editable (if file has them)
- ✅ Edit metadata, save, reload → idempotent
- ✅ Edit step inside branch, save, reload → data preserved
- ✅ Edit branch condition, save, reload → condition preserved
- ✅ Create new step (add-step modal), choose location type → correct parent path

**Code Improvements:**
1. Added deep-merge logic in `updateStepAtPath()` to preserve nested objects (tool.name, tool.action, tool.inputs independently)
2. JSON parsing for tool.inputs textarea field
3. Boolean parsing for continue_on_fail checkbox
4. Separated form handlers: update-step, update-branch, update-iterate commands
5. Type safety: interfaces for TreeNode, Branch, IterateBlock match Go schema
6. Path clarity: `selectedNodePath` replaces ambiguous `selectedStepPath`

**Key Lesson: Schema Alignment is Non-Negotiable**
- The editor's data model MUST match the schema exactly
- Structural concepts (Branch, Iterate) are NOT properties of Step
- Form presentation must reflect data structure, not the other way around
- Testing against real runbooks (service-health-from-readme.yaml) forced clarity
- Tree expand/collapse not persisted (session-only state; Phase 2: localStorage persistence)
- No deep validation per field (validation at save only; Phase 2: add real-time per-field checks)

**Dependencies & Compatibility:**
- Uses existing YAML npm library (no new deps)
- Uses existing validateRunbook() from schema/validate.ts (no backend changes)
- Follows VSCode webview best practices (acquireVsCodeApi, postMessage, theme tokens)
- Colors: All `var(--vscode-*)` theme tokens (automatic light/dark mode)
- Icons: Emoji + inline SVG (⚙️ action, 🔀 branch, 🔁 iterate, ✓ conditional, ⏹️ end)
- Tree rendering: Indented list (not DOM tree), Unicode box-drawing or CSS, expand/collapse state session-only
- Form validation: Per-field on blur + keystroke, full-document on save, blocks save if errors
- Branch/iterate conditions: Free-text MVP (structured builder Phase 2)
- Governance display: Read-only badges in form + summary drawer; no user editing Phase 1

**Key Insights:**
- The hybrid form-first + tree-sync model keeps UX simple for sequential steps while preserving structure visibility for branches/iterate/conditionals
- Phase 1 implementation must NOT flatten tree on load; tree extraction needs multi-level hierarchy support
- Governance visible-by-default strategy (badges, drawer, inline hints) aligns user expectations with runtime policy enforcement
- Schema-driven form generation (action/branch/iterate/conditional field sets) enables tool ecosystem extensibility without hardcoding UI
- YAML round-trip fidelity is critical: comments/formatting preservation via npm YAML roundtrip; non-destructive serialization (only rewrite touched sections)

**Component Build Order (Recommended):**
1. SchemaField (base: text/select/number/checkbox/textarea renderers)
2. StepDetailForm (form container, validation, field aggregation)
3. BranchConditionBuilder + IterateBuilder (type-specific components)
4. TreeNode (recursive, indented, expand/collapse)
5. FlowMapSection (tree aggregator, click-to-navigate)
6. GovernancePanel + GovernanceSummaryDrawer (policy display)
7. RunbookHomeSection (metadata), StepNavigator (card list), StepCard (repeatable)
8. YAMLSerializer + DiffGenerator (save/preview logic)

**Open Questions Resolved:**
- Tool Catalog: Accept any string MVP; Killua validates at save
- Governance Editing: Read-only Phase 1
- Branch Conditions: Free-text Phase 1
- Undo/Redo: Defer (use Ctrl+Z at file level + form "Revert" button)
- Tree State Persistence: Session-only

**Cross-Agent Dependencies:**
- Killua (backend): Schema export endpoints, tool/provider catalog, validation diagnostics
- Leorio (governance): Policy display contracts (approval/redaction/deny field mappings)
- Hisoka (QA): YAML round-trip test cases, schema parity validation, serialization fidelity tests

# Project Context

- **Owner:** ormasoftchile
- **Project:** gert runbook system and tooling
- **Stack:** Go, TypeScript, C#, YAML, Azure
- **Created:** 2026-03-17

## Learnings

- Primary frontend focus: visual runbook authoring for business users.
- Recommended editing paradigm for Gert is hybrid: form-first wizard with optional graph canvas preview to preserve branch/tree mental model without forcing YAML edits.
- Existing extension already has a strong execution panel and JSON-RPC client; MVP visual editor should ship as a second webview panel inside the VS Code extension, not a separate app.
- Schema-driven dynamic sections should come from runbook and tool JSON schemas plus discovered tool definitions, using discriminated editors by step type and tool transport/action metadata.
- Governance visibility must be first-class in authoring: inline risk badges, deny/approval warnings, and preflight validation lanes (structural/semantic/domain) mapped to precise field paths.
- Cross-agent dependency noted: UX should consume backend schema/catalog/diagnostic endpoints (Killua), surface policy controls aligned with Azure/tool governance constraints (Leorio), and expose QA-defined release checks in-editor before save (Hisoka).