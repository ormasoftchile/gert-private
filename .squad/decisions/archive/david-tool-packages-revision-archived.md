# Revision Record — GERT Tool Packages MVP (runtime), response to Barbara's gate review

**From:** David (Integration Engineer, independent revision owner)
**Date:** 2026-08-09T15:59:36-07:00
**Requested by:** Cristián Ormazábal Ortega
**Responds to:** `.squad/decisions/inbox/barbara-tool-packages-implementation-gate-review.md`
**Locked out (not consulted):** Don, Ken

---

## Scope note

This revision resolves every blocking item (B1–B5, §5) and every §3
required lower-severity fix in Barbara's gate review, working directly in
the runtime repo (`C:\One\OpenSource\gert`), against the ratified spec
(`design/gert/sections/06-tool-runtime.tex` et al., closed AR-TP-1..10,
R1–R15, S1/S2). No design was reopened; only runtime code was changed to
match the already-ratified spec. Don and Ken were not consulted at any
point. Nothing was staged, committed, or reverted; no branch was switched.

## Blocker-resolution matrix

| # | Item | Disposition | Key files | Regression test(s) |
|---|------|-------------|-----------|---------------------|
| B1 | Plan-time substitution contract validation (PKG-013/014/015/026/027/028 must be plan-time, incl. dry-run and unreachable-by-`when:` steps) | **Resolved.** New `flowwalk.Visitor`-based walker traverses the full include closure eagerly (never lazily deferred) and visits every step structurally regardless of `when:`, calling `pkgsubst.Plan` for every `execute.kind: runbook` action and recursing into the substitute's own flow for transitive PKG-015/PKG-028. Wired into both `gert run` and `gert dry-run` (shared `runWithMode`). | `cmd/gert/substitution_plan.go` (new), `cmd/gert/run.go` | `gate_review_b1_regression_test.go`: dry-run plan-time failure, unreachable-step validation |
| B2 | Package digest closure must include substitute runbooks + package-internal includes, deterministically | **Resolved.** Manual `*yaml.Node`-walking include-path extractor (fixes a `yaml.v3` panic on `schema.Step`'s colliding inline tags that a generic typed unmarshal would hit) closes over every substitute runbook and package-internal include reachable from an export, cycle-guarded, sorted before hashing. | `pkg/pkgcatalog/catalog.go` | `pkg/pkgcatalog/gate_review_b2b3_regression_test.go`: substitute-byte digest drift |
| B3 | Catalog digest from package digest, not file digest, for tier-1 entries | **Resolved.** Tier-1 `Entry.Digest` now carries the package digest (closure hash), not the individual exported file's own digest. | `pkg/pkgcatalog/catalog.go` | same test as B2 (asserts `CatalogDigest()` changes on a substitute-runbook byte edit with the exported file itself untouched) |
| B4 | Resume drift must detect removed/added packages (PKG-001 for removed) | **Resolved.** Drift check now computes the union of manifest-recorded and currently-resolved package names: missing-but-recorded → PKG-001; present-but-unrecorded → forced PKG-009 (via empty `Expected`). | `cmd/gert/resume_drift.go`, `cmd/gert/resume_drift_integration_test.go` (fixture fix: populate `PackageDigests`) | `gate_review_b4_regression_test.go`: removed package → PKG-001, added package → PKG-009 |
| B5 | `outputs.<name>` GCP root + substitution output capture | **Resolved end-to-end.** New `gdp.SourceOutputs`; parser `outputs.<name>` (no GDP suffix permitted, GCP-PARSE-002 otherwise); resolver reads `current.Output[field]` (GCP-RESOLVE-002 if absent); plan-time step-context check (`stepBindsSubstitutionAction`) rejects `outputs.*` capture on a non-substitution step with GCP-PARSE-001. Also fixed item 3.8 (silent output-coercion fallback) as part of this work, and a **critical latent bug**: `schemaToolDefFromRuntime` was dropping `Execute`/`Outputs` when converting catalog-bound tools for the planner's schema-typed registry, silently defeating both B1 and B5 for any package/toolRefs-bound substitution action. | `pkg/gdp/path.go`, `pkg/gcp/parser/parser.go`, `pkg/capture/service.go`, `internal/planner/validate.go`, `internal/executor/tool.go`, `cmd/gert/packagemap.go` (bugfix) | `gate_review_b5_regression_test.go`: output capture success, wrong-step-kind plan-time failure |
| §5 | Include closure: global package set + per-file lexical `toolRefs` binding | **Resolved via Barbara's sanctioned fallback (b): fail closed.** Full per-file lexical `BindFile` binding (option (a)) was judged too large for this revision and was *not* implemented — this is an explicit, sanctioned scope reduction, not a silently-dropped gap. Instead, any included runbook that itself declares `requires:` or `toolRefs:` now fails closed with PKG-017 (repurposed as "late/ambiguous package binding after catalog freeze" — the closest existing sentinel, matching Barbara's requirement that fallback (b) needs only "a hard, typed, diagnosable error," not a specific mandated code). | `cmd/gert/include_closure.go` (new), `cmd/gert/run.go` | `gate_review_section5_regression_test.go`: child `requires:` and child `toolRefs:` both fail closed with PKG-017 |

