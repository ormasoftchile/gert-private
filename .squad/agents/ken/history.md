# ken — Work Summary

**Period:** 2026-08-15 to 2026-08-19
**Total commits:** 15+
**Full history archived:** history-archive.md

---

## Summary

Ken (Core Dev) focused on Phase 2 VS Code extension work, with emphasis on security-critical redaction proofs and architecture validation. Four major deliverables:

1. **Runhandoff Extraction (2026-08-19)** — Extracted security-critical handoff sequence into `src/runHandoff.ts` (vscode-free, DI). Addressed 4th vacuous-mutation defect: original proof targeted `buildRunChatQuery()` which cannot receive raw inputs by signature. Cristiano's mutation leaked inputs into query; all 208 tests passed (proof was blind). Fixed by extracting call site, injecting REAL collaborators, and proving data flow. Result: 215/215/0/0.

2. **Usable Chat Run UX (2026-08-18)** — Pending-run store (single-use, 30s TTL), chat-open command, required-input prompting, arg parser. All paths tested with live mutations. Result: 202/202/0/0.

3. **Layered MCP Invocation Model (2026-08-18)** — Restored Petals' 3-layer precedence (pump active → direct invoke; pump inactive + armed → cached token retry; neither → deny). Regression fix for dead `/arm-mcp` token and `gert.previewGraph` pre-existing path. Result: 208/208/0/0.

4. **In-Handler Pump Architecture (2026-08-18)** — Pump processor holds chat handler open with `Promise.race([terminal, cancel, deadline])`. Tool invocations routed through pump, not deferred. Empirically unproven in live VS Code session. Result: 164/164/0/0.

---

## Key Learnings

### Vacuity in Redaction Proofs (4th Occurrence)

A test of a pure helper function whose signature structurally cannot receive the sensitive data is vacuous. The test cannot detect a mutation at the production call site. **Remedy:** Extract call-site logic into testable module, inject REAL collaborators, prove data flow is observable.

### Systemic Bug #13 — Stale Build Artifacts

Tests importing from gitignored `out/` directory without auto-recompile invalidate mutation testing in both directions (false green on mutation, false red on revert). Fixed by adding `"pretest": "npm run compile"` to package.json. **Binding rule:** Always regenerate build artifacts after source mutations, before running tests.

### Mutation Proof Discipline

1. Proofs must target production wiring, not pure helpers.
2. Non-vacuity guards required — prove data actually flowed through.
3. Path-specific assertions for N-path functions (N separate tests).
4. Hung test suite is not a mutation kill — add bounded per-test timeout.

### Token Authorization vs. Execution Lifetime

`vscode.lm.invokeTool()` enforces execution lifetime (must be called during active handler), not token possession. The cached `/arm-mcp` token alone is insufficient for deferred invocations. Pre-emptive gates that refuse calls without a pump make legitimate layer-2 best-effort paths unreachable.

### Command Source Verification

`vscode.commands.executeCommand('workbench.action.chat.open', {...})` sourced from VS Code `chatActions.ts`. Not in `@types/vscode`; requires source inspection. Argument shape: `IChatViewOpenOptions { query, isPartialQuery?: boolean }`.

---

---

## Learnings — Probe Kills ICM MCP Session (2026-08-19)

### Root Cause: 4 Unconditional invokeTool Calls Exhaust VS Code's MCP Restart Budget

`runProbe()` (`probeToken.ts:305-315`) executes T1–T4 unconditionally — no early exit on failure. Each attempt calls `vscode.lm.invokeTool(..., cancellation)` where `cancellation` is the live chat handler `_token` (`extension.ts:144`).

The `icm-mcp` entry in `$APPDATA\Code\User\mcp.json` is type `http` (remote HTTPS), not a local stdio process. VS Code's MCP client maintains an in-memory session for it. Each `invokeTool` failure causes VS Code to attempt an MCP client reconnect. After 4 consecutive reconnect failures within one chat turn, VS Code's MCP state machine for `icm-mcp` enters a permanently-disabled state that persists for the lifetime of the VS Code session — this is why only a VS Code reload (or machine reboot) recovers it.

### Proven Facts
- `runProbe()` always executes all 4 attempts (`probeToken.ts:307–315`) — no early exit guard.
- `extension.ts:144`: `_token` (chat CancellationToken) is passed as `cancellation` to every invokeTool call.
- `mcp.json`: `icm-mcp` = `{ type: "http", url: "https://icm-mcp-prod.azure-api.net/v1/" }` — VS Code owns its connection lifecycle, not the OS.
- VS Code's MCP client state for HTTP servers is in-memory and scoped to the VS Code session.

