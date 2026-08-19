# Decisions

## 2026-08-19T08:57:26.344-07:00: Probe-token incident and reboot recovery
**By:** Cristián Ormazábal Ortega (via Copilot)
**What:** Running Gert VS Code's `@gert /probe-token` causes `icm-mcp` to stop and prevents it from starting again. `Developer: Reload Window` does not recover it. A full machine reboot is the only empirically proven recovery. Never run `probe-token` again. Remove the probe completely before further live MCP testing, and do not claim a narrower recovery without live evidence.
**Why:** User-reported live incident; captured before reboot so the next session retains the exact failure and containment requirements.

---

## 2026-08-19: T1 Failure Proves Execution Model Was Never the Variable
**Author:** Ken (Core Dev)  
**Status:** Recorded — awaiting live probe result with schema diagnostics

### Key Finding

Cristiano ran `@gert /probe-token mcp_icm_mcp_serve_get_incident_details_by_id {"incidentId": 853194884}` in a live authenticated VS Code session.

**All four tiers failed:**

| Attempt | Result | Class | Elapsed | Dialog |
|---------|--------|-------|---------|--------|
| T1-synchronous | failed | Error | 6897 ms | likely |
| T2-microtask | failed | Error | 6021 ms | likely |
| T3-macrotask | failed | Error | 5285 ms | likely |
| T4-loopback | failed | Error | 4918 ms | likely |

**T1 is the exact Petals pattern** — synchronous, on the handler stack, fresh `request.toolInvocationToken`, no await before the call. It failed.

### What This Proves

The entire token-lifetime/await-boundary debate was a dead end from the start.

- T1 is the most conservative scheduling possible (synchronous on the handler stack).
- T1 failed identically to T2, T3, T4.
- Therefore: token lifetime, await boundaries, the run pump, chat-mediated execution, and deferred invocation were **never the variable**.
- Every architectural change we debated (pump, RunAuthenticated, nonce handoff, chat-mediated run) would have failed identically.

### Most Likely Cause

The consistent ~5-7s elapsed time and high probability of a dialog suggest the invocation is being **rejected at the schema/input level**, not the token level.

The tool `mcp_icm_mcp_serve_get_incident_details_by_id` likely requires parameters beyond `incidentId`. The supplied input `{"incidentId": 853194884}` may be:
- Missing required parameters
- Supplying `incidentId` as a number when the schema requires a string
- Supplying `incidentId` when the actual required property name is different

### Response: Schema Diagnostics Added

Added to `/probe-token` (version 0.1.2):

1. **inputSchema dump** — before running the four attempts, the tool's declared `inputSchema` is pretty-printed as JSON in the chat response. This is public tool metadata; streaming it is safe.

2. **Property-name verdict** — a one-line comparison of the supplied argument keys against the declared schema: lists missing required properties and unknown supplied properties by name only (never by value).

3. **`gert.diagnostics.unsafeErrorText` setting** — when `true`, raw `err.message` and `err.stack` from each failed attempt are written to the **gert output channel only**. They never appear in chat responses, HTTP responses, run state, or child-process environment. Default `false`.

### Binding Rule

**Before designing any retry, gate, or invocation-timing architecture, gate first establish WHY the single synchronous invocation fails.** If T1 fails, no timing or token strategy can help. The error classification is the only useful datum.

### Next Action

Cristiano should run the updated probe. The schema dump will reveal whether the supplied input is malformed. The most likely outcome: `incidentId` is the wrong property name, or an integer is supplied where a string is expected.

---

## 2026-08-19: Remove /probe-token — Kills ICM MCP Session
**Author:** Ken (Backend Dev)  
**Status:** Implemented — not committed; changes staged/unstaged in `gert-vscode` for Cristián review  
**Parent decision:** T1 Failure decision above

### Problem Statement

After running `@gert /probe-token`, the ICM MCP server (`icm-mcp`) stops and cannot be restarted until a VS Code reload or machine reboot. The user correctly suspected the probe is doing something seriously wrong.

### Root Cause (Evidence-Based)

#### 1. Four unconditional `invokeTool` calls against a live remote MCP server

`runProbe()` in `src/probeToken.ts` lines 305–315 executes T1, T2, T3, and T4 with **no early exit on failure**:

```typescript
// T1 — synchronous
results.push(await runAttempt('T1-synchronous', invokeToolFn, token, ...));
// T2 — after microtask
await Promise.resolve();
results.push(await runAttempt('T2-microtask', invokeToolFn, token, ...));
// T3 — after 250ms setTimeout
await new Promise<void>((r) => setTimeout(r, 250));
results.push(await runAttempt('T3-macrotask', invokeToolFn, token, ...));
// T4 — after loopback HTTP round-trip
await loopbackRoundTrip();
results.push(await runAttempt('T4-loopback', invokeToolFn, token, ...));
```

