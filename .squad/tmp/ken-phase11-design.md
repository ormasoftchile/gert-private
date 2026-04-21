# Phase 11 — Evidence & Replay

**Author:** Ken (Software Architect)  
**Date:** 2026-04-24  
**Requested by:** Cristian  
**Depends on:** Phase 10 (Adapters, sealed at commit c570a9c)

---

## 1 Overview

Phase 11 delivers four capabilities that transform gert's trace file from a passive log into an active operational tool:

| # | Capability | Summary |
|---|-----------|---------|
| 1 | **Evidence collection** | Structured evidence capture at each step — stdout, stderr, file attachments, operator attestations — embedded in trace events via a non-intrusive hook |
| 2 | **Trace replay** | Deterministic re-execution of a past run from its JSONL trace or scenario file, suppressing real side effects |
| 3 | **Run resumption** | Crash recovery: scan trace → locate last checkpoint → reconstruct state → continue execution from next step |
| 4 | **Evidence querying** | Read/query evidence from a completed or in-progress run via `TraceReader` and `run.evidence` RPC |

After Phase 11, every run produces a self-contained artifact that answers: *what happened*, *what evidence was collected*, and *how can execution be recovered or replicated*.

### What does NOT ship in Phase 11

- OpenTelemetry span integration (deferred to Phase 12+; the spec defines it but the dependency is optional)
- `gert gc` retention enforcement (future CLI command)
- Cross-run evidence aggregation / dashboards

---

## 2 Package Layout

### New files

```
v2/
├── pkg/
│   ├── evidence/
│   │   ├── doc.go               # Package docs — leaf package, no internal imports
│   │   ├── evidence.go          # EvidenceRecord, EvidenceKind, EvidenceValue
│   │   ├── hash.go              # HashFile(), HashBytes() — SHA256 utilities
│   │   └── evidence_test.go     # Unit tests for evidence model + hashing
│   └── trace/
│       ├── reader.go            # TraceReader interface (new)
│       └── filter.go            # TraceFilter predicate type (new)
├── internal/
│   ├── evidence/
│   │   ├── doc.go               # Package docs
│   │   ├── collector.go         # DefaultCollector — captures evidence from executor output
│   │   ├── collector_test.go    # Collector tests
│   │   ├── hook.go              # EvidenceHook — non-intrusive post-step hook
│   │   ├── hook_test.go         # Hook tests
│   │   ├── attachment.go        # Attachment storage (copy + SHA256 dedup)
│   │   └── attachment_test.go   # Attachment tests
│   ├── replay/
│   │   ├── doc.go               # Package docs
│   │   ├── engine.go            # ReplayEngine — drives replay from trace/scenario
│   │   ├── engine_test.go       # Replay engine tests
│   │   ├── scenario.go          # Scenario file loader (YAML)
│   │   ├── scenario_test.go     # Scenario tests
│   │   ├── executor.go          # ReplayExecutor — wraps real executor, suppresses side effects
│   │   └── executor_test.go     # ReplayExecutor tests
│   ├── resume/
│   │   ├── doc.go               # Package docs
│   │   ├── resume.go            # ResumeFromTrace() — scan → checkpoint → rebuild state
│   │   ├── resume_test.go       # Resume tests
│   │   ├── scanner.go           # TraceScanner — reads JSONL line by line, finds checkpoints
│   │   └── scanner_test.go      # Scanner tests
│   ├── trace/
│   │   ├── jsonl_reader.go      # JSONLReader — TraceReader implementation (new)
│   │   └── jsonl_reader_test.go # Reader tests (new)
│   └── serve/
│       └── rpc.go               # MODIFIED: add run.evidence, run.resume methods
├── cmd/
│   └── gert/
│       └── main.go              # MODIFIED: add --resume flag to exec subcommand
```

### Modified files

| File | Change |
|------|--------|
| `internal/engine/engine.go` | Implement `Resume()` (currently returns `ErrNotImplemented`); add evidence hook call site in `executeStep` |
| `internal/serve/rpc.go` | Add `run.evidence` and `run.resume` RPC methods to switch dispatch |
| `internal/adapter/wire.go` | Wire `EvidenceHook` into `EngineConfig`; wire `RunStore` for resume |
| `internal/adapter/options.go` | Add `RunDir` field to `WireOptions` |
| `cmd/gert/main.go` | Add `--resume <run-id>` flag |
| `pkg/engine/engine.go` | Add `EvidenceHook` field to `EngineConfig` (optional) |
| `pkg/engine/run.go` | Add `Evidence` field to `StepResult` |

### No changes to

| Package | Reason |
|---------|--------|
| `pkg/trace/writer.go` | TraceWriter interface is stable |
| `pkg/trace/event.go` | EventKind constants are stable (checkpoint event uses existing infra) |
| `internal/trace/jsonl_writer.go` | Writer is unchanged |
| `internal/trace/multi_writer.go` | MultiWriter is unchanged |

---

## 3 Evidence Model

### 3.1 `pkg/evidence` — Leaf Package

`pkg/evidence` is a **leaf package** with zero imports from `internal/*` or other `pkg/*` packages. It depends only on the Go standard library.

```go
package evidence

import "time"

// EvidenceKind is the type discriminator for evidence items.
type EvidenceKind string

const (
    EvidenceKindText       EvidenceKind = "text"
    EvidenceKindChecklist  EvidenceKind = "checklist"
    EvidenceKindAttachment EvidenceKind = "attachment"
    EvidenceKindSnapshot   EvidenceKind = "snapshot"
)

// EvidenceRecord is a single piece of evidence captured at a step.
// Embedded in the step/completed trace event payload.
type EvidenceRecord struct {
    // Name is the evidence field name (e.g., "stdout", "observation", "screenshot").
    Name string `json:"name"`

    // Kind discriminates the evidence type.
    Kind EvidenceKind `json:"kind"`

    // Value holds text content (for Kind=text).
    Value string `json:"value,omitempty"`

    // Items holds checklist responses (for Kind=checklist).
    // Keys are checklist item labels; values are "checked" or "unchecked".
    Items map[string]string `json:"items,omitempty"`

    // Path is the relative path under attachments/ (for Kind=attachment).
    Path string `json:"path,omitempty"`

    // SHA256 is the hex-encoded SHA256 digest (for Kind=attachment).
    SHA256 string `json:"sha256,omitempty"`

    // Size is the file size in bytes (for Kind=attachment).
    Size int64 `json:"size,omitempty"`

    // CapturedAt is when this evidence was collected.
    CapturedAt time.Time `json:"captured_at"`
}

// EvidenceSet is an ordered collection of evidence records for a single step.
type EvidenceSet struct {
    StepID  string            `json:"step_id"`
    Records []EvidenceRecord  `json:"records"`
}

// ChecklistItem is a single item in a checklist evidence prompt.
type ChecklistItem struct {
    Label    string `json:"label"`
    Required bool   `json:"required"`
}
```

### 3.2 SHA256 Hashing Utilities

```go
package evidence

import (
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"
    "os"
)

// HashFile returns the hex-encoded SHA256 digest and file size.
func HashFile(path string) (sha256hex string, size int64, err error) {
    f, err := os.Open(path)
    if err != nil {
        return "", 0, fmt.Errorf("evidence: open %s: %w", path, err)
    }
    defer f.Close()

    h := sha256.New()
    n, err := io.Copy(h, f)
    if err != nil {
        return "", 0, fmt.Errorf("evidence: hash %s: %w", path, err)
    }
    return hex.EncodeToString(h.Sum(nil)), n, nil
}

// HashBytes returns the hex-encoded SHA256 digest of the given bytes.
func HashBytes(data []byte) string {
    h := sha256.Sum256(data)
    return hex.EncodeToString(h[:])
}
```

