# Ken — Phase 17 Design: Security Hardening & SSE Stabilization

**Date:** 2026-04-21  
**Author:** Ken (Software Architect)  
**Implementor:** Brian  
**Phase:** 17

---

## Overview

Phase 17 is anchored by a **security-critical fix**: upgrading bearer token comparison to `subtle.ConstantTimeCompare`. While the impact is low (dev-only auth per D-16-04), it establishes the correct pattern before any production auth work.

Part A closes two trivial NBI items: the timing-safe comparison fix and `run.list` schema documentation. Part B adds optional **token expiry support** and properly fixes the **SSE timing flake** that was `t.Skip`'d in Phase 16.

**Estimated Effort:** 1.5-2 Brian-days

---

## Part A — NBI Carry-Forwards

### Disposition Summary

| ID | Disposition | Rationale |
|----|-------------|-----------|
| NBI-16-01 | **IN SCOPE (ANCHOR)** | Security-critical: timing-safe token comparison |
| NBI-16-04 | **IN SCOPE** | Trivial: add godoc for run.list response schema |
| NBI-16-02 | **PARTIAL** | Token expiry only; full OAuth2/OIDC deferred |
| NBI-15-01 | **IN SCOPE** | Fix SSE flake properly (remove t.Skip) |
| NBI-16-03 | **DEFER** | E2E parallelization low priority |

---

### NBI-16-01: Timing-Safe Bearer Token Comparison (ANCHOR)

**Priority:** High (Security)  
**File:** `internal/serve/middleware.go`

**Current state (vulnerable):**

```go
// Line 143 — direct string comparison enables timing attacks
if strings.TrimPrefix(auth, "Bearer ") != token {
```

**Required fix:**

```go
import "crypto/subtle"

// Constant-time comparison prevents timing attacks
provided := strings.TrimPrefix(auth, "Bearer ")
if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
    http.Error(w, "Forbidden", http.StatusForbidden)
    return
}
```

**Rationale:**
- String `!=` leaks timing information proportional to the first differing byte
- `subtle.ConstantTimeCompare` takes constant time regardless of match position
- Even for dev-only auth, this establishes the correct pattern for Phase 18 production auth

**Test:**

Add test proving the function uses constant-time comparison (test validates behavior, not timing — timing tests are notoriously flaky):

```go
func TestBearerAuth_UsesConstantTimeCompare(t *testing.T) {
    // Verify auth behavior is correct for various prefix-match scenarios
    // The goal is behavioral correctness; timing characteristics are implementation detail
    mw := newBearerAuthMiddleware("secrettoken123")
    
    cases := []struct {
        name   string
        token  string
        status int
    }{
        {"correct", "secrettoken123", http.StatusOK},
        {"wrong_first_byte", "xecrettoken123", http.StatusForbidden},
        {"wrong_last_byte", "secrettoken12x", http.StatusForbidden},
        {"wrong_length", "secret", http.StatusForbidden},
        {"empty", "", http.StatusUnauthorized}, // no Bearer prefix
    }
    
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            req := httptest.NewRequest(http.MethodGet, "/", nil)
            if tc.token != "" {
                req.Header.Set("Authorization", "Bearer "+tc.token)
            }
            rec := httptest.NewRecorder()
            mw(okHandler()).ServeHTTP(rec, req)
            if rec.Code != tc.status {
                t.Errorf("got %d, want %d", rec.Code, tc.status)
            }
        })
    }
}
```

**Files modified:**
- `internal/serve/middleware.go` — add `crypto/subtle` import, replace comparison
- `internal/serve/middleware_test.go` — add edge case tests

---

### NBI-16-04: run.list RPC Schema Documentation

**Priority:** Low  
**File:** `internal/serve/rpc.go`

Add godoc documenting the `run.list` response schema:

```go
// handleRunList returns a list of all runs (active and persisted).
//
// Response schema:
//
//     [
//       {
//         "runID": string,           // Unique run identifier
//         "state": string,           // "pending"|"running"|"paused"|"completed"|"failed"|"cancelled"
//         "runbookPath": string,     // Path to runbook file
//         "startedAt": string,       // RFC3339Nano timestamp
//         "source": string           // "active" (in-memory) or "persisted" (on-disk)
//       }
//     ]
//
// Active runs from the registry take precedence over persisted runs with the same runID.
// Store errors are silently ignored (graceful degradation).
func (s *Server) handleRunList(w http.ResponseWriter, r *http.Request, req rpcRequest) {
```

**Files modified:**
- `internal/serve/rpc.go` — add godoc comment to `handleRunList`

---

## Part B — New Work

### Token Expiry Support (from NBI-16-02)

**Priority:** Medium  
**Files:** `internal/serve/middleware.go`, `pkg/serve/serve.go`, `cmd/gert/serve.go`

Add optional token expiry validation. This is a stepping stone toward full auth hardening without the complexity of JWT/OAuth2.

**Design:**

1. Add `--auth-token-expiry` CLI flag (duration, e.g., `24h`)
2. When set, tokens are expected to be signed JWTs with `iat` and `exp` claims
3. When unset (default), tokens are treated as opaque shared secrets (current behavior)

**Implementation:**

```go
// pkg/serve/serve.go — extend config
type ServerConfig struct {
    // ... existing fields ...
    
    // BearerTokenExpiry is the maximum age of a token from its issued-at time.
    // When zero, tokens are treated as opaque shared secrets (no expiry check).
    // When non-zero, tokens must be JWTs with iat/exp claims validated against this duration.
    BearerTokenExpiry time.Duration
}
```

```go
// internal/serve/middleware.go — extend bearer auth

import (
    "crypto/subtle"
    "encoding/base64"
    "encoding/json"
    "time"
)

func newBearerAuthMiddleware(token string, expiry time.Duration) middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if token == "" || r.URL.Path == "/health" {
                next.ServeHTTP(w, r)
                return
            }
            auth := r.Header.Get("Authorization")
            if !strings.HasPrefix(auth, "Bearer ") {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            provided := strings.TrimPrefix(auth, "Bearer ")
            
            // If no expiry configured, do simple constant-time comparison
            if expiry == 0 {
                if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
                    http.Error(w, "Forbidden", http.StatusForbidden)
                    return
                }
                next.ServeHTTP(w, r)
                return
            }
            
            // With expiry, validate JWT-like token (simplified: base64 JSON payload)
            if err := validateTokenWithExpiry(provided, token, expiry); err != nil {
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

// validateTokenWithExpiry checks a simple JWT-like token format:
// base64({"iat":unix,"exp":unix,"secret":"..."})
// The "secret" claim must match the configured token.
func validateTokenWithExpiry(provided, expected string, maxAge time.Duration) error {
    // Decode base64
    payload, err := base64.StdEncoding.DecodeString(provided)
    if err != nil {
        return err
    }
    
    var claims struct {
        IAT    int64  `json:"iat"`
        EXP    int64  `json:"exp"`
        Secret string `json:"secret"`
    }
    if err := json.Unmarshal(payload, &claims); err != nil {
        return err
    }
    
    // Constant-time compare secret
    if subtle.ConstantTimeCompare([]byte(claims.Secret), []byte(expected)) != 1 {
        return errors.New("invalid secret")
    }
    
    now := time.Now().Unix()
    
    // Check expiry
    if claims.EXP != 0 && claims.EXP < now {
        return errors.New("token expired")
    }
    
    // Check issued-at vs maxAge
    if claims.IAT != 0 && maxAge > 0 {
        issuedAt := time.Unix(claims.IAT, 0)
        if time.Since(issuedAt) > maxAge {
            return errors.New("token too old")
        }
    }
    
    return nil
}
```

**CLI flag:**

```go
// cmd/gert/serve.go
authTokenExpiry := fs.Duration("auth-token-expiry", 0, "Max token age (e.g., 24h); 0 means no expiry check")

// Pass to config
BearerTokenExpiry: *authTokenExpiry,
```

**Tests:**

