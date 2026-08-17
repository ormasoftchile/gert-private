# Gert Core → SQL Live-Site Operations: Phase 1B Implementation Plan — Rev 2

**Date:** 2026-08-17
**Supersedes:** Rev 1 (2026-08-17, same file, preserved in git history)
**From:** Barbara (Lead/Architect, Gert Core)
**To:** SQL Live-Site Operations

---

## 1. Corrections Accepted

All eight corrections are accepted. Don and David verified each against the codebase; the findings hold.

Correction 2 deserves specific acknowledgment: our Rev 1 proof targeted Gert's sample `tools/icm.tool.yaml` (`get_incident`/`update_incident`/`resolve_incident`, underscores), which is a different contract from your actual `icm.get-incident` (hyphen) with typed outputs. A green test against our sample would have proven nothing about your scenario — the same class of error as a test that cannot fail. We will not repeat it.

---

## 2. Per-Correction Disposition

| # | Correction | Accepted | What Changes | Evidence |
|---|-----------|----------|--------------|----------|
| 1 | Split managed-identity (IMDS) from workload-identity (federated) | Yes | Item 1 ships IMDS only; workload identity deferred to its agreed later phase. Estimate reduced to 1.5d. | `internal/tool/auth.go:29` — plain string switch, no ambient detection |
| 2 | ICM proof targets the wrong contract | Yes | Item 4 rebuilt against `icm.get-incident` / `tsg-recommendation.recommend` with their typed outputs. Gated on artifact delivery. Estimate increased to 3d. | `tools/icm.tool.yaml` — different action names, no typed outputs |
| 3 | Consumer classifications apply to their contract, not our sample | Yes | `icm.get-incident` → `read-only`; `tsg-recommendation.recommend` → `read-only`. Our sample tool classified separately. | Their explicit declaration |
| 4 | Auth/endpoint override safety — four of six conditions already satisfied | Yes | Item 2 gains PLAN-013 same-commit constraint. +0.5d. TokenGate / `allowed_hosts` already complete. | `internal/tool/auth_gate.go:23,82`; `validate_transport.go:60-74`; `runtime.go:71-74` |
| 5 | INDETERMINATE trace-safe evidence — seven fields absent | Yes | Item 3 gains `*IndeterminateRecord` struct with all seven fields. Estimate increased to 5.5-6d. | `pkg/engine/run.go:160-190` — existing `StepResult` lacks all seven |
| 6 | Profileless non-interactive must fail fast in Phase 1B | Yes | New Item 5 added. Reversal of our deferral accepted. | `cmd/gert/run.go:166` — `TTYOutput: true` hardcoded; zero `isatty()` in codebase |
| 7 | External production validation decoupled from mock proof | Yes | Live ICM validation is a separate integration gate they control. Mock proof is fully ours to deliver. | Removes external blocker on code-complete |
| 8 | `--package-map` proven execution-wired; `--profile` is not | Yes | Item 4 hard-serial after Item 2 (profile must be execution-wired first). | `cmd/gert/packagemap_integration_test.go:23` — proves package-map wiring |

---

## 3. What Is Already Satisfied

These conditions from correction 4 are implemented and shipped. File references for verification:

| Condition | Implementation | Citation |
|-----------|---------------|----------|
| TokenGate exists and validates host before token acquisition | `TokenGate.AttachToken` checks `req.URL.Hostname()` against `allowedHosts` before any token operation | `internal/tool/auth_gate.go:82` |
| TokenGate constructed with allowed hosts from tool definition | `NewTokenGate` receives `def.Auth.AllowedHosts` | `internal/tool/runtime.go:71-74` |
| Redirect bypass blocked (MCP-013) | TokenGate rejects post-redirect hosts not in allowlist | `internal/tool/auth_gate.go:82` (same check) |
| `allowed_hosts` validated at parse time (MCP-010/MCP-011) | `ValidateTransportConfig` rejects missing/invalid hosts | `internal/tool/validate_transport.go:60-74` |
| Token never acquired for non-allowed host (MCP-012) | Host check precedes token acquisition in AttachToken | `internal/tool/auth_gate.go:82` |

