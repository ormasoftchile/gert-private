# gert v2 Schema Validation Methodology

**Author:** John (Schema & YAML Specialist)  
**Date:** 2026-04-18  
**Purpose:** Rigorous stress-testing methodology for translating real-world prose runbooks to gert v2 schema  
**Status:** Draft v1

---

## Executive Summary

This document defines a systematic methodology for translating prose runbooks into gert v2 schema and validating that ALL semantics are fully expressed and retained without ambiguity. The methodology provides binary pass/fail criteria, gap classification, and a scoring rubric to identify schema weaknesses.

The core question: **Can we express this real-world operational procedure in gert v2 schema such that no semantically-meaningful information is lost?**

---

## 1. Translation Protocol

### 1.1 Step-by-Step Translation Process

Follow this protocol when translating a prose runbook to gert v2 schema:

#### 1.1.1 Pre-Translation Phase

1. **Read the entire runbook** — Understand the full scope before beginning translation
2. **Identify the runbook kind** — Classify as `mitigation`, `reference`, `composable`, or `rca` based on purpose
3. **Extract metadata** — Identify runbook name, description, and any contextual information
4. **List all tools mentioned** — Note every external tool, command, or system referenced
5. **Identify all data flows** — Mark where information is produced, transformed, and consumed

#### 1.1.2 Structural Decomposition

**Step Boundary Identification:**

A step boundary exists when:

- A discrete action is described (command execution, tool invocation, human decision)
- A new actor takes control (system → human, or human → system)
- Data is captured or transformed
- Control flow changes (branching, looping, delegation)
- Time elapses deliberately (wait, retry, poll)
- A terminal outcome is reached

**Negative cases (NOT step boundaries):**

- Explanatory prose or context (belongs in `title`, `subtitle`, or `prose` section)
- Multiple related actions by the same actor in sequence (may be one step with multiple `args`)
- Conditional logic expressed as part of an action (use `when` guard)

#### 1.1.3 Step Type Classification

For each identified step, apply this decision tree:

```
1. Is this a command-line invocation?
   YES → type: cli
   
2. Is this calling a named tool?
   YES → type: tool
   
3. Does the user select from predefined options and store the result?
   YES → type: choice
   
4. Does the user pick an execution path (branch to different runbooks/sections)?
   YES → type: decision
   
5. Does the user provide unstructured input (text, files, screenshots)?
   YES → type: collector
   
6. Does this invoke another runbook inline?
   YES → type: invoke
   
7. Is this conditional logic with multiple arms?
   YES → type: branch
   
8. Does this repeat steps over a collection or until convergence?
   YES → iterate (TreeNode sibling, not a step type)
   
9. Are these independent steps running concurrently?
   YES → type: parallel
   
10. Does this verify a condition and fail if false?
    YES → type: assert
    
11. Does this register a rollback action for saga pattern?
    YES → type: compensate
    
12. Is this a terminal outcome declaration?
    YES → type: end
```

**Disambiguation rules:**

- **choice vs. decision:** If the result is a VALUE stored in a variable → `choice`. If the result is a CONTROL FLOW fork → `decision`.
- **branch vs. decision:** `branch` is automatic (condition evaluated by template); `decision` is human-driven (user picks).
- **collector vs. tool:** `collector` is unstructured user input; `tool` invokes a defined external action.
- **iterate vs. parallel:** `iterate` repeats the SAME steps sequentially; `parallel` runs DIFFERENT step groups concurrently.

#### 1.1.4 Modeling Branching

**Automatic branching (condition-driven):**

Use `type: branch` or inline `branches:` sibling:

- Each branch has a `condition` (Go template returning bool)
- Exactly one arm executes (first match wins)
- Use `else: true` for catch-all arm
- Empty branch arm = execution continues past the branch

**Human-driven branching (user decision):**

Use `type: decision`:

- User is presented with 2+ routes
- Each route has `runbook` (invoke target) XOR `goto` (jump to step ID)
- Optionally store chosen label in `variable` for audit trail

**Key distinction:** If the prose says "if condition X, do Y", use `branch`. If it says "ask the operator which path to take", use `decision`.

#### 1.1.5 Modeling Data Flow

**Variable sources:**

1. **Inputs** — Declared in top-level `inputs:` map
   - Must specify `type` (string, number, boolean, secret)
   - Use `from:` binding to declare resolution source (prompt, env, file, provider, var)
   - Mark `required: true` if mandatory

2. **Vars** — Default values in top-level `vars:` map

