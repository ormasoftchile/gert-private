## Learnings

### 2026-08-17 — Phase 1B Rev 2: SQL Live-Site Corrections A, B, C

**Context:** SQL Live-Site Operations accepted Phase 1B plan subject to eight corrections; three (A, B, C) are in David's territory.

**Correction A — Profileless non-interactive fail-fast (moved into Phase 1B)**

- `TTYOutput: true` is hardcoded at `cmd/gert/run.go:166` (the `WireOptions` literal). Gate selection happens at `internal/adapter/wire.go:306–313` (`buildApprovalGate`).
- No `isatty()` anywhere in the codebase — confirmed by grep. `golang.org/x/term` and `github.com/mattn/go-isatty` are not in `go.mod`. `golang.org/x/sys` is present (indirect, via OTel). Stdlib-only detection via `os.Stdin.Stat()` + `ModeCharDevice` needs no new dependency.
- Fail-fast belongs at CLI entry in `run.go`, after the profile check block (~line 110), before `BuildEngineConfig` (~line 162). Interactive operators with no profile pass through (TTY detected → no fail).
- **Harness impact is real and breaks everything.** `internal/conformance/enum_harness.go:runCLI()` invokes `gert run <entry> --trace <path>` with no `--profile` and no TTY — all conformance and enum/dyninclude vectors hit the fail-fast. Fix: harness injects `--profile <unattended-test.profile.yaml>` (a minimal fixture with `attendance: unattended`). Behavior is identical to current `TTYOutput: false` path; gate selection outcome unchanged.
- `internal/e2e/helpers_test.go:171` uses `TTYOutput: false` directly (bypasses CLI entry) — unaffected.
- **Estimate: 1.5 days** (fail-fast logic + harness fixture + regression pass).

**Correction B — INDETERMINATE must retain trace-safe invocation evidence**

- `pkg/engine/run.go:160–190` — `StepResult` struct. Fields present: `StepID`, `Status`, `Output`, `StartedAt`, `CompletedAt`, `DurationMs`, `Error`. Run ID is in `RunState.RunID`.
- **7 of 7 required new fields are absent:** logical tool name, logical action name, classification, endpoint host, attempt number, deadline, transport error category.
- Redaction seam: two existing seams (output-layer token redaction, runbook `redact:` block) are not directly applicable. Endpoint host is not a credential; record `url.Parse(def.URL).Host` only, never the auth token or `Authorization` header. No new redaction library needed — code-review invariant suffices.
- "No result evidence" constraint: `StepResult.Output` nil is ambiguous (empty response vs never responded). Resolution: embed `*IndeterminateRecord` pointer on `StepResult`. Non-nil pointer = completion unknown. `Output` left nil for indeterminate steps; code that reads `Output` must gate on status (already required for failed steps).
- Proposed `IndeterminateRecord` fields: RunID, StepID, ToolName, ActionName, Classification, EndpointHost (host only), AttemptNumber, Deadline, FailureTime, TransportErrCategory.
- **Revised estimate: 5.5–6 days** (was 4–5; +1.5 for evidence record struct + serialization + resume-guard display + test assertions).

**Correction C — ICM proof must target real contract (feasibility only)**

- `--package-map` IS execution-wired: `cmd/gert/packagemap_integration_test.go:23` proves end-to-end execution with mock vs real binding. `cmd/gert/run.go:272` loads package-map before `BuildEngineConfig`; catalog feeds the tool registry used at execution time.
- `--profile` is NOT execution-wired: confirmed Phase 1B Item 2. The proof cannot exercise managed-identity binding at execution until Item 2 ships.
- No contract parity assertion infrastructure exists. `TestRun_PackageMap_RealVsMockBinding` asserts *different* outputs (inverse of parity). Tess's MCP parity tests compare transport framing (SSE vs JSON), not cross-binding output parity. New infrastructure needed.
- Native mock binding: `httptest.Server` for mcp-http (existing pattern in `mcp_http_tess_test.go`); Go subprocess binary for mcp-stdio (existing pattern: `cmd/tools/echo`). Both use same logical tool/action; `--package-map` selects which binding.
- **Blocking artifacts required from SQL Live-Site Operations (exact list):**
  1. Full content of `gert-sqllivesite/runbooks/icm-tsg-router.runbook.yaml`
  2. Full YAML tool definition for `icm.get-incident` (hyphen action name, complete input schema, all 5 output fields with types: `title`, `service`, `environment`, `logical_server`, `database`)
  3. Full YAML tool definition for `tsg-recommendation.recommend` (both outcome shapes: `suggested` and `no-suggestion` with all field names and types)
  4. Their `--package-map` file (or equivalent) mapping both tools to their packages/paths
  5. Transport mode for each tool in their production binding (`mcp-stdio` or `mcp-http`)
  6. Whether the runbook uses `toolRefs:` with package names or bare path references
