# Phase 7 Extension Host — Spec Audit & Fixture Planning

**Auditor:** Barbara  
**Date:** 2026-04-20  
**Requested by:** Cristian

---

## 1. Spec Summary

### Location
- **Primary spec:** `design/gert-v2/sections/04-extension-runtime.tex` (450 lines)
- **Normative reference:** `specs/002-extension-runtime-v0/spec.md` (300+ lines)
- **Markdown companion:** `design/gert-v2/spec/04-extension-runtime.md` (290 lines)

The TeX document supersedes the normative spec for design purposes (per line 11-14 of the TeX file).

### Extension Lifecycle Stages

1. **discovered**: Manifest found and parsed; no process yet.
2. **verified**: Manifest validated; capabilities evaluated against policy.
3. **starting**: Executable launched; stdin/stdout pipes attached.
4. **initializing**: `extension/initialize` sent; awaiting response.
5. **ready**: `contributions/list` completed; accepting invocations.
6. **draining**: `extension/shutdown` sent; waiting for in-flight completions.
7. **stopped**: Process has exited cleanly.
8. **crashed**: Process exited unexpectedly.

### Communication Protocol

**Transport:** JSON-RPC 2.0 over stdio (primary). Future: grpc, mcp.

**Handshake flow:**
1. Host sends `extension/initialize` with:
   - Host identity (name, version, apiVersion)
   - Protocol version
   - **Granted capabilities** (subset of requested)
   - Run context (runId, mode)
2. Extension responds with:
   - Extension identity (name, version)
   - Protocol version
   - **Acknowledged capabilities**
3. Host sends `contributions/list`
4. Extension returns:
   - `tools`: array of tool definitions
   - `providers`: array of provider definitions  
   - `schemaExtensions`: array of schema contributions
   - `policyContributions`: array of governance rules

**Health/lifecycle:**
- `extension/ping` — periodic health check (5s timeout)
- `extension/shutdown` — graceful termination (10s timeout, SIGTERM/SIGKILL escalation)

### Contribution Types

| Type | Capability Required | What It Contributes |
|------|---------------------|---------------------|
| **Tools** | `capability/tool-registration` | New tool definitions (name, transport, command, args). Extensions register tools that the host can invoke. |
| **Providers** | `capability/provider-registration` | New input providers that resolve `from:` prefixes in runbook inputs. |
| **Schema extensions** | `capability/schema-extension` | Namespaced additional fields on runbook schema (e.g. `x-acme.*`). Cannot modify core schema. |
| **Policy contributions** | `capability/policy-contribution` | Additional governance rules evaluated during pre-flight/approval. Cannot override builtins. |

### Grants/Permissions Model

**Capability Taxonomy** (10 capabilities defined):

1. `capability/tool-registration` — register tools
2. `capability/schema-extension` — add namespaced schema fields
3. `capability/event-subscription` — subscribe to runtime events
4. `capability/policy-contribution` — register governance rules
5. `capability/provider-registration` — register input providers
6. `capability/file-read` — read files (scoped to `file-read-paths`)
7. `capability/file-write` — write files (scoped to `file-write-paths`)
8. `capability/network` — outbound network (scoped to `network-hosts`)
9. `capability/env-read` — read env vars (scoped to `env-read-patterns`)
10. `capability/run-state-read` — read current run state (readonly)

**Grant model:**
- Extension manifest declares **requested** capabilities.
- Host policy evaluates and returns **granted** subset during initialize.
- Extension MUST NOT execute operations outside granted set.
- Host MUST reject invocations requiring non-granted capabilities (error code `-32010`).

**Enforcement:**
- Capability check before dispatching any extension-initiated request.
- File/network constraints enforced at syscall boundary (OS-level where available: pledge/unveil, seccomp).

### Project Manifest Format

**Extensions declared in 3 places** (precedence order):

