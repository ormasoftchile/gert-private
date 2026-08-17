## 2026-08-16 — Architectural Evaluation: Runtime Portability for Gert Runbooks (Barbara)

**Date:** 2026-08-16  
**By:** Barbara (Lead / Architect)  
**Requested by:** Cristiano (ormasoftchile)  
**Input:** SQL Live-Site Operations implementation request, §1–§15  

**Verdict: Accept-with-Modifications.** The ask is architecturally sound. The layering aligns with the seam that already exists: `pkg/tool.ToolTransport` interface and `ToolRuntime.Invoke` dispatch. However, §13's "no tool contract schema changes" non-goal contradicts the ask's own primary concern (capability preflight requires declared context support). Recommended: acknowledge the schema change is needed (additive, non-breaking); split preflight into static (Tier 0, default, mandatory) and dynamic tiers (Tier 2, opt-in); add `gert plan` dry-run; resolve whether library-vs-subprocess embedding is the target for Phase 2.

**Full analysis:** See `.squad/decisions/archive/barbara-runtime-portability-ask-evaluation-2026-08-16.md` — comprehensive architecture ruling with phasing critique, contradiction analysis (schema vs. non-goal), and 5 error-code split for capability preflight.

---

## 2026-08-16 — Ground-Truth Report: Runtime Portability (Don)

**Date:** 2026-08-16T15:59:48-07:00  
**Prepared by:** Don (Backend Dev)  
**For:** Barbara (architecture evaluation) and Gert Core Team

**Verdict: All claims accurate except run-gert.ps1 (does not exist in Gert core).** 7 of 8 "exists" claims verified TRUE; `run-gert.ps1` claim FALSE (not in source, may be in consumer repo). All 5 "does not exist" claims verified absent. KEY FINDING: `ToolGovernance.AllowedEnvironments` and `RequiresCapabilities` already exist in schema (`pkg/schema/tool.go:50`) but are completely unenforced in runtime code — **zero references** in any Go file. This is a low-cost path to delivering Cristiano's core concern (preflight "not configured for this environment" error) without waiting for full runtime binding resolver. Also found: `mcp-http` transport fully implemented with SSE parsing, TokenGate enforcement, `AuthProvider` interface, and `AzureCLIAuthProvider`. Phase estimates: Phase 1 is optimistic (should be 6–8wk, not 4–6); Phase 2 wildly optimistic without answering OQ-2 (library vs subprocess); Phase 3 adds idempotency/reconnect infrastructure underspecified in current ask.

**Full report:** See `.squad/decisions/archive/don-runtime-portability-ground-truth-2026-08-16.md` — complete verification matrix with 13 evidence bullets and phase-by-phase scope assessment.

---

## 2026-08-16 — Integration Critique: Runtime Portability (David)

**Date:** 2026-08-16T15:59:48-07:00  
**Prepared by:** David (Integration Engineer)  
**For:** Cristiano and architecture team

**Verdict: Three critical blockers in integration protocol; one critical phase-ordering risk; multiple tiering and error-taxonomy gaps.** (1) §10.1's preflight steps 2–3 answer the wrong question — they conflate static "is this configured?" with dynamic "can I reach it now?"; must split into Tier 0 (static, default, mandatory), Tier 1 (local reachability, opt-in), and Tier 2 (live, opt-in). (2) Error taxonomy `binding/tool-not-found` too coarse; must split into 5 codes: `config/tool-unresolved`, `config/no-binding-for-profile`, `config/binding-incomplete`, `auth/credential-failure`, `transport/endpoint-unreachable`. (3) Host bridge protocol missing four essential fields: `protocol_version`, host capability advertisement (required for Tier 0 preflight), `run_id/step_id` correlation (audit), and explicit `CancelRequest` message type for IPC. (4) **Phase-ordering risk:** §10.4 (reconnect/late-result handling) deferred to Phase 3 (8–12 weeks), but direct-HTTP MCP transport ships in Phase 1 with timeouts/disconnections that leave mutating-action state indeterminate. Mitigation: Phase 1 ships explicit "halt-on-timeout, no retry" policy, not undefined behavior. For destructive actions, must choose: implement proper Phase 3 semantics before ship, OR restrict Phase 1 to read-only tools only.

**Full critique:** See `.squad/decisions/archive/david-runtime-portability-integration-critique-2026-08-16.md` — 3 areas (preflight, host bridge protocol, idempotency), 16 detailed findings, complete tiered-preflight specification, revised error taxonomy with operator messages, and explicit risk call-outs.

---

## 2026-08-15 — Dynamic Runbook Includes Feature — COMPLETE (Phase 4 Sessions)

**Status:** Feature implemented, tested, reviewed, approved. Ready for merge.

**Summary:** Runtime-resolved (dynamic) runbook includes in the `gert` Go repo. Catalog-based dynamic include form (`include: {runbook_ref: "${...}", resolve_from: catalog, with: {...}}`) resolving an identity against approved package-export catalogs at execution time — never an arbitrary filesystem path. Driving use case: an ICM orchestrator that takes an incident ID, applies deterministic rules to suggest a TSG, checks whether that TSG exists as a gert runbook, asks the operator to confirm, then dynamically includes it.

**Spawn Manifest — Sessions in order:**

1. **Barbara** (architect, claude-opus-4.6) — authored the binding architecture contract that gated all implementation streams. Issued rulings B-1 through B-21 across multiple arbitration rounds. Served as the code-review gate. Outcome: **APPROVE WITH CONDITIONS** — all 11 user requirements MET, three documentation-only conditions.

2. **Tess** (tester, claude-sonnet-4.6) — authored conformance vectors before implementation existed. Drove real CLI end-to-end. Found 5 defects by exercising the actual binary. Closed corpus at 26 pass / 3 skip / 0 fail. Outcome: corpus closed, all skips permanent with closing rulings.

3. **Ken** (backend, claude-sonnet-4.6) — Stream 1: schema, parser, errkit error taxonomy (DINC-001..013). Fixed DEF-003 and DEF-005. Locked out after DINC-007 reclassification, unable to apply Barbara B-15 himself.

4. **Don** (backend, claude-sonnet-4.6) — Streams 2 & 3: planner, preview/dry-run rendering, executor (`executeDynamic`). Fixed DEF-001 (two visitors bug), applied B-15 on Ken's behalf during lockout, fixed DEF-006 (entry governance never seeded).

5. **David** (backend, claude-sonnet-4.6) — Stream 4: pin recording, replay re-binding, resume-drift detection. Filed two deferred defects (B-18, B-19/B-20). Corrected coordinator's description three times.

**Architecture Rulings — B-1 through B-21 (Complete List):**

All rulings below are from Barbara's binding contract and rulings document, incorporated with full authority:

**B-1** (DINC-001): Rendered Reference Validation — Empty, path-like, or non-ASCII refs rejected before catalog lookup.
**B-2** (DINC-002): Reference Scope — Bare IDs and package-qualified IDs only; no paths, no URIs, no relative components.
**B-3** (DINC-003): Catalog-Only Constraint — Structurally enforced: no filesystem access, no exec.Command, no http.Client on resolution path.
**B-4** (DINC-004): Include Cycle Detection — Tracked via call stack in context; cycle detected at runtime before execution.
**B-5** (DINC-005): Include Depth Limit — Max 8 levels deep; exceeded depth raises DINC-005.
**B-6** (DINC-006): Required Input Validation — Child inputs checked for required fields; missing required raises DINC-006.
**B-7** (DINC-W007): Extra Input Key Warning — Child runbook receives unexpected input keys; emitted as warning, non-fatal.
**B-8** (DINC-008): Input Enum Validation — Input values matched against declared enums; mismatch raises DINC-008.
**B-9** (DINC-009): Output Schema Mismatch — Child output structure validated against child schema; mismatch raises DINC-009.
**B-10** (DINC-010): Child Package Missing — Child's `requires:` entry not in frozen catalog; raises DINC-010.
**B-11** (DINC-011): Child Tool Unresolvable — Child's `toolRefs:` entry not in frozen catalog; raises DINC-011.
**B-12** (DINC-W001): Governance Widening Warning — Parent governance composed with child governance; any widening emits DINC-W001.
**B-13** (DINC-W002): Deprecated Fields Warning — Deprecated field values emit DINC-W002.
**B-14** (DINC-W003): Include Resolution Warning — Reserved for informational includes-resolution warnings.
**B-15** (DINC-W007 reclassification): Errkit sentinel must classify as `DINC-W007`/class `"DINC-W"`, not `DINC-007`/class `"DINC"`. Required pre-ship.
**B-16** (Frozen Catalog Invariant): Resolved package set is locked at plan time. Include execution cannot trigger new package downloads.
**B-17** (On-Not-Found Behavior): `on_not_found: continue` allows runbook to not exist; sets `result.Vars["runbook_found"] = false`; step completes (not failed).
**B-18** (DEFERRED): Static Include Governance Gap — Non-dynamic branch does not enforce composed governance. Deferred due to production risk.
**B-19** (DEFERRED): when: Field Inert — `CollectorField.When` works; `Step.When` and `IncludeConfig.When` are inert. Code fix deferred; documentation applied.
**B-20** (Documentation): when: Remediation — Schema `description` fields added to three inert `when:` entries. Example warning comment added.
**B-21** (Documentation): require_approval Scope — Schema `description` added noting TTY-only enforcement; auto-approves non-interactive.

**Conformance Corpus Status:**
- **Total vectors:** 29 (designed corpus)
- **Pass:** 26
- **Skip:** 3 (permanent, documented)
- **Fail:** 0

**Deferred Defect Records (Open Work):**

**B-18 — Static Include Governance Gap** (filed by David, 2026-08-15)
Non-dynamic branch of `IncludeExecutor.Execute` does not enforce composed governance. Both eager-static and lazy-static affected. Dynamic includes (fixed) unaffected. Deferred due to production risk; requires separate migration with deprecation/flag plan. Full analysis in `.squad/decisions/inbox/defect-static-include-governance-gap.md`.

**B-19 / B-20 — when: Field Inert in Two of Three Definitions** (filed by David)
`CollectorField.When` works; `Step.When` and `IncludeConfig.When` are schema-accepted but inert at runtime. Code fix deferred; documentation remediation applied. Open work tracked separately.

**B-21 — require_approval TTY-Only Enforcement** (ruled by Barbara)
Enforced only in TTY/interactive mode; auto-approves non-interactive runs. Ruled acceptable as designed for v1 MVP. Documented in schema and examples.

**Key Findings from Implementation:**
- DEF-001: CLI preflight crash on every dynamic-include runbook (fixed by Don: two visitors bug)
- DEF-003: Governance composed but never enforced in dynamic path (fixed by Ken: added enforcement at execution boundary)
- DEF-006: Entry runbook governance never seeded (fixed by Don: seed governance at plan time)

**Architecture Review:** Barbara's review gate approved with three documentation-only conditions, all satisfied. All 11 user requirements verified met.

**Inbox Merged:** 13 files totaling ~200KB (barbara-dynamic-include-contract, barbara-dynamic-include-rulings, barbara-dynamic-include-review, tess-dynamic-include-vectors, tess-dynamic-include-open-questions, tess-dynamic-include-e2e, ken-dynamic-include-stream1, ken-dynamic-include-governance, don-dynamic-include-stream2, don-dynamic-include-stream3, david-dynamic-include-stream4, defect-static-include-governance-gap, defect-when-field-not-evaluated)

## Inbox Merged

Files merged: 9

# Architecture Contract: Streamable HTTP MCP Transport

**Author:** Barbara (Lead / Architect)
**Date:** 2026-08-16T00:55:39Z
**Status:** RATIFIED — all implementation streams build against this contract
**Continues from:** B-21 (dynamic-include rulings). New rulings start at B-22.

---

## 0. Preamble — Existing Architecture Confirmed

Before specifying the new work, I confirm the following about the existing codebase (verified by reading the source directly):

**The transport seam already exists.** `pkg/tool.ToolTransport` is:
```go
type ToolTransport interface {
    Invoke(ctx context.Context, def ToolDef, action string, args map[string]any) (*ToolResult, error)
    Close() error
}
```

`DefaultToolRuntime.Invoke` (`internal/tool/runtime.go`) dispatches on `def.Transport` via a switch statement. Persistent transports (JSONRPC, MCP) are pooled in `r.persistent[toolName]`. This is the integration point for the new HTTP transport.

**The stdio MCP transport is self-contained.** `MCPTransport` (`internal/tool/mcp.go`) owns:
- Process lifecycle (`StartProcess`, `ensureStarted`)
- Content-Length framing (`writeMCPMessage`, `readMCPMessage`, `readContentLength`)
- MCP lifecycle (`initialize` → read response → `initialized` notification)
- `tools/call` and `tools/list` dispatch
- Response parsing (`mcpResponse`, `mcpCallResult`, `mcpContent`)

The JSON-RPC logic is welded to stdio pipes. **No refactor of `MCPTransport` is required.** The new HTTP transport implements the same `ToolTransport` interface independently. The two share response-parsing types (extracted to a shared file) but not I/O logic.

**Schema transport config** (`pkg/schema/tool.go`):
```go
type TransportConfig struct {
    Type    Transport         // legacy enum
    Mode    string            // canonical (overrides Type)
    Command string
    Args    []string
    Env     map[string]string
}
```

Currently has no URL or auth fields. These must be added.

---

## 1. Schema Extension

### 1.1 TransportConfig additions

```go
// pkg/schema/tool.go — additions to TransportConfig
type TransportConfig struct {
    // ... existing fields ...
    URL  string      `yaml:"url,omitempty"  json:"url,omitempty"`
    Auth *AuthConfig `yaml:"auth,omitempty" json:"auth,omitempty"`
}

type AuthConfig struct {
    Provider     string   `yaml:"provider"             json:"provider"`
    Scope        string   `yaml:"scope,omitempty"       json:"scope,omitempty"`
    AllowedHosts []string `yaml:"allowed_hosts"         json:"allowed_hosts"`
}
```

### 1.2 New transport constant

```go
// pkg/tool/tool.go
const TransportMCPHTTP TransportType = "mcp-http"
```

Also add to `pkg/schema/tool.go`:
```go
const TransportMCPHTTP Transport = "mcp-http"
```

### 1.3 Validation rules

| Condition | Error |
|---|---|
| `mode: mcp-http` without `url` | Schema validation error |
| `mode: mcp-http` with `command` or `args` | Schema validation error (mutually exclusive with url) |
| `url` does not start with `https://` | `MCP-001` (HTTPS required for remote endpoints) |
| `auth` configured without `allowed_hosts` | `MCP-010` (fatal: allowed_hosts required when auth present) |
| `url` host not in `allowed_hosts` | `MCP-011` (fatal: host mismatch caught at validation) |
| `auth.provider` is not a recognized provider name | `MCP-002` (unknown auth provider) |

**B-22 Ruling — HTTPS only:** Remote MCP endpoints MUST use HTTPS. Plain HTTP is rejected at validation time. No `--allow-insecure` escape hatch in this iteration. Rationale: MCP tool calls carry operator credentials and can mutate production systems (IcM ticket actions). Allowing HTTP would let a network-position attacker intercept bearer tokens. Localhost exceptions are not needed because a local MCP server would use `mode: mcp` (stdio).

### 1.4 Target YAML shape (matches user requirement exactly)

```yaml
transport:
  mode: mcp-http
  url: https://icm-mcp-prod.azure-api.net/v1/
  auth:
    provider: azure-cli
    scope: api://icmmcpapi-prod/mcp.tools
    allowed_hosts:
      - icm-mcp-prod.azure-api.net
```

---

## 2. HTTP MCP Transport Implementation

### 2.1 Type and location

```go
// internal/tool/mcp_http.go
type MCPHTTPTransport struct {
    mu          sync.Mutex
    url         string
    auth        AuthProvider   // §4
    sessionID   string         // from Mcp-Session-Id response header
    initialized bool
    httpClient  *http.Client
}
```

Implements `pkg/tool.ToolTransport`.

### 2.2 Lifecycle — initialize → tools/call

On first `Invoke` call (lazy, same pattern as stdio):

1. **Send `initialize` request** — HTTP POST to `url` with:
   - Body: JSON-RPC 2.0 `initialize` message (same params as stdio: `protocolVersion`, `capabilities`, `clientInfo`)
   - Header: `Content-Type: application/json`
   - Header: `Accept: application/json, text/event-stream`
   - Header: `MCP-Protocol-Version: 2025-03-26`
   - Header: `Authorization: Bearer <token>` (from auth provider, §4)

2. **Parse response** — detect Content-Type:
   - `application/json` → parse body directly as JSON-RPC response
   - `text/event-stream` → parse SSE stream, extract `data:` lines, assemble JSON-RPC response (§3)

3. **Capture `Mcp-Session-Id`** from response headers. Store in `t.sessionID`.

4. **Send `notifications/initialized`** — HTTP POST to `url` with:
   - Body: JSON-RPC 2.0 notification (no `id` field)
   - Header: `Mcp-Session-Id: <captured value>` (if server provided one)
   - No response body expected (HTTP 204 or 202 accepted)

5. Mark `t.initialized = true`.

Subsequent `Invoke` / `ListTools` calls:
- Send `tools/call` or `tools/list` as HTTP POST
- Include `Mcp-Session-Id` header if present
- Include `Authorization` header (refreshed if expired, §4.3)
- Include `MCP-Protocol-Version: 2025-03-26`
- Parse response (JSON or SSE, §3)
- Convert to `*ToolResult` exactly as stdio does

### 2.3 Protocol version

**B-23 Ruling — Version pinning:** gert advertises `MCP-Protocol-Version: 2025-03-26` (the current Streamable HTTP spec). If the server returns an HTTP 4xx with a body indicating version mismatch, emit `MCP-003` (fatal). If the server simply ignores the header and responds successfully, proceed — interoperability with older servers that don't enforce version headers is acceptable.

### 2.4 Session ID semantics

| Scenario | Behavior |
|---|---|
| Server returns `Mcp-Session-Id` on initialize | Store; send on all subsequent requests |
| Server omits `Mcp-Session-Id` | Proceed without it; do not fail |
| Server returns HTTP 404 on a request with session ID | Session expired — re-initialize (once) then retry the failed request. If re-init also fails, emit `MCP-004` (fatal). |

The session ID lives on the `MCPHTTPTransport` instance, which is pooled per tool name in `DefaultToolRuntime.persistent`. This means the session survives across multiple tool calls within a single run — correct behavior.

### 2.5 Close

`Close()` sends a JSON-RPC `shutdown` notification (best-effort, no response expected) if a session is active, then drops the HTTP client. No process to kill.

---

## 3. Transport Framing — SSE Handling

### 3.1 Content-Type detection

On every HTTP response:
- If `Content-Type` starts with `application/json` → read full body, parse as single JSON-RPC response.
- If `Content-Type` starts with `text/event-stream` → parse as SSE stream (§3.2).
- Otherwise → `MCP-005` (unexpected content type).

### 3.2 SSE parsing

SSE events consist of lines. The parser handles:
- `data: <json>` — append to current event's data buffer (with `\n` between multi-line data)
- Empty line — event boundary; dispatch accumulated data
- `event:` — ignored (MCP uses only the default event type)
- `id:` — ignored (MCP does not use SSE last-event-id)
- Lines starting with `:` — comments, ignored

For a given JSON-RPC request with `id: N`, the SSE stream may contain:
- Zero or more JSON-RPC notifications (no `id` field) — these are progress/log messages; log them but do not treat as the response.
- Exactly one JSON-RPC response with `id: N` — this is the terminal response. Once received, stop reading the stream.

If the stream closes (HTTP connection ends) before a response with matching `id` is received → `MCP-006` (incomplete SSE stream).

### 3.3 Timeout

HTTP requests use the context deadline from `ctx`. If the context has no deadline, a default 120-second timeout is applied. This is configurable via a future `timeout:` field on `TransportConfig` (not in this iteration — hardcode 120s).

---

## 4. Authentication Provider

### 4.1 Interface

```go
// internal/tool/auth.go
type AuthProvider interface {
    // Token returns a valid bearer token. Implementations handle
    // acquisition, caching, and refresh.
    Token(ctx context.Context) (string, error)
}
```

### 4.2 `azure-cli` provider

```go
// internal/tool/auth_azurecli.go
type AzureCLIAuthProvider struct {
    scope string
    // cached token + expiry
    mu       sync.Mutex
    token    string
    expiry   time.Time
}
```

Acquires a token by executing:
```
az account get-access-token --scope <scope> --query accessToken -o tsv
```

**B-24 Ruling — No credential in YAML, trace, log, or error:**
- The `scope` field is the only auth-related value that appears in YAML. It is a resource identifier, not a credential.
- The acquired bearer token MUST NOT appear in: YAML files, trace events, log output, error messages, step output, `ToolResult.Stdout`, or diagnostic dumps.
- If auth fails, the error message reports the failure reason (e.g., "az CLI not authenticated") but NEVER includes the token value.
- The `Authorization` header value is never logged by gert's HTTP client (use a non-logging transport or strip auth headers before any debug logging).

### 4.3 Token caching and refresh

