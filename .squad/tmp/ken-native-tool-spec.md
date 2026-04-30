# Architecture Spec: Native CLI Tool Transport

**Author:** Ken (Software Architect)  
**Date:** 2025-04-24  
**Status:** Draft → Brian to implement  

---

## 1. Problem Statement

The gert v2 tool system currently supports `stdio`, `jsonrpc`, and `mcp` transports, all of which expect tools to speak a JSON protocol. However, native CLI utilities like `ping`, `curl`, `nslookup` are argv-style commands that write plain text to stdout/stderr. The v1 `.tool.yaml` format handled this via per-action `argv:` templates. We need to add a `native` transport type to v2 that spawns a binary with rendered argv, captures output, and returns exit code — no JSON protocol required.

---

## 2. Schema Changes

### 2.1 Add `TransportNative` to `pkg/schema/tool.go`

**File:** `/Volumes/Projects/gert/pkg/schema/tool.go`

**Modification:** Add new constant after line 62:

```go
const (
	TransportStdio   Transport = "stdio"
	TransportJSONRPC Transport = "jsonrpc"
	TransportMCP     Transport = "mcp"
	TransportNative  Transport = "native"  // ← ADD
)
```

### 2.2 Add `Argv []string` to `schema.ToolAction`

**File:** `/Volumes/Projects/gert/pkg/schema/tool.go`

**Modification:** Add `Argv` field to `ToolAction` struct (after line 37):

```go
// ToolAction is a single invocable action declared in a tool definition.
type ToolAction struct {
	Description string              `yaml:"description,omitempty" json:"description,omitempty"`
	Argv        []string            `yaml:"argv,omitempty"        json:"argv,omitempty"`  // ← ADD
	Args        map[string]*ArgDef  `yaml:"args,omitempty"        json:"args,omitempty"`
	Returns     string              `yaml:"returns,omitempty"     json:"returns,omitempty"`
}
```

**Semantics:**
- `Argv` is a slice of Go `text/template` strings
- Each element is rendered at invocation time with `args` as the template data
- Example: `argv: ["-c", "{{ .count }}", "{{ .host }}"]`
- Only used when `Transport.Type == "native"`

### 2.3 Add `TransportNative` to `pkg/tool/tool.go`

**File:** `/Volumes/Projects/gert/pkg/tool/tool.go`

**Modification:** Add constant after line 12:

```go
const (
	TransportStdio   TransportType = "stdio"
	TransportJSONRPC TransportType = "stdio-jsonrpc"
	TransportMCP     TransportType = "mcp"
	TransportNative  TransportType = "native"  // ← ADD
)
```

### 2.4 Add `Actions map[string]*ToolAction` to `pkg/tool.ToolDef`

**File:** `/Volumes/Projects/gert/pkg/tool/tool.go`

**Modification:** Add `Actions` field to `ToolDef` struct (after line 35):

```go
// ToolDef — tool definition (matches schema.ToolDef shape)
type ToolDef struct {
	Name      string
	Source    string // "builtin://slack-notify", "tool://auth-service", etc.
	Transport TransportType
	Command   string
	Args      []string
	Env       map[string]string
	Actions   map[string]*ToolAction  // ← ADD: per-action definitions
}

// ToolAction mirrors schema.ToolAction for runtime use
type ToolAction struct {
	Description string
	Argv        []string            // Go text/template strings
	Args        map[string]*ArgDef
	Returns     string
}

// ArgDef mirrors schema.ArgDef for runtime use
type ArgDef struct {
	Type        string
	Required    bool
	Description string
	Default     any
}
```

**Rationale:**
- The runtime needs action definitions to do argv template rendering
- Currently `ToolDef` only has top-level `Command` and `Args` — no per-action metadata
- `Actions` maps action name → action definition (argv, args, returns)

---

## 3. New File: `internal/tool/native.go`

**File:** `/Volumes/Projects/gert/internal/tool/native.go` (create new)

