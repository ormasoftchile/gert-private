# Decision: Iterate Visual Redesign — Test Contract

**Date:** 2026-06-XX  
**By:** Knov (Playwright & E2E Testing Specialist)  
**Task:** Write Playwright tests for iterate block visual redesign

---

## Decision: CSS Class Names as the Test Contract

Tests assert on `.wf-iterate-container`, `.wf-fork-diamond`, `.wf-join-diamond` in the SVG DOM. These are the classes Kurapika must add to the graph renderer. The tests **intentionally fail until those classes are shipped** — they are the acceptance criteria, not post-hoc coverage.

**Why this matters:** Any implementation that passes these tests is correct from the testing perspective. If Kurapika uses different class names, the tests should be updated at the same time.

---

## Decision: Secondary Runtime Attributes Use Soft Assertions

`data-iterate-pass` and `data-iterate-lane` (runtime expansion attributes) are checked with `console.warn`, not `expect()`. Reason: runtime expansion is a separate milestone and blocking the static tests on unimplemented runtime features would cause false failures in CI before the feature lands.

Once Kurapika ships runtime expansion, the soft checks should be promoted to hard `expect()` assertions.

---

## Decision: No Scenario Injection in Runtime Tests

The web app does not expose `scenarioDir` to the web UI (the `exec/start` call in `runbookRunner.ts` hardcodes `mode: 'real'`). Runtime tests therefore run against the real gert engine with real tool invocations.

**Recommendation:** Kurapika or Killua should add scenario/replay support to the web runner to enable deterministic runtime testing. Until then, runtime tests depend on tool availability and may flake in CI environments without the `curl` gert tool registered.

**Tracking:** Runtime tests have `test.setTimeout(120000)` and graceful fallbacks for failed invocations (iterate steps have `continue_on_fail: true`).

---

## Files Delivered

| File | Tests | Purpose |
|------|-------|---------|
| `web/tests/specs/iterate-sequential.spec.ts` | 4 | Sequential iterate: container rect, no back-edge, header text, runtime passes |
| `web/tests/specs/iterate-parallel.spec.ts` | 5 | Parallel iterate: fork diamond, join diamond, symmetric count, no wrong class, runtime columns |

**Screenshots produced:**
- `web/tests/screenshots/iterate-sequential-static.png`
- `web/tests/screenshots/iterate-sequential-runtime.png`
- `web/tests/screenshots/iterate-parallel-static.png`
- `web/tests/screenshots/iterate-parallel-runtime.png`
