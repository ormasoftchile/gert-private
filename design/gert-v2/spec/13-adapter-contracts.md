# Adapter Contracts

Gert v2 decouples the core runtime from presentation adapters (TUI, Web, VS Code) and integration systems (CI/CD, observability backends, approval workflows) through explicitly versioned contracts. An **adapter contract** is a stable, versioned interface that gert commits to maintain across minor and patch releases. Adapters observe, render, or orchestrate gert runs — they own no execution logic.

Four primary adapter surfaces are defined: JSON-RPC execution contract (`exec/v2`), WebSocket event stream contract (`events/v2`), tool invocation contract (`tool/v2`), and extension handshake contract (`extension/v2`).

---

## Contract Categories

### JSON-RPC Execution Contract (`exec/v2`)

**Transport:** stdio JSON-RPC 2.0 (`gert serve --stdio`) or HTTP POST (`gert serve --http`, endpoint `/rpc`)

#### Methods

| Method         | Description                                                                                 |
|----------------|---------------------------------------------------------------------------------------------|
| `exec/start`   | Start a new runbook execution. Params: `runbookPath`, `inputs`, `mode`, `actorId`. Returns: `runId`, `status`. |
| `exec/next`    | Advance the run by one step or until a pause point. Params: `runId`. Returns: `stepId`, `status`, `output`. |
| `exec/cancel`  | Cancel an in-progress run. Params: `runId`, `reason`. Returns: `cancelled: bool`. |
| `exec/status`  | Get current run status. Params: `runId`. Returns: `status`, `currentStepId`, `progress`. |
| `run/list`     | List all runs. Params: `filter`, `limit`. Returns: `runs: [RunSummary]`. |
| `run/get`      | Get full run details. Params: `runId`. Returns: `run: RunDetails`. |

#### Request — `exec/start`

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "exec/start",
  "params": {
    "runbookPath": "./runbooks/incident-triage.yaml",
    "inputs": {
      "incident_id": "INC-12345",
      "severity": "high"
    },
    "mode": "real",
    "actorId": "ops@acme.example"
  }
}
```

#### Response — `exec/start`

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "runId": "run-abc123",
    "status": "running",
    "createdAt": "2026-04-18T12:00:00Z",
    "traceFile": "./traces/run-abc123.jsonl"
  }
}
```

#### Error Envelope

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "error": {
    "code": 1001,
    "message": "run not found",
    "data": {
      "runId": "run-xyz789",
      "category": "client_error",
      "retryable": false
    }
  }
}
```

#### Stability Guarantee (`exec/v2`)

- All required fields in request/response schemas are stable.
- Optional fields may be added in minor versions; clients MUST ignore unknown fields.
- Required fields will not be removed or renamed (breaking change → `exec/v3`).
- Error codes are stable; new codes may be added but existing codes will not change semantics.

---

### WebSocket Event Stream Contract (`events/v2`)

**Transport:** WebSocket (`wss://` or `ws://`) at `/ws` or `/events`

#### Connection Lifecycle

1. Client establishes WebSocket connection to `/ws`.
2. Client sends a `subscribe` message with `runId`.
3. Server streams all events for that run in order.
4. Client may disconnect at any time; server closes gracefully.

#### Subscribe Message

```json
{
  "action": "subscribe",
  "runId": "run-abc123",
  "sinceSequence": 0
}
```

#### Event Envelope

Events are transmitted as JSON objects (one per WebSocket message), identical to the trace event format from §12:

```json
{
  "eventType": "step/started",
  "timestamp": "2026-04-18T12:00:00Z",
  "runId": "run-abc123",
  "stepId": "lookup_incident",
  "sequence": 42,
  "payload": {
    "stepType": "tool",
    "toolName": "acme.pd-lookup"
  }
}
```

Required fields on all events: `eventType`, `timestamp` (RFC 3339), `runId`, `sequence` (monotonically increasing from 0), `payload`.

#### Reconnection Protocol

Client may reconnect and provide `sinceSequence` to resume from the last received event:

```json
{
  "action": "subscribe",
  "runId": "run-abc123",
  "sinceSequence": 42
}
```

Server replays all events with `sequence > 42`.

#### Stability Guarantee (`events/v2`)

- Envelope fields (`eventType`, `timestamp`, `runId`, `sequence`) are stable.
- New event types may be added in minor versions; existing types will not change their `eventType` string.
- Payload schemas may add optional fields; clients MUST ignore unknown fields.

---

### Tool Invocation Contract (`tool/v2`)

All tools — built-in, project-level, or extension-contributed — implement this contract.

**Transport:** stdio JSON-RPC 2.0

