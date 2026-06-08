### 2026-06-04T23:56:26-04:00: User directive — GXL Phase 1 OQ1 + OQ2 ratification

**By:** ormasoftchile (Germán, via Copilot)
**Context:** `.squad/decisions/inbox/barbara-gxl-phase1-plan.md` § 7 Open Questions

**Decisions:**

| # | Question | Ratified Answer |
|---|----------|-----------------|
| **OQ1** | Does `false and missing_var` short-circuit (no error) or evaluate both sides? | **Short-circuit.** `and`/`or` are short-circuiting. `false and X` returns `false` without evaluating `X`. `true or X` returns `true` without evaluating `X`. Matches Python, SQL, JavaScript. Must be specified in `gxl.ebnf` semantics annex. |
| **OQ2** | Can `capture.default:` apply to subtree captures (objects/arrays) or only scalars? | **Scalars only.** `capture.default:` is permitted only when the declared capture type is a scalar (string, number, boolean, null). Subtree captures (object/array) MUST NOT declare a default — authors must guard the consuming step with `when:` instead. Parser/planner rejects subtree + default at plan time (error code `GCP-DEFAULT-SUBTREE`). |

**Why:** Unblocks Stream C (Conformance Corpus) which begins Day 3 of Phase 1. Both decisions are now binding for the grammar files (Stream A) and conformance test vectors.

**Status:** Ratified — design contract for Phase 1.

**Open questions remaining (Barbara owns, no user input needed):**
- OQ3: Portable JSON value model — Barbara to spec in Stream A (RFC 8259 types).
- OQ4: Add `list.indexOf` to stdlib namespace — Barbara to add to grammar.

---

# Stream A — Reference Grammar Complete

**Author:** Barbara — Lead / Architect  
**Date:** 2026-06-04T23:56:26-04:00  
**Status:** Complete — files committed to `design/gert/grammar/`  
**References:**
- `design/gert/grammar/gxl.ebnf` (produced)
- `design/gert/grammar/gis.ebnf` (produced)
- `design/gert/grammar/gcp.ebnf` (produced)
- `.squad/decisions/inbox/copilot-directive-gxl-oq1-oq2-ratification.md` (input)
- `.squad/decisions/inbox/barbara-gxl-phase1-plan.md` (Stream A plan)

---

## 1. Files Produced

| File | Size | Description |
|------|------|-------------|
| `design/gert/grammar/gxl.ebnf` | ~25 KB | GERT Expression Language — boolean expression grammar |
| `design/gert/grammar/gis.ebnf` | ~16 KB | GERT Interpolation Syntax — `${...}` template grammar |
| `design/gert/grammar/gcp.ebnf` | ~26 KB | GERT Capture Path — capture path grammar |

All three files include:
- EBNF header (title, version `1.0.0-draft`, date, references, evaluation sites)
- Lexical grammar
- Syntactic grammar with explicit precedence and associativity
- Semantics annex / policy sections
- Error class catalog
- Stream A open issues

---

## 2. Key Design Calls Made

### 2.1 Identifier scope: ASCII-only (GXL §2.3)
**Decision:** GXL identifiers are ASCII-only (`a-z`, `A-Z`, `0-9`, `_`).  
**Rationale:** Variable names originate from YAML keys and step IDs (ASCII by convention). Unicode identifiers admit visual-spoofing attacks with no upside for the current authoring population. If Unicode is needed in v2, NFC normalization and safe Unicode block allowlisting must be specified.

