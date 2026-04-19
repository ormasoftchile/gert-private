# Gert v2: Research Brief on Runbooks and Governance Models
**Prepared by:** Dennis, CS Researcher  
**Date:** 2026-04-18  
**Version:** 1.0  

**Purpose:** This document synthesizes academic and industry research on runbooks, workflow orchestration, governance, and traceability to inform the gert v2 design. It serves as the foundational knowledge base for the entire design team.

---

## Executive Summary

Gert v1 has successfully implemented core features including governed runbook execution, evidence capture, JSONL trace logs, and tool/provider abstractions. The v2 design presents an opportunity to apply industry learnings and academic research to build a next-generation platform that addresses:

1. **Workflow orchestration patterns** from systems like Temporal, Argo, and Prefect
2. **Governance models** from ITIL, policy-as-code frameworks (OPA, Cedar), and compliance standards
3. **Traceability patterns** including event sourcing, deterministic replay, and distributed tracing
4. **Human-in-the-loop patterns** for approval gates, escalation, and attestation
5. **Schema design patterns** from YAML-based DSLs with versioning and self-describing contracts

This brief provides specific recommendations grounded in real-world systems and academic literature.

---

## 1. Runbook Landscape

### 1.1 Definitions and Evolution

**What is a Runbook?**

From SRE and DevOps practice:
- **Traditional Runbook:** Step-by-step procedural documentation (wiki, PDF, Google Docs) describing how to perform operational tasks or respond to incidents
- **Executable Runbook:** Machine-readable, automated procedures that can be executed programmatically with minimal manual intervention

Key properties distinguishing executable from static runbooks:
- **Codified actions** - Steps defined as scripts or orchestration workflows
- **Inputs/outputs** - Parameterized and reusable
- **Integration** - Connected to APIs, monitoring, alerting, infrastructure
- **Versioned** - Treated as code artifacts
- **Testable** - Can be validated and tested like software

**Evolution Timeline:**
1. **Paper/Wiki Era:** Manual procedures, prone to staleness and human error
2. **Script Era:** Bash/Python scripts, but no standardization or governance
3. **Executable Runbook Era:** Structured YAML/JSON definitions with governance, audit trails, and tool integration

### 1.2 Industry Tools and Systems

**AWS Systems Manager (SSM) Runbooks**
- YAML/JSON automation documents (runbooks)
- Step automation: AWS API calls, Lambda invocation, shell scripts
- Human approval steps for sensitive actions
- Cross-account/region execution
- Rate controls and concurrency management
- Conditional logic and error handling
- Pre-built automation documents from AWS
- **Key Pattern:** Approval gates integrated directly into step definitions

**PagerDuty Runbook Automation** (formerly Rundeck)
- SaaS solution integrated with incident response
- Self-service operations for non-engineers
- Granular RBAC and approval workflows
- Real-time audit logs, cross-team visibility
- **Key Pattern:** Tight coupling between incident context and runbook execution

**Rundeck** (Open Source)
- Job scheduling and orchestration
- Rich plugin ecosystem for integrations
- Fine-grained RBAC and audit logging
- Multi-step automated jobs with conditional logic
- Works across clouds, on-prem, hybrid
- **Key Pattern:** Self-service operational automation with access controls

**StackStorm**
- Event-driven automation ("IFTTT for Ops")
- Rule engine: triggers, rules, actions
- Powerful workflows (Mistral, Orquesta) with branching and loops
- 100+ integration packs
- **Key Pattern:** Event-driven orchestration triggered by diverse IT events

**Comparison Insight:** Industry tools differ in their primary trigger model:
- SSM: On-demand or event-driven (CloudWatch, Config)
- PagerDuty: Incident-driven
- Rundeck: Schedule-driven + on-demand
- StackStorm: Event-driven automation

### 1.3 Academic Research

**Key Papers and Resources:**

1. **"Runbook Engineering and SOP Design in High Availability Environments"**
   - Explores principles of runbook engineering for SRE teams
   - Highlights automation-ready documentation for rapid incident response
   - Focus on auditability, compliance, and MTTR reduction
   - Source: IJSRET Vol.10 Issue 6

2. **"Optimal Automated Generation of Playbooks" (DBSec 2024, Springer)**
   - Multi-criteria optimization for playbook generation (Pareto front)
   - Ontology-based approach for SOAR platforms
   - Applicable to operational automation beyond security

