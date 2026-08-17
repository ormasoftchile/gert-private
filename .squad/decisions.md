# Phase 1B Rev2 Decisions

**Dated:** 2026-08-17  
**Status:** Merged from inbox  
**Sources:** don (auth corrections), david (integration corrections), barbara (design rulings)

---

# Don — Phase 1B Rev2 Auth Corrections

**Date:** 2026-08-17  
**Author:** Don (Backend/Core)  
**Requested by:** Cristiano (ormasoftchile)  
**Status:** Findings complete — awaiting Barbara / team ratification

---

## Correction A — Managed-Identity vs Workload-Identity Split

### Verdict: Correct. Structurally trivial.

SQL Live-Site Operations is right to require separate registered names. Ambient auto-detection (check env var, fall back to IMDS if absent) would mean the same profile behaves differently on different hosts, which violates deterministic binding. One name = one mechanism.

### Feasibility

`NewAuthProvider` (`internal/tool/auth.go:29`) is a switch on a string. Adding `"managed-identity"` as its own case is one line of code plus a new implementation file. Adding `"workload-identity"` later follows identically. `knownAuthProviders` (`internal/tool/validate_transport.go:16`) is `map[string]bool`; one new entry per provider.

**Fail-closed confirmed:** Unknown provider names return `MCP-002` from the `default:` branch of `NewAuthProvider` and are rejected at parse time by `knownAuthProviders`. There is no ambient detection, no silent fallback, no path through which an unrecognized provider name produces a working credential. Fail-closed holds.

### Revised Estimate

| Item | Prior estimate (combined) | Revised (IMDS only) |
|------|--------------------------|---------------------|
| `managed-identity` | 2 days (IMDS + workload identity) | **1.5 days** (IMDS only) |
| `workload-identity` | included above | **deferred** (own phase, own estimate) |

IMDS path is simpler than AzureCLI (no subprocess, single HTTP call to `169.254.169.254`, token caching, 5-minute refresh). Workload identity federated token exchange was the larger half of the original estimate.

---

## Correction B — Auth and Endpoint Override Safety

### B-1: Does TokenGate exist?

**Yes. Fully implemented. File: `internal/tool/auth_gate.go`.**

- `TokenGate` struct at `auth_gate.go:23`.
- Constructed at `runtime.go:71–74` inside the `TransportMCPHTTP` case.
- `NewTokenGate(provider, def.Auth.Scope, def.Auth.AllowedHosts)` — provider and allowed_hosts come from the tool definition's auth block.
- `AttachToken` at `auth_gate.go:82` checks `req.URL.Hostname()` against `allowedHosts` **before** acquiring any token. Host not in list → returns `MCP-012`, request not sent.
- Redirect prevention: `NewMCPHTTPTransport` refuses HTTP redirects when a gate is present (`MCP-013`), preventing redirect-based host bypass.
- Host matching: exact, case-insensitive, port-stripped. Wildcards explicitly unsupported (a `*.azure-api.net` entry fails closed, not open).

**Condition 6 of Correction B (TokenGate repeats host validation before attaching token) is already satisfied.** No new work needed here, provided the gate is constructed with the correct `allowedHosts` when a profile overrides the endpoint.

---

### B-2: Is `allowed_hosts` enforced at runtime?

**Yes. Doubly enforced. Not a parsed-but-unread field.**

This was the most important question. The answer is unambiguous:

**Enforcement path 1 — parse time:**
`ValidateTransportConfig` (`validate_transport.go:60–74`):
- MCP-010: auth present but `allowed_hosts` empty or missing → parse-time error.
- MCP-011: URL hostname not in `allowed_hosts` → parse-time error.
Called from `ParseToolFile` (tool file scan), so invalid tool files fail before any invocation.

**Enforcement path 2 — runtime, per request:**
`TokenGate.AttachToken` (`auth_gate.go:82–91`):
- Every HTTP request goes through `AttachToken`.
- `req.URL.Hostname()` checked against `g.allowedHosts` before token acquisition.
- Not in list → MCP-012 returned, request blocked, token never acquired or sent.

