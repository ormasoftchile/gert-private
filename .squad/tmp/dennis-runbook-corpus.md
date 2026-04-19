# Real-World Runbook Corpus for gert v2 Schema Validation

**Author:** Dennis (CS Researcher)  
**Date:** 2026-04-18  
**Purpose:** Stress-test gert v2 schema across diverse real-world runbook patterns

---

## Selection Criteria

This corpus covers the widest possible space of real human operational needs:

### Coverage Dimensions

**DOMAINS:**
- SRE/Operations (incident response, scaling, health checks)
- DevOps/Deployment (CI/CD, canary releases, rollbacks)
- Compliance/Audit (SOC2 evidence, GDPR, regulatory)
- Security (incident containment, forensics, breach response)
- IT/Onboarding (employee setup, access provisioning)
- Data Engineering (migrations, ETL, pipeline orchestration)
- Finance (approval chains, purchase orders, budget gates)
- Regulated Industries (medical devices, FDA/HIPAA, quality gates)

**COMPLEXITY PATTERNS:**
- Linear: A → B → C (simple sequential)
- Branching: if X then Y else Z (conditional routing)
- Parallel: fan-out/fan-in (concurrent execution, wait-all)
- Looping/Retry: repeat until condition or max attempts
- Nested: runbook invokes runbook (composition)
- Multi-party: multiple approvers, escalation chains

**INTERACTION PATTERNS:**
- Fully automated (no human input)
- Human approval gates (single approver)
- Multi-level approval (escalation chains)
- User data collection (forms, file uploads)
- Artifact collection (logs, screenshots, evidence files)
- Multi-party consensus (quorum approval, N-of-M)

**FAILURE MODES:**
- Compensating actions (rollback, undo)
- Partial completion (some steps succeed, some fail)
- Timeout/SLA enforcement (escalate if not approved in X time)
- Skip-on-condition (optional steps, feature flags)
- Retry with backoff (transient failure handling)

**EDGE CASES:**
- Very long runbooks (20+ steps)
- Single-step runbooks (wrapper for governance)
- No human interaction (fully automated)
- All human steps (orchestrated manual process)
- Cross-system workflows (API calls, webhooks, external tools)

---

## Runbook 1: Kubernetes Pod Incident Response

**Domain:** SRE/Operations  
**Complexity:** Branching + Looping  
**Interaction:** Human approval gates + automated remediation

### Summary
Automated incident response for unhealthy Kubernetes pods detected by monitoring. Includes automated diagnostics, decision-based remediation (restart vs. scale vs. escalate), and evidence collection for post-incident review.

### Steps

1. **Detect Alert** (automated, type: extension)
   - Triggered by Prometheus AlertManager webhook
   - Collects: pod name, namespace, cluster, alert severity, timestamp
   - Fails if: webhook payload malformed or missing required fields

2. **Gather Pod Diagnostics** (automated, type: cli)
   - Executes: `kubectl get pod <pod> -n <namespace> -o json`
   - Executes: `kubectl describe pod <pod> -n <namespace>`
   - Executes: `kubectl logs <pod> -n <namespace> --tail=100`
   - Stores: pod manifest, events, recent logs as evidence artifacts
   - Fails if: kubectl not configured, cluster unreachable, pod not found

3. **Analyze Failure Mode** (automated, type: decision)
   - Evaluates pod status JSON:
     - If CrashLoopBackOff or ImagePullBackOff → route to "Fix Config"
     - If OOMKilled or resource limits hit → route to "Scale Resources"
     - If NodeNotReady → route to "Check Node Health"
     - Else → route to "Escalate to On-Call"
   - Decision logic uses expressions on pod.status.conditions
   - Fails if: pod status JSON invalid or missing required fields

4. **Branch: Fix Config** (if CrashLoopBackOff)
   - Step 4a: Collect recent config changes from Git (type: cli)
     - Executes: `git log --oneline --since="1 hour ago" -- k8s/<namespace>/<pod>.yaml`
   - Step 4b: Human approval: "Restart pod with previous config?" (type: approval)
     - Shows: diff of current vs. previous config
     - Timeout: 5 minutes → escalate to senior SRE
   - Step 4c: Rollback config (type: cli)
     - Executes: `kubectl apply -f <previous-config>.yaml`
   - Step 4d: Wait for pod healthy (type: cli, retry loop)
     - Executes: `kubectl wait --for=condition=Ready pod/<pod> -n <namespace> --timeout=120s`
     - Retries: 3 times with 10s backoff
     - Fails if: pod never becomes ready

5. **Branch: Scale Resources** (if OOMKilled)
   - Step 5a: Human approval: "Double memory limit for pod?" (type: approval)
     - Shows: current resource usage, limits, and requested increase
   - Step 5b: Patch deployment (type: cli)
     - Executes: `kubectl patch deployment <deploy> -n <namespace> --patch='...'`
   - Step 5c: Wait for new pod ready (type: cli, retry loop)

6. **Branch: Check Node Health** (if NodeNotReady)
   - Step 6a: SSH to node (type: cli)
     - Executes: `ssh <node> 'uptime && df -h && free -m'`
   - Step 6b: Collect node logs (type: cli)
     - Executes: `ssh <node> 'journalctl -u kubelet --since "10 minutes ago"'`
   - Step 6c: Escalate to infrastructure team (type: approval)
     - Approvers: infra-oncall@company.com
     - Timeout: 10 minutes

7. **Branch: Escalate to On-Call** (if unknown failure)
   - Step 7a: Create PagerDuty incident (type: extension)
     - Calls PagerDuty API: create incident with high severity
     - Attaches: all collected diagnostics
   - Step 7b: Wait for acknowledgment (type: approval)
     - Approver: on-call engineer (from PagerDuty)
     - Timeout: 15 minutes → escalate to manager

8. **Post-Incident Evidence** (automated, type: cli)
   - Bundle all logs, configs, decisions into tar.gz
   - Upload to S3: `s3://incidents/<timestamp>-<pod>.tar.gz`
   - Record incident summary in incident database

9. **Close Alert** (automated, type: extension)
   - Calls AlertManager API: resolve alert
   - Sends Slack notification: "Incident resolved for pod <pod>"

### Complexity Tags
- Branching (4-way decision based on failure mode)
- Looping (retry on pod health check)
- Multi-party (different approvers for different branches)
- Timeout/SLA (escalation on approval timeout)

### Key Schema Challenges

1. **Dynamic branching based on JSON evaluation** — Step 3 needs to parse kubectl JSON output and route to one of four branches. Schema must support expression language powerful enough to navigate nested JSON (pod.status.conditions[?(@.type=="Ready")].status).

2. **Retry loops with backoff** — Steps 4d and 5c retry kubectl wait commands. Schema needs retry policy (max attempts, backoff strategy) and failure handling (fail runbook vs. continue).

3. **Approval timeout with escalation** — Steps 4b, 6c, 7b have timeouts. Schema must define what happens on timeout: escalate to different approver, fail step, or skip. Escalation may change approver list dynamically.

4. **Evidence artifact collection** — Steps 2, 8 collect files (logs, configs, diagnostics). Schema must support artifact capture with metadata (filename, size, hash) and storage destination (local, S3, trace attachment).

5. **Cross-branch resumption** — If runbook is interrupted (gert crashes, operator cancels), resume must work regardless of which branch was taken. Trace must record decision history.

6. **Compensation/rollback** — If Step 4c (config rollback) fails, original config is lost. Schema needs compensating action (restore original config) registered at step 4b.

---

## Runbook 2: Production Deployment with Canary + Rollback

**Domain:** DevOps/Deployment  
**Complexity:** Linear → Parallel → Branching (rollback on failure)  
**Interaction:** Human approval + automated monitoring

### Summary
Deploys new application version to production using canary strategy: deploy to 10% traffic, monitor metrics for 10 minutes, then promote to 100% or rollback. Includes automated health checks, metric queries, and rollback compensation.

### Steps

1. **Pre-Deployment Check** (automated, type: assert)
   - Asserts:
     - Git tag exists: `git rev-parse <tag>`
     - Docker image exists: `docker manifest inspect <image>:<tag>`
     - Staging tests passed: query CI system for test results
   - Fails if: any assertion fails (abort deployment)

2. **Human Approval: Deploy to Production** (type: approval)
   - Shows: Git changelog between current and new version
   - Shows: Docker image digest
   - Approvers: release-managers@company.com
   - Timeout: 30 minutes (deployment window closes)

3. **Tag Deployment Start** (automated, type: extension)
   - Calls observability API: create deployment marker with timestamp, version, actor
   - Stores: deployment_id for later correlation

4. **Register Rollback Compensation** (automated, internal)
   - Registers compensating action: "Rollback to version <current>" to be invoked if any subsequent step fails
   - Compensation steps: scale canary to 0%, wait 30s, delete canary deployment

5. **Deploy Canary (10% traffic)** (automated, parallel fan-out)
   - Step 5a: Create canary deployment (type: cli)
     - Executes: `kubectl apply -f canary-deployment.yaml`
   - Step 5b: Update service weights (type: cli)
     - Executes: `kubectl patch service <svc> --patch='{"spec":{"weights":[{"name":"prod","weight":90},{"name":"canary","weight":10}]}}'`
   - Parallel execution: both must succeed before proceeding
   - Fails if: deployment times out, service patch fails

