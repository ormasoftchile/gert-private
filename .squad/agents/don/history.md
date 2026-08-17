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

---

## Learnings — Phase 1B Rev2 Auth Corrections (2026-08-17)

### Correction A: Managed-Identity vs Workload-Identity Split

**Verdict:** Correct and straightforward.

`NewAuthProvider` (`internal/tool/auth.go:29`) is a single switch on the provider name string. Adding `"managed-identity"` as its own case and `"workload-identity"` as a separate future case is mechanical — no structural change required. `knownAuthProviders` (`internal/tool/validate_transport.go:16`) is a `map[string]bool`; adding an entry takes one line.

**Fail-closed confirmed:** An unknown provider name hits the `default:` branch of `NewAuthProvider` and returns `MCP-002`. `knownAuthProviders` rejects the name at parse time (before runtime). There is no ambient detection anywhere in the provider selection path — the switch is exhaustive on the explicit string. A profile or tool file with an unlisted provider name fails hard at scan time.

**Revised estimate for `managed-identity` (IMDS only):** 1.5 days. Workload identity (`AZURE_FEDERATED_TOKEN_FILE` exchange) was the bulk of the complexity in the combined 2-day estimate. IMDS only is a single HTTP call to `http://169.254.169.254/metadata/identity/oauth2/token`, token caching, and 5-minute refresh — simpler than AzureCLI (no subprocess). Workload identity deferred to its own phase; estimate for that when scoped.

---

### Correction B: Auth and Endpoint Override Safety

#### TokenGate existence and behavior

**TokenGate EXISTS and is fully implemented.** File: `internal/tool/auth_gate.go`.

- Constructed at `internal/tool/runtime.go:71–74` inside the `TransportMCPHTTP` case of `DefaultToolRuntime.Invoke`.
- `NewTokenGate(provider, def.Auth.Scope, def.Auth.AllowedHosts)` — provider and allowed_hosts come directly from the tool definition's auth block.
- `AttachToken` (`auth_gate.go:82`) checks `req.URL.Hostname()` against `allowedHosts` **before** acquiring any token. If the host is not in the list, it returns `MCP-012` and the request is not sent. This is the correct order.
- Host matching is exact, case-insensitive, port-stripped. Wildcards are deliberately not supported. Redirect prevention is in `NewMCPHTTPTransport` (redirects trigger `MCP-013` when a gate is present).

**TokenGate already does runtime host validation before token attachment.** Requirement 6 of Correction B is already satisfied structurally, as long as the gate is constructed with the correct `allowedHosts`. The outstanding question is: when a profile overrides the endpoint, does the gate's `allowedHosts` get updated to match?

#### `allowed_hosts` enforcement status

**Enforced — doubly.**

1. **Static (parse time):** `ValidateTransportConfig` (`validate_transport.go:60–74`) checks that `allowed_hosts` is non-empty when auth is configured (MCP-010) and that the URL hostname is in `allowed_hosts` (MCP-011). This fires at tool file scan.
2. **Runtime (per request):** `TokenGate.AttachToken` (`auth_gate.go:82–91`) checks `req.URL.Hostname()` against `allowedHosts` on every request. Returns MCP-012 and blocks the request if the host is not listed.

`tools/icm.tool.yaml:31` declares `allowed_hosts: [icm-mcp-prod.azure-api.net]`. That value reaches `def.Auth.AllowedHosts` at `runtime.go:74` and is passed directly to `NewTokenGate`. **This field is enforced, not parsed-and-dropped.** Distinct from `AllowedEnvironments`, `ProfileToolOverride.Endpoint`, and `Contract.Idempotent` which are parsed-but-unread.

#### The gap: profile endpoint override vs. TokenGate

`ProfileToolOverride.Endpoint` (`pkg/schema/profile.go:108`) is parsed but **never read in execution** (confirmed in Phase 1B prior investigation — `internal/tool/runtime.go:70–77` reads only `def.Auth` and `def.URL` from the tool definition, not from any profile). When runtime binding (Item 2) is implemented and the profile's endpoint override is applied, the following must hold:

- The overridden URL host must be checked against `allowed_hosts` **at plan time (Tier 0)** before execution begins.
- `TokenGate` must be constructed with `allowedHosts` from the tool definition's auth block — not from the profile. The profile can change where the token goes (endpoint), but the list of hosts that may receive it must remain statically declared in the tool definition.
- If the profile endpoint's host is NOT in `def.Auth.AllowedHosts`, planning must fail with a new PLAN-01x sentinel — not fail mid-execution when `AttachToken` fires MCP-012.

**Tier 0 seam for endpoint-vs-allowed_hosts check:** `checkToolEnvironmentPreflight` in `internal/planner/preflight.go:89`. This function already receives `def *schema.ToolDef` and `profile *schema.RuntimeProfile`. The planner has `p.profile` (set at `planner.go:65`). Adding a new check here that:
1. Reads `profile.Tools[def.Name].Endpoint` (if set)
2. Parses its hostname
3. Checks it against `def.Auth.AllowedHosts`
4. Returns a new `PLAN-013` / `ErrEndpointNotInAllowedHosts` if not found

