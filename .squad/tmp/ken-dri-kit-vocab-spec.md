# DRI Kit Vocabulary Spec — Implementation Brief for Brian

**Author:** Ken (Software Architect)
**Date:** 2026-07-21
**Status:** APPROVED — Brian may begin implementation
**Input:** Dennis's DRI Systems Survey, DRI Kit Manual (10 chapters), home kit reference pattern

---

## 1. Kit Identity

| Field | Value |
|-------|-------|
| **Kit name (YAML)** | `ops/v1` — declared in `kit:` field |
| **Kit namespace** | `gert.ops` — all step types prefixed `ops.*` |
| **Go module path** | `github.com/ormasoftchile/gert-domain-dri` |
| **Go module (during monorepo phase)** | `github.com/ormasoftchile/gert/domains/dri` |
| **Runbook file convention** | `*.ops.yaml` |
| **Core schema version** | `apiVersion: runbook/v2` (required) |

**Naming rationale:** The kit is called `ops/v1` in YAML (short, matches `kit:` field convention from the manual). The namespace `gert.ops` provides the step type prefix. The Go module uses `dri` in its path because the implementation is the DRI accountability model; the YAML-facing name `ops` is the user-facing brand.

---

## 2. Step Vocabulary

The `gert.ops` Kit defines **five step types** and **three evidence types**. This is the authoritative vocabulary from the DRI Kit Manual (chapters 02, 06, 07, 09).

### Deviation from Dennis's 8-Step Consensus List

Dennis's survey recommended 8 incident-lifecycle primitives. After architectural review, those 8 are **lifecycle concepts**, not step types. The DRI Kit Manual resolves them into 5 concrete step types through composition:

| Dennis's Concept | Resolution | Where It Lives |
|-----------------|------------|----------------|
| **notify** | Compiler side-effect + `ops.manual` evidence | Notification is injected by compiler when approval/incident steps are lowered; not a standalone step |
| **escalate** | `ops.approval.on_timeout: escalate` + `ops.incident.sla` | Escalation is a timeout policy within approval and incident wrappers |
| **investigate** | `ops.manual` with `requires_role: responder` | Investigation steps are manual procedures inside `ops.incident` |
| **mitigate** | `ops.cli` or `ops.manual` inside `ops.incident` | Mitigation is execution of CLI commands or manual procedures |
| **postmortem** | Outside runbook scope — evidence bundle is the PIR artifact | Manual ch.07: "Review stage is outside the runbook scope" |
| **acknowledge** | IC takeover in `ops.incident` + role assignment | Implicit when incident-commander takes DRI authority |
| **assign-role** | `meta.roles` declaration + `requires_role` field | Roles are metadata, not runtime steps |
| **status-update** | `ops.manual` with attestation evidence | Status updates are manual steps where the operator confirms communication |

**Architectural justification:** Adding 8 thin step types that compile to the same core primitives as the 5 existing types would create vocabulary bloat without semantic value. The 5 types in the manual were designed to be **compositional** — complex incident patterns are built by nesting `ops.manual` and `ops.cli` steps inside `ops.incident` and `ops.change-request` wrappers. This mirrors Terraform's approach (few resource types, rich composition) rather than AWS SSM's approach (many action types, shallow composition).

---

### 2.1 `ops.cli` — Audited Command-Line Step

**Purpose:** Execute an audited CLI command with governance (allowlist, redaction) and automatic evidence capture.

**Compilation target:** Core `cli` step + governance metadata annotations.

**Required fields:**
| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique step identifier |
| `type` | string | Must be `ops.cli` |
| `title` | string | Human-readable step title |
| `command` | string | Executable name (argv[0]) |

**Optional fields:**
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `args` | []string | `[]` | Command arguments; template expressions supported |
| `allowed_commands` | []string | inherits from `meta.governance` | Step-level command allowlist |
| `redact` | []RedactRule | `[]` | RE2 pattern/replace pairs for stdout/stderr |
| `capture` | map[string]string | `{}` | Variable name → output stream mapping |
| `evidence.auto` | bool | `false` | Auto-capture output as evidence |
| `evidence.label` | string | — | Label for auto-captured evidence (required when auto=true) |
| `requires_role` | enum | — | Role required to execute (dri, responder, etc.) |
| `timeout` | duration | — | Step execution timeout |

**Example YAML:**
```yaml
- step:
    id: rollback_deploy
    type: ops.cli
    title: "Rollback to previous version"
    command: kubectl
    args: [rollout, undo, "deployment/{{ .service_name }}", "-n", "{{ .namespace }}"]
    allowed_commands: [kubectl]
    evidence:
      auto: true
      label: "kubectl rollback output"
```

---

### 2.2 `ops.manual` — Manual Procedure with Evidence

**Purpose:** Present a human operator with instructions, require role-gated execution, and collect structured evidence.

**Compilation target:** Core `manual` step + role gate metadata + evidence spec.

**Required fields:**
| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique step identifier |
| `type` | string | Must be `ops.manual` |
| `title` | string | Human-readable step title |
| `requires_role` | enum | Role required to execute (dri, approver, change-manager, responder, incident-commander) |
| `instructions` | string | Markdown/text instructions; template expressions supported |

**Optional fields:**
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `evidence` | []EvidenceSpec | `[]` | Evidence requirements for this step |
| `timeout` | duration | — | Step completion timeout |

