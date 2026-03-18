# Research: n8n and Azure Logic Apps Designer Features Applicable to Gert

**Author:** Gon (Lead Architect)  
**Date:** 2026-03-18  
**Requested by:** Cristián Ormazábal Ortega  
**Status:** Research — no code changes  

---

## Executive Summary

Gert is a YAML-driven runbook orchestration system with a VS Code visual editor (form + graph authoring, live execution viewer). This analysis evaluates 17 features from n8n and Azure Logic Apps Designer for applicability to gert's roadmap. Of these, 7 are recommended for adoption/adaptation at high priority, 6 at medium, and 4 can be skipped or deferred indefinitely.

Gert's key differentiators — YAML as truth, governance-first execution, tool/provider abstraction, and VS Code-native UX — mean that many features from these platforms need adaptation rather than direct transplant. The biggest gaps are in **discoverability** (tool catalog/search), **expression authoring** (autocomplete for template variables), **per-step I/O inspection** (richer trace display), and **concurrency** (parallel iteration).

---

## Part 1: n8n Feature Evaluation

### 1. Node Marketplace / Catalog

**What they do:** n8n offers 400+ pre-built integration nodes (Slack, GitHub, Postgres, etc.) browsable in a searchable catalog panel. Users drag nodes from the catalog onto the canvas. Community nodes can be installed from npm.

**Gert equivalent:** Tool definitions are YAML files in `tools/` or referenced via `gert.yaml` paths. `tools/list` serve endpoint returns available tools. Runbooks reference tools by name. No browse UI, no install-from-URL, no community registry.

**Gap:** No visual tool browser in the editor. No mechanism to discover or install tools beyond manually placing `.tool.yaml` files. No package registry for sharing tool packs across projects.

**Recommendation:** **Adapt — HIGH priority**  
Gert should NOT replicate n8n's monolithic node ecosystem. Instead:
- **Tool catalog panel** in the editor: searchable list populated from `tools/list` endpoint, showing name, description, available actions. Clicking a tool inserts a pre-filled step template.
- **Tool packs**: bundled collections of related tools (e.g., `azure-tools`, `observability-tools`) distributed as git repos or archives. `gert.yaml` already has `paths.tools` — extend to support `tool_packs: [{name, source, version}]` with source being a local path, git URL, or archive URL.
- **Install-from-URL**: `gert tool install <url>` CLI command that fetches a tool YAML or tool pack into the project's tools directory. No central registry needed initially — URL-based is sufficient and avoids registry maintenance burden.
- **Community contrib later**: A GitHub-based catalog (like Homebrew taps) where users discover tools. Not needed for MVP.

**Priority:** HIGH — tool discoverability is the single biggest friction point for new users.

---

### 2. Credential Management

**What they do:** n8n has a built-in encrypted credential store. Each node type declares its credential requirements (API key, OAuth2, etc.). Users configure credentials once and reuse them across workflows. Credentials are never exposed in workflow JSON.

**Gert equivalent:** Provider system resolves credentials via environment variables, JSON-RPC provider config, or MCP server config. Governance layer can deny specific env vars and redact outputs. No dedicated credential abstraction — credentials are ambient (env vars) or provider-configured.

**Gap:** No first-class credential objects. No credential sharing across runbooks. No encrypted credential storage. Credentials leak into env var space without explicit scoping.

**Recommendation:** **Adapt — MEDIUM priority**  
A full credential store is over-engineering for gert's use case (ops runbooks, not SaaS integrations). Instead:
- **Credential references in tool definitions**: Tools can declare `credentials: [{name: "api_key", env: "MYSERVICE_API_KEY", description: "..."}]`. This documents what a tool needs without storing values.
- **Governance integration**: The existing `deny_env_vars` and redaction system already controls credential exposure. Extend governance to validate that required credential env vars are present before execution (preflight check, surfaced in `exec/dryRun`).
- **VS Code secret storage**: For the editor experience, use VS Code's `SecretStorage` API to store credentials locally. Not in YAML, not in git. Injected at runtime.
- **Skip central credential store**: Gert runs in operator environments (laptops, CI, Azure VMs) where credential management is already handled by the platform (Key Vault, env injection, managed identity). A built-in store would duplicate what already exists.

**Priority:** MEDIUM — credential documentation in tool defs is quick; VS Code secret storage integration is non-trivial but valuable.

