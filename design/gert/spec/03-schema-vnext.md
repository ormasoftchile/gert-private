# Schema vNext

This is the normative specification for all gert v2 data structures: runbook documents, tool definitions, and provider definitions. It supersedes all v1 schema documentation and defines the contracts for the parser, runtime, and VS Code extension. Design philosophy: *minimal core, extensible periphery* — core language fields are small, typed, and stable; everything else is declared as an extension with an explicit namespace.

## Schema Identity and Versioning

### apiVersion Values

Every gert document carries a mandatory `apiVersion` field.

| Document kind | v1 value | v2 value |
|---------------|----------|----------|
| Runbook | `runbook/v1` | `runbook/v2` |
| Tool definition | `tool/v0` | `tool/v2` |
| Provider definition | `provider/v0` | `provider/v2` |

Format: `<kind>/<major>`. Minor and patch changes do not alter the `apiVersion` string.

### Self-Description via `$schema`

Every v2 document SHOULD carry a `$schema` field:

```yaml
$schema: "https://schemas.gert.dev/runbook/v2.json"
apiVersion: runbook/v2
id: service-health
name: Service Health Check
```

Schema URLs:
```
https://schemas.gert.dev/runbook/v2.json
https://schemas.gert.dev/tool/v2.json
https://schemas.gert.dev/provider/v2.json
```

`$schema` is optional at runtime (the runtime uses `apiVersion` for dispatch) but strongly recommended for IDE validation and offline validation with any JSON Schema Draft 2020-12 validator (ajv, jsonschema, check-jsonschema).

### Versioning Strategy

- **Major increment** (e.g. `runbook/v1` → `runbook/v2`): A breaking change. The parser MUST NOT silently accept the old format. Automated migration via `gert migrate`.
- **Minor/patch changes within a major**: Additive, backward-compatible. New optional fields may be added; existing fields MUST NOT be removed or renamed.

**What constitutes a breaking change:**
- Renaming or removing a required field
- Narrowing an existing enum (removing a valid value)
- Changing a field type (e.g. string → object)
- Changing the meaning of a field such that existing runbooks produce different runtime behavior
- Removing a step type

Adding new optional fields, adding new enum values, and loosening existing constraints are **not** breaking changes.

### Compatibility Contract: v1 → v2

| v1 field | v2 status | Notes |
|----------|-----------|-------|
| `apiVersion: runbook/v0` | Removed | Handled by `gert migrate` |
| `apiVersion: runbook/v1` | Deprecated | Compatibility mode in v2 parser |
| `meta.name` | Promoted | Becomes top-level `name` |
| `meta.kind` | Promoted | Becomes top-level `kind` |
| `meta.description` | Promoted | Becomes top-level `description` |
| `meta.vars` | Retained | Moved to top-level `vars` |
| `meta.inputs` | Retained | Moved to top-level `inputs` |
| `meta.governance` | Retained | Moved to top-level `governance` |
| `meta.prose` | Retained | Moved to top-level `prose` |
| `meta.defaults` | Retained | Moved to top-level `defaults` |
| `imports` | Retained | Unchanged |
| `tools` | Replaced | Replaced by `toolRefs` (typed) |
| `flow` | Retained | Canonical execution root |
| `tree` | Removed | Use `flow` |
| `steps` | Removed | Flat step list was never canonical |
| `step.type: collector` | Renamed | Split into `choice`, `decision`, `collector` |
| `step.type: router` | Renamed | Becomes `type: end` |
| `step.type: choice` | New | User selects from options; stores result |
| `step.type: decision` | New | User picks runbook path; routes execution |
| `step.type: noop` | Removed | Use `when:` guard on any step |
| `step.type: extension` | Retained | Unchanged |
| `step.with.argv` | Deprecated | Use `type: cli` with `command:` |
| `step.assertions` | Deprecated | Use `step.type: assert` |

The v2 parser reads `runbook/v1` documents in **compatibility mode**: all v1 field paths are accepted and silently normalized to their v2 equivalents before validation. v2 will NOT support `runbook/v0`.

### `gert migrate` Behavior

```bash
# Dry-run preview
gert migrate --dry-run runbook.yaml

# In-place rewrite
gert migrate runbook.yaml

# Migrate a whole directory
gert migrate ./runbooks/
```

Migration rules applied automatically:
1. Rewrite `apiVersion` to `runbook/v2`.
2. Add `$schema` field.
3. Promote `meta.name`, `meta.kind`, `meta.description` to top-level.
4. Move remaining `meta.*` fields to their v2 top-level equivalents.
5. Rename `tree:` to `flow:`.
6. Replace generic `step.type: manual` with three precise types: `choice`, `decision`, `collector`.
7. Rename `step.type: router` → `end`.
8. Rewrite `tools: [name]` shorthand to `toolRefs:` typed declarations.
9. Add `id:` to runbook if absent (derived from `name`).

## Core Runbook Fields (v2)

### Top-Level Field Inventory

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `$schema` | string | No | JSON Schema URL (recommended) |
| `apiVersion` | string | Yes | Must be `runbook/v2` |
| `id` | string | Yes | Unique machine identifier (slug) |
| `name` | string | Yes | Human-readable display name |
| `kind` | enum | No | `mitigation \| reference \| composable \| rca` |
| `description` | string | No | Short prose summary |
| `vars` | map | No | Default variable values |
| `inputs` | map | No | Input declarations |
| `outputs` | map | No | Output declarations |
| `toolRefs` | array | No | Tool references |
| `imports` | map | No | Named runbook imports |
| `defaults` | object | No | Default step settings |
| `governance` | object | No | Governance policy |
| `prose` | object | No | Human-readable TSG sections |
| `metadata` | map | No | Arbitrary key-value annotations |
| `flow` | array | Yes | Execution tree (TreeNode array) |

