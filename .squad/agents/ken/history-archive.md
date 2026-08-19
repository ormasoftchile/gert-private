# ken

## 2026-08-18 — Layered MCP Model Regression Fix (commit d98cc8b)

**Sprint:** Folded into the usable-chat-run-UX round per Cristiano's stoppage.

**Defect:** The exclusive `hasActivePump?.() === false` gate made `/arm-mcp` dead code. When no pump was active but a cached token was armed, `no_active_run` fired immediately. This regressed `gert.previewGraph` (pre-existing webview path) and misread "unreliable" as "forbidden".

**Fix:** Restored Petals' layered model. Layer 2 reuses `invokeWithTwoAttempts` — same retry semantics, no drift. `/arm-mcp` response text updated to describe best-effort layer 2.

**Tests:** 208/208/0/0 (was 202/202/0/0). 6 new tests: LAYER-1, LAYER-2, LAYER-2b, LAYER-4, LAYER-5a, LAYER-5b. PUMP-2 updated (unarmed case now tests true floor).

**Mutations killed:** LAYER-1 (precedence inversion), LAYER-2/2b/4/5a/5b (layer-2 deletion), LAYER-4 (no_active_run conflation), LAYER-5a (token logged in retry line).

---

## 2026-08-18 — Usable Chat Run UX (commit 4d5dbb9)

**Sprint:** Usable Chat Run UX — @gert /run active-editor default, picker, pending-run store, arg parse.

**Status:** ACCEPTED. 202/202/0/0 tests. 8 files changed (874 insertions, 16 deletions). SHA 4d5dbb9.

**Items delivered:**
- Active editor default: bare `@gert /run` auto-uses active `*.runbook.yaml`
- QuickPick picker: multi-root safe via `vscode.workspace.findFiles`, readable labels
- Required input collection: `filterRequiredInputs` prompts only `required===true` inputs
- `gert.runAuthenticated` editor-title/context command; opens chat with nonce handoff
- Pending-run store: single-use, 30s TTL, secrets never in query or logs
- Explicit `parseRunArgs` replaces positional `parts[0].includes('=')` heuristic
- `formatRunStartLog` extracted pure for path-B redaction proof
- Run document link on terminal completion: `[Open run document](${base}/runs/${id}/document)`

**Mutations killed (6):**
- PARSE-5: lastIndexOf mutation on value split
- STORE-2/2b: remove `_store.delete(nonce)` (single-use)
- STORE-3: remove TTL expiry check
- RRES-2: remove `.runbook.yaml` check on active editor
- QBLD-redact-path-A: store-embedded nonce proves buildRunChatQuery never leaks inputs
- QBLD-redact-path-B: embed-secret mutation proves formatRunStartLog never leaks inputs

---

## Learnings (2026-08-18 — ken-14, commit 4d5dbb9)

**Two redaction paths, not one.** The ask explicitly requires injection on each path that could leak: (A) the chat query string and (B) the output-channel log line. Covering only path A again produces a vacuous proof for path B. Fix: extract `formatRunStartLog` as a pure function so path B is testable without a VS Code host. Both paths now have a live mutation proof.

**workbench.action.chat.open: source confirmed.** Command ID is `CHAT_OPEN_ACTION_ID = 'workbench.action.chat.open'` exported from VS Code source `src/vs/workbench/contrib/chat/browser/actions/chatActions.ts` (main branch). Argument shape is `IChatViewOpenOptions { query: string; isPartialQuery?: boolean }`. Confirmed additionally by GitHub issue microsoft/vscode#210819 and a direct web fetch of the source file. The `isPartialQuery: false` flag causes VS Code to submit the prompt immediately rather than staging it for user review.

**Pending-run store single-use is security-critical, not just an optimization.** An unclaimed entry after TTL is inert. But more importantly, `claimPendingRun` removes the entry BEFORE checking expiry — so even an expired entry is consumed on the first claim attempt. This prevents a race where two rapid claims both get `undefined` but the entry stays in the store.

**PowerShell string replacement is unreliable for complex patterns.** The RRES-2 mutation revert using `-replace` incorrectly reassembled the condition, leaving a regression that only showed up on the next test run. Lesson: prefer `edit` tool for reverts; use PowerShell replacement only for simple, unambiguous substitutions and verify with `view` immediately after.