---

### 3. Expression Editor with Autocomplete

**What they do:** n8n provides an inline expression editor with autocomplete for `$json`, `$input`, `$node["name"].json`, previous node data references, and JavaScript functions. Shows live preview of expression results against pinned test data.

**Gert equivalent:** Go template expressions `{{ .varname }}` used in tool args, branch conditions, iterate `until` conditions. The structured branch condition builder (implemented in Phase D) provides dropdown-based expression building for common patterns. `getAvailableVariables()` collects vars, inputs, and captures. No general-purpose autocomplete for expressions elsewhere.

**Gap:** No autocomplete for template expressions in tool args, iterate conditions, or step instructions. No live preview of expression results. Variables from upstream steps are not discoverable except in the condition builder.

**Recommendation:** **Adopt — HIGH priority**  
This is where gert's UX becomes significantly better:
- **Template expression autocomplete in all text fields**: When the user types `{{ .`, show a dropdown of available variables (from `getAvailableVariables()` scoped to the current step position). Extend to tool arg fields, iterate `until`, step instructions — not just branch conditions.
- **Scope-aware variable list**: Only show variables that are defined or captured *before* the current step in the tree walk. The condition builder already collects all variables; add ordering awareness.
- **Live preview**: When editing an expression, show the evaluated result using `meta.vars` defaults and any pinned/scenario test data. This requires the Go backend's template engine or a lightweight JS-side template evaluator.
- **Go template function reference**: Show available Sprig functions (contains, eq, regexMatch, etc.) in autocomplete with signatures and examples.

**Priority:** HIGH — expression errors are the #1 source of runbook bugs. Autocomplete prevents them.

---

### 4. Error Handling Per Node

**What they do:** n8n allows per-node configuration: retry count, retry interval, continue-on-error, and an explicit error output that can connect to a separate error-handling branch.

**Gert equivalent:** Steps support `continue_on_fail: true` (skip failures). Gate steps have `on_error` behavior. Governance has approval gates. No retry mechanism. No error-specific branch routing.

**Gap:** No retry capability. No per-step error branches (distinct from conditional branches). No configurable retry intervals or backoff.

**Recommendation:** **Adapt — HIGH priority**  
Retry is critical for operational runbooks where transient failures (DNS timeouts, API rate limits) are expected:
- **Retry configuration per step**: `retry: {max: 3, interval: 5s, backoff: exponential}`. Belongs in the step schema. Engine implemention is straightforward — wrap step execution in a retry loop.
- **Error branches**: A special branch condition `_error` that activates when the step fails (after retries exhausted). This is distinct from data-conditional branches and would be a new branch evaluation type in the engine. Syntax: `branches: [{condition: "_error", label: "Step failed", steps: [...]}]`.
- **Skip n8n's error output wiring**: n8n's explicit error connection adds visual noise. Gert's tree model naturally handles error branches as a branch condition — no new visual construct needed.

**Priority:** HIGH — operational runbooks without retries are incomplete.

---

### 5. Webhook Triggers

**What they do:** n8n workflows can be triggered by incoming webhooks, cron schedules, file watchers, and event queues. The trigger node is the workflow's entry point.

**Gert equivalent:** Execution is always imperative: `gert run <runbook>` from CLI, or "Run" button in VS Code. No trigger/event-driven execution. The `ext/serve` layer handles JSON-RPC sessions but is not an event listener.

**Gap:** No event-driven execution. No webhook receiver. No scheduled execution.

**Recommendation:** **Skip for now — LOW priority**  
Gert is designed for operator-initiated incident response and operational procedures, not for event-driven automation. Adding triggers would:
- Require a persistent daemon/server (gert is currently short-lived CLI + serve sessions)
- Overlap with existing orchestrators (Azure Logic Apps, n8n itself, cron, Azure Functions)
- Dilute gert's focus on governed, auditable, human-initiated runbooks

If triggers become needed, the right approach is an **external trigger bridge** — a lightweight adapter that receives webhooks/events and calls `gert run` with parameters. This keeps gert's runtime model clean.

**Priority:** LOW — external orchestrators handle this. Gert should focus on the execution experience.

---

### 6. Pinned Data / Test Data

