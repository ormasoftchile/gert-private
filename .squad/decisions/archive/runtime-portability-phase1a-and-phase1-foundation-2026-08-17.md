# Archived Decisions Block — Archived Phase 1A runtime portability + Phase 1 foundation

**Effort slug:** runtime-portability-phase1a-and-phase1-foundation  
**Source:** .squad/decisions.md lines 29–1338 at archival time  
**Archived at:** 2026-08-19T17:08:50-07:00  
**Ruling:** .squad/decisions.md live entry `2026-08-19 — decisions-ledger-archival-ruling`  
**Payload SHA-256:** 6a9679ebce89f89adfe9ca0b2ce7576161bca0758be4c71565d1cb3a4da43e1e  
**Payload bytes:** 68396

---

## PHASE 1A: Runtime Portability (Active — Phase 1B Built On This)

*Six correspondence bodies (2026-08-16 to 2026-08-17) are archived to `.squad/decisions/archive/`. Binding rulings extracted below. All prose, negotiation history, and implementation detail is in the archive files.*

---

### Classification Field

**Ruling:** `classification` is a per-action field (`Classification *string` on `ToolAction`), not per-tool. Four-value enum: `read-only | mutating | destructive | unspecified` (nil pointer = unspecified). A single tool legitimately has read-only `get-incident` and destructive `kill-process`. This field is permanent — no deprecation horizon.

**Rationale:** Per-tool classification cannot express mixed-action tools; nil pointer cleanly distinguishes absent intent from explicit value; `omitempty` on pointer causes absent YAML fields to unmarshal as nil.

→ [Gate Closure Revised](decisions/archive/phase1a-gate-closure-revised.md) §2a; [Implementation Impact](decisions/archive/phase1a-don-implementation-impact.md) §B

---

### Nil/Absent vs. Explicit `requires-approval: false` — Precedence

**Ruling:** A non-nil `ToolGovernance` pointer with `RequiresApproval: false` = author explicitly wrote a governance block opting out → honor as legacy `classification: read-only` equivalent (no prompt). A nil `ToolGovernance` pointer (governance block entirely absent) = author never expressed intent = `unspecified`. This distinction exists in the schema today via `*ToolGovernance` being a pointer.

**Rationale:** Preserves existing explicit author signals without forcing tool authors to re-classify all legacy tools; migration pressure comes from plan-time PKG-W warnings, not breaking changes.

→ [Final Acknowledgment](decisions/archive/phase1a-final-acknowledgment.md) §1; [Final Impact Assessment](decisions/archive/phase1a-don-final-impact-assessment.md) Q3

---

### Approval Gate Precedence Table

**Ruling:** Governing precedence for all invocation paths:

| `requires-approval` | `classification` | Behavior |
|---|---|---|
| `true` | any (incl. nil/unspecified) | Gate fires. Explicit legacy bool wins. |
| `false` or absent | `read-only` | No gate. |
| `false` or absent | `mutating` | Gate fires if profile approval mode ≥ interactive. Auto-approve in test context. |
| `false` or absent | `destructive` | Gate ALWAYS fires. |
| `false` or absent | nil (unspecified), interactive | Gate fires. |
| `false` or absent | nil (unspecified), unattended | Denied (fail-closed). |

**Rationale:** Classification is an opt-in to reduced friction; absence must never grant additional execution rights.

→ [Gate Closure Revised](decisions/archive/phase1a-gate-closure-revised.md) §2a; [Implementation Impact](decisions/archive/phase1a-don-implementation-impact.md) §B

---

### `legacy_unspecified_policy` Profile Field

**Ruling:** Profile-level field `legacy_unspecified_policy: allow | prompt`. Default: `prompt` (new rule). Existing deployments set `allow` during migration window to preserve pre-classification behavior. Suppress prompt only — does NOT relax retry, idempotency, or late-result behavior.

**Rationale:** Named, explicit, auditable escape hatch. `allow` suppresses the migration prompt; it does NOT set `result.Allowed = true` or modify classification.

