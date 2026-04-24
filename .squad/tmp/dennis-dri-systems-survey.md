# DRI/Runbook Systems Vocabulary Survey
**Author:** Dennis (CS Researcher)  
**Date:** 2026-04-18  
**Purpose:** Inform gert dri-kit design by surveying domain vocabulary patterns in existing runbook/DRI systems

---

## Executive Summary

**Key Findings for Ken:**

1. **Domain vocabulary separation is the exception, not the norm** — Most runbook systems (PagerDuty, Blameless, FireHydrant) bundle domain-specific incident primitives directly into their core engines. Only ecosystem-focused orchestration platforms (GitHub Actions, Terraform, Ansible, Rundeck) achieve true vocabulary separation via versioned, independently-distributed packages.

2. **Separation triggers: ecosystem scale + multi-domain applicability** — Systems separate vocabularies when they aim to serve *many domains* (not just incident response) and want community/vendor contributions. Single-domain SaaS tools (PagerDuty, Blameless) don't separate because they own the entire vertical.

3. **Go modules are the natural fit for gert's domain kit model** — Terraform (providers), Ansible (collections), and Prefect (blocks) all use their language's native package manager for vocabulary distribution. For Go-based gert, domain kits as versioned Go modules follow industry precedent.

4. **DRI vocabulary is remarkably consistent** — The five primitives (notify, escalate, investigate, mitigate, postmortem) appear universally across ITIL, SRE, and all surveyed systems. These are *incident-domain-specific*, not generic orchestration primitives, and belong in a dri-kit, not gert core.

5. **Anti-pattern: bundling domain into core** — AWS SSM Automation embeds AWS-specific actions (`aws:createImage`, `aws:changeInstanceState`) into core schema. This makes the engine AWS-coupled. Gert's separation of DRI vocabulary into `gert.ops` avoids this trap.

---

## System-by-System Survey

| System | Core Primitives | Domain Vocabulary | Distribution Unit | Separation Approach |
|--------|----------------|-------------------|-------------------|---------------------|
| **PagerDuty Runbooks** | HTTP request, run script, notification | Incident-specific: escalate, page, status update | SaaS platform (no distribution) | ❌ No separation — incident primitives bundled into platform |
| **Blameless** | Detection, acknowledge, triage, assign, resolution | Incident lifecycle: escalate, communicate, postmortem | SaaS platform (no distribution) | ❌ No separation — incident vocabulary is the product |
| **FireHydrant** | Notifications, webhooks, task creation, channel creation | Runbook steps: assign roles, update status page, signal types | SaaS platform (no distribution) | ❌ No separation — runbook vocabulary is platform-specific |
| **OpsLevel** | Maturity checks (primitives for governance) | Service catalog: runbook presence, SLO checks, ownership | SaaS platform (no distribution) | ❌ No separation — service maturity is the domain |
| **AWS SSM Automation** | `aws:executeAutomation`, `aws:branch`, `aws:sleep` | AWS-specific: `aws:createImage`, `aws:changeInstanceState`, `aws:invokeLambdaFunction` | Document schemas (YAML/JSON) | ⚠️ Partial — AWS domain embedded in action namespace, but documents are portable |
| **Argo Workflows** | `steps` (sequential), `dag` (dependency graph), `container` | Generic orchestration — no domain vocabulary | Container images, Helm charts | ✅ Full separation — domain logic lives in container images, not Argo schema |
| **Temporal** | Workflow (orchestration), Activity (side effects) | No domain vocabulary — users define domain in code | Language SDKs (Go, TypeScript, Java, Python) | ✅ Full separation — domain is user code, Temporal provides execution primitives |
| **Prefect** | Flow (workflow), Task (unit of work), Block (config/infra) | Domain-specific Blocks: `S3Bucket`, `EmailCredentials`, `PostgresConnector` | Python packages (PyPI): `prefect-aws`, `prefect-gcp` | ✅ Full separation — "collections" are versioned PyPI packages |
| **Rundeck** | Node Step (per-node execution), Workflow Step (once per job) | Step plugins provide domain vocabularies (e.g., Ansible, AWS, Kubernetes) | JAR plugins (Java), uploaded to Rundeck | ✅ Full separation — plugins are independently versioned, official vs community |
| **GitHub Actions** | Step (command/action), Job (step container), Workflow (job orchestrator) | Domain actions: `actions/checkout`, `aws-actions/configure-aws-credentials` | Git repositories (versioned by tag): `owner/repo@v1` | ✅ Full separation — actions are distributed via GitHub repos, Marketplace for discovery |
| **Terraform** | Resource, Data Source, Module | Domain providers: `hashicorp/aws`, `hashicorp/azurerm`, modules for landing zones | Provider binaries (Registry), Modules (Registry or VCS) | ✅ Full separation — providers are versioned Go plugins, modules are versioned HCL |
| **Ansible** | Task, Role, Module | Domain collections: `amazon.aws`, `community.postgresql`, `cisco.nxos` | Collections (tar.gz via Ansible Galaxy): `namespace.collection@version` | ✅ Full separation — official, certified, and community tiers; semantic versioning |
| **Atlassian Opsgenie** | Alert, Incident, Responder, Escalation Policy | Response Plan: notify teams, trigger integrations (Statuspage, Slack, Jira) | SaaS platform (no distribution) | ❌ No separation — incident response vocabulary is core product |

