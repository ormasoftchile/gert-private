# Project Context

- **Owner:** ormasoftchile
- **Project:** gert runbook system and tooling
- **Stack:** Go, TypeScript, C#, YAML, Azure
- **Created:** 2026-03-17

## Learnings

- Primary backend focus: runbook engine, schema handling, and extensible providers/tools.
- Visual editor can stay YAML-canonical by reusing strict/flexible loaders in pkg/schema (Load/decodeRunbookFlexible) and project-aware compat resolvers (ResolveToolPathCompat, ResolveRunbookPathCompat).
- Existing serve JSON-RPC already exposes execution/session surfaces; adding read-only metadata endpoints there is the lowest-risk path to drive UI generation without touching engine execution semantics.
- Tool/provider extensibility is already alias- and capability-driven (ToolDefinition, ProviderDefinition->ToolDefinition, ToolManager); editor contracts should consume these registries instead of transport-specific hardcoding.
- Cross-agent contract expectation: backend metadata/diagnostic APIs should directly power Kurapika's form model, preserve Gon's YAML-canonical MVP boundary, and produce machine-checkable outputs for Hisoka's reviewer gates.
- **Editor MVP Compatibility Review (2026-03-17)**: Reviewed runbookEditorPanel.ts for schema/runtime compatibility. Found critical issues: simple linear step extraction + rebuild path destroys branches/iterates, ignores complex step types (manual/assert), discards all tool configuration, loses YAML formatting/comments. Implemented safety gates: detect non-linear structures at render time, block tree editing with warnings when unsafe, preserve metadata-only editing regardless of complexity. Tree corruption prevented by structural validation on save.
- Runbook schema constraints: top-level tree nodes enforce step XOR iterate (exclusive), branches require conditions referencing previous outputs, iterate blocks have convergence (max+until) vs list (over+as) modes, steps support 9+ types. Editor MVP scope: safe only for simple linear tool-step trees. Complex/branched/iterated runbooks gracefully degrade to metadata editing with clear UI warnings.
- Tree safety pattern: check structure at render-time (not save-time), fail early with visible blockers, always allow metadata editing, validate rebuild hasn't corrupted required fields before fs.write().

## Schema Clarification (2026-03-18)

**Task**: Clarify TreeNode structure misalignment between TypeScript form and Go schema for Hisoka's MVP rejection fix.

**Findings**:
- **TreeNode structure** (Go authoritative): `{step?, iterate?, branches?}` — exactly one of step/iterate must be non-zero, branches are optional on any node.
- **Branch vs Step**: Branches are a **structural container** on TreeNode (not a step type). Step types are: tool, manual, assert, end, extension, cli, invoke. ~~branch~~ and ~~parallel~~ are artifacts—do NOT use.
- **Field locations**:
  - Branch conditions: on `Branch.condition` (required).
  - Iterate config (max, until, over, as): on `IterateBlock`, NOT on steps inside.
  - Tool config (name, action, args, capture): on `Step.Tool` (ToolStepConfig), only when type=tool.
  - Governance (rules, redaction, deny env vars): on global `Meta.Governance`, not per-step (except Step.Approvals for manual).
- **Navigation paths**: Use `[n, 'field', m, 'subfield', ...]` notation with 0-based indices and key access.
- **Read-only fields**: APIVersion, (mostly) Step.ID, Meta.Source. Governance rules shadow editable fields at runtime.

**Outcome**: Produced SCHEMA_CLARIFICATION.md (authoritative Go-backed doc) with TreeNode spec, complete YAML examples (branches + iterates), path notation, field locations, and validation rules. Ready for Kurapika's form redesign.

## Execution Path Events (2026-03-18)

**Task**: Add `event/branchResolved` and `event/iteratePassEnd` to the Go serve layer for frontend execution viewer annotation.

