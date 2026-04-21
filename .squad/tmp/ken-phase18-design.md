# Ken — Phase 18 Design: JWT Signature Verification & CRUD Completion

**Date:** 2026-04-21  
**Author:** Ken (Staff Architect)  
**Implementor:** Brian  
**Phase:** 18

---

## Overview

Phase 18 is anchored by a **critical security fix**: JWT signature verification. The current implementation (Phase 17) validates `exp` and `iat` claims but **does not verify the signature** — any attacker can forge a JWT with arbitrary claims and bypass authentication. This is a complete authentication bypass.

Part A closes the security vulnerability with HMAC-SHA256 signature verification. Part B adds the `run.delete` RPC to complete CRUD operations on the serve layer. Part C applies the proven `WaitForSubscriber` pattern to the WebSocket timing flake.

**Estimated Effort:** 2-2.5 Brian-days

---

## Security Analysis: The Forgery Attack

### Current Vulnerability (NBI-17-01)

The current JWT validation in `middleware.go` accepts any token that:
1. Has three dot-separated segments (looks like a JWT)
2. Has an `exp` claim in the future
3. Has an `iat` claim within `maxAge`

**Attack vector:**

```bash
# Attacker creates a forged JWT with no signature
HEADER=$(echo -n '{"alg":"none","typ":"JWT"}' | base64 -w0 | tr '/+' '_-' | tr -d '=')
PAYLOAD=$(echo -n '{"exp":'$(($(date +%s)+3600))',"iat":'$(date +%s)'}' | base64 -w0 | tr '/+' '_-' | tr -d '=')
FORGED_JWT="${HEADER}.${PAYLOAD}."

# This WORKS with current implementation
curl -H "Authorization: Bearer $FORGED_JWT" http://localhost:7778/rpc
```

**Why this matters:** Any client can create a valid-looking JWT and authenticate to `gert serve`. The token comparison on line 154 only works if the server knows the exact JWT string — but that defeats the purpose of JWT entirely.

### The Root Cause

The architecture assumes the server "knows" the JWT (via `--auth-token`), then compares it byte-for-byte. This is fundamentally broken for JWTs. JWTs are designed to be:
1. **Created by the server** (or trusted issuer) with a signature
2. **Presented by the client** with the same signature
3. **Verified by the server** using the secret key

The current design conflates "the JWT is the secret" with "the JWT is signed by a secret."

---

## Part A — JWT Signature Verification (ANCHOR)

### Design: HMAC-SHA256 Verification

**New flag:** `--auth-jwt-secret <base64-encoded-secret>`

When `--auth-jwt-secret` is set:
1. Require the JWT to have `alg: HS256` (HMAC-SHA256)
2. Compute the expected signature: `HMAC-SHA256(header.payload, secret)`
3. Constant-time compare the provided signature against computed
4. Then validate `exp` and `iat` as before

**Flag semantics:**

| `--auth-token` | `--auth-jwt-secret` | Behavior |
|----------------|---------------------|----------|
| unset | unset | Open access (no auth) |
| set | unset | Plain bearer token (constant-time compare) |
| unset | set | JWT auth with signature verification |
| set | set | **Error at startup** — mutually exclusive |

**Fail-secure invariant:** If `--auth-jwt-secret` is NOT set but the provided token looks like a JWT, **REJECT it** with 401. Rationale: silently accepting unverified JWTs is the vulnerability we're fixing.

### Implementation

**File:** `internal/serve/middleware.go`

