# Ken — Phase 20 Design (RE-SCOPED): API Completeness, Multiple CORS Origins, Proxy Trust Flag

**Date:** 2026-04-21 (original) — **RE-SCOPED 2026-04-21**
**Author:** Ken (Staff Architect)  
**Implementor:** Brian  
**Phase:** 20

> **CORRECTION NOTICE:** The original Phase 20 design included `run.delete` (Part A) and a WebSocket timing
> flake fix (Part C), both of which were already shipped in Phase 18 (commit 24d863e). This file is the
> corrected, re-scoped design containing only genuinely new work.

---

## Overview

Phase 19 sealed rate limiting and E2E parallelization. After reading the actual codebase state, the
following items from the original design are confirmed **already done**:

- `run.delete` RPC — `handleRunDelete` at `v2/internal/serve/rpc.go:635`, dispatch at line 105, error
  constant `rpcRunDeleteRunning = -32020` at line 37, safety invariant enforced.
- WebSocket timing flake fix — `WaitForSubscriber` applied at `v2/internal/serve/ws_test.go:78`.

Phase 20 (re-scoped) contains **three genuinely new parts**:

1. **Part A (ANCHOR):** API completeness — `completedAt` surfacing + `run.get` godoc + testdata fixtures
2. **Part B:** Multiple CORS origins — make `--cors-origin` a repeatable flag
3. **Part C:** `--trust-proxy-headers` security flag — XFF trust must be explicit opt-in

**Phase budget:** 1.5 Brian-days


---

## NBI Queue Analysis (Corrected)

| NBI ID | Item | Status | Disposition |
|--------|------|--------|-------------|
| NBI-17-02 | Token rotation/revocation | **CLOSED** (Phase 19) | WONT_FIX — short expiry sufficient |
| NBI-17-04 | Rate limiting | **CLOSED** (Phase 19) | ✅ DONE |
| NBI-16-03 | E2E test parallelization | **CLOSED** (Phase 19) | ✅ DONE |
| NBI-17-03 | run.delete RPC | **CLOSED** (Phase 18, 24d863e) | ✅ DONE — handleRunDelete at rpc.go:635 |
| NBI-17-05 | WS timing flake fix | **CLOSED** (Phase 18, 24d863e) | ✅ DONE — WaitForSubscriber at ws_test.go:78 |
| NBI-16-05 | run.list/run.get API schema docs | OPEN (partial) | ✅ IN SCOPE Part A — run.get godoc missing; completedAt gap |
| NBI-16-07 | CompletedAt for persisted runs | OPEN | ✅ IN SCOPE Part A — serve-layer gap |
| NBI-16-06 | Multiple CORS origins | OPEN | ✅ IN SCOPE Part B — reclassified from DEFER |
| NBI-16-01 | SSE test sync fix | **CLOSED** (Phase 17) | ✅ DONE via WaitForSubscriber |
| NBI-18-01 | OAuth2/OIDC | OPEN | DEFER — external auth is v2.1 |

**Assessment:** The data-plane (run.delete, WS flake) is fully clean. Three meaningful gaps remain:
completedAt is saved to disk but not returned by the API for persisted runs; the CORS flag doesn't
support multiple origins; the rate limiter trusts X-Forwarded-For unconditionally (security risk).

---

## Part A — API Completeness: `completedAt` Surfacing + `run.get` Godoc + Fixtures (NBI-16-05, NBI-16-07)

### What's Missing (confirmed by reading the code)

1. **`handleRunGet` has no godoc.** `handleRunList` at `rpc.go:372–387` has a full schema comment.
   `handleRunGet` at `rpc.go:442` has none.

2. **`completedAt` is absent from persisted `run.get` responses.** For active registry entries,
   `handleRunGet` correctly emits `completedAt` when `entry.CompletedAt` is non-zero (line 479–481).
   For persisted runs loaded via `store.LoadState`, the result map at lines 489–500 **never** includes
   `completedAt` — even though `engine.RunState.CompletedAt` is populated by `SaveState` (via
   `json.Marshal(state)`) at the time the run completes.

3. **`completedAt` is absent from all `run.list` responses.** Neither the active-run block (lines
   393–410) nor the persisted-run block (lines 419–431) includes `completedAt`.

4. **No testdata fixtures exist.** `v2/testdata/` directory does not exist. There are no example
   request/response JSON files to document the wire contract.

### Deliverables

#### A1 — Add godoc to `handleRunGet` (`v2/internal/serve/rpc.go:442`)

Insert the following comment immediately before `func (s *Server) handleRunGet(...)`:

```go
// handleRunGet returns the full state of a single run by ID.
//
// Params:
//   {"runID": string}
//
// Response schema (active run):
//
//{
//  "runID":            string,   // Unique run identifier
//  "state":            string,   // "pending"|"running"|"paused"|"completed"|"failed"|"cancelled"
//  "runbookPath":      string,   // Path to runbook file
//  "startedAt":        string,   // RFC3339Nano timestamp
//  "currentStep":      string,   // ID of the step currently executing (empty if none)
//  "currentStepIndex": number,   // 0-based index into plan.steps (-1 before first step)
//  "vars":             object,   // Runtime variable map (string→string)
//  "completedAt":      string,   // RFC3339Nano; present only when state is terminal
//  "source":           string    // "active"
//}
//
// Response schema (persisted run):
//
//{
//  "runID":            string,
//  "state":            string,
//  "runbookPath":      string,
//  "startedAt":        string,
//  "currentStep":      string,
//  "currentStepIndex": number,
//  "vars":             object,
//  "completedAt":      string,   // RFC3339Nano; present when terminal and timestamp was recorded
//  "source":           string    // "persisted"
//}
//
// Error: -32602 (Invalid params) if runID is missing.
// Error: -32010 (Run not found) if no active or persisted run matches runID.
```

#### A2 — Fix `handleRunGet` persisted path to include `completedAt`

**File:** `v2/internal/serve/rpc.go`  
**Location:** The `if s.store != nil` block starting at line 486.

Current code (lines 489–500):
```go
result := map[string]any{
    "runID":            runState.RunID,
    "state":            string(runState.Status),
    "runbookPath":      runState.RunbookPath,
    "startedAt":        runState.StartedAt.Format(time.RFC3339Nano),
    "currentStep":      runState.CurrentStep,
    "currentStepIndex": runState.CurrentStepIndex,
    "vars":             runState.Vars,
    "source":           "persisted",
}
```

Replace with:
```go
result := map[string]any{
    "runID":            runState.RunID,
    "state":            string(runState.Status),
    "runbookPath":      runState.RunbookPath,
    "startedAt":        runState.StartedAt.Format(time.RFC3339Nano),
    "currentStep":      runState.CurrentStep,
    "currentStepIndex": runState.CurrentStepIndex,
    "vars":             runState.Vars,
    "source":           "persisted",
}
if !runState.CompletedAt.IsZero() {
    result["completedAt"] = runState.CompletedAt.Format(time.RFC3339Nano)
}
```

#### A3 — Fix `handleRunList` to include `completedAt`

**File:** `v2/internal/serve/rpc.go`

**Active-run block** (around line 404): Add `completedAt` when `entry.CompletedAt` is non-zero.

```go
item := map[string]any{
    "runID":       entry.ID,
    "state":       string(state),
    "runbookPath": entry.RunbookPath,
    "startedAt":   startedAt.Format(time.RFC3339Nano),
    "source":      "active",
}
if !entry.CompletedAt.IsZero() {
    item["completedAt"] = entry.CompletedAt.Format(time.RFC3339Nano)
}
response = append(response, item)
```

**Persisted-run block** (around line 423): Add `completedAt` from `RunState.CompletedAt`.

```go
item := map[string]any{
    "runID":       run.RunID,
    "state":       string(run.Status),
    "runbookPath": run.RunbookPath,
    "startedAt":   run.StartedAt.Format(time.RFC3339Nano),
    "source":      "persisted",
}
if !run.CompletedAt.IsZero() {
    item["completedAt"] = run.CompletedAt.Format(time.RFC3339Nano)
}
response = append(response, item)
```

#### A4 — Testdata Fixtures

Create `v2/internal/serve/testdata/` with example request/response pairs documenting the wire API.

**`v2/internal/serve/testdata/run.list.response.json`** — example with one active and one persisted run:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": [
    {
      "runID": "r-active-001",
      "state": "running",
      "runbookPath": "runbooks/deploy.yaml",
      "startedAt": "2026-04-21T10:00:00.000000000Z",
      "source": "active"
    },
    {
      "runID": "r-done-002",
      "state": "completed",
      "runbookPath": "runbooks/check.yaml",
      "startedAt": "2026-04-20T09:00:00.000000000Z",
      "completedAt": "2026-04-20T09:05:32.000000000Z",
      "source": "persisted"
    }
  ]
}
```

**`v2/internal/serve/testdata/run.get.active.response.json`**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "runID": "r-active-001",
    "state": "running",
    "runbookPath": "runbooks/deploy.yaml",
    "startedAt": "2026-04-21T10:00:00.000000000Z",
    "currentStep": "step-deploy",
    "currentStepIndex": 2,
    "vars": {"env": "prod", "region": "us-east-1"},
    "source": "active"
  }
}
```

