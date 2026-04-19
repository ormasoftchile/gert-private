# gert v2 Schema Stress Test — Architectural Conclusions

**Author:** Ken (Software Architect)  
**Date:** 2026-04-18  
**Status:** PART A COMPLETE (Independent Review), PART B SKIPPED (John's translations not yet available)

---

## Executive Summary

**Overall Schema Readiness Verdict: NEEDS TARGETED FIXES**

The gert v2 schema is architecturally sound for 80% of real-world operational runbooks, but has **3 systemic gaps** that block clean translation of 6/10 corpus runbooks. The gaps are addressable with targeted schema additions; no fundamental redesign is required.

- **Runbooks that PASS cleanly:** 4/10 (40%)
- **Runbooks that PASS with workarounds:** 4/10 (40%)
- **Runbooks that FAIL current schema:** 2/10 (20%)

**Top 3 Systemic Gaps:**

| # | Gap | Severity | Affected Runbooks | Impact |
|---|-----|----------|-------------------|--------|
| 1 | **No business-day timeout** — `timeout` and `approvals.timeout` use wall-clock duration, not calendar-aware business days | CRITICAL | 3, 7, 8 | All enterprise/regulated workflows require business-day SLAs |
| 2 | **No M-of-N quorum approval** — `approvals.min` requires N approvals but can't express "3-of-5 must approve" where pool is larger than required count | CRITICAL | 7, 8, 10 | Multi-party governance (board, dual-attestation) blocked |
| 3 | **No external event trigger/resume** — No mechanism to pause runbook and resume when external webhook arrives (e.g., FDA clearance, PagerDuty ack) | HIGH | 8, 9 | Long-running runbooks with external dependencies blocked |

---

## Per-Runbook Verdict Table

| # | Runbook | Domain | Complexity | Verdict | Key Gap |
|---|---------|--------|------------|---------|---------|
| 1 | K8s Pod Incident Response | SRE | Branching+Loop | **PASS WITH NOTES** | G6: JSON path branching requires verbose workaround |
| 2 | Production Deployment w/ Canary | DevOps | Saga+Parallel | **PASS WITH NOTES** | G6: Cross-branch convergence requires careful step structure |
| 3 | New Employee Onboarding | HR/IT | Parallel+Multi-party | **FAIL** | G2: Business-day timeout not supported; G4: M-of-N approval missing |
| 4 | SOC2 Evidence Collection | Compliance | Nested Invoke | **PASS** | Clean; invoke pattern works well |
| 5 | Security Breach Containment | Security | Branch+Parallel | **PASS WITH NOTES** | G6: Nested parallel within branch is verbose but expressible |
| 6 | Database Migration | Data Engineering | Linear+Saga | **PASS** | Clean; compensate pattern works well |
| 7 | Financial Approval Chain | Finance | Multi-level approval | **FAIL** | G2: Business-day timeout; G4: M-of-N quorum approval |
| 8 | Medical Device Release (FDA) | Regulated | Multi-attestation | **PASS WITH NOTES** | G4: Dual approval exists; G3: 90-day external pause lacks trigger |
| 9 | On-Call Escalation Ladder | SRE | Loop+Auto | **PASS WITH NOTES** | G6: Sequential iterate works; external ack trigger verbose |
| 10 | GDPR Data Deletion | Compliance | Parallel+Verify | **PASS** | Clean; best-effort parallel + verification loop work well |

**Summary:** 4 PASS, 4 PASS WITH NOTES, 2 FAIL

---

## PART A: Independent Architectural Predictions

### Runbook 1: Kubernetes Pod Incident Response

**Schema Constructs Predicted:**
- `type: branch` with 4 arms for failure mode routing (CrashLoopBackOff → fix config, OOMKilled → scale, etc.)
- `type: cli` for kubectl commands with `capture:` for stdout/JSON
- `type: collector` with `approvals:` for human gates
- `retry:` on health check steps
- `type: compensate` for config rollback

**Difficulty Areas:**
1. **Dynamic branching on JSON** — Branch condition needs to evaluate `pod.status.conditions[?(@.type=="Ready")].status`. Go templates can do this with nested range/index, but it's verbose.
2. **Escalation timeout** — Step 4b has "timeout: 5 minutes → escalate to senior SRE". Current `approvals.escalate_to` exists but unclear if timeout swaps the approver pool dynamically.
3. **Evidence artifact bundling** — Step 8 bundles all artifacts into tar.gz. Schema captures individual outputs but no first-class "bundle artifacts" operation.

**Prediction:** PASS WITH NOTES — All constructs available, JSON branching is verbose but works.

---

### Runbook 2: Production Deployment with Canary + Rollback

**Schema Constructs Predicted:**
- `type: compensate` with `on: failure` for rollback registration
- `type: parallel` with `join.wait_for: all` for concurrent canary deploy
- `iterate` with `until:` for metric convergence monitoring
- `type: branch` for promote vs. rollback decision
- `type: assert` for pre-deployment checks

**Difficulty Areas:**
1. **Saga scope** — Compensation registered at step 4 applies to steps 5-9. Current `compensate` registers at point of declaration; need to verify scope extends forward correctly.
2. **Cross-branch convergence** — Step 11 runs after BOTH promote and rollback branches. Schema needs explicit join point or the branches must merge before step 11.
3. **Timeout triggers rollback** — Step 9a approval timeout should trigger rollback branch (step 10), not just fail. Need `on_timeout: goto` or similar.

**Prediction:** PASS WITH NOTES — Saga pattern works; cross-branch join requires careful tree structure.

---

### Runbook 3: New Employee Onboarding

**Schema Constructs Predicted:**
- `type: collector` with rich field types (text, email, date, dropdown)
- `type: parallel` with 5 branches for concurrent provisioning
- `approvals:` on multiple steps with timeouts
- `type: branch` for conditional access provisioning

**Difficulty Areas:**
1. **Business-day timeout** — Steps 2, 3, 4d, 6, 8, 10 use "business days" for timeouts. Schema `timeout` field is a duration string (e.g., "48h"), not calendar-aware. **CRITICAL GAP.**
2. **M-of-N approval** — Step 10 requires BOTH HR director AND IT security officer (2-of-2). Current `approvals.min` doesn't express the pool size. What if 3 people could approve but 2 must?
3. **Dynamic assignee** — Step 6 is assigned to the employee (from step 1 email). Schema needs `assignee: "{{ .employee_email }}"` support.
4. **Autocomplete input** — Step 1 "Manager" field uses autocomplete from employee directory. Schema `type: text` doesn't support autocomplete/typeahead from external API.

**Prediction:** FAIL — Business-day timeout is blocking; workaround (calculate hours from business-day calendar) is error-prone and defeats purpose.

---

### Runbook 4: SOC2 Evidence Collection

**Schema Constructs Predicted:**
- `type: invoke` for 15 sub-runbook calls
- `imports:` map for sub-runbook resolution
- `invoke.inputs` and `invoke.outputs` for data passing
- `type: collector` with `approvals:` for attestation
- Artifact capture from each sub-runbook

**Difficulty Areas:**
1. **Bulk invoke with consistent inputs** — All 15 sub-runbooks need audit_id. Schema supports `invoke.inputs` per call but no "broadcast" pattern.
2. **Partial failure handling** — If CC3.1 fails but others succeed, is partial evidence acceptable? `invoke.gate.stop_if` handles this but semantics unclear.
3. **Artifact aggregation** — Step 9 combines all sub-runbook outputs. Need to capture file paths from each invoke and aggregate.

**Prediction:** PASS — Invoke pattern works well; 15 invoke steps are verbose but correct.

---

### Runbook 5: Security Breach Containment

**Schema Constructs Predicted:**
- `type: choice` for severity triage
- `type: branch` for 4-way severity routing
- `type: parallel` nested inside branch arms for concurrent containment
- `type: compensate` for rollback if production breaks
- `approvals:` with escalation for exec approval

**Difficulty Areas:**
1. **Choice with timeout default** — Step 2 defaults to "High" on 10-minute timeout. Schema `choice` doesn't have `default` that activates on timeout (only user skip).
2. **Nested parallel within branch** — Step 4c is parallel (4 tasks) inside "Critical Containment" branch. Schema supports nested structures but untested depth.
3. **Concurrent forensics + containment** — Step 8 runs in parallel with steps 4-7. Need top-level parallel with two independent branches, each containing sequential steps.
4. **Mid-runbook goto** — Step 6b decision routes to High Containment (step 5) which is EARLIER. Schema `decision.routes[].goto` typically goes forward; backward goto breaks DAG assumption.

**Prediction:** PASS WITH NOTES — Nested parallel works; backward goto is a design limitation (acceptable—restructure to avoid).

---

### Runbook 6: Database Migration

**Schema Constructs Predicted:**
- `type: assert` for pre-flight checks
- `type: cli` with `retry:` for transient failure handling
- `type: compensate` for multi-step rollback (DROP COLUMN, DROP INDEX, pg_restore)
- `iterate` with `until:` for health convergence
- `type: parallel` for smoke tests
- `type: approval` with timeout → rollback

**Difficulty Areas:**
1. **Time-based wait** — Step 7 sleeps until maintenance window datetime. Schema `delay:` is duration; no `delay_until:` field for datetime. **Minor gap.**
2. **Multi-step compensation** — Step 9 registers 3-step rollback with fallback logic. Schema `compensate.steps` is a list but no conditional logic within compensation.
3. **Convergence-based iterate** — Step 13 loops until 10 CONSECUTIVE healthy checks. Schema `iterate.until` can express this with a counter variable but it's verbose.

**Prediction:** PASS — Core saga pattern works; datetime wait is minor workaround (calculate delay at runtime).

---

### Runbook 7: Financial Approval Chain

**Schema Constructs Predicted:**
- `type: collector` for purchase request form
- `type: approval` at 5 levels (manager → finance → director/VP/CFO → board)
- `type: branch` for amount-based routing ($50k/$100k/$500k thresholds)
- `approvals:` with M-of-N for board (3-of-5)

**Difficulty Areas:**
1. **Dynamic approver lookup** — Steps 2, 5, 6, 7 look up approvers from HR system/org chart at runtime. Schema `approvals.roles` is static; need provider-resolved approvers.
2. **Business-day timeout** — All approvals use business days. **CRITICAL GAP** (same as Runbook 3).
3. **M-of-N quorum** — Step 7b requires 3-of-5 board members. Schema `approvals.min: 3` but no `approvals.pool` to define the 5 eligible approvers. **CRITICAL GAP.**
4. **Conditional nested step** — Step 7b (board) only runs if amount >= $1M. Within the CFO branch, need conditional execution. Schema `when:` guard works.

**Prediction:** FAIL — Both business-day timeout AND M-of-N quorum are blocking gaps.

---

### Runbook 8: Medical Device Software Release (FDA)

**Schema Constructs Predicted:**
- `type: approval` with checklist for 7+ attestations
- `type: choice` for regulatory determination
- `type: branch` for submission type routing
- `type: collector` for 510(k) file uploads
- Long-running pause (90 days) for FDA review

**Difficulty Areas:**
1. **Conditional step** — Step 3 (clinical validation) only required if software affects clinical functionality. Schema `when:` guard works.
2. **Choice with rationale** — Step 5 regulatory determination must include rationale for audit. Schema `choice.variable` stores selection but no mandatory rationale field.
3. **90-day external pause** — Step 8e waits for FDA clearance (external event). Schema has no webhook/callback trigger to resume runbook. **HIGH GAP.**
4. **Digital signature metadata** — Steps need 21 CFR Part 11 compliant signatures (identity, timestamp, certificate). Schema `approvals` captures approval but not cryptographic signature details.

**Prediction:** PASS WITH NOTES — Most works; FDA pause requires manual resume (acceptable for v2.0).

---

### Runbook 9: On-Call Escalation Ladder

**Schema Constructs Predicted:**
- `iterate` with convergence for each escalation level (primary, secondary, manager, director)
- `type: tool` or `type: extension` for PagerDuty API calls
- Polling pattern within iterate for acknowledgment check

**Difficulty Areas:**
1. **Sequential iterate with early exit** — Steps 3, 4, 5, 6 are sequential iterates; once ANY converges, skip remaining. Schema can express this with a flag variable and `when:` guards.
2. **Polling within iterate** — Each pass calls PagerDuty API and checks status. Schema `tool` + `capture` + `iterate.until` handles this.
3. **Cross-iterate convergence** — Once any level acknowledges, proceed to step 10. Need shared variable (`acknowledged: true`) and guards on subsequent iterates.
4. **External event trigger** — Steps 3b, 4b, etc. wait for PagerDuty acknowledgment. Could use polling (verbose) or webhook (not supported).

**Prediction:** PASS WITH NOTES — Polling-based escalation works; external webhook trigger would be cleaner but not blocking.

---

### Runbook 10: GDPR Data Deletion

**Schema Constructs Predicted:**
- `type: parallel` for searching 6 systems
- `type: parallel` with `join.on_failure: continue` for best-effort deletion
- `iterate` for backup deletion and verification loop
- `type: approval` for legal review and DPO attestation
- `type: collector` for identity verification

**Difficulty Areas:**
1. **Best-effort parallel** — Step 6 deletion should continue even if some systems fail. Schema `join.on_failure: continue` supports this.
2. **Variable-length data inventory** — Step 4 outputs vary (0 to 1000s of records). Schema variables are strings; complex data needs JSON serialization.
3. **Verification with generous retry** — Step 8 allows 1-hour retry for replication lag. Schema `retry.interval` can be "1h" but retry for expected delay vs. transient error is semantically different.

**Prediction:** PASS — All patterns are expressible; best-effort parallel and verification loop work well.

---

## Systemic Gaps (appear in 3+ runbooks)

### Gap S1: Business-Day Timeout

**Classification:** G2 — Missing Field  
**Severity:** CRITICAL  
**Affected Runbooks:** 3 (Onboarding), 7 (Financial), 8 (FDA)  
**Appears in:** 15+ individual steps across the corpus

**Current State:**
- `timeout` field accepts duration string (e.g., "48h", "5m")
- `approvals.timeout` same behavior
- No calendar awareness

**What's Missing:**
- Business-day calculation (skip weekends, holidays)
- Configurable calendar (different regions have different holidays)
- Duration unit: "2bd" (2 business days)

**Proposed Fix:**
Add to §03 Common Step Fields:

```yaml
timeout: "48h"                    # wall-clock (existing)
timeout_business_days: 2          # NEW: business days
timeout_calendar: "us-federal"    # NEW: holiday calendar reference
```

Add to §11 Governance:
- Default calendar configuration at project level
- Holiday calendar definition schema

**Affected Sections:** §03 (Schema), §11 (Governance), §02 (Runtime—evaluation logic)

---

### Gap S2: M-of-N Quorum Approval

**Classification:** G4 — Missing Interaction Model  
**Severity:** CRITICAL  
**Affected Runbooks:** 7 (Financial, board approval), 8 (FDA, dual attestation), 10 (GDPR, DPO + legal)  
**Appears in:** 5+ individual approval steps

**Current State:**
- `approvals.min: N` requires N approvals to proceed
- `approvals.roles` lists eligible roles
- All eligible approvers can approve; no upper bound

**What's Missing:**
- Explicit pool definition: "These 5 people are the board"
- Quorum semantics: "3 of these 5 must approve"
- Distinction between AND (all must) vs. OR (any of) vs. M-of-N

**Proposed Fix:**
Extend §03 `approvals:` field:

```yaml
approvals:
  mode: quorum                    # NEW: quorum | all | any
  min: 3                          # minimum required
  pool:                           # NEW: explicit approver pool
    - user: ceo@company.com
    - user: cfo@company.com
    - role: board-member          # expands to all users with this role
  pool_size: 5                    # NEW: optional; for M-of-N display "3 of 5"
```

**Affected Sections:** §03 (Schema), §14 (JSON-RPC—approval protocol), §11 (Governance—RBAC)

---

### Gap S3: External Event Trigger/Resume

**Classification:** G3 — Missing Flow Construct  
**Severity:** HIGH  
**Affected Runbooks:** 8 (FDA clearance), 9 (PagerDuty ack)  
**Appears in:** 4+ steps requiring external system callbacks

**Current State:**
- Runbooks can pause waiting for human input (approval, collector)
- No mechanism to pause waiting for external system event
- Polling within iterate is the workaround

**What's Missing:**
- Webhook endpoint registration: "Resume this run when webhook X is called"
- External event correlation: match incoming webhook to paused run
- Timeout on external wait (different from approval timeout)

**Proposed Fix:**
Add new step type to §03:

```yaml
- step:
    id: wait_for_fda
    type: external_event           # NEW step type
    title: Wait for FDA clearance
    external_event:
      source: fda-estar            # registered webhook source
      match:                       # correlation criteria
        submission_id: "{{ .fda_submission_id }}"
      timeout: "90d"
      on_timeout: escalate
    capture:
      clearance_status: json.status
```

Add to §02 Architecture:
- Webhook endpoint in API/Adapter layer
- Run correlation by event match criteria
- Event → run resume dispatch

**Affected Sections:** §03 (Schema), §02 (Architecture), §06 (Events), §13 (JSON-RPC API)

---

## Isolated Gaps (appear in 1-2 runbooks)

### Gap I1: Datetime-Based Delay

**Classification:** G2 — Missing Field  
**Runbook:** 6 (Database Migration, step 7)  
**In-Scope for v2:** No — Acceptable v2.1 deferral

**Description:** Step 7 waits until maintenance window start time (datetime), not for a duration. Current `delay:` field accepts duration only.

**Workaround:** Calculate delay at runtime: `delay: "{{ .delay_until_maintenance }}"` where the variable is computed in a prior CLI step.

**Recommendation:** Add `delay_until: "{{ .maintenance_start_time }}"` in v2.1.

---

### Gap I2: Choice with Timeout Default

**Classification:** G2 — Missing Field  
**Runbook:** 5 (Security Breach, step 2)  
**In-Scope for v2:** Yes — Should fix

**Description:** Step 2 asks security officer to triage severity; if no response in 10 minutes, default to "High" (err on side of caution).

**Current State:** `choice` has `default:` but it only activates if user explicitly skips; timeout doesn't trigger default.

**Proposed Fix:** Add `timeout_default:` field to `choice` step type.

---

### Gap I3: Backward Goto in Decision

**Classification:** G3 — Missing Flow Construct  
**Runbook:** 5 (Security Breach, step 6b)  
**In-Scope for v2:** No — Intentional limitation

**Description:** Step 6b decision wants to route to "High Containment" (step 5) which was BEFORE the decision point.

**Design Decision:** gert v2 assumes DAG execution; backward jumps create cycles and complicate trace/replay semantics.

**Recommendation:** Document as intentional limitation. Workaround: restructure runbook to avoid backward references (extract High Containment as sub-runbook, invoke it from multiple points).

---

### Gap I4: Dynamic Approver Lookup

**Classification:** G2 — Missing Field  
**Runbooks:** 7 (Financial), 3 (Onboarding)  
**In-Scope for v2:** Yes — Should fix

**Description:** Approver is not a static role but looked up at runtime from HR system or org chart.

**Current State:** `approvals.roles` is a static list of role names.

**Proposed Fix:** Allow provider-resolved approvers:

```yaml
approvals:
  min: 1
  roles:
    - from: hr.manager_of          # provider-resolved
      args:
        employee: "{{ .requester_email }}"
```

---

### Gap I5: Signature Metadata for Compliance

**Classification:** G2 — Missing Field  
**Runbook:** 8 (FDA)  
**In-Scope for v2:** No — Acceptable v2.1 deferral

**Description:** 21 CFR Part 11 requires digital signature metadata (identity, timestamp, certificate thumbprint, signature algorithm).

**Current State:** `approvals` records who approved and when but not cryptographic signature details.

**Recommendation:** Defer to v2.1 compliance module. For v2.0, capture in `metadata:` field for audit purposes.

---

## Schema Improvement Recommendations

### Priority: CRITICAL (blocks real use)

| # | What to Add/Change | Affected Section | Effort |
|---|-------------------|------------------|--------|
| 1 | **Business-day timeout support** — Add `timeout_business_days`, `timeout_calendar` fields to step and approval schemas | §03, §11 | Medium |
| 2 | **M-of-N quorum approval** — Add `approvals.mode`, `approvals.pool`, `approvals.pool_size` fields | §03, §14 | Medium |

### Priority: IMPORTANT (workaround exists but ugly)

| # | What to Add/Change | Affected Section | Effort |
|---|-------------------|------------------|--------|
| 3 | **External event trigger** — Add `type: external_event` step type with webhook registration | §03, §02, §06, §13 | High |
| 4 | **Choice with timeout default** — Add `timeout_default:` field to choice step | §03 | Low |
| 5 | **Dynamic approver lookup** — Allow `approvals.roles[].from: provider` syntax | §03 | Medium |

### Priority: NICE-TO-HAVE

| # | What to Add/Change | Affected Section | Effort |
|---|-------------------|------------------|--------|
| 6 | **Datetime-based delay** — Add `delay_until:` field | §03 | Low |
| 7 | **Signature metadata** — Add `approvals.signature_metadata` schema for compliance | §03, §11 | Medium |

---

## Design Limitations (Acceptable)

The following are intentional boundaries of gert v2, not failures:

### L1: No Backward Goto (DAG-Only Execution)

**Limitation:** Decision `goto` can only reference steps that appear LATER in the flow tree.

**Rationale:** Backward jumps create cycles, complicating:
- Trace replay determinism
- Checkpoint/resume semantics
- Visual workflow representation (no loops)

**Workaround:** Extract cyclic patterns as sub-runbooks; invoke multiple times.

**Status:** Documented as intentional design constraint.

---

### L2: No Dynamic Step Generation

**Limitation:** Steps are defined statically in YAML; cannot generate steps at runtime based on data.

**Rationale:** Static step graph enables:
- Pre-execution validation
- Deterministic replay
- Visual planning before execution

**Workaround:** Use `iterate` over a dynamic list; each pass executes the same step structure with different variable bindings.

**Status:** Documented as intentional design constraint.

---

### L3: No Live-Streaming Dashboard

**Limitation:** gert v2 is CLI/TUI-first; no rich graphical dashboard during execution.

**Rationale:** Cross-platform CLI portability is a core design goal. Rich dashboards are adapter responsibilities (VS Code extension, future web UI).

**Workaround:** Include external dashboard URLs in `prose:` section; operators open dashboards separately.

**Status:** Documented as intentional design constraint.

---

### L4: No Weighted Voting in Approvals

**Limitation:** All approvers have equal weight; cannot express "CEO vote counts as 2."

**Rationale:** Weighted voting adds significant complexity for a rare use case.

**Workaround:** Model weighted voting as multiple approval steps or require explicit CEO approval as a separate gate.

**Status:** Documented as intentional design constraint; may revisit in v2.2+.

---

## Verdict for ormasoftchile

**Plain language summary:**

The gert v2 schema is ready for **SRE incident response**, **DevOps deployments with rollback**, **compliance evidence collection**, and **fully-automated escalation workflows**. These cover the core operational use cases.

The schema is **not yet ready** for **enterprise governance workflows** that require business-day SLAs and multi-party quorum approvals (finance, HR, regulated industries). Two critical gaps block clean translation:

1. **Business-day timeout** — All enterprise approval workflows use "business days," not wall-clock hours
2. **M-of-N quorum approval** — Board approvals, dual attestation, and separation-of-duties patterns need explicit pool definitions

**To reach production readiness for enterprise use cases, the team needs to:**

1. Add business-day timeout support (§03 schema + §11 governance calendar config)
2. Add quorum approval model (§03 schema + §14 JSON-RPC approval protocol)

These are additive changes (backward compatible) and can be implemented in a focused sprint before release.

**Recommended action:** Prioritize S1 and S2 fixes before declaring schema "implementation-ready." External event trigger (S3) can be deferred to v2.1 as it has acceptable polling workarounds.

---

## Appendix: Gap-to-Runbook Cross-Reference

| Gap | RB1 | RB2 | RB3 | RB4 | RB5 | RB6 | RB7 | RB8 | RB9 | RB10 |
|-----|-----|-----|-----|-----|-----|-----|-----|-----|-----|------|
| S1: Business-day timeout | | | ✗ | | | | ✗ | ✗ | | |
| S2: M-of-N quorum | | | ✗ | | | | ✗ | ✗ | | ✗ |
| S3: External event trigger | | | | | | | | ✗ | ✗ | |
| I1: Datetime delay | | | | | | ✗ | | | | |
| I2: Choice timeout default | | | | | ✗ | | | | | |
| I3: Backward goto | | | | | ✗ | | | | | |
| I4: Dynamic approver | | | ✗ | | | | ✗ | | | |
| I5: Signature metadata | | | | | | | | ✗ | | |

✗ = Gap affects this runbook

---

*This analysis is based solely on the corpus and schema spec. John's translations file was not available at time of writing. When available, scores should be cross-referenced to validate these predictions.*