### 3.3 `StepResult.Evidence` Addition

```go
// In pkg/engine/run.go — add to StepResult:
type StepResult struct {
    // ... existing fields unchanged ...

    // Evidence holds structured evidence collected during this step.
    // nil if no evidence was collected.
    Evidence []evidence.EvidenceRecord `json:"evidence,omitempty"`
}
```

This requires `pkg/engine` to import `pkg/evidence`. Since `pkg/evidence` is a leaf package, this creates no cycle.

---

## 4 Evidence Collection

### 4.1 Design Principle: Non-Intrusive Hooks

Evidence collection MUST NOT modify executor implementations. Instead, the engine invokes an optional `EvidenceHook` after each step completes. Executors opt-in by populating `StepResult.Output` with capturable data; the hook transforms output into structured `EvidenceRecord` values.

### 4.2 `EvidenceHook` Interface

```go
// In pkg/engine/engine.go — add to EngineConfig:
type EngineConfig struct {
    // ... existing fields ...

    // EvidenceHook is called after each step execution to collect evidence.
    // Optional: if nil, no evidence is collected beyond what executors
    // place in StepResult.Output.
    EvidenceHook EvidenceHook
}

// EvidenceHook collects evidence from a completed step.
// Implementations MUST NOT modify the StepResult; they return new evidence records.
type EvidenceHook interface {
    // Collect examines the step result and returns evidence records.
    // The run directory is provided for attachment storage.
    Collect(ctx context.Context, step ResolvedStep, result *StepResult, runDir string) ([]evidence.EvidenceRecord, error)
}
```

### 4.3 `internal/evidence.DefaultCollector`

```go
package evidence

import (
    "context"
    "time"

    evidencepkg "github.com/ormasoftchile/gert/v2/pkg/evidence"
    "github.com/ormasoftchile/gert/v2/pkg/engine"
)

// DefaultCollector implements engine.EvidenceHook.
// It extracts evidence from StepResult.Output following step-kind conventions.
type DefaultCollector struct {
    attachmentStore *AttachmentStore
}

func NewDefaultCollector(runDir string) *DefaultCollector {
    return &DefaultCollector{
        attachmentStore: NewAttachmentStore(runDir),
    }
}

func (c *DefaultCollector) Collect(
    ctx context.Context,
    step engine.ResolvedStep,
    result *engine.StepResult,
    runDir string,
) ([]evidencepkg.EvidenceRecord, error) {
    if result == nil || result.Output == nil {
        return nil, nil
    }

    var records []evidencepkg.EvidenceRecord
    now := time.Now()

    // CLI steps: capture stdout/stderr as text evidence.
    if stdout, ok := result.Output["stdout"].(string); ok && stdout != "" {
        records = append(records, evidencepkg.EvidenceRecord{
            Name:       "stdout",
            Kind:       evidencepkg.EvidenceKindText,
            Value:      stdout,
            CapturedAt: now,
        })
    }
    if stderr, ok := result.Output["stderr"].(string); ok && stderr != "" {
        records = append(records, evidencepkg.EvidenceRecord{
            Name:       "stderr",
            Kind:       evidencepkg.EvidenceKindText,
            Value:      stderr,
            CapturedAt: now,
        })
    }

    // Tool steps: capture response JSON as text evidence.
    if response, ok := result.Output["response"].(string); ok && response != "" {
        records = append(records, evidencepkg.EvidenceRecord{
            Name:       "tool_response",
            Kind:       evidencepkg.EvidenceKindText,
            Value:      response,
            CapturedAt: now,
        })
    }

    // File attachments: steps can place paths in output["attachments"].
    if attachments, ok := result.Output["attachments"].([]any); ok {
        for _, att := range attachments {
            path, ok := att.(string)
            if !ok {
                continue
            }
            rec, err := c.attachmentStore.Store(path)
            if err != nil {
                continue // best-effort: log but don't fail the step
            }
            rec.CapturedAt = now
            records = append(records, *rec)
        }
    }

    // Checklist evidence from manual/collector steps.
    if checklist, ok := result.Output["checklist"].(map[string]any); ok {
        items := make(map[string]string, len(checklist))
        for k, v := range checklist {
            if s, ok := v.(string); ok {
                items[k] = s
            }
        }
        records = append(records, evidencepkg.EvidenceRecord{
            Name:       "checklist",
            Kind:       evidencepkg.EvidenceKindChecklist,
            Items:      items,
            CapturedAt: now,
        })
    }

    return records, nil
}
```

### 4.4 Attachment Storage

```go
package evidence

import (
    "fmt"
    "io"
    "os"
    "path/filepath"

    evidencepkg "github.com/ormasoftchile/gert/v2/pkg/evidence"
)

// AttachmentStore copies files into the run's attachments/ directory,
// deduplicating by SHA256 content hash.
type AttachmentStore struct {
    dir string // .runbook/runs/<run-id>/attachments/
}

func NewAttachmentStore(runDir string) *AttachmentStore {
    dir := filepath.Join(runDir, "attachments")
    _ = os.MkdirAll(dir, 0o755)
    return &AttachmentStore{dir: dir}
}

// Store copies a file into the attachment directory and returns an EvidenceRecord.
// Files with identical content (same SHA256) share one copy.
func (s *AttachmentStore) Store(sourcePath string) (*evidencepkg.EvidenceRecord, error) {
    sha, size, err := evidencepkg.HashFile(sourcePath)
    if err != nil {
        return nil, err
    }

    ext := filepath.Ext(sourcePath)
    destName := sha + ext
    destPath := filepath.Join(s.dir, destName)

    // Deduplicate: skip copy if content-addressed file exists.
    if _, err := os.Stat(destPath); err == nil {
        return &evidencepkg.EvidenceRecord{
            Name:   filepath.Base(sourcePath),
            Kind:   evidencepkg.EvidenceKindAttachment,
            Path:   filepath.Join("attachments", destName),
            SHA256: sha,
            Size:   size,
        }, nil
    }

    // Copy file.
    src, err := os.Open(sourcePath)
    if err != nil {
        return nil, fmt.Errorf("evidence: open source %s: %w", sourcePath, err)
    }
    defer src.Close()

    dst, err := os.Create(destPath)
    if err != nil {
        return nil, fmt.Errorf("evidence: create attachment %s: %w", destPath, err)
    }
    if _, err := io.Copy(dst, src); err != nil {
        dst.Close()
        return nil, err
    }
    if err := dst.Sync(); err != nil {
        dst.Close()
        return nil, err
    }
    dst.Close()

    return &evidencepkg.EvidenceRecord{
        Name:   filepath.Base(sourcePath),
        Kind:   evidencepkg.EvidenceKindAttachment,
        Path:   filepath.Join("attachments", destName),
        SHA256: sha,
        Size:   size,
    }, nil
}
```

### 4.5 Integration Point: Engine `executeStep`

In `internal/engine/engine.go`, after the executor returns and redaction is applied, the evidence hook is invoked:

```go
// After redaction and before emitting step/completed, in executeStep():

// ── EVIDENCE COLLECTION (Phase 11) ──────────────────────────────
if h.engine.cfg.EvidenceHook != nil && result.Status == enginepkg.StepStatusCompleted {
    runDir := h.runDir() // .runbook/runs/<run-id>/
    records, evErr := h.engine.cfg.EvidenceHook.Collect(ctx, step, result, runDir)
    if evErr == nil && len(records) > 0 {
        result.Evidence = records
    }
}
// ── END EVIDENCE COLLECTION ─────────────────────────────────────
```

The evidence records are then embedded in the `step/completed` trace event payload (see §5).

---

## 5 JSONL Trace Format

### 5.1 Envelope (unchanged from Phase 10)

```json
{
  "seq":        <int64>,
  "ts":         "<RFC3339Nano>",
  "kind":       "<string>",
  "run_id":     "<string>",
  "runbook_id": "<string>",
  "event_id":   "<string>",
  "payload":    <object>,
  "sig":        "<string>"    // optional HMAC-SHA256
}
```

### 5.2 Complete Event Catalogue with Evidence Payloads

#### `run/started`
```json
{
  "kind": "run/started",
  "payload": {
    "run_id":       "<string>",
    "runbook_path": "<string>",
    "actor":        "<string>",
    "mode":         "real" | "replay" | "dry-run",
    "inputs":       { "<key>": "<value>" },
    "client":       "cli" | "serve" | "mcp"
  }
}
```

#### `step/started`
```json
{
  "kind": "step/started",
  "payload": {
    "step_id":   "<string>",
    "kind":      "<string>",
    "step_name": "<string>"
  }
}
```

#### `step/completed` (with evidence)
```json
{
  "kind": "step/completed",
  "payload": {
    "step_id":     "<string>",
    "exit_status":  0,
    "duration_ms": 1234,
    "output":      { "<key>": "<value>" },
    "captures":    { "<key>": "<value>" },
    "evidence": [
      {
        "name":       "stdout",
        "kind":       "text",
        "value":      "pod-abc  1/1  Running  0\n",
        "captured_at": "2026-04-24T10:30:00.000000Z"
      },
      {
        "name":       "screenshot.png",
        "kind":       "attachment",
        "path":       "attachments/a1b2c3d4...sha256.png",
        "sha256":     "a1b2c3d4e5f6...",
        "size":       84210,
        "captured_at": "2026-04-24T10:30:01.000000Z"
      }
    ]
  }
}
```

#### `step/failed`
```json
{
  "kind": "step/failed",
  "payload": {
    "step_id":     "<string>",
    "error_type":  "exit_code" | "timeout" | "governance" | "tool_error",
    "error":       "<string>",
    "exit_status": 1,
    "duration_ms": 5678
  }
}
```

#### `step/skipped`
```json
{
  "kind": "step/skipped",
  "payload": {
    "step_id": "<string>",
    "reason":  "condition_false" | "branch_not_taken" | "iterate_exhausted" | "dry_run"
  }
}
```

#### `step/retrying`
```json
{
  "kind": "step/retrying",
  "payload": {
    "step_id":      "<string>",
    "attempt":      2,
    "max_attempts": 3,
    "backoff_ms":   1000
  }
}
```

#### `step/resumed`
```json
{
  "kind": "step/resumed",
  "payload": {
    "step_id": "<string>",
    "source":  "wait_for_event" | "resume"
  }
}
```

#### `governance/command_checked`
```json
{
  "kind": "governance/command_checked",
  "payload": {
    "step_id":       "<string>",
    "command":        "<string>",
    "allowed":       true,
    "denied":        false,
    "matched_rules": ["allowlist-k8s"]
  }
}
```

#### `governance/approval_requested`
```json
{
  "kind": "governance/approval_requested",
  "payload": {
    "step_id": "<string>",
    "reason":  "<string>"
  }
}
```

#### `governance/approval_received`
```json
{
  "kind": "governance/approval_received",
  "payload": {
    "step_id":  "<string>",
    "approver": "<string>",
    "method":   "interactive" | "api" | "mcp"
  }
}
```

#### `governance/redaction_applied`
```json
{
  "kind": "governance/redaction_applied",
  "payload": {
    "step_id":    "<string>",
    "rule_count": 3
  }
}
```

#### `tool/invoked`
```json
{
  "kind": "tool/invoked",
  "payload": {
    "tool_name":      "<string>",
    "transport":      "stdio" | "jsonrpc" | "mcp",
    "correlation_id": "<uuid>",
    "step_id":        "<string>"
  }
}
```

#### `tool/completed`
```json
{
  "kind": "tool/completed",
  "payload": {
    "correlation_id": "<uuid>",
    "exit_code":      0,
    "duration_ms":    450,
    "truncated":      false
  }
}
```

#### `input/prompted`
```json
{
  "kind": "input/prompted",
  "payload": {
    "step_id":     "<string>",
    "prompt_text": "<string>",
    "sla_seconds": 0
  }
}
```

#### `input/received`
```json
{
  "kind": "input/received",
  "payload": {
    "step_id":     "<string>",
    "response":    "<string>",
    "attestation": "operator_confirmed" | "skipped",
    "approver":    "<string>"
  }
}
```

#### `event/received`
```json
{
  "kind": "event/received",
  "payload": {
    "step_id": "<string>",
    "event":   { ... }
  }
}
```

#### `checkpoint`
```json
{
  "kind": "checkpoint",
  "payload": {
    "snapshot_file":     "step-0003.json",
    "step_index":        3,
    "last_completed_id": "deploy-pods"
  }
}
```

#### `run/completed`
```json
{
  "kind": "run/completed",
  "payload": {
    "run_id":      "<string>",
    "duration_ms": 45678,
    "final_status": "passed" | "failed",
    "step_count":  10,
    "passed":      8,
    "failed":      1,
    "skipped":     1
  }
}
```

#### `run/cancelled`
```json
{
  "kind": "run/cancelled",
  "payload": {
    "run_id": "<string>",
    "signal": "SIGINT",
    "reason": "<string>"
  }
}
```

### 5.3 Checkpoint Event Insertion

After each successful step, the engine writes a checkpoint event with a corresponding snapshot file. The checkpoint is the glue between the trace sequence and the snapshot directory:

```go
// In internal/engine/engine.go, after recording step result in executeStep():
if h.store != nil && result.Status == enginepkg.StepStatusCompleted {
    state := h.run.Snapshot()
    if err := h.store.SaveState(ctx, state); err == nil {
        h.emitEventLocked(ctx, "checkpoint", map[string]any{
            "snapshot_file":     fmt.Sprintf("step-%04d.json", h.run.CurrentStepIndex),
            "step_index":       h.run.CurrentStepIndex,
            "last_completed_id": step.ID,
        })
    }
}
```

---

## 6 Trace Reader

### 6.1 `TraceReader` Interface

```go
// In pkg/trace/reader.go:
package trace

import "context"

// TraceReader reads events from a JSONL trace file or run store.
// Implementations MUST be safe for concurrent use.
type TraceReader interface {
    // ReadAll returns all events in sequence order.
    ReadAll(ctx context.Context) ([]TraceEvent, error)

    // ReadSince returns events with sequence > afterSeq.
    ReadSince(ctx context.Context, afterSeq int64) ([]TraceEvent, error)

    // ReadFiltered returns events matching the filter predicate.
    ReadFiltered(ctx context.Context, filter TraceFilter) ([]TraceEvent, error)
}
```

### 6.2 `TraceFilter` Type

