# Compiler Contract: gert-domain-home → GERT Core

**Date:** 2026-04-24  
**Architect:** Ken  
**Domain:** gert-domain-home (household maintenance tracking)  
**Target:** GERT v2 Core (github.com/ormasoftchile/gert/v2)

---

## Executive Summary

The compiler transforms domain model types (`Property`, `Routine`, `IncidentTemplate`, `Delegation`) into GERT v2 runbook YAML documents. **Option B (YAML output)** is the cleanest boundary: the compiler produces runbook YAML strings that GERT's existing parser can load without creating a compile-time Go dependency on v2 schema types.

---

## GERT v2 Core Inventory

### What Exists (Verified)

**Module:** `github.com/ormasoftchile/gert/v2`  
**Location:** `/Users/cristianormazabal/Projects/gert/v2/`

#### 1. Schema Package (`v2/pkg/schema/`)

Core types the parser produces:

- **`Runbook`** — Top-level runbook document (runbook.go:4)
  - Fields: `APIVersion`, `ID`, `Name`, `Kind`, `Description`, `Vars`, `Inputs`, `Outputs`, `Flow`, `Governance`, etc.
  - `Kind`: `mitigation`, `reference`, `composable`, `rca` (runbook.go:33-40)

- **`Step`** — Unified step struct (step.go:39)
  - Common: `ID`, `Type`, `Title`, `Subtitle`, `When`, `Timeout`, `Delay`, `Retry`, `ContinueOnFail`, `Capture`, `Export`, `Contract`
  - Type discriminator: `StepType` (step.go:3-35)
  - Step types available:
    - `cli` — Shell command execution
    - `tool` — Tool invocation (via .tool.yaml)
    - `include` — Nested runbook inclusion
    - `choice` — Single/multi-select user prompt
    - `decision` — Branching decision prompt
    - `collector` — Multi-field form input
    - `branch` — Conditional branching
    - `iterate` — Loop over collection
    - `parallel` — Parallel execution
    - `approve` — Approval gate
    - `assert` — Assertion validation
    - `compensate` — Error compensation
    - `wait_for_event` — External event wait
    - `end` — Terminal step

- **Type-specific specs** (steps.go):
  - `CLISpec` — command, args, run, shell, env, workdir, stdin
  - `ToolCallSpec` → `ToolInvocation` (name, action, args, version)
  - `ChoiceSpec` → `ChoiceOption[]` (label, value, hint)
  - `DecisionSpec` → `DecisionRoute[]` (label, runbook, goto, hint)
  - `CollectorSpec` → `CollectorField[]` (name, type, label, required, validation)
  - `ApproveSpec` → `ApprovalGate` (mode, roles, pool, required, timeout)
  - Many others

- **Flow control nodes** (runbook.go:89-95, steps.go:143-171):
  - `FlowNode` — Union of Step, Iterate, Parallel
  - `IterateNode` — (id, over, as, max, until, collect, steps)
  - `ParallelNode` — (id, branches, join)

- **Governance types** (runbook.go:66-87):
  - `GovernanceConfig` — require_approval, rules, allow/deny commands, deny_env_vars, redact
  - `GovernanceRule` — effects, action, min_approvers
  - `RedactRule` — pattern (regex), replace

#### 2. Engine Package (`v2/pkg/engine/`)

Runtime execution types:

- **`ExecutionPlan`** (run.go:247) — The immutable plan the planner produces:
  - Fields: `RunID`, `RunbookPath`, `Steps[]`, `Tools map[string]*ToolDef`, `Providers map[string]*ProviderDef`, `Governance`, `Metadata`
  - `Steps` are `ResolvedStep[]` (run.go:259): `ID`, `Kind`, `Spec`, `Capture`, `Depth`, `Origin`

- **`RunHandle`** (run.go:15) — Control surface for active run
  - Methods: `Next(ctx) (*StepResult, error)`, `Approve()`, `SubmitEvidence()`, `Cancel()`, `State()`, `Events()`