3. **Captures** — Output from step execution
   - `cli` steps: capture `stdout`, `stderr`, `exit_code`
   - `tool` steps: capture `stdout`, `stderr`, or `json.<path>` for structured data
   - Store in named variable via `capture:` map

**Variable consumption:**

- Reference via Go template: `{{ .variable_name }}`
- Available in: `title`, `subtitle`, `args`, `when`, `condition`, `stdin`, tool args, etc.

**Scope rules:**

- Global scope by default (all steps see all captures)
- Use `scope:` field on a step to create isolated scope
- Use `export:` array to publish specific variables from scope

**Write-before-read validation:**

- Semantic validator checks that variables are written before being read
- Branching creates multiple paths; validator warns if write is only on some paths

#### 1.1.6 Modeling Failure and Compensation

**Failure handling:**

1. **Default behavior:** Step failure halts execution
2. **Continue on failure:** Set `continue_on_fail: true` to proceed past failure
3. **Retry logic:** Use `retry:` object with `max`, `interval`, `backoff`

**Compensation (saga pattern):**

Use `type: compensate` to register rollback actions:

- Compensation steps execute in LIFO order on failure/cancel
- `compensate.on:` specifies trigger (`failure`, `cancel`, `any`)
- Must appear AFTER the step being compensated in tree walk order

**Example pattern:**

```yaml
- step:
    id: scale_down
    type: tool
    title: Scale service to 0 replicas
    tool: {name: kubectl, action: scale, args: {replicas: "0"}}

- step:
    id: compensate_scale
    type: compensate
    compensate:
      on: failure
      steps:
        - step:
            id: restore_scale
            type: tool
            title: Restore original scale
            tool: {name: kubectl, action: scale, args: {replicas: "{{ .original_count }}"}}
```

#### 1.1.7 Modeling Human Interaction

**Type-specific patterns:**

| Step Type    | Use When                                  | Stores Result In         | Affects Control Flow |
|--------------|-------------------------------------------|--------------------------|----------------------|
| `choice`     | User selects from dropdown/radio buttons  | Variable                 | No                   |
| `decision`   | User picks execution path                 | Variable (optional)      | Yes (routes)         |
| `collector`  | User provides text/files/images           | Multiple variables       | No                   |
| `approval`   | User approves/rejects a gate              | N/A (implicit)           | Yes (blocks/fails)   |

**Approval gates:**

- Not a step type — a property of `collector` steps
- Use `approvals:` field with `min`, `roles`, `timeout`, `on_timeout`, `escalate_to`
- Blocks execution until approval threshold met

**Example: Evidence collection with approval:**

```yaml
- step:
    id: collect_rca
    type: collector
    title: Collect root cause analysis evidence
    prompt: |
      Provide all relevant evidence for the incident investigation.
    fields:
      - {name: incident_summary, type: multiline, required: true}
      - {name: error_log, type: file, required: true}
      - {name: dashboard_screenshot, type: image, required: false}
    approvals:
      min: 1
      roles: [DRI, incident-commander]
      timeout: 30m
      on_timeout: fail
```

#### 1.1.8 Modeling Parallelism

Use `type: parallel` for fan-out/fan-in patterns:

**Characteristics:**

- Multiple independent step groups execute concurrently
- Each branch is isolated during execution (no shared variable writes)
- Join semantics: `wait_for: all | any | majority`
- After join, all captures merged (last-write-wins on conflicts)
- Failure handling: `on_failure: fail | continue`

**When to use:**

- Prose describes "simultaneously", "in parallel", "concurrently"
- Steps are explicitly independent (no sequential dependency)
- Order of execution doesn't matter for correctness

**When NOT to use:**

- Steps must execute in sequence (use normal flow)
- One step depends on output of another (use sequential flow with captures)

#### 1.1.9 Modeling Nested Runbooks

Use `type: invoke` for runbook composition:

**Declaration pattern:**

1. Declare import in `imports:` map (alias → path)
2. Use `invoke.runbook:` to reference import alias
3. Pass inputs via `invoke.inputs:` (template-expanded)
4. Extract outputs via `invoke.outputs:` (local-var ← child-output-name)

**Gate semantics:**

Use `invoke.gate.stop_if:` to halt parent execution based on child outcome category:

```yaml
invoke:
  runbook: connectivity-check
  inputs:
    service_name: "{{ .hostname }}"
  outputs:
    health: health_status
  gate:
    stop_if: [escalated]  # Halt parent if child escalates
```

**Circular invoke detection:**

Semantic validator ensures invoke graph is a DAG (no cycles).

---

## 2. Completeness Criteria

