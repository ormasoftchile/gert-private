# Don — History Summary

## Overview

**Stream:** Backend transport and authentication. Implemented MCP HTTP transport, verified Phase 1B auth gaps.

**Key Contributions:**
- Designed and implemented MCPHTTPTransport for mode: mcp-http (Slice B, Phase 1A)
- Verified managed-identity auth absence (Claim 1, Phase 1B) — requires 2-day implementation
- Verified headless ICM proof absence (Claim 2, Phase 1B) — requires 1-day proof + external credential provisioning

---

## Phase 1A Work Summary

### MCP HTTP Transport Implementation (Complete)

**Files added:**
- internal/tool/mcp_http.go — MCPHTTPTransport implementing ToolTransport
- internal/tool/mcp_types.go — Shared mcpContent, mcpCallResult, mcpResponse types
- internal/tool/mcp_http_test.go — 10 tests (JSON/SSE, multi-line, sessions, MCP-005/006)

**Files modified:**
- internal/tool/mcp.go — Deduplicated types, updated Invoke to use resp.callResult()
- internal/tool/runtime.go — Replaced placeholder with NewMCPHTTPTransport dispatch
- pkg/schema/tool.go — Removed duplicate AuthConfig type
- pkg/tool/tool.go — Removed duplicate TransportMCPHTTP constant

**Context:** Streams A (Ken) and C (David) had already landed:
- TransportMCPHTTP constant, URL/Auth fields, AuthConfig type
- AuthProvider interface and AzureCLIAuthProvider
- MCP-001…009 sentinels, validate_transport.go validation

**Key Design:** Used json.RawMessage for mcpResponse.Result to support both tools/call and tools/list shapes.

---

## Phase 1B Scope Verification (2026-08-17)

### Claim 1: Managed-Identity Auth Absent — CONFIRMED

**Evidence:**
- NewAuthProvider (internal/tool/auth.go:29) recognizes only "azure-cli" case; others return MCP-002
- Static validation rejects "managed-identity" at parse time (knownAuthProviders)
- No Azure SDK in go.mod; implementation can use stdlib net/http + IMDS/Workload Identity

**What is needed:**
- New file internal/tool/auth_managed_identity.go
- ManagedIdentityAuthProvider struct implementing Token(ctx) and Invalidate()
- One case in NewAuthProvider, one entry in knownAuthProviders
- Token caching with 5-minute refresh buffer (follow AzureCLI pattern)

**Estimate:** 2 days (implement both IMDS and Workload Identity paths + unit tests)

### Claim 2: Headless ICM Proof Absent — CONFIRMED

**Evidence:**
- icm-tsg-router does not exist (zero matches)
- tools/icm.tool.yaml exists with name: icm, three actions (get_incident, update_incident, resolve_incident)
- No runbook invokes it via managed identity
- ICM tests use fabricated in-memory catalog, not real MCP endpoint

**What self-contained read-only proof requires:**
1. Managed-identity auth provider (prerequisite on Claim 1)
2. Runbook or invocation calling icm.get_incident only (read-only action)
3. Mock MCP server (net/http/httptest) accepting Authorization bearer token, returning minimal incident JSON
4. Integration test wiring them together

**Production requirements (external dependencies):**
- Managed-identity credential bound to runner (VM MSI or Workload Identity in CI)
- Scope grant for api://icmmcpapi-prod/mcp.tools
- Network access to icm-mcp-prod.azure-api.net
- Cannot provision/assert today

**Estimate:** 1 day after managed-identity lands (proof work only; production credential provisioning external, TBD)

---

## Phase 1B Impact

- Must implement managed-identity auth before ICM proof can be demonstrated
- Auth precedence ruling: Profile top-level auth overrides tool-definition auth
- Both claims verified with file:line evidence; no speculation
- Timeline: ~2-3 days for both Claims 1 & 2 sequentially

