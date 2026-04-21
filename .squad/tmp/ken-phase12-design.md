# Phase 12 — OpenTelemetry Integration

**Author:** Ken (Software Architect)  
**Date:** 2026-07-18  
**Requested by:** Cristian  
**Depends on:** Phase 11 (Evidence & Replay, sealed per ken-phase11-review.md)

---

## 1 Overview

Phase 12 delivers **first-class OpenTelemetry span integration** for gert v2 runs. Each run, step, tool invocation, and input resolution becomes a span in a trace hierarchy, enabling operators to correlate gert executions with downstream systems instrumented with OTel.

This phase also addresses three non-blocking housekeeping items from the Phase 11 review:
1. RunStore unit tests (225 lines previously untested)
2. Atomic attachment writes (temp→rename pattern)
3. Warn logging on attachment storage failures

### What Ships in Phase 12

| # | Deliverable | Summary |
|---|-------------|---------|
| 1 | **OTel span integration** | Spans for run, step, tool call, input prompt; hierarchical parent-child relationships |
| 2 | **Pluggable TracerProvider** | Optional dependency; gert works without OTel SDK |
| 3 | **Context propagation** | `context.Context` carries span through engine → executors → tools |
| 4 | **Attribute conventions** | Standard span attributes for step kind, run mode, tool transport, etc. |
| 5 | **RunStore tests** | Full unit test coverage for `internal/runstore` |
| 6 | **Atomic attachment writes** | Temp → sync → rename pattern in `AttachmentStore.Store()` |
| 7 | **Attachment error logging** | Warn-level logging on storage failures |

### What Does NOT Ship in Phase 12

- Metrics (OTel metrics API) — deferred to Phase 14+
- Baggage propagation to external tools — requires tool protocol extension
- Automatic span injection into CLI subprocess env vars
- `gert gc` trace retention — future CLI command
- OTel log bridge (OTel logging API) — gert uses NDJSON trace as primary log format

---

## 2 Package Layout

### New Files

```
v2/
├── pkg/
│   └── otel/
│       ├── doc.go              # Package docs — optional OTel integration
│       ├── tracer.go           # TracerProvider wrapper, noop fallback
│       ├── attributes.go       # Attribute key constants (semconv-aligned)
│       └── tracer_test.go      # Unit tests with in-memory SpanExporter
├── internal/
│   └── runstore/
│       └── dir_store_test.go   # NEW: unit tests for DirRunStore
```

### Modified Files

| File | Change |
|------|--------|
| `pkg/engine/engine.go` | Add `TracerProvider` field to `EngineConfig` (optional) |
| `internal/engine/engine.go` | Start/end spans at run, step, tool, input boundaries |
| `internal/evidence/attachment.go` | Use atomic write pattern (temp→sync→rename) |
| `internal/evidence/collector.go` | Add warn logging on attachment storage errors |
| `internal/adapter/wire.go` | Wire `TracerProvider` from env/options into `EngineConfig` |
| `internal/adapter/options.go` | Add `OTelEndpoint` and `OTelServiceName` to `WireOptions` |
| `cmd/gert/main.go` | Add `--otel-endpoint` and `--otel-service` flags |

### No Changes To

| Package | Reason |
|---------|--------|
| `pkg/trace/*` | gert's NDJSON trace is independent of OTel spans |
| `pkg/evidence/*` | Leaf package, no engine dependencies |
| `internal/replay/*` | Replay uses same engine; spans inherit naturally |
| `internal/resume/*` | Resume uses same engine; spans inherit naturally |

---

## 3 Key Architectural Decisions

### D-12-01: OTel Dependency Model — Pluggable Interface with Noop Default

**Decision:** OTel is **optional and pluggable**. The engine depends on a `TracerProvider` interface, not the OTel SDK directly. When no provider is configured, a noop tracer is used (zero allocation, no-op spans).

**Interface:**
```go
// In pkg/otel/tracer.go

package otel

import (
    "context"
)

// TracerProvider creates Tracers. When nil, NoopTracerProvider is used.
// This interface is a subset of go.opentelemetry.io/otel/trace.TracerProvider.
type TracerProvider interface {
    Tracer(name string, opts ...TracerOption) Tracer
}

// Tracer creates spans. Maps to OTel's Tracer interface.
type Tracer interface {
    Start(ctx context.Context, spanName string, opts ...SpanStartOption) (context.Context, Span)
}

// Span represents an active operation. Maps to OTel's Span interface.
type Span interface {
    End(opts ...SpanEndOption)
    SetAttributes(kv ...Attribute)
    RecordError(err error, opts ...EventOption)
    SetStatus(code StatusCode, description string)
}

// Attribute is a key-value pair attached to a span.
type Attribute struct {
    Key   string
    Value any
}

// StatusCode indicates span outcome.
type StatusCode int

const (
    StatusUnset StatusCode = iota
    StatusOK
    StatusError
)
```