- **`StepSpec` interface** (stepspec.go:6) — Implemented by all step types from schema package
  - Method: `StepKind() string`

- **`Run`** (run.go:88) — Mutable internal run state
  - Fields: `ID`, `Status`, `Plan *ExecutionPlan`, `Vars map[string]any`, `StepResults map[string]*StepResult`, `CurrentStepIndex`, `StartedAt`, `CompletedAt`, `Error`, `Sequence`, `Actor`, `Mode`

- **`StepResult`** (run.go:150) — Result of step execution
  - Fields: `StepID`, `Status`, `Outcome`, `Output map[string]any`, `Vars map[string]any`, `StartedAt`, `CompletedAt`, `DurationMs`, `Error`, `Evidence[]`

- **`RunMode`** (run.go:49): `real`, `dry-run`, `replay`
- **`RunStatus`** (run.go:75): `pending`, `running`, `waiting`, `completed`, `failed`, `cancelled`
- **`StepStatus`** (run.go:184): `pending`, `running`, `completed`, `failed`, `skipped`, `waiting`, `denied`

#### 3. Planner Package (`v2/pkg/planner/`)

Planning layer (not directly used by compiler, but context):

- **`Planner` interface** (planner.go:16)
  - `Plan(ctx, *parser.ParsedRunbook) (*engine.ExecutionPlan, error)`
- **`RunbookLoader` interface** (planner.go:26)
  - `Load(ctx, path string) (*parser.ParsedRunbook, error)`
- **`ToolRegistry` interface** (planner.go:36)
  - `Lookup(ctx, name, action string) (*schema.ToolDef, error)`

#### 4. Parser Package (`v2/pkg/parser/`)

YAML ingestion:

- **`Parser` interface** (parser.go:35)
  - `Parse(ctx, path string) (*ParsedRunbook, error)`
  - `ParseBytes(ctx, source []byte) (*ParsedRunbook, error)`
- **`ParsedRunbook`** (parser.go:14)
  - `Source string`, `Runbook *schema.Runbook`, `ResolvedIncludes []*ParsedRunbook`, `Warnings []ParseWarning`

#### 5. Governance Package (`v2/pkg/governance/`)

Policy enforcement:

- **`GovernancePolicy` interface** (policy.go:5)
  - `CheckCommand(command string) (allowed bool, matchedRule string)`
  - `FilterEnvVars(vars) (filtered, blocked)`
  - `RedactionPatterns() []*RedactionPattern`
- **`RedactionPattern`** (policy.go:40): `Pattern`, `Replacement`

#### 6. Other Packages

- **`v2/pkg/input`** — `PromptProvider` interface for user input (choice, decision, form)
- **`v2/pkg/evidence`** — Evidence recording types
- **`v2/pkg/eventbus`** — Event dispatch
- **`v2/pkg/trace`** — Trace event writing/reading
- **`v2/pkg/extension`** — Extension manifest/capability
- **`v2/pkg/tool`** — Tool definition types
- **`v2/pkg/provider`** — Provider definition types
- **`v2/pkg/platform`** — Platform abstraction
- **`v2/pkg/serve`** — HTTP/SSE server

### What DOES NOT Exist

GERT v2 core **does not** provide:

- ❌ Timer/cron scheduling system
- ❌ Recurring run triggers
- ❌ Delegation/permission management
- ❌ Away-mode or user role tracking
- ❌ Inventory/stock tracking
- ❌ Notification system

These are **domain-specific concerns** that the home domain must implement outside the compiler (e.g., scheduler daemon, notification service).

---

## Output Strategy

### Decision: **Option B — Compiler Produces YAML**

The compiler will produce YAML bytes (as `string` or `[]byte`) representing a valid GERT v2 runbook. The consumer (scheduler, CLI, web service) passes this YAML to GERT's `Parser.ParseBytes()` or writes it to disk for `Parser.Parse()`.

#### Why Option B?

