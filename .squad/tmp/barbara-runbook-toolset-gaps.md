# Runbook and Toolset Gap Analysis

**Analyst:** Barbara (Integrations Specialist)  
**Date:** 2026-04-19  
**Context:** Correctness strategy — integration test coverage across all 13 phases

---

## Executive Summary

### Gap 1: Sample Runbooks
**Current state:** 10 runbooks exist (r01–r10) covering 8 of 14 step types. Missing: `iterate`, `approve`, `decision`, `sleep`, `log`, `set`.  
**Critical gap:** No dedicated runbooks testing `iterate`, `approve` (standalone), or `decision` (routing). These require minimal focused fixtures.  
**Blocked phases:** Phase 5 (Step Types), Phase 8 (Input Providers), Phase 11 (Evidence & Replay).

### Gap 2: Standard Toolset
**Current state:** Spec references "built-in tool registry" but does NOT define what's in it. Runbooks reference 10+ builtin tools (slack, pagerduty, aws, etc.) that don't exist.  
**Critical gap:** No spec for reference/test tools. No minimal tool set for testing stdio/jsonrpc/mcp transports.  
**Blocked phases:** Phase 6 (Tool Runtime), Phase 11 (Evidence & Replay), Phase 13 (Acceptance).

---

## Gap 1: Sample Runbooks — Phase-by-Phase Analysis

### Phase 0: Foundation — types, interfaces, JSON Schema
**Needs runbooks?** No  
**Rationale:** Foundation is pure types and interfaces. No execution engine exists yet. JSON Schema artifacts are validated by static tooling, not runbook execution.

---

### Phase 1: Parser & Schema — v2 parser, validation
**Needs runbooks?** Partial  
**What exists:** All 10 runbooks (r01–r10) serve as parser fixtures. Cover 8 of 14 step types.  
**What's missing:** 6 step types have NO runbook coverage:
- `iterate` — loop over collection, or convergence with `until`
- `approve` — standalone approval gate (not embedded in `collector.approvals`)
- `decision` — user selects execution path (router)
- `sleep` — pause execution for N seconds
- `log` — emit structured log event
- `set` — explicit variable assignment

**Impact:** Parser can be implemented for these types using only the spec, but will have NO real-world validation fixtures until integration tests are written in later phases. This creates a blind spot where parser bugs may not surface until Phase 5+ when step executors are written.

**Which runbooks cover what:**

| Step Type | Covered By | Count |
|-----------|-----------|-------|
| `cli` | r01, r02, r05, r06, r09, r10 | 87 occurrences |
| `tool` | r01, r02, r03, r04, r05 | 31 occurrences |
| `choice` | r05, r08 | 2 occurrences |
| `collector` | r01, r02, r03, r04, r05, r06, r07, r08, r10 | 64 occurrences |
| `branch` | r01, r02, r03, r05, r06, r07, r08, r09, r10 | 14 occurrences |
| `parallel` | r01, r02, r03, r04, r05, r06, r10 | 11 occurrences |
| `wait_for_event` | r01, r05 | 2 occurrences |
| `include` | r04, r09 | 7 occurrences |
| `compensate` | r02, r05, r06 | 3 occurrences |
| `assert` | r02, r06 | 2 occurrences |
| `end` | r01, r02, r03, r04, r05, r06, r07, r08, r09, r10 | 10 occurrences |
| **`iterate`** | ❌ NONE | 0 |
| **`approve`** | ❌ NONE | 0 |
| **`decision`** | ❌ NONE | 0 |
| **`sleep`** | ❌ NONE | 0 |
| **`log`** | ❌ NONE | 0 |
| **`set`** | ❌ NONE | 0 |

**Note on `iterate`:** r02 (canary-deploy) has PROSE describing a 10-minute monitoring loop with 60 iterations, but the schema.yaml does NOT use an `iterate` node. It uses 17 `cli` steps in sequence instead. This is a workaround, not a real `iterate` test.

---

