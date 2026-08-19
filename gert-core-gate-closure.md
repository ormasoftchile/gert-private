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
