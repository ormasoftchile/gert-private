# Project Context

- **Project:** gert
- **Created:** 2026-03-17

## Core Context

Agent Scribe initialized and ready for work.

## Recent Updates

📌 Team initialized on 2026-03-17

## Learnings

Initial setup complete.
- 2026-03-17 visual editor planning batch logged: wrote 5 orchestration entries, created a session log, merged 5 decision inbox files into `decisions.md`, and appended cross-agent alignment notes to active agent histories.
- `.squad/` is gitignored in this repository, so Scribe changes remain local unless force-added or ignore rules change.

## Phase 18 Kickoff

- **Commit:** 83c741e
- **Date:** 2026-04-21
- **Summary:** Phase 18 kickoff commit — Ken design artifacts staged and committed before Brian implementation phase
- **Files committed:** .squad/agents/barbara/history.md, .squad/agents/ken/history.md, .squad/tmp/ken-phase18-design.md
- **Design anchors:** JWT HMAC-SHA256, run.delete RPC, WebSocket timing flake fix

## Phase 18 Final Commits

- **Commits:** `24d863e` (v2 implementation) + `66c4676` (.squad state)
- **Date:** 2026-04-21T14:12:35Z
- **Status:** SEALED — Ken APPROVED
- **Summary:** Phase 18 complete — JWT signature verification (NBI-17-01), run.delete RPC (NBI-17-03), WS timing flake fixed (NBI-17-05)
- **Tests:** All pass under -race -count=3
- **Next:** Phase 19 queue established with NBI-17-02 (token rotation), NBI-16-03 (E2E parallelization), NBI-17-04 (rate limiting)

## Phase 19 Kickoff

- **Commit:** `b15ccbe`
- **Date:** 2026-04-21
- **Summary:** Phase 19 kickoff commit — Ken design artifacts + Barbara preflight validation
- **Files committed:** .squad/agents/barbara/history.md, .squad/agents/ken/history.md, .squad/agents/scribe/history.md, .squad/identity/now.md, .squad/tmp/ken-phase19-design.md
- **Design anchors:** 
  - Part A (ANCHOR): Per-IP rate limiting via golang.org/x/time/rate, --rate-limit flag
  - Part B: E2E test parallelization (t.Parallel safe — no shared state)
  - NBI-17-02: Closed WONT_FIX (short expiry + secret rotation is sufficient)
- **Validation:** Barbara ALL GREEN on 24d863e baseline

## Phase 19 Final Commits

- **Commits:** `c0f9189` (v2 implementation) + `75055b8` (.squad state)
- **Date:** 2026-04-21T14:22:10Z
- **Status:** SEALED — Ken APPROVED
- **Summary:** Phase 19 complete — Per-IP rate limiting (NBI-17-04), E2E test parallelization (NBI-16-03), NBI-17-02 closed WONT_FIX
- **Part A (rate limiting anchor):** golang.org/x/time v0.15.0; newRateLimitMiddleware per-IP token bucket; burst=2*limit; 10k entry LRU cap; 5-min stale cleanup; /health exempt; middleware order CORS→RateLimit→Auth; --rate-limit flag; 7 tests
- **Part B (E2E parallelization):** t.Parallel() on all 11 E2E tests; safe via t.TempDir() isolation
- **Tests:** All pass under -race -count=3
- **Next:** Phase 20 queue TBD from Ken's Phase 19 review

## Auth Design Merge

- **Date:** 2026-04-25
- **Action:** Merged Ken's auth layer design (`ken-auth-design.md`) into `decisions.md` as "Auth Layer Design — gert-domain-home Full-Stack"
- **Source:** `.squad/decisions/inbox/ken-auth-design.md` (deleted after merge)
- **Summary:** Sign in with Apple (owner + delegate), home-api JWT (HS256, 24h), delegate invite flow (v1), GERT_SERVICE_TOKEN for gert serve, X-Test-Token Maestro bypass (testenv build tag only), v0 security constraints and DB schema additions.

## Auth v2 Decision Merge

- **Date:** 2026-07-21
- **Action:** Merged Ken's revised auth design (`ken-auth-revise.md`) into `decisions.md`, replacing "Auth Layer Design — gert-domain-home Full-Stack" with "Auth Layer Design v2 — Multi-Provider (Apple + Google)"
- **Source:** `.squad/decisions/inbox/ken-auth-revise.md` (deleted after merge)
- **Summary:** Added Google Sign-In alongside Apple. Unified `POST /auth/signin { provider, identity_token }` endpoint. Provider abstraction with JWKS verification (`lestrrat-go/jwx/v2`) for both providers. `GoogleSignIn-iOS` via SPM. `users` table now uses `(provider, provider_sub)` unique index — replaces `apple_sub`. Delegate invite flow is provider-agnostic. Both providers required in v0; account linking deferred to v1. New env var: `GOOGLE_CLIENT_ID`.