### Phase 2: Planner — import resolution, tool discovery, ExecutionPlan
**Needs runbooks?** Yes  
**What exists:**
- r04 (soc2-evidence) has 6 `include` steps → tests import resolution  
- r09 (oncall-escalation) has 1 `include` step  
- r01, r02, r03, r04, r05 reference tools → tests tool discovery

**What's missing:**
- No runbook testing nested includes (A includes B, B includes C)
- No runbook testing cyclic include detection (A → B → A)
- No runbook testing tool name collision (project-level vs builtin)
- No runbook testing MCP tool discovery (dynamic catalog from `tools/list`)

**Impact:** Phase 2 planner can implement basic import resolution, but edge cases (cycles, multi-level nesting, MCP dynamic discovery) have no fixtures. These will require new minimal runbooks or enhancement of r04.

---

### Phase 3: Runtime Core & Trace — engine loop, event bus, trace.jsonl
**Needs runbooks?** Yes  
**What exists:** All 10 runbooks are execution fixtures. Every runbook produces a `trace.jsonl` output.

**What's missing:**
- No runbook specifically designed for trace format validation (HMAC chaining, event ordering, sequence monotonicity)
- No runbook testing run suspension/cancellation mid-execution
- No runbook testing run timeout (wall-clock or business-day)

**Impact:** Phase 3 can use existing runbooks for basic trace emission. Golden trace tests (Phase 11 dependency) will require ALL step types to be covered — currently 6 are missing.

---

### Phase 4: Governance Engine — allowlist/denylist, env blocking, redaction, RBAC, approval gates
**Needs runbooks?** Yes  
**What exists:**
- All runbooks with `collector` steps (9 runbooks) test approval gates via `approvals:` field
- r07 (financial-approval) has 8 `collector` steps with approval gates and business-day timeouts
- No runbooks explicitly test **standalone `approve` step**

**What's missing:**
- No runbook testing `type: approve` (dedicated approval gate without data collection)
- No runbook testing RBAC enforcement (role-based step gating)
- No runbook testing redaction rules (`governance.redact` field)
- No runbook testing environment blocking (`allowed-environments: [real, dry-run]`)
- No runbook testing capability gates (`requires-capabilities: [capability/network]`)

**Impact:** Phase 4 governance rules can be tested via unit tests for most features. Approval gate logic embedded in `collector` is covered by existing runbooks, but standalone `approve` step has no fixture. RBAC/redaction/env-blocking will require new minimal runbooks.

---

### Phase 5: Step Types — all 14 v2 step types
**Needs runbooks?** Yes — CRITICAL  
**What exists:** 8 of 14 step types have runbook coverage (see Phase 1 table above).

**What's missing:** 6 step types have NO runbook coverage:
1. **`iterate`** (HIGH PRIORITY) — loop semantics, `over` vs `until`, `collect` accumulation, early exit conditions
2. **`approve`** (MEDIUM) — standalone approval gate, quorum mode, business-day timeout
3. **`decision`** (MEDIUM) — user picks execution path, `goto` vs `runbook` routing
4. **`sleep`** (LOW) — pause for N seconds
5. **`log`** (LOW) — emit structured log event
6. **`set`** (LOW) — explicit variable assignment

**Impact:** Phase 5 is BLOCKED for these 6 step types until minimal runbooks are created. Without real runbook fixtures, step executor implementations will be untested against real schemas. Unit tests alone are insufficient — integration tests require parseable YAML runbooks.

**Recommendation:** Create 3 new minimal runbooks before Phase 5:
- **r11-iterate-loop.yaml** — tests `iterate.over` (list), `iterate.until` (convergence), `collect` accumulation
- **r12-approval-quorum.yaml** — tests standalone `type: approve` with `mode: quorum`, business-day timeout
- **r13-decision-routing.yaml** — tests `type: decision` with `goto` and `runbook` routes

Sleep/log/set can be tested via unit tests or added to existing runbooks as single-step additions.

---