3. **"Delivering Consistency and Automation with Operational Runbooks" (IBM Redbook)**
   - Deep dive into IBM Runbook Automation service
   - Blends analytics with automation for IT operations
   - Cloud and on-premises integration scenarios

4. **"Implementing RunOps Engineering" (European Journal of Advances in Engineering and Technology)**
   - Integration of automation, IaC, CI/CD, and observability
   - Real-world case studies
   - Future trends: AI/ML, serverless, edge computing in operational automation

5. **"Runbooks as Code: A Comprehensive Tutorial for Site Reliability Engineering"**
   - Version-controlled, executable code artifacts
   - Benefits: automation, consistency, auditability
   - Integration with Ansible, Terraform, Git workflows

**Key Academic Insight:** The transition to "runbooks as code" mirrors the evolution of infrastructure management (IaC), emphasizing version control, testing, and automation pipelines.

---

## 2. Workflow Orchestration Patterns

### 2.1 Workflow Engine Landscape

**Temporal**
- Durable execution with state persistence
- Native saga pattern support with compensation activities
- Deterministic replay for debugging and audit
- At-least-once processing guarantees → requires idempotent activities
- Signals and queries for workflow interaction
- **Key Pattern:** Compensation registered per activity, executed in reverse on failure

**Prefect**
- Python-native, API-first design
- Dynamic DAGs (runtime construction)
- Advanced observability and monitoring
- Hybrid/local/cloud deployment flexibility
- **Key Pattern:** Functional workflow definition with task.map() for parallelization

**Argo Workflows**
- Kubernetes-native, containerized workflow engine
- Each step runs as a Kubernetes pod
- YAML/JSON CRD-based definitions
- Step-level retry strategies
- **Key Pattern:** Fully containerized execution with K8s scheduling

**Apache Airflow**
- Batch/ETL focus, Python-based
- Cron-based scheduling with timetables
- Rich plugin ecosystem
- TaskGroups for fan-out/fan-in patterns
- **Key Pattern:** Python DAG definitions with operators and sensors

**AWS Step Functions**
- JSON/YAML state machine definitions
- Service integrations (Lambda, ECS, SNS, etc.)
- Error handling with retry and catch states
- Human approval tasks via callbacks
- **Key Pattern:** State machine model with explicit error handlers

### 2.2 Core Workflow Patterns

**Saga Pattern**
- Sequence of local transactions
- Each transaction paired with compensating transaction
- On failure, execute compensations in reverse order
- **Implementation:** Temporal saga helper, manual compensation chains
- **Gert v1 Context:** Not explicitly supported; manual compensation via branch logic

**Compensation**
- Rollback or mitigation of completed steps
- Must be idempotent (may be called multiple times)
- Examples: refund payment, release resource, restore backup
- **Best Practice:** Register compensation immediately after forward action

**Fan-out / Fan-in**
- Parallel execution of independent tasks, then join
- **Airflow:** TaskGroup or multiple downstream tasks
- **Argo:** Parallel steps in YAML
- **Prefect:** task.map() for parallelization
- **Gert v1 Context:** Limited parallelism; branches are conditional, not parallel

**Retry with Backoff**
- Automatic retry of failed steps with exponential backoff
- Configurable max attempts and delay
- Essential for transient failures (network, rate limits)
- **Industry Standard:** max_attempts: 3-5, backoff: exponential with jitter

**Idempotency**
- Steps must handle repeated execution without unintended side effects
- Strategies: unique operation IDs, deduplication tables, conditional updates
- **Critical for:** At-least-once execution guarantees

### 2.3 Step-level Semantics

**Atomicity**
- Step completes fully or has no effect (all-or-nothing)
- Non-atomic steps require compensating transactions
- Examples of atomic: database transaction; non-atomic: email sent

**Observability**
- Structured logging with step start/success/failure
- Metrics: duration, retry count, failure rate
- Distributed tracing: correlation IDs across services
- Events: step transition events (started, succeeded, failed)
- **Best Practice:** Structured logs + correlated IDs + custom hooks

**Error Handling**
- Retry policies (with backoff)
- Task timeouts to avoid stuck execution
- Fallback steps or compensating transactions
- Manual intervention (escalation, human approval)
- Dead-letter queues for failed steps
- **Pattern Summary:** Retry → Compensate → Escalate

### 2.4 YAML DSL vs. Code-defined Workflows

