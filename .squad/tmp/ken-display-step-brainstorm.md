# Brainstorm: Display / Presentation Step for gert v2

**Author:** Ken (Software Architect)  
**Date:** 2026-04-28  
**Requested by:** Cristian  
**Context:** `collect-health` runbook loops over services, accumulates a `report` variable,
and needs to DISPLAY that report to the operator. No current step type fits cleanly.

---

## 1. Problem Statement

The current step inventory has no dedicated "show this to the operator without requiring input" primitive.

| Current workaround | Why it fails |
|--------------------|--------------|
| `type: noop` + `title:` | `title` is a one-liner display label, not a body. The spec explicitly says noop has no output/side effects. Abusing title for multi-line reports violates the documented contract. |
| `type: collector` | Requires user input. Wrong semantic entirely. |
| `type: end` + outcome | Only fires at termination. Reports may be mid-flow. |
| `type: tool` + write-file | Output goes to a file. Operator must know where to look. Invisible in TUI naturally. |

This is a real gap — not a misuse of existing primitives.

---

## 2. Candidates Evaluated

### 2.1 `type: display` — New Dedicated Step Type (RECOMMENDED)

A step that renders content to the operator without blocking for input.

**Proposed schema:**
```yaml
- step:
    id: show_health_report
    type: display
    title: "Service Health Report"   # common field, rendered as heading
    display:
      content: "{{ .report }}"       # template-rendered body
      format: markdown               # text | markdown | table (default: text)
      pause: false                   # if true, operator must acknowledge before advancing
```

**`DisplaySpec` struct:**
```go
type DisplaySpec struct {
    Content string `yaml:"content"          json:"content"`
    Format  string `yaml:"format,omitempty" json:"format,omitempty"` // "text"|"markdown"|"table"
    Pause   bool   `yaml:"pause,omitempty"  json:"pause,omitempty"`
}
```

**Execution semantics:**
1. Evaluate `content` through the template engine (same evaluator used by `title`/`subtitle`)
2. Write rendered content to the operator surface (stdout in CLI; a panel/card in TUI)
3. If `pause: true` → wait for operator acknowledgement (Enter / "OK" button)
4. If `pause: false` → fire-and-continue; advance immediately
5. Record a `step_display` event in the trace (content included, subject to redaction)
6. Never fails (unless template evaluation fails)

**Behavior across surfaces:**
| Surface | `pause: false` | `pause: true` |
|---------|---------------|---------------|
| CLI / pipe | `fmt.Println(rendered)` | `fmt.Println(rendered)` + `bufio.Scanner` read line |
| TUI | Render card in step panel | Render card with "OK / Continue" button |
| Headless / CI | Write to stdout, auto-advance | Write to stdout, auto-advance (pause ignored — CI has no human) |
| `gert run --dry-run` | Emit trace event, no output | Emit trace event, no output |

**Template variables:** full support — same `{{ .varName }}` engine used everywhere.  
**Markdown rendering:** optional. In TUI, render with the same renderer used for `collector.prompt`. In CLI, either render to ANSI (if terminal) or strip to plaintext (if pipe).

**Why this is the right primitive:**
- Clean semantic: "show this, no input required"
- Consistent with taxonomy — every step type has a dedicated spec
- Works mid-flow, not only at termination
- Reuses the existing template evaluator — zero new engine surface
- Trace-compatible (content captured in JSONL event)
- Does not mutate state → `contract` does not apply (parallel to noop)
- `pause: false` is zero-cost in headless mode; `pause: true` degrades gracefully to auto-advance
- Operator-facing visibility is explicit in the runbook YAML — self-documenting

---

### 2.2 Tool-Based Export (`type: tool` + write-file / report-writer)

Use an existing tool step with a `gert/io` tool that writes content to stdout or a file.

```yaml
- step:
    id: export_report
    type: tool
    tool:
      name: gert/io
      action: write
      args:
        content: "{{ .report }}"
        dest: stdout
```

**Pros:**
- No new step type — composable, reuses existing infrastructure
- Report is a file artifact (audit trail, CI artifact upload)
- Can write multiple formats simultaneously (stdout + file + S3 sink)

**Cons (why it loses):**
1. **Invisible in TUI.** Tool execution is a background operation — content doesn't get rendered in the operator panel unless the TUI has special-cased `gert/io:write` to stdout. That's a leaky abstraction.
2. **Not self-evident in YAML.** A runbook author shouldn't have to know about `gert/io` to show text. The intent ("display this") is buried in tool invocation syntax.
3. **Tool registry coupling.** The `gert/io` tool must exist in the registry. A display step requires no external registration — it's a core runtime behavior.
4. **Headless mismatch.** In headless environments, stdout from a tool step is mixed with execution logs. No clean way to distinguish "tool output" from "operator message."
5. **Trace semantics differ.** Tool steps emit tool-execution events; display should emit a distinct `step_display` event for clear auditability.