**Changes**:
- **`event/branchResolved`** — emitted once per branch condition evaluation at all 3 branch evaluation sites in `ext/serve/pkg/serve/serve.go`:
  1. `handleTreeNext` pending manual phase2 (line ~928)
  2. `executeTreeStep` branch-type routing nodes (line ~1330)
  3. `executeTreeStep` regular step post-outcome branch evaluation (line ~1474)
  - Payload: `{parentStepId, branchIndex, condition, taken}`
  - Emits for each branch evaluated (stops after first `taken=true`, matching existing `break` semantics).

- **`event/iteratePassEnd`** — emitted at the end of every iterate pass (before convergence/advance decision) at both watchpoint handlers:
  1. Convergence-mode watchpoint handler (line ~975)
  2. List-mode over-watchpoint handler (line ~1019)
  - Payload: `{iterateStepId, pass (1-based), max, converged}`
  - Uses synthetic `nodeID` (format `iterate-<stepIdx>`) threaded through watchpoint structs for correlation.

- **Watchpoint struct changes**: Added `nodeID string` field to `iterateWatchpoint` and `iterateOverWatchpoint`. Updated `pushIteratePass` and `pushIterateOverPass` signatures to accept `nodeID`.

- **VS Code extension wiring**:
  - Added `BranchResolvedEvent` and `IteratePassEndEvent` interfaces to `vscode/src/serve/client.ts`.
  - Added `event/branchResolved` and `event/iteratePassEnd` case handlers to `runbookPanel.ts` `handleEvent` switch, following existing event dispatch pattern (console.log + updateWebview). No SVG/render changes.

**Patterns followed**: Existing `sendEvent` notification pattern (zero-cost JSON-RPC when no listener). All events additive — no changes to existing event shapes, engine behavior, or execution semantics.

## Editor Backend APIs: Tool Catalog, Dry-Run, Schema Fields (2026-03-18)

**Task**: Add six new JSON-RPC endpoints to `ext/serve/pkg/serve/serve.go` and wire them into `vscode/src/serve/client.ts` for the visual editor's form generation and preview features.

**Endpoints added (Go)**:

1. **`tools/list`** — Discovers project root via `schema.DiscoverProject`, scans the `tools/` directory for `*.tool.yaml` files, loads each with `schema.LoadToolFile`, returns array of `{name, version, description, actions}` with arg schemas per action. Accepts optional `cwd` param.

2. **`tools/get`** — Takes `{name, cwd?}`, resolves via `project.ResolveToolRef`, returns the full `ToolDefinition` as JSON (apiVersion, meta, transport, governance, actions with full arg/capture/governance detail).

3. **`exec/dryRun`** — Takes same params as `exec/start`. Runs `schema.ValidateFile` for 3-phase validation, flattens tree for step count, collects `tools:` as tool dependencies, scans `meta.governance.rules` for approval/deny rules and `deny_env_vars`. Returns `{valid, errors[], tree, stepCount, toolDeps[], governanceWarnings[]}`. Does NOT create an engine or execute anything.

4. **`schema/stepFields`** — Returns hardcoded field definitions per step type: `tool`, `manual`, `assert`, `end`, `extension`, `cli`, `invoke`. Each includes the relevant YAML fields for that step type. No params required.

5. **`schema/toolArgs`** — Takes `{tool, action, cwd?}`, resolves tool via project, loads definition, returns `{args: {name: {type, required, description?, default?, enum?, redact?}}}` for the specified action.

**TypeScript client wiring**:
- Added interfaces: `ToolArgInfo`, `ToolActionInfo`, `ToolInfo`, `ToolDefinition`, `DryRunResult`.
- Added methods: `toolsList(cwd?)`, `toolsGet(name, cwd?)`, `execDryRun(params)`, `schemaStepFields()`, `schemaToolArgs(tool, action, cwd?)`.

**Compile verification**: Both `go build` (exit 0) and `npm run compile` + `tsc --noEmit` (exit 0) pass clean.

**Patterns followed**: All endpoints are read-only/stateless — they do not create engines, modify server state, or call `saveSession()`. Tool discovery reuses existing `schema.DiscoverProject + FallbackProject` and `project.ResolveToolRef` paths. Same dispatch/switch pattern as existing methods.
