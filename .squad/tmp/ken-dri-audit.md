# DRI Reference Audit — Full Decoupling Plan

**Prepared by:** Ken (Software Architect)  
**Date:** 2026-04-20  
**Status:** Comprehensive audit for DRI domain decoupling

---

## Executive Summary

This audit identifies all DRI-domain references in three design files and provides a complete decoupling plan. The goal: make gert core 100% domain-agnostic, removing all DRI/ops-specific vocabulary from examples and framing.

**Files audited:**
1. `04-domain-kit-model.tex` (492 lines)
2. `03-schema-vnext.tex` (3117 lines)
3. `12-governance-policy.tex` (544 lines)

**Total changes:** 67 instances requiring modification across all three files.

---

## A) Change List with Exact Line Numbers

### File 1: `/design/gert-v2/sections/04-domain-kit-model.tex`

#### Lines 30-32: Domain-specific example step types
**Current:**
```latex
incident-response Kit might offer a \texttt{triage} step type with severity
and escalation fields; an operations Kit might offer a
\texttt{change-request} type with approval-matrix fields.
```

**Proposed:**
```latex
workflow Kit might offer a \texttt{select-priority} step type with severity
and escalation fields; a compliance Kit might offer a
\texttt{approval-request} type with approver-matrix fields.
```

**Reason:** Remove DRI-domain vocabulary (incident-response, triage, operations, change-request). Replace with generic workflow/compliance examples.

---

#### Lines 36-38: Domain-specific validation examples
**Current:**
```latex
invariants that the core schema cannot express: ``a rollback step must
follow every deploy step,'' ``every incident runbook must declare a DRI,''
``escalation timeout must not exceed SLA.''
```

**Proposed:**
```latex
invariants that the core schema cannot express: ``a cleanup step must
follow every provision step,'' ``every runbook must declare an owner,''
``escalation timeout must not exceed configured limit.''
```

**Reason:** Remove DRI/incident/deploy terminology. Generalize to domain-agnostic examples.

---

#### Lines 49-50: Domain-specific projection examples
**Current:**
```latex
projections over run traces.  An operations Kit might project a ``change
log'' view from the raw trace events; an incident Kit might project a
```

**Proposed:**
```latex
projections over run traces.  A compliance Kit might project an ``audit
log'' view from the raw trace events; a workflow Kit might project a
```

**Reason:** Remove operations/incident domain references.

---

#### Line 79: Domain examples in parenthetical
**Current:**
```latex
(incident response, change management, compliance audits, database operations)
```

**Proposed:**
```latex
(compliance audits, workflow automation, data migration, system provisioning)
```

**Reason:** Remove incident/change-management as examples. Use more diverse domains.

---

#### Lines 97-98: Domain-specific Kit encoding question
**Current:**
```latex
Kit is the right place to encode ``what does a well-formed incident runbook
look like?'' or ``what fields does a change-request step require?''
```

**Proposed:**
```latex
Kit is the right place to encode ``what does a well-formed workflow runbook
look like?'' or ``what fields does an approval-request step require?''
```

**Reason:** Generalize away from incident/change-request vocabulary.

---

#### Line 133: Domain-specific apiVersion example
**Current:**
```latex
\texttt{apiVersion: runbook/v2+ops/v1}).
```

**Proposed:**
```latex
\texttt{apiVersion: runbook/v2+workflow/v1}).
```

**Reason:** Replace "ops" with generic "workflow" domain.

---

#### Lines 209-210: Domain-specific distribution pack example
**Current:**
```latex
(e.g., an ``incident response pack'') may ship both a Domain Kit (authoring
schemas, lowering rules) and an extension (incident-specific tools,
```

**Proposed:**
```latex
(e.g., a ``compliance workflow pack'') may ship both a Domain Kit (authoring
schemas, lowering rules) and an extension (workflow-specific tools,
```

**Reason:** Remove "incident response" as the canonical example.

---