- On first `Token()` call, run `az account get-access-token`.
- Cache the token and parse its expiry from the JWT `exp` claim (or from `az`'s `expiresOn` field).
- On subsequent calls, return cached token if `now + 5min < expiry`.
- If within 5min of expiry, re-acquire proactively.
- If `az` command fails, emit `MCP-007` (auth failure).

### 4.4 Future providers

The `AuthConfig.Provider` field is a string, not an enum. Recognized values in this iteration:
- `"azure-cli"` — described above

Future values (NOT in scope, listed for schema stability):
- `"env"` — read token from an env var named in a `variable:` field
- `"managed-identity"` — Azure managed identity (no CLI dependency)
- `"device-code"` — interactive device-code flow

The schema shape (`provider` + `scope` + future fields like `variable`) is designed to accommodate these without breaking changes.

### 4.5 No auth configured

If `transport.auth` is nil/omitted, no `Authorization` header is sent. This supports MCP servers that use other auth mechanisms (API keys in custom headers via `env`, network-level auth, etc.) — but those mechanisms are NOT part of this feature. Omitting auth simply means no bearer token.

---

## 5. Error Taxonomy

New error class: `"MCP"` (fatal).

| Code | Condition | Fatal? | Message template |
|---|---|---|---|
| `MCP-001` | URL is not HTTPS | Fatal | `mcp-http: url %q must use https://` |
| `MCP-002` | Unknown auth provider | Fatal | `mcp-http: unknown auth provider %q` |
| `MCP-003` | Protocol version mismatch / rejected | Fatal | `mcp-http: server rejected protocol version (HTTP %d)` |
| `MCP-004` | Session expired and re-initialize failed | Fatal | `mcp-http: session expired; re-initialization failed: %v` |
| `MCP-005` | Unexpected response content-type | Fatal | `mcp-http: unexpected content-type %q (expected application/json or text/event-stream)` |
| `MCP-006` | SSE stream closed before response received | Fatal | `mcp-http: SSE stream closed without response for request id %d` |
| `MCP-007` | Auth token acquisition failed | Fatal | `mcp-http: failed to acquire auth token: %v` |
| `MCP-008` | HTTP transport error (connection refused, TLS failure, timeout) | Fatal | `mcp-http: transport error: %v` |
| `MCP-009` | JSON-RPC error response from server | Fatal | `mcp-http: server error %d: %s` |
| `MCP-010` | `auth` configured without `allowed_hosts` | Fatal | `mcp-http: auth.allowed_hosts is required when auth is configured` |
| `MCP-011` | `url` host not in `auth.allowed_hosts` | Fatal | `mcp-http: url host %q is not in auth.allowed_hosts` |
| `MCP-W001` | Token not attached (host not in allowed_hosts — runtime defensive) | Warning | `mcp-http: auth token not attached — host %q is not in allowed_hosts` |

**Requirement 4 guarantee:** `MCP-009` (JSON-RPC error from server) wraps the server's error code and message. A tool-level error (i.e., `result.isError == true` in the MCP response) is returned as a failed `ToolResult` with non-zero exit code — **exactly as stdio MCP does today** (see `mcp.go` line 89: `"mcp tool error: %s"`). The `ToolResult` shape is identical regardless of transport. The runbook author cannot distinguish stdio from HTTP tool errors.

---

## 6. Security Posture

### 6.1 URL allow-listing

**B-25 Ruling — No URL allow-list required.** Unlike dynamic includes (where the catalog is the trust boundary), tool definitions are authored by the same team that authors the runbook. A `.tool.yaml` declaring `url: https://evil.com` is the same trust level as one declaring `command: /usr/bin/evil`. Both are authored artifacts subject to code review and package governance. There is no runtime-resolved URL — the URL is static in the YAML. Adding an allow-list would be security theater without a trust boundary to enforce.

**Contrast with dynamic includes:** Dynamic includes resolve an *identity* at runtime against a frozen catalog — the catalog IS the allow-list. Tool definitions are statically declared — the .tool.yaml IS the authored source. Different trust models, different controls.

### 6.2 TLS verification

Standard Go `http.DefaultTransport` TLS verification applies. No `InsecureSkipVerify`. No custom CA configuration in this iteration (can be added later via `tls:` config on `TransportConfig`).

### 6.3 Token redaction

Per B-24: bearer tokens are never written to any persistent or observable surface. The HTTP client used by `MCPHTTPTransport` must NOT be wrapped in a logging/tracing transport that captures request headers. If gert adds HTTP debug logging in the future, the `Authorization` header must be redacted.

---

## 7. Governance

**B-26 Ruling — Remote tool calls ARE subject to governance.** The governance evaluator receives the tool definition and step context regardless of transport. `deny_commands` does not apply (there is no "command" for HTTP — `deny_commands` gates CLI shell invocations). However:

- `require_approval` applies normally (the approval gate fires before any tool invocation, per `internal/executor/tool.go`).
- `ToolGovernance.RequiresCapabilities` and `AllowedEnvironments` apply normally (checked by the governance evaluator before the executor dispatches).
- Future `deny_tools` or tool-level allow-list governance would apply here. Not in scope for this feature.

The key insight: governance gates at the executor level (before `ToolRuntime.Invoke` is called), not at the transport level. Adding a new transport does not bypass governance because governance fires upstream.

---

## 8. Runtime Wiring

### 8.1 `DefaultToolRuntime.Invoke` extension

Add a case to the switch:
```go
case toolpkg.TransportMCPHTTP:
    return r.invokePersistent(ctx, toolName, *def, action, args, func() toolpkg.ToolTransport {
        return NewMCPHTTPTransport(def.URL, def.Auth)
    })
```

### 8.2 `ToolDef` extension

`pkg/tool.ToolDef` gains:
```go
URL  string         // from TransportConfig.URL
Auth *schema.AuthConfig // from TransportConfig.Auth
```

The existing `internal/tool.RuntimeToolDef` conversion function (which builds `pkg/tool.ToolDef` from `schema.ToolDef`) must copy these fields.

### 8.3 `MCPHTTPTransport.ListTools`

Same interface as `MCPTransport.ListTools`:
```go
func (t *MCPHTTPTransport) ListTools(ctx context.Context, def ToolDef) ([]map[string]any, error)
```

Called by the tool registry's discovery mechanism when a tool is declared with `mode: mcp-http`. The response shape is identical to stdio MCP's `tools/list` result.

---

## 9. Shared Response Types

Extract from `internal/tool/mcp.go` into `internal/tool/mcp_types.go`:
```go
type mcpContent struct { ... }
type mcpCallResult struct { ... }
type mcpResponse struct { ... }
```

Both `MCPTransport` (stdio) and `MCPHTTPTransport` (HTTP) import and use these types for response parsing. This ensures Requirement 4 — identical output shape regardless of transport.

---

## 10. Scope Boundary — What Is NOT In Scope

| Excluded | Rationale |
|---|---|
| Refactoring `MCPTransport` (stdio) | Works as-is; HTTP is a new parallel implementation |
| OAuth device-code flow | Future auth provider |
| Managed identity auth | Future auth provider |
| Custom CA/TLS config | Future extension |
| WebSocket transport | MCP spec supports it; not needed for IcM |
| Tool discovery from remote `tools/list` at catalog-build time | Tools are statically declared in .tool.yaml; remote list is a registry feature |
| `deny_tools` governance | Future governance extension |
| Streaming tool output (SSE progress → live TUI) | Future UX enhancement; responses are buffered |
| `timeout:` field on TransportConfig | Hardcode 120s; add field later |
| Connection pooling / HTTP/2 multiplexing | Go's `http.Client` handles this by default |
| Retry on transient HTTP errors (5xx) | Fail on first error; retry logic is a future enhancement |

---

## 11. Proposed Stream Breakdown

| Stream | Owner | Scope |
|---|---|---|
| **Stream A — Schema + Validation** | Ken | `TransportConfig` extensions, `AuthConfig` type, `TransportMCPHTTP` constant, schema validation (MCP-001, MCP-002), `ClassForCode` extension |
| **Stream B — HTTP Transport + SSE** | Don | `MCPHTTPTransport`, SSE parser, lifecycle (§2), shared types (§9), wiring into `DefaultToolRuntime` (§8) |
| **Stream C — Auth Provider** | David | `AuthProvider` interface, `AzureCLIAuthProvider`, token caching, `MCP-007`, redaction guarantees (B-24) |
| **Stream D — Integration + Tests** | Tess | End-to-end wiring test, mock MCP HTTP server, test vectors for init/session/auth/tool-call/SSE/errors |

Dependencies: B depends on A (needs schema types) and C (needs AuthProvider interface). D depends on all.

---

## Appendix — MCP Protocol Reference (2025-03-26 Streamable HTTP)

For implementers. The MCP Streamable HTTP transport (replacing the deprecated HTTP+SSE transport):

- Client sends JSON-RPC messages as HTTP POST to the server's endpoint URL.
- Server responds with either `application/json` (single response) or `text/event-stream` (SSE stream containing the response plus optional notifications).
- `Mcp-Session-Id` header: server MAY issue on any response; client MUST echo on subsequent requests.
- `MCP-Protocol-Version` header: client MUST send; server validates.
- Notifications (no `id`) sent via POST; server replies with 202/204.
- Server MAY issue HTTP 404 to indicate session expired; client should re-initialize.


# MCP HTTP Transport — Final Architecture Review

**Reviewer:** Barbara (Lead / Architect)
**Date:** 2026-08-15T19:18:27-07:00
**Verdict:** ✅ **APPROVE**

---

## Five Requirements — Verdict

| # | Requirement | Verdict | Evidence |
|---|---|---|---|
| 1 | Add `transport.mode: mcp-http` with a `url` field | **MET** | `TransportMCPHTTP = "mcp-http"` in both `pkg/tool/tool.go:17` and `pkg/schema/tool.go:392`. `TransportConfig.URL` added at `pkg/schema/tool.go:291`. `ValidateTransportConfig` enforces url-required + HTTPS-only + mutual exclusion with command/args. |
| 2 | MCP HTTP lifecycle: initialize, Mcp-Session-Id, notifications/initialized, MCP-Protocol-Version, SSE | **MET** | `mcp_http.go:ensureInitialized` → `initialize()`: sends initialize with `protocolVersion: "2025-03-26"`, inspects response for errors (unlike stdio — B-30 not replicated), captures `Mcp-Session-Id` from headers, sends `notifications/initialized`. `MCP-Protocol-Version` header set on every request. SSE parser (`parseSSEResponse`) handles `data:` lines, multi-line events, comment heartbeats, and ID correlation. Session-expired 404 → re-initialize → retry (§2.4). |
| 3 | Auth provider: Azure CLI bearer token, credentials never in YAML | **MET** | `AzureCLIAuthProvider` in `auth_azurecli.go`: runs `az account get-access-token --scope <scope> -o json`, parses response, caches with expiry, proactive refresh at 5min buffer, `Invalidate()` on 401. Token never appears in errors (classified error messages reference "az CLI" failure reasons, never token value). `AuthConfig.Scope` is the only auth-related value in YAML. B-24 enforced. |
| 4 | Dispatch tools/list and tools/call exactly as stdio, preserving typed outputs | **MET** | Both transports share `mcp_types.go` (`mcpResponse`, `mcpCallResult`, `mcpContent`, `callResult()`). Decode path in `toolResult()`: `content[0].Text` → `Stdout`; JSON parse → `Output` map. Same as stdio (mcp.go lines 69-78). Error message format: `"mcp tool error: %s"` — identical. **One minor note:** HTTP returns a `ToolResult{ExitCode:1, Stderr:text}` alongside the error on tool-level `IsError`; stdio returns `nil, error`. The error is identical; the ToolResult difference is invisible to callers (who check `err != nil` first). Not a requirement failure — the observable behavior from a runbook author's perspective is indistinguishable. |
| 5 | Schema/runtime wiring, validation, and tests | **MET** | Schema: `TransportConfig` extended with `URL` and `Auth *AuthConfig` (with `AllowedHosts`). Runtime: `DefaultToolRuntime.Invoke` has `case TransportMCPHTTP` dispatching to `invokePersistent` → `NewMCPHTTPTransport`. Validation: `ValidateTransportConfig` checks MCP-001/002/010/011 at scan time. Tests: `mcp_http_test.go` + `mcp_http_tess_test.go` + `validate_transport_test.go` — 27/27 passing per verified report. |

---

## B-32 Realisation — FULLY BUILT AS RULED

| Control | Implementation | Verified |
|---|---|---|
| `auth:` requires `allowed_hosts` (MCP-010) | `ValidateTransportConfig`: if `auth != nil && len(AllowedHosts) == 0` → MCP-010 fatal | ✅ |
| URL host in allowed_hosts static check (MCP-011) | `ValidateTransportConfig`: parses URL, extracts `u.Hostname()`, lowercase exact match against list | ✅ |
| Runtime host mismatch fatal (MCP-012) | `TokenGate.AttachToken`: `hostAllowed()` → `errkit.New("MCP-012", ...)` | ✅ Confirmed by source read |
| Redirects refused on authenticated requests (MCP-013) | `NewMCPHTTPTransport`: `client.CheckRedirect = func(...) { return http.ErrUseLastResponse }` when gate != nil. `send()` detects 3xx → `errkit.New("MCP-013", ...)` naming the Location. | ✅ |
| Host matching: `u.Hostname()`, lowercase, exact, no wildcard | Both `ValidateAuthConfig` and `TokenGate.hostAllowed` use identical logic: `strings.ToLower(u.Hostname())` vs `strings.ToLower(entry)` | ✅ One definition, no drift |

---

## B-27: mcp/authAttached Trace Event — NOT DEAD CODE

`TokenGate.AttachToken` emits via `trace.EmitterFromContext(ctx)` on every successful token attachment. The emitter is the same one wired through `engine.go:569` → TraceWriter.Append — the same path used by all other trace events (include/resolved, step/started, etc.). It fires on every authenticated request, which means every `tools/call` and `tools/list` invocation on an authenticated MCP HTTP tool. This is not dead code — it is executed on the hot path of the feature's primary use case.

---

## B-24: Token Escape Surface Audit

| Surface | Safe? | Evidence |
|---|---|---|
| Error messages | ✅ | `classifyAzError` never includes token; MCP-007/012 errors reference scope, host, stderr — never token |
| Trace events | ✅ | `mcp/authAttached` payload: `{url_host, scope}` only |
| ToolResult.Stdout/Stderr/Output | ✅ | Token never enters ToolResult — it exists only in the `Authorization` header |
| HTTP debug logging | ✅ | No HTTP debug/trace transport installed; `http.Client{}` has no `Transport` override that would log headers |
| Panic stack | ⚠️ Acceptable | If `AttachToken` panics after `provider.Token()` returns, the token is on the stack. This is inherent to any in-memory secret and is not a design flaw — panic stacks are not a normal observability surface. |

**Verdict: B-24 is satisfied.** No credential material reaches any persistent or normal-observability surface.

---

## B-31: Documentation Adequacy

Go doc comments on `AuthConfig` (`pkg/schema/tool.go:298-365`) are extensive — they explain the replay threat, the `allowed_hosts` mechanism, and exactly what happens on mismatch. The wildcard non-support is documented in `TokenGate.AttachToken`'s doc comment. An author who writes `*.azure-api.net` gets MCP-012 with a message naming the host and referencing `allowed_hosts`.

Missing: an example `.tool.yaml` file in `examples/`. This is a **nice-to-have**, not a blocker — the doc comments and the schema types are sufficient for an engineer reading code or IDE hover-docs. I recommend adding one before broader adoption but do not condition approval on it.

---

## B-30: Stdio ensureStarted Defect — STILL CORRECT TO DEFER

The HTTP path deliberately does NOT replicate it (confirmed: `initialize()` checks `initResp.Error` before marking `t.initialized = true`). The stdio defect is tracked. The two paths are now asymmetric in a good way — the new code is correct, the old code has a known defect that will be fixed in a dedicated cleanup pass. Deferral remains the right call.

---

## Cross-Feature Question: Dynamic Include + MCP HTTP + Token Replay

A runtime-selected child runbook can declare an `mcp-http` tool with `auth:`. B-32's `allowed_hosts` is the control. Is it sufficient?

**Yes.** The chain of controls is:

1. **Frozen catalog** — the child runbook must come from an approved package. An external attacker cannot inject a tool definition.
2. **MCP-011 (static, at tool scan time)** — the declared URL's host must be in `allowed_hosts`. A package author who sets `url: https://evil.com` must ALSO set `allowed_hosts: [evil.com]` — the mismatch is caught structurally.
3. **The residual threat** — a compromised package author sets BOTH `url: https://evil.com` AND `allowed_hosts: [evil.com]`. B-32 cannot prevent this because the attacker controls the authored artifact. But at this point, the attacker can also set `url: https://icm-mcp-prod.azure-api.net` and issue malicious commands directly — a strictly more powerful attack that no host allow-list prevents. The allow-list stops *accidental* token leakage and *partial* compromise (attacker can modify URL but not allowed_hosts, or vice versa); it cannot stop a fully compromised author.

**The frozen catalog is the real trust boundary.** `allowed_hosts` is defense-in-depth against partial compromise or carelessness. Together they are sufficient.

---

## Deferred Items Confirmed

| Item | Status | Still correct to defer? |
|---|---|---|
| B-18 (static-include governance gap) | Tracked | ✅ |
| B-19/B-20 (`when:` inert) | Tracked, schema remediation required | ✅ |
| B-30 (stdio ensureStarted) | Tracked | ✅ |
| DEF-002 (DINC-009 unreachable) | B-16, permanent | ✅ |

---

## Final Verdict

**✅ APPROVED.** No conditions. The implementation is complete, correct, and faithful to the contract and all rulings through B-33. The security posture (B-32 host restriction, B-33 fatal enforcement + no redirects, B-24 token redaction) is structurally sound. The five requirements are met. The cross-feature interaction with dynamic includes is adequately controlled by the frozen catalog + allowed_hosts defense-in-depth.


# MCP HTTP — Binding Rulings B-22 through B-26

**Author:** Barbara (Lead / Architect)
**Date:** 2026-08-16T00:55:39Z
**Status:** RATIFIED — implementers build directly against these rulings

---

## B-22 — HTTPS only for remote MCP endpoints

**Date:** 2026-08-16T00:55:39Z

**Ruling:** Remote MCP endpoints MUST use HTTPS. Plain HTTP URLs are rejected at validation time with `MCP-001`. No `--allow-insecure` flag in this iteration.

**Rationale:** MCP tool calls carry bearer tokens and can mutate production systems (e.g., IcM ticket remediation actions). Allowing HTTP would let a network-position attacker intercept credentials. Localhost/stdio servers use `mode: mcp` (subprocess) and are unaffected.

---

## B-23 — Protocol version pinning at 2025-03-26

**Date:** 2026-08-16T00:55:39Z

**Ruling:** gert advertises `MCP-Protocol-Version: 2025-03-26` on all HTTP MCP requests. If the server responds with an HTTP error indicating version rejection, emit `MCP-003` (fatal). If the server ignores the header and responds successfully, proceed without error.

**Rationale:** The 2025-03-26 spec defines Streamable HTTP (replacing the deprecated SSE transport). Pinning to this version is forward-looking — it's the current stable spec. Graceful degradation for servers that ignore the header ensures interop with older implementations.

---

## B-24 — No credential material in YAML, traces, logs, or errors

**Date:** 2026-08-16T00:55:39Z

**Ruling:** Bearer tokens acquired by any auth provider MUST NOT appear in: YAML files, trace events, log output, error messages, `ToolResult` fields, step output, or any diagnostic dump. The `scope` field is a resource identifier (not a credential) and may appear in diagnostics.

**Rationale:** Tokens are short-lived secrets. Any persistence or observability surface that captures them creates a credential leakage vector. The only place a token may exist is in-memory within the `AuthProvider` and in the outgoing HTTP `Authorization` header (which must not be captured by debug logging).

---

## B-25 — No URL allow-list required

**Date:** 2026-08-16T00:55:39Z

**Ruling:** Any HTTPS URL is acceptable for `transport.url`. No allow-list, registry, or pre-approval mechanism is required.

**Rationale:** Tool definitions are authored artifacts (.tool.yaml files) subject to the same code-review and package-governance controls as any other authored configuration. The URL is static — not runtime-resolved. Contrast with dynamic includes, where the resolved identity is runtime-variable and the catalog serves as the allow-list. Different trust models require different controls. Adding an allow-list here would be security theater: the same author who can set a malicious URL can also set a malicious `command:`.

---

## B-26 — Remote tool calls subject to governance

**Date:** 2026-08-16T00:55:39Z

**Ruling:** Governance fires at the executor level (before `ToolRuntime.Invoke`), not at the transport level. Adding a new transport does not bypass governance. Specifically: `require_approval`, `RequiresCapabilities`, and `AllowedEnvironments` apply to MCP HTTP tools identically to stdio tools. `deny_commands` does not apply (there is no shell command to deny).

**Rationale:** The tool executor's governance pre-flight (`internal/executor/tool.go`) checks all policies before invoking the tool runtime. This is transport-agnostic by design — the executor doesn't know or care which transport will be used. The new transport plugs in below this gate.

---

## B-27 — B-25 narrowed: audience-scoped tokens are sufficient; no host allow-list needed

**Date:** 2026-08-15T18:03:00-07:00

**Amends:** B-25

**Challenge:** A dynamically-included child runbook (resolved at runtime from the frozen catalog) can declare an MCP HTTP tool pointing to any HTTPS host. David's auth provider attaches a live Azure bearer token. The operator running the parent never reviewed the child's URL. The URL is "authored" but not "reviewed by the person accepting the risk." This is a credential-forwarding-to-attacker-chosen-host risk — the gap is not the URL itself but the **combination of URL + attached credential**.

**Revised Ruling:** B-25 is **narrowed** as follows:

**Unauthenticated MCP HTTP requests:** B-25 stands unchanged. Any HTTPS URL is acceptable. No allow-list. The request carries no credential — SSRF risk is minimal.

**Authenticated MCP HTTP requests (where `transport.auth` is configured):** No host allow-list is required, because the existing mitigations are structural and sufficient:

1. **Frozen catalog** — child runbooks come only from approved packages. An arbitrary third party cannot inject a tool URL.
2. **Audience-scoped tokens** — a token acquired with `scope: api://icmmcpapi-prod/mcp.tools` carries that audience in its claims. Any properly configured Azure AD resource server that is NOT the intended audience rejects the token server-side. An attacker-chosen host receives a token it cannot validate (wrong audience) and gains nothing useful.
3. **HTTPS** — prevents interception in transit.
4. **Short-lived** — Azure CLI tokens expire in ~1 hour.

The threat model is: a compromised or careless package author places a malicious URL in a child runbook's tool definition. The frozen catalog prevents injection by outsiders. Against an insider (compromised package author), the audience-scoped token provides defense-in-depth — the token is useless at hosts that aren't the declared resource.

**One mandatory safeguard:** The auth provider MUST emit a trace event `mcp/authAttached` with payload `{url_host: <host-portion-only>, scope: <scope>}` (NO token value) on every authenticated request. This enables audit detection of scope-vs-host mismatch without adding a runtime gate.

**Implementation note for David:** Emit this trace event inside `MCPHTTPTransport` immediately before sending an authenticated request. The event is informational (non-gating, non-fatal). If someone declares `scope: api://icmmcpapi-prod/mcp.tools` but `url: https://evil.com`, the trace shows the discrepancy for post-hoc audit.

**Why no allow-list:** An allow-list would require a new configuration surface (where is the list? who maintains it? how does it interact with package catalogs?). The existing mitigations (catalog trust + audience-scoped tokens) close the gap structurally without adding operational burden. If a future threat model shows audience validation is insufficient (e.g., a server that accepts any audience), we can add a host allow-list then — but that would indicate a broken resource server, not a broken gert design.

---

## B-28 — Stdio env-var credentials are a pre-existing surface; B-24 does not retroactively cover them

**Date:** 2026-08-15T18:03:00-07:00

**Question:** Does B-24's redaction mandate extend to stdio MCP's `transport.env` map, which today can carry secrets (e.g., `API_KEY=xxx`) passed to the subprocess via `mergeEnv`?

**Ruling:** **No. B-24 applies exclusively to the new HTTP MCP auth provider.** The stdio env-var path is a pre-existing surface with different characteristics:

1. `transport.env` values are placed in YAML by the operator — they are authored, not runtime-acquired. B-24 exists specifically because the HTTP auth provider acquires a credential at runtime that the operator never typed into any file.

2. Env values are passed to a local subprocess and never traverse a network from gert itself. Whether the subprocess uses them over a network is outside gert's control.

3. These values are NOT currently in traces or logs (only in YAML and process memory). That is acceptable status-quo behavior.

**Action:** Documentation-only. Note that `transport.env` values in `.tool.yaml` should be treated as sensitive by operators (use CI vault injection or env-var indirection rather than literal secrets in committed YAML). No code change. Not a defect — a usage guidance gap.

**Not in scope:** Secret-reference resolution in `transport.env` (e.g., vault references). That is a future platform feature.

---

## B-29 — Protocol version divergence: stdio stays at 2024-11-05; HTTP uses 2025-03-26; this is correct

**Date:** 2026-08-15T18:04:00-07:00

**Question:** Stdio MCP advertises `protocolVersion: "2024-11-05"`. HTTP MCP advertises `MCP-Protocol-Version: 2025-03-26`. Should they match?

**Ruling:** They MUST NOT match. They are different protocol revisions for different transports.

- `2024-11-05` is the MCP version that defines the stdio/Content-Length framing model. The existing `MCPTransport` was built against it and interoperates with stdio MCP servers implementing that version. Changing it risks breaking compatibility with existing servers that validate the version field.
- `2025-03-26` is the MCP version that defines Streamable HTTP (POST + SSE responses, `Mcp-Session-Id` header). HTTP MCP servers expect this version in the `MCP-Protocol-Version` header.

These are not two implementations of the same spec at different versions — they are two different transport specifications. The version field means "I speak this protocol," and the two transports speak different protocols.

**Documentation:** No operator-facing documentation is needed beyond the tool schema itself (`mode: mcp` vs `mode: mcp-http`). An operator never chooses between transports for the same server — a server is either stdio or HTTP. The version is an implementation detail that follows from the transport choice.

---

## B-30 — Stdio `ensureStarted` defect: file and defer

**Date:** 2026-08-15T18:04:00-07:00

**Question:** Stdio `ensureStarted` sets `initialized = true` without checking whether the `initialize` response is an error. Fix now or defer?

**Ruling:** **Defer.** File as a defect. Do NOT fix in this feature.

**Rationale:**
1. This is a pre-existing defect in the stdio path. It has existed since `MCPTransport` was written and affects only stdio MCP tools, which are working in production (presumably with servers that don't reject initialization).
2. Fixing it means modifying `internal/tool/mcp.go` — the existing stdio transport — as part of a feature that is supposed to ADD a new transport without touching the old one. That violates the scope boundary.
3. Don's HTTP transport MUST NOT replicate it (already communicated — his `initialize` path inspects the response and emits MCP-003 on failure).

**Accumulation acknowledgment:** This is the third deferred pre-existing defect alongside B-18 (static-include governance gap) and B-19/B-20 (inert `when:`). I accept this accumulation deliberately. Each has the same shape: a real defect, adjacent to the stream, with regression risk if fixed as a side-effect. They are tracked, not forgotten. When this feature ships, a cleanup pass addressing all three should be prioritized.

**File to:** `.squad/decisions/inbox/defect-stdio-mcp-initialize-unchecked.md` with location `internal/tool/mcp.go:ensureStarted`, the fact that `initialized = true` is set unconditionally, and that the response body is discarded without error checking.

---

## B-31 — Go-only validation is acceptable; author-facing documentation lives in Go doc comments and the tool spec section

**Date:** 2026-08-15T18:04:00-07:00

**Question:** No JSON Schema governs `.tool.yaml`. B-20's schema-description remediation pattern cannot apply here. Is Go-only validation acceptable, and where does documentation live?

**Ruling:** **Go-only validation is acceptable.** This is the existing pattern for ALL tool validation and changing it is out of scope.

The absence of a `.tool.yaml` JSON Schema is a pre-existing architectural choice (noted by Ken: "No JSON Schema governs .tool.yaml" — C3 in the existing code comments). Tool definitions are validated by `internal/tool.ParseToolFile` and the schema types in `pkg/schema/tool.go`. This works and is well-tested.

**Author-facing documentation for `mcp-http` and `auth:` lives in:**
1. **Go doc comments on `TransportConfig`, `AuthConfig`** — these are the source of truth for any future generated documentation.
2. **`design/gert/sections/06-tool-runtime.tex`** — the existing tool runtime spec section. Ken or the implementer of Stream A should add a subsection for `mode: mcp-http` with the transport fields and auth provider shape.
3. **A `.tool.yaml` example** in `examples/` demonstrating the MCP HTTP configuration.

B-20's pattern (schema `description` fields) was appropriate because a JSON Schema existed and IDE completions were actively misleading users. Here, no schema exists, so no misleading completions are generated. The documentation surface is the spec section and examples — standard for this codebase.

---

## B-32 — B-25/B-27 revised: token attachment restricted to declared hosts (replay risk accepted as real)

**Date:** 2026-08-15T18:06:00-07:00

**Supersedes:** B-27's conclusion that "audience-scoped tokens are structurally limited" is **withdrawn as technically incorrect.** The revised ruling below replaces B-25/B-27's reasoning on authenticated requests.

**The corrected threat model:** A bearer token sent to an attacker-controlled host can be **replayed** against the legitimate audience (the real IcM endpoint) for the token's lifetime. Audience restriction constrains which server will *accept* the token, not who may *present* it. Possession is authorization. Token exfiltration to a malicious host IS a compromise regardless of audience scoping — this is the confused-deputy / token-replay pattern.

**Revised Ruling:** Token attachment is restricted to explicitly declared hosts.

### Mechanism

`AuthConfig` gains an `allowed_hosts` field:

```yaml
transport:
  mode: mcp-http
  url: https://icm-mcp-prod.azure-api.net/v1/
  auth:
    provider: azure-cli
    scope: api://icmmcpapi-prod/mcp.tools
    allowed_hosts:
      - icm-mcp-prod.azure-api.net
```

**Enforcement rule:** Before attaching an `Authorization` header, the auth provider checks that the request URL's host (scheme + hostname + port) matches an entry in `auth.allowed_hosts`. If no match, the request is sent **without** the `Authorization` header and a `MCP-W001` warning is emitted (new warning-class code): `"mcp-http: auth token not attached — host %q is not in allowed_hosts"`.

**Validation rules:**
- If `auth` is configured and `allowed_hosts` is empty or omitted → `MCP-010` (fatal): `"mcp-http: auth.allowed_hosts is required when auth is configured"`. Rationale: forcing explicit host declaration is the entire point of this control.
- If `url`'s host is not in `allowed_hosts` at validation time → `MCP-011` (fatal): `"mcp-http: url host %q is not in auth.allowed_hosts"`. Catches the misconfiguration statically rather than at runtime.

### Why this is the right narrowing

1. **Closes the replay vector.** A compromised package author who declares `url: https://evil.com` cannot also make the token go there, because `allowed_hosts` is validated against `url` at parse time (MCP-011). To exfiltrate the token, they would need to control both the URL and the allowed_hosts declaration — but if they can do that, they can also set `url` to the legitimate host and issue malicious tool calls directly, which is a strictly more powerful attack that no allow-list prevents.

2. **Does not burden the unauthenticated case.** `allowed_hosts` is only required when `auth` is configured. An MCP HTTP tool without auth has no token to protect and no restriction.

3. **Small implementation surface.** A host-match check in the HTTP transport before header attachment. David already has a policy hook routed for this.

4. **Prevention, not just detection.** B-27's `mcp/authAttached` trace event remains (for audit), but this is the gate that stops the send.

### Updated target YAML (canonical)

```yaml
transport:
  mode: mcp-http
  url: https://icm-mcp-prod.azure-api.net/v1/
  auth:
    provider: azure-cli
    scope: api://icmmcpapi-prod/mcp.tools
    allowed_hosts:
      - icm-mcp-prod.azure-api.net
```

### Error codes added

| Code | Class | Condition | Fatal? |
|---|---|---|---|
| `MCP-010` | `MCP` | `auth` configured without `allowed_hosts` | Fatal |
| `MCP-011` | `MCP` | `url` host not in `allowed_hosts` | Fatal |
| `MCP-W001` | `MCP-W` | Token not attached because host not in `allowed_hosts` (runtime, if URL is somehow different from validated url — defensive) | Warning |

### For the record

Token replay from a malicious host was **considered and accepted as a real threat**. The original B-25/B-27 reasoning that audience scoping structurally prevents exploitation was incorrect — audience scoping prevents the attacker from *consuming* the token at their own service, but does not prevent *replaying* it against the legitimate service. The `allowed_hosts` restriction is the prevention mechanism. The frozen catalog remains the primary trust boundary (only approved package authors can declare tool URLs), and `allowed_hosts` is defense-in-depth against a compromised author.

---

## B-33 — Runtime host mismatch is FATAL; redirects disabled on authenticated requests

**Date:** 2026-08-15T18:25:00-07:00

### Part 1: Fatal, not warning

**Ruling:** A runtime host mismatch MUST be fatal (`MCP-012`, class `MCP`). The request is NOT sent. The token is NOT attached. Execution stops with a clear error naming the mismatched host and pointing at `allowed_hosts`.

**Rationale:** B-32 exists to prevent token exfiltration. A warning that allows the request to proceed — with or without the token — defeats the control entirely:
- With token attached: the control is decorative (the token reaches the unapproved host).
- Without token attached: the request silently downgrades to unauthenticated, producing a confusing 401 from the far end that no operator will diagnose as a security gate firing.

Fatal is the only semantics that actually prevents the thing B-32 was designed to prevent.

**Ken must reclassify `ErrMCPW001`:** Change from `class: "MCP-W"` / warning to `code: "MCP-012"` / `class: "MCP"` / fatal. Remove `ErrMCPW001` entirely (it was never shipped externally — the frozen-catalog invariant on error codes applies only post-ship). Replace with:

```go
ErrMCP012 = &Error{code: "MCP-012", class: "MCP"}
```

Message: `"mcp-http: request to host %q blocked — not in auth.allowed_hosts %v"`

### Part 2: Redirects disabled on authenticated requests

**Ruling:** The `http.Client` used by `MCPHTTPTransport` for authenticated requests MUST have `CheckRedirect` set to reject ALL redirects (return `http.ErrUseLastResponse`). The request fails with `MCP-013`:

```go
ErrMCP013 = &Error{code: "MCP-013", class: "MCP"}
```

Message: `"mcp-http: redirect from %q to %q rejected — redirects are disabled on authenticated requests"`

**Rationale:** Per-hop host checking is more permissive but complex to get right (must strip the `Authorization` header before the redirect if the target host differs, race between check and send, interaction with HTTP/2 push). Disabling redirects entirely is simpler, correct, and matches the security posture of every major OAuth client library (which strip `Authorization` on cross-origin redirects by default). A legitimate MCP server that requires redirects can be addressed by the operator updating `url:` to point at the final endpoint.

For **unauthenticated** requests (no `auth:` configured), default redirect-following behavior is acceptable — there is no token to protect.

### Part 3: Host-matching semantics confirmed as binding

**Ruling:** The following is the single canonical definition for host matching, used by both MCP-011 (static validation) and MCP-012 (runtime check):

1. Parse `url` with `net/url.Parse`
2. Extract hostname via `u.Hostname()` (strips port)
3. Lowercase both the extracted hostname and each `allowed_hosts` entry
4. **Exact string equality** — no prefix matching, no suffix matching, no wildcards, no regex, no glob

This is the binding definition. If the static and runtime checks use different comparison logic, that is a bug.


# Stream C — Auth Provider (David)

**Status:** Implementation complete. `go build ./...` exit 0. All packages outside `internal/tool` pass. Note: `mcp_http_tess_test.go` has a pre-existing syntax error in `TestMCPHTTPTransport_SSE_CorrectIDSelected` (DEF-012 test missing server setup — untracked file, not caused by Stream C changes); `internal/tool` tests require Tess to fix that before `go test ./internal/tool/...` can pass.

---

## Update — emitter wiring audit + Don collision resolved (2026-08-15T19:12)

**Don's `emit_ctx.go` has not landed** — `internal/tool/emit_ctx.go` does not exist in the repo. My earlier solution (`pkg/trace/emitter.go`) already provides the same functionality and is what `auth_gate.go` currently imports. There is no import cycle.

**B-27 `mcp/authAttached` reaches the real trace stream — provably.**

Full production chain traced:
1. `internal/engine/engine.go:569` — `executor.WithEventEmitter(spanCtx, fn)` installs the emitter; `fn` calls `h.emitEventLocked(spanCtx, kind, payload)`.
2. `executor.WithEventEmitter` delegates to `trace.WithEventEmitter(ctx, e)` (same context key as `trace.EmitterFromContext`).
3. The enriched context is passed to `exec.Execute(emitterCtx, ...)` → `runtime.Invoke(ctx, ...)` → `invokePersistent(ctx, ...)` → `transport.Invoke(ctx, ...)`.
4. `MCPHTTPTransport.send(ctx, ...)` creates `reqCtx` as a child of `ctx` (adds deadline only), passes to `buildHTTPRequest(reqCtx, ...)` → `gate.AttachToken(reqCtx, req)`.
5. `AttachToken` calls `trace.EmitterFromContext(reqCtx)` — same key → receives the engine emitter.
6. `emitEventLocked` writes `trace.TraceEvent{Kind: "mcp/authAttached", Payload: {url_host, scope}}` to `h.engine.cfg.TraceWriter.Append` (the persistent trace file) AND broadcasts in-process.

The emitter is not merely called in tests — it is the engine's production emitter, and it writes to the trace file.

**MCP-013 redirect ownership:** I own it. `CheckRedirect` is configured in `NewMCPHTTPTransport` (my `mcp_http.go` changes); the policy decision (authenticated only, `http.ErrUseLastResponse`) is entirely within my stream. Don's transport receives the configured client.

---

Ken edited `auth_gate.go` as part of the MCP-012 reclassification. On inspection the file was already in the correct state for both changes he described:
- `AttachToken` already returned `errkit.New("MCP-012", ...)` as a fatal error (applied in my previous B-32 pass).
- `ValidateAuthConfig` already used `u.Hostname()` — not `u.Host` — so port-stripping was already correct.

The one genuine divergence was the **MCP-012 error message text**. My message read:
```
mcp-http: host %q is not in auth.allowed_hosts — token not attached; add %q to allowed_hosts or update url:
```
Ken's registered canonical form is:
```
mcp-http: request host %q is not in auth.allowed_hosts — token not attached (update allowed_hosts or url: to match)
```
Updated `auth_gate.go` to match Ken's exact wording. `go build ./...` exit 0; all packages outside `internal/tool` pass.

**Zero `MCP-W` references** anywhere in the codebase — confirmed by grep across all `.go` files.

---

Barbara's final ruling: `CheckRedirect` must return `http.ErrUseLastResponse` (not a custom error) so the redirect response is surfaced to `send()` rather than becoming a transport error wrapped in MCP-008.

**Changes in this pass:**
- `NewMCPHTTPTransport` (`mcp_http.go`): `CheckRedirect` now returns `http.ErrUseLastResponse` for authenticated transports. Unauthenticated transports (gate == nil) retain nil CheckRedirect and follow redirects normally.
- `send()` (`mcp_http.go`): Added 3xx detection block — when gate is non-nil and status is 3xx, emit `errkit.New("MCP-013", "...redirected to <Location>...— update url: to <Location>")` with the `Location` header value. This gives the operator an actionable message naming the destination.
- `TestMCPHTTPTransport_CheckRedirectInstalledOnAuthenticated` (`mcp_http_test.go`): Updated to assert `errors.Is(err, http.ErrUseLastResponse)` instead of `errkit.ErrMCP013`.
- `TestMCPHTTPTransport_TokenNeverReachesRedirectTarget` (new test): Security proof — redirecting server (in `allowed_hosts`) 302s to evil server; asserts evil server never receives any Authorization header AND error is MCP-013.

**errkit.Error.Is() semantics confirmed:** Code-based comparison (`e.code == t.code`), so `errors.Is(errkit.New("MCP-013", msg), errkit.ErrMCP013)` returns true. Don's `TestMCPHTTPTransport_RedirectBlocked_Authenticated` continues to pass without modification.

**Redirect policy stated:**
- Authenticated: redirect → hard stop (`http.ErrUseLastResponse` → `send()` detects 3xx → MCP-013 with Location). Token NEVER forwarded.
- Unauthenticated: follow redirects normally (no restriction).

---

---

## Update — B-32 revised: host-mismatch is now fatal (2026-08-15T18:47)

User instruction superseded the B-32 written text on the non-fatal/warning behavior.
New behavior: host not in `allowed_hosts` → hard MCP-012 error, request not sent.

Changes:
- `TokenGate.AttachToken`: returns `MCP-012` (fatal) on host mismatch instead of nil + mcp/authSkipped event
- `EventKindMCPAuthSkipped` removed from `pkg/trace/event.go` — no longer emitted
- `NewMCPHTTPTransport`: sets `CheckRedirect` to block redirects on authenticated transport (MCP-013); Don had already implemented this with `errkit.ErrMCP013`
- `auth_gate_test.go`: updated `TestTokenGate_RejectsDisallowedHost` (was non-fatal, now asserts MCP-012); added `TestTokenGate_SuffixConfusionRejected`; added `TestTokenGate_MCP012ErrorIsActionable`
- All 38 gate/auth tests pass; `go build ./...` exit 0

**Host-matching semantics (stated explicitly):**
- Match on `req.URL.Hostname()` (port-stripped) — exact case-insensitive
- No wildcards (a `*` entry is treated as a literal hostname and will never match)
- No IDN/punycode normalization
- Port specs in allowed_hosts entries are matched literally (not stripped) — use plain hostnames

**Redirect decision:** authenticated transports refuse all redirects (MCP-013). An operator who needs a redirect path must update `url:` to point to the final endpoint.

---

B-32 superseded B-25/B-27 on token attachment policy. New requirements implemented:

- `schema.AuthConfig.AllowedHosts []string` — required when auth is configured (Ken landed this in `pkg/schema/tool.go`)
- `TokenGate` struct in `internal/tool/auth_gate.go` — the single policy decision point for all token attachment
- `ValidateAuthConfig(rawURL, auth)` — static validation: MCP-010 (no allowed_hosts) and MCP-011 (url host not in list)
- `AttachToken(ctx, req)` — runtime enforcement + B-27 trace event emission
- `EventKindMCPAuthAttached` and `EventKindMCPAuthSkipped` added to `pkg/trace/event.go`
- `trace.EventEmitter` + `trace.WithEventEmitter` + `trace.EmitterFromContext` added to `pkg/trace/emitter.go` (cycle-free emitter sharing — `internal/executor/events.go` updated to delegate there)
- `MCPHTTPTransport` updated: `auth AuthProvider` → `gate *TokenGate`; added HTTP 401 invalidate-and-retry path
- `runtime.go`: validates auth config before construction; constructs `TokenGate` with `AllowedHosts`
- 11 gate tests in `auth_gate_test.go` including B-24 event-payload redaction proof

**Note on B-32 runtime behavior:** Host-mismatch at runtime (after static validation) is **non-fatal** per B-32. The request proceeds unauthenticated; `mcp/authSkipped` event is emitted with `MCP-W001` code. The static MCP-011 check catches the common misconfiguration at parse time.

---

## AuthProvider Interface (for Don — Stream B)

```go
// Package: github.com/ormasoftchile/gert/internal/tool

// AuthProvider acquires bearer tokens for HTTP transports that require authentication.
// Implementations handle acquisition, caching, and refresh internally.
//
// The token returned by Token is an in-memory secret. It must not be written
// to any persistent or observable surface (traces, logs, error messages, step
// output). See B-24.
type AuthProvider interface {
    // Token returns a valid bearer token, refreshing proactively when within
    // five minutes of expiry.
    Token(ctx context.Context) (string, error)

    // Invalidate clears any cached token, forcing re-acquisition on the next
    // Token call. Call after receiving HTTP 401 to handle mid-run token expiry.
    Invalidate()
}

// NewAuthProvider(provider, scope string) (AuthProvider, error)
// Recognized provider values: "azure-cli"
// Returns MCP-002 error for unknown provider.
```

**Don's usage pattern for mid-run 401:**
```go
// On HTTP 401 response:
auth.Invalidate()
token, err = auth.Token(ctx)  // forces re-acquisition
// retry the request with the new token
```

---

## Files Owned

- `internal/tool/auth.go` — `AuthProvider` interface + `NewAuthProvider` factory
- `internal/tool/auth_azurecli.go` — `AzureCLIAuthProvider` implementation
- `internal/tool/auth_azurecli_test.go` — 16 tests including B-24 redaction proof

---

## Caching and Refresh Strategy

- **Proactive refresh:** Token is refreshed when `now + 5min >= expiry`. The old (still-valid) token is returned on probe failure — graceful degradation, not error.
- **Hard expiry:** At the hard expiry point, the cache is invalid and re-acquisition is mandatory.
- **Mid-run 401 path:** `Don.MCPHTTPTransport` should call `Invalidate()` on receiving HTTP 401, then `Token()` again. `AzureCLIAuthProvider` handles the mutex and re-acquisition internally.
- **Expiry parsing:** Prefers `expires_on` (Unix timestamp, TZ-safe) from `az`'s JSON output; falls back to `expiresOn` ("2006-01-02 15:04:05.000000" in local time); falls back to `now + 50min` if both are absent/malformed.
- **Concurrency:** `sync.Mutex` guards the cache; concurrent calls during refresh block rather than stampede.

---

## Four Failure Modes (B-24 / actionable messages)

| Scenario | Error code | Message (summarised) |
|----------|------------|----------------------|
| `az` not on PATH | MCP-007 | `az CLI is not installed or not on PATH — install from https://aka.ms/installazurecli and retry` |
| Not logged in | MCP-007 | `not authenticated — run 'az login' and retry` |
| No scope consent | MCP-007 | `no consent for scope "<scope>" — grant application consent or run 'az login' with the required scope` |
| Malformed JSON output | MCP-007 | `az CLI returned malformed output: <json decode error>` or `missing accessToken field` |

Classification uses `errors.As(err, &exec.Error{})` to detect the not-installed case, then stderr heuristics (`AADSTS` → consent, `Please run 'az login'` → auth) for the others.

---

## B-24 Redaction Proof

Test: `TestAzureCLIAuthProvider_TokenNeverLeaksIntoDiagnostics` in `auth_azurecli_test.go`.

The test seeds the cache with a known token value, then forces all four failure paths and the generic error path. It asserts the known token string is absent from every error message returned. It also asserts that a second cache load doesn't return a stale token after `Invalidate()`.

Redaction guarantees by design:
- `classifyAzError` is only called on command failure — no token has been produced at that point.
- `azTokenResponse` is decoded in-memory and the raw `stdout` bytes are discarded.
- The `token` field on `AzureCLIAuthProvider` is mutex-protected and never serialized, traced, or logged.
- Error messages contain only: stderr text from `az`, the scope string (not a secret), and static operator guidance.

---

## Dependencies Not Yet Complete (Ken — Stream A)

- `errkit.ClassForCode("MCP-007")` returns `""` — Ken has not yet registered the `"MCP"` class in `errkit/errors.go`. Errors work; they just have empty class in the structured error.
- `schema.AuthConfig` type — Ken owns this. When he lands it, `runtime.go` already consumes it at the wire point (`def.Auth.Provider`, `def.Auth.Scope`).

---

## Cross-Stream Seam

Don's `MCPHTTPTransport` (Stream B) receives an `AuthProvider` from `runtime.go`'s switch case for `TransportMCPHTTP`. That wiring is already in place in `internal/tool/runtime.go` — it calls `NewAuthProvider(def.Auth.Provider, def.Auth.Scope)` and passes the result to `NewMCPHTTPTransport(def.URL, auth)`.

Don should call `auth.Token(ctx)` to get the bearer token and set it as `Authorization: Bearer <token>` on each HTTP request. For mid-run 401 handling, call `auth.Invalidate()` then `auth.Token(ctx)` again before the retry.

---

## Test Results

```
go test -count=1 ./internal/tool/... -run "TestAzureCLI|TestNewAuth|TestParseAz"
ok  github.com/ormasoftchile/gert/internal/tool  (all 16 PASS)
```

Full suite: `internal/tool` has pre-existing failures unrelated to Stream C:
- `TestStdioTransport_*` — missing `.testtools` binaries (pre-existing)
- `TestMCPTransport_*` — same
- `TestMCPHTTPTransport_SSEMultiLineData` — Don's in-flight SSE test (not mine)


# Don — Stream B: Streamable HTTP MCP Transport

**Date:** 2026-08-16
**Status:** COMPLETE
**Author:** Don (Backend Dev)
**Feature:** `mode: mcp-http` transport for GERT tool runtime

---

## What Was Built

### New files

| File | Purpose |
|---|---|
| `internal/tool/mcp_http.go` | `MCPHTTPTransport` implementing `pkg/tool.ToolTransport` |
| `internal/tool/mcp_types.go` | Shared `mcpContent`, `mcpCallResult`, `mcpResponse` types (extracted from `mcp.go`); `mcpResponse.Result` is `json.RawMessage` to support both `tools/call` and `tools/list` shapes |
| `internal/tool/mcp_http_test.go` | 10 tests: JSON/SSE response paths, multi-line SSE, session handling, MCP-005, MCP-006 (implied by session expiry), runtime wiring proof |

### Modified files

| File | Change |
|---|---|
| `internal/tool/mcp.go` | Removed duplicate types (now in `mcp_types.go`); updated `Invoke` to use `resp.callResult()` |
| `internal/tool/runtime.go` | Replaced Ken's placeholder `TransportMCPHTTP` case with real `NewMCPHTTPTransport` dispatch |
| `pkg/schema/tool.go` | Removed my accidental duplicate `AuthConfig` type (Ken had already added it); removed duplicate `TransportMCPHTTP` constant |
| `pkg/tool/tool.go` | Removed duplicate `TransportMCPHTTP` constant (Ken had already added it); `URL`/`Auth` fields I added were net-new |

### Streams A and C: already landed

By the time I started implementing, **Ken (Stream A)** had already landed:
- `TransportMCPHTTP` constant in both schema and runtime packages
- `URL`/`Auth` fields on `schema.TransportConfig`  
- `AuthConfig` type in `pkg/schema/tool.go`
- `mapTransport` case for `schema.TransportMCPHTTP`
- `URL`/`Auth` copy in `RuntimeToolDef`
- MCP-001…MCP-009 sentinels in `pkg/errkit/errors.go`
- `validate_transport.go` (MCP-001, MCP-002, MCP-010, MCP-011 validation)

**David (Stream C)** had also already landed:
- `AuthProvider` interface in `internal/tool/auth.go`
- `AzureCLIAuthProvider` in `internal/tool/auth_azurecli.go`

---

## Design Decisions Made

### `json.RawMessage` for `mcpResponse.Result`

The original stdio `MCPTransport` had `Result *mcpCallResult` in `mcpResponse`, which only works for `tools/call` responses. For `tools/list`, the result shape is `{"tools": [...]}`. Changing to `json.RawMessage` lets callers decode into the appropriate shape. Added `callResult()` helper so stdio code doesn't regress.

### No `http404Error` interface assertion in tests

Session-expiry detection uses a private sentinel type `*http404Error` rather than the HTTP 404 status code directly. This avoids accidentally triggering re-init on a 404 that happens for other reasons (e.g. wrong URL path from day one). The re-init-once guard ensures `MCP-004` fires on double failure.

### Token redaction (B-24)

`buildHTTPRequest` sets `Authorization: Bearer <token>` but the token is never returned from any function, never stored in `ToolResult`, and is not included in any error message. `AzureCLIAuthProvider.classifyAzError` explicitly does not include the token (which would not exist at failure time anyway).

---

## Test Coverage

| Test | Proves |
|---|---|
| `TestMCPHTTPTransport_JSONResponse` | Happy path, session ID capture |
| `TestMCPHTTPTransport_SSEResponse` | SSE framing, notification skip |
| `TestMCPHTTPTransport_SSEMultiLineData` | Multi-line SSE event + heartbeat comment skip |
| `TestMCPHTTPTransport_ToolError` | `isError:true` → error, matching stdio shape |
| `TestMCPHTTPTransport_SessionID` | Server omitting session ID — no error |
| `TestMCPHTTPTransport_UnexpectedContentType` | MCP-005 |
| `TestMCPHTTPTransport_ProtocolVersionHeader` | `MCP-Protocol-Version: 2025-03-26` on all requests |
| `TestMCPHTTPTransport_RuntimeWiring` | **WIRING PROOF** — `DefaultToolRuntime` dispatches `mcp-http` to `MCPHTTPTransport` |
| `TestMCPHTTPTransport_SessionExpiry` | 404 → re-init → retry; `initCount == 2` asserted |
| `TestMCPHTTPTransport_ListTools` | `tools/list` response parsing |

---

## Corrections Applied from Ken's Recon (Ken MCP-HTTP Recon)

Ken's recon landed after initial implementation and identified four items. All addressed:

### Correction 1 — ID correlation under SSE (important)

Ken's finding: the stdio MCPTransport has no real JSON-RPC id correlation — it assumes the next message off the pipe is the response. This is survivable on a synchronous pipe but **wrong for SSE**, where a server can interleave notifications between the terminal response.

**Fix applied:** `parseSSEResponse` now takes `expectedID int` and explicitly skips:
- Events with no `id` field (notifications/progress messages)
- Events whose `id` != `expectedID` (stale responses)

Only returns when it finds `id == expectedID`. The `parseJSONResponse` path also verifies the ID. The post-hoc ID check in `doRequest` was removed since correlation now happens at read time. Note: calls are serialized behind `t.mu`, so there's only one outstanding request at a time — but the explicit ID check makes this invariant visible and protects against future concurrency changes.

### Correction 2 — Protocol version divergence (noted, not resolved)

Stdio advertises `2024-11-05`; HTTP transport pins to `2025-03-26` per B-23. This divergence is real and documented. B-23's value is used; I have not resolved the divergence in stdio (out of scope per Ken's recon). Awaiting Barbara's ruling if she wants to address it.

### Correction 3 — Result decoding matches stdio exactly

`toolResult()` uses `callResult()` (decodes `json.RawMessage` → `mcpCallResult`) and follows the identical path: `content[0].text` → `Stdout`; if text parses as JSON → `Output`. `isError:true` → non-zero ExitCode + error. Protocol error → `MCP-009`. This is proven by `TestMCPHTTPTransport_ToolError`.

### Correction 4 — Initialize response inspection (stdio defect NOT replicated)

Ken's finding: stdio `ensureStarted` sets `initialized = true` before checking the initialize response — so a JSON-RPC error on init is silently treated as success.

**Fix applied:** `initialize()` now calls `doRequest()` and explicitly checks `initResp.Error != nil` before marking `t.initialized = true`. A server that returns a JSON-RPC error on initialize causes `Invoke` to fail with "mcp-http: initialize failed: server error N: <message>". Proven by `TestMCPHTTPTransport_InitializeError`.

### Import cycle fix (Ken's auth_gate.go)

Ken's `auth_gate.go` imported `internal/executor` for `executor.EmitterFromContext` (which didn't exist yet), creating a build-breaking import cycle: `internal/tool` → `internal/executor` → `internal/tool`.

