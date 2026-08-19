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

## Key Learnings (2026-08-18)

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
