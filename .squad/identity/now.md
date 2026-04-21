---
updated_at: 2026-04-21T15:30:00Z
focus_area: gert v2 implementation — Phase 20 sealed, v2.0 MVP complete
current_phase: 20 (SEALED)
next_phase: 21 (TBD) or v2.0 tag
active_issues:
  - NBI-20-01: handleRunList godoc missing completedAt field (non-blocking, low priority)
  - NBI-18-01: OAuth2/OIDC external token validation (v2.1)
  - NBI-16-06 multiple CORS origins: DONE (Phase 20 Part B)
  - NBI-16-07 completedAt persisted: DONE (Phase 20 Part A)
---

# What We're Focused On

Implementing gert v2 phase-by-phase. Phases 0–20 complete and sealed.

**Phase 20 sealed:** `f60f6b6` — completedAt surfacing, repeatable CORS origins, trust-proxy-headers (APPROVED by Ken)
- Part A (NBI-16-05, NBI-16-07): CompletedAt added to RunState; surfaced in run.get persisted + run.list active/persisted paths; 3 tests + 4 testdata fixtures
- Part B (NBI-16-06): repeatedStringFlag; --cors-origin repeatable; 4 tests
- Part C (security): --trust-proxy-headers flag; XFF trust default-off; prevents rate-limit IP-forge bypass; 2 tests
- Deviation accepted: RunState.CompletedAt didn't exist — Brian added it to pkg/engine/run.go and wired in engine.go
- Validation: go test ./... -race -count=1 — 31 packages ✅

**Phase 20 closes:**
- NBI-17-03 run.delete: CLOSED Phase 18
- NBI-17-05 WS timing flake: CLOSED Phase 18
- NBI-16-05 API schema docs: CLOSED Phase 20
- NBI-16-07 completedAt persisted: CLOSED Phase 20
- NBI-16-06 multiple CORS origins: CLOSED Phase 20

**Open NBI (carry-forward):**
- NBI-20-01: handleRunList godoc missing completedAt (low priority, non-blocking)
- NBI-18-01: OAuth2/OIDC (v2.1)
- NBI-16-06 --cors-origin: DONE

**Phase 21 / v2.0 tag:** Ken described Phase 20 as completing the v2.0 MVP feature set. Consider tagging v2.0 or opening Phase 21 for NBI-20-01 + any new items.

Team: Ken (architect/reviewer), Brian (Go implementor), Barbara (preflight/integrations), Scribe (logger).
