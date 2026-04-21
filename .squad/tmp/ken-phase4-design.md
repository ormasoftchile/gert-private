# Phase 4 Design: Governance Engine

**Author:** Ken (Software Architect)
**Date:** 2026-04-20
**Status:** Ready for implementation
**Implementor:** Brian

---

## Package Structure

### Public contracts (`pkg/governance/`)

| File | Purpose |
|------|---------|
| `policy.go` | **EXISTS.** `GovernancePolicy` interface, `AllowList`, `DenyList`, `PolicyRule`, `RedactionPattern` |
| `doc.go` | **EXISTS.** Package documentation |
| `evaluator.go` | **NEW.** `PolicyEvaluator` interface, `StepInfo`, `EvaluationResult`, `MatchedRule` types |
| `evidence.go` | **NEW.** `Evidence` struct, `ApprovalRecord` struct (value types, JSON-serialisable) |
| `approval.go` | **NEW.** `ApprovalGate` interface |

### Private implementations (`internal/governance/`)

| File | Purpose |
|------|---------|
| `evaluator.go` | Concrete `PolicyEvaluator` — orchestrates command check, env-var filter, approval gate |
| `builder.go` | `BuildPolicy` — constructs `GovernancePolicy` from `schema.GovernanceConfig`; merges runbook-level + step-level |
| `evidence.go` | `EvidenceCollector` — constructs `Evidence` records from evaluation results |
| `approval.go` | `NoOpApprovalGate`, `TerminalApprovalGate` implementations |
| `evaluator_test.go` | 14+ test cases (see Test Plan) |
| `builder_test.go` | Builder and merge tests |

### Test fakes (`pkg/testutil/`)

| File | Purpose |
|------|---------|
| `fake_governance_policy.go` | **NEW.** `FakeGovernancePolicy` implementing `governance.GovernancePolicy` |
| `fake_approval_gate.go` | **NEW.** `FakeApprovalGate` implementing `governance.ApprovalGate` |

### Import dependency graph (no cycles)

```
pkg/governance       ← no imports from gert (leaf package)
pkg/schema           ← no imports from gert (leaf package)
pkg/trace            ← no imports from gert (leaf package)

internal/governance  → pkg/governance, pkg/schema, pkg/trace
                     ✗ MUST NOT import internal/engine

internal/engine      → pkg/engine, pkg/governance, pkg/trace
                       (wiring: receives PolicyEvaluator + ApprovalGate via EngineConfig)

pkg/engine           → pkg/governance (for ExecutionPlan.Governance field)
```

---

## Interface Contracts

### StepInfo (governance-local input type)

```go
// pkg/governance/evaluator.go

// StepInfo carries the step metadata the evaluator needs.
// The engine converts ResolvedStep → StepInfo before calling Evaluate.
// This avoids an import from pkg/governance → pkg/engine.
type StepInfo struct {
    ID      string
    Kind    string            // "cli", "tool", etc.
    Command string            // argv[0] for cli steps; empty for non-command steps
    EnvVars map[string]string // environment variables passed to the step
}
```

**Rationale:** `pkg/engine.ResolvedStep` cannot be referenced from `pkg/governance` without creating a cycle (`pkg/engine` already imports `pkg/governance` for `ExecutionPlan.Governance`). `StepInfo` is a small, focused projection that carries exactly what governance needs: the step ID (for trace correlation), the kind (for rule filtering), the command (for allow/deny matching), and the env vars (for env-var filtering).

### PolicyEvaluator

```go
// pkg/governance/evaluator.go

// PolicyEvaluator performs governance pre-flight checks on steps.
// The engine calls Evaluate before invoking any StepExecutor.
//
// Contract:
//   - Returns EvaluationResult with Allowed=true if the step may proceed
//   - Returns EvaluationResult with Denied=true if the step is blocked (deny-wins)
//   - Returns EvaluationResult with RequiresApproval=true if a gate must be cleared
//   - error is reserved for infrastructure failures (evaluator crash, not policy denial)
//   - Deny ALWAYS takes precedence over allow (invariant enforced by all implementations)
type PolicyEvaluator interface {
    Evaluate(ctx context.Context, step StepInfo) (EvaluationResult, error)
}
```

### EvaluationResult