**Fix:** Created `internal/tool/emit_ctx.go` defining `EmitFunc`, `WithToolEmitter`, and `ToolEmitterFromContext` in `internal/tool/` itself. Updated `auth_gate.go` to use `ToolEmitterFromContext` (no executor import). The executor can seed the context via `WithToolEmitter` without creating a cycle. This also unblocked `pkg/pjvm` and `internal/conformance` which were failing due to the cycle.

## Test Results (post-corrections)

- `go build ./...` ✅
- `internal/tool`: 11/11 `TestMCPHTTP*` pass; overall pass except `TestStdioTransport_*` (pre-existing known failure)
- `internal/conformance`: ✅ (was failing due to import cycle — now fixed)
- `pkg/pjvm`: ✅ (same fix)
- All other packages: ✅


---

## Session expiry and no-auth paths

- **Server omits `Mcp-Session-Id`**: proceed normally; no `Mcp-Session-Id` header sent on subsequent requests.
- **Server returns HTTP 404 mid-run**: session is cleared, `initialize` is retried once. On double failure: `MCP-004`.
- **No auth configured** (`transport.auth` absent): no `Authorization` header. Compliant with B-24 and §4.5.

---

## Follow-up Round (2026-08-16 — post-corrections)

### Item 1 — emit_ctx.go was dead code

