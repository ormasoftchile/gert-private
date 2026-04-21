# Phase 8 Input Provider Framework — Spec Audit

**Date:** 2026-04-20  
**Auditor:** Barbara (Integrations Specialist)  
**Spec File:** `sections/14-input-provider-framework.tex` (1003 lines, 11 sections)  
**Module:** `github.com/ormasoftchile/gert/v2`

---

## Executive Summary

Phase 8 delivers the **Input Provider Framework**, enabling runbooks to resolve input values from external sources (environment variables, files, Vault, PagerDuty, etc.) at run time. The framework shares the same transport layer as tools (stdio/stdio-jsonrpc/mcp) and provides transparent audit trails for all resolved values.

**Current Status:**
- ✅ Public interface `pkg/input/InputProvider` exists and is wired into `EngineConfig`
- ✅ Built-in `prompt` provider (`internal/input/terminal.go`) implements `PromptChoice`, `PromptDecision`, and `PromptForm` for interactive steps
- ✅ Schema includes `Input.From` field (line 48 in `runbook.go`)
- ⚠️ **Gap:** No provider registry, resolution pipeline, or external provider support yet
- ⚠️ **Gap:** Built-in providers (`env`, `file`, `workspace`) not implemented
- ⚠️ **Gap:** Dynamic options protocol (`inputProvider/getOptions`) not implemented
- ⚠️ **Gap:** Autocomplete search protocol (`inputProvider/search`) not implemented

---

## Spec Summary (Section 14)

### What Phase 8 Delivers

**1. Provider Definition Schema** (§14.2)
- Providers declared in `.provider.yaml` files (project-level: `providers/<name>.provider.yaml`)
- Schema includes: metadata, transport config, resolution method, input/output schemas, cache policy, governance requirements
- Example: `pd.provider.yaml` for PagerDuty input resolution

**2. Resolution Protocol** (§14.3)
- When a runbook input has `from: pd.incident.title`, the framework:
  1. Matches prefix to provider (`pd.*` → PagerDuty provider)
  2. Starts provider process if not running
  3. Batches all inputs for same provider into one JSON-RPC call
  4. Calls `provider/resolve` method
  5. Merges resolved values into run variables
  6. Falls back to `fallback:` strategy for unresolved bindings
- Fallback modes: `prompt` (default), `default`, `fail`
- Caching: `run`, `step`, or `none` scope

**3. Built-in Providers** (§14.4)
- **env**: Reads from environment variables (`from: env.ACME_API_KEY`)
- **file**: Reads from files with optional JSON Pointer (`from: file.~/.ssh/id_rsa` or `from: file./etc/config.json#/database/host`)
- **prompt**: Interactive terminal prompt (default fallback)
- **workspace**: Reads from `.gert/config.yaml` workspace config (`from: workspace.defaults.region`)

**4. Interactive Step Contracts** (§14.5)
Already implemented in codebase:
- **choice**: User selects from fixed options → `PromptChoice`
- **decision**: User selects routing label → `PromptDecision`
- **collector**: User fills multi-field form → `PromptForm`

**5. Dynamic Options Protocol** (§14.5.3)
- For `options_from.provider` in select fields
- JSON-RPC method: `inputProvider/getOptions`
- Providers return array of `{value, label}` objects
- Supports caching with `cacheTtlSeconds`

**6. Autocomplete Search Protocol** (§14.5.4)
- For `type: autocomplete` fields in collector steps
- JSON-RPC method: `inputProvider/search`
- Live search with 500ms timeout
- Degrades to static select in CLI mode

**7. Provider Composition** (§14.7)
- Inputs can declare priority-ordered chain: `from: [env.TOKEN, vault.secret/token, prompt]`
- First successful provider wins

**8. Provider Lifecycle** (§14.8)
- Persistent processes for `stdio-jsonrpc` / `mcp` (started once, kept alive)
- Per-resolution for `stdio` transport
- Lazy startup on first use
- Graceful shutdown: `shutdown` method → 5s grace → SIGTERM → 5s → SIGKILL

---

## Gap Analysis

### What EXISTS in v2 Codebase

✅ **pkg/input/input.go**
- `InputProvider` interface with 3 methods: `PromptChoice`, `PromptDecision`, `PromptForm`
- Request/Response types for all 3 interactive step types
- Matches spec §14.5 contracts exactly

✅ **internal/input/terminal.go**
- `TerminalInputProvider` implements the `prompt` built-in provider
- Reads from stdin/stdout for interactive steps
- Used by choice/decision/collector executors

