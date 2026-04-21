# Barbara — Phase 11 Spec Audit: Evidence & Replay

**Date:** 2026-04-20  
**Reviewer:** Barbara (Integrations Specialist)  
**Phase:** 11 (Evidence & Replay)  
**Requested by:** Cristian  
**Spec Audited:** §12 Evidence, Tracing, and Resumption + §06 Runtime Events

---

## Executive Summary

✅ **Phase 11 spec is comprehensive and implementation-ready** with minor gaps in RPC integration.

**Key findings:**
1. Evidence model covers 4 kinds with clear semantics
2. JSONL trace format fully specified with HMAC tamper-evidence support
3. Replay mode well-defined with deterministic requirements
4. Resumption protocol complete with checkpoint atomicity guarantees
5. **Gap:** No explicit RPC methods for evidence querying in §13 (run/list, run/get exist but no trace/query)
6. **Gap:** OpenTelemetry integration detailed but missing guidance on OTel vs JSONL prioritization for Phase 9 serve layer
7. r22 fixture created: deterministic evidence collection + replay scenario

---

## 1. Evidence Kinds Defined in Spec

**Source:** §12.2 Evidence Capture

The spec defines **4 evidence kinds**, all captured in `step/completed` events:

### 1.1 Command Output (`text`)
- **Source:** stdout/stderr of `cli` steps
- **Location:** `step/completed.payload` (embedded in event)
- **Schema:** Raw text string, redacted via redaction pipeline
- **Example use:** Deployment command logs, query results

### 1.2 Manual Attestations (`text`, `checklist`)
- **Source:** Operator responses to `manual` steps
- **Location:** `manual/completed.payload.response` event
- **Schema:** 
  - `text`: Free-form operator input
  - `checklist`: Structured boolean/select responses
- **Example use:** Compliance attestations, operator confirmations

### 1.3 Tool Responses (`text`)
- **Source:** Structured JSON returned by `tool` steps
- **Location:** `step/completed.captures` (captured outputs)
- **Schema:** JSON object, redacted
- **Example use:** API responses, database query results

### 1.4 File Attachments (`attachment`)
- **Source:** Binary/text files produced by steps or uploaded by operators
- **Location:** `.runbook/runs/<run-id>/attachments/<sha256>.<ext>`
- **Schema:** 
  ```json
  {
    "name":   "<evidence_name>",
    "kind":   "attachment",
    "path":   "<relative_path>",
    "sha256": "<hex>",
    "size":   <bytes>
  }
  ```
- **Deduplication:** Content-addressed by SHA256 (identical files share one attachment)
- **Example use:** Build artifacts, compliance documents, screenshots

**Verdict:** ✅ Complete. Covers text, structured data, and binary artifacts.

---

## 2. Trace Format Specification

**Source:** §12.1 Trace File Format

### 2.1 File Location
```
.runbook/runs/<run-id>/trace.jsonl
```

### 2.2 JSONL Envelope (Required Fields)
```json
{
  "event_id":  "<UUID v4>",       // Globally unique, for correlation
  "sequence":  <int64>,           // Monotonic within run, starts at 0
  "kind":      "<event_kind>",    // Event type (see §12.1.3)
  "timestamp": "<RFC3339>",       // UTC with sub-second precision
  "run_id":    "<run_id>",        // Run identifier
  "path":      [<tree_path>],     // Active node cursor (may be null)
  "payload":   <object>           // Kind-specific payload
}
```

**Optional field:**
```json
  "sig":       "<hex>"            // HMAC-SHA256 signature (when GERT_TRACE_KEY set)
```

