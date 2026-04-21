---
updated_at: 2026-04-21T13:30:01Z
focus_area: gert v2 implementation — Phase 16 sealed, Phase 17 next
active_issues:
  - NBI-16-01: Use subtle.ConstantTimeCompare for bearer token validation (anchor for Phase 17)
---

# What We're Focused On

Implementing gert v2 phase-by-phase. Phases 0–16 complete and sealed.

**Phase 16 sealed:** `ab6f554` — run.list/run.get RPC wiring, CORS, bearer auth (APPROVED 8/10 by Ken)
- Part A (NBI-15-02): run.list merges registry + store, deduplication by RunID; run.get with 404 path
- Part B (NBI-15-03): CORS middleware (--cors-origin), bearer auth (--auth-token), gert serve CLI
- All 14 new tests pass; 4 deviations accepted; 1 non-blocking security item (NBI-16-08)
- Validation: go test ./... -race -count=3 — all 32 packages pass

**Phase 17 starting next:** NBI items from Phase 16 review:

**Anchor Item:**
- **NBI-16-01**: Use `subtle.ConstantTimeCompare` for bearer token validation (prevents timing attacks)

**Carry-forward items:**
- NBI-15-01: E2E test parallelization (low priority)
- NBI-16-02: Full auth hardening for production (OAuth2/OIDC/API keys)
- NBI-16-05: run.list RPC schema documentation

Team: Ken (architect/reviewer), Brian (Go implementor), Barbara (preflight/integrations), Scribe (logger).
