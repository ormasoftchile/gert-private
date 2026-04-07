# Orchestration Log: killua-json-fix

**Timestamp:** 2026-04-06T10:09:48Z  
**Agent:** Killua (Backend Engineer, Go)  
**Task:** Fix double-sendResult bug in executeTreeStep for invoke steps  
**Status:** ✅ Complete

## Dispatch

Killua was tasked to fix the JSON crash in `edge-case-branch.runbook.yaml` identified during Hisoka's full sweep.

## Root Cause Analysis

- Runbook has manual step with branches → each branch points to invoke step
- `handleTreeNext` auto-advance loop evaluates branch and calls `executeTreeStep`
- `executeTreeStep` sends result after branch evaluation
- Loop continues, pops invoke step, enters invoke context
- Child step sends second result
- Two JSON objects written to output stream → parsing error

## Work

1. Added `suppressResult` variadic parameter to `executeTreeStep`
2. When called from auto-advance loop, pass `suppressResult=true`
3. Guard updated: skip sendResult if `len(invokeStack) > 0 || suppressResult`
4. Tested with 26 Playwright tests: all passing

## Files Changed

- `ext/serve/pkg/serve/serve.go` — 8 line change to executeTreeStep signature and callers

## Commits

- 4f8f192 — fix: double-sendResult bug in executeTreeStep for invoke steps

## Invariant Established

> `executeTreeStep` must not call `sendResult` when the caller will continue processing. The `suppressResult` parameter makes this explicit at the call site.

## Outcomes

- JSON double-send bug fixed
- edge-case-branch now completes successfully
- All 26 Playwright tests passing
- Ready for final verification sweep
