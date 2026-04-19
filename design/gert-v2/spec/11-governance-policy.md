# Governance and Policy

Governance is gert's core differentiator. This section defines the v2 policy primitives (allowlists, denylists, env-var blocking, output redaction, approval gates), the evaluation order and checkpoint model, the RBAC model, redaction guarantees, audit trail requirements, and the extension-contributed policy contract. Every primitive is host-enforced; none may be delegated to extensions or deferred.

---

## Governance Primitives

### Command Allowlist (`allowed_commands`)

Whitelist of executable base names (argv[0], path-stripped) that `cli` steps and tool subprocess invocations may run. If non-empty, any command not in the list is blocked before fork. Empty list (`[]`) = permissive (all commands allowed). Glob patterns are **not** supported in v2.0 (exact match only).

```yaml
meta:
  governance:
    allowed_commands: [kubectl, curl, jq, aws]
```

**Evaluation point:** Runtime Core, immediately before subprocess fork (after template evaluation).  
**Trace event:** `governance/commandBlocked` — fields: `stepId`, `command`, `reason: "not_in_allowlist"`.

---

### Command Denylist (`denied_commands`)

Explicit list of executable names unconditionally blocked. **Denylist takes precedence over allowlist**: a command in both is denied.

```yaml
meta:
  governance:
    denied_commands: [rm, dd, shutdown, reboot]
```

**Evaluation point:** Runtime Core, same checkpoint as allowlist; denylist check runs first.  
**Trace event:** `governance/commandBlocked` — fields: `stepId`, `command`, `reason: "in_denylist"`.

---

### Environment Variable Blocking (`deny_env_vars`)

Glob patterns matched against env var names. Matching vars are stripped from the subprocess environment before execution. Applied to the full process environment (not just explicitly passed vars).

```yaml
meta:
  governance:
    deny_env_vars:
      - "*_SECRET*"
      - "*_TOKEN"
      - "AWS_*"
      - "GITHUB_*"
```

**Evaluation point:** Runtime Core, immediately before subprocess fork.  
**Trace event:** `governance/envVarBlocked` — fields: `stepId`, `varNames: [string]` (names only, never values).

---

### Output Redaction (`redact`)

List of RE2 regex rules applied to captured stdout/stderr before output is stored in run variables or written to the trace. Rules applied in declaration order. Replace string may use RE2 backreferences (`$1`). Applied **before** any capture variable assignment or trace write.

```yaml
meta:
  governance:
    redact:
      - pattern: "token=[A-Za-z0-9_\\-]{20,}"
        replace: "token=[REDACTED]"
      - pattern: "password: .+"
        replace: "password: [REDACTED]"
      - pattern: "(?i)bearer\\s+[A-Za-z0-9._\\-]+"
        replace: "Bearer [REDACTED]"
```

**Evaluation point:** Runtime Core, immediately after subprocess/tool output is captured.  
**Trace event:** `governance/redactionApplied` — fields: `stepId`, `ruleCount` (count of matched rules, not the patterns themselves).

---

### Approval Gates