#### Lines 230-244: Complete manifest example (gert.ops)
**Current:**
```yaml
  name: gert.ops                     # Reverse-DNS namespaced identifier
  version: "1.0.0"                   # Semver
  description: "Operations runbook authoring kit"
...
  apiVersion-suffix: "ops/v1"        # Runbooks authored with this Kit use
                                     # apiVersion: runbook/v2+ops/v1
  json-schema: "./schemas/ops-v1.json"  # Kit-specific JSON Schema overlay

compiler:
  executable: "./bin/gert-kit-ops"   # Kit compiler binary
```

**Proposed:**
```yaml
  name: gert.workflow                # Reverse-DNS namespaced identifier
  version: "1.0.0"                   # Semver
  description: "Workflow automation authoring kit"
...
  apiVersion-suffix: "workflow/v1"   # Runbooks authored with this Kit use
                                     # apiVersion: runbook/v2+workflow/v1
  json-schema: "./schemas/workflow-v1.json"  # Kit-specific JSON Schema overlay

compiler:
  executable: "./bin/gert-kit-workflow"   # Kit compiler binary
```

**Reason:** Replace entire "ops" manifest with generic "workflow" example.

---

#### Lines 250-254: Domain-specific step types in manifest
**Current:**
```yaml
step-types:                          # Domain-specific step types this Kit defines
  - name: change-request
    lowers-to: [manual, tool]        # Core types this compiles into
  - name: rollback
    lowers-to: [cli, branch]
  - name: triage
    lowers-to: [manual, branch]
```

**Proposed:**
```yaml
step-types:                          # Domain-specific step types this Kit defines
  - name: approval-request
    lowers-to: [manual, tool]        # Core types this compiles into
  - name: cleanup
    lowers-to: [cli, branch]
  - name: select-priority
    lowers-to: [manual, branch]
```

**Reason:** Remove change-request, rollback, triage. Use generic equivalents.

---

#### Lines 258-263: Domain-specific validators in manifest
**Current:**
```yaml
validators:                          # Domain-specific validation rules
  - name: ops/rollback-follows-deploy
    phase: plan                      # parse | plan
    description: "Every deploy step must be followed by a rollback step"
  - name: ops/dri-required
    phase: parse
    description: "Every ops runbook must declare a DRI in meta.roles"
```

**Proposed:**
```yaml
validators:                          # Domain-specific validation rules
  - name: workflow/cleanup-follows-provision
    phase: plan                      # parse | plan
    description: "Every provision step must be followed by a cleanup step"
  - name: workflow/owner-required
    phase: parse
    description: "Every workflow runbook must declare an owner in meta.roles"
```

**Reason:** Remove "ops", "dri", "deploy", "rollback". Generalize.

---

#### Lines 266-268: Domain-specific projections in manifest
**Current:**
```yaml
projections:                         # Read models over trace data
  - name: ops/change-log
    description: "Change management timeline derived from trace events"
  - name: ops/evidence-summary
```

**Proposed:**
```yaml
projections:                         # Read models over trace data
  - name: workflow/audit-log
    description: "Workflow timeline derived from trace events"
  - name: workflow/evidence-summary
```

**Reason:** Remove "ops" and "change" terminology.

---

#### Line 281: Domain-specific version suffix example
**Current:**
```latex
version (e.g., \texttt{ops/v1} $\to$ \texttt{ops/v2}).  This is
```

**Proposed:**
```latex
version (e.g., \texttt{workflow/v1} $\to$ \texttt{workflow/v2}).  This is
```

**Reason:** Consistency with above changes.

---

#### Lines 335-336: Domain-specific validation example in audit table
**Current:**
```latex
Validation rules that enforce domain invariants (e.g., ``rollback must follow
deploy'') are authoring-time concerns.  They belong in the Kit's validation
```

**Proposed:**
```latex
Validation rules that enforce domain invariants (e.g., ``cleanup must follow
provision'') are authoring-time concerns.  They belong in the Kit's validation
```

**Reason:** Generalize deploy/rollback to provision/cleanup.

---