A translation is **COMPLETE** if ALL of the following criteria are satisfied:

### C1: Step Coverage
**Binary check:** Every action described in the prose has a corresponding step in the schema.

**How to verify:**
1. List all imperative verbs in the prose ("check", "restart", "verify", "ask", "collect")
2. For each verb, locate the step that performs that action
3. If any verb has no matching step → FAIL

### C2: Decision Point Representation
**Binary check:** Every decision point (human or automated) in the prose is represented by a schema construct.

**Decision point types:**
- Automated condition → `branch` with `condition:` field
- Human path selection → `decision` step
- Human value selection → `choice` step
- Approval gate → `approvals:` field on `collector` step

**How to verify:**
1. Identify all "if/else" statements in prose
2. Identify all "choose", "select", "decide" instructions
3. Identify all approval requirements
4. Each must map to a schema construct → else FAIL

### C3: Data Flow Completeness
**Binary check:** Every piece of information captured, transformed, or consumed in the prose is represented in the schema.

**How to verify:**
1. List all data items mentioned in prose (hostnames, IDs, status codes, logs, etc.)
2. For each item, trace its lifecycle:
   - **Source:** input declaration, var, capture, or hardcoded?
   - **Consumers:** which steps reference it?
3. If any data item has unknown source or unreachable consumers → FAIL

### C4: Timing and Sequencing
**Binary check:** All timing constraints (delays, timeouts, retries, polling) are expressed in the schema.

**Elements to check:**
- Explicit waits → `delay:` field
- Timeouts → `timeout:` field on step
- Retry logic → `retry:` object
- Polling loops → `iterate` with `until:` condition
- Approval timeouts → `approvals.timeout` and `on_timeout`

**How to verify:**
1. Search prose for time expressions ("wait 30 seconds", "timeout after 5 minutes", "retry 3 times")
2. Each must map to a schema timing field → else FAIL

### C5: Failure Path Coverage
**Binary check:** All described failure scenarios have corresponding schema constructs.

**Failure mechanisms:**
- Step failure → default behavior (halt) or `continue_on_fail: true`
- Compensation → `type: compensate` with `on: failure`
- Assertion violation → `type: assert`
- Timeout behavior → `on_timeout:` field

**How to verify:**
1. List all "if this fails" or "on error" clauses in prose
2. Each must map to a failure-handling construct → else FAIL

### C6: Human Interaction Fidelity
**Binary check:** All human interaction patterns are represented with the correct step type.

**Patterns:**
- "Select from options" → `type: choice`
- "Decide which path" → `type: decision`
- "Provide input/evidence" → `type: collector`
- "Approve/reject" → `approvals:` field

**How to verify:**
1. List all sentences addressing the operator
2. Classify each by interaction type
3. Each must use the appropriate step type → else FAIL

### C7: Parallelism Declaration
**Binary check:** All explicitly parallel activities are modeled with `type: parallel`.

**How to verify:**
1. Search prose for "simultaneously", "concurrently", "in parallel", "at the same time"
2. Each must use `type: parallel` → else FAIL
3. If prose specifies join semantics ("wait for all", "wait for any"), verify `join.wait_for` matches

### C8: Nested Runbook References
**Binary check:** All references to other runbooks are declared via `invoke` steps with proper imports.

**How to verify:**
1. Search prose for "see runbook X", "execute procedure Y", "follow checklist Z"
2. Each must have corresponding entry in `imports:` map
3. Each must be invoked via `type: invoke` step → else FAIL

### C9: Governance and Approval Requirements
**Binary check:** All approval gates, risk classifications, and governance requirements are expressed.

**How to verify:**
1. List all approval requirements in prose ("requires manager approval", "DRI must sign off")
2. Check for risk/compliance language ("high-risk", "requires change ticket")
3. Each must map to `approvals:` or `contract.effects:` → else FAIL

### C10: Terminal Outcomes
**Binary check:** All described end states are modeled with `type: end` steps.

**Outcome categories:** resolved, escalated, no_action, needs_rca

**How to verify:**
1. Identify all termination points in prose ("incident resolved", "escalate to on-call", "no action needed")
2. Each must be a `type: end` step with appropriate `outcome.category` → else FAIL

---

## 3. Fidelity Criteria

A translation has **HIGH FIDELITY** if ALL of the following criteria are satisfied:

### F1: Semantic Equivalence
**Binary check:** Executing the schema would produce the same observable outcomes as executing the prose description.