```go
// pkg/governance/evaluator.go

// EvaluationResult is the outcome of a pre-flight governance check.
// It carries enough information for the engine to act without knowing policy internals.
//
// Interpretation rules (engine MUST follow these):
//   1. If Denied == true: do NOT execute the step. Return StepResult with Status "denied".
//   2. If RequiresApproval == true AND Denied == false: call ApprovalGate before proceeding.
//   3. If Allowed == true AND Denied == false AND RequiresApproval == false: proceed normally.
//   4. Use FilteredEnvVars (not the original) as the step's environment.
//   5. Attach Evidence to the trace event.
type EvaluationResult struct {
    // Allowed is true when the step passes all governance checks.
    Allowed bool `json:"allowed"`

    // Denied is true when a deny rule matched. Deny ALWAYS wins over allow.
    Denied bool `json:"denied"`

    // DenyReason is a human-readable explanation when Denied is true.
    DenyReason string `json:"deny_reason,omitempty"`

    // RequiresApproval is true when the policy or step has require_approval set.
    // Only meaningful when Denied is false.
    RequiresApproval bool `json:"requires_approval,omitempty"`

    // MatchedRules lists every rule that matched during evaluation (both allow and deny).
    MatchedRules []MatchedRule `json:"matched_rules,omitempty"`

    // BlockedEnvVars lists the names of environment variables removed by policy.
    // Names only — never values (values must never appear in traces or logs).
    BlockedEnvVars []string `json:"blocked_env_vars,omitempty"`

    // FilteredEnvVars is the sanitised environment the step should execute with.
    // This is the input EnvVars minus any variables blocked by policy.
    FilteredEnvVars map[string]string `json:"filtered_env_vars,omitempty"`

    // Evidence is the structured governance record to attach to the trace.
    Evidence Evidence `json:"evidence"`
}
```

### MatchedRule

```go
// pkg/governance/evaluator.go

// MatchedRule records a single rule match during evaluation.
type MatchedRule struct {
    // RuleID identifies the rule (e.g., "allow:kubectl", "deny:rm").
    RuleID string `json:"rule_id"`

    // Kind is the rule type: "allow", "deny", "env_deny".
    Kind string `json:"kind"`

    // Pattern is the glob or string that matched.
    Pattern string `json:"pattern"`

    // Matched is the actual value that was matched against the pattern.
    Matched string `json:"matched,omitempty"`
}
```

### Evidence

```go
// pkg/governance/evidence.go

// Evidence is a structured record of governance evaluation for audit and tracing.
// It is a VALUE type — no pointers, no interfaces, fully JSON-serialisable.
// Attach to trace events as structured metadata.
type Evidence struct {
    // EvaluatedAt is the RFC3339 timestamp of when evaluation occurred.
    EvaluatedAt string `json:"evaluated_at"`

    // StepID is the step that was evaluated.
    StepID string `json:"step_id"`

    // PolicyApplied describes the policy source (e.g., "runbook:governance", "merged:runbook+step").
    PolicyApplied string `json:"policy_applied"`

    // RulesEvaluated is the total count of rules checked.
    RulesEvaluated int `json:"rules_evaluated"`

    // RulesMatched lists every rule that produced a match.
    RulesMatched []MatchedRule `json:"rules_matched"`

    // Outcome is "allowed", "denied", or "approval_required".
    Outcome string `json:"outcome"`

    // DenyReason is set when Outcome == "denied".
    DenyReason string `json:"deny_reason,omitempty"`

    // ApprovalRecord is set when approval was requested and received.
    ApprovalRecord *ApprovalRecord `json:"approval_record,omitempty"`

    // BlockedEnvVars lists env var names stripped from the step's environment.
    BlockedEnvVars []string `json:"blocked_env_vars,omitempty"`

    // RedactionsApplied is the count of redaction patterns that matched output.
    // Set during post-execution redaction pass, not during pre-flight.
    RedactionsApplied int `json:"redactions_applied"`
}
```

### ApprovalRecord

```go
// pkg/governance/evidence.go

// ApprovalRecord is the result of an approval gate decision.
// Value type, JSON-serialisable, suitable for embedding in Evidence and trace events.
type ApprovalRecord struct {
    // Approver is the identity of the person who approved (e.g., "alice@corp.com").
    Approver string `json:"approver"`

    // ApprovedAt is the RFC3339 timestamp of the approval decision.
    ApprovedAt string `json:"approved_at"`

    // Token is an opaque string for audit trail correlation.
    // Generated by the ApprovalGate implementation (e.g., UUID).
    Token string `json:"token"`
}
```

### ApprovalGate