Chain is unbroken: `tools/icm.tool.yaml:31` → `def.Auth.AllowedHosts` → `NewTokenGate` → runtime enforcement.

---

## 4. Revised Work Items

### Item 1: Managed Identity Auth Provider (IMDS Only)

| Aspect | Detail |
|--------|--------|
| **Ships** | `internal/tool/auth_managed_identity.go` — IMDS endpoint (`169.254.169.254/metadata/identity/oauth2/token`), 5-minute proactive refresh mirroring AzureCLI pattern. One new case in `NewAuthProvider`. One entry in `knownAuthProviders`. |
| **Not shipped** | Workload identity (federated token exchange). Deferred to its agreed later phase. |
| **Dependencies** | None (stdlib `net/http` only). |
| **Estimate** | 1.5 days |
| **Acceptance** | Unit tests: IMDS happy path (httptest mock), IMDS unavailable → MCP-002, token refresh before expiry, context cancellation propagation. |

### Item 2: Executable Runtime Binding + PLAN-013 Endpoint Safety

| Aspect | Detail |
|--------|--------|
| **Ships** | (a) Profile field threaded through `adapter.WireOptions` → `BuildEngineConfig` → `DefaultToolRuntime` → `runtime.go`. `ProfileToolOverride.Endpoint` applied as URL override. Top-level profile `auth.provider` selects credential mechanism. (b) **PLAN-013**: `checkToolEnvironmentPreflight` (`internal/planner/preflight.go:89`) gains endpoint-vs-allowed_hosts validation. |
| **Critical constraint** | PLAN-013 check and endpoint-override wiring MUST ship in the same commit. No window where overrides execute unvalidated. |
| **Seam** | `checkToolEnvironmentPreflight` already receives `def *schema.ToolDef` and `profile *schema.RuntimeProfile`. No new parameter threading. |
| **Dependencies** | None (concurrent with Items 1, 3, 5). |
| **Estimate** | 3.5-4.5 days (includes +0.5d for PLAN-013) |
| **Acceptance** | Tests: endpoint override routes correctly; PLAN-013 rejects endpoint not covered by `allowed_hosts`; profile auth selects managed-identity; tool-definition mode never rewritten. |

### Item 3: INDETERMINATE + Trace-Safe Evidence Record

| Aspect | Detail |
|--------|--------|
| **Ships** | (a) New `StepStatusIndeterminate` value. (b) `*IndeterminateRecord` struct embedded on `StepResult` (see section 5 design rule). (c) Classification-aware timeout branch: non-read-only actions that timeout → INDETERMINATE + halt; read-only timeouts → failed (retry eligible). (d) `gert resume` guard requiring `--acknowledge-indeterminate`. |
| **IndeterminateRecord fields** | `RunID`, `StepID`, `ToolName`, `ActionName`, `Classification`, `EndpointHost`, `AttemptNumber`, `Deadline`, `FailureTime`, `TransportErrCategory` |
| **"Never credentials" rule** | `EndpointHost` = `url.Parse(def.URL).Host` — a hostname only. Never token or Authorization value. Existing redaction seams (output-layer scrubbing, runbook `redact:` block) are not applicable to transport metadata and not reused here. |
| **Dependencies** | None (concurrent with Items 1, 2, 5). |
| **Estimate** | 5.5-6 days |
| **Acceptance** | Vectors: {read-only, destructive, unspecified} x {timeout, network-error} with expected status, halt behavior, and all IndeterminateRecord fields populated. |

### Item 4: ICM Contract Proof