**How to verify:**
1. Walk through schema execution mentally
2. At each step, ask: "Would this produce the same result as described in prose?"
3. Pay attention to:
   - Variable bindings (same values captured?)
   - Conditional logic (same branches taken?)
   - Output artifacts (same data recorded?)
4. If any step would behave differently → FAIL

### F2: No Information Loss
**Binary check:** No semantically-meaningful information from the prose is absent in the schema.

**Acceptable omissions:**
- Explanatory prose that doesn't change behavior (can go in `prose:` section)
- Examples or illustrations (can go in `description`, `hint`, or `subtitle`)
- Formatting/style choices

**Unacceptable omissions:**
- Required inputs or parameters
- Conditional logic
- Error handling or compensation
- Data capture or transformation
- Approval/governance requirements

**How to verify:**
1. For each sentence in prose, ask: "Is this information represented in schema?"
2. If information affects behavior and is missing → FAIL

### F3: Execution Path Preservation
**Binary check:** All possible execution paths described in prose are possible in schema.

**How to verify:**
1. Enumerate all execution paths in prose (follow all "if/else", "choose", "loop" branches)
2. Enumerate all execution paths in schema (walk the flow tree)
3. Sets must match (schema paths = prose paths) → else FAIL

**Edge case:** If prose is ambiguous about paths, document the ambiguity in `prose:` section but pick ONE interpretation in schema (acceptable).

### F4: Variable Binding Correctness
**Binary check:** All variable references resolve correctly at runtime; no unbound variable access.

**How to verify:**
1. For each template expression in schema (`{{ .var_name }}`):
   - Trace backwards to find where `var_name` is written
   - Verify write happens on ALL paths leading to the read (or variable has default value)
2. If any read could be unbound → FAIL

**Note:** The semantic validator (Phase 2) checks this automatically. If validator passes, F4 passes.

### F5: Step Type Precision
**Binary check:** Each step uses the MOST PRECISE step type available.

**Anti-patterns (precision violations):**
- Using `type: cli` with `sh -c "complex script"` when a `type: tool` would be more appropriate
- Using `type: collector` with single text field when `type: choice` would be more appropriate
- Using `type: branch` for a decision that should be `type: decision` (human-driven)

**How to verify:**
1. For each step, ask: "Is there a more specific step type that would better capture the intent?"
2. If yes → FAIL

### F6: Timing Preservation
**Binary check:** All timing behaviors match prose description.

**How to verify:**
1. For each timing expression in prose, check corresponding schema field
2. Values must match (or be equivalent):
   - "wait 30 seconds" → `delay: "30s"`
   - "timeout after 5 minutes" → `timeout: "5m"`
   - "retry up to 3 times" → `retry.max: 3`
3. If any value differs → FAIL

### F7: Governance Alignment
**Binary check:** Schema governance settings match prose risk/approval requirements.

**How to verify:**
1. List all risk/approval language in prose
2. Check corresponding `approvals:` fields, `contract.effects:`, `governance:` settings
3. If approval threshold, roles, or effects mismatch → FAIL

### F8: Human Prompt Clarity
**Binary check:** All interactive steps have clear, unambiguous prompts derived from prose.

**How to verify:**
1. For each `choice`, `decision`, `collector` step, read the `prompt:` field
2. Ask: "Would an operator understand what action to take?"
3. If prompt is vague, missing context, or contradicts prose → FAIL

### F9: Deterministic Evaluation
**Binary check:** Template expressions in conditions/guards are deterministic given the variable context.

**Non-deterministic patterns to avoid:**
- Calling non-deterministic template functions in `condition:` (e.g., `now()`, `random()`)
- Conditions that depend on global state not captured in variables

**How to verify:**
1. For each `condition:`, `when:`, `until:` field, analyze template expression
2. If expression depends on anything other than variables in scope → FAIL

### F10: Artifact Integrity
**Binary check:** All evidence capture mechanisms (files, logs, screenshots) use proper artifact storage.

**How to verify:**
1. For prose mentioning "capture log", "attach screenshot", "save output":
   - Check for `capture:` field (cli/tool steps) or `type: file/image` field (collector steps)
2. If artifact mentioned in prose but not captured in schema → FAIL

---

## 4. Gap Classification

When a translation fails completeness or fidelity, classify the gap using this taxonomy:

### G1 — Missing Step Type
**Definition:** The prose describes an action pattern for which no step type exists.

**Symptom:** Translator forced to use workaround (e.g., `type: cli` with complex script, or multiple steps where one would be natural).

**Example:**
- Prose: "Send a notification to Slack channel #incidents"
- Schema: No `type: notify` step; must use `type: tool` or `type: cli` with curl