✅ **pkg/engine/engine.go**
- `EngineConfig.InputProvider` field (line 81, optional)
- Comment: "supplies interactive input for choice/decision/collector steps"

✅ **pkg/schema/runbook.go**
- `Input.From` field (line 48): `yaml:"from,omitempty"`
- Input declarations support `from:` binding syntax

✅ **Executors wired to InputProvider**
- `internal/executor/choice.go`: `NewChoiceExecutor(in input.InputProvider, ...)`
- `internal/executor/decision.go`: `NewDecisionExecutor(in input.InputProvider, ...)`
- `internal/executor/collector.go`: `NewCollectorExecutor(in input.InputProvider, ...)`

✅ **pkg/schema/provider.go**
- Stub `ProviderDef` schema exists (lines 1–18)
- Matches `.provider.yaml` structure

✅ **Existing test fixtures use `from:` bindings**
- `r01-k8s-incident/schema.yaml`: `from: webhook.pod_name`, `from: webhook.severity`
- `r02-canary-deploy/schema.yaml`: `from: prompt`, `from: env.CURRENT_VERSION`
- `r03-employee-onboarding/schema.yaml`: `from: prompt`
- `r07-financial-approval/schema.yaml`: `from: prompt`

---

### What is MISSING (Phase 8 Deliverables)

