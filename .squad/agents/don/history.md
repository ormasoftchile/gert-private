Don — History Summary

## Overview

**Stream:** Backend transport and authentication. Implemented MCP HTTP transport, verified Phase 1B auth gaps, delivered managed-identity provider and credential-leak assertions. Fixed dry-run regression and serve security issue. Phase 2 Rev 3: serve --package-map + declarative vscode_input adaptation. TSG binding logical-contract-only (no transport).

**Key Contributions:**
- MCP HTTP transport implementation (Phase 1A)
- Phase 1B Item 1: Managed-identity auth provider
- Phase 1B Item 6: Credential non-leakage assertion suite
- Live defect: dry-run profileless fail-fast regression (commit d19f933)
- Live defect: serve all-interface bind without auth (commit a6b1106)
- Phase 2 Rev 3 Deliverable A: serve --package-map (commit 4da94b2)
- Phase 2 Rev 3 Deliverable B: declarative vscode_input adaptation (commit 4da94b2)
- TSG binding logical-contract-only (no transport, six args documented)

---

## Team Update (2026-08-19T16:13:34Z)

**From Scribe:** Ken completed removal of `/probe-token` command from gert-vscode. MCP probe deletion resolves VS Code session-termination risk. Root cause: 4 unconditional `vscode.lm.invokeTool` calls exhausted MCP restart budget. Probe measurement was complete (T1 failed = token lifetime not the variable). Decisions merged to decisions.md.

---

## ARCHIVED WORK (2026-08-15 to Early 2026-08-18)

**Summary:** Delivered MCP HTTP transport, managed-identity auth provider, credential assertions, parity-script fix, and TSG logical contract.

**Key features archived:**
- Phase 1A MCP HTTP transport with loopback validation and credential sweeping
- Phase 1B managed-identity auth provider with IMDS integration
- Credential non-leakage assertion suite (5 tests, 2 mutation controls)
- CLI integration test fix: load-bearing --package-map and --profile flags
- Synthetic contract parity proof (native + mcp-http bindings)
- Parity script bidirectional validation (MISSING-TESTDATA / ORPHAN-TESTDATA guards)
- TSG filename rename and parity tracking

**All archived learnings:** Available in session logs and decisions archive.

---

## 2026-08-18 — Dry-Run Gate Scope Fix (Commits d19f933, a6b1106, 06b0a83)

**Issue:** Phase 1B fail-fast gate refused profileless non-interactive execution on all modes, including dry-run. This broke VS Code extension's "Validate Inputs" command and CI validation callers that use `gert dry-run` without `--profile`.

**Rationale discovery:** The gate's stated reason ("without profile, TerminalApprovalGate blocks stdin") was empirically wrong. `buildApprovalGate` installs TerminalApprovalGate only when `attended==true` (TTY context). In non-TTY it installs NoOpApprovalGate, which never blocks. The real rationale: profileless real execution lacks approval scope declaration. For dry-run, `DryRunExecutorRegistry` replaces all executors with no-ops; the governance gap doesn't apply.

**Decision:** Scope gate to `mode == engine.RunModeReal` only. Phase 1B acceptance criterion preserved exactly. Dry-run exempt.

**Implementation:**
- Check added to `cmd/gert/run.go` conditional on `mode == engine.RunModeReal`
- Commit `d19f933`

**Consumer impact:** None for real-execution callers. VS Code extension and CI validation using dry-run now work correctly.

---

## 2026-08-18 — Serve Security Fix (Commits a6b1106, 06b0a83)

**Issue:** `gert serve` defaulted `--addr` to `:7778` (all interfaces). VS Code extension spawned `gert serve --addr :${port}` without auth flags. Made Gert server reachable from the local network with no authentication.

**Decision:**
1. Change `--addr` default to `127.0.0.1:7778`. VS Code extension probes `127.0.0.1:0` and uses localhost, so unaffected.
2. Add startup safety check in `runServe`: if bind address is non-loopback AND no auth configured, refuse with actionable error.
3. Operator who needs public bind must add `--auth-token` or `--auth-jwt-secret`.
4. No opt-out flag. Fail-closed is the correct security posture.

**Implementation:**
- Check added to `cmd/gert/serve.go` before `net.Listen`
- Commit `a6b1106`

**Consumer impact:** Anyone using non-loopback without auth must now add auth (correct behavior).

---

## Key Learnings

- **Stated rationale must be empirically verified.** The fail-fast gate's documented reason was wrong; the actual gate behavior depends on attended/TTY, not just profile presence. Verify implementation against stated rationale.
- **Dry-run is not a real execution context.** Governance gaps in a mode that executes nothing are not real governance failures. Scope checks accordingly.
- **Default security posture:** fail-closed, not fail-open. Public binds default to localhost; non-loopback requires explicit auth configuration.
