# ken

Current role: Implementation engineer.

## ARCHIVED WORK (2026-08-15 to 2026-08-17)

**Summary:** 13 completed features across dynamic runbook includes, runtime portability (Phase 1), reachability gate, approval enforcement, gert-vscode reproducibility, phase 2 prep, and skip-defect regression guard.

**Key deliverables:**
- Dynamic runbook includes (feature complete, approved)
- Runtime portability Phase 1 (profiles, approval gates, fail-closed defaults)
- Reachability gate infrastructure and 72-vector conformance suite
- Skip-defect regression guard (rule6 CI enforcement chain added to repoBoundary)
- gert-vscode reproducibility (tracked all source/test files, clean checkout builds)
- gert-vscode Phase 2 prep (engines.vscode floor raised to 1.95.0, extension-host test harness added)
- Gert reproducibility (91 untracked files committed, 45 modified tracked files recovered)

**All 2026-08-15 to 2026-08-17 learnings:** Available in session log `.squad/log/2026-08-18T22-23-36Z-vscode-live-blockers.md` and decisions archive.

---

## 2026-08-18 — Live-Site Blockers: Config Scoping + Registry Parser (Commit 1cd7542)

**Feature:** Two fixes for SQL Live-Site operations blockers reported after Rev 3. No cross-repo coupling; all tests hermetic.

**Outcome:** ✅ COMPLETE — commit 1cd7542, tests 127/127 pass. Verified from `git worktree add --detach`. All four mutation proofs measured.

### Defect 1: Active-Project Settings Not Used

`spawnServer()` was calling `vscode.workspace.getConfiguration('gert')` with no resource URI. In a multi-root workspace this reads from `workspaceFolders[0]`'s scope, not from the active runbook's folder. SQL Live-Site's folder had `gert.packageMap` set; folder 0 did not. Result: `--package-map` was silently omitted from argv.

**Fix:** Exported `GetScopedSetting` type and `readGertSpawnConfig(getSetting)` from `serverLaunch.ts`. `spawnServer()` creates a getter scoped to `vscode.Uri.file(runbookPath)` and passes it to `readGertSpawnConfig`. Both `binaryPath` AND `packageMap` are read through this scoped getter.

**Judgement on binaryPath:** YES, legitimately folder-scoped. Different projects in a multi-root workspace may build different local gert binaries. Reading from the wrong scope silently uses the wrong binary.

**Coordinator verdict:** ACCEPTED (independent verification confirmed parity).

### Defect 2: Registry Parser Rejects Real Tool YAML

`buildRegistryFromDir()` only handled flat `name:`, sequence-form `actions:`, and `transport.mode`. Real consumer files use `meta: { name: icm }`, which silently produced zero registry entries → `tool_not_found`.

**Go-core confirmation (tool.go, manually verified):**
- Axis 1: `ToolDef.UnmarshalYAML` copies `meta.Name` to `Name` when non-empty.
- Axis 2: `decodeToolActions` handles `SequenceNode` (canonical) and `MappingNode` (legacy, key IS action name).
- Axis 3: `TransportConfig.UnmarshalYAML` copies `Mode` to `Type` when `Mode != ""`; callers key on `Type`.

**Fix:** `buildRegistryFromDir` now mirrors all three axes. `effectiveMode = transport.mode ?? transport.type`.

**Removed:** `tsg-recommendation.tool.yaml` and all its test assertions (SQL LS consumer contract must not appear in platform fixtures per squad decisions; forced revert from both repos).

**New tests (5):** consumer ICM exact fixture (tmpdir), mapping-form actions, legacy transport.type, meta.name precedence, drift guard.

**Coordinator verdict:** ACCEPTED (independent verification confirmed).

---

## 2026-08-18 — Defect 1 Wiring Test (Commit fadc2fa)

**Task:** Make the vscode.Uri.file(runbookPath) resource argument at serverManager.ts lines 50 + 80 load-bearing under test.