**Rationale:**
- gert binaries remain small (~10MB) without OTel SDK bloat (~30MB with all exporters)
- Users opt-in to OTel by importing `go.opentelemetry.io/otel` and wiring a real provider
- Testing uses the same noop or an in-memory recorder
- Interface is forward-compatible with OTel Go SDK (method signatures match)

**Trade-off:** Users must wire OTel themselves; no auto-discovery. This is intentional — OTel configuration is environment-specific (Jaeger vs OTLP vs stdout).

---

### D-12-02: Span Structure — Hierarchical Mapping to gert Concepts

**Decision:** Spans form a tree rooted at the run, with children for each step, and grandchildren for tool calls and input prompts.

```
run: deploy-pods (span_id=A)
├── step: check-namespace (span_id=B, parent=A)
├── step: create-pod (span_id=C, parent=A)
│   ├── tool: kubectl (span_id=D, parent=C)
│   └── tool: kubectl (span_id=E, parent=C)
├── step: verify-health (span_id=F, parent=A)
│   └── input: prompt (span_id=G, parent=F)
└── step: notify (span_id=H, parent=A)
```

**Span Names:**
| gert Concept | Span Name Format | Example |
|--------------|------------------|---------|
| Run | `gert.run` | `gert.run` |
| Step | `gert.step.{kind}` | `gert.step.cli`, `gert.step.tool` |
| Tool call | `gert.tool.{transport}` | `gert.tool.stdio`, `gert.tool.mcp` |
| Input prompt | `gert.input.{type}` | `gert.input.choice`, `gert.input.collector` |
| Parallel branch | `gert.branch.{label}` | `gert.branch.deploy-a` |

**Span Kinds:**
- Run: `SpanKindInternal` (orchestration)
- Step: `SpanKindInternal` (orchestration)
- Tool call: `SpanKindClient` (calling external process)
- Input prompt: `SpanKindInternal` (waiting for human)

**Rationale:** This structure matches gert's execution model and enables:
- Filtering by span name prefix (`gert.step.*`)
- Waterfall visualization in Jaeger/Tempo
- Duration breakdown at each level

---

### D-12-03: Context Propagation — Span in Context Through Entire Call Chain

**Decision:** The `context.Context` passed through the engine carries the current span. Executors receive a context with the step span as parent; tool runtime receives a context with the tool span as parent.

**Propagation Points:**

```
Engine.Start(ctx)
    └─ startSpan(ctx, "gert.run") → runCtx
        └─ executeStep(runCtx, step)
            └─ startSpan(runCtx, "gert.step.cli") → stepCtx
                └─ executor.Execute(stepCtx, step, vars)
                    └─ (if tool) startSpan(stepCtx, "gert.tool.stdio") → toolCtx
                        └─ transport.Invoke(toolCtx, req)
```

**Implementation in `internal/engine/engine.go`:**

```go
func (h *runHandle) executeStep(ctx context.Context, step enginepkg.ResolvedStep) error {
    // Start step span
    spanCtx, span := h.tracer().Start(ctx, "gert.step."+step.Kind,
        otel.WithAttributes(
            otel.Attribute{Key: "gert.step.id", Value: step.ID},
            otel.Attribute{Key: "gert.step.kind", Value: step.Kind},
        ),
    )
    defer span.End()

    // ... existing code uses spanCtx instead of ctx ...
    result, execErr := exec.Execute(spanCtx, step, h.run.Vars)

    if execErr != nil {
        span.RecordError(execErr)
        span.SetStatus(otel.StatusError, execErr.Error())
    } else if result.Status == enginepkg.StepStatusFailed {
        span.SetStatus(otel.StatusError, "step failed")
    } else {
        span.SetStatus(otel.StatusOK, "")
    }
    // ...
}
```

**Rationale:**
- Standard OTel pattern; context is the propagation vehicle
- Executors and tools can create child spans without engine changes
- W3C Trace Context headers can be extracted from context for external calls

---

### D-12-04: Attribute Conventions — Semantic Conventions Alignment

**Decision:** Span attributes follow OTel semantic conventions where applicable, with `gert.*` prefix for gert-specific attributes.

**Standard Attributes (all spans):**