→ [Final Acknowledgment](decisions/archive/phase1a-final-acknowledgment.md) §1; [Gate Closure Revised](decisions/archive/phase1a-gate-closure-revised.md) §2a

---

### Plan-Time PKG-W Warning for Unclassified Actions

**Ruling:** For each unclassified action when the profile is non-test, emit a PKG-W warning at plan time: `"action '<name>' has no classification; treating as unspecified. Add classification: read-only to suppress."` Non-fatal, stderr.

**Rationale:** Makes migration pressure visible without breaking runs.

→ [Final Acknowledgment](decisions/archive/phase1a-final-acknowledgment.md) §1

---

### `unspecified` Behaviors (Retry, Late-Result, Unattended)

**Ruling:** `unspecified` actions: no retry; late/lost result → INDETERMINATE + halt; unattended → deny (fail-closed); test context → auto-approve (native transport only per transport rules).

**Rationale:** Conservative defaults; absence of classification must never grant execution rights.

→ [Gate Closure Revised](decisions/archive/phase1a-gate-closure-revised.md) §2a

---

### Profile Attendance Field

**Ruling:** `attendance` is a top-level enum on the profile schema: `attended | unattended`. Orthogonal to `context` (execution host) and `approval.scope` (permitted classifications). Controls approval gate selection only.

Profile schema shape confirmed:
```yaml
context: vscode-operator      # execution HOST
attendance: unattended         # approval semantics
approval:
  scope:
    allow_read: true
    allow_mutating: false
    allow_destructive: false
```

**Rationale:** Separates execution host from human presence; an autonomous agent inside VS Code is unattended despite the host being `vscode-operator`.

→ [Final Acknowledgment](decisions/archive/phase1a-final-acknowledgment.md) §2; [Gate Closure Revised](decisions/archive/phase1a-gate-closure-revised.md) §2b

---

### `Attended *bool` on WireOptions

**Ruling:** Add `Attended *bool` to WireOptions. When the profile declares `attendance:`, set it explicitly. When absent, derive from TTY detection (or keep legacy `TTYOutput` default). Gate selection: profile value wins over TTYOutput inference when non-nil.

**Rationale:** TTYOutput currently drives three distinct things — (a) approval gate selection, (b) stdin/stdout for collector/choice steps, (c) terminal prompt rendering. Profile `attendance` replaces ONLY (a). Physical IO wiring for (b) and (c) remains driven by TTYOutput.

→ [Final Acknowledgment](decisions/archive/phase1a-final-acknowledgment.md) §2; [Final Impact Assessment](decisions/archive/phase1a-don-final-impact-assessment.md) Q1

---

### `config/attendance-mismatch` Tier 0 Error

**Ruling:** Profile declares `attended` but no TTY is available → Tier 0 failure with code `config/attendance-mismatch`. A profile promising an approver who cannot be reached is a configuration error.

→ [Final Acknowledgment](decisions/archive/phase1a-final-acknowledgment.md) §2

---

### TTYOutput Hardcode Bug (Live Latent Issue)

**Ruling (disclosure):** `cmd/gert/run.go` hardcodes `TTYOutput: true` unconditionally — no `isatty()` detection. Consequence: `gert run` in a CI pipeline blocks on stdin if any approval fires. Profile-declared `attendance: unattended` is the fix. The bug is documented; addressed by the `Attended *bool` mechanism above.

→ [Gate Closure Revised](decisions/archive/phase1a-gate-closure-revised.md) §2b; [Final Impact Assessment](decisions/archive/phase1a-don-final-impact-assessment.md) Q1

---

### `UnattendedApprovalGate` — Phase 1 Minimal Gate

**Ruling:** New gate type denies all `mutating`, `destructive`, and `unspecified` actions in unattended contexts. Phase 1 scope: fail-closed denial. Phase 3 scope: full evidence-recording gate with `PolicyID`, `RunID`, `StepID`, `RetryCount`, `Classification` on `ApprovalRecord`.