```go
package tool

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"text/template"

	toolpkg "github.com/ormasoftchile/gert/pkg/tool"
)

// NativeCLITransport spawns a native binary with argv per invocation.
// Does NOT speak JSON protocol; captures stdout/stderr as plain text.
type NativeCLITransport struct{}

// Invoke runs the native CLI tool with rendered argv and returns captured output.
func (t *NativeCLITransport) Invoke(ctx context.Context, def toolpkg.ToolDef, action string, args map[string]any) (*toolpkg.ToolResult, error) {
	actionDef, ok := def.Actions[action]
	if !ok {
		return nil, fmt.Errorf("native tool %s: action %q not found", def.Name, action)
	}

	// Render argv templates with args as data
	renderedArgv, err := renderArgv(actionDef.Argv, args)
	if err != nil {
		return nil, fmt.Errorf("native tool %s action %s: render argv: %w", def.Name, action, err)
	}

	if len(renderedArgv) == 0 && len(actionDef.Argv) > 0 {
		return nil, fmt.Errorf("native tool %s action %s: argv rendered to empty (misconfiguration)", def.Name, action)
	}

	// Start the process
	proc, err := StartProcess(ctx, def.Command, renderedArgv, def.Env)
	if err != nil {
		return nil, fmt.Errorf("native tool %s: start process: %w", def.Name, err)
	}

	// No stdin write for native tools (they don't read JSON)
	_ = proc.stdin.Close()

	// Capture stdout/stderr concurrently
	var wg sync.WaitGroup
	wg.Add(2)

	var stdoutBuf []byte
	var stderrBuf []byte
	go func() {
		defer wg.Done()
		stdoutBuf, _ = io.ReadAll(proc.stdout)
	}()
	go func() {
		defer wg.Done()
		stderrBuf, _ = io.ReadAll(proc.stderr)
	}()

	waitErr := proc.Wait()
	wg.Wait()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	exitCode := 0
	if waitErr != nil {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("native tool %s: wait: %w", def.Name, waitErr)
		}
	}

	result := &toolpkg.ToolResult{
		ExitCode: exitCode,
		Stdout:   string(stdoutBuf),
		Stderr:   string(stderrBuf),
		Output:   nil, // Native tools emit plain text, not structured JSON
	}

	// Non-zero exit is an error for native tools
	if exitCode != 0 {
		return result, fmt.Errorf("native tool %s exited with code %d", def.Name, exitCode)
	}
	return result, nil
}

// Close is a no-op for native transport (stateless, one-shot invocations).
func (t *NativeCLITransport) Close() error {
	return nil
}

// renderArgv renders each argv element as a Go text/template with args as data.
func renderArgv(argv []string, args map[string]any) ([]string, error) {
	if len(argv) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(argv))
	for i, tmplStr := range argv {
		tmpl, err := template.New(fmt.Sprintf("argv[%d]", i)).Parse(tmplStr)
		if err != nil {
			return nil, fmt.Errorf("argv[%d]: parse template: %w", i, err)
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, args); err != nil {
			return nil, fmt.Errorf("argv[%d]: execute template: %w", i, err)
		}
		out = append(out, buf.String())
	}
	return out, nil
}
```

**Design Notes:**
- Follows the same pattern as `StdioTransport` (reuses `StartProcess`, concurrent IO reads)
- **No JSON protocol** — stdin is immediately closed, stdout/stderr captured as text
- Error if action not found or argv is empty (prevents silent misconfiguration)
- `renderArgv` uses Go `text/template` package (same as gert v2 expression evaluator)

---

## 4. Modified Files

### 4.1 `internal/tool/runtime.go`

**Change:** Add case for `TransportNative` in `Invoke` switch (after line 52):