**Current sprint:** Three commits (93214d0, d173ad7+130b81e, f667f0a) all ACCEPTED and deployed to gert-vscode.

**Status:**
- Closed-enum invocation error classifier: 143/143 tests, 3 mutation proofs
- toolInvocationToken extract-to-pure architecture: 153/153 tests, 6 mutation proofs
- npm pretest hook (automatic recompilation): 153/153 tests, 2 mutation directions validated
- Systemic bug class #13 (stale build artifacts) identified, documented, fixed, and binding rule applied to all verification work on this engagement

**Prior 2026-08-15–2026-08-17 work:** 13 completed features archived (dynamic runbook includes, runtime portability Phase 1, reachability gate, skip-defect regression guard, gert-vscode reproducibility, Phase 2 prep, Gert reproducibility). Full records in decisions-archive.md and .squad/log/2026-08-18T22-23-36Z-vscode-live-blockers.md.

**SQL Live-Site blockers:** Config scoping (multi-root workspace) and registry parser (meta.Name + transport.mode / transport.type) fixed in commit 1cd7542 (127/127 tests); static source guard (repoBoundary/rule7) added in commit 08de611 (130/130 tests) to prevent regression.

---

## 2026-08-18 — Stale Build Artifacts: Systemic Bug Class #13

**Alert:** All mutation testing on this engagement requires build artifact regeneration after source mutations.

**Finding:** npm test on gert-vscode runs against gitignored out/ directory. Stale out/ after source edit invalidates mutation testing in both directions:
- **False green:** mutate source, skip recompile → npm test runs old out/; mutation appears harmless.
- **False red:** revert source, skip recompile → npm test runs mutated out/; clean tree appears broken.

**Root cause:** No automatic recompilation before npm test.

**Fix:** Added "pretest": "npm run compile" to package.json (commit f667f0a). npm lifecycle hook fires automatically before every 
pm test invocation.

**Binding rule (all verification agents):**
1. After source edit (mutation or revert), regenerate build artifact (tsc, cargo build, npm run compile, make, etc.).
2. Run test suite AFTER artifact regeneration.
3. Trust test results only with this ordering.

**Scope:** All repos with tests importing gitignored build output + no auto-recompile + mutation testing as primary verification.

**Proof:** Mutation worktree _ken_v6:
- Direction 1: mutate chatParticipantGate.ts, 
pm test without manual compile → 152/1 fail (pretest compiled automatically, mutation caught).
- Direction 2: revert source, 
pm test without manual compile → 153/153 pass (pretest recompiled, clean state verified).

Before pretest hook: Direction 2 would have returned 152/1 false red (stale out/ from mutation still present).

---

## Critical Lessons (must re-read every spawn)

- **Mutation proofs must target the production wiring, not a pure helper.** This mistake has happened twice on this engagement: a test mutated a standalone helper function and declared the behavior covered, but the call site in the production path was never exercised. Mutation evidence is only load-bearing when the mutated path is the one actually exercised by the running system.
- **Never cite a repo "precedent" without grepping for it first.** SQL Live-Site cited "Petals" as a proven pattern; the name appeared in documentation but no implementation was found in the actual codebase. Citing an unverified precedent wastes a round-trip and can justify a wrong design. Always grep before claiming a pattern exists.
- **`npm test` against stale `out/` invalidates mutation testing in both directions.** See Systemic Bug Class #13 below.

## 2026-08-18 — Vacuity in redaction proofs; bounded deadlock detection

**Defect: A redaction test that only exercises one path through a function does not prove the other paths are safe.** PUMP-8 forced `Canceled` on attempt 1, so it only exercised the retry path. A token leak injected on the attempt-1-success path (the most common path) produced 161/161/0/0 — the test was completely blind to it. This is the third time a proof has targeted the wrong layer on this engagement.

**Pattern to prevent this:** For any function with N distinct execution paths, write N explicit redaction assertions — one per path — and confirm each assertion has a non-vacuity guard (at least one log line was produced). The "non-vacuity control" canary in the old test proved only that the scanner's `includes()` works on a crafted string; it said nothing about whether real log lines were seen. Those are different claims.

