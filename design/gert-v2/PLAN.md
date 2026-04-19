# Gert v2 — Implementation Plan

**Author:** Barbara (Software Architect)  
**Date:** 2026-04-18  
**Status:** Authoritative build roadmap — agents pick up one phase at a time.  
**Source of truth:** All spec files in `design/gert-v2/spec/` supersede this document where they conflict.

---

## 1. Problem & Approach

### What v2 Solves

v1 delivered the core value proposition — governed, executable runbooks with audit traces — but accumulated five architectural constraints that cannot be fixed incrementally. First, `gert serve` was tightly coupled to the Runtime Core; adding a new adapter (web, CI, mobile) required changes to execution semantics. Second, the schema versioning was brittle: there were no formal compatibility contracts, no `$schema` field, and no automated migration toolchain, leaving production users stranded between versions. Third, extensions had no formal capability model — third-party tool authors had to reverse-engineer which internal packages to import, and a crash in any extension could take down the host. Fourth, v1 offered no saga/compensation pattern, meaning a partially-executed runbook that modified infrastructure left systems in undefined intermediate state. Fifth, there was no OpenTelemetry support, preventing correlation between runbook execution and infrastructure observability data.

### Architectural Philosophy

v2's governing rule is that **dependencies flow inward toward the Core Domain**. The Parser knows nothing about the Runtime; the Runtime knows nothing about adapters; no adapter may import internal packages. Every component boundary is a Go interface with a named, versioned contract. Extensions run as child processes and are granted explicit capabilities; they cannot weaken host-level governance and they cannot crash the host. Adapters observe, render, and forward user input — they own no execution logic. The JSONL trace file is the authoritative source of truth for every run; all UI is derived from events, not from internal state.

### Build Sequencing

The build is sequenced as a dependency chain: shared types and interfaces first (Phase 0), then the parse-plan-execute pipeline (Phases 1–3), then governance enforcement (Phase 4), then step type implementations that exercise the pipeline (Phase 5), then the integration surfaces — tools, extensions, and providers — that step types call into (Phases 6–8), then the server and adapter layer that drives runs from the outside (Phases 9–10), then the durability and replay features that depend on a complete execution model (Phase 11), then the operational layer of observability and security hardening (Phase 12), then migration tooling (Phase 13), and finally the comprehensive test and acceptance gate (Phase 14). Each phase produces an independently testable increment; no phase should leave a subsequent phase guessing about an interface.

---

## 2. Principles

1. **Spec files are the source of truth.** Every deliverable must be traceable to a section in `spec/*.md`. If the spec and this plan conflict, the spec wins. If the spec is silent, resolve via the open questions in §09 before writing code.

2. **Interfaces before implementations.** Each phase begins by landing the Go interfaces in `internal/` that subsequent phases will consume. An interface with a `MemFoo` stub implementation is a shippable phase deliverable; a real implementation without a tested interface is not.

3. **No adapter knows about execution semantics.** The TUI, VS Code adapter, and `gert serve` layer MUST NOT call `Runtime` methods directly. They attach via `RunHandle.Events()` and forward user input through `RunHandle.Approve()`, `RunHandle.SubmitEvidence()`, and `RunHandle.Cancel()`. The `no_import` contract test enforces this at CI time.

4. **Each phase must be independently testable.** Every phase defines concrete exit criteria — specific `go test` invocations, CLI commands, and trace assertions — that can be evaluated without completing subsequent phases.

5. **Governance is enforced in the Runtime Core, never delegated.** Allowlist, denylist, env blocking, and redaction are evaluated by the `GovernanceEngine` inside the execution loop before any subprocess fork. Extensions may add restrictions via `ContributedPolicyRules()` but may never relax host policy. No exception.

6. **Open questions from §09 are blockers, not tasks.** Q2 (concurrency model) and Q3 (trace format compatibility) MUST be resolved and recorded in `.squad/decisions.md` before Phase 3 begins. Any phase that touches an unresolved question must surface the dependency explicitly in its entry criteria.

7. **Security by construction, not by addition.** Redaction, capability grants, Ed25519 signature verification, and tool binary checksums are designed in from Phase 0 interfaces. They are not retrofitted in a hardening phase.

8. **v2.0 scope is bounded.** Saga/compensation (`compensate` step type), retry policy (`step/retrying`), OPA integration, `parallel` step execution, and gRPC transport for VS Code are explicitly deferred to v2.1. Any phase that encounters a v2.1 feature should implement a stub that panics with `"not implemented in v2.0"` rather than silently skipping the feature.

---

## 3. Phase Table

| Phase | Name | What it delivers | Key spec sections | Depends on |
|-------|------|-----------------|-------------------|------------|
| 0 | Foundation | Module scaffold, shared types, all Go interfaces, embedded JSON Schema, `apiVersion` parsing | §02, §03, §06, §13 | — |
| 1 | Parser & Schema | Full `Parser` implementation, v2 JSON Schema validation, v1 compat mode, `ParsedRunbook` | §03, §08 | 0 |
| 2 | Planner | `Planner` implementation, import resolution, tool discovery (static), `ExecutionPlan`, cycle detection | §02, §03, §05 | 1 |
| 3 | Runtime Core & Trace | Engine execution loop, `DirRunStore`/`MemRunStore`, event bus, `TraceWriter`, run lifecycle, `RunHandle` | §02, §06, §12 | 2, **Q2 resolved**, **Q3 resolved** |
| 4 | Governance Engine | Full allowlist/denylist, env blocking, redaction pipeline, approval gates, RBAC pre-flight | §07, §11 | 3 |
| 5 | Step Types | All v2.0 step types: `cli`, `tool`, `choice`, `decision`, `collector`, `end`, `branch`, `iterate`, `include`, `approve`, `assert`, `wait_for_event` | §02, §03 | 3, 4 |
| 6 | Tool Runtime | Tool discovery (static + MCP), all three transports (`stdio`/`stdio-jsonrpc`/`mcp`), capability gate, output capture, cancellation | §05, §13 | 5 |
| 7 | Extension Host | Manifest loading, lifecycle state machine, JSON-RPC handshake, capability grants, crash isolation, policy contribution | §04, §07, §13 | 6 |
| 8 | Input Provider Framework | `provider/v2` schema, resolution protocol, prefix routing, built-in providers (`env`/`file`/`prompt`/`workspace`), composition chains | §14, §05 | 6 |
| 9 | `gert serve` & Adapter Contracts | `exec/v2` JSON-RPC methods, `events/v2` WebSocket, RBAC enforcement, TLS/mTLS, health/ready endpoints, contract test suite | §13, §07, §15 | 3, 4, 5, 6, 7, 8 |
| 10 | Adapters | Bubble Tea TUI adapter, VS Code extension TypeScript client, `exec/v2` + `events/v2` compliance | §13, §01 (G3, G7) | 9 |
| 11 | Evidence, Resumption & Replay | Advanced snapshots, `gert exec --resume`, scenario replay, `gert test`, `gert trace migrate`, HMAC chaining | §12, §08 | 3, 4, 5 |
| 12 | Observability & Security Hardening | OTel spans (all layers), Prometheus metrics, structured logging, `gert diagnose`, `gert verify`, Ed25519 extension signing, tool binary checksums | §15, §07 | 6, 7, 9 |
| 13 | Migration Tooling | `gert migrate`, v1 compat shim, `gert trace migrate`, field mapping, diff report | §10 | 1, 3 |
| 14 | Testing & Acceptance | Contract test suite, integration fixture corpus, golden traces, performance benchmarks, v1 corpus regression ≥95% | §08 | All phases |

---

## 4. Per-Phase Detail

---

### Phase 0 — Foundation

**Goal:** Establish the Go module, package layout, shared types, all top-level Go interfaces, and the embedded JSON Schema artifact that all subsequent phases depend on.

