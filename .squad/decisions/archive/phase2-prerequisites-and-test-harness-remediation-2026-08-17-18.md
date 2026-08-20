# Archived Decisions Block — Archived Phase 2 prerequisites + test-harness remediation

**Effort slug:** phase2-prerequisites-and-test-harness-remediation  
**Source:** .squad/decisions.md lines 2512–3350 at archival time  
**Archived at:** 2026-08-19T17:08:50-07:00  
**Ruling:** .squad/decisions.md live entry `2026-08-19 — decisions-ledger-archival-ruling`  
**Payload SHA-256:** 55d42796a1f355f083f578ad2d10d2a42a92066679aedcae471625db15a640bf  
**Payload bytes:** 44594

---

# Decision: dry-run gate scope and serve loopback default

**Author:** Don  
**Date:** 2026-08-17  
**Status:** Implemented (commits d19f933, a6b1106, 06b0a83)  
**Requested by:** Cristiano (ormasoftchile)

---

## Decision 1: Profileless non-interactive fail-fast is scoped to RunModeReal only

**Context:** Phase 1B added a fail-fast gate at `cmd/gert/run.go` that refused profileless non-interactive execution with exit code 2. The gate applied to all modes dispatched through `runWithMode`, including `dry-run`. This broke the VS Code extension's "Validate Inputs" command (extension calls `gert dry-run` via `execFile`, no TTY, no profile) and any CI validation caller.

**Empirical finding:** The gate's stated rationale ("without a profile the engine installs TerminalApprovalGate which blocks stdin") is factually wrong. `buildApprovalGate` (`internal/adapter/wire.go:330`) installs `TerminalApprovalGate` only when `attended==true` (i.e., TTY context). In non-TTY (CI) context it installs `NoOpApprovalGate`, which never blocks, regardless of profile presence. The correct rationale for the real-execution gate is governance completeness: profileless real execution lacks declared approval scope, context, and operator identity. For dry-run, `DryRunExecutorRegistry` (`internal/adapter/wire.go:121`) replaces all executors with no-ops; no step is executed and the governance gap doesn't apply.

**Decision:** Scope the gate to `mode == engine.RunModeReal`. The Phase 1B acceptance criterion (profileless non-interactive `gert run` fails immediately with exact error+fix text) is preserved exactly. Dry-run is exempt.

**Consumer impact:** None for real execution callers (no change). VS Code extension and CI validation callers using `gert dry-run` without `--profile` now work correctly.

---

## Decision 2: gert serve default binds loopback; non-loopback requires auth

**Context:** `cmd/gert/serve.go` defaulted `--addr` to `:7778` (all interfaces). The VS Code extension spawned `gert serve --addr :${port}` without auth flags. This made the Gert server reachable from the local network with no authentication whenever the graph preview was open.

**Decision:**
1. Change `--addr` default to `127.0.0.1:7778`. The VS Code extension probes `127.0.0.1:0` for a free port and communicates via `localhost`, so it is unaffected.
2. Add a startup safety check in `runServe`: if the resolved bind address is non-loopback AND no auth is configured, refuse with a clear actionable error. Operator who genuinely needs a public bind must configure `--auth-token` or `--auth-jwt-secret`.
3. Loopback + no auth always proceeds (local development, extension).
4. No opt-out flag is added — fail closed is the correct security posture. An operator who needs public bind simply adds auth.

**Implementation note:** The `net.Listen` call in `internal/serve/server.go:165` is NOT patched. It faithfully binds whatever addr it is handed. The check is in the CLI layer (`cmd/gert/serve.go`) before any network operation.

**Consumer impact:** Anyone using `gert serve` with a non-loopback addr and no auth (e.g., `gert serve --addr 0.0.0.0:7778`) must now add auth. This is the correct behaviour — they were previously running an unauthenticated public server.

---

# Decision: Load-Bearing Flag Standard for CLI Integration Tests

**Date:** 2026-08-17
**Author:** Don (Backend/Tool-Runtime)
**Inbox target:** decisions.md (team decisions)

## Context

Phase 1B Item 4 CLI integration test initially had two flags (`--package-map`, `--profile`) that were empirically confirmed no-ops: removing either flag left the test green. This is structurally equivalent to a parsed-but-unreachable field — the exact failure class the reachability gate was built to catch.

## Decision

A CLI integration test that exercises `--package-map` or `--profile` must be structured so that removing either flag changes the exit code:

1. **`--package-map` is load-bearing** when the project config (`requires:`) has NO entry for the package under test. The tool is resolved only through the package-map override. Without the flag, PKG-011/PKG-001 fires → exitValidation.

2. **`--profile` is load-bearing** when the tool definition's static transport URL is a placeholder that cannot serve real requests (e.g., `https://127.0.0.1/` with no server), and the profile's endpoint override redirects to a live mock server. Without the flag, the transport fails to connect → exitFailure.

Pointing `--package-map` at a different directory that produces the same output proves only that flag parsing works, not that resolution works. This is insufficient.

## IMDS Injection Seam

Added `SetIMDSEndpointForTest(endpoint string) func()` to `internal/tool/auth_managed_identity.go` (production file, not test file) to allow `cmd/gert` integration tests to redirect managed-identity IMDS calls to a mock server. This seam:
- Uses a package-level `imdsEndpointOverride string` var
- Must not be called from `t.Parallel()` tests
- Is the correct pattern when the type under test is constructed inside `runRun()` with no injection seam accessible from the test package

## Files Changed (commit d53a45f)

