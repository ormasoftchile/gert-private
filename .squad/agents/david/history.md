# David — History Summary (Archived content in history-archive.md)

**Role:** Integration Engineer

**Current Focus:** Phase 1B execution: fail-fast, indeterminate state semantics, ICM contract proof.

## Phase 1A Work Summary

**Stream C — Auth Provider Interface & Azure CLI Implementation:** Completed. Designed AuthProvider interface, implemented AzureCLIAuthProvider with token caching, 5-minute refresh, invalidation on 401.

**Key files:** `internal/tool/auth_azurecli.go`, auth validation, error taxonomy redesign.

## Phase 1B Rev 2 Corrections (2026-08-17)

### Correction A: Profileless Non-Interactive Fail-Fast

- **Implementation:** CLI entry in `cmd/gert/run.go` after profile check, uses `os.Stdin.Stat()` + `ModeCharDevice` (stdlib only).
- **Harness fix:** Inject unattended test profile fixture at `internal/conformance/testdata/unattended-test.profile.yaml`.
- **Estimate:** 1.5 days total.

### Correction B: INDETERMINATE Evidence Record

- **Seven missing fields:** Tool name, action name, classification, endpoint host, attempt number, deadline, transport error category.
- **Design:** Embed `*IndeterminateRecord` pointer on `StepResult` (non-fabrication approach).
- **Estimate:** 5.5–6 days (revised from 4–5).

### Correction C: ICM Proof Target Real Contract

- **Finding:** Proof was targeting Gert's sample contract (underscores) not Live-Site's actual contract (hyphens).
- **Status:** `--package-map` is execution-wired; `--profile` is not (Phase 1B Item 2).
- **Blocking:** Six artifacts required from SQL Live-Site Operations.
- **Estimate:** 3 days (revised from 1 day, after Item 2 completion).

## Key Phase 1B Findings

**Fail-fast straightforward:** No `isatty()` anywhere; stdlib detection works on Windows.

**INDETERMINATE evidence:** Nil pointer = completion unknown; avoids fabricated output.

**ICM contract accuracy:** Correction identified underscores vs hyphens — wrong tool definition would yield false-positive test.

**Profile binding:** Not yet wired; Item 4 gated on Item 2 completion.

## Phase 1B Rev 2 Scope (Revised)

| Item | Estimate | Status |
|------|----------|--------|
| 1. Managed Identity | 1.5 days | — |
| 2. Runtime Binding | 3–4 days | — |
| 3. INDETERMINATE | 5.5–6 days | — |
| 4. ICM proof | 3 days | Gated on Item 2 + artifacts |
| 5. Fail-fast | 1.5 days | NEW |
| 6. Credential-leak assertions | 1 day | NEW |

**Revised parallelized estimate:** 9–10 days.

## Learnings

- **Contract parity must use actual schemas, not samples.** Wrong tool definition = false-positive proof.
- **Fail-fast at CLI boundary catches configuration errors early.** Correct seam: after profile check, before engine construction.
- **Evidence record requires explicit non-fabrication design.** Pointer presence = completion unknown; nil Output avoids false inferences.
- **Profile execution wiring is distinct from profile schema.** Schema fields exist; execution path does not. Verify end-to-end reachability.
- **PLAN-013 must ship in same commit as endpoint-override wiring.** A window between commits where overrides execute without validation is a security regression. One commit = no window.
- **Ratified Rule A is enforced structurally, not by runtime assertion.** ProfileToolOverride has no Scope/AllowedHosts fields; they simply cannot be set. Provider override IS wired; TokenGate always reads scope+allowedHosts from def.
- **Reachability tests need two servers, not one.** A single-server reachability test passes trivially if the production read is removed (requests go to the same server regardless). Two servers (default vs override) produce a diagnostic assertion that fails directionally.
- **Transport HTTPS enforcement is at ValidateTransportConfig (scan time), not ValidateAuthConfig (runtime).** Unit tests that bypass YAML parsing can use plain HTTP test servers without tripping the HTTPS check.
- **`docs/` being gitignored is the same failure class as 91 untracked files.** Files force-added survive locally but are invisible on a clean checkout. Remove the blanket ignore rule; audit each file first; replace with narrow rules only for genuinely machine-specific or generated content. The `git worktree add --detach` verification is the only check that counts — working tree looks fine even with the problem present.

