# Decision: Phase 2 Architectural Ruling — VS Code Authenticated MCP Runtime Binding

**Date:** 2026-08-17
**Author:** Barbara (Lead/Architect)
**Status:** ACCEPT WITH MODIFICATIONS

## Ruling

Phase 2 is accepted with three modifications:

1. **Output contract enforcement for non-substituted tools is prerequisite core work.** The consumer's mutation control ("remove the output contract parity check and prove drift is caught") assumes a runtime check that does not exist. This is the fifth instance of the declared-but-unenforced pattern class. Must land before the bridge is load-bearing.

2. **Three extension prerequisites are blocking:** reproducibility remediation, `engines.vscode` raise to ≥1.90, and extension-host test harness (`@vscode/test-electron` or equivalent).

3. **The `serve` all-interfaces/no-auth security issue must be fixed before or with Phase 2**, not after.

## Ownership

- **Core:** `vscode-mcp` transport chain, loopback bridge server, output contract enforcement, request-ID dedup, bridge versioning, credential sweep extension, `serve` security fix.
- **Extension:** Reproducibility remediation, `engines.vscode` raise, test harness, bridge client, tool resolution via `vscode.lm.tools`, result normalization, authorization-unavailable detection, disconnect signaling.
- **Consumer:** Nothing new. Unchanged runbook. Manual acceptance in their environment.

## Executor-Kind Constraint

The `vscode-mcp` binding MUST remain within the existing `"tool"` executor kind. Recommended structural enforcement: extract classification-injection + approval-gate logic into a named function whose signature requires a non-nil `*ToolClassification`, making governance bypass a compile-time error.

## Estimate

10–14 days parallelized after prerequisites (3–4 days serial in extension repo).

## References

- Full evaluation: `gert-core-phase2-evaluation.md`
- Phase 1B completion: `gert-core-phase1b-completion.md`
- Reachability gate: `TestCLI_ReachabilityGate`
