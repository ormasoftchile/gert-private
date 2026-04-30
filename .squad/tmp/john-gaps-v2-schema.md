# Schema Specification: Gap Features for gert v2
**Author:** John (YAML/Schema Specialist)  
**Date:** 2026-04-30  
**Status:** Ready for Implementation  
**Target:** gert v2.0

---

## Executive Summary

This document specifies the Go schema design for three approved gap features:
1. **`type: noop`** — No-operation step for delay/capture/transform without execution
2. **`required_evidence`** — Schema-enforced evidence collection (step-level and field-level)
3. **`on_error`** — Explicit error routing (goto/stop/continue)

All designs follow established conventions from `pkg/schema/` (yaml/json tags, pointer vs value types, string constants pattern, StepKind() registration).

---

## Feature 1: type: noop — NoopSpec

### Overview

A no-operation step that executes no command or tool but can apply:
- Delay (via `delay: 5s`)
- Variable capture/transformation (via `capture:`)
- Conditional execution (via `when:`)
- Timeout enforcement (via `timeout:`)

Common use cases:
- Pure wait steps (settling time, cooldown periods)
- Variable accumulation (building report strings across iterations)
- Placeholder steps (for UI rendering hints or future implementation)

### Go Schema

**File:** `pkg/schema/steps.go`

```go
// NoopSpec holds the fields for a step of type "noop".
// A noop step performs no action but can apply delay, capture variables,
// and participate in conditional execution. All behavior is controlled by
// common Step fields (delay, capture, when, timeout).
type NoopSpec struct{}

// StepKind implementation for noop step type.
func (s *NoopSpec) StepKind() string { return "noop" }
```

**File:** `pkg/schema/step.go`

Add to Step struct's type-specific payloads:

```go
// Add after line 67 (after EndSpec):
NoopSpec *NoopSpec `yaml:",inline" json:"-"` // populated when Type == "noop"
```

**File:** `pkg/schema/step.go` — StepType constants

Add to const block (after line 34):

```go
// Add after StepTypeEnd:
StepTypeNoop StepType = "noop"
```

### YAML Examples

**Example 1: Pure delay (wait for system to settle)**
```yaml
- step:
    id: wait_for_settle
    type: noop
    title: "Wait for system to settle"
    delay: 5s
```

**Example 2: Variable accumulation in iteration**
```yaml
flow:
  - iterate:
      id: collect_health
      over: hosts
      as: host
      steps:
        - step:
            id: check_host
            type: cli
            run: "curl {{ .host }}/health"
            capture:
              last_status: "{{ .stdout }}"

        - step:
            id: accumulate_report
            type: noop
            title: "Record {{ .host }} result"
            capture:
              report: "{{ .report }}{{ .host }}: {{ .last_status }}\n"
```

**Example 3: Placeholder for future work**
```yaml
- step:
    id: future_integration
    type: noop
    title: "TODO: Call external API when provider is ready"
    when: "env.INTEGRATION_ENABLED == 'true'"
```

### Validation Constraints

- **NoopSpec struct:** Always valid (no fields to validate)
- **Step-level validation:** Standard Step fields apply (when, timeout, delay, capture all valid)
- **Parser behavior:** When `Type == "noop"`, parser populates `Step.NoopSpec = &NoopSpec{}`
- **Executor behavior:** Noop executor:
  1. Evaluates `when` condition (skip if false)
  2. Applies `delay` (if set)
  3. Evaluates `capture` expressions (update run variables)
  4. Advances to next step (no command execution)

### Integration Notes for Brian

1. **Parser (internal/parser/):**
   - Add `"noop"` to step type dispatch map
   - When `Type == "noop"`, populate `Step.NoopSpec = &NoopSpec{}`
   - No type-specific validation needed (struct has no fields)

2. **Executor (internal/executor/):**
   - Create `noop_executor.go` with minimal implementation:
     - Check `when` condition
     - Apply `delay` (time.Sleep or timer)
     - Evaluate `capture` map
     - Return success (no error condition possible)

3. **Planner (internal/planner/):**
   - No special handling needed (noop is a leaf step, no includes/branches)

---

## Feature 2: required_evidence — EvidenceRequirement

### Overview

Schema-enforced evidence collection with three evidence kinds:
- **text** — Free-form text notes (investigation log, justification, observations)
- **checklist** — Multi-item checklist (pre-flight checks, compliance verification)
- **attachment** — File/image upload (screenshots, log files, diagnostic output)

