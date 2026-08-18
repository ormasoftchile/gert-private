# Don — History Summary

## Overview

**Stream:** Backend transport and authentication. Implemented MCP HTTP transport, verified Phase 1B auth gaps, delivered managed-identity provider and credential-leak assertions. Fixed dry-run regression and serve security issue. Phase 2 Rev 3: serve --package-map + declarative vscode_input adaptation. TSG binding fail-closed (VSCODE-003).

**Key Contributions:**
- MCP HTTP transport implementation (Phase 1A)
- Phase 1B Item 1: Managed-identity auth provider (2026-08-17)
- Phase 1B Item 6: Credential non-leakage assertion suite (2026-08-17)
- Live defect: dry-run profileless fail-fast regression (2026-08-17, commit d19f933)
- Live defect: serve all-interface bind without auth (2026-08-17, commit a6b1106)
- Phase 2 Rev 3 Deliverable A: serve --package-map (2026-08-18, commit 4da94b2)
- Phase 2 Rev 3 Deliverable B: declarative vscode_input adaptation (2026-08-18, commit 4da94b2)
- TSG binding fail-closed + six logical args (2026-08-18, gert commit 1824af7, private commit b598128)

---

## TSG Filename Rename + Parity Script Fix (2026-08-18)

### Problem
`tsg-recommendation.vscode-mcp.tool.yaml` was named for a binding it no longer
declares (transport removed). The `.vscode-mcp.` infix implied a VS Code MCP
binding; the file's own description said "No transport is bound." Also, the
parity script's `.endsWith('.vscode-mcp.tool.yaml')` filter would silently
drop coverage if any file was renamed.

### Rename
`tsg-recommendation.vscode-mcp.tool.yaml` → `tsg-recommendation.contract.tool.yaml`
on both sides via `git mv`. Updated `deliverable_contract_test.go` path reference.

### Parity script fix (bidirectional set comparison)
- Was: one-sided glob on `.vscode-mcp.` infix — coverage silently halved on rename
- Now: enumerate `.tool.yaml` from BOTH sides; fail if either side has a file
  the other lacks (MISSING-TESTDATA / ORPHAN-TESTDATA errors)
- `TESTDATA_SYNTHETIC_ONLY` exclusion set for intentional testdata-only fixtures
  (`vscode-mcp-ops-synthetic.tool.yaml`); argument for exclusion list over pure
  set comparison: testdata contains synthetic fixtures that are not deliverables;
  pure set comparison would false-fail on them
- `MIN_COMPARED=2` guard: comparison is never vacuously empty
- Printed union count so coverage drop is visible in CI logs

### Mutation controls (all measured)
| Scenario | Exit | Output |
|---|---|---|
| Rename deliverable only | 1 | MISSING-TESTDATA + ORPHAN-TESTDATA + min-count |
| Rename both consistently | 0 | 2 matched, 0 drifted |
| Delete deliverable | 1 | ORPHAN-TESTDATA + min-count |

### Commits
- gert: `374fef0` on `feature/tsg-logical-contract`
- gert-private: `0a7399d`

### Worktree
63 ok / 23 no-test / 0 FAIL, exit 0.

---



### Correction from Cristiano
Task 2 (VSCODE-003 schema field, fail-closed check, reachability probe) was
cancelled. tsg-recommendation is a SQL Live-Site tool, not a Gert deliverable.
Inventing new schema surface to describe their unresolved state was out of
scope. The platform guarantee already exists: absent tool → bridge returns
`tool_unavailable` → branch step cannot consume → run terminates (ba9d6d4).

### What was discarded
- `TransportConfig.ProviderUnresolved` field from `pkg/schema/tool.go`
- VSCODE-003 check from `validate_transport.go`
- Reachability registry entry and probe
- All changes to `deliverable_contract_test.go` from task 2
- Branch `feature/tsg-unresolved-provider-binding` (deleted, commit 1824af7)
- gert-private commits b598128, fa7c497 (reverted by 307c8f1)

### What was implemented
**Task 1 kept:** Six logical args added exactly as specified:
`incident_id`, `title`, `service`, `environment`, `logical_server`, `database`
— all `type: string`, `required: true`.

**Fabricated transport removed:** `vscode_tool: tsg-recommendation-recommend`
and `transport.mode: vscode-mcp` stripped from both files.