**Deliverables:**
- `go.mod` at `github.com/ormasoftchile/gert/v2` (Go 1.22+)
- `internal/schema/` — embedded v2 JSON Schema files (`runbook/v2.json`, `tool/v2.json`, `provider/v2.json`) via `embed.FS`; `ParseAPIVersion(data []byte) (string, error)` dispatcher
- `internal/types/` — canonical shared types: `DurableEvent`, `Event`, `RunState`, `StepKind` (all 14 enum values), `RunMode`, `StepResult`, `GovernancePolicy`, `EvidenceValue`, `ApprovalDecision`, `TreePath`
- `internal/parser/interface.go` — `Parser` interface, `ParsedRunbook`, `ParseOptions`
- `internal/planner/interface.go` — `Planner` interface, `ExecutionPlan`, `ResolvedStep`, `PlanOptions`
- `internal/engine/interface.go` — `Runtime`, `RunHandle`, `RunOptions`, `Adapter` interfaces
- `internal/governance/interface.go` — `GovernanceEngine`, `PolicyRule`, `PolicyDecision`
- `internal/toolruntime/interface.go` — `ToolRuntime`, `ToolInvocation`, `ToolResult`
- `internal/exthost/interface.go` — `ExtensionHost` interface
- `internal/providers/interface.go` — `ProviderRegistry`, `ProviderDef`, `ResolutionResult`
- `internal/runstore/interface.go` — `RunStore`, `TraceWriter` interfaces (DirRunStore contract)
- `pkg/testutil/` — `ReadTrace`, `AssertEvent`, `AssertEventOrder`, `AssertNoEvent` helpers
- `Makefile` with targets: `build`, `test`, `test-integration`, `lint`, `schema-validate`

**Spec references:**
- §02 Architecture: all five component interfaces, dependency direction
- §03 Schema: apiVersion values (`runbook/v2`, `tool/v2`, `provider/v2`), `$schema` URL
- §06 Events: `DurableEvent` envelope fields (`event_id`, `run_id`, `kind`, `sequence`, `payload`, `timestamp`)
- §13 Adapter Contracts: four contract surface names (`exec/v2`, `events/v2`, `tool/v2`, `extension/v2`)

**Entry criteria:** Clean Go module with `go.work` or standalone. No external dependencies beyond `encoding/json`, `context`, `io` in the `internal/` tree.

**Exit criteria:**
- `go build ./...` succeeds with zero errors
- `go vet ./...` produces no warnings
- `go test ./...` passes (interfaces have no-op stub implementations that compile)
- All 14 `StepKind` enum values compile and are covered by a `switch` with a `default: panic` — verified by test
- Embedded JSON Schema files load without error in `TestSchemaLoad`

**Open questions:** None blocking this phase.

---

### Phase 1 — Parser & Schema

**Goal:** Implement a `Parser` that validates a YAML byte stream against the v2 JSON Schema, applies structural semantic checks, and returns an immutable `ParsedRunbook`.

**Deliverables:**
- `internal/parser/parser.go` — `DefaultParser` implementing `Parser` interface
- `internal/parser/validate.go` — JSON Schema Draft 2020-12 structural validation via `github.com/santhosh-tekuri/jsonschema/v6` (or equivalent); validation errors carry file/line/column/message
- `internal/parser/compat.go` — v1→v2 field normalization shim: `meta.*` promotions, `tree`→`flow`, `collector`→choice/decision/collector, `router`→`end` (Q1 choice a or c determines whether shim is default-on or `--compat-v1`)
- `internal/parser/semantic.go` — structural semantic checks: duplicate step IDs, unknown step types, malformed Go template expressions (parsed not evaluated), `from:` binding syntax validation
- `internal/parser/expression.go` — template syntax validator (uses `text/template.Parse` with custom funcmap stubs for all v2 built-ins listed in §03)
- `testdata/runbooks/valid/` — 12 valid fixture runbooks (one per top-level runbook feature)
- `testdata/runbooks/invalid/` — 15 invalid fixtures (structural and semantic errors)
- `cmd/gert/validate.go` — `gert validate <path>` subcommand (exits 0 = valid, 1 = invalid)

**Spec references:**
- §03: `apiVersion` values, top-level field inventory, step type taxonomy (14 types), expression language, `from:` binding syntax, v1→v2 compat contract table
- §08 §Contract Tests: schema contract tests — every `testdata/` fixture must pass/fail as expected

**Entry criteria:** Phase 0 complete; `internal/schema/` embeds all three JSON Schema files.

**Exit criteria:**
- `gert validate testdata/runbooks/valid/*.yaml` exits 0 for all valid fixtures
- `gert validate testdata/runbooks/invalid/*.yaml` exits 1 for all invalid fixtures, each with a structured error message containing path, line, and message
- `go test ./internal/parser/...` covers all 14 step type parse paths
- v1 runbook (`apiVersion: runbook/v1`) parses successfully in compat mode with deprecation warning to stderr
- v0 runbook (`apiVersion: runbook/v0`) is rejected with error: `"v0 schema not supported; migrate to v1 first"`

**Open questions:**
- **Q1** (v1 compat mode): which option is chosen (full transparent shim vs. `--compat-v1` flag)? Decision affects whether the compat shim in `parser/compat.go` is on-by-default or opt-in. Must be resolved before implementation begins.

---

### Phase 2 — Planner

**Goal:** Implement a `Planner` that resolves all import cycles, discovers tool definitions (static path only), validates `from:` bindings, and returns an `ExecutionPlan`.

**Deliverables:**
- `internal/planner/planner.go` — `DefaultPlanner` implementing `Planner` interface
- `internal/planner/imports.go` — recursive import resolution (DFS with cycle detection); each resolved `include` step is inlined at `Depth+1` with `Origin` breadcrumb
- `internal/planner/tools.go` — static tool discovery: built-in registry → `tools/<name>.tool.yaml` → `--tool-dir` paths → `requires:` packages. Tool definition parser for `tool/v2` YAML
- `internal/planner/providers.go` — static `from:` binding resolution to registered `ProviderDef` by prefix; unmatched `from:` bindings left unresolved (runtime fallback to interactive prompt)
- `internal/planner/governance.go` — hydration of `GovernancePolicy` from runbook's `governance:` block; compilation of redaction regexps
- `internal/schema/tool-v2.go` — `ToolDef` struct with all `tool/v2` fields (§05 schema)
- `testdata/runbooks/cyclic.yaml` — cyclic import fixture
- `cmd/gert/plan.go` — `gert plan <path>` (debug subcommand: prints `ExecutionPlan` as JSON, not in production binary)

**Spec references:**
- §02 Planner: key invariants (all tool names resolved, all imports acyclic, safe to plan without running)
- §05 Tool Runtime: tool discovery phases 1 (static only — MCP discovery is Phase 6)
- §03: `include` step cycle detection, `from:` binding syntax, `toolRefs` field
- §11 Governance: YAML policy block parsing (allowlist, denylist, env patterns, redaction rules)

**Entry criteria:** Phase 1 complete; `ParsedRunbook` is available.

**Exit criteria:**
- `go test ./internal/planner/...` passes all table-driven tests including:
  - linear runbook with 3 tool refs → `ExecutionPlan` has 3 `ResolvedStep` entries each with `ToolDef` populated
  - cyclic import → plan-time error `"cyclic include: A.yaml → B.yaml → A.yaml"`
  - unknown tool ref → plan-time error listing missing tool name and search paths
  - `from: env.API_KEY` binding → `ProviderDef` of kind `env` present in plan
- Planning does not read from the network or execute subprocesses (pure I/O from filesystem)

**Open questions:** None blocking this phase.

---

### Phase 3 — Runtime Core & Trace

**Goal:** Implement the engine execution loop, the append-only JSONL trace writer, the bounded event bus, and the `RunHandle` — the minimum required to run a single-step runbook end to end.

