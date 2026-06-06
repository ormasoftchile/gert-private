# Runtime Migration Plan: GXL/GIS/GCP Expression Engine

**Author:** Barbara — Lead / Architect  
**Date:** 2026-06-05T18:33:34-07:00  
**Status:** PROPOSED — awaiting ormasoftchile review  
**Target Repo:** `ormasoftchile/gert` (Go runtime)  
**Reference Design:** `ormasoftchile/gert-private` (this repo)

---

## 1. Current State Survey

### 1.1 Expression/Interpolation Surface in `ormasoftchile/gert`

| Layer | Path | Types | Responsibility |
|-------|------|-------|----------------|
| **Interface contracts** | `pkg/expr/expr.go` | `Evaluator` (template interpolation), `ConditionEvaluator` (boolean eval) | Stable API boundary; all consumers depend on these two interfaces |
| **Template impl** | `internal/expr/template.go` | `TemplateEvaluator` | `{{ }}` interpolation via `text/template`; stdlib: contains, hasPrefix, hasSuffix, toLower, toUpper, trimSpace, join |
| **Condition impl** | `internal/expr/condition.go` | `SimpleConditionEvaluator` | Boolean eval via `expr-lang/expr` v1.17.8; stdlib: `str.contains`, `str.startsWith`, `str.endsWith`, `str.toLower`, `str.toUpper`, `str.trim`; auto-strips `{{ }}` wrapper |
| **Tests** | `internal/expr/template_test.go` | 4 tests | Exercises basic interpolation, nested access, missingkey=error |
| **Tests** | `internal/expr/condition_test.go` | 34 tests | Exercises comparisons, boolean ops, `vars.` prefix, `str.*` builtins, `{{ }}` stripping |
| **Test fake** | `pkg/testutil/fake_expr_evaluator.go` | `FakeExprEvaluator` | Controllable stub for unit tests (implements `Evaluator`) |

### 1.2 All Callers of Evaluator / ConditionEvaluator

**Wiring points (where instances are created and injected):**
- `pkg/run/run.go:225-226` — creates `TemplateEvaluator` + `SimpleConditionEvaluator`, injects into engine config
- `internal/adapter/wire.go:59-60` — same pattern for adapter-based wiring

**Direct consumers (hold a reference to the interface):**

| Consumer | File | Uses |
|----------|------|------|
| CLIExecutor | `internal/executor/cli.go` | `Evaluator` — resolves command, args, stdin, workdir, shell, env, run script |
| ToolExecutor | `internal/executor/tool.go` | `Evaluator` — resolves tool name, action, string args |
| BranchExecutor | `internal/executor/branch.go` | `ConditionEvaluator` — evaluates branch conditions |
| IterateExecutor | `internal/executor/iterate.go` | Both — resolves templates + evaluates until conditions |
| CollectorExecutor | `internal/executor/collector.go` | Both — resolves prompts + evaluates field `when` conditions |
| AssertExecutor | `internal/executor/assert.go` | `Evaluator` — resolves assertion expressions |
| ChoiceExecutor | `internal/executor/choice.go` | `Evaluator` — resolves prompt text |
| DecisionExecutor | `internal/executor/decision.go` | `Evaluator` — resolves prompt text |
| DisplayExecutor | `internal/executor/display.go` | `Evaluator` — resolves display content |
| IncludeExecutor | `internal/executor/include.go` | `Evaluator` — resolves include references |
| NoopExecutor | `internal/executor/noop.go` | `Evaluator` — resolves noop message |
| helpers.go | `internal/executor/helpers.go` | `resolveTemplate()`, `resolveStringSlice()`, `evalCondition()` — shared utilities |
| Registry | `internal/executor/registry.go` | Wires both evaluators into all executors |
| Engine config | `pkg/engine/engine.go:77-83` | Holds both evaluators on `Config` struct |

**Total blast radius: 12 executor types + 2 wiring points + 1 engine config struct + helpers.**

### 1.3 Tests Depending on expr-lang or text/template Semantics

