# GIS Optional-Chaining Proposal

**Author:** Barbara — Lead / Architect  
**Date:** 2026-06-05T07:28:56-07:00  
**Status:** PROPOSAL — pending review by ormasoftchile  
**Scope:** GIS interpolation paths only (not GXL, not GCP)  
**Triggered by:** ormasoftchile request for explicit opt-in tolerance on missing path segments

---

## 0. Motivation

GIS currently mandates **hard-error** (`GIS-PATH-MISSING`) on any unresolvable path segment inside `${...}` (gis.ebnf §4.3). This is correct as a default — it prevents silent data loss in audit-critical templates.

However, real-world runbooks encounter **legitimately optional** data:

- API responses where a field may or may not be present depending on account tier
- Configuration objects where an optional section is omitted
- Captured subtrees where an upstream step conditionally populates a nested key

Authors currently have no GIS-native way to express "I know this might be missing; give me the empty string." They must pre-populate via `vars:` or rely on `when:` guards at the step level — both of which are heavyweight for simple "render if present, skip if not" cases.

This proposal adds **optional-chaining** (`?.`) as an explicit opt-in escape hatch. The hard-error default is **unchanged** for plain `${...}` paths.

---

## 1. Grammar Extension (EBNF Delta)

### 1.1 New Productions (additive to gxl.ebnf §3, GDP)

```ebnf
(* Current GDP rule in gxl.ebnf §3: *)
GDP = IDENT { DOT IDENT | LBRACKET INTEGER RBRACKET } ;

(* Extended GDP rule: *)
GDP = IDENT { PathSegment } ;

PathSegment = DotAccess
            | OptionalDotAccess
            | BracketAccess
            | OptionalBracketAccess
            ;

DotAccess            = '.' IDENT ;
OptionalDotAccess    = '?.' IDENT ;
BracketAccess        = '[' INTEGER ']' ;
OptionalBracketAccess = '?.[' INTEGER ']' ;
            (* Note: '?.[' is a THREE-character token, not '?.' + '['.
               This matches JS/TS optional bracket access: obj?.[0] *)
```

### 1.2 Tokenization Rule

The lexer MUST recognize `?.` as a **single two-character token** (`OPTIONAL_DOT`) distinct from `?` followed by `.`:

- **Lookahead:** When the lexer encounters `?`, it checks the next character:
  - If next is `.` → emit `OPTIONAL_DOT` token, consume both characters.
  - If next is `[` → emit `OPTIONAL_BRACKET` token (`?.[`), but wait — see below.
  - Otherwise → emit `?` as itself (currently unused in GXL GDP; would be a parse error).

For `?.[`:
- When the lexer encounters `?`, checks next char is `.`, checks char after that is `[`:
  - If `?.` followed by `[` → emit `OPTIONAL_BRACKET_OPEN` token (three chars `?.[`), then continue to parse INTEGER and `]` as normal bracket contents.
  - If `?.` NOT followed by `[` → emit `OPTIONAL_DOT` and continue with IDENT.

This is a **longest-match** rule. No ambiguity arises because `?` has no standalone meaning in GDP paths today.

### 1.3 Precedence and Associativity

- `?.` and `.` have **identical precedence** (they are both path-segment separators).
- Path segments are evaluated **left-to-right** (left-associative, as today).
- Mixed chains are legal: `a.b?.c.d` means:
  - `a` — mandatory (hard error if missing)
  - `.b` — mandatory (hard error if `a` lacks `b`)
  - `?.c` — optional (if `a.b` lacks `c`, short-circuit to `""`)
  - `.d` — part of the optional tail (NOT evaluated if `c` was missing)

**Key rule:** Once any `?.`-guarded segment triggers a miss, the **entire remaining chain** returns `""`. There is no "resumption" after a miss. This matches JS/TS semantics exactly.

### 1.4 `?.[N]` Optional Array Indexing — INCLUDED

**Recommendation: IN SCOPE.** Rationale:

- Without `?.[N]`, an author writing `${items?.name}` gets tolerance on the field miss, but `${items[0]?.name}` would hard-error on the index. Inconsistent.
- `${items?.[0]?.name}` is the idiomatic JS pattern; authors don't need new mental models.
- The implementation cost is minimal (one additional token type, same short-circuit logic).

