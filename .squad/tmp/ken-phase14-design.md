# Ken — Phase 14 Design: E2E Integration Tests & NBI Closure

**Date:** 2026-07-21  
**Author:** Ken (Software Architect)  
**Implementor:** Brian  
**Phase:** 14

---

## Overview

Phase 14 addresses three NBI carry-forwards from Phase 13 (Part A) and delivers the **End-to-End Integration Test Suite** (Part B) — the highest-value quality investment for v2.0 GA.

**Estimated Effort:** 3-4 days for Brian

---

## Part A — NBI Carry-Forwards

### NBI-13-01: `WithTLS(*tls.Config)` for OTLP Adapter

**Priority:** Medium  
**File:** `pkg/otel/adapter/otlp.go`

**Current state:** Adapter defaults to insecure (non-TLS) for local dev. Production OTLP collectors typically require TLS.

**Implementation:**

```go
// WithTLS enables TLS with the given config. If tlsCfg is nil,
// uses the system default TLS config.
func WithTLS(tlsCfg *tls.Config) Option {
    return func(c *config) {
        c.insecure = false
        c.tlsConfig = tlsCfg
    }
}
```

Modify `NewOTLPTracerProvider`:

```go
if cfg.insecure {
    dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
} else {
    creds := credentials.NewTLS(cfg.tlsConfig) // nil → system default
    dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
}
```

**Tests:**
- `TestNewOTLPTracerProvider_WithTLS_NilConfig` — uses system default certs
- `TestNewOTLPTracerProvider_WithTLS_CustomConfig` — uses provided tls.Config

**Files modified:**
- `pkg/otel/adapter/otlp.go` — add `WithTLS` option, update config struct
- `pkg/otel/adapter/otlp_test.go` — add TLS tests

---

### NBI-13-02: Additional `gert gc` Unit Tests

**Priority:** Low  
**File:** `cmd/gert/gc_test.go`

Add tests for edge cases not covered in Phase 13:

| Test | Description |
|------|-------------|
| `TestGc_RunningStatus_NeverDeleted` | Explicitly verify D-13-03 invariant |
| `TestGc_MixedStatuses` | Mix of completed, failed, cancelled, running |
| `TestGc_StatusFilter_Running_Rejected` | `--status=running` silently excluded |
| `TestGc_OlderThan_EdgeCase` | Run exactly at cutoff boundary |

**Files modified:**
- `cmd/gert/gc_test.go` — 4 new test functions

---

### NBI-13-03: `gert ls --output=json` Schema Documentation

**Priority:** Low  
**File:** `cmd/gert/ls.go` (godoc comment)

Document the JSON output schema in the godoc for `lsMain`:

```go
// lsMain lists runs in the given directory.
//
// JSON output schema (--output=json):
//
//     [
//       {
//         "RunID": "string (UUID v4)",
//         "RunbookPath": "string (relative path)",
//         "Status": "string (pending|running|waiting|completed|failed|cancelled)",
//         "CurrentStep": "string (step ID, may be empty)",
//         "CurrentStepIndex": number,
//         "Vars": object (runtime variables),
//         "StartedAt": "string (RFC3339Nano)",
//         "UpdatedAt": "string (RFC3339Nano)"
//       }
//     ]
func lsMain(args []string, runDir string, w io.Writer) int { ... }
```

**Files modified:**
- `cmd/gert/ls.go` — godoc comment for `lsMain`

---

## Part B — End-to-End Integration Test Suite

### Rationale

After 13 phases, v2 has:
- **73 test files** (all unit or package-level)
- **Zero integration tests** that exercise the full stack

This is the single largest quality gap. An e2e test exercises:
1. Parser → Planner → Engine → Executors → Trace → RunStore
2. Real file I/O, real subprocess execution, real timing
3. Catches integration bugs that unit tests cannot (dependency wiring, event ordering, file format compatibility)

**Why now:** v2 is feature-complete (Phases 0–13). No more moving targets. E2E tests can lock in expected behavior before v2.0 GA.

**Why over other candidates:**
- `gert serve` hardening: Security features are not blocking for internal use
- Schema validation: Parser already rejects malformed runbooks; nice-to-have
- MCP transport: Scope too large for one phase
- `mergeContexts` optimization: Bounded goroutine is stable; pure performance
- `run.list` wiring: Already works (returns in-memory registry); disk persistence is Phase 15+

---

### Package Layout

```
v2/
└── internal/
    └── e2e/
        ├── doc.go           # Package documentation
        ├── e2e_test.go      # Test entrypoint and fixtures
        ├── helpers_test.go  # Test utilities (file setup, assertions)
        └── testdata/
            ├── echo-runbook.yaml      # Simple: 1 CLI step
            ├── vars-runbook.yaml      # Variable interpolation
            ├── branch-runbook.yaml    # Conditional branching
            ├── iterate-runbook.yaml   # Loop with early exit
            └── manual-runbook.yaml    # Manual step (skipped)
```

---

### Key Interfaces

**Test Harness:**

```go
// E2EHarness configures and runs full engine executions.
type E2EHarness struct {
    WorkDir     string           // temp directory for this test
    RunbookPath string           // path to runbook YAML
    Inputs      map[string]any   // pre-populated inputs
    Timeout     time.Duration    // max execution time
}

// Run executes the runbook through the full stack.
// Returns the final RunState and any error.
func (h *E2EHarness) Run(t *testing.T) (engine.RunState, error)

// AssertCompleted verifies run completed successfully.
func (h *E2EHarness) AssertCompleted(t *testing.T, state engine.RunState)

// AssertTrace verifies expected events appear in trace.
func (h *E2EHarness) AssertTrace(t *testing.T, kinds ...trace.EventKind)
```

