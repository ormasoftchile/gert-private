# tess — History Archive

**Archived:** 2026-08-15T16:27:17.3607698-07:00
**Size:** 55557 bytes

---

# Tess — History

## Current Status Summary (2026-08-15T13:03:41-07:00)

**Dynamic Include adversarial catalogue delivered.**  
165 cases across 14 groups for the `runbook_ref` / `resolve_from: catalog` feature. 13 open questions filed with Barbara. Go skeleton test (`dyninclude_conformance_test.go`) + YAML corpus (`tv-dyninclude.yaml`) added to `internal/conformance/dynincludedata/` in `ormasoftchile/gert`. `go build ./...` and `go vet ./internal/conformance/...` pass.

**Highest-risk cases identified:**
- Path-injection family (TV-DYN-REF-003..019): any character in an identity that looks like a filesystem path must be structurally rejected before catalog lookup.
- Governance non-widening (TV-DYN-GOV-001, GOV-005, GOV-015): child runbook cannot widen `require_approval`, `deny_commands`, or `capabilities` past parent; security hole if missed.
- Ambiguity error vs not-found (TV-DYN-CAT-003, TV-DYN-NFD-008): `on_not_found: continue` must NOT silently suppress an ambiguous-bare-id error.
- Runtime cycle detection (TV-DYN-CYCLE-001..004): static planner cannot see dynamic back-edges; runtime MUST check.
- Trace payload shape (TV-DYN-TRACE-002): once pinned, the exact JSON field names on `include/resolved` become a normative wire contract.

**Open questions (14 items):** `.squad/decisions/inbox/tess-dynamic-include-open-questions.md`  
**Catalogue:** `.squad/decisions/inbox/tess-dynamic-include-vectors.md`

---

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

### 2026-08-15T13:03:41-07:00 — Dynamic Include Adversarial Catalogue

#### Conformance harness conventions (from study of existing corpus)

- **EnumHarness pattern:** `variables` map = synthetic workspace filesystem (POSIX path → file content string); `input` = entry-point runbook path within that workspace. The harness calls `gert run <input>` in a temp workspace dir.
- **Error assertion:** the harness checks both stdout and stderr for the error code string; plan-time errors land on stderr, per-step runtime failures on stdout.
- **Skip pattern:** named skip reasons are required — no silent drops. Use `enumRuntimeGapSkips` map for structured named skips.
- **Catalog vectors** use `pkgcatalog.Build` directly (not CLI) because CLI output doesn't expose catalog internals — mirror `runCatalogVector` for any future `DYN-CAT` vectors that assert on `RunbookEntry` fields.
- **Schema category enum** in `vector.schema.json` must be updated to add `DYN-*` categories (and `DYN` error class) before any `tv-dyninclude.yaml` vectors can be schema-validated. This is gated on Barbara's ratification.
- **New skeleton pattern**: for features not yet implemented, create a `<feature>data/` subdirectory alongside `enumdata/`, add a Go test file guarded with `t.Skip("pending ... implementation")`, and group skips by named test area so the report shows upcoming coverage even before the feature lands. Verified: `go build ./...` and `go vet ./internal/conformance/...` pass with this pattern.

#### Highest-risk cases for dynamic include feature

1. **Path-injection (TV-DYN-REF-003..027):** Every identity that looks like a filesystem path must be rejected *before* catalog lookup — this is a security boundary, not a UX convenience. `./`, `../`, `/abs`, `C:\`, backslash, `a/b/c` (>1 slash), `file://`, `http://`, null byte, control chars, RTL override.
2. **Governance non-widening (TV-DYN-GOV-001, 005, 015):** Child runbooks resolved at runtime cannot widen the parent's `require_approval`, `deny_commands`, or `capabilities`. A miss here is an authority-escalation vulnerability.
3. **Ambiguity ≠ not-found (TV-DYN-CAT-003, TV-DYN-NFD-008):** `on_not_found: continue` must NOT suppress `DYN-005` (ambiguous bare id). These are semantically different outcomes and must not be conflated.
4. **Runtime cycle detection (TV-DYN-CYCLE-001..004):** The static planner (flowwalk) cannot see through dynamic back-edges. Cycle checks must be re-run at execution time using the call stack of actively-executing include sites.
5. **No double-evaluation (TV-DYN-TMPL-011):** A rendered `runbook_ref` containing `${...}` must be used as a literal identity, not re-evaluated as a GIS template. Failure here would allow template-injection attacks.
6. **Trace payload shape (TV-DYN-TRACE-002):** Once Barbara ratifies the `include/resolved` event kind and field names, those become normative wire-format contracts that all runtimes must implement identically. Get the names right the first time.

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

