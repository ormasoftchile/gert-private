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
