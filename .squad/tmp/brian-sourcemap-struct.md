# Go Package Design: sourcemap.yaml Support

**Author:** Brian (Go Programmer)  
**Date:** 2026-04-22  
**Purpose:** Define formal Go types and validation for Kit Traceability Layer 2 (sourcemap.yaml sidecar)

---

## 1. Package Location

```
v2/pkg/kit/sourcemap/
```

**Rationale:** Kit-specific tooling belongs under `v2/pkg/kit/`. The sourcemap is a kit artifact (not core GERT schema), so it gets its own sub-package for encapsulation.

---

## 2. Go Struct Definition

```go
// Package sourcemap provides types and validation for Kit Traceability Layer 2 source maps.
// Source maps are compiler-emitted sidecars that map lowered step IDs back to kit source.
package sourcemap

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// SourceMap is the root structure of a {runbook-name}.sourcemap.yaml file.
// It maps compiled step IDs back to their kit source definitions.
type SourceMap struct {
	// Version is the source map format version (currently "1")
	Version string `yaml:"version"`

	// Kit is the fully-qualified kit ID with version (e.g., "vacation.gert.io/v0")
	Kit string `yaml:"kit"`

	// Runbook is the name of the lowered runbook this map corresponds to
	Runbook string `yaml:"runbook"`

	// LoweredAt is the ISO8601 timestamp when this runbook was lowered
	LoweredAt time.Time `yaml:"lowered_at"`

	// Entries maps step IDs to their source metadata
	Entries map[string]Entry `yaml:"entries"`
}

// Entry describes a single step's provenance in kit source.
type Entry struct {
	// Kind is the kit concept type (DayFlavor, MealSlot, StayTemplate, Activity, etc.)
	Kind string `yaml:"kind"`

	// Name is the instance name from kit source
	Name string `yaml:"name"`

	// SourceFile is the relative path to the kit source YAML file
	SourceFile string `yaml:"source_file"`

	// SourceLine is the line number where the concept is defined (optional but recommended)
	SourceLine int `yaml:"source_line,omitempty"`

	// --- Optional fields (concept-specific metadata) ---

	// Slot identifies which slot this step belongs to (e.g., "afternoon-activity")
	Slot string `yaml:"slot,omitempty"`

	// Day identifies which day this step is part of
	Day int `yaml:"day,omitempty"`

	// Role identifies the sub-role of a step (e.g., "weather-branch", "rain-fallback")
	Role string `yaml:"role,omitempty"`

	// Field identifies which template field this step validates (e.g., "credits.spa")
	Field string `yaml:"field,omitempty"`

	// TimeWindow specifies the time window for slots (e.g., "18:00-20:00")
	TimeWindow string `yaml:"time_window,omitempty"`

	// ParentConcept references the parent template or concept name
	ParentConcept string `yaml:"parent_concept,omitempty"`

	// ParentStep references the parent step ID (for nested steps)
	ParentStep string `yaml:"parent_step,omitempty"`
}

// ValidationError represents a single validation failure.
type ValidationError struct {
	// Field is the path to the invalid field (e.g., "entries.vacation.day.2.afternoon-activity.kind")
	Field string

	// Message is a human-readable error description
	Message string

	// Severity is "error" or "warning"
	Severity string
}

// Error implements the error interface for ValidationError
func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s (%s)", e.Severity, e.Field, e.Message)
}
```

---

## 3. Key Functions

### 3.1 Load

```go
// Load reads and parses a sourcemap.yaml file.
// Returns an error if the file cannot be read or parsed.
func Load(path string) (*SourceMap, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read sourcemap file %q: %w", path, err)
	}

	var sm SourceMap
	if err := yaml.Unmarshal(data, &sm); err != nil {
		return nil, fmt.Errorf("failed to parse sourcemap YAML: %w", err)
	}

	return &sm, nil
}
```

### 3.2 Validate