**`id` field:** A slug matching `^[a-z][a-z0-9-]*[a-z0-9]$`. Uniquely identifies a runbook within a project. Changing it is a breaking change for any runbook that includes it by id.

**`kind` values:**
- `mitigation` — Incident response procedure; governance defaults to `require-approval` for destructive effects.
- `reference` — Documentation-first runbook; dry-run by default.
- `composable` — Sub-runbook intended for include; not directly executable from CLI.
- `rca` — Root-cause analysis template.

### toolRefs

```yaml
toolRefs:
  - name: curl
  - name: nslookup
  - name: k8s
    path: ./tools/k8s.tool.yaml   # override default discovery
    alias: kubectl-wrapper         # optional local alias
```

Default discovery path for `name: foo` is `tools/foo.tool.yaml` relative to the runbook file.

### Minimal Valid v2 Runbook

```yaml
$schema: "https://schemas.gert.dev/runbook/v2.json"
apiVersion: runbook/v2
id: hello-world
name: Hello World

flow:
  - step:
      id: greet
      type: cli
      title: Print greeting
      command: echo
      args: ["Hello, world!"]
      capture:
        output: stdout
  - step:
      id: done
      type: end
      title: Complete
      outcome:
        category: resolved
        code: success
```

### Complete Annotated Example

```text
$schema: "https://schemas.gert.dev/runbook/v2.json"
apiVersion: runbook/v2
id: service-health-check
name: Service Health Check
kind: mitigation
description: |
  Validate DNS resolution and HTTP endpoint reachability
  for a named service. Escalates on failure.

vars:
  default_timeout: "30s"

inputs:
  hostname:
    type: string
    required: true
    description: "Hostname to check"
    from: prompt
  incident_id:
    type: string
    required: false
    description: "Incident tracking ID"
    from: incident.id       # provider-resolved

outputs:
  health_status:
    type: string
    description: "Final health assessment"
    from: captures.http_result

toolRefs:
  - name: curl
  - name: nslookup

governance:
  rules:
    - effects: ["network.probe"]
      action: allow
    - effects: ["service.restart"]
      action: require-approval
      min_approvers: 2
  redact:
    - pattern: "(?i)Authorization:\\s*\\S+"
      replace: "Authorization: <redacted>"

defaults:
  timeout: "60s"

prose:
  safety: |
    Run this runbook only during an active incident. Do not
    loop — each execution creates a trace record.
  prerequisites: "Requires curl >= 7.80 and nslookup on PATH."

metadata:
  team: platform-sre
  runbook-url: "https://wiki.example.com/runbooks/service-health"
  severity: P2

flow:
  - step:
      id: check_dns
      type: tool
      title: "Resolve DNS: {{ .hostname }}"
      tool:
        name: nslookup
        action: lookup
        args:
          host: "{{ .hostname }}"
      capture:
        dns_result: stdout
      contract:
        effects: ["network.probe"]
        reads: ["hostname"]
        writes: ["dns_result"]

  - step:
      id: check_http
      type: tool
      title: "HTTP health check: {{ .hostname }}"
      tool:
        name: curl
        action: head
        args:
          url: "https://{{ .hostname }}/healthz"
      capture:
        http_result: stdout
      timeout: "{{ .default_timeout }}"

  - step:
      id: evaluate
      type: branch
      title: Evaluate health results
      branches:
        - condition: '{{ contains .http_result "200" }}'
          label: Healthy
          steps:
            - step:
                id: resolved
                type: end
                title: Service is healthy
                outcome:
                  category: resolved
                  code: http_ok
        - condition: '{{ not (contains .http_result "200") }}'
          label: Unhealthy
          steps:
            - step:
                id: escalate
                type: end
                title: Escalate to on-call
                outcome:
                  category: escalated
                  code: http_fail
```

## Input and Output Declarations

### Input Declarations

```yaml
inputs:
  <name>:
    type: string | number | boolean | secret
    required: true | false          # default: true
    description: "..."
    default: "..."                  # valid only when required: false
    example: "..."
    pattern: "^[a-z]+"             # regex validation (string only)
    from: <binding>                 # resolution source
```

**`type` field values:**
- `string` — Default. UTF-8 text.
- `number` — Integer or float, passed as string in templates.
- `boolean` — `true` or `false`.
- `secret` — String, redacted in traces and log output.

**`from:` binding syntax:**

| Binding | Behavior |
|---------|---------|
| `prompt` | Ask the operator interactively at runtime |
| `env.<NAME>` | Read from environment variable `NAME` |
| `file.<path>` | Read first line from a local file |
| `<provider>.<field>` | Delegate to a registered input provider (e.g. `incident.id`) |
| `var.<name>` | Copy from another variable or capture |

If `from` is omitted, the field MUST have a `default` value or the runtime will reject the runbook at semantic validation time.

### Output Declarations

```yaml
outputs:
  <name>:
    type: string | number | boolean | secret
    description: "..."
    from: captures.<captureName>    # must reference a capture variable
```

The `from: captures.*` binding is the only supported form in v2. The named capture must be written by at least one step in the runbook.

### Expression Language

v2 retains **Go templates** (`text/template`) as the expression language for all interpolated fields (`title`, `args`, `when`, `condition`, `instructions`, etc.).

The template data context is the run variable map: all `vars`, resolved `inputs`, and accumulated `captures` merged into a flat `map[string]string`. Access by dot notation: `{{ .hostname }}`.

**Built-in template functions (v2 additions):**