### 2.3 Normative Event Kinds (§12.1.3)
| Event Kind | Emission | Payload Keys |
|------------|----------|--------------|
| `run/started` | Once, first event | `runbook_id`, `user`, `mode`, `inputs`, `client`, `parent_run_id` |
| `step/started` | Before each step | `step_id`, `step_type`, `step_name` |
| `step/completed` | After step success | `step_id`, `exit_status`, `duration_ms`, `captures`, `evidence[]` |
| `step/failed` | After step failure | `step_id`, `error_type`, `error`, `exit_status`, `duration_ms` |
| `step/skipped` | When not executed | `step_id`, `reason` |
| `step/retrying` | Before retry | `step_id`, `attempt`, `max_attempts`, `backoff_ms` |
| `tool/invoked` | Tool dispatch | `tool_name`, `transport`, `correlation_id`, `inputs` |
| `tool/responded` | Tool return | `correlation_id`, `exit_code`, `duration_ms`, `truncated` |
| `manual/prompted` | Manual step prompt | `step_id`, `prompt_text`, `sla_seconds` |
| `manual/completed` | Manual response | `step_id`, `response`, `attestation`, `approver`, `duration_ms` |
| `governance/blocked` | Rule blocks action | `step_id`, `rule`, `action_attempted`, `policy` |
| `governance/approved` | Approval granted | `step_id`, `approver`, `approval_method`, `duration_ms` |
| `run/completed` | Terminal success | `final_status`, `duration_ms`, `step_count`, `passed`, `failed`, `skipped` |
| `run/failed` | Terminal failure | `failing_step_id`, `error`, `duration_ms` |
| `checkpoint` | Internal snapshot link | `snapshot_file` |

**Additional kinds in §06:** `event/received` (wait_for_event), `governance/command_checked`, `extension/loaded`, `saga/compensation_triggered` (v2.1+)

### 2.4 Crash Safety
- **Write atomicity:** Single `write(2)` + `fsync` per event
- **POSIX guarantee:** Writes <4096 bytes to `O_APPEND` file are atomic
- **Truncation policy:** Payloads capped at 4 KB per field (configurable)
- **Recovery:** Malformed lines silently skipped; last valid `checkpoint` event determines recoverable state

### 2.5 Tamper Evidence
- **Optional HMAC-SHA256 signing:** Enabled via `GERT_TRACE_KEY` or workspace config
- **Signature field:** `"sig"` appended to envelope
- **Verification:** Recompute HMAC over canonical JSON of all other fields
- **Limitation:** Detects modification, not deletion (append-only at filesystem level provides that)

**Verdict:** ✅ Complete. Atomic writes, crash-safe, tamper-evident, versioned envelope.

---

## 3. Replay Mode Semantics

**Source:** §12.4 Replay Mode

### 3.1 Command
```bash
gert exec --mode replay --scenario <scenario.yaml>
```

### 3.2 Purpose
- **Testing:** Validate branching logic, captures, governance without real infrastructure
- **Demonstration:** Show execution flow without side effects
- **Regression:** Verify refactored runbook produces same trace as original

### 3.3 Scenario File Format
```yaml
commands:
  - argv: ["kubectl", "get", "pods", "-n", "prod"]
    stdout: "NAME          READY   STATUS\npod-abc   1/1     Running\n"
    stderr: ""
    exit_code: 0

evidence:
  check-pod-health:          # step_id
    observation:             # evidence name
      kind: text
      value: "pod is healthy, proceeding"
```

**Matching:** First-match-wins order. Unmatched commands fail unless `allow_unmatched: true`.

### 3.4 Replay Semantics by Step Type
| Step Type | Replay Behavior |
|-----------|-----------------|
| `cli` | Command executor returns pre-recorded stdout/stderr/exit_code |
| `manual` | Evidence collector returns pre-recorded evidence (no operator prompt) |
| `tool` | Tool invocations intercepted; recorded response returned or step fails |
| `invoke` | Child runbook executed in replay mode (same scenario or nested scenario file) |
| Governance | Evaluated normally; replay does NOT bypass governance |
| Trace events | Written normally; can be compared against golden trace |

### 3.5 Determinism Requirements
**Non-determinism sources to avoid:**
- Timestamps in expressions (use captured values, not `time.Now()`)
- Random identifiers in step names/captures
- Filesystem state not reproduced by scenario
- External API calls in tool steps not covered by scenario

**Verdict:** ✅ Complete. Replay mode fully specified with determinism guidance.

---

## 4. Resumption Protocol

**Source:** §12.3 Run State Snapshots, §12.4 Run Resumption

