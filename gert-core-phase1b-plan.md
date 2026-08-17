# Gert Core → SQL Live-Site Operations: Phase 1B Implementation Plan

**Date:** 2026-08-17  
**From:** Barbara (Lead/Architect, Gert Core)  
**To:** SQL Live-Site Operations  

---

## 1. Scope Correction Acknowledged

You are correct: the round-2 phase agreement explicitly listed managed identity, conservative timeout semantics, executable runtime binding, and one read-only ICM proof as Phase 1 deliverables, and our status report labeled the governance/profile foundation as the complete phase when it was not. We accept the Phase 1A/1B split without qualification.

---

## 2. Verification Evidence — All Four Claims Confirmed

| # | Claim | Verified Finding | Citation |
|---|-------|-----------------|----------|
| 1 | Managed identity absent | `NewAuthProvider` (`internal/tool/auth.go:29`) has one arm: `"azure-cli"`; default → MCP-002. `knownAuthProviders` (`internal/tool/validate_transport.go:16`) = `{"azure-cli": true}` — parse-time rejection. Existing test at `internal/tool/auth_azurecli_test.go:360` asserts MCP-002 for `"managed-identity"`. go.mod has zero Azure SDK deps; IMDS and Workload Identity are stdlib `net/http`. | Confirmed by Don |
| 2 | ICM proof absent | No `icm-tsg-router` exists anywhere. `tools/icm.tool.yaml` exists with `transport.mode: mcp-http`, `auth.provider: azure-cli`, actions `get_incident`/`update_incident`/`resolve_incident` — but **no action declares a `classification`** (see §5). | Confirmed by Don |
| 3 | INDETERMINATE / halt-on-timeout absent | Zero occurrences of `INDETERMINATE` in codebase. Step status enum (`pkg/engine/run.go:194–203`): 7 values, no indeterminate. `Contract.Idempotent` (`pkg/schema/step.go:93`) is never read by retry or timeout paths. On timeout, `engine.go:660–664` calls `failRun()` unconditionally — destructive and read-only actions treated identically. | Confirmed by David |
| 4 | Profile is plan-time only (runtime binding absent) | Parse chain: `cmd/gert/run.go:101–108` → `:137` planner config → `:167` attendance → `:456` plan metadata → `internal/engine/engine.go:127/250` ProfileEvaluator. Engine ONLY reads profile there. `adapter.WireOptions` has no Profile field. Transport at `internal/tool/runtime.go:70–77` constructs from tool definition only. `ProfileToolOverride.Endpoint` (`pkg/schema/profile.go:108`) is parsed and unit-tested but **never read in execution path**. | Confirmed by David |

---

## 3. Phase 1B Work Plan

### Item 1: Managed Identity Auth Provider

| Aspect | Detail |
|--------|--------|
| **Ships** | `internal/tool/auth_managed_identity.go` — IMDS + Workload Identity (`AZURE_FEDERATED_TOKEN_FILE` token exchange), 5-minute proactive refresh mirroring AzureCLI pattern. One new case in `NewAuthProvider`. One entry in `knownAuthProviders`. |
| **Dependencies** | None (stdlib-only). |
| **Estimate** | 2 days |
| **Acceptance** | Unit tests: IMDS happy path (httptest mock), WI happy path, token refresh, timeout propagation via context. Integration test with real credential deferred to §6. |

### Item 2: Executable Runtime Binding (Profile → Transport)

| Aspect | Detail |
|--------|--------|
| **Ships** | Profile field added to `adapter.WireOptions`. `BuildEngineConfig` → `DefaultToolRuntime` threads profile. `ProfileToolOverride.Endpoint` applied as URL override at `runtime.go`. Top-level profile auth applied to `NewAuthProvider` (see §4 ruling). |
| **Dependencies** | None (concurrent with Item 1). |
| **Estimate** | 3–4 days |
| **Acceptance** | Tests: profile endpoint override routes to correct URL; profile top-level auth selects managed-identity provider; tool-definition mode is never rewritten by profile. |

### Item 3: Conservative Timeout / INDETERMINATE Semantics

| Aspect | Detail |
|--------|--------|
| **Ships** | New `StepStatusIndeterminate` value. Classification-aware timeout branch: destructive/unspecified actions that timeout → INDETERMINATE + run halted; read-only timeouts → failed (retry eligible). `gert resume` guard: halted-INDETERMINATE runs require explicit `--acknowledge-indeterminate` to continue. |
| **Dependencies** | None (concurrent with Items 1+2). |
| **Estimate** | 4–5 days |
| **Acceptance** | 8+ vectors: {read-only, destructive, unspecified} × {timeout, network-error} with expected status and halt behavior. |

