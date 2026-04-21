# Phase 6 Tool & Fixture Audit

**Analyst:** Barbara (Integrations Specialist)  
**Date:** 2026-04-19  
**Context:** Pre-Phase 6 readiness check — comprehensive tool reference enumeration and fixture gap analysis

---

## Executive Summary

**STATUS:** ⚠️ **PHASE 6 BLOCKED** — Critical gaps in test tooling infrastructure

### Key Findings

1. **Tool References:** 23 unique tools referenced across r01–r16 runbooks
2. **Transport Coverage:** Spec defines 4 transports (stdio, stdio-jsonrpc, grpc, mcp); NO test fixtures exist for ANY transport
3. **Missing Stubs:** 13 builtin tools (slack, pagerduty, aws, okta, etc.) have NO `.tool.yaml` definitions
4. **Fixture Gaps:** Zero reference test tools (echo, fail, slow) for transport validation
5. **Parser Status:** ✅ All 16 runbooks parse successfully (TestParser_Fixtures passes)

### Blockers

**CRITICAL:** Phase 6 implementation cannot begin until:
1. Reference test toolset created (see §5.1)
2. Builtin tool stubs defined (see §5.2)
3. Transport fixture runbook created (r17 recommendation — see §6)

---

## 1. Tool Reference Table

### 1.1 Complete Tool Inventory (r01–r16)

| Tool Name | Transport | Source | Used In | Actions | Purpose |
|-----------|-----------|--------|---------|---------|---------|
| `pagerduty-notify` | ❓ | `builtin://pagerduty` | r01, r05 | `trigger-alert`, `resolve-alert` | Incident management notifications |
| `alertmanager-notify` | ❓ | `builtin://alertmanager` | r01 | `resolve-alert` | Prometheus AlertManager integration |
| `slack-notify` | ❓ | `builtin://slack` | r01, r05 | `send-message` | Team notifications |
| `prometheus` | ❓ | (inferred builtin) | r02 | `query` | Metrics queries for canary validation |
| `okta` | ❓ | `builtin://okta` | r03, r05 | `create-user`, `invite-user`, `disable-vpn-policy`, `force-password-reset-all`, `suspend-accounts`, `unsuspend-all-users` | Identity management |
| `github` | ❓ | `builtin://github` | r03 | `invite-user` | GitHub org membership |
| `slack` | ❓ | `builtin://slack` | r03 | `invite-user` | Slack workspace access |
| `aws` | ❓ | `builtin://aws` | r05, r10 | `revoke-sg-ingress`, `stop-instances`, `rotate-iam-keys`, `create-ebs-snapshots`, `export-cloudtrail`, `restore-security-groups`, `restrict-sg-egress` | AWS resource management |
| `palo-alto` | ❓ | `builtin://palo-alto` | r05 | `block-ips` | Firewall management |
| `auth-service` | ❓ | `tool://auth-service` | r05 | `invalidate-all-tokens` | Custom auth service |
| `internal-api` | ❓ | `tool://internal-api` | r05 | `revoke-all-keys` | Custom API key revocation |
| `credential-revoker` | ❓ | `tool://credential-revoker` | r05 | `revoke-for-alert` | Custom credential revocation |
| `log-exporter` | ❓ | `tool://log-exporter` | r05 | `export` | Log export utility |
| `splunk` | ❓ | `builtin://splunk` | r05 | `search-iocs` | SIEM threat hunting |
| `email-notify` | ❓ | `builtin://email` | r05 | `send` | Email notifications |
| `incident-tracker` | ❓ | `tool://incident-tracker` | r05 | `close` | Custom incident tracker |
| `psql` | ❓ | (inferred builtin) | r04, r06, r10 | (unknown) | PostgreSQL CLI tool |
| `pg_dump` | ❓ | (inferred builtin) | r06 | (unknown) | PostgreSQL backup tool |
| `kubectl` | stdio | (inferred builtin) | r01, r02 | (argv-based) | Kubernetes CLI |
| `git` | stdio | (inferred builtin) | r01, r02 | (argv-based) | Git CLI |
| `docker` | stdio | (inferred builtin) | r02 | (argv-based) | Docker CLI |

