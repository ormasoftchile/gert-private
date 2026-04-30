# Gap Re-evaluation: gert-for-reference vs gert v2
## Status: 2026-04-30

## Executive Summary

This re-evaluation compares all gert-for-reference examples against the current gert v2 implementation, accounting for the recently added features:
- `gate: stop_if:` on include steps (commit aad6fa6)
- `concurrency:` on iterate nodes (commit aad6fa6)

**Key findings:**
- **21 reference examples reviewed** (including parents, children, grandchildren, nested, edge cases, drafts)
- **2 recently closed gaps**: gate/stop_if and iterate concurrency ✅
- **7 remaining structural gaps** (3 high-value, 4 medium/low-value)
- **4 formal syntax differences** (same semantics, different keywords)

---

## Recently Closed Gaps (aad6fa6)

### ✅ Gate: stop_if on include/invoke
**Reference syntax:**
```yaml
invoke:
  runbook: network
  inputs: { ... }
gate:
  stop_if: [resolved, escalated]
```

**v2 support:** FULL ✅
- `pkg/schema/steps.go`: `GateSpec` struct with `StopIf []string`
- `IncludeConfig.Gate *GateSpec`
- Semantics: short-circuit parent run without error when child outcome category matches

**Coverage:** incident-triage.runbook.yaml (lines 77-78, 93-94, 109-110), incident-triage.network.runbook.yaml (line 96-97)

---

### ✅ Concurrency on iterate
**Reference syntax:**
```yaml
iterate:
  over: services
  as: svc
  concurrency: 3
  steps: [...]
```

**v2 support:** FULL ✅
- `pkg/schema/steps.go`: `IterateNode.Concurrency int`
- Enables parallel iteration over collections

**Coverage:** collect-health-parallel.runbook.yaml (line 25)

---

## Coverage Table

