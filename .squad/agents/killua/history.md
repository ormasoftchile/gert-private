# Project Context

- **Owner:** ormasoftchile
- **Project:** gert runbook system and tooling
- **Stack:** Go, TypeScript, C#, YAML, Azure
- **Created:** 2026-03-17

## Learnings

- **Invoke state propagation (2026-03-21):** The Go engine (`executeInvokeStep`) and serve layer correctly propagate invoke-child step state using `invokeChild: true` + `parentStepId` on events. The `snapshotStateMachine.ts` correctly stores these with fully-qualified keys (`parentStepId::childStepId`). However, the detail pane in `runbookPanel.ts` strips the `::` prefix when looking up state (`this.stepStates.get(bareId)`), causing child steps to show "pending" even when completed. The graph renderer works correctly because `treeOps.ts` merges child trees with FQ-prefixed IDs. Key insight: any UI code that resolves invoke-child state must use the FQ key from `viewingStepId`, not the stripped base ID.
- Primary backend focus: runbook engine, schema handling, and extensible providers/tools.
- Visual editor can stay YAML-canonical by reusing strict/flexible loaders in pkg/schema (Load/decodeRunbookFlexible) and project-aware compat resolvers (ResolveToolPathCompat, ResolveRunbookPathCompat).
- Existing serve JSON-RPC already exposes execution/session surfaces; adding read-only metadata endpoints there is the lowest-risk path to drive UI generation without touching engine execution semantics.
- Tool/provider extensibility is already alias- and capability-driven (ToolDefinition, ProviderDefinition->ToolDefinition, ToolManager); editor contracts should consume these registries instead of transport-specific hardcoding.
- Cross-agent contract expectation: backend metadata/diagnostic APIs should directly power Kurapika's form model, preserve Gon's YAML-canonical MVP boundary, and produce machine-checkable outputs for Hisoka's reviewer gates.
- **Editor MVP Compatibility Review (2026-03-17)**: Reviewed runbookEditorPanel.ts for schema/runtime compatibility. Found critical issues: simple linear step extraction + rebuild path destroys branches/iterates, ignores complex step types (manual/assert), discards all tool configuration, loses YAML formatting/comments. Implemented safety gates: detect non-linear structures at render time, block tree editing with warnings when unsafe, preserve metadata-only editing regardless of complexity. Tree corruption prevented by structural validation on save.
- Runbook schema constraints: top-level tree nodes enforce step XOR iterate (exclusive), branches require conditions referencing previous outputs, iterate blocks have convergence (max+until) vs list (over+as) modes, steps support 9+ types. Editor MVP scope: safe only for simple linear tool-step trees. Complex/branched/iterated runbooks gracefully degrade to metadata editing with clear UI warnings.
- Tree safety pattern: check structure at render-time (not save-time), fail early with visible blockers, always allow metadata editing, validate rebuild hasn't corrupted required fields before fs.write().

## Schema Clarification (2026-03-18)

**Task**: Clarify TreeNode structure misalignment between TypeScript form and Go schema for Hisoka's MVP rejection fix.

**Findings**:
- **TreeNode structure** (Go authoritative): `{step?, iterate?, branches?}` — exactly one of step/iterate must be non-zero, branches are optional on any node.
- **Branch vs Step**: Branches are a **structural container** on TreeNode (not a step type). Step types are: tool, manual, assert, end, extension, cli, invoke. ~~branch~~ and ~~parallel~~ are artifacts—do NOT use.
- **Field locations**:
  - Branch conditions: on `Branch.condition` (required).
  - Iterate config (max, until, over, as): on `IterateBlock`, NOT on steps inside.
  - Tool config (name, action, args, capture): on `Step.Tool` (ToolStepConfig), only when type=tool.
  - Governance (rules, redaction, deny env vars): on global `Meta.Governance`, not per-step (except Step.Approvals for manual).
