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
**What:** Phase 2 will extend the form editor to support branches and iterate nesting (tree-based editing). New TreeNode structure for internal representation.
**Why:** Phase 1 was flat/linear only. Phase 2 adds full tree editing capability, preserving branch/conditional data.

### 2026-04-15: No cross-repo dependencies in design/gert-v2
**By:** Gon (Lead Architect)
**What:** `/design/gert-v2` repo will NOT depend on `/gert/v2` or other cross-repo paths during build. All references are doc-only.
**Why:** Design repo is independent, artifact-focused; avoids tool lock-in and merge/deployment complexity during rapid iteration.

### 2026-04-18: Bug fix: r10 YAML shell quoting
**By:** John (Schema)
**What:** r10 fixture had shell-escaped quoting (`'"'"'` instead of `'`) in approvers list; fixed to plain single quotes.
**Why:** YAML parser was failing on shell syntax; real runbooks will not use shell quoting.

### 2026-04-19T21:32:56Z: Enhancement to r02 (Use Iterate Node)

**By:** Barbara (Integrations Specialist)  
**Status:** PROPOSED  
**Priority:** LOW (defer to post-Phase 5)

**Problem:** r02 prose describes 10-minute monitoring loop with 60 iterations but uses 17 sequential `cli` steps (workaround). True `iterate` step would align schema with prose.

**Recommendation:** Defer to post-Phase 5 (can implement after r11 written and iterate executor ready)

**Rationale:**
- Tests iterate semantics in real-world canary deployment scenario
- Reduces r02 verbosity (17 steps → 1 iterate with 3 nested)
- Aligns schema with prose intent

**Effort:** ~30 lines YAML refactor  
**Owner:** John (Schema) to refactor r02  
**Timing:** After r11 is written and iterate executor implemented (Phase 5)

---

## 2026-04-19T22:27:31Z: Phase 1 Parser Implementation Complete

**By:** Brian (Go Programmer)  
**Status:** APPROVED  
**Priority:** CRITICAL (blocking Phase 1 complete)

**Decision:** Phase 1 parser implementation (v2/internal/parser/) is production-ready.

**Deliverables:**
- 6 files: parser.go, unmarshal.go, validate_structural.go, validate_semantic.go, errors.go, parser_test.go
- 14/14 tests passing
- go build clean (no errors/warnings)
- Two-phase YAML decode strategy for schema.Step inline spec conflicts
- All 10 runbook fixtures (r01–r10) parse successfully
- JSON Schema validation via github.com/santhosh-tekuri/jsonschema/v6
- Semantic validation for runbook integrity

**Key Design:**
- Raw `yaml.Node` capture enables selective field decoding (avoids "duplicated key" panic)
- Never calls `yaml.Node.Decode()` on types transitively containing `schema.Step` or `schema.FlowNode`
- Type-discriminator-based spec unmarshaling (e.g., switch on step.type, unmarshal only matching spec)

**Schema Updates:**
- runbook.schema.json: 11 relaxations for fixture compatibility (GovernanceConfig, ToolRef, CollectorFieldType, Assertion, etc.)
- Go schema: 4 type updates (GovernanceRule, RedactRule, ToolRef.Source/Actions, ToolInvocation.Args any)
- Added StepTypeExtension = "extension"

**Rationale:**
- Two-phase decode prevents the yaml.v3 "duplicated key" panic that blocked v1 design
- Fixture compatibility ensures parser handles real runbooks reliably
- Clear error context aids debugging and UX

**Impact:**
- Parser can now be integrated into serve APIs (serves backend for CLI and editor)
- Unblocks Phase 2 (engine integration, serve endpoint for validation diagnostics)

---

## 2026-04-19T22:27:31Z: StepSpec Interface Ready

**By:** Ken (Architect)  
**Status:** APPROVED  
**Priority:** CRITICAL (blocking Phase 1 complete)

**Decision:** StepSpec interface and ResolvedStep.Spec type-safety integration is complete.

**Deliverables:**
- StepSpec interface with StepKind() method (v2/pkg/engine/stepspec.go)
- ResolvedStep.Spec: any → StepSpec (v2/pkg/engine/run.go)
- StepKind() implemented on all 14 step types (v2/pkg/schema/steps.go)

**Step Types Implementing StepKind():**
- CLISpec, ToolCallSpec, IncludeSpec, ChoiceSpec, DecisionSpec, CollectorSpec
- BranchSpec, IterateNode, ParallelNode
- ApproveSpec, AssertSpec, CompensateSpec, WaitForEventSpec, EndSpec

**Rationale:**
- Compile-time verification ensures all step types implement StepSpec
- Engine dispatch code now type-safe: `step.Spec.StepKind()` vs. reflection
- Adding new step types will fail at compile time if StepKind() missing

**Build Status:**
- `go build ./...` ✓
- `go vet ./...` ✓

**Impact:**
- Engine can now use StepSpec for type-safe dispatch logic
- Unblocks Phase 2 (ResolvedStep serialization and executor implementation)
