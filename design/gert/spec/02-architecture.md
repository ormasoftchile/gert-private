# Architecture

The v2 system defines five primary components, strict dependency rules, a complete data flow from runbook invocation to trace archive, and a concurrency model. The single most important rule: **dependencies flow inward toward the Core Domain.** The Runtime Core knows nothing about TUI, VS Code, or JSON-RPC. Adapters know nothing about execution semantics. Every component boundary is enforced by a Go interface; no component imports a package "above" it in the dependency graph.

## Dependency Direction

```
  Adapter Layer
       |
       v
  Runtime Core  <---- Extension Host
       |                    |
       v                    v
    Planner           Tool Runtime
       |
       v
     Parser
       |
       v
  Schema / JSON Schema artifact
```

No upward arrow exists. The Parser knows nothing about the Runtime. The Runtime knows nothing about adapters. The Extension Host is a peer of the Runtime Core, contributing tools and policy rules but not owning execution.

## Primary Components

### Parser

**Responsibility:** Transform raw YAML bytes into a validated, semantically-resolved `ParsedRunbook` value. Validates schema structure and surface-level semantic rules (duplicate step IDs, unknown step types, malformed Go template expressions). Defers deep semantic validation (e.g., whether a `from:` binding can actually be resolved) to later pipeline stages.

```go
// Parser transforms raw YAML bytes into a validated ParsedRunbook.
type Parser interface {
    // Parse reads YAML from r, validates against the v2 JSON Schema,
    // applies structural semantic checks, and returns a ParsedRunbook.
    // Errors carry structured diagnostics (file, line, column, message).
    Parse(ctx context.Context, r io.Reader, opts ParseOptions) (*ParsedRunbook, error)
}

type ParseOptions struct {
    // SchemaVersion overrides the apiVersion field for compatibility shims.
    SchemaVersion string
    // StrictMode causes deprecation warnings to be surfaced as errors.
    StrictMode bool
}

// ParsedRunbook is an immutable, fully-validated representation of a runbook.
// All fields have been structurally validated. Templates are not yet evaluated.
type ParsedRunbook struct {
    Meta       RunbookMeta
    Governance GovernancePolicy  // nil if no governance block
    Steps      []ParsedStep
    Imports    map[string]string // alias -> path (not yet resolved)
    Tools      []string          // tool names (not yet discovered)
    Schema     SchemaRef         // version and $schema URL
}
```

**Dependencies:** JSON Schema artifact (static embedded file) and expression evaluator (for template syntax validation). No runtime dependencies.

**Key Invariants:**
- A `ParsedRunbook` returned without error is structurally valid against the v2 JSON Schema.
- All step IDs are unique within a `ParsedRunbook`.
- No I/O occurs during parsing (file imports are recorded as paths, not resolved).
- Parsing is pure and deterministic: the same input always produces the same output.

### Planner

**Responsibility:** Transform a `ParsedRunbook` into an `ExecutionPlan`: a resolved, ordered, dependency-annotated representation of every step the runtime will execute. Resolves imports, discovers and validates tool definitions, resolves input provider bindings, and produces static step ordering. Does **not** evaluate dynamic conditions (branch predicates, iterate count expressions) — those are runtime concerns.

An `ExecutionPlan` is a *flat, ordered list* of resolved steps with pre-computed metadata, not a DAG. Branch logic and iteration are runtime decisions because they depend on captured variables evaluated during execution.