**Defect: A hung test suite is not a mutation kill.** Mutation #7 (remove `pump.close()` from the finally block) caused an infinite hang rather than a named test failure. No timeout was set, so the CI runner would have burned until the job-level limit. A deadlock must fail with a bounded, named message. Fixed by adding `--test-timeout=5000` to npm test (slowest legitimate test measured at 95.8ms; 5000ms = ~52x headroom). Re-confirmed: mutation #7 now fails at 5001–5006ms with PUMP-5/6/7 named.

**Binding rule:** Every test suite that exercises async code must have a bounded per-test timeout. Measure the slowest legitimate test before choosing the value; do not guess.

---

## Learnings (2026-08-18 — In-Handler Invocation Pump, commit 9dfd345)

**Central finding:** Token capture alone does not authorize `vscode.lm.invokeTool()` after the chat request returns. The old `invocation_token_unavailable` pre-invoke gate was built on a false premise. VS Code enforces *execution lifetime*, not token possession.

**Petals is the correct reference, but only its synchronous-handler path is proven.** The `invokeMcpTool` function in Petals invokes tools *immediately inside the active chat handler*, then retries with `token=undefined` on Canceled. Background-path invocations in Petals are not reliably authenticated — Petals itself acknowledges this via its fallback.

**Two-attempt invocation predicate must be word-boundary, not substring.** Matching `/\bCanceled\b/` rather than `includes("cancel")` avoids false positives on unrelated messages. The named failure mode (`invocation_error` in the output channel) makes VS Code string changes observable rather than silent.

**The pre-invoke gate should check run state, not token state.** `hasActivePump?.() === false` correctly rejects tool calls that arrive with no active `/run` handler. When the method is absent (undefined), the gate does not fire — backward-compatible for existing stubs.

**`await pumpTask` in finally is mandatory.** Removing `pump.close()` from the finally block causes the pump processor's `while (!pump.closed)` to hang forever; `await pumpTask` then deadlocks. Mutation 7 confirmed this as an infinite hang (test suite kill), not a mere assertion failure.

**`POST /runs` returns 201 before any tool call.** The handler must be held open by `Promise.race([terminal, cancel, deadline])`, not by awaiting the HTTP POST. The run runs in a background goroutine in Gert Core; the extension must poll or SSE to learn terminal state.

**Empirically unproven:** Whether VS Code accepts `vscode.lm.invokeTool()` called from an awaited continuation (pump processor) inside the handler, versus requiring synchronous handler stack context. Only a live VS Code session settles this. The failure mode, if real, is named: `invocation_error` in the output channel.

**`/arm-mcp` is retired to diagnostic.** The stored token is irrelevant for run invocations. The command remains accessible as a diagnostic to force MCP server discovery (~60s latency on first use), but its response now explicitly states it does not authorize deferred runs.

- **Closed-enum classifiers > regex passthrough:** Allowlist of categories prevents future exception variants from inadvertently exposing provider internals.
- **Extract-to-pure seams for wiring tests:** Host-provided APIs cannot be called in unit tests. Extract the call into a pure function with callback-shaped parameters; test with spies.
- **Static source scans complement runtime mocking:** A spy test proves the helper works, but direct call sites remain unguarded. Reproboundary/rule7 scans source at test-run time to flag unscoped getConfiguration calls.
- **Scoping decisions must be explicit:** Every runtime configuration read is either folder-scoped (context-aware) or window-scoped (global). Document both.
- **Mutation 2 shows self-healing behavior:** The existing redact() call correctly strips the capability secret even when the full raw message is forwarded. Other secrets (bearer token, tool args) still leak and tests still fail — the pattern is working.
- **Non-vacuity guards are critical:** A redaction test that asserts a secret is absent from an empty response always passes. Non-vacuity control proved the scanner inspects a non-empty body.
- **Consumer contracts accumulate silently:** tsg-recommendation appeared in 8 places across 3 files. Full-repo grep required before declaring fixtures clean.
- **toolInvocationToken unavailable outside ChatRequestHandler:** No VS Code API to obtain this token in non-chat context. Bridges calling invokeTool with undefined should classify failures as invocation_token_unavailable, not invocation_error.
- **Reproducibility proof: worktree at commit, not working tree:** Mutation tests must verify in a clean worktree at the target commit to avoid confounding with untracked files.
- **Stale build artifacts corrupt evidence in both directions:** This is the 13th distinct bug class on this engagement. The binding rule applies to all future mutation testing.