### Phase 6: Tool Runtime — stdio/jsonrpc/mcp transports, capability gate
**Needs runbooks?** Yes  
**What exists:**
- r01, r02, r03, r04, r05 use `type: tool` steps (31 occurrences)
- r01 references builtin tools: `slack-notify`, `pagerduty-notify`, `alertmanager-notify`
- r05 references builtin tools: `aws`, `okta`, `palo-alto`, `splunk`, `email`, `pagerduty`, `slack`

**What's missing:**
- NO `.tool.yaml` definitions exist for any of these builtin tools
- NO reference tools for testing stdio/jsonrpc/mcp transports
- NO test tools for failure modes (non-zero exit, timeout, crash, invalid JSON)
- NO MCP reference server for testing MCP transport

**Impact:** Phase 6 is BLOCKED — cannot test tool runtime without actual tool definitions. See Gap 2 below for full toolset analysis.

---

### Phase 7: Extension Host — manifest lifecycle, Ed25519 trust, crash isolation
**Needs runbooks?** No  
**Rationale:** Extension host is tested via extension manifests (`.gert-extension.yaml`), not runbooks. Extension lifecycle and trust verification are integration tests at the extension boundary, not runbook execution tests.

---

### Phase 8: Input Provider — provider framework, 4 built-in providers
**Needs runbooks?** Partial  
**What exists:**
- r07 uses `options_from` field on `select` type → tests dynamic provider lookup
- All runbooks with `collector` steps (9 runbooks) test the `prompt` provider (built-in, interactive)

**What's missing:**
- No runbook testing `env` provider (environment variable lookup)
- No runbook testing `file` provider (read from file)
- No runbook testing `workspace` provider (workspace metadata lookup)
- No runbook testing provider composition chains (`from: [provider-a, provider-b]`)
- No runbook testing provider failure/fallback behavior

**Impact:** Phase 8 can implement built-in providers using only the spec. Runbook fixtures for provider behavior are missing, but can be added as enhancements to r07 or new minimal runbooks.

---

### Phase 9: gert serve — RPC, WebSocket events, RBAC, TLS
**Needs runbooks?** Partial  
**What exists:**
- r01, r05 use `type: wait_for_event` with `source: webhook` → tests inbound event receive in `gert serve` mode

**What's missing:**
- No runbook testing WebSocket event streaming
- No runbook testing RBAC enforcement (role-based execution gating)
- No runbook testing run cancellation via RPC
- No runbook testing multi-client concurrent execution (isolation)

**Impact:** Phase 9 serve mode can be tested with r01/r05 for basic webhook receive. RBAC/WebSocket/concurrency require new fixtures or contract tests (not runbooks).

---

### Phase 10: Adapters — Bubble Tea TUI + VS Code client
**Needs runbooks?** No  
**Rationale:** Adapter contracts are tested via contract test suites (`testdata/contracts/`), not runbooks. Runbooks are execution fixtures; adapters are client implementations. Adapters consume `exec/v2` and `events/v2` contracts — those contracts are tested separately.

---

### Phase 11: Evidence & Replay — snapshots, --resume, gert test, HMAC chaining
**Needs runbooks?** Yes — CRITICAL  
**What exists:** All 10 runbooks produce `trace.jsonl` outputs suitable for HMAC verification.

**What's missing:**
- No runbook testing `--resume` from checkpoint (requires run suspension mid-execution)
- No runbook testing `gert test` mode (dry-run with assertions)
- No runbook specifically designed for snapshot/resume cycle
- **CRITICAL:** 6 step types (iterate, approve, decision, sleep, log, set) have NO trace coverage because they have NO runbooks

**Impact:** Phase 11 is BLOCKED for the 6 missing step types. Cannot validate HMAC chaining or replay semantics for step types that have no runbook fixtures. Requires r11/r12/r13 minimal runbooks (see Phase 5).

---

### Phase 12: Observability — OTel, Prometheus, structured logs, gert verify
**Needs runbooks?** Partial  
**What exists:** All 10 runbooks emit structured logs and trace events suitable for OTel span extraction.