### Item 4: Headless ICM Read-Only Proof

| Aspect | Detail |
|--------|--------|
| **Ships** | Runbook invoking `icm.get_incident` only. `httptest` mock MCP server validating `Authorization: Bearer <token>` header format and correct scope. Integration test exercising managed-identity → ICM transport chain end-to-end (against mock). |
| **Dependencies** | **Blocked on Item 1** (needs managed identity). **Blocked on §6** (production validation requires their credential provisioning). |
| **Estimate** | 1 day (mock proof); production validation date controlled by them. |

### Sequencing

```
         Day 1–2        Day 2–6         Day 2–7        Day 3 (after Item 1)
        ┌───────┐      ┌─────────┐     ┌──────────┐   ┌───────┐
        │Item 1 │      │ Item 2  │     │  Item 3  │   │Item 4 │
        │Mgd Id │      │ Binding │     │INDETERM. │   │ICM prf│
        └───────┘      └─────────┘     └──────────┘   └───────┘
```

Items 1, 2, 3 can parallelize. Item 4 is serial after Item 1.

---

## 4. Design Decision: Auth Precedence Ruling

### Ruling

**Phase 1B:** Profile **top-level** `auth` configuration reaches transport construction and WINS over the tool definition's `auth.provider` when both are specified.

**Phase 3 (unchanged):** Per-tool `auth:` override in profile `tools:` blocks remains rejected by the loader. The loader MUST error on `profile.tools[name].auth` until Phase 3.

### Precedence Rule

When profile top-level auth and tool-definition auth disagree:

> **Profile wins.** The profile is the execution context binding; the tool definition is the portable interface contract.

### Rationale

This is what managed identity in CI actually requires: the tool definition says `auth.provider: azure-cli` (the default/interactive binding), the CI profile says `auth.provider: managed-identity` (the headless binding), and the profile must win or the run cannot execute headlessly.

### Why This Is Safe / Not a Mode Rewrite

The Phase 1A ruling that "a profile must not silently rewrite a tool definition's transport MODE" remains intact. The distinction:

- **Transport mode** (`subprocess`, `mcp-stdio`, `mcp-http`) determines the protocol semantics, serialization format, and failure model. Rewriting it silently would change what the tool *is*.
- **Auth provider** (`azure-cli`, `managed-identity`) is a credential-binding parameter within a fixed mode. It determines *who presents the token*, not *how the tool is invoked*. The transport mode, URL, and scope remain unchanged.

A profile selecting `managed-identity` over `azure-cli` is analogous to a deployment environment selecting a service principal over a developer identity — the interface contract is unchanged; only the authentication substrate varies.

### Enforcement

- The loader validates: profile top-level `auth.provider` must name a value in `knownAuthProviders`. Unknown values → parse error.
- Runtime logs when profile auth overrides tool-definition auth (INFO level, not hidden).
- Per-tool `auth:` in profile `tools:` blocks → loader error with message citing Phase 3 deferral.

---

## 5. ICM Classification Finding

**Current state:** All three actions in `tools/icm.tool.yaml` (`get_incident`, `update_incident`, `resolve_incident`) declare no `classification` field. Under the governance rules shipped in Phase 1A, unclassified actions default to `unspecified`.

**Behavioral consequence today (by execution context):**

| Action | Effective Classification | Interactive Operator | CI/Headless | Test Context |
|--------|------------------------|---------------------|-------------|--------------|
| `get_incident` | unspecified | active confirmation required | deny unless explicitly permitted by policy | auto-approve only for deterministic mock/native bindings |
| `update_incident` | unspecified | active confirmation required | deny unless explicitly permitted by policy | auto-approve only for deterministic mock/native bindings |
| `resolve_incident` | unspecified | active confirmation required | deny unless explicitly permitted by policy | auto-approve only for deterministic mock/native bindings |

This is the conservative default working as designed — fail-closed, context-dependent approval, INDETERMINATE on ambiguous result (after Item 3 ships).

**Recommended declarations (for your confirmation):**

We recommend the following, but these are **your** operational decisions for **your** incident tooling — we are offering a starting point, not a prescription.

```yaml
# get_incident — read-only, idempotent, safe to retry
classification: read-only

# update_incident — mutating, state change
classification: mutating

# resolve_incident — mutating, state change
classification: mutating
```