`tools/icm.tool.yaml:31` declares `allowed_hosts: [icm-mcp-prod.azure-api.net]`. That value flows to `def.Auth.AllowedHosts` in the registry, then to `NewTokenGate` at `runtime.go:74`. The chain is unbroken. **`allowed_hosts` is enforced, not decorative.**

---

### B-3: Tier 0 seam for endpoint-vs-allowed_hosts preflight check

**Where:** `checkToolEnvironmentPreflight` in `internal/planner/preflight.go:89`.

**Why this is the right seam:**
- Already called from `resolveTool()` (`planner.go:568`) for every tool referenced in a plan.
- Already receives `def *schema.ToolDef` (carries `def.Auth.AllowedHosts`) and `profile *schema.RuntimeProfile` (carries `profile.Tools[name].Endpoint`).
- Pattern is established: check conditions, return `*plannerPkg.PlanError` with a PLAN-0xx code.

**What the check would do:**
```
if profile.Tools[def.Name] != nil && profile.Tools[def.Name].Endpoint != "" {
    overrideHost = parse hostname from profile.Tools[def.Name].Endpoint
    if overrideHost not in def.Auth.AllowedHosts → return PLAN-013 / ErrEndpointNotInAllowedHosts
}
```

**Data availability:** The planner has `p.profile` (set at `planner.go:65`). The tool def has `Auth.AllowedHosts`. Both are available at the point `checkToolEnvironmentPreflight` is called. No new planner fields required. No new threading required.

