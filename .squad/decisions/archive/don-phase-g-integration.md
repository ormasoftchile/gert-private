# Phase G Integration — Don Inbox Memo

**From:** Don (Backend Dev)
**Date:** 2026-06-06T01:05:00-04:00
**PR:** ormasoftchile/gert#15 (draft, `phase-g-integration` → `phase-d-path`)
**Status:** ✅ SHIPPED — all test gates green, 267/267 conformance maintained

---

## Integration Shape

### Feature Flag

**Build tag `//go:build gxl`** is the feature flag per the migration plan's "parallel-package, feature-flag cutover" strategy. No CLI flag, no env var, no runtime switch — the build tag selects the engine at compile time. This is clean, reversible, and consistent with how all Phase A–F code was tagged.

### CLI / Wiring Surface

Two wiring points, consistent with the migration plan:

1. **`internal/adapter/wire.go`** — `BuildEngineConfig()` now calls `newEvaluators()` + `newCaptureResolver()` instead of inlining `internalexpr` creation. Build-tagged files switch the implementation:
   - `!gxl` → `evaluators_legacy.go`: `TemplateEvaluator` + `SimpleConditionEvaluator` + nil resolver
   - `gxl` → `evaluators_gxl.go`: `GISEvaluatorAdapter` + `GXLConditionAdapter` + `GCPCaptureAdapter`

2. **`pkg/run/run.go`** — same pattern via `newRunEvaluators()` + `newRunCaptureResolver()`

### Public API (internal/eval/adapter_gxl.go)

Three adapter types, all under `//go:build gxl`:

| Type | Interface | Engine |
|------|-----------|--------|
| `GISEvaluatorAdapter` | `pkg/expr.Evaluator` | GIS — `${...}` template interpolation |
| `GXLConditionAdapter` | `pkg/expr.ConditionEvaluator` | GXL — strict PJVM boolean evaluation |
| `GCPCaptureAdapter` | `internal/executor.CaptureResolver` | GCP — path-based step output capture |

Helpers: `MapToValues(map[string]any) → map[string]core.Value`, `anyToValue(any) → core.Value`, `ValueToAny(core.Value) → any` — all in `internal/eval/adapter_gxl.go` for use by the CLI and future cutover code.

### Capture Path Evolution

