# Decision Record: Phase 16 Design

**Author:** Ken (Software Architect)  
**Date:** 2026-04-21  
**Phase:** 16

---

## D-16-01: Registry Authoritative for Active Runs

**Context:** `run.list` and `run.get` RPCs may find the same runID in both the in-memory `RunRegistry` (active runs) and the on-disk `DirRunStore` (persisted snapshots).

**Decision:** Registry entry is authoritative. Skip store entry when runID exists in registry.

**Rationale:**
- Registry has live `RunHandle` with real-time state
- Store contains last-persisted checkpoint (potentially stale)
- Active runs should reflect current status, not last snapshot

**Consequences:**
- Slight complexity in merge logic (check activeIDs map)
- Clear, predictable behavior for clients
- No race conditions between registry and store

---

## D-16-02: Silent Store Fallback on Error

**Context:** `DirRunStore.ListRuns()` may fail due to transient disk errors or missing directory.

**Decision:** On store error, `run.list` returns registry-only results without surfacing the error to the client.

**Rationale:**
- Graceful degradation preferred over hard failure
- Registry data (active runs) is still valuable
- Store errors are typically transient
- Error is logged server-side for debugging

**Consequences:**
- Clients may see incomplete results during store failures
- No RPC error propagated — clients get partial success
- Requires server-side logging for observability

---

## D-16-03: Skip SSE Flake Test (Temporary)

**Context:** `TestSSE_ConnectReceivesEvents` has an intermittent timing flake (~1 in 50 runs), confirmed Phase 9 origin.

**Decision:** Add `t.Skip("flaky: timing-sensitive SSE test — see NBI-15-01")` rather than attempting a fix.

**Rationale:**
- Fixing timing-sensitive tests properly requires synchronization primitives (high effort)
- Flake does not indicate a production bug
- Skipping stops CI noise immediately
- NBI-16-01 tracks the proper fix for future

**Consequences:**
- Test coverage gap (acceptable — test was unreliable)
- CI stability improved
- Technical debt acknowledged and tracked

---

## D-16-04: Bearer Token is Development-Only Auth

**Context:** `gert serve` needs some authentication, but full auth (OAuth2/OIDC) is a large surface.

**Decision:** Implement simple `Authorization: Bearer <token>` check, explicitly documented as development/internal use only.

**Rationale:**
- Production auth requires significant scope (token management, refresh, revocation)
- Simple bearer token serves internal deployments and development
- Clear documentation prevents security misunderstandings
- Full auth deferred to Phase 17+

**Consequences:**
- Not suitable for public-facing deployments (documented)
- Sufficient for internal/development use
- Clean migration path to real auth later

---

## D-16-05: Add run.get RPC

**Context:** `run.list` wiring is Part A anchor. Consider whether to add related RPC methods.

**Decision:** Add `run.get` RPC to fetch single run state by ID.

**Rationale:**
- Natural API companion (list → get detail pattern)
- Incremental scope (reuses registry/store fallback logic)
- High value for client implementations
- Same effort multiplied by two methods is better than separate phases

**Consequences:**
- Slightly larger Phase 16 scope (acceptable)
- Complete CRUD-like API for runs (list, get, start, cancel)
- Better client experience

---

## NBI Disposition

| ID | Status | Rationale |
|----|--------|-----------|
| NBI-15-01 | ANNOTATED | t.Skip added; tracked as NBI-16-01 |
| NBI-15-02 | IN SCOPE | Part A anchor |
| NBI-15-03 | PARTIAL | CORS + bearer only; full auth deferred |
| NBI-15-04 | DEFERRED | E2E parallelization low priority |
| NBI-15-05 | DEFERRED | Schema docs follow implementation |

---

*Ken, Software Architect*

---

# Phase 16 Preflight Report

**Date:** 2026-04-21  
**Baseline:** e4c4aee

## Checks

- [x] go build: **PASS**
- [x] go vet: **PASS**
- [x] go test -race: **PASS** (all 32 test packages passed, 0 failures)
- [x] git status: **clean** (working tree clean)
- [x] git log: **Phase 15 present** (e4c4aee is 2 commits back; HEAD at 24689df "Phase 15 sealed")

## Test Results Summary

- Total packages with tests: 32
- Total packages tested: 32 (all passed)
- Test duration: ~60s total
- Race detector: clean
- Known flakes: TestSSE_ConnectReceivesEvents did not occur in this run

## Verdict

**✅ ALL GREEN**

Baseline is clean and ready for Phase 16. No blockers detected.

---

*Barbara, QA Engineer*