1. **Workspace config:** `.gert/extensions.yaml`
   ```yaml
   extensions:
     - path: "./extensions/acme-pack"   # directory with gert-extension.yaml
   ```

2. **Runbook top-level field:**
   ```yaml
   apiVersion: runbook/v2
   meta:
     name: incident-triage
   extensions:
     - path: "./extensions/acme-pack"
   ```

3. **CLI override:** `--extension <path>`

**Extension manifest:** `gert-extension.yaml` (YAML format in TeX spec, JSON in normative spec)

Required fields:
- `apiVersion: extension/v2`
- `meta.name` (reverse-DNS, e.g. `acme.incident-pack`)
- `meta.version` (semver)
- `compatibility.api-version` (semver range, e.g. `>=2.0.0 <3.0.0`)
- `compatibility.protocol-version` (e.g. `2.0`)
- `entry-point.executable` (path)
- `transport` (`stdio-jsonrpc` | `grpc` | `mcp`)
- `capabilities` (array of capability strings)

Conditional fields:
- `file-read-paths` (required if `capability/file-read` requested)
- `network-hosts` (required if `capability/network` requested)
- `env-read-patterns` (required if `capability/env-read` requested)

Optional:
- `platform.os`, `platform.arch` — OS/arch constraints

### Example Extension Snippets

**Manifest example** (from TeX spec lines 116-161):
```yaml
apiVersion: extension/v2
meta:
  name: acme.incident-pack
  version: "1.2.0"
  description: "Acme incident resolution tools and providers"
  author: "Acme Platform Team <platform@acme.example>"
compatibility:
  api-version: ">=2.0.0 <3.0.0"
  protocol-version: "2.0"
entry-point:
  executable: "./bin/acme-extension"
  args: ["--config", "acme.yaml"]
transport: stdio-jsonrpc
capabilities:
  - capability/tool-registration
  - capability/provider-registration
  - capability/network
file-read-paths:
  - "./runbooks/**"
network-hosts:
  - "api.pagerduty.com"
  - "*.acme.example"
env-read-patterns:
  - "ACME_API_*"
  - "PD_TOKEN"
platform:
  os: ["linux", "darwin", "windows"]
  arch: ["amd64", "arm64"]
```

**Tool contribution shape** (from normative spec lines 216-226):
```json
{
  "name": "acme.pd-lookup",
  "description": "Query PagerDuty incident by ID",
  "inputSchema": { ... },
  "outputSchema": { ... },
  "requiredCapabilities": ["capability/network"],
  "timeoutMsDefault": 5000
}
```

**Provider contribution shape** (from TeX spec line 197):
```json
{
  "name": "pd",
  "prefixes": ["pd."]
}
```

---

## 2. Runbook Coverage — Existing Extension Usage

**Result:** Zero runbooks currently declare or use extensions.

**Evidence:**
- `grep -r "extensions:" testdata/runbooks/` → no matches
- `grep -ri "extension" testdata/runbooks/` → 22 matches, all in comments/assessments
  - r01 schema.yaml line 61: `# Previously: type: extension (incorrect)` — a corrected mistake
  - Other matches are prose references to "VS Code extension" in assessment docs

**Notable:**
- r01 uses `type: wait_for_event` (was incorrectly `type: extension` per comment)
- All runbooks (r01–r17) use only **builtin tools** (`slack-notify`, `pagerduty-notify`, `alertmanager-notify`)
- No runbook has an `extensions:` top-level field

**Conclusion:** Phase 7 implementation will introduce the **first runbook that uses extensions**.

---

## 3. Fixture Recommendations

### 3.1 New Runbook: r18-extension-contrib

**Purpose:** End-to-end test of extension lifecycle + contributed tool invocation.

**Scenario:** Extension contributes a custom tool; runbook uses that tool in a step.