- **Navigation paths**: Use `[n, 'field', m, 'subfield', ...]` notation with 0-based indices and key access.
- **Read-only fields**: APIVersion, (mostly) Step.ID, Meta.Source. Governance rules shadow editable fields at runtime.

**Outcome**: Produced SCHEMA_CLARIFICATION.md (authoritative Go-backed doc) with TreeNode spec, complete YAML examples (branches + iterates), path notation, field locations, and validation rules. Ready for Kurapika's form redesign.

## Execution Path Events (2026-03-18)

**Task**: Add `event/branchResolved` and `event/iteratePassEnd` to the Go serve layer for frontend execution viewer annotation.

**Changes**:
- **`event/branchResolved`** — emitted once per branch condition evaluation at all 3 branch evaluation sites in `ext/serve/pkg/serve/serve.go`:
  1. `handleTreeNext` pending manual phase2 (line ~928)
  2. `executeTreeStep` branch-type routing nodes (line ~1330)
  3. `executeTreeStep` regular step post-outcome branch evaluation (line ~1474)
  - Payload: `{parentStepId, branchIndex, condition, taken}`
  - Emits for each branch evaluated (stops after first `taken=true`, matching existing `break` semantics).

- **`event/iteratePassEnd`** — emitted at the end of every iterate pass (before convergence/advance decision) at both watchpoint handlers:
  1. Convergence-mode watchpoint handler (line ~975)
  2. List-mode over-watchpoint handler (line ~1019)
  - Payload: `{iterateStepId, pass (1-based), max, converged}`
  - Uses synthetic `nodeID` (format `iterate-<stepIdx>`) threaded through watchpoint structs for correlation.

- **Watchpoint struct changes**: Added `nodeID string` field to `iterateWatchpoint` and `iterateOverWatchpoint`. Updated `pushIteratePass` and `pushIterateOverPass` signatures to accept `nodeID`.

- **VS Code extension wiring**:
  - Added `BranchResolvedEvent` and `IteratePassEndEvent` interfaces to `vscode/src/serve/client.ts`.
  - Added `event/branchResolved` and `event/iteratePassEnd` case handlers to `runbookPanel.ts` `handleEvent` switch, following existing event dispatch pattern (console.log + updateWebview). No SVG/render changes.

**Patterns followed**: Existing `sendEvent` notification pattern (zero-cost JSON-RPC when no listener). All events additive — no changes to existing event shapes, engine behavior, or execution semantics.

## Editor Backend APIs: Tool Catalog, Dry-Run, Schema Fields (2026-03-18)

**Task**: Add six new JSON-RPC endpoints to `ext/serve/pkg/serve/serve.go` and wire them into `vscode/src/serve/client.ts` for the visual editor's form generation and preview features.

**Endpoints added (Go)**:

1. **`tools/list`** — Discovers project root via `schema.DiscoverProject`, scans the `tools/` directory for `*.tool.yaml` files, loads each with `schema.LoadToolFile`, returns array of `{name, version, description, actions}` with arg schemas per action. Accepts optional `cwd` param.

2. **`tools/get`** — Takes `{name, cwd?}`, resolves via `project.ResolveToolRef`, returns the full `ToolDefinition` as JSON (apiVersion, meta, transport, governance, actions with full arg/capture/governance detail).

3. **`exec/dryRun`** — Takes same params as `exec/start`. Runs `schema.ValidateFile` for 3-phase validation, flattens tree for step count, collects `tools:` as tool dependencies, scans `meta.governance.rules` for approval/deny rules and `deny_env_vars`. Returns `{valid, errors[], tree, stepCount, toolDeps[], governanceWarnings[]}`. Does NOT create an engine or execute anything.

4. **`schema/stepFields`** — Returns hardcoded field definitions per step type: `tool`, `manual`, `assert`, `end`, `extension`, `cli`, `invoke`. Each includes the relevant YAML fields for that step type. No params required.