**What they do:** n8n lets users "pin" sample data on any node. When testing, pinned data replaces live execution. Users can manually enter test data or capture it from a real run.

**Gert equivalent:** Replay system (`pkg/replay`) with scenario files containing pre-recorded command outputs. `StepScenario` provides per-step replay data. `exec/dryRun` validates without executing. Scenarios are external files, not inline on steps.

**Gap:** No per-step inline test data pinning. No "capture from real run → pin for testing" workflow. Scenario files are separate artifacts, not visible in the editor.

**Recommendation:** **Adapt — MEDIUM priority**  
Gert's replay system is already more sophisticated than n8n's pinned data (full scenario files with command matching). The gap is discoverability and editor integration:
- **Scenario browser in editor**: Show available scenarios for the current runbook, let users select one for dry-run preview. Display per-step expected outputs inline in the form.
- **Capture-to-scenario**: After a real run, offer to save the trace as a scenario file. The trace already contains all step inputs/outputs — transform `trace.jsonl` → scenario format.
- **Per-step data pinning in editor**: Allow users to override specific step outputs in the visual editor for testing. Stored as a lightweight scenario file, not inline in the runbook YAML.
- **Live preview with test data**: Combine with expression autocomplete — evaluate expressions against pinned/scenario data and show results.

**Priority:** MEDIUM — the replay system exists; editor integration makes it accessible to non-CLI users.

---

### 7. Sub-Workflow / Reusable Components

**What they do:** n8n's "Execute Workflow" node calls another workflow by ID, passing parameters and receiving outputs. Parent workflow waits for child completion.

**Gert equivalent:** `invoke` step type calls child runbooks by path. Engine tracks `ChainDepth`, `ParentRunID`, `ChildRuns`. The execution viewer has chain breadcrumb navigation, invoke node badges, and parent minimap (Phase D). Input passing via `meta.inputs`.

**Gap:** Minimal. Gert's invoke system is already more capable than n8n's:
- Chain visualization (breadcrumb + minimap) exceeds n8n's flat view
- Governance carries across chains
- Input passing is well-defined via `meta.inputs`