**Suggested content:**
```yaml
apiVersion: runbook/v2
id: extension-contrib-demo
name: Extension Contribution Demo
kind: reference
description: |
  Validates Phase 7 extension runtime: manifest discovery, handshake,
  contribution registration, and tool invocation.

extensions:
  - path: "./fixtures/acme-stub-extension"

toolRefs:
  - name: acme.echo
    source: extension://acme-stub-extension

inputs:
  message:
    type: string
    required: true
    default: "hello from r18"

flow:
  - step:
      id: invoke_extension_tool
      type: tool
      title: "Invoke extension-contributed tool"
      tool:
        name: acme.echo
        action: echo
        args:
          message: "{{ .message }}"
      capture:
        response: json.echoed

  - step:
      id: complete
      type: end
      title: Extension tool invocation complete
      outcome:
        category: success
        code: extension_tool_ok
```

**Assessment notes:**
- Validates extension manifest parsing
- Validates handshake protocol
- Validates tool contribution registration
- Validates tool invocation dispatch
- Validates capability enforcement (if tool requires network/file caps)

### 3.2 Reference Extension Binary: acme-stub-extension

**Location:** `design/gert-v2/testdata/runbooks/r18-extension-contrib/fixtures/acme-stub-extension/`

**Contents:**
```
fixtures/acme-stub-extension/
├── gert-extension.yaml       # Extension manifest
└── bin/
    └── acme-stub             # Minimal Go binary or shell script
```

**Manifest (`gert-extension.yaml`):**
```yaml
apiVersion: extension/v2
meta:
  name: acme-stub-extension
  version: "0.1.0"
  description: "Minimal stub extension for Phase 7 testing"
  author: "gert-v2 team"
compatibility:
  api-version: ">=2.0.0 <3.0.0"
  protocol-version: "2.0"
entry-point:
  executable: "./bin/acme-stub"
  args: []
transport: stdio-jsonrpc
capabilities:
  - capability/tool-registration
platform:
  os: ["linux", "darwin"]
  arch: ["amd64", "arm64"]
```

**Binary behavior:**
- Implements JSON-RPC 2.0 over stdio
- Responds to `extension/initialize` → returns extension identity + acknowledges capabilities
- Responds to `contributions/list` → returns one tool: `acme.echo`
- Responds to `tools/invoke` with action `echo` → echoes input message
- Responds to `extension/ping` → `{ "status": "ok" }`
- Responds to `extension/shutdown` → exits gracefully

**Implementation strategy:**
- Option A: Go binary using Phase 6 JSON-RPC primitives
- Option B: Shell script with `jq` for JSON parsing (simpler, portable)
- **Recommendation:** Go binary (reuses Phase 4 transport code; easier to maintain)

### 3.3 Extension Policy Fixture (Optional)

**File:** `.gert/extension-policy.yaml` (if host policy is implemented)

```yaml
default: deny
extensions:
  acme-stub-extension:
    allow:
      - capability/tool-registration
```

**Note:** Spec mentions policy file but doesn't mandate format. May be deferred to Phase 7 implementation decision.

### 3.4 Workspace Extension Config Fixture

**File:** `testdata/runbooks/r18-extension-contrib/.gert/extensions.yaml`

```yaml
extensions:
  - path: "./fixtures/acme-stub-extension"
```

**Purpose:** Tests workspace-level extension discovery (distinct from runbook-level `extensions:` field).

---

## 4. Schema Gaps

### 4.1 Missing Types in `v2/pkg/schema/`

**Current state:**
- ✅ `Runbook` type exists (`runbook.go`)
- ✅ `ProviderDef` type exists (`provider.go`)
- ❌ **No `ExtensionManifest` type**
- ❌ **No `extensions` field on `Runbook` struct**
- ❌ **No `WorkspaceConfig` or `.gert/extensions.yaml` schema**

**Required additions:**

#### 4.1.1 ExtensionManifest Type

**File:** `v2/pkg/schema/extension.go` (new file)

