# ken

## 2026-08-18 — FINAL SUMMARY: Invocation Token Bridge + Pretest Complete

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