**Rationale:** `ApprovalRecord` extension is backward-compatible (additive `omitempty` fields). Phase 1 needs fail-closed; full audit trail follows in Phase 3.

→ [Gate Closure Revised](decisions/archive/phase1a-gate-closure-revised.md) §2b; [Implementation Impact](decisions/archive/phase1a-don-implementation-impact.md) §C

---

### Approval Gate on Direct-Invocation Path

**Ruling:** The `ApprovalGate` currently fires ONLY on the substitution path (`executor/tool.go:executeSubstitution()`). The direct-invocation path (`Execute()` → `runtime.Invoke()`) has no approval check. Phase 1 adds classification-aware approval check to `Execute()` so ALL tool invocations — substituted and direct — pass through the gate.

→ [Gate Closure Revised](decisions/archive/phase1a-gate-closure-revised.md) §2b; [Implementation Impact](decisions/archive/phase1a-don-implementation-impact.md) §B, §C

---

### `AllowedEnvironments` — Profile-Context Allowlist

**Ruling:** `AllowedEnvironments` on `ToolGovernance` becomes the profile-context allowlist. Values are the canonical context vocabulary: `cli-operator`, `vscode-operator`, `ci`, `headless-server`, `test`. Matching is exact, case-sensitive against the runtime profile's `context:` field. Empty = all contexts allowed (backward compatible).

→ [Gate Closure Revised](decisions/archive/phase1a-gate-closure-revised.md) §2c; [Implementation Impact](decisions/archive/phase1a-don-implementation-impact.md) §A

---

### `AllowedModes` — RunMode Field (New)

**Ruling:** New field `AllowedModes []string` on `ToolGovernance` (`yaml:"allowed-modes,omitempty"`) for the RunMode dimension. Carries `real | dry-run | replay` vocabulary. Empty = all modes allowed. Migration: the 25 existing `allowed-environments: ["real"]` fixtures in `tv-enum.yaml` → `allowed-modes: ["real"]`. This is a test-data-only change; no runtime breakage (field was never enforced).

**Rationale:** `"real"` in the conformance fixtures is a RunMode indicator (maps to `engine.RunModeReal`), not a deployment context. Overloading one field for both axes would silently destroy the existing dry-run safety intent.

→ [Gate Closure Revised](decisions/archive/phase1a-gate-closure-revised.md) §2c; [Implementation Impact](decisions/archive/phase1a-don-implementation-impact.md) §A

---

### `RequiresCapabilities` — Deferred

**Ruling:** Do NOT enforce or populate `RequiresCapabilities` until the profile schema defines what a "capability" is. Zero values exist anywhere in the codebase; the vocabulary is undefined. Leave for post-Phase 1 design.

→ [Gate Closure Revised](decisions/archive/phase1a-gate-closure-revised.md) §2c; [Implementation Impact](decisions/archive/phase1a-don-implementation-impact.md) §A

---

### Test-Context Transport Rules

**Ruling:**

| Transport mode | In test context | Override |
|---|---|---|
| `native` | Always allowed | — |
| `mcp` (subprocess) | Blocked by default | Allowed with `transport.allow_subprocess_in_test: true` |
| `mcp-http` | Never allowed | No override |

The subprocess opt-in is an explicit, named, auditable author assertion — not a sandbox. HTTP endpoints are absolutely banned in test context.

→ [Final Acknowledgment](decisions/archive/phase1a-final-acknowledgment.md) §3; [Final Impact Assessment](decisions/archive/phase1a-don-final-impact-assessment.md) Q2

---

### Transport Mode — Profile Does Not Override

**Ruling:** The profile does NOT override a tool's declared transport mode. Switching transport mode changes operational semantics and is not a parameter substitution. Correct mechanisms: (1) the package provides multiple tool definitions (one per context), selected via `--package-map`; (2) the resolver matches on the tool's declared mode and provides parameters (endpoint URL, auth config) for that mode only.