### Hypotheses (not VS Code source-verified)
- VS Code applies a progressive back-off or crash-count gate after N consecutive MCP session failures and stops retrying for the VS Code session lifetime.
- The chat `_token` being passed as the `cancellation` argument to `invokeTool` may cause VS Code to associate the MCP session lifecycle with the handler turn; when the handler returns, VS Code cancels in-flight MCP state.

### Safe Recovery
**`Developer: Reload Window`** (`Ctrl+Shift+P`). Resets extension host and all VS Code MCP client state. Machine reboot is not required and is excessive. (Not 100% proven if server-side auth tokens are also invalidated — but the VS Code state reset is the primary fix.)

### Minimal Code Fix
Remove `probe-token` entirely: it is marked THROWAWAY in source (`probeToken.ts:1`) and its measurement is complete (all tiers fail identically). Steps:
1. Remove the `probe-token` entry from `chatParticipants.commands` in `package.json`.
2. Remove the `probe-token` branch from the chat handler in `extension.ts` (~lines 125–144).
3. Delete `src/probeToken.ts` and `test/probeToken.test.js`.
The `buildSchemaDump`/`schemaVerdict` utilities in `probeToken.ts` are not used anywhere else — safe to delete.

### Binding Rule
A diagnostic probe that calls `invokeTool` N times unconditionally against a real remote MCP server is not safe in a production extension. Any future diagnostic must either: (a) call invokeTool at most once, (b) stop on first failure, or (c) operate against a mock/stub, never against a live MCP endpoint.

---

---

## Learnings — Probe-Token Removal (2026-08-19)

### What Was Removed

Deleted `src/probeToken.ts` (419 lines) and `test/probeToken.test.js` (456 lines). Removed the `probe-token` dispatch branch from `src/extension.ts` (24 lines net reduction). The `package.json` already had pre-existing unstaged changes removing the `probe-token` chatParticipants command entry and the `gert.diagnostics.unsafeErrorText` configuration property — those were aligned with the task and left as-is.

### Where the Code Actually Lived

- `src/extension.ts` lines ~124–146: the `if (request.command === 'probe-token') { ... }` block and the `handleProbeToken` call.
- The "unknown command" help text at ~line 149 also referenced `/probe-token` and was updated to mention `/run` instead.
- `out/probeToken.js` and `out/probeToken.js.map` were NOT git-tracked (confirmed via `git ls-files`); left for next compile to overwrite.

### Surprises

1. **No import for `handleProbeToken` in extension.ts** — the function was called at line 127 but never imported. The extension would have failed to compile with the probe-token branch in place. Removal fixed a latent compile error rather than introducing one.
2. **package.json was already partially cleaned** — the `probe-token` command entry and `gert.diagnostics.unsafeErrorText` config had already been removed in the working tree before this session. The pre-existing changes were aligned with the decision; I did not revert them.
3. **No `probe-token` in the `gert.diagnostics` configuration namespace required follow-up** — the `unsafeErrorText` config reference inside the probe-token branch was also eliminated by removing the branch, so no residual dead config references remained in code.

### Verification Results

- `npm run compile`: ✅ exit 0, zero errors.
- `npm test`: ✅ 185/185 passed (was 215 before Petals lifecycle port, probe tests accounted for the delta). 0 failures. 0 pre-existing failures.
- `git status`: exactly 4 files changed — `package.json` (pre-existing), `src/extension.ts` (edited), `src/probeToken.ts` (deleted), `test/probeToken.test.js` (deleted). No unintended changes.

### Binding Rule

When retiring a VS Code extension command that calls `vscode.lm.invokeTool` against a live MCP endpoint: check the manifest (`package.json` chatParticipants commands), the dispatch branch in the handler, any related configuration contributions, the source module, its test file, and `out/` build artifacts (verify whether tracked). A missing `import` for a called function is a latent compile error — deletion fixes it rather than introducing one.

---

## Outstanding Items

- Live VS Code validation of pump processor (awaited continuation in handler)
- Cached token reliability for layer-2 fallback (empirically unproven)
- `isPartialQuery: false` auto-submission behavior (not unit-testable)

---

## Binding Rules Documented

1. Extract call-site logic with real collaborators for security-sensitive data flow testing.
2. Regenerate build artifacts after every source mutation before running tests.
3. All async tests require bounded per-test timeout (measure slowest legitimate test).
4. Static scans complement runtime mocking for host-provided APIs.
5. Layer-ordering proofs exercise each layer independently.
6. Non-vacuity controls mandatory in every redaction test.

See history-archive.md for full session-by-session details, mutation tables, and test counts.

---

## Team Update (2026-08-19T16:13:34Z)

**From Scribe:** Ken's `/probe-token` removal successfully completed and implemented in gert-vscode. Tests pass (185/185). Changes unstaged in gert-vscode for Cristián review. Root cause of probe danger: 4 unconditional invokeTool calls exhausted VS Code's MCP restart budget for the session. See decisions.md for full decision record and binding rules.