| Attribute | Type | Description |
|-----------|------|-------------|
| `gert.run.id` | string | Run UUID |
| `gert.runbook.path` | string | Path to runbook file |
| `gert.run.mode` | string | `real`, `replay`, `dry-run` |
| `gert.actor` | string | User/service executing the run |

**Step-Specific Attributes:**

| Attribute | Type | Description |
|-----------|------|-------------|
| `gert.step.id` | string | Step ID from runbook |
| `gert.step.kind` | string | `cli`, `tool`, `manual`, etc. |
| `gert.step.index` | int | Position in execution plan |
| `gert.step.duration_ms` | int | Execution duration |

**Tool-Specific Attributes:**

| Attribute | Type | Description |
|-----------|------|-------------|
| `gert.tool.name` | string | Tool definition name |
| `gert.tool.action` | string | Action being invoked |
| `gert.tool.transport` | string | `stdio`, `jsonrpc`, `mcp` |
| `process.command` | string | (semconv) Command executed |
| `process.exit_code` | int | (semconv) Exit code |

**Input-Specific Attributes:**

| Attribute | Type | Description |
|-----------|------|-------------|
| `gert.input.type` | string | `choice`, `decision`, `collector` |
| `gert.input.provider` | string | Provider name (env, vault, prompt) |

**Attribute Constants File:**

```go
// In pkg/otel/attributes.go
package otel

// Attribute keys following OTel semantic conventions and gert conventions.
const (
    AttrRunID        = "gert.run.id"
    AttrRunbookPath  = "gert.runbook.path"
    AttrRunMode      = "gert.run.mode"
    AttrActor        = "gert.actor"
    AttrStepID       = "gert.step.id"
    AttrStepKind     = "gert.step.kind"
    AttrStepIndex    = "gert.step.index"
    AttrStepDuration = "gert.step.duration_ms"
    AttrToolName     = "gert.tool.name"
    AttrToolAction   = "gert.tool.action"
    AttrToolTransport= "gert.tool.transport"
    AttrInputType    = "gert.input.type"
    AttrInputProvider= "gert.input.provider"
)
```

---

### D-12-05: Export Target — Pluggable via TracerProvider

**Decision:** gert does not configure export targets. The user provides a `TracerProvider` that is already configured with the desired exporter (OTLP, Jaeger, stdout, etc.).

**Built-in Convenience:** `pkg/otel` provides a `NewStdoutTracerProvider()` for debugging:

```go
// NewStdoutTracerProvider returns a TracerProvider that writes spans to stdout.
// Useful for debugging; not for production.
func NewStdoutTracerProvider() TracerProvider {
    // Implementation wraps OTel stdout exporter
}
```

**CLI Flags for Convenience:**

```
gert run --otel-endpoint=http://localhost:4317 ...
gert run --otel-stdout ...
```

When `--otel-endpoint` is provided, `cmd/gert` constructs an OTLP exporter and wires it. When `--otel-stdout` is provided, spans are written to stderr in JSON format.

**Rationale:**
- gert library users (serve, MCP) may have their own OTel setup
- CLI users get a zero-config option (`--otel-stdout`) for debugging
- Production users configure OTLP endpoint via flag or `OTEL_EXPORTER_OTLP_ENDPOINT` env var

---

### D-12-06: Integration Points — Span Open/Close Locations

**Decision:** Spans are opened and closed at precise lifecycle points in the engine.

| Event | Span Operation | Location |
|-------|----------------|----------|
| `run/started` | Start `gert.run` span | `runHandle` constructor, before first event |
| `run/completed` | End `gert.run` span | `finishRun()`, after final event |
| `step/started` | Start `gert.step.*` span | `executeStep()`, before executor call |
| `step/completed` | End step span | `executeStep()`, after result recorded |
| `tool/invoked` | Start `gert.tool.*` span | Tool executor, before transport call |
| `tool/completed` | End tool span | Tool executor, after transport returns |
| `input/prompted` | Start `gert.input.*` span | Input provider, before blocking |
| `input/received` | End input span | Input provider, after response |

**Parallel Branches:** Each branch gets its own span as a child of the step span:

```go
// In parallel step execution
for _, branch := range branches {
    branchCtx, branchSpan := tracer.Start(stepCtx, "gert.branch."+branch.Label)
    go func(ctx context.Context, span Span) {
        defer span.End()
        // execute branch steps with ctx
    }(branchCtx, branchSpan)
}
```

**Error Recording:**
- Infrastructure errors: `span.RecordError(err)` + `SetStatus(StatusError, ...)`
- Step failures (exit code != 0): `SetStatus(StatusError, "step failed")`
- Skipped steps: `SetStatus(StatusUnset, "skipped")` — not an error