```go
// Validate checks a SourceMap against the Kit Traceability Contract (5 rules).
// Returns a slice of ValidationErrors (empty if valid).
func Validate(sm *SourceMap) []ValidationError {
	var errs []ValidationError

	// Rule 2: Required top-level fields
	if sm.Version == "" {
		errs = append(errs, ValidationError{
			Field:    "version",
			Message:  "version field is required",
			Severity: "error",
		})
	}
	if sm.Kit == "" {
		errs = append(errs, ValidationError{
			Field:    "kit",
			Message:  "kit field is required (must be fully-qualified kit ID)",
			Severity: "error",
		})
	}
	if sm.Runbook == "" {
		errs = append(errs, ValidationError{
			Field:    "runbook",
			Message:  "runbook field is required",
			Severity: "error",
		})
	}
	if sm.LoweredAt.IsZero() {
		errs = append(errs, ValidationError{
			Field:    "lowered_at",
			Message:  "lowered_at timestamp is required",
			Severity: "error",
		})
	}

	// Rule 4: Version check
	if sm.Version != "1" {
		errs = append(errs, ValidationError{
			Field:    "version",
			Message:  fmt.Sprintf("unsupported version %q (expected \"1\")", sm.Version),
			Severity: "error",
		})
	}

	// Validate each entry
	for stepID, entry := range sm.Entries {
		// Rule 1: Step ID structure (delegated to validateStepID helper)
		if err := validateStepID(stepID, sm.Kit); err != nil {
			errs = append(errs, ValidationError{
				Field:    fmt.Sprintf("entries.%s", stepID),
				Message:  err.Error(),
				Severity: "error",
			})
		}

		// Required entry fields
		if entry.Kind == "" {
			errs = append(errs, ValidationError{
				Field:    fmt.Sprintf("entries.%s.kind", stepID),
				Message:  "kind is required for every entry",
				Severity: "error",
			})
		}
		if entry.Name == "" {
			errs = append(errs, ValidationError{
				Field:    fmt.Sprintf("entries.%s.name", stepID),
				Message:  "name is required for every entry",
				Severity: "error",
			})
		}
		if entry.SourceFile == "" {
			errs = append(errs, ValidationError{
				Field:    fmt.Sprintf("entries.%s.source_file", stepID),
				Message:  "source_file is required for every entry",
				Severity: "error",
			})
		}

		// Warning if source_line is missing (recommended but not required)
		if entry.SourceLine == 0 {
			errs = append(errs, ValidationError{
				Field:    fmt.Sprintf("entries.%s.source_line", stepID),
				Message:  "source_line is missing (recommended for debugging)",
				Severity: "warning",
			})
		}
	}

	// Rule 3: Determinism check (cannot be enforced at parse time, only via test)
	// This is a compile-time contract verified by comparing two compilations.
	// We can add a note in the validation summary.

	return errs
}

// validateStepID checks if a step ID follows the Kit Traceability Convention:
// {kit-prefix}.{concept-kind}.{concept-name}.{sub-element}
func validateStepID(stepID, kitPrefix string) error {
	// Extract kit prefix from kit ID (e.g., "vacation.gert.io/v0" → "vacation")
	// This is a simplified check; full implementation would parse the kit ID properly.
	
	// Check if step ID starts with expected prefix
	// Full validation would check:
	// 1. Segments are kebab-case
	// 2. At least 3 segments (prefix, kind, name)
	// 3. No disallowed characters
	
	// Placeholder for now — full implementation in v2.1
	if stepID == "" {
		return fmt.Errorf("step ID cannot be empty")
	}
	
	return nil
}
```

---

## 4. Validation Rules (Kit Traceability Contract)

The `Validate()` function enforces the 5-rule Kit Traceability Contract from §4.10.4:

### Rule 1: Step IDs MUST follow structured naming convention
- **Check:** Parse step ID to verify `{kit-prefix}.{concept-kind}.{concept-name}.{sub-element}` format
- **Implementation:** `validateStepID()` helper validates segment structure and kebab-case
- **Error:** "Step ID %q does not follow {kit-prefix}.{kind}.{name}.{sub} convention"

### Rule 2: Source map MUST include required fields
- **Check:** Verify `version`, `kit`, `runbook`, `lowered_at`, `entries` are all non-empty
- **Implementation:** Direct field checks in `Validate()`
- **Error:** "Field %q is required" for each missing field