**Deliverables:**
- `internal/engine/engine.go` — `DefaultRuntime` implementing `Runtime`; `DefaultRunHandle` implementing `RunHandle`
- `internal/engine/loop.go` — `Next()` implementation: pre-flight → dispatch stub → post-flight → checkpoint
- `internal/engine/events.go` — bounded event bus (`chan Event` of configurable buffer, discard-on-full for in-process subscribers)
- `internal/runstore/dir.go` — `DirRunStore` implementing `RunStore`: `WriteTrace` (O_APPEND + fsync), `ReadTrace`, `ReadTraceSince`, `SaveCheckpoint` (atomic rename), `LoadLatestCheckpoint`, `AcquireLock`, `ReleaseLock`, `WriteManifest`
- `internal/runstore/mem.go` — `MemRunStore` for unit tests; full `RunStore` interface compliance
- `internal/engine/runstate.go` — `RunState` struct (§12 fields): `RunID`, `RunbookPath`, `Mode`, `Actor`, `Path` (TreePath cursor), `Vars`, `Captures`, `History`, `InvokeStack`, `LastCheckpointSeq`
- Mandatory trace events emitted: `run/started`, `step/started`, `step/completed`/`step/failed`/`step/skipped`, `run/completed`, `run/cancelled`
- Run directory structure: `.runbook/runs/<run-id>/trace.jsonl`, `snapshots/`, `attachments/`, `run.yaml`, `lock`
- `cmd/gert/exec.go` — `gert exec <path> --as <actor> [--mode real|dry-run]`

**Spec references:**
- §02 Runtime Core: `Runtime`/`RunHandle` interfaces, key invariants, execution phases 3–5
- §06 Events: event envelope schema, all required vs. optional events, in-order delivery guarantee
- §12: trace file format, JSONL envelope, `DirRunStore.WriteTrace` crash safety, atomic snapshot write
- §08: Core Runtime Contract test: `ConformanceTest` must pass

**Entry criteria:** Phase 2 complete. **Q2 (concurrency model) MUST be resolved** — the engine architecture differs significantly between single-run-per-process (option a) and concurrent runs (option b/c). **Q3 (trace format compat) MUST be resolved** — the JSONL envelope fields are fixed at this phase.

**Exit criteria:**
- `ConformanceTest` (§08) passes against `DefaultRuntime` with a fixture single-step `cli` stub runbook
- `go test -race ./internal/engine/...` passes (no data races on the event bus)
- `gert exec testdata/runbooks/valid/hello-world.yaml --as test` produces:
  - A `trace.jsonl` with `run/started` as first line and `run/completed` as last line
  - All lines are valid JSON and structurally valid against `spec/event-schema.json`
  - Sequence numbers are monotonically increasing from 0
- `DirRunStore` write+fsync benchmark: p50 < 5ms, p99 < 20ms (§08 perf criteria)
- A partial write that fails mid-event does not corrupt prior events (verified by `TestTraceCorruptionRecovery`)

**Open questions:**
- **Q2** (concurrency model) — BLOCKER: determines `RunHandle` ownership and state isolation model
- **Q3** (trace format compat) — BLOCKER: determines `DurableEvent` field names (seq vs. sequence, type vs. kind, data vs. payload)

---

### Phase 4 — Governance Engine

**Goal:** Implement the complete governance enforcement pipeline: command allowlist/denylist, environment variable blocking, output redaction, approval gates, RBAC pre-flight, and all governance trace events.

**Deliverables:**
- `internal/governance/allowlist.go` — `AllowlistChecker`: exact-match-only (no glob in v2.0) against `allowed_commands`; emits `governance/command_checked`
- `internal/governance/denylist.go` — `DenylistChecker`: runs before allowlist; emits `governance/command_checked` with `verdict: denied`; denylist match is a hard stop
- `internal/governance/envblock.go` — `EnvBlocker`: glob-pattern match against `deny_env_vars`; strips matching vars; emits `governance/envVarBlocked` (var names only, never values)
- `internal/governance/redact.go` — `RedactionPipeline`: compiles RE2 patterns at plan time; applies to all captured output before trace write or variable store; emits `governance/redaction_applied`
- `internal/governance/approval.go` — `ApprovalGate`: `ApprovalDecision` validation (actor, role, same-actor prohibition, Ed25519 signature if server mode, replay protection); `governance/approvalRequired`, `governance/approvalDecision`, `governance/approvalGranted`/`Rejected`/`TimedOut` events
- `internal/governance/rbac.go` — `RBACChecker`: `requires_role`/`requires_any_role`/`requires_all_roles` on runbook and per-step; emits `governance/accessDenied`; evaluated at run pre-flight and per-step pre-flight
- `internal/governance/engine.go` — `DefaultGovernanceEngine` composing all checkers; exposes `CheckCommand`, `FilterEnvVars`, `Redact`, `CheckRBAC`, `EvaluateApproval`
- `internal/governance/contract.go` — `ContractEvaluator`: evaluates step `contract.effects` against `governance.rules`; emits `governance/contractWarning` for `RiskCritical` effects without approval gate
- `testdata/governance/` — fixture policies: permissive, restrictive, approval-gated, denylist-only

**Spec references:**
- §11 Governance: all six primitives, evaluation order (denylist → allowlist → approval → env → redact), policy evaluation model table, approval gate design, UX contract, approval recording in trace
- §07 Security: credential handling, mandatory redaction, approval token Ed25519 signing
- §08: `TestGovernanceAllowlist` table-driven test pattern, capability violation test

**Entry criteria:** Phase 3 complete; `GovernanceEngine` interface defined in Phase 0.

**Exit criteria:**
- Table-driven tests cover: exact allowlist match (allow), non-match (deny), denylist match (deny even if in allowlist), env var glob strip, redaction pattern match, approval gate quorum-met and single-rejection-vetoes
- `gert exec` of a runbook with `denied_commands: [rm]` that attempts `rm` → `step/failed` with `error_code: governance_blocked` in trace
- Same actor cannot submit two approvals for the same gate (`TestSameActorDoubleApproval`)
- `on_timeout: fail` approval gate → `governance/approvalTimedOut` in trace within deadline
- Redaction output never contains original sensitive value in any trace field (property-based test with `testing/quick`)

**Open questions:**
- **Q4** (extension policy composition) — Additive-only model assumed; if decision changes, `ContributedPolicyRules()` enforcement logic in Phase 7 must change
- **Q12** (approval timeout/escalation) — Q12 option (a), (b), or (c) determines whether `escalate_to` and `on_timeout: escalate` fields are parsed or produce a "not implemented" error in v2.0

---

### Phase 5 — Step Types

**Goal:** Implement all v2.0 step type executors: `cli`, `tool`, `choice`, `decision`, `collector`, `end`, `branch`, `iterate`, `include`, `approve`, `assert`, and `wait_for_event`.

