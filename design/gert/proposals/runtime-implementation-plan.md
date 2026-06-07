# Runtime Implementation Plan: GXL/GIS/GCP Expression Engine (Greenfield)

**Author:** Barbara — Lead / Architect  
**Date:** 2026-06-07 (v2 — revised per rubber-duck critique)  
**Status:** PROPOSED (v2)  
**Target Repos:**
- P0: `ormasoftchile/gert-private` (corpus + spec hygiene)
- P1–P8: `ormasoftchile/gert` (Go runtime)

**Reference Design:** `ormasoftchile/gert-private` (this repo)

---

## Revision History

| Version | Date | Driver |
|---|---|---|
| v1 | 2026-06-07 | Initial greenfield plan |
| **v2** | **2026-06-07** | Rubber-duck critique: 5 blocking + 3 non-blocking findings. Brady approved bundling corpus/spec fixes as P0. All 8 findings addressed in this revision. |

### v1 → v2 Changes Summary

1. **Added P0** (corpus + schema + section-09 hygiene) in gert-private — landing as a PR closing #1 BEFORE any runtime work.
2. **Corrected PJVM number model** — single finite float64, no distinct int64 type (finding #6).
3. **Corrected GIS spec** — escape is `\${` not `$${`; GXL parser delegation for `${...}` end detection; per-segment optional chaining (finding #5).
4. **Corrected GCP vector count** — 41, not 43 (finding #3). Total remains 267.
5. **Expanded GCP scope** — split into parser + GDP resolver + capture service + executor refactor; 6–10 units not 2 (finding #4).
6. **Added parse-gate phase (P5)** — `ValidatedPlan` integration into planner was wrongly out-of-scope in v1 (finding #1).
7. **Replaced big-bang cutover with staged runtime routing** — engine-selection seam, per-subsystem cutover (finding #2).
8. **Defined cutover gate methodology** — benchmark command, dataset, thresholds, soak workload (finding #7).
9. **Honest effort estimate** — 32–49 work units, not 20 (finding #8).

---

## Supersedes

This plan **supersedes** [`design/gert/proposals/runtime-migration-plan.md`](./runtime-migration-plan.md) (ratified 2026-06-05).

**Why:** The migration plan assumed that Phases A–H were being executed against a codebase that already contained partial GXL/GIS/GCP machinery. Post-merge audit (2026-06-07) revealed this was false:

| Component | Migration plan assumed | Actual state in `ormasoftchile/gert` |
|---|---|---|
| GXL parser | Partially built (sketch commits) | **Missing.** Only parser reachable is `expr-lang/expr/parser`. |
| GXL evaluator | expr-lang adapter to be replaced | **Missing.** Runtime still calls `exprlang.Compile/Run` through `pkg/expr/expr.go` shim. |
| GIS interpolator | `text/template` → `${...}` migration | **Wrong syntax.** Uses Go `text/template` `{{...}}`; no `?.` chaining. |
| GCP resolver | Keyword-switch to be upgraded | **Missing.** Flat keyword-enum (`stdout`/`stderr`/`exit_code`). No path resolver. |

**Consequence:** Phase 2 is **greenfield implementation**, not incremental migration.

---

## A. Build Order

### Proposed Sequence

```
P0: Corpus + Schema + Spec Hygiene (gert-private — closes #1)
 ↓
P1: PJVM (corrected number model) + Error Model + Conformance Scaffold
 ↓
P2: GXL Parser (lexer + recursive descent + AST)
 ↓
P3: GXL Evaluator (AST walker + stdlib + clock)
 ↓  ← Parallelism window: GCP parser + GDP resolver ONLY (not full GCP)
P4: GIS Interpolator (corrected: \${ escape, GXL delegation, per-segment ?.)
 ↓
P5: Parse-Gate / ValidatedPlan (planner integration)
 ↓
P6: Staged Runtime Routing (engine-selection seam, per-subsystem cutover)
 ↓
P7: Full GCP Capture Service + Executor Refactor + Cross-Step Semantics
 ↓
P8: Performance Measurement + Soak + expr-lang Deletion
```

### Parallelism Window (P3 timeframe)

During P3 (GXL evaluator), a second engineer can begin **GCP parser + shared GDP resolver only**. This is safe because:
- GCP parser needs only PJVM (P1) and the GDP path grammar (a subset shared with GXL).
- GDP resolver walks PJVM trees — no dependency on the GXL evaluator.
- Full GCP integration (capture service, executor wiring, cross-step semantics) is deferred to P7 because it depends on the runtime routing seam from P6.

The full GCP capture service (P7) is **NOT parallelizable** with P3. It requires:
- The engine-selection seam (P6) so captures flow through the new path
- Step output model established by the evaluator (P3)
- Executor refactoring that touches production code

### Rationale for Ordering

| Dependency | Why |
|---|---|
| P0 before P1 | Harness cannot accept vectors that fail YAML lint or schema validation. P0 makes the corpus trustworthy. |
| P1 before everything in `gert` | Every component produces/consumes PJVM values; error codes are the conformance contract. |
| P2 before P3 | Evaluator walks an AST the parser produces. |
| P3 before P4 | GIS `${expr}` contains full GXL expressions (per `gis.ebnf` §2.4). The interpolator delegates to the evaluator for expression resolution. |
| P4 before P5 | Parse-gate must parse GIS templates; needs the GIS parser. |
| P5 before P6 | Runtime routing delivers `ValidatedPlan` to executors. Cannot route without the gate. |
| P6 before P7 | Full GCP integration touches executor capture logic. Must use the new routing seam, not the old path. |
| P7 before P8 | Can't benchmark or soak until all engines are wired. Can't delete expr-lang until new path is proven. |

### Engineer Assignment (2 engineers)

- **Engineer A (critical path):** P1 → P2 → P3 → P4 → P5 → P6 (assists P7) → P8
- **Engineer B:** P0 (anyone; can be A), joins at GCP parser/GDP resolver during P3 window → P7 (owns capture service + executor refactor)

---

## B. Side-by-Side Strategy with Staged Routing

### Chosen: Side-by-side with engine-selection seam + staged per-subsystem cutover

v1 proposed side-by-side with build-tag cutover. Rubber-duck finding #2 identified this as a big-bang trap: P1–P5 test new engines only through the harness, then P6 flips 12 executors + capture logic + fixture syntax + tests simultaneously.

**v2 strategy — staged runtime routing:**

1. **Engine-selection seam (P6):** Introduce an internal `EngineRouter` interface that dispatches expression evaluation, interpolation, and capture resolution to either old (`internal/expr/`) or new (`pkg/gxl/`, `pkg/gis/`, `pkg/gcp/`) engines per operation type.

2. **Per-subsystem cutover (P6 sub-PRs):**
   - **P6a:** Route GXL condition evaluation through new parser+evaluator for all executors. Old engine remains for interpolation and capture. CI runs both paths.
   - **P6b:** Route GIS interpolation through new interpolator for string fields. Old `text/template` path remains for capture.
   - **P6c:** Route GCP capture through new capture service for one executor family (cli), then extend to tool, http_call, include, noop.

3. **Parallel CI validation:** During P6a–P6c, CI matrix runs:
   - Default build: old path (must pass existing 58 tests)
   - `gxl` build tag: new path (must pass 267/267 conformance + fixture suite)
   - Both builds run on every PR.

4. **expr-lang deletion (P8):** Only after the new path has been exercised under production routing for ≥2 merged PRs per subsystem AND the soak passes.

### Why staged routing, not build-tag-only

| Concern | Build-tag-only (v1) | Staged routing (v2) |
|---|---|---|
| Production exercise before cutover | None — harness only | Each subsystem exercises production codepath before next subsystem cuts over |
| Bisect on failure | 12 executors flip at once | Subsystem-level isolation |
| CI coverage of new path | Conformance harness only | Conformance + fixture + integration tests under `gxl` build |
| Risk of big-bang | High (finding #2) | Low — incremental |

### Rejected: Replace-in-place

Big-bang risk. 12 executor types + 58 tests flip simultaneously. The runtime currently works — breaking it during development is unacceptable.

### Rejected: Runtime feature flag

Overhead, branching logic in hot paths, must exercise both paths in CI indefinitely. Build tag is cheaper for the parallel-CI period; the engine-selection seam is internal wiring, not user-facing config.

---

## C. Per-Component Specification

### C.0 Corpus + Schema Hygiene (P0 — gert-private)

| Attribute | Value |
|---|---|
| **Repo** | `ormasoftchile/gert-private` (this repo) |
| **Scope** | **Split between Tess and Barbara (2026-06-07 Brady coordination decision).** Tess: Fix 5 corpus YAML files, reconcile `schema.json`, add `make verify-corpus` target. Barbara: Add corpus interpretation contract to section 09 (subsection "Conformance Corpus Interpretation Contract"). Both execute in parallel. |
| **Deliverables** | See §E P0 for full list |

### C.1 PJVM Package (P1 — corrected)

| Attribute | Value |
|---|---|
| **Package path** | `internal/pjvm/` |
| **Public facade** | `pkg/pjvm/` exports `Value` interface, concrete types, `FromAny(v any) Value` conversion, equality/comparison/truthiness. |
| **Number model** | **Single representation: finite float64.** No NaN, no Infinity, no distinct int64 type. This matches the PJVM spec in `gis.ebnf` §4.1 and section 03b. Integer-valued floats (42.0) are representable but are still float64. |
| **Conversion rules** | `FromAny()`: Go `int`/`int64`/`uint` → `float64` (error if outside ±2^53 safe integer range). Go `float64` NaN/Infinity → `Null` + `PJVM-CONVERT-001` warning. JSON `number` → `float64`. YAML integer → `float64`. |
| **Canonical serialization** | `ToString()`: integer-valued → no decimal ("42"); non-integer → minimal digits ("3.14"); negative zero → "0"; ≥1e21 → scientific ("1e+21"). Matches ECMAScript `Number::toString`. `ToJSON()`: RFC 8259 number format; objects with sorted keys. |
| **Equality/Comparison** | `==` is value equality on the PJVM type. `<`/`>` defined for numbers and strings only; all other type pairs → `GXL-TYPE-001`. |
| **Truthiness** | `null` → false, `false` → false, `0` → false, `""` → false, empty array → false, empty object → false. All other values → true. |
| **Files** | `value.go`, `convert.go`, `compare.go`, `truthiness.go`, `serialize.go` |
| **Tests** | Exhaustive unit tests: all YAML unmarshal edge cases, large integers, nested nulls, NaN/Infinity rejection, negative zero, round-trip serialization. |
| **Estimated size** | ~5 files, ~500 lines. |

### C.2 GXL Parser (P2)

| Attribute | Value |
|---|---|
| **Package path** | `internal/gxl/parser/` |
| **Public facade** | `pkg/gxl/` exports `Parse(input string) (*ast.Expr, error)` |
| **Internal files** | `lexer.go`, `token.go`, `parser.go`, `ast.go`, `errors.go` |
| **AST nodes** | `BinaryExpr`, `UnaryExpr`, `CallExpr` (namespace + method), `PathExpr` (GDP with per-segment optional flag), `Literal` (bool/number/string/null), `Ident`, `BracketAccess`, `DotAccess`, `OptionalDotAccess`, `OptionalBracketAccess` |
| **Error model** | Typed `ParseError{Code string, Pos Position, Message string}`. Codes: `GXL-PARSE-001`..`GXL-PARSE-010`, `GXL-BANNED-*` (per §03d error catalog). |
| **Test approach** | Unit tests per production rule + 83/83 tv-gxl-parse vectors as integration bar |
| **Estimated size** | ~6 files, ~1200 lines. |

### C.3 GXL Evaluator (P3)

| Attribute | Value |
|---|---|
| **Package path** | `internal/gxl/eval/` |
| **Public facade** | `pkg/gxl/` exports `Eval(expr *ast.Expr, env *Env) (pjvm.Value, error)` and `EvalBool(input string, vars map[string]any) (bool, error)` |
| **Internal files** | `evaluator.go`, `stdlib.go` (str/list/regex/math/len), `clock.go` (Clock interface + SystemClock + FixedClock), `env.go` (variable binding, map→PJVM conversion), `errors.go` |
| **PJVM value type** | Uses `pjvm.Value` with corrected number model (finite float64 only — no int64 discriminated union). |
| **GDP path resolution** | Variable access uses GDP paths. The evaluator imports the shared GDP resolver from `internal/gdp/` (see C.6). |
| **Error model** | Typed `EvalError{Code string, Pos Position, Message string}`. Codes per §03d error catalog. |
| **Test approach** | Unit tests per stdlib function + per operator + 92/92 tv-gxl-eval + 36/36 tv-gxl-path vectors |
| **Estimated size** | ~8 files, ~2000 lines. |

### C.4 GIS Interpolator (P4 — corrected)

| Attribute | Value |
|---|---|
| **Package path** | `internal/gis/` |
| **Public facade** | `pkg/gis/` exports `Interpolate(template string, env *gxl.Env) (string, error)` |
| **Internal files** | `scanner.go`, `resolver.go`, `interpolator.go`, `errors.go` |
| **Escape handling** | `\${` is the escape for literal `${` (per `gis.ebnf` §2.3, OI-GIS-01 ratification). `$${` is **NOT** a recognized escape — parsers MUST reject it. `\\` → literal `\`. `\n`, `\r`, `\t` → corresponding control characters. Unknown `\X` → pass-through with `GIS-PARSE-002` warning. |
| **Expression end detection** | The GIS scanner does NOT use naive brace counting to find the closing `}` of `${...}`. It delegates the content inside `${` to the GXL parser, which returns when it encounters the end-of-expression signal (unmatched `}` at expression depth 0). This correctly handles string literals containing `}` and nested grouping. Per `gis.ebnf` §2.4 and OI-GIS-04. |
| **Optional chaining model** | Per-segment optional chaining, NOT a single `optional bool` flag on the expression. Each GDP segment in a GIS path can independently be `?.` (OptionalDotAccess) or `?.[N]` (OptionalBracketAccess). The AST represents this as `[]GISPathSegment` where each segment carries its own optional flag. Once any optional segment misses, the entire remaining tail short-circuits to `""`. Per `gis.ebnf` §2.5. |
| **Rejected forms** | `${}` → `GIS-PARSE-004`. `$${foo}` → parse error (not a recognized escape). `${?.root}` → root identifier may not be optional. |
| **Error model** | `GIS-PARSE-001`..`GIS-PARSE-004`, `GIS-PATH-MISSING`, `GIS-TYPE-001`, `GIS-TYPE-002`. |
| **Test approach** | Unit tests for scanner edge cases + 15/15 tv-gis-path vectors |
| **Estimated size** | ~5 files, ~900 lines. |

### C.5 Parse-Gate / ValidatedPlan (P5)

| Attribute | Value |
|---|---|
| **Package path** | `internal/planner/` (extends existing planner in `ormasoftchile/gert`) |
| **Integration target** | The existing planner (`internal/planner/planner.go`) currently calls `flowwalk.Walk()` and returns `engine.ExecutionPlan`. P5 adds a validation pipeline BETWEEN `Walk()` and the return. |
| **Validation pipeline** | Per spec §03d: (1) YAML load (already done by `pkg/parser`), (2) GXL parse all boolean-expression fields, (3) GIS parse all string-template fields, (4) GCP parse all capture paths, (5) semantic validation (variable scope, undefined-capture detection, `GCP-DEFAULT-SUBTREE` prohibition, grammar-version recording). |
| **ValidatedPlan type** | Unexported struct `validatedPlan` in `internal/planner/` with no exported constructor other than `Plan()`. Fields per §03d: `runbook_id`, `runbook_hash` (SHA-256), `grammar_versions` (gxl/gis/gcp version strings), `validated_at`, `expression_count` (gxl/gis/gcp counts), empty `errors` list. |
| **plan.validated event** | Emitted as the first event after plan load, before any step event. Payload includes runbook_id, runbook_hash, grammar_versions, validated_at, expression_count. Per §03d and §07. |
| **Execution blocking** | `RunHandle.Next()` MUST NOT execute any step from a plan that has not been validated. If the plan is not a `validatedPlan`, the runtime refuses with a hard error before any step dispatches. |
| **Planner files touched** | `planner.go` (add validation call), new `validate.go` (pipeline), new `validated_plan.go` (type), new `validate_test.go`. |
| **Test approach** | Integration tests: valid runbook → `plan.validated` event emitted → steps execute. Invalid GXL/GIS/GCP expression → `PLAN-001` error → no steps dispatched. Invalid capture variable reference → `PLAN-002` → blocked. |
| **Estimated size** | ~4 new files, ~800 lines. |

### C.6 Shared GDP Resolver

| Attribute | Value |
|---|---|
| **Package path** | `internal/gdp/` |
| **Purpose** | Shared GDP (GERT Dotted Path) parser and resolver used by GXL evaluator (variable access), GIS interpolator (path resolution), and GCP resolver (capture path suffixes). Extracted to avoid triple-implementation divergence (v1 risk #5, now a design decision). |
| **Grammar source** | GDP is defined in `gxl.ebnf` §3 (GXL Primary → GDP) and extended in `gis.ebnf` §2.5 (optional chaining). GCP uses the same GDP suffix grammar after source prefix stripping. |
| **API** | `Parse(input string) ([]Segment, error)` — parses a dotted path into segments. `Resolve(segments []Segment, root pjvm.Value) (pjvm.Value, error)` — walks a PJVM tree. |
| **Segment types** | `RootIdent{name}`, `DotAccess{field}`, `BracketAccess{index}`, `OptionalDotAccess{field}`, `OptionalBracketAccess{index}`. |
| **Estimated size** | ~3 files, ~400 lines. |

### C.7 GCP Capture Service (P7 — expanded scope)

v1 budgeted 2 work units for "GCP resolver (GDP paths on PJVM trees)." This was materially under-scoped (finding #4). The actual GCP implementation requires:

| Sub-component | Description | Depends on |
|---|---|---|
| **GCP parser** | Parses capture path strings into `CapturePath` AST. Source prefix recognition (`exit_code`, `stdout`, `stderr`, `json`, `yaml`, `http.*`, `event.*`, `step.*`). Header name handling. Error codes `GCP-PARSE-001`..`GCP-PARSE-006`. | PJVM (P1), GDP resolver (C.6) |
| **GDP resolver** | Shared with GXL/GIS — see C.6. Resolves GDP suffix on PJVM trees. | PJVM (P1) |
| **Step output model** | Defines how each executor type produces its output as PJVM values. CLI: `{stdout: string, stderr: string, exit_code: number}`. HTTP: `{status: number, body: string, headers: {name: string}}`. Tool: PJVM value from tool output. Include: passthrough. Noop: empty. JSON/YAML parsing of stdout into PJVM tree. | PJVM (P1) |
| **Capture service** | Orchestrates: (1) look up source prefix, (2) obtain raw output from step output model, (3) parse JSON/YAML if structured source, (4) resolve GDP suffix, (5) classify scalar vs subtree, (6) apply `capture_defaults` policy, (7) store into run variable context. | GCP parser, GDP resolver, step output model |
| **Executor refactor** | Current capture logic is scattered across `cli`, `tool`, `include`, `noop`, `http_call` executors with keyword switches (`stdout`/`stderr`/`exit_code`). Refactor all executor capture blocks to delegate to the capture service. | Capture service, engine-selection seam (P6) |
| **Planner/default validation** | `GCP-DEFAULT-SUBTREE`: `capture.default` declared for subtree capture → plan-time error. `GCP-TYPE-001`: default value type incompatible with capture path. `GCP-PARSE-006`: `step.{id}.*` references undefined step ID → plan-time rejection. | ValidatedPlan (P5) |
| **Cross-step semantics** | `step.{id}.stdout`, `step.{id}.json.*` — look up output of a previously executed step by step ID. Requires step output storage in run context. | Capture service, executor refactor |
| **HTTP semantics** | `http.status`, `http.body`, `http.body.{gdp}`, `http.headers.{name}` — case-insensitive header lookup, JSON body parsing. | Capture service |
| **Event semantics** | `event.id`, `event.body`, `event.body.{gdp}`, `event.headers.{name}` — event payload access for event-driven runbooks. | Capture service |

**What can parallelize with P3:** GCP parser + GDP resolver only (pure parsing, no production wiring).

**What CANNOT parallelize with P3:** Capture service, executor refactor, cross-step/HTTP/event semantics. These require the engine-selection seam (P6) and production runtime context.

| Attribute | Value |
|---|---|
| **Package path** | `internal/gcp/` (parser, resolver), `internal/capture/` (capture service), `internal/gdp/` (shared) |
| **Public facade** | `pkg/gcp/` exports `ParseCapturePath`, `ResolveCapture`, `ResolveCaptures` |
| **Error model** | `GCP-PARSE-001`..`GCP-PARSE-006`, `GCP-RESOLVE-001`..`GCP-RESOLVE-005`, `GCP-DEFAULT-SUBTREE`, `GCP-TYPE-001` |
| **Test approach** | Unit tests per source prefix + 41/41 tv-gcp-path vectors + integration tests for cross-step and HTTP capture |
| **Estimated size** | ~10 files, ~1800 lines across `internal/gcp/` + `internal/capture/` + `internal/gdp/`. |

---

## D. Conformance Harness Integration

### Background

Fido's PR #20 in `ormasoftchile/gert` attempted a full conformance harness but is broken because there's no engine to dispatch to. Issue #17 tracks the harness requirement.

### Prerequisite: P0 Corpus Hygiene

**Before** the harness can load vectors, the corpus must be valid (finding #3). P0 fixes:
- All 5 `tv-*.yaml` files pass strict YAML lint
- `schema.json` declares every field that real vectors use
- `make verify-corpus` runs clean
- Exact vector counts: 92 eval + 83 parse + 36 path + 15 GIS + 41 GCP = **267**

### Strategy: Incremental Harness Growth

1. **P1 delivers the scaffold:** `internal/conformance/harness_test.go` loads all 267 vectors, dispatches each to a runner interface, initially marks all as `t.Skip("engine not implemented")`.

2. **Each subsequent phase un-skips its vectors:**
   - P2 → `tv-gxl-parse` (83 vectors)
   - P3 → `tv-gxl-eval` (92) + `tv-gxl-path` (36)
   - P4 → `tv-gis-path` (15)
   - P7 → `tv-gcp-path` (41)

3. **Definition of "267/267":** No skips. No sanitization. No input normalization. Schema-valid inputs only. Exact vector counts as listed. The harness MUST record grammar versions used during evaluation and include them in the test report.

4. **PR #20 disposition:** Close it. The scaffold in P1 replaces its intent.

---

## E. Phased Plan with Acceptance Gates

### P0: Corpus + Schema + Spec Hygiene (gert-private)

| Attribute | Detail |
|---|---|
| **Repo** | `ormasoftchile/gert-private` (this repo) |
| **Scope (Tess)** | (1) Fix all 5 corpus YAML files for valid YAML: quote `description:` fields containing special characters (`:`, `#`, `{`, `}`); explicit string quoting where the intent is the literal string `"null"` (not YAML null). (2) Reconcile `schema.json` to declare every field that real vectors use: `expected.assert`, `expected.type`, `expected.pattern`, `expected.timing`, and any other key paths found by auditing all 5 corpora. (3) Add a Make target `make verify-corpus` that runs strict YAML lint (e.g., `yamllint --strict`) + JSON Schema validation against all 267 vectors and fails on any error. |
| **Scope (Barbara)** | (4) Add a corpus interpretation contract to `design/gert/sections/09-testing-and-acceptance.tex` (new subsection "Conformance Corpus Interpretation Contract"): normative prose covering string output quoting semantics, null vs empty-string distinction, schema validation, grammar version recording, and no-silent-skips policy. |
| **Execution** | Tess and Barbara execute in parallel (this spawn IS the parallel execution; Tess runs `tess/p0-corpus-hygiene` while Barbara executes section 09 contract). |
| **Acceptance criterion** | `make verify-corpus` passes. All 267 vectors validate against `schema.json`. Zero YAML lint warnings in strict mode. PR closes `gert-private#1`. |
| **Lands** | As a PR in this repo, merged BEFORE any P1 work in `ormasoftchile/gert`. |
| **Work units** | Tess: 2–4; Barbara: (included in this task) |

### P1: PJVM + Error Model + Conformance Scaffold

| Attribute | Detail |
|---|---|
| **Repo** | `ormasoftchile/gert` |
| **Scope** | `internal/pjvm/` (Value types with corrected number model: single finite float64, no int64 discriminated union, no NaN, no Infinity), `internal/conformance/` (harness scaffold, vector loader, schema validation), `testdata/vectors/` (vendored copy + sync script per DRIFT-DETECTION-001), error code registry. |
| **PJVM normative requirements** | Single number = finite float64. Explicit conversion rules from YAML/JSON/`map[string]any`. Canonical string serialization (integer-valued → no decimal; ≥1e21 → scientific; negative zero → "0"). Canonical JSON serialization (sorted object keys). Equality/comparison/truthiness tests per §4. No NaN/Infinity — `FromAny()` maps these to `Null` + `PJVM-CONVERT-001` warning. |
| **Acceptance criterion** | `go test ./internal/pjvm/...` green (all type operations including float64 edge cases, large integer conversion, NaN/Infinity rejection, round-trip serialization). Harness loads 267 vectors without panic. All vectors skip cleanly. `make verify-vectors` passes in CI. |
| **Work units** | 4–5 |
| **Cutover decision** | None — pure addition. |

### P2: GXL Parser

| Attribute | Detail |
|---|---|
| **Scope** | `internal/gxl/parser/` (lexer, parser, AST, errors), `pkg/gxl/` (public `Parse` function). GDP path parsing for GXL variable access. |
| **Acceptance criterion** | 83/83 tv-gxl-parse vectors green (un-skipped and passing). Zero regressions in default build. |
| **Work units** | 5–7 |
| **Cutover decision** | None — parser has no production consumers yet. |

### P3: GXL Evaluator ∥ (GCP parser + GDP resolver)

| Attribute | Detail |
|---|---|
| **Scope (Engineer A)** | `internal/gxl/eval/` (evaluator, stdlib, clock, env), `pkg/gxl/` (public `Eval`/`EvalBool`). GDP path resolution for variable access via shared `internal/gdp/`. |
| **Scope (Engineer B, parallel)** | `internal/gcp/parser.go` (GCP path parser — source prefix recognition, header names, error codes `GCP-PARSE-001`..`GCP-PARSE-005`). `internal/gdp/` (shared GDP resolver — if not already extracted by Engineer A). |
| **Acceptance criterion (A)** | 92/92 tv-gxl-eval + 36/36 tv-gxl-path vectors green. `now()` tested with FixedClock. Zero regressions in default build. |
| **Acceptance criterion (B)** | GCP parser unit tests pass for all source prefixes (local, step, http, event). GDP resolver unit tests pass. GCP parser can parse all 41 tv-gcp-path vector inputs without error (evaluation deferred to P7). |
| **Work units** | A: 5–7 evaluator, B: 3–4 GCP parser + GDP resolver |
| **Cutover decision** | None — no production consumers yet. |
| **Spec blockers** | ✅ RESOLVED (2026-06-07): Truthy() semantics for empty collections — Option (c) Strict. Non-bool in boolean context → `GXL-TYPE-002`. No implicit truthy/falsy coercion. See [`.squad/decisions/inbox/barbara-truthy-arbitration.md`](../../../.squad/decisions/inbox/barbara-truthy-arbitration.md) and spec §03a §Truthy Coercion in Boolean Context. |

### P4: GIS Interpolator

| Attribute | Detail |
|---|---|
| **Scope** | `internal/gis/` (scanner with corrected `\${` escape, GXL parser delegation for expression-end detection, per-segment optional chaining), `pkg/gis/` (public `Interpolate`). |
| **Acceptance criterion** | 15/15 tv-gis-path vectors green. Integration tests: mixed literal + expression segments, optional chain miss → empty string at segment level, `\${` escape produces literal `${`, `$${` is rejected, nested `${str.trim("${foo}")}` handled correctly via GXL delegation, empty `${}` → `GIS-PARSE-004`. |
| **Work units** | 4–6 |
| **Cutover decision** | None — no production consumers yet. |

### P5: Parse-Gate / ValidatedPlan

| Attribute | Detail |
|---|---|
| **Scope** | Extend `internal/planner/planner.go` with validation pipeline per §03d. New files: `validate.go`, `validated_plan.go`, `validate_test.go`. Integration with existing planner's `Plan()` function. |
| **Validation pipeline** | (1) GXL parse all `when`/`condition`/`until`/`iterate.until`/`branches[].condition`/`include.when`/`collector.fields[].when` fields. (2) GIS parse all `title`/`args`/`stdin`/`tool.argv[]`/`display.content`/`assert[].subject`/`assert[].expected`/artifact paths/`include.with` values. (3) GCP parse all `capture:` map values. (4) Semantic validation: variable scope, undefined-capture detection, `GCP-DEFAULT-SUBTREE`, grammar-version recording. |
| **Output** | `validatedPlan` (unexported) wrapping `engine.ExecutionPlan` + grammar versions + expression counts + runbook hash + validation timestamp. |
| **Acceptance criterion** | Valid runbook → `plan.validated` event emitted → steps execute. Invalid GXL expression → `PLAN-001` → no steps dispatched. Invalid GIS template → `PLAN-001` → blocked. Invalid capture path → `PLAN-001` → blocked. Undefined capture variable → `PLAN-002` → blocked. `GCP-DEFAULT-SUBTREE` → `PLAN-003` → blocked. Forbidden `{{ }}` syntax → `PLAN-004`. Forbidden infix operator → `PLAN-005`. |
| **Work units** | 3–5 |
| **Cutover decision** | This phase modifies the planner's return type. The `ValidatedPlan` wrapper is internal, so existing callers continue to work. The run-start function is updated to require `validatedPlan` — this is the governance checkpoint. |

### P6: Staged Runtime Routing

| Attribute | Detail |
|---|---|
| **Scope** | Introduce `EngineRouter` interface in `internal/engine/`. Wire per-subsystem routing. CI matrix for both build paths. |
| **P6a — GXL conditions** | Route all `when`/`condition`/`until` evaluation through `pkg/gxl.EvalBool`. Old expr-lang path remains for interpolation and capture. |
| **P6b — GIS interpolation** | Route all string-template fields through `pkg/gis.Interpolate`. Old `text/template` path remains for capture. |
| **P6c — GCP capture (staged)** | Route `cli` executor capture through new `pkg/gcp.ResolveCaptures`. Verify. Extend to `tool`, `http_call`, `include`, `noop`. |
| **P6 CI** | CI matrix: default build (old path, 58 tests green) + `gxl` build (new path, 267/267 conformance + fixture suite green). Both must pass on every PR. |
| **Acceptance criterion** | Each sub-PR (P6a, P6b, P6c) passes CI on both build paths. No regressions in default build. New path passes its conformance subset. |
| **Work units** | 3–5 |
| **Cutover decision** | No final cutover here — that's P8. Each P6 sub-PR is a partial cutover for one subsystem. |

### P7: Full GCP Capture Service + Executor Refactor

| Attribute | Detail |
|---|---|
| **Scope** | `internal/capture/` (capture service orchestration), executor refactor (cli, tool, http_call, include, noop — delegate capture to service), cross-step semantics (`step.{id}.*`), HTTP capture semantics (`http.status`/`http.body`/`http.headers.*`), event capture semantics (`event.id`/`event.body`/`event.headers.*`), `capture_defaults` policy enforcement, scalar vs subtree classification, step output model / PJVM conversion, step output storage for cross-step lookups. **POLICY (Brady 2026-06-07): Whenever P7 implementers discover capture-logic divergences between the current executors and the GCP spec, the answer is document-and-fix: bring the behavior into spec conformance. The GCP spec is the contract. Backwards compatibility with existing accidental behavior is NOT a constraint.** |
| **Discovery contract** | P7 implementers MUST log every capture-logic divergence they find during executor refactoring — what the code currently did, what the spec says, and what they changed it to — in the PR body with the `P7-divergence:` tag. The fix is mandatory; the spec is the contract. |
| **Acceptance criterion** | 41/41 tv-gcp-path vectors green (fully evaluated, not just parsed). Integration tests for cross-step capture, HTTP header case-insensitivity, JSON/YAML body parsing, bare-root capture, `GCP-DEFAULT-SUBTREE` rejection, `GCP-RESOLVE-001` (step not yet executed). All executor capture paths routed through capture service. All divergence fixes documented in PR body. |
| **Work units** | 6–10 |
| **Cutover decision** | After P7, the new GCP path is exercised through the engine-selection seam (P6c). |

### P8: Performance Measurement + Soak + expr-lang Deletion

| Attribute | Detail |
|---|---|
| **Scope** | Performance benchmark, soak workload methodology, expr-lang deletion, build-tag removal. |
| **Benchmark methodology** | Command: `go test -bench=BenchmarkEngine -benchmem -count=5 ./internal/conformance/...` against a benchmark dataset comprising: (a) all 267 conformance vectors evaluated in sequence, (b) all 22 fixture runbooks planned and executed. Baseline: last commit before P6a (the final commit where expr-lang serves all production paths). |
| **Per-category thresholds** | GXL parse: ≤2× baseline ns/op. GXL eval: ≤2× baseline ns/op. GIS interpolation: ≤2× baseline ns/op. GCP resolution: ≤3× baseline ns/op (new capability — no direct baseline; threshold is vs synthetic equivalent). Full fixture run: ≤2× baseline wall-clock. |
| **Allocation/memory regression cap** | ≤1.5× allocs/op vs baseline. ≤2× bytes/op vs baseline. |
| **Race/panic-free requirement** | `go test -race ./...` passes with zero data races, zero panics, for the full test suite including all fixture runbooks. |
| **Soak workload methodology** | Soak acceptance criteria are per-category performance thresholds + race/panic-free requirement (above). The *soak infrastructure choice* (CI capacity vs dedicated runner, scheduled frequency, etc.) is DEFERRED by Brady (2026-06-07) to post-P7. P8 begins with a placeholder soak workload (e.g., 22-fixture suite + 267-vector suite run in-band during CI). When the infrastructure decision lands, runtime engineers can swap the workload and infrastructure without changing the acceptance criteria. Soak acceptance remains: zero failures across all scheduled runs. |
| **expr-lang deletion criteria** | All of: (1) soak passes per methodology above, (2) new path has been exercised under production routing for ≥2 merged PRs per subsystem (conditions, interpolation, capture), (3) 267/267 conformance, (4) all fixture runbooks green, (5) performance within thresholds. Only then: delete `internal/expr/`, drop `expr-lang/expr` from `go.mod`, remove `//go:build gxl` tag, remove `EngineRouter` (new engines become the only path). |
| **Acceptance criterion** | `go.sum` no longer references `expr-lang`. Single build path. CI green. Performance report committed to `docs/perf/`. |
| **Work units** | 3–5 |

### Summary

| Phase | Work Units | Cumulative (low) | Cumulative (high) | Parallelizable? | Repo |
|---|---|---|---|---|---|
| P0 | 2–4 | 2 | 4 | — | gert-private |
| P1 | 4–5 | 6 | 9 | — | gert |
| P2 | 5–7 | 11 | 16 | — | gert |
| P3 (eval) | 5–7 | 16 | 23 | GCP parser ∥ | gert |
| P3 (GCP parser+GDP) | 3–4 | 16 | 23 | Yes (with P3 eval) | gert |
| P4 | 4–6 | 20 | 29 | — | gert |
| P5 | 3–5 | 23 | 34 | — | gert |
| P6 | 3–5 | 26 | 39 | — | gert |
| P7 | 6–10 | 32 | 49 | — | gert |
| P8 | 3–5 | 35 | 54 | — | gert |
| **Total** | **38–58** | — | — | Critical path: **32–49** (GCP parser absorbed into P3 window) | |

With 2 engineers and P3 parallelism: **~25–38 calendar work-days** (5–8 weeks with buffer).

> **Honesty note (finding #8):** v1 estimated 20 units. This was optimistic. The honest range is 32–49 critical-path units. The increase comes from: P0 (2–4 units, new), PJVM normative work (+1–2), GCP full scope (+4–8), parse-gate (+3–5), staged routing (+3–5), defined soak (+1–2).

---

## F. Risks + Mitigations

| # | Risk | Impact | Likelihood | Mitigation |
|---|---|---|---|---|
| 1 | **Spec ambiguity discovered mid-build** | Blocks implementation until Barbara arbitrates. | Medium | 24h arbitration SLA during P2–P7. Spec is blocking — don't guess. |
| 2 | **Performance regression vs expr-lang** | expr-lang is JIT-compiled Go; our interpreter is an AST walker. Could be 5–10× slower. | Medium | P8 cutover gate with defined per-category thresholds. If breached: add bytecode compilation before cutover. |
| 3 | **Error-message divergence breaks CLI tests** | Existing CLI tests assert exact error text from expr-lang. | High (certain) | Addressed in P6 sub-PRs. Risk is test rewrite volume — staged routing makes each sub-PR smaller. |
| 4 | **PJVM type interop at language boundaries** | Executors pass `map[string]any` from YAML. Edge cases: large integers, nested nulls, NaN. | Medium | P1 builds `FromAny()` with exhaustive tests + explicit ±2^53 range checking. |
| 5 | **GDP path grammar divergence across engines** | Three engines share GDP but might drift. | Low | `internal/gdp/` is a shared package from the start (v2 design decision). |
| 6 | **Corpus issues block P1** | If P0 is incomplete, harness can't load vectors. | Low | P0 lands first, in this repo, with `make verify-corpus` as the gate. |
| 7 | **Staged routing adds temporary complexity** | Engine-selection seam is extra code during transition. | Medium | Seam is deleted in P8. It's internal wiring with no public API surface. Acceptable cost for safe cutover. |
| 8 | **GCP executor refactor scope creep** | Capture logic is scattered across 5+ executor types with keyword switches. | Medium | P7 is budgeted at 6–10 units (honest). Capture service is a clean abstraction that replaces scattered logic. |

---

## G. Out of Scope — Explicit

This plan does **NOT** cover:

- **C# runtime implementation** — separate repo, separate plan, consumes same conformance vectors. C# will implement against the same PJVM spec, grammars, and §03d parse-gate contract. No code sharing with Go beyond the design artifacts in this repo.
- **TypeScript runtime implementation** — same rationale as C#.
- **Web platform integration** (App Service, Cosmos DB, Durable Functions) — architecture exists in `design/web-platform/`; no runtime dependency on GXL engine choice. Web platform consumes the C# runtime, which consumes the same design artifacts.
- **CLI UX changes** beyond error-message taxonomy — no new commands, no new flags. The error-message format change (expr-lang error strings → GXL/GIS/GCP error codes) is within scope (P6); new CLI features are not.
- **The 46 unrelated Windows-portability test failures** Hudson triaged — separate issue track. These are path-separator and line-ending issues unrelated to expression engines. They should not block or be conflated with P0–P8.
- **Grammar evolution** — `gxl.ebnf`, `gis.ebnf`, `gcp.ebnf` are frozen for the duration of this plan. Any grammar changes require a separate proposal + conformance vector update FIRST.
- **Migrator tooling** — Stream E was removed per scope decision (2026-06-04). Users migrating `{{ }}` → `${...}` do so manually with CHANGELOG guidance.
- **Performance optimization beyond thresholds** — bytecode compiler, partial evaluation, caching. Deferred unless P8 gate is breached.

### Why these are out of scope

| Exclusion | Reason |
|---|---|
| C# / TS runtimes | Different repos, different teams, different timelines. They consume the same design artifacts but implement independently. Including them here would bloat scope and create false dependencies. |
| Web platform | Orthogonal architecture. Web platform depends on C# runtime, not Go runtime. No coupling. |
| CLI UX beyond errors | This plan is about expression engine internals. CLI UX is a separate concern. |
| Windows test failures | Triaged by Hudson as path-separator issues. Mixing them into P0–P8 creates noise and false coupling. |

---

## H. Rubber-Duck Finding Traceability

Every finding from the rubber-duck critique (2026-06-07) is addressed:

| # | Finding | Status | Where addressed |
|---|---|---|---|
| 1 | Parse-time enforcement wrongly out of scope | **Fixed** | P5 (parse-gate / ValidatedPlan) — mandatory phase. §C.5. |
| 2 | Build-tag side-by-side creates big-bang cutover | **Fixed** | §B rewritten: staged routing with engine-selection seam. P6a/P6b/P6c per-subsystem cutover. |
| 3 | Harness/corpus assumptions false; P1 impossible | **Fixed** | P0 added as mandatory prerequisite. Correct vector counts: 92+83+36+15+41=267. §C.0, §E P0. |
| 4 | GCP materially under-scoped | **Fixed** | GCP split into parser (P3∥), GDP resolver (shared), capture service + executor refactor (P7). 6–10 units. §C.7. |
| 5 | GIS implementation notes contradict grammar | **Fixed** | §C.4 corrected: `\${` escape, GXL delegation for expression-end, per-segment `?.`. All verified against `gis.ebnf`. |
| 6 | PJVM number model wrong | **Fixed** | §C.1 corrected: single finite float64, no int64 discriminated union. Explicit conversion rules. |
| 7 | Cutover gate too vague | **Fixed** | §E P8: benchmark command, dataset, per-category thresholds, alloc/memory cap, race-free requirement, active soak workload (not passive). |
| 8 | Effort estimate optimistic | **Fixed** | Total: 32–49 critical-path units (was 20). Per-phase estimates honest. §E summary table. |

---

## References

- Superseded plan: [`design/gert/proposals/runtime-migration-plan.md`](./runtime-migration-plan.md)
- Rubber-duck critique: rubber-duck-plan review (2026-06-07), 5 blocking + 3 non-blocking findings
- Grammars: `design/gert/grammar/{gxl,gis,gcp}.ebnf`
- Spec sections: `design/gert/sections/03a-expression-language.tex`, `03b-interpolation-syntax.tex`, `03c-capture-paths.tex`, `03d-parse-time-enforcement.tex`, `09-testing-and-acceptance.tex`
- Conformance vectors: `design/gert/conformance/tv-*.yaml` (267 total: 92+83+36+15+41)
- Schema: `design/gert/conformance/schema.json`
- Issue #17 (gert): conformance harness
- Issue #1 (gert-private): corpus YAML hygiene — closed by P0
- PR #20 (gert): broken harness — to be closed, replaced by P1 scaffold
- OQ-M1..M5 ratification: `.squad/decisions.md`
- §03d parse-gate contract: `design/gert/sections/03d-parse-time-enforcement.tex`