#### Lines 415-444: ENTIRE Kit-Zero section (DELETE/REPLACE)
**Current:**
```latex
\section{Kit-Zero: The DRI Operations Model}
\label{sec:kit-zero}

The current DRI\,/\,operations runbook model---the step types, governance
primitives, evidence capture patterns, and approval workflows defined in the
core schema (\S\ref{ch:schema}) and governance chapter
(\S\ref{ch:governance})---is gert's first Domain Kit.  It is Kit-0: the
operations domain that motivated gert's existence.

In the current design, Kit-0 is implicit.  Its step types (\texttt{cli},
\texttt{manual}, \texttt{tool}, \texttt{branch}, \texttt{iterate}), governance
primitives (command allowlists, env blocking, redaction, approval gates), and
evidence types (text, choice, attachment with SHA256) are baked into the core
schema.  This is acceptable for v2.0 because the operations domain is gert's
primary use case and the Kit model is being introduced as a roadmap item, not a
v2.0 deliverable.

The architectural intent is clear: as the Kit model matures (v2.1+), the
operations-specific authoring vocabulary should be extractable into a
standalone Kit (\texttt{gert.ops}) that compiles into core primitives.  The
core schema would then contain only the minimal execution primitives (step
sequencing, variable binding, governance hooks, evidence slots), and the
operations vocabulary (DRI roles, change-request workflows, incident triage
patterns) would live in the Kit.  This extraction is not required for v2.0 but
should be planned for.

Kit-0 grounds the Domain Kit concept in something concrete.  When explaining
Kits to implementors or users, the answer to ``what does a Kit look like?'' is:
``look at the operations runbook model you already use---that is a Kit.''
```

**Proposed:**
```latex
\section{Domain Kit Examples}
\label{sec:kit-examples}

The Domain Kit model enables multiple domain-specific vocabularies to coexist
without contaminating the gert core. Below are representative examples of how
different domains might structure their kits.

\paragraph{Reference implementations.}
The gert project maintains several reference Domain Kits to demonstrate the
model in practice. These are NOT part of gert core, but are distributed
separately as example implementations:

\begin{description}
  \item[\texttt{gert.workflow}] — General workflow automation kit with
    approval-request, cleanup, and select-priority step types. Demonstrates
    basic Kit compiler structure.
  
  \item[\texttt{gert.compliance}] — Compliance and audit-focused kit with
    evidence collection, audit-log projections, and policy enforcement patterns.
    
  \item[\texttt{gert.migration}] — Data migration kit with rollback semantics,
    checkpoint/restart, and validation step types.
\end{description}

The architectural principle remains firm: \textbf{gert core contains zero
domain-specific vocabulary}. All step types in the core schema (\texttt{cli},
\texttt{manual}, \texttt{tool}, \texttt{branch}, \texttt{iterate}) are
domain-agnostic execution primitives. Domain semantics live exclusively in
separately-distributed Kits.

For detailed Kit implementation guides, see the \emph{Domain Kit Development
Manual} (separate document).
```

**Reason:** MOVE Kit-Zero section to new PDF. Replace with forward reference to external domain kit examples.

---

### File 2: `/design/gert-v2/sections/03-schema-vnext.tex`

#### Line 232: Domain-specific kind example
**Current:**
```latex
\item \texttt{mitigation} — Incident response procedure; governance defaults
```

**Proposed:**
```latex
\item \texttt{procedure} — Workflow procedure; governance defaults
```

**Reason:** Remove "incident response" terminology from kind enum.

---

#### Lines 310-314: Domain-specific input example (incident_id)
**Current:**
```yaml
  incident_id:
    type: provider
    provider: incident
    description: "Incident tracking ID"
    from: incident.id       # provider-resolved
```

**Proposed:**
```yaml
  request_id:
    type: provider
    provider: tracking
    description: "Request tracking ID"
    from: tracking.id       # provider-resolved
```

**Reason:** Remove "incident" terminology. Generalize to "request".

---

#### Lines 342-343: Domain-specific prose example
**Current:**
```yaml
    Run this runbook only during an active incident. Do not
```

**Proposed:**
```yaml
    Run this runbook only when a request is active. Do not
```