**Option chosen: (a) keep as logical-contract-only record (no transport key).**
Schema evidence: `ValidateTransportConfig` switch has no default case and no
empty-mode check — a file with no `transport:` key is schema-valid at
`ParseToolFile` level. `mapTransport` fails on empty mode but `testdata/` is
excluded from all scans (`scanSchemaDir` skips dirs named `testdata`).

### Files changed (gert, branch feature/tsg-logical-contract, commit 16e2d6e)
- `internal/tool/testdata/tsg-recommendation.vscode-mcp.tool.yaml`
- `internal/tool/deliverable_contract_test.go`

### Mutation control
Removed `database` arg from YAML → `arg "database" is missing from action 'recommend'` → FAIL (measured).

### Detached worktree
63 ok / 23 no-test / 0 FAIL, exit 0.

### Parity
`check-deliverable-parity.js` exit 0: 2 matched, 0 drifted.

---

## TSG Unresolved Provider Binding (2026-08-18, DISCARDED)

### Problem
The TSG deliverable had `vscode_tool: tsg-recommendation-recommend` — a guessed name, identical defect class to the `icm-get-incident` guess caught in Rev 2. SQL Live-Site confirmed the LOGICAL contract (six args) but could NOT supply the PROVIDER contract (registered VS Code MCP tool name + parameter schema).

### Approach
Added `TransportConfig.ProviderUnresolved bool` (`provider_unresolved: yaml key`) to `pkg/schema/tool.go`. When `true` on a `vscode-mcp` transport, `ValidateTransportConfig` returns VSCODE-003 at scan time.

**Earliest achievable fail point:** ValidateTransportConfig is called from ParseToolFile, which is called during scan (ScanDir/ScanSchemaDir). The tool never enters the registry, so planning and runtime are never reached. This is the same path used by VSCODE-001 (empty vscode_tool) and VSCODE-002 (invalid vscode_input).

**Why not runtime:** A runtime check would allow the tool into the registry and would reach the bridge. The scan-time check fires before any registry insertion.

**Error message distinguishes "unresolved" from "not found":**
> vscode-mcp: provider_unresolved is true — the VS Code MCP provider contract (registered tool name and parameter schema) has not been supplied for this binding. This is NOT a "tool not found" error: nobody has told us the real vscode.lm.tools name yet.

### Five-point chain
- `pkg/schema/tool.go` — `ProviderUnresolved bool` field declared
- `internal/tool/validate_transport.go` — VSCODE-003 check; also rejects `provider_unresolved` on non-vscode-mcp (mirrors VSCODE-001 pattern)
- `internal/tool/scan.go` — no change; already calls ValidateTransportConfig via ParseToolFile
- `internal/tool/runtime.go` — no change; never reached for provider_unresolved tools
- `cmd/gert/reachability_registry_test.go` + probe — `TransportConfig.ProviderUnresolved` entry with testCLI_ProviderUnresolved_Reachable

### Mutation control
Disabled the ProviderUnresolved check; measured:
- `TestCLI_ReachabilityGate/TransportConfig.ProviderUnresolved`: PASS → FAIL (got exitRuntime=1, wanted exitValidation=2)
- `TestDeliverableContracts_TSGRecommendation`: PASS → FAIL (ParseToolFile succeeded when it must fail)

### Parity
`check-deliverable-parity.js` exit 0: both icm and tsg deliverables matched testdata fixtures.

### Detached worktree
`go build ./...` exit 0, `go test ./...` exit 0: 63 ok / 23 no-test / 0 FAIL.

---

## Phase 2 Rev 3 Implementation Summary (2026-08-18)

### Deliverable A: gert serve --package-map (commit 4da94b2)

Added `--package-map` to `gert serve` with the same help text and `loadPackageMap` semantics as `run.go:71` (followed `run` not `plan`; run.go has the more complete help text). Loads at startup before any network operations (fail-stop: missing or malformed → exitValidation). Applies package-map `tool-paths:` to both the planner registry (`newServingToolRegistry`, variadic `extraPaths`) and the execution registry (`adapter.WireOptions.ExtraToolScanPaths` → `OverlayRegistry.Override`). Extra paths are scanned LAST so they override base-scan duplicates deterministically.

**Why run.go over plan.go:** run.go flag has `"Path to a package-map file (config/v1 shape: requires:/tool-paths:) overriding the project's .gert/config.yaml package bindings"` — more explicit, matches the full schema description. plan.go says `"overriding project package bindings"` — less precise.

### Deliverable B: declarative vscode_input adaptation (commit 4da94b2)