6. **Wait for Canary Healthy** (automated, type: cli, retry loop)
   - Executes: `kubectl wait --for=condition=Available deployment/canary --timeout=120s`
   - Retries: 5 times with 15s backoff
   - Fails if: canary never becomes available → triggers rollback

7. **Monitor Canary Metrics (10 minutes)** (automated, type: iterate)
   - Loop for 10 minutes (60 iterations, 10s interval):
     - Query Prometheus: `rate(http_requests_total{deployment="canary",status=~"5.."}[1m])`
     - Query Prometheus: `histogram_quantile(0.95, http_request_duration_seconds{deployment="canary"})`
     - Collect: error rate, p95 latency
   - Convergence condition: error rate < 1%, p95 latency < 200ms for 5 consecutive checks
   - Early exit: if error rate > 5% or p95 > 500ms → fail step → triggers rollback
   - Fails if: metrics unreachable, query syntax error

8. **Decision: Promote or Rollback** (automated, type: decision)
   - Evaluates collected metrics:
     - If all metrics within SLO → route to "Promote"
     - If any metric outside SLO → route to "Rollback"
   - Decision logged to trace with full metric snapshot

9. **Branch: Promote to 100%** (if metrics healthy)
   - Step 9a: Human approval: "Promote canary to 100%?" (type: approval)
     - Shows: canary metrics summary (error rate, latency, throughput)
     - Approvers: release-managers@company.com
     - Timeout: 10 minutes → auto-rollback
   - Step 9b: Scale canary to 100% (type: cli)
     - Executes: `kubectl patch service <svc> --patch='{"spec":{"weights":[{"name":"canary","weight":100}]}}'`
   - Step 9c: Delete old prod deployment (type: cli)
     - Executes: `kubectl delete deployment prod`
   - Step 9d: Rename canary → prod (type: cli)
     - Executes: `kubectl patch deployment canary --patch='{"metadata":{"name":"prod"}}'`
   - Step 9e: Tag deployment success (type: extension)
     - Calls observability API: mark deployment as successful

10. **Branch: Rollback** (if metrics unhealthy OR step 9a timeout)
    - Step 10a: Scale canary to 0% (type: cli)
      - Executes: `kubectl patch service <svc> --patch='{"spec":{"weights":[{"name":"prod","weight":100},{"name":"canary","weight":0}]}}'`
    - Step 10b: Wait 30 seconds (type: cli)
      - Executes: `sleep 30`
    - Step 10c: Delete canary deployment (type: cli)
      - Executes: `kubectl delete deployment canary`
    - Step 10d: Tag deployment failure (type: extension)
      - Calls observability API: mark deployment as rolled back
    - Step 10e: Send alert (type: extension)
      - Sends Slack message: "Deployment rolled back due to unhealthy metrics"

11. **Post-Deployment Verification** (automated, convergent on both branches)
    - Verify prod deployment healthy (type: cli)
    - Verify service endpoints responding (type: cli)
    - Run smoke tests (type: cli): `curl https://api.company.com/health`

12. **Clear Rollback Compensation** (automated, internal)
    - Unregisters rollback compensation (deployment succeeded or already rolled back)

### Complexity Tags
- Linear → Parallel → Branching
- Looping (metrics monitoring with early exit)
- Compensation/Saga (rollback registered at step 4, executed on failure)
- Timeout/SLA (approval timeout triggers rollback)

### Key Schema Challenges

1. **Saga pattern / compensation registration** — Step 4 registers rollback actions to be invoked if *any* later step fails. Schema must support compensating action registration with scope (applies to steps 5-9) and trigger condition (any failure, specific failures, timeout).

2. **Parallel fan-out with wait-all** — Step 5 executes two kubectl commands in parallel and waits for both to succeed. Schema must support parallel execution with synchronization (wait-all vs. wait-any) and failure handling (fail if any fail, fail if all fail).

3. **Iterate with early exit and convergence** — Step 7 loops for 10 minutes but exits early if metrics degrade. Schema needs iterate block with: max iterations, interval, convergence condition (5 consecutive healthy checks), and failure condition (error rate > 5%).

4. **Conditional rollback trigger** — Rollback (step 10) is triggered by: (a) step 7 metrics failure, (b) step 8 decision, OR (c) step 9a approval timeout. Schema must support multiple trigger sources for a branch.

5. **Approval timeout with default action** — Step 9a timeout causes rollback (not escalation or manual intervention). Schema needs timeout action: escalate, skip, fail, or default-choice.

6. **Cross-branch convergence** — Step 11 runs after both promote and rollback branches. Schema must support join point (fan-in) where execution resumes on the main path regardless of which branch was taken.

7. **Evidence correlation** — Deployment_id from step 3 must propagate to all subsequent steps (OpenTelemetry trace context). Schema must support context propagation across steps and branches.

---

## Runbook 3: New Employee Onboarding

**Domain:** HR/IT  
**Complexity:** Parallel (independent provisioning tasks) + Multi-party approval  
**Interaction:** Human data collection + multi-party approval

### Summary
Onboards new employee by provisioning accounts, devices, and access. Includes HR data collection, manager approval, parallel IT provisioning tasks (Okta, GitHub, Slack, laptop), and compliance attestation.

### Steps

1. **Collect Employee Information** (human, type: collector)
   - Prompts HR for:
     - Full name (text input)
     - Email (email validation)
     - Start date (date picker)
     - Department (dropdown: Engineering, Sales, Marketing, Finance)
     - Manager (autocomplete from employee directory)
     - Job title (text input)
     - Office location (dropdown: SF, NYC, London, Remote)
   - Stores: employee_data object
   - Fails if: required fields missing, email already exists in system

2. **Manager Approval: Confirm Hire** (type: approval)
   - Shows: employee_data summary
   - Approver: manager specified in step 1
   - Timeout: 2 business days → escalate to HR director
   - Stores: manager approval timestamp and signature

3. **Background Check Verification** (human, type: manual)
   - Instructions: "Verify background check status in HireRight portal"
   - Checklist:
     - [ ] Background check completed
     - [ ] No disqualifying issues
     - [ ] Uploaded background check report to employee folder
   - Assignee: HR coordinator
   - Timeout: 5 business days → escalate to HR manager
   - Stores: attestation that background check passed

4. **Parallel Provisioning Tasks** (automated + human, parallel fan-out)
   - All tasks execute concurrently, wait-all before proceeding

   **Task 4a: Create Okta Account** (type: extension)
   - Calls Okta API: create user with employee_data
   - Assigns groups based on department
   - Sends welcome email with temporary password
   - Fails if: Okta API error, email already exists

   **Task 4b: Create GitHub Account** (type: extension)
   - Calls GitHub API: invite user to organization
   - Adds to teams based on department (engineering only)
   - Fails if: GitHub API error, user already exists

   **Task 4c: Create Slack Account** (type: extension)
   - Calls Slack API: invite user to workspace
   - Adds to channels: #general, #<department>, #announcements
   - Fails if: Slack API error, email already in workspace

   **Task 4d: Order Laptop** (human, type: manual)
   - Instructions: "Order laptop in IT procurement system"
   - Checklist:
     - [ ] Selected laptop model (MacBook Pro for eng, MacBook Air for others)
     - [ ] Entered shipping address: <office_location> or <home_address_if_remote>
     - [ ] Estimated delivery date: <date>
   - Assignee: IT procurement team
   - Timeout: 3 business days → escalate to IT manager

   **Task 4e: Schedule IT Onboarding Call** (human, type: manual)
   - Instructions: "Schedule 30-minute IT onboarding call on employee's first day"
   - Checklist:
     - [ ] Calendar invite sent to employee
     - [ ] Zoom link included
   - Assignee: IT onboarding coordinator

5. **Wait for All Provisioning** (automated, join point)
   - Waits for tasks 4a-4e to complete
   - Fails if: any task fails (abort onboarding)

6. **Security Training** (human, type: approval)
   - Instructions: "Employee must complete security training in LMS"
   - Checklist:
     - [ ] Completed security awareness training
     - [ ] Completed phishing simulation
     - [ ] Signed acceptable use policy
   - Assignee: employee (self-service)
   - Timeout: 7 business days → blocks access to sensitive systems
   - Stores: training completion date and certificate

7. **Access Provisioning** (automated, type: extension)
   - Reads department from employee_data
   - Provisions access based on role:
     - Engineering: GitHub repos, AWS dev account, Datadog
     - Sales: Salesforce, HubSpot
     - Marketing: Google Ads, Mailchimp
     - Finance: NetSuite, Bill.com
   - Calls respective APIs for each system
   - Fails if: any API call fails → retry 3 times with 1 minute backoff

8. **Manager Confirmation: Access Provisioned** (human, type: approval)
   - Shows: list of provisioned accounts and access levels
   - Approver: manager
   - Question: "Confirm access is appropriate for employee's role?"
   - Timeout: 1 business day → auto-approve (assume correct)

9. **Welcome Email** (automated, type: extension)
   - Sends personalized welcome email to employee
   - Includes: credentials, first-day instructions, org chart, manager contact
   - Stores: email sent timestamp