**Example YAML:**
```yaml
- step:
    id: verify_health
    type: ops.manual
    title: "Verify service health post-deploy"
    requires_role: dri
    instructions: |
      1. Open Grafana dashboard for {{ .service_name }}.
      2. Confirm error rate < 0.1% for 5 minutes.
      3. Screenshot the dashboard.
    evidence:
      - type: ops.evidence.screenshot
        label: "Post-deploy Grafana"
        required: true
```

---

### 2.3 `ops.approval` — Blocking Approval Gate

**Purpose:** Pause execution until one or more designated approvers explicitly sign off. Supports SLA timeout with escalation.

**Compilation target:** Core `manual` step with approval-specific evidence record + timeout/escalation metadata.

**Required fields:**
| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique step identifier |
| `type` | string | Must be `ops.approval` |
| `title` | string | Gate title shown to approvers |

**Optional fields:**
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `description` | string | — | Context for approvers; template expressions supported |
| `approvers` | []string | inherits from `meta.roles.approver` | Override approver list |
| `requires_any` | bool | `true` | `true`=any one suffices; `false`=all must sign |
| `timeout` | duration | inherits from `meta.sla.approval-timeout` | Approval timeout |
| `on_timeout` | enum | `"escalate"` | `escalate` or `abort` |
| `escalation_target` | string | inherits from `meta.sla.escalation-target` | Identity notified on timeout |
| `evidence` | []EvidenceSpec | `[]` | Evidence required from approver |

**Example YAML:**
```yaml
- step:
    id: change_board_signoff
    type: ops.approval
    title: "Change board sign-off"
    approvers: ["@change-board", "@sre-lead"]
    requires_any: true
    timeout: 24h
    on_timeout: escalate
    escalation_target: "@vp-engineering"
```

---

### 2.4 `ops.change-request` — Change Lifecycle Wrapper

**Purpose:** Group a deployment or rollback sequence under a formal change request lifecycle: pre-execution approval, step execution with evidence, rollback injection on failure, and change record closure.

**Compilation target:** Compiler expands to a **sequence** of core steps:
1. Pre-execution `manual` step (change-manager approval gate)
2. The contained `steps` (each individually compiled)
3. Post-execution `manual` step (DRI sign-off)
4. Conditional rollback branch (injected if `rollback` is specified and any step fails)

**Required fields:**
| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique step identifier |
| `type` | string | Must be `ops.change-request` |
| `title` | string | Human-readable wrapper title |
| `steps` | []Step | Change execution steps (nested, compiled individually) |

**Optional fields:**
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `change-id` | string | — | External ticket reference; templates supported |
| `risk` | enum | `"medium"` | `low`, `medium`, `high`, `emergency` |
| `rollback` | RollbackSpec | — | Rollback specification |
| `rollback.runbook` | string | — | Relative path to rollback runbook |
| `rollback.inputs` | map[string]string | `{}` | Input bindings for rollback |

**Example YAML:**
```yaml
- step:
    id: deploy_v2
    type: ops.change-request
    title: "Deploy v2.4.1 to production"
    change-id: "{{ .change_ticket }}"
    risk: low
    rollback:
      runbook: rollback-service.runbook.yaml
      inputs:
        service_name: "{{ .service_name }}"
    steps:
      - step:
          id: apply_manifests
          type: ops.cli
          command: kubectl
          args: [apply, -f, "{{ .manifest_path }}"]
```

---

### 2.5 `ops.incident` — Incident Response Wrapper

**Purpose:** Declare an operational incident and activate incident-response mode: IC authority takeover, SLA timers, mandatory evidence capture, and automatic escalation.

**Compilation target:** Compiler expands to a **sequence** of core steps:
1. `manual` step — incident declaration (IC confirmation, writes `incident/declared` trace event metadata)
2. The contained `steps` (each individually compiled, role-gated to responder/IC)
3. `manual` step — IC resolution sign-off (writes `incident/resolved` trace event metadata)
4. SLA timeout metadata injected as step-level timeouts on contained steps

**Required fields:**
| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique step identifier |
| `type` | string | Must be `ops.incident` |
| `title` | string | Incident title |
| `severity` | enum | `sev1`, `sev2`, `sev3`, `sev4` |
| `steps` | []Step | Incident response steps (nested) |

**Optional fields:**
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `incident-id` | string | — | External ticket reference |
| `commander` | string | inherits from `meta.roles.incident-commander` | IC identity |
| `sla.mitigation-timeout` | duration | — | Time to first mitigation |
| `sla.resolution-timeout` | duration | — | Total resolution time limit |
| `sla.escalation-target` | string | inherits from `meta.sla.escalation-target` | Notified on SLA breach |

**Example YAML:**
```yaml
- step:
    id: payment_incident
    type: ops.incident
    title: "Payment processing degradation"
    severity: sev2
    commander: "@ic-oncall"
    sla:
      mitigation-timeout: 30m
      resolution-timeout: 4h
    steps:
      - step:
          id: triage
          type: ops.manual
          title: "Initial triage"
          requires_role: responder
          instructions: "Check error rates, latency, recent deploys."
```

---

### 2.6 Evidence Types (shared sub-vocabulary)

Three evidence types usable in any `evidence:` list:

#### `ops.evidence.screenshot`
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | YES | `ops.evidence.screenshot` |
| `label` | string | YES | Short identifier in evidence bundle |
| `required` | bool | NO (default: false) | Step cannot complete without this |
| `description` | string | NO | Instructions for submitter |

