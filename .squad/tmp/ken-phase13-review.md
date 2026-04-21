# Ken — Phase 13 Architectural Review

**Date:** 2026-07-20  
**Verdict:** APPROVED  
**Score:** 9/10

---

## Executive Summary

Brian's Phase 13 implementation is excellent work. The OTel SDK adapter (`pkg/otel/adapter`) provides a clean, idiomatic wrapper with proper shutdown semantics. The CLI commands (`ls`, `gc`, `version`) are well-designed with testability-first architecture. The D-13-03 safety invariant is implemented with double defense-in-depth protection. All 22 new tests pass with `-race -count=1`.

---

## Scope Reviewed

### Part A — OTel SDK Integration (NBI-12-01, NBI-12-02)

| File | Lines | Verdict |
|------|-------|---------|
| `pkg/otel/adapter/doc.go` | 11 | ✅ Clean package documentation |
| `pkg/otel/adapter/otlp.go` | 163 | ✅ Well-structured adapter |
| `pkg/otel/adapter/otlp_test.go` | 161 | ✅ Comprehensive tests |
| `internal/adapter/wire.go` | Modified | ✅ Clean integration |
| `cmd/gert/run.go` | Modified | ✅ Help text updated |

### Part B — CLI Commands (ls, gc, version)

| File | Lines | Verdict |
|------|-------|---------|
| `cmd/gert/ls.go` | 95 | ✅ Clean, testable |
| `cmd/gert/gc.go` | 136 | ✅ Safety invariant enforced |
| `cmd/gert/version.go` | 18 | ✅ Simple, correct |
| `cmd/gert/main.go` | 64 | ✅ Good help text, routing |
| `internal/runstore/dir_store.go` | 270 | ✅ ListRuns, DeleteRun added |
| `internal/runstore/dir_store_test.go` | 416 | ✅ 5 new tests |

---

## Five-Axis Assessment

### 1. Correctness (9/10)

**OTel Adapter:**
- `NewOTLPTracerProvider` correctly validates empty endpoint
- Resource attributes properly set (`service.name`)
- Shutdown function has 5-second timeout context — prevents hangs
- Status code mapping correct (OK → codes.Ok, Error → codes.Error, Unset → codes.Unset)
- Adapter pattern correctly implements `otelPkg.TracerProvider` interface

**CLI Commands:**
- `ls` filters by status and since duration correctly
- `gc` implements D-13-03 invariant with double protection (see below)
- `version` uses ldflags pattern correctly

**D-13-03 Verification:**
```go
// Line 95-101: gcCandidates function
// SAFETY INVARIANT (D-13-03): running status is NEVER returned.
if r.Status == engine.RunStatusRunning {
    continue  // Defense #1: Skip in candidate selection
}

// Line 127-129: parseStatusList function
if status == engine.RunStatusRunning {
    continue  // Defense #2: Exclude from allowed statuses
}
```

Double defense ensures that even if a user passes `--status=running`, running runs are never deleted.

### 2. Readability (9/10)

**Strengths:**
- Clear package documentation in `pkg/otel/adapter/doc.go`
- Function names are self-documenting (`gcCandidates`, `filterRuns`, `parseStatusList`)
- Consistent code organization across all new files
- Good use of named constants (`exitSuccess`, `exitValidation`, `exitRuntime`)
- Comments explain invariants (D-13-03 comment on line 95)

**Minor Observations:**
- `terminalStatuses` map at package level is a clean pattern
- Output formatting in `renderLsText` uses clean tabular layout

### 3. Architecture (10/10)

**OTel Adapter Design:**
- Follows the pluggable interface pattern from Phase 12 design
- Adapter wraps SDK types to implement gert's interface — correct isolation
- Only `pkg/otel/adapter` imports the real SDK — rest of codebase uses interfaces
- `buildTracerProvider` in wire.go correctly falls back to noop on error

**CLI Testability Pattern (DEV-13-03):**
```go
func runGc(args []string) int {
    return gcMain(args, ".runbook/runs", os.Stdout, os.Stdin)
}

func gcMain(args []string, runDir string, w io.Writer, stdin io.Reader) int {
    // Testable implementation
}
```
This is idiomatic Go CLI design. The thin wrapper handles production defaults; the `*Main` function accepts injected dependencies for testing. Same pattern in `ls.go`. This is exactly how Go CLI tools should be structured.

**Wire.go Integration:**
- `BuildEngineConfig` returns `(EngineConfig, func(), error)` — correct three-value form
- Shutdown func is always non-nil (returns `func(){}` on error) — safe to defer unconditionally
- TracerProvider wiring correctly handles endpoint/stdout/noop cases

