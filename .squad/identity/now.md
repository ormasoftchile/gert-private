---
updated_at: 2026-04-21T13:51:52Z
focus_area: gert v2 implementation — Phase 17 sealed, Phase 18 next
active_issues:
  - NBI-17-01: Full auth hardening (JWT signatures, key rotation) — anchor for Phase 18
  - NBI-17-02: E2E test parallelization (carry-forward from NBI-16-03)
---

# What We're Focused On

Implementing gert v2 phase-by-phase. Phases 0–17 complete and sealed.

**Phase 17 sealed:** `933cb57` — timing-safe auth, JWT expiry, SSE flake fixed (APPROVED 9/10 by Ken)
- Part A (NBI-16-01): `subtle.ConstantTimeCompare` for bearer token — timing attacks prevented
- Part B (NBI-16-02): JWT expiry validation via --auth-token-expiry flag; `exp` and `iat` claims checked
- Part C (NBI-15-01): `WaitForSubscriber` eliminates SSE timing flake; TestSSE_ConnectReceivesEvents deterministic
- Additional (NBI-16-04): handleRunList godoc with full schema
- 6 new middleware tests; 1 test restored; all pass under -race -count=3
- Validation: go test ./... -race -count=3 — 156 tests ✅

**Phase 18 starting next:** NBI items from Phase 17 review:

**Anchor Item:**
- **NBI-17-01**: Full auth hardening — JWT signature verification, API key rotation (prevents forged tokens)

**Priority Carry-forward items:**
- NBI-17-02: E2E test parallelization (low priority, from Phase 16)
- NBI-17-03: run.delete RPC (CRUD completion)
- NBI-17-04: Rate limiting for `gert serve`
- NBI-17-05: Apply `WaitForSubscriber` to WebSocket timing flake (TestWS_RunCompleted_ReceivesTerminal)

Team: Ken (architect/reviewer), Brian (Go implementor), Barbara (preflight/integrations), Scribe (logger).