**What's missing:**
- No runbook testing OTel span propagation across `include` boundaries
- No runbook testing distributed trace context across tool invocations
- No runbook testing Prometheus metric emission (step duration, error rate, approval wait time)

**Impact:** Phase 12 observability features can be tested using existing runbooks. Span propagation and metric cardinality testing may require new fixtures or enhancement of r04 (includes).

---

### Phase 13: Acceptance — integration corpus, golden traces, perf benchmarks
**Needs runbooks?** Yes — CRITICAL  
**What exists:** 10 runbooks serve as the integration corpus baseline.

**What's missing:**
- 6 step types (iterate, approve, decision, sleep, log, set) have NO golden traces
- No runbook designed for performance benchmarking (e.g., 1000-step runbook, deeply nested includes)
- No runbook testing worst-case scenarios (max retry, max timeout, max approval wait)

**Impact:** Phase 13 acceptance testing is INCOMPLETE. Cannot declare v2 production-ready when 43% of step types (6/14) have no runbook coverage. Requires r11/r12/r13 minimal runbooks before acceptance phase.

---

## Gap 1: What Needs to Be Created

### New Runbooks Required (HIGH PRIORITY)

#### r11-iterate-loop.yaml
**Purpose:** Test iterate semantics  
**Coverage:**
- `iterate.over` with comma-separated list (3+ items)
- `iterate.as` custom loop variable name
- `iterate.collect` accumulation across passes
- `iterate.until` convergence condition
- `iterate.max` limit enforcement
- Early exit on failure (`continue_on_fail: false` inside iterate)

**Step types used:** iterate, cli, assert, end  
**Size:** ~40 lines  
**Blocks:** Phase 5, Phase 11, Phase 13

---

#### r12-approval-quorum.yaml
**Purpose:** Test standalone approval gate  
**Coverage:**
- `type: approve` (not embedded in collector)
- `approvals.mode: quorum` with `pool` and `required`
- `timeout_business_days` with timezone
- `business_calendar` (e.g., `us-federal`)
- `on_timeout: fail` vs `on_timeout: continue`

**Step types used:** approve, end  
**Size:** ~30 lines  
**Blocks:** Phase 4, Phase 5, Phase 11, Phase 13

---

#### r13-decision-routing.yaml
**Purpose:** Test decision step routing  
**Coverage:**
- `type: decision` with 3+ routes
- Route with `goto` (jump to step ID)
- Route with `runbook` (invoke external runbook)
- `variable` storage of selected route label (audit trail)

**Step types used:** decision, include, end  
**Size:** ~50 lines  
**Blocks:** Phase 5, Phase 8 (provider), Phase 11, Phase 13

---

### Enhancements to Existing Runbooks (MEDIUM PRIORITY)

#### r02-canary-deploy.yaml
**Current:** Uses 17 sequential `cli` steps for monitoring loop.  
**Enhancement:** Replace with `iterate.max: 60` + `iterate.until` convergence condition.  
**Why:** r02 prose describes iteration, but schema doesn't use `iterate` step type. This is a missed opportunity.

#### r07-financial-approval.yaml
**Current:** Uses `collector.approvals` for all approval gates.  
**Enhancement:** Add one standalone `type: approve` step for legal review.  
**Why:** r07 is the approval-focused runbook, but doesn't test standalone approve step.

---

### Low-Priority Additions (can be deferred to Phase 5+)

#### sleep, log, set step types
**Where to add:**
- r01: Add `type: sleep` with `duration: 10s` before retry loop
- r05: Add `type: log` with structured metadata before alert notification
- r06: Add `type: set` to explicitly assign migration status variable

**Why low priority:** These are simple step types with minimal logic. Can be validated via unit tests. Real runbook coverage is nice-to-have, not blocking.

---

## Gap 2: Standard Toolset — What Is It?

### Current State

**Spec references but does not define:**
- §05 Tool Runtime line 12: "The built-in tool registry compiled into the host binary."
- §05 line 31: "Exact match in the built-in registry."