- `internal/tool/auth_managed_identity.go` — added `SetIMDSEndpointForTest`
- `cmd/gert/synthetic_contract_integration_test.go` — rewrote main composition test to use mcp-http binding with load-bearing flags

---

# Decision: Synthetic Contract Proof — Split CLI and Runtime Tests

**Author:** Don (Backend/Tool-Runtime)
**Date:** 2026-08-17
**Context:** Phase 1B Item 4 — Dual-Binding Mechanism Proof
**Commit:** 5c0c002

---

## Decision

The mcp-http binding proof is split across two test layers, not consolidated into a single CLI integration test.

- **CLI integration test** (`cmd/gert/synthetic_contract_integration_test.go`): exercises `--package-map` AND `--profile` together in a single `runRun()` execution using the **native** binding. Profile carries a `provider: managed-identity` override for the tool — which native transport correctly ignores.

- **Runtime test** (`internal/tool/synthetic_contract_proof_test.go`): exercises the **mcp-http** binding directly via `MCPHTTPTransport + TokenGate + ManagedIdentityAuthProvider` with mock httptest servers (both IMDS and MCP server). Uses the package-internal `imdsHTTPClient` injection seam.

---

## Rationale

`ValidateTransportConfig` enforces HTTPS for the static URL in mcp-http tool YAML files. `httptest.NewServer` produces plain HTTP. The workaround (static URL = `https://127.0.0.1/` + profile endpoint override to `http://127.0.0.1:PORT`) works at the runtime layer but introduces fragility in the CLI path (profile must be carefully crafted to pass PLAN-013 while pointing at the test server). It also couples the managed-identity token acquisition to the full CLI execution path where `DefaultToolRuntime` creates a `ManagedIdentityAuthProvider` via `NewAuthProvider()` with no injection seam for the IMDS client.

The two-layer split is cleaner:
- CLI layer: proves the flag composition mechanism works (selection + parameterization).
- Runtime layer: proves the mcp-http binding + managed-identity chain works (IMDS → token → TokenGate → Authorization header).

Neither layer is weaker than the other — they test different invariants.

---

## Parity Harness Compatibility

The fixture is designed so that Tess's `pkg/contractparity` harness can compare both bindings without changes to the fixture. Required interface:

1. `writeOpsSyntheticNativePackage(t, root, opsMockBin string)` — writes native package files to a temp dir.
2. `newOpsSyntheticMCPServer(t)` — starts a test MCP server implementing the ops-synthetic contract.
3. Action names: `"status-query"` and `"pattern-search"`; tool name: `"ops-synthetic"`.
4. Test inputs: `target="api-gateway"` for status-query; `query="FOUND:pattern"` and `query="no-match"` for pattern-search.
5. The harness verifies both bindings return the same JSON shape (field names and types), not byte-identical values.

`writeOpsSyntheticNativePackage` is in `cmd/gert` (package `main`, test-only), so the parity harness will need its own copy or a shared helper moved to a testutil package when wired.

---

## No Change Required to Existing Tests

`TestRun_PackageMap_RealVsMockBinding` (`cmd/gert/packagemap_integration_test.go`) deliberately asserts that outputs DIFFER between bindings. The ops-synthetic tests are additive and do not conflict.

---

# Decision: TSG Unresolved Provider Binding (VSCODE-003)

> ⚠️ **VOID — REVERTED, OUT OF SCOPE.**
> This work was reverted at the user's explicit order. The SQL Live-Site consumer contract must never live in Gert.
> Reverting commits: `gert` `18093a5`, `gert-private` `f3ad30e`.
> Do not re-implement. See Decision 4 in `# Decision: VS Code Extension Config Scoping and Registry Parse Axes` which confirms this revert and establishes the rule going forward.

**Date:** 2026-08-18  
**Author:** Don (Core Dev, Gert Core)  
**Requested by:** Cristiano  
**Status:** ~~Implemented — awaiting merge gate~~ **VOID — reverted (gert 18093a5, gert-private f3ad30e)**

---

## Context

SQL Live-Site Operations supplied the TSG binding's LOGICAL contract (six
args: `incident_id`, `title`, `service`, `environment`, `logical_server`,
`database`) but could NOT supply the PROVIDER contract (VS Code MCP registered
tool name + parameter schema), because the tool was not present in
`vscode.lm.tools` during the live session.

The deliverable had `vscode_tool: tsg-recommendation-recommend` — a guessed
name. This is the same defect class as `icm-get-incident` caught in Rev 2.

---

## Decision

**Do NOT invent the provider tool name.** Mark the binding explicitly unresolved
with a new schema field `provider_unresolved: true` on `TransportConfig`.

**Fail closed at scan time (VSCODE-003).** `ValidateTransportConfig` in
`internal/tool/validate_transport.go` rejects any `vscode-mcp` tool with
`provider_unresolved: true` during `ParseToolFile`, before the tool enters the
registry. This is the earliest achievable fail point in the transport chain.

**Error text is unambiguous:**
> This is NOT a "tool not found" error: nobody has told us the real
> vscode.lm.tools name yet.

This distinguishes the unresolved-contract state from MCP server downtime.

---

## Rationale

**Why scan time, not runtime?**  
The prior vscode_input fail-closed checks (VSCODE-001, VSCODE-002) also fire
at scan time via `ValidateTransportConfig` → `ParseToolFile`. This is
consistent with the established pattern. Runtime would allow the tool into the
registry and would reach the bridge — a worse failure mode.

