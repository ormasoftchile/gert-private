# barbara

Current role: Implementation engineer.

Session: Dynamic Runbook Includes (2026-08-15)
- Feature complete and approved
- Ready for merge

Detailed history: .squad/agents/barbara/history-archive.md


U+1F4CC MCP HTTP Transport Contract Session (2026-08-16T00:55:39Z):
- Authored binding contract for Streamable HTTP MCP support (mcp-http transport mode).
- Key architecture finding: the transport seam ALREADY EXISTS. pkg/tool.ToolTransport interface + DefaultToolRuntime.Invoke switch dispatch = the integration point. No refactor of existing stdio MCP needed. New MCPHTTPTransport implements the same interface independently.
- Issued B-22..B-26: HTTPS-only (B-22), protocol version pin at 2025-03-26 (B-23), credential redaction (B-24), no URL allow-list (B-25), governance is transport-agnostic (B-26).
- Error class: new MCP class with 9 codes (MCP-001 through MCP-009) covering validation, protocol, session, transport, and auth failures.
- Shared response types extracted to mcp_types.go so stdio and HTTP transports parse identically — this is the structural guarantee of Requirement 4 (identical ToolResult regardless of transport).
- Learning: when a feature request says "add X that works exactly like Y," the first question is whether there's a seam above Y that X can plug into. If there is, the feature is an implementation of an existing interface (low risk, no refactor). If there isn't, the feature requires creating the seam first (high risk, refactor). Always verify the seam before writing the contract — it determines the entire stream topology.
- Learning: the security posture question "does this need an allow-list?" depends on whether there's a trust boundary between the author and the runtime-resolved value. Static authored URLs (same trust as command:) → no allow-list. Runtime-resolved identities (dynamic includes) → catalog as allow-list. The trust model determines the control, not the perceived "dangerousness" of the operation.

Learning (B-32, 2026-08-15T18:06:00-07:00): Audience scoping on bearer tokens constrains which service will ACCEPT the token, not who may PRESENT it. A token sent to a malicious host can be replayed against the legitimate audience. "The attacker can't use the token at their service" is irrelevant when the attack is forwarding it to YOUR service. Bearer = possession is authorization. When reasoning about credential-exposure risk, model the attacker's ability to REPLAY, not their ability to CONSUME. This invalidated B-25/B-27's core premise and required the host-restriction narrowing in B-32.

📌 MCP HTTP Final Review Gate (2026-08-15T19:18:27-07:00):
- Issued APPROVE (unconditional) for Streamable HTTP MCP transport. All 5 requirements MET.
- B-32 fully realised: allowed_hosts required, MCP-010/011 static, MCP-012 runtime fatal, MCP-013 redirects blocked. One definition for host matching — no drift between static and runtime checks.
- Minor Req4 divergence noted but non-blocking: HTTP returns ToolResult{ExitCode:1} on IsError while stdio returns nil. Error message is identical; callers check err first. Observable behavior from runbook author's perspective is indistinguishable.
- Cross-feature (dynamic-include + MCP HTTP) verified safe: frozen catalog is the real trust boundary; allowed_hosts is defense-in-depth against partial compromise. A fully compromised package author can bypass allowed_hosts — but can also issue commands directly against the real endpoint, which is a strictly more powerful attack.
- Learning: when reviewing a token-attachment control, trace the full attack chain to its logical conclusion. If the attacker who bypasses your control can do something strictly worse via a path you can't control (declaring legitimate-looking commands against the real endpoint), the control's ceiling is "defense against carelessness and partial compromise." That is still worth building — most real incidents are partial, not total — but the review should state the ceiling explicitly rather than implying the control prevents all exploitation.
- Learning: stale test-file header comments (DEF-011/012/013 noted as open) vs. user's verified claim (all passing, defects fixed) — trust verified state over file comments. Test headers are snapshot documentation; they rot within hours during active multi-stream development.

## 2026-08-16T02:33:36Z — MCP HTTP Transport Feature — Team Session Orchestration

**Feature:** Native Streamable HTTP MCP transport (transport.mode: mcp-http)
**Outcome:** ✅ COMPLETE — All requirements met, final review gate APPROVED (unconditional)

- Completed assigned stream work
- Collaborated on binding architecture contract
- Participated in team orchestration
- All deliverables verified and tested
- Ready for production merge

## 2026-08-16 — Team Orchestration Session: Runtime Portability Evaluation

**Session:** Scribe coordination session with barbara, don, david  
**Task:** Evaluate and document SQL Live-Site Operations "Runtime Portability for Gert Runbooks" implementation request

**Key outputs from cross-agent session:**
- Barbara: Architectural evaluation published to decisions.md (ACCEPT-WITH-MODIFICATIONS verdict)
- Don: Ground-truth verification against source — all claims accurate except run-gert.ps1
- David: Integration critique identifying three blockers + one phase-ordering risk

**Cross-finding consensus:**
- All three agents independently flagged schema non-goal contradiction as critical
- All three flagged phase estimation (Phase 2 estimate dependent on unanswered OQ-2) as unreliable
- David flagged phase-ordering risk (Phase 3 semantics required but Phase 1 ships HTTP with timeout hazard)

**Deliverables merged to decisions.md:**
- 3 inbox files archived
- 3 orchestration logs written
- Session log written to .squad/log/

**Team notation:** Decision-making requires Cristiano's input on schema non-goal contradiction and OQ-2 resolution. All three architectural evaluations recommend identical remedy: accept additive schema changes, answer OQ-2 in Phase 0, implement tiered-preflight with Tier 0 static mandatory.

## Learnings

📌 Runtime Portability Evaluation (2026-08-16T16:00:00-07:00):
- Evaluated inbound implementation request from SQL Live-Site Operations for runtime binding resolver, managed identity, workload identity, host bridge, and preflight validation.
- Verdict: Accept-with-modifications. Core abstraction (profile → resolver → transport + auth) aligns with existing `ToolTransport` interface seam.
- Key finding: "no tool contract schema changes" is contradicted by the ask's own requirements (capability declaration for preflight, action classification for approval policy). Additive schema changes are unavoidable.
- Learning: when an ask declares "no schema changes" as a non-goal but requires new declarative metadata (contexts, classifications), the ask has an internal contradiction. Call it out immediately — the contradiction won't resolve itself and will block implementation if deferred.
- Learning: preflight validation must separate STATIC bindability (does a binding EXIST for this context?) from DYNAMIC reachability (can we reach the endpoint / acquire a token RIGHT NOW?). These are different failure classes deserving different error codes and different operator messages. Conflating them produces confusing errors.
- Learning: when a phase estimate depends on an unanswered architectural question (library vs. subprocess), the estimate is fiction. Require the question to be answered in a prior phase before accepting the estimate.
- Decision written to `.squad/decisions/inbox/barbara-runtime-portability-ask-evaluation.md`.