#### `ops.evidence.command-output`
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | YES | `ops.evidence.command-output` |
| `label` | string | YES | Short identifier |
| `required` | bool | NO | Step cannot complete without this |
| `capture_from` | string | COND | Run variable to capture (required when not auto-captured) |

#### `ops.evidence.attestation`
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | YES | `ops.evidence.attestation` |
| `label` | string | YES | Short identifier |
| `required` | bool | NO | Step cannot complete without this |
| `prompt` | string | YES | Statement the executor affirms verbatim |

---

## 3. What Does NOT Belong in This Kit

| Concept | Reason for Exclusion | Where It Belongs |
|---------|---------------------|-----------------|
| `http` / HTTP request | Generic orchestration primitive; no DRI semantics | gert core (or future `gert.http` kit) |
| `cli` (unqualified) | Core gert step type; `ops.cli` extends it, not replaces it | gert core |
| `branch` / conditional | Core flow control primitive | gert core |
| `iterate` / loop | Core flow control primitive | gert core |
| `wait` / `sleep` | Generic orchestration primitive | gert core timeout/delay |
| Variable substitution | Core template engine feature | gert core |
| Retries / timeout (generic) | Core execution options on any step | gert core |
| `ops.notify` (standalone) | Notification is a side-effect the compiler injects during lowering of `ops.approval` and `ops.incident`, not a standalone step. Runbooks that need ad-hoc notification use `ops.cli` (e.g., `slack-cli post`) or `ops.manual` with instructions. | Compiler expansion; core `cli` for explicit notification |
| `ops.postmortem` | DRI Kit Manual ch.07 explicitly places post-incident review outside runbook scope. The evidence bundle IS the postmortem artifact. | Out of scope — PIR is a process, not a runbook step |
| `ops.triage` | Investigation/triage is `ops.manual` with `requires_role: responder`. No unique compilation semantics. | `ops.manual` variant |
| `ops.create-channel` | Platform-specific integration (Slack/Teams); belongs in a messaging integration, not DRI vocabulary | Future `gert.slack` or `gert.teams` kit |
| `ops.create-ticket` | Platform-specific (Jira/Linear); external system integration | Future `gert.jira` kit or core `tool` extension |

---

## 4. Model Types (for `domains/dri/pkg/model/`)

### 4.1 Top-Level Types

```go
// OpsRunbook is the top-level model for a parsed .ops.yaml file.
type OpsRunbook struct {
    APIVersion string       `yaml:"apiVersion"`
    Kit        string       `yaml:"kit"`
    Meta       RunbookMeta  `yaml:"meta"`
    Flow       []FlowNode   `yaml:"flow"`
}

type RunbookMeta struct {
    Name        string           `yaml:"name"`
    Kind        RunbookKind      `yaml:"kind"`
    Description string           `yaml:"description"`
    Roles       RoleDeclaration  `yaml:"roles,omitempty"`
    ChangeID    string           `yaml:"change-id,omitempty"`
    SLA         SLAPolicy        `yaml:"sla,omitempty"`
    Governance  GovernancePolicy `yaml:"governance,omitempty"`
    Inputs      []InputDef       `yaml:"inputs,omitempty"`
}

type RunbookKind string
const (
    KindChange      RunbookKind = "change"
    KindIncident    RunbookKind = "incident"
    KindOperational RunbookKind = "operational"
)
```

### 4.2 Role Types

```go
type RoleDeclaration struct {
    DRI               string   `yaml:"dri,omitempty"`
    Approver          StringOrList `yaml:"approver,omitempty"`
    ChangeManager     string   `yaml:"change-manager,omitempty"`
    Responder         StringOrList `yaml:"responder,omitempty"`
    IncidentCommander string   `yaml:"incident-commander,omitempty"`
    Observer          StringOrList `yaml:"observer,omitempty"`
}

// StringOrList handles YAML fields that accept either a single string or a list.
// Implement yaml.Unmarshaler.
type StringOrList []string

type RoleType string
const (
    RoleDRI              RoleType = "dri"
    RoleApprover         RoleType = "approver"
    RoleChangeManager    RoleType = "change-manager"
    RoleResponder        RoleType = "responder"
    RoleIncidentCommander RoleType = "incident-commander"
    RoleObserver         RoleType = "observer"
)
```

### 4.3 SLA and Governance Types

```go
type SLAPolicy struct {
    ApprovalTimeout  Duration `yaml:"approval-timeout,omitempty"`
    ExecutionTimeout Duration `yaml:"execution-timeout,omitempty"`
    EscalationTarget string   `yaml:"escalation-target,omitempty"`
}

type GovernancePolicy struct {
    AllowedCommands []string     `yaml:"allowed_commands,omitempty"`
    DeniedCommands  []string     `yaml:"denied_commands,omitempty"`
    DenyEnvVars     []string     `yaml:"deny_env_vars,omitempty"`
    Redact          []RedactRule `yaml:"redact,omitempty"`
}

type RedactRule struct {
    Pattern string `yaml:"pattern"`
    Replace string `yaml:"replace"`
}

// Duration wraps time.Duration with YAML marshaling (e.g., "30m", "4h").
// Follow home kit's duration.go pattern.
type Duration struct {
    time.Duration
}
```