| Test File | Count | Semantics Relied On |
|-----------|-------|---------------------|
| `internal/expr/condition_test.go` | 34 cases | expr-lang `contains` infix, `str.*` namespace, `vars.` map access, `{{ }}` prefix strip, `AsBool()` type enforcement |
| `internal/expr/template_test.go` | 4 cases | `text/template` `.name` dot-prefix access, `missingkey=error`, `{{ }}` delimiters |
| `internal/executor/branch_test.go` | ~10 cases using real evaluator | expr-lang semantics directly (lines 114-252) |
| `internal/executor/cli_test.go` | 3 cases | `TemplateEvaluator` for `{{ }}` interpolation |
| `internal/executor/iterate_test.go` | 1 case | Both evaluators |
| `internal/executor/display_test.go` | 1 case | `TemplateEvaluator` |
| `internal/executor/noop_test.go` | 1 case | `TemplateEvaluator` |
| `internal/executor/assert_test.go` | 2 cases | `TemplateEvaluator` |
| `internal/executor/tool_test.go` | 1 case | `TemplateEvaluator` |
| `internal/adapter/wire_test.go` | 1 case | Asserts non-nil evaluators |

**Total: ~58 test cases** that will need adaptation or replacement.

### 1.4 Fixture-Execution Flow

```
YAML file → internal/parser (parseRunbook, YAML decode)
         → pkg/schema (Runbook struct with Steps, FlowNodes, Capture maps)
         → internal/planner (expansion, validation)
         → pkg/engine (orchestration loop)
         → internal/executor/* (per-step-type execution)
              ├── Evaluator.Eval(template, vars) — string interpolation
              ├── ConditionEvaluator.EvalBool(cond, vars) — boolean conditions
              └── step.Capture map[string]string — ad-hoc key→source capture
```

**Capture mechanism today:** Purely keyword-based. `step.Capture` is `map[string]string` where values are fixed strings like `"stdout"`, `"stderr"`, `"exit_code"`. The CLI executor does a `switch source` over known keywords. There is NO path-expression resolution — no dot-access, no array indexing, no JSON-path-like evaluation. This is **significantly simpler** than GCP spec requires.

---

## 2. Gap Analysis

| # | Capability | GXL/GIS/GCP Spec Requires | Current Runtime Has | Gap |
|---|-----------|---------------------------|---------------------|-----|
| 1 | **GXL Parser** | Recursive-descent per gxl.ebnf; precedence: Or/And/Not/Cmp/Add/Mul/Unary/Primary; forbidden-token rejection (GXL-BANNED-*) | expr-lang/expr (CEL-like; different precedence, different token set) | **Full replacement** |
| 2 | **GXL Evaluator** | Strict PJVM typing, no coercion, short-circuit &&/\|\|, `str.*`/`list.*`/`regex.*`/`math.*`/`len()`/`now()` stdlib | expr-lang with ad-hoc `str.*` map injection, no PJVM, no list/regex/math/len/now | **Full replacement** |
| 3 | **GIS Interpolation** | `${expr}` delimiters, GDP path resolution with `.` and `[N]` access, optional chaining `?.`/`?.[N]`, stdlib in interpolation context | `{{ .path }}` text/template with dot-prefix access, FuncMap helpers, `missingkey=error` | **Full replacement** |
| 4 | **GCP Capture Paths** | GDP path expressions resolving into step output (bare root, dot member, bracket index per gcp.ebnf) | Flat `map[string]string` with keyword switch (`"stdout"`, `"stderr"`, `"exit_code"`) | **New subsystem** — current capture is keyword-enum, not path-based |
| 5 | **PJVM** | Portable JSON Value Model: typed Null/Bool/Number/String/Array/Object with defined equality, comparison, and truthiness rules | `any`/`map[string]any` from YAML unmarshaling; no type enforcement | **New foundational type** |
| 6 | **Injected Clock** | `now()` returns value from injected `Clock` interface (per Q2 ratification); test determinism requires `FixedClock` | No `now()` function; no clock abstraction | **New interface + stdlib entry** |
| 7 | **Conformance Harness** | Must pass 264 vectors across 5 corpora (tv-gxl-parse, tv-gxl-eval, tv-gxl-path, tv-gis-path, tv-gcp-path) | No conformance harness; tests are hand-written per-feature unit tests | **New test infrastructure** |
| 8 | **Forbidden Token Detection** | GXL-BANNED-AND, GXL-BANNED-OR, GXL-BANNED-NOT, GXL-BANNED-CONTAINS — parse-time rejection with specific error codes | expr-lang silently accepts `&&`, `||`, `!`, infix `contains` | **Behavioral inversion** |
| 9 | **Error Codes** | Spec-mandated error codes (GXL-*, GIS-*, GCP-*) for all failure modes | Generic Go errors with no structured codes | **New error taxonomy** |