**Severity:** HIGH if workaround is verbose or error-prone; MEDIUM if workaround is clean.

**Remediation signal:** Consider adding a new step type (e.g., `type: notify`, `type: http`).

---

### G2 — Missing Field
**Definition:** A step type exists but lacks a field required to fully express the prose semantics.

**Symptom:** Required information has no place to go; must be stored in comments, `metadata:`, or external documentation.

**Example:**
- Prose: "Retry with exponential backoff, max delay 5 minutes"
- Schema: `retry:` has `max`, `interval`, `backoff` but no `max_interval` field

**Severity:** HIGH if information is behavioral; LOW if information is purely metadata.

**Remediation signal:** Extend the step type with a new optional field.

---

### G3 — Missing Flow Construct
**Definition:** A control flow pattern in the prose has no direct schema equivalent.

**Symptom:** Forced to use multiple branch/iterate constructs in an awkward combination, or cannot express the pattern at all.

**Example:**
- Prose: "Try procedure A; if that fails, try procedure B; if both fail, escalate"
- Schema: No `try/catch` or `fallback` construct; must use nested branches with failure flags

**Severity:** CRITICAL if pattern is common in operational runbooks; MEDIUM if rare.

**Remediation signal:** Add a new control flow construct (e.g., `type: fallback`, `type: try`).

---

### G4 — Missing Interaction Model
**Definition:** A human interaction pattern cannot be expressed with current step types.

**Symptom:** Interactive behavior described in prose has no corresponding schema construct.

**Example:**
- Prose: "Show the operator a live-updating dashboard during deployment"
- Schema: No `type: monitor` or streaming UI construct; best approximation is iterate+poll with text output

**Severity:** HIGH if interaction is critical to workflow; MEDIUM if optional/nice-to-have.

**Remediation signal:** Extend the interaction model (e.g., add `type: monitor`, add streaming support).

---

### G5 — Semantic Loss
**Definition:** The prose concept is expressible in schema, but only approximately; meaning is degraded.

**Symptom:** Translator must make assumptions, simplify, or lose nuance.

**Example:**
- Prose: "Ask the operator to verify the deployment looks healthy (check metrics, logs, canary signal)"
- Schema: `type: collector` with `fields: [{name: verified, type: text}]` captures the confirmation but loses the specific verification criteria

**Severity:** MEDIUM if degradation is minor; HIGH if critical context is lost.

**Remediation signal:** Add richer field types (e.g., structured checklists, nested forms).

---

### G6 — Verbosity / Workaround Required
**Definition:** The prose concept is fully expressible, but requires an awkward multi-step workaround.

**Symptom:** Schema is correct but significantly more complex than the prose description.

**Example:**
- Prose: "Wait until the deployment is complete (up to 10 minutes)"
- Schema: Requires an `iterate` block with `max` and `until` condition, plus a delay step inside the loop

**Severity:** LOW if workaround is clean; MEDIUM if workaround obscures intent.

**Remediation signal:** Add syntactic sugar or a more direct construct (e.g., `type: wait_until`).

---

## 5. Scoring Rubric

For each runbook translation, produce a scorecard:

### 5.1 Completeness Score

**Formula:**

```
Completeness Score = (Satisfied Criteria / Total Criteria) × 100%
```

Where Total Criteria = 10 (C1–C10).

**Interpretation:**

- **100%** → All criteria satisfied; translation is complete
- **90–99%** → Minor gaps; likely acceptable with notes
- **80–89%** → Significant gaps; requires remediation
- **<80%** → Translation is incomplete; major work needed

### 5.2 Fidelity Score

**Formula:**

```
Fidelity Score = (Satisfied Criteria / Total Criteria) × 100%
```

Where Total Criteria = 10 (F1–F10).

**Interpretation:**

- **100%** → Perfect semantic fidelity
- **90–99%** → High fidelity; minor ambiguities acceptable
- **80–89%** → Moderate fidelity loss; may affect correctness
- **<80%** → Low fidelity; schema does not accurately represent prose

### 5.3 Gap Inventory

List all identified gaps with classification:

**Format:**

| Gap ID | Type | Severity | Description | Affected Criteria |
|--------|------|----------|-------------|-------------------|
| G1-001 | G1   | HIGH     | No step type for webhook notification | C1, F5 |
| G3-002 | G3   | MEDIUM   | No try/fallback construct for error recovery | C5, F3 |

### 5.4 Composite Verdict

Apply this decision matrix:

```
IF Completeness ≥ 90% AND Fidelity ≥ 90% AND all CRITICAL gaps resolved:
  → PASS

ELIF Completeness ≥ 80% AND Fidelity ≥ 80% AND no CRITICAL gaps:
  → PASS WITH NOTES
  (Include mitigation plan for identified gaps)

ELSE:
  → FAIL
  (Translation is not production-ready; major gaps must be addressed)
```

### 5.5 Scorecard Template

```markdown
## Validation Scorecard

**Runbook:** [Name]
**Source:** [Prose document reference]
**Translator:** [Name]
**Date:** [YYYY-MM-DD]

---

### Completeness Score: X/10 (Y%)

| Criterion | Status | Notes |
|-----------|--------|-------|
| C1: Step Coverage | ✅ / ❌ | ... |
| C2: Decision Points | ✅ / ❌ | ... |
| C3: Data Flow | ✅ / ❌ | ... |
| C4: Timing | ✅ / ❌ | ... |
| C5: Failure Paths | ✅ / ❌ | ... |
| C6: Human Interaction | ✅ / ❌ | ... |
| C7: Parallelism | ✅ / ❌ | ... |
| C8: Nested Runbooks | ✅ / ❌ | ... |
| C9: Governance | ✅ / ❌ | ... |
| C10: Terminal Outcomes | ✅ / ❌ | ... |

---

### Fidelity Score: X/10 (Y%)

| Criterion | Status | Notes |
|-----------|--------|-------|
| F1: Semantic Equivalence | ✅ / ❌ | ... |
| F2: No Information Loss | ✅ / ❌ | ... |
| F3: Execution Paths | ✅ / ❌ | ... |
| F4: Variable Binding | ✅ / ❌ | ... |
| F5: Step Type Precision | ✅ / ❌ | ... |
| F6: Timing Preservation | ✅ / ❌ | ... |
| F7: Governance Alignment | ✅ / ❌ | ... |
| F8: Human Prompt Clarity | ✅ / ❌ | ... |
| F9: Deterministic Evaluation | ✅ / ❌ | ... |
| F10: Artifact Integrity | ✅ / ❌ | ... |

---

### Gap Inventory

| Gap ID | Type | Severity | Description | Remediation |
|--------|------|----------|-------------|-------------|
| ...    | ...  | ...      | ...         | ...         |

---

### Verdict: PASS / PASS WITH NOTES / FAIL

**Rationale:**
[Explain the verdict based on scores and gaps]

**Next Steps:**
[If PASS WITH NOTES or FAIL, list required actions]
```

---

## 6. Schema Improvement Signal

### 6.1 Gap Aggregation Across Runbooks

As multiple runbooks are translated, gaps accumulate. Track them in a gap registry:

**Registry format:**

| Gap Type | Count | Affected Runbooks | Common Pattern | Priority |
|----------|-------|-------------------|----------------|----------|
| G1       | 5     | incident-*, deploy-* | Webhook notifications missing | P0 |
| G3       | 3     | rollback-*, failover-* | Try/fallback pattern missing | P1 |
| G6       | 8     | Various | Wait-until requires verbose iterate | P2 |

### 6.2 Signal Types

#### Signal 1: Schema Extensions Needed

**Trigger:** G1 or G2 gaps appear in ≥3 runbooks with similar pattern.

**Action:** Design and implement a new step type, field, or construct.

**Example:**
- Gap: G1 (no webhook step type) appears in 5 runbooks
- Signal: Add `type: webhook` step with fields for URL, method, payload, headers

---

#### Signal 2: Spec Clarifications Needed

**Trigger:** Translators consistently misinterpret a schema construct, or validation passes but execution fails.

**Action:** Revise normative text in §03 to add examples, tighten definitions, or add "MUST/MUST NOT" language.

**Example:**
- Gap: Translators confused about when to use `branch` vs. `decision`
- Signal: Add disambiguation table and decision tree to §3.1.3 (Step Type Classification)

---

#### Signal 3: Design Limitations (Acceptable)

**Trigger:** G4 or G5 gaps appear but are inherent to gert's design philosophy.

**Action:** Document as an explicit non-goal; provide guidance on alternative approaches.

**Example:**
- Gap: G4 (no live-streaming dashboard UI)
- Signal: gert v2 is CLI/TUI-first and doesn't support rich graphical dashboards. Acceptable limitation. Workaround: use external dashboard URL in `prose:` section.

---

### 6.3 Improvement Workflow

When a gap accumulates to threshold:

1. **Triage:** Review all instances of the gap; confirm pattern is real and common
2. **Design:** Propose a schema extension, field addition, or spec clarification
3. **Prototype:** Implement the change in §03 normative spec
4. **Validate:** Re-translate affected runbooks using new construct; verify gap resolution
5. **Document:** Add examples to spec; update migration guide if breaking change
6. **Deprecate workarounds:** Update existing runbooks to use new construct

---

## 7. Usage Guidelines

### 7.1 When to Apply This Methodology

**Required:**
- Before declaring gert v2 schema "implementation-ready"
- When evaluating a new step type or major schema change
- As part of acceptance testing for Brian's parser and Ken's runtime

**Recommended:**
- When translating any real-world runbook from prose to schema
- When a user reports "I can't express X in gert"
- As part of quarterly schema review and quality assurance

### 7.2 Scoring Thresholds for Production Readiness

**For gert v2 schema to be considered production-ready:**

- At least 10 diverse real-world runbooks must be translated
- Average Completeness Score ≥ 95%
- Average Fidelity Score ≥ 95%
- Zero CRITICAL gaps (G1, G3 with HIGH severity)
- All HIGH-severity gaps have documented workarounds

### 7.3 Iteration Strategy

**If schema fails production readiness:**

1. Prioritize gaps by severity and frequency
2. Address P0 gaps (CRITICAL) first via schema extensions
3. Address P1 gaps (HIGH) next via spec clarifications or new constructs
4. Re-translate failed runbooks using updated schema
5. Repeat until production readiness thresholds met

---

## 8. Examples

### 8.1 Example Translation (Passing)

**Prose runbook:**

```
Runbook: DNS Health Check

1. Resolve the hostname using nslookup
2. Capture the IP address
3. If resolution fails, escalate to network team
4. Otherwise, verify the IP is in the expected subnet (10.0.x.x)
5. If verification passes, mark incident as resolved
6. If verification fails, escalate to infrastructure team
```

**Schema translation:**

```yaml
$schema: "https://schemas.gert.dev/runbook/v2.json"
apiVersion: runbook/v2
id: dns-health-check
name: DNS Health Check

inputs:
  hostname:
    type: string
    required: true
    from: prompt

flow:
  - step:
      id: resolve
      type: cli
      title: "Resolve {{ .hostname }}"
      command: nslookup
      args: ["{{ .hostname }}"]
      capture:
        output: stdout
        exit: exit_code

  - step:
      id: check_exit
      type: branch
      branches:
        - condition: '{{ ne .exit "0" }}'
          label: Resolution failed
          steps:
            - step:
                id: escalate_network
                type: end
                title: Escalate to network team
                outcome:
                  category: escalated
                  code: dns_resolution_failed

        - else: true
          label: Resolution succeeded
          steps:
            - step:
                id: verify_subnet
                type: assert
                title: Verify IP in expected subnet
                assert:
                  - type: contains
                    subject: "{{ .output }}"
                    expected: "10.0."

            - step:
                id: check_subnet
                type: branch
                branches:
                  - condition: '{{ contains .output "10.0." }}'
                    label: Subnet verified
                    steps:
                      - step:
                          id: resolved
                          type: end
                          title: DNS health check passed
                          outcome:
                            category: resolved
                            code: dns_healthy

                  - else: true
                    label: Subnet mismatch
                    steps:
                      - step:
                          id: escalate_infra
                          type: end
                          title: Escalate to infrastructure team
                          outcome:
                            category: escalated
                            code: dns_subnet_mismatch
```

**Scorecard:**

- Completeness: 10/10 (100%) — All steps, decisions, data flows represented
- Fidelity: 10/10 (100%) — Execution paths match prose exactly
- Gaps: None
- **Verdict: PASS**

---

### 8.2 Example Translation (Failing)

**Prose runbook:**

```
Runbook: Deploy Service with Canary

1. Send a Slack notification to #deployments: "Starting canary deploy"
2. Scale canary deployment to 10% traffic
3. Monitor error rate for 5 minutes; if error rate > 2%, roll back
4. If canary is healthy, scale to 100%
5. Send success notification to Slack
```

**Schema translation (attempt):**

