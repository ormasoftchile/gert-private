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

### 2026-03-18: Visualization adapts to host environment

**By:** Cristián Ormazábal Ortega (via Copilot)  
**Status:** Directive  

**What:** Run visualizations (tree view, workflow graph) must be host-environment-dependent. VS Code and web can support full graph/SVG rendering with multiple view modes. A TUI version must restrict to tree-based layout only. The visualization layer should be designed with an adapter pattern — not hardcoded to a single rendering surface.

**Why:** gert targets multiple surfaces (VS Code extension, web, TUI). The rendering engine needs a clean abstraction boundary so each host picks the best visualization it can support.

---

### 2026-03-18: Authoring editor graph integration (Phase B)

**By:** Gon (Lead Architect)  
**Status:** Implemented  

**What:**
1. Graph as view, tree as truth — SVG graph derived from `TreeNode[]` via `treeToWorkflow()`. All mutations operate on tree model.
2. stepId-based click mapping via `buildStepIdToPathMap()`, decoupled from treeToGraph internals.
3. Selection type determined by node content, not path shape.
4. Structural coloring (step-type-based) in authoring context, not execution coloring.

**Why:** Branches and iterate blocks must survive edit→save→reload. Graph and form track same `selectedNodePath`. Kurapika can enhance `treeToGraph.ts` independently.

**Risks:** Shallow branch traversal may show incomplete graphs for complex runbooks. Iterate nodes without `id` fields won't be clickable (acceptable for MVP).

---

### 2026-03-18: Test suite strategy for graph + editor contracts

**By:** Hisoka (QA Reviewer)  
**Status:** Implemented  

**What:** Two automated test suites (71 total tests, all passing): `treeToWorkflow()` transformation contract (37 tests) and editor YAML round-trip fidelity contract (34 tests). Implementation-independent — tests inputs/outputs, not internals.

**Why:** Kurapika and Gon both modifying graph/editor code simultaneously. Contract tests prevent invisible regressions.

**Constraints:** Tests must not break on treeToGraph or editor panel internal refactors. Jest with ts-jest.

**Risks accepted:** No webview integration tests this sprint. Inline fixtures only. Empty branch arm orphaned path flagged for Kurapika.

---

### 2026-03-18: Execution path event design

**By:** Killua (Backend Engineer)  
**Status:** Implemented  

**What:** Two new additive JSON-RPC events:
1. `event/branchResolved` — per branch condition evaluation: `{parentStepId, branchIndex, condition, taken}`.
2. `event/iteratePassEnd` — per iterate pass completion: `{iterateStepId, pass, max, converged}` with synthetic `iterate-<stepIdx>` IDs.

**Why:** Frontend needs per-branch and per-pass granularity for graph annotation. Synthetic IDs avoid schema changes. Zero-cost when no UI listener active. Purely additive — no engine behavior changes.

---

### 2026-03-18: Execution viewer graph — production layout & rendering

**By:** Kurapika (Frontend Engineer)  
**Status:** Implemented  

**What:** Production-quality top-to-bottom recursive bounding-box layout replacing POC left-to-right BFS. Themed SVG rendering with VS Code CSS custom properties, distinct node shapes per type, zero runtime dependencies.

**Why:** Users running incident response runbooks need at-a-glance flow understanding. Full rewrite of `treeToGraph.ts` layout and `runbookPanel.ts` `renderGraphSvg()`.

---

### 2026-03-18: User directive — no temp files outside workspace

**By:** Cristián Ormazábal Ortega (via Copilot)  
**Status:** Directive  

**What:** Never create temporary files outside the working directory. All temp/scratch files must stay within the project workspace folder.

**Why:** User directive — keeps the filesystem clean and all artifacts discoverable within the repo.

---

### 2026-03-18: Editor backend APIs are stateless read-only endpoints

**By:** Killua (Backend Engineer)  
**Status:** Implemented  

**What:** Added six JSON-RPC endpoints (`tools/list`, `tools/get`, `exec/dryRun`, `schema/stepFields`, `schema/toolArgs`) to the serve layer. All are stateless and read-only — they do not create engines, modify server state, or touch execution sessions.

**Why:** The visual editor needs tool catalog data, argument schemas, and dry-run validation to drive dynamic form rendering and preflight checks. These must come from the Go backend (source of truth for schema, tool parsing, and governance rules) rather than being hardcoded in TypeScript.

**Key constraints:**
- Tool discovery uses existing `schema.DiscoverProject` + `project.ResolveToolRef` — no new resolution logic.
- `exec/dryRun` validates with the full 3-phase pipeline but never instantiates an engine or executor.
- `schema/stepFields` returns step-type field definitions derived from Go struct tags — if schema structs change, this endpoint must be updated.
- All endpoints accept optional `cwd` to support multi-root workspace scenarios.

**Impact:** Kurapika can replace hardcoded field lists with `schema/stepFields` and `schema/toolArgs` calls. Gon's editor architecture can use `tools/list` for the tool picker and `exec/dryRun` for save-time validation. Hisoka can gate on `exec/dryRun` governance warnings for reviewer checks.

---

### 2026-03-18: Graph rendering — visual readability defaults

**By:** Kurapika (Frontend Engineer)  
**Status:** Applied  

**What:** Node width 240px (was 180) with 28-char truncation. Join nodes invisible (4px, 0.4 opacity). Minimum untaken branch opacity 0.55. Bezier convergence adds `dx * 0.3` to control points.

**Why:** Live test on `service-health-branching` revealed overlapping labels, confusing join nodes, unreadable dim branches, and crossing edges.

---

### 2026-03-18: Editor graph click fix, vizAdapter abstraction, comment-preserving save

**By:** Gon (Lead Architect)  
**Status:** Implemented  

**What:**
1. Fixed iterate graph nodes not responding to clicks — missing `.id` on in-editor iterate blocks broke stepId→path mapping. Fix assigns IDs on creation and backfills synthetic IDs in `buildStepIdToPathMap()`.
2. Created `vizAdapter.ts` with `VizAdapter` interface, `VizHost` type, and `createVizAdapter()` factory. Graph adapter wraps `treeToWorkflow` + SVG. TUI adapter renders indented text tree.
3. Save handler now uses `YAML.parseDocument()` for comment-preserving AST round-trips via `doc.set()`.

**Why:** Click→form is the primary navigation contract. Host-adaptive viz fulfills the user directive for multi-surface rendering. Comment preservation fulfills Killua's compatibility decision (2026-03-17) for non-destructive normalization.

---

### 2026-03-18: Governance display contract via JSON-RPC

**By:** Leorio (Integrations Engineer)  
**Status:** Implemented  

**What:** `governance/evaluate` read-only JSON-RPC endpoint in `ext/serve`. Takes runbook path, evaluates governance policy per step without executing, returns structured assessment. TypeScript interfaces in `vscode/src/serve/governance.ts`. Client method `governanceEvaluate()` wired in `client.ts`.

**Why:** Visual editor needs inline governance info (risk badges, approval gates, redaction indicators, policy violations) without running the runbook. Decoupled from execution per "backend read APIs for editor metadata" decision (Killua, 2026-03-17).

**Key choices:** Walk-all-nodes (badges every visible node), merged governance sources (single call), no execution side effects (safe on every open/save), arrays default to `[]` not `null`.

---

## Governance

- All meaningful changes require team consensus
- Document architectural decisions here
- Keep history focused on work, decisions focused on direction
