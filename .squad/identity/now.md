---
updated_at: 2026-04-21T15:00:00Z
focus_area: gert v2 implementation — Phase 19 sealed, Phase 20 re-scoped
current_phase: 19 (SEALED)
next_phase: 20 (RE-SCOPED)
active_issues:
  - Phase 20 design corrected after codebase audit revealed Parts A and C were already shipped
---

# What We're Focused On

Implementing gert v2 phase-by-phase. Phases 0–19 complete and sealed.

**Phase 19 sealed:** `c0f9189` + `75055b8` — Per-IP rate limiting, E2E parallelization (APPROVED by Ken)
- Part A (NBI-17-04): newRateLimitMiddleware with per-IP token bucket (golang.org/x/time v0.15.0); burst=2*limit; 10k entry LRU cap; 5-min stale cleanup; /health exempt; middleware order CORS→RateLimit→Auth; --rate-limit flag; 7 tests
- Part B (NBI-16-03): E2E parallelization — t.Parallel() on all 11 tests; safe via t.TempDir() isolation
- NBI-17-02: Closed WONT_FIX (short expiry + secret rotation sufficient for JWT lifecycle)
- Validation: go test ./... -race -count=3 — all tests ✅

**Phase 20 queue (RE-SCOPED):** Original design was stale — Parts A and C (run.delete, WS flake) were already shipped in Phase 18 (commit 24d863e). Corrected scope:

| Part | NBI | Item |
|------|-----|------|
| A (ANCHOR) | NBI-16-05 + NBI-16-07 | `completedAt` surfacing in `run.get`/`run.list` + `handleRunGet` godoc + testdata fixtures |
| B | NBI-16-06 | Multiple CORS origins — `--cors-origin` repeatable flag |
| C | — | `--trust-proxy-headers` security flag — XFF trust must be opt-in (rate-limit bypass fix) |

**Budget:** ~1.4 Brian-days  
**Design:** `.squad/tmp/ken-phase20-design.md` (re-scoped)  
**Decision record:** `.squad/decisions/inbox/ken-phase20-design.md` (re-scoped)

Team: Ken (architect/reviewer), Brian (Go implementor), Barbara (preflight/integrations), Scribe (logger).