| Aspect | YAML DSL | Code-defined |
|--------|----------|--------------|
| **Expressiveness** | Limited (no loops, complex logic) | Full programming language |
| **Readability** | High for simple workflows | Lower; requires code knowledge |
| **Reusability** | Limited (anchors, includes) | High (functions, classes) |
| **Testing** | Vendor-specific simulators | Standard unit/integration tests |
| **Dynamic Logic** | Difficult/impossible | Easy (runtime DAG construction) |
| **Tooling** | IDE support via JSON Schema | Full IDE support (debugger, autocomplete) |
| **Best For** | CI/CD, standard pipelines | Data engineering, ML, complex logic |

**Industry Trend:** Hybrid approaches emerging (e.g., Argo with Jsonnet, GitHub Actions with expressions and reusable workflows).

**Recommendation for Gert v2:** Retain YAML as canonical format (compatibility, auditability), but consider expression language for complex conditionals and transformations.

---

## 3. Governance Models

### 3.1 Approval Gates and Change Management

**ITIL Change Advisory Board (CAB) Model**

Approval gates in change management process:

1. **Change Proposal Gate**
   - Early review of major/high-risk changes
   - Approvers: Change Manager, CAB
   - Outcome: Go/no-go for detailed planning

2. **Change Assessment Gate**
   - Review of fully documented Request for Change (RFC)
   - Approvers: CAB, ECAB (emergency), or delegated authority
   - Outcome: Approve, reject, or request more info

3. **Pre-Implementation Gate**
   - Final readiness review before deployment
   - Verifies: back-out plan, communication, testing
   - Approvers: CAB/ECAB

4. **Post-Implementation Review (PIR)**
   - Review outcome and lessons learned
   - Close change record, capture metrics

**Change Classification Patterns:**
- **Standard Change:** Pre-authorized, low-risk, minimal approval
- **Normal Change:** Full CAB review at multiple gates
- **Emergency Change:** Expedited ECAB approval, post-implementation review mandatory

**Key Insight:** Approval gates should be risk-based; not all changes require full CAB review.

**Gert v1 Context:** `approvals.min` and `approvals.roles` implement basic approval gates. v2 opportunity: support emergency paths, escalation on timeout, PIR capture.

### 3.2 RBAC in Operational Tools

**Permission Delegation Patterns:**

1. **Direct Role Assignment**
   - Simple: User → Role → Permissions
   - Example: Admin grants "DBA" role

2. **Role Grouping (Composite Roles)**
   - Roles composed of other roles (inheritance)
   - Example: "DevOps" = "CI/CD User" + "Production Viewer"

3. **Delegated Administration**
   - Users with certain roles can assign roles to others
   - Scoped delegation (e.g., project leads manage team permissions)
   - Example: GCP IAM "Role Administrator"

4. **Attribute-Based Delegation**
   - Role assignment based on user attributes (department, project)
   - Dynamic groups (Azure AD Dynamic Groups)

**Policy Enforcement Patterns:**

1. **Centralized Policy Enforcement**
   - Central authority enforces all policies (IdP, policy engine)
   - Advantage: Consistency, ease of audit
   - Example: OPA, Azure Policy

2. **Decentralized / Distributed Enforcement**
   - Individual services enforce policies locally
   - Advantage: Scalability, resilience
   - Example: Kubernetes RBAC

3. **Just-in-Time (JIT) Access**
   - Roles granted for limited time, require approval
   - Advantage: Least privilege, reduced attack surface
   - Example: Azure PIM, AWS IAM Access Analyzer

4. **Policy as Code**
   - Policies managed, versioned, tested as code in CI/CD
   - Advantage: Traceability, change management
   - Example: OPA with GitOps

**Best Practices:**
- Least privilege principle
- Separation of duties (creator ≠ approver)
- Regular access reviews and audits
- Automated role revocation on role changes
- Delegation constraints (which roles can delegate what)

**Gert v1 Context:** `governance.allowed_commands`, `governance.deny_env_vars` are simple policies. v2 opportunity: RBAC for runbook execution, policy-as-code integration (OPA).

### 3.3 Policy-as-Code: OPA vs. Cedar

**Open Policy Agent (OPA)**
- Language: Rego (declarative, logic-based)
- Scope: General-purpose policy (authz, compliance, infrastructure)
- Ecosystem: CNCF, cloud-agnostic, broad integrations
- Tooling: `opa test`, Conftest, Gatekeeper (K8s)
- Governance: Mature policy-as-code workflows (Git, CI/CD, bundles)
- **Best for:** Multi-cloud, cross-platform governance

