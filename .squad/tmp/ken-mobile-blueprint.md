# Go v2 Mobile Implementation Blueprint

**Author:** Ken (Software Architect)  
**Date:** 2026-04-26  
**Target:** Brian (Go Implementation Lead)  

## Mission

This blueprint guides implementation of all mobile-related changes to the GERT v2 Go codebase. These changes enable the system to:

1. Track which client (CLI, server, mobile-ios, mobile-android) initiated a run
2. Define platform-specific tool implementations in tool definitions
3. Validate compiled kits against target platforms
4. Ingest mobile-submitted runs via HTTP API
5. Load and validate platform kits from a registry

## Prerequisites

Before starting implementation, ensure:
- You have reviewed the current v2 codebase structure
- All existing tests pass
- You understand the trace event system (`pkg/trace/event.go`)
- You understand tool definitions (`pkg/schema/tool.go`)

---

## Change 1: Add `client` Field to `run/started` Event

**Goal:** Record which client type initiated a run in the trace for audit and debugging.

### Files to Modify

#### `pkg/trace/event.go`

No code changes needed — the event system uses dynamic payloads (`json.RawMessage`). The `client` field will be part of the payload map.

#### `internal/engine/engine.go`

**Location:** Line ~270 (where `run/started` event is emitted in `Next()` method)

**Current code:**
```go
h.emitEventLocked(stepCtx, trace.EventKindRunStarted, map[string]any{
	"run_id":       h.run.ID,
	"runbook_path": h.run.Plan.RunbookPath,
	"actor":        h.run.Actor,
	"mode":         string(h.run.Mode),
})
```

**Change to:**
```go
h.emitEventLocked(stepCtx, trace.EventKindRunStarted, map[string]any{
	"run_id":       h.run.ID,
	"runbook_path": h.run.Plan.RunbookPath,
	"actor":        h.run.Actor,
	"mode":         string(h.run.Mode),
	"client":       h.run.Client,
})
```

#### `pkg/engine/run.go`

**Location:** Add `Client` field to the `Run` struct (line ~88)

**Add field:**
```go
// Client identifies which client type initiated the run.
// Valid values: "cli", "server", "mobile-ios", "mobile-android"
Client string
```

**Location:** Update `NewRun` function (line ~132) to accept client from opts

**Current code:**
```go
run := &Run{
	ID:               id,
	Status:           RunStatusPending,
	Plan:             plan,
	Vars:             make(map[string]any),
	StepResults:      make(map[string]*StepResult),
	CurrentStepIndex: -1,
	Actor:            opts.Actor,
	Mode:             opts.Mode,
}
```

**Change to:**
```go
run := &Run{
	ID:               id,
	Status:           RunStatusPending,
	Plan:             plan,
	Vars:             make(map[string]any),
	StepResults:      make(map[string]*StepResult),
	CurrentStepIndex: -1,
	Actor:            opts.Actor,
	Mode:             opts.Mode,
	Client:           opts.Client,
}
```

**Location:** Add `Client` field to `RunOptions` struct (line ~38)

**Add field:**
```go
// Client identifies which client type is initiating the run.
// Valid values: "cli", "server", "mobile-ios", "mobile-android"
Client string
```

#### `cmd/gert/run.go`

**Location:** Set client field when creating run options (line ~100+)

**Find where `engine.RunOptions{}` is constructed and add:**
```go
opts := engine.RunOptions{
	Mode:        mode,
	Actor:       actor,
	Vars:        varsMap,
	ScenarioDir: scenarioDir,
	Store:       store,
	OnEvent:     onEventCallback,
	Client:      "cli", // CLI client
}
```

#### `internal/serve/rpc.go`

**Location:** Find the RPC handler for `run.start` and add client field

**Add:**
```go
opts := engine.RunOptions{
	// ... existing fields ...
	Client: "server", // server client
}
```

### Test Updates

Create test: `internal/engine/engine_test.go`