---

## Cross-Cutting Patterns

### **When Systems Separate Domain Vocabularies**

**Separation Triggers:**
1. **Multi-domain applicability** — The orchestration engine is designed to serve *many* domains (cloud, networking, databases, CI/CD), not just one vertical.
2. **Ecosystem strategy** — The platform wants community/vendor contributions to expand its reach (GitHub Actions Marketplace, Ansible Galaxy, Terraform Registry).
3. **Language-native packaging exists** — If the platform is built in a language with strong package management (Go, Python, JavaScript), using that for domain distribution is natural.
4. **Deployment flexibility** — Users need to version, audit, and independently upgrade domain vocabularies without touching core.

**Systems That Separate:**
- **GitHub Actions** — Actions are versioned Git repos; marketplace for discovery
- **Terraform** — Providers are Go plugins (Registry); modules are HCL (Registry or VCS)
- **Ansible** — Collections are Python packages (Galaxy); official/certified/community tiers
- **Prefect** — Collections are PyPI packages (`prefect-aws`, `prefect-gcp`)
- **Rundeck** — Plugins are JAR files; uploaded to Rundeck instance
- **Argo Workflows** — Domain logic in container images, not Argo schema
- **Temporal** — Domain logic is user code (Go, TypeScript, etc.), not Temporal primitives

**Systems That Don't Separate:**
- **PagerDuty, Blameless, FireHydrant, OpsLevel, Opsgenie** — SaaS products where incident/ops vocabulary *is* the product. No need for separation because they own the entire vertical and have no multi-domain ambitions.

---

### **Distribution Units**

| Ecosystem | Distribution Unit | Versioning | Registry/Marketplace |
|-----------|------------------|------------|----------------------|
| **GitHub Actions** | Git repository (`owner/repo@tag`) | Git tags (semantic) | GitHub Marketplace |
| **Terraform** | Provider binary (Go plugin) | Semantic versioning | Terraform Registry |
| **Terraform** | Module (HCL code) | Semantic versioning | Terraform Registry, VCS |
| **Ansible** | Collection (tar.gz, Python) | Semantic versioning | Ansible Galaxy, Automation Hub |
| **Prefect** | Python package (PyPI) | Semantic versioning | PyPI |
| **Rundeck** | JAR plugin (Java) | Plugin-defined | Rundeck plugin repo (community) |
| **AWS SSM** | Document (YAML/JSON schema) | Document versions | AWS Systems Manager (per-account) |
| **Go ecosystem** | Go module (`github.com/org/pkg/v2`) | Semantic import paths (v2+) | No central registry (VCS-based) |

**Key Pattern:** When ecosystems separate domain vocabularies, they use the **native package manager** of their implementation language:
- **Go** → Go modules (import paths with version suffixes for v2+)
- **Python** → PyPI packages (semantic versioning)
- **HCL/Terraform** → Registry (providers) + VCS (modules)
- **Ansible** → Galaxy (collections)
- **JavaScript/TypeScript** → npm packages (for tools like Temporal SDK)

---

### **Official vs. Community Vocabularies**

**Three-Tier Model (Ansible):**
1. **Official** — Maintained by Ansible/Red Hat core team
2. **Certified** — Vendor-maintained (Cisco, AWS, etc.), tested and certified by Red Hat
3. **Community** — Open-source contributions, best-effort support

**Two-Tier Model (GitHub Actions):**
1. **Official** — `actions/*` organization (GitHub-maintained)
2. **Community** — Everyone else (variable quality, support, security)

**Single-Tier Model (Terraform):**
- **HashiCorp-maintained** providers (aws, azurerm, google) are de facto official
- **Community** providers are clearly labeled in Registry
- **Verified** badge for trusted community providers

