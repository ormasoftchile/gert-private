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