**Add test function:**
```go
func TestEngine_RunStartedIncludesClient(t *testing.T) {
	t.Parallel()
	
	plan := &enginepkg.ExecutionPlan{
		RunID:       "test-run",
		RunbookPath: "test.yaml",
		Steps:       []enginepkg.ResolvedStep{},
		Tools:       map[string]*schema.ToolDef{},
		Providers:   map[string]*schema.ProviderDef{},
	}
	
	store := newMockStore(t)
	eng := New(enginepkg.EngineConfig{Store: store})
	
	opts := enginepkg.RunOptions{
		Mode:   enginepkg.RunModeReal,
		Actor:  "test-actor",
		Client: "cli",
	}
	
	handle, err := eng.Start(context.Background(), plan, opts)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	
	// Advance to trigger run/started
	_, _ = handle.Next(context.Background())
	
	// Verify event payload includes client
	events := store.Events()
	if len(events) == 0 {
		t.Fatal("expected at least one event")
	}
	
	startedEvent := events[0]
	if startedEvent.Kind != "run/started" {
		t.Errorf("first event: want run/started, got %s", startedEvent.Kind)
	}
	
	client, ok := startedEvent.Payload["client"]
	if !ok {
		t.Error("run/started event missing 'client' field")
	}
	if client != "cli" {
		t.Errorf("client: want 'cli', got %v", client)
	}
}
```

### Validation

Valid client values (document in code):
- `"cli"` — gert CLI command
- `"server"` — gert serve RPC
- `"mobile-ios"` — iOS SDK
- `"mobile-android"` — Android SDK

---

## Change 2: Per-Platform `impl` Blocks in Tool Definitions

**Goal:** Allow tools to declare platform-specific implementations while maintaining a single tool definition.

### Files to Modify

#### `pkg/schema/tool.go`

**Location:** Modify `ToolDef` struct (line ~13)

**Current:**
```go
type ToolDef struct {
	APIVersion  string                  `yaml:"apiVersion"            json:"apiVersion"`
	Name        string                  `yaml:"name"                  json:"name"`
	Version     string                  `yaml:"version,omitempty"     json:"version,omitempty"`
	Description string                  `yaml:"description,omitempty" json:"description,omitempty"`
	Transport   TransportConfig         `yaml:"transport"             json:"transport"`
	Actions     map[string]*ToolAction  `yaml:"actions"               json:"actions"`
	Metadata    map[string]string       `yaml:"metadata,omitempty"    json:"metadata,omitempty"`
}
```

**Change to:**
```go
type ToolDef struct {
	APIVersion  string                  `yaml:"apiVersion"            json:"apiVersion"`
	Name        string                  `yaml:"name"                  json:"name"`
	Version     string                  `yaml:"version,omitempty"     json:"version,omitempty"`
	Description string                  `yaml:"description,omitempty" json:"description,omitempty"`
	
	// Transport is the default/legacy transport config (required for backward compatibility)
	Transport   TransportConfig         `yaml:"transport"             json:"transport"`
	
	// Impl maps platform name to platform-specific implementation.
	// Keys: "ios", "android", "server", etc.
	// If present, overrides Transport for that platform.
	Impl        map[string]*PlatformImpl `yaml:"impl,omitempty"        json:"impl,omitempty"`
	
	Actions     map[string]*ToolAction  `yaml:"actions"               json:"actions"`
	Metadata    map[string]string       `yaml:"metadata,omitempty"    json:"metadata,omitempty"`
	
	// RequiresCapabilities lists capability tokens this tool needs (e.g., "capability/camera")
	RequiresCapabilities []string        `yaml:"requires-capabilities,omitempty" json:"requires-capabilities,omitempty"`
}
```

**Location:** Add new `PlatformImpl` struct (after `Transport` const block, line ~54)

**Add:**
```go
// PlatformImpl describes how a tool is implemented on a specific platform.
type PlatformImpl struct {
	// Transport is the platform-specific transport type.
	// Examples: "native-sdk", "stdio", "jsonrpc"
	Transport string `yaml:"transport" json:"transport"`
	
	// Handler is the platform-specific handler identifier.
	// For native-sdk on iOS: "GertSDK.Camera.capture"
	// For native-sdk on Android: "com.gert.platform.tools.CameraCaptureTool"
	Handler   string `yaml:"handler" json:"handler"`
}
```