| Criterion | Option A (Go Structs) | Option B (YAML) | Option C (IR) |
|-----------|----------------------|-----------------|---------------|
| **Dependency weight** | Heavy (requires v2 pkg import) | Light (only YAML lib) | N/A (no IR exists) |
| **Boundary clarity** | Blurred (tight coupling) | Clean (text contract) | N/A |
| **Testability** | Schema changes break tests | Golden file diffs clear | N/A |
| **Versioning** | Breaking changes painful | YAML schema stable | N/A |
| **GERT design intent** | Parser is the ingestion layer | ✅ Parser is designed for YAML | N/A |
| **Future GERT changes** | Requires recompile/update | Resilient to Go struct changes | N/A |

**Option A rejected:** Creates tight coupling. Every schema type change in v2 requires domain recompilation. The domain becomes a v2 internal client rather than an external consumer.

**Option C not viable:** GERT v2 has no intermediate representation (IR) type. The `ExecutionPlan` is produced by the planner after parsing, not consumed directly.

**Option B advantages:**
1. Domain model is **independent** of GERT v2 Go types
2. Compiler output can be **version-checked** (e.g., reject if GERT v2 < 2.1)
3. YAML is human-readable for debugging
4. Golden file testing is idiomatic (compare `expected.runbook.yaml` vs actual)
5. GERT's parser handles all validation — compiler doesn't duplicate schema rules

---

## Compilation Boundary

**Input:** Domain model Go types (`model.Routine`, `model.IncidentTemplate`, `model.Delegation`)  
**Output:** YAML string conforming to GERT v2 runbook schema  
**Consumer:** GERT v2 `parser.Parser.ParseBytes()`

```
┌─────────────────────┐
│  .home.yaml file    │
│  (domain model)     │
└──────────┬──────────┘
           │
           v
┌─────────────────────┐
│  model.Loader       │
│  pkg/loader/        │
└──────────┬──────────┘
           │
           v
┌─────────────────────┐
│  model.Property     │   ← In-memory domain types
│  model.Routine[]    │
│  model.Incident[]   │
│  model.Delegation   │
└──────────┬──────────┘
           │
           v
┌─────────────────────┐
│  compiler.Compile*  │   ← THIS LAYER
│  pkg/compiler/      │
└──────────┬──────────┘
           │
           v  YAML string
           │
┌─────────────────────┐
│  parser.ParseBytes  │   ← GERT v2 core ingestion
│  v2/pkg/parser/     │
└──────────┬──────────┘
           │
           v
┌─────────────────────┐
│  schema.Runbook     │   ← GERT v2 validated AST
└──────────┬──────────┘
           │
           v
┌─────────────────────┐
│  planner.Plan()     │   ← Resolve includes, tools, etc.
└──────────┬──────────┘
           │
           v
┌─────────────────────┐
│ engine.ExecutionPlan│   ← Ready for runtime
└─────────────────────┘
```

---

## Compiler Function Signatures

### Package: `github.com/ormasoftchile/gert-domain-home/pkg/compiler`