```yaml
$schema: "https://schemas.gert.dev/runbook/v2.json"
apiVersion: runbook/v2
id: deploy-canary
name: Deploy Service with Canary

flow:
  - step:
      id: notify_start
      type: cli
      title: Notify deployment start
      command: curl
      args:
        - "-X"
        - "POST"
        - "https://hooks.slack.com/..."
        - "-d"
        - '{"text": "Starting canary deploy"}'

  - step:
      id: scale_canary
      type: tool
      tool:
        name: kubectl
        action: scale
        args:
          deployment: myservice-canary
          replicas: "1"

  # PROBLEM: No built-in way to "monitor for 5 minutes and conditionally rollback"
  # Must use verbose iterate + assert + compensate workaround

  - iterate:
      id: monitor_loop
      max: 10
      steps:
        - step:
            id: wait
            type: cli
            command: sleep
            args: ["30"]

        - step:
            id: check_errors
            type: tool
            tool:
              name: kubectl
              action: get-metrics
            capture:
              error_rate: json.error_rate

        - step:
            id: check_threshold
            type: branch
            branches:
              - condition: '{{ gt .error_rate "2.0" }}'
                label: Error rate exceeded
                steps:
                  - step:
                      id: rollback
                      type: tool
                      tool:
                        name: kubectl
                        action: scale
                        args:
                          deployment: myservice-canary
                          replicas: "0"
                  - step:
                      id: fail
                      type: end
                      outcome:
                        category: escalated
                        code: canary_failed

  # PROBLEM: Slack notification duplicated (G6: verbosity)
  - step:
      id: notify_success
      type: cli
      title: Notify deployment success
      command: curl
      args:
        - "-X"
        - "POST"
        - "https://hooks.slack.com/..."
        - "-d"
        - '{"text": "Canary deploy succeeded"}'
```

**Scorecard:**

- Completeness: 8/10 (80%)
  - ❌ C1: Slack notification requires awkward curl invocation (G1: missing step type)
  - ❌ C4: "Monitor for 5 minutes" requires verbose iterate workaround (G6: verbosity)
- Fidelity: 8/10 (80%)
  - ❌ F5: Using `cli` for Slack instead of more precise `type: notify` (G1)
  - ❌ F2: "Monitor" semantics lost in iterate loop (G3: missing flow construct)
- Gaps:
  - **G1-001:** No `type: notify` or `type: webhook` step type (HIGH severity)
  - **G6-001:** No `type: monitor` or `type: wait_until` construct (MEDIUM severity)
- **Verdict: PASS WITH NOTES**
  - Runbook is functionally correct but verbosity and precision gaps identified
  - Recommendation: Add `type: webhook` step and `type: monitor` construct to schema

---

## 9. Changelog

- **v1 (2026-04-18):** Initial methodology defined by John

---

## Appendix A: Step Type Quick Reference

| Step Type    | Purpose                                | Key Fields                        |
|--------------|----------------------------------------|-----------------------------------|
| `cli`        | Execute command-line tool              | `command`, `args`, `capture`      |
| `tool`       | Invoke named tool action               | `tool.name`, `tool.action`        |
| `choice`     | User selects from options              | `prompt`, `options`, `variable`   |
| `decision`   | User picks execution path              | `prompt`, `routes`, `variable`    |
| `collector`  | User provides unstructured input       | `prompt`, `fields`, `approvals`   |
| `invoke`     | Call another runbook inline            | `invoke.runbook`, `invoke.inputs` |
| `branch`     | Conditional execution (automated)      | `branches[].condition`            |
| `iterate`    | Loop over collection or until converge | `over`, `max`, `until`            |
| `parallel`   | Concurrent fan-out/fan-in              | `branches`, `join`                |
| `assert`     | Verify conditions                      | `assert[]`                        |
| `compensate` | Register rollback action (saga)        | `compensate.steps`, `on`          |
| `end`        | Declare terminal outcome               | `outcome.category`, `outcome.code`|

---

## Appendix B: Validation Checklist (Printable)

```
☐ C1: All actions in prose have schema steps
☐ C2: All decision points represented
☐ C3: All data flows traced
☐ C4: All timing constraints expressed
☐ C5: All failure paths covered
☐ C6: All human interactions use correct step type
☐ C7: Parallelism explicitly declared
☐ C8: Nested runbooks properly imported/invoked
☐ C9: Governance/approval requirements expressed
☐ C10: Terminal outcomes declared

☐ F1: Schema execution matches prose behavior
☐ F2: No behavioral information lost
☐ F3: All execution paths preserved
☐ F4: All variables bound correctly
☐ F5: Most precise step types used
☐ F6: Timing values preserved
☐ F7: Governance settings aligned
☐ F8: Interactive prompts clear
☐ F9: Conditions are deterministic
☐ F10: Artifacts captured with integrity

☐ Gap inventory complete
☐ Scores calculated
☐ Verdict assigned
☐ Next steps documented
```

---

**END OF METHODOLOGY**