10. **Compliance Attestation** (human, type: approval)
    - Approvers: HR director AND IT security officer (both must approve)
    - Question: "Attest that onboarding completed per company policy?"
    - Shows: summary of all completed steps
    - Timeout: 2 business days → flag for compliance review
    - Stores: dual signatures

11. **Close Onboarding Ticket** (automated, type: extension)
    - Updates HR system: mark employee as onboarded
    - Closes Jira ticket
    - Sends Slack notification to HR and IT: "Onboarding complete for <name>"

### Complexity Tags
- Parallel (5 concurrent provisioning tasks)
- Multi-party (manager approval, dual attestation, employee self-service)
- Long-running (spans multiple days with timeout escalations)

### Key Schema Challenges

1. **Complex data collection form** — Step 1 collects 7 fields with different input types (text, email, date, dropdown, autocomplete). Schema must support rich input validation (email format, date ranges, dropdown options from external API).

2. **Parallel fan-out with heterogeneous tasks** — Step 4 runs 3 automated API calls + 2 human tasks concurrently. Schema must support parallel execution with mixed step types (extension + manual) and wait-all synchronization.

3. **Business day timeout calculations** — Steps 2, 3, 4d, 6, 8, 10 use business days for timeouts (not wall-clock time). Schema must support calendar-aware timeout calculation (skip weekends, holidays).

4. **Multi-party approval with AND logic** — Step 10 requires TWO approvers (HR director AND IT security officer). Schema must support M-of-N approval (both must approve, vs. any-one, vs. quorum).

5. **Conditional access provisioning based on data** — Step 7 provisions different systems based on department field from step 1. Schema must support conditional logic (if department == "Engineering" then provision AWS) within a single step or as dynamic branch selection.

6. **Escalation chains** — Steps 2, 3, 4d, 6 have escalation on timeout (escalate to different approver, not fail). Schema must support escalation policy: primary approver, escalate to secondary after timeout.

7. **Self-service human step** — Step 6 is assigned to the employee (subject of the runbook), not the operator. Schema must support dynamic assignee (use value from step 1 email field).

8. **Failure handling for partial provisioning** — If task 4c (Slack) fails but 4a/4b/4d/4e succeed, what happens? Full rollback (delete Okta, GitHub accounts) or partial completion? Schema must define failure handling strategy for parallel blocks.

---

## Runbook 4: SOC2 Evidence Collection for Audit

**Domain:** Compliance/Audit  
**Complexity:** Nested (invoke sub-runbooks per control) + Artifact collection  
**Interaction:** Automated evidence collection + human attestation

### Summary
Collects evidence for SOC2 audit across 15 controls. Each control invokes a specialized evidence-collection sub-runbook. Aggregates artifacts (logs, screenshots, configs, reports) into audit package. Includes compliance officer attestation.

### Steps

1. **Initialize Audit Package** (automated, type: cli)
   - Creates: directory structure for audit evidence
   - Generates: audit_id (timestamp-based UUID)
   - Creates: manifest.json to track collected evidence
   - Fails if: insufficient disk space

2. **Control CC1.1: Access Reviews** (automated, type: invoke)
   - Invokes sub-runbook: `soc2/cc1.1-access-reviews.yaml`
   - Sub-runbook steps:
     - Export user list from Okta (API call)
     - Export access logs for past 90 days (API call)
     - Generate report: users with admin access
     - Take screenshots of access control settings
   - Outputs: `cc1.1-evidence.tar.gz` (uploaded to audit package)
   - Fails if: sub-runbook fails

3. **Control CC1.2: Password Policy** (automated, type: invoke)
   - Invokes sub-runbook: `soc2/cc1.2-password-policy.yaml`
   - Sub-runbook steps:
     - Export Okta password policy settings
     - Export MFA enforcement rules
     - Query audit log: password reset events
   - Outputs: `cc1.2-evidence.tar.gz`

4. **Control CC2.1: Risk Assessment** (human, type: collector)
   - Prompts compliance officer for:
     - Date of most recent risk assessment (date)
     - Risk register file (file upload)
     - Mitigation status (dropdown: Complete, In Progress, Planned)
   - Stores: uploaded risk register in audit package
   - Fails if: risk assessment older than 1 year

5. **Control CC3.1: Security Monitoring** (automated, type: invoke)
   - Invokes sub-runbook: `soc2/cc3.1-security-monitoring.yaml`
   - Sub-runbook steps:
     - Export CloudTrail logs for past 90 days (AWS API)
     - Export Datadog security alerts
     - Export intrusion detection logs
     - Generate summary report: security events by severity
   - Outputs: `cc3.1-evidence.tar.gz`

6. **Control CC4.1: Change Management** (automated, type: invoke)
   - Invokes sub-runbook: `soc2/cc4.1-change-management.yaml`
   - Sub-runbook steps:
     - Export GitHub pull request history (API call)
     - Export deployment logs from CI/CD system
     - Export Jira change tickets
     - Generate report: changes with approval vs. without
   - Outputs: `cc4.1-evidence.tar.gz`

7. **Control CC5.1: Incident Response** (automated, type: invoke)
   - Invokes sub-runbook: `soc2/cc5.1-incident-response.yaml`
   - Sub-runbook steps:
     - Export PagerDuty incident history
     - Export post-incident reports from wiki
     - Export security incident log
     - Generate summary: MTTD, MTTR metrics
   - Outputs: `cc5.1-evidence.tar.gz`

8. **[... 8 more control sub-runbooks, same pattern ...]** (automated, type: invoke)
   - Controls CC6.1 through CC6.8 (logical/physical access, encryption, network security, etc.)
   - Each invokes specialized sub-runbook
   - Each produces evidence artifact

9. **Aggregate Evidence** (automated, type: cli)
   - Combines all evidence files into single archive
   - Generates: audit-evidence-<audit_id>.tar.gz
   - Computes: SHA256 hash of archive
   - Updates: manifest.json with file listing and hashes

10. **Generate Audit Report** (automated, type: extension)
    - Reads manifest.json
    - Generates HTML report:
      - Control ID, status (evidence collected or missing), artifact filename, file hash
    - Includes: collection timestamp, gert run_id, operator identity
    - Outputs: `audit-report-<audit_id>.html`

11. **Compliance Officer Review** (human, type: approval)
    - Shows: audit report HTML (inline preview)
    - Shows: list of collected artifacts
    - Approver: compliance-officer@company.com
    - Question: "Attest that evidence is complete and accurate?"
    - Checklist:
      - [ ] All 15 controls have evidence
      - [ ] No evidence files are missing or corrupted
      - [ ] Evidence covers required date ranges
    - Timeout: 5 business days → escalate to CISO
    - Stores: compliance officer signature and timestamp

12. **Security Officer Attestation** (human, type: approval)
    - Approver: security-officer@company.com
    - Question: "Attest that evidence collection process followed security controls?"
    - Timeout: 3 business days → escalate to CISO

13. **Upload to Secure Vault** (automated, type: extension)
    - Uploads audit package to S3 bucket with encryption
    - S3 path: `s3://compliance-evidence/<year>/<audit_id>/`
    - Enables: object lock (WORM) for 7 years
    - Records: S3 URI in compliance database

14. **Notify Auditor** (automated, type: extension)
    - Sends email to external auditor: auditor@auditfirm.com
    - Includes: S3 presigned URL (expires in 7 days)
    - Includes: audit report HTML
    - Includes: SHA256 hash of evidence archive

15. **Record Audit Completion** (automated, type: extension)
    - Updates compliance tracking system
    - Records: audit_id, completion date, evidence location, attestations
    - Triggers: reminder for next audit (1 year from now)

### Complexity Tags
- Nested (15 sub-runbook invocations)
- Artifact collection (logs, files, screenshots from each control)
- Multi-party (dual attestation)
- Long-running (spans days)

### Key Schema Challenges

1. **Nested runbook invocation** — Steps 2, 3, 5, 6, 7, 8 invoke sub-runbooks. Schema must support: (a) sub-runbook path resolution, (b) input passing (audit_id to sub-runbooks), (c) output capture (evidence files), (d) failure propagation (sub-runbook failure fails parent).

2. **File artifact collection and aggregation** — Each sub-runbook produces tar.gz file. Parent runbook aggregates them. Schema must support: (a) file output from sub-runbook, (b) file input to parent step, (c) artifact metadata (filename, size, hash), (d) artifact storage location (local, S3, trace attachment).

3. **Evidence integrity (hashing and signing)** — Step 9 computes SHA256 of evidence archive. Step 11 stores compliance officer signature. Schema must support cryptographic operations: hash computation, digital signatures, verification.

4. **Sub-runbook output as approval evidence** — Step 11 approval shows audit report HTML generated in step 10. Schema must support rendering file content in approval prompt (HTML preview, not just filename).

5. **Object lock / WORM storage** — Step 13 uploads to S3 with object lock (immutable for 7 years). Schema must support storage policies beyond simple upload (retention, versioning, encryption).

6. **Presigned URL generation** — Step 14 generates S3 presigned URL with expiration. Schema must support dynamic URL generation (not just static URLs).

7. **Bulk invocation with consistent inputs** — 15 sub-runbooks all need audit_id and date range. Schema must support input templating or variable scoping (define audit_id once, propagate to all sub-runbooks).

