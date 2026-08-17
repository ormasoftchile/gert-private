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

## Learnings

- **Injectable TTY detection prevents pervasive test breakage.** A package-level function var for `isInteractiveTTY`, stubbed to `true` in TestMain, is the cleanest pattern for features that depend on TTY state without touching every existing test.
- **makeWorkDir returns relative paths; use filepath.Abs before os.Chdir.** After chdir, a relative `dir` variable used in filepath.Join produces double-nested paths. Always resolve absolute before any chdir when seeding test fixtures.
- **Resume test strategy: seed state, verify error changes.** For `--acknowledge-indeterminate`, seed a `RunStatusIndeterminate` snapshot via `SaveState`, then assert the flag changes the specific error returned — not that the run succeeds (plan isn't in-memory store).
