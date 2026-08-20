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

### 2026-08-19 — ICM MCP incident severity corrected

The previous model for the `/probe-token` incident understated the failure mode. Cristián reported that live ICM MCP test/diagnostic invocations corrupted the identity broker on the previous machine; the machine is now broken and retired. This was not just VS Code MCP restart-budget exhaustion and was not recoverable by Reload Window or reboot.

Work moved to `CPC-crist-LKO5U`, where `icm-mcp` starts. The prior `401 status sending message to https://icm-mcp-prod.azure-api.net/v1/` blocker was machine-local to the retired box and is no longer active. Treat "never burn live ICM invoke attempts on diagnostics" as hard law: zero-invoke paths (`/arm-mcp`) or mocks only; stop on first failure.