---

## 3. Migration Strategy

### Philosophy

**Parallel-package, feature-flag cutover.**

Rationale: The blast radius (12 executors, 2 wiring points, 58 tests) is large but the interface surface is small (2 interfaces, 3 methods total). The safest approach is:
1. Build the new engine in a parallel package (`internal/eval/`) while the old `internal/expr/` continues to serve production.
2. Adapt the new engine to satisfy the SAME `pkg/expr.Evaluator` and `pkg/expr.ConditionEvaluator` interfaces.
3. Feature-flag the switch at the wiring layer (`pkg/run/run.go`, `internal/adapter/wire.go`).
4. Once 264 conformance vectors pass and integration tests are green, delete the old package and drop dependencies.

This avoids big-bang, preserves revert-to-one-commit, and lets the old path remain functional throughout.

---

### Phase A: Foundation (PJVM + Clock + Conformance Harness)

**What:** Introduce `internal/eval/core/` with PJVM value types, Clock interface, and the conformance test harness that reads `tv-*.yaml` from a design-repo checkout.

**Package layout:**
```
internal/eval/
  core/           ← PJVM types (Value, Null, Bool, Number, String, Array, Object)
  core/clock.go   ← Clock interface + SystemClock + FixedClock
  harness/        ← conformance_test.go, schema loader, dispatch
```

**Test strategy:** Harness loads vectors; all runners initially return `NotImplementedError`; CI marks these as "expected skip" rather than failures. Tests go green stream-by-stream in later phases.

**Dependencies:** None on existing code. Can land as pure addition.

**Exit criteria:** `go test ./internal/eval/...` passes; PJVM unit tests cover all constructors; harness loads all 264 vectors without panic.

**Estimated size:** Medium (2-3 days)

---

### Phase B: GXL Parser + Parse Conformance (tv-gxl-parse)

**What:** Recursive-descent GXL parser matching `gxl.ebnf` exactly. Produces AST. Rejects forbidden tokens with spec error codes.

**Package layout:**
```
internal/eval/gxl/
  lexer.go        ← tokenizer (identifiers, operators, literals, forbidden-token detection)
  parser.go       ← recursive-descent (OrExpr → AndExpr → ... → Primary)
  ast.go          ← node types
  errors.go       ← GXL-BANNED-*, GXL-PARSE-* error codes
```

**Test strategy:** Harness flips `gxlParse` runner to real parser; tv-gxl-parse.yaml (83 vectors) must go green.

**Dependencies:** Phase A (PJVM for literal representation in AST).

**Exit criteria:** 83/83 tv-gxl-parse vectors green. No existing tests affected (new package, no consumers yet).

**Estimated size:** Medium (2-3 days)

---

### Phase C: GXL Evaluator + Eval Conformance (tv-gxl-eval)

**What:** AST walker implementing GXL semantics: strict PJVM typing, short-circuit, arithmetic, comparison, stdlib (`str.*`, `list.*`, `regex.*`, `math.*`, `len()`, `now()`).

**Package layout:**
```
internal/eval/gxl/
  evaluator.go    ← Walk(ast, env) → Value
  stdlib.go       ← all stdlib functions (str, list, regex, math, len, now)
  env.go          ← variable binding (accepts map[string]any → PJVM conversion)
```

