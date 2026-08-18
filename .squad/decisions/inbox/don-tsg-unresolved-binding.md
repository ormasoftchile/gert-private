# Decision: TSG Unresolved Provider Binding (VSCODE-003)

**Date:** 2026-08-18  
**Author:** Don (Core Dev, Gert Core)  
**Requested by:** Cristiano  
**Status:** Implemented — awaiting merge gate

---

## Context

SQL Live-Site Operations supplied the TSG binding's LOGICAL contract (six
args: `incident_id`, `title`, `service`, `environment`, `logical_server`,
`database`) but could NOT supply the PROVIDER contract (VS Code MCP registered
tool name + parameter schema), because the tool was not present in
`vscode.lm.tools` during the live session.

The deliverable had `vscode_tool: tsg-recommendation-recommend` — a guessed
name. This is the same defect class as `icm-get-incident` caught in Rev 2.

---

## Decision

**Do NOT invent the provider tool name.** Mark the binding explicitly unresolved
with a new schema field `provider_unresolved: true` on `TransportConfig`.

**Fail closed at scan time (VSCODE-003).** `ValidateTransportConfig` in
`internal/tool/validate_transport.go` rejects any `vscode-mcp` tool with
`provider_unresolved: true` during `ParseToolFile`, before the tool enters the
registry. This is the earliest achievable fail point in the transport chain.

**Error text is unambiguous:**
> This is NOT a "tool not found" error: nobody has told us the real
> vscode.lm.tools name yet.

This distinguishes the unresolved-contract state from MCP server downtime.

---

## Rationale

**Why scan time, not runtime?**  
The prior vscode_input fail-closed checks (VSCODE-001, VSCODE-002) also fire
at scan time via `ValidateTransportConfig` → `ParseToolFile`. This is
consistent with the established pattern. Runtime would allow the tool into the
registry and would reach the bridge — a worse failure mode.

**Why not just leave vscode_tool absent?**  
An absent `vscode_tool` is valid and currently means pass-through (logical arg
names forwarded to the provider). With an unknown provider schema, pass-through
is a silent-wrong-args hazard. `provider_unresolved: true` makes the intent
explicit and forces the error.

**Why a schema field and not a comment?**  
Comments are not machine-checked. A schema field enforced in the validation
chain cannot be silently ignored, and the reachability gate ensures the check
cannot be removed without a test failing.

---

## What was NOT done

The provider tool name and parameter schema were NOT invented. No placeholder,
plausible guess, or synthetic example appears in any deliverable or test
fixture that could be mistaken for a real registered name.

The reachability probe uses the synthetic name `tsg-unresolved-probe` —
obviously non-real, never in a deliverable.

---

## Resolution path

When SQL Live-Site Operations confirms the provider contract:
1. Set `vscode_tool:` to the real registered name.
2. Add `vscode_input:` mappings (provider key → logical arg).
3. Remove `provider_unresolved: true`.
4. Update `TestDeliverableContracts_TSGRecommendation` to assert ParseToolFile
   SUCCEEDS and check outputs (current test asserts failure — must be inverted).
5. Sync `gert/internal/tool/testdata/` with the deliverable.

---

## Files changed

**gert (branch: feature/tsg-unresolved-provider-binding, commit 1824af7):**
- `pkg/schema/tool.go` — `ProviderUnresolved bool` on TransportConfig
- `internal/tool/validate_transport.go` — VSCODE-003 check
- `internal/tool/testdata/tsg-recommendation.vscode-mcp.tool.yaml` — updated
- `internal/tool/deliverable_contract_test.go` — updated
- `cmd/gert/reachability_registry_test.go` — new entry
- `cmd/gert/reachability_probes_test.go` — new probe

**gert-private (commit b598128):**
- `deliverables/phase2/tsg-recommendation.vscode-mcp.tool.yaml` — updated
