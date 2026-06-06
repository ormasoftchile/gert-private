# Don — Phase H Cutover

PR: https://github.com/ormasoftchile/gert/pull/16

## Summary

Phase H is complete on `phase-h-cutover`: GXL/GIS/GCP is unconditional and the legacy `text/template` + `expr-lang` runtime has been removed.

## Files deleted

Deleted 9 files (~670 lines):

- `examples/collect-health/check-service-gxl.runbook.yaml` (~80 lines) — promoted to canonical name
- `examples/collect-health/collect-health-gxl.runbook.yaml` (~104 lines) — promoted to canonical name
- `internal/adapter/evaluators_legacy.go` (~22 lines)
- `internal/engine/runbook_paths_legacy_test.go` (~9 lines)
- `internal/expr/condition.go` (~63 lines)
- `internal/expr/condition_test.go` (~262 lines)
- `internal/expr/template.go` (~63 lines)
- `internal/expr/template_test.go` (~45 lines)
- `pkg/run/evaluators_legacy.go` (~22 lines)

## Files modified

78 files changed total: 68 modified, 9 deleted, 1 added (`CHANGELOG.md`). Key modified areas:

- Removed `//go:build gxl` from GXL/GIS/GCP runtime and conformance files.
- Updated runtime factories to use GXL/GIS/GCP unconditionally.
- Updated executor tests to use `internal/eval` adapters after deleting `internal/expr`.
- Removed `github.com/expr-lang/expr` from `go.mod`/`go.sum` via `go mod tidy`.

## Runbooks migrated

12 runbook files touched:

- 2 collect-health GXL variants promoted to canonical filenames.
- 10 legacy runbooks mechanically migrated from `{{ .varname }}` to `${varname}`.

Examples:

- `{{ .service_name }} ({{ .incident_id }})` → `${service_name} (${incident_id})`
- `{{ join .results "\n" }}` → `${results}`
- `exitCode` capture path → `exit_code`
- Inline Go-template conditional capture in `collect-health-parallel/check-service.runbook.yaml` replaced with GXL `branch` steps using `str.contains(...)` / `not str.contains(...)`.

## CHANGELOG

Added `0.4.0-phase-h` entry documenting:

- GXL/GIS/GCP as the only runtime engine
- deleted legacy packages/files
- GIS mandatory-miss-as-error (Barbara Option A, commit `e819895`)
- GCP `exit_code` snake_case capture path
- GXL condition syntax migration notes

## Validation

- `go build ./...` passes.
- `go test ./...` confirms 267/267 conformance vectors and 4/4 integration tests green after cutover.
- Only acknowledged pre-existing failure remains: `pkg/preview/render/markdown TestRender_Regions` missing `/Volumes/Projects/gert-domain-dri/pkg/compiler/testdata/change-request.golden.yaml`.

## Deferred / handoffs

No cutover blockers deferred. Follow-up is the known markdown preview fixture-path issue, unrelated to Phase H runtime migration.