**`v2/internal/serve/testdata/run.get.persisted.response.json`**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "runID": "r-done-002",
    "state": "completed",
    "runbookPath": "runbooks/check.yaml",
    "startedAt": "2026-04-20T09:00:00.000000000Z",
    "completedAt": "2026-04-20T09:05:32.000000000Z",
    "currentStep": "step-verify",
    "currentStepIndex": 3,
    "vars": {"env": "prod"},
    "source": "persisted"
  }
}
```

**`v2/internal/serve/testdata/run.get.error.not_found.json`**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {"code": -32010, "message": "Run not found"}
}
```

### Tests

Add to `v2/internal/serve/rpc_test.go`:

1. **TestRPC_RunGet_Persisted_CompletedAt** — save a completed RunState with non-zero CompletedAt,
   call `run.get`, assert `completedAt` is present in the response.
2. **TestRPC_RunList_CompletedAt_ActiveRun** — complete a run (set `entry.CompletedAt`), call
   `run.list`, assert `completedAt` is present in that item.
3. **TestRPC_RunList_CompletedAt_PersistedRun** — save a completed RunState with non-zero
   CompletedAt, call `run.list`, assert `completedAt` appears.

**Estimated:** 30 LOC implementation changes, 60 LOC tests, 4 fixture files.

---

## Part B — Multiple CORS Origins (NBI-16-06)

### Current State

`v2/cmd/gert/serve.go` line 27:
```go
corsOrigin := fs.String("cors-origin", "", "Allowed CORS origin (empty = allow all)")
```

This is a single-string flag. To allow multiple origins, the operator must pick one. The
`ServerConfig.AllowedOrigins` is already `[]string` — the gap is purely in the CLI flag parsing.

### Design

Replace the single string with a custom `repeatedFlag` type implementing `flag.Value`, allowing
`--cors-origin` to be specified multiple times:

```
gert serve --cors-origin https://app.example.com --cors-origin https://staging.example.com
```

**File:** `v2/cmd/gert/serve.go`

```go
// repeatedStringFlag implements flag.Value for a repeatable string flag.
type repeatedStringFlag []string

func (f *repeatedStringFlag) String() string { return strings.Join(*f, ",") }
func (f *repeatedStringFlag) Set(v string) error {
    *f = append(*f, v)
    return nil
}

// In runServe:
var corsOrigins repeatedStringFlag
fs.Var(&corsOrigins, "cors-origin", "Allowed CORS origin; may be repeated (empty = allow all)")

// Replace the single corsOrigin check:
cfg := servepkg.ServerConfig{
    ...
    AllowedOrigins: []string(corsOrigins),
    ...
}
```

### Behaviour

- Zero `--cors-origin` flags: `AllowedOrigins` is nil → wildcard `*` in CORS middleware (existing
  behaviour).
- One or more `--cors-origin` flags: `AllowedOrigins` is populated with all provided values.
- The CORS middleware in `v2/internal/serve/middleware.go` already iterates `AllowedOrigins`, so no
  middleware changes are needed.

### Tests

Add to `v2/internal/serve/middleware_test.go` (or a new `cors_test.go`):

1. **TestCORS_MultipleOrigins_Allowed** — configure two allowed origins; each gets reflected back in
   `Access-Control-Allow-Origin`.
2. **TestCORS_MultipleOrigins_Rejected** — third origin not in list → no CORS headers.
3. **TestCORS_SingleOrigin_BackwardCompat** — single origin still works.

Add a CLI flag test in `v2/cmd/gert/serve_test.go` (or the existing flag test file):

4. **TestServeFlags_CorsOrigin_Repeatable** — parse `--cors-origin A --cors-origin B`, assert
   `AllowedOrigins == ["A","B"]`.

**Estimated:** 25 LOC implementation, 50 LOC tests.

---

## Part C — `--trust-proxy-headers` Security Flag

### Problem

`v2/internal/serve/ratelimit.go` (the `extractIP` function) unconditionally reads
`X-Forwarded-For`:

```go
if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
    if ip := strings.Split(xff, ",")[0]; ip != "" {
        return strings.TrimSpace(ip)
    }
}
host, _, _ := net.SplitHostPort(r.RemoteAddr)
return host
```

