# Session Log: Input Collection Fix

**Date:** 2026-04-07T00:48:41Z  
**Sprint:** Input Collection Bug Fixes  
**Status:** ✅ Complete

## Summary

Fixed input collection functionality by addressing three interconnected issues:
1. WebSocket disconnect loop preventing server communication
2. Test selector mismatch breaking e2e test validation
3. Documentation of fixes for team knowledge

## Issues Resolved

### 1. WebSocket Disconnect Loop (Killua)
- **Root Cause:** Playwright click() triggered tab button twice, causing repeated dispose/recreate cycles
- **Fix:** Added currentTab guard in web/src/main.ts to prevent redundant tab loads
- **Commit:** f309294
- **Status:** ✅ Resolved

### 2. Test Selector Issue (Illumi)
- **Root Cause:** Selector `:has-text("Run")` was non-deterministic and matched wrong button
- **Fix:** Changed to `[data-testid="run-button"]` for specific, reliable targeting
- **Commit:** 8e0ff0e
- **Status:** ✅ Resolved

### 3. Visual Verification (Knov)
- **Task:** Confirm input form appears after fixes
- **Result:** Input form now appears before execution as expected
- **Status:** ✅ Verified

## Test Results

All 13 tests now passing:
- input-form visible
- input-field-server_name visible  
- start-run-button visible
- Input variables properly collected
- Execution proceeds with provided values

## Technical Details

**Files Modified:**
- web/src/main.ts (currentTab guard)
- web/vite.config.ts (proxy target)
- web/playwright.config.ts (HMR config)
- Test files (selector fixes)

**Commits:**
- f309294: WebSocket fix
- 8e0ff0e: Test selector fix

## Impact

Users can now:
- See input forms when runbooks require prompt inputs
- Provide values for prompt inputs before execution
- Execute runbooks with user-provided data
- Run automated tests with stable selectors

## Lessons

1. Guard against re-entrancy in UI event handlers
2. Use data-testid attributes for reliable test targeting
3. Monitor WebSocket connection stability in browser apps
4. Verify schema vs. code consistency (from earlier fixes)
