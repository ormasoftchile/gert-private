# Phase 2 Go Runtime Plan — GXL/GIS/GCP

Date: 2026-06-05T09:25:14.584-07:00  
Owner: Don — Backend Dev

## Scope

Phase 2 implements the Go runtime surface for the normative GXL, GIS, and GCP artifacts only:

- `design/gert/grammar/gxl.ebnf`
- `design/gert/grammar/gis.ebnf`
- `design/gert/grammar/gcp.ebnf`
- `design/gert/sections/03a-expression-language.tex`
- `design/gert/sections/03b-interpolation-syntax.tex`
- `design/gert/sections/03c-capture-paths.tex`
- `design/gert/sections/03d-parse-time-enforcement.tex`
- `design/gert/conformance/tv-gxl-eval.yaml`
- `design/gert/conformance/tv-gxl-parse.yaml`
- `design/gert/conformance/tv-gxl-path.yaml`
- `design/gert/conformance/tv-gis-path.yaml`
- `design/gert/conformance/schema.json`

No parser, evaluator, or resolver may encode semantics not present in those artifacts.

## Package Layout

Chosen layout:

```text
internal/eval/
  conformance_test.go      # corpus loader and table-driven harness
  core/                    # shared PJVM, diagnostics, path primitives
  gxl/                     # GXL lexer/parser/evaluator
  gis/                     # GIS template parser/renderer, GXL embedding, GIS optional chaining
  gcp/                     # GCP parser/resolver for captures
```

Rationale: a monolithic `internal/eval` package would make cross-grammar shortcuts too easy, especially around GIS optional chaining leaking into GXL or GCP. Fully independent packages would duplicate PJVM and diagnostic contracts. The selected shared-core + per-grammar layout keeps the PJVM and structured errors common while preserving grammar boundaries.

Rejected options:

- `internal/eval/` only: too much coupling; governance and parity hazards.
- `internal/gxl`, `internal/gis`, `internal/gcp` only: clearer names, but no obvious home for shared PJVM and conformance harness.
- Public packages on Day 1: premature. The runtime can expose stable public wrappers later, after the parse gate and `ValidatedPlan` boundary are shaped.

## PJVM Value Model

Chosen representation: a typed sum encoded as a struct-with-kind.

```go
type Kind uint8

const (
    KindInvalid Kind = iota
    KindNull
    KindBool
    KindNumber
    KindString
    KindArray
    KindObject
)

type Value struct {
    Kind   Kind
    Bool   bool
    Number float64
    String string
    Array  []Value
    Object map[string]Value
}

type Bindings map[string]Value
```

Rationale: PJVM has exactly six normative types. `interface{}` would be quick, but would admit Go-specific types (`int`, `json.Number`, `time.Time`, `nil` ambiguity, non-finite floats) and push type checks into scattered assertions. A struct-with-kind makes illegal states visible and gives future constructors one place to reject NaN/Infinity and canonicalize JSON/YAML ingestion.

Trade-off: the struct can temporarily hold irrelevant payload fields. Day 2 should add constructors and validation helpers to enforce one active payload per kind before evaluator code consumes values.

## Implementation Stream Ordering

1. **PJVM + conformance harness** — everything else depends on shared value fidelity and visible corpus counts.
2. **GXL parser** — GXL is embedded by GIS and defines GDP expression syntax; parse vectors are the fastest feedback loop.
3. **GXL evaluator + path resolution** — unlocks `tv-gxl-eval.yaml` and `tv-gxl-path.yaml`; validates strict typing and short-circuit semantics.
4. **GIS parser/renderer** — builds on GXL parser/evaluator, then adds template segmentation and string coercion.
5. **GIS optional path chaining** — isolated GIS-only GDP extension; implement after baseline GXL to avoid leaking `?.` into GXL/GCP.
6. **GCP parser/resolver** — capture paths feed runtime bindings, but current corpus has no GCP vectors in the Day 1 set; implement after GXL/GIS corpus is green.
7. **Parse-gate integration** — after grammar surfaces pass standalone corpus, wire `ValidatedPlan` and no-bypass runtime contracts.

## Conformance Harness

The harness lives in `internal/eval/conformance_test.go` and discovers every `design/gert/conformance/tv-*.yaml` file. Day 1 loads and counts vectors only; each vector is a skipped subtest until runtime APIs exist.

Target shape:

```go
func TestConformance(t *testing.T) {
    files := discoverCorpus("design/gert/conformance/tv-*.yaml")
    for _, file := range files {
        vectors := loadYAML(file)
        t.Run(filepath.Base(file), func(t *testing.T) {
            for _, vector := range vectors {
                t.Run(vector.ID, func(t *testing.T) {
                    switch vector.Category {
                    case "GXL-PARSE": runGXLParseVector(t, vector)
                    case "GXL-EVAL", "GXL-PATH": runGXLEvalVector(t, vector)
                    case "GIS-PATH", "GIS-INTERP", "GIS-TYPE": runGISVector(t, vector)
                    case "GCP-PARSE", "GCP-RESOLVE", "GCP-DEFAULT": runGCPVector(t, vector)
                    default: t.Fatalf("unknown category")
                    }
                })
            }
        })
    }
}
```