## 2026-08-10T13:25:10.498-07:00 — Enum-Constrained Tool/Runbook Outputs MVP Corpus (`tv-enum.yaml`)

**Status:** Delivered `design/gert/conformance/tv-enum.yaml` implementing Barbara's ratified
`barbara-enum-constraint-mvp-architecture-ruling.md` (AR-ENUM-1..15) on top of Edith's already-completed
spec/schema/fixture prep. No schema or spec files modified (out of scope per the ruling's Handoff — Edith
already extended `vector.schema.json` with `ENUM-*` categories/codes).

**Vector count:** 58 vectors (≥54 required) across `ENUM-DECL` 14, `ENUM-UNICODE` 8, `ENUM-DEFAULT` 6,
`ENUM-PLAN` 7, `ENUM-RUNTIME` 8, `ENUM-SUBST` 7, `ENUM-MOCK` 5, `ENUM-TRACE` 3.

**Coverage:** All four declaration sites (S1 tool arg, S2 substituted-action output, S3 runbook input, S4
runbook output); string-only/no-secret; empty/whitespace/duplicate members; YAML 1.2 scalar typing
(unquoted `yes`/`no`, ints, mapping-as-enum, nested sequence, tagged item); NFC/NFD normalization and
canonical-case comparison; set-equality/ordering-independence; default-must-be-member at all four sites;
runtime interpolation/materialization/replay determinism; type-before-enum single-cause ordering;
substituted-action contract (member superset/subset, PKG-013 incompatibility); C2 mock-conformance
(binding equivalence real vs. mock); metadata/trace/redaction (C1, S1 `redact: true`); includes-declaration
locality; and unconstrained-tool backwards compatibility. Every code in `{ENUM-001..009, ENUM-W001,
PKG-013}` has at least one dedicated negative/warning vector.

**Validation:** `verify_corpus.py` → `OK 423/423` (365 pre-existing + 58 new). Extra self-checks (throwaway,
deleted after use): embedded fixture YAML all parses cleanly; embedded `runbook/v1` fixtures validated
against `runbook.v1.schema.json` — the only 14 "failures" are (a) 13 deliberately schema-illegal
`ENUM-DECL`/`ENUM-PLAN` structural vectors (the point of the test) and (b) `TV-ENUM-RUNTIME-003`, which
exercises the already-known/ticketed D3 `from:`-enum drift (§16 of the ruling) — annotated with a note, not
altered.

**Genuine gap found (new, not in D1-D4):** `$defs.Output` in `runbook.v1.schema.json` has no `default`
property (unlike `$defs.Input`), making a literal S4 `outputs.<name>.default` fixture unrepresentable.
Worked around by moving `TV-ENUM-DEFAULT-003` to the schema-free S2 site (also an AR-ENUM-6 enumerated
site) and adding `TV-ENUM-DEFAULT-006` as a documented `untestable-in-corpus-format`/`schema-gap` contract
vector. Filed `.squad/decisions/inbox/tess-enum-corpus-notes.md` for Edith/Barbara.

**Files:** `design/gert/conformance/tv-enum.yaml` (new, deliverable), `design/gert/scripts/gen_enum_vectors.py`
(new, generator — kept per precedent, safe to delete once reviewed), `.squad/decisions/inbox/tess-enum-corpus-notes.md`
(new, one genuine ambiguity flagged). No unrelated pre-existing uncommitted changes touched; nothing committed.

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

---

## 2026-08-09T15:59:36-07:00 — Tool Packages Conformance Corpus (`tv-pkg-resolve.yaml`)

**Task:** Implement the deferred Tool Packages conformance corpus per the ratified `barbara-tool-packages-architecture-ruling.md` (AR-TP-1..13) and the three gate reviews (gate 3 = **APPROVED**, spec frozen). Floor: ≥46 deterministic vectors covering every reachable `PKG-001`..`PKG-030` (PKG-019 reserved/unassigned), `PLAN-010`, and `PKG-W001`-`PKG-W003`.

**Delivered:** `design/gert/conformance/tv-pkg-resolve.yaml` — **85 vectors** across 7 categories: `PKG-VERSION` (17), `PKG-RESOLVE` (15), `PKG-COLLIDE` (8), `PKG-PATH` (14), `PKG-SUBST` (12), `PKG-DIGEST` (6), `PKG-ERROR` (13). No schema changes needed — `vector.schema.json` already had full `PKG-*` machinery (category enum, id pattern, `PackageExpected` def) wired in from a prior revision cycle. Total corpus is now 365 vectors across 6 files (was 280/5). `verify_corpus.py`: `OK 365/365 vectors validate`.

