## Truthy Conformance Vectors — Memo to Tess

**Date:** 2026-06-07
**From:** Barbara (Lead/Architect)
**To:** Tess (Conformance)
**Re:** New vectors for `tv-gxl-eval.yaml` per Truthy arbitration (Option c: Strict)

---

### Context

The Truthy arbitration (see `barbara-truthy-arbitration.md`) chose **Option (c) Strict**: non-bool in boolean context → `GXL-TYPE-002`. The following vectors pin this behavior normatively so every runtime (Go, C#, TS) can test against the same expectations.

### Vectors to Add

All vectors below test a value in boolean context. The expression form is `<value>` used as a boolean condition (e.g., as the operand of `not`, or as a `when:` condition). Expected error code is `GXL-TYPE-002` for all non-bool types.

#### Existing assumed behavior to pin down (sanity vectors)

| # | Input Expression | Expected Result | Notes |
|---|---|---|---|
| 1 | `false` | `false` (no error) | Bool literal — sanity baseline |
| 2 | `true` | `true` (no error) | Bool literal — sanity baseline |

These may already be covered by existing vectors. If so, mark as "already covered" and skip.

#### New vectors (the Truthy arbitration additions)

| # | Input Expression | Expected Result | Notes |
|---|---|---|---|
| 3 | `null` in boolean context | `GXL-TYPE-002` | null is not bool |
| 4 | `0` in boolean context | `GXL-TYPE-002` | Number — not bool, even if zero |
| 5 | `1` in boolean context | `GXL-TYPE-002` | Number — not bool, even if nonzero |
| 6 | `-1` in boolean context | `GXL-TYPE-002` | Negative number — not bool |
| 7 | `0.0` in boolean context | `GXL-TYPE-002` | Float zero — not bool |
| 8 | `""` (empty string) in boolean context | `GXL-TYPE-002` | Empty string — not bool |
| 9 | `"false"` (string) in boolean context | `GXL-TYPE-002` | String containing "false" — not bool |
| 10 | `" "` (whitespace string) in boolean context | `GXL-TYPE-002` | Non-empty string — still not bool |
| 11 | `[]` (empty array) in boolean context | `GXL-TYPE-002` | **KEY VECTOR** — the original question. Empty array is not bool. |
| 12 | `[0]` (non-empty array) in boolean context | `GXL-TYPE-002` | Non-empty array — still not bool |
| 13 | `[null]` (array with null) in boolean context | `GXL-TYPE-002` | Array containing null — still not bool |
| 14 | `{}` (empty object) in boolean context | `GXL-TYPE-002` | **KEY VECTOR** — the original question. Empty object is not bool. |
| 15 | `{"a": 0}` (non-empty object) in boolean context | `GXL-TYPE-002` | Non-empty object — still not bool |

#### Recommended expression forms for the vectors

Use variable binding to place each value in boolean context. Example YAML structure:

```yaml
- id: TV-GXL-EVAL-XXX
  description: "Truthy: null in boolean context → GXL-TYPE-002"
  input:
    expression: "x"
    variables:
      x: null
  expected:
    error: "GXL-TYPE-002"
  tags: [truthy, type-error, arbitration-2026-06-07]
```

For literal forms (numbers, strings), the expression can be the literal itself in a boolean position:

```yaml
- id: TV-GXL-EVAL-XXX
  description: "Truthy: 0 in boolean context → GXL-TYPE-002"
  input:
    expression: "not 0"
  expected:
    error: "GXL-TYPE-002"
  tags: [truthy, type-error, arbitration-2026-06-07]
```

### Summary

- **2 sanity vectors** (existing behavior, may already be covered)
- **13 new vectors** (3 through 15 above)
- **Total: up to 15**, net new ~13
- **All non-bool vectors expect `GXL-TYPE-002`** — this is the whole point
- **Key vectors: #11 (`[]`) and #14 (`{}`)** — these are the ones that triggered the arbitration

### Source

Decision: `.squad/decisions/inbox/barbara-truthy-arbitration.md`
Spec: `design/gert/sections/03a-expression-language.tex` §Truthy Coercion in Boolean Context
