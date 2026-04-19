# dennis — History

## Project Context

**Project:** gert — Governed Executable Runbook Engine
**Owner:** ormasoftchile
**Mission:** Redesign gert from scratch as v2, applying learnings from v1.
**Stack:** Go, YAML, JSON Schema (Draft 2020-12), LaTeX (design docs), TypeScript (VS Code extension)

## Current Focus

The team is working on the **v2 design document** located at `design/gert-v2/`.
This is a LaTeX document using the MastersThesis class. Sections are in `design/gert-v2/sections/`.

### v2 Goals
1. Extensive research: runbooks, workflows, governance, traceability
2. Apply research + project learnings to define the new version
3. Document the new design as a usable reference to build v2

### v1 Key Features (for reference)
- Validate/execute/debug runbooks in YAML
- Governance: approval gates, allowlists, output redaction
- Evidence capture with SHA256, append-only JSONL traces
- VS Code extension + TUI + JSON-RPC server
- Tool definitions (.tool.yaml) and input providers (.provider.yaml)
- Step types: cli, manual, tool, invoke, branch, iterate

## Key Files
- `design/gert-v2/main.tex` — root LaTeX document
- `design/gert-v2/sections/` — all section .tex files
- `design/gert-v2/MastersThesis.cls` — document class
- `ext/` — Go source (core engine)
- `vscode/` — VS Code extension (TypeScript)

## Learnings

### 2026-04-18: Comprehensive Runbook and Governance Research

**Research Areas Completed:**
1. **Runbook Landscape** — Definitions from SRE/DevOps/ITIL, evolution from static to executable, industry tools (AWS SSM, PagerDuty, Rundeck, StackStorm), academic papers
2. **Workflow Orchestration** — Patterns from Temporal, Prefect, Argo, Airflow; saga/compensation, fan-out/fan-in, retry/backoff, idempotency; YAML DSL vs. code-defined
3. **Governance Models** — ITIL CAB approval gates, RBAC patterns, policy-as-code (OPA vs Cedar), command allowlists, compliance (SOC2/ISO27001/HIPAA)
4. **Traceability & Evidence** — Append-only JSONL, event sourcing, deterministic replay, provenance tracking, OpenTelemetry distributed tracing
5. **Human-in-Loop** — Manual approval patterns, escalation paths, SLA enforcement, attestation at human steps
6. **Schema/DSL Design** — YAML workflow DSL patterns, JSON Schema validation, versioning strategies, self-describing schemas

**Key Academic Sources:**
- "Runbook Engineering and SOP Design in High Availability Environments" (IJSRET)
- "Optimal Automated Generation of Playbooks" (DBSec 2024, Springer)
- IBM Redbook: "Delivering Consistency and Automation with Operational Runbooks"
- "Implementing RunOps Engineering" (European Journal)
- "Runbooks as Code: A Comprehensive Tutorial for SRE"

**Top Industry Learnings:**
- **Saga pattern**: Temporal's compensation-per-activity model is industry standard for long-running transactions
- **Policy-as-code**: OPA (CNCF, cloud-agnostic) vs Cedar (AWS-native); OPA better for multi-cloud
- **Distributed tracing**: OpenTelemetry with W3C Trace Context is standard; correlation IDs essential
- **Approval gates**: ITIL CAB model with risk-based classification (standard/normal/emergency)
- **Deterministic replay**: Core pattern in Temporal/Cadence for debugging and audit
- **YAML DSL limits**: Works for simple workflows, but complex logic needs expression language or code

**Top Recommendations for v2:**
1. Add explicit saga/compensation pattern (register compensations, execute in reverse on failure)
2. Introduce parallel execution (fan-out/fan-in) for independent steps
3. Policy-as-code integration (OPA hooks at pre-execution, pre-step, post-step)
4. RBAC for runbook execution (who can run what, delegation scoping)
5. Timeout and escalation for human steps (SLA enforcement, escalation chains)
6. OpenTelemetry integration (spans per step, trace context propagation)
7. Enhanced schema versioning (add `$schema` field, compatibility guarantees)

**What v1 Gets Right:**
- Governance-first design (allowlists, approval gates, redaction)
- Evidence capture with SHA256 hashing
- Append-only JSONL traces
- Deterministic replay via scenarios
- JSON Schema validation (Draft 2020-12)