**Test strategy:** Harness flips `gxlEval` runner; tv-gxl-eval.yaml vectors must go green. `now()` tested via FixedClock injection.

**Dependencies:** Phase B (parser produces AST for evaluator to walk).

**Exit criteria:** tv-gxl-eval.yaml green (count TBD — depends on TESS-AMBIG resolution). All existing tests still pass (no consumer changes yet).

**Estimated size:** Large (3-5 days — stdlib is the bulk of the work)

---

### Phase D: GXL Path Resolution + Adapter (tv-gxl-path)

**What:** GDP path evaluator for GXL context (variable lookup with dot/bracket access into PJVM values). Then: build an adapter that wraps the GXL evaluator to satisfy `pkg/expr.ConditionEvaluator`.

**Package layout:**
```
internal/eval/gxl/
  path.go         ← GDP resolution: ident.field[n].field → Value
internal/eval/
  adapter.go      ← GXLConditionAdapter implements pkg/expr.ConditionEvaluator
                     (accepts map[string]any, converts to PJVM env, calls parser+evaluator)
```

**Test strategy:** tv-gxl-path.yaml green. Adapter unit tests prove it satisfies the interface contract. Existing `condition_test.go` semantics are covered by conformance vectors.

**Dependencies:** Phase C (evaluator).

**Exit criteria:** tv-gxl-path.yaml green. `GXLConditionAdapter` passes interface satisfaction check.

**Estimated size:** Medium (2-3 days)

---

### Phase E: GIS Interpolation Engine (tv-gis-path)

**What:** `${...}` parser + GDP path resolver for interpolation context. Handles optional chaining (`?.`, `?.[N]`), stdlib calls in interpolation, and `GIS-PATH-MISSING` errors for non-optional misses.

**Package layout:**
```
internal/eval/gis/
  parser.go       ← scans for ${...} delimiters, parses GDP path + optional chaining
  resolver.go     ← resolves GDP paths against PJVM environment
  interpolator.go ← full string interpolation (literal segments + resolved expressions)
  errors.go       ← GIS-PATH-MISSING, GIS-PARSE-* error codes
internal/eval/
  adapter.go      ← GISEvaluatorAdapter implements pkg/expr.Evaluator
                     (accepts template string + map[string]any, returns interpolated string)
```

**Test strategy:** tv-gis-path.yaml green. Adapter unit tests. Edge cases: nested `${}`, escaped `$${}`, optional chain to empty string, stdlib in interpolation.

**Dependencies:** Phase A (PJVM), Phase D (GDP path resolution — can share resolution logic).

**Exit criteria:** tv-gis-path.yaml green. `GISEvaluatorAdapter` passes interface satisfaction check.

**Estimated size:** Large (3-4 days — optional chaining logic is intricate)

---

### Phase F: GCP Capture Path Engine (tv-gcp-path)

**What:** GDP path resolver for capture context. Evaluates `capture:` map values as path expressions into step output (not flat keyword switch).

**Package layout:**
```
internal/eval/gcp/
  resolver.go     ← resolves GCP capture paths against step output (PJVM-typed)
  errors.go       ← GCP-PATH-MISSING, GCP-PARSE-* error codes
```

**Integration point:** The engine's capture-processing logic (currently `cli.go:138-151` switch statement) must be replaced with GCP path resolution. This touches executor internals.

**Test strategy:** tv-gcp-path.yaml green. Integration test: CLI executor with GCP capture paths resolving into structured output.

**Dependencies:** Phase A (PJVM).

**Exit criteria:** tv-gcp-path.yaml (41 vectors) green.

**Estimated size:** Medium (2-3 days)

---

### Phase G: Consumer Migration + Feature Flag

**What:** Wire the new adapters into the runtime, guarded by a build tag or config flag.

**Changes:**
1. `pkg/run/run.go` — behind flag, instantiate `GXLConditionAdapter` and `GISEvaluatorAdapter` instead of `SimpleConditionEvaluator` and `TemplateEvaluator`.
2. `internal/adapter/wire.go` — same.
3. `internal/executor/cli.go` (and similar) — replace keyword-switch capture with GCP resolver call (requires injecting a `CaptureResolver` into executor constructors or moving capture logic to engine layer).
4. Update `Registry` config to carry optional `CaptureResolver`.