### Files to Create

#### `pkg/tool/platform.go` (new file)

```go
package tool

import (
	"fmt"
	
	"github.com/ormasoftchile/gert/pkg/schema"
)

// ValidatePlatformAvailability checks if a tool has an implementation for the target platform.
// Returns nil if tool is available on the platform.
// Returns error if tool is missing impl for target platform or if impl is invalid.
func ValidatePlatformAvailability(toolDef *schema.ToolDef, targetPlatform string) error {
	if toolDef == nil {
		return fmt.Errorf("tool definition is nil")
	}
	
	// If tool has no platform-specific impls, it's assumed to work everywhere (backward compat)
	if len(toolDef.Impl) == 0 {
		return nil
	}
	
	// Check if platform impl exists
	platformImpl, ok := toolDef.Impl[targetPlatform]
	if !ok {
		return &PlatformUnavailableError{
			ToolName: toolDef.Name,
			Platform: targetPlatform,
			Available: availablePlatforms(toolDef),
		}
	}
	
	// Validate impl fields
	if platformImpl.Transport == "" {
		return fmt.Errorf("tool %s: platform %s impl missing transport", toolDef.Name, targetPlatform)
	}
	if platformImpl.Handler == "" {
		return fmt.Errorf("tool %s: platform %s impl missing handler", toolDef.Name, targetPlatform)
	}
	
	return nil
}

// PlatformUnavailableError indicates a tool is not available on the target platform.
type PlatformUnavailableError struct {
	ToolName  string
	Platform  string
	Available []string
}

func (e *PlatformUnavailableError) Error() string {
	return fmt.Sprintf("tool %s not available on platform %s (available: %v)", 
		e.ToolName, e.Platform, e.Available)
}

func availablePlatforms(toolDef *schema.ToolDef) []string {
	platforms := make([]string, 0, len(toolDef.Impl))
	for platform := range toolDef.Impl {
		platforms = append(platforms, platform)
	}
	return platforms
}
```

### Test Updates

Create test: `pkg/tool/platform_test.go`

```go
package tool

import (
	"testing"
	
	"github.com/ormasoftchile/gert/pkg/schema"
)

func TestValidatePlatformAvailability(t *testing.T) {
	tests := []struct {
		name        string
		toolDef     *schema.ToolDef
		platform    string
		expectError bool
	}{
		{
			name: "tool with no impl blocks (legacy) - always available",
			toolDef: &schema.ToolDef{
				Name: "legacy-tool",
			},
			platform:    "ios",
			expectError: false,
		},
		{
			name: "tool available on ios",
			toolDef: &schema.ToolDef{
				Name: "camera.capture",
				Impl: map[string]*schema.PlatformImpl{
					"ios": {
						Transport: "native-sdk",
						Handler:   "GertSDK.Camera.capture",
					},
				},
			},
			platform:    "ios",
			expectError: false,
		},
		{
			name: "tool not available on android",
			toolDef: &schema.ToolDef{
				Name: "camera.capture",
				Impl: map[string]*schema.PlatformImpl{
					"ios": {
						Transport: "native-sdk",
						Handler:   "GertSDK.Camera.capture",
					},
				},
			},
			platform:    "android",
			expectError: true,
		},
		{
			name: "impl missing transport",
			toolDef: &schema.ToolDef{
				Name: "camera.capture",
				Impl: map[string]*schema.PlatformImpl{
					"ios": {
						Handler: "GertSDK.Camera.capture",
					},
				},
			},
			platform:    "ios",
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePlatformAvailability(tt.toolDef, tt.platform)
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}
```

---

## Change 3: Compiler `--target` Flag

**Goal:** Add compiler validation to ensure all tools in a kit have implementations for the target platform.

### Context

GERT v2 doesn't currently have a `gert compile` command. This is a *new* command to add to the CLI.

The compiler's job is to:
1. Load a kit directory (containing `manifest.json` and `tools/*.tool.yaml`)
2. Validate all tools have `impl` blocks for the target platform
3. Emit a `manifest.json` recording the target platform

