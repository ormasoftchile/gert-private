# Gert Core — Phase 1 Status Report to SQL Live-Site Operations

**From:** Gert Core Team (Barbara, Lead/Architect)  
**Date:** 2026-08-17  
**Status:** Phase 1 code-complete. All tests green. Chapter 17 specification shipped.

---

## 1. What Shipped

Phase 1 implements the governance and profile foundation negotiated across our six-round exchange. All items below are on `main`, build-clean, test-green.

| Capability | Commits | Territory |
|---|---|---|
| Tri-state `RequiresApproval *bool`, per-action `Classification *string`, `AllowedModes` | a2e7db0 | pkg/schema/tool.go |
| GovernanceEvaluator wired in all production paths | c810b96 | internal/adapter/wire.go, pkg/run/run.go, engine Start/Resume |
| Runtime profile schema, loader, `--profile` flag | 773f332 | pkg/schema/profile.go, cmd/gert/run.go |
| ProfileApprovalGate + declared attendance | ff429d4, d7b77c1, b420dda | internal/governance/profile_evaluator.go, wire.go |
| Tier 0 static preflight (PLAN-010/011/012) | b982804, d7b77c1 | internal/planner/preflight.go |
| `gert plan --profile`, `--show-profiles`, binding table | 2968006 | cmd/gert/plan.go |
| Classification validation at parse time (PKG-010) | 7c7402d | internal/tool/scan.go |
| Fixture migration (`allowed-environments: ["real"]` → `allowed-modes`) | Tess | internal/conformance/enumdata/tv-enum.yaml |
| Conformance corpus: 72 vectors (62 pass, 10 skipped with named reasons, 0 fail) | Tess | |
| Orthogonality guard: evaluator-level coercion probe | ed432c7 | internal/engine/profile_approval_test.go |
| Chapter 17 specification (LaTeX) | 0c96e21, 48fd214 | design/gert/sections/17-runtime-profiles.tex (in `gert-private` repo, not the code repo) |

---

## 2. Your Four Validation Points

You specified that implementation validation must demonstrate these four properties. Here is the evidence for each.

### 2.1 GovernanceEvaluator is wired in production

The evaluator is constructed per-run in `engine.Start()` and `engine.Resume()` from `plan.GovernanceSource`. It is wired unconditionally in `internal/adapter/wire.go:161` (main engine) and `:392` (sub-engine), plus both entry points in `pkg/run/run.go`. There is no nil-check guard — the evaluator is always present.

### 2.2 Policy composes runbook and per-tool governance

`BuildEvaluator` (internal/governance/builder.go) is variadic: it accepts the gate plus any number of governance configs. The engine populates `StepInfo.ToolApprovalTriState` directly from `toolDef.Governance.RequiresApproval` (engine.go:529) and `StepInfo.ToolClassification` from `action.Classification` (engine.go:536). The evaluator's decision combines the runbook-level `requireApproval` (from `GovernanceSource`) with the per-tool tri-state via monotone OR: `e.requireApproval || step.ToolRequiresApproval`. A tool-level `true` can never be suppressed by a runbook-level `false`.

### 2.3 Approval applies to substituted, stdio-MCP, and HTTP-MCP calls

The governance evaluation is a pre-dispatch chokepoint in `engine.go executeStep()` (line ~505–559). It executes BEFORE the transport-specific executor is invoked. All step types — substitution (runbook-backed), stdio-MCP (subprocess), HTTP-MCP, and native process — pass through this single gate. No transport-specific bypass path exists.

### 2.4 Approval state never changes classification, retry, or late-result behavior

**Structural guarantee:** `RequiresApproval` is a `*bool` on `ToolGovernance`; `Classification` is a `*string` on `ToolAction`. They live on different structs, are populated from different schema fields, and flow into different `StepInfo` fields. No code in the repository derives one from the other in either direction. `RetryConfig` (pkg/schema/step.go:52) carries its own `Idempotent bool` with no reference to RequiresApproval. The 401 retry in `internal/tool/mcp_http.go:284–296` is token refresh only.

**Behavioral guard (ed432c7):** Two tests (`TestApproval_LegacyOptOut_ClassificationNotCoerced_Destructive` and `_Mutating`) exercise the ProfileEvaluator with `ToolApprovalTriState=&false` + an explicit destructive/mutating classification, under a profile with `allow_read: false`. If the evaluator ever coerces classification to `read-only`, the read-only matrix row denies the step and the test fails. The test also asserts `ToolClassification` is unmutated after evaluation.

**Conformance guard:** Vectors GOV-009 and GOV-010 pin that `requires-approval: false + classification: destructive` and `+ mutating` are schema-valid and execute successfully.

**Transparency note:** Our first attempt at the orthogonality guard (GOV-009/GOV-010 alone) would have passed regardless of whether coercion existed, because the conformance harness runs without a profile. Our own gate review caught this. The evaluator-level probe (ed432c7) was added specifically to close that gap — it uses an attended profile with a restrictive scope so that coercion produces a *different observable outcome* (denial vs. approval suppression).

---

## 3. The Orthogonality Guarantee

Your round-6 correction — "`RequiresApproval: false` influences approval routing ONLY; it must never assign, imply, or coerce `classification: read-only`" — is now enforced at two layers:

