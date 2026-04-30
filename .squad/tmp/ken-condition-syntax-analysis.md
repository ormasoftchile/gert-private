# Condition Syntax Design Analysis — String Operations in gert v2

**Author:** Ken (Software Architect)  
**Date:** 2025-01-21  
**Context:** Runbook condition expressions using expr-lang v1.17.8  
**Requested by:** Cristian

---

## Problem Statement

Runbook authors need to check string containment (and related operations: starts_with, ends_with, case-insensitive matching) in `condition:` fields. The natural syntax `x contains "y"` is an **infix operator** in expr-lang, which the user has explicitly rejected as poor UX for runbook authors.

Attempting to use `contains(x, "y")` as a function call fails because `contains` is a reserved keyword in expr-lang's lexer, producing: `unexpected token Operator("contains")`.

**Technical constraints:**
- expr-lang v1.17.8 (no library change)
- Reserved keywords: `contains`, `matches`, `startsWith`, `endsWith`, `in`, `not`, `and`, `or`
- Functions can be registered as `map[string]any{"funcName": goFunc}`
- Non-keyword names work as function calls: `strContains(x, "y")` compiles fine
- Member-access notation is supported: `obj.Method(args...)`

---

## Design Options

### Option 1: Prefixed Function Names

**Syntax:**
```yaml
condition: 'str.contains(dns_output, "Address")'
condition: 'str.startsWith(filename, "/etc")'
condition: 'str.endsWith(path, ".yaml")'
condition: 'str.equalsIgnoreCase(method, "GET")'
```

**Implementation:**
Register a helper object with methods in the expr-lang environment:
```go
env := map[string]any{
    "str": &StringHelpers{},
}

type StringHelpers struct{}
func (s *StringHelpers) Contains(haystack, needle string) bool { ... }
func (s *StringHelpers) StartsWith(s, prefix string) bool { ... }
func (s *StringHelpers) EndsWith(s, suffix string) bool { ... }
func (s *StringHelpers) EqualsIgnoreCase(a, b string) bool { ... }
```

**Pros (UX):**
- ✅ Clear namespace separation (`str.` prefix indicates string operations)
- ✅ Reads like object-oriented method calls (familiar to many developers)
- ✅ Self-documenting: `str.contains` is unambiguous about what it operates on
- ✅ Autocomplete-friendly (tooling can suggest `str.*` methods)
- ✅ Avoids keyword conflicts by design (methods on an object)

**Cons (UX):**
- ❌ Slightly more verbose than bare function names
- ❌ Requires remembering the `str.` prefix

**Pros (Architecture):**
- ✅ Extensible: Can add `str.trim()`, `str.replace()`, `str.split()` later without name collisions
- ✅ Consistent pattern: Can apply to other domains (`date.add()`, `json.get()`, etc.)
- ✅ No naming conflicts with future expr-lang additions
- ✅ Type-safe: Go methods provide clear signatures

**Cons (Architecture):**
- ❌ Requires defining a helper struct per namespace (minor boilerplate)

---

### Option 2: Snake-Case Function Names

**Syntax:**
```yaml
condition: 'str_contains(dns_output, "Address")'
condition: 'str_starts_with(filename, "/etc")'
condition: 'str_ends_with(path, ".yaml")'
condition: 'str_equals_ignore_case(method, "GET")'
```

**Implementation:**
Register top-level functions with non-keyword names:
```go
env := map[string]any{
    "str_contains":            strings.Contains,
    "str_starts_with":         strings.HasPrefix,
    "str_ends_with":           strings.HasSuffix,
    "str_equals_ignore_case":  strEqualsFoldWrapper,
}
```

**Pros (UX):**
- ✅ Simple function call syntax (no dots or objects)
- ✅ Snake-case feels more "scripting-friendly" (common in ops/shell contexts)
- ✅ Clear prefix still indicates string operations

**Cons (UX):**
- ❌ Snake-case is less common in Go/YAML ecosystems (camelCase dominates)
- ❌ Name length increases (`str_starts_with` vs `str.startsWith` vs `strStartsWith`)

**Pros (Architecture):**
- ✅ Minimal implementation (direct function registration)
- ✅ Predictable naming scheme (`str_*` for string ops)

**Cons (Architecture):**
- ❌ Less extensible: No natural grouping for tooling/documentation
- ❌ Flat namespace: If we add 20 string functions, the environment gets cluttered
- ❌ No compile-time type safety (just `func(args ...any)` if not careful)

---

### Option 3: CamelCase Function Names

**Syntax:**
```yaml
condition: 'strContains(dns_output, "Address")'
condition: 'strStartsWith(filename, "/etc")'
condition: 'strEndsWith(path, ".yaml")'
condition: 'strEqualsIgnoreCase(method, "GET")'
```

**Implementation:**
Same as Option 2, but with camelCase:
```go
env := map[string]any{
    "strContains":          strings.Contains,
    "strStartsWith":        strings.HasPrefix,
    "strEndsWith":          strings.HasSuffix,
    "strEqualsIgnoreCase":  strEqualsFoldWrapper,
}
```

**Pros (UX):**
- ✅ CamelCase matches Go/YAML conventions (more familiar to target audience)
- ✅ Shorter than snake-case (`strStartsWith` vs `str_starts_with`)
- ✅ Clear prefix indicates string operations

**Cons (UX):**
- ❌ Runs words together without delimiter (less readable than dot notation)
- ❌ Looks like "invented syntax" rather than a standard pattern

**Pros (Architecture):**
- ✅ Simple implementation (direct function registration)
- ✅ Aligns with Go naming conventions

