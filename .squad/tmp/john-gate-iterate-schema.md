# Schema Spec: `gate: stop_if:` on `include` + `concurrency:` on `iterate`

**Author:** John (YAML/Schema Specialist)  
**Date:** 2026-04-26  
**Status:** Normative Draft  
**Requested by:** Cristian

---

## Feature 1: `gate: stop_if:` on `type: include` Steps

### 1.1 Go Struct — Confirmed and Refined

Inspected `/Volumes/Projects/gert/pkg/schema/steps.go`. The existing `IncludeConfig` struct at line 33:

```go
type IncludeConfig struct {
	Runbook string            `yaml:"runbook"        json:"runbook"`
	With    map[string]string `yaml:"with,omitempty" json:"with,omitempty"`
	When    string            `yaml:"when,omitempty" json:"when,omitempty"`
}
```

**Required addition** (style-consistent with existing file — both `yaml:` and `json:` tags, pointer for optional nested struct, `omitempty` on both tags):

```go
// IncludeConfig is the nested include config block.
type IncludeConfig struct {
	Runbook string            `yaml:"runbook"        json:"runbook"`
	With    map[string]string `yaml:"with,omitempty" json:"with,omitempty"`
	When    string            `yaml:"when,omitempty" json:"when,omitempty"`
	Gate    *GateSpec         `yaml:"gate,omitempty" json:"gate,omitempty"`
}

// GateSpec configures a gate on an include step that can stop execution
// based on the child runbook's terminal outcome.
type GateSpec struct {
	StopIf []string `yaml:"stop_if,omitempty" json:"stop_if,omitempty"`
}
```

**Style notes:**
- `*GateSpec` uses pointer (nullable), consistent with `*ApprovalGate`, `*ParallelJoin`, `*OutcomeDeclaration`.
- Both `yaml:` and `json:` tags present on every field, matching all other structs in the file.
- `omitempty` on both tags for all optional fields.
- Comment on `GateSpec` follows the imperative-sentence docstring style of the file.

---

### 1.2 JSON Schema Fragment

The `gate:` property is added inside the `include` object (which is the value of the `include:` key on a step with `type: include`).

```json
{
  "$defs": {
    "GateSpec": {
      "type": "object",
      "additionalProperties": false,
      "required": ["stop_if"],
      "properties": {
        "stop_if": {
          "type": "array",
          "description": "List of outcome category codes. If the child runbook ends with a type:end step whose outcome.category matches any value in this list, the parent runbook halts immediately at this include step.",
          "minItems": 1,
          "items": {
            "type": "string",
            "minLength": 1
          }
        }
      }
    },
    "IncludeConfig": {
      "type": "object",
      "additionalProperties": false,
      "required": ["runbook"],
      "properties": {
        "runbook": {
          "type": "string",
          "minLength": 1,
          "description": "Path to the runbook file to include, relative to the workspace root."
        },
        "with": {
          "type": "object",
          "additionalProperties": { "type": "string" },
          "description": "Input variable bindings passed to the included runbook."
        },
        "when": {
          "type": "string",
          "minLength": 1,
          "description": "Go template expression. If it evaluates to a falsy value the include step is skipped entirely."
        },
        "gate": {
          "$ref": "#/$defs/GateSpec",
          "description": "Optional gate that inspects the child runbook's terminal outcome and stops parent execution if the outcome category matches."
        }
      }
    }
  }
}
```

**Key constraints encoded in the schema:**
| Constraint | JSON Schema rule |
|---|---|
| `gate:` is optional | not in `required` |
| If `gate:` is present, `stop_if:` is required | `GateSpec.required: ["stop_if"]` |
| `stop_if:` must have at least one element | `minItems: 1` |
| Each element is a non-empty string | `items.type: string`, `items.minLength: 1` |
| No extra keys in `gate:` | `additionalProperties: false` |

---

### 1.3 YAML Examples

#### Example A — Single `stop_if` value
```yaml
- type: include
  include:
    runbook: runbooks/check-service-health.yaml
    gate:
      stop_if: [resolved]
```

#### Example B — Multiple `stop_if` values
```yaml
- type: include
  include:
    runbook: runbooks/mitigate-incident.yaml
    with:
      target_service: "{{ .vars.service }}"
    gate:
      stop_if: [resolved, mitigated]
```

#### Example C — `include` without `gate` (unchanged behavior)
```yaml
- type: include
  include:
    runbook: runbooks/notify-oncall.yaml
    with:
      channel: "#incidents"
    when: "{{ .vars.notify_enabled }}"
```

---

### 1.4 Validation Constraints (Normative)

1. **Structural (JSON Schema Phase 1):**
   - `gate:` is optional on any `type: include` step.
   - When `gate:` is present, `stop_if:` is required and must be a non-empty array.
   - Every element of `stop_if:` must be a non-empty string.
   - `gate:` must not appear on any step type other than `include`.

2. **Semantic (Phase 2 runtime validation):**
   - The values in `stop_if:` are not validated against a fixed enum at parse time — they are matched against `outcome.category` at runtime.
   - A warning (not an error) should be emitted at plan time if the referenced child runbook contains no `type: end` step with a matching `outcome.category` value.

