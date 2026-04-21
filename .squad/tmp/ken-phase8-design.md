# Phase 8 — Input Provider Framework

**Author:** Ken (Software Architect)  
**Date:** 2026-04-22  
**Module:** `github.com/ormasoftchile/gert/v2`  
**Spec Reference:** §14 (Input Provider Framework)  
**Depends on:** Phase 5 (Step Executors), Phase 6 (Tool Runtime), Phase 7 (Extension Host)  
**Estimated LOC:** ~2200 (implementation) + ~1800 (tests)

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Existing State Analysis](#existing-state-analysis)
3. [Architecture Decisions (D1–D8)](#architecture-decisions)
4. [Interface Design (pkg/input/)](#interface-design)
5. [Built-in Providers (internal/input/)](#built-in-providers)
6. [Input Resolution Engine](#input-resolution-engine)
7. [Integration with Prompt Step Executor](#integration-with-prompt-step-executor)
8. [Integration with EngineConfig](#integration-with-engineconfig)
9. [Replay / Trace Integration (Preview)](#replay--trace-integration)
10. [Package Layout](#package-layout)
11. [Test Plan (26 tests)](#test-plan)
12. [Phase 7 Housekeeping](#phase-7-housekeeping)
13. [Implementation Sequence](#implementation-sequence)
14. [Open Questions](#open-questions)

---

## Executive Summary

Phase 8 introduces the **Input Provider Framework** — the system that resolves dynamic input values at run time. When a runbook declares an input with a `from:` binding (e.g., `from: env.ACME_API_KEY` or `from: vault.secret/token`), the provider framework:

1. Matches the binding prefix to a registered provider
2. Dispatches a resolution request (batched where possible)
3. Returns the concrete value before the first step that needs it
4. Falls back gracefully (prompt / default / fail) on resolution failure

This phase also extends the existing `InputProvider` interface (currently limited to interactive step prompting) into a **two-tier architecture**:

- **Tier 1 — Input Resolution** (`InputResolver`): resolves `from:` bindings to concrete values (env, file, vault, workspace, external JSON-RPC providers)
- **Tier 2 — Interactive Prompting** (`InputProvider`): handles choice/decision/collector step types (already implemented in Phase 5)

Both tiers share a common `InputRegistry` that manages provider lifecycles and dispatches requests.

---

## Existing State Analysis

### What exists today

| Component | Location | Status |
|-----------|----------|--------|
| `InputProvider` interface | `pkg/input/input.go` | ✅ 3 methods: `PromptChoice`, `PromptDecision`, `PromptForm` |
| `TerminalInputProvider` | `internal/input/terminal.go` | ✅ Full implementation for stdin/stdout |
| `FakeInputProvider` | `pkg/testutil/fake_input_provider.go` | ✅ Thread-safe test double |
| `EngineConfig.InputProvider` | `pkg/engine/engine.go:81` | ✅ Already wired as optional field |
| Choice/Decision/Collector executors | `internal/executor/` | ✅ All use `input.InputProvider` |
| `schema.Input.From` field | `pkg/schema/runbook.go:48` | ✅ `From string` field parsed from YAML |
| Input resolution at plan time | — | ❌ Not implemented |
| `from:` binding dispatch | — | ❌ Not implemented |
| Provider chaining | — | ❌ Not implemented |
| Env/file/vault/workspace providers | — | ❌ Not implemented |
| Provider capability advertisement | — | ❌ Not implemented |

### Key insight

The existing `InputProvider` interface handles **interactive prompting** (choice/decision/collector steps). Phase 8 adds **input resolution** (resolving `from:` bindings). These are related but distinct concerns. The design must preserve backward compatibility while adding the resolution layer.

---

## Architecture Decisions

### D1: Interface Shape — Unified vs. Split

**Decision:** SPLIT into `InputResolver` (value resolution) and `InputProvider` (interactive prompts).

**Rationale:** The existing `InputProvider` with its 3 methods (`PromptChoice`, `PromptDecision`, `PromptForm`) is already deployed across 3 executors and a test double. Adding `Resolve(binding string) (string, error)` to this interface would violate the Interface Segregation Principle — env/file/vault providers don't prompt, and the terminal provider doesn't resolve `from:` bindings.

**Concrete shape:**
```go
// InputResolver resolves from: bindings to concrete values.
type InputResolver interface {
    Resolve(ctx context.Context, req ResolveRequest) (*ResolveResponse, error)
}

// InputProvider handles interactive step types (unchanged from Phase 5).
type InputProvider interface {
    PromptChoice(ctx context.Context, req ChoiceRequest) (*ChoiceResponse, error)
    PromptDecision(ctx context.Context, req DecisionRequest) (*DecisionResponse, error)
    PromptForm(ctx context.Context, req FormRequest) (*FormResponse, error)
}
```

The `InputRegistry` composes both: it IS an `InputResolver` AND holds a reference to the active `InputProvider`.

---

### D2: Registry vs. Single-Provider EngineConfig Field

**Decision:** Keep `EngineConfig.InputProvider` for interactive prompts (backward compat). ADD `EngineConfig.InputRegistry` for resolution.

**Rationale:** The engine already has `InputProvider input.InputProvider` at line 81 of `pkg/engine/engine.go`. Three executors depend on it. We preserve this field for interactive prompts and add a new `InputRegistry` field for `from:` binding resolution. The registry internally delegates to the appropriate resolver.

```go
// In EngineConfig:
InputProvider input.InputProvider    // Interactive prompts (choice/decision/collector)
InputRegistry input.InputRegistry    // from: binding resolution (env/file/vault/chain)
```

If `InputRegistry` is nil, `from:` bindings fail at plan time with a clear error.

---

### D3: Caching Strategy

**Decision:** Per-run caching with scope override per provider.

**Rationale:** The spec defines three cache scopes: `run`, `step`, `none`. Default is `run` (resolved value reused for all subsequent steps referencing the same binding). The cache is an in-memory map keyed by `(runID, bindingString)`.

**Implementation:**
- The `InputRegistry` owns a `*resolveCache` instance per run.
- Cache entries are never persisted — they live in the `Run` struct's variable map after first resolution.
- External providers can override via `cache.scope` in their `.provider.yaml`.
- Built-in providers (env, file, workspace) always use `scope: run` — env vars don't change mid-run.

---

### D4: Secret Masking in Logs

**Decision:** Mark resolver responses with `Sensitive bool`. The engine's trace writer redacts sensitive values.

**Rationale:** Vault-sourced inputs and `env.` bindings matching common patterns (`*KEY*`, `*SECRET*`, `*TOKEN*`, `*PASSWORD*`) are automatically marked sensitive. The `ResolveResponse` carries `Sensitive: true`, and the trace writer records `"value": "***REDACTED***"` instead of the actual value. This integrates with Phase 4's existing redaction machinery.

**Implementation:** `ResolveResponse.Sensitive` is a hint. The engine's governance redactor can also independently match values. Both mechanisms are defense-in-depth.

---

### D5: Validation — Where, When, By Whom

**Decision:** Validation occurs at two points:
1. **Plan time** (static): schema type checking (string/number/bool against `Input.Type`)
2. **Resolution time** (runtime): provider-returned values are validated against the input schema before being merged into the variable map.

**Rationale:** Fail-fast is essential. If an env var should be a number but contains "abc", we fail before any steps execute — not mid-run after steps have made irreversible changes.

The `InputRegistry.ResolveAll()` method performs validation after all bindings are resolved. Validation failures respect the input's `fallback:` strategy (prompt / default / fail).

---

### D6: Default Provider Chain Order

**Decision:** When no explicit `from:` is declared, inputs are resolved in this order:
1. CLI `--var` flags (highest priority, injected as static values)
2. Workspace config (`.gert/config.yaml`)
3. Environment variables (if `Input.Type` is `string` and input name maps to a reasonable env var convention)
4. Interactive prompt (lowest priority, only in `real`/`debug` modes)

When an explicit `from:` chain is declared (e.g., `from: [env.TOKEN, vault.secret/token, prompt]`), the chain is evaluated left-to-right, first non-empty wins.

**Rationale:** Matches spec §14 section "Provider Composition". The default chain is conservative — it only auto-resolves from env/workspace for inputs that explicitly declare `from:`. Undeclared inputs with no `from:` and no `--var` always prompt.

---

### D7: Prompt Step Integration

**Decision:** Interactive step executors (choice/decision/collector) continue to receive `input.InputProvider` directly. They do NOT go through `InputRegistry` for their interactive prompting.

**Rationale:** The prompt executors need the full `InputProvider` interface (choice/decision/form). They don't resolve `from:` bindings — they collect real-time interactive input. The registry is for pre-flight resolution; the executors are for in-flight interaction. These are different lifecycle moments.

**Binding:** `InputRegistry` exposes its `InputProvider` reference so the engine can wire it to executors:
```go
registry.Provider() input.InputProvider  // returns the active interactive provider
```

This means: if you set up an `InputRegistry`, you don't also need to separately configure `EngineConfig.InputProvider` — the registry provides it. But for backward compat, if only `InputProvider` is set (no registry), executors still work as today.

---

### D8: Replay Metadata Design

**Decision:** `ResolveResponse` carries enough metadata for deterministic replay without coupling to Phase 12.

**Fields:**
```go
type ResolveResponse struct {
    Value     string            // The resolved value
    Source    string            // Provider ID that resolved it (e.g., "env", "vault", "prompt")
    Binding   string            // Original from: binding string
    CacheKey  string            // Stable cache key for replay: "(binding)" or "(provider:binding)"
    Sensitive bool              // If true, value is redacted in traces
    Metadata  map[string]string // Provider-specific metadata (e.g., env var name, vault path)
    Timestamp time.Time         // When resolution occurred
}
```

**Replay contract:** Phase 12 records a `map[string]ResolveResponse` in the run trace. On replay, these are loaded into a `StaticResolver` that returns recorded values verbatim. The `CacheKey` ensures deterministic replay even if provider ordering or availability changes.

---

## Interface Design (pkg/input/)

### File: `pkg/input/resolver.go` (NEW)

```go
package input

import (
    "context"
    "time"
)

// InputResolver resolves from: bindings to concrete values.
// Implementations: EnvResolver, FileResolver, VaultResolver, WorkspaceResolver,
// ChainResolver, StaticResolver, ExternalResolver (JSON-RPC).
type InputResolver interface {
    // Resolve resolves a single binding to a concrete value.
    // Returns ErrUnresolved if the provider cannot resolve the binding.
    Resolve(ctx context.Context, req ResolveRequest) (*ResolveResponse, error)

    // Prefix returns the binding prefix this resolver handles (e.g., "env.", "vault.").
    // Empty string means "catch-all" (used by ChainResolver and StaticResolver).
    Prefix() string
}

// ResolveRequest describes what the engine needs resolved.
type ResolveRequest struct {
    // Binding is the full from: value (e.g., "env.ACME_API_KEY", "vault.secret/token").
    Binding string

    // InputName is the declared input name in the runbook's inputs: block.
    InputName string

    // InputType is the declared type (string, number, boolean, etc.) for validation.
    InputType string

    // RunID is the current run identifier (for audit and context).
    RunID string

    // StepID is the step requesting resolution (empty for pre-flight).
    StepID string

    // Mode is the execution mode (real, dry-run, replay, debug).
    Mode string

    // Vars is the current variable scope (for template evaluation in bindings).
    Vars map[string]any
}

// ResolveResponse holds the resolved value and metadata.
type ResolveResponse struct {
    // Value is the resolved concrete value as a string.
    // The engine performs type coercion based on Input.Type.
    Value string

    // Source identifies which provider resolved this value.
    Source string

    // Binding is the original from: binding string (echo for tracing).
    Binding string

    // CacheKey is a stable key for caching and replay.
    // Typically "(source:binding)" — e.g., "env:ACME_API_KEY".
    CacheKey string

    // Sensitive indicates the value should be redacted in traces/logs.
    Sensitive bool

    // Metadata carries provider-specific context (for audit/replay).
    Metadata map[string]string

    // Timestamp records when resolution completed.
    Timestamp time.Time
}

// ErrUnresolved is returned when a resolver cannot resolve a binding.
// This is not necessarily an error — the registry may try the next resolver in chain.
type ErrUnresolved struct {
    Binding string
    Reason  string
}

func (e *ErrUnresolved) Error() string {
    if e.Reason != "" {
        return "input: unresolved binding " + e.Binding + ": " + e.Reason
    }
    return "input: unresolved binding " + e.Binding
}

// IsUnresolved returns true if the error is an ErrUnresolved.
func IsUnresolved(err error) bool {
    _, ok := err.(*ErrUnresolved)
    return ok
}
```

### File: `pkg/input/registry.go` (NEW)

```go
package input

import "context"

// InputRegistry manages multiple resolvers and an interactive provider.
// It dispatches from: bindings to the correct resolver by prefix matching,
// and provides the active InputProvider for interactive step types.
type InputRegistry interface {
    // Register adds a resolver to the registry.
    // Resolvers are matched by longest prefix first.
    Register(resolver InputResolver) error

    // ResolveAll resolves all from: bindings for a run's declared inputs.
    // Returns a map of input name -> ResolveResponse.
    // Unresolved bindings are handled per each input's fallback strategy.
    ResolveAll(ctx context.Context, inputs map[string]*InputDecl, runCtx RunContext) (map[string]*ResolveResponse, error)

    // Resolve resolves a single binding.
    // Dispatches to the registered resolver with the longest matching prefix.
    Resolve(ctx context.Context, req ResolveRequest) (*ResolveResponse, error)

    // Provider returns the active InputProvider for interactive prompts.
    // Returns nil if no interactive provider is configured.
    Provider() InputProvider

    // SetProvider configures the active interactive provider.
    SetProvider(p InputProvider)

    // Shutdown gracefully shuts down all resolver processes.
    Shutdown(ctx context.Context) error
}

// InputDecl represents a runbook input declaration for resolution.
// This is a resolution-focused view of schema.Input.
type InputDecl struct {
    Name        string
    Type        string
    Required    bool
    Description string
    Default     any
    From        any    // string or []string (chain)
    Fallback    string // "prompt" | "default" | "fail"
    AllowEmpty  bool
}

// RunContext provides execution context to resolvers.
type RunContext struct {
    RunID  string
    Mode   string // "real" | "dry-run" | "replay" | "debug"
    Actor  string
    Vars   map[string]any
}

// FallbackPrompt is the fallback strategy constant for interactive prompt.
const FallbackPrompt = "prompt"

// FallbackDefault uses the input's declared default value.
const FallbackDefault = "default"

// FallbackFail aborts the run immediately.
const FallbackFail = "fail"
```

### File: `pkg/input/input.go` (EXTENDED — add to existing)

The existing file is preserved intact. We add one new method to the existing `InputProvider` interface via a **separate optional interface** (to avoid breaking existing implementations):

```go
// InputValueProvider is an optional extension of InputProvider that can also
// resolve simple text inputs (for use as the "prompt" fallback in resolution).
// Implementations that want to be used as resolution fallbacks should implement this.
type InputValueProvider interface {
    InputProvider

    // PromptValue prompts for a single text value (used as fallback during resolution).
    PromptValue(ctx context.Context, req ValueRequest) (*ValueResponse, error)
}

// ValueRequest describes a simple value prompt (fallback during resolution).
type ValueRequest struct {
    InputName   string
    Description string
    Type        string // "string", "number", "boolean"
    Default     string
    Sensitive   bool   // If true, input should be masked (e.g., password entry)
}

// ValueResponse holds the prompted value.
type ValueResponse struct {
    Value string
}
```

---

## Built-in Providers (internal/input/)

### 1. EnvResolver (`internal/input/env.go`)

```go
package input

import (
    "context"
    "os"
    "strings"
    "time"

    inputpkg "github.com/ormasoftchile/gert/v2/pkg/input"
)

// EnvResolver resolves from: bindings prefixed with "env.".
// Example: from: env.ACME_API_KEY → os.Getenv("ACME_API_KEY")
type EnvResolver struct {
    // LookupEnv is the environment lookup function (default: os.LookupEnv).
    // Injectable for testing.
    LookupEnv func(key string) (string, bool)
}

// Compile-time interface check.
var _ inputpkg.InputResolver = (*EnvResolver)(nil)

func NewEnvResolver() *EnvResolver {
    return &EnvResolver{LookupEnv: os.LookupEnv}
}

func (r *EnvResolver) Prefix() string { return "env." }

func (r *EnvResolver) Resolve(ctx context.Context, req inputpkg.ResolveRequest) (*inputpkg.ResolveResponse, error) {
    // Strip prefix: "env.ACME_API_KEY" → "ACME_API_KEY"
    envVar := strings.TrimPrefix(req.Binding, "env.")
    if envVar == "" {
        return nil, &inputpkg.ErrUnresolved{Binding: req.Binding, Reason: "empty env var name"}
    }

    lookup := r.LookupEnv
    if lookup == nil {
        lookup = os.LookupEnv
    }

    val, exists := lookup(envVar)
    if !exists {
        return nil, &inputpkg.ErrUnresolved{Binding: req.Binding, Reason: "env var not set"}
    }

    return &inputpkg.ResolveResponse{
        Value:     val,
        Source:    "env",
        Binding:   req.Binding,
        CacheKey:  "env:" + envVar,
        Sensitive: isSensitiveName(envVar),
        Metadata:  map[string]string{"env_var": envVar},
        Timestamp: time.Now(),
    }, nil
}

// isSensitiveName heuristically determines if an env var name suggests a secret.
func isSensitiveName(name string) bool {
    upper := strings.ToUpper(name)
    patterns := []string{"KEY", "SECRET", "TOKEN", "PASSWORD", "CREDENTIAL", "PRIVATE"}
    for _, p := range patterns {
        if strings.Contains(upper, p) {
            return true
        }
    }
    return false
}
```

### 2. StaticResolver (`internal/input/static.go`)

```go
package input

// StaticResolver resolves from a pre-seeded map.
// Used for: --var CLI flags, test fixtures, replay scenarios.
type StaticResolver struct {
    values map[string]string
}

var _ inputpkg.InputResolver = (*StaticResolver)(nil)

func NewStaticResolver(values map[string]string) *StaticResolver {
    m := make(map[string]string, len(values))
    for k, v := range values {
        m[k] = v
    }
    return &StaticResolver{values: m}
}

func (r *StaticResolver) Prefix() string { return "" } // catch-all

func (r *StaticResolver) Resolve(ctx context.Context, req inputpkg.ResolveRequest) (*inputpkg.ResolveResponse, error) {
    // Look up by input name first, then by full binding string.
    val, ok := r.values[req.InputName]
    if !ok {
        val, ok = r.values[req.Binding]
    }
    if !ok {
        return nil, &inputpkg.ErrUnresolved{Binding: req.Binding, Reason: "not in static map"}
    }
    return &inputpkg.ResolveResponse{
        Value:     val,
        Source:    "static",
        Binding:   req.Binding,
        CacheKey:  "static:" + req.InputName,
        Sensitive: false,
        Metadata:  map[string]string{"input_name": req.InputName},
        Timestamp: time.Now(),
    }, nil
}

// Set adds or overrides a value in the static map.
func (r *StaticResolver) Set(name, value string) {
    r.values[name] = value
}
```

### 3. PromptResolver (`internal/input/prompt_resolver.go`)

```go
package input

// PromptResolver resolves inputs by interactively prompting the user.
// It wraps an InputValueProvider and uses it for fallback resolution.
// This is used when: from: prompt, or as the default fallback.
type PromptResolver struct {
    provider inputpkg.InputValueProvider
}

var _ inputpkg.InputResolver = (*PromptResolver)(nil)

func NewPromptResolver(p inputpkg.InputValueProvider) *PromptResolver {
    return &PromptResolver{provider: p}
}

func (r *PromptResolver) Prefix() string { return "prompt" }

func (r *PromptResolver) Resolve(ctx context.Context, req inputpkg.ResolveRequest) (*inputpkg.ResolveResponse, error) {
    if r.provider == nil {
        return nil, &inputpkg.ErrUnresolved{Binding: req.Binding, Reason: "no interactive provider configured"}
    }
    // In dry-run/replay modes, prompting is not allowed.
    if req.Mode == "dry-run" || req.Mode == "replay" {
        return nil, &inputpkg.ErrUnresolved{Binding: req.Binding, Reason: "interactive prompt not available in " + req.Mode + " mode"}
    }

    resp, err := r.provider.PromptValue(ctx, inputpkg.ValueRequest{
        InputName:   req.InputName,
        Description: "", // filled by registry from InputDecl
        Type:        req.InputType,
        Sensitive:   false,
    })
    if err != nil {
        return nil, err
    }
    return &inputpkg.ResolveResponse{
        Value:     resp.Value,
        Source:    "prompt",
        Binding:   req.Binding,
        CacheKey:  "prompt:" + req.InputName,
        Sensitive: false,
        Timestamp: time.Now(),
    }, nil
}
```

### 4. VaultResolver (`internal/input/vault.go`)

```go
package input

// VaultResolver is a stub resolver for vault-like secret stores.
// In v2.0, this provides a compile-compatible placeholder.
// Real vault integration is planned for Phase 15 (Secret Management).
//
// Usage: from: vault.secret/path/to/key
type VaultResolver struct {
    // Client is the vault client interface.
    // In v2.0, this is always nil (stub).
    Client VaultClient
}

// VaultClient is the interface a real vault implementation must satisfy.
type VaultClient interface {
    Read(ctx context.Context, path string) (string, error)
}

var _ inputpkg.InputResolver = (*VaultResolver)(nil)

func NewVaultResolver(client VaultClient) *VaultResolver {
    return &VaultResolver{Client: client}
}

func (r *VaultResolver) Prefix() string { return "vault." }

func (r *VaultResolver) Resolve(ctx context.Context, req inputpkg.ResolveRequest) (*inputpkg.ResolveResponse, error) {
    if r.Client == nil {
        return nil, &inputpkg.ErrUnresolved{
            Binding: req.Binding,
            Reason:  "vault client not configured (stub: vault integration planned for Phase 15)",
        }
    }
    path := strings.TrimPrefix(req.Binding, "vault.")
    val, err := r.Client.Read(ctx, path)
    if err != nil {
        return nil, &inputpkg.ErrUnresolved{Binding: req.Binding, Reason: err.Error()}
    }
    return &inputpkg.ResolveResponse{
        Value:     val,
        Source:    "vault",
        Binding:   req.Binding,
        CacheKey:  "vault:" + path,
        Sensitive: true, // vault values are always sensitive
        Metadata:  map[string]string{"vault_path": path},
        Timestamp: time.Now(),
    }, nil
}
```

### 5. ChainResolver (`internal/input/chain.go`)

```go
package input

// ChainResolver tries resolvers in order until one succeeds.
// Used to implement from: [env.TOKEN, vault.secret/token, prompt] chains.
type ChainResolver struct {
    resolvers []inputpkg.InputResolver
}

var _ inputpkg.InputResolver = (*ChainResolver)(nil)

func NewChainResolver(resolvers ...inputpkg.InputResolver) *ChainResolver {
    return &ChainResolver{resolvers: resolvers}
}

func (r *ChainResolver) Prefix() string { return "" } // catch-all

func (r *ChainResolver) Resolve(ctx context.Context, req inputpkg.ResolveRequest) (*inputpkg.ResolveResponse, error) {
    var lastErr error
    for _, resolver := range r.resolvers {
        resp, err := resolver.Resolve(ctx, req)
        if err == nil {
            return resp, nil
        }
        if !inputpkg.IsUnresolved(err) {
            // Hard error (context cancelled, etc.) — don't continue chain.
            return nil, err
        }
        lastErr = err
    }
    if lastErr != nil {
        return nil, lastErr
    }
    return nil, &inputpkg.ErrUnresolved{Binding: req.Binding, Reason: "all chain resolvers failed"}
}
```

### 6. FileResolver (`internal/input/file.go`)

```go
package input

// FileResolver resolves from: bindings prefixed with "file.".
// Example: from: file./etc/config.json#/database/host
// Supports optional JSON Pointer fragment for structured files.
type FileResolver struct {
    // ReadFile is injectable for testing (default: os.ReadFile).
    ReadFile func(path string) ([]byte, error)
}

var _ inputpkg.InputResolver = (*FileResolver)(nil)

func NewFileResolver() *FileResolver {
    return &FileResolver{ReadFile: os.ReadFile}
}

func (r *FileResolver) Prefix() string { return "file." }

func (r *FileResolver) Resolve(ctx context.Context, req inputpkg.ResolveRequest) (*inputpkg.ResolveResponse, error) {
    raw := strings.TrimPrefix(req.Binding, "file.")
    // Split path and optional JSON pointer fragment.
    path, pointer := splitFragment(raw)
    // Expand ~ to home directory.
    path = expandHome(path)

    data, err := r.ReadFile(path)
    if err != nil {
        return nil, &inputpkg.ErrUnresolved{Binding: req.Binding, Reason: err.Error()}
    }

    val := string(data)
    if pointer != "" {
        val, err = extractJSONPointer(data, pointer)
        if err != nil {
            return nil, &inputpkg.ErrUnresolved{Binding: req.Binding, Reason: "JSON pointer extraction failed: " + err.Error()}
        }
    }

    return &inputpkg.ResolveResponse{
        Value:     strings.TrimSpace(val),
        Source:    "file",
        Binding:   req.Binding,
        CacheKey:  "file:" + raw,
        Sensitive: false,
        Metadata:  map[string]string{"path": path, "pointer": pointer},
        Timestamp: time.Now(),
    }, nil
}
```

### 7. WorkspaceResolver (`internal/input/workspace.go`)

```go
package input

// WorkspaceResolver resolves from: bindings prefixed with "workspace.".
// Reads values from .gert/config.yaml using dot-notation paths.
// Example: from: workspace.defaults.region
type WorkspaceResolver struct {
    // ConfigPath is the path to .gert/config.yaml.
    ConfigPath string
    // cache holds the parsed config (loaded lazily on first resolve).
    config map[string]any
    once   sync.Once
}

var _ inputpkg.InputResolver = (*WorkspaceResolver)(nil)

func NewWorkspaceResolver(configPath string) *WorkspaceResolver {
    return &WorkspaceResolver{ConfigPath: configPath}
}

func (r *WorkspaceResolver) Prefix() string { return "workspace." }

func (r *WorkspaceResolver) Resolve(ctx context.Context, req inputpkg.ResolveRequest) (*inputpkg.ResolveResponse, error) {
    r.once.Do(func() { r.config = r.loadConfig() })

    key := strings.TrimPrefix(req.Binding, "workspace.")
    val, ok := dotGet(r.config, key)
    if !ok {
        return nil, &inputpkg.ErrUnresolved{Binding: req.Binding, Reason: "key not found in workspace config"}
    }

    return &inputpkg.ResolveResponse{
        Value:     fmt.Sprint(val),
        Source:    "workspace",
        Binding:   req.Binding,
        CacheKey:  "workspace:" + key,
        Sensitive: false,
        Metadata:  map[string]string{"config_key": key},
        Timestamp: time.Now(),
    }, nil
}
```

### 8. Registry Implementation (`internal/input/registry.go`)

```go
package input

// Registry is the concrete InputRegistry implementation.
// It manages resolvers (prefix-matched) and an interactive provider.
type Registry struct {
    resolvers []inputpkg.InputResolver // ordered: longest prefix first
    provider  inputpkg.InputProvider
    mu        sync.RWMutex
}

var _ inputpkg.InputRegistry = (*Registry)(nil)

func NewRegistry() *Registry {
    return &Registry{}
}

func (reg *Registry) Register(resolver inputpkg.InputResolver) error {
    reg.mu.Lock()
    defer reg.mu.Unlock()
    reg.resolvers = append(reg.resolvers, resolver)
    // Sort by prefix length descending (longest match wins).
    sort.Slice(reg.resolvers, func(i, j int) bool {
        return len(reg.resolvers[i].Prefix()) > len(reg.resolvers[j].Prefix())
    })
    return nil
}

func (reg *Registry) Resolve(ctx context.Context, req inputpkg.ResolveRequest) (*inputpkg.ResolveResponse, error) {
    reg.mu.RLock()
    defer reg.mu.RUnlock()

    for _, resolver := range reg.resolvers {
        prefix := resolver.Prefix()
        if prefix == "" || strings.HasPrefix(req.Binding, prefix) {
            resp, err := resolver.Resolve(ctx, req)
            if err == nil {
                return resp, nil
            }
            if !inputpkg.IsUnresolved(err) {
                return nil, err // hard error
            }
            // If prefix matched but resolution failed, don't try other prefix resolvers.
            if prefix != "" {
                return nil, err
            }
            // catch-all resolvers: try next
        }
    }
    return nil, &inputpkg.ErrUnresolved{Binding: req.Binding, Reason: "no matching resolver"}
}

func (reg *Registry) ResolveAll(ctx context.Context, inputs map[string]*inputpkg.InputDecl, runCtx inputpkg.RunContext) (map[string]*inputpkg.ResolveResponse, error) {
    results := make(map[string]*inputpkg.ResolveResponse, len(inputs))
    var failures []string

    for name, decl := range inputs {
        if decl.From == nil {
            continue // no from: binding — skip pre-flight resolution
        }

        bindings := normalizeFromField(decl.From) // string → []string{s}

        var resolved *inputpkg.ResolveResponse
        var lastErr error

        for _, binding := range bindings {
            req := inputpkg.ResolveRequest{
                Binding:   binding,
                InputName: name,
                InputType: decl.Type,
                RunID:     runCtx.RunID,
                Mode:      runCtx.Mode,
                Vars:      runCtx.Vars,
            }
            resp, err := reg.Resolve(ctx, req)
            if err == nil && (resp.Value != "" || decl.AllowEmpty) {
                resolved = resp
                break
            }
            lastErr = err
        }

        if resolved != nil {
            results[name] = resolved
            continue
        }

        // Apply fallback strategy.
        switch decl.Fallback {
        case inputpkg.FallbackDefault:
            if decl.Default != nil {
                results[name] = &inputpkg.ResolveResponse{
                    Value:   fmt.Sprint(decl.Default),
                    Source:  "default",
                    Binding: fmt.Sprint(decl.From),
                    CacheKey: "default:" + name,
                }
            }
        case inputpkg.FallbackFail:
            failures = append(failures, fmt.Sprintf("input %q: %v", name, lastErr))
        case inputpkg.FallbackPrompt, "":
            // Attempt interactive prompt fallback.
            if reg.provider != nil {
                if vp, ok := reg.provider.(inputpkg.InputValueProvider); ok {
                    resp, err := vp.PromptValue(ctx, inputpkg.ValueRequest{
                        InputName:   name,
                        Description: decl.Description,
                        Type:        decl.Type,
                    })
                    if err == nil {
                        results[name] = &inputpkg.ResolveResponse{
                            Value:   resp.Value,
                            Source:  "prompt",
                            Binding: fmt.Sprint(decl.From),
                            CacheKey: "prompt:" + name,
                        }
                        continue
                    }
                }
            }
            if decl.Required {
                failures = append(failures, fmt.Sprintf("input %q: unresolved (no interactive provider)", name))
            }
        }
    }

    if len(failures) > 0 {
        return results, &ResolutionError{Failures: failures}
    }
    return results, nil
}

func (reg *Registry) Provider() inputpkg.InputProvider {
    reg.mu.RLock()
    defer reg.mu.RUnlock()
    return reg.provider
}

func (reg *Registry) SetProvider(p inputpkg.InputProvider) {
    reg.mu.Lock()
    reg.provider = p
    reg.mu.Unlock()
}

func (reg *Registry) Shutdown(ctx context.Context) error {
    // No-op for built-in resolvers. External resolvers (Phase 15) will need process shutdown.
    return nil
}

// ResolutionError is returned when one or more inputs cannot be resolved.
type ResolutionError struct {
    Failures []string
}

func (e *ResolutionError) Error() string {
    return "input resolution failed: " + strings.Join(e.Failures, "; ")
}
```

---

## Input Resolution Engine

### Pre-flight Resolution Flow

Resolution happens **before step execution begins**, during the engine's `Start()` method:

```
Engine.Start()
  ├── ExtensionHost.Load()        (Phase 7)
  ├── InputRegistry.ResolveAll()  (Phase 8 — NEW)
  │     ├── For each input with from: binding
  │     │     ├── Prefix match → dispatch to resolver
  │     │     ├── On success: merge into run.Vars
  │     │     └── On failure: apply fallback (prompt/default/fail)
  │     └── Return resolved map (or error if fallback: fail triggered)
  ├── Merge resolved values into Run.Vars
  └── Return RunHandle
```

### Integration Point (internal/engine/engine.go)

The resolution is added to `Start()` after extension loading but before the RunHandle is returned:

```go
// In impl.Start() — after ExtensionHost.Load(), before RunHandle creation:
if e.cfg.InputRegistry != nil && plan.Inputs != nil {
    runCtx := input.RunContext{
        RunID: runID,
        Mode:  string(opts.Mode),
        Actor: opts.Actor,
        Vars:  plan.InitialVars,
    }
    resolved, err := e.cfg.InputRegistry.ResolveAll(ctx, plan.Inputs, runCtx)
    if err != nil {
        return nil, fmt.Errorf("input resolution failed: %w", err)
    }
    // Merge resolved values into the run's initial variable map.
    for name, resp := range resolved {
        plan.InitialVars[name] = resp.Value
    }
}
```

### ExecutionPlan Extension

Add to `pkg/engine/plan.go`:

```go
type ExecutionPlan struct {
    // ... existing fields ...

    // Inputs declares the runbook's input variables with resolution metadata.
    // Populated from schema.Runbook.Inputs during planning.
    Inputs map[string]*input.InputDecl
}
```

The planner (Phase 2) already has access to `schema.Runbook.Inputs`. Phase 8 adds a mapping step:

```go
// In planner: convert schema.Input to input.InputDecl
for name, schemaInput := range runbook.Inputs {
    plan.Inputs[name] = &input.InputDecl{
        Name:        name,
        Type:        schemaInput.Type,
        Required:    schemaInput.Required,
        Description: schemaInput.Description,
        Default:     schemaInput.Default,
        From:        schemaInput.From,
        Fallback:    FallbackPrompt, // default
    }
}
```

---

## Integration with Prompt Step Executor

### Current State

The choice/decision/collector executors already receive `input.InputProvider` via `RegistryConfig.InputProvider` and are wired in `NewDefaultRegistry()`. This works correctly today.

### Phase 8 Change

When `InputRegistry` is configured, the engine derives the `InputProvider` from it:

```go
// In the engine setup / cmd layer:
var provider input.InputProvider
if registry != nil {
    provider = registry.Provider() // registry holds the terminal provider
}
// Fallback: if InputProvider is explicitly set on EngineConfig, use that.
if provider == nil {
    provider = cfg.InputProvider
}
```

The executor `RegistryConfig` continues to receive `input.InputProvider` — no change needed in executor code.

### TerminalInputProvider Enhancement

The existing `TerminalInputProvider` in `internal/input/terminal.go` must implement `InputValueProvider` (the extended interface that adds `PromptValue`):

```go
// Add to terminal.go:
func (t *TerminalInputProvider) PromptValue(ctx context.Context, req input.ValueRequest) (*input.ValueResponse, error) {
    if err := ctx.Err(); err != nil {
        return nil, err
    }
    prompt := req.InputName
    if req.Description != "" {
        prompt += " (" + req.Description + ")"
    }
    if req.Default != "" {
        prompt += " [" + req.Default + "]"
    }
    if _, err := fmt.Fprintf(t.Out, "%s: ", prompt); err != nil {
        return nil, err
    }

    reader := bufio.NewReader(t.In)
    line, err := reader.ReadString('\n')
    if err != nil && !errors.Is(err, io.EOF) {
        return nil, err
    }
    line = strings.TrimSpace(line)
    if line == "" && req.Default != "" {
        line = req.Default
    }
    return &input.ValueResponse{Value: line}, nil
}
```

Compile-time check:
```go
var _ input.InputValueProvider = (*TerminalInputProvider)(nil)
```

---

## Integration with EngineConfig

### Changes to `pkg/engine/engine.go`

```go
// EngineConfig — add one field:

    // InputRegistry resolves from: bindings during pre-flight.
    // Optional: if nil, from: bindings are not resolved (inputs must be
    // supplied via --var or remain at their default values).
    InputRegistry input.InputRegistry
```

The existing `InputProvider` field remains. The two fields serve different purposes:
- `InputProvider`: interactive step execution (choice/decision/collector)
- `InputRegistry`: pre-flight `from:` binding resolution

If both are set, the registry's `Provider()` takes precedence for interactive steps. If only `InputProvider` is set, that's used directly (Phase 5 backward compat).

### Changes to `pkg/engine/plan.go`

```go
// ExecutionPlan — add:

    // Inputs declares input variables with resolution metadata.
    Inputs map[string]*input.InputDecl

    // InitialVars holds the merged variable map (CLI vars + resolved inputs + defaults).
    InitialVars map[string]any
```

---

## Replay / Trace Integration (Preview)

### Design for Phase 12 Compatibility

Phase 12 (Evidence & Replay) will need to:
1. **Record** all resolved inputs with their metadata during a real run
2. **Replay** a recorded run using the exact same input values

Phase 8 enables this by:

1. **ResolveResponse carries full metadata** — `Source`, `CacheKey`, `Timestamp`, and `Metadata` fields provide everything needed to reconstruct resolution history.

2. **StaticResolver enables deterministic replay** — Phase 12 loads the recorded `map[string]ResolveResponse` from the trace file and constructs a `StaticResolver` pre-seeded with those values. The `StaticResolver` has highest priority in the chain, so it shadows all other resolvers.

3. **Trace event for input resolution** — Phase 8 emits a `trace.EventKindInputResolved` event for each resolved input:

```go
// New trace event kind:
EventKindInputResolved EventKind = "input/resolved"

// Payload:
{
    "input_name": "api_token",
    "source":     "env",
    "binding":    "env.ACME_API_TOKEN",
    "cache_key":  "env:ACME_API_TOKEN",
    "sensitive":  true,
    "value":      "***REDACTED***"  // or actual value if not sensitive
}
```

4. **No Phase 12 coupling** — Phase 8 code never imports Phase 12 packages. It simply records enough metadata that Phase 12 can later consume it.

### Replay Scenario

```go
// Phase 12 will do:
recorded := loadInputsFromTrace(traceFile)
static := input.NewStaticResolver(recorded)
registry := input.NewRegistry()
registry.Register(static) // highest priority (catch-all with empty prefix)
// ... register other resolvers as normal ...
// StaticResolver wins for any input that was recorded in the trace.
```

---

## Package Layout

```
v2/
├── pkg/input/
│   ├── input.go              # InputProvider interface (existing, EXTENDED with InputValueProvider)
│   ├── resolver.go           # InputResolver interface, ResolveRequest/Response, ErrUnresolved (NEW)
│   └── registry.go           # InputRegistry interface, InputDecl, RunContext, constants (NEW)
│
├── internal/input/
│   ├── terminal.go           # TerminalInputProvider (existing, ADD PromptValue method)
│   ├── env.go                # EnvResolver (NEW)
│   ├── static.go             # StaticResolver (NEW)
│   ├── file.go               # FileResolver (NEW)
│   ├── workspace.go          # WorkspaceResolver (NEW)
│   ├── vault.go              # VaultResolver stub (NEW)
│   ├── chain.go              # ChainResolver (NEW)
│   ├── prompt_resolver.go    # PromptResolver (NEW)
│   ├── registry.go           # Registry implementation (NEW)
│   ├── helpers.go            # Shared utilities (expandHome, dotGet, splitFragment, etc.)
│   ├── env_test.go           # (NEW)
│   ├── static_test.go        # (NEW)
│   ├── file_test.go          # (NEW)
│   ├── workspace_test.go     # (NEW)
│   ├── vault_test.go         # (NEW)
│   ├── chain_test.go         # (NEW)
│   ├── prompt_resolver_test.go # (NEW)
│   ├── registry_test.go      # (NEW)
│   └── terminal_test.go      # (extend existing if present)
│
├── pkg/testutil/
│   ├── fake_input_provider.go  # (existing — ADD FakeInputValueProvider, FakeResolver)
│   └── fake_input_registry.go  # FakeInputRegistry for engine tests (NEW)
│
├── pkg/engine/
│   ├── engine.go             # Add InputRegistry field to EngineConfig
│   └── plan.go               # Add Inputs and InitialVars to ExecutionPlan
│
└── internal/engine/
    └── engine.go             # Add resolution call in Start(), Shutdown in completeRun/failRun
```

---

## Test Plan (26 Tests)

### Group 1: EnvResolver (4 tests)

| # | Test | Description |
|---|------|-------------|
| 1 | `TestEnvResolver_Found` | Set env var, resolve succeeds, value matches |
| 2 | `TestEnvResolver_NotFound` | Unset env var, returns ErrUnresolved |
| 3 | `TestEnvResolver_EmptyString` | Env var set to "", resolves to "" (valid — AllowEmpty controls acceptance) |
| 4 | `TestEnvResolver_SensitiveDetection` | Var names containing KEY/SECRET/TOKEN are marked Sensitive=true |

### Group 2: StaticResolver (4 tests)

| # | Test | Description |
|---|------|-------------|
| 5 | `TestStaticResolver_FoundByName` | Input name exists in map, resolves |
| 6 | `TestStaticResolver_FoundByBinding` | Binding string exists in map, resolves |
| 7 | `TestStaticResolver_NotFound` | Missing key returns ErrUnresolved |
| 8 | `TestStaticResolver_Override` | Set() overrides existing value |

### Group 3: FileResolver (3 tests)

| # | Test | Description |
|---|------|-------------|
| 9 | `TestFileResolver_PlainFile` | Reads file contents as value |
| 10 | `TestFileResolver_JSONPointer` | Extracts nested JSON field via #/path/to/key |
| 11 | `TestFileResolver_NotFound` | Missing file returns ErrUnresolved |

### Group 4: WorkspaceResolver (3 tests)

| # | Test | Description |
|---|------|-------------|
| 12 | `TestWorkspaceResolver_Found` | Dot-notation key found in config |
| 13 | `TestWorkspaceResolver_NestedKey` | Deep key (a.b.c) resolves correctly |
| 14 | `TestWorkspaceResolver_NotFound` | Missing key returns ErrUnresolved |

### Group 5: ChainResolver (3 tests)

| # | Test | Description |
|---|------|-------------|
| 15 | `TestChainResolver_FirstWins` | First resolver succeeds, second not called |
| 16 | `TestChainResolver_Fallback` | First fails, second succeeds |
| 17 | `TestChainResolver_AllFail` | All resolvers fail, returns last ErrUnresolved |

### Group 6: VaultResolver (2 tests)

| # | Test | Description |
|---|------|-------------|
| 18 | `TestVaultResolver_Stub_NoClient` | Returns ErrUnresolved with "not configured" reason |
| 19 | `TestVaultResolver_WithClient` | Mock client returns value, always marked Sensitive |

### Group 7: PromptResolver (2 tests)

| # | Test | Description |
|---|------|-------------|
| 20 | `TestPromptResolver_Interactive` | Mocked InputValueProvider returns value |
| 21 | `TestPromptResolver_NonInteractiveMode` | In dry-run mode, returns ErrUnresolved |

### Group 8: Registry (4 tests)

| # | Test | Description |
|---|------|-------------|
| 22 | `TestRegistry_PrefixDispatch` | "env." binding dispatches to EnvResolver |
| 23 | `TestRegistry_LongestPrefixWins` | "vault.secret." preferred over "vault." if both registered |
| 24 | `TestRegistry_ResolveAll_Success` | Multiple inputs all resolve, results map correct |
| 25 | `TestRegistry_ResolveAll_FallbackFail` | Required input with fallback:fail causes error |

### Group 9: Engine Integration (1 test)

| # | Test | Description |
|---|------|-------------|
| 26 | `TestEngine_InputResolution_BeforeSteps` | Configured InputRegistry resolves inputs before first step executes; resolved values available in vars |

---

## Phase 7 Housekeeping

These are blocking follow-up items from the Phase 7 review (scored 7/10 on lifecycle management and engine integration). They MUST be completed in Phase 8 before the new resolution flow is wired in.

### H1: Engine Must Call `ExtensionHost.Shutdown()` on Run Completion

**Issue:** Extension child processes leak because neither `completeRun()` nor `failRun()` calls `ExtensionHost.Shutdown()`.

**File:** `v2/internal/engine/engine.go` — lines 538-576

**Fix:**

```go
// In completeRun() — add before cancelFn():
func (h *runHandle) completeRun(ctx context.Context) (*enginepkg.StepResult, error) {
    h.run.Status = enginepkg.RunStatusCompleted
    h.run.CompletedAt = time.Now()
    h.done.Store(true)

    h.emitEventLocked(ctx, trace.EventKindRunCompleted, map[string]any{
        "run_id":      h.run.ID,
        "duration_ms": h.run.CompletedAt.Sub(h.run.StartedAt).Milliseconds(),
    })

    // Phase 8 housekeeping: shut down extension processes.
    if h.engine.cfg.ExtensionHost != nil {
        _ = h.engine.cfg.ExtensionHost.Shutdown(ctx)
    }

    safeClose(h.events, &h.eventsClosed)
    h.cancelFn()
    return nil, io.EOF
}

// In failRun() — add before cancelFn():
func (h *runHandle) failRun(ctx context.Context, stepID string, err error) (*enginepkg.StepResult, error) {
    h.run.Status = enginepkg.RunStatusFailed
    h.run.CompletedAt = time.Now()
    h.run.Error = err
    h.done.Store(true)

    h.emitEventLocked(ctx, trace.EventKindStepFailed, map[string]any{
        "step_id": stepID,
        "error":   err.Error(),
    })
    h.emitEventLocked(ctx, trace.EventKindRunCompleted, map[string]any{
        "run_id":      h.run.ID,
        "error":       err.Error(),
        "duration_ms": h.run.CompletedAt.Sub(h.run.StartedAt).Milliseconds(),
    })

    // Phase 8 housekeeping: shut down extension processes.
    if h.engine.cfg.ExtensionHost != nil {
        _ = h.engine.cfg.ExtensionHost.Shutdown(ctx)
    }

    safeClose(h.events, &h.eventsClosed)
    h.cancelFn()
    return nil, err
}
```

Also add Shutdown in the signal cancellation goroutine (`Start()` line ~88):
```go
// In the signal handler:
if h.engine.cfg.ExtensionHost != nil {
    _ = h.engine.cfg.ExtensionHost.Shutdown(ctx)
}
```

**Assigned to:** Brian  
**Effort:** ~15 min  
**Test:** Add `TestEngine_ExtensionHost_ShutdownCalledOnComplete` and `TestEngine_ExtensionHost_ShutdownCalledOnFail` using `FakeExtensionHost`.

---

### H2: Runbook `extensions:` Declarations Must Be Plumbed to `Load()`

**Issue:** `e.cfg.ExtensionHost.Load(ctx, nil)` at `v2/internal/engine/engine.go:48` means runbook-level extension declarations are never passed to the host.

**File:** `v2/internal/engine/engine.go` — line 48

**Fix:**

Build a `ProjectManifest` from the execution plan's metadata:

```go
// Replace:
//   e.cfg.ExtensionHost.Load(ctx, nil)
// With:
manifest := buildManifest(plan)
if err := e.cfg.ExtensionHost.Load(ctx, manifest); err != nil {
    return nil, err
}
```

Helper function:
```go
func buildManifest(plan *enginepkg.ExecutionPlan) *extension.ProjectManifest {
    if plan.Metadata == nil || plan.Metadata.Extensions == nil {
        return nil // still valid — host falls back to directory discovery
    }
    refs := make([]extension.ExtRef, 0, len(plan.Metadata.Extensions))
    for _, ext := range plan.Metadata.Extensions {
        refs = append(refs, extension.ExtRef{
            Name:   ext.Name,
            Path:   ext.Path,
            Grants: ext.Grants,
        })
    }
    return &extension.ProjectManifest{Extensions: refs}
}
```

Note: `plan.Metadata.Extensions` comes from the planner copying `schema.Runbook.Extensions` into the plan metadata. If `ExecutionPlan.Metadata` doesn't yet carry extensions, add:

```go
// In pkg/engine/plan.go, PlanMetadata:
type PlanMetadata struct {
    RunbookID   string
    RunbookPath string
    // ... existing fields ...
    Extensions  []*schema.ExtensionRef // NEW: from runbook.Extensions
}
```

**Assigned to:** Brian  
**Effort:** ~30 min  
**Test:** Add `TestEngine_ExtensionHost_ManifestPassedToLoad` that verifies `FakeExtensionHost.LoadCalls[0].Manifest` is non-nil when plan has extensions.

---

## Implementation Sequence

Phase 8 should be implemented in this order:

| Step | Task | Depends On | Assigned |
|------|------|-----------|----------|
| 1 | H1: Add Shutdown calls in engine | — | Brian |
| 2 | H2: Plumb manifest to Load() | — | Brian |
| 3 | `pkg/input/resolver.go` — interfaces | — | Brian |
| 4 | `pkg/input/registry.go` — interfaces | Step 3 | Brian |
| 5 | `pkg/input/input.go` — add InputValueProvider | — | Brian |
| 6 | `internal/input/env.go` + test | Step 3 | Brian |
| 7 | `internal/input/static.go` + test | Step 3 | Brian |
| 8 | `internal/input/file.go` + test | Step 3 | Brian |
| 9 | `internal/input/workspace.go` + test | Step 3 | Brian |
| 10 | `internal/input/vault.go` + test | Step 3 | Brian |
| 11 | `internal/input/chain.go` + test | Step 3 | Brian |
| 12 | `internal/input/prompt_resolver.go` + test | Steps 3, 5 | Brian |
| 13 | `internal/input/registry.go` + test | Steps 3, 4, 6-12 | Brian |
| 14 | `internal/input/terminal.go` — add PromptValue | Step 5 | Brian |
| 15 | `pkg/testutil/fake_input_registry.go` | Step 4 | Barbara |
| 16 | `pkg/engine/engine.go` — add InputRegistry field | Step 4 | Brian |
| 17 | `pkg/engine/plan.go` — add Inputs, Extensions | — | Brian |
| 18 | `internal/engine/engine.go` — wire resolution in Start() | Steps 13, 16, 17 | Brian |
| 19 | Engine integration test | Step 18 | Brian |
| 20 | `go vet ./...` + `go test -race -count=3 ./...` | All | Brian |

**Estimated total effort:** 3-4 working sessions.

---

## Open Questions

### Q1: Schema `From` field — string or []string?

The current `schema.Input.From` is typed as `string`. The spec allows chains (`from: [env.TOKEN, vault.secret/token, prompt]`). Do we change `From` to `any` (string or []string) or add a separate `FromChain []string` field?

**Proposed answer:** Change to `From any` in schema and normalize at plan time. This is a backwards-compatible parse change (single string → `[]string{s}`). The planner's `InputDecl.From` is always `any` and `normalizeFromField()` handles both.

### Q2: External JSON-RPC providers (`.provider.yaml`)

The spec describes external providers that run as JSON-RPC processes. Should Phase 8 implement these or defer to Phase 15?

**Proposed answer:** DEFER to Phase 15. Phase 8 implements only built-in resolvers (env, file, workspace, vault stub, static, prompt, chain). External provider processes share machinery with the Extension Host and Tool Runtime, so it's natural to add them when we build the full provider lifecycle (Phase 15). The `InputRegistry` interface is designed to accept external resolvers later with zero interface changes.

### Q3: Trace event kind — does `trace.EventKind` need extending?

**Proposed answer:** Yes. Add `EventKindInputResolved EventKind = "input/resolved"` to `pkg/trace/event.go`. This is a leaf addition with no dependency changes.

---

## Import Graph Impact

```
pkg/input/ (leaf — no internal imports)
  ↑
internal/input/ (imports pkg/input only)
  ↑
internal/engine/ (imports pkg/input, internal/input for registry construction)
  ↑
pkg/engine/ (imports pkg/input for EngineConfig types)
```

No cycles introduced. `pkg/input/` remains a leaf package. `internal/input/` imports only `pkg/input`. This follows the established dependency direction (all packages depend inward toward pkg/).

---

## Acceptance Criteria

Phase 8 is APPROVED when:

1. ✅ `go build ./...` — clean
2. ✅ `go vet ./...` — clean
3. ✅ `go test -race -count=3 ./...` — all pass
4. ✅ All 26 tests in the test plan exist and pass
5. ✅ `EngineConfig.InputRegistry` field exists and is documented
6. ✅ `EnvResolver` resolves `from: env.X` bindings in a runbook
7. ✅ `StaticResolver` enables test/replay scenarios
8. ✅ `ChainResolver` implements left-to-right fallback
9. ✅ `TerminalInputProvider` implements `InputValueProvider`
10. ✅ Engine calls `ExtensionHost.Shutdown()` on run complete/fail/cancel (H1)
11. ✅ Engine passes manifest to `ExtensionHost.Load()` (H2)
12. ✅ No import cycles
13. ✅ `ResolveResponse` carries replay-compatible metadata

---

## Appendix: Wire Compatibility Notes

### Phase 12 Replay Contract (Preview)

When Phase 12 is implemented, it will:
1. Read `input/resolved` trace events from a completed run
2. Construct `map[inputName]string` from recorded values
3. Build a `StaticResolver` with those values
4. Register it as the first (highest-priority) resolver in a new registry
5. Run the engine with this registry — all inputs resolve from recorded values

This contract requires only that Phase 8's `ResolveResponse` is JSON-serializable and that `StaticResolver` can be seeded from a `map[string]string`. Both are guaranteed by the design above.

### External Provider Contract (Phase 15 Preview)

When Phase 15 adds external `.provider.yaml` support, it will:
1. Add an `ExternalResolver` that wraps a JSON-RPC process (similar to Tool Runtime)
2. Register it in the `InputRegistry` with the provider's declared prefix
3. The registry's prefix-matching dispatch routes bindings to it automatically
4. Batching (spec §14.3.3) is handled inside `ExternalResolver.Resolve()` by collecting all bindings with the same prefix before sending a single RPC

No changes to `InputRegistry` or `InputResolver` interfaces will be needed.
