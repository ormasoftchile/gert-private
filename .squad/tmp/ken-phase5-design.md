# Phase 5 Design: Step Type Executors

**Author:** Ken (Software Architect)  
**Date:** 2026-04-21  
**Status:** Ready for implementation  
**Implementor:** Brian  
**Depends on:** Phase 3 (engine), Phase 4 (governance)

---

## Package Structure

### New public contracts (`pkg/`)

| File | Purpose |
|------|---------|
| `pkg/expr/expr.go` | `Evaluator` interface and `TemplateEvaluator` implementation — resolves `{{var}}` expressions in step fields |
| `pkg/expr/condition.go` | `ConditionEvaluator` interface — evaluates boolean condition strings for `when`, `BranchArm.Condition` |
| `pkg/expr/doc.go` | Package documentation |
| `pkg/input/input.go` | `InputProvider` interface — collects user input for choice/decision/collector steps |
| `pkg/input/doc.go` | Package documentation |

### New private implementations (`internal/executor/`)

| File | Purpose |
|------|---------|
| `internal/executor/registry.go` | Concrete `ExecutorRegistry` implementation + `NewDefaultRegistry()` factory |
| `internal/executor/cli.go` | `CLIExecutor` — shell subprocess execution via Platform |
| `internal/executor/tool.go` | `ToolExecutor` — delegates to a `ToolRegistry` (Phase 6 completes transport) |
| `internal/executor/include.go` | `IncludeExecutor` — no-op pass-through (planner already flattened) |
| `internal/executor/choice.go` | `ChoiceExecutor` — prompts user via InputProvider, stores selection in vars |
| `internal/executor/decision.go` | `DecisionExecutor` — prompts user via InputProvider, stores route label+value |
| `internal/executor/collector.go` | `CollectorExecutor` — multi-field form via InputProvider, stores all fields |
| `internal/executor/branch.go` | `BranchExecutor` — evaluates conditions, executes matching arm inline |
| `internal/executor/iterate.go` | `IterateExecutor` — loops over collection or count, manages loop vars |
| `internal/executor/approve.go` | `ApproveExecutor` — delegates to `governance.ApprovalGate` from EngineConfig |
| `internal/executor/assert.go` | `AssertExecutor` — evaluates assertions, returns denied/failed on failure |
| `internal/executor/compensate.go` | `CompensateExecutor` — registers compensation steps, invokes on failure |
| `internal/executor/end.go` | `EndExecutor` — sets terminal outcome marker in StepResult |
| `internal/executor/doc.go` | Package documentation |

### New expression/condition implementations (`internal/expr/`)

| File | Purpose |
|------|---------|
| `internal/expr/template.go` | `TemplateEvaluator` — implements `pkg/expr.Evaluator` using `text/template` |
| `internal/expr/condition.go` | `SimpleConditionEvaluator` — implements `pkg/expr.ConditionEvaluator` using `text/template` boolean evaluation |
| `internal/expr/template_test.go` | Unit tests for template evaluation |
| `internal/expr/condition_test.go` | Unit tests for condition evaluation |

### New input implementations (`internal/input/`)

| File | Purpose |
|------|---------|
| `internal/input/terminal.go` | `TerminalInputProvider` — reads from stdin/stdout for `gert run` |

### New test infrastructure (`pkg/testutil/`)

| File | Purpose |
|------|---------|
| `pkg/testutil/fake_input_provider.go` | `FakeInputProvider` — pre-programmed responses for tests |
| `pkg/testutil/fake_expr_evaluator.go` | `FakeExprEvaluator` — passthrough or fixed-result evaluator for tests |

### Sentinel types (no executor — engine handles natively)

| Kind | Strategy | Rationale |
|------|----------|-----------|
| `parallel` | Not registered | Engine's `executeStep` dispatches to `executeParallel` before registry lookup |
| `wait_for_event` | Not registered | Engine's `executeStep` dispatches to `executeWaitForEvent` before registry lookup |

### Import dependency graph (no cycles)

```
pkg/expr              ← no gert imports (leaf package)
pkg/input             ← no gert imports (leaf package)
pkg/engine            ← pkg/governance, pkg/eventbus, pkg/trace, pkg/platform, pkg/schema

internal/expr         → pkg/expr
internal/input        → pkg/input, pkg/platform
internal/executor     → pkg/engine, pkg/schema, pkg/expr, pkg/input, pkg/governance, pkg/platform
                      ✗ MUST NOT import internal/engine
                      ✗ MUST NOT import internal/expr (uses pkg/expr interface)
                      ✗ MUST NOT import internal/input (uses pkg/input interface)

internal/engine       → pkg/engine, internal/executor (only for NewDefaultRegistry wiring)
```

---

## Architectural Decisions

### D1: Package layout — Flat `internal/executor/{kind}.go`

**Decision:** One file per executor type, flat under `internal/executor/`.

**Rationale:**
- All 14 executors share the same interface (`StepExecutor`) and similar dependencies. Sub-packages would add import ceremony with no isolation benefit.
- The executor package is already `internal/` — further nesting is over-engineering.
- Flat layout mirrors `pkg/schema/steps.go` (all spec types in one package) and keeps the mental model simple.
- `internal/executor/registry.go` hosts the concrete `ExecutorRegistry`.
- Total: ~16 files including `doc.go` and `registry.go`. Manageable.

