# Condition Syntax Feasibility Report: Substring Check Functions

**Author:** Brian (Go Programmer)  
**Date:** 2026-04-24  
**Context:** gert v2 condition evaluator using expr-lang/expr v1.17.8

---

## Executive Summary

**Problem:** Runbook authors need to check if a string variable contains a substring in branch/iterate conditions. The naive function name `contains` is a **reserved keyword** in expr-lang and cannot be used as a function name (only as infix operator).

**Tested:** I verified 4 implementation approaches using live expr-lang experimentation. **Options A, B, and C are all feasible.** Option D is not viable.

**Recommendation:** **Option B (Namespace approach)** — provides the best balance of discoverability, future extensibility, and avoiding reserved word conflicts.

---

## Current Implementation Context

**File:** `internal/expr/condition.go`

The condition evaluator:
1. Uses `map[string]any` as the environment
2. Registers helper functions in `conditionBuiltins` map (lines 12-18)
3. Merges user variables + builtins into the env (lines 39-46)
4. Compiles with `expr.Compile(cond, expr.Env(env), expr.AsBool())`
5. Evaluates with `expr.Run(program, env)`

**Current builtins:**
```go
var conditionBuiltins = map[string]any{
    "contains":  strings.Contains,  // ❌ BROKEN - reserved keyword
    "hasPrefix": strings.HasPrefix,
    "hasSuffix": strings.HasSuffix,
    "toLower":   strings.ToLower,
    "toUpper":   strings.ToUpper,
}
```

**Test evidence (line 71 of condition_test.go):**
```go
{
    name:      "string contains",
    condition: `s contains "sub"`,  // ✅ Works as INFIX operator
    vars:      map[string]any{"s": "substring"},
    want:      true,
},
```

The test uses `s contains "sub"` (infix operator syntax), NOT `contains(s, "sub")` (function call).

---

## Option A: Rename Approach

**Strategy:** Register `strings.Contains` under a non-reserved function name.

### Implementation

**Code change (condition.go):**
```go
var conditionBuiltins = map[string]any{
    "strContains": strings.Contains,  // Renamed from "contains"
    "hasPrefix":   strings.HasPrefix,
    "hasSuffix":   strings.HasSuffix,
    "toLower":     strings.ToLower,
    "toUpper":     strings.ToUpper,
}
```

**Runbook usage:**
```yaml
steps:
  - name: check-env
    if: strContains(environment, "prod")
    # OR
    if: hasSubstr(branch_name, "hotfix")
```

### Tested Function Names

All work correctly:
- ✅ `strContains(x, "y")` — explicit string prefix
- ✅ `hasSubstr(x, "y")` — matches Ruby/JS `.includes()` semantics
- ✅ `includes(x, "y")` — familiar to JS/Python developers

### Feasibility: ✅ CONFIRMED

Verified via live expr-lang test. All three function names compile and execute correctly.

### Tradeoffs

**Pros:**
- Minimal code change (one-line fix)
- No API surface area increase
- Consistent with existing function registration pattern
- Zero performance overhead

**Cons:**
- Discoverability: Users might try `contains()` first and get a confusing error
- No clear place to add future string helpers (each becomes a top-level function)
- Name bikeshedding: Team must choose one canonical name

**Migration impact:**
- Existing runbooks using infix `s contains "sub"` continue to work (backward compatible)
- New runbooks use function-call syntax `strContains(s, "sub")`

---

## Option B: Namespace Approach ⭐ RECOMMENDED

**Strategy:** Register a `StringHelpers` struct in the env under the key `"str"`. Methods become callable as `str.Contains(x, "y")`.

### Implementation

**Code change (condition.go):**
```go
// StringHelpers provides string utility functions for condition expressions.
type StringHelpers struct{}

func (StringHelpers) Contains(s, substr string) bool {
    return strings.Contains(s, substr)
}

func (StringHelpers) HasPrefix(s, prefix string) bool {
    return strings.HasPrefix(s, prefix)
}

func (StringHelpers) HasSuffix(s, suffix string) bool {
    return strings.HasSuffix(s, suffix)
}

func (StringHelpers) ToLower(s string) string {
    return strings.ToLower(s)
}

func (StringHelpers) ToUpper(s string) string {
    return strings.ToUpper(s)
}

var conditionBuiltins = map[string]any{
    "str": StringHelpers{},
}
```