**Note:** This check is currently a no-op to add because `ProfileToolOverride.Endpoint` is never applied at runtime today (Item 2 — runtime binding — hasn't shipped). The check should be added at the same time Item 2 wires the endpoint override, so the constraint and the feature land together.

**Requirement satisfied:** "An endpoint override outside `allowed_hosts` must fail during planning — not midway through execution." ✓ (once Item 2 + PLAN-013 land together)

---

### B-4: Can profile top-level auth clobber tool scope/audience?

**Currently: structurally impossible — `RuntimeProfile` has no `auth` field.**

`pkg/schema/profile.go:130–137` shows the `RuntimeProfile` struct fields: `APIVersion`, `ID`, `Context`, `Attendance`, `Approval`, `Transport`, `Tools`. No top-level `Auth` field. No `Provider`, `Scope`, or `AllowedHosts` on `ProfileToolOverride` either.

The comment at `profile.go:95` says `ProfileTransport` "may carry future auth parameters" but that is a forward comment, not an implemented field.

**Risk is not current. Risk is in the design of Item 2.** When profile-level auth override is added to the schema to enable `managed-identity` selection from CI profiles, the design MUST:
- Allow `provider` to be set at profile level (mechanism selection — the legal operation).
- NOT allow `scope` or `audience` to be set at profile level, unless the tool definition explicitly declares them overridable.
- NOT allow `allowed_hosts` to be narrowed or widened by the profile.

**Structural prevention that must be built into the schema change for Item 2:**
`ProfileToolOverride` may gain a `Provider string` field. It must NOT gain `Scope`, `Audience`, or `AllowedHosts`. The token gate must always be constructed with `def.Auth.Scope` and `def.Auth.AllowedHosts` from the tool definition — the profile may only substitute the credential mechanism (the thing that acquires the token), not the audience the token is scoped for or the hosts it may be sent to.

---

### B-5: Estimate for the six conditions

| Condition | Status | Work |
|-----------|--------|------|
| 1. Transport is mcp-http | Already enforced by `ValidateTransportConfig` | 0 |
| 2. Provider explicitly named | Already enforced by `knownAuthProviders` + MCP-002 | 0 |
| 3. Scope explicitly defined | Already enforced (scope required in auth block) | 0 |
| 4. Endpoint host in `allowed_hosts` | **Not yet checked at plan time for profile overrides** | 0.5 days (PLAN-013 check in `checkToolEnvironmentPreflight`) |
| 5. Endpoint compatibility checked at Tier 0 preflight | Same as condition 4 — same implementation | included above |
| 6. TokenGate repeats host validation before token attachment | **Already satisfied** by `AttachToken` at `auth_gate.go:82` | 0 |

**Net additional estimate: +0.5 days**, folded into Item 2 (runtime binding). The PLAN-013 check must land in the same commit as the profile endpoint override wiring — they are a single atomic change.

---

## Summary

| Topic | Status |
|-------|--------|
| Correction A feasibility | Trivial. One switch case, one map entry. |
| Correction A revised estimate | 1.5 days (IMDS only) |
| TokenGate exists | Yes — `internal/tool/auth_gate.go:23`, fully implemented |
| TokenGate host validation | Yes — before token acquisition, on every request |
| `allowed_hosts` enforced | **Yes** — parse time (MCP-010/011) + runtime (MCP-012). Not decorative. |
| Tier 0 seam for endpoint check | `checkToolEnvironmentPreflight` in `preflight.go:89` — data available, pattern established |
| Profile can clobber tool scope | Not currently possible (no auth field on profile). Must be prevented structurally in Item 2 schema design. |
| Net added work | +0.5 days folded into Item 2 |


---

# David — Phase 1B Rev 2 Decisions
**Date:** 2026-08-17  
**Author:** David (Integration Engineer)  
**Requested by:** Cristiano (ormasoftchile)  
**Context:** SQL Live-Site Operations accepted Phase 1B plan with eight corrections; three in David's territory.

---

## CORRECTION A — Profileless Non-Interactive Fail-Fast (moved into Phase 1B)

### Verdict: Accept. Implementation straightforward; harness impact is real and scoped.

### Where `TTYOutput: true` is hardcoded

`cmd/gert/run.go:166` — the `WireOptions` literal passed to `adapter.BuildEngineConfig`:
```go
TTYOutput: true,
```
This is the sole source. It controls two downstream gates:

- `internal/adapter/wire.go:306–313` — `buildApprovalGate()`: if TTYOutput (or `Attended` override) is true → `TerminalApprovalGate`; else → `NoOpApprovalGate`.
- `internal/adapter/wire.go:306` — `buildInputProvider()` and `buildPromptProvider()`: TTYOutput gates whether a real terminal prompt provider is installed.

The `Attended` override path (`opts.Attended != nil`) was added in Phase 1A. When a profile with declared `attendance: unattended` is supplied, `attendedFromProfile()` (`run.go:744`) returns `&false`, and `buildApprovalGate()` picks `NoOpApprovalGate` regardless of `TTYOutput`. This is the correct CI path — **but only when a profile is present**.

### Non-interactivity detection

**Confirmed: zero `isatty()` calls anywhere in the codebase.** Grep for `isatty`, `ModeCharDevice`, `Stdin.Stat`, `golang.org/x/term`, `go-isatty` — all return zero matches in `.go` source files.

**Dependency situation:**
- `golang.org/x/term` is **not** in `go.mod` (direct or indirect).
- `github.com/mattn/go-isatty` is **not** in `go.mod`.
- `golang.org/x/sys` **is** present as an indirect dependency (required by OpenTelemetry).

**Recommended approach: stdlib only, no new dependency.**

```go
fi, err := os.Stdin.Stat()
nonInteractive := err != nil || (fi.Mode()&os.ModeCharDevice == 0)
```

`os.Stdin.Stat()` returns file info; `ModeCharDevice` is set when stdin is a real terminal. When stdin is a pipe, redirected file, or `/dev/null`, `ModeCharDevice` is clear. This is the standard Go idiom (used by the Go toolchain itself). No new import beyond `os`.

On Windows: `os.Stdin.Stat()` works identically via Go's runtime syscall layer. No platform-specific code needed.

### Where the fail-fast belongs

**CLI entry in `run.go`, after the profile check block (line ~110), before `BuildEngineConfig` (line ~162).** Specifically:

```
[parse flags]
[parse runtimeProfile — line 101–110]
← INSERT fail-fast here ←
[build plannerImpl]
[BuildEngineConfig]
[run execution]
```

The check must fire before `BuildEngineConfig` because that call is where `TerminalApprovalGate` is installed. Firing it at planner preflight would be too late (planner is constructed after line 110, used later during planning). Firing it at gate construction is deep in adapter internals — wrong layer. CLI entry is the right seam: it owns the profile check and can fail before any side effects.

Required output:
```
error: non-interactive execution requires an unattended runtime profile
fix: pass --profile <profile>
```
Exit code: existing `exitValidation` (exit 1) is appropriate — this is a configuration error, not a runtime failure.

### Harness impact — this is a real blocker

**The conformance harness breaks.** `internal/conformance/enum_harness.go:runCLI()` invokes `gert run <entry> --trace <path>` with no `--profile` and no TTY — stdin is a pipe or `/dev/null` inside `exec.Command`. Every vector (31 conformance + all enum + all dyninclude) would hit the fail-fast and get exit-validation instead of the expected outcome.

The harness comment already acknowledges the profileless-run hazard: `dyninclude_conformance_test.go:116` notes it runs "without TTY so gets NoOpApprovalGate." The fail-fast closes exactly this gap at the CLI boundary.

**Required harness opt-in:** The harness must supply a minimal unattended test profile when invoking `gert run`. Cleanest approach:

1. Add a fixture file `internal/conformance/testdata/unattended-test.profile.yaml` with `attendance: unattended` and no tool overrides.
2. In `runCLI()` (`enum_harness.go`), append `"--profile", profileFixturePath` to the `exec.Command` args.

This is a deliberate author assertion that the conformance harness runs in unattended mode — consistent with its existing `TTYOutput: false` behavior, now made explicit at the CLI boundary rather than only at the adapter layer.

**No vector results change.** The harness already uses `NoOpApprovalGate` (confirmed by the existing skip comment). Supplying an unattended profile selects the same gate through the declared-attendance path rather than the `TTYOutput=false` path. Behavior is identical; the fail-fast is satisfied.

### Decision

- Fail-fast lives at `cmd/gert/run.go` after profile check, using `os.Stdin.Stat()`/`ModeCharDevice`. No new dependency.
- Detection at CLI entry, before `BuildEngineConfig`. Interactive operators with no profile pass through (TTY detected, no fail).
- Harness updated: `internal/conformance/enum_harness.go` injects `--profile <unattended-test.profile.yaml>`. Harness behavior unchanged; fail-fast satisfied.
- `internal/e2e/helpers_test.go:171` already uses `TTYOutput: false` directly (bypasses CLI entry) — no change needed there.

### Estimate

- Fail-fast implementation: 0.5 days (20-line check + error message + test)
- Harness opt-in + fixture: 0.5 days
- Full regression pass to confirm zero breaks: 0.5 days
- **Total: 1.5 days**

---

## CORRECTION B — INDETERMINATE Must Retain Trace-Safe Invocation Evidence

### What the existing step record persists

`pkg/engine/run.go:160–190` — `StepResult` struct. Fields:

| Field | Type | Present? |
|-------|------|----------|
| `StepID` | string | ✓ |
| `Status` | StepStatus | ✓ (needs new `indeterminate` value) |
| `Output` | map[string]any | ✓ (but ambiguous for "no result" — see below) |
| `StartedAt` | time.Time | ✓ |
| `CompletedAt` | time.Time | ✓ |
| `DurationMs` | int64 | ✓ |
| `Error` | error | ✓ (carries transport error string, not category) |
| Logical tool name | — | **ABSENT** |
| Logical action name | — | **ABSENT** |
| Classification | — | **ABSENT** |
| Endpoint host | — | **ABSENT** |
| Attempt number | — | **ABSENT** |
| Deadline (context deadline) | — | **ABSENT** |
| Transport error category | — | **ABSENT** |

Run ID lives in `RunState.RunID` (not in `StepResult` directly) — but it is always available at the call site when StepResult is recorded.

**7 of the 7 required new fields are absent from the current struct.**

### "Never credentials" requirement

The codebase has two redaction seams:

1. **Auth token redaction in transport output** (`internal/tool/mcp_http_tess_test.go:979`, `internal/tool/auth_azurecli_test.go:244`): scrubs bearer tokens from MCP result content and error messages. This is output-layer redaction.
2. **Runbook-level `redact:` block** (`schemas/runbook.schema.json:262`): author-declared redaction rules for variable substitution values.

Neither seam is directly reusable for the indeterminate record. However, the endpoint host requirement is simpler: **the host is not a credential.** The host comes from `def.URL` (the tool definition's URL field, which is transport configuration — not auth). We store the hostname/host-portion only, not the full URL with any query params or path tokens that might embed credentials.

**No new redaction infrastructure is required.** The implementation rule is: record `url.Parse(def.URL).Host`, never the auth provider's token, never the raw `Authorization` header value. This is a code-review invariant, not a new library seam. Document it in the `IndeterminateRecord` field comment.

### "Must not require fabricated output" constraint

`StepResult.Output` is `map[string]any`. Its zero value is `nil`. **The problem:** a tool that successfully returned an empty response also produces `Output: nil`. There is currently no way to distinguish "timed out before any response" from "returned nil outputs."

**With `StepStatusIndeterminate` as the status, this ambiguity resolves.** `StepStatusIndeterminate` means by definition that completion is unknown — the `Output` field carries no semantic weight for that status value. However, to make this explicit and defensible against future code that might zero-check `Output`:

- Add `CompletionUnknown bool` to `StepResult`, set to `true` only when `Status == StepStatusIndeterminate`.
- Alternatively, embed an `*IndeterminateRecord` pointer on `StepResult` that is non-nil only for indeterminate steps — its presence is the signal.

Recommendation: the embedded pointer approach. `IndeterminateRecord` is the evidence record, and its non-nil presence is the "completion unknown" signal. `Output` is left nil for indeterminate steps; code that reads `Output` for display or capture must gate on status first (which it already must for failed steps).

### Proposed `IndeterminateRecord` struct

```go
type IndeterminateRecord struct {
    RunID             string        // always available at recording site
    StepID            string        // mirrors StepResult.StepID
    ToolName          string        // logical tool name from ResolvedStep
    ActionName        string        // logical action name from ResolvedStep
    Classification    string        // schema.ToolAction.Classification
    EndpointHost      string        // url.Parse(def.URL).Host — never credentials
    AttemptNumber     int           // 1-based; always 1 in Phase 1B (no retry)
    Deadline          time.Time     // context deadline at step start; zero if none
    FailureTime       time.Time     // time.Now() at indeterminate detection
    TransportErrCategory string     // "deadline-exceeded" | "transport-lost" | "unknown"
}
```

Serialized as part of `RunState` (the store already marshals `StepResult` via `SaveState`). `gert resume` reads it to surface the indeterminate evidence without requiring any tool result payload.

### Revised estimate for Item 3

Previous estimate: 4–5 days.

Additional scope from this correction:
- `IndeterminateRecord` struct definition and JSON serialization tags: 0.5 days
- Recording site in engine timeout path (extract tool/action/classification from `ResolvedStep`): 0.5 days  
- Resume-guard reading and displaying `IndeterminateRecord` fields: 0.5 days
- Tests: add evidence-record assertions to the 8+ timeout/lost-transport vectors: 0.5 days

**Revised estimate: 5.5–6 days.**

---

## CORRECTION C — ICM Proof Must Target the Real Contract

### Does `--package-map` + `--profile` composition work at execution time?

**`--package-map`: YES — confirmed execution-wired.**

`TestRun_PackageMap_RealVsMockBinding` (`cmd/gert/packagemap_integration_test.go:23`) proves it end-to-end: the same unchanged runbook resolves a different tool binary based on `--package-map`, and the step result reflects the mock binary's output. The package-map feeds `mergePackageBindings()` → `pkgcatalog.Build()` → tool registry → executor. It is full-stack execution machinery, not planner-only.

Citation: `cmd/gert/run.go:272` — `loadPackageMap(*packageMapPath)` called before `BuildEngineConfig`; its merged catalog feeds `newToolRegistry()` → `adapter.WireOptions.ToolScanDir` path and the overlay registry used at execution time.

**`--profile`: NOT execution-wired yet.** Confirmed from Phase 1B scope — this is Phase 1B Item 2. Profile is plan-time only today (`engine.go:127/250` ProfileEvaluator only). `adapter.WireOptions` has no Profile field.

**Consequence for the ICM proof:** The proof CANNOT exercise `--profile` at execution (endpoint/auth override) until Item 2 ships. The proof is blocked on Item 2 by design — the requirement to exercise "a direct HTTP managed-identity binding selected through package-map, with the profile not rewriting transport mode" requires profile-to-transport wiring that does not exist yet.

### Contract parity assertion infrastructure

**No existing infrastructure compares outputs across two bindings of the same logical tool.** `TestRun_PackageMap_RealVsMockBinding` asserts the bindings produce *different* outputs ("real-binding" vs "mock-binding") — the inverse of parity. Tess's MCP parity tests (`TV-MCP-PARITY-001 through -004`, `mcp_http_tess_test.go:842`) compare SSE vs JSON delivery of the same binding — same tool, same server, different transport framing. Neither is cross-binding parity.

A contract parity test requires: run the runbook twice (once with native mock binding, once with mcp-http binding), capture structured outputs from both, assert that `title`, `service`, `environment`, `logical_server`, `database` fields match across both runs. This is new test infrastructure — probably ~1 day to build the comparison harness for this specific scenario.

### Native mock binding — what Gert supports today

The existing pattern is an `httptest.Server` (for mcp-http) or a real subprocess binary (for subprocess/stdio). For the ICM proof:

- **mcp-http binding**: `httptest.NewServer` returning valid MCP JSON responses for `icm.get-incident` and `tsg-recommendation.recommend`. Existing pattern in `internal/tool/mcp_http_tess_test.go`. A `--package-map` file points to a tool definition whose URL is the `httptest` server's URL.
- **Native mock binding**: A small Go binary (`cmd/tools/icm-mock` or similar) that accepts MCP stdio and returns deterministic responses. Existing pattern: `cmd/tools/echo`. A `--package-map` file points to a tool definition whose transport is `mcp-stdio` pointing at the mock binary.

Both bindings use the same logical tool name and action — that's the parity contract. The `--package-map` file selects which binding is used without touching the runbook.

### What we need FROM them — blocking dependency list

This list is exact. The proof cannot start until all items are received.

| # | Artifact | Why required |
|---|---------|--------------|
| 1 | `gert-sqllivesite/runbooks/icm-tsg-router.runbook.yaml` — full file content | The proof must use their actual runbook, not a surrogate |
| 2 | Full YAML tool definition for `icm.get-incident` — action name must use hyphen, complete input schema (all parameter names and types), output schema with all five fields: `title`, `service`, `environment`, `logical_server`, `database` and their types | Required to build contract-identical mock server and validate output parity |
| 3 | Full YAML tool definition for `tsg-recommendation.recommend` — both outcome shapes: `suggested` (full field set) and `no-suggestion` (full field set) including field names and types | Required to build both mock outcomes and assert parity across bindings |
| 4 | Their `--package-map` file (or its equivalent config) that maps `icm.get-incident` and `tsg-recommendation.recommend` to their tool file paths/packages | Required to replicate the binding selection the proof must exercise |
| 5 | Transport mode declaration for each tool in production: whether `icm.get-incident` is `mcp-stdio` or `mcp-http` in their current production binding | Required to determine which native mock binding is contract-identical |
| 6 | Whether the runbook uses `toolRefs:` with package names or bare path references | Determines how `--package-map` must be structured for the proof |

Until these are received, Item 4 (ICM proof) is on hold regardless of Item 1/2/3 completion.

### Estimate for Item 4 under new contract

- Contract-identical mock server (httptest for mcp-http + stdio mock for native): 1 day
- Cross-binding parity assertion harness: 1 day  
- Profile-at-execution wiring through `--profile` (depends on Item 2): included in Item 2 (3–4 days)
- Both recommendation outcomes (`suggested` / `no-suggestion`): 0.5 day
- Integration test exercising `--package-map` + `--profile` together: 0.5 day

**Item 4 total: 3 days after Item 2 ships AND artifacts received.** (Previous estimate: 1 day — was scoped against our sample tool definition, which is not their contract. New scope is materially larger.)

**Item 4 cannot start until:**
1. Item 2 (profile runtime binding, 3–4 days) ships
2. All 6 artifacts above are received from SQL Live-Site Operations

---

## Revised Phase 1B Estimate

| Work | Previous | Revised | Delta |
|------|----------|---------|-------|
| Item 1: Managed Identity | 2 days | 2 days | — |
| Item 2: Runtime Binding | 3–4 days | 3–4 days | — |
| Item 3: INDETERMINATE semantics + evidence record | 4–5 days | 5.5–6 days | +1.5 days |
| Item 4: ICM proof (new contract) | 1 day | 3 days | +2 days |
| **Correction A: Fail-fast + harness** | deferred | 1.5 days | +1.5 days (new) |
| **Total parallelized** | ~6–7 days | ~8–9 days | +2–2.5 days |

Item 4 remains serial after Item 2 AND gated on external artifact delivery.


---

# Phase 1B Rev 2 — Ratified Design Rulings

**Date:** 2026-08-17
**Author:** Barbara (Lead/Architect, Gert Core)
**Status:** Ratified in Phase 1B Rev 2 plan

---

## Ruling A: Profile Auth Schema Invariant

When Item 2 extends the `RuntimeProfile` schema, the profile:

- **MAY** add a `Provider` field (credential mechanism selection — e.g., `managed-identity` vs. `azure-cli`).
- **MUST NOT** add `Scope` or `AllowedHosts`.

The gate is always constructed from `def.Auth.Scope` and `def.Auth.AllowedHosts` — the tool definition's declared contract. The profile substitutes *who acquires* the token, never *what it is scoped for* or *where it may go*.

**Current state:** `RuntimeProfile` has no top-level `Auth` field today. `ProfileToolOverride` has only `Endpoint` and `Mode`. Scope-clobbering is structurally impossible. This rule ensures it remains so as the schema evolves.

**Implements:** Their requirement that top-level auth "must not silently replace a tool-specific audience, scope, or host policy."

---

## Ruling B: `*IndeterminateRecord` as Non-Fabricated-Output Representation

`StepResult.Output == nil` is ambiguous (a tool returning empty outputs also yields nil). The representation for "completion unknown" is:

- Embed `*IndeterminateRecord` on `StepResult`.
- Non-nil pointer = completion unknown; `Output` remains nil (not fabricated).
- Fields: `RunID`, `StepID`, `ToolName`, `ActionName`, `Classification`, `EndpointHost`, `AttemptNumber`, `Deadline`, `FailureTime`, `TransportErrCategory`.

**Design rationale:** A nil `Output` combined with a nil `IndeterminateRecord` means "tool returned empty output" (a valid result). A nil `Output` combined with a non-nil `IndeterminateRecord` means "we do not know what happened" (trace-safe evidence without fabrication).

**Implements:** Their requirement that trace-safe evidence "must not require fabricated output."