Minor gaps:
- No "output passing" from child back to parent (child captures don't propagate up)
- No invoke step "template" — you must know the exact runbook path

**Recommendation:** **Adapt — LOW priority**  
- **Output propagation**: Allow invoke steps to capture child runbook outputs back into parent captures. Syntax: `capture: {child_result: outcome.code}`. Requires engine change to pass child RunManifest outcome back.
- **Invoke with discovery**: In the editor tool catalog, also list available runbooks as invoke targets.

**Priority:** LOW — invoke works well for current use cases. Output propagation is a nice-to-have.

---

### 8. Sticky Notes / Annotations

**What they do:** n8n lets users place sticky notes on the canvas — colored rectangles with markdown text, positioned freely. Used for documentation, section labels, reviewer notes.

**Gert equivalent:** Step `title` and `instructions` fields serve as inline documentation. No free-form annotations on the graph. No step grouping or section labels beyond the tree structure.

**Gap:** No way to annotate the workflow graph with context that isn't tied to a step. No section/group labels. No reviewer notes visible in the visual view.

**Recommendation:** **Adapt — LOW priority**  
- **Step descriptions** are sufficient for most annotation needs. Gert runbooks are smaller than typical n8n workflows.
- **Section comments in YAML** (preserved by the comment-preserving save) serve as annotations for the YAML view.
- **Graph annotations**: If needed, add an optional `annotations` array in runbook meta: `[{text, position: {after_step: "step_id"}}]`. Rendered as colored boxes in the graph. Low effort, low urgency.
- **Skip free-form canvas positioning**: Gert's graph layout is deterministic from the tree structure. Free positioning breaks reproducibility and adds state that doesn't belong in YAML.

**Priority:** LOW — existing mechanisms are adequate. Revisit if runbook complexity grows significantly.

---

## Part 2: Azure Logic Apps Designer Feature Evaluation

### 9. Designer Surface / Layout

**What they do:** Logic Apps Designer uses a sequential top-to-bottom layout with parallel branches shown as horizontal lanes. Steps are tall cards with inline I/O preview. Connections are straight vertical lines with "+" insert points between steps.

**Gert equivalent:** Top-to-bottom SVG graph with recursive bounding-box layout. Nodes typed by shape (ellipse, diamond, dashed rect, rounded rect). Bezier edges with conditional/back-edge classification. "+" insert points not present — adding steps is via form buttons.

**Gap:** 
- No inline I/O preview on graph nodes (nodes show type icon + title only)
- No "+" insert points between graph nodes for quick step insertion
- Parallel lanes not visualized (gert doesn't have parallel execution yet)

**Recommendation:** **Adapt — MEDIUM priority**  
- **"+" insert points on edges**: Add clickable "+" icons on sequential edges. Clicking opens the add-step flow at that position. This is the single highest-impact UX improvement for the authoring graph — it makes the graph feel editable, not just viewable.
- **Inline step summary on nodes**: Show 1-line summary on graph nodes (tool name + action, or branch count, or iterate `over` value). Currently implemented as title-only. Moderate effort.
- **Skip parallel lanes**: Gert doesn't support parallel execution. Don't add visual constructs for non-existent semantics.

**Priority:** MEDIUM — "+" insert points are a clear UX win. Inline summaries are polish.

---

### 10. Connector Ecosystem / Discovery

**What they do:** Logic Apps has 1000+ managed connectors maintained by Azure. Connectors are searchable by category, have standardized auth, versioned APIs, and consistent UX. Users search "Send email" and get connector suggestions.

**Gert equivalent:** File-based `.tool.yaml` definitions. `tools/list` endpoint returns available tools. Tool packs referenced in `gert.yaml`. No search by intent, no categories, no versioning beyond what's in the YAML `meta.version`.

**Gap:** No intent-based search ("I want to check DNS" → finds nslookup tool). No categories or tags. No tool versioning or dependency resolution.

**Recommendation:** **Adapt — HIGH priority** (overlaps with n8n catalog recommendation)  
- **Tool metadata enrichment**: Add `meta.tags: [networking, dns, diagnostics]` and `meta.category: network` to tool schema. Low cost, high value for search.
- **Intent-based search in editor**: Fuzzy search across tool name, description, action descriptions, and tags. When user types "DNS" in the tool catalog, surface nslookup.
- **Tool versioning**: `meta.version` already exists. Add `requires: {gert: ">=0.2.0"}` for compatibility checks. Tool packs can declare version constraints.
- **Skip managed connector model**: Azure's connector model works because Microsoft maintains 1000+ connectors. Gert is tool-author-driven. The value is in making discovery excellent, not in centralizing maintenance.

**Priority:** HIGH — discovery is the bottleneck. Tags + search are low-effort, high-impact.

---

### 11. Run History / Run Details (Per-Step I/O)

**What they do:** Logic Apps shows a detailed run history panel. Each past run is expandable. For each step: input data, output data, timestamps, duration, status, and raw HTTP request/response. Filtering by status and date range.

**Gert equivalent:** `trace.jsonl` with `TraceEvent` per step (type, timestamp, run_id, step result with stdout/stderr/exit_code/duration). `RunManifest` with step summary counts. Execution viewer shows step states (running/passed/failed) on graph nodes with progress indicators. `stepDetails` map contains per-step result data. No historical run browser — only the current/most recent run.

**Gap:**
- No run history browser (past runs aren't browsable in the editor)
- Per-step I/O display exists in the execution viewer but is minimal (state badge, not full stdout/stderr)
- No input display (what args were passed to a tool)
- No duration/timestamp per step in the UI
- No filtering/search across runs

**Recommendation:** **Adopt — HIGH priority**  
This is the most impactful gap for operational use:
- **Per-step I/O panel**: When clicking a step node in the execution viewer, show a detail panel with: resolved tool args (input), stdout/stderr (output), exit code, duration, timestamp. All data is already in `stepDetails` / trace — it just needs a UI surface.
- **Run history sidebar**: List past runs from `.runbook/runs/` directory. Each run expandable to show manifest summary. Click to replay the trace in the execution viewer.
- **Step diff between runs**: Compare step outputs between two runs of the same runbook. Useful for incident postmortems ("what changed between the failing run and the recovery run?").

**Priority:** HIGH — operators need to inspect what happened. The data exists; the UI doesn't surface it.

---

### 12. Parameters Panel

**What they do:** Logic Apps has a dedicated "Parameters" panel for workflow-level values (connection strings, config values, etc.). Parameters are typed, have default values, and are referenced throughout the workflow using `@parameters('name')`.

**Gert equivalent:** `meta.vars` for static variables, `meta.inputs` for user-prompted or provider-resolved inputs. Both are top-level in the runbook YAML. The editor form renders these under "Runbook Home."

**Gap:** Minor. Gert's `meta.vars` + `meta.inputs` cover the same ground. What's missing:
- No type annotations on vars (all strings)
- No validation rules on inputs (only `from: prompt` vs provider)
- No "parameters panel" as a distinct, always-visible UI surface

**Recommendation:** **Adapt — LOW priority**  
- **Input validation rules**: Add `validate: {pattern: "^https://", min_length: 5}` to input definitions. Engine checks at input resolution time. Minor schema extension.
- **Typed vars**: Add optional `type: string|number|boolean` to vars. Engine coerces/validates at template resolution. Low urgency since everything is string-based today.
- **Skip dedicated panel**: The Runbook Home form already serves this purpose. A separate "Parameters" panel is Logic Apps–specific UX that doesn't add value in gert's 2-column layout.

**Priority:** LOW — existing mechanism is adequate. Input validation rules are the most useful addition.

---

### 13. Peek Code

**What they do:** Logic Apps "Peek Code" shows the raw JSON definition of any action/trigger/connector. Users can inspect and hand-edit the underlying JSON when the designer is insufficient.

**Gert equivalent:** No per-step YAML peek. The editor's "Preview Diff" shows full YAML changes before save. Users can always edit YAML directly in VS Code tabs.

**Gap:** No way to see the YAML fragment for a single step without looking at the full document.

**Recommendation:** **Adopt — MEDIUM priority**  
This is natural for gert and easy to implement:
- **Peek YAML**: Right-click or button on any step node → show a read-only panel with the YAML fragment for that step/iterate/branch. Use `YAML.stringify()` on the TreeNode at the selected path.
- **Edit & apply**: Allow editing the YAML fragment and applying it back to the tree. The comment-preserving serializer handles the merge.
- **Copy YAML**: One-click copy of step YAML to clipboard. Enables sharing step definitions across runbooks.

**Priority:** MEDIUM — quick to implement, aligns with YAML-as-truth philosophy, power users will love it.

---

### 14. Concurrency Control

**What they do:** Logic Apps supports concurrent loop iterations with configurable degree of parallelism (1–50). ForEach loops can run items in parallel. Sequential mode is opt-in.

**Gert equivalent:** `IterateBlock` supports convergence mode (`max` + `until`) and list mode (`over` + `as`). All iteration is sequential. No parallel execution within or across iterate blocks. Engine processes one step at a time.

**Gap:** No parallel iteration. No concurrent step execution. For iterate blocks over large lists, this means linear execution time scaling.

**Recommendation:** **Adapt — MEDIUM priority**  
Concurrency is valuable but must respect governance:
- **Parallel iterate**: Add `concurrency: N` to IterateBlock. Engine spawns up to N goroutines for list-mode iteration. Each goroutine gets its own capture scope (no shared mutation). Results merged after all complete.
- **Governance constraint**: `requires_approval` steps within a parallel iterate must serialize (can't prompt for approval in parallel). Engine automatically reduces concurrency to 1 for approval-gated steps.
- **Convergence mode stays sequential**: Convergence loops (`until` condition) are inherently sequential — each pass depends on the previous state. No parallelism.
- **Skip full DAG parallelism**: n8n and Logic Apps can run independent branches in parallel. Gert's tree model is sequential by design. Adding DAG parallelism would require a fundamentally different execution model. Not worth it.

**Priority:** MEDIUM — valuable for list-mode iterates over 10+ items. Not blocking for most runbooks.

---

### 15. Managed Identity / Auth

**What they do:** Logic Apps connectors authenticate via Azure Managed Identity, eliminating credential management for Azure resources. The connector framework handles token acquisition, refresh, and scoping transparently.

**Gert equivalent:** Provider configs specify transport (JSON-RPC, MCP, CLI) and connection details. Environment variables carry credentials. No managed identity integration. No token management.

**Gap:** When gert tools interact with Azure resources (Key Vault, ARM, etc.), users must manually configure service principals, tokens, or env vars. No transparent auth.

**Recommendation:** **Adapt — MEDIUM priority**  
- **Azure provider with managed identity**: A new provider type `azure-identity` that uses `DefaultAzureCredential` (Azure SDK Go) to acquire tokens. Tools declared with `provider: azure-identity` get tokens injected automatically.
- **Token injection via env**: Rather than changing tool execution, the provider injects `AZURE_ACCESS_TOKEN` (or tool-specific env vars) before tool execution. Tools use the token as a normal env var.
- **Don't build a generic auth framework**: Each cloud provider (Azure, AWS, GCP) has its own SDK credential chain. Support Azure first (explicit user affinity), others via provider plugins.

**Priority:** MEDIUM — significant value for Azure-heavy environments. Leorio's integration model (2026-03-17 decision) already suggests this direction.

---

### 16. Integration Accounts / Schema Validation

**What they do:** Logic Apps Integration Accounts validate messages against XML/JSON schemas, transform data between formats, and manage partner agreements. This is enterprise B2B integration.

**Gert equivalent:** JSON Schema validation for runbooks (`schemas/runbook-v0.json`, `runbook-v1.json`, `tool-v0.json`). Deep validation in `pkg/schema/deep_validate.go`. `exec/dryRun` validates the full pipeline. `schema/bundle` endpoint returns schemas at runtime.

**Gap:** Minimal for gert's domain. Schema validation of runbook/tool definitions is comprehensive. What doesn't exist:
- No validation of tool *output* against a schema (tool captures are untyped strings)
- No data transformation between steps (no map/filter/transform operators)

**Recommendation:** **Skip — LOW priority**  
- Gert is an operational runbook engine, not an integration platform. B2B schema validation and data transformation are out of scope.
- **Typed captures** (optional schema for tool outputs) could be useful eventually but add complexity without solving current user problems.
- If data transformation is needed, use a tool step that calls `jq` or a transform utility.

**Priority:** LOW — existing schema validation is strong. Don't scope-creep into data integration.

---

### 17. Monitoring / Alerts

**What they do:** Logic Apps integrates with Azure Monitor for run metrics (success rate, duration, failure rate), log queries (KQL across runs), and alert rules (notify on failure, SLA breach). Dashboards show workflow health over time.

**Gert equivalent:** No monitoring integration. `RunManifest` captures per-run metadata. `trace.jsonl` captures per-step detail. Execution viewer shows live state. No aggregation, no alerting, no dashboards.

**Gap:** No aggregate run metrics. No failure alerting. No historical dashboards. Operators must manually check run outputs.

**Recommendation:** **Adapt — MEDIUM priority**  
Full Azure Monitor integration is too heavy for gert's current stage, but basic monitoring hooks have high value:
- **Structured run summary output**: `gert run` exits with a structured JSON summary (already close — RunManifest is written). Ensure it's parseable by external monitoring tools.
- **Exit code semantics**: Document and enforce exit codes: 0 = success, 1 = step failure, 2 = governance block, 3 = input error. External monitoring can alert on non-zero.
- **Webhook notification on completion**: Optional `meta.notify: {webhook: "https://...", on: [failure, success]}`. Engine POSTs the RunManifest to the webhook URL on completion. Enables integration with any monitoring system (PagerDuty, Teams, Slack, Azure Monitor via Logic App).
- **Azure Monitor exporter (future)**: A `gert-monitor` extension that reads `.runbook/runs/` and pushes metrics to Azure Monitor / App Insights. Not in core — as an ext/ package.

**Priority:** MEDIUM — webhook notification is high-value and low-effort. Full monitoring integration is a separate extension.

---

## Summary Matrix

| # | Feature | Source | Gert Today | Recommendation | Priority |
|---|---------|--------|-----------|----------------|----------|
| 1 | Tool catalog / marketplace | n8n | `tools/list` endpoint, file-based | **Adapt**: tool catalog panel, tool packs, install-from-URL | **HIGH** |
| 2 | Credential management | n8n | Env vars + providers | **Adapt**: credential refs in tools, VS Code SecretStorage | MEDIUM |
| 3 | Expression autocomplete | n8n | Condition builder only | **Adopt**: autocomplete in all fields, scope-aware, live preview | **HIGH** |
| 4 | Error handling / retry | n8n | `continue_on_fail` only | **Adapt**: retry config, error branches | **HIGH** |
| 5 | Webhook triggers | n8n | None (CLI-driven) | **Skip**: external orchestrators handle this | LOW |
| 6 | Pinned data / test data | n8n | Replay/scenarios | **Adapt**: editor integration, capture-to-scenario | MEDIUM |
| 7 | Sub-workflow / reusable | n8n | `invoke` steps + chain viz | **Adapt**: output propagation (minor) | LOW |
| 8 | Sticky notes / annotations | n8n | Step title/instructions | **Adapt**: optional annotations array (minor) | LOW |
| 9 | Designer surface | Logic Apps | SVG graph, bounding-box layout | **Adapt**: "+" insert points on edges, inline summaries | MEDIUM |
| 10 | Connector discovery | Logic Apps | File-based, no search | **Adapt**: tags, fuzzy search, categories | **HIGH** |
| 11 | Run history / per-step I/O | Logic Apps | trace.jsonl, basic states | **Adopt**: I/O panel, run history browser, step diff | **HIGH** |
| 12 | Parameters panel | Logic Apps | `meta.vars` + `meta.inputs` | **Adapt**: input validation rules (minor) | LOW |
| 13 | Peek code | Logic Apps | Preview diff only | **Adopt**: per-step YAML peek + copy | MEDIUM |
| 14 | Concurrency control | Logic Apps | Sequential only | **Adapt**: parallel iterate with governance constraints | MEDIUM |
| 15 | Managed identity / auth | Logic Apps | Env vars | **Adapt**: Azure provider with DefaultAzureCredential | MEDIUM |
| 16 | Schema validation (B2B) | Logic Apps | Runbook/tool JSON Schema | **Skip**: out of scope for operational runbooks | LOW |
| 17 | Monitoring / alerts | Logic Apps | None | **Adapt**: webhook notify, structured exit codes | MEDIUM |

---

## Recommended Implementation Phases

### Phase 1 — High Priority (next sprint cycle)
1. **Tool catalog panel** in editor (#1, #10) — searchable tool list from `tools/list`, tag metadata
2. **Expression autocomplete** (#3) — scope-aware variable dropdown in all template fields
3. **Retry per step** (#4) — `retry: {max, interval, backoff}` schema + engine support
4. **Per-step I/O panel** (#11) — click step in execution viewer → see resolved args, stdout, stderr, duration

### Phase 2 — Medium Priority (following sprint)
5. **"+" edge insert points** (#9) — clickable add-step on graph edges
6. **Run history browser** (#11) — list past runs, load trace into viewer
7. **Peek YAML** (#13) — per-step YAML fragment view
8. **Error branches** (#4) — `_error` condition type in branch evaluation

### Phase 3 — Medium Priority (planned)
9. **Parallel iterate** (#14) — `concurrency: N` for list-mode
10. **Scenario browser** (#6) — editor integration for replay scenarios
11. **Webhook notify** (#17) — `meta.notify` on run completion
12. **Credential refs** (#2) — tool credential documentation + VS Code SecretStorage

### Deferred
13. Tool packs / install-from-URL (#1) — after catalog panel proves useful
14. Azure managed identity provider (#15) — when Azure tooling demand materializes
15. Annotations (#8) — if runbook complexity grows
16. Input validation rules (#12) — minor schema extension when needed

---

## Key Architectural Principles

1. **Don't replicate SaaS platforms.** Gert is a governed CLI tool with a VS Code editor, not a hosted automation platform. Features should enhance the operator experience, not recreate n8n/Logic Apps inside VS Code.

2. **Data already exists — surface it.** The biggest wins (run history, per-step I/O, expression autocomplete) don't require new data collection. The engine already captures everything; the UI just doesn't expose it.

3. **Governance gates must apply to new features.** Parallel iterate must respect approval gates. Retry must respect governance timeouts. Credential refs must honor redaction rules. Every feature must pass through the governance lens.

4. **Tool ecosystem grows by making authoring easy, not by centralizing.** n8n and Logic Apps invest in maintaining 400–1000+ integrations. Gert should invest in making tool YAML trivially easy to write, discover, and share. The community creates tools; gert makes them findedable and usable.

5. **Expression safety over power.** n8n lets users write arbitrary JavaScript in expressions. Gert's Go template system is intentionally constrained. Autocomplete should guide users toward correct expressions, not expand the expression language.