**Test strategy:** Run full test suite under both flag states. All existing tests must pass under old flag. Under new flag, existing tests that rely on `{{ }}` syntax will fail — they are replaced with GXL/GIS equivalent assertions in the same PR.

**Dependencies:** Phases D, E, F (all adapters must exist).

**Exit criteria:** Under new flag: 264 conformance vectors green + all integration/e2e tests green. Under old flag: existing tests unchanged.

**Estimated size:** Medium (2-3 days)

---

### Phase H: Cutover + Dependency Drop

**What:** Remove old code, drop external dependencies, delete feature flag.

**Preconditions (ALL must be true):**
1. 264/264 conformance vectors green in CI
2. All 22 runbook fixtures execute successfully under new engine
3. All integration/e2e tests green under new engine (flag=new)
4. Performance benchmarks show no >2x regression on any vector category
5. No open P0/P1 bugs against new engine in issue tracker
6. At least 1 week of soak time with flag=new as default

**Deletions:**
- `internal/expr/` (entire directory)
- `pkg/testutil/fake_expr_evaluator.go` (replaced by conformance harness)
- `go.mod`: remove `github.com/expr-lang/expr`
- `go.mod`: `text/template` is stdlib so stays, but `internal/expr/template.go` import is removed

**Exit criteria:** `go mod tidy` clean. `go test ./...` green. No import of `expr-lang/expr`. No `{{ }}` interpolation syntax in any runtime code path.

**Estimated size:** Small (1 day)

---

### Phase Ordering Rationale

```
A → B → C → D ─┐
                ├─→ G → H
A → E ──────────┤
A → F ──────────┘
```

- **A first** because everything depends on PJVM and harness.
- **B→C→D** is the GXL critical path (most complex, highest risk).
- **E** (GIS) can parallelize with C/D if a second engineer is available.
- **F** (GCP) can parallelize with B/C/D/E — it's the simplest stream.
- **G** (wiring) requires all three adapters.
- **H** (cutover) is ceremonial once G is proven.

---

## 4. Risk Register

| # | Risk | Likelihood | Impact | Mitigation |
|---|------|-----------|--------|------------|
| R1 | **Semantic divergence between expr-lang and GXL** — existing runbooks rely on undocumented expr-lang behaviors (e.g., `contains` infix operator, implicit type coercion, nil propagation) that GXL deliberately forbids | High | High | Audit all 34 condition tests. Any behavior the tests rely on that GXL rejects is a BREAKING CHANGE requiring fixture update or spec amendment. Surface these in Phase D adapter testing. |
| R2 | **Executor coupling to `{{ }}` syntax** — some step fields in existing runbook YAML use `{{ .var }}` which won't match `${var}`. Schema migration is required in addition to engine swap. | Certain | Medium | The 22 design fixtures already use GXL/GIS/GCP syntax. But if there are USER-authored runbooks (outside this repo), they need a migration path. Document syntax-change as breaking in release notes. |
| R3 | **Performance regression** — hand-rolled recursive-descent + tree-walk may be slower than compiled expr-lang programs for complex expressions | Medium | Low | expr-lang compiles to bytecode; GXL will tree-walk. Mitigate: benchmark the conformance corpus; if any vector category >2x slower, add compilation cache or bytecode backend later. Unlikely to be user-visible given expression simplicity in real runbooks. |
| R4 | **Conformance vectors reveal spec ambiguities** — TESS-AMBIG-3 (boolean ordering), TESS-AMBIG-4 (array equality), TESS-CONFLICT-2 (unknown method), OI-GXL-03 (negative modulo) are unresolved | High | Medium | Each ambiguity that surfaces during implementation gets filed as an issue in gert-private. Design fix + new vector lands here; gert runtime resumes. Built into the workflow (§5). |
| R5 | **Capture path upgrade breaks CLI executor** — current CLI hardcodes `"stdout"/"stderr"/"exit_code"` keywords; GCP paths like `output.stdout` or `output.exit_code` require structural output wrapping | High | Medium | Phase F must define the PJVM shape of step output before GCP resolver can operate. Recommend: executor returns `{"stdout": "...", "stderr": "...", "exit_code": N}` as PJVM Object; capture paths resolve against this. Backward compat: keyword strings remain valid GDP paths (bare root identifiers). |
| R6 | **Parallel development conflicts** — if other gert features land while migration is in-flight, merge conflicts in executor files | Medium | Low | Feature-flag approach isolates changes. New eval code is in `internal/eval/` (new directory); only Phase G touches existing files. Keep G as a single focused PR. |
| R7 | **The 4-day sketch code (commits 97ce48b..5c550c0)** — is it usable? | — | — | **Recommendation: YES, use as starting point, not drop-in.** The sketch covers PJVM, Clock, harness, lexer, parser, evaluator, stdlib — exactly Phases A-C. It passed tv-gxl-parse (83/83) and tv-gxl-eval in its environment. However: it was written for `internal/eval/` in gert-private (wrong repo), never ran in the gert module context, and imports may need adjustment. Cherry-pick the relevant files, adapt package paths, verify against current conformance corpus. Do NOT treat it as production-ready — it had no code review and no integration testing. |