**Gert Implication:** If gert dri-kit becomes a multi-kit ecosystem, adopt Ansible's three-tier model:
- **Official kits** — `gert.ops` (DRI), maintained by gert core team
- **Certified kits** — Vendor-contributed (e.g., `gert.aws`, `gert.k8s`), audited by gert maintainers
- **Community kits** — User-contributed, best-effort support

---

## DRI/Incident Vocabulary Taxonomy

### **Universal DRI Primitives (Appear in ALL Systems)**

These five primitives are the **consensus vocabulary** across ITIL, SRE, PagerDuty, Blameless, FireHydrant, and Opsgenie:

| Primitive | ITIL Term | SRE Term | Description | Domain-Specific? |
|-----------|-----------|----------|-------------|------------------|
| **Notify** | Incident Communication | Paging, Alerts, Status Updates | Communicate incident status to stakeholders (technical teams, management, end-users) | ✅ Yes — incident-domain |
| **Escalate** | Escalation (Functional, Hierarchical) | Escalation Policy | Move incident to higher expertise or authority due to severity/impact | ✅ Yes — incident-domain |
| **Investigate** | Incident Investigation and Diagnosis | Triage, Root Cause Analysis | Diagnose the problem to determine cause, scope, and remediation steps | ✅ Yes — incident-domain |
| **Mitigate** | Resolution and Recovery, Workaround | Rollback, Failover, Remediation | Implement changes/workarounds to reduce impact or restore service | ✅ Yes — incident-domain |
| **Postmortem** | Post-Incident Review, Problem Management | Blameless Postmortem | Analyze incident after resolution to capture lessons learned and improve future response | ✅ Yes — incident-domain |

**Additional Incident Primitives (Not Universal, But Common):**
- **Acknowledge** — Responder confirms ownership (PagerDuty, Opsgenie, Blameless)
- **Triage** — Assess and prioritize based on impact/urgency (Blameless, FireHydrant)
- **Assign Role** — Designate Incident Commander, Scribe, etc. (FireHydrant, Blameless)
- **Create Channel** — Auto-create Slack/Teams war room (FireHydrant, PagerDuty)
- **Update Status Page** — Communicate to external users (FireHydrant, Opsgenie, Statuspage)
- **Create Task/Ticket** — Open Jira/Linear ticket for follow-up (FireHydrant, PagerDuty)

---

### **What's Generic vs. Domain-Specific?**

**Generic Orchestration Primitives (Belong in gert core, NOT dri-kit):**
- Sequential execution (Argo `steps`, gert `sequence`)
- Parallel execution (Argo `dag`, gert fan-out/fan-in)
- Conditional branching (AWS SSM `aws:branch`, gert `branch`)
- Loops/iteration (gert `iterate`)
- HTTP requests (PagerDuty, generic step)
- Run script/command (Rundeck node step, gert `cli`)
- Wait/sleep (AWS SSM `aws:sleep`, Temporal timers)
- Variable substitution (all systems)
- Retries, timeouts (all systems)

**DRI/Incident-Specific Primitives (Belong in gert.ops dri-kit):**
- Notify (stakeholders, chat, status page)
- Escalate (on-call, management)
- Investigate (gather diagnostics, query logs/metrics)
- Mitigate (rollback, failover, patch)
- Postmortem (trigger retrospective, capture timeline)
- Acknowledge (claim ownership)
- Assign role (IC, scribe, SME)
- Triage (assess severity/impact)

**Why These Are Domain-Specific:**
- They encode *incident response semantics* — concepts like "escalation policy," "on-call rotation," "status page," "blameless postmortem"
- They don't exist in generic workflow orchestration (Argo, Temporal, Airflow have no notion of "escalation" or "status page")
- They map to ITIL/SRE best practices, which are domain knowledge, not orchestration patterns

---

## Recommendations for gert dri-kit

### **Concepts to Include in gert.ops Domain Kit**

Based on consensus vocabulary across surveyed systems:

1. **notify** — Send notifications to stakeholders (Slack, email, PagerDuty, status page)
   - *Rationale:* Universal in all DRI systems; encodes incident communication semantics
   
2. **escalate** — Escalate incident to higher authority/expertise
   - *Rationale:* Core ITIL/SRE concept; appears in PagerDuty, Opsgenie, Blameless
   
3. **investigate** — Gather diagnostics, query logs/metrics, run diagnostic commands
   - *Rationale:* Core incident response step; maps to ITIL "Investigation and Diagnosis"
   
4. **mitigate** — Take action to reduce impact or restore service (rollback, failover, patch)
   - *Rationale:* Core ITIL "Resolution and Recovery"; SRE "Remediation"
   