8. **Partial failure handling** — If step 5 (CC3.1) fails but other controls succeed, is partial evidence acceptable? Schema must support failure strategy: fail-fast (abort on first failure) vs. best-effort (collect all possible evidence, report failures).

---

## Runbook 5: Security Breach Containment and Forensics

**Domain:** Security Incident Response  
**Complexity:** Branching (breach severity) + Parallel (containment + forensics) + Compensating actions  
**Interaction:** Human decision + automated containment

### Summary
Responds to detected security breach. Assesses breach severity, executes containment actions (isolate systems, revoke credentials), runs forensics in parallel, notifies stakeholders, and initiates remediation. Includes rollback if containment causes production outage.

### Steps

1. **Receive Security Alert** (automated, type: extension)
   - Triggered by: SIEM (Splunk, Datadog Security) webhook
   - Collects: alert ID, severity, affected systems, alert description, timestamp
   - Creates: PagerDuty incident (high urgency)
   - Fails if: webhook payload invalid

2. **Security Officer Triage** (human, type: choice)
   - Shows: alert details, affected systems, initial indicators
   - Prompt: "Assess breach severity:"
   - Options:
     - **Critical** — Active data exfiltration, ransomware, or RCE
     - **High** — Compromised credentials, unauthorized access to production
     - **Medium** — Suspicious activity, potential phishing
     - **Low** — False positive, benign anomaly
   - Assignee: security-oncall@company.com
   - Timeout: 10 minutes → default to High (err on side of caution)
   - Stores: severity assessment and triage notes

3. **Branch by Severity** (automated, type: decision)
   - Routes based on step 2 choice:
     - If Critical → "Critical Containment"
     - If High → "High Containment"
     - If Medium → "Medium Containment"
     - If Low → "Low Containment" (just log and close)

4. **Branch: Critical Containment** (if Critical)
   - Step 4a: Exec approval: "Initiate full lockdown?" (type: approval)
     - Approver: CISO (escalate to CEO if unavailable within 5 minutes)
     - Warning: "This will cause production outage"
   - Step 4b: Register rollback compensation (automated, internal)
     - Registers: "Restore network access, unsuspend accounts" if step 4c/4d/4e fail
   - Step 4c: Isolate affected systems (automated, type: extension, parallel)
     - Parallel tasks:
       - Disable AWS security group ingress (AWS API)
       - Shut down compromised EC2 instances (AWS API)
       - Block attacker IPs at firewall (Palo Alto API)
       - Disable VPN access (Okta API)
     - Wait-all before proceeding
   - Step 4d: Revoke all production credentials (automated, type: extension, parallel)
     - Parallel tasks:
       - Rotate AWS IAM keys (AWS API)
       - Invalidate JWT tokens (auth service API)
       - Force password reset for all users (Okta API)
       - Revoke API keys (internal API)
     - Wait-all before proceeding
   - Step 4e: Snapshot affected systems for forensics (automated, type: extension, parallel)
     - Parallel tasks:
       - Create EBS snapshots of affected EC2 volumes
       - Export CloudTrail logs (past 7 days)
       - Export application logs (past 24 hours)
       - Take memory dumps of running processes
     - Store: snapshots in isolated forensics S3 bucket
   - Step 4f: Verify production health (automated, type: cli, retry)
     - Checks: API endpoints responding, no service degradation
     - Retries: 5 times with 30s backoff
     - If fails: trigger rollback compensation (restore access)

5. **Branch: High Containment** (if High)
   - Step 5a: Suspend compromised accounts (automated, type: extension)
     - Reads: compromised user IDs from alert
     - Calls Okta API: suspend accounts
   - Step 5b: Revoke credentials for affected users (automated, type: extension)
     - Rotates: AWS keys, GitHub tokens, API keys for affected users
   - Step 5c: Isolate affected hosts (automated, type: extension)
     - Modifies security groups to block external traffic
     - Does NOT shut down instances (minimize production impact)
   - Step 5d: Snapshot for forensics (same as 4e but only affected hosts)

6. **Branch: Medium Containment** (if Medium)
   - Step 6a: Human investigation (human, type: manual)
     - Instructions: "Investigate suspicious activity in SIEM"
     - Checklist:
       - [ ] Reviewed user login history
       - [ ] Checked for lateral movement
       - [ ] Determined if compromise occurred
     - Assignee: security analyst
     - Timeout: 1 hour → escalate to senior analyst
   - Step 6b: Decision: "Compromise confirmed?" (human, type: choice)
     - Options: Yes (route to High Containment), No (route to Low Containment)

7. **Branch: Low Containment** (if Low)
   - Step 7a: Log false positive (automated, type: cli)
     - Appends to false positive log
   - Step 7b: Close PagerDuty incident (automated, type: extension)
   - Step 7c: End runbook (automated, type: end)

8. **Forensics Investigation** (automated + human, parallel with containment)
   - Runs in parallel with containment steps (4, 5, 6)
   - Step 8a: Analyze logs for IOCs (automated, type: extension)
     - Queries Splunk: search for known IOCs (IP addresses, file hashes, domains)
     - Generates: IOC match report
   - Step 8b: Malware analysis (human, type: manual)
     - Instructions: "Submit suspicious files to VirusTotal and internal sandbox"
     - Assignee: malware analyst
   - Step 8c: Timeline reconstruction (human, type: manual)
     - Instructions: "Build attack timeline from logs"
     - Assignee: forensics lead

9. **Stakeholder Notification** (automated, type: extension)
   - Sends email to: CISO, CTO, CEO, legal counsel
   - Includes: severity, affected systems, containment actions, initial findings
   - If severity == Critical: also send to board of directors

10. **Legal/Compliance Assessment** (human, type: approval)
    - Approver: legal counsel
    - Question: "Is breach notification required (GDPR, CCPA, state laws)?"
    - Shows: affected systems, user data types, jurisdictions
    - Timeout: 4 hours → escalate to outside counsel

11. **Customer Notification** (conditional, human, type: approval)
    - Only runs if step 10 == "Yes"
    - Approver: CEO AND legal counsel (both must approve)
    - Shows: draft customer notification email
    - Timeout: 24 hours (regulatory deadline)

12. **Remediation Planning** (human, type: collector)
    - Prompts security team for:
      - Root cause (text)
      - Remediation steps (multi-line text)
      - Responsible team (dropdown)
      - Target completion date (date)
    - Stores: remediation plan in incident database

13. **Close Incident** (automated, type: extension)
    - Updates incident status: contained (not resolved)
    - Creates: follow-up Jira ticket for remediation
    - Archives: all forensics evidence to S3 with 7-year retention
    - Sends: Slack notification to security team

### Complexity Tags
- Branching (4-way by severity)
- Parallel (containment + forensics run concurrently)
- Compensation/rollback (restore access if containment breaks production)
- Multi-party (exec approval, dual approval for customer notification)

### Key Schema Challenges

1. **Human choice with timeout default** — Step 2 asks security officer to choose severity, defaults to "High" on 10-minute timeout. Schema must support choice step with default option.

2. **Escalating approval path** — Step 4a approver is CISO, but if unavailable within 5 minutes, escalates to CEO. Schema must support approval with primary/backup approvers and dynamic escalation.

3. **Compensation registration with conditional trigger** — Step 4b registers rollback actions to execute if step 4f fails (production health check). Schema must support conditional compensation: only execute if specific step fails, not all failures.

4. **Parallel containment + forensics** — Steps 4c-4f (containment) run in parallel with step 8 (forensics). Two independent parallel flows, not fan-out/fan-in. Schema must support concurrent branches, not just sequential branches.

5. **Nested parallel within branch** — Step 4c is parallel (4 tasks), nested inside branch "Critical Containment". Schema must support parallel blocks within conditional branches.

6. **Mid-runbook decision with branch routing** — Step 6b (human decision) routes to either High Containment (step 5) or Low Containment (step 7). Schema must support jumping to different branches mid-flow, not just forward progression.

7. **Conditional step execution** — Step 11 (customer notification) only runs if step 10 == "Yes". Schema must support step-level conditional execution (different from branch routing).

8. **High-stakes approval with production impact warning** — Step 4a shows warning about production outage. Schema must support rich approval prompts: warnings, impact descriptions, not just yes/no questions.

---

## Runbook 6: Database Migration with Dry-Run and Validation

**Domain:** Data Engineering  
**Complexity:** Linear + Looping (retry) + Compensating actions  
**Interaction:** Human approval + automated validation

### Summary
Migrates production database schema (add columns, indexes, foreign keys). Includes dry-run on staging, validation queries, production migration, post-migration validation, and automatic rollback on failure.

### Steps

1. **Pre-Flight Checks** (automated, type: assert)
   - Asserts:
     - Migration script exists: `ls migrations/20260418_add_user_preferences.sql`
     - Staging database reachable: `pg_isready -h staging-db`
     - Production database reachable: `pg_isready -h prod-db`
     - Backup completed within past 24h: query backup metadata
   - Fails if: any assertion fails

2. **Dry-Run on Staging** (automated, type: cli)
   - Executes: `psql -h staging-db -f migrations/20260418_add_user_preferences.sql`
   - Captures: stdout, stderr, execution time
   - Fails if: exit code != 0 (syntax error, constraint violation)

