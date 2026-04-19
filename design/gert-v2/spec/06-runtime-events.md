# Runtime Events and Determinism

Every state change in the gert v2 runtime produces a structured event. Events serve three purposes: (1) adapters subscribe to render UI state without coupling to execution logic, (2) events are appended to the JSONL trace file for audit and replay, and (3) events propagate to observability systems. This section defines the event envelope schema, delivery semantics, the complete normative event catalog, event channels, subscription API, wire format, and required vs. optional event classification.

---

## Event System Design

### Role of Events

Events are **fan-out notifications to subscribers**, not a command or control channel. The Runtime Core emits events as a consequence of state transitions; events do not trigger state transitions.

**Key principles:**
- **One-way flow:** Events flow from the Runtime Core outward to subscribers. Subscribers MUST NOT write events back to the runtime.
- **Non-blocking emission:** Event emission MUST NOT block step execution. The event bus uses a bounded channel with a discard-on-full policy for in-process subscribers.
- **Immutable once emitted:** Events are value objects and cannot be modified after emission.
- **Best-effort delivery to in-process subscribers:** If a subscriber channel is full, the event is dropped. The trace file write is the authoritative record.

### Delivery Semantics

**At-least-once delivery.** The JSONL trace file receives every event via a synchronous write before the runtime proceeds. In-process subscribers receive events on a best-effort basis; if the subscriber channel is full, the event is discarded without blocking.

**In-order per run.** All events within a single run are totally ordered by their `sequence` field. Subscribers processing events asynchronously MUST tolerate out-of-order arrival from concurrent runs.

**Replay determinism.** Events written to the trace file are sufficient to reconstruct the execution timeline. Replay mode re-emits the same events in the same order, allowing adapters to render a historical run indistinguishably from a live run.

---

## Event Envelope

Every event carries a mandatory envelope. Consumers MUST tolerate unknown fields to support additive schema evolution.

| Field | Type | Description |
|---|---|---|
| `event_id` | UUID v4 | Globally unique event identifier |
| `run_id` | UUID | Execution run identifier (stable across pauses/resumes) |
| `runbook_id` | string | Runbook name or path from runbook YAML `meta.name` |
| `timestamp` | RFC3339 | UTC timestamp with microsecond precision |
| `kind` | string | Event type (e.g., `step/started`, `governance/approved`) |
| `sequence` | int64 | Monotonically increasing integer within this run (starting from 0) |
| `payload` | object | Event-specific data (schema varies per `kind`) |

- **Sequence numbers** are scoped to a single `run_id`. They are assigned by `RunStore.WriteTrace` and persist across pause/resume cycles. Not globally unique.
- **Timestamps** use RFC3339 with at least microsecond precision (e.g., `2026-04-18T14:32:05.123456Z`). Use event-specific `duration_ms` fields for duration calculations.
- **Event IDs** are UUID v4, assigned at emission time, globally unique. Used for correlation in distributed tracing and deduplication in forwarding pipelines.