---

### D-12-07: Relationship to Evidence/Trace — Complementary, Not Replacing

**Decision:** OTel spans and gert's NDJSON trace serve different purposes and coexist.

| Aspect | gert NDJSON Trace | OTel Spans |
|--------|-------------------|------------|
| **Purpose** | Audit log, replay source, evidence store | Distributed tracing, performance analysis |
| **Persistence** | File per run, indefinite | Exporter-dependent, often time-limited |
| **Content** | Full payloads, evidence records | Span names, attributes, timing |
| **Consumers** | `gert replay`, `gert serve`, compliance | Jaeger, Tempo, Honeycomb, Datadog |
| **Required** | Yes (core feature) | No (optional integration) |

**Correlation:** The `gert.run.id` span attribute matches the `run_id` in NDJSON events, enabling cross-reference:

```
OTel span: gert.run (gert.run.id="abc-123") → Jaeger
NDJSON: {"run_id": "abc-123", "kind": "step/completed", ...} → trace.jsonl
```

**No Duplication:** OTel spans do not contain evidence payloads, tool responses, or captured variables. These remain in the NDJSON trace. Spans contain:
- Timing (start, end, duration)
- Success/failure status
- Key attributes (step ID, tool name, etc.)
- Error messages (not full output)

---

## 4 Housekeeping Items from Phase 11

### 4.1 RunStore Unit Tests

**File:** `v2/internal/runstore/dir_store_test.go` (new)

**Test Cases:**

| Test | Description |
|------|-------------|
| `TestDirRunStore_SaveLoadState` | Round-trip state serialization |
| `TestDirRunStore_LoadState_Latest` | Selects highest-index snapshot |
| `TestDirRunStore_LoadState_SkipsTmpFiles` | Ignores `.tmp` files |
| `TestDirRunStore_LoadState_Empty` | Returns `os.ErrNotExist` on empty dir |
| `TestDirRunStore_WriteTrace` | Appends JSONL event |
| `TestDirRunStore_RegisterPlan` | Caches plan by run ID |
| `TestDirRunStore_TracePath` | Returns correct path |
| `TestDirRunStore_Concurrent` | Race-safe access from multiple goroutines |
| `TestWriteFileAtomic_PartialWrite` | Temp file cleaned on error |

**Implementation Notes:**
- Use `t.TempDir()` for isolation
- Verify file contents with `os.ReadFile`
- `TestDirRunStore_Concurrent`: 10 goroutines writing 100 events each, `-race`

### 4.2 Atomic Attachment Writes

**File:** `v2/internal/evidence/attachment.go`

**Before (current):**
```go
dst, err := os.Create(destPath)
// write directly to final path
```

**After:**
```go
tmpPath := destPath + ".tmp"
dst, err := os.Create(tmpPath)
// ... write and sync ...
if err := os.Rename(tmpPath, destPath); err != nil {
    os.Remove(tmpPath) // cleanup on rename failure
    return nil, err
}
```

**Rationale:** If the process crashes during write, the final path is either missing (no partial file) or complete. The dedup check (`os.Stat(destPath)`) correctly returns "not found" for partial writes.

### 4.3 Attachment Error Logging

**File:** `v2/internal/evidence/collector.go`

**Before (current):**
```go
if err != nil {
    continue  // best-effort: log but don't fail the step
}
```

**After:**
```go
if err != nil {
    log.Printf("[WARN] evidence: failed to store attachment %s: %v", path, err)
    continue  // best-effort: don't fail the step
}
```

**Future Enhancement:** Emit a `trace.EventKind("evidence/attachment_error")` event for systematic monitoring. Not in Phase 12 scope.

---

## 5 Test Plan

### 5.1 Unit Tests — `pkg/otel`

| Test | Description |
|------|-------------|
| `TestNoopTracer_Start` | Returns valid ctx and span, no allocations |
| `TestNoopSpan_End` | No-op, doesn't panic |
| `TestNoopSpan_SetAttributes` | Accepts any attributes, no-op |
| `TestAttribute_Types` | String, int, bool, float attribute values |
| `TestStdoutTracerProvider` | Writes span JSON to buffer |

### 5.2 Integration Tests — `internal/engine`

| Test | Description |
|------|-------------|
| `TestEngine_OTelSpans_Hierarchy` | Run → step → tool parent-child |
| `TestEngine_OTelSpans_Parallel` | Branch spans have correct parent |
| `TestEngine_OTelSpans_Error` | Failed step sets StatusError |
| `TestEngine_OTelSpans_Skipped` | Skipped step sets StatusUnset |
| `TestEngine_OTelSpans_Noop` | Nil provider doesn't panic |