```go
// pkg/governance/approval.go

// ApprovalGate requests human approval before a step executes.
// Implementations are injected into the engine at construction time.
//
// Contract:
//   - RequestApproval blocks until approval is granted or denied
//   - Returns ApprovalRecord on approval, error on rejection or timeout
//   - The context carries cancellation (e.g., run cancelled while waiting)
//   - Implementations MUST NOT call back into the engine (no re-entrancy)
type ApprovalGate interface {
    RequestApproval(ctx context.Context, stepID string, reason string) (ApprovalRecord, error)
}
```

### GovernancePolicy additions

No changes to the existing `GovernancePolicy` interface. The `PolicyEvaluator` wraps it:

```go
// internal/governance/evaluator.go

// evaluator is the concrete PolicyEvaluator implementation.
type evaluator struct {
    policy       governance.GovernancePolicy
    approvalGate governance.ApprovalGate
}

func NewEvaluator(policy governance.GovernancePolicy, gate governance.ApprovalGate) governance.PolicyEvaluator {
    return &evaluator{policy: policy, approvalGate: gate}
}
```

### New StepStatus constant

```go
// pkg/engine/run.go — add to existing StepStatus constants

StepStatusDenied StepStatus = "denied"
```

And corresponding `StepOutcome`:
```go
StepOutcomeDenied StepOutcome = "denied"
```

---

## Policy Merge Rules

### Sources