3. **Staging Validation Queries** (automated, type: cli)
   - Executes validation SQL:
     ```sql
     SELECT column_name FROM information_schema.columns 
     WHERE table_name = 'users' AND column_name = 'preferences';
     
     SELECT indexname FROM pg_indexes 
     WHERE tablename = 'users' AND indexname = 'idx_users_preferences';
     ```
   - Asserts: new column and index exist
   - Fails if: validation query returns no rows

4. **Estimate Production Migration Time** (automated, type: cli)
   - Executes: `EXPLAIN ANALYZE <migration_sql>` on production replica
   - Parses: estimated execution time from EXPLAIN output
   - Stores: estimated_duration (for approval prompt)

5. **DBA Approval: Proceed to Production** (human, type: approval)
   - Shows: migration SQL, dry-run results, estimated duration
   - Approver: dba-team@company.com
   - Question: "Approve production migration?"
   - Warning: "This will acquire table lock for ~<estimated_duration>"
   - Timeout: 8 hours (maintenance window expires)

6. **Schedule Maintenance Window** (human, type: collector)
   - Prompts DBA for:
     - Maintenance start time (datetime)
     - Expected duration (minutes)
     - Notification message (text)
   - Sends: maintenance notification to #engineering Slack channel
   - Stores: maintenance window metadata

7. **Wait for Maintenance Window** (automated, type: cli)
   - Sleeps until maintenance start time
   - Executes: `sleep $(( $(date -d '<maintenance_start>' +%s) - $(date +%s) ))`

8. **Take Pre-Migration Snapshot** (automated, type: cli)
   - Executes: `pg_dump -h prod-db -Fc -f /backup/pre-migration-$(date +%s).dump`
   - Verifies: dump file size > 0, no errors in pg_dump output
   - Stores: snapshot path for rollback
   - Fails if: insufficient disk space, dump fails

9. **Register Rollback Compensation** (automated, internal)
   - Registers: "Restore from snapshot and revert schema" if any subsequent step fails
   - Compensation steps:
     - Drop new column: `ALTER TABLE users DROP COLUMN preferences;`
     - Drop new index: `DROP INDEX idx_users_preferences;`
     - Restore from snapshot if DROP fails: `pg_restore`

10. **Execute Production Migration** (automated, type: cli, with retry)
    - Executes: `psql -h prod-db -f migrations/20260418_add_user_preferences.sql`
    - Timeout: 30 minutes (kill if exceeds)
    - Retries: 2 times with 5 minute backoff (handles transient deadlocks)
    - Captures: stdout, stderr, execution time
    - Fails if: exit code != 0, timeout, or max retries exceeded → triggers rollback

11. **Production Validation Queries** (automated, type: cli, retry loop)
    - Executes same validation SQL as step 3, but on production
    - Retries: 5 times with 10s backoff (handles replication lag)
    - Asserts: new column and index exist, no data corruption
    - Fails if: validation fails → triggers rollback

12. **Smoke Tests** (automated, type: cli, parallel)
    - Parallel tests:
      - Test 1: Insert new row with preferences column (type: cli)
      - Test 2: Query with new index (type: cli, check EXPLAIN uses index)
      - Test 3: Update existing row preferences column (type: cli)
      - Test 4: Verify foreign key constraints (type: cli)
    - All tests must pass
    - Fails if: any test fails → triggers rollback

13. **Application Health Check** (automated, type: cli, iterate)
    - Loop for 5 minutes (30 iterations, 10s interval):
      - Query: `curl https://api.company.com/health`
      - Check: response status 200, response time < 1s
    - Convergence: 10 consecutive healthy checks
    - Early exit: if any check fails → triggers rollback

14. **DBA Confirmation: Migration Successful** (human, type: approval)
    - Shows: migration execution time, validation results, smoke test results, health checks
    - Approver: dba-team@company.com
    - Question: "Confirm migration is stable?"
    - Timeout: 30 minutes → auto-rollback (DBA not monitoring)

15. **Clear Rollback Compensation** (automated, internal)
    - Unregisters rollback (migration confirmed successful)

16. **Post-Migration Cleanup** (automated, type: cli)
    - Executes: `VACUUM ANALYZE users;` (rebuild statistics)
    - Sends: maintenance complete notification to Slack
    - Records: migration in schema_versions table

17. **Delete Pre-Migration Snapshot** (automated, type: cli)
    - Deletes: `/backup/pre-migration-*.dump`
    - Only runs if step 14 approved (migration confirmed stable)

### Complexity Tags
- Linear with looping (retry on transient failures)
- Compensation/rollback (automatic revert on failure)
- Iterate (health check with convergence)

### Key Schema Challenges

1. **Time-based wait** — Step 7 sleeps until maintenance window. Schema must support datetime-based wait (not just duration), handling timezone, DST.

2. **Compensation with multi-step rollback** — Step 9 registers 3-step rollback: (1) DROP COLUMN, (2) DROP INDEX, (3) pg_restore if DROP fails. Schema must support multi-step compensation with fallback logic.

3. **Automatic rollback on approval timeout** — Step 14 timeout triggers rollback (not escalation). Schema must support timeout action: execute compensation vs. escalate vs. fail.

4. **Retry with backoff for transient failures** — Step 10 retries migration on deadlock. Schema must distinguish transient errors (retry) from permanent errors (fail immediately).

5. **Conditional cleanup** — Step 17 only deletes snapshot if step 14 approved. Schema must support conditional step execution based on earlier step outcome.

6. **Compensation scope** — Rollback compensation registered at step 9 applies to steps 10-14. Schema must define compensation scope (which steps are protected).

7. **Convergence-based iterate** — Step 13 loops until 10 consecutive healthy checks. Schema must support convergence condition (not just boolean exit condition).

8. **Evidence capture for audit** — All SQL queries, execution times, validation results must be captured for audit trail. Schema must support detailed evidence capture for CLI steps (stdout, stderr, exit code, duration, timestamp).

---

## Runbook 7: Financial Approval for Large Purchase

**Domain:** Finance  
**Complexity:** Multi-level approval chain with escalation  
**Interaction:** Human data collection + multi-party approval

### Summary
Handles approval workflow for large capital expenditure ($50k+). Includes requester data collection, manager approval, finance approval, executive approval (based on amount), and vendor notification.

### Steps

1. **Purchase Request Form** (human, type: collector)
   - Prompts requester for:
     - Item description (text, max 500 chars)
     - Vendor name (text)
     - Amount (currency, USD)
     - Business justification (multi-line text)
     - Budget line item (dropdown from finance system)
     - Requested delivery date (date)
     - Requester email (email)
     - Department (dropdown)
   - Validations:
     - Amount > $50,000 (else route to simplified approval)
     - Budget line item has sufficient funds
   - Stores: purchase_request object
   - Fails if: budget insufficient

2. **Manager Approval: Justify Business Need** (human, type: approval)
   - Shows: purchase_request summary
   - Approver: requester's manager (lookup from HR system using requester email)
   - Question: "Approve purchase as necessary for business?"
   - Timeout: 2 business days → escalate to director
   - Stores: manager approval timestamp and comments

3. **Finance Review: Budget Verification** (human, type: approval)
   - Shows: purchase_request, manager approval, current budget status
   - Approver: finance-team@company.com
   - Question: "Confirm budget availability and procurement policy compliance?"
   - Checklist:
     - [ ] Budget line item has sufficient funds
     - [ ] Vendor is on approved vendor list
     - [ ] Purchase complies with procurement policy
   - Timeout: 3 business days → escalate to finance director

4. **Approval Tier Decision** (automated, type: decision)
   - Evaluates purchase_request.amount:
     - If amount < $100k → route to "Director Approval"
     - If amount >= $100k AND < $500k → route to "VP Approval"
     - If amount >= $500k → route to "CFO Approval"

5. **Branch: Director Approval** (if $50k-$100k)
   - Step 5a: Approver: department director (lookup from org chart)
   - Timeout: 3 business days → escalate to VP

6. **Branch: VP Approval** (if $100k-$500k)
   - Step 6a: Approver: department VP (lookup from org chart)
   - Timeout: 5 business days → escalate to CFO

7. **Branch: CFO Approval** (if $500k+)
   - Step 7a: Approver: CFO
   - Timeout: 7 business days → escalate to CEO
   - Additional check: Board approval required if amount >= $1M
   - Step 7b: Board Approval (if amount >= $1M) (human, type: approval)
     - Approvers: board-members@company.com (quorum: 3 of 5 must approve)
     - Timeout: 14 business days → defer to next board meeting

8. **Procurement: Vendor Notification** (automated, type: extension)
   - Sends: purchase order to vendor via email
   - Includes: PO number, item description, amount, delivery date
   - Calls: procurement system API to create PO record
   - Stores: PO number

9. **Contract Execution** (human, type: manual)
   - Instructions: "Upload signed contract and vendor quote"
   - File uploads:
     - Signed contract (PDF)
     - Vendor quote (PDF)
   - Assignee: procurement-team@company.com
   - Timeout: 10 business days → escalate to procurement manager

10. **Finance: Record Purchase** (automated, type: extension)
    - Calls: finance system API to record transaction
    - Debits: budget line item
    - Creates: accounts payable entry
    - Stores: transaction ID