---

### 1.5 Semantics: What `stop_if` Matches Against

`stop_if` values are matched against the **`outcome.category`** field of the child runbook's terminal `type: end` step, as declared in `EndSpec.Outcome.Category`:

```go
type EndSpec struct {
	Outcome *OutcomeDeclaration `yaml:"outcome,omitempty" json:"outcome,omitempty"`
}

type OutcomeDeclaration struct {
	Category string `yaml:"category,omitempty" json:"category,omitempty"`
	Code     string `yaml:"code,omitempty"     json:"code,omitempty"`
}
```

**Matching rule:**
- After the child runbook reaches a `type: end` step, the engine evaluates the step's `outcome.category` value.
- If `outcome.category` is present in the parent's `gate.stop_if` list (case-sensitive string equality), the parent runbook halts at the include step.
- If the child runbook has no `type: end` step, or `outcome.category` is absent, `gate` has no effect.
- Halting means the parent runbook's execution terminates at that include step — subsequent steps in the parent are not executed.

**Example child runbook end step:**
```yaml
- type: end
  outcome:
    category: resolved
    code: auto-remediated
```

If the parent include has `gate: { stop_if: [resolved] }`, execution of the parent stops here.

---

## Feature 2: `concurrency:` on `iterate:` Nodes

### 2.1 Go Struct — Confirmed and Refined

Inspected `IterateNode` at line 148 of `steps.go`:

```go
type IterateNode struct {
	ID      string            `yaml:"id"                json:"id"`
	Over    string            `yaml:"over,omitempty"    json:"over,omitempty"`
	As      string            `yaml:"as,omitempty"      json:"as,omitempty"`
	Max     int               `yaml:"max,omitempty"     json:"max,omitempty"`
	Until   string            `yaml:"until,omitempty"   json:"until,omitempty"`
	Collect map[string]string `yaml:"collect,omitempty" json:"collect,omitempty"`
	Steps   []FlowNode        `yaml:"steps"             json:"steps"`
}
```

**Required addition** (style-consistent — `int` with `omitempty`, placed logically after `Max` since both are numeric iteration-control fields):

```go
// IterateNode is the top-level iterate node in a flow array.
type IterateNode struct {
	ID          string            `yaml:"id"                  json:"id"`
	Over        string            `yaml:"over,omitempty"      json:"over,omitempty"`
	As          string            `yaml:"as,omitempty"        json:"as,omitempty"`
	Max         int               `yaml:"max,omitempty"       json:"max,omitempty"`
	Concurrency int               `yaml:"concurrency,omitempty" json:"concurrency,omitempty"`
	Until       string            `yaml:"until,omitempty"     json:"until,omitempty"`
	Collect     map[string]string `yaml:"collect,omitempty"   json:"collect,omitempty"`
	Steps       []FlowNode        `yaml:"steps"               json:"steps"`
}
```

**Style notes:**
- `int` (not `*int`) matching `Max int` convention — zero value `0` is the meaningful "sequential" sentinel, so pointer indirection is unnecessary.
- `omitempty` on both tags — omitted field (zero value 0) means sequential, consistent with omitting `max`.
- Field placed after `Max` as both are numeric loop-control parameters.
- Column-aligned tags, consistent with the file's existing style.

---

### 2.2 JSON Schema Fragment

```json
{
  "$defs": {
    "IterateNode": {
      "type": "object",
      "additionalProperties": false,
      "required": ["id", "steps"],
      "properties": {
        "id": {
          "type": "string",
          "minLength": 1,
          "description": "Unique identifier for this iterate node within the runbook."
        },
        "over": {
          "type": "string",
          "minLength": 1,
          "description": "Go template expression that evaluates to the list to iterate over."
        },
        "as": {
          "type": "string",
          "minLength": 1,
          "description": "Variable name bound to the current iteration item."
        },
        "max": {
          "type": "integer",
          "minimum": 1,
          "description": "Maximum number of iterations. Iteration stops when this limit is reached, even if the list is longer or until is not yet satisfied."
        },
        "concurrency": {
          "type": "integer",
          "minimum": 0,
          "description": "Maximum number of iterations to run concurrently. 0 (default, same as omitting) and 1 both mean sequential execution. Values >= 2 run that many iterations in parallel. When concurrency > 1 and collect: is set, results are appended in completion order, which is non-deterministic."
        },
        "until": {
          "type": "string",
          "minLength": 1,
          "description": "Go template expression evaluated after each iteration. Iteration stops when this expression is truthy."
        },
        "collect": {
          "type": "object",
          "additionalProperties": { "type": "string" },
          "description": "Maps output variable names in the parent scope to Go template expressions evaluated within each iteration. When concurrency > 1, values are appended in completion order (non-deterministic)."
        },
        "steps": {
          "type": "array",
          "minItems": 1,
          "items": { "$ref": "#/$defs/FlowNode" },
          "description": "Steps to execute on each iteration."
        }
      }
    }
  }
}
```

