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

## 2026-08-16T02:33:36Z — MCP HTTP Transport Feature — Team Session Orchestration

**Feature:** Native Streamable HTTP MCP transport (transport.mode: mcp-http)
**Outcome:** ✅ COMPLETE — All requirements met, final review gate APPROVED (unconditional)

- Completed assigned stream work
- Collaborated on binding architecture contract
- Participated in team orchestration
- All deliverables verified and tested
- Ready for production merge

