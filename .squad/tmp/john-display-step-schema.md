# Display Step Schema Design

**Author:** John (YAML/Schema Specialist)  
**Date:** 2026-07-16  
**Context:** `collect-health` runbook needs to surface an accumulated `report` variable to the operator after an `iterate` loop. No existing step type covers "display content, no input required."  
**Requested by:** Cristian

---

## Problem Statement

After an `iterate` loop that accumulates a `report` variable, the runbook has no way to present that value to the operator before proceeding or terminating. The gap is classified **G1** under the validation methodology: *"action has no direct step type."*

The four candidates evaluated:

| Approach | Verdict |
|---|---|
| `type: display` (new step type) | ✅ **Recommended — primary** |
| `end` + `summary:` field | ✅ **Viable — terminal-only variant** |
| Extend `noop` with `output:` | ❌ Rejected — semantic dilution |
| Tool-based file export | ❌ Rejected — heavyweight, tool-dependent |

---

## Approach A — `type: display` (Recommended)

### Rationale

A dedicated step type is the right abstraction. It:
- Has a single, unambiguous responsibility (emit rendered content)
- Composes naturally with `when:`, `delay:`, and `iterate` loops for mid-runbook display
- Is traceable: the engine logs that a display step executed, what it rendered
- Does not corrupt any existing step type's semantics
- Is forward-extensible: `format:`, `pause:`, and `level:` can be added without a breaking change

### Go Struct

```go
// DisplaySpec holds the fields for a step of type "display".
// A display step renders content to the operator's terminal with no input required.
// Content is rendered as a Go text/template string against the current variable scope.
type DisplaySpec struct {
    Content string `yaml:"content"            json:"content"`
    Format  string `yaml:"format,omitempty"   json:"format,omitempty"`
    Title   string `yaml:"title,omitempty"    json:"title,omitempty"`
}

func (s *DisplaySpec) StepKind() string { return "display" }
```

### Field-by-Field Specification

#### `content:` (required, string)

The body of the message to present to the operator.

- **Template rendering:** Yes — rendered via `TemplateEvaluator` (Go `text/template`), the same
  engine used for CLI `run:`, `args:`, and collector field `label:` strings. Variable access
  uses `{{ .varName }}` dot-notation per the established two-evaluator decision (text/template
  for string interpolation; expr-lang/expr only for boolean conditions).
- **YAML multiline:** Yes — authors use `|` or `>` blocks. The literal text, including preserved
  newlines from `|`, is passed to the template engine before display.
- **Empty string:** Not valid. Semantic validation rejects a `display` step with blank `content`.

#### `format:` (optional, enum, default `"text"`)

A render hint for the TUI/front-end layer. The engine records it in the trace but does not
process it; rendering is the TUI's responsibility.

| Value | Meaning |
|---|---|
| `text` | Plain pre-formatted text. Display verbatim, preserve whitespace. (default) |
| `markdown` | Render with Markdown formatting (bold, lists, headers, code blocks). |

Rationale for limiting to two values now: `json` and `table` can be added later without
breaking existing runbooks. Premature enumeration of render formats creates maintenance burden.

#### `title:` (optional, string)

A short heading rendered above `content`, e.g. in a styled panel or as a bold label.
Template-rendered (`{{ .varName }}` supported). If omitted, only `content` is shown.

Rationale for separating `title` from `content`: TUI implementations can style the heading
distinctly (bold, colour, border). Embedding `## Health Report` into `content` with Markdown
works but is not as reliable across render targets (plain terminal vs. VS Code panel).

### Common Step Field Compatibility

| Field | Applicable | Notes |
|---|---|---|
| `id:` | ✅ Required | Used in audit trace, goto targets |
| `type:` | ✅ Required | `"display"` |
| `title:` (Step) | ✅ | Step-level title (navigation label); distinct from `DisplaySpec.title` (content heading) |
| `subtitle:` | ✅ | Shown in step list navigator |
| `when:` | ✅ | Conditional display: `when: report != ""` |
| `delay:` | ✅ | Pause before showing content |
| `timeout:` | ⚠️ Allowed but rarely meaningful | A display step cannot time out in the traditional sense |
| `capture:` | ❌ Disallowed (semantic validation) | Display produces no output; capture has nothing to bind |
| `retry:` | ❌ Disallowed (semantic validation) | Display cannot fail in a retriable sense |
| `continue_on_fail:` | ⚠️ Allowed | Harmless; engine-level consistency |
| `on_error:` | ⚠️ Allowed | Handles edge cases (e.g., template render error) |
| `required_evidence:` | ❌ Not applicable | Display produces no action to evidence |
| `scope:` | ✅ Allowed | Display can be scoped to a sub-run scope |
| `export:` | ❌ Not applicable | Nothing to export |
| `contract:` | ✅ Optional | Authors can declare `reads: [report]` for governance audits |

