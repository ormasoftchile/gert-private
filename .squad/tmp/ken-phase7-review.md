# Phase 7 Review — Extension Host

**Reviewer:** Ken (Software Architect)  
**Date:** 2026-04-21  
**Commit range:** Phase 7 untracked + modified files  
**Validation:** `go build/vet/test -race -count=3` ✅ (pre-verified)

---

## Scores

| # | Criterion | Score | Notes |
|---|-----------|-------|-------|
| 1 | Spec compliance | 7/10 | State enum diverges from §04 (6 states vs 8); capability naming deviates; hello-ext manifest doesn't match design's richness |
| 2 | Interface correctness | 9/10 | `ExtensionHost` interface matches design exactly; `ToolRegistry.Register()` added to both interface and impl correctly |
| 3 | Protocol correctness | 8/10 | JSON-RPC 2.0 codec is clean; handshake follows initialize → contributions/list; init params simpler than spec (no runContext, no protocolVersion) |
| 4 | Isolation model | 9/10 | One process per extension, enforced via `exec.CommandContext`; no shared state leaks between processes |
| 5 | Capability/grants model | 8/10 | Intersection logic (`intersectGrants`) correct; enforcement gate works; only 6 of 10 spec capabilities implemented (acceptable for v2.0 MVP if documented) |
| 6 | Lifecycle management | 7/10 | Load/health/Shutdown implemented; **engine never calls Shutdown()** — extension processes leak on run completion; no auto-restart (correct per D6) |
| 7 | Test coverage | 8/10 | 36 test functions (exceeds 28 target); integration tests use real binary; missing: `discovery_test.go`, `process_test.go`, duplicate `Register()` test |
| 8 | Contribution registration | 9/10 | Tools/providers/policy all registered eagerly at Load time via handshake; capability enforcement gates each type; `Register()` called on tool registry |
| 9 | Engine integration | 7/10 | `ExtensionHost.Load()` called before run start ✅; passes `nil` manifest (loses runbook-level extensions); **no Shutdown() call at run end** |
| 10 | Code quality | 8/10 | Proper mutex coverage; atomic timing vars for tests; context respected; `readLoop` goroutine properly handles EOF; minor: `cmd.Stderr = io.Discard` loses debug info |

---

## Total: 80/100 — APPROVED (borderline)

**Verdict:** APPROVED with required follow-ups

All criteria ≥ 7. No blocking scores (≤3). The implementation is solid and passes race detection with 3 iterations. The two 7/10 scores (spec compliance, engine integration) reflect gaps that are easily fixable without architectural change.

---

## Critical Issues

None. No data loss risk, no security vulnerability, no broken core functionality.

---

## Important Issues (must address before Phase 8)

### I1: Engine never calls `ExtensionHost.Shutdown()` — process leak

**File:** `v2/internal/engine/engine.go:47-51`

The engine calls `Load()` at run start but **never** calls `Shutdown()` when the run completes, fails, or is cancelled. Extension child processes survive indefinitely.

**Fix:** Add `Shutdown()` call in `completeRun()` and `failRun()`:
```go
if e.cfg.ExtensionHost != nil {
    _ = e.cfg.ExtensionHost.Shutdown(ctx)
}
```

Or use a deferred call pattern at the `Start()` level that fires when the runHandle completes.

---

### I2: `Load(ctx, nil)` — runbook extensions never passed to host

**File:** `v2/internal/engine/engine.go:48`

```go
e.cfg.ExtensionHost.Load(ctx, nil)
```

The `nil` manifest means the host only discovers extensions from `.gert/extensions/` directory scan. Runbook-level `extensions:` declarations (from `schema.Runbook.Extensions`) are never plumbed to the host. Per D7, the manifest should be constructed from the execution plan's runbook metadata.

**Fix:** Build a `ProjectManifest` from `plan.Metadata` or add the runbook's extension refs to the plan and pass them to `Load()`.

---

### I3: State enum names diverge from spec §04

**File:** `v2/pkg/extension/state.go`

Spec defines 8 states: `discovered, verified, starting, initializing, ready, draining, stopped, crashed`.

Implementation defines 7 states with different names: `Unloaded, Discovered, Starting, Handshaking, Loaded, Failed, Shutdown`.

Missing states: `verified`, `ready` (→ `Loaded`), `draining`, `crashed` (→ `Failed`).

This makes it impossible to distinguish between "extension failed handshake" and "extension crashed during execution" — both are `StateFailed`.

**Fix:** Add at minimum `StateCrashed` distinct from `StateFailed`, and `StateReady` distinct from `StateLoaded`, to enable proper §04 state machine tracing.

---

### I4: Capability constant names don't match spec taxonomy

**File:** `v2/pkg/extension/capability.go`

| Spec name | Implementation name |
|-----------|-------------------|
| `capability/env-read` | `capability/read-env` |
| `capability/file-read` | `capability/read-files` |
| N/A | `capability/exec-process` (non-spec) |