```go
package compiler

import (
	"context"
	"fmt"
	"github.com/ormasoftchile/gert-domain-home/pkg/model"
	"gopkg.in/yaml.v3"
)

// CompileRoutine transforms a domain routine into a GERT v2 runbook YAML string.
//
// Output runbook structure:
// - apiVersion: v2
// - id: routine-{routine.ID}
// - name: {routine.Label}
// - kind: composable (routines are reusable composable workflows)
// - flow: single collector step with evidence capture + optional approval
//
// The runbook does NOT contain scheduling logic. Scheduling is the caller's responsibility
// (e.g., cron daemon calls `gert run` when the routine is due).
//
// Returns error if:
// - prop is nil
// - routine is nil
// - routine.ID is empty
// - routine.Evidence.Type is invalid
func CompileRoutine(ctx context.Context, prop *model.Property, routine *model.Routine) (string, error)

// CompileIncidentTemplate transforms a domain incident template into a GERT v2 runbook YAML string.
//
// Output runbook structure:
// - apiVersion: v2
// - id: incident-{template.ID}
// - name: {template.Label}
// - kind: mitigation (incidents are reactive mitigation workflows)
// - inputs: zone (string), asset (string, optional)
// - flow: steps compiled from template.Steps[]
//   - human_task → collector step (multi-field form with evidence)
//   - decision → decision step (routes to step IDs via goto)
//   - parallel → parallel node
//
// Returns error if:
// - prop is nil
// - template is nil
// - template.ID is empty
// - template.Steps is empty
// - any step has invalid type or missing required fields
// - dependency graph (DependsOn) has cycles
func CompileIncidentTemplate(ctx context.Context, prop *model.Property, template *model.IncidentTemplate) (string, error)

// CompileDelegation transforms a domain delegation into a GERT v2 governance policy YAML fragment.
//
// Output: NOT a runbook. This produces a governance config snippet that can be merged
// into runbooks or loaded as a policy overlay.
//
// Structure:
// - governance:
//     rules:
//       - effects: [execute]
//         action: require-approval (if delegate needs approval for specific actions)
//     allow_commands: [...]
//     deny_commands: [...] (if delegate has restricted permissions)
//
// Returns error if:
// - prop is nil
// - delegation is nil
// - delegation.Active.From/To are invalid dates
// - delegation.Delegate.Contact is not E.164 format
func CompileDelegation(ctx context.Context, prop *model.Property, delegation *model.Delegation) (string, error)

// CompileProperty compiles the entire property file into a runbook catalog.
//
// Output: A directory manifest (YAML or JSON) listing all compiled runbooks:
//   {
//     "routines": [
//       {"id": "routine-clean-gutters", "path": "./routines/clean-gutters.runbook.yaml"},
//       ...
//     ],
//     "incidents": [
//       {"id": "incident-pipe-leak", "path": "./incidents/pipe-leak.runbook.yaml"},
//       ...
//     ],
//     "delegation": {
//       "policy_path": "./delegation.policy.yaml",
//       "active_from": "2026-07-01",
//       "active_to": "2026-07-15"
//     }
//   }
//
// Does NOT write files to disk. Caller decides how to persist the catalog and runbooks.
//
// Returns error if any sub-compilation fails.
func CompileProperty(ctx context.Context, prop *model.Property) (*Catalog, error)

// Catalog is the compilation output for a full property file.
type Catalog struct {
	// Routines maps routine ID to compiled YAML.
	Routines map[string]string

	// Incidents maps incident template ID to compiled YAML.
	Incidents map[string]string

	// DelegationPolicy is the compiled delegation governance fragment (may be empty).
	DelegationPolicy string

	// Metadata holds property-level metadata.
	Metadata CatalogMetadata
}

// CatalogMetadata holds property-level metadata.
type CatalogMetadata struct {
	PropertyName     string
	CompiledAt       string // ISO8601 timestamp
	CompilerVersion  string // e.g., "v0.1.0"
	GERTMinVersion   string // e.g., "v2.0.0"
	RoutineCount     int
	IncidentCount    int
	DelegationActive bool
}
```

---

## Compilation Examples

### Example 1: Routine Compilation

**Input (domain model):**

```yaml
# property.home.yaml excerpt
routines:
  - id: clean-gutters
    label: Clean roof gutters
    description: Remove leaves and debris from gutters and downspouts
    zone: exterior
    cadence:
      every: 90d
    evidence:
      type: photo
      prompt: Take a photo showing clean gutters and downspouts
    executor:
      role: homeowner
    notifications:
      remind_hours_before: 24
      escalate_hours_overdue: 48
```

**Output (compiled YAML):**