```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "crypto/subtle"
    "encoding/base64"
    // ...
)

// newBearerAuthMiddleware returns authentication middleware.
// Three modes:
//   1. token="" && jwtSecret=nil → no auth (open access)
//   2. token!="" && jwtSecret=nil → plain bearer token
//   3. token="" && jwtSecret!=nil → JWT auth with HMAC-SHA256 verification
func newBearerAuthMiddleware(token string, jwtSecret []byte, expiry time.Duration) middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // /health always exempt
            if r.URL.Path == "/health" {
                next.ServeHTTP(w, r)
                return
            }
            
            // No auth configured → open access
            if token == "" && jwtSecret == nil {
                next.ServeHTTP(w, r)
                return
            }
            
            auth := r.Header.Get("Authorization")
            if !strings.HasPrefix(auth, "Bearer ") {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            provided := strings.TrimPrefix(auth, "Bearer ")
            
            // Mode: JWT auth
            if jwtSecret != nil {
                if err := verifyJWT(provided, jwtSecret, expiry); err != nil {
                    http.Error(w, "Unauthorized", http.StatusUnauthorized)
                    return
                }
                next.ServeHTTP(w, r)
                return
            }
            
            // Mode: Plain bearer token
            // FAIL-SECURE: If token looks like JWT but no jwtSecret configured, reject it.
            // This prevents accidentally accepting unverified JWTs.
            if looksLikeJWT(provided) {
                http.Error(w, "Unauthorized: JWT tokens require --auth-jwt-secret", http.StatusUnauthorized)
                return
            }
            
            if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

// verifyJWT validates the JWT signature, exp, and iat claims.
func verifyJWT(token string, secret []byte, maxAge time.Duration) error {
    parts := strings.SplitN(token, ".", 3)
    if len(parts) != 3 {
        return errors.New("invalid JWT format")
    }
    header, payload, signature := parts[0], parts[1], parts[2]
    
    // Decode and validate header
    headerJSON, err := base64.RawURLEncoding.DecodeString(header)
    if err != nil {
        return fmt.Errorf("invalid header encoding: %w", err)
    }
    var hdr struct {
        Alg string `json:"alg"`
        Typ string `json:"typ"`
    }
    if err := json.Unmarshal(headerJSON, &hdr); err != nil {
        return fmt.Errorf("invalid header: %w", err)
    }
    if hdr.Alg != "HS256" {
        return fmt.Errorf("unsupported algorithm: %s (only HS256 supported)", hdr.Alg)
    }
    
    // Compute expected signature
    signingInput := header + "." + payload
    mac := hmac.New(sha256.New, secret)
    mac.Write([]byte(signingInput))
    expectedSig := mac.Sum(nil)
    
    // Decode provided signature
    providedSig, err := base64.RawURLEncoding.DecodeString(signature)
    if err != nil {
        return fmt.Errorf("invalid signature encoding: %w", err)
    }
    
    // Constant-time compare signatures
    if !hmac.Equal(expectedSig, providedSig) {
        return errors.New("invalid signature")
    }
    
    // Validate time claims (reuse existing logic)
    return validateJWTExpiry(token, maxAge)
}
```

**File:** `cmd/gert/serve.go`

```go
func runServe(args []string) int {
    // ...
    authToken := fs.String("auth-token", "", "Bearer token for auth (empty = no auth; mutually exclusive with --auth-jwt-secret)")
    authJWTSecret := fs.String("auth-jwt-secret", "", "Base64-encoded HMAC-SHA256 secret for JWT verification (mutually exclusive with --auth-token)")
    authTokenExpiry := fs.Duration("auth-token-expiry", 0, "Max JWT token age from iat (e.g. 24h); 0 = no expiry check")
    // ...
    
    if err := fs.Parse(args); err != nil {
        // ...
    }
    
    // Validate mutual exclusivity
    if *authToken != "" && *authJWTSecret != "" {
        fmt.Fprintf(os.Stderr, "serve: --auth-token and --auth-jwt-secret are mutually exclusive\n")
        return exitValidation
    }
    
    // Decode JWT secret if provided
    var jwtSecretBytes []byte
    if *authJWTSecret != "" {
        var err error
        jwtSecretBytes, err = base64.StdEncoding.DecodeString(*authJWTSecret)
        if err != nil {
            fmt.Fprintf(os.Stderr, "serve: --auth-jwt-secret must be valid base64: %v\n", err)
            return exitValidation
        }
        if len(jwtSecretBytes) < 32 {
            fmt.Fprintf(os.Stderr, "serve: --auth-jwt-secret must be at least 32 bytes (256 bits) for security\n")
            return exitValidation
        }
    }
    
    // ...
    cfg := servepkg.ServerConfig{
        // ...
        BearerToken:       *authToken,
        JWTSecret:         jwtSecretBytes,  // NEW FIELD
        BearerTokenExpiry: *authTokenExpiry,
    }
}
```