```go
package schema

// ExtensionManifest is the parsed definition of a gert-extension.yaml file.
type ExtensionManifest struct {
    APIVersion    string                  `yaml:"apiVersion"       json:"apiVersion"`
    Meta          ExtensionMeta           `yaml:"meta"             json:"meta"`
    Compatibility ExtensionCompatibility  `yaml:"compatibility"    json:"compatibility"`
    EntryPoint    ExtensionEntryPoint     `yaml:"entry-point"      json:"entry-point"`
    Transport     string                  `yaml:"transport"        json:"transport"` // stdio-jsonrpc | grpc | mcp
    Capabilities  []string                `yaml:"capabilities"     json:"capabilities"`
    FileReadPaths []string                `yaml:"file-read-paths,omitempty"  json:"file-read-paths,omitempty"`
    FileWritePaths []string               `yaml:"file-write-paths,omitempty" json:"file-write-paths,omitempty"`
    NetworkHosts  []string                `yaml:"network-hosts,omitempty"    json:"network-hosts,omitempty"`
    EnvReadPatterns []string              `yaml:"env-read-patterns,omitempty" json:"env-read-patterns,omitempty"`
    Platform      *ExtensionPlatform      `yaml:"platform,omitempty" json:"platform,omitempty"`
}

type ExtensionMeta struct {
    Name        string `yaml:"name"                   json:"name"`
    Version     string `yaml:"version"                json:"version"`
    Description string `yaml:"description,omitempty"  json:"description,omitempty"`
    Author      string `yaml:"author,omitempty"       json:"author,omitempty"`
}

type ExtensionCompatibility struct {
    APIVersion      string `yaml:"api-version"       json:"api-version"`
    ProtocolVersion string `yaml:"protocol-version"  json:"protocol-version"`
}

type ExtensionEntryPoint struct {
    Executable string   `yaml:"executable" json:"executable"`
    Args       []string `yaml:"args,omitempty" json:"args,omitempty"`
}

type ExtensionPlatform struct {
    OS   []string `yaml:"os,omitempty"   json:"os,omitempty"`
    Arch []string `yaml:"arch,omitempty" json:"arch,omitempty"`
}
```

#### 4.1.2 Runbook Extensions Field

**File:** `v2/pkg/schema/runbook.go` (line 14, add after ToolRefs)

```go
type Runbook struct {
    // ... existing fields ...
    ToolRefs    []*ToolRef          `yaml:"toolRefs,omitempty"    json:"toolRefs,omitempty"`
    Extensions  []*ExtensionRef     `yaml:"extensions,omitempty"  json:"extensions,omitempty"`  // NEW
    Imports     map[string]string   `yaml:"imports,omitempty"     json:"imports,omitempty"`
    // ... rest of fields ...
}

type ExtensionRef struct {
    Path string `yaml:"path" json:"path"`  // Path to directory containing gert-extension.yaml
}
```

#### 4.1.3 Workspace Config Type (Optional)

**File:** `v2/pkg/schema/workspace.go` (new file)

```go
package schema

// WorkspaceConfig is the parsed definition of .gert/extensions.yaml.
type WorkspaceConfig struct {
    Extensions []*ExtensionRef `yaml:"extensions" json:"extensions"`
}
```

**Note:** May be deferred if Phase 7 only implements runbook-level extension discovery.

### 4.2 Schema Gap Summary

| Type | Status | Action Required |
|------|--------|-----------------|
| `ExtensionManifest` | ❌ Missing | Ken must design; add to `schema/extension.go` |
| `Runbook.Extensions` | ❌ Missing | Add `Extensions []*ExtensionRef` field to `schema/runbook.go` |
| `WorkspaceConfig` | ❌ Missing | Ken decides if Phase 7 includes workspace discovery |
| `ProviderDef` | ✅ Exists | No gap |
| Tool contribution schema | ❓ Unclear | Need `ContributionManifest` type? Or inline in extension protocol? |