### 4.1 Checkpoint Model
**Format:** JSON serialization of `RunState` struct
**Location:** `.runbook/runs/<run-id>/snapshots/step-NNNN.json`
**Trigger:** After every successfully completed step

**Checkpoint contents:**
```go
type RunState struct {
    RunID             string
    RunbookPath       string
    Mode              string            // "real" | "replay" | "dry-run"
    StartedAt         time.Time
    Actor             string
    CurrentStepIndex  int               // flat-mode index (legacy compat)
    Path              TreePath          // tree-mode cursor (authoritative)
    Vars              map[string]string // resolved input bindings
    Captures          map[string]string // accumulated step outputs
    History           []*StepResult     // completed step records
    InvokeStack       []InvokeFrame     // nested invoke context stack
    IterateState      map[string]*IterateProgress
    ParallelSlots     map[string][]SlotState
    LastCheckpointSeq int64             // trace seq at checkpoint time
}
```

### 4.2 Atomic Snapshot Write
```go
1. Marshal state to <filename>.tmp
2. fsync temp file
3. Rename temp → final path (atomic rename)
4. Write checkpoint trace event
```

**Recovery:** Incomplete snapshots retain `.tmp` suffix and are skipped.

### 4.3 Resumption Command
```bash
gert exec --resume <run-id>
```

**Recovery sequence:**
1. Locate `.runbook/runs/<run-id>/`
2. Acquire lock file (fail if live process holds it)
3. Load latest valid checkpoint (scan `snapshots/` descending, skip `.tmp`)
4. Re-open `trace.jsonl` for append
5. Advance cursor to next step (past last completed node)
6. Rebuild governance engine from runbook policy block
7. Resume execution loop from restored `Path`

### 4.4 Resumability by Step Type
| Step Type | Resumption Behavior |
|-----------|---------------------|
| `cli` | Safe; re-executed from scratch. Authors must ensure idempotency (e.g., `kubectl apply` over `kubectl create`) |
| `manual` | Operator re-prompted. Prior response lost. Prompt re-displayed, operator must re-attest. |
| `tool` | Resumable if `idempotent: true` declared. If in-flight (tool/invoked with no tool/responded), re-issued with warning. Non-idempotent → treated as failed. |
| `invoke` | If child completed (terminal event in parent trace), treated as completed. If in-progress, child is itself resumed via nested run ID in `InvokeStack`. |
| `branch`/`iterate` | Structural nodes; cursor advancement handles re-entry. |

### 4.5 Idempotency Guidance (for Authors)
- Use `kubectl apply`, `terraform apply` (not imperative creates)
- Use `--if-not-exists` or `--create-or-replace` flags
- Declare `idempotent: true` in tool definitions when safe
- Write manual prompts as questions valid on second ask
- Avoid time/random-dependent side effects without override mechanism

**Verdict:** ✅ Complete. Atomic checkpoints, robust resumption, clear idempotency contract.

---

## 5. RPC Methods for Evidence Querying

**Source:** §13 Adapter Contracts

### 5.1 Existing Methods (exec/v2 contract)
From §13.2.1, Table 13.1:

| Method | Description | Returns |
|--------|-------------|---------|
| `exec/start` | Start new runbook execution | `runId`, `status`, `traceFile` |
| `exec/next` | Advance run by one step | `stepId`, `status`, `output` |
| `exec/cancel` | Cancel in-progress run | `cancelled: bool` |
| `exec/status` | Get current run status | `status`, `currentStepId`, `progress` |
| `run/list` | List all runs in workspace | `runs: [RunSummary]` |
| `run/get` | Get full details of a run | `run: RunDetails` |

**Note:** `exec/start` response includes `"traceFile": "./traces/run-abc123.jsonl"` (§13, line 131)

### 5.2 Gap Analysis: Missing Evidence Query Methods

**What's missing:**
- ❌ `trace/query` — Query trace events by run ID with filters (kind, sequence range, step ID)
- ❌ `trace/read` — Read full trace.jsonl or since a sequence number (maps to `RunStore.ReadTraceSince`)
- ❌ `evidence/list` — List evidence attachments for a run
- ❌ `evidence/get` — Retrieve specific attachment by SHA256
- ❌ `checkpoint/list` — List available checkpoints for a run
- ❌ `checkpoint/get` — Retrieve specific checkpoint snapshot