...is a natural extension of the existing pattern. The data is available at that point: `def` carries `Auth.AllowedHosts`, `profile` carries the tool override. No new planner fields are needed.

The planner does NOT currently have access to a resolved profile-overridden endpoint — because profile endpoint override isn't wired yet. When Item 2 (runtime binding) lands, the endpoint resolution must happen here or be passed in. The simplest approach: `checkToolEnvironmentPreflight` reads `profile.Tools[def.Name].Endpoint` directly and validates it.

#### Profile top-level auth clobbering tool scope/audience

**Currently not a risk — because there is no top-level `auth` field on `RuntimeProfile`.** 

`pkg/schema/profile.go:130–137` shows `RuntimeProfile` fields: `APIVersion`, `ID`, `Context`, `Attendance`, `Approval`, `Transport`, `Tools`. No `Auth` field exists at the top level. The comment at `profile.go:95` says `ProfileTransport` carries "future auth parameters" but that is a comment, not a field.

`ProfileToolOverride` (`profile.go:103`) has `Endpoint` and `Mode` — no `Provider`, `Scope`, or `AllowedHosts` fields.

**Structural prevention today:** There is no mechanism by which a profile can inject a credential mechanism, scope, or audience into the runtime path — because those fields don't exist on the profile schema. Any implementation of profile-level auth override for Phase 1B must add these fields, and that is where the SQL Live-Site ruling must be enforced structurally.

**What must be prevented (design constraint for Item 2):** When adding profile-level auth to the schema, `provider` may be added (mechanism selection), but `scope`/`audience` must NOT be overridable at the profile level unless the tool definition has explicitly declared them overridable. The profile must not silently replace a tool's declared scope.

#### Additional work estimate for the six conditions

Conditions 1, 2, 3 (transport is mcp-http, provider explicitly named, scope explicitly defined) are **already enforced** by existing schema validation — `ValidateTransportConfig` rejects auth without an explicit provider name and scope. Nothing new needed for those.

Conditions 4 + 5 (host in allowed_hosts + Tier 0 preflight check): ~0.5 days. The seam is ready; the check is a hostname parse and list lookup inside `checkToolEnvironmentPreflight`. Requires a new PLAN-01x sentinel in errkit. Straightforward.

Condition 6 (TokenGate repeats host validation before attaching token): **already satisfied** by existing `AttachToken` implementation. Zero new work, as long as the gate is constructed with correct `allowedHosts` when profile endpoint override is applied (covered by conditions 4+5).

**Net additional estimate:** +0.5 days, folded into Item 2 (runtime binding). Not a separate work item.

---

## Phase 1B Impact

- Must implement managed-identity auth before ICM proof can be demonstrated
- Auth precedence ruling: Profile top-level auth overrides tool-definition auth
- Both claims verified with file:line evidence; no speculation
- Timeline: ~2-3 days for both Claims 1 & 2 sequentially

---

## Phase 1B Rev 2 Acceptance and Corrections (2026-08-17)

**Status:** All eight corrections from SQL Live-Site Operations verified correct.

### Key Finding: TokenGate Already Implemented and Enforced

TokenGate (`internal/tool/auth_gate.go:23`) is complete and production-ready. Most significant finding: this was the **first field in the codebase found to be genuinely enforced at runtime** rather than parsed-but-dead.

- `AttachToken` (`auth_gate.go:82`) validates host before acquiring token on every request.
- `allowed_hosts` enforced at parse time (MCP-010/011) and runtime (MCP-012).
- Redirect bypass blocked (MCP-013).
- Four of six endpoint-safety conditions already satisfied; only PLAN-013 (Tier 0 preflight) is new (+0.5 days).

### Phase 1B Rev 2 Scope

| Item | Estimate | Status |
|------|----------|--------|
| 1. Managed Identity (IMDS only) | 1.5 days | Revised down from 2 days |
| 2. Runtime Binding + PLAN-013 | 3–4 days | Includes endpoint override validation |
| 3. INDETERMINATE + evidence | 5.5–6 days | Revised up from 4–5 days |
| 4. ICM proof (real contract) | 3 days | Revised up from 1 day; gated on Item 2 + artifacts |
| 5. Fail-fast + harness | 1.5 days | New (moved from Item 2) |
| 6. Credential-leak assertions | 1 day | New Item 6 |

**Revised estimate:** 9–10 days parallelized (was 8–9 days).

### Design Ruling: Profile Auth Schema Invariant

When Item 2 extends `RuntimeProfile` schema:
- MAY add `provider` field (credential mechanism selection)
- MUST NOT add `scope` or `allowed_hosts`

Token gate is always constructed from tool definition's declared auth — profile substitutes credential acquisition mechanism only, never the token's scope or destination hosts.