5. **`schema/toolArgs`** — Takes `{tool, action, cwd?}`, resolves tool via project, loads definition, returns `{args: {name: {type, required, description?, default?, enum?, redact?}}}` for the specified action.

## Per-step Retry, Parallel Iterate, Error Branches (2026-03-18)

**Task**: Add three engine features — per-step retry with backoff, parallel iterate (concurrency), and `_error` branches.

**Changes**:

### Task 1: Per-step retry with backoff
- **Schema**: Added `RetryConfig` struct (`max`, `interval`, `backoff`) and `Retry *RetryConfig` field on `Step` in `pkg/schema/schema.go`.
- **Engine**: New `executeStepWithRetry()` method in `pkg/engine/engine.go`. Wraps `executeStep` with retry loop: parses interval, applies linear or exponential backoff, emits `event/stepRetrying` with `{stepId, attempt, maxAttempts, nextInterval}`. Respects context cancellation during wait.
- **Validation**: `pkg/schema/validate.go` validates retry config: max >= 0, backoff must be "linear" or "exponential", interval must be parseable as Go duration.
- **JSON Schema**: Auto-generated via `scripts/gen-schema.go` — `RetryConfig` def and `retry` ref on Step.

### Task 2: Parallel iterate (concurrency)
- **Schema**: Added `Concurrency int` to `IterateBlock` in `pkg/schema/schema.go` (minimum=1 in jsonschema tag).
- **Engine**: Implemented full iterate execution in `runTree` — was completely missing before (tests were failing).
  - `runIterate()` dispatches to convergence or list mode.
  - `runIterateConvergence()` — max+until loop with outcome-aware early termination.
  - `runIterateList()` — sequential over/as iteration. Delegates to parallel when `concurrency > 1`.
  - `runIterateListParallel()` — bounded concurrency via semaphore channel. Each goroutine gets a scoped engine clone (`cloneForIteration`) with its own vars/captures copy. Captures merge back in declaration order (deterministic).
  - `iterateHasApproval()` — recursive check forces concurrency=1 when any step has approval requirements.
  - Convergence mode (`until`) stays sequential by design (order-dependent).
  - Emits `event/iterateParallel`, `event/iteratePassStart`, `event/iteratePassEnd` for UI.
- **JSON Schema**: Auto-generated — `concurrency` on IterateBlock with minimum=1.

### Task 3: Error branches
- **Schema**: `condition: "_error"` is a reserved branch condition. No new struct needed — uses existing `Branch.Condition`.
- **Engine**: After step failure in `runTree`, checks for `_error` branch via `findErrorBranch()`. If found, sets `{{ .error }}` variable with the failure message and executes the error branch. `continue_on_fail` still applies if no `_error` branch exists. Normal (success) branch evaluation skips `_error` branches.
- **Validation**: Warns if `_error` branch exists without non-error condition branches. Errors if `_error` is used on iterate blocks.

**Tests**: All pre-existing iterate tests now pass (9/9 — were all failing before). Pre-existing `TestStepDelayCancellation` and testdata-path tool tests remain failing (unrelated).

**Build**: `go build -o gert.exe ./cmd/gert/` succeeds. Schemas regenerated via `go run scripts/gen-schema.go`.

**TypeScript client wiring**:
- Added interfaces: `ToolArgInfo`, `ToolActionInfo`, `ToolInfo`, `ToolDefinition`, `DryRunResult`.
- Added methods: `toolsList(cwd?)`, `toolsGet(name, cwd?)`, `execDryRun(params)`, `schemaStepFields()`, `schemaToolArgs(tool, action, cwd?)`.

**Compile verification**: Both `go build` (exit 0) and `npm run compile` + `tsc --noEmit` (exit 0) pass clean.

**Patterns followed**: All endpoints are read-only/stateless — they do not create engines, modify server state, or call `saveSession()`. Tool discovery reuses existing `schema.DiscoverProject + FallbackProject` and `project.ResolveToolRef` paths. Same dispatch/switch pattern as existing methods.

