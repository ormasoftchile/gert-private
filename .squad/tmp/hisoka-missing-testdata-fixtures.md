# Decision: Missing testdata fixtures block 28 Go tests

**By:** Hisoka (QA Reviewer)
**Date:** 2026-03-18
**Status:** Flagged

## Context

Go test run across `pkg/...` reveals 28 test failures in 5 packages — all caused by missing `testdata/` directory. The tests expect fixtures at `../../testdata/` relative to test files (e.g. `testdata/valid/minimal.yaml`, `testdata/tools/kubectl.tool.yaml`, `testdata/scenarios/minimal-scenario.yaml`, `testdata/tools/mock-jsonrpc-server.go`).

Additionally, 11 iterate tests in `pkg/engine` fail with `unknown step type: ""` — the test fixtures create steps without a `type` field, which the engine now rejects.

## Impact

- CI pipeline (`go test ./pkg/...`) will report failures on every run
- Cannot distinguish new regressions from known fixture gaps
- Green-build trust eroded before CI even ships

## Recommendation

1. **Killua:** Either add the `testdata/` directory with required fixtures, or skip tests that depend on missing fixtures with `t.Skip("requires testdata fixtures")`
2. **Engine iterate tests:** Update test step fixtures to include a valid `type` field (e.g. `type: "tool"`)
3. **CI pipeline:** Consider adding a `continue-on-error: true` annotation to Go tests until fixtures land, or split into "core tests" (passing) and "integration tests" (fixture-dependent)