```go
func (r *DefaultToolRuntime) Invoke(ctx context.Context, toolName string, action string, args map[string]any) (*toolpkg.ToolResult, error) {
	// ... existing lookup logic ...

	switch def.Transport {
	case toolpkg.TransportStdio:
		return (&StdioTransport{}).Invoke(ctx, *def, action, args)
	case toolpkg.TransportJSONRPC:
		return r.invokePersistent(ctx, toolName, *def, action, args, func() toolpkg.ToolTransport {
			return &JSONRPCTransport{}
		})
	case toolpkg.TransportMCP:
		return r.invokePersistent(ctx, toolName, *def, action, args, func() toolpkg.ToolTransport {
			return &MCPTransport{}
		})
	case toolpkg.TransportNative:  // ← ADD
		return (&NativeCLITransport{}).Invoke(ctx, *def, action, args)
	default:
		return nil, fmt.Errorf("tool runtime: unsupported transport %q", def.Transport)
	}
}
```

**Rationale:** Dispatch native tools to `NativeCLITransport` (stateless, like stdio).

---

### 4.2 `internal/tool/scan.go`

**Change 1:** Add `TransportNative` mapping in `mapTransport` (after line 108):

```go
func mapTransport(t schema.Transport) (toolpkg.TransportType, error) {
	switch t {
	case schema.TransportStdio:
		return toolpkg.TransportStdio, nil
	case schema.TransportJSONRPC:
		return toolpkg.TransportJSONRPC, nil
	case schema.TransportMCP:
		return toolpkg.TransportMCP, nil
	case schema.TransportNative:  // ← ADD
		return toolpkg.TransportNative, nil
	default:
		return "", fmt.Errorf("unsupported transport %q", t)
	}
}
```

**Change 2:** Copy `Actions` map from schema → runtime in `runtimeToolDef` (after line 97):

```go
func runtimeToolDef(def *schema.ToolDef) (toolpkg.ToolDef, error) {
	if def == nil {
		return toolpkg.ToolDef{}, fmt.Errorf("nil tool definition")
	}
	transport, err := mapTransport(def.Transport.Type)
	if err != nil {
		return toolpkg.ToolDef{}, err
	}

	// Convert schema actions to runtime actions
	actions := make(map[string]*toolpkg.ToolAction, len(def.Actions))
	for name, schemaAction := range def.Actions {
		if schemaAction == nil {
			continue
		}
		runtimeArgs := make(map[string]*toolpkg.ArgDef, len(schemaAction.Args))
		for argName, schemaArg := range schemaAction.Args {
			if schemaArg == nil {
				continue
			}
			runtimeArgs[argName] = &toolpkg.ArgDef{
				Type:        schemaArg.Type,
				Required:    schemaArg.Required,
				Description: schemaArg.Description,
				Default:     schemaArg.Default,
			}
		}
		actions[name] = &toolpkg.ToolAction{
			Description: schemaAction.Description,
			Argv:        schemaAction.Argv,
			Args:        runtimeArgs,
			Returns:     schemaAction.Returns,
		}
	}

	return toolpkg.ToolDef{
		Name:      def.Name,
		Source:    "tool://" + def.Name,
		Transport: transport,
		Command:   def.Transport.Command,
		Args:      def.Transport.Args,
		Env:       def.Transport.Env,
		Actions:   actions,  // ← NEW: copy actions
	}, nil
}
```

**Rationale:** The runtime needs action definitions to do argv rendering. Copy the full actions map from schema to runtime.

---

## 5. Tool YAML Format (v2)

Create a `tools/` directory at repo root for native CLI tool definitions.

### 5.1 `tools/ping.tool.yaml`