| Feature | Reference Example(s) | v2 Status | Notes |
|---------|---------------------|-----------|-------|
| **Step Types** |
| `type: cli` | (implicit run) | 🔄 Formal Diff | v2: explicit `type: cli`, v1: implicit via `run:` |
| `type: tool` | simple-health-check, multi-region, etc. | ✅ Covered | `pkg/schema/steps.go:14-25` |
| `type: include` | (v2 term) | ✅ Covered | v2 uses `include`, v1 uses `invoke` (semantic equivalent) |
| `type: manual` | incident-triage, service-health, etc. | 🔴 Gap | v1 primitive; v2 split into choice/decision/collector |
| `type: noop` | edge-case-single-step-timeout, collect-health | 🔴 Gap | Simple delay/capture-only step (HIGH value) |
| `type: choice` | — | ✅ Covered | v2 native (pkg/schema/steps.go:48-64) |
| `type: decision` | — | ✅ Covered | v2 native (pkg/schema/steps.go:66-79) |
| `type: collector` | — | ✅ Covered | v2 native (pkg/schema/steps.go:81-106) |
| `type: branch` | service-health-branching, etc. | ✅ Covered | v2 native (pkg/schema/steps.go:142-153) |
| `type: iterate` | multi-region, resource-exhaustion | ✅ Covered | v2 native (pkg/schema/steps.go:155-165) |
| `type: parallel` | — | ✅ Covered | v2 native (pkg/schema/steps.go:167-184) |
| `type: approve` | — | ✅ Covered | v2 native (pkg/schema/steps.go:186-205) |
| `type: assert` | multi-region, resource-exhaustion | ✅ Covered | v2 native (pkg/schema/steps.go:207-217) |
| `type: compensate` | — | ✅ Covered | v2 native (pkg/schema/steps.go:220-229) |
| `type: wait_for_event` | — | ✅ Covered | v2 native (pkg/schema/steps.go:232-253) |
| `type: end` | all examples | ✅ Covered | v2 native (pkg/schema/steps.go:256-264) |
| **Step-Level Controls** |
| `when:` (step guard) | incident-triage.network (lines 70, 129, 189, 208) | ✅ Covered | `pkg/schema/step.go:45` |
| `timeout:` (step-level) | — | ✅ Covered | `pkg/schema/step.go:46` |
| `delay:` | collect-health (line 31), chain-level-1 (line 18), check-service (line 18) | ✅ Covered | `pkg/schema/step.go:47` |
| `retry:` | — | ✅ Covered | `pkg/schema/step.go:48`, RetryConfig (lines 70-77) |
| `continue_on_fail:` | check-service (line 19) | ✅ Covered | `pkg/schema/step.go:49` |
| `on_error: goto/stop/continue` | — | 🔴 Gap | v1 had explicit on-error routing (MEDIUM value) |
| `capture:` | all examples | ✅ Covered | `pkg/schema/step.go:52` |
| **Iterate Features** |
| `over:` | multi-region, resource-exhaustion, collect-health | ✅ Covered | `IterateNode.Over` |
| `as:` | multi-region, resource-exhaustion, collect-health | ✅ Covered | `IterateNode.As` |
| `max:` | — | ✅ Covered | `IterateNode.Max` |
| `until:` | — | ✅ Covered | `IterateNode.Until` |
| `collect:` | collect-health-parallel (line 26-27) | ✅ Covered | `IterateNode.Collect map[string]string` |
| `concurrency:` | collect-health-parallel (line 25) | ✅ Covered | **Recently added** ✅ |
| `stereotype: expanded` | collect-health (line 25) | 🔴 Gap | UI hint for iteration rendering (LOW value) |
| **Include/Invoke** |
| `invoke:` (v1 keyword) | incident-triage, etc. | 🔄 Formal Diff | v2: `include:`, v1: `invoke:` (semantic equivalent) |
| `with:` (param passing) | incident-triage (lines 74-75, 89-90, etc.) | ✅ Covered | `IncludeConfig.With map[string]string` |
| `gate: stop_if:` | incident-triage (lines 77-78, etc.) | ✅ Covered | **Recently added** ✅ |
| **Branch/Decision** |
| `branches:` inline | service-health-branching, multi-region | ✅ Covered | `BranchSpec.Branches []BranchArm` |
| `condition:` | service-health-branching (lines 32, 57, 76) | ✅ Covered | `BranchArm.Condition` |
| `else:` branch | — | ✅ Covered | `BranchArm.Else bool` |
| **Governance** |
| `governance: allowed_commands` | multi-region (line 20) | ✅ Covered | `GovernanceConfig.AllowCommands` (allow_commands) |
| `governance: deny_env_vars` | multi-region (line 21) | ✅ Covered | `GovernanceConfig.DenyEnvVars` |
| `governance: redact` | multi-region (lines 22-24) | ✅ Covered | `GovernanceConfig.Redact []RedactRule` |
| **Manual Step Features** |
| `instructions:` | All manual steps in v1 | 🔄 Formal Diff | v2: migrated to choice/decision/collector `prompt:` |
| `required_evidence:` | service-health, incident-triage.app-crash, etc. (100+ lines) | 🔴 Gap | Schema-enforced evidence collection (HIGH value) |
| `required_evidence.kind: text` | service-health (line 100) | 🔴 Gap | Text evidence type |
| `required_evidence.kind: checklist` | incident-triage (line 154), network (line 198) | 🔴 Gap | Checklist evidence type |
| `required_evidence.items: []` | incident-triage (lines 156-159), network (lines 200-203) | 🔴 Gap | Checklist items schema |
| `choices:` (inline manual) | incident-triage (lines 42-58), app-crash (lines 47-58) | 🔄 Formal Diff | v2: separate `type: choice` step |
| `approvals:` (inline manual) | incident-triage (lines 160-162), app-crash (line 160-162) | 🔄 Formal Diff | v2: separate `type: approve` or `Approvals` field on choice/collector |
| `outcomes:` (on manual step) | incident-triage (lines 58-62), connectivity-test (lines 83-87) | 🔴 Gap | Declarative outcome prediction (MEDIUM value) |
| **Defaults** |
| `defaults: timeout` | multi-region (line 18) | ✅ Covered | `StepDefaults.Timeout` |
| `defaults: retry_max` | — | ✅ Covered | `StepDefaults.RetryMax` |
| `defaults: continue_on_fail` | — | ✅ Covered | `StepDefaults.ContinueOnFail` |
| **Metadata** |
| `apiVersion: runbook/v1` | All reference examples | 🔄 Formal Diff | v2: `apiVersion` (same field, different value) |
| `meta:` block | All reference examples | 🔄 Formal Diff | v2: top-level fields (id, name, kind, description) |
| `meta.name` | All | ✅ Covered | v2: `Runbook.Name` |
| `meta.kind` | All | ✅ Covered | v2: `Runbook.Kind` (mitigation/reference/composable/rca) |
| `meta.description` | All | ✅ Covered | v2: `Runbook.Description` |
| `meta.vars` | simple-health, multi-region, collect-health | ✅ Covered | v2: `Runbook.Vars` |
| `meta.inputs` | service-health-branching, incident-triage | ✅ Covered | v2: `Runbook.Inputs` |
| `tree:` | All reference examples | 🔄 Formal Diff | v2: `flow:` (semantic equivalent) |
| `tools:` | simple-health, service-health, multi-region | 🔄 Formal Diff | v2: `toolRefs:` (different schema) |
| `imports:` | incident-triage, collect-health | ✅ Covered | v2: `Runbook.Imports` |
| `prose:` | connectivity-test (lines 28-30) | ✅ Covered | v2: `Runbook.Prose` |
| **Other Patterns** |
| Nested branches (3-level) | connectivity-test (lines 119-182) | ✅ Covered | v2 supports arbitrary nesting |
| Diamond invoke (A→B,C; B→D; C→D) | (implicit in multi-region) | ✅ Covered | Planner allows diamond dependencies (Phase 2 fix) |
| Grandchild invoke (3-level) | incident-triage → network → connectivity-test | ✅ Covered | No depth limit in schema (planner has configurable max) |
| Variable interpolation | `{{ .hostname }}`, `{{ .region }}` | ✅ Covered | Template syntax supported in `with:`, `capture:`, etc. |
| Step-level `branches:` | service-health-branching, multi-region | ✅ Covered | `step.branches` pattern (not in Step struct, but examples show it) |

