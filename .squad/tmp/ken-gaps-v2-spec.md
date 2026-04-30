# gert v2 Gap Implementation — Architecture Specification

**Author:** Ken (Software Architect)  
**Date:** 2026-04-30  
**Status:** APPROVED FOR IMPLEMENTATION  
**Target:** v2.0

---

## Executive Summary

This document specifies the architecture for three high-value features approved for gert v2.0:

1. **`type: noop`** — Easy complexity, HIGH value
2. **`required_evidence`** — Medium complexity, HIGH value  
3. **`on_error:` routing** — Easy/Medium complexity, MEDIUM value

Each feature is designed to integrate seamlessly with the existing v2 architecture while maintaining backward compatibility with all current runbooks.

**Key Architectural Principles:**
- `type: noop` is a first-class step type, not a CLI workaround
- `required_evidence` is enforcement metadata, NOT a data collector (collectors stay separate)
- `on_error: goto` targets must be top-level flow steps (no jumping into branches/iterates)
- All 3 features work with existing step-level controls (`when:`, `timeout:`, `delay:`)

---

## Feature 1: type: noop

### Rationale

Reference examples (edge-case-single-step-timeout.runbook.yaml, collect-health.runbook.yaml) use `type: noop` for:
- Pure delay steps (wait without side effects)
- Variable transformations (capture/accumulate report text)
- Stereotyping/rendering hints

Current v2 workaround requires hacky `type: cli` with `run: "true"` or `run: "echo"`, which pollutes trace logs with unnecessary command executions.

### Behavioral Contract

**Core behavior:** A noop step does nothing except advance the execution plan.

**Execution semantics:**
1. Apply `delay:` if present (block for duration)
2. Apply `capture:` if present (evaluate and bind variables)
3. Record step execution event in trace
4. Immediately advance to next step

**No side effects:**
- Does NOT execute any command
- Does NOT invoke tools or external processes
- Does NOT wait for user input
- Does NOT branch or iterate

### Schema Changes

**File:** `pkg/schema/step.go`

Add constant to `StepType` enum:
```go
const (
	// ... existing types ...
	StepTypeNoop     StepType = "noop"  // NEW: no-operation step
	// ... remaining types ...
)
```

**File:** `pkg/schema/steps.go`

Add spec type:
```go
// NoopSpec holds the fields for a step of type "noop".
// A noop step performs no action but can apply delay and capture variables.
type NoopSpec struct {
	// Intentionally empty — noop has no type-specific fields.
	// All behavior is driven by common Step fields: Delay, Capture, etc.
}

// StepKind implementation.
func (s *NoopSpec) StepKind() string { return "noop" }
```

**File:** `pkg/schema/step.go`

Add to `Step` struct type-specific payloads:
```go
type Step struct {
	// ... existing common fields ...
	
	// Type-specific payloads — exactly one is non-nil for a valid step.
	CLI              *CLISpec          `yaml:",inline" json:"-"`
	ToolCall         *ToolCallSpec     `yaml:",inline" json:"-"`
	IncludeSpec      *IncludeSpec      `yaml:",inline" json:"-"`
	NoopSpec         *NoopSpec         `yaml:",inline" json:"-"` // NEW
	// ... remaining types ...
}
```

### Interaction with Step-Level Fields

| Field | Applies? | Behavior |
|-------|----------|----------|
| `delay:` | ✅ YES | Block for duration before advancing |
| `capture:` | ✅ YES | Evaluate and bind variables |
| `when:` | ✅ YES | Skip step if condition is false |
| `timeout:` | ✅ YES | Fail if delay exceeds timeout |
| `retry:` | ⚠️ PARTIAL | Applies only if timeout is set (rare) |
| `continue_on_fail:` | ⚠️ PARTIAL | Only relevant if timeout occurs |
| `on_error:` | ⚠️ PARTIAL | Only triggers if timeout occurs (see Feature 3) |
| `scope:`, `export:` | ✅ YES | Standard variable scoping |
| `contract:` | ❌ NO | Noop has no effects, reads, or writes |

### Executor Behavior

