# GERT Expression Language & Interpolation Syntax — Architecture Proposal

**Date:** 2026-06-04  
**Author:** Barbara — Lead / Architect  
**Status:** Proposal — awaiting team review and ormasoftchile approval  
**Supersedes:** `expr-lang/expr` boolean evaluation + Go `text/template` interpolation  
**References:** `design/gert/expression-evaluation-audit.md`, `.squad/decisions/inbox/copilot-directive-no-go-style-eval.md`, `.squad/decisions/inbox/copilot-direction-option-b.md`

---

## 1. GERT Expression Language (GXL) — Boolean Expressions

### 1.1 Purpose

GXL replaces `expr-lang/expr` for all boolean fields: `when`, `condition`, `until`, `iterate.until`, `branches[].condition`, `include.when`, and `collector.fields[].when`. It is a tiny, deterministic, side-effect-free language that any runtime (Go, C#, TypeScript, browser WASM) can implement from the grammar alone.

### 1.2 Grammar (EBNF)

```ebnf
Expression     = OrExpr ;
OrExpr         = AndExpr { "or" AndExpr } ;
AndExpr        = NotExpr { "and" NotExpr } ;
NotExpr        = "not" NotExpr | Comparison ;
Comparison     = Addition [ CompOp Addition ] ;
Addition       = Unary { ("+" | "-") Unary } ;
Unary          = [ "-" ] Primary ;
Primary        = Literal
               | Identifier
               | NamespaceCall
               | "(" Expression ")"
               ;

NamespaceCall  = Namespace "." Method "(" ArgList ")" ;
Namespace      = "str" | "list" | "len" | "regex" | "math" ;
Method         = IDENT ;
ArgList        = [ Expression { "," Expression } ] ;

Identifier     = IDENT ;
Literal        = STRING | NUMBER | BOOL | NULL ;

CompOp         = "==" | "!=" | "<" | ">" | "<=" | ">=" ;

(* Tokens *)
IDENT          = LETTER { LETTER | DIGIT | "_" } ;
STRING         = '"' { CHAR } '"' | "'" { CHAR } "'" ;
NUMBER         = DIGIT { DIGIT } [ "." DIGIT { DIGIT } ] ;
BOOL           = "true" | "false" ;
NULL           = "null" ;
LETTER         = "a".."z" | "A".."Z" | "_" ;
DIGIT          = "0".."9" ;
```

### 1.3 Token Set

| Category | Tokens |
|----------|--------|
| Identifiers | `/[a-zA-Z_][a-zA-Z0-9_]*/` — resolved from the flat run variable map |
| String literals | Double- or single-quoted. Escape sequences: `\\`, `\"`, `\'`, `\n`, `\t` |
| Number literals | Integer or decimal float. No hex, no octal, no scientific notation. |
| Boolean literals | `true`, `false` |
| Null literal | `null` |
| Comparison ops | `==`, `!=`, `<`, `>`, `<=`, `>=` |
| Logical ops | `and`, `or`, `not` |
| Arithmetic ops | `+`, `-` (for numeric comparisons; string concat NOT supported) |
| Grouping | `(`, `)` |
| Namespace call | `namespace.method(args...)` |

### 1.4 Spelling Choice: `and` / `or` / `not`

**Decision:** Use keyword logical operators `and`, `or`, `not`.

**Rationale:**
1. **Visual break from Go/C.** The whole point of GXL is to be demonstrably not a Go subset. `&&`/`||`/`!` invite the question "is this just expr with a new name?"
2. **YAML-friendliness.** `!` in YAML requires quoting (`'!str.contains(...)'`). `not` does not.
3. **Readability for non-dev operators.** GERT runbooks are authored by SREs, compliance officers, and business analysts — `and`/`or`/`not` are natural English.
4. **Precedent:** Python, SQL, Ansible conditions, Jinja2 all use keyword operators in similar contexts.

### 1.5 Type System

| Type | Description |
|------|-------------|
| `string` | UTF-8 text. All captured values default to string unless explicitly typed. |
| `number` | IEEE 754 float64. Comparisons use numeric semantics. |
| `bool` | `true` or `false`. |
| `null` | Represents an unset or missing variable. |
| `list` | Ordered sequence (only accessible via `list.*` helpers, not indexable in GXL). |

**Coercion rules: NONE.**

- `"5" == 5` → parse error ("cannot compare string to number")
- `null == ""` → `false` (null is its own type)
- `null == null` → `true`
- Using a `null` value in `<`/`>`/`<=`/`>=` → evaluation error
- `not null` → evaluation error (not is only defined on bool)
- Any comparison where either side is `null` returns `false` EXCEPT `== null` and `!= null`

**No implicit truthiness.** A bare identifier is valid in boolean position ONLY if the variable is of type `bool`. `when: acknowledged` is valid if `acknowledged` is `bool`; `when: hostname` (a string) is a type error caught at parse-time if possible, or evaluation-time otherwise.

### 1.6 Allowlisted Helper Namespaces

All function calls MUST go through a namespace. Bare function calls are forbidden except `len()` which is a top-level builtin.

#### `len` (top-level)

| Signature | Return | Description |
|-----------|--------|-------------|
| `len(s: string)` | `number` | Character count |
| `len(l: list)` | `number` | Element count |

#### `str` namespace

| Function | Signature | Return | Description |
|----------|-----------|--------|-------------|
| `str.contains` | `str.contains(s: string, substr: string)` | `bool` | True if s contains substr |
| `str.startsWith` | `str.startsWith(s: string, prefix: string)` | `bool` | True if s begins with prefix |
| `str.endsWith` | `str.endsWith(s: string, suffix: string)` | `bool` | True if s ends with suffix |
| `str.toLower` | `str.toLower(s: string)` | `string` | Lowercase |
| `str.toUpper` | `str.toUpper(s: string)` | `string` | Uppercase |
| `str.trim` | `str.trim(s: string)` | `string` | Strip leading/trailing whitespace |
| `str.trimPrefix` | `str.trimPrefix(s: string, prefix: string)` | `string` | Remove prefix if present |
| `str.trimSuffix` | `str.trimSuffix(s: string, suffix: string)` | `string` | Remove suffix if present |

#### `list` namespace

| Function | Signature | Return | Description |
|----------|-----------|--------|-------------|
| `list.contains` | `list.contains(l: list, item: string\|number\|bool)` | `bool` | True if list contains item |

#### `regex` namespace

| Function | Signature | Return | Description |
|----------|-----------|--------|-------------|
| `regex.match` | `regex.match(s: string, pattern: string)` | `bool` | True if s matches RE2 pattern |

**Note:** Pattern MUST be RE2 syntax (already mandated by GERT0002 Roslyn analyzer). No PCRE, no backreferences.

#### `math` namespace (deferred — v2)

Reserved. Not included in initial release. Numeric comparison operators (`<`, `>`, etc.) cover the immediate needs.

### 1.7 Error Model

| Category | Trigger | Behavior |
|----------|---------|----------|
| **Parse error** | Syntax violation, unknown namespace, unknown method, disallowed construct | Rejected at load/plan time. Runbook does not enter execution. Error message includes line/col and offending token. |
| **Type error** | Comparing incompatible types, using non-bool in boolean position, null in ordered comparison | Rejected at plan-time where inferrable; otherwise evaluation error at runtime. |
| **Evaluation error** | Variable is null when null is not permitted in that position, regex pattern invalid | Hard step failure. Step transitions to `failed`. Execution halts unless `continue_on_fail: true`. |
| **Runtime error** | Should not exist — all errors are parse or evaluation. No I/O, no side effects, no dynamic dispatch. | N/A |

### 1.8 Explicitly Forbidden

- ❌ Function calls outside the allowlist
- ❌ Member access on identifiers (`foo.bar` — use capture paths instead)
- ❌ Dynamic dispatch / reflection
- ❌ Pipe operators (`|`)
- ❌ Infix `contains`, `startsWith`, `matches` (must use `str.*` namespace)
- ❌ Assignment (`=`, `:=`, `let`)
- ❌ String concatenation in expressions (`+` is numeric only)
- ❌ Array indexing (`items[0]`)
- ❌ Object/map construction (`{key: value}`)
- ❌ Ternary operator (`? :`)
- ❌ `in` operator
- ❌ Any construct requiring host-language semantics

---

## 2. GERT Interpolation Syntax (GIS) — String Templates

### 2.1 Purpose

GIS replaces Go `text/template` for all string fields: `title`, `args`, `stdin`, tool `argv[]`, `display.content`, assertion `subject`/`expected`, artifact paths, `event.id`, and any other string field that references run variables.

### 2.2 Delimiter Choice: `${...}`

**Decision:** Use `${expression}` as the interpolation delimiter.

**Rationale:**
1. **Does NOT collide with Go templates.** `{{ }}` would invite confusion about whether the grammar inside is Go template or GIS. `${ }` is visually distinct.
2. **Familiar.** Shell, ES6 template literals, Terraform, GitHub Actions, and Docker Compose all use `${...}`.
3. **YAML-safe.** `${foo}` does not require quoting in YAML string values (unlike `{{ }}` which some YAML processors treat specially).
4. **Single-char escape:** `$${` produces literal `${` (doubling the `$`).
5. **No conflict with environment variable references** in tool definitions, which already use `${ENV_VAR}` — but those are in a different evaluation phase (process environment, resolved by the OS, not by GIS). GIS runs on authored runbook fields; env-var substitution happens in tool process spawning. The phases are disjoint.

### 2.3 Path Grammar: GERT Dotted Path (GDP)

**Decision:** Use a GERT-native dotted path syntax. Not JSON Pointer, not JMESPath.

**Rationale:**
- JSON Pointer (`/foo/bar/0`) is unfamiliar to YAML authors and ugly inside `${}`.
- JMESPath is too powerful (filters, projections, multi-select) — we'd have to subset it anyway and that creates "which subset?" drift across runtimes.
- A dotted path is what authors already write in fixtures (`json.items[0].metadata.name`). Formalize what exists.

**Grammar:**

```ebnf
GISExpression  = Path ;
Path           = Segment { "." Segment } ;
Segment        = IDENT [ Index ] ;
Index          = "[" INTEGER "]" ;
IDENT          = LETTER { LETTER | DIGIT | "_" } ;
INTEGER        = DIGIT { DIGIT } ;
```

**Examples:**
- `${hostname}` — top-level variable
- `${service_config.region}` — nested path into a structured capture
- `${items[0].name}` — array index access
- `${build_id}` — simple scalar

### 2.4 What's Allowed Inside `${...}`

**Decision:** Identifier path ONLY. No GXL expressions, no ternary, no function calls.

**Rationale (smallest viable surface):**
- Every interpolation site in Don's audit needs variable substitution only — no boolean logic.
- If authors need conditional text, they should use a `branch` step or `display` step with `when` guards.
- Keeping GIS to path-only makes it trivially implementable (string split on `.`, walk a nested map, index arrays). No parser combinator library needed.
- This prevents scope creep: if we allow `${x == "y" ? "a" : "b"}`, we've recreated templates.

### 2.5 Missing-Key Behavior

**Decision:** Hard error.

**Rationale:**
- GERT's value proposition is traceability and determinism. Silent empty-string substitution hides bugs and makes audit trails unreliable.
- The planner can detect most missing references at plan-time (static analysis of variable scope vs. interpolation references — already described in `03-schema-vnext.tex:3530-3547`).
- If a variable legitimately might not exist, the author should use `when:` to guard the step, or provide a default in `vars:`.
- Configurable behavior ("maybe error, maybe empty") creates cross-runtime divergence — exactly what we're eliminating.

### 2.6 Escape Syntax

| Input | Output |
|-------|--------|
| `$${` | Literal `${` |
| `$$` not followed by `{` | Literal `$$` |

### 2.7 Explicit Non-Goals

- ❌ Pipelines (`${value \| toLower}`)
- ❌ Custom function maps
- ❌ `fromJSON` / `fromYAML` inline parsing — authors must use typed captures or structured input fields
- ❌ Conditional/ternary expressions inside `${}`
- ❌ Filters or projections
- ❌ Default/fallback syntax (`${var:-default}`)
- ❌ Arithmetic inside interpolation

### 2.8 Structured Data Access

Structured data must be declared at capture time so the runtime stores a typed value in the variable map. Interpolation then uses dotted paths, for example `${service_config.region}`.

This keeps parsing at the point of capture, not inline during rendering. The runbook is more auditable and the variable map is self-describing.

---

## 3. Coverage Matrix

### 3.1 Evaluation Site Mapping

| # | Evaluation Site | Current | Replacement | Notes |
|---|----------------|---------|-------------|-------|
| 1 | General condition language (`when`, `condition`, `until`) | `expr-lang/expr` | **GXL** | Direct replacement. `&&`→`and`, `\|\|`→`or`, `!`→`not` |
| 2 | Common step guard (`step.when`) | `expr-lang/expr` | **GXL** | Same as #1 |
| 3 | Branch routing (`branches[].condition`) | Conflicted (spec says Go template, examples use expr) | **GXL** | Resolve spec conflict: it's GXL, period |
| 4 | Iterate convergence (`iterate.until`) | `expr-lang/expr` | **GXL** | Same as #1 |
| 5 | Iterate list source (`iterate.over`) | Mixed (JSONPath `$.services`, Go template) | **GIS path expression** | `iterate.over: ${services}` or a bare identifier `services` referencing a list variable |
| 6 | Include control (`include.when`, `include.with`) | Mixed expr/template | **GXL** for `include.when`; **GIS** for `include.with` values | Clean split: boolean → GXL, string → GIS |
| 7 | Collector dynamic fields (`fields[].when`) | Conflicted (table says Go template, text says expr) | **GXL** | Critical: browser must evaluate same grammar as backend |
| 8 | CLI step interpolation (`title`, `args`, `stdin`) | Go `text/template` | **GIS** | `{{ .hostname }}` → `${hostname}` |
| 9 | Tool step argument interpolation (`tool.args`) | Go `text/template` | **GIS** | Same as #8 |
| 10 | Native/stdio tool argv rendering | Go `text/template` | **GIS** | `{{ .arg_name }}` → `${arg_name}` |
| 11 | Assertion operands (`subject`, `expected`) | Go template strings | **GIS** | Assertion operators remain structured |
| 12 | Display step content (`display.content`) | Go `text/template` | **GIS** | Simple variable substitution |
| 13 | Capture paths | Under-specified mini-language | **GERT Capture Path (GCP)** — see §3.3 | Formalized as its own grammar |
| 14 | Wait-for-event filters (`event.filter`) | Literal key-value map | **Keep as-is (structured predicate)** | Already portable, no expression needed |
| 15 | Provider input binding (`input.from`) | Provider prefix dispatch | **Keep as-is (structured)** | Not an expression surface |

### 3.2 Sites Where Structured Predicates Are Preferred Over GXL/GIS

| Site | Recommendation |
|------|---------------|
| `event.filter` (#14) | Keep as structured predicate map. Already portable. |
| `input.from` (#15) | Keep as provider-prefix dispatch. Not an expression. |
| Assertion operators (#11) | Keep structured assertion types (`contains`, `eq`, `matches`, etc.). Only the *operand values* (`subject`, `expected`) become GIS. The assertion type itself remains a structured enum. |
| `iterate.over` (#5) | Consider a bare identifier (variable name referencing a list) rather than `${...}` wrapping. This avoids the question of "is this an expression?" — it's just a variable reference. |

### 3.3 GERT Capture Path (GCP) — Portable Grammar

The capture path surface (`json.items[0].metadata.name`, `json.items | length`, `stdout.incident.id`) needs its own small grammar:

```ebnf
CapturePath    = Source "." Path
               | Source
               ;
Source         = "stdout" | "stderr" | "exitCode" | "json" | "yaml" ;
Path           = Segment { "." Segment } ;
Segment        = IDENT [ Index ] ;
Index          = "[" INTEGER "]" ;
```

**Changes from current state:**
- **Remove pipe syntax.** `json.items | length` becomes a post-capture GXL expression: capture into a variable, then `len(captured_items)` in a condition. This eliminates an entire evaluation surface.
- **`json` and `yaml` sources** imply the runtime parses stdout/stderr as JSON/YAML before path traversal.
- **`exitCode`** returns a number directly (no path traversal).

**Examples:**
```yaml
capture:
  pod_status: json.status.phase
  first_item: json.items[0].metadata.name
  raw_output: stdout
  return_code: exitCode
```

The `| length` pattern is retired. Authors who need the length of a captured list use:
```yaml
capture:
  all_items: json.items    # captures the list
# Then in a condition:
when: len(all_items) > 0
```

---

## 4. Parse-Time Enforcement Plan

### 4.1 Validation Architecture

```
┌─────────────────────────────────────────────────┐
│  YAML Load (schema.yaml / tool.yaml)            │
│  ↓                                              │
│  Schema Validation (JSON Schema / structural)   │
│  ↓                                              │
│  ┌───────────────────────────────────────────┐  │
│  │  GXL Parser — validates all boolean fields │  │
│  │  • Tokenize → Parse → Type-check          │  │
│  │  • Rejects unknown namespaces/methods      │  │
│  │  • Rejects forbidden constructs            │  │
│  └───────────────────────────────────────────┘  │
│  ↓                                              │
│  ┌───────────────────────────────────────────┐  │
│  │  GIS Parser — validates all string fields  │  │
│  │  • Finds ${...} tokens                     │  │
│  │  • Validates path grammar                  │  │
│  │  • Records referenced variables            │  │
│  └───────────────────────────────────────────┘  │
│  ↓                                              │
│  ┌───────────────────────────────────────────┐  │
│  │  GCP Parser — validates all capture paths  │  │
│  │  • Validates source prefix                 │  │
│  │  • Validates path grammar                  │  │
│  └───────────────────────────────────────────┘  │
│  ↓                                              │
│  ┌───────────────────────────────────────────┐  │
│  │  Semantic Validator (Planner)              │  │
│  │  • Variable scope analysis [SEM-007]       │  │
│  │  • Unreachable-variable detection          │  │
│  │  • Type inference where possible           │  │
│  └───────────────────────────────────────────┘  │
│  ↓                                              │
│  Plan ready → RunHandle.Next() may proceed      │
└─────────────────────────────────────────────────┘
```

### 4.2 Where in the Runtime/Loader

| Phase | Component | What it validates |
|-------|-----------|-------------------|
| Load | `RunbookLoader` / `ToolLoader` | YAML structure, schema conformance |
| Parse | `GXLParser`, `GISParser`, `GCPParser` | Syntax of every expression/interpolation/path field |
| Plan | `Planner` / `SemanticValidator` | Variable scope, type compatibility, reachability |
| Gate | `RunHandle.Next()` pre-check | Asserts plan is valid before any step executes (defense-in-depth) |

**Key rule:** Nothing reaches `RunHandle.Next()` without passing all three parsers AND the semantic validator. The planner produces a `ValidatedPlan` — the runtime accepts ONLY `ValidatedPlan`, never raw parsed YAML.

### 4.3 Error UX for Runbook Authors

```
ERROR [GXL-001] schema.yaml:152 — Parse error in field 'condition'
  Expression: pod_json contains "CrashLoopBackOff"
                       ^^^^^^^^
  Infix 'contains' is not supported. Use: str.contains(pod_json, "CrashLoopBackOff")

ERROR [GIS-002] schema.yaml:66 — Invalid interpolation in field 'title'
  Value: Check pod {{ .pod_name }} in namespace {{ .namespace }}
         ^^^^^^^^^^
  Go template syntax '{{ }}' is not supported. Use: Check pod ${pod_name} in namespace ${namespace}

ERROR [GXL-003] schema.yaml:226 — Parse error in field 'condition'
  Expression: error_rate > 0.05 || p95_latency > 0.5
                                ^^
  Operator '||' is not supported. Use 'or' instead.

ERROR [SEM-007] schema.yaml:88 — Variable reference error
  Field 'args[0]' references variable 'build_id' which is not in scope.
  Variable 'build_id' is captured at step 'run_build' (line 72) which is not
  a guaranteed predecessor of this step.
```

### 4.4 Interaction with Planner / RunHandle.Next

1. `RunbookLoader.Load()` calls all three parsers. If ANY fails, `Load()` returns errors — no plan is produced.
2. `Planner.Plan()` takes a parsed runbook and performs semantic analysis. If it fails, no `ValidatedPlan` is produced.
3. `RunHandle` constructor requires a `ValidatedPlan`. The type system prevents unvalidated execution.
4. `RunHandle.Next()` does NOT re-validate — it trusts the plan. This keeps hot-path execution fast.
5. Defense-in-depth: if a runtime implementor somehow bypasses the type constraint, each evaluator (`GXLEvaluator`, `GISEvaluator`) will fail-fast on invalid syntax at evaluation time rather than producing undefined behavior.

---

## 5. Conformance Test Corpus

For a non-Go runtime (C#, TypeScript) to claim GXL/GIS parity, it MUST pass:

#### GXL Conformance Tests

| Category | Test Cases |
|----------|-----------|
| Literals | `true`, `false`, `null`, `"hello"`, `'hello'`, `42`, `3.14`, `0` |
| Identifiers | Simple variable lookup, missing variable → error |
| Comparison | `==`, `!=` for all type pairs; `<`, `>`, `<=`, `>=` for numbers; cross-type → error |
| Logical | `and`, `or`, `not`; short-circuit behavior; precedence (`not` > `and` > `or`) |
| Null handling | `x == null`, `x != null`, `null == null`, `null < 1` → error |
| Grouping | `(a or b) and c` vs `a or b and c` |
| Namespace calls | All `str.*` functions with edge cases (empty string, Unicode) |
| `len()` | String length, list length, null → error |
| `regex.match` | RE2 patterns, invalid pattern → evaluation error |
| `list.contains` | Present/absent items, type mismatch → error |
| Truthiness rejection | Bare string identifier in boolean position → error |
| Forbidden syntax | `&&` → parse error; `\|\|` → parse error; `!x` → parse error; `x contains y` → parse error; `x.y` → parse error; `f()` unknown → parse error |

#### GIS Conformance Tests

| Category | Test Cases |
|----------|-----------|
| Simple substitution | `${var}` resolves from variable map |
| Nested path | `${a.b.c}`, `${a[0].b}` |
| Missing key | `${nonexistent}` → hard error |
| Escape | `$${literal}` → `${literal}` |
| Multiple interpolations | `Hello ${first} ${last}` |
| No expression | `${a == b}` → parse error |
| Adjacent text | `prefix${x}suffix` |
| Empty string variable | `${""}` → parse error (not a path) |

#### GCP Conformance Tests

| Category | Test Cases |
|----------|-----------|
| Simple sources | `stdout`, `stderr`, `exitCode` |
| JSON path | `json.field`, `json.nested.field`, `json.items[0]` |
| Invalid source | `xml.field` → parse error |
| Invalid index | `json.items[-1]` → parse error |
| Pipe rejection | `json.items \| length` → parse error |

**Reference:** `design/web-platform/c-sharp-governance-parity.md` already defines template parity as a gate (lines 291–336). The GXL/GIS conformance corpus replaces and supersedes those template parity vectors.

---

## 6. Open Questions for Team / ormasoftchile

1. **`iterate.over` syntax:** Should it be a bare variable name (`over: services`) or a GIS expression (`over: ${services}`)? Bare name is simpler but introduces a third micro-syntax. GIS expression is consistent but feels heavy for what's always a single variable reference.

2. **Default values for variables:** With `fromJSON`/`fromYAML` removed and missing-key as hard error, how do authors express "use this value if the variable wasn't captured"? Options:
   - (a) `vars:` section with defaults (already exists)
   - (b) A `default:` field on `capture` declarations
   - (c) Both
   
3. **`str.*` naming convention:** Should we keep camelCase (`startsWith`, `endsWith`, `toLower`) or switch to snake_case (`starts_with`, `ends_with`, `to_lower`)? camelCase matches the existing spec; snake_case is more YAML-natural. This is a one-time decision that locks forever.

4. **Assertion type `contains`:** The assertion operator `type: contains` (used in r14, r17, r18) is a structured predicate, not an infix expression. It stays as-is. Confirm this understanding is correct — it's not the same as the forbidden `x contains "y"` infix in GXL.

5. **Structured captures for JSON/YAML:** When a step captures `json.config.region`, does the runtime store just the scalar value at that path, or should there be a way to capture an entire subtree as a structured value in the variable map? The latter is needed for `iterate.over` on a list. Proposal: capture paths that resolve to an object/array store the full subtree.

6. **Timeline:** Does this block on all other work (spec rewrites, C# runtime, web platform) or can it proceed in parallel with A6 implementation? My recommendation: spec rewrite is a prerequisite for C# runtime implementation; A6 infra work can proceed in parallel since it doesn't touch expression evaluation.

---

## Appendix A: Precedence Table

| Precedence (highest first) | Operator |
|----------------------------|----------|
| 1 | `not`, unary `-` |
| 2 | `*`, `/` (reserved, not in v1) |
| 3 | `+`, `-` |
| 4 | `<`, `>`, `<=`, `>=` |
| 5 | `==`, `!=` |
| 6 | `and` |
| 7 | `or` |

---

*End of proposal. Awaiting review from Don (portability audit), David (integration impact), and ormasoftchile (open questions).*