## Tool Catalog, Schema Bundle & Dry-Run Enhancements (2026-03-18)

**Task**: Refine tool catalog API, add schema bundle endpoint, and enrich dry-run response for visual editor form generation.

**Changes**:

1. **`tools/list` response format fix** — Changed `actions` from keyed map to array with `name` field per element: `[{name, description, args}]`. Matches the task spec and is easier for frontend dropdown rendering.

2. **`tools/detail` dispatch** — Added as alias to `handleToolsGet` (same handler). Both `tools/get` and `tools/detail` now work for requesting full single-tool definitions.

3. **`schema/bundle` endpoint (new)** — Returns `{runbook, tool, project}` where each is a generated JSON Schema from Go structs via `schema.GenerateJSONSchema()`, `schema.GenerateToolJSONSchema()`, and new `schema.GenerateProjectJSONSchema()`. No params required.

4. **`GenerateProjectJSONSchema()` (new in `pkg/schema/export.go`)** — Generates Draft 2020-12 JSON Schema from the `Project` struct, matching the pattern of the existing runbook/tool schema generators.

5. **`exec/dryRun` response enrichment** — Added two new fields:
   - `steps`: flat array of `{id, type, title, dependencies}` per step. Dependencies inferred from `when` expressions and tool arg template references containing `.steps.<id>` patterns.
   - `warnings`: array of non-error validation findings (`{phase, path, message}`), separated from `errors`.

6. **`extractDependencies` helper** — Scans step `when` expressions and tool arg values for `.steps.<id>` references to build the dependency graph.

**TypeScript client updates**:
- `ToolActionInfo` now includes `name: string` field.
- `ToolInfo.actions` changed from `Record<string, ToolActionInfo>` to `ToolActionInfo[]`.
- Added `DryRunStep` interface, `SchemaBundle` interface.
- `DryRunResult` extended with `steps: DryRunStep[]` and `warnings` array.
- Added methods: `schemaBundle()`, `toolsDetail(name, cwd?)`.

**Compile verification**: Both `go build -o gert.exe ./cmd/gert/` (exit 0) and `tsc --noEmit` (exit 0) pass clean.

## event/invokeStarted with Child Tree Data (2026-03-18)

**Task**: Enrich the `event/invokeStarted` event with child runbook tree data, and add `parentStepId` to all child step events.

**Changes (Go — `ext/serve/pkg/serve/serve.go`)**:
- **`enterInvoke`**: Moved `event/invokeStarted` emission to after child runbook load/validation. Enriched payload: `{parentStepId, childRunbook, childTree, childSteps}` — `childTree` uses `resolveTreeForDisplay()` (same format as `exec/start` tree), `childSteps` uses `buildStepSummaries(flattenTreeSteps())` (same flat format as `exec/start` steps).
- **`currentInvokeParentStepID()` helper**: Returns the `invokeStepID` from the top of the invoke stack. Used by all child event emission sites.
- **All 7 `invokeChild: true` sites**: Added `parentStepId` field alongside `invokeChild` — covers `event/stepStarted` (tree step start), `event/stepCompleted` (5 paths: manual success, manual error, executeTreeStep error, executeTreeStep success, outcome-in-invoke x2).

**Changes (TypeScript)**:
- **`vscode/src/serve/client.ts`**: Added `InvokeStartedEvent` interface with `parentStepId`, `childRunbook`, `childTree`, `childSteps`.
- **`vscode/src/views/runbookPanel.ts`**: Import `InvokeStartedEvent`. Existing `event/invokeStarted` handler at line 387 already consumes the new payload shape (stores in `invokeChildren` map, initializes child step states).

**Compile verification**: `go build -o gert.exe ./cmd/gert/` (exit 0) and `npm run compile` (0 errors, 0 warnings) pass clean.

## Step Delay User Feedback (2026-03-20)

**Task**: Add console output and structured event emission to `applyStepDelay` in `pkg/engine/engine.go`.

