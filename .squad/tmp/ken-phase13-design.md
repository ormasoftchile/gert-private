# Phase 13 — CLI Polish & OTLP Adapter

**Author:** Ken (Software Architect)  
**Date:** 2026-07-20  
**Requested by:** Cristian  
**Depends on:** Phase 12 (OpenTelemetry Integration, sealed)

---

## 1 Overview

Phase 13 delivers **two cohesive improvements**: (A) NBI carry-forward items from Phase 12 (OTLP adapter and CLI help updates) and (B) CLI polish commands (`gert ls`, `gert gc`, `gert version`). These combine into a single release that makes gert's production experience complete.

### Part A — NBI Carry-Forwards (Required)

| ID | Item | Est. | Priority |
|----|------|------|----------|
| NBI-12-01 | OTLP adapter package (`pkg/otel/adapter`) | 4-6h | Medium |
| NBI-12-02 | `--otel-endpoint` help text update | 30min | Low |
| NBI-12-03 | `context.AfterFunc` consideration | N/A | Deferred |

**NBI-12-03 Decision:** Go 1.25.7 is the current minimum (per `v2/go.mod`). `context.AfterFunc` was added in Go 1.21. However, the `mergeContexts` pattern is bounded (max 3 goroutines per step) and has been stable through Phases 11-12. **Defer to Phase 14+** as a low-priority optimization. Phase 13 should focus on user-facing features.

### Part B — CLI Polish (New Scope)

| # | Command | Summary |
|---|---------|---------|
| 1 | `gert ls` | List runs from `.runbook/runs/` with status, timestamps |
| 2 | `gert gc` | Clean up stale runs beyond retention (default 7 days) |
| 3 | `gert version` | Print version, commit hash, build date |
| 4 | Help text polish | Consistent descriptions for all commands |

**Why Part B?** With Phases 0-12 complete, gert v2 has a fully functional engine but gaps in production UX:
- No way to list past runs (must manually browse `.runbook/runs/`)
- No way to clean up stale runs (disk accumulates traces/attachments)
- No `--version` flag or `gert version` command
- Help text is minimal ("usage: gert <command> [args]")

These are 2-3 day scope items that close UX gaps before v2.0 GA.

### What Ships

| # | Deliverable |
|---|-------------|
| 1 | `pkg/otel/adapter` — OTLP TracerProvider wiring |
| 2 | `--otel-endpoint` flag becomes functional |
| 3 | `gert ls [--status=running|completed|failed] [--since=24h]` |
| 4 | `gert gc [--older-than=7d] [--dry-run]` |
| 5 | `gert version` / `gert --version` |
| 6 | CLI help text polish (all commands) |

### What Does NOT Ship