11. **Notify Requester** (automated, type: extension)
    - Sends: email to requester with PO number, expected delivery date
    - Includes: link to track purchase status

12. **Close Approval Workflow** (automated, type: extension)
    - Updates: workflow status to "approved"
    - Archives: all approvals and documents
    - Sends: Slack notification to finance team

### Complexity Tags
- Multi-level approval chain (manager → finance → director/VP/CFO/board)
- Branching (approval tier by amount)
- Multi-party with quorum (board approval requires 3 of 5)
- Long-running (spans weeks)

### Key Schema Challenges

1. **Dynamic approver lookup** — Step 2 approver is requester's manager (looked up from HR system). Step 5/6/7 approvers are director/VP/CFO (looked up from org chart). Schema must support dynamic approver resolution via external API or database query.

2. **Multi-level escalation chain** — Each approval has timeout → escalate to next level. Schema must support escalation hierarchy: manager → director → VP → CFO → CEO.

3. **Amount-based branching** — Step 4 routes based on numeric comparison (amount thresholds). Schema must support numeric comparison in decision logic.

4. **Quorum approval** — Step 7b requires 3 of 5 board members to approve. Schema must support M-of-N approval logic (not just all-must-approve or any-one).

5. **Conditional nested approval** — Step 7b (board) only runs if amount >= $1M. Schema must support conditional step within a branch.

6. **Budget validation before proceeding** — Step 1 checks budget availability. If insufficient, runbook should fail early (not proceed to approvals). Schema must support pre-condition checks with abort.

7. **Business day timeout** — All timeouts use business days. Schema must support calendar-aware timeout (already noted in Runbook 3).

8. **Artifact collection from human** — Step 9 collects two file uploads (contract, quote). Schema must support file upload in manual steps with metadata.

---

## Runbook 8: Medical Device Software Release (FDA/Quality)

**Domain:** Regulated Industry (Medical Device)  
**Complexity:** Linear + Multi-party attestation + Extensive evidence  
**Interaction:** Human attestation + automated validation

### Summary
Releases software update for FDA-regulated Class II medical device. Includes design verification, risk analysis, clinical validation, quality assurance attestation, regulatory documentation, and FDA submission preparation. Highly regulated with extensive evidence capture.

### Steps

1. **Design Verification Review** (human, type: approval)
   - Shows: design verification report (DVR) for software changes
   - Approver: quality-assurance@company.com
   - Question: "Attest that design verification testing is complete per IEC 62304 Level C?"
   - Checklist:
     - [ ] Unit tests passed (>95% coverage)
     - [ ] Integration tests passed
     - [ ] System tests passed
     - [ ] Traceability matrix updated (requirements → tests)
     - [ ] Test results uploaded to QMS
   - Timeout: 5 business days → escalate to QA manager
   - Stores: QA attestation signature

2. **Risk Analysis Verification** (human, type: approval)
   - Shows: risk analysis document (ISO 14971)
   - Approver: risk-management@company.com
   - Question: "Attest that risk analysis is complete and all risks are acceptable?"
   - Checklist:
     - [ ] Hazard analysis completed
     - [ ] Risk control measures implemented
     - [ ] Residual risks evaluated and accepted
     - [ ] Risk management file updated
   - Timeout: 5 business days → escalate to risk manager

3. **Clinical Validation** (human, type: approval)
   - Shows: clinical validation summary
   - Approver: clinical-affairs@company.com
   - Question: "Attest that clinical validation demonstrates safety and efficacy?"
   - Checklist:
     - [ ] Clinical study completed (if required)
     - [ ] Clinical data analyzed
     - [ ] No adverse events reported
     - [ ] Clinical validation report uploaded to QMS
   - Only required if: software changes affect clinical functionality
   - Timeout: 10 business days → escalate to clinical manager

4. **Cybersecurity Assessment** (human, type: approval)
   - Shows: cybersecurity assessment report
   - Approver: security-team@company.com
   - Question: "Attest that cybersecurity risks are mitigated per FDA guidance?"
   - Checklist:
     - [ ] Threat modeling completed
     - [ ] Vulnerability scanning passed
     - [ ] Penetration testing completed
     - [ ] Security controls implemented
     - [ ] Cybersecurity bill of materials (CBOM) generated
   - Timeout: 5 business days → escalate to security manager

5. **Regulatory Affairs Review** (human, type: approval)
   - Shows: software version, changes summary, validation summary
   - Approver: regulatory-affairs@company.com
   - Question: "Determine regulatory submission requirement:"
   - Options:
     - **No submission** — Minor software change, exempt per FDA guidance
     - **Letter to File** — Moderate change, document in DHF
     - **Special 510(k)** — Significant change, submit to FDA
   - Stores: submission determination and rationale

6. **Branch by Submission Type** (automated, type: decision)
   - Routes based on step 5 choice:
     - If "No submission" → skip to step 10
     - If "Letter to File" → step 7
     - If "Special 510(k)" → step 8

7. **Branch: Letter to File** (if moderate change)
   - Step 7a: Generate letter to file (automated, type: extension)
     - Template: software change summary, verification results, risk analysis
   - Step 7b: Upload to QMS (human, type: manual)
     - Instructions: "Upload letter to file to document control system"
   - Step 7c: Quality Manager signature (human, type: approval)
     - Approver: quality-manager@company.com

8. **Branch: Special 510(k) Submission** (if significant change)
   - Step 8a: Prepare 510(k) submission package (human, type: manual)
     - Instructions: "Compile 510(k) submission per FDA template"
     - File uploads:
       - Software description
       - Verification and validation report
       - Risk analysis
       - Cybersecurity documentation
       - Labeling
     - Assignee: regulatory-affairs@company.com
     - Timeout: 30 business days
   - Step 8b: Quality Manager review (human, type: approval)
     - Approver: quality-manager@company.com
   - Step 8c: CEO signature (human, type: approval)
     - Approver: CEO (legally responsible party)
   - Step 8d: Submit to FDA (automated, type: extension)
     - Uploads: 510(k) package to FDA eSTAR portal
     - Stores: FDA submission ID
   - Step 8e: Wait for FDA clearance (human, type: manual)
     - Instructions: "Monitor FDA review status, respond to questions"
     - Timeout: 90 days (FDA statutory review period)
     - Note: This step may pause runbook for months

9. **Document Control** (automated, type: extension)
   - Updates: document management system with new software version
   - Archives: design history file (DHF) documents
   - Generates: device history record (DHR) entry

10. **Manufacturing Release Approval** (human, type: approval)
    - Shows: summary of all attestations, submission status
    - Approvers: quality-manager@company.com AND ceo@company.com (both must approve)
    - Question: "Approve release to manufacturing?"
    - Stores: dual signatures

11. **Sign Software Bill of Materials** (automated, type: cli)
    - Generates: SBOM (SPDX format) for software release
    - Signs: SBOM with company code signing certificate
    - Uploads: signed SBOM to artifact repository

12. **Build Release Artifact** (automated, type: cli)
    - Executes: `./build-release.sh <version>`
    - Generates: signed binary, release notes, installation instructions
    - Computes: SHA256 hash of release artifact
    - Uploads: to secure artifact repository

13. **Release to Manufacturing** (automated, type: extension)
    - Notifies: manufacturing team via email
    - Creates: Jira ticket for manufacturing process
    - Sends: release package to manufacturing file share

14. **Post-Market Surveillance Setup** (automated, type: extension)
    - Configures: device telemetry monitoring for new version
    - Sets: alert rules for adverse events
    - Creates: post-market surveillance dashboard

15. **Close Release Record** (automated, type: extension)
    - Records: release in product lifecycle management (PLM) system
    - Stores: all attestation signatures, submission documents, evidence
    - Generates: release certificate (PDF with all signatures)
    - Archives: to QMS with 10-year retention

### Complexity Tags
- Linear with branching (submission type)
- Multi-party attestation (5+ approvers across quality, risk, clinical, security, regulatory, CEO)
- Extensive evidence capture (all documents, signatures, test results)
- Long-running (may pause for FDA review, 90+ days)

### Key Schema Challenges

1. **Conditional step execution based on device risk** — Step 3 (clinical validation) only required if software affects clinical functionality. Schema must support step-level conditional execution with explanation for audit trail.

2. **Human choice with regulatory implications** — Step 5 decision determines regulatory path. Choice must be recorded with rationale for FDA audit. Schema must support choice steps with mandatory rationale field.

3. **Long pause for external process** — Step 8e waits for FDA clearance (may take 90+ days). Runbook must pause and resume months later. Schema must support long-running pauses with external triggers (webhook from FDA portal).

4. **Extensive signature capture** — 7+ approvals with digital signatures for 21 CFR Part 11 compliance. Schema must support signature metadata: signer identity, timestamp, signature algorithm, certificate thumbprint.

5. **Document versioning and archival** — All documents (DVR, risk analysis, 510(k)) must be versioned and archived for 10 years. Schema must support artifact versioning and retention policies.

6. **Cryptographic signing of release artifact** — Step 11 signs SBOM with code signing certificate. Schema must support cryptographic operations with private key management.

7. **Audit trail for entire process** — FDA requires complete audit trail: who did what, when, with what evidence. Schema must capture detailed provenance for every step, approval, and artifact.

