# Design Note: Switch from Go-Template Polish Notation to Infix Expressions

**Author:** Ken (Architect)  
**Date:** 2026-07-16  
**Requested by:** Cristian  
**Implementor:** Brian  
**Library:** `expr-lang/expr` (github.com/expr-lang/expr)

---

## 1. Motivation

The current expression evaluator uses Go `text/template` with Polish/prefix notation:
```
{{ and (eq .env "prod") (gt .count 1) }}
```

This is unreadable for operators writing runbooks. The replacement uses `expr-lang/expr` for natural infix syntax:
```
env == "prod" && count > 1
```

---

## 2. What Changes

### 2.1 `v2/pkg/expr/expr.go` — Public Interfaces (NO CHANGE)

```go
type Evaluator interface {
    Eval(tmpl string, vars map[string]any) (string, error)
}

type ConditionEvaluator interface {
    EvalBool(condition string, vars map[string]any) (bool, error)
}
```

These signatures are **unchanged**. Only internal implementations change.

### 2.2 `v2/internal/expr/template.go` — TemplateEvaluator

**Keeps existing implementation.** `TemplateEvaluator` continues to use `text/template` for string interpolation (`Evaluator.Eval`). This handles shell command templates like:
```
kubectl get pod {{ .pod_name }} -n {{ .namespace }}
```

No changes needed here.

### 2.3 `v2/internal/expr/expr_evaluator.go` — NEW: ExprEvaluator

New file. Implements `ConditionEvaluator` using `expr-lang/expr`:

```go
package expr

import (
    "fmt"
    "strings"

    "github.com/expr-lang/expr"
    pkgExpr "github.com/ormasoftchile/gert/v2/pkg/expr"
)

// ExprEvaluator evaluates boolean conditions using expr-lang/expr (infix syntax).
type ExprEvaluator struct{}

var _ pkgExpr.ConditionEvaluator = (*ExprEvaluator)(nil)

func (e *ExprEvaluator) EvalBool(condition string, vars map[string]any) (bool, error) {
    cond := strings.TrimSpace(condition)
    if cond == "" {
        return true, nil // empty condition = always true (preserve existing behavior)
    }

    program, err := expr.Compile(cond, expr.Env(vars), expr.AsBool())
    if err != nil {
        return false, fmt.Errorf("compile condition %q: %w", cond, err)
    }

    output, err := expr.Run(program, vars)
    if err != nil {
        return false, fmt.Errorf("eval condition %q: %w", cond, err)
    }

    result, ok := output.(bool)
    if !ok {
        return false, fmt.Errorf("condition %q: expected bool, got %T", cond, output)
    }
    return result, nil
}
```

### 2.4 `v2/internal/expr/condition.go` — SimpleConditionEvaluator

**Replace entire implementation.** The old code wraps conditions in `{{ if ... }}true{{ else }}false{{ end }}` and delegates to `TemplateEvaluator`. The new implementation:

```go
package expr

import (
    "fmt"
    "strings"

    "github.com/expr-lang/expr"
    pkgExpr "github.com/ormasoftchile/gert/v2/pkg/expr"
)

// SimpleConditionEvaluator evaluates boolean conditions using expr-lang/expr.
type SimpleConditionEvaluator struct{}

var _ pkgExpr.ConditionEvaluator = (*SimpleConditionEvaluator)(nil)

func NewSimpleConditionEvaluator(_ pkgExpr.Evaluator) *SimpleConditionEvaluator {
    return &SimpleConditionEvaluator{}
}

func (e *SimpleConditionEvaluator) EvalBool(condition string, vars map[string]any) (bool, error) {
    cond := strings.TrimSpace(condition)
    if cond == "" {
        return true, nil
    }

    program, err := expr.Compile(cond, expr.Env(vars), expr.AsBool())
    if err != nil {
        return false, fmt.Errorf("compile condition %q: %w", cond, err)
    }

    output, err := expr.Run(program, vars)
    if err != nil {
        return false, fmt.Errorf("eval condition %q: %w", cond, err)
    }

    result, ok := output.(bool)
    if !ok {
        return false, fmt.Errorf("condition %q: expected bool, got %T", cond, output)
    }
    return result, nil
}
```