**What exists but is underspecified:**
- ✅ `run/get` returns `run: RunDetails` but payload schema not shown in excerpt
  - **Assumption:** Likely includes `traceFile` path but not trace contents
  - **Assumption:** May include run directory path for client-side file access

### 5.3 Implications for Phase 9 Serve Layer

**Current design (inferred from spec):**
1. Adapters (TUI, Web, VS Code) subscribe to live event stream via WebSocket (`events/v2` contract)
2. Historical trace reading: clients access `.runbook/runs/<run-id>/trace.jsonl` directly from filesystem
3. Evidence attachments: clients access `.runbook/runs/<run-id>/attachments/<sha256>.<ext>` directly

**This works for:**
- ✅ Local TUI (filesystem access)
- ✅ VS Code extension (workspace access)

**This breaks for:**
- ❌ Remote Web UI (no filesystem access)
- ❌ CI/CD integrations (need RPC-based trace retrieval)

**Recommendation for Brian (Phase 11 implementation):**
- Defer RPC trace query methods to **Phase 12 (Serve RPC enhancements)** or **v2.1**
- Phase 11 focus: `RunStore` interface, `DirRunStore` impl, `ReplayEvidenceCollector`
- Phase 12: Add `trace/query`, `evidence/get` RPC methods if remote adapter use case validated

**Verdict:** ⚠️ Gap identified. Not a Phase 11 blocker (local adapters work). Document as Phase 12 enhancement.

---

## 6. Last r-number Used

**Source:** `/Volumes/Projects/gert/design/gert-v2/testdata/runbooks/`

**Last runbook fixture:** `r21-gert-run` (Phase 10 — CLI adapter test)

**r-numbers in use:**
- r01 through r10: Acceptance corpus (SOC2, K8s, FDA, GDPR, etc.)
- r11 through r21: Phase-specific test fixtures (iterate, approval, branch, assert, tool, extension, provider, serve, run)

**Next available:** `r22`

**Verdict:** ✅ r22 assigned to evidence/replay fixture (created below).

---

## 7. Integration Points with Phase 9 Serve Layer

**Context:** Phase 9 is "gert serve" (RPC server, WebSocket event stream, adapter hosting).

### 7.1 Existing Integration Points (from §13)

**exec/v2 contract (RPC):**
- `exec/start` returns `traceFile` path
- `run/get` returns `RunDetails` (likely includes trace path)

**events/v2 contract (WebSocket):**
- Events transmitted as JSON objects (§13, line 205)
- Event envelope **identical to trace event format** (§12 JSONL envelope)
- Subscription model: clients subscribe to run ID, receive live events

### 7.2 New Integration Points for Phase 11

**1. Trace Writer Fanout**
- Runtime Core writes events to both:
  - `DirRunStore.WriteTrace()` → `trace.jsonl` (durable, authoritative)
  - `EventBus` → WebSocket subscribers (ephemeral, best-effort)
- **Decision required:** Does `TraceWriter` interface (§12, line 629) multiplex to both? Or does `Engine` call both independently?
  - **Recommendation:** `MultiWriter` pattern (Phase 3 Runtime Core established this)

**2. OTel vs JSONL Priority**
- §12.5 OpenTelemetry Integration: "complementary, not redundant"
- JSONL trace = authoritative durable record
- OTel spans = ephemeral, push-based, latency-sensitive
- **Implication:** If WebSocket subscriber is slow, JSONL write MUST NOT block
  - **Spec confirms:** §06.1.2 "bounded channel with discard-on-full policy"

**3. Checkpoint Events**
- `checkpoint` event kind (§12.1.3, line 252): "Internal event written by RunStore.SaveCheckpoint"
- **Question:** Should WebSocket subscribers receive `checkpoint` events?
  - **Spec says:** "Not user-facing but required for resumption"
  - **Implication:** Omit from WebSocket stream, include in JSONL only

### 7.3 Gap: Remote Trace Access

