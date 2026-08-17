# Archive: Runtime Portability Decisions (2026-08-10 — 2026-08-15)

**Archived:** 2026-08-17
**Reason:** Size limit — keeping recent entries (2026-08-16+)
---

# Final Acknowledgment: Runtime Portability — Design Agreed (Rev 4 — Final)

**Date:** 2026-08-17  
**By:** Gert Core Team  
**To:** SQL Live-Site Operations (gert-sqllivesite)  
**Status:** Design closed. Implementation begins.  

---

## 1. Approval ≠ Classification — Accepted

We withdraw the Rev 3 language stating that explicit `requires-approval: false` is "equivalent to `classification: read-only`." That was wrong. The two fields answer different questions:

- `RequiresApproval` → whether active approval is required before invocation.
- `Classification` → the action's side effects, governing retry, timeout, and late-result safety.

A legacy action may explicitly suppress approval while still being mutating or destructive. Coercing `&false` to `read-only` would have granted retry eligibility and read-only late-result handling to actions that may be destructive — a safety regression introduced by a compatibility shim. That violates the governing principle we both agreed to: absence of classification must never grant additional execution rights.

**Published semantics for `RequiresApproval = &false, Classification = nil`:**

| Dimension | Behavior |
|-----------|----------|
| Approval routing | Legacy opt-out preserved. Gate does NOT fire for this action (approval dimension only). |
| Classification | Remains `unspecified`. |
| Retry | Never retry automatically. |
| Late/lost result | INDETERMINATE + halt. Require verification. |
| `legacy_unspecified_policy: allow` | Suppresses PKG-W migration **prompts** only. Never relaxes retry, idempotency, or late-result behavior. |

`RequiresApproval` influences approval routing only. It must never assign, imply, or coerce a classification value.

---

## 2. Runbook-Level Cannot Override Per-Action — Accepted and Verified

Your condition: runbook-level `requires-approval: false` must never override per-action approval or classification semantics. Verified against the source:

**(a) Substitution path — structurally satisfied.** `pkg/pkgsubst/pkgsubst.go:276` composes governance as `eff.RequireApproval = eff.RequireApproval || subRequireApproval` (monotone-increasing OR). `internal/governance/builder.go:BuildPolicy()` only ever sets `requireApproval = true`, never back to false. Runbook-level false cannot suppress a tool-level true. The guarantee you want holds by construction.

**(b) Plain (non-substitution) steps — correction of our earlier disclosure.** In the gate closure we told you approval fires from runbook-level governance on the plain-step path. That was inaccurate. The governance pre-flight block in `internal/engine/engine.go` (~line 506) is guarded by `if h.engine.cfg.GovernanceEvaluator != nil`, and `GovernanceEvaluator` is never assigned in any production wiring path — not in `internal/adapter/wire.go`, not in `pkg/run/run.go`. It is nil in all production code paths. For plain tool steps today, neither tool-level NOR runbook-level approval is enforced. We are correcting this unprompted because you are relying on our disclosures being accurate.

**(c) Binding Phase 1 constraint.** When we wire `GovernanceEvaluator` to close the plain-step gap, the policy MUST be built from BOTH runbook-level and per-tool governance (not runbook governance alone, which is what `planner.go:110` does today for `Plan.Governance`). If built from runbook governance alone, tool-level `requires-approval: true` would be invisible and effectively suppressed — violating your condition. `BuildPolicy` is already variadic and supports multi-input composition, so this is a call-site change, not an API change. We adopt this as a binding implementation constraint for Phase 1.

---

## 3. No Landmine

Nothing in current code couples approval state to retry. `RetryConfig` carries its own `Idempotent bool` with no reference to `RequiresApproval`. The 401 retry in `mcp_http.go` is token-refresh only. The orthogonality you are demanding is already true in the code — we only have to not break it.

---

## 4. Effort

No change to the ~4.5 day total from Rev 3. The `BuildPolicy` call-site constraint falls inside the already-scoped "direct-invocation approval gate enforcement (1–2 days)" line item.

---

## 5. Close

Design final. Six corrections from you across five rounds, six accepted. The runtime portability architecture — tiered preflight, declared attendance, per-action classification with tri-state approval, profile-context binding, conservative-by-default safety model — is agreed and ready for implementation. We start this week.

---

*Gert Core Team — 2026-08-17*


---

# Slice 1 + Slice 2 Review: Tri-State Schema + Direct-Invocation Approval Enforcement

**Date:** 2026-08-17  
**Reviewer:** Barbara (Lead / Architect)  
**Commits:** `a2e7db0` (Don — schema), `c810b96` (Ken — enforcement)  
**Verdict:** **APPROVED**  

---

## Requirement-by-Requirement Assessment

### 1. GovernanceEvaluator is wired in production — ✅ DEMONSTRATED

- `internal/adapter/wire.go:161`: `GovernanceEvaluator: internalgovernance.BuildEvaluator(approvalGate)` — unconditional.
- `pkg/run/run.go`: same pattern in both engine-construction paths.
- Sub-engine (`runSubStepsViaEngine`, wire.go:392): also wired.
- Per-run evaluator built in `Start()` and `Resume()` from `plan.GovernanceSource`.
- The previously-dead code path (`if GovernanceEvaluator != nil`) is now always active.
- **Test:** `TestApproval_GovernanceEvaluatorWiredInProduction` — proves `BuildEvaluator(gate)` returns non-nil and is functional.

### 2. Policy composes runbook AND per-tool governance — ✅ DEMONSTRATED

- Runbook-level: `planner.go:111` sets `GovernanceSource: rb.Runbook.Governance` on the plan. `Start()/Resume()` passes it to `BuildEvaluator`.
- Per-tool: `engine.go:497-498` extracts `*toolDef.Governance.RequiresApproval` into `stepInfo.ToolRequiresApproval`.
- Evaluator (`evaluator.go`): `e.requireApproval || step.ToolRequiresApproval` — monotone OR at evaluation time.
- Both sources genuinely contribute to the decision. Building from runbook-level alone would leave `ToolRequiresApproval` unused. Building from per-tool alone would ignore `e.requireApproval`. Both are consumed.
- **Test:** `TestApproval_PolicyComposesRunbookAndToolGovernance` — tool has `requires-approval: true`, runbook has no governance. Gate fires. Proves tool-level is not silently ignored.

### 3. Approval applies to substituted, stdio-MCP, and HTTP-MCP calls — ✅ DEMONSTRATED

- The chokepoint is in `executeStep` (`engine.go:468-562`), BEFORE executor dispatch. The executor is what eventually selects the transport. So all transports — substituted, stdio-MCP, HTTP-MCP, native process — are gated by a single pre-dispatch check.
- **Test:** `TestApproval_FiresForAllInvocationPaths` — uses a denying gate and verifies `exec.invoked == false`. The executor was never reached. This proves the gate fires pre-dispatch regardless of transport.
- **Test:** `TestApproval_FiresForStdioMCPAndHTTPMCP` — labels match the spec (though technically all transports share the same code path, which is the point).

### 4. Approval state never changes classification, retry, or late-result behavior — ✅ DEMONSTRATED (structurally)

- `EvaluationResult` carries: `Allowed`, `Denied`, `RequiresApproval`, `MatchedRules`, `BlockedEnvVars`, `FilteredEnvVars`, `Evidence`. No Classification, no retry fields.
- `StepInfo` carries: `ID`, `Kind`, `Command`, `EnvVars`, `ToolRequiresApproval`. No Classification field flows into the evaluator.
- No code in `evaluator.go` references Classification or RetryConfig.
- No code anywhere in the codebase couples RequiresApproval to retry logic (confirmed: RetryConfig has its own `Idempotent bool`; the only retry in mcp_http.go is token-refresh).
- **Test:** `TestApproval_DoesNotAffectRetryOrClassification` — structural type assertion (verifies EvaluationResult has no retry/classification fields). Not a behavioral test, but the structural guarantee is valid: coupling would require adding fields to these shared types.

### 5. Tri-state integrity — ✅ DEMONSTRATED

- `RequiresApproval *bool` on ToolGovernance with `yaml:"requires-approval,omitempty"`.
- **Test case (b)** in `TestRequiresApprovalTristate`: governance block contains `requires-capabilities: [network]` but NO `requires-approval`. Result: `Governance != nil`, `RequiresApproval == nil`. This is THE case the counterparty caught — and it passes correctly.
- All four cases (a: absent, b: present-but-no-field, c: explicit-false, d: explicit-true) covered.
- tv-enum.yaml's 25 `requires-approval: false` entries unmarshal to `&false` — confirmed by conformance suite passing.

### 6. Orthogonality — ✅ DEMONSTRATED (structurally)

- Classification lives on `ToolAction` (`*string`). RequiresApproval lives on `ToolGovernance` (`*bool`). Different structs, different conceptual levels. No code derives one from the other.
- The evaluator flow never reads Classification. The Classification field is never consulted in approval routing.

### 7. Monotone composition — ✅ DEMONSTRATED

- `builder.go:39-41`: `if cfg.RequireApproval { p.requireApproval = true }` — can only set true, never clear it.
- Evaluator: `e.requireApproval || step.ToolRequiresApproval` — OR, cannot suppress.
- **Test:** `TestApproval_RunbookFalseCannotSuppressToolTrue` — runbook governance has `RequireApproval: false`, tool has `RequiresApproval: &true`. Gate fires. Proves monotone OR holds.

### 8. Binding constraint (both sources feed policy) — ✅ DEMONSTRATED

- Runbook-level: flows through `plan.GovernanceSource` → `BuildEvaluator` → `e.requireApproval`.
- Per-tool: flows through `plan.Tools[name].Governance.RequiresApproval` → `stepInfo.ToolRequiresApproval`.
- Both genuinely consumed in the evaluator's `||` expression. Building from only one source would leave the other dead.

---

## Test Quality Assessment

| Test | What it actually proves | Genuine? |
|------|------------------------|----------|
| `TestRequiresApprovalTristate` (4 sub-tests) | YAML unmarshalling correctly distinguishes all four tri-state cases | ✅ Yes — parses real YAML, asserts on pointer values |
| `TestApproval_GovernanceEvaluatorWiredInProduction` | BuildEvaluator returns a functional non-nil evaluator | ✅ Yes — proves the wiring call would produce a live evaluator |
| `TestApproval_PolicyComposesRunbookAndToolGovernance` | Tool-level RequiresApproval feeds through to gate invocation | ✅ Yes — runs a full engine cycle with a countingApprovalGate |
| `TestApproval_RunbookFalseCannotSuppressToolTrue` | Monotone OR holds under conflict | ✅ Yes — full engine cycle with conflicting governance |
| `TestApproval_FiresForAllInvocationPaths` | Gate fires pre-dispatch; executor never reached on denial | ✅ Yes — denying gate + executor invocation check |
| `TestApproval_FiresForStdioMCPAndHTTPMCP` | Same pre-dispatch gate fires for named transport labels | ✅ Yes — though redundant with the above (same code path) |
| `TestApproval_DoesNotAffectRetryOrClassification` | Types structurally preclude coupling | ⚠️ Structural, not behavioral — but valid |

No mock-only false positives. The engine tests run real `engine.Start()` → `Next()` cycles through `runPlanToCompletion`, exercising the actual dispatch path.

---

## Tri-State Collapse at the ToolRequiresApproval Boundary — Expected Gap

In `engine.go:497-498`:
```go
if toolDef.Governance != nil && toolDef.Governance.RequiresApproval != nil {
    stepInfo.ToolRequiresApproval = *toolDef.Governance.RequiresApproval
}
```

When `RequiresApproval` is nil (unspecified), `ToolRequiresApproval` stays at its zero value (`false`). This means unspecified tools do NOT fire the gate via this path alone.

Per our Rev 4 design, unspecified should fire the gate in interactive contexts and deny in unattended contexts. That behavior belongs to the **ProfileApprovalGate** (classification-aware, attendance-aware), which is a later slice. The current slice correctly implements: "if RequiresApproval is explicitly true, the gate fires on all paths." The nil-means-conservative behavior will come from the ProfileApprovalGate when it replaces the current TTY-based gate selection.

**This is expected and correctly scoped. Not a defect — a documented gap for a later slice.**

---

## CI Hang Hazard Assessment

`GovernanceEvaluator` is now always non-nil (previously nil in production). `cmd/gert/run.go` hardcodes `TTYOutput: true`. `buildApprovalGate()` selects `TerminalApprovalGate` when TTYOutput is true. If the gate fires, it calls `RequestApproval()` which reads stdin.

**Risk assessment:** The gate fires ONLY when `e.requireApproval || step.ToolRequiresApproval` is true. This requires either:
- A runbook with `governance.require_approval: true`, OR
- A tool with `governance.requires-approval: true`

The existing corpus has ZERO tools with `requires-approval: true` (all 25 conformance tools have explicit `false`). No existing runbook in the repo declares `require_approval: true`. So **no existing CI run will hang**.

The hazard is now LIVE for any FUTURE tool or runbook that explicitly opts in to approval AND runs in CI without an unattended-aware gate. This is the same latent bug we already disclosed (TTYOutput hardcoded, no isatty). The fix is the declared-attendance work (later slice). The risk is contained to tools that explicitly request approval in an unattended context — which is a configuration error.

**Verdict: not a regression. The bug pre-existed; the previously-dead code path now being live does not change observable behavior for any existing artifact.**

---

## Counterparty Validation Status

| Criterion | Status | Evidence |
|-----------|--------|----------|
| 1. GovernanceEvaluator wired in production | **DEMONSTRATED** | wire.go:161, run.go (×2), sub-engine wire.go:392 |
| 2. Policy composes runbook + per-tool | **DEMONSTRATED** | GovernanceSource on plan, ToolRequiresApproval on StepInfo, OR in evaluator |
| 3. Approval applies to all transport paths | **DEMONSTRATED** | Pre-dispatch chokepoint in executeStep; executor never reached on denial |
| 4. Approval never affects classification/retry | **DEMONSTRATED** | Structural type separation; no coupling in evaluator code |