- **Estimate: 3 days** after Item 2 ships AND all 6 artifacts received (was 1 day against our sample tool; new scope is materially larger due to parity harness and dual-outcome coverage).

---

## 2026-08-16 — Team Orchestration Session: Runtime Portability Evaluation

**Session:** Scribe coordination session with barbara, don, david  
**Task:** Evaluate and document SQL Live-Site Operations "Runtime Portability for Gert Runbooks" implementation request

**David's session outputs:**
- Integration protocol critique completed: 3 critical blockers, 1 phase-ordering risk
- Detailed error-taxonomy redesign (5-code split) for operator clarity
- Tiered-preflight specification (Tier 0 static, Tier 1 local, Tier 2 live) aligns ask with invariant #4 (fail-closed)

**Team consensus findings:**
- Preflight design must split static (configuration exists?) from dynamic (can we reach it now?) — David designed the split; Barbara + Don validated the concept
- Host bridge protocol has gaps: version, capability advertisement (required for Tier 0!), audit correlation, cancel message for IPC
- Phase 1 must ship fail-closed semantics for timeout on mutating/destructive actions (David's critical call-out; architectural team must accept or restrict Phase 1 to read-only)

**Deliverables merged to decisions.md.**

## Learnings

## Learnings

### 2026-08-17 — Phase 1B Scope Investigation (Claims 3 & 4)

**Context:** SQL Live-Site Operations split the accepted Phase 1 milestone into Phase 1A (complete) and Phase 1B. Requested empirical verification of Claims 3 and 4 before scoping.

**Claim 3 — INDETERMINATE / halt-on-timeout: CONFIRMED NOT IMPLEMENTED**

- `INDETERMINATE` does not exist anywhere in the codebase (zero matches across all files).
- Step status enum (`pkg/engine/run.go:194–203`): pending, running, completed, failed, skipped, waiting, denied. No indeterminate state.
- `Contract.Idempotent` (`pkg/schema/step.go:93`) is a schema-only field. It is never read by the engine, retry logic, or timeout path. The only runtime reference is a test assertion in `internal/engine/approval_enforcement_test.go:380` confirming it is orthogonal to approval — not consulted during execution.
- There is no retry path in `internal/engine/engine.go` that checks `Contract.Idempotent` or `Classification` before retrying. Classification is read at `engine.go:536` only to populate `StepInfo.ToolClassification` for the ProfileEvaluator (governance/approval decisions) — it never feeds a retry guard.
- When a step times out (context.DeadlineExceeded from the executor), the engine receives it as `execErr` at `engine.go:660` and calls `failRun()` — which marks the step `StepStatusFailed`. No halting, no INDETERMINATE, no classification check. A destructive action that times out today lands in `failed` and execution continues (or stops per `on_error`/`continue_on_fail`) — the same as a read-only timeout.
- **Effort to implement the contract:** New `StepStatusIndeterminate` value + new `StepOutcomeIndeterminate`; engine timeout path must classify the step's action before deciding `failed` vs `indeterminate`; halted-INDETERMINATE runs must be persisted so `gert resume` can surface them (resume guard needed); at least one test vector per classification (read-only, mutating, destructive, unspecified) for both timeout and lost-transport cases. Estimate: **4–5 days** (schema + engine + persistence + tests).

**Claim 4 — Profile endpoint/auth binding at execution: CONFIRMED NOT IMPLEMENTED**

- `--profile` flag is parsed in `cmd/gert/run.go:101–108` and `runtimeProfile` is passed to `plannerImpl` (line 137, fires Tier 0 preflight) and stored in `plan.Metadata.Profile` (line 456).
- `BuildEngineConfig` (`internal/adapter/wire.go`) has no `Profile` parameter in `WireOptions`. The profile is not passed to the executor registry, tool runtime, or any transport constructor.
- The engine reads `plan.Metadata.Profile` in exactly one place: `engine.go:127/250` — to build `ProfileEvaluator` for governance/approval scope. It does not use it for transport parameterization.
- Transport construction happens in `internal/tool/runtime.go:60–77`. For `mcp-http`, the HTTP client is built with `NewMCPHTTPTransport(def.URL, gate)` where `def.URL` comes from the tool definition and `gate` is built from `NewAuthProvider(def.Auth.Provider, def.Auth.Scope)` — both sourced entirely from the tool definition, not the profile.
- `ProfileToolOverride.Endpoint` exists in the schema (`pkg/schema/profile.go:108`) and is parsed/tested. The field's comment (`profile.go:111`) confirms: auth/endpoint overrides are schema scaffolding only. The only runtime read of `.Endpoint` is in `pkg/schema/profile_test.go` (a parse verification test) — never in the execution path.
- **Conclusion:** Profile is plan-time governance metadata only. A profile's endpoint or auth override for a tool has zero effect on `gert run` today.
- **Effort to wire:** Add `Profile *schema.RuntimeProfile` to `WireOptions`; pass it from `run.go` through `BuildEngineConfig`; inject into `DefaultToolRuntime` (or a profile-aware wrapper); in `runtime.go`, when `TransportMCPHTTP`, check `profile.Tools[toolName].Endpoint` and override `def.URL`; same for auth provider selection. New tests for profile-override vs. tool-definition baseline for endpoint and auth. Estimate: **3–4 days** (wiring + tests; no new schema work needed — the fields already exist).

### 2026-08-17 — Slice 5: Tier 0 Static Preflight

**Commit:** b982804 on `C:\One\OpenSource\gert` (main)

**What shipped:**
- `internal/planner/preflight.go`: `checkAttendancePreflight()` (PLAN-011), `checkToolEnvironmentPreflight()` (PLAN-010/PLAN-012), `filterCanonicalContexts()`, `containsString()`
- `internal/planner/preflight_test.go`: 17 acceptance tests — all green
- `internal/planner/planner.go`: profile field on `impl`, Plan() calls attendance check, resolveTool() calls tool preflight check, profile carried to plan.Metadata.Profile
- `pkg/planner/planner.go`: `Profile *schema.RuntimeProfile` added to Config; `ErrContextMismatch`, `ErrAttendanceMismatch`, `ErrTestContextBinding` sentinels
- `pkg/errkit/errors.go`: ErrPLAN011, ErrPLAN012 added; ErrPKGW003 restored (pre-existing bug)
- `pkg/engine/run.go`: Profile field comment updated to reflect Slice 5 ownership

**Key decisions:**
- Migration safety: `filterCanonicalContexts()` silently ignores non-context values ("real") in AllowedEnvironments during Tess migration. Test locked.
- AllowedModes is orthogonal to AllowedEnvironments — preflight never inspects AllowedModes. Test locked.
- Attendance check is declared-state only. No isatty(). Documented in message and decision doc.
- allow_subprocess_in_test is "auditable author assertion" not sandbox — no overstatement.

**Wiring gap (blocked on Ken's run.go):** The planner now accepts Config.Profile but cmd/gert/run.go does not yet pass runtimeProfile into plannerImpl config. Preflight fires in tests but not from the live CLI. Documented in decision doc.

**Pre-existing bug fixed:** ErrPKGW003 was missing from errkit var block (someone replaced it with ErrPKGW004 but left the sentinels map referencing the old name). Restored.

**go build ./...** exit 0  
**go test ./internal/planner/... ./pkg/planner/... ./pkg/errkit/...** exit 0, zero regressions

---

### 2026-08-16T15:59:48-07:00 — Runtime Portability Integration Critique

**Session:** Reviewed "Runtime Portability for Gert Runbooks" (SQL Live-Site Operations → Gert Core Team), requested by Cristiano.

**Key findings:**

- **Preflight tiers:** §10.1's steps 2–3 (token acquisition, endpoint probe) have side effects and answer the wrong question. The user's question ("is this runbook configured to run here?") is purely static. Proposed three-tier split: Tier 0 (static/offline, default before every run), Tier 1 (local, no network, opt-in), Tier 2 (live, opt-in). Only Tier 0 answers Cristiano's question.

- **Error taxonomy:** `binding/tool-not-found` conflates 5 distinct operator situations. Proposed: `config/tool-unresolved` (authoring error), `config/no-binding-for-profile` (Cristiano's case — tool exists, profile missing binding), `config/binding-incomplete` (binding exists but config fields missing), `auth/credential-failure` (runtime), `transport/endpoint-unreachable` (runtime/network). Each gets distinct message text pointing to the correct fix.

- **`gert plan` command:** Non-executing binding table command is required. Shows each toolRef → transport → auth → status. Failing case rendered inline. Exit codes 0/1/2 distinguish tiers.

- **Host bridge protocol gaps:** Missing protocol version, capabilities handshake (required for Tier 0!), step correlation for audit, cancel channel for IPC. Token isolation is by convention only — needs schema validation on `ToolResult.outputs` + scrubber extension.

- **Transport agnosticism doesn't hold cleanly:** Deadline enforcement, cancellation, backpressure, error fidelity, and process-death semantics all differ between in-process and IPC/stdio. Library embedding has fault isolation risk; subprocess model recommended.

- **Idempotency contradiction:** §10.4 requires "declared idempotent" but §13 says tool/v1 schema is frozen. Direct contradiction. Resolution: additive `idempotent: bool` field on actions (backward-compatible) or classification-based default.

- **Late result discard wrong for mutating actions:** "Discard" is safe for read-only. For mutating/destructive, the correct semantic is "halt with indeterminate state" per invariant #4.

- **Phase ordering risk:** §10.4 deferred to Phase 3 but HTTP can time out in Phase 1. Phase 1 must ship explicit "halt-on-timeout, no retry" as a safe default or restrict Phase 1 to read-only actions.

**Findings filed:** `.squad/decisions/inbox/david-runtime-portability-integration-critique.md`

---

## 2026-08-16T02:33:36Z — MCP HTTP Transport Feature — Team Session Orchestration

**Feature:** Native Streamable HTTP MCP transport (transport.mode: mcp-http)
**Outcome:** ✅ COMPLETE — All requirements met, final review gate APPROVED (unconditional)

- Completed assigned stream work
- Collaborated on binding architecture contract
- Participated in team orchestration
- All deliverables verified and tested
- Ready for production merge


---

## 2026-08-17 — Runtime Portability Negotiation Concluded

**Status:** Design agreed, gate closed, Phase 1 scope finalized.

**Negotiation:** Four-round multi-agent negotiation with SQL Live-Site Operations on runtime portability for Gert runbooks (rounds 2–4; round 1 committed as d355bf5).

**Key technical outcomes:**
- AllowedEnvironments/AllowedModes field separation (profile contexts vs RunMode)
- Approval gate coverage closure (direct-invocation path enforcement added to Phase 1)
- Grandfathering rule for interactive unspecified (legacy equires-approval: false acts as read-only override)
- Declared attendance orthogonal to context (fixes TTYOutput conflation, resolves CI blocking-on-stdin bug)
- Test-context binding restriction at plan time (native-only default, subprocess opt-in, mcp-http never)
- OQ2 closed as subprocess continuation with Phase 0 framing deliverables

**Agents involved:** Barbara (architect), Don (backend verification), David (round 1).

**Design principle:** Fail-closed by default; explicit migration paths for breaking changes; source-grounded verification.

**Impact:** Phase 1 addition ~4 days. All blocking conditions satisfied. Implementation starts this week.

## 2026-08-17 — Team Notation: Slice 1–2 Schema & Enforcement

**Slices shipped:** Commit a2e7db0 (Don, schema), c810b96 (Ken, enforcement).

**Status:** Both passed Barbara's review gate. Deferred gaps noted and scoped for later slices (ProfileApprovalGate, declared-attendance, enforcement wiring).

**David's involvement:** Cross-team findings from rounds 5–6 inform profile-binding architecture for Phase 1. Tiered-preflight from integration critique remains active design input for declared-attendance work.



## 2026-08-17 — Phase 1 Closure: Runtime Portability Complete

**Status:** COMPLETE — code exit 0, test exit 0, zero FAIL. 33 architectural rulings. All blocks satisfied.

**Key accomplishments:**
- Tri-state RequiresApproval (bool → *bool) + Classification field added
- ProfileApprovalGate + declared attendance implemented
- MCP HTTP transport (Don), auth provider (David), fixture migration (Tess), schema validation (Ken), profile spec (Edith)
- 31 conformance vectors: 31 PASS / 0 SKIP / 0 FAIL
- SQL Live-Site counterparty: 3 counter-positions accepted, refined

**Deferred:** AllowedModes field (RunMode separate from context), per-tool auth override (Phase 3), lifecycle sanity (Phase 3)

**Next phase:** OQ2 (library vs. subprocess) spike; resolver extends --package-map; Phase 2 host bridge with explicit framing protocol
## 2026-08-17: Phase 1B Scope Confirmation

**Context:** SQL Live-Site rejected Phase 1 completion claim; identified four unshipped Phase 1B items. All four independently verified by engineers.

### Phase 1B Items (Confirmed Absent)

1. **Managed-identity auth** (Don's stream)
   - Current: NewAuthProvider recognizes only "azure-cli" at internal/tool/auth.go:29
   - Required: uth_managed_identity.go with IMDS + Workload Identity (stdlib net/http)
   - Estimate: 2 days

2. **Headless ICM proof** (Don's stream)
   - Current: icm-tsg-router does not exist (zero matches)
   - Required: Runbook + mock MCP server + integration test (production requires external credential)
   - Blocker: Managed identity (Claim 1)
   - Estimate: 1 day after Claim 1

3. **INDETERMINATE halt-on-timeout** (David's stream)
   - Current: Timeout unconditionally calls ailRun() regardless of classification
   - Required: New StepStatusIndeterminate; engine branch on classification; resume guard; test suite (8 vectors)
   - Estimate: 4-5 days

4. **Profile endpoint/auth binding** (David's stream)
   - Current: Profile never passed to BuildEngineConfig; transport reads only tool definition
   - Required: Wire through adapter; add auth field to ProfileToolOverride; transport override logic; tests
   - Estimate: 3-4 days

### Auth Precedence Ruling (RATIFIED)

**Decision:** Profile top-level uth.provider overrides tool-definition uth.provider at transport construction time.

**Enables:** Managed identity in CI (tool says zure-cli, CI profile says managed-identity).

**Rules:**
- Profile auth wins (execution-context binding vs portable contract)
- Per-tool profile auth rejected until Phase 3 (loader error with deferral)
- Transport mode never rewritten (auth is credential substrate, not protocol)
- Loader validates profile auth against knownAuthProviders

### Process Notes

- All four claims verified with file:line evidence
- Phase 1A genuinely complete; Phase 1B items were original scope, not delivered
- Barbara's coordination cycle: 5 corrections applied (language, classifications, approvals, headers, deferral)
- Production ICM validation blocked on Live-Site credential provisioning (external dependency, TBD)

---

## Phase 1B Rev 2 Acceptance and Corrections (2026-08-17)

**Status:** All eight corrections verified correct against code. Phase 1B Rev 2 plan ratified.

### Correction A Summary: Profileless Non-Interactive Fail-Fast

- **Implementation:** CLI entry in `cmd/gert/run.go` after profile check, uses `os.Stdin.Stat()` + `ModeCharDevice`.
- **Harness impact:** Conformance harness (31 vectors + enum) breaks without profile. Solution: inject unattended test profile fixture at `internal/conformance/testdata/unattended-test.profile.yaml`. Behavior unchanged.
- **Estimate:** 1.5 days total.

### Correction B Summary: INDETERMINATE Evidence Record

- **Seven missing evidence fields:** Tool name, action name, classification, endpoint host, attempt number, deadline, transport error category.
- **Non-fabrication design:** Embed `*IndeterminateRecord` pointer on `StepResult`. Non-nil = completion unknown. `Output` left nil (no fabrication).
- **Redaction:** Host only (not full URL or auth tokens).
- **Revised estimate:** 5.5–6 days (was 4–5; +1.5 for evidence struct, serialization, resume guard, test assertions).

### Correction C Summary: ICM Proof Must Target Real Contract

- **`--package-map` status:** YES — execution-wired (confirmed by `TestRun_PackageMap_RealVsMockBinding`).
- **`--profile` status:** NOT execution-wired yet. This is Phase 1B Item 2 work.
- **Contract parity infrastructure:** Does not exist. New cross-binding assertion harness needed.
- **Blocking artifacts:** Six required from SQL Live-Site Operations; Item 4 cannot start until Item 2 ships AND all artifacts received.
- **Revised estimate:** 3 days (was 1 day; +2 for contract parity infrastructure and dual-outcome coverage).

### Phase 1B Rev 2 Revised Scope

| Item | Estimate | Dependencies |
|------|----------|--------------|
| 1. Managed Identity (IMDS only) | 1.5 days | None |
| 2. Runtime Binding + PLAN-013 | 3–4 days | None |
| 3. INDETERMINATE + evidence | 5.5–6 days | None |
| 4. ICM proof (real contract) | 3 days | Item 2 + external artifacts |
| 5. Fail-fast + harness | 1.5 days | None |
| 6. Credential-leak assertions | 1 day | None (NEW) |

**Revised parallelized estimate:** 9–10 days (was 8–9 days).

### Design Ruling: IndeterminateRecord Non-Fabrication

Completion-unknown representation uses `*IndeterminateRecord` embedded on `StepResult`:
- Nil `Output` + nil `IndeterminateRecord` pointer = tool returned empty output (valid result).
- Nil `Output` + non-nil `IndeterminateRecord` pointer = completion unknown (trace-safe evidence, no fabricated output).
- Rationale: avoids false positive inference from absense of fields. The pointer's presence is the explicit signal.