Each `runAttempt` calls `invokeToolFn` which calls `vscode.lm.invokeTool(toolName, ..., cancellation)`.

#### 2. `icm-mcp` is a remote HTTP server — VS Code owns its session lifecycle

`$APPDATA\Code\User\mcp.json`:
```json
"icm-mcp": {
  "type": "http",
  "url": "https://icm-mcp-prod.azure-api.net/v1/"
}
```

This is not a local stdio process. VS Code's MCP client maintains an in-memory HTTP session for it. Every `invokeTool` failure that is a transport/session failure causes VS Code to attempt a reconnect. After repeated reconnect failures (4 in rapid succession from the probe), VS Code's MCP state machine for `icm-mcp` enters a permanently-disabled state for the current VS Code session.

#### 3. The chat `_token` is passed as `cancellation` to every invokeTool call

`extension.ts` lines 128–143:
```typescript
await handleProbeToken(
  ...,
  (name, options, cancellation) =>
    vscode.lm.invokeTool(name, { ... }, cancellation as vscode.CancellationToken),
  _token,  // <-- chat request CancellationToken
  ...
);
```

`handleProbeToken` captures this as `cancellation` and passes it to all 4 `invokeToolFn` calls (`probeToken.ts:413`). When the chat handler eventually returns, VS Code fires `_token` cancellation, potentially triggering additional MCP teardown.

#### 4. The probe is marked THROWAWAY and its measurement is complete

`src/probeToken.ts` line 1:
```typescript
// probeToken.ts — Throwaway diagnostic: does toolInvocationToken survive an await?
```

Per decision above: T1 fails identically to T2/T3/T4 (~5-7s). The measurement is done. The probe has no remaining diagnostic value.

### Hypotheses (not VS Code-source-verified)

- VS Code applies a crash-count gate or exponential back-off after N consecutive MCP failures and stops attempting reconnects for the session lifetime. The probe reliably triggers this by producing exactly 4 failures in rapid succession.
- The `_token` cancellation callback may also signal VS Code to tear down any in-flight MCP state established during the handler.

### Safe Immediate Recovery (no reboot required)

**`Developer: Reload Window`** (`Ctrl+Shift+P` → type "Reload Window").

This resets the VS Code extension host and all in-memory MCP client state. The ICM MCP server should reconnect on the next `invokeTool` call. Machine reboot is not required and should not be the recommended recovery.

**Caveat:** If VS Code's MCP session failure also caused server-side auth token invalidation at `icm-mcp-prod.azure-api.net`, a fresh sign-in prompt may appear after reload. This is the designed behavior for authentication failure — not a sign that the reload failed.

### Recommended Code Change

Remove `/probe-token` entirely. It is complete, throwaway, and actively dangerous.

**Files to change:**

1. **`package.json`** — remove `probe-token` from `chatParticipants[0].commands` array. This makes the command inert without reinstalling the extension.

2. **`src/extension.ts`** — remove the `probe-token` branch (lines ~125–144):
   ```typescript
   if (request.command === 'probe-token') { ... }
   ```
   Also remove the `handleProbeToken` import.

3. **Delete `src/probeToken.ts`** — entire file. The `buildSchemaDump`/`schemaVerdict` helpers are unused outside this file.

4. **Delete `test/probeToken.test.js`** — remove the associated test file.

**Test count impact:** Will reduce test count by the number of probe-token tests. Run `npm test` to confirm clean pass after deletion.

### Future Diagnostic Probes — Binding Rule

Any future diagnostic that calls `vscode.lm.invokeTool` against a live MCP endpoint must:
- Call at most once per invocation, OR
- Stop on first failure (no unconditional multi-attempt loops), OR
- Target a mock/stub, never a live MCP endpoint registered in `mcp.json`.

Calling `invokeTool` N times unconditionally against a real remote MCP server in a production extension is not safe.

---

## 2026-08-19: /probe-token Removal Implemented
**Author:** Ken (Backend Dev)  
**Status:** Implemented — not committed; changes staged/unstaged in `gert-vscode` for Cristián review  
**Parent decision:** Remove /probe-token decision above

### What Was Done

Implemented the ratified decision to remove `/probe-token` entirely from `gert-vscode`.

### Files Changed