**File:** `pkg/serve/serve.go`

```go
type ServerConfig struct {
    // ...
    BearerToken       string        // Plain bearer token (mutually exclusive with JWTSecret)
    JWTSecret         []byte        // HMAC-SHA256 secret for JWT verification (mutually exclusive with BearerToken)
    BearerTokenExpiry time.Duration // Max age for JWT tokens (applies to JWTSecret mode only)
}
```

### Test Cases

```go
// makeSignedJWT creates a valid HMAC-SHA256 signed JWT for testing.
func makeSignedJWT(secret []byte, exp, iat int64) string {
    header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
    payload := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"exp":%d,"iat":%d}`, exp, iat)))
    signingInput := header + "." + payload
    
    mac := hmac.New(sha256.New, secret)
    mac.Write([]byte(signingInput))
    sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
    
    return header + "." + payload + "." + sig
}

func TestBearerAuth_JWTSignatureValid(t *testing.T) {
    secret := []byte("this-is-a-32-byte-secret-key!!!")
    now := time.Now()
    jwt := makeSignedJWT(secret, now.Add(1*time.Hour).Unix(), now.Unix())
    
    mw := newBearerAuthMiddleware("", secret, 24*time.Hour)
    handler := mw(http.HandlerFunc(okHandler))
    
    req := httptest.NewRequest(http.MethodPost, "/rpc", nil)
    req.Header.Set("Authorization", "Bearer "+jwt)
    rec := httptest.NewRecorder()
    handler.ServeHTTP(rec, req)
    
    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200 for valid signed JWT, got %d", rec.Code)
    }
}

func TestBearerAuth_JWTSignatureInvalid(t *testing.T) {
    secret := []byte("this-is-a-32-byte-secret-key!!!")
    wrongSecret := []byte("wrong-32-byte-secret-key!!!!!!!")
    now := time.Now()
    jwt := makeSignedJWT(wrongSecret, now.Add(1*time.Hour).Unix(), now.Unix())
    
    mw := newBearerAuthMiddleware("", secret, 24*time.Hour)
    handler := mw(http.HandlerFunc(okHandler))
    
    req := httptest.NewRequest(http.MethodPost, "/rpc", nil)
    req.Header.Set("Authorization", "Bearer "+jwt)
    rec := httptest.NewRecorder()
    handler.ServeHTTP(rec, req)
    
    if rec.Code != http.StatusUnauthorized {
        t.Fatalf("expected 401 for invalid signature, got %d", rec.Code)
    }
}

func TestBearerAuth_JWTForgedNoSignature(t *testing.T) {
    // Attacker forges a JWT with alg:none
    secret := []byte("this-is-a-32-byte-secret-key!!!")
    forged := makeTestJWT(time.Now().Add(1*time.Hour).Unix(), time.Now().Unix()) // unsigned
    
    mw := newBearerAuthMiddleware("", secret, 24*time.Hour)
    handler := mw(http.HandlerFunc(okHandler))
    
    req := httptest.NewRequest(http.MethodPost, "/rpc", nil)
    req.Header.Set("Authorization", "Bearer "+forged)
    rec := httptest.NewRecorder()
    handler.ServeHTTP(rec, req)
    
    if rec.Code != http.StatusUnauthorized {
        t.Fatalf("expected 401 for forged JWT, got %d", rec.Code)
    }
}

