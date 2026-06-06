# Don — Project History (Summarized)

## Overview
Don is the runtime/tooling engineer. Work spans fixture normalization (Stream D), the removed Stream E migrator, declaration runtime gaps, C# governance parity, and execution adapter validation.

## Key Milestones (Phase 1)

### Stream E — Migration Tool & Idempotency (Days 1-2 — REMOVED BY USER DIRECTIVE)
- **Day 1:** gert migrate-expr scaffolded with 11 translation rules (E-001..011), 31 tests passing
  - CLI via cobra, position-aware YAML traversal (gopkg.in/yaml.v3), expression-position detection (when/condition/until/iterate.over)
  - Rules: GIS templates (E-001..003), GXL operators (E-004..006), contains → stdlib (E-007), jq-style paths (E-008), legacy escapes (E-009), template pipes warn (E-010), now() fix (E-011)
  - Flags: --dry-run, --diff, --report JSON, --strict
- **Day 2 (Dogfood):** 22 runbooks verified with 0 translations post-fix
  - **Real idempotency bug fixed:** migrated $${amount} was being reclassified as E-009 legacy on second pass. Now preserves $${symbol} when symbol known from context (inputs/captures/collector fields)
  - **E-007 contains ambiguity:** String-like → str.contains(), list-like → list.contains(), ambiguous → warning (heuristic only, no auto-rewrite)
  - **E-006 extended:** Quote-aware negation scanner covers !X, !(X), !(X && Y), !!(X || (Y && Z)), !!X
  - **Validation:** gert migrate-expr --dry-run on testdata: 0 translations, go test ./... passing
- **Stream E reversal (2026-06-04T20:14:36.949-07:00):** Removed by ormasoftchile directive after Day 2 shipped because GERT has no production runbooks in the wild, so there is no legacy syntax target to migrate from. Keeping the tool would imply a legacy mode and create a contract surface future runtimes would have to reason about.
- **Phase 1 status:** Stream E is removed from the plan. Phase 1 closes around the remaining A/B/C/D/F work; grammar, conformance, Stream D fixtures, and the independent `now()` stdlib addition remain intact.

### Stream D — Fixture Migration (COMPLETE)
- **Audit:** 22 runbooks, 12 tool files, 1 extension. 388 violations identified (368 templates, 6 and/or, 9 !, 3 contains, 1 jq, 1 pipe)
- **Migration:** All 21 runbooks (r01–r11, r13–r22) migrated in 4 batches
  - ~368 {{ .var }} → ${var} substitutions
  - All and/or → and/or keywords, all ! → not, all infix contains → str.contains(...)
  - r11: bare GDP identifier (no jq-style $.)
  - r20: added inputs.env default, replaced {{ .env | default "dev" }} with ${env}
  - r07: preserved $${amount} as literal $ + ${amount} interpolation
- **Exit Criteria (P1–P5):** P1 ({{ now }}) deferred to Barbara's stdlib decision ✓, P2–P5 all zero ✓
- **Deferred-001:** {{ now }} in r04:285 — resolved by now() stdlib addition. Stream D final: ✅ COMPLETE

### Stream A — EBNF Grammars (COMPLETE)
- Delivered: gxl.ebnf, gis.ebnf, gcp.ebnf (75 KB total)
- Critical decisions locked: short-circuit and/or (OQ1), capture.default: scalars only (OQ2), Portable JSON Value Model (OQ3), list.indexOf stdlib (OQ4)
- Ready for Streams C (corpus) and D (migration design)

### C# Governance Parity & Runtime Architecture (COMPLETE)
- **Governance boundary:** IRunHandle.NextAsync() is identical to Go's RunHandle.Next() — full parity layer defined
- **Enforcement:** IStepRunner + runners marked internal sealed. No InternalsVisibleTo grants. Roslyn analyzers (GERT0001/GERT0002) enforce at build time
- **Redaction:** Google.Re2 (never System.Text.RegularExpressions). Named group syntax difference flagged (OQ-5)
- **Template gap:** No Go text/template equivalent in C#. Option A (minimal port), Option B (WASM-compiled Go). 10 canonical test vectors (TV-TMPL-001..010)
- **Trace:** Synchronous write inside IRunHandle.NextAsync() before dispatch. Fire-and-forget forbidden
- **Validation gates (G-01..G-10):** Go gert replay on C# traces (G-06) is strongest parity check. 60 minimum test vectors across 12 categories