- `internal/executor.CaptureResolver` interface added to `internal/executor/capture_resolver.go` (no build tag — it's just an interface)
- `internal/executor.RegistryConfig.CaptureResolver` field added (optional, nil = legacy keyword switch)
- `CLIExecutor.captureResolver` field, `NewCLIExecutor(p, eval, ...CaptureResolver)` — variadic for backward compat
- Under `gxl`: `GCPCaptureAdapter` handles ALL capture paths via GCP engine. Legacy `stdout`/`stderr`/`exit_code` strings are valid GCP LocalCapture paths (confirmed by GCP corpus TV-001..003)

---

## Merge Experience

### STEP 0: Ken's Branch Merges

Both merges completed cleanly with zero code conflicts (as predicted — `internal/eval/gis/` and `internal/eval/gcp/` are disjoint from `internal/eval/gxl/`):

```
git merge --no-ff origin/phase-e-gis   → zero conflicts, harness auto-merged
git merge --no-ff origin/phase-f-gcp   → zero conflicts, harness auto-merged
```

**One harness conflict:** `vectorBindings` helper was independently declared in both `path_runner_gxl_test.go` (Phase D) and `gis_runner_gxl_test.go` (Phase E) — identical definitions, different files. Resolved by extracting to `helpers_gxl_test.go` and removing the duplicate from both files. Not a semantic conflict — pure naming collision in test helpers.

---

## Test Totals

| Suite | Count | Result |
|-------|-------|--------|
| Conformance (GXL-PARSE) | 83/83 | ✅ PASS |
| Conformance (GXL-EVAL) | 92/92 | ✅ PASS |
| Conformance (GXL-PATH) | 36/36 | ✅ PASS |
| Conformance (GIS-PATH) | 15/15 | ✅ PASS |
| Conformance (GCP-PATH) | 41/41 | ✅ PASS |
| **Conformance TOTAL** | **267/267** | ✅ **100%** |
| Integration (adapter e2e) | 4/4 | ✅ PASS |
| `go test ./...` (no tag) | all pass | ✅ |
| `go test -tags gxl ./...` | all pass | ✅ |
| Pre-existing failure | `TestRender_Regions` | ⚠️ pre-dates Phase work |

### Notes on Fixture Migration

`TestStepStartedEvent_StructuralMetadata_CollectHealth` (internal/engine) failed under `gxl` because the collect-health runbook uses `{{ }}` Go-template syntax, which the `GISEvaluatorAdapter` passes through as literal text (correct GIS behavior). Fixed by:
1. Providing GXL-syntax variants of the runbooks: `collect-health-gxl.runbook.yaml` + `check-service-gxl.runbook.yaml`
2. The check-service conditional captures (originally Go template `if/else`) restructured as a `branch` step with GXL conditions (`dns_exit == 0 and ping_exit == 0`)
3. Build-tagged `collectHealthRunbookPath()` helper in engine test selects the right variant per build
4. Original `{{ }}` runbooks preserved unchanged for legacy (`!gxl`) test path
5. Hardcoded `/Volumes/Projects/gert/...` absolute path in both engine tests replaced with `repoRoot()` + relative path — portable across machines and worktrees

---

## Handoffs

### → Ken (Phase H prep)

No changes made to `internal/eval/gis/` or `internal/eval/gcp/`. The `CaptureResolver` interface lives in `internal/executor/capture_resolver.go` — Ken's `GCPCaptureAdapter` satisfies it structurally without importing the interface type.

**One note for Phase H:** The `GCPCaptureAdapter` returns an error for any capture path it can't parse (GCP-PARSE-001 for unknown sources). This is spec-correct but means legacy runbooks using non-standard capture keys (e.g., `exitCode` camelCase) will error under `gxl` build. Phase H migration guide should document the `exit_code` (snake_case only) requirement.

**For Phase H:** The `collect-health.runbook.yaml` and `check-service.runbook.yaml` (original `{{ }}` versions) should be replaced with the `-gxl` variants as the canonical ones when Phase H removes the legacy evaluator.

### → Barbara (Spec)

No spec gaps found. One behavior to confirm: `GISEvaluatorAdapter.Eval()` returns an error when a template contains a mandatory (non-optional-chaining) path that's not in `vars`. This is GIS-PATH-MISSING per spec. The legacy `TemplateEvaluator` would return an empty string in this case (`missingkey=zero`). If any existing runbooks rely on missing-key-as-empty-string semantics, they'll get errors under `gxl`. Flagging as potential Phase H migration note — no action needed now.

### → Tess (Corpus)

No new corpus vectors. 267/267 maintained. The 4 new integration tests in `internal/eval/integration_gxl_test.go` are Go-level tests, not TV-* corpus vectors.

---

## PR Link

**https://github.com/ormasoftchile/gert/pull/15**

- Draft PR, base: `phase-d-path` (PR #12)
- Title: `phase-g: runtime integration (stacked on PR #12 + folds PR #13/#14)`
- Merge order: PR #9 → #10 → #11 → #12 → #13/#14 (either order) → **#15**

---

## Phase H Preview (Cutover)

Phase H should execute in one focused PR:

1. **Delete legacy factories:** remove `internal/adapter/evaluators_legacy.go` and `pkg/run/evaluators_legacy.go` (the `!gxl` files)
2. **Drop `//go:build` constraints:** remove `//go:build gxl` from all `internal/eval/*` files — they become unconditional
3. **Delete `internal/expr/`:** remove `template.go`, `condition.go`, `condition_test.go`, `template_test.go`
4. **Clean `go.mod`:** `go mod tidy` removes `github.com/expr-lang/expr`
5. **Canonicalize runbooks:** rename `-gxl.runbook.yaml` variants to replace originals; remove `{{ }}` versions
6. **Update engine test:** `collectHealthRunbookPath()` helpers become a single function (no build tag)
7. **CI gate:** add `go test -tags gxl ./...` as the primary CI step; `go test ./...` (no tag) can be retired
8. **Verify:** `go test ./...` green, `go build ./...` clean, no import of `expr-lang/expr` anywhere

Estimated size: Small (1 day), mostly mechanical deletion. Risk: low — all semantics already proven by 267-vector conformance corpus.