| Function | Signature |
|----------|-----------|
| `contains` | `(s, substr string) bool` |
| `hasPrefix` | `(s, prefix string) bool` |
| `hasSuffix` | `(s, suffix string) bool` |
| `trim` | `(s string) string` |
| `toLower` | `(s string) string` |
| `toUpper` | `(s string) string` |
| `split` | `(s, sep string) []string` |
| `join` | `(elems []string, sep string) string` |
| `env` | `(name string) string` |
| `now` | `() string` (RFC3339) |
| `runID` | `() string` |
| `not` | `(v bool) bool` |
| `eq, ne, lt, gt, le, ge` | Standard comparison |
| `fromYAML` | `(s string) interface{}` |
| `fromJSON` | `(s string) interface{}` |

All `text/template` built-ins (`and`, `or`, `index`, `len`, `printf`, etc.) remain available.

**`fromYAML` and `fromJSON`:** Parse a raw string variable into a structured value. The raw string is *always* stored; these functions parse on each template evaluation. A parse error is a hard step failure.

```text
{{/* Access a key from a JSON-typed text field */}}
{{ $cfg := fromJSON .service_config }}
{{ $cfg.region }}

{{/* Access a nested key from a YAML-typed text field */}}
{{ $spec := fromYAML .deploy_spec }}
{{ $spec.replicas }}
```

**Template errors:** A template evaluation error in a `condition` or `when` field is a hard step failure (not a soft skip). The step transitions to `failed` and execution halts unless `continue_on_fail: true` is set.

## Step Types (v2 Inventory)

v2 defines **14 step types:**

| Type | Category | Purpose |
|------|----------|---------|
| `cli` | Execution | Execute a local command |
| `tool` | Execution | Invoke a declared tool action |
| `include` | Execution/Flow | Expand another runbook inline |
| `choice` | User Input | Operator selects from predefined options |
| `decision` | User Input | Operator selects an execution path |
| `collector` | User Input | Operator provides structured input across 9 field types |
| `branch` | Flow Control | Conditional fan-out to labelled step sequences |
| `iterate` | Flow Control | Loop over a collection |
| `parallel` | Flow Control | Fan-out concurrent branches with join semantics |
| `approve` | Governance | Standalone approval gate with quorum and business-day timeout |
| `assert` | Governance | Evaluate a guard expression and halt on failure |
| `compensate` | Governance | Declare a saga compensation action |
| `wait_for_event` | Synchronisation | Pause until an inbound event arrives |
| `end` | Terminal | Declare a terminal outcome |

### Common Step Fields

Available on every step type:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique within the runbook |
| `type` | enum | Yes | Step type |
| `title` | string | No | Template; displayed in TUI/trace |
| `subtitle` | string | No | Secondary template annotation |
| `when` | string | No | Guard expression; skips step if false |
| `timeout` | string | No | Duration (e.g. `30s`); overrides default |
| `delay` | string | No | Pre-execution pause (e.g. `5s`) |
| `retry` | object | No | Retry config |
| `continue_on_fail` | bool | No | Continue execution on failure |
| `scope` | string | No | Named scope for variable isolation |
| `export` | string array | No | Variables to export from scope |
| `capture` | map | No | Output capture bindings |
| `contract` | object | No | Behavioral contract for governance |

**Retry configuration:**

```yaml
retry:
  max: 3              # maximum retry attempts (required)
  interval: "5s"      # initial wait between attempts
  backoff: linear     # linear | exponential (default: linear)
```

**Step contract:**

```yaml
contract:
  effects:
    - network.probe
    - service.restart
  reads:   [hostname, port]
  writes:  [http_result]
  deterministic: true
  idempotent: false
  secrets: [auth_token]
```

Effects are free-form strings matched against `governance.rules[].effects`. The `secrets` list names variables whose values must be redacted in traces.

### Step Type: cli

Executes a local command. Two mutually exclusive input modes:
- **Exec form** (`command` + `args`): invokes the binary directly without a shell.
- **Run form** (`run`): shell-script string or a map of per-shell scripts.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `command` | string | See note | Executable name or full path. Mutually exclusive with `run`. |
| `args` | string array | No | Arguments (templates expanded). Only valid with `command`. |
| `run` | string \| map | See note | Shell-string or per-shell map. Mutually exclusive with `command`. |
| `shell` | string | No | Shell to use with `run` string form. Enum: `auto \| bash \| sh \| pwsh \| cmd`. Default: `auto`. Ignored when `run` is a map. |
| `env` | map | No | Additional environment variables |
| `workdir` | string | No | Working directory |
| `stdin` | string | No | Template string piped to stdin |
| `capture` | map | No | `stdout \| stderr \| exit_code` → var |

*Note:* Exactly one of `command` or `run` must be present.

**Run form: map — fallback table:**

| Requested shell | Falls back to |
|-----------------|--------------|
| `bash` | `sh` |
| `sh` | (none — fail) |
| `pwsh` | `cmd` (Windows only) |
| `cmd` | (none — fail) |

```text
# Exec form
- step:
    id: tail_logs
    type: cli
    title: "Tail last 100 lines from {{ .log_file }}"
    command: tail
    args: ["-n", "100", "{{ .log_file }}"]
    capture:
      output: stdout
      exit: exit_code
    contract:
      effects: [filesystem.read]
      reads: [log_file]
      writes: [output]
```

```text
# Run string form with explicit shell
- step:
    id: run_script
    type: cli
    shell: bash
    run: |
      if [ "{{ .env }}" = "prod" ]; then
        echo "Production deploy"
      fi
    capture:
      stdout: script_output
```

```text
# Run map form (multi-shell)
- step:
    id: clear_cache
    type: cli
    run:
      bash: rm -rf /tmp/cache/{{ .service }}
      cmd: rd /s /q C:\tmp\cache\{{ .service }}
      pwsh: Remove-Item -Recurse -Force C:\tmp\cache\{{ .service }}
    capture:
      exit_code: clear_result
```

