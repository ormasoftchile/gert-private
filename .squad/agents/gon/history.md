# Project Context

- **Owner:** ormasoftchile
- **Project:** gert runbook system and tooling
- **Stack:** Go, TypeScript, C#, YAML, Azure
- **Created:** 2026-03-17

## Learnings

- Team seeded to explore a visual runbook editor for business users.
- Existing VS Code extension already has a mature run webview surface and JSON-RPC client, making a side-by-side visual editor in the same extension the fastest MVP path.
- Extensibility pressure points are dynamic tool/provider resolution and project-aware package references, so the editor model should be AST + schema metadata, not static forms.
- Recommendation drafted: hybrid visual editor with YAML source of truth, schema-driven inspector, and plugin-like step cards resolved from tool/provider definitions.
- Cross-agent alignment: MVP should be phased as contracts-first (Killua), form-first UX (Kurapika), governance-visible Azure/tool abstraction (Leorio), and QA release gates on round-trip + v0/v1 parity (Hisoka).

## 2026-03-17 Design Direction Session

### Problem Identified
- Kurapika's Phase 1 MVP flattens branch/conditional/iterate structures in tree model, causing data loss when editing non-trivial runbooks.
- Root cause: UX designed after implementation started, not before.
- Critical insight: form-first is correct, but **form must synchronize with a visible hierarchical tree model**, not flatten it.

### Design Solution: Hybrid Form-First + Tree Map
- **Left panel:** Step-by-step form editor (Runbook Home, Step Detail, Step Navigator)
- **Right panel:** Synchronized read-only/navigation tree showing full structure (branches, iterates, conditionals with indentation)
- **Interactions:** Click tree node → jump to form; form edits reflect in tree in real time
- **Key win:** Preserves data fidelity while keeping form-first UX for non-technical users

### MVP Scope Clarified
- **In:** Form authoring (seq steps, branches, iterate), tree visualization, governance visibility, valid YAML serialization
- **Out:** Tree-based drag/drop editing, governance policy editing, tool catalog browser, undo/history, templates
- **Rationale:** Business users need structure visibility; engineers need YAML transparency. Tree map is the "check" on form operations.

### UX Principles Established
1. Structure visibility over simplicity — always show tree, never flatten
2. Form as progress, not endpoint — governance/validation at save gate
3. Respect YAML as truth — form output must be round-trippable and pass existing validation

### Key Design Decisions Rationalized
- **Form-first vs. tree-first:** Form-first + synchronized tree (rejects pure tree editor as error-prone for common case)
- **Flat vs. hierarchical:** Hierarchical with indentation; form presents one level at a time (solves Phase 1 flattening bug)
- **Edit-in-place vs. modal:** Non-modal form on right side keeps structure visible
- **Read-only vs. editable governance:** Read-only badges + governance panel Phase 1 (policy is runtime/admin concern, not editor concern)

### Next Work
- Kurapika: align on layout, tree rendering tech (SVG/canvas), interaction details
- Killua: confirm schema introspection for form generation
- Leorio: confirm governance metadata accessibility for UI display
- Hisoka: define round-trip test harness (form → YAML → form → YAML idempotent)
- Gon (next): prototype lo-fi mockup to validate layout before webview rebuild

## Learnings

### 2026-03-18 Workflow-First Visual Paradigm Assessment

- **The tree IS the graph.** `TreeNode { step, branches, iterate }` maps to graph topology without a new data model. Edges are sequential (array order), conditional (branches), and loop (iterate back-edge). A separate edge list is not needed and would create YAML round-trip risk.

- **Tree→graph transformation belongs client-side.** The full topology is already sent to the extension in `exec/start → tree`. The `pkg/diagram/diagram.go` Mermaid generator is the reference implementation to port to `treeToGraph.ts`. Server-side graph API would couple view topology to the serve release cycle.

- **Go serve changes are additive and small.** Two events: `event/branchResolved` (which branch was taken) and `event/iteratePassEnd` (loop pass state). Both are ~10-line additions to `ext/serve/pkg/serve/serve.go`. Not blockers for the execution viewer PoC.

- **`rebuildTree()` in RunbookEditorPanel is a P0 bug** before any graph authoring. It flattens branches and iterate blocks, causing data loss. Fix precedes Phase B work.

- **Sequence: execution viewer first.** Read-only → no round-trip risk → validates `treeToGraph` against real runbooks → proves graph library choice. Phase B (authoring) inherits the proven graph renderer.

- **Key constraint for graph authoring:** restrict editing operations to add-sequential / add-branch / add-iterate. No free-connect. Any topology not expressible in the YAML tree model must be blocked in the UI.

- **`pkg/diagram/diagram.go` is a gold-standard reference** — already generates correct Mermaid from the same tree structure, including branches, iterate, and conditions. `treeToGraph.ts` should mirror its traversal logic.