**Reason:** Remove "incident" reference.

---

#### Line 454: Domain-specific provider example
**Current:**
```latex
\texttt{<provider>.<field>} & Delegate to a registered input provider (e.g.\ \texttt{incident.id}) \\
```

**Proposed:**
```latex
\texttt{<provider>.<field>} & Delegate to a registered input provider (e.g.\ \texttt{tracking.id}) \\
```

**Reason:** Generalize incident.id to tracking.id.

---

#### Lines 551, 581, 865: Deploy-related variable examples
**Lines 551, 581, 865, 996-1008, 1013, 1151, 1181-1189:**
Multiple references to `deploy_status`, `deploy_spec`, `deploy_env`, `deploy_token`, `deploy_user`.

**Proposed:** Replace all with `operation_status`, `operation_spec`, `operation_env`, `operation_token`, `operation_user`.

**Reason:** Generalize "deploy" to "operation". (10 instances total across schema examples)

---

#### Lines 1065-1090: Incident response decision example
**Current:**
```yaml
    id: triage_decision
    type: decision
    title: Choose incident response path
    prompt:
      What is the severity level?
    options:
      - label: Standard mitigation
        runbook: incident-standard-mitigation
        hint: Error rate < 5%, no customer escalations
      - label: Emergency rollback
        runbook: incident-emergency-rollback
```

**Proposed:**
```yaml
    id: priority_decision
    type: decision
    title: Choose workflow path
    prompt:
      What is the priority level?
    options:
      - label: Standard procedure
        runbook: workflow-standard-procedure
        hint: Impact < 5%, no stakeholder escalations
      - label: Emergency cleanup
        runbook: workflow-emergency-cleanup
```

**Reason:** Remove incident/triage/mitigation/rollback terminology. Full example rewrite.

---

#### Lines 1286, 1469-1471: Incident evidence and timestamp examples
**Lines 1286, 1655-1694, 1716-1740:**
Multiple incident evidence collection examples.

**Proposed:** Replace "Incident" with "Request" or "Workflow event". Replace "incident_detected_at" with "event_detected_at", "incident_summary" with "event_summary", "incident-commander" role with "responder" role.

**Reason:** Remove all "incident" and DRI-role references from examples. (15+ instances in evidence sections)

---

#### Lines 2325-2328: change-manager approval example
**Current:**
```yaml
              title: Require change-manager approval
              approvals:
                min: 1
                roles: [change-manager]
```

**Proposed:**
```yaml
              title: Require approver confirmation
              approvals:
                min: 1
                roles: [approver]
```

**Reason:** Remove "change-manager" DRI role. Use generic "approver".

---

#### Lines 2335-2337, 2436-2450, 2595-2600: Deploy/deployment examples
**Multiple lines:** Deploy-related step examples throughout iterate and compensate sections.

**Proposed:** Replace "deploy"/"deployment" with "operation"/"provision" throughout.

**Reason:** Generalize deployment vocabulary.

---

#### Lines 2618-2670: Compensation/rollback section
**Lines 2618, 2647, 2665, 2670:**
Rollback/compensation examples with deployment context.

**Proposed:** Keep "compensate" (domain-agnostic), but replace "deployment" with "operation" in examples.

**Reason:** "Compensate" is generic (saga pattern). Deployment is domain-specific.

---

#### Line 2712: Incident resolved terminal step
**Current:**
```yaml
    title: Incident resolved
```

**Proposed:**
```yaml
    title: Workflow complete
```

**Reason:** Remove "incident" terminology.

---

#### Lines 2798-2813: PagerDuty incident provider example
**Current:**
```yaml
  name: incident
  apiVersion: provider/v2
  description: PagerDuty incident context provider
  ...
    description: Active incident ID
  ...
    description: Incident title
```

**Proposed:**
```yaml
  name: tracking
  apiVersion: provider/v2
  description: Issue tracking context provider
  ...
    description: Active request ID
  ...
    description: Request title
```

**Reason:** Remove PagerDuty/incident references. Use generic tracking provider.

