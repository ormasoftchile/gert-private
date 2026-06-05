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