---

## 5. Cross-Repo Coordination

### 5.1 Spec Ambiguity Resolution Flow

```
gert runtime dev finds ambiguity
  → Opens Issue in ormasoftchile/gert-private titled "SPEC-AMBIG: [description]"
  → Tags: `spec-ambiguity`, `blocks-runtime`
  → Barbara (or designee) resolves in gert-private:
      1. Updates grammar/spec section if needed
      2. Adds conformance vector(s) that pin the resolved behavior
      3. Closes issue with commit reference
  → gert runtime resumes implementation against updated vectors
```

### 5.2 Conformance Vector Distribution

**Recommendation: Git submodule** pointing at `ormasoftchile/gert-private` checked out to `specs/conformance/` in the gert repo.

Rationale:
- **Submodule** provides pinned, reproducible vector sets (CI always tests against a known commit).
- `go test` can reference `specs/conformance/tv-*.yaml` directly.
- Updating is explicit: `git submodule update` in a PR — visible in review.
- Alternatives considered:
  - *Scheduled sync* — implicit, hard to debug when tests suddenly break.
  - *Manual copy* — error-prone, drift risk.
  - *Go embed* — requires vectors in the Go module; couples release cadence.

**Submodule path:** `specs/design` → maps to `design/gert/conformance/` in gert-private.

**Privacy note:** gert-private is private. If gert is public, the submodule must use a deploy key or the conformance files must be published to a separate public artifact. This is an **open question for ormasoftchile** (see §8).

### 5.3 Governance / Review Gates

| Gate | Scope | Reviewer(s) | Criteria |
|------|-------|-------------|----------|
| Per-PR | Each phase PR | 1 engineer + 1 architect review | Tests pass, no regressions, follows package layout |
| Phase-level | End of each phase | ormasoftchile sign-off | Conformance vector counts match expected; no blocked ambiguities |
| Pre-cutover (Phase G→H) | Migration complete | Full team review | All 264 green, fixtures execute, perf acceptable, soak period passed |

---

## 6. Estimated Effort

| Phase | Size | Estimate | Critical Path? |
|-------|------|----------|----------------|
| A: Foundation | Medium | 2-3 days | Yes (blocks all) |
| B: GXL Parser | Medium | 2-3 days | Yes |
| C: GXL Evaluator | Large | 3-5 days | Yes (longest single phase) |
| D: GXL Path + Adapter | Medium | 2-3 days | Yes |
| E: GIS Interpolation | Large | 3-4 days | Parallelizable |
| F: GCP Capture | Medium | 2-3 days | Parallelizable |
| G: Consumer Migration | Medium | 2-3 days | Yes (after D+E+F) |
| H: Cutover | Small | 1 day | Yes (final) |

