# Evidence, Tracing, and Resumption

Gert v2 captures a complete, durable record of every execution for audit, compliance, incident response, and operational replay. This section specifies the append-only JSONL trace format, evidence capture model, per-step snapshot contract, run resumption semantics, replay mode, and OpenTelemetry integration. Every run produces a self-contained artifact that answers: *what happened*, *what evidence was collected*, and *how can execution be recovered or replicated*.

---

## Trace File Format

### File Location and Naming

Each run produces a single trace file at:

```
.runbook/runs/<run-id>/trace.jsonl
```

The `<run-id>` is a URL-safe string assigned at run creation (e.g. `20260418T143022-a3f9c1`). All run artefacts are co-located:

```
.runbook/runs/<run-id>/
  trace.jsonl          # append-only JSONL event log
  snapshots/           # per-step state checkpoints
    step-0000.json
    step-0001.json
    ...
  attachments/         # binary evidence referenced from trace events
    <sha256>.bin
    ...
  run.yaml             # completion manifest (written on terminal event)
  annotations.jsonl    # operator notes (append-only)
  lock                 # PID lock (removed on clean exit)
```

### JSONL Envelope

Every line in `trace.jsonl` is a self-contained JSON object. Consumers MUST tolerate unknown fields.

```
{
  "event_id":  <string>,    // UUID v4 for correlation
  "sequence":  <int64>,     // monotonically increasing within this run
  "kind":      <string>,    // event kind
  "timestamp": <RFC3339>,   // UTC timestamp with sub-second precision
  "run_id":    <string>,    // run identifier
  "path":      <array>,     // tree-path to the active node (may be null)
  "payload":   <object>     // kind-specific payload
}
```

`sequence` provides total order over events in a single run; it is not trusted for cross-run ordering. `event_id` is a UUID v4 assigned at emission time for correlation with external systems.

### Event Type Catalogue

All user-supplied string fields (command arguments, outputs, prompt responses) pass through the redaction pipeline before being written.

#### `run/started`

Emitted once, as the first event in every trace file.

```
"payload": {
  "runbook_id":    <string>,  // runbook name/path
  "user":          <string>,  // actor identity
  "mode":          <string>,  // "real" | "replay" | "dry-run"
  "inputs":        <object>,  // resolved input values (redacted)
  "client":        <string>,  // "cli" | "tui" | "web" | "mcp"
  "parent_run_id": <string>   // omitted if not an invoke child
}
```

#### `step/started`

Emitted immediately before executing a step.

```
"payload": {
  "step_id":   <string>,  // stable step identifier
  "step_type": <string>,  // "cli" | "manual" | "tool" | "invoke"
  "step_name": <string>   // human-readable name from runbook YAML
}
```

#### `step/completed`

Emitted after a step exits cleanly.

```
"payload": {
  "step_id":    <string>,
  "exit_status": <int>,      // exit code for cli steps; 0 for others
  "duration_ms": <int64>,
  "captures":   <object>,    // captured output variables (redacted)
  "evidence":   [
    {
      "name":   <string>,
      "kind":   "text" | "checklist" | "attachment",
      "value":  <string>,    // for kind=text
      "items":  <object>,    // for kind=checklist
      "path":   <string>,    // relative path for kind=attachment
      "sha256": <string>,    // hex SHA256 for kind=attachment
      "size":   <int64>      // bytes for kind=attachment
    }
  ]
}
```

#### `step/failed`

Emitted when a step exits with non-zero status or returns a runtime error.

```json
"data": {
  "step_id":    <string>,
  "error_type": <string>,  // "exit_code" | "timeout" | "governance"
                           // | "tool_error" | "invoke_failed"
  "error":      <string>,  // human-readable message (redacted)
  "exit_status": <int>,    // for error_type=exit_code
  "duration_ms": <int64>
}
```

#### `step/skipped`

Emitted when a step is not executed due to branch evaluation or iterate exhaustion.

```json
"data": {
  "step_id": <string>,
  "reason":  "branch_not_taken" | "iterate_exhausted" | "dry_run"
}
```

#### `step/retrying`

Emitted before each retry attempt.

```json
"data": {
  "step_id":       <string>,
  "attempt":       <int>,     // attempt number (1 = first retry)
  "max_attempts":  <int>,
  "backoff_ms":    <int64>
}
```

#### `tool/invoked`

Emitted when the tool runtime dispatches a tool call.

```json
"data": {
  "tool_name":      <string>,
  "transport":      "stdio" | "jsonrpc" | "mcp",
  "correlation_id": <string>,  // UUIDv4, links to tool/responded
  "inputs":         <object>   // sanitized inputs
}
```

#### `tool/responded`

Emitted when the tool call returns.