**What the spec DOES define:**
- Tool discovery algorithm (4 sources: builtin → project → required-packages → config)
- `.tool.yaml` schema with `apiVersion: tool/v2`
- Three transport modes: stdio, stdio-jsonrpc, mcp

**What the spec DOES NOT define:**
- Which tools are in the built-in registry
- What actions each builtin tool provides
- Which transport each builtin tool uses
- How to test stdio/jsonrpc/mcp transports independently

---

### What Runbooks Reference (builtin tools with NO definitions)

| Tool Name | Referenced By | Transport Assumed | Actions Used |
|-----------|---------------|-------------------|--------------|
| slack-notify | r01, r05 | stdio | send-message |
| pagerduty-notify | r01, r05 | stdio | create-incident |
| alertmanager-notify | r01 | stdio | resolve |
| aws | r05 | stdio-jsonrpc | iam-revoke, s3-block |
| okta | r05 | stdio-jsonrpc | disable-user |
| palo-alto | r05 | stdio-jsonrpc | block-ip |
| splunk | r05 | stdio-jsonrpc | query |
| email | r05 | stdio | send |

**Problem:** These tools are referenced in runbook `toolRefs` sections with `source: builtin://name`, but NO `.tool.yaml` definitions exist anywhere in the repository.

**Impact:**
- Runbooks r01 and r05 cannot execute — tool resolution will fail
- Phase 6 tool runtime cannot be tested — no actual tools to invoke
- Phase 13 acceptance corpus is broken — 2 of 10 runbooks are non-executable

---

### What's Missing: Reference Tools for Testing

To test the tool runtime (Phase 6), the following reference tools are REQUIRED:

#### 1. Minimal stdio Tool: `echo`
**Purpose:** Simplest possible tool — returns its input unchanged  
**Transport:** stdio  
**Binary:** `/bin/echo` (exists on all platforms)  
**Actions:** `echo` (single action, takes one arg)  
**Use case:** Test stdio transport invocation, argument templating, stdout capture

**Definition:** `tools/echo.tool.yaml`
```yaml
apiVersion: tool/v2
meta:
  name: echo
  version: "1.0.0"
  description: "Echo input to stdout"
  binary: echo
transport:
  mode: stdio
actions:
  echo:
    description: "Echo a message"
    argv: ["{{ .message }}"]
    args:
      message:
        type: string
        required: true
    capture:
      stdout:
        format: text
```

---

#### 2. Failing Tool: `fail`
**Purpose:** Always exits with non-zero code  
**Transport:** stdio  
**Binary:** Custom Go binary (100 lines)  
**Actions:** `fail` (exits 1), `timeout` (sleeps forever)  
**Use case:** Test tool error handling, timeout enforcement, retry logic

**Definition:** `tools/fail.tool.yaml`
```yaml
apiVersion: tool/v2
meta:
  name: fail
  version: "1.0.0"
  description: "Always fails for testing"
  binary: gert-test-fail
transport:
  mode: stdio
actions:
  fail:
    description: "Exit with code 1"
    argv: ["--exit-code", "1"]
    timeout: "5s"
  timeout:
    description: "Sleep forever to trigger timeout"
    argv: ["--sleep", "9999"]
    timeout: "2s"
```

**Binary location:** `testdata/tools/gert-test-fail/main.go`  
**Deliverable:** Phase 6 must include this binary in the test suite.

---

#### 3. Slow Tool: `slow`
**Purpose:** Delays before returning (tests timeout and cancellation)  
**Transport:** stdio  
**Binary:** Custom Go binary (50 lines)  
**Actions:** `delay` (sleeps N seconds then succeeds)  
**Use case:** Test per-tool timeout, run-level cancellation, graceful shutdown

**Definition:** `tools/slow.tool.yaml`
```yaml
apiVersion: tool/v2
meta:
  name: slow
  version: "1.0.0"
  description: "Delays before success"
  binary: gert-test-slow
transport:
  mode: stdio
actions:
  delay:
    description: "Sleep then exit 0"
    argv: ["--delay", "{{ .seconds }}"]
    timeout: "10s"
    args:
      seconds:
        type: integer
        required: true
        validation:
          min: 0
          max: 60
```