**Critical path (serial):** A → B → C → D → G → H = **12-18 days**

**With parallelism (2 engineers):**
- Engineer 1: A → B → C → D → G → H
- Engineer 2: — → — → E → — → F → (assists G)
- **Total calendar time: ~12-15 days** (3 weeks with buffer)

**Bottleneck:** Phase C (GXL Evaluator + stdlib). The stdlib surface is large and every function must match spec semantics exactly. This is where most ambiguity-resolution cycles will land.

**If using the 4-day sketch:** Phases A-C could compress by ~40% (sketch covers most of the logic). Realistic estimate with sketch reuse: **10-12 days critical path**.

---

## 7. Non-Goals

This plan explicitly does NOT cover:

1. **C# or TypeScript runtime implementations** — separate repos, separate plans
2. **Parse-gate implementation** — separate proposal (Barbara backlog, due before Day 9)
3. **Schema migration tooling for user runbooks** — Stream E was removed; users adopt new syntax manually
4. **New step types or executor features** — only expression/interpolation/capture engines change
5. **Web platform integration** — web platform consumes the runtime; no changes needed there until runtime ships
6. **Performance optimization beyond "not catastrophically slow"** — bytecode compilation is a future concern
7. **GERT CLI UX changes** — `cmd/gert/` is unaffected by engine internals
8. **Governance evaluator** — `pkg/governance/evaluator.go` is unrelated to expression evaluation (naming collision only)

---

## 8. Open Questions for ormasoftchile

| # | Question | Options | Barbara's Lean |
|---|----------|---------|----------------|
| OQ-M1 | **Conformance vector distribution**: submodule vs. published artifact? | (a) Git submodule into gert (requires gert-private access from gert CI) (b) Publish tv-*.yaml to a public gert-conformance repo (c) Vendor/copy into gert with sync script | (a) if both repos stay private; (b) if gert goes public |
| OQ-M2 | **Sketch reuse**: Should the gert team cherry-pick commits 97ce48b..5c550c0 as starting material? | (a) Yes — cherry-pick, adapt, verify (b) Start fresh using spec + vectors only | (a) — the code is directionally correct and saves 3-5 days; just don't treat it as reviewed |
| OQ-M3 | **Feature flag mechanism**: build tag vs. runtime config? | (a) Build tag `//go:build gxl` (compile-time, zero overhead) (b) Runtime flag `--eval-engine=gxl` (runtime switch, useful for A/B but slight overhead) | (a) for simplicity; the flag exists only during migration |
| OQ-M4 | **Breaking syntax change communication**: How are existing gert users (if any) notified that `{{ }}` → `${}`? | (a) Major version bump (v2) (b) Deprecation period with both syntaxes supported (c) Just ship it (pre-1.0, no stability promise) | (c) if gert is pre-1.0; (b) if anyone is in production |
| OQ-M5 | **Who implements?** Single engineer or pair? | (a) One engineer, serial (b) Two engineers, parallel streams | (b) if timeline matters — E and F are fully parallelizable with B/C/D |

---

## Appendix: Package Layout Summary (Proposed Final State)

```
ormasoftchile/gert/
  pkg/
    expr/
      expr.go            ← interfaces (Evaluator, ConditionEvaluator) — UNCHANGED
    engine/
      engine.go          ← Config struct — UNCHANGED (holds same interfaces)
  internal/
    eval/
      core/
        value.go         ← PJVM types
        clock.go         ← Clock interface
        yaml.go          ← YAML → PJVM loader
      gxl/
        lexer.go
        parser.go
        ast.go
        evaluator.go
        stdlib.go
        path.go
        errors.go
      gis/
        parser.go
        resolver.go
        interpolator.go
        errors.go
      gcp/
        resolver.go
        errors.go
      adapter.go         ← GXLConditionAdapter, GISEvaluatorAdapter
      harness/
        conformance_test.go
        schema.go
    expr/                ← DELETED in Phase H
    executor/            ← capture logic updated in Phase F/G
```