| Aspect | Detail |
|--------|--------|
| **Ships** | (a) Runbook targeting `icm.get-incident` (hyphen, their contract) with typed outputs `title`, `service`, `environment`, `logical_server`, `database`. (b) Second runbook targeting `tsg-recommendation.recommend` with outcomes `suggested` / `no-suggestion`. (c) Mock MCP server implementing both contracts. (d) Cross-binding parity harness: same runbook under real and mock bindings, deep-compare structured outputs. |
| **Contract source** | Their actual tool definitions and schema — NOT reconstructed from prose. |
| **`--package-map` usage** | Already execution-wired (`TestRun_PackageMap_RealVsMockBinding`, `cmd/gert/packagemap_integration_test.go:23`). Same tool name, `--package-map` selects binding without touching the runbook. |
| **Dependencies** | **Serial after Item 2** (profile must be execution-wired for `--profile` to reach transport). **Gated on artifact delivery** (section 8). |
| **Estimate** | 3 days (starts only after Item 2 ships AND artifacts arrive) |
| **Acceptance** | Mock proof passes with correct typed outputs; parity harness validates output structure matches between bindings. |

### Item 5: Profileless Non-Interactive Fail-Fast

| Aspect | Detail |
|--------|--------|
| **Ships** | CLI entry `cmd/gert/run.go`, immediately after the profile-check block (~line 110), before `BuildEngineConfig` (~line 162). Detection: `fi, err := os.Stdin.Stat(); nonInteractive := err != nil || (fi.Mode()&os.ModeCharDevice == 0)`. If non-interactive AND no `--profile` supplied → hard error with actionable message, before any side effects. |
| **No new dependencies** | `golang.org/x/term` and `mattn/go-isatty` both absent from go.mod; stdlib detection sufficient. Works on Windows via Go runtime. |
| **Dependencies** | None (concurrent with Items 1, 2, 3). |
| **Estimate** | 1.5 days |
| **Acceptance** | Test: `gert run` with no profile, piped stdin → exit with error message. Test: `gert run --profile unattended.yaml` with piped stdin → proceeds normally. |

### Sequencing

```
         Day 1-1.5      Day 1-4.5       Day 1-6        Day 1-1.5
        +----------+   +-----------+   +----------+   +----------+
        | Item 1   |   |  Item 2   |   |  Item 3  |   |  Item 5  |
        |IMDS only |   |  Binding  |   |INDETERM. |   |Fail-fast |
        +----------+   +-----+-----+   +----------+   +----------+
                             |
                             v (after Item 2 ships + artifacts arrive)
                       +-----------+
                       |  Item 4   |
                       | ICM proof |  3 days
                       +-----------+
```

Items 1, 2, 3, 5 parallelize. Item 6 starts after Item 1, fits within the parallel window. Item 4 is serial after Item 2, gated on artifacts.

---

## 5. Ratified Design Rules

### Rule A: Profile Auth Schema Invariant

When Item 2 extends the `RuntimeProfile` schema, the profile:

- **MAY** add a `Provider` field (credential mechanism selection — e.g., `managed-identity` vs. `azure-cli`).
- **MUST NOT** add `Scope` or `AllowedHosts`.

The gate is always constructed from `def.Auth.Scope` and `def.Auth.AllowedHosts` — the tool definition's declared contract. The profile substitutes *who acquires* the token, never *what it is scoped for* or *where it may go*.

`RuntimeProfile` has no top-level `Auth` field today. `ProfileToolOverride` has only `Endpoint` and `Mode`. Scope-clobbering is currently structurally impossible. This rule ensures it remains so.

This directly implements their requirement: top-level auth "must not silently replace a tool-specific audience, scope, or host policy."

### Rule B: `*IndeterminateRecord` as Non-Fabricated-Output Representation

`StepResult.Output == nil` is ambiguous (a tool returning empty outputs also yields nil). The representation for "completion unknown" is:

- Embed `*IndeterminateRecord` on `StepResult`.
- Non-nil pointer = completion unknown; `Output` remains nil (not fabricated).
- Fields: `RunID`, `StepID`, `ToolName`, `ActionName`, `Classification`, `EndpointHost`, `AttemptNumber`, `Deadline`, `FailureTime`, `TransportErrCategory`.

This satisfies their requirement that trace-safe evidence "must not require fabricated output."

---

## 6. Conformance Harness Disclosure

