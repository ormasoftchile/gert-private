# Brian — Phase 18 Implementation Summary

**Date:** 2026-04-21
**Phase:** 18
**Implementor:** Brian

---

## Files Modified

### Part A — JWT Signature Verification

**`v2/pkg/serve/serve.go`**
- Added `JWTSecret []byte` field to `ServerConfig` (next to `BearerToken`)
- Updated `BearerTokenExpiry` doc comment to reflect JWT mode usage

**`v2/internal/serve/middleware.go`**
- Added `crypto/hmac`, `crypto/sha256` imports
- Rewrote `newBearerAuthMiddleware` — new signature: `(token string, expiry time.Duration, jwtSecret []byte) middleware`
- Three-mode logic: open access / plain bearer (fail-secure) / JWT signature verification
- Added `verifyJWT(token string, secret []byte, maxAge time.Duration) error` — HMAC-SHA256, constant-time comparison, HS256 only, delegates time checks to existing `validateJWTExpiry`
- Updated `looksLikeJWT` to require **non-empty** segments (design spec; also prevents alg:none tokens with empty sig from triggering fail-secure)
- Updated `withMiddleware` to pass `s.cfg.JWTSecret` to `newBearerAuthMiddleware`

**`v2/cmd/gert/serve.go`**
- Added `encoding/base64` import
- Added `--auth-jwt-secret` flag (string, Base64-encoded HMAC-SHA256 secret)
- Mutual exclusivity check: `--auth-token` + `--auth-jwt-secret` → `exitValidation`
- Base64 decode + 32-byte minimum enforcement
- Sets `cfg.JWTSecret = jwtSecretBytes`

**`v2/internal/serve/middleware_test.go`**
- Added `crypto/hmac`, `crypto/sha256` imports
- Renamed `makeTestJWT` → `makeUnsignedJWT` (alg:none, empty sig — used only for algorithm rejection tests)
- Added `makeSignedJWT(t, secret, exp, iat)` helper
- Updated ALL existing `newBearerAuthMiddleware` calls to add `nil` third parameter
- Rewrote `TestBearerAuth_JWTValid/Expired/TooOld/InvalidFormat` to use JWT signature mode with `makeSignedJWT`
- Added 5 new tests: `TestBearerAuth_JWTSignatureValid`, `TestBearerAuth_JWTSignatureInvalid`, `TestBearerAuth_JWTForgedNoSignature`, `TestBearerAuth_PlainTokenRejectsJWTLookalike`, `TestBearerAuth_JWTWrongAlgorithm`

### Part B — run.delete RPC

**`v2/internal/serve/rpc.go`**
- Added `rpcRunDeleteRunning = -32020` constant
- Added `case "run.delete":` to dispatch switch
- Added `handleRunDelete` handler: checks registry for running state, loads store state, checks for running status in store, deletes via type assertion on `DeleteRun`

**`v2/internal/serve/rpc_test.go`**
- Added 4 tests: `TestRPC_RunDelete`, `TestRPC_RunDelete_Running`, `TestRPC_RunDelete_NotFound`, `TestRPC_RunDelete_MissingRunID`

### Part C — WebSocket Timing Flake Fix

**`v2/internal/serve/ws_test.go`**
- Added `WaitForSubscriber` call in `TestWS_RunCompleted_ReceivesTerminal` between `dialWS` and `Broadcast`, matching the SSE pattern from Phase 17

---

## Deviations from Design

1. **Error codes for run.delete**: Design specified `-32001` (running) and `-32002` (not found), but these conflict with existing `rpcRunbookParseErr = -32001` and `rpcRunbookInvalid = -32002`. Used `rpcRunDeleteRunning = -32020` for running guard, and reused existing `rpcRunNotFound = -32010` for not found. Tests reflect these values.

2. **`validateJWTExpiry` not renamed**: Design suggested renaming to `validateJWTTimeClaims`. Kept existing function name and called it from `verifyJWT` as `validateJWTExpiry(token, maxAge)`. Functionally equivalent.

3. **`looksLikeJWT` change affects alg:none tokens**: Updating `looksLikeJWT` to require non-empty third segment means `makeTestJWT` tokens (alg:none, empty sig "...") no longer "look like JWTs." This is intentional — empty-signature tokens are not legitimate JWTs. The old JWT expiry tests were updated to use the new JWT signature mode.

---

## Test Count

- `internal/serve` package: 60+ tests, all passing
- New tests added: 5 (Part A middleware) + 4 (Part B rpc) = 9
- `go test ./... -race -count=1`: all packages PASS