❌ **pkg/provider/** package is empty (only doc.go)
- No `Provider` interface
- No `ProviderRegistry` for prefix matching
- No resolution pipeline implementation

❌ **No built-in provider implementations**
- `env` provider: read from `os.Getenv()`
- `file` provider: read from filesystem with JSON Pointer support
- `workspace` provider: read from `.gert/config.yaml`
- (Note: `prompt` provider already exists as `internal/input/terminal.go`)

❌ **No external provider support**
- No process lifecycle management (spawn, ready-pattern, shutdown)
- No JSON-RPC `provider/resolve` dispatcher
- No batching logic for same-provider inputs
- No caching (run/step/none scope)

❌ **No dynamic options protocol**
- `inputProvider/getOptions` method not implemented
- No `options_from` field resolver in choice/collector executors

❌ **No autocomplete search protocol**
- `inputProvider/search` method not implemented
- No autocomplete field support in collector executor
- No CLI degradation path

❌ **No provider composition**
- Priority-ordered chain `from: [env.TOKEN, vault.secret, prompt]` not supported
- Parser/schema may not support `from` as array (currently string field)

❌ **No provider discovery**
- No loading of `.provider.yaml` files from `providers/` directory
- No validation of provider definitions against schema

❌ **No input resolution at run start**
- Run engine doesn't call provider framework before first step
- No pre-flight resolution phase
- No trace events for resolved values (`input.resolved`, `input.fallback`)

---

## r19 Fixture Runbook Description

**File:** `testdata/runbooks/r19-input-provider/schema.yaml`

**Purpose:** Exercise the Input Provider Framework with all major resolution paths.

**What r19 Tests:**

1. **env provider**: Input resolved from environment variable
2. **file provider**: Input resolved from a file (simulated with comment)
3. **prompt provider**: Input resolved via interactive prompt (with default)
4. **Chained providers**: Input with fallback chain (env → prompt)
5. **Value flow validation**: Shell step uses resolved input value to prove end-to-end flow
6. **Dynamic options** (future): Commented placeholder for `options_from` once implemented
7. **Collector step**: Multi-field form using resolved inputs
8. **Governance metadata**: Appropriate evidence and compliance fields

**Parser Impact:**
- Currently `Input.From` is `string` — r19 uses simple string values
- Parser will accept the YAML but may not resolve providers yet (expected until Phase 8 implements resolution pipeline)
- Schema validation will pass; runtime will fail at resolution (expected)

**What Parser Needs (Phase 8):**
1. Support `from:` as either `string` or `[]string` for provider chains
2. Pre-flight resolution phase in run engine before step 0
3. Provider registry to map prefixes to provider instances
4. Fallback logic for unresolved bindings
5. Trace events: `input.resolved`, `input.provider_started`, `input.fallback`

---

## Recommendations for Brian

### Priority 1: Core Infrastructure

1. **Provider Registry** (`pkg/provider/registry.go`)
   - Prefix matching: longest match wins
   - Provider instance lifecycle (lazy start, persistent processes)
   - Registration API for built-in and external providers

2. **Resolution Pipeline** (`pkg/provider/resolver.go`)
   - Pre-flight phase: resolve all inputs before step 0
   - Batching: group inputs by provider prefix
   - Fallback handling: `prompt`, `default`, `fail`
   - Trace events for audit trail

3. **Built-in Providers** (`internal/provider/`)
   - `env.go`: `EnvProvider` reads from `os.Getenv()`
   - `file.go`: `FileProvider` reads files, parses JSON Pointer fragments
   - `workspace.go`: `WorkspaceProvider` reads `.gert/config.yaml`
   - (Reuse existing `internal/input/terminal.go` as `prompt` provider)

### Priority 2: External Provider Support

4. **Process Lifecycle** (`pkg/provider/process.go`)
   - Spawn provider binary
   - Wait for ready-pattern (reuse tool transport logic from Phase 6)
   - JSON-RPC transport wrapper
   - Graceful shutdown sequence

5. **JSON-RPC Dispatcher** (`pkg/provider/rpc.go`)
   - `provider/resolve` method handler
   - Request/response envelope validation
   - Error handling and fallback

### Priority 3: Advanced Features

6. **Dynamic Options**
   - `inputProvider/getOptions` implementation
   - Integrate with choice/collector executors
   - Caching support

7. **Autocomplete Search**
   - `inputProvider/search` implementation
   - 500ms timeout enforcement
   - CLI degradation logic

8. **Provider Composition**
   - Schema change: `Input.From` as `string | []string`
   - Chain evaluation: try left-to-right, stop on first success

### Testing Strategy

- **Unit tests**: Each built-in provider in isolation
- **Contract tests**: Provider interface compliance (choice/decision/collector)
- **Integration tests**: Full resolution pipeline with mock providers
- **Golden traces**: `input.resolved` events in deterministic trace output
- **r19 fixture**: End-to-end validation with real provider instances

### Open Questions

1. **Q: Should `from:` support template expressions?**
   - Example: `from: env.{{ .environment }}_API_KEY`
   - Spec doesn't mention this; clarify with Ken

2. **Q: How do providers authenticate?**
   - Vault provider needs token, PagerDuty needs OAuth
   - Spec §14.2 shows `env-read-patterns` in governance but no auth mechanism
   - Clarify provider credential management

3. **Q: What happens if provider crashes mid-resolution?**
   - Spec §14.8 says "apply fallback strategy for in-flight bindings"
   - Should run continue or abort? (Per-input fallback vs. run-level failure)

4. **Q: Are provider processes shared across runs?**
   - Spec says "kept alive for entire run" but doesn't specify cross-run pooling
   - Performance vs. isolation trade-off

---

## Appendix: Spec Compliance Checklist

Phase 8 implementation MUST satisfy these spec requirements:

- [ ] §14.2: Parse `.provider.yaml` files with all required fields
- [ ] §14.3.1: Prefix matching algorithm (longest match wins)
- [ ] §14.3.1: Batch inputs by provider prefix
- [ ] §14.3.1: Lazy provider startup on first use
- [ ] §14.3.1: `provider/resolve` JSON-RPC call
- [ ] §14.3.1: Merge resolved values into variable map
- [ ] §14.3.1: Fallback strategy: prompt/default/fail
- [ ] §14.3.2: Cache policy: run/step/none scope
- [ ] §14.4.1: Built-in `env` provider
- [ ] §14.4.2: Built-in `file` provider with JSON Pointer
- [ ] §14.4.3: Built-in `prompt` provider (reuse existing terminal.go)
- [ ] §14.4.4: Built-in `workspace` provider
- [ ] §14.5.1: `PromptChoice` already exists ✅
- [ ] §14.5.2: `PromptDecision` already exists ✅
- [ ] §14.5.3: `PromptForm` already exists ✅
- [ ] §14.5.3: Dynamic options protocol `inputProvider/getOptions`
- [ ] §14.5.4: Autocomplete search protocol `inputProvider/search`
- [ ] §14.5.4: 500ms timeout for search requests
- [ ] §14.5.4: CLI degradation for autocomplete → select
- [ ] §14.6: Provider capability matrix (choice/decision/collector/getOptions/search)
- [ ] §14.7: Provider composition (priority-ordered chain)
- [ ] §14.8: Process lifecycle: startup, ready-pattern, shutdown
- [ ] §14.8: Graceful shutdown: method → 5s → SIGTERM → 5s → SIGKILL
- [ ] §14.8: Crash handling: log event, apply fallback

**Total:** 26 requirements (3 already satisfied)

---

**End of Audit**