**Note:** `NewSimpleConditionEvaluator` retains the `Evaluator` parameter in its signature for backward compat with call sites in `registry.go`, but no longer uses it. Alternatively, Brian may choose to unify into a single `ExprEvaluator` type and update `registry.go` accordingly.

### 2.5 `v2/internal/expr/*_test.go` — Update All Tests

All condition strings change from Polish to infix:

| Old test string | New test string |
|----------------|----------------|
| `{{ and (eq .env "prod") (ne .status "bad") (gt .count 1) (lt .count 10) }}` | `env == "prod" && status != "bad" && count > 1 && count < 10` |
| `{{ or (eq .a "x") (not (eq .b "y")) }}` | `a == "x" \|\| b != "y"` |
| `$.flag` | `flag` |

### 2.6 `design/gert-v2/testdata/` — Fixture Runbooks

All `condition:` and `when:` values in test fixture YAML files must be updated.

**Files affected:**
- `testdata/runbooks/r01-k8s-incident/schema.yaml` (5 expressions)
- `testdata/runbooks/r02-canary-deploy/schema.yaml` (3 expressions)
- `testdata/runbooks/r03-employee-onboarding/schema.yaml` (5 expressions)
- `testdata/runbooks/r07-financial-approval/schema.yaml` (4 expressions)

**Example transformations:**
```yaml
# OLD
condition: '{{ or (contains .pod_json "CrashLoopBackOff") (contains .pod_json "ImagePullBackOff") }}'
# NEW
condition: 'contains(pod_json, "CrashLoopBackOff") || contains(pod_json, "ImagePullBackOff")'

# OLD
when: '{{ eq .office_location "Remote" }}'
# NEW
when: 'office_location == "Remote"'

# OLD
condition: '{{ and (gt .amount "50000") (lt .amount "100000") }}'
# NEW
condition: 'amount > 50000 && amount < 100000'
```

**Important:** With expr-lang, numeric comparisons use actual numbers, not string comparison. The fixture runbooks will need schema adjustments where `amount` should be typed as number, not string.

---

## 3. Expression Syntax Reference (for Brian)

### 3.1 Migration Table

| Old (Polish / Go template) | New (infix / expr-lang) |
|---------------------------|------------------------|
| `{{ eq .env "prod" }}` | `env == "prod"` |
| `{{ ne .status "bad" }}` | `status != "bad"` |
| `{{ gt .count 1 }}` | `count > 1` |
| `{{ lt .count 10 }}` | `count < 10` |
| `{{ ge .count 5 }}` | `count >= 5` |
| `{{ le .count 10 }}` | `count <= 10` |
| `{{ and (eq .a "x") (gt .b 0) }}` | `a == "x" && b > 0` |
| `{{ or (eq .a "x") (not (eq .b "y")) }}` | `a == "x" \|\| b != "y"` |
| `{{ not .flag }}` | `!flag` |
| `{{ contains .s "substr" }}` | `contains(s, "substr")` |
| `$.flag` (JSONPath-style) | `flag` |
| `{{ if .flag }}true{{ else }}false{{ end }}` | `flag` |

### 3.2 Variable Access Convention

- **Old:** `.varname` (Go template dot notation) or `$.varname` (JSONPath-style)
- **New:** `varname` (direct access, no prefix)

`expr-lang/expr` uses the variable map directly. `vars["env"]` is accessed as just `env` in expressions. No `.` prefix, no `$` prefix.

The `$` JSONPath prefix support is **dropped** — it was a convenience shim for a single test case and adds no value with the new syntax.

### 3.3 Supported Operators

| Category | Operators |
|----------|-----------|
| Comparison | `==`, `!=`, `>`, `<`, `>=`, `<=` |
| Logical | `&&`, `\|\|`, `!` |
| Arithmetic | `+`, `-`, `*`, `/`, `%` |
| Grouping | `(...)` |
| String | `contains()`, `startsWith()`, `endsWith()`, `len()` |
| Membership | `in` (e.g., `"x" in list`) |
| Ternary | `condition ? a : b` |