```go
// In pkg/trace/filter.go:
package trace

// TraceFilter is a predicate for filtering trace events during reads.
type TraceFilter struct {
    // Kinds restricts to these event kinds. Empty means all kinds.
    Kinds []EventKind

    // StepID restricts to events referencing this step ID.
    // Matched against payload["step_id"]. Empty means all steps.
    StepID string

    // AfterSeq returns only events with sequence > this value.
    AfterSeq int64

    // BeforeSeq returns only events with sequence < this value. 0 means no upper bound.
    BeforeSeq int64
}

// Matches returns true if the event passes the filter.
func (f TraceFilter) Matches(event TraceEvent) bool {
    if f.AfterSeq > 0 && event.Sequence <= f.AfterSeq {
        return false
    }
    if f.BeforeSeq > 0 && event.Sequence >= f.BeforeSeq {
        return false
    }
    if len(f.Kinds) > 0 {
        matched := false
        for _, k := range f.Kinds {
            if event.Kind == k {
                matched = true
                break
            }
        }
        if !matched {
            return false
        }
    }
    // StepID filtering requires payload inspection; done in implementation.
    return true
}
```

### 6.3 `internal/trace.JSONLReader`

```go
package trace

import (
    "bufio"
    "context"
    "encoding/json"
    "os"
    "sync"

    tracepkg "github.com/ormasoftchile/gert/v2/pkg/trace"
)

// JSONLReader reads trace events from a JSONL file.
type JSONLReader struct {
    path string
    mu   sync.Mutex
}

// NewJSONLReader constructs a reader for the given trace file path.
func NewJSONLReader(path string) *JSONLReader {
    return &JSONLReader{path: path}
}

// ReadAll returns all events from the trace file, skipping malformed lines.
func (r *JSONLReader) ReadAll(ctx context.Context) ([]tracepkg.TraceEvent, error) {
    return r.ReadFiltered(ctx, tracepkg.TraceFilter{})
}

// ReadSince returns events with sequence > afterSeq.
func (r *JSONLReader) ReadSince(ctx context.Context, afterSeq int64) ([]tracepkg.TraceEvent, error) {
    return r.ReadFiltered(ctx, tracepkg.TraceFilter{AfterSeq: afterSeq})
}

// ReadFiltered reads events matching the given filter.
// Malformed JSON lines are silently skipped (crash safety contract from §12.1.4).
func (r *JSONLReader) ReadFiltered(ctx context.Context, filter tracepkg.TraceFilter) ([]tracepkg.TraceEvent, error) {
    r.mu.Lock()
    defer r.mu.Unlock()

    f, err := os.Open(r.path)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    var events []tracepkg.TraceEvent
    scanner := bufio.NewScanner(f)
    scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 1MB max line

    for scanner.Scan() {
        if ctx.Err() != nil {
            return events, ctx.Err()
        }
        line := scanner.Bytes()
        if len(line) == 0 {
            continue
        }

        var event tracepkg.TraceEvent
        if err := json.Unmarshal(line, &event); err != nil {
            continue // skip malformed lines (crash recovery)
        }

        if !filter.Matches(event) {
            continue
        }

        // StepID filter requires payload inspection.
        if filter.StepID != "" {
            var payload map[string]any
            if err := json.Unmarshal(event.Payload, &payload); err != nil {
                continue
            }
            if stepID, _ := payload["step_id"].(string); stepID != filter.StepID {
                continue
            }
        }

        events = append(events, event)
    }

    return events, scanner.Err()
}
```

---

## 7 Replay Engine

### 7.1 Architecture

Replay mode re-uses the **same engine pipeline** (parse → plan → governance → execute loop) with a `RunModeReplay` flag that swaps real executors for `ReplayExecutor` wrappers. This ensures governance rules, condition evaluation, and event emission all behave identically to a real run.

```
┌─────────────┐    ┌──────────┐    ┌──────────────────┐    ┌─────────────┐
│   Scenario   │───▶│  Replay  │───▶│  ReplayExecutor  │───▶│   Engine    │
│   (.yaml)    │    │  Engine  │    │  (wraps real)     │    │  Pipeline   │
└─────────────┘    └──────────┘    └──────────────────┘    └─────────────┘
                                         │ returns pre-recorded
                                         │ stdout/stderr/response
```

### 7.2 Scenario File Format

```yaml
# scenario.yaml
commands:
  - argv: ["kubectl", "get", "pods", "-n", "prod"]
    stdout: "NAME  READY  STATUS  RESTARTS\npod-abc  1/1  Running  0\n"
    stderr: ""
    exit_code: 0

  - argv: ["terraform", "plan"]
    stdout: "No changes. Infrastructure is up-to-date.\n"
    stderr: ""
    exit_code: 0

evidence:
  check-pod-health:
    observation:
      kind: text
      value: "pod is healthy, proceeding"

tools:
  check-status:
    response: '{"status": "healthy", "uptime": 99.9}'
    exit_code: 0

allow_unmatched: false   # unmatched commands fail (default)
```

### 7.3 `internal/replay.Scenario`

```go
package replay

import (
    "os"

    "gopkg.in/yaml.v3"
)

// Scenario holds pre-recorded responses for replay mode.
type Scenario struct {
    Commands       []CommandFixture          `yaml:"commands"`
    Evidence       map[string]map[string]EvidenceFixture `yaml:"evidence"`
    Tools          map[string]ToolFixture    `yaml:"tools"`
    AllowUnmatched bool                      `yaml:"allow_unmatched"`
}

type CommandFixture struct {
    Argv     []string `yaml:"argv"`
    Stdout   string   `yaml:"stdout"`
    Stderr   string   `yaml:"stderr"`
    ExitCode int      `yaml:"exit_code"`
    matched  bool
}

type EvidenceFixture struct {
    Kind  string            `yaml:"kind"`
    Value string            `yaml:"value,omitempty"`
    Items map[string]string `yaml:"items,omitempty"`
}

type ToolFixture struct {
    Response string `yaml:"response"`
    ExitCode int    `yaml:"exit_code"`
}

// LoadScenario reads and parses a scenario YAML file.
func LoadScenario(path string) (*Scenario, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var s Scenario
    if err := yaml.Unmarshal(data, &s); err != nil {
        return nil, err
    }
    return &s, nil
}

// MatchCommand returns the first unmatched fixture whose argv matches.
// Commands are matched in declaration order (first match wins).
func (s *Scenario) MatchCommand(argv []string) (*CommandFixture, bool) {
    for i := range s.Commands {
        if s.Commands[i].matched {
            continue
        }
        if argvEqual(s.Commands[i].Argv, argv) {
            s.Commands[i].matched = true
            return &s.Commands[i], true
        }
    }
    return nil, false
}
```

### 7.4 `ReplayExecutor`