→ [Conditional Acceptance](decisions/archive/phase1a-conditional-acceptance.md) §D

---

### `--profile` / `--package-map` Composition Precedence

**Ruling:** `--package-map` wins at the YAML-selection layer (which package/tool definition backs each toolRef). `--profile` wins at runtime parameterization (auth, endpoint, approval policy). Resolver MUST consume `plan.Tools` (post-catalog, post-package-map resolved tool set) — NOT re-resolve toolRefs independently. If `--package-map` selects a `native` mock while the profile expects `mcp-http`, the Tier 0 check surfaces the mismatch before execution.

→ [Gate Closure Revised](decisions/archive/phase1a-gate-closure-revised.md) Binding Model; [Conditional Acceptance](decisions/archive/phase1a-conditional-acceptance.md) §D, §E

---

### Per-Tool Profile Overrides

**Ruling:** Profiles allow per-tool binding overrides (endpoint URL, and in Phase 3, auth). Phase 1 implements endpoint-only overrides with inherited profile-level auth. Per-tool auth override is Phase 3 scope.

→ [Conditional Acceptance](decisions/archive/phase1a-conditional-acceptance.md) §D

---

### Tiered Preflight Design

**Ruling:**
- **Tier 0** (static/offline, mandatory before every run, zero I/O): toolRef resolution, binding structural completeness, env var interpolation, `AllowedEnvironments` match. Cannot be skipped.
- **Tier 1** (local/no-network, opt-in `--preflight=local`): PATH checks, IMDS reachability.
- **Tier 2** (live/network, opt-in `--preflight=live`): token acquisition, endpoint probe. Adds to Tier 0, never replaces it.

Five error codes — Tier 0: `config/tool-unresolved`, `config/no-binding-for-profile`, `config/binding-incomplete`; Tier 2: `auth/credential-failure`, `transport/endpoint-unreachable`. Static-vs-dynamic distinction must be explicit in message text.

`config/no-binding-for-profile` message must include: tool name, transport mode, profile ID, AND "note: this runbook runs correctly in profiles: X, Y" (computable at Tier 0).

→ [Conditional Acceptance](decisions/archive/phase1a-conditional-acceptance.md) §C

---

### `gert plan` Command

**Ruling:** `gert plan --profile <id> <runbook>` ships early Phase 1 (2–3 days). Uses existing `internalplanner.Plan()` output. Labeled: "binding compatibility report — shows whether each tool's declared transport and environment constraints are compatible with the selected profile." Does NOT validate endpoint reachability or auth token acquisition. Exit codes: 0 = Tier 0 clean; 1 = Tier 0 failures; 2 = Tier 2 failures (only with `--preflight=live`). `--output=json` for CI.

`gert plan --show-profiles <runbook>`: zero matches → exit 0, empty stdout, single-line stderr: "no profile in <search-paths> provides a complete binding for all toolRefs in <runbook>."

→ [Gate Closure Revised](decisions/archive/phase1a-gate-closure-revised.md) `gert plan`; [Conditional Acceptance](decisions/archive/phase1a-conditional-acceptance.md) §C; [Implementation Impact](decisions/archive/phase1a-don-implementation-impact.md) §D

---

### Late-Result Classification-Aware Semantics

**Ruling:**

| Classification | On late/lost result | Runbook behavior |
|---|---|---|
| `read-only` | Discard, log warning | Continue (or retry if idempotent) |
| `mutating` | INDETERMINATE | Halt. Operator must verify before resuming. |
| `destructive` | INDETERMINATE | Halt. Operator must verify before resuming. |

**Rationale:** Discarding late results for mutating/destructive actions creates a false picture of system state and enables dangerous double-execution.

→ [Conditional Acceptance](decisions/archive/phase1a-conditional-acceptance.md) §E.1

---

### Reconnect Retry — Re-Approval Required