### Files to Create

#### `cmd/gert/compile.go` (new file)

```go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	
	"gopkg.in/yaml.v3"
	
	"github.com/ormasoftchile/gert/pkg/schema"
	"github.com/ormasoftchile/gert/pkg/tool"
)

// runCompile implements the `gert compile` command.
// Validates a platform kit against target platforms.
func runCompile(args []string) int {
	fs := flag.NewFlagSet("compile", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	
	targetFlag := fs.String("target", "", "Target platform: ios, android, or mobile (both)")
	outputDir := fs.String("output", "", "Output directory for manifest.json (defaults to kit dir)")
	
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitValidation
	}
	
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: gert compile <kit-directory> --target <platform>")
		return exitValidation
	}
	
	kitDir := fs.Arg(0)
	target := *targetFlag
	
	if target == "" {
		fmt.Fprintln(os.Stderr, "error: --target is required")
		return exitValidation
	}
	
	// Expand "mobile" to both ios and android
	var platforms []string
	switch target {
	case "ios":
		platforms = []string{"ios"}
	case "android":
		platforms = []string{"android"}
	case "mobile":
		platforms = []string{"ios", "android"}
	default:
		fmt.Fprintf(os.Stderr, "error: invalid target %q (valid: ios, android, mobile)\n", target)
		return exitValidation
	}
	
	// Load manifest
	manifestPath := filepath.Join(kitDir, "manifest.json")
	manifestData, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading manifest: %v\n", err)
		return exitRuntime
	}
	
	var manifest KitManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing manifest: %v\n", err)
		return exitRuntime
	}
	
	// Load all tools
	toolsDir := filepath.Join(kitDir, "tools")
	toolFiles, err := filepath.Glob(filepath.Join(toolsDir, "*.tool.yaml"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error scanning tools: %v\n", err)
		return exitRuntime
	}
	
	var validationErrors []string
	
	for _, toolFile := range toolFiles {
		toolData, err := ioutil.ReadFile(toolFile)
		if err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("%s: read error: %v", filepath.Base(toolFile), err))
			continue
		}
		
		var toolDef schema.ToolDef
		if err := yaml.Unmarshal(toolData, &toolDef); err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("%s: parse error: %v", filepath.Base(toolFile), err))
			continue
		}
		
		// Validate availability on each target platform
		for _, platform := range platforms {
			if err := tool.ValidatePlatformAvailability(&toolDef, platform); err != nil {
				validationErrors = append(validationErrors, fmt.Sprintf("%s: %v", filepath.Base(toolFile), err))
			}
		}
	}
	
	if len(validationErrors) > 0 {
		fmt.Fprintln(os.Stderr, "Validation errors:")
		for _, e := range validationErrors {
			fmt.Fprintf(os.Stderr, "  - %s\n", e)
		}
		return exitValidation
	}
	
	// Update manifest with target
	manifest.Target = platforms
	
	// Write output manifest
	outDir := *outputDir
	if outDir == "" {
		outDir = kitDir
	}
	outPath := filepath.Join(outDir, "manifest.json")
	
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error encoding manifest: %v\n", err)
		return exitRuntime
	}
	
	if err := ioutil.WriteFile(outPath, manifestJSON, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing manifest: %v\n", err)
		return exitRuntime
	}
	
	fmt.Printf("✅ Kit %s validated for platform(s): %s\n", manifest.Name, strings.Join(platforms, ", "))
	fmt.Printf("Manifest written to: %s\n", outPath)
	
	return exitSuccess
}

// KitManifest is the manifest.json structure for platform kits.
type KitManifest struct {
	Name                 string   `json:"name"`
	Version              string   `json:"version"`
	Description          string   `json:"description"`
	Target               []string `json:"target"`
	ProvidesCapabilities []string `json:"provides-capabilities,omitempty"`
	Requires             []string `json:"requires,omitempty"`
}
```

#### `cmd/gert/main.go` — Register compile command

**Location:** Line ~17 (in the switch statement)

**Add case:**
```go
case "compile":
	os.Exit(runCompile(os.Args[2:]))
```