func TestBearerAuth_PlainTokenRejectsJWTLookalike(t *testing.T) {
    // When using plain token mode, JWT-like tokens should be rejected
    // This is the fail-secure behavior
    mw := newBearerAuthMiddleware("plaintoken", nil, 0)
    handler := mw(http.HandlerFunc(okHandler))
    
    jwtLike := "eyJhbGciOiJub25lIn0.eyJleHAiOjk5OTk5OTk5OTl9."
    req := httptest.NewRequest(http.MethodPost, "/rpc", nil)
    req.Header.Set("Authorization", "Bearer "+jwtLike)
    rec := httptest.NewRecorder()
    handler.ServeHTTP(rec, req)
    
    if rec.Code != http.StatusUnauthorized {
        t.Fatalf("expected 401 when JWT presented to plain token mode, got %d", rec.Code)
    }
}

func TestBearerAuth_JWTWrongAlgorithm(t *testing.T) {
    secret := []byte("this-is-a-32-byte-secret-key!!!")
    // Create token claiming RS256 (not supported)
    header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
    payload := base64.RawURLEncoding.EncodeToString([]byte(`{"exp":9999999999,"iat":1}`))
    
    mw := newBearerAuthMiddleware("", secret, 24*time.Hour)
    handler := mw(http.HandlerFunc(okHandler))
    
    req := httptest.NewRequest(http.MethodPost, "/rpc", nil)
    req.Header.Set("Authorization", "Bearer "+header+"."+payload+".fakesig")
    rec := httptest.NewRecorder()
    handler.ServeHTTP(rec, req)
    
    if rec.Code != http.StatusUnauthorized {
        t.Fatalf("expected 401 for unsupported algorithm, got %d", rec.Code)
    }
}
```

### Breaking Change Documentation

**BREAKING CHANGE:** JWT tokens now require signature verification.

**Before (Phase 17):**
```bash
# Server accepts any JWT with valid exp claim
gert serve --auth-token "any.jwt.here" --auth-token-expiry 24h
```

**After (Phase 18):**
```bash
# Option 1: Plain bearer token (non-JWT strings only)
gert serve --auth-token "mysecrettoken"

# Option 2: JWT with signature verification
JWT_SECRET=$(openssl rand -base64 32)
gert serve --auth-jwt-secret "$JWT_SECRET" --auth-token-expiry 24h