**Binary location:** `testdata/tools/gert-test-slow/main.go`  
**Deliverable:** Phase 6 must include this binary in the test suite.

---

#### 4. JSON Output Tool: `json-emitter`
**Purpose:** Emits structured JSON to stdout  
**Transport:** stdio  
**Binary:** Custom Go binary (100 lines)  
**Actions:** `emit` (returns JSON object with templated fields)  
**Use case:** Test JSON output capture (`capture: json.field.path`), schema validation

**Definition:** `tools/json-emitter.tool.yaml`
```yaml
apiVersion: tool/v2
meta:
  name: json-emitter
  version: "1.0.0"
  description: "Emits structured JSON"
  binary: gert-test-json-emitter
transport:
  mode: stdio
actions:
  emit:
    description: "Emit JSON object"
    argv: ["--key", "{{ .key }}", "--value", "{{ .value }}"]
    args:
      key:
        type: string
        required: true
      value:
        type: string
        required: true
    capture:
      stdout:
        format: json
```

**Binary location:** `testdata/tools/gert-test-json-emitter/main.go`  
**Output example:** `{"key": "status", "value": "running"}`  
**Deliverable:** Phase 6 must include this binary in the test suite.

---

#### 5. JSON-RPC Reference Server: `jsonrpc-test-server`
**Purpose:** Persistent process accepting JSON-RPC requests  
**Transport:** stdio-jsonrpc  
**Binary:** Custom Go binary (300 lines)  
**Actions:** `ping`, `add`, `error` (triggers JSON-RPC error response)  
**Use case:** Test stdio-jsonrpc transport, startup/shutdown, persistent process lifecycle, error envelope handling

**Definition:** `tools/jsonrpc-test-server.tool.yaml`
```yaml
apiVersion: tool/v2
meta:
  name: jsonrpc-test
  version: "1.0.0"
  description: "JSON-RPC test server"
  binary: gert-test-jsonrpc-server
transport:
  mode: stdio-jsonrpc
  startup:
    argv: ["--server-mode"]
    ready-pattern: "ready"
    timeout: "10s"
    shutdown-method: "shutdown"
actions:
  ping:
    description: "Returns pong"
    args: {}
  add:
    description: "Add two numbers"
    args:
      a:
        type: integer
        required: true
      b:
        type: integer
        required: true
  error:
    description: "Trigger JSON-RPC error"
    args:
      code:
        type: integer
        required: true
```

**Binary location:** `testdata/tools/gert-test-jsonrpc-server/main.go`  
**Deliverable:** Phase 6 must include this binary in the test suite.

---

#### 6. MCP Reference Server: `mcp-test-server`
**Purpose:** MCP-compliant server with dynamic tool discovery  
**Transport:** mcp  
**Binary:** Custom Go binary (500 lines)  
**Tools advertised:** `mcp-ping`, `mcp-lookup`  
**Use case:** Test MCP transport, `initialize` handshake, `tools/list`, `tools/call`, `tools/cancel`

**Declaration:** In `.gert/config.yaml` or runbook meta:
```yaml
tools:
  mcp-servers:
    - name: test-mcp
      binary: gert-test-mcp-server
      args: ["--stdio"]
      transport: mcp
      startup:
        timeout: "15s"
```

**Binary location:** `testdata/tools/gert-test-mcp-server/main.go`  
**MCP tools advertised:**
- `mcp-ping` — returns "pong"
- `mcp-lookup` — takes `id` arg, returns mock data

**Deliverable:** Phase 6 must include this binary in the test suite.

---

### Standard Builtin Tools (for r01, r05 compatibility)

The following builtin tools are referenced by existing runbooks. They MUST be defined as stubs (or real implementations) before r01/r05 can execute.

#### Approach 1: Stub Implementations (Phase 6 deliverable)
Create minimal `.tool.yaml` definitions that satisfy tool resolution but don't perform real actions.