One helper per category family keeps parser-only, evaluator, interpolation, and resolver expectations separate while still reporting one subtest per vector ID. The harness must fail on unknown fields after schema validation is added; executing a malformed corpus file would make CI meaningless.

## Proposed Runtime API Surface

These are the intended public contracts. They should remain thin wrappers around grammar packages and must not provide a path around the planner gate for runbook execution.

```go
// ParseExpression parses a GXL expression and returns an opaque expression tree.
// It performs parse-time validation only; no variables are read.
func ParseExpression(source string) (*gxl.Expression, error)

// EvaluateExpression evaluates a parsed GXL expression against PJVM bindings.
// The evaluator must implement strict PJVM typing and GXL short-circuit rules.
func EvaluateExpression(expr *gxl.Expression, bindings core.Bindings) (core.Value, error)

// EvaluateExpressionSource parses and evaluates a GXL expression in one call.
// This is acceptable for tests and tooling, but runbook execution must use the planner path.
func EvaluateExpressionSource(source string, bindings core.Bindings) (core.Value, error)

// InterpolateString renders a GIS template against PJVM bindings.
// It supports full embedded GXL expressions and GIS-only optional chaining.
func InterpolateString(template string, bindings core.Bindings) (string, error)

// ParseCapturePath parses a GCP capture path into an opaque path plan.
// Step ID validation and default policy checks are planner responsibilities.
func ParseCapturePath(source string) (*gcp.Path, error)

// ResolveCapturePath resolves a parsed GCP path against captured step data.
// It returns the PJVM value to bind under the capture variable name.
func ResolveCapturePath(path *gcp.Path, sources gcp.Sources) (core.Value, error)
```

Future execution APIs must accept only a `ValidatedPlan` product from the planner. No exported function should construct a run handle from raw YAML or unvalidated expressions.

## Day-by-Day Breakdown

- **Day 2 — PJVM + harness hardening:** add PJVM constructors/validation, YAML-to-PJVM conversion, schema-aware vector structs, and category dispatch with expected-result comparison stubs.
- **Day 3 — GXL lexer/parser:** implement tokenization, recursive-descent parser, parse diagnostics, and make `tv-gxl-parse.yaml` green.
- **Day 4 — GXL evaluator:** implement literals, arithmetic, comparison, `and`/`or` short-circuit, type errors, and stdlib functions; make `tv-gxl-eval.yaml` green.
- **Day 5 — GXL GDP paths:** implement object/index traversal and strict path errors; make `tv-gxl-path.yaml` green and stabilize shared path primitives.
- **Day 6 — GIS baseline:** implement template segmentation, escapes, embedded GXL evaluation, PJVM string coercion, and ordinary hard-error path propagation.
- **Day 7 — GIS optional chaining + full Day 1 corpus:** implement GIS-only `?.` / `?.[N]`, verify all 223 current vectors pass, then decide whether to add GCP vectors before parse-gate work.
- **Day 8 — GCP parser/resolver:** implement capture path parsing and resolution once GCP conformance vectors exist or are supplied by Tess.
- **Day 9 — Parse gate integration:** implement planner-facing no-bypass contracts, grammar version constants, structured diagnostics, and `ValidatedPlan` construction boundary.

## Risks and Open Questions

1. **No GCP conformance file is listed in the current 223-vector bar.** GCP is normative, but Day 1 corpus files are GXL/GIS only. We need Tess/ormasoftchile to confirm when GCP vectors land and whether Phase 2 done requires them in addition to the current 223.
2. **GIS optional chaining inside full GXL expressions has a documented tension.** The spec says GIS supports full GXL and optional GDP segments in GIS; the implementation must localize the GDP override to GIS parsing without accepting `?.` in normal GXL fields.
3. **`now()` makes pure evaluation time-dependent.** The conformance corpus must avoid fixed timestamp expectations or provide matcher semantics; otherwise deterministic CI will be brittle.
4. **Number string coercion requires ECMAScript-compatible formatting details.** Go can approximate via `strconv.FormatFloat`, but the `1e20`/`1e21` thresholds need explicit tests before cross-runtime parity is trusted.
5. **Structured diagnostics require source locations.** The current vector schema checks codes and values, but parse-gate implementation must also carry location/snippet data for runbook validation.
6. **Open parse-gate questions remain unresolved:** in-flight grammar upgrades, statically reachable step set, GIS warning trace shape, plan storage policy, and PLAN-* corpus coverage.