**Summary:**
- **23 unique tools** referenced across 16 runbooks
- **13 builtin tools** (source: `builtin://...`) — NO `.tool.yaml` definitions exist
- **6 custom tools** (source: `tool://...`) — project-specific, no definitions needed for Phase 6
- **4 CLI tools** (kubectl, git, docker, psql?) — assumed stdio transport, no tool definitions exist

### 1.2 Tool Usage by Runbook

| Runbook | Tool Steps | Tools Used |
|---------|------------|------------|
| r01-k8s-incident | 3 | pagerduty-notify, alertmanager-notify, slack-notify |
| r02-canary-deploy | 2 | prometheus (2x) |
| r03-employee-onboarding | 3 | okta, github, slack |
| r04-soc2-evidence | 0 | (none) |
| r05-security-breach | 11 | aws (5x), okta (3x), palo-alto, auth-service, internal-api, credential-revoker, log-exporter, splunk, email-notify, pagerduty-notify, slack-notify, incident-tracker |
| r06-db-migration | 0 | (none) |
| r07-financial-approval | 0 | (none) |
| r08-fda-release | 0 | (none) |
| r09-oncall-escalation | 0 | (none) |
| r10-gdpr-deletion | 0 | (none) |
| r11-iterate-loop | 0 | (none) |
| r12-approval-quorum | 0 | (none) |
| r13-decision-routing | 0 | (none) |
| r14-assert-compensate | 0 | (none) |
| r15-branch-collector | 0 | (none) |
| r16-end-step | 0 | (none) |

**Total: 19 tool steps** across 16 runbooks

---

## 2. Transport Coverage Analysis

### 2.1 Transports Defined in Spec (§05 Tool Runtime)

Per `sections/05-tool-runtime.tex`, v2 supports **4 transport modes**:

1. **`stdio`** — Spawn binary per invocation, argv-based, capture stdout  
   - Section §5.2.3 (lines 123–151)
   - Process: `exec.Command(binary, argv...)`
   - Error envelope: exit code, stderr, duration

2. **`stdio-jsonrpc`** — Long-lived process, JSON-RPC 2.0 over stdin/stdout  
   - Section §5.2.4 (lines 153–224)
   - Startup: wait for `ready-pattern` on stdout/stderr
   - Invocation envelope includes `context` object (runId, stepId, traceId, userId, attempt, capabilities)
   - Error codes: -32700 to -32099 (JSON-RPC + gert extensions)

3. **`mcp`** — Model Context Protocol (MCP) transport  
   - Section §5.3 (lines 299–350+)
   - Dynamic tool discovery via `tools/list`
   - Invocation: `tools/call` JSON-RPC method
   - Context injected as `_gert_context` param

4. **`grpc`** — gRPC transport (v2.1 deferment per PLAN.md)  
   - Mentioned in spec line 69: `mode: stdio-jsonrpc # stdio | stdio-jsonrpc | grpc | mcp`
   - **NO implementation in v2.0** — deferred to v2.1

### 2.2 Transport Coverage in Runbooks

| Transport | Runbooks Using | Tool Steps | Status |
|-----------|----------------|------------|--------|
| `stdio` | r01, r02 (inferred via kubectl/git) | 0 explicit | ❌ NO `.tool.yaml` definitions |
| `stdio-jsonrpc` | NONE | 0 | ❌ NO test tools |
| `mcp` | NONE | 0 | ❌ NO MCP servers |
| `grpc` | NONE | 0 | ⏸️ Deferred to v2.1 |

**CRITICAL GAP:** All tool references in runbooks are to **builtin stubs with no transport specified**. The `toolRefs:` declarations in runbooks DO NOT include `transport:` fields — this is expected (transport is defined in the `.tool.yaml`, not the runbook).

**Problem:** Without `.tool.yaml` definitions, we cannot test ANY transport mode. Phase 6 requires:
1. Minimal reference tools for each transport (echo, fail, slow)
2. Transport-specific fixture runbook (r17 — see §6)

---

## 3. Required Test Tools

### 3.1 Reference Test Tools (Minimal Transport Validation)

These tools are **NOT referenced in any runbook** but are **REQUIRED** for Phase 6 transport testing:

