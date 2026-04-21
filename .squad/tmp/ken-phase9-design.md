# Phase 9 — `gert serve` (HTTP/WS/SSE Server)

**Author:** Ken (Architect)  
**Date:** 2026-04-22  
**Status:** DESIGN COMPLETE  
**Implementor:** Brian (assigned)

---

## 1. Overview

Phase 9 introduces `gert serve` as an HTTP-based adapter to the gert v2 runtime. This replaces the v1 stdio-only JSON-RPC model with a network-accessible server that supports:

- **JSON-RPC 2.0 over HTTP POST** — synchronous request/response for run control
- **WebSocket** — real-time event streaming to connected clients
- **Server-Sent Events (SSE)** — fallback event streaming for environments without WebSocket support
- **Health endpoint** — liveness probe for container orchestrators

The design doc in `sections/02-architecture.tex` establishes that `gert serve` is an **adapter** in the API/Adapter Layer. It does NOT modify execution semantics — it translates network I/O into `Engine.Start()` / `RunHandle.Next()` calls and forwards `EventBus` events to connected clients.

---

## 2. Package Layout

```
v2/
├── cmd/
│   └── serve/
│       └── main.go              — binary entry: gert serve
│
├── pkg/
│   └── serve/
│       ├── serve.go             — public types: ServerConfig, RunEvent, etc.
│       └── doc.go               — package documentation
│
├── internal/
│   └── serve/
│       ├── server.go            — Server struct, Start/Stop lifecycle
│       ├── rpc.go               — JSON-RPC 2.0 dispatch over HTTP POST /rpc
│       ├── ws.go                — WebSocket hub /ws (event broadcast)
│       ├── sse.go               — SSE broadcaster /events
│       ├── health.go            — GET /health → JSON status
│       ├── middleware.go        — CORS, request ID, logging middleware
│       ├── registry.go          — RunRegistry: thread-safe run handle map
│       ├── events.go            — EventBridge: eventbus → WS/SSE fanout
│       ├── server_test.go       — lifecycle tests
│       ├── rpc_test.go          — JSON-RPC method tests
│       ├── ws_test.go           — WebSocket integration tests
│       ├── sse_test.go          — SSE integration tests
│       ├── health_test.go       — health endpoint tests
│       ├── registry_test.go     — RunRegistry concurrency tests
│       └── events_test.go       — EventBridge tests
│
└── pkg/
    └── testutil/
        └── fake_serve_client.go — test helper: HTTP/WS client for integration tests
```

### Dependency Rules

- `internal/serve` imports `pkg/serve` (public types), `pkg/engine`, `pkg/eventbus`
- `internal/serve` does NOT import `internal/engine` or any other `internal/` package
- `pkg/serve` is a leaf package (no internal imports)
- `cmd/serve/main.go` imports `internal/serve` for construction + `pkg/serve` for config

---

## 3. Endpoints

| Method | Path       | Content-Type          | Auth | Description                           |
|--------|------------|-----------------------|------|---------------------------------------|
| POST   | `/rpc`     | application/json      | None | JSON-RPC 2.0 envelope                 |
| GET    | `/ws`      | —                     | None | WebSocket upgrade → event stream      |
| GET    | `/events`  | text/event-stream     | None | SSE fallback → same event stream      |
| GET    | `/health`  | application/json      | None | Liveness/readiness probe              |

### 3.1 POST /rpc — JSON-RPC 2.0

Request body MUST be a valid JSON-RPC 2.0 request:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "run.start",
  "params": { "runbookPath": "ops/deploy.yaml", "inputs": {"env": "prod"} }
}
```

Response follows JSON-RPC 2.0:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": { "runID": "550e8400-e29b-41d4-a716-446655440000" }
}
```

### 3.2 GET /ws — WebSocket

Upgrade to WebSocket. Server pushes events as JSON frames. Client may send ping/pong but no application-level messages are expected from client.

### 3.3 GET /events — SSE

Standard SSE connection. Each event is:

```
event: step/started
data: {"type":"step/started","runID":"...","stepID":"...","ts":"..."}

```

### 3.4 GET /health

```json
{"status": "ok", "version": "v2", "uptime_seconds": 142}
```

Always returns HTTP 200. No authentication.

---

## 4. JSON-RPC Method Contracts

### 4.1 `run.start`

Start a new runbook execution.

**Request params:**
```json
{
  "runbookPath": "string (required) — path to .yaml runbook file",
  "inputs": "map[string]any (optional) — input variables",
  "mode": "string (optional) — 'real'|'dry-run', default 'real'",
  "actor": "string (optional) — identity, defaults to 'rpc-client'"
}
```

**Response result:**
```json
{
  "runID": "string — UUID v4 of the created run"
}
```

**Errors:**
| Code   | Message                  | When                                    |
|--------|--------------------------|-----------------------------------------|
| -32602 | Invalid params           | Missing runbookPath or invalid inputs    |
| -32000 | Runbook not found        | File does not exist                      |
| -32001 | Parse error              | Runbook YAML is malformed                |
| -32002 | Validation failed        | Schema validation fails                  |
| -32003 | Plan error               | Planner rejects the runbook              |

### 4.2 `run.next`

Advance the run by one step.

**Request params:**
```json
{
  "runID": "string (required) — UUID of an active run"
}
```

**Response result:**
```json
{
  "stepID": "string — ID of the step that was executed",
  "state": "string — 'completed'|'failed'|'skipped'|'waiting'",
  "output": "map[string]any — step output (may be null)"
}
```

**Errors:**
| Code   | Message                  | When                                    |
|--------|--------------------------|-----------------------------------------|
| -32602 | Invalid params           | Missing or empty runID                   |
| -32010 | Run not found            | No run with this ID in registry          |
| -32011 | Run not active           | Run already completed/failed/cancelled   |
| -32012 | Run completed            | No more steps (io.EOF equivalent)        |

### 4.3 `run.cancel`

Cancel a running execution.

**Request params:**
```json
{
  "runID": "string (required)",
  "reason": "string (optional) — cancellation reason"
}
```

**Response result:**
```json
{}
```

**Errors:**
| Code   | Message                  | When                                    |
|--------|--------------------------|-----------------------------------------|
| -32602 | Invalid params           | Missing runID                            |
| -32010 | Run not found            | No run with this ID                      |
| -32011 | Run not active           | Already terminated                       |

### 4.4 `run.status`

Get current state of a run.

**Request params:**
```json
{
  "runID": "string (required)"
}
```

**Response result:**
```json
{
  "runID": "string",
  "state": "string — pending|running|waiting|completed|failed|cancelled",
  "currentStep": "string — step ID or empty",
  "runbookPath": "string",
  "startedAt": "string — RFC3339",
  "updatedAt": "string — RFC3339"
}
```

**Errors:**
| Code   | Message                  | When                                    |
|--------|--------------------------|-----------------------------------------|
| -32602 | Invalid params           | Missing runID                            |
| -32010 | Run not found            | No run with this ID                      |

### 4.5 `run.list`

List all runs in the registry (active + recently completed).

**Request params:**
```json
{}
```

**Response result:**
```json
[
  {
    "runID": "string",
    "state": "string",
    "runbookPath": "string",
    "startedAt": "string — RFC3339"
  }
]
```

**Errors:** None (always succeeds; returns empty array if no runs).

---

## 5. Event Wire Format (WebSocket + SSE)

All events pushed to clients use a uniform JSON envelope:

```json
{
  "type": "string — event kind (e.g., 'step/started')",
  "runID": "string — UUID of the run",
  "sequence": 42,
  "ts": "string — RFC3339 with microsecond precision",
  "payload": {}
}
```

### 5.1 Event Types Forwarded