[DIAGRAM: Timeline of a two-step run showing mandatory events (run/started, step/started×2, step/completed×2, run/completed) plus conditional governance/approval events between step 2's started and completed.]

---

## Event Catalog

### Run Lifecycle Events

#### `run/started`

**Emitted:** Once, as the first event in every execution. MUST be the first line in the trace file.

```json
{
  "runbook_id":    "<string>",
  "version":       "<string>",
  "actor":         "<string>",
  "input_hash":    "<string>",
  "mode":          "<string>",
  "client":        "<string>",
  "parent_run_id": "<string>"
}
```

- `input_hash`: SHA256 hex digest of the JSON-serialized input dictionary with keys sorted lexicographically.
- `mode`: `"real"` | `"dry-run"` | `"replay"`
- `client`: `"cli"` | `"tui"` | `"web"` | `"vscode"` | `"mcp"`
- `parent_run_id`: present only if this run was initiated by an `invoke` step.

**Required:** YES. Every run MUST emit this event.

#### `run/completed`

**Emitted:** Once, as the terminal event for a successful or terminated run.

```json
{
  "outcome":     "<string>",
  "step_count":  "<int>",
  "passed":      "<int>",
  "failed":      "<int>",
  "skipped":     "<int>",
  "duration_ms": "<int64>"
}
```

- `outcome`: `"success"` | `"failure"` | `"cancelled"`
- If `outcome` is `"failure"`, the last event before this SHOULD be `step/failed` or `governance/blocked`.

**Required:** YES. Every run that reaches a terminal state MUST emit this event.

#### `run/cancelled`

**Emitted:** When a run is cancelled by operator action (SIGINT, API call, or approval rejection).

```json
{
  "reason":        "<string>",
  "cancelled_by":  "<string>",
  "step_id":       "<string>"
}
```

- `reason`: `"user_interrupt"` | `"approval_rejected"` | `"timeout"`
- Followed immediately by `run/completed` with `outcome: "cancelled"`.

**Required:** YES, if a run is cancelled before natural completion.

---

### Step Lifecycle Events

#### `step/started`

**Emitted:** Immediately before a step executes (after governance checks pass).

```json
{
  "step_id":   "<string>",
  "step_type": "<string>",
  "index":     "<int>"
}
```

- `step_type`: `"cli"` | `"manual"` | `"tool"` | `"invoke"` | `"branch"` | `"iterate"` | `"parallel"` | `"compensate"`
- `index`: zero-based position in the ExecutionPlan.

**Required:** YES. Every step that begins execution MUST emit this event.

#### `step/completed`

**Emitted:** After a step exits successfully.

```json
{
  "step_id":     "<string>",
  "outcome":     "<string>",
  "duration_ms": "<int64>",
  "exit_code":   "<int>"
}
```

- `outcome`: `"success"` | `"skipped"` (skipped for non-taken branch paths or exhausted iterate loops)
- `exit_code`: present only for `cli` and `tool` steps; always 0 for other types.

**Required:** YES, for every step that completes without error.

#### `step/failed`

**Emitted:** When a step exits with a non-zero exit code or runtime error.

```json
{
  "step_id":       "<string>",
  "error_code":    "<string>",
  "error_message": "<string>",
  "retryable":     "<bool>",
  "exit_code":     "<int>"
}
```

- `error_code`: `"exit_nonzero"` | `"timeout"` | `"tool_error"` | `"governance_blocked"` | `"invoke_failed"`
- `error_message` is redacted per governance rules.
- If `retryable` is true, this event will be followed by `step/retrying`.

**Required:** YES, for every step that fails.

#### `step/skipped`

**Emitted:** When a step is not executed due to control flow.

```json
{
  "step_id": "<string>",
  "reason":  "<string>"
}
```

- `reason`: `"branch_not_taken"` | `"iterate_exhausted"` | `"dry_run"` | `"cancelled"`

**Required:** YES, for every step that is planned but not executed.

#### `step/retrying`

**Emitted:** Before each retry attempt (v2.1 feature; requires retry policy).

```json
{
  "step_id":      "<string>",
  "attempt":      "<int>",
  "max_attempts": "<int>",
  "backoff_ms":   "<int64>"
}
```

- `attempt`: 1 = first retry, 2 = second, etc.

**Required:** YES if retry is attempted. Optional in v2.0 (retry not yet implemented).

#### `step/resumed`

**Emitted:** When a step transitions from WAITING state back to RUNNING state (e.g., after a `wait_for_event` receives its event).

```json
{
  "step_id":      "<string>",
  "resume_reason": "<string>"
}
```

- `resume_reason`: `"event_received"` | `"timeout_expired"` | `"cancelled"`

**Required:** YES, for every step that transitions from WAITING to RUNNING.

---

### External Event Events

#### `event/received`

Emitted when an external event arrives and matches a waiting `wait_for_event` step.

```json
{
  "event_id":      "<string>",
  "source":        "<string>",
  "channel":       "<string>",
  "payload":       "<object>",
  "filter_matched": "<string>"
}
```

- `event_id` — the step ID of the waiting `wait_for_event` step
- `source` — event source kind: `webhook` | `message` | `signal` | `channel`
- `channel` — the channel name (for `source: channel`)
- `payload` — the event payload (captured verbatim for audit)
- `filter_matched` — the filter expression that matched (if any)

**Ordering:** Emitted BEFORE the `step/resumed` event (which transitions the step from WAITING → RUNNING).

**Required:** YES, for every external event that matches a waiting step.

---

### Governance Events

#### `governance/command_checked`

**Emitted:** When a command is evaluated against allowlist and denylist.

```json
{
  "step_id":      "<string>",
  "command":      "<string>",
  "verdict":      "<string>",
  "rule_matched": "<string>"
}
```

- `command`: base name (argv[0] stripped of path)
- `verdict`: `"allowed"` | `"denied"`
- `rule_matched`: `"allowlist"` | `"denylist"` | `"default_allow"`

**Required:** MAY be emitted. Recommended for audit trails.

#### `governance/approval_requested`

**Emitted:** When a step's approval gate is entered.

```json
{
  "step_id":         "<string>",
  "approver_roles":  ["<string>"],
  "min_approvals":   "<int>",
  "timeout_seconds": "<int>",
  "message":         "<string>"
}
```

- `timeout_seconds`: 0 if no timeout.
- `message` is redacted.

**Required:** YES, when approval gates are used.

#### `governance/approval_received`

**Emitted:** Each time an approver submits a decision.

```json
{
  "step_id":  "<string>",
  "approver": "<string>",
  "role":     "<string>",
  "decision": "<string>",
  "comment":  "<string>"
}
```

- `decision`: `"approved"` | `"rejected"`
- `comment` is redacted.

**Required:** YES, for every approval decision (including rejections).

#### `governance/redaction_applied`

**Emitted:** When redaction rules match and modify captured output.

```json
{
  "step_id":       "<string>",
  "field_path":    "<string>",
  "pattern_count": "<int>"
}
```

- `field_path`: `"stdout"` | `"stderr"` | `"capture.<var_name>"`
- Actual redaction patterns are NOT logged; only the fact that redaction occurred is recorded.

**Required:** MAY be emitted. Useful for audit compliance.

---

### Extension and Tool Events

#### `extension/loaded`

**Emitted:** When an extension successfully completes the handshake.

```json
{
  "extension_id": "<string>",
  "version":      "<string>",
  "capabilities": ["<string>"]
}
```

**Required:** MAY be emitted.

#### `extension/unloaded`

**Emitted:** When an extension process exits or is terminated.

```json
{
  "extension_id": "<string>",
  "reason":       "<string>"
}
```

- `reason`: `"clean_exit"` | `"crash"` | `"timeout"` | `"killed"`

**Required:** MAY be emitted.

#### `tool/invoked`

**Emitted:** When a tool call is dispatched (before the tool executes).

```json
{
  "step_id":        "<string>",
  "tool_name":      "<string>",
  "transport":      "<string>",
  "correlation_id": "<string>"
}
```

- `transport`: `"stdio"` | `"jsonrpc"` | `"mcp"`
- `correlation_id` links to `tool/completed`.

**Required:** MAY be emitted. Recommended for observability.

#### `tool/completed`

**Emitted:** When a tool call returns.

```json
{
  "step_id":        "<string>",
  "tool_name":      "<string>",
  "correlation_id": "<string>",
  "exit_code":      "<int>",
  "duration_ms":    "<int64>"
}
```

**Required:** MAY be emitted.

---

### Human-in-the-Loop Events

#### `input/prompted`

**Emitted:** When a manual step or input provider displays a prompt.

```json
{
  "step_id":    "<string>",
  "prompt_id":  "<string>",
  "input_type": "<string>",
  "prompt":     "<string>"
}
```

- `step_id`: omitted if prompt is from provider (not a step)
- `input_type`: `"text"` | `"checklist"` | `"attachment"`
- `prompt` is redacted.

**Required:** YES for manual steps. MAY be emitted for provider prompts.

#### `input/received`

**Emitted:** When the operator submits a response.

```json
{
  "step_id":   "<string>",
  "prompt_id": "<string>",
  "redacted":  "<bool>"
}
```

- The actual input value is NOT included (recorded via `step/completed`).
- `redacted` indicates whether the input passed through the redaction pipeline.

**Required:** YES for manual steps.

---

### Saga and Compensation Events (v2.1+)

Reserved for the saga/compensation pattern targeted for v2.1.

#### `saga/compensation_triggered`

**Emitted:** When a step fails and has registered compensation handlers.

```json
{
  "failed_step_id":     "<string>",
  "compensation_steps": ["<string>"]
}
```

**Required:** YES if compensation is triggered (v2.1+).

#### `saga/compensation_completed`

**Emitted:** After all compensation steps complete.

```json
{
  "outcome": "<string>"
}
```

- `outcome`: `"success"` | `"partial"` | `"failed"`

**Required:** YES if compensation is triggered (v2.1+).

---

## Event Channels

### In-Process Event Bus

A Go `chan Event` with a fixed buffer size (default: 100 events). The Runtime Core writes to the channel; adapters read from it. If the channel is full, new events are discarded.

```go
type RunHandle interface {
    // Events returns a read-only channel of events for this run.
    // The channel is closed when the run enters a terminal state.
    Events() <-chan Event
    // ... other methods ...
}
```

The event channel is created at `Runtime.Start()` and closed when the run emits `run/completed` or `run/cancelled`. Subscribers MUST drain the channel to avoid blocking (though events are discarded on slow consumers).

### WebSocket Broadcast (HTTP Adapter)

For `gert serve --http`, events are broadcast to WebSocket clients. The HTTP adapter subscribes to `RunHandle.Events()` and forwards to all connected WebSocket clients for a given `run_id`.

**Wire format:** JSON-encoded event envelopes, one per WebSocket message frame.

**Delivery semantics:** Best-effort. If a WebSocket client is slow, the adapter MAY drop events. Clients SHOULD use `GET /runs/:id/trace` to fetch the authoritative trace if gaps are detected.

### JSONL Trace File

The append-only JSONL trace file is the durable, authoritative record of all events with guaranteed at-least-once delivery.

Every event is written synchronously to `.runbook/runs/<run-id>/trace.jsonl` before the runtime proceeds. File writes use `O_APPEND` mode for Unix atomicity. One JSON object per line.

---

## Event Subscriptions

```go
type EventSubscriber interface {
    Subscribe(ctx context.Context, opts SubscribeOptions) (<-chan Event, error)
}

type SubscribeOptions struct {
    // KindPrefix filters events to those matching the prefix.
    // Examples: "step/", "governance/", "run/started"
    KindPrefix string

    // RunID filters events to a specific run. If empty, all runs.
    RunID string

    // BufferSize is the channel buffer size. Default: 100.
    BufferSize int
}
```

**Example use cases:**
- TUI adapter subscribes with `KindPrefix: "step/"` to render step progress.
- Governance dashboard subscribes with `KindPrefix: "governance/"` for policy violations.
- Observability forwarder subscribes with no filters to send all events to an OpenTelemetry collector.

---

## Wire Format

Events are serialized as JSON. The wire format is identical for JSONL trace files, WebSocket broadcasts, and JSON-RPC notifications.

### Example: `step/started`

```json
{
  "event_id":    "a3f9c1e2-8d4b-4e1f-9a2c-3d5e6f7a8b9c",
  "run_id":      "20260418T143022-a3f9c1",
  "runbook_id":  "deploy-frontend",
  "timestamp":   "2026-04-18T14:30:25.123456Z",
  "kind":        "step/started",
  "sequence":    3,
  "payload": {
    "step_id":   "build_assets",
    "step_type": "cli",
    "index":     2
  }
}
```

### Example: `governance/approval_requested`

```json
{
  "event_id":    "b1c2d3e4-5f6a-7b8c-9d0e-1f2a3b4c5d6e",
  "run_id":      "20260418T143022-a3f9c1",
  "runbook_id":  "deploy-frontend",
  "timestamp":   "2026-04-18T14:32:10.456789Z",
  "kind":        "governance/approval_requested",
  "sequence":    12,
  "payload": {
    "step_id":         "deploy_to_prod",
    "approver_roles":  ["SRE", "PM"],
    "min_approvals":   2,
    "timeout_seconds": 3600,
    "message":         "Deploying to production. Confirm change window."
  }
}
```

---

## Required vs. Optional Events

| Event Kind | Status | Notes |
|---|---|---|
| `run/started` | REQUIRED | First event in every trace |
| `run/completed` | REQUIRED | Terminal event |
| `run/cancelled` | REQUIRED | If cancelled |
| `step/started` | REQUIRED | For every executed step |
| `step/completed` | REQUIRED | For every successful step |
| `step/failed` | REQUIRED | For every failed step |
| `step/skipped` | REQUIRED | For every skipped step |
| `step/retrying` | OPTIONAL | v2.1+ retry feature |
| `governance/command_checked` | OPTIONAL | Useful for audit trails |
| `governance/approval_requested` | REQUIRED | If approval gates used |
| `governance/approval_received` | REQUIRED | If approval gates used |
| `governance/redaction_applied` | OPTIONAL | Compliance logging |
| `extension/loaded` | OPTIONAL | Debugging |
| `extension/unloaded` | OPTIONAL | Debugging |
| `tool/invoked` | OPTIONAL | Observability |
| `tool/completed` | OPTIONAL | Observability |
| `input/prompted` | REQUIRED | For manual steps |
| `input/received` | REQUIRED | For manual steps |
| `saga/compensation_triggered` | REQUIRED | v2.1+ if saga used |
| `saga/compensation_completed` | REQUIRED | v2.1+ if saga used |

Adapters MUST handle all REQUIRED events. They SHOULD gracefully ignore unknown event kinds to support forward compatibility.

---

## Determinism and Replay Semantics

### Replay Mode

In replay mode, the Runtime Core reads events from a historical trace file and re-emits them in sequence order. No subprocesses are forked, no tools are invoked, no governance checks are evaluated.

**Replay guarantees:**
- Every event is re-emitted with the same `sequence`, `timestamp`, `kind`, and `payload`.
- New `event_id` values are assigned (event IDs are not stable across replays).
- Step execution does not occur.

### Non-determinism Boundaries

The following are NOT captured in events and break full determinism:
- System time (`timestamp` fields use wall-clock time, not logical time)
- External network calls (DNS, HTTP, database)
- File system state outside `.runbook/`
- Random UUIDs (`event_id` and `correlation_id` are regenerated in replay)

For full deterministic replay (testing/debugging), use dry-run mode or snapshot the external environment.