**Validation rules (semantic phase):**
1. `content` must be non-empty after trimming.
2. `format` must be one of `["text", "markdown"]` if present.
3. `capture` must be absent (or empty map).
4. `retry` must be absent.
5. `required_evidence` must be absent.

### YAML Example — `collect-health` Use Case

```yaml
# collect-health.yaml
apiVersion: runbook/v2
id: collect-health
name: Service Health Check
description: Polls each service and displays a consolidated health report.
inputs:
  - name: services
    type: array
    description: List of service names to check

flow:
  # ── 1. Initialise report accumulator ─────────────────────────────────────
  - id: init-report
    type: noop
    capture:
      report: '""'                          # empty string seed for accumulator

  # ── 2. Loop over every service ───────────────────────────────────────────
  - id: health-loop
    iterate:
      over: services
      as: svc
      collect:
        report: report                      # accumulate across iterations
    steps:
      - id: check-svc
        type: cli
        run: |
          curl -sf "http://{{ .svc }}/health" \
            && echo "{{ .svc }}: ✅ healthy" \
            || echo "{{ .svc }}: ❌ unreachable"
        capture:
          svc_status: stdout

      - id: append-report
        type: noop
        capture:
          report: '{{ .report }}{{ .svc_status }}\n'   # accumulate line

  # ── 3. Display the consolidated report ───────────────────────────────────
  - id: show-report
    type: display
    title: Service Health Report          # heading above content
    format: markdown
    content: |
      ## Health Check Results

      {{ .report }}

      > Run completed at {{ .run.started_at }}

  # ── 4. Terminate with declared outcome ───────────────────────────────────
  - id: done
    type: end
    outcome:
      category: success
      code: health-check-complete
```

### Field Commentary on the Example

| Field | Purpose |
|---|---|
| `title: Service Health Report` | TUI renders this as a bold panel header, distinct from step navigation label |
| `format: markdown` | Operator's terminal renders `##` heading and `>` blockquote; degrades gracefully to text |
| `content: \|` | YAML literal block preserves newlines; Go template engine interpolates `{{ .report }}` |
| `{{ .run.started_at }}` | Illustrates access to engine-injected run metadata (forward-compatible placeholder) |
| No `capture:` | Display produces no output; nothing to bind |
| `when:` omitted | Always runs; caller can guard with `when: report != ""` if needed |

---

## Approach B — Enriched `end` with `summary:` (Terminal-Only Variant)

### Rationale

If the display is always the last step before termination, enriching `EndSpec` is a lower-cost
alternative: no new step type registration, no new executor branch. The `summary:` field is
displayed immediately before the run completes.

**Limitation:** Can only display content at the very end of a runbook. Cannot display mid-flow,
inside a branch arm, or after an iterate loop followed by further steps. This makes it
complementary to `type: display`, not a replacement.

### Go Struct Diff

```go
// EndSpec holds the fields for a step of type "end".
type EndSpec struct {
    Outcome *OutcomeDeclaration `yaml:"outcome,omitempty" json:"outcome,omitempty"`
    Summary string              `yaml:"summary,omitempty" json:"summary,omitempty"` // NEW
}
```

### Field Specification

#### `summary:` (optional, string)

A terminal message rendered to the operator immediately before the run record is closed.

- Template-rendered via `TemplateEvaluator` (same as `display.content`)
- Supports YAML multiline (`|`)
- If omitted, `end` behaves exactly as before — full backward compatibility
- `format:` is intentionally omitted from `end`; terminal summaries are expected to be plain
  text or minimal markdown. Adding `format:` to `end` in the future is non-breaking.

### YAML Example

```yaml
- id: done
  type: end
  outcome:
    category: success
    code: health-check-complete
  summary: |
    Health Check Report
    ===================

    {{ .report }}
```

### When to Prefer `end` + `summary:` Over `type: display`

| Scenario | Preferred |
|---|---|
| Display at end of runbook, always | `end` + `summary:` is sufficient |
| Display mid-flow (e.g., after iterate, before another step) | `type: display` required |
| Display conditionally (`when:`) | `type: display` (cleaner) |
| Display inside a `branch` arm | `type: display` required |
| Display with `format: markdown` for rich output | `type: display` (no `format:` on `end`) |