8. **Quorum approval for manufacturing release** — Step 10 requires quality manager AND CEO (both must approve, not either). Same as Runbook 7 board approval.

---

## Runbook 9: On-Call Escalation Ladder

**Domain:** SRE/Operations  
**Complexity:** Looping (escalation retry) + Multi-party (escalation chain)  
**Interaction:** Automated escalation + human acknowledgment

### Summary
Escalates critical alerts through on-call chain until acknowledged. Tries primary on-call (page + wait), then secondary, then manager, then director. Includes timeout at each level and simultaneous notification to all if no acknowledgment after 30 minutes.

### Steps

1. **Receive Critical Alert** (automated, type: extension)
   - Triggered by: monitoring system (Datadog, PagerDuty) webhook
   - Collects: alert severity, affected service, alert description, timestamp
   - Only proceeds if: severity == "critical"
   - Stores: alert_id for correlation

2. **Lookup On-Call Schedule** (automated, type: extension)
   - Calls PagerDuty API: get current on-call schedule for service
   - Returns:
     - Primary on-call engineer
     - Secondary on-call engineer
     - Manager on-call
     - Director on-call
   - Stores: escalation_chain list
   - Fails if: no on-call schedule configured

3. **Escalation Loop: Try Primary** (automated, type: iterate)
   - Max passes: 3
   - Interval: 5 minutes
   - Step 3a: Page primary on-call (type: extension)
     - Calls PagerDuty API: create high-urgency incident assigned to primary
     - Sends: SMS, phone call, push notification
   - Step 3b: Wait for acknowledgment (type: extension)
     - Polls PagerDuty API: check incident status
     - Wait: 5 minutes
   - Convergence: incident status == "acknowledged"
   - Early exit if: incident acknowledged before max passes
   - If not acknowledged after 3 passes → proceed to step 4

4. **Escalation Loop: Try Secondary** (automated, type: iterate)
   - Max passes: 3
   - Interval: 5 minutes
   - Same pattern as step 3, but page secondary on-call
   - If not acknowledged after 3 passes → proceed to step 5

5. **Escalation Loop: Try Manager** (automated, type: iterate)
   - Max passes: 2
   - Interval: 5 minutes
   - Same pattern, but page manager on-call
   - If not acknowledged after 2 passes → proceed to step 6

6. **Escalation Loop: Try Director** (automated, type: iterate)
   - Max passes: 2
   - Interval: 5 minutes
   - Same pattern, but page director on-call
   - If not acknowledged after 2 passes → proceed to step 7

7. **Emergency: Broadcast to All** (automated, type: extension)
   - Sends: SMS + phone call + push notification to ALL engineers in escalation chain simultaneously
   - Creates: war room Zoom meeting, sends link
   - Sends: Slack message to #critical-alerts channel mentioning @here
   - Calls: backup escalation contact (VP Engineering or CTO)

8. **Wait for Any Acknowledgment** (automated, type: extension, polling loop)
   - Polls PagerDuty API every 1 minute
   - Wait: up to 60 minutes
   - Convergence: any engineer acknowledges incident
   - If no acknowledgment after 60 minutes → proceed to step 9

9. **Executive Escalation** (automated, type: extension)
   - Sends: SMS + phone call to CTO and CEO
   - Creates: critical incident record in incident management system
   - Triggers: automated failover (if configured for service)

10. **Incident Acknowledged** (automated, convergent from steps 3-9)
    - Records: who acknowledged, timestamp, escalation level reached
    - Sends: Slack message to #critical-alerts: "<engineer> acknowledged incident"
    - Creates: incident response Slack channel
    - Invokes: incident response runbook (separate runbook)

11. **Post-Incident Escalation Report** (automated, type: extension)
    - Generates: escalation timeline (who was paged when, who acknowledged)
    - Sends: report to engineering leadership
    - Flags: if escalation reached manager+ level (indicates on-call issue)

### Complexity Tags
- Looping (retry escalation at each level)
- Multi-party (escalation chain)
- Early exit (stop escalating once acknowledged)
- Automated (no human interaction, fully automated escalation)

### Key Schema Challenges

1. **Nested iterate loops** — Steps 3, 4, 5, 6 are four sequential iterate blocks, each with convergence condition (acknowledged). Schema must support sequential iterates with early exit (once acknowledged, skip remaining escalation levels).

2. **Polling with convergence** — Each iterate polls PagerDuty API to check acknowledgment status. Schema must support polling pattern: call API, check condition, wait, repeat.

3. **Dynamic escalation chain** — Escalation chain (primary, secondary, manager, director) is looked up at runtime from PagerDuty API. Schema must support dynamic iteration over list (not hardcoded escalation levels).

4. **Cross-iterate convergence** — Once any iterate (step 3, 4, 5, 6, or 8) converges, runbook proceeds to step 10 (incident acknowledged). Schema must support convergence point across multiple loops.

5. **Fully automated runbook** — No human approval or manual steps. Schema must support fully automated runbooks (common for incident response).

6. **Time-based escalation** — Escalation timing is critical (5 min per attempt, 3 attempts = 15 min per level). Schema must enforce precise timing (not just approximate).

7. **Simultaneous notification** — Step 7 pages all engineers at once (not sequential). Different from parallel step execution (no wait-all, just send all notifications).

---

## Runbook 10: Customer Data Deletion (GDPR/CCPA Compliance)

**Domain:** Compliance (GDPR/CCPA)  
**Complexity:** Parallel (deletion across systems) + Verification loop + Attestation  
**Interaction:** Human approval + automated deletion

### Summary
Processes customer data deletion request (GDPR "right to be forgotten"). Verifies request authenticity, obtains legal approval, deletes data from all systems (database, S3, logs, backups, analytics), verifies deletion, and generates deletion certificate.

### Steps

1. **Receive Deletion Request** (automated, type: extension)
   - Triggered by: customer submits deletion request via web form
   - Collects: customer email, request reason, request timestamp
   - Generates: deletion_request_id
   - Stores: request in compliance database

2. **Identity Verification** (human, type: manual)
   - Instructions: "Verify customer identity per data protection policy"
   - Checklist:
     - [ ] Customer responded to verification email
     - [ ] Customer provided identity document (if high-value account)
     - [ ] Verification completed within 30 days
   - Assignee: privacy-team@company.com
   - Timeout: 30 days (GDPR deadline)
   - Stores: verification attestation

3. **Legal Review** (human, type: approval)
   - Shows: deletion request, customer account details
   - Approver: legal-counsel@company.com
   - Question: "Approve data deletion? Check for legal holds."
   - Checklist:
     - [ ] No active litigation involving customer
     - [ ] No regulatory investigation requiring data retention
     - [ ] No other legal basis to retain data
   - Timeout: 5 business days
   - Stores: legal approval

4. **Search All Data Stores** (automated, type: extension, parallel)
   - Parallel searches across all systems:
     - Step 4a: Search production database (type: cli)
       - Executes: `SELECT * FROM users WHERE email = '<customer_email>'`
       - Executes: `SELECT * FROM orders WHERE user_id = '<user_id>'`
       - Stores: list of tables containing customer data
     - Step 4b: Search S3 buckets (type: cli)
       - Executes: `aws s3api list-objects --query "Contents[?contains(Key, '<user_id>')]"`
       - Stores: list of S3 keys
     - Step 4c: Search application logs (type: extension)
       - Queries Elasticsearch: `email:<customer_email>`
       - Stores: log entries with customer data
     - Step 4d: Search analytics (type: extension)
       - Queries Mixpanel API: get user profile
       - Stores: analytics events
     - Step 4e: Search CRM (type: extension)
       - Queries Salesforce API: get account records
       - Stores: CRM records
     - Step 4f: Search support tickets (type: extension)
       - Queries Zendesk API: search tickets by email
       - Stores: ticket IDs
   - Wait-all before proceeding
   - Stores: aggregated inventory of customer data

5. **DPO Review: Deletion Scope** (human, type: approval)
   - Shows: inventory of customer data from step 4
   - Approver: data-protection-officer@company.com
   - Question: "Confirm deletion scope is complete?"
   - Warning: "Deletion is permanent and cannot be undone"
   - Timeout: 3 business days

6. **Parallel Deletion** (automated, type: extension, parallel)
   - Execute deletions concurrently across all systems:
     - Step 6a: Delete from production database (type: cli)
       - Executes: `DELETE FROM users WHERE user_id = '<user_id>'`
       - Executes: `DELETE FROM orders WHERE user_id = '<user_id>'`
       - (Additional DELETE statements for all tables from step 4a)
     - Step 6b: Delete from S3 (type: cli)
       - Executes: `aws s3 rm <s3_key>` for each key from step 4b
     - Step 6c: Delete from logs (type: extension)
       - Redacts customer email/PII from Elasticsearch (replace with "[DELETED]")
     - Step 6d: Delete from analytics (type: extension)
       - Calls Mixpanel API: delete user profile
     - Step 6e: Delete from CRM (type: extension)
       - Calls Salesforce API: delete account records
     - Step 6f: Delete from support tickets (type: extension)
       - Calls Zendesk API: redact customer PII from tickets
   - Failure handling: best-effort deletion (log failures, continue with other systems)