| Type              | Payload                                              |
|-------------------|------------------------------------------------------|
| `run/started`     | `{"runbookPath": "...", "mode": "real"}`             |
| `step/started`    | `{"stepID": "...", "stepKind": "shell"}`             |
| `step/completed`  | `{"stepID": "...", "output": {...}, "status": "..."}` |
| `step/failed`     | `{"stepID": "...", "error": "..."}`                  |
| `run/completed`   | `{"outcome": "success", "duration_ms": 1234}`       |
| `run/failed`      | `{"error": "...", "failedStep": "..."}`              |
| `run/cancelled`   | `{"reason": "user request"}`                         |

### 5.2 SSE Wire Format

```
event: step/started
id: 42
data: {"type":"step/started","runID":"abc","sequence":42,"ts":"2026-04-22T10:00:00.000000Z","payload":{"stepID":"s1","stepKind":"shell"}}

```

- `event:` = the type field
- `id:` = the sequence number (enables `Last-Event-ID` reconnection)
- `data:` = full JSON envelope on a single line

### 5.3 WebSocket Wire Format

Each message is a single JSON object (the envelope above), sent as a text frame.

### 5.4 Subscription Filtering

On connect, clients may provide a query parameter to filter by run:

- `GET /ws?runID=abc` — only events for run `abc`
- `GET /events?runID=abc` — same for SSE
- No filter → all events from all runs

---

## 6. Integration with Engine

### 6.1 Architecture Diagram

```
┌────────────────────────────────────────────────────────────┐
│  cmd/serve/main.go                                          │
│    → constructs EngineConfig + ServerConfig                  │
│    → calls NewServer(cfg).Start(ctx)                         │
└────────────────────────────────────────────────────────────┘
         │
         ▼
┌────────────────────────────────────────────────────────────┐
│  internal/serve/server.go                                    │
│    Server {                                                  │
│      engine   engine.Engine                                  │
│      registry *RunRegistry                                   │
│      hub      *EventHub                                      │
│      router   *http.ServeMux                                 │
│    }                                                         │
└───────────────┬─────────────────────┬──────────────────────┘
                │ run.start           │ events
                ▼                     ▼
┌─────────────────────┐  ┌─────────────────────────────────┐
│ pkg/engine.Engine    │  │ pkg/eventbus.EventBus            │
│   .Start() → Handle │  │   .Subscribe() → <-chan Event    │
│   Handle.Next()      │  └─────────────────────────────────┘
│   Handle.Cancel()    │              │
│   Handle.State()     │              ▼
└─────────────────────┘  ┌─────────────────────────────────┐
                          │ internal/serve/events.go          │
                          │   EventBridge                     │
                          │     → WS Hub broadcast            │
                          │     → SSE broadcaster             │
                          └─────────────────────────────────┘
```

### 6.2 Flow: `run.start`

1. RPC handler parses request, validates params
2. Load runbook file from `runbookPath`
3. Parse + validate + plan (using existing parser/planner pipeline)
4. Call `engine.Start(ctx, plan, opts)` → returns `RunHandle`
5. Create `RunEntry` in `RunRegistry` with handle + metadata
6. Start event pump goroutine: drains `handle.Events()` → hub broadcast
7. Return `{"runID": entry.ID}` to client

### 6.3 Flow: `run.next`

1. Look up run in registry by ID
2. Call `handle.Next(ctx)` — blocks until step completes
3. Map `StepResult` → RPC response shape
4. If `io.EOF`, mark run state as completed, return error code -32012

### 6.4 Event Pump Goroutine

For each active run, a goroutine:

```go
func (s *Server) pumpEvents(entry *RunEntry) {
    for ev := range entry.Handle.Events() {
        wireEv := toWireEvent(ev, entry.ID)
        s.hub.Broadcast(wireEv)
    }
    // channel closed → run terminated
    entry.MarkCompleted()
}
```

---

## 7. RunRegistry

Thread-safe in-memory registry of active and recently-completed runs.

### 7.1 Data Structure

