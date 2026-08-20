# Archived Decisions Block — Archived Phase 1B Rev2/Rev3 implementation

**Effort slug:** runtime-portability-phase1b-rev2-rev3-implementation  
**Source:** .squad/decisions.md lines 1339–2309 at archival time  
**Archived at:** 2026-08-19T17:08:50-07:00  
**Ruling:** .squad/decisions.md live entry `2026-08-19 — decisions-ledger-archival-ruling`  
**Payload SHA-256:** 8228a7e975aae50eb1c3825eb0b00b4c78d005121407420c37a3c79a1ef1a9f3  
**Payload bytes:** 58036

---

## PHASE 1B REV 2: Corrections and Scope

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





---

# Decision: Contract Proof Ownership Split (Phase 1B Rev 3)

**Date:** 2026-08-17
**Author:** Barbara (Lead/Architect, Gert Core)
**Status:** Active

## Ruling

Gert core does not take a dependency on consumer repositories for contract proofs. Consumer-specific contracts are owned and proven by the consumer team in their own repo.

## Applies to

- `gert-sqllivesite` ICM contract (`icm.get-incident`, `tsg-recommendation.recommend`)
- Any future consumer-specific contract proof

## Rationale

1. **Dependency direction:** Gert is a library/tool; consumers depend on it, not the reverse. Importing consumer contracts into Gert means Gert's CI breaks when their contract evolves.
2. **Precedent:** Round 1 of the Phase 1 negotiation established this boundary when `run-gert.ps1` was identified as a `gert-sqllivesite` artifact and removed from Gert scope.
3. **Separation of concerns:** Gert proves that the binding mechanism works (package-map, profile composition, transport mode integrity, parity across bindings). Consumers prove that their specific contract passes through that mechanism correctly.

## What Gert owns

- Synthetic fixtures structurally equivalent to real contracts (multi-field typed outputs, branching outcomes)
- The contract-parity harness infrastructure
- Mechanism correctness: `--package-map` + `--profile` composition, transport mode integrity, credential attachment

## What consumers own

- Their actual tool definitions and schemas
- Runbooks exercising their specific tools
- End-to-end proof that their contract works under both bindings

## Exports available to consumers

- The parity harness can be exported as a reusable helper if requested.
- Documentation and worked examples for writing consumer-side proofs.

---

# Ruling: Reachability Gate Design

**Date:** 2026-08-17
**Author:** Ken
**Status:** SHIPPED — commit 56b1bc4

## Problem

Four schema fields were declared, schema-validated, unit-tested, and never
read by production code. In every case the test suite was green. The pattern:
a unit test reads the field from the struct; no test exercises the field
through `cmd/gert/run.go`. Phase 1B adds new fields with the same shape.

## Decision

**A reachability test must prove a field changes observable CLI behavior
through the real `runRun()` code path.** Reading the field in a unit test
that bypasses `cmd/gert/run.go` does NOT count.

## Mechanism

A registry table in `cmd/gert/reachability_registry_test.go` with
`TestCLI_ReachabilityGate` as the enforcement test. Every schema field or
CLI feature must appear as either:
- `statusReachable` with a non-nil `TestFunc` (the reachability probe), or
- `statusKnownDead` with a non-empty `DeadReason` citing the item that will wire it.

There is no silent option. A `statusReachable` entry with `nil TestFunc`
fails CI immediately. A `statusKnownDead` entry with empty `DeadReason`
also fails CI.

## Alternatives considered

- **Lint-style scanner**: would need to parse YAML schemas and match Go
  struct fields; brittle and complex. Rejected.
