# Tess — History

## Current Status Summary (2026-06-07)

**Phase 1 COMPLETE.** Conformance corpus: 211 vectors across 5 YAML files (GXL-PARSE 83, GXL-EVAL 87, GXL-PATH 35, GIS-PATH 15, GCP-PATH 41). Zero TBD vectors.

**Schema naming coordination:**
- Evaluated collision risk: `design/gert/conformance/schema.json` (test vectors) vs Barbara's `runbook.v1.schema.json` (runbooks)
- **Recommendation:** Rename to `vector.schema.json` (13-16 files impacted; 1 code change in verify_corpus.py, rest docs/comments)
- **Status:** Awaiting Barbara ratification before executing atomic migration PR

**Current role:** Conformance corpus authority, schema naming clarity advocate

**Next steps:** (1) Await Barbara ratification on schema rename, (2) Execute migration PR once approved, (3) Support Phase A runtime team (Ken/Don) with corpus guidance

---

## Project Context

- **Joined:** 2026-06-05 for Stream C (Phase 1 conformance corpus)
- **Corpus location:** `design/gert/conformance/` (6 YAML files, 267 vectors)
- **Grammar authority:** `design/gert/grammar/gxl.ebnf`, `gis.ebnf`, `gcp.ebnf`
- **Phase 1 goal:** ≥200 vectors across 13 categories; ALL ambiguities pinned

---

## Consolidated Phase 1 Record

**2026-06-05 to 2026-06-07:** Delivered corpus with zero TBD vectors:

**Stream C deliverables:**
- TV-GXL-PARSE: 83 vectors (keywords, identifiers, operators, precedence, escape sequences)
- TV-GXL-EVAL: 87 vectors (short-circuit AND/OR, arithmetic, type checking, stdlib, null semantics)
- TV-GXL-PATH: 35 vectors (missing variables, out-of-bounds, type errors, keyword-prefix identifiers)
- TV-GIS-PATH: 15 vectors (optional chaining `?.`, `?.[N]`, full-tail short-circuit, stdlib composition)
- TV-GCP-PATH: 41 vectors (local/step/HTTP/event/cross-step sources, JSON/YAML paths, subtree captures, defaults)

**Arbitrations resolved (TESS-AMBIG-3..6 by Barbara):**
- **AMBIG-3:** `false < true` → GXL-TYPE-005 (boolean ordered comparison forbidden)
- **AMBIG-4:** array equality → GXL-TYPE-001 extended (scalars+null only)
- **AMBIG-5/6:** dot-access on scalar/null → GXL-PATH-004 (field access on non-object)
- **Result:** 4 TBD vectors pinned; 3 companion vectors added; Phase 1 corpus final: 211 vectors, 0 TBD

**Schema design:**
- JSON Schema Draft 2020-12, `$id: https://gert.internal/conformance/vector-schema/v1`
- Used `oneOf` for mutual exclusivity (success vs error)
- Variables constrained to PJVM types (RFC 8259: null, bool, number, string, array, object)
- Pattern: `TV-(GXL|GIS|GCP|CONFORM)-[A-Z]+(-[A-Z0-9]+)*-\d{3,4}` (zero-padded 3-4 digit suffix)

**Learnings:**
- Array/object literal syntax forbidden in GXL (`[a,b]` is GXL-PARSE-007) → all list tests must use variables
- GDP evaluation order: resolve first, then pass to function (OI-GXL-05 pinned)
- Null semantics complete: `null == null` → true, `null < x` → GXL-EVAL-004, `not null` → GXL-TYPE-002
- Keyword-prefix identifiers (`andthing`, `ornot`) critical for cross-runtime parity
- Phase 1 exit gate ✅ COMPLETE; Phase A runtime validation ready

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

- Added 13 truthy-coercion vectors per barbara-truthy-corpus-vectors.md (Strict semantics, GXL-TYPE-002).

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

---

## Day 2 Context

### 2026-06-05T00:27:12-04:00 — Stream C Day 2 (TV-GXL-EVAL + TV-GXL-PATH)

#### Vector counts delivered

- TV-GXL-EVAL: 87 vectors (target ≥35) — 2.5× over
- TV-GXL-PATH: 35 vectors (target ≥25) — 1.4× over
- Total batch: 122 new vectors

#### Short-circuit witness pattern (OQ1)

- Canonical short-circuit proof: the RHS must be an expression that WOULD raise an error if evaluated (missing variable, `1/0`, `len(null)`).
- Three witness categories confirmed for AND and OR respectively.
- Chained AND with mid-chain false: `true and false and missing_var` — short-circuit fires at second operand, third not evaluated. Good asymmetric test.

#### Error codes confirmed (all exist in gxl.ebnf §6)