```go
package replay

import (
    "context"
    "fmt"
    "time"

    "github.com/ormasoftchile/gert/v2/pkg/engine"
)

// ReplayExecutor wraps a real executor and returns pre-recorded responses
// instead of executing real commands. Used when RunMode == "replay".
type ReplayExecutor struct {
    scenario *Scenario
    kind     string
}

func NewReplayExecutor(kind string, scenario *Scenario) *ReplayExecutor {
    return &ReplayExecutor{kind: kind, scenario: scenario}
}

func (e *ReplayExecutor) Execute(ctx context.Context, step engine.ResolvedStep, vars map[string]any) (*engine.StepResult, error) {
    now := time.Now()

    switch e.kind {
    case "cli":
        return e.executeCLI(step, now)
    case "tool":
        return e.executeTool(step, now)
    case "manual", "choice", "decision", "collector":
        return e.executeManual(step, now)
    default:
        // Structural steps (parallel, iterate, branch) pass through.
        return &engine.StepResult{
            StepID:      step.ID,
            Status:      engine.StepStatusCompleted,
            Outcome:     engine.StepOutcomeSuccess,
            StartedAt:   now,
            CompletedAt: now,
        }, nil
    }
}

func (e *ReplayExecutor) executeCLI(step engine.ResolvedStep, now time.Time) (*engine.StepResult, error) {
    // Extract argv from step spec.
    argv := extractArgv(step)
    fixture, ok := e.scenario.MatchCommand(argv)
    if !ok {
        if e.scenario.AllowUnmatched {
            return &engine.StepResult{
                StepID:      step.ID,
                Status:      engine.StepStatusCompleted,
                Outcome:     engine.StepOutcomeSuccess,
                Output:      map[string]any{"stdout": "", "stderr": "", "exit_code": 0},
                StartedAt:   now,
                CompletedAt: now,
            }, nil
        }
        return &engine.StepResult{
            StepID:      step.ID,
            Status:      engine.StepStatusFailed,
            Outcome:     engine.StepOutcomeFailed,
            Error:       fmt.Errorf("replay: no matching command fixture for %v", argv),
            StartedAt:   now,
            CompletedAt: now,
        }, nil
    }

    status := engine.StepStatusCompleted
    outcome := engine.StepOutcomeSuccess
    if fixture.ExitCode != 0 {
        status = engine.StepStatusFailed
        outcome = engine.StepOutcomeFailed
    }

    return &engine.StepResult{
        StepID:  step.ID,
        Status:  status,
        Outcome: outcome,
        Output: map[string]any{
            "stdout":    fixture.Stdout,
            "stderr":    fixture.Stderr,
            "exit_code": fixture.ExitCode,
        },
        StartedAt:   now,
        CompletedAt: now,
    }, nil
}

func (e *ReplayExecutor) executeTool(step engine.ResolvedStep, now time.Time) (*engine.StepResult, error) {
    toolName := extractToolName(step)
    fixture, ok := e.scenario.Tools[toolName]
    if !ok {
        if e.scenario.AllowUnmatched {
            return &engine.StepResult{
                StepID:      step.ID,
                Status:      engine.StepStatusCompleted,
                Outcome:     engine.StepOutcomeSuccess,
                Output:      map[string]any{"response": "{}"},
                StartedAt:   now,
                CompletedAt: now,
            }, nil
        }
        return &engine.StepResult{
            StepID:  step.ID,
            Status:  engine.StepStatusFailed,
            Outcome: engine.StepOutcomeFailed,
            Error:   fmt.Errorf("replay: no matching tool fixture for %s", toolName),
            StartedAt:   now,
            CompletedAt: now,
        }, nil
    }

    return &engine.StepResult{
        StepID:  step.ID,
        Status:  engine.StepStatusCompleted,
        Outcome: engine.StepOutcomeSuccess,
        Output:  map[string]any{"response": fixture.Response, "exit_code": fixture.ExitCode},
        StartedAt:   now,
        CompletedAt: now,
    }, nil
}

func (e *ReplayExecutor) executeManual(step engine.ResolvedStep, now time.Time) (*engine.StepResult, error) {
    stepEvidence, ok := e.scenario.Evidence[step.ID]
    if !ok {
        return &engine.StepResult{
            StepID:      step.ID,
            Status:      engine.StepStatusCompleted,
            Outcome:     engine.StepOutcomeSuccess,
            Output:      map[string]any{"attestation": "replay_auto"},
            StartedAt:   now,
            CompletedAt: now,
        }, nil
    }

    output := make(map[string]any, len(stepEvidence))
    for name, fixture := range stepEvidence {
        switch fixture.Kind {
        case "text":
            output[name] = fixture.Value
        case "checklist":
            output["checklist"] = fixture.Items
        }
    }

    return &engine.StepResult{
        StepID:      step.ID,
        Status:      engine.StepStatusCompleted,
        Outcome:     engine.StepOutcomeSuccess,
        Output:      output,
        StartedAt:   now,
        CompletedAt: now,
    }, nil
}
```

### 7.5 `ReplayExecutorRegistry`

```go
package replay

import "github.com/ormasoftchile/gert/v2/pkg/engine"

// ReplayExecutorRegistry wraps an existing registry, replacing all executors
// with ReplayExecutor instances that use scenario fixtures.
type ReplayExecutorRegistry struct {
    inner    engine.ExecutorRegistry
    scenario *Scenario
}

func NewReplayExecutorRegistry(inner engine.ExecutorRegistry, scenario *Scenario) *ReplayExecutorRegistry {
    return &ReplayExecutorRegistry{inner: inner, scenario: scenario}
}

func (r *ReplayExecutorRegistry) Register(kind string, exec engine.StepExecutor) {
    r.inner.Register(kind, exec)
}

func (r *ReplayExecutorRegistry) Lookup(kind string) engine.StepExecutor {
    if r.inner.Lookup(kind) == nil {
        return nil // don't fabricate executors for unknown kinds
    }
    return NewReplayExecutor(kind, r.scenario)
}
```

### 7.6 Wiring: Adapter Integration

In `internal/adapter/wire.go`, when `Mode == "replay"`:

```go
if opts.Mode == "replay" {
    scenario, err := replay.LoadScenario(opts.ScenarioFile)
    if err != nil {
        return engine.EngineConfig{}, fmt.Errorf("adapter: load scenario: %w", err)
    }
    execRegistry = replay.NewReplayExecutorRegistry(execRegistry, scenario)
}
```

This follows the Phase 10 pattern of `DryRunExecutorRegistry` wrapping.

---

## 8 Run Resumption

### 8.1 `Engine.Resume()` Implementation

The stub in `internal/engine/engine.go` (line 136–138) currently returns `ErrNotImplemented`. Phase 11 replaces it with:

```go
func (e *impl) Resume(ctx context.Context, runID string, opts enginepkg.RunOptions) (enginepkg.RunHandle, error) {
    if opts.Store == nil {
        return nil, errors.New("engine: RunStore is required for Resume")
    }

    // 1. Load last checkpoint from store.
    state, err := opts.Store.LoadState(ctx, runID)
    if err != nil {
        return nil, fmt.Errorf("engine: load checkpoint for %s: %w", runID, err)
    }

    // 2. Scan trace for additional context (orphaned tool calls, etc.)
    resumeCtx, err := resume.ScanTrace(ctx, opts.Store, runID, state)
    if err != nil {
        return nil, fmt.Errorf("engine: scan trace for %s: %w", runID, err)
    }

    // 3. Re-parse the runbook at its original path.
    // (The runbook YAML must not have changed between runs.)
    plan, err := e.reloadPlan(ctx, state.RunbookPath)
    if err != nil {
        return nil, fmt.Errorf("engine: reload plan for %s: %w", runID, err)
    }

    // 4. Reconstruct Run from checkpoint state.
    run := resume.RebuildRun(runID, plan, state, resumeCtx, opts)

    // 5. Construct handle starting from the next step after checkpoint.
    runCtx, runCancel := context.WithCancel(context.Background())
    h := &runHandle{
        engine:   e,
        run:      run,
        events:   make(chan enginepkg.Event, 256),
        store:    opts.Store,
        onEvent:  opts.OnEvent,
        mu:       &sync.Mutex{},
        runCtx:   runCtx,
        cancelFn: runCancel,
    }
    h.started.Store(true) // already started (resumed)

    // 6. Emit step/resumed event.
    h.emitEventLocked(runCtx, trace.EventKindStepResumed, map[string]any{
        "run_id": runID,
        "source": "resume",
        "resumed_from_step": state.CurrentStep,
    })

    // 7. Signal handling (identical to Start).
    sigCh := e.cfg.Platform.NotifySignals(runCtx)
    go func() {
        // ... same signal handler as Start() ...
    }()

    return h, nil
}
```

