# Project Context

- **Owner:** ormasoftchile
- **Project:** gert runbook system and tooling
- **Stack:** Go, TypeScript, C#, YAML, Azure
- **Created:** 2026-03-17

## Learnings

- Integrations focus: blend CLI and Azure operations into governed runbook execution.
- Existing runtime already supports transport decoupling for tools (stdio/jsonrpc/mcp) and approval gates at action level (`requires_approval`, `approval_min`), so Azure integration should be modeled as tool/provider contracts instead of a new step type.
- Governance engine enforces command allow/deny, env var blocking, and output redaction centrally, which enables editor controls to map directly to runbook/tool schema fields and runtime enforcement.
- Current MCP implementation supports spawn mode only; connect mode is explicitly not implemented, so external Azure integrations should start with spawnable adapters or JSON-RPC wrappers.
- Cross-agent governance alignment: Azure scenarios should remain within Gon/Killua's existing tool/provider contracts, while Kurapika surfaces approvals/redaction in UX and Hisoka validates governance-safe editing in release gates.

## 2026-03-18: Governance display contracts for visual editor

### What was done
- Added `governance/evaluate` JSON-RPC endpoint to `ext/serve/pkg/serve/serve.go`.
- Endpoint loads a runbook by path, walks all tree nodes (including branches/iterates), evaluates each step's contract against the runbook's governance policy, checks tool-level and action-level governance overrides, and validates CLI commands against allow/deny lists.
- Returns: overall risk level, per-step risk/approval/redaction/warnings, denied env vars, allowed commands, and violation list.
- Created `vscode/src/serve/governance.ts` with TypeScript interfaces (`GovernanceEvaluateResult`, `GovernanceStepResult`, `GovernanceRedactionRule`, `GovernanceRuleSummary`, `RiskLevel`) matching the JSON response shape.
- Wired `governanceEvaluate(runbookPath)` method into `vscode/src/serve/client.ts`.
- Both `go build ./ext/serve/...` and `npm run compile` pass clean.

### Learnings
- `governance.EvaluateContract` takes a `*contract.Contract` and `*schema.GovernancePolicy` — the contract must be built from the step's `StepContract` override, not from the schema `GovernanceContract` type (which is for rule matching).
- Tool definitions can add redaction rules at both tool-governance and action-governance levels; both must be merged into the step result.
- Tree walking must recurse into `node.Branches[].Steps` and `node.Iterate.Steps` to cover all step locations.