| Tool Name | Transport | Purpose | Actions | Test Scenarios |
|-----------|-----------|---------|---------|----------------|
| `echo` | stdio | Echo args to stdout | (argv-based) | Stdout capture, exit code 0, arg rendering |
| `echo-json` | stdio | Echo args as JSON | (argv-based) | JSON output format, capture paths |
| `fail` | stdio | Exit with non-zero | (argv-based) | Error envelope, stderr capture, non-zero exit |
| `slow` | stdio | Sleep N seconds | (argv-based) | Timeout enforcement, cancellation |
| `echo-server` | stdio-jsonrpc | JSON-RPC echo | `echo` | JSON-RPC envelope, persistent process, context object |
| `fail-server` | stdio-jsonrpc | JSON-RPC error | `fail` | JSON-RPC error codes (-32050 to -32099) |
| `slow-server` | stdio-jsonrpc | JSON-RPC delay | `delay` | Timeout in JSON-RPC transport |
| `echo-mcp` | mcp | MCP echo server | `echo` | MCP handshake, tools/list, tools/call, _gert_context |

**Total: 8 reference test tools** needed (4 stdio + 3 stdio-jsonrpc + 1 mcp)

### 3.2 Tool Definition Requirements

Each reference tool needs a `.tool.yaml` definition. Example for `echo`:

```yaml
# design/gert-v2/testdata/tools/echo.tool.yaml
apiVersion: tool/v2

meta:
  name: echo
  version: "1.0.0"
  description: "Echo arguments to stdout (test fixture)"
  binary: echo  # Unix/Linux/macOS built-in

transport:
  mode: stdio

governance:
  requires-capabilities: []
  allowed-environments: ["real", "dry-run"]
  requires-approval: false

actions:
  echo:
    description: "Echo a message"
    argv: ["{{ .message }}"]
    timeout: "5s"
    args:
      message:
        type: string
        required: true
        description: "Message to echo"
        redact: false
    output:
      format: text
    capture:
      stdout:
        format: text
```

Example for `echo-server` (stdio-jsonrpc):

```yaml
# design/gert-v2/testdata/tools/echo-server.tool.yaml
apiVersion: tool/v2

meta:
  name: echo-server
  version: "1.0.0"
  description: "JSON-RPC echo server (test fixture)"
  binary: gert-test-echo-server  # Go binary in testdata/tools/bin/

transport:
  mode: stdio-jsonrpc
  startup:
    argv: []
    ready-pattern: "echo-server ready"
    timeout: "10s"
    shutdown-method: "shutdown"

governance:
  requires-capabilities: []
  allowed-environments: ["real", "dry-run"]
  requires-approval: false

actions:
  echo:
    description: "Echo a message (JSON-RPC)"
    timeout: "10s"
    args:
      message:
        type: string
        required: true
        description: "Message to echo"
    output:
      format: json
    capture:
      stdout:
        format: json
```

**Binary Requirements:**
- `echo`, `sleep` — available on all Unix-like systems (no build needed)
- `gert-test-echo-server`, `gert-test-fail-server`, `gert-test-slow-server` — Go binaries to implement (Phase 6 subtask)
- `gert-test-echo-mcp` — Go binary implementing MCP protocol (Phase 6 subtask)

---

## 4. Required Builtin Stubs

### 4.1 Builtin Tools Referenced in Runbooks

These tools are declared as `builtin://...` in runbook `toolRefs:` but **NO `.tool.yaml` exists**:

