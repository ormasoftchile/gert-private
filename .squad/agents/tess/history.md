# Tess — History

## Project Context

- **Project:** GERT — Governed Executable Runbook Technology
- **Owner / User:** ormasoftchile (Germán)
- **Tech Stack:** Go (runtime), Azure (web platform), TypeScript (extensions/web), C# (parallel runtime, planned)
- **Joined:** 2026-06-05 for the GXL/GIS/GCP Phase 1 conformance corpus (Stream C)
- **Corpus location:** `design/gert/conformance/` (to be created)
- **Grammar source-of-truth:** `design/gert/grammar/gxl.ebnf`, `gis.ebnf`, `gcp.ebnf`

## Day 1 Context

- The conformance corpus IS the spec — anything not covered by a test vector is undefined and may diverge across runtimes (per ratified Phase 1 plan).
- Target ≥ 200 vectors across 13 categories. Stream C is XL (12 days), critical path for Phase 2 unlock.
- Ratified semantics binding for vectors:
  - OQ1: `and`/`or` short-circuit (write vectors confirming `false and {error}` → false, not error).
  - OQ2: `capture.default:` scalars only — subtree + default → `GCP-DEFAULT-SUBTREE`.
  - OI-GIS-01: `\${` is the canonical literal-dollar-brace escape.
  - OI-GCP-02: Bare root capture (`http.body` with no GDP path) is legal.
- Portable JSON value model (OQ3): null, boolean, number (IEEE 754 double, no NaN/Inf serialized), string (UTF-8), array, object. No special date/binary types.

## Learnings

### 2026-06-05T00:14:04-04:00 — Stream C Day 1 (Schema + TV-GXL-PARSE)

#### Schema design choices

- Used JSON Schema Draft 2020-12 `oneOf` to enforce mutual exclusivity between success (`value`) and failure (`error_class` + `error_code`) expected shapes. This prevents vectors that are ambiguously both.
- `variables` values are constrained to the GERT Portable JSON Value Model via a `$ref JsonValue` definition. NaN and Infinity are excluded at schema level — they can't be written as JSON anyway, but the constraint is explicit.
- `expected.error_code` pattern allows `TBD` as a sentinel for undecided vectors. This lets the corpus move forward without blocking on every open question.
- Proposed **GXL-FUNC** as the 13th category (Barbara's plan lists 12 explicitly). Flagged in inbox for ratification.
- Heavily commented the schema because Phase 2 implementers will use it directly.

#### Vector ID convention confirmed

- Pattern: `TV-(GXL|GIS|GCP|CONFORM)-[A-Z]+(-[A-Z0-9]+)*-\d{3,4}`
- Zero-padded 3-digit suffix, sequential within each YAML file.
- Files are per-category (one YAML file per category prefix).

#### Error codes not found in gxl.ebnf (for follow-up)

- None missing — all 10 GXL-PARSE-00x codes have clear trigger conditions.
- BUT: **code conflict** between §2.3 body (GXL-PARSE-007) and §6.1 catalog (GXL-PARSE-010) for keyword-as-identifier. Raised to Barbara in inbox.

#### Ambiguities surfaced

1. **GXL-PARSE-007 vs GXL-PARSE-010**: Grammar body says GXL-PARSE-007 for keyword-as-identifier; error catalog says GXL-PARSE-010. Both can't be right. Vectors 081–083 marked `undecided`.
2. **`1.2.3` classification**: Lexer likely produces NUMBER(1.2) + DOT + NUMBER(3), so parse fails with GXL-PARSE-001 not GXL-PARSE-003. But grammar's "malformed numeric token" phrasing is vague.
3. **String concatenation GXL-PARSE-007**: Parser can't detect this at parse time (no type info). Deferred to TV-GXL-EVAL as GXL-TYPE-003.
4. **`str.foo` (no parens)**: Ambiguous whether error is GXL-PARSE-006, GXL-PARSE-010, or GXL-PARSE-001 depending on PEG fallthrough order. Vector 083 marked undecided.

#### Coverage gaps to test next

- TV-GXL-EVAL: short-circuit witnesses, type mismatch, null propagation, division-by-zero, negative modulo (OI-GXL-03)
- TV-GXL-PATH: missing variable, missing intermediate segment, index-out-of-bounds, index-on-non-array
- TV-GIS-*: escape sequence `\${` (OI-GIS-01), template with zero interpolations, nested interpolations
- TV-GCP-*: bare root capture `http.body` (OI-GCP-02)
- GXL-TYPE-004 (wrong arg count) — surfaces at plan time; needs a dedicated vector in GXL-EVAL or GXL-ERROR category