**Verdict: Runner-up.** This approach works for the *export* concern (machine-readable file artifact), but fails the *display* concern (operator-visible, TUI-native). The two concerns are orthogonal — export is not display.

---

### 2.3 Extend `noop` with a `body:` / `message:` Field

Add an optional `body:` field to `NoopSpec` (or introduce a `message:` on the common Step).

```yaml
- step:
    id: show_report
    type: noop
    title: "Health Report"
    message: "{{ .report }}"   # new common field
```

**Pros:**
- No new step type
- Minimal schema change

**Cons:**
- **Violates noop's contract.** The spec (§ noop) explicitly states: "A noop step does not execute commands, invoke tools, wait for user input, or **perform any external actions**." Rendering output to an operator *is* an external action.
- **`NoopSpec` is an empty struct for a reason.** Its emptiness is load-bearing — it signals "no payload." Adding fields means noop is no longer noop.
- **Ambiguous semantics.** Is a noop with `message:` a display? A log line? A trace annotation? Dedicated step type answers this cleanly.
- **TUI complexity.** TUI would need a special case: "if noop has `message:`, render differently than regular noops." Better to have a distinct type with its own renderer.

**Verdict: Rejected.** This is a hack that corrupts a clean primitive.

---

### 2.4 Enrich `end` with an `output:` Block

Extend `EndSpec` to support rendering content immediately before the runbook exits.

```yaml
- step:
    id: finish
    type: end
    end:
      outcome:
        category: success
      output:
        content: "{{ .report }}"
        format: markdown
```

**Pros:**
- Natural fit when report is the terminal summary of a runbook
- Keeps the "exit screen" concept in one place

**Cons:**
- **Reports are not always terminal.** A multi-phase runbook might want to show partial results at multiple checkpoints — `end` only fires once.
- **Conflates termination with presentation.** These are independent concerns. An `end` step should declare outcome, not decide how to format a report.
- **Excludes the common case.** The `collect-health` example shows output mid-flow, then might continue with a `choice` (e.g., "attempt auto-remediation?"). The report needs to appear *before* that choice, not at end.

**Verdict: Viable only as a complement.** `end` could optionally include a summary (a nice UX touch — "runbook complete, here's what happened"), but it cannot be the primary display primitive.

---

### 2.5 Hybrid: `type: display` + Tool Export

Use `type: display` for operator-facing output AND a tool step for file artifacts.

```yaml
- step:
    id: show_report
    type: display
    display:
      content: "{{ .report }}"
      format: markdown

- step:
    id: save_report
    type: tool
    tool:
      name: gert/io
      action: write
      args:
        content: "{{ .report }}"
        dest: "reports/{{ .run_id }}.md"
```

**Assessment:** This is correct architecture — display and export ARE orthogonal concerns. The runbook author should be able to:
- Display for humans (always, any surface)
- Export for machines (optional, when audit/artifact needed)

This hybrid is not a separate approach — it's the natural composition once `type: display` exists.

---

## 3. Prior Art Analysis

| System | Pattern | Lesson |
|--------|---------|--------|
| **Shell scripts** | `echo`, `printf` | Display is a first-class primitive; no one abuses `sleep` (noop) to print output |
| **Ansible** | `debug` task with `msg:` | Dedicated display task, supports template vars, `verbosity` controls when it fires — proves dedicated type is the right call |
| **GitHub Actions** | `run: echo "..."` + step summary (`GITHUB_STEP_SUMMARY`) | Two separate mechanisms: stdout for operators, summary for rendered artifacts. Confirms display vs. export are orthogonal |
| **Dagger** | Terminal attachment (`container.terminal()`) | Interactive display is separate from execution |
| **Temporal** | Activities have return values; workflow code renders them | Display is a runtime concern, not an activity concern — analogous to: CLI/TUI renders display steps, engine just emits events |

**Key lesson from Ansible `debug`:** A `debug` task with `msg: "{{ var }}"` has been the right answer for 10+ years in operator automation. gert's equivalent is `type: display` with `content: "{{ .report }}"`. The precedent is clear.

---

## 4. Recommended Approach: `type: display`

### Decision

**Add `type: display` as a new step type in the Utility category.**

**Rationale:**
1. Semantic clarity — explicit intent, no overloading of existing types
2. Consistent with the schema pattern — each step type has a `*Spec` struct
3. Mid-flow capable — not constrained to termination
4. Template-powered — reuses existing evaluator
5. Surface-adaptive — TUI renders richly; CLI writes to stdout; headless auto-advances through `pause:true`
6. Trace-friendly — distinct `step_display` event type
7. Zero new engine complexity — no process execution, no I/O waiting (unless `pause: true`)
8. Proven pattern — mirrors Ansible `debug`, GitHub Actions step summary