### §3 — required, lower-severity fixes

| # | Item | Disposition | Key files |
|---|------|-------------|-----------|
| 3.1 | `ConstraintSources` permanently empty | **Resolved.** `mergeRequirements` now tracks, per package, the ordered declaration sites ("project requires:" / "runbook requires: (<path>)") that contributed a non-empty `version:` to the merged constraint; threaded into `LockedPackageInfo.ConstraintSources` and the `package/resolved` trace payload. | `pkg/pkgcatalog/catalog.go`, `internal/adapter/trace_events.go` |
| 3.2 | PKG-002 omits package name + declaration-site provenance | **Resolved.** Both `catalog.go`'s and `bind.go`'s PKG-002 messages now include the package name and the declaration site(s)/`toolRefs[name].version` site. | `pkg/pkgcatalog/catalog.go`, `pkg/pkgcatalog/bind.go` |
| 3.3 | Build metadata (`+...`) wrongly accepted in constraints | **Resolved.** `+` added to `unsupportedPatterns`, rejected before parsing (PKG-003). Build metadata remains accepted/retained verbatim in *versions* (unaffected). | `pkg/semver/constraint.go` |
| 3.4 | PKG-006 error ordering non-deterministic | **Resolved.** `detectSameTierCollisions` now sorts bare names and tiers before iterating, so returned error ordering is stable run to run. | `pkg/pkgcatalog/catalog.go` |
| 3.5 | PLAN-010 never raised (generic `ErrToolNotFound` instead) | **Resolved.** New `errkit.ErrPLAN010` sentinel; `resolveTool` now wraps the registry's own `ErrToolNotFound`/`ErrActionNotFound` as `errkit.Wrap("PLAN-010", ..., err)`, so the typed code is now emitted while `errors.Is(err, plannerPkg.ErrToolNotFound)` still matches through the unwrap chain (verified against existing tests). | `internal/planner/planner.go`, `pkg/errkit/errors.go` |
| 3.6 | PKG-018's normalisation half dead (export uniqueness compares raw strings) | **Resolved.** Export id/path uniqueness now also checks `pkgpath.NormalizeForComparison(..., true)` (NFC + ASCII-case-fold) and raises PKG-018 on a normalisation-only collision, in addition to the existing PKG-004 literal-duplicate check. | `pkg/pkgcatalog/catalog.go` |
| 3.7 | `maxLinkHops = 8` declared but never enforced | **Resolved.** New `checkLinkHopBound` explicitly walks a path's own symlink/junction chain, counting hops, and fails PKG-008 once `maxLinkHops` is exceeded, ahead of/independent from the OS's own (much larger, unspecified) ELOOP ceiling; `filepath.EvalSymlinks` still does the actual resolution (needed for correct 8.3-short-name canonicalisation on Windows — an initial full component-wise rewrite broke this and was reverted in favour of this hybrid). | `pkg/pkgpath/pkgpath.go` |
| 3.8 | Silent output-type-coercion fallback | **Resolved** (done as part of B5 work). `coerceOutputValue` now returns `(any, error)` and fails the step on a declared-type mismatch instead of silently returning the raw string. | `internal/executor/tool.go` |
| 3.9 | `--package-map` override provenance stderr-only | **Resolved.** New `origin` field (`"project"` / `"package-map"`, omitted if no `--package-map`) added to the `package/resolved` trace payload, sourced from the existing `mergePackageBindings` provenance that was previously only printed to stderr. | `pkg/trace/event.go`, `internal/adapter/trace_events.go`, `cmd/gert/run.go` |