**Runbook usage:**
```yaml
steps:
  - name: check-env
    if: str.Contains(environment, "prod")
  
  - name: check-branch
    if: str.HasPrefix(branch, "release/")
  
  - name: normalize
    # Can combine with other operations
    if: str.ToLower(status) == "ok"
```

### Feasibility: ✅ CONFIRMED

Verified via live expr-lang test. Method calls on struct values work perfectly.

**Test output:**
```
✓ str.Contains(status, "prod") = true
✓ str.HasPrefix(status, "prod") = true
✓ str.HasSuffix(status, "ion") = true
```

### Tradeoffs

**Pros:**
- **Discoverability:** `str.` prefix makes string functions obvious (IDE autocomplete-friendly)
- **Namespace safety:** No collision with reserved words or future expr-lang keywords
- **Extensibility:** Easy to add `str.Trim()`, `str.Split()`, `str.Replace()` later
- **Consistency:** Mirrors Go's `strings.Contains()` package structure
- **Future-proof:** Can add `math.`, `time.`, `json.` namespaces without polluting top-level env

**Cons:**
- Slightly more verbose: `str.Contains()` vs `contains()`
- Requires defining a helper struct (10-20 LOC)
- Two breaking changes from current state:
  1. Remove broken `"contains"` function registration
  2. Move other string functions into `str.` namespace OR leave as top-level (inconsistent)

**Consistency decision required:**
- **Option B1 (Recommended):** Move ALL string functions to `str.` namespace
  - Remove: `hasPrefix`, `hasSuffix`, `toLower`, `toUpper` from top-level
  - Migration: `hasPrefix(x, "y")` → `str.HasPrefix(x, "y")`
- **Option B2 (Gradual):** Keep existing functions at top-level, add `str.Contains()` only
  - Inconsistent but backward-compatible

### Migration Impact (B1)

**Breaking change checklist:**
1. Top-level functions removed: `hasPrefix`, `hasSuffix`, `toLower`, `toUpper`
2. New namespace: `str.Contains()`, `str.HasPrefix()`, `str.HasSuffix()`, `str.ToLower()`, `str.ToUpper()`
3. Infix operator `s contains "sub"` still works (expr-lang built-in)

**Mitigation:** Add a deprecation notice in v2.0 docs, full migration in v2.1.

---

## Option C: Typed Env Struct

**Strategy:** Replace `map[string]any` with a typed struct. Fields are variables, methods are functions.

### Implementation

**Code change (condition.go):**
```go
// ConditionEnv is the typed environment for condition evaluation.
type ConditionEnv struct {
    // User variables (populated dynamically)
    Vars map[string]any
    
    // Namespace helpers
    Str StringHelpers
}

// Contains is a top-level method on the env.
func (e ConditionEnv) Contains(s, substr string) bool {
    return strings.Contains(s, substr)
}

// GetField allows dynamic variable access (required for expr-lang).
func (e ConditionEnv) GetField(name string) (any, bool) {
    val, ok := e.Vars[name]
    return val, ok
}

func (e *SimpleConditionEvaluator) EvalBool(condition string, vars map[string]any) (bool, error) {
    // ...existing validation...
    
    env := ConditionEnv{
        Vars: vars,
        Str:  StringHelpers{},
    }
    
    // CRITICAL: expr-lang requires a map[string]any for variable lookup.
    // Typed structs work for METHODS but not FIELDS from user vars.
    // This approach FAILS for dynamic variables.
    
    program, err := exprlang.Compile(cond, exprlang.Env(env), exprlang.AsBool())
    // ...
}
```

### Feasibility: ⚠️ PARTIAL

**What works:**
- ✅ Methods on the env: `Contains(status, "prod")`
- ✅ Namespaced helpers: `Str.Contains(status, "prod")`

**What breaks:**
- ❌ Dynamic user variables: `environment == "prod"`
  - expr-lang expects variables to be struct fields OR map entries
  - Runtime variables from runbook execution are dynamic (not known at compile time)
  - Workaround: Use `Vars.environment` for all variable access (ugly)

**Test output:**
```
✓ Status == "production" = true       // Static struct field
✓ Count > 40 = true                   // Static struct field
✓ Str.Contains(Status, "prod") = true // Method call
✓ Contains(Status, "prod") = true     // Top-level method
```

### Critical Issue: Dynamic Variables