1. **Schema acceptance** (GOV-009/GOV-010): proves the combination is valid and the runbook executes.
2. **Routing non-coercion** (ed432c7): proves the evaluator does not silently reroute classification when the legacy opt-out fires.

The ProfileEvaluator's legacy opt-out path is:
```go
if step.ToolApprovalTriState != nil && !*step.ToolApprovalTriState {
    result.RequiresApproval = false
    return result, nil
}
```
It sets `RequiresApproval = false` and returns. It does not read, write, or reference `ToolClassification`. The classification matrix is never entered.

`legacy_unspecified_policy: allow` suppresses the approval prompt only. It sets `result.RequiresApproval = false` within the `default` (unspecified) matrix branch. It does not assign a classification, does not affect retry eligibility, and does not change late-result handling.

---

## 4. Known Limitations

These are genuine gaps. Do not build against them as if they were implemented.

| Limitation | Scope | Mitigation path |
|---|---|---|
| **Profileless CI hang hazard.** `cmd/gert/run.go` hardcodes `TTYOutput: true`. Without `--profile`, `TerminalApprovalGate` is installed even in non-interactive contexts. If a tool declares `requires-approval: true`, the process blocks on stdin and exits 3. | Profileless runs only. Any run with `--profile <unattended>` is safe — declared attendance selects NoOpApprovalGate. | Pending: `isatty()` detection seam or an explicit `--unattended` flag. Tier 1. |
| **Test-context subprocess is not sandboxed.** `allow_subprocess_in_test: true` is an auditable author assertion. The subprocess inherits the full parent environment via `os.Environ()`. No network isolation, no filesystem jail, no seccomp. | Test context only. | By design. Documented in all error messages and Chapter 17. Gert performs no OS-level isolation. |
| **`RequiresCapabilities` is specified but not enforced.** The field exists on the schema; no runtime code reads it. | All contexts. | Phase 3 scope (host bridge + capability negotiation). |
| **Per-tool auth override is Phase 3.** Phase 1 supports per-tool endpoint override only; auth is always inherited from the profile's top-level auth configuration. | Profiles with heterogeneous auth requirements per tool. | Phase 3: `tools.<tool-id>.auth:` block in the profile schema. |
| **`isatty()` does not exist in this codebase.** Attendance check (PLAN-011) is against declared state only. A process declaring `attendance: attended` in a headless environment is not detected at runtime. | All contexts without a profile. | Tier 1. |
| **`AllowedEnvironments` is enforced (PLAN-010). `AllowedModes` is declared only** — no runtime code checks it against the active RunMode yet. | Tools relying on AllowedModes for mode restriction. | Near-term; the check is trivial once RunMode is accessible at plan time. |

---

## 5. Context Vocabulary and Profile ID Format

These are now documented in Chapter 17 (`design/gert/sections/17-runtime-profiles.tex` in the `gert-private` governance repo — not the `gert` code repo) and enforced in code.

### Profile contexts (closed enum, `AllowedEnvironments` values)

| Value | Meaning |
|---|---|
| `cli-operator` | Human at a terminal running `gert run` directly |
| `vscode-operator` | Human in VS Code; Gert invoked via extension |
| `ci` | Automated CI/CD pipeline; no interactive channel |
| `headless-server` | Long-running server process; no operator present |
| `test` | Conformance/integration test; deterministic behavior required |

A tool's `governance.allowed-environments` list restricts which profile contexts may invoke the tool. If the active profile's context is not in the list, PLAN-010 fires at plan time before any step executes.

Non-canonical values in `allowed-environments` (e.g., `"real"`, `"dry-run"`) are silently ignored during the migration window — they are RunMode discriminators being migrated to `allowed-modes` and do not restrict profile context.

### Profile ID format

`[a-z0-9][a-z0-9-]*`, maximum 64 characters. Profiles are stored as `<id>.yaml`. The constraint is enforced at parse time; an invalid ID produces a validation error.

### Attendance (orthogonal to context)

| Value | Meaning |
|---|---|
| `attended` | An operator is present and can respond to approval prompts |
| `unattended` | No interactive channel; approval must be pre-authorized or denied |

Attendance is declared on the profile, not inferred from the terminal. `attended + ci` is rejected at plan time (PLAN-011) because CI is structurally unattended.

---

## 6. Phase 2 Prerequisites

Phase 2 (host tool bridge + VS Code adapter) depends on one resolved decision and one Phase 0 spike:

**Resolved:** The extension retains subprocess isolation. Gert runs as a subprocess with versioned correlated IPC. VS Code APIs and auth (including `vscode.lm.invokeTool`) remain in the extension process. The extension translates between the IPC protocol and host-native capabilities. This was your stated preference (OQ-2 response) and we adopted it.

**Phase 0 spike required:** Define the IPC message schema for the host tool bridge — specifically the tool-call request/response envelope, capability advertisement, and the cancellation/deadline propagation contract. This spike should produce a schema file that both teams can implement against independently.

---

## Summary

Phase 1 is complete. The four properties you named are demonstrated in code and tested. The orthogonality guarantee from your round-6 correction is enforced at two independent layers. The context vocabulary and profile ID format are documented and ready for your team to populate tool declarations.

Known limitations are stated above as-is. Build against the "enforced" column, not the "specified" column, until we report otherwise.

We will notify you when Phase 2 begins. If you need anything clarified before then, we are available.

— Barbara, Gert Core Team