**Problem**: `applyStepDelay` silently slept with zero user feedback — no console output, no event for VS Code extension or JSONL trace.

**Changes**:
- **`pkg/engine/engine.go`** (`applyStepDelay`): Added `fmt.Printf("  ⏳ Waiting %s...\n", delay)` and `e.emit("event/stepDelaying", ...)` with `{stepId, delay}` payload, placed before the timer wait. Follows `stepRetrying` pattern exactly.
- **`pkg/engine/delay_example_test.go`**: Extended `TestExampleSingleStepTimeoutRunbookAppliesDelay` to register an `OnEvent` listener, assert exactly 1 `event/stepDelaying` event fires with correct `stepId` and `delay` values.

**Pattern**: Engine feedback events follow `fmt.Printf` + `e.emit()` paired pattern (console for humans, structured event for tooling). Event names use `event/step<Verb>ing` convention.

## Serve-Layer stepDelaying Event (2026-03-20)

**Task**: Wire `event/stepDelaying` through the JSON-RPC serve layer so VS Code extension receives it.

**Problem**: The engine's `e.emit("event/stepDelaying")` uses the engine's `OnEvent` callback, which the serve layer never wires up. The serve layer has its own event system via `s.sendEvent()` (JSON-RPC notifications). Engine events go nowhere when running via VS Code.

**Changes**:
- **`ext/serve/pkg/serve/serve.go`**: Added `event/stepDelaying` emission via `s.sendEvent` in both execution paths:
  1. **Flat-mode** (`handleExecNext`): After `event/stepStarted`, before `s.engine.ExecuteStep()`. Checks `step.Delay`, parses duration, emits if valid and > 0.
  2. **Tree-mode** (`executeTreeStep`): After the branch-type block, before `s.engine.ExecuteTreeStep()`. Same logic.
- Payload: `{stepId, delay}` — matches engine event shape.

**Pattern learned**: Engine events (`e.emit`) and serve events (`s.sendEvent`) are **independent systems**. Any engine event that needs to reach the VS Code extension must ALSO be emitted in the serve layer via `s.sendEvent()` at the corresponding execution point. The serve layer is the JSON-RPC bridge — engine callbacks are not connected in serve mode.

## HTTP Transport for Web Clients (2026-04-05)

**Task**: Add `gert serve --http` to expose the JSON-RPC API over HTTP + WebSocket for browser clients.

**Implementation**:
- **New transport layer** (`serve_http.go`): Wraps existing `Server` with HTTP/WebSocket capabilities. Zero duplication of business logic — purely a transport adapter.
- **Dual-mode operation**: `gert serve` (stdio, unchanged) vs `gert serve --http --port N` (HTTP). Flag-controlled via cobra command.
- **Three endpoints**:
  1. `POST /rpc` — Synchronous JSON-RPC requests. Temporarily swaps `Server.writer` with `responseWriter` to capture responses, distinguishes responses (has `id`) from events (has `method`).
  2. `GET /ws` — WebSocket for event streaming. All connected clients receive broadcasts via `broadcastWriter`.
  3. `GET /health` — Simple health check (`{"status":"ok"}`).
- **Response routing strategy**: Messages with `id` (responses) go to HTTP body, messages with `method` (events) go to WebSocket broadcast. Avoids dual-delivery of events.
- **CORS**: Middleware allows all `localhost` and `127.0.0.1` origins (any port) for local dev servers.
- **Graceful shutdown**: Signal handler (SIGINT/SIGTERM) with 5-second timeout for in-flight requests.
- **Dependency**: Added `github.com/gorilla/websocket@v1.5.1` (de facto standard, stable).

**Key patterns**:
- **Writer swapping**: The serve layer uses `Server.writer` as the output channel. HTTP mode temporarily replaces it per-request to capture responses, then restores. Broadcast mode uses a persistent `broadcastWriter` that multicasts to WebSocket clients.
- **No struct changes to Server**: Existing `Server` struct untouched — HTTP layer is an external wrapper (`HTTPServer`) that composes `Server`.
- **Event vs Response distinction**: Parse outgoing JSON to check for `id` (response) or `method` (event). Responses return to HTTP caller, events broadcast to WebSocket.