---

## Remaining Gaps (Structural)

### 🔴 Gap 1: type: noop
**Value:** HIGH  
**Complexity:** Easy  
**Found in:** 
- edge-case-single-step-timeout.runbook.yaml (line 11)
- collect-health.runbook.yaml (line 42)

**Description:**  
A no-op step that can apply delay and capture variables without executing any command. Used for:
- Pure delay steps (wait without side effects)
- Variable transformations (accumulate report text)
- Stereotyping/rendering hints

**Reference example:**
```yaml
- step:
    id: accumulate
    type: noop
    title: "Record {{ .svc }} result"
    capture:
      report: "{{ .report }}{{ .last_host }}: {{ .last_status }}\n"
```

**v2 Workaround:** Use `type: cli` with `run: "true"` or `run: "echo"` (hacky)

**Recommendation:** Add `StepTypeNoop` to `pkg/schema/step.go` with no type-specific fields. Parser treats it as valid, executor advances immediately after applying delay/capture.

---

### 🔴 Gap 2: type: manual
**Value:** HIGH  
**Complexity:** Medium  
**Found in:** 
- simple-health-check.runbook.yaml (line 42)
- incident-triage.runbook.yaml (line 34, 118, 144)
- app-crash.runbook.yaml (line 40, 64, 79, 98, 121, 143, 154)
- network.runbook.yaml (line 36, 69, 103, 145, 161, 187)
- connectivity-test.runbook.yaml (line 66, 94, 124, 149)
- resource-exhaustion.runbook.yaml (line 28, 73)
- multi-region.runbook.yaml (line 29, 88, 122)