**Pseudocode:**
```go
func (e *NoopExecutor) Execute(ctx context.Context, step *ResolvedStep) error {
	// 1. Apply delay if present
	if step.Delay != "" {
		duration, err := parseDuration(step.Delay)
		if err != nil {
			return fmt.Errorf("invalid delay: %w", err)
		}
		
		// Respect timeout
		if step.Timeout != "" {
			timeout, _ := parseDuration(step.Timeout)
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
		}
		
		select {
		case <-time.After(duration):
			// Delay complete
		case <-ctx.Done():
			return fmt.Errorf("noop timeout: %w", ctx.Err())
		}
	}
	
	// 2. Apply capture if present
	if len(step.Capture) > 0 {
		for varName, expr := range step.Capture {
			value, err := e.evaluator.Eval(ctx, expr)
			if err != nil {
				return fmt.Errorf("capture failed for %s: %w", varName, err)
			}
			e.scope.Set(varName, value)
		}
	}
	
	// 3. Record trace event
	e.trace.Record(StepCompleted{
		StepID: step.ID,
		Type:   "noop",
		// No output, no command, no side effects
	})
	
	// 4. Done — advance to next step
	return nil
}
```

### Edge Cases

#### Noop inside iterate
**Behavior:** Standard iteration semantics apply.

Example:
```yaml
iterate:
  over: items
  as: item
  steps:
    - step:
        id: record
        type: noop
        title: "Record {{ .item }}"
        capture:
          log: "{{ .log }}{{ .item }}\n"
```

Each iteration executes the noop step independently. Capture accumulates across iterations if the variable is scoped correctly.

#### Noop inside branch
**Behavior:** Standard branch semantics apply.

Example:
```yaml
branches:
  - condition: status == "healthy"
    steps:
      - step:
          id: wait_before_next
          type: noop
          delay: 5s
```

The noop executes only if the branch condition is true.

#### Noop inside parallel
**Behavior:** Standard parallel semantics apply. Each parallel branch can contain noop steps.

#### Noop with timeout but no delay
**Behavior:** Step completes immediately (no delay to timeout). Timeout is ignored.

Rationale: Timeout only applies when there's a blocking operation (delay).

### What NOT to Support

❌ **No on_error routing complexities for noop**  
A noop should not have elaborate error handling. The only error case is timeout during delay, which follows standard error semantics (fail step, propagate error).

❌ **No contract declarations**  
Noop has no side effects by definition. Declaring `contract: { effects: [...] }` on a noop step is a parse error.

❌ **No retry without timeout**  
Retry only makes sense if the step can fail. Noop without timeout cannot fail, so retry is meaningless.

### FlowNode Union Integration

**File:** `pkg/schema/runbook.go` (or wherever FlowNode is defined)

Noop is a `Step`, not a standalone FlowNode (unlike `IterateNode`, `ParallelNode`). No changes needed to FlowNode union.