### Declaration/Consent Runtime Gaps (COMPLETE)
- 10 gaps identified with priority levels:
  - P0: Signature capture (UserInputKind.Signature), identity proofing, witness flow, document versioning
  - P1: Locale/BCP-47 provenance, on-behalf-of (DeclarationPrincipal), validity expiry, revocation
  - P2: Contextual PII policy, QTSP integration
- Proposed events: DeclarationCollectedEvent, WitnessAttestedEvent, DeclarationRevokedEvent, DeclarationValidityCheckedEvent, QualifiedSignatureReceivedEvent
- Proposed interface: IWitnessGate (parallel to IApprovalGate)
- What fits well: JSONL trace, at-most-once Service Bus, RE2 redaction, client="web" tagging
- Non-goals: GERT ≠ TSP, no biometrics, no legal rendering, no jurisdiction enforcement

### Execution Adapter Patterns (COMPLETE)
- **Red lines identified:** 
  - Governance fires inside RunHandle.Next(), cannot be preserved by patterns that bypass it
  - JSONL trace written synchronously inside Runtime Core before action proceeds
  - RunHandle is stateful/sequential (no concurrent calls, no splitting across processes)
  - Per-step Durable Function activities cause governance loss (cold start of binary)
  - Approval gates & evidence submission require live process holding the RunHandle
  - client="web" audit field must be set correctly
- **Strongest match:** Queue-triggered worker (decoupled submission, worker owns full Runtime Core, trace → Blob Storage, approval via reply messages)