### Step Type: choice

Prompts the user to select from a set of predefined options and stores the result in a variable.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `prompt` | string | Yes | Question or prompt displayed to user |
| `options` | array | Yes | List of option objects |
| `variable` | string | Yes | Variable name to store selected value |
| `default` | string | No | Default option value if user skips |
| `multiple` | bool | No | Allow selecting more than one option (default: false) |
| `min_selections` | integer | No | Minimum number of selections when `multiple: true` |
| `max_selections` | integer | No | Maximum number of selections when `multiple: true` |

**Option structure:** `label` (string, required), `value` (string, required), `hint` (string, optional).

**Constraints:**
- At least 2 options must be defined.
- All `option.value` entries must be unique within a step.
- If `default` is provided, it must match one of the option values.
- When `multiple: true`, stored variable becomes an array of selected values.
- `min_selections` must be ≥ 1; `max_selections` must be ≤ total number of options and ≥ `min_selections`.

```yaml
- step:
    id: select_environment
    type: choice
    title: Select deployment target
    prompt: Which environment are you deploying to?
    options:
      - label: Staging
        value: staging
        hint: Non-production environment for validation
      - label: Production
        value: production
        hint: Live customer-facing environment
      - label: Canary (10%)
        value: canary
        hint: Gradual rollout to 10% of production traffic
    variable: deploy_env
    default: staging
```

### Step Type: decision

Presents the user with 2 or more execution paths and routes flow to the selected one. A control-flow primitive: the user picks a branch, and execution forks to a different runbook or labeled section. Also acceptable name: `router` (alias).

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `prompt` | string | Yes | Question or scenario description |
| `routes` | array | Yes | List of route objects (minimum 2) |
| `variable` | string | No | Variable to store selected route label (audit) |

**Route structure:** `label` (required), `runbook` (mutually exclusive with `goto`), `goto` (mutually exclusive with `runbook`), `hint` (optional).

**Constraints:**
- At least 2 routes must be defined.
- Each route must specify exactly one of `runbook` or `goto`, not both.
- All `route.label` entries must be unique within a step.

```yaml
- step:
    id: triage_decision
    type: decision
    title: Choose incident response path
    prompt: |
      Based on the error rate and customer impact, which response
      path should we take?
    routes:
      - label: Standard mitigation
        runbook: incident-standard-mitigation
        hint: Error rate < 5%, no customer escalations
      - label: Emergency rollback
        runbook: incident-emergency-rollback
        hint: Error rate > 10% or critical customer impact
      - label: Escalate to on-call
        goto: escalate_to_oncall
        hint: Unclear root cause; needs senior diagnosis
    variable: chosen_path
```

### Step Type: collector

Prompts the user to provide structured input across a rich set of field types. All fields presented and submitted atomically.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `prompt` | string | Yes | Markdown instructions displayed to user |
| `fields` | array | Yes | List of field objects to collect (minimum 1) |
| `approvals` | object | No | Approval gate config |

**Field structure:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Variable name to store collected value |
| `type` | enum | Yes | Field type |
| `label` | string | Yes | Prompt or label text shown to user |
| `required` | bool | No | If true, user must provide a value (default: false) |
| `when` | template | No | Go template — field shown only when truthy; binding skipped when falsy |
| `hint` | string | No | Additional context or validation help |
| `default` | any | No | Pre-filled default value (must match field type) |
| `validation` | object | No | Type-specific validation constraints |
| `options` | array | No | Option list for `select` type |
| `options_from` | object | No | Dynamic option list provider |
| `multiple` | bool | No | Allow multiple selections for `select` and `autocomplete` (default: false) |
| `multiline` | bool | No | Enable multi-line textarea for `text` type (default: false) |
| `ephemeral` | bool | No | Value never written to audit log or persisted; in-memory only (default: false) |

**Field type inventory (10 types):**

| Type | Description | Notes |
|------|-------------|-------|
| `text` | Single-line or multi-line text | `multiline: true` for textarea; `format` for structured validation |
| `number` | Integer or float | Supports `validation.min`, `max`, `step` |
| `integer` | Integer-only | Shorthand for `number` with `step: 1` |
| `date` | ISO 8601 date (YYYY-MM-DD) | UI hint: date picker |
| `datetime` | RFC 3339 datetime with timezone | UI hint: datetime picker |
| `boolean` | True/false checkbox | Stored as boolean in variable space |
| `select` | Dropdown or option list | `multiple: true` for multi-select |
| `autocomplete` | Live search text field | Options fetched from provider as user types; degrades to `select` in CLI |
| `file` | File attachment | Stored as artifact with SHA256 hash |
| `image` | Image/screenshot attachment | Stored as artifact |

**Per-type validation and storage rules:**

| Type | Validation | Variable Storage |
|------|-----------|-----------------|
| `text` | Required; optional `min_length`, `max_length`, `pattern` (RE2); `format: yaml\|json` for multiline | JSON string (raw) |
| `number` | Parse as float64; `min`, `max` (inclusive); `step` | JSON number (float64) |
| `integer` | Parse as int64; reject fractional | JSON number (int64) |
| `date` | Must match `YYYY-MM-DD`; `min`/`max` as ISO date strings | JSON string (`YYYY-MM-DD`) |
| `datetime` | Must be valid RFC 3339; `min`/`max` normalised to UTC | JSON string (RFC 3339) |
| `boolean` | Truthy: `true`, `"true"`, `1`, `"yes"`; Falsy: `false`, `"false"`, `0`, `"no"` | JSON boolean |
| `select` (single) | Value must match one of `options[*].value` | JSON string |
| `select` (multiple) | All values must be members; `min_selections`, `max_selections` | JSON array of strings |
| `file`, `image` | File size and MIME type constraints; SHA-256 computed after storage | Artifact reference |
| `url` | Must parse as valid URL (scheme required) | JSON string |