**Recommendation for Ken:**
- Design `ExtensionManifest` type (highest priority)
- Add `extensions` field to `Runbook` (required for r18)
- Decide: does Phase 7 include workspace discovery or only runbook-level?
- Decide: do contribution schemas live in `pkg/schema` or in extension protocol package?

---

## 5. Registry Gap — Dynamic Tool Registration

### 5.1 Current State

**Phase 6 registry:** `v2/internal/tool/MapRegistry`

**Interface:** `v2/pkg/tool/tool.go`
```go
type ToolRegistry interface {
    Lookup(name string) (*ToolDef, bool)
    All() []ToolDef
}
```

**Implementation:** `v2/internal/tool/registry.go`
- Constructor: `NewMapRegistry(defs []ToolDef)` — accepts slice at construction
- **No `Register(def ToolDef)` method**
- Thread-safe read (RWMutex) but **no write after construction**

### 5.2 Gap Analysis

**Extension lifecycle requires:**
1. Host constructs registry with builtin tools (Phase 6 ✅)
2. Host discovers extensions (Phase 7 startup)
3. Host sends `contributions/list` to each extension
4. Extension returns array of `ToolDef` contributions
5. **Host must dynamically register these tools into the registry** ❌ NOT SUPPORTED

**Current registry is read-only after construction.**

### 5.3 Required Augmentation

**Option A: Add `Register()` method to `ToolRegistry` interface**

```go
// v2/pkg/tool/tool.go
type ToolRegistry interface {
    Lookup(name string) (*ToolDef, bool)
    All() []ToolDef
    Register(def ToolDef) error  // NEW: dynamic registration
}
```

**Implementation in `MapRegistry`:**
```go
// v2/internal/tool/registry.go
func (r *MapRegistry) Register(def ToolDef) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    if _, exists := r.tools[def.Name]; exists {
        return fmt.Errorf("tool %q already registered", def.Name)
    }
    r.tools[def.Name] = def
    return nil
}
```

**Option B: Add `RegisterBatch()` for efficiency**

```go
func (r *MapRegistry) RegisterBatch(defs []ToolDef) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    for _, def := range defs {
        if _, exists := r.tools[def.Name]; exists {
            return fmt.Errorf("tool %q already registered", def.Name)
        }
    }
    for _, def := range defs {
        r.tools[def.Name] = def
    }
    return nil
}
```

**Recommendation:** Option A (single `Register`) is sufficient. Batch can be added later if profiling shows lock contention.

### 5.4 Additional Considerations

**Extension tool namespacing:**
- Spec recommends `namespace.tool` format (e.g. `acme.pd-lookup`)
- Registry lookup uses full name → natural collision prevention
- No code change needed if extension authors follow convention

**Tool unregistration:**
- Spec doesn't mention hot-reload or extension unload
- Phase 7 likely doesn't need `Unregister()`
- Defer until Phase 7 design clarifies extension lifecycle

**Thread safety:**
- Current `MapRegistry` uses `sync.RWMutex` → already thread-safe for reads
- Adding write requires `Lock()` → **implementation is straightforward**

### 5.5 Registry Gap Summary

| Feature | Status | Action Required |
|---------|--------|-----------------|
| Read-only registry | ✅ Implemented (Phase 6) | None |
| Dynamic registration | ❌ Missing | Add `Register(def ToolDef) error` method |
| Thread-safe writes | ⚠️ Mutex exists | Use `Lock()` in new `Register()` method |
| Collision detection | ❓ Not implemented | Return error if tool name exists |
| Batch registration | ❌ Missing | Optional optimization |
| Unregister | ❌ Missing | Not needed for Phase 7 (defer) |

**Blocker status:** ⚠️ **Medium-severity gap**  
Phase 7 cannot proceed without dynamic registration. However, the fix is small (10 lines) and low-risk.