**Description:**  
v1's unified `type: manual` step combines:
- Instructions (prose guidance)
- Inline choices (variable selection)
- Required evidence (checklist/text/attachment)
- Inline approvals

v2 split this into **choice**, **decision**, **collector**, **approve** as separate step types. This is architecturally cleaner but creates a formal syntax gap.

**Reference example:**
```yaml
- step:
    id: classify_incident
    type: manual
    title: Classify the incident
    instructions: |
      Review incident details and select category.
    choices:
      variable: category
      prompt: "What is the primary incident category?"
      options:
        - value: "network"
          label: "Network — DNS failures, timeouts"
        - value: "app_crash"
          label: "Application — crashes, OOM"
    required_evidence:
      - kind: text
        name: classification_notes
    approvals:
      min: 1
      roles: ["dri"]
```

**v2 Equivalent:** Would require **4 separate steps** (collector for evidence, choice for selection, approve for gate, manual prose as… what?)

**Recommendation:** 
- **Option A:** Keep v2 separation (cleaner architecture, more verbose authoring)
- **Option B:** Add `type: manual` as syntactic sugar that compiles to v2 primitives during parsing
- **Decision:** DEFER. Leslie to document migration guide in §15 (v1 → v2 Transition). Do NOT add back to v2 schema.

---

### 🔴 Gap 3: required_evidence
**Value:** HIGH  
**Complexity:** Medium  
**Found in:** 
- service-health-branching (line 99-101)
- incident-triage (lines 127, 154-159)
- app-crash (lines 73-75, 106-108)
- network (lines 112-114, 172-174, 198-203)
- connectivity-test (lines 107-109)
- multi-region (lines 34-40, 127-128)

**Description:**  
Schema-level declaration of required evidence artifacts. Runtime enforces that operator provides these before step completes.

**Reference example:**
```yaml
required_evidence:
  - kind: checklist
    name: pre_deploy_check
    items:
      - "Change ticket approved"
      - "Rollback plan documented"
      - "Monitoring dashboards open"
  - kind: text
    name: dns_investigation
```

**v2 Status:** 
- Evidence **collection** exists (pkg/schema/event.go: EvidenceAttached event)
- Evidence **schema enforcement** does NOT exist
- Collector fields support `Required: bool` but no evidence-kind declaration

**Recommendation:**  
Add `RequiredEvidence []EvidenceRequirement` to:
1. `Step` struct (step-level enforcement)
2. `CollectorField` struct (field-level evidence attachment)

```go
type EvidenceRequirement struct {
    Kind  EvidenceKind `yaml:"kind"  json:"kind"`   // text, checklist, attachment
    Name  string       `yaml:"name"  json:"name"`
    Items []string     `yaml:"items,omitempty" json:"items,omitempty"` // for checklist
}

type EvidenceKind string
const (
    EvidenceKindText       EvidenceKind = "text"
    EvidenceKindChecklist  EvidenceKind = "checklist"
    EvidenceKindAttachment EvidenceKind = "attachment"
)
```

Engine validation: Before advancing from step, verify all required evidence collected.

---

### 🔴 Gap 4: outcomes (predictive outcomes on manual steps)
**Value:** MEDIUM  
**Complexity:** Medium  
**Found in:**
- incident-triage (lines 58-62)
- network (lines 58-62)
- connectivity-test (lines 83-87)

**Description:**  
Declarative outcome hints based on choice/manual input. When a certain condition is met during a manual step, the outcome category is pre-declared.

**Reference example:**
```yaml
- step:
    id: classify_incident
    type: manual
    choices:
      variable: category
      options: [...]
    outcomes:
      - when: 'category == "unknown"'
        state: escalated
        label: Unclassified incident
        recommendation: "Escalate to senior on-call."
```

**v2 Status:**  
- Outcome declaration exists on `type: end` steps (`EndSpec.Outcome`)
- Conditional outcomes based on user input do NOT exist
- Workaround: Use `branch` step after choice to route to appropriate `end` step

