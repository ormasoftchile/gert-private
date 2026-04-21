# Brian — Phase 19 Implementation Summary

**Date:** 2026-04-21
**Phase:** 19
**Status:** ✅ Complete

---

## Part A — Rate Limiting

### Files Created
- `v2/internal/serve/ratelimit.go` — `newRateLimitMiddleware`, `ipLimiters`, `rateLimiterEntry`, `extractIP`
- `v2/internal/serve/ratelimit_test.go` — 7 tests

### Files Modified
- `v2/pkg/serve/serve.go` — added `RateLimit int` to `ServerConfig`
- `v2/cmd/gert/serve.go` — added `--rate-limit` flag (int, default 0)
- `v2/internal/serve/middleware.go` — inserted `newRateLimitMiddleware(s.cfg.RateLimit)` between CORS and Auth
- `v2/go.mod` / `v2/go.sum` — added `golang.org/x/time v0.15.0`

### Tests (all pass under `-race -count=1`)
1. `TestRateLimit_Disabled` — limit=0 passes all requests
2. `TestRateLimit_BelowLimit` — requests within limit pass (200)
3. `TestRateLimit_ExceedsLimit` — limit=1, burst=2; 3rd request gets 429
4. `TestRateLimit_BurstAllowed` — limit=3, burst=6; all 6 burst pass, 7th is 429
5. `TestRateLimit_HealthExempt` — /health always 200 even when limited
6. `TestRateLimit_PerIP` — different IPs have independent buckets
7. `TestRateLimit_XForwardedFor` — XFF header used for IP extraction

### Middleware chain order (CORS → RateLimit → Auth):
```go
newCORSMiddleware(s.cfg.AllowedOrigins),
newRateLimitMiddleware(s.cfg.RateLimit),  // NEW
newBearerAuthMiddleware(s.cfg.BearerToken, s.cfg.BearerTokenExpiry, s.cfg.JWTSecret),
```

---

## Part B — E2E Parallelization

Added `t.Parallel()` as first statement in all 11 E2E test functions in `v2/internal/e2e/e2e_test.go`.

(Ken's design listed 12 tests but the file has 11 — `TestE2E_CancelMidRun` is the 11th; the design count was approximate.)

---

## NBI-17-02

No implementation needed (WONT_FIX per Ken).

---

## Test Results

```
go test ./... -race -count=1 -timeout=180s
```

All packages pass. No race conditions detected.

---

## Deviations

None. Implementation matches Ken's design exactly.