**Ephemeral fields:**
- Value IS bound into variable space for downstream templates.
- Audit record stores `"[REDACTED]"` — never the actual value.
- NEVER written to trace, checkpoints, or evidence store.
- Dropped on run suspension; operator must re-supply on resume.
- Valid on any field type; warning (not error) on `file`/`image`.

**Dynamic forms (`when` expressions):**
- Fields evaluated left-to-right in array order.
- A field's `when` MAY reference only variables declared **earlier** in the same `fields` array, or global run variables. Forward references are a schema validation error.
- When `when` = false: field is hidden, not prompted, variable binding is skipped entirely (not set to null/empty).
- A `required: true` field with `when` is only required when `when` is truthy.

**`text` field pattern validation (RE2):**
- `validation.pattern` MUST be valid RE2 (not full ECMA 262).
- ECMA 262 features absent from RE2 (lookahead, lookbehind, backreferences) are **rejected at schema load time**.
- Order: required check → min_length → max_length → pattern → format. 3-attempt re-prompt limit is a combined total across all validation steps.

**Validation rules:**
- `fields` MUST have at least one entry (`minItems: 1`).
- Each field `name` must be a valid identifier; names must be unique within a step.
- For `select`: exactly one of `options` or `options_from` must be present.
- `autocomplete`: `options_from` MUST be present with both `provider` and `field`; static `options` is a validation error.
- `validation.format` only meaningful when `multiline: true`; declared on non-multiline → warning, ignored at runtime.
- `validation.format: toml` is a validation warning in v2.0 (deferred).
- `when` forward references → schema validation error at parse time.

**Error variable schema on exhausted re-prompts:**

```json
{
  "step_id":  "gather-approval-details",
  "kind":     "collector.validation_failed",
  "attempts": 3,
  "fields": {
    "amount":     "value 49000 is below minimum 50000",
    "start_date": "2025-12-01 is before minimum 2026-01-01",
    "department": "\"legal\" is not a valid option"
  }
}
```

**Approval gate (on collector or choice steps):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `min` | integer | Yes | Minimum approvals required |
| `roles` | array | Yes | List of role names authorized to approve |
| `timeout` | string | No | Duration before timeout (e.g. `4h`) |
| `on_timeout` | enum | No | `escalate`, `fail`, `skip` |
| `escalate_to` | array | No | Roles to escalate to if timeout occurs |

### Step Type: approve

Dedicated standalone approval gate. Suspends execution until authorized roles approve or timeout fires. No data-collection fields.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | string | No | Human-readable approval prompt |
| `approvals` | object | Yes | Approval gate config |
| `timeout` | string | No | Wall-clock duration (e.g. `48h`); mutually exclusive with `timeout_business_days` |
| `timeout_business_days` | integer | No | Timeout in business days; requires `timezone` |
| `timezone` | string | No | IANA timezone (e.g. `America/New_York`); required with `timeout_business_days` |
| `business_calendar` | string | No | Named calendar ID (e.g. `us-federal`, `uk-banking`); defaults to Mon–Fri no holidays |
| `on_timeout` | string | No | `fail`, `continue`, or `branch:<id>` |
| `contract` | object | No | Behavioral contract for governance |

**`approvals` block:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `mode` | string | No | `all` (default) — all roles must approve; `any` — one suffices; `quorum` — M-of-N from `pool` |
| `roles` | string[] | No | Authorized role names; used with `mode: all` or `mode: any`; mutually exclusive with `pool` |
| `pool` | string[] | No | Eligible role names for quorum; mutually exclusive with `roles` |
| `required` | integer | No | Approvals required from `pool`; used with `mode: quorum`; must be ≥ 1 and ≤ `len(pool)` |

**Validation rules (all MUST be enforced before execution):**
1. `timeout` and `timeout_business_days` are mutually exclusive; both present is an error.
2. `timeout_business_days` requires `timezone`; omitting `timezone` is an error.
3. `approvals.mode: quorum` requires both `approvals.pool` and `approvals.required`.
4. `approvals.required` must satisfy `1 ≤ required ≤ len(pool)`.
5. `approvals.roles` and `approvals.pool` are mutually exclusive.
6. `approvals.mode: all` or `mode: any` requires `approvals.roles` (not `pool`).

```yaml
- step:
    id: legal_review
    type: approve
    title: "Legal team approval"
    approvals:
      roles: [legal-counsel]
    timeout_business_days: 5
    timezone: "America/New_York"
    business_calendar: us-federal
    on_timeout: branch:escalate_legal
```

```yaml
- step:
    id: security_quorum
    type: approve
    title: "Security committee approval (2 of 5)"
    approvals:
      mode: quorum
      pool: [security-lead, ciso, security-eng-1,
             security-eng-2, security-eng-3]
      required: 2
    timeout_business_days: 3
    timezone: "Europe/London"
    on_timeout: fail
```

### Step Type: tool

Invokes a named tool action. The tool must be declared in `toolRefs`.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `tool.name` | string | Yes | Tool name (must match toolRefs entry) |
| `tool.action` | string | Yes | Action declared in tool definition |
| `tool.args` | map | No | Argument key-value map (templates expanded) |
| `tool.version` | string | No | Optional version pin |
| `capture` | map | No | `stdout \| stderr \| json.<path>` → var |

Output capture supports `json.<path>` form for JSON-structured outputs:

```yaml
capture:
  pod_name: json.items[0].metadata.name
  status:   json.items[0].status.phase
```