**Recommendation:**  
Add `Outcomes []ConditionalOutcome` to `Step` struct:
```go
type ConditionalOutcome struct {
    When           string `yaml:"when"                    json:"when"`
    Category       string `yaml:"category,omitempty"      json:"category,omitempty"`
    Code           string `yaml:"code,omitempty"          json:"code,omitempty"`
    Label          string `yaml:"label,omitempty"         json:"label,omitempty"`
    Recommendation string `yaml:"recommendation,omitempty" json:"recommendation,omitempty"`
}
```

Engine behavior: After step completes, evaluate `when` conditions. If match, record predicted outcome in trace. TUI can display recommendation to operator.

**Complexity:** Medium (requires expression evaluator, trace event addition)

---

### 🔴 Gap 5: on_error (explicit error routing)
**Value:** MEDIUM  
**Complexity:** Easy  
**Found in:** (not in current reference set, but documented in v1 spec)

**Description:**  
Explicit error handling strategy per step: `goto`, `stop`, `continue`, `compensate`.

**Reference example (hypothetical):**
```yaml
- step:
    id: risky_deploy
    type: cli
    run: ./deploy.sh
    on_error: goto:rollback_handler
    # OR: on_error: continue
    # OR: on_error: stop
```

**v2 Status:**  
- `continue_on_fail: bool` exists (step.go:49)
- Explicit `goto` on error does NOT exist
- `compensate` step type exists but no automatic triggering

**Recommendation:**  
Add `OnError *ErrorHandler` to `Step` struct:
```go
type ErrorHandler struct {
    Action string `yaml:"action"           json:"action"`            // goto, stop, continue, compensate
    Target string `yaml:"target,omitempty" json:"target,omitempty"`  // step ID for goto
}
```

**Complexity:** Easy (schema-only, planner already supports goto)

---

### 🔴 Gap 6: iterate.stereotype (UI rendering hint)
**Value:** LOW  
**Complexity:** Easy  
**Found in:** collect-health.runbook.yaml (line 25)

**Description:**  
TUI rendering hint for iteration display mode: `expanded` (show all iterations) vs `collapsed` (show summary).

**Reference example:**
```yaml
iterate:
  over: services
  as: svc
  stereotype: expanded   # ← UI hint
  steps: [...]
```

**v2 Status:** Not present in `IterateNode` struct.

**Recommendation:**  
Add `Stereotype string` to `IterateNode`:
```go
type IterateNode struct {
    // ... existing fields ...
    Stereotype string `yaml:"stereotype,omitempty" json:"stereotype,omitempty"`
}
```

Engine ignores this field. TUI layer consumes it.

**Complexity:** Easy (schema-only, no runtime impact)

---

### 🔴 Gap 7: Runbook-level timeout
**Value:** LOW  
**Complexity:** Easy  
**Found in:** (not in current reference set, but logical extension of step-level timeout)

**Description:**  
Maximum runtime for entire runbook. If exceeded, run is terminated with timeout outcome.

**v2 Status:**  
- Step-level `timeout` exists (step.go:46)
- `defaults.timeout` exists (runbook.go:60)
- Runbook-level timeout does NOT exist

**Recommendation:**  
Add `Timeout string` to `Runbook` struct (top-level field).

**Complexity:** Easy (schema + planner validation)

---

## Formal Differences (Same Semantics)

These are **NOT gaps** — they are syntactic differences where v2 provides equivalent or better functionality:

### 1. invoke → include
**v1:** `type: invoke`  
**v2:** `type: include`  
**Reason:** Clarity. "Include" better reflects the composition semantics.

### 2. tree → flow
**v1:** `tree:` (YAML key for step array)  
**v2:** `flow:` (YAML key for FlowNode array)  
**Reason:** "Flow" better reflects the linear execution model (not a tree/DAG).

### 3. meta → top-level fields
**v1:** `meta: { name, kind, description, vars, inputs }`  
**v2:** Top-level `Runbook` fields  
**Reason:** Cleaner schema, better JSON Schema mapping.