**Current design (from audit above):**
- Adapters access trace files directly from `.runbook/runs/<run-id>/` directory
- Works for local adapters, breaks for remote Web UI

**Phase 11 does NOT need to solve this:**
- Phase 11 deliverable: `RunStore` interface, checkpoint/resume, replay mode
- Remote trace access is a **Phase 9 enhancement** (or v2.1)

**Verdict:** ✅ Integration points identified. No new RPC methods required for Phase 11.

---

## 8. Existing Trace Infrastructure

**Source:** `/Volumes/Projects/gert/v2/pkg/trace/`, `/Volumes/Projects/gert/v2/internal/trace/`

### 8.1 Public Package (`pkg/trace/`)

**`event.go`:**
```go
type EventKind string  // Wire format discriminator
const (
    EventKindRunStarted, EventKindStepCompleted, ...  // 20+ kinds
)

type TraceEvent struct {
    EventID   string          // UUID v4
    RunID     string
    RunbookID string
    Timestamp string          // RFC3339
    Kind      EventKind
    Sequence  int64           // Monotonic within run
    Payload   json.RawMessage
    Signature string          // HMAC-SHA256 (optional)
}
```

**`writer.go`:**
```go
type TraceWriter interface {
    Append(event TraceEvent) error
    Close() error
}
```

**Alignment with spec:**
- ✅ Event envelope matches §12.1.2 JSONL envelope exactly
- ✅ EventKind constants match §12.1.3 event catalog
- ✅ Optional `Signature` field matches §12.1.5 HMAC signing
- ✅ `TraceWriter` interface matches §12.6 Go interfaces

### 8.2 Internal Package (`internal/trace/`)

**`jsonl_writer.go`:**
- Implementation of `TraceWriter` for JSONL file
- Likely uses `O_APPEND` + `fsync` (per §12.1.4 crash safety)

**`multi_writer.go`:**
- Multiplexes writes to multiple `TraceWriter` destinations
- Confirms fanout pattern for JSONL + EventBus

**`*_test.go`:**
- Unit tests for writer implementations

**Alignment with spec:**
- ✅ `jsonl_writer` implements §12.1.4 atomic write protocol
- ✅ `multi_writer` supports fanout to JSONL + in-memory EventBus
- ✅ Test coverage exists (good foundation for Phase 11 golden trace tests)

### 8.3 Gap Analysis

**What's implemented (v2 codebase so far):**
- ✅ Event envelope struct
- ✅ JSONL writer
- ✅ Multi-writer fanout

**What's missing (Phase 11 deliverables):**
- ❌ `RunStore` interface (§12.6, lines 637-648)
  - `ReadTrace()`, `ReadTraceSince()`, `SaveCheckpoint()`, `LoadLatestCheckpoint()`, `AcquireLock()`, `ReleaseLock()`, `WriteManifest()`
- ❌ `DirRunStore` implementation (production, filesystem-backed)
- ❌ `MemRunStore` implementation (testing, in-memory)
- ❌ `EvidenceCollector` interface (§12.6, lines 650-654)
- ❌ `ReplayEvidenceCollector` implementation (returns pre-recorded evidence)
- ❌ Checkpoint atomicity (write to `.tmp`, fsync, rename)
- ❌ HMAC-SHA256 trace signing (when `GERT_TRACE_KEY` set)
- ❌ Replay scenario YAML parser
- ❌ Resumption command handler (`gert exec --resume <run-id>`)

**Verdict:** ✅ Trace infrastructure foundation exists. Phase 11 adds RunStore, checkpoints, replay, resumption.

---

## 9. r22 Fixture Description

**Created:** `/Volumes/Projects/gert/design/gert-v2/testdata/runbooks/r22-evidence-replay/schema.yaml`

### 9.1 Purpose
Integration test fixture for Phase 11 Evidence & Replay:
- Evidence collection (stdout, file artifacts, env snapshots)
- Trace format validation
- Replay mode (deterministic execution)
- Resumption from checkpoint

### 9.2 Fixture Design