```go
type RunEntry struct {
    ID          string
    RunbookPath string
    Handle      engine.RunHandle
    Cancel      context.CancelFunc
    State       engine.RunStatus      // mirrors Handle.State().Status
    StartedAt   time.Time
    CompletedAt time.Time             // zero until terminal
}

type RunRegistry struct {
    mu      sync.RWMutex
    entries map[string]*RunEntry
}
```

### 7.2 Operations

```go
func (r *RunRegistry) Add(entry *RunEntry)
func (r *RunRegistry) Get(runID string) (*RunEntry, bool)
func (r *RunRegistry) Remove(runID string)
func (r *RunRegistry) List() []*RunEntry
func (r *RunRegistry) GC(maxAge time.Duration) int  // remove completed runs older than maxAge
```

### 7.3 Garbage Collection

A background goroutine runs every 60 seconds and calls `GC(cfg.MaxRunAge)`. Completed/failed/cancelled runs older than `MaxRunAge` are evicted from the map. Default `MaxRunAge`: 1 hour.

---

## 8. Server Lifecycle

### 8.1 Construction

```go
func NewServer(cfg ServerConfig) (*Server, error)
```

Validates config, creates engine from `cfg.EngineConfig`, initializes:
- HTTP mux with routes
- WebSocket hub
- SSE broadcaster  
- RunRegistry
- Middleware chain

### 8.2 Start

```go
func (s *Server) Start(ctx context.Context) error
```

1. Bind to `cfg.Addr` (default `:7778`)
2. Start GC goroutine
3. Start HTTP server with graceful shutdown wired to context cancellation
4. Block until context is cancelled or server error

### 8.3 Stop

```go
func (s *Server) Stop(ctx context.Context) error
```

1. Cancel all active runs (sends `run/cancelled` events)
2. Close all WebSocket connections (1000 Normal Closure)
3. Drain SSE connections (close response writers)
4. Shutdown HTTP server with `ctx` deadline
5. Wait for all event pump goroutines to exit

---

## 9. ServerConfig

```go
// ServerConfig holds all configuration for the gert serve HTTP server.
type ServerConfig struct {
    // Addr is the TCP address to listen on (default: ":7778").
    Addr string

    // EngineConfig is passed to engine construction.
    EngineConfig engine.EngineConfig

    // ReadTimeout is the maximum duration for reading request body.
    // Default: 30s.
    ReadTimeout time.Duration

    // WriteTimeout is the maximum duration before timing out response writes.
    // Default: 60s.
    WriteTimeout time.Duration

    // MaxRunAge is how long completed/failed runs remain in the registry.
    // Default: 1h. After this duration, GC removes them.
    MaxRunAge time.Duration

    // WSPingInterval is how often the server pings WebSocket clients.
    // Default: 30s. Clients that don't pong within 10s are disconnected.
    WSPingInterval time.Duration

    // EventBufferSize is the channel buffer for per-client event delivery.
    // Default: 256. Events are dropped for slow consumers.
    EventBufferSize int
}
```

---

## 10. cmd/serve/main.go

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/ormasoftchile/gert/v2/internal/serve"
    servepkg "github.com/ormasoftchile/gert/v2/pkg/serve"
)