**Cedar (AWS)**
- Language: Cedar (resource/principal-centric, IAM-like)
- Scope: Authorization only
- Ecosystem: AWS-centric (Verified Permissions), emerging OSS
- Tooling: Less mature outside AWS
- Governance: First-class in AWS context
- **Best for:** AWS-native apps, IAM-style policies

**Comparison Summary:**

| Aspect | OPA/Rego | Cedar |
|--------|----------|-------|
| Use Case | General policy | AuthZ only |
| Community | Large, CNCF | AWS-focused |
| Language | Logic programming | Resource/principal |
| Testing | CLI, lib, CI | Maturing |
| Audit | Decision logs | AWS-native logs |
| Governance Tooling | Rich (GitOps, versioning) | Early-stage outside AWS |

**Recommendation for Gert v2:** If policy-as-code is required, OPA is better aligned with cloud-agnostic goals. Consider policy hooks for command allowlists, approval requirements, redaction rules.

### 3.4 Command Allowlists/Denylists

**Governance Primitive Pattern:**
- **Allowlist:** Explicitly permitted commands/tools (default deny)
- **Denylist:** Explicitly forbidden commands (default allow)
- **Best Practice:** Allowlist for high-security environments, denylist for flexibility with guardrails

**Industry Examples:**
- AWS SSM: Command document whitelisting
- Rundeck: ACL policies on job execution
- HashiCorp Sentinel: Policy checks before Terraform apply

**Gert v1 Context:** `governance.allowed_commands` is allowlist-based. Strong pattern for v2 to retain and potentially enhance with regex patterns, context-aware policies.

### 3.5 Compliance Frameworks: Audit Trail Requirements

**SOC 2**
- **Requirement:** Audit logs of system activity, user access, admin actions
- **Retention:** Typically 90 days to 1 year
- **Protection:** Logs must be tamper-proof and regularly reviewed

**ISO 27001 (A.12.4)**
- **Requirement:** Event logging and monitoring (user activities, exceptions, security events)
- **Content:** User IDs, activities, timestamps, source/destination, outcomes
- **Protection:** Regular review, protected from unauthorized changes

**HIPAA (45 CFR 164.312(b))**
- **Requirement:** Record and evaluate access to ePHI
- **Content:** Who accessed, when, what actions
- **Retention:** Minimum 6 years

**Operational Tools for Compliance:**
- SIEM: Splunk, LogRhythm, Sumo Logic, ELK Stack
- Cloud-native: AWS CloudTrail, Azure Monitor/Sentinel, GCP Audit Logs
- File integrity: Tripwire, OSSEC

**Gert v1 Context:** Append-only JSONL traces address many audit requirements. v2 opportunity: structured compliance reports, retention policies, log signing/verification.

---

## 4. Traceability & Evidence

### 4.1 Append-only Audit Log Patterns

**JSONL Pattern:**
- Each line = one JSON object (event)
- Never modify or delete lines (append-only)
- Events are self-describing with timestamp, actor, operation, entity

**Example Event:**
```jsonl
{"event_id": "abcd1234", "timestamp": "2024-06-18T18:40:23Z", "user": "alice", "operation": "STEP_START", "step_id": "resolve_dns", "runbook_id": "rb-123"}
{"event_id": "efgh5678", "timestamp": "2024-06-18T18:40:45Z", "user": "alice", "operation": "STEP_COMPLETE", "step_id": "resolve_dns", "exit_code": 0, "captured": {"dns_output": "..."}}
```

**Event Sourcing:**
- Application state reconstructed by replaying events
- Current state = f(all events)
- Benefits: Complete audit trail, debugging, deterministic replay

**Immutable Ledger:**
- Once written, records cannot be altered
- Use cryptographic hashing for tamper evidence
- Examples: Blockchain-based audit logs, AWS QLDB

**Best Practices:**
- Use unique event IDs (UUID)
- ISO 8601 UTC timestamps
- Include correlation IDs for cross-system tracing
- Consider cryptographic signatures for high-assurance environments

**Gert v1 Context:** JSONL trace files are append-only. v2 opportunity: event versioning, cryptographic signatures, long-term archival strategies.

### 4.2 Evidence Capture Patterns

**What Constitutes Evidence?**
- **Outputs:** Command stdout/stderr, tool responses
- **Attestations:** Manual confirmations, checklists, sign-offs
- **Artifacts:** Screenshots, log files, configuration snapshots
- **Hashes:** SHA256 of outputs for integrity verification
- **Metadata:** Who, when, where, why