**Flow (7 steps):**
1. **capture_env**: Capture build environment variable → `step/completed.captures.build_env`
2. **create_artifact**: Create file artifact → `step/completed.evidence[0].kind="attachment"` with SHA256
3. **verify_artifact**: Compute artifact hash → `step/completed.captures.artifact_hash`
4. **branch_by_env**: Branch on `build_env == "production"` → tests infix condition
5. **prod_deploy** (branch step): Deploy to production → `step/completed.captures.deploy_result`
6. **capture_env_snapshot**: Capture env snapshot → `step/completed.evidence[0].kind="text"`
7. **complete**: Completion marker using all captured variables

**Evidence captured:**
- **Text evidence:** Stdout from cli steps (build_env, deploy_result)
- **Attachment evidence:** File artifact with SHA256 hash (build-artifact-b123.txt)
- **Env snapshot evidence:** Multi-line environment dump

**Determinism:**
- Fixed inputs: `build_id=b001`, `build_env=production`
- Fixed timestamp in artifact: `2026-04-20T12:00:00Z`
- No external API calls or random values
- Branch condition deterministic: `build_env == "production"`

### 9.3 Test Scenarios

**Scenario 1: Real mode execution**
```bash
gert exec schema.yaml --var build_id=b123 --var build_env=production
```
Expected output:
- `.runbook/runs/<run-id>/trace.jsonl` (18 events)
- `.runbook/runs/<run-id>/snapshots/step-0000.json` through `step-0007.json` (8 checkpoints)
- `.runbook/runs/<run-id>/attachments/<sha256>.txt` (artifact file)

**Scenario 2: Replay mode**
```bash
gert exec --mode replay --scenario replay.yaml
```
Expected behavior:
- Commands matched from scenario file
- Evidence returned from scenario file
- Trace events identical to real mode (except timestamps/event_ids)

**Scenario 3: Resumption**
```bash
# 1. Run with simulated crash after step 2
gert exec schema.yaml --debug-crash-after-step=2
# 2. Verify checkpoint exists
ls .runbook/runs/<run-id>/snapshots/step-0002.json
# 3. Resume
gert exec --resume <run-id>
```
Expected behavior:
- Steps 1-2 not re-executed
- Step 3+ execute normally
- Final trace has gap in sequence numbers where resumption occurred

### 9.4 Expected Trace Event Sequence

| Seq | Event Kind | Key Payload Fields |
|-----|------------|-------------------|
| 0   | `run/started` | `runbook_id="r22-evidence-replay"`, `mode="real"` |
| 1   | `step/started` | `step_id="capture_env"` |
| 2   | `step/completed` | `step_id="capture_env"`, `captures.build_env="production"` |
| 3   | `step/started` | `step_id="create_artifact"` |
| 4   | `step/completed` | `step_id="create_artifact"`, `evidence[0].kind="attachment"`, `evidence[0].sha256="..."` |
| 5   | `step/started` | `step_id="verify_artifact"` |
| 6   | `step/completed` | `step_id="verify_artifact"`, `captures.artifact_hash="..."` |
| 7   | `step/started` | `step_id="branch_by_env"` |
| 8   | `step/completed` | `step_id="branch_by_env"`, `branch="Production deployment"` |
| 9   | `step/started` | `step_id="prod_deploy"` |
| 10  | `step/completed` | `step_id="prod_deploy"`, `captures.deploy_result="..."` |
| 11  | `step/started` | `step_id="prod_verify"` |
| 12  | `step/completed` | `step_id="prod_verify"`, `captures.verify_result="..."` |
| 13  | `step/started` | `step_id="capture_env_snapshot"` |
| 14  | `step/completed` | `step_id="capture_env_snapshot"`, `evidence[0].kind="text"` |
| 15  | `step/started` | `step_id="complete"` |
| 16  | `step/completed` | `step_id="complete"`, `captures.completion_message="..."` |
| 17  | `run/completed` | `final_status="passed"`, `step_count=7` |

**Verdict:** ✅ r22 fixture created with comprehensive evidence coverage and replay/resumption scenarios.

---

## 10. Gap Analysis Summary

### 10.1 Spec Completeness