5. **postmortem** — Trigger retrospective process, capture incident timeline
   - *Rationale:* SRE best practice (blameless postmortems); ITIL "Problem Management"
   
6. **acknowledge** — Claim ownership/responsibility for incident
   - *Rationale:* PagerDuty/Opsgenie pattern; signals accountability
   
7. **assign-role** — Designate Incident Commander, Scribe, Subject Matter Expert
   - *Rationale:* FireHydrant/Blameless pattern; encodes DRI accountability model
   
8. **status-update** — Update external status page or stakeholder communication
   - *Rationale:* Statuspage/FireHydrant pattern; distinct from generic "notify"

---

### **Concepts to Exclude (Too Generic, Belong in gert Core)**

These appear in DRI runbooks but are **generic orchestration primitives**, not domain-specific:

1. **HTTP request** — Generic action, belongs in gert core (or a generic HTTP extension)
2. **Run script/command** — Already in gert core as `cli` step
3. **Wait/sleep** — Generic orchestration primitive (use gert core timeout/delay)
4. **Branch/conditional** — Already in gert core as `branch` step
5. **Iterate/loop** — Already in gert core as `iterate` step
6. **Variable substitution** — Core gert feature, not domain-specific
7. **Retries, timeouts** — Core orchestration features

**Why Exclude These:**
- They appear in *all* orchestration systems (Argo, Temporal, Airflow, etc.), not just DRI/incident systems
- No DRI-specific semantics — "run a script" is the same whether it's an incident runbook or a deployment runbook
- Duplicating these in dri-kit would violate the "gert core = zero domain vocabulary" principle

---

### **Separation/Distribution Patterns to Borrow**

**1. Use Go Modules for dri-kit Distribution**
- **Precedent:** Terraform providers (Go plugins), Temporal SDKs (Go modules)
- **Pattern:** `gert.ops` as `github.com/ormasoftchile/gert-domain-ops` (or `gert.io/ops`)
- **Versioning:** Semantic import paths — `gert.io/ops/v2` for breaking changes
- **Benefits:** Native Go tooling, no custom registry needed, versioning via Git tags

**2. Adopt Three-Tier Vocabulary Model (If Ecosystem Grows)**
- **Precedent:** Ansible (official/certified/community collections)
- **Pattern:**
  - **Official:** `gert.ops` (DRI), `gert.compliance` (future) — maintained by gert core team
  - **Certified:** `gert.aws`, `gert.k8s` — vendor-contributed, audited by gert maintainers
  - **Community:** `github.com/user/gert-domain-custom` — user-contributed, best-effort
- **Benefits:** Clear trust/support tiers, encourages ecosystem contributions

**3. Vocabulary as Thin Compilation Layer (Not Runtime Extension)**
- **Precedent:** Terraform modules compile to provider resources; GitHub composite actions compile to step sequences
- **Pattern:** `gert.ops` dri-kit compiles high-level DRI steps (`notify`, `escalate`) to gert core primitives (`cli`, `invoke`, `manual`)
- **Benefits:** No runtime coupling, no plugin sandboxing needed, pure schema transformation
- **Already Decided:** This is the gert Domain Kit Model (see `.squad/decisions.md`)

**4. Document-Based Distribution (Not Binary Plugins)**
- **Precedent:** AWS SSM Automation documents (YAML/JSON), Ansible collections (Python + YAML)
- **Pattern:** `gert.ops` distributes JSON Schema vocabulary + compiler (Go code)
- **Benefits:** Human-readable, auditable, no binary trust issues, works with existing gert validator

**5. Explicit Schema Versioning in Runbooks**
- **Precedent:** Terraform `required_providers` block, GitHub Actions `uses: action@version`
- **Pattern:** Runbook YAML declares kit dependencies:
  ```yaml
  kits:
    - name: gert.ops
      version: "^1.0.0"
  ```
- **Benefits:** Runbook is self-documenting, prevents version skew, enables reproducibility

---

### **Anti-Patterns to Avoid**

**1. Embedding Domain Vocabulary in Core Schema (AWS SSM Anti-Pattern)**
- **Problem:** AWS SSM embeds AWS-specific actions (`aws:createImage`, `aws:changeInstanceState`) in core automation schema
- **Consequence:** Engine is AWS-coupled; can't be used for non-AWS orchestration
- **Gert's Solution:** `gert.ops` is a *separate package*, not part of gert core schema

**2. No Versioning for Domain Vocabularies**
- **Problem:** If vocabulary changes break runbooks, users can't pin to stable version
- **Precedent:** Early Ansible (pre-collections) had this problem — roles were unversioned
- **Gert's Solution:** Domain kits are versioned Go modules; runbooks declare version constraints

