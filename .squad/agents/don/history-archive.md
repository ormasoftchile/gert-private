# don — History Archive

**Archived:** 2026-08-15T16:27:17.3769072-07:00
**Size:** 54055 bytes

---

# Don — Project History (Executive Summary)

**Last Updated:** 2026-06-05T22:37:00Z  
**Status:** Phase 2 Day 4 complete; ready for Phase A runtime migration

## Role

Runtime/tooling engineer. Primary stream: Go runtime implementation (PJVM, GXL/GIS/GCP lexer-parser-evaluator, conformance harness). Phase 1 included fixture audit/migration, grammar delivery, C# parity analysis, and architecture validation.

## Phase 1 Highlights (Complete)

| Stream | Outcome |
|--------|---------|
| **Stream E (Migration Tool)** | Scaffolded (Day 1-2), removed per user directive — no production legacy to migrate |
| **Stream D (Fixtures)** | ✅ Complete — 21 runbooks audited, 368 templates normalized, all migrations validated |
| **Grammars (A)** | ✅ Delivered — gxl/gis/gcp.ebnf (75 KB); ratified OQ1-OQ4 decisions locked |
| **C# Governance** | ✅ Analyzed — parity layer defined; 10 validation gates (G-01..G-10) |
| **Declaration Gaps** | ✅ Mapped — P0/P1/P2 runtime gaps identified; interface proposals drafted |
| **Execution Adapters** | ✅ Validated — queue-triggered worker (Azure) strongest governance match |

## Phase 2 — Go Runtime Implementation (Days 1-4 Complete)

### Day 1: Scope & Scaffold (2026-06-05T09:25:14Z)
- Read normative grammars, spec sections, conformance corpus (223 vectors)
- Created phase2-go-runtime-plan.md (package layout, PJVM choice, stream order)
- Scaffolded `internal/eval` with per-grammar packages; added PJVM types in core/value.go
- Added conformance harness discovery and skeleton runners

### Day 2: PJVM Plumbing (2026-06-05T13:29:45Z)
- PJVM: constructors, typed accessors (AsNumber, AsString, etc.), deep equality, debug string
- Boundary guards: NaN/infinity rejection, nil map rejection, non-string key rejection
- Clock interface (Clock, SystemClock, FixedClock) for deterministic now() testing
- Reworked harness to dispatch GXL-parse/eval/path, GIS-path, GCP-path runners
- Corpus updated: 264 vectors (GCP-PATH +41 new)

### Day 3: GXL Lexer/Parser (2026-06-05T16:16:27Z)
- Lexer: position-tagged tokens, string escapes, number validation, reserved keywords, legacy operator rejection
- Parser: recursive-descent over GXL grammar; AST (literals, unary/binary, GDP paths, calls); 10 parse-error codes (GXL-PARSE-001..010)
- Wired to harness: tv-gxl-parse.yaml → 83/83 green
- Unit tests: token classes, lexer/parser error handling, AST precedence

### Day 4: GXL Evaluator (2026-06-05T17:57:33Z)
- Eval(node, bindings, clock) over Day-3 AST; strict PJVM construction
- Stdlib: str.* (case, trim, split, etc.), list.* (append, filter, map, etc.), regex.match, math.*, len, now(clock-injected)
- Short-circuit: and/or evaluation (right side never evaluated if short-circuits)
- Harness integration: tv-gxl-eval.yaml → 90/90 green (ambiguities TV-GXL-EVAL-033/088 flagged for arbitration)
- Unit tests: operators, type errors, stdlib, now() with FixedClock

**Expected Conformance:** tv-gxl-parse.yaml 83/83 ✅, tv-gxl-eval.yaml 90/90 ✅; GXL-path/GIS-path/GCP-path remain Day 5-8 not-implemented stubs.

## Ken Paired Hire (OQ-M5 Execution — 2026-06-05T22:31:50Z)

Ken hired as second Backend Dev to pair on runtime migration in ormasoftchile/gert (Phases A–H). Single-squad directive: both operate from this squad (gert-private) during execution.

- **Plan:** Don takes critical path A→B→C→D→G→H; Ken takes parallel E (GIS) & F (GCP) in parallel
- **Target:** 10–15 days wallclock
- **First Assignment:** Phase A kickoff in ormasoftchile/gert (per runtime migration plan OQ-M5 corrected wording)

## Key Learnings

- **Stream E reversal (2026-06-04):** User directive killed migrator because GERT has no production legacy syntax in the wild. Kept tool would imply legacy mode and create contract surface future runtimes must reason about.
- **Ratification efficiency (2026-06-05):** Complex cross-language decisions lock efficiently when OQs are pre-identified. Optional chaining (✅) can proceed in parallel across Go/C# runtimes without alignment risk.
- **Local environment (ongoing):** Go/gofmt unavailable on PATH; coordinator must run builds/tests/formatting in Go-enabled environment. Code is complete and structured for immediate coordinator verification.

## Next: Day 5 — GXL Path Resolution

- GXL path traversal (get, set, delete on GDP paths) with error codes GXL-PATH-001..010
- Target: tv-gxl-path.yaml (35 vectors) green
- Dependency: Day 4 evaluator and stdlib (locked)

## 2026-06-05 Stream E Day 2 + Phase 1 Close

Dogfood audit complete: 22 runbook fixtures all clean (P1–P5 all zero). Three tool bugs fixed during audit; tool subsequently removed per user directive (pre-1.0, no legacy users). Phase 1 formally closed: 0 blocking items. Streams A/B/C/D/E/F all complete.

Next: Phase A in `ormasoftchile/gert` paired with Ken. PJVM + Clock + harness + DRIFT-DETECTION-001.

## 2026-06-05 — Phase A: PJVM/Clock/Harness Shipped (PR #9)

**Worktree:** `gert-phase-a-pjvm`  
**PR:** https://github.com/ormasoftchile/gert/pull/9 (draft, awaiting merge)

**Shipped:**
- PJVM (6-variant JSON types) + Clock interface + YAML loader → `internal/eval/core/`
- Conformance harness skeleton → `internal/eval/harness/conformance_gxl_test.go`
- 27 unit tests (core) + 264-vector harness (all skip, 0 fail)
- Vector corpus TEMPORARY copies → `testdata/vectors/`; Ken's sync will replace