Four spec capabilities missing entirely: `schema-extension`, `event-subscription`, `file-write`, `network`.

**Fix:** Rename to match spec strings exactly. Missing capabilities can be declared as constants with a comment `// Not implemented in v2.0` — this allows manifest validation to accept them without enforcement logic.

---

## Non-blocking Recommendations

### R1: `cmd.Stderr = io.Discard` loses extension debug output

**File:** `v2/internal/extension/process.go:47`

Extension stderr is valuable for diagnosing crashes. Consider capturing to a buffer (bounded, e.g., last 4KB) for inclusion in error messages when an extension enters `StateFailed`.

---

### R2: Missing `discovery_test.go` and `process_test.go`

The design doc specifies tests for discovery edge cases (duplicate names, directory scan, CLI override) and process management (start failure, kill escalation). These are currently covered indirectly through host_test.go's integration test but lack focused unit coverage.

---

### R3: `Register()` duplicate test missing in `registry_test.go`

The `MapRegistry.Register()` correctly returns an error on duplicate names (line 53), but there's no test exercising this path. Add:
```go
func TestMapRegistry_Register_Duplicate(t *testing.T) {
    reg := NewMapRegistry([]toolpkg.ToolDef{{Name: "dup"}})
    if err := reg.Register(toolpkg.ToolDef{Name: "dup"}); err == nil {
        t.Fatalf("expected duplicate error")
    }
}
```

---

### R4: hello-ext manifest is minimal vs design doc's reference

**File:** `v2/cmd/extensions/hello-ext/gert-extension.yaml`

The design doc specifies the reference extension should contribute both a tool AND a policy rule, with richer manifest fields (`apiVersion: extension/v2`, `meta.author`, `compatibility.api-version` as semver range, `transport` field, `platform` constraints). The actual implementation is a stripped-down version.

This is fine for v2.0 testing, but consider enriching it to serve as documentation for extension authors.

---

### R5: `initParams` is simpler than spec wire format

**File:** `v2/internal/extension/handshake.go:18-21`

Spec's `extension/initialize` params include `host`, `protocolVersion`, `grantedCapabilities`, `runContext`. Implementation sends only `hostVersion` and `grants`. Extensions depending on protocol version negotiation or run context won't receive them.

Acceptable for v2.0 (only the hello-ext reference extension exists), but should be expanded before third-party extension support.

---

### R6: Health ping uses parent context — may cancel prematurely

**File:** `v2/internal/extension/health.go:43`

```go
ctx, cancel := context.WithTimeout(parent, loadPingTimeout())
```

If `parent` is the healthCtx (from `context.Background()` in host.go:52), this is fine. But the code path passes `ctx` from `startHealthLoop`'s first parameter, which is the `healthCtx`. Confirmed correct — no issue, just documenting the flow for future maintainers.

---

## What's Done Well

1. **Clean separation of concerns**: `proto.go` (codec), `handshake.go` (protocol), `health.go` (liveness), `grants.go` (authz), `discovery.go` (resolution) are each focused and testable independently.

2. **Race-safe design**: `sync.RWMutex` on host, `sync.Mutex` on codec, `atomic` for test timing overrides — all correct patterns. Passes `-race -count=3`.

3. **TestMain build harness**: Building the hello-ext binary in `TestMain` and setting `GERT_EXT_DIR` is an elegant pattern that gives true integration coverage without external setup.

4. **Defensive nil checks**: `performHandshake` checks `proc == nil || proc.codec == nil`; `registerContributions` checks `res == nil`; `toolContrib.Def == nil` is handled. This prevents panics in degraded states.

5. **FakeExtensionHost for engine tests**: Clean test double in `pkg/testutil` with proper interface satisfaction. Demonstrates the injectable dependency pattern works.

6. **CapabilitySet as `map[string]struct{}`**: More idiomatic than `map[string]bool` from the design doc. `struct{}` uses zero bytes.

---

## Verification Summary

| Check | Result |
|-------|--------|
| Tests reviewed | ✅ 36 test functions across 8 files; all exercise real subprocess |
| Build verified | ✅ Pre-validated (`go build ./...` clean) |
| Security checked | ✅ No secrets in code; extension paths validated; capabilities gate actions; no command injection (entrypoint resolved from manifest, not user input) |
| Race detection | ✅ `-race -count=3` passed per validation report |
| Barbara's gap (Register) | ✅ `Register(def ToolDef) error` exists on both `ToolRegistry` interface and `MapRegistry` |

---

## Summary

Phase 7 delivers a working Extension Host with correct isolation, protocol, and contribution registration. The architecture is clean and maintainable. Two integration bugs (missing `Shutdown()` call, `nil` manifest passed to `Load()`) need fixing before Phase 8 can build on top of extension lifecycle guarantees. State enum naming should be aligned to spec for traceability. Total score 80/100 — approved with follow-ups.