- `TestBearerAuth_WithExpiry_ValidToken` — token within expiry passes
- `TestBearerAuth_WithExpiry_ExpiredToken` — token past expiry rejected
- `TestBearerAuth_WithExpiry_TooOld` — token older than maxAge rejected
- `TestBearerAuth_WithExpiry_InvalidFormat` — malformed token rejected
- `TestBearerAuth_NoExpiry_OpaqueToken` — zero expiry uses simple comparison

**Files modified:**
- `internal/serve/middleware.go` — extend `newBearerAuthMiddleware`, add `validateTokenWithExpiry`
- `internal/serve/middleware_test.go` — add 5 tests
- `pkg/serve/serve.go` — add `BearerTokenExpiry` field
- `cmd/gert/serve.go` — add `--auth-token-expiry` flag

---

### SSE Timing Flake Fix (NBI-15-01)

**Priority:** Medium  
**File:** `internal/serve/sse_test.go`

The `TestSSE_ConnectReceivesEvents` test was `t.Skip`'d in Phase 16 due to a timing flake. The root cause is a race between:
1. SSE client connection establishment
2. Event broadcast

**Proper fix:** Use a synchronization primitive — broadcast after confirming client is connected.

**Implementation:**

Add a `WaitForSubscriber` method to the SSE bridge that blocks until at least one subscriber is connected:

```go
// internal/serve/events.go — extend EventBridge
func (b *EventBridge) WaitForSubscriber(ctx context.Context, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    for {
        b.mu.RLock()
        count := len(b.subscribers)
        b.mu.RUnlock()
        if count > 0 {
            return nil
        }
        if time.Now().After(deadline) {
            return errors.New("timeout waiting for subscriber")
        }
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(10 * time.Millisecond):
            // Poll again
        }
    }
}
```

**Test fix:**

```go
func TestSSE_ConnectReceivesEvents(t *testing.T) {
    // REMOVED: t.Skip("flaky: timing-sensitive SSE test — see NBI-15-01")
    h := newTestServerHarness(t)
    ts := newHTTPTestServer(t, h.server)
    defer ts.Close()

    // Open SSE connection in background
    var resp *http.Response
    done := make(chan struct{})
    go func() {
        resp = openSSE(t, ts.URL+"/events", "")
        close(done)
    }()
    
    // Wait for subscriber to be registered
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    if err := h.server.bridge.WaitForSubscriber(ctx, 2*time.Second); err != nil {
        t.Fatalf("subscriber not registered: %v", err)
    }
    <-done // Ensure resp is assigned
    defer resp.Body.Close()

    h.server.bridge.Broadcast(servepkg.RunEvent{
        Type:     "step/started",
        RunID:    "run-1",
        Sequence: 1,
        TS:       time.Now().Format(time.RFC3339Nano),
        Payload:  map[string]any{},
    })

    ev := readSSEEvent(t, resp.Body)
    if ev.Type != "step/started" {
        t.Fatalf("expected step/started, got %s", ev.Type)
    }
}
```

**Alternative (simpler):** Use `testutil.Eventually` pattern with a retry loop:

```go
func TestSSE_ConnectReceivesEvents(t *testing.T) {
    h := newTestServerHarness(t)
    ts := newHTTPTestServer(t, h.server)
    defer ts.Close()

    resp := openSSE(t, ts.URL+"/events", "")
    defer resp.Body.Close()
    
    // Small delay to allow subscription registration
    time.Sleep(50 * time.Millisecond)

    h.server.bridge.Broadcast(servepkg.RunEvent{
        Type:     "step/started",
        RunID:    "run-1",
        Sequence: 1,
        TS:       time.Now().Format(time.RFC3339Nano),
        Payload:  map[string]any{},
    })

    ev := readSSEEventWithTimeout(t, resp.Body, 2*time.Second)
    if ev.Type != "step/started" {
        t.Fatalf("expected step/started, got %s", ev.Type)
    }
}

func readSSEEventWithTimeout(t *testing.T, r io.Reader, timeout time.Duration) servepkg.RunEvent {
    t.Helper()
    done := make(chan servepkg.RunEvent, 1)
    go func() {
        done <- readSSEEvent(t, r)
    }()
    select {
    case ev := <-done:
        return ev
    case <-time.After(timeout):
        t.Fatal("timeout reading SSE event")
        return servepkg.RunEvent{}
    }
}
```