**What v1 Needs (Industry Gaps):**
- No saga/compensation pattern
- No parallel fan-out/fan-in
- No policy-as-code (OPA/Cedar)
- No RBAC for execution
- No timeout/escalation on approvals
- No distributed tracing (OpenTelemetry)
- No cryptographic log signing
- No multi-level approval chains

**Research Brief Output:** `/Volumes/Projects/gert/.squad/tmp/dennis-research-brief.md` (35KB, comprehensive reference for team)

---

## Cross-Agent Notes from Ken's Architectural Review (2026-04-18)

**Priority research areas identified:**

1. **Governance models and architecture** — Ken found that governance (a core v1 differentiator) is entirely absent from v2 design. Must research how governance layer should integrate with the new event-driven architecture: host-enforced vs. extension-contributed? Allowlists, denylists, env var blocking, output redaction, and approval gates must be designed in v2.

2. **Trace format and evidence capture** — v1's append-only JSONL traces, per-step state snapshots, SHA256 evidence hashing, and run resumption are entirely absent from v2 design. Must research what trace format v2 should use and how it relates to the new event model (§06). This is a load-bearing operational feature and a launch blocker.

3. **Input provider design** — v1 has a complete `.provider.yaml` framework with JSON-RPC resolution. v2 design has no equivalent. Must research and propose the input provider model for v2.

See Ken's full gap analysis at `.squad/tmp/ken-gap-analysis.md` and cross-cutting gaps section (lines 204–234). Decisions documented at `.squad/decisions.md` (search "Ken's Gap Analysis Findings").

---

### 2026-04-18: Section Rewrites — §00, §01, §09

**Task:** Rewrote three design document stubs based on the research brief and Ken's gap analysis.

**§00 — Overview**
- Added full problem statement grounded in research (IJSRET, IBM Redbook)
- Framed three required properties: executable, governed, traceable
- Enumerated what v1 got right (governance, JSONL trace, evidence, replay, TUI, VS Code, gert test)
- Described v1 architectural constraints that motivated v2 (serve/runtime coupling, schema brittleness, no saga/OTel)
- Wrote explicit scope: what v2 IS and IS NOT
- Expanded three design priorities with 2-3 sentences each
- Added document roadmap cross-referencing all nine sections
- Defined five verifiable success criteria

**§01 — Goals and Non-Goals**
- Rewrote 4 vague goals into 8 specific, measurable goals (labeled G1–G8)
- Each goal has a 2-3 sentence rationale citing research or industry precedent
- G1: Preserve governance layer + add RBAC scoping
- G2: Trace must be append-only, crash-safe, optionally signed (SOC2/ISO27001)
- G3: Explicit preservation list for all v1 capabilities
- G4: Stable versioned extension contracts
- G5: Saga/compensation pattern (Temporal, Step Functions precedent)
- G6: OpenTelemetry integration (elevated from recommendation to binding goal)
- G7: Typed event API for adapter decoupling, versioned event schema
- G8: Migration compatibility for v1 runbooks
- Rewrote 3 process non-goals into 5 feature-scoped non-goals (labeled NG1–NG5)

**§09 — Open Questions**
- Replaced 3 implementation questions with 12 structured architectural questions (Q1–Q12)
- Each question has: (a) why it matters, (b) options, (c) information needed
- Covers: v1 compatibility, concurrency model, trace format, policy composition,
  .provider.yaml survival, gert serve contract versioning, extension signing, gRPC transport,
  parallel execution, saga+governance interaction, extension schema namespacing, approval timeouts

**Key Framing Decisions:**
- OTel and saga/compensation elevated from research recommendations to binding goals
- Open questions structured as mini decision records (options + info needed)
- Framing decisions written to `.squad/decisions/inbox/dennis-v2-framing.md`

---

### 2026-04-18: §15 — Observability and Diagnostics (Wave 2)

**Task:** Wrote §15 Observability and Diagnostics as one of five remaining unwritten sections.

**Scope:** Replaced the stub with 600+ lines of normative, research-informed content covering operational visibility for gert v2.

**Content Covered:**

