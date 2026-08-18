# Session Log: VS Code Live Blockers — Hermetic Regression Tests

**Session:** vscode-live-blockers  
**Timestamp:** 2026-08-18T22:23:36Z  
**Effort:** Defect remediation + regression prevention for multi-root workspace configuration scoping

## Summary

Three agents (ken-7, ken-8, ken-9) delivered fixes for VS Code live blockers discovered in multi-root workspace execution:

1. **Config Scoping Defect** — `binaryPath` and `packageMap` read from wrong scope, causing wrong tool invocation
2. **Registry Parse Drift** — TypeScript rebuild did not mirror all three parse axes from Go core
3. **Scoping Seam** — Extracted `makeScopedGetter` pure function to make config scoping testable
4. **Static Guard** — Added `repoBoundary/rule7` to prevent future unscoped config reads

## Artifacts Produced

- **Commits:** `1cd7542` (ken-7, defects 1+2), `fadc2fa` (ken-8, seam), `08de611` (ken-9, guard)
- **Fast-forward:** gert-vscode main from `f1e5c51` to `424ce26` (ken/hermetic-regression-tests branch)
- **Test Coverage:** 
  - Defect2 drift guard: 6 shape-matrix entries, 1 mutation control
  - Ken-8 spy test: callback verification, 1 mutation control
  - Ken-9 static guard: 7 real sites + 4 independent mutations
- **Decisions Merged:** 14 inbox entries covering Phase 2 architectural ruling, contract parity, output enforcement, TSG binding resolution

## Coordinator Verdicts

- **Ken-7, defect2:** ACCEPTED (independent verification confirmed parity)
- **Ken-7, defect1:** REJECTED (mutation proof was vacuous)
- **Ken-8:** Identified residual untested adapter lambda (resolved by ken-9 work)
- **Ken-9:** ACCEPTED (independent verification confirmed)

## Key Verified Facts

- gert-vscode `08de611`: 0 untracked, npm ci 0, compile 0, 130 tests / 130 pass / 0 fail / 0 skipped
- Rule7 inspects 6 real getConfiguration call sites (7 raw matches, 1 is JSDoc)
- 4 mutations independently re-run, all produced real failures
- Gert-private `18093a5`: 0 untracked, build 0, test 0, 63 ok / 23 no-test / 0 FAIL