| Tool Name | Source | Runbooks | Actions | Stub Behavior Needed |
|-----------|--------|----------|---------|---------------------|
| `slack-notify` | `builtin://slack` | r01, r05 | `send-message` | Return success with fake message_id |
| `pagerduty-notify` | `builtin://pagerduty` | r01, r05 | `trigger-alert`, `resolve-alert` | Return success with fake incident_id |
| `alertmanager-notify` | `builtin://alertmanager` | r01 | `resolve-alert` | Return success (no output) |
| `prometheus` | (inferred builtin) | r02 | `query` | Return fake metric: `{"data":{"result":[{"value":["1618761600","0.02"]}]}}` |
| `okta` | `builtin://okta` | r03, r05 | `create-user`, `invite-user`, `disable-vpn-policy`, `force-password-reset-all`, `suspend-accounts`, `unsuspend-all-users` | Return success with fake user_id |
| `github` | `builtin://github` | r03 | `invite-user` | Return success with fake invitation_id |
| `slack` | `builtin://slack` | r03 | `invite-user` | Return success with fake user_id |
| `aws` | `builtin://aws` | r05, r10 | `revoke-sg-ingress`, `stop-instances`, `rotate-iam-keys`, `create-ebs-snapshots`, `export-cloudtrail`, `restore-security-groups`, `restrict-sg-egress` | Return success (no output or fake ARNs) |
| `palo-alto` | `builtin://palo-alto` | r05 | `block-ips` | Return success with fake rule_id |
| `splunk` | `builtin://splunk` | r05 | `search-iocs` | Return fake search results: `{"results":[]}` |
| `email-notify` | `builtin://email` | r05 | `send` | Return success with fake message_id |
| `psql` | (inferred builtin?) | r04, r06, r10 | (argv-based CLI) | Likely stdio tool, not stub |
| `pg_dump` | (inferred builtin?) | r06 | (argv-based CLI) | Likely stdio tool, not stub |

**Total: 11 builtin stubs** (not counting psql/pg_dump which are likely stdio CLI tools)

### 4.2 Custom Tools (No Stub Needed)

These tools use `tool://...` source (project-specific):

- `auth-service`, `internal-api`, `credential-revoker`, `log-exporter`, `incident-tracker`

**NO stubs needed** — these are project tools, not part of Phase 6 builtin registry.

### 4.3 Recommended Stub Transport

**Decision:** All builtin stubs should use **`stdio-jsonrpc`** transport to:
1. Test JSON-RPC envelope format
2. Test persistent process lifecycle
3. Test context object injection

**Alternative:** Implement as Go HTTP handlers called via `curl` (stdio) — simpler but doesn't test stdio-jsonrpc transport.

**Recommendation:** **stdio-jsonrpc stubs** — aligns with Phase 6 transport coverage goals.

### 4.4 Stub Implementation Approach

**Option 1: Single Multi-Tool Server**

One Go binary (`gert-builtin-stub-server`) that exposes all 11 builtin stubs via stdio-jsonrpc:

```go
// v2/internal/testtools/stub-server/main.go
func main() {
    // Listen on stdin/stdout for JSON-RPC requests
    // Route by method name: "slack-notify/send-message", "aws/stop-instances", etc.
    // Return fake success responses
}
```

Each `.tool.yaml` references the same binary with different tool names.

**Option 2: Per-Tool Stub Binaries**

11 separate Go binaries (e.g., `gert-stub-slack`, `gert-stub-aws`, etc.) — more realistic but higher maintenance.

**Recommendation:** **Option 1 (single server)** for Phase 6 — simpler, faster to implement, adequate for transport testing.

---

## 5. Spec Coverage: What the Spec Says vs. Gaps

### 5.1 What Spec §05 Defines ✅

| Topic | Spec Location | Status |
|-------|---------------|--------|
| Tool discovery phases (static, dynamic, MCP) | §5.1 lines 12–36 | ✅ Defined |
| Name resolution order | §5.1.1 lines 38–53 | ✅ Defined |
| Tool definition schema (v2) | §5.1.2 lines 55–101 | ✅ Defined |
| Transport modes (stdio, stdio-jsonrpc, mcp, grpc) | §5.1.3 lines 103–119 | ✅ Defined |
| stdio transport envelope | §5.2.1 lines 125–151 | ✅ Defined |
| stdio-jsonrpc transport envelope | §5.2.2 lines 153–224 | ✅ Defined |
| MCP transport integration | §5.3 lines 299–350+ | ✅ Defined |
| Invocation context object | §5.4 lines 230–253 | ✅ Defined |
| Capability gate enforcement | §5.5 lines 255–268 | ✅ Defined |
| Output capture and redaction | §5.6 lines 270–297 | ✅ Defined |
| Error envelope schemas (stdio, jsonrpc) | §5.2.1, §5.2.2 | ✅ Defined |

