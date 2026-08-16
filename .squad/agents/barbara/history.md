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