Evidence requirements attach at two levels:
1. **Step-level** — `Step.RequiredEvidence []EvidenceRequirement` (enforces evidence before step completes)
2. **Field-level** — `CollectorField.Evidence *EvidenceRequirement` (attaches evidence to specific collector field)

### Go Schema

**File:** `pkg/schema/steps.go` — Add new types at end of file (after EndSpec):

```go
// EvidenceKind enumerates the valid types of evidence that can be required.
type EvidenceKind string

const (
	EvidenceKindText       EvidenceKind = "text"       // Free-form text notes
	EvidenceKindChecklist  EvidenceKind = "checklist"  // Multi-item checklist
	EvidenceKindAttachment EvidenceKind = "attachment" // File/image upload
)

// EvidenceRequirement declares a required evidence artifact.
// Used in step-level enforcement (Step.RequiredEvidence) or field-level
// attachment (CollectorField.Evidence).
type EvidenceRequirement struct {
	Kind  EvidenceKind `yaml:"kind"            json:"kind"`
	Name  string       `yaml:"name"            json:"name"`
	Label string       `yaml:"label,omitempty" json:"label,omitempty"`
	Items []string     `yaml:"items,omitempty" json:"items,omitempty"` // Required for kind: checklist
}
```

**File:** `pkg/schema/step.go` — Add to Step struct (after line 53):

```go
// Add after Contract field:
RequiredEvidence []EvidenceRequirement `yaml:"required_evidence,omitempty" json:"required_evidence,omitempty"`
```

**File:** `pkg/schema/steps.go` — CollectorField struct

Add field after line 106 (after `FromStep`):

```go
// Add after FromStep field:
Evidence *EvidenceRequirement `yaml:"evidence,omitempty" json:"evidence,omitempty"`
```

### YAML Examples

**Example 1: Step-level evidence (deployment verification)**
```yaml
- step:
    id: deploy_service
    type: cli
    run: ./deploy.sh {{ .environment }}
    title: "Deploy service to {{ .environment }}"
    required_evidence:
      - kind: text
        name: deployment_notes
        label: "Deployment notes"

      - kind: checklist
        name: pre_deploy_check
        label: "Pre-deployment checklist"
        items:
          - "Change ticket approved"
          - "Rollback plan documented"
          - "Monitoring dashboards open"
          - "Stakeholders notified"

      - kind: attachment
        name: deployment_screenshot
        label: "Deployment dashboard screenshot"
```

**Example 2: Field-level evidence (collector field attachment)**
```yaml
- step:
    id: collect_deploy_info
    type: collector
    prompt: "Provide deployment information"
    fields:
      - name: environment
        type: select
        label: "Target environment"
        options:
          - { value: "staging", label: "Staging" }
          - { value: "production", label: "Production" }

      - name: change_ticket
        type: text
        label: "Change ticket ID"
        required: true
        evidence:
          kind: text
          name: change_ticket_link
          label: "Link to change ticket"

      - name: rollback_plan
        type: text
        label: "Rollback plan"
        multiline: true
        evidence:
          kind: attachment
          name: rollback_doc
          label: "Upload rollback procedure document"
```

**Example 3: Mixed evidence on incident triage**
```yaml
- step:
    id: classify_incident
    type: choice
    prompt: "What is the primary incident category?"
    variable: category
    options:
      - value: "network"
        label: "Network — DNS failures, timeouts"
      - value: "app_crash"
        label: "Application — crashes, OOM"
      - value: "resource_exhaustion"
        label: "Resource — CPU/memory/disk full"
    required_evidence:
      - kind: text
        name: classification_notes
        label: "Classification justification"

      - kind: checklist
        name: initial_checks
        label: "Initial investigation checklist"
        items:
          - "Checked monitoring dashboards"
          - "Reviewed recent alerts"
          - "Identified impacted services"
```

### Validation Constraints

1. **EvidenceKind validation:**
   - Must be one of: `text`, `checklist`, `attachment`
   - Empty or unknown values are invalid

2. **EvidenceRequirement validation:**
   - `kind` field is REQUIRED
   - `name` field is REQUIRED (must be unique within step scope)
   - `label` field is OPTIONAL (defaults to `name` if not set)
   - `items` field:
     - REQUIRED when `kind == "checklist"`
     - MUST have at least 1 item when present
     - FORBIDDEN when `kind == "text"` or `kind == "attachment"`

3. **Step-level validation (Step.RequiredEvidence):**
   - Array can be empty (no evidence required)
   - All `name` values must be unique within the step
   - Each element must pass EvidenceRequirement validation