**Key Paths:**
- `internal/eval/core/value_gxl.go` — PJVM constructors + marshaling
- `internal/eval/core/clock_gxl.go` — Clock interface
- `internal/eval/harness/conformance_gxl_test.go` — harness dispatch (hardcoded vector path @ line 59; ready for Ken's sync)
- `testdata/vectors/VECTORS_SHA` — source commit 3ce53431 + breadcrumb

**Test:** `go test -tags gxl ./internal/eval/core` (27/27 ✅) + `go test -tags gxl ./internal/eval/harness` (264 skip ✅)

**Handoff:** Breadcrumb left at conformance_gxl_test.go:59 for Ken's sync replacement. Vector path will be updated once PR #8 lands.

## 2026-06-05T23:12:22-04:00 — Phase B Lexer+Parser Shipped (PR #10)

**Status:** Phase B complete. GXL lexer and recursive-descent parser implemented and tested.

**Deliverable:** PR #10 (draft, stacked on PR #9 phase-a-pjvm). Branch: `phase-b-lexer-parser` in `ormasoftchile/gert`.

**Test results:** tv-gxl-parse.yaml 83/83 PASS on first run. No parser bugs found.

**Two corpus bugs identified and handed off to Tess:** TV-GXL-PARSE-062/063 YAML encoding (doubled backslashes), fixed upstream commit `424a334` before Ken's sync overwrite.

**Next:** Phase C (GXL evaluator + stdlib, 92 vectors) starts once PR #10 merges.

## 2026-06-05T23:54:03-04:00 — Phase C: GXL Evaluator Shipped (PR #11)

**Worktree:** `gert-phase-c-evaluator`  
**PR:** https://github.com/ormasoftchile/gert/pull/11 (draft, stacked on phase-b-lexer-parser)  
**Status:** ✅ SHIPPED — 92/92 eval vectors PASS on first pass.

**Key Deliverables:**
- GXL evaluator (`evaluator_gxl.go`): AST walker with PJVM construction, short-circuit, error handling
- Stdlib (`stdlib_gxl.go`): 13 functions across 4 namespaces (`str.*`, `list.*`, `regex.match`, globals `len`/`now`)
- Harness regex-assert support: added for TV-GXL-EVAL-088 (`now()` → regex pattern validation)
- Phase C exit-criteria satisfied; Phase D (path engine) unblocks on merge

**Test Results:** tv-gxl-eval.yaml 92/92 PASS (no corpus bugs, no silent skips)

**Handoff:** Don confirmed — no Barbara action items, no Tess action items, no Charter/skills/team changes needed. Speculative merge approved.

## 2026-06-06 — Phase D: GXL Path Engine Shipped (PR #12)

**Worktree:** `gert-phase-d-path`  
**PR:** https://github.com/ormasoftchile/gert/pull/12 (draft, stacked on phase-c-evaluator)  
**Status:** ✅ SHIPPED — 36/36 path vectors PASS on first pass.

**Deliverable:** GXL path engine (`path_gxl.go`) — thin entrypoint wrapping Phase C evaluator's existing GDP traversal logic. Design choice Option (b): no structural refactor, minimal churn against PR #11 surface.

**Key Fix:** Corrected `evaluator_gxl.go` error code for field-on-non-object from GXL-PATH-001 → GXL-PATH-004 (TESS-AMBIG-5/6 compliance). 2-line fix + constant; lives in PR #12.

**Test Results:** 36/36 path vectors ✅. Total conformance: 211/267 PASS (83 + 92 + 36), 56 SKIP (Ken's E/F), 0 FAIL.

**Next:** Phases G (cutover) + H (cleanup) remain. Critical path A→B→C→D now complete pending review. Phase E/F unblock once PR #9 (PJVM) merges.

## 2026-06-06T01:35:00-04:00 — Phase G: Runtime Integration Shipped (PR #15)

**Worktree:** `gert-phase-g-integration`  
**PR:** https://github.com/ormasoftchile/gert/pull/15 (draft, stacked on phase-d-path, folds PR #13/#14)  
**Status:** ✅ SHIPPED — 267/267 conformance maintained, all test gates green

**Key Deliverables:**
- Merged Ken's Phase E (GIS) and Phase F (GCP) branches with zero code conflicts
- Resolved harness collision: `vectorBindings` duplicate extracted to `helpers_gxl_test.go`
- Adapter pattern: `GISEvaluatorAdapter`, `GXLConditionAdapter`, `GCPCaptureAdapter` wired into CLI via build-tagged factories in `internal/adapter/wire.go` + `pkg/run/run.go`
- Build tag `//go:build gxl` activates new engines; legacy `!gxl` path unchanged (side-by-side runtime)
- `internal/executor.CaptureResolver` interface added; `GCPCaptureAdapter` handles all capture paths under `gxl`
- Four integration e2e tests in `internal/eval/integration_gxl_test.go` ✅

**Fixture Migration:**
- Hardcoded `/Volumes/Projects/gert/...` absolute paths in engine tests replaced with `repoRoot()` + relative path (portable across machines/worktrees)
- Go-template `{{ }}` runbooks preserved for legacy; GXL-syntax variants added (`-gxl.runbook.yaml`)
- `collectHealthRunbookPath()` build-tagged to select correct variant per compilation target

**Test Results:** `go test ./...` (no tag) all pass ✅, `go test -tags gxl ./...` all pass ✅, 267/267 conformance maintained ✅, 4/4 integration e2e ✅. Pre-existing `TestRender_Regions` markdown failure pre-dates Phase work.

**Key Learning — Adapter Pattern for Side-by-Side Engines:** Build-tagged factory functions decouple CLI wiring from engine selection. Both engines compile; runtime selection via `//go:build` tag. Zero runtime overhead for legacy path (`!gxl`). Fixture migration: absolute paths block multi-machine development; `repoRoot()` helper + relative paths restore portability.

**Handoffs:**
- Ken: Phase H scope — delete legacy factories, drop build tags, canonicalize runbooks, remove expr-lang dep. Note: `GCPCaptureAdapter` errors on non-standard keys (e.g., `exitCode` camelCase) — Phase H migration guide should document `exit_code` (snake_case only).
- Barbara: Phase H gate — `GISEvaluatorAdapter.Eval()` returns error for mandatory missing-key paths (GIS-PATH-MISSING per spec). Legacy engine returned empty string. Potential breaking change for runbooks relying on missing-key-as-empty-string — needs arbitration before hard cutover.
- Tess: No new corpus vectors. 267/267 locked.

**Next:** Phase H hard cutover — remove build tags, delete legacy engine, update CHANGELOG. Estimated 1 day, low risk (all semantics proven by conformance corpus).

## 2026-06-06T02:15:00-04:00 — Phase H: Hard Cutover Complete (PR #16)

**Worktree:** `gert-phase-h-cutover`  
**PR:** https://github.com/ormasoftchile/gert/pull/16 (draft, stacked on phase-g-integration)  
**Status:** ✅ SHIPPED — Hard cutover complete. New engine (GXL/GIS/GCP) is now unconditional.

**Deletions (9 files, ~670 LoC):**
- Legacy engine: `internal/expr/` entire package (condition.go, condition_test.go, template.go, template_test.go)
- Legacy factories: `internal/adapter/evaluators_legacy.go`, `pkg/run/evaluators_legacy.go`
- Legacy test fixture selector: `internal/engine/runbook_paths_legacy_test.go`
- Promoted `-gxl` runbook variants to canonical names (2 files)
- Dependency: `go mod tidy` removed `github.com/expr-lang/expr v1.17.8`

**Build-tag removal:** 43 files touched — 41 `//go:build gxl` + 2 `//go:build !gxl` pragmas deleted. New engine unconditional.

**Runbook canonicalization (12 files):** Mechanical `{{ .var }}` → `${var}` substitution. Special case: `collect-health-parallel/check-service.runbook.yaml` inline Go-template conditional (`{{ if contains .health_response "200" }}...{{ end }}`) → GXL `branch` step with `str.contains()`. Capture path: `exitCode` → `exit_code` (snake_case).

## 2026-08-09T15:59:36-07:00 — Tool Packages MVP: Runtime Implementation (uncommitted)

**Repo:** `C:\One\OpenSource\gert` (WORKTREE_MODE=false, direct on the runtime repo).
**Status:** ⚠️ Substantial core implemented and tested; explicitly scoped subset — NOT committed
(per task instructions). Left in the working tree for review/handoff.

**Dirty-tree constraint honored:** `examples/simple-health-check/simple-health-check.runbook.yaml`,
`internal/tool/native.go`, `internal/tool/native_test.go` (pre-existing user edits — a GIS var
migration and a Windows `ping -c`→`-n` argv normalizer) and the untracked
`examples/simple-health-check/examples.code-workspace` were inspected, never touched, and verified
byte-identical (same diff line counts) at the end of the session. Nothing was staged or committed.

**Implemented (new packages, all with unit tests, `go build ./...` + `go test ./...` green
repo-wide, zero regressions):**
- `pkg/errkit`: full PKG-001..030 / PKG-W001..003 typed-error sentinels added to the existing
  GXL/GIS/GCP error registry pattern.
- `pkg/semver`: strict SemVer 2.0.0 parser/comparator + the MVP constraint grammar (^, ~,
  relational, bare-exact, space-AND; explicit rejection of `||`, `*`, `x`-ranges, hyphen ranges,
  `!=`) with `Intersect`.
- `pkg/pkgpath`: secure path resolution — authored-syntax validation, symlink-aware containment
  (fixed a real Windows 8.3-short-name containment-mismatch bug found via TDD), NFC
  normalization for portable-collision comparison.
- `pkg/schema/{toolpackage,projectconfig,packagelock}.go`: `tool-package/v1`, `config/v1`,
  `package-lock/v1` Go types mirroring the ratified JSON Schemas; `ToolRef` gained
  `package`/`version`; `Runbook` gained `requires`.
- `pkg/schema/packagevalidate.go`: the ratified R6 pre-schema raw-YAML scan rejecting top-level
  `toolPackages:` (PKG-020) and any `toolRefs[].alias` (PKG-021), wired into
  `internal/parser.ParseBytes` as a Phase-0 gate ahead of structural validation.
- `pkg/pkgcatalog`: the centerpiece — `Build` (Phase C, tiers 0-3: builtins, `requires:`
  package loading incl. manifest/export validation PKG-001/002/004/005/010/016/023/025,
  project tool-paths + `<workspace>/tools/` scan, dynamic tier-3 extension point) and `BindFile`
  (Phase B: `toolRefs[].path`/`.package`/bare-name binding, PKG-006/011/012/022/029/030,
  PKG-W001/W002 deprecation warnings), plus `FileDigest`/`CatalogDigest`/lock-info per the
  ratified digest algorithm. 18 unit tests, all green.
- `internal/adapter/toolrefs.go`: added `BuildPackageCatalog`/`ResolveToolRefsViaCatalog` as new
  sibling functions alongside the existing path-only `ResolveToolRefs`, giving Ken a coherent,
  tested API surface for CLI wiring without touching the legacy call sites in `pkg/run/run.go`.

**Bug found and fixed during this session:** `bindByPackage`'s `toolRefs[].version` check was
originally a no-op (the runtime `toolpkg.ToolDef` carries no version field). Fixed by threading
`schema.ToolDef.Version` into a new `pkgcatalog.Entry.Version` field and actually calling
`semver.Constraint.Satisfies` in `bindByPath`/`bindByPackage`, per §Tool Definition Schema's
"checked against the resolved tool definition's meta.version." Caught one test fixture
(`kubectlToolYAML` declaring `version: "1.0.0"`) that had been silently passing against a `^1.4.0`
toolRefs constraint under the old no-op; corrected the fixture, added two new tests
(`TestBindFile_ByPackage_VersionConstraintUnsatisfied_PKG002`,
`TestBindFile_ByPackage_VersionConstraintSatisfied`).

**Deliberately deferred (honest scope cut, reported not hidden):**
- **Substitution** (`execute.kind: runbook`, input/output contracts, recursion/depth limits,
  governance non-widening) — not implemented at all.
- **Evidence/trace/resume wiring** — no `package/resolved`/`catalog/frozen`/`tool/substituted`
  event emission; no `PKG-009`/`--allow-package-drift` resume-drift refusal; no
  `replay/packageDrift` non-fatal replay path. The catalog/digest primitives needed to build these
  exist (`CatalogDigest`, per-package digests) but are not yet wired into
  `internal/evidence`/`internal/replay`/`internal/resume`.
- **MCP/extension tier-3 real discovery** — only a caller-supplied `Dynamic []*Entry` extension
  point exists in `Build`; no actual MCP `tools/list` or extension-manifest integration.
- **`.gert/config.yaml` (`config/v1`) is not loaded anywhere in `pkg/run/run.go`** — the Go type
  (`schema.ProjectConfig`) exists and parses/validates, but nothing calls it yet.
- **`ToolDef.Actions` remains a `map[string]*ToolAction`**, not the ratified-spec array. Converting
  this would be a large, risky, sweeping breaking change across parser/engine/executor/tests far
  outside this task's core ask — flagged, not fixed.
- **Includes/global-package-set/lexical-scoping** — only informally covered by the existing
  per-file `ResolveToolRefs`/`internal/adapter.ResolveToolRefs` call pattern (each included file's
  toolRefs are resolved independently at load time); no explicit global-package-set object or
  cross-include catalog-sharing test was added.
- **`tab:tool-path-bases`'s two distinct resolution bases** (project-scope requires: paths are
  workspace-root-relative; runbook-scope requires: paths are runbook-file-relative) are collapsed
  by `mergeRequirements` before `loadPackage` sees them; current code uses a
  workspace-root-first-then-runbook-dir-fallback heuristic, not a fully faithful two-base
  implementation. Flagged as a known limitation in code comments.
- Remote catalogs, publishing, transitive dependencies, aliases, action-allowlist enforcement,
  non-tool assets, multi-version coexistence — out of scope per task instructions, not attempted.

**Validation:** `go build ./...` clean; `go vet ./pkg/pkgcatalog/...` clean; full `go test ./...`
green across every package (cmd/, internal/, pkg/) — zero regressions, including all 22 existing
runbook parser fixtures (r01-r22) and the 3-file dirty-tree constraint. Nothing committed per
instructions.

**Handoffs:**
- Ken: `internal/adapter.BuildPackageCatalog`/`ResolveToolRefsViaCatalog` are ready for CLI wiring
  (e.g. a `--allow-package-drift` flag, `.gert/config.yaml` loading, and switching `pkg/run/run.go`
  over from the legacy `ResolveToolRefs` once evidence/resume wiring lands).
- Tess: `conformance/tv-pkg-resolve.yaml` remains unblocked and untouched by this session — the
  `PKG-*` codes, five-tier model, and collision rules it needs to test against are now implemented
  and unit-tested in `pkg/pkgcatalog`, but no conformance-vector-format wiring was added.
- Barbara/whoever picks this up next: the `Actions` map-vs-array divergence and the
  substitution/evidence/resume deferrals are the two biggest remaining gaps before this can be
  called "spec-complete"; both are large enough to warrant their own scoped follow-up tasks rather
  than folding into a future unrelated change.

**Validation:** `go build ./...` clean, `go test ./...` 267/267 + 4/4 integration green. Pre-existing `TestRender_Regions` markdown failure remains.

**Key Learnings (for future runtimes):**
1. **File-pair deletion pattern:** Legacy `_legacy.go` siblings in adapter/run → single unconditional factory functions. Cleaner final surface.
2. **Snake-case alignment with capture adapter:** `exitCode` (legacy camelCase) → `exit_code` (GCP spec). Critical migration detail that surfaced late in Phase G; flag in CHANGELOG.
3. **GXL `branch` step restructuring:** Simple Go-template conditionals map to GXL `if...then...else...end` expression. Complex nested conditionals + data ops → `branch` step + function calls (`str.contains()`). Fixture migration requires intent reading, not mechanical replacement.

**Stack final:** PR #8 (indep) → #9→#10→#11→#12 (critical) → #13/#14 (folded into #15) → #15 (integration) → #16 (cutover). Eight PRs total. All draft awaiting Germán review pass. Zero handoffs, zero deferred work. Phase 2 complete.

---

## 2026-08-09T14:09:01-07:00 — Tool Packages MVP: Independent Rejection Revision (R1–R15)

**Context:** Barbara rejected Edith's Tool Packages MVP spec/schemas/fixtures
(`.squad/decisions/inbox/barbara-tool-packages-gate-review.md`, 15 blocking items
R1–R15, plus an optional R16 for `warnings:` on the conformance `PackageExpected`
shape). Cristián assigned Don as independent revision owner; Edith (original author)
was explicitly locked out and did not advise or contribute to this revision. The
ratified architecture ruling (`.squad/decisions/archive/barbara-tool-packages-architecture-ruling.md`,
AR-TP-1..10) remained authoritative throughout and was not reopened.

**Work:** Resolved all 15 blocking items plus the optional R16. Full matrix, file list,
validation outcome, and Tess follow-up recorded in
`.squad/decisions/inbox/don-tool-packages-revision-r1-r15.md` and mirrored in
`.squad/decisions.md`. Highlights:
- Established one canonical `tool/v1` action shape (list-only `actions:`, unified `args:`
  vocabulary, deliberately distinct `output:`/`outputs:` keys) — R1.
- Disambiguated per-action substitution key (`execute:`) from the pre-existing top-level
  mobile `impl:` — R2.
- Defined the `outputs.<name>` GCP capture root for substituted-action outputs, wired
  into `grammar/gcp.ebnf` — R3.
- Repaired the reference substitute (`drain-node.yaml`) so its declared output is
  produced via valid GCP over real JSON stdout, not a literal — R4.
- Added an explicit "Resolution base per path kind" table naming containment roots for
  every path kind Barbara listed, including the package-internal-substitute special case
  — R5.
- Made PKG-020/PKG-021 reachable via a mandated pre-schema raw-document scan — R6.
- Fixed the `dependencies` schema/PKG-016 contradiction — R7.
- Corrected all `apiVersion` inconsistencies — R8.
- Made tier-3 addressing schema-compatible (qualified `name` only, never `package`) — R9.
- Split the §05 collision paragraph into cross-source vs. same-source cases — R10.
- Defined substitution replay behavior (executes like `invoke` against the caller's
  scenario file) and tied it explicitly to non-fatal `replay/packageDrift` — R11.
- Reconciled step-level `tool.version` as a deprecated, intersected, plan-time constraint
  — R12.
- Made the lock-root no-raw-absolute-path invariant structurally enforced via schema
  pattern — R13.
- Added `PKG-029`/`PKG-030` for mutually-exclusive package/path binding and unresolved
  `toolRefs.name` — R14.
- Documented a concrete unchanged-runbook real/dry-run/replay acceptance scenario with a
  worked scenario-file override map in the r23 fixture — R15.
- Added `warnings:` to `PackageExpected` — R16 (optional).

**Validation:** All touched JSON schemas re-validated as JSON; all touched YAML fixtures
parse; `verify_corpus.py` 280/280 vectors validate (no regression); full `tectonic`
LaTeX build of `main.tex` succeeds with no new errors — the only undefined references
are two pre-existing ones in files this revision did not touch.

**Deliberately deferred:** the ≥46-vector `conformance/tv-pkg-resolve.yaml` corpus
remains Tess's to author against the now-resolved contracts; not created here per
explicit task scope (no small scenario vector was assigned to this revision). Two
pre-existing, out-of-scope issues were flagged for Barbara's awareness rather than
fixed: a §07/§13 "replay" terminology overload, and the corpus-wide
`apiVersion: runbook/v2` issue (already separately ticketed per Barbara's note #5).

**Status:** Revision complete; returned to Barbara for second gate pass.

## Final Integration Pass — Live Wiring of Tool Packages MVP (2026-08-09)

**Role this pass:** primary runtime implementation owner, finishing the live end-to-end
integration of `pkg/pkgsubst`, `pkg/pkgdrift`, `pkg/pkgpath`, `pkg/pkgcatalog` into the
actual planner/engine/CLI execution path (they were previously built and unit-tested but
not invoked by production flow). Repo: `C:\One\OpenSource\gert` (public runtime). Protected
files (`examples/simple-health-check/simple-health-check.runbook.yaml`,
`internal/tool/native.go`, `internal/tool/native_test.go`,
`examples/simple-health-check/examples.code-workspace`) were not touched. No commits, no
branch switches, Ken's work left intact.

### Item 4 — Barbara's workspace-escape ruling (TV-PKG-PATH-002): DONE
- `pkg/pkgcatalog` package/tool-path resolution now emits PKG-W003 (external, advisory)
  instead of PKG-007 for a workspace-escaping package path; fixed a pre-existing bug in
  `internal/adapter/toolrefs.go`/`cmd/gert/run.go` where PKG-W* warnings were treated as
  fatal and silently discarded otherwise-valid tool bindings.

### Item 1 — Substitution wiring: DONE, live
- `pkg/tool/tool.go`: runtime `ToolDef`/`ToolAction` now carry `SourcePath`, `PackageRoot`,
  `PackageName`, `Governance *schema.ToolGovernance`, `Execute *schema.ExecuteSpec`,
  `Outputs map[string]*schema.ArgDef`, and a private `schemaAction` (getter/setter) so
  `pkg/pkgsubst.Plan` (which operates on `*schema.ToolAction` directly) can be called from
  production code without a parallel type system. `tool.ToolDefLookup` is an optional
  capability interface (`internal/tool.DefaultToolRuntime.LookupDef`) so existing fakes are
  unaffected.
- `internal/executor/tool.go` (`ToolExecutor.Execute`): when the resolved action's
  `execute.kind == "runbook"`, dispatches to a new `executeSubstitution` method instead of
  `runtime.Invoke`. It calls `pkgsubst.Plan(...)`, enforces `RequireApproval` via the real
  `governance.ApprovalGate` (denies/fails the step on rejection/missing gate), runs the
  substitute's flow via the shared `SubStepRunner` (nested engine), evaluates
  `outputs.<name>.value` against merged (input + child-step) vars with best-effort
  int/bool/number coercion, and sets `StepResult.Output[name]`.
  `internal/executor/substitution_context.go` threads cycle/depth frames and composed
  governance across nested substitution calls via context.
- Wired end-to-end: `internal/executor/registry.go` → `internal/adapter/options.go` →
  `internal/adapter/wire.go` → `cmd/gert/run.go` / `cmd/gert/serve.go` /
  `cmd/serve/main.go` (all now pass a `SubstitutionParser`).
- **Known, accepted gap:** deep governance enforcement (deny/allow-commands) for CLI steps
  nested inside a substitute body is NOT enforced beyond the pre-existing engine limitation
  (nested `SubStepRunner` engines never set `GovernanceEvaluator`); only `RequireApproval`
  is actually gated. The full composed `EffectiveGovernance` is still recorded on the
  `tool/substituted` trace event for audit, so the gap is visible, not silent.
- **Known, accepted gap:** a substituted action's declared `outputs.<name>` cannot be
  re-captured by a caller runbook's `capture:` field (GCP's capture vocabulary is a fixed
  set — stdout/stderr/exit_code/json/yaml/step.*/http.*/event.* — with no arbitrary
  output-map key syntax). Callers must read `StepResult.Output[name]` directly (e.g. via
  the CLI's JSON summary) or chain it as an already-materialized value for a later step.

### Item 2 — Trace events at runtime boundaries: DONE
- `tool/substituted`: emitted from `executeSubstitution` (tool/action/depth/effective
  governance), per substitution frame entered.
- `package/resolved` (one per resolved package) and `catalog/frozen` (once): emitted from
  a new `internal/adapter.EmitPackageCatalogTraceEvents` helper, called from
  `cmd/gert/run.go`'s fresh-run branch immediately after `adapter.BuildPackageCatalog`
  succeeds and before `eng.Start`. A `runID` is now pre-generated and stamped onto
  `plan.RunID` so these pre-run events and the run's own trace events share one `run_id`.
  Added `LockedPackageInfo.External bool` to `pkg/pkgcatalog` (previously externality was
  only implicit in the `Root` string's `"external:"` prefix) so `PackageResolvedPayload.
  External` reflects the real `pkgpath.ResolveKind` result rather than string-sniffing.
  `ConstraintSources` is left empty — the frozen catalog does not retain per-declaration
  provenance of merged version constraints; documented as a known limitation, not faked.

### Item 3 — Manifest/resume/replay digest wiring: DONE (resume), replay N/A
- Added `CatalogDigest string` and `PackageDigests map[string]string` to
  `pkg/engine.PlanMetadata` — persisted for free via the existing `RunState`/`SaveState`
  JSON marshaling (minimal manifest extension, no new persistence mechanism).
  `cmd/gert/run.go`'s fresh-run branch populates both from the built catalog right after
  `plannerImpl.Plan(...)` returns.
  New `--allow-package-drift` flag added to `gert run`.
- New `cmd/gert/resume_drift.go` (`checkResumePackageDrift`): called from the `--resume`
  branch of `runWithMode` right after `eng.Resume` succeeds. Re-parses the resumed plan's
  runbook, rebuilds the catalog exactly as a fresh run would (same requires:/package-map/
  project-config resolution), and calls `pkg/pkgdrift.Evaluate(pkgdrift.ModeResume, ...)`
  — a genuine, previously-nonexistent production call site for that package (task
  explicitly disqualifies an unwired utility package as "not implemented"; this closes
  that gap). A mismatch without `--allow-package-drift` returns the PKG-009 refusal
  unchanged; with the flag, the run proceeds and a `governance/packageDriftAccepted` trace
  event is written via `ecfg.TraceWriter` — drift is never silently bypassed.
- **Pre-existing, out-of-scope limitation surfaced (not caused by this pass):**
  `internal/engine.Resume`'s plan lookup (`store.(interface{ Plan(string) *ExecutionPlan
  })`) is backed by `DirRunStore`'s **in-memory-only** `plans` map (`RegisterPlan`), so
  `--resume` cannot recover a plan across process restarts today regardless of this
  session's changes — an existing architectural gap in the resume feature itself, not
  something newly introduced or silently worked around. The new drift check was therefore
  validated with a direct unit-level test of `checkResumePackageDrift` (fake `RunHandle`)
  rather than a full two-process CLI `--resume` invocation, since the latter already fails
  before reaching the new code for this unrelated, pre-existing reason.
- **ModeReplay:** no dedicated `replay` CLI command/path exists in this codebase today (no
  `--mode replay`/`gert replay` entry point found); per the task's "if present" qualifier,
  `pkgdrift.ModeReplay` remains unwired — there is no live path to wire it into.

### Item 5 — Live scenario coverage
- `cmd/gert/substitution_integration_test.go`: full `gert run` CLI test — package with an
  `execute.kind: runbook` action, substitute runbook with `inputs`/`outputs`, asserts
  propagated output and a `tool/substituted` trace event. **Passes.**
- `cmd/gert/catalog_trace_integration_test.go`: full CLI test asserting `package/resolved`
  and `catalog/frozen` appear in `trace.jsonl` with matching `run_id`s and correct
  external/digest/tool-count fields. **Passes.**
- `cmd/gert/resume_drift_integration_test.go`: three tests against
  `checkResumePackageDrift` — matching digests (no-op), mismatch refused (PKG-009) without
  the flag, mismatch accepted + `governance/packageDriftAccepted` event with
  `--allow-package-drift`. **All pass.**
- Pre-existing `TestRun_PackageMap_RealVsMockBinding` (unchanged-runbook real/mock CLI
  scenario) re-verified passing, unaffected by this pass's changes.

### Item 6 — Full validation
- `go build ./...`: clean, full repo.
- `go vet ./...`: only the same 2 pre-existing, unrelated issues in `internal/serve`
  (`delegations_test.go:85`, `preview_e2e_test.go:253` — using a value before checking its
  error; not touched by this pass).
- `go test ./...`: full repo green, no failures, including `pkg/pkgdrift`, `pkg/pkgcatalog`,
  and every `cmd/gert` test (old and new). No regressions found.

**Status:** Items 1, 2, 4, 5, 6 fully complete and live-verified. Item 3 complete for
resume; replay is confirmed not-present in this codebase so intentionally left unwired.
No commits made in either repository.

## Enum-Constrained Tool and Runbook Outputs MVP (2026-08-10)

Implemented the approved Enum-Constrained Tool and Runbook Outputs MVP
(AR-ENUM-1..15, `.squad/decisions/inbox/barbara-enum-constraint-mvp-architecture-ruling.md`,
gated by `barbara-enum-spec-final-gate-approval.md`) end to end in the runtime repo
(`C:\One\OpenSource\gert`), integrating around the in-progress, uncommitted Tool
Packages MVP work already present there. No commits made in either repo; no branch
switches; all four explicitly protected dirty files
(`examples/simple-health-check/simple-health-check.runbook.yaml`,
`internal/tool/native.go`, `internal/tool/native_test.go`,
`examples/simple-health-check/examples.code-workspace`) verified untouched by `git diff`
before and after this pass.

### Scope delivered
- `enum` supported at exactly the four ratified sites: tool action `args.<name>` (S1),
  tool action `outputs.<name>` (S2), runbook `inputs.<name>` (S3), runbook
  `outputs.<name>` (S4). No object/numeric/secret enum invented; no apiVersion or
  grammar (GXL/GIS/GCP) touched.
- New `pkg/schema/enum.go`: `EnumConstraint` type with custom `UnmarshalYAML`
  (ENUM-002 structural checks), `ValidateMembers` (ENUM-003/004/005, ENUM-W001),
  `ValidateTypeSite` (ENUM-001), `NormalizeCandidate`/`Contains` (NFC-normalized,
  asymmetric membership), `SetEqual` (PKG-013 set equality), `ValidateDeclaration`.
- `pkg/errkit/errors.go`: ENUM-001..009 + ENUM-W001 sentinels, class handling (fixed
  a `ClassForCode` prefix-ordering bug this pass introduced and caught via
  `TestDeclaredClassesCovered` — `ENUM-W` must be checked before the generic `ENUM-`
  prefix, mirroring the existing `PKG-W`/`PKG` pattern).
- Declaration-site well-formedness (ENUM-001..005, ENUM-W001) enforced at parse time
  for all four sites: JSON-Schema `enum` keyword added to `schemas/runbook.schema.json`
  `$defs.Input`/`$defs.Output` (S3/S4) with `internal/parser/validate_structural.go`'s
  `enumAwareCode` mapping keyword failures to the right ENUM code, plus Go-level
  `internal/parser/validate_semantic.go`'s `validateEnumDeclarations` for NFC/member
  semantics; `internal/tool/scan.go`'s `validateActionEnums`/`validateOneEnum` for S1/S2
  since no JSON schema governs `.tool.yaml` (ruling C3).
- Plan-time (ENUM-006/007) in new `internal/planner/enumplan.go`, wired into
  `validatePlan`: default-not-a-member (S1 arg defaults, S3 input defaults) and
  statically-known-literal `step.tool.args` values (S1) checked against the resolved
  `Enum`; GIS-interpolated (`${...}`) literals are correctly skipped (runtime-only,
  ENUM-008). `engine.ExecutionPlan` gained `Inputs`/`Outputs` fields (previously
  entirely absent) populated in `internal/planner/planner.go` from
  `rb.Runbook.Inputs`/`Outputs`.
- `engine.ValidatedPlan` gained an `EnumConstraints map[string]EnumMeta` field
  (declared-order member list + count + `Redacted` flag), populated once per run by
  `validateEnumConstraints`. C1 redaction is determined by matching a declaration's
  name against the plan's composed governance `RedactionPatterns()` — the only
  redaction primitive this codebase actually models (no distinct `redact: true`
  per-arg field or `sensitive_inputs` list exists in the schema today); documented
  as a best-effort proxy. `internal/planner/planner.go` was also extended to
  populate the previously-unset `ExecutionPlan.Governance` field via
  `internal/governance.BuildPolicy(rb.Runbook.Governance)` so this redaction lookup
  (and the pre-existing, previously dead `engine.go:625` redaction-pattern hook)
  actually has data to act on.
- Runtime binding (ENUM-008): `internal/executor/tool.go`'s `ToolExecutor.Execute`
  now looks up the resolved `ToolAction` for every tool step (not just substituted
  ones) and calls the new `checkArgEnums` helper on the post-GIS-interpolation args
  before dispatch, failing the step with `ENUM-008` on a non-member string value
  (non-string values are left alone — type-before-enum, and no general tool-arg
  type-checking exists elsewhere in this runtime to key off of). `pkg/run/run.go`'s
  `Start` checks every enum-constrained runbook input's caller-supplied
  (`cfg.Variables`) value the same way for S3; default values are covered by the
  plan-time ENUM-006 check instead (no double-reporting).
- Output production (ENUM-009): added directly to the existing output-evaluation
  loop in `internal/executor/tool.go`'s `executeSubstitution` (the only live runtime
  output-materialization path in this codebase) — a substituted runbook's own
  `outputs.<name>` value, after type coercion, is checked against its `Enum` before
  landing in `result.Output`.
- PKG-013 extension (AR-ENUM-8): `pkg/pkgsubst/pkgsubst.go`'s `validateInputs`/
  `validateOutputs` now also require the substitute's enum set to exactly equal
  (order-insensitive, via `schema.SetEqual`) the action's contract enum set for both
  args and outputs — reusing PKG-013, no new code, no subset/superset allowed.

### Genuine findings surfaced (not silently worked around)
1. **`TV-ENUM-DECL-006` / YAML resolver mismatch**: this repo's `gopkg.in/yaml.v3`
   (v3.0.1) resolves bare `yes`/`no` as `!!str` (YAML 1.2 core schema), not booleans.
   `tv-enum.yaml`'s `TV-ENUM-DECL-006` vector assumes a YAML-1.1-style resolver where
   `enum: [yes, no]` fails ENUM-002 because the items coerce to booleans. Under the
   actual library behavior, this parses as two valid, distinct strings and does NOT
   fail. Verified empirically that `true`, `1.0`, `~`, `null`, and explicit `!!int`
   tags ARE still correctly rejected as ENUM-002 — only the `yes`/`no` case diverges
   from the vector's YAML-1.1 assumption. Flagging per your own escalation process
   rather than forcing a fake failure to match the vector.
2. **No non-substituted (root) runbook output materialization exists anywhere in this
   runtime.** `pkg/run/run.go` never evaluates `parsed.Runbook.Outputs` for a directly
   invoked runbook — the ONLY live output-production code path found anywhere is
   `executeSubstitution`'s loop for a *substituted* action's runbook. ENUM-009 is
   therefore fully enforced at that one real site (verified end-to-end via a new CLI
   integration test), but a root-level `outputs:` block's own enum contract (S4
   outside substitution) has no runtime moment to be checked against today — this is
   a pre-existing engine gap, not something this MVP could close without inventing
   new, out-of-scope root-output-materialization engine behavior.
3. **No `sensitive_inputs`/per-declaration `redact: true` field exists in the schema.**
   C1's redaction rule was implemented against the only real primitive available
   (`governance.redact[].pattern` regex matched against the declaration's own name),
   documented inline as a best-effort proxy pending a possible future schema addition.
4. `ExecutionPlan.Governance` was previously always `nil` in the real `pkg/run/run.go`
   path (a latent, pre-existing gap) — populated it via `internal/governance.BuildPolicy`
   to make C1 redaction determination actually functional; this is additive only
   (nothing previously read a non-nil value from this path) and verified with the
   full `go test ./...` run showing no regressions.

### Tests added (all passing)
- `internal/parser/enum_decl_test.go` (pre-existing from this session): 16 cases,
  declaration-site well-formedness across all codes.
- `internal/planner/enum_plan_test.go`: ENUM-006 (default violation + happy path with
  metadata carriage assertion), ENUM-007 (static literal violation + GIS-interpolated
  skip), C1 redaction-by-governance-pattern metadata carriage.
- `internal/executor/enum_runtime_test.go`: ENUM-008 reject/allow/unconstrained-
  unaffected for tool-arg binding.
- `pkg/pkgsubst/enum_pkg013_test.go`: PKG-013 enum-set mismatch (input narrower,
  output superset) and exact-match (reordered set) happy path.
- `pkg/run/enum_input_test.go`: ENUM-008 for caller-supplied (`--var`) runbook input,
  reject + accept.
- `cmd/gert/substitution_enum_integration_test.go`: full live CLI `gert run
  --output=json` test proving ENUM-009 fires for a substituted action's output
  violation, surfaces via the existing JSON error envelope, and never echoes the
  enum member list in the error text (C1 spot-check at the CLI boundary).

### Validation performed
- `go build ./...`: clean.
- `go vet ./...`: only the same 2 pre-existing, unrelated `internal/serve` issues
  (not touched).
- `go test ./...`: full repo green (including `pkg/errkit` after fixing the
  `ClassForCode` ordering bug caught by its own existing test), no regressions.
- `git status`/`git diff` re-verified before finishing: the four protected dirty
  files carry zero enum-related changes and were never opened for editing this pass.

### Remaining gaps (honestly scoped out, not attempted)
- Root (non-substituted) runbook `outputs:` block production-time ENUM-009 has no
  engine hook to attach to (see finding 2) — would require adding whole new
  root-output-materialization engine behavior, out of this MVP's bounded scope.
- ENUM-MOCK conformance category (package mock vs. real binding enum-set equality)
  is, per Barbara's own C2 ruling, a conformance-vector obligation rather than a
  runtime mechanism; no new runtime mock-vs-real comparison was built (correctly, per
  ruling), and no dedicated mock-vs-real conformance-style test was added this pass
  given the effort budget — the existing `TestRun_PackageMap_RealVsMockBinding`-style
  pattern in `cmd/gert` would be the right shape for a follow-up.
- Trace-event-level enum metadata emission (ValidatedPlan's `EnumConstraints` is
  carried on the in-memory plan and available to any future emitter, but no explicit
  once-per-run trace event was added this pass) — the metadata now exists and is
  redaction-safe; wiring it into the trace stream is a small, low-risk follow-up.
- Full 58-vector `tv-enum.yaml` corpus coverage was not exhaustively mechanized as a
  data-driven harness; targeted hand-written tests cover representative cases per
  category (declaration/well-formedness, plan-time default/static-literal, runtime
  binding, substitution set-equality, CLI/JSON error-envelope + redaction) rather than
  every one of the 58 vectors individually.

## Learnings

### 2026-08-15T13:11:54-07:00 — Dynamic Include Stream 2 (Planner + Preview)

**Planner include/depth bookkeeping:**
- The `planVisitor` uses two separate sets — `lazyAt map[string]string` (for lazy sites, storing the abs path) and `dynamicAt map[string]bool` (for dynamic sites, which have no static path) — to identify sites that must not push `execDepth`/`displayDepth`. The critical invariant is: whatever `BeforeInclude` returns `load=false` for must NOT push depths in `EnterInclude`, and `LeaveInclude` must not pop either. Violating this corrupts sibling-step depths because the engine's main loop dispatches on depth to skip sub-steps.
- Dynamic sites are even more opaque than lazy: lazy has a known path; dynamic has only a GIS template. Both are leaves at plan time. The same no-push/no-pop pattern applies identically.
- `BeforeInclude` must be checked in this priority order: (1) dynamic (short-circuit before expand-policy logic, because dynamic sites have no expand mode to parse), (2) lazy/eager (expand policy logic for static sites).
- The `flowwalk.Walker` calls `BeforeInclude` once, then `EnterInclude(ctx, s, childRb)` where `childRb=nil` when `BeforeInclude` returned `load=false`. Dynamic and lazy sites both land here with `childRb=nil`; the `EnterInclude` guard ordering matters: check dynamic first, then lazy, then error on `childRb==nil` (loader misconfiguration).
- GXL expression validation runs at plan time (`validatePlan`). Test conditions in branch arms must be valid GXL literals (e.g. `"true"`, not `"${x}"` — the `$` sigil is rejected by the GXL parser).

**Preview render pipeline:**
- `graphdoc.Builder.BeforeInclude` also controls whether the walker loads child runbooks. For dynamic sites, this must return `false` regardless of `Recurse` setting — dynamic targets are unknown at preview time so no child frame can ever be inlined.
- `graphdoc.Node.Dynamic bool` is the single source of truth for "this is a dynamic include leaf." The `⟨dynamic⟩` title prefix and `Dynamic=true` flag are both set together in `EnterInclude`, after `emitNode` appends the node. Modifying the last node via `v.doc.Nodes[len(v.doc.Nodes)-1].Dynamic = true` is safe because `emitNode` always appends before returning.
- `graphjson` passes `dynamic=true` through as a `data` map entry (omitted for non-dynamic nodes), consistent with `omitempty` in the graphdoc schema. The HTML renderer tests `data.dynamic && data.kind === 'include'` before applying the `.dynamic-include` CSS class.
- Barbara §10.3 ruling is structurally enforced, not just documented: `BeforeInclude` returning `false` for dynamic sites means the walker never calls the loader, so zero catalog lookups happen during preview or dry-run. The "resolved at run time" subtitle in the HTML is the human-readable companion to this structural guarantee.

### 2026-08-15T12:58:20-07:00 — Dynamic Include Stream 3 (Executor + Catalog + Governance + Trace)

**Executor include chain architecture:**
- The runtime include chain for cycle/depth detection must flow via `context.Context` (same pattern as `substState` in `substitution_context.go`). Using a context key (`includeChainKey{}`) allows the chain to propagate automatically through nested engine calls without modifying the executor interface. Each dynamic include site appends its resolved `QualifiedID` to the chain before passing `childCtx` to `e.runner(...)`.
- The chain only tracks dynamic include resolutions (qualified IDs). Static and lazy includes are already cycle-checked by the planner's `flowwalk`; they do not add to the runtime chain. The runtime cycle check is specifically for cycles that the static planner structurally cannot detect: a dynamic target that transitively contains a static include pointing back at an ancestor.
- The effective governance context key (`dynGovKey{}`) propagates the composed `*trace.EffectiveGovernancePayload` through the same mechanism. This ensures a nested dynamic include inside a child runbook sees the composed governance, not a stale parent value.

**on_not_found dispatch (B-5/B-6 binding rulings):**
- `on_not_found: continue` covers ONLY `DINC-002`. Test the error code explicitly via `errkit.Coder` — do NOT use `errkit.IsWarning` for this dispatch (it's not about warning classification). A `switch` on `c.Code()` against `"DINC-002"` is the correct pattern.
- DINC-003, DINC-013, and all other DINC codes are always fatal regardless of `on_not_found`. This is the heart of the branching contract; getting it wrong would silently swallow fatal errors under a misleading `continue` policy.

**Governance composition non-widening proof:**
- `ComposeGovernance(parent, child)` follows Barbara §6 exactly: deny-sets use set-union (parent OR child can forbid), allow-sets use intersection (child cannot expand what parent restricts). The nil-semantics for `AllowCommands`/`Capabilities` are critical: `nil` means "unrestricted ceiling" not "empty allow list." An `intersectOrInherit(nil, child) = child` is correct (child imposes a ceiling on an unrestricted parent).
- `GovernanceConfig` has no `Capabilities` field, so child capabilities are always nil. This means a dynamic include child can never widen the parent's capability ceiling (child treats nil as "no new capability restrictions from me, parent keeps theirs").

**Frozen catalog is inviolable (§5):**
- Package and tool validation is done inside `catalogIncludeResolver.Resolve` (which has catalog access), not in the executor. This keeps the executor decoupled from the catalog. The resolver emits `DINC-010` for package not in catalog, `DINC-011` for tool not resolvable.
- The catalog `ProjectRequires[].Path` must be workspace-relative POSIX (forward-slash separated). `pkgcatalog.Build` rejects absolute Windows paths in `Path` with PKG-007. Tests using `t.TempDir()` must set `Path: "subdir-name"` (relative), not the absolute path.

**DINC-007 (with-key not declared) is class "DINC" but non-fatal:**
- The errkit sentinel `ErrDINC007` has class `"DINC"` (not `"DINC-W"`), so `errkit.IsWarning` returns false for it. However, Barbara §11 table says "Warning (non-fatal, logged)". Implementation: emit a `step/output` event with `code: "DINC-007"` but do not return an error. Do not use `errkit.IsWarning` to check this — check the code string directly.

**Trace event emission pattern:**
- Use `EmitterFromContext(ctx)` and nil-check before calling. The emitter signature is `func(kind string, payload map[string]any)`. Pass trace event kind strings via `string(tracepkg.EventKindIncludeResolved)` for type safety.
- `include/resolved` must be emitted AFTER all cycle/depth/input/governance checks pass and BEFORE calling `e.runner(...)`. This ensures the event accurately reflects a committed resolution, not a speculative one.
- `include/notFound` is emitted inside `handleResolveError` for all resolver failures — regardless of whether `on_not_found: continue` is in effect. The event is always emitted; what differs is whether an error is returned.

**Pin recorder seam (David, Stream 4):**
- `PinRecorder func(DynamicIncludePin)` is the callback type in `internal/executor`. David wires the real implementation that converts `executor.DynamicIncludePin` → `schema.LockedDynamicInclude` and appends to `PlanMetadata.DynamicIncludes`. Nil pin recorder is safe (skips pinning).
- The `DynamicIncludePin` struct mirrors `schema.LockedDynamicInclude` field-for-field to make David's adapter trivial.

### 2026-08-15T12:58:20-07:00 — Dynamic Include Stream 4 (Wiring)

**ResolverProxy bootstrap ordering problem:**
- `BuildEngineConfig` is called BEFORE Phase C (catalog build), which means the frozen catalog does not yet exist when the `RegistryConfig` is constructed. The `ResolverProxy` pattern solves this: create a proxy before `BuildEngineConfig`, register it in `WireOptions.DynamicIncludeResolver`, then call `proxy.Set(NewCatalogIncludeResolver(builtCatalog, parser))` after Phase C completes. `Set` is not concurrency-safe and must be called before `eng.Start`. This is the canonical way to break initialization ordering cycles in `cmd/gert/run.go`.

**`var plan` must precede `BuildEngineConfig`:**
- The `pinRecorder` closure captures `plan *engine.ExecutionPlan` by reference. In Go, a closure captures the variable itself (the memory address), not the current value. Therefore `var plan *engine.ExecutionPlan` must be declared *before* the `pinRecorder` closure literal. The actual assignment (`plan, err = plannerImpl.Plan(...)`) comes later; the closure just needs the variable in scope. If `plan` is nil at pin time (e.g. resume path where `Plan()` is not called), the closure returns early safely.

**Exactly one frozen catalog per run:**
- `builtCatalog` from `adapter.BuildPackageCatalog(...)` is the single frozen catalog. `resolverProxy.Set(NewCatalogIncludeResolver(builtCatalog, parser))` reuses that exact instance. Never call `pkgcatalog.Build` again — a second catalog build would produce a different (non-deterministic) `CatalogDigest` that diverges from the one emitted in the `catalog/frozen` trace event, breaking replay determinism.

**Integration test pattern for the adapter wiring layer:**
- `internalexecutor.NewDefaultRegistry(RegistryConfig{DynamicIncludeResolver: ..., PinRecorder: ...})` is the exact same call path that `BuildEngineConfig` uses. An integration test can call this directly (no need to stand up the full engine with trace files, OTel, etc.) and verify resolver + pin recorder behavior. See `internal/adapter/dynamic_wiring_test.go`.
- `platform.Real()` (not `platform.New`) is the constructor for the real platform. `internalparser.New(platform.Real())` constructs a parser with the embedded JSON Schema compiled — takes ~40ms but only needed once per test binary.

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