**Location:** Update `printUsage()` function (line ~54)

**Add line:**
```go
fmt.Fprintln(os.Stderr, "  compile   Validate a platform kit")
```

### Test Updates

Create test: `cmd/gert/compile_test.go`

```go
package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestRunCompile_ValidKit(t *testing.T) {
	// Setup test kit directory
	tmpDir, err := ioutil.TempDir("", "gert-test-kit-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	// Write manifest
	manifestJSON := `{
  "name": "test-kit",
  "version": "1.0.0",
  "description": "Test kit",
  "target": []
}`
	if err := ioutil.WriteFile(filepath.Join(tmpDir, "manifest.json"), []byte(manifestJSON), 0644); err != nil {
		t.Fatal(err)
	}
	
	// Create tools directory
	toolsDir := filepath.Join(tmpDir, "tools")
	if err := os.Mkdir(toolsDir, 0755); err != nil {
		t.Fatal(err)
	}
	
	// Write a tool with ios impl
	toolYAML := `name: test.tool
version: "1.0"
apiVersion: gert.dev/v2
transport:
  type: stdio
  command: /bin/echo
impl:
  ios:
    transport: native-sdk
    handler: Test.Handler
  android:
    transport: native-sdk
    handler: com.test.Handler
actions:
  run: {}
`
	if err := ioutil.WriteFile(filepath.Join(toolsDir, "test.tool.yaml"), []byte(toolYAML), 0644); err != nil {
		t.Fatal(err)
	}
	
	// Run compile
	exitCode := runCompile([]string{tmpDir, "--target", "mobile"})
	if exitCode != exitSuccess {
		t.Errorf("expected exitSuccess, got %d", exitCode)
	}
	
	// Verify manifest was updated
	manifestData, err := ioutil.ReadFile(filepath.Join(tmpDir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	
	// Should contain target field now
	if !contains(string(manifestData), `"target"`) {
		t.Error("manifest should contain target field")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) && 
		(s == substr || len(s) > len(substr) && 
		 (s[:len(substr)] == substr || contains(s[1:], substr)))
}
```

---

## Change 4: Run Ingest API

**Goal:** Add HTTP endpoints to receive mobile-submitted run traces and attachments.

### Background

Mobile clients will:
1. Execute runs locally (using local engine or SDK)
2. Capture trace events (JSONL format)
3. POST the complete trace to server via `/api/v1/runs/ingest`
4. POST any attachments (photos, etc.) via `/api/v1/runs/{run-id}/attachments/{sha256}`

### Files to Modify

#### `internal/serve/server.go`

**Location:** In the `routes()` method (line ~98), add new handlers:

**Add:**
```go
s.mux.HandleFunc("POST /api/v1/runs/ingest", s.handleIngestRun)
s.mux.HandleFunc("POST /api/v1/runs/{runID}/attachments/{sha256}", s.handleIngestAttachment)
```

### Files to Create

#### `internal/serve/ingest.go` (new file)