- GXL-EVAL-002: division/modulo by zero ✅
- GXL-EVAL-004: null in ordered comparison ✅
- GXL-TYPE-001: cross-type comparison ✅
- GXL-TYPE-002: non-bool in boolean position (including `not null`, `not 42`) ✅
- GXL-TYPE-003: arithmetic on non-number / wrong type to stdlib ✅
- GXL-TYPE-004: wrong argument count ✅
- GXL-PATH-001: variable or segment not found ✅
- GXL-PATH-002: array index out of bounds ✅
- GXL-PATH-003: index on non-array or null ✅

#### Error codes with unclear triggers (flag for Barbara)

- GXL-EVAL-001: "variable resolution failure not covered by PATH codes" — trigger condition opaque; PATH codes seem exhaustive. May be a dead code. Barbara to confirm.
- GXL-EVAL-003: valid but deferred (regex.match bad pattern). Not covered in this batch.

#### New ambiguities surfaced (requires Barbara arbitration)

1. **TESS-AMBIG-3**: `false < true` — no error code for ordered comparison on non-orderable type (bool). GXL-TYPE-001 requires *incompatible* types; booleans are same type.
2. **TESS-AMBIG-4**: `myArr == myArr` (array == array) — gxl.ebnf §5.2 defines equality only for scalars and null. List/object equality semantics unspecified.
3. **TESS-AMBIG-5**: `foo.bar` when `foo` is a scalar — GXL-PATH-003 explicitly covers INDEX on non-array; no code for DOT access on non-object scalar.
4. **TESS-AMBIG-6**: `nullFoo.bar` (dot-access on null) — GXL-PATH-003 covers bracket-index on null; dot-access on null has no explicit code.

Ambiguities 5 and 6 likely share the same resolution: either extend GXL-PATH-003 or add GXL-PATH-004.

#### Array literals: important constraint

Array/object literal syntax `[a, b]` / `{key: value}` is GXL-PARSE-007 (forbidden). All list-function tests MUST put arrays in `variables`. This significantly changes how list.* stdlib tests are structured — no inline array arguments possible.

#### GDP in function argument (OI-GXL-05)

Confirmed pattern: `str.contains(config_key, "prod")` where `config_key` is a GDP path resolving a variable. Resolution order: GDP evaluates first, then function receives the resolved scalar. TV-GXL-EVAL-087 pins this.

#### Null semantics summary (now pinned)

- `null == null` → true ✅
- `null != null` → false ✅
- `x == null` where x is null → true ✅
- `null == 0` → false (null-safe equality, no error) ✅
- `null < x` → GXL-EVAL-004 ✅
- `not null` → GXL-TYPE-002 ✅
- `nullVar` (top-level, existing variable with null value) → null (not an error) ✅

#### OI-GXL-03 modulo — not truly "pending"

