# Gert Core Team Decisions Record

**Status:** Active working set (runtime-portability effort, Phase 1A and 1B)

## Archive Policy

Decisions are archived **by effort completion**, not by file size. This file contains all currently binding decisions for active efforts. When an effort completes and moves to historical reference only, its decisions are archived to .squad/decisions/archive/ as a unit. Size-based archiving causes active decision sets to be incorrectly archived out of the live working set.

---

## PHASE 1A: Runtime Portability (Active — Phase 1B Built On This)

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


---

# Phase 1 Slices 3–7 Gate Review

**Date:** 2026-08-17  
**Reviewer:** Barbara (Lead / Architect)  
**Verdict:** APPROVED-WITH-CONDITIONS  
**Scope:** Slices 3 (profile schema), 4 (ProfileApprovalGate + attendance), 5 (Tier 0 preflight), 7 (fixture migration + governance vectors), classification validation (GOV-014)

---

## Verdict: APPROVED-WITH-CONDITIONS

Two conditions, both non-blocking for merge but required before we report to the counterparty:

1. **[REQUIRED BEFORE REPORTING]** GOV-009/GOV-010 vectors do not actually catch silent routing coercion (see §A below). Add one behavioral test (not conformance vector) that exercises the ProfileEvaluator with `ToolApprovalTriState=&false + ToolClassification="destructive"` and asserts the classification field on StepInfo is unchanged after evaluation. This pins the "never assign or coerce" guarantee at the behavioral level. **Owner: Ken** (he owns the ProfileEvaluator; Don is locked out as original Classification author).

2. **[REQUIRED BEFORE PHASE 2]** David's `gert plan` command must emit the `config/no-binding-for-profile` message format we committed to — including the "note: this runbook runs correctly in profiles: X, Y" line. That line is the single highest-value operator output in the preflight design. It is computable from plan.Tools + discovered profile files. Verify it is present when the `gert plan` slice ships. **Owner: David.**

---

## Counterparty Validation Points — Status

| Point | Status | Evidence |
|-------|--------|---------|
| 1. GovernanceEvaluator wired in production | **DEMONSTRATED** (from Slice 2, reconfirmed: wire.go:161, run.go ×2, sub-engine ×1, per-run in Start/Resume) |
| 2. Policy composes runbook + per-tool | **DEMONSTRATED** (BuildEvaluator from GovernanceSource + ToolApprovalTriState on StepInfo + OR in evaluator) |
| 3. Approval applies to all transport paths | **DEMONSTRATED** (pre-dispatch chokepoint in executeStep, before executor) |
| 4. Approval never changes classification/retry | **DEMONSTRATED structurally** (different types, different fields, no derivation code) — but see §A for the test gap |

---

## §A. GOV-009/GOV-010 Weakness (Condition 1)

These vectors exercise `requires-approval: false + classification: destructive` (GOV-009) and `+ mutating` (GOV-010). They prove the COMBINATION is schema-valid and the runbook executes successfully. Their note claims "This vector would FAIL if anyone adds code that derives classification from requires-approval."

**That claim is only partially true.** The vectors would catch a REJECTION (code that errors on the combination). They would NOT catch a silent routing coercion inside the ProfileEvaluator (e.g., `if tristate == &false { classification = "read-only" }`), because:
- The conformance harness runs without a profile (NoOpApprovalGate)
- Routing changes in the ProfileEvaluator are invisible without a profile
- The runbook would still complete successfully

The real protection today is structural: `ToolClassification` is populated from `action.Classification` in engine.go:536, never derived from RequiresApproval. The ProfileEvaluator's legacy opt-out path (`if !*step.ToolApprovalTriState { result.RequiresApproval = false; return }`) only suppresses approval — it never touches classification. This is sound, but a regression test at the evaluator level would make it fail-safe.

---

## §B. Nil Profile Preservation

Verified. Three layers all nil-guard:
- `profileEvaluator.Evaluate()`: `if pe.profile == nil { return result, nil }` — transparent
- `checkToolEnvironmentPreflight()`: `if profile == nil { return nil }` — no checks
- `checkAttendancePreflight()`: `if profile == nil { return nil }` — no checks

No `--profile` flag → `runtimeProfile == nil` → all new code paths are inert. Existing behavior unchanged.

---

## §C. Unspecified Matrix — Verified Correct

The ProfileEvaluator's `default` branch (covering nil classification AND any unrecognized value) implements:
- Test context → auto-approve ✅
- Attended → gate fires (NOT warn-and-proceed) ✅
- Unattended + legacy_allow → suppress prompt only ✅
- Unattended + prompt (default) → deny ✅

The comment explicitly states the governing principle: "absence — or garbage — must never grant additional execution rights." Commit 31c0aa0 adds defense-in-depth tests for unrecognized values.

---

## §D. Legacy Unspecified Policy — Verified Correct

`legacy_unspecified_policy: allow` suppresses the approval prompt ONLY. The evaluator sets `result.RequiresApproval = false` and returns. It does NOT set `result.Allowed = true` with a different classification, does NOT modify ToolClassification, does NOT affect any retry or late-result field (none exist on EvaluationResult). The comment in the code says: "Suppresses the migration prompt only. Does NOT relax retry, idempotency, or late-result behavior."

---

## §E. CI Hang Hazard

The hazard is genuinely closed for profile-using paths. `attendedFromProfile()` extracts attendance → `WireOptions.Attended` → `buildApprovalGate()` selects NoOpApprovalGate when `Attended == &false`. A CI profile with `attendance: unattended` will never block on stdin.

Residual risk: profileless runs in CI where a tool declares `requires-approval: true`. This is the pre-existing `TTYOutput: true` hardcode bug, not introduced by this work. The Tier 0 attendance check (PLAN-011) catches the case where someone uses `--profile ci` with `attendance: attended` — it fails before execution. The profileless case remains a known limitation pending isatty() / `--unattended` flag (documented in preflight.go comments).

---

## §F. Test-Context Transport Rules — Verified Correct

- `native` → always allowed ✅
- `mcp` → blocked unless `transport.allow_subprocess_in_test: true` ✅
- `mcp-http` → NEVER allowed, no override ✅
- All error messages describe the subprocess opt-in as "an auditable author assertion, NOT a sandbox" ✅
- preflight.go header comment explicitly states "subprocess inherits the full parent environment (exec.Command + mergeEnv uses os.Environ())" ✅

---

## Ratification Decisions

### 1. Error class: PKG-010 for classification validation — RATIFIED

`PKG-010` already covers "package export references a file that fails any validation check." Classification validation is such a check. The error message includes the full actionable detail (invalid value, allowed values list). Inventing a new `TOOL-PARSE` class would fragment the error namespace without functional benefit. The conformance vector (GOV-014) correctly uses `PackageExpected` shape with `error.code: PKG-010`.

**Discoverability note:** The nested message ("action 'ping': classification 'invalid-class' is not valid") is clear to an external consumer. The `PKG-010` wrapper says WHERE in the system the error was caught (package catalog loading); the nested message says WHAT failed. This pattern is consistent with other PKG-0xx errors. Acceptable.

### 2. Tess Q1 — TV-CONFORM-GOV-004: recast, do not bypass

**Decision:** The harness MUST NOT gain an approval-bypass mechanism. That would create a test path where approval is not enforced — a safety invariant violation.

**Required action:** Recast TV-CONFORM-GOV-004 to expect the specific error that occurs when approval is required but no interactive channel is available. The harness runs with `TTYOutput: false` → `NoOpApprovalGate` → auto-approves → vector would actually PASS. Wait — re-reading: the harness uses NoOpApprovalGate which auto-approves. So the tool WOULD execute successfully. The SKIP reason says "the CLI harness has no mechanism to supply approval" but NoOp auto-approves...