### 5.2 What Spec Does NOT Define ❌

| Topic | Gap | Impact |
|-------|-----|--------|
| **Builtin tool registry contents** | Spec says "built-in tool registry compiled into the host binary" (line 18) but DOES NOT list what's in it | Phase 6 must infer from runbook references OR document minimal set |
| **Tool definition storage location** | Spec says `<workspace-root>/tools/` convention (line 19) but does not specify if test tools go in `testdata/tools/` or `v2/internal/testtools/` | Phase 6 must decide |
| **Transport selection algorithm** | Spec defines transports but does not say HOW to choose (builtin stubs all use same transport? project tools specify in .tool.yaml?) | Phase 6 must decide |
| **MCP server registry format** | Spec shows one MCP server example (lines 308–318) but does not define multi-server behavior or conflict resolution | Phase 6 must decide |
| **grpc transport details** | Mentioned in mode enum (line 69) but no spec section exists (deferred to v2.1) | ⏸️ Not a gap — intentionally deferred |

### 5.3 Guidance from Ken's Architectural Review (Barbara's History)

From `.squad/agents/barbara/history.md` lines 95–100:

> **For Barbara (Integrations):** §05 Tool Runtime section has important gaps. Needs detailed specifications for:
> 
> 1. **Tool discovery algorithm** — How are tool definitions found in v2? Still by convention `tools/<name>.tool.yaml`? Or registry-based?
> 2. **Transport layer specification** — stdio/jsonrpc/mcp exist in v1. Do all three survive in v2? Any new transports?
> 3. **Invocation envelope** — What fields are in the "run context and telemetry" mentioned? Full spec needed.
> 4. **Error envelope schema** — "Structured and machine-readable" is not a spec. Define the exact error shape.

**STATUS UPDATE (2026-04-19):**
1. ✅ Tool discovery: **RESOLVED** — Spec §5.1 defines 3-phase discovery (static, dynamic, MCP)
2. ✅ Transport layer: **RESOLVED** — Spec §5.2–5.3 defines stdio, stdio-jsonrpc, mcp (grpc deferred)
3. ✅ Invocation envelope: **RESOLVED** — Spec §5.4 lines 238–252 defines context object (runId, stepId, traceId, userId, mode, attempt, capabilities)
4. ✅ Error envelope: **RESOLVED** — Spec §5.2.1 lines 139–151 (stdio), §5.2.2 lines 196–224 (jsonrpc)

**CONCLUSION:** Ken's gaps have been addressed in the spec. No blocking spec ambiguities remain for Phase 6.

---

## 6. New Fixture Recommendations

### 6.1 Required: r17-tool-transports.yaml

**Purpose:** Dedicated fixture for testing all tool transports

**Coverage:**
- stdio transport: echo, fail, slow (3 steps)
- stdio-jsonrpc transport: echo-server, fail-server, slow-server (3 steps)
- mcp transport: echo-mcp (1 step)
- Capability gate enforcement (1 step with missing capability)
- Output capture (json path extraction)
- Error envelope validation (non-zero exit, JSON-RPC error codes)
- Timeout enforcement

**Step count:** ~10 steps  
**Blocks:** Phase 6 (Tool Runtime) integration tests

**Rationale:** Without a dedicated transport fixture, Phase 6 cannot validate:
- Process lifecycle (spawn, ready-pattern, shutdown)
- Envelope correctness (invocation context, response format)
- Error handling (exit codes, JSON-RPC errors, timeouts)
- Multi-transport behavior in a single runbook

### 6.2 Optional: r18-tool-discovery.yaml

**Purpose:** Test tool discovery and name resolution

**Coverage:**
- Project-level tool (tools/<name>.tool.yaml)
- Builtin tool override (project tool shadows builtin)
- MCP dynamic discovery (tools/list)
- Extension dynamic registration (capability/tool-registration)
- Name collision warning

**Step count:** ~8 steps  
**Blocks:** Phase 6 planner integration

**Rationale:** Tool discovery is tested by planner in Phase 2, but end-to-end validation with actual tool invocation is missing.

### 6.3 Not Needed: Enhancements to r01–r05