### 4.4 Step Types

```go
// FlowNode is the discriminated union for all flow elements.
// Only one of the embedded types is non-nil.
type FlowNode struct {
    Step *Step `yaml:"step,omitempty"`
}

type Step struct {
    ID   string   `yaml:"id"`
    Type StepType `yaml:"type"`
    Title string  `yaml:"title"`

    // ops.cli fields
    Command         string            `yaml:"command,omitempty"`
    Args            []string          `yaml:"args,omitempty"`
    AllowedCommands []string          `yaml:"allowed_commands,omitempty"`
    Redact          []RedactRule      `yaml:"redact,omitempty"`
    Capture         map[string]string `yaml:"capture,omitempty"`

    // ops.manual fields
    RequiresRole    RoleType     `yaml:"requires_role,omitempty"`
    Instructions    string       `yaml:"instructions,omitempty"`

    // ops.approval fields
    Description      string   `yaml:"description,omitempty"`
    Approvers        []string `yaml:"approvers,omitempty"`
    RequiresAny      *bool    `yaml:"requires_any,omitempty"`
    OnTimeout        string   `yaml:"on_timeout,omitempty"`
    EscalationTarget string   `yaml:"escalation_target,omitempty"`

    // ops.change-request fields
    ChangeID string       `yaml:"change-id,omitempty"`
    Risk     RiskLevel    `yaml:"risk,omitempty"`
    Rollback *RollbackSpec `yaml:"rollback,omitempty"`

    // ops.incident fields
    Severity    Severity     `yaml:"severity,omitempty"`
    IncidentID  string       `yaml:"incident-id,omitempty"`
    Commander   string       `yaml:"commander,omitempty"`
    SLA         *IncidentSLA `yaml:"sla,omitempty"`

    // Shared fields
    Evidence    interface{}  `yaml:"evidence,omitempty"`  // See §4.5
    Timeout     Duration     `yaml:"timeout,omitempty"`
    Steps       []FlowNode   `yaml:"steps,omitempty"`     // For wrappers (change-request, incident)
}

type StepType string
const (
    StepCLI           StepType = "ops.cli"
    StepManual        StepType = "ops.manual"
    StepApproval      StepType = "ops.approval"
    StepChangeRequest StepType = "ops.change-request"
    StepIncident      StepType = "ops.incident"
)

type RiskLevel string
const (
    RiskLow       RiskLevel = "low"
    RiskMedium    RiskLevel = "medium"
    RiskHigh      RiskLevel = "high"
    RiskEmergency RiskLevel = "emergency"
)

type Severity string
const (
    Sev1 Severity = "sev1"
    Sev2 Severity = "sev2"
    Sev3 Severity = "sev3"
    Sev4 Severity = "sev4"
)
```

### 4.5 Evidence and Sub-Types

```go
type EvidenceSpec struct {
    Type        EvidenceType `yaml:"type"`
    Label       string       `yaml:"label"`
    Required    bool         `yaml:"required,omitempty"`
    Description string       `yaml:"description,omitempty"`
    // ops.evidence.command-output
    CaptureFrom string `yaml:"capture_from,omitempty"`
    // ops.evidence.attestation
    Prompt string `yaml:"prompt,omitempty"`
}

type EvidenceType string
const (
    EvidenceScreenshot    EvidenceType = "ops.evidence.screenshot"
    EvidenceCommandOutput EvidenceType = "ops.evidence.command-output"
    EvidenceAttestation   EvidenceType = "ops.evidence.attestation"
)

type RollbackSpec struct {
    Runbook string            `yaml:"runbook"`
    Inputs  map[string]string `yaml:"inputs,omitempty"`
}

type IncidentSLA struct {
    MitigationTimeout Duration `yaml:"mitigation-timeout,omitempty"`
    ResolutionTimeout Duration `yaml:"resolution-timeout,omitempty"`
    EscalationTarget  string   `yaml:"escalation-target,omitempty"`
}

type InputDef struct {
    Name        string `yaml:"name"`
    Description string `yaml:"description,omitempty"`
    Type        string `yaml:"type,omitempty"`
    Default     string `yaml:"default,omitempty"`
    Required    bool   `yaml:"required,omitempty"`
}
```

### 4.6 Design Note: Step as Flat Struct (not Discriminated Union)

The model uses a single `Step` struct with optional fields, not separate structs per step type. This mirrors how the YAML is authored (all fields on one object) and keeps the loader simple. The compiler validates that the correct fields are populated for each type.

**Alternative considered:** Separate `CLIStep`, `ManualStep`, `ApprovalStep`, etc. with a `StepVariant` interface. Rejected because:
- YAML unmarshaling is cleaner with a single target type
- The loader doesn't need to type-switch during parsing
- The compiler already validates field combinations — double validation adds no value
- This matches the v2 core schema pattern (single `Step` struct with inline specs)

---

## 5. Schema (for `domains/dri/pkg/schema/`)

JSON Schema validation rules that Brian should encode. The schema file should be embeddable via `//go:embed schema.json`.

### 5.1 Top-Level Schema

```
type: object
required: [apiVersion, kit, meta, flow]
properties:
  apiVersion: { const: "runbook/v2" }
  kit: { const: "ops/v1" }
  meta: { $ref: "#/$defs/RunbookMeta" }
  flow: { type: array, items: { $ref: "#/$defs/FlowNode" }, minItems: 1 }
```