---

## 2. Semantics Specification

### 2.1 Short-Circuit Rule

When evaluating a GDP path left-to-right, if any segment marked with `?.` (or `?.[N]`) encounters a miss condition (defined below), evaluation **immediately stops** and the entire GDP expression resolves to the **empty string `""`**.

No subsequent segments are evaluated. No side effects occur past the miss point (GXL has no side effects today, but this is stated for forward-compatibility).

This matches [TC39 Optional Chaining](https://tc39.es/proposal-optional-chaining/) semantics:
> "If the operand at the left-hand side of the ?. operator evaluates to undefined or null, the expression evaluates to undefined instead of throwing an error."

In our case, "evaluates to undefined" → resolves to `""` (the locked decision for GIS string output).

### 2.2 What Counts as "Missing" (Triggers Short-Circuit)

| Condition | `?.` behavior | Plain `.` behavior |
|-----------|--------------|-------------------|
| Identifier not in context map | Short-circuit → `""` | GXL-PATH-001 (hard error) |
| Value is `null` | Short-circuit → `""` | GXL-PATH-001 on next segment (dot-access on null) |
| Object lacks the next key | Short-circuit → `""` | GXL-PATH-001 (hard error) |
| Array index out of bounds (`?.[N]`) | Short-circuit → `""` | GXL-PATH-002 (hard error) |
| Index on non-array value (`?.[N]` on a string/object/number) | Short-circuit → `""` | GXL-PATH-003 (hard error) |

### 2.3 What Does NOT Trigger the Default

These are **present values** — they resolve normally even through `?.`:

| Condition | Result |
|-----------|--------|
| Value is empty string `""` | Render `""` (the value IS empty string; this is not a miss) |
| Value is `false` | Render `"false"` per PJVM coercion |
| Value is `0` | Render `"0"` per PJVM coercion |
| Value is empty array `[]` | Render `"[]"` per PJVM coercion |
| Value is empty object `{}` | Render `"{}"` per PJVM coercion |

**Critical distinction:** `null` triggers the default; empty string does NOT. This matches JS where `null?.foo` → `undefined` but `""?.length` → `0` (non-nullish).

### 2.4 Mixed Paths — Detailed Examples

```
Expression: ${a.b?.c.d}
Context:    { a: { b: { c: { d: "found" } } } }
Result:     "found"

Expression: ${a.b?.c.d}
Context:    { a: { b: {} } }             # b exists but lacks c
Result:     ""                            # ?.c misses → short-circuit, .d not evaluated

Expression: ${a.b?.c.d}
Context:    { a: { b: null } }            # b is null
Result:     ""                            # ?.c on null → short-circuit

Expression: ${a.b?.c.d}
Context:    { a: {} }                     # a exists, but lacks b
Result:     GIS-PATH-MISSING (hard error) # .b is NOT optional — plain dot

Expression: ${a.b?.c.d}
Context:    {}                            # a missing entirely
Result:     GIS-PATH-MISSING (hard error) # 'a' is the root, always mandatory
```

### 2.5 No Nullish-Coalescing (`??`)

The `??` operator (nullish-coalescing) is **NOT in scope** for this proposal. Authors who want a non-empty default when a path is absent have two existing mechanisms:

1. `vars:` pre-population in the runbook step definition
2. `capture.default:` in GCP capture expressions (per locked Q2)

If real usage demonstrates that `${user?.name ?? "Anonymous"}` is needed, it can be proposed separately. The grammar reserves `??` for this purpose (it is not a valid GXL token today).

---

## 3. Scope Clarification

### 3.1 GIS Only

This extension applies **exclusively** to GIS interpolation `${...}` paths. It does NOT extend to:

- **GXL boolean expressions** (in `when:`, `assert:`, `loop.while:`). In GXL boolean context, `user?.email == "x"` creates a type-theoretic question: what does `"" == "x"` mean? Is it `false`? Is the comparison skipped? These semantics are non-trivial and out of scope. Revisit only if real usage demands.

- **GCP capture paths.** Capture paths already have `capture.default:` (per locked Q2 from the GXL/GIS/GCP ratification session). That mechanism solves the same problem at the capture layer. Don't duplicate.

### 3.2 Interaction with GXL Full Expressions

Inside `${...}`, authors may write full GXL expressions (e.g., `${str.toLower(user?.name)}`). Optional chaining applies to the **GDP path resolution** step only. If `user?.name` resolves to `""` due to a miss, the stdlib call receives `""` as its argument — which is valid.

However: `${user?.name + " suffix"}` → if `user` is missing, the GDP resolves to `""`, then `"" + " suffix"` → `" suffix"`. Authors should be aware that optional chains compose with string concatenation. This is expected behavior, not a bug.

---

## 4. Conformance Corpus Additions

New file: `design/gert/conformance/tv-gis-path.yaml`

(GIS-PATH is a new conformance category — the existing `tv-gxl-path.yaml` tests GXL GDP resolution at the expression level; GIS-PATH tests interpolation-level behavior including the optional-chaining extension and `GIS-PATH-MISSING` error mapping.)

### 4.1 Proposed Test Vectors

```yaml
# tv-gis-path.yaml (proposed — do NOT create until ratified)
# Category:    GIS-PATH
# Grammar ref: design/gert/grammar/gis.ebnf §4.3 + optional-chaining extension
# Author:      Tess (Conformance Tester) — vectors proposed by Barbara
# Date:        2026-06-05

vectors:

  # --- Simple optional miss ---
  - id: TV-GIS-PATH-001
    category: GIS-PATH
    description: "Simple optional path on missing top-level variable resolves to empty string."
    template: "Hello ${user?.name}"
    variables: {}
    expected:
      value: "Hello "
    tags: [optional-chaining, simple-miss, top-level]

  # --- Deep optional miss ---
  - id: TV-GIS-PATH-002
    category: GIS-PATH
    description: "Deep optional chain — all segments optional, root missing → empty string."
    template: "${a?.b?.c?.d}"
    variables: {}
    expected:
      value: ""
    tags: [optional-chaining, deep-miss]

  # --- Short-circuit: miss in middle stops evaluation ---
  - id: TV-GIS-PATH-003
    category: GIS-PATH
    description: "Short-circuit — once ?.b misses, remaining .c.d are not evaluated."
    template: "${a?.b.c.d}"
    variables: {a: {}}
    expected:
      value: ""
    tags: [optional-chaining, short-circuit]

  # --- Mixed . and ?. — mandatory prefix hard-errors ---
  - id: TV-GIS-PATH-004
    category: GIS-PATH
    description: "Mixed path — mandatory .b before ?.c; b missing → hard error."
    template: "${a.b?.c}"
    variables: {a: {}}
    expected:
      error_class: GIS-PATH
      error_code: GIS-PATH-MISSING
    tags: [optional-chaining, mixed, hard-error]

  # --- Present-but-null triggers optional default ---
  - id: TV-GIS-PATH-005
    category: GIS-PATH
    description: "Value is null — ?.field on null short-circuits to empty string."
    template: "${config?.timeout}"
    variables: {config: null}
    expected:
      value: ""
    tags: [optional-chaining, null-value]

  # --- Present-but-empty-string does NOT trigger default ---
  - id: TV-GIS-PATH-006
    category: GIS-PATH
    description: "Value is empty string — this IS the resolved value, not a miss."
    template: "[${user?.name}]"
    variables: {user: {name: ""}}
    expected:
      value: "[]"
    note: "Empty string is a legitimate value. Optional chaining does not treat it as missing."
    tags: [optional-chaining, empty-string, no-trigger]

  # --- Present-but-false does NOT trigger default ---
  - id: TV-GIS-PATH-007
    category: GIS-PATH
    description: "Value is false — renders normally through optional chain."
    template: "${settings?.verbose}"
    variables: {settings: {verbose: false}}
    expected:
      value: "false"
    tags: [optional-chaining, false-value, no-trigger]

  # --- Optional bracket indexing: out-of-bounds ---
  - id: TV-GIS-PATH-008
    category: GIS-PATH
    description: "Optional bracket ?.[5] on a 3-element array → empty string."
    template: "${items?.[5]}"
    variables: {items: [10, 20, 30]}
    expected:
      value: ""
    tags: [optional-chaining, bracket, out-of-bounds]

  # --- Optional bracket indexing: success ---
  - id: TV-GIS-PATH-009
    category: GIS-PATH
    description: "Optional bracket ?.[0] on a populated array → resolves element."
    template: "${items?.[0]}"
    variables: {items: [10, 20, 30]}
    expected:
      value: "10"
    tags: [optional-chaining, bracket, success]

  # --- Combined: ?.[N] then ?.field ---
  - id: TV-GIS-PATH-010
    category: GIS-PATH
    description: "Combined ?.[0]?.name — array access then field access, both optional."
    template: "${results?.[0]?.name}"
    variables: {results: [{name: "Alice"}]}
    expected:
      value: "Alice"
    tags: [optional-chaining, bracket, combined, success]

  # --- Combined: ?.[N] miss then ?.field not evaluated ---
  - id: TV-GIS-PATH-011
    category: GIS-PATH
    description: "Combined ?.[0]?.name — empty array, bracket misses, chain short-circuits."
    template: "${results?.[0]?.name}"
    variables: {results: []}
    expected:
      value: ""
    tags: [optional-chaining, bracket, combined, short-circuit]

  # --- Root variable is always mandatory ---
  - id: TV-GIS-PATH-012
    category: GIS-PATH
    description: "Root variable without ?. is always mandatory — even if later segments are optional."
    template: "${missing?.field}"
    variables: {}
    expected:
      value: ""
    note: "The ?. on 'field' makes the root→field hop optional. Since root itself triggers the miss at the ?. boundary, this short-circuits to empty string."
    tags: [optional-chaining, root-miss]

  # --- Mandatory root + missing root = hard error ---
  - id: TV-GIS-PATH-013
    category: GIS-PATH
    description: "Plain ${missing.field} — no optional chaining — remains hard error."
    template: "${missing.field}"
    variables: {}
    expected:
      error_class: GIS-PATH
      error_code: GIS-PATH-MISSING
    tags: [no-optional, hard-error, regression]
```

### 4.2 Numbering Scheme

- Category: `GIS-PATH`
- IDs: `TV-GIS-PATH-001` through `TV-GIS-PATH-NNN`
- Separate from `TV-GXL-PATH-*` (which tests expression-level GDP resolution without interpolation context)

---

## 5. Idiomatic Examples

### 5.1 Optional API Response Fields

```yaml
# API returns user profile; "company" field is only present for enterprise accounts
steps:
  - tool: http.get
    args: "${api_base}/users/${user_id}"
    capture:
      http.body.profile: user_profile
  - display:
      content: |
        Name: ${user_profile.name}
        Email: ${user_profile.email}
        Company: ${user_profile?.company}
```

If the user is on a free account and `company` is absent, the third line renders as `Company: ` (empty) rather than failing the entire runbook.

### 5.2 Optional Configuration Sections

```yaml
# Runbook supports optional monitoring config
vars:
  service_name: "payments"
steps:
  - tool: deploy
    args: "--name ${service_name} --monitoring-endpoint ${config?.monitoring?.endpoint}"
```

If monitoring config isn't provided, the flag renders with an empty value. The tool handles the empty arg gracefully (tool's responsibility, not GERT's).

### 5.3 Optional Captured Subtree Members

```yaml
steps:
  - tool: api.call
    capture:
      http.body: response
  - display:
      content: "Request ID: ${response?.metadata?.request_id}"
```

Some API responses include a `metadata` block; others don't. The optional chain handles both gracefully.

### 5.4 Optional Array Element Access

```yaml
steps:
  - tool: search
    capture:
      http.body.results: search_results
  - display:
      content: "Top result: ${search_results?.[0]?.title}"
```

If the search returns zero results, renders as `Top result: ` rather than failing.

### 5.5 Anti-Pattern: DON'T Use `?.` on Required Fields

```yaml
# ❌ BAD: Hiding bugs with optional chaining
steps:
  - display:
      content: "Order total: ${order?.total}"
      # If 'order' should ALWAYS be present at this point in the runbook,
      # using ?. silently hides a logic error (missing capture, wrong variable name).
      # Use plain ${order.total} — the hard error tells you something is wrong.
```

**Guidance:** Use `?.` only when the absence is **expected and tolerable**. If a path "should" always exist, leave it as plain `.` so the hard error acts as your safety net.

---

## 6. Parse-Time Enforcement Implications

The parse-time enforcement plan (§03d) validates that every GXL expression inside `${...}` is syntactically valid at plan time. With optional chaining:

- **Parser validates syntax:** `?.` and `?.[N]` are syntactically legal tokens. The parser can confirm the expression is well-formed.
- **Parser CANNOT validate necessity:** Without a schema describing which variables/fields are guaranteed to exist, the parser cannot warn that `?.` is used on a path that will "always" resolve. This is a purely **runtime** contract.
- **No change to the parse gate contract:** The parse gate still rejects malformed expressions. It does not (and cannot) reject semantically questionable uses of `?.`.

**Future (non-blocking):** If/when GERT gains a variable schema (declaring which variables exist and their types at each step), the parser could emit `GIS-WARN-001: optional chaining on known-required path` as a **warning** (not error). This is deferred and does not block this proposal.

---

## 7. Migration of Existing Spec/Fixtures

- **No auto-migration.** `?.` is never added automatically. Authors opt in per use case.
- **Existing fixture corpus** (Stream D, `design/gert/testdata/runbooks/`): No changes needed. These runbooks were authored with full knowledge of which fields exist. They use plain `${...}` correctly.
- **Existing conformance vectors** (`tv-gxl-path.yaml`, `tv-gxl-parse.yaml`, `tv-gxl-eval.yaml`): No changes. These test GXL-level behavior, not GIS optional-chaining. The new `tv-gis-path.yaml` is additive.

---

## 8. Open Questions for ormasoftchile

| # | Question | Barbara's Recommendation | Needs User Input? |
|---|----------|-------------------------|-------------------|
| OQ-OC-1 | Is `?.[N]` (optional bracket indexing) in or out? | **IN.** Without it, authors hit hard-error on array access even when they've opted into tolerance. Inconsistent UX. | Yes — confirm or override. |
| OQ-OC-2 | Should `${?.root}` (optional on the root identifier itself) be legal syntax? | **No.** The root identifier is always the first GDP segment; if you don't know whether a variable exists at all, you should use `when:` guards. Making the root optional would mask typos. However, `${root?.field}` IS legal (the `?.` guards the hop from root to field). | Confirm. |
| OQ-OC-3 | Error code for the "resolved to empty via optional chain" case — should it emit a **diagnostic trace** (not error) in the JSONL audit trail? | **Recommend yes** — a `GIS-PATH-OPTIONAL-MISS` info-level trace event would help debugging without failing the run. But it's a separate concern (audit trail schema). Defer to runtime design phase? | Advisory. |
| OQ-OC-4 | Does `?.` compose with stdlib calls? E.g., `${str.toLower(user?.name)}` — if `user` is missing, `str.toLower("")` is called. Is this the desired behavior, or should the optional-chain propagate through the function call? | **Recommend: no propagation through function calls.** The GDP resolves first; if it short-circuits to `""`, the function receives `""`. Functions don't know about optional-chaining. This is simple and predictable. | Confirm. |

---

## 9. Summary of Locked Decisions (This Proposal)

| Decision | Value | Rationale |
|----------|-------|-----------|
| Missing path default value | `""` (empty string) | GIS produces string output; empty string is identity in concatenation; avoids "null" in user text |
| Short-circuit semantics | JS/TS-compatible full-tail short-circuit | No invented semantics; authors already know the rules |
| Scope | GIS `${...}` only | GXL boolean and GCP capture have separate mechanisms |
| `?.[N]` bracket indexing | IN (recommended) | Consistency; avoids partial-tolerance footgun |
| `??` nullish coalescing | OUT (deferred) | `vars:` and `capture.default:` cover the use case today |
| Existing fixtures | No migration | Opt-in only; existing corpus is correct as-is |

---

## Appendix A: Grammar Diff Summary

```
# In gxl.ebnf §3, the GDP production changes from:
GDP = IDENT { '.' IDENT | '[' INTEGER ']' } ;

# To:
GDP = IDENT { PathSegment } ;
PathSegment = '.' IDENT
            | '?.' IDENT
            | '[' INTEGER ']'
            | '?.[' INTEGER ']'
            ;
```

The `?.` token is added to the lexer as a two-character atomic token (not decomposable). The `?.[` sequence is recognized as the start of an optional bracket access (three chars: `?`, `.`, `[`).

---

*End of proposal. Awaiting review.*