**Root cause of prior rejection:** The previous mutation changed readGertSpawnConfig to always return empty — a pure-helper mutation. The test exercised only the pure helper's input/output behavior, not the wiring. Deleting the resource URI from the getConfiguration call left the suite green.

**Fix approach:** Extracted `makeScopedGetter(runbookPath, getConfigurationFn)` as a pure exported function in serverLaunch.ts. It calls `getConfiguration('gert', { fsPath: runbookPath })` and returns a GetScopedSetting. serverManager.ts is a one-line caller. Unit tests spy on the getConfigurationFn argument and assert `resource.fsPath === runbookPath`.

**Additional scoping fixes:**
- `serverUrl` and `autoStartServer` in `ensureRunning()` scoped to `vscode.Uri.file(runbookPath)`.
- `binaryPath` in `extension.ts` previewProse and validateInputs scoped to the active runbook path.
- `mcpBridge.toolNameOverrides` at activation intentionally left unscoped (read before any runbook is open).

**Mutation proof:** Deleting the resource argument from makeScopedGetter → test fails: "makeScopedGetter must pass runbookPath as resource.fsPath."

**Result:** 129 total / 129 pass / 0 fail / 0 skip (clean worktree, compile=0).

**Coordinator finding:** Residual untested adapter lambda (resolved by rule7 work below).

---

## 2026-08-18 — Static Source Guard: repoBoundary/rule7 (Commit 08de611)

**Task:** Add a static source guard so that dropping a resource URI from any `getConfiguration(` call in `src/` is caught by the test suite.

**Problem:** Even after `makeScopedGetter` was made load-bearing (commit fadc2fa), the two direct `vscode.workspace.getConfiguration` call sites in `serverManager.ts` (lines 50 and 80) were unguarded. Mutating either to drop the resource argument produced 129/129 (suite stayed green).

**Fix:** Added `repoBoundary/rule7` to `test/repoBoundary.test.js`. The rule:
- Scans all `src/**/*.ts` files from disk (new files auto-trip it)
- Fails on any `getConfiguration(` call site that neither has a second argument nor a `// window-scoped: <reason>` marker in the preceding comment block
- Asserts `totalCallSites > 0` to prevent false-green on empty scan

**Call sites verified:** 6 real sites (7 raw matches, 1 is JSDoc comment).

**Marker applied:** `extension.ts:76` (`mcpBridge.toolNameOverrides`) — legitimately window-scoped (read at activation, before any runbook is open).

**Mutation proofs (all three measured):**
- Drop resource from serverManager.ts:80 → rule7 fires
- Drop resource from serverManager.ts:50 → rule7 fires
- Remove window-scoped comment from extension.ts:76 → rule7 fires

**Result:** 130 total / 130 pass / 0 fail / 0 skip (clean worktree, compile=0). All verified from `git worktree add --detach`. gert-vscode fast-forward from f1e5c51 to 424ce26.

**Coordinator verdict:** ACCEPTED (independent verification confirmed).

---

## Key Learnings (2026-08-18)

- **Mutating a pure helper does not cover its caller's wiring.** The spy test made the helper load-bearing, but direct call sites remained unguarded. A static source scan is the correct complement when runtime mocking is impractical.
- **Extract-to-pure is the right seam for wiring tests.** Host-provided modules cannot be called in unit tests without an extension host. Extract the call into a pure function accepting a callback-shaped parameter; test with a spy. The caller's delegation line becomes compile-time checkable.
- **Scoping decisions must be stated, not inferred.** Every `getConfiguration('gert')` call must be examined: is a runbook path in scope? If yes, it's a candidate for folder-scoping. Leaving it unscoped is an oversight unless explicitly documented (e.g., "window-scoped: read at activation, no runbook open").
- **A working-tree build is not a reproducibility proof.** Mutation proofs must be done in the working tree (for clarity), but final verification must use a worktree at the commit to avoid confounding with untracked files.
- **Consumer contracts accumulate silently.** The `tsg-recommendation` consumer name appeared in 8 places across 3 files. A full-repo grep for the consumer name is required before declaring the fixture clean.
