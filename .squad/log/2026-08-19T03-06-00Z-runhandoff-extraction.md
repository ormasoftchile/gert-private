# Session Log: Runhandoff Extraction — Security-Critical Redaction Fix

**Date:** 2026-08-19T03:06:00Z  
**Duration:** Phase 1B Item 0 (concurrent with Phase 2 work)  
**Outcome:** SUCCESS

## Problem Statement

`runAuthenticated()` in `extension.ts` collected operator inputs (potentially containing secrets like ICM incident IDs), then passed them through a chat query string to `buildRunChatQuery()`. The original redaction tests targeted only the `buildRunChatQuery()` function — which structurally cannot receive the sensitive data because its signature is `(nonce, runbookPath) → string`. This is the **fourth occurrence** of the vacuous-mutation defect class: testing a pure helper whose signature proves it never sees the data, then claiming redaction is guaranteed.

**Live mutation proof (Cristiano):** Appending `collectedInputs` as `k=v` pairs to the query string at the production call site in `runAuthenticated()` passed 208/208 tests without a single failure. The test suite was structurally blind to the leak.

## Solution

Extract steps 3–5 of `runAuthenticated()` into `src/runHandoff.ts`:

1. **Pure module** — no `vscode` imports, testable with `node --test`
2. **Dependency injection** — collaborators passed as `RunHandoffDeps` interface
3. **Real collaborators** — `stashPendingRun`, `buildRunChatQuery`, `promptForInput` are REAL production implementations, not stubs
4. **Instrumented output** — `RunHandoffResult` exposes collected inputs so tests can assert on them

## Key Learning

**The real risk lives at the production call site, not in the helper.** By extracting call-site logic into a testable module with injected REAL collaborators, HANDOFF-4 now verifies that `collectedInputs` actually reaches the store. A future mutation that strips the store call is immediately caught.

This is the **remedy for the systemic vacuity defect class**: extract call-site logic, inject real collaborators, and write non-vacuity controls that prove the data flow is observable.

## Results

- **Test count:** 208 → 215 (HANDOFF-1..7)
- **Exit code:** 0 (all passing)
- **Mutation #1 (Cristiano's):** HANDOFF-3 fails as expected
- **Mutation #2:** HANDOFF-4 fails as expected
- **Mutation controls verified:** Yes, both mutations are detected

## Files Changed

- `src/runHandoff.ts` (new)
- `src/extension.ts` (refactored)
- `test/runHandoff.test.js` (new)

## Scope Boundary

`git grep -in "tsg" -- src test` returns nothing. The scope remains clean.