Item 5 (fail-fast) affects the conformance harness. `internal/conformance/enum_harness.go:runCLI()` invokes `gert run` with no `--profile` and no TTY. After Item 5 ships, every conformance/enum/dyninclude vector would hit the fail-fast gate.

**Fix:** Add `internal/conformance/testdata/unattended-test.profile.yaml` declaring `attendance: unattended`. Update `runCLI()` to pass `--profile` pointing to it.

**Behavior is identical:** the harness previously got `NoOpApprovalGate` via hardcoded `TTYOutput: false` at adapter level. It will now get the same gate via declared attendance. The path through `buildApprovalGate()` (`internal/adapter/wire.go:306-313`) resolves identically.

**No vector results change.** The e2e tests (`internal/e2e/helpers_test.go:171`) use `TTYOutput: false` at the adapter level directly; they are unaffected.

---

## 7. Revised Estimate

| Item | Days | Parallelizable? | Serial dependency |
|------|------|-----------------|-------------------|
| 1: Managed Identity (IMDS) | 1.5 | Yes | — |
| 2: Runtime Binding + PLAN-013 | 3.5-4.5 | Yes | — |
| 3: INDETERMINATE + evidence | 5.5-6 | Yes | — |
| 4: ICM contract proof | 3 | No | After Item 2 + artifacts |
| 5: Profileless fail-fast | 1.5 | Yes | — |
| 6: Credential-leak assertions | 1 | Yes | After Item 1 |
| **Critical path** | **~9-10 days** | | Items 1,2,3,5 parallel (longest: Item 3 at 6d); Item 6 after Item 1 (fits within Item 3's window); Item 4 at 3d after Item 2 + artifacts |

Serial chain: Item 2 must ship before Item 4 can start. Item 4's 3 days begin only after artifacts arrive (section 8). If artifacts arrive before Item 2 completes, the critical path is Item 3's 6 days + Item 4's 3 days = 9 days. If artifacts arrive late, Item 4 slides accordingly. Item 6 (1 day) starts after Item 1 (1.5 days) and fits within the parallel window of Item 3.

Production ICM validation: separate integration gate controlled by them (correction 7). Not included in this estimate.

---

## 8. BLOCKING ARTIFACT REQUEST

**Item 4 cannot start without the following eight artifacts.** This is a hard precondition.

| # | Artifact | Purpose |
|---|----------|---------|
| 1 | `icm.get-incident` tool definition (YAML or equivalent) with typed output schema (`title`, `service`, `environment`, `logical_server`, `database`) | Contract we are proving against |
| 2 | `tsg-recommendation.recommend` tool definition with outcome schema (`suggested`, `no-suggestion`) | Second contract in the proof |
| 3 | A runbook (or runbook fragment) that exercises both tools in their intended sequence | Ensures our proof runs the same workflow they run |
| 4 | The `--package-map` binding entry for their mock server, including the explicit schema the mock must implement | Correct mock construction. We will not infer schema from prose. |
| 5 | Expected structured output examples for both tools (at least one success case each) | Parity assertion targets |
| 6 | Their runtime profile for headless/CI execution (or the relevant fields: provider, endpoint, attendance) | End-to-end binding validation |
| 7 | Transport mode for each tool in their production binding (`mcp-stdio` vs `mcp-http`) | Determines whether the HTTP proof needs a separate contract-identical `mcp-http` binding selected through package-map, with the profile not rewriting transport mode |
| 8 | Whether their runbook uses `toolRefs:` with package names or bare file-path references | Determines whether package-map substitution applies to their runbook as written |

**Why we will not reconstruct from prose:** Guessing a contract schema from description is exactly how a proof validates the wrong thing — this is the error correction 2 caught. We will build the proof against their actual definitions or not at all. This includes mock schema (artifact 4) — if they cannot supply the package-map entry, we need their explicit schema delivered, not permission to infer it.

**Delivery options:** Repo access to `gert-sqllivesite` (not currently available on this machine — confirmed absent from `C:\One\OpenSource\`), or the eight files sent directly. Either works; please advise which you prefer.

---

## 9. Their Phase 1B Acceptance Criteria — Coverage Map

Their verbatim acceptance criteria, mapped to the item that satisfies each:

> Phase 1B is complete when:

| # | Criterion (verbatim) | Covered by | Notes |
|---|---------------------|-----------|-------|
| 1 | managed identity and workload identity are not ambiguously combined | Item 1 | Correction 1 accepted: IMDS only, workload identity deferred. No ambient detection, no fallback between them. |
| 2 | profile/provider/endpoint compatibility fails closed | Item 2 (PLAN-013) | Endpoint-vs-allowed_hosts validated at preflight; unknown provider → parse error; same-commit constraint. |
| 3 | endpoint overrides respect `allowed_hosts` | Item 2 (PLAN-013) + section 3 (already satisfied) | TokenGate enforcement already shipped (`auth_gate.go:82`). PLAN-013 adds preflight check. |
| 4 | the exact SQL Live-Site contract passes native-mock and managed-identity HTTP bindings | Item 4 | Gated on artifacts (section 8). Built against their definitions, not our sample. |
| 5 | package-map/profile composition is demonstrated | Item 4 | The proof exercises both flags *together*: `--package-map` selects the mock binding, `--profile` selects managed-identity + endpoint. Composition, not independent use. |
| 6 | INDETERMINATE state and acknowledgment are tested | Item 3 | Vectors cover timeout/network-error × classification; `--acknowledge-indeterminate` gate tested. |
| 7 | profileless non-interactive execution fails immediately | Item 5 | Stdlib TTY detection; hard error before side effects. |
| 8 | no credential appears in runbook state, results, traces, or errors | **Item 6 (NEW)** | **Not previously covered.** See Item 6 below. |

**Gap acknowledged:** Criterion 8 was not covered in Rev 2 draft. Item 6 is added to address it.

### Item 6: Credential-Leak Assertions

| Aspect | Detail |
|--------|--------|
| **Ships** | Negative test suite asserting no credential material (tokens, Authorization header values, IMDS response bodies) appears in: (a) serialized `RunState`; (b) `StepResult` fields including `Output` and `Error`; (c) `IndeterminateRecord` fields (hostname only, never token); (d) trace/log output; (e) error strings from the managed-identity provider on auth failure. |
| **Why this is new work** | Item 1 introduces a new credential path (IMDS token acquisition). Pre-existing redaction seams (output-layer token scrubbing; runbook `redact:` block at `schemas/runbook.schema.json:262`) are not applicable to transport metadata. The "hostname only, never token" rule on `IndeterminateRecord` is currently enforced by code review and field comments, not by a test. |
| **Approach** | Run a managed-identity integration test (httptest IMDS mock returning a known synthetic token). After execution, scan all serialized state, results, and captured log output for the synthetic token string. Any match → test failure. |
| **Dependencies** | After Item 1 (needs the managed-identity provider to exercise). Can parallelize with Items 2, 3, 5. |
| **Estimate** | 1 day |
| **Acceptance** | Tests pass (token absent from all outputs). Test fails if the synthetic token is intentionally leaked (proves the assertion is not vacuous). |

---

## 10. Explicitly NOT in Phase 1B

| Item | Status | Phase |
|------|--------|-------|
| Workload Identity (federated token exchange) | Deferred per correction 1 | Agreed later phase |
| Per-tool `auth:` override in profiles | Loader rejects | Phase 3 |
| Reconnect/retry hardening | Schema-specified, not executed | Phase 2+ |
| VS Code bridge / host integration | Out of scope | Phase 2 |
| `RequiresCapabilities` enforcement | Specified-not-enforced | Phase 3 |
| `AllowedModes` enforcement | Specified-not-enforced | Phase 2 |
| Full reconnect-on-session-expiry for mcp-http | MCP-006 sentinel exists | Phase 2 |
| Live ICM production validation | Their integration gate | Their timeline |

---

*End of plan. Artifact delivery method and timeline requested.*