```text
- step:
    id: list_pods
    type: tool
    title: "List pods in {{ .namespace }}"
    tool:
      name: kubectl
      action: get-pods
      args:
        namespace: "{{ .namespace }}"
        output: json
    capture:
      pod_count: json.items | length
      first_pod: json.items[0].metadata.name
    contract:
      effects: [k8s.read]
      reads: [namespace]
      writes: [pod_count, first_pod]
```

### Step Type: wait_for_event

Pauses runbook execution until an inbound event arrives. Valid **only in `gert serve` mode**.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `event.source` | string | Yes | `webhook`, `message`, `signal`, or `channel` |
| `event.id` | string | No | Logical event name or channel; templates expanded |
| `event.filter` | map | No | Key-value pairs that must match in the incoming event payload |
| `event.payload_schema` | string | No | JSON Schema URL to validate the incoming payload |
| `capture` | map | No | `json.<path>` → variable |
| `timeout` | string | No | Wall-clock duration; default: no timeout |
| `on_timeout` | string | No | `fail` (default), `continue`, or `branch:<step-id>` |

Event source semantics:
- `webhook` — runtime opens HTTP POST listener; sender POSTs a JSON body.
- `message` — runtime subscribes to a message-queue topic or queue name given by `event.id`.
- `signal` — runtime registers a named OS or process signal handler.

  **Allowed signals by platform:**
  - Linux: `SIGINT`, `SIGTERM`, `SIGUSR1`, `SIGUSR2`, `SIGHUP`
  - macOS: `SIGINT`, `SIGTERM`, `SIGUSR1`, `SIGUSR2`, `SIGHUP`
  - Windows: `SIGINT` only (CTRL_C_EVENT)

  Semantic validation MUST reject unsupported signals for the host OS at parse time.

- `channel` — runtime uses a gert-internal named channel.

Runtime endpoint available as `{{ .gert.event.<id>.endpoint }}`. Webhook token available as `{{ .gert.event.<id>.token }}` (single-use, expires on resume or timeout).

```yaml
- step:
    id: wait_prometheus_alert
    type: wait_for_event
    title: "Wait for Prometheus alert to fire"
    event:
      source: webhook
      id: prometheus.alert.firing
      filter:
        alertname: "HighErrorRate"
        severity: critical
    capture:
      alert_value: json.commonAnnotations.value
      alert_labels: json.commonLabels
    timeout: 30m
    on_timeout: fail
```

### Step Type: include

Expands another runbook's steps inline into the current run path. **Not** a sub-procedure call — shares the caller's variable space entirely.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `include.runbook` | string | Yes | Path or ID of the runbook to include. Relative paths resolve from the including runbook's location. |
| `include.with` | map | No | Variable overrides injected into shared scope before expansion. |
| `include.when` | string | No | Boolean expression. If false, include is skipped entirely. |

```text
- step:
    id: run_db_checks
    type: include
    title: "Run database pre-flight checks"
    include:
      runbook: ./shared/db-preflight.runbook.yaml
      with:
        db_host: "{{ .target_db }}"
        check_mode: read-only
    contract:
      effects: [db.read]
      reads: [target_db]
      writes: []
```

Cycle detection: hard validation error:
```
cyclic include detected: A -> B -> A
```

### Step Type: branch

Evaluates a list of conditions and executes the first matching arm.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `branches` | array | Yes | List of Branch objects |
| `branches[].condition` | string | Yes | Go template evaluating to `true`/`false` |
| `branches[].label` | string | No | Display label for diagram/trace |
| `branches[].steps` | array | No | Steps to execute if condition matches |
| `branches[].else` | bool | No | `true` marks a catch-all arm (new in v2) |

At most one branch arm executes. If no arm matches and no `else: true` arm exists, execution continues past the branch node with no steps run.

```text
- step:
    id: route_on_env
    type: branch
    title: Route by environment
    branches:
      - condition: '{{ eq .environment "production" }}'
        label: Production path
        steps:
          - step:
              id: require_approval
              type: manual
              title: Require change-manager approval
              approvals:
                min: 1
                roles: [change-manager]
      - condition: '{{ eq .environment "staging" }}'
        label: Staging path
        steps: [...]
      - else: true
        label: Unknown environment — abort
        steps:
          - step:
              id: abort
              type: end
              title: Unknown environment
              outcome:
                category: escalated
                code: unknown_env
```

### Step Type: iterate

Repeats a block of steps over a collection or until a convergence condition is met. `iterate` is a TreeNode sibling (not a `step.type` enum value).

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `iterate.over` | string | * | Template resolving to comma-separated list |
| `iterate.as` | string | No | Loop variable name (default: `item`) |
| `iterate.max` | int | * | Max iterations (convergence mode) |
| `iterate.until` | string | * | Convergence condition expression |
| `iterate.collect` | map | No | Accumulate captures across passes |
| `iterate.steps` | array | Yes | Steps to execute each pass |

\* `over` OR (`max` + `until`) required; mutually exclusive.

Loop variable `{{ .iteration }}` (0-indexed) is always available. In list mode, `{{ .item }}` (or the `as` name) holds the current element.

```text
# List iteration
- iterate:
    id: check_all_services
    over: "{{ .service_list }}"   # e.g. "api,worker,scheduler"
    as: svc
    collect:
      results: "{{ .svc }}:{{ .http_result }}"
    steps:
      - step:
          id: probe
          type: tool
          ...

# Convergence iteration
- iterate:
    id: wait_for_deploy
    max: 10
    until: '{{ eq .deploy_status "Running" }}'
    steps: [...]
```

### Step Type: parallel

