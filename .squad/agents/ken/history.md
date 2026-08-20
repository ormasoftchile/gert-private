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

## 2026-08-19 — gert-vscode F5 bootstrap fixed

- Preserved Cristián's single F5 launch configuration with `preLaunchTask: gert-vscode: prepare debug` and `..\gert` PATH prepend.
- Added conditional npm bootstrap for clean clones: `npm ci` runs when `node_modules` is absent or stale, otherwise skips.
- Added Go resolution for CLI builds: PATH first, then common Windows Go install locations, with clear errors for missing Go or missing sibling `..\gert` repo. Correction: this guards stale inherited process PATH after Go is installed during an editor session, plus genuinely missing Go; it is not fixing a broken machine PATH.
- Verified from the stale-process state: `Get-Command go` was unavailable, the build script exercised the well-known-location fallback to `C:\Program Files\Go\bin\go.exe`, dependencies installed, `npm run compile` produced `out\extension.js`, Go build produced `..\gert\gert.exe`, and `npm test` passed 186/186.
- Did not invoke ICM tools; Extension Development Host launch remains unverified because I cannot press F5 in VS Code from this session.

## 2026-08-19 — serve-compatible package-map setting clarified

- Cristián confirmed F5 bootstrap works live: extension resolved `C:\One\OpenSource\gert\gert.exe` and spawned `gert serve`.
- The remaining startup failure was `serve` rejecting the SQL Live-Site `requires:` map, not binary resolution.
- Confirmed `--package-map` is constructed from the folder-scoped `gert.packageMap` setting via `resolvePackageMapPath`; there is no package-map globbing in the extension path.
- Added extension docs/tests for the serve-compatible `*.serve-package-map.yaml` setting and a regression test proving explicit serve-map selection wins when both run and serve maps coexist.
- Verified with the same argument builder and cwd `C:\One\gert-sqllivesite`: `gert.exe serve --addr 127.0.0.1:<port> --package-map C:\One\gert-sqllivesite\packages\incident-routing.vscode-mcp.serve-package-map.yaml` returned `/preview/` status 200 and stayed running until terminated.
- This only proves serve starts; it does not prove the live run is end-to-end sound because the serve map cannot carry the runbook-exporting `sql-livesite-tsgs` package and the catalog include can still no-op.


## 2026-08-19 — F5 bootstrap and serve package-map handoff

F5 now works from a clean clone. Ken delivered conditional npm bootstrap plus Go fallback resolution for stale process environments (commit `5463e86`) and explicit serve-compatible package-map configuration (commit `f92197a`). Coordinator verified rebuild after deleting both `out/extension.js` and `gert.exe`; Go was absent from the inherited process PATH, proving fallback resolution carried the build.

False-green domain note: the serve-compatible map unblocks listener startup only. It cannot carry `sql-livesite-tsgs` runbook exports. The router's `resolve_from: catalog`/`on_not_found: continue` path is the sixth false-green/vacuity occurrence on this engagement; extension evidence must distinguish server startup from actual TSG execution.