```go
package serve

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	
	"github.com/ormasoftchile/gert/pkg/trace"
)

// handleIngestRun accepts a streaming JSONL trace from a mobile client
// and reconstructs the run in the server's run store.
//
// Request body: newline-delimited JSON (JSONL) trace events
// Response: 200 OK with JSON body {"run_id": "...", "status": "ingested"}
func (s *Server) handleIngestRun(w http.ResponseWriter, r *http.Request) {
	// Auth check (use existing JWT middleware)
	// For now, just proceed
	
	scanner := bufio.NewScanner(r.Body)
	defer r.Body.Close()
	
	var events []trace.TraceEvent
	var runID string
	
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		
		var event trace.TraceEvent
		if err := json.Unmarshal(line, &event); err != nil {
			http.Error(w, fmt.Sprintf("line %d: invalid JSON: %v", lineNum, err), http.StatusBadRequest)
			return
		}
		
		// Validate event structure
		if event.EventID == "" || event.RunID == "" || event.Kind == "" {
			http.Error(w, fmt.Sprintf("line %d: missing required fields", lineNum), http.StatusBadRequest)
			return
		}
		
		if runID == "" {
			runID = event.RunID
		} else if event.RunID != runID {
			http.Error(w, fmt.Sprintf("line %d: run_id mismatch (expected %s, got %s)", lineNum, runID, event.RunID), http.StatusBadRequest)
			return
		}
		
		events = append(events, event)
	}
	
	if err := scanner.Err(); err != nil {
		http.Error(w, fmt.Sprintf("error reading request body: %v", err), http.StatusBadRequest)
		return
	}
	
	if len(events) == 0 {
		http.Error(w, "no events in trace", http.StatusBadRequest)
		return
	}
	
	// Write events to run store
	// The store should create the trace file if it doesn't exist
	runDir := filepath.Join(s.cfg.RunsDir, runID)
	tracePath := filepath.Join(runDir, "trace.jsonl")
	
	// Create trace writer
	traceFile, err := s.cfg.Platform.OpenAppend(tracePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("error opening trace file: %v", err), http.StatusInternalServerError)
		return
	}
	defer traceFile.Close()
	
	// Write each event
	encoder := json.NewEncoder(traceFile)
	for _, event := range events {
		if err := encoder.Encode(event); err != nil {
			http.Error(w, fmt.Sprintf("error writing event: %v", err), http.StatusInternalServerError)
			return
		}
	}
	
	// Respond with success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"run_id": runID,
		"status": "ingested",
		"events": len(events),
	})
}

// handleIngestAttachment accepts a single evidence attachment
// and stores it in the run's evidence directory.
//
// URL params: runID, sha256
// Request body: raw binary attachment data
// Response: 200 OK with JSON body {"sha256": "...", "stored": true}
func (s *Server) handleIngestAttachment(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("runID")
	sha256 := r.PathValue("sha256")
	
	if runID == "" || sha256 == "" {
		http.Error(w, "missing runID or sha256", http.StatusBadRequest)
		return
	}
	
	// Validate sha256 format (64 hex chars)
	if len(sha256) != 64 {
		http.Error(w, "invalid sha256 format", http.StatusBadRequest)
		return
	}
	
	// Create evidence directory
	evidenceDir := filepath.Join(s.cfg.RunsDir, runID, "evidence")
	if err := os.MkdirAll(evidenceDir, 0755); err != nil {
		http.Error(w, fmt.Sprintf("error creating evidence directory: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Write attachment
	attachmentPath := filepath.Join(evidenceDir, sha256)
	outFile, err := os.Create(attachmentPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("error creating attachment file: %v", err), http.StatusInternalServerError)
		return
	}
	defer outFile.Close()
	
	written, err := io.Copy(outFile, r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("error writing attachment: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Respond with success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"sha256": sha256,
		"stored": true,
		"bytes":  written,
	})
}
```

**Note:** This code assumes `s.cfg.RunsDir` and `s.cfg.Platform` are available. You may need to add these to `ServerConfig`.

#### `pkg/serve/config.go` — Add RunsDir and Platform to ServerConfig

**Location:** Find `ServerConfig` struct and add:

```go
// RunsDir is the directory where run state and traces are stored.
// Defaults to ".runbook/runs"
RunsDir string

// Platform abstracts OS-specific behavior (for OpenAppend, etc.)
Platform platform.Platform
```

### Test Updates

Create test: `internal/serve/ingest_test.go`