- `gert serve` rate limiting (Phase 14+)
- Authentication tokens for `gert serve` (Phase 14+)
- MCP tool transport enhancements (Barbara's domain, Phase 14+)
- Integration tests (Phase 14+ — tests now use unit + golden files)
- Schema validation enhancements (schema is complete)

---

## 2 Part A: OTLP Adapter

### 2.1 Package Layout

```
v2/
├── pkg/
│   └── otel/
│       ├── adapter/
│       │   ├── doc.go              # Package docs
│       │   ├── otlp.go             # NewOTLPTracerProvider
│       │   └── otlp_test.go        # Unit tests (mock OTLP endpoint)
```

### 2.2 Interface

```go
package adapter

import (
    otelPkg "github.com/ormasoftchile/gert/v2/pkg/otel"
)

// Option configures an OTLP TracerProvider.
type Option func(*config)

// WithServiceName sets the service.name resource attribute.
func WithServiceName(name string) Option

// WithHeaders adds HTTP headers to OTLP requests.
func WithHeaders(headers map[string]string) Option

// WithInsecure allows non-TLS connections (for local development).
func WithInsecure() Option

// NewOTLPTracerProvider creates a TracerProvider that exports spans via OTLP gRPC.
// Returns gert's otelPkg.TracerProvider interface, not the SDK type.
func NewOTLPTracerProvider(endpoint string, opts ...Option) (otelPkg.TracerProvider, func(), error)

// The returned func() shuts down the exporter and flushes pending spans.
```

### 2.3 Dependencies

**New dependencies** (v2/go.mod):
```
go.opentelemetry.io/otel v1.28.0
go.opentelemetry.io/otel/sdk v1.28.0
go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.28.0
```

**Decision D-13-01:** These are conditional dependencies — only imported by `pkg/otel/adapter`. Users who don't use `--otel-endpoint` don't pay the binary size cost (dead code elimination). No build tags needed.

### 2.4 Wire Integration

Modify `internal/adapter/wire.go`:

```go
func buildTracerProvider(opts WireOptions) (otelPkg.TracerProvider, func()) {
    if opts.OTelEndpoint != "" {
        provider, shutdown, err := adapter.NewOTLPTracerProvider(
            opts.OTelEndpoint,
            adapter.WithServiceName(opts.OTelServiceName),
        )
        if err != nil {
            // Log warning, fall back to noop
            log.Printf("gert: failed to create OTLP tracer: %v", err)
            return nil, func() {}
        }
        return provider, shutdown
    }
    if opts.OTelStdout {
        return &stdoutTracerProvider{serviceName: opts.OTelServiceName}, func() {}
    }
    return nil, func() {} // noop via ResolveProvider
}
```

### 2.5 Help Text Update (NBI-12-02)

In `cmd/gert/run.go`, change:
```go
otelEndpoint := fs.String("otel-endpoint", "", "OTLP gRPC endpoint for OTel spans (e.g. http://localhost:4317)")
```

To:
```go
otelEndpoint := fs.String("otel-endpoint", "", "OTLP gRPC endpoint (e.g. localhost:4317)")
```

Remove "reserved for future use" note since it's now functional.

---

## 3 Part B: CLI Commands

### 3.1 `gert ls` — List Runs

**Usage:**
```
gert ls [flags]
  -status string   Filter by status: running, completed, failed, cancelled (default: all)
  -since duration  Show runs from last duration (e.g. 24h, 7d) (default: 7d)
  -output string   Output format: text, json (default: text)
```

**Implementation:**

1. Scan `.runbook/runs/*/snapshots/` for latest snapshot files
2. Parse JSON, extract RunState (RunID, Status, RunbookPath, StartedAt, UpdatedAt)
3. Filter by status and since
4. Sort by StartedAt descending
5. Output as table (text) or JSON array

**Package location:** `cmd/gert/ls.go`

**Interface with runstore:**

```go
// Add to internal/runstore/dir_store.go
func (s *DirRunStore) ListRuns(ctx context.Context) ([]engine.RunState, error)
```

### 3.2 `gert gc` — Garbage Collection

**Usage:**
```
gert gc [flags]
  -older-than duration  Delete runs older than duration (default: 7d)
  -status string        Only delete runs with this status (default: completed,failed,cancelled)
  -dry-run              Show what would be deleted without deleting
  -force                Skip confirmation prompt
```

**Implementation:**

1. Call `ListRuns()` to get all runs
2. Filter by age (UpdatedAt or StartedAt older than threshold)
3. Filter by status (never delete "running")
4. In dry-run mode: print list and exit
5. In normal mode: prompt for confirmation (unless --force), then delete directories

**Package location:** `cmd/gert/gc.go`

**Interface with runstore:**

```go
// Add to internal/runstore/dir_store.go
func (s *DirRunStore) DeleteRun(ctx context.Context, runID string) error
```

### 3.3 `gert version`

**Usage:**
```
gert version
gert --version
```

**Output:**
```
gert v2.0.0 (commit abc1234) built 2026-07-20T12:00:00Z
```

**Implementation:**

Use `-ldflags` at build time:
```go
// cmd/gert/version.go
var (
    Version   = "dev"
    Commit    = "unknown"
    BuildDate = "unknown"
)
```

Build with:
```bash
go build -ldflags "-X main.Version=v2.0.0 -X main.Commit=$(git rev-parse --short HEAD) -X main.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
```

### 3.4 Help Text Polish

Update `cmd/gert/main.go` `printUsage()`:

```go
func printUsage() {
    fmt.Fprintln(os.Stderr, "gert — Governed Executable Runbook Engine")
    fmt.Fprintln(os.Stderr, "")
    fmt.Fprintln(os.Stderr, "Usage: gert <command> [flags]")
    fmt.Fprintln(os.Stderr, "")
    fmt.Fprintln(os.Stderr, "Commands:")
    fmt.Fprintln(os.Stderr, "  run       Execute a runbook")
    fmt.Fprintln(os.Stderr, "  dry-run   Validate a runbook without side effects")
    fmt.Fprintln(os.Stderr, "  ls        List past runs")
    fmt.Fprintln(os.Stderr, "  gc        Clean up old runs")
    fmt.Fprintln(os.Stderr, "  version   Print version information")
    fmt.Fprintln(os.Stderr, "")
    fmt.Fprintln(os.Stderr, "Run 'gert <command> --help' for command-specific flags.")
}
```

---

## 4 Files to Create/Modify

### New Files

| Path | Purpose |
|------|---------|
| `pkg/otel/adapter/doc.go` | Package documentation |
| `pkg/otel/adapter/otlp.go` | OTLP TracerProvider implementation |
| `pkg/otel/adapter/otlp_test.go` | Unit tests |
| `cmd/gert/ls.go` | `gert ls` command |
| `cmd/gert/gc.go` | `gert gc` command |
| `cmd/gert/version.go` | `gert version` command + build vars |

### Modified Files

| Path | Change |
|------|--------|
| `v2/go.mod` | Add OTel SDK dependencies |
| `internal/adapter/wire.go` | Wire OTLP provider when endpoint configured |
| `internal/runstore/dir_store.go` | Add `ListRuns()`, `DeleteRun()` |
| `internal/runstore/dir_store_test.go` | Tests for new methods |
| `cmd/gert/main.go` | Add ls, gc, version commands; polish help |
| `cmd/gert/run.go` | Update --otel-endpoint help text |

---

## 5 Test Plan

### Unit Tests

| Package | Test |
|---------|------|
| `pkg/otel/adapter` | `TestNewOTLPTracerProvider_Success` — mock endpoint accepts spans |
| `pkg/otel/adapter` | `TestNewOTLPTracerProvider_InvalidEndpoint` — returns error |
| `pkg/otel/adapter` | `TestShutdown_FlushesSpans` — verify flush on shutdown |
| `internal/runstore` | `TestDirRunStore_ListRuns` — returns all runs |
| `internal/runstore` | `TestDirRunStore_ListRuns_Empty` — handles empty dir |
| `internal/runstore` | `TestDirRunStore_DeleteRun` — removes run directory |
| `internal/runstore` | `TestDirRunStore_DeleteRun_NotFound` — graceful on missing |

### CLI Tests

| Test | Description |
|------|-------------|
| `TestLs_Empty` | `gert ls` with no runs prints empty list |
| `TestLs_WithRuns` | `gert ls` shows runs from snapshots |
| `TestLs_FilterStatus` | `gert ls -status=completed` filters |
| `TestLs_OutputJSON` | `gert ls -output=json` outputs valid JSON |
| `TestGc_DryRun` | `gert gc --dry-run` lists but doesn't delete |
| `TestGc_DeletesOldRuns` | `gert gc --older-than=0s --force` deletes all |
| `TestGc_PreservesRunning` | `gert gc` never deletes running runs |
| `TestVersion` | `gert version` outputs expected format |

### Integration

| Test | Scope |
|------|-------|
| Manual: OTLP endpoint | Run with Jaeger/Zipkin, verify spans appear |
| Manual: `gert ls` | Create runs, verify listing |
| Manual: `gert gc` | Verify cleanup behavior |

---

## 6 Validation Gate

Phase 13 is complete when:

```bash
cd v2
go build ./...         # exit 0
go vet ./...           # exit 0
go test ./... -race -count=3  # all pass
```

Plus manual verification:
- [ ] `gert --help` shows all commands
- [ ] `gert version` prints version
- [ ] `gert ls` lists runs (empty if none)
- [ ] `gert gc --dry-run` shows what would be deleted
- [ ] `gert run ... --otel-endpoint=localhost:4317` exports spans (with OTLP collector running)

---

## 7 Risk Assessment

| Risk | Mitigation |
|------|------------|
| OTel SDK adds ~10MB to binary | Dead code elimination keeps core lean; only used if endpoint configured |
| `gert gc` could delete wrong data | Never delete "running" status; require --force for non-dry-run |
| Version info missing at build time | Defaults to "dev/unknown" — acceptable for local builds |
| OTLP connection failures | Log warning, fall back to noop — never fail the run |

---

## 8 Timeline

| Day | Tasks |
|-----|-------|
| 1 | OTLP adapter package + tests + wire integration |
| 2 | `gert ls` + `gert gc` commands with tests |
| 3 | `gert version` + help polish + manual testing |
| 4 | Buffer for iteration and review fixes |

**Estimated total:** 3-4 days for Brian.

---

## 9 Decisions Summary

| ID | Decision | Rationale |
|----|----------|-----------|
| D-13-01 | No build tags for OTel SDK | Dead code elimination is sufficient; simpler build |
| D-13-02 | Defer NBI-12-03 (context.AfterFunc) | Low impact, bounded goroutine count; focus on UX |
| D-13-03 | `gert gc` never deletes running status | Safety invariant to prevent data loss |
| D-13-04 | Version via -ldflags | Standard Go pattern; works with CI/CD |
| D-13-05 | Part B over integration tests | User-facing UX gaps are higher priority than additional test coverage |

---

*Ken, Software Architect*