Full detailed history: `.squad/agents/david/history-archive.md`

## Phase 1B Rev 3 Update (2026-08-17)

### Item 4 Ownership Restructure

**Status Update:** Item 4 ("Dual-Binding Mechanism Proof") no longer imports consumer contracts and is not artifact-gated.

- Gert core owns mechanism proof using synthetic fixtures
- Consumer-specific contracts (e.g., SQL Live-Site ICM contract) are owned by consumer team in their repo
- Item 4 is now serial after Item 2 only; no external dependencies
- Full details: Contract Proof Ownership Split decision (decisions.md)

## Phase 1B Item 2 Complete (2026-08-17)

### Profile Execution Wiring + PLAN-013

**Status:** SHIPPED in commit `85bfa4a`.

Wired `ProfileToolOverride.Endpoint` and `Provider` through `adapter.WireOptions` → `BuildEngineConfig` → `DefaultToolRuntime.SetProfile` → `Invoke`. PLAN-013 preflight check (endpoint override host must be in `allowed_hosts`) ships in same commit — no unvalidated execution window. Item 4 is now unblocked.

## Phase 1B Items 5+B Complete (2026-08-17)

### Profileless Non-Interactive Fail-Fast + --acknowledge-indeterminate

**Status:** SHIPPED in commit `d1314cc`.

**Item 5:** Added `interactiveTTYDetect` package-level var (overridable in tests) so existing profileless CLI tests aren't broken. `TestMain` in `cmd/gert/test_main_test.go` sets it to `true` for all tests; only the fail-fast-specific test overrides it to `false`. Fixed `TTYOutput: true` hardcode → `isInteractiveTTY()`. Fail-fast fires before any approval gate.

**Part B (Tess complement):** Added `--acknowledge-indeterminate` flag wired to `engine.RunOptions.AcknowledgeIndeterminate` in the resume path.

**Conformance harness:** Fixed by injecting `internal/conformance/testdata/unattended-test.profile.yaml`. Vectors unchanged: **72 / 62 / 10 / 0**.

**Reachability registry:** Graduated `ProfileToolOverride.Endpoint` from `statusKnownDead` to `statusReachable` with `testCLI_ProfileToolOverrideEndpoint_Reachable` probe (PLAN-013 path).

## vscode-mcp Transport Complete (2026-08-17)

### Phase 2 Item 2 — vscode-mcp Transport Mode, End-to-End

**Status:** SHIPPED in commits `a35c0ae`, `3cd262e`, `9c39638`.

**Five chain points wired:**
1. `pkg/schema/tool.go` — `TransportVSCodeMCP Transport = "vscode-mcp"`
2. `pkg/tool/tool.go` — `TransportVSCodeMCP TransportType = "vscode-mcp"` (string identical in both layers)
3. `internal/tool/validate_transport.go` — explicit `case schema.TransportVSCodeMCP:` rejecting `url:`, `auth:`, `command:` with clear errors. Does NOT inherit mcp-http's HTTPS requirement.
4. `internal/tool/scan.go` — `mapTransport` case, `TransportVSCodeMCP` → `toolpkg.TransportVSCodeMCP`.
5. `internal/tool/runtime.go` — `Invoke` switch arm dispatching to `VSCodeMCPTransport` via `invokePersistent`.

**Bridge client:** `internal/tool/vscode_mcp_bridge.go`. Loopback-only, per-session capability secret (crypto/rand 32 bytes hex), versioned wire contract `vscode-mcp-bridge/v1`, context-aware cancellation, version mismatch hard-fails, idempotency support structured, credential sweep covers cap secret.

**Bridge server:** NOT built — extension team owns server side. Wire contract is fully defined in vscode_mcp_bridge.go. Interface is clean.

**Tests:** 10 test cases (success, typed output, timeout, cancellation, duplicate request, malformed, capability rejection, extension disconnect, version mismatch, secret redaction) + negative control (sweep detects leak). All passing. Mutation controls documented and measured.