```go
type Planner interface {
    // Plan resolves imports, discovers tools, validates bindings,
    // and returns a fully-resolved ExecutionPlan ready for the runtime.
    Plan(ctx context.Context, rb *ParsedRunbook, opts PlanOptions) (*ExecutionPlan, error)
}

type PlanOptions struct {
    BaseDir     string            // working directory for import resolution
    ToolDirs    []string          // directories to search for .tool.yaml files
    ProviderDir string            // directory for .provider.yaml files
    Vars        map[string]string // CLI-supplied variable overrides
}

// ExecutionPlan is an immutable, fully-resolved plan for a single run.
type ExecutionPlan struct {
    RunID       string
    RunbookPath string
    Steps       []ResolvedStep       // ordered; includes inlined include steps
    Tools       map[string]*ToolDef  // name -> resolved tool definition
    Providers   map[string]*ProviderDef // binding name -> provider
    Governance  *GovernancePolicy
    Metadata    PlanMetadata
}

// ResolvedStep is a single step in the plan, fully resolved.
type ResolvedStep struct {
    ID       string
    Kind     StepKind      // cli, choice, decision, collector, tool, include, branch, iterate
    Spec     StepSpec      // kind-specific parameters (resolved, not templated)
    Contract *contract.Contract  // effects/reads/writes/idempotent/deterministic
    Depth    int           // chain depth (0 = root runbook)
    Origin   string        // source runbook path (for inlined include steps)
}
```

**Dependencies:** Parser (for import resolution), Tool Registry, Provider Registry. MUST NOT depend on Runtime Core, Extension Host, or any adapter.

**Key Invariants:**
- An `ExecutionPlan` returned without error contains only resolvable steps.
- All tool names referenced by steps appear in `ExecutionPlan.Tools`.
- Import graphs are acyclic; the Planner detects and rejects cycles.
- All `from:` bindings that declare a provider appear in `ExecutionPlan.Providers`.
- The Planner does not execute or side-effect; it is safe to plan without running.

### Runtime Core

**Responsibility:** Own the execution loop — advancing through an `ExecutionPlan` step by step, evaluating templates and branch predicates, dispatching tool invocations, collecting evidence, writing the append-only trace, enforcing governance pre-flight checks, emitting structured events, and managing mutable run state. Does **not** own user presentation, serialization formats, or network transport.

```go
type Runtime interface {
    // Start initializes a run from a plan and returns a RunHandle.
    // The run does not advance until Next is called.
    Start(ctx context.Context, plan *ExecutionPlan, opts RunOptions) (RunHandle, error)

    // Resume restores a previously checkpointed run from the run store.
    Resume(ctx context.Context, runID string, opts RunOptions) (RunHandle, error)
}

type RunHandle interface {
    // Next advances the run by one step. Blocks until the step completes
    // (or is paused waiting for evidence/approval).
    // Returns io.EOF when the run has no more steps.
    Next(ctx context.Context) (*StepResult, error)

    // Approve records approval for the current step's approval gate.
    Approve(ctx context.Context, decision ApprovalDecision) error

    // SubmitEvidence records evidence for interactive steps (choice, decision, collector).
    // For choice: stores selected option. For decision: stores route selection.
    // For collector: stores multi-field form data and artifact attachments.
    SubmitEvidence(ctx context.Context, stepID string, ev map[string]*EvidenceValue) error

    // Cancel requests a graceful stop of the run.
    Cancel(ctx context.Context, reason string) error

    // State returns the current mutable run state (read-only snapshot).
    State() RunState

    // Events returns a channel of structured events emitted during execution.
    Events() <-chan Event
}

type RunOptions struct {
    Mode        RunMode           // real, dry-run, replay
    Actor       string            // --as identity
    Vars        map[string]string // variable overrides
    ScenarioDir string            // replay scenario directory
    Store       RunStore          // durable store for checkpointing
    OnEvent     func(Event)       // synchronous event hook (nil = use Events channel)
}
```

**Dependencies:** Governance Engine, Tool Runtime, Expression Evaluator, Evidence store, Trace Writer. Does NOT depend on Extension Host, API/Adapter Layer, or any adapter.

**Key Invariants:**
- The trace is append-only: no event is ever deleted or modified after being written.
- Governance pre-flight is enforced before every `cli` and `tool` step dispatch.
- Redaction is applied to all captured output before it is written to the trace.
- The run state is checkpointed after every step completion.
- Events are emitted in a total order that matches step execution order.
- The runtime is re-entrant per run: multiple callers may call `Next` sequentially (but not concurrently) on the same `RunHandle`.

### Extension Host