**Deliverables:**
- `internal/engine/steps/` — one file per step type:
  - `cli.go` — exec form (`command`/`args`) and run form (shell-string); governance pre-flight calls; subprocess fork; stdout/stderr/exit_code capture; redaction applied before store; `governance/command_checked` event
  - `tool.go` — stub dispatcher to `ToolRuntime.Invoke()` (full implementation in Phase 6); capability gate check; output capture and redaction
  - `choice.go` — emits `input/prompted`; blocks on `SubmitEvidence`; validates selection against declared options; stores result in named variable
  - `decision.go` — emits `input/prompted`; blocks on `SubmitEvidence`; advances `TreePath` cursor to selected route; does NOT store route as variable
  - `collector.go` — emits `input/prompted` with field schema; blocks on `SubmitEvidence`; validates each field type (text, checklist, attachment, etc.); SHA256-hashes file attachments; stores to `attachments/` and records in `step/completed.evidence`
  - `end.go` — writes `OutcomeRecord` with `category` and `code`; triggers `run/completed`
  - `branch.go` — evaluates all condition templates; selects first-match arm; emits `step/completed` with selected arm label; advances cursor to first step of arm
  - `iterate.go` — evaluates collection expression; writes `iteratePassStart`/`iteratePassEnd`; updates `IterateState` in `RunState`; emits `step/skipped` with `reason: iterate_exhausted` when done
  - `include.go` — cycle-check (defence-in-depth); evaluates `include.when`; applies `include.with` overrides; inlines steps at `Depth+1` with `Origin` breadcrumb; include step itself NOT emitted to trace
  - `approve.go` — standalone approval gate (same mechanics as `step.approvals` but as a first-class step type); `all`/`any`/`quorum` modes from schema
  - `assert.go` — evaluates guard expression; `step/failed` if false with `error_code: assertion_failed`
  - `wait_for_event.go` — **`gert serve` mode only**; serializes run state; registers external event listener; returns from `Next()` with `RunHandle.State().Status = "waiting"`; `gert exec` rejects with error
- `internal/engine/steps/registry.go` — dispatcher mapping `StepKind` → executor function
- Stubs for v2.1 types: `parallel.go`, `compensate.go` — both `panic("not implemented in v2.0")`
- `testdata/runbooks/` — one fixture per step type (happy path + failure path)

**Spec references:**
- §02 Architecture: step dispatch table, step pre-/post-flight contracts, `include` executor contract, `wait_for_event` executor contract, lifecycle model
- §03 Schema: all 14 step type field inventories, `cli` shell resolution, `include` cycle detection algorithm, common step fields (`when`, `retry`, `capture`, `contract`, `scope`)
- §06 Events: `input/prompted`, `input/received` (required for `choice`/`decision`/`collector`)
- §12: `step/completed.evidence` format, SHA256 attachment hashing, `EvidenceValue` kinds

**Entry criteria:** Phases 3 and 4 complete; `GovernanceEngine` integrated into `Next()` pre-flight.

**Exit criteria:**
- Each step type has ≥1 happy-path and ≥1 failure-path integration test
- `gert exec` of a runbook with a `branch` step produces exactly one arm's steps in the trace
- `gert exec` of a runbook with `iterate` over a 3-element list produces 3 trace pass records
- A `cli` step with `when: "{{ eq .skip true }}"` where `skip=true` emits `step/skipped` with `reason: branch_not_taken`
- `wait_for_event` step in `gert exec` (non-serve) mode exits with error code and message within 1 second
- `assert` step with false condition emits `step/failed` with `error_code: assertion_failed` and terminates the run

**Open questions:**
- **Q9** (parallel execution model) — `parallel` step is v2.1; the stub must compile but not execute. If Q9 is resolved to "explicit parallel blocks in v2.0," Phase 5 must be reopened.

---

### Phase 6 — Tool Runtime

**Goal:** Implement the complete tool invocation layer: all three transports (`stdio`, `stdio-jsonrpc`, `mcp`), tool discovery (static + MCP), capability gate, output capture and redaction, and cancellation propagation.

**Deliverables:**
- `internal/toolruntime/runtime.go` — `DefaultToolRuntime` implementing `ToolRuntime`
- `internal/toolruntime/discovery.go` — 4-phase resolution order: built-in registry → project `tools/` → workspace `tool-paths` config → `requires:` packages; name collision warning; tool catalog
- `internal/toolruntime/transports/stdio.go` — per-invocation `exec.Command`; controlled env; stdout/stderr captured; non-zero exit → `TOOL_NONZERO_EXIT` error envelope (§05)
- `internal/toolruntime/transports/jsonrpc.go` — persistent process per run; `startup.ready-pattern` detection; JSON-RPC 2.0 invocation/response/error envelope (§05 envelopes); `tools/cancel` request; per-invocation timeout via `context.WithTimeout`
- `internal/toolruntime/transports/mcp.go` — MCP `initialize`/`initialized`/`tools/list` handshake; name registered as `<server>/<tool>`; `tools/call` with `_gert_context` extension; MCP content array → string flattening
- `internal/toolruntime/context.go` — `InvocationContext` struct; injected as `GERT_*` env vars (stdio) or `context` object (jsonrpc/mcp)
- `internal/toolruntime/capture.go` — `stdout`/`stderr`/`exitCode` capture; dot-notation JSON path extraction; redaction pipeline applied before variable store
- `internal/toolruntime/capgate.go` — `CheckCapabilities`: validates `governance.requires-capabilities` against run's granted set; `CAPABILITY_DENIED` fast-fail; `allowed-environments` mode check
- `internal/toolruntime/cancel.go` — run-level cancel propagation: `tools/cancel` to in-flight JSON-RPC/MCP; SIGTERM + 5s grace + SIGKILL to stdio
- `internal/toolruntime/checksum.go` — optional SHA256 binary verification (§07); `governance/toolChecksumMismatch` trace event on mismatch
- `tools/` — at least 2 built-in tool definitions as `tool/v2` YAML fixtures
- `testdata/tools/` — mock tool binaries for integration testing

**Spec references:**
- §05: complete tool runtime spec (all subsections); MCP integration; invocation context fields; error code ranges; cancellation; tool versioning
- §13 Adapter Contracts: `tool/v2` invocation and success/error envelopes
- §06 Events: `tool/invoked`, `tool/completed` (optional, recommended)
- §07: tool binary checksum verification

**Entry criteria:** Phase 5 complete; `tool` step type stub replaced with real dispatcher.

**Exit criteria:**
- `go test ./internal/toolruntime/...` covers all three transports with mock binaries
- `gert exec` of a runbook with a `tool` step using `stdio` transport: correct stdout captured in trace
- `stdio-jsonrpc` transport: single persistent process handles 3 sequential invocations (verified via process PID stability)
- MCP discovery: `tools/list` response registers 2 tools under `<server>/<name>` namespace
- Capability gate: tool requiring `capability/network` on a run without that capability → `step/failed` with `error_code: CAPABILITY_DENIED` before subprocess fork
- Tool with binary checksum mismatch → `governance/toolChecksumMismatch` in trace, step fails, process not forked
- Per-invocation timeout: tool that sleeps > declared timeout → `step/failed` with `error_code: timeout` within 500ms of deadline

**Open questions:** None blocking this phase.

---

### Phase 7 — Extension Host

**Goal:** Implement the `ExtensionHost`: manifest discovery, compatibility evaluation, process lifecycle, JSON-RPC capability handshake, contributions registration, and crash isolation.

**Deliverables:**
- `internal/exthost/host.go` — `DefaultExtensionHost` implementing `ExtensionHost`
- `internal/exthost/discovery.go` — 4-source resolution order: built-in → workspace config (`.gert/extensions.yaml`) → runbook `extensions:` field → `--extension` CLI flag; deduplication by `meta.name`; precedence rules
- `internal/exthost/manifest.go` — `gert-extension.yaml` v2 parser; structural validation; capability taxonomy (10 named capabilities, §04 table); compatibility range check against host API version
- `internal/exthost/lifecycle.go` — 8-state machine: `discovered → verified → starting → initializing → ready → draining → stopped → crashed`; transitions with proper event emissions
- `internal/exthost/handshake.go` — `extension/initialize` (host→ext): sends `grantedCapabilities`, `runContext`; validates `acknowledgedCapabilities`; error codes `-32001`, `-32002`, `-32003`
- `internal/exthost/contributions.go` — `contributions/list` call post-handshake; merges contributed tools into `ToolRuntime` catalog; merges contributed providers into `ProviderRegistry`; contributed policy rules registered with `GovernanceEngine`
- `internal/exthost/ping.go` — periodic ping loop (default 5s); unresponsive extension → `SIGTERM`; crash recovery
- `internal/exthost/sandbox.go` — environment variable filtering (only `env-read-patterns` forwarded); path/host validation for `file-read`/`file-write`/`network` capabilities; OS-specific enforcement stubs (seccomp/pledge noted, host-side validation for all platforms)
- `internal/exthost/trust.go` — trust level evaluation: local-trusted (no sig), project-scoped (optional sig), remote/signed (mandatory Ed25519 verification); `governance/extensionRejected` trace event on failure
- `cmd/gert/extension.go` — `gert extension verify <manifest>` command
- `pkg/exttest/` — `Harness`, `WithManifest`, `WithCapabilities`, `InvokeTool`, `RegisteredTools` (§08 extension test harness)