```yaml
$schema: https://gert.run/schemas/v2/runbook.schema.json
apiVersion: v2
id: routine-clean-gutters
name: Clean roof gutters
kind: composable
description: Remove leaves and debris from gutters and downspouts

inputs:
  zone:
    type: string
    required: true
    description: Zone where the routine is performed
    default: exterior
  due_date:
    type: string
    required: false
    description: Scheduled due date (ISO8601)

outputs:
  completed_at:
    type: string
    value: "{{ now() }}"
  evidence_hash:
    type: string
    value: "{{ steps.capture_evidence.evidence[0].hash }}"

metadata:
  domain: home
  routine_id: clean-gutters
  executor_role: homeowner
  cadence: 90d
  remind_hours_before: "24"
  escalate_hours_overdue: "48"

flow:
  - step:
      id: capture_evidence
      type: collector
      title: Clean roof gutters
      subtitle: Remove leaves and debris from gutters and downspouts
      prompt: Take a photo showing clean gutters and downspouts
      fields:
        - name: photo
          type: image
          label: Photo of clean gutters
          required: true
          hint: Take a photo showing clean gutters and downspouts
        - name: notes
          type: text
          label: Additional notes (optional)
          required: false
          multiline: true
      export:
        - completed_at
        - photo
```

**Key mappings:**

| Domain Field | GERT Field | Notes |
|--------------|------------|-------|
| `routine.ID` | `id` (prefixed "routine-") | Namespacing prevents ID collision |
| `routine.Label` | `name`, `title` | Both runbook name and step title |
| `routine.Description` | `description`, `subtitle` | Context for both |
| `routine.Evidence.Type` | `fields[].type` | `photo` → `image`, `note` → `text`, `checklist` → multiple `boolean` fields |
| `routine.Evidence.Prompt` | `prompt`, `fields[].hint` | User-facing instruction |
| `routine.Zone` | `inputs.zone.default` | Pre-populated but overridable |
| `routine.Executor.Role` | `metadata.executor_role` | Hint for scheduler/UI (not governance enforcement) |
| `routine.Notifications.*` | `metadata.*` | Scheduling metadata (not GERT runtime concern) |

### Example 2: Incident Template Compilation

**Input (domain model):**

```yaml
# property.home.yaml excerpt
incident_templates:
  - id: pipe-leak
    label: Pipe leak emergency response
    description: Respond to a water pipe leak
    zones: [kitchen, bathroom]
    steps:
      - id: shutoff-water
        type: human_task
        label: Shut off main water valve
        description: Locate and close the main water shutoff valve
        evidence:
          type: photo
          prompt: Photo of closed valve
      
      - id: assess-damage
        type: decision
        label: Assess leak severity
        choices:
          - value: minor
            label: Minor leak (dripping)
            next_step: temporary-fix
          - value: major
            label: Major leak (spraying)
            next_step: call-plumber
      
      - id: temporary-fix
        type: human_task
        label: Apply temporary fix
        description: Use pipe tape or bucket to contain leak
        evidence:
          type: photo
        depends_on: [assess-damage]
      
      - id: call-plumber
        type: human_task
        label: Contact emergency plumber
        description: Call 24/7 plumber service
        evidence:
          type: note
          prompt: Record plumber name and ETA
        depends_on: [assess-damage]
```

**Output (compiled YAML):**

```yaml
$schema: https://gert.run/schemas/v2/runbook.schema.json
apiVersion: v2
id: incident-pipe-leak
name: Pipe leak emergency response
kind: mitigation
description: Respond to a water pipe leak

inputs:
  zone:
    type: string
    required: true
    description: Zone where the incident occurred
  asset:
    type: string
    required: false
    description: Specific asset involved (if any)
  reported_by:
    type: string
    required: false
    description: Person who reported the incident

outputs:
  outcome:
    type: string
    value: "{{ steps.assess_damage.choice }}"
  resolved_at:
    type: string
    value: "{{ now() }}"

metadata:
  domain: home
  incident_template_id: pipe-leak
  applicable_zones: kitchen,bathroom

flow:
  - step:
      id: shutoff_water
      type: collector
      title: Shut off main water valve
      subtitle: Locate and close the main water shutoff valve
      prompt: Photo of closed valve
      fields:
        - name: photo
          type: image
          label: Photo of closed valve
          required: true
      export:
        - photo

  - step:
      id: assess_damage
      type: decision
      title: Assess leak severity
      prompt: Choose the severity of the leak
      variable: leak_severity
      routes:
        - label: Minor leak (dripping)
          goto: temporary_fix
        - label: Major leak (spraying)
          goto: call_plumber

  - step:
      id: temporary_fix
      type: collector
      title: Apply temporary fix
      subtitle: Use pipe tape or bucket to contain leak
      when: "{{ steps.assess_damage.choice == 'minor' }}"
      fields:
        - name: photo
          type: image
          label: Photo of temporary fix
          required: true

  - step:
      id: call_plumber
      type: collector
      title: Contact emergency plumber
      subtitle: Call 24/7 plumber service
      when: "{{ steps.assess_damage.choice == 'major' }}"
      prompt: Record plumber name and ETA
      fields:
        - name: notes
          type: text
          label: Plumber name and ETA
          required: true
          multiline: true
```