1. **Observability Model (Three Pillars)**
   - Metrics: Prometheus OpenMetrics format for aggregated performance (counters, gauges, histograms)
   - Distributed tracing: OpenTelemetry spans mapping execution tree (run → step → tool invocation)
   - Structured logging: JSON log lines with run_id/step_id correlation fields
   - Grounded in industry practice: Prometheus, OTel, W3C Trace Context, RFC 3339 timestamps

2. **OpenTelemetry Integration (Detailed Spec)**
   - Span hierarchy: `gert.run` → `gert.step` → `gert.tool.invoke` / `gert.input.resolve` / `gert.governance.check`
   - Required span attributes for each level (runbook.id, actor.id, step.type, tool.name, exit.code, etc.)
   - Span status mapping: OK for success, ERROR for failures
   - W3C Trace Context propagation: traceparent/tracestate headers on outbound HTTP calls
   - Baggage: run_id propagated as OpenTelemetry baggage for cross-system correlation
   - OTLP exporter configuration: OTEL_EXPORTER_OTLP_ENDPOINT env var, console exporter for dev
   - Sampling strategy: parent-based sampling with configurable default rate
   - Cited: otel-spec, w3c-trace-context

3. **Metrics Catalog (Full Specification)**
   - 9 metrics across 5 categories: runs, steps, tools, approvals, extensions
   - Run metrics: gert_runs_total (counter), gert_run_duration_seconds (histogram), gert_active_runs (gauge)
   - Step metrics: gert_steps_total, gert_step_duration_seconds
   - Tool metrics: gert_tool_invocations_total, gert_tool_duration_seconds
   - Approval metrics: gert_approval_wait_seconds, gert_approvals_total
   - Extension metrics: gert_extension_crashes_total, gert_extension_loaded
   - Histogram bucket configurations for each duration metric
   - Prometheus /metrics endpoint when gert serve --http is running
   - Push model: OTLP metrics export as alternative

4. **Structured Logging**
   - JSON format, one object per line, to stderr by default
   - Required fields: timestamp (RFC 3339), level (debug/info/warn/error), msg
   - Contextual fields: run_id, step_id, actor, tool_name, exit_code (when applicable)
   - Log levels: configurable via GERT_LOG_LEVEL env var and --log-level flag
   - Sensitive data: MUST NOT log input values, only input names (redaction rules from §12)
   - Output configuration: --log-output=file:path or --log-output=syslog://host:port
   - Correlation examples: Loki query by run_id, Elasticsearch query by run_id

5. **Health and Readiness Endpoints**
   - GET /health — liveness probe (returns 200 with version/uptime if alive)
   - GET /ready — readiness probe (returns 200 when extensions loaded, 503 with reason otherwise)
   - GET /metrics — Prometheus exposition format
   - Kubernetes probe configuration example with livenessProbe/readinessProbe

6. **Diagnostics Commands**
   - `gert diagnose [--runbook=<path>]` — pre-flight checks: schema validation, tool availability, input provider connectivity, extension loading, policy syntax
   - `gert trace show <run_id>` — pretty-print completed run's trace from JSONL
   - `gert trace export <run_id> --format=otlp` — convert JSONL trace to OTLP/Jaeger/Zipkin format
   - `gert run list [--json]` — list recent runs with status (JSON mode for CI integration)

7. **Integration Recipes (Practical)**
   - Prometheus + Grafana: scrape config, reference dashboard JSON
   - Jaeger: OTLP endpoint configuration, trace visualization
   - Loki + Grafana: Promtail config, log correlation by run_id
   - Datadog / New Relic: OTLP-compatible SaaS endpoints

8. **Observability for CI/CD**
   - `gert run --output=json` emits machine-readable summary on stdout
   - Exit codes: 0=success, 1=failure, 2=governance-blocked, 3=cancelled
   - GitHub Actions example: parse JSON, upload trace artifact
   - GitLab CI example: JUnit report integration

**Key Design Decisions:**