**Operational Workflow for Evidence:**
1. **Initiation:** Triggered by event or process requirement
2. **Documentation:** Context (who, what, when, where, how)
3. **Acquisition:** Securely collect evidence, maintain integrity
4. **Transfer & Storage:** Secure location, controlled access
5. **Chain of Custody:** Log every handoff or access
6. **Validation:** Hash verification, integrity checks
7. **Reporting:** Audit-ready records

**Attestation Patterns:**
- **Manual:** Human signs statement (followed policy)
- **Automated:** System-generated logs/trails
- **Cryptographic:** Digital signatures for non-repudiation

**Gert v1 Context:** Evidence capture with text, checklists, attachments, SHA256 hashing. Strong foundation. v2 opportunity: timestamped attestations, multi-party signatures, evidence retention policies.

### 4.3 Deterministic Replay

**Definition:** Ability to replay execution exactly as it happened originally.

**Requirements:**
- Record all inputs and non-deterministic events (timestamps, random values, external API responses)
- Replay with recorded events, not live data
- Same result guaranteed

**Use Cases:**
- **Debugging:** Reproduce failures consistently
- **Auditing:** Verify what happened at each step
- **Testing:** Validate fixes against historical scenarios

**Industry Examples:**
- **Temporal:** Deterministic workflow replay built-in
- **Cadence:** Event history replay for debugging
- **Gert v1:** Scenario replay with recorded command responses

**Implementation Patterns:**
- Avoid non-deterministic operations in core logic (use `random`, `now()` as external inputs)
- Record all side-effecting operations (DB, HTTP, command execution)
- Store snapshots at each step for state reconstruction

**Gert v1 Context:** Scenario replay with `scenario.yaml` (pre-recorded responses). Strong pattern. v2 opportunity: live execution recording mode, replay from production traces.

### 4.4 Provenance Tracking

**Definition:** Record of origin, history, and custody of data/evidence.

**Why Critical:**
- Legal admissibility
- Compliance audits
- Integrity verification

**Patterns:**
- **Chain of Custody Logs:** Time-stamped record of each transfer/access
- **Metadata Tagging:** System/user actions, contextual data
- **Immutable Ledgers:** Tamper-evident logs
- **Audit Trails:** All evidence-related activities tracked

**W3C PROV Model:**
- **Entities:** Data/artifacts
- **Activities:** Processes that use/generate entities
- **Agents:** People/systems performing activities
- **Relationships:** wasGeneratedBy, wasAttributedTo, wasDerivedFrom

**Gert v1 Context:** JSONL traces provide provenance at step level. v2 opportunity: explicit provenance graph, W3C PROV compliance for high-assurance use cases.

### 4.5 Distributed Tracing: OpenTelemetry

**Trace Context (W3C Standard):**
- **Trace ID:** Uniquely identifies entire distributed trace
- **Span ID:** Identifies specific operation within trace
- **Parent Span ID:** Links child operations
- **Trace Flags:** Sampling decisions

**Propagation:**
- HTTP headers: `traceparent`, `tracestate`, `baggage`
- Example: `traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01`

**Correlation IDs:**
- Unique identifier for request/transaction
- Trace ID often serves as correlation ID
- Attached to logs, metrics, traces for unified analysis

**Best Practices:**
- Propagate trace context at every request boundary
- Ensure logs and traces share IDs for correlation
- Use auto-instrumentation for consistent context
- Use tracing backends (Jaeger, Zipkin) for visualization

**Gert v1 Context:** No explicit distributed tracing. v2 opportunity: OpenTelemetry spans for steps, correlation IDs in traces, integration with observability platforms.

---

## 5. Human-in-the-Loop Patterns

### 5.1 Manual Approval Steps

**Process Pattern:**
1. Workflow reaches approval step
2. Workflow pauses, notifies approver (email, dashboard, chat)
3. Approver reviews context and decides (approve/reject)
4. Workflow continues or terminates based on decision
5. Decision logged to audit trail

**Why Use Approvals:**
- Compliance (financial, security, legal)
- Business rules (threshold-based, e.g., expenses > $1000)
- Quality assurance for critical operations
- Change management gates (ITIL)

**Multi-level Approvals:**
- Parallel: Multiple approvers, all must approve
- Serial: Sequential approvers (escalation chain)
- Threshold: N of M approvers

