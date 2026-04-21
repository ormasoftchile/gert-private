# Ken — Phase 19 Design: Rate Limiting, E2E Parallelization, Token Rotation Decision

**Date:** 2026-04-21  
**Author:** Ken (Staff Architect)  
**Implementor:** Brian  
**Phase:** 19

---

## Overview

Phase 19 addresses the three NBI carry-forwards from Phases 16-18. After careful analysis, this phase focuses on **production hardening** of `gert serve` with rate limiting (Part A), **test infrastructure improvement** with E2E parallelization (Part B), and a **formal architectural decision** on token rotation (closed as WONT_FIX).

**Phase budget:** 2 Brian-days

---

## NBI Dispositions

| NBI ID | Item | Disposition | Rationale |
|--------|------|-------------|-----------|
| NBI-17-02 | Token rotation/revocation | **WONT_FIX** | Short expiry + secret rotation is sufficient; blocklist adds complexity without meaningful security benefit |
| NBI-16-03 | E2E test parallelization | **IN_SCOPE** (Part B) | Analysis confirms harness is isolation-safe; port allocation needs fix |
| NBI-17-04 | Rate limiting | **IN_SCOPE** (Part A) | Production hardening; simple token bucket per IP |

---

## NBI-17-02 Analysis: Token Rotation — WONT_FIX

### Current State

Phase 18 delivered:
- `--auth-jwt-secret` with HMAC-SHA256 signature verification
- `--auth-token-expiry` for maximum token age (iat claim check)
- Mutual exclusivity between `--auth-token` and `--auth-jwt-secret`

### Considered: In-Memory Token Blocklist

The proposed design would add:
- RPC method `auth.revoke` with `{"token": "..."}`
- Server-side `map[string]time.Time` for revoked tokens
- Automatic eviction when token's natural expiry passes
- ~100 LOC implementation

### Why WONT_FIX

**1. Security benefit is marginal:**
- With 15-minute token expiry (recommended), a stolen token has a 15-minute window
- Blocklist reduces this to "time until operator notices and revokes"
- Real-world compromise detection is rarely < 15 minutes
- If an attacker has a valid token, they've likely already exfiltrated what they need

**2. Operational complexity is high:**
- Blocklist is in-memory; server restart clears it anyway
- No persistence = no benefit across restarts
- Distributed deployments (multiple `gert serve` instances) would need shared state
- We'd need to add TTL management, cleanup goroutines, memory limits

**3. Better alternatives exist:**
- **Short expiry:** 5-15 minute tokens limit exposure window
- **Secret rotation:** Change `--auth-jwt-secret` to invalidate ALL tokens (same effect as restart)
- **External auth:** If revocation is critical, use an external identity provider (Keycloak, Auth0, etc.)

**4. Architectural principle:**
- gert serve is a thin adapter layer, not an identity provider
- JWT verification is "good enough" for the intended use case
- Adding stateful auth management crosses architectural boundaries

### Final Decision

**Status:** WONT_FIX (closed)

**Rationale documented in decision ledger:**
> "Token revocation via in-memory blocklist adds complexity without meaningful security benefit. The recommended deployment is short expiry (5-15 minutes) combined with secret rotation when compromise is suspected. If fine-grained revocation is required, deploy behind an identity-aware proxy (OAuth2 Proxy, Keycloak, etc.)."

---

## Part A — Rate Limiting (ANCHOR)

### Scope

Add per-IP rate limiting to `gert serve` endpoints using stdlib `golang.org/x/time/rate`.

**Endpoints:**
- `/rpc` — rate limited
- `/ws` — rate limited (connection establishment)
- `/events` — rate limited (connection establishment)
- `/health` — **exempt** (monitoring must not be rate-limited)

### Design

**Flag:** `--rate-limit N` (requests per second per IP, default: 0 = disabled)

**Implementation:**
- Token bucket algorithm via `rate.Limiter`
- Per-IP limiter stored in `map[string]*rate.Limiter`
- Cleanup goroutine evicts stale entries (no activity for 5 minutes)
- Memory cap: 10,000 IP entries; evict LRU on overflow

**New middleware:** `rateLimitMiddleware`