**Spec references:**
- §04: complete extension runtime spec (all subsections); capability taxonomy (full table); manifest format; discovery model; handshake protocol; lifecycle state machine; versioning; sandboxing
- §07: extension trust model (3 levels), Ed25519 signature verification algorithm, `governance/extensionRejected` event
- §13 Adapter Contracts: `extension/v2` contract
- §08: extension harness, handshake conformance tests

**Entry criteria:** Phase 6 complete; `ToolRuntime` accepts dynamically registered tool definitions.

**Exit criteria:**
- `go test ./internal/exthost/...` covers: successful handshake, incompatible version rejection, capability denial graceful degradation, ping timeout crash recovery
- Extension that requests a capability not granted: `acknowledgedCapabilities` matches granted set, extension degrades gracefully without hard failure
- Extension process crash mid-invocation: host continues; `extension.crashed` trace event emitted; in-flight call returns `-32099` error; no host panic
- Extension with `remote/signed` trust level and invalid Ed25519 signature: rejected before subprocess started; `governance/extensionRejected` in trace
- `pkg/exttest.Harness` allows testing an extension without a real runbook
- `go test -race ./internal/exthost/...` passes

**Open questions:**
- **Q7** (extension signing model): Which option (a–d) is chosen determines whether `trust.go` requires signatures for all remote extensions, only those above a capability threshold, or supports Sigstore/cosign. Resolution needed before implementing `trust.go`.
- **Q11** (extension schema namespace): determines how extension-contributed schema fragments are validated at `contributions/list` time.

---

### Phase 8 — Input Provider Framework

**Goal:** Implement the input provider framework: `provider/v2` schema, prefix-based routing, JSON-RPC resolution protocol, built-in providers, composition chains, and provider lifecycle.

**Deliverables:**
- `internal/providers/registry.go` — `DefaultProviderRegistry` implementing `ProviderRegistry`; prefix-routing (longest-match wins); provider deduplication
- `internal/providers/schema.go` — `provider/v2` YAML parser; `ProviderDef` struct with all fields from §14 schema
- `internal/providers/resolver.go` — resolution flow: prefix match → provider start → batch collection → `provider/resolve` JSON-RPC call → result merge → fallback chain
- `internal/providers/transport.go` — provider transport layer (same three transports as tool runtime: `stdio`, `stdio-jsonrpc`, `mcp`); request envelope (§14 resolution protocol)
- `internal/providers/builtin/env.go` — `env` built-in: reads `env.<NAME>`, strips `GERT_` prefix guard
- `internal/providers/builtin/file.go` — `file` built-in: reads first line of `file.<path>`; path validation
- `internal/providers/builtin/prompt.go` — `prompt` built-in: interactive terminal prompt for `text`/`secret`/`checklist`/`attachment`; supports `choice`, `decision`, `collector` step types (§14 §6)
- `internal/providers/builtin/workspace.go` — `workspace` built-in: reads from `.gert/config.yaml` static values
- `internal/providers/cache.go` — per-provider cache (scopes: `run` | `step` | `none`; TTL enforcement)
- `internal/providers/crash.go` — provider crash handling; `provider.crashed` trace event; fallback to interactive prompt on crash
- `providers/` — at least 2 example `provider/v2` YAML definitions as fixtures

**Spec references:**
- §14: complete input provider framework spec (all subsections); provider definition schema; resolution protocol; resolution request/response envelopes; built-in providers; composition chains; lifecycle
- §03: `from:` binding syntax, `inputs` declaration, `secret` type handling
- §07: sensitive value handling; `sensitive: true` marking; no-secret-to-trace guarantee
- §08: input resolution contract tests

**Entry criteria:** Phase 6 complete (transport layer shared with tool runtime). Phase 5 complete (`choice`/`decision`/`collector` step types ready to call provider resolution).

**Exit criteria:**
- `go test ./internal/providers/...` covers: prefix routing (longest match wins), `provider/resolve` batch call, cache hit/miss, fallback on unresolved binding
- `from: env.API_KEY` → value read from environment without subprocess invocation
- `from: pd.incident.title` → JSON-RPC `provider/resolve` sent to mock PD provider process
- Provider that crashes mid-resolution: run continues; `provider.crashed` trace event; next step picks up from fallback prompt (not from crashed provider)
- A `secret` type input: value reaches step `cli` as env var but is NEVER written to `trace.jsonl` (verified by scanning trace file)

**Open questions:**
- **Q5** (`.provider.yaml` survival): Which option is chosen (preserve as-is, redesign as extension capability, hybrid) determines whether `ProviderDef` is a first-class schema or is deprecation-tracked.

---

### Phase 9 — `gert serve` & Adapter Contracts

**Goal:** Implement the `gert serve` daemon: `exec/v2` JSON-RPC methods, `events/v2` WebSocket event stream, RBAC enforcement, TLS/mTLS, and health/ready endpoints. Publish the contract test suite.

**Deliverables:**
- `cmd/gert/serve.go` — `gert serve --stdio | --http :<port>` subcommand
- `internal/server/rpc.go` — JSON-RPC 2.0 dispatcher: `exec/start`, `exec/next`, `exec/cancel`, `exec/status`, `run/list`, `run/get` (§13 method table)
- `internal/server/websocket.go` — `events/v2` WebSocket at `/ws`; subscribe message handling; `sinceSequence` reconnection protocol; best-effort delivery (drop on slow consumer)
- `internal/server/auth.go` — bearer token validation (tokens.yaml); mTLS client certificate CN/SAN extraction; actor identity injection into request context
- `internal/server/rbac.go` — role enforcement: viewer (read-only), operator (exec/approve/cancel), admin (governance config); `POST /approvals` endpoint with Ed25519 signature validation (5-minute clock skew, replay protection via single-use `approvalId`)
- `internal/server/tls.go` — TLS 1.3 (1.2 for compat); mandatory TLS for non-localhost listeners; certificate/key flag validation at startup
- `internal/server/health.go` — `GET /health` (liveness), `GET /ready` (readiness with extension load status), `GET /metrics` (Prometheus, Phase 12 wires real metrics)
- `internal/server/errors.go` — canonical error code registry (1xxx–5xxx from §13 table) as Go constants
- `internal/server/version.go` — `X-Gert-Contract` header negotiation; `protocolVersion` in initialize; unsupported version → HTTP 400 / JSON-RPC `-32001`
- `testdata/contracts/` — contract test YAML fixtures for all 4 surfaces (§13): `exec-v2/`, `events-v2/`, `tool-v2/`, `extension-v2/`
- `cmd/gert/contract.go` — `gert contract test --adapter=<name>` runner
- `specs/contracts/exec-v2.openapi.yaml`, `specs/contracts/events-v2.openapi.yaml` — OpenAPI 3.1 specs

**Spec references:**
- §13: all four contract surfaces, version format, breaking/non-breaking change policy, deprecation policy, error code registry, contract test suite structure
- §07: RBAC roles/permissions table, bearer token and mTLS, approval submission and host verification sequence, WebSocket security
- §15: health/ready endpoint format, Kubernetes probe configuration

**Entry criteria:** Phases 3–8 complete; a runbook can be executed end-to-end in-process.