### 3.4 Built-in Functions (from expr-lang/expr)

- `len(x)` — length of string, array, or map
- `contains(haystack, needle)` — string or array contains
- `startsWith(s, prefix)` — string starts with prefix
- `endsWith(s, suffix)` — string ends with suffix
- `upper(s)`, `lower(s)` — case conversion
- `trim(s)` — whitespace trim
- `matches(s, regex)` — regex match (returns bool)

---

## 4. Implementation Notes for Brian

### 4.1 Core Pattern

```go
import "github.com/expr-lang/expr"

func evalBool(condition string, vars map[string]any) (bool, error) {
    program, err := expr.Compile(condition, expr.Env(vars), expr.AsBool())
    if err != nil {
        return false, err
    }
    output, err := expr.Run(program, vars)
    if err != nil {
        return false, err
    }
    return output.(bool), nil
}
```

### 4.2 Key Behaviors

1. **Empty condition → true** (preserve existing behavior)
2. **Non-boolean result → hard error** (not silent false)
3. **Unknown variable → compile error** (expr-lang validates against `expr.Env(vars)`)
4. **Compile errors → wrapped with condition string** for debuggability

### 4.3 expr.AsBool() Enforcement

Use `expr.AsBool()` as a compile option. This tells expr-lang the expression MUST return bool, and it will produce a compile-time error if the expression's return type is not bool.

### 4.4 Thread Safety

`expr.Compile()` returns a `*vm.Program` which is safe for concurrent use. If we want to cache compiled programs (future optimization), the Program can be stored and reused across goroutines.

### 4.5 go.mod Addition

```
require github.com/expr-lang/expr v1.16.x
```

(Use latest stable at time of implementation.)

---

## 5. Interface Compatibility

### 5.1 `pkg/expr` — No Changes

```go
// Evaluator resolves {{template}} expressions in step field values.
type Evaluator interface {
    Eval(tmpl string, vars map[string]any) (string, error)
}

// ConditionEvaluator evaluates boolean condition expressions.
type ConditionEvaluator interface {
    EvalBool(condition string, vars map[string]any) (bool, error)
}
```

Both interfaces are **unchanged**. The `ConditionEvaluator.EvalBool` signature takes a condition string and vars map — this works perfectly with expr-lang since the condition string is now natural infix syntax and the vars map is passed directly to `expr.Env()` and `expr.Run()`.

### 5.2 Call Sites — No Changes Required

All executor call sites use the interface:
- `v2/internal/executor/branch.go` — `evalCondition(e.condition, arm.Condition, vars)`
- `v2/internal/executor/iterate.go` — `evalCondition(e.condition, spec.Until, workingVars)`
- `v2/internal/executor/collector.go` — `evalCondition(e.condition, field.When, vars)`
- `v2/internal/executor/helpers.go` — `func evalCondition(cond expr.ConditionEvaluator, ...)`
- `v2/internal/executor/registry.go` — wires `ConditionEvaluator` into executors

None of these care about the condition syntax — they pass opaque strings to `EvalBool`.

---

## 6. Decision: String Interpolation (D4)

**Decision: Keep `text/template` for string interpolation.**

Rationale:
- Shell command templates (`kubectl get pod {{ .pod_name }}`) use Go template syntax naturally
- `args` fields reference variables with `{{ .var }}` — this is string interpolation, not boolean logic
- Changing string interpolation to expr-lang would require a different syntax (e.g., `${varname}`) and touches far more surface area
- Minimal blast radius: only boolean condition evaluation changes

**What stays on text/template:**
- `Evaluator.Eval()` — string interpolation for shell commands, instructions, etc.
- `TemplateEvaluator` — unchanged

**What moves to expr-lang:**
- `ConditionEvaluator.EvalBool()` — boolean conditions in `condition:` and `when:` fields
- `SimpleConditionEvaluator` — new implementation using expr-lang

---

## 7. Test Plan for Brian