**Cons (Architecture):**
- ❌ Same namespace/extensibility issues as Option 2
- ❌ Harder to parse visually: `strEqualsIgnoreCase` vs `str.equalsIgnoreCase`

---

### Option 4: Hybrid — Object Namespaces + Helpers Module

**Syntax:**
```yaml
condition: 'str.contains(dns_output, "Address")'
condition: 'str.startsWith(filename, "/etc")'
condition: 'date.add(now, "1h")'
condition: 'json.get(response, "status.code")'
```

**Implementation:**
Same as Option 1, but establish it as a **framework-wide pattern**:
```go
env := map[string]any{
    "str":  &StringHelpers{},
    "date": &DateHelpers{},
    "json": &JSONHelpers{},
    // future: "math", "file", "net", etc.
}
```

**Pros (UX):**
- ✅ All benefits of Option 1 (clarity, readability, autocomplete)
- ✅ Establishes a **consistent pattern** across all helper domains
- ✅ Runbook authors learn one pattern (`namespace.method()`) that applies everywhere

**Cons (UX):**
- ❌ Same as Option 1 (requires remembering prefixes)

**Pros (Architecture):**
- ✅ **Future-proof:** Adding new domains (crypto, regex, etc.) follows the same pattern
- ✅ **Discoverability:** Tooling can introspect namespaces and suggest methods
- ✅ **Testability:** Each namespace is a separate struct, easy to unit test
- ✅ **Documentation:** Can generate reference docs per namespace
- ✅ **Minimal collision risk:** Namespaces isolate reserved words

**Cons (Architecture):**
- ❌ Requires upfront design of namespace boundaries (but this is a one-time cost)

---

## Recommendation: **Option 4 (Hybrid — Object Namespaces + Helpers Module)**

### Rationale

**Primary reasons:**

1. **UX clarity:** `str.contains(x, "y")` reads as clear English and avoids the "invented syntax" feel of `strContains`. The dot notation is familiar to anyone who has written JavaScript, Python, or Go.

2. **Consistency:** Establishing `namespace.method()` as the standard pattern for all helper functions creates a coherent mental model. When runbook authors need date math, JSON extraction, or regex matching, they'll look for `date.*`, `json.*`, `regex.*` — not guess at prefix conventions.

3. **Extensibility:** As gert v2 adds more helper functions (likely in v2.1+), the namespace approach prevents environment pollution. Without namespaces, we'd have 50+ top-level functions competing with variable names.

4. **Tooling support:** IDEs and expression editors can autocomplete `str.` to show available methods. Flat namespaces lose this discoverability.

5. **Architectural coherence:** This aligns with gert's design principle of **explicit boundaries**. Just as tools/providers are isolated via JSON-RPC, helper functions are isolated via namespaces.

**Why not the other options?**

- **Option 2/3 (bare functions):** Short-term simple, long-term messy. When we add `str_trim`, `str_replace`, `str_format`, `str_pad_left`, etc., the flat namespace becomes unwieldy. No natural grouping for documentation.

- **Option 1 alone:** This is actually Option 4 with only one namespace implemented. The recommendation is to commit to this as the **framework-wide pattern**, not just a string-specific fix.

### Implementation Plan

**Phase 1 (Immediate):**
1. Create `v2/internal/expr/helpers.go` with `StringHelpers` struct
2. Implement: `Contains`, `StartsWith`, `EndsWith`, `EqualsIgnoreCase`
3. Register in condition evaluator as `"str": &expr.StringHelpers{}`
4. Update condition evaluation tests to use `str.*` syntax

**Phase 2 (Next sprint):**
5. Document the namespace pattern in `docs/runbook-reference/conditions.md`
6. Add `DateHelpers` with `Now()`, `Add()`, `Format()`, `Parse()`
7. Add `JSONHelpers` with `Get()`, `Has()`, `Type()`

**Phase 3 (v2.1):**
8. Add `RegexHelpers` with `Match()`, `Replace()`, `Split()`
9. Add `MathHelpers` with `Round()`, `Abs()`, `Min()`, `Max()`
10. Consider `FileHelpers`, `NetHelpers` based on usage patterns

### Migration Notes

**For existing conditions (if any):**
- No breaking change if this is implemented before v2.0 release
- If runbooks exist with infix syntax (`x contains "y"`), provide a migration tool or linter warning

**Documentation:**
```markdown
## String Operations

Check if a string contains a substring:
condition: 'str.contains(dns_output, "Address")'

Check prefix or suffix:
condition: 'str.startsWith(path, "/etc")'
condition: 'str.endsWith(filename, ".yaml")'

Case-insensitive comparison:
condition: 'str.equalsIgnoreCase(method, "GET")'
```

---

## Summary

The `namespace.method()` pattern (Option 4) provides the best balance of **runbook authoring UX** and **long-term architectural coherence**. It avoids keyword collisions, reads clearly, and establishes a consistent framework for all future helper functions.

**Next steps:**
1. Confirm approach with Cristian (user acceptance of `str.contains()` syntax)
2. Assign implementation to Brian (expr helpers module)
3. Add to v2.0 spec as normative syntax for conditions
4. Update runbook examples in documentation

---

## Appendix: Rejected Alternatives

**Custom DSL:** Building a custom expression parser would give full control over syntax but violates the "no library switch" constraint and adds maintenance burden.

**Template syntax:** Using Go templates (`{{ if contains .dns_output "Address" }}`) breaks the expr-lang uniformity and loses type checking.

**Macro expansion:** Pre-processing conditions to replace `contains(x, y)` with `x contains y` is fragile (what about nested parens?) and hides the actual syntax from authors.