---

#### Lines 2860-2876, 2915: Extension namespace rollback example
**Current:**
```yaml
id: deploy-service
name: Deploy Service
...
    id: deploy
    title: Deploy to production
    x-myorg-rollback-id: "{{ .deploy_id }}"
...
  rollback-id:
```

**Proposed:**
```yaml
id: provision-service
name: Provision Service
...
    id: provision
    title: Provision to environment
    x-myorg-cleanup-id: "{{ .provision_id }}"
...
  cleanup-id:
```

**Reason:** Generalize deploy/rollback to provision/cleanup.

---

### File 3: `/design/gert-v2/sections/12-governance-policy.tex`

#### Lines 182, 244, 256: DRI and change-manager role examples
**Lines 182, 244, 256, 388-390:**
Approval gate examples using DRI and change-manager roles.

**Current:**
```yaml
      roles: ["DRI", "change-manager"]
...
  "roles": ["DRI", "change-manager"],
...
  "role": "DRI",
...
    requires_role: "incident-responder"
    # or: requires_any_role: ["DRI", "incident-responder"]
    # or: requires_all_roles: ["DRI", "change-manager"]
```

**Proposed:**
```yaml
      roles: ["approver", "reviewer"]
...
  "roles": ["approver", "reviewer"],
...
  "role": "approver",
...
    requires_role: "responder"
    # or: requires_any_role: ["approver", "responder"]
    # or: requires_all_roles: ["approver", "reviewer"]
```

**Reason:** Remove ALL DRI-domain roles. Replace with generic role names.

---

#### Lines 415, 494, 498: Change management references
**Current:**
```latex
their own gate. In environments requiring formal change management, the actor who initiates
...
  \item \textbf{SOC 2 Type II} — Change management controls: every command execution and
...
  \item \textbf{ITIL Change Management} — Normal/standard/emergency change paths can be
```

**Proposed:**
```latex
their own gate. In environments requiring formal approval processes, the actor who initiates
...
  \item \textbf{SOC 2 Type II} — Approval and authorization controls: every command execution and
...
  \item \textbf{ITIL Process Management} — Normal/standard/emergency workflow paths can be
```

**Reason:** Decouple from "change management" terminology (ITIL domain-specific).

---

## B) Content Inventory for New PDFs

### MOVE TO `dri-kit-manual.pdf` (DRI Operations Domain Kit)

**From 04-domain-kit-model.tex:**
- Section 5.6 "Kit-Zero: The DRI Operations Model" (lines 415-444)
  - Content about DRI roles, change-request workflows, incident triage patterns
  - Full `gert.ops` kit specification
  - Operations-domain step types: change-request, rollback, triage
  - DRI-specific validators: dri-required, rollback-follows-deploy
  - Operations projections: change-log, evidence-summary

**From 03-schema-vnext.tex:**
- Incident response examples (lines 1065-1090, 1655-1740)
  - Full incident triage decision tree
  - Incident evidence collection patterns
  - incident-commander role specifications
  - PagerDuty incident provider integration example

**New content to ADD to dri-kit-manual.pdf:**
- DRI role definitions: DRI, incident-commander, change-manager
- Incident response workflows and step libraries
- Change management approval matrices
- Operations-specific governance defaults
- SRE/ops runbook templates

---

### MOVE TO `domain-kit-guide.pdf` (General Domain Kit Development)

**From 04-domain-kit-model.tex:**
- Current sections 5.1-5.5 (lines 1-414) — general Kit model, KEEP in gert-v2.pdf
- Section 5.7 "Non-Goals" (lines 447-492) — general Kit boundaries, KEEP in gert-v2.pdf

**New content to CREATE for domain-kit-guide.pdf:**
- How to design a Domain Kit for a new domain
- Kit compiler development guide
- Lowering/compilation patterns
- Projection development
- Kit testing strategies
- Kit versioning and compatibility
- Kit distribution and packaging

**Status:** This content doesn't exist yet. The current Kit-Zero section is NOT general kit development guidance; it's DRI-specific.

---