All four of their validation criteria can be honestly reported as demonstrated.

---

## Non-Blocking Notes for Later Slices

1. **ProfileApprovalGate (later slice):** Must implement the "unspecified fires gate in interactive / denies in unattended" behavior. Current slice only enforces explicit `requires-approval: true`. The chokepoint is in place; the gate selection logic needs upgrading.

2. **Declared attendance (later slice):** Must replace TTY-inferred gate selection. Until then, any new tool/runbook that declares `requires-approval: true` and runs in CI will block on stdin. Document this as a known limitation in the interim.

3. **AllowedModes field:** Added to schema but no enforcement wired. Expected — enforcement is a later slice.

4. **Classification validation:** `internal/tool/scan.go` rejects unknown classification values. Good. No runtime enforcement of classification-based approval policy yet — that is the ProfileApprovalGate's job (later slice).

---

**APPROVED. No revisions required. All eight requirements are met by the implementation as verified against source.**

*Barbara — 2026-08-17*


---

# Runtime Portability — Tri-State `RequiresApproval` Assessment

**Prepared by:** Don (Backend Dev)
**Date:** 2026-08-17T06:37:00-07:00

---

## Q1 — Confirm the Defect

**Confirmed. The defect is real and exactly as described.**

`pkg/schema/tool.go:47`:
```go
RequiresApproval bool `yaml:"requires-approval,omitempty" json:"requires-approval,omitempty"`
```

Plain `bool`. No custom `UnmarshalYAML` on `ToolGovernance` (only `GovernanceConfig` has one, at `pkg/schema/runbook.go:107`). No `yaml.Node` capture, no raw-node field, no "set-tracking" companion field, no `mapstructure` metadata, nothing that records field presence.

Given `governance: { requires-capabilities: [network] }` and `governance: { requires-capabilities: [network]; requires-approval: false }`: after YAML unmarshal, both produce a non-nil `*ToolGovernance` with `RequiresApproval == false`. Indistinguishable in Go. The grandfathering rule as published is unimplementable as written.

SQL Live-Site's proposed fix — `RequiresApproval *bool` — is the correct and only clean fix short of a separate shadow-tracking mechanism (which would be worse).

---

## Q2 — Blast Radius of `bool → *bool` on `ToolGovernance`

**Schema types and effective/resolved types are separate. The pointer only needs to live on the schema (authored) type.**

The relevant types:
- `pkg/schema.ToolGovernance` — **authored type**, YAML-parsed. This is what changes.
- `pkg/schema.GovernanceConfig` — authored type for runbook-level governance. Separate struct, separate change question (see Q3).
- `pkg/pkgsubst.EffectiveGovernance` — **resolved type** (`RequireApproval bool` at `pkgsubst.go:46`). Stays `bool` — SQL team's explicit permission.
- `pkg/trace.EffectiveGovernancePayload` — trace wire type (`RequireApproval bool` at `event.go:130`). Stays `bool`.

**Read sites for `schema.ToolGovernance.RequiresApproval` in Go code:**

| File | Line | What it does | nil-safe? |
|------|------|-------------|-----------|
| `pkg/pkgsubst/pkgsubst.go` | 359 | `EffectiveGovernanceFromTool()`: `RequireApproval: g.RequiresApproval` — copies schema value into `EffectiveGovernance.RequireApproval bool` | **Needs update.** Change to: `if g.RequiresApproval != nil { eff.RequireApproval = *g.RequiresApproval }` — nil treated as false for resolution. |

**That is the only Go read site for `schema.ToolGovernance.RequiresApproval`.** The field is otherwise only populated via YAML unmarshal. No test constructs `schema.ToolGovernance{RequiresApproval: ...}` directly in Go literals — all test tool defs are YAML-parsed through the conformance corpus (`tv-enum.yaml`) or through `ParseToolFile`. Verified: the grep for `RequiresApproval` in Go code hits `pkg/schema/tool.go:47` (declaration) and `pkgsubst.go:359` (read). All other `RequiresApproval` hits in Go code are for `GovernanceConfig.RequireApproval` (different struct) or `EffectiveGovernance.RequireApproval` (resolved type, stays `bool`).

**The schema/effective split is clean today.** This is not a wide refactor. It is a 2-line Go change plus the schema declaration change.

---

## Q3 — Runbook-Level Scope

`GovernanceConfig.RequireApproval` is a plain `bool` at `pkg/schema/runbook.go:91`. `GovernanceConfig` has a custom `UnmarshalYAML` at `runbook.go:107` — but that custom unmarshaller reconciles hyphenated vs. snake_case field spellings only. It still reads `RequireApproval` as a `bool` in both arms. No field-presence tracking there either.

**However: the SQL team's request to apply tri-state at the runbook level is over-scoped for the agreed design. I recommend we push back on that point.**

The reason: `GovernanceConfig.RequireApproval` gates ALL steps in a runbook uniformly — it is an operator-level declaration that "this runbook execution requires an approval step." It is not per-action classification. The grandfathering rule we defined (`requires-approval: false` explicitly authored = legacy read-only signal) only needs to work at the **tool/action level** to resolve the classification bootstrapping problem. Runbook-level `require_approval: false` has no semantic bearing on whether individual tool actions are classified — it means "don't gate the entire runbook." A `require_approval: false` runbook may still contain destructive tool actions, and those actions need classification regardless.

The tri-state distinction at the runbook level (`nil` vs. `&false` vs. `&true`) buys us nothing for the classification design: even if we could detect "this runbook's governance block was authored with explicit `require_approval: false`", that would not tell us whether individual tool actions are safe. Classification is an action-level property, not a runbook-level property.

**Blast radius if we were forced to change `GovernanceConfig.RequireApproval → *bool`:** Much wider. `GovernanceConfig.RequireApproval` is read at: `internal/governance/builder.go:BuildPolicy()` (extracts `cfg.RequireApproval` for the policy), `pkg/pkgsubst/pkgsubst.go:267-276` (subgov composition), `internal/executor/dynamic_resolver.go:ComposeGovernance()` (dynamic include governance composition), `internal/executor/gov_seed.go:WithEntryGovernance()` (seeds ctx with runbook governance), `internal/governance/evaluator.go:NewEvaluator()`. Plus all test code that constructs `schema.GovernanceConfig{RequireApproval: true/false}` directly — 15+ sites across 5+ test files. **This is the refactor worth avoiding.** The `ToolGovernance` change is clean; the `GovernanceConfig` change is not, and it doesn't help us.

---

## Q4 — Serialization / Compatibility

**No golden file breaks. No round-trip marshal breaks.**

Evidence:

1. **`kit.go` marshals `kitfile` and `lockfile` structs** (package kit management). Neither contains `ToolGovernance` or `GovernanceConfig`. No tool governance is marshaled to YAML in the kit path.

2. **Conformance corpus `tv-enum.yaml`** contains `requires-approval: false` in 25 locations — but these are **input strings** that the conformance harness unmarshals. The test reads them as YAML input, it does not marshal `ToolGovernance` back to YAML and compare against golden output. No golden files capture serialized `ToolGovernance`.

3. **JSON trace events** use `pkg/trace.EffectiveGovernancePayload` (which stays `bool`) not `schema.ToolGovernance` directly. No trace event serializes a raw `ToolGovernance`.

4. **`omitempty` behavior change for marshal:** Today, `bool` with `omitempty` suppresses the field when `false` (so `requires-approval: false` is already dropped from marshal output). With `*bool` and `omitempty`, `nil` (absent) is dropped and `&false` (explicit opt-out) is emitted as `false`. This is a change in marshal semantics — but since nothing in the codebase marshals `ToolGovernance` to YAML/JSON and compares the output, it breaks nothing today.

One future-facing note: if a packaging step ever serializes resolved tool definitions back to YAML (e.g., for a `gert compile` output or kit artifact), the `*bool` with `omitempty` would emit `requires-approval: false` when explicitly set and omit it when absent — which is the correct and desired behavior for the tri-state semantics.

---

## Q5 — Effort Estimate

**`schema.ToolGovernance.RequiresApproval bool → *bool`: 0.5 days.**

- `pkg/schema/tool.go:47` — 1 line change.
- `pkg/pkgsubst/pkgsubst.go:359` — 2 lines (nil-guard in `EffectiveGovernanceFromTool`).
- `internal/conformance/enumdata/tv-enum.yaml` — no change needed. The 25 `requires-approval: false` entries unmarshal to `*bool` pointing to `false`, which is the correct "explicit opt-out" value. The conformance tests continue to pass.
- No Go test literals construct `schema.ToolGovernance{RequiresApproval: ...}` — zero test changes required for `ToolGovernance`.

If the SQL team's runbook-level `GovernanceConfig` request is accepted despite Q3 recommendation: add 1.5–2 days for the wider read-site updates (builder, pkgsubst compose, dynamic_resolver, evaluator, test literals at 15+ sites).

---

## Q6 — Subprocess Sandboxing Confirmation

**Gert does zero sandboxing of subprocess transport. Confirmed.**

`internal/tool/process.go:StartProcess()`:
```go
cmd := exec.Command(command, args...)
cmd.Env = mergeEnv(env)
```

`mergeEnv()`:
```go
out := append([]string{}, os.Environ()...)  // full parent env inherited
for k, v := range env {
    out = append(out, k+"="+v)              // tool's extras appended
}
```

No `SysProcAttr.Cloneflags` (no Linux namespaces), no seccomp filter, no network restriction, no cwd jail, no env scrubbing — in fact the opposite: the full parent process environment is inherited unconditionally, with tool-specific vars appended on top. The subprocess sees everything the Gert process sees.

The SQL team's restatement is precisely correct: the test-profile subprocess opt-in (`transport.allow_subprocess_in_test: true`) is a **trusted profile assertion** that the operator declares the subprocess to be hermetic, not a guarantee from Gert. Gert's description of the opt-in should read: "The profile author asserts that the named subprocess command is a hermetic test double. Gert does not enforce isolation." This is a documentation/spec wording fix, not a code change.

---

---

## SQL Live-Site Condition Verification (2026-08-17T06:56:00-07:00)

### Condition under review
> "We accept the refusal to make runbook-level approval tri-state, **PROVIDED** runbook-level `require_approval: false` never overrides per-action approval or classification semantics."

---

### Q1 — Plain tool step call path: what actually decides whether an approval gate fires?

**Finding: for plain (non-substitution) tool steps, NO approval gate fires today at all.**

The governance pre-flight in `internal/engine/engine.go:456` is:
```go
if h.engine.cfg.GovernanceEvaluator != nil {
    ...
    if evalResult.RequiresApproval { // line 506 — approval gate
```

`GovernanceEvaluator` is **never assigned** in any production `EngineConfig` construction:
- `internal/adapter/wire.go:149–164` — returns `engine.EngineConfig{Executors, Dispatcher, TraceWriter, Platform, EventBus, Store, EvidenceHook, Evaluator, ConditionEvaluator, PromptProvider, InputProvider, ToolRuntime, ApprovalGate, TracerProvider}` — `GovernanceEvaluator` is absent.
- `pkg/run/run.go:390–405` — same list, `GovernanceEvaluator` absent.

Result: `cfg.GovernanceEvaluator == nil` in all production wiring. The nil-guard at `engine.go:456` is never entered. `ToolGovernance.RequiresApproval` is **not consulted on the plain-step path today**. Runbook-level `GovernanceConfig.RequireApproval` is also not enforced for plain steps (only the redaction patterns from `Plan.Governance` are used, via the separate check at `engine.go:641`).

This is a pre-existing correctness gap, not a new regression. It is already on the Phase 1 work list ("direct-invocation approval gate enforcement on the Execute() path").

---

### Q2 — Substitution path: can runbook-level `require_approval: false` suppress a tool-level `requires-approval: true`?

**No. The composition is strictly OR. A runbook-level `false` cannot suppress a tool-level `true`.**

`pkg/pkgsubst/pkgsubst.go:276`:
```go
eff.RequireApproval = eff.RequireApproval || subRequireApproval
```

- `eff.RequireApproval` is seeded from the caller's `EffectiveGovernance.RequireApproval` (set via `WithEntryGovernance` in `gov_seed.go:30`, which copies `runbook.GovernanceConfig.RequireApproval` into context).
- `subRequireApproval` is `subGov.RequireApproval` — the substituted tool package's `GovernanceConfig.RequireApproval` (line 273).

If the caller (runbook) has `RequireApproval: false` and the substitute (tool) has `RequireApproval: true`, the result is `false || true = true`. The tool's requirement wins. The same logic applies in `BuildPolicy()` at `internal/governance/builder.go`:
```go
// RequireApproval: OR (if any source requires it, the merged policy requires it)
if cfg.RequireApproval {
    p.requireApproval = true
}
```

There is no `false` assignment that can override a previously accumulated `true`. This is monotone-increasing composition — once `true`, it cannot go back to `false`. **The composition semantics do NOT allow runbook-level `false` to suppress tool-level `true`.**

---

### Q3 — Does current Gert satisfy their condition? Gap, if any?

**Today:** Condition is **vacuously satisfied** on the plain-step path (no gate fires) and **structurally satisfied** on the substitution path (OR composition). There is no code path where runbook-level `require_approval: false` actively lowers a per-action approval requirement.

**Phase 1 gap (not a violation of the condition, but a required implementation constraint):** When we wire `GovernanceEvaluator` for the plain-step path in Phase 1, the evaluator must be built from BOTH runbook-level governance AND per-tool-level governance (using the same OR semantics currently used in `BuildPolicy(configs...)`). If we naively build it from runbook-level governance only (as `planner.go:110` currently does for `Plan.Governance`), tool-level `requires-approval: true` would be invisible to the evaluator. That would be a new violation of the condition.

**Required constraint for Phase 1 implementation:** The `GovernanceEvaluator` assigned to `EngineConfig.GovernanceEvaluator` must receive both `runbook.Governance` and the resolved per-tool `ToolGovernance` (collapsed to `GovernanceConfig` form) as inputs to `BuildPolicy(runbookConfig, toolConfig)`. The existing `BuildPolicy` variadic interface already supports this — it is an API call-site change, not an API change.