### 5.2 RunbookMeta Schema

```
type: object
required: [name, kind, description]
properties:
  name: { type: string, pattern: "^[a-z0-9][a-z0-9-]*$" }
  kind: { enum: ["change", "incident", "operational"] }
  description: { type: string, minLength: 1 }
  roles: { $ref: "#/$defs/RoleDeclaration" }
  change-id: { type: string }
  sla: { $ref: "#/$defs/SLAPolicy" }
  governance: { $ref: "#/$defs/GovernancePolicy" }
  inputs: { type: array, items: { $ref: "#/$defs/InputDef" } }
```

### 5.3 Step Schema (Discriminated by `type`)

Use JSON Schema `if/then` to validate fields per step type:

**ops.cli:**
```
if: { properties: { type: { const: "ops.cli" } } }
then:
  required: [id, type, title, command]
  properties:
    command: { type: string, minLength: 1 }
    args: { type: array, items: { type: string } }
    allowed_commands: { type: array, items: { type: string } }
    redact: { type: array, items: { $ref: "#/$defs/RedactRule" } }
    capture: { type: object, additionalProperties: { enum: ["stdout", "stderr", "combined"] } }
    evidence: { $ref: "#/$defs/CLIEvidence" }
    requires_role: { $ref: "#/$defs/RoleType" }
    timeout: { $ref: "#/$defs/Duration" }
```

**ops.manual:**
```
if: { properties: { type: { const: "ops.manual" } } }
then:
  required: [id, type, title, requires_role, instructions]
  properties:
    requires_role: { $ref: "#/$defs/RoleType" }
    instructions: { type: string, minLength: 1 }
    evidence: { type: array, items: { $ref: "#/$defs/EvidenceSpec" } }
    timeout: { $ref: "#/$defs/Duration" }
```

**ops.approval:**
```
if: { properties: { type: { const: "ops.approval" } } }
then:
  required: [id, type, title]
  properties:
    description: { type: string }
    approvers: { type: array, items: { type: string }, minItems: 1 }
    requires_any: { type: boolean }
    timeout: { $ref: "#/$defs/Duration" }
    on_timeout: { enum: ["escalate", "abort"] }
    escalation_target: { type: string }
    evidence: { type: array, items: { $ref: "#/$defs/EvidenceSpec" } }
```

**ops.change-request:**
```
if: { properties: { type: { const: "ops.change-request" } } }
then:
  required: [id, type, title, steps]
  properties:
    change-id: { type: string }
    risk: { enum: ["low", "medium", "high", "emergency"] }
    rollback: { $ref: "#/$defs/RollbackSpec" }
    steps: { type: array, items: { $ref: "#/$defs/FlowNode" }, minItems: 1 }
```

**ops.incident:**
```
if: { properties: { type: { const: "ops.incident" } } }
then:
  required: [id, type, title, severity, steps]
  properties:
    severity: { enum: ["sev1", "sev2", "sev3", "sev4"] }
    incident-id: { type: string }
    commander: { type: string }
    sla: { $ref: "#/$defs/IncidentSLA" }
    steps: { type: array, items: { $ref: "#/$defs/FlowNode" }, minItems: 1 }
```

### 5.4 Shared Definitions

**RoleType:**
```
enum: ["dri", "approver", "change-manager", "responder", "incident-commander", "observer"]
```

**Duration:**
```
type: string
pattern: "^[0-9]+(s|m|h|d)$"
```

**EvidenceSpec:**
```
type: object
required: [type, label]
properties:
  type: { enum: ["ops.evidence.screenshot", "ops.evidence.command-output", "ops.evidence.attestation"] }
  label: { type: string, minLength: 1 }
  required: { type: boolean }
  description: { type: string }
  capture_from: { type: string }
  prompt: { type: string }
allOf:
  - if: { properties: { type: { const: "ops.evidence.attestation" } } }
    then: { required: [type, label, prompt] }
```

### 5.5 Cross-Field Validation (Semantic — compiler enforces, not JSON Schema)

These rules cannot be expressed in JSON Schema and must be enforced in the compiler:

1. If any step has `type: ops.change-request`, then `meta.roles.change-manager` MUST be declared → error `OPS002`
2. If any step has `type: ops.incident`, then `meta.roles.responder` MUST be declared → error `OPS003`
3. Every `ops.manual` step MUST have `requires_role` → error `OPS004`
4. If `ops.approval` has `on_timeout: escalate`, then either `escalation_target` on the step or `meta.sla.escalation-target` must exist
5. If `ops.cli` has `evidence.auto: true`, then `evidence.label` is required
6. `rollback.runbook` path must be validated as a relative path (no `..` traversal, no absolute paths)
7. Step IDs must be unique across the entire flow (including nested steps)

---

## 6. Compiler Mapping (for `domains/dri/pkg/compiler/`)

The compiler transforms `model.OpsRunbook` → YAML string (gert core `runbook/v2` format). Output uses only core primitives: `cli`, `manual`, `tool`, `branch`, `iterate`.

### 6.1 `ops.cli` → core `cli`

**Lowering:**
1. Map `command` → core `cli.command`
2. Map `args` → core `cli.args`
3. Map `allowed_commands` → core `governance.allowed_commands` (merged with runbook-level)
4. Map `redact` → core `governance.redact` entries
5. Map `capture` → core `cli.capture`
6. If `evidence.auto: true`, inject an additional core `manual` step immediately after the `cli` step:
   - ID: `{original_id}.evidence`
   - Type: `manual` (core)
   - Prompt: "Review and confirm captured output: {evidence.label}"
   - This manual step carries the evidence metadata