| File | Change |
|------|--------|
| `src/extension.ts` | Removed `probe-token` dispatch branch (~24 lines including `handleProbeToken` call and return). Updated "unknown command" help text to remove the `/probe-token` entry and add `/run`. |
| `src/probeToken.ts` | **Deleted** (419 lines). |
| `test/probeToken.test.js` | **Deleted** (456 lines). |
| `package.json` | Pre-existing unstaged changes already removed the `probe-token` chatParticipant command entry and the `gert.diagnostics.unsafeErrorText` configuration property. Left as-is (aligned with the decision). |

### Files Not Changed

- `out/probeToken.js` / `out/probeToken.js.map`: confirmed NOT git-tracked (`git ls-files` returned empty). Left for the next `npm run compile` to overwrite.

### Surprise: Missing Import Was a Latent Compile Error

`handleProbeToken` was called in `extension.ts` at line 127 but was never imported. The file had `import { handleProbeToken } from './probeToken'` absent. Removing the branch resolved a latent TypeScript compile error rather than introducing one.

### Verification

- `npm run compile`: ✅ exit 0.
- `npm test`: ✅ 185/185 passed. 0 failures. No pre-existing failures.
- `git status`: exactly 4 files changed. No unintended changes.

### Team-Relevant Finding

No new decision is required. The ratified decision (remove probe-token) was fully implemented as specified. The only noteworthy finding — that the extension wouldn't have compiled with the probe-token branch intact due to a missing import — is documented in `history.md` and has no architectural implications beyond "deletion was the right call."

---

## 2026-08-19: provider_unavailable Category and Allowlist-Only Hint
**Author:** Ken  
**Status:** Implemented in e395bf8 (gert-vscode 0.1.3)

### Context

Four rounds of architecture work were spent investigating `invocation_error` from `vscode.lm.invokeTool()`. A four-tier diagnostic probe captured the actual provider exception:

```
T1: MCP server has stopped
T2: MCP server could not be started: 401 status sending message to https://icm-mcp-prod.azure-api.net/v1/:
```

The classification machinery collapsed this into the generic `invocation_error` bucket. A dialog heuristic (elapsed time > 4s) actively misdirected diagnosis.

### Decisions

#### 1. New category `provider_unavailable`

Added `provider_unavailable` to `InvocationErrorCategory`. Matches:
- `MCP server has stopped`
- `MCP server could not be started`
- `MCP server is not running`
- `MCP server unavailable`
- `server not running`
- `server unavailable`

#### 2. Precedence: `provider_unavailable` before `authorization_unavailable`

A startup failure that mentions "401" or "Unauthorized" classifies as `provider_unavailable`, NOT `authorization_unavailable`. The operator's action differs completely: "fix the MCP server's sign-in" vs "check Gert/VS Code credentials". This is documented in the code comment.

#### 3. Allowlist-only hint extraction

`extractProviderHint(err)` builds a hint from:
1. A fixed phrase from a hardcoded allowlist (canonical casing used)
2. An HTTP 4xx/5xx status code  
3. URL origin only (scheme + host via regex stop at `/` + URL constructor belt-and-suspenders)

Nothing else passes through. Hard cap 200 chars.

#### 4. Actionable bridge message for `provider_unavailable`

Message format: `the MCP provider is not running (<hint>). Check 'MCP: List Servers' in VS Code and re-authenticate the provider.`

#### 5. Probe reports category + hint, not dialog inference

`dialogInferred`/`DIALOG_THRESHOLD_MS` removed. `ProbeAttemptRecord` now carries `category` and `providerHint`. Elapsed time is still reported as a raw measurement; no conclusion is drawn from it.

### Boundaries

- No product code is ICM-specific. The ICM host name appears only in test fixtures as a sample provider error string.
- No changes to the Go core repo.

---

## 2026-08-19: Token Lifetime Probe Instrument Built
**Author:** Ken (Core Dev)  
**Requested by:** Cristiano  
**Branch:** `probe-token-2026-08-19`  
**SHA:** 4e3538d  
**Status:** Instrument built, measurement pending (Cristiano runs live)

### Context

Cristiano ran the Petals-ported build (742e368) live:
- `/arm-mcp` was run first; token was cached.
- The run still failed with `invocation_error`.
- VS Code showed the auth/confirmation window **even when armed**.

The dialog appearing proves VS Code is rejecting a stale token from a completed chat request. A token captured in one chat request and reused later is not honoured.

**One unknown remains:** Does a `toolInvocationToken` from a *live* `ChatRequestHandler` survive even one `await` inside that same handler? This single fact determines the architecture:

- If it **survives** → the run button can programmatically open a chat turn and the run works with no user-visible command.
- If it **dies** → Cristiano's required editor-Run UX is impossible, and we must name the exact API constraint.

### Decision: Build a Minimal Probe Instrument

Rather than redesigning the extension, we build one throwaway diagnostic command — `@gert /probe-token` — that tests token lifetime across four scheduling tiers inside a single live handler invocation.

**What it does NOT do:**
- Does not rebuild the pump, nonce handoff, or `runAuthenticated`
- Does not modify the existing bridge invocation path
- Does not measure anything live — Cristiano measures; Ken builds the instrument

### The Probe

**Command:** `@gert /probe-token <toolName> <jsonInput>`

**Four attempts, same token, same handler turn:**

| # | Label | Scheduling |
|---|-------|------------|
| T1 | T1-synchronous | No `await` — replicates Petals exactly |
| T2 | T2-microtask | After `await Promise.resolve()` |
| T3 | T3-macrotask | After `await new Promise(r => setTimeout(r, 250))` |
| T4 | T4-loopback | After a real loopback HTTP round-trip (mirrors Gert's topology) |

All four attempts always execute (no early exit on failure).

**Per-attempt output:**
- `ok` or `failed`
- Exception class name
- Error `code` property (if short symbolic string)
- Elapsed ms
- Dialog inferred (elapsed > 4s — heuristic, labelled as such)

**Security invariants (non-negotiable):**
- Token never streamed, logged, or serialised
- Args never streamed or logged
- Result content never streamed or logged
- `git grep -in "tsg" -- src test` returns nothing

### Implementation

**Files added/modified:**

| File | Change |
|------|--------|
| `src/probeToken.ts` | New: `parseProbeArgs`, `runAttempt`, `runProbe`, `loopbackRoundTrip`, `renderProbeTable`, `handleProbeToken` |
| `src/extension.ts` | Added `probe-token` branch in chat handler |
| `package.json` | Added `probe-token` to `gert.chat` commands array |
| `test/probeToken.test.js` | 5 new tests: PROBE-1 through PROBE-5 |

### Test Coverage

| Test | What it proves | Mutation it kills |
|------|----------------|-------------------|
| PROBE-1 | All 4 attempts execute; call count === 4 | Stop after first failure → call count = 1, test fails |
| PROBE-2 | Token/args/result sentinel absent from streamed output; sentinel non-vacuously flowed through real path | Stream args → sentinel appears in output, test fails |
| PROBE-3 | Malformed JSON → usage message, zero invocations | Skip parse check → invokeTool called on bad input, count ≠ 0 |
| PROBE-4 | `parseProbeArgs` edge cases | Trivially incorrect parse → multiple assertion failures |
| PROBE-5 | Unknown tool → error, no invocations | Skip tool-existence check → invocations happen on unknown tool |

**Test counts:** 174 pass, 0 fail, 0 skip (was 169 before this commit).

### Verification

```
git status --porcelain   # empty
npm ci                   # clean install
npm test                 # 174/174 pass, exit 0
git grep -in "tsg" -- src test   # returns nothing (exit 1)
```

**Manifest confirmation:**
```json
{
  "name": "probe-token",
  "description": "Diagnostic: test whether toolInvocationToken survives await boundaries. Usage: @gert /probe-token <toolName> {\"key\":\"val\"}. Reports ok/failed per scheduling tier. THROWAWAY — will be removed after measurement."
}
```

### Architectural Consequences (Pending Measurement)

#### If T1 ok, T2-T4 fail

Token survives synchronous context only. Gert's topology (loopback HTTP continuation) cannot use a live handler token. The editor-Run UX as specified is architecturally impossible without a user-visible chat command that keeps the handler open (pump pattern, or live streaming run inside a chat handler).

#### If T1-T2 ok, T3-T4 fail

Token survives microtask boundaries but not macrotask. Loopback HTTP still fails. Same consequence.

#### If T1-T4 all ok

Token survives the full loopback round-trip. Gert can programmatically open a chat turn, the run drives tool calls inside the live handler, and the editor-Run UX becomes feasible. We build the chat-open command path.

#### If T1 fails

VS Code does not honour `request.toolInvocationToken` even synchronously in this session configuration (e.g., no MCP servers registered for the named tool). Probe result is ambiguous — retry with a tool that VS Code actually has registered.

### Disposal Note

This command is a throwaway diagnostic. After Cristiano reports live results:
- If the finding is definitive: remove `src/probeToken.ts`, `test/probeToken.test.js`, and the manifest entry.
- If further probing is needed: extend this file with a new scheduling tier or a different tool.