```json
"data": {
  "correlation_id": <string>,
  "exit_code":      <int>,
  "duration_ms":    <int64>,
  "truncated":      <bool>     // true if output was capped
}
```

#### `manual/prompted`

Emitted when a manual step displays its prompt.

```json
"data": {
  "step_id":     <string>,
  "prompt_text": <string>,   // the prompt shown (redacted)
  "sla_seconds": <int>       // 0 if no SLA configured
}
```

#### `manual/completed`

Emitted when the operator submits a response.

```json
"data": {
  "step_id":      <string>,
  "response":     <string>,   // operator's text response (redacted)
  "attestation":  <string>,   // "operator_confirmed" | "skipped"
  "approver":     <string>,   // identity of responding actor
  "duration_ms":  <int64>
}
```

#### `governance/blocked`

Emitted when a governance rule blocks execution.

```json
"data": {
  "step_id":          <string>,
  "rule":             <string>,  // allowlist | denylist | env_block
                                 // | output_redact | approval_gate
  "action_attempted": <string>,
  "policy":           <string>
}
```

#### `governance/approved`

Emitted when an approval gate is cleared.

```json
"data": {
  "step_id":         <string>,
  "approver":        <string>,
  "approval_method": <string>,  // "interactive" | "mcp" | "api"
  "duration_ms":     <int64>
}
```

#### `run/completed`

Terminal event for a successful run.

```json
"data": {
  "final_status": "passed" | "failed" | "skipped",
  "duration_ms":  <int64>,
  "step_count":   <int>,
  "passed":       <int>,
  "failed":       <int>,
  "skipped":      <int>
}
```

#### `run/failed`

Terminal event for a run that halted due to an unrecoverable error.

```json
"data": {
  "failing_step_id": <string>,
  "error":           <string>,
  "duration_ms":     <int64>
}
```

#### `checkpoint`

Internal event written by `RunStore.SaveCheckpoint` to link a snapshot file to the trace sequence. Required for resumption.

```json
"data": {
  "snapshot_file": <string>  // e.g. "step-0003.json"
}
```

### Crash Safety

The trace file is opened with `O_APPEND`; each event is written as a single `write(2)` call followed by `fsync`. On POSIX, a write under `PIPE_BUF` (4096 bytes on Linux) to an `O_APPEND` file is atomic. Large payloads are truncated at a configurable limit (default 4 KB per field) before encoding.

```go
// WriteTrace appends one event atomically and fsyncs.
func (d *DirRunStore) WriteTrace(event DurableEvent) error {
    p := filepath.Join(d.baseDir, "trace.jsonl")
    f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
        return fmt.Errorf("open trace: %w", err)
    }
    defer f.Close()

    data, err := json.Marshal(event)
    if err != nil {
        return fmt.Errorf("marshal event: %w", err)
    }
    data = append(data, '\n') // JSONL delimiter
    if _, err := f.Write(data); err != nil {
        return fmt.Errorf("write trace: %w", err)
    }
    return f.Sync()
}
```

On recovery, any line that fails `json.Unmarshal` is silently skipped. The last clean `checkpoint` event determines the recoverable state.

### Tamper Evidence

When a signing key is configured (via `GERT_TRACE_KEY` or workspace config), the runtime appends a `"sig"` field to the envelope containing the HMAC-SHA256 of the canonical JSON of all other fields. This detects modification of existing events (but not deletion — that is provided by the append-only filesystem discipline).

---

## Evidence Capture

### What Constitutes Evidence

- **Command output** — stdout/stderr of a `cli` step, embedded in `step/completed`.
- **Manual attestations** — operator-provided text, checklist responses, and explicit confirmations from `manual` steps.
- **Tool responses** — structured JSON from a `tool` step, captured in `step/completed.captures`.
- **File attachments** — binary or text files attached by an operator or produced by a step, stored in `attachments/` and referenced by SHA256 digest.

### SHA256 Hashing

All file attachments are content-addressed by SHA256 digest.

```go
// HashFile returns hex SHA256 and file size.
func HashFile(path string) (sha256hex string, size int64, err error)

// NewAttachmentEvidence creates an EvidenceValue with hash and size.
func NewAttachmentEvidence(path string) (*EvidenceValue, error)
```

The hash is stored in `step/completed` alongside the relative path under `attachments/`. Consumers can verify integrity by re-hashing the file.

### File Attachments

When a step produces a binary artefact, the runtime:
1. Copies the file to `.runbook/runs/<run-id>/attachments/<sha256>.<ext>`.
2. Emits an `EvidenceValue` of kind `attachment` with fields `path`, `sha256`, and `size`.

Attachments are **deduplicated by content**: identical files share one attachment entry; trace events may reference the same `sha256` independently.

### Evidence Retention

Configured in workspace `gert.yaml`:

- **`retention.runs`** — number of completed runs to retain (default: 50).
- **`retention.max_age_days`** — maximum age in days (default: 90).
- **`retention.attachments`** — include attachments in retention (default: true).

`gert gc` enforces retention by removing oldest run directories (deletes `attachments/` first).

---

## Run State Snapshots

### Checkpoint Model

A checkpoint is a complete serialisation of `RunState` written to `snapshots/step-NNNN.json` after every successfully completed step.

```go
type RunState struct {
    RunID             string
    RunbookPath       string
    Mode              string            // "real" | "replay" | "dry-run"
    StartedAt         time.Time
    Actor             string
    CurrentStepIndex  int               // flat-mode index (legacy compat)
    Path              TreePath          // tree-mode cursor (authoritative)
    Vars              map[string]string // resolved input bindings
    Captures          map[string]string // accumulated step outputs
    History           []*StepResult     // completed step records
    InvokeStack       []InvokeFrame     // nested invoke context stack
    IterateState      map[string]*IterateProgress
    ParallelSlots     map[string][]SlotState
    LastCheckpointSeq int64             // trace seq at checkpoint time
}
```

### Atomic Snapshot Write

Snapshots are written atomically: marshal to `.tmp` file, fsync, then rename to final path. An incomplete snapshot retains the `.tmp` suffix and is skipped during recovery.

```go
func (d *DirRunStore) SaveCheckpoint(state *RunState) (string, error) {
    filename := fmt.Sprintf("step-%04d.json", len(state.History))
    finalPath := filepath.Join(d.baseDir, "snapshots", filename)
    tmpPath   := finalPath + ".tmp"

    data, _  := json.MarshalIndent(state, "", "  ")
    os.WriteFile(tmpPath, data, 0644)
    tf, _ := os.Open(tmpPath); tf.Sync(); tf.Close()
    os.Rename(tmpPath, finalPath)              // atomic rename
    // ... then write checkpoint trace event
}
```

### Minimum Resumption State

The state required to resume from step N is exactly `step-NNNN.json`. It contains:
- All resolved input values (`Vars`)
- Outputs captured by completed steps (`Captures`)
- The tree cursor (`Path`) identifying which node executes next
- Branch decisions already taken (encoded in `History`)
- Iterate progress counters and parallel slot states
- The invoke stack for nested runbooks

The runbook YAML itself is **not** stored in the snapshot; it is re-read from `RunbookPath` at resume time. Operators must not modify the runbook between the original run and resumption.

---

## Run Resumption

### `gert exec --resume` Contract

```bash
gert exec --resume <run-id>
```

Recovery sequence:

1. Locate `.runbook/runs/<run-id>/`.
2. Attempt to acquire the `lock` file. Fail if a live process holds it.
3. Call `RunStore.LoadLatestCheckpoint()` — scan `snapshots/` descending, skip `.tmp` files, return newest valid JSON.
4. Re-open `trace.jsonl` for append.
5. Advance the tree `Path` cursor past the last completed node.
6. Re-build the governance engine from the runbook's policy block.
7. Resume the execution loop from the restored `Path`.

Resumption does **not** re-execute any step recorded in `History`.

### Resumability by Step Type

- **`cli` steps** — safe to resume; re-executed from scratch. Authors should ensure idempotency (e.g. `kubectl apply`).
- **`manual` steps** — operator is re-prompted; prior response lost; prompt re-displayed and operator must re-attest.
- **`tool` steps** — resumable if idempotent. If `tool/invoked` exists with no `tool/responded`, call is re-issued if `idempotent: true`; otherwise treated as **failed**.
- **`invoke` steps** — if child run completed (terminal event in parent trace), treated as completed. If child was in progress, child is itself resumed via nested run ID in `InvokeStack`.
- **`branch`/`iterate`** — structural nodes; cursor advancement handles re-entry correctly.

### Partial Step Resumption

If a `tool/invoked` event has no matching `tool/responded`:

1. Runtime logs a warning identifying the orphaned correlation ID.
2. If `idempotent: true`: tool call is re-issued from the beginning.
3. If `idempotent: false` (default): step is treated as **failed**; execution follows normal failure path.

### Idempotency Guidance

- Use `kubectl apply` and `terraform apply` instead of imperative create commands.
- Use `--if-not-exists` or `--create-or-replace` flags where available.
- Declare `idempotent: true` in the tool definition when safe.
- Write manual prompts as questions valid on a second ask.
- Avoid steps whose side effects depend on the current time or a random seed without an explicit override.

---

## Replay Mode

```bash
gert exec --mode replay --scenario <scenario.yaml>
```

No real commands are executed and no real tools are invoked. Used for testing, demonstration, and regression.

### Scenario File Format