```yaml
apiVersion: tool/v2
name: ping
version: "1.0"
description: Send ICMP ECHO_REQUEST packets to test network reachability

transport:
  type: native
  command: ping

actions:
  check:
    description: Ping a host with a fixed packet count
    argv:
      - "-c"
      - "{{ .count }}"
      - "{{ .host }}"
    args:
      host:
        type: string
        required: true
        description: Hostname or IP address to ping
      count:
        type: string
        required: false
        default: "4"
        description: Number of ICMP packets to send
    returns: text

  check-timeout:
    description: Ping a host with a deadline per packet
    argv:
      - "-c"
      - "{{ .count }}"
      - "-W"
      - "{{ .timeout }}"
      - "{{ .host }}"
    args:
      host:
        type: string
        required: true
        description: Hostname or IP address to ping
      count:
        type: string
        required: false
        default: "4"
        description: Number of ICMP packets to send
      timeout:
        type: string
        required: false
        default: "5"
        description: Time in seconds to wait for each reply
    returns: text
```

---

### 5.2 `tools/curl.tool.yaml`

```yaml
apiVersion: tool/v2
name: curl
version: "1.0"
description: Transfer data from or to a server using HTTP/HTTPS

transport:
  type: native
  command: curl

actions:
  get:
    description: Perform an HTTP GET request
    argv:
      - "-s"
      - "-X"
      - "GET"
      - "{{ .url }}"
    args:
      url:
        type: string
        required: true
        description: URL to fetch
    returns: text

  post:
    description: Perform an HTTP POST request with JSON body
    argv:
      - "-s"
      - "-X"
      - "POST"
      - "-H"
      - "Content-Type: application/json"
      - "-d"
      - "{{ .body }}"
      - "{{ .url }}"
    args:
      url:
        type: string
        required: true
        description: URL to POST to
      body:
        type: string
        required: true
        description: JSON body to send
    returns: text

  head:
    description: Perform an HTTP HEAD request
    argv:
      - "-s"
      - "-I"
      - "{{ .url }}"
    args:
      url:
        type: string
        required: true
        description: URL to query
    returns: text

  download:
    description: Download a file to disk
    argv:
      - "-s"
      - "-o"
      - "{{ .output }}"
      - "{{ .url }}"
    args:
      url:
        type: string
        required: true
        description: URL to download from
      output:
        type: string
        required: true
        description: Local file path to save to
    returns: text
```

---

### 5.3 `tools/nslookup.tool.yaml`

```yaml
apiVersion: tool/v2
name: nslookup
version: "1.0"
description: Query DNS name servers for domain name or IP address mapping

transport:
  type: native
  command: nslookup

actions:
  lookup:
    description: Look up a hostname or IP using default DNS server
    argv:
      - "{{ .name }}"
    args:
      name:
        type: string
        required: true
        description: Hostname or IP address to query
    returns: text

  lookup-server:
    description: Look up a hostname using a specific DNS server
    argv:
      - "{{ .name }}"
      - "{{ .server }}"
    args:
      name:
        type: string
        required: true
        description: Hostname or IP address to query
      server:
        type: string
        required: true
        description: DNS server to query
    returns: text

  reverse:
    description: Perform reverse DNS lookup (PTR record)
    argv:
      - "-type=PTR"
      - "{{ .ip }}"
    args:
      ip:
        type: string
        required: true
        description: IP address to reverse lookup
    returns: text

  query-type:
    description: Query a specific DNS record type
    argv:
      - "-type={{ .type }}"
      - "{{ .name }}"
    args:
      name:
        type: string
        required: true
        description: Domain name to query
      type:
        type: string
        required: false
        default: "A"
        description: DNS record type (A, AAAA, MX, TXT, etc.)
    returns: text
```

---

## 6. Runbook `toolRefs:` Wiring

### 6.1 YAML Format in Runbooks

Tools are referenced in the runbook header via `toolRefs:`. Each ref has a `name` and a `path` (relative to the runbook file).

**Example:** `examples/network-check.runbook.yaml`