1. **Runbook-level:** `schema.GovernanceConfig` from `Runbook.Governance`
2. **Step-level:** `schema.GovernanceConfig` from `Step.Governance` (if present; steps currently don't have this field — see Open Questions)

### Merge semantics (step overrides runbook)

| Field | Merge Rule | Rationale |
|-------|-----------|-----------|
| `AllowCommands` | **Step replaces runbook** if step specifies any. Empty step list = inherit runbook. | Step knows its command; runbook is a broad default. |
| `DenyCommands` | **Union** (runbook ∪ step). Both lists are combined. | Deny is additive — more denials are always safer. |
| `DenyEnvVars` | **Union** (runbook ∪ step). | Same as deny commands — additive. |
| `RequireApproval` | **OR** (runbook ∨ step). If either requires approval, approval is required. | Approval cannot be removed by a step. |
| `Rules` | **Concatenate** (runbook first, then step). Later rules do not override earlier rules — deny-wins applies across all rules. | Preserves evaluation order. |
| `Redact` | **Concatenate** (runbook first, then step). All patterns applied. | Redaction is additive. |

### Builder function signature

```go
// internal/governance/builder.go

// BuildPolicy constructs a concrete GovernancePolicy from one or more GovernanceConfig sources.
// Configs are applied in order: earlier configs are lower precedence.
// Merge semantics: deny is additive, allow is replaceable, approval is OR'd.
func BuildPolicy(configs ...*schema.GovernanceConfig) governance.GovernancePolicy

// BuildEvaluator constructs a PolicyEvaluator from schema configs and an approval gate.
func BuildEvaluator(gate governance.ApprovalGate, configs ...*schema.GovernanceConfig) governance.PolicyEvaluator
```

### Concrete GovernancePolicy implementation

```go
// internal/governance/builder.go

// policy is the concrete GovernancePolicy produced by BuildPolicy.
type policy struct {
    allow       []string          // glob patterns (empty = permissive)
    deny        []string          // glob patterns (always evaluated first)
    denyEnvVars []string          // glob patterns for env var names
    redact      []*governance.RedactionPattern
    requireApproval bool
}
```

### Evaluation order within Evaluate()

1. **Deny check** — iterate deny patterns against `StepInfo.Command`. First match → `Denied=true`, short-circuit.
2. **Allow check** — if allow list is non-empty, check command against allow patterns. No match → `Denied=true`.
3. **Env-var filter** — iterate env var deny patterns against `StepInfo.EnvVars` keys. Build `FilteredEnvVars` and `BlockedEnvVars`.
4. **Approval check** — if `requireApproval` is true, set `RequiresApproval=true`.
5. **Build Evidence** — populate `Evidence` struct with all match data.

Deny-wins invariant: if the command matches BOTH an allow and a deny pattern, the deny wins. This is enforced by evaluating deny FIRST and short-circuiting.

---

## Engine Integration Points

### EngineConfig additions

```go
// pkg/engine/engine.go — extend EngineConfig

type EngineConfig struct {
    // ... existing fields ...

    // GovernanceEvaluator performs pre-flight governance checks on each step.
    // Optional: if nil, all steps are allowed (no governance).
    GovernanceEvaluator governance.PolicyEvaluator

    // ApprovalGate handles approval requests for steps requiring approval.
    // Optional: if nil, approval-required steps fail with an error.
    ApprovalGate governance.ApprovalGate
}
```

### Integration point 1: executeStep pre-flight (engine.go ~line 168)

**Location:** `internal/engine/engine.go`, `executeStep()` method, AFTER `step/started` emission but BEFORE executor lookup.

```go
// executeStep runs a single step and returns the result.
func (h *runHandle) executeStep(ctx context.Context, step enginepkg.ResolvedStep) (*enginepkg.StepResult, error) {
    // ... existing dispatch for parallel/wait_for_event ...

    // Standard step: emit step/started
    h.emitEventLocked(ctx, trace.EventKindStepStarted, map[string]any{
        "step_id": step.ID,
        "kind":    step.Kind,
    })

    startedAt := time.Now()

    // ── GOVERNANCE PRE-FLIGHT (Phase 4) ──────────────────────────────
    if h.engine.cfg.GovernanceEvaluator != nil {
        stepInfo := governance.StepInfo{
            ID:      step.ID,
            Kind:    step.Kind,
            Command: extractCommand(step),    // helper: extracts argv[0] from CLISpec
            EnvVars: extractEnvVars(step),    // helper: extracts env from CLISpec
        }

        evalResult, evalErr := h.engine.cfg.GovernanceEvaluator.Evaluate(ctx, stepInfo)
        if evalErr != nil {
            return h.failRun(ctx, step.ID, fmt.Errorf("governance evaluation failed: %w", evalErr))
        }

        // Emit governance/command_checked trace event.
        h.emitEventLocked(ctx, trace.EventKindGovernanceCommandChecked, map[string]any{
            "step_id":       step.ID,
            "command":       stepInfo.Command,
            "allowed":       evalResult.Allowed,
            "denied":        evalResult.Denied,
            "matched_rules": evalResult.MatchedRules,
            "evidence":      evalResult.Evidence,
        })

        // DENIED: return immediately, do NOT call executor.
        if evalResult.Denied {
            result := &enginepkg.StepResult{
                StepID:      step.ID,
                Status:      enginepkg.StepStatusDenied,
                Outcome:     enginepkg.StepOutcomeDenied,
                StartedAt:   startedAt,
                CompletedAt: time.Now(),
                DurationMs:  time.Since(startedAt).Milliseconds(),
                Error:       &GovernanceDeniedError{
                    StepID: step.ID,
                    Reason: evalResult.DenyReason,
                },
                Output: map[string]any{
                    "governance_evidence": evalResult.Evidence,
                },
            }
            h.run.StepResults[step.ID] = result
            h.emitEventLocked(ctx, trace.EventKindStepFailed, map[string]any{
                "step_id":    step.ID,
                "error":      evalResult.DenyReason,
                "error_type": "governance",
            })
            return result, nil
        }

        // APPROVAL REQUIRED: call the gate, record the result.
        if evalResult.RequiresApproval {
            h.emitEventLocked(ctx, trace.EventKindGovernanceApprovalRequested, map[string]any{
                "step_id": step.ID,
                "reason":  "policy requires approval",
            })

            if h.engine.cfg.ApprovalGate == nil {
                return h.failRun(ctx, step.ID, fmt.Errorf("step %s requires approval but no ApprovalGate is configured", step.ID))
            }

            // Release mutex while waiting for human approval.
            h.mu.Unlock()
            record, approvalErr := h.engine.cfg.ApprovalGate.RequestApproval(ctx, step.ID, "governance policy requires approval")
            h.mu.Lock()

            if approvalErr != nil {
                return h.failRun(ctx, step.ID, fmt.Errorf("approval denied for step %s: %w", step.ID, approvalErr))
            }

            h.emitEventLocked(ctx, trace.EventKindGovernanceApprovalReceived, map[string]any{
                "step_id":  step.ID,
                "approver": record.Approver,
                "token":    record.Token,
            })

            // Update evidence with the approval record.
            evalResult.Evidence.ApprovalRecord = &record
        }

        // Replace step env vars with filtered set.
        applyFilteredEnvVars(step, evalResult.FilteredEnvVars)
    }
    // ── END GOVERNANCE PRE-FLIGHT ────────────────────────────────────

    // Look up executor. (existing code continues unchanged)
    exec := h.engine.cfg.Executors.Lookup(step.Kind)
    // ...
}
```

### Integration point 2: post-execution redaction (engine.go ~line 225)

**Location:** `internal/engine/engine.go`, `executeStep()` method, AFTER executor returns but BEFORE result is recorded or trace event is emitted.

```go
    // Execute the step (existing code)
    h.mu.Unlock()
    result, execErr := exec.Execute(ctx, step, h.run.Vars)
    h.mu.Lock()

    // ── POST-EXECUTION REDACTION (Phase 4) ────────────────────────────
    if h.engine.cfg.GovernanceEvaluator != nil && h.run.Plan.Governance != nil {
        patterns := h.run.Plan.Governance.RedactionPatterns()
        if len(patterns) > 0 {
            redactionCount := applyRedaction(result.Output, patterns)
            if redactionCount > 0 {
                h.emitEventLocked(ctx, trace.EventKindGovernanceRedactionApplied, map[string]any{
                    "step_id":    step.ID,
                    "rule_count": redactionCount,
                })
            }
        }
    }
    // ── END REDACTION ─────────────────────────────────────────────────

    // Populate timing if executor didn't set it. (existing code continues)
```

### Helper functions (engine.go)

```go
// extractCommand returns argv[0] from a CLI step spec.
// Returns "" for non-CLI steps.
func extractCommand(step enginepkg.ResolvedStep) string {
    if cli, ok := step.Spec.(*schema.CLISpec); ok && cli != nil {
        return cli.Command
    }
    return ""
}

// extractEnvVars returns the environment variables from a CLI step spec.
// Returns nil for non-CLI steps.
func extractEnvVars(step enginepkg.ResolvedStep) map[string]string {
    if cli, ok := step.Spec.(*schema.CLISpec); ok && cli != nil {
        return cli.Env
    }
    return nil
}

// applyFilteredEnvVars replaces the step's env vars with the filtered set.
func applyFilteredEnvVars(step enginepkg.ResolvedStep, filtered map[string]string) {
    if cli, ok := step.Spec.(*schema.CLISpec); ok && cli != nil && filtered != nil {
        cli.Env = filtered
    }
}

// applyRedaction applies redaction patterns to all string values in the output map.
// Returns the count of patterns that matched at least once.
func applyRedaction(output map[string]any, patterns []*governance.RedactionPattern) int {
    // Walk output map recursively, apply regexp.ReplaceAllString on string values.
    // Count distinct patterns that matched.
    // Implementation uses pre-compiled regexps from RedactionPattern.
    matched := 0
    for _, p := range patterns {
        re := regexp.MustCompile(p.Pattern)
        if redactMap(output, re, p.Replacement) {
            matched++
        }
    }
    return matched
}

// GovernanceDeniedError is returned when a step is denied by governance policy.
type GovernanceDeniedError struct {
    StepID string
    Reason string
}

func (e *GovernanceDeniedError) Error() string {
    return "governance: step " + e.StepID + " denied: " + e.Reason
}
```

---

## Redaction Model

### Principle: pre-flight evaluation is SEPARATE from post-execution redaction

The governance evaluator performs a **pre-flight check** (allow/deny/approval). Redaction is a **post-execution pass** applied to executor output.

### When redaction happens

```
Step lifecycle:
  1. step/started event
  2. Governance pre-flight (Evaluate) ← allow/deny/approval
  3. Executor runs
  4. Executor returns StepResult with Output
  5. ★ REDACTION PASS ← applied here, to StepResult.Output
  6. Record result in run state
  7. Emit step/completed event (with redacted output)
  8. Merge vars (with redacted values)
```

### What gets redacted

| Output path | Redacted? | How |
|-------------|-----------|-----|
| `StepResult.Output` (map values) | ✅ Yes | Regex replace on all string values, recursively |
| `StepResult.Vars` (captured variables) | ✅ Yes | Same regex replace |
| Trace event payloads | ✅ Yes (indirectly) | Because we redact Output/Vars BEFORE emitting trace events |
| In-process event channel | ✅ Yes (indirectly) | Same — redaction happens before fan-out |
| `StepResult.Error` messages | ❌ No | Error messages may contain command names but not output; redaction of errors would obscure debugging |

### Redaction implementation

```go
// internal/governance/redaction.go

// Redactor applies a set of compiled redaction patterns to string values.
type Redactor struct {
    patterns []compiledPattern
}

type compiledPattern struct {
    re          *regexp.Regexp
    replacement string
}

// NewRedactor compiles all RedactionPatterns. Returns error if any pattern is invalid RE2.
func NewRedactor(patterns []*governance.RedactionPattern) (*Redactor, error)

// RedactString applies all patterns to s, returning the scrubbed string and match count.
func (r *Redactor) RedactString(s string) (string, int)

// RedactMap recursively walks a map and redacts all string values in-place.
// Returns total match count across all patterns and all values.
func (r *Redactor) RedactMap(m map[string]any) int
```

### Trace events with redacted fields

The `governance/redaction_applied` trace event contains:
- `step_id`: which step's output was scrubbed
- `rule_count`: how many distinct patterns matched (not the patterns themselves — patterns could leak structure)

The event does NOT contain:
- The original (pre-redaction) values (never written anywhere)
- The pattern strings (could reveal what secrets look like)

---

## Test Plan

### Evaluator tests (`internal/governance/evaluator_test.go`)

| # | Test Name | What It Verifies |
|---|-----------|-----------------|
| 1 | `TestEvaluate_AllowedCommand` | Command in allowlist → `Allowed=true`, `Denied=false` |
| 2 | `TestEvaluate_DeniedCommand` | Command in denylist → `Denied=true`, `DenyReason` populated |
| 3 | `TestEvaluate_DenyOverridesAllow` | Command in both allowlist AND denylist → `Denied=true` (deny-wins invariant) |
| 4 | `TestEvaluate_EmptyAllowlist_AllAllowed` | Empty allowlist → all commands allowed (permissive mode) |
| 5 | `TestEvaluate_NonEmptyAllowlist_UnlistedDenied` | Non-empty allowlist, command not in list → `Denied=true` |
| 6 | `TestEvaluate_GlobPatternMatching` | Allowlist pattern `kube*` matches `kubectl` |
| 7 | `TestEvaluate_EnvVarBlocking` | Env var matching `*_SECRET` removed from `FilteredEnvVars`, name appears in `BlockedEnvVars` |
| 8 | `TestEvaluate_EnvVarBlockingMultiplePatterns` | Multiple deny patterns, multiple env vars — correct union filtering |
| 9 | `TestEvaluate_RequiresApproval` | `RequireApproval=true` → `RequiresApproval=true`, `Denied=false` |
| 10 | `TestEvaluate_DeniedOverridesApproval` | Denied command with `RequireApproval=true` → still `Denied=true` (deny takes precedence over approval) |
| 11 | `TestEvaluate_EvidencePopulated` | Evidence struct has correct `StepID`, `EvaluatedAt`, `RulesMatched`, `Outcome` |
| 12 | `TestEvaluate_NonCLIStep_AlwaysAllowed` | Non-CLI step (empty Command) → `Allowed=true` (command checks only apply to commands) |
| 13 | `TestEvaluate_NilPolicy_AlwaysAllowed` | Nil/empty policy → `Allowed=true`, empty evidence |
| 14 | `TestEvaluate_EvidenceJSON_Roundtrip` | `json.Marshal` → `json.Unmarshal` on Evidence produces identical struct |

### Builder tests (`internal/governance/builder_test.go`)

| # | Test Name | What It Verifies |
|---|-----------|-----------------|
| 15 | `TestBuildPolicy_AllowCommands` | `GovernanceConfig.AllowCommands` → policy checks allowlist correctly |
| 16 | `TestBuildPolicy_DenyCommands` | `GovernanceConfig.DenyCommands` → policy checks denylist correctly |
| 17 | `TestBuildPolicy_MergeRunbookAndStep` | Runbook allows `[kubectl, curl]`, step allows `[kubectl]` → merged policy allows only `kubectl` |
| 18 | `TestBuildPolicy_MergeDenyIsAdditive` | Runbook denies `[rm]`, step denies `[dd]` → merged policy denies both |
| 19 | `TestBuildPolicy_MergeApprovalIsOR` | Runbook `RequireApproval=false`, step `RequireApproval=true` → merged requires approval |
| 20 | `TestBuildPolicy_RedactionPatterns` | `GovernanceConfig.Redact` → policy returns compiled patterns |

### Redaction tests (`internal/governance/redaction_test.go`)

| # | Test Name | What It Verifies |
|---|-----------|-----------------|
| 21 | `TestRedactor_SimplePattern` | Pattern `token=[A-Za-z0-9]+` redacts `token=abc123` to `token=[REDACTED]` |
| 22 | `TestRedactor_MultiplePatterns` | Multiple patterns all applied to same string |
| 23 | `TestRedactor_RecursiveMapRedaction` | Nested `map[string]any` with string values at multiple depths — all redacted |
| 24 | `TestRedactor_NoMatch_NoChange` | Input with no sensitive data → output unchanged, match count = 0 |

### Approval tests (`internal/governance/approval_test.go`)

| # | Test Name | What It Verifies |
|---|-----------|-----------------|
| 25 | `TestNoOpApprovalGate_AlwaysApproves` | Returns `ApprovalRecord` with non-empty Token, no error |
| 26 | `TestNoOpApprovalGate_ContextCancelled` | Cancelled context → returns context error |

### Fake tests (`pkg/testutil/`)

| # | Test Name | What It Verifies |
|---|-----------|-----------------|
| 27 | `TestFakeGovernancePolicy_CompileTimeCheck` | `var _ governance.GovernancePolicy = (*FakeGovernancePolicy)(nil)` compiles |
| 28 | `TestFakeApprovalGate_CompileTimeCheck` | `var _ governance.ApprovalGate = (*FakeApprovalGate)(nil)` compiles |

---

## Fake Implementations

### FakeGovernancePolicy

```go
// pkg/testutil/fake_governance_policy.go

type FakeGovernancePolicy struct {
    // AllowAll makes CheckCommand always return (true, "").
    AllowAll bool

    // DenyCommands lists commands that CheckCommand will deny.
    DenyCommands []string

    // DenyEnvPatterns lists env var name patterns to block.
    DenyEnvPatterns []string

    // Redactions are the redaction patterns to return.
    Redactions []*governance.RedactionPattern

    // CheckCommandCalls records all CheckCommand invocations.
    CheckCommandCalls []string
}

var _ governance.GovernancePolicy = (*FakeGovernancePolicy)(nil)

func (f *FakeGovernancePolicy) CheckCommand(command string) (bool, string) {
    f.CheckCommandCalls = append(f.CheckCommandCalls, command)
    for _, deny := range f.DenyCommands {
        if deny == command {
            return false, "deny:" + command
        }
    }
    return true, ""
}

func (f *FakeGovernancePolicy) FilterEnvVars(vars map[string]string) (map[string]string, []string) {
    // Simple implementation: block exact matches from DenyEnvPatterns
    filtered := make(map[string]string)
    var blocked []string
    for k, v := range vars {
        isBlocked := false
        for _, p := range f.DenyEnvPatterns {
            if matched, _ := filepath.Match(p, k); matched {
                isBlocked = true
                break
            }
        }
        if isBlocked {
            blocked = append(blocked, k)
        } else {
            filtered[k] = v
        }
    }
    return filtered, blocked
}

func (f *FakeGovernancePolicy) RedactionPatterns() []*governance.RedactionPattern {
    return f.Redactions
}
```

### FakeApprovalGate

```go
// pkg/testutil/fake_approval_gate.go

type FakeApprovalGate struct {
    // Approve controls whether RequestApproval returns success or error.
    Approve bool

    // Approver is the identity to include in the ApprovalRecord.
    Approver string

    // Error is returned when Approve is false.
    Error error

    // Calls records all RequestApproval invocations.
    Calls []ApprovalCall
    mu    sync.Mutex
}

type ApprovalCall struct {
    StepID string
    Reason string
    At     time.Time
}

var _ governance.ApprovalGate = (*FakeApprovalGate)(nil)

func NewFakeApprovalGate(approve bool) *FakeApprovalGate {
    return &FakeApprovalGate{
        Approve:  approve,
        Approver: "test-approver",
        Error:    errors.New("approval rejected"),
    }
}

func (f *FakeApprovalGate) RequestApproval(ctx context.Context, stepID string, reason string) (governance.ApprovalRecord, error) {
    f.mu.Lock()
    f.Calls = append(f.Calls, ApprovalCall{StepID: stepID, Reason: reason, At: time.Now()})
    f.mu.Unlock()

    if err := ctx.Err(); err != nil {
        return governance.ApprovalRecord{}, err
    }
    if !f.Approve {
        return governance.ApprovalRecord{}, f.Error
    }
    return governance.ApprovalRecord{
        Approver:   f.Approver,
        ApprovedAt: time.Now().UTC().Format(time.RFC3339Nano),
        Token:      uuid.New().String(),
    }, nil
}
```

---

## Open Questions for Brian

1. **Step-level governance field:** The current `schema.Step` struct does NOT have a `Governance *GovernanceConfig` field. The builder's merge logic is designed for it, but Brian needs to decide: add the field to `Step` now (schema change) or defer step-level governance to Phase 5? **Recommendation:** Defer. Build the merge infrastructure, but for Phase 4 only runbook-level governance is active. The builder accepts `...*schema.GovernanceConfig` variadic so step-level is trivially addable later.

2. **Glob matching library:** `filepath.Match` is limited (no `**`, no `{a,b}`). Should Brian use `filepath.Match` for v2.0 and upgrade to `github.com/gobwas/glob` later, or adopt a richer library now? **Recommendation:** Use `filepath.Match` for Phase 4. The spec says "exact match only" for v2.0 (§11: "Glob patterns are not supported in v2.0 (exact match only)"). However, the `GovernancePolicy` interface uses "glob patterns" in its doc comments, so Brian should use `path.Match` (which handles basic globs) and document the supported pattern syntax.

3. **Redaction pattern pre-compilation:** Should `BuildPolicy` pre-compile all `RedactRule` patterns into `*regexp.Regexp` at construction time, or compile lazily per step? **Recommendation:** Pre-compile at construction time (in `BuildPolicy`). This fails fast on invalid patterns and avoids per-step allocation.

4. **Thread-safety of evaluator:** The evaluator is called from `executeStep` while `h.mu` is held. Is additional internal synchronisation needed? **Answer:** No. The evaluator is stateless (reads policy, produces result). The policy is immutable after construction. No mutex needed inside the evaluator.

5. **`StepStatusDenied` vs reusing `StepStatusFailed`:** The task says "return a StepResult with Status: denied". This requires adding a new `StepStatusDenied` constant. Brian should verify that all existing engine consumers (adapters, trace writers, event handlers) handle unknown status values gracefully. **Recommendation:** Add `StepStatusDenied` and `StepOutcomeDenied`. Update the `emitEventLocked` switch to handle the denied case.

6. **TerminalApprovalGate stdin/stdout interaction:** The `TerminalApprovalGate` needs to prompt on stdout and read from stdin. The engine currently doesn't carry stdin/stdout handles. Brian should pass `io.Reader`/`io.Writer` to the gate constructor. **Recommendation:** `NewTerminalApprovalGate(in io.Reader, out io.Writer)`.

7. **Evidence attachment to trace events:** Should evidence be a top-level field in the trace payload or nested under a `"governance"` key? **Recommendation:** Nest under `"governance"` key in the trace payload to avoid polluting the top-level namespace.

---

## Correctness Gate (Self-Review)

- [x] Every interface method is in the `pkg/` layer: `PolicyEvaluator`, `ApprovalGate` in `pkg/governance/`; implementations in `internal/governance/`
- [x] No circular imports: `internal/governance` → `pkg/governance`, `pkg/schema`, `pkg/trace` (NOT `internal/engine`); `StepInfo` type avoids the cycle
- [x] `EvaluationResult` carries `Allowed`, `Denied`, `RequiresApproval`, `FilteredEnvVars`, `Evidence` — engine acts on these fields without knowing policy internals
- [x] `Evidence` is a value type (no interfaces, no channels, no mutexes), all fields have JSON tags, testable with `json.Marshal/Unmarshal`
- [x] `ApprovalGate` is an interface injected via `EngineConfig` — engine holds the interface, not a concrete type
- [x] Deny-wins semantics are explicit: `PolicyEvaluator` doc contract says "Deny ALWAYS takes precedence over allow"; evaluator evaluates deny first and short-circuits
- [x] Redaction is a post-execution pass in `executeStep`, not mixed into the evaluator

---

## Summary

Phase 4 adds four new files to `pkg/governance/` (public contracts) and four new files to `internal/governance/` (implementations). The `StepInfo` type breaks the potential import cycle between `pkg/governance` and `pkg/engine`. The evaluator is injected into the engine via `EngineConfig.GovernanceEvaluator`. Deny-wins is enforced by evaluation order (deny first, short-circuit). Redaction is a separate post-execution pass. Evidence is a JSON-serialisable value type attached to trace events. 28 tests cover allow/deny/merge/redaction/approval/fakes.