**Responsibility:** Manage the lifecycle of out-of-process extensions (discovery, launch, capability negotiation, RPC dispatch, sandboxing, graceful shutdown). Exposes extension-contributed tools and providers to the Tool Runtime and Provider Registry. Does **not** execute runbook steps directly.

```go
type ExtensionHost interface {
    // Load discovers and launches all extensions declared in the project manifest.
    Load(ctx context.Context, manifest *ProjectManifest) error

    // Shutdown gracefully stops all running extensions.
    Shutdown(ctx context.Context) error

    // ContributedTools returns all tools contributed by loaded extensions.
    ContributedTools() []*ToolDef

    // ContributedProviders returns all providers contributed by loaded extensions.
    ContributedProviders() []*ProviderDef

    // ContributedPolicyRules returns governance rules contributed by extensions.
    // These compose with host policy; host policy takes precedence on conflict.
    ContributedPolicyRules() []PolicyRule
}
```

**Key Invariants:**
- No extension operates without an explicit capability grant from the host.
- Extension crashes do not crash the host process.
- The Extension Host enforces timeout and cancellation on all RPC calls to extensions.
- Extension-contributed policy rules are additive; they can only *restrict*, not *relax*, host-defined policy.

### API / Adapter Layer

**Responsibility:** The boundary between the Runtime Core and all external consumers (TUI, VS Code extension, web clients, CI pipelines). Each adapter is a thin, stateless renderer: subscribes to the event stream, presents state to the user, and forwards user input (approvals, evidence, cancellation) back to the `RunHandle`. Adapters own **no** execution logic.

```go
// Adapter is the interface every consumer of the Runtime Core must implement.
type Adapter interface {
    // Attach binds the adapter to a RunHandle before the first Next call.
    Attach(handle RunHandle) error

    // Run blocks until the run completes or is cancelled. It is responsible
    // for draining Events(), rendering output, and forwarding user input.
    Run(ctx context.Context) error
}

// The gert serve JSON-RPC layer implements Adapter over stdio.
// The TUI (Bubble Tea) implements Adapter over a terminal.
// Future: a web adapter implements Adapter over WebSocket.
```

**Key Invariants:**
- An adapter may not call `Next` more than once concurrently.
- An adapter may not modify run state directly; all state changes go through `RunHandle`.
- Adapters must handle the case where the event channel is closed (run ended) without blocking.
- An adapter failure must not corrupt the run trace or run state.

## Data Flow

### Phase 1: Parse

User invokes `gert exec runbook.yaml --as eng@corp.com`. CLI reads YAML and calls `Parser.Parse()`. Parser validates against embedded v2 JSON Schema, checks structural semantic rules, returns `ParsedRunbook`. If validation fails, structured diagnostics are returned and execution does not proceed.

### Phase 2: Plan

`ParsedRunbook` passed to `Planner.Plan()`. Planner:
1. Resolves all `imports:` recursively (parse → plan each imported runbook, inline steps with incremented `Depth`).
2. Discovers `.tool.yaml` files for each named tool.
3. Resolves `from:` input bindings to their declared providers.
4. Validates all step references (branch targets, iterate bodies) are valid step IDs.
5. Constructs and returns an `ExecutionPlan`.

### Phase 3: Run Initialization

`Runtime.Start(plan, opts)` is called:
1. Generates `RunID` (format: `YYYYMMDDTHHmmss-xxxxxxxx`).
2. Creates run directory: `.runbook/runs/<runID>/`.
3. Opens append-only trace file: `.runbook/runs/<runID>/trace.jsonl`.
4. Writes `run/started` trace event (runbook path, actor, mode, plan hash).
5. Initializes `GovernanceEngine` from the plan's governance policy.
6. Compiles redaction rules from the governance policy.
7. Returns `RunHandle`; run is now paused at step 0.

### Phase 4: Step Execution Loop

Adapter calls `RunHandle.Next(ctx)` in a loop until `io.EOF`.

**Step Pre-flight (all step types):**
1. Write `step/started` trace event.
2. Emit `StepStarted` runtime event to the event channel.
3. Evaluate template expressions in the step spec against current run variables.