```yaml
apiVersion: runbook/v2
id: network-check
name: Network Connectivity Check

toolRefs:
  - name: ping
    path: ../../tools/ping.tool.yaml
  - name: nslookup
    path: ../../tools/nslookup.tool.yaml
  - name: curl
    path: ../../tools/curl.tool.yaml

flow:
  - step:
      id: ping-google
      type: tool
      title: Ping Google DNS
      tool: ping
      action: check
      args:
        host: "8.8.8.8"
        count: "3"

  - step:
      id: dns-lookup
      type: tool
      title: Resolve example.com
      tool: nslookup
      action: lookup
      args:
        name: "example.com"

  - step:
      id: http-check
      type: tool
      title: Fetch HTTP headers
      tool: curl
      action: head
      args:
        url: "https://example.com"
```

**Path resolution:** The `path:` is relative to the runbook file's directory. The loader resolves it to an absolute path before parsing the tool definition.

---

### 6.2 Loader Changes

**Context:** The runbook parser (`internal/parser/parser.go`) currently parses `ToolRefs` into the schema but does NOT resolve the paths or load the tool definitions.

**New requirement:** The adapter/planner layer must resolve `toolRefs` at load time and register them into the tool registry before execution.

**Recommended approach (in `internal/adapter/wire.go` or new file `internal/adapter/toolrefs.go`):**

```go
// resolveToolRefs loads tool definitions referenced in a runbook's toolRefs field.
// Returns a slice of ToolDef to register into the tool registry.
func resolveToolRefs(runbookPath string, refs []*schema.ToolRef) ([]toolpkg.ToolDef, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	runbookDir := filepath.Dir(runbookPath)
	defs := make([]toolpkg.ToolDef, 0, len(refs))
	for _, ref := range refs {
		if ref.Path == "" {
			continue // skip refs without path (might be builtin or remote)
		}
		// Resolve relative path
		absPath := filepath.Join(runbookDir, ref.Path)
		schemaDef, err := parseToolFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("resolve tool %s at %s: %w", ref.Name, ref.Path, err)
		}
		runtimeDef, err := runtimeToolDef(schemaDef)
		if err != nil {
			return nil, fmt.Errorf("convert tool %s: %w", ref.Name, err)
		}
		defs = append(defs, runtimeDef)
	}
	return defs, nil
}

// parseToolFile is copied from internal/tool/scan.go (or refactored to shared location)
func parseToolFile(path string) (*schema.ToolDef, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var def schema.ToolDef
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, err
	}
	if def.Name == "" {
		return nil, fmt.Errorf("tool definition missing name: %s", path)
	}
	return &def, nil
}
```

**Integration point (in `BuildEngineConfig` or planner):**

```go
// After parsing runbook, before building execution plan:
parsedRunbook, err := parser.Parse(ctx, runbookPath)
if err != nil {
	return err
}

// Resolve toolRefs and register them
toolDefs, err := resolveToolRefs(runbookPath, parsedRunbook.Runbook.ToolRefs)
if err != nil {
	return err
}
for _, def := range toolDefs {
	if err := toolRegistry.Register(def); err != nil {
		return fmt.Errorf("register tool %s: %w", def.Name, err)
	}
}
```

**Rationale:**
- Tools are declared at runbook scope (not global)
- Path resolution happens at parse time (fail fast if tool file missing)
- Tools are registered into the existing `ToolRegistry` before planner runs
- Planner's tool lookup (`Lookup(ctx, name, action)`) works unchanged

---

### 6.3 Alternative: Lazy Loading in Planner

If the planner already has a `ToolRegistry` reference, it could resolve toolRefs lazily when planning:

```go
// In planner.Plan()
for _, ref := range rb.ToolRefs {
	if ref.Path == "" {
		continue
	}
	toolDef, err := loadToolFromPath(runbookDir, ref.Path)
	if err != nil {
		return nil, err
	}
	_ = p.toolRegistry.Register(toolDef)
}
```

**Trade-off:** This couples the planner to filesystem I/O. Prefer eager loading in the adapter layer (cleaner separation).

---

## 7. Decision Log Entry

**File:** `.squad/decisions/inbox/ken-native-tool.md`