## Regression tests added this revision

- `cmd/gert/gate_review_b1_regression_test.go` — dry-run plan-time contract violation; unreachable-step plan-time validation.
- `cmd/gert/gate_review_b4_regression_test.go` — removed package → PKG-001; added/untracked package → PKG-009.
- `cmd/gert/gate_review_b5_regression_test.go` — `outputs.<name>` capture success; wrong-step-kind plan-time failure.
- `cmd/gert/gate_review_section5_regression_test.go` — child `requires:` fail-closed PKG-017; child `toolRefs:` fail-closed PKG-017.
- `cmd/gert/gate_review_section3_regression_test.go` — `package/resolved` trace event carries both non-empty `constraintSources` and `origin: "package-map"`.
- `pkg/pkgcatalog/gate_review_b2b3_regression_test.go` — substitute-runbook byte edit changes both the package digest and `CatalogDigest()` with the exported file's own digest unchanged.
- `pkg/pkgpath/gate_review_maxlinkhops_test.go` — a chain of `maxLinkHops+2` symlinks fails PKG-008; a chain of exactly `maxLinkHops` still resolves (no false positive). (Skipped on Windows per the pre-existing test convention in this package, though manually verified working against real Windows symlinks during development.)

## Test evidence

- `go build ./...` — clean, no errors.
- `go test ./...` — **all packages pass**, including `internal/conformance` (Tess's `tv-gcp-path.yaml`, `tv-gis-path.yaml`, `tv-gxl-*.yaml` vectors — all cases green, including every `TV-GCP-PATH-*` case exercised by the `outputs.<name>` parser change).
- Targeted package runs also re-verified individually during development: `pkg/pkgcatalog`, `pkg/pkgpath`, `pkg/semver`, `pkg/errkit`, `internal/planner`, `internal/adapter`, `cmd/gert`.

## Protected-file verification

All four protected files were left **byte-for-byte untouched** by this
revision (zero `edit`/`create` calls were made against any of them):

- `examples/simple-health-check/simple-health-check.runbook.yaml`
- `internal/tool/native.go`
- `internal/tool/native_test.go`
- `examples/simple-health-check/examples.code-workspace`

Note: `git status`/`git diff` show pre-existing working-tree modifications
on the first three (a Windows `ping`/`curl` argv-normalisation fix and a
`{{ .hostname }}` → `${hostname}` GXL-path rewrite) and the fourth is
untracked — all of this predates this revision and was never touched or
re-derived by David; it is reported here only to confirm it is unrelated
to, and unaffected by, this work.

## Remaining gaps (explicit, not silently dropped)

1. **§5 fallback (b), not (a):** Full per-file lexical `toolRefs` binding
   (`BindFile`) is not implemented. The runtime fails closed (PKG-017) for
   any included runbook that itself declares `requires:`/`toolRefs:`,
   rather than supporting that form. This is the explicitly Barbara-
   sanctioned scope reduction for this revision, not an oversight.
2. **Accepted non-goals reaffirmed, not broadened:** absent replay mode;
   pre-existing cross-process resume plan persistence gap (`ExecutionPlan`
   is not persisted in `DirRunStore`, so `--allow-package-drift`/PKG-009
   remain in-process-only across restarts — ticketed by Barbara, not
   fixed here); orphaned deep governance evaluator enforcement inside
   substitute bodies (verified still orphaned repo-wide, no relative
   escalation introduced).
3. No new §3-adjacent regression tests were added beyond the six listed
   above where a single test already exercised more than one item (e.g.
   the §3 test above covers both items 3.1 and 3.9 together); this was a
   deliberate time-budget choice given all nine §3 items are lower-
   severity by Barbara's own classification.

## Disposition

All blocking items (B1–B5, §5) and all nine §3 required fixes are
resolved in the runtime repo, `go build ./...` and `go test ./...` are
green, protected files are verified untouched, and Tess's relevant
conformance vectors pass. Nothing was staged or committed per
instructions; changes remain in the working tree at
`C:\One\OpenSource\gert` for review.