**Governance Pre-flight (cli and tool steps):**
1. **Command check:** `GovernanceEngine.CheckCommand(argv[0])` evaluates the denylist (deny takes precedence) then the allowlist. Violation writes `governance/commandBlocked` trace event; step fails.
2. **Env var check:** `GovernanceEngine.FilterEnvVars()` removes variables matching `deny_env_vars` patterns. Blocked variable names written to `governance/envVarBlocked` trace event.
3. **Contract check:** Step's `Contract.Risk()` evaluated. Steps with `RiskCritical` without explicit approval gate emit `governance/contractWarning` (not a hard failure in v2.0; error in strict mode).
4. **Approval gate:** If step declares `approvals.min > 0`, runtime pauses and emits `ApprovalRequired` event. Blocks until `RunHandle.Approve()` is called with sufficient approvals.

**Dispatch (step-type-specific):**

| Step Type | Behavior |
|-----------|----------|
| `cli` | Fork subprocess. Capture stdout/stderr. Apply redaction before storing. |
| `tool` | Call `ToolRuntime.Invoke(toolName, input)`. Dispatch to appropriate transport. Redaction applied to output. |
| `choice` | Emit `ChoiceRequired` event. Block until `SubmitEvidence()` called. Store result as named variable. |
| `decision` | Emit `DecisionRequired` event. Block until `SubmitEvidence()` called. Alter execution flow by jumping to selected runbook or label. |
| `collector` | Emit `CollectorRequired` event with field schema. Block until `SubmitEvidence()` called. Validate each submitted value. SHA256-hash file/image attachments. Store typed values as run variables. |
| `include` | Resolve referenced runbook. Validate for cycles (DFS). Evaluate `include.when`; skip if false. Apply `include.with` overrides. Inline-expand steps at `Depth+1`. Include step itself does NOT appear in trace. |
| `branch` | Evaluate branch predicates against current variables. Select matching arm. Write `event/branchResolved`. Advance cursor. |
| `iterate` | Evaluate iteration expression. Write `event/iteratePassStart`. Execute body. Write `event/iteratePassEnd`. Repeat. |
| `wait_for_event` | Serialise run state. Register event listener. Set run state to `WAITING`. Return — execution suspended. Only valid in `gert serve` mode; `gert run` MUST reject with error: `"step type 'wait_for_event' requires gert serve mode"`. |

**Step Type Classification:**

| Category | Step Types | Behavior |
|----------|-----------|----------|
| Execution | `cli`, `tool` | Execute external commands. Require governance pre-flight. |
| Interactive | `choice`, `decision`, `collector` | Block execution, emit event requiring user input. Write evidence to trace. |
| Control Flow | `branch`, `iterate`, `include` | Alter execution path or loop body. `include` does not emit a trace entry. |
| Synchronisation | `wait_for_event` | Suspend run until external event arrives. `gert serve` only. |
| Approval Gate | `approve` | Pause pending sign-off. Supports `all`/`any`/`quorum` modes. |

**Step Post-flight (all step types):**
1. Apply outcome evaluation; record `OutcomeRecord`.
2. Write captured variables to run state.
3. Checkpoint run state to `.runbook/runs/<runID>/snapshots/<stepID>.json`.
4. Write `step/completed` trace event (outcome, captures, elapsed time).
5. Emit `StepCompleted` runtime event.

### Phase 5: Run Completion

When `Next` returns `io.EOF`:
1. Write `run/completed` trace event (final outcome, step counts, elapsed time).
2. Flush and sync trace file.
3. Compute SHA256 of trace file; write `run/evidenceHash` record.
4. Optionally archive run directory (compress to `.runbook/archive/`).
5. Close event channel.

## CLI Step Executor Contract

Two execution forms (mutually exclusive):
- **Exec form** (`command`/`args`) — bypasses shell entirely. Injection-safe.
- **Shell-string form** (`run: "<string>"`) — passes script to shell interpreter.
- **Multi-shell map form** (`run: {bash: "…", cmd: "…", …}`) — per-shell scripts; first available shell selected at runtime.