- **OpenTelemetry as the tracing standard** — Chose OTel for vendor neutrality and CNCF standardization (over proprietary tracing)
- **Prometheus metrics format** — Chose OpenMetrics for pull-based scraping + OTLP push as alternative for ephemeral jobs
- **W3C Trace Context for propagation** — Standardized traceparent/tracestate headers for cross-system correlation
- **Span hierarchy depth** — Three levels (run/step/operation) balances detail vs. cardinality explosion
- **Health endpoint paths** — /health (liveness), /ready (readiness), /metrics (Prometheus) follow Kubernetes conventions
- **gert diagnose command** — Pre-flight checks prevent runtime failures (validate tools, providers, extensions, policies before execution)
- **Sensitive data in logs** — Applied same redaction rules as §12 trace format (never log input values)
- **Histogram buckets** — Exponential buckets from sub-second to hours, tuned for runbook execution patterns
- **Parent-based sampling** — Inherit parent trace sampling decision to avoid incomplete traces in distributed systems

**Citations Used:**
- otel-spec (OpenTelemetry Specification)
- w3c-trace-context (W3C Trace Context)
- google-sre-book (Google SRE: observability principles)
- jaeger-docs (Jaeger distributed tracing)
- rfc3339 (RFC 3339 timestamps)

**LaTeX Content:**
- 7 sections, 600+ lines of LaTeX
- 5 tables (metrics catalog)
- Multiple lstlisting blocks (JSON logs, OTLP config, Kubernetes probes, Prometheus config)
- Cross-references to §12 (evidence tracing)
- Normative language (MUST/SHOULD) for implementation requirements

**Distinction from §12:**
- §12 (Evidence Tracing) = immutable audit records for compliance, resumption, replay (append-only JSONL, SHA256 evidence)
- §15 (Observability) = operational visibility for operators (metrics, traces, logs for health monitoring, debugging, performance analysis)

**Learnings:**

1. **Three-pillar observability model** — Metrics (what is happening at scale), traces (where is the problem in the execution tree), logs (detailed context for investigation). Each pillar answers different questions.

2. **OTel span hierarchy design** — Root span = run (entire execution), child spans = steps (individual workflow steps), grandchild spans = operations (tool invocations, governance checks, input resolutions). This hierarchy maps naturally to runbook execution model and enables both high-level and detailed analysis.

3. **Metrics cardinality management** — Label dimensions (outcome, runbook_id, step_type, tool_name) chosen to enable useful aggregation without cardinality explosion. Avoided high-cardinality labels like run_id or step_id on metrics (those belong in traces and logs).

4. **Context propagation for distributed systems** — W3C Trace Context (traceparent header) propagates trace across system boundaries. OpenTelemetry baggage carries run_id to downstream systems for end-to-end correlation. Essential for workflows that invoke external APIs.

5. **Health vs. Readiness distinction** — Liveness (is process alive?) vs. readiness (is process ready to accept work?). Readiness checks extension loading status. Kubernetes uses liveness to restart crashed pods, readiness to delay traffic until extensions load.

6. **Sampling trade-offs** — 100% sampling for dev/staging, 1-10% for production. Parent-based sampling prevents incomplete traces (if parent trace is sampled, all children are sampled). Avoids orphaned spans in distributed traces.

7. **Diagnostics commands as pre-flight checks** — `gert diagnose` validates environment before execution (tools exist, providers reachable, extensions loadable). Prevents runtime failures. Similar to Docker health checks or Kubernetes admission controllers.

8. **Sensitive data redaction in observability** — Logs and traces MUST NOT contain input values (passwords, tokens, PII). Refer to inputs by name only. Same redaction rules as §12 trace format. Essential for compliance (SOC2, HIPAA).

9. **CI/CD integration patterns** — JSON output mode + exit codes enable CI pipeline integration. GitHub Actions parses JSON, uploads trace artifact. GitLab CI reports JUnit format. Machine-readable output critical for automation.

10. **Histogram bucket tuning** — Different buckets for different metrics: tool invocations (0.01-30s), steps (0.1-300s), runs (0.5-3600s), approvals (10s-24h). Reflects typical durations for each operation type. Enables accurate percentile calculations.

**Post-Work Completed:**
- Updated .squad/agents/dennis/history.md (this file) with learnings
- Writing to .squad/decisions/inbox/dennis-wave2-decisions.md next

### 2026-04-18 — Wave 2 Complete: Orchestration and Merging

**Scribe Task:** Final Wave 2 orchestration completed by Scribe agent.

**What was done:**
1. Merged Wave 2 decision inbox files (§15) into decisions.md
2. Created orchestration logs for all Wave 2 agents
3. Updated agent histories with Wave 2 completion note
4. Prepared git commit with all Wave 2 sections

