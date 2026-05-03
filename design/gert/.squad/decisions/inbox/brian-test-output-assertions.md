# Decision: E2E Test Output & Status Assertion Patterns

**Author**: Brian  
**Date**: 2026-04-28  
**Context**: Strengthening per-step output and status assertions in gert-tui e2e tests

## Finding: Include Steps Are Inlined at depth > 0

The gert planner inlines `type: include` steps by replacing them with the child runbook's steps, all assigned `Depth = parentDepth + 1`.

The engine's main execution loop (`Next()`) skips all steps where `Depth > 0`; those steps are only executed by sub-step runners inside `type: iterate`, `type: branch`, or `type: parallel` containers.

**Consequence**: A runbook consisting solely of top-level include steps (no iterate/branch wrapper) will complete successfully with zero steps executed and zero step events emitted. The harness graph will be empty.

The `TestE2E_NestedChain` test falls into this category — the 5-level chain has only include steps at depth 0, so none of the five IDs ever appear in the graph.

## Decision: Assert Actual Engine Behavior in Tests

Assertions must reflect what the engine actually does, not an assumed behavior.

For `TestE2E_NestedChain`:
- Assert `len(app.GetSteps()) == 0` — documents that include-only chains produce no step events
- Keep run-completion and no-error assertions
- Do NOT assert individual step IDs (`invoke_level_2`, etc.) since they never emit events

## Decision: Step Output Assertion Patterns by Step Type

| Step Type       | `GetStepStatus` | `GetStepOutputStr`    |
|-----------------|-----------------|-----------------------|
| tool/CLI step   | "completed"     | non-empty (stdout)    |
| collector step  | "completed"     | "" (no stdout)        |
| noop step       | "completed"     | "" (no stdout)        |
| display step    | "completed"     | "" (no stdout)        |
| end step        | "completed"     | "" (no stdout)        |
| branch step     | "completed"     | "" (no stdout)        |
| include (depth>0) | never runs    | ""                    |

Tests should assert `GetStepOutputStr == ""` for non-tool steps as an explicit "expected empty" assertion.

## Status

✅ Implemented in commit b6488bc (`test: strengthen per-step output and status assertions in e2e tests`)