**Exit criteria:**
- `gert contract test --adapter=builtin` passes 100% for all contract surfaces
- `gert serve --http :8443 --tls-cert ... --tls-key ...` starts and `GET /health` → 200 within 2s
- `exec/start` → `exec/next` loop via JSON-RPC over stdio completes a 3-step fixture runbook
- WebSocket subscriber receives all required events in sequence order; `sinceSequence: 5` replay returns events 6, 7, 8
- Non-localhost listener without TLS cert → startup fails with clear error before binding port
- Viewer-role actor calling `exec/start` → HTTP 403 / JSON-RPC error code 2xxx
- Approval submission with invalid Ed25519 signature → rejected with structured error

**Open questions:**
- **Q6** (JSON-RPC contract versioning) — determines whether v1 JSON-RPC method names are preserved or negotiated via `protocolVersion`; affects `rpc.go` routing layer
- **Q8** (gRPC transport) — decision must be "defer to v2.1" or "add as opt-in alongside stdio" before this phase; gRPC server stub included only if opted in

---

### Phase 10 — Adapters

**Goal:** Implement the Bubble Tea TUI adapter and the VS Code extension TypeScript client, both consuming only the `exec/v2` and `events/v2` contracts.

**Deliverables:**
- `adapter/tui/` — Bubble Tea TUI adapter implementing `Adapter` interface:
  - `adapter/tui/app.go` — `TUIAdapter.Attach(RunHandle)` + `TUIAdapter.Run(ctx)` main loop
  - `adapter/tui/render.go` — step list renderer: pending/running/completed/failed states; progress indicator
  - `adapter/tui/approval.go` — approval gate UI: display roles/message; receive actor decision via keypress or form
  - `adapter/tui/evidence.go` — `choice`/`decision`/`collector` interactive UI; form fields by evidence type
  - `adapter/tui/events.go` — event fan-in: drains `RunHandle.Events()` channel and triggers model updates
- `adapter/vscode/` — VS Code extension TypeScript client (existing `vscode/` extended for v2):
  - `adapter/vscode/src/client.ts` — JSON-RPC 2.0 over stdio client; `exec/start`, `exec/next`, `exec/cancel`, `exec/status`, `run/list`
  - `adapter/vscode/src/events.ts` — WebSocket `events/v2` subscriber; `sinceSequence` reconnect logic
  - `adapter/vscode/src/approve.ts` — approval submission via `POST /approvals`
  - `adapter/vscode/src/evidence.ts` — UI forms for `choice`/`decision`/`collector`
  - Contract conformance: 100% `exec-v2` and `events-v2` contract test pass rate (`gert contract test --adapter=vscode`)

**Spec references:**
- §01 G3 (preserve TUI, VS Code extension), G7 (adapter decoupling — no internal package imports)
- §13: `exec/v2` and `events/v2` stability guarantees
- §02 API/Adapter Layer: `Adapter` interface, no-execution-logic invariant, event channel draining
- §06 Events: all required events that adapters MUST handle

**Entry criteria:** Phase 9 complete; `gert serve --stdio` and `gert serve --http` are operational.

**Exit criteria:**
- TUI renders a 5-step runbook end to end without blocking on any step
- TUI approval gate pauses display and accepts `y`/`n` keypress; decision forwarded via `RunHandle.Approve()`
- VS Code extension passes `gert contract test --adapter=vscode` 100%
- Neither adapter imports any package from `internal/engine/`, `internal/governance/`, or `internal/toolruntime/` (enforced by `no_import` contract test in CI)
- TUI handles closed `Events()` channel (run ended) without panic

**Open questions:** None blocking this phase (Q8 gRPC already resolved or deferred in Phase 9).

---

### Phase 11 — Evidence, Resumption & Replay

**Goal:** Implement run resumption (`gert exec --resume`), scenario-based replay (`--mode replay`), `gert test` scenario runner, trace archive migration, and HMAC trace chaining.

**Deliverables:**
- `internal/runstore/resume.go` — `Runtime.Resume(runID)`: acquire lock, `LoadLatestCheckpoint`, advance `TreePath` past completed steps, re-open trace for append
- `internal/runstore/hmac.go` — HMAC-SHA256 chain: `H_0 = HMAC(key, runID)`; `H_i = HMAC(key, H_{i-1} || E_i)`; `integrityHash` field appended to each trace event when `GERT_TRACE_KEY` set; final hash in `run/completed`
- `internal/replay/replay.go` — `ReplayRuntime`: reads scenario file; matches commands in declaration order (first-match); returns pre-recorded stdout/stderr/exit_code; governance rules evaluated normally; `gert exec --mode replay --scenario <file>`
- `internal/replay/scenario.go` — scenario YAML parser: `commands[]`, `evidence{}`, `allow_unmatched` flag; v1 scenario format forward-compatible
- `internal/replay/collector.go` — `ReplayEvidenceCollector`: returns pre-recorded evidence values; no operator prompt
- `cmd/gert/test.go` — `gert test <runbook.yaml>`: discovers `scenarios/<name>/` directories; runs each scenario; evaluates `test.yaml` assertions (`outcome`, `steps_passed`, `captures.*`, `trace[]` with `data.*` dotted paths)
- `cmd/gert/trace.go` — `gert trace show <run-id>` (pretty-print), `gert trace export <run-id> --format=otlp|jaeger|zipkin`, `gert trace migrate --to-v2 <path>` (§10 v1→v2 envelope rewrite)
- `internal/evidence/hash.go` — `HashFile(path)` → (sha256hex, size); `NewAttachmentEvidence(path)` → `EvidenceValue`
- `testdata/scenarios/` — 7 scenario fixtures covering §08 integration test corpus requirements

**Spec references:**
- §12: all resumption subsections; `gert exec --resume` contract; resumability by step type; partial step resumption; idempotency guidance; HMAC chaining algorithm
- §08: `gert test` feature; scenario format; `test.yaml` assertions; `ConformanceTest`; golden trace comparisons; v1 scenario compat
- §10 Migration: trace archive migration tool; v1→v2 envelope field mappings

**Entry criteria:** Phases 3, 4, 5 complete. `DirRunStore` checkpoint writes operational.

**Exit criteria:**
- `gert exec --resume <run-id>` after a simulated crash (SIGKILL after step 2 of 5): resumes from step 3, trace is continuous (sequence numbers unbroken), `run/completed` emitted
- `gert test testdata/runbooks/branch.runbook.yaml` passes all 3 branch scenario assertions
- HMAC chain: modify one trace event manually; `gert trace verify <run-id>` reports chain broken at correct sequence number
- `gert trace migrate --to-v2 testdata/v1-traces/*.jsonl` rewrites all files with v2 envelope fields; original backed up as `.v1.jsonl`
- `gert test` p50 < 500ms, p99 < 1s for a 10-step replay scenario (§08 perf criteria)
- Tool with `idempotent: false` and orphaned `tool/invoked` (no `tool/responded`) → step treated as failed on resume

**Open questions:** None blocking this phase.

---

### Phase 12 — Observability & Security Hardening

**Goal:** Wire OpenTelemetry spans and Prometheus metrics through all execution layers; implement structured JSON logging; add `gert diagnose`; complete Ed25519 extension signing, tool binary checksums, and `gert verify`.