I verified: `auth_gate.go` already imports `pkg/trace` and calls `trace.EmitterFromContext(ctx)` directly (Ken landed that with MCP-012). The `emit_ctx.go` file I created during the import-cycle fix defined a parallel `toolEmitKey{}` context key that nothing in production ever seeded. `WithToolEmitter` had zero callers outside tests. Deleted `emit_ctx.go`. Production audit-trail path is now correct: `engine.go` seeds `trace.WithEventEmitter`, `auth_gate.go` reads it with `trace.EmitterFromContext`.

### Item 2 — End-to-end wiring: two-test proof

`TestMCPHTTPTransport_RuntimeWiring` (already existed) proves `DefaultToolRuntime.Invoke` dispatches through the `TransportMCPHTTP` switch to `invokePersistent` → `NewMCPHTTPTransport`. The gap it leaves: it constructs `toolpkg.TransportMCPHTTP` directly in Go code, not via the `schema → mapTransport` conversion that `ScanDir`/`ParseToolFile` uses.

Added `TestMCPHTTPTransport_SchemaRuntimeWiring`: calls `RuntimeToolDef` on a `schema.ToolDef` with `Transport.Type = schema.TransportMCPHTTP`, asserts the result has `Transport = toolpkg.TransportMCPHTTP` and `URL` correctly propagated, then invokes through `DefaultToolRuntime`. This proves the `schema.TransportMCPHTTP → mapTransport → toolpkg.TransportMCPHTTP → MCPHTTPTransport` path that a `.tool.yaml` file with `mode: mcp-http` takes.

The two tests together prove the full chain from schema constant through runtime wiring.

### Item 3 — MCP-013 redirect blocking (B-30)

Implemented in `NewMCPHTTPTransport`: when `gate != nil`, `httpClient.CheckRedirect` is set to return `fmt.Errorf("%w: ...", errkit.ErrMCP013)`. Unauthenticated transports (gate == nil) have no `CheckRedirect` installed and follow redirects normally.

Note: `CheckRedirect` fires BEFORE `AttachToken` is called for the redirect request, so a redirect from an allowed host to a different host is blocked before the token can be forwarded.

Three tests added:
- `TestMCPHTTPTransport_RedirectBlocked_Authenticated`: live 307 redirect + gate with correct allowed_hosts → `errors.Is(err, errkit.ErrMCP013)` ✓
- `TestMCPHTTPTransport_NoCheckRedirectOnUnauthenticated`: structural check, `gate == nil → httpClient.CheckRedirect == nil` ✓
- `TestMCPHTTPTransport_CheckRedirectInstalledOnAuthenticated`: structural + functional check, installed `CheckRedirect` function unwraps to `ErrMCP013` ✓

Tess's `TestMCPHTTPTransport_AuthenticatedRedirect_Blocked_MCP013` has a hard `t.Skip("BLOCKED-DEF-011: ...")`. The blocking condition is now resolved; Tess should remove the skip.

### Item 4 — Ken's auth_gate.go edits (reconciliation)

Current `auth_gate.go` already has:
- `u.Hostname()` fix (not `u.Host`) in both `AttachToken` and `ValidateAuthConfig` ✓
- `trace.EmitterFromContext` (not `emit_ctx.go` key) ✓
- **MCP-012 fatal** behavior in `AttachToken` for disallowed host ✓ (Ken landed this; it was not yet visible in my earlier read)

My earlier diagnostic that MCP-012 was "not yet landed" was wrong — the file was in intermediate state when I read it. Current state: fully reconciled, no conflicts.

### Test results (follow-up complete)

- `go build ./...` ✅
- `go test ./... -count=1`: all packages pass; no new failures vs. known pre-existing `TestStdioTransport_*` and `internal/serve TestSSE_FilterByRunID`
- `go vet ./...`: only pre-existing two `internal/serve` warnings


# Ken — MCP-HTTP Transport Recon

**Date:** 2026-08-16T00:55:39Z
**Author:** Ken (Backend Dev)
**Purpose:** Ground-truth survey for Barbara's gating architecture contract. No design, no production code.

---

## Q2 FIRST — The Critical Seam

**The `ToolTransport` interface exists and is the clean seam. No extraction needed.**

`pkg/tool/tool.go`:
```go
type ToolTransport interface {
    Invoke(ctx context.Context, def ToolDef, action string, args map[string]any) (*ToolResult, error)
    Close() error
}
```

All four existing transports (`StdioTransport`, `JSONRPCTransport`, `MCPTransport`, `NativeCLITransport`) implement it. A new `MCPHTTPTransport` would implement the same two methods without touching any existing type.

**What is NOT separable:** The `writeMCPMessage` / `readMCPMessage` / `readContentLength` helpers in `mcp.go` are Content-Length framed stdio I/O — they're coupled to `*bufio.Reader`, `*bufio.Writer`, and `*ProcessHandle`. They are NOT reusable for HTTP. The HTTP transport rewrites the transport layer entirely but inherits the same interface contract and the same result-decoding logic.

**What can be shared:** The response shape types (`mcpContent`, `mcpCallResult`, `mcpResponse`, `jsonrpcError`) are small and could be moved to a new `internal/tool/mcptypes.go` for sharing, or simply redeclared in the HTTP transport file. They contain no I/O logic.

**ID correlation note:** The current `MCPTransport` does NOT implement request-ID correlation. It serializes all calls behind `t.mu` (sync.Mutex) and reads the next message, assuming it is the response to the pending request. For Streamable HTTP MCP, this assumption does NOT hold if the server can deliver SSE notifications between response frames. The HTTP transport needs real ID correlation (a `map[int]chan jsonrpcResponse` or equivalent). This is a new implementation requirement, not a port of existing code.

---

## 1. The Stdio MCP Transport as It Exists

**File:** `internal/tool/mcp.go`

**Struct:**
```go
type MCPTransport struct {
    mu          sync.Mutex
    proc        *ProcessHandle
    reader      *bufio.Reader
    writer      *bufio.Writer
    nextID      int
    initialized bool
}
```

**Entry points:**
- `Invoke(ctx, def, action, args) (*ToolResult, error)` — holds `t.mu` for the entire call
- `ListTools(ctx, def) ([]map[string]any, error)` — same pattern
- `Close() error` — sends `shutdown` request, calls `proc.Kill()`
- `ensureStarted(ctx, def) error` — private; called at start of Invoke/ListTools

**Lifecycle (ensureStarted → Invoke → Close):**

1. `ensureStarted`: calls `StartProcess(ctx, def.Command, def.Args, def.Env)` → `*ProcessHandle` (process is live); creates `bufio.Reader`/`Writer` wrapping process stdio; sends `initialize` request with `protocolVersion: "2024-11-05"`, reads ONE response (no ID check on result); sends `notifications/initialized` notification (no ID, no response expected); sets `t.initialized = true`.

2. `Invoke` / `ListTools`: write request via `writeMCPMessage` (Content-Length framed), flush, call `readMCPMessage` (reads Content-Length header + body), unmarshal response, decode result.

3. `Close`: sends `shutdown` request, flushes, kills process.

**Wire framing:** LSP-style Content-Length headers — `Content-Length: N\r\n\r\n{body}`. Implemented by:
- `writeMCPMessage(w *bufio.Writer, payload any)` — marshals to JSON, writes header + body
- `readMCPMessage(ctx, r *bufio.Reader, proc *ProcessHandle) ([]byte, error)` — reads header via `readContentLength`, then `io.ReadFull` for body; on `ctx.Done()` kills the process
- `readContentLength(r *bufio.Reader) (int, error)` — reads lines until blank line, parses Content-Length