**Why not just leave vscode_tool absent?**  
An absent `vscode_tool` is valid and currently means pass-through (logical arg
names forwarded to the provider). With an unknown provider schema, pass-through
is a silent-wrong-args hazard. `provider_unresolved: true` makes the intent
explicit and forces the error.

**Why a schema field and not a comment?**  
Comments are not machine-checked. A schema field enforced in the validation
chain cannot be silently ignored, and the reachability gate ensures the check
cannot be removed without a test failing.

---

## What was NOT done

The provider tool name and parameter schema were NOT invented. No placeholder,
plausible guess, or synthetic example appears in any deliverable or test
fixture that could be mistaken for a real registered name.

The reachability probe uses the synthetic name `tsg-unresolved-probe` —
obviously non-real, never in a deliverable.

---

## Resolution path

When SQL Live-Site Operations confirms the provider contract:
1. Set `vscode_tool:` to the real registered name.
2. Add `vscode_input:` mappings (provider key → logical arg).
3. Remove `provider_unresolved: true`.
4. Update `TestDeliverableContracts_TSGRecommendation` to assert ParseToolFile
   SUCCEEDS and check outputs (current test asserts failure — must be inverted).
5. Sync `gert/internal/tool/testdata/` with the deliverable.

---

## Files changed

**gert (branch: feature/tsg-unresolved-provider-binding, commit 1824af7):**
- `pkg/schema/tool.go` — `ProviderUnresolved bool` on TransportConfig
- `internal/tool/validate_transport.go` — VSCODE-003 check
- `internal/tool/testdata/tsg-recommendation.vscode-mcp.tool.yaml` — updated
- `internal/tool/deliverable_contract_test.go` — updated
- `cmd/gert/reachability_registry_test.go` — new entry
- `cmd/gert/reachability_probes_test.go` — new probe

**gert-private (commit b598128):**
- `deliverables/phase2/tsg-recommendation.vscode-mcp.tool.yaml` — updated

---

# Decision: VS Code Extension Config Scoping and Registry Parse Axes

**Date:** 2026-08-18  
**Author:** Ken  
**Commit:** 1cd7542 (gert-vscode main)  
**Context:** SQL Live-Site live blockers after Rev 3

---

## Decision 1: binaryPath is folder-scoped (same as packageMap)

`serverManager.ts spawnServer()` now reads BOTH `gert.binaryPath` AND `gert.packageMap` through a `GetScopedSetting` callback scoped to `vscode.Uri.file(runbookPath)`.

**Judgement on binaryPath:** YES, folder-scoped. In a multi-root workspace, different projects may declare different local gert binary builds. Reading `binaryPath` from the wrong scope silently invokes the wrong binary for the active project. This is the same class of defect as the `packageMap` bug that caused the live-site incident.

**What is NOT folder-scoped:** `serverUrl` and `autoStartServer` (read in `ensureRunning()`, not `spawnServer()`). These are global user preferences — a user's choice to auto-spawn a server or point at an external URL applies regardless of which project folder is active.

---

## Decision 2: GetScopedSetting as the injectable bridge

`GetScopedSetting = (key: string, defaultValue: string) => string` is the minimal injectable type for unit-testing scoped config reads without an extension host. It has no vscode dependency and is tested via two mock instances (one simulating scoped-folder settings, one simulating the wrong-scope fallback). The mutation that makes this test meaningful: change `readGertSpawnConfig` to ignore its `getSetting` argument → the multi-root test fails.

---

## Decision 3: Three Go-core parse axes, permanently mirrored

`buildRegistryFromDir` must mirror THREE axes from `tool.go`:

| Axis | Go source | TypeScript implementation |
|------|-----------|--------------------------|
| meta.name wins over flat name | `ToolDef.UnmarshalYAML`: `if raw.Meta.Name != "" { t.Name = raw.Meta.Name }` | `meta?.name` overwrites `def.name` when non-empty |
| Sequence OR mapping actions | `decodeToolActions`: `SequenceNode` vs `MappingNode` | `Array.isArray` → sequence; `typeof === 'object'` → mapping |
| transport.mode wins over transport.type | `TransportConfig.UnmarshalYAML`: `if t.Mode != "" { t.Type = Transport(t.Mode) }` | `effectiveMode = transport.mode ?? transport.type` |

**Drift guard:** `test/toolDefinitionRegistry.test.js` contains a `defect2-drift-guard` test that asserts all six shape-matrix entries are present in the registry. Any future removal from `buildRegistryFromDir` triggers a failure with the message `shape "X" must produce registry key "Y"`. This test must be updated whenever a new axis is added to core.

---

## Decision 4: Consumer contracts must not appear in platform test fixtures

`tsg-recommendation` fixture and all its assertions were removed from gert-vscode as part of this work. This is the third time this contract has been reverted (gert 18093a5, gert-private f3ad30e, now gert-vscode 1cd7542). The `icm/get-incident` fixture is the sole consumer reference permitted — it was explicitly supplied by the consumer as the required regression fixture.

**Rule going forward:** Platform test fixtures use neutral names (`ops-*`, `alpha`, `fallback-tool`, etc.). A consumer's exact YAML is acceptable only when the consumer explicitly supplies it as the required regression vector in the task description and it is used verbatim without augmenting with their field names.

---

# Decision: Static source guard for getConfiguration scoping

**Date:** 2026-08-18  
**Author:** Ken  
**Commit:** 08de611 (gert-vscode main)

## Decision

