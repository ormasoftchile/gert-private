# Phase 8 Review — Input Provider Framework

**Reviewer:** Ken (Software Architect)  
**Date:** 2026-04-22  
**Implementor:** Brian  
**Test Status:** 26 tests pass (`go test ./... -race -count=3`)

---

## Scores

| # | Criterion | Score | Notes |
|---|-----------|-------|-------|
| 1 | Spec compliance | 7 | Core framework delivered correctly. Simplified from full spec (no FileResolver, WorkspaceResolver, no external JSON-RPC providers, no pre-flight `ResolveAll` in engine). Acceptable as Phase 8 MVP — advanced providers are Priority 2-3. |
| 2 | Interface correctness | 9 | `InputProvider` and `InputRegistry` in `pkg/input/` are clean, minimal, and correct. `PromptProvider` cleanly separated. Leaf package has zero internal imports. Single-method `Provide()` is a better ISP design than the multi-method `InputResolver` I specified. |
| 3 | Provider correctness | 8 | EnvProvider, StaticProvider, VaultProvider (env-stub), PromptProvider all behave correctly. Context cancellation checked. Missing: FileResolver, WorkspaceResolver (deferred). |
| 4 | Chain/registry logic | 9 | Priority ordering via `sort.SliceStable` correct. Chain walks left-to-right, first non-nil wins. `ErrNoProvider` sentinel on full miss. `SetDefault` fallback works. Simpler than prefix-matching but equally correct for current provider set. |
| 5 | Engine integration | 8 | Default chain (env→vault→prompt) constructed in `Start()`. `PromptProvider` wired to choice/decision/collector. `InputProvider` wired to prompt executor. Missing: pre-flight `ResolveAll` from `from:` bindings — inputs with `from:` are not auto-resolved yet. |
| 6 | Phase 7 housekeeping | 10 | `shutdownExtensions()` deferred in `completeRun`, `failRun`, Cancel, and called in SIGINT handler. `ExtensionHost.Load()` called with real `ProjectManifest` built from `plan.Metadata.Extensions`. Both fixes verified. |
| 7 | Test coverage | 9 | 26 tests, all pass with `-race`. Covers: happy path, miss (nil return), error propagation, chain fallthrough, ErrNoProvider sentinel, enum validation, context cancellation, sensitive propagation, metadata override, priority ordering. |
| 8 | Sensitive data handling | 8 | `Sensitive` flag propagates from `InputRequest` → `InputResponse` via `ensureResponse()` helper. No value logging observed in any provider. VaultProvider correctly marks all resolved values as sensitive-by-convention (env-prefixed stub). |
| 9 | Replay metadata | 8 | `InputResponse.Source` populated with provider name. `CacheKey` set to `StepID:VarName` (stable, deterministic). Sufficient for Phase 12 replay construction via `StaticProvider`. Missing: `Timestamp` and `Metadata` map from full design (minor). |
| 10 | Code quality | 9 | No races (mutex in registry, t.Setenv in tests). Context cancellation checked first in all providers. Clean error handling with sentinel. Compile-time interface check on `TerminalInputProvider`. `ensureResponse` DRYs metadata population. |

---

## Total: 8.5/10 — APPROVED ✅

---

## Summary

Brian delivered a **pragmatically simplified** version of the Phase 8 design that captures the essential architecture while deferring complexity that isn't yet needed. The key architectural decisions are sound:

1. **Single-method `Provide()`** instead of my multi-method `InputResolver` — better ISP, easier to implement new providers.
2. **Priority-based registry** instead of prefix-matching — simpler, and correct for the current provider set (env/vault/static/prompt all resolve any variable name).
3. **Clean two-interface split** (`InputProvider` for resolution, `PromptProvider` for interactive) — matches design intent D1/D7.
4. **Phase 7 housekeeping fully resolved** — Shutdown deferred in all terminal paths, ProjectManifest properly constructed.

---

## Non-blocking Recommendations

### R1: Pre-flight resolution integration (next phase)

The engine's `Start()` method constructs the default chain but doesn't yet call `ResolveAll()` for inputs with `from:` bindings. This means runbooks declaring `from: env.TOKEN` won't have values auto-resolved before step execution. Wire this in Phase 9 or a housekeeping pass:

```go
// In Start(), after chain construction:
if plan.Inputs != nil {
    for name, inp := range plan.Inputs {
        if inp.From != "" {
            resp, err := e.cfg.InputProvider.Provide(ctx, input.InputRequest{VarName: name, Metadata: map[string]string{"from": inp.From}})
            // merge into initial vars...
        }
    }
}
```

### R2: Add FileResolver and WorkspaceResolver

These are spec-required (§14.4) but not blocking for the framework skeleton. Track as Phase 8b or Phase 10 deliverable.

### R3: Compile-time interface guard on all providers

`TerminalInputProvider` has a guard. Add explicit guards to all internal providers:

```go
var _ inputpkg.InputProvider = (*EnvProvider)(nil)
var _ inputpkg.InputProvider = (*StaticProvider)(nil)
var _ inputpkg.InputProvider = (*VaultProvider)(nil)
var _ inputpkg.InputProvider = (*PromptProvider)(nil)
var _ inputpkg.InputProvider = (*ChainProvider)(nil)
```

### R4: EnvProvider — distinguish "not set" from "empty"

Currently `os.Getenv` returns "" for both unset and empty vars. Use `os.LookupEnv` to distinguish:
```go
val, exists := os.LookupEnv(name)
if !exists { return nil, nil }
```
This matters for `AllowEmpty` semantics.

### R5: Registry thread-safety under concurrent Resolve

`defaultRegistry.Resolve()` copies the slice under RLock — correct. But `Register()` appends and sorts without verifying no concurrent Resolve is mid-iteration on the old slice. The copy-on-read pattern handles this correctly, but add a comment documenting the safety guarantee.

---

## Blocking Issues

None. Phase 8 is approved.