- **TestCLI_*_Reachable naming convention alone** (Barbara's proposal):
  a convention without enforcement. Engineers forget. Rejected.
- **Registry with only live entries**: forces engineers to write the probe
  before the field is wired (impossible). The KNOWN-DEAD escape valve is
  essential; it makes the dead state explicit and traceable rather than silent.

## Four founding entries

| Feature | Status | Note |
|---|---|---|
| AllowedEnvironments | REACHABLE | PLAN-010 via `runRun()`; probe confirms |
| RequiresCapabilities | KNOWN-DEAD | No active Phase 1B item |
| ProfileToolOverride.Endpoint | KNOWN-DEAD | David Phase 1B Item 2 |
| Contract.Idempotent | KNOWN-DEAD | Phase 2 retry/idempotency |

## CI wiring

`go test ./...` in `.github/workflows/go-test.yml` runs `TestCLI_ReachabilityGate`
as part of the `cmd/gert` package. No CI config changes were needed.

## Negative-control verification

Setting `TestFunc: nil` on the `AllowedEnvironments` entry produces:
```
--- FAIL: TestCLI_ReachabilityGate/AllowedEnvironments
    feature "AllowedEnvironments" is marked statusReachable but TestFunc is nil
FAIL
```
The gate provably fires.

---

# Decision: Managed Identity Provider Shape (Phase 1B Item 1)

**Date:** 2026-08-17
**Author:** Don (Backend, Gert Core)
**Status:** Implemented and committed

---

## Context

Phase 1B Item 1 required implementing `managed-identity` as an IMDS-only auth provider. Two design decisions were non-obvious and are recorded here.

---

## Decision 1: `imdsHTTPClient` function injection seam

**Decision:** The provider holds a `httpClient imdsHTTPClient` field typed as `func(*http.Request) (*http.Response, error)`. In production it is nil (falls back to a real `http.Client`). In tests it is replaced with a closure that rewrites the request URL to point at an `httptest.Server`.

**Alternatives considered:**
- Inject an `*http.Client` — would require either a custom `Transport` or replacing the entire client. A `Transport` rewrite is more complex and harder to read.
- Use an `http.RoundTripper` interface — adds an interface definition and a wrapper type just to rewrite a URL.
- The function seam is already the pattern used elsewhere in this codebase (see `azRunner` in `auth_azurecli.go`) — consistent.

**Why:** Single-function seam is the smallest, most readable test shim for a provider that makes exactly one kind of HTTP request. No interface definition required.

---

## Decision 2: `NewAuthProviderWithClientID` rather than config struct expansion

**Decision:** Added `NewAuthProviderWithClientID(provider, scope, clientID string)` as a secondary constructor alongside `NewAuthProvider`. `NewAuthProvider` calls it with `clientID=""`. The `clientID` comes from a future `AuthConfig.ClientID` field (not added in this item).

**Why not add `ClientID` to `AuthConfig` now:**
- Adding schema fields has downstream effects (YAML parsing, validation, documentation).
- `AuthConfig.ClientID` is only meaningful for `managed-identity`. Adding it to the shared struct without wiring it through `ValidateTransportConfig` (i.e., warning on non-MI providers) risks silently ignored config.
- The constructor exists so Item 2 (Runtime Binding) can wire `clientID` when it reads `AuthConfig` and constructs the provider. The schema change belongs in Item 2.

**Invariant established:** `NewAuthProvider` is the public surface. `NewAuthProviderWithClientID` is the extension point for runtime binding. No caller needs to know about `clientID` until Item 2 lands.

---

## Decision 3: No ambient environment variable inspection

**Decision:** `ManagedIdentityAuthProvider` contains zero `os.Getenv` calls. It does not check `AZURE_FEDERATED_TOKEN_FILE`, `AZURE_CLIENT_ID`, `AZURE_TENANT_ID`, or any other Azure SDK ambient environment variable.

**Why:** Correction 1 from SQL Live-Site Operations was explicit: managed-identity and workload-identity must be separate providers precisely because a provider that silently chooses its mechanism based on ambient state makes identical profiles behave differently across hosts. The determinism requirement is structural, not policy — a code reviewer can verify it by `grep`ing for `os.Getenv` in the file.

---

## What Item 2 Must Do

When Item 2 (Runtime Binding) wires `AuthConfig` through the runtime:
1. Add `ClientID string` to `AuthConfig` (schema change, YAML field `client_id`).
2. Validate in `ValidateTransportConfig` that `client_id` is only set when `provider: managed-identity`.
3. Call `NewAuthProviderWithClientID(provider, scope, auth.ClientID)` instead of `NewAuthProvider`.

---

# Decision: Profile Execution Wiring + PLAN-013

**Date:** 2026-08-17
**Author:** David (Integration)
**Status:** Shipped — commit `85bfa4a`, gert repo main branch
**Related:** Phase 1B Item 2; counterparty correction 4; PLAN-013

---

## Context

`gert run --profile <id>` parsed and validated profiles but the selected
endpoint and auth parameters never reached the transport. The counterparty
flagged this as their outstanding item 4. `ProfileToolOverride.Endpoint` was
one of four instances of the project's systemic dead-field bug.

---

## Decisions Made

### 1. Threading path: profile flows through WireOptions, not a separate lookup

Profile is passed `adapter.WireOptions.Profile` → `BuildEngineConfig` →
`DefaultToolRuntime.SetProfile(profile)`. This is the same thread as
`--package-map` wiring. The profile is NOT re-read from disk inside the
runtime; it is set once before the first invocation. Call `SetProfile` before
returning from `BuildEngineConfig` so callers cannot forget.

### 2. Ratified Rule A enforcement is structural, not runtime assertion

`ProfileToolOverride` has no `Scope` or `AllowedHosts` fields. Those fields
cannot be set through a profile — the schema physically prevents it. `Provider`
IS added to `ProfileToolOverride` so the caller can specify which credential
mechanism acquires the token. `TokenGate` ALWAYS reads `def.Auth.Scope` and
`def.Auth.AllowedHosts` from the tool definition, even when Provider comes
from the profile.

### 3. PLAN-013 ships in the same commit as endpoint-override wiring

No window exists where overrides execute unvalidated. If the check were
separate, an attacker or misconfiguration in that window could forward a bearer
token to an unapproved host. The constraint is: one commit, both changes, or
neither lands.

### 4. PLAN-013 check lives in `checkToolEnvironmentPreflight` (non-test branch)

The seam already receives `def` and `profile`. PLAN-013 runs inside the
`profile.Context != schema.ProfileContextTest` branch because test-context
mcp-http is already unconditionally blocked by PLAN-012 (and PLAN-013 only
applies to mcp-http). For non-test contexts, the check validates:
- Tool transport is mcp-http (otherwise endpoint override is a no-op)
- Tool has auth configured with AllowedHosts (required for the check)
- Override endpoint's hostname is in AllowedHosts (exact, case-insensitive)

Failure: `ErrEndpointHostNotAllowed` at plan time, before step 1 executes.

### 5. Reachability test uses two in-process MCP servers

`TestCLI_ProfileEndpointOverride_Reachable` (in `internal/tool/runtime_profile_test.go`,
`package tool`) creates two httptest.Server instances: `defaultSrv` (written
into the tool definition URL) and `overrideSrv` (written into the profile's
per-tool endpoint). With the profile wired in, all traffic goes to `overrideSrv`.
Removing the `effectiveURL = override.Endpoint` read causes traffic to flow to
`defaultSrv` and the assertion fails. Plain HTTP is used because
`ValidateTransportConfig` (HTTPS enforcement) is called at scan time, not at
runtime; bypassing YAML parsing bypasses that check cleanly.

The test is written standalone per the "TestCLI_<Feature>_Reachable" convention
(Ken's pattern); a shared helper can adopt it later without changing the test's
logic.

### 6. Provider override uses the same `NewAuthProvider` function

When a profile override specifies Provider, the runtime calls
`NewAuthProvider(override.Provider, def.Auth.Scope)` — same function as the
non-override path. If the profile provider name is not recognized, the error
propagates from `NewAuthProvider` (MCP-002). The `_` discard of the error is
intentional: the transport factory is called lazily; the caller's first
`Invoke` will propagate the error. This mirrors existing behavior for the
non-override path.

---

## Constraints Preserved

| Constraint | Mechanism |
|---|---|
| Scope/AllowedHosts never from profile | No fields on ProfileToolOverride for them |
| TokenGate always from def | runtime.go reads `def.Auth.Scope`, `def.Auth.AllowedHosts` unconditionally |
| Transport mode never rewritten by profile | PROF-001 at parse time; no mode field in execution path |
| Endpoint override validated before execution | PLAN-013 same commit; preflight fires before plan.Start |
| --package-map and --profile compose | Package map selects tool def; profile parameterizes it; neither touches the other's domain |

---

## Files Changed

| File | Change |
|---|---|
| `pkg/schema/profile.go` | Added `Provider` field to `ProfileToolOverride` |
| `pkg/planner/planner.go` | Added `ErrEndpointHostNotAllowed` (PLAN-013 sentinel) |
| `internal/planner/preflight.go` | PLAN-013 check in `checkToolEnvironmentPreflight` |
| `internal/planner/preflight_test.go` | Six PLAN-013 unit tests |
| `internal/adapter/options.go` | Added `Profile *schema.RuntimeProfile` to `WireOptions` |
| `internal/adapter/wire.go` | `toolRuntime.SetProfile(opts.Profile)` after construction |
| `internal/tool/runtime.go` | `profile` field + `SetProfile` + wired in `Invoke` (mcp-http) |
| `internal/tool/runtime_profile_test.go` | `TestCLI_ProfileEndpointOverride_Reachable` reachability test |
| `cmd/gert/run.go` | `Profile: runtimeProfile` in `WireOptions` literal |

Commit: `85bfa4a` on gert repo main branch.

---

# Decision: Phase 1B Item 3 — INDETERMINATE Semantics

**Date:** 2026-08-17  
**Author:** Tess  
**Status:** SHIPPED — commit 22ad3e7

---

## What Was Shipped

**`pkg/engine/run.go`**
- `StepStatusIndeterminate StepStatus = "indeterminate"` — distinct from Failed and Completed.
- `RunStatusIndeterminate RunStatus = "indeterminate"` — run-level halt state, not terminal failure.
- `IndeterminateRecord` struct with all 7 required evidence fields (ratified by counterparty).
- `*IndeterminateRecord` on `StepResult` (pointer, not value, per Ratified Rule B — nil-output is ambiguous for tools that legitimately return nothing; a pointer gives an unambiguous signal).
- `AcknowledgeIndeterminate bool` on `RunOptions` — the engine-side half of `--acknowledge-indeterminate`.
- `ErrIndeterminate` and `ErrIndeterminateAcknowledgmentRequired` sentinel errors.

**`pkg/schema/tool.go`**
- `Idempotent *bool` on `ToolAction` — declares an action safe to retry after transport loss. Only meaningful for `classification: read-only`; ignored (and never retried) for mutating/destructive/unspecified.

**`internal/engine/engine.go`**
- Replaced unconditional `failRun()` after `execErr != nil` on tool steps with classification-aware branching:
  - `read-only` → `failRun` (retry future work when `idempotent: true`)
  - `mutating` → `haltIndeterminate`
  - `destructive` → `haltIndeterminate`
  - `unspecified` (nil) → `haltIndeterminate` (conservative)
- Same branching applied to the step-result path (executor returns `StepStatusFailed` + `context.DeadlineExceeded` without an `execErr`).
- `haltIndeterminate(...)` — emits `step/indeterminate` and `run/indeterminate` trace events, sets `RunStatusIndeterminate`, closes the run handle.
- `buildIndeterminateRecord(...)` — populates all 7 fields; `EndpointHost = url.Parse(URL).Hostname()` only (credential invariant enforced here).
- `resolveStepClassification(...)` — extracts `ToolAction.Classification` from the plan's tool map.
- `requiresIndeterminate(classification *string) bool` — returns `true` for mutating/destructive/unspecified, `false` for read-only. **The approval state (`RequiresApproval`) is never consulted here — this is the orthogonality contract.**
- `isTransportLoss(err)` — detects `context.DeadlineExceeded` and `context.Canceled`.
- Resume guard: `Resume()` returns `ErrIndeterminateAcknowledgmentRequired` when `state.Status == RunStatusIndeterminate && !opts.AcknowledgeIndeterminate`.

---

## Key Decisions

### Rule B: pointer, not sentinel
`*IndeterminateRecord` is a pointer because `Output == nil` is ambiguous — a tool that returns nothing also yields nil Output. The non-nil pointer is an unambiguous signal that completion could not be established. Do not encode this as a sentinel value in any existing field.

### EndpointHost invariant
`EndpointHost` is set via `url.Parse(toolDef.Transport.URL).Hostname()` — the parsed hostname only. Under no circumstances do credentials, tokens, Authorization header values, query strings, or full URLs appear in this field, its JSON serialization, trace events, or error messages. The credential sweep test (INDET-010) uses a high-entropy sentinel to verify this non-vacuously.

### Orthogonality: approval state is never consulted in timeout routing
`RequiresApproval` (including `false`) is an approval-routing opt-out only. It never assigns, implies, or coerces `classification: read-only`. The `requiresIndeterminate()` function receives only the classification pointer; the approval state is not passed. INDET-012 vectors prove this with nil/false approval combinations on mutating actions.

### unspecified → INDETERMINATE (conservative)
When `classification` is nil (unspecified), `requiresIndeterminate` returns `true`. This is the conservative default: we cannot know whether an unspecified action is side-effect-free. If callers want read-only timeout behavior they must declare `classification: read-only`.

### context.Canceled treated as transport loss
Both `context.DeadlineExceeded` and `context.Canceled` are treated as transport loss. `context.Canceled` is included because the engine may cancel a step context for reasons outside the engine's own run-cancel path (e.g., a step-level timeout context). The category string distinguishes them (`context-canceled` vs `context-deadline-exceeded`).

### Non-timeout errors are not affected
A non-transport error (e.g., malformed response, auth failure) on any classification still calls `failRun()`. The INDETERMINATE path is specifically for transport losses where the server may have processed the request despite the client receiving an error.

### Resume guard is engine-side only
The `--acknowledge-indeterminate` CLI flag wiring belongs in `cmd/gert/run.go` (David's file). The engine-side behavior is complete: `Resume()` returns `ErrIndeterminateAcknowledgmentRequired` when the guard fires. The CLI flag is a thin wire step.

---

## Test Coverage

30 vectors in `internal/engine/indeterminate_test.go`:

| Vector | What It Proves |
|--------|---------------|
| INDET-001 | read-only timeout → StepStatusFailed, NOT INDETERMINATE |
| INDET-002 | mutating timeout → INDETERMINATE |
| INDET-003 | destructive timeout → INDETERMINATE |
| INDET-004 | unspecified timeout → INDETERMINATE (conservative) |
| INDET-005 | destructive ≠ read-only divergence (regression guard) |
| INDET-006 | All 7 fields populated in IndeterminateRecord |
| INDET-006b | Deadline recorded when context carries one |
| INDET-007 | context.Canceled categorized correctly |
| INDET-008 | Non-timeout error → failRun for all 4 classifications |
| INDET-009 | Step-result-level transport loss — both classifications |
| INDET-010 | Credential sentinel absent from record AND trace events; sweep non-vacuous |
| INDET-011 | Resume blocked without acknowledge; allowed with it; non-INDET run not affected |
| INDET-012 | RequiresApproval nil/false/true does NOT change classification or timeout routing |
| INDET-013 | step/indeterminate and run/indeterminate events emitted |
| INDET-014 | read-only+idempotent:true does not trigger INDETERMINATE |

**Destructive-vs-read-only divergence is proven by INDET-005**, which asserts the statuses differ and would fail if anyone made the paths converge again.

---

## Deferred

- Actual retry loop for `read-only` + `idempotent: true` (no retry infrastructure today; the classification check and flag exist, falling through to `failRun`).
- `cmd/gert/run.go`: wire `--acknowledge-indeterminate` flag → `RunOptions.AcknowledgeIndeterminate` (David's file, not edited).
- `legacy_unspecified_policy: allow` mentioned in design: not present in schema today; if added it must NOT relax retry, idempotency, or INDETERMINATE behavior (same orthogonality contract as approval state).

---

# Decision: Credential Non-Leakage Assertion Design (Phase 1B Item 6)

**Date:** 2026-08-17
**Author:** Don (Backend, Gert Core)
**Status:** Implemented and committed (c7decfd)

---

## Context

The counterparty's final acceptance criterion (criterion 8 of 8) is:
> *No credential appears in runbook state, results, traces, or errors.*
> *Test fails if the synthetic token is intentionally leaked.*

This is the criterion that was dropped from an earlier draft and then added back. The implementation is in `internal/tool/auth_credential_leak_test.go`.

---

## Decision 1: `credentialSweeper` pattern — collect all surfaces, scan once

Rather than checking each surface inline with `if strings.Contains(err.Error(), token)`, all swept values are registered in a `credentialSweeper` and a single `scan()` call at the end reports all matches.

**Why:** A per-surface inline check stops at the first leak. If a single error on the non-200 path and a trace field both leaked, the test would only report one. The sweeper reports all surfaces simultaneously — useful for debugging multiple simultaneous leaks.

**Why not a single large string.Contains over a marshaled aggregate:** Different surfaces have different types (errors, maps, structs). Marshal all of them and concatenate — you lose the surface name in the failure message. `credentialSweeper.scan()` reports `surface "trace[0].url_host" contains sentinel`, which is immediately actionable.

---

## Decision 2: Non-vacuity assertions in every positive test

Every success-path test explicitly asserts that the code under test actually DID something. `TestCredentialLeak_SuccessPath_TokenNotOnAnyOutputSurface` checks that the Authorization header was captured by the MCP server, proving the token was actually attached. `TestCredentialLeak_TraceEvent_NoTokenInAuthAttached` checks that the `mcp/authAttached` event was emitted.

**Why:** A test that asserts "the token does not appear on surface X" is vacuously true if the token is never acquired or the surface is never populated. Non-vacuity checks turn an absence assertion into a presence-then-absence assertion.

---

## Decision 3: Negative control is a permanent first-class test

`TestCredentialLeak_SweepDetectsIntentionalLeak` runs on every invocation of `go test ./internal/tool/...`. It deliberately injects the sentinel into 8 surface types and asserts the sweeper fires on each one.

**Why it must be permanent:** An absence-assertion test without a working sweep is the same as no test at all. This is the same failure mode as the dead-field bugs that have bitten this project four times. If someone changes the sweep mechanism and breaks it, this test will catch it immediately.

**Why 8 sub-cases instead of one:** Each surface type (plain string, trace kind, trace payload, trace JSON, ToolResult.Stdout, ToolResult JSON, IndeterminateRecord JSON, error string) uses a different code path through the sweeper. Testing all 8 proves the sweeper reaches all surfaces, not just the one the author had in mind.

---

## Decision 4: IndeterminateRecord is explicitly swept

`IndeterminateRecord.EndpointHost` is swept in its own test. The struct is serialized to JSON and the JSON is swept. An inline negative control inside the same test proves a bad EndpointHost value would be caught.

**Why explicit:** `IndeterminateRecord` is new (Tess's Item 3). It is the newest place a credential could hide — if a future engineer adds a `TokenValue` or `LastToken` field to the struct by mistake, the sweep will catch it. Making the sweep explicit rather than implicit in a general JSON serialization ensures it cannot be silently skipped.

---

## Decision 5: Test lives in `internal/tool`, not `cmd/gert`

The test exercises `ManagedIdentityAuthProvider`, `TokenGate`, and `MCPHTTPTransport` directly — all in `internal/tool`. No CLI layer is needed. This keeps the test fast (no subprocess), deterministic (no process environment), and offline (no network or credentials).

**What this does NOT cover:** The test does not sweep the full engine's `StepResult` JSON as written to a run store or JSONL trace file. That coverage would require wiring through the engine, which David owns. If Item 3 (IndeterminateRecord in the engine) or Item 2 (runtime binding) introduce new surfaces, those items should add their own leak tests.

---

## Surfaces covered (and their sweep coverage)

| Surface | Test |
|---------|------|
| `mcp/authAttached` trace event (all fields) | `TestCredentialLeak_TraceEvent_NoTokenInAuthAttached` |
| All trace events JSON-serialised | `TestCredentialLeak_SuccessPath_TokenNotOnAnyOutputSurface` |
| `ToolResult.Stdout` | `TestCredentialLeak_SuccessPath_TokenNotOnAnyOutputSurface`, `TestCredentialLeak_TokenNotInStepOutput` |
| `ToolResult.Stderr` | `TestCredentialLeak_SuccessPath_TokenNotOnAnyOutputSurface` |
| `ToolResult` JSON | `TestCredentialLeak_TokenNotInStepOutput` |
| `ToolResult.Output` map values | `TestCredentialLeak_SuccessPath_TokenNotOnAnyOutputSurface` |
| Error from non-200 IMDS | `TestCredentialLeak_Non200IMDS_ErrorDoesNotLeakToken` |
| Error from malformed JSON | `TestCredentialLeak_MalformedJSON_ErrorDoesNotLeakToken` |
| Error from context cancellation | `TestCredentialLeak_ContextCancellation_ErrorDoesNotLeakToken` |
| Error from MCP-012 host rejection | `TestCredentialLeak_MCP012_ErrorDoesNotLeakToken` |
| Error after Invalidate + failing re-acquire | `TestCredentialLeak_Invalidate_DoesNotExposeToken` |
| `IndeterminateRecord` JSON | `TestCredentialLeak_IndeterminateRecord_NoToken` |
| `IndeterminateRecord.EndpointHost` string | `TestCredentialLeak_IndeterminateRecord_NoToken` |

---

# Decision Record: Phase 1B Items 5+B — Profileless Fail-Fast + --acknowledge-indeterminate

**Date:** 2026-08-17  
**Author:** David (Integration)  
**Commit:** d1314cc  
**Canonical ledger:** `.squad/decisions.md`

---

## Decision 1: Injectable TTY detection for test compatibility

**Context:** The profileless non-interactive fail-fast check calls `isInteractiveTTY()`, which inspects `os.Stdin`. In `go test`, stdin is a pipe (non-TTY), so ALL existing profileless CLI tests would fail with the new check.

**Decision:** Delegate `isInteractiveTTY()` to a package-level function variable `interactiveTTYDetect`. The production default (`defaultInteractiveTTYDetect`) reads `os.Stdin.Stat()`. `TestMain` in `cmd/gert/test_main_test.go` overrides it to `func() bool { return true }` for all tests. Only `TestCLI_Profileless_NonInteractive_FailFast` temporarily restores the false-returning stub.

**Rejected alternative:** Adding `--profile` to every existing profileless test. Too wide, touches many files not owned by this agent, and obscures the purpose of those tests.

**Invariant:** `interactiveTTYDetect` is unexported. It is a testing seam, not a public API. No code outside `run.go` and `*_test.go` reads it.

---

## Decision 2: Fail-fast fires before engine construction

**Context:** The counterparty rejected "warn and continue" because a hung CI job is a real operational problem, not a cosmetic one.

**Decision:** The check fires immediately after the profile loading block, before `adapter.BuildEngineConfig` is called. This means no goroutines are started, no approval gate is installed, and no resources are allocated before the fast exit.

**Error format:**
```
error: non-interactive execution requires an unattended runtime profile
fix: pass --profile <profile>
```
Exit code: `exitValidation` (2) — this is a configuration error, not a runtime failure.

---

## Decision 3: --acknowledge-indeterminate wired only to Resume

**Context:** `engine.RunOptions.AcknowledgeIndeterminate` only matters for resume. A fresh `Start` cannot produce a prior INDETERMINATE state.

**Decision:** The flag is parsed unconditionally but wired only in the `eng.Resume` call, not in `eng.Start`. The `eng.Start` call does not receive `AcknowledgeIndeterminate`; it would be dead. This matches Tess's engine design intent.

---

## Decision 4: Conformance harness gets an unattended profile, not a TTY stub

**Context:** `internal/conformance/enum_harness.go` invokes `gert run` as a subprocess. Subprocesses always have non-TTY stdin. After the fail-fast check, the harness would fail on every invocation.

**Decision:** Add `internal/conformance/testdata/unattended-test.profile.yaml` and have `NewEnumHarness` resolve and pass it via `--profile`. The profile declares `attendance: unattended` and allows all approval scopes, so no conformance vector's semantics change.

**Verification:** Conformance counts before and after: **72 vectors / 62 pass / 10 skip / 0 fail** — identical.

---

## Decision 5: Reachability registry graduation in same commit

**Context:** `ProfileToolOverride.Endpoint` was `statusKnownDead` with a note to graduate it after Item 2. Item 2 shipped in 85bfa4a; the registry wasn't updated in that commit.

**Decision:** Graduate `ProfileToolOverride.Endpoint` to `statusReachable` in this commit (d1314cc) with probe `testCLI_ProfileToolOverrideEndpoint_Reachable` (PLAN-013 path: mcp-http tool + disallowed endpoint override → exitValidation). The probe fails if `effectiveURL = override.Endpoint` is removed from `runtime.go`.



---