First-class step-level primitives. See [Approval Gate Design](#approval-gate-design) below.

---

## Policy Evaluation Model

Evaluation order within each checkpoint is fixed and MUST NOT be reordered by extensions.

| Checkpoint        | Primitives Evaluated             | Phase        |
|-------------------|----------------------------------|--------------|
| Parser validation | Schema structural rules          | Parse        |
| Planner validation| Import cycles, tool existence    | Plan         |
| Run pre-flight    | Extension policy rules           | Run init     |
| Step pre-flight   | Denylist check                   | Runtime Core |
| Step pre-flight   | Allowlist check                  | Runtime Core |
| Step pre-flight   | Approval gate                    | Runtime Core |
| Subprocess fork   | Env var blocking                 | Runtime Core |
| Output captured   | Redaction                        | Runtime Core |
| Post-step         | Contract risk warning            | Runtime Core |

**Order of precedence** within the pre-flight checkpoint for `cli`/`tool` steps:

1. Denylist check (hard block — step fails immediately if matched).
2. Allowlist check (hard block — step fails immediately if non-empty list and no match).
3. Approval gate (soft pause — step held until quorum met).
4. Env var blocking (applied at fork, silent strip, not a failure condition).

A hard-block violation writes the corresponding trace event, returns an error from `RunHandle.Next()`, and places the run in the `policy_violation` terminal state. **Resumption from a policy violation is not supported in v2.0.**

---

## Approval Gate Design

### Schema Declaration

```text
steps:
  - id: delete_database
    kind: cli
    command: "kubectl delete namespace {{ .targetNamespace }}"
    approvals:
      min: 2
      roles: ["DRI", "change-manager"]
      timeout: 24h
      on_timeout: fail   # or: escalate
      escalate_to: ["senior-sre", "director-of-eng"]
      message: "Deleting {{ .targetNamespace }} is irreversible. Confirm change window."
```

### UX Contract

When the Runtime Core reaches a step with an `approvals` block:

1. Writes `governance/approvalRequired` trace event (`stepId`, `message`, `roles`, `min`, `timeout`).
2. Emits an `ApprovalRequired` runtime event on the event channel.
3. Returns from `Next()` with a `StepResult` of kind `PendingApproval`. The step does **not** execute yet.
4. Subsequent calls to `Next()` on the same step return `ErrAwaitingApproval` until the approval quorum is met.

The adapter (VS Code extension, TUI, or `gert serve` client) is responsible for presenting the approval request and collecting decisions.

### Approval Decision Protocol

```go
type ApprovalDecision struct {
    StepID   string    // step to approve (must match current paused step)
    Actor    string    // identity of the approver (e.g., "alice@corp.com")
    Role     string    // role the approver is acting in (must be in step.approvals.roles)
    Approved bool      // true = approved, false = rejected
    Comment  string    // optional justification
    At       time.Time // timestamp of decision
}
```

**Runtime validates:**
- `Actor` identity is established (from `--as` flag or adapter authenticated session).
- `Role` is in the step's declared `approvals.roles`.
- Same actor may not approve the same gate twice.
- If `Approved = false`, the gate is **immediately rejected** (single rejection vetos), run enters `rejected` terminal state.

When approval count reaches `min`, the runtime writes `governance/approvalGranted` and the next `Next()` call proceeds with normal step execution.

### Approval Recording in the Trace

```json
// governance/approvalRequired
{
  "sequence": 14,
  "event": "governance/approvalRequired",
  "stepId": "delete_database",
  "message": "Deleting prod-ns is irreversible. Confirm change window.",
  "roles": ["DRI", "change-manager"],
  "min": 2,
  "timeout": "24h",
  "at": "2026-04-18T10:00:00Z"
}

// governance/approvalDecision (written once per decision)
{
  "sequence": 15,
  "event": "governance/approvalDecision",
  "stepId": "delete_database",
  "actor": "alice@corp.com",
  "role": "DRI",
  "approved": true,
  "comment": "Change window confirmed with customer.",
  "at": "2026-04-18T10:05:00Z"
}

// governance/approvalGranted (written when quorum is met)
{
  "sequence": 16,
  "event": "governance/approvalGranted",
  "stepId": "delete_database",
  "approvals": ["alice@corp.com", "bob@corp.com"],
  "at": "2026-04-18T10:07:00Z"
}
```

### Timeout and Escalation

- **`on_timeout: fail`** — run enters `timed_out` terminal state; `governance/approvalTimedOut` written to trace.
- **`on_timeout: escalate`** — runtime emits `ApprovalEscalated` event; if `escalate_to` is declared, a new `ApprovalRequired` targets those roles and the timeout clock restarts. Escalation is attempted once; if the escalated request also times out, the run fails.

Timeout enforcement uses `context.WithDeadline` in the approval wait loop.

---

## Policy-as-Code

### v2.0: Structured YAML Policy Blocks

v2.0 uses structured YAML policy blocks in `meta.governance`. The vocabulary is small and closed (six primitives). Blocks are self-contained, JSON Schema validated at authoring time.

### v2.1+: OPA Integration Hooks

```go
// OPA evaluation hook interface (v2.1 target)
type PolicyEngine interface {
    // EvaluatePreStep is called before each step pre-flight.
    EvaluatePreStep(ctx context.Context, input PolicyInput) (PolicyDecision, error)

    // EvaluatePreRun is called once after plan construction (runbook-level RBAC).
    EvaluatePreRun(ctx context.Context, input PolicyInput) (PolicyDecision, error)
}

type PolicyInput struct {
    Runbook  RunbookMeta
    Step     ResolvedStep
    Actor    string
    Vars     map[string]string
    Contract *contract.Contract
}

type PolicyDecision struct {
    Allow  bool
    Reason string
    RequiredApprovers []string
}
```

v2.1 `PolicyEngine` implementations:
1. **Built-in** — evaluates structured YAML policy blocks (v2.0 backward compatible).
2. **OPA** — calls an OPA bundle from `.gert/policy/*.rego`.

The two compose: built-in runs first, OPA runs second. A denial from either blocks the step.

---

## RBAC Model

### Identity Establishment

Actor identity is established via the `--as <identity>` flag. **v2.0 limitation: identity is not authenticated.** The `--as` flag is self-asserted; enforcement is the responsibility of surrounding infrastructure.

### Runbook-Level Access Control

```yaml
meta:
  governance:
    requires_role: "incident-responder"
    # or: requires_any_role: ["DRI", "incident-responder"]
    # or: requires_all_roles: ["DRI", "change-manager"]
```

Evaluated during run initialization (pre-flight). Failure writes `governance/accessDenied` and the run fails immediately.

### Step-Level Role Restriction

```text
steps:
  - id: promote_to_production
    kind: cli
    requires_role: "release-manager"
    command: "kubectl set image deployment/app app={{ .imageTag }}"
```

Distinct from approval gates: `requires_role` *prevents* the step from executing unless the actor holds the role; approval gates require *external* approval.

### Separation of Duties

The same-actor-cannot-approve invariant (approval gate design) ensures a step author cannot self-approve their own gate.

---

## Redaction

### Guarantees

- Redaction is applied to all output paths: captured variables, trace events, and event channel payloads.
- Rules are compiled (regex pre-compiled) during run initialization, not per step.
- Applied unconditionally in all run modes (real, dry-run, replay).
- The original (pre-redaction) value is **never** written to disk in any path.

### Explicit Field Marking (v2.1)

```yaml
# In .tool.yaml (v2.1 target)
outputs:
  - name: api_key
    sensitive: true   # value redacted from trace and captures
```

---

## Audit Trail Requirements

Every governance event is a first-class trace entry. Guaranteed governance trace events:

| Trace Event                         | Written When                                          |
|-------------------------------------|-------------------------------------------------------|
| `governance/commandBlocked`         | Command blocked by denylist or allowlist              |
| `governance/envVarBlocked`          | Env vars stripped before subprocess fork              |
| `governance/redactionApplied`       | Redaction rules matched captured output               |
| `governance/approvalRequired`       | A step's approval gate is entered                     |
| `governance/approvalDecision`       | An approver submits approve or reject                 |
| `governance/approvalGranted`        | Approval quorum is met                                |
| `governance/approvalRejected`       | An approver rejects the gate                          |
| `governance/approvalTimedOut`       | Approval timeout expires                              |
| `governance/accessDenied`           | RBAC check fails (run or step level)                  |
| `governance/contractWarning`        | Step contract risk level is critical                  |
| `governance/policyViolation`        | Generic policy violation (OPA engine, v2.1)           |

### Trace Integrity

- **Append-only write discipline:** trace writer never seeks backward or truncates; each write is followed by `fsync`.
- **SHA256 evidence hash:** at run completion, SHA256 of the entire trace file is written as the final record and stored in `trace.sha256`.
- **Monotonic sequence numbers:** gaps in sequence numbers signal corruption or tampering.

### Compliance Alignment

- **SOC 2 Type II** — every command execution and approval logged with actor identity and timestamp.
- **HIPAA** — env var blocking and redaction prevent PHI from appearing in traces; approval gates enforce separation of duties.
- **ITIL Change Management** — `requires_role` and multi-approver gates model normal/standard/emergency change paths.

---

## Extension-Contributed Policy

Extensions contribute governance rules through `ContributedPolicyRules()` on the Extension Host. Invariants:

1. **Additive only.** Extensions may add restrictions; they may never remove or override host-defined policy.
2. **Host policy takes precedence on conflict.** More restrictive interpretation applies.
3. **Extension rules are visible in the trace.** `governance/extensionPolicyApplied` event names the extension and rule ID.
4. **Extension rules are not trusted with approval authority.** Extensions may not grant approvals; decisions must originate from a human actor via `RunHandle.Approve()`.

### Extension Policy Rule Schema

```go
// PolicyRule is contributed by an extension and evaluated at pre-step checkpoint.
type PolicyRule struct {
    ID          string       // unique within the extension's namespace
    Description string
    StepKinds   []StepKind   // step types this rule applies to (empty = all)
    CommandGlob string       // if set, only applies to commands matching this glob
    Action      PolicyAction // Deny or Warn
    Reason      string       // human-readable reason written to trace event
}
```

Extension policy is loaded during the Extension Host's `Load()` phase, before the first `Next()` call. Rules are **static for the lifetime of a run**; extensions may not add or remove rules after the run has started.