### 2.2 Number model: IEEE 754 float64, scientific notation included (GXL §2.5)
**Decision:** All GXL numbers are IEEE 754 double-precision float64. Integer literals, decimal literals, and scientific notation (`1.5e10`) are all supported. Hex/octal/binary not supported. NaN/Infinity not representable as literals.  
**Rationale:** Matches the Portable JSON Value Model (OQ3). All runtimes (Go, C#) have native float64. Scientific notation covers environmental thresholds (e.g., `1.5e-3` latency bounds).

### 2.3 GIS escape convention: `\${` (GIS §2.3)
**Decision:** `\${` is the escape for a literal `${` in a GIS template.  
**Rationale:** Specified in Stream A task. This is a **deviation from the original proposal** (proposal §2.6 used `$${` — double-dollar). The migration tool (Stream E) must handle `$${` → `\${` conversion. Flagged as OI-GIS-01. Team ratification recommended.

### 2.4 GIS embedded expression: full GXL (GIS §2.4)
**Decision:** The expression inside `${...}` is a full GXL `Expression`, not just a GDP path.  
**Rationale:** The grammar superset keeps GIS formally clean (one grammar, not two). In practice, authors use bare GDP paths. Future IDE tooling can leverage expression-awareness without a grammar break.

### 2.5 OQ3 — Portable JSON Value Model: RFC 8259 types (GIS §4)
**Decision:** PJVM types are `null`, `boolean`, `number` (float64, no NaN/Inf), `string` (UTF-8), `array`, `object`. No date, binary, undefined, int64, or decimal.  
**Rationale:** Strict RFC 8259 subset with explicit NaN/Inf exclusion. Ensures Go↔C# round-trip fidelity. Object key sorting in canonical serialization ensures deterministic `${capture_var}` output for audit trail comparison.

### 2.6 OQ4 — list.indexOf added (GXL §4.3)
**Decision:** `list.indexOf(l: list, item: scalar) → number` returns the zero-based index of the first matching element, or `-1` if not found.  
**Rationale:** Ratified in the OQ1/OQ2 directive as part of OQ4 resolution. `-1` sentinel (vs. null) aligns with Go, Java, and C# conventions. Conformance corpus must include a not-found vector.

### 2.7 OQ2 — capture.default: scalars only (GCP §6)
**Decision:** `capture.default:` is permitted only for scalar captures (null, boolean, number, string). Subtree captures (object, array) MUST NOT declare a default. Violation → `GCP-DEFAULT-SUBTREE` at plan time.  
**Rationale:** Directly implements the ratified OQ2 decision. The prohibition forces authors to use `when: varname != null` guards for optional subtrees, which is the correct pattern for traceability.

### 2.8 exit_code normalization (GCP §3.4, OI-GCP-01)
**Decision:** Canonical spelling is `exit_code` (snake_case). `exitCode` is rejected at parse time (GCP-PARSE-001).  
**Rationale:** GERT field naming convention is snake_case throughout. Don's audit found `exitCode` as a legacy form. Stream E migration tool must convert it.

### 2.9 GDP shared grammar (GXL §3, GIS §2.4, GCP §4)
**Decision:** The GERT Dotted Path (GDP) grammar is identical in all three languages: `IDENT { "." IDENT | "[" INTEGER "]" }` (in GXL/GIS) and `"." PathSegment { PathSegment }` (in GCP, appended to source prefix).  
**Rationale:** Consistency. A GDP path in `when: config.region == "us-east-1"` navigates the same way as `${config.region}` in GIS and `json.config.region` in GCP.

---

## 3. Ratified OQ Resolutions Incorporated

| OQ | Decision | Incorporated In |
|----|----------|----------------|
| **OQ1** | `and`/`or` short-circuit | GXL §3 (OrExpr, AndExpr), §5.1 |
| **OQ2** | `capture.default:` scalars only; subtrees → `GCP-DEFAULT-SUBTREE` | GCP §6 |
| **OQ3** | Portable JSON Value Model (Barbara to spec) | GIS §4 |
| **OQ4** | `list.indexOf` added to stdlib | GXL §4.3 |

---

## 4. Ambiguities Surfaced During Grammar Writing

These are documented as Stream A Open Issues in each file and summarized here for team action:

### High-priority (Phase 2 blockers if unresolved):

| ID | File | Issue | Impact |
|----|------|-------|--------|
| OI-GIS-01 | gis.ebnf | `\${` vs `$${` escape convention conflict with original proposal | Migration tool (Stream E) must handle both; team must ratify canonical form |
| OI-GIS-02 | gis.ebnf | Bare `$` in tool argv coexistence with GIS (echo.tool.yaml, jsonrpc-server.tool.yaml) | Stream D fixture audit needed |
| OI-GCP-02 | gcp.ebnf | Root JSON capture without path: `json` (no GDP) is currently a parse error | May block common patterns; team ratification needed |
| OI-GXL-06 | gxl.ebnf | `len()` argument count: grammar and semantics must explicitly state exactly 1 argument | Conformance corpus vector needed |

### Medium-priority (Phase 2 implementation guidance):

| ID | File | Issue |
|----|------|-------|
| OI-GXL-03 | gxl.ebnf | Modulo semantics for negative operands (Go fmod vs Python floor) — need conformance vectors |
| OI-GXL-05 | gxl.ebnf | GDP path in function argument: resolution semantics need explicit conformance vector |
| OI-GIS-04 | gis.ebnf | Nested `${...}` inside GXL string literals — parser must not split naively |
| OI-GCP-01 | gcp.ebnf | `exitCode` → `exit_code` normalization — Stream E migration tool scope |
| OI-GCP-03 | gcp.ebnf | Capture path validation scope for includes/branches — feeds Stream F |
| OI-GCP-05 | gcp.ebnf | `http.*` capture lifetime scope clarification needed in spec |
| OI-GCP-06 | gcp.ebnf | YAML 1.2 core schema must be specified explicitly; timestamp behavior |

### Low-priority (design notes for future versions):

| ID | File | Issue |
|----|------|-------|
| OI-GXL-01 | gxl.ebnf | String interning / mutability semantics |
| OI-GXL-02 | gxl.ebnf | Integer overflow in arithmetic (silent float64 rounding) |
| OI-GXL-04 | gxl.ebnf | `not null` → TYPE-002; suggest `defined()` builtin for v2 |
| OI-GXL-07 | gxl.ebnf | Multiline expressions in YAML block scalars — valid, needs authoring guide note |
| OI-GIS-03 | gis.ebnf | Maximum interpolation output size limits |
| OI-GIS-05 | gis.ebnf | Case operation locale sensitivity (invariant locale chosen) |
| OI-GCP-04 | gcp.ebnf | Multiple captures from same path (valid, no deduplication) |

---

## 5. Deviations from Proposal / OQ Resolutions

| Deviation | Original | This Grammar | Justification |
|-----------|----------|--------------|---------------|
| GIS escape | `$${` (double-dollar) | `\${` | Stream A task specification; see OI-GIS-01 |
| GIS embedded expression | GDP path only | Full GXL Expression | Task spec; superset is backward-compatible |
| Scientific notation | Not in proposal | Included | Task spec; common in threshold expressions |
| `str.length` | Not in proposal | Added (aliases `len()`) | Task spec; namespace consistency |
| Cross-step sources | Not in proposal | `step.{id}.*` added | Task spec; required for multi-step runbooks |
| HTTP sources | Not in proposal | `http.*` added | Task spec |
| Event sources | Extended | `event.headers.*`, `event.id` added | Task spec |
| `exit_code` spelling | Mixed (`exitCode`/`exit_code`) | `exit_code` canonical | Consistency; Don's audit flagged both |

---

## 6. Stream Unblocking

These streams can now start:

| Stream | Can Start? | Notes |
|--------|-----------|-------|
| **B — Spec Rewrite** | ✅ Yes | Grammar files are available |
| **C — Conformance Corpus** | ✅ Yes | Error codes and semantics are defined |
| **D — Fixture Migration** | ✅ Yes | Grammar constrains which patterns are valid |
| **F — Parser Gate Spec** | ✅ Yes | Error code catalog is complete |
| **Phase 2 — Go/C# Parsers** | ⏳ Phase 2 | Awaiting conformance corpus (Stream C) |

---

# GXL/GIS/GCP — Phase 1 Detailed Plan

**Author:** Barbara — Lead / Architect  
**Date:** 2026-06-04T23:36:26-04:00  
**Status:** Proposed — awaiting ormasoftchile ratification  
**References:**  
- `design/gert/expression-language-proposal.md` (design contract)  
- `design/gert/expression-evaluation-audit.md` (Don's inventory)  
- `.squad/decisions.md` § "2026-06-04 — GXL/GIS/GCP — Open question resolutions"

---

## 1. Phase 1 Goal Statement

Phase 1 is complete when: (a) the GERT specification (`design/gert/sections/*.tex`) contains zero references to `expr-lang/expr` or Go `text/template` as authored-surface grammar; (b) a machine-readable conformance corpus of ≥200 test vectors exists, covering GXL, GIS, and GCP at the lexer, parser, evaluator, and semantic-error levels; (c) all 22 shipped runbook fixtures plus 12 tool fixtures compile cleanly against the new parsers (or a validation script that enforces the new grammar); and (d) a formal EBNF grammar file exists that both Go and C# runtimes will parse from directly. **The conformance corpus IS the spec** — any behavior not covered by a test vector is undefined and may diverge across runtimes.

---

## 2. Work Stream Decomposition

### Stream A — Reference Grammar (EBNF/PEG File)

| Field | Value |
|-------|-------|
| **Owner** | Barbara |
| **Inputs** | `design/gert/expression-language-proposal.md` §1.2, §2.3, §3.3; Q1–Q5 resolutions |
| **Deliverables** | `design/gert/grammar/gxl.ebnf`, `design/gert/grammar/gis.ebnf`, `design/gert/grammar/gcp.ebnf` |
| **Acceptance criteria** | Each grammar file parses with a standard EBNF/PEG validator (e.g., `abnfgen`, `pigeon`, or Railroad Diagram Generator). Every token, production, and precedence rule from the proposal is formalized. Grammar files are the single source of truth — spec .tex and runtime implementations derive from them. |
| **Estimated effort** | **S** (1–2 days) |
| **Blocks** | Nothing — first to start |
| **Blocked-by** | None |

---

### Stream B — Spec Rewrite

| Field | Value |
|-------|-------|
| **Owner** | Barbara + new agent: **Spec Editor** (to be spawned) |
| **Inputs** | Grammar files (Stream A); Don's audit (15 evaluation sites); proposal §5.1 rewrite table |
| **Deliverables** | Modified `.tex` files (see §4 below). New sections: `design/gert/sections/03a-expression-language.tex` (GXL), `design/gert/sections/03b-interpolation-syntax.tex` (GIS), `design/gert/sections/03c-capture-paths.tex` (GCP) |
| **Acceptance criteria** | `grep -rn "expr-lang\|text/template\|fromJSON\|fromYAML\|{{\s*\." design/gert/sections/` returns zero hits. All 15 evaluation sites from Don's audit reference the correct replacement (GXL/GIS/GCP/structured). Spec compiles (LaTeX) without errors. |
| **Estimated effort** | **L** (5–7 days) |
| **Blocks** | Stream D (fixtures reference spec examples), Stream B-corpus (test vectors reference spec semantics) |
| **Blocked-by** | Stream A (grammar must be final before spec text is written) |

---

### Stream C — Conformance Corpus

| Field | Value |
|-------|-------|
| **Owner** | Don + new agent: **Conformance Tester** |
| **Inputs** | Grammar files (Stream A); proposal §5.4 conformance tables; Q1–Q5 resolutions |
| **Deliverables** | `design/gert/conformance/` directory with YAML test vector files: one file per category (TV-GXL-PARSE.yaml, TV-GXL-EVAL.yaml, etc.). Minimum 200 vectors total. |
| **Acceptance criteria** | Each vector has: `id`, `input` (expression/template/path string), `variables` (context map), `expected` (result value OR `error_class`). A schema validator (`design/gert/conformance/schema.json`) validates every vector file. Total count ≥ 200. Categories cover all §3 items below. |
| **Estimated effort** | **XL** (8–12 days) |
| **Blocks** | Phase 2 runtimes cannot start without this corpus |
| **Blocked-by** | Stream A (grammar); partially Stream B (semantic edge cases surface during spec rewrite) |

---

### Stream D — Fixture Migration

| Field | Value |
|-------|-------|
| **Owner** | Don |
| **Inputs** | Proposal §5.2 fixture table; grammar files (Stream A); Q1 resolution (`iterate.over: ${services}`) |
| **Deliverables** | All 22 runbook fixtures + 12 tool fixtures rewritten in-place. A validation script (`scripts/validate-fixtures.sh`) that runs the GXL/GIS/GCP parser stubs against every fixture and exits 0. |
| **Acceptance criteria** | Zero Go template syntax (`{{ }}`) in any fixture. Zero `&&`/`||`/`!` operators. Zero `contains` infix. Zero `$.path` JSONPath. Zero `| length` pipe. Validation script passes in CI. |
| **Estimated effort** | **M** (3–4 days) |
| **Blocks** | Nothing downstream in Phase 1 |
| **Blocked-by** | Stream A (grammar); Stream B (spec must clarify `capture.default:` syntax from Q2 before fixtures using defaults can be migrated) |

---

### Stream E — Migration Tooling (`gert migrate-expr`)

| Field | Value |
|-------|-------|
| **Owner** | Don |
| **Inputs** | Grammar files (Stream A); fixture migration patterns (Stream D learnings) |
| **Deliverables** | `cmd/gert-migrate/main.go` (or standalone script) that: reads a YAML runbook, applies mechanical transformations, writes result. Handles: `{{ .x }}` → `${x}`, `&&` → `and`, `||` → `or`, `!` → `not`, `contains` infix → `str.contains()`, `$.x` → bare `x`, pipe removal. |
| **Acceptance criteria** | Running the tool on the pre-migration fixture corpus produces output identical to the manually-migrated fixtures (diff = 0). Tool reports non-mechanical transformations it cannot handle (e.g., `fromJSON` patterns requiring structural changes). |
| **Estimated effort** | **M** (3–4 days) |
| **Blocks** | Nothing (convenience tool, not a gate) |
| **Blocked-by** | Stream D (patterns must be validated manually first; tool automates them) |

---

### Stream F — Parser/Planner Gate Specification

| Field | Value |
|-------|-------|
| **Owner** | Barbara |
| **Inputs** | Grammar files (Stream A); proposal §4 parse-time enforcement plan |
| **Deliverables** | `design/gert/sections/03d-parse-time-enforcement.tex` — normative spec section defining: (1) error codes (GXL-001 through GXL-NNN, GIS-001…, GCP-001…, SEM-001…); (2) error message format; (3) the `ValidatedPlan` type contract; (4) the guarantee that `RunHandle.Next()` never executes unvalidated expressions. |
| **Acceptance criteria** | Every error class from the conformance corpus (Stream C) maps to a named error code in this document. The `ValidatedPlan` contract is sufficient for both Go and C# to implement independently. |
| **Estimated effort** | **M** (2–3 days) |
| **Blocks** | Phase 2 runtime implementors need this as their interface contract |
| **Blocked-by** | Stream A; Stream C (error categories emerge during corpus writing) |

---

## 3. Test Vector Categories (Conformance Corpus Skeleton)

| Category ID | Description | Target Count | Input → Expected |
|-------------|-------------|:---:|---|
| **TV-GXL-PARSE** | Lexer/parser correctness — valid tokens, operator precedence, keyword recognition, string escape sequences, number formats | 30 | Expression string → AST (or parse-success flag) |
| **TV-GXL-EVAL** | Boolean evaluation semantics — `and`/`or`/`not`, comparison operators, short-circuit, namespace function return values | 35 | Expression + variable map → `true`/`false`/value |
| **TV-GXL-PATH** | GDP dotted-path resolution inside GXL identifiers — objects, arrays, missing keys | 15 | Expression + nested variable map → value or error |
| **TV-GXL-ERROR** | Error class taxonomy — type mismatch, missing variable, null in ordered comparison, divide-by-zero, unknown namespace, forbidden syntax | 25 | Expression + variables → `error_class` string |
| **TV-GIS-INTERP** | `${...}` interpolation — single, multiple, adjacent text, escaped `$${`, no-interpolation passthrough | 20 | Template string + variables → rendered string |
| **TV-GIS-PATH** | GDP inside `${...}` resolves identically to direct GXL path — nested objects, arrays, index access | 15 | Template string + nested map → rendered string |
| **TV-GIS-TYPE** | Type coercion to string for interpolation — number, bool, null, array→JSON, object→JSON | 10 | Template string + typed variables → rendered string or error |
| **TV-GIS-ERROR** | GIS error cases — missing key (hard error), invalid path syntax, unclosed `${`, expression inside `${}` | 10 | Template string → `error_class` |
| **TV-GCP-PARSE** | Capture-path parsing — valid sources (`stdout`, `stderr`, `exitCode`, `json`, `yaml`), dot segments, index segments | 15 | Path string → parsed structure or parse error |
| **TV-GCP-RESOLVE** | Capture-path resolution against typed step outputs — scalar, nested object, array element, subtree capture (Q5) | 15 | Path + step output → captured value |
| **TV-GCP-DEFAULT** | `capture.default:` fallback behavior (Q2) — path fails → default used; path succeeds → default ignored | 8 | Path + output + default → value |
| **TV-NAMESPACE** | Stdlib namespace functions — `str.startsWith`, `str.toLower`, `str.contains`, `str.trim`, `list.contains`, `list.indexOf`, `len`, `regex.match` | 25 | Function call + args → return value or error |
| **TV-CONFORM-CROSSCUT** | Integration scenarios spanning GXL+GIS+GCP — e.g., iterate over subtree capture, conditional on captured list length, interpolation of captured nested path | 12 | Full step context (captures + variables + expression/template) → final evaluated result |
| | **TOTAL** | **≥ 235** | |

### Vector File Format (example)

```yaml
# design/gert/conformance/tv-gxl-eval.yaml
schema: "conformance/v1"
category: "TV-GXL-EVAL"
vectors:
  - id: TV-GXL-EVAL-001
    description: "Simple boolean and"
    input: "active and verified"
    variables:
      active: true
      verified: true
    expected:
      value: true

  - id: TV-GXL-EVAL-002
    description: "Short-circuit — second operand not evaluated when first is false"
    input: "false and missing_var"
    variables: {}
    expected:
      error_class: null  # short-circuit means missing_var is never resolved
      value: false

  - id: TV-GXL-EVAL-003
    description: "Type error — string in boolean position"
    input: "hostname"
    variables:
      hostname: "prod-1"
    expected:
      error_class: "TYPE_ERROR"
      message_contains: "expected bool, got string"
```

---

## 4. Spec Surface Audit

### Files Requiring Modification

| File | Line Range | Action | Replacement |
|------|-----------|--------|-------------|
| `03-schema-vnext.tex` | 412–426 | **Replace** | Remove `expr-lang/expr` reference. Insert GXL reference pointing to `03a-expression-language.tex` |
| `03-schema-vnext.tex` | 435–440 | **Replace** | `&&`/`||`/`!` operator table → `and`/`or`/`not` |
| `03-schema-vnext.tex` | 454–473 | **Replace** | Examples rewritten with `and`/`or`/`not` |
| `03-schema-vnext.tex` | 475–560 | **Edit** | Keep `str.*` section; remove `expr-lang` references; add "forbidden infix" note |
| `03-schema-vnext.tex` | 562–591 | **Delete & replace** | Remove "Go templates" paragraph + `fromJSON`/`fromYAML`. Insert GIS reference pointing to `03b-interpolation-syntax.tex` |
| `03-schema-vnext.tex` | 593–598 | **Edit** | Error model references GXL/GIS, not `expr` |
| `03-schema-vnext.tex` | 886–913 | **Edit** | CLI step: "templates expanded" → "GIS interpolated" |
| `03-schema-vnext.tex` | 1250–1262 | **Edit** | Collector `fields[].when`: "Go template" → "GXL expression" |
| `03-schema-vnext.tex` | 2144 | **Edit** | `tool.args`: "templates expanded" → "GIS interpolated" |
| `03-schema-vnext.tex` | 2518–2532 | **Replace** | Branch: "Go template evaluating to true/false" → "GXL boolean expression" |
| `03-schema-vnext.tex` | 2615–2669 | **Replace** | Iterate: `until` → GXL; `over` → `${var}` GIS expression (Q1) |
| `03-schema-vnext.tex` | 3154–3165 | **Edit** | Tool argv: remove "Go templates" → GIS |
| `03-schema-vnext.tex` | 3317–3337 | **Edit** | Tool rendering: remove Go template references |
| `03-schema-vnext.tex` | 3530–3574 | **Edit** | Semantic validation: update to GIS variable references |
| `02-architecture.tex` | 467–477 | **Edit** | CLI interpolation → GIS |
| `02-architecture.tex` | 586–587, 1385–1412 | **Edit** | Display content → GIS |
| `06-tool-runtime.tex` | 112–117 | **Edit** | argv rendering → GIS |
| `06-tool-runtime.tex` | 282–299 | **Replace** | Capture paths → GCP formal grammar reference |

### New Sections to Create

| File | Content |
|------|---------|
| `design/gert/sections/03a-expression-language.tex` | Full GXL specification (grammar, type system, error model, namespace functions, forbidden constructs) |
| `design/gert/sections/03b-interpolation-syntax.tex` | Full GIS specification (delimiter, GDP grammar, missing-key behavior, escape, non-goals) |
| `design/gert/sections/03c-capture-paths.tex` | Full GCP specification (sources, path grammar, subtree capture semantics, `capture.default:`) |
| `design/gert/sections/03d-parse-time-enforcement.tex` | Error codes, `ValidatedPlan` contract, parser gate architecture |

---

## 5. Fixture Migration Scope

### Summary Counts

| Category | Fixture Count | Transformation Type | Automation Level |
|----------|:---:|---|---|
| `{{ .var }}` → `${var}` | **21** runbooks + **3** tools | GIS interpolation | **Mechanical** — regex/sed |
| `&&` → `and`, `||` → `or` | **4** runbooks (r01, r02, r07, r10) | GXL operator swap | **Mechanical** |
| `!x` → `not x` | **2** runbooks (r09, r10) | GXL operator swap | **Mechanical** |
| `contains` infix → `str.contains()` | **1** runbook (r01) | GXL function call rewrite | **Semi-mechanical** (need to identify subject/argument) |
| `$.services` JSONPath → `${services}` | **1** runbook (r11) | GIS + iterate.over (Q1) | **Manual review** |
| `{{ .env \| default "dev" }}` pipe/default → `vars:` default + `${env}` | **2** runbooks (r20, r21) | Structural (add `vars:` entry, simplify template) | **Manual** |
| `fromJSON` pipeline → typed capture + dotted path | **0** in current fixtures (pattern exists in spec examples only) | Structural | **Manual** (if found in future fixtures) |
| No changes needed | **2** runbooks (r12, r15) | — | — |

**Total fixtures impacted:** 22 runbook fixtures (of 22 total) + ~3 tool fixtures (of 12 total) = **25 files to touch**. Of those, **2 require no changes** and **2 require manual structural rewrite**. The remaining **21** are fully mechanical.

### Tool Fixtures

| Tool | Change |
|------|--------|
| `echo.tool.yaml` | `${GERT_TOOLS_DIR}` already uses `${}` — but this is env-var substitution, not GIS. Verify no Go template usage. |
| `jsonrpc-server.tool.yaml` | Same as echo — env-var `${}`, not GIS. Likely no change. |
| `mcp-server.tool.yaml` | Same pattern. Likely no change. |
| Remaining 9 tools | Audit for `{{ }}` patterns in `argv`, `args`, descriptions. |

---

## 6. Ordering & Critical Path

```
Week 1                    Week 2                    Week 3
─────────────────────────────────────────────────────────────────
[A: Grammar ████]
                          [B: Spec Rewrite ████████████████████████]
              [C: Conformance Corpus ████████████████████████████████████████████]
                                      [D: Fixture Migration ██████████████]
                                                            [E: Migrate Tool ██████████]
                          [F: Parser Gate Spec █████████████████████]
```

### Numbered sequence with dependencies:

1. **Stream A — Reference Grammar** (days 1–2)  
   → No dependencies. Starts immediately.

2. **Stream B — Spec Rewrite** (days 3–9)  
   → Depends on A. Can start as soon as grammar files land.

3. **Stream C — Conformance Corpus** (days 3–14)  
   → Depends on A. Starts in parallel with B. Some vectors (error classes) refine during B.

4. **Stream F — Parser Gate Spec** (days 3–7)  
   → Depends on A. Parallel with B and C. Feeds error codes into C.

5. **Stream D — Fixture Migration** (days 7–10)  
   → Depends on A + partial B (needs `capture.default:` syntax from Q2 to be written).

6. **Stream E — Migration Tooling** (days 10–14)  
   → Depends on D (manual migration patterns inform automation).

### Critical Path

**A → C → Phase 2 unlock**

The conformance corpus is the longest stream (XL) and the hard gate for Phase 2. Everything else (spec rewrite, fixture migration, tooling) is important but not on the critical path to unblocking the Go and C# runtime implementations. If C runs late, Phase 2 cannot start.

Secondary critical path: **A → B → D** (spec must be rewritten before fixtures are migrated, since D needs the `capture.default:` syntax defined in B).

---

## 7. Risks & Open Questions

### Risks

| # | Risk | Likelihood | Impact | Mitigation |
|---|------|:---:|:---:|---|
| R1 | Conformance corpus size underestimated — edge cases in null handling, Unicode, nested paths may balloon count | Medium | Medium | Start with the 235 target, accept that Phase 2 will ADD vectors as implementations surface ambiguities. The corpus is living. |
| R2 | `capture.default:` semantics (Q2) interact badly with subtree captures (Q5) — what's the default for a missing subtree? | Medium | High | Barbara must spec this explicitly in Stream B. Proposal: default must be same type as expected capture (scalar default for scalar path; no default allowed for subtree captures — author must guard with `when:`). |
| R3 | Short-circuit semantics of `and`/`or` create conformance ambiguity — does `false and missing_var` error or return false? | High | High | **Decision needed before Stream C starts.** Proposal: short-circuit (no error). Aligns with Python/SQL. Must be in grammar spec (Stream A). |
| R4 | LaTeX compilation of modified spec may break due to cross-references | Low | Low | Spec Editor agent handles this; use `latexmk` in CI. |
| R5 | Migration tool (Stream E) cannot handle all patterns mechanically — some fixtures need manual judgment | Low | Low | The tool reports what it can't transform; Don handles residuals manually. Already scoped: only 2 fixtures need manual work. |

### Open Questions (must resolve before Day 3)

| # | Question | Proposed Answer | Who Decides |
|---|----------|-----------------|-------------|
| OQ1 | Does `false and missing_var` short-circuit (no error) or evaluate both sides (error on missing)? | Short-circuit. Matches Python, SQL, JS. Reduces footgun. | Barbara proposes → ormasoftchile ratifies |
| OQ2 | Can `capture.default:` apply to subtree captures, or only scalars? | Scalars only. Subtree default is too complex to specify portably. | Barbara proposes → ormasoftchile ratifies |
| OQ3 | What is the portable JSON value model for subtree captures? (Q5 says "define it" but doesn't) | JSON RFC 8259 types: null, boolean, number (float64), string, array, object. No special date/time/binary. | Barbara specs in Stream A |
| OQ4 | Should GXL support `list.indexOf`? (Q3 mentions it but proposal §1.6 doesn't list it) | Yes — add to namespace. Returns number (-1 if not found). | Barbara adds to grammar |

### New Agents Needed

| Agent | Role | Spawned By | When |
|-------|------|-----------|------|
| **Spec Editor** | Rewrites `.tex` files per Barbara's instructions. Handles LaTeX formatting, cross-references, compilation. Strong technical writing. | Barbara | Day 2 (after grammar lands) |
| **Conformance Tester** | Writes test vectors. Methodical, adversarial (finds edge cases). Understands type systems, Unicode, null semantics. | Don | Day 3 (after grammar + error codes outlined) |
| **Parser Engineer** | Phase 2 agent (not Phase 1). Will implement GXL/GIS/GCP parsers in Go and C# from grammar files. | Barbara | Phase 2 start |

---

## 8. Phase 1 Exit Criteria

Phase 1 is complete and Phase 2 is unblocked when ALL of the following are true:

- [ ] `design/gert/grammar/gxl.ebnf` exists and validates with a standard EBNF checker
- [ ] `design/gert/grammar/gis.ebnf` exists and validates
- [ ] `design/gert/grammar/gcp.ebnf` exists and validates
- [ ] `grep -rn "expr-lang\|text/template\|fromJSON\|fromYAML" design/gert/sections/` returns 0 hits (excluding historical notes)
- [ ] `design/gert/sections/03a-expression-language.tex` exists (GXL full spec)
- [ ] `design/gert/sections/03b-interpolation-syntax.tex` exists (GIS full spec)
- [ ] `design/gert/sections/03c-capture-paths.tex` exists (GCP full spec)
- [ ] `design/gert/sections/03d-parse-time-enforcement.tex` exists (error codes + ValidatedPlan)
- [ ] `design/gert/conformance/schema.json` exists and validates all vector files
- [ ] `design/gert/conformance/` contains ≥ 200 test vectors across all categories
- [ ] Every test vector has: `id`, `input`, `variables`, `expected` (value or error_class)
- [ ] All 22 runbook fixtures contain zero `{{ }}`, zero `&&`/`||`/`!`, zero `contains` infix, zero `$.path`
- [ ] `scripts/validate-fixtures.sh` exists and exits 0 against the full fixture corpus
- [ ] Short-circuit semantics (OQ1) and subtree-default policy (OQ2) are ratified in `.squad/decisions.md`
- [ ] Barbara signs off on grammar/spec consistency
- [ ] Don signs off on conformance corpus completeness
- [ ] ormasoftchile ratifies the full package as the Phase 2 contract

**Phase 2 unlock:** Once all boxes are checked, Go and C# runtime teams receive the grammar files + conformance corpus as their implementation contract. They implement parsers and evaluators independently, targeting 100% corpus pass rate as their ship gate.

---

## Appendix: Effort Summary

| Stream | Owner | Effort | Calendar Days (est.) |
|--------|-------|:---:|:---:|
| A — Reference Grammar | Barbara | S | 2 |
| B — Spec Rewrite | Barbara + Spec Editor | L | 7 |
| C — Conformance Corpus | Don + Conformance Tester | XL | 12 |
| D — Fixture Migration | Don | M | 4 |
| E — Migration Tooling | Don | M | 4 |
| F — Parser Gate Spec | Barbara | M | 3 |
| **Total (critical path)** | | | **~14 days** |
| **Total (all streams, with parallelism)** | | | **~14 days** |

The critical path is 14 calendar days assuming Stream C (conformance corpus) starts on Day 3 and takes 12 days. Streams B, D, E, F run in parallel and finish within that window.


---

### 2026-06-07: Q2 — GCP optional chaining (`?.`) arbitration

**Decision:** Option (a) — No `?.` in GCP. GIS keeps `?.` (ratified earlier); GCP does not support it.

**Arbitrated by:** Barbara (Lead/Architect)
**Date:** 2026-06-07T15:06:00-07:00

---

#### Rationale

1. **Zero vector impact.** None of the 41 tv-gcp-path conformance vectors use `?.`. The 6 P7-blocked vectors require the capture service and step output model — not optional chaining. Adding `?.` would create new surface area with no existing test demand.

2. **Different concerns, different syntax.** GIS interpolates into strings where empty-on-miss is cosmetically safe (`"Hello, ${user?.name}"` → `"Hello, "`). GCP captures structured data into PJVM where silent empty-string substitution would mask structural errors in step outputs. The existing `capture_defaults:` mechanism provides explicit, type-checked, runbook-declared defaults — superior to implicit path-level coercion.

3. **Blast radius: zero.** No grammar change. No runtime extension. No corpus changes. No conformance vector reclassification. GNC's implementation already rejects `?.` — this decision ratifies her safe default.

4. **Cross-runtime parity.** Clean, unambiguous answer for C#/TS: "GCP paths are strict; missing segments raise GCP-RESOLVE-002/003. Use `capture_defaults:` for optional-value patterns." No ambiguity about whether `?.` means empty-string or Miss sentinel.

5. **User mental model.** Yes, `vars.user?.id` works differently in `${...}` (GIS) vs `capture.path:` (GCP). This is acceptable because they serve different purposes: interpolation vs structured extraction. The runbook schema already separates these syntactically (`string templates` vs `capture: map entries`), so the contexts are unambiguous.

#### Spec changes made

Added normative subsection `§subsec:gcp:traversal:no-optional-chaining` to `03c-capture-paths.tex` documenting:
- `?.` MUST be rejected in GCP paths (GCP-PARSE-005)
- Rationale for the asymmetry with GIS
- Correct pattern using `capture_defaults:`

#### P7-blocked vectors (unchanged)

The 6 P7-blocked vectors depend on capture service infrastructure (plan-time validation of step IDs, default-policy enforcement, type checking) — none require `?.` syntax. They remain P7-blocked regardless of this decision.

#### Follow-up needed

None. No grammar changes, no runtime extension, no corpus modifications.


---

### 2026-06-07: Q1 — Missing GDP resolution section (03d-gdp.tex)

**Decision:** Option (B) — Do not create a separate `03d-gdp.tex`. GDP resolution semantics remain where they already live.

**Arbitrated by:** Barbara (Lead/Architect)
**Date:** 2026-06-07T15:06:00-07:00

---

#### Rationale

The "missing 03d-gdp.tex" was a misidentification. The file `03d-parse-time-enforcement.tex` already exists and is correctly referenced by the runtime plan for parse-gate semantics — not for GDP resolution. GDP resolution semantics are already fully specified across two locations:

1. **`03a-expression-language.tex` §subsec:gxl:gdp** — defines the GDP grammar (`GDP = IDENT { "." IDENT | "[" INTEGER "]" }`) and variable-access traversal rules.
2. **`03c-capture-paths.tex` §sec:gcp:traversal** — defines GDP path traversal in the GCP context (source-qualified), including error codes for missing keys (GCP-RESOLVE-002) and out-of-bounds indices (GCP-RESOLVE-003).

Section 03c explicitly states: "GDP itself is defined in §subsec:gxl:gdp and is referenced here; it is not duplicated." GNC built her resolver successfully from these two sections plus the grammar — confirming they are sufficient. Creating a third location would introduce maintenance burden and duplication risk without adding new normative content.

#### Cross-runtime impact

Zero. C#/TS implementers follow the same two sections. No references need updating — the runtime plan's `§03d` citations correctly point at parse-time enforcement.

#### What changed

No spec files created or removed. This decision confirms the status quo and closes the question.


---

# Decision: Runtime Implementation Plan (Greenfield) — v2

**Author:** Barbara — Lead / Architect  
**Date:** 2026-06-07 (v2)  
**Status:** PROPOSED  
**Ref:** `design/gert/proposals/runtime-implementation-plan.md`  
**Input:** Rubber-duck plan critique (2026-06-07) — 5 blocking + 3 non-blocking findings. Brady approved bundling corpus/spec fixes into this revision as a formal P0 phase.

---

## Context

Post-merge audit revealed the `ormasoftchile/gert` runtime does NOT implement GXL/GIS/GCP. The four engines are entirely missing. Phase 2 is greenfield, not maintenance. The prior migration plan (`runtime-migration-plan.md`, ratified 2026-06-05) is superseded.

The v1 implementation plan (same date) was reviewed by rubber-duck, which identified 5 blocking and 3 non-blocking issues. This v2 addresses all 8 findings.

---

## Decisions

### 1. Build Order: P0 → P1 → P2 → P3 ∥ (GCP parser + GDP resolver) → P4 → P5 → P6 → P7 → P8

**Rationale:**

- **P0 (corpus hygiene) must precede P1** because the conformance harness cannot accept vectors that fail YAML lint or schema validation. Without trustworthy inputs, no acceptance gate is meaningful. Brady approved this bundling.
- **Parser (P2) before evaluator (P3)** because the evaluator walks an AST the parser produces.
- **GXL evaluator (P3) before GIS interpolator (P4)** because `gis.ebnf` §2.4 requires full GXL expressions inside `${...}`. The interpolator delegates to the evaluator.
- **GCP parser + GDP resolver can parallelize with P3** because they need only PJVM (P1) and the GDP path grammar, not the full evaluator. However, the full GCP capture service (P7) CANNOT parallelize with P3 — it requires the engine-selection seam (P6), step output model, and executor refactoring that touches production code.
- **Parse-gate (P5) before runtime routing (P6)** because the runtime routing delivers `ValidatedPlan` to executors. Cannot route without the gate.
- **Full GCP integration (P7) after staged routing (P6)** because capture logic touches 5+ executor types. Must use the new routing seam, not the old path.
- **expr-lang deletion (P8) last** because it requires proven production exercise + soak.

**v1 → v2 change:** v1 had GCP as a 2-unit phase parallel with P3 evaluator. This was under-scoped. GCP is now split: parser+GDP resolver (3–4 units, parallelizable) and capture service + executor refactor (6–10 units, sequential after P6). v1 also omitted P0 and P5 entirely.

### 2. Strategy: Side-by-side with staged runtime routing (engine-selection seam)

**Rationale:**

v1 proposed build-tag-only side-by-side: new engines exercised ONLY by the conformance harness until a big-bang cutover in P6 flipped 12 executors + capture logic + tests + dependency deletion together. The rubber-duck critique (finding #2) identified this as a "test one thing, ship another" trap — the production path was never exercised under the new engines until the moment of cutover.

**v2 strategy:** Keep the build tag for CI matrix coverage, but add an internal `EngineRouter` that enables per-subsystem cutover:
- **P6a:** Route GXL condition evaluation through new engines. Old path remains for interpolation/capture.
- **P6b:** Route GIS interpolation through new engines. Old path remains for capture.
- **P6c:** Route GCP capture through new engines, one executor family at a time.

Each P6 sub-PR exercises the new path through production routing. CI runs both build paths. expr-lang is not deleted until P8, after the soak proves the new path under real routing.

**Rejected alternative (v1 approach):** Build-tag-only with big-bang cutover. This was the v1 decision. It is now **rejected** because:
- Production code never exercises new engines until cutover day
- Bisecting failures across 12 simultaneous executor flips is painful
- The conformance harness passing does not prove production wiring works

**Rejected: Replace-in-place.** Big-bang risk on 12 executors + 58 tests.

**Rejected: Runtime feature flag.** Overhead, hot-path branching, indefinite dual-path CI.

**Mitigation for staged routing complexity:** The `EngineRouter` is internal wiring (~100 lines), not a public API. It is deleted in P8 when new engines become the only path. The added complexity is temporary and bounded.

### 3. Parse-Gate / ValidatedPlan: Mandatory phase

**Rationale:**

v1 marked parse-gate implementation (OPQ-GATE-01..07) as "out of scope — separate proposal cycle." The rubber-duck critique (finding #1) identified this as wrong: the spec (`design/gert/sections/03d-parse-time-enforcement.tex`) mandates that execution is blocked from unvalidated plans. 267/267 on engine APIs would not prove conformance if the planner still lazy-parses at execution time.

**Decision:** Parse-gate is P5 — a mandatory phase. It is NOT deferred to a separate proposal. The implementation:
- Extends `internal/planner/planner.go` with a validation pipeline per §03d
- Produces an unexported `validatedPlan` type (no external construction possible)
- Emits `plan.validated` event with grammar versions, expression counts, runbook hash
- Blocks `RunHandle.Next()` from unvalidated plans
- Adds integration tests proving invalid expressions fail BEFORE any step dispatches

**Architectural impact:** This touches the existing planner package in `ormasoftchile/gert`. The current `Plan()` function returns `*engine.ExecutionPlan` directly. P5 wraps this with validation. The planner files involved: `planner.go` (existing), `validate.go` (new), `validated_plan.go` (new), `validate_test.go` (new).

### 4. GCP Scope: Full capture system, not just GDP resolver

**Rationale:**

v1 budgeted 2 units for "GCP resolver (GDP paths on PJVM trees)." The rubber-duck critique (finding #4) identified this as materially under-scoped. Real GCP requires: source prefixes (`http.*`, `event.*`, `step.*`), local/cross-step output lookup, JSON/YAML parsing, header semantics, bare-root capture, scalar vs subtree classification, `capture_defaults`, default/subtree policy errors. Current gert capture logic is scattered across CLI/tool/include/noop executors with keyword switches.

**Decision:** GCP is split:
- **GCP parser + GDP resolver** (3–4 units): Pure parsing. Can parallelize with P3 evaluator.
- **Capture service + executor refactor** (6–10 units): Production wiring. Sequential after P6 (requires engine-selection seam). This is P7.

The shared GDP resolver (`internal/gdp/`) is extracted as a first-class shared package from the start — not "extract later if duplication exceeds 100 lines" (v1 risk #5). This is a design decision, not a contingency.

### 5. Cutover Gate: Defined methodology

**Rationale:**

v1 specified "≤ 2× perf" and "48h soak" without defining benchmark command, dataset, thresholds, or soak workload. The rubber-duck critique (finding #7) flagged this as unactionable.

**Decision:** Cutover gate (P8) requires:

| Criterion | Methodology |
|---|---|
| **Benchmark** | `go test -bench=BenchmarkEngine -benchmem -count=5 ./internal/conformance/...` against 267 vectors + 22 fixture runbooks. Baseline: last commit before P6a. |
| **GXL parse/eval** | ≤ 2× baseline ns/op |
| **GIS interpolation** | ≤ 2× baseline ns/op |
| **GCP resolution** | ≤ 3× baseline ns/op (new capability) |
| **Full fixture run** | ≤ 2× baseline wall-clock |
| **Allocation cap** | ≤ 1.5× allocs/op, ≤ 2× bytes/op vs baseline |
| **Race-free** | `go test -race ./...` with zero data races, zero panics |
| **Soak** | Scheduled CI job: full fixture + conformance suite every 2 hours for 48 consecutive hours on a dedicated soak branch. Zero failures across all runs. NOT passive dev-branch time. |

### 6. Honest Effort Estimate: 32–49 critical-path work units

v1 estimated 20 work units. This was optimistic (finding #8). The honest estimate:

| Phase | v1 | v2 | Delta reason |
|---|---|---|---|
| P0 (corpus) | — | 2–4 | New phase (finding #3) |
| P1 (PJVM) | 3 | 4–5 | Normative number model, conversion rules (finding #6) |
| P2 (GXL parser) | 3 | 5–7 | Realistic for recursive descent + operator precedence |
| P3 (GXL eval) | 5 | 5–7 | Stdlib breadth |
| GCP parser+GDP | (2 in P4) | 3–4 | Split from old P4 |
| P4 (GIS) | 3 | 4–6 | Corrected escape, GXL delegation, per-segment ?. (finding #5) |
| P5 (parse-gate) | — | 3–5 | New phase (finding #1) |
| P6 (routing) | 4 | 3–5 | Staged routing (finding #2) |
| P7 (GCP full) | — | 6–10 | Split from old P4 (finding #4) |
| P8 (perf/soak) | (in P6) | 3–5 | Defined methodology (finding #7) |
| **Total** | **20** | **32–49** | |

With 2 engineers and P3 parallelism: ~25–38 calendar work-days (5–8 weeks with buffer).

---

## Blast Radius

- **P0:** Changes gert-private only (corpus, schema, spec). Zero runtime impact.
- **P1–P4:** Pure additions to `ormasoftchile/gert`. No production impact. New packages only.
- **P5:** Modifies `internal/planner/planner.go` — adds validation pipeline. The `Plan()` function now returns a validated result. Internal change; public API unchanged.
- **P6:** Introduces engine-selection seam. Each sub-PR (P6a/P6b/P6c) touches specific executor wiring. Staged — not all at once.
- **P7:** Refactors capture logic across 5+ executor types. This is the highest-risk phase after P6 in terms of blast radius.
- **P8:** Deletes `internal/expr/`, drops `expr-lang/expr` from `go.mod`, removes build tag and `EngineRouter`. Irreversible once merged.

## Open Items Carried Forward

- OQ-GATE-01 (in-flight grammar upgrades) — deferred; separate proposal before Day 9
- TESS-CONFLICT-2 (str.unknownMethod) — needs arbitration before P3
- OI-GXL-03 (negative modulo) — needs arbitration before P3

## 2026-06-07 — Three open questions resolved

Brady answered three critical questions during v2 preparation. These resolutions are applied to the plan and memo below.

### Q1 Resolution: P0 Split

**Question:** Who owns which parts of P0? Corpus YAML + schema + Make target, vs spec prose?

**Brady's Answer (2026-06-07):** Split. Tess executes the corpus data work (YAML linting, schema.json reconciliation, `make verify-corpus`). Barbara executes the spec contract piece (section 09 subsection). They run in parallel (this spawn IS that parallel execution; Tess runs `tess/p0-corpus-hygiene` while Barbara runs this task).

**Rationale:** Domain ownership + throughput. Tess owns data quality; Barbara owns prose/spec. Parallel execution halves P0 calendar time.

**Application:** Plan §C.0 and §E P0 updated to reflect the split. Both sub-tasks land in the same PR closing gert-private#1, but they execute independently. Acceptance remains: `make verify-corpus` passes + corpus interpretation contract lands in section 09 + all 267 vectors validate.

---

### Q2 Resolution: Soak Infrastructure Decision

**Question:** Which infrastructure pattern for the soak workload — scheduled CI capacity, or dedicated runner? How many hours, what frequency?

**Brady's Answer (2026-06-07):** DEFERRED to post-P7. P8 acceptance criteria stay (per-category performance thresholds, race-free, soak methodology). The *infrastructure choice* (capacity vs runner, 2h vs 4h frequency, 48h vs 72h duration) is parked pending runtime eng assessment AFTER all previous phases complete.

**Rationale:** Runtime infrastructure decisions depend on production capacity, load patterns, and cost-effectiveness. Brady wants to decide based on empirical data from P6/P7, not speculation now.

**Application:** Plan §E P8 updated. Soak acceptance methodology (per-category perf thresholds + race-free + zero failures across scheduled runs) is firm. Soak infrastructure choice is marked DEFERRED. P8 begins with a placeholder workload (22-fixture suite + 267-vector suite); runtime engineers can swap infra + workload when the decision lands without changing acceptance criteria.

---

### Q3 Resolution: P7 Executor Refactor Scope — Fix, not Preserve

**Question:** When P7 implementers find capture-logic divergences from the GCP spec in the existing 5 executor types (cli, tool, http_call, include, noop), what's the policy — preserve backwards compat, or fix to spec?

**Brady's Answer (2026-06-07):** FIX. Document-and-fix. When divergences are found between current capture logic and the GCP spec, bring behavior into spec conformance. The GCP spec is the contract. Backwards compatibility with existing accidental behavior is NOT a constraint. (User directive — preserves the conformance-as-contract principle.)

**Rationale:** Spec is the source of truth. Executors have accumulated ad-hoc behavior over time. Alignment to spec during P7 is the right time to fix them. Backwards compat would lock in the drift permanently.

**Application:** Plan §E P7 updated with explicit policy statement + discovery contract: P7 implementers MUST log every capture-logic divergence (what was, what spec says, what changed) in PR body with `P7-divergence:` tag. The fix is mandatory; the spec is the contract. This prevents silent behavior changes and keeps the audit trail.

---

## Action

Awaiting Brady's resource commitment. If approved:
- P0 can begin immediately in `ormasoftchile/gert-private` (closes #1)
- P1 begins in `ormasoftchile/gert` only AFTER P0 PR merges


---

# Decision: Runtime Migration Plan — Headline Picks

**Author:** Barbara — Lead / Architect  
**Date:** 2026-06-05T18:33:34-07:00  
**Status:** PROPOSED — awaiting ormasoftchile review  
**Document:** `design/gert/proposals/runtime-migration-plan.md`

---

## Headline Picks

| # | Decision | Value |
|---|----------|-------|
| 1 | Package strategy | **Parallel-package** (`internal/eval/`) alongside existing `internal/expr/`; adapter satisfies existing interfaces; old code deleted at cutover |
| 2 | Phase ordering | **A→B→C→D→G→H** (critical path); E and F parallelizable |
| 3 | Feature flag | **Build tag** (`//go:build gxl`) for zero-overhead compile-time switch during migration |
| 4 | Cutover criteria | 264/264 vectors green + 22 fixtures execute + perf within 2x + 1 week soak |
| 5 | Dependency drop timing | **Phase H only** — after soak period; revert = one commit |
| 6 | Sketch reuse | **Yes** — cherry-pick 97ce48b..5c550c0 as starting material (not drop-in) |
| 7 | Conformance distribution | **Git submodule** (pending OQ-M1 on repo visibility) |
| 8 | Estimated total effort | 12-18 days serial; ~12-15 days with 2 engineers; ~10-12 with sketch reuse |

## Key Risks

1. **R1 (HIGH):** Semantic divergence — expr-lang `contains` infix and implicit coercion are FORBIDDEN in GXL. Existing runbooks/tests that use these break.
2. **R4 (HIGH):** Unresolved spec ambiguities (TESS-AMBIG-3/4, OI-GXL-03) will block implementation until resolved in gert-private.
3. **R5 (HIGH):** Capture path upgrade requires structural output wrapping — CLI executor's keyword-switch must become GDP path resolution.

## Open Questions (Require ormasoftchile Input)

- OQ-M1: Conformance vector distribution mechanism (submodule vs. published artifact)
- OQ-M2: Confirm sketch cherry-pick approach
- OQ-M3: Build tag vs. runtime flag for migration period
- OQ-M4: Breaking syntax change communication strategy
- OQ-M5: Team size (1 vs. 2 engineers)


---

## Truthy() Arbitration for Empty Collections — Decision Record

**Date:** 2026-06-07
**Author:** Barbara (Lead/Architect)
**Requested by:** Coordinator (relaying Booster, P1 PJVM)
**Status:** RATIFIED

---

### Question

PJVM `Truthy()` is called by GXL boolean coercion in contexts like `if x:` or `${x ? a : b}`. Spec section 03a says "no implicit truthiness" but does not explicitly pin down the behavior for every PJVM type (especially empty collections `[]` and `{}`) when they end up in a boolean context. Booster's P1 PJVM picked JS-style truthiness to make tests pass; without a normative answer, P3's GXL evaluator would lock in a de-facto behavior that C#/TS runtimes would have to reverse-engineer.

### Options Considered

- **(a) JS-style:** empty array/object → truthy. Only canonical falsy set (`false`, `null`, `0`, `""`) is falsy.
- **(b) Python-style:** empty array/object → falsy. Empty collections join the falsy set.
- **(c) Strict:** non-bool value in boolean context → `GXL-TYPE-002` error. No implicit coercion at all.

### Decision

**Option (c) — Strict.** A value in boolean context MUST have PJVM type `bool`. Any non-bool value raises `GXL-TYPE-002`. There is no `Truthy()` coercion function.

### Rationale

The spec already mandates this. Section 03a states "Authors MUST NOT rely on implicit truthiness" and `gxl.ebnf` §5.2 states "A non-bool value in boolean position → GXL-TYPE-002." Choosing (a) or (b) would contradict existing normative text and require a spec *change*, not a clarification. Option (c) confirms what the spec already says.

From a cross-runtime parity perspective, strict typing is trivially portable: every runtime checks `type == bool` and raises `GXL-TYPE-002` otherwise. Options (a) and (b) each require every runtime to implement an identical truthiness table — a needless divergence surface. The blast radius of (c) is zero: it changes no existing behavior, adds no new coercion paths, and leaves the type system closed. Users who want emptiness checks use explicit idioms (`len(x) > 0`, `x != null`, `n != 0`, `s != ""`), which are self-documenting and unambiguous.

The "is this list empty?" idiom (`if myList:`) is common in Python/JS, but GXL is not those languages. GXL's value proposition is deterministic, auditable evaluation — implicit coercion undermines that. The explicit `len(myList) == 0` form is four characters longer and infinitely clearer in an audit trail.

### Cross-Runtime Implication

All GXL runtime implementations (Go, C#, TypeScript) MUST enforce the strict boolean-context gate identically: only `bool` values pass; all other PJVM types trigger `GXL-TYPE-002` — no runtime may introduce its own truthy/falsy table.

### Spec Section Updated

`design/gert/sections/03a-expression-language.tex` — new subsection "Truthy Coercion in Boolean Context" (§`sec:gxl:semantics:truthy`) with normative rule, worked examples for all PJVM types, recommended idioms, and cross-runtime note.

### Grammar

`design/gert/grammar/gxl.ebnf` — no change needed. Boolean context is a runtime evaluation concept, not a grammar-level production. The EBNF already documents the rule in §5.2 commentary ("A non-bool value in boolean position → GXL-TYPE-002"). The grammar defines where boolean expressions appear syntactically (after `not`, both sides of `and`/`or`); the type check is enforced at eval time, not parse time.

### Corpus Vectors Added

See [barbara-truthy-corpus-vectors.md](barbara-truthy-corpus-vectors.md) — ~14 new conformance vectors for Tess to add to `tv-gxl-eval.yaml`.

### Impact on Booster's P1 PJVM

If Booster's P1 PJVM implements a `Truthy()` function with JS-style semantics, it must be corrected: the function should accept only `bool` and return the bool value directly, or (preferably) the evaluator should perform the type check inline without a separate `Truthy()` method. Either way, the runtime MUST raise `GXL-TYPE-002` for non-bool inputs in boolean context.


---

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


---

RESOLVED 2026-06-07 — see barbara-gdp-section-03d.md and barbara-gcp-optional-chaining.md

### 2026-06-07T19-40-00Z: Two GCP spec questions surfaced by GNC (P3 parallel work)

**By:** Coordinator (relaying GNC from gert#26)

**Context:** GNC built the GCP parser + GDP resolver and ran the 41 tv-gcp-path vectors. Result: 35 runnable, 6 explicitly P7-skipped. During implementation, two spec ambiguities surfaced.

---

#### Question 1 — Missing spec section: `design/gert/sections/03d-gdp.tex`

The runtime implementation plan and Barbara's section 09 contract both reference section 03d (GDP — GERT Dotted Path resolution semantics) as the source of truth for path resolution rules. GNC searched and **the file does not exist** in the repo. She inferred GDP semantics from `gcp.ebnf`, `03c-paths.tex`, and the conformance vectors themselves.

**Action needed:** Barbara (or Edith if spec prose) to either:
- Write `03d-gdp.tex` formalizing the GDP resolution rules GNC inferred (and that the 36/36 tv-gxl-path passes prove correct), OR
- Confirm that GDP semantics live entirely in `03c-paths.tex` and update the plan + section 09 contract to stop referencing `03d`.

Either way, the gap should be closed — a runtime implementer reading the plan today would hit the same wall.

---

#### Question 2 — GCP optional chaining (`?.`) conflicts with spec/vectors

GNC found that GCP optional chaining is undefined in the current spec/grammar, and several vectors that LOOK like they want `?.` GCP behavior actually trigger inconsistent results. She **rejected `?.` in the GCP parser** as the safe default — vectors using GCP `?.` now fail (or fall into the 6 P7-skipped bucket if they require capture service).

**Three options for Barbara to arbitrate:**

- (a) **No `?.` in GCP.** GIS keeps `?.` (Brady's earlier ratification), GCP doesn't. Capture defaults handled via `capture.default:` in the runbook, not via path syntax. Document that GIS `?.` and GCP path semantics are intentionally different.
- (b) **Add `?.` to GCP**, mirror GIS semantics (miss → empty-string, JS-style short-circuit). Update grammar, update spec section 03c (or 03d), update affected vectors. GNC would need a follow-up PR to extend her parser + resolver.
- (c) **Add `?.` to GCP but with GCP-specific semantics** (miss → typed Miss sentinel that capture machinery interprets in P7). More work but possibly cleaner for capture defaults.

**Recommendation from GNC (paraphrased):** Option (a) is the safest current default and aligns with how the existing vectors actually behave. If we want `?.` in GCP, it needs an explicit normative decision and grammar update.

**Impact if option (b) or (c):** A few vectors currently in the "35 passing" or "6 P7-skipped" buckets may move; runtime needs a follow-up extension.

---

**Neither question blocks P3 GXL evaluator** (FAO is being dispatched now). They block the next round of GCP corpus completion and the `03d-gdp.tex` cleanup.

**Action items:** Barbara to arbitrate Q2 and decide on Q1's path (write the missing section or amend the plan references) when next active.


---

**RESOLVED — see [barbara-truthy-arbitration.md](barbara-truthy-arbitration.md) (2026-06-07, Option c: Strict).**

### 2026-06-07T19-00-00Z — Truthy() semantics for empty arrays/objects (needs arbitration)

**By:** Coordinator (relaying Booster from gert#24 P1 work)

**What:** PJVM `Truthy()` is called by GXL boolean coercion (e.g., `if x:`). Section 03a says no implicit truthiness, but empty array `[]` and empty object `{}` need a deterministic answer because the GXL evaluator (lands in P3) will call `Truthy()` on whatever the user passes to a boolean context. Booster's P1 implementation picked one in code to keep PJVM testable, but the choice is not pinned down in the spec.

**Options:**
- (a) **JS-style:** empty array/object → truthy. Only the canonical falsy set (`false`, `null`, `0`, `""`) is falsy.
- (b) **Python-style:** empty array/object → falsy. Falsy set expands to include empty collections.
- (c) **Strict:** any non-bool value passed to a boolean context → parse-gate error OR runtime error. No implicit coercion at all.

**Why it matters now:** P3 GXL evaluator cannot land without this answer. Cross-runtime parity for C# / TS later depends on a single answer that ships in the corpus.

**Recommendation from Booster:** Option (a) — JS-style. Predictable, matches what most users will expect from `${someList ? "yes" : "no"}` style expressions, and is the dominant convention in adjacent expression languages.

**Action items:**
1. Barbara (or Brady) arbitrate Option (a) / (b) / (c)
2. Tess add conformance vectors covering empty `[]` and `{}` in boolean context once decided
3. Update section 03a (or equivalent) to make the rule normative
4. Block P3 (GXL evaluator) until resolved — without this, the evaluator has no defensible answer for `Truthy([])`


---

---
author: david
date: 2026-06-07T19-07-00-07-00
slug: ntfs-safe-filenames-policy
status: proposed
---

# Decision: All GERT-family repos must enforce colon-free filenames

## Context

On 2026-06-07, a fresh clone of `gert-tui` on Windows failed completely because 19 historical Scribe log files contained raw ISO-8601 colons in their filenames (e.g., `2026-04-29T01:00:25Z-scaffold-session.md`). Windows NTFS forbids `:` in path components. `git checkout` silently wiped the entire working tree, leaving the repo unusable until a GitHub API tree-rewrite was performed.

This is the **third** recorded occurrence of this class of bug across GERT-family repos (previous instances documented in `gert-private/.squad/identity/wisdom.md`). The pattern is always the same: Scribe or an agent generates a log file with an unescaped timestamp and commits it without validation.

## Decision

1. **Filename convention (non-negotiable):** All files committed to any GERT-family repository must use NTFS-safe characters only. For timestamp segments in filenames, use `T\d\d-\d\d-\d\dZ` (hyphens), never `T\d\d:\d\d:\d\dZ` (colons). Also forbidden: `<`, `>`, `"`, `|`, `?`, `*`, trailing `.` or space.

2. **Pre-commit hook (all repos):** Add a pre-commit hook that rejects any staged path containing `:`, `<`, `>`, `"`, `|`, `?`, or `*`. Sample check:
   ```sh
   git diff --cached --name-only | grep -P '[:<>"|?*]' && echo "ERROR: Windows-forbidden char in filename" && exit 1
   ```

3. **`.gitattributes` guard:** Each repo should have:
   ```
   * text=auto eol=lf
   ```
   to prevent CRLF drift, but this does not block bad filenames — the pre-commit hook is the primary gate.

4. **Agent generation rule:** Any agent (Scribe, Coordinator, or domain agent) that generates filenames with timestamps must apply the substitution `s/:/-/g` at generation time, not as a post-hoc fix. This is already documented in `wisdom.md`; this decision makes it a ratifiable team contract.

## Impact

- Affects: gert-private, gert-tui, gert (runtime), and any future GERT-family repo
- Owner: Coordinator to propagate hook to all active repos; each agent charter to reference this rule
- Urgency: High — every Windows clone is at risk until hooks are in place


---

# Don Phase 2 Day 4 — GXL Evaluator

Date: 2026-06-05T17:57:33.200-07:00

## Evaluator Entry + Walk Strategy

Implemented `gxl.Eval(node Node, bindings map[string]core.Value, clock core.Clock) (core.Value, error)`. The evaluator walks the Day-3 AST directly with strict PJVM typing: literals construct PJVM values, paths resolve from bindings, unary/binary nodes enforce operator semantics, calls dispatch to closed stdlib namespaces, and `and`/`or` perform AST-level short-circuit evaluation.

## Stdlib Implemented

- `len`: strings by rune count; lists by element count.
- `now`: zero-argument top-level builtin using injected `core.Clock`, UTC `YYYY-MM-DDTHH:MM:SSZ`.
- `str`: `contains`, `startsWith`, `endsWith`, `toLower`, `toUpper`, `trim`, `length`, `trimPrefix`, `trimSuffix`.
- `list`: `contains`, `indexOf`, `length`.
- `regex`: `match` with Go RE2-compatible `regexp` compilation per call; invalid patterns return `GXL-EVAL-003`.
- `math`: no functions implemented because `gxl.ebnf` reserves `math` for v2 and current vectors exercise none.

## Vector Pass Count

Expected `tv-gxl-eval.yaml`: 90 of 90 green after coordinator runs Go. Local verification is blocked because `go` is not on PATH in this environment.

## Flagged for Edith/Barbara

- `TV-GXL-EVAL-033` / `TESS-AMBIG-3`: bool ordered comparison still carries `error_code: TBD`; evaluator returns `TBD` to keep the ambiguity visible.
- `TV-GXL-EVAL-086` / `TESS-AMBIG-4`: array/list equality still carries `error_code: TBD`; evaluator returns `TBD` rather than inventing deep equality.
- `TV-GXL-EVAL-088`: corpus still specifies regex assertion for `now()` while Phase 2 Q2 ratified injected-clock exact assertions. Harness supports regex now and fixed-clock fields if the corpus is updated.

## Day 5 Readiness

Day 5 GIS/GXL integration is unblocked: GIS can parse embedded expressions with `gxl.Parse` and call `gxl.Eval(ast, bindings, clock)` directly. GDP traversal currently exists inside the evaluator for Day 4 argument/root resolution; Day 5 can harden path-specific runner coverage without changing the Eval entry signature.


---

# Decision Needed: Extensions Must Consume a Published `runbook/v1` JSON Schema

**Raised by:** Leslie (Frontend Dev)  
**Date:** 2026-06-07T19:07:00-07:00  
**Context:** gert-vscode extension audit (see `.squad/agents/leslie/gert-vscode-audit.md`, Open Question #1)

---

## Background

During the gert-vscode audit I found that the extension provides **no YAML validation** for `.runbook.yaml` files, and there is no centrally published machine-readable schema for `runbook/v1`. If we add validation to the extension ad-hoc (hand-authoring a JSON Schema locally in the extension repo), we will have at least three consumers that each maintain their own version:

1. The VS Code extension (`contributes.yamlValidation`)
2. Any CI lint step on runbook repos
3. The web portal's runbook editor/step renderer (if it ever gets an authoring mode)

These will diverge. They've already diverged once (the P0–P8 migration collapsed v2 framing back into v1, and any stale copy of the schema would still reference a v2 structure).

## Proposal

**The `gert` repo should publish a canonical `runbook.v1.schema.json`** as a versioned artifact — either committed to the repo and referenced by a stable URL, or generated at build time from the Go struct definitions and exported as a release asset.

All tooling (extension, CI, portal) should reference this single artifact, **not** maintain local copies.

## Decisions Needed

1. **Who owns the schema?** Proposed: Barbara (runtime architect). She owns the YAML structure definition; she should also own its machine-readable expression.

2. **What format?** JSON Schema Draft-07 (compatible with VS Code's YAML language server and `ajv`).

3. **How is it versioned and distributed?** Options:
   - Committed at `gert/schemas/runbook.v1.schema.json` (stable path, easy to reference via raw GitHub URL)
   - Published as a GitHub release asset
   - Hosted on a stable URL (e.g., `https://schema.gert.run/runbook/v1`)

4. **Should `capture_defaults:` values accept GIS expressions or only literals?** This is a schema precision question that should be resolved before the schema is authored.

## Impact if Not Decided

The gert-vscode Phase 2 work (YAML schema + snippets) cannot start until this is resolved — or it will start with a local hand-authored schema that becomes a liability the moment the canonical schema diverges.

## Suggested Resolution Timeline

Before gert-vscode Phase 2 begins (i.e., before the grammar work from Phase 1 is merged).

---

## DECISIONS MERGED FROM INBOX (2026-06-07T19:28:42Z)

# Decision: Canonical Home for `runbook/v1` JSON Schema

**Author:** Barbara (Lead / Architect)  
**Date:** 2026-06-07T19:14:39-07:00  
**Status:** DECIDED  
**Triggered by:** Leslie's gert-vscode audit (F-06, Open Question #1)

---

## Context

The gert-vscode extension needs a JSON Schema for `*.runbook.yaml` to power IDE features (validation, hover, completions). Three candidates for canonical home:

- **(A) gert-private/design/gert/** — alongside normative grammars. Schema generated from spec/EBNF.
- **(B) gert (Go runtime) `pkg/schema/`** — Go structs are source of truth; JSON Schema generated from them.
- **(C) gert-vscode** — ad-hoc schema in the extension itself.

Additionally considered:
- **(D) Separate repo `gert-schemas/`** — schema lives alone.

### Trade-off Matrix

| Criterion | A (design repo) | B (Go runtime) | C (extension) | D (separate repo) |
|---|---|---|---|---|
| Single source of truth | ✅ Spec IS truth | ⚠️ Go structs may drift from spec | ❌ Guaranteed drift | ⚠️ Adds coordination |
| Language portability (C#/TS ports) | ✅ Language-neutral | ❌ Go-specific; ports must reverse-engineer | ❌ | ✅ Language-neutral |
| Drift risk | Low (spec changes → schema regeneration in same PR) | Medium (spec changes in gert-private, structs updated in gert separately) | High (three repos to sync) | Medium (one more repo to coordinate) |
| Generation tooling | Needs authoring (LaTeX/EBNF → JSON Schema is non-trivial) | Mature (`invopop/jsonschema`, struct tags exist) | N/A (hand-authored) | Same as A or B depending on source |
| Operational simplicity | ✅ Already the spec authority repo | ✅ Runtime already parses YAML | ❌ | ❌ Repo sprawl |
| Who edits when grammar changes | Same author (Edith/Barbara) in same PR | Requires cross-repo PR | Requires third-party update | Cross-repo PR |

### Key Observations

1. **gert-private is the normative spec authority** (ratified 2026-06-05). All runtimes implement against it.
2. The Go `pkg/schema/` structs are an *implementation* of the spec — they are a consumer, not the source. When the C# port arrives, it will have its own structs. The schema must serve all of them.
3. The existing `design/gert/conformance/schema.json` is a **conformance vector schema** (validates `tv-*.yaml` files) — it is NOT a runbook schema. It is a sibling, not a candidate to repurpose.
4. gert-private already hosts all normative artifacts: grammars, LaTeX sections, conformance vectors, proposals. A runbook schema is a natural addition.
5. The `DESIGN ONLY` directive (2026-06-05) explicitly permits "grammar files, spec sections, conformance corpora" — a JSON Schema describing the runbook format is a design artifact, not runtime code.

---

## Decision

**Option A — `gert-private/design/gert/schemas/runbook.v1.schema.json`** is the canonical home.

The runbook JSON Schema is a **normative design artifact**, hand-authored and maintained in this repo alongside the spec sections that define its semantics. It is the single source of truth that all consumers (IDE extensions, CI linters, web portals, runtime schema loaders) reference.

---

## Publish/Consume Mechanics

### Authoring

- **File path:** `design/gert/schemas/runbook.v1.schema.json`
- **Method:** Hand-authored JSON Schema (draft 2020-12), informed by the Go struct surface (`pkg/schema/runbook.go`, `step.go`) as a reference implementation, but normatively governed by the spec sections (§02 runbook structure, §03a–d expression grammar, §09 testing).
- **Rationale for hand-authored over generated:** LaTeX→JSON Schema generation tooling does not exist and would be a bespoke project. Go struct→JSON Schema generation (`invopop/jsonschema`) produces a Go-biased artifact (Go type names, Go-specific omitempty semantics). A hand-authored schema can precisely encode YAML-specific constraints (pattern properties, conditional sub-schemas per step type, GXL expression string patterns) that no struct-tag generator would emit correctly.

### Publishing

1. **Primary:** Committed in `gert-private` at the path above. Consumers fetch via git (submodule, sparse checkout, or raw URL from a tagged release).
2. **Secondary (future):** Submit to [SchemaStore.org](https://www.schemastore.org/) once the schema stabilizes (post-v1.0 release). This gives free IDE support without any extension dependency.
3. **Tertiary (future):** Publish as an npm package (`@gert/schemas`) if the ecosystem warrants it.

### Version Contract

- The schema's `$id` tracks the runbook API version literally: `https://gert.dev/schemas/runbook/v1`.
- The schema file itself does NOT carry independent semver. It evolves in lockstep with `runbook/v1`. If `runbook/v2` is ever introduced, a new file `runbook.v2.schema.json` is created.
- Breaking schema changes (new required fields, removed fields) require a spec section update in the same PR — enforced by review gate.

### Ownership

- **Primary owner:** Barbara (architecture) — gates structural changes.
- **Day-to-day maintenance:** Edith (spec editor) updates the schema when spec sections change.
- **Review gate:** Any PR touching `design/gert/schemas/` requires Barbara approval.

---

## Consumer Rules

### gert-vscode

- **Mechanism:** Vendored copy at `gert-vscode/schemas/runbook.v1.schema.json`.
- **Sync:** CI job (`scripts/sync-schema.sh` or equivalent) fetches the schema from `gert-private` at a pinned git tag/SHA, diffs against vendored copy, fails on drift. Same pattern as DRIFT-DETECTION-001 for conformance vectors.
- **Registration:** `package.json` → `contributes.yamlValidation` points to the vendored copy. Requires Red Hat YAML extension as peer dependency OR VS Code's built-in `yaml.schemas` setting.

### gert-tui

- **Same schema, same mechanism** (vendored copy + CI drift check). The TUI can use the schema for input validation and tab-completion of runbook fields if it ever adds an editor mode.

### gert (Go runtime)

- The Go runtime **does not consume the JSON Schema at runtime** — it uses its own Go structs for deserialization. However:
  - The runtime's CI SHOULD validate that its struct tags produce YAML output conforming to the schema (a conformance gate, not a runtime dependency).
  - When the schema changes, Don/Ken update Go structs to match.

### Future C# / TS port runtimes

- **Consume, not generate.** Each port fetches the canonical schema from `gert-private` and either:
  - Uses it directly for validation (TS/C# JSON Schema validator libraries are mature), or
  - Generates language-specific types FROM the schema (e.g., `quicktype` for C#/TS) — the schema remains the source of truth.
- This is the key advantage of Option A: the schema is language-neutral and can serve as a generation input rather than a generation output.

---

## Boundary Statement

The JSON Schema validates **structure and types only**:

| In scope (schema validates) | Out of scope (runtime-only enforcement) |
|---|---|
| Required/optional fields per step type | GXL expression *semantics* (type correctness, Truthy=Strict) |
| Field value types (string, number, boolean, object, array) | GIS interpolation resolution (path existence, variable binding) |
| Enum values for `type`, `kind`, `apiVersion` | GCP capture path validity (requires runtime context) |
| Pattern constraints on IDs (`^[a-z][a-z0-9_-]*$`) | Cross-step reference integrity (step ID exists in flow) |
| `capture` / `capture_defaults` key shape | Expression evaluation results |
| Structural nesting (flow → steps → sub-steps) | Governance rule enforcement (approval policies) |
| `$schema` and `apiVersion` presence | Timeout/duration parsing beyond string format |

**The schema is a structural gatekeeper. The runtime evaluator is the semantic gatekeeper.** An author can have a schema-valid runbook that fails at plan time (PLAN-* errors) or eval time (GXL-* errors). This is by design — the schema catches typos and structural mistakes early; the runtime catches logic errors.

---

## Open Ports

1. **[Tess]** — Confirm that `design/gert/conformance/schema.json` (the vector schema) and `design/gert/schemas/runbook.v1.schema.json` (the runbook schema) are clearly distinct artifacts with no naming confusion. Consider renaming the vector schema file to `conformance-vector.schema.json` for clarity.

2. **[Don/Ken]** — Once the runbook schema exists, add a CI gate in `ormasoftchile/gert` that validates all `testdata/fixtures/*.runbook.yaml` files against it. This catches struct drift without manual review.

---

## Action Items

| # | Assignee | Task | Blocks |
|---|---|---|---|
| 1 | **Edith** | Author `design/gert/schemas/runbook.v1.schema.json` (initial draft) using Go structs as reference and spec sections as authority. Barbara reviews. | gert-vscode Phase 2 |
| 2 | **Barbara** | Write stub `design/gert/schemas/README.md` describing schema location, lifecycle, and consumer rules. | — |
| 3 | **Leslie** | Once schema exists: vendor into gert-vscode, add `contributes.yamlValidation`, write `scripts/sync-schema.sh` + CI drift check. | gert-vscode Phase 2 |
| 4 | **Don/Ken** | Add CI gate in `ormasoftchile/gert` validating fixture runbooks against the canonical schema. | Phase H exit criteria |
| 5 | **Tess** | Evaluate renaming `design/gert/conformance/schema.json` → `conformance-vector.schema.json` to avoid ambiguity with the new runbook schema. | — |
| 6 | **Edith** | Update `design/gert/sections/09-testing-and-acceptance.tex` to reference the runbook schema as a normative artifact (brief paragraph in §Test Framework or new §Schema Validation section). | — |

---

## References

- Leslie's audit: `.squad/agents/leslie/gert-vscode-audit.md` (F-06, Open Questions #1)
- Go runtime structs: `gert/pkg/schema/runbook.go`, `step.go`
- Conformance vector schema (sibling, NOT this): `design/gert/conformance/schema.json`
- DESIGN ONLY directive: `.squad/decisions.md` § URGENT (2026-06-05T18:13:09-07:00)
- DRIFT-DETECTION-001 pattern: `.squad/decisions.md` § Runtime Migration Plan RATIFIED

---

*Signed: Barbara — 2026-06-07T19:14:39-07:00*


---

### 2026-06-07T19:28:42-07:00: User directive  disable GERT-family GitHub Actions
**By:** ormasoftchile (via Copilot)
**What:** GitHub Actions across the GERT-family repos (gert, gert-private, gert-vscode, gert-tui) are firing too often and spending too much budget. Disable them. Re-enable case-by-case when work justifies it.
**Why:** Cost control. The recent rebuild has triggered many CI runs across 3-OS matrices; spend is too high to leave unattended.


---

# Open Questions — Runbook v1 Schema

**Author:** Edith (Spec Editor)  
**Date:** 2026-06-07T19:14:39-07:00  
**Status:** PENDING — awaiting Barbara arbitration  
**Context:** Arose during authoring of `design/gert/schemas/runbook.v1.schema.json`.

---

## SQ-001 — `name` field on Step: legacy or active?

**Observed:** `examples/nav-test/nav-test.runbook.yaml` uses a `name:` field on steps:
```yaml
- step:
    id: step_alpha
    name: Alpha Step
    type: cli
    run: echo "ALPHA_UNIQUE_OUTPUT_XYZ"
```

**Go struct:** `pkg/schema/step.go` `Step` struct has `Title string yaml:"title"` and `Subtitle string yaml:"subtitle"` but **no** `name` field.

**Schema action taken:** `name` has been included in the Step common properties as an optional string, with a note that it is a legacy field and `title` is preferred.

**Question for Barbara:** Is `name:` on a step a supported alias for `title:`, a deprecated field, or an unintentional omission from the Go struct? Should the schema:
- (a) Keep it as an optional legacy alias (current choice)
- (b) Deprecate it with a `deprecated: true` annotation
- (c) Remove it entirely (would fail the nav-test example)

**Blocks:** Nothing immediately — (a) is a safe default that lets all examples pass.

---

## SQ-002 — `extension` step type: payload undefined

**Observed:** `pkg/schema/step.go` declares `StepTypeExtension StepType = "extension"` as "v1 retained", but the `Step` struct has no corresponding `ExtensionSpec *ExtensionSpec yaml:",inline"` field. No examples of extension steps exist in `gert/examples/`.

**Schema action taken:** `extension` is included in the `type` enum with an empty `then` block (only common fields allowed). With `unevaluatedProperties: false`, any extension-specific payload field would currently be rejected.

**Question for Barbara:** What fields does an `extension` step carry? Does it use a fixed payload, or is the payload extension-defined (open object)? Should the schema:
- (a) Keep `extension` in the enum with an empty then (current choice) — too strict if extension steps have fields
- (b) Allow any additional properties for extension steps (`"then": {"unevaluatedProperties": true}`) — permissive but unspecified
- (c) Define a minimal extension step payload (e.g., `type` + some `config` object)
- (d) Remove `extension` from the enum entirely until it is specified

**Blocks:** gert-vscode hover/completion for extension steps.

---

## SQ-003 — Input `type` enum: `list` alias and complete normative set

**Observed:** `examples/incident-triage/resource-exhaustion.runbook.yaml` declares an input with `type: list`:
```yaml
inputs:
  instances:
    type: list
    required: false
    description: Affected instances to walk through
    default: [ "i-1", "i-2" ]
```

`list` is not a PJVM type name (PJVM defines: null, boolean, number, string, array, object per `03b §sec:gis:portable-json`). The Go `Input.Type` field is a plain `string` with no enum constraint.

**Schema action taken:** The Input `type` enum has been set to:
`["string", "number", "integer", "boolean", "array", "object", "list"]`

`list` and `array` are both accepted as the schema conservatively admits both until the spec clarifies.

**Question for Barbara:** 
1. Is `list` a normative type alias for `array` in the GERT input declaration vocabulary?
2. Is `integer` a separate type from `number` for input validation purposes?
3. Are `object` and `array` (or `list`) valid input types in `runbook/v1`? If so, what does the Go runtime do when it receives a YAML list as an input value?
4. What is the complete normative set of input type values? Should the spec codify this enum?

**Blocks:** Normative enum in spec section §Input Declarations.

---

## SQ-004 — Step ID pattern: `^[a-z][a-z0-9_-]*$` — runbook `id` excluded?

**Observed:** Barbara's boundary statement in `decisions/inbox/barbara-runbook-v1-schema-canonical-source.md` lists "Pattern constraints on IDs (`^[a-z][a-z0-9_-]*$`)" as in scope for schema validation.

**Also observed:** Runbook-level IDs in examples include dots:
- `id: incident-triage.network` (`network.runbook.yaml`)
- `id: incident-triage.resource-exhaustion` (`resource-exhaustion.runbook.yaml`)

These would fail `^[a-z][a-z0-9_-]*$` due to the `.` character.

**Schema action taken:** The pattern `^[a-z][a-z0-9_-]*$` has been applied only to **step** `id` fields and **iterate/parallel node** `id` fields. The runbook root `id` field is constrained only by `minLength: 1`.

**Question for Barbara:** Should the ID pattern differ between:
- Runbook `id` (appears to allow dots, hyphens)
- Step/iterate/parallel `id` (pattern `^[a-z][a-z0-9_-]*$` applies)

If so, what is the normative runbook `id` pattern? Does the spec section on runbook structure define this?

**Blocks:** ID validation strictness for gert-vscode.

---

## SQ-005 — `iterate` and `parallel` as step `type` values

**Observed:** `pkg/schema/step.go` includes:
```go
StepTypeIterate  StepType = "iterate"
StepTypeParallel StepType = "parallel"
```

in the `StepType` const block. However, all examples use `iterate:` and `parallel:` as **FlowNode-level** keys, not as `step: {type: iterate}`. No example shows a step with `type: iterate` or `type: parallel`.

**Schema action taken:** `iterate` and `parallel` are **NOT** in the step `type` enum. They are only modelled as FlowNode-level siblings of `step:`.

**Question for Barbara:** Are `iterate` and `parallel` valid step `type` values for any use case, or are they purely internal Go runtime type tags? Should they be added to the schema's step `type` enum?

**Blocks:** Would affect schema if some use cases embed iterate/parallel inside step wrappers.

---

*Edith — 2026-06-07T19:14:39-07:00*


---

# GitHub Actions Cost Control – Disabled Workflows

**Date:** 2026-06-07T19:28:42-07:00  
**Reason:** Cost control directive (ormasoftchile) – all CI workflows disabled across GERT-family repos pending case-by-case re-enable decisions.  
**Executed by:** John (Azure Platform Engineer, ops capacity)

---

## Summary

| Repo | Runs Cancelled | Workflows Disabled | Workflows Kept Active |
|------|-----------------|-------------------|----------------------|
| ormasoftchile/gert | 0 | 3 | 1 (Dependabot) |
| ormasoftchile/gert-private | 0 | 4 | 0 |
| ormasoftchile/gert-vscode | 0 | 1 | 0 |
| ormasoftchile/gert-tui | 0 | 0 | 0 |
| **TOTAL** | **0** | **8** | **1** |

---

## Workflows Disabled

### ormasoftchile/gert (3 disabled)
- **ID:** 257684578 | **Name:** E2E Tests | **Path:** `.github/workflows/e2e.yml`
- **ID:** 290175172 | **Name:** Verify Conformance Vectors | **Path:** `.github/workflows/verify-vectors.yml`
- **ID:** 290858320 | **Name:** Go Tests | **Path:** `.github/workflows/go-test.yml`

### ormasoftchile/gert-private (4 disabled)
- **ID:** 273812331 | **Name:** Squad Heartbeat (Ralph) | **Path:** `.github/workflows/squad-heartbeat.yml`
- **ID:** 273812332 | **Name:** Squad Issue Assign | **Path:** `.github/workflows/squad-issue-assign.yml`
- **ID:** 273812333 | **Name:** Squad Triage | **Path:** `.github/workflows/squad-triage.yml`
- **ID:** 273812334 | **Name:** Sync Squad Labels | **Path:** `.github/workflows/sync-squad-labels.yml`

### ormasoftchile/gert-vscode (1 disabled)
- **ID:** 270830724 | **Name:** CI | **Path:** `.github/workflows/ci.yml`

### ormasoftchile/gert-tui
- No workflows found.

---

## Workflows Left Active (Security Carve-Out)

### ormasoftchile/gert (1 kept active)
- **ID:** 236778558 | **Name:** Dependency Graph | **Path:** `dynamic/dependabot/update-graph`
  - **Reason:** Dependabot security scanning. Security posture prioritized over cost control for dependency vulnerability tracking.

---

## Verification

- ✓ **In-flight runs cancelled:** 0 found across all repos (no active or queued runs to cancel)
- ✓ **Disabled workflows confirmed:** All listed workflows now in `disabled` state
- ✓ **Active runs post-disable:** 0 active runs across all repos
- ✓ **No commits or branch changes:** Only `gh workflow disable` and `gh run cancel` executed per directive

---

## Re-Enable Runbook

When re-enabling workflows for a specific repo/workflow, use:

```bash
gh workflow enable <id> --repo ormasoftchile/{repo}
```

**Examples:**
```bash
# Re-enable Go Tests in gert
gh workflow enable 290858320 --repo ormasoftchile/gert

# Re-enable Squad Heartbeat in gert-private
gh workflow enable 273812331 --repo ormasoftchile/gert-private

# Re-enable CI in gert-vscode
gh workflow enable 270830724 --repo ormasoftchile/gert-vscode
```

To list all workflows (including disabled) in a repo:
```bash
gh workflow list --repo ormasoftchile/{repo} --all --json id,name,state,path
```

---

## Notes

- No Edith in-flight runs were cancelled (no runs found).
- Dependabot kept active to maintain security scanning for vulnerable dependencies.
- All 8 disabled workflows are now dormant and will not consume CI minutes.
- Squad workflows (gert-private) will stop sending heartbeats, issue assignments, and triage messages until re-enabled.


---

# Evaluation: Conformance Schema Naming Collision Risk

**Date:** 2026-06-07T19:14:39-07:00  
**Requestor:** ormasoftchile  
**Evaluator:** Tess (Conformance Tester)  
**Status:** EVALUATION ONLY — no code changes, no file moves  

---

## Context: The Collision Risk

Barbara's decision (2026-06-07) establishes `design/gert/schemas/runbook.v1.schema.json` as the canonical JSON Schema for GERT runbook documents. This creates a legitimate naming collision risk:

- **`design/gert/schemas/runbook.v1.schema.json`** — validates RUNBOOKS (structural, top-level document shape)
- **`design/gert/conformance/schema.json`** — validates CONFORMANCE TEST VECTORS (tv-*.yaml files, not runbooks)

Both are schemas in `design/gert/`, both are legitimate, but they validate different artifacts. New contributors landing in either directory might confuse them. I evaluated the blast radius and recommend a rename to eliminate ambiguity.

---

## Current State

### What My Schema Validates

**Confirmed:** The schema at `design/gert/conformance/schema.json` validates **conformance test vectors only** — NOT runbooks.

- **Scope:** Every YAML file matching `design/gert/conformance/tv-*.yaml` (currently 6 files)
- **Schema $id:** `https://gert.internal/conformance/vector-schema/v1`
- **Content:** Test vector objects containing `id`, `category`, `description`, `input`, `variables`, `expected` fields
- **Documentation:** The schema's own `description` states: "Validates every file in design/gert/conformance/tv-*.yaml. This schema IS the contract between the corpus author (Tess) and Phase 2 implementers."

### Where It Lives

1. **Master copy:** `P:\Projects\gert-private\design\gert\conformance\schema.json`
2. **Test copy:** `P:\Projects\gert\internal\conformance\testdata\schema.json` (vendored, synced by `scripts/sync-vectors.sh` per PR #8)

---

## Referrer Survey (Blast Radius)

### Files in `gert-private` That Reference the Schema Path

| File | Type | Reference Count | Notes |
|------|------|-----------------|-------|
| `.squad/agents/tess/history.md` | Narrative | 2 | Schema update decision entries |
| `.squad/agents/tess/charter.md` | Charter | 2 | "owns the vector schema" |
| `.squad/decisions.md` | Decisions | 5+ | Decision history, multi-entry |
| `.squad/decisions/decisions.md` | Decisions | 5+ | Acceptance criteria, scope |
| `design/gert/conformance/tv-gxl-parse.yaml` | Vector YAML | 1 | Comment: "Schema ref: design/gert/conformance/schema.json" |
| `design/gert/conformance/tv-gxl-eval.yaml` | Vector YAML | 1 | Comment: "Schema ref: ..." |
| `design/gert/conformance/tv-gxl-path.yaml` | Vector YAML | 1 | Comment: "Schema ref: ..." |
| `design/gert/conformance/tv-gis-path.yaml` | Vector YAML | 1 | Comment: "Schema ref: ..." |
| `design/gert/conformance/tv-gcp-path.yaml` | Vector YAML | 1 | Comment: "Schema ref: ..." |
| `design/gert/scripts/verify_corpus.py` | Python | 1 | Line 12: `(root / "schema.json").read_text()` |
| `design/gert/sections/09-testing-and-acceptance.tex` | LaTeX | 1 | Path ref in text |
| `design/gert/schemas/README.md` | Documentation | 1 | Explicit clarification: "This is NOT the conformance vector schema" |

### Files in `gert` Repo (Runtime)

| File | Type | Reference | Notes |
|------|------|-----------|-------|
| `internal/conformance/loader.go` | Go | Indirect | Line 23: `filepath.Join(dir, "schema.json")` — expects filename in dir |
| `internal/conformance/testdata/schema.json` | JSON | Copy | Synced copy; needs new filename when sync script runs |
| `scripts/sync-vectors.sh` | Bash | Planned | PR #8 (not yet merged): copies tv-*.yaml + schema.json |

### Summary

**Direct references:** 13 files in gert-private, 3 in gert (1 indirect, 1 copy, 1 planned)  
**Total blast radius if renamed:** ≤16 files, mostly documentation/comment updates  
**Code changes:** 1 file (verify_corpus.py), everything else is grep-and-replace

---

## Recommendation: RENAME to `vector.schema.json`

### Why This Name

**`vector.schema.json`** is the most precise and unambiguous choice:

1. **Specificity:** Directly names what it validates — test VECTORS (the collection of tv-*.yaml files)
2. **Pattern consistency:** Mirrors Barbara's naming for `runbook.v1.schema.json` (document-type + schema)
3. **Distinction from runbooks:** No one landing in the directory will confuse `vector.schema.json` with `runbook.v1.schema.json`
4. **Schema $id alignment:** The schema's $id is `https://gert.internal/conformance/vector-schema/v1` — the new filename reinforces "vector" as the key concept
5. **Future-proof:** If we ever add `vector.v2.schema.json` later, the naming is ready

### Candidates Considered and Rejected

| Candidate | Pros | Cons | Verdict |
|-----------|------|------|---------|
| `tv-schema.json` | Mirrors TV-* prefix | Confuses file prefix with schema role; "tv-" suffix means test vector ID, not vector file | **Rejected** |
| `conformance-vector.schema.json` | Most verbose, zero ambiguity | Redundant (conformance + vector); too long | **Rejected** |
| `vector.v1.schema.json` | Versioned, mirrors runbook pattern | Implies future v2 prematurely | **Not preferred** |

---

## Cost Analysis

### Files Requiring Changes

**Hard changes (code/logic):** 1
- `design/gert/scripts/verify_corpus.py` line 12

**Soft changes (documentation/comments):** 12
- 5 vector YAML files (comment updates only)
- 4 decision/history/charter MD files (grep-and-replace path references)
- 1 LaTeX spec file (path reference)
- 1 README clarification

**Transitive (CI/runtime, no code change):** 2
- `scripts/sync-vectors.sh` — copy the new filename (documentation requirement, not code defect)
- `internal/conformance/loader.go` — **NO CHANGE** (receives dir path, constructs filename relative to it)

**Total atomic PR scope:** 13 file edits, 1 file rename

---

## Migration Plan (If Approved)

Execute as a single atomic PR to gert-private:

1. **Rename the file**  
   `design/gert/conformance/schema.json` → `design/gert/conformance/vector.schema.json`

2. **Update verification script**  
   `design/gert/scripts/verify_corpus.py` line 12:  
   ```python
   schema = json.loads((root / "vector.schema.json").read_text(encoding="utf-8"))
   ```

3. **Update vector YAML comments** (5 files)  
   Each file's top comment:  
   ```yaml
   # Schema ref: design/gert/conformance/vector.schema.json
   ```

4. **Update decision/history docs** (4 files)  
   Grep-and-replace `design/gert/conformance/schema.json` → `design/gert/conformance/vector.schema.json`  
   Files: `decisions.md`, `decisions/decisions.md`, `.squad/agents/tess/history.md`, `.squad/agents/tess/charter.md`

5. **Update LaTeX spec** (1 file)  
   `design/gert/sections/09-testing-and-acceptance.tex` — update path reference

6. **Simplify README clarification** (1 file)  
   `design/gert/schemas/README.md` — update the existing note explaining the distinction

7. **Update CI/sync documentation** (1 note)  
   When `scripts/sync-vectors.sh` ships in PR #8 (gert repo), ensure it copies `vector.schema.json` instead of `schema.json`

8. **CI verification**  
   - `make verify-corpus` passes after rename
   - All vector files re-validate against the renamed schema

### Implementation Checklist

- [ ] File rename committed
- [ ] `verify_corpus.py` updated
- [ ] Vector YAML comments updated (5 files)
- [ ] Documentation grep-replaced (4 files)
- [ ] LaTeX spec updated
- [ ] README updated
- [ ] `make verify-corpus` passes (267 vectors)
- [ ] Single atomic commit: `Rename conformance schema for clarity: schema.json → vector.schema.json`

---

## Open Questions / Blockers for Barbara

**None.** This is a straightforward rename with no contract changes:

- The conformance loader in gert continues to work (receives dir path, constructs filename)
- The schema $id does not change (remains `https://gert.internal/conformance/vector-schema/v1`)
- Vector semantics are unchanged
- No runtime behavior change
- Test copy in gert will be updated by Ken when he merges the sync script

---

## Conclusion

**Recommendation:** **RENAME to `vector.schema.json`** to eliminate naming collision risk and improve clarity.

**Cost:** 13 file edits + 1 rename, all safe grep-and-replace or trivial updates.

**Risk:** Minimal — this is a rename, not a schema change. Tests will pass if the rename is executed atomically.

**Timeline:** 1–2 hours to execute (grep-and-replace + verify + commit).

Approve to proceed, and I will execute the migration in a follow-up PR.

>>>>>>> origin/main
