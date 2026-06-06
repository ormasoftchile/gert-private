# Memo: YAML Escape Fix — TV-GXL-PARSE-062 / TV-GXL-PARSE-063

**From:** Tess (Test Engineer / Conformance Steward)
**To:** Squad / Don / Ken
**Date:** 2026-06-05T23:12:22-04:00
**Re:** Fix for Don's "Vector Bug Report for Tess" in `don-phase-b-lexer-parser.md`

---

## Summary

Two conformance vectors in `design/gert/conformance/tv-gxl-parse.yaml` had
incorrect YAML encoding that caused their `input` fields to contain a
double-backslash sequence (`\\q`, `\\xFF`) instead of the intended
single-backslash invalid-escape sequences (`\q`, `\xFF`). This made the
GXL parser see valid escape sequences and return `parse_ok`, defeating the
purpose of the negative test.

**Commit:** `424a334`
**Pushed to:** `gert-private/main`

---

## Root Cause

Single-quoted YAML scalars do **not** interpret backslash escapes — only the
`''` sequence (escaped single-quote) is special. The original encoding:

```yaml
input: '"hello \\q world"'   # WRONG: yields "hello \\q world" (two backslashes)
input: '"hex \\xFF"'          # WRONG: yields "hex \\xFF" (two backslashes)
```

produced two-character sequences `\\q` and `\\xFF`, which the GXL lexer
parsed as a valid `\\` escape followed by a literal character — resulting in
`parse_ok` instead of `GXL-PARSE-004`.

---

## Fix Applied

**File:** `design/gert/conformance/tv-gxl-parse.yaml`

| Vector | Before | After |
|---|---|---|
| TV-GXL-PARSE-062 | `'"hello \\q world"'` | `'"hello \q world"'` |
| TV-GXL-PARSE-063 | `'"hex \\xFF"'` | `'"hex \xFF"'` |

---

## Verification

### YAML `repr` check (Python `yaml.safe_load`)

```
Index 61: TV-GXL-PARSE-062
  input repr: '"hello \\q world"'   ← Python repr; actual string has single backslash ✓
  input value: "hello \q world"
  len: 16

Index 62: TV-GXL-PARSE-063
  input repr: '"hex \\xFF"'          ← actual string has single backslash ✓
  input value: "hex \xFF"
  len: 10
```

Both vectors now present the intended invalid-escape GXL source strings to
the parser, which should produce `GXL-PARSE-004` rather than `parse_ok`.

### Schema validation (`design/gert/conformance/schema.json`)

83/83 vectors loaded successfully. One pre-existing schema error (unrelated to
this fix) was noted: `TV-GXL-PARSE-010` has a `null` entry in its `tags`
array — this predates the current change and was NOT introduced by this fix.

### Spot-checks (3 other vectors — no regressions)

| Index | ID | input (repr) |
|---|---|---|
| 0 | TV-GXL-PARSE-001 | `'42'` |
| 30 | TV-GXL-PARSE-031 | `'count < 100'` |
| 60 | TV-GXL-PARSE-061 | `'1.2.3'` |

All spot-checked vectors are unchanged.

---

## Coordination Note

This fix is now on `gert-private/main` (commit `424a334`) ahead of Phase A
PR #8 (Ken's sync infra). Ken's CI sync will pick up the corrected vectors,
protecting Don's Phase B 83/83 pass rate from dropping to 81/83 upon PR
merge.