`command` and `run` are mutually exclusive; both present is a schema validation error.

**Shell resolution for string form:**
- `shell: auto` (default): Linux/macOS → `/bin/sh`; Windows → `pwsh` falling back to `cmd.exe`.
- Explicit `shell: bash|sh|pwsh|cmd`: `exec.LookPath(shell)` at plan time; not found → plan-time error.

**Invocation patterns:**

| Shell | Invocation |
|-------|-----------|
| `bash` | `["bash", "-c", script]` |
| `sh` | `["sh", "-c", script]` |
| `pwsh` | `["pwsh", "-NonInteractive", "-Command", script]` |
| `cmd` | `["cmd.exe", "/c", script]` |

**Template expansion:** applied to `run` string *before* it is handed to the shell.

**Security:** Shell-string form is NOT immune to injection. Template variables containing metacharacters (`$`, `` ` ``, `;`, `|`, `&`) will be interpreted by the shell. For untrusted input, use exec form.

**Multi-shell map resolution:**
1. Exact match (first available shell matching a map key).
2. Platform-family fallback: `bash` → `sh`; `pwsh` → `cmd` (Windows only).
3. No match → step fails.

**Platform variables injected at run start:**

| Variable | Type | Values |
|----------|------|--------|
| `gert.platform.os` | string | `"linux"` \| `"darwin"` \| `"windows"` |
| `gert.platform.arch` | string | `"amd64"` \| `"arm64"` \| `"arm"` |
| `gert.platform.shell` | string | Shell name selected for current step (not available inside the same step's `run` script) |

**Validation rules:**

| Rule | Severity | Message |
|------|----------|---------|
| `command` and `run` both present | Error | `Fields 'command' and 'run' are mutually exclusive on type: cli` |
| `run` is a map AND `shell` is set | Warning | `'shell' is ignored when 'run' is a map` |
| `run` map has unknown key | Error | `Unknown shell '{key}' in run map. Valid values: bash, sh, pwsh, cmd` |
| `shell` explicit but not found | Plan-time Error | `Shell '{shell}' not found on this host` |
| `run` map, no key resolves | Run-time Error | `No shell from map {keys} available on this host` |

## Include Step Executor Contract

The `include` step expands a referenced runbook's steps directly into the caller's execution queue — no new run context, no isolated variable scope, no separate audit entry.

**Executor algorithm:**
1. Resolve the referenced runbook by relative path or registry ID.
2. Cycle check (defence-in-depth; primary enforcement is at load time).
3. Evaluate `include.when` — if false, skip all steps.
4. Apply `include.with` overrides to the current variable scope.
5. Inline-expand: replace include step in queue with referenced runbook's steps at `Depth+1`, `Origin` set to the referenced path.
6. Continue execution — expanded steps execute as if defined inline.

**Key properties:**
- No new run context — `RunID`, variable scope, and audit log all shared.
- Include step itself is NOT emitted to trace. Expanded steps appear with `Origin` breadcrumb: `[included from: ./shared/db-preflight.runbook.yaml]`.
- Shared variable scope — mutations by included steps visible to all subsequent steps.

**Cycle detection:** DFS over transitive closure of all `type: include` references at load time. A cycle found at any depth is a hard error:
```
cyclic include: A.yaml -> B.yaml -> A.yaml
```

## Lifecycle Model

**Run states:**
- `created` → `planning` → `ready` → `running`
- From `running`: → `completed` | `failed` | `cancelled` | `suspended`
- From `suspended`: → `resuming` → `running`

**Step states:**
- `pending` → `executing` → `completed`
- From `executing`: → `waiting_for_input` (loops back) | `failed` | `skipped` | `compensating`

**Step advancement is always client-driven.** The runtime does not auto-advance. The `--auto` flag causes the CLI adapter to call `Next` in a tight loop, but the single-step contract is retained.

**Pause and Resume:** A run is paused when `Next` returns a non-terminal result requiring external input (interactive steps or approval gates). The checkpoint written after each completed step is sufficient to resume after process restart. Resumption via `Runtime.Resume(runID)`.

## Wait-for-Event Executor Contract

**WAITING state** is distinct from user-initiated PAUSED state:

| State | Trigger | Exit Condition |
|-------|---------|----------------|
| `RUNNING` | `exec/start` or resume | Active step execution loop |
| `PAUSED` | User-initiated; interactive/approval step | `SubmitEvidence` or `Approve` RPC call |
| `WAITING` | `wait_for_event` step reached | Event Dispatcher delivers matching event → `RUNNING`; or timeout with `on_timeout: fail` → `FAILED`; or timeout with `on_timeout: branch:<id>` → `RUNNING` at branch |
| `COMPLETED` | All steps finished | Terminal |
| `FAILED` | Unhandled error or `on_timeout: fail` | Terminal |

Clients MUST differentiate `WAITING` and `PAUSED` in UX. Clients MUST NOT send `exec/next` while the run is in `WAITING` state.

**Suspend path:**
1. Serialise run state to run store (durable checkpoint).
2. Register event listener with `EventDispatcher.Register(...)`.
3. Set `run.state = WAITING`, write `step/waiting` trace event, return from goroutine.

**Resume path (on event arrival):**
1. Load serialised run state from store.
2. Validate payload against `step.event.payload_schema` (JSON Schema Draft 2020-12).
3. Apply `step.event.filter` predicates — mismatch discards event; listener stays registered.
4. Apply `step.capture` mappings to `run.variables`.
5. Write `step/eventReceived` trace event.
6. Set `run.state = RUNNING` and call `ExecuteNextStep(run)`.

**Run persistence requirement:** Runs in `WAITING` state MUST survive a `gert serve` process restart. Serialised snapshot MUST include: step index/depth, all variable bindings, call stack, and active listener registration (including timeout deadline as absolute timestamp). On restart, server MUST re-register all persisted `WAITING` listeners before accepting new connections.

**Serve-only constraint:** If a runbook with `wait_for_event` is executed via `gert run`, the Runtime Core MUST fail at dispatch time with:
```
step type 'wait_for_event' requires gert serve mode
```

**Webhook HMAC security:**
1. Generate 32 random bytes via `crypto/rand`.
2. Store secret alongside listener registration.
3. Expose `POST /events/{run-id}/{event-id}` — validate `X-Gert-Signature: sha256=<hex>`.
4. Inject secret into run variable `gert.event.{event-id}.token`.
5. Reject requests with invalid/missing signatures with `HTTP 401`.
6. Invalidate secret after first accepted delivery (one-time use).

## Approve Step Executor Contract

### GAP-1: Business Calendar Engine

A stateless utility component within Runtime Core (no external process, no network calls for built-in calendars).

```go
type BusinessCalendar interface {
    // IsBusinessDay reports whether t falls on a business day (Mon–Fri, non-holiday)
    // in the given timezone.
    IsBusinessDay(t time.Time, tz *time.Location) bool

    // AddBusinessDays returns the instant that is exactly n business days after t
    // in the given timezone.  n must be >= 0.
    AddBusinessDays(t time.Time, days int, tz *time.Location) time.Time

    // ElapsedBusinessDays counts completed business days between start and end
    // in the given timezone.  Returns 0 if end <= start.
    ElapsedBusinessDays(start, end time.Time, tz *time.Location) int
}
```

**Built-in calendars (v2.0):**

| Calendar ID | Definition |
|-------------|-----------|
| `default` | Monday–Friday, no holidays |
| `us-federal` | Monday–Friday, US federal public holidays |
| `uk-banking` | Monday–Friday, UK bank holidays for England and Wales |

Custom calendar definitions (YAML-declared holiday lists) are deferred to v2.1.

**Timeout calculation:** One-time conversion at step activation — `deadline = calendar.AddBusinessDays(now, step.timeout_business_days, tz)`. The resulting deadline is stored as a plain `time.Time`; no business-calendar logic is needed at check time.

- If `timeout_business_days` and `timeout` (wall-clock) are both present, the *earlier* deadline applies. Planner warning emitted.

### GAP-2: Quorum Approval Tracker

**ApprovalRecord per step instance:**

```text
ApprovalRecord {
  step_id:     string
  run_id:      string
  mode:        all | any | quorum
  pool:        []string    // eligible approver role names
  required:    int         // 0 = all (for mode: all / any)
  approvals:   []Approval  // received approvals
  rejections:  []Approval  // received rejections
}

Approval {
  approver_id: string      // verified identity of the submitter
  role:        string      // which pool role the approver satisfied
  timestamp:   time.Time
  notes:       string      // optional free-text from the approver
}
```

**Progression rules:**

| Mode | Proceed when |
|------|-------------|
| `all` | `len(approvals) == len(pool)` — every pool member has approved |
| `any` | `len(approvals) >= 1` — at least one pool member has approved |
| `quorum` | `len(approvals) >= required` — required count reached |

**Deduplication:** Each pool member may submit **exactly one** approval. Duplicate submissions return `ErrDuplicateApproval`.

**Deadlock detection** (evaluated after every approval/rejection event):
```text
remaining_possible = len(pool) - len(rejections) - len(approvals)
if (len(approvals) + remaining_possible) < required:
    // quorum is impossible — fail immediately
    step.state = FAILED
    emit(step/failed, {reason: "QuorumImpossible", ...})
```

**Audit trail:** Every approval/rejection appended to trace as `approve/decision` event with: `approver_id`, `role`, `decision` (`approve`|`reject`), `timestamp` (RFC 3339), `notes`. Satisfies SOC 2 Type II and FDA 21 CFR Part 11.

**Input Provider integration:** Approval decisions delivered via `input/submitApproval` with fields `run_id`, `step_id`, `decision`, `notes`. Before recording, runtime:
1. Resolves `approver_id` from authenticated session (or `--as` flag).
2. Verifies `approver_id` maps to at least one role in `approvals.pool`. Unknown identities → `ErrNotAuthorised`.
3. Checks for duplicate (`ErrDuplicateApproval` if already recorded).
4. Appends record, re-evaluates progression rule and deadlock invariant.
5. Persists updated `ApprovalRecord` to run store.

### Composition

Business-day timeout and quorum approval **compose freely**. Both run independently: Quorum Tracker processes approvals/rejections without consulting the calendar; timeout scheduler checks `now >= deadline` without consulting quorum state. Whichever fires first wins.

## Collector Field Validation Contract

When executor receives evidence for a `collector` step, it validates each field value *before* writing any value to run state or trace.

**Validation algorithm (per field, in submission order):**
1. **Required check** — if `required: true` and value absent/empty → validation error.
2. **Type coercion and format check** — parse/coerce to declared type. Parse failure → validation error.
3. **Constraint check** — evaluate `validation.min`, `validation.max`, `validation.step`, option-list membership. Constraint violation → validation error.
4. **Re-prompt** — any validation error triggers new `CollectorRequired` event with `errors` map (field name → message). Provider SHOULD re-render form highlighting invalid fields.
5. **Max retries** — after **3** re-prompt attempts, step fails: `step.state = FAILED`, writes `step/failed` trace event, populates `gert.error` reserved variable.

## Cancellation

`RunHandle.Cancel(ctx, reason)` initiates graceful cancellation:
1. Currently-executing subprocess (if any) receives `SIGTERM` then `SIGKILL` after 5-second grace period.
2. Runtime writes `run/cancelled` trace event with reason and actor.
3. Subsequent `Next` calls return `ErrRunCancelled`.
4. Run state checkpointed as `cancelled` (resumable).

Cancellation does **not** invoke compensation handlers in v2.0. Saga/compensation is deferred to v2.1.

## Crash Recovery

If the gert process crashes mid-run, trace and checkpoint files remain in a consistent state due to append-only, sync-after-write trace discipline. On restart, call `Runtime.Resume(runID)` to continue from the last completed step. Steps already recorded as completed in the trace are not re-executed.

## Concurrency Model

**Single-run-per-process.** `gert serve` is the exception: a long-running process that may host multiple sequential runs for a single VS Code session. Concurrent runs within a single `gert serve` instance are out of scope for v2.0.

**Goroutine model within a single run:**
- **One goroutine per run:** step execution loop runs on a single goroutine; steps execute sequentially.
- **Event fan-out goroutine:** reads from internal event buffer, writes to `Events()` channel.
- **Subprocess goroutine:** drains stdout/stderr concurrently when `cli` step spawns a subprocess.
- **Tool RPC goroutine:** tool invocations over JSON-RPC or MCP, one goroutine per call.

**Thread safety:**
- `RunHandle` is **not safe for concurrent use**. Adapter must serialize all calls. `gert serve` enforces this with a per-session mutex.
- `Events()` channel is safe for a single consumer goroutine.
- `GovernanceEngine` is **read-only after construction** — safe for concurrent reads.
- Trace Writer is protected by an internal mutex.

## `gert serve` Integration

`gert serve` is a JSON-RPC 2.0 server classified as an **adapter in the API/Adapter Layer**. It is a long-lived process that:
1. Starts a stdio JSON-RPC 2.0 listener (newline-delimited JSON over stdin/stdout).
2. On `exec/start`, calls `Runtime.Start()` and stores the `RunHandle` in a session map (keyed by `RunID`).
3. On `exec/next`, calls `RunHandle.Next()` and returns `StepResult` as RPC response.
4. Drains `Events()` channel in a background goroutine and forwards each event as a JSON-RPC notification.
5. On `exec/approve`, `exec/submitEvidence`, or `exec/cancel`, dispatches to the corresponding `RunHandle` method.

**Event forwarding:** Every event from Runtime Core forwarded as a JSON-RPC *notification* (no `id` field) with method `event/<eventType>`.

**RPC Method Summary:**

```bash
exec/start          → Start a new run (returns RunID, first StepResult)
exec/next           → Advance run by one step (returns StepResult)
exec/approve        → Submit approval decision for current gate
exec/submitEvidence → Submit evidence for interactive steps (choice/decision/collector)
exec/cancel         → Cancel the active run
exec/list           → List all runs for this workspace (active + completed)
exec/resume         → Resume a previously interrupted run
runbook/validate    → Validate a runbook YAML (parse + plan, no exec)
runbook/diagram     → Generate a Mermaid or ASCII diagram
event/<type>        → Notification from server to client (not a request)

// HTTP endpoints (gert serve --http, Event Dispatcher)
POST /events/{run-id}/{event-id}  → Deliver a webhook event to a waiting run
                                    (HMAC-SHA256 authenticated)
```

## Event Dispatcher

Lives exclusively inside `gert serve`. Not instantiated by `gert run`. The Runtime Core receives a `DispatcherHandle` at construction time (nil in `gert run` mode); `wait_for_event` checks for nil handle and fails immediately.

```go
type EventDispatcher interface {
    // Register creates a listener for an inbound event on a waiting run.
    // Returns a one-time HMAC token (non-empty for webhook source only).
    Register(ctx context.Context, reg ListenerRegistration) (token string, err error)

    // Deregister removes a listener (called on run completion or cancellation).
    Deregister(runID, eventID string) error

    // Send posts a payload to a named in-process channel listener.
    Send(eventID string, payload json.RawMessage) error

    // RestoreFromStore re-registers all WAITING listeners after process restart.
    RestoreFromStore(store RunStore) error
}

type ListenerRegistration struct {
    RunID          string
    EventID        string
    Source         EventSource          // webhook | message | signal | channel
    Filter         map[string]string
    PayloadSchema  string               // JSON Schema URL, may be empty
    TimeoutAt      time.Time
    OnTimeout      TimeoutPolicy        // fail | continue | branch:<step-id>
}
```

**Webhook transport:** `POST /events/{run-id}/{event-id}` authenticated with HMAC-SHA256. Returns: `HTTP 202` on success; `404` if no listener; `401` on signature failure; `410 Gone` if run no longer in `WAITING` state.

**Timeout enforcement:** min-heap of timeout deadlines; background goroutine fires on earliest deadline.