---

### Test Cases

| Test | Runbook | Validates |
|------|---------|-----------|
| `TestE2E_SimpleEcho` | echo-runbook.yaml | Parser → Planner → Engine → CLI Executor → Trace |
| `TestE2E_VarInterpolation` | vars-runbook.yaml | Input resolution, output capture, var propagation |
| `TestE2E_BranchTrue` | branch-runbook.yaml | Branch condition evaluation, path selection |
| `TestE2E_BranchFalse` | branch-runbook.yaml | Alternate branch path |
| `TestE2E_IterateAll` | iterate-runbook.yaml | Iterate over list, loop completion |
| `TestE2E_IterateEarlyExit` | iterate-runbook.yaml | until_output early exit |
| `TestE2E_ManualSkip` | manual-runbook.yaml | Manual step auto-skipped in non-interactive mode |
| `TestE2E_TracePersistence` | echo-runbook.yaml | Trace file written, parseable, contains expected events |
| `TestE2E_ResumeFromCheckpoint` | vars-runbook.yaml | Pause mid-run, resume, verify state continuity |
| `TestE2E_CancelMidRun` | iterate-runbook.yaml | Cancel during iteration, verify clean shutdown |

**Coverage goals:**
- All step types (cli, branch, iterate, manual) exercised
- Full event flow from run.started to run.completed
- Trace file written and readable
- Resume/checkpoint verified
- Cancellation verified

---

### Testdata Runbooks

**echo-runbook.yaml** (simplest case):
```yaml
name: e2e-echo
version: "1.0"
steps:
  - id: echo-step
    name: Echo test
    cli:
      command: echo
      args: ["hello", "e2e"]
```

**vars-runbook.yaml** (variable propagation):
```yaml
name: e2e-vars
version: "1.0"
inputs:
  - name: greeting
    type: string
steps:
  - id: step1
    name: Echo greeting
    cli:
      command: echo
      args: ["{{.greeting}}"]
    outputs:
      message: stdout
  - id: step2
    name: Use output
    cli:
      command: echo
      args: ["received: {{.message}}"]
```

**branch-runbook.yaml** (conditional):
```yaml
name: e2e-branch
version: "1.0"
inputs:
  - name: flag
    type: bool
steps:
  - id: branch-step
    name: Conditional
    branch:
      condition: "{{.flag}}"
      then:
        - id: then-step
          name: Then path
          cli:
            command: echo
            args: ["then"]
      else:
        - id: else-step
          name: Else path
          cli:
            command: echo
            args: ["else"]
```

---

### Files to Create

| File | Description |
|------|-------------|
| `internal/e2e/doc.go` | Package doc |
| `internal/e2e/e2e_test.go` | 10 test functions |
| `internal/e2e/helpers_test.go` | Harness, assertions |
| `internal/e2e/testdata/echo-runbook.yaml` | Simple CLI |
| `internal/e2e/testdata/vars-runbook.yaml` | Variable flow |
| `internal/e2e/testdata/branch-runbook.yaml` | Branching |
| `internal/e2e/testdata/iterate-runbook.yaml` | Iteration |
| `internal/e2e/testdata/manual-runbook.yaml` | Manual step |

---

### Files to Modify

| File | Change |
|------|--------|
| `pkg/otel/adapter/otlp.go` | Add `WithTLS` option |
| `pkg/otel/adapter/otlp_test.go` | Add TLS tests |
| `cmd/gert/gc_test.go` | 4 new edge case tests |
| `cmd/gert/ls.go` | JSON schema godoc |

---

## Design Decisions

### D-14-01: E2E Tests in `internal/e2e/` Package

**Decision:** E2E tests live in a dedicated `internal/e2e/` package, not alongside existing unit tests.

**Rationale:**
- Clear separation of test types (unit vs integration)
- E2E tests have different characteristics (slower, file I/O, subprocess)
- Can run unit tests quickly with `go test ./internal/...` excluding e2e
- Common pattern in Go codebases (Kubernetes, Vitess, CockroachDB)

### D-14-02: E2E Tests Use Real Executors, Not Fakes

**Decision:** E2E tests execute real CLI commands (echo, cat) via the real CLI executor.

**Rationale:**
- The entire point of e2e is exercising real behavior
- Commands are simple POSIX utilities available on all CI runners
- Fakes would reduce e2e to another unit test

### D-14-03: context.AfterFunc Deferred (Again)

**Decision:** NBI-12-03 (`mergeContexts` optimization) remains deferred.

**Rationale:**
- Current goroutine pattern is bounded and stable
- E2E test suite has higher value per Brian-day
- Go 1.25.7 `context.AfterFunc` is available; can be added anytime
- When added, e2e tests will validate it didn't break anything

### D-14-04: No Parallel E2E Tests Initially

**Decision:** E2E tests run sequentially with `t.Parallel()` disabled initially.

**Rationale:**
- Simpler debugging when tests share temp directory roots
- Subprocess execution may have race-like behaviors
- Can enable parallel later once suite is stable

---

## Validation Gate

```bash
cd v2
go build ./...            # Must exit 0
go vet ./...              # Must exit 0
go test ./... -race -count=3  # Must pass all packages including internal/e2e
```

---

## Summary

| Item | Scope | Files |
|------|-------|-------|
| NBI-13-01 | TLS option for OTLP | 2 |
| NBI-13-02 | gc edge case tests | 1 |
| NBI-13-03 | ls JSON schema doc | 1 |
| Part B | E2E test suite | 8 new |

**Total new files:** 8  
**Total modified files:** 4  
**Estimated effort:** 3-4 Brian-days

---

*Ken, Software Architect*