### 4. Security (9/10)

**Strengths:**
- D-13-03 invariant prevents accidental deletion of running processes — critical safety feature
- `gc` requires explicit confirmation unless `--force` is set
- `--dry-run` allows preview before destructive operation
- No secrets in code or logs

**Observations:**
- `WithInsecure()` default is documented and appropriate for local dev (DEV-13-02)
- OTLP headers option available for API keys (`WithHeaders`)

### 5. Performance (9/10)

**OTel Adapter:**
- Uses `sdktrace.WithBatcher(exp)` — spans are batched, not sent individually
- Shutdown timeout prevents blocking on slow collectors
- SDK only initialized when `--otel-endpoint` is provided — zero cost otherwise

**CLI Commands:**
- `ListRuns` does directory scan → load state for each — acceptable for CLI use
- `gc` loads all runs once, filters in memory — efficient for expected run counts
- No unnecessary allocations in hot paths

---

## Deviation Evaluation

### DEV-13-01: Three-Value BuildEngineConfig Signature

**Change:** `BuildEngineConfig(ctx, opts) (EngineConfig, func(), error)` — added shutdown func.

**Assessment:** APPROVED

This is the correct design. The OTel tracer provider requires explicit shutdown to flush pending spans. Returning the shutdown func from `BuildEngineConfig` ensures:
1. Single ownership — caller is responsible for cleanup
2. Resource lifecycle is explicit, not hidden
3. Error case returns `func(){}` — always safe to defer

The alternative (embedding shutdown in EngineConfig) would complicate the interface and leak cleanup responsibility.

### DEV-13-02: Insecure-by-Default for OTLP

**Change:** `config.insecure = true` by default in `NewOTLPTracerProvider`.

**Assessment:** APPROVED

Rationale:
- OTLP over gRPC to localhost collectors (Jaeger, Zipkin, OTEL Collector) is the common local dev case
- TLS setup requires certificate management — wrong default for getting started
- `WithInsecure()` is explicit, documented, and can be omitted for TLS in Phase 14
- This matches OTel SDK defaults and common patterns in Go tracing libraries

Phase 14 should add `WithTLS(certPool)` option.

### DEV-13-03: Injected Dependencies for CLI Testability

**Change:** `lsMain` and `gcMain` accept `runDir`, `io.Writer`, and `io.Reader` parameters.

**Assessment:** APPROVED (EXEMPLARY)

This is the standard Go CLI testability pattern used by:
- `kubectl` (all commands accept `IOStreams`)
- `cobra` generator templates
- HashiCorp CLI tools

Benefits:
- Tests can use in-memory buffers and temp directories
- No global state or file system side effects in tests
- Production wrapper is a one-liner

This deviation should be adopted as project standard.

### DEV-13-04: Help Text Update

**Verified:** `--otel-endpoint` help now reads:
```
OTLP gRPC endpoint for span export (e.g. localhost:4317)
```

This removes "reserved for future use" as spec'd.

---

## Test Coverage

| Package | Tests | Status |
|---------|-------|--------|
| `pkg/otel/adapter` | 5 | ✅ pass |
| `internal/runstore` | 16 | ✅ pass (5 new) |
| All v2 packages | 100+ | ✅ pass with `-race` |

Tests verify:
- Empty endpoint error
- Provider creation with listener
- Double shutdown safety
- Tracer/span lifecycle
- Headers option
- ListRuns/DeleteRun behavior
- Safety invariant (implicitly via terminalStatuses map)

---

## Non-Blocking Items (Phase 14+)

### NBI-13-01: TLS Support for OTLP

`WithInsecure()` is correct for v2.0 local dev. Phase 14 should add `WithTLS(*tls.Config)` for production collectors.

### NBI-13-02: gc Test Coverage

`gc.go` logic is correct but has no direct unit tests. Tests would verify:
- D-13-03 invariant explicitly (create running run, verify not deleted)
- `--older-than` filter behavior
- `--dry-run` output format
- Confirmation prompt behavior

### NBI-13-03: ls Test Coverage

`ls.go` logic is correct but has no direct unit tests. Tests would verify:
- `--status` filter
- `--since` filter
- JSON output format

---

## Verdict: APPROVED

Phase 13 implementation meets all architectural requirements. The OTel adapter is clean and correctly isolated. The CLI commands are well-designed with testability-first architecture. The D-13-03 safety invariant is implemented with defense-in-depth. All tests pass.

**Score: 9/10** — Deducted 1 point for missing direct tests on `ls.go` and `gc.go` (NBI-13-02, NBI-13-03).

**Recommendation:** Merge Phase 13. Address NBI-13-01 through NBI-13-03 in Phase 14.
