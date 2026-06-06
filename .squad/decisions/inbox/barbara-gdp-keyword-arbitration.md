# Barbara — GDP Namespace Keyword Arbitration

**Author:** Barbara — Lead / Architect  
**Date:** 2026-06-05T23:12:22-04:00  
**Triggered by:** Don's Phase B memo — `don-phase-b-lexer-parser.md` (Grammar Note section)  
**Affects:** `gxl.ebnf` §2–3, `design/gert/sections/03a-expression-language.tex`, Don PR #10

---

## Option Chosen: **Option A — Legitimize the pragmatic parser behavior**

**Rationale:** Option A is consistent with all three gate vectors (TV-GXL-PARSE-028, -048, -083) and introduces no new parser ambiguity. The `str.foo` case that could break Option A does NOT break it, because the dual-role carve-out is precisely scoped to "NOT immediately followed by DOT" — when a namespace keyword IS followed by DOT it remains committed to NamespaceCall only, preserving the GXL-PARSE-001 result for `str.foo`. Option A also matches the parser already shipped in Don's PR #10, so no implementation change is needed.

---

## Decision Rule Codified

> Namespace prefix keywords `str`, `list`, and `regex` serve a **dual role**:
>
> 1. When immediately followed by `.`: committed to `NamespaceCall` production only.
>    — If NamespaceCall fails (e.g. method but no `(`), all Primary alternatives fail → `GXL-PARSE-001`.
> 2. When NOT immediately followed by `.`: treated as a valid GDP root identifier under PEG ordered alternation.
>    — NamespaceCall fails immediately (requires DOT), falls through to GDP → parse proceeds.
>
> This carve-out does NOT extend to `math`, `len`, `now`, `and`, `or`, `not`, `true`, `false`, or `null`.

---

## Files Modified

| File | Change Summary |
|------|---------------|
| `design/gert/grammar/gxl.ebnf` | §2 keyword comment updated: "also may not be identifiers" → dual-role description. §2 IDENT note extended with explicit Exception block describing the carve-out, scope, and references to all three key vectors. §3 NamespaceCall comment updated: `str.foo` explanation revised to reference dual-role rule (DOT-commitment). §3 GDP production note extended with `DUAL-ROLE NOTE` paragraph. |
| `design/gert/sections/03a-expression-language.tex` | Keywords table rows for `str`/`list`/`regex` annotated "dual-role GDP root". New "Dual-role namespace keywords" paragraph added after table. Identifiers subsection: GXL-PARSE-010 rule updated with explicit exception sentence. GDP subsection: new "Dual-role namespace keywords as GDP roots" paragraph added with full rule, both examples, and scope guard. NamespaceCall `str.foo` note: revised to cite DOT-commitment rule rather than "str is a keyword, not IDENT". |

---

## Verification Trace

### Gate Vectors

| Vector | Input | Expected | Trace under Option A | Result |
|--------|-------|----------|----------------------|--------|
| TV-GXL-PARSE-028 | `list[10]` | `parse_ok` | `list` not followed by `.` → NamespaceCall fails; GDP carve-out applies → `list` is valid GDP root → `list[10]` parses as GDP with bracket-index | ✅ parse_ok |
| TV-GXL-PARSE-048 | `str.contains(message, "error")` | `parse_ok` | `str` followed by `.` → NamespaceCall: `str.contains(...)` → known method, valid ArgList → parse_ok | ✅ parse_ok |
| TV-GXL-PARSE-083 | `str.foo` | `GXL-PARSE-001` | `str` followed by `.` → DOT-committed to NamespaceCall; `foo` matches Method, but EOF/no `(` → NamespaceCall fails; GDP carve-out inapplicable (followed by DOT); all Primary alternatives fail → GXL-PARSE-001 | ✅ GXL-PARSE-001 |

### Regression Check — Surrounding Namespace Vectors

| Vector | Input | Expected | Trace | Result |
|--------|-------|----------|-------|--------|
| TV-GXL-PARSE-049 | `str.trim(label)` | `parse_ok` | `str` + `.` → NamespaceCall → known method → parse_ok | ✅ |
| TV-GXL-PARSE-050 | `list.contains(tags, "prod")` | `parse_ok` | `list` + `.` → NamespaceCall → known method → parse_ok | ✅ |
| TV-GXL-PARSE-051 | `regex.match(path, "^/api/")` | `parse_ok` | `regex` + `.` → NamespaceCall → known method → parse_ok | ✅ |
| TV-GXL-PARSE-052 | `str.contains(message, "x") and len(items) > 0` | `parse_ok` | `str` + `.` → NamespaceCall → parse_ok for whole expr | ✅ |
| TV-GXL-PARSE-053 | `str.startsWith(config.prefix, "prod")` | `parse_ok` | `str` + `.` → NamespaceCall → parse_ok | ✅ |
| TV-GXL-PARSE-081 | `and` | `GXL-PARSE-010` | `and` is KW_AND — NOT in carve-out set → keyword-as-identifier → GXL-PARSE-010 | ✅ |
| TV-GXL-PARSE-082 | `or` | `GXL-PARSE-010` | `or` is KW_OR — NOT in carve-out set → GXL-PARSE-010 | ✅ |

No regressions found. Zero vectors in the corpus test bare `str`, `list`, or `regex` without a following `.` or `[`/other token already covered by the vectors above.

---

## Commit SHA

See git log after push.

---

## Don's PR #10 Parser — Action Required?

**No.** Don's parser already implements the correct behavior: when a namespace keyword (`str`, `list`, `regex`) is NOT followed by `.`, it falls through from NamespaceCall and treats the token as a GDP root. This memo simply ratifies that behavior as the normative spec. Don's PR #10 requires zero code changes to match the updated spec.

TV-GXL-PARSE-028 (`list[10]`) remains `parse_ok` — unchanged from the corpus as committed.

---

## Scribe Note

No changes to `decisions.md` or `team.md` required — this is a spec clarification within Barbara's arbiter authority. Don may merge PR #10 once Phase A+B gate clears.