### D2: CLI executor — Platform-based subprocess model

**Decision:** CLI executor uses a new `Platform.Exec(ctx, ExecRequest) (*ExecResult, error)` method.

**Rationale:**
- `Platform` already abstracts `DefaultShell()` and `ExecSuffix()`. Adding subprocess execution is the natural extension.
- Injecting Platform makes CLI tests hermetic — `FakePlatform` can return canned stdout/stderr without forking real processes.
- The `ExecRequest` value type carries command, args, env, workdir, stdin. `ExecResult` returns exit code, stdout, stderr.
- For Phase 5, stdout and stderr are captured fully into `StepResult.Output["stdout"]` and `StepResult.Output["stderr"]`. Streaming is a Phase 6+ concern (adapter-driven).

**New types on Platform:**

```go
// ExecRequest describes a subprocess to run.
type ExecRequest struct {
    Command string
    Args    []string
    Env     map[string]string
    Workdir string
    Stdin   string
    Shell   string // override DefaultShell(); empty = use default
}

// ExecResult holds captured subprocess output.
type ExecResult struct {
    ExitCode int
    Stdout   string
    Stderr   string
}

// Platform interface gains:
Exec(ctx context.Context, req ExecRequest) (*ExecResult, error)
```

### D3: Variable expression evaluation — `text/template` via `pkg/expr.Evaluator`

**Decision:** Use Go's `text/template` with a thin wrapper exposed as `pkg/expr.Evaluator`.

**Rationale:**
- `text/template` supports `{{.var}}`, `{{ eq .a "b" }}`, `{{ if }}`, pipes — sufficient for all v2 step field interpolation.
- The existing fixtures already use Go template syntax (`{{ .svc }}`, `{{ eq .chosen_env "Production" }}`), confirming this is the right fit.
- Wrapping it in `pkg/expr.Evaluator` interface allows future replacement with CEL/OPA without touching executor code.
- The evaluator is a stateless value type: `Eval(template string, vars map[string]any) (string, error)`.
- No custom functions in Phase 5. Built-in `text/template` functions (`eq`, `ne`, `lt`, `gt`, `and`, `or`, `not`, `len`, `index`) cover all current fixture needs.

### D4: Condition evaluation — Same `text/template` engine, boolean output

**Decision:** `ConditionEvaluator` uses the same `text/template` engine, wrapping conditions in `{{ if EXPR }}true{{ else }}false{{ end }}` and parsing the result.