**Deliverables:**
- `internal/otel/tracer.go` — OTel span hierarchy: `gert.run` (root) → `gert.step` (per step) → `gert.tool.invoke` (per tool call) → `gert.input.resolve` (per provider resolution) → `gert.governance.check` (per policy evaluation); all required attributes from §15 table; W3C `traceparent`/`tracestate` injection into HTTP tool calls and `OTEL_TRACEPARENT` env for stdio tools
- `internal/otel/metrics.go` — Prometheus metrics: all metrics from §15 catalog (runs, steps, tools, approvals, extensions) with correct label sets and histogram bucket boundaries
- `internal/otel/logging.go` — structured JSON logger (using `log/slog`): RFC 3339 timestamp, `level`, `msg`, `run_id`, `step_id`, `actor`, `tool_name`; `GERT_LOG_LEVEL` env var; `--log-output=file:<path>|syslog://` flag; sensitive value redaction applied to all log lines
- `internal/server/metrics.go` — `/metrics` Prometheus exposition endpoint wired to real metrics collectors
- `cmd/gert/diagnose.go` — `gert diagnose [--runbook=<path>]`: 5 checks (schema validation, tool availability, provider connectivity, extension loading, OPA policy syntax); structured output; exit 1 on any failure
- `internal/toolruntime/checksum.go` (complete) — SHA256 binary verification before invocation; `governance/toolChecksumMismatch` trace event
- `internal/exthost/trust.go` (complete) — full Ed25519 verification: public key lookup in `$HOME/.gert/trusted-keys/` or `/etc/gert/trusted-keys/`; canonical JSON reconstruction; `notBefore`/`notAfter` window check
- `cmd/gert/verify.go` — `gert verify --workspace .`: checks all tool checksums + all extension signatures; structured output; exit 1 on any error

**Spec references:**
- §15: complete observability spec (OTel hierarchy, all required attributes, metrics catalog, structured logging format, health endpoints, diagnostic commands, CI integration recipes)
- §07: tool binary checksums; extension signature verification algorithm; `gert verify` command; security recommendations
- §12 OTel integration: `runWithOTel`, `executeStep` span examples, OTLP config

**Entry criteria:** Phases 6, 7, 9 complete; all integration surfaces wired.

**Exit criteria:**
- `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317 gert exec runbook.yaml` produces a `gert.run` root span visible in local Jaeger within 5 seconds of run completion
- `gert serve --http :8080` → `curl /metrics` returns all 12 metric families from the §15 catalog
- `gert diagnose --runbook testdata/runbooks/valid/hello-world.yaml` exits 0 with 5 green check marks
- `gert diagnose --runbook testdata/runbooks/invalid/missing-tool.yaml` exits 1 with structured error identifying the missing tool
- `gert verify --workspace .` with a tool whose SHA256 is wrong → FAILED exit with tool name and path in output
- OTel no-op path: without `OTEL_EXPORTER_OTLP_ENDPOINT` set, `gert exec` has zero overhead from tracer (verified by benchmark showing <1% difference)

**Open questions:** None blocking this phase.

---

### Phase 13 — Migration Tooling

**Goal:** Implement `gert migrate`, the v1 compatibility shim removal path, `gert trace migrate`, and the complete field mapping transformation algorithm.

**Deliverables:**
- `internal/migrate/runbook.go` — v1→v2 runbook migration algorithm (10 transformations in §10 order): `apiVersion` rewrite, `$schema` addition, `id` derivation, `meta.*` promotion, `tree`→`flow`, step type renames, `toolRefs` rewrite, v2 validation pass, rollback on failure
- `internal/migrate/tool.go` — `tool/v0`→`tool/v2` migration: adds `transport`, renames `executable`→`command`, adds `capabilities` stub (marked `# TODO: manual review`)
- `internal/migrate/provider.go` — `provider/v0`→`provider/v2` migration: namespace updates, `resolve_method` addition; partial (manual namespace updates flagged with comments)
- `internal/migrate/trace.go` — `gert trace migrate --to-v2`: `seq`→`sequence`, `type`→`kind`, `data`→`payload`, adds `event_id`/`run_id`/`runbook_id` from context; `.v1.jsonl` backup
- `cmd/gert/migrate.go` — `gert migrate [--to-v2] [--dry-run] [--check] [--output PATH] [--force] [--backup] <path|dir>`: diff report output format from §10; walk directory for bulk migration
- `internal/parser/compat.go` — v1 shim (from Phase 1): `--strict` flag makes deprecation warnings errors; `v2.1` removal target documented with `// TODO(v2.1): remove shim`
- `testdata/migration/` — v1 runbook corpus subset: 20 runbooks exercising all transformation paths (field promotions, step renames, complex governance blocks)

**Spec references:**
- §10: complete migration spec; v1→v2 compatibility matrix; field mapping table; migration algorithm; diff report format; provider/tool migration; extension migration (v2 handshake migration guide); rollout strategy; v1 compat shim behavior and removal timeline

**Entry criteria:** Phase 1 complete (parser handles both v1 and v2); Phase 3 complete (validator used for post-migration validation).

**Exit criteria:**
- `gert migrate --to-v2 testdata/migration/*.yaml` produces valid v2 YAML for all 20 fixtures (verified by `gert validate`)
- `gert migrate --to-v2 --dry-run testdata/migration/simple.yaml` produces diff report with ≥4 change lines and makes no filesystem writes
- `gert migrate --to-v2 --check testdata/migration/already-v2.yaml` exits 0 (no migration needed)
- `gert migrate --to-v2 testdata/migration/custom-meta-field.yaml` exits 1 with structured error identifying unsupported custom field, no partial rewrite
- `gert trace migrate --to-v2 testdata/v1-traces/complex.jsonl`: all events valid against v2 envelope schema; `.v1.jsonl` backup created
- 95% of v1 runbook corpus (§08 regression test) migrates without manual fixup

**Open questions:** None blocking this phase.

---

### Phase 14 — Testing & Acceptance

**Goal:** Assemble the complete integration fixture corpus, golden traces, contract compliance verification, performance benchmarks, and v1 corpus regression gate. This phase proves the system is shippable.

**Deliverables:**
- `testdata/runbooks/integration/` — 7 mandatory integration fixtures (§08 list): linear (all step types), branching (both arms + governance-blocked arm), iterate (list mode), nested invoke (parent + 2 children), retry (transient failure + success), resumption (crash after step 2), governance (allowlist violation + denylist + approval gate)
- `testdata/golden/` — golden JSONL traces for all deterministic scenarios (normalized timestamps and run IDs)
- `testdata/contracts/` — complete contract test suite for all 4 surfaces (finalized in Phase 9; executed here for acceptance)
- `internal/engine/conformance_test.go` — `ConformanceTest(t, Runtime)` (§08 pattern); run against `DefaultRuntime`
- `internal/engine/noimport_test.go` — `TestNoAdapterImports`: uses `go/build` analysis to verify no package in `adapter/` imports `internal/engine/`, `internal/governance/`, or `internal/toolruntime/`
- `internal/bench_test.go` — `testing.B` benchmarks for all 7 perf criteria from §08 table
- `scripts/ci/regression.sh` — v1 corpus regression: `gert test` against all v1 runbook fixtures with golden output comparison; target ≥95% pass rate
- `.github/workflows/pr.yaml` — PR gate (5 checks: unit tests, schema validation, contract tests, linter, vet)
- `.github/workflows/integration.yaml` — integration gate (3 checks: integration tests, regression corpus, golden traces)
- `.github/workflows/nightly.yaml` — nightly gate (3 checks: perf benchmarks vs. baseline, race detector, extension conformance suite)
- `CHANGELOG.md` — v2.0.0 entry with all breaking changes

**Spec references:**
- §08: all acceptance criteria (AC 1–8); integration fixture corpus list; `ConformanceTest`; schema contract tests; event bus determinism tests; performance acceptance criteria table; CI/CD gate requirements
- §01: success definition checklist (5 items) — all must pass

**Entry criteria:** ALL prior phases complete.