# Clients must present properly signed JWTs:
# jwt.io or similar to create: alg=HS256, secret=$JWT_SECRET
```

**Migration path:** Users currently using `--auth-token` with JWT-formatted tokens must switch to `--auth-jwt-secret` and generate properly signed tokens.

---

## Part B — run.delete RPC (NBI-17-03)

### Design

Add JSON-RPC method `run.delete` to complete CRUD:

| Method | Implemented | Phase |
|--------|-------------|-------|
| `run.list` | ✅ | 13 |
| `run.get` | ✅ | 13 |
| `run.delete` | ❌ → ✅ | **18** |

**RPC schema:**

```json
{
    "jsonrpc": "2.0",
    "method": "run.delete",
    "params": {
        "runID": "abc123"
    },
    "id": 1
}
```

**Response:**

```json
{
    "jsonrpc": "2.0",
    "result": {
        "deleted": true
    },
    "id": 1
}
```

**Error cases:**

| Error | Code | Message |
|-------|------|---------|
| Run is currently running | -32001 | "cannot delete running run" |
| Run not found | -32002 | "run not found" |

### Safety Invariant

**Running runs MUST NOT be deleted.** This matches `gert gc` behavior.

```go
func (s *Server) handleRunDelete(ctx context.Context, params json.RawMessage) (any, *jsonRPCError) {
    var p struct {
        RunID string `json:"runID"`
    }
    if err := json.Unmarshal(params, &p); err != nil {
        return nil, &jsonRPCError{Code: -32602, Message: "invalid params"}
    }
    if p.RunID == "" {
        return nil, &jsonRPCError{Code: -32602, Message: "runID required"}
    }
    
    // Load run state to check status
    state, err := s.runStore.LoadState(ctx, p.RunID)
    if err != nil {
        if errors.Is(err, runstore.ErrRunNotFound) {
            return nil, &jsonRPCError{Code: -32002, Message: "run not found"}
        }
        return nil, &jsonRPCError{Code: -32603, Message: err.Error()}
    }
    
    // Safety invariant: never delete running runs
    if state.Status == runtypes.StatusRunning {
        return nil, &jsonRPCError{Code: -32001, Message: "cannot delete running run"}
    }
    
    if err := s.runStore.DeleteRun(ctx, p.RunID); err != nil {
        return nil, &jsonRPCError{Code: -32603, Message: err.Error()}
    }
    
    return map[string]bool{"deleted": true}, nil
}
```

### Test Cases

```go
func TestRPC_RunDelete(t *testing.T) {
    h := newTestServerHarness(t)
    ctx := context.Background()
    
    // Create a completed run
    runID := "test-delete-run"
    state := &runtypes.RunState{
        RunID:  runID,
        Status: runtypes.StatusCompleted,
    }
    if err := h.runStore.SaveState(ctx, runID, state); err != nil {
        t.Fatalf("SaveState: %v", err)
    }
    
    // Delete it
    resp := h.call(t, "run.delete", map[string]string{"runID": runID})
    if resp.Error != nil {
        t.Fatalf("unexpected error: %+v", resp.Error)
    }
    
    // Verify deleted
    _, err := h.runStore.LoadState(ctx, runID)
    if err == nil {
        t.Fatal("expected run to be deleted")
    }
}

func TestRPC_RunDelete_Running(t *testing.T) {
    h := newTestServerHarness(t)
    ctx := context.Background()
    
    runID := "running-run"
    state := &runtypes.RunState{
        RunID:  runID,
        Status: runtypes.StatusRunning,
    }
    if err := h.runStore.SaveState(ctx, runID, state); err != nil {
        t.Fatalf("SaveState: %v", err)
    }
    
    resp := h.call(t, "run.delete", map[string]string{"runID": runID})
    if resp.Error == nil {
        t.Fatal("expected error when deleting running run")
    }
    if resp.Error.Code != -32001 {
        t.Fatalf("expected error code -32001, got %d", resp.Error.Code)
    }
}

func TestRPC_RunDelete_NotFound(t *testing.T) {
    h := newTestServerHarness(t)
    
    resp := h.call(t, "run.delete", map[string]string{"runID": "no-such-run"})
    if resp.Error == nil {
        t.Fatal("expected error for non-existent run")
    }
    if resp.Error.Code != -32002 {
        t.Fatalf("expected error code -32002, got %d", resp.Error.Code)
    }
}
```

---

## Part C — WebSocket Timing Flake Fix (NBI-17-05)

### Analysis

`TestWS_RunCompleted_ReceivesTerminal` has the same timing race that plagued `TestSSE_ConnectReceivesEvents` before Phase 17:

```go
// Current code (line 66-86 of ws_test.go)
conn := dialWS(t, ts.URL+"/ws?runID=run-1")
defer conn.Close(websocket.StatusNormalClosure, "bye")

// RACE: Broadcast may happen before subscription is registered
h.server.bridge.Broadcast(servepkg.RunEvent{...})