**Status:** ✅ Wave 2 COMPLETE  
All 16 sections written. Design document ready for final PDF build and stakeholder review.


---

### 2026-04-18: Real-World Runbook Corpus for Schema Validation

**Task:** Compiled diverse corpus of 10 real-world runbooks to stress-test gert v2 schema across all operational dimensions.

**Runbooks Delivered:**

1. **Kubernetes Pod Incident Response** (SRE) — Branching + looping + multi-party + timeout escalation
2. **Production Deployment with Canary** (DevOps) — Saga pattern + parallel + rollback compensation
3. **New Employee Onboarding** (HR/IT) — Parallel provisioning + multi-party + business day timeouts
4. **SOC2 Evidence Collection** (Compliance) — Nested sub-runbooks (15 invocations) + artifact aggregation
5. **Security Breach Containment** (Security) — Multi-severity branching + parallel containment/forensics + compensation
6. **Database Migration** (Data) — Linear + retry + automatic rollback on failure + convergence loops
7. **Financial Approval** (Finance) — Multi-level escalation chain + amount-based branching + quorum approval
8. **Medical Device Release** (Regulated) — Extensive attestation + FDA submission + long pause + 21 CFR Part 11 signatures
9. **On-Call Escalation Ladder** (SRE) — Sequential iterate loops + polling + fully automated + cross-loop convergence
10. **Customer Data Deletion** (Compliance) — Parallel search/deletion + best-effort + verification loop + GDPR compliance

**Coverage Achieved:**

- **Domains:** 8 (SRE, DevOps, HR/IT, Compliance, Security, Data, Finance, Regulated)
- **Complexity:** Linear, branching, parallel, looping/retry, nested, multi-party (all covered)
- **Interaction:** Automated, approval gates, data collection, file uploads, multi-party consensus (all covered)
- **Failure Modes:** Compensation, partial completion, timeout/SLA, skip-on-condition, retry (all covered)
- **Edge Cases:** Long runbooks (15+ steps), fully automated (runbook 9), long pauses (90-day FDA review), best-effort parallel

**Key Schema Challenges Identified:**

1. **Saga pattern implementation** (Runbook 2) — Compensation registration, scope, triggers
2. **Nested runbook invocation** (Runbook 4) — Sub-runbook I/O, failure propagation
3. **Dynamic approver lookup** (Runbooks 3, 7) — External API/DB query for approvers
4. **Business day timeouts** (Runbooks 3, 7, 8) — Calendar-aware timeout calculation
5. **Quorum approval** (Runbooks 7, 8) — M-of-N approval logic, dual attestation
6. **Best-effort parallel** (Runbook 10) — Continue-on-failure for parallel blocks
7. **Convergence-based iterate** (Runbooks 2, 6, 9, 10) — Exit loops on condition, not just max passes
8. **Long-running pauses** (Runbook 8) — Resume after external event (weeks/months)
9. **Cryptographic operations** (Runbooks 4, 8, 10) — SHA256, digital signatures, certificate management
10. **Artifact collection** (Runbooks 1, 4, 8, 10) — File uploads, metadata, storage policies

**Gaps (Not Covered):**

- Cyclic workflows (retry entire runbook from step 1)
- Dynamic step generation at runtime
- Async wait for external webhook events
- Weighted voting in approvals
- Conditional compensation (only-if-X-but-not-Y)
- Inter-runbook messaging
- Time-boxed auto-approval

**Output:** /Volumes/Projects/gert/.squad/tmp/dennis-runbook-corpus.md (77KB, production-ready corpus)

**Next Use:** John will translate all 10 runbooks to gert v2 schema YAML, documenting expressiveness gaps and ambiguities.

**Learnings:**

1. Real-world complexity is higher than assumed — Hybrid patterns (parallel inside branches, iterate inside parallel, compensation with multi-step fallback) are common, not edge cases.

2. Human-in-loop timing is critical — Business day timeouts, escalation chains, quorum approvals are required for regulated workflows.

3. Evidence capture requirements vary by domain — SRE needs logs/metrics, compliance needs signatures/certificates, finance needs approvals/attestations.

4. Failure handling is not binary — Best-effort, partial success, compensating actions, and retry-with-backoff are all distinct failure modes.