**Reachability:** `TransportVSCodeMCP` registered as `statusReachable`. Mutation verification: removing `mapTransport` case causes `TestCLI_ReachabilityGate/TransportVSCodeMCP` to FAIL (exit 1, "caused a validation/scan failure"). Restoring: PASS. Non-vacuous confirmed.

**Indeterminate:** `requiresIndeterminate` returns false for `read-only` by design in `engine.go`. vscode-mcp stays in `"tool"` executor kind; `executeStep` classification injection fires automatically. No reimplementation needed or done.

**Fixture:** `internal/tool/testdata/vscode-mcp-ops-synthetic.tool.yaml` demonstrating contract-identical vscode-mcp binding for `pkg/contractparity` parity.

**Build/Test:** `go build ./...` exit 0; `go test ./...` exit 0, all packages pass.

## Learnings

- **vscode-mcp transport: loopback validation must be at construction time, not invocation time.** validateLoopback fires on every Invoke call — but it should also fire at construction in a future improvement. For now, the call-time check is the safety net.
- **Bridge server deferral is the right split.** The wire contract (defined in vscode_mcp_bridge.go) is the boundary artifact. Extension team implements server; core implements client. Trying to build both in one task would block on extension-repo prerequisites (P0-P2 from Barbara's ruling).
- **Five-chain-point reachability is non-negotiable.** Steps 1-3 without 4-5 creates a parse-and-validate sink that is unreachable at runtime. This repo has shipped that exact bug four times. The reachability probe mutation test is the only check that counts.
- **Capability secret must be generated with crypto/rand, not math/rand.** Security model requires unpredictable secrets. Panicking on crypto/rand failure is correct — there is no safe fallback.
- **Unused imports in test files fail CI immediately on Windows.** The agent added `encoding/json`, `net/http`, `net/http/httptest` to reachability_probes_test.go but did not use them in the probe function. Always verify `go build` before committing test files.
- **Injectable TTY detection prevents pervasive test breakage.** A package-level function var for `isInteractiveTTY`, stubbed to `true` in TestMain, is the cleanest pattern for features that depend on TTY state without touching every existing test.
- **makeWorkDir returns relative paths; use filepath.Abs before os.Chdir.** After chdir, a relative `dir` variable used in filepath.Join produces double-nested paths. Always resolve absolute before any chdir when seeding test fixtures.
- **Resume test strategy: seed state, verify error changes.** For `--acknowledge-indeterminate`, seed a `RunStatusIndeterminate` snapshot via `SaveState`, then assert the flag changes the specific error returned — not that the run succeeds (plan isn't in-memory store).

## Binding Docs Complete (2026-08-17)

### Deliverable 3 — Contract-Identical Bindings Guide

**Status:** SHIPPED in commit `e6521f6`.

**Doc path:** `docs/contract-identical-bindings.md` (cross-linked from README).

**Key findings from implementation read:**

- `docs/` is in `.gitignore` ("# Design docs (local)") but existing perf files are tracked via force-add. Used same pattern (`git add -f`).
- PLAN-013 endpoint override check fires in `checkToolEnvironmentPreflight` **before** the test-context transport check, not after. Both are Tier 0 (no I/O). Described accurately.
- `ProfileToolOverride` has no `Scope` or `AllowedHosts` fields — Rule A is structural, not runtime-asserted. Confirmed.
- PROF-001 (transport-mode rewriting) is enforced at profile parse time in `validateProfile`, not at plan time. Profile file is rejected immediately on load.
- `classification` is per-action on `ToolAction`, pointer type (`*string`), so nil is structurally unspecified. Confirmed orthogonality to `requires-approval` via the struct comment in schema/tool.go.
- MCP-012 fires in `TokenGate.AttachToken` at HTTP dispatch time — this is the runtime backstop; PLAN-013 is the planning-time gate for profile endpoint overrides. Both described correctly.
- `allowed_hosts` wildcards (`*.azure-api.net`) are explicitly NOT supported and are treated as literal hostnames — documented in `auth_gate.go` comments. Included in doc.

**No implementation contradictions found.** All described behaviors match the source.