**Ruling:** A reconnect retry is a new invocation event. Interactive contexts: re-prompt required (original approval covers a specific invocation at a specific time). Unattended contexts: retry must be recorded in approval evidence including original `run_id` and retry flag.

→ [Conditional Acceptance](decisions/archive/phase1a-conditional-acceptance.md) §E.2

---

### Phase 1 Halt-on-Timeout (Option A)

**Ruling:** Phase 1 commits to halt-on-timeout with no-retry policy. Timeout → typed `transport/timeout` error. For mutating/destructive actions, error message includes: "This action's completion state is unknown. Manual verification is required before retrying." No automatic retry, no reconnect. Runbook is terminal.

**Rationale:** Leaves Phase 1-2 window safe while full reconnect/idempotency semantics (Phase 3) are being built.

→ [Conditional Acceptance](decisions/archive/phase1a-conditional-acceptance.md) §E.3

---

### Host Bridge Protocol Requirements (Before Phase 2)

**Ruling:** The following are REQUIRED before Phase 2 implementation begins:
- `protocol_version` field on the registration handshake; version negotiation or hard-fail semantics.
- Host capability advertisement at registration: `{adapter_id, protocol_version, tools: [{tool_name, actions: [string]}]}`.
- Correlation IDs: `run_id` and `step_id` on `ToolRequest` (in addition to `request_id`).
- `CancelRequest { request_id }` message type on cancel channel.
- Framing protocol document: message envelope format, all message types with `message_type` discriminator, length-prefix or NDJSON framing, ordering guarantees, malformed-message handling.

→ [Conditional Acceptance](decisions/archive/phase1a-conditional-acceptance.md) §F

---

### Profile File Location Precedence

**Ruling:** Profiles discovered in order: (1) CLI flag `--profile <path>` (highest); (2) repo-local `.gert/profiles/<id>.yaml`; (3) user config `~/.config/gert/profiles/<id>.yaml`. First match wins. Consistent with `.gert/config.yaml` and `--package-map` precedence.

→ [Conditional Acceptance](decisions/archive/phase1a-conditional-acceptance.md) §G OQ1

---

### Profile ID Format

**Ruling:** `[a-z0-9][a-z0-9-]*`, max 64 characters. No uppercase, no spaces, no special characters beyond hyphen. Leading digit allowed. Required for shell quoting safety, tab-completion, and filename-stem use.

→ [Slices Gate Review](decisions/archive/phase1a-slices-gate-review.md) Ratification §4

---

### OQ2 Closed — Subprocess with IPC

**Ruling:** VS Code extension uses a Gert subprocess. Subprocess with IPC is the architecture (fault isolation: extension crash must not kill a running Gert execution). Binding decision required before Phase 2 from SQL Live-Site Ops. Phase 0 spike confirms.

→ [Final Acknowledgment](decisions/archive/phase1a-final-acknowledgment.md) §4; [Conditional Acceptance](decisions/archive/phase1a-conditional-acceptance.md) §G OQ2

---

### OQ5 Closed — AuthProvider Deadline Propagation

**Ruling:** `AuthProvider.Token(ctx context.Context)` already propagates deadlines. `AzureCLIAuthProvider.acquire()` passes context to `exec.CommandContext(ctx, ...)`. Managed identity and workload identity providers must follow the same pattern. No spike needed.

→ [Conditional Acceptance](decisions/archive/phase1a-conditional-acceptance.md) §A.6

---

### Auth Token Caching

**Ruling:** All auth providers must follow the `AzureCLIAuthProvider` caching pattern: in-memory caching with proactive refresh at 5-minute-before-expiry buffer, fallback to still-valid cached token if early refresh fails. Per-call acquisition is unacceptable (IMDS: 200ms+ per call).

→ [Conditional Acceptance](decisions/archive/phase1a-conditional-acceptance.md) §G OQ3

---

### No Profile Inheritance