Example: `tools/builtin/slack.tool.yaml`
```yaml
apiVersion: tool/v2
meta:
  name: slack-notify
  version: "1.0.0"
  description: "Slack notification stub"
  binary: gert-test-stub
transport:
  mode: stdio
actions:
  send-message:
    description: "Send message to Slack"
    argv: ["--action", "slack", "--message", "{{ .message }}"]
    args:
      message:
        type: string
        required: true
```

**Binary:** `gert-test-stub` — accepts any action, echoes to stdout, exits 0.

**Builtin stubs required:**
1. `slack-notify`
2. `pagerduty-notify`
3. `alertmanager-notify`
4. `aws`
5. `okta`
6. `palo-alto`
7. `splunk`
8. `email`

**Location:** `testdata/tools/builtin/*.tool.yaml`  
**Compiled into binary:** These definitions are embedded in the gert binary during build (Phase 6 deliverable).

---

#### Approach 2: Real Implementations (Deferred to v2.1)
Implement actual integrations with Slack/PagerDuty/AWS/etc. This is a v2.1 feature — NOT required for Phase 6 or Phase 13 acceptance.

---

### What Needs to Be Created: Toolset Summary

| Tool | Type | Binary | Definition | Phase |
|------|------|--------|-----------|-------|
| `echo` | Reference (stdio) | `/bin/echo` | `testdata/tools/echo.tool.yaml` | Phase 6 |
| `fail` | Reference (stdio) | `gert-test-fail` | `testdata/tools/fail.tool.yaml` | Phase 6 |
| `slow` | Reference (stdio) | `gert-test-slow` | `testdata/tools/slow.tool.yaml` | Phase 6 |
| `json-emitter` | Reference (stdio) | `gert-test-json-emitter` | `testdata/tools/json-emitter.tool.yaml` | Phase 6 |
| `jsonrpc-test` | Reference (jsonrpc) | `gert-test-jsonrpc-server` | `testdata/tools/jsonrpc-test.tool.yaml` | Phase 6 |
| `mcp-test` | Reference (mcp) | `gert-test-mcp-server` | MCP server binary | Phase 6 |
| `slack-notify` | Builtin stub | `gert-test-stub` | `testdata/tools/builtin/slack.tool.yaml` | Phase 6 |
| `pagerduty-notify` | Builtin stub | `gert-test-stub` | `testdata/tools/builtin/pagerduty.tool.yaml` | Phase 6 |
| `alertmanager-notify` | Builtin stub | `gert-test-stub` | `testdata/tools/builtin/alertmanager.tool.yaml` | Phase 6 |
| `aws` | Builtin stub | `gert-test-stub` | `testdata/tools/builtin/aws.tool.yaml` | Phase 6 |
| `okta` | Builtin stub | `gert-test-stub` | `testdata/tools/builtin/okta.tool.yaml` | Phase 6 |
| `palo-alto` | Builtin stub | `gert-test-stub` | `testdata/tools/builtin/palo-alto.tool.yaml` | Phase 6 |
| `splunk` | Builtin stub | `gert-test-stub` | `testdata/tools/builtin/splunk.tool.yaml` | Phase 6 |
| `email` | Builtin stub | `gert-test-stub` | `testdata/tools/builtin/email.tool.yaml` | Phase 6 |

**Total deliverables:** 14 tool definitions + 6 custom binaries + 1 generic stub binary

---

## Which Phases Are Blocked?

### Blocked by Missing Runbooks (Gap 1)

| Phase | Blocked By | Impact |
|-------|-----------|--------|
| Phase 5 | No runbooks for `iterate`, `approve`, `decision` | Cannot integration-test these step executors |
| Phase 8 | No runbooks testing provider composition chains | Provider fallback untested |
| Phase 11 | 6 step types have no trace coverage | HMAC chaining incomplete; replay untested for 43% of step types |
| Phase 13 | 6 step types have no golden traces | Acceptance corpus incomplete; cannot declare production-ready |