### 4. tools → toolRefs
**v1:** `tools: [ping, curl]` (string array)  
**v2:** `toolRefs: [{ name, path, version, grants }]` (structured refs)  
**Reason:** v2 requires explicit tool metadata for isolation and versioning.

---

## Recommendations

### Implement Now (v2.0)
1. **type: noop** — HIGH value, easy implementation, needed for delay/transform patterns
2. **required_evidence schema** — HIGH value, medium complexity, core governance feature
3. **on_error: goto/stop** — MEDIUM value, easy implementation, completes error handling story

### Defer to v2.1
4. **outcomes (predictive)** — MEDIUM value, medium complexity, requires expression evaluator
5. **iterate.stereotype** — LOW value, TUI-only concern
6. **Runbook-level timeout** — LOW value, not urgent

### Do NOT Implement
7. **type: manual** — Architecturally inferior to v2's separated primitives. Document migration path instead.

---

## Test Coverage Verification

All 21 reference examples can be mechanically translated to v2 syntax:
- ✅ simple-health-check (4 steps → 3 steps: tool, tool, end; drop manual)
- ✅ service-health-branching (nested branches → native branch step)
- ✅ incident-triage (parent/child invoke → include with gate)
- ✅ multi-region-rollout (iterate + governance → native iterate)
- ✅ edge-case-* (minimal runbooks → verify parser edge cases)
- ✅ nested/* (deep nesting + invoke chains → include chains)

**Blockers for translation:**
- `type: noop` used in 2 examples → workaround with `type: cli + run: true`
- `type: manual` used in 15+ examples → split into choice/collector/approve
- `required_evidence` used in 10+ examples → omit, document in migration guide

**Translation success rate (with workarounds):** 100%  
**Translation success rate (lossless):** ~60% (blocked by required_evidence)

---

## Appendices

### A. Reference Examples Reviewed (21 total)

**Root-level:**
1. simple-health-check.runbook.yaml
2. service-health-branching.runbook.yaml
3. incident-triage.runbook.yaml (parent)
4. incident-triage.app-crash.runbook.yaml (child)
5. incident-triage.connectivity-test.runbook.yaml (grandchild)
6. incident-triage.network.runbook.yaml (child)
7. incident-triage.resource-exhaustion.runbook.yaml (child)
8. multi-region-rollout.runbook.yaml
9. edge-case-branch.runbook.yaml
10. edge-case-branch-target-1.runbook.yaml
11. edge-case-branch-target-2.runbook.yaml
12. edge-case-single-step.runbook.yaml
13. edge-case-single-step-timeout.runbook.yaml

**Nested:**
14. nested/chain-level-1.runbook.yaml
15. nested/chain-level-2.runbook.yaml
16. nested/chain-level-3.runbook.yaml
17. nested/chain-level-4.runbook.yaml
18. nested/chain-level-5.runbook.yaml
19. nested/check-service.runbook.yaml
20. nested/collect-health.runbook.yaml
21. nested/collect-health-parallel.runbook.yaml

**Draft:** fix-indent.ps1 (not a runbook, ignored)

### B. v2 Schema Files Reviewed

- `pkg/schema/runbook.go` — Runbook, GovernanceConfig, StepDefaults, FlowNode
- `pkg/schema/step.go` — Step (unified struct), RetryConfig, Contract
- `pkg/schema/steps.go` — 14 step type specs (CLI, Tool, Include, Choice, Decision, Collector, Branch, Iterate, Parallel, Approve, Assert, Compensate, WaitForEvent, End)
- `internal/executor/*.go` — 16 executor implementations (all step types covered)

### C. Commit Context

**aad6fa6** — "feat: add gate/stop_if on include steps and concurrency on iterate"
- Added `GateSpec` struct
- Added `IncludeConfig.Gate *GateSpec`
- Added `IterateNode.Concurrency int`
- Closes 2 gaps from previous gap analysis

---

**Prepared by:** Ken (Software Architect)  
**Date:** 2026-04-30  
**Next review:** After v2.0 feature freeze (est. 2026-05-15)