**Key constraints encoded in the schema:**
| Constraint | JSON Schema rule |
|---|---|
| `concurrency:` is optional | not in `required` |
| Value must be a non-negative integer | `type: integer`, `minimum: 0` |
| 0 = sequential (same as omitting) | documented in description; zero is the default |
| 1 = sequential (explicit) | subsumed by `minimum: 0` + runtime semantics |
| `>= 2` = concurrent | runtime enforces fan-out |
| No fractional values | `type: integer` |

---

### 2.3 YAML Examples

#### Example A — Concurrent with `collect:`
```yaml
- iterate:
    id: deploy-services
    over: "{{ .vars.services }}"
    as: svc
    concurrency: 3
    collect:
      deploy_results: "{{ .vars.deploy_result }}"
    steps:
      - type: cli
        cli:
          run: deploy-service --name "{{ .vars.svc }}"
          env:
            SERVICE: "{{ .vars.svc }}"
```
> ⚠️ **Note:** `deploy_results` will be appended in the order services finish deploying, not the order they appear in `.vars.services`. Callers must not rely on positional index correspondence.

#### Example B — `concurrency: 0` (sequential, explicit)
```yaml
- iterate:
    id: patch-hosts
    over: "{{ .vars.hosts }}"
    as: host
    concurrency: 0      # same as omitting — sequential execution guaranteed
    collect:
      patch_log: "{{ .vars.patch_output }}"
    steps:
      - type: cli
        cli:
          run: patch-host --host "{{ .vars.host }}"
```
> `concurrency: 0` is semantically identical to omitting the field. Results in `patch_log` are appended in iteration order (deterministic).

#### Example C — Dynamic list iterate without `concurrency` (no change)
```yaml
- iterate:
    id: validate-endpoints
    over: "{{ .vars.endpoints }}"
    as: endpoint
    until: "{{ .vars.validation_failed }}"
    steps:
      - type: cli
        cli:
          run: curl -sf "{{ .vars.endpoint }}/health"
```

---

### 2.4 Validation Constraints (Normative)

1. **Structural (JSON Schema Phase 1):**
   - `concurrency:` is optional.
   - When present, must be a non-negative integer (`>= 0`).
   - Float, string, or negative values are schema violations.
   - `concurrency: 0` and `concurrency: 1` are valid and mean sequential.

2. **Semantic (Phase 2 runtime validation):**
   - If `over:` is absent (counter-based loop), `concurrency > 1` is a semantic error — concurrent counter loops produce undefined variable state.
   - If `until:` references a variable that is also written by a concurrent iteration step, a race condition warning must be emitted at plan time.

---

### 2.5 Behavior of `collect:` When Concurrent

This is a normative behavioral contract implementors must respect.

**Sequential mode** (`concurrency: 0` or `1`, or field omitted):
- Iterations run one at a time in list order.
- `collect:` values are appended in deterministic list order.
- Callers may safely use positional index correspondence between `over:` list and `collect:` results.

**Concurrent mode** (`concurrency: N` where `N >= 2`):
- Up to `N` iterations run simultaneously.
- Each iteration's collected values are appended to the parent variable **in the order that iteration completes**, not in the order of the source list.
- **Result order is non-deterministic.** Callers must not assume `result[i]` corresponds to `source[i]`.
- The runtime must document this contract clearly in error messages and trace output.
- Implementors: use a goroutine fan-out with a completion channel; append to the result list inside the completion handler (not the launch handler).

**Example of the non-determinism:**
```
over: [A, B, C]
concurrency: 3

# If B finishes first, then A, then C:
collect result order: [B-result, A-result, C-result]
# NOT: [A-result, B-result, C-result]
```

Trace output (JSONL evidence) must record the source item alongside each collected value so consumers can reconstruct the correspondence if needed:
```json
{"step": "deploy-services", "iteration": 2, "item": "svc-B", "collected": {"deploy_results": "ok"}}
```

---

## Summary of Go Struct Changes

### `IncludeConfig` (add `Gate` field)
```go
type IncludeConfig struct {
	Runbook string            `yaml:"runbook"        json:"runbook"`
	With    map[string]string `yaml:"with,omitempty" json:"with,omitempty"`
	When    string            `yaml:"when,omitempty" json:"when,omitempty"`
	Gate    *GateSpec         `yaml:"gate,omitempty" json:"gate,omitempty"`
}

type GateSpec struct {
	StopIf []string `yaml:"stop_if,omitempty" json:"stop_if,omitempty"`
}
```

### `IterateNode` (add `Concurrency` field)
```go
type IterateNode struct {
	ID          string            `yaml:"id"                    json:"id"`
	Over        string            `yaml:"over,omitempty"        json:"over,omitempty"`
	As          string            `yaml:"as,omitempty"          json:"as,omitempty"`
	Max         int               `yaml:"max,omitempty"         json:"max,omitempty"`
	Concurrency int               `yaml:"concurrency,omitempty" json:"concurrency,omitempty"`
	Until       string            `yaml:"until,omitempty"       json:"until,omitempty"`
	Collect     map[string]string `yaml:"collect,omitempty"     json:"collect,omitempty"`
	Steps       []FlowNode        `yaml:"steps"                 json:"steps"`
}
```