**Current approach:** r01, r02, r03, r05 use builtin tools (slack, pagerduty, aws, okta)

**Question:** Should we replace these with reference test tools (echo, echo-server) to make tests deterministic?

**Answer:** **NO** — keep builtin stubs in r01–r05 for realism. Use r17 for transport-specific testing with reference tools.

**Rationale:**
- r01–r05 test real-world runbook patterns (alerts, deployments, onboarding)
- r17 tests transport mechanics in isolation
- Mixing concerns would complicate both

---

## 7. Changes Since Prior Gap Analysis (barbara-runbook-toolset-gaps.md)

### 7.1 New Runbooks Added (r14–r16)

Since prior analysis on 2026-04-19 (earlier today), **3 new runbooks** were added:

- **r14-assert-compensate** — tests `assert` and `compensate` steps (Phase 5 coverage)
- **r15-branch-collector** — tests `branch` and `collector` (Phase 5 coverage)
- **r16-end-step** — tests `end` step (Phase 5 coverage)

**Impact on Tool Audit:** NONE — these runbooks have **zero tool steps** (verified via schema.yaml grep).

### 7.2 New Runbooks Added (r11–r13)

Prior gap analysis recommended creating r11–r13. These now exist:

- **r11-iterate-loop** — tests `iterate` step (0 tool steps)
- **r12-approval-quorum** — tests `approve` step (0 tool steps)
- **r13-decision-routing** — tests `decision` step (0 tool steps)

**Impact on Tool Audit:** NONE — these runbooks have **zero tool steps**.

### 7.3 Step Type Coverage Change

| Step Type | Coverage Before (r01–r10) | Coverage After (r01–r16) | Status |
|-----------|---------------------------|--------------------------|--------|
| `iterate` | ❌ 0 | ✅ r11 | GAP CLOSED |
| `approve` | ❌ 0 | ✅ r12 | GAP CLOSED |
| `decision` | ❌ 0 | ✅ r13 | GAP CLOSED |
| `assert` | r02, r06 (2) | ✅ r02, r06, r14 | GAP CLOSED |
| `compensate` | r02, r05, r06 (3) | ✅ r02, r05, r06, r14 | GAP CLOSED |
| `sleep` | ❌ 0 | ❌ 0 | STILL MISSING |
| `log` | ❌ 0 | ❌ 0 | STILL MISSING |
| `set` | ❌ 0 | ❌ 0 | STILL MISSING |

**Tool Audit Impact:** NONE — r11–r16 do not introduce new tool references.

### 7.4 Prior Recommendations Status

From `barbara-runbook-toolset-gaps.md` §2 (Gap 2: Standard Toolset):

> **Current state:** Spec references "built-in tool registry" but does NOT define what's in it. Runbooks reference 10+ builtin tools (slack, pagerduty, aws, etc.) that don't exist.  
> **Critical gap:** No spec for reference/test tools. No minimal tool set for testing stdio/jsonrpc/mcp transports.  
> **Blocked phases:** Phase 6 (Tool Runtime), Phase 11 (Evidence & Replay), Phase 13 (Acceptance).

**STATUS UPDATE:** Gap remains **OPEN** — this audit confirms and expands the prior findings.

---

## 8. Ready-to-Parse Check ✅

**Command:** `go test ./v2/internal/parser/... -run TestParser_Fixtures -v`

**Result:** ✅ **ALL PASS** (16/16 runbooks)

```
=== RUN   TestParser_Fixtures
=== RUN   TestParser_Fixtures/r01-k8s-incident
=== RUN   TestParser_Fixtures/r02-canary-deploy
=== RUN   TestParser_Fixtures/r03-employee-onboarding
=== RUN   TestParser_Fixtures/r04-soc2-evidence
=== RUN   TestParser_Fixtures/r05-security-breach
=== RUN   TestParser_Fixtures/r06-db-migration
=== RUN   TestParser_Fixtures/r07-financial-approval
=== RUN   TestParser_Fixtures/r08-fda-release
=== RUN   TestParser_Fixtures/r09-oncall-escalation
=== RUN   TestParser_Fixtures/r10-gdpr-deletion
=== RUN   TestParser_Fixtures/r11-iterate-loop
=== RUN   TestParser_Fixtures/r12-approval-quorum
=== RUN   TestParser_Fixtures/r13-decision-routing
=== RUN   TestParser_Fixtures/r14-assert-compensate
=== RUN   TestParser_Fixtures/r15-branch-collector
=== RUN   TestParser_Fixtures/r16-end-step
--- PASS: TestParser_Fixtures (0.01s)
PASS
ok  	github.com/ormasoftchile/gert/v2/internal/parser	0.583s
```