4. **Field-level validation (CollectorField.Evidence):**
   - Can be nil (no evidence required on field)
   - If set, must pass EvidenceRequirement validation
   - `name` must be unique within the collector step scope (collision with step-level or other field evidence is invalid)

### Runtime Enforcement

**Engine behavior (executor layer):**

1. **Before step completes:**
   - Check all `Step.RequiredEvidence` entries
   - For each requirement, verify evidence artifact exists in trace events
   - Evidence events: `EvidenceTextProvided`, `EvidenceChecklistCompleted`, `EvidenceAttachmentUploaded`
   - If any evidence is missing, step CANNOT complete (executor blocks until all evidence collected)

2. **Collector field evidence:**
   - When collecting field value, check if `CollectorField.Evidence` is set
   - TUI prompts for evidence attachment before accepting field value
   - Evidence is recorded as separate trace event linked to field name

3. **Trace event linkage:**
   - Evidence events include: `step_id`, `evidence_name`, `evidence_kind`, `evidence_value` (text/checklist_items/file_path)
   - Evidence validator correlates events with `Step.RequiredEvidence` array

### Integration Notes for Brian

1. **Parser validation (internal/parser/):**
   - Add `EvidenceKind` enum validation (text/checklist/attachment)
   - For `kind: checklist`, enforce `items` array is non-empty
   - For `kind: text` or `attachment`, reject if `items` is present
   - Check `name` uniqueness within step scope (Step.RequiredEvidence + all CollectorField.Evidence)

2. **Executor (internal/executor/):**
   - Create `evidence_validator.go` helper:
     - Tracks evidence events during step execution
     - Validates all required evidence collected before step completes
   - Modify step executors to call evidence validator at end of step

3. **Event schema (pkg/schema/event.go):**
   - Add evidence event types (if not already present):
     - `EvidenceTextProvided(step_id, name, text_value)`
     - `EvidenceChecklistCompleted(step_id, name, items_checked)`
     - `EvidenceAttachmentUploaded(step_id, name, file_path, sha256)`

---

## Feature 3: on_error — Error Routing

### Overview

Explicit error handling strategy per step. Controls what happens when a step fails (non-zero exit code, timeout, assertion failure).

Three routing modes:
1. **continue** — Log error, continue to next step (same as `continue_on_fail: true`)
2. **stop** — Terminate run with error outcome (default behavior)
3. **goto: <step_id>** — Jump to error handler step (e.g., rollback, cleanup, compensation)

### Design Decision: String vs Struct

Reviewed existing patterns in `pkg/schema/steps.go`:
- `WaitForEventSpec.OnTimeout string` (line 234) — simple string (step ID or action)
- `ParallelJoin.OnFailure string` (line 183) — simple string (step ID or action)
- `ApproveSpec.OnTimeout string` (line 192) — simple string (step ID or action)
- `ApprovalGate.OnTimeout string` (line 203) — simple string (step ID or action)

**Precedent:** All existing timeout/failure routing fields use **simple string** format.

**Chosen approach:** **Tagged string** (consistent with existing patterns)

Format:
- `"continue"` — continue to next step
- `"stop"` — terminate run with error outcome
- `"goto:<step_id>"` — jump to error handler step (e.g., `"goto:rollback_handler"`)

Alternative considered: Struct with `action` and `target` fields (rejected for complexity and inconsistency with existing fields).

### Go Schema

**File:** `pkg/schema/step.go` — Add to Step struct (after line 49):

```go
// Add after ContinueOnFail field:
OnError string `yaml:"on_error,omitempty" json:"on_error,omitempty"`
```

**Notes:**
- No separate struct needed (simple string field)
- Validation happens in parser (check format: continue|stop|goto:<step_id>)
- Planner validates `goto:<step_id>` references exist in flow

### YAML Examples

**Example 1: Continue on error (non-critical health check)**
```yaml
- step:
    id: optional_health_check
    type: cli
    run: "curl {{ .service }}/health"
    on_error: continue
    capture:
      health_status: "{{ .stdout }}"
```

**Example 2: Stop on error (critical deployment step)**
```yaml
- step:
    id: deploy_database_migration
    type: cli
    run: ./migrate.sh up
    on_error: stop  # Explicitly halt (same as default, but documented)
```

