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
**What:** Rebuilt runbookEditorPanel.ts with tree-based architecture preserving branches, iterates, conditionals. 2-column dual-panel layout with path-based navigation and round-trip safe serialization.
**Why:** Phase 1 flattened structures; tree-based model aligns with Go schema and preserves all runbook structure.

### 2026-03-18: UI specification for runbook visual editor
**By:** Kurapika (Frontend Engineer)
**What:** Defined 2-column adaptive grid layout (60% form / 40% tree), header toolbar with validation badge, governance summary, step navigator, and responsive breakpoint behavior.
**Why:** Provides concrete interaction design spec for the editor implementation.

### 2026-03-18: Add-step handler bugs fixed
**By:** Kurapika (Frontend Engineer)
**What:** Fixed 3 critical bugs: iterate node detection in getSelectionType(), double-appended 'steps' in branch path construction, missing iterate path prefix in add-step handler.
**Why:** Bugs prevented step insertion into branch and iterate nodes.

### 2026-03-18: Schema alignment fix applied — all Hisoka blockers resolved
**By:** Kurapika (Frontend Engineer)
**What:** All 6 Hisoka blockers fixed. Data model corrected (TreeNode with optional step/iterate/branches), form rendering is context-aware, selectedStepPath renamed to selectedNodePath.
**Why:** Resolves structural data loss vectors identified in QA review.

### 2026-03-18: Workflow graph view POC implemented
**By:** Kurapika (Frontend Engineer)
**What:** Implemented toggleable SVG workflow graph visualization in RunbookPanel. New treeToGraph.ts module with deterministic hierarchical layout. Nodes typed (step, condition, join, iterate, start, end), edges classified (sequential, conditional, back-edge), execution path highlighted green.
**Why:** Delivers the confirmed workflow-first visual paradigm for the execution viewer.

### 2026-03-18T16:32:00Z: MVP editor rejection — 6 critical blockers (SUPERSEDED)
**By:** Hisoka (QA Reviewer)
**What:** Rejected MVP due to 6 blockers: invisible iterate blocks, step types include structural types, branch condition maps to wrong object, unmapped fields, unsafe save, missing validation.
**Why:** Structural data loss vectors in TypeScript UI model vs Go schema.
**Status:** SUPERSEDED by Hisoka's subsequent approval after all blockers were fixed.

### 2026-03-18T21:15:00Z: MVP editor approved — all blockers fixed
**By:** Hisoka (QA Reviewer)
**What:** Approved MVP for manual testing. All 6 blockers verified fixed: iterate blocks visible, step types correct, branch conditions mapped to Branch object, form rendering context-aware, YAML round-trip safe.
**Why:** Editor data model now correctly aligns with Go schema.

## Governance

- All meaningful changes require team consensus
- Document architectural decisions here
- Keep history focused on work, decisions focused on direction
