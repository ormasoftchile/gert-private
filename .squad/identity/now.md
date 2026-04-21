---
updated_at: 2026-04-21T14:12:35Z
focus_area: gert v2 implementation — Phase 18 sealed, Phase 19 next
current_phase: 18 (SEALED)
next_phase: 19
active_issues:
  - NBI-17-02: Token rotation/revocation (carry-forward for Phase 19)
  - NBI-16-03: E2E test parallelization (carry-forward for Phase 19)
---

# What We're Focused On

Implementing gert v2 phase-by-phase. Phases 0–18 complete and sealed.

**Phase 18 sealed:** `24d863e` + `66c4676` — JWT signature verification, run.delete RPC, WS flake fixed (APPROVED by Ken)
- Part A (NBI-17-01): HMAC-SHA256 JWT signature verification with --auth-jwt-secret flag; verifyJWT checks alg=HS256, signature first, then exp/iat
- Part B (NBI-17-03): handleRunDelete RPC; running runs cannot be deleted (rpcRunDeleteRunning=-32020); 4 new tests
- Part C (NBI-17-05): TestWS_RunCompleted_ReceivesTerminal fixed with WaitForSubscriber before Broadcast
- Security: --auth-token and --auth-jwt-secret mutually exclusive; hmac.Equal for constant-time comparison
- 5 new auth tests + updated JWT tests using signed tokens; all pass under -race -count=3
- Validation: go test ./... -race -count=3 — all tests ✅

**Phase 19 queue (next):** NBI items from Phase 18 review:

**Priority items:**
- **NBI-17-02**: Token rotation/revocation (JWT lifecycle management)
- **NBI-16-03**: E2E test parallelization (from Phase 16 carry-forward)
- **NBI-17-04**: Rate limiting for `gert serve`
- Any additional items from Ken's Phase 18 review notes

Team: Ken (architect/reviewer), Brian (Go implementor), Barbara (preflight/integrations), Scribe (logger).
