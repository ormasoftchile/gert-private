# Decision: Phase 1 Visual Runbook Editor MVP Implementation

**Date:** 2026-03-17  
**Author:** Kurapika (Frontend Engineer)  
**Status:** Implemented  

## Context

Implemented Phase 1 MVP of visual runbook editor inside VS Code extension per sprint requirements. Goal was to provide a form-based editing experience for non-technical users while maintaining YAML as canonical format and preserving compatibility with existing execution flow.

## Decision

### Architecture Choices

**1. Webview Panel Pattern**
- Implemented as a new webview panel (`gertRunbookEditor`) that launches alongside or in place of active editor
- Reuses existing extension infrastructure (vscode.commands, webview APIs)
- Preserves separation between visual editor and execution engine (RunbookPanel)

**2. YAML as Single Source of Truth**
- Editor loads and parses YAML on open, serializes back on save
- No dual-sync issues or state drift
- Existing validation (`validateRunbook`) runs before save; warns on errors but allows force-save
- Uses `yaml` npm package (already in deps) for parsing/serializing

**3. Simplified Tree Model for MVP**
- Flattens tree data structure for linear step list in UI (extractSteps → rebuildTree)
- Extracts only `step.id`, `step.type`, `step.title` per step
- Rebuilds as simple linear tree: `[{ step: { id, type, title } }]` without branching
- **Limitation:** Does not preserve branch/routing structure in current iteration. Existing branches in loaded YAML will be flattened.

**4. Form-First UI**
- Metadata section: name, kind, description (text inputs)
- Steps section: table of step cards, each with id/type/title inputs
- Add/remove/reorder buttons (↑↓ arrows, remove btn)
- Validation status badge + error list (first 10 errors shown)
- Save/Cancel buttons at bottom

**5. Client-Side Message Passing**
- Webview sends commands via `vscode.postMessage()`: add-step, remove-step, move-step, save, cancel, validate-preview
- Extension listens for messages in `handleWebviewMessage()`
- No round-trip to gert binary; validation is instant (JavaScript-only)

**6. Safety & XSS Prevention**
- All dynamic text escaped via `escapeHtml()` method
- HTML template uses string interpolation, not innerHTML dumps
- Input fields bound to data attributes for event delegation
- VSCode theming variables used for consistent styling

## Tradeoffs

### Accepted Limitations
- **No graph/branch editor yet:** Step list is linear; complex branching must be edited in YAML raw view for now.
- **No tool catalog integration:** Tool definitions not loaded; users type type/id manually.
- **No field-level validation:** Only full-document validation on save.
- **No undo/redo:** Changes rebuild full tree each operation; no transaction history.

### Architectural Debt (Intentional)
- `rebuildTree()` creates generic tree; fine for MVP but would need refinement to preserve branch metadata in future.
- No support for non-step tree nodes (resources, outcomes, conditions) in editor UI; raw YAML still works.

## Validation & Safety

- Extension compiles without errors (esbuild successful)
- Existing Jest test suite confirms schema validation paths are functional
- HTML escaping verified across all dynamic interpolations
- Works with missing/empty runbook sections (initializes defaults)

## Follow-Up Work (Post-MVP)

1. **Branch/routing support:** Enhance tree model to preserve and edit branches UI
2. **Tool catalog:** Populate type/action selectors from discovered tools (Killua integration)
3. **Governance badges:** Mark steps with policy compliance/risk (Leorio integration)
4. **Field validators:** Per-field hints and inline validation before save
5. **Undo/History:** Transaction-based tree mutations for editor sessions