### 7.1 Unit Tests (ExprEvaluator / SimpleConditionEvaluator)

| # | Test Case | Expression | Vars | Expected |
|---|-----------|-----------|------|----------|
| 1 | Empty condition | `""` | `{}` | `true, nil` |
| 2 | Simple equality (true) | `env == "prod"` | `{"env": "prod"}` | `true, nil` |
| 3 | Simple equality (false) | `env == "prod"` | `{"env": "staging"}` | `false, nil` |
| 4 | Inequality | `status != "bad"` | `{"status": "ok"}` | `true, nil` |
| 5 | Greater than | `count > 1` | `{"count": 5}` | `true, nil` |
| 6 | Less than | `count < 10` | `{"count": 5}` | `true, nil` |
| 7 | Greater-or-equal | `count >= 5` | `{"count": 5}` | `true, nil` |
| 8 | Less-or-equal | `count <= 10` | `{"count": 10}` | `true, nil` |
| 9 | Boolean AND | `a == "x" && b > 0` | `{"a": "x", "b": 3}` | `true, nil` |
| 10 | Boolean OR | `a == "x" \|\| b != "y"` | `{"a": "z", "b": "z"}` | `true, nil` |
| 11 | NOT | `!flag` | `{"flag": false}` | `true, nil` |
| 12 | contains() | `contains(s, "sub")` | `{"s": "substring"}` | `true, nil` |
| 13 | startsWith() | `startsWith(s, "pre")` | `{"s": "prefix"}` | `true, nil` |
| 14 | len() | `len(s) > 3` | `{"s": "hello"}` | `true, nil` |
| 15 | Non-boolean → error | `count + 1` | `{"count": 5}` | `false, error` |
| 16 | Unknown variable → error | `unknown == "x"` | `{}` | `false, error` |
| 17 | Nested grouping | `(a == "x" \|\| b == "y") && count > 0` | `{"a": "z", "b": "y", "count": 1}` | `true, nil` |
| 18 | Bare bool variable | `flag` | `{"flag": true}` | `true, nil` |
| 19 | Syntax error → error | `== bad` | `{}` | `false, error` |
| 20 | String in condition | `env in ["prod", "staging"]` | `{"env": "prod"}` | `true, nil` |

### 7.2 Integration Scope

- Verify `BranchExecutor` evaluates infix `condition:` strings correctly
- Verify `IterateExecutor` evaluates infix `until:` strings correctly
- Verify `CollectorExecutor` evaluates infix `when:` strings correctly
- Run existing integration tests after fixture update

---

## 8. Migration Checklist

- [ ] Add `github.com/expr-lang/expr` to `v2/go.mod`
- [ ] Create `v2/internal/expr/expr_evaluator.go` (or refactor `condition.go` in place)
- [ ] Remove `{{ }}` wrapping logic from `SimpleConditionEvaluator`
- [ ] Remove `$.` JSONPath shim
- [ ] Update `condition_test.go` — all test strings to infix
- [ ] Update testdata runbook YAML files (4 files, ~17 expressions)
- [ ] Update `design/gert-v2/sections/03-schema-vnext.tex` §Expression Language
- [ ] Run `go test ./...` — all green
- [ ] Update design doc examples that reference old syntax

---

## 9. Risks and Mitigations

| Risk | Mitigation |
|------|-----------|
| expr-lang dependency adds weight | Single dependency, zero transitive deps, well-maintained (5k+ stars) |
| Existing runbooks break | All in-repo fixtures updated as part of this change; v2 has no external users yet |
| Type coercion surprises | Use `expr.AsBool()` to enforce bool return; document that numeric comparisons need actual numbers |
| String values compared as numbers in old fixtures | Audit fixtures — some use `gt .amount "50000"` (string comparison); convert to numeric with typed vars |

---

## 10. Out of Scope

- Custom function registration (future: can add gert-specific functions to expr environment)
- Expression caching/compilation caching (future optimization)
- Changes to `Evaluator.Eval()` / `TemplateEvaluator` (stays on text/template)
- Design doc LaTeX rewrite (separate PR after implementation lands)