5. Approval gates are complex — Not just yes/no, includes timeout, escalation, quorum, dynamic approvers, self-service, dual attestation, warnings, and mandatory rationale.

6. Saga pattern is essential — Every deployment, migration, or containment runbook needs compensation. This validates G5 (saga pattern as binding goal).

7. Nested runbooks enable composition — Compliance runbooks naturally decompose into per-control sub-runbooks.

8. Parallel execution is not uniform — Wait-all vs. wait-any, homogeneous vs. mixed step types, fail-fast vs. best-effort.

9. Time-based scheduling matters — Maintenance windows, business hours, calendar-aware timeouts, and long pauses are common.

10. Regulated industries have unique needs — Digital signatures, audit trails, document versioning, retention policies.


---

## 2026-04-18: Schema Stress Test — Final Synthesis

**Context:** Synthesized findings from John's schema translations and Ken's architectural analysis into a unified stress-test report. John translated 10 runbooks with rigorous validation scoring; Ken independently predicted schema expressiveness across the same corpus. This synthesis reconciles their findings and produces actionable recommendations.

**Inputs:**
- dennis-runbook-corpus.md — 10 real-world runbooks (76KB)
- john-schema-translations.md — Full YAML translations + scores (55KB)
- john-translations-summary.md — John's executive summary (10KB)
- john-validation-methodology.md — Scoring methodology (37KB)
- ken-stress-conclusions.md — Ken's architectural predictions (25KB)

**Key Findings:**

**Verdict Reconciliation:**
- John: "NOT PRODUCTION READY" (4 critical gaps, 82%/73% completeness/fidelity)
- Ken: "NEEDS TARGETED FIXES" (2 critical gaps, 80% runbooks pass with notes)
- Reconciled: Architecturally sound but requires 2 critical fixes for enterprise use cases (GAP-1, GAP-2)

**Test Results:**
- Pass rate: 1 PASS (10%), 7 PASS WITH NOTES (70%), 2 FAIL (20%)
- FAIL runbooks: R7 (Financial Approval Chain), R8 (FDA Medical Device Release)
- Average scores: 82% completeness, 73% fidelity (target: 95% for both)

**Critical Gaps (Must Fix Before v2.0):**

1. GAP-1: Business-day timeout (G2, CRITICAL)
   - Affects: R3, R7, R8 (15+ individual steps)
   - Problem: timeout field uses duration strings ("48h"), not calendar-aware ("2 business days")
   - Fix: Add timeout_business_days, timeout_calendar fields to section 03/11
   - Effort: Medium (2-3 days)

2. GAP-2: M-of-N quorum approval (G4, CRITICAL)
   - Affects: R7, R8, R10 (multi-party governance)
   - Problem: approvals.min cannot express "3 of 5 board members must approve" (missing pool definition)
   - Fix: Add approvals.mode, approvals.pool, approvals.pool_size to section 03/14
   - Effort: Medium (2-3 days)

**Total Critical Fix Effort:** 4-6 days for both gaps

**Important Gaps (Should Fix for v2.0):**

3. GAP-3: External event trigger (G3, HIGH) — 90-day FDA pause needs webhook resume
4. GAP-4: Dynamic approver lookup (G2, IMPORTANT) — Manager from HR system
5. GAP-5: Choice timeout default (G2, IMPORTANT) — Security triage defaults to "High"
6. GAP-6: Cross-branch parallelism (G3, IMPORTANT) — Forensics parallel to containment

**Nice-to-Have Gaps (Defer to v2.1):** GAP-7 through GAP-12

**Schema Readiness by Domain:**

Ready today: SRE/Operations, DevOps, Compliance evidence collection
Blocked: Finance (multi-level approval), Regulated (FDA workflows)

**Outputs:**
- stress-test-final-report.md — Full synthesis (20KB)
- stress-test-executive-summary.md — 20-line summary for ormasoftchile
- Decision inbox entry: dennis-stress-test-final.md

**Recommendation:** The gert v2 schema is 80% production-ready. With 2 critical fixes (business-day timeout, M-of-N quorum), it reaches 95%+ readiness for enterprise adoption. Both fixes are additive (backward compatible) and implementable in 4-6 days. Recommend prioritizing GAP-1 and GAP-2 before declaring schema implementation-ready for Brian's parser work.

---