```go
package serve

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	
	"github.com/ormasoftchile/gert/pkg/platform"
	"github.com/ormasoftchile/gert/pkg/trace"
)

func TestHandleIngestRun(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "gert-ingest-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	// Create server
	s := &Server{
		cfg: servepkg.ServerConfig{
			RunsDir:  tmpDir,
			Platform: platform.Real(),
		},
	}
	
	// Create JSONL trace body
	events := []trace.TraceEvent{
		{
			EventID:   "evt-1",
			RunID:     "run-123",
			RunbookID: "rb-1",
			Timestamp: "2026-04-26T10:00:00Z",
			Kind:      trace.EventKindRunStarted,
			Sequence:  1,
			Payload:   json.RawMessage(`{"actor":"mobile-user","client":"mobile-ios"}`),
		},
		{
			EventID:   "evt-2",
			RunID:     "run-123",
			RunbookID: "rb-1",
			Timestamp: "2026-04-26T10:00:05Z",
			Kind:      trace.EventKindRunCompleted,
			Sequence:  2,
			Payload:   json.RawMessage(`{"status":"success"}`),
		},
	}
	
	var buf bytes.Buffer
	for _, ev := range events {
		data, _ := json.Marshal(ev)
		buf.Write(data)
		buf.WriteByte('\n')
	}
	
	// Make request
	req := httptest.NewRequest("POST", "/api/v1/runs/ingest", &buf)
	w := httptest.NewRecorder()
	
	s.handleIngestRun(w, req)
	
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}
	
	// Verify trace file was written
	tracePath := filepath.Join(tmpDir, "run-123", "trace.jsonl")
	if _, err := os.Stat(tracePath); os.IsNotExist(err) {
		t.Error("trace file was not created")
	}
}
```

---

## Change 5: Platform Kit Registry

**Goal:** Define a registry format for platform kits and implement loading/validation.

### Files to Create

#### `pkg/platform/registry.go` (new file)

```go
package platform

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// RegistryEntry describes a single platform kit in the registry.
type RegistryEntry struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Path        string   `json:"path"`
	Target      []string `json:"target"`
	Description string   `json:"description,omitempty"`
}

// Registry manages the platform kit registry.
type Registry struct {
	path    string
	entries map[string]RegistryEntry
}

// LoadRegistry loads the platform kit registry from disk.
// If the registry file doesn't exist, returns an empty registry.
func LoadRegistry(path string) (*Registry, error) {
	reg := &Registry{
		path:    path,
		entries: make(map[string]RegistryEntry),
	}
	
	data, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		// Empty registry is OK
		return reg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read registry: %w", err)
	}
	
	var entries []RegistryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}
	
	for _, entry := range entries {
		reg.entries[entry.Name] = entry
	}
	
	return reg, nil
}

// Lookup returns the registry entry for a kit by name.
func (r *Registry) Lookup(name string) (RegistryEntry, bool) {
	entry, ok := r.entries[name]
	return entry, ok
}

// Register adds or updates a kit in the registry.
func (r *Registry) Register(entry RegistryEntry) error {
	if entry.Name == "" {
		return fmt.Errorf("kit name is required")
	}
	if entry.Path == "" {
		return fmt.Errorf("kit path is required")
	}
	
	r.entries[entry.Name] = entry
	return r.save()
}

// All returns all registered kits.
func (r *Registry) All() []RegistryEntry {
	entries := make([]RegistryEntry, 0, len(r.entries))
	for _, entry := range r.entries {
		entries = append(entries, entry)
	}
	return entries
}

// save persists the registry to disk.
func (r *Registry) save() error {
	entries := r.All()
	
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode registry: %w", err)
	}
	
	// Ensure directory exists
	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create registry directory: %w", err)
	}
	
	if err := ioutil.WriteFile(r.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}
	
	return nil
}

// DefaultRegistryPath returns the default path for the platform kit registry.
// Typically: ~/.gert/platform-kits/registry.json
func DefaultRegistryPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".gert", "platform-kits", "registry.json")
}

// InitBuiltinRegistry ensures the built-in gert-mobile-platform kit is registered.
func InitBuiltinRegistry() error {
	regPath := DefaultRegistryPath()
	reg, err := LoadRegistry(regPath)
	if err != nil {
		return err
	}
	
	// Check if gert-mobile-platform already registered
	if _, ok := reg.Lookup("gert-mobile-platform"); ok {
		return nil
	}
	
	// Register built-in mobile platform kit
	// In a real implementation, this would download or reference a known location
	entry := RegistryEntry{
		Name:        "gert-mobile-platform",
		Version:     "0.1.0",
		Path:        "https://github.com/ormasoftchile/gert-mobile-platform",
		Target:      []string{"ios", "android"},
		Description: "Shared mobile capability tools for gert runbooks",
	}
	
	return reg.Register(entry)
}
```