---

## Learnings - Petals Lifecycle Port (2026-08-19)

### Stop-and-Reverse: Architectural Premise Was Wrong From Commit One

The entire pump/runAuthenticated/nonce-handoff stack was invented to solve a problem Petals never had: the fear that invokeTool requires an active chat handler stack. Petals invokeMcpTool calls getToolToken() (possibly undefined) and invokes unconditionally. The pre-invoke gate was our addition, not a VS Code requirement. Reading the reference implementation first would have prevented three commits of work.

Binding rule: Before adding a gate or refusal, verify the reference implementation actually has one.

### Token Presence Is Dialog Suppression, Not Authorization

Petals source comment (mcpBridge.ts:10): the token is "to avoid confirmation dialogs." It is NOT an authorization credential. invokeTool with toolInvocationToken: undefined is entirely valid -- VS Code may show a consent dialog but will not reject the call for lack of a token.

### Two-Attempt Pattern Is Inline, Not a Separate Queue

The Petals retry (Canceled + token present -> retry without token) is a two-line try/catch in the invocation path, not a pump, queue, or separate module. We overcomplicated it by building RunPump, RunLoop, RunClient for a problem that needed three lines.

### Deletion Is Progress

Removing 57 tests, 6 source files, 2100+ lines of code, and a user-facing command is unambiguously better than extending the wrong abstraction. Resistance to deletion is the anti-pattern.

### Vacuous Proof Pattern (5th Occurrence -- Different Shape)

INVTOKEN-1 tested that the bridge returned no_active_run and invokeCount === 0. When we removed the gate, INVTOKEN-1 would vacuously pass (it tested the thing we just deleted). Replaced with a spy-count assertion proving invokeTool IS called. Pattern: after removing a gate, tests that asserted the gate fired must be REPLACED, not merely deleted.

### Test Count Accounting

Before: 215. Removed 57 pump/handoff tests. Added 11 petalsLifecycle tests. Replaced 1 INVTOKEN-1 assertion. Net: 215 - 57 + 11 = 169. All pass.

---

## Architectural Record: VS Code toolInvocationToken Is NOT Authorization

**Commit:** 742e368 (2026-08-19)  
**Decision reference:** `.squad/decisions.md` — "Decision: Port Petals Invocation Lifecycle, Remove Pump/RunAuthenticated Stack"

The VS Code toolInvocationToken serves ONE purpose: to suppress confirmation dialogs. It is not an authorization credential. Petals has no pre-invoke gate and never refuses invocation based on token presence.

**Invariant:** McpBridge.handle() must:
1. Read cachedToken (may be undefined)
2. Invoke UNCONDITIONALLY with cachedToken
3. If Canceled AND cachedToken !== undefined: retry with undefined
4. Otherwise: classify error and return

The bridge NEVER refuses to invoke due to missing token. Token absence means undefined is passed; VS Code may show a consent dialog, which is the designed behavior.

**Consequences for Future Work:**
- No pre-invoke gate exists. @gert /arm-mcp is optional dialog suppression only.
- gert.runAuthenticated (deferred invocation via second action) was invented and is now removed.
- The pump/queue pattern does not exist in Petals and should not be added to Gert.
- Two-attempt retry (try with token, catch Canceled + token present -> retry undefined) is the only control flow.

---

## Learnings — Probe Schema Diagnostics (2026-08-19)

### T1 Failure Reverses All Prior Architecture Assumptions

Cristiano's live probe showed T1 (synchronous, on handler stack, exact Petals pattern) failed in ~7s, identically to T2/T3/T4. Token lifetime, await boundaries, the run pump, chat-mediated execution — none of these were ever the variable. Every architectural commit since the probe was designed to solve a problem that doesn't exist at the execution-timing layer.

**Binding rule:** Before any retry/gate/timing architecture, establish WHY the single synchronous invocation fails. If T1 fails, nothing else matters.

### Schema Mismatch Is the Most Likely Cause

A consistent ~5-7s failure with a blocking dialog on every attempt is the signature of a malformed invocation, not a token issue. The tool's declared `inputSchema` was never inspected. The supplied `{"incidentId": 853194884}` may be:
- Missing additional required parameters
- Supplying `incidentId` as integer when schema requires string
- Using the wrong property name entirely

### Diagnostic Hygiene: Public Metadata Is Safe to Stream

The tool's `inputSchema` is public metadata registered in `vscode.lm.tools`. It is safe to stream to the chat response. Property names in the verdict are safe. Supplied argument VALUES are not safe (same rule as tokens/args/results). The two categories must never be conflated.

### unsafeErrorText Pattern: Hard Boundary Between Diagnostic Surfaces