## Learnings (2026-08-18 — ken-13, commit 4560d82 — Third Vacuity Occurrence)

**PUMP-8 was vacuous. Forcing one branch means the other branches are untested.**

PUMP-8 (original) forced attempt 1 to throw `Canceled`. That means the only path executed was the Canceled retry path. The attempt-1-success path, the non-Canceled failure path, and the Canceled-retry-failure path were never exercised. A token leak injected on the attempt-1-success path produced 161/161/0/0 — the test could not see it.

**The canary distinction: proving the scanner versus proving observation.**

The "non-vacuity control" in the old test asserted that a crafted string `"token=abc123"` contains `"abc123"`. That proves `includes()` works. It does not prove the scanner ever received a real log line from the leaking path. Those are different claims. A canary over a crafted string is not a substitute for real log output from the path under test.

**Corrective pattern (binding):**

For any function with N distinct execution paths, write N path-specific redaction assertions. Each must:
1. Exercise exactly one path through the function.
2. Assert the token is absent from real output produced by that path.
3. Include a non-vacuity guard that confirms at least one log line was produced (from real execution, not a crafted string).

PUMP-8 was split into PUMP-8a (attempt-1 success), PUMP-8b (non-Canceled failure), PUMP-8c (Canceled retry success), PUMP-8d (Canceled retry failure). Each confirmed as 163/164 before acceptance.

This is the **third occurrence** of a proof targeting the wrong layer on this engagement. See also: closed-enum classifier (first), toolInvocationToken extract-to-pure (second).

## Learnings (2026-08-18 — Layered MCP fix + Usable Chat Run UX, commits 4d5dbb9 + d98cc8b)

**Unreliable ≠ forbidden.** The live-site failure of the cached-token path was treated as proof that the path should be hard-forbidden. The correct reading is that it is unreliable — those are different claims and only one justifies a hard refusal. The fix is to attempt the call, classify the failure with a named category, and surface it — not to refuse pre-emptively and make the armed token dead code.

**Layer-ordering proofs require exercising each layer separately.** LAYER-1 proves layer 1 is selected when pump is active. LAYER-2 proves layer 2 is selected when armed and no pump. PUMP-2 proves layer 3 (no_active_run) fires only when unarmed. The three tests together cover every state combination; any one test alone would miss two states.

**Two redaction paths, not one.** For any feature that handles secrets, identify all paths through which a secret could escape (chat query, log lines, HTTP body, etc.) and inject a real leak on each. Covering only path A (chat query via buildRunChatQuery) left path B (log lines via formatRunStartLog) unproved until the second round. Both now have live mutation proofs.

**workbench.action.chat.open confirmed from VS Code source.** Command ID `'workbench.action.chat.open'` sourced from `CHAT_OPEN_ACTION_ID` exported in `src/vs/workbench/contrib/chat/browser/actions/chatActions.ts` (microsoft/vscode, main branch). Argument shape `IChatViewOpenOptions { query: string; isPartialQuery?: boolean }`. Corroborated by GitHub issue microsoft/vscode#210819. Not in `@types/vscode` — must be sourced from VS Code internals.

**PowerShell `-replace` is unreliable for mutations with backtick-interpolated strings.** Use the `edit` tool for all source mutations and reverts. PowerShell `-replace` with special characters (backticks, `${}`) can silently fail or partially apply, corrupting the test tree. Discovered when the RRES-2 revert partially applied, leaving a residual bug caught only on the next run.



---

## 2026-08-18 — Run Authenticated handoff extraction (commit 0cde638)

**Sprint:** Security-critical redaction proof was vacuous — all QBLD tests operated on `buildRunChatQuery` which cannot receive raw inputs by signature. The production call site (`runAuthenticated()` in extension.ts) was completely uncovered: Cristiano's mutation leaked `collectedInputs` into the chat query and 208/208 tests still passed.