```yaml
commands:
  - argv: ["kubectl", "get", "pods", "-n", "prod"]
    stdout: "NAME          READY   STATUS    RESTARTS\npod-abc   1/1     Running   0\n"
    stderr: ""
    exit_code: 0

evidence:
  check-pod-health:          # step_id
    observation:             # evidence name
      kind: text
      value: "pod is healthy, proceeding"
```

Commands matched in declaration order (first match wins). Unmatched commands fail the scenario unless `allow_unmatched: true`.

### Replay Semantics

- `cli` steps — returns pre-recorded stdout/stderr/exit_code.
- `manual` steps — returns pre-recorded evidence values; no operator prompt.
- `tool` steps — returns matching recorded response; step fails if no match.
- `invoke` steps — child runbook executed in replay mode using same (or nested) scenario.
- Governance rules — evaluated **normally**; replay does not bypass governance.
- Trace events — written normally; replay trace can be compared against a golden trace.

### Determinism Requirement

Non-determinism sources to avoid: timestamps in expressions, random identifiers in step names, file system state not reproduced by the scenario, external API calls in tool steps not covered by scenario fixtures.

---

## OpenTelemetry Integration

OTel spans are optimised for real-time observability; the JSONL trace for durability and offline analysis. Both run simultaneously.

### Span Hierarchy

```yaml
Span: gert.run                    # root span, covers entire run
  Span: gert.step                 # one per step executed
    Span: gert.tool.invoke        # one per tool call within a step
  Span: gert.step                 # next step
  ...
```

```go
func (e *Engine) runWithOTel(ctx context.Context) error {
    ctx, runSpan := otel.Tracer("gert").Start(ctx, "gert.run",
        trace.WithAttributes(
            attribute.String("gert.run_id",    e.State.RunID),
            attribute.String("gert.runbook",   e.Runbook.Meta.Name),
            attribute.String("gert.actor",     e.State.Actor),
            attribute.String("gert.mode",      e.State.Mode),
        ),
    )
    defer runSpan.End()
    return e.execute(ctx)
}

func (e *Engine) executeStep(ctx context.Context, step Step) error {
    ctx, stepSpan := otel.Tracer("gert").Start(ctx, "gert.step",
        trace.WithAttributes(
            attribute.String("gert.step_id",   step.ID),
            attribute.String("gert.step_type", step.Type),
            attribute.String("gert.step_name", step.Name),
        ),
    )
    defer stepSpan.End()
    // ...
}
```

### Span Attributes

| Attribute            | Spans       | Description                    |
|----------------------|-------------|--------------------------------|
| `gert.run_id`        | all         | Run identifier                 |
| `gert.runbook`       | all         | Runbook name                   |
| `gert.actor`         | run         | Actor identity                 |
| `gert.mode`          | run         | Execution mode                 |
| `gert.step_id`       | step, tool  | Step identifier                |
| `gert.step_type`     | step        | Step type                      |
| `gert.tool_name`     | tool        | Tool name                      |
| `gert.transport`     | tool        | Transport (stdio/jsonrpc/mcp)  |

Trace context propagated into tool invocations via `traceparent` header (HTTP MCP) or `OTEL_TRACEPARENT` env var (stdio tools). The trace event `"seq"` is embedded as `gert.trace_seq` span attribute for correlation.

### Configuration

OTel is opt-in, configured via standard OTEL environment variables:

```
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
OTEL_SERVICE_NAME=gert
OTEL_TRACES_SAMPLER=always_on
```

When `OTEL_EXPORTER_OTLP_ENDPOINT` is not set, the OTel SDK is initialised with a no-op tracer (zero overhead). No `--otel` flag exists; configuration is environment-only.

---

## Go Interface Reference

```go
// TraceWriter is the interface the engine uses to record events.
// Safe for concurrent use from a single goroutine (the engine's execution loop).
type TraceWriter interface {
    Write(event DurableEvent) error
    Close() error
}

// RunStore manages all artefacts for a single run.
type RunStore interface {
    RunID() string
    WriteTrace(event DurableEvent) error
    ReadTrace() ([]DurableEvent, error)
    ReadTraceSince(afterSeq int64) ([]DurableEvent, error)
    SaveCheckpoint(state *RunState) (string, error)
    LoadLatestCheckpoint() (*RunState, int64, error)
    AcquireLock() error
    ReleaseLock() error
    WriteManifest(manifest *RunManifest) error
}

// EvidenceCollector gathers evidence at manual steps.
type EvidenceCollector interface {
    Collect(ctx context.Context, stepID string,
        prompts []Prompt) (map[string]*EvidenceValue, error)
}
```

- **`DirRunStore`** — production implementation (filesystem).
- **`MemRunStore`** — in-memory implementation for unit tests.
- **`ReplayEvidenceCollector`** — returns pre-recorded evidence for scenario tests.