7. If `requires_role` is set, inject it as `x-ops-role` annotation in step metadata

**Compiler validation:**
- `command` must not be empty
- If `evidence.auto: true`, `evidence.label` must be present
- `capture` stream values must be one of: `stdout`, `stderr`, `combined`

### 6.2 `ops.manual` → core `manual`

**Lowering:**
1. Map `title` → core `manual.title`
2. Map `instructions` → core `manual.instructions`
3. Map `requires_role` → `x-ops-role` annotation
4. For each evidence entry, append evidence metadata as `x-ops-evidence-*` annotations
5. If evidence has `required: true`, the step gets a `condition` on the evidence variable being non-empty

**Compiler validation:**
- `requires_role` must be a valid role enum value
- `instructions` must not be empty

### 6.3 `ops.approval` → core `manual` (approval variant)

**Lowering:**
1. Emit a core `manual` step with:
   - Instructions = `description` field (or title if no description)
   - `x-ops-gate: approval`
   - `x-ops-approvers: [list]`
   - `x-ops-requires-any: true/false`
2. Inject `ops.evidence.attestation` as required evidence with prompt "I approve: {title}"
3. If `timeout` is set, map to core step `timeout`
4. If `on_timeout: escalate`:
   - Inject a `branch` step after the manual step that checks for timeout
   - The escalation branch emits an `x-ops-escalation-target` annotation
5. If `on_timeout: abort`:
   - The core step timeout causes run failure (default gert behavior)

**Compiler validation:**
- If `approvers` is empty AND `meta.roles.approver` is empty, emit warning (gate will accept any user)
- `on_timeout` must be `escalate` or `abort` (or empty)

### 6.4 `ops.change-request` → sequence of core steps

**Lowering (ordered expansion):**

```
{id}.pre-approval       → core manual (change-manager approval gate)
{id}.step.{child.id}    → compiled child steps (recursive)
{id}.post-signoff       → core manual (DRI attestation: "Change complete")
{id}.rollback-branch    → core branch (condition: any child failed)
  {id}.rollback.invoke  → core cli (gert run {rollback.runbook} with inputs)
```

1. **Pre-approval gate:** Emit `manual` step requiring `change-manager` role. Include change-id, risk, and step summary in instructions. Evidence: attestation "I authorize this change."
2. **Child steps:** Compile each step in `steps` recursively (they may be `ops.cli`, `ops.manual`, etc.)
3. **Post-execution sign-off:** Emit `manual` step requiring `dri` role. Evidence: attestation "Change execution complete."
4. **Rollback injection:** If `rollback` is specified, emit a `branch` step conditioned on any child step failure. The branch executes the rollback runbook via core `cli` (invoking `gert run {rollback.runbook} --input key=value`).

**Compiler validation:**
- `steps` must not be empty
- If `rollback.runbook` is set, validate the path format (but NOT file existence — that's the loader's job at a different phase)
- If runbook `meta.kind` is `change`, at least one `ops.change-request` step should exist (warning, not error)

### 6.5 `ops.incident` → sequence of core steps

**Lowering (ordered expansion):**

```
{id}.declare            → core manual (IC confirmation, incident declaration)
{id}.step.{child.id}    → compiled child steps (recursive)
{id}.resolve            → core manual (IC resolution sign-off)
```

1. **Declaration step:** Emit `manual` step requiring `incident-commander` role. Instructions include severity, incident-id, and commander identity. Evidence: attestation "I am assuming incident command." Metadata: `x-ops-incident-declared: true`, `x-ops-severity: {severity}`.
2. **Child steps:** Compile each step in `steps` recursively. Inject `x-ops-incident-id` metadata on each child.
3. **Resolution sign-off:** Emit `manual` step requiring `incident-commander` role. Evidence: attestation "Incident resolved. Root cause: [describe]." Metadata: `x-ops-incident-resolved: true`.
4. **SLA timeouts:** If `sla.mitigation-timeout` is set, apply it as `timeout` on the first child step. If `sla.resolution-timeout` is set, apply it as `timeout` on the resolution sign-off step. Timeout behavior: escalation to `sla.escalation-target`.

**Compiler validation:**
- `severity` must be a valid enum
- `steps` must not be empty
- If runbook `meta.kind` is `incident`, at least one `ops.incident` step should exist (warning)

### 6.6 Metadata Annotations (`x-ops-*`)

The compiler uses `x-ops-*` namespaced annotations to carry DRI Kit semantics through core GERT without modifying core schema. These annotations:
- Are preserved in the runbook YAML (core parser ignores unknown `x-*` fields)
- Are available in trace events for projection/reporting
- Do NOT affect core execution behavior

| Annotation | Carried On | Purpose |
|-----------|-----------|---------|
| `x-ops-role` | any step | Required role for execution |
| `x-ops-gate` | approval steps | Gate type identifier |
| `x-ops-approvers` | approval steps | Approver list |
| `x-ops-requires-any` | approval steps | Any-vs-all approval policy |
| `x-ops-escalation-target` | approval/incident | Escalation identity |
| `x-ops-evidence` | any step | Evidence spec (serialized) |
| `x-ops-incident-declared` | incident declaration | Marks incident start |
| `x-ops-incident-resolved` | incident resolution | Marks incident end |
| `x-ops-severity` | incident steps | Incident severity |
| `x-ops-incident-id` | incident child steps | External incident ticket |
| `x-ops-change-id` | change-request steps | External change ticket |
| `x-ops-risk` | change-request | Risk classification |