```go
// rateLimitMiddleware returns middleware that rate-limits requests per IP.
// limit=0 disables rate limiting. The limiter uses a token bucket with
// burst size of 2*limit to allow brief spikes.
func newRateLimitMiddleware(limit int) middleware {
    if limit <= 0 {
        return func(next http.Handler) http.Handler { return next }
    }
    
    limiters := &ipLimiters{
        m:       make(map[string]*rateLimiterEntry),
        limit:   rate.Limit(limit),
        burst:   limit * 2,
        maxSize: 10000,
    }
    go limiters.cleanupLoop()
    
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // /health exempt
            if r.URL.Path == "/health" {
                next.ServeHTTP(w, r)
                return
            }
            
            ip := extractIP(r)
            if !limiters.Allow(ip) {
                http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

type ipLimiters struct {
    mu      sync.Mutex
    m       map[string]*rateLimiterEntry
    limit   rate.Limit
    burst   int
    maxSize int
}

type rateLimiterEntry struct {
    limiter  *rate.Limiter
    lastSeen time.Time
}

func (l *ipLimiters) Allow(ip string) bool {
    l.mu.Lock()
    defer l.mu.Unlock()
    
    entry, ok := l.m[ip]
    if !ok {
        // Enforce memory cap
        if len(l.m) >= l.maxSize {
            l.evictOldest()
        }
        entry = &rateLimiterEntry{
            limiter: rate.NewLimiter(l.limit, l.burst),
        }
        l.m[ip] = entry
    }
    entry.lastSeen = time.Now()
    return entry.limiter.Allow()
}

func (l *ipLimiters) cleanupLoop() {
    ticker := time.NewTicker(60 * time.Second)
    for range ticker.C {
        l.mu.Lock()
        cutoff := time.Now().Add(-5 * time.Minute)
        for ip, entry := range l.m {
            if entry.lastSeen.Before(cutoff) {
                delete(l.m, ip)
            }
        }
        l.mu.Unlock()
    }
}

func (l *ipLimiters) evictOldest() {
    var oldestIP string
    var oldestTime time.Time
    for ip, entry := range l.m {
        if oldestTime.IsZero() || entry.lastSeen.Before(oldestTime) {
            oldestIP = ip
            oldestTime = entry.lastSeen
        }
    }
    if oldestIP != "" {
        delete(l.m, oldestIP)
    }
}

func extractIP(r *http.Request) string {
    // Trust X-Forwarded-For if present (first IP)
    if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
        if ip := strings.Split(xff, ",")[0]; ip != "" {
            return strings.TrimSpace(ip)
        }
    }
    // Fall back to RemoteAddr (strip port)
    host, _, _ := net.SplitHostPort(r.RemoteAddr)
    return host
}
```

### Configuration Changes

**File:** `pkg/serve/serve.go`

```go
type ServerConfig struct {
    // ... existing fields ...
    
    // RateLimit is the maximum requests per second per IP.
    // Default: 0 (disabled). Typical value: 10-100.
    RateLimit int
}
```

**File:** `cmd/gert/serve.go`

```go
cmd.Flags().IntVar(&cfg.RateLimit, "rate-limit", 0, 
    "requests per second per IP (0 = disabled)")
```

### Middleware Stack Position

