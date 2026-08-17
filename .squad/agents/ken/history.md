# ken

Current role: Implementation engineer.

Session: Dynamic Runbook Includes (2026-08-15)
- Feature complete and approved
- Ready for merge

Session: MCP-HTTP Recon (2026-08-16)
- Survey complete; report filed to `.squad/decisions/inbox/ken-mcp-http-recon.md`
- No production code written

Session: MCP-HTTP Stream A — Schema + Validation (2026-08-16)
- Complete; report filed to `.squad/decisions/inbox/ken-mcp-http-streama.md`
- AuthConfig (Provider/Scope/AllowedHosts string fields), TransportMCPHTTP constant, URL field on TransportConfig
- B-32 revised: ErrMCPW001/MCP-W class removed; reclassified as ErrMCP012/MCP-012 (fatal); ErrMCP013 added (redirect blocked); MCP-W class fully purged; auth_gate.go u.Host→u.Hostname() drift fixed
- Host matching: exact, case-insensitive, parsed hostname via net/url — no substring/prefix/wildcard
- ValidateTransportConfig enforces all Barbara §1.3/B-22/B-14/B-32 rules with actionable error messages
- MCP-001..013 sentinels in errkit; all classes clean
- B-31 documentation: TransportConfig/AuthConfig Go doc comments with replay-risk rationale; 06-tool-runtime.tex mcp-http subsections (transport fields, protocol version split, redirects, auth detail); tools/icm.tool.yaml example
- 22 tests pass; go build clean

Detailed history: .squad/agents/ken/history-archive.md

## 2026-08-17 — Slice 2: Approval Enforcement

**Feature:** Close the live safety gap where GovernanceEvaluator was never wired in production — governance pre-flight was dead for all non-substitution tool invocations.

**Outcome:** ✅ COMPLETE — All four counterparty acceptance criteria met, all tests pass.

## Learnings

- **Wiring vs. plan-time construction**: `GovernanceEvaluator` in `EngineConfig` is built at wiring time, before any runbook is known. The right design is to build a *per-run* evaluator in `engine.Start()`/`Resume()` from `plan.GovernanceSource`, stored in `runHandle`. The wired evaluator is a non-nil fallback; the per-run one is the live enforcement.
- **Chokepoint selection**: The engine pre-flight (`executeStep`) is the right single chokepoint for approval enforcement — it fires for ALL step kinds before executor dispatch, covering stdio-MCP, HTTP-MCP, and process transports without touching any transport layer.
- **Monotone-OR composition**: The "runbook-level false cannot suppress tool-level true" invariant must be explicitly enforced. Adding `ToolRequiresApproval bool` to `governance.StepInfo` and OR-ing it in the evaluator achieves this without breaking the existing `composeGovernance` path used by substitution.
- **Don's *bool migration**: `ToolGovernance.RequiresApproval` was already changed to `*bool` by Don. The nil-guard in `EffectiveGovernanceFromTool` needed to be accounted for; reading `*toolDef.Governance.RequiresApproval` in the engine required explicit nil checks on both the `Governance` pointer and the `RequiresApproval` pointer.
- **`failRun` propagates as error from `Next()`**: When the approval gate denies, `failRun` causes `Next()` to return a non-EOF error. Test helpers that fatalf on any non-EOF error break; instead they must handle this as a valid termination path.

## 2026-08-17 — Runtime Portability Slice 2 (Commit c810b96)

**Status:** SHIPPED & REVIEWED ✅

**Outcome:** Closed live safety gap. Approval gate now wired in production paths; governance pre-flight active for all transports.

**Changes:** GovernanceEvaluator unconditionally wired (wire.go, run.go, sub-engine). Per-run PolicyEvaluator built in Start()/Resume() from plan.GovernanceSource. StepInfo.ToolRequiresApproval added; evaluator ORs with policy-level requireApproval (monotone composition preserved). Chokepoint: executeStep pre-dispatch, before executor selection.

**Test Coverage:** Six named acceptance tests; engine.Start() → Next() cycles verify all four counterparty criteria. Full suite passes; no regressions.

**Team Note:** Passed Barbara's review gate. Expected gap (unspecified-classification gate behavior) deferred to ProfileApprovalGate slice as designed.