Executes a set of independent step groups concurrently, then joins. New in v2.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `branches` | array | Yes | Each branch is an independent group |
| `join.wait_for` | enum | No | `all \| any \| majority` (default: `all`) |
| `join.on_failure` | enum | No | `fail \| continue` (default: `fail`) |

**Constraint:** Nested `parallel` steps are forbidden in v2.0. A `parallel` step MUST NOT appear as a step inside a parallel branch. Semantic validation MUST reject this with error `parallel/nested-forbidden`.

Parallel branches are isolated from each other's variable writes during execution. After join, all `capture` values from all branches are merged into the caller's variable map. Write conflicts (two branches writing the same variable) → last-writer-wins, warning emitted.

```text
- step:
    id: parallel_checks
    type: parallel
    title: Run DNS and HTTP checks concurrently
    branches:
      - label: DNS check
        steps:
          - step:
              id: dns_check
              type: tool
              tool: { name: nslookup, action: lookup, args: { host: "{{ .hostname }}" } }
              capture: { dns_result: stdout }
      - label: HTTP check
        steps:
          - step:
              id: http_check
              type: tool
              tool: { name: curl, action: head, args: { url: "https://{{ .hostname }}/healthz" } }
              capture: { http_result: stdout }
    join:
      wait_for: all
      on_failure: fail
```

### Step Type: assert

Evaluates a list of assertions against captured variables. Fails the run if any assertion does not pass.

Assertion types: `contains`, `not_contains`, `matches` (regex), `eq`, `ne`, `lt`, `gt`, `exit_code`, `json_path`.

```text
- step:
    id: verify_deployment
    type: assert
    title: Verify deployment succeeded
    assert:
      - type: contains
        subject: "{{ .deploy_output }}"
        expected: "successfully rolled out"
      - type: eq
        subject: "{{ .exit_code }}"
        expected: "0"
      - type: json_path
        subject: "{{ .api_response }}"
        path: "$.status"
        expected: "active"
```

### Step Type: compensate

Registers a compensation (rollback) action executed in reverse order if the run fails or is cancelled after this step. Implements the saga pattern. New in v2.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `compensate.on` | enum | No | `failure \| cancel \| any` (default: `failure`) |
| `compensate.steps` | array | Yes | Steps to execute as compensation |

Compensation steps execute in LIFO order relative to the sequence of `compensate` nodes encountered during forward execution.

```text
- step:
    id: scale_down
    type: tool
    title: Scale down service
    tool: { name: kubectl, action: scale, args: { deployment: "{{ .deployment }}", replicas: "0" } }

- step:
    id: register_scale_up
    type: compensate
    title: Compensation — scale back up on failure
    compensate:
      on: failure
      steps:
        - step:
            id: scale_up_comp
            type: tool
            title: Restore service scale
            tool:
              name: kubectl
              action: scale
              args:
                deployment: "{{ .deployment }}"
                replicas: "{{ .original_replicas }}"
```

### Step Type: end

Declares a terminal outcome and halts execution. Renamed from `router` in v1.

```text
- step:
    id: resolved
    type: end
    title: Incident resolved
    outcome:
      category: resolved       # resolved | escalated | no_action | needs_rca
      code: dns_healthy
      meta:
        resolution_time: "{{ .elapsed }}"
```

## Tool Definition Schema (v2)

```text
$schema: "https://schemas.gert.dev/tool/v2.json"
apiVersion: tool/v2

meta:
  name: kubectl
  version: "1.2"
  description: Kubernetes CLI wrapper
  binary: kubectl

transport:
  mode: stdio         # stdio | jsonrpc | mcp
  # jsonrpc options:
  # start: ["node", "server.js"]
  # port: 3000
  # mcp options:
  # start: ["python", "mcp_server.py"]

actions:
  get-pods:
    description: List pods in a namespace
    argv: ["get", "pods", "-n", "{{ .namespace }}", "-o", "json"]
    args:
      namespace:
        type: string
        required: true
        description: Kubernetes namespace
      output:
        type: string
        required: false
        default: json
    capture:
      stdout:
        format: json    # json | text | lines
    contract:
      effects: [k8s.read]
      reads: [namespace]
    governance:
      action: allow
```

**New in v2:**
- `capture.stdout.format` — declares output format; enables structured `json.<path>` capture in runbook steps.
- `actions.<name>.contract` — per-action behavioral contract automatically merged with step-level contract override.
- `actions.<name>.governance` — per-action governance policy; overrides runbook-level rules for this action only.
- `meta.version` — explicit tool version for auditing and pinning.

## Provider Definition Schema (v2)

```yaml
$schema: "https://schemas.gert.dev/provider/v2.json"
apiVersion: provider/v2

meta:
  name: incident
  version: "1.0"
  description: PagerDuty incident context provider
  kind: input-provider    # input-provider | enrichment-provider

transport:
  mode: jsonrpc
  start: ["./providers/pagerduty-provider"]

fields:
  id:
    type: string
    description: Active incident ID
  title:
    type: string
    description: Incident title
  severity:
    type: string
    enum: [P1, P2, P3, P4]
  affected_service:
    type: string
    description: Primary impacted service name

capabilities:
  - input-resolution
```

The `fields` map defines the provider's schema. Each field name corresponds to a `from: <provider>.<field>` binding in a runbook input. The semantic validator checks that all `from: <provider>.*` bindings reference a field declared in a loaded provider definition.

## Namespace Convention for Extensions

### The `x-<namespace>:` Convention

Extension fields MUST:
1. Start with `x-`.
2. Be followed by a namespace component (lowercase alphanumeric and hyphens, matching `[a-z][a-z0-9-]+`).
3. Be declared in an extension manifest registered via `gert extension register` or in `gert.yaml`.

