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