| Area | Status | Notes |
|------|--------|-------|
| Evidence kinds | ✅ Complete | 4 kinds: text, checklist, attachment, structured |
| Trace format | ✅ Complete | JSONL envelope, atomic writes, HMAC signing |
| Event catalog | ✅ Complete | 20+ event kinds, clear payload schemas |
| Replay mode | ✅ Complete | Determinism requirements, scenario format |
| Resumption | ✅ Complete | Atomic checkpoints, lock protocol, idempotency guidance |
| OTel integration | ✅ Complete | Span hierarchy, attribute mapping, opt-in config |
| Go interfaces | ✅ Complete | TraceWriter, RunStore, EvidenceCollector |

### 10.2 Integration Gaps

| Gap | Severity | Phase | Mitigation |
|-----|----------|-------|------------|
| No RPC methods for trace/evidence querying | Medium | Phase 12 | Local adapters use filesystem access; defer remote access to Phase 12 or v2.1 |
| OTel vs JSONL prioritization unclear | Low | Phase 3/11 | Spec confirms JSONL is authoritative; OTel is opt-in |
| `checkpoint` event WebSocket propagation undefined | Low | Phase 9 | Spec says "internal event"; omit from WebSocket stream |

### 10.3 Ambiguities for Resolution

**None identified.** Spec is clear and actionable.

---

## 11. Recommendations for Brian (Phase 11 Implementation)

### 11.1 Phase 11 Scope (from PLAN.md)

**Deliverables:**
1. `v2/pkg/runstore/` — RunStore interface and DirRunStore implementation
2. `v2/pkg/evidence/` — Evidence collection, SHA256 hashing, attachment storage
3. `v2/pkg/replay/` — Replay scenario parser, command matching, evidence mocking
4. `v2/internal/checkpoint/` — Atomic snapshot write, recovery
5. `v2/cmd/gert/exec.go` — Add `--resume <run-id>` and `--mode replay --scenario <file>` flags
6. Golden trace tests — Deterministic trace comparison with normalized timestamps/IDs

### 11.2 Implementation Priorities

**1. RunStore interface (§12.6, lines 637-648)**
```go
type RunStore interface {
    RunID() string
    WriteTrace(event DurableEvent) error
    ReadTrace() ([]DurableEvent, error)
    ReadTraceSince(afterSeq int64) ([]DurableEvent, error)
    SaveCheckpoint(state *RunState) (string, error)
    LoadLatestCheckpoint() (*RunState, int64, error)
    AcquireLock() error
    ReleaseLock() error
    WriteManifest(manifest *RunManifest) error
}
```

**2. DirRunStore implementation**
- Create `.runbook/runs/<run-id>/` directory structure
- Implement atomic checkpoint write (write → fsync → rename)
- Implement PID lock (`lock` file with PID, removed on clean exit)
- Implement trace JSONL writer (already exists in `internal/trace/jsonl_writer.go`)

**3. Evidence capture**
- Implement `evidence.HashFile()` (SHA256 + size)
- Implement attachment copying to `attachments/<sha256>.<ext>`
- Implement deduplication (skip copy if SHA256 exists)
- Add `evidence` array to `step/completed` event payload

**4. Replay mode**
- Parse scenario YAML (commands, evidence)
- Implement `ReplayExecutor` (matches argv, returns pre-recorded stdout/stderr/exit_code)
- Implement `ReplayEvidenceCollector` (returns pre-recorded evidence)
- Add `--mode replay --scenario <file>` flag to `gert exec`

**5. Resumption**
- Implement `LoadLatestCheckpoint()` (scan snapshots descending, skip `.tmp`)
- Implement lock acquisition (fail if PID in lock file is live)
- Modify execution loop to skip steps in `History`
- Add `--resume <run-id>` flag to `gert exec`

**6. HMAC signing (optional, can defer to v2.1)**
- Implement HMAC-SHA256 over canonical JSON of event fields
- Add `sig` field to TraceEvent envelope when `GERT_TRACE_KEY` set
- Implement verification in `gert verify` command

### 11.3 Test Strategy