**In-Memory SpanExporter:**

```go
type RecordingExporter struct {
    mu    sync.Mutex
    spans []*SpanData
}

func (e *RecordingExporter) ExportSpans(ctx context.Context, spans []SpanData) error {
    e.mu.Lock()
    e.spans = append(e.spans, spans...)
    e.mu.Unlock()
    return nil
}

func (e *RecordingExporter) Spans() []SpanData {
    e.mu.Lock()
    defer e.mu.Unlock()
    return slices.Clone(e.spans)
}
```

**Test Example:**

```go
func TestEngine_OTelSpans_Hierarchy(t *testing.T) {
    exporter := &RecordingExporter{}
    provider := NewTestTracerProvider(exporter)
    
    cfg := enginepkg.EngineConfig{
        // ... standard config ...
        TracerProvider: provider,
    }
    eng := engine.New(cfg)
    
    // Execute a 3-step plan with one tool call
    h, _ := eng.Start(ctx, plan, opts)
    // ... drive execution ...
    
    spans := exporter.Spans()
    
    // Verify hierarchy
    runSpan := findSpan(spans, "gert.run")
    stepSpans := findSpans(spans, "gert.step.*")
    toolSpan := findSpan(spans, "gert.tool.*")
    
    assert.Equal(t, runSpan.SpanID, stepSpans[0].ParentSpanID)
    assert.Equal(t, stepSpans[1].SpanID, toolSpan.ParentSpanID)
}
```

### 5.3 RunStore Tests

See §4.1 for test cases. Run with:
```
go test ./internal/runstore/... -race -count=5
```

---

## 6 Validation Gate

Phase 12 is complete when:

```bash
# Build
go build ./...                    # exit 0

# Vet
go vet ./...                      # exit 0

# Test (with race detector)
go test ./... -race -count=1      # all pass

# Verify OTel integration (manual)
gert run examples/simple.runbook.yaml --otel-stdout 2>&1 | grep "gert.run"
# Should output span JSON
```

---

## 7 Implementation Notes for Brian

### 7.1 OTel SDK Dependency

**Do NOT import OTel SDK in `pkg/otel`.** The interface abstracts the SDK. Concrete implementations that use the real SDK live in `internal/adapter/otel_adapter.go` or in user code.

Test code may use `go.opentelemetry.io/otel/sdk/trace/tracetest` for the in-memory exporter, imported only in `*_test.go` files.

### 7.2 Span Context Extraction for Tools

For tools that support W3C Trace Context (future), extract headers:

```go
import "go.opentelemetry.io/otel/propagation"

func injectTraceContext(ctx context.Context, env map[string]string) {
    prop := propagation.TraceContext{}
    carrier := propagation.MapCarrier(env)
    prop.Inject(ctx, carrier)
}
```

This injects `TRACEPARENT` and `TRACESTATE` into the environment. **Not in Phase 12 scope** — tools must declare trace context support.

### 7.3 Parallel Branch Span Ordering

Branch spans may end in any order (parallel execution). The trace viewer handles this. Ensure each branch span has the correct parent (the parallel step span, not the run span).

### 7.4 Noop Tracer Implementation

```go
type noopTracerProvider struct{}

func (noopTracerProvider) Tracer(name string, opts ...TracerOption) Tracer {
    return noopTracer{}
}

type noopTracer struct{}

func (noopTracer) Start(ctx context.Context, name string, opts ...SpanStartOption) (context.Context, Span) {
    return ctx, noopSpan{}
}

type noopSpan struct{}

func (noopSpan) End(...SpanEndOption)                  {}
func (noopSpan) SetAttributes(...Attribute)            {}
func (noopSpan) RecordError(error, ...EventOption)     {}
func (noopSpan) SetStatus(StatusCode, string)          {}
```

---

## 8 Summary

Phase 12 introduces **optional OpenTelemetry span integration** with a pluggable interface that keeps gert lightweight by default. Spans map to runs, steps, tool calls, and input prompts in a hierarchical structure, with attributes following OTel semantic conventions where applicable.

The design preserves gert's NDJSON trace as the authoritative audit log while enabling operators to correlate executions with distributed tracing backends like Jaeger, Tempo, or Honeycomb.

Three Phase 11 housekeeping items (RunStore tests, atomic attachment writes, error logging) are included to maintain code quality.

**Estimated Effort:** 3-4 days for Brian

**Next Phase (13+):**
- OTel metrics integration (run duration histograms, step counts)
- Baggage propagation to tools
- `gert gc` trace retention with OTel-aware cleanup