Five-point transport chain thread:
1. `pkg/schema/tool.go` — `VSCodeInputMapping` type + `VSCodeInput map[string]*VSCodeInputMapping` on `ToolAction`
2. `pkg/tool/tool.go` — `VSCodeInput map[string]*schema.VSCodeInputMapping` on runtime `ToolAction`
3. `internal/tool/validate_transport.go` — `ValidateVSCodeInputActions` (VSCODE-002 for wrong transport, from: undeclared arg, unsupported coerce:)
4. `internal/tool/scan.go` — `ValidateVSCodeInputActions` called from `ParseToolFile`; `VSCodeInput` carried through `RuntimeToolDef`
5. `internal/tool/runtime.go` — `applyVSCodeInputAdaptation` called before bridge wire in `TransportVSCodeMCP` case

New file `internal/tool/vscode_input.go`: `applyVSCodeInputAdaptation` + coerce helpers. Fail-closed: required/absent, unmapped arg, lossy coerce all fail before bridge request.

### Key Learnings

**OverlayRegistry.Override pattern:** The correct mechanism for priority overrides in the serve path. Extra-path tools are scanned into the overlay after the base scan, and `Override` (not `Register`) is used so they unconditionally win. This mirrors how catalog-resolved toolRefs win in the run path.

**ValidateVSCodeInputActions placement:** Must be called AFTER `ValidateVSCodeToolActions` in `ParseToolFile` so the transport mode is already confirmed clean before checking per-action vscode_input content.

---

## Learnings

### 2026-08-17: Dry-run and serve security fixes

**Dry-run fail-fast gate (commit d19f933):**
The Phase 1B fail-fast gate comment said "without a profile the engine installs TerminalApprovalGate which blocks in CI." This is empirically wrong. `buildApprovalGate` (internal/adapter/wire.go:330) installs `TerminalApprovalGate` only when `attended==true`; in non-TTY (CI) contexts it installs `NoOpApprovalGate`, which never blocks. In dry-run mode specifically, `DryRunExecutorRegistry` (internal/adapter/wire.go:121) replaces all executors with no-ops; governance/approval runs before executor dispatch (internal/engine/engine.go:592-604) but cannot block in non-TTY. The gate was over-broad. Fix: add `mode == engine.RunModeReal` guard. Phase 1B contract unchanged.