**Fix:** Extracted steps 3-5 of `runAuthenticated()` into `src/runHandoff.ts` — a pure, vscode-free module with injected collaborators for prompting, stashing, query building, and command execution. `runAuthenticated()` becomes a thin wiring shim supplying the real vscode implementations. The security-critical redaction logic now has direct test coverage without requiring a VS Code runtime.

**Tests:** 215/215/0/0 (was 208/208/0/0). 7 new tests: HANDOFF-1..7.
- HANDOFF-3 is the primary leak guard: asserts the chat query does not contain the sentinel secret value.
- HANDOFF-4 is the non-vacuity control: proves inputs flow correctly INTO the store (redaction is not achieved by silently dropping data).
- HANDOFF-5 proves executeCommand is called with the correct command and query.
All tests use the real production `stashPendingRun` and `buildRunChatQuery` collaborators.

**Mutations killed:**
- Cristiano's mutation (leak collectedInputs into query at call site) → HANDOFF-3 fails
- M2: stash `{}` instead of collectedInputs → HANDOFF-4 fails
- M3: suppress executeCommand call → HANDOFF-5 fails

## Learnings

- **Vacuous tests are worse than no tests.** If the function under test cannot receive the sensitive data at all by its signature, testing "it doesn't leak" proves nothing. The test must exercise the actual call site.
- **Non-vacuity controls are mandatory.** Every redaction test needs a paired proof that the data actually flows through (so the redaction isn't achieved by dropping it). HANDOFF-4 is the model: it claims the pending entry via the nonce and asserts the secret is there.
- **DI enables precise surgical testing.** By injecting stashPendingRun and buildRunChatQuery as real collaborators (not stubs), the test exercises the actual production sequence while remaining vscode-free. Stubs would recreate the vacuity problem.
- **Async wrappers fix Thenable/Promise mismatch.** vscode.commands.executeCommand returns Thenable<T>, not Promise<T>. Wrapping with `async (cmd, opts) => { await ... }` at the shim site avoids TS2739 without weakening the interface type.

---

## 2026-08-19 — Runhandoff Extraction (commit 0cde638, follow-up)

**Context:** Ken's security-critical extraction (HANDOFF-1..7 tests, 215/215/0/0) addressed the fourth occurrence of the vacuous-mutation defect class. The original redaction tests (QBLD-1/2/3) targeted `buildRunChatQuery()` whose signature *cannot* receive raw input values. Every assertion was vacuous.

**Coordinator verification from detached worktree at 0cde638:**
- Zero untracked files (including generated files)
- `npm ci` clean
- `npm test` → 215/215/0/0
- Reachability gate: TestCLI_ReachabilityGate passes
- Mutation #1 (Cristiano's original leak at call site): HANDOFF-3 fails as expected
- Mutation #2 (stash `{}` instead of inputs): HANDOFF-4 fails as expected

**Key design judgment verified:** Using REAL production collaborators (stashPendingRun, buildRunChatQuery) not stubs is what makes the test load-bearing. A stub would recreate the vacuity problem — it would return a fixed string regardless of inputs, and the mutation proof would be invisible again.

**Systemic learning recorded:** This is the **fourth occurrence** of the vacuous-mutation defect class on this engagement:
1. Classification validation (Go side, Tess) — unit test on ParseToolFile returns error; endpoint reachable from CLI?
2. AllowedEnvironments ghost field (David) — schema field exists, Tier 0 check passes in unit tests; is the check wired to run.go?
3. vscode-mcp transport classification injection (Ken, Phase 2 prep) — helper function receives classification, never tested at engine.go invocation site
4. runAuthenticated() redaction (Ken, this session) — buildRunChatQuery signature cannot receive secrets; production call site never tested

**Remedy adopted:** Extract call-site logic into a vscode-free DI module and inject REAL collaborators in tests. The extraction pattern makes the sensitive data flow observable and testable without requiring a VS Code runtime.

**Future prevention:** Before marking a redaction proof complete, ask: "Where does the actual production call to this function live? Is that call site tested?" If the function's signature structurally prevents it from receiving the sensitive data, the call site is untested and the proof is vacuous.