func main() {
    addr := flag.String("addr", ":7778", "listen address")
    flag.Parse()

    cfg := servepkg.ServerConfig{
        Addr: *addr,
        // EngineConfig populated from environment/flags
    }

    srv, err := serve.NewServer(cfg)
    if err != nil {
        log.Fatalf("gert serve: %v", err)
    }

    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    fmt.Fprintf(os.Stderr, "gert serve listening on %s\n", cfg.Addr)
    if err := srv.Start(ctx); err != nil {
        log.Fatalf("gert serve: %v", err)
    }
}
```

---

## 11. Middleware Stack

Applied to all routes in order:

1. **Request ID** — generates UUID, sets `X-Request-ID` header, adds to context
2. **Logging** — structured JSON log (method, path, status, duration, request_id)
3. **CORS** — wide open for development (`*` origin, all methods); locked down via config in future
4. **Recovery** — catches panics, logs stack trace, returns 500

WebSocket and SSE handlers bypass timeout middleware (long-lived connections).

---

## 12. Test Plan (24 tests)

### RPC Tests (rpc_test.go)

| # | Test                                    | Validates                                      |
|---|------------------------------------------|------------------------------------------------|
| 1 | TestRPC_RunStart_Success                 | Returns runID, run appears in registry         |
| 2 | TestRPC_RunStart_InvalidParams           | Missing runbookPath → -32602                   |
| 3 | TestRPC_RunStart_RunbookNotFound         | Bad path → -32000                              |
| 4 | TestRPC_RunNext_Advances                 | StepResult returned with output                |
| 5 | TestRPC_RunNext_EOF                      | After last step → -32012                       |
| 6 | TestRPC_RunNext_RunNotFound              | Unknown runID → -32010                         |
| 7 | TestRPC_RunCancel_Stops                  | Run state → cancelled                          |
| 8 | TestRPC_RunStatus_ReflectsState          | Returns correct state/currentStep              |
| 9 | TestRPC_RunList_ShowsActiveRuns          | Lists all registry entries                     |
| 10| TestRPC_UnknownMethod                    | → -32601 Method not found                      |
| 11| TestRPC_MalformedJSON                    | → -32700 Parse error                           |
| 12| TestRPC_MissingJsonrpcField              | → -32600 Invalid Request                       |

### WebSocket Tests (ws_test.go)

| # | Test                                    | Validates                                      |
|---|------------------------------------------|------------------------------------------------|
| 13| TestWS_ConnectReceivesEvents             | step/started + step/completed delivered         |
| 14| TestWS_FilterByRunID                     | Only events for requested run                  |
| 15| TestWS_RunCompleted_ReceivesTerminal     | run/completed event received                   |
| 16| TestWS_SlowConsumer_DropsEvents          | Buffer full → no block, event dropped          |

### SSE Tests (sse_test.go)

| # | Test                                    | Validates                                      |
|---|------------------------------------------|------------------------------------------------|
| 17| TestSSE_ConnectReceivesEvents            | text/event-stream format correct               |
| 18| TestSSE_LastEventID_Replay               | Reconnect with Last-Event-ID resumes           |
| 19| TestSSE_FilterByRunID                    | Filter works same as WS                        |

### Health Tests (health_test.go)

| # | Test                                    | Validates                                      |
|---|------------------------------------------|------------------------------------------------|
| 20| TestHealth_Returns200                    | Status OK, version present                     |

### Registry Tests (registry_test.go)

| # | Test                                    | Validates                                      |
|---|------------------------------------------|------------------------------------------------|
| 21| TestRegistry_ConcurrentAddGet            | No race under concurrent access                |
| 22| TestRegistry_GC_RemovesOldRuns           | Completed runs older than MaxRunAge removed    |

### Server Lifecycle Tests (server_test.go)

| # | Test                                    | Validates                                      |
|---|------------------------------------------|------------------------------------------------|
| 23| TestServer_StartStop_Graceful            | Clean shutdown, no goroutine leaks             |
| 24| TestServer_CancelActiveRuns_OnStop       | Active runs cancelled before shutdown          |

All tests MUST pass with `-race`.

---

## 13. Decisions

### D1: HTTP Framework — `net/http` stdlib only

**Decision:** Use Go standard library `net/http` with `http.ServeMux` (Go 1.22+ enhanced routing).

**Rationale:** gert has zero framework dependencies by convention. The enhanced mux in Go 1.22+ supports method+path routing (`POST /rpc`, `GET /ws`) natively. No Gin/Echo/Chi needed for 4 routes.

### D2: WebSocket Library — `github.com/coder/websocket`

**Decision:** Use `github.com/coder/websocket` (formerly `nhooyr.io/websocket`).

**Rationale:** `gorilla/websocket` is archived/unmaintained. `coder/websocket` is the actively maintained successor with a simpler API, proper context support, and `io.Reader`/`io.Writer` interfaces. It supports `net/http` middleware natively and has no CGO dependency.

### D3: SSE — Pure stdlib

**Decision:** Implement SSE as a plain `http.Handler` with `text/event-stream` Content-Type and chunked transfer encoding. Use `http.Flusher` interface for immediate delivery.

**Rationale:** SSE is trivial over HTTP/1.1 chunked responses. No library needed.

### D4: JSON-RPC Version — 2.0 strict

**Decision:** Require `"jsonrpc": "2.0"` in all requests. Reject requests missing this field with error code -32600 (Invalid Request).

**Rationale:** Aligns with the spec in `02-architecture.tex` which explicitly references JSON-RPC 2.0. No backward compatibility needed (greenfield).

### D5: Run Execution Model — Step-by-step (client-driven)

**Decision:** Execution is client-driven via `run.next`. The server does NOT auto-advance steps.

**Rationale:** This matches the existing `RunHandle.Next()` interface. The adapter (VS Code extension, TUI, web client) controls pacing. Clients that want full auto-execution simply loop `run.next` until EOF. This preserves the ability to insert human approval gates between steps. A future `run.startAuto` method could wrap this pattern for convenience.

### D6: Auth — None in Phase 9

**Decision:** No authentication or authorization in this phase. CORS allows all origins.

**Rationale:** Phase 9 is development/local use. Auth is explicitly a future phase (the design doc states this). Security hardening will be a separate phase with token-based auth and restricted CORS.

### D7: Run Cleanup — MaxRunAge TTL with GC

**Decision:** Completed runs are garbage-collected after `MaxRunAge` (default 1 hour). No explicit `run.dispose` method.

**Rationale:** Simplest correct behavior. Clients don't need to manage run lifecycle. The GC goroutine runs every 60s. If a client needs longer retention, they increase `MaxRunAge` in config. Explicit dispose can be added later if needed.

### D8: Event Buffering — 256-slot channel, drop on full

**Decision:** Each connected client (WS or SSE) gets a 256-event buffered channel. If the buffer fills (slow consumer), events are silently dropped. No backpressure to the engine.

**Rationale:** Matches the spec's "Non-blocking emission" principle. The authoritative record is the JSONL trace file. Clients that need guaranteed delivery must read the trace file. The 256 buffer is generous for typical step-by-step execution (most runs have <50 steps). A dropped-event counter is exposed on the health endpoint for observability.

---

## 14. Open Questions (for Cristian)

1. **Port number:** Spec doesn't mandate a port. I chose `:7778` (gert = 7th letter... close enough). Should this match v1's port?
2. **Batch execution:** Should `run.startAuto` (fire-and-forget, auto-advance all steps) be included in Phase 9, or deferred to Phase 10?
3. **Run persistence across restarts:** The spec mentions crash recovery for `wait_for_event`. Should Phase 9 support `RunStore`-backed restart, or is in-memory-only acceptable for the initial cut?

---

## 15. Implementation Notes for Brian

1. **Start with `pkg/serve/serve.go`** — define `ServerConfig`, `RunEvent` envelope type. Leaf package, no dependencies beyond stdlib + `pkg/engine`.
2. **Then `internal/serve/registry.go`** — standalone, testable in isolation with `-race`.
3. **Then `internal/serve/server.go`** — skeleton with Start/Stop and route registration.
4. **Then `internal/serve/health.go`** — simplest endpoint, proves the server runs.
5. **Then `internal/serve/rpc.go`** — core RPC dispatch; use a method map `map[string]Handler`.
6. **Then `internal/serve/ws.go` + `sse.go`** — event broadcasting; share an `EventHub` interface.
7. **Then `internal/serve/events.go`** — bridge from `eventbus.EventBus` to the hub.
8. **Finally `cmd/serve/main.go`** — wire it all together.

Each file should have its `_test.go` sibling written before or alongside.

---

## 16. Dependency Additions

```
github.com/coder/websocket v1.8+   (WebSocket support)
```

No other external dependencies. Everything else is stdlib.