**Resolution:** Create r11, r12, r13 minimal runbooks before Phase 5 begins.

---

### Blocked by Missing Toolset (Gap 2)

| Phase | Blocked By | Impact |
|-------|-----------|--------|
| Phase 6 | No reference tools for stdio/jsonrpc/mcp | Cannot test transport layer independently |
| Phase 6 | No builtin tool definitions | r01 and r05 are non-executable |
| Phase 11 | No tool replay fixtures | Cannot test `--resume` with tool invocations |
| Phase 13 | r01 and r05 broken | Acceptance corpus has 20% failure rate before code is written |

**Resolution:** Define 6 reference tools + 8 builtin stubs as Phase 6 deliverables. Write `gert-test-stub` generic binary (100 lines Go) to satisfy all builtin stubs.

---

## Recommendations

### Immediate Actions (Before Phase 5)
1. **Create r11-iterate-loop.yaml** — 40 lines, tests iterate semantics
2. **Create r12-approval-quorum.yaml** — 30 lines, tests standalone approve
3. **Create r13-decision-routing.yaml** — 50 lines, tests decision routing
4. **Enhance r02** — replace 17 cli steps with `iterate` node
5. **Spec update** — Add §05 appendix defining the 6 reference tools and 8 builtin stubs

### Phase 6 Deliverables (Tool Runtime)
1. **Write 6 reference tool binaries** (Go, ~1000 lines total):
   - `gert-test-fail` (100 lines)
   - `gert-test-slow` (50 lines)
   - `gert-test-json-emitter` (100 lines)
   - `gert-test-jsonrpc-server` (300 lines)
   - `gert-test-mcp-server` (500 lines)
   - `gert-test-stub` (100 lines) — generic stub for all builtin tools
2. **Write 14 `.tool.yaml` definitions** (6 reference + 8 builtin stubs)
3. **Compile builtin stubs into gert binary** — embed 8 builtin `.tool.yaml` files at build time

### Phase 11 Deliverables (Evidence & Replay)
1. **Generate golden traces for r11, r12, r13** — ensures all 14 step types have trace coverage
2. **Add HMAC verification tests** — validates chaining for all step types

### Phase 13 Deliverables (Acceptance)
1. **Run full corpus** — all 13 runbooks (r01–r13) must execute and produce valid traces
2. **Coverage report** — 14/14 step types covered, 3/3 transports covered (stdio/jsonrpc/mcp)

---

## Decision Points

### Should builtin tools be real or stubs?
**Recommendation:** Stubs for v2.0. Real implementations deferred to v2.1.  
**Rationale:** Real Slack/PagerDuty/AWS integrations require API keys, network access, and external dependencies. This bloats the test environment. Stubs satisfy tool resolution and transport testing without external dependencies.

### Should reference tools be Go binaries or shell scripts?
**Recommendation:** Go binaries.  
**Rationale:** Cross-platform compatibility (Windows, macOS, Linux). Shell scripts would require bash/sh availability and complicate Windows CI. Go binaries are self-contained, fast to build, and platform-agnostic.

### Should MCP reference server implement full MCP spec?
**Recommendation:** Minimal compliance — `initialize`, `tools/list`, `tools/call`, `tools/cancel`.  
**Rationale:** Full MCP spec includes resources, prompts, and sampling — not used by gert. Minimal server tests the critical path (tool discovery and invocation) without implementing unused features.

---

## Spec Gaps Identified

1. **§05 Tool Runtime** does NOT define what's in the "built-in tool registry"
   - Add appendix §05-A: Built-in Tool Catalog (8 stubs)
   - Add appendix §05-B: Reference Tools for Testing (6 tools)

2. **§03 Schema vNext** does NOT include example `.tool.yaml` for each transport
   - Add 3 examples: stdio (echo), stdio-jsonrpc (jsonrpc-test), mcp (mcp-test)

3. **§11 Evidence & Replay** does NOT specify test fixtures for replay
   - Add note: "Replay tests require runbook coverage for all 14 step types"

---

**End of Analysis**