---

## Rejected Approaches

### `noop` + `output:` — Rejected

`noop` is explicitly documented as having zero type-specific fields. All behaviour comes from
common Step fields. Adding `output:` to `NoopSpec` violates its contract: "no action, no side
effect, no visible output." An operator writing `type: noop` expects nothing to be shown;
making that a lie by adding optional display output is a semantic trap.

Additionally, `noop` is used as a placeholder/spacer in flow designs. Silently rendering
content from a noop step would surprise authors who add `output:` for other reasons.

### Tool-based export — Rejected for this use case

Displaying content via `type: tool` (e.g., writing to a file then cat-ing it) introduces
unnecessary tool coupling, creates artefacts that may need cleanup, and requires the tool to
be declared and available. It is the right pattern for persistent artefacts (audit reports,
PDFs) but not for transient operator display.

---

## JSON Schema Fragment (Preferred Approach: `type: display`)

This fragment extends the step `$defs` in the gert runbook JSON Schema
(`https://schemas.gert.dev/runbook/v2`).

```json
{
  "$defs": {
    "DisplaySpec": {
      "type": "object",
      "description": "Renders content to the operator. No user input required.",
      "required": ["content"],
      "additionalProperties": false,
      "properties": {
        "content": {
          "type": "string",
          "minLength": 1,
          "description": "Content to display. Go text/template interpolation supported ({{ .varName }}). YAML multiline blocks (|) are preserved."
        },
        "format": {
          "type": "string",
          "enum": ["text", "markdown"],
          "default": "text",
          "description": "Render hint for the TUI. 'text': plain preformatted. 'markdown': render with rich formatting."
        },
        "title": {
          "type": "string",
          "description": "Optional heading rendered above content. Template-rendered."
        }
      }
    },

    "Step": {
      "if": {
        "properties": { "type": { "const": "display" } },
        "required": ["type"]
      },
      "then": {
        "properties": {
          "capture": { "not": {} },
          "retry":   { "not": {} },
          "required_evidence": { "not": {} },
          "export":  { "not": {} }
        },
        "required": ["id", "type", "content"],
        "allOf": [{ "$ref": "#/$defs/DisplaySpec" }]
      }
    }
  }
}
```

**Notes on the schema fragment:**
- `"not": {}` in the `then` block is the JSON Schema Draft 2020-12 idiom for "this property must be absent."
- `"required": ["id", "type", "content"]` enforces that `content` is always present when `type` is `display`.
- The `if/then` pattern matches the existing approach used for other step type constraints in the bundle.

---

## Forward Compatibility — Fields We May Want Later

These are deliberately excluded now but designed to be addable without breaking changes:

| Field | Future use | Why omitted now |
|---|---|---|
| `pause: bool` | Require operator to press a key to continue | Blurs into `approve` semantics; needs UX decision |
| `level: info\|warn\|error` | Colour/icon hint for severity | No current severity model in display context |
| `format: json` | Pretty-print a JSON variable | Complex: requires knowing the variable is JSON |
| `format: table` | Render array-of-maps as a table | Requires structured data, not a string variable |
| `truncate: int` | Max lines before "... N more lines" | Can be added as opt-in; no current need |
| `capture: acknowledged` | Record that operator saw the display | Future audit trail; current evidence model handles this differently |

---

## Implementation Notes for Brian

1. Register `StepTypeDisplay StepType = "display"` in `pkg/schema/step.go`.
2. Add `DisplaySpec *DisplaySpec yaml:",inline" json:"-"` to `Step` struct.
3. Engine executor for `display` calls `TemplateEvaluator.Eval(content, vars)`, then writes to the configured output stream (same stream as CLI stdout when `--output=text`).
4. Semantic validator adds rules: no `capture`, no `retry`, no `required_evidence` on `display` steps.
5. TUI layer reads `format` and `title` to render appropriately; engine is agnostic.
6. Trace event: `{"kind": "step_display", "step_id": "...", "format": "...", "rendered_bytes": N}` — do not log rendered content to avoid leaking sensitive variable values.

---

## Decision Recommendation

**Adopt both approaches:**

1. **Add `type: display`** as the primary, general-purpose solution. This resolves G1 for the
   `collect-health` use case and any future mid-flow display need.

2. **Add `summary:` to `EndSpec`** as a low-cost complement for the common pattern of displaying
   a final result on termination. It is additive, backward-compatible, and requires only a one-field
   struct change.

Neither approach conflicts with the other. Together they cover 100% of the "display without input"
use cases identified.