**Gert v1 Context:** `approvals.min` and `approvals.roles` support basic approval. v2 opportunity: multi-level approval chains, parallel approvals, conditional approval based on risk.

### 5.2 Escalation Paths

**Triggers for Escalation:**
- **Timeout:** Approver doesn't respond within SLA
- **Rejection:** Explicit denial, escalate to higher authority
- **Multiple Failures:** Repeated failed attempts
- **Exception Conditions:** Error or unhandled scenario

**Escalation Process:**
1. Initial approver assigned (User A)
2. SLA timer starts (e.g., 24 hours)
3. Reminders sent at intervals (50%, 75%, 90%)
4. On timeout, escalate to User B (manager, director)
5. Escalation logged to audit trail

**Best Practices:**
- Define clear escalation chains (L1 → L2 → L3)
- Automatic reminders reduce bottlenecks
- Audit all escalations for post-mortem
- Consider auto-approval/auto-reject for breached SLAs (with appropriate controls)

**Gert v1 Context:** No explicit escalation support. v2 opportunity: timeout-based escalation, escalation chains, configurable reminder schedules.

### 5.3 Evidence Collection at Human Steps

**Attestation Patterns:**
- **Checklists:** Operator confirms completion of items
- **Free-text:** Operator provides notes/observations
- **Attachments:** Screenshots, logs, diagnostic outputs
- **Sign-offs:** Digital signature or confirmation timestamp

**SLA Enforcement:**
- Human steps often have time constraints
- Timeout enforcement with escalation or auto-fail
- Monitoring and alerting on pending human tasks

**Gert v1 Context:** `required_evidence` with checklists, text, attachments. Strong pattern. v2 opportunity: timestamped attestations, multi-party sign-offs, evidence templates.

---

## 6. Schema & DSL Design Patterns

### 6.1 YAML-based Workflow DSLs

**What Works:**
- **Declarative structure:** Clear "what" not "how"
- **Human-readable:** Easy to review and audit
- **Tool standardization:** IDE support, schema validation
- **Parameterization:** Reusable with inputs
- **Version pinning:** Explicit action/tool versions

**What Doesn't Work:**
- **Complex logic:** Loops, dynamic branches awkward or impossible
- **Reusability:** Limited (anchors, includes are clunky)
- **Testing:** Difficult to unit test YAML
- **Scaling complexity:** Large workflows become unwieldy

**Industry Examples:**
- **GitHub Actions:** `uses: actions/checkout@v3`, `with` parameters, expressions for conditionals
- **Argo Workflows:** Template reuse, container steps, DAG/steps modes
- **GitLab CI:** YAML with `!reference`, `extends`, `include`
- **CircleCI:** Orbs for reusable workflow components

**Common Pitfalls:**
- Over-reliance on string templating (hard to debug)
- No schema validation (errors caught at runtime)
- Versioning gaps (breaking changes in reusable workflows)

**Gert v1 Context:** YAML DSL with Go templates. v2 opportunity: stronger expression language, improved error messages, schema-driven validation.

### 6.2 JSON Schema for Validation

**Best Practices:**
- **Versioned schemas:** Draft 2020-12, explicitly versioned
- **Self-describing:** Documents reference schema URL
- **Extension points:** `additionalProperties`, `oneOf`, `$defs`
- **Validation at authoring time:** IDE integration (VSCode, IntelliJ)

**Example Schema Reference:**
```yaml
$schema: "https://gert.dev/schemas/runbook/v1.json"
apiVersion: runbook/v1
...
```

**Tools:**
- JSON Schema validators (ajv, jsonschema)
- VSCode extension for schema-based autocomplete
- `gert schema export` for embedded schema distribution

**Gert v1 Context:** JSON Schema Draft 2020-12 for runbook validation. Strong foundation. v2 opportunity: tool/provider schema composition, schema marketplace, validation error localization.

### 6.3 Versioning Strategies

**Schema Versioning:**
- **Semantic versioning:** MAJOR.MINOR.PATCH
- **API version field:** `apiVersion: runbook/v1`
- **Compatibility guarantees:**
  - MAJOR: breaking changes
  - MINOR: backward-compatible additions
  - PATCH: backward-compatible fixes

**Workflow Versioning:**
- **Version pinning:** `uses: actions/checkout@v3`
- **Lock files:** Pin dependencies (e.g., `package-lock.json` equivalent)
- **Migration paths:** Tools to upgrade v0 → v1 (`gert migrate`)