**Key mappings:**

| Domain Field | GERT Field | Notes |
|--------------|------------|-------|
| `template.ID` | `id` (prefixed "incident-") | Namespace separation |
| `template.Steps[].Type` | `step.type` | `human_task` → `collector`, `decision` → `decision`, `parallel` → parallel node |
| `template.Steps[].DependsOn` | `step.when` | Converted to conditional expression `{{ steps.X.status == 'completed' }}` |
| `step.Choices[].NextStep` | `routes[].goto` | Direct step ID reference |
| `step.Evidence.Type` | `fields[].type` | Same mapping as routines |

### Example 3: Delegation Compilation

**Input (domain model):**

```yaml
# property.home.yaml excerpt
delegation:
  delegate:
    name: María González
    contact: "+56912345678"
    email: maria@example.com
  active:
    from: "2026-07-01"
    to: "2026-07-15"
  assigns:
    - routine: water-plants
    - routine: check-mail
    - zone: garden
  permissions:
    can_report_incidents: true
    can_modify_routines: false
    can_view_history: true
  notifications:
    remind_delegate_hours_before: 12
    notify_owner_on_completion: true
    notify_owner_if_overdue: true
```

**Output (YAML governance fragment):**

```yaml
# delegation.policy.yaml
# Valid from: 2026-07-01 to 2026-07-15
# Delegate: María González (+56912345678)

apiVersion: v2
kind: policy

governance:
  rules:
    - effects: [execute]
      action: allow
      # Delegate can execute all assigned routines without approval
    
    - effects: [modify_routine]
      action: deny
      # Delegate cannot reschedule routines
  
  allow_commands:
    - gert
    - echo
    - ls
    - cat
    # Conservative allowlist for delegate (no destructive commands)
  
  deny_commands:
    - rm
    - mv
    - chmod
    - chown
    # Prevent destructive operations
  
  deny_env_vars:
    - AWS_*
    - GERT_ADMIN_*
    # Prevent privilege escalation via env vars

metadata:
  delegate_name: María González
  delegate_contact: "+56912345678"
  delegate_email: maria@example.com
  active_from: "2026-07-01"
  active_to: "2026-07-15"
  assigned_routines: water-plants,check-mail
  assigned_zones: garden
  permissions_report_incidents: "true"
  permissions_modify_routines: "false"
  permissions_view_history: "true"
  notify_delegate_hours_before: "12"
  notify_owner_on_completion: "true"
  notify_owner_if_overdue: "true"
```

**Key mappings:**

| Domain Field | GERT Field | Notes |
|--------------|------------|-------|
| `delegation.Delegate.*` | `metadata.delegate_*` | Contact info for notification system |
| `delegation.Active.*` | `metadata.active_*` | Scheduler checks if delegation is active |
| `delegation.Assigns[]` | `metadata.assigned_*` | Scheduler filters runbooks by assignment |
| `delegation.Permissions.CanReportIncidents` | `governance.rules[].action` | `true` → allow incidents, `false` → deny |
| `delegation.Permissions.CanModifyRoutines` | `governance.rules[].action` | `false` → deny modify effect |
| `delegation.Permissions.*` | `metadata.permissions_*` | Enforcement happens in domain scheduler/UI |
| `delegation.Notifications.*` | `metadata.notify_*` | Consumed by notification service |