**Correction:** If the harness auto-approves, the vector should just run and PASS (the runbook completes). Tess should investigate whether the vector actually fails or was pre-emptively skipped. If it passes under NoOp, un-skip it. If the ProfileEvaluator denies it (because the test has a profile that doesn't allow the action), adjust the fixture. **Owner: Tess.** Un-skip or recast by end of Phase 1.

### 3. Tess Q3 — CONFORM-CROSSCUT category: DEFER

Current category is fine. Revisit when governance vectors exceed 20. No action now.

### 4. Edith OQ-1 — Profile ID format: DECIDED

`[a-z0-9][a-z0-9-]*`, max 64 characters. Required for shell quoting safety, tab-completion, and filename-stem use (profiles stored as `<id>.yaml`). Leading digit allowed. No uppercase, no spaces, no special characters beyond hyphen. Edith patches §17.1.

### 5. Edith OQ-2 — Per-tool endpoint + auth inheritance: DECIDED

Per-tool overrides inherit profile-level auth by default. If a per-tool override needs distinct auth, add an `auth:` block inside `tools.<tool-id>:` in the profile. Phase 1 implements endpoint-only overrides with inherited auth. Per-tool auth override is Phase 3 scope. Edith patches §17.4 with this note.

### 6. Edith OQ-3 — `gert plan --show-profiles` zero matches: DECIDED

Exit 0 with empty output + a single-line stderr message: "no profile in <search-paths> provides a complete binding for all toolRefs in <runbook>." Empty output is machine-parseable (no false positives for CI scripts). Non-zero exit is wrong here because "zero matches" is a valid, informative answer — not a failure. Edith patches §17.6.

---

## Systemic Process Recommendation

Two instances this session of "checks wired to nothing from the CLI entry point":
1. `AllowedEnvironments` / `RequiresCapabilities` were declared but unenforced (the ghost fields that started this project).
2. David's Tier 0 preflight was implemented in the planner but unreachable from `gert run` until d7b77c1 wired `Profile` into the planner config.

**Recommended convention (adopt for Phase 2+):** Every enforcement function MUST have a companion integration test that exercises it via the CLI entry point (`cmd/gert/run.go` or the equivalent wire path). Unit tests against the planner or evaluator prove correctness; an integration test proves reachability. Name them `TestCLI_<Feature>_Reachable`. If the function is unreachable, the test fails immediately — not after months of silent dormancy.

This is the gap class the whole project exists to eliminate. It recurred inside the project itself. A convention is cheaper than catching it in review.

---

## Summary

All negotiated contract points are satisfied. The implementation is structurally sound and matches the six-round negotiated design. Two conditions required before external reporting (one behavioral test for the coercion invariant, one `gert plan` output format requirement). Six ratification decisions issued. One process recommendation for the team.

---

*Barbara — 2026-08-17*

---

# David — Slice 5: Tier 0 Static Preflight

**Date:** 2026-08-17  
**Commit:** b982804  
**Status:** Complete

## What was implemented

Three Tier 0 (static, offline, no I/O) preflight checks that run at plan time before any step executes.

### Diagnostic codes and where each check runs

| Code | Check | Location |
|------|-------|----------|
| PLAN-010 | AllowedEnvironments vs profile context | `resolveTool()` in `internal/planner/planner.go` → `checkToolEnvironmentPreflight()` in `internal/planner/preflight.go` |
| PLAN-011 | Attendance declared attended but context is structurally unattended | `Plan()` in `internal/planner/planner.go` → `checkAttendancePreflight()` in `internal/planner/preflight.go` |
| PLAN-012 | Test-context transport rule violation | `resolveTool()` → `checkToolEnvironmentPreflight()` |

### Error sentinel chain

All three checks use the same PlanError wrapping pattern:
```
PlanError{Code: errkit.Wrap("PLAN-0NN", msg, plannerPkg.ErrXxx)}
```
- `errors.Is(err, errkit.ErrPLAN0NN)` → true (code match via errkit.Error.Is)
- `errors.Is(err, plannerPkg.ErrContextMismatch)` etc → true (via PlanError→errkit→cause unwrap chain)

### Attendance check: what it can and cannot detect

The attendance check compares declared state only:
- `attendance: attended` + `context: ci` → PLAN-011 (ci is structurally unattended)
- `attendance: attended` + `context: headless-server` → PLAN-011

**Cannot detect** whether a real TTY is connected. `cmd/gert/run.go` hardcodes `TTYOutput: true`. No `isatty()` seam exists in the codebase. Real terminal detection is a Tier 1 concern. This limitation is documented in the error message text.

### Test-context transport: wording discipline

The mcp subprocess opt-in (`allow_subprocess_in_test: true`) is described as an **auditable author assertion** in all messages, comments, and docs. It is NOT described as hermetic, sandboxed, or isolated. `internal/tool/process.go StartProcess()` is a bare `exec.Command` with `mergeEnv()` appending `os.Environ()` — no namespaces, no seccomp, no network jail, no cwd jail.

### Migration safety

`AllowedEnvironments` values that are not canonical `ProfileContext` strings are silently ignored by `filterCanonicalContexts()`. This handles the ~25 tools that carry `allowed-environments: ["real"]` (a RunMode discriminator being migrated to `AllowedModes` by Tess). Without this guard, those tools would incorrectly fail the AllowedEnvironments check for any profile context. Test locked: `TestPreflight_AllowedEnvironments_OnlyRunModeLegacyValues`.

### AllowedModes non-interference

The preflight check never inspects `AllowedModes`. Test locked: `TestPreflight_AllowedModes_DoesNotInterfereWithAllowedEnvironments`.

## What the `gert plan` slice must surface

The `gert plan --profile <profile> <runbook>` command (not implemented in this slice) should surface the PLAN-010 output as a binding table row:

```
  toolRef    transport  auth  status
  ─────────────────────────────────────────────────────────────────────
  icm        mcp-http   ...   ✗ [PLAN-010] not configured for context "ci"
                               allowed-environments: [cli-operator, test]
```

The error.Detail field contains the full actionable message. The step ID is in PlanError.StepID.

## What I could NOT soundly detect

- **Real TTY presence.** Attendance check is declared-state only. Adding a real `isatty()` seam requires wiring through `adapter.BuildEngineConfig`'s `TTYOutput` field and exposing it as a context value the planner can read. Tracked as Tier 1.
- **Whether the operator is actually at the terminal for `vscode-operator` or `cli-operator`.** A process could declare `attendance: attended` and `context: cli-operator` while running headlessly. The check cannot detect this.

## Note for Ken (cmd/gert/run.go)

`cmd/gert/run.go` currently sets `plan.Metadata.Profile = runtimeProfile` after `plannerImpl.Plan()` returns. The planner now also sets it from `Config.Profile` inside `Plan()`, so both paths produce consistent state. However, the `plannerImpl` is currently constructed WITHOUT passing `runtimeProfile` into `plannerPkg.Config.Profile`. This means the preflight checks do not fire from the CLI path today — they fire only when tests construct the planner with `Config.Profile` set.

**Required change in run.go** (Ken's territory — noting here):
```go
plannerImpl := internalplanner.New(plannerpkg.Config{
    Loader:       &fileRunbookLoader{parser: parserImpl},
    Tools:        registry,
    ExpandPolicy: expand.Policy{Default: expandDefault},
    Profile:      runtimeProfile,   // ADD THIS
})
```

Without this, `gert run --profile ci runbooks/x.yaml` does NOT perform Tier 0 preflight. The checks exist and are tested; the CLI wiring is the missing piece.

## Pre-existing bug fixed

`pkg/errkit/errors.go` referenced `ErrPKGW003` in the sentinels map but the var declaration was removed (replaced by `ErrPKGW004`) by a different in-flight change. This caused a compile error. Restored `ErrPKGW003 = &Error{code: "PKG-W003", class: "PKG-W"}` to fix it (this is the "package root resolves outside the workspace" sentinel — PKG-W004 is a new, different code for action classification).

---

# Decision: Classification Validation — Error Class and Conformance Vector

**Author:** Don (Backend Dev)  
**Date:** 2026-08-17  
**Status:** Shipped — commit 7c7402d  
**Requires ratification:** Barbara (error class choice; see §Error Class Decision below)

---

## Context

`ToolAction.Classification *string` was added in Slice 1 (commit a2e7db0). The Slice 1 report claimed the field shipped "with `ParseToolFile` validation." That claim was imprecise — the validation (`validateActionClassifications` in `internal/tool/scan.go`) exists and was correct, but the report did not state the file/function, only the package boundary. Tess was blocked writing a conformance vector because she could not find a matching `error_class` value.

---

## What validation already exists (verified this session)

File: `internal/tool/scan.go`, function `validateActionClassifications`, called from `ParseToolFile`.

Valid values: `"read-only"`, `"mutating"`, `"destructive"`, `"unspecified"`.

- Absent (nil pointer) → accepted. Nil means unspecified.
- Any non-nil pointer whose value is not in `validClassifications` → rejected with: `%s: action %q: classification %q is not valid (allowed: read-only, mutating, destructive, unspecified)`
- This includes explicit empty string `""` — the pointer exists precisely so nil (absent) is distinguishable from an authored empty value, and an authored empty value is a mistake.

No code changes to validation were needed — it was already correct.

---

## Error Class Decision

**Chosen class:** `PKG` / **Chosen code:** `PKG-010` (existing)

**Reasoning:**

When `ParseToolFile` fails during package catalog loading (which is the harness-exercisable path), `pkgcatalog/catalog.go` wraps the error:
```go
errs = append(errs, errkit.New("PKG-010", fmt.Sprintf(
    "package %q: export %q references missing/invalid file %s: %v",
    req.Package, exp.ID, exp.Path, err)))
```

So when classification validation fails, the error that reaches `gert run`'s stderr is:
```
PKG-010: package "acme.gov-test": export "probe" references missing/invalid file tools/probe.tool.yaml: tools/probe.tool.yaml: action "ping": classification "invalid-class" is not valid (allowed: read-only, mutating, destructive, unspecified)
```

`PKG-010` is already registered in `errkit` with class `PKG`. `PackageExpected.error.code` in `vector.schema.json` already matches `^PKG-\d{3}$`. No new errkit class, no schema changes needed.

**Rejected alternative: new `TOOL-PARSE` class.** This would require adding a new class to `errkit.ClassForCode`, `errkit.Classes()`, `errkit.sentinels`, and modifying `vector.schema.json`'s `ErrorExpected.error_class` enum AND `error_code` pattern. The effort is unjustified when `PKG-010` already genuinely covers "package tool export references a file that fails any validation check" — classification validation is such a check.

**Orthogonality invariants GOV-009/GOV-010 are not affected.** Those vectors exercise valid combinations (requires-approval:false + destructive, requires-approval:false + mutating) that `validateActionClassifications` correctly accepts. They continue to pass.

---

## Conformance vector: TV-CONFORM-GOV-014

**ID:** `TV-CONFORM-GOV-014`  
**Category:** `CONFORM-CROSSCUT`  
**Shape:** `PackageExpected` with `error: { code: PKG-010 }`

```yaml
- id: TV-CONFORM-GOV-014
  category: CONFORM-CROSSCUT
  description: 'Invalid classification value in a package-exported tool MUST be rejected at catalog-load time with PKG-010.'
  # ... (fixture: probe.tool.yaml with classification: "invalid-class")
  expected:
    error:
      code: PKG-010
```

**Why PackageExpected, not ErrorExpected:** `ErrorExpected` is for GXL/GIS/GCP/ENUM error classes. Classification errors surface as `PKG-010`, which uses `PackageExpected`. This matches the pattern already established by other GOV vectors that assert on catalog/plan failures.

---

## Unit tests added

File: `internal/tool/validate_classification_test.go` — 7 tests:

1. All 4 valid values accepted
2. Absent (nil) valid
3. Unknown value rejected
4. Explicit empty string rejected  
5. GOV-009 invariant: requires-approval:false + destructive valid
6. GOV-010 invariant: requires-approval:false + mutating valid

---

## Files changed in commit 7c7402d

- `internal/tool/validate_classification_test.go` (new)
- `internal/conformance/enumdata/tv-enum.yaml` (GOV-014 added)
- `internal/conformance/enum_conformance_test.go` (count 71→72)

---

## Action required from Barbara

Ratify the `PKG-010` error class choice for invalid-classification-value vectors. If a new `TOOL-PARSE` class is preferred instead, the change requires:
1. `pkg/errkit/errors.go`: add to `ClassForCode`, `Classes()`, sentinels
2. `internal/conformance/enumdata/vector.schema.json`: add to `ErrorExpected.error_class` enum and `error_code` pattern
3. Update GOV-014 to use `ErrorExpected` with `error_class: TOOL-PARSE` + new code

The current `PKG-010` path requires no schema changes and is functionally equivalent for harness verification.

---

# Decision: Slice 3 — Runtime Profile Schema, Loader, and --profile Flag

**Author:** Don (Backend Dev)  
**Date:** 2026-08-17  
**Status:** Shipped — commit 773f332  
**Slice:** 3 of Phase 1 (foundation slice; blocks Ken's ProfileApprovalGate and David's preflight checks)

---

## What was built

### New file: `pkg/schema/profile.go`

Package: `github.com/ormasoftchile/gert/pkg/schema`

**Key exported types and constants:**

```go
// Top-level type
type RuntimeProfile struct {
    APIVersion string
    ID         string
    Context    ProfileContext
    Attendance ProfileAttendance
    Approval   ProfileApproval
    Transport  ProfileTransport
    Tools      map[string]*ProfileToolOverride
}

// Context — closed enum, exactly 5 values
type ProfileContext string
const (
    ProfileContextCLIOperator    ProfileContext = "cli-operator"
    ProfileContextVSCodeOperator ProfileContext = "vscode-operator"
    ProfileContextCI             ProfileContext = "ci"
    ProfileContextHeadlessServer ProfileContext = "headless-server"
    ProfileContextTest           ProfileContext = "test"
)

// Attendance — top-level, orthogonal to context
type ProfileAttendance string
const (
    ProfileAttendanceAttended   ProfileAttendance = "attended"
    ProfileAttendanceUnattended ProfileAttendance = "unattended"
)

// LegacyUnspecifiedPolicy
type ProfileLegacyUnspecifiedPolicy string
const (
    ProfileLegacyPolicyAllow  ProfileLegacyUnspecifiedPolicy = "allow"
    ProfileLegacyPolicyPrompt ProfileLegacyUnspecifiedPolicy = "prompt"
)

// Approval block
type ProfileApproval struct {
    Scope                   ProfileApprovalScope
    LegacyUnspecifiedPolicy ProfileLegacyUnspecifiedPolicy
}
type ProfileApprovalScope struct {
    AllowRead        bool
    AllowMutating    bool
    AllowDestructive bool
}

// Transport block
type ProfileTransport struct {
    AllowSubprocessInTest bool
}

// Per-tool override
type ProfileToolOverride struct {
    Endpoint string  // ok to set
    Mode     string  // PROF-001: any non-empty value is rejected at validation
}

// Loader functions
func ParseProfileFile(path string) (*RuntimeProfile, error)
func ParseProfileBytes(data []byte) (*RuntimeProfile, error)

// API version constant
const RuntimeProfileAPIVersion = "runtime-profile/v1"
```

### Modified: `pkg/engine/run.go`

Added to `PlanMetadata`:
```go
Profile *schema.RuntimeProfile
```
This is how Ken's `ProfileApprovalGate` and David's preflight checks receive the profile. Read from `plan.Metadata.Profile`. **Must consume `plan.Tools` (post-catalog, post-package-map), never re-resolve toolRefs independently.**

### Modified: `cmd/gert/run.go`

- New flag: `--profile <path>` (takes a file path to a runtime-profile/v1 YAML)
- Loads and validates the profile before `eng.Start`
- Stores result in `plan.Metadata.Profile`
- Added `"--profile"` to `splitRunArgs` argument-consuming list

---

## Composition rule (contractual, promised in writing)

- `--package-map` wins at **YAML SELECTION** — it decides which tool definition/package binds. Do not change this machinery.
- `--profile` parameterizes **EXECUTION AFTER SELECTION** — context, attendance, approval scope, per-tool params.
- They **compose**, they do not compete. A profile must never re-resolve or override which package was selected.
- Any profile-aware resolution must **consume `plan.Tools`** (post-catalog, post-package-map), not re-resolve toolRefs independently. A mismatch is an error, never a silent override.

---

## Validation constraints enforced at parse time

| Rule | Behavior |
|------|----------|
| `apiVersion` must be `"runtime-profile/v1"` | Hard error |
| `context` must be one of the 5 canonical values | Hard error with list |
| `attendance` must be `attended` or `unattended` | Hard error |
| `legacy_unspecified_policy` absent | Defaults to `"prompt"` |
| `legacy_unspecified_policy` unknown value | Hard error |
| Any tool override sets `mode:` (PROF-001) | Hard error — transport-mode rewriting forbidden |
| Profiles are FLAT — no inheritance, no extends | Structural — no such field exists |

---

## What is explicitly NOT done in this slice

- No `ProfileApprovalGate` or attendance-driven gate selection → Ken's slice
- No Tier 0 preflight checks (attendance-mismatch, test-context-non-native-binding, AllowedEnvironments) → David's slice
- No `gert plan` command → later slice
- No conformance fixture migration → Tess
- No changes to `internal/adapter/wire.go`, `pkg/run/run.go`, `internal/engine/engine.go`, `internal/governance/*`

---

## Test coverage

File: `pkg/schema/profile_test.go` — 10 tests, all passing:

1. `TestParseProfile_ValidRoundTrip` — full round-trip
2. `TestParseProfile_AllFiveContextsAccepted` — 5 subtests, one per context
3. `TestParseProfile_UnknownContextRejected`
4. `TestParseProfile_UnknownAttendanceRejected`
5. `TestParseProfile_AttendanceOrthogonalToContext` — vscode-operator+unattended valid, not coerced
6. `TestParseProfile_FlatProfileNoInheritance` — no inherited/merged fields
7. `TestParseProfile_TransportModeRewriteRejected` — PROF-001
8. `TestParseProfile_UnknownLegacyPolicyRejected`
9. `TestParseProfile_LegacyPolicyDefaultsToPrompt`
10. `TestParseProfile_WrongAPIVersionRejected`

`go build ./...` exit 0. `go test ./...` exit 0, zero regressions.

---

# Don — Slice 6: `gert plan` Static Analysis Command

**Date:** 2026-08-17  
**Author:** Don (Backend Dev)  
**Status:** Shipped — commit bd2ab86  
**Ratification needed:** Barbara (command surface + exit codes), counterparty status report

---

## Summary

`gert plan` is the last Phase 1 deliverable. It provides static, side-effect-free analysis of a runbook against a runtime profile — answering "is this runbook configured to run in this context?" before anyone runs it.

**Command surface:**

```
gert plan [--profile <path>] [--package-map <path>] [--output text|json] <runbook-path>
```

All flags are optional. The command **never executes**, **never authenticates**, **never touches the network**.

---

## Files Changed

| File | Change |
|------|--------|
| `cmd/gert/plan.go` | New — full implementation |
| `cmd/gert/plan_test.go` | New — 12 integration + unit tests |
| `cmd/gert/main.go` | Modified — `case "plan"` dispatch, usage line |
| `cmd/gert/packagemap.go` | Modified — `schemaToolDefFromRuntime` bug fix (see below) |

---

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Plan clean; Tier 0 preflight passed |
| `2` (exitValidation) | Tier 0 violation (PLAN-010/011/012), parse error, or invalid flags |
| `3` (exitRuntime) | OS-level error (e.g. can't read file) |

**CI gating use case:** a CI job can run `gert plan --profile ci-unattended.yaml runbook.yaml` and gate on exit code 0. A `denied` action outcome does NOT cause a non-zero exit — it is informational. Only Tier 0 preflight violations cause non-zero exit at plan time.

---

## Output Shape

### Text (default)

```
Profile:    ci-unattended
  context:    ci
  attendance: unattended

Tier 0 preflight: PASS

Tool bindings:
  NAME            PACKAGE              TRANSPORT     STATUS
  ─────────────────────────────────────────────────────────
  icm             acme.ops@1.2.0       mcp           ok

Action approval consequences:
  TOOL    ACTION    CLASSIFICATION    OUTCOME
  ──────────────────────────────────────────────────────────────
  icm     create    mutating          denied (mutating action denied in unattended context ...)
  icm     get       read-only         allowed
```

### JSON (`--output json`)

```json
{
  "runbook": "/path/to/runbook.yaml",
  "profile": { "id": "ci-unattended", "context": "ci", "attendance": "unattended" },
  "preflight": { "pass": true },
  "tools": [{ "name": "icm", "package": "acme.ops@1.2.0", "transport": "mcp", "status": "ok" }],
  "actions": [
    { "tool": "icm", "action": "create", "classification": "mutating", "outcome": "denied", "deny_reason": "..." },
    { "tool": "icm", "action": "get",    "classification": "read-only", "outcome": "allowed" }
  ]
}
```

On Tier 0 failure, the JSON output is an error envelope:
```json
{ "preflight": { "pass": false, "code": "PLAN-011", "detail": "..." } }
```

---

## Approval Consequence Matrix (static)

Mirrors `internal/governance/profile_evaluator.go` exactly. Pure function, no evaluator import.

| classification | test context | attended | unattended |
|---|---|---|---|
| `read-only` | `allowed` | `allowed` if `scope.AllowRead`; else `denied` | `allowed` if `scope.AllowRead`; else `denied` |
| `mutating` | `allowed` | `approval-required` | `allowed` if `scope.AllowMutating`; else `denied` |
| `destructive` | `allowed` | `approval-required` | `allowed` if `scope.AllowDestructive`; else `denied` |
| `unspecified` | `allowed` | `approval-required` | `allowed` if `legacy_unspecified_policy=allow`; else `denied` |

Special cases:
- `requires-approval: false` → `legacy-opt-out (gate suppressed)` — does NOT derive classification: read-only (GOV-009/010 invariant).
- `requires-approval: true` → `approval-required (explicit)` — always, regardless of scope.
- No profile → `no-profile`.

---

## Composition Rules (contractual, unchanged)

- `--package-map` wins at YAML selection (which tool definition/package binds). `mergePackageBindings()` / `mergeToolPaths()` unchanged.
- `--profile` parameterizes execution after selection: context, attendance, per-tool approval analysis.
- They compose, never compete. Profile must never re-resolve or override which package was selected.
- `plan.Tools` (post-catalog, post-package-map) is the source of truth. No independent toolRef re-resolution.

---

## Bug Fix in `schemaToolDefFromRuntime` (packagemap.go)

**Problem:** The `schemaToolDefFromRuntime` function converts a runtime `toolpkg.ToolDef` to a `*schema.ToolDef` for the planner schema registry. It was not copying `Classification` from `ToolAction` or `Governance` from `ToolDef`. Without this, every action in `plan.Tools` had `Classification == nil` (appears as "unspecified") and `Governance == nil` (no requires-approval opt-out recognized).

**Why not caught earlier:** No prior code path read `Classification` from `plan.Tools` at the schema level. Ken's approval gate reads from the runtime registry's own `ToolAction`, not from `plan.Tools`. `gert plan` is the first consumer that needs the schema-level classification for display.

**Fix:** 
1. `a.SchemaAction().Classification` — `SchemaAction()` was already implemented for substitution planning (pkgsubst.Plan needs the original schema action). Reusing it avoids adding a new `Classification` field to the runtime `ToolAction` type.
2. `def.Governance` — directly carried through to `schema.ToolDef.Governance`.

**Scope note:** This is `cmd/gert/packagemap.go`, which is in my territory (cmd/gert). The fix is tightly coupled to `gert plan`'s needs and does not change any existing behavior — `schemaToolDefFromRuntime` output was previously only consumed by the planner for flow validation, which doesn't read Classification or Governance from `plan.Tools`.

---

## What `gert plan` does NOT do (scope discipline)

- Does NOT execute anything.
- Does NOT implement a ProfileApprovalGate (that is Ken's runtime gate).
- Does NOT output PKG-W warnings (no tool actions run at plan time; PKG-W is a per-execution nudge, not a static analysis warning).
- Does NOT change plan.go to cause a non-zero exit when actions have `denied` outcome — that would be a new policy decision outside Slice 6.
- Does NOT change David's preflight.go semantics or Ken's gate logic.

---

## Counterparty status report items

1. `gert plan` is available as of commit bd2ab86.
2. `gert plan --profile ci-unattended.yaml runbook.yaml` exits 0 if Tier 0 clean; exits 2 if PLAN-010, PLAN-011, or PLAN-012 fires.
3. JSON output via `--output json` supports CI parsing.
4. Approval denial is surfaced as informational (`outcome: "denied"`) in the actions table — CI may parse this to detect pre-run denial without running the runbook.
5. The `--package-map` and `--profile` flags compose correctly in `gert plan` with the same precedence as `gert run`.


---

## Barbara Gate-Review Addition (commit 2968006, 2026-08-17)

### Profile ID constraint (ratified)

ID must match `[a-z0-9][a-z0-9-]*`, max 64 chars. Profiles are stored as `<id>.yaml`; the ID is a filename stem. Empty ID is also rejected. Enforced in `validateProfile()`.

### config/no-binding-for-profile diagnostic

When `gert plan --profile <path> runbook.yaml` fails (Tier 0 violation):

```
config/no-binding-for-profile: PLAN-011: profile "bad" declares attendance: attended but context "ci" cannot provide an interactive channel
  note: this runbook runs correctly in profiles: good
```

Or when no alternative is found:
```
config/no-binding-for-profile: ...
  note: no discovered profile provides a complete binding for this runbook
```

The alternatives are found by scanning the **directory containing the selected profile file** (no additional flag needed). The search is best-effort; errors are ignored so a missing sibling doesn't mask the primary error.

### --show-profiles `<dirs>` flag

Comma-separated list of directories. Scans for `*.yaml` profile files, tests each against the runbook, reports compatibility.

**Exit codes for --show-profiles:**
- `0` always (Barbara's ruling: zero matches is informative, not a failure)
- Zero matches: empty stdout, single-line stderr: `no profile in <dirs> provides a complete binding for all toolRefs in <runbook>`
- Non-zero matches: compatibility table (text) or structured JSON

**JSON shape (`--output json`):**
```json
{
  "runbook": "/path/to/runbook.yaml",
  "search_dirs": ["./profiles"],
  "profiles": [
    { "id": "ci-prod", "context": "ci", "attendance": "unattended", "compatible": true },
    { "id": "bad",     "context": "ci", "attendance": "attended",   "compatible": false, "error": "PLAN-011: ..." }
  ]
}
```

### Updated command surface

```
gert plan [--profile <path>] [--package-map <path>] [--show-profiles <dirs>] [--output text|json] <runbook-path>
```

### Exit code semantics (final, for counterparty status report)

| Exit code | Meaning |
|-----------|---------|
| `0` | Plan clean, or `--show-profiles` (any number of matches) |
| `2` | Tier 0 violation (`--profile` specified), parse/catalog error |
| `3` | OS-level runtime error |

**Critical distinction**: `--show-profiles` zero matches is exit 0. Tier 0 violations (when `--profile` is given) are exit 2. These must NOT be conflated — the CI use case depends on this.

---

# Edith — Post-Work Decision Note: Runtime Profiles Spec

**Date:** 2026-08-17T07:49:00-07:00
**Author:** Edith (Spec Editor)
**Deliverable:** `design/gert/sections/17-runtime-profiles.tex`

---

## Placement Choice

**LaTeX chapter file in `design/gert/sections/`.**

The repo uses a LaTeX design document (`design/gert/main.tex`) with numbered
chapter files under `design/gert/sections/`. The new section follows this
convention as `17-runtime-profiles.tex` and is included via
`\input{sections/17-runtime-profiles}` appended to `main.tex`. No new format was
introduced.

---

## Source of Truth Consumed

- `.squad/decisions.md` — Rev 4 Final Acknowledgment (2026-08-17), Barbara
  arch-eval (2026-08-16), David integration-critique (2026-08-16), Don
  ground-truth (2026-08-16) and Slice-1 report (2026-08-17), Ken Slice-2 report
  (2026-08-17).

All twelve normative topics in the task were traced to decisions recorded in
`decisions.md`. No semantic was invented.

---

## OQ-1 — Profile ID format → §17.1  ✅ RESOLVED

**Barbara ruling (2026-08-17):** `[a-z0-9][a-z0-9-]*`, max 64 characters.
Rationale: shell-quoting safety, tab-completion, filesystem stem (`<id>.yaml`).
Leading digit permitted. No uppercase, no spaces, no underscores.
Encoded in §17.1 field-by-field rules paragraph for `id`. Marked [IMPL].

## OQ-2 — Per-tool endpoint override + auth inheritance → §17.1, §17.4  ✅ RESOLVED

**Barbara ruling (2026-08-17):** Per-tool overrides inherit profile-level auth
by default. Per-tool `auth:` override block is Phase 3 scope. Phase 1 implements
endpoint-only overrides; the loader MUST reject a per-tool `auth:` block in
Phase 1. Encoded in §17.1 (tools field paragraph) and §17.11 (implementation
status table with Phase 3 row). Marked [IMPL — endpoint + inheritance; per-tool
auth is SPEC Phase 3].

## OQ-3 — `gert plan --show-profiles` with zero matches → §17.6  ✅ RESOLVED

**Barbara ruling (2026-08-17):** Exit 0 with empty stdout, single-line stderr:
`no profile in <search-paths> provides a complete binding for all toolRefs in <runbook>`.
Rationale: zero matches is informative, not a failure; non-zero exit would break
CI scripts. Encoded in §17.6 (`gert plan` subsection) with rationale.

## config/no-binding-for-profile message format → §17.6 error taxonomy

Barbara flagged this as the highest-value operator output in the design.
Precise message format (including mandatory `note:` line showing which profiles
DO work) encoded in §17.6 (new `\subsubsection`). The note line is computable
at Tier 0 and is the single line that turns a refusal into a next action.

## Implementation status updates

Committed landing data from Barbara:
- Profile schema/loader/`--profile` (773f332): [IMPL]
- ProfileApprovalGate + declared attendance (d7b77c1, b420dda): [IMPL]
- Tier 0 PLAN-011/012 (b982804): [IMPL]
All reflected in §17.11 status table and in attendance tcolorbox (§17.3) and
governance-composition tcolorbox (§17.9).

---

*All open questions resolved. No outstanding items.*

---

## Contradictions Found in Recorded Decisions

**None.** The six negotiation rounds in decisions.md are internally consistent.
The one near-contradiction (Rev 3 claim that `requires-approval: false` ≡
`classification: read-only`) is explicitly withdrawn in Round 1 of Rev 4; the
record is clean and the correction is the normative baseline. No papering-over
was required.

---

## Notes on [IMPL] vs. [SPEC] Tagging

Every normative claim in the spec is tagged [IMPL] or [SPEC]. The following
items are particularly important to keep honest:

- `AllowedEnvironments` enforcement: schema field exists [IMPL], Tier 0 enforcement
  via PLAN-011 is [IMPL] as of b982804; remaining preflight checks (auth provider,
  env-var interpolation, attendance mismatch) are still [SPEC].
- Profile-driven attendance: now [IMPL] via ProfileApprovalGate (d7b77c1, b420dda).
- Subprocess non-sandboxing: confirmed [IMPL]. The sentence "Gert does not enforce
  isolation" is deliberate and non-negotiable in any future revision of this section.
- Per-tool auth override: Phase 3 scope, loader MUST reject in Phase 1. [SPEC]

---

## Commits

- Initial spec: `0c96e21` (2026-08-17)
- OQ patch + impl-status update: staged for `0c96e21` follow-on commit

---

# Decision Record: ken-slice4-profile-approval-gate

**Date:** 2026-08-17
**Author:** Ken (Backend Dev)
**Commit:** ff429d4
**Slice:** 4 — ProfileApprovalGate + declared attendance

---

## What Shipped

### New types and files

**`internal/governance/profile_evaluator.go`**
- `profileEvaluator` struct implementing `governance.PolicyEvaluator`
- Constructor: `NewProfileEvaluator(base PolicyEvaluator, profile *schema.RuntimeProfile) PolicyEvaluator`
- Nil profile → fully transparent (no behavior change)
- Wraps any base evaluator; does not depend on concrete evaluator type

**`pkg/governance/evaluator.go` additions to `StepInfo`:**
- `ToolApprovalTriState *bool` — raw tri-state from `ToolGovernance.RequiresApproval`: nil=unspecified, &false=legacy opt-out, &true=explicit requirement
- `ToolClassification *string` — action classification: "read-only", "mutating", "destructive", or nil

**`internal/adapter/options.go` additions to `WireOptions`:**
- `Attended *bool` — approval attendance override. nil = fall back to TTYOutput.

**`cmd/gert/run.go`:**
- `attendedFromProfile(*schema.RuntimeProfile) *bool` helper
- Passes `Attended: attendedFromProfile(runtimeProfile)` to `WireOptions`

---

## Classification Matrix Implemented

| classification | test context | attended | unattended |
|---|---|---|---|
| read-only | allow | allow if scope.AllowRead; else deny | allow if scope.AllowRead; else deny |
| mutating | allow | gate | allow if scope.AllowMutating; else deny |
| destructive | allow | gate | allow if scope.AllowDestructive; else deny |
| unspecified | allow | GATE FIRES | allow if legacy_allow; else deny |

**Legacy opt-out (`ToolApprovalTriState == &false`):**
- Takes priority over the matrix (checked before classification lookup)
- Suppresses the approval gate
- Does NOT assign classification: read-only
- Does NOT relax retry, idempotency, or late-result handling

---

## Resolution Order (Attendance for Approval Gate)

1. `WireOptions.Attended != nil` → profile-declared attendance wins
   - `ProfileAttendanceAttended` → `&true` → TerminalApprovalGate
   - `ProfileAttendanceUnattended` → `&false` → NoOpApprovalGate
2. `Attended == nil` → fall back to `TTYOutput` (no regression)

**Important:** This only affects `buildApprovalGate()`. The `buildInputProvider()` and `buildPromptProvider()` functions continue to use `TTYOutput` unchanged.

---

## Chokepoint

The `ProfileEvaluator` wraps the per-run evaluator that lives in `runHandle.governanceEvaluator`. This evaluator is already the single chokepoint for ALL step types (from Slice 2). No transport-specific changes were needed.

The wrapping happens in `engine.Start()` and `engine.Resume()`:
```go
h.governanceEvaluator = internalGov.BuildEvaluator(e.cfg.ApprovalGate, plan.GovernanceSource)
h.governanceEvaluator = internalGov.NewProfileEvaluator(h.governanceEvaluator, plan.Metadata.Profile)
```

---

## PKG-W004 Warning

Emitted as `governance/unclassified_action` trace events (one per tool action with nil Classification) at run/started time. This is a nudge toward explicit classification labeling. Non-fatal. Does not block execution.

Note: `pkg/errkit/errors.go` already contained `ErrPKGW004` from pre-existing in-flight work (commit b982804) — no change needed there.

---

## What the Later Slices Must Know

**`gert plan` slice:**
- The PKG-W004 warning is emitted at run time, not plan time. If plan-time reporting is desired, the planner's `Plan()` return signature would need to return `(plan, warnings, error)`.
- `plan.Metadata.Profile` is the profile available at run time; the planner receives it via ParseRunbook → BuildEngineConfig → plan.Metadata.Profile.

**Tier 0 checks (wiring gap — CLOSED, commit d7b77c1):**
- David's Slice 5 Tier 0 preflight checks (PLAN-010/011/012) were implemented and passing in unit tests, but `cmd/gert/run.go` constructed the planner without `Profile: runtimeProfile`, so all three checks were dead in production.
- Fix: one line added to the `plannerpkg.Config{}` construction in `run.go`.
- Regression guard: `cmd/gert/profile_tier0_integration_test.go` (6 tests) goes through `runRun()` — the exact CLI code path. Unit tests bypassing `run.go` cannot catch this class of wiring gap.
- **If you add new `pkg/planner.Config` fields that should be populated from CLI flags, always add a `cmd/gert/` end-to-end test that verifies the field actually reaches the planner.**

**Tier 0 slice (David):**
- `ToolClassification *string` is now populated in `governance.StepInfo` at the engine pre-flight phase. David's Tier 0 preflight checks can read this from StepInfo if needed.
- `ToolApprovalTriState *bool` in StepInfo is a raw passthrough from the schema — it is NOT the same as the base evaluator's `ToolRequiresApproval bool`. Don't conflate them.

**Schema consumers:**
- `ToolGovernance.RequiresApproval *bool` (Don's Slice 1) is the tri-state in the authored schema.
- `StepInfo.ToolApprovalTriState *bool` is the same value passed through to the ProfileEvaluator.
- `StepInfo.ToolRequiresApproval bool` is the dereferenced value used by the base evaluator for monotone-OR. Both fields coexist in StepInfo.

---

## Surprises / Gotchas

1. **`pkg/trace/event.go` was pre-modified**: Adding a new `EventKind` constant there and then staging with `git add` would have swept in ~143 lines of unrelated in-flight work. Used inline `trace.EventKind("governance/unclassified_action")` in engine.go to keep the commit clean.

2. **`ErrPKGW004` already existed**: The pre-existing in-flight work (commit b982804) had already added `ErrPKGW004` to `pkg/errkit/errors.go`. No change needed.

3. **Git stash side-effect**: Running `git stash`/`git stash pop` to verify pre-existing failures caused some files to be staged that weren't mine. Required careful per-file `git add` and verification with `git status --short`.

4. **`ProfileContext` is orthogonal to attendance**: "cli-operator" and "vscode-operator" contexts do NOT imply attendance. The profile must explicitly declare attendance. Confirmed this matches `schema.go` comments.

5. **Unattended + `&false` opt-out**: The legacy opt-out (`ToolApprovalTriState == &false`) suppresses the gate even in unattended context where unspecified would be denied. This was a deliberate design choice: legacy opt-out is a grandfathering mechanism, not a profile-governed one. Tests cover this.

---

## Barbara's Gate — Condition 1 (commit ed432c7, 2026-08-17)

**Condition:** Tess's GOV-009/GOV-010 conformance vectors run without a profile → NoOpApprovalGate → routing changes inside ProfileEvaluator are invisible. A coercion bug like `if tristate == &false { classification = "read-only" }` would pass both vectors either way.

**Closed with:** `TestApproval_LegacyOptOut_ClassificationNotCoerced_Destructive` and `TestApproval_LegacyOptOut_ClassificationNotCoerced_Mutating` in `internal/engine/profile_approval_test.go`.

**Probe design:** attended profile with `allow_read=false`. If evaluator coerces "destructive"/"mutating" → "read-only", the read-only matrix row with `allow_read=false` denies the step → `Denied=true` → test fails loudly. The test also asserts `info.ToolClassification` is not mutated after `Evaluate()` returns.

**Current structural guarantee:** `ToolClassification` is populated from `action.Classification` at `engine.go:536` and the legacy opt-out path only sets `result.RequiresApproval = false`. These tests exist so they stop passing the day someone "simplifies" the evaluator.

Barbara's condition is satisfied. Both tests pass (`go test ./internal/engine/... -run TestApproval_LegacyOptOut_ClassificationNotCoerced` exits 0).

---

# tess-slice7-fixture-migration.md

**From:** Tess (Conformance Tester)  
**To:** Barbara (Architect)  
**Date:** 2026-08-17  
**Subject:** Slice 7 migration complete + two open questions needing arbitration

---

## What was done

### 1. Fixture migration — COMPLETE

Audited and migrated all `allowed-environments` occurrences in the conformance corpus:

| File | Occurrences | Action |
|------|-------------|--------|
| `internal/conformance/enumdata/tv-enum.yaml` | **25** | Migrated → `allowed-modes: ["real"]` |
| All other repo files (yaml/json/go/md) | 0 | Nothing to do |

Every occurrence was inside a `governance:` block on a package-exported tool definition, value `["real"]` — unambiguously RunMode intent. No ambiguous cases found. The counterparty count of "~25" is confirmed as **exactly 25**.

`pkg/schema/tool.go` still declares `AllowedEnvironments []string` (yaml: `allowed-environments`) — the field survives for its new deployment-context semantics. No runtime reads of either field today; enforcement is a later slice.

### 2. JSON schema — NO CHANGES NEEDED

`schemas/runbook.schema.json` does not declare `allowed-environments` or `allowed-modes`. There is no tool definition JSON schema file. Nothing to update. When a tool JSON schema is added, both fields should appear there.

### 3. Governance conformance vectors — 13 ADDED (TV-CONFORM-GOV-001..013)

New vectors use category `CONFORM-CROSSCUT` (valid in the existing schema enum). Total corpus: 58 → 71 vectors.

| ID | Subject | Result |
|----|---------|--------|
| TV-CONFORM-GOV-001 | governance absent → RequiresApproval nil | PASS |
| TV-CONFORM-GOV-002 | governance present, no requires-approval → nil NOT false ← the catch | PASS |
| TV-CONFORM-GOV-003 | requires-approval: false → explicit &false | PASS |
| TV-CONFORM-GOV-004 | requires-approval: true → schema valid | **SKIP** (see Q1) |
| TV-CONFORM-GOV-005 | classification absent → nil | PASS |
| TV-CONFORM-GOV-006 | classification: "read-only" → valid | PASS |
| TV-CONFORM-GOV-007 | classification: "destructive" → valid | PASS |
| TV-CONFORM-GOV-008 | per-action: read-only + destructive in same tool → valid | PASS |
| TV-CONFORM-GOV-009 | **ORTHOGONALITY**: requires-approval: false + destructive → valid | PASS |
| TV-CONFORM-GOV-010 | **ORTHOGONALITY**: requires-approval: false + mutating → valid | PASS |
| TV-CONFORM-GOV-011 | AllowedModes absent → unrestricted | PASS |
| TV-CONFORM-GOV-012 | AllowedModes: ["real"] → single value | PASS |
| TV-CONFORM-GOV-013 | AllowedModes: ["real", "dry-run"] → multi-value | PASS |

**TV-CONFORM-GOV-009 and TV-CONFORM-GOV-010 are the most important vectors.** They pin the contract that was the subject of six rounds of negotiation: `requires-approval: false` NEVER implies, assigns, or coerces `classification` to read-only. Any future code change that derives classification from requires-approval will immediately break these two vectors.

---

## Open Questions for Barbara

### Q1 — requires-approval: true enforcement is LIVE (not "later slice")

TV-CONFORM-GOV-004 is currently SKIPPED with reason: "requires-approval: true is actively enforced by the engine; the CLI harness has no mechanism to supply approval — ticket T-GOVN-APPROVAL-HARNESS."

**Finding:** Contrary to the Slice 7 brief ("neither field is read by any runtime code today"), `requires-approval: true` IS enforced. Running `gert run` on a tool with `requires-approval: true` immediately prompts for an approver identity interactively and exits non-zero if stdin is not a terminal.

**Question:** Should the harness gain a `--no-require-approval` / `--dry-approve` flag so we can test the schema-acceptance contract (tool parses, schema valid) independently of the approval gate? Or should the vector be recast to expect a specific error code (whichever code the engine emits when running non-interactively without approval)?

### Q2 — Classification validation: no TOOL-PARSE error class exists

The contract requires "an unknown classification value MUST be REJECTED with a validation error." I cannot write the rejection vector because:

1. No `TOOL-PARSE` error class exists in `ErrorExpected.error_class` (the enum is `GXL-PARSE, GXL-TYPE, ... ENUM` — no tool-definition parse error class).
2. Classification validation is not currently enforced in `pkg/schema` (the field is `*string` with no enum check).

**Request:** Please ratify:
- A `TOOL-PARSE` error class (and a corresponding error code like `TOOL-PARSE-001` for invalid-classification-value).
- Or an alternative placement (is classification a plan-time schema error? Should it use an existing error class?).

Until this is ratified, the rejection vector is deferred. I have documented it in the decision note and will add it to the corpus as soon as the error class lands.

### Q3 — Category placement for future governance vectors

Current governance vectors use `CONFORM-CROSSCUT`. As governance coverage grows, a dedicated `GOVN-*` category (or a new prefix like `TOOL-GOV`) would be cleaner. For now `CONFORM-CROSSCUT` works. Raising this for a future triage rather than requiring immediate decision.

---

## Summary

- **25 fixtures migrated** (exact counterparty count confirmed)
- **No other repo files** had `allowed-environments`
- **JSON schema** — no changes needed
- **13 governance vectors added**, all passing or explicitly skipped with named reasons
- **Orthogonality vectors GOV-009, GOV-010** are the critical deliverable — they prevent silent coercion of classification from requires-approval
- **Two open items**: harness approval bypass (Q1) and TOOL-PARSE error class (Q2)

---

## Q1 Update — 2026-08-17 (Barbara ruling implemented)

**Empirical finding:** `cmd/gert/run.go` hardcodes `TTYOutput: true` unconditionally. The
`buildApprovalGate` function maps `TTYOutput: true → TerminalApprovalGate`. The harness subprocess
always runs with stdin=devnull (no TTY), so TerminalApprovalGate prompts on stdout ("Approval
required... Enter approver identity:") then exits 3. The `NoOpApprovalGate` path (TTYOutput:false
→ auto-approves) is never reached by the `gert run` CLI. This disproves the "harness runs with
TTYOutput:false → NoOpApprovalGate → auto-approves" hypothesis.

**Barbara's ruling respected:** No approval-bypass mechanism added to the harness.

**Resolution:** GOV-004 was **recasted** (fixture changed, not skipped). The new fixture uses
`toolRefs:` (forces pkgcatalog.Build to parse the tool YAML with `requires-approval:true`) but has
no tool step in the flow (so TerminalApprovalGate is never invoked). This proves
schema-acceptance at plan/load time, which is the contract the vector was written to pin.

**Result:** TV-CONFORM-GOV-004 → PASS. Removed from enumRuntimeGapSkips. Permanent skip no longer exists.

**Additional finding:** TV-CONFORM-GOV-014 (classification validation, PKG-010) was added to the
corpus by in-flight team commits and is already PASSING. Classification IS enforced at catalog-load
time. Q2 (TOOL-PARSE error class) is partially moot for the rejection vector — PKG-010 is the
correct error code because the enforcement point is ParseToolFile inside the catalog loader.




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

# Decision: Phase 2 Architectural Ruling — VS Code Authenticated MCP Runtime Binding

**Date:** 2026-08-17
**Author:** Barbara (Lead/Architect)
**Status:** ACCEPT WITH MODIFICATIONS

## Ruling

Phase 2 is accepted with three modifications:

1. **Output contract enforcement for non-substituted tools is prerequisite core work.** The consumer's mutation control ("remove the output contract parity check and prove drift is caught") assumes a runtime check that does not exist. This is the fifth instance of the declared-but-unenforced pattern class. Must land before the bridge is load-bearing.

2. **Three extension prerequisites are blocking:** reproducibility remediation, `engines.vscode` raise to ≥1.90, and extension-host test harness (`@vscode/test-electron` or equivalent).

3. **The `serve` all-interfaces/no-auth security issue must be fixed before or with Phase 2**, not after.

## Ownership

- **Core:** `vscode-mcp` transport chain, loopback bridge server, output contract enforcement, request-ID dedup, bridge versioning, credential sweep extension, `serve` security fix.
- **Extension:** Reproducibility remediation, `engines.vscode` raise, test harness, bridge client, tool resolution via `vscode.lm.tools`, result normalization, authorization-unavailable detection, disconnect signaling.
- **Consumer:** Nothing new. Unchanged runbook. Manual acceptance in their environment.

## Executor-Kind Constraint

The `vscode-mcp` binding MUST remain within the existing `"tool"` executor kind. Recommended structural enforcement: extract classification-injection + approval-gate logic into a named function whose signature requires a non-nil `*ToolClassification`, making governance bypass a compile-time error.

## Estimate

10–14 days parallelized after prerequisites (3–4 days serial in extension repo).

## References

- Full evaluation: `gert-core-phase2-evaluation.md`
- Phase 1B completion: `gert-core-phase1b-completion.md`
- Reachability gate: `TestCLI_ReachabilityGate`

---

# Decision: Contract-Identical Bindings Documentation

**Date:** 2026-08-17
**Author:** David (Integration Engineer)
**Status:** Shipped — commit `e6521f6` in `ormasoftchile/gert`

---

## What Was Decided

Authored `docs/contract-identical-bindings.md` as Deliverable 3 for SQL Live-Site Operations.
Cross-linked from `README.md` under a new Documentation section.

---

## Key Decisions Made in This Work

### 1. `docs/` directory uses `git add -f`

The `.gitignore` excludes `docs/` under the comment "# Design docs (local)". However,
`docs/perf/` files are already tracked via force-add. The binding doc follows the same
pattern. Team should decide whether to remove `docs/` from `.gitignore` now that the
directory contains consumer-facing documentation.

### 2. PLAN-013 fires during planning (Tier 0), not at execution

Verified directly in `internal/planner/preflight.go`. The endpoint override host check
is in `checkToolEnvironmentPreflight`, which is called from `resolveTool()` during
`planner.Plan()`. No execution step runs before this check. The document describes
this accurately.

### 3. Rule A is structural, not runtime-asserted

`ProfileToolOverride` has no `Scope` or `AllowedHosts` fields in `pkg/schema/profile.go`.
The constraint "profiles may not override scope or allowed_hosts" is enforced by the
absence of those fields, not by a runtime assertion. This is stronger than a check.

### 4. PROF-001 (transport-mode rewriting) fires at profile parse time

`validateProfile` in `pkg/schema/profile.go` checks for `override.Mode != ""` and
returns an error immediately. This happens when the profile file is loaded, before
any planning or execution. Document accurately describes this.

### 5. No implementation contradictions found

Every behavior described in the document was verified against the real implementation.
The document does not describe intended behavior anywhere; it describes actual behavior.

---

## What Was Not Done

- Deliverable 2 (worked example) is blocked on Don's synthetic fixture. Not attempted.
- No changes to `pkg/contractparity/**` (Tess owns), `internal/tool/**`, or `cmd/gert/**`.

---

# Decision: vscode-mcp Transport Design

**Date:** 2026-08-17
**Author:** David
**Status:** SHIPPED (commits a35c0ae, 3cd262e, 9c39638)

---

## Context

SQL Live-Site Operations needs to execute an unchanged runbook from VS Code using the user's already-authenticated ICM MCP tool. This requires a new `vscode-mcp` transport mode in Gert core that delegates tool invocation to a VS Code extension bridge, without exposing any bearer token to Gert.

Barbara's ruling (phase2-evaluation.md) explicitly accepted this as Phase 2 Item 2. This commit implements the full five-point transport chain plus bridge client.

---

## Decision: Loopback Bridge Client, Bridge Server Deferred

### What ships:
- Full five-point transport chain: schema constant → pkg/tool constant → validation → scan/mapTransport → runtime Invoke arm
- Bridge client (`VSCodeMCPTransport`) implementing the client side of the loopback bridge contract
- Per-session capability secret (crypto/rand, 32 bytes, hex-encoded); never logged
- Context-aware cancellation via `http.NewRequestWithContext` (same pattern as mcp_http.go)
- Versioned, correlated request/response contract (`vscode-mcp-bridge/v1`)
- Hard fail on version mismatch — no negotiation in v1
- Duplicate-request idempotency: structural support (map[string]*pendingRequest) but v1 generates fresh UUIDs per call; callers opt in by reusing IDs
- Extension disconnect, malformed response, capability rejection all handled with clear errors
- Credential sweep extended to cover capability secret
- Reachability probe: `TransportVSCodeMCP` registered as `statusReachable`

### What does NOT ship (explicitly deferred):
- **Bridge server**: The VS Code extension owns the server side. Core defines the wire contract (documented in vscode_mcp_bridge.go) and implements the client. Extension owns: bind 127.0.0.1, listen on port, validate capability_proof, invoke `vscode.lm.invokeTool()`, return typed result.
- **Bridge startup handshake**: In v1, capability secret is generated by Gert and must reach the extension out-of-band (e.g. via environment variable or a startup message). The mechanism for communicating the secret from Gert to the extension is not defined in this transport — that is a bridge server concern.

---

## Security Model

- Loopback bind ONLY: `validateLoopback()` rejects any URL that does not start with `http://127.0.0.1`. No escape hatch.
- Capability secret: 32 cryptographically random bytes (crypto/rand), hex-encoded. Generated once per VSCodeMCPTransport instance (per-session). Sent in every request body as `capability_proof`. Never appears in logs, traces, errors, or results — enforced by TestVSCodeMCP_CapabilitySecretRedaction (five paths) + TestVSCodeMCP_CapabilitySecretSweepDetectsLeak (negative control).
- No bearer token reaches Gert. Authorization stays inside VS Code entirely.
- Unauthenticated localhost requests rejected by bridge via 401/403 → `capability_rejected` error code.

---

## Wire Contract (v1)

**Request (POST to bridge URL, JSON body):**
```json
{
  "version": "vscode-mcp-bridge/v1",
  "request_id": "<uuid-v4>",
  "tool": "<tool-name>",
  "action": "<action-name>",
  "args": { ... },
  "deadline_unix_ms": 1234567890,
  "capability_proof": "<hex-64-chars>"
}
```

**Response (success):**
```json
{
  "version": "vscode-mcp-bridge/v1",
  "request_id": "<same-uuid>",
  "result": { ... }
}
```

**Response (error):**
```json
{
  "version": "vscode-mcp-bridge/v1",
  "request_id": "<same-uuid>",
  "error": { "code": "<stable-code>", "message": "<redacted>" }
}
```

**Error codes:**
- `bridge_disconnected` — extension has disconnected or been closed
- `capability_rejected` — capability secret invalid (401/403 HTTP or this code in body)

**Version mismatch**: hard fail; no negotiation.

---

## Executor-Kind Constraint (from Barbara's ruling)

vscode-mcp is a transport variant, NOT a new executor kind. It remains within `"tool"` executor kind. The engine's `executeStep` classification injection at lines ~527-543 fires for `step.Kind == "tool"` regardless of transport. `requiresIndeterminate` (engine.go:~1280) returns false for `"read-only"` classification — this is automatic and correct for vscode-mcp because classification is per-action on the tool definition, not per-transport.

---

## Bridge URL Configuration

In v1: environment variable `GERT_VSCODE_BRIDGE_URL` (default: `http://127.0.0.1:7779`). No `url:` field in `.tool.yaml` — the bridge is a loopback singleton, not an operator-configured endpoint.

---

## What Remains for the Extension Team

1. Bridge server: bind 127.0.0.1 only, validate capability_proof on every request, invoke `vscode.lm.invokeTool()`, return typed result in the v1 response shape
2. Startup handshake: mechanism for Gert to communicate the per-session capability secret to the extension (stdout handshake recommended; secret is never a user credential)
3. Extension disconnect signaling: send 503 or `bridge_disconnected` error on deactivation/window close
4. Prerequisites P0-P2 from Barbara's ruling (extension reproducibility, engines.vscode minimum, test harness)

---

# Decision: dry-run gate scope and serve loopback default

**Author:** Don  
**Date:** 2026-08-17  
**Status:** Implemented (commits d19f933, a6b1106, 06b0a83)  
**Requested by:** Cristiano (ormasoftchile)

---

## Decision 1: Profileless non-interactive fail-fast is scoped to RunModeReal only

**Context:** Phase 1B added a fail-fast gate at `cmd/gert/run.go` that refused profileless non-interactive execution with exit code 2. The gate applied to all modes dispatched through `runWithMode`, including `dry-run`. This broke the VS Code extension's "Validate Inputs" command (extension calls `gert dry-run` via `execFile`, no TTY, no profile) and any CI validation caller.

**Empirical finding:** The gate's stated rationale ("without a profile the engine installs TerminalApprovalGate which blocks stdin") is factually wrong. `buildApprovalGate` (`internal/adapter/wire.go:330`) installs `TerminalApprovalGate` only when `attended==true` (i.e., TTY context). In non-TTY (CI) context it installs `NoOpApprovalGate`, which never blocks, regardless of profile presence. The correct rationale for the real-execution gate is governance completeness: profileless real execution lacks declared approval scope, context, and operator identity. For dry-run, `DryRunExecutorRegistry` (`internal/adapter/wire.go:121`) replaces all executors with no-ops; no step is executed and the governance gap doesn't apply.

**Decision:** Scope the gate to `mode == engine.RunModeReal`. The Phase 1B acceptance criterion (profileless non-interactive `gert run` fails immediately with exact error+fix text) is preserved exactly. Dry-run is exempt.

**Consumer impact:** None for real execution callers (no change). VS Code extension and CI validation callers using `gert dry-run` without `--profile` now work correctly.

---

## Decision 2: gert serve default binds loopback; non-loopback requires auth

**Context:** `cmd/gert/serve.go` defaulted `--addr` to `:7778` (all interfaces). The VS Code extension spawned `gert serve --addr :${port}` without auth flags. This made the Gert server reachable from the local network with no authentication whenever the graph preview was open.

**Decision:**
1. Change `--addr` default to `127.0.0.1:7778`. The VS Code extension probes `127.0.0.1:0` for a free port and communicates via `localhost`, so it is unaffected.
2. Add a startup safety check in `runServe`: if the resolved bind address is non-loopback AND no auth is configured, refuse with a clear actionable error. Operator who genuinely needs a public bind must configure `--auth-token` or `--auth-jwt-secret`.
3. Loopback + no auth always proceeds (local development, extension).
4. No opt-out flag is added — fail closed is the correct security posture. An operator who needs public bind simply adds auth.

**Implementation note:** The `net.Listen` call in `internal/serve/server.go:165` is NOT patched. It faithfully binds whatever addr it is handed. The check is in the CLI layer (`cmd/gert/serve.go`) before any network operation.

**Consumer impact:** Anyone using `gert serve` with a non-loopback addr and no auth (e.g., `gert serve --addr 0.0.0.0:7778`) must now add auth. This is the correct behaviour — they were previously running an unauthenticated public server.

---

# Decision: Load-Bearing Flag Standard for CLI Integration Tests

**Date:** 2026-08-17
**Author:** Don (Backend/Tool-Runtime)
**Inbox target:** decisions.md (team decisions)

## Context

Phase 1B Item 4 CLI integration test initially had two flags (`--package-map`, `--profile`) that were empirically confirmed no-ops: removing either flag left the test green. This is structurally equivalent to a parsed-but-unreachable field — the exact failure class the reachability gate was built to catch.

## Decision

A CLI integration test that exercises `--package-map` or `--profile` must be structured so that removing either flag changes the exit code:

1. **`--package-map` is load-bearing** when the project config (`requires:`) has NO entry for the package under test. The tool is resolved only through the package-map override. Without the flag, PKG-011/PKG-001 fires → exitValidation.

2. **`--profile` is load-bearing** when the tool definition's static transport URL is a placeholder that cannot serve real requests (e.g., `https://127.0.0.1/` with no server), and the profile's endpoint override redirects to a live mock server. Without the flag, the transport fails to connect → exitFailure.

Pointing `--package-map` at a different directory that produces the same output proves only that flag parsing works, not that resolution works. This is insufficient.

## IMDS Injection Seam

Added `SetIMDSEndpointForTest(endpoint string) func()` to `internal/tool/auth_managed_identity.go` (production file, not test file) to allow `cmd/gert` integration tests to redirect managed-identity IMDS calls to a mock server. This seam:
- Uses a package-level `imdsEndpointOverride string` var
- Must not be called from `t.Parallel()` tests
- Is the correct pattern when the type under test is constructed inside `runRun()` with no injection seam accessible from the test package

## Files Changed (commit d53a45f)

- `internal/tool/auth_managed_identity.go` — added `SetIMDSEndpointForTest`
- `cmd/gert/synthetic_contract_integration_test.go` — rewrote main composition test to use mcp-http binding with load-bearing flags

---

# Decision: Synthetic Contract Proof — Split CLI and Runtime Tests

**Author:** Don (Backend/Tool-Runtime)
**Date:** 2026-08-17
**Context:** Phase 1B Item 4 — Dual-Binding Mechanism Proof
**Commit:** 5c0c002

---

## Decision

The mcp-http binding proof is split across two test layers, not consolidated into a single CLI integration test.

- **CLI integration test** (`cmd/gert/synthetic_contract_integration_test.go`): exercises `--package-map` AND `--profile` together in a single `runRun()` execution using the **native** binding. Profile carries a `provider: managed-identity` override for the tool — which native transport correctly ignores.

- **Runtime test** (`internal/tool/synthetic_contract_proof_test.go`): exercises the **mcp-http** binding directly via `MCPHTTPTransport + TokenGate + ManagedIdentityAuthProvider` with mock httptest servers (both IMDS and MCP server). Uses the package-internal `imdsHTTPClient` injection seam.

---

## Rationale

`ValidateTransportConfig` enforces HTTPS for the static URL in mcp-http tool YAML files. `httptest.NewServer` produces plain HTTP. The workaround (static URL = `https://127.0.0.1/` + profile endpoint override to `http://127.0.0.1:PORT`) works at the runtime layer but introduces fragility in the CLI path (profile must be carefully crafted to pass PLAN-013 while pointing at the test server). It also couples the managed-identity token acquisition to the full CLI execution path where `DefaultToolRuntime` creates a `ManagedIdentityAuthProvider` via `NewAuthProvider()` with no injection seam for the IMDS client.

The two-layer split is cleaner:
- CLI layer: proves the flag composition mechanism works (selection + parameterization).
- Runtime layer: proves the mcp-http binding + managed-identity chain works (IMDS → token → TokenGate → Authorization header).

Neither layer is weaker than the other — they test different invariants.

---

## Parity Harness Compatibility

The fixture is designed so that Tess's `pkg/contractparity` harness can compare both bindings without changes to the fixture. Required interface:

1. `writeOpsSyntheticNativePackage(t, root, opsMockBin string)` — writes native package files to a temp dir.
2. `newOpsSyntheticMCPServer(t)` — starts a test MCP server implementing the ops-synthetic contract.
3. Action names: `"status-query"` and `"pattern-search"`; tool name: `"ops-synthetic"`.
4. Test inputs: `target="api-gateway"` for status-query; `query="FOUND:pattern"` and `query="no-match"` for pattern-search.
5. The harness verifies both bindings return the same JSON shape (field names and types), not byte-identical values.

`writeOpsSyntheticNativePackage` is in `cmd/gert` (package `main`, test-only), so the parity harness will need its own copy or a shared helper moved to a testutil package when wired.

---

## No Change Required to Existing Tests

`TestRun_PackageMap_RealVsMockBinding` (`cmd/gert/packagemap_integration_test.go`) deliberately asserts that outputs DIFFER between bindings. The ops-synthetic tests are additive and do not conflict.

---

# Decision: TSG Unresolved Provider Binding (VSCODE-003)

**Date:** 2026-08-18  
**Author:** Don (Core Dev, Gert Core)  
**Requested by:** Cristiano  
**Status:** Implemented — awaiting merge gate

---

## Context

SQL Live-Site Operations supplied the TSG binding's LOGICAL contract (six
args: `incident_id`, `title`, `service`, `environment`, `logical_server`,
`database`) but could NOT supply the PROVIDER contract (VS Code MCP registered
tool name + parameter schema), because the tool was not present in
`vscode.lm.tools` during the live session.

The deliverable had `vscode_tool: tsg-recommendation-recommend` — a guessed
name. This is the same defect class as `icm-get-incident` caught in Rev 2.

---

## Decision

**Do NOT invent the provider tool name.** Mark the binding explicitly unresolved
with a new schema field `provider_unresolved: true` on `TransportConfig`.

**Fail closed at scan time (VSCODE-003).** `ValidateTransportConfig` in
`internal/tool/validate_transport.go` rejects any `vscode-mcp` tool with
`provider_unresolved: true` during `ParseToolFile`, before the tool enters the
registry. This is the earliest achievable fail point in the transport chain.

**Error text is unambiguous:**
> This is NOT a "tool not found" error: nobody has told us the real
> vscode.lm.tools name yet.

This distinguishes the unresolved-contract state from MCP server downtime.

---

## Rationale

**Why scan time, not runtime?**  
The prior vscode_input fail-closed checks (VSCODE-001, VSCODE-002) also fire
at scan time via `ValidateTransportConfig` → `ParseToolFile`. This is
consistent with the established pattern. Runtime would allow the tool into the
registry and would reach the bridge — a worse failure mode.

**Why not just leave vscode_tool absent?**  
An absent `vscode_tool` is valid and currently means pass-through (logical arg
names forwarded to the provider). With an unknown provider schema, pass-through
is a silent-wrong-args hazard. `provider_unresolved: true` makes the intent
explicit and forces the error.

**Why a schema field and not a comment?**  
Comments are not machine-checked. A schema field enforced in the validation
chain cannot be silently ignored, and the reachability gate ensures the check
cannot be removed without a test failing.

---

## What was NOT done

The provider tool name and parameter schema were NOT invented. No placeholder,
plausible guess, or synthetic example appears in any deliverable or test
fixture that could be mistaken for a real registered name.

The reachability probe uses the synthetic name `tsg-unresolved-probe` —
obviously non-real, never in a deliverable.

---

## Resolution path

When SQL Live-Site Operations confirms the provider contract:
1. Set `vscode_tool:` to the real registered name.
2. Add `vscode_input:` mappings (provider key → logical arg).
3. Remove `provider_unresolved: true`.
4. Update `TestDeliverableContracts_TSGRecommendation` to assert ParseToolFile
   SUCCEEDS and check outputs (current test asserts failure — must be inverted).
5. Sync `gert/internal/tool/testdata/` with the deliverable.

---

## Files changed

**gert (branch: feature/tsg-unresolved-provider-binding, commit 1824af7):**
- `pkg/schema/tool.go` — `ProviderUnresolved bool` on TransportConfig
- `internal/tool/validate_transport.go` — VSCODE-003 check
- `internal/tool/testdata/tsg-recommendation.vscode-mcp.tool.yaml` — updated
- `internal/tool/deliverable_contract_test.go` — updated
- `cmd/gert/reachability_registry_test.go` — new entry
- `cmd/gert/reachability_probes_test.go` — new probe

**gert-private (commit b598128):**
- `deliverables/phase2/tsg-recommendation.vscode-mcp.tool.yaml` — updated

---

# Decision: VS Code Extension Config Scoping and Registry Parse Axes

**Date:** 2026-08-18  
**Author:** Ken  
**Commit:** 1cd7542 (gert-vscode main)  
**Context:** SQL Live-Site live blockers after Rev 3

---

## Decision 1: binaryPath is folder-scoped (same as packageMap)

`serverManager.ts spawnServer()` now reads BOTH `gert.binaryPath` AND `gert.packageMap` through a `GetScopedSetting` callback scoped to `vscode.Uri.file(runbookPath)`.

**Judgement on binaryPath:** YES, folder-scoped. In a multi-root workspace, different projects may declare different local gert binary builds. Reading `binaryPath` from the wrong scope silently invokes the wrong binary for the active project. This is the same class of defect as the `packageMap` bug that caused the live-site incident.

**What is NOT folder-scoped:** `serverUrl` and `autoStartServer` (read in `ensureRunning()`, not `spawnServer()`). These are global user preferences — a user's choice to auto-spawn a server or point at an external URL applies regardless of which project folder is active.

---

## Decision 2: GetScopedSetting as the injectable bridge

`GetScopedSetting = (key: string, defaultValue: string) => string` is the minimal injectable type for unit-testing scoped config reads without an extension host. It has no vscode dependency and is tested via two mock instances (one simulating scoped-folder settings, one simulating the wrong-scope fallback). The mutation that makes this test meaningful: change `readGertSpawnConfig` to ignore its `getSetting` argument → the multi-root test fails.

---

## Decision 3: Three Go-core parse axes, permanently mirrored

`buildRegistryFromDir` must mirror THREE axes from `tool.go`:

| Axis | Go source | TypeScript implementation |
|------|-----------|--------------------------|
| meta.name wins over flat name | `ToolDef.UnmarshalYAML`: `if raw.Meta.Name != "" { t.Name = raw.Meta.Name }` | `meta?.name` overwrites `def.name` when non-empty |
| Sequence OR mapping actions | `decodeToolActions`: `SequenceNode` vs `MappingNode` | `Array.isArray` → sequence; `typeof === 'object'` → mapping |
| transport.mode wins over transport.type | `TransportConfig.UnmarshalYAML`: `if t.Mode != "" { t.Type = Transport(t.Mode) }` | `effectiveMode = transport.mode ?? transport.type` |

**Drift guard:** `test/toolDefinitionRegistry.test.js` contains a `defect2-drift-guard` test that asserts all six shape-matrix entries are present in the registry. Any future removal from `buildRegistryFromDir` triggers a failure with the message `shape "X" must produce registry key "Y"`. This test must be updated whenever a new axis is added to core.

---

## Decision 4: Consumer contracts must not appear in platform test fixtures

`tsg-recommendation` fixture and all its assertions were removed from gert-vscode as part of this work. This is the third time this contract has been reverted (gert 18093a5, gert-private f3ad30e, now gert-vscode 1cd7542). The `icm/get-incident` fixture is the sole consumer reference permitted — it was explicitly supplied by the consumer as the required regression fixture.

**Rule going forward:** Platform test fixtures use neutral names (`ops-*`, `alpha`, `fallback-tool`, etc.). A consumer's exact YAML is acceptable only when the consumer explicitly supplies it as the required regression vector in the task description and it is used verbatim without augmenting with their field names.

---

# Decision: Static source guard for getConfiguration scoping

**Date:** 2026-08-18  
**Author:** Ken  
**Commit:** 08de611 (gert-vscode main)

## Decision

Add `repoBoundary/rule7` to `test/repoBoundary.test.js`: every `getConfiguration(` call in `src/**/*.ts` must either supply a second resource argument or carry a `// window-scoped: <reason>` comment in the immediately preceding comment block.

## Context

After `makeScopedGetter` was extracted and its spy test made load-bearing (commit fadc2fa), the two direct `vscode.workspace.getConfiguration` call sites in `serverManager.ts` (lines 50 and 80) were still unguarded by any test. Mutating either site to drop the resource URI produced zero test failures. The originally reported live defect (multi-root workspace using the wrong folder's settings) was fully reintroducible from the published commit.

Module mocking of `vscode` is impractical under plain `node --test` because `vscode` is host-provided and unresolvable without `--experimental-test-module-mocks` — a poor trade for a two-line guard.

## Alternative considered: expand the spy test

A second spy in `serverManager.test.js` could have directly tested the two call sites. This was rejected because `vscode.workspace.getConfiguration` is a host-provided API that cannot be imported in a unit test environment. The extraction pattern (`makeScopedGetter`) works precisely because the API is passed in; it cannot be applied to the call sites themselves without either mocking the entire `vscode` module or restructuring both call sites into extractable helpers, which would be disproportionate.

## Why static scan is the right tool here

- No runtime dependency — scans source text, no host environment needed.
- Self-updating — new `src/` files auto-trip the rule without any allowlist change.
- Zero false negatives — every `getConfiguration(` site must be either scoped or explicitly justified with a reason comment.
- Consistent with existing repo style — `repoBoundary.test.js` already guards structural rules at the source level (cross-repo paths, skip declarations, orphaned files).

## Scope of the rule

- Searches `src/**/*.ts`; ignores comment-only lines (leading `//` or `*`).
- Resource-scoped: detected by a comma after `getConfiguration(` at top-level parenthesis depth (same line or next line).
- Window-scoped: detected by `// window-scoped:` anywhere in the contiguous comment block immediately preceding the call line.
- `totalCallSites > 0` assertion prevents false-green on an empty scan.

## Window-scoped exception applied

`extension.ts:76` — `mcpBridge.toolNameOverrides`. Read at extension activation before any runbook is open. There is no resource to scope to at this point; window-scope is the only correct choice. Marker text records the reason in source for future reviewers.

---

# Decision: Use makeScopedGetter as the testable seam for vscode config scoping

**Date:** 2026-08-18  
**Author:** Ken  
**Scope:** gert-vscode — serverLaunch.ts / serverManager.ts

## Decision

`vscode.workspace.getConfiguration` calls that need folder-scoping MUST go through
`makeScopedGetter(runbookPath, getConfigurationFn)` (exported from `serverLaunch.ts`)
rather than being inlined into `serverManager.ts` or `extension.ts`.

## Rationale

`vscode.workspace.getConfiguration` cannot be called in unit tests without an extension
host. Inlining the call makes the resource URI argument (e.g.
`vscode.Uri.file(runbookPath)`) invisible to any test that can run under `node --test`.

`makeScopedGetter` accepts a `GetConfigurationFn` callback. Unit tests pass a spy that
records `resource.fsPath` and assert it equals the runbook path. Deleting the resource
argument from `makeScopedGetter` fails the spy test. The remaining delegation in
`serverManager.ts` is compile-time safe: the `GetConfigurationFn` signature requires
`resource`, so dropping it is a TypeScript error.

## Scoping judgment (documented decisions)

| Setting | Location | Scoped? | Reason |
|---|---|---|---|
| `binaryPath` | serverManager.ts spawnServer | YES | per-folder builds |
| `packageMap` | serverManager.ts spawnServer | YES | per-folder map path |
| `serverUrl` | serverManager.ts ensureRunning | YES | per-folder external server |
| `autoStartServer` | serverManager.ts ensureRunning | YES | per-folder spawn opt-out |
| `binaryPath` | extension.ts previewProse | YES | runbook path in scope |
| `binaryPath` | extension.ts validateInputs | YES | runbook path in scope |
| `mcpBridge.toolNameOverrides` | extension.ts activate | NO | read at activation, no runbook open |

## Binding rule going forward

Any new `getConfiguration('gert')` call in a function that has a `runbookPath` in scope
MUST use `makeScopedGetter` (or equivalent). Leaving it unscoped is an oversight unless
the call site pre-dates any runbook being open (activation only).

---

# Ruling: Reproducibility Fix — Committing 91 Untracked Build Inputs

**Date:** 2026-08-17
**Author:** Ken
**Status:** SHIPPED — final commit `eeb9d66`

## Problem

`main` was not reproducible from a clean checkout. Two root causes:

1. **91 untracked source files**: entire packages (`pkg/pkgcatalog`, `pkg/pkgpath`,
   `pkg/semver`, `pkg/pkgdrift`), production source files (`include_closure.go`,
   `auth_gate.go`, `mcp_http.go`, etc.), test files, and conformance data — all
   present in the working tree but never committed to git.

2. **45 modified tracked files not committed**: in-flight changes to `pkg/schema/steps.go`,
   `pkg/trace/event.go`, `pkg/engine/validated_plan.go`, and many others were present
   in the working tree but not in git. The working tree built because of this; a
   clean checkout failed with symbol-not-found errors.

## Classification of files

- **91 untracked files**: all committed except one.
- **1 deliberately NOT committed**: `examples/simple-health-check/examples.code-workspace`
  — a VS Code workspace file referencing local absolute paths including
  `../../../../gert-sqllivesite` (a private local repo). Added `*.code-workspace`
  to `.gitignore` instead.
- **No secrets found**: the `token =` / `Authorization:` / `secret =` pattern matches
  were all variable names, doc comments, or synthetic sentinel values (`TESS_SENTINEL_TOKEN_D42E9B1C`).
  The `tools/icm.tool.yaml` URL and scope are public endpoint/scope identifiers, not credentials.

## Commits (in order)

| Hash | Contents |
|---|---|
| `90435b0` | pkg/semver, pkg/pkgpath, pkg/pkgdrift, pkg/pkgcatalog (missing packages) |
| `97182d6` | pkg/schema additions (enum, packagelock, packagevalidate, projectconfig, toolpackage) |
| `f3f8129` | internal/tool additions (MCP HTTP, auth, overlay) |
| `29de6e5` | internal/adapter, executor, planner, parser, replay, serve additions |
| `cce0720` | internal/conformance test data + engine enum test |
| `603ee7e` | cmd/gert CLI source and integration tests |
| `c3512ab` | pkg/errkit, pkgsubst, preview, run, trace additions |
| `37aef73` | tools/icm.tool.yaml + .gitignore *.code-workspace |
| `41165f0` | internal/conformance/dyninclude_conformance_test.go (missed in batch) |
| `57c2f7a` | Modified: pkg/schema/steps.go, runbook.go, go.mod |
| `28bca1b` | Modified: pkg/trace/event.go, pkg/engine/validated_plan.go |
| `d737875` | Modified: internal/tool/mcp.go, native.go, runtime.go |
| `17ae2f2` | Modified: internal/adapter, executor, governance, parser, planner, replay, serve |
| `eeb9d66` | Modified: pkg/*, cmd/*, examples, README, schemas |

## Verification

From `git worktree add --detach C:\One\OpenSource\_gert_verify HEAD`:
- `go build ./...` → exit 0
- `go test ./...` → exit 0, 63 packages all `ok`
- `TestCLI_ReachabilityGate` → PASS from clean checkout
- `internal/tool/auth_gate.go` tracked: confirmed
- `cmd/gert/packagemap_integration_test.go` tracked: confirmed
- `git ls-files --others --exclude-standard` → empty (zero untracked source files)

## Root cause

The team was committing feature work in the production files (tracked, modified)
but leaving the new files it introduced (untracked) on disk. The working tree
always built because Go uses the filesystem. CI never caught this because CI
does a clean checkout — but CI was not being run against `main` at the time of
the failures.

The fix requires no process rule (we have the reachability gate now). The
structural discipline is: every PR must pass `go build ./...` and `go test ./...`
from a clean checkout of its branch.

---

# ken-skip-defect.md

**Date:** 2026-08-18  
**Author:** Ken  
**Requested by:** Cristiano  
**Branch:** `ken/hermetic-regression-tests` (gert-vscode)  
**Commits:** `fc273b6` (rev 1 — rejected), `0b98842` (rev 2 — rejected), `424ce26` (rev 3 — current)  
**Status:** Rev 3 delivered — awaiting Cristiano's merge gate

---

## Rev 1 Rejection Reason

`test/integration/cli.test.js` was not reached by any script or CI job. Converting a visible skip into an invisible orphan is strictly worse. The guard's remediation text compounded the defect by instructing developers to move tests to `test/integration/` without wiring them up.

## Rev 2 Rejection Reason

`test:integration` script existed in `package.json` but was not invoked by any CI job. Rule5 certified `test/integration/*.test.js` files as "reachable" because the glob matched — but rule5 only proves script reachability, not CI invocation. A script not called by CI certifies orphans while executing nothing. Rule5 alone is not transitive all the way to CI.

## Rev 3 (commit 424ce26) — Rule6 added

**Rule6:** every npm script containing `node --test` must be invoked by at least one step in `.github/workflows/*.yml`. Combined with rule5, this makes the chain transitive and complete:

```
test file → matched by a script glob (rule5)
          → that script is invoked by CI (rule6)
```

**`test:integration` deleted:** no unwired scripts. Rule6 forces wiring at the moment the script is reintroduced.

**Regex fix:** `isScriptInvokedByCI` used `\b` which matches `test` inside `test:integration` (`:` is a non-word char). Fixed to `(?!\S)`. Bug discovered during mutation 3.

**Mutation results (424ce26):**

| Mutation | Rule | Direction |
|---|---|---|
| `test:unwired` added to package.json (not in CI) | rule6 | RED |
| `npm run test:unwired` added as CI step | — | GREEN 123 |
| Mutations 1+2 restored; `npm test` removed from ci.yml | rule6 | RED |
| Restored | — | GREEN 123 |

**Detached-worktree results (424ce26):**
```
tests 123 · pass 123 · fail 0 · skipped 0
```

---

## Background

Cristiano identified that `test/enumRuntimeRegression.test.js` on `gert-vscode` main (`f1e5c51`) contained four tests that silently skipped in any clean CI checkout:

```
{ skip: !gertBin && 'no gert binary found; set GERT_BIN or build ../gert/gert(.exe)' }
```

The skip condition was `true` whenever `GERT_BIN` was unset and `../gert/gert(.exe)` did not exist — which is always the case in a detached worktree of `gert-vscode` alone. The reported count of 118/118 was wrong; the true result was **114 pass / 4 skipped**.

This is the **ninth occurrence** of the team's systemic "appears covered but is not reachable" bug class. The mirror defect on the Go side (`gert/internal/tool/deliverable_contract_test.go` reading `../gert-private/`) was fixed by Don in `21e9c6c`/`d0d8aba` with `TestRepoBoundary_NoExternalPaths`. This document records the JS-side fix.

---

## What the Four Tests Asserted

| Test | AR-CE ID | Client function tested | Option |
|------|----------|----------------------|--------|
| ENUM-008 survives `deriveFailureMessage` | CE-V-04/D-3 | `deriveFailureMessage(err)` | (a) fixture |
| Declared enum member accepted by CLI | CE-S-01/CE-V-01 | *none — pure CLI acceptance* | (b) integration |
| ENUM-W001 extracted by `warningLines` | CE-W-01 | `warningLines(stderr)` | (a) fixture |
| `extractInputDecls` parses graphjson | CE-D-01/CE-D-02 | `extractInputDecls(doc)` | (a) fixture |

---

## Decisions Made

### CE-V-04/D-3, CE-W-01, CE-D-01/CE-D-02 — Option (a): Committed Fixtures

The relevant CLI output was captured from the gert binary at f1e5c51 (2026-08-18) and committed as:

- `test/fixtures/enum-error-stderr.txt` — stderr from `gert dry-run --var env_name=not-a-member enum.runbook.yaml`
- `test/fixtures/enum-warn-stderr.txt` — stderr from `gert dry-run --var env_name=prod enum-warn.runbook.yaml`
- `test/fixtures/enum-preview-graphjson.json` — stdout from `gert preview --format graphjson enum.runbook.yaml` (absolute runbook paths stripped; not asserted on and machine-specific)

Each test now passes the fixture content through the client-side helper function and asserts the same structural invariants as before.

**Declared drift risk:** if the CLI changes the ENUM-008 prefix, ENUM-W001 format, or renames the `inputs` key in the graphjson Document, these tests will continue to pass against stale fixtures. The live-binary integration suite (`test/integration/cli.test.js`) is the intended drift defence — it must be run wherever `GERT_BIN` is available.

### CE-S-01/CE-V-01 — Option (b): Integration Test

This test's only assertion was `assert.match(stdout, /dry-run complete/)` after `pexec(gertBin, ['dry-run', '--var', 'env_name=prod', FIXTURE])`. No extension client function is in the loop. There is no fixture approach that tests any extension code here — the test is a pure CLI acceptance check.

The test was moved to `test/integration/cli.test.js`. That file:
- Is NOT picked up by `npm test` (`test/integration/` is a subdirectory, the glob is `test/*.test.js`)
- Fails (never skips) when `GERT_BIN` is absent
- Documents clearly what it requires and why

The contract guarded by CE-S-01/CE-V-01 (the CLI does not reject a declared enum member) is protected at the Go level by `gert`'s own test suite.

---

## Guard Added: `test/repoBoundary.test.js`

Mirrors `gert/internal/tool/repo_boundary_test.go` (`TestRepoBoundary_NoExternalPaths`).

Walks `test/` and `src/` and fails on:

1. **Sibling-repo name references** — `gert-private` anywhere in a test/source file.
2. **Cross-repo path traversals** — `path.join(__dirname, '..', '..', 'gert'` and `'../gert'` variants.
3. **`process.env.GERT_BIN` in unit test files** — the canonical escape hatch to the sibling binary; prohibited in `test/*.test.js` (integration files in `test/integration/` are exempt).
4. **`skip:` in test declarations** — the `{ skip: ... }` Node.js test option; any occurrence in a unit test file fails the guard.

**Allowlist:** empty — no justified exceptions currently exist. The allowlist mechanism is present (with a mandatory-justification-per-entry contract) for future use.

**Mutation test results:**

| Mutation | Guard result |
|----------|-------------|
| Added `{ skip: 'probe' }` to CE-V-04 test | ✖ RED — `contains a skip: test option` |
| Added `path.join(__dirname, '..', '..', 'gert'` comment | ✖ RED — `contains cross-repo path traversal` |
| Both removed | ✔ GREEN — 118/118 pass |

---

## Verification Results

From a detached worktree of `fc273b6`:

```
git worktree add --detach <wt> fc273b6
git status --porcelain          # empty
npm ci                          # exit 0
npm run compile                 # exit 0
npm test
```

```
tests    118
pass     118
fail       0
skipped    0
```

**Before this fix (f1e5c51):**
```
tests    118
pass     114
fail       0
skipped    4
```

---

## Hard Rules Compliance

- ✅ No `git add -A`; all staging by explicit path.
- ✅ Branch from `main` (`f1e5c51`), not from `squad/rev3-tsg-absence`.
- ✅ `git status --porcelain` confirmed empty before staging — session 89b67e11's work was absent.
- ✅ No assertion weakened.
- ✅ Previously-skipped tests did not fail on first run (fixtures matched expected patterns).
- ✅ Guard mutation-tested with measured before/after output. "Verified by code review" was not used.
- ✅ Detached worktree used for final verification — never the working tree.
- ✅ Full four-number breakdown reported (tests / pass / fail / skipped).

---

## Rev 2 Changes (commit 0b98842)

| File | Status |
|------|--------|
| `test/integration/cli.test.js` | Deleted — CE-S-01/CE-V-01 orphan removed; see rejection reason above |
| `.github/workflows/ci.yml` | Modified — added `npm test` step (tests never ran in CI before); documented integration test pattern in comments |
| `package.json` | Modified — added `test:integration` script for future binary-requiring tests |
| `test/repoBoundary.test.js` | Modified — split into 5 per-rule tests; added rule5 (orphan detection); fixed rule3 remediation message |

**Orphan detection (rule5):** reads `package.json` scripts at runtime; extracts `node --test <glob>` patterns; fails any `test/**/*.test.js` not matched by at least one pattern. Current configured globs: `test/*.test.js`, `test/integration/**/*.test.js`.

**Mutation results (0b98842):**

| Mutation | Rule | Direction |
|---|---|---|
| `test/orphan-dir/orphan-probe.test.js` (no script covers it) | rule5 | RED |
| Moved to `test/integration/` (covered by `test:integration`) | — | GREEN 122 |
| Restored | — | GREEN 122 |

**Detached-worktree results (0b98842):**

```
git status --porcelain  →  empty
npm ci                  →  exit 0
npm run compile         →  exit 0
npm test
  tests    122
  pass     122
  fail       0
  skipped    0
```

---

## Files Changed (All Commits, Explicit Paths)

| File | Status |
|------|--------|
| `test/enumRuntimeRegression.test.js` | Modified (fc273b6) — rewritten (4→3 tests, fixture-based) |
| `test/fixtures/enum-error-stderr.txt` | Added (fc273b6) — ENUM-008 stderr fixture |
| `test/fixtures/enum-warn-stderr.txt` | Added (fc273b6) — ENUM-W001 stderr fixture |
| `test/fixtures/enum-preview-graphjson.json` | Added (fc273b6) — graphjson fixture (paths sanitized) |
| `test/integration/cli.test.js` | Added (fc273b6) then Deleted (0b98842) — orphan removed |
| `test/repoBoundary.test.js` | Added (fc273b6), Modified (0b98842) — 5 per-rule tests + orphan detection |
| `.github/workflows/ci.yml` | Modified (0b98842) — added npm test to CI |
| `package.json` | Modified (0b98842) — added test:integration script |


---

# Decision: Contract-Parity Harness API Shape

**Date:** 2026-08-17
**Author:** Tess
**Status:** Informational — no arbitration needed; records design choices made while implementing consumer Item 1.

---

## Context

The SQL Live-Site Operations team requested a reusable contract-parity test harness
(Item 1 of 3). This records the design decisions made in `pkg/contractparity`.

---

## Decisions

### 1. Package location: `pkg/contractparity` (not `internal/`)

The consumer explicitly said they do not want to copy a Gert-specific test. The
package must be importable from a foreign repo. `internal/` is inaccessible outside
the module; `pkg/` is the correct location.

### 2. Pure function + thin adapter split

The comparison logic (`CompareOutputs`, `CompareMeta`, `Check`) is separated from the
`testing.T` binding (`AssertParity`, `RequireParity`). Rationale:
- Core logic is testable without `*testing.T`.
- Consumers who want structured diffs (e.g. for CI reporting) can call `Check` and
  inspect `Report` without going through a test framework.
- The testing adapter is a 3-line wrapper; no hidden complexity.

### 3. `Invoke` signature: `func(ctx, args) (map[string]any, error)`

Rather than importing `pkg/tool.ToolResult`, the harness accepts `map[string]any`
directly. This removes a Gert-specific type dependency from a package intended for
foreign consumers. Don's bindings can wrap `ToolResult.Output` trivially.

### 4. `ActionMeta` fields are all `*T` (nil = skip)

A consumer that does not wish to assert classification or idempotency simply leaves the
pointer nil; the harness silently skips that field. This avoids forcing consumers to
fill in fields they don't care about, and avoids false violations when one binding
declares a field the other omits.

### 5. `CompareMeta` takes an existing `*Report` and appends

Rather than returning a second separate Report, `CompareMeta(r, ...)` appends meta
violations to an existing output-comparison report. Callers get one unified Report with
all violations in sorted order, suitable for a single diagnostic message.

### 6. `ViolationMissingKey` vs `ViolationExtraKey` are distinct kinds

Keeping them separate lets consumers distinguish "B is incomplete" from "B leaks
internal fields". Both are parity failures but have different remediation paths.

---

## Non-decisions (deferred to consumers)

- The harness does NOT assert on `ExitCode` or `Stdout/Stderr` — only `Output` map
  shape and `ActionMeta`. Consumers with stricter contracts can extend `CompareMeta`.
- Deep recursive comparison of nested maps is intentionally not implemented; only the
  top-level reflect.Type is compared. If a consumer needs recursive shape checking they
  can compose multiple `CompareOutputs` calls on sub-maps.

---

# Design Decision: Runtime Output Contract Enforcement for Non-Substituted Tool Actions

**Date:** 2026-08-17  
**Author:** Tess (conformance/test engineer)  
**Commit:** ca867a1  
**Files changed:** `internal/executor/tool.go`, `internal/executor/tool_output_contract_test.go`, `cmd/gert/reachability_registry_test.go`, `cmd/gert/reachability_probes_test.go`

---

## Problem

`returns:` / `outputs:` declarations on non-substituted `.tool.yaml` actions (native, mcp-stdio, mcp-http transports) were never checked at runtime. The executor assembled `result.Output` with a verbatim copy of `res.Output` and injected `stdout`, `stderr`, `exit_code` without consulting the declared contract at all.

Substituted actions (`execute.kind: runbook`) already enforced their output contracts via `executeSubstitution` → `coerceOutputAny`. This was the fifth instance of the declared-but-unenforced bug class on the project.

---

## Decision

Add `enforceOutputContract()` to `internal/executor/tool.go` called after `result.Output` is assembled for every non-substituted tool execution.

### Rule 1 — stdout / stderr / exit_code are process-level channels

These three keys are injected unconditionally by the executor and exist on every result regardless of the declared contract. They are never part of the semantic `outputs:` contract.

**Enforcement rule:**
- They are NEVER flagged as "undeclared" even if absent from `outputs:`.
- They are NEVER required to be present in `outputs:` declarations.
- They are always passed through to the result unchanged.

This preserves all existing runbooks that capture `{{ step.stdout }}` etc. without any tool YAML change.

### Rule 2 — Empty / absent outputs: declaration = unconstrained

If a `.tool.yaml` action declares no `outputs:` block (empty map or nil), enforcement is skipped entirely and all outputs pass through unchanged. This is the current state of 100% of existing tool YAML files — zero tools are affected by this change.

**Bounded escape hatch:** the escape only applies when `declaredOutputs` is nil or empty. Any tool that adds `outputs:` to its YAML immediately gets enforcement. This cannot silently swallow the whole feature because the reachability probe (see below) runs a tool WITH declared outputs and asserts the enforcement fires.

### Rule 3 — Fail closed, not warn-and-continue

Any violation (missing declared output, undeclared extra output, type-incompatible value) fails the step immediately with `StepStatusFailed` and a structured error message naming the tool, action, and all offending fields. All errors are collected and reported together (not first-error-only).

### Rule 4 — Reuse coerceOutputAny, do not diverge

The same `coerceOutputAny` function used by the substituted path is called by `enforceOutputContract`. Both enforcement paths agree on type coercion semantics. Any divergence would itself be a contract bug.

---

## Implementation Details

`resolvedActionDef *tool.ToolAction` is captured in the `Execute()` method after the `ToolDefLookup` call (non-substituted branch only). If the runtime does not implement `ToolDefLookup` or the tool/action is not found, `resolvedActionDef` remains nil and enforcement is skipped — this is intentional graceful degradation for unknown tools.

After `result.Output` is assembled from `res.Output` (and before the result is returned), enforcement is called:

```go
if resolvedActionDef != nil && len(resolvedActionDef.Outputs) > 0 {
    validated, cerr := enforceOutputContract(toolName, action, resolvedActionDef.Outputs, result.Output)
    if cerr != nil {
        // fail step
    }
    result.Output = validated
}
```

---

## Reachability

Entry added to `reachabilityRegistry` in `cmd/gert/reachability_registry_test.go`:

```
"ToolAction.Outputs (non-substituted enforcement)" → statusReachable
```

Probe (`testCLI_OutputContractEnforcement_Reachable`): creates a native tool YAML that declares `outputs: {result: string}`, runs it via gert, asserts `code != exitSuccess`. Native transport never populates `res.Output` — so "result" is always missing → enforcement always fails the step → process exits with code 1. If the production enforcement call is removed, the step would succeed (no enforcement = pass-through) and the probe would fire `t.Fatal`.

---

## Mutation Control Results (measured, not reasoned)

Mutation applied: remove `var resolvedActionDef`, `resolvedActionDef = actionDef`, and the enforcement block from `Execute()`.

| Test | Result without enforcement |
|------|---------------------------|
| `TestToolExecutor_OutputContract_Missing_Fail` | FAIL — "MUTATION CONTROL FAILED" |
| `TestToolExecutor_OutputContract_Undeclared_Fail` | FAIL — "MUTATION CONTROL FAILED" |
| `TestToolExecutor_OutputContract_TypeMismatch_Fail` | FAIL — "MUTATION CONTROL FAILED" |
| `TestToolExecutor_OutputContract_Satisfied_Pass` | PASS (correct — enforcement not needed for pass case) |
| `TestToolExecutor_OutputContract_NoDeclaration_PassThrough` | PASS (correct — no declaration, no enforcement) |

---

## Tests — Full Coverage

16 tests in `internal/executor/tool_output_contract_test.go`:

**Pure `enforceOutputContract` function (9):**
- `_NilDeclaration_PassThrough` — nil declaredOutputs: all pass
- `_EmptyDeclaration_PassThrough` — empty map: all pass
- `_Satisfied_Pass` — declared field present with correct type
- `_Missing_Fail` — declared field absent → error
- `_Undeclared_Fail` — extra field not declared → error
- `_TypeMismatch_Fail` — wrong type → error
- `_ProcessChannels_NeverFlagged` — stdout/stderr/exit_code never flagged
- `_MultipleErrors_AllReported` — all violations collected, not first-only
- `_TypeCoercionFloat_Pass` — float64 accepted for "number" type (matches coerceOutputAny)

**Execute-path integration / mutation controls (7):**
- `Satisfied_Pass` — declared output present, step completes
- `Missing_Fail` (mutation control) — declared output absent, step fails
- `Undeclared_Fail` (mutation control) — extra output, step fails
- `TypeMismatch_Fail` (mutation control) — wrong type, step fails
- `NoDeclaration_PassThrough` — no outputs declaration, pass-through
- `ProcessChannels_AlwaysPresent` — stdout/stderr/exit_code in result even when declared outputs satisfied
- `NoDefRegistered_PassThrough` — runtime has no ToolDefLookup, pass-through

---

## Pre-existing Tests Modified

None. All 82 packages passed without test changes. The 0 existing tools that declare `outputs:` on non-substituted actions take the unconstrained path — no behavioral change.