**Why `mutating` rather than `destructive`:** Under the agreed matrix, `mutating` and `destructive` receive **identical** lost/late-result handling (INDETERMINATE + halt + manual verification required). The distinction is about operator signalling and approval policy, not retry safety. We recommend `mutating` because updating or resolving an incident changes state but does not destroy data — the incident remains queryable and the action is reversible (you can reactivate a resolved incident). `destructive` would be appropriate for irreversible operations (e.g., permanent data deletion). However, if your operational model treats incident resolution as effectively irreversible in practice, `destructive` is equally valid from a safety perspective — the runtime behavior is the same.

**Action requested:** Please confirm the intended classification for each action. We will update the upstream tool definition to match once confirmed.

---

## 6. External Dependencies Required From SQL Live-Site Operations

These are on the critical path for production validation of the ICM proof. We cannot promise a production-validation date because we do not control these:

| Dependency | Purpose | Needed By |
|-----------|---------|-----------|
| Managed-identity credential bound to a test runner (VM MSI or Workload Identity federation in CI) | Authenticate to ICM MCP API without interactive login | Before production ICM proof can execute |
| `api://icmmcpapi-prod/mcp.tools` scope grant to that identity | Authorization to call ICM MCP endpoints | Same |
| Network access from test runner to `icm-mcp-prod.azure-api.net` | Reachability | Same |
| Confirmation of intended `classification` values for ICM actions | Correct governance behavior | Before we update upstream tool definition |

We will ship the mock-validated proof (Item 4) on our timeline. Production validation is gated on these four items from your side. Please provide an expected timeline or contact for provisioning.

---

## 7. Explicitly NOT in Phase 1B

| Item | Status | Phase |
|------|--------|-------|
| Per-tool `auth:` override in profiles | Loader rejects; specified-not-enforced | Phase 3 |
| Reconnect/retry hardening (beyond classification-aware timeout) | Specified in schema (`Contract.Idempotent`) but not executed | Phase 2+ |
| Workload Identity beyond basic `AZURE_FEDERATED_TOKEN_FILE` exchange | If exotic federation scenarios surface, deferred | Phase 2 |
| VS Code bridge / host integration | Out of scope | Phase 2 |
| `RequiresCapabilities` enforcement | Specified-not-enforced | Phase 3 |
| `AllowedModes` enforcement | Specified-not-enforced. Phase 1A report characterized this as "near-term; the check is trivial once RunMode is accessible at plan time." We are re-scoping to Phase 2 (not Phase 3) because the prerequisite — RunMode threading through plan context — is part of the Phase 2 host bridge work that introduces distinct run modes. The enforcement itself remains trivial once that lands. | Phase 2 |
| Full reconnect-on-session-expiry for mcp-http | MCP-006 sentinel exists; automatic reconnect deferred | Phase 2 |

---

## 8. Estimate

| Work | Days (sequential) | Parallelizable? |
|------|-------------------|-----------------|
| Item 1: Managed Identity | 2 | Yes (independent) |
| Item 2: Runtime Binding | 3–4 | Yes (independent) |
| Item 3: INDETERMINATE semantics | 4–5 | Yes (independent) |
| Item 4: ICM Mock Proof | 1 | No (after Item 1) |
| **Total sequential** | **10–12** | |
| **Total parallelized (Items 1+2+3 concurrent, then Item 4)** | **~6–7 days engineering time** | |

Production ICM validation: date controlled by dependency provisioning from your side (§6). Not included in our estimate.

---

## 9. Profileless Stdin-Block Hazard

**The hazard is real and unfixed.** Without `--profile`, `TerminalApprovalGate` is installed even in non-interactive contexts. If any tool declares `requires-approval: true`, the process blocks on stdin indefinitely and exits 3. In a CI/headless live-site scenario, this is an availability failure during incident response — the runbook hangs instead of executing.

**Your operational rule** — every CI invocation must supply an unattended profile — is a **correct and effective mitigation**. With an unattended profile supplied, the engine installs the appropriate non-interactive gate and the hazard does not manifest.

**Underlying cause:** `TTYOutput: true` is hardcoded rather than derived from profile attendance declaration or actual terminal detection.

**Phase 1B disposition:** We are deferring the fix to Phase 2. This is a **scheduling decision, not a severity judgment**. The fix requires output-layer refactoring (terminal detection, gate selection, and output formatting are entangled) that belongs with the Phase 2 host bridge work. The defect remains real; the mitigation remains effective; the fix remains planned.

We will document the "always supply a profile in CI" requirement in the profile specification and add a startup warning when no profile is supplied and stdin is not a TTY.

---

*End of plan. Questions or provisioning timelines welcome.*