**Exit criteria — the v2 release gate (§01 §08 acceptance criteria):**
1. `TestNoAdapterImports` passes — Runtime Core has no direct adapter dependency
2. Extension handshake conformance suite passes for all built-in extensions
3. Capability violation fast-fail tests pass for each of the 10 capability types
4. Each adapter's integration test passes against mock core
5. Each of the 7 fixture runbooks has ≥1 happy-path and ≥1 failure-path end-to-end test
6. v1 corpus regression: ≥95% pass `gert validate` without modification
7. All 7 perf benchmarks meet §08 targets on reference machine (4-core, 8GB, SSD)
8. `gert test` successfully replays all scenarios in `testdata/scenarios/`
9. `gert contract test --adapter=builtin` and `--adapter=vscode` both pass 100%
10. `go test -race ./...` passes (nightly gate)

**Open questions:** By this phase, all open questions from §09 must be resolved and recorded in `.squad/decisions.md`.

---

## 5. Cross-Cutting Concerns

### Security (thread from Phase 0)

Security is not a late phase — it is a constraint on every interface. Key invariants threaded through from Phase 0:
- **Redaction before write**: the `TraceWriter` and `MemRunStore` accept only pre-redacted data. The contract is enforced by the `GovernanceEngine` in the execution loop, not by the store. A store that writes un-redacted data is a bug.
- **Secret type inputs**: the `inputs.type: secret` value is marked sensitive in memory from the moment it is resolved. Passing a sensitive value to any log, trace event, or captured variable without redaction is a bug detected by property-based tests in Phase 4.
- **Capability grants are immutable per run**: the granted capability set for a run is sealed at `Runtime.Start()`. No step or extension can widen it at runtime.
- **Ed25519 signatures for approvals**: even in `--auth-mode bearer` mode, approval submissions require a valid signature. Anonymous self-approval is impossible by construction.
- **Tool binary checksums**: the `--skip-checksum` flag does not exist. Checksum verification is conditional on the `checksum:` field being present — but if present, it is always verified.

### Observability (thread from Phase 3)

OTel tracer is initialized in `cmd/gert/main.go` before the Runtime starts. All components receive a `context.Context` carrying the active span. The tracer is a no-op when `OTEL_EXPORTER_OTLP_ENDPOINT` is unset (zero overhead). Prometheus metrics are registered in an `init()` function in `internal/otel/metrics.go`; counters are incremented via helper functions, not via direct collector access. Structured logging uses `log/slog` with a `run_id` attribute injected into the context at `Runtime.Start()` and propagated through all callsites.

### Testing Strategy

Three test layers per package:
- **White-box unit tests** (`_test.go` co-located): table-driven, fast, no I/O. Target: every exported function, every error branch.
- **Contract tests** (`_contract_test.go`): verify interface invariants. Run on every PR. Any implementation satisfying an interface must pass its contract test.
- **Integration tests** (`_integration_test.go`, build tag `//go:build integration`): spin up real processes. Run on merge to main.

`pkg/testutil/` helpers are available from Phase 0 and are used by all phases. No test may use `time.Sleep` for synchronization — use channels and timeouts.

Golden trace tests use normalized comparison: `run_id`, `event_id`, and all timestamps are replaced with fixed sentinel values before comparison. The `-update` flag regenerates golden files.

### v2.1 Deferment Tracking

The following features are explicitly deferred to v2.1. Each must have a stub that compiles, panics clearly, and is tagged with `// TODO(v2.1)`:
- `parallel` step type
- `compensate` step type and saga event types (`saga/compensation_triggered`, `saga/compensation_completed`)
- `step/retrying` event (retry policy field parses but does nothing)
- OPA `PolicyEngine` interface (built-in YAML policy engine is the only implementation)
- gRPC transport for VS Code adapter
- v1 compatibility shim removal (shim present in v2.0, removed in v2.1)
- Explicit `sensitive: true` field marking on tool outputs

---

## 6. Risk Register

| # | Risk | Likelihood | Impact | Mitigation |
|---|------|-----------|--------|-----------|
| R1 | **Q2 (concurrency model) left unresolved through Phase 3 start** — the engine's `RunHandle`, event bus, and `RunStore` are architected differently for single-run-per-process vs. concurrent runs. A wrong choice here requires rewriting the engine core. | High | Critical | Q2 must be resolved and recorded in `.squad/decisions.md` before any code in Phase 3 is written. The preferred default is option (b) — multiple concurrent runs in-process — because `gert serve` requires this for concurrent VS Code panels. If unresolvable in 3 days, default to option (a) and accept a Phase 3 rework cost for serve mode. |
| R2 | **14 step types is a large surface area for Phase 5** — at this count, a single phase implementing all step types is a significant blast radius if a shared executor contract assumption is wrong. | High | High | Split Phase 5 into two sub-phases if necessary: (5a) execution steps (`cli`, `tool`) + terminal (`end`), then (5b) interactive, control flow, and governance steps. The sub-phase split is at the team's discretion; both must exit-criteria-verify before Phase 6 begins. |
| R3 | **`gert migrate` cannot achieve 95% lossless migration of the v1 corpus** — custom `meta.*` fields, provider field namespacing, and non-standard step patterns in production runbooks may require manual intervention at rates above 5%. | Medium | High | Run `gert migrate --dry-run` against a sample of production runbooks from representative deployments before Phase 13 begins. If manual fixup rate >5%, file an issue and adjust the migration algorithm before declaring Phase 13 done. |
| R4 | **Extension crash isolation not portable** — seccomp (Linux) and pledge/unveil (OpenBSD) are specified in §07 for filesystem/network sandboxing. macOS and Windows have no direct equivalents. Host-side argument validation is the fallback, but it is less robust. | Medium | Medium | Accept host-side path/URL validation as the v2.0 implementation on macOS and Windows. Document the limitation. File a v2.1 issue for macOS sandbox-exec integration. Never advertise seccomp as a cross-platform feature. |
| R5 | **VS Code extension gRPC transport decision (Q8) delays Phase 10** — if Q8 is unresolved, the adapter design is unclear (stdio JSON-RPC vs. gRPC vs. HTTP/WebSocket). | Medium | Medium | Default to Q8 option (a): retain JSON-RPC stdio for v2.0. gRPC is v2.1. This is already the spec's preferred option and unblocks Phase 10 immediately. |
| R6 | **OTel SDK adds measurable latency to the execution loop** — the spec requires OTel for every step span. If the SDK's synchronous span creation adds >5ms to step startup latency, the `<50ms` p99 criterion (§08 perf table) is at risk. | Low | Medium | Benchmark the OTel SDK in Phase 12 against the Phase 3 baseline. If overhead exceeds 2ms p50, switch to the OTel async batch processor with a larger export interval. Use no-op sampler in benchmark runs. |
| R7 | **JSON Schema Draft 2020-12 Go library maturity** — `santhosh-tekuri/jsonschema/v6` is the leading Go library but is still relatively young. Schema edge cases (dynamic `$ref`, unevaluated properties) may behave differently than `ajv` (the reference implementation). | Low | Medium | Write a schema test harness that cross-validates all `testdata/` fixtures against both the Go library and a reference `ajv` run (Node.js, CI-only). Any discrepancy blocks Phase 1 exit. If the library is inadequate, evaluate `sourcemeta/jsonschema` or a custom validator for the bounded v2 schema surface. |
| R8 | **v1 trace format consumers may be broken by Q3 decision** — operators with SIEM integrations or compliance dashboards built against the v1 `seq`/`type`/`data` field names face a breaking change regardless of which Q3 option is chosen (new format or superset). | Medium | Medium | Q3 option (b) — backward-compatible superset — is strongly preferred: emit both `seq` and `sequence`, both `type` and `kind`, both `data` and `payload` for the first v2.1 release window, then deprecate the v1 names. This mitigates migration friction at the cost of trace line size. Decision must be recorded before Phase 3 begins. |

---

*This plan is Barbara's authoritative build roadmap. Agents picking up a phase should read the corresponding spec sections before writing code. Open questions that are phase blockers must be resolved in `.squad/decisions.md` before work begins.*