**Best Practices:**
- Always pin versions of external dependencies
- Provide migration tools for breaking changes
- Test old versions with new runtime (compatibility matrix)
- Deprecation warnings before removal

**Gert v1 Context:** `apiVersion: runbook/v0` and `v1`, `gert migrate` for upgrades. v2 opportunity: clear compatibility matrix, automated migration testing, deprecation timelines.

### 6.4 Self-describing Schemas

**Pattern:**
- Schema URL embedded in document
- Tooling discovers schema automatically
- Validation and autocomplete without external config

**Example:**
```yaml
$schema: "https://gert.dev/schemas/runbook/v2.json"
apiVersion: runbook/v2
```

**Benefits:**
- No external schema configuration required
- IDE support works out-of-box
- Clear schema version contract

**Gert v1 Context:** Not currently self-describing (relies on `apiVersion` field). v2 opportunity: add `$schema` field for JSON Schema-compliant tooling.

---

## 7. Key Learnings & Recommendations for Gert v2

### 7.1 What Gert v1 Gets Right

1. **Governance-first design:** Command allowlists, env var blocking, redaction, approval gates
2. **Evidence capture:** Text, checklists, attachments with SHA256 hashing
3. **Append-only traceability:** JSONL traces, per-step state snapshots
4. **Tool abstraction:** Pluggable tools with stdio/jsonrpc/mcp transports
5. **Deterministic replay:** Scenario-based testing and replay
6. **JSON Schema validation:** Strict schema enforcement (Draft 2020-12)
7. **Multi-mode execution:** Real, dry-run, replay modes

### 7.2 Industry Learnings Not Yet in v1

1. **Workflow patterns:**
   - **Saga/compensation:** No explicit compensation pattern
   - **Parallel execution:** Branches are conditional, not parallel fan-out
   - **Retry with backoff:** Limited step-level retry configuration

2. **Governance:**
   - **Policy-as-code:** No integration with OPA/Cedar for dynamic policy
   - **RBAC:** No role-based execution permissions
   - **Escalation:** No timeout-based escalation chains
   - **Change classification:** No standard/normal/emergency change paths

3. **Traceability:**
   - **Distributed tracing:** No OpenTelemetry integration
   - **Provenance graphs:** No explicit entity-activity-agent relationships
   - **Log signing:** No cryptographic evidence integrity

4. **Human-in-loop:**
   - **Multi-level approvals:** No parallel or serial approval chains
   - **SLA enforcement:** No automated timeout/escalation
   - **Attestation timestamps:** No cryptographic timestamp authority

5. **Schema/DSL:**
   - **Expression language:** Go templates powerful but can be verbose
   - **Self-describing schemas:** Missing `$schema` field
   - **Schema composition:** Limited composability for project-specific extensions

### 7.3 Top Recommendations for v2

#### 1. **Add Explicit Saga/Compensation Pattern**
- Allow steps to register compensation handlers
- On workflow failure or cancellation, execute compensations in reverse order
- **Rationale:** Industry standard for long-running transactions, enables safe rollback

#### 2. **Introduce Parallel Execution (Fan-out/Fan-in)**
- Support parallel step groups with join semantics
- **Syntax suggestion:**
  ```yaml
  parallel:
    - step: {id: check_dns, ...}
    - step: {id: check_http, ...}
  join:
    wait_for: all  # or any, majority
  ```
- **Rationale:** Common pattern in workflow engines, improves execution speed

#### 3. **Policy-as-Code Integration**
- Provide hooks for OPA evaluation at runtime
- Policy checkpoints: pre-execution, pre-step, post-step
- **Example policies:**
  - `runbook.yaml` requires approval if commands include `kubectl delete`
  - Steps modifying production require 2 approvals
- **Rationale:** Enables centralized, auditable governance at scale

#### 4. **RBAC for Runbook Execution**
- Role-based permissions: who can execute which runbooks
- Delegation: project leads can grant execution rights within scope
- **Example:**
  - Role: `incident-responder` → can execute `kind: mitigation` runbooks
  - Role: `change-manager` → can approve `kind: change` runbooks
- **Rationale:** Aligns with enterprise operational tooling, reduces over-privileged execution

#### 5. **Timeout and Escalation for Human Steps**
- Configure SLA timers on manual and approval steps
- On timeout: escalate to alternate approver, send notifications, or fail
- **Syntax suggestion:**
  ```yaml
  approvals:
    min: 1
    roles: ["DRI"]
    timeout: 24h
    on_timeout: escalate
    escalate_to: ["manager", "director"]
  ```