### Files to Modify

#### `cmd/gert/compile.go` — Use registry for validation

**Location:** In `runCompile`, before loading manifest, add registry check:

```go
// Load platform registry
reg, err := platform.LoadRegistry(platform.DefaultRegistryPath())
if err != nil {
	fmt.Fprintf(os.Stderr, "warning: failed to load platform registry: %v\n", err)
}

// Optional: verify kit is in registry
// (This can be enforced or just advisory)
```

### Test Updates

Create test: `pkg/platform/registry_test.go`

```go
package platform

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestRegistry_RegisterAndLookup(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "gert-registry-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	regPath := filepath.Join(tmpDir, "registry.json")
	
	reg, err := LoadRegistry(regPath)
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}
	
	// Register a kit
	entry := RegistryEntry{
		Name:    "test-kit",
		Version: "1.0.0",
		Path:    "/path/to/kit",
		Target:  []string{"ios"},
	}
	
	if err := reg.Register(entry); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	
	// Lookup
	found, ok := reg.Lookup("test-kit")
	if !ok {
		t.Fatal("kit not found after registration")
	}
	
	if found.Version != "1.0.0" {
		t.Errorf("version: want 1.0.0, got %s", found.Version)
	}
	
	// Verify registry file was written
	if _, err := os.Stat(regPath); os.IsNotExist(err) {
		t.Error("registry file was not created")
	}
	
	// Reload and verify persistence
	reg2, err := LoadRegistry(regPath)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	
	found2, ok := reg2.Lookup("test-kit")
	if !ok {
		t.Fatal("kit not found after reload")
	}
	
	if found2.Path != "/path/to/kit" {
		t.Errorf("path: want /path/to/kit, got %s", found2.Path)
	}
}
```

---

## Integration Checklist

After implementing all changes, verify:

- [ ] All existing tests still pass: `go test ./...`
- [ ] New tests pass: `go test ./pkg/tool ./pkg/platform ./cmd/gert ./internal/serve`
- [ ] CLI commands work:
  - [ ] `gert run <runbook> --trace /tmp/trace.jsonl` (client=cli recorded)
  - [ ] `gert compile <kit-dir> --target ios` (validation works)
  - [ ] `gert compile <kit-dir> --target mobile` (validates both platforms)
- [ ] Server endpoints respond:
  - [ ] `POST /api/v1/runs/ingest` (accepts JSONL trace)
  - [ ] `POST /api/v1/runs/{run-id}/attachments/{sha256}` (stores file)
- [ ] Registry operations:
  - [ ] `platform.LoadRegistry()` creates registry if missing
  - [ ] `platform.InitBuiltinRegistry()` registers gert-mobile-platform

---

## Key Invariants

1. **Client field:** Must be one of: `"cli"`, `"server"`, `"mobile-ios"`, `"mobile-android"`
2. **Platform impl validation:** Tools with `impl` blocks MUST have an entry for the target platform
3. **JSONL integrity:** Ingest endpoint MUST validate run_id consistency across all events
4. **Attachment SHA256:** Must be exactly 64 hexadecimal characters
5. **Registry atomicity:** Registry writes must be atomic (write temp file, rename)

---

## Error Handling Patterns

- **Validation errors:** Return `exitValidation` (code 2), print to stderr
- **Runtime errors:** Return `exitRuntime` (code 3), print to stderr
- **HTTP errors:** Use appropriate status codes (400 for client errors, 500 for server errors)
- **Missing impl:** Return `PlatformUnavailableError` with list of available platforms

---

## Migration Notes

All changes are backward compatible:
- Existing tools without `impl` blocks continue to work (assumed platform-agnostic)
- Existing `run/started` events will just be missing `client` field (optional field)
- New endpoints don't break existing RPC API

---

## Questions for Implementation

If you encounter issues, check:
1. Does `pkg/serve/config.go` exist? If not, ServerConfig may be in a different file
2. Are there existing patterns for HTTP route registration you should follow?
3. Does the codebase use a different JSON library than `encoding/json`?
4. Are there existing middleware patterns for auth that ingest should use?

**End of Blueprint**