**`initialize` handling:** Done in `ensureStarted`. The response is read and discarded (no field inspection). `notifications/initialized` is sent immediately after (no response read because it's a notification). Neither server-advertised capabilities nor server info are inspected.

**Defect noted (not fixing):** `ensureStarted` sets `t.initialized = true` before checking whether the initialize response is an error object. A server that replies with `{"jsonrpc":"2.0","id":0,"error":{"code":-32603,"message":"..."}}` to `initialize` will be silently treated as initialized.

---

## 3. `tools/list` and `tools/call` Dispatch

**Dispatch path:**

1. `internal/executor/tool.go` (not surveyed; calls `ToolRuntime.Invoke`)
2. `DefaultToolRuntime.Invoke` (`internal/tool/runtime.go`) looks up `def.Transport`:
   ```go
   case toolpkg.TransportMCP:
       return r.invokePersistent(ctx, toolName, *def, action, args, func() toolpkg.ToolTransport {
           return &MCPTransport{}
       })
   ```
3. `invokePersistent` caches the transport by tool name in `r.persistent map[string]toolpkg.ToolTransport` (created once, reused for all calls to the same tool).

**Result decoding** (`mcp.go::Invoke`):
```go
text := resp.Result.Content[0].Text
result := &toolpkg.ToolResult{ExitCode: 0, Stdout: text}
var parsed map[string]any
if json.Unmarshal([]byte(text), &parsed) == nil {
    result.Output = parsed
}
```
- Text content → `ToolResult.Stdout`
- If text parses as JSON object → also populates `ToolResult.Output`
- `resp.Result.IsError == true` → `fmt.Errorf("mcp tool error: %s", content[0].Text)` — returned as Go error
- `resp.Error != nil` (JSON-RPC protocol error) → `fmt.Errorf("mcp error %d: %s", code, message)`

**`ListTools`** returns `[]map[string]any` (raw tool descriptors from `result.tools`). No type mapping.

**Typed output preservation:** Only `Output map[string]any` on `ToolResult`. Works when the server emits JSON in the content text. No typed schema enforcement at the transport layer.

---

## 4. Transport Schema and Validation

**Schema type:** `pkg/schema/tool.go`:
```go
type TransportConfig struct {
    Type    Transport         `yaml:"type,omitempty"    json:"type"`
    Mode    string            `yaml:"mode,omitempty"    json:"mode,omitempty"`
    Command string            `yaml:"command,omitempty" json:"command,omitempty"`
    Args    []string          `yaml:"args,omitempty"    json:"args,omitempty"`
    Env     map[string]string `yaml:"env,omitempty"     json:"env,omitempty"`
}

type Transport string

const (
    TransportStdio   Transport = "stdio"
    TransportJSONRPC Transport = "jsonrpc"   // note: runtime constant is "stdio-jsonrpc"
    TransportMCP     Transport = "mcp"
    TransportNative  Transport = "native"
)
```

`UnmarshalYAML` on `TransportConfig` copies `Mode` into `Type` when `Mode != ""`, so `transport.mode: mcp` is equivalent to `transport.type: mcp` after parsing. The `Mode` field is retained verbatim.

**No JSON Schema governs tool files.** From `scan.go` comment: "No JSON Schema governs .tool.yaml (C3, no tool.v1.schema.json)". There is exactly one JSON Schema file in `schemas/`: `runbook.schema.json`. There is no `tool.schema.json`. Validation is entirely Go-side.

**No `oneOf` discrimination on mode.** All mode-specific fields (`Command`, `Args`, `Env`) are in a flat struct with no guards. Adding `url` and `auth` fields follows the same flat pattern — there is no JSON Schema to update for a oneOf constraint.

**Validation point at runtime:** `internal/tool/scan.go::mapTransport`:
```go
func mapTransport(t schema.Transport) (toolpkg.TransportType, error) {
    switch t {
    case schema.TransportStdio:   return toolpkg.TransportStdio, nil
    case schema.TransportJSONRPC: return toolpkg.TransportJSONRPC, nil
    case schema.TransportMCP:     return toolpkg.TransportMCP, nil
    case schema.TransportNative:  return toolpkg.TransportNative, nil
    default:
        return "", fmt.Errorf("unsupported transport %q", t)
    }
}
```
A `.tool.yaml` with `mode: mcp-http` today fails here with `unsupported transport "mcp-http"`.

**Second validation point:** `DefaultToolRuntime.Invoke` in `runtime.go` has a parallel switch on `def.Transport` with `default: return nil, fmt.Errorf("tool runtime: unsupported transport %q", ...)`. Both must be extended for `mcp-http`.

**`ToolDef` in `pkg/tool/tool.go`** carries only `Command string`, `Args []string`, `Env map[string]string` from the transport config. There are NO `URL` or `Auth` fields anywhere in `ToolDef`. These must be added to both `schema.TransportConfig` and `toolpkg.ToolDef`, and threaded through `RuntimeToolDef` in `scan.go`.

---

## 5. Existing HTTP and Auth Machinery

**HTTP server (not client):** `internal/serve/` is entirely server-side. `net/http.Server`, `http.Handler`, `http.ServeMux`. Not reusable for an outbound HTTP MCP client.

**SSE in `internal/serve/sse.go`:** Server-side SSE writer:
- `writeSSE(w http.ResponseWriter, ev RunEvent)` — writes `event:/id:/data:` lines
- `writeSSEHeartbeat(w http.ResponseWriter)` — writes `: hb\n\n` comment frame

This is **server-side only**. There is **no SSE client reader** anywhere in the codebase. The HTTP transport must implement SSE parsing from scratch: read `data:` prefixed lines, strip prefix, JSON-unmarshal the payload.

**HTTP client code:** There is NO reusable outbound HTTP client wrapper anywhere. The only HTTP client usage is in tests (`internal/serve/*_test.go`) via `httptest.NewServer` + `http.Get`/`http.Post`. Production code has no `http.Client`, no retry policy, no timeout helper.

**Webhook receiver** (`internal/eventbus/webhook.go`): HTTP *server* receiving inbound POSTs with HMAC-SHA256 signature verification. Not reusable for client.

**Auth / credentials / token machinery:** There is NO bearer token handling, no Azure CLI (`az`) invocation, no credential provider interface for HTTP, no `Authorization` header construction anywhere in the codebase. The input provider chain (`internal/input/`) handles env vars and Vault for runbook vars — it does not serve as an HTTP auth mechanism. All auth for `mcp-http` must be built from scratch.

**Azure CLI specifically:** Zero occurrences of `az`, `azure`, `AzureCLI`, or `azure-cli` anywhere in the repo. An `azure-cli` auth provider would exec `az account get-access-token --scope <scope>` and parse the JSON output. That is new infrastructure.

**Redaction helpers:** `pkg/governance/` has `RedactionPattern` for output scrubbing. There is no HTTP header redaction. Bearer tokens in `Authorization` headers sent in trace/logs must be explicitly sanitized — no existing helper does this.

---

## 6. Error Taxonomy

**Conformance error classes in `pkg/errkit/errors.go`:**
- `GXL-PARSE`, `GXL-TYPE`, `GXL-PATH`, `GXL-EVAL` — expression language
- `GIS-PARSE`, `GIS-PATH`, `GIS-TYPE`, `GIS-EVAL` — string interpolation
- `GCP-PARSE`, `GCP-RESOLVE`, `GCP-DEFAULT`, `GCP-TYPE`, `GCP-EVAL` — container platform
- `PKG`, `PKG-W` — package system
- `PLAN` — planner
- `ENUM`, `ENUM-W` — enum validation
- `DINC`, `DINC-W` — dynamic includes

**Tool / MCP transport errors: none exist.** All errors from `MCPTransport`, `StdioTransport`, etc. are plain `fmt.Errorf` strings with no conformance code or class. There is no `TOOL-*` or `MCP-*` family in the errkit registry.

**Proposed class for new errors:** `MCPH` (MCP over HTTP) would be consistent with the existing naming pattern — short, subsystem-prefixed, all-caps. `ClassForCode` in `errors.go` would need a new case for `strings.HasPrefix(code, "MCPH-")`. Barbara should confirm the name; I'm reporting the gap.

**`IsWarning` pattern:** The generalized `strings.HasSuffix(class, "-W")` convention (from my DINC work) means a `MCPH-W` class would be treated as a warning automatically without changing `IsWarning`.

---

## 7. Tool Registry and Wiring

**End-to-end path for a new `mcp-http` tool:**

1. `ScanDir` → `ParseToolFile` → `yaml.Unmarshal` into `schema.ToolDef` with `TransportConfig.Mode = "mcp-http"` → `TransportConfig.UnmarshalYAML` copies `Mode` → `Type = "mcp-http"`
2. `RuntimeToolDef` → calls `mapTransport("mcp-http")` → **FAILS** today with "unsupported transport"
3. If `mapTransport` is extended: returns new `toolpkg.TransportMCPHTTP` constant; `RuntimeToolDef` must also copy `URL` and `Auth` from `schema.TransportConfig` to `toolpkg.ToolDef`
4. `buildToolRegistry` in `adapter/wire.go` → wraps in `OverlayRegistry` — no changes needed here
5. `NewDefaultToolRuntime(registry)` — no changes needed
6. Tool step executor → `DefaultToolRuntime.Invoke` → switch on `def.Transport` → **FALLS TO DEFAULT** today; must add `case toolpkg.TransportMCPHTTP:` returning `r.invokePersistent(..., func() ToolTransport { return &MCPHTTPTransport{url: def.URL} })`

**Dead-code risk:** Two hard-error defaults guard the dispatch path. Missing either case (`mapTransport` or `Invoke` switch) produces a runtime error at first use — same failure mode as a typo in the transport name. Not a silent failure, but still a gap that must be caught by tests.

**`ToolDef` in `pkg/tool/tool.go` needs new fields:**
```go
// NEW — needed for mcp-http
URL  string       // transport.url
Auth *AuthConfig  // transport.auth (new type TBD)
```
These must be populated in `RuntimeToolDef` (scan.go) and threaded into `MCPHTTPTransport`.

**`ListTools` is NOT part of `ToolTransport` interface.** Only `MCPTransport` and the new HTTP transport need it. `DefaultToolRuntime` does not expose it. If Barbara's design needs dynamic tool discovery at runtime, a second interface or a separate method needs to be defined. Currently `ListTools` is called from nowhere in the production dispatch path — it exists only in `mcp_test.go` and the `MCPTransport` struct.

---

## 8. Test Infrastructure

**Existing infrastructure for stdio MCP:**
- `TestMain` in `internal/tool/test_main_test.go`: builds real Go binaries from `cmd/tools/` into a temp `.testtools/` dir, sets `PATH` and `GERT_TOOLS_DIR`, runs the test suite, cleans up.
- `cmd/tools/mcp-server/main.go`: a complete content-length-framed stdio MCP server fixture. Handles `initialize`, `notifications/initialized`, `tools/list`, `tools/call` (echo/fail/slow), `shutdown`.
- Tests in `internal/tool/mcp_test.go`: 4 tests, all using the compiled `mcp-server` binary.

**What is NOT there for HTTP testing:**
- No `httptest.NewServer` MCP fixture. One needs to be written.
- No SSE client reader (needed by both the production transport and any response-reading test helper).
- No HTTP server fixture in `cmd/tools/` directory.

**What an HTTP MCP test server needs:**
1. An `httptest.NewServer` that accepts POST at a fixed path
2. Handles `initialize`: reads JSON-RPC body, returns `{"result":{"protocolVersion":"...","serverInfo":{...}}}` with `Mcp-Session-Id: <uuid>` response header
3. Handles `notifications/initialized`: returns 202 Accepted (no body required per spec)
4. Handles `tools/list` and `tools/call`: may return either a direct JSON response or an SSE stream starting with `data: {jsonrpc response}\n\n`
5. Session state: validates that subsequent requests carry the `Mcp-Session-Id` header from step 2

**Recommended test structure:**
- `internal/tool/mcp_http_test.go` using `httptest.NewServer`
- A `fakeHTTPMCPServer` struct (analogous to `cmd/tools/mcp-server/main.go` but as an `http.Handler`)
- No separate binary needed — in-process httptest is sufficient and faster

**SSE parsing for production:**
The production transport must parse SSE lines from the response body. The spec format:
```
data: {"jsonrpc":"2.0","id":1,"result":{...}}\n\n
```
This is a net-new implementation. `bufio.NewScanner` on the response body, scan for `data:` prefix, strip prefix, JSON-unmarshal. Handle `: ` comment lines (heartbeats) by skipping. No existing helper.

---

## Summary of Gaps Barbara Must Design Against

| Gap | Severity | Location |
|-----|----------|----------|
| `TransportConfig` has no `url` or `auth` fields | Blocking | `pkg/schema/tool.go` |
| `toolpkg.ToolDef` has no `URL` or `Auth` fields | Blocking | `pkg/tool/tool.go` |
| `mapTransport` doesn't handle `mcp-http` | Blocking | `internal/tool/scan.go` |
| `DefaultToolRuntime.Invoke` doesn't handle `mcp-http` | Blocking | `internal/tool/runtime.go` |
| No HTTP client wrapper (no retry, no timeout) | Must build | new |
| No SSE client reader | Must build | new |
| No Azure CLI token provider | Must build | new |
| No bearer-token redaction for traces/logs | Must build | new (or governance extension) |
| No HTTP MCP test server fixture | Must build | `internal/tool/` tests |
| No errkit class for HTTP MCP errors | Barbara decides | `pkg/errkit/errors.go` |
| No ID correlation in MCPTransport (in-scope for HTTP) | Must implement | new MCPHTTPTransport only |
| `ListTools` not in `ToolTransport` interface; needed for mcp-http? | Barbara decides | `pkg/tool/tool.go` |


# Ken — Stream A: Schema + Validation Report

**Date:** 2026-08-16  
**Feature:** Native Streamable HTTP MCP transport (`mcp-http`)  
**Status:** COMPLETE — all 17 tests pass, `go build ./...` exit 0

---

## AuthConfig shape and extensibility

```go
// pkg/schema/tool.go
type AuthConfig struct {
    Provider string `yaml:"provider"`
    Scope    string `yaml:"scope,omitempty"`
}
```

**Extensibility story:** `Provider` is a plain string — no Go enum, no `oneOf` in YAML. Adding a second provider (e.g., `managed-identity`) requires:
1. Add its name to `knownAuthProviders` in `internal/tool/validate_transport.go` — one line.
2. Implement the `AuthProvider` interface in `internal/tool/auth_<provider>.go`.
3. Add a `case` in `NewAuthProvider` (David's file).

The `AuthConfig` struct itself is unchanged — no breaking change to `.tool.yaml` syntax or any schema type.

B-24 compliance: `auth:` names a *provider* and a *scope* only. There is no field that can hold a credential, token, or secret. Structurally impossible to inline credential material.

---

## Validation rules and error messages

All rules enforced in `internal/tool/validate_transport.go::ValidateTransportConfig`, called from `ParseToolFile` at scan time. Errors are `%w`-wrapped with the file path, so `errors.Is(err, errkit.ErrMCP001)` works through the chain.

| Rule | Condition | Sentinel | Error message (operator-actionable) |
|------|-----------|----------|--------------------------------------|
| mcp-http requires url | `url` absent | ErrMCP001 | `mcp-http: transport.url is required` |
| url must be HTTPS | `url` doesn't start with `https://` | ErrMCP001 | `mcp-http: url "<url>" must use https://` |
| command rejected on mcp-http | `command` non-empty on mcp-http | plain | `mcp-http: transport.command is not valid for mode mcp-http (mode mcp-http uses url:, not command:)` |
| args rejected on mcp-http | `args` non-empty on mcp-http | plain | `mcp-http: transport.args is not valid for mode mcp-http (mode mcp-http uses url:, not command/args)` |
| auth.provider must be known | provider not in `knownAuthProviders` | ErrMCP002 | `mcp-http: unknown auth provider "<name>" (recognized providers: azure-cli)` |
| mcp requires command | `command` empty on mcp | plain | `mcp: transport.command is required for mode mcp` |
| url rejected on mcp | `url` non-empty on mcp | plain | `mcp: transport.url is not valid for mode mcp (mode mcp uses command:, not url:)` |
| auth rejected on mcp | `auth` non-nil on mcp | plain | `mcp: transport.auth is not valid for mode mcp (auth is only used by mode mcp-http)` |

Following Barbara's B-14 principle: cross-arm fields are **rejected, not silently ignored**. A tool author who writes `url:` under `mode: mcp` gets an error telling them exactly what's wrong and what to use instead.

---

## Switch/discrimination points on transport mode

All enumerated to confirm no dead-code miss (recon finding: hard-error defaults).

| File | Location | `mcp-http` handled? |
|------|----------|---------------------|
| `internal/tool/scan.go` | `mapTransport` switch | ✅ `case schema.TransportMCPHTTP: return toolpkg.TransportMCPHTTP, nil` |
| `internal/tool/runtime.go` | `DefaultToolRuntime.Invoke` switch | ✅ stub returning "not yet wired — Stream B pending" |
| `internal/tool/validate_transport.go` | `ValidateTransportConfig` switch | ✅ primary enforcement |

The stub in `runtime.go` surfaces loudly at call time (returns an error), not silently. Don replaces it with `NewMCPHTTPTransport` when Stream B lands.

---

## B-32 Addendum: allowed_hosts

**Date:** 2026-08-15 (B-32 ruling incorporated)

Barbara reversed B-25/B-27. Audience-scoped tokens prevent a malicious host from consuming a token, but not from replaying it against the legitimate endpoint. `allowed_hosts` is the prevention mechanism.

### AuthConfig shape (revised)

```go
type AuthConfig struct {
    Provider     string   `yaml:"provider"`
    Scope        string   `yaml:"scope,omitempty"`
    AllowedHosts []string `yaml:"allowed_hosts,omitempty"`
}
```

`AllowedHosts` is required whenever `auth:` is present. Entries are bare hostnames (no scheme, no port). Matching is exact, case-insensitive, parsed host — no substring or wildcard. This prevents `icm.evil.com` from matching `icm.com` and `icm-mcp.azure-api.net.evil.com` from matching `icm-mcp.azure-api.net`. **David must use identical semantics** in the runtime host check before header attachment.

### New validation rules

| Rule | Condition | Sentinel | Error message |
|------|-----------|----------|----------------|
| allowed_hosts required when auth configured | `auth != nil && len(AllowedHosts) == 0` | ErrMCP010 | `mcp-http: auth.allowed_hosts is required when auth is configured (B-32: …)` |
| url host must be in allowed_hosts | url host not in AllowedHosts (static check) | ErrMCP011 | `mcp-http: url host "<host>" is not in auth.allowed_hosts [...]` |

No `allowed_hosts` requirement when `auth:` is absent — unauthenticated transports are unconstrained (B-32 explicitly preserves this).

### MCP-012 reclassification (B-32 revised ruling)

**Date:** 2026-08-15

`ErrMCPW001` / `MCP-W001` removed. Reclassified as `ErrMCP012` / `MCP-012`, class `MCP`, fatal.

- `MCP-W` class removed entirely from errkit (`ClassForCode`, `Classes()`, `classSentinels`, `codeOrder`).
- `ErrMCP013` added: redirect blocked on authenticated request — `http.ErrUseLastResponse`; David raises this in the HTTP transport.
- `auth_gate.go`: `AttachToken` now returns `errkit.New("MCP-012", ...)` on host mismatch instead of emitting a warning and proceeding. `u.Host` → `u.Hostname()` fixed in both `AttachToken` and `ValidateAuthConfig` to match binding semantics.
- `pkg/schema/tool.go` doc comment updated.
- Repo-wide sweep confirmed: no remaining references to `MCP-W001`, `ErrMCPW001`, or `MCP-W` class.

**MCP-012 message:** `mcp-http: request host "<host>" is not in auth.allowed_hosts — token not attached (update allowed_hosts or url: to match)`

All 22 tests pass. `go build ./...` and `pkg/errkit` tests clean.


### New/updated tests (5 added, 2 updated)

- `TestValidateTransport_MCPHTTP_ValidWithAuth` — updated with `AllowedHosts`
- `TestValidateTransport_MCPHTTP_MissingAllowedHosts_MCP010` — NEW
- `TestValidateTransport_MCPHTTP_HostNotInAllowedHosts_MCP011` — NEW
- `TestValidateTransport_MCPHTTP_SubstringHostDoesNotMatch_MCP011` — NEW (replay-attack guard)
- `TestValidateTransport_MCPHTTP_NoAuth_AllowedHostsNotRequired` — NEW
- `TestParseToolFile_MCPHTTP_ValidWithAuth` — updated with `allowed_hosts:` in YAML
- `TestParseToolFile_MCPHTTP_UnknownProvider_Fails` — updated with `allowed_hosts:` to isolate MCP-002

All 22 tests pass. `go build ./...` and `pkg/errkit` tests clean.

### Host-matching semantics (for David to match)

1. Parse `cfg.URL` with `net/url.Parse` — use `u.Hostname()` (strips port if present)
2. Lowercase both the parsed hostname and each `AllowedHosts` entry
3. Exact string equality only — no `strings.HasPrefix`, no `strings.Contains`, no wildcard expansion
4. Port in `AllowedHosts` entries is not supported in this iteration; entries must be bare hostnames

If a future ruling adds wildcard support (e.g. `*.azure-api.net`), only `hostInList` in `validate_transport.go` and the parallel runtime check need updating — the schema and `AuthConfig` struct are unchanged.


- **`pkg/schema/tool.go`** — `TransportMCPHTTP` constant; `URL string` + `Auth *AuthConfig` on `TransportConfig`; `AuthConfig` struct
- **`pkg/tool/tool.go`** — `TransportMCPHTTP TransportType = "mcp-http"` constant
- **`pkg/errkit/errors.go`** — `MCP` class; `ErrMCP001`..`ErrMCP009` sentinels; `ClassForCode`, `codeOrder`, `classSentinels`, `sentinels`, `Classes()` all updated
- **`internal/tool/scan.go`** — `mapTransport` case; `URL`/`Auth` threading in `RuntimeToolDef`; `ValidateTransportConfig` call in `ParseToolFile`
- **`internal/tool/runtime.go`** — stub `case toolpkg.TransportMCPHTTP`
- **`internal/tool/validate_transport.go`** (new) — full validation logic
- **`internal/tool/validate_transport_test.go`** (new) — 17 tests (11 unit + 5 integration through `ParseToolFile` + 1 `toolFileTemplate` const)

---

## Test results

```
=== 17 new tests: ALL PASS ===
TestValidateTransport_MCPHTTP_ValidMinimal          PASS
TestValidateTransport_MCPHTTP_ValidWithAuth         PASS
TestValidateTransport_MCPHTTP_MissingURL_MCP001     PASS
TestValidateTransport_MCPHTTP_HTTPNotHTTPS_MCP001   PASS
TestValidateTransport_MCPHTTP_CommandRejected       PASS
TestValidateTransport_MCPHTTP_ArgsRejected          PASS
TestValidateTransport_MCPHTTP_UnknownProvider_MCP002 PASS
TestValidateTransport_MCPStdio_ValidMinimal         PASS
TestValidateTransport_MCPStdio_MissingCommand       PASS
TestValidateTransport_MCPStdio_URLRejected          PASS
TestValidateTransport_MCPStdio_AuthRejected         PASS
TestParseToolFile_MCPHTTP_ValidMinimal              PASS
TestParseToolFile_MCPHTTP_ValidWithAuth             PASS
TestParseToolFile_MCPHTTP_MissingURL_Fails          PASS
TestParseToolFile_MCPHTTP_HTTPScheme_Fails          PASS
TestParseToolFile_MCPHTTP_UnknownProvider_Fails     PASS
```

Pre-existing failures (not mine):
- `internal/tool TestStdioTransport_*` — missing `.testtools/internal-tool/json-emitter.exe` (build artifact)
- No `internal/serve` failures (green)
- Two `go vet` warnings in `internal/serve` (pre-existing, not mine)

`go build ./...` exit 0.


# Tess — MCP HTTP Stream D Final Decision Report
**Date:** 2026-08-16T02:29:31Z  
**Requested by:** Cristián  
**Context:** Stream C (David's AzureCLIAuthProvider) — final adversarial pass  
**Status:** CLOSED

---

## Team-relevant finding: David's redaction sentinel is weak

`auth_azurecli_test.go` defines `knownToken = "******"`. Six asterisks can appear in truncated Go error formatting (e.g., `"...got ******: ..."`) or in log output using redaction markers. A test asserting `!strings.Contains(msg, "******")` can pass even when the real token is present, as long as no six-asterisk run appears in the message.

**This is a gap in David's redaction test, not a gap in production code.** The production redaction behavior is correct — `AzureCLIAuthProvider` never passes the raw token to any error constructor, only the classified failure reason. The weakness is that the test would not catch a future regression where a developer accidentally logs the token, because the sentinel would not collide with normal output in the same way a high-entropy value would.

**Recommendation:** Change `knownToken` in `auth_azurecli_test.go` to a high-entropy sentinel like `TESS_SENTINEL_TOKEN_D42E9B1C`. Tess's `TestAzureCLIProvider_SentinelNeverInAnyErrorOutput` already uses this value and covers all five error paths independently. The two test files are complementary, but David's should be strengthened.

**Action:** Barbara or Cristián to decide whether to ask David to update `knownToken` before the next commit, or accept the complementary coverage as-is.

---

## AUTH-003 tightening — confirmed safe

`TestMCPHTTPTransport_SuffixConfusion_TokenNotAttached` now asserts `err != nil` unconditionally (was conditional). The test passes after tightening because DEF-013 is genuinely fixed: `AttachToken` returns fatal `MCP-012` on host mismatch. If someone regresses it back to warn-and-continue, the test will now fail immediately.

---

## Final counts

- **Stream D vectors:** 27 PASS / 0 SKIP / 0 FAIL  
- **Group I (AzureCLI):** 4 PASS / 0 SKIP / 0 FAIL  
- **Total Tess vectors:** 31 PASS / 0 SKIP / 0 FAIL  
- **Full suite:** 62 packages, `go test ./... -count=1` exit 0  


# Tess — MCP HTTP Stream D Report
**Date:** 2026-08-16 (updated after stale-defect correction pass)  
**Requested by:** Cristián  
**Feature:** Native Streamable HTTP MCP transport (`mode: mcp-http`)  
**Test file:** `internal/tool/mcp_http_tess_test.go`

---

## Lead verdicts (required)

### Redirect-with-token (TV-MCP-AUTH-002) — **PASS. DEF-011 was stale — B-33 Part 2 IS implemented.**

`NewMCPHTTPTransport` installs `CheckRedirect` when `gate != nil`, returning `ErrMCP013`. A 302 from an allowed host is refused before the redirect fires. `TestMCPHTTPTransport_AuthenticatedRedirect_Blocked_MCP013` **PASSES**: `Invoke` returns an MCP-013 error, and the evil server confirms it received zero requests — the `Authorization` header never reached the redirect destination. The initial defect report (DEF-011) was filed against mid-flight code; the fix had landed before tests ran.

### Interleaved-notification SSE (TV-MCP-SSE-001) — **PASS. Both notification skip and ID correlation correct.**

`parseSSEResponse` skips nil-id events (notifications) AND checks `resp.ID == expectedID` — both guards are present. `TestMCPHTTPTransport_SSE_NotificationBeforeResponse` confirms notification skip. `TestMCPHTTPTransport_SSE_CorrectIDSelected` confirms that a stale response (id=99) before the matching response is skipped and the correct answer is returned. Both **PASS**. The initial DEF-012 report was also stale.

---

## End-to-end token-leak sweep (B-24 / B-27)

Scope: all observable surfaces through a complete authenticated `Invoke` call — error messages, `ToolResult` fields, and trace event payloads. Sentinel: `TESS_SENTINEL_TOKEN_D42E9B1C`.

Methodology in `TestMCPHTTPTransport_TokenRedaction_NotInAnyOutput`:
1. A real `TokenGate` with a `mockAuthProvider` returning the sentinel token
2. The server echoes the `Authorization` header back as the tool result text (to verify the token IS reaching the server — baseline confirmation)
3. A `trace.WithEventEmitter` captures all emitted events for the call
4. Sweep: error messages, `result.Stderr`, non-stdout `result.Output` fields, all trace event payloads (JSON-serialized)

**Result: PASS. No leak found.**
- `result.Stderr` is empty — not leaked.
- No non-stdout `result.Output` field contains the sentinel.
- `mcp/authAttached` event was emitted (B-27 confirmed) with payload `{url_host, scope}` — no token in the serialized payload.
- Auth provider failure errors (`TestMCPHTTPTransport_AuthFailure_ErrorHasNoToken`) contain only the MCP-007 failure reason, not the sentinel. **PASS.**

---

## B-27 emitter wiring verdict (requested)

`mcp/authAttached` **IS wired in production.** Evidence:

`internal/engine/engine.go` calls `executor.WithEventEmitter(spanCtx, func(...) { h.emitEventLocked(...) })` before every `exec.Execute` call. This `emitterCtx` is passed as `ctx` to the executor, which calls `runtime.Invoke(ctx, ...)`, which flows through `MCPHTTPTransport.Invoke(ctx, ...)` → `buildHTTPRequest(ctx, ...)` → `gate.AttachToken(ctx, req)`. `AttachToken` calls `trace.EmitterFromContext(ctx)` which finds the engine-installed emitter. The event is then routed through `emitEventLocked` → `trace.TraceEvent` → persisted in the run store.

`TestMCPHTTPTransport_TokenRedaction_NotInAnyOutput` proves this end-to-end: when a real emitter is installed in ctx, `mcp/authAttached` fires and is captured. Without an emitter in ctx, `EmitterFromContext` returns nil (safe no-op per the emitter contract). There is no gap — the engine always provides the emitter.

---

## Defects found (summary — all resolved or stale)

| DEF | What | Status |
|-----|------|--------|
| DEF-007 | Import cycle `auth_gate.go` → `internal/executor` | RESOLVED before tests ran |
| DEF-008 / DEF-009 / DEF-010 | Compile failures (`t.auth`, arg count, type mismatch) | RESOLVED by Don before tests ran |
| DEF-011 | No `CheckRedirect` on `httpClient` | STALE — fix was already in HEAD |
| DEF-012 | `parseSSEResponse` no `id == expectedID` check | STALE — fix was already in HEAD |
| DEF-013 | `AttachToken` returns nil on host mismatch | STALE — `MCP-012` return was already in HEAD |

No open security defects. All three reported security issues (DEF-011/DEF-012/DEF-013) were filed against mid-flight code and were already fixed by the time tests ran against stable HEAD.

---

## Vector results

| # | Test | Status |
|---|------|--------|
| TV-MCP-INIT-001 | `TestMCPHTTPTransport_InitRejected_NotMarkedInitialized` | **PASS** |
| TV-MCP-INIT-002 | `TestMCPHTTPTransport_B30_NotReplicatedFromStdio` | **PASS** |
| TV-MCP-INIT-003 | `TestMCPHTTPTransport_ProtocolVersionHeader_2025` | **PASS** |
| TV-MCP-SESS-001 | `TestMCPHTTPTransport_NoSessionID_Accepted` | **PASS** |
| TV-MCP-SESS-002 | `TestMCPHTTPTransport_SessionIDSentOnSubsequentCalls` | **PASS** |
| TV-MCP-SESS-003 | `TestMCPHTTPTransport_SessionIDUpdatedOnLaterResponse` | **PASS** |
| TV-MCP-SESS-004 | `TestMCPHTTPTransport_SessionExpiry_ReinitFails_MCP004` | **PASS** |
| TV-MCP-AUTH-001 | `TestMCPHTTPTransport_AllowedHost_TokenAttached` | **PASS** |
| TV-MCP-AUTH-002 | `TestMCPHTTPTransport_AuthenticatedRedirect_Blocked_MCP013` | **PASS** |
| TV-MCP-AUTH-003 | `TestMCPHTTPTransport_SuffixConfusion_TokenNotAttached` | **PASS** |
| TV-MCP-AUTH-004 | `TestMCPHTTPTransport_RuntimeHostMismatch_Fatal_MCP012` | **PASS** |
| TV-MCP-SSE-001 | `TestMCPHTTPTransport_SSE_NotificationBeforeResponse` | **PASS** |
| TV-MCP-SSE-002 | `TestMCPHTTPTransport_SSE_CorrectIDSelected` | **PASS** |
| TV-MCP-SSE-003 | `TestMCPHTTPTransport_SSE_StreamEndsWithoutResponse_MCP006` | **PASS** |
| TV-MCP-SSE-004 | `TestMCPHTTPTransport_SSE_MultiLineData_HeartbeatIgnored` | **PASS** |
| TV-MCP-FAIL-001 | `TestMCPHTTPTransport_ConnectionRefused_MCP008` | **PASS** |
| TV-MCP-FAIL-002 | `TestMCPHTTPTransport_ServerError5xx` | **PASS** |
| TV-MCP-FAIL-003 | `TestMCPHTTPTransport_MalformedJSON` | **PASS** |
| TV-MCP-FAIL-004 | `TestMCPHTTPTransport_ContextCancelled_Error` | **PASS** |
| TV-MCP-FAIL-005 | `TestMCPHTTPTransport_JSONRPCError_MCP009` | **PASS** |
| TV-MCP-FAIL-006 | `TestMCPHTTPTransport_UnexpectedContentType_MCP005` | **PASS** |
| TV-MCP-PARITY-001 | `TestMCPHTTPTransport_JSONOutput_Stdout_And_OutputMap` | **PASS** |
| TV-MCP-PARITY-002 | `TestMCPHTTPTransport_PlainTextOutput_StdoutOnly` | **PASS** |
| TV-MCP-PARITY-003 | `TestMCPHTTPTransport_IsError_ExitCode1_MCPToolError` | **PASS** |
| TV-MCP-PARITY-004 | `TestMCPHTTPTransport_SSEResult_SameShapeAsJSON` | **PASS** |
| TV-MCP-REDACT-001 | `TestMCPHTTPTransport_TokenRedaction_NotInAnyOutput` | **PASS** |
| TV-MCP-REDACT-002 | `TestMCPHTTPTransport_AuthFailure_ErrorHasNoToken` | **PASS** |

**Final summary: 27 PASS / 0 SKIP / 0 FAIL** (of 27 Stream D vectors)

---

## Stream C extension (David's AzureCLIAuthProvider) — 2026-08-16T02:29:31Z

**Updated after final pass. 4 Group I adversarial vectors added. Total: 31 PASS / 0 SKIP / 0 FAIL.**

### AUTH-003 tightening
`TestMCPHTTPTransport_SuffixConfusion_TokenNotAttached` had a conditional `if err != nil { check MCP-012 }` that would silently pass on a regression to nil-return. Tightened to `if err == nil { t.Fatal(...) }` — error is now required unconditionally, matching the shape of TV-MCP-AUTH-004. **HOLDS after tightening.** DEF-013 is fixed, `AttachToken` returns fatal MCP-012, Invoke returns non-nil error.

### Comment cleanup
File header, AUTH-003, AUTH-004 stale `// BLOCKED: DEF-013` annotations rewritten. DEF-011/012/013 now marked RESOLVED in header. No false-defect commentary survives.

### Group I — AzureCLI adversarial vectors

| # | Test | Status |
|---|------|--------|
| TV-AZ-001 | `TestAzureCLIProvider_SentinelNeverInAnyErrorOutput` | **PASS** |
| TV-AZ-002 | `TestAzureCLIProvider_NoRetryLoopOnFailure` | **PASS** |
| TV-AZ-003 | `TestAzureCLIProvider_InvalidateThenFailDoesNotReturnStale` | **PASS** |
| TV-AZ-004 | `TestAzureCLIProvider_InvalidateCalledOnTransport401` | **PASS** |

### Sentinel sweep against David's provider

Sentinel: `TESS_SENTINEL_TOKEN_D42E9B1C`. Swept across: all 5 error message paths (az not installed, not logged in, no scope consent, malformed output, generic failure), `Invalidate()`+re-acquire failure, and 401-triggered re-acquire failure. **CLEAN — sentinel not found in any error message, no panic output.** 

David's own redaction test uses `knownToken = "******"` — asterisks can appear in truncated Go error output, making that sentinel collision-prone. Our sentinel is high-entropy and collision-proof. The two tests are complementary; no contradiction.

### Expiry cache behavior verified
`AzureCLIAuthProvider` caches by expiry. `Invalidate()` clears both `token` and `expiry` fields. After `Invalidate()`, a subsequent `Token()` call with a failing runner returns an error (stale token not returned). Cache hit is confirmed: second `Token()` call with a fixed-expiry success runner does not spawn a second shell process. 

### No retry loop
A single failure returns one error. No spin/retry loop exists in `Token()`.

### 401-driven Invalidate
`TestAzureCLIProvider_InvalidateCalledOnTransport401` proves `Invalidate()` is called when the transport receives HTTP 401, so a stale token is discarded before the re-auth attempt rather than being retried forever.

### Build and test status after final pass

```
go build ./...            → exit 0
go test ./internal/tool   → 31 Tess vectors PASS, 0 SKIP, 0 FAIL  
go test ./... -count=1    → exit 0 (62 packages, all green)
```

---

## Key findings summary

1. **B-30 verified**: HTTP transport checks `initResp.Error != nil` before setting `initialized = true`. The stdio defect was NOT replicated. ✓
2. **Protocol version verified**: HTTP transport sends `MCP-Protocol-Version: 2025-03-26`. Does not use stdio's `2024-11-05`. ✓
3. **Requirement 4 (transport parity) verified**: `toolResult()` logic is identical between JSON and SSE paths. Same `ToolResult` shape from both transports. ✓
4. **SSE interleaved notification**: correctly skipped (nil id). ✓
5. **SSE ID correlation**: correct — stale responses with wrong id are skipped. ✓
6. **B-33 Part 1 (host mismatch fatal)**: `AttachToken` returns `MCP-012`, request not sent. ✓
7. **B-33 Part 2 (redirect blocking)**: `CheckRedirect` returns `MCP-013` on authenticated transports. ✓
8. **B-27 (mcp/authAttached wiring)**: event fires and is routed through engine's emitEventLocked in production. ✓
9. **B-24 (token redaction)**: transport layer clean — sentinel not found in errors, result fields, or trace event payloads. ✓

---

## Build and test status

```
go build ./...   → exit 0
go test ./...    → exit 0 (62 packages, all green)
```




---

# Final Acknowledgment: Runtime Portability — Design Agreed (Rev 2)

**Date:** 2026-08-17  
**By:** Gert Core Team  
**To:** SQL Live-Site Operations (gert-sqllivesite)  
**Status:** Design agreed. Implementation begins.  

---

## Accepted

All items accepted. Corrections incorporated below, plus two disclosures from our implementation costing.

---

## 1. Interactive `unspecified` — Corrected

Accepted. Human presence does not guarantee human attention. We adopt your principle as the governing design rule for the entire classification feature:

> **Classification is an opt-in to reduced friction. Absence of classification must never grant additional execution rights.**

Final behavior table:

| Context | `unspecified` behavior |
|---------|----------------------|
| Interactive operator | **Gate fires.** Active confirmation required. |
| CI / headless | Denied unless explicitly permitted by policy. |
| Test (native-only) | Auto-approve. |
| Retry | Never. |
| Late/lost result | INDETERMINATE + halt. |

### Disclosure: migration impact of "interactive unspecified → gate fires"

While costing this change we found a breaking regression we must address together. Today:

- The approval gate on plain (non-substituted) tool calls fires based on runbook-level governance, NOT per-tool `requires-approval`. Tool-level `RequiresApproval` only applies on the substitution path.
- So today, a plain `icm.get-incident` call in interactive mode triggers **zero prompts**.
- Under the new rule, every unclassified tool action in interactive mode would prompt. **A 10-step runbook goes from 0 prompts to 10 prompts.**
- Scope: every tool in gert-sqllivesite is currently unclassified. You would be hit hardest on day 1.

**Mitigation — grandfathered explicit opt-out:**

| Governance state | Behavior |
|------------------|----------|
| `requires-approval: false` **explicitly written** | Treat as legacy equivalent to `classification: read-only`. No prompt. The author explicitly opted out. |
| `requires-approval: true` explicitly written | Gate fires, unchanged. |
| Governance block **absent entirely** (nil pointer) | `unspecified`. Interactive: gate fires. Unattended: deny. |

**Why this is implementable:** `ToolGovernance` is a pointer (`*ToolGovernance`) on the tool definition. A nil pointer = no governance block = author never expressed intent = unspecified. A non-nil pointer with `RequiresApproval: false` = author explicitly wrote a governance block opting out = honor that. This distinction exists in the schema today.

**Migration path:**

1. **New profile field: `legacy_unspecified_policy: allow | prompt`.** Default: `prompt` (enforces the new rule). Existing deployments set `allow` during migration to preserve pre-classification behavior. Explicit, named, auditable escape hatch.
2. **Plan-time warning (PKG-W level):** For each unclassified action when the profile is non-test: `"action 'icm.get-incident' has no classification; treating as unspecified. Add classification: read-only to suppress."` Migration pressure visible without breaking runs.
3. You add `classification: read-only` to your tool actions at your own pace. When complete, remove `legacy_unspecified_policy: allow` from your profile.

This reinforces rather than weakens your principle: absence of a governance block still means unspecified and still means the gate fires. We are only honoring an explicit prior opt-out that an author already wrote down.

---

## 2. Attendance as a Separate Profile Property — Accepted

You are separating execution host (context) from human presence (attendance). Correct — an autonomous agent inside VS Code is unattended despite the host being `vscode-operator`.

**Profile schema:**

```yaml
apiVersion: runtime-profile/v1
id: vscode-autonomous
context: vscode-operator        # execution HOST
attendance: unattended           # approval semantics
approval:
  scope:
    allow_read: true
    allow_mutating: false
    allow_destructive: false
```

`attendance` is a top-level enum: `attended | unattended`. Orthogonal to `context` and `approval.scope`:

- `context` → which tool bindings are valid (AllowedEnvironments).
- `attendance` → which approval gate behavior applies.
- `approval.scope` → which classifications the profile permits at all.

**Disclosure: TTYOutput conflates three properties.** Current Gert hardcodes `TTYOutput: true` unconditionally in `cmd/gert/run.go` — there is no `isatty()` detection. Consequence: `gert run` in a CI pipeline today gets the TerminalApprovalGate and will **block on stdin** if any approval fires. This is a live latent bug, and it is exactly the failure mode your declared-attendance requirement fixes.

Additionally, `TTYOutput` currently drives three distinct things: (a) approval gate selection, (b) physical stdin/stdout availability for collector/choice steps, and (c) terminal-rendered prompts. Profile-declared `attendance` replaces ONLY (a). Physical IO availability (b, c) remains driven by TTY detection because those concern whether stdin/stdout exist, not whether a human is attentive. Implementation: `Attended *bool` on WireOptions; nil inherits legacy TTY inference; profile populates it explicitly when present.

**Edge cases:**

- Profile declares `attended` but no TTY → **Tier 0 failure** (`config/attendance-mismatch`). A profile promising an approver who cannot be reached is a configuration error.
- `attended: false` with `context: cli-operator` → **Legal.** Your autonomous-agent case. Gets unattended rules.

---

## 3. Test Context Must Not Bind to Production — Accepted (revised)

**Tier 0 rule (corrected):**

| Transport mode | In test context |
|---------------|----------------|
| `native` | Always allowed. |
| `mcp` (subprocess) | **Blocked by default.** Allowed with explicit profile opt-in: `transport.allow_subprocess_in_test: true`. |
| `mcp-http` | **Never allowed.** No override. |

Rationale for the subprocess escape hatch: a hermetic local fake MCP server (`fake-icm-mcp` binary producing deterministic responses, no real endpoint) is a legitimate pattern for integration-testing the MCP transport layer itself. Banning subprocess entirely forecloses that. The opt-in is explicit, named, auditable — and HTTP endpoints remain absolutely forbidden.

Error code: `config/test-context-non-native-binding`  
Message (mcp-http case):
```
error: tool "icm" uses transport mode "mcp-http" in a test-context profile.
  Test profiles must not bind to HTTP endpoints.
  fix: use a package-map pointing to your mock package, or change the profile context.
```

Message (subprocess, no opt-in):
```
error: tool "icm" uses transport mode "mcp" (subprocess) in a test-context profile.
  Subprocess transport requires explicit opt-in in test context.
  fix: set transport.allow_subprocess_in_test: true in the profile, or use native transport.
```

This check reads `plan.Tools` (post-resolution, post-package-map), so it ships with the early `gert plan` work at zero additional effort. The `--package-map` interaction validates exactly as designed: test profile + production package-map → reads `mcp-http` → Tier 0 error. Right layer, correct behavior.

The approval rule remains sound: native transport has no real side effects; subprocess fakes under explicit opt-in are by definition hermetic (the author vouched by opting in); HTTP is banned. Auto-approve in test context is safe.

---

## 4. OQ2 — Closed: Subprocess

Confirmed. Extension already uses a Gert subprocess. Phase 0 validates framing and lifecycle:

| Phase 0 deliverable | Owner |
|---------------------|-------|
| Framing protocol document | Gert Core |
| Lifecycle spec (start, stop, crash-recovery, version detection) | Gert Core + extension team |
| Capability-advertisement handshake | Gert Core |
| CancelRequest message type | Gert Core |
| Feasibility validation against current extension | SQL Live-Site Ops |

Library integration only if subprocess proof fails on a blocking criterion.

---

## 5. Final Status

**Design agreed. Gate closed. Implementation starts.**

New Phase 1 items from this round:

| Item | Effort |
|------|--------|
| Per-action `Classification *string` on ToolAction | 0.5 days |
| `legacy_unspecified_policy` profile field + gate routing | 0.5 days |
| Plan-time PKG-W warning for unclassified actions | 0.5 days |
| Declared `attendance` on WireOptions + profile | 0.5 days |
| `config/attendance-mismatch` Tier 0 check | included |
| `config/test-context-non-native-binding` Tier 0 check | 0.5 days |
| Direct-invocation approval gate enforcement (Execute() path) | 1–2 days |

These join the existing Phase 1 scope. Total addition: ~4 days.

You have corrected us four times in this exchange and been right every time — your unspecified correction surfaced the migration hazard we would otherwise have shipped blind. The design is better for it. We start work this week.

---

*Gert Core Team — 2026-08-17*


---

# Gate Closure: Runtime Portability for Gert Runbooks (Revised)

**Date:** 2026-08-16  
**By:** Gert Core Team  
**To:** SQL Live-Site Operations (gert-sqllivesite)  
**Re:** Gate ruling on your reply to our conditional acceptance  
**Status:** GATE CLOSED — work authorized to begin  

---

## Verdict: All blocking conditions are satisfied. Gate is closed.

This revision incorporates source-grounded findings from our implementation review completed after the initial closure. It corrects our earlier ruling on environment vocabulary, discloses a coverage gap in the current approval gate that is load-bearing for your §11 requirements, and refines several technical details.

---

## Condition Status

| # | Condition | Status |
|---|-----------|--------|
| 1 | Accept additive schema extensions | **SATISFIED.** |
| 2 | Accept tiered preflight | **SATISFIED.** |
| 3 | Accept corrected late-result semantics | **SATISFIED.** |
| 4 | Close host bridge protocol gaps before Phase 2 | **SATISFIED.** |
| 5 | Answer OQ2 before Phase 2 | **SATISFIED.** |
| 6 | Revised phasing and estimates | **SATISFIED.** |

---

## Rulings on Your Three Counter-Positions

### 2a. `classification: unspecified` — ACCEPTED

You are right. Defaulting absent classification to `read-only` would silently mark existing mutating tools as safe. `unspecified` is the correct conservative default.

**Schema detail.** `classification` is a **per-action** field, not per-tool. A single tool legitimately has read-only `get-incident` and destructive `kill-process`. In Go: `Classification *string` on the `ToolAction` struct — a pointer so `nil` (absent/unspecified) is cleanly distinguishable from an explicit value. `yaml:"classification,omitempty"`. Four-value enum: `read-only | mutating | destructive | unspecified` (nil = unspecified). This is permanent — no deprecation horizon.

**Reconciled precedence table.** This supersedes our earlier draft:

| `requires-approval` | `classification` | Behavior |
|---------------------|-------------------|----------|
| `true` | any (incl. nil/unspecified) | Gate fires. Explicit legacy bool wins; preserved. |
| `false` or absent | `read-only` | No gate. |
| `false` or absent | `mutating` | Gate fires if profile approval mode ≥ interactive. Auto-approve in test context. |
| `false` or absent | `destructive` | Gate ALWAYS fires. |
| `false` or absent | nil (unspecified) | **FAIL CLOSED** in unattended contexts. In interactive contexts: warn and proceed (human present to notice). |

**All code paths** (retry, late-result, approval) must handle `unspecified` explicitly — no default-case fallthrough to `read-only`:

| Dimension | `unspecified` behavior |
|-----------|----------------------|
| Retry | No retry. |
| Late-result | INDETERMINATE + halt (conservative, same as mutating). |
| Unattended approval | Denied (fail-closed). |

### 2b. Unattended approval gate — ACCEPTED, with scoping and a disclosure

**Your concern is well-founded — more so than either of us realized.** We owe you a disclosure before stating the fix.

**Disclosure: current approval gate coverage is partial.** The existing `ApprovalGate` enforcement fires ONLY on the substitution path (`executor/tool.go:executeSubstitution()`, where `planResult.EffectiveGovernance.RequireApproval` is checked). The direct-invocation path (`Execute()` → `runtime.Invoke()`) — which is the path for plain process-backed, stdio-MCP, and mcp-http tool calls — has **no approval check at all**. A destructive tool invoked directly today bypasses the gate entirely.

This means the `ApprovalGate` interface you are relying on as the enforcement seam for §11 currently covers only substituted actions. Shipping a `ProfileApprovalGate` without closing the direct-invocation path would give you a false sense of safety.

**Phase 1 fix:** We will add the classification-aware approval check to `Execute()` on the direct-invocation path at the same time we ship the `ProfileApprovalGate`. This is a ~1–2 day addition. After Phase 1, ALL tool invocations — substituted and direct — pass through the gate.

**Phase 1 gate behavior (`ProfileApprovalGate`):**
- Reads the runtime profile's `approval.scope` (which classifications are permitted).
- `read-only` actions: auto-approves.
- `mutating`, `destructive`, `unspecified` actions: **denies** unless the profile explicitly permits that classification.
- Denial is a typed `approval/denied` error naming the profile, action, and classification.

This is fail-closed by denial. No evidence infrastructure needed.

**Phase 3 upgrades to evidence-recording:** Adds policy ID, identity (managed identity object ID, workload identity subject), run/step/retry provenance, and a structured approval-evidence trace event to `ApprovalRecord`. Good news: the current `ApprovalRecord` type (`pkg/governance/evidence.go`) is a value type with only `Approver`, `ApprovedAt`, `Token`. Adding `PolicyID`, `RunID`, `StepID`, `RetryCount`, `Classification` as `omitempty` fields breaks no existing trace readers and requires no interface signature change. Backward-compatible.

Phase 1 = fail-closed by denial; Phase 3 = fail-closed by requiring evidence. Both safe.

### 2c. Environment/context vocabulary — CORRECTED from our initial ruling

Our initial ruling said `AllowedEnvironments` values are free-form strings and the existing `"real"` values in conformance fixtures are "a test vocabulary, not a production standard." That was wrong. Here is what we found on closer examination.

**The two-axis problem.** The existing 25 occurrences of `allowed-environments: ["real"]` in `tv-enum.yaml` sit on a `kubectl` tool that drains nodes (real side effects). The value `"real"` maps to Gert's existing `engine.RunModeReal` constant — one of three `RunMode` values: `real | dry-run | replay`. The original corpus author was marking tools that must not be bypassed by dry-run mode.

`RunMode` and deployment context are **two orthogonal axes**:

| Axis | Values | Meaning |
|------|--------|---------|
| **RunMode** (exists today) | `real`, `dry-run`, `replay` | Whether the engine actually executes steps |
| **Profile context** (new) | `cli-operator`, `vscode-operator`, `ci`, `headless-server`, `test` | Which deployment environment the runbook runs in |

Overloading one field for both axes would silently destroy the existing dry-run safety intent — a `kubectl drain` tool marked `allowed-environments: ["real"]` (meaning "don't skip in dry-run") would suddenly mean "only runs in a profile named 'real'", which is not what the author intended.

**Our resolution:**

1. **`AllowedEnvironments`** becomes the profile-context allowlist, as you assumed. Values are your §6 context vocabulary: `cli-operator`, `vscode-operator`, `ci`, `headless-server`, `test`. Matching is exact, case-sensitive, against the runtime profile's `context:` field. Empty = all contexts allowed (backward compatible).

2. **NEW field: `AllowedModes []string`** on `ToolGovernance` for the RunMode dimension. `yaml:"allowed-modes,omitempty"`. Carries the `real | dry-run | replay` vocabulary. Empty = all modes allowed.

3. **Migration:** We migrate the 25 existing `allowed-environments: ["real"]` fixtures to `allowed-modes: ["real"]`. This is a test-data-only change in `tv-enum.yaml`. Zero runtime breakage because neither field is enforced today — the enforcement code we write will read the new field names.

4. **`RequiresCapabilities`:** Deferred. We will NOT enforce or populate this field until the profile schema settles what a "capability" is (transport types? auth providers? host features?). Populating it now creates another ghost field. Phase 1 does not need it.

5. **Vocabulary convention:** We will document a recommended context vocabulary table in the Phase 0 profile spec, based on your §6 matrix. Values are free-form strings (not a closed enum), but we provide the canonical set for interoperability. Confirm you will use your §6 values; we will enforce against them.

**This is no longer blocking the early win.** We define the vocabulary and the field semantics. You confirm your context values match your §6 matrix (or tell us otherwise). The fixture migration is ours to do.

---

## Binding Model — Confirmed

Your statement matches our position:

> A profile must not silently rewrite a tool definition's transport mode. Different physical implementations remain separate tool definitions/packages selected through the existing binding machinery. `--profile` and `--package-map` compose rather than compete.

**Composition precedence:**

1. `--package-map` wins at **YAML selection**: which package's tool definition resolves each `toolRef`. `mergePackageBindings()` applies CLI-over-project precedence; `package/resolved` trace events record `origin: project | package-map`.

2. `--profile` wins at **runtime parameterization**: auth provider, endpoint URLs, approval policy, context identity. It operates on the already-resolved tool definition.

3. **Architectural constraint for Phase 1:** The binding resolver MUST consume `plan.Tools` (the post-catalog, post-package-map resolved tool set returned by `planner.Plan()`), NOT re-resolve toolRefs independently. This guarantees both flags operate on the same resolved tool set. If `--package-map` selects a `native` mock while the profile expects `mcp-http`, the Tier 0 compatibility check surfaces the mismatch before execution — correct behavior, because the operator explicitly chose an incompatible combination.

---

## `gert plan` — Profile Compatibility Report

We confirm this ships early in Phase 1 as you requested. Implementation scope:

- Reuses `run.go`'s existing wiring (~150 lines, extractable to a shared helper) to load profile, package-map, and call `planner.Plan()`.
- Checks each resolved tool's `AllowedEnvironments` against the profile context and `transport.mode` compatibility.
- Prints a binding table: one row per toolRef (transport, auth, endpoint, status).
- Exit codes: 0 = Tier 0 clean, 1 = Tier 0 failures. `--output=json` for CI.
- **Estimated effort: 2–3 days.**

**Explicit limits (set expectations now):** This is a **profile compatibility report**, not a full binding table. It reports what the tool YAML declares. It cannot validate endpoint reachability or token acquisition (those are Tier 2 runtime probes). It cannot show profile-overridden transport configs (that requires the resolver, which lands later in Phase 1). We will label it accordingly in the CLI help text.

---

## Open Items

| Item | Owner | Blocks |
|------|-------|--------|
| Confirm your context vocabulary matches §6 values (`cli-operator`, `ci`, `headless-server`, `test`, `vscode-operator`) | SQL Live-Site Ops | Early win (minor — confirmation only, not design) |
| `AllowedModes` field + fixture migration | Gert Core | Part of early win |
| Profile spec — context semantics, `--profile`/`--package-map` composition, Tier 0 rules | Gert Core | Phase 1 |
| OQ2 — library vs. subprocess spike | Joint | Phase 2 |

---

## Phase Agreement — Confirmed

- **Early win (immediate):** `AllowedEnvironments` enforcement in `resolveTool()`, `AllowedModes` field + fixture migration, approval gate on direct-invocation path. Pending only your context vocabulary confirmation.
- **Phase 0 (2 weeks):** Profile spec, host architecture decision, framing protocol, managed-identity feasibility.
- **Phase 1 (6–8 weeks):** Binding resolver, tiered preflight, `gert plan` (profile compatibility report, early), managed identity, per-action `classification`/`idempotent` fields, `ProfileApprovalGate` (fail-closed by denial), halt-on-timeout, ICM proof.
- **Phase 2 (re-estimated after Phase 0):** Generic host transport, VS Code adapter.
- **Phase 3 (4–6 weeks):** Workload identity, reconnect/idempotency hardening, approval evidence recording.

We are ready to start. Confirm your context vocabulary and we begin this week.

---

*Gert Core Team — 2026-08-16*


---

# Gert Core Team Response: Runtime Portability for Gert Runbooks

**Date:** 2026-08-16  
**From:** Gert Core Team  
**To:** SQL Live-Site Operations (gert-sqllivesite)  
**Re:** Implementation Request — Runtime Portability for Gert Runbooks  
**Status:** Conditional acceptance  

---

## Verdict: ACCEPT-WITH-MODIFICATIONS

We accept the architecture direction. The layering — invariant tool contract → runtime binding resolver → transport + auth — is the right shape and maps cleanly onto the existing `ToolTransport` interface and `AuthProvider`/`TokenGate` machinery. Your consumer scenario is legitimate, and the five-context matrix is the correct target.

We will not implement it as specified. The document has four categories of issue: factual corrections to your current-state section, a structural contradiction between your stated non-goals and your actual requirements, a preflight design that answers the wrong question for the consumer's core need, and safety defects in the late-result / reconnect semantics that are blocking.

Conditions of acceptance, in priority order:

1. **[BLOCKING]** Acknowledge the §13 contradiction and accept additive schema extensions (§A below).
2. **[BLOCKING]** Accept our tiered preflight redesign replacing §10.1 (§C below).
3. **[BLOCKING]** Accept the corrected late-result semantics for mutating/destructive actions (§E below).
4. **[REQUIRED-BEFORE-PHASE-2]** Close the host bridge protocol gaps (§F below).
5. **[REQUIRED-BEFORE-PHASE-2]** Answer open question #2 (library vs. subprocess) — we have a recommendation but need your extension architecture constraints (§G below).
6. **[ADVISORY]** Accept our revised phasing and estimates (§H below).

---

## §A. Corrections to Current-State Claims

### A.1 — `run-gert.ps1` (§3.2, your table row "CLI runner")

`run-gert.ps1` does not exist in Gert core. No `.ps1` file of any name exists in the Gert repository. This is a gert-sqllivesite artifact. Please correct the attribution — it confused our source verification and will confuse anyone reviewing the request downstream.

### A.2 — Plan-time tool resolution already exists

Your §10.1 frames preflight as if no plan-time validation exists today. That is incorrect. Gert already has a plan-time tool resolution gate:

- `planner.go:resolveTool()` (line 533) verifies every tool step's tool+action can be resolved from the registry. Unresolvable tools fail with `PLAN-010` — a typed `PlanError` — before execution begins.
- `cmd/gert/run.go` calls `adapter.ResolveToolRefsViaCatalog()` and exits with code 2 (`exitValidation`) on failure.
- `internal/planner/validate.go` runs additional structural checks.

What is missing is not preflight itself, but **environment-contextual binding validation within** the existing plan-time gate. The resolver extends this gate; it does not replace it.

### A.3 — `--package-map` already exists

Your document describes "pointing Gert at different package directories" as the current context-switching mechanism but does not mention `--package-map`, which is the formalized version of exactly that. `--package-map` accepts a `config/v1` file that overrides the project's `requires:` and `tool-paths:` bindings, so the same runbook resolves different tool packages. This is already tested (including trace provenance recording — each resolved package records whether it came from the project config or the package-map override).

The binding resolver should extend `--package-map` semantics — it is the existing foundation for "same runbook, different bindings." Do not design the resolver as if this mechanism doesn't exist.

### A.4 — The ghost fields: `AllowedEnvironments` and `RequiresCapabilities` [BLOCKING — early win]

Your document asks for capability preflight but missed the fact that Gert's tool governance schema (`pkg/schema/tool.go`, `ToolGovernance` struct) **already declares** two fields directly relevant to your need:

```go
RequiresCapabilities []string `yaml:"requires-capabilities,omitempty"`
AllowedEnvironments  []string `yaml:"allowed-environments,omitempty"`
```

These fields are parsed, serialized, and appear in conformance test fixtures. They are **not enforced at runtime** — zero lines of Go code read or check them. They are dead.

Wiring enforcement into the existing `planner.go:resolveTool()` — comparing a tool's `AllowedEnvironments` against a runtime context string — would deliver the consumer's core need ("tell me this runbook can't run in this environment") with:

- Zero schema changes (the fields already exist in the schema)
- Zero new abstractions (it's a string-set check inside an existing plan-time gate)
- A ~2-day implementation effort

**We propose starting this immediately, in parallel with Phase 0, as an early win.** It does not require the full binding resolver — it is a plan-time check using existing declared metadata. Your tool authors can start populating `allowed-environments:` on tool definitions today. The full resolver (Phase 1) builds on top of this foundation.

This is the lowest-cost path to your core need and we are surprised the request missed it.

### A.5 — `RequiresApproval` / `ApprovalGate` already exists

Your §11 proposes action classification and approval policy. The approval enforcement pattern already exists in Gert:

- `governance.ApprovalGate` interface with `RequestApproval(ctx, stepID, reason)` returning an `ApprovalRecord`.
- `TerminalApprovalGate` (interactive: prompts stdin/stdout) and `NoOpApprovalGate` (non-interactive: auto-approves).
- Gate selection is already context-dependent: `TTYOutput=true` → `TerminalApprovalGate`; `TTYOutput=false` → `NoOpApprovalGate`.
- Enforced at substitution time in `executor/tool.go:executeSubstitution()` and for dynamic includes via `IncludeExecutor.WithApprovalGate()`.
- `ToolGovernance.RequiresApproval` is the existing per-tool declaration, already enforced.

Your §11's classification scheme (read-only / mutating / destructive) is a finer-grained replacement for the boolean `RequiresApproval`. That is a reasonable evolution, but the approval enforcement pattern — `ApprovalGate.RequestApproval()` gated by policy before execution — is the pattern §11 must follow, not reinvent.

### A.6 — §14 Open Question 5 is CLOSED

OQ5 asks: "What deadline and cancellation APIs does the current AuthProvider interface expose?"

**Answer:** `AuthProvider.Token(ctx context.Context)` accepts a context. `AzureCLIAuthProvider.acquire()` passes that context to `exec.CommandContext(ctx, azPath, args...)`. The OS kills the `az` subprocess on context cancellation or deadline expiration. Deadline propagation works today. The managed identity and workload identity providers must follow the same pattern (use `http.NewRequestWithContext(ctx, ...)` for IMDS/token-exchange calls).

Drop this Phase 0 spike. It is already answered.

---

## §B. The Schema-Freeze Contradiction [BLOCKING]

§13 declares as explicit non-goals: "No runbook schema changes" and "No tool contract schema changes."

Your own document contradicts this in at least three places:

1. **§10.1 (preflight)** requires knowing which contexts a tool supports — that is a tool-level declaration. Without it, you can only discover "not configured here" by attempting to bind and failing at runtime.
2. **§10.4 (idempotency)** says "unless the tool action is declared idempotent" — but there is no `idempotent` field on the action schema. You cannot declare idempotency without a schema change.
3. **§11 (action classification)** defines read-only / mutating / destructive, but never says where this classification is declared. It must be on the tool action schema.

**Our position:** We accept no-BREAKING-changes as a constraint. We require ADDITIVE schema extensions — new optional fields with safe defaults that do not break existing tool definitions:

| Field | Schema location | Default | Purpose |
|-------|----------------|---------|---------|
| `allowed-environments` | `tool/v1` governance block | `[]` (all environments) | Static preflight: which contexts this tool supports |
| `requires-capabilities` | `tool/v1` governance block | `[]` (no special capabilities) | Static preflight: host capabilities required |
| `classification` | Per-action | `read-only` (safe default) | Approval policy per §11 |
| `idempotent` | Per-action | `false` (safe default) | Reconnect/retry policy per §10.4 |

The first two already exist in the schema (§A.4). The latter two are new but backward-compatible — unset means the safe default applies.

**Please acknowledge this contradiction and confirm you accept additive extensions.** We will not design a resolver that requires runtime discovery for information that should be statically declared.

---

## §C. Preflight Redesign [BLOCKING]

### The problem with §10.1

Your preflight (§10.1 steps 1–4) conflates three fundamentally different operations:

- **Step 1** (resolve toolRef → transport + auth) is a static config-graph traversal. Zero I/O. Microseconds.
- **Step 2** (validate auth can obtain a token) is a real credential operation — may prompt for login, hit IMDS, contact AAD token exchange, consume rate limits, emit tenant audit events.
- **Step 3** (validate transport endpoint reachable) is a network probe subject to transient failures.

An operator who sees `binding/auth-unavailable` cannot tell whether the issue is "this tool is not configured for this environment" (fix: add a binding to the profile) or "IMDS hasn't warmed up yet" (fix: wait 30 seconds and retry). These require completely different responses. The error taxonomy (§10.2) is too coarse to distinguish them.

### Our required design: Tiered Preflight

**Tier 0 — Static / offline. Zero I/O. MANDATORY before every run. Not optional, not skippable.**

Pure config-graph traversal:

1. Every `toolRef` resolves to a `tool.yaml` in a loaded package (existing PLAN-010 gate).
2. The resolved tool's `transport.mode` has a binding entry in the active runtime profile.
3. That binding entry is structurally complete: `mcp-http` needs `endpoint` + `auth.provider`; `mcp` (subprocess) needs `command`; `native` needs nothing; `host-bridge` needs a registered adapter ID.
4. The referenced auth provider is a known type with required config present.
5. Environment variable interpolations (e.g. `${GERT_MCP_ENDPOINT}`) resolve to non-empty values.
6. `AllowedEnvironments`, if declared on the tool, includes the current profile's context.

Nothing invoked, nothing dialed, no tokens. Runs in CI, runs in an editor, runs with zero credentials. This is the check that answers your consumer's core need.

**Tier 1 — Local / no network. Opt-in: `--preflight=local`.**

- Subprocess `command` is on PATH and executable.
- `az` is on PATH for `azure-cli` provider.
- IMDS `169.254.169.254` responds within 500ms (link-local reachability, not a token call).
- Native transport runbook files exist on disk.

**Tier 2 — Live / network. Opt-in: `--preflight=live`.**

- Token acquisition attempted (your §10.1 step 2).
- Endpoint probed (your §10.1 step 3).
- This is where `auth/credential-failure` and `transport/endpoint-unreachable` surface.

`--preflight=live` ADDS Tier 2 on top of Tier 0 — never instead of it.

### Corrected Error Taxonomy

Replaces the §10.2 table. Five codes, split by static vs. dynamic:

| Code | Tier | Meaning | Operator message includes |
|------|------|---------|--------------------------|
| `config/tool-unresolved` | 0 | toolRef names a tool in no loaded package | Runbook path, checked packages, fix guidance |
| `config/no-binding-for-profile` | 0 | **The consumer's case.** Tool has no binding in the active profile | Tool name, declared transport mode, profile ID, AND which profiles DO work (computable at Tier 0) |
| `config/binding-incomplete` | 0 | Binding exists, required config absent | Missing field name AND the unset env var |
| `auth/credential-failure` | 2 | Token acquisition failed at runtime | "This is a runtime error, not a configuration error. The binding is valid." |
| `transport/endpoint-unreachable` | 2 | Endpoint probe failed at runtime | "This is a runtime/network error. The binding and credentials are valid." |

The static-vs-dynamic distinction must be explicit **in the message text**. Operator action is completely different for each class.

The `config/no-binding-for-profile` message format:

```
error: tool "icm" has no binding in runtime profile "headless-server".
  tool transport declared: mcp-http
  profile "headless-server" does not declare a binding for transport mode
  "mcp-http" / tool "icm".
  fix: add a binding for "icm" to the "headless-server" profile, or run
       in a profile that includes it.
  note: this runbook runs correctly in profiles: cli-workstation, test
```

That final `note:` line — which profiles DO work — is computable at Tier 0 from the binding table alone and is the single highest-value line in this entire design.

### `gert plan` command

We will implement `gert plan --profile <id> <runbook>` as a non-executing binding table report:

```
$ gert plan --profile headless-server runbooks/icm-tsg-router.runbook.yaml
  toolRef               transport    auth              endpoint                   status
  icm                   mcp-http     managed-identity  ${ICM_MCP_ENDPOINT}        ✓ bound
  tsg-recommendation    mcp-http     managed-identity  ${TSG_MCP_ENDPOINT}        ✓ bound

$ gert plan --profile vscode-extension runbooks/icm-tsg-router.runbook.yaml
  toolRef               transport    auth              endpoint                   status
  icm                   host-bridge  extension-managed  —                         ✗ config/no-binding-for-profile
  tsg-recommendation    host-bridge  extension-managed  —                         ✗ config/no-binding-for-profile
```

Exit codes: 0 = Tier 0 clean, 1 = Tier 0 failures, 2 = Tier 2 failures (only with `--preflight=live`).

`--output=json` for CI consumption. Companion `gert plan --show-profiles <runbook>` lists all profiles in which every toolRef has a complete binding.

This belongs in Phase 1, not deferred.

---

## §D. Transport Mode Conflict and Per-Tool Overrides [REQUIRED-BEFORE-PHASE-1]

### The unspecified conflict

Your §5.1 says the tool's `transport` block declares WHAT transport a tool uses; the profile determines HOW it's satisfied. But your document never addresses the conflict case:

- Tool declares `transport.mode: mcp` (subprocess, `command: icm-mcp`)
- Profile says `transport.mcp.mode: direct-http`

Is this an override? An error? A silent re-interpretation?

**Our position: the profile does NOT override the tool's declared transport mode.** Switching from `mode: mcp` (subprocess-owned auth) to `mode: mcp-http` (Gert-brokered auth) changes operational semantics — it is not a parameter substitution. The correct mechanisms are:

1. The package provides **multiple tool definitions** (one per context), selected by the resolver. Mock packages already work exactly this way. This is proven.
2. Or, the resolver matches on the tool's declared mode and provides **parameters** for that mode (endpoint URL, auth config), without changing the mode itself.

Mechanism (1) is already proven by `--package-map`. The resolver should formalize it, not replace it.

### Per-tool overrides in profiles

A flat profile that says "all tools use direct-http" fails when one tool is mock-only or subprocess-only. Profiles must allow per-tool binding overrides:

```yaml
apiVersion: runtime-profile/v1
id: integration-test
context: ci
defaults:
  transport: direct-http
  auth: workload-identity
overrides:
  tsg-recommendation:
    transport: native-mock
    auth: none
```

Without this, you cannot express a partially-mocked integration test profile — a scenario we consider essential.

---

## §E. Safety Defects [BLOCKING]

### E.1 — Late-result semantics for mutating/destructive actions

§10.4 says: "Late results (arriving after deadline or cancellation) must be discarded with a logged warning — never applied."

For read-only actions, discarding late results is correct. For mutating/destructive actions, it is a **safety defect**.

Scenario: `dsconsole.reissue-update-slo` is called. Connection is lost at 3 seconds. Deadline fires at 30 seconds. Gert discards the late success. Gert records failure. Reality: the SLO WAS reissued. The operator now has a false picture of system state. Retry double-executes. For `dsconsole.kill-sql-process` or `dsconsole.stop-database-copy`, anything less than halting with an indeterminate state is unacceptable.

**Required correction:** §10.4 must split behavior by action classification:

| Classification | On late/lost result | Runbook behavior |
|---------------|-------------------|-----------------|
| `read-only` | Discard, log warning | Continue (or retry if idempotent) |
| `mutating` | Record as INDETERMINATE | Halt. Trace records state is unknown. Operator must verify before resuming. |
| `destructive` | Record as INDETERMINATE | Halt. Trace records state is unknown. Operator must verify before resuming. |

This directly follows from your own Invariant #4 (fail-closed).

### E.2 — Reconnect retry requires re-approval

§10.4 (reconnect/retry) and §11 (approval) are never cross-referenced. A reconnect retry is a new invocation event:

- In interactive contexts: the original approval covers a specific invocation at a specific time. Reconnect must re-prompt.
- In unattended contexts: the retry must be recorded in §11.3 approval evidence, including that it IS a retry and the original `run_id`.

### E.3 — Phase 1 / Phase 3 ordering creates an unsafe window [BLOCKING]

§10.4 (reconnect/idempotency) is Phase 3. But Phase 1 ships direct-HTTP to a headless server with managed identity. HTTP calls time out — slow servers, intermittent networks, IMDS cold starts, cross-region latency. For the entire duration of Phase 1 and Phase 2 (8–12 weeks by your estimates), every timeout on a mutating/destructive action leaves the runbook in an indeterminate state with no recovery semantics and no operator guidance.

**Required mitigation (choose one):**

**Option A (preferred):** Phase 1 commits to an explicit halt-on-timeout, no-retry policy:
- Timeout halts the runbook with a typed `transport/timeout` error.
- For mutating/destructive actions, the error message includes: "This action's completion state is unknown. Manual verification is required before retrying."
- No automatic retry, no reconnect. Runbook is terminal.
- This is safe, honest, and implementable in Phase 1 without pulling Phase 3 forward.

**Option B:** Phase 1 is formally restricted to `read-only` tool actions as an explicit acceptance criterion. Mutating/destructive actions over direct-HTTP are deferred to Phase 3 when the recovery semantics are in place.

We strongly prefer Option A. Confirm which you accept.

---

## §F. Host Bridge Protocol Gaps [REQUIRED-BEFORE-PHASE-2]

The `ToolRequest`/`ToolResult` types in §7.2 are underspecified. Before Phase 2 implementation begins, the following must be resolved:

### F.1 — Protocol version (REQUIRED)

The VS Code extension auto-updates independently of Gert core. Without a `protocol_version` field on the registration handshake, mismatched versions will silently misinterpret fields. Add a version field; define version negotiation or hard-fail semantics.

### F.2 — Host capability advertisement (REQUIRED)

At registration time, the host adapter must advertise:

```
HostCapabilities {
  adapter_id:       string
  protocol_version: string
  tools: [{
    tool_name: string,
    actions:   [string]
  }]
}
```

This is not independent of the preflight design. It IS the data source that makes Tier 0 preflight work when the transport is a host bridge. Without it, `config/no-binding-for-profile` is unanswerable in the VS Code context, and the consumer's core need is unmet.

### F.3 — Correlation IDs (REQUIRED)

`ToolRequest` needs `run_id` and `step_id` in addition to `request_id`. `TokenGate` already emits per-auth-attachment trace events (host + scope, B-24/B-27); without step correlation, that event cannot be joined to the runbook step in the audit log. For headless mutating actions this is a compliance gap.

### F.4 — `CancelRequest` message type (REQUIRED)

`ToolResult.status` includes `"cancelled"` but there is no cancel channel. §10.3 says "propagate cancellation to the transport adapter" — this is unimplementable over IPC without an explicit `CancelRequest { request_id }` message type.

### F.5 — Streaming/progress (ADVISORY)

Tool calls lasting 30+ seconds with no progress signal look hung. If streaming is out of scope for Phase 2, say so explicitly in the spec — do not leave it as a silent gap.

### F.6 — Framing protocol document (REQUIRED-BEFORE-PHASE-2)

If Gert runs as a subprocess (our recommendation per §G), stdout carries both execution trace events and `ToolResult` messages on one channel. The "interaction channel" named in the Phase 0 spike is not specified enough to implement against. Require an explicit framing protocol document before Phase 2 ships: message envelope format, all message types with a `message_type` discriminator, length-prefix or NDJSON framing, ordering guarantees, malformed-message handling.

### F.7 — Token isolation is convention, not enforcement (ADVISORY)

Invariant #5 ("no token leakage") is currently a gentleman's agreement. Actual enforcement requires: (a) `ToolRequest` has no field capable of carrying a token, (b) schema-validate `ToolResult.outputs` against the tool's declared output schema before Gert core processes or traces it, stripping unknown fields with a logged warning, (c) extend the existing B-24 scrubber to cover tool result outputs written to the trace. This is advisory for Phase 2, but should be on the Phase 3 hardening checklist.

---

## §G. Open Questions — Our Answers

### OQ1 — Profile file location and format

**Position:** All three. Profiles should be discoverable in a precedence order:

1. CLI flag: `--profile <path>` (highest precedence, overrides everything)
2. Repo-local: `.gert/profiles/<id>.yaml` (version-controlled, team-shared)
3. User config: `~/.config/gert/profiles/<id>.yaml` (per-user defaults)

Profile IDs are resolved in this order; first match wins. `gert plan --show-profiles` lists all discovered profiles and their source locations. This is consistent with how `.gert/config.yaml` and `--package-map` work today. `--profile` and `--package-map` should compose — profile provides auth/transport parameters, package-map provides tool resolution.

### OQ2 — Library vs. subprocess for VS Code

**Our recommendation: subprocess with IPC.** Reason: fault isolation. A VS Code extension crash (or a bug in the bridge adapter) must not kill a running Gert execution. In-process embedding means a panic in either direction is fatal to both. Subprocess isolation contains blast radius.

However, this is a question about YOUR extension's architecture, not ours. We need you to evaluate fault isolation, latency, and the multiplexing cost (§F.6) against your extension's constraints and give us a binding answer before Phase 2 design begins. Phase 0 should include a spike evaluating both options, with fault isolation as a primary criterion alongside latency.

### OQ3 — Auth token caching and refresh

**Answer:** Follow the existing precedent. `AzureCLIAuthProvider` (verified: `auth_azurecli.go:47-90`) already implements in-memory caching with proactive refresh at a 5-minute-before-expiry buffer, with fallback to the still-valid cached token if early refresh fails. Managed identity and workload identity providers must follow the same pattern. Per-call acquisition is unacceptable — IMDS alone can be 200ms+ per call.

### OQ4 — Profile inheritance

**Position:** No inheritance. Keep profiles flat. The complexity of inheritance resolution (which fields merge, which override, conflict semantics, circular inheritance) is not justified by the use case. Instead, per-tool overrides within a profile (§D) provide the necessary composability. If you need a "base azure" configuration shared across profiles, extract it as a YAML anchor or use a profile template convention in your repo — that is a consumer-side concern, not a runtime feature.

### OQ5 — AuthProvider deadline behavior

**CLOSED.** See §A.6 above. Already works. Drop this spike.

---

## §H. Revised Phasing and Estimates

### Early Win — AllowedEnvironments enforcement (can start immediately, ~1 week)

Wire the existing dead `AllowedEnvironments` and `RequiresCapabilities` fields in `ToolGovernance` into `planner.go:resolveTool()`. Compare declared `AllowedEnvironments` against a runtime context string (from `--profile` or a new `--context` flag). Plan-time failure with a typed error. Zero schema changes, zero new abstractions. Delivers the consumer's core capability check before any Phase 0 work completes.

**We can start this now.** Confirm you agree and we will land it independently.

### Phase 0 — Spikes (2 weeks) — accepted scope, plus additions

Your Phase 0 scope is correct, minus the OQ5 spike (already answered), plus:

- **[ADD]** Resolve OQ2 (library vs subprocess) with a binding decision, evaluating fault isolation as primary criterion.
- **[ADD]** Resolve profile file location (OQ1 — we propose the three-level precedence above; confirm or counter).
- **[ADD]** Prototype `gert plan --profile X` dry-run (validates Tier 0 preflight without live infrastructure).
- **[DROP]** AuthProvider deadline behavior spike (OQ5 — answered).

### Phase 1 — Resolver + Managed Identity + ICM Proof (6–8 weeks, not 4–6)

Your 4–6 week estimate is optimistic. Specific risks:

- `DefaultToolRuntime` caches transports in a `persistent map[string]ToolTransport` keyed by tool name. Profile-scoped endpoints/auth require re-scoping that cache — `OverlayRegistry` is the existing precedent, but adapting it adds work.
- `gert plan` and the tiered preflight taxonomy are included in our Phase 1, not deferred.
- The `classification` and `idempotent` action schema additions must land here so Phase 1's halt-on-timeout messages (§E.3 Option A) can reference the action's classification.

Phase 1 scope (ours):

| Work item | From your ask | Modified? |
|-----------|--------------|-----------|
| Runtime profile schema | §6 | Yes — add per-tool overrides (§D) |
| Binding resolver | §5.2 | Yes — extends `--package-map`, not replaces |
| Managed identity AuthProvider | §8.2 | Unchanged |
| Tiered preflight (Tier 0 mandatory, Tier 1/2 opt-in) | §10.1 | Redesigned (§C) |
| Corrected error taxonomy (5 codes) | §10.2 | Redesigned (§C) |
| `gert plan` command | Not in your ask | Added |
| `classification` + `idempotent` additive schema fields | Not in your ask | Added (§B) |
| Halt-on-timeout with classification-aware messages | §E.3 Option A | Added |
| ICM proof on headless server | Your Phase 1 | Unchanged |

### Phase 2 — Host Bridge + VS Code (6–10 weeks, not 4–6)

Your 4–6 week estimate covers only the Gert-core side. The extension-side adapter, framing protocol, capability advertisement, and cancel channel are unaccounted for. Additionally, the estimate is unreliable until OQ2 (library vs subprocess) is answered in Phase 0. We will re-estimate after Phase 0 delivers a binding decision.

Before Phase 2 starts, the framing protocol document (§F.6) and host capability advertisement schema (§F.2) must be specified. We will not implement against an unspecified IPC channel.

### Phase 3 — Workload Identity + Hardening (4–6 weeks, not 3–4)

Your estimate omits unestimated transport work: `MCPHTTPTransport` today has no `request_id` on outbound JSON-RPC, no idempotency key, no late-result discard. That is new transport plumbing, not just a new AuthProvider. Add the late-result classification-aware handling from §E.1, re-approval on reconnect from §E.2, and token isolation enforcement from §F.7.

---

## §I. What We Commit To

1. **Immediate:** Wire `AllowedEnvironments` / `RequiresCapabilities` enforcement into `resolveTool()`. Confirm you want this and we start this week.
2. **Phase 0 (2 weeks from agreement):** Spikes as revised above. Deliverable: spike report + binding decision on OQ2 + profile schema draft.
3. **Phase 1 (6–8 weeks after Phase 0):** Resolver + managed identity + `gert plan` + tiered preflight + ICM proof.
4. **Phase 2 (re-estimated after Phase 0):** Host bridge + VS Code adapter. Scoped after OQ2 is resolved.
5. **Phase 3 (4–6 weeks after Phase 2):** Workload identity + reconnect/idempotency + hardening.

## What We Are Not Committing To Yet

- **Device code auth (§8.4).** Lower priority than the five-context matrix. Deferred indefinitely.
- **Profile inheritance (OQ4).** Rejected in favor of per-tool overrides.
- **Streaming/progress for the host bridge (§F.5).** Advisory, not Phase 2 scope. Revisit in Phase 3 hardening.
- **Multi-cloud auth.** Agreed — out of scope, per your §13.

---

## Next Steps

1. Confirm you accept the additive schema extensions (§B). This is blocking.
2. Confirm you accept tiered preflight (§C) replacing §10.1. This is blocking.
3. Confirm you accept halt-on-timeout for Phase 1 mutating/destructive actions (§E.3 Option A or B). This is blocking.
4. Confirm you want the `AllowedEnvironments` early win started immediately.
5. Answer OQ2 (library vs subprocess) with your extension architecture constraints, or confirm you will evaluate it as part of Phase 0.
6. Correct the `run-gert.ps1` attribution.

We are ready to start the early win and Phase 0 upon agreement on the blocking items.

---

*Gert Core Team — 2026-08-16*


---

# Runtime Portability — Final Impact Assessment (Three Questions)

**Prepared by:** Don (Backend Dev)
**Date:** 2026-08-17T06:15:17-07:00
**For:** Barbara (close-out ruling)

---

## Q1 — Attendance: What Does Declaring It Actually Cost?

### Current wiring — confirmed

`buildApprovalGate()` body in `internal/adapter/wire.go:319`:

```go
func buildApprovalGate(opts WireOptions) governance.ApprovalGate {
    if opts.TTYOutput {
        return internalgovernance.NewTerminalApprovalGate(os.Stdin, os.Stdout)
    }
    return internalgovernance.NewNoOpApprovalGate()
}
```

`TTYOutput` is a plain `bool` field on `WireOptions` (`internal/adapter/options.go:25`). It is NOT auto-detected from terminal state — it is a **hardcoded constant** at every call site:

| Call site | Value set | Source |
|-----------|-----------|--------|
| `cmd/gert/run.go:148` | `TTYOutput: true` | Hardcoded — `gert run` always gets Terminal gate |
| `cmd/gert/serve.go:110` | `TTYOutput: false` | Hardcoded — `gert serve` always gets NoOp |
| `cmd/serve/main.go:83` | `TTYOutput: false` | Hardcoded — standalone serve binary |
| `internal/e2e/helpers_test.go:171` | `TTYOutput: false` | Hardcoded — test harness always non-interactive |
| `internal/perf/fixtures.go:102` | `TTYOutput: false` | Hardcoded — perf fixtures |

**`gert run` hardcodes `TTYOutput: true` regardless of whether an actual TTY is attached.** There is no `isatty()`/`term.IsTerminal()` detection anywhere. A CI pipeline running `gert run` gets the Terminal gate and blocks on stdin — a bug that profile-declared attendance would fix.

### What else `TTYOutput` drives beyond gate selection

`TTYOutput` branches in TWO additional places in `wire.go`:

1. **`buildInputProvider()` (line 305):** `if opts.TTYOutput { promptInput = internalinput.NewPromptProvider(os.Stdin, os.Stdout) }` — whether stdin/stdout are wired as an input provider for collector/choice steps.

2. **`buildPromptProvider()` (line 312):** `if !opts.TTYOutput { return nil }` — whether a `TerminalInputProvider` is constructed for human-in-the-loop prompt steps.

These are NOT the same semantic as "is a human present to approve." A non-interactive CI run may still want structured prompt inputs from env vars (the `ChainProvider` already handles that via `envProvider` regardless of TTYOutput). **This means `TTYOutput` currently conflates three distinct properties: (a) approval attendance, (b) stdin/stdout terminal interactivity for collector steps, and (c) prompt rendering.** Decoupling them is right.

### Recommended shape

Add `Attended *bool` to `WireOptions`. When the profile declares `attended: true` or `attended: false`, set it explicitly. When absent, derive from TTY detection (or keep the legacy `TTYOutput` default). The gate selection becomes:

```
if opts.Attended != nil {
    attended = *opts.Attended  // profile wins
} else {
    attended = opts.TTYOutput  // legacy inference
}
```

The stdin/stdout wiring for prompt/input providers should remain driven by `TTYOutput` (it's about whether physical IO streams are usable, not about approval semantics). These are now separate switches.

**Call sites to update:** `buildApprovalGate()` — 1 function, 3 lines changed. No other logic changes. The `WireOptions` struct gets one field. The profile loader populates it when profile has `attended:` declared. `cmd/gert/run.go` sets `TTYOutput: true` and leaves `Attended` nil (inherits `TTYOutput` inference). CI profile sets `Attended: false` explicitly, regardless of whether `gert run` was invoked with a TTY.

**Effort: 0.5 days.** Field addition, gate selection update, profile loader sets the field. No other structural changes.

---

## Q2 — Test-Context Binding Restriction: Statically Enforceable?

### Is `transport.mode` available on `plan.Tools` at plan time?

Yes. `internalplanner.Plan()` returns `plan.Tools map[string]*schema.ToolDef`, and each `ToolDef.Transport` is a `TransportConfig` struct with `Type schema.Transport` (the `mode` value, already resolved from the YAML `mode:` field via `UnmarshalYAML`'s `Mode → Type` copy). This is populated by the catalog resolution + `RuntimeToolDef()` conversion path, not by any transport runtime. **The transport mode is fully available at plan time without executing anything.** The test-context check lands squarely in the early `gert plan` work.

### Is "must be native" too strict?

Yes — too strict. Checking the repo's own test/conformance posture:

- The **e2e test harness** (`internal/e2e/helpers_test.go`) injects a `mockToolRuntime` that overrides dispatch entirely. Transport mode in the tool YAML is irrelevant — the mock runtime intercepts before any transport runs.
- The **conformance corpus** (`tv-enum.yaml`) uses `transport.mode: stdio` on all 25 fixture tool definitions (the same `kubectl` tool with `allowed-environments: ["real"]`). Transport mode in a conformance fixture is structural metadata, not execution intent.
- **No fixture or test in the Gert repo uses `transport.mode: mcp` (subprocess) in a test context.** Zero matches across test YAML files.

However, the right rule must account for a legitimate use case the SQL team themselves will encounter: **a hermetic local test server** — a deterministic fake MCP subprocess (`fake-icm-mcp`) that responds predictably without hitting a real endpoint. A strict "native only" rule would ban this valid testing pattern. The SQL team's `tests/packages/incident-routing-mock/` is already `transport.mode: native`, which is correct for deterministic runbook-backed mocks. But a team running a local fake subprocess for integration testing of the MCP transport layer itself needs `mode: mcp` to be allowed.

**Proposed corrected rule for `context: test`:**

| Transport mode | Default | Override |
|----------------|---------|----------|
| `native` | ✅ Always allowed | — |
| `mcp` (subprocess) | ❌ Blocked by default | Allowed with explicit profile opt-in: `transport.allow_subprocess_in_test: true` |
| `mcp-http` | ❌ Never allowed in test context | No override — real HTTP endpoints are production |

The opt-in for subprocess keeps the fail-closed default (test = native, deterministic) while not breaking the legitimate "hermetic local fake server" pattern. The key invariant: `mcp-http` is always banned in test context — that's the actual production-endpoint protection the SQL team cares about.

### Interaction with `--package-map`

The check is exactly the right layer. `--package-map` resolves first (determines which `.tool.yaml` backs each toolRef). The profile test-context check reads `plan.Tools` after catalog resolution. If a test-context profile is combined with a `--package-map` pointing at the real production package (which declares `transport.mode: mcp-http`), the check reads `mcp-http` from the resolved `plan.Tools` entry and fires a Tier 0 error: "test context profile does not permit mcp-http transport for tool 'icm'." Exactly the mistake it should catch. The layering makes this work for free.

**Effort: 0.5 days.** The check is a loop over `plan.Tools` in the new `gert plan` command. One function, ~15 lines. Ships with the early `gert plan --profile` work.

---

## Q3 — Interactive `unspecified` Fires the Gate: Migration Hazard

### Is the regression real?

Yes, and it is significant. Here is the actual scope:

- The Gert core repo has 4 real `.tool.yaml` files in `tools/`. `ping`, `nslookup`, and `curl` use the legacy map-form actions (no `- name:` prefix) and have no `classification` or `requires-approval` — they would become `unspecified`.
- The `icm.tool.yaml` in `tools/` has 3 actions via the canonical list form, no `classification`.
- The conformance corpus (`tv-enum.yaml`) has 75 tool actions embedded as fixture strings; 25 of those have `requires-approval: false` explicitly set, the remaining 50 have neither field.
- **The gert-sqllivesite production tools (outside this repo) are entirely unclassified.** Every one of their `icm`, `tsg-recommendation`, and any other tool actions would become `unspecified` on day 1 of classification shipping.

However, there is an important nuance from the current approval wiring: the gate for plain (non-substitution) tool calls in interactive mode currently fires based on **runbook-level governance** (`engine.go:506` — `evalResult.RequiresApproval` from `GovernancePolicy`), not on tool-level `requires-approval`. Tool-level `RequiresApproval` is only checked on the substitution path. So today, a plain `icm.get-incident` call in interactive mode does NOT prompt — there is no gate. Under the new rule, if `classification` is absent and we fire the gate, every plain tool call suddenly prompts. For a runbook with 10 tool steps, that is 10 approval prompts where there were 0 before. **This is a breaking UX regression for every existing interactive runbook.**

### Recommendation: Grandfathering via explicit `requires-approval: false`

The SQL team's principle is right: classification is an opt-in to REDUCED friction. Absence must not grant additional execution rights. But they also said "existing `RequiresApproval` behavior remains authoritative for legacy definitions." These two principles together define the migration path.

**Proposed rule:**

1. `requires-approval: false` explicitly set on a tool action (or inherited from the tool's governance block) acts as a **legacy classification override** equivalent to `classification: read-only` for the purpose of the classification gate. The action is treated as read-only: no prompt, no approval evidence required. This is the grandfathering clause.

2. `requires-approval: true` explicitly set continues to fire the gate as today (unchanged).

3. `classification` absent AND `requires-approval` absent (or default `false` not explicitly set — the ambiguity from `omitempty` on bool) → `unspecified`. In interactive contexts: fires the gate. In test contexts: auto-approve only for native transport (per Q2). In unattended contexts: deny.

4. A profile-level field `legacy_unspecified_policy: allow | prompt` that defaults to `prompt` in new profiles and can be set to `allow` for a migration window. This lets existing deployments opt in to the old behavior for a deprecation period, explicitly, not by default.

**Why grandfathering on `requires-approval: false` is the right anchor:** It is already an explicit author signal. The author of `kubectl.tool.yaml` who wrote `requires-approval: false` was explicitly saying "this tool does not need human approval." Under the proposed scheme, that explicit signal is preserved without requiring tool authors to touch their YAML. The gate fires only for tools where the author made no explicit decision either way.

**The practical migration checklist:**
1. Ship `Classification *string` on `ToolAction`.
2. Ship `legacy_unspecified_policy` on the profile schema with default `prompt` and a clear deprecation notice.
3. Emit a `PKG-W` warning (non-fatal, stderr) for every unclassified action encountered at plan time when the profile is non-test: "action 'icm.get-incident' has no classification; treating as unspecified. Add `classification: read-only` to suppress this warning."
4. Existing deployments that need continuity set `legacy_unspecified_policy: allow` in their profile during the migration window. They see the warnings but don't get prompted.
5. Set a deprecation deadline: profiles that don't set `legacy_unspecified_policy` get `prompt` behavior once `classification` ships, with the explicit opt-out available.

This gives the SQL team their principle (unspecified fires the gate by default in new profiles) while giving existing runbook operators a named, explicit, auditable escape hatch — not a silent reversal of the principle.

**Effort for Q3 plumbing:** `Classification *string` addition: 0.5 days. `legacy_unspecified_policy` field on profile schema + routing in gate: 0.5 days. PKG-W warning at plan time: 0.5 days. **Total: ~1.5 days**, plus documentation of the migration path.


---

# Implementation Impact Assessment: Runtime Portability Counter-Positions

**Prepared by:** Don (Backend Dev)
**Date:** 2026-08-16T16:52:05-07:00
**For:** Barbara (ruling) and Gert Core Team
**Re:** SQL Live-Site Operations counter-positions (1) classification default, (2) fail-closed unattended gate, (3) environment identifier semantics + `gert plan` early delivery

Source read: `C:\One\OpenSource\gert` (read-only).

---

## A. Environment Vocabulary Collision (HIGHEST PRIORITY — blocker on the early win)

### Every occurrence of the two fields across the entire repo

A full-repo search across `.go`, `.yaml`, `.yml`, `.md`, and `.tex` files finds:

**`allowed-environments` / `AllowedEnvironments`:**
- `pkg/schema/tool.go:45–46` — declaration only.
- `internal/conformance/enumdata/tv-enum.yaml` — **25 occurrences**, every single one the same value: `["real"]`.
- **No other file anywhere in the repo contains either spelling.**
- In Go code: the field is declared in `ToolGovernance` and is carried through YAML unmarshal. Zero Go code ever reads or acts on it.

**`requires-capabilities` / `RequiresCapabilities`:**
- `pkg/schema/tool.go:45` — declaration only.
- **Zero occurrences anywhere else in the entire repo.** No test data, no fixtures, no conformance corpus, no docs.

### What the value `"real"` means

The conformance fixture (`tv-enum.yaml`) defines the `kubectl` tool in a mock package with `allowed-environments: ["real"]`. Reading the context around it and the `RunMode` constants in the codebase, `"real"` is a RunMode discriminator value — it appears to mean "this tool must not run under `dry-run` mode" (i.e., the tool has real side effects). The value maps semantically to `engine.RunModeReal` (`"real"` / `"dry-run"` / `"replay"`), NOT to a deployment context. This is confirmed by the fact that zero test data uses any other value and the field was never wired to anything: whoever wrote the schema field in `ToolGovernance` left a note for the enforcement to come later, using a value that parallels the only runtime-mode distinction Gert already has.

### Vocabulary collision with the proposed profile contexts

The SQL Live-Site ask proposes profile `context` values: `cli-operator`, `vscode-operator`, `ci`, `headless-server`, `test`. These are deployment/execution-context identifiers. `"real"` is a RunMode indicator. **These are orthogonal axes:**

| Axis | Example values | Meaning |
|------|---------------|---------|
| RunMode (existing) | `real`, `dry-run`, `replay` | Whether side effects execute |
| Profile context (new) | `cli-operator`, `headless-server`, `test` | Where/how Gert runs |

A value like `allowed-environments: ["real"]` in the current fixtures says "don't dry-run me." It says nothing about whether the tool works on a headless server vs. a developer workstation.

### Recommendation on the value space for `AllowedEnvironments`

**Do not reuse `"real"` as a profile-context identifier.** The existing value must be treated as a RunMode qualifier, not a deployment context. There are two clean paths:

**Option A — Separate axes, two fields:**
Rename (or re-purpose) `AllowedEnvironments` explicitly for profile contexts, and add a separate `AllowedModes` (or enforce the RunMode check via existing DryRunExecutor logic). Then populate `AllowedEnvironments` with the profile-context vocabulary: `cli-operator`, `headless-server`, `ci`, `test`, etc.

Migration: any tool currently using `allowed-environments: ["real"]` must be re-evaluated — the intent was likely `allowed-modes: ["real"]`, not a context restriction. Since the field is unenforced today, there is no runtime breakage. The 25 fixture occurrences are all in one test file (`tv-enum.yaml`) which the Gert team controls.

**Option B — Keep one field, define `"real"` as a reserved legacy alias:**
Define `"real"` as meaning "this tool may not run in dry-run or replay mode." All other values are profile-context identifiers. This is expedient but creates a mixed-axis vocabulary in one field. Not recommended.

**My recommendation: Option A.** The field is unenforced, so there is no backwards-compatibility cost. Define `AllowedEnvironments` as a profile-context allowlist with values matching exactly the profile's declared `context` field (confirming the SQL team's assumption in counter-position (3)). Add a separate `AllowedModes: []string` to `ToolGovernance` for the RunMode dimension. The 25 fixture occurrences in `tv-enum.yaml` need a one-line update to `allowed-modes: ["real"]` — that is the full migration scope.

**Adopted environment identifier vocabulary (proposed):**
`cli-operator`, `vscode-operator`, `ci`, `headless-server`, `test`

These match the SQL team's profile `context` field values exactly. A tool with `allowed-environments: ["headless-server", "ci"]` would only be planner-reachable when the selected profile declares `context: headless-server` or `context: ci`.

### Analysis for `RequiresCapabilities`

Zero values exist anywhere in the codebase. The field is a completely blank slate. No defined vocabulary, no design document, no comment beyond the struct declaration. It is not analogous to `AllowedEnvironments` — where there were at least 25 real uses with a consistent value — it has truly never been used.

**What a "capability" is here is undefined.** It could mean:
- Transport-level capabilities (e.g., `mcp-http`, `subprocess`),
- Auth capabilities (e.g., `azure-cli`, `managed-identity`),
- Host capabilities (e.g., `interactive-prompt`, `headless`), or
- Something else entirely.

**Recommendation:** Don't touch `RequiresCapabilities` in Phase 1. Define it only after the profile schema and binding resolver are designed, at which point "capabilities" will have a concrete referent (the profile's declared transport/auth capabilities). Using it now without a defined vocabulary creates another ghost field.

---

## B. Unspecified-Classification Implementation Impact

### Where classification needs to be read

There is no `classification` field anywhere in the codebase today. Adding it means touching:

1. **Schema** (`pkg/schema/tool.go` — `ToolAction` struct): Add `Classification string` (or typed) field on `ToolAction`, not on `ToolGovernance`. Classification is per-action (an `icm` tool may have a read-only `get-incident` and a mutating `close-incident`). The `ToolGovernance` block is per-tool, not per-action. This is a schema extension.

2. **Approval gating in `executor/tool.go:Execute()`**: The plain (non-substitution) path at line ~95 calls `e.runtime.Invoke(ctx, toolName, action, args)` with no classification check whatsoever — there is no gate on the direct invocation path. The `RequiresApproval` gate only exists on the substitution path (`executeSubstitution`, line ~147). **This means the existing `RequiresApproval` is only enforced for `execute.kind: runbook` actions today, not for plain process-backed or HTTP-backed tool actions.** Classification-gating for plain actions would require a new check in `Execute()` before `e.runtime.Invoke()`.

3. **Substitution path in `executor/tool.go:executeSubstitution()`** (line ~195): Already checks `planResult.EffectiveGovernance.RequireApproval` and calls `e.approvalGate.RequestApproval()`. Classification-based gating would be added here alongside or as a replacement for the bool check.

4. **Preflight / planner**: `internal/planner/planner.go:resolveTool()` is the natural place to add a classification-vs-profile check at plan time. This requires the profile to be threaded into the planner context.

5. **Retry/late-result logic in transport layer**: Does not exist yet. `MCPHTTPTransport` has no retry semantics today other than the one 401-retry and the 404-reinit cycle. No retry-on-idempotent-only logic exists anywhere. This is new infrastructure. The SQL team's requirement that unspecified actions must NOT get automatic retry is a guard against future retry logic being added naively — there is nothing to change today, but the constraint must be built in from the start when retry is added in Phase 3.

### `RequiresApproval` (bool) × `classification` (string) — precedence table

The critical finding above: `RequiresApproval` is today only enforced on the substitution path. For this precedence table, I'm defining what the safe rule should be for all paths once classification is added:

| RequiresApproval | Classification | Safe rule |
|-----------------|----------------|-----------|
| `true` | `read-only` | Gate fires (explicit override wins; a tool author who marks a read-only action as requires-approval probably has a reason) |
| `true` | `mutating` or `destructive` | Gate fires |
| `true` | `unspecified` | Gate fires (legacy: existing behavior preserved) |
| `false` | `read-only` | No gate |
| `false` | `mutating` | Gate fires if runtime profile's approval mode is `interactive` or stricter; no gate in test/mock |
| `false` | `destructive` | Gate always fires |
| `false` | `unspecified` | **Fail closed in unattended contexts.** In interactive contexts: treat as legacy `requires-approval: false` (no gate, with a PKG-W warning that classification is missing). See below. |
| not set (omitempty) | `unspecified` | Same as `false`+`unspecified` |

**Safe rule for the `RequiresApproval` bool:** It remains the authoritative gate for legacy tools that have it set. For new tools, classification is the primary mechanism. If classification is `unspecified` AND `requires-approval: false` (or absent) AND the runtime profile is unattended, the unattended gate should DENY — the SQL team's requirement is satisfied without changing legacy tools that have `requires-approval: true` (those keep working as before).

### `unspecified` in the existing schema types

`ToolAction` does not currently have a classification field at all. Adding one:

```go
// In pkg/schema/tool.go, ToolAction struct:
Classification string `yaml:"classification,omitempty" json:"classification,omitempty"`
```

With `omitempty`, YAML unmarshalling produces an empty string `""` for a missing field. An empty string is distinguishable from an explicit `""` only if we use a pointer or a sentinel. **Using a pointer (`*string`) is the right representation here:** `nil` means genuinely absent ("unspecified"), while a pointer to `""` would mean explicitly empty (which we'd treat as an error). For the bool `RequiresApproval`, `omitempty` on a `bool` does the right thing — `false` and absent are indistinguishable, which is why the legacy field works as-is.

**Concrete recommendation:** `Classification *string` on `ToolAction` with `yaml:"classification,omitempty"`. Recognized values: `"read-only"`, `"mutating"`, `"destructive"`. `nil` = unspecified. This is cleanly representable with the existing YAML parsing machinery (`omitempty` on a pointer causes absent fields to unmarshal as `nil`).

---

## C. Fail-Closed Unattended Approval Gate

### Current gate implementations — exact signatures and fields

**Interface** (`pkg/governance/approval.go:9`):
```go
type ApprovalGate interface {
    RequestApproval(ctx context.Context, stepID string, reason string) (ApprovalRecord, error)
}
```

**`ApprovalRecord`** (`pkg/governance/evidence.go:33`):
```go
type ApprovalRecord struct {
    Approver   string `json:"approver"`
    ApprovedAt string `json:"approved_at"`
    Token      string `json:"token"`
}
```

**`NoOpApprovalGate`** (`internal/governance/approval.go:17`): Always returns `ApprovalRecord{Approver: "noop-gate", ApprovedAt: ..., Token: <uuid>}`. Never denies. No awareness of classification, profile, or action semantics.

**`TerminalApprovalGate`** (`internal/governance/approval.go:35`): Prompts on stdin, reads approver identity, returns record. Blocks until input or ctx cancellation. No awareness of classification.

**Gate selection** (`internal/adapter/wire.go:buildApprovalGate()`):
```go
func buildApprovalGate(opts WireOptions) governance.ApprovalGate {
    if opts.TTYOutput {
        return internalgovernance.NewTerminalApprovalGate(os.Stdin, os.Stdout)
    }
    return internalgovernance.NewNoOpApprovalGate()
}
```

The binary is: TTY present → Terminal gate; no TTY → NoOp gate. There is no third option today. `gert serve` sets `TTYOutput: false` and gets the NoOp gate. A headless server running `gert run` also gets NoOp via the same path.

### What the SQL team's evidence requirement adds

They require the approval record to carry: **policy ID, approving identity, run ID, step ID, retry count, and classification**. Current `ApprovalRecord` has only: `Approver`, `ApprovedAt`, `Token`. Missing fields: `PolicyID`, `RunID`, `StepID` (stepID is passed as a parameter to `RequestApproval` but not stored in the record), `RetryCount`, `Classification`.

Since `ApprovalRecord` is a value type embedded in `Evidence` and written to the trace JSONL, extending it is a **backward-compatible additive change** — new fields with `omitempty` do not break existing trace readers. The interface signature does not change (the record is returned, not a parameter). This is a small, low-risk change.

### Effort estimate for two options

**Option (i) — Phase 1 minimal: fail-closed unattended gate that DENIES all mutating/destructive/unspecified actions outright.**

This is a new gate type: `UnattendedApprovalGate`. Implementation:
- Receives an action classification (needs to be threaded into the call site — `RequestApproval` signature passes `stepID` and `reason` today, not classification). Either add classification to the `reason` string (expedient) or extend the interface (cleaner but requires updating all three existing call sites).
- If classification is `read-only`: return approval without a record (read-only actions don't need an approval gate).
- If classification is `mutating`, `destructive`, or `unspecified`: return error (deny).
- Effort: **1–2 days** for the gate itself, plus 1 day to thread the profile selection into `buildApprovalGate`. No evidence infrastructure needed.

**Option (ii) — Full evidence-recording gate as specified.**

- Extend `ApprovalRecord` with `PolicyID`, `RunID`, `StepID`, `RetryCount`, `Classification` fields.
- `UnattendedApprovalGate` checks the runtime profile for allowed action classifications, records evidence, writes a `governance/approvalGranted` trace event.
- Requires profile to be threaded from CLI flags through `WireOptions` to `buildApprovalGate` and into the gate implementation.
- Effort: **1 week** (gate + record extension + profile threading + trace event definition + tests).

**Recommendation for Phase 1:** Build Option (i) (fail-closed deny) with a clean extension point. The full evidence gate is Phase 3 work (the SQL team themselves put "approval evidence for unattended" in Phase 3 at §11.3). Phase 1 needs the gate to not be a silent yes — that is Option (i). The full audit trail can follow.

One important note: `RequiresApproval` enforcement currently only fires on the substitution path (`executeSubstitution`, `executor/tool.go:~195`). Plain tool invocations (direct `e.runtime.Invoke`) have no gate at all today. The Phase 1 gate must add a classification check to the non-substitution path in `Execute()` — otherwise a plain `mcp-http` tool call with `classification: destructive` would bypass the gate entirely.

---

## D. `gert plan --profile` Early Delivery Feasibility

### What `gert preview` already does

There is currently no `gert plan` command. The closest is `gert preview`, which parses a runbook, optionally recurses through includes, and renders it as prose/mermaid/graphjson. It does NOT plan (no tool resolution, no planner step, no execution plan structure). The underlying `internalplanner.Plan()` is only invoked inside `gert run` and `gert dry-run`.

### What Plan() already produces

`internalplanner.Plan()` returns `*engine.ExecutionPlan` which contains:
- `Steps []engine.ResolvedStep` — every step resolved and flattened
- `Tools map[string]*schema.ToolDef` — **the resolved tool set**, keyed by name, with full transport config, governance, and action definitions
- `Metadata.RunbookID`, `PlannedAt`, etc.

**This is the key finding:** the plan already contains the complete resolved tool set. A `gert plan --profile <id> <runbook>` command can:
1. Load the profile YAML.
2. Run `internalplanner.Plan()` using the existing wiring (which already resolves toolRefs via the package catalog and populates `plan.Tools`).
3. For each tool in `plan.Tools`, compare the profile's declared context against the tool's `AllowedEnvironments`, compare the tool's `transport.mode` against what the profile's transport section declares, and report whether the binding resolves.
4. Print a binding table: tool name → transport mode → auth provider → profile match status.

**This is genuinely shippable before the full binding resolver exists.** The "binding table" is computed from data Plan() already returns. The profile loading is a new YAML parser (small). The match logic is a simple loop over `plan.Tools`. No new executor, no transport dispatch, no ApprovalGate changes needed.

**What it cannot do yet (and must be labeled clearly):**
- It cannot validate that the transport endpoint is reachable or that auth can actually obtain a token — those are runtime probes, not plan-time checks.
- It cannot inject profile-overridden transport configs — the tool YAMLs in phase 1 still bake the transport in. The plan output shows what the tool YAML declares, not what the profile would override (that's the resolver work).
- It cannot validate `AllowedEnvironments` enforcement semantics until the enforcement is wired into the planner.

**Honest label for early delivery:** call it `gert plan --profile <id> <runbook>` and document it as "binding compatibility report — shows whether each tool's declared transport and environment constraints are compatible with the selected profile. Does not validate endpoint reachability or auth token acquisition." That is accurate and useful without being misleading.

**Effort for the early command:** 2–3 days. It's a new top-level command in `cmd/gert/main.go`, a YAML loader for `runtime-profile/v1`, a loop over `plan.Tools`, and a formatted output. The existing `internalplanner.Plan()` call and `adapter.BuildPackageCatalog` wiring from `run.go` can be reused nearly verbatim.

---

## E. `--profile` / `--package-map` Composition

### How `--package-map` currently works

`cmd/gert/run.go` loads two config/v1 files: the project's `.gert/config.yaml` and optionally the `--package-map` file. Both have the same schema shape (`requires: []PackageRequirement`, `tool-paths: []string`). `mergePackageBindings()` in `packagemap.go` merges them: **package-map entries win over project entries for the same package name** (CLI precedence). The merged list then drives `adapter.BuildPackageCatalog()`, which resolves which `.tool.yaml` file on disk backs each toolRef.

**Effect:** `--package-map` determines which YAML file provides the tool definition for a given logical name. It redirects at the package/file level. It does NOT override the transport block inside that YAML.

**Trace provenance:** `package/resolved` trace events include `"origin": "project" | "package-map"` so auditors can see when a package-map override was used. This is already implemented.

### Where the collision occurs

If a profile selects a per-tool transport binding (e.g., "use `mcp-http` with managed identity for this tool") AND a `--package-map` redirects the same toolRef to a completely different `.tool.yaml` (e.g., a mock package with `transport.mode: native`), the question is: which wins?

**The natural precedence from the code structure:** `--package-map` operates at the catalog/registry layer (which YAML file to load). The profile binding resolver will operate at the transport-config layer (which transport config to use for an already-loaded tool). These are different abstraction layers.

**Recommended rule for Barbara's reply:**

> `--package-map` wins at the YAML-selection layer: it determines which tool definition file backs each toolRef. The runtime profile's transport binding applies after YAML selection: it can override the transport/auth config of the resolved definition, but it cannot override which logical toolRef maps to which package. If a `--package-map` redirects `icm` to a mock package, the profile sees the mock tool definition — not the production one — as its binding target. This means `--package-map` and `--profile` compose without conflict when used for their respective intended purposes: `--package-map` for package-level substitution (mock vs. real), `--profile` for transport/auth/context selection within the selected package's tool definitions.

**The one conflict case:** if someone uses `--package-map` to redirect a toolRef to a mock (native transport) AND selects a headless-server profile that requires `mcp-http`, the plan-time binding check will correctly report a mismatch: the tool as loaded has `transport.mode: native`, the profile requires `direct-http`. This is the right behavior — the operator explicitly chose an incompatible combination.

**Implementation implication:** the binding resolver should operate on the plan's already-resolved `Tools` map (populated after `--package-map` merging and catalog resolution), not on raw toolRef declarations. This is consistent with building `gert plan` on top of `internalplanner.Plan()` output (section D).

---

## Summary Table

| Question | Finding | Action required |
|----------|---------|----------------|
| A: vocabulary collision | `"real"` is a RunMode value, not a profile context. New vocabulary needed. | Define two fields: `AllowedEnvironments` (profile contexts) + `AllowedModes` (RunMode). Update 25 fixtures in `tv-enum.yaml`. |
| A: RequiresCapabilities | Zero real values anywhere. Undefined vocabulary. | Leave for post-Phase 1 design. |
| B: classification field | Does not exist. Must add `Classification *string` to `ToolAction`. | Schema addition + enforcement in `Execute()` and `executeSubstitution()`. |
| B: RequiresApproval interaction | Bool only enforced on substitution path today. Plain tool calls have no gate. | Add gate to non-substitution `Execute()` path in Phase 1. |
| C: NoOpApprovalGate | Silently approves everything. Gate selection is binary (TTY/no-TTY). | Add `UnattendedApprovalGate` (deny-all for unattended). `ApprovalRecord` extension is backward-compatible. |
| C: effort | Minimal gate: 1–2 days. Full evidence gate: 1 week. | Phase 1: minimal. Phase 3: full. |
| D: gert plan | No `plan` command exists. `preview` doesn't plan. Plan() already returns resolved tools. | Early `gert plan --profile` is shippable in 2–3 days before binding resolver. Label it "compatibility report." |
| E: composition | `--package-map` operates at YAML-selection layer; profile at transport-config layer. Natural layering, no deep conflict. | Resolver should operate on post-catalog `plan.Tools`. Precedence rule: package-map wins YAML selection; profile wins transport config. |