#### Invocation Request

```json
{
  "jsonrpc": "2.0",
  "id": "invocation-5",
  "method": "invoke",
  "params": {
    "action": "lookup",
    "inputs": {
      "id": "INC-12345"
    },
    "context": {
      "runId": "run-abc123",
      "stepId": "lookup_incident",
      "traceId": "trace-xyz789",
      "userId": "ops@acme.example",
      "mode": "real",
      "attempt": 1
    }
  }
}
```

#### Success Response

```json
{
  "jsonrpc": "2.0",
  "id": "invocation-5",
  "result": {
    "outputs": {
      "incident": {
        "id": "INC-12345",
        "severity": "high",
        "status": "open"
      }
    },
    "exitCode": 0,
    "telemetry": {
      "durationMs": 320,
      "startedAt": "2026-04-18T12:00:00Z",
      "finishedAt": "2026-04-18T12:00:00.320Z"
    }
  }
}
```

#### Error Response

```json
{
  "jsonrpc": "2.0",
  "id": "invocation-5",
  "error": {
    "code": -32050,
    "message": "incident not found",
    "data": {
      "category": "runtime",
      "retryable": false,
      "details": {
        "incidentId": "INC-12345",
        "statusCode": 404
      }
    }
  }
}
```

#### Cancellation Request

```json
{
  "jsonrpc": "2.0",
  "id": "cancel-5",
  "method": "cancel",
  "params": {
    "invocationId": "invocation-5",
    "reason": "run-cancelled"
  }
}
```

#### MCP Tool Adapter

When an MCP server is used as a tool source, the gert host wraps the MCP `tools/call` protocol:

1. Translates the `invoke` request to an MCP `tools/call` request.
2. Forwards the `context` object as `_gert_context` in the MCP request (gert extension to MCP protocol).
3. Translates the MCP response back to the tool invocation contract response.
4. Flattens MCP `content` arrays to a single string for capture.

#### Stability Guarantee (`tool/v2`)

- Request/response envelope fields are stable; `context` object fields are stable (new fields may be added).
- Error code ranges are stable.
- Tools MUST ignore unknown fields in requests.

---

### Extension Handshake Contract (`extension/v2`)

**Transport:** stdio JSON-RPC 2.0

#### Initialize Request

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "extension/initialize",
  "params": {
    "host": {
      "name": "gert",
      "version": "2.0.0",
      "apiVersion": "2.0"
    },
    "protocolVersion": "2.0",
    "grantedCapabilities": [
      "capability/tool-registration",
      "capability/network"
    ],
    "runContext": {
      "runId": "run-abc123",
      "mode": "real"
    }
  }
}
```

#### Initialize Response

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "extension": {
      "name": "acme.incident-pack",
      "version": "1.2.0"
    },
    "protocolVersion": "2.0",
    "acknowledgedCapabilities": [
      "capability/tool-registration",
      "capability/network"
    ]
  }
}
```

#### Heartbeat (ping/pong)

```
Request:
{
  "jsonrpc": "2.0",
  "id": "42",
  "method": "extension/ping",
  "params": {}
}

Response:
{
  "jsonrpc": "2.0",
  "id": "42",
  "result": {
    "status": "ok"
  }
}
```

#### Stability Guarantee (`extension/v2`)

- `extension/initialize` request/response schema is stable.
- New capabilities may be added; extensions MUST ignore unknown capabilities in `grantedCapabilities`.
- `extension/ping` method signature is stable.
- Extension error codes (§4) are stable.

---

## Contract Versioning

Version format: `{surface}/{major}` (e.g. `exec/v2`, `events/v2`). There is no minor version — any breaking change bumps the major version.

### Version Declaration

**HTTP-based contracts:**

```
GET /ws HTTP/1.1
Host: gert.acme.example:8443
X-Gert-Contract: events/v2
```

Server responds with `X-Gert-Contract: events/v2`. Unsupported version → HTTP 400:

```json
{
  "error": {
    "code": 4001,
    "message": "unsupported contract version",
    "data": {
      "requestedVersion": "events/v3",
      "supportedVersions": ["events/v2"]
    }
  }
}
```

**stdio-based contracts:** Version declared in the first `initialize` message as `"protocolVersion": "2.0"`. Unsupported version returns error code `-32001`.

### Breaking vs. Non-Breaking Changes

**Non-breaking (no version bump):**
- Adding optional fields to requests or responses
- Adding new methods, error codes, or event types

Consumers MUST implement forward compatibility: ignore unknown fields, methods, and event types.