This means **any client can forge their IP** by sending a fake `X-Forwarded-For: 1.2.3.4` header,
bypassing per-IP rate limiting entirely. Trusting proxy headers is only safe when `gert serve` is
running behind a known reverse proxy (nginx, Caddy, etc.).

This was noted in Phase 19 (D-19-02) but not acted on. It is a correctness bug for any public
deployment: rate limiting has zero effect if an attacker forges XFF.

### Design

#### C1 — Add `TrustProxyHeaders bool` to `ServerConfig`

**File:** `v2/pkg/serve/serve.go`

```go
// TrustProxyHeaders controls whether X-Forwarded-For and X-Real-IP headers are trusted
// for IP extraction in the rate-limit middleware.
// Enable ONLY when gert serve is deployed behind a trusted reverse proxy (nginx, Caddy, etc.).
// When false (default), rate limiting uses RemoteAddr exclusively.
// WARNING: enabling this on a directly-internet-facing server allows IP spoofing.
TrustProxyHeaders bool
```

#### C2 — Thread `TrustProxyHeaders` through to the rate-limit middleware

**File:** `v2/internal/serve/ratelimit.go`

Change `newRateLimitMiddleware` to accept the flag:

```go
func newRateLimitMiddleware(limit int, trustProxyHeaders bool) func(http.Handler) http.Handler {
    ...
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ip := extractIP(r, trustProxyHeaders)
            ...
        })
    }
}

func extractIP(r *http.Request, trustProxy bool) string {
    if trustProxy {
        if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
            if ip := strings.Split(xff, ",")[0]; ip != "" {
                return strings.TrimSpace(ip)
            }
        }
    }
    host, _, _ := net.SplitHostPort(r.RemoteAddr)
    return host
}
```

**File:** `v2/internal/serve/server.go` — pass `cfg.TrustProxyHeaders` when constructing middleware.

#### C3 — Add `--trust-proxy-headers` flag to CLI

**File:** `v2/cmd/gert/serve.go`

```go
trustProxyHeaders := fs.Bool("trust-proxy-headers", false,
    "Trust X-Forwarded-For for rate-limit IP extraction (only safe behind a reverse proxy)")
```

Wire into `ServerConfig.TrustProxyHeaders`.

### Tests

Update `v2/internal/serve/ratelimit_test.go`:

1. **TestRateLimit_XFF_Ignored_WhenTrustDisabled** — send `X-Forwarded-For: 1.2.3.4`, confirm rate
   limiting uses `RemoteAddr` not the forged IP (i.e., requests from different XFF but same
   RemoteAddr are bucketed together).
2. **TestRateLimit_XFF_Trusted_WhenTrustEnabled** — same setup with `trustProxyHeaders=true`,
   confirm XFF is used (existing behaviour).
3. Existing `TestRateLimit_XFF` test must be updated to pass `trustProxyHeaders=true` explicitly.

**Estimated:** 25 LOC implementation, 40 LOC tests.

---

## Summary

| Part | NBI | Files Touched | LOC | Effort |
|------|-----|---------------|-----|--------|
| A — completedAt + godoc + fixtures | NBI-16-05, NBI-16-07 | `rpc.go`, `rpc_test.go`, 4 new fixture files | ~90 | 0.5 day |
| B — Multiple CORS origins | NBI-16-06 | `serve.go` (cmd), `middleware_test.go` | ~75 | 0.4 day |
| C — Trust-proxy flag | — | `serve.go` (pkg), `ratelimit.go`, `server.go`, `serve.go` (cmd), `ratelimit_test.go` | ~65 | 0.5 day |
| **Total** | | **7 files** | **~230** | **~1.4 days** |

## Verification Commands

```bash
# After implementation:
cd v2
go build ./...
go test ./internal/serve/... -race -count=1
go test ./cmd/gert/... -race -count=1

# Confirm completedAt appears for a real persisted run:
# 1. Start a run, let it complete
# 2. curl -s -X POST http://localhost:7778/rpc \
#      -d '{"jsonrpc":"2.0","id":1,"method":"run.get","params":{"runID":"<id>"}}' | jq .result.completedAt

# Confirm multiple origins:
# gert serve --cors-origin https://a.example.com --cors-origin https://b.example.com &
# curl -H "Origin: https://a.example.com" -I http://localhost:7778/health
# # Should see: Access-Control-Allow-Origin: https://a.example.com

# Confirm trust-proxy-headers default is off:
# gert serve &
# curl -H "X-Forwarded-For: 1.1.1.1" http://localhost:7778/health  # should use actual RemoteAddr
```
