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

## Governance

- All meaningful changes require team consensus
- Document architectural decisions here
- Keep history focused on work, decisions focused on direction