**Serve default bind (commit a6b1106):**
Changed `--addr` default from `:7778` (all interfaces) to `127.0.0.1:7778`. Added startup safety check: non-loopback + no auth → refuse with actionable error. The check lives in `runServe` (cmd/gert/serve.go) before any network operations; `internal/serve/server.go:165` net.Listen call is not touched (it faithfully binds what it's given, which is correct). Loopback + no auth is always safe (local dev, extension unaffected).

**Untracked-files incident (sixth occurrence, 2026-08-17):** When editing serve.go, I read the working-tree state (which already had `ExcludeTestTools` added by another agent) rather than the committed state. My a6b1106 committed that usage without committing the definition, breaking HEAD. Fixed by separate commit 06b0a83 with an explicit acknowledgment. Process note: always diff `git show HEAD:path` against the working tree before staging to detect in-flight changes from other agents.

## Phase 1A Work Summary

### MCP HTTP Transport Implementation (Complete)

**Files added/modified:** internal/tool/mcp_http.go, mcp_types.go, mcp_http_test.go; updated mcp.go, runtime.go, pkg/schema/tool.go, pkg/tool/tool.go.

**Context:** Used json.RawMessage for mcpResponse.Result to support both tools/call and tools/list response shapes.

---

## Phase 1B Implementation Summary

### Item 1: Managed Identity Provider (Commit 0ce2054)

**Status:** COMPLETE

- `ManagedIdentityAuthProvider` in internal/tool/auth_managed_identity.go
- IMDS HTTP endpoint only (no Azure SDK); zero ambient environment variable inspection
- Token caching with 5-minute refresh buffer, graceful-degradation on proactive refresh failure
- `imdsHTTPClient` function injection seam for test compatibility
- `NewAuthProviderWithClientID` constructor for future runtime binding (Item 2)
- 13 tests covering IMDS endpoint, timeouts, malformed JSON, cache behavior, context cancellation

**Key Design:** Determinism structural guarantee — no `os.Getenv` calls for AZURE_FEDERATED_TOKEN_FILE or other workload-identity env vars. Profile mechanism selection only, never ambient auto-selection.

### Item 6: Credential Non-Leakage Assertions (Commit c7decfd)

**Status:** COMPLETE

- `credentialSweeper` pattern collects all surfaces, unified scan at end (not per-surface)
- Non-vacuity assertions: every test proves code-under-test actually executed
- Permanent negative control `TestCredentialLeak_SweepDetectsIntentionalLeak` with 8 sub-cases (one per surface type)
- Surfaces covered: 13 total including trace events, ToolResult, errors, IndeterminateRecord, MCP-012 errors
- Tests live in internal/tool (no CLI layer needed)

**Key Design:** Error failure paths are highest-risk — four tests specifically target errors to prove they exclude HTTP bodies and full URLs.

---

## Design Rulings

### Managed-Identity Scope

Provider implementation must be deterministic across hosts. Environment-variable-based auto-selection (Correction 1) prevents identical profiles from behaving differently. `NewAuthProvider` switches on explicit string name only.

### Credential Sweep Invariant

All credential surfaces must be swept or test cannot pass. If sweeper breaks, negative control immediately fails CI. No vacuous "token did not appear" assertions without proof that token was actually used.

### EndpointHost Credential Invariant

`IndeterminateRecord.EndpointHost` must be parsed hostname only. No credentials, tokens, URLs, or query strings. Test verifies non-vacuously with high-entropy sentinel.

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


## Phase 1B Rev 3 Update (2026-08-17)

### Item 4 Ownership Restructure

**Status Update:** Item 4 ("Dual-Binding Mechanism Proof") no longer imports consumer contracts and is not artifact-gated.

- Gert core owns mechanism proof using synthetic fixtures
- Consumer-specific contracts (e.g., SQL Live-Site ICM contract) are owned by consumer team in their repo
- Item 4 is now serial after Item 2 only; no external dependencies
- Full details: Contract Proof Ownership Split decision (decisions.md)

---

## Learnings — Phase 1B Item 1: Managed Identity IMDS (2026-08-17)

### Implementation Complete

**Files shipped:**
- `internal/tool/auth_managed_identity.go` — `ManagedIdentityAuthProvider` implementing `Token(ctx)` and `Invalidate()`
- `internal/tool/auth_managed_identity_test.go` — 13 tests (httptest mock; happy path; user-assigned clientID; non-200; malformed JSON; context cancellation; cache hit; expiry-triggered refresh; IMDS unreachable; token-not-in-error; parseIMDSExpiry variants)
- `internal/tool/auth.go` — `NewAuthProvider` dispatches `"managed-identity"`; added `NewAuthProviderWithClientID` for optional client ID threading
- `internal/tool/validate_transport.go` — `knownAuthProviders` entry for `"managed-identity"` + updated MCP-002 error message
- `internal/tool/auth_azurecli_test.go` — Updated `TestNewAuthProvider_UnknownProvider` to use `"workload-identity"`; added `TestNewAuthProvider_ManagedIdentity` and `TestNewAuthProvider_WorkloadIdentityReturns002`

**Build and test:** `go build ./...` and `go test ./internal/tool/...` both pass (46 tests, 0 failures).

### Key Observations

1. **Caching pattern transplanted exactly.** AzureCLIAuthProvider's graceful-degradation pattern (still-valid token returned on proactive-refresh failure) is replicated verbatim. Both providers now use the same `tokenRefreshBuffer` constant from `auth_azurecli.go`.

2. **imdsHTTPClient seam.** Injecting `imdsHTTPClient func(*http.Request) (*http.Response, error)` let tests rewrite the URL to an httptest.Server without any interface bloat. This is the correct seam for a single-endpoint HTTP-only provider.

3. **No ambient detection.** The provider has zero `os.Getenv` calls. `AZURE_FEDERATED_TOKEN_FILE`, `AZURE_CLIENT_ID`, and similar workload-identity env vars are not read. The determinism requirement from Correction 1 is structurally enforced, not just by policy.

4. **Body exclusion from errors.** Non-200 and malformed-JSON errors surface the HTTP status code and endpoint URL only — never the response body. This is asserted by `TestManagedIdentity_Non200DoesNotLeakBody` which sends a body containing a fake token value and verifies the token string does not appear in the error.

5. **Pre-existing stray file issue.** During commit, two pre-staged files from other agents (`cmd/gert/reachability_probes_test.go`, `cmd/gert/reachability_registry_test.go`) and one more (`specs/reachability-gate.md`) appeared in the first commit attempt despite showing as `??` in status. Root cause: those files were already in the git index when I ran `git stash` to verify the pre-existing failure; the stash pop restored them as staged. Used `git rm --cached` + `git commit --amend` to produce a clean 5-file commit. **Lesson:** never run `git stash` when other agents have staged work in a shared index. For pre-existing test verification, use a separate worktree or just run the test from a known clean branch.

---

## Learnings — Phase 1B Item 6: Credential Non-Leakage Assertions (2026-08-17)

### Implementation Complete

**File shipped:** `internal/tool/auth_credential_leak_test.go`

**Tests (10 total, 17 sub-tests via negative control):**
- `TestCredentialLeak_SuccessPath_TokenNotOnAnyOutputSurface` — full provider→gate→transport→ToolResult path; proves token was delivered (auth header captured) but not on any output surface
- `TestCredentialLeak_Non200IMDS_ErrorDoesNotLeakToken` — non-200 IMDS body with embedded sentinel; error must not include the body
- `TestCredentialLeak_MalformedJSON_ErrorDoesNotLeakToken` — 200 OK with malformed JSON embedding the sentinel; JSON parse error must not echo it
- `TestCredentialLeak_ContextCancellation_ErrorDoesNotLeakToken` — IMDS hangs, context cancelled; cancellation error must not include partial response
- `TestCredentialLeak_MCP012_ErrorDoesNotLeakToken` — token pre-cached; request to disallowed host; MCP-012 must not include the cached token
- `TestCredentialLeak_TraceEvent_NoTokenInAuthAttached` — mcp/authAttached carries url_host+scope only; asserts event was emitted (non-vacuity)
- `TestCredentialLeak_IndeterminateRecord_NoToken` — correctly-constructed record sweeps clean; inline negative control for EndpointHost surface
- **`TestCredentialLeak_SweepDetectsIntentionalLeak` (8 sub-tests)** — MANDATORY NEGATIVE CONTROL; deliberately injects sentinel into every surface type and asserts sweep fires
- `TestCredentialLeak_TokenNotInStepOutput` — echo-args MCP server; even when server echoes call args, no token appears in ToolResult
- `TestCredentialLeak_Invalidate_DoesNotExposeToken` — failing re-acquire after Invalidate must not expose previous cached token in error

### Key Design Decisions

1. **`credentialSweeper` struct pattern.** Collecting all surfaces into a single `credentialSweeper` and calling `scan()` at the end gives a unified report naming every surface that leaked, rather than stopping at the first. An auditor can see all leak sites simultaneously.

2. **Non-vacuity checks in every test.** Every success-path test has an explicit assertion that the token WAS actually used (e.g., auth header captured, trace event emitted). Without this, a test that never exercises the auth path would pass vacuously and certify nothing.

3. **Negative control is a permanent test.** `TestCredentialLeak_SweepDetectsIntentionalLeak` is not a one-time experiment — it runs on every `go test` invocation and would fail if anyone broke the sweeper. This satisfies the counterparty's acceptance criterion verbatim: *"test fails if the synthetic token is intentionally leaked."*

4. **IndeterminateRecord sweep.** The record is structurally covered (Tess's Item 3): both the correctly-constructed version (no sentinel) and the deliberately bad version (sentinel in EndpointHost) are tested. If a future change to `IndeterminateRecord` accidentally adds a token field, the sweep will catch it.

5. **Error failure paths are the highest-risk surfaces.** Four of the ten tests specifically target error paths — these are the most likely leak vectors because naive implementations include HTTP response bodies or full URLs in errors. All four pass, proving the current implementation excludes sensitive content.

### Confirmed

**The sweep fires on intentional leaks.** `TestCredentialLeak_SweepDetectsIntentionalLeak` has 8 sub-cases, each injecting the sentinel into a different surface type (plain string, trace kind, trace payload value, trace JSON, ToolResult.Stdout, ToolResult JSON, IndeterminateRecord JSON, error string). All 8 fire.

---

## Learnings — Phase 1B Item 4: Synthetic Contract Proof (2026-08-17)

### Implementation Complete (commit 5c0c002)

**Files shipped:**
- `cmd/tools/ops-mock/main.go` — deterministic native mock binary for ops-synthetic contract
- `cmd/gert/synthetic_contract_integration_test.go` — CLI integration proof: --package-map + --profile together in runRun()
- `internal/tool/synthetic_contract_proof_test.go` — mcp-http binding + 3 invariant tests
- `internal/tool/auth_credential_leak_test.go` — extended with `TestCredentialLeak_SyntheticContract_MCPHTTPWithManagedIdentity`

### Key Decisions

1. **CLI test uses native binding; mcp-http tested at runtime layer.** The MCP-HTTP binding cannot be used through the full runRun() CLI path because ValidateTransportConfig requires HTTPS for the static YAML URL, and httptest.NewServer produces plain HTTP. The workaround (https://127.0.0.1/ as static URL + profile endpoint override to http://127.0.0.1:PORT) works at the runtime layer but not needed in CLI test. CLI test proves --package-map + --profile compose; internal/tool test proves mcp-http + managed-identity. Together both requirements are met without HTTPS complexity.

2. **imdsHTTPClient seam used for MI proof.** ManagedIdentityAuthProvider.httpClient (package-private) is accessible in package tool tests; set it to route IMDS calls to a mock httptest.Server. This keeps tests offline and deterministic.

3. **Both Action B outcomes exercised in both bindings.** Native: two pattern-search steps in the same runbook with FOUND: and non-FOUND: queries. MCP-HTTP: two separate transport.Invoke calls. Both pass.

4. **PLAN-013 invariant test in cmd/gert.** The endpoint-outside-allowed_hosts check is a planner behavior, so it belongs in the CLI integration test package (not in internal/tool which would create an import cycle).

5. **Absolute paths before chdirForTest.** A relative dir from makeWorkDir becomes invalid after Chdir. Fix: call filepath.Abs() to get absolute path before calling chdirForTest. This is the same pattern used in profile_tier0_integration_test.go.

### Parity Harness Interface

For Tess's fixture-agnostic harness in pkg/contractparity to compare the two bindings, it needs: (1) `writeOpsSyntheticNativePackage(t, root, opsMockBin)` to write the native package to a temp dir, (2) a function to construct an MCPHTTPTransport with a mock server for the mcp-http binding, (3) the action names "status-query" and "pattern-search" with the tool name "ops-synthetic", (4) test inputs: target="api-gateway" for status-query; "FOUND:pattern" and "no-match" for pattern-search. The parity harness should verify both bindings return the same JSON shape (not necessarily byte-identical values).

---

## Learnings — Phase 1B Item 4 Fix: Load-Bearing Flags (2026-08-17)

### The Core Problem

A "no-op flag" is structurally identical to a parsed-but-unreachable field — the flag appears in the test, the run succeeds, but removing the flag changes nothing. This is the same failure class the reachability gate was built to catch for YAML fields.

### Lesson: Load-bearing requires an observable difference, not just flag presence

The prior test had `--package-map` pointing to an identical package and `--profile` targeting a native binding that ignores auth. Both were no-ops. The fix required a design where removing either flag produces a different exit code:

- `--package-map` load-bearing: project config has `requires: []`; without the flag the tool is not found (exit 2). A flag pointing at a different-but-equivalent package proves only that parsing works, not that resolution works.
- `--profile` load-bearing: mcp-http static URL is a placeholder (`https://127.0.0.1/`); the profile overrides it to the live mock server. Without the flag, the TLS dial to port 443 fails (exit 1).

### Lesson: CLI tests can use SetIMDSEndpointForTest via a production-code export seam

`ManagedIdentityAuthProvider.httpClient` (package-private) is injectable only within `package tool`. But the IMDS endpoint can be redirected via an exported `SetIMDSEndpointForTest` function added to `auth_managed_identity.go` (not a test file). This allows `cmd/gert` integration tests to control IMDS without package-level access. The function uses a package-level string var; callers must not run in parallel.

### Lesson: HTTPS-at-scan-time vs. HTTP-at-runtime is the right split

`ValidateTransportConfig` checks HTTPS on the static YAML URL. Profile endpoint overrides are not subject to that check — they are validated only against `allowed_hosts` (host membership, not scheme). This means a plain HTTP `httptest.NewServer` URL can be used as a profile endpoint override in tests, with `https://127.0.0.1/` as the static placeholder.

### Empirical Results (commit d53a45f)

Before fix: removing either flag left `TestSyntheticContract_CLI_PackageMap_Profile_NativeBinding` green (no-ops confirmed).

After fix (`TestSyntheticContract_CLI_PackageMap_Profile_MCPHTTPBinding`):
- Without `--profile`: exit 1 (exitFailure) — TLS dial `https://127.0.0.1/` refused
- Without `--package-map`: exit 2 (exitValidation) — PKG-011 tool not found