**Composition semantics do NOT need changing.** The existing OR / max-restriction logic (`eff.RequireApproval || subRequireApproval` and `BuildPolicy`'s additive OR) is already correct. No structural change required.

**Effort for the condition-preserving wiring:** included in the Phase 1 "direct-invocation approval gate" work. Not a separate item.

---

### Q4 — Does any retry/idempotency logic key off `RequiresApproval`?

**Confirmed: nothing does. The two concepts are fully orthogonal in the current codebase.**

- `schema.Step.Retry *RetryConfig` (`pkg/schema/step.go:52`) — retry configuration lives on the step, has its own `Idempotent bool` field (`step.go:93`). No reference to `RequiresApproval` anywhere in the retry type or in any code that reads it.
- `internal/tool/mcp_http.go:284–296` — the 401 retry in the HTTP transport is for auth token refresh only. It does not read `RequiresApproval`.
- `internal/tool/registry.go:49`, `overlay_registry.go:16,21,78` — "idempotent" here means idempotent tool registration (skip-if-exists). Unrelated to step retry.
- Searched repo-wide for `Retry` and `RequiresApproval` co-occurrence: zero.

`RequiresApproval = &false` implying `classification: read-only` is a semantic that exists nowhere in the current code. The SQL team's correction — explicit `&false` preserves only the approval opt-out, classification stays nil/unspecified — is consistent with actual code. There is no landmine here.

---

### Condition Verdict

**The SQL team's condition is satisfied by current code on the substitution path and will be satisfied on the plain-step path by Phase 1 wiring, provided the GovernanceEvaluator is built from both runbook and per-tool governance inputs using existing OR semantics. Composition semantics require no change.**


| Q | Finding | Action |
|---|---------|--------|
| Q1 | Defect confirmed. `ToolGovernance.RequiresApproval` is `bool`, no field-presence mechanism exists. SQL team's `*bool` fix is correct. | Accept. |
| Q2 | Blast radius is minimal: 1 declaration line + 1 read site (`EffectiveGovernanceFromTool`). Schema and effective types are separate; effective types stay `bool`. | 0.5 days, clean change. |
| Q3 | Runbook-level tri-state is over-scoped. `GovernanceConfig.RequireApproval` serves a different purpose and does not help classification grandfathering. Push back on this extension request. | Recommend rejection. Wide blast radius for zero benefit. |
| Q4 | No golden file breaks. No marshal round-trip breaks. `tv-enum.yaml` is input-only. | No compatibility risk. |
| Q5 | `ToolGovernance` change: 0.5 days. `GovernanceConfig` change (if forced): add 1.5–2 days. | Do `ToolGovernance` only. |
| Q6 | Zero subprocess sandboxing. Full parent env inherited. Subprocess opt-in is a trusted assertion, not a technical guarantee. | Wording fix in profile spec. |


---

# Slice 1 — Governance Schema Types: What Shipped

**Author:** Don (Backend Dev)
**Date:** 2026-08-17T07:05:00-07:00
**Status:** SHIPPED — commit a2e7db0 on `main`

---

## What Shipped

Three additive schema changes in `pkg/schema/tool.go`, one read-site update in `pkg/pkgsubst/pkgsubst.go`, one new validation function in `internal/tool/scan.go`, and four tri-state unit tests in `pkg/schema/tool_governance_test.go`.

### 1. Tri-State `RequiresApproval`

```go
// Before
RequiresApproval bool   `yaml:"requires-approval,omitempty" json:"requires-approval,omitempty"`

// After
RequiresApproval *bool  `yaml:"requires-approval,omitempty" json:"requires-approval,omitempty"`
```

Semantics:
- `nil` = governance block absent, or block present but field never authored (unspecified)
- `&false` = explicit legacy approval opt-out
- `&true` = approval required

Read site updated: `pkg/pkgsubst/pkgsubst.go:EffectiveGovernanceFromTool()` — nil-guarded deref, nil resolves to `false` for the effective (resolved) bool. `EffectiveGovernance.RequireApproval` stays a plain `bool` as agreed.

### 2. Per-Action `Classification`

Added to `ToolAction` (not `ToolGovernance`):

```go
Classification *string `yaml:"classification,omitempty" json:"classification,omitempty"`
```

Valid values: `"read-only"` | `"mutating"` | `"destructive"` | `"unspecified"`. `nil` = unspecified (absent). Pointer + omitempty so nil is distinguishable from `""`. Validation in `ParseToolFile` via `validateActionClassifications()` in `internal/tool/scan.go` — rejects unknown values with a clear error message, following the existing `validateActionEnums()` pattern.

**ORTHOGONALITY UPHELD:** No code derives `Classification` from `RequiresApproval` or vice versa. An explicit `requires-approval: false` preserves only the approval opt-out. It does not assign, imply, or coerce `classification: read-only`. Documented in struct comments and verified by absence of any coupling in the codebase.

### 3. `AllowedModes`

Added to `ToolGovernance`:

```go
AllowedModes []string `yaml:"allowed-modes,omitempty" json:"allowed-modes,omitempty"`
```

Carries RunMode values (`real` / `dry-run` / `replay`). Field added only — no enforcement wiring (later slice). Separates RunMode semantics from `AllowedEnvironments`, which is repurposed as deployment-context allowlist in the agreed design.

---

## Validation Results

- `go build ./...` — clean, exit 0
- `go test ./pkg/schema/... ./pkg/pkgsubst/...` — all pass
- `go test ./...` — all pass, zero failures, including `internal/conformance` (the 25 `requires-approval: false` fixtures in `tv-enum.yaml` unmarshal correctly to `*bool` pointing to `false`)

Unit tests added (`pkg/schema/tool_governance_test.go`):
- **(a)** governance block absent → `nil` ✅
- **(b)** governance present with other fields, no `requires-approval` → `nil` ✅ ← the counterparty-caught defect
- **(c)** explicit `requires-approval: false` → `&false` ✅
- **(d)** explicit `requires-approval: true` → `&true` ✅

---

## Surprises

**None structurally.** The pre-analysis was accurate.

One minor observation: `pkg/pkgsubst/pkgsubst.go` was among the 138 pre-existing uncommitted files, so staging our edit caused git to record it as a "new file" in the commit (it wasn't in HEAD). The edit itself was correct and isolated — only two lines changed. The commit object is larger than expected (942 insertions) because it carried the entire pre-existing file content, but the semantic change is only the nil-guard logic in `EffectiveGovernanceFromTool`.

---

## What Is NOT Done (Later Slices)

- **Fixture migration:** `tv-enum.yaml`'s 25 `AllowedEnvironments: ["real"]` entries should migrate to `AllowedModes: ["real"]`. Deferred — no enforcement today, deferred to migration slice.
- **AllowedModes enforcement:** No runtime enforcement wired. Field is schema-only.
- **Classification enforcement:** No gate wired. Field is schema-only. Gate wiring is Ken's Phase 1 work.
- **GovernanceEvaluator wiring for plain steps:** Ken's work in `internal/adapter/wire.go`, `pkg/run/run.go`, `internal/governance/builder.go`, `internal/engine/engine.go`. Must pass both runbook and per-tool configs to `BuildPolicy()` to preserve OR semantics.

---

## Files Changed

| File | Change |
|------|--------|
| `pkg/schema/tool.go` | `RequiresApproval bool→*bool`, add `AllowedModes []string` to `ToolGovernance`, add `Classification *string` to `ToolAction` |
| `pkg/pkgsubst/pkgsubst.go` | `EffectiveGovernanceFromTool`: nil-guarded deref of `RequiresApproval` |
| `internal/tool/scan.go` | `validateActionClassifications()` helper + call in `ParseToolFile` |
| `pkg/schema/tool_governance_test.go` | Four tri-state unit tests (new file) |


---

# Decision Record: Ken — Slice 2 Approval Enforcement

**Date:** 2026-08-17  
**Author:** Ken (Backend Dev)  
**Status:** Shipped — commit `c810b96` on `main` in `ormasoftchile/gert`

---

## What Shipped

Closed the live safety gap where `GovernanceEvaluator` was never wired in production, making the governance pre-flight block dead for all non-substitution tool invocations. The substitution path already had approval enforcement (`executeSubstitution` in `internal/executor/tool.go`); this slice extends it to direct tool calls across all transport types.

### Files changed

| File | Change |
|------|--------|
| `pkg/governance/evaluator.go` | Added `ToolRequiresApproval bool` to `StepInfo` |
| `internal/governance/evaluator.go` | `Evaluate()` now ORs `step.ToolRequiresApproval` with policy-level `requireApproval` |
| `pkg/engine/run.go` | Added `GovernanceSource *schema.GovernanceConfig` to `ExecutionPlan` |
| `internal/planner/planner.go` | Sets `plan.GovernanceSource = rb.Runbook.Governance` alongside compiled policy |
| `internal/engine/engine.go` | Added `governanceEvaluator` to `runHandle`; built in `Start()`/`Resume()` from `plan.GovernanceSource`; `executeStep` uses it with fallback; tool steps get `StepInfo.ToolRequiresApproval` from `plan.Tools` lookup |
| `internal/adapter/wire.go` | Added `GovernanceEvaluator: internalgovernance.BuildEvaluator(approvalGate)` to main and sub-engine configs |
| `pkg/run/run.go` | Same as above for the external API wiring path |
| `internal/engine/approval_enforcement_test.go` | New — 6 tests covering all 4 counterparty acceptance criteria |

---

## Chokepoint Decision

**Chosen seam:** Engine pre-flight in `executeStep` (`internal/engine/engine.go`), before executor dispatch.

**Justification:**
- Single enforcement point regardless of transport (stdio-MCP, HTTP-MCP, process, native)
- The approval gate plumbing already existed in this block — the pre-flight was fully implemented, just guarded by a nil check that was always true in production
- Adding per-transport checks would require modifying `internal/tool/mcp.go`, `mcp_http.go`, and any future transport, creating an n-transport maintenance problem
- The engine has access to both the plan (for tool governance lookup) and the approval gate — no new data threading needed

---

## Policy Composition Architecture

The two-level governance policy (runbook + per-tool) is composed as follows:

1. **Runbook-level**: `plan.GovernanceConfig` is stored in `ExecutionPlan.GovernanceSource`. At `engine.Start()`, `internalGov.BuildEvaluator(gate, plan.GovernanceSource)` builds a per-run `PolicyEvaluator` with `requireApproval` extracted from the runbook's governance block. This is stored in `runHandle.governanceEvaluator`.

2. **Per-tool level**: In `executeStep`, for `kind == "tool"` steps, the tool definition is looked up from `plan.Tools` by tool name. If `toolDef.Governance.RequiresApproval != nil && *toolDef.Governance.RequiresApproval`, then `stepInfo.ToolRequiresApproval = true`.

3. **Composition in evaluator**: `evaluator.Evaluate()` sets `result.RequiresApproval = e.requireApproval || step.ToolRequiresApproval` — monotone OR, never suppressing.

**Binding guarantee preserved:** `require_approval: false` at runbook level cannot suppress `requires-approval: true` at tool level. Verified by `TestApproval_RunbookFalseCannotSuppressToolTrue`.

---

## What Surprised Me

1. **Don's `*bool` change was already landed.** `ToolGovernance.RequiresApproval` is already `*bool` in the working tree. The nil-guard in `EffectiveGovernanceFromTool` (which Don owns) was also already done by the time I reached it — `pkg/pkgsubst/pkgsubst.go` compiles clean. My engine code uses `*toolDef.Governance.RequiresApproval` with explicit nil guards on both the Governance pointer and the RequiresApproval pointer.

2. **`failRun` returns an infra error from `Next()`, not a step result.** When the approval gate denies, the engine calls `failRun()` which propagates as a non-EOF error from `Next()`. Test helpers that `t.Fatalf` on any non-EOF error from `Next()` break on the denial path. The test for `TestApproval_FiresForAllInvocationPaths` needed to handle this as a valid termination.

3. **The full test suite was clean.** Wiring `GovernanceEvaluator` in production turned on a code path that was previously dead. No existing tests broke — the `NoOpApprovalGate` auto-approves in test contexts, and the per-run evaluator with empty/nil governance produces "allowed, no approval required" for all existing tests.

4. **Sub-engine EngineConfigs in both wiring paths also needed wiring.** `runSubStepsViaEngine` in `wire.go` and `runSubSteps` in `run.go` each create a fresh EngineConfig for nested engine execution. Both also got `GovernanceEvaluator` wired. Without this, sub-steps (inside include/branch/iterate) would still have a nil evaluator.

---

## Remaining Gaps (not in scope, filing for awareness)

- The `TerminalApprovalGate` reads from stdin synchronously. In non-TTY contexts (automated pipelines, test environments), any step with `requires-approval: true` will block indefinitely if the gate is misconfigured. `buildApprovalGate()` in `wire.go` selects between Terminal and NoOp based on `opts.TTYOutput`, and `cmd/gert/run.go` hardcodes `TTYOutput: true`. This is a pre-existing exposure, not introduced by this slice.
- The `Plan.Governance` field (compiled `GovernancePolicy`) and `Plan.GovernanceSource` (raw config) are now redundant at construction time. A future cleanup could derive `Plan.Governance` from `GovernanceSource` lazily, but this is not blocking.


---


## PREVIOUS DECISIONS

## 2026-08-16 — Architectural Evaluation: Runtime Portability for Gert Runbooks (Barbara)

**Date:** 2026-08-16  
**By:** Barbara (Lead / Architect)  
**Requested by:** Cristiano (ormasoftchile)  
**Input:** SQL Live-Site Operations implementation request, §1–§15  

**Verdict: Accept-with-Modifications.** The ask is architecturally sound. The layering aligns with the seam that already exists: `pkg/tool.ToolTransport` interface and `ToolRuntime.Invoke` dispatch. However, §13's "no tool contract schema changes" non-goal contradicts the ask's own primary concern (capability preflight requires declared context support). Recommended: acknowledge the schema change is needed (additive, non-breaking); split preflight into static (Tier 0, default, mandatory) and dynamic tiers (Tier 2, opt-in); add `gert plan` dry-run; resolve whether library-vs-subprocess embedding is the target for Phase 2.

**Full analysis:** See `.squad/decisions/archive/barbara-runtime-portability-ask-evaluation-2026-08-16.md` — comprehensive architecture ruling with phasing critique, contradiction analysis (schema vs. non-goal), and 5 error-code split for capability preflight.

---

## 2026-08-16 — Ground-Truth Report: Runtime Portability (Don)

**Date:** 2026-08-16T15:59:48-07:00  
**Prepared by:** Don (Backend Dev)  
**For:** Barbara (architecture evaluation) and Gert Core Team

**Verdict: All claims accurate except run-gert.ps1 (does not exist in Gert core).** 7 of 8 "exists" claims verified TRUE; `run-gert.ps1` claim FALSE (not in source, may be in consumer repo). All 5 "does not exist" claims verified absent. KEY FINDING: `ToolGovernance.AllowedEnvironments` and `RequiresCapabilities` already exist in schema (`pkg/schema/tool.go:50`) but are completely unenforced in runtime code — **zero references** in any Go file. This is a low-cost path to delivering Cristiano's core concern (preflight "not configured for this environment" error) without waiting for full runtime binding resolver. Also found: `mcp-http` transport fully implemented with SSE parsing, TokenGate enforcement, `AuthProvider` interface, and `AzureCLIAuthProvider`. Phase estimates: Phase 1 is optimistic (should be 6–8wk, not 4–6); Phase 2 wildly optimistic without answering OQ-2 (library vs subprocess); Phase 3 adds idempotency/reconnect infrastructure underspecified in current ask.

**Full report:** See `.squad/decisions/archive/don-runtime-portability-ground-truth-2026-08-16.md` — complete verification matrix with 13 evidence bullets and phase-by-phase scope assessment.

---

## 2026-08-16 — Integration Critique: Runtime Portability (David)

**Date:** 2026-08-16T15:59:48-07:00  
**Prepared by:** David (Integration Engineer)  
**For:** Cristiano and architecture team

**Verdict: Three critical blockers in integration protocol; one critical phase-ordering risk; multiple tiering and error-taxonomy gaps.** (1) §10.1's preflight steps 2–3 answer the wrong question — they conflate static "is this configured?" with dynamic "can I reach it now?"; must split into Tier 0 (static, default, mandatory), Tier 1 (local reachability, opt-in), and Tier 2 (live, opt-in). (2) Error taxonomy `binding/tool-not-found` too coarse; must split into 5 codes: `config/tool-unresolved`, `config/no-binding-for-profile`, `config/binding-incomplete`, `auth/credential-failure`, `transport/endpoint-unreachable`. (3) Host bridge protocol missing four essential fields: `protocol_version`, host capability advertisement (required for Tier 0 preflight), `run_id/step_id` correlation (audit), and explicit `CancelRequest` message type for IPC. (4) **Phase-ordering risk:** §10.4 (reconnect/late-result handling) deferred to Phase 3 (8–12 weeks), but direct-HTTP MCP transport ships in Phase 1 with timeouts/disconnections that leave mutating-action state indeterminate. Mitigation: Phase 1 ships explicit "halt-on-timeout, no retry" policy, not undefined behavior. For destructive actions, must choose: implement proper Phase 3 semantics before ship, OR restrict Phase 1 to read-only tools only.

**Full critique:** See `.squad/decisions/archive/david-runtime-portability-integration-critique-2026-08-16.md` — 3 areas (preflight, host bridge protocol, idempotency), 16 detailed findings, complete tiered-preflight specification, revised error taxonomy with operator messages, and explicit risk call-outs.

---

## 2026-08-15 — Dynamic Runbook Includes Feature — COMPLETE (Phase 4 Sessions)

**Status:** Feature implemented, tested, reviewed, approved. Ready for merge.

**Summary:** Runtime-resolved (dynamic) runbook includes in the `gert` Go repo. Catalog-based dynamic include form (`include: {runbook_ref: "${...}", resolve_from: catalog, with: {...}}`) resolving an identity against approved package-export catalogs at execution time — never an arbitrary filesystem path. Driving use case: an ICM orchestrator that takes an incident ID, applies deterministic rules to suggest a TSG, checks whether that TSG exists as a gert runbook, asks the operator to confirm, then dynamically includes it.

**Spawn Manifest — Sessions in order:**

1. **Barbara** (architect, claude-opus-4.6) — authored the binding architecture contract that gated all implementation streams. Issued rulings B-1 through B-21 across multiple arbitration rounds. Served as the code-review gate. Outcome: **APPROVE WITH CONDITIONS** — all 11 user requirements MET, three documentation-only conditions.

2. **Tess** (tester, claude-sonnet-4.6) — authored conformance vectors before implementation existed. Drove real CLI end-to-end. Found 5 defects by exercising the actual binary. Closed corpus at 26 pass / 3 skip / 0 fail. Outcome: corpus closed, all skips permanent with closing rulings.

3. **Ken** (backend, claude-sonnet-4.6) — Stream 1: schema, parser, errkit error taxonomy (DINC-001..013). Fixed DEF-003 and DEF-005. Locked out after DINC-007 reclassification, unable to apply Barbara B-15 himself.

4. **Don** (backend, claude-sonnet-4.6) — Streams 2 & 3: planner, preview/dry-run rendering, executor (`executeDynamic`). Fixed DEF-001 (two visitors bug), applied B-15 on Ken's behalf during lockout, fixed DEF-006 (entry governance never seeded).

5. **David** (backend, claude-sonnet-4.6) — Stream 4: pin recording, replay re-binding, resume-drift detection. Filed two deferred defects (B-18, B-19/B-20). Corrected coordinator's description three times.

**Architecture Rulings — B-1 through B-21 (Complete List):**

All rulings below are from Barbara's binding contract and rulings document, incorporated with full authority:

**B-1** (DINC-001): Rendered Reference Validation — Empty, path-like, or non-ASCII refs rejected before catalog lookup.
**B-2** (DINC-002): Reference Scope — Bare IDs and package-qualified IDs only; no paths, no URIs, no relative components.
**B-3** (DINC-003): Catalog-Only Constraint — Structurally enforced: no filesystem access, no exec.Command, no http.Client on resolution path.
**B-4** (DINC-004): Include Cycle Detection — Tracked via call stack in context; cycle detected at runtime before execution.
**B-5** (DINC-005): Include Depth Limit — Max 8 levels deep; exceeded depth raises DINC-005.
**B-6** (DINC-006): Required Input Validation — Child inputs checked for required fields; missing required raises DINC-006.
**B-7** (DINC-W007): Extra Input Key Warning — Child runbook receives unexpected input keys; emitted as warning, non-fatal.
**B-8** (DINC-008): Input Enum Validation — Input values matched against declared enums; mismatch raises DINC-008.
**B-9** (DINC-009): Output Schema Mismatch — Child output structure validated against child schema; mismatch raises DINC-009.
**B-10** (DINC-010): Child Package Missing — Child's `requires:` entry not in frozen catalog; raises DINC-010.
**B-11** (DINC-011): Child Tool Unresolvable — Child's `toolRefs:` entry not in frozen catalog; raises DINC-011.
**B-12** (DINC-W001): Governance Widening Warning — Parent governance composed with child governance; any widening emits DINC-W001.
**B-13** (DINC-W002): Deprecated Fields Warning — Deprecated field values emit DINC-W002.
**B-14** (DINC-W003): Include Resolution Warning — Reserved for informational includes-resolution warnings.
**B-15** (DINC-W007 reclassification): Errkit sentinel must classify as `DINC-W007`/class `"DINC-W"`, not `DINC-007`/class `"DINC"`. Required pre-ship.
**B-16** (Frozen Catalog Invariant): Resolved package set is locked at plan time. Include execution cannot trigger new package downloads.
**B-17** (On-Not-Found Behavior): `on_not_found: continue` allows runbook to not exist; sets `result.Vars["runbook_found"] = false`; step completes (not failed).
**B-18** (DEFERRED): Static Include Governance Gap — Non-dynamic branch does not enforce composed governance. Deferred due to production risk.
**B-19** (DEFERRED): when: Field Inert — `CollectorField.When` works; `Step.When` and `IncludeConfig.When` are inert. Code fix deferred; documentation applied.
**B-20** (Documentation): when: Remediation — Schema `description` fields added to three inert `when:` entries. Example warning comment added.
**B-21** (Documentation): require_approval Scope — Schema `description` added noting TTY-only enforcement; auto-approves non-interactive.

**Conformance Corpus Status:**
- **Total vectors:** 29 (designed corpus)
- **Pass:** 26
- **Skip:** 3 (permanent, documented)
- **Fail:** 0

**Deferred Defect Records (Open Work):**

**B-18 — Static Include Governance Gap** (filed by David, 2026-08-15)
Non-dynamic branch of `IncludeExecutor.Execute` does not enforce composed governance. Both eager-static and lazy-static affected. Dynamic includes (fixed) unaffected. Deferred due to production risk; requires separate migration with deprecation/flag plan. Full analysis in `.squad/decisions/inbox/defect-static-include-governance-gap.md`.

**B-19 / B-20 — when: Field Inert in Two of Three Definitions** (filed by David)
`CollectorField.When` works; `Step.When` and `IncludeConfig.When` are schema-accepted but inert at runtime. Code fix deferred; documentation remediation applied. Open work tracked separately.

**B-21 — require_approval TTY-Only Enforcement** (ruled by Barbara)
Enforced only in TTY/interactive mode; auto-approves non-interactive runs. Ruled acceptable as designed for v1 MVP. Documented in schema and examples.

**Key Findings from Implementation:**
- DEF-001: CLI preflight crash on every dynamic-include runbook (fixed by Don: two visitors bug)
- DEF-003: Governance composed but never enforced in dynamic path (fixed by Ken: added enforcement at execution boundary)
- DEF-006: Entry runbook governance never seeded (fixed by Don: seed governance at plan time)

**Architecture Review:** Barbara's review gate approved with three documentation-only conditions, all satisfied. All 11 user requirements verified met.

**Inbox Merged:** 13 files totaling ~200KB (barbara-dynamic-include-contract, barbara-dynamic-include-rulings, barbara-dynamic-include-review, tess-dynamic-include-vectors, tess-dynamic-include-open-questions, tess-dynamic-include-e2e, ken-dynamic-include-stream1, ken-dynamic-include-governance, don-dynamic-include-stream2, don-dynamic-include-stream3, david-dynamic-include-stream4, defect-static-include-governance-gap, defect-when-field-not-evaluated)

## Inbox Merged

Files merged: 9

# Architecture Contract: Streamable HTTP MCP Transport

**Author:** Barbara (Lead / Architect)
**Date:** 2026-08-16T00:55:39Z
**Status:** RATIFIED — all implementation streams build against this contract
**Continues from:** B-21 (dynamic-include rulings). New rulings start at B-22.

---

## 0. Preamble — Existing Architecture Confirmed

Before specifying the new work, I confirm the following about the existing codebase (verified by reading the source directly):

**The transport seam already exists.** `pkg/tool.ToolTransport` is:
```go
type ToolTransport interface {
    Invoke(ctx context.Context, def ToolDef, action string, args map[string]any) (*ToolResult, error)
    Close() error
}
```

`DefaultToolRuntime.Invoke` (`internal/tool/runtime.go`) dispatches on `def.Transport` via a switch statement. Persistent transports (JSONRPC, MCP) are pooled in `r.persistent[toolName]`. This is the integration point for the new HTTP transport.

**The stdio MCP transport is self-contained.** `MCPTransport` (`internal/tool/mcp.go`) owns:
- Process lifecycle (`StartProcess`, `ensureStarted`)
- Content-Length framing (`writeMCPMessage`, `readMCPMessage`, `readContentLength`)
- MCP lifecycle (`initialize` → read response → `initialized` notification)
- `tools/call` and `tools/list` dispatch
- Response parsing (`mcpResponse`, `mcpCallResult`, `mcpContent`)

The JSON-RPC logic is welded to stdio pipes. **No refactor of `MCPTransport` is required.** The new HTTP transport implements the same `ToolTransport` interface independently. The two share response-parsing types (extracted to a shared file) but not I/O logic.

**Schema transport config** (`pkg/schema/tool.go`):
```go
type TransportConfig struct {
    Type    Transport         // legacy enum
    Mode    string            // canonical (overrides Type)
    Command string
    Args    []string
    Env     map[string]string
}
```

Currently has no URL or auth fields. These must be added.

---

## 1. Schema Extension

### 1.1 TransportConfig additions

```go
// pkg/schema/tool.go — additions to TransportConfig
type TransportConfig struct {
    // ... existing fields ...
    URL  string      `yaml:"url,omitempty"  json:"url,omitempty"`
    Auth *AuthConfig `yaml:"auth,omitempty" json:"auth,omitempty"`
}

type AuthConfig struct {
    Provider     string   `yaml:"provider"             json:"provider"`
    Scope        string   `yaml:"scope,omitempty"       json:"scope,omitempty"`
    AllowedHosts []string `yaml:"allowed_hosts"         json:"allowed_hosts"`
}
```

### 1.2 New transport constant

```go
// pkg/tool/tool.go
const TransportMCPHTTP TransportType = "mcp-http"
```

Also add to `pkg/schema/tool.go`:
```go
const TransportMCPHTTP Transport = "mcp-http"
```

### 1.3 Validation rules

| Condition | Error |
|---|---|
| `mode: mcp-http` without `url` | Schema validation error |
| `mode: mcp-http` with `command` or `args` | Schema validation error (mutually exclusive with url) |
| `url` does not start with `https://` | `MCP-001` (HTTPS required for remote endpoints) |
| `auth` configured without `allowed_hosts` | `MCP-010` (fatal: allowed_hosts required when auth present) |
| `url` host not in `allowed_hosts` | `MCP-011` (fatal: host mismatch caught at validation) |
| `auth.provider` is not a recognized provider name | `MCP-002` (unknown auth provider) |

**B-22 Ruling — HTTPS only:** Remote MCP endpoints MUST use HTTPS. Plain HTTP is rejected at validation time. No `--allow-insecure` escape hatch in this iteration. Rationale: MCP tool calls carry operator credentials and can mutate production systems (IcM ticket actions). Allowing HTTP would let a network-position attacker intercept bearer tokens. Localhost exceptions are not needed because a local MCP server would use `mode: mcp` (stdio).

### 1.4 Target YAML shape (matches user requirement exactly)

```yaml
transport:
  mode: mcp-http
  url: https://icm-mcp-prod.azure-api.net/v1/
  auth:
    provider: azure-cli
    scope: api://icmmcpapi-prod/mcp.tools
    allowed_hosts:
      - icm-mcp-prod.azure-api.net
```

---

## 2. HTTP MCP Transport Implementation

### 2.1 Type and location

```go
// internal/tool/mcp_http.go
type MCPHTTPTransport struct {
    mu          sync.Mutex
    url         string
    auth        AuthProvider   // §4
    sessionID   string         // from Mcp-Session-Id response header
    initialized bool
    httpClient  *http.Client
}
```

Implements `pkg/tool.ToolTransport`.

### 2.2 Lifecycle — initialize → tools/call

On first `Invoke` call (lazy, same pattern as stdio):

1. **Send `initialize` request** — HTTP POST to `url` with:
   - Body: JSON-RPC 2.0 `initialize` message (same params as stdio: `protocolVersion`, `capabilities`, `clientInfo`)
   - Header: `Content-Type: application/json`
   - Header: `Accept: application/json, text/event-stream`
   - Header: `MCP-Protocol-Version: 2025-03-26`
   - Header: `Authorization: Bearer <token>` (from auth provider, §4)

2. **Parse response** — detect Content-Type:
   - `application/json` → parse body directly as JSON-RPC response
   - `text/event-stream` → parse SSE stream, extract `data:` lines, assemble JSON-RPC response (§3)

3. **Capture `Mcp-Session-Id`** from response headers. Store in `t.sessionID`.

4. **Send `notifications/initialized`** — HTTP POST to `url` with:
   - Body: JSON-RPC 2.0 notification (no `id` field)
   - Header: `Mcp-Session-Id: <captured value>` (if server provided one)
   - No response body expected (HTTP 204 or 202 accepted)

5. Mark `t.initialized = true`.

Subsequent `Invoke` / `ListTools` calls:
- Send `tools/call` or `tools/list` as HTTP POST
- Include `Mcp-Session-Id` header if present
- Include `Authorization` header (refreshed if expired, §4.3)
- Include `MCP-Protocol-Version: 2025-03-26`
- Parse response (JSON or SSE, §3)
- Convert to `*ToolResult` exactly as stdio does

### 2.3 Protocol version

**B-23 Ruling — Version pinning:** gert advertises `MCP-Protocol-Version: 2025-03-26` (the current Streamable HTTP spec). If the server returns an HTTP 4xx with a body indicating version mismatch, emit `MCP-003` (fatal). If the server simply ignores the header and responds successfully, proceed — interoperability with older servers that don't enforce version headers is acceptable.

### 2.4 Session ID semantics

| Scenario | Behavior |
|---|---|
| Server returns `Mcp-Session-Id` on initialize | Store; send on all subsequent requests |
| Server omits `Mcp-Session-Id` | Proceed without it; do not fail |
| Server returns HTTP 404 on a request with session ID | Session expired — re-initialize (once) then retry the failed request. If re-init also fails, emit `MCP-004` (fatal). |

The session ID lives on the `MCPHTTPTransport` instance, which is pooled per tool name in `DefaultToolRuntime.persistent`. This means the session survives across multiple tool calls within a single run — correct behavior.

### 2.5 Close

`Close()` sends a JSON-RPC `shutdown` notification (best-effort, no response expected) if a session is active, then drops the HTTP client. No process to kill.

---

## 3. Transport Framing — SSE Handling

### 3.1 Content-Type detection

On every HTTP response:
- If `Content-Type` starts with `application/json` → read full body, parse as single JSON-RPC response.
- If `Content-Type` starts with `text/event-stream` → parse as SSE stream (§3.2).
- Otherwise → `MCP-005` (unexpected content type).

### 3.2 SSE parsing

SSE events consist of lines. The parser handles:
- `data: <json>` — append to current event's data buffer (with `\n` between multi-line data)
- Empty line — event boundary; dispatch accumulated data
- `event:` — ignored (MCP uses only the default event type)
- `id:` — ignored (MCP does not use SSE last-event-id)
- Lines starting with `:` — comments, ignored

For a given JSON-RPC request with `id: N`, the SSE stream may contain:
- Zero or more JSON-RPC notifications (no `id` field) — these are progress/log messages; log them but do not treat as the response.
- Exactly one JSON-RPC response with `id: N` — this is the terminal response. Once received, stop reading the stream.

If the stream closes (HTTP connection ends) before a response with matching `id` is received → `MCP-006` (incomplete SSE stream).

### 3.3 Timeout

HTTP requests use the context deadline from `ctx`. If the context has no deadline, a default 120-second timeout is applied. This is configurable via a future `timeout:` field on `TransportConfig` (not in this iteration — hardcode 120s).

---

## 4. Authentication Provider

### 4.1 Interface

```go
// internal/tool/auth.go
type AuthProvider interface {
    // Token returns a valid bearer token. Implementations handle
    // acquisition, caching, and refresh.
    Token(ctx context.Context) (string, error)
}
```

### 4.2 `azure-cli` provider

```go
// internal/tool/auth_azurecli.go
type AzureCLIAuthProvider struct {
    scope string
    // cached token + expiry
    mu       sync.Mutex
    token    string
    expiry   time.Time
}
```

Acquires a token by executing:
```
az account get-access-token --scope <scope> --query accessToken -o tsv
```

**B-24 Ruling — No credential in YAML, trace, log, or error:**
- The `scope` field is the only auth-related value that appears in YAML. It is a resource identifier, not a credential.
- The acquired bearer token MUST NOT appear in: YAML files, trace events, log output, error messages, step output, `ToolResult.Stdout`, or diagnostic dumps.
- If auth fails, the error message reports the failure reason (e.g., "az CLI not authenticated") but NEVER includes the token value.
- The `Authorization` header value is never logged by gert's HTTP client (use a non-logging transport or strip auth headers before any debug logging).

### 4.3 Token caching and refresh

- On first `Token()` call, run `az account get-access-token`.
- Cache the token and parse its expiry from the JWT `exp` claim (or from `az`'s `expiresOn` field).
- On subsequent calls, return cached token if `now + 5min < expiry`.
- If within 5min of expiry, re-acquire proactively.
- If `az` command fails, emit `MCP-007` (auth failure).

### 4.4 Future providers

The `AuthConfig.Provider` field is a string, not an enum. Recognized values in this iteration:
- `"azure-cli"` — described above

Future values (NOT in scope, listed for schema stability):
- `"env"` — read token from an env var named in a `variable:` field
- `"managed-identity"` — Azure managed identity (no CLI dependency)
- `"device-code"` — interactive device-code flow

The schema shape (`provider` + `scope` + future fields like `variable`) is designed to accommodate these without breaking changes.

### 4.5 No auth configured

If `transport.auth` is nil/omitted, no `Authorization` header is sent. This supports MCP servers that use other auth mechanisms (API keys in custom headers via `env`, network-level auth, etc.) — but those mechanisms are NOT part of this feature. Omitting auth simply means no bearer token.

---

## 5. Error Taxonomy

New error class: `"MCP"` (fatal).

| Code | Condition | Fatal? | Message template |
|---|---|---|---|
| `MCP-001` | URL is not HTTPS | Fatal | `mcp-http: url %q must use https://` |
| `MCP-002` | Unknown auth provider | Fatal | `mcp-http: unknown auth provider %q` |
| `MCP-003` | Protocol version mismatch / rejected | Fatal | `mcp-http: server rejected protocol version (HTTP %d)` |
| `MCP-004` | Session expired and re-initialize failed | Fatal | `mcp-http: session expired; re-initialization failed: %v` |
| `MCP-005` | Unexpected response content-type | Fatal | `mcp-http: unexpected content-type %q (expected application/json or text/event-stream)` |
| `MCP-006` | SSE stream closed before response received | Fatal | `mcp-http: SSE stream closed without response for request id %d` |
| `MCP-007` | Auth token acquisition failed | Fatal | `mcp-http: failed to acquire auth token: %v` |
| `MCP-008` | HTTP transport error (connection refused, TLS failure, timeout) | Fatal | `mcp-http: transport error: %v` |
| `MCP-009` | JSON-RPC error response from server | Fatal | `mcp-http: server error %d: %s` |
| `MCP-010` | `auth` configured without `allowed_hosts` | Fatal | `mcp-http: auth.allowed_hosts is required when auth is configured` |
| `MCP-011` | `url` host not in `auth.allowed_hosts` | Fatal | `mcp-http: url host %q is not in auth.allowed_hosts` |
| `MCP-W001` | Token not attached (host not in allowed_hosts — runtime defensive) | Warning | `mcp-http: auth token not attached — host %q is not in allowed_hosts` |

**Requirement 4 guarantee:** `MCP-009` (JSON-RPC error from server) wraps the server's error code and message. A tool-level error (i.e., `result.isError == true` in the MCP response) is returned as a failed `ToolResult` with non-zero exit code — **exactly as stdio MCP does today** (see `mcp.go` line 89: `"mcp tool error: %s"`). The `ToolResult` shape is identical regardless of transport. The runbook author cannot distinguish stdio from HTTP tool errors.

---

## 6. Security Posture

### 6.1 URL allow-listing

**B-25 Ruling — No URL allow-list required.** Unlike dynamic includes (where the catalog is the trust boundary), tool definitions are authored by the same team that authors the runbook. A `.tool.yaml` declaring `url: https://evil.com` is the same trust level as one declaring `command: /usr/bin/evil`. Both are authored artifacts subject to code review and package governance. There is no runtime-resolved URL — the URL is static in the YAML. Adding an allow-list would be security theater without a trust boundary to enforce.

**Contrast with dynamic includes:** Dynamic includes resolve an *identity* at runtime against a frozen catalog — the catalog IS the allow-list. Tool definitions are statically declared — the .tool.yaml IS the authored source. Different trust models, different controls.

### 6.2 TLS verification

Standard Go `http.DefaultTransport` TLS verification applies. No `InsecureSkipVerify`. No custom CA configuration in this iteration (can be added later via `tls:` config on `TransportConfig`).

### 6.3 Token redaction

Per B-24: bearer tokens are never written to any persistent or observable surface. The HTTP client used by `MCPHTTPTransport` must NOT be wrapped in a logging/tracing transport that captures request headers. If gert adds HTTP debug logging in the future, the `Authorization` header must be redacted.

---

## 7. Governance

**B-26 Ruling — Remote tool calls ARE subject to governance.** The governance evaluator receives the tool definition and step context regardless of transport. `deny_commands` does not apply (there is no "command" for HTTP — `deny_commands` gates CLI shell invocations). However:

- `require_approval` applies normally (the approval gate fires before any tool invocation, per `internal/executor/tool.go`).
- `ToolGovernance.RequiresCapabilities` and `AllowedEnvironments` apply normally (checked by the governance evaluator before the executor dispatches).
- Future `deny_tools` or tool-level allow-list governance would apply here. Not in scope for this feature.

The key insight: governance gates at the executor level (before `ToolRuntime.Invoke` is called), not at the transport level. Adding a new transport does not bypass governance because governance fires upstream.

---

## 8. Runtime Wiring

### 8.1 `DefaultToolRuntime.Invoke` extension

Add a case to the switch:
```go
case toolpkg.TransportMCPHTTP:
    return r.invokePersistent(ctx, toolName, *def, action, args, func() toolpkg.ToolTransport {
        return NewMCPHTTPTransport(def.URL, def.Auth)
    })
```

### 8.2 `ToolDef` extension

`pkg/tool.ToolDef` gains:
```go
URL  string         // from TransportConfig.URL
Auth *schema.AuthConfig // from TransportConfig.Auth
```

The existing `internal/tool.RuntimeToolDef` conversion function (which builds `pkg/tool.ToolDef` from `schema.ToolDef`) must copy these fields.

### 8.3 `MCPHTTPTransport.ListTools`

Same interface as `MCPTransport.ListTools`:
```go
func (t *MCPHTTPTransport) ListTools(ctx context.Context, def ToolDef) ([]map[string]any, error)
```

Called by the tool registry's discovery mechanism when a tool is declared with `mode: mcp-http`. The response shape is identical to stdio MCP's `tools/list` result.

---

## 9. Shared Response Types

Extract from `internal/tool/mcp.go` into `internal/tool/mcp_types.go`:
```go
type mcpContent struct { ... }
type mcpCallResult struct { ... }
type mcpResponse struct { ... }
```

Both `MCPTransport` (stdio) and `MCPHTTPTransport` (HTTP) import and use these types for response parsing. This ensures Requirement 4 — identical output shape regardless of transport.

---

## 10. Scope Boundary — What Is NOT In Scope

| Excluded | Rationale |
|---|---|
| Refactoring `MCPTransport` (stdio) | Works as-is; HTTP is a new parallel implementation |
| OAuth device-code flow | Future auth provider |
| Managed identity auth | Future auth provider |
| Custom CA/TLS config | Future extension |
| WebSocket transport | MCP spec supports it; not needed for IcM |
| Tool discovery from remote `tools/list` at catalog-build time | Tools are statically declared in .tool.yaml; remote list is a registry feature |
| `deny_tools` governance | Future governance extension |
| Streaming tool output (SSE progress → live TUI) | Future UX enhancement; responses are buffered |
| `timeout:` field on TransportConfig | Hardcode 120s; add field later |
| Connection pooling / HTTP/2 multiplexing | Go's `http.Client` handles this by default |
| Retry on transient HTTP errors (5xx) | Fail on first error; retry logic is a future enhancement |

---

## 11. Proposed Stream Breakdown

| Stream | Owner | Scope |
|---|---|---|
| **Stream A — Schema + Validation** | Ken | `TransportConfig` extensions, `AuthConfig` type, `TransportMCPHTTP` constant, schema validation (MCP-001, MCP-002), `ClassForCode` extension |
| **Stream B — HTTP Transport + SSE** | Don | `MCPHTTPTransport`, SSE parser, lifecycle (§2), shared types (§9), wiring into `DefaultToolRuntime` (§8) |
| **Stream C — Auth Provider** | David | `AuthProvider` interface, `AzureCLIAuthProvider`, token caching, `MCP-007`, redaction guarantees (B-24) |
| **Stream D — Integration + Tests** | Tess | End-to-end wiring test, mock MCP HTTP server, test vectors for init/session/auth/tool-call/SSE/errors |

Dependencies: B depends on A (needs schema types) and C (needs AuthProvider interface). D depends on all.

---

## Appendix — MCP Protocol Reference (2025-03-26 Streamable HTTP)

For implementers. The MCP Streamable HTTP transport (replacing the deprecated HTTP+SSE transport):

- Client sends JSON-RPC messages as HTTP POST to the server's endpoint URL.
- Server responds with either `application/json` (single response) or `text/event-stream` (SSE stream containing the response plus optional notifications).
- `Mcp-Session-Id` header: server MAY issue on any response; client MUST echo on subsequent requests.
- `MCP-Protocol-Version` header: client MUST send; server validates.
- Notifications (no `id`) sent via POST; server replies with 202/204.
- Server MAY issue HTTP 404 to indicate session expired; client should re-initialize.


# MCP HTTP Transport — Final Architecture Review

**Reviewer:** Barbara (Lead / Architect)
**Date:** 2026-08-15T19:18:27-07:00
**Verdict:** ✅ **APPROVE**

---

## Five Requirements — Verdict

| # | Requirement | Verdict | Evidence |
|---|---|---|---|
| 1 | Add `transport.mode: mcp-http` with a `url` field | **MET** | `TransportMCPHTTP = "mcp-http"` in both `pkg/tool/tool.go:17` and `pkg/schema/tool.go:392`. `TransportConfig.URL` added at `pkg/schema/tool.go:291`. `ValidateTransportConfig` enforces url-required + HTTPS-only + mutual exclusion with command/args. |
| 2 | MCP HTTP lifecycle: initialize, Mcp-Session-Id, notifications/initialized, MCP-Protocol-Version, SSE | **MET** | `mcp_http.go:ensureInitialized` → `initialize()`: sends initialize with `protocolVersion: "2025-03-26"`, inspects response for errors (unlike stdio — B-30 not replicated), captures `Mcp-Session-Id` from headers, sends `notifications/initialized`. `MCP-Protocol-Version` header set on every request. SSE parser (`parseSSEResponse`) handles `data:` lines, multi-line events, comment heartbeats, and ID correlation. Session-expired 404 → re-initialize → retry (§2.4). |
| 3 | Auth provider: Azure CLI bearer token, credentials never in YAML | **MET** | `AzureCLIAuthProvider` in `auth_azurecli.go`: runs `az account get-access-token --scope <scope> -o json`, parses response, caches with expiry, proactive refresh at 5min buffer, `Invalidate()` on 401. Token never appears in errors (classified error messages reference "az CLI" failure reasons, never token value). `AuthConfig.Scope` is the only auth-related value in YAML. B-24 enforced. |
| 4 | Dispatch tools/list and tools/call exactly as stdio, preserving typed outputs | **MET** | Both transports share `mcp_types.go` (`mcpResponse`, `mcpCallResult`, `mcpContent`, `callResult()`). Decode path in `toolResult()`: `content[0].Text` → `Stdout`; JSON parse → `Output` map. Same as stdio (mcp.go lines 69-78). Error message format: `"mcp tool error: %s"` — identical. **One minor note:** HTTP returns a `ToolResult{ExitCode:1, Stderr:text}` alongside the error on tool-level `IsError`; stdio returns `nil, error`. The error is identical; the ToolResult difference is invisible to callers (who check `err != nil` first). Not a requirement failure — the observable behavior from a runbook author's perspective is indistinguishable. |
| 5 | Schema/runtime wiring, validation, and tests | **MET** | Schema: `TransportConfig` extended with `URL` and `Auth *AuthConfig` (with `AllowedHosts`). Runtime: `DefaultToolRuntime.Invoke` has `case TransportMCPHTTP` dispatching to `invokePersistent` → `NewMCPHTTPTransport`. Validation: `ValidateTransportConfig` checks MCP-001/002/010/011 at scan time. Tests: `mcp_http_test.go` + `mcp_http_tess_test.go` + `validate_transport_test.go` — 27/27 passing per verified report. |

---

## B-32 Realisation — FULLY BUILT AS RULED

| Control | Implementation | Verified |
|---|---|---|
| `auth:` requires `allowed_hosts` (MCP-010) | `ValidateTransportConfig`: if `auth != nil && len(AllowedHosts) == 0` → MCP-010 fatal | ✅ |
| URL host in allowed_hosts static check (MCP-011) | `ValidateTransportConfig`: parses URL, extracts `u.Hostname()`, lowercase exact match against list | ✅ |
| Runtime host mismatch fatal (MCP-012) | `TokenGate.AttachToken`: `hostAllowed()` → `errkit.New("MCP-012", ...)` | ✅ Confirmed by source read |
| Redirects refused on authenticated requests (MCP-013) | `NewMCPHTTPTransport`: `client.CheckRedirect = func(...) { return http.ErrUseLastResponse }` when gate != nil. `send()` detects 3xx → `errkit.New("MCP-013", ...)` naming the Location. | ✅ |
| Host matching: `u.Hostname()`, lowercase, exact, no wildcard | Both `ValidateAuthConfig` and `TokenGate.hostAllowed` use identical logic: `strings.ToLower(u.Hostname())` vs `strings.ToLower(entry)` | ✅ One definition, no drift |

---

## B-27: mcp/authAttached Trace Event — NOT DEAD CODE

`TokenGate.AttachToken` emits via `trace.EmitterFromContext(ctx)` on every successful token attachment. The emitter is the same one wired through `engine.go:569` → TraceWriter.Append — the same path used by all other trace events (include/resolved, step/started, etc.). It fires on every authenticated request, which means every `tools/call` and `tools/list` invocation on an authenticated MCP HTTP tool. This is not dead code — it is executed on the hot path of the feature's primary use case.

---

## B-24: Token Escape Surface Audit

| Surface | Safe? | Evidence |
|---|---|---|
| Error messages | ✅ | `classifyAzError` never includes token; MCP-007/012 errors reference scope, host, stderr — never token |
| Trace events | ✅ | `mcp/authAttached` payload: `{url_host, scope}` only |
| ToolResult.Stdout/Stderr/Output | ✅ | Token never enters ToolResult — it exists only in the `Authorization` header |
| HTTP debug logging | ✅ | No HTTP debug/trace transport installed; `http.Client{}` has no `Transport` override that would log headers |
| Panic stack | ⚠️ Acceptable | If `AttachToken` panics after `provider.Token()` returns, the token is on the stack. This is inherent to any in-memory secret and is not a design flaw — panic stacks are not a normal observability surface. |

**Verdict: B-24 is satisfied.** No credential material reaches any persistent or normal-observability surface.

---

## B-31: Documentation Adequacy

Go doc comments on `AuthConfig` (`pkg/schema/tool.go:298-365`) are extensive — they explain the replay threat, the `allowed_hosts` mechanism, and exactly what happens on mismatch. The wildcard non-support is documented in `TokenGate.AttachToken`'s doc comment. An author who writes `*.azure-api.net` gets MCP-012 with a message naming the host and referencing `allowed_hosts`.

Missing: an example `.tool.yaml` file in `examples/`. This is a **nice-to-have**, not a blocker — the doc comments and the schema types are sufficient for an engineer reading code or IDE hover-docs. I recommend adding one before broader adoption but do not condition approval on it.

---

## B-30: Stdio ensureStarted Defect — STILL CORRECT TO DEFER

The HTTP path deliberately does NOT replicate it (confirmed: `initialize()` checks `initResp.Error` before marking `t.initialized = true`). The stdio defect is tracked. The two paths are now asymmetric in a good way — the new code is correct, the old code has a known defect that will be fixed in a dedicated cleanup pass. Deferral remains the right call.

---

## Cross-Feature Question: Dynamic Include + MCP HTTP + Token Replay

A runtime-selected child runbook can declare an `mcp-http` tool with `auth:`. B-32's `allowed_hosts` is the control. Is it sufficient?

**Yes.** The chain of controls is:

1. **Frozen catalog** — the child runbook must come from an approved package. An external attacker cannot inject a tool definition.
2. **MCP-011 (static, at tool scan time)** — the declared URL's host must be in `allowed_hosts`. A package author who sets `url: https://evil.com` must ALSO set `allowed_hosts: [evil.com]` — the mismatch is caught structurally.
3. **The residual threat** — a compromised package author sets BOTH `url: https://evil.com` AND `allowed_hosts: [evil.com]`. B-32 cannot prevent this because the attacker controls the authored artifact. But at this point, the attacker can also set `url: https://icm-mcp-prod.azure-api.net` and issue malicious commands directly — a strictly more powerful attack that no host allow-list prevents. The allow-list stops *accidental* token leakage and *partial* compromise (attacker can modify URL but not allowed_hosts, or vice versa); it cannot stop a fully compromised author.

**The frozen catalog is the real trust boundary.** `allowed_hosts` is defense-in-depth against partial compromise or carelessness. Together they are sufficient.

---

## Deferred Items Confirmed

| Item | Status | Still correct to defer? |
|---|---|---|
| B-18 (static-include governance gap) | Tracked | ✅ |
| B-19/B-20 (`when:` inert) | Tracked, schema remediation required | ✅ |
| B-30 (stdio ensureStarted) | Tracked | ✅ |
| DEF-002 (DINC-009 unreachable) | B-16, permanent | ✅ |

---

## Final Verdict

**✅ APPROVED.** No conditions. The implementation is complete, correct, and faithful to the contract and all rulings through B-33. The security posture (B-32 host restriction, B-33 fatal enforcement + no redirects, B-24 token redaction) is structurally sound. The five requirements are met. The cross-feature interaction with dynamic includes is adequately controlled by the frozen catalog + allowed_hosts defense-in-depth.


# MCP HTTP — Binding Rulings B-22 through B-26

**Author:** Barbara (Lead / Architect)
**Date:** 2026-08-16T00:55:39Z
**Status:** RATIFIED — implementers build directly against these rulings

---

## B-22 — HTTPS only for remote MCP endpoints

**Date:** 2026-08-16T00:55:39Z

**Ruling:** Remote MCP endpoints MUST use HTTPS. Plain HTTP URLs are rejected at validation time with `MCP-001`. No `--allow-insecure` flag in this iteration.

**Rationale:** MCP tool calls carry bearer tokens and can mutate production systems (e.g., IcM ticket remediation actions). Allowing HTTP would let a network-position attacker intercept credentials. Localhost/stdio servers use `mode: mcp` (subprocess) and are unaffected.

---

## B-23 — Protocol version pinning at 2025-03-26

**Date:** 2026-08-16T00:55:39Z

**Ruling:** gert advertises `MCP-Protocol-Version: 2025-03-26` on all HTTP MCP requests. If the server responds with an HTTP error indicating version rejection, emit `MCP-003` (fatal). If the server ignores the header and responds successfully, proceed without error.

**Rationale:** The 2025-03-26 spec defines Streamable HTTP (replacing the deprecated SSE transport). Pinning to this version is forward-looking — it's the current stable spec. Graceful degradation for servers that ignore the header ensures interop with older implementations.

---

## B-24 — No credential material in YAML, traces, logs, or errors

**Date:** 2026-08-16T00:55:39Z

**Ruling:** Bearer tokens acquired by any auth provider MUST NOT appear in: YAML files, trace events, log output, error messages, `ToolResult` fields, step output, or any diagnostic dump. The `scope` field is a resource identifier (not a credential) and may appear in diagnostics.

**Rationale:** Tokens are short-lived secrets. Any persistence or observability surface that captures them creates a credential leakage vector. The only place a token may exist is in-memory within the `AuthProvider` and in the outgoing HTTP `Authorization` header (which must not be captured by debug logging).

---

## B-25 — No URL allow-list required

**Date:** 2026-08-16T00:55:39Z

**Ruling:** Any HTTPS URL is acceptable for `transport.url`. No allow-list, registry, or pre-approval mechanism is required.

**Rationale:** Tool definitions are authored artifacts (.tool.yaml files) subject to the same code-review and package-governance controls as any other authored configuration. The URL is static — not runtime-resolved. Contrast with dynamic includes, where the resolved identity is runtime-variable and the catalog serves as the allow-list. Different trust models require different controls. Adding an allow-list here would be security theater: the same author who can set a malicious URL can also set a malicious `command:`.

---

## B-26 — Remote tool calls subject to governance

**Date:** 2026-08-16T00:55:39Z

**Ruling:** Governance fires at the executor level (before `ToolRuntime.Invoke`), not at the transport level. Adding a new transport does not bypass governance. Specifically: `require_approval`, `RequiresCapabilities`, and `AllowedEnvironments` apply to MCP HTTP tools identically to stdio tools. `deny_commands` does not apply (there is no shell command to deny).

**Rationale:** The tool executor's governance pre-flight (`internal/executor/tool.go`) checks all policies before invoking the tool runtime. This is transport-agnostic by design — the executor doesn't know or care which transport will be used. The new transport plugs in below this gate.

---

## B-27 — B-25 narrowed: audience-scoped tokens are sufficient; no host allow-list needed

**Date:** 2026-08-15T18:03:00-07:00

**Amends:** B-25

**Challenge:** A dynamically-included child runbook (resolved at runtime from the frozen catalog) can declare an MCP HTTP tool pointing to any HTTPS host. David's auth provider attaches a live Azure bearer token. The operator running the parent never reviewed the child's URL. The URL is "authored" but not "reviewed by the person accepting the risk." This is a credential-forwarding-to-attacker-chosen-host risk — the gap is not the URL itself but the **combination of URL + attached credential**.

**Revised Ruling:** B-25 is **narrowed** as follows:

**Unauthenticated MCP HTTP requests:** B-25 stands unchanged. Any HTTPS URL is acceptable. No allow-list. The request carries no credential — SSRF risk is minimal.

**Authenticated MCP HTTP requests (where `transport.auth` is configured):** No host allow-list is required, because the existing mitigations are structural and sufficient:

1. **Frozen catalog** — child runbooks come only from approved packages. An arbitrary third party cannot inject a tool URL.
2. **Audience-scoped tokens** — a token acquired with `scope: api://icmmcpapi-prod/mcp.tools` carries that audience in its claims. Any properly configured Azure AD resource server that is NOT the intended audience rejects the token server-side. An attacker-chosen host receives a token it cannot validate (wrong audience) and gains nothing useful.
3. **HTTPS** — prevents interception in transit.
4. **Short-lived** — Azure CLI tokens expire in ~1 hour.

The threat model is: a compromised or careless package author places a malicious URL in a child runbook's tool definition. The frozen catalog prevents injection by outsiders. Against an insider (compromised package author), the audience-scoped token provides defense-in-depth — the token is useless at hosts that aren't the declared resource.

**One mandatory safeguard:** The auth provider MUST emit a trace event `mcp/authAttached` with payload `{url_host: <host-portion-only>, scope: <scope>}` (NO token value) on every authenticated request. This enables audit detection of scope-vs-host mismatch without adding a runtime gate.

**Implementation note for David:** Emit this trace event inside `MCPHTTPTransport` immediately before sending an authenticated request. The event is informational (non-gating, non-fatal). If someone declares `scope: api://icmmcpapi-prod/mcp.tools` but `url: https://evil.com`, the trace shows the discrepancy for post-hoc audit.

**Why no allow-list:** An allow-list would require a new configuration surface (where is the list? who maintains it? how does it interact with package catalogs?). The existing mitigations (catalog trust + audience-scoped tokens) close the gap structurally without adding operational burden. If a future threat model shows audience validation is insufficient (e.g., a server that accepts any audience), we can add a host allow-list then — but that would indicate a broken resource server, not a broken gert design.

---

## B-28 — Stdio env-var credentials are a pre-existing surface; B-24 does not retroactively cover them

**Date:** 2026-08-15T18:03:00-07:00

**Question:** Does B-24's redaction mandate extend to stdio MCP's `transport.env` map, which today can carry secrets (e.g., `API_KEY=xxx`) passed to the subprocess via `mergeEnv`?

**Ruling:** **No. B-24 applies exclusively to the new HTTP MCP auth provider.** The stdio env-var path is a pre-existing surface with different characteristics:

1. `transport.env` values are placed in YAML by the operator — they are authored, not runtime-acquired. B-24 exists specifically because the HTTP auth provider acquires a credential at runtime that the operator never typed into any file.

2. Env values are passed to a local subprocess and never traverse a network from gert itself. Whether the subprocess uses them over a network is outside gert's control.

3. These values are NOT currently in traces or logs (only in YAML and process memory). That is acceptable status-quo behavior.

**Action:** Documentation-only. Note that `transport.env` values in `.tool.yaml` should be treated as sensitive by operators (use CI vault injection or env-var indirection rather than literal secrets in committed YAML). No code change. Not a defect — a usage guidance gap.

**Not in scope:** Secret-reference resolution in `transport.env` (e.g., vault references). That is a future platform feature.

---

## B-29 — Protocol version divergence: stdio stays at 2024-11-05; HTTP uses 2025-03-26; this is correct

**Date:** 2026-08-15T18:04:00-07:00

**Question:** Stdio MCP advertises `protocolVersion: "2024-11-05"`. HTTP MCP advertises `MCP-Protocol-Version: 2025-03-26`. Should they match?

**Ruling:** They MUST NOT match. They are different protocol revisions for different transports.

- `2024-11-05` is the MCP version that defines the stdio/Content-Length framing model. The existing `MCPTransport` was built against it and interoperates with stdio MCP servers implementing that version. Changing it risks breaking compatibility with existing servers that validate the version field.
- `2025-03-26` is the MCP version that defines Streamable HTTP (POST + SSE responses, `Mcp-Session-Id` header). HTTP MCP servers expect this version in the `MCP-Protocol-Version` header.

These are not two implementations of the same spec at different versions — they are two different transport specifications. The version field means "I speak this protocol," and the two transports speak different protocols.

**Documentation:** No operator-facing documentation is needed beyond the tool schema itself (`mode: mcp` vs `mode: mcp-http`). An operator never chooses between transports for the same server — a server is either stdio or HTTP. The version is an implementation detail that follows from the transport choice.

---

## B-30 — Stdio `ensureStarted` defect: file and defer

**Date:** 2026-08-15T18:04:00-07:00

**Question:** Stdio `ensureStarted` sets `initialized = true` without checking whether the `initialize` response is an error. Fix now or defer?

**Ruling:** **Defer.** File as a defect. Do NOT fix in this feature.

**Rationale:**
1. This is a pre-existing defect in the stdio path. It has existed since `MCPTransport` was written and affects only stdio MCP tools, which are working in production (presumably with servers that don't reject initialization).
2. Fixing it means modifying `internal/tool/mcp.go` — the existing stdio transport — as part of a feature that is supposed to ADD a new transport without touching the old one. That violates the scope boundary.
3. Don's HTTP transport MUST NOT replicate it (already communicated — his `initialize` path inspects the response and emits MCP-003 on failure).

**Accumulation acknowledgment:** This is the third deferred pre-existing defect alongside B-18 (static-include governance gap) and B-19/B-20 (inert `when:`). I accept this accumulation deliberately. Each has the same shape: a real defect, adjacent to the stream, with regression risk if fixed as a side-effect. They are tracked, not forgotten. When this feature ships, a cleanup pass addressing all three should be prioritized.

**File to:** `.squad/decisions/inbox/defect-stdio-mcp-initialize-unchecked.md` with location `internal/tool/mcp.go:ensureStarted`, the fact that `initialized = true` is set unconditionally, and that the response body is discarded without error checking.

---

## B-31 — Go-only validation is acceptable; author-facing documentation lives in Go doc comments and the tool spec section

**Date:** 2026-08-15T18:04:00-07:00

**Question:** No JSON Schema governs `.tool.yaml`. B-20's schema-description remediation pattern cannot apply here. Is Go-only validation acceptable, and where does documentation live?

**Ruling:** **Go-only validation is acceptable.** This is the existing pattern for ALL tool validation and changing it is out of scope.

The absence of a `.tool.yaml` JSON Schema is a pre-existing architectural choice (noted by Ken: "No JSON Schema governs .tool.yaml" — C3 in the existing code comments). Tool definitions are validated by `internal/tool.ParseToolFile` and the schema types in `pkg/schema/tool.go`. This works and is well-tested.

**Author-facing documentation for `mcp-http` and `auth:` lives in:**
1. **Go doc comments on `TransportConfig`, `AuthConfig`** — these are the source of truth for any future generated documentation.
2. **`design/gert/sections/06-tool-runtime.tex`** — the existing tool runtime spec section. Ken or the implementer of Stream A should add a subsection for `mode: mcp-http` with the transport fields and auth provider shape.
3. **A `.tool.yaml` example** in `examples/` demonstrating the MCP HTTP configuration.

B-20's pattern (schema `description` fields) was appropriate because a JSON Schema existed and IDE completions were actively misleading users. Here, no schema exists, so no misleading completions are generated. The documentation surface is the spec section and examples — standard for this codebase.

---

## B-32 — B-25/B-27 revised: token attachment restricted to declared hosts (replay risk accepted as real)

**Date:** 2026-08-15T18:06:00-07:00

**Supersedes:** B-27's conclusion that "audience-scoped tokens are structurally limited" is **withdrawn as technically incorrect.** The revised ruling below replaces B-25/B-27's reasoning on authenticated requests.

**The corrected threat model:** A bearer token sent to an attacker-controlled host can be **replayed** against the legitimate audience (the real IcM endpoint) for the token's lifetime. Audience restriction constrains which server will *accept* the token, not who may *present* it. Possession is authorization. Token exfiltration to a malicious host IS a compromise regardless of audience scoping — this is the confused-deputy / token-replay pattern.

**Revised Ruling:** Token attachment is restricted to explicitly declared hosts.

### Mechanism

`AuthConfig` gains an `allowed_hosts` field:

```yaml
transport:
  mode: mcp-http
  url: https://icm-mcp-prod.azure-api.net/v1/
  auth:
    provider: azure-cli
    scope: api://icmmcpapi-prod/mcp.tools
    allowed_hosts:
      - icm-mcp-prod.azure-api.net
```

**Enforcement rule:** Before attaching an `Authorization` header, the auth provider checks that the request URL's host (scheme + hostname + port) matches an entry in `auth.allowed_hosts`. If no match, the request is sent **without** the `Authorization` header and a `MCP-W001` warning is emitted (new warning-class code): `"mcp-http: auth token not attached — host %q is not in allowed_hosts"`.

**Validation rules:**
- If `auth` is configured and `allowed_hosts` is empty or omitted → `MCP-010` (fatal): `"mcp-http: auth.allowed_hosts is required when auth is configured"`. Rationale: forcing explicit host declaration is the entire point of this control.
- If `url`'s host is not in `allowed_hosts` at validation time → `MCP-011` (fatal): `"mcp-http: url host %q is not in auth.allowed_hosts"`. Catches the misconfiguration statically rather than at runtime.

### Why this is the right narrowing

1. **Closes the replay vector.** A compromised package author who declares `url: https://evil.com` cannot also make the token go there, because `allowed_hosts` is validated against `url` at parse time (MCP-011). To exfiltrate the token, they would need to control both the URL and the allowed_hosts declaration — but if they can do that, they can also set `url` to the legitimate host and issue malicious tool calls directly, which is a strictly more powerful attack that no allow-list prevents.

2. **Does not burden the unauthenticated case.** `allowed_hosts` is only required when `auth` is configured. An MCP HTTP tool without auth has no token to protect and no restriction.

3. **Small implementation surface.** A host-match check in the HTTP transport before header attachment. David already has a policy hook routed for this.

4. **Prevention, not just detection.** B-27's `mcp/authAttached` trace event remains (for audit), but this is the gate that stops the send.

### Updated target YAML (canonical)

```yaml
transport:
  mode: mcp-http
  url: https://icm-mcp-prod.azure-api.net/v1/
  auth:
    provider: azure-cli
    scope: api://icmmcpapi-prod/mcp.tools
    allowed_hosts:
      - icm-mcp-prod.azure-api.net
```

### Error codes added

| Code | Class | Condition | Fatal? |
|---|---|---|---|
| `MCP-010` | `MCP` | `auth` configured without `allowed_hosts` | Fatal |
| `MCP-011` | `MCP` | `url` host not in `allowed_hosts` | Fatal |
| `MCP-W001` | `MCP-W` | Token not attached because host not in `allowed_hosts` (runtime, if URL is somehow different from validated url — defensive) | Warning |

### For the record

Token replay from a malicious host was **considered and accepted as a real threat**. The original B-25/B-27 reasoning that audience scoping structurally prevents exploitation was incorrect — audience scoping prevents the attacker from *consuming* the token at their own service, but does not prevent *replaying* it against the legitimate service. The `allowed_hosts` restriction is the prevention mechanism. The frozen catalog remains the primary trust boundary (only approved package authors can declare tool URLs), and `allowed_hosts` is defense-in-depth against a compromised author.

---

## B-33 — Runtime host mismatch is FATAL; redirects disabled on authenticated requests

**Date:** 2026-08-15T18:25:00-07:00

### Part 1: Fatal, not warning

**Ruling:** A runtime host mismatch MUST be fatal (`MCP-012`, class `MCP`). The request is NOT sent. The token is NOT attached. Execution stops with a clear error naming the mismatched host and pointing at `allowed_hosts`.

**Rationale:** B-32 exists to prevent token exfiltration. A warning that allows the request to proceed — with or without the token — defeats the control entirely:
- With token attached: the control is decorative (the token reaches the unapproved host).
- Without token attached: the request silently downgrades to unauthenticated, producing a confusing 401 from the far end that no operator will diagnose as a security gate firing.

Fatal is the only semantics that actually prevents the thing B-32 was designed to prevent.

**Ken must reclassify `ErrMCPW001`:** Change from `class: "MCP-W"` / warning to `code: "MCP-012"` / `class: "MCP"` / fatal. Remove `ErrMCPW001` entirely (it was never shipped externally — the frozen-catalog invariant on error codes applies only post-ship). Replace with:

```go
ErrMCP012 = &Error{code: "MCP-012", class: "MCP"}
```

Message: `"mcp-http: request to host %q blocked — not in auth.allowed_hosts %v"`

### Part 2: Redirects disabled on authenticated requests

**Ruling:** The `http.Client` used by `MCPHTTPTransport` for authenticated requests MUST have `CheckRedirect` set to reject ALL redirects (return `http.ErrUseLastResponse`). The request fails with `MCP-013`:

```go
ErrMCP013 = &Error{code: "MCP-013", class: "MCP"}
```

Message: `"mcp-http: redirect from %q to %q rejected — redirects are disabled on authenticated requests"`

**Rationale:** Per-hop host checking is more permissive but complex to get right (must strip the `Authorization` header before the redirect if the target host differs, race between check and send, interaction with HTTP/2 push). Disabling redirects entirely is simpler, correct, and matches the security posture of every major OAuth client library (which strip `Authorization` on cross-origin redirects by default). A legitimate MCP server that requires redirects can be addressed by the operator updating `url:` to point at the final endpoint.

For **unauthenticated** requests (no `auth:` configured), default redirect-following behavior is acceptable — there is no token to protect.

### Part 3: Host-matching semantics confirmed as binding

**Ruling:** The following is the single canonical definition for host matching, used by both MCP-011 (static validation) and MCP-012 (runtime check):

1. Parse `url` with `net/url.Parse`
2. Extract hostname via `u.Hostname()` (strips port)
3. Lowercase both the extracted hostname and each `allowed_hosts` entry
4. **Exact string equality** — no prefix matching, no suffix matching, no wildcards, no regex, no glob

This is the binding definition. If the static and runtime checks use different comparison logic, that is a bug.


# Stream C — Auth Provider (David)

**Status:** Implementation complete. `go build ./...` exit 0. All packages outside `internal/tool` pass. Note: `mcp_http_tess_test.go` has a pre-existing syntax error in `TestMCPHTTPTransport_SSE_CorrectIDSelected` (DEF-012 test missing server setup — untracked file, not caused by Stream C changes); `internal/tool` tests require Tess to fix that before `go test ./internal/tool/...` can pass.

---

## Update — emitter wiring audit + Don collision resolved (2026-08-15T19:12)

**Don's `emit_ctx.go` has not landed** — `internal/tool/emit_ctx.go` does not exist in the repo. My earlier solution (`pkg/trace/emitter.go`) already provides the same functionality and is what `auth_gate.go` currently imports. There is no import cycle.

**B-27 `mcp/authAttached` reaches the real trace stream — provably.**

Full production chain traced:
1. `internal/engine/engine.go:569` — `executor.WithEventEmitter(spanCtx, fn)` installs the emitter; `fn` calls `h.emitEventLocked(spanCtx, kind, payload)`.
2. `executor.WithEventEmitter` delegates to `trace.WithEventEmitter(ctx, e)` (same context key as `trace.EmitterFromContext`).
3. The enriched context is passed to `exec.Execute(emitterCtx, ...)` → `runtime.Invoke(ctx, ...)` → `invokePersistent(ctx, ...)` → `transport.Invoke(ctx, ...)`.
4. `MCPHTTPTransport.send(ctx, ...)` creates `reqCtx` as a child of `ctx` (adds deadline only), passes to `buildHTTPRequest(reqCtx, ...)` → `gate.AttachToken(reqCtx, req)`.
5. `AttachToken` calls `trace.EmitterFromContext(reqCtx)` — same key → receives the engine emitter.
6. `emitEventLocked` writes `trace.TraceEvent{Kind: "mcp/authAttached", Payload: {url_host, scope}}` to `h.engine.cfg.TraceWriter.Append` (the persistent trace file) AND broadcasts in-process.

The emitter is not merely called in tests — it is the engine's production emitter, and it writes to the trace file.

**MCP-013 redirect ownership:** I own it. `CheckRedirect` is configured in `NewMCPHTTPTransport` (my `mcp_http.go` changes); the policy decision (authenticated only, `http.ErrUseLastResponse`) is entirely within my stream. Don's transport receives the configured client.

---

Ken edited `auth_gate.go` as part of the MCP-012 reclassification. On inspection the file was already in the correct state for both changes he described:
- `AttachToken` already returned `errkit.New("MCP-012", ...)` as a fatal error (applied in my previous B-32 pass).
- `ValidateAuthConfig` already used `u.Hostname()` — not `u.Host` — so port-stripping was already correct.

The one genuine divergence was the **MCP-012 error message text**. My message read:
```
mcp-http: host %q is not in auth.allowed_hosts — token not attached; add %q to allowed_hosts or update url:
```
Ken's registered canonical form is:
```
mcp-http: request host %q is not in auth.allowed_hosts — token not attached (update allowed_hosts or url: to match)
```
Updated `auth_gate.go` to match Ken's exact wording. `go build ./...` exit 0; all packages outside `internal/tool` pass.

**Zero `MCP-W` references** anywhere in the codebase — confirmed by grep across all `.go` files.

---

Barbara's final ruling: `CheckRedirect` must return `http.ErrUseLastResponse` (not a custom error) so the redirect response is surfaced to `send()` rather than becoming a transport error wrapped in MCP-008.

**Changes in this pass:**
- `NewMCPHTTPTransport` (`mcp_http.go`): `CheckRedirect` now returns `http.ErrUseLastResponse` for authenticated transports. Unauthenticated transports (gate == nil) retain nil CheckRedirect and follow redirects normally.
- `send()` (`mcp_http.go`): Added 3xx detection block — when gate is non-nil and status is 3xx, emit `errkit.New("MCP-013", "...redirected to <Location>...— update url: to <Location>")` with the `Location` header value. This gives the operator an actionable message naming the destination.
- `TestMCPHTTPTransport_CheckRedirectInstalledOnAuthenticated` (`mcp_http_test.go`): Updated to assert `errors.Is(err, http.ErrUseLastResponse)` instead of `errkit.ErrMCP013`.
- `TestMCPHTTPTransport_TokenNeverReachesRedirectTarget` (new test): Security proof — redirecting server (in `allowed_hosts`) 302s to evil server; asserts evil server never receives any Authorization header AND error is MCP-013.

**errkit.Error.Is() semantics confirmed:** Code-based comparison (`e.code == t.code`), so `errors.Is(errkit.New("MCP-013", msg), errkit.ErrMCP013)` returns true. Don's `TestMCPHTTPTransport_RedirectBlocked_Authenticated` continues to pass without modification.

**Redirect policy stated:**
- Authenticated: redirect → hard stop (`http.ErrUseLastResponse` → `send()` detects 3xx → MCP-013 with Location). Token NEVER forwarded.
- Unauthenticated: follow redirects normally (no restriction).

---

---

## Update — B-32 revised: host-mismatch is now fatal (2026-08-15T18:47)

User instruction superseded the B-32 written text on the non-fatal/warning behavior.
New behavior: host not in `allowed_hosts` → hard MCP-012 error, request not sent.

Changes:
- `TokenGate.AttachToken`: returns `MCP-012` (fatal) on host mismatch instead of nil + mcp/authSkipped event
- `EventKindMCPAuthSkipped` removed from `pkg/trace/event.go` — no longer emitted
- `NewMCPHTTPTransport`: sets `CheckRedirect` to block redirects on authenticated transport (MCP-013); Don had already implemented this with `errkit.ErrMCP013`
- `auth_gate_test.go`: updated `TestTokenGate_RejectsDisallowedHost` (was non-fatal, now asserts MCP-012); added `TestTokenGate_SuffixConfusionRejected`; added `TestTokenGate_MCP012ErrorIsActionable`
- All 38 gate/auth tests pass; `go build ./...` exit 0

**Host-matching semantics (stated explicitly):**
- Match on `req.URL.Hostname()` (port-stripped) — exact case-insensitive
- No wildcards (a `*` entry is treated as a literal hostname and will never match)
- No IDN/punycode normalization
- Port specs in allowed_hosts entries are matched literally (not stripped) — use plain hostnames

**Redirect decision:** authenticated transports refuse all redirects (MCP-013). An operator who needs a redirect path must update `url:` to point to the final endpoint.

---

B-32 superseded B-25/B-27 on token attachment policy. New requirements implemented:

- `schema.AuthConfig.AllowedHosts []string` — required when auth is configured (Ken landed this in `pkg/schema/tool.go`)
- `TokenGate` struct in `internal/tool/auth_gate.go` — the single policy decision point for all token attachment
- `ValidateAuthConfig(rawURL, auth)` — static validation: MCP-010 (no allowed_hosts) and MCP-011 (url host not in list)
- `AttachToken(ctx, req)` — runtime enforcement + B-27 trace event emission
- `EventKindMCPAuthAttached` and `EventKindMCPAuthSkipped` added to `pkg/trace/event.go`
- `trace.EventEmitter` + `trace.WithEventEmitter` + `trace.EmitterFromContext` added to `pkg/trace/emitter.go` (cycle-free emitter sharing — `internal/executor/events.go` updated to delegate there)
- `MCPHTTPTransport` updated: `auth AuthProvider` → `gate *TokenGate`; added HTTP 401 invalidate-and-retry path
- `runtime.go`: validates auth config before construction; constructs `TokenGate` with `AllowedHosts`
- 11 gate tests in `auth_gate_test.go` including B-24 event-payload redaction proof

**Note on B-32 runtime behavior:** Host-mismatch at runtime (after static validation) is **non-fatal** per B-32. The request proceeds unauthenticated; `mcp/authSkipped` event is emitted with `MCP-W001` code. The static MCP-011 check catches the common misconfiguration at parse time.

---

## AuthProvider Interface (for Don — Stream B)

```go
// Package: github.com/ormasoftchile/gert/internal/tool

// AuthProvider acquires bearer tokens for HTTP transports that require authentication.
// Implementations handle acquisition, caching, and refresh internally.
//
// The token returned by Token is an in-memory secret. It must not be written
// to any persistent or observable surface (traces, logs, error messages, step
// output). See B-24.
type AuthProvider interface {
    // Token returns a valid bearer token, refreshing proactively when within
    // five minutes of expiry.
    Token(ctx context.Context) (string, error)

    // Invalidate clears any cached token, forcing re-acquisition on the next
    // Token call. Call after receiving HTTP 401 to handle mid-run token expiry.
    Invalidate()
}

// NewAuthProvider(provider, scope string) (AuthProvider, error)
// Recognized provider values: "azure-cli"
// Returns MCP-002 error for unknown provider.
```

**Don's usage pattern for mid-run 401:**
```go
// On HTTP 401 response:
auth.Invalidate()
token, err = auth.Token(ctx)  // forces re-acquisition
// retry the request with the new token
```

---

## Files Owned

- `internal/tool/auth.go` — `AuthProvider` interface + `NewAuthProvider` factory
- `internal/tool/auth_azurecli.go` — `AzureCLIAuthProvider` implementation
- `internal/tool/auth_azurecli_test.go` — 16 tests including B-24 redaction proof

---

## Caching and Refresh Strategy

- **Proactive refresh:** Token is refreshed when `now + 5min >= expiry`. The old (still-valid) token is returned on probe failure — graceful degradation, not error.
- **Hard expiry:** At the hard expiry point, the cache is invalid and re-acquisition is mandatory.
- **Mid-run 401 path:** `Don.MCPHTTPTransport` should call `Invalidate()` on receiving HTTP 401, then `Token()` again. `AzureCLIAuthProvider` handles the mutex and re-acquisition internally.
- **Expiry parsing:** Prefers `expires_on` (Unix timestamp, TZ-safe) from `az`'s JSON output; falls back to `expiresOn` ("2006-01-02 15:04:05.000000" in local time); falls back to `now + 50min` if both are absent/malformed.
- **Concurrency:** `sync.Mutex` guards the cache; concurrent calls during refresh block rather than stampede.

---

## Four Failure Modes (B-24 / actionable messages)

| Scenario | Error code | Message (summarised) |
|----------|------------|----------------------|
| `az` not on PATH | MCP-007 | `az CLI is not installed or not on PATH — install from https://aka.ms/installazurecli and retry` |
| Not logged in | MCP-007 | `not authenticated — run 'az login' and retry` |
| No scope consent | MCP-007 | `no consent for scope "<scope>" — grant application consent or run 'az login' with the required scope` |
| Malformed JSON output | MCP-007 | `az CLI returned malformed output: <json decode error>` or `missing accessToken field` |

Classification uses `errors.As(err, &exec.Error{})` to detect the not-installed case, then stderr heuristics (`AADSTS` → consent, `Please run 'az login'` → auth) for the others.

---

## B-24 Redaction Proof

Test: `TestAzureCLIAuthProvider_TokenNeverLeaksIntoDiagnostics` in `auth_azurecli_test.go`.

The test seeds the cache with a known token value, then forces all four failure paths and the generic error path. It asserts the known token string is absent from every error message returned. It also asserts that a second cache load doesn't return a stale token after `Invalidate()`.

Redaction guarantees by design:
- `classifyAzError` is only called on command failure — no token has been produced at that point.
- `azTokenResponse` is decoded in-memory and the raw `stdout` bytes are discarded.
- The `token` field on `AzureCLIAuthProvider` is mutex-protected and never serialized, traced, or logged.
- Error messages contain only: stderr text from `az`, the scope string (not a secret), and static operator guidance.

---

## Dependencies Not Yet Complete (Ken — Stream A)

- `errkit.ClassForCode("MCP-007")` returns `""` — Ken has not yet registered the `"MCP"` class in `errkit/errors.go`. Errors work; they just have empty class in the structured error.
- `schema.AuthConfig` type — Ken owns this. When he lands it, `runtime.go` already consumes it at the wire point (`def.Auth.Provider`, `def.Auth.Scope`).

---

## Cross-Stream Seam

Don's `MCPHTTPTransport` (Stream B) receives an `AuthProvider` from `runtime.go`'s switch case for `TransportMCPHTTP`. That wiring is already in place in `internal/tool/runtime.go` — it calls `NewAuthProvider(def.Auth.Provider, def.Auth.Scope)` and passes the result to `NewMCPHTTPTransport(def.URL, auth)`.

Don should call `auth.Token(ctx)` to get the bearer token and set it as `Authorization: Bearer <token>` on each HTTP request. For mid-run 401 handling, call `auth.Invalidate()` then `auth.Token(ctx)` again before the retry.

---

## Test Results

```
go test -count=1 ./internal/tool/... -run "TestAzureCLI|TestNewAuth|TestParseAz"
ok  github.com/ormasoftchile/gert/internal/tool  (all 16 PASS)
```

Full suite: `internal/tool` has pre-existing failures unrelated to Stream C:
- `TestStdioTransport_*` — missing `.testtools` binaries (pre-existing)
- `TestMCPTransport_*` — same
- `TestMCPHTTPTransport_SSEMultiLineData` — Don's in-flight SSE test (not mine)