**Breaking (major version bump):**
- Removing or renaming required fields
- Changing semantics of existing methods
- Removing methods
- Changing error code meanings
- Incompatible transport changes

### Deprecation Policy

- Previous version supported for **12 months** after new version GA date.
- Both versions supported simultaneously during deprecation period.
- After 12 months, old version is removed.

### Forward and Backward Compatibility

- **Forward (clients):** MUST ignore unknown fields in responses.
- **Backward (servers):** MUST accept requests that omit optional fields; MUST NOT require new optional fields.

---

## Error Codes

Canonical error code registry. Codes are stable across contract versions.

| Code | Description                                      | Retryable |
|------|--------------------------------------------------|-----------|
| **1xxx — Execution Errors** |||
| 1001 | Run not found (invalid `runId`)                  | No        |
| 1002 | Step failed (runtime error in step execution)    | Maybe     |
| 1003 | Run cancelled (user or timeout cancellation)     | No        |
| 1004 | Run already exists (duplicate `runId`)           | No        |
| **2xxx — Governance Errors** |||
| 2001 | Command denied (not in allowlist or in denylist) | No        |
| 2002 | Approval timeout (no approval received in time)  | No        |
| 2003 | Approval rejected                                | No        |
| 2004 | Insufficient approvals (min not reached)         | No        |
| **3xxx — Schema Errors** |||
| 3001 | Schema validation failed                         | No        |
| 3002 | Unknown step type                                | No        |
| 3003 | Import cycle detected                            | No        |
| **4xxx — Integration Errors** |||
| 4001 | Tool not found                                   | No        |
| 4002 | Extension handshake failed                       | Maybe     |
| 4003 | Input resolution failed                          | Maybe     |
| 4004 | Unsupported contract version                     | No        |
| **5xxx — Server Errors** |||
| 5001 | Internal error                                   | Maybe     |
| 5002 | Resource exhausted                               | Yes       |
| 5003 | Service unavailable                              | Yes       |

**Error envelope structure:**

```json
{
  "error": {
    "code": 2001,
    "message": "command denied: not in allowlist",
    "data": {
      "category": "governance",
      "retryable": false,
      "details": {
        "command": "rm",
        "reason": "in_denylist"
      }
    }
  }
}
```

- Existing error codes will not change their meaning.
- New codes may be added in minor versions.
- Clients MUST handle unknown error codes gracefully.

---

## Contract Test Suite

Located in `testdata/contracts/`:

```
testdata/contracts/
  exec-v2/
    01-start-run.test.yaml
    02-cancel-run.test.yaml
    03-list-runs.test.yaml
  events-v2/
    01-subscribe.test.yaml
    02-reconnect.test.yaml
  tool-v2/
    01-invoke.test.yaml
    02-cancel.test.yaml
  extension-v2/
    01-initialize.test.yaml
    02-ping.test.yaml
```

Each test case YAML contains: `description`, `request`, `expectedResponse`, `preconditions`.

```bash
$ gert contract test --adapter=vscode
Running contract tests for adapter: vscode
  ✓ exec-v2/01-start-run (passed)
  ✓ exec-v2/02-cancel-run (passed)
  ✗ events-v2/01-subscribe (failed: missing 'sequence' field)

Contract test results: 2 passed, 1 failed
```

Adapter implementations SHOULD achieve 100% pass rate before release. Failed contract tests block merges to main branch.

---

## Contract Stability Guarantees

1. **Semantic versioning:** major version bumps indicate breaking changes.
2. **12-month deprecation period:** old versions supported for 12 months after new major version.
3. **Additive changes only:** within a major version, only non-breaking changes permitted.
4. **Error code stability:** codes do not change meaning within a major version.
5. **Forward compatibility:** clients must ignore unknown fields; servers must accept omitted optional fields.

---

## Contract Documentation and Tooling

- OpenAPI 3.1 specs: `specs/contracts/exec-v2.openapi.yaml`, `specs/contracts/events-v2.openapi.yaml`
- JSON Schema (Draft 2020-12) in `specs/contracts/schemas/` — used for runtime validation, contract testing, and client codegen
- Reference implementations:
  - VS Code extension (`vscode/`) — reference for `exec/v2` and `events/v2`
  - Built-in tools (`ext/builtin/tools/`) — reference for `tool/v2`
  - MCP adapter (`ext/toolruntime/mcp_adapter.go`) — wrapping external protocol to satisfy `tool/v2`

```bash
$ gert codegen client --language=typescript --output=./clients/ts/
Generated TypeScript client for exec/v2 and events/v2
```

Official client libraries: TypeScript (VS Code, web), Python (CI/CD), Go (extension authors).
