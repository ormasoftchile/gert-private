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