7. **Delete from Backups** (automated, type: cli, iterate)
   - Loop through all backup generations (daily for 30 days):
     - Identify: backups containing customer data
     - Recreate: backup without customer data (or mark records as deleted)
   - Max passes: 30 (daily backups)
   - Stores: list of modified backups

8. **Verification Loop** (automated, type: iterate)
   - Max passes: 5
   - Interval: 1 hour (allow replication lag)
   - Step 8a: Verify database deletion (type: cli)
     - Executes: `SELECT * FROM users WHERE user_id = '<user_id>'`
     - Asserts: no rows returned
   - Step 8b: Verify S3 deletion (type: cli)
     - Executes: `aws s3api head-object --key <s3_key>`
     - Asserts: object not found
   - Step 8c: Verify analytics deletion (type: extension)
     - Queries Mixpanel API: get user profile
     - Asserts: user not found
   - Convergence: all verifications pass
   - Fails if: data still present after 5 passes → manual investigation required

9. **Generate Deletion Certificate** (automated, type: extension)
   - Generates PDF certificate:
     - Deletion request ID
     - Customer email (redacted: first 2 chars + ***)
     - Deletion timestamp
     - Systems deleted from (list)
     - DPO signature
     - Company signature
   - Signs: PDF with company digital signature
   - Stores: certificate in compliance archive (7-year retention)

10. **DPO Attestation** (human, type: approval)
    - Shows: deletion certificate, verification results
    - Approver: data-protection-officer@company.com
    - Question: "Attest that data deletion is complete and verified?"
    - Stores: DPO signature

11. **Notify Customer** (automated, type: extension)
    - Sends: email to customer confirming deletion
    - Includes: deletion certificate (PDF attachment)
    - Records: notification sent timestamp (GDPR requires confirmation)

12. **Update Compliance Log** (automated, type: extension)
    - Records deletion in GDPR compliance log:
      - Request date, completion date, duration
      - Data inventory, systems deleted from
      - Verification results, DPO attestation
    - Flags: if deletion took >30 days (GDPR violation)

13. **Close Deletion Request** (automated, type: extension)
    - Updates: deletion request status to "completed"
    - Archives: all evidence (inventory, deletion logs, certificate)
    - Sends: Slack notification to privacy team

### Complexity Tags
- Parallel (search across 6 systems, delete across 6 systems)
- Looping (backup deletion, verification loop)
- Multi-party (legal, DPO attestations)
- Long-running (30-day deadline)
- Best-effort failure handling (some deletions may fail)

### Key Schema Challenges

1. **Dynamic data inventory** — Step 4 searches 6+ systems and stores inventory. Inventory size unknown (could be 0 records or 1000s). Schema must support variable-length data structures as step outputs.

2. **Best-effort parallel deletion** — Step 6 attempts deletion across all systems but continues even if some fail. Schema must support failure mode: continue-on-failure vs. fail-fast for parallel blocks.

3. **Backup iteration over date range** — Step 7 loops through 30 daily backups. Schema must support iteration over date ranges (not just numeric counter).

4. **Verification with eventual consistency** — Step 8 verifies deletion but allows 1-hour retry (replication lag). Schema must support retry with generous backoff (not fail immediately).

5. **Digital signature of certificate** — Step 9 signs PDF with company certificate. Schema must support cryptographic signing with certificate management.

6. **Compliance deadline tracking** — Step 12 flags if deletion took >30 days (GDPR deadline). Schema must support SLA tracking and violation reporting.

7. **Partial failure reporting** — If step 6c (log redaction) fails but other deletions succeed, report partial success. Schema must capture detailed failure reasons per sub-task in parallel block.

8. **Data redaction vs. deletion** — Steps 6c and 6f redact (not delete) data from logs and support tickets (can't delete logs, only redact PII). Schema must distinguish deletion operations (remove record) from redaction operations (replace sensitive fields).

---

## Coverage Summary

### Domain Coverage
✅ SRE/Operations (Runbooks 1, 9)  
✅ DevOps/Deployment (Runbook 2)  
✅ HR/IT (Runbook 3)  
✅ Compliance/Audit (Runbooks 4, 10)  
✅ Security (Runbook 5)  
✅ Data Engineering (Runbook 6)  
✅ Finance (Runbook 7)  
✅ Regulated Industry (Runbook 8)

### Complexity Coverage
✅ Linear (Runbooks 6, 8)  
✅ Branching (Runbooks 1, 2, 5, 7, 8)  
✅ Parallel (Runbooks 2, 3, 4, 5, 10)  
✅ Looping/Retry (Runbooks 1, 2, 6, 9, 10)  
✅ Nested/Invoke (Runbook 4)  
✅ Multi-party (Runbooks 3, 4, 5, 7, 8, 10)

### Interaction Pattern Coverage
✅ Fully automated (Runbook 9)  
✅ Human approval gates (all runbooks)  
✅ User data collection (Runbooks 3, 4, 7, 10)  
✅ File/artifact collection (Runbooks 1, 4, 7, 8)  
✅ Multi-party approval (Runbooks 3, 7, 8, 10)  
✅ Self-service (Runbook 3 step 6)

### Failure Mode Coverage
✅ Compensating actions/rollback (Runbooks 1, 2, 5, 6)  
✅ Partial completion (Runbooks 3, 10)  
✅ Timeout/SLA (Runbooks 1, 2, 3, 4, 5, 6, 7, 8, 9, 10)  
✅ Skip-on-condition (Runbooks 5, 8)  
✅ Retry with backoff (Runbooks 1, 2, 6, 10)

### Edge Case Coverage
✅ Long runbooks (Runbook 8: 15 steps)  
✅ No human interaction (Runbook 9: fully automated)  
✅ All human steps (Runbook 7: mostly human approvals)  
✅ Cross-system workflows (Runbooks 3, 4, 10)  
✅ Long-running with pauses (Runbook 8: 90-day FDA review)

---

## Gaps and Limitations

### Not Covered in This Corpus
1. **Cyclic workflows** — None of these runbooks loop back to earlier steps (DAG assumption). Real-world: "retry entire deployment from step 1 if step 10 fails."
2. **Dynamic step generation** — No runbook generates steps at runtime based on data (e.g., "for each affected server, add a remediation step").
3. **Async wait for external event** — Runbook 8 step 8e waits for FDA clearance, but no webhook trigger specified. Schema needs external event trigger pattern.
4. **Quorum with weighted voting** — Runbook 7 has 3-of-5 board approval, but no weighted voting (e.g., CEO vote counts as 2).
5. **Conditional compensation** — Compensation always executes on failure. No example of "execute compensation only if step X completed but step Y failed."
6. **Inter-runbook communication** — Runbook 4 invokes sub-runbooks but doesn't pass messages back (e.g., sub-runbook can't ask parent for additional input).
7. **Time-boxed approval** — No example of "approve within 1 hour or automatically choose option X" (Runbook 2 step 9a rolls back on timeout, but doesn't auto-approve).
8. **Dynamic escalation policy** — Escalation chains are predefined. No example of "escalate based on runbook runtime" or "escalate to on-call for service X."

### Recommendations for Additional Runbooks (Future)
- **Cyclic retry pattern:** "Deploy → Test → If fail, rollback and retry from deploy (max 3 times)"
- **Dynamic fan-out:** "For each microservice in outage, invoke diagnostic runbook"
- **External event trigger:** "Wait for Jira ticket status == 'Resolved' (webhook callback)"
- **Weighted approval:** "Quorum: 3 votes required, CEO = 2 votes, board members = 1 vote each"

---

## Usage Instructions for Schema Translation

For John (or other schema translator):

1. **Start with simplest runbooks** — Runbook 6 (linear migration) or Runbook 9 (automated escalation) have fewer edge cases.

2. **Stress-test with Runbook 4** — Nested invocation with 15 sub-runbooks. Tests composition, input/output passing, failure propagation.

3. **Stress-test with Runbook 2** — Saga pattern with rollback compensation. Tests core v2 goal (saga pattern).

4. **Stress-test with Runbook 10** — Best-effort parallel deletion. Tests partial failure handling.

5. **Stress-test with Runbook 8** — Long-running pause for external process. Tests resume-after-weeks.

6. **For each runbook:**
   - Attempt full gert v2 schema translation (YAML)
   - Document: what works, what's ambiguous, what's impossible
   - Identify: missing schema primitives, unclear semantics, expressiveness gaps

7. **Aggregate findings:**
   - Which patterns are common across runbooks? (approval timeout, retry, parallel)
   - Which patterns need first-class schema support vs. workarounds?
   - Which patterns are fundamentally unsupported by current design?

---

## References

This corpus is informed by:
- **Academic:** IJSRET "Runbook Engineering," IBM Redbook "Operational Runbooks," DBSec 2024 "Optimal Playbook Generation"
- **Industry Tools:** AWS Systems Manager Automation, PagerDuty Runbook Automation, Rundeck, StackStorm
- **Workflow Patterns:** Temporal (saga pattern), Airflow (DAG), Argo Workflows (fan-out/in)
- **Compliance:** GDPR (right to be forgotten), FDA 21 CFR Part 11 (digital signatures), SOC2 (evidence retention)
- **Real-World:** Anonymized runbooks from SRE teams, DevOps practices, regulated industries

Each runbook reflects realistic operational practice and has been validated against actual industry patterns.