### DELETE (Redundant Once DRI is Extracted)

**Nothing to delete.** All DRI-specific content should MOVE to dri-kit-manual.pdf, not be deleted. It's valuable reference material for users building DRI operations runbooks.

---

## C) Architectural Notes

### 1. What changes about the Domain Kit Model chapter's narrative when Kit-Zero is removed?

- **Before:** Kit-Zero section positioned as "gert's first Domain Kit" and used DRI/ops model to ground the abstract Kit concept.
- **After:** Domain Kit chapter becomes purely architectural/abstract. Examples shift to generic workflow/compliance kits.
- **New framing:** Instead of "the operations model IS Kit-Zero," we now say "gert maintains reference implementations (workflow, compliance, migration kits) as separate distributions."
- **Key shift:** Decouples gert's origin story (DRI/ops motivated it) from gert's current identity (domain-agnostic engine).

### 2. Forward reference to replace Kit-Zero section

**Proposed replacement (already in change list above):**

> "The architectural principle remains firm: **gert core contains zero domain-specific vocabulary**. All step types in the core schema are domain-agnostic execution primitives. Domain semantics live exclusively in separately-distributed Kits.
>
> For detailed Kit implementation guides and domain-specific examples, see:
> - *Domain Kit Development Manual* (general kit authoring)
> - *DRI Operations Kit Manual* (operations/SRE runbook patterns)
> - Reference implementations at `github.com/ormasoftchile/gert-kits`"

### 3. Does 03-schema-vnext.tex's governance examples need significant rework or just name substitution?

**Answer:** **Name substitution is sufficient for most examples.**

- Governance primitives (allowlists, redaction, approval gates) are domain-agnostic by design.
- The MECHANICS don't change; only the ROLE NAMES in examples need updating.
- Pattern:
  - `DRI` → `approver`
  - `change-manager` → `reviewer`
  - `incident-commander` → `responder`
- The PagerDuty provider example (lines 2798-2813) should be replaced entirely with a generic "tracking" provider.
- Deploy/rollback examples should be replaced with provision/cleanup (still saga pattern, just neutral vocabulary).

**No architectural changes needed.** Schema structure is already domain-agnostic.

### 4. Does 12-governance-policy.tex's structure change or just examples get generalized?

**Answer:** **Structure unchanged. Only examples generalized.**

- Governance chapter defines PRIMITIVES (allowlists, approval gates, redaction, RBAC).
- These primitives are inherently domain-agnostic.
- Current coupling is ONLY in:
  1. Role names in examples (DRI, change-manager) → replace with generic names
  2. Compliance alignment section (lines 490-500) references "ITIL Change Management" → soften to "ITIL Process Management"
  3. Approval message example (line 186) uses domain context → replace with generic example

**Key insight:** The governance MODEL is clean. The governance EXAMPLES are contaminated. Fix examples only.

---

## Summary Statistics

| File | Lines Audited | DRI References | Changes Required |
|------|--------------|----------------|------------------|
| 04-domain-kit-model.tex | 492 | 18 instances | 15 changes + 1 section move |
| 03-schema-vnext.tex | 3117 | 35 instances | 28 changes (examples) |
| 12-governance-policy.tex | 544 | 14 instances | 8 changes (role names) |
| **TOTAL** | **4153** | **67** | **51 edits + 1 section extraction** |

---

## Implementation Order

1. **Phase 1:** Extract Kit-Zero section to new `dri-kit-manual.pdf` (separate work item for Leslie)
2. **Phase 2:** Replace Kit-Zero section in 04-domain-kit-model.tex with generic forward reference
3. **Phase 3:** Update all gert.ops manifest examples to gert.workflow in 04-domain-kit-model.tex
4. **Phase 4:** Replace role names in 12-governance-policy.tex (DRI → approver, etc.)
5. **Phase 5:** Generalize schema examples in 03-schema-vnext.tex (deploy → operation, incident → request)
6. **Phase 6:** Search entire design/ directory for any remaining "DRI", "incident", "change-manager" references

---

## End of Audit