Rate limiting goes **after** CORS (so OPTIONS requests are handled) and **before** auth (so rate-limited requests don't burn auth cycles):

```go
func (s *Server) withMiddleware(next http.Handler) http.Handler {
    stack := []middleware{
        requestIDMiddleware,
        loggingMiddleware,
        newCORSMiddleware(s.cfg.AllowedOrigins),
        newRateLimitMiddleware(s.cfg.RateLimit),  // NEW
        newBearerAuthMiddleware(s.cfg.BearerToken, s.cfg.BearerTokenExpiry, s.cfg.JWTSecret),
        recoveryMiddleware,
    }
    // ...
}
```

### Tests

1. **TestRateLimitMiddleware_Disabled** — limit=0 passes all requests
2. **TestRateLimitMiddleware_BelowLimit** — requests within limit pass
3. **TestRateLimitMiddleware_ExceedsLimit** — excess requests get 429
4. **TestRateLimitMiddleware_BurstAllowed** — burst (2x limit) allowed
5. **TestRateLimitMiddleware_HealthExempt** — /health never rate-limited
6. **TestRateLimitMiddleware_PerIP** — different IPs have separate limits
7. **TestRateLimitMiddleware_Cleanup** — stale entries evicted

**Estimated:** 150-200 LOC implementation, 150 LOC tests

---

## Part B — E2E Test Parallelization

### Analysis of Current State

**File:** `v2/internal/e2e/helpers_test.go`

The `E2EHarness` already creates per-test isolation:
- `t.TempDir()` creates isolated temp directory (line 69)
- `RunDir` and `TraceDir` are under that temp directory
- `Store` is per-harness (not shared)
- No global state is mutated

**Current tests in `e2e_test.go`:**
- 12 tests, all use `NewHarness(t, "...runbook.yaml")`
- Each creates its own engine, store, trace file
- No shared mutable state between tests

### The One Issue: Port Allocation (NOT applicable)

Reviewed `helpers_test.go` — E2E tests do **NOT** start HTTP servers. They:
1. Create an in-process engine via `adapter.BuildEngineConfig`
2. Run the engine directly via `eng.Start(ctx, plan, opts)`
3. Drive execution via `handle.Next(ctx)`

There is no port allocation in E2E tests. The serve layer is tested separately in `internal/serve/*_test.go`, which uses `httptest.NewServer` (automatic port allocation).

### Verdict: Parallelization is Safe NOW

**Action:** Add `t.Parallel()` to all 12 E2E tests.

```go
func TestE2E_SimpleEcho(t *testing.T) {
    t.Parallel()  // ADD THIS LINE
    h := NewHarness(t, "echo-runbook.yaml")
    // ...
}
```

Repeat for all 12 tests. No harness changes required.

### Tests to Parallelize

1. `TestE2E_SimpleEcho`
2. `TestE2E_VarInterpolation`
3. `TestE2E_BranchTrue`
4. `TestE2E_BranchFalse`
5. `TestE2E_IterateAll`
6. `TestE2E_IterateEarlyExit`
7. `TestE2E_ManualSkip`
8. `TestE2E_TracePersistence`
9. `TestE2E_ResumeFromCheckpoint`
10. `TestE2E_ToolStep`
11. `TestE2E_CancelMidRun`

**Estimated:** 15 minutes; add one line per test

### Validation

Run with race detector and high count to verify no races:

```bash
go test -v -race -count=5 ./v2/internal/e2e/...
```

---

## Implementation Order

1. **Part A (Rate Limiting):** ~1.5 Brian-days
   - Add `RateLimit` field to `ServerConfig`
   - Implement `newRateLimitMiddleware` in `middleware.go`
   - Wire into middleware stack
   - Add CLI flag `--rate-limit`
   - Write 7 tests
   - Verify with `go test -race -count=3 ./v2/internal/serve/...`

2. **Part B (E2E Parallelization):** ~0.5 Brian-days
   - Add `t.Parallel()` to all 12 E2E tests
   - Verify with `go test -v -race -count=5 ./v2/internal/e2e/...`
   - Measure speedup (expect ~3-4x with parallelization)

---

## Exit Criteria

### Part A — Rate Limiting

- [ ] `ServerConfig.RateLimit` field added
- [ ] `--rate-limit` CLI flag wired
- [ ] `rateLimitMiddleware` implemented with:
  - Per-IP token bucket via `golang.org/x/time/rate`
  - Memory cap (10k entries)
  - Cleanup goroutine (5-minute TTL)
  - /health exempt
- [ ] All 7 tests pass under `-race -count=3`

### Part B — E2E Parallelization

- [ ] All 12 E2E tests have `t.Parallel()`
- [ ] `go test -race -count=5` passes without races
- [ ] CI time for e2e package reduced (measure before/after)

### Decision

- [ ] NBI-17-02 closed as WONT_FIX in decision ledger

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Rate limiter memory leak | Low | Medium | 10k cap + cleanup goroutine |
| E2E race conditions | Low | Medium | Race detector verification |
| X-Forwarded-For spoofing | Medium | Low | Accept for now; document as known limitation |

---

## Appendix: X-Forwarded-For Trust Model

The rate limiter trusts `X-Forwarded-For` when present. This is appropriate when:
- `gert serve` is behind a trusted reverse proxy (nginx, Envoy, etc.)
- The proxy overwrites/sanitizes X-Forwarded-For

If deployed directly on the internet without a proxy, an attacker can spoof the header to bypass rate limiting. This is documented but not fixed in Phase 19 — the typical deployment model uses a proxy.

Future enhancement (out of scope): `--trust-proxy-headers` flag to explicitly enable/disable XFF trust.