Runbook conditions reference variables set at runtime:
```yaml
steps:
  - name: dns-check
    tool: shell
    action: run
    spec:
      cmd: nslookup example.com
      capture_output: dns_result  # Dynamic variable
  
  - name: conditional-step
    if: dns_result contains "not found"  # Needs runtime variable lookup
```

**Problem:** `dns_result` is not known until step execution. Typed struct fields must be declared at compile time.

**Workaround:** Force prefix: `Vars.dns_result contains "not found"`
- Breaks ergonomics (every variable needs `Vars.` prefix)
- Inconsistent with current design

### Tradeoffs

**Pros:**
- Type safety for helper methods
- IDE autocomplete for method names
- Methods can be named `Contains` (no collision)

**Cons:**
- **BLOCKER:** Requires `Vars.` prefix for all dynamic variables
- Significant refactor (20+ lines of env construction logic)
- Loss of ergonomics: `Vars.environment` vs `environment`
- Incompatible with existing runbooks without migration script

**Verdict:** ❌ NOT RECOMMENDED due to dynamic variable access issue.

---

## Option D: expr-lang Operator/Patch

**Strategy:** Modify expr-lang lexer to allow `contains` as a function name, or register a custom operator.

### Investigation

**Lexer analysis (github.com/expr-lang/expr@v1.17.8):**
- `contains` is a **reserved keyword token** in `parser/lexer/lexer.go`
- Used for infix operator: `"string" contains "sub"` → calls builtin `strings.Contains`
- Token type: `Operator("contains")`

**Attempted workarounds:**

1. **Custom operators:** expr-lang does not support registering custom operators (lexer is closed)
2. **Lexer fork:** Would require maintaining a fork of expr-lang (maintenance burden)
3. **Preprocess conditions:** Transform `contains(x, y)` → `x contains y` before compilation
   - Fragile (breaks on nested calls, complex expressions)
   - Regex-based transformation is error-prone

**Test result:**
```
✗ contains(status, "prod"): compile error: unexpected token Operator("contains") (1:1)
 | contains(status, "prod")
 | ^
```

The lexer sees `contains` and emits an `Operator` token, expecting infix usage. Function call context is irrelevant.

### Feasibility: ❌ NOT VIABLE

**Reasons:**
1. expr-lang does not expose operator customization API
2. Forking expr-lang creates a maintenance burden (security patches, upgrades)
3. Runtime preprocessing is fragile and error-prone

**Note:** The infix operator `s contains "sub"` already works. The issue is function-call syntax for consistency with other helpers like `hasPrefix(s, "x")`.

---

## Comparison Matrix

| Criterion | Option A (Rename) | Option B (Namespace) | Option C (Typed Env) | Option D (Patch) |
|-----------|-------------------|----------------------|----------------------|------------------|
| **Feasibility** | ✅ Confirmed | ✅ Confirmed | ⚠️ Partial (Vars. prefix) | ❌ Not viable |
| **Code change** | 1 line | 20 lines | 40+ lines | Fork expr-lang |
| **Ergonomics** | Good (`strContains`) | Excellent (`str.Contains`) | Poor (`Vars.x`) | N/A |
| **Discoverability** | Medium | High (`str.` prefix) | High (IDE) | N/A |
| **Extensibility** | Low (top-level pollution) | High (namespaces) | Medium | N/A |
| **Backward compat** | ✅ Infix still works | ⚠️ Breaking (if migrate all) | ❌ Breaking | N/A |
| **Maintenance** | Low | Low | Medium (custom env logic) | High (fork) |
| **Consistency** | Medium (mix of styles) | High (all under `str.`) | High (typed) | N/A |

---

## Recommendation

**Adopt Option B (Namespace approach) with full migration to `str.` namespace (B1).**

### Rationale

1. **Discoverability:** `str.Contains()` is intuitive for ops engineers familiar with Go/JS/Python
2. **Future-proof:** Can add `math.`, `time.`, `file.` namespaces without keyword conflicts
3. **Consistency:** All string functions under one namespace (no top-level pollution)
4. **Extensibility:** Easy to add `str.Trim()`, `str.Split()`, `str.Join()` later

### Implementation Plan