**Testing**:
- Build: `go build ./...` passes clean (no compilation errors).
- Smoke: Server starts on `:7777`, `/health` returns 200 + `{"status":"ok"}`.
- JSON-RPC: `POST /rpc` with `schema/stepFields` returns correct multi-step field definitions.
- Flags: `gert serve --help` shows `--http` and `--port` with correct defaults.

**Documentation**:
- `web/README.md`: Quickstart, endpoint reference (HTTP + WebSocket), example curl/JS code, CORS notes, logging/shutdown.
- Decision doc: `.squad/decisions/inbox/killua-http-transport.md` with context, architecture, alternatives, impact.

**Outcome**: Web clients can now connect to `http://localhost:7777` and use the same JSON-RPC API as the VS Code extension. Stdio mode remains unchanged (backward compatible). Ready for frontend integration.

## WebSocket Event Contract Audit (2026-04-05)

**Task**: Audit and extend WebSocket event broadcasting for RunbookPanel execution viewer (Phase 2 of HTTP transport).

**Audit Results**:
- **18 distinct events** emitted by serve layer via `s.sendEvent()`: stepStarted, stepCompleted, stepSkipped, stepDelaying, branchResolved, invokeStarted, invokeCompleted, iterateStarted/Pass/PassStart/PassEnd/Converged/Failed, inputRequired, outcomeReached, runCompleted, runRecovered, runbook/staleSource.
- **All events automatically broadcast** via `broadcastWriter` (io.Writer implementation set as `Server.writer` in HTTP mode).
- **Zero gaps found** — every event the VS Code extension (`eventHandlers.ts`) expects is already being broadcast.

**Architecture Pattern**:
- **Writer-based broadcasting** — `broadcastWriter.Write()` intercepts all `s.send()` output and multicasts to WebSocket clients.
- **Zero event-specific code** — the io.Writer abstraction means all future events automatically broadcast without maintenance.
- **Transport-agnostic events** — same JSON-RPC notification shapes in stdio (VS Code) and HTTP (web) modes.
- **Response vs Event routing** — messages with `id` return to HTTP caller, messages with `method` broadcast to WebSocket.

**Deliverables**:
1. **`web/docs/ws-events.md`** — Complete event contract with 18 event types, full JSON payloads, field descriptions, sequence diagram, usage examples, ordering guarantees. Authoritative contract for Illumi (frontend) and Knov (test).
2. **`.squad/decisions/inbox/killua-ws-events.md`** — Decision doc with audit findings, coverage analysis, architecture benefits, no-code-change conclusion.

**Event Categories Documented**:
- Run Lifecycle (runCompleted, runRecovered)
- Step Lifecycle (stepStarted, stepCompleted, stepSkipped, stepDelaying)
- Branch & Conditions (branchResolved, outcomeReached)
- Invoke/Sub-Runbooks (invokeStarted, invokeCompleted with childTree/childSteps)
- Iterate/Loops (iterateStarted, iteratePass, iteratePassStart/End, iterateConverged/Failed)
- Interactive (inputRequired for manual steps)
- Metadata (runbook/staleSource)

**Key Learning**: The Phase 1 HTTP transport already delivered full event coverage via the `broadcastWriter` pattern. No additional implementation was needed — only documentation of the existing contract. The io.Writer abstraction future-proofs the system: any new event added to `serve.go` will automatically broadcast to WebSocket clients without touching `serve_http.go`.

**Pattern Reinforced**: Writer-swapping (`Server.writer = broadcastWriter` in HTTP mode, stdio in VS Code mode) cleanly separates transport (serve_http.go) from business logic (serve.go). Both transports emit identical JSON-RPC events — single source of truth.