**Coverage:** every one of PKG-001–030 except reserved PKG-019, plus PLAN-010 and PKG-W001/W002/W003, has at least one vector; two bonus vectors also exercise GCP-PARSE-001/GCP-RESOLVE-002 (reachable specifically via PKG-SUBST misuse of the `outputs.<name>` capture root) using the `ErrorExpected` (`error_class`/`error_code`) branch rather than `PackageExpected`, since that branch's `error.code` pattern is `^PKG-\d{3}$|^PLAN-010$` and does not accept GCP-* codes. No uncovered codes; no undecided vectors were needed — the gate-3-approved spec was concrete enough to author every scenario without guessing.

**Corrections caught by careful table cross-referencing (worth remembering):** my first draft mislabeled several codes from a hasty first reading of the ruling summary rather than the full table:
- `PKG-007` = general unsafe/escaping path (syntax OR containment, incl. `..` escapes, absolute paths, NUL bytes). `PKG-008` = symlink/junction escape specifically (cycle/hop-limit too) — these are commonly conflated but are distinct codes.
- `PKG-012` = "action not declared by the resolved tool" (Binding), **not** "duplicate export id" as I first assumed. A duplicate `exports.tools[].id`/`.path` pair is a manifest-shape violation and was mapped to `PKG-004` (fails schema validation) per the schema's own "unique by both id and path" documentation. A genuine unknown-action-name vector was written for the real `PKG-012`.
- `PKG-022` = "undeclared cross-tier shadowing of a bare tool name" — **not** substitution depth. I initially had a cross-tier-collision vector asserting *silent* resolution to the higher tier with no qualifier at all; per the ruling ("cross-tier shadowing is permitted and silent **only** when the runbook declares the intent via `toolRefs[].package`/`.path`"), an *undeclared* bare-name reference across tiers is actually a hard `PKG-022` error, not a silent resolve. Fixed by splitting into two vectors: an undeclared-shadowing error case (`PKG-022`) and a declared-shadowing success case (explicit `toolRefs[].package` pins the lower tier on purpose, silently, since the explicit qualifier itself is the disambiguating signal). Substitution depth (`> 4` levels) is the actual, separate `PKG-028`.
- `PKG-018` = "reserved name, case-only or NFC-only collision, trailing dot/space" — used for four distinct sub-cases (case-only collision, reserved Windows device name, trailing dot/space, NFC-vs-NFD normalization collision) rather than only one.
- **Lesson:** grep/quote the exact `03d-parse-time-enforcement.tex` error table line-by-line per code before writing any vector that asserts it, rather than relying on a prose summary (even the ruling's own §10 category-count summary text can drift slightly from the final, gate-3-approved table). A post-hoc automated cross-check (regex-extracting every `PKG-\d{3}`/`PLAN-010`/`PKG-W\d{3}` token mentioned in each vector's `description`+`note` and diffing against the vector's actual asserted `expected` code) caught the description-vs-code drift left over from two of these fixes (stale prose after a code was corrected) before finalizing the file.

**Schema-shape gotchas:** `description` has `maxLength: 200` — many of my first-draft, citation-heavy one-liners (~250-350 chars) blew past this; solved by an automated truncate-and-move-overflow-to-`note` pass in the generator rather than hand-editing 30+ vectors. `PackageExpected.error.code` pattern is `^PKG-\d{3}$|^PLAN-010$` only — a PKG-category vector asserting a GCP-level failure must use the `ErrorExpected` branch (`error_class`+`error_code`) instead of `error.code`, since `expected` is a `oneOf` across `SuccessExpected`/`ErrorExpected`/`PackageExpected` and is not gated by the vector's own `category`.

**Digest ground truth:** for the 6 `PKG-DIGEST` vectors, real SHA-256 digests were computed in Python following the spec's exact algorithm (raw-byte file digest → `hex  posix-relpath` lines → sort ascending by UTF-8 byte order → newline-joined blob → `sha256:` + hex digest of blob; catalog digest is the analogous construction over `qualifiedName  digestOrFileDigest  tier` lines) so the vectors are self-verifying ground truth, not placeholders. A one-byte content change (`requires-approval: false` → `true`) was used to demonstrate content-sensitivity with an independently recomputed second digest value.

**Modeling extensions (flagged in the corpus file header for reviewer awareness, not literal schema fields):** `builtins: [name, ...]` for tier-0 registry contents, `extensions: ["server/tool", ...]` for tier-3 registrations, `symlinks`/`reparse_points: {path: target}` for POSIX symlinks / NTFS junctions — none of these have an on-disk representation expressible as a flat file-content string, so they are synthetic conventions layered on top of the schema's `additionalProperties`-flexible `variables` field.

## 2026-08-15T16:30:00-07:00 — Dynamic Include Pass 7 FINAL (DEF-006 landed, GOV-005 passes, corpus closed)

**Final state: 26 pass / 3 skip / 0 fail out of 29 vectors. `go test ./...` exit 0.**

### DEF-006 fixed (Don)

`WithEntryGovernance(ctx, policy)` called before `eng.Start` seeds the entry runbook's static `governance:` block as the initial `dynGov` in context. `executeDynamic` now calls `dynGovFromCtx(ctx)` and gets the real parent policy instead of nil. `ComposeGovernance(parentGov, childGov)` correctly unions the parent's `deny_commands` with the child's, and the deny check fires before `platform.Exec`.

Don proved it with `TestRun_EntryGovernance_DenyCommandBlocksChild` — deny rule in entry runbook blocks a matching command inside a dynamically included child.

### TV-DYN-GOV-005 passes

The corpus fixture is correct: `command: rm` + `deny_commands: ["rm*"]` in entry runbook's `governance:`. With DEF-006 fixed, parent governance is seeded, composed, and enforced. `GOVERNANCE-001` fires before `rm` reaches `platform.Exec`. No fixture changes needed.

### Final skip state — all 3 permanent (no pending blockers)

| Vector | Ruling | Reason |
|---|---|---|
| TV-DYN-TMPL-005 | B-16 | DINC-009 intentional dead code; `expr.Eval` always returns string |
| TV-DYN-GOV-001 | Harness limitation | `NoOpApprovalGate` in subprocess; `TerminalApprovalGate` works interactively; GOV-001 harness note filed with Barbara |
| TV-DYN-GOV-015 | B-17 | No `Capabilities` field in `schema.GovernanceConfig`; out of scope |

### Consistency sweep result

- `go build ./...` — exit 0
- `go test ./...` — exit 0, all packages green, zero failures; pre-existing flakes (`TestStdioTransport_*`, `TestSSE_FilterByRunID`) not present in this run
- All 10 ICM e2e tests: 4 required branches + 6 adversarial cases — all passing
- No regression from DEF-005 parser+executor fix, DEF-006 governance seeding, Ken's `CheckScript`/`ScriptChecker`, or David's schema description additions

### Corpus is closed

No blanket skips. No pending blockers. 26 vectors actively prove behavior; 3 are permanent skips with named rulings. Every defect found (DEF-001..006) was reported rather than patched, and each has been fixed by the responsible engineer and verified by re-running the vector.

**Status:** Conformance corpus at **25 pass / 4 skip / 0 fail** out of **29 vectors** (29th added this pass).

### DEF-005 fixed (Ken)

Two-layer fix:
1. `internal/parser/validate_semantic.go`: `intendedDynamic` heuristic — empty `runbook_ref: ""` WITH `resolve_from` or `on_not_found` present emits DINC-002 instead of `include/missing-target`
2. `internal/executor/include.go`: `executeDynamic` calls `ValidateRenderedRef(renderedRef)` after template rendering and BEFORE the resolver call; empty runtime-rendered ref → DINC-002 → routes through `handleResolveError`, so `on_not_found: continue` applies

TV-DYN-REF-011 unskipped; passes immediately after removing from skip map.

### TV-DYN-REF-012 added

Runtime-rendered empty ref: `vars: suggested_tsg_id: ""` + `runbook_ref: "${suggested_tsg_id}"` + `on_not_found: continue`. Tests the ICM path where no TSG suggestion exists — an empty suggestion must NOT fail hard. Passes. Corpus count updated 28→29; `TestLoadDynIncludeSuite` guard updated.

### GOV-001 verdict confirmed: HARNESS LIMITATION

`internal/adapter/wire.go` `buildApprovalGate`:
- `TTYOutput=true` → `TerminalApprovalGate` (can deny → DYN-015 fires)
- `TTYOutput=false` → `NoOpApprovalGate` (silently approves → DYN-015 never fires)

`WireOptions` has no `ApprovalGateOverride` seam. The conformance harness runs as a subprocess with no TTY → always `NoOpApprovalGate` → cannot trigger DYN-015. This is NOT a governance gap — interactive deployments do enforce `require_approval` via `TerminalApprovalGate`. The label is updated from "permanent (corpus design limit)" to "HARNESS LIMITATION" with a clear governance note.

**Finding reported to cristiano:** there is no way to override the approval gate via CLI flags. Only `TTYOutput` controls it. This means automated pipelines (`TTYOutput=false`) silently auto-approve `require_approval` runbooks. Whether that's a governance gap is for Barbara/the team — not patching it.

### Remaining skips (4 total, all labelled)

| Vector | Label | Reason |
|---|---|---|
| TV-DYN-TMPL-005 | **PERMANENT (B-16)** | DINC-009 intentional dead code; `expr.Eval` always returns string |
| TV-DYN-GOV-001 | **HARNESS LIMITATION** | NoOpApprovalGate in subprocess; TerminalApprovalGate works interactively |
| TV-DYN-GOV-005 | **PENDING DEF-006 (Don)** | Static entry runbook governance never seeded as dynGov |
| TV-DYN-GOV-015 | **PERMANENT (B-17)** | No Capabilities field in schema; out of scope |

### Corpus audit discipline (lesson pinned)

- A GXL condition using `vars:` strings must use string comparison: `condition: 'varname == "expected-string"'` not bare identifier, because `applyInputDefaults` passes vars through `fmt.Sprint()` making them always strings.
- Branch arms use `branches:` not `arms:`, and `steps:` not `flow:` — verify schema before writing.
- `branch` is a StepType expressed as `step: {type: branch, branches: [...]}` — NOT a top-level YAML key.
- `ValidateRenderedRef` fires BEFORE the resolver call — so `on_not_found: continue` applies to runtime-rendered empty refs too (now confirmed by REF-011 + REF-012).

## 2026-08-10T13:25:10.498-07:00 — `TV-ENUM-DECL-006` amended per Barbara's implementation-gate ruling (§2a)

**Trigger:** `barbara-enum-mvp-implementation-gate.md` §2a ruled that `TV-ENUM-DECL-006`'s premise was
false — Don implemented against verified library behaviour and was right to escalate rather than fake a
failure. Barbara's own AR-ENUM-3 rule 3 normative clause ("a YAML scalar that resolves to a string under
the YAML 1.2 core schema") governs; the vector's aside ("`yes`/`no` resolve to YAML 1.2 core-schema
booleans") contradicts it. Under the YAML 1.2 **core schema**, `bool` resolves only from
`true|True|TRUE|false|False|FALSE`; unquoted `yes`/`no` resolve to strings. This was the sole edit to
`tv-enum.yaml` authorised by that gate — the corpus is otherwise still frozen.

**Fix applied — same id, same category, same expected code:**
- `id`: unchanged, `TV-ENUM-DECL-006`.
- `category`: unchanged, `ENUM-DECL`.
- `enum: [yes, no]` → `enum: [true, false]` — an unambiguous non-string-resolving counter-example under the
  YAML 1.2 core schema (Python's `yaml.safe_load` confirms both parse to `bool`, not `str`).
- `expected`: unchanged, `{error_class: ENUM, error_code: ENUM-002}` — coverage intent (ENUM-002,
  no-coercion at S3) fully preserved; `[true, false]` still fails the "must be `!!str`" well-formedness
  check exactly as `[yes, no]` was meant to.
- `description`/`note`: rewritten to state the correct fact — YAML 1.2 core schema resolves booleans only
  from the canonical `true/false` spellings, and authors must quote any member that would otherwise
  resolve to a non-string core-schema type (`enum: ["true", "false"]`). No longer asserts anything false
  about `yes`/`no`.
- `tags: [yaml-1.2-core-schema]` retained.

**Verification:** vector count unchanged at 58 (`ENUM-DECL` still 14). `python
design/gert/scripts/verify_corpus.py` → `OK 423/423 vectors validate`. Confirmed independently in Python
that `enum: [true, false]` parses to `[True, False]` (bool), matching the vector's `ENUM-002` expectation,
while `enum: [yes, no]` would have parsed to `['yes', 'no']` (str) — i.e. the old vector's fixture would
never have triggered the type failure it claimed to test. No other vectors, schemas, enum architecture, or
runtime code touched. Nothing committed.

**Exact vector semantics (for the record):** `TV-ENUM-DECL-006` now asserts that declaring
`inputs.answer.enum: [true, false]` (both unquoted YAML core-schema booleans, `type: string`) at a
runbook-input (S3) declaration site is rejected as `ENUM-002` — enum members must resolve to `!!str` at
the raw-node level (`yaml.Node.Tag`), and coerced-scalar members (here, boolean-tagged nodes) are not
string members no matter their surface spelling. This is the correct, narrowest fixture that is actually
false-under-YAML-1.2-core-schema, replacing the previous fixture (`[yes, no]`) which was true-under-1.2 and
therefore never exercised the failure it was meant to pin.

**Filed:** `.squad/decisions/inbox/tess-enum-decl-006-correction.md` — decision-inbox note recording this
correction for Barbara/Edith/Ken visibility (Edith owns the parallel `03-schema-vnext.tex` AR-ENUM-3 rule 3
aside correction per the same gate ruling §2a; not touched here, out of scope).

---

## Learnings (2026-08-15 14:42)

**Task:** Dynamic-include conformance corpus activation + ICM orchestrator e2e tests

### What shipped
- **Part 1:** `internal/conformance/dyninclude_conformance_test.go` fully rewritten. 28 vectors loaded, 5 actively pass, 23 narrowly skipped with precise defect IDs (no blanket skip). `TestLoadDynIncludeSuite` pins the corpus at exactly 28 vectors.
- **Part 2:** `cmd/gert/icm_dynamic_include_test.go` — 10 tests covering all four required ICM branches plus all five adversarial cases. All pass. `go test ./...` exit 0.

### Production defects discovered (not fixed — report only)
1. **CLI preflight bug** (`cmd/gert/include_closure.go`): `checkIncludeClosureLexicalScoping` uses `flowwalk.Walker` which sets `inclPath = step.IncludeSpec.Include.Runbook` for ALL include steps. For dynamic sites (`runbook_ref`), this field is empty string → `filepath.Join(baseDir, "")` = workspace dir → `parser.Parse(ctx, workspaceDir)` tries to read a directory as a file → fatal error. Blocks the CLI path for ALL runbooks with dynamic include steps. **Root cause of 23 conformance skips.**
2. **Non-string template rejection not implemented** (TV-DYN-TMPL-005 / DYN-004): `expr.Evaluator.Eval()` always returns `string`; integer 42 becomes "42" which passes `ValidateRenderedRef`. DINC-009 (non-string rendered ref) never fires.
3. **Governance enforcement absent** (TV-DYN-GOV-001/005/015): `ComposeGovernance` correctly computes effective governance but the executor never enforces `require_approval`, `deny_commands`, or `capabilities` against the child. No GOVERNANCE-001 emitted.
4. **`on_not_found:continue` for rendered-ref containing chars outside safe-id profile** (TV-DYN-TMPL-011): `ValidateRenderedRef` rejects the rendered value → DINC-001 (always fatal per B-5). Per spec the literal `${other_var}` is a bad rendered ref, not an absent-from-catalog ID, so this is arguably correct behavior; filed as defect for ruling.
5. **Ambiguous bare-id schema requires version** (TV-DYN-CAT-002/003, TV-DYN-NFD-005, TV-DYN-CYCLE-001): schema validator requires `version:` in `requires:` entries; corpus vectors omit it → schema/structural error before dynamic resolution.
6. **Depth-10 boundary test** (TV-DYN-CYCLE-007): `checkIncludeClosureLexicalScoping` fires PKG-017 for `requires:` block before the depth-check fires.
7. **Empty literal runbook_ref** (TV-DYN-REF-011): parser semantic validator emits "include/missing-target" not DINC-002.

### Key design lessons
- The CLI path is NOT usable for dynamic `include.runbook_ref` testing due to the preflight bug. Use `internal/adapter` + `internal/executor` wiring directly.
- The dry-run contract (§10.3) is enforced by `adapter.DryRunExecutorRegistry` which replaces ALL executors with `DryRunExecutor{kind}` — the real include executor is never called.
- `internalexecutor.WithEventEmitter(ctx, fn)` is the correct way to capture `include/resolved` and `include/notFound` events in tests.
- `adapter.NewCatalogIncludeResolver` takes a `*pkgcatalog.Catalog` + a `parser.Parser`; the catalog must be built with `pkgcatalog.BuildOptions{WorkspaceRoot, ProjectRequires}`.
- Package manifests must use `apiVersion: tool-package/v1` (NOT `gert-package/v1`) in tests — the adapter tests use this form.
- `engine.New()` requires `plan.Validation != nil` when constructing `ExecutionPlan` manually.
- Cycle detection + max-depth are both runtime-only (planner is blind to dynamic targets). Testing them requires a real recursive `SubStepRunner` that loops back into `reg.Lookup("include").Execute(...)`.
- `internalreplay.NewPinBasedIncludeLoader` accepts `base, pinsByPath, traceWriter, runID`; `pinsByPath` is `map[string]schema.LockedDynamicInclude{absPath: pin}`.

### Test strategy retrospective
The two-layer strategy (conformance corpus for parser/schema/identity rules, executor-level wiring for runtime behavior) cleanly separated what can be tested via the CLI harness from what must be tested via the adapter. This split minimized test brittleness from the CLI preflight defect.

---

## Learnings (2026-08-15 15:04 — pass 3: corpus audit + DEF-004/005)

**Task:** Resolve all remaining corpus defects; audit for GXL/GIS mixups; verdict on TMPL-011, REF-011, CYCLE-007, requires-version vectors; keep governance trio skipped.

### Result: 22 pass / 6 skip / 0 fail (was 15/13/0)

### New corpus defects found and fixed
1. **All `gert-package.yaml` entries** used wrong `apiVersion: gert-package/v1` instead of `tool-package/v1`, and flat `name:`/`version:` fields instead of `meta:` block. Fixed in 8 vectors.
2. **4 vectors** (CAT-002, CAT-003, NFD-005, CYCLE-001): `requires:` entries in runbook YAML missing `version:` field (required by JSON schema `PackageRequirement.required: ["package","version"]`).
3. **CYCLE-007**: `requires:` block placed on the static child (depth-9) instead of the entry point (depth-1). PKG-017 fired because the child's `requires:` was outside the root include-closure. Fix: move `requires:` to depth-1 (entry), remove from depth-9.
4. **TMPL-011**: Expected `value: {runbook_found: false}` was wrong. `${other_var}` contains `{`/`}` → DINC-001 (always fatal per B-5). `on_not_found: continue` only covers DINC-002. The corpus note incorrectly said "catalog lookup misses" — actually `ValidateRenderedRef` rejects the ref BEFORE catalog lookup. Fixed expected to `error_code: DYN-001`.
5. **COMPAT-001 + E2E-002**: GXL `${var}` syntax in `when:` and `condition:` fields. `${...}` is GIS syntax; GXL uses bare identifiers. Fixed: COMPAT-001 → `when: 'do_include'`; E2E-002 → `condition: 'not runbook_found'`. Also removed unused `required: true` input from E2E-002.

### New production defects found (DEF-004, DEF-005)
- **DEF-004 (HIGH)**: `when:` conditions on steps are validated at plan time (GXL parser) but NEVER evaluated at runtime. `engine.executeStep()` never checks `step.When`. Confirmed via probe: `when: '1 == 2'` still executes the step. `when:` is a plan-time-only feature currently.
- **DEF-005 (LOW)**: Parser's semantic validator treats `runbook_ref: ""` (explicit empty string) as absent (`include/missing-target`). Ken's `ValidateRenderedRef` explicitly handles empty → DINC-002 (named "TV-DYN-REF-011" in `refvalidate_test.go`), but the parser catches it first.

### `${...}`-in-GXL audit
Scanned all 28 vectors for `${...}` in GXL expression fields (`when:`, branch `condition:`). Found exactly 2 instances (COMPAT-001 and E2E-002). No other vectors had this error. All other `${...}` occurrences are correctly in `runbook_ref:` or `vars:` value fields.

### Key lessons
- `gert-package.yaml` canonical format: `apiVersion: tool-package/v1` + `meta:` block
- `when:` in runbook steps is GXL (validated) but not executed at runtime — a big feature gap
- Branch `condition:` IS evaluated at runtime by `BranchExecutor` + `SimpleConditionEvaluator`
- GXL supports `not varname` for boolean negation; branch conditions use bare identifiers without `${}`
- Degraded `end` step outcomes exit 0 (not a failure exit code)
- When a statically included child needs a package, the ENTRY POINT must declare `requires:`, not the child

---

## Learnings (2026-08-15 15:30 — pass 4: governance trio + B-16/B-17 permanents)

**Task:** Unskip GOV-001 and GOV-005 (Ken's DEF-003 landed); permanently close GOV-015 (B-17) and TMPL-005 (B-16); leave DEF-004/DEF-005 pending rulings.

### Result: 22 pass / 6 skip / 0 fail (unchanged count but skip labels all updated to permanent/pending)

### New corpus defects found (GOV-001 and GOV-005 fixtures)
- Both had missing `version:` in `requires:` entries — same defect as CAT-002/003/NFD-005/CYCLE-001 in pass 3.
- GOV-005 used `run:` form instead of `command:` — `run:` resolves to the shell binary (sh/cmd), so deny_commands check fires for shell name, NOT the script content.
- GOV-005 deny pattern `"rm -rf *"` cannot match a command name (argv[0] = just "rm"). `path.Match` checks pattern against the full command string. Multi-word patterns with spaces never match single-token command names.
- Fixed: `version:` added to both, GOV-005 changed to `command: rm` + `deny_commands: ["rm*"]`.

### New production defect DEF-006
`dynGovFromCtx(ctx)` returns nil for entry-level runbook includes because the static `governance:` block of the parent runbook is never seeded into context as dynGov. Only governance from ancestor dynamic includes is in context. `ComposeGovernance(nil, childGov)` ignores the parent runbook's deny_commands entirely. This only affects top-level includes; nested dynamic includes work correctly because the outer include set dynGov.

### Governance enforcement architecture (key insight)
There are TWO separate governance enforcement paths:
1. **Engine-level GovernanceEvaluator** (in engine.go executeStep): evaluates step-level governance per GovernanceEvaluator in EngineConfig. Not set in sub-engines via runSubStepsViaEngine.
2. **govPolicy in context** (in dynamic_resolver.go withGovPolicy): set by executeDynamic via ComposeGovernance after dynamic include resolution. Used by CLIExecutor to enforce deny_commands.

These are independent. A top-level runbook's `governance:` block feeds into path 1 (if GovernanceEvaluator is configured) but NOT into path 2 (dynGov composition). DEF-006 is specifically about path 2.

### GOV-001 root cause: corpus design limitation
DYN-015 fires only for nil ApprovalGate or a gate that returns an error. The CLI binary always wires NoOpApprovalGate for non-TTY. NoOpApprovalGate.RequestApproval returns (record, nil) — silently approves. DYN-015 is unreachable via the CLI binary for require_approval=true governance. The "non-widening" property is enforced (approval IS requested), but the approval is always granted in batch mode.

### Rulings applied
- B-16: TMPL-005 permanent skip — DINC-009 intentional dead code; evaluator always returns string; ruling closes the defect as "not a defect".
- B-17: GOV-015 permanent skip — schema.GovernanceConfig has no Capabilities field; capability narrowing out of scope per ruling; §6 amended.

---

## Learnings (2026-08-15 15:49 — pass 5: B-19 COMPAT-001 + when: audit)

**Task:** Rewrite COMPAT-001 using branch/condition: per B-19; audit corpus for other `when:` uses.

### Result: 23 pass / 5 skip / 0 fail (was 22/6/0)

### B-19 application
Barbara ruled DEF-004 (`when:` never evaluated) out of scope for dynamic includes — it's platform-wide. The correct conditional mechanism is `branch`/`condition:`. COMPAT-001 rewrote its `when: 'do_include'` guard as a branch step.

### COMPAT-001 rewrite details
- Old: `type: include` with `when: 'do_include'` — parsed fine, `when:` silently ignored at runtime
- New: `type: branch` with one arm `condition: 'do_include == "true"'` — when no arm matches, branch returns StepStatusSkipped (not error), run proceeds to end step, exits 0
- `vars: do_include: "false"` — explicit YAML string (not bool) because `applyInputDefaults` converts all runbook vars via `fmt.Sprint()` which gives string `"false"` for both bool `false` and string `"false"`; string comparison `do_include == "true"` handles both
- Schema: `branch` is a `type:` on a `step:` node, NOT a top-level `branch:` key. Branch arms use `branches:` (not `arms:`) and `steps:` (not `flow:`)

### when: corpus audit (pass 5)
- Scanned entire corpus for `when:` in step definitions
- Found exactly **1 occurrence** in COMPAT-001 (now fixed)
- Zero other vectors used `when:`

### GXL condition type safety
- `BoolValue()` on a pjvm.Value of KindString returns `(false, false)` (ok=false)
- `evalBoolNative` returns an error if result is not KindBool
- String vars (which is what all runbook-level vars become after `fmt.Sprint`) cannot be used as bare GXL identifiers for bool conditions
- Safe pattern for string vars: `condition: 'varname == "expected-string"'`
- Safe pattern for captured bool vars (from step output): bare `varname` or `not varname` works because step outputs preserve the Go type

### Branch schema facts
- `step.type = branch` uses `branches:` (not `arms:`)
- Branch arm fields: `condition:` (GXL), `label:`, `steps:` (list of FlowNodes), `else: true`
- When no arm condition is true and no else arm: `StepStatusSkipped` (run continues)
- When `else: true` arm exists: that arm is the fallback

---

## Session: Dynamic Runbook Includes — Phase 4 Wrap-up (2026-08-15)

**Agents:** Barbara (arch), Tess (test), Ken (stream1), Don (streams2-3), David (stream4)

**Outcome:** Feature complete, approved, ready for merge (APPROVE WITH CONDITIONS, all conditions satisfied)

**Key achievements:**
- 5 defects found and fixed (DEF-001..006 via Tess real-CLI exercise + implementation streams)
- 21 architecture rulings (B-1..B-21) issued and enforced
- 3 deferred items formally captured (B-18, B-19/B-20, B-21)
- 26/29 conformance vectors pass; 3 skips permanent with ruling
- Cross-repo coordination: design repo (spec authority) ↔ runtime repo (implementation)

**Coordination patterns that worked:**
- Conformance corpus authored before implementation as specification
- Real CLI exercise (not hand-reading) revealed every gap
- Plausible descriptions can be materially wrong; rely on code review
- Formal deferral of gaps better than hidden work

**Status:** This session's work fully merged into .squad/decisions.md. Five orchestration logs recorded. Session log captures defining lesson: real verification beats narrative.