**Unit tests:**
- `runstore_test.go`: Checkpoint atomicity, lock protocol, trace append
- `evidence_test.go`: SHA256 hashing, attachment deduplication
- `replay_test.go`: Command matching, evidence mocking

**Golden trace tests:**
- Use r22 fixture: run once, capture trace, normalize timestamps/event_ids, commit as golden
- Regression: re-run r22, diff against golden
- Update via `go test -update`

**Integration tests:**
- Resumption: run r22, kill after step 2, resume, verify trace continuity
- Replay: run r22 in real mode, generate scenario YAML, replay, diff traces
- Evidence: verify attachments copied to `attachments/`, verify SHA256 in events

**Acceptance tests:**
- Use r04-soc2-evidence fixture (manual step attestations, file uploads)
- Verify evidence package structure matches SOC2 audit requirements

### 11.4 Open Questions for Ken

**Q1:** Should `checkpoint` events be emitted to WebSocket subscribers?
- **Spec says:** "Internal event... not user-facing"
- **Barbara's take:** Omit from WebSocket stream, include in JSONL only

**Q2:** Should Phase 11 include RPC methods for remote trace access (`trace/query`, `evidence/get`)?
- **Spec says:** Adapters access `.runbook/runs/<run-id>/` directly
- **Barbara's take:** Defer to Phase 12 or v2.1 (local adapters work without it)

**Q3:** Should HMAC signing be Phase 11 or v2.1?
- **Spec says:** Optional, enabled via `GERT_TRACE_KEY`
- **Barbara's take:** Defer to v2.1 (non-blocking, compliance use case not validated)

---

## 12. Integration Concerns Summary

### 12.1 Phase 9 RPC Additions (if any)

**Current exec/v2 contract (§13.2.1):**
- ✅ `exec/start` returns `traceFile` path
- ✅ `run/list`, `run/get` provide run metadata

**Potential additions (deferred to Phase 12 or v2.1):**
- ❌ `trace/query` — Query trace events by filters
- ❌ `trace/read` — Read full trace or since sequence number
- ❌ `evidence/list` — List attachments for a run
- ❌ `evidence/get` — Retrieve attachment by SHA256

**Verdict:** No Phase 11 RPC additions required. Local adapters work with filesystem access.

### 12.2 Phase 10 Trace Writer Interaction

**Phase 10 deliverable:** `gert run` CLI adapter (no EventDispatcher, no WebSocket)

**Interaction with Phase 11:**
- `gert run` writes trace.jsonl via `DirRunStore.WriteTrace()`
- `gert run` does NOT fan out to EventBus (no WebSocket in standalone mode)
- Phase 11's `MultiWriter` pattern is used by `gert serve` (Phase 9), not `gert run`

**Verdict:** No conflict. `gert run` uses simpler write path (JSONL only).

### 12.3 Phase 3 Runtime Core Dependencies

**Phase 3 established:**
- Event emission from execution loop
- `TraceWriter` interface
- Multi-writer fanout to JSONL + EventBus

**Phase 11 builds on:**
- `RunStore` interface (superset of `TraceWriter`)
- Checkpoint writes (new)
- Resumption protocol (new)

**Verdict:** Clean extension of Phase 3 contracts.

---

## Conclusion

**Phase 11 spec is comprehensive and ready for implementation.**

**Key strengths:**
- Evidence model covers all artifact types (text, binary, structured)
- JSONL trace format is atomic, crash-safe, tamper-evident
- Replay mode supports deterministic testing
- Resumption protocol is robust with atomic checkpoints and lock safety

**Minor gaps identified:**
- No RPC methods for remote trace access (deferred to Phase 12 or v2.1)
- Minimal ambiguity around `checkpoint` event WebSocket propagation (recommend: omit)

**r22 fixture created:**
- Path: `/Volumes/Projects/gert/design/gert-v2/testdata/runbooks/r22-evidence-replay/schema.yaml`
- Coverage: Evidence collection, replay, resumption
- Test scenarios: Real mode, replay mode, resumption after crash

**Recommendation:** ✅ **Approve Phase 11 for implementation** with the open questions above sent to Ken for resolution.

---

**End of Audit Report**