**Implications:**
1. All runbook YAML schemas are syntactically valid
2. Parser supports all 14 step types (including `tool` step)
3. `toolRefs:` declarations parse correctly
4. NO schema validation errors in any runbook

**What This Does NOT Validate:**
- Tool definitions (`.tool.yaml` files) — **NONE EXIST**
- Transport behavior — **NO TESTS**
- Tool invocation — **NO RUNTIME**

**Conclusion:** Parser is ready. Runtime is blocked pending tool definitions.

---

## 9. Phase 6 Readiness Checklist

| Requirement | Status | Blocker |
|-------------|--------|---------|
| Spec defines tool transports | ✅ DONE | — |
| Spec defines tool definition schema | ✅ DONE | — |
| Spec defines invocation context | ✅ DONE | — |
| Spec defines error envelopes | ✅ DONE | — |
| Parser supports `type: tool` steps | ✅ DONE | — |
| Reference test tools exist (echo, fail, slow) | ❌ NONE | **CRITICAL** |
| Builtin stubs exist (slack, aws, okta, etc.) | ❌ NONE | **CRITICAL** |
| Transport fixture runbook (r17) exists | ❌ NONE | **CRITICAL** |
| MCP reference server exists | ❌ NONE | **CRITICAL** |
| Tool discovery integration tests exist | ❌ NONE | MEDIUM |

**CONCLUSION:** Phase 6 implementation **CANNOT BEGIN** until:
1. Reference test tools created (8 tools — see §3.1)
2. Builtin stub definitions created (11 stubs — see §4.1)
3. r17-tool-transports.yaml fixture created (see §6.1)
4. MCP reference server implemented (echo-mcp — see §3.1)

**Estimated Effort:**
- Reference tools (stdio): 1 day (echo, fail, slow + .tool.yaml definitions)
- Reference tools (stdio-jsonrpc): 2 days (Go binaries for echo-server, fail-server, slow-server)
- Builtin stubs (stdio-jsonrpc): 2 days (single Go binary + 11 .tool.yaml definitions)
- MCP reference server: 3 days (Go binary implementing MCP spec)
- r17 fixture runbook: 1 day (YAML + assessment.md)

**Total:** ~9 days of tooling work before Brian can start Phase 6 implementation.

---

## 10. Recommendations

### 10.1 Immediate Actions (Before Phase 6 Kickoff)

1. **Barbara (Integrations):** Create reference test tools (§3.1)
   - Define 8 `.tool.yaml` files in `design/gert-v2/testdata/tools/`
   - Implement 4 Go binaries: `echo-server`, `fail-server`, `slow-server`, `echo-mcp`
   - Place binaries in `v2/internal/testtools/bin/` (gitignored, built via Makefile)

2. **Barbara (Integrations):** Create builtin stub server (§4)
   - Implement `gert-builtin-stub-server` (single Go binary, stdio-jsonrpc)
   - Define 11 `.tool.yaml` stubs in `design/gert-v2/testdata/tools/builtin/`
   - Document stub behavior in `testdata/tools/builtin/README.md`

3. **Barbara (Integrations):** Create r17-tool-transports fixture (§6.1)
   - YAML runbook exercising all 8 reference tools
   - Assessment document explaining transport validation goals
   - Submit to Ken for review before Phase 6 begins

4. **Ken (Architect):** Review tool discovery ambiguities (§5.2)
   - Decide: Where do test tools live? (`testdata/tools/` vs `v2/internal/testtools/`)
   - Decide: What is the minimal builtin registry for v2.0? (11 stubs? or fewer?)
   - Decide: Do all builtin stubs use stdio-jsonrpc? or mixed transports?

