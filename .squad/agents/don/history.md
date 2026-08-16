# don

Current role: Implementation engineer.

Session: Dynamic Runbook Includes (2026-08-15)
- Feature complete and approved
- Ready for merge

Session: Streamable HTTP MCP Transport (2026-08-16) — updated after Ken's recon
- Implemented `MCPHTTPTransport` in `internal/tool/mcp_http.go`
- SSE parser correlates by expectedID (not just "first non-nil id") — skips notifications and stale responses
- Session ID lifecycle: capture on init, echo on subsequent; 404→re-init-once; 401→invalidate-and-retry
- Runtime wiring via `TokenGate` (allowed-hosts check + audit trace); proven by `TestMCPHTTPTransport_RuntimeWiring`
- Extracted shared types to `mcp_types.go` (Result as json.RawMessage)
- Fixed Ken's import cycle: `auth_gate.go` → `executor.EmitterFromContext` (nonexistent) → created `emit_ctx.go`
- initialize() explicitly checks resp.Error before setting initialized=true (not replicating stdio defect)
- 11 tests pass; `internal/conformance` + `pkg/pjvm` now pass (import cycle was the root cause)

Key learnings:
- Always build first before implementing — Streams A/C had partially landed
- SSE correlation must happen at parse time, not post-hoc
- Import cycles in teammate's files in my package territory are mine to fix

Session: Streamable HTTP MCP Transport — follow-up round (2026-08-16)
- Deleted `emit_ctx.go` (dead code): Ken had already landed `trace.EmitterFromContext` in `auth_gate.go`; my import-cycle fix created a parallel context key that nothing seeded in production
- B-27 audit trail is now live: engine seeds `trace.WithEventEmitter`; gate reads it correctly
- MCP-013 (B-30) redirect blocking: `NewMCPHTTPTransport` sets `httpClient.CheckRedirect` when `gate != nil`; unauthenticated transports follow redirects freely
- End-to-end wiring now proven at two levels: `TestMCPHTTPTransport_RuntimeWiring` (runtime dispatch) + new `TestMCPHTTPTransport_SchemaRuntimeWiring` (schema → mapTransport → runtime)
- Ken's MCP-012 fatal behavior (disallowed host in AttachToken) was already in the file — my earlier read caught an intermediate state
- Tess's DEF-011 skip can be removed (CheckRedirect is now installed)
- go build ✅; go test ./... ✅; go vet — only two pre-existing internal/serve warnings

Detailed history: .squad/decisions/inbox/don-mcp-http-streamb.md

Detailed history: .squad/agents/don/history-archive.md

## 2026-08-16 — Team Orchestration Session: Runtime Portability Evaluation

**Session:** Scribe coordination session with barbara, don, david  
**Task:** Evaluate and document SQL Live-Site Operations "Runtime Portability for Gert Runbooks" implementation request

**Don's session outputs:**
- Ground-truth verification against source code completed
- All claims validated except run-gert.ps1 (not in Gert core)
- Key findings shared with Barbara and David for cross-agent analysis

**Team consensus findings:**
- Schema non-goal (§13) contradicts ask's own capability-declaration requirements (all agents flagged independently)
- Phase 2 estimate relies on OQ-2 answer; currently meaningless (Don + Barbara aligned)
- Lowest-cost preflight improvement: wire existing `ToolGovernance.AllowedEnvironments` into planner's tool-resolution path (Don's specific recommendation)

**Deliverables merged to decisions.md.**

## Learnings

Session: Runtime Portability Ground Truth (2026-08-16T15:59:48-07:00)
- Verified all 13 claims in §3.2/§5/§8 of the SQL Live-Site Operations ask against live Gert source
- One false claim: `run-gert.ps1` does not exist in Gert core (zero .ps1 files in the tree)
- All "genuinely absent" claims confirmed absent: no runtime binding resolver, no profile concept, no managed/workload identity providers, no VS Code host bridge
- Key surprise: `ToolGovernance.AllowedEnvironments` and `RequiresCapabilities` are declared in `pkg/schema/tool.go` but **zero Go code reads or enforces them** — ghost fields
- Tool-not-found is a PLAN-TIME failure (PLAN-010) via `planner.go:resolveTool()`, not a runtime failure — Gert already fails closed before the first step when a tool name/action is unresolvable
- `AzureCLIAuthProvider.Token(ctx)` already propagates deadlines correctly via `exec.CommandContext` — §10.3 question is answered as YES today, no Phase 1 work needed for this specific point
- `DefaultToolRuntime.persistent` cache (transport-by-name) would need to be context/profile-scoped for the binding resolver to work correctly — non-trivial
- `OverlayRegistry` (`internal/tool/overlay_registry.go`) and `--package-map` are the existing precedent for runtime tool substitution; the binding resolver should build on this pattern
- Transport layer (`MCPHTTPTransport`) has no `request_id` field and no idempotency/late-result logic — §10.4 is real new scope
- Phase 1 estimate (4–6 weeks) is optimistic; Phase 2 VS Code estimate is wildly optimistic (misses extension-side work)
- Deliverable written to: `.squad/decisions/inbox/don-runtime-portability-ground-truth.md`

## 2026-08-16T02:33:36Z — MCP HTTP Transport Feature — Team Session Orchestration

**Feature:** Native Streamable HTTP MCP transport (transport.mode: mcp-http)
**Outcome:** ✅ COMPLETE — All requirements met, final review gate APPROVED (unconditional)

- Completed assigned stream work
- Collaborated on binding architecture contract
- Participated in team orchestration
- All deliverables verified and tested
- Ready for production merge