Raw `err.message` from a provider is potentially sensitive (it may contain user data, credentials, or internal system paths). The setting `gert.diagnostics.unsafeErrorText` gates it to the output channel only. The hard boundary — never in chat/HTTP/state/env — is enforced by code review and mutation-tested:
- Mutation: route raw message to chat response → PROBE-7 or PROBE-8 fails.
- Mutation: print input VALUE in verdict → PROBE-6 fails.

### Error Property Enumeration

`Error.prototype.message` and `stack` are non-enumerable in V8 (ECMAScript spec §20.5.5.3 sets Enumerable: false). `Object.keys(err)` naturally excludes them. Custom properties added after construction (e.g., `err.code = 'X'`) are own-enumerable and safe to log. Explicitly excluding `message` and `stack` in the short-props loop is belt-and-suspenders against non-standard Error subclasses.

---

## Learnings — Token Lifetime Probe (2026-08-19)

### The One Unknown: Does toolInvocationToken Survive an `await`?

Cristiano's live test confirmed the cached `/arm-mcp` token is rejected (VS Code shows the auth dialog even when the store is armed). This proves a stale token from a completed chat turn is not honoured. But nobody has established whether a token from a *live* handler survives even one `await`. The `@gert /probe-token` command was built as a minimal diagnostic instrument to answer this question empirically.

### Probe Architecture

Four scheduling tiers, all in one live ChatRequestHandler turn, same token:
- T1: synchronous — before any `await` (replicates Petals exactly)
- T2: after `await Promise.resolve()` (microtask boundary)
- T3: after `await new Promise(r => setTimeout(r, 250))` (macrotask boundary)
- T4: after a real loopback HTTP round-trip (mirrors Gert's actual topology)

All four attempts always execute regardless of earlier failures (no early exit on failure).

### Security Discipline Maintained

The probe never logs, streams, or serialises the token, tool args, or result content. Only: label, ok/failed, exception class, error code (if short symbolic string), elapsed ms, dialog inferred from elapsed time.

### Manifest Entry Is Non-Negotiable for Chat Commands

A chat slash command absent from `package.json` contributes → chatParticipants → commands is **inert** — VS Code will not route it. This is the bug that has bitten the engagement twelve times. Always add the manifest entry before claiming a command works.

### Mutation Test Non-Vacuity Rule (Probe-Specific)

PROBE-2 uses a sentinel that flows through `handleProbeToken → runProbe → runAttempt → invokeToolFn` — the real production path. Non-vacuity is verified by asserting the sentinel was received by the spy. A test that only checks a pure renderer (not the full path) cannot detect a mutation at the call site.


**For Ken's successors:** Read Petals mcpBridgeGeneric.ts:320-345 and mcpBridge.ts:10-22 before designing any token handling or tool invocation flow. If you want to add a gate or refusal, first show that Petals has one.

---

## Learnings — Provider Unavailable Triage (2026-08-19)

### Allowlist-Only Hint Extraction

`extractProviderHint` is the correct pattern for surfacing safe operator-facing context from an untrusted provider error: build the hint entirely from matched pieces (fixed phrase, HTTP status code, URL origin). Never pass arbitrary provider text through even one character. The URL path is excluded by the regex `[^\s/?#]*` (structural exclusion), not by a post-capture strip — that is a stronger guarantee because the path never enters the computation at all.

### Precedence Must Be Deliberate and Documented

When a new category can overlap with an existing one (here: `provider_unavailable` vs `authorization_unavailable` on a startup 401), the precedence ordering must be documented in a code comment at the check site. A test asserting the precedence (the overlap fixture) is mandatory — otherwise the ordering is invisible and can be silently broken.

### URL Path Is Structurally Excluded, Not Behaviorally

The regex `[^\s/?#]*` stops at the first `/`, `?`, or `#`. This means the URL path/query/fragment never enters `urlMatch[0]`. The `new URL()` parse + origin extraction is belt-and-suspenders for edge cases. The mutation `urlMatch[0]` instead of `u.protocol + '//' + u.host` does NOT change the output because `urlMatch[0]` is already origin-only. The protection is structural.

### Mutation Testing for Hint Redaction

The correct mutation for leaking URL path would be to change the regex stop characters — but even then the URL constructor extracts only the origin. Tests asserting path/query absence are behavioral (they prove the guarantee) but the mutation proof is structural rather than code-path-based.

### `dialogInferred` Was Actively Misleading

The `DIALOG_THRESHOLD_MS` heuristic inferred a consent dialog from elapsed time > 4s. The live probe showed ~5s was the MCP server's failed start round-trip, not a consent dialog. This heuristic was the source of one full round of wasted architectural work. Any time-based behavioral inference that could explain a slow failure via a wrong cause is a liability — remove it and report the real classification instead.