```yaml
# Runbook level
x-myorg-cost-center: "engineering-ops"
x-myorg-change-id: "CHG-20260418-001"
x-otel-service-name: "payment-service"
```

```text
# Step level
- step:
    id: deploy
    type: tool
    title: Deploy to production
    x-myorg-risk-level: high
    x-myorg-rollback-id: "{{ .deploy_id }}"
```

### Extension Validation

1. **Unregistered extension fields** — Any field matching `^x-[a-z][a-z0-9-]+$` is accepted without a registered schema. `gert validate` emits a `WARNING` for each unregistered extension field.
2. **Registered extension fields** — May ship a JSON Schema fragment. Once registered, violations are errors, not warnings.
3. **Unknown non-extension fields** — Any field that does not match a core field and does not start with `x-` is a hard validation error (`additionalProperties: false`).

### Extension Registration Format

```yaml
# .gert/extensions/myorg.yaml
apiVersion: extension/v1
namespace: myorg
description: MyOrg internal runbook annotations

fields:
  cost-center:
    type: string
    description: Cost center code for billing
    pattern: "^[a-z]+-[a-z]+"
  risk-level:
    type: string
    enum: [low, medium, high, critical]
  change-id:
    type: string
    pattern: "^CHG-[0-9]{8}-[0-9]{3}$"
  rollback-id:
    type: string

applies-to: [runbook, step]
```

## Structural vs. Semantic Validation

v2 validation runs in two sequential phases. Both must pass for a runbook to be considered valid.

### Phase 1: Structural Validation

Pure JSON Schema Draft 2020-12 evaluation. Checks:
- Required fields present (`id`, `name`, `apiVersion`, `flow`).
- Field types correct.
- Enum values valid.
- Pattern constraints satisfied (e.g. `id` must match slug pattern; `timeout` must match `[0-9]+(s|m|h)`).
- `minItems`/`minLength` constraints met.
- `additionalProperties` — no unknown core fields present.

**Implementation note:** Go runtime uses `invopop/jsonschema` to generate the JSON Schema from struct tags at startup, then validates against it. `yaml.v3 KnownFields(true)` provides a pre-JSON-Schema pass rejecting unknown fields.

### Phase 2: Semantic Validation

Enforces domain rules that cannot be expressed in JSON Schema. Runs after structural validation. Requires full runbook AST and loaded tool/provider definitions.

Semantic checks:
1. **Step ID uniqueness** — All step IDs in the flat tree walk must be unique.
2. **Include target resolution** — Every `include.runbook` must resolve to an existing file.
3. **Branch condition variable references** — Each `condition` template references only in-scope variables.
4. **Tool reference resolution** — Every `tool.name` must match a `toolRefs` entry; the `.tool.yaml` file must be loadable and structurally valid.
5. **Tool action existence** — `tool.action` must be declared in the referenced tool definition.
6. **Provider binding resolution** — Every `inputs.*.from: <provider>.<field>` must resolve to a loaded provider with that field declared.
7. **Cyclic include detection** — Include graph must be a DAG.
8. **Capture write-before-read** — Warning if write is only on some branches.
9. **Governance policy pre-check** — Static governance rules evaluated against step contracts. Steps with `deny` effects are flagged as errors; `require-approval` effects as warnings.
10. **Output declaration completeness** — Every declared `output` must reference a capture written by at least one step.
11. **Compensate ordering** — Each `compensate` step must appear after the step it compensates in tree walk order.

**Error reporting format:**

```
$ gert validate service-health.runbook.yaml
[SEM-003] step "check_http": tool action "head" not declared in
          tools/curl.tool.yaml (declared actions: get, post, download)
          --> service-health.runbook.yaml:34:7
[SEM-007] step "eval": template references variable "dns_result" which
          is only written on branch "DNS check" — may be unbound
          --> service-health.runbook.yaml:51:12
          hint: add a default value in vars: or guard with {{ if .dns_result }}
```

Each error includes: error code (e.g. `SCH-001`, `SEM-007`), file path and YAML line/column, human-readable message, suggested fix (where computable).

## Schema Artifacts

**Shipped artifacts:**

| Artifact | Description |
|----------|-------------|
| `runbook.v2.schema.json` | JSON Schema Draft 2020-12 for runbook documents |
| `tool.v2.schema.json` | JSON Schema for tool definitions |
| `provider.v2.schema.json` | JSON Schema for provider definitions |

Generated at runtime from Go struct tags via `invopop/jsonschema` and served by `gert schema export`. Also embedded in the binary and served at `https://schemas.gert.dev/*`.

```bash
# Export runbook schema (default)
gert schema export

# Export specific schema
gert schema export --kind runbook   # runbook | tool | provider | bundle

# Export full bundle (all three)
gert schema export --kind bundle > gert-v2-schemas.json

# Validate a document against the exported schema
gert schema export | check-jsonschema --schemafile - runbook.yaml
```

**VS Code integration** — `yaml-language-server` configured via:

```text
# .vscode/settings.json (auto-generated by gert vscode init)
{
  "yaml.schemas": {
    "https://schemas.gert.dev/runbook/v2.json": "**/*.runbook.yaml",
    "https://schemas.gert.dev/tool/v2.json":    "**/*.tool.yaml",
    "https://schemas.gert.dev/provider/v2.json": "**/*.provider.yaml"
  }
}
```

VS Code extension additional capabilities:
- Step-type-aware field completion via `schema/stepFields` JSON-RPC method.
- Tool argument completion via `schema/toolArgs`.
- Inline governance warnings derived from `exec/dryRun` pre-check.

**Schema version manifest:**

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://schemas.gert.dev/runbook/v2.json",
  "x-gert-version": "2.0.0",
  "x-gert-released": "2026-04-18",
  "title": "gert runbook v2",
  ...
}
```