- **Rationale:** Prevents runbook stalls, aligns with ITIL change management

#### 6. **OpenTelemetry Integration for Distributed Tracing**
- Emit OpenTelemetry spans for each step
- Propagate trace context across invoked runbooks and tools
- Correlation ID in all log entries and events
- **Benefits:** Unified observability with existing monitoring infrastructure
- **Rationale:** Industry standard for distributed systems, enables cross-system correlation

#### 7. **Enhanced Schema Versioning and Migration**
- Add `$schema` field for self-describing documents
- Semantic versioning with clear compatibility guarantees
- Automated compatibility testing (old runbooks on new runtime)
- **Rationale:** Reduces migration friction, aligns with JSON Schema best practices

### 7.4 Open Questions Remaining in Industry

1. **Policy Execution Location:**
   - Should complex policies execute in-process or out-of-process?
   - Trade-off: performance vs. isolation

2. **Evidence Retention:**
   - How long to retain evidence artifacts (attachments, logs)?
   - Who owns long-term archival (runbook engine vs. external storage)?

3. **Multi-tenant Isolation:**
   - How to safely execute untrusted runbooks in shared infrastructure?
   - Sandboxing, resource limits, network isolation?

4. **AI-generated Runbooks:**
   - How to validate and govern AI-generated operational procedures?
   - Human-in-loop requirements, testing standards?

5. **Real-time Collaboration:**
   - Should multiple operators be able to execute the same runbook concurrently?
   - Concurrency control, state synchronization?

---

## 8. References

### Academic Papers
1. "Runbook Engineering and SOP Design in High Availability Environments," IJSRET Vol.10 Issue 6
2. "Optimal Automated Generation of Playbooks," DBSec 2024, Springer
3. "Delivering Consistency and Automation with Operational Runbooks," IBM Redbook
4. "Implementing RunOps Engineering," European Journal of Advances in Engineering and Technology
5. "Runbooks as Code: A Comprehensive Tutorial for Site Reliability Engineering," SRE School

### Industry Systems
- AWS Systems Manager Automation Documentation
- PagerDuty Runbook Automation Product Documentation
- Rundeck Documentation (Open Source)
- StackStorm Documentation
- Temporal Workflow Engine Documentation
- Prefect Documentation
- Argo Workflows Documentation
- Apache Airflow Documentation

### Standards and Specifications
- ITIL 4 (Change Management, Service Operation)
- W3C Trace Context Specification
- W3C PROV Model (Provenance)
- OpenTelemetry Specification
- JSON Schema Draft 2020-12
- ISO 27001 (Information Security)
- SOC 2 (Security and Availability)
- HIPAA (Health Insurance Portability and Accountability Act)

### Policy Frameworks
- Open Policy Agent (OPA) Documentation
- Cedar Policy Language (AWS)
- HashiCorp Sentinel Documentation

### Books and Guides
- Google SRE Book: "Elimination of Toil"
- Google SRE Workbook: "Practical SRE"

---

## 9. Conclusion

Gert v1 has established a strong foundation in governance, traceability, and evidence capture. The v2 design presents an opportunity to incorporate proven patterns from workflow orchestration systems (Temporal, Argo, Prefect), policy frameworks (OPA, Cedar), and compliance standards (ITIL, SOC2, ISO27001).

**Priority Areas for v2:**
1. **Workflow sophistication:** Saga/compensation, parallel execution, enhanced retry/backoff
2. **Policy-as-code:** OPA integration, RBAC, dynamic governance
3. **Human-in-loop:** Timeout/escalation, multi-level approvals, attestation timestamps
4. **Observability:** OpenTelemetry integration, distributed tracing, correlation IDs
5. **Schema evolution:** Self-describing schemas, compatibility guarantees, migration tooling

This research brief provides the evidence-based foundation for architectural decisions. The patterns and recommendations here should guide schema design (John), architecture (Ken), integrations (Barbara), and documentation (Leslie).

**Next Steps:**
1. Ken to review recommendations and draft architectural principles
2. John to evaluate schema impacts of proposed patterns
3. Barbara to assess integration requirements (OPA, OpenTelemetry)
4. Team to prioritize recommendations for v2.0 vs. future releases

---

**Document Status:** Complete  
**Review Cycle:** Pending team review  
**Updates:** Append learnings to Dennis history, decision brief to inbox