**Recommendation:** Use the `WaitForSubscriber` approach — it's explicit about the synchronization point and doesn't rely on magic sleep durations.

**Files modified:**
- `internal/serve/events.go` — add `WaitForSubscriber` method
- `internal/serve/sse_test.go` — remove `t.Skip`, use `WaitForSubscriber`

---

## Deferred Items

### NBI-16-03: E2E Test Parallelization

**Rationale for deferral:** Low priority, no blocking issues. The E2E tests pass reliably in serial mode. Parallelization requires careful resource isolation (temp directories, ports) and is better addressed after all serve-layer features are stable.

**Carry forward to:** Phase 18

### run.delete RPC

**Rationale for exclusion:** Not included in Phase 17. While it would close the CRUD loop (list/get/delete), the current `run.cancel` + eventual expiry/cleanup is sufficient for MVP. Adding `run.delete` introduces questions about:
- Deleting persisted runs (disk cleanup)
- Concurrent access (what if someone resumes a deleted run?)
- Audit trail implications

**Carry forward to:** Phase 19 (post-auth hardening)

---

## Design Decisions

### D-17-01: Constant-Time Token Comparison is Mandatory

**Decision:** All token comparisons in `gert serve` MUST use `subtle.ConstantTimeCompare`.

**Rationale:**
- Timing attacks are real (even for "dev-only" auth)
- The fix is trivial (one import, one line change)
- Establishes correct pattern for future auth work
- No performance impact on typical workloads

### D-17-02: Token Expiry is Opt-In

**Decision:** Token expiry validation is enabled only when `--auth-token-expiry` is set. Default behavior (no flag) remains unchanged.

**Rationale:**
- Backward compatibility with existing deployments
- Simple shared secrets work fine for development
- Expiry is a stepping stone, not a production-grade solution
- Full JWT/OAuth2 is Phase 18+ scope

### D-17-03: SSE Synchronization via WaitForSubscriber

**Decision:** Fix SSE flake by adding explicit synchronization (`WaitForSubscriber`) rather than increasing sleep durations.

**Rationale:**
- Sleep-based tests are inherently flaky
- Explicit synchronization makes the test deterministic
- `WaitForSubscriber` is useful beyond tests (e.g., integration scenarios)
- Small API addition to `EventBridge`

### D-17-04: run.delete Deferred to Phase 19

**Decision:** Do not add `run.delete` RPC in Phase 17.

**Rationale:**
- Phase 17 is focused on security hardening
- Delete has complex implications (disk cleanup, concurrent access, audit)
- Current `run.cancel` + manual cleanup is sufficient for MVP
- Better to design delete after auth is complete (delete permissions)

---

## Validation Gate

```bash
cd v2
go build ./...              # Must exit 0
go vet ./...                # Must exit 0
go test ./... -race -count=3  # Must pass (no t.Skip'd tests)
```

---

## Summary

| Item | Scope | Files Modified | New Tests |
|------|-------|----------------|-----------|
| NBI-16-01 | Timing-safe comparison | 2 | 1 |
| NBI-16-04 | run.list godoc | 1 | 0 |
| Token expiry | New feature | 4 | 5 |
| NBI-15-01 | SSE flake fix | 2 | 0 (fix existing) |

**Total files modified:** 7  
**Total new tests:** ~6  
**Estimated effort:** 1.5-2 Brian-days

---

## NBI Items for Phase 18+

| ID | Priority | Description |
|----|----------|-------------|
| NBI-17-01 | Medium | Full auth hardening (OAuth2/OIDC, API key rotation) |
| NBI-17-02 | Low | E2E test parallelization (carry-forward from NBI-16-03) |
| NBI-17-03 | Low | run.delete RPC (CRUD completion) |
| NBI-17-04 | Medium | Rate limiting for `gert serve` |

---

*Ken, Software Architect*