### Rule 3: Source map MUST be deterministic
- **Check:** Cannot be enforced at parse time — requires comparing two compilations
- **Implementation:** Validation returns a note: "Determinism is enforced via test (compile twice, diff outputs)"
- **Error:** N/A (verified by test suite, not runtime validation)

### Rule 4: Source map MUST version the kit
- **Check:** `version` field is `"1"` and `kit` field is a valid fully-qualified ID
- **Implementation:** Version check + kit ID format validation
- **Error:** "Unsupported version %q (expected \"1\")" or "Kit ID %q is invalid"

### Rule 5: Compiler SHOULD populate `meta` fields (if GERT v2 supports it)
- **Check:** This is about the lowered runbook, not the source map
- **Implementation:** Not validated by this package (checked in runbook schema validation)
- **Error:** N/A (outside sourcemap scope)

---

## 5. Integration with `gert-kit lint`

The `gert-kit` CLI (kit compiler tooling) would invoke this package for validation:

```go
// In gert-kit/cmd/vacation/lint.go

package main

import (
	"fmt"
	"os"

	"github.com/ormasoftchile/gert/v2/pkg/kit/sourcemap"
	"github.com/spf13/cobra"
)

var lintCmd = &cobra.Command{
	Use:   "lint [sourcemap-file]",
	Short: "Validate a sourcemap.yaml file against the Kit Traceability Contract",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		checkSchema, _ := cmd.Flags().GetBool("check")

		sm, err := sourcemap.Load(path)
		if err != nil {
			return fmt.Errorf("failed to load sourcemap: %w", err)
		}

		if checkSchema {
			errs := sourcemap.Validate(sm)
			if len(errs) == 0 {
				fmt.Println("✓ Source map is valid")
				return nil
			}

			// Report errors
			fmt.Printf("✗ Source map validation failed (%d issues):\n", len(errs))
			for _, e := range errs {
				fmt.Printf("  [%s] %s: %s\n", e.Severity, e.Field, e.Message)
			}

			// Exit with non-zero if any errors (warnings OK)
			hasErrors := false
			for _, e := range errs {
				if e.Severity == "error" {
					hasErrors = true
					break
				}
			}
			if hasErrors {
				os.Exit(1)
			}
		}

		return nil
	},
}

func init() {
	lintCmd.Flags().Bool("check", false, "Validate against Kit Traceability Contract schema")
}
```

**Usage:**
```bash
# Compile the kit
gert-kit vacation compile templates/floripa-5day.yaml --output build/

# Lint the source map
gert-kit vacation lint build/stay-floripa.sourcemap.yaml --check schema

# Expected output:
# ✓ Source map is valid
# (or)
# ✗ Source map validation failed (3 issues):
#   [error] version: version field is required
#   [warning] entries.vacation.day.2.afternoon-activity.source_line: source_line is missing (recommended for debugging)
#   [error] entries.vacation.credits.spa.check.kind: kind is required for every entry
```

---

## 6. Error Reporting

### Error Types

1. **Parse Errors** (from `Load()`)
   - File not found
   - YAML syntax errors
   - Type mismatches (e.g., `lowered_at` not a timestamp)

2. **Validation Errors** (from `Validate()`)
   - Missing required fields
   - Invalid field values
   - Schema version mismatches
   - Step ID convention violations

3. **Warnings** (from `Validate()`)
   - Missing optional but recommended fields (e.g., `source_line`)
   - Deprecated features

### User-Facing Error Messages

All errors include:
- **Field path** — Exact location of the issue (e.g., `entries.vacation.day.2.afternoon-activity.kind`)
- **Severity** — `error` or `warning`
- **Message** — Human-readable description with guidance

Example:
```
✗ Source map validation failed (4 issues):
  [error] version: version field is required
  [error] kit: kit field is required (must be fully-qualified kit ID)
  [error] entries.vacation.day.2.afternoon-activity.kind: kind is required for every entry
  [warning] entries.vacation.day.2.afternoon-activity.source_line: source_line is missing (recommended for debugging)
```

---

## 7. Testing Strategy