**3. Binary Plugins Without Sandboxing**
- **Problem:** Rundeck JAR plugins run in JVM; no isolation, security risk
- **Precedent:** Jenkins plugins have had numerous security issues due to lack of sandboxing
- **Gert's Solution:** Domain kits are *compilers*, not runtime plugins — compile to gert core, no sandboxing needed

**4. SaaS-Only Vocabulary (No Portability)**
- **Problem:** PagerDuty/FireHydrant runbooks only work in their platforms
- **Consequence:** Vendor lock-in, can't version-control or test locally
- **Gert's Solution:** Runbooks are portable YAML files; dri-kit is open-source Go module

**5. Domain Vocabulary Without Standard Taxonomy**
- **Problem:** If every DRI system invents its own terms, no shared understanding
- **Precedent:** ITIL and SRE books exist precisely to standardize incident vocabulary
- **Gert's Solution:** `gert.ops` vocabulary aligns with ITIL/SRE consensus terms (notify, escalate, investigate, mitigate, postmortem)

---

## Citations and Sources

### **Systems Surveyed (Primary Sources)**

1. **PagerDuty Runbooks**
   - https://support.pagerduty.com/docs/runbooks
   - https://support.pagerduty.com/docs/response-plays

2. **Blameless Incident Management**
   - Web search: incident management primitives, blameless postmortem workflow

3. **FireHydrant**
   - https://docs.firehydrant.com/docs/runbooks
   - https://docs.firehydrant.com/docs/integrations
   - https://docs.firehydrant.com/docs/incidents

4. **OpsLevel**
   - Web search: service catalog, maturity checks, runbook primitives

5. **AWS Systems Manager Automation**
   - https://docs.aws.amazon.com/systems-manager/latest/userguide/automation-actions.html
   - https://docs.aws.amazon.com/systems-manager/latest/userguide/automation-document-structure-syntax.html

6. **Argo Workflows**
   - Web search: template types, DAG vs steps, workflow primitives

7. **Temporal**
   - https://docs.temporal.io/concepts/workflows
   - https://docs.temporal.io/dev-guide/workers/versioning

8. **Prefect**
   - Web search: flow/task/block architecture, Prefect collections

9. **Rundeck**
   - Web search: node steps vs workflow steps, step plugins

10. **GitHub Actions**
    - https://docs.github.com/en/actions/creating-actions/about-actions#versioning-your-action
    - https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions

11. **Terraform**
    - https://registry.terraform.io/browse/providers
    - https://registry.terraform.io/modules
    - Web search: provider versioning, module distribution

12. **Ansible**
    - https://galaxy.ansible.com/
    - Web search: collections, official vs certified vs community

13. **Atlassian Opsgenie / Statuspage**
    - https://docs.opsgenie.com/docs/response-plans-overview
    - https://docs.opsgenie.com/docs/statuspage-integration
    - https://developer.statuspage.io/

### **Taxonomies and Standards**

14. **ITIL (IT Infrastructure Library)**
    - Web search: incident management process, escalation, post-incident review

15. **SRE (Site Reliability Engineering)**
    - Web search: incident response, blameless postmortems, ITIL vs SRE taxonomy

16. **Go Modules and Ecosystem Patterns**
    - https://go.dev/ref/mod
    - https://go.dev/blog/v2-go-modules

---

## Appendix: Vocabulary Separation Decision Matrix

Use this matrix to decide if a primitive belongs in **gert core** vs. **domain kit**:

| Question | If YES → | If NO → |
|----------|----------|---------|
| Does this primitive appear in non-DRI orchestration systems (Argo, Airflow, Temporal)? | **gert core** (generic) | Continue to next question |
| Does this primitive encode DRI/incident-specific semantics (escalation policy, on-call, status page, blameless postmortem)? | **dri-kit** (domain-specific) | Continue to next question |
| Does this primitive map to an ITIL or SRE best practice term? | **dri-kit** (domain-specific) | **gert core** (generic) |

**Examples:**
- **Sequential execution** → Appears in Argo, Airflow, Temporal → **gert core**
- **Escalate** → Encodes DRI semantics (escalation policy, on-call) → **dri-kit**
- **HTTP request** → Appears in all orchestration systems, no DRI semantics → **gert core**
- **Postmortem** → Maps to ITIL "Problem Management" and SRE "Blameless Postmortem" → **dri-kit**
- **Branch/conditional** → Appears in all orchestration systems → **gert core**
- **Notify** → Encodes incident communication semantics (stakeholders, status page) → **dri-kit**

---

**End of Survey**