Add `repoBoundary/rule7` to `test/repoBoundary.test.js`: every `getConfiguration(` call in `src/**/*.ts` must either supply a second resource argument or carry a `// window-scoped: <reason>` comment in the immediately preceding comment block.

## Context

After `makeScopedGetter` was extracted and its spy test made load-bearing (commit fadc2fa), the two direct `vscode.workspace.getConfiguration` call sites in `serverManager.ts` (lines 50 and 80) were still unguarded by any test. Mutating either site to drop the resource URI produced zero test failures. The originally reported live defect (multi-root workspace using the wrong folder's settings) was fully reintroducible from the published commit.

Module mocking of `vscode` is impractical under plain `node --test` because `vscode` is host-provided and unresolvable without `--experimental-test-module-mocks` — a poor trade for a two-line guard.

## Alternative considered: expand the spy test

A second spy in `serverManager.test.js` could have directly tested the two call sites. This was rejected because `vscode.workspace.getConfiguration` is a host-provided API that cannot be imported in a unit test environment. The extraction pattern (`makeScopedGetter`) works precisely because the API is passed in; it cannot be applied to the call sites themselves without either mocking the entire `vscode` module or restructuring both call sites into extractable helpers, which would be disproportionate.

## Why static scan is the right tool here

- No runtime dependency — scans source text, no host environment needed.
- Self-updating — new `src/` files auto-trip the rule without any allowlist change.
- Zero false negatives — every `getConfiguration(` site must be either scoped or explicitly justified with a reason comment.
- Consistent with existing repo style — `repoBoundary.test.js` already guards structural rules at the source level (cross-repo paths, skip declarations, orphaned files).

## Scope of the rule

- Searches `src/**/*.ts`; ignores comment-only lines (leading `//` or `*`).
- Resource-scoped: detected by a comma after `getConfiguration(` at top-level parenthesis depth (same line or next line).
- Window-scoped: detected by `// window-scoped:` anywhere in the contiguous comment block immediately preceding the call line.
- `totalCallSites > 0` assertion prevents false-green on an empty scan.

## Window-scoped exception applied

`extension.ts:76` — `mcpBridge.toolNameOverrides`. Read at extension activation before any runbook is open. There is no resource to scope to at this point; window-scope is the only correct choice. Marker text records the reason in source for future reviewers.

---

# Decision: Use makeScopedGetter as the testable seam for vscode config scoping

**Date:** 2026-08-18  
**Author:** Ken  
**Scope:** gert-vscode — serverLaunch.ts / serverManager.ts

## Decision

`vscode.workspace.getConfiguration` calls that need folder-scoping MUST go through
`makeScopedGetter(runbookPath, getConfigurationFn)` (exported from `serverLaunch.ts`)
rather than being inlined into `serverManager.ts` or `extension.ts`.

## Rationale

`vscode.workspace.getConfiguration` cannot be called in unit tests without an extension
host. Inlining the call makes the resource URI argument (e.g.
`vscode.Uri.file(runbookPath)`) invisible to any test that can run under `node --test`.

`makeScopedGetter` accepts a `GetConfigurationFn` callback. Unit tests pass a spy that
records `resource.fsPath` and assert it equals the runbook path. Deleting the resource
argument from `makeScopedGetter` fails the spy test. The remaining delegation in
`serverManager.ts` is compile-time safe: the `GetConfigurationFn` signature requires
`resource`, so dropping it is a TypeScript error.

## Scoping judgment (documented decisions)

| Setting | Location | Scoped? | Reason |
|---|---|---|---|
| `binaryPath` | serverManager.ts spawnServer | YES | per-folder builds |
| `packageMap` | serverManager.ts spawnServer | YES | per-folder map path |
| `serverUrl` | serverManager.ts ensureRunning | YES | per-folder external server |
| `autoStartServer` | serverManager.ts ensureRunning | YES | per-folder spawn opt-out |
| `binaryPath` | extension.ts previewProse | YES | runbook path in scope |
| `binaryPath` | extension.ts validateInputs | YES | runbook path in scope |
| `mcpBridge.toolNameOverrides` | extension.ts activate | NO | read at activation, no runbook open |

## Binding rule going forward

Any new `getConfiguration('gert')` call in a function that has a `runbookPath` in scope
MUST use `makeScopedGetter` (or equivalent). Leaving it unscoped is an oversight unless
the call site pre-dates any runbook being open (activation only).

---

# Ruling: Reproducibility Fix — Committing 91 Untracked Build Inputs

**Date:** 2026-08-17
**Author:** Ken
**Status:** SHIPPED — final commit `eeb9d66`

## Problem

`main` was not reproducible from a clean checkout. Two root causes:

1. **91 untracked source files**: entire packages (`pkg/pkgcatalog`, `pkg/pkgpath`,
   `pkg/semver`, `pkg/pkgdrift`), production source files (`include_closure.go`,
   `auth_gate.go`, `mcp_http.go`, etc.), test files, and conformance data — all
   present in the working tree but never committed to git.

2. **45 modified tracked files not committed**: in-flight changes to `pkg/schema/steps.go`,
   `pkg/trace/event.go`, `pkg/engine/validated_plan.go`, and many others were present
   in the working tree but not in git. The working tree built because of this; a
   clean checkout failed with symbol-not-found errors.

## Classification of files

- **91 untracked files**: all committed except one.
- **1 deliberately NOT committed**: `examples/simple-health-check/examples.code-workspace`
  — a VS Code workspace file referencing local absolute paths including
  `../../../../gert-sqllivesite` (a private local repo). Added `*.code-workspace`
  to `.gitignore` instead.
- **No secrets found**: the `token =` / `Authorization:` / `secret =` pattern matches
  were all variable names, doc comments, or synthetic sentinel values (`TESS_SENTINEL_TOKEN_D42E9B1C`).
  The `tools/icm.tool.yaml` URL and scope are public endpoint/scope identifiers, not credentials.

## Commits (in order)

| Hash | Contents |
|---|---|
| `90435b0` | pkg/semver, pkg/pkgpath, pkg/pkgdrift, pkg/pkgcatalog (missing packages) |
| `97182d6` | pkg/schema additions (enum, packagelock, packagevalidate, projectconfig, toolpackage) |
| `f3f8129` | internal/tool additions (MCP HTTP, auth, overlay) |
| `29de6e5` | internal/adapter, executor, planner, parser, replay, serve additions |
| `cce0720` | internal/conformance test data + engine enum test |
| `603ee7e` | cmd/gert CLI source and integration tests |
| `c3512ab` | pkg/errkit, pkgsubst, preview, run, trace additions |
| `37aef73` | tools/icm.tool.yaml + .gitignore *.code-workspace |
| `41165f0` | internal/conformance/dyninclude_conformance_test.go (missed in batch) |
| `57c2f7a` | Modified: pkg/schema/steps.go, runbook.go, go.mod |
| `28bca1b` | Modified: pkg/trace/event.go, pkg/engine/validated_plan.go |
| `d737875` | Modified: internal/tool/mcp.go, native.go, runtime.go |
| `17ae2f2` | Modified: internal/adapter, executor, governance, parser, planner, replay, serve |
| `eeb9d66` | Modified: pkg/*, cmd/*, examples, README, schemas |

## Verification

From `git worktree add --detach C:\One\OpenSource\_gert_verify HEAD`:
- `go build ./...` → exit 0
- `go test ./...` → exit 0, 63 packages all `ok`
- `TestCLI_ReachabilityGate` → PASS from clean checkout
- `internal/tool/auth_gate.go` tracked: confirmed
- `cmd/gert/packagemap_integration_test.go` tracked: confirmed
- `git ls-files --others --exclude-standard` → empty (zero untracked source files)

## Root cause

The team was committing feature work in the production files (tracked, modified)
but leaving the new files it introduced (untracked) on disk. The working tree
always built because Go uses the filesystem. CI never caught this because CI
does a clean checkout — but CI was not being run against `main` at the time of
the failures.

The fix requires no process rule (we have the reachability gate now). The
structural discipline is: every PR must pass `go build ./...` and `go test ./...`
from a clean checkout of its branch.

---

# ken-skip-defect.md

**Date:** 2026-08-18  
**Author:** Ken  
**Requested by:** Cristiano  
**Branch:** `ken/hermetic-regression-tests` (gert-vscode)  
**Commits:** `fc273b6` (rev 1 — rejected), `0b98842` (rev 2 — rejected), `424ce26` (rev 3 — current)  
**Status:** Rev 3 delivered — awaiting Cristiano's merge gate

---

## Rev 1 Rejection Reason

`test/integration/cli.test.js` was not reached by any script or CI job. Converting a visible skip into an invisible orphan is strictly worse. The guard's remediation text compounded the defect by instructing developers to move tests to `test/integration/` without wiring them up.

## Rev 2 Rejection Reason

`test:integration` script existed in `package.json` but was not invoked by any CI job. Rule5 certified `test/integration/*.test.js` files as "reachable" because the glob matched — but rule5 only proves script reachability, not CI invocation. A script not called by CI certifies orphans while executing nothing. Rule5 alone is not transitive all the way to CI.

## Rev 3 (commit 424ce26) — Rule6 added

**Rule6:** every npm script containing `node --test` must be invoked by at least one step in `.github/workflows/*.yml`. Combined with rule5, this makes the chain transitive and complete:

```
test file → matched by a script glob (rule5)
          → that script is invoked by CI (rule6)
```

**`test:integration` deleted:** no unwired scripts. Rule6 forces wiring at the moment the script is reintroduced.

**Regex fix:** `isScriptInvokedByCI` used `\b` which matches `test` inside `test:integration` (`:` is a non-word char). Fixed to `(?!\S)`. Bug discovered during mutation 3.

**Mutation results (424ce26):**

| Mutation | Rule | Direction |
|---|---|---|
| `test:unwired` added to package.json (not in CI) | rule6 | RED |
| `npm run test:unwired` added as CI step | — | GREEN 123 |
| Mutations 1+2 restored; `npm test` removed from ci.yml | rule6 | RED |
| Restored | — | GREEN 123 |

**Detached-worktree results (424ce26):**
```
tests 123 · pass 123 · fail 0 · skipped 0
```

---

## Background

Cristiano identified that `test/enumRuntimeRegression.test.js` on `gert-vscode` main (`f1e5c51`) contained four tests that silently skipped in any clean CI checkout:

```
{ skip: !gertBin && 'no gert binary found; set GERT_BIN or build ../gert/gert(.exe)' }
```

The skip condition was `true` whenever `GERT_BIN` was unset and `../gert/gert(.exe)` did not exist — which is always the case in a detached worktree of `gert-vscode` alone. The reported count of 118/118 was wrong; the true result was **114 pass / 4 skipped**.

This is the **ninth occurrence** of the team's systemic "appears covered but is not reachable" bug class. The mirror defect on the Go side (`gert/internal/tool/deliverable_contract_test.go` reading `../gert-private/`) was fixed by Don in `21e9c6c`/`d0d8aba` with `TestRepoBoundary_NoExternalPaths`. This document records the JS-side fix.

---

## What the Four Tests Asserted

| Test | AR-CE ID | Client function tested | Option |
|------|----------|----------------------|--------|
| ENUM-008 survives `deriveFailureMessage` | CE-V-04/D-3 | `deriveFailureMessage(err)` | (a) fixture |
| Declared enum member accepted by CLI | CE-S-01/CE-V-01 | *none — pure CLI acceptance* | (b) integration |
| ENUM-W001 extracted by `warningLines` | CE-W-01 | `warningLines(stderr)` | (a) fixture |
| `extractInputDecls` parses graphjson | CE-D-01/CE-D-02 | `extractInputDecls(doc)` | (a) fixture |

---

## Decisions Made

### CE-V-04/D-3, CE-W-01, CE-D-01/CE-D-02 — Option (a): Committed Fixtures

The relevant CLI output was captured from the gert binary at f1e5c51 (2026-08-18) and committed as:

- `test/fixtures/enum-error-stderr.txt` — stderr from `gert dry-run --var env_name=not-a-member enum.runbook.yaml`
- `test/fixtures/enum-warn-stderr.txt` — stderr from `gert dry-run --var env_name=prod enum-warn.runbook.yaml`
- `test/fixtures/enum-preview-graphjson.json` — stdout from `gert preview --format graphjson enum.runbook.yaml` (absolute runbook paths stripped; not asserted on and machine-specific)

Each test now passes the fixture content through the client-side helper function and asserts the same structural invariants as before.

**Declared drift risk:** if the CLI changes the ENUM-008 prefix, ENUM-W001 format, or renames the `inputs` key in the graphjson Document, these tests will continue to pass against stale fixtures. The live-binary integration suite (`test/integration/cli.test.js`) is the intended drift defence — it must be run wherever `GERT_BIN` is available.

### CE-S-01/CE-V-01 — Option (b): Integration Test

This test's only assertion was `assert.match(stdout, /dry-run complete/)` after `pexec(gertBin, ['dry-run', '--var', 'env_name=prod', FIXTURE])`. No extension client function is in the loop. There is no fixture approach that tests any extension code here — the test is a pure CLI acceptance check.

The test was moved to `test/integration/cli.test.js`. That file:
- Is NOT picked up by `npm test` (`test/integration/` is a subdirectory, the glob is `test/*.test.js`)
- Fails (never skips) when `GERT_BIN` is absent
- Documents clearly what it requires and why

The contract guarded by CE-S-01/CE-V-01 (the CLI does not reject a declared enum member) is protected at the Go level by `gert`'s own test suite.

---

## Guard Added: `test/repoBoundary.test.js`

Mirrors `gert/internal/tool/repo_boundary_test.go` (`TestRepoBoundary_NoExternalPaths`).

Walks `test/` and `src/` and fails on:

1. **Sibling-repo name references** — `gert-private` anywhere in a test/source file.
2. **Cross-repo path traversals** — `path.join(__dirname, '..', '..', 'gert'` and `'../gert'` variants.
3. **`process.env.GERT_BIN` in unit test files** — the canonical escape hatch to the sibling binary; prohibited in `test/*.test.js` (integration files in `test/integration/` are exempt).
4. **`skip:` in test declarations** — the `{ skip: ... }` Node.js test option; any occurrence in a unit test file fails the guard.

**Allowlist:** empty — no justified exceptions currently exist. The allowlist mechanism is present (with a mandatory-justification-per-entry contract) for future use.

**Mutation test results:**

| Mutation | Guard result |
|----------|-------------|
| Added `{ skip: 'probe' }` to CE-V-04 test | ✖ RED — `contains a skip: test option` |
| Added `path.join(__dirname, '..', '..', 'gert'` comment | ✖ RED — `contains cross-repo path traversal` |
| Both removed | ✔ GREEN — 118/118 pass |

---

## Verification Results

From a detached worktree of `fc273b6`:

```
git worktree add --detach <wt> fc273b6
git status --porcelain          # empty
npm ci                          # exit 0
npm run compile                 # exit 0
npm test
```

```
tests    118
pass     118
fail       0
skipped    0
```

**Before this fix (f1e5c51):**
```
tests    118
pass     114
fail       0
skipped    4
```

---

## Hard Rules Compliance

- ✅ No `git add -A`; all staging by explicit path.
- ✅ Branch from `main` (`f1e5c51`), not from `squad/rev3-tsg-absence`.
- ✅ `git status --porcelain` confirmed empty before staging — session 89b67e11's work was absent.
- ✅ No assertion weakened.
- ✅ Previously-skipped tests did not fail on first run (fixtures matched expected patterns).
- ✅ Guard mutation-tested with measured before/after output. "Verified by code review" was not used.
- ✅ Detached worktree used for final verification — never the working tree.
- ✅ Full four-number breakdown reported (tests / pass / fail / skipped).

---

## Rev 2 Changes (commit 0b98842)

| File | Status |
|------|--------|
| `test/integration/cli.test.js` | Deleted — CE-S-01/CE-V-01 orphan removed; see rejection reason above |
| `.github/workflows/ci.yml` | Modified — added `npm test` step (tests never ran in CI before); documented integration test pattern in comments |
| `package.json` | Modified — added `test:integration` script for future binary-requiring tests |
| `test/repoBoundary.test.js` | Modified — split into 5 per-rule tests; added rule5 (orphan detection); fixed rule3 remediation message |

**Orphan detection (rule5):** reads `package.json` scripts at runtime; extracts `node --test <glob>` patterns; fails any `test/**/*.test.js` not matched by at least one pattern. Current configured globs: `test/*.test.js`, `test/integration/**/*.test.js`.

**Mutation results (0b98842):**

| Mutation | Rule | Direction |
|---|---|---|
| `test/orphan-dir/orphan-probe.test.js` (no script covers it) | rule5 | RED |
| Moved to `test/integration/` (covered by `test:integration`) | — | GREEN 122 |
| Restored | — | GREEN 122 |

**Detached-worktree results (0b98842):**

```
git status --porcelain  →  empty
npm ci                  →  exit 0
npm run compile         →  exit 0
npm test
  tests    122
  pass     122
  fail       0
  skipped    0
```

---

## Files Changed (All Commits, Explicit Paths)

| File | Status |
|------|--------|
| `test/enumRuntimeRegression.test.js` | Modified (fc273b6) — rewritten (4→3 tests, fixture-based) |
| `test/fixtures/enum-error-stderr.txt` | Added (fc273b6) — ENUM-008 stderr fixture |
| `test/fixtures/enum-warn-stderr.txt` | Added (fc273b6) — ENUM-W001 stderr fixture |
| `test/fixtures/enum-preview-graphjson.json` | Added (fc273b6) — graphjson fixture (paths sanitized) |
| `test/integration/cli.test.js` | Added (fc273b6) then Deleted (0b98842) — orphan removed |
| `test/repoBoundary.test.js` | Added (fc273b6), Modified (0b98842) — 5 per-rule tests + orphan detection |
| `.github/workflows/ci.yml` | Modified (0b98842) — added npm test to CI |
| `package.json` | Modified (0b98842) — added test:integration script |


---

# Decision: Contract-Parity Harness API Shape

**Date:** 2026-08-17
**Author:** Tess
**Status:** Informational — no arbitration needed; records design choices made while implementing consumer Item 1.

---

## Context

The SQL Live-Site Operations team requested a reusable contract-parity test harness
(Item 1 of 3). This records the design decisions made in `pkg/contractparity`.

---

## Decisions

### 1. Package location: `pkg/contractparity` (not `internal/`)

The consumer explicitly said they do not want to copy a Gert-specific test. The
package must be importable from a foreign repo. `internal/` is inaccessible outside
the module; `pkg/` is the correct location.

### 2. Pure function + thin adapter split

The comparison logic (`CompareOutputs`, `CompareMeta`, `Check`) is separated from the
`testing.T` binding (`AssertParity`, `RequireParity`). Rationale:
- Core logic is testable without `*testing.T`.
- Consumers who want structured diffs (e.g. for CI reporting) can call `Check` and
  inspect `Report` without going through a test framework.
- The testing adapter is a 3-line wrapper; no hidden complexity.

### 3. `Invoke` signature: `func(ctx, args) (map[string]any, error)`

Rather than importing `pkg/tool.ToolResult`, the harness accepts `map[string]any`
directly. This removes a Gert-specific type dependency from a package intended for
foreign consumers. Don's bindings can wrap `ToolResult.Output` trivially.

### 4. `ActionMeta` fields are all `*T` (nil = skip)

A consumer that does not wish to assert classification or idempotency simply leaves the
pointer nil; the harness silently skips that field. This avoids forcing consumers to
fill in fields they don't care about, and avoids false violations when one binding
declares a field the other omits.

### 5. `CompareMeta` takes an existing `*Report` and appends

Rather than returning a second separate Report, `CompareMeta(r, ...)` appends meta
violations to an existing output-comparison report. Callers get one unified Report with
all violations in sorted order, suitable for a single diagnostic message.

### 6. `ViolationMissingKey` vs `ViolationExtraKey` are distinct kinds

Keeping them separate lets consumers distinguish "B is incomplete" from "B leaks
internal fields". Both are parity failures but have different remediation paths.

---

## Non-decisions (deferred to consumers)

- The harness does NOT assert on `ExitCode` or `Stdout/Stderr` — only `Output` map
  shape and `ActionMeta`. Consumers with stricter contracts can extend `CompareMeta`.
- Deep recursive comparison of nested maps is intentionally not implemented; only the
  top-level reflect.Type is compared. If a consumer needs recursive shape checking they
  can compose multiple `CompareOutputs` calls on sub-maps.

---

# Design Decision: Runtime Output Contract Enforcement for Non-Substituted Tool Actions

**Date:** 2026-08-17  
**Author:** Tess (conformance/test engineer)  
**Commit:** ca867a1  
**Files changed:** `internal/executor/tool.go`, `internal/executor/tool_output_contract_test.go`, `cmd/gert/reachability_registry_test.go`, `cmd/gert/reachability_probes_test.go`

---

## Problem

`returns:` / `outputs:` declarations on non-substituted `.tool.yaml` actions (native, mcp-stdio, mcp-http transports) were never checked at runtime. The executor assembled `result.Output` with a verbatim copy of `res.Output` and injected `stdout`, `stderr`, `exit_code` without consulting the declared contract at all.

Substituted actions (`execute.kind: runbook`) already enforced their output contracts via `executeSubstitution` → `coerceOutputAny`. This was the fifth instance of the declared-but-unenforced bug class on the project.

---

## Decision

Add `enforceOutputContract()` to `internal/executor/tool.go` called after `result.Output` is assembled for every non-substituted tool execution.

### Rule 1 — stdout / stderr / exit_code are process-level channels

These three keys are injected unconditionally by the executor and exist on every result regardless of the declared contract. They are never part of the semantic `outputs:` contract.

**Enforcement rule:**
- They are NEVER flagged as "undeclared" even if absent from `outputs:`.
- They are NEVER required to be present in `outputs:` declarations.
- They are always passed through to the result unchanged.

This preserves all existing runbooks that capture `{{ step.stdout }}` etc. without any tool YAML change.

### Rule 2 — Empty / absent outputs: declaration = unconstrained

If a `.tool.yaml` action declares no `outputs:` block (empty map or nil), enforcement is skipped entirely and all outputs pass through unchanged. This is the current state of 100% of existing tool YAML files — zero tools are affected by this change.

**Bounded escape hatch:** the escape only applies when `declaredOutputs` is nil or empty. Any tool that adds `outputs:` to its YAML immediately gets enforcement. This cannot silently swallow the whole feature because the reachability probe (see below) runs a tool WITH declared outputs and asserts the enforcement fires.

### Rule 3 — Fail closed, not warn-and-continue

Any violation (missing declared output, undeclared extra output, type-incompatible value) fails the step immediately with `StepStatusFailed` and a structured error message naming the tool, action, and all offending fields. All errors are collected and reported together (not first-error-only).

### Rule 4 — Reuse coerceOutputAny, do not diverge

The same `coerceOutputAny` function used by the substituted path is called by `enforceOutputContract`. Both enforcement paths agree on type coercion semantics. Any divergence would itself be a contract bug.

---

## Implementation Details

`resolvedActionDef *tool.ToolAction` is captured in the `Execute()` method after the `ToolDefLookup` call (non-substituted branch only). If the runtime does not implement `ToolDefLookup` or the tool/action is not found, `resolvedActionDef` remains nil and enforcement is skipped — this is intentional graceful degradation for unknown tools.

After `result.Output` is assembled from `res.Output` (and before the result is returned), enforcement is called:

```go
if resolvedActionDef != nil && len(resolvedActionDef.Outputs) > 0 {
    validated, cerr := enforceOutputContract(toolName, action, resolvedActionDef.Outputs, result.Output)
    if cerr != nil {
        // fail step
    }
    result.Output = validated
}
```

---

## Reachability

Entry added to `reachabilityRegistry` in `cmd/gert/reachability_registry_test.go`:

```
"ToolAction.Outputs (non-substituted enforcement)" → statusReachable
```

Probe (`testCLI_OutputContractEnforcement_Reachable`): creates a native tool YAML that declares `outputs: {result: string}`, runs it via gert, asserts `code != exitSuccess`. Native transport never populates `res.Output` — so "result" is always missing → enforcement always fails the step → process exits with code 1. If the production enforcement call is removed, the step would succeed (no enforcement = pass-through) and the probe would fire `t.Fatal`.

---

## Mutation Control Results (measured, not reasoned)

Mutation applied: remove `var resolvedActionDef`, `resolvedActionDef = actionDef`, and the enforcement block from `Execute()`.

| Test | Result without enforcement |
|------|---------------------------|
| `TestToolExecutor_OutputContract_Missing_Fail` | FAIL — "MUTATION CONTROL FAILED" |
| `TestToolExecutor_OutputContract_Undeclared_Fail` | FAIL — "MUTATION CONTROL FAILED" |
| `TestToolExecutor_OutputContract_TypeMismatch_Fail` | FAIL — "MUTATION CONTROL FAILED" |
| `TestToolExecutor_OutputContract_Satisfied_Pass` | PASS (correct — enforcement not needed for pass case) |
| `TestToolExecutor_OutputContract_NoDeclaration_PassThrough` | PASS (correct — no declaration, no enforcement) |

---

## Tests — Full Coverage

16 tests in `internal/executor/tool_output_contract_test.go`:

**Pure `enforceOutputContract` function (9):**
- `_NilDeclaration_PassThrough` — nil declaredOutputs: all pass
- `_EmptyDeclaration_PassThrough` — empty map: all pass
- `_Satisfied_Pass` — declared field present with correct type
- `_Missing_Fail` — declared field absent → error
- `_Undeclared_Fail` — extra field not declared → error
- `_TypeMismatch_Fail` — wrong type → error
- `_ProcessChannels_NeverFlagged` — stdout/stderr/exit_code never flagged
- `_MultipleErrors_AllReported` — all violations collected, not first-only
- `_TypeCoercionFloat_Pass` — float64 accepted for "number" type (matches coerceOutputAny)

**Execute-path integration / mutation controls (7):**
- `Satisfied_Pass` — declared output present, step completes
- `Missing_Fail` (mutation control) — declared output absent, step fails
- `Undeclared_Fail` (mutation control) — extra output, step fails
- `TypeMismatch_Fail` (mutation control) — wrong type, step fails
- `NoDeclaration_PassThrough` — no outputs declaration, pass-through
- `ProcessChannels_AlwaysPresent` — stdout/stderr/exit_code in result even when declared outputs satisfied
- `NoDefRegistered_PassThrough` — runtime has no ToolDefLookup, pass-through

---

## Pre-existing Tests Modified

None. All 82 packages passed without test changes. The 0 existing tools that declare `outputs:` on non-substituted actions take the unconstrained path — no behavioral change.


---