### 7.1 Unit Tests (`sourcemap_test.go`)

```go
func TestLoad_ValidSourceMap(t *testing.T) {
	sm, err := sourcemap.Load("testdata/valid-sourcemap.yaml")
	require.NoError(t, err)
	assert.Equal(t, "1", sm.Version)
	assert.Equal(t, "vacation.gert.io/v0", sm.Kit)
	assert.Len(t, sm.Entries, 5)
}

func TestValidate_MissingVersion(t *testing.T) {
	sm := &sourcemap.SourceMap{
		Kit:       "vacation.gert.io/v0",
		Runbook:   "stay-floripa",
		LoweredAt: time.Now(),
		Entries:   map[string]sourcemap.Entry{},
	}
	errs := sourcemap.Validate(sm)
	assert.Len(t, errs, 1)
	assert.Equal(t, "version", errs[0].Field)
	assert.Equal(t, "error", errs[0].Severity)
}

func TestValidate_EntryMissingKind(t *testing.T) {
	sm := &sourcemap.SourceMap{
		Version:   "1",
		Kit:       "vacation.gert.io/v0",
		Runbook:   "stay-floripa",
		LoweredAt: time.Now(),
		Entries: map[string]sourcemap.Entry{
			"vacation.day.2.afternoon-activity": {
				// Kind missing
				Name:       "beach-day",
				SourceFile: "templates/beach-day.yaml",
			},
		},
	}
	errs := sourcemap.Validate(sm)
	assert.Contains(t, errs[0].Field, "kind")
	assert.Equal(t, "error", errs[0].Severity)
}
```

### 7.2 Integration Test (Determinism)

```bash
# scripts/test-kit-traceability.sh
#!/usr/bin/env bash
set -e

# Compile twice
gert-kit vacation compile templates/floripa-5day.yaml --output build-v1/
gert-kit vacation compile templates/floripa-5day.yaml --output build-v2/

# Compare outputs (must be identical)
diff -u build-v1/stay-floripa.sourcemap.yaml build-v2/stay-floripa.sourcemap.yaml

echo "✓ Kit Traceability Contract Rule 3 (determinism) verified"
```

---

## 8. Implementation Roadmap

### Phase 1: Package Skeleton (v2.1)
- Create `v2/pkg/kit/sourcemap/` package
- Define `SourceMap` and `Entry` structs
- Implement `Load()` function
- Write unit tests for parsing

### Phase 2: Validation (v2.1)
- Implement `Validate()` function
- Implement `validateStepID()` helper
- Write unit tests for all 5 contract rules
- Add test fixtures in `testdata/`

### Phase 3: CLI Integration (v2.1)
- Add `gert-kit vacation lint` command
- Wire up `--check schema` flag to call `Validate()`
- Add determinism integration test to CI

### Phase 4: Compiler Integration (v2.2)
- Modify kit compiler to emit `sourcemap.yaml` during lowering
- Populate all entry fields from kit source AST
- Add source line tracking during parsing

---

## 9. Open Questions

1. **Step ID validation depth:** Should `validateStepID()` parse the full kit ID to extract the prefix, or rely on a parameter? (Proposed: parameter for simplicity.)

2. **Entry extensibility:** Should we allow arbitrary extra fields in `Entry` for kit-specific metadata? (Proposed: Yes, via `yaml:",inline"` for future-proofing.)

3. **Version evolution:** When sourcemap format version "2" arrives, should we support multiple versions in one package or separate packages? (Proposed: Same package, version-dispatch in `Validate()`.)

---

## 10. Summary

This design provides:
- **Formal Go types** for sourcemap.yaml using `gopkg.in/yaml.v3`
- **Validation logic** enforcing the 5-rule Kit Traceability Contract
- **CLI integration** via `gert-kit lint --check schema`
- **Clear error reporting** with field paths and severity levels
- **Test strategy** covering unit tests and determinism verification

**Implementation scope:** ~200 lines of Go code + ~150 lines of tests. Can be delivered in v2.1 alongside Step.Meta support.

**Next steps:** File decision, append to history, await v2.1 implementation milestone.