```markdown
# Decision: Native CLI Tool Transport

**Date:** 2025-04-24  
**Decider:** Ken (Architect)  
**Implementor:** Brian  
**Status:** Approved → Implementation Pending  

---

## Context

The gert v2 tool system supports three transports: `stdio`, `jsonrpc`, and `mcp`. All three expect tools to speak a JSON protocol (JSON request on stdin, JSON response on stdout).

Native CLI utilities like `ping`, `curl`, `nslookup` are argv-style commands that:
- Accept arguments via command-line flags (not JSON stdin)
- Write plain text output to stdout/stderr (not JSON)
- Return exit codes (not structured responses)

The v1 `.tool.yaml` format handled this via per-action `argv:` templates. The v2 schema does not have `argv` on `ToolAction`, and the runtime has no `native` transport type.

---

## Decision

Add a `native` transport type to gert v2 that supports native CLI tools via argv-style invocation.

**Schema changes:**
1. Add `TransportNative Transport = "native"` to `pkg/schema/tool.go`
2. Add `Argv []string` field to `schema.ToolAction` (Go text/template strings)
3. Add `TransportNative TransportType = "native"` to `pkg/tool/tool.go`
4. Add `Actions map[string]*ToolAction` to `pkg/tool.ToolDef` (runtime needs action metadata)

**Implementation:**
- New file `internal/tool/native.go` with `NativeCLITransport`
- `Invoke(ctx, def, action, args)` renders argv templates, spawns process, captures output
- No JSON protocol — stdin closed immediately, stdout/stderr captured as text
- Error if action not found or argv is empty (fail fast on misconfiguration)

**Tool definitions:**
- Create `tools/` directory at repo root
- Add `ping.tool.yaml`, `curl.tool.yaml`, `nslookup.tool.yaml` in v2 format
- Format: top-level `name:`, `transport: {type: native, command: <binary>}`, `actions:` with `argv:` and `args:`

**Runbook integration:**
- Runbooks reference tools via `toolRefs: [{name: ping, path: ../../tools/ping.tool.yaml}]`
- Adapter layer resolves refs at load time, registers tools before execution
- Path is relative to runbook file

---

## Rationale

**Why not extend `stdio` transport?**
- `stdio` has a contract: JSON request on stdin, JSON response on stdout
- Native tools break that contract (they don't read stdin, they write plain text)
- Mixing two protocols in one transport type creates ambiguity

**Why `text/template` for argv?**
- Consistent with gert v2 expression evaluator (already uses `text/template`)
- Simple, predictable, no new syntax to learn
- Supports basic variable substitution (no complex logic needed)

**Why `Actions` on runtime `ToolDef`?**
- The runtime needs argv templates to render arguments
- Schema → runtime conversion must carry action metadata
- Alternative (store schema in runtime) couples runtime to schema types

**Why `tools/` at repo root?**
- Centralizes common CLI utilities (reusable across runbooks)
- Matches v1 pattern (`gert-for-reference/tools/`)
- Enables future registry/distribution (tools can be published separately)

---

## Consequences

**Positive:**
- ✅ Native CLI tools (ping, curl, dig, etc.) are first-class gert tools
- ✅ Consistent with v1 pattern (argv templates, per-action configuration)
- ✅ No subprocess overhead (direct exec, no wrapper scripts)
- ✅ Clean separation from JSON-protocol tools (explicit transport type)

**Negative:**
- ⚠️ Template errors surface at runtime (not parse time)
- ⚠️ No structured output parsing (tools emit plain text)
- ⚠️ Argv rendering is string-based (no type safety for arguments)

**Mitigations:**
- Template parse errors fail fast (first invocation, not silent)
- Plain text output is acceptable (governance/evidence already captures text)
- Arg type validation happens in schema (type: string, type: int, etc.)

---

## Alternatives Considered

### 1. Wrapper Script Transport

**Idea:** Add a `script` transport that wraps CLI tools in bash/python scripts that emit JSON.

**Rejected because:**
- Adds indirection (spawn script → spawn tool)
- Requires users to write wrapper scripts (boilerplate)
- Shell injection risk (dynamic argv construction in bash)

### 2. Extend `stdio` with `protocol: "text"` flag

**Idea:** Add `transport: {type: stdio, protocol: text}` to opt out of JSON.

**Rejected because:**
- Mixes two incompatible protocols in one transport type
- `stdio` name implies JSON protocol (existing contract)
- `native` is clearer intent (argv + text output)

### 3. Inline argv in runbook steps

**Idea:** Allow steps to declare `argv: ["-c", "{{ .count }}"]` inline.

**Rejected because:**
- Duplicates argv across every step (DRY violation)
- No single source of truth for tool contract
- Tool definitions enable reuse + versioning

---

## Implementation Checklist

- [ ] Add `TransportNative` to `pkg/schema/tool.go`
- [ ] Add `Argv []string` to `schema.ToolAction`
- [ ] Add `TransportNative` to `pkg/tool/tool.go`
- [ ] Add `Actions map[string]*ToolAction` to `pkg/tool.ToolDef`
- [ ] Implement `internal/tool/native.go` (NativeCLITransport)
- [ ] Update `internal/tool/runtime.go` dispatch (add native case)
- [ ] Update `internal/tool/scan.go` (mapTransport + copy Actions)
- [ ] Create `tools/ping.tool.yaml`
- [ ] Create `tools/curl.tool.yaml`
- [ ] Create `tools/nslookup.tool.yaml`
- [ ] Implement `resolveToolRefs()` in adapter layer
- [ ] Wire toolRefs resolution into `BuildEngineConfig` or planner
- [ ] Write integration test: load runbook with toolRefs, invoke native tool
- [ ] Update runbook JSON schema to accept `argv:` on actions
- [ ] Document native transport in `docs/tools.md`

---

## Follow-Up Work (Deferred)

**v2.1 or later:**
- Template validation at parse time (compile templates during tool load)
- Output parsing hints (regex capture groups, JSON detection)
- Argv type coercion (convert int args to strings automatically)
- Tool catalog/registry (publish tools to shared index)

---

## References

- v1 tool format: `/Volumes/Projects/gert-for-reference/tools/ping.tool.yaml`
- v2 stdio transport: `internal/tool/stdio.go`
- v2 tool schema: `pkg/schema/tool.go`
- v2 tool runtime: `pkg/tool/tool.go`, `internal/tool/runtime.go`
```