**Important:** Delegation policy is NOT a runbook. It's a governance overlay that the scheduler applies when starting runs during the delegation window. The scheduler must:

1. Check if current time is within `active_from`..`active_to`
2. Filter routines by `assigned_routines` and `assigned_zones`
3. Merge `delegation.policy.yaml` into the runbook's governance config before calling GERT
4. Pass `--actor delegate:maria@example.com` to `gert run` for audit trail

---

## go.mod Changes Required

### domains/home/go.mod

**No v2 dependency required.** The compiler only produces YAML strings. The domain module remains independent:

```go
module github.com/ormasoftchile/gert-domain-home

go 1.23

require (
	gopkg.in/yaml.v3 v3.0.1  // YAML marshaling
)
```

### Consumers (scheduler, CLI) Require v2

If a **separate component** (e.g., `gert-home-scheduler` or `cmd/home-daemon`) calls the compiler and then invokes GERT, **that component** needs the v2 dependency:

```go
module github.com/ormasoftchile/gert-home-scheduler

go 1.23

require (
	github.com/ormasoftchile/gert-domain-home v0.1.0
	github.com/ormasoftchile/gert/v2 v2.0.0
)

replace (
	github.com/ormasoftchile/gert-domain-home => ../domains/home
	github.com/ormasoftchile/gert/v2 => ../v2
)
```

### Workspace Configuration (go.work)

Already correct (confirmed):

```go
go 1.25.7

use (
	.
	./domains/home
	./v2
	// ... other modules
)
```

No changes needed. The workspace allows local development without published versions.

---

## Validation Strategy

### 1. Compiler Unit Tests

Test each compiler function with domain model fixtures:

```go
// pkg/compiler/routine_test.go
func TestCompileRoutine_PhotoEvidence(t *testing.T) {
	prop := &model.Property{Name: "Test House"}
	routine := &model.Routine{
		ID:    "clean-gutters",
		Label: "Clean gutters",
		Cadence: model.Cadence{Every: model.Duration("90d")},
		Evidence: model.Evidence{
			Type:   model.EvidenceTypePhoto,
			Prompt: "Photo of clean gutters",
		},
	}
	
	yaml, err := CompileRoutine(context.Background(), prop, routine)
	require.NoError(t, err)
	
	// Golden file comparison
	golden := filepath.Join("testdata", "routine-photo-evidence.golden.yaml")
	if *update {
		os.WriteFile(golden, []byte(yaml), 0644)
	}
	expected, _ := os.ReadFile(golden)
	assert.Equal(t, string(expected), yaml)
	
	// Structural validation (YAML is well-formed)
	var rb map[string]any
	err = yaml.Unmarshal([]byte(yaml), &rb)
	require.NoError(t, err)
	
	assert.Equal(t, "v2", rb["apiVersion"])
	assert.Equal(t, "routine-clean-gutters", rb["id"])
	assert.Equal(t, "composable", rb["kind"])
}
```

### 2. Integration Tests (Compiler → Parser)

Verify that compiled YAML is accepted by GERT's parser:

```go
// pkg/compiler/integration_test.go
func TestCompileRoutine_ParsesInGERT(t *testing.T) {
	// Requires v2 dependency (integration test module)
	routine := &model.Routine{...}
	
	yaml, err := compiler.CompileRoutine(ctx, prop, routine)
	require.NoError(t, err)
	
	// Pass to GERT parser
	parser := parser.NewParser()
	parsed, err := parser.ParseBytes(ctx, []byte(yaml))
	require.NoError(t, err)
	
	assert.Equal(t, "routine-clean-gutters", parsed.Runbook.ID)
	assert.Equal(t, "v2", parsed.Runbook.APIVersion)
	assert.Len(t, parsed.Runbook.Flow, 1)
}
```