**Example 3: Goto error handler (rollback pattern)**
```yaml
flow:
  - step:
      id: acquire_lock
      type: cli
      run: etcdctl lock deploy_lock
      on_error: goto:cleanup

  - step:
      id: deploy_service
      type: cli
      run: ./deploy.sh
      on_error: goto:rollback_handler

  - step:
      id: verify_deployment
      type: cli
      run: ./verify.sh
      on_error: goto:rollback_handler

  - step:
      id: release_lock
      type: cli
      run: etcdctl unlock deploy_lock
      goto: end_success

  # Error handlers:
  - step:
      id: rollback_handler
      type: cli
      title: "Rollback deployment"
      run: ./rollback.sh

  - step:
      id: cleanup
      type: cli
      title: "Release resources"
      run: etcdctl unlock deploy_lock || true

  - step:
      id: end_success
      type: end
      outcome:
        category: success
```

**Example 4: Compensate pattern (saga transaction)**
```yaml
flow:
  - step:
      id: reserve_inventory
      type: cli
      run: ./reserve.sh {{ .product_id }} {{ .quantity }}
      on_error: stop  # Cannot proceed without inventory

  - step:
      id: charge_payment
      type: cli
      run: ./charge.sh {{ .customer_id }} {{ .amount }}
      on_error: goto:compensate_inventory

  - step:
      id: ship_order
      type: cli
      run: ./ship.sh {{ .order_id }}
      on_error: goto:compensate_payment

  - step:
      id: end_success
      type: end
      outcome:
        category: success

  # Compensation handlers:
  - step:
      id: compensate_payment
      type: cli
      title: "Refund payment"
      run: ./refund.sh {{ .customer_id }} {{ .amount }}
      goto: compensate_inventory

  - step:
      id: compensate_inventory
      type: cli
      title: "Release inventory reservation"
      run: ./release.sh {{ .product_id }} {{ .quantity }}
      goto: end_failure

  - step:
      id: end_failure
      type: end
      outcome:
        category: failure
        code: transaction_rolled_back
```

### Validation Constraints

1. **Format validation (parser):**
   - If `on_error` is set, value must match one of:
     - `"continue"`
     - `"stop"`
     - `"goto:<step_id>"` (where `<step_id>` is any non-empty string)
   - Empty string is invalid (omit field instead)