**Phase 1 (Immediate):**
```go
// internal/expr/condition.go

type StringHelpers struct{}

func (StringHelpers) Contains(s, substr string) bool     { return strings.Contains(s, substr) }
func (StringHelpers) HasPrefix(s, prefix string) bool    { return strings.HasPrefix(s, prefix) }
func (StringHelpers) HasSuffix(s, suffix string) bool    { return strings.HasSuffix(s, suffix) }
func (StringHelpers) ToLower(s string) string            { return strings.ToLower(s) }
func (StringHelpers) ToUpper(s string) string            { return strings.ToUpper(s) }
func (StringHelpers) TrimSpace(s string) string          { return strings.TrimSpace(s) }

var conditionBuiltins = map[string]any{
    "str": StringHelpers{},
    
    // Deprecated (remove in v2.1):
    // "hasPrefix": strings.HasPrefix,
    // "hasSuffix": strings.HasSuffix,
    // "toLower":   strings.ToLower,
    // "toUpper":   strings.ToUpper,
}
```

**Phase 2 (Add tests):**
```go
// internal/expr/condition_test.go

{
    name:      "str.Contains - positive match",
    condition: `str.Contains(status, "prod")`,
    vars:      map[string]any{"status": "production"},
    want:      true,
},
{
    name:      "str.Contains - negative match",
    condition: `str.Contains(status, "dev")`,
    vars:      map[string]any{"status": "production"},
    want:      false,
},
{
    name:      "str.HasPrefix",
    condition: `str.HasPrefix(branch, "release/")`,
    vars:      map[string]any{"branch": "release/v2.0"},
    want:      true,
},
{
    name:      "str.ToLower normalization",
    condition: `str.ToLower(env) == "prod"`,
    vars:      map[string]any{"env": "PROD"},
    want:      true,
},
{
    name:      "infix contains still works",
    condition: `status contains "prod"`,
    vars:      map[string]any{"status": "production"},
    want:      true,
},
```

**Phase 3 (Documentation):**
- Add `str.` examples to runbook authoring guide
- Document infix `contains` operator as alternative
- Migration guide: `hasPrefix(x, "y")` → `str.HasPrefix(x, "y")`

### Migration Strategy

**v2.0 (Current):**
- Add `str.Contains()` (NEW)
- Keep `hasPrefix()`, `hasSuffix()`, etc. (DEPRECATED)
- Document both syntaxes

**v2.1 (Next release):**
- Remove deprecated top-level functions
- `str.` namespace becomes canonical

**Breakage mitigation:**
- Infix `s contains "sub"` continues to work (expr-lang built-in)
- Ops engineers have 2 syntaxes during transition period

---

## Alternative: Quick Fix (Option A)

If team prefers minimal change for v2.0, use **Option A with `strContains`** as interim solution:

```go
var conditionBuiltins = map[string]any{
    "strContains": strings.Contains,  // Quick fix
    "hasPrefix":   strings.HasPrefix,
    "hasSuffix":   strings.HasSuffix,
    "toLower":     strings.ToLower,
    "toUpper":     strings.ToUpper,
}
```

**Defer full namespace migration to v2.1.**

---

## Appendix: Live Test Results

```
=== Option A: Rename approach (strContains) ===
✓ strContains(status, "prod") = true
✓ hasSubstr(status, "duct") = true
✓ includes(status, "ion") = true

=== Option B: Namespace approach (str.Contains) ===
✓ str.Contains(status, "prod") = true
✓ str.HasPrefix(status, "prod") = true
✓ str.HasSuffix(status, "ion") = true

=== Option C: Typed env struct ===
✓ Status == "production" = true
✓ Count > 40 = true
✓ Str.Contains(Status, "prod") = true
✓ Contains(Status, "prod") = true

=== Option D: 'contains' as reserved word ===
✗ contains(status, "prod"): compile error: unexpected token Operator("contains") (1:1)
 | contains(status, "prod")
 | ^
✓ status contains "prod" = true
```

**Test methodology:** Compiled and executed expr-lang programs with github.com/expr-lang/expr v1.17.8 (current gert v2 dependency).

---

## Questions for Team Decision

1. **Namespace commitment:** Full migration to `str.` namespace in v2.0, or gradual in v2.1?
2. **Function name (if Option A):** `strContains`, `hasSubstr`, or `includes`?
3. **Backward compat:** Deprecation period for top-level functions, or immediate removal?
4. **Documentation:** Should we recommend infix (`s contains "x"`) or function call (`str.Contains(s, "x")`) as primary syntax?

---

**End of Report**