**Recommendation for Ken:** Approve augmenting `ToolRegistry` interface with `Register()`. Brian should implement this as part of Phase 7 (or as a quick Phase 6.1 patch).

---

## Appendices

### A. Tool Contribution JSON Schema (from normative spec)

Per `specs/002-extension-runtime-v0/spec.md` lines 216-226:

```json
{
  "name": "string (required, namespace.tool format)",
  "description": "string (required)",
  "inputSchema": "json schema object (required)",
  "outputSchema": "json schema object (required)",
  "requiredCapabilities": ["array of capability strings (required)"],
  "timeoutMsDefault": "number (required)"
}
```

**Gap:** No Go type for this in `pkg/schema`. Ken should decide if `ToolDef` from `pkg/tool` suffices or if a separate `ContributedToolDef` is needed.

### B. Provider Contribution Shape (inferred from spec)

From TeX spec line 197 example:

```json
{
  "name": "pd",
  "prefixes": ["pd."]
}
```

**Gap:** `pkg/schema/provider.go` defines `ProviderDef` with transport/fields, but no `prefixes` field. Extension-contributed providers may have different schema than `.provider.yaml` files.

**Recommendation:** Ken clarifies if extension providers use same schema or distinct type.

### C. Phase Dependencies

Phase 7 depends on:
- ✅ Phase 4: JSON-RPC transport (used for stdio-jsonrpc handshake)
- ✅ Phase 5: Provider resolution (extensions contribute providers)
- ✅ Phase 6: Tool registry (extensions contribute tools)
- ❌ Phase 6 augmentation: dynamic `Register()` method (gap identified above)

### D. Spec Ambiguities / Open Questions for Ken

1. **Extension-contributed tool schema:** Does `ToolDef` (from `pkg/tool`) match the contribution JSON shape from normative spec? Or are they distinct?
2. **Provider contribution schema:** `ProviderDef` has `transport`/`fields`. Extension providers only declare `prefixes`. Same type or different?
3. **Workspace discovery:** Phase 7 scope includes `.gert/extensions.yaml` discovery, or only runbook-level `extensions:` field?
4. **Policy enforcement:** Host policy file (`gert-extension-policy.yaml`) is mentioned in normative spec but not in TeX spec. Implement in Phase 7 or defer?
5. **Multiple transports:** Spec mentions `grpc` and `mcp` as future transports. Phase 7 only implements `stdio-jsonrpc`?

---

## Summary

**Spec is comprehensive.** 04-extension-runtime.tex and normative spec together define:
- 8-state lifecycle
- 10 capabilities with scoped grants
- 4 contribution types (tools, providers, schema, policy)
- JSON-RPC protocol with 5 core methods

**Runbooks are extension-free.** r01–r17 use zero extensions. r18 will be first.

**Fixtures needed:**
1. **r18** runbook with `extensions:` field + tool invocation
2. **acme-stub-extension** reference binary (Go or shell)
3. **gert-extension.yaml** manifest fixture
4. **(Optional)** workspace `.gert/extensions.yaml` config

**Schema gaps:**
- Missing `ExtensionManifest` type
- Missing `Runbook.Extensions` field
- Unclear if workspace config in Phase 7 scope
- Ambiguity on tool/provider contribution types vs existing `ToolDef`/`ProviderDef`

**Registry gap:**
- `ToolRegistry` lacks `Register()` method for dynamic contributions
- **Blocker** for Phase 7 but trivial fix (~10 lines)
- Recommend augmenting interface as part of Phase 7 or quick Phase 6 patch

**Next steps:**
1. Ken designs `ExtensionManifest` schema
2. Ken clarifies tool/provider contribution types
3. Ken decides workspace discovery scope
4. Brian implements `ToolRegistry.Register()` (or Ken pre-patches Phase 6)
5. Brian creates r18 runbook + acme-stub-extension fixtures
6. Brian implements Phase 7 host-side extension manager