### Schema Addition Required

1. Add `StepTypeDisplay StepType = "display"` to `step.go`
2. Add `DisplaySpec` struct to `steps.go`
3. Add `DisplaySpec *DisplaySpec` inline field to `Step`
4. Add `func (s *DisplaySpec) StepKind() string { return "display" }`
5. Update step taxonomy diagram and table in `03-schema-vnext.tex`
6. Add `§ step-display` subsection to spec

### Step Taxonomy Placement

`type: display` belongs in the **Utility** category alongside `noop` — it has no side effects on external systems, no user input, no flow control.

---

## 5. Key Design Constraints to Respect

1. **No input required.** `type: display` must NEVER block waiting for user input unless `pause: true` is set. Default is fire-and-continue.
2. **Headless safety.** `pause: true` MUST auto-continue in headless/CI mode. The runtime should detect TTY absence and skip the pause.
3. **Content is a template.** `content:` is evaluated through the standard template evaluator. All current variables in scope are available.
4. **Trace inclusion.** Rendered content (post-template-evaluation) SHOULD be recorded in the trace event. Subject to the same redaction rules as other captured values.
5. **No state mutation.** `type: display` does not `capture:` variables. If you want to capture while displaying, use a separate `noop` step with `capture:` or chain with `type: cli`.
6. **`contract:` does not apply.** Like `noop`, display has no effects, reads, or writes in the governance sense.
7. **Format defaults to `text`.** Markdown rendering is progressive enhancement — CLI may strip ANSI, TUI renders rich. Don't require markdown support in the engine.

---

## 6. Risks and Open Questions

### Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Large content floods CLI output | Medium | Add optional `max_lines:` or truncation hint; TUI can paginate |
| `pause: true` confuses headless runs | Medium | Auto-continue on non-TTY; emit a warning event in trace |
| `format: markdown` creates rendering inconsistency across surfaces | Low | Document: engine does no rendering; surface adapters decide. Spec only declares *intent* |
| Authors abuse `display` as a logger (hundreds of display steps) | Low | Document intended use: operator-facing summaries. `trace:` or `capture:` for internal logging |

### Open Questions

1. **Should `display` support a `target:` field?** (`stdout`, `stderr`, `tui-panel`, `log-only`) — likely over-engineering for v2.0. Defer.
2. **Should `end` optionally reference a `display` step's output as its summary?** Might be a nice UX touch — "runbook ended, summary: [rendered display]". Defer to v2.1.
3. **Multi-line template with YAML blocks:** How does `content: |` interact with the template evaluator? Should be fine — template evaluator handles strings regardless of YAML scalar style. Verify in implementation.
4. **`format: table` feasibility:** Table rendering from a Go template variable (string) is ambiguous. Either require structured input (map/slice variable) or defer `table` format to v2.1. Start with `text` and `markdown` only.
5. **Redaction of `content` in trace:** If `content` contains a variable that was marked `secret`, should the trace event redact the rendered content? Yes — but this requires the evaluator to propagate the secret-taint through template expansion. Note as a v2.0 constraint: "content is redacted at the variable-reference level, not the rendered string level."

---

## 7. Recommended YAML Example (Final Form)

```yaml
# collect-health runbook — final display step
- step:
    id: show_health_report
    type: display
    title: "Service Health Report"
    display:
      content: |
        ## Health Check Results

        {{ .report }}

        **Run completed at:** {{ .completed_at }}
      format: markdown
      pause: false

# Optional: export artifact for CI / audit trail
- step:
    id: export_health_report
    when: "{{ .export_enabled }}"
    type: tool
    tool:
      name: gert/io
      action: write
      args:
        content: "{{ .report }}"
        dest: "artifacts/health-{{ .run_id }}.md"
```

---

## 8. Summary

| | `type: display` | `type: tool` (export) | `noop` + title | `end` + output |
|-|----------------|----------------------|---------------|----------------|
| **Mid-flow** | ✅ | ✅ | ✅ (title only) | ❌ |
| **TUI-native** | ✅ | ❌ | Partial | ❌ |
| **Template body** | ✅ | ✅ | ❌ (title = one-liner) | Possible |
| **No input** | ✅ | ✅ | ✅ | ✅ |
| **Self-documenting** | ✅ | ❌ | ❌ | Partial |
| **Schema clean** | ✅ | ✅ | ❌ (abuses noop) | Partial |
| **Headless safe** | ✅ | ✅ | ✅ | ✅ |

**Winner: `type: display`** — clean, composable, consistent, mid-flow capable.  
**Runner-up: tool export** — valid for *artifacts*, orthogonal to display, not a replacement.

---

*Ken — Software Architect, gert v2*
