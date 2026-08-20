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

---

## 2026-08-19 — Systemic Bug Class #16: Chat Command with No Handler Body (5th Occurrence)

**Defect:** The /run chat command was declared in package.json but had no handler implementation in src/extension.ts. The manifest entry remained in place, so VS Code routed all @gert /run requests to the extension, but the handler body fell through isArmCommand() to "Unknown command" for every invocation since the Petals lifecycle port (commit 742e368).

**Why it shipped:** All 169 unit tests passed before this discovery. None exercise the extension.ts handler body, which requires a live VS Code host to invoke. The handler absence was therefore invisible to the test suite.

**Discovery:** Ken's root-cause analysis during the ICM MCP blocker investigation (2026-08-19).

**Impact:** The single most important code path (@gert /run) was non-functional for every user since the Petals port. This is the **5th occurrence** of the vacuity defect class on this engagement — a pattern indicating systematic test-harness/coverage gaps in the extension layer.

**Note:** This specific occurrence is the most costly yet, because it disabled the primary user-facing entry point for a complex feature.

---

## 2026-08-19 — Live ICM MCP `serve --package-map` Startup Unblock

**Issue:** The extension spawned `gert serve --package-map packages/incident-routing.vscode-mcp.package-map.yaml`; that file contained `requires:` entries, and `serve` intentionally rejects non-empty `requires:` package maps before binding the listener.

**Source-grounded finding:** `tool-paths:` is a top-level string list in `schema.ProjectConfig`. In `serve`, those strings are passed directly to `newServingToolRegistry(".", ...)` and `adapter.WireOptions.ExtraToolScanPaths`; relative paths resolve from the process cwd (`c:\One\gert-sqllivesite` in the extension), not from the package-map file.

**Action:** Added `C:\One\gert-sqllivesite\packages\incident-routing.vscode-mcp.serve-package-map.yaml` with `tool-paths: [./packages/incident-routing-vscode-mcp]`, preserving the original `requires:` map for `run`/`plan` and future catalog-capable `serve`.

**Semantic caveat:** This is an unblock, not full equivalence. `tool-paths:` cannot express `^1.0.0` version constraints, package identity/provenance, or `sql-livesite-tsgs` exported runbooks. It only exposes `.tool.yaml` definitions; the two immediate bare tools (`icm`, `tsg-recommendation`) are covered by scanning `incident-routing-vscode-mcp`.

**Verification:** From cwd `C:\One\gert-sqllivesite`, ran `C:\One\OpenSource\gert\gert.exe serve --addr 127.0.0.1:65123 --package-map C:\One\gert-sqllivesite\packages\incident-routing.vscode-mcp.serve-package-map.yaml`. Output: `gert serve: listening on 127.0.0.1:65123`; process remained running after 3 seconds, then was stopped. No ICM tools were invoked.

---

## 2026-08-19 — Serve Catalog Gap Loudness Patch

**Correction:** In current `serve`, a reached `resolve_from: catalog` include is not a partial-catalog silent no-op: `cmd/gert/serve.go` never wires `DynamicIncludeResolver`, so a reached dynamic catalog include fails with no resolver configured. The tactical map still cannot prove TSG exports, but the immediate false-green path is narrower than first feared.

**Startup feasibility:** Pure startup preflight is not honest because serve does not know the requested runbook until `run.start` / `POST /runs`. The valid seam is request-time parse/plan.

**Implementation in `C:\One\OpenSource\gert`:** Added `SERVE-W001` request-time warning for served runbooks with `requires:`; added explicit include step output (`warning`/`stderr`) and `runbook_skipped_reason` when DINC-002 is continued under `on_not_found: continue`.

**Validation:** `go test ./internal/executor ./internal/serve` passed using `C:\Program Files\Go\bin\go.exe`. Rebuilt `C:\One\OpenSource\gert\gert.exe` and verified `gert serve` still starts with the tactical map: `gert serve: listening on 127.0.0.1:65124`.

**Live-read guidance:** Today's tactical-map run can prove live ICM retrieval (`get_icm_incident`) and the current local recommendation path (`recommend_tsg`, currently no-suggestion). It cannot prove `sql-livesite-tsgs` catalog export resolution or GEODR0001 execution. If a future recommendation reaches `invoke_suggested_tsg` under current serve, expect a failure, not a validated TSG run.


## 2026-08-19 — Serve package-map unblock and false-green diagnostics

Don created the tactical serve-compatible SQL Live-Site package map and implemented SERVE-W001 plus DINC-002 loud diagnostics in `ormasoftchile/gert` (commit `23bf406`). Full suite/build verified by coordinator: 84 packages, 62 passed, 0 failed, build clean.

False-green domain note: `invoke_suggested_tsg` uses `resolve_from: catalog` with `on_not_found: continue`; without loud diagnostics, a missing catalog export could look like `icm-tsg-routing-complete` success without TSG execution. This is the sixth occurrence of the engagement's vacuity/false-green defect class. Runtime diagnostics must make skipped dynamic includes explicit in output, not only traces.
