# Session Log — Phase 1B Planning

**Timestamp:** 2026-08-17T16-55-53Z
**Session:** phase1b-planning
**Duration:** 2 turns (don, david) + 1 turn (barbara coordination)

---

## Executive Summary

SQL Live-Site rejected our "Phase 1 complete" claim. The rejection was correct. We overclaimed. They identified four unshipped Phase 1 items; all four independently confirmed by our own engineers with file:line evidence. Phase 1B scope is now defined; Phase 1A/1B split is ratified.

---

## SQL Live-Site Decision

**Status:** Accepted Phase 1A, rejected Phase 1 completion claim.

The four claims they issued have now been verified by our engineers:

1. **Managed-identity auth absent** (Don) — CONFIRMED
   - NewAuthProvider at internal/tool/auth.go:29 recognizes only "azure-cli"
   - managed-identity falls to default case, returns MCP-002
   - Static validation rejects it at parse time
   - No Azure SDK in go.mod; implementation can use stdlib net/http only

2. **Headless ICM proof absent** (Don) — CONFIRMED
   - icm-tsg-router does not exist (zero matches across all files)
   - 	ools/icm.tool.yaml exists with three actions but unintegrated
   - No runbook invokes it via managed identity
   - Read-only proof requires: managed identity provider + runbook + mock MCP server + integration test
   - Production requires: Live-Site credential grant (external dependency, TBD)

3. **INDETERMINATE halt-on-timeout absent** (David) — CONFIRMED
   - INDETERMINATE state does not exist (zero codebase occurrences)
   - Step status enum has 7 values; indeterminate is not one of them
   - Contract.Idempotent field declared but never read at runtime
   - Timeout handling unconditionally calls ailRun() regardless of action classification
   - Today: destructive action timeout == read-only action timeout (both fail immediately)
   - Contract requires: classify action, mark INDETERMINATE on timeout, halt run, require operator verification

4. **Profile endpoint/auth binding absent** (David) — CONFIRMED
   - --profile flag parsed but profile never passed to BuildEngineConfig
   - Transport construction reads ONLY from tool definition
   - ProfileToolOverride.Endpoint in schema but only read in test, never in execution
   - Profile consulted only for approval scope (governance), not transport binding
   - Implementation requires: profile wiring through wire.go, schema update for auth field, transport override logic

---

## Claim Severity Assessment

This is the **third recurrence** of a reachability bug class:

1. **AllowedEnvironments** (Phase 1A scope) — declared field, dead code, months unread
2. **Tier 0 checks** (David's Phase 1A verification) — wired to nothing in execution path
3. **Profile endpoint/auth binding** (Claim 4, Phase 1B) — declared field, unread in execution path

We have a pattern: declaring schema fields in governance layers without wiring them through to transport/execution. Barbara's Phase 1B plan addresses this with explicit transport-layer cross-checks.

---

## Phase 1B Scope

### Claims 1 & 2 (Don's Stream)

| Item | Estimate | Blocker | Notes |
|---|---|---|---|
| Managed identity auth provider | 2 days | None | New file uth_managed_identity.go; IMDS + Workload Identity; stdlib only |
| Headless ICM proof | 1 day | Managed identity | Runbook + mock MCP + integration test (production requires external credential) |
| **Combined** | **~2-3 days** | — | Managed identity unblocks ICM proof |

### Claims 3 & 4 (David's Stream)

| Item | Estimate | Blocker | Notes |
|---|---|---|---|
| INDETERMINATE status + halt-on-timeout | 4-5 days | None | New status enum value; engine timeout path branch; resume guard; test suite (8 vectors) |
| Profile endpoint/auth wiring | 3-4 days | None | Wire through adapter; schema auth field; transport override; tests |
| **Combined (sequential)** | **7-9 days** | — | Can be parallelized |
| **Combined (parallelized)** | **4-5 days** | — | Two engineers on separate streams |

### Production Blockers

- **ICM validation:** Live-Site credential provisioning (external dependency, timeline TBD)

---

## Auth Precedence Ruling (RATIFIED)

**Decision:** Profile top-level uth.provider overrides tool-definition uth.provider at transport construction time.

This enables managed identity in CI: tool definition declares zure-cli (interactive), CI profile declares managed-identity (headless). Rationale is service principal selection pattern from deployment environments.

---

## Process Notes

1. **Coordinator review cycle:** Barbara's draft underwent 5 corrections (language softening, ICM classification fixes, approval matrix flattening, header redaction, AllowedModes deferral to Phase 2). All applied.

2. **Evidence standard:** All four Live-Site claims verified with file:line citations, not speculation.

3. **Overclaim analysis:** Phase 1A genuinely completed (governance + profile foundation). Phase 1B items were in the original Phase 1 scope but not delivered. The team moved faster on Phase 1A than on Phase 1B integration.

---

## Artifacts Generated

- gert-core-phase1b-plan.md — Phase 1B scope and implementation strategy
- gert-core-phase1-status-report.md — Updated status with Phase 1A/1B split
- .squad/decisions.md — Merged three inbox decisions (don/david/barbara scope + auth ruling)
- .squad/orchestration-log/{timestamp}-don.md — Don's findings summary
- .squad/orchestration-log/{timestamp}-david.md — David's findings summary
- .squad/orchestration-log/{timestamp}-barbara.md — Barbara's coordination log
- .squad/decisions/archive/archive-2026-08-17T09-55-00Z.md — Archived decisions (194KB → 157KB after merge)

---

## Health Notes

- decisions.md was 194KB (well over 150KB soft cap) before merge; archiving applied size-based cap
- All inbox files (3) successfully merged and deleted
- No deduplication needed (distinct decision areas: claims 1-4, auth precedence)
- Phase 1B timeline now determined; production credential provisioning remains external blocker