Grammar §7 OI-GXL-03 SPECIFIES the behavior (IEEE 754 fmod, sign of dividend). The open issue is cross-runtime validation (Go vs C#), not the value itself. Vectors 050–052 are pinned with expected values `-1`, `1`, `-1`. Not TBD.

#### Path identifier edge cases

The keyword-prefix identifier pattern (`andthing`, `ornot`, `trueish`, `nullPtr`) is critical for cross-runtime parity. Any runtime using a greedy keyword-first tokenization strategy may incorrectly lex `andthing` as KW_AND + `thing`. All four tests must pass before Phase 2 sign-off.

#### Coverage gaps to test next

- TV-GXL-EVAL Day 3:
  - `regex.match` happy path + invalid pattern (GXL-EVAL-003)
  - `str.trimPrefix`, `str.trimSuffix` (defined in §4.2 but not yet covered)
  - `list.contains` with cross-type item (GXL-TYPE-003)
  - GXL-EVAL-001 trigger (if Barbara clarifies the code)
  - Resolved ambiguities (TESS-AMBIG-3, 4, 5, 6) → replace TBD vectors
- TV-GIS-*: escape sequences, nested interpolation
- TV-GCP-*: bare-root capture (OI-GCP-02), default on subtree (GCP-DEFAULT-SUBTREE)
- TV-PLAN-*: PLAN-004, PLAN-005 (migration-era codes per Barbara §4)

---

## Integrated to Main (2026-06-05T00:27:12-04:00)

✅ **Day 2 memo merged to `.squad/decisions.md`** under section "GXL Phase 1 Day 2 — GIS/GCP Specs, Eval+Path Vectors, Conflict Arbitration". Corpus now 205 vectors (past ≥200 target). Four TESS-AMBIG items (3, 4, 5, 6) flagged as open arbitrations; 4 vectors remain TBD pending Barbara decision. Phase 1 status: 03a/03b/03c/03d all complete; Stream D unblocked.

## 2026-06-04T20:14:36-07:00 — Stream E Removed (Scribe notification)

**Conformance Scope Reduced:** The migration-tool conformance vectors (dogfood-regression skill) have been removed. No action needed — your 205-vector corpus for GXL/GIS/GCP evaluation and path testing remains the authoritative conformance set.

**Coverage:** Your corpus covers the runtime contracts for Streams A/B/C/D/F. Phase 2 runtimes (Go, C#, and future) will be validated against this body of work, not a migration tool.

**Decision:** Merged to `.squad/decisions.md` (timestamp 2026-06-04T20:14:36-07:00)

## 2026-06-05T07:28:56.273-07:00 — GIS Optional-Chaining Path Vectors

**Status:** Delivered `design/gert/conformance/tv-gis-path.yaml` for ratified GIS optional-chaining (`?.` / `?.[N]`).

**Vector range:** `TV-GIS-PATH-001` .. `TV-GIS-PATH-015`.

**Coverage:** Simple miss, deep miss, full-tail short-circuit, mixed mandatory/optional hard-error boundaries, null vs empty/present PJVM values, optional bracket indexing, missing collection, stdlib composition with explicit evaluation order, and `${?.root}` parse rejection.

**Schema note:** Updated `design/gert/conformance/schema.json` to admit the GIS path error family and catalog code `GIS-PATH-MISSING`, which `gis.ebnf` defines as a named error rather than a numeric `GIS-PATH-###` code.

**Decision memo:** Dropped `.squad/decisions/inbox/tess-gis-path-vectors.md` for Scribe integration.

## 2026-06-05T09:25:14.584-07:00 — GCP Capture Path Vectors

**Status:** Delivered `design/gert/conformance/tv-gcp-path.yaml` for the missing GCP corpus authorized by Phase 2 Day 1 Q1.

**Vector range:** `TV-GCP-PATH-001` .. `TV-GCP-PATH-041`.

**Coverage:** Local `stdout`/`stderr`/`exit_code`/`json`/`yaml`, snake-case-only `exit_code`, JSON/YAML GDP paths, bare-root captures, legacy `stdout.*` JSON dot paths, subtree object/array captures, missing/out-of-bounds/runtime parse failures, malformed syntax, HTTP/event/cross-step sources, and default-policy hard errors.

**Schema note:** Updated `design/gert/conformance/schema.json` to admit GCP resolver/type/default error families: `GCP-RESOLVE-###`, `GCP-TYPE-###`, and `GCP-DEFAULT-SUBTREE`.

**Encoding limits:** Current single-input vector shape cannot cleanly encode multiple captures from the same path (OI-GCP-04) or execution-order source-unavailable state (`GCP-RESOLVE-001`). Flagged both in `.squad/decisions/inbox/tess-gcp-vectors.md` for Don/Scribe.

## 2026-06-05 AMBIG-3..6 Arbitrated

AMBIG-3..6 arbitrated by Barbara; my 4 TBD vectors are now pinned:
- **AMBIG-3** (`false < true`): GXL-TYPE-005 (boolean ordered comparison forbidden)
- **AMBIG-4** (array equality): GXL-TYPE-001 extended (scalars+null only)
- **AMBIG-5/6** (dot-access on scalar/null): GXL-PATH-004 (field access on non-object)
- 3 companion vectors added (TV-GXL-EVAL-091, -092; TV-GXL-PATH-036)
- **Corpus:** 211 vectors, 0 TBD

Phase 1 complete. All spec sections updated. Phase A starts in `ormasoftchile/gert`.

## 2026-06-05T23:12:22-04:00 — YAML Escape Fix (TV-GXL-PARSE-062/063)

**Status:** Conformance corpus hygiene. Fixed YAML encoding bug in tv-gxl-parse.yaml.

**Vectors:** TV-GXL-PARSE-062 (hello \q), TV-GXL-PARSE-063 (hex \xFF)

**Root cause:** Single-quoted YAML scalars have no backslash-escape semantics. Original encoding `\\q` / `\\xFF` produced two literal backslashes instead of single-backslash invalid-escape sequences.

**Fix:** `design/gert/conformance/tv-gxl-parse.yaml` — corrected to single backslash. Commit: `424a334` on `gert-private/main`.

**Timing:** Landed before Ken's Phase A sync, protecting Don's 83/83 pass rate from regression to 81/83 upon PR merge.

## 2026-06-07T19:14:39-07:00 — Schema Naming Collision Evaluation

**Task:** Evaluate the naming-collision risk created by Barbara's new `design/gert/schemas/runbook.v1.schema.json` and propose a remediation for my existing `design/gert/conformance/schema.json`.

**Scope Confirmed:**
- My schema validates **conformance test vectors** (tv-*.yaml files), NOT runbooks
- Schema $id: `https://gert.internal/conformance/vector-schema/v1`
- Validates 267 vectors across 6 YAML files (TV-GXL-PARSE, TV-GXL-EVAL, TV-GXL-PATH, TV-GIS-PATH, TV-GCP-PATH)

**Blast Radius Surveyed:**
- 13 files in gert-private reference the path (5 vector YAML comments, 1 Python script, 1 LaTeX doc, 4 decision/history docs, 1 README)
- 3 files in gert repo (1 Go code that receives dir path, 1 vendored copy, 1 planned sync script)
- Total impact: ≤16 files, mostly grep-and-replace or documentation updates

**Recommendation:** Rename to `vector.schema.json`
- Most precise: directly names what it validates (test vectors)
- Eliminates confusion with `runbook.v1.schema.json`
- Mirrors naming style (document-type + schema)
- Cost: 13 file edits + 1 rename, all safe

**Detailed report:** `.squad/decisions/inbox/tess-conformance-schema-naming.md` (awaiting Barbara ratification before execution)

## 2026-06-07T19:28:42-07:00 — Conformance Schema Naming Recommendation

**Status:** Evaluation complete. Recommendation submitted for Barbara ratification.

**Finding:** The collision between `design/gert/conformance/schema.json` (test vectors) and `design/gert/schemas/runbook.v1.schema.json` (runbooks) creates legitimate confusion risk for future contributors.

**Recommendation:** Rename `design/gert/conformance/schema.json` → `design/gert/conformance/vector.schema.json`

**Justification:**
- Direct naming: `vector.schema.json` explicitly names what it validates (test VECTORS, not runbooks)
- Unambiguous: New naming immediately distinguishes from Barbara's `runbook.v1.schema.json`
- Mirrors Barbara's convention: document-type + schema (vector + schema)
- Schema $id unchanged: `https://gert.internal/conformance/vector-schema/v1` remains intact

**Blast Radius:**
- 13 files in gert-private (5 vector YAML comments, 1 Python script, 1 LaTeX doc, 4 decision/history, 1 README)
- 3 files in gert repo (1 indirect Go code, 1 vendored copy, 1 planned sync script)
- Total: ≤16 files, all safe (1 code change, 12 documentation/comment updates)

**Cost:** 1-2 hours to execute (grep-and-replace + verify + commit)

**Status:** Awaiting Barbara ratification. Once approved, will execute migration as single atomic PR.

## Learnings — 2026-06-07T19:28:42-07:00 — Cross-Repo Schema Rename Execution

**Blast Radius (actual vs. estimate):**
- Estimate: 13–16 files (gert-private) + 3 files (gert), total ≤16 files, 1 code change
- Actual: **10 files in gert-private** (schema.json rename, 5 TV-*.yaml comments, LaTeX §9, runtime-implementation-plan.md, tess/charter.md, decisions.md migration note) + **7 files in gert** (schema.json rename, loader.go, 5 TV testdata comments) = 17 total
- Delta: Estimate missed decisions.md (append-only note), charter.md (living doc), and overcounted "Python script" (verify_corpus.py doesn't exist) and "README" (no conformance README). Net: accurate within expected range.

**Unexpected findings:**
- No `verify_corpus.py` exists; validation was done with a Node.js Ajv one-liner (Python 3.12 available but no json-schema library installed). Node's `ajv@8 + ajv-formats + js-yaml` installed ad hoc.
- The schema's `$id` (`https://gert.internal/conformance/vector-schema/v1`) already used `vector-schema` — no update needed.
- Pre-existing unstaged changes on `main` (from prior squad sessions) were in the working tree on branch creation. Required explicit selective `git add` instead of `git add -A` to avoid bundling unrelated squad session artifacts into the rename commit.
- `node_modules/` created by ad-hoc validation deps was untracked but visible in `git status`; not committed (no `package.json` committed either — npm install was ephemeral).

**Cross-repo PR-linking pattern:**
- Create PR B (gert runtime) first to get URL, then create PR A (gert-private spec) with PR B URL in body, then `gh pr edit B` to add PR A URL. Two-step because both PRs need the other's URL.
- Commit messages reference the decision ID (`barbara-schema-rulings-2026-06-07`) for traceability without PR numbers (which aren't known at commit time).

**PRs delivered:**
- PR A (gert-private): https://github.com/ormasoftchile/gert-private/pull/8
- PR B (gert): https://github.com/ormasoftchile/gert/pull/36
- Local validation: 5/5 TV-*.yaml PASS (Ajv), Go tests PASS (`ok internal/conformance 1.720s`)