### 3. End-to-End Tests (Compiler → Parser → Planner → Engine)

Full execution test (requires v2):

```go
func TestE2E_RoutineExecution(t *testing.T) {
	routine := &model.Routine{...}
	
	// 1. Compile
	yaml, err := compiler.CompileRoutine(ctx, prop, routine)
	require.NoError(t, err)
	
	// 2. Parse
	parser := parser.NewParser()
	parsed, err := parser.ParseBytes(ctx, []byte(yaml))
	require.NoError(t, err)
	
	// 3. Plan
	planner := planner.New(planner.Config{...})
	plan, err := planner.Plan(ctx, parsed)
	require.NoError(t, err)
	
	// 4. Execute
	engine := engine.New(engine.Config{...})
	run, err := engine.Start(ctx, plan, engine.RunOptions{
		Mode: engine.RunModeDryRun,
	})
	require.NoError(t, err)
	
	// 5. Advance to collector step
	result, err := run.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, "capture_evidence", result.StepID)
	assert.Equal(t, engine.StepStatusWaiting, result.Status)
}
```

---

## Open Questions & Next Steps

### Open Questions

1. **Schema versioning:** Should the compiler embed `$schema` URL? Current examples use `https://gert.run/schemas/v2/runbook.schema.json` (schema.go doesn't declare this constant).

2. **Metadata conventions:** The compiler emits `metadata.domain = home` and other domain-specific fields. Should GERT v2 reserve a `metadata.domain` namespace for domain compilers?

3. **Evidence type mapping:** Domain uses `photo`, `note`, `checklist`. GERT has `image`, `text`, `boolean[]`. Should there be a canonical mapping table in the GERT docs?

4. **Delegation governance merge:** How does the scheduler merge delegation policy into routine runbooks? Append to `governance.rules[]`? Or should GERT support policy layering/includes?

5. **Catalog format:** `CompileProperty()` returns a `Catalog` struct. Should this be serialized as JSON manifest, YAML, or both? Should it be a standalone `.catalog.yaml` or embedded in a directory structure?

### Next Steps

| Step | Owner | Deliverable |
|------|-------|-------------|
| 1. Review this contract | Ken | Approve or request changes |
| 2. Implement `CompileRoutine` | Brian | `pkg/compiler/routine.go` + tests |
| 3. Implement `CompileIncidentTemplate` | Brian | `pkg/compiler/incident.go` + tests |
| 4. Implement `CompileDelegation` | Brian | `pkg/compiler/delegation.go` + tests |
| 5. Implement `CompileProperty` | Brian | `pkg/compiler/property.go` + tests |
| 6. Create integration test module | Barbara | `tests/integration/go.mod` with v2 dep |
| 7. E2E test (compile → parse → plan → run) | Barbara | `tests/integration/e2e_test.go` |
| 8. Document compiler in domain README | Leslie | `domains/home/README.md` § Compilation |

---

## Appendix: Alternative Architectures Considered

### Alt 1: Two-phase compilation (rejected)

**Approach:** Compiler produces an intermediate JSON representation, then a separate "lowering" pass converts JSON → YAML.

**Rejected because:** Adds unnecessary complexity. YAML generation is straightforward; no benefit to intermediate format.

### Alt 2: Template-based compilation (rejected)

**Approach:** Use Go `text/template` with `.runbook.yaml.tmpl` files.

**Rejected because:**
- Hard to test (templates are stringly typed)
- Difficult to compose (no type safety)
- Error messages are cryptic
- Go structs + YAML marshaling is more maintainable

### Alt 3: Direct ExecutionPlan construction (rejected)

**Approach:** Compiler produces `engine.ExecutionPlan` directly, bypassing parser/planner.

**Rejected because:**
- Tight coupling to v2 internals
- Bypasses all parser validation
- Can't use GERT's include resolution, tool lookup, etc.
- Defeats the purpose of having a parser/planner layer

---

**END OF CONTRACT**
