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
