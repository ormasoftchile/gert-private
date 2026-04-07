# Input Form Verification Test Report
**Tester:** Knov  
**Date:** April 6, 2026, 8:03 PM  
**Task:** Verify input collection form appears for runbooks with `from: prompt` inputs  
**Commit Tested:** b3ad8ac

## Test Setup
- Built gert: ✅ Success
- Started gert server (port 7778): ✅ Running
- Started Vite dev server (port 5173): ✅ Running  
- Test runbook: `examples/service-health-branching.runbook.yaml`
- Input definition in runbook:
  ```yaml
  inputs:
    server_name:
      from: prompt
      description: Server to check
  ```

## Test Execution

### Steps Performed:
1. ✅ Opened http://localhost:5173
2. ✅ Clicked "Runbook Runner" tab
3. ✅ Entered runbook path: `examples/service-health-branching.runbook.yaml`
4. ✅ Clicked "Run" button
5. ⏱️ Waited 3 seconds for form to appear
6. ❌ Input form did NOT appear

### Screenshots Captured:
- `input-verify-1-before-run.png` - Before clicking Run
- `input-verify-2-after-run-click.png` - After clicking Run (3s wait)
- `input-verify-debug.png` - Debug screenshot
- `input-verify-error.png` - Error state

### Observable Behavior:
- **Before Run:** Shows runbook path input and Run button
- **After Run:** Page remains unchanged, still showing empty state message  
  "👆 Enter a runbook path above and click Run to start execution"
- **Expected:** Should show input form with `data-testid="input-form"` containing a field for `server_name`
- **Actual:** No input form appeared, no UI change at all

## Root Cause Analysis

### Code Fix Verification:
The fix in commit b3ad8ac IS present in the source code:
```typescript
// Line 1301 in web/src/views/runbookRunner.ts
if ((def as InputDef).from === 'prompt') {  // ✅ Correct
  userInputs[name] = def as InputDef;
}
```

### Server Communication Test:
Direct RPC call to server works correctly:
```bash
$ curl -X POST http://localhost:7778/rpc \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"schema/runbook","params":{"runbook":"examples/service-health-branching.runbook.yaml"}}'

Response: ✅ SUCCESS
{"jsonrpc":"2.0","id":1,"method":"","result":{"inputs":{"server_name":{"from":"prompt","description":"Server to check"}},...}}
```

### WebSocket Connection Issue:
Browser console logs show repeating pattern:
```
[log] gert: WebSocket connected
[log] gert: WebSocket closed
[log] gert: WebSocket connected
[log] gert: WebSocket closed
[log] gert: WebSocket connected
```

The WebSocket connection is unstable, connecting and immediately closing. This prevents the `schema/runbook` RPC call from completing in the browser, even though direct HTTP POST works.

### Diagnosis:
The fix to check for `from === 'prompt'` is correct and in place. However, the **WebSocket connection instability** prevents the schema retrieval step from succeeding, so the code that checks for prompt inputs never executes.

## Test Result: ❌ FAIL

**Does the input form appear?** **NO**

**What I see:**
- The page does not change after clicking Run
- No input form with `data-testid="input-form"` is rendered
- The empty state message remains visible
- WebSocket connections repeatedly fail

## Recommendation:
The code fix (b3ad8ac) is correct, but there's a **separate infrastructure issue** with WebSocket connectivity between the Vite dev server and the gert backend. The HTTP RPC endpoint works fine, but the browser client cannot maintain a WebSocket connection, which appears to be blocking or interfering with the schema retrieval flow.

**Next Steps:**
1. Investigate why WebSocket `/ws` endpoint keeps closing
2. Check if RunbookRunner properly handles schema/runbook calls when WebSocket is unavailable
3. Consider making schema/runbook work purely over HTTP POST (which is proven to work)

---
**Test Environment:**
- OS: Darwin (macOS)
- Gert binary: Built from commit b3ad8ac
- Node/npm: Using project's package.json dependencies
- Browser: Chromium (Playwright)
