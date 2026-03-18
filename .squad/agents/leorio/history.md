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