---

## 8. Summary

This spec introduces a `native` transport type for gert v2 that enables native CLI tools (ping, curl, nslookup) to be invoked via argv-style arguments without requiring a JSON protocol.

**Key components:**
1. **Schema additions:** `TransportNative`, `Argv []string`, `Actions map`
2. **New transport:** `NativeCLITransport` in `internal/tool/native.go`
3. **Runtime wiring:** Dispatch in `runtime.go`, scan changes in `scan.go`
4. **Tool definitions:** `tools/ping.tool.yaml`, `curl.tool.yaml`, `nslookup.tool.yaml`
5. **Runbook integration:** `toolRefs:` in runbook header, loader resolves paths

**Implementation order:**
1. Schema changes (pkg/schema, pkg/tool)
2. Runtime support (scan.go, runtime.go, native.go)
3. Tool definitions (tools/*.tool.yaml)
4. Loader wiring (resolveToolRefs in adapter)
5. Integration test (load runbook with toolRefs, invoke native tool)

**Verification:**
- Compile succeeds (type checker validates new fields)
- Integration test loads ping.tool.yaml and invokes `check` action
- Output contains ICMP statistics (proves argv rendering + process spawn works)

---

**Next:** Brian to implement. Barbara to add unit tests for `renderArgv` and integration test for native tool invocation.