### 8.2 `internal/resume` Package

#### TraceScanner

```go
package resume

import (
    "context"
    "encoding/json"

    "github.com/ormasoftchile/gert/v2/pkg/engine"
    tracepkg "github.com/ormasoftchile/gert/v2/pkg/trace"
)

// ResumeContext holds metadata gathered from scanning the trace file.
type ResumeContext struct {
    // LastCheckpointSeq is the sequence number of the last checkpoint event.
    LastCheckpointSeq int64

    // OrphanedToolCalls is a list of tool/invoked events with no matching tool/completed.
    OrphanedToolCalls []string // correlation IDs

    // CompletedStepIDs is the set of step IDs that have step/completed events.
    CompletedStepIDs map[string]bool

    // LastSeq is the highest sequence number seen.
    LastSeq int64
}

// ScanTrace reads the trace file and builds a ResumeContext.
func ScanTrace(ctx context.Context, store engine.RunStore, runID string, state engine.RunState) (*ResumeContext, error) {
    // Use TraceReader to read all events (via RunStore)
    // Scan for:
    // 1. Last checkpoint event → confirms snapshot alignment
    // 2. tool/invoked without tool/completed → orphaned calls
    // 3. Set of completed step IDs → for deduplication
    // 4. Maximum sequence number → for continuing sequence

    rc := &ResumeContext{
        CompletedStepIDs: make(map[string]bool),
    }

    // ... scan implementation: iterate events, build context ...

    return rc, nil
}
```

#### Run Reconstruction

```go
package resume

import (
    "github.com/ormasoftchile/gert/v2/pkg/engine"
)

// RebuildRun reconstructs a Run from checkpoint state and trace context.
func RebuildRun(
    runID string,
    plan *engine.ExecutionPlan,
    state engine.RunState,
    rc *ResumeContext,
    opts engine.RunOptions,
) *engine.Run {
    run := &engine.Run{
        ID:               runID,
        Status:           engine.RunStatusRunning,
        Plan:             plan,
        Vars:             make(map[string]any),
        StepResults:      make(map[string]*engine.StepResult),
        CurrentStepIndex: findStepIndex(plan, state.CurrentStep),
        StartedAt:        state.StartedAt,
        Actor:            opts.Actor,
        Mode:             opts.Mode,
        Sequence:         rc.LastSeq, // continue from last sequence
    }

    // Restore vars from checkpoint.
    for k, v := range state.Vars {
        run.Vars[k] = v
    }

    // Mark completed steps so they aren't re-executed.
    for stepID := range rc.CompletedStepIDs {
        run.StepResults[stepID] = &engine.StepResult{
            StepID: stepID,
            Status: engine.StepStatusCompleted,
        }
    }

    return run
}

// findStepIndex returns the index of the step with the given ID.
func findStepIndex(plan *engine.ExecutionPlan, stepID string) int {
    for i, step := range plan.Steps {
        if step.ID == stepID {
            return i
        }
    }
    return -1
}
```

### 8.3 Resumability Rules

| Step Type | Resume Behaviour |
|-----------|-----------------|
| `cli` | Re-execute from scratch (must be idempotent) |
| `manual` | Re-prompt operator (prior response lost) |
| `tool` (idempotent) | Re-issue call |
| `tool` (non-idempotent) | Mark failed; follow error path |
| `invoke` (child completed) | Treat as completed |
| `invoke` (child in-progress) | Resume child via nested run ID |
| `parallel`/`iterate`/`branch` | Structural; cursor advancement handles re-entry |

### 8.4 Orphaned Tool Call Handling

```go
// In resume.go, during reconstruction:
for _, correlationID := range rc.OrphanedToolCalls {
    stepID := findStepForCorrelation(plan, correlationID)
    if stepID == "" {
        continue
    }
    toolDef := findToolDef(plan, stepID)
    if toolDef != nil && toolDef.Idempotent {
        // Will be re-executed normally (step not in CompletedStepIDs).
        log.Printf("resume: re-executing orphaned idempotent tool call %s (step %s)", correlationID, stepID)
    } else {
        // Non-idempotent: mark as failed.
        run.StepResults[stepID] = &engine.StepResult{
            StepID: stepID,
            Status: engine.StepStatusFailed,
            Error:  fmt.Errorf("tool call %s was interrupted and tool is not idempotent", correlationID),
        }
        log.Printf("resume: marking orphaned non-idempotent tool call %s (step %s) as failed", correlationID, stepID)
    }
}
```

### 8.5 PID Lock Protocol

```go
// RunStore implementations must support locking for resume:
// 1. Attempt to acquire .runbook/runs/<run-id>/lock
// 2. If lock exists and PID is alive → return ErrRunLocked
// 3. If lock exists and PID is stale → remove and reacquire
// 4. Write current PID to lock file
// 5. Remove lock on clean exit (defer)
```

### 8.6 CLI Entry Point

```bash
gert exec --resume <run-id>
```

Maps to `Engine.Resume(ctx, runID, opts)`. The adapter reads `--resume` and calls `Resume` instead of `Start`.

---

## 9 RPC Additions

### 9.1 New RPC Methods

Two new JSON-RPC methods are added to the Phase 9 `internal/serve/rpc.go` dispatch table:

```go
// In handleRPC switch statement:
case "run.evidence":
    s.handleRunEvidence(w, r, req)
case "run.resume":
    s.handleRunResume(w, r, req)
```

### 9.2 `run.evidence`

Query evidence from a completed or in-progress run.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "run.evidence",
  "params": {
    "runID":  "<string>",          // required
    "stepID": "<string>",          // optional: filter to specific step
    "kinds":  ["text", "attachment"]  // optional: filter by evidence kind
  }
}
```

**Success Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "runID": "<string>",
    "evidence": [
      {
        "stepID": "deploy-pods",
        "records": [
          {
            "name": "stdout",
            "kind": "text",
            "value": "pod-abc  1/1  Running  0\n",
            "captured_at": "2026-04-24T10:30:00Z"
          }
        ]
      }
    ]
  }
}
```

**Implementation:**