Parser validation:
- `Type == "noop"` → ensure `NoopSpec` is populated (even though it's empty)
- `Type == "noop"` → reject if `contract` is present

---

## Feature 2: required_evidence

### Rationale

v1 examples (service-health-branching, incident-triage, multi-region) use `required_evidence:` to declare that operators MUST provide specific evidence artifacts before a step can complete.

Current v2 has:
- Evidence **collection** (`EvidenceAttached` trace events)
- Evidence **capture** (collector fields with `Required: bool`)

Missing:
- Evidence **schema enforcement** (what KIND of evidence: text, checklist, attachment)
- Evidence **validation** at step completion (block advancement if missing)

### Behavioral Contract

**Core behavior:** Runtime enforces that operator provides declared evidence before step completion.

**Three evidence kinds:**
1. **text** — Free-form text input (e.g., "DNS investigation notes")
2. **checklist** — List of items operator must acknowledge (e.g., pre-deployment checks)
3. **attachment** — File upload with SHA256 content addressing (e.g., logs, screenshots)

**Placement options:**

**Option A: Step-level `required_evidence:` field**  
Any step type can declare required evidence:
```yaml
- step:
    id: investigate
    type: tool
    tool:
      name: curl
      action: health_check
    required_evidence:
      - kind: text
        name: findings
      - kind: attachment
        name: screenshot
```

**Option B: Collector field-level evidence attachment**  
Collector fields can have `evidence_kind:` metadata:
```yaml
- step:
    id: collect_info
    type: collector
    prompt: "Provide deployment evidence"
    fields:
      - name: change_ticket
        type: text
        required: true
        evidence_kind: text   # NEW: declares this field is evidence
      - name: rollback_plan
        type: file
        required: true
        evidence_kind: attachment
```

**Decision:** Support BOTH.
- Step-level for ANY step type (tool, cli, choice, etc.)
- Field-level for collector steps (more granular)

Evidence vs. collector fields:
- **Evidence** declares WHAT must be captured (metadata)
- **Collector fields** declare HOW to capture it (UI form)

A collector field can BE evidence (via `evidence_kind`), but evidence can also be declared on non-collector steps.

### Schema Changes

**File:** `pkg/schema/step.go`

Add to `Step` struct:
```go
type Step struct {
	// ... existing common fields ...
	RequiredEvidence []EvidenceRequirement `yaml:"required_evidence,omitempty" json:"required_evidence,omitempty"` // NEW
	// ... type-specific payloads ...
}
```

**File:** `pkg/schema/evidence.go` (NEW FILE)

```go
package schema

// EvidenceRequirement declares a required evidence artifact.
type EvidenceRequirement struct {
	Kind  EvidenceKind `yaml:"kind"              json:"kind"`
	Name  string       `yaml:"name"              json:"name"`
	Label string       `yaml:"label,omitempty"   json:"label,omitempty"`  // Human-readable label
	Items []string     `yaml:"items,omitempty"   json:"items,omitempty"`  // For checklist kind
	Hint  string       `yaml:"hint,omitempty"    json:"hint,omitempty"`   // Guidance text
}

// EvidenceKind enumerates the types of evidence.
type EvidenceKind string

const (
	EvidenceKindText       EvidenceKind = "text"
	EvidenceKindChecklist  EvidenceKind = "checklist"
	EvidenceKindAttachment EvidenceKind = "attachment"
)

// Validate checks structural validity of an evidence requirement.
func (er *EvidenceRequirement) Validate() error {
	if er.Name == "" {
		return fmt.Errorf("evidence requirement must have a name")
	}
	
	switch er.Kind {
	case EvidenceKindText, EvidenceKindAttachment:
		if len(er.Items) > 0 {
			return fmt.Errorf("evidence kind %s cannot have items", er.Kind)
		}
	case EvidenceKindChecklist:
		if len(er.Items) == 0 {
			return fmt.Errorf("checklist evidence must have at least one item")
		}
	default:
		return fmt.Errorf("unknown evidence kind: %s", er.Kind)
	}
	
	return nil
}
```

**File:** `pkg/schema/steps.go`

Add to `CollectorField`:
```go
type CollectorField struct {
	// ... existing fields ...
	EvidenceKind EvidenceKind `yaml:"evidence_kind,omitempty" json:"evidence_kind,omitempty"` // NEW: marks field as evidence
}
```

### Engine Enforcement

**When to validate:** Before advancing from a step, NOT during execution.

**Rationale:** Evidence is provided by the operator AFTER the step action completes but BEFORE the next step begins. The engine must block step completion until evidence is satisfied.

**Validation logic:**

```go
func (e *Engine) CompleteStep(ctx context.Context, stepID string) error {
	step := e.plan.GetStep(stepID)
	
	// 1. Check if step has required evidence
	if len(step.RequiredEvidence) == 0 {
		return nil  // No evidence required, proceed
	}
	
	// 2. Validate each evidence requirement
	providedEvidence := e.trace.GetEvidenceForStep(stepID)
	
	for _, req := range step.RequiredEvidence {
		ev, found := providedEvidence[req.Name]
		if !found {
			return ErrEvidenceMissing{
				StepID:       stepID,
				EvidenceName: req.Name,
				EvidenceKind: req.Kind,
			}
		}
		
		// Validate kind matches
		if ev.Kind != req.Kind {
			return ErrEvidenceKindMismatch{
				StepID:   stepID,
				Expected: req.Kind,
				Got:      ev.Kind,
			}
		}
		
		// Validate checklist items (all must be checked)
		if req.Kind == EvidenceKindChecklist {
			if !allItemsChecked(req.Items, ev.CheckedItems) {
				return ErrChecklistIncomplete{
					StepID:       stepID,
					EvidenceName: req.Name,
					Missing:      missingItems(req.Items, ev.CheckedItems),
				}
			}
		}
		
		// Validate attachment has content
		if req.Kind == EvidenceKindAttachment {
			if ev.ContentHash == "" {
				return ErrAttachmentMissing{
					StepID:       stepID,
					EvidenceName: req.Name,
				}
			}
		}
	}
	
	// All evidence satisfied
	return nil
}
```

### Error Semantics

**What happens if operator skips required evidence?**

**Option A: BLOCK** — Step cannot be marked complete until evidence is provided (operator cannot advance).

**Option B: WARN** — Display warning but allow advancement (soft enforcement).

**Option C: FAIL** — Terminate run with error (hard enforcement).

**Decision: BLOCK (Option A)**

Rationale:
- Evidence is a governance control (compliance requirement)
- Allowing skip defeats the purpose
- Operator can abandon run if they refuse to provide evidence (explicit choice)

TUI behavior:
- Display "Waiting for evidence" indicator
- Show checklist of missing evidence requirements
- Disable "Next Step" button until all evidence is provided

### TUI Contract

**How TUI surfaces evidence requirements:**

1. **Before step execution:** Display evidence requirements in step preview:
   ```
   Step: Investigate DNS failure
   
   Required Evidence:
   ☐ Text: Investigation findings
   ☐ Checklist: Pre-escalation checks
     ☐ Senior on-call notified
     ☐ Incident ticket created
   ☐ Attachment: Network trace logs
   ```

2. **After step execution:** Show evidence collection UI:
   - Text evidence → Multiline text input
   - Checklist evidence → Checkbox list
   - Attachment evidence → File upload widget

3. **Completion gating:** Disable step completion button until all evidence is provided.

### Trace Event

**New event type:** `EvidenceProvided`

**File:** `pkg/schema/event.go` (or wherever events are defined)

```go
type EvidenceProvided struct {
	EventBase
	StepID       string       `json:"step_id"`
	EvidenceName string       `json:"evidence_name"`
	EvidenceKind EvidenceKind `json:"evidence_kind"`
	
	// Kind-specific payloads (exactly one is populated)
	TextContent    *string  `json:"text_content,omitempty"`     // For kind: text
	CheckedItems   []string `json:"checked_items,omitempty"`    // For kind: checklist
	ContentHash    *string  `json:"content_hash,omitempty"`     // SHA256 for kind: attachment
	ContentSize    *int64   `json:"content_size,omitempty"`     // Bytes for kind: attachment
	ContentType    *string  `json:"content_type,omitempty"`     // MIME type for kind: attachment
	AttachmentPath *string  `json:"attachment_path,omitempty"`  // Filesystem path for kind: attachment
}
```

**When recorded:** Immediately when operator submits evidence, BEFORE step completion.

### Evidence vs. Collector Fields

**Key distinction:**

| Aspect | required_evidence | collector fields |
|--------|------------------|------------------|
| Purpose | WHAT must be captured (schema enforcement) | HOW to capture it (UI form definition) |
| Placement | Any step type | Only collector steps |
| Enforcement | Engine blocks completion | Field validation only |
| Trace impact | Creates EvidenceProvided events | Creates standard collector input events |

**Can they coexist?**

YES. A collector step can have BOTH:
```yaml
- step:
    id: collect_deployment_info
    type: collector
    prompt: "Provide deployment evidence"
    fields:
      - name: change_ticket
        type: text
        required: true
    required_evidence:
      - kind: attachment
        name: approval_email
```

Here:
- `change_ticket` is a collector field (TUI renders text input)
- `approval_email` is required evidence (TUI renders file upload)

Both must be provided before step completion.

### Keep It Simple

**What NOT to do:**

❌ **Don't auto-promote collector fields to evidence**  
A `required: true` collector field is NOT automatically evidence. Evidence must be explicitly declared via `required_evidence:` or `evidence_kind:`.

❌ **Don't support conditional evidence** (yet)  
v1 has no `when:` on evidence requirements. v2.0 won't add it. Defer to v2.1 if needed.

❌ **Don't support evidence templates**  
v1 has no evidence template system. v2.0 won't add it.

---

## Feature 3: on_error routing

### Rationale

v1 had explicit error handling strategies. v2 has `continue_on_fail: bool` but no `goto` on error.

Use cases:
- Jump to rollback step on deployment failure
- Jump to escalation step on timeout
- Jump to cleanup step on any error

This enables saga/compensation patterns without requiring separate compensate steps for every error case.

### Behavioral Contract

**Three routing modes:**

1. **`on_error: stop`** — Terminate run with error (current default behavior)
2. **`on_error: continue`** — Suppress error, advance to next step (equivalent to `continue_on_fail: true`)
3. **`on_error: goto: <step_id>`** — Jump to named step on error (NEW behavior)

**Relationship to `continue_on_fail`:**

When BOTH `on_error` and `continue_on_fail` are present, `on_error` takes precedence.

Example:
```yaml
- step:
    id: deploy
    type: cli
    run: ./deploy.sh
    continue_on_fail: true     # IGNORED
    on_error: goto:rollback    # WINS
```

If only `continue_on_fail: true` is present, it behaves as `on_error: continue`.

### Schema Changes

**File:** `pkg/schema/step.go`

Add to `Step` struct:
```go
type Step struct {
	// ... existing common fields ...
	OnError *ErrorHandler `yaml:"on_error,omitempty" json:"on_error,omitempty"` // NEW
	// ... type-specific payloads ...
}
```

**File:** `pkg/schema/error_handler.go` (NEW FILE)

```go
package schema

// ErrorHandler declares the error handling strategy for a step.
type ErrorHandler struct {
	// Action specifies the error handling mode.
	// Valid values: "stop", "continue", "goto"
	Action string `yaml:"action,omitempty" json:"action,omitempty"`
	
	// Target is the step ID to jump to when Action == "goto".
	// Must be a top-level flow step ID (not a step inside a branch/iterate).
	Target string `yaml:"target,omitempty" json:"target,omitempty"`
}

// Simplified YAML syntax support:
// on_error: stop              → { action: "stop" }
// on_error: continue          → { action: "continue" }
// on_error: goto:rollback     → { action: "goto", target: "rollback" }

// UnmarshalYAML allows both string and struct syntax.
func (eh *ErrorHandler) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Try string syntax first
	var str string
	if err := unmarshal(&str); err == nil {
		switch {
		case str == "stop":
			eh.Action = "stop"
			return nil
		case str == "continue":
			eh.Action = "continue"
			return nil
		case strings.HasPrefix(str, "goto:"):
			eh.Action = "goto"
			eh.Target = strings.TrimPrefix(str, "goto:")
			return nil
		default:
			return fmt.Errorf("invalid on_error value: %s", str)
		}
	}
	
	// Try struct syntax
	type Alias ErrorHandler
	var alias Alias
	if err := unmarshal(&alias); err != nil {
		return err
	}
	*eh = ErrorHandler(alias)
	return nil
}

// Validate checks structural validity.
func (eh *ErrorHandler) Validate() error {
	switch eh.Action {
	case "stop", "continue":
		if eh.Target != "" {
			return fmt.Errorf("on_error action %s cannot have a target", eh.Action)
		}
	case "goto":
		if eh.Target == "" {
			return fmt.Errorf("on_error action goto requires a target")
		}
	default:
		return fmt.Errorf("unknown on_error action: %s", eh.Action)
	}
	return nil
}
```

### YAML Syntax Examples

**String syntax (recommended):**
```yaml
on_error: stop              # Explicit stop (same as default)
on_error: continue          # Suppress error, advance
on_error: goto:rollback     # Jump to rollback step
```

**Struct syntax (alternative):**
```yaml
on_error:
  action: goto
  target: rollback
```

Both are valid. Parser supports both.

### Relationship to `compensate:` Steps

`on_error: goto` can target ANY top-level step, including:
- Regular steps (cli, tool, etc.)
- Compensate steps
- Manual intervention steps
- End steps

Example: Automatic compensation
```yaml
flow:
  - step:
      id: deploy
      type: cli
      run: ./deploy.sh
      on_error: goto:rollback   # Jump to compensate step on error
  
  - step:
      id: rollback
      type: compensate
      compensate:
        on: deploy_failure
        steps:
          - step:
              id: revert
              type: cli
              run: ./rollback.sh
```

When `deploy` fails, execution jumps directly to `rollback` step.

### Scope: Step-Level Only

`on_error` only applies to the step it's declared on. It is NOT inherited by sub-steps.

Example:
```yaml
- step:
    id: parent
    type: branch
    on_error: goto:cleanup     # Applies to parent step only
    branches:
      - condition: status == "ok"
        steps:
          - step:
              id: child
              type: cli
              run: ./task.sh
              # child does NOT inherit parent's on_error
```

If `child` fails, it follows default error behavior (stop), NOT parent's `on_error: goto:cleanup`.

To apply error handling to child steps, declare `on_error` on EACH child.

### Error Capture: Error Variables

When `on_error: goto` fires, the following variables are automatically set in the scope:

- `__error_message` — Error message text
- `__error_step_id` — ID of the step that failed
- `__error_code` — Error code (if available)
- `__error_timestamp` — ISO8601 timestamp of error

These variables are available to the target step and all subsequent steps.

Example:
```yaml
flow:
  - step:
      id: deploy
      type: cli
      run: ./deploy.sh
      on_error: goto:log_failure
  
  - step:
      id: log_failure
      type: cli
      run: |
        echo "Deployment failed: {{ .__error_message }}"
        echo "Failed step: {{ .__error_step_id }}"
```

### Loop Detection

**Invalid: goto same step**

```yaml
- step:
    id: retry_forever
    type: cli
    run: ./task.sh
    on_error: goto:retry_forever   # ❌ PARSE ERROR
```

**Detection:** Parser detects self-referential goto at parse time. Planner rejects it.

**Error message:** `on_error: goto target cannot be the same step (would create infinite loop)`

### Interaction with iterate

**Question:** If an iterate sub-step has `on_error: goto`, does it jump within the iterate or exit to the parent?

**Answer:** Exits to parent (goto targets top-level step IDs only).

Example:
```yaml
flow:
  - iterate:
      id: deploy_all
      over: services
      as: svc
      steps:
        - step:
            id: deploy_one
            type: cli
            run: ./deploy.sh {{ .svc }}
            on_error: goto:rollback_all   # Jumps to top-level rollback_all step
  
  - step:
      id: rollback_all
      type: cli
      run: ./rollback_all.sh
```

When `deploy_one` fails during iteration, execution:
1. Exits the `deploy_all` iterate
2. Jumps to top-level `rollback_all` step

**Rationale:** Goto targets are resolved at plan time (top-level flow). Iterates are dynamically unrolled at runtime, so sub-steps cannot have local goto targets.

**Validation:** Parser validates that all `on_error: goto` targets exist in the top-level flow.

### Interaction with branch

Similar to iterate: `on_error: goto` inside a branch arm jumps to top-level steps, not local branch steps.

Example:
```yaml
flow:
  - step:
      id: check_and_deploy
      type: branch
      branches:
        - condition: env == "prod"
          steps:
            - step:
                id: deploy_prod
                type: cli
                run: ./deploy_prod.sh
                on_error: goto:notify_oncall   # Jumps to top-level notify_oncall
  
  - step:
      id: notify_oncall
      type: tool
      tool:
        name: pagerduty
        action: trigger
```

### Interaction with parallel

`on_error: goto` inside a parallel branch exits the parallel block and jumps to top-level step.

**Edge case:** What if multiple parallel branches have `on_error: goto` to different targets?

**Behavior:** First error wins. Subsequent errors are ignored (parallel block is already exiting).

### Trace Event

When `on_error: goto` fires, record:

```go
type StepErrorRouted struct {
	EventBase
	StepID       string `json:"step_id"`
	ErrorMessage string `json:"error_message"`
	ErrorCode    string `json:"error_code,omitempty"`
	TargetStepID string `json:"target_step_id"`
	Action       string `json:"action"`  // "goto"
}
```

This distinguishes "error routed to recovery" from "error suppressed" (continue) and "error terminated run" (stop).

### Precedence Rules

When BOTH `continue_on_fail` and `on_error` are present:

1. `on_error` wins (explicit strategy)
2. `continue_on_fail` is ignored

Migration path for v1 runbooks:
- `continue_on_fail: true` → equivalent to `on_error: continue`
- Keep supporting `continue_on_fail` for backward compatibility (don't deprecate yet)

### Planner Validation

Planner must validate:

1. **Target exists:** `on_error: goto:X` → step X must exist in top-level flow
2. **No self-loops:** `on_error: goto:X` where X is the same step → reject
3. **No forward jumps into branches/iterates:** `goto:X` where X is inside a branch/iterate → reject
4. **Cycles allowed:** `A -> goto:B -> goto:A` is valid (user's responsibility to avoid infinite loops)

### What NOT to Support

❌ **No conditional error routing** (yet)  
v1 has no `on_error: { when: ..., goto: ... }`. v2.0 won't add it. Defer to v2.1 if needed.

❌ **No error type matching**  
v1 has no `on_error: { if_error_code: 404, goto: not_found }`. v2.0 won't add it.

❌ **No automatic retry with goto**  
`on_error: goto` is manual routing, not automatic retry. Use `retry:` for retries.

---

## Cross-Feature Interactions

### noop + required_evidence

A noop step CAN have required evidence:

```yaml
- step:
    id: pause_for_approval
    type: noop
    delay: 10s
    required_evidence:
      - kind: checklist
        name: pre_deploy_checks
        items:
          - "Change ticket approved"
          - "Rollback plan ready"
```

Execution:
1. Wait 10 seconds
2. Block completion until operator provides checklist
3. Advance to next step

Valid and useful for manual checkpoints.

### noop + on_error

A noop step CAN have `on_error` (rare but valid):

```yaml
- step:
    id: timed_gate
    type: noop
    delay: 5m
    timeout: 5m
    on_error: goto:timeout_handler
```

If delay exceeds timeout (shouldn't happen unless timeout < delay), route to error handler.

Edge case: Useful for testing timeout behavior.

### required_evidence + on_error

If a step has required evidence and fails, `on_error` fires BEFORE evidence collection.

```yaml
- step:
    id: risky_deploy
    type: cli
    run: ./deploy.sh
    on_error: goto:rollback
    required_evidence:
      - kind: text
        name: deploy_log
```

Execution:
1. Run `./deploy.sh`
2. If it fails → jump to `rollback` step (evidence is NOT collected)
3. If it succeeds → block completion until operator provides `deploy_log` evidence

**Rationale:** Evidence is only collected on successful execution. Errors bypass evidence collection.

### All 3 features on one step

Valid example:

```yaml
- step:
    id: complex_step
    type: noop
    delay: 10s
    timeout: 15s
    on_error: goto:timeout_handler
    required_evidence:
      - kind: text
        name: checkpoint_notes
    capture:
      timestamp: "{{ now }}"
```

Execution:
1. Wait 10 seconds (or timeout after 15)
2. If timeout → jump to `timeout_handler`
3. If successful → capture `timestamp` variable
4. Block completion until operator provides `checkpoint_notes` evidence
5. Advance to next step

All features compose cleanly.

---

## Implementation Notes for Brian

### Parsing

**Parser changes:**

1. Add `StepTypeNoop` to parser's type discriminator
2. Parse `required_evidence:` array on all steps
3. Parse `on_error:` field with custom UnmarshalYAML (string or struct)

**Validation rules:**

- Noop steps: reject `contract:` if present
- required_evidence: validate kind, items, name
- on_error: validate action, target (if goto)

### Planning

**Planner changes:**

1. Resolve `on_error: goto` targets (validate target step exists)
2. Detect self-loops (goto same step)
3. Build error routing graph for trace visualization (optional)

**No changes needed for:**
- noop steps (planner treats them as standard steps)
- required_evidence (planner doesn't validate evidence, engine does)

### Execution

**Executor changes:**

1. Add `NoopExecutor` (minimal: delay + capture)
2. Add evidence validation to step completion logic
3. Add error routing to error handler (catch errors, check `on_error`, route accordingly)
4. Set error variables (`__error_message`, etc.) when routing

**Engine state machine:**

Current:
```
Step Start → Execute → Step Complete → Next Step
                ↓ error
              Run Failed
```

New:
```
Step Start → Execute → Step Complete → Validate Evidence → Next Step
                ↓ error          ↑ missing evidence
         Check on_error          └─ Block until provided
                ↓
      stop / continue / goto
```

### Trace Events

**New events:**

1. `EvidenceProvided` — When operator submits evidence
2. `StepErrorRouted` — When `on_error: goto` fires

**Event ordering:**

```
StepStarted
  → StepExecuting
  → StepCompleted (if no error)
  → EvidenceProvided (if required_evidence present, one per evidence item)
  → StepAdvanced
OR
  → StepFailed (if error)
  → StepErrorRouted (if on_error: goto)
  → StepStarted (target step)
```

### TUI Changes

**Required evidence UI:**

- Display evidence requirements before step execution
- Show evidence collection form after step execution
- Disable "Next Step" button until evidence is provided
- Visual indicators: ☐ (pending) → ☑ (provided)

**Error routing UI:**

- When `on_error: goto` fires, show notification: "Error occurred. Routing to step: <target>"
- Highlight error routing in trace view

---

## Backward Compatibility

### Existing runbooks

All three features are ADDITIVE (opt-in). Existing runbooks without these features continue to work unchanged.

### Migration from v1

**type: noop** — Direct syntax match. No migration needed.

**required_evidence** — Direct syntax match. No migration needed.

**on_error** — v1 syntax is compatible with v2:
- `on_error: stop` → same
- `on_error: continue` → same
- `on_error: goto:X` → same

---

## Testing Strategy

### Unit Tests

**Parser:**
- Parse noop step (with/without delay, capture)
- Parse required_evidence (all kinds: text, checklist, attachment)
- Parse on_error (all actions: stop, continue, goto)
- Reject invalid schemas (noop with contract, goto without target, etc.)

**Planner:**
- Resolve on_error goto targets
- Detect self-loops
- Validate goto targets exist in top-level flow

**Executor:**
- Noop executor (delay, capture, timeout)
- Evidence validation (missing, kind mismatch, checklist incomplete)
- Error routing (goto, continue, stop)

### Integration Tests

**Scenarios:**

1. Noop step with delay and capture (verify variable binding)
2. Step with required evidence (verify blocking until provided)
3. Step with on_error: goto (verify routing to target step)
4. All 3 features combined (noop + evidence + on_error)
5. Error routing inside iterate (verify exits to top-level)
6. Error routing inside branch (verify exits to top-level)

### Golden Trace Comparison

Add to `testdata/golden/`:

1. `noop-delay-capture.trace.jsonl` — Noop step execution trace
2. `required-evidence-text.trace.jsonl` — Text evidence collection
3. `required-evidence-checklist.trace.jsonl` — Checklist evidence collection
4. `on-error-goto.trace.jsonl` — Error routing trace
5. `evidence-blocking.trace.jsonl` — Blocked on missing evidence

---

## Rollout Plan

### Phase 1: Schema + Parser (Week 1)

- Add `StepTypeNoop`, `NoopSpec`
- Add `RequiredEvidence`, `EvidenceRequirement`, `EvidenceKind`
- Add `OnError`, `ErrorHandler`
- Parser validation rules
- Unit tests

### Phase 2: Planner (Week 1)

- Planner validation (goto targets, self-loops)
- Integration tests

### Phase 3: Executor (Week 2)

- Noop executor
- Evidence validation logic
- Error routing logic
- Set error variables
- Trace events
- Integration tests

### Phase 4: TUI (Week 2)

- Evidence collection UI
- Error routing notifications
- Visual indicators

### Phase 5: Documentation (Week 3)

- Update schema reference
- Add examples to runbook guide
- Migration notes (v1 → v2)

---

## Open Questions (for Team Review)

### Q1: Evidence validation strictness

Should missing evidence:
- **BLOCK** step completion (operator cannot advance) ← RECOMMENDED
- **WARN** but allow advancement (soft enforcement)
- **FAIL** run (hard enforcement)

**Ken's recommendation:** BLOCK. Evidence is governance control; skip defeats purpose.

### Q2: Error variables namespace

Should error variables be:
- **Double-underscore prefix** (`__error_message`) ← RECOMMENDED (avoids collision)
- **Reserved namespace** (`error.message`)
- **Global variables** (`GERT_ERROR_MESSAGE`)

**Ken's recommendation:** Double-underscore prefix (minimal, no namespace collision).

### Q3: Cycle detection depth

Should planner detect/prevent cycles in `on_error: goto` chains?
- **No** — User's responsibility (allow infinite loops) ← RECOMMENDED
- **Yes** — Reject any cycle (graph analysis)

**Ken's recommendation:** No. Allow cycles. User can create infinite loops (their choice). Runbook-level timeout prevents runaway.

---

**End of Specification**

**Next Steps:**
1. Team review (Ken, Brian, John, Barbara)
2. Approval gate (Ken sign-off)
3. Implementation (Brian, Phase 1-5)
4. Integration testing (Barbara)
5. Documentation (Leslie)
