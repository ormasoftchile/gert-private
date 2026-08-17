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