```go
func (s *Server) handleRunEvidence(w http.ResponseWriter, r *http.Request, req rpcRequest) {
    var params struct {
        RunID  string   `json:"runID"`
        StepID string   `json:"stepID"`
        Kinds  []string `json:"kinds"`
    }
    if err := decodeParams(req.Params, &params); err != nil || strings.TrimSpace(params.RunID) == "" {
        writeRPC(w, rpcResponse{
            JSONRPC: "2.0",
            ID:      normalizeID(req.ID),
            Error:   &rpcError{Code: rpcInvalidParams, Message: "Invalid params"},
        })
        return
    }

    entry, ok := s.registry.Get(params.RunID)
    if !ok {
        writeRPC(w, rpcResponse{
            JSONRPC: "2.0",
            ID:      normalizeID(req.ID),
            Error:   &rpcError{Code: rpcRunNotFound, Message: "Run not found"},
        })
        return
    }

    // Read trace events of kind step/completed and extract evidence.
    reader := s.traceReaderForRun(params.RunID)
    filter := tracepkg.TraceFilter{
        Kinds:  []tracepkg.EventKind{tracepkg.EventKindStepCompleted},
        StepID: params.StepID,
    }
    events, err := reader.ReadFiltered(r.Context(), filter)
    if err != nil {
        writeRPC(w, rpcResponse{
            JSONRPC: "2.0",
            ID:      normalizeID(req.ID),
            Error:   &rpcError{Code: rpcInternalError, Message: "Internal error"},
        })
        return
    }

    // Extract evidence from payloads, apply kind filter.
    result := extractEvidence(events, params.Kinds)

    writeRPC(w, rpcResponse{
        JSONRPC: "2.0",
        ID:      normalizeID(req.ID),
        Result: map[string]any{
            "runID":    params.RunID,
            "evidence": result,
        },
    })
}
```

### 9.3 `run.resume`

Resume a previously interrupted run.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "run.resume",
  "params": {
    "runID": "<string>",   // required: ID of the interrupted run
    "actor": "<string>"    // optional: override actor identity
  }
}
```

**Success Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "runID": "<string>",
    "resumedFromStep": "<string>"
  }
}
```

**Error Codes:**
- `-32010` (`rpcRunNotFound`): No run with that ID exists in the store
- `-32011` (`rpcRunNotActive`): Run is already completed/cancelled (cannot resume)
- `-32013` (`rpcRunLocked`): Another process holds the PID lock

**Implementation:**

```go
const rpcRunLocked = -32013

func (s *Server) handleRunResume(w http.ResponseWriter, r *http.Request, req rpcRequest) {
    var params struct {
        RunID string `json:"runID"`
        Actor string `json:"actor"`
    }
    if err := decodeParams(req.Params, &params); err != nil || strings.TrimSpace(params.RunID) == "" {
        writeRPC(w, rpcResponse{
            JSONRPC: "2.0",
            ID:      normalizeID(req.ID),
            Error:   &rpcError{Code: rpcInvalidParams, Message: "Invalid params"},
        })
        return
    }

    actor := params.Actor
    if actor == "" {
        actor = "rpc-client"
    }

    handle, err := s.engine.Resume(r.Context(), params.RunID, engine.RunOptions{
        Mode:  engine.RunModeReal,
        Actor: actor,
        Store: s.store,
    })
    if err != nil {
        code := rpcInternalError
        msg := "Internal error"
        if strings.Contains(err.Error(), "not found") {
            code = rpcRunNotFound
            msg = "Run not found"
        } else if strings.Contains(err.Error(), "locked") {
            code = rpcRunLocked
            msg = "Run locked by another process"
        }
        writeRPC(w, rpcResponse{
            JSONRPC: "2.0",
            ID:      normalizeID(req.ID),
            Error:   &rpcError{Code: code, Message: msg},
        })
        return
    }

    state := handle.State()
    runCtx, cancel := context.WithCancel(context.Background())
    entry := &RunEntry{
        ID:          params.RunID,
        RunbookPath: state.RunbookPath,
        Handle:      handle,
        Cancel:      cancel,
        State:       state.Status,
        StartedAt:   state.StartedAt,
    }
    s.registry.Add(entry)
    s.wg.Add(1)
    go s.pumpEvents(runCtx, entry)

    writeRPC(w, rpcResponse{
        JSONRPC: "2.0",
        ID:      normalizeID(req.ID),
        Result: map[string]any{
            "runID":           params.RunID,
            "resumedFromStep": state.CurrentStep,
        },
    })
}
```

---

## 10 Test Plan

### Evidence Model (`pkg/evidence`)

| # | Test | Description |
|---|------|-------------|
| T01 | `TestHashFile_ValidFile` | SHA256 of a known file matches expected digest |
| T02 | `TestHashFile_NotFound` | Returns error for non-existent path |
| T03 | `TestHashBytes_Empty` | SHA256 of empty input matches known digest |
| T04 | `TestEvidenceRecord_JSONRoundTrip` | Marshal/unmarshal preserves all fields |
| T05 | `TestEvidenceKind_Values` | All kind constants are distinct and non-empty |

### Evidence Collection (`internal/evidence`)

| # | Test | Description |
|---|------|-------------|
| T06 | `TestDefaultCollector_CLI_StdoutStderr` | Extracts stdout + stderr from cli step output |
| T07 | `TestDefaultCollector_ToolResponse` | Extracts tool response from tool step output |
| T08 | `TestDefaultCollector_Attachment` | Copies file, deduplicates by SHA256, returns record |
| T09 | `TestDefaultCollector_Checklist` | Extracts checklist from manual step output |
| T10 | `TestDefaultCollector_NilOutput` | Returns nil for nil/empty output |
| T11 | `TestAttachmentStore_Dedup` | Two identical files produce one attachment entry |
| T12 | `TestEvidenceHook_IntegrationWithEngine` | Engine calls hook after step, evidence appears in trace event |

### Trace Reader (`internal/trace`)

| # | Test | Description |
|---|------|-------------|
| T13 | `TestJSONLReader_ReadAll` | Reads all events from valid JSONL |
| T14 | `TestJSONLReader_ReadSince` | Returns only events after given seq |
| T15 | `TestJSONLReader_SkipMalformed` | Skips corrupted lines, returns valid events |
| T16 | `TestJSONLReader_FilterByKind` | Filters by event kind |
| T17 | `TestJSONLReader_FilterByStepID` | Filters by step_id in payload |
| T18 | `TestJSONLReader_EmptyFile` | Returns empty slice for empty file |
| T19 | `TestJSONLReader_CancelledContext` | Respects context cancellation mid-read |

### Replay (`internal/replay`)

| # | Test | Description |
|---|------|-------------|
| T20 | `TestLoadScenario_Valid` | Parses valid scenario YAML |
| T21 | `TestLoadScenario_Invalid` | Returns error for malformed YAML |
| T22 | `TestReplayExecutor_CLI_MatchedFixture` | Returns pre-recorded stdout/stderr/exit_code |
| T23 | `TestReplayExecutor_CLI_NoMatch` | Returns error when allow_unmatched=false |
| T24 | `TestReplayExecutor_CLI_AllowUnmatched` | Returns empty output when allow_unmatched=true |
| T25 | `TestReplayExecutor_Tool` | Returns pre-recorded tool response |
| T26 | `TestReplayExecutor_Manual` | Returns pre-recorded evidence values |
| T27 | `TestReplayExecutorRegistry_Wrapping` | Wraps known kinds, returns nil for unknown |
| T28 | `TestReplay_EndToEnd` | Full replay of a 3-step runbook produces identical trace events |
| T29 | `TestReplay_GovernanceStillEnforced` | Governance blocks same step in replay as in real |

### Resumption (`internal/resume`)

| # | Test | Description |
|---|------|-------------|
| T30 | `TestScanTrace_FindsLastCheckpoint` | Identifies correct checkpoint event from trace |
| T31 | `TestScanTrace_OrphanedToolCall` | Detects tool/invoked without tool/completed |
| T32 | `TestRebuildRun_RestoredState` | Vars, step results, and sequence are correct |
| T33 | `TestResume_SkipsCompletedSteps` | Resumed run does not re-execute completed steps |
| T34 | `TestResume_ContinuesFromCheckpoint` | Execution continues from step after checkpoint |
| T35 | `TestResume_NonIdempotentToolFails` | Orphaned non-idempotent tool call is marked failed |
| T36 | `TestResume_NoStore_ReturnsError` | Resume with nil store returns meaningful error |