**Rationale:**
- Conditions in `BranchArm.Condition` and `Step.When` already use template syntax (`{{ eq .chosen_env "Production" }}`).
- Using the same engine means one syntax to learn, one implementation to test.
- `ConditionEvaluator.Eval(condition string, vars map[string]any) (bool, error)` — returns typed bool.
- CEL is deferred to v2.1 (per Dennis's research brief and existing governance deferral decision).

### D5: InputProvider — New `pkg/input.InputProvider` interface

**Decision:** Define `InputProvider` in `pkg/input/` as a new leaf package.

**Rationale:**
- Choice, decision, and collector steps all require user interaction. The engine can't call `fmt.Scan` directly (non-interactive modes, VS Code extension, JSON-RPC server).
- `InputProvider` is injected into executors, not into the engine directly. This keeps the engine protocol-agnostic.
- Platform already handles OS abstractions (signals, temp dir, exec). User input is a different concern — it's UI/protocol specific, not OS specific. Mixing them violates SRP.
- `FakeInputProvider` enables deterministic testing of all interactive steps.

**Interface:**

```go
package input

import "context"

// InputProvider collects user input for interactive steps.
// Implementations: TerminalInputProvider (stdin), FakeInputProvider (tests),
// RPCInputProvider (JSON-RPC server, Phase 6+).
type InputProvider interface {
    // PromptChoice presents options and returns the selected value(s).
    // single-select returns one value; multi-select returns CSV or []string depending on mode.
    PromptChoice(ctx context.Context, req ChoiceRequest) (*ChoiceResponse, error)

    // PromptDecision presents routes and returns the selected route label.
    PromptDecision(ctx context.Context, req DecisionRequest) (*DecisionResponse, error)

    // PromptForm presents a multi-field form and returns all field values.
    PromptForm(ctx context.Context, req FormRequest) (*FormResponse, error)
}

// ChoiceRequest describes a choice prompt.
type ChoiceRequest struct {
    StepID   string
    Prompt   string
    Options  []Option
    Default  string
    Multiple bool
    Min, Max int
}

// Option is a single selectable option.
type Option struct {
    Label string
    Value string
    Hint  string
}

// ChoiceResponse holds the user's selection.
type ChoiceResponse struct {
    Selected []string // value(s) chosen
}

// DecisionRequest describes a decision prompt.
type DecisionRequest struct {
    StepID string
    Prompt string
    Routes []Route
}

// Route is a single route option.
type Route struct {
    Label string
    Hint  string
}

// DecisionResponse holds the user's route selection.
type DecisionResponse struct {
    Label string // label of the chosen route
}

// FormRequest describes a multi-field form.
type FormRequest struct {
    StepID string
    Prompt string
    Fields []FormField
}

// FormField describes one field in a form.
type FormField struct {
    Name     string
    Type     string // "text", "number", "boolean", "select", etc.
    Label    string
    Required bool
    Default  any
    Hint     string
    Options  []Option // for select/autocomplete types
}

// FormResponse holds all field values.
type FormResponse struct {
    Values map[string]any
}
```

### D6: include executor — Registered no-op pass-through

**Decision:** Register an `IncludeExecutor` that returns `StepStatusCompleted` with no vars and no output.

**Rationale:**
- The planner already flattens included runbooks into the plan's step list. By the time the engine sees an `include` step, its substeps are already in-line.
- However, the planner *may* emit the include step itself as a marker in the plan (e.g., for trace clarity: "step/started include X").
- Registering a no-op executor is safer than leaving it unregistered (which would cause `ExecutorNotFoundError`). The include step becomes a benign pass-through that emits trace events normally.
- If the planner fully elides include steps (no marker), this executor is never called. Safe either way.

### D7: parallel and wait_for_event — Not registered (engine-native)

**Decision:** Do NOT register executors for `parallel` or `wait_for_event`. The engine handles them natively before reaching the registry.

**Rationale:**
- `executeStep()` in `internal/engine/engine.go` already has a `switch step.Kind` that dispatches `parallel` and `wait_for_event` before the executor lookup (lines 161–170).
- If the engine dispatches correctly, the registry is never consulted. If the engine fails to dispatch (spec doesn't implement the provider interface), the step falls through to the executor lookup and gets `ExecutorNotFoundError` — which is the correct failure mode (it means the planner produced a malformed step).
- A sentinel executor that panics would hide planner bugs. An explicit "not registered → error" is more diagnostic.
- Document this in `NewDefaultRegistry()` as a comment so Brian doesn't accidentally register them.

### D8: ExecutorRegistry concrete implementation — `internal/executor/registry.go`

**Decision:** Concrete `MapRegistry` lives in `internal/executor/registry.go`.

**Rationale:**
- The registry is a simple `map[string]StepExecutor` with `sync.RWMutex`. It belongs alongside the executors it maps to.
- `internal/executor/` already hosts all executor implementations; the registry that wires them is a natural companion.
- The engine imports `internal/executor.NewDefaultRegistry()` during construction.
- NOT `internal/engine/registry.go` — that would couple the registry to the engine's package, making it harder to test executors in isolation.

---

## Interface Contracts

### `pkg/expr.Evaluator`

```go
package expr

// Evaluator resolves template expressions in step field values.
// Implementations MUST be safe for concurrent use.
type Evaluator interface {
    // Eval resolves all template expressions in tmpl using vars.
    // Returns the resolved string, or an error if the template is malformed
    // or references an undefined variable.
    Eval(tmpl string, vars map[string]any) (string, error)
}
```

### `pkg/expr.ConditionEvaluator`

```go
package expr

// ConditionEvaluator evaluates boolean condition expressions.
// Used for Step.When guards and BranchArm.Condition.
type ConditionEvaluator interface {
    // EvalBool evaluates a condition expression and returns true/false.
    // Returns an error if the expression is malformed.
    EvalBool(condition string, vars map[string]any) (bool, error)
}
```

### `pkg/input.InputProvider`

(See D5 above for full interface definition.)

### New Platform method: `Exec`

```go
// Added to pkg/platform.Platform interface:
Exec(ctx context.Context, req ExecRequest) (*ExecResult, error)
```

---

## Executor Specifications

### 1. `cli` — CLIExecutor

**Spec type:** `schema.CLISpec`  
**Dependencies:** `platform.Platform` (for `Exec`), `expr.Evaluator` (for template resolution)

**Execute() contract:**
1. Resolve `{{var}}` templates in `Command`, `Args`, `Env` values, `Workdir`, and `Stdin` using `Evaluator`.
2. Determine shell: use `CLISpec.Shell` if set, else `Platform.DefaultShell()`.
3. Build `ExecRequest` from resolved fields.
4. Call `Platform.Exec(ctx, req)`.
5. Populate `StepResult`:
   - `Output["stdout"]` = captured stdout
   - `Output["stderr"]` = captured stderr
   - `Output["exit_code"]` = exit code (int)
   - `Status` = `StepStatusCompleted` if exit code == 0, else `StepStatusFailed`
   - `Vars` = populated from `Step.Capture` map (e.g., `capture: { msg: stdout }` → `Vars["msg"] = stdout`)
6. Return `StepResult`, nil. Infrastructure errors (exec failure, ctx timeout) return nil, error.

**Run field handling:** If `CLISpec.Run` is set (string or map), it represents an inline script. If string: pass as single shell argument. If map: each key is a named script (future; Phase 5 uses first/only value).

### 2. `tool` — ToolExecutor

**Spec type:** `schema.ToolCallSpec`  
**Dependencies:** `engine.ToolRegistry` (if defined), `expr.Evaluator`

**Execute() contract:**
1. Resolve `{{var}}` templates in `ToolInvocation.Args` values.
2. Look up tool definition from `ExecutionPlan.Tools[name]` (available via step context — see note below).
3. In Phase 5: return `StepStatusCompleted` with `Output["tool_name"]`, `Output["action"]`, `Output["args"]`. Full tool invocation transport is Phase 6.
4. If tool not found: return `StepResult` with `StepStatusFailed`, error message.

**Note:** The executor needs access to the plan's tool definitions. This is passed via a new `ToolLookup` function injected at construction time: `type ToolLookup func(name string) *schema.ToolDef`.

### 3. `include` — IncludeExecutor

**Spec type:** `schema.IncludeSpec`  
**Dependencies:** None

**Execute() contract:**
1. Return `StepResult` with `Status = StepStatusCompleted`, `Outcome = StepOutcomeSuccess`.
2. No vars, no output. The planner already inlined the included runbook's steps.

### 4. `choice` — ChoiceExecutor

**Spec type:** `schema.ChoiceSpec`  
**Dependencies:** `input.InputProvider`, `expr.Evaluator`

**Execute() contract:**
1. Resolve `{{var}}` templates in `Prompt` and option labels/hints.
2. Build `ChoiceRequest` from `ChoiceSpec`.
3. Call `InputProvider.PromptChoice(ctx, req)`.
4. Store result in `Vars[ChoiceSpec.Variable]` = selected value (string for single, `[]string` for multiple).
5. Set `Status = StepStatusCompleted`.
6. If InputProvider returns error (user cancelled, timeout): `Status = StepStatusFailed`.

### 5. `decision` — DecisionExecutor

**Spec type:** `schema.DecisionSpec`  
**Dependencies:** `input.InputProvider`, `expr.Evaluator`

**Execute() contract:**
1. Resolve `{{var}}` templates in `Prompt` and route labels/hints.
2. Build `DecisionRequest` from `DecisionSpec`.
3. Call `InputProvider.PromptDecision(ctx, req)`.
4. Store result:
   - `Vars[DecisionSpec.Variable]` = selected route's label (if Variable is set)
   - `Output["route_label"]` = selected label
   - `Output["route_goto"]` = goto target (if any)
   - `Output["route_runbook"]` = runbook reference (if any)
5. Set `Status = StepStatusCompleted`.
6. Engine reads `Output["route_goto"]` for jump dispatch (engine-level concern, not executor).

### 6. `collector` — CollectorExecutor

**Spec type:** `schema.CollectorSpec`  
**Dependencies:** `input.InputProvider`, `expr.Evaluator`, `expr.ConditionEvaluator`

**Execute() contract:**
1. Resolve `{{var}}` templates in `Prompt` and field labels/hints/defaults.
2. Evaluate each field's `When` condition to determine which fields to present. Skip fields where `When` evaluates to false.
3. Build `FormRequest` with active fields only.
4. Call `InputProvider.PromptForm(ctx, req)`.
5. Validate required fields are present (non-empty). Validate field-level constraints (min_length, max_length, pattern) if set.
6. Store all field values in `Vars`: `Vars[field.Name]` = value for each field.
7. Mark ephemeral fields in `Output["ephemeral_fields"]` = list of field names (for trace redaction awareness).
8. Set `Status = StepStatusCompleted`.

### 7. `branch` — BranchExecutor

**Spec type:** `schema.BranchSpec`  
**Dependencies:** `expr.ConditionEvaluator`, sub-executor dispatch

**Execute() contract:**
1. Iterate `BranchSpec.Branches` in order.
2. For each arm: if `Else == true`, this is the default arm (last resort). Otherwise, evaluate `Condition` via `ConditionEvaluator.EvalBool(condition, vars)`.
3. First arm whose condition is true (or the `else` arm if no condition matches): execute its `Steps` sequentially using a provided step-execution callback.
4. If no arm matches and no else arm exists: return `StepStatusSkipped`.
5. Return the aggregate `StepResult`:
   - `Status` = last sub-step status, or `StepStatusCompleted` if all sub-steps succeeded
   - `Vars` = merged vars from all sub-steps
   - `Output["matched_arm"]` = label of the matched arm

**Sub-step execution:** Branch needs to execute nested steps. This is a callback injected at construction: `type SubStepRunner func(ctx context.Context, steps []schema.FlowNode, vars map[string]any) ([]*engine.StepResult, error)`. The engine provides this callback, bridging executor and engine without a direct import.

### 8. `iterate` — IterateExecutor

**Spec type:** `schema.IterateNode`  
**Dependencies:** `expr.Evaluator`, `expr.ConditionEvaluator`, sub-executor dispatch

**Execute() contract:**
1. Resolve `Over` expression to get the collection. If `Over` is a JSONPath expression (`$.services`), look up the value in vars. If it's a comma-separated string or number, generate the range.
2. For each item in the collection:
   a. Set loop variable `Vars[As]` = current item.
   b. Execute `Steps` sequentially using the SubStepRunner callback.
   c. Evaluate `Until` condition (if set) — break if true.
   d. Check `Max` — break if iteration count exceeds max.
   e. Evaluate `Collect` expressions and accumulate values.
3. After loop completes:
   - `Vars` = collected variables from `Collect` map (last values win)
   - `Status` = `StepStatusCompleted` if all iterations succeeded
   - `Output["iterations"]` = number of iterations completed

### 9. `parallel` — Engine-native (no executor)

**Not registered in ExecutorRegistry.** Engine dispatches via `executeParallel()` directly when `step.Spec` implements `parallelBranchProvider`. See D7.

### 10. `approve` — ApproveExecutor

**Spec type:** `schema.ApproveSpec`  
**Dependencies:** `governance.ApprovalGate` (from EngineConfig)

**Execute() contract:**
1. Build approval request from `ApproveSpec.Approvals` (pool, required count, timeout).
2. Call `ApprovalGate.RequestApproval(ctx, stepID, reason)`.
3. If approved: `Status = StepStatusCompleted`, `Output["approver"]`, `Output["token"]`.
4. If rejected: `Status = StepStatusDenied`, `Output["reason"]`.
5. If timeout: `Status = StepStatusFailed` (or `StepStatusDenied` per `OnTimeout` config).
6. Return nil error. Only infrastructure failures (gate unreachable) return error.

**Note:** The governance pre-flight in `executeStep` already calls the ApprovalGate for governance-triggered approvals. The `approve` executor handles *explicit* approval steps authored by the runbook writer. Both paths use the same `ApprovalGate` interface but with different triggers.

### 11. `assert` — AssertExecutor

**Spec type:** `schema.AssertSpec`  
**Dependencies:** `expr.Evaluator`

**Execute() contract:**
1. For each `Assertion` in `AssertSpec.Assert`:
   a. Resolve `Subject` and `Expected` templates.
   b. Evaluate based on `Type`:
      - `"eq"`: subject == expected
      - `"ne"`: subject != expected
      - `"contains"`: subject contains expected
      - `"matches"`: subject matches expected (regexp)
      - `"exists"`: subject is non-empty
   c. If assertion fails: record failure details.
2. If all assertions pass: `Status = StepStatusCompleted`.
3. If any assertion fails: `Status = StepStatusFailed`, `Output["failures"]` = list of failure details.
4. Return nil error (assertion failure is a step-level failure, not infrastructure).

### 12. `compensate` — CompensateExecutor

**Spec type:** `schema.CompensateSpec`  
**Dependencies:** SubStepRunner callback

**Execute() contract:**
1. Register the compensation steps (from `CompensateConfig.Steps`) in a compensation stack accessible via `StepResult`.
2. Return `StepResult` with `Status = StepStatusCompleted` and `Output["compensation_registered"] = true`.
3. The compensation steps are stored in `Vars["__compensation_<stepID>"]` as a serialized list of step specs.
4. The engine checks for compensation registrations when a subsequent step fails and `CompensateConfig.On` matches the failure trigger.

**Phase 5 scope:** Register compensation metadata. Actual compensation execution (running the registered steps on failure) requires engine-level support that bridges Phase 5 and the engine's failure handling. The executor stores the metadata; the engine reads it.

**Compensation trigger flow (engine responsibility):**
- On step failure, engine scans `Run.Vars` for `__compensation_*` keys.
- For each registered compensation whose `On` condition matches, engine executes the compensation steps in LIFO order (stack semantics).
- This engine logic is part of Phase 5 but lives in `internal/engine/`, not `internal/executor/`.

### 13. `wait_for_event` — Engine-native (no executor)

**Not registered in ExecutorRegistry.** Engine dispatches via `executeWaitForEvent()` directly when `step.Spec` implements `waitEventProvider`. See D7.

### 14. `end` — EndExecutor

**Spec type:** `schema.EndSpec`  
**Dependencies:** None

**Execute() contract:**
1. Read `EndSpec.Outcome` (category + code).
2. Set `StepResult`:
   - `Status = StepStatusCompleted`
   - `Output["terminal"] = true` — marker for the engine to observe
   - `Output["outcome_category"]` = category (e.g., "success", "failure", "rollback")
   - `Output["outcome_code"]` = code (e.g., "DEPLOY_COMPLETE")
   - `Vars["__run_outcome_category"]` = category
   - `Vars["__run_outcome_code"]` = code
3. Return result.
4. The engine, after merging vars from `end` step, checks for `__run_outcome_*` keys and transitions the run to the terminal state indicated.

---

## Expression / Condition Evaluation

### Supported syntax (Phase 5)

| Feature | Syntax | Example |
|---------|--------|---------|
| Variable interpolation | `{{ .varname }}` | `{{ .namespace }}` |
| Nested access | `{{ .parent.child }}` | `{{ .config.region }}` |
| Equality comparison | `{{ eq .a "value" }}` | `{{ eq .chosen_env "Production" }}` |
| Inequality | `{{ ne .a "value" }}` | `{{ ne .status "failed" }}` |
| Numeric comparison | `{{ lt .count 10 }}` | `{{ gt .retries 3 }}` |
| Boolean logic | `{{ and EXPR EXPR }}` | `{{ and (eq .a "x") (ne .b "y") }}` |
| Negation | `{{ not EXPR }}` | `{{ not .dry_run }}` |
| JSONPath variable ref | `$.inputname` | `$.services` (resolved to var lookup) |

### Implementation detail

**`internal/expr/template.go`:**

```go
type TemplateEvaluator struct{}

func (e *TemplateEvaluator) Eval(tmpl string, vars map[string]any) (string, error) {
    // If tmpl contains no {{ delimiters, return as-is (fast path).
    // Parse with text/template, execute with vars as dot context.
    // Return rendered string.
    // Errors: malformed template, undefined variable (with option("missingkey", "error")).
}
```

**`internal/expr/condition.go`:**

```go
type SimpleConditionEvaluator struct {
    expr Evaluator
}

func (e *SimpleConditionEvaluator) EvalBool(condition string, vars map[string]any) (bool, error) {
    // Wrap condition: `{{ if CONDITION }}true{{ else }}false{{ end }}`
    // Evaluate via TemplateEvaluator.
    // Parse result as bool.
    // Edge case: empty condition = true (unconditional).
}
```

**JSONPath `$` prefix handling:**

Expressions starting with `$.` are converted to a Go template dot-access before evaluation:
- `$.services` → `.services` (look up `services` key in vars map)
- This is a syntactic sugar layer in the evaluator, not a full JSONPath implementation.
- Full JSONPath (nested arrays, filters) is deferred to v2.1.

---

## InputProvider

### Interface (in `pkg/input/input.go`)

See D5 above for the full interface and all request/response types.

### FakeInputProvider (`pkg/testutil/fake_input_provider.go`)

```go
type FakeInputProvider struct {
    // ChoiceResponses maps stepID → pre-programmed ChoiceResponse.
    ChoiceResponses map[string]*ChoiceResponse
    // DecisionResponses maps stepID → pre-programmed DecisionResponse.
    DecisionResponses map[string]*DecisionResponse
    // FormResponses maps stepID → pre-programmed FormResponse.
    FormResponses map[string]*FormResponse
    // DefaultError is returned when no response is registered (nil = use defaults).
    DefaultError error
    // Calls records all prompt invocations for assertions.
    Calls []InputCall
}

type InputCall struct {
    StepID string
    Kind   string // "choice", "decision", "form"
    At     time.Time
}
```

**Behaviour:**
- If a response is registered for the step ID, return it.
- If no response is registered and DefaultError is nil, return a sensible default (first option for choice, first route for decision, empty form for collector).
- If DefaultError is set, return it for any unregistered step.

### TerminalInputProvider (`internal/input/terminal.go`)

```go
type TerminalInputProvider struct {
    In  io.Reader
    Out io.Writer
}
```

- PromptChoice: prints prompt + numbered options, reads line, validates selection.
- PromptDecision: prints prompt + numbered routes, reads line.
- PromptForm: prints each field label, reads value per field, validates.
- On context cancellation, returns error immediately.

---

## ExecutorRegistry Implementation

### `internal/executor/registry.go`

```go
package executor

import (
    "sync"
    "github.com/ormasoftchile/gert/v2/pkg/engine"
)

// MapRegistry is the concrete ExecutorRegistry backed by a sync.RWMutex map.
type MapRegistry struct {
    mu   sync.RWMutex
    execs map[string]engine.StepExecutor
}

var _ engine.ExecutorRegistry = (*MapRegistry)(nil)

func NewMapRegistry() *MapRegistry {
    return &MapRegistry{execs: make(map[string]engine.StepExecutor)}
}

func (r *MapRegistry) Register(kind string, exec engine.StepExecutor) {
    r.mu.Lock()
    r.execs[kind] = exec
    r.mu.Unlock()
}

func (r *MapRegistry) Lookup(kind string) engine.StepExecutor {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return r.execs[kind]
}
```

### `NewDefaultRegistry()` factory

```go
// NewDefaultRegistry constructs a MapRegistry pre-loaded with all Phase 5 executors.
// Requires all dependencies to be injected.
//
// NOTE: "parallel" and "wait_for_event" are intentionally NOT registered.
// The engine dispatches these natively in executeStep() before consulting
// the registry. If they reach the registry, it means the planner produced
// a malformed step spec (missing provider interface), and ExecutorNotFoundError
// is the correct diagnostic.
func NewDefaultRegistry(deps ExecutorDeps) *MapRegistry {
    r := NewMapRegistry()
    r.Register("cli",            NewCLIExecutor(deps.Platform, deps.Expr))
    r.Register("tool",           NewToolExecutor(deps.ToolLookup, deps.Expr))
    r.Register("include",        NewIncludeExecutor())
    r.Register("choice",         NewChoiceExecutor(deps.Input, deps.Expr))
    r.Register("decision",       NewDecisionExecutor(deps.Input, deps.Expr))
    r.Register("collector",      NewCollectorExecutor(deps.Input, deps.Expr, deps.Cond))
    r.Register("branch",         NewBranchExecutor(deps.Cond, deps.SubStepRunner))
    r.Register("iterate",        NewIterateExecutor(deps.Expr, deps.Cond, deps.SubStepRunner))
    r.Register("approve",        NewApproveExecutor(deps.ApprovalGate))
    r.Register("assert",         NewAssertExecutor(deps.Expr))
    r.Register("compensate",     NewCompensateExecutor())
    r.Register("end",            NewEndExecutor())
    // "parallel" — NOT registered (engine-native, see executeParallel)
    // "wait_for_event" — NOT registered (engine-native, see executeWaitForEvent)
    return r
}

// ExecutorDeps bundles all dependencies required to construct the default executor set.
type ExecutorDeps struct {
    Platform      platform.Platform
    Expr          expr.Evaluator
    Cond          expr.ConditionEvaluator
    Input         input.InputProvider
    ApprovalGate  governance.ApprovalGate
    ToolLookup    func(name string) *schema.ToolDef
    SubStepRunner func(ctx context.Context, steps []schema.FlowNode, vars map[string]any) ([]*engine.StepResult, error)
}
```

---

## Test Plan

### Unit tests per executor (`internal/executor/*_test.go`)

| # | Test Name | Executor | Asserts |
|---|-----------|----------|---------|
| 1 | `TestCLI_SimpleCommand` | cli | stdout captured in Output, exit 0 → completed |
| 2 | `TestCLI_NonZeroExit` | cli | exit 1 → StepStatusFailed, stderr captured |
| 3 | `TestCLI_TemplateResolution` | cli | `{{.name}}` in command resolved from vars |
| 4 | `TestCLI_EnvVarInjection` | cli | env map passed to ExecRequest |
| 5 | `TestCLI_ContextCancellation` | cli | cancelled ctx → error |
| 6 | `TestTool_LookupAndStub` | tool | tool name/action/args in Output |
| 7 | `TestTool_UnknownTool` | tool | StepStatusFailed on missing tool |
| 8 | `TestInclude_PassThrough` | include | StepStatusCompleted, no vars/output |
| 9 | `TestChoice_SingleSelect` | choice | selected value stored in Vars[variable] |
| 10 | `TestChoice_MultiSelect` | choice | multiple values stored as []string |
| 11 | `TestChoice_DefaultOnNoInput` | choice | default value used when provider returns default |
| 12 | `TestDecision_RouteSelection` | decision | label + goto stored in Output |
| 13 | `TestDecision_VarCapture` | decision | Vars[variable] = label |
| 14 | `TestCollector_AllFields` | collector | all field values in Vars |
| 15 | `TestCollector_ConditionalField` | collector | field with false When condition skipped |
| 16 | `TestCollector_Validation` | collector | required field missing → StepStatusFailed |
| 17 | `TestBranch_FirstMatch` | branch | first true condition arm executes |
| 18 | `TestBranch_ElseArm` | branch | else arm executes when no condition matches |
| 19 | `TestBranch_NoMatch` | branch | StepStatusSkipped when no arm matches |
| 20 | `TestIterate_OverList` | iterate | loops N times, loop var set each iteration |
| 21 | `TestIterate_MaxBreak` | iterate | stops at Max iterations |
| 22 | `TestIterate_UntilBreak` | iterate | stops when Until condition is true |
| 23 | `TestIterate_Collect` | iterate | Collect expressions accumulated in Vars |
| 24 | `TestApprove_Approved` | approve | StepStatusCompleted, approver in Output |
| 25 | `TestApprove_Rejected` | approve | StepStatusDenied |
| 26 | `TestAssert_AllPass` | assert | StepStatusCompleted |
| 27 | `TestAssert_EqFail` | assert | StepStatusFailed, failure details in Output |
| 28 | `TestAssert_ContainsMatch` | assert | "contains" type passes |
| 29 | `TestCompensate_Registration` | compensate | compensation metadata stored in Vars |
| 30 | `TestEnd_OutcomeMarker` | end | terminal=true and outcome in Output |
| 31 | `TestEnd_NoOutcome` | end | StepStatusCompleted, no outcome vars |

### Expression evaluator tests (`internal/expr/*_test.go`)

| # | Test Name | Asserts |
|---|-----------|---------|
| 32 | `TestEval_SimpleVar` | `{{ .name }}` → resolved value |
| 33 | `TestEval_NoDelimiters` | passthrough, no parse |
| 34 | `TestEval_MissingVar` | error on undefined with missingkey=error |
| 35 | `TestEval_NestedAccess` | `{{ .config.region }}` works |
| 36 | `TestCondition_EqTrue` | `{{ eq .env "prod" }}` → true |
| 37 | `TestCondition_EqFalse` | `{{ eq .env "staging" }}` → false |
| 38 | `TestCondition_Empty` | empty condition → true |
| 39 | `TestCondition_BooleanLogic` | `{{ and (eq .a "x") (ne .b "y") }}` |

### Registry tests (`internal/executor/registry_test.go`)

| # | Test Name | Asserts |
|---|-----------|---------|
| 40 | `TestMapRegistry_RegisterLookup` | registered executor is found |
| 41 | `TestMapRegistry_LookupMissing` | returns nil for unregistered kind |
| 42 | `TestMapRegistry_ConcurrentAccess` | no races under parallel Register+Lookup |
| 43 | `TestNewDefaultRegistry_AllKindsRegistered` | 12 kinds registered (all except parallel, wait_for_event) |

### InputProvider tests (`pkg/testutil/fake_input_provider_test.go`)

| # | Test Name | Asserts |
|---|-----------|---------|
| 44 | `TestFakeInput_PreProgrammedChoice` | returns registered response |
| 45 | `TestFakeInput_DefaultOnMissing` | returns default for unregistered step |
| 46 | `TestFakeInput_CallRecording` | Calls slice tracks all invocations |

---

## Fixture Integration

### Existing fixtures → executor coverage

| Fixture | Step Types Exercised | Executors Tested |
|---------|---------------------|-----------------|
| r01-k8s-incident | cli, wait_for_event, parallel, branch, collector | cli, branch, collector (parallel/wait_for_event = engine-native) |
| r02-canary-deploy | assert, collector, compensate, parallel, cli, iterate, tool | assert, collector, compensate, cli, iterate, tool |
| r03-employee-onboarding | collector, parallel, tool | collector, tool |
| r04-soc2-evidence | cli, include, collector | cli, include, collector |
| r05-security-breach | wait_for_event, choice, branch, collector, compensate, parallel, tool | choice, branch, collector, compensate, tool |
| r06-db-migration | assert, cli, collector, compensate | assert, cli, collector, compensate |
| r07-financial-approval | collector, branch | collector, branch |
| r08-fda-release | collector | collector |
| r09-oncall-escalation | iterate, branch, cli | iterate, branch, cli |
| r10-gdpr-deletion | collector, parallel, cli | collector, cli |
| r11-iterate-loop | cli, iterate | cli, iterate |
| r12-approval-quorum | cli, approve | cli, approve |
| r13-decision-routing | cli, decision, branch | cli, decision, branch |

### Coverage gaps — new fixtures needed

| Fixture | Purpose | Step Types |
|---------|---------|------------|
| r14-end-outcomes | Tests `end` step with success/failure/rollback outcomes | end, cli |
| r15-choice-multiselect | Tests multi-select choice with min/max constraints | choice, cli |
| r16-assert-types | Tests all assertion types (eq, ne, contains, matches, exists) | assert, cli |
| r17-compensate-trigger | Tests compensation execution on failure | compensate, cli |
| r18-branch-else | Tests branch with else arm and no matching condition | branch, cli |

### Missing from fixtures entirely

| Executor | Status | Remedy |
|----------|--------|--------|
| `end` | No fixture uses `type: end` | Add r14-end-outcomes |
| `choice` (multi) | r05 has choice but not multi-select | Add r15-choice-multiselect |

---

## Open Questions for Brian

1. **SubStepRunner callback wiring:** The branch and iterate executors need a `SubStepRunner` callback that the engine provides. The cleanest approach is to wire this in `NewDefaultRegistry()` by passing a closure over the engine's `executeStep` method. Does this cause any concerns about the engine importing `internal/executor` and passing a self-reference?

2. **Capture map processing:** `Step.Capture` maps output keys to var names (e.g., `capture: { init_msg: stdout }`). Should capture processing live in the executor (each executor reads it from the step), or in the engine (post-execution, engine reads `step.Capture` and maps `Output` keys to `Vars`)? Engine-level processing is DRYer but requires the engine to know about Capture semantics.

3. **Compensate LIFO execution:** The engine needs to scan for `__compensation_*` vars and execute them on failure. Should this be a distinct method on the engine (`executeCompensations`) or integrated into the existing `failRun` method?

4. **Decision step goto dispatch:** The decision executor stores `Output["route_goto"]` but the engine currently advances linearly (`CurrentStepIndex++`). Jump-to-step-by-ID requires engine support. Is this Phase 5 or Phase 6?

5. **CLISpec.Run field:** Fixtures don't use `Run` (they use `Command` + `Args`). Should the CLI executor support `Run` in Phase 5, or defer it as the field's semantics (string vs. map) need clarification?

---

## Correctness Checklist (pre-commit self-review)

- [x] No circular imports: `internal/executor` imports `pkg/engine`, `pkg/schema`, `pkg/expr`, `pkg/input`, `pkg/governance`, `pkg/platform` — NOT `internal/engine`
- [x] CLI executor uses `Platform.Exec()` for subprocess (testable via `FakePlatform`)
- [x] Expression evaluator is a stateless value type (`TemplateEvaluator` struct, no fields)
- [x] InputProvider is an interface (injectable, `FakeInputProvider` for tests)
- [x] `include` = registered no-op pass-through (D6)
- [x] `parallel` = not registered, engine-native (D7)
- [x] `wait_for_event` = not registered, engine-native (D7)
- [x] `end` sets `__run_outcome_*` vars and `Output["terminal"]` marker
- [x] ExecutorRegistry concrete type (`MapRegistry`) lives in `internal/executor/` (D8)
- [x] All 14 types covered — none skipped: 12 registered executors + 2 engine-native