**Ruling:** No profile inheritance. Profiles are flat. Per-tool overrides within a profile provide composability. Profile inheritance complexity (merge semantics, conflict resolution, circular inheritance) is not justified by the use case.

→ [Conditional Acceptance](decisions/archive/phase1a-conditional-acceptance.md) §G OQ4

---

### Additive Schema Extensions (No Breaking Changes)

**Ruling:** All schema changes are ADDITIVE only. New fields are optional with safe defaults that do not break existing tool definitions. Existing tool definitions continue working unchanged.

| Field | Schema location | Default |
|---|---|---|
| `allowed-environments` | `tool/v1` governance block | `[]` (all contexts allowed) |
| `allowed-modes` | `tool/v1` governance block | `[]` (all modes allowed) |
| `classification` | Per-action on ToolAction | nil/unspecified |
| `idempotent` | Per-action on ToolAction | `false` |

→ [Conditional Acceptance](decisions/archive/phase1a-conditional-acceptance.md) §B

---

### `idempotent` Field on ToolAction

**Ruling:** New optional field `idempotent: bool` on ToolAction, default `false`. Controls retry/reconnect policy. Unspecified actions must never get automatic retry even when a retry infrastructure is later added (constraint baked in from the start).

→ [Conditional Acceptance](decisions/archive/phase1a-conditional-acceptance.md) §B

---

### PKG-010 for Classification Validation — Ratified

**Ruling:** Classification validation errors (invalid `classification` value) use the existing `PKG-010` error class. Error message includes: invalid value + allowed values list + action name. No new error class needed. The conformance vector GOV-014 correctly uses `PackageExpected` shape with `error.code: PKG-010`.

→ [Slices Gate Review](decisions/archive/phase1a-slices-gate-review.md) Ratification §1

---

### TV-CONFORM-GOV-004 — Must Not Gain Approval Bypass

**Ruling:** The conformance harness MUST NOT gain an approval-bypass mechanism (safety invariant violation). TV-CONFORM-GOV-004 must be un-skipped or recast to expect the specific observable result (pass under NoOpApprovalGate, or correct profile-based denial). Owner: Tess. Due: end of Phase 1.

→ [Slices Gate Review](decisions/archive/phase1a-slices-gate-review.md) Ratification §2

---

### Integration-Test Reachability Convention

**Ruling:** Every enforcement function must have a companion integration test exercised via the CLI entry point (`TestCLI_<Feature>_Reachable`). Unit tests prove correctness; integration tests prove reachability. If the function is unreachable, the test fails immediately. Adopted for Phase 2+.

**Rationale:** Two instances of "checks wired to nothing from the CLI entry point" occurred in Phase 1A itself (`AllowedEnvironments` ghost fields; David's Tier 0 preflight unreachable from `gert run` until d7b77c1).

→ [Slices Gate Review](decisions/archive/phase1a-slices-gate-review.md) Systemic Process Recommendation

---

### Phase Agreement (Summary)

**Ruling:** Phase timelines agreed:
- **Early win:** AllowedEnvironments enforcement, AllowedModes field + fixture migration, approval gate on direct-invocation path.
- **Phase 0 (2 weeks):** Profile spec, host architecture decision, framing protocol, OQ2 binding decision.
- **Phase 1 (6–8 weeks):** Binding resolver, tiered preflight, `gert plan`, managed identity, classification/idempotent schema, ProfileApprovalGate, halt-on-timeout, ICM proof.
- **Phase 2 (re-estimated after Phase 0):** Host bridge + VS Code adapter (scoped after OQ2).
- **Phase 3 (4–6 weeks):** Workload identity, reconnect/idempotency, full approval evidence.

Not in scope: device code auth (§8.4 deferred), profile inheritance (rejected), multi-cloud auth.

→ [Final Acknowledgment](decisions/archive/phase1a-final-acknowledgment.md) §5; [Gate Closure Revised](decisions/archive/phase1a-gate-closure-revised.md) Phase Agreement

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