### RPC (`internal/serve`)

| # | Test | Description |
|---|------|-------------|
| T37 | `TestRPC_RunEvidence_Success` | Returns evidence for completed run |
| T38 | `TestRPC_RunEvidence_StepFilter` | Filters evidence by step ID |
| T39 | `TestRPC_RunEvidence_NotFound` | Returns -32010 for unknown run |
| T40 | `TestRPC_RunResume_Success` | Resumes interrupted run, returns handle |
| T41 | `TestRPC_RunResume_NotFound` | Returns -32010 for unknown run |
| T42 | `TestRPC_RunResume_AlreadyCompleted` | Returns -32011 for terminal run |

---

## 11 Decisions

### D1: `pkg/evidence` as a Leaf Package

**Context:** Evidence types are needed by `pkg/engine` (for `StepResult.Evidence`) and `internal/evidence` (for collection logic). Where should the types live?

**Decision:** Create `pkg/evidence` as a leaf package with zero dependencies on other gert packages. Only depends on Go stdlib.

**Consequences:**
- `pkg/engine` can import `pkg/evidence` without creating cycles
- External consumers can use evidence types without pulling in engine internals
- `internal/evidence` implements collection logic using `pkg/evidence` types
- Mirrors the `pkg/trace` → `internal/trace` pattern from Phase 10

### D2: Evidence Collection via Hook — Not Executor Modification

**Context:** Evidence must be captured at each step. Two approaches: (a) modify every executor to call an evidence collector, or (b) have the engine call a hook after each step.

**Decision:** Option (b) — an `EvidenceHook` in `EngineConfig`, called by the engine after step completion. Executors are unaware of evidence collection.

**Consequences:**
- Zero changes to Phase 5's executor implementations
- Single integration point in `internal/engine/engine.go` (one `if` block in `executeStep`)
- Easy to disable (set `EvidenceHook` to nil)
- Evidence collection can be extended without touching executor code
- Trade-off: the hook cannot capture pre-execution state (but snapshot checkpoints handle that)

### D3: Trace Reader as Symmetric Interface to TraceWriter

**Context:** Phase 10 defined `TraceWriter` (append-only). Phase 11 needs to read traces for evidence queries, replay, and resume.

**Decision:** Add `TraceReader` interface to `pkg/trace` with `ReadAll`, `ReadSince`, and `ReadFiltered` methods. `internal/trace.JSONLReader` is the file-based implementation.

**Consequences:**
- Symmetric with TraceWriter (same package, same event type)
- Filter predicates allow efficient querying without loading all events into memory for simple cases
- `ReadSince` enables incremental reading for SSE/WS event streaming
- Malformed line skipping provides crash-safe recovery (per §12.1.4 spec)

### D4: Replay via Executor Wrapping — Same Pattern as Dry-Run

**Context:** Phase 10 established `DryRunExecutorRegistry` wrapping. Replay is similar: intercept executors, return pre-recorded output.

**Decision:** `ReplayExecutorRegistry` wraps the real registry, replacing each executor with `ReplayExecutor` that returns scenario fixtures. Governance, conditions, and events run normally.

**Consequences:**
- Full pipeline exercised (governance can block replay steps, trace events are written)
- Pattern is established: `DryRunExecutorRegistry` (Phase 10) → `ReplayExecutorRegistry` (Phase 11)
- Regression testing: compare replay trace against golden trace
- Scenario files are YAML (consistent with runbook format)
- Only `gopkg.in/yaml.v3` dependency (already used throughout the project)

### D5: Resume Scans Trace + Loads Checkpoint — Two-Phase Recovery

**Context:** Resume could work from (a) trace only, (b) checkpoint only, or (c) both.

**Decision:** Option (c) — two-phase: load checkpoint snapshot for state, then scan trace for context (orphaned tool calls, maximum sequence). The checkpoint provides the authoritative state; the trace provides supplementary metadata.

**Consequences:**
- Checkpoint is the fast path (one file read for state)
- Trace scan catches edge cases (orphaned tool calls, sequence gaps)
- If snapshot is corrupted, resume fails cleanly (no partial state)
- Align with spec §12.3.1: "the last clean checkpoint event determines the recoverable state"

### D6: Non-Idempotent Orphaned Tool Calls Fail — Not Re-Execute

**Context:** A tool/invoked with no tool/completed means the call was interrupted. Should we re-execute?

**Decision:** If `idempotent: true`, re-execute. Otherwise, mark the step as failed and follow the normal failure path.

**Consequences:**
- Safe default: never re-execute a potentially destructive operation
- Runbook authors must declare `idempotent: true` on safe tools
- Warning is logged for operator awareness
- Aligns with spec §12.3.5: "default: the step is treated as failed"

### D7: `run.evidence` RPC Returns Evidence from Trace Events

**Context:** Evidence could be served from (a) in-memory RunHandle state, (b) the trace file, or (c) a separate evidence store.

**Decision:** Option (b) — read from trace file via `TraceReader`. The `step/completed` events carry embedded evidence records.

**Consequences:**
- Works for completed runs (no in-memory state needed)
- Works for in-progress runs (trace is append-only, events are already written)
- Single source of truth (the trace file)
- No separate evidence database to maintain
- Slight latency for large traces (mitigated by `ReadFiltered` with step ID)

### D8: Checkpoint Event Kind Uses String Literal — Not New EventKind Constant

**Context:** The `checkpoint` event is an internal implementation detail, not a user-facing event kind. Should we add `EventKindCheckpoint` to `pkg/trace/event.go`?

**Decision:** Use string literal `"checkpoint"` in the engine code. Do NOT add a constant to `pkg/trace/event.go`.

**Consequences:**
- `pkg/trace` stays focused on user-facing event types
- Checkpoint events are engine-internal; consumers should not depend on them
- The resume scanner matches on the string directly
- If checkpoint events are later promoted to public API, the constant can be added then
- Consistent with the spec categorization: "not user-facing but required for resumption"

---

## Appendix A: Dependency Graph

```
pkg/evidence (leaf — stdlib only)
    ↑
pkg/engine (imports pkg/evidence for StepResult.Evidence)
    ↑
internal/evidence (imports pkg/evidence, pkg/engine)
    ↑
internal/engine (imports internal/evidence via EvidenceHook)
    ↑
internal/adapter (imports internal/engine, internal/evidence, internal/replay, internal/resume)

pkg/trace (reader.go, filter.go — new files, no new dependencies)
    ↑
internal/trace (jsonl_reader.go — imports pkg/trace)
    ↑
internal/resume (imports internal/trace, pkg/engine, pkg/trace)

internal/replay (imports pkg/engine — reads scenario YAML via gopkg.in/yaml.v3)
```

## Appendix B: Migration Notes

- `StepResult.Evidence` is a new optional field — existing code that creates `StepResult` values is unaffected (zero value is nil slice)
- `EngineConfig.EvidenceHook` is optional — nil means no evidence collection (backward compatible)
- `TraceReader` is a new interface — no existing code is affected
- `Engine.Resume()` changes from returning `ErrNotImplemented` to functional — callers that check for `ErrNotImplemented` will see real behaviour
- No new external dependencies: `gopkg.in/yaml.v3` is already in `go.sum`