### 6.7 Step ID Convention

All compiled step IDs follow the naming convention from the Kit reverse mapping decision:

```
ops.{wrapper-type}.{wrapper-id}.{sub-element}
```

Examples:
- `ops.change-request.deploy_v2.pre-approval`
- `ops.change-request.deploy_v2.step.pull_image`
- `ops.change-request.deploy_v2.post-signoff`
- `ops.incident.payment_incident.declare`
- `ops.incident.payment_incident.step.triage`
- `ops.incident.payment_incident.resolve`

For non-wrapper steps (ops.cli, ops.manual, ops.approval), the ID is preserved as-is from the source YAML.

---

## 7. Kit Declaration in Runbook YAML

### 7.1 Declaration Syntax

```yaml
apiVersion: runbook/v2
kit: ops/v1

meta:
  name: deploy-payment-service
  kind: change
  description: "Deploy payment service v2.4.1"
```

The `kit: ops/v1` field activates the ops kit compiler. There is no `kits:` array in v1 — a runbook uses exactly one kit (or no kit for core-only runbooks).

### 7.2 Loader Behavior When `kit: ops/v1` Is Declared

1. **Validate kit availability:** Check that the `ops/v1` kit compiler is registered in the compiler registry. If not, error `OPS001`.
2. **Load with kit schema:** Apply the `ops/v1` JSON Schema for validation (in addition to core `runbook/v2` schema).
3. **Parse into kit model:** Unmarshal YAML into `model.OpsRunbook` (not core `schema.Runbook`).
4. **Compile:** Pass `model.OpsRunbook` to `compiler.Compile()` → core GERT YAML string.
5. **Emit:** Write compiled YAML to output path. The core parser/planner/engine sees only core step types.

### 7.3 Future: Multi-Kit Declaration

When gert supports multi-kit composition (v2.1+), the declaration will evolve to:

```yaml
apiVersion: runbook/v2
kits:
  - name: gert.ops
    version: "^1.0.0"
  - name: gert.k8s
    version: "^0.3.0"
```

For now, `kit: ops/v1` is the only supported syntax. Brian should implement the single-kit path. The loader interface should be designed to accommodate the future `kits:` array without breaking changes.

### 7.4 Compiler Entry Point

```go
package compiler

// Compile transforms a parsed OpsRunbook into gert core runbook YAML.
// Returns the YAML string ready for gert core parser consumption.
func Compile(ctx context.Context, runbook *model.OpsRunbook) (string, error)

// CompileToFlowNodes transforms a parsed OpsRunbook into a slice of
// core FlowNode structs (for programmatic consumers that don't need YAML).
// Optional — implement if Brian finds it useful for testing.
func CompileToFlowNodes(ctx context.Context, runbook *model.OpsRunbook) ([]map[string]interface{}, error)
```

---

## 8. Open Questions for Brian

### 8.1 `x-ops-*` Annotation Mechanism

The compiler needs to emit `x-ops-*` annotations on core steps. **Question:** Does the core `runbook/v2` schema allow arbitrary `x-*` fields on step objects? If not, Brian needs to either:
- (a) Use `metadata:` or `labels:` map on steps (check if core supports this)
- (b) Propose a schema extension for annotation passthrough
- (c) Encode annotations as YAML comments (lost during parsing — not viable)

**Recommendation:** Check `v2/pkg/schema/step.go` for `additionalProperties` or `x-*` support. If absent, raise with Ken for a core schema amendment.

### 8.2 Rollback Runbook Invocation

`ops.change-request` rollback compiles to invoking another runbook. **Question:** What is the core mechanism for one runbook to invoke another?
- (a) `cli` step running `gert run rollback.runbook.yaml --input k=v`
- (b) Core `invoke` step type (does this exist?)
- (c) `tool` step calling a gert-specific tool

**Recommendation:** Use option (a) for v1 — shell out to `gert run`. This keeps the compiler simple and avoids adding new core step types.

### 8.3 Evidence Metadata in Core YAML

Evidence specs (`ops.evidence.*`) are kit vocabulary. **Question:** How should evidence requirements be represented in the compiled core YAML?
- (a) As `x-ops-evidence` annotation (serialized JSON array)
- (b) As core `manual` step fields that the engine already understands
- (c) As collector fields (the core has a `collector` step type with structured fields)

**Recommendation:** Check if core `manual` or `collector` step types support structured evidence. If core has `collector` with `fields` (screenshot, attestation), map evidence specs to collector fields. If not, use `x-ops-evidence` annotations.

### 8.4 Role Enforcement at Runtime

The compiler injects `x-ops-role` annotations, but **who enforces them at runtime?**
- (a) The gert core engine checks `x-ops-role` before step execution (requires core awareness of kit annotations)
- (b) A gert.ops runtime extension registers a policy that reads `x-ops-role` annotations
- (c) Role enforcement is advisory only in v1 (annotation for audit trail, no runtime block)

**Recommendation:** Start with option (c) for v1. Role annotations are captured in the trace for audit. Runtime enforcement requires either core changes or an extension — defer to v1.1.