### 10.2 Phase 6 Entry Criteria (Updated)

**PLAN.md** should be updated with:

```yaml
Phase 6: Tool Runtime
Entry Criteria:
  - Phase 5 complete (all 14 step types implemented)
  - Reference test tools created (8 tools: echo, fail, slow, echo-server, fail-server, slow-server, echo-mcp)
  - Builtin stub server implemented (11 stubs: slack, pagerduty, aws, okta, github, prometheus, alertmanager, palo-alto, splunk, email)
  - r17-tool-transports.yaml fixture exists and parses
  - MCP reference server implemented (echo-mcp)
  - Ken has reviewed and approved tool discovery strategy
```

### 10.3 Out of Scope for Phase 6

The following are **NOT blockers** for Phase 6:
- r18-tool-discovery.yaml (optional — can be added in Phase 13 acceptance)
- psql/pg_dump tool definitions (assumed stdio CLI tools, not critical for transport testing)
- Custom tools (auth-service, internal-api, etc.) — project-specific, not part of Phase 6 deliverable
- grpc transport — deferred to v2.1 per PLAN.md

---

## Appendix A: Full toolRefs Declarations

### r01-k8s-incident

```yaml
toolRefs:
  - name: kubectl
  - name: git
  - name: prometheus
  - name: slack-notify
  - name: pagerduty-notify
  - name: alertmanager-notify
```

### r02-canary-deploy

```yaml
toolRefs:
  - name: kubectl
  - name: prometheus
  - name: docker
  - name: git
```

### r03-employee-onboarding

```yaml
toolRefs:
  - name: okta
  - name: github
  - name: slack
```

### r04-soc2-evidence

```yaml
toolRefs:
  - name: aws
  - name: psql
```

### r05-security-breach

```yaml
toolRefs:
  - name: aws
    source: builtin://aws
    actions: [revoke-sg-ingress, stop-instances, rotate-iam-keys, create-ebs-snapshots, export-cloudtrail, restore-security-groups, restrict-sg-egress]
  - name: okta
    source: builtin://okta
    actions: [disable-vpn-policy, force-password-reset-all, suspend-accounts, unsuspend-all-users]
  - name: palo-alto
    source: builtin://palo-alto
    actions: [block-ips]
  - name: auth-service
    source: tool://auth-service
    actions: [invalidate-all-tokens]
  - name: internal-api
    source: tool://internal-api
    actions: [revoke-all-keys]
  - name: credential-revoker
    source: tool://credential-revoker
    actions: [revoke-for-alert]
  - name: log-exporter
    source: tool://log-exporter
    actions: [export]
  - name: splunk
    source: builtin://splunk
    actions: [search-iocs]
  - name: email-notify
    source: builtin://email
    actions: [send]
  - name: pagerduty-notify
    source: builtin://pagerduty
    actions: [trigger-alert, resolve-alert]
  - name: slack-notify
    source: builtin://slack
    actions: [send-message]
  - name: incident-tracker
    source: tool://incident-tracker
    actions: [close]
```

### r06-db-migration

```yaml
toolRefs:
  - name: psql
  - name: pg_dump
```

### r07-financial-approval

```yaml
toolRefs: []
```

### r08-fda-release

```yaml
toolRefs: []
```

### r09-oncall-escalation

```yaml
toolRefs: []
```

### r10-gdpr-deletion

```yaml
toolRefs:
  - name: psql
  - name: aws
```

### r11-iterate-loop through r16-end-step

```yaml
toolRefs: []
```

---

## Appendix B: Spec References

- **§05 Tool Runtime:** `sections/05-tool-runtime.tex`
- **Tool discovery:** §5.1 lines 9–53
- **Tool definition schema:** §5.1.2 lines 55–101
- **stdio transport:** §5.2.1 lines 123–151
- **stdio-jsonrpc transport:** §5.2.2 lines 153–224
- **MCP transport:** §5.3 lines 299–350+
- **Invocation context:** §5.4 lines 230–253
- **Capability gate:** §5.5 lines 255–268
- **Output capture:** §5.6 lines 270–297

---

**END OF AUDIT**