ev := readWSEvent(t, conn)  // May timeout if broadcast was missed
```

### Fix

Apply the same `WaitForSubscriber` pattern:

```go
func TestWS_RunCompleted_ReceivesTerminal(t *testing.T) {
    h := newTestServerHarness(t)
    ts := newHTTPTestServer(t, h.server)
    defer ts.Close()

    conn := dialWS(t, ts.URL+"/ws?runID=run-1")
    defer conn.Close(websocket.StatusNormalClosure, "bye")

    // Wait for subscription to be registered before broadcasting
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    if err := h.server.bridge.WaitForSubscriber(ctx, "run-1"); err != nil {
        t.Fatalf("WaitForSubscriber: %v", err)
    }

    h.server.bridge.Broadcast(servepkg.RunEvent{
        Type:     "run/completed",
        RunID:    "run-1",
        Sequence: 9,
        TS:       time.Now().Format(time.RFC3339Nano),
        Payload:  map[string]any{},
    })

    ev := readWSEvent(t, conn)
    if ev.Type != "run/completed" {
        t.Fatalf("expected run/completed, got %s", ev.Type)
    }
}
```

**Note:** The `WaitForSubscriber` method already exists from Phase 17 (SSE fix). We're just applying it to the WS test.

---

## Deferred Items

### NBI-17-02: Token Rotation / Revocation

**Decision: DEFER to Phase 19**

An in-memory token blocklist adds complexity (map management, memory growth) with limited value for the current use case. The server already requires restart to change the JWT secret. Token rotation is better addressed with:
1. Short token expiry (`--auth-token-expiry 1h`)
2. External token management (OAuth2/OIDC) in Phase 19+

### NBI-16-03 / NBI-17-02: E2E Test Parallelization

**Decision: DEFER to Phase 19**

Adding `t.Parallel()` to E2E tests requires careful analysis of shared state (temp directories, port allocation). Low priority relative to security fix. Scope for Phase 19 after auth hardening is complete.

### NBI-17-04: Rate Limiting

**Decision: DEFER to Phase 19**

Rate limiting is important for production but lower priority than fixing the authentication bypass.

---

## Disposition Summary

| ID | Disposition | Rationale |
|----|-------------|-----------|
| NBI-17-01 | **IN SCOPE (ANCHOR)** | Critical security: JWT signature verification |
| NBI-17-03 | **IN SCOPE (Part B)** | CRUD completion, small surface |
| NBI-17-05 | **IN SCOPE (Part C)** | Trivial: apply existing WaitForSubscriber |
| NBI-17-02 | **DEFER** | Token rotation adds complexity, short expiry is sufficient |
| NBI-16-03 | **DEFER** | E2E parallelization low priority |
| NBI-17-04 | **DEFER** | Rate limiting lower priority than auth fix |

---

## Key Decisions

| ID | Decision |
|----|----------|
| D-18-01 | JWT signature verification uses HMAC-SHA256 only (no RSA, no ES256) |
| D-18-02 | `--auth-token` and `--auth-jwt-secret` are mutually exclusive |
| D-18-03 | Fail-secure: JWT-like tokens rejected if no jwtSecret configured |
| D-18-04 | Minimum secret length: 32 bytes (256 bits) |
| D-18-05 | `run.delete` refuses running runs (safety invariant) |

---

## Deliverables

**Modified files:**
- `internal/serve/middleware.go` — JWT signature verification
- `internal/serve/middleware_test.go` — New test cases
- `internal/serve/rpc.go` — `run.delete` handler
- `internal/serve/rpc_test.go` — Delete RPC tests
- `internal/serve/ws_test.go` — WaitForSubscriber fix
- `pkg/serve/serve.go` — `JWTSecret` config field
- `cmd/gert/serve.go` — `--auth-jwt-secret` flag

**New files:** 0

**New tests:** ~8

**Estimated effort:** 2-2.5 Brian-days

---

## NBI Items for Phase 19+

| ID | Priority | Description |
|----|----------|-------------|
| NBI-18-01 | Medium | OAuth2/OIDC token validation (external identity providers) |
| NBI-18-02 | Medium | Rate limiting for `gert serve` (carry-forward from NBI-17-04) |
| NBI-18-03 | Low | E2E test parallelization (carry-forward from NBI-16-03) |
| NBI-18-04 | Low | Token revocation/blocklist (if needed beyond expiry) |

---

*Ken, Staff Architect — 2026-04-21*
