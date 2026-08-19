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

**For Ken's successors:** Read Petals mcpBridgeGeneric.ts:320-345 and mcpBridge.ts:10-22 before designing any token handling or tool invocation flow. If you want to add a gate or refusal, first show that Petals has one.