2. **Reference validation (planner):**
   - If `on_error` starts with `"goto:"`, extract step_id
   - Verify step_id exists in the runbook flow
   - Cyclic goto is allowed (planner doesn't enforce DAG for error paths)

3. **Precedence (executor):**
   - If `on_error` is set, it takes precedence over `continue_on_fail`
   - `on_error: continue` is equivalent to `continue_on_fail: true`
   - `on_error: stop` overrides `continue_on_fail: true`

4. **Interaction with retry (executor):**
   - Retry logic runs BEFORE on_error routing
   - If step has `retry: { max: 3 }`, all retries are attempted first
   - After max retries exhausted, on_error routing applies

### Relationship to continue_on_fail

**Deprecation status:** `continue_on_fail` is NOT deprecated in v2 (retained for backward compatibility and simple cases).

**Recommendation:**
- Use `continue_on_fail: true` for simple "ignore errors" cases
- Use `on_error: continue` for explicit documentation (same semantics)
- Use `on_error: goto:<step_id>` for complex error handling (rollback, compensation, cleanup)

**Future v2.1 consideration:** Add lint warning when both `continue_on_fail` and `on_error` are set (conflicting intent).

### Integration Notes for Brian

1. **Parser validation (internal/parser/):**
   - Add format validator for `on_error` field:
     - Accept: `continue`, `stop`, `goto:<step_id>`
     - Reject: empty string, unknown values, malformed goto (e.g., `goto:` with no step_id)
   - Extract `goto:<step_id>` references, add to step dependency graph

2. **Planner (internal/planner/):**
   - When building execution plan, resolve `goto:<step_id>` to step index
   - Store in plan metadata: `error_target_index int` (or -1 for continue/stop)
   - Validate referenced step exists in flow

3. **Executor (internal/executor/):**
   - After step fails (and retries exhausted):
     - If `on_error == "continue"`, log error and advance to next step
     - If `on_error == "stop"`, terminate run with error outcome
     - If `on_error == "goto:<step_id>"`, set next_step_index to target step
   - Trace event: `ErrorRouted(step_id, on_error_action, target_step_id)`

---

## Implementation Checklist for Brian

### Phase 1: Schema Updates (pkg/schema/)

- [ ] **steps.go** — Add `NoopSpec` struct + `StepKind()` method
- [ ] **steps.go** — Add `EvidenceKind` type + constants
- [ ] **steps.go** — Add `EvidenceRequirement` struct
- [ ] **steps.go** — Add `CollectorField.Evidence *EvidenceRequirement`
- [ ] **step.go** — Add `StepTypeNoop` constant
- [ ] **step.go** — Add `Step.NoopSpec *NoopSpec` field
- [ ] **step.go** — Add `Step.RequiredEvidence []EvidenceRequirement` field
- [ ] **step.go** — Add `Step.OnError string` field

### Phase 2: Parser (internal/parser/)

- [ ] **parser.go** — Add `"noop"` to step type dispatch
- [ ] **validator.go** — Add `EvidenceKind` enum validation
- [ ] **validator.go** — Add `EvidenceRequirement.items` validation (required for checklist)
- [ ] **validator.go** — Add evidence `name` uniqueness check (step scope)
- [ ] **validator.go** — Add `on_error` format validation (continue|stop|goto:<id>)
- [ ] **validator.go** — Add `goto:<step_id>` reference existence check

### Phase 3: Executor (internal/executor/)

- [ ] **noop_executor.go** — Create executor for noop steps (delay + capture)
- [ ] **evidence_validator.go** — Create evidence collection validator
- [ ] **step_executor.go** — Add evidence validation before step completion
- [ ] **step_executor.go** — Add on_error routing logic (continue/stop/goto)
- [ ] **event.go** — Add evidence trace events (if not present)

### Phase 4: Tests

- [ ] **parser_test.go** — Test noop step parsing
- [ ] **parser_test.go** — Test evidence requirement validation (all kinds)
- [ ] **parser_test.go** — Test on_error format validation
- [ ] **executor_test.go** — Test noop executor (delay + capture)
- [ ] **executor_test.go** — Test evidence collection enforcement
- [ ] **executor_test.go** — Test on_error routing (continue/stop/goto)

---

## JSON Schema Snippets (for reference)

**NoopSpec (empty object):**
```json
{
  "type": "object",
  "properties": {},
  "additionalProperties": false
}
```

**EvidenceRequirement:**
```json
{
  "type": "object",
  "required": ["kind", "name"],
  "properties": {
    "kind": {
      "type": "string",
      "enum": ["text", "checklist", "attachment"]
    },
    "name": {
      "type": "string",
      "minLength": 1
    },
    "label": {
      "type": "string"
    },
    "items": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "minItems": 1
    }
  },
  "allOf": [
    {
      "if": {
        "properties": { "kind": { "const": "checklist" } }
      },
      "then": {
        "required": ["items"]
      }
    },
    {
      "if": {
        "properties": { "kind": { "enum": ["text", "attachment"] } }
      },
      "then": {
        "not": {
          "required": ["items"]
        }
      }
    }
  ]
}
```

**Step.on_error (string with pattern):**
```json
{
  "type": "string",
  "oneOf": [
    { "const": "continue" },
    { "const": "stop" },
    { "pattern": "^goto:.+$" }
  ]
}
```

---

## Notes for Leslie (LaTeX Documentation)

When documenting these features in the design doc:

1. **§03 Schema vNext:**
   - Add `type: noop` to step type enumeration table
   - Add `required_evidence` to Step common fields table
   - Add `evidence` to CollectorField schema table
   - Add `on_error` to Step common fields table
   - Include validation rules for each

2. **§05 Execution Model:**
   - Document noop executor behavior (delay → capture → advance)
   - Document evidence enforcement (blocking step completion)
   - Document on_error routing (retry precedence, goto resolution)

3. **§15 v1 → v2 Migration:**
   - Document noop workaround in v1 (cli + run: true)
   - Document evidence migration (v1 had implicit evidence, v2 has schema-enforced)
   - Document on_error as replacement for continue_on_fail in complex cases

---

## Open Questions / Future Work

1. **Evidence expiry:** Should evidence have TTL or version constraints? (v2.1 consideration)
2. **Evidence templates:** Should checklists support templated items (e.g., "Check {{ .region }} dashboard")? (v2.1 consideration)
3. **on_error compensation:** Should `on_error: compensate` auto-trigger compensate steps? (deferred to v2.1, requires saga executor)
4. **Evidence validation rules:** Should evidence support regex patterns or JSON schema validation? (v2.1 consideration)

---

## Revision History

| Date       | Author | Change                          |
|------------|--------|---------------------------------|
| 2026-04-30 | John   | Initial schema specification    |