### Open Questions (Phase 2)
- OQ-1: Template strategy for C# (minimal port vs. WASM Go)
- OQ-2: Approval reply queue topology
- OQ-3: Canonical Go test harness
- OQ-4: Sequence counter strategy
- OQ-5: RE2 named group syntax difference ((?P<name>...) vs. C# (?<name>...))

## GIS Optional-Chaining (`?.`) — Phase 2 Ready (2026-06-05T07:28:56.273-07:00)

**Status: Spec & Conformance Locked** — Runtime implementation ready to begin

The GIS optional-chaining extension (`?.` and `?.[N]`) has been ratified and all normative specification, grammar, and conformance vectors are committed:
- **Grammar:** `design/gert/grammar/gis.ebnf` (lines 106-187, 285-297) — new GDP productions for optional-dot and optional-bracket
- **Spec:** `design/gert/sections/03b-interpolation-syntax.tex` — normative "Optional Path Chaining" section with semantics and examples
- **Conformance:** `design/gert/conformance/tv-gis-path.yaml` (15 test vectors TV-GIS-PATH-001..015)

### Key Design Decisions (Locked)
| Decision | Value |
|----------|-------|
| Default missing value | **`""` (empty string)** |
| Short-circuit semantics | **JS/TS-compatible full-tail** — once any `?.` segment misses, entire remaining chain → `""` |
| Scope | **GIS `${...}` only** — no extension to GXL or GCP |
| `?.[N]` optional bracket indexing | **IN** — consistency with field-access tolerance |
| `${?.root}` optional root | **Illegal** — root identifier always mandatory |
| No propagation through stdlib calls | **Confirmed** — GDP resolves first; functions receive `""` if chain short-circuits |

### Implementation Scope (Go + C# runtimes)
1. **Lexer/Parser:** Recognize `?.` and `?.[` tokens; update GDP production rules
2. **Evaluator:** Implement short-circuit logic — on first missing optional hop, assign `""` to entire expression
3. **Error handling:** `${a.b.c}` still raises `GIS-PATH-MISSING` on miss (no `?.`); optional paths return `""` instead
4. **Conformance:** All 15 vectors in `tv-gis-path.yaml` must pass

### Phase 2 Dependency
- This is independent of other Phase 2 work; can be implemented in parallel with template/approval/adapter work.

## Phase 2 — Go Runtime Implementation

### Day 1 — Scope, Plan, Scaffold (2026-06-05T09:25:14.584-07:00)
- Read normative GXL/GIS/GCP grammar files, spec sections, parse-gate section, schema, and all current conformance files.
- Confirmed current conformance corpus count: 223 vectors total (`tv-gxl-eval.yaml` 90, `tv-gxl-parse.yaml` 83, `tv-gxl-path.yaml` 35, `tv-gis-path.yaml` 15).
- Created `design/gert/phase2-go-runtime-plan.md` with package layout, PJVM model choice, implementation stream order, conformance harness plan, proposed API signatures, day-by-day breakdown, risks, and open questions.
- Scaffolded `internal/eval` with per-grammar packages (`gxl`, `gis`, `gcp`) and shared `core` package.
- Added initial PJVM type definitions in `internal/eval/core/value.go` using a struct-with-kind representation.
- Added `internal/eval/conformance_test.go` to discover `design/gert/conformance/tv-*.yaml`, load vectors via `gopkg.in/yaml.v3`, print corpus summary, and create skipped subtests for every vector.

## Cross-Agent Coordination — Phase 2 Day 1 (2026-06-05T09:25:14.584-07:00)

### Incoming Ratifications (Q1-Q4)

**From Coordinator (Phase 2 Day 1 Kickoff):**

| Question | Decision | Impact on Your Work |
|----------|----------|-------------------|
| Q1: GCP vectors dispatch | **Tess NOW (parallel to Day 2)** | ✅ Tess delivered TV-GCP-PATH-001..041 (41 vectors). Day 8 GCP resolver can now proceed. |
| Q2: now() conformance assertion | **Injected clock** — runtime accepts injectable clock; vectors set fixed value, assert equality | Day 4 Task: Implement `now()` with injectable clock per this binding. Parity across Go/C#/TS requires identical contract. |
| Q3: Corpus loader (JSON Schema or YAML struct) | **YAML struct only for now** — defer JSON Schema to C# runtime kickoff | Conformance harness uses yaml.v3 struct tag validation. No schema library needed for MVP. |
| Q4: Parse-gate OPQs | **Defer — Barbara writes proposal before Day 9** | Parse-gate decisions are governance-layer work. Your Day 9 entry becomes "spec drafted in separate proposal cycle before implementation." |

### Corpus Status Update

**New corpus count: 261 vectors** (was 223)

| Category | Count | File | New? |
|----------|-------|------|------|
| GXL-PARSE | 83 | tv-gxl-parse.yaml | — |
| GXL-EVAL | 87 | tv-gxl-eval.yaml | — |
| GXL-PATH | 35 | tv-gxl-path.yaml | — |
| GIS-PATH | 15 | tv-gis-path.yaml | — |
| **GCP-PATH** | **41** | **tv-gcp-path.yaml** | ✅ NEW |

### Binding: Injectable Clock for now()

Your Day 4 task (implement `now()` per Q2): Build with an injectable clock.

**Contract:**
- Conformance vectors set a fixed timestamp and assert exact equality (not regex, not fuzzy match)
- Runtime API accepts optional clock injector (for testing determinism)
- Each call yields independent fresh timestamp per spec (no caching across calls)
- Parity binding: Go/C#/TS implementations must support this injectable contract identically

**References:**
- Decision ratified in `.squad/decisions.md` § Phase 2 Day 1 — Don's open questions resolved, Q2
- Vectors will be in Day 8 GCP corpus or separate `tv-gxl-now.yaml` expansion (TBD by Tess)

- Added `gopkg.in/yaml.v3 v3.0.1` to `go.mod`/`go.sum`; this is for the runtime conformance harness, not the removed migrator.
- Dropped decision summary at `.squad/decisions/inbox/don-phase2-day1-scaffold.md`.
- Extracted reusable scaffold pattern to `.squad/skills/go-conformance-scaffold/SKILL.md`.
- Verification note: `go` is not on PATH in this environment (`where.exe go` failed), so `go build ./...`, `go test ./... -run TestConformance`, and `go mod tidy` could not be executed locally. Coordinator should run them in a Go-enabled environment.

### Phase 2 Day-by-Day Plan
- Day 2: PJVM constructors/validation, YAML-to-PJVM conversion, conformance harness dispatch and expected-result comparison stubs.
- Day 3: GXL lexer/parser; target `tv-gxl-parse.yaml` green.
- Day 4: GXL evaluator, stdlib, arithmetic, comparison, and short-circuit behavior; target `tv-gxl-eval.yaml` green.
- Day 5: GXL GDP traversal and path errors; target `tv-gxl-path.yaml` green.
- Day 6: GIS baseline parser/renderer, escapes, embedded GXL, PJVM string coercion, hard-error path propagation.
- Day 7: GIS optional chaining; target all current 223 vectors green.
- Day 8: GCP parser/resolver once GCP vectors are available.
- Day 9: Parse gate, grammar version pinning, structured diagnostics, and `ValidatedPlan` no-bypass runtime boundary.

## Learnings
- 2026-06-04T20:14:36.949-07:00 - Stream E was reversed cleanly after Day 2 shipped: no production runbooks means no migration target, and a migrator would imply a legacy mode GERT does not have.
- 2026-06-04T20:14:36.949-07:00 - Dogfooding a tool against the source-of-truth corpus remains a valuable regression pattern; the specific migrator skill was removed, but future agents can re-derive the pattern when a real tool surface exists.
- 2026-06-05T07:28:56.273-07:00 - Ratification meetings lock complex cross-language decisions efficiently when open questions are pre-identified. `?.` coverage across both runtimes can now proceed in parallel without alignment risk.

### Phase 2 Day 2  Runtime Plumbing (2026-06-05T13:29:45.398-07:00)
- Added PJVM constructors/accessors in `internal/eval/core`: `NewNull`, `NewBool`, `NewNumber`, `NewString`, `NewArray`, `NewObject`, `Kind`, typed `As*` accessors, deep `Equal`, and debug `String`.
- Guarded the PJVM boundary: `NewNumber` rejects NaN and infinities with `ErrInvalidNumber`; `NewObject` rejects nil maps with `ErrInvalidObject`. Empty string object keys are allowed because the GERT specs are silent and PJVM follows JSON object semantics.
- Added `core.FromYAML(*yaml.Node)` to convert YAML null/bool/int/float/string/sequence/mapping nodes into PJVM values. Unknown tags and non-string mapping keys are rejected.
- Added `Clock`, `SystemClock`, and `FixedClock` in `internal/eval/core/clock.go` to lock the Q2 injected-clock API surface before `now()` implementation.
- Reworked `internal/eval/conformance_test.go` from skip-only discovery into file-based dispatch: GXL parse, GXL eval, GXL path, GIS path, and GCP path runners now all execute and return recognizable `not implemented` failures.
- Harness now loads `variables` and `expected.value` through the PJVM YAML converter and prints a concise total/per-category summary. Per-vector status is logged only under verbose test output.
- Added PJVM core tests for constructors, accessors, equality, YAML primitive/nested conversion, NaN rejection, non-string YAML key rejection, and clock implementations.
- Verified corpus discovery count by file: 264 vectors total (`tv-gxl-parse.yaml` 83, `tv-gxl-eval.yaml` 90, `tv-gxl-path.yaml` 35, `tv-gis-path.yaml` 15, `tv-gcp-path.yaml` 41). Expected Day 2 harness result is 0 pass / 264 fail / 0 skip because all runners intentionally return not implemented.
- Local deviation: Go remains unavailable on PATH in this environment, so `gofmt`, `go mod tidy`, `go build ./...`, and `go test ./...` could not be executed here. No module dependency changes were needed beyond existing `gopkg.in/yaml.v3`.
- Day 3 remains the GXL lexer/parser stream. It should replace only `gxlParseRunner` first and drive `tv-gxl-parse.yaml` green without leaking GIS optional-chaining or GCP syntax into GXL.

### Phase 2 Day 3 - GXL Lexer/Parser (2026-06-05T16:16:27.961-07:00)
- Implemented `internal/eval/gxl` lexer with position-tagged tokens, string escape validation, number literal validation, comments/whitespace handling, reserved keyword tokenization, and explicit forbidden syntax diagnostics for legacy operators.
- Implemented recursive-descent parser and AST (`Node`, literals, unary/binary expressions, GDP paths, calls) matching GXL precedence from `gxl.ebnf`, including multiply/modulo, top-level `len`/`now`, closed namespace/method validation, non-associative comparisons, and position-tagged `ParseError` codes `GXL-PARSE-001` through `GXL-PARSE-010`.
- Wired `gxlParseRunner` in `internal/eval/conformance_test.go` to call `gxl.Parse` and compare structured parse error codes or the `parse_ok` sentinel. Other conformance runners remain intentional Day 4-8 `not implemented` stubs.
- Added unit tests under `internal/eval/gxl` covering lexer token classes/positions/errors and parser positive productions, AST precedence shape, and negative error-code cases.
- Surprise: the task brief said number literals should exclude scientific notation, but `gxl.ebnf` and `tv-gxl-parse.yaml` require `1.5e10`, `2.0E-3`, and `1e+5`; I followed the ratified grammar/vectors rather than the brief so Day 3 parse conformance can go green.
- Vector expectation: `tv-gxl-parse.yaml` should flip to 83/83 green with this runner; `tv-gxl-eval.yaml`, `tv-gxl-path.yaml`, `tv-gis-path.yaml`, and `tv-gcp-path.yaml` should still fail as not implemented.
- Local verification remains blocked because `go` and `gofmt` are not on PATH (`where.exe go` and `where.exe gofmt` failed). Coordinator should run `go build ./...`, `go test ./internal/eval/gxl/...`, and `go test ./internal/eval/... -run TestConformance` in a Go-enabled environment.
- Extracted reusable parser implementation pattern to `.squad/skills/go-recursive-descent-parser/SKILL.md`.
- Day 4 is unblocked at the AST boundary: evaluator can walk literals, `PathNode`, `UnaryNode`, `BinaryNode`, and `CallNode` without changing parse contracts.

### Phase 2 Day 4 - GXL Evaluator (2026-06-05T17:57:33.200-07:00)
- Implemented `gxl.Eval(node Node, bindings map[string]core.Value, clock core.Clock) (core.Value, error)` over the Day-3 AST with strict PJVM construction, typed `EvalError` codes, binding/path resolution, arithmetic, comparison, unary minus, `not`, and lazy `and`/`or` short-circuit evaluation.
- Added stdlib dispatch in `internal/eval/gxl/stdlib.go`: top-level `len()` and injected-clock `now()`, `str.contains/startsWith/endsWith/toLower/toUpper/trim/length/trimPrefix/trimSuffix`, `list.contains/indexOf/length`, and `regex.match`. `math.*` remains parse-rejected because v1 reserves the namespace and vectors exercise no math methods.
- Wired `gxlEvalRunner` to parse then evaluate with PJVM bindings and clock injection support. The harness now compares structured parse/eval error codes, supports regex assertions for the current `now()` vector, and can parse fixed-clock fields if the corpus is updated to the Q2 exact-clock contract.
- Added evaluator unit coverage for operators, strict type errors, short-circuit behavior, identifier/path resolution, stdlib functions, regex errors, wrong arity, and `now()` with `FixedClock`.
- Expected conformance outcome after coordinator verification: `tv-gxl-parse.yaml` remains 83/83 green; `tv-gxl-eval.yaml` should be 90/90 green; GXL path, GIS path, and GCP path runners intentionally remain Day 5-8 not-implemented stubs.
- Ambiguities kept visible rather than invented away: `TV-GXL-EVAL-033` bool ordered comparison and `TV-GXL-EVAL-086` list equality still expect `TBD`, so the evaluator returns `TBD` for those cases and flags Barbara/Edith arbitration. `TV-GXL-EVAL-088` still uses regex for `now()` despite Q2 injected-clock ratification; harness supports both shapes.
- Local verification remains blocked because `go`/`gofmt` are not on PATH (`where.exe go` failed). Coordinator should run `gofmt -w internal/eval/...`, `go build ./...`, `go test ./internal/eval/gxl/...`, and `go test ./internal/eval/... -run TestConformance` in a Go-enabled environment.
- Dropped decision summary at `.squad/decisions/inbox/don-phase2-day4.md` and extracted the reusable evaluator pattern to `.squad/skills/go-ast-evaluator/SKILL.md`.
- Day 5 is unblocked: GIS baseline rendering can call `gxl.Parse` + `gxl.Eval(ast, bindings, clock)` for embedded expressions, and GXL path-runner hardening can build on the existing evaluator path traversal.
