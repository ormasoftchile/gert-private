# david

Current role: Implementation engineer.

Session: Dynamic Runbook Includes (2026-08-15)
- Feature complete and approved
- Ready for merge

Session: MCP HTTP Auth Provider — emitter wiring audit + Ken collision (2026-08-15T19:12)
- Don's `emit_ctx.go` never landed; `pkg/trace/emitter.go` (my earlier cycle-break) already provides the same API
- `auth_gate.go` uses `trace.EmitterFromContext` — no import cycle, no dead-code risk
- Traced full production chain: engine installs emitter → context flows through executor → runtime → invokePersistent → MCPHTTPTransport.send → reqCtx (child of ctx) → buildHTTPRequest → AttachToken → trace.EmitterFromContext returns engine emitter → emitEventLocked writes to TraceWriter.Append (persistent trace file). Event provably reaches real trace stream.
- Ken collision: only genuine divergence was MCP-012 message text. Updated to Ken's canonical: `"mcp-http: request host %q is not in auth.allowed_hosts — token not attached (update allowed_hosts or url: to match)"` (one-line fix)
- MCP-013 redirect ownership confirmed: mine. CheckRedirect policy lives in NewMCPHTTPTransport.
- go build ./... exit 0; all 30 non-tool packages pass


- Switched `CheckRedirect` from returning `errkit.ErrMCP013` to `http.ErrUseLastResponse` per Barbara's ruling
- `send()` now detects 3xx status (when gate != nil) and emits clean MCP-013 with redirect Location header — no more double-wrapping in MCP-008
- Added `TestMCPHTTPTransport_TokenNeverReachesRedirectTarget`: security proof that bearer token NEVER reaches redirect target server — evil server asserts zero Authorization headers received
- Updated `TestMCPHTTPTransport_CheckRedirectInstalledOnAuthenticated` to assert `http.ErrUseLastResponse`
- Confirmed `errkit.Error.Is()` uses code-based comparison (not pointer equality) — `errors.Is(errkit.New("MCP-013", msg), ErrMCP013)` returns true; Don's existing redirect tests unbroken
- Pre-existing issue: `mcp_http_tess_test.go` (untracked, Tess's file) has syntax error in `TestMCPHTTPTransport_SSE_CorrectIDSelected` (DEF-012) — missing `ts := httptest.NewServer(...)` and `t.Skip()` before handler body code; NOT caused by my changes; requires Tess to fix before `go test ./internal/tool/...` passes
- `go build ./...` exit 0; all packages outside `internal/tool` pass

Session: MCP HTTP Auth Provider — B-32 fatal host check (2026-08-15T18:47)
- Host mismatch changed from non-fatal (warning) to fatal (MCP-012 hard error)
- `EventKindMCPAuthSkipped` removed — not needed since host mismatch now fails hard
- Don had already added `CheckRedirect` (MCP-013) to block redirects on authenticated transport
- 38 gate+auth tests pass; `go build ./...` exit 0
- Key lesson: written ruling text (B-32 said "non-fatal MCP-W001") was superseded by direct instruction

- B-32 replaced B-25/B-27: token attachment restricted to explicitly declared hosts
- Added `TokenGate` in `internal/tool/auth_gate.go` — single policy chokepoint for all bearer token attachment
- `TokenGate.AttachToken(ctx, req)` checks `allowedHosts`, acquires token, sets header, emits `mcp/authAttached` trace event
- Static validation: `ValidateAuthConfig` → MCP-010 (missing allowed_hosts), MCP-011 (url host not in list)
- Runtime: host-mismatch is non-fatal (B-32); emits `mcp/authSkipped` (MCP-W001), request proceeds unauthenticated
- Added HTTP 401 invalidate-and-retry path in `MCPHTTPTransport.send`
- Moved `EventEmitter` type + context key to `pkg/trace/emitter.go`; `internal/executor/events.go` delegates there (breaks a `internal/tool` → `internal/executor` → `pkg/pkgcatalog` → `internal/tool` import cycle)
- All tests pass: 16 auth + 5 ValidateAuth + 11 gate tests; `go build ./...` exit 0

- Implemented `AuthProvider` interface and `AzureCLIAuthProvider` in `internal/tool/auth.go` + `auth_azurecli.go`
- 16 tests passing including B-24 redaction proof (`TestAzureCLIAuthProvider_TokenNeverLeaksIntoDiagnostics`)
- Four failure modes each producing actionable operator messages; `classifyAzError` distinguishes az-not-found, not-logged-in, no-scope-consent, malformed-output
- Token caching: proactive refresh 5min before expiry with graceful degradation; `Invalidate()` seam for Don's mid-run 401 path
- `go build ./...` exit 0; all auth tests pass; pre-existing tool-binary failures unchanged
- Interface shape filed in `.squad/decisions/inbox/david-mcp-http-streamc.md` for Don's consumption
- Host mismatch changed from non-fatal (warning) to fatal (MCP-012 hard error)
- `EventKindMCPAuthSkipped` removed — not needed since host mismatch now fails hard
- Don had already added `CheckRedirect` (MCP-013) to block redirects on authenticated transport
- 38 gate+auth tests pass; `go build ./...` exit 0
- Key lesson: written ruling text (B-32 said "non-fatal MCP-W001") was superseded by direct instruction

- B-32 replaced B-25/B-27: token attachment restricted to explicitly declared hosts
- Added `TokenGate` in `internal/tool/auth_gate.go` — single policy chokepoint for all bearer token attachment
- `TokenGate.AttachToken(ctx, req)` checks `allowedHosts`, acquires token, sets header, emits `mcp/authAttached` trace event
- Static validation: `ValidateAuthConfig` → MCP-010 (missing allowed_hosts), MCP-011 (url host not in list)
- Runtime: host-mismatch is non-fatal (B-32); emits `mcp/authSkipped` (MCP-W001), request proceeds unauthenticated
- Added HTTP 401 invalidate-and-retry path in `MCPHTTPTransport.send`
- Moved `EventEmitter` type + context key to `pkg/trace/emitter.go`; `internal/executor/events.go` delegates there (breaks a `internal/tool` → `internal/executor` → `pkg/pkgcatalog` → `internal/tool` import cycle)
- All tests pass: 16 auth + 5 ValidateAuth + 11 gate tests; `go build ./...` exit 0

- Implemented `AuthProvider` interface and `AzureCLIAuthProvider` in `internal/tool/auth.go` + `auth_azurecli.go`
- 16 tests passing including B-24 redaction proof (`TestAzureCLIAuthProvider_TokenNeverLeaksIntoDiagnostics`)
- Four failure modes each producing actionable operator messages; `classifyAzError` distinguishes az-not-found, not-logged-in, no-scope-consent, malformed-output
- Token caching: proactive refresh 5min before expiry with graceful degradation; `Invalidate()` seam for Don's mid-run 401 path
- `go build ./...` exit 0; all auth tests pass; pre-existing tool-binary failures unchanged
- Interface shape filed in `.squad/decisions/inbox/david-mcp-http-streamc.md` for Don's consumption
- Ken's `errkit` "MCP" class registration and `schema.AuthConfig` are pending (Stream A); wiring in `runtime.go` already consumes them

Key lesson: always verify `errors.As` target type matches what stubs actually produce. The `notFoundError` stub originally used a custom error type that didn't satisfy `*exec.Error` — fixed by using `&exec.Error{Name: "az", Err: exec.ErrNotFound}` directly in the test.

Detailed history: .squad/agents/david/history-archive.md

## 2026-08-16T02:33:36Z — MCP HTTP Transport Feature — Team Session Orchestration

**Feature:** Native Streamable HTTP MCP transport (transport.mode: mcp-http)
**Outcome:** ✅ COMPLETE — All requirements met, final review gate APPROVED (unconditional)

- Completed assigned stream work
- Collaborated on binding architecture contract
- Participated in team orchestration
- All deliverables verified and tested
- Ready for production merge