### 8.5 Duration Parsing

The home kit has `duration.go` for YAML duration marshaling. **Question:** Should the dri kit use the same implementation, or import a shared utility?

**Recommendation:** Copy the pattern from home kit's `duration.go` into `domains/dri/pkg/model/duration.go`. When both kits are extracted to separate repos, evaluate extracting a shared `gert-kit-common` module. For now, duplication is cheaper than premature abstraction.

### 8.6 Loader File Discovery

**Question:** Should the loader discover `.ops.yaml` files by convention, or accept explicit file paths?

**Recommendation:** Accept explicit file paths only (like home kit's loader). File discovery is the application layer's responsibility, not the kit's. `loader.LoadFile(ctx, path) (*model.OpsRunbook, error)`.

### 8.7 Integration Test Strategy

The home kit validated at the parser boundary (map-based YAML validation). **Question:** Should the dri kit follow the same strategy, or attempt deeper validation?

**Recommendation:** Follow home kit's proven pattern:
1. Compile fixture `.ops.yaml` files → core YAML strings
2. Unmarshal compiled YAML into `map[string]interface{}`
3. Assert structural correctness (all required core fields present, correct types)
4. Golden file comparison for regression detection

Do NOT attempt `schema.Runbook` type roundtrip (blocked by same inline spec issue documented in Phase 4).

---

## Appendix A: Package Layout

```
domains/dri/
├── go.mod                          # module github.com/ormasoftchile/gert/domains/dri
├── go.sum
├── pkg/
│   ├── model/
│   │   ├── model.go                # OpsRunbook, RunbookMeta, Step, etc.
│   │   ├── duration.go             # Duration type with YAML marshaling
│   │   ├── types.go                # Enums: StepType, RoleType, RiskLevel, Severity, etc.
│   │   └── stringorlist.go         # StringOrList YAML unmarshaler
│   ├── loader/
│   │   ├── loader.go               # LoadFile(ctx, path) → *model.OpsRunbook
│   │   └── loader_test.go
│   ├── compiler/
│   │   ├── compiler.go             # Compile(ctx, *OpsRunbook) → (string, error)
│   │   ├── compile_cli.go          # ops.cli lowering
│   │   ├── compile_manual.go       # ops.manual lowering
│   │   ├── compile_approval.go     # ops.approval lowering
│   │   ├── compile_changerequest.go # ops.change-request expansion
│   │   ├── compile_incident.go     # ops.incident expansion
│   │   ├── compiler_test.go        # Unit tests (per step type)
│   │   └── testdata/               # Golden files
│   └── schema/
│       ├── schema.go               # Schema() → []byte (embedded JSON Schema)
│       ├── schema.json             # Embedded JSON Schema
│       └── validate.go             # Validate([]byte) → []ValidationError
├── examples/
│   ├── deploy-service.ops.yaml     # Change request example
│   ├── incident-response.ops.yaml  # Incident response example
│   └── operational-check.ops.yaml  # Operational runbook example
├── integration_test.go             # Integration tests (go:build integration)
├── cmd/
│   └── ops-validate/
│       └── main.go                 # CLI tool: compile + pretty-print
└── README.md
```

## Appendix B: Compilation Example

**Input (`.ops.yaml`):**
```yaml
apiVersion: runbook/v2
kit: ops/v1
meta:
  name: deploy-api
  kind: change
  description: "Deploy API service"
  roles:
    dri: "@alice"
    change-manager: "@change-board"
flow:
  - step:
      id: deploy
      type: ops.change-request
      title: "Deploy API v2.0"
      risk: low
      steps:
        - step:
            id: apply
            type: ops.cli
            title: "Apply manifests"
            command: kubectl
            args: [apply, -f, manifests/]
            evidence:
              auto: true
              label: "kubectl apply output"
```

**Output (compiled core YAML):**
```yaml
apiVersion: runbook/v2
meta:
  name: deploy-api
  kind: reference
  description: "Deploy API service"
  governance:
    allowed_commands: [kubectl]
flow:
  - step:
      id: ops.change-request.deploy.pre-approval
      type: manual
      title: "Change-manager authorization: Deploy API v2.0"
      instructions: |
        Review and authorize the following change:
        Change: Deploy API v2.0
        Risk: low
        Steps: apply (kubectl apply)
      x-ops-gate: approval
      x-ops-role: change-manager
      x-ops-change-id: ""
      x-ops-risk: low
  - step:
      id: ops.change-request.deploy.step.apply
      type: cli
      title: "Apply manifests"
      command: kubectl
      args: [apply, -f, manifests/]
      x-ops-role: dri
  - step:
      id: ops.change-request.deploy.step.apply.evidence
      type: manual
      title: "Confirm evidence: kubectl apply output"
      instructions: "Review the captured output from 'Apply manifests'."
      x-ops-evidence: command-output
  - step:
      id: ops.change-request.deploy.post-signoff
      type: manual
      title: "DRI sign-off: Deploy API v2.0"
      instructions: "Confirm the change is complete and the service is healthy."
      x-ops-gate: attestation
      x-ops-role: dri
```

---

**End of Spec.**

*This document is Brian's implementation brief. All architectural decisions are recorded in `.squad/decisions/inbox/ken-dri-kit-vocab.md`. Questions in §8 should be resolved before the first PR.*
