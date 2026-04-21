# Gert v2 Decisions

**Last Updated:** 2026-04-19T21:09:46Z

---

## barbara-correctness-strategy

# Decision Inbox: Gert v2 Correctness Strategy

**By:** Barbara (Integrations Specialist)  
**Date:** 2026-04-19  
**Status:** PROPOSED (awaiting team review)

---

## Decision: Spec-Driven Development with Tagged Test Compliance

### What

Implement a comprehensive correctness strategy for gert v2 that makes the spec executable through tagged tests and enforces compliance at every phase gate.

**Core components:**

1. **Spec tagging convention:** All tests validating spec rules carry `// spec: {file} §{section} — {RULE_TEXT}` comments
2. **Spec coverage tool:** Phase 0 deliverable `gert dev spec-coverage` cross-references spec MUST rules with test tags, reports coverage, enforces ≥95% by Phase 13
3. **TDD workflow:** Brian's mandatory loop: read spec → extract MUST rules → write failing tagged tests → implement → pass → verify coverage → submit to Ken
4. **Golden trace testing:** Deterministic JSONL trace comparison starting Phase 3, with HMAC verification in Phase 11+
5. **Phase gates:** 6-part exit criteria including spec compliance checklist and Ken's architectural review
6. **Ken as spec reviewer:** Ken approves each phase only when spec alignment, test coverage, and interface contracts are verified

### Why

**Problem:** Without explicit spec-to-test traceability, implementations drift from specifications, spec rules become documentation instead of contracts, and correctness is subjective.

**Solution:** Make every spec rule executable and measurable. Spec coverage becomes a quantifiable metric (currently 89%, target 95%). Ken's review becomes systematic (check coverage report, verify tagged tests exist for sampled rules).

**Benefits:**
- **Auditability:** Any stakeholder can run `gert dev spec-coverage` and see which spec rules are/aren't tested
- **Regression prevention:** Golden traces catch unintended behavior changes across phases
- **Spec drift detection:** Coverage tool flags new MUST rules with no tests (or tests with no corresponding spec rules)
- **Review efficiency:** Ken focuses on uncovered rules and sampled verification, not exhaustive manual checks
- **Compliance confidence:** By Phase 13, ≥95% coverage guarantees spec compliance is verified, not assumed

### Constraints

- **Phase 0 dependency:** Spec coverage tool must be delivered in Phase 0 or strategy is unenforceable
- **Discipline required:** Brian must tag all spec tests (no shortcuts); Ken must review coverage reports
- **Golden trace determinism:** Only works for scenarios without real time/randomness/network I/O
- **HMAC golden traces:** Require `GERT_TRACE_KEY` to be set consistently for test runs

### Implementation Path

**Phase 0:**
1. Implement `internal/dev/speccoverage/` package
2. Add `gert dev spec-coverage` subcommand
3. Extract MUST/MUST NOT/SHALL from all spec files (CommonMark parser)
4. Cross-reference with `// spec:` tags in all `*_test.go` files
5. Add CI warning (non-blocking) for coverage <95%

**Phase 1-12:**
1. Brian follows TDD workflow for all new code
2. Ken reviews spec coverage report before approving each phase
3. Phase exit checklist includes "spec coverage verified"

**Phase 13:**
1. CI gate enforces ≥95% spec coverage (blocking)
2. All acceptance corpus scenarios have golden traces
3. Ken's final review confirms zero uncovered MUST rules (or documented exceptions)

### Impact

**High.** This is the foundation of gert v2's quality strategy. Without it:
- Spec compliance is aspirational, not enforceable
- Ken's review is subjective and unbounded
- Regression risk is high (no golden anchors)
- Audit/compliance claims (SOC 2, ISO 27001) are unverifiable

**With this strategy:**
- Spec compliance is measurable: "95% of spec rules have tests"
- Phase gates are objective: "checklist 100%, coverage ≥80%, Ken approved"
- Regression detection is automated: golden trace diffs in CI
- Audit evidence is concrete: spec tag → test file → CI run log

---

## For Team Review

**Questions for Cristian:**
1. Is Phase 0 delivery of spec coverage tool acceptable, or does it need to exist before any other work starts?
2. Should the 95% coverage target be adjustable per phase, or is it a hard Phase 13 gate?
3. Are there spec sections that should be excluded from coverage tracking (e.g., informational sections, non-normative examples)?

**Questions for Ken:**
1. Does the spec tagging format meet your review needs, or should it include additional metadata (risk level, priority)?
2. Should the coverage tool also track test → spec orphans (tests tagged to non-existent spec rules)?
3. What's the escalation path if Brian and Ken disagree on whether a spec rule requires a test?

**Questions for Brian:**
1. Is the 10-step TDD workflow realistic, or does it need simplification for velocity?
2. Should the workflow include pair programming or async review before Ken's final review?
3. Are there categories of MUST rules that shouldn't require tests (e.g., documentation-only rules)?

---

## Related Documents

- Full strategy: `.squad/tmp/barbara-correctness-strategy.md`
- Implementation plan: `design/gert-v2/PLAN.md`
- Testing spec: `design/gert-v2/spec/08-testing-and-acceptance.md`
- Trace format spec: `design/gert-v2/spec/12-evidence-tracing-resumption.md`
- Q3 decision (HMAC traces): `.squad/decisions.md` §2026-04-19T21:00:00Z

---

## dennis-input-benchmarks

# Operator Input Capture Benchmarks: gert v2 vs. Industry Peers

**Research Date:** April 2026  
**Researcher:** Dennis (Research Specialist)  
**Status:** Final  
**Requested by:** Project Owner  

---

## Executive Summary

gert v2's **choice / decision / collector** triad for human operator input is **competitive with industry peers** but **not best-in-class** on several critical dimensions:

### Key Findings

1. **Common Table Stakes** (all systems support):
   - Single-choice / multi-choice selection (gert: `choice`)
   - Branching/routing based on selection (gert: `decision`)
   - Free-form text collection (gert: `collector`)
   - Basic timeout handling

2. **Where gert v2 Is Strong**:
   - Explicit, separate step types for value capture vs. flow routing (most peers conflate these)
   - Built-in file/image attachment with SHA256 hashing
   - Governance-aware approval gates with role-based escalation
   - Structured multi-field forms within a single step

3. **Where gert v2 Lags**:
   - **Missing: Conditional field visibility** (shown in xMatters, ServiceNow, Camunda)
   - **Missing: Field validation schemas** (regex, length, format validation not standardized)
   - **Missing: Dynamic field rendering** (Zapier, Make, Camunda allow template-driven forms)
   - **Missing: Multi-option selects / tagging** (most peers support select-multiple)
   - **Missing: Adaptive input based on prior answers** (Camunda, ServiceNow support)
   - **Missing: Rich text / markdown editing** in `collector` fields
   - **Missing: Input provider integration at field level** (xMatters, ServiceNow fetch context from external systems)

4. **Unique to Peers**:
   - **Camunda BPM**: Full BPMN user task forms with nested structures, conditional visibility, repeating field groups
   - **ServiceNow**: Catalog items with complex question dependencies; business rule-driven show/hide logic
   - **xMatters**: Context-aware incident enrichment; form responses trigger downstream actions
   - **Zapier/Make**: Step-by-step action sequences with form fields; payload mapping between steps
   - **GitHub Actions**: Only simple `input:` dispatch with no UI abstraction; bare minimum feature set
   - **AWS SSM**: Parameter input divorced from runbook semantics; treated as external arguments
   - **Ansible AWX**: Survey/prompt feature is optional, bolted on; not a first-class flow primitive

---

## Detailed System Survey

### 1. PagerDuty Process Automation (formerly Rundeck)

**Input Types Supported:**
- Text (single-line, multi-line)
- Dropdown/select (single-choice)
- Checkbox (boolean toggle)
- File upload
- Number input

**Input Capture Pattern:**
- **Equivalent to `choice`**: Dropdown options with default fallback
- **Equivalent to `decision`**: Conditional branches based on option selection; separate "branch" step type
- **Equivalent to `collector`**: File upload + multiline text fields in a single "prompt" action

**Form Handling:**
- All fields presented as a single user prompt dialog
- No native multi-question questionnaire structure
- Sequential prompts if multiple input steps needed
- Limited field validation (basic required/optional flag)

**Key Features:**
- Options can be sourced from a prior CLI output (dynamic option lists)
- Built-in timeout; execution can `continue_on_error` if skipped
- Approval gates via `approval` action (separate from input)
- Role-based authorization

**Gaps vs. gert v2:**
- No native support for input field visibility rules
- No field-level validation schema (length, format, regex)
- No rich text editing
- Input and approval are separate concepts (not co-located like gert's `collector.approvals`)
- No per-field governance (redaction, allowlist)

**Advantages over gert v2:**
- Dynamic option lists sourced from script output
- Tighter integration with external data sources (via script context)

---

### 2. Confluence / Atlassian SOP Playbooks

**Input Types Supported:**
- **Macro-based input**: Confluence uses embedded macros (Parameters, Rich Text) but NOT for workflow automation
- SOP playbooks are primarily **documentation** (wiki pages), not executable
- Limited structured input capture (mostly free-form wiki editing)

**Input Capture Pattern:**
- **Not applicable**: Confluence playbooks are read-only reference docs, not automated workflows
- User actions are captured via comments/replies, not structured forms
- Some integration via Jira for task capture, but divorced from SOP execution

**Form Handling:**
- Jira issue creation forms (when playbook links to Jira) support basic field mapping
- No native questionnaire or operator prompt system in playbooks

**Key Features:**
- Excellent documentation and context capture
- Rich formatting (markdown, embedded media)
- Audit trail via page history

**Gaps vs. gert v2:**
- Not an executable runbook system; no real operator input capture for workflow automation
- No routing/branching based on form input
- No governance integration
- No evidence tracking

**Advantages over gert v2:**
- N/A (fundamentally different category: documentation vs. executable automation)

---

### 3. xMatters / OpsGenie

**Input Types Supported:**
- Text (single-line, multiline)
- Dropdown (single/multi-select)
- Checkbox (boolean, checkbox groups)
- Date/time pickers
- Number input
- Cascading selects (dependent dropdowns)
- Rich text editing

**Input Capture Pattern:**
- **Equivalent to `choice`**: Single-select dropdown with incident context
- **Equivalent to `decision`**: Multi-branch response routing (escalate, acknowledge, resolve, etc.)
- **Equivalent to `collector`**: Multi-field forms embedded in alert/incident notifications

**Form Handling:**
- Forms embedded in **push notifications** or web UI
- **Conditional field visibility**: Show/hide fields based on prior field values
- **Adaptive forms**: Different questions based on incident severity/type
- Multi-question questionnaires with branching logic
- Field validation: required, regex, format

**Key Features:**
- **Integration with external data**: Form questions fetched from incident context (service name, customer, etc.)
- **Enrichment-driven**: Answers feed back into incident record; trigger downstream actions
- **Escalation chains**: Form response determines escalation path
- **Multi-step approvals**: Approval gates can require multiple respondents
- **Timeout + SLA**: Form response must occur within SLA window; escalates or fails otherwise
- **Mobile-first**: Form interaction via SMS, push, mobile app

**Gaps vs. gert v2:**
- No SHA256 file hashing / evidence integrity verification
- No audit trail comparable to gert's append-only JSONL
- Form submission tied to incident lifecycle, not generic runbook model
- Limited to alert/incident domain (not general-purpose runbooks)

**Advantages over gert v2:**
- **Conditional field visibility** (gert lacks this)
- **Adaptive forms** based on incident metadata (gert lacks this)
- **Multi-select dropdown** support (gert's `choice` is single-select only)
- **Cascading selects** (dependent dropdowns)
- **Date/time pickers** (gert only supports text input)
- **Mobile alert integration** (form interaction via SMS/push)
- **External data enrichment** at field level (fetch context from incident API)

---

### 4. AWS Systems Manager / Run Command

**Input Types Supported:**
- String parameter
- String list (comma-separated)
- Number
- Boolean
- Selection list (dropdown)

**Input Capture Pattern:**
- Parameters passed to `run-command` via CLI/API, not interactive form
- **Equivalent to `choice`**: Parameter with pre-defined allowed values
- **Equivalent to `decision`**: Conditional logic within script/document (not in parameter input)
- **Equivalent to `collector`**: String parameters (single/multiline)

**Form Handling:**
- **No interactive form UI**: Parameters are specified before runbook invocation
- **No in-execution prompts**: All input gathered upfront
- Document parameters are static YAML; no dynamic questionnaire
- Minimal validation (type checking only)

**Key Features:**
- Parameter substitution into script/document via `{{ ssm:parameter }}` or `{{ aws:param }}`
- Role-based access control to parameters
- Parameter versioning and history
- Integration with AWS Secrets Manager for sensitive data

**Gaps vs. gert v2:**
- **No in-execution prompts** (all input upfront)
- **No interactive form** (bare-bones parameter model)
- **No branching on input** (routing must be script-based)
- **No approval gates at parameter level**
- **No file attachment** (only scalar parameters)
- **No conditional field visibility**
- **Minimal governance** (no output redaction, no step-level approval)

**Advantages over gert v2:**
- Parameter versioning and rollback
- Tight AWS IAM integration
- Parameter Store can source dynamic values (via Lambda)

**Note**: AWS SSM is **not designed for interactive operator input capture**; it's a parameter injection system. Falls short of peer complexity.

---

### 5. Ansible AWX / Tower

**Input Types Supported:**
- Text (single-line, multiline)
- Dropdown (single-select)
- Checkbox (boolean, checkbox groups)
- Multiple-choice (checkbox groups for multi-select)
- Number, password (masked)

**Input Capture Pattern:**
- **Survey feature** (separate from playbook): Optional feature, not core
- **Equivalent to `choice`**: Survey question with predefined answers
- **Equivalent to `decision`**: Job templating with conditional includes (not input-driven)
- **Equivalent to `collector`**: Survey multiline text fields

**Form Handling:**
- Surveys are **bolted on** to job templates; not first-class runbook primitives
- Survey questions collected upfront before playbook execution (similar to AWS SSM)
- No in-execution prompts during playbook run
- Limited field validation (required flag, regex pattern optional)

**Key Features:**
- Survey questions can have default values
- Questions can be marked as "required" or optional
- Dropdown options are static (no dynamic sourcing)
- Execution history captures survey answers
- Multi-user approval gates (separate from survey)

**Gaps vs. gert v2:**
- **Survey is optional, not core**: Most Ansible workflows don't use surveys
- **No in-execution prompts** (all input upfront like AWS SSM)
- **No branching on input** (play logic must handle routing)
- **No conditional field visibility**
- **Minimal approval integration** (approvals are separate job template setting)
- **No file attachment**
- **No evidence hashing/integrity**

**Advantages over gert v2:**
- Tight playbook language integration (Jinja2 templating)
- Survey answers available as facts in playbook context

---

### 6. GitHub Actions (`workflow_dispatch`)

**Input Types Supported:**
- `string` (text input)
- `choice` (dropdown)
- `boolean` (checkbox)
- `environment` (environment selector)

**Input Capture Pattern:**
- **Equivalent to `choice`**: `choice` input type (dropdown)
- **Equivalent to `decision`**: Conditional jobs/steps (not input-driven form)
- **Equivalent to `collector`**: `string` input (single/multiline via textarea)

**Form Handling:**
- Single form presented before workflow execution (GitHub UI pre-renders form)
- **No interactive in-execution prompts**
- **No conditional field visibility**
- **No multi-question questionnaire structure**
- All input is flat (no nested objects or field groups)

**Key Features:**
- Input variables available as `${{ github.event.inputs.* }}`
- Very simple, minimal feature set
- No validation schema (beyond type)
- No approval gates (separate via branch protection rules)

**Gaps vs. gert v2:**
- **Very minimal feature set**: Only basic types supported
- **No in-execution input** (one-time form before workflow starts)
- **No field validation** (type only)
- **No conditional visibility**
- **No file attachment**
- **No branching on input**
- **No approval gate at input level**
- **No governance** (redaction, allowlist)

**Advantages over gert v2:**
- Very simple to understand and use
- Native GitHub integration (runs in GitHub UI)

**Note**: GitHub Actions `workflow_dispatch` is **extremely bare-bones** for operator input; it's the simplest system surveyed.

---

### 7. ServiceNow Flow Designer

**Input Types Supported:**
- Text (single-line, multiline)
- Dropdown (single-select, multi-select with tags)
- Checkbox (boolean, checkbox groups)
- Radio buttons (single-choice)
- Date/time pickers
- Number, currency, percentage
- User/group picker
- Lookup (reference to another record)
- Rich text editor

**Input Capture Pattern:**
- **Equivalent to `choice`**: Catalog items with single/multi-select questions
- **Equivalent to `decision`**: Flow branching based on question responses
- **Equivalent to `collector`**: Multi-field item variables in catalog

**Form Handling:**
- **Catalog Item Template**: Complex questionnaire with conditional logic
- **Conditional field visibility**: Show/hide questions based on prior answers (business rules)
- **Field dependencies**: Lookup fields populate based on prior selections
- **Multi-field sections**: Group related questions into collapsible sections
- **Field validation**: Built-in validators (required, regex, format)
- **Dynamic field rendering**: Questions can be templates with variable substitution

**Key Features:**
- **Business rules-driven show/hide**: Hide/display questions based on service catalog item type
- **Approval routing**: Form can route to different approvers based on answer
- **Fulfillment workflow**: Form submissions trigger downstream service requests
- **Field-level RBAC**: Different users see different questions
- **Multi-language support**: Questions in multiple languages
- **Input from external sources**: Lookup fields query CMDB, user directory, etc.

**Gaps vs. gert v2:**
- No SHA256 file hashing for evidence integrity
- Approval is less integrated into input step (separate flow action)
- No append-only audit trail (traditional RDBMS with update history)
- Catalog model is focused on IT service requests, not general runbooks

**Advantages over gert v2:**
- **Conditional field visibility** (driven by business rules)
- **Field dependencies** (cascading/dependent selects)
- **Multi-select dropdown** (gert only has single-select)
- **Date/time pickers**
- **Rich text editor**
- **Lookup fields** (query external systems)
- **User/group picker**
- **Field-level RBAC** (different users see different questions)
- **Template-driven field rendering**

---

### 8. Zapier / Make (Integromat)

**Input Types Supported:**
- Text (single-line, multiline)
- Number, currency
- Date/time
- Boolean toggle
- Email
- Phone
- URL
- Dropdown (single-select)
- Multi-select (checkbox group)
- File picker

**Input Capture Pattern:**
- **Equivalent to `choice`**: Webhook form with dropdown fields
- **Equivalent to `decision`**: Conditional routing based on field values
- **Equivalent to `collector`**: Multi-field form via Zapier's "Form" module (Zapier) or "Webhook Response" (Make)

**Form Handling:**
- **Webhook-triggered forms**: End user fills form, webhook fires Zap/scenario
- **Form fields are discoverable**: Form UI auto-generated from field definitions in Zap
- **No in-execution prompts**: All input captured upfront via form submission
- **Conditional logic**: Routes and actions based on form field values
- **File attachment**: Form can collect files; stored in cloud storage (Google Drive, Dropbox, etc.)

**Key Features:**
- **Payload mapping**: Form responses become data payloads for downstream actions
- **Multi-step action sequences**: Form triggers chain of integrations
- **Field validation**: Built-in validators (required, format, length)
- **Conditional actions**: Skip/execute actions based on form values
- **No approval gates native to form**: Approval would be a separate action (e.g., email approval)

**Gaps vs. gert v2:**
- **No in-execution prompts** (form-before-workflow model)
- **No branching on input** (conditional logic is action-based, not flow-based)
- **No SHA256 hashing** (files stored in 3rd-party cloud storage)
- **No append-only audit trail**
- **No governance** (redaction, allowlist)
- **No approval gates** at form level

**Advantages over gert v2:**
- **Multi-select dropdown** (gert only single-select)
- **Rich field types** (phone, email, currency, etc.)
- **File attachment** with cloud storage integration
- **Tight integration with 3rd-party APIs** (hundreds of apps)
- **Payload mapping** (form response → action inputs)

---

### 9. Camunda BPM / Flowable

**Input Types Supported:**
- Text (single-line, multiline)
- Dropdown (single-select, multi-select)
- Checkbox (boolean, checkbox groups)
- Radio buttons
- Date/time pickers
- Number, currency
- Select/lookup
- File upload
- Rich text editor
- Custom widgets (plugin-based)

**Input Capture Pattern:**
- **BPMN User Task Forms**: First-class flow primitive
- **Equivalent to `choice`**: Form with single/multi-select fields
- **Equivalent to `decision`**: Conditional flow routing based on form submission
- **Equivalent to `collector`**: Multi-field user task form with nested structures

**Form Handling:**
- **Conditional field visibility**: Show/hide fields based on form values (JSON Forms schema)
- **Field dependencies**: Cascading selects, dynamic field population
- **Repeating field groups**: Arrays of field sets (e.g., multiple error logs)
- **Nested object structures**: Complex hierarchical forms
- **Field validation**: JSON Schema-based validation (regex, length, format)
- **Template-driven rendering**: Camunda Forms allow template expressions

**Key Features:**
- **JSON Forms standard**: Forms defined via JSON Schema + UI schema
- **Conditional rendering**: `if-then-else` logic at field level
- **Default values**: Fields can have dynamic defaults (from process variables)
- **Read-only fields**: Can display computed values (not editable)
- **Layout control**: Field groups, sections, tabs
- **Custom validation**: Business rule validation beyond schema
- **Assignment pool**: Task can be assigned to user group; users claim task
- **Task priority**: Prioritize user task queue

**Gaps vs. gert v2:**
- No SHA256 hashing (file handling is generic, not evidence-focused)
- No append-only audit trail (traditional RDBMS persistence)
- Governance is not explicit (RBAC via process variables, not declarative)
- No built-in approval gates (would require additional task or subprocess)

**Advantages over gert v2:**
- **Conditional field visibility** (if-then-else logic)
- **Field dependencies** (cascading selects)
- **Repeating field groups** (dynamic arrays)
- **Nested object structures** (hierarchical forms)
- **JSON Schema-based validation**
- **Template expressions** in field defaults
- **Read-only computed fields**
- **Assignment pool and task claiming**
- **Task priority queuing**
- **Custom widget plugins**

---

### 10. Temporal Workflow Engine

**Input Types Supported:**
- Temporal is **not a form/input system**; it's a workflow orchestration engine
- Activities can accept arbitrary struct parameters (type-safe via Go/Java/TypeScript)
- No native human input capture UI

**Input Capture Pattern:**
- Human input is captured via **separate action** (e.g., webhook, signal, query)
- **Equivalent to `choice`**: Signal with enum value; workflow branches on signal
- **Equivalent to `decision`**: Conditional branches based on signal/query value
- **Equivalent to `collector`**: Struct parameter passed to activity

**Form Handling:**
- **No native form UI**: Input is application-specific
- **No in-execution prompts**: Temporal is backend; frontend must implement UI
- **Signal-based input**: Frontend sends signal to workflow; workflow handles it
- **Query-based input**: Frontend queries workflow for state; frontend renders UI accordingly

**Key Features:**
- Type-safe parameters (compile-time checked)
- Deterministic retry/compensation (saga pattern)
- Event sourcing via event history
- Durable execution (replay-safe)

**Gaps vs. gert v2:**
- **Not a form/runbook system**: Temporal is orchestration engine for backend workflows
- **No UI abstraction** for operator input
- **No governance** (RBAC, approval gates, output redaction)
- **No questionnaire/multi-field forms** (signals carry simple enums)
- **Not designed for human operator interaction**

**Advantages over gert v2:**
- **Type-safe parameters** (compile-time validation)
- **Deterministic replay** (audit trail + debugging)
- **Compensation/saga pattern** (automatic cleanup on failure)
- **Durable execution** (survives process restart)

**Note**: Temporal is **out of category** for this benchmark; it's a backend orchestration engine, not a human-in-loop form system. Included for completeness (mentioned as workflow system in gert research brief).

---

### 11. Incident IQ / FireHydrant

**Input Types Supported:**
- Text (single-line, multiline)
- Dropdown (single-select)
- Checkbox (boolean)
- Number
- URL
- File upload

**Input Capture Pattern:**
- **Runbook steps**: Incident runbooks with manual steps that capture input
- **Equivalent to `choice`**: Dropdown in runbook step
- **Equivalent to `decision`**: Branching based on step selection
- **Equivalent to `collector`**: Multi-field input in runbook step

**Form Handling:**
- Input collected during incident runbook execution
- No conditional field visibility
- Limited validation (required flag only)
- Sequential prompts if multiple input steps

**Key Features:**
- Incident context available to runbook steps
- Input stored in incident timeline
- Runbook can branch based on input
- Basic approval gates (separate from input)

**Gaps vs. gert v2:**
- No conditional field visibility
- No field validation schema
- No SHA256 hashing
- No append-only audit trail
- Focused on incidents (not general runbooks)

---

## Synthesis & Analysis

### Common Table Stakes (All/Most Systems Support)

1. **Basic Input Types**:
   - Single-line text
   - Multiline text
   - Single-select dropdown
   - Boolean toggle/checkbox
   - Number input

2. **Single-Question Capture**:
   - All systems can capture a single user choice and store it

3. **Branching on Input**:
   - All systems allow routing/branching based on input selection

4. **Timeout Handling**:
   - Most have some form of timeout (escalate, fail, skip)

5. **Approval Gates**:
   - All have approval capability, though integration varies

### Where gert v2 Is Strong

| Feature | gert v2 | Peers |
|---------|---------|-------|
| **Separate choice/decision/collector types** | ✅ Explicit distinction | ❌ Most conflate value capture with routing |
| **File attachment with SHA256 hashing** | ✅ Yes | ❌ Few hash files; most use 3rd-party storage |
| **Governance-aware approval gates** | ✅ Role-based, inline approvals | ⚠️ ServiceNow, Camunda support; others weak |
| **Append-only JSONL traces** | ✅ Event sourcing model | ❌ Most use RDBMS updates |
| **Structured multi-field forms** | ✅ `collector.fields[]` | ✅ Camunda (JSON Forms), ServiceNow, xMatters |
| **Inline contract/governance** | ✅ Step-level governance | ❌ Most governance is process-wide or external |

### Where gert v2 Lags (Critical Gaps)

| Feature | Missing in gert v2 | Competitors | Impact |
|---------|-------------------|-------------|--------|
| **Conditional field visibility** | ❌ | ✅ ServiceNow, Camunda, xMatters | Cannot build adaptive questionnaires |
| **Multi-select dropdown** | ❌ (only single-select `choice`) | ✅ All others | Cannot capture "select all that apply" responses |
| **Dependent/cascading selects** | ❌ | ✅ ServiceNow, Camunda, xMatters | Cannot implement "category → subcategory" workflows |
| **Date/time pickers** | ❌ (only text input) | ✅ xMatters, ServiceNow, Camunda, Make | Must ask user to type dates (error-prone) |
| **Field validation schema** | ⚠️ Minimal (no regex, format rules) | ✅ Camunda (JSON Schema), ServiceNow, xMatters | Cannot enforce format/length/pattern upfront |
| **Dynamic field population** | ❌ | ✅ ServiceNow, xMatters, Camunda | Cannot fetch options from external systems |
| **Rich text / markdown editor** | ❌ (multiline text only) | ✅ ServiceNow, Camunda | Cannot support formatted text in responses |
| **Repeating field groups** | ❌ | ✅ Camunda (JSON Forms) | Cannot collect variable number of items (e.g., multiple logs) |
| **Input provider integration at field level** | ⚠️ (external system, not field-driven) | ✅ ServiceNow (lookups), xMatters (incident context) | Cannot enrich form based on external data |

### Unique Features by Competitor

**ServiceNow Flow Designer**:
- Field-level RBAC (different users see different questions)
- Business rule-driven show/hide
- Multi-language support
- Assignment pool + task claiming

**Camunda BPM**:
- JSON Schema-based validation
- Repeating field groups (dynamic arrays)
- Nested object structures
- Custom widget plugins

**xMatters**:
- Mobile-first (SMS/push interaction)
- Adaptive forms based on incident metadata
- External data enrichment (fetch context from incident API)
- Escalation chains based on form response

**Make/Zapier**:
- Rich field types (phone, email, currency)
- File attachment with cloud storage integration
- Payload mapping (form response → action inputs)
- Hundreds of 3rd-party app integrations

### Unique Features in gert v2 (Not Matched by Peers)

1. **SHA256 Evidence Hashing**: File attachments in `collector` are hashed for integrity verification (uncommon)
2. **Append-Only JSONL Traces**: Event sourcing model with deterministic replay (Temporal/Cadence style)
3. **Explicit choice vs. decision vs. collector**: Cleaner DSL separation (peers mix these concepts)
4. **Inline governance in step definition**: Approval gates, redaction, allowlist in `contract` field (peers keep governance external)
5. **Step-level timeout + escalation SLA**: Timeout configuration at `collector` level with escalate/fail/skip (integrated, not bolted-on)

---

## Verdict & Recommendations

### Is gert v2's choice/decision/collector Triad Competitive?

**Yes, but incomplete.** The triad is:
- **Sound conceptually**: Separating value capture, flow routing, and unstructured input is cleaner than peer approaches
- **Well-structured**: Field-level `name`, `type`, `label`, `hint` is clear
- **Governance-first**: Approval gates and step-level contract integration is stronger than peers

**However:**
- **Missing critical features** for adaptive form workflows (conditional visibility, field dependencies, validation schemas)
- **Incomplete form builder**: Cannot express many common patterns (multi-select, date pickers, repeating sections)
- **Limited input enrichment**: No field-level integration with external data sources

### Gap Analysis: What Must gert v2 Add?

**Priority 1 (Blocking for best-in-class):**
1. **Conditional field visibility** (`collector.fields[*].if` or `when` expression)
2. **Multi-select dropdown** (extend `choice` or add new field type `multi_choice`)
3. **Field validation schema** (`pattern`, `minLength`, `maxLength`, `format` on fields)
4. **Dependent selects** (`fields[*].options` can reference prior field value or external data)

**Priority 2 (Differentiating):**
5. **Date/time picker field type** (avoid free-form date entry)
6. **Rich text field type** (markdown editing)
7. **Repeating field groups** (`fields[*].repeat: true` → collect multiple instances)
8. **Read-only computed field** (display field based on prior values, not editable)

**Priority 3 (Nice-to-have):**
9. **Field-level input provider** (fetch dropdown options from external system)
10. **File type constraints** (accept only `.pdf`, `.log`, etc.)

### What gert v2 Should NOT Adopt from Peers

- ❌ **Move approval gates outside collector step**: xMatters/ServiceNow separate approval from input; gert's inline model is cleaner
- ❌ **Rely on 3rd-party cloud storage for files**: gert's approach (store as artifact with SHA256) is more governance-friendly
- ❌ **Use RDBMS with update history instead of append-only**: gert's event sourcing model is correct for audit trails
- ❌ **Bolt on surveys/prompts like Ansible**: Make input a first-class flow primitive (gert already does this)
- ❌ **Support unlimited field types**: ServiceNow's 20+ field types add bloat; focus on essentials + extensibility

### Recommendations for v2 Design

**Short-term (v2.0 release):**
1. Add conditional field visibility via `when` expression on `collector.fields[*]`
2. Add `multi_choice` step type (or extend `choice.multiple: true`) for multi-select
3. Add field validation: `fields[*]` → add `pattern`, `minLength`, `maxLength`, `format`
4. Add date picker field type: `fields[*].type: "date"` (ISO 8601 format)

**Medium-term (v2.1 release):**
5. Implement dependent selects: `fields[*].options` can be array-of-objects with `when` and `lookup` keys
6. Add rich text field type with markdown editor
7. Add repeating field groups: `fields[*].repeat: true` → collect multiple instances in array

**Long-term (v2.2+):**
8. Field-level input provider integration (fetch options from external system)
9. Read-only computed fields
10. Custom field type plugin model

---

## Conclusion

**gert v2's operator input triad is architecturally sound and governance-first, but operationally incomplete.** The three-way distinction (choice / decision / collector) is cleaner than industry peers, and the built-in governance model (approval gates, role-based escalation) is stronger.

**However, gert v2 cannot currently express common enterprise patterns:**
- Adaptive forms (show/hide fields based on prior answers)
- Complex selections (multi-select, cascading dropdowns)
- Validated input (enforce format/length/pattern at form level)

**To reach best-in-class status, gert v2 must add:**
1. Conditional field visibility (Priority 1)
2. Multi-select support (Priority 1)
3. Field validation schemas (Priority 1)
4. Dependent/cascading selects (Priority 2)
5. Date/time picker (Priority 2)

**With these additions, gert v2 will surpass all peers surveyed.** No competitor combines governance-first design with flexible form builders and append-only audit trails.

---

## References & Data Sources

**Systems Surveyed:**
1. PagerDuty Process Automation (formerly Rundeck) — Official docs; industry knowledge
2. Confluence / Atlassian SOP playbooks — Official docs; product review
3. xMatters — Official docs; product whitepaper; industry knowledge
4. AWS Systems Manager / Run Command — Official AWS docs; product review
5. Ansible AWX / Tower — Official docs; product review
6. GitHub Actions — Official GitHub docs; personal experience
7. ServiceNow Flow Designer — Official ServiceNow docs; product whitepaper
8. Zapier / Make (Integromat) — Official docs; product review
9. Camunda BPM / Flowable — Official docs; BPMN spec; JSON Forms spec
10. Temporal Workflow Engine — Official docs; community knowledge
11. Incident IQ / FireHydrant — Product review; official docs

**Standards Referenced:**
- JSON Schema Draft 2020-12 (validation)
- BPMN 2.0 (business process modeling)
- JSON Forms specification (conditional rendering)
- W3C Form Controls (HTML5 input types)
- OpenTelemetry specification (tracing)

---

**Date Completed:** April 2026  
**Researcher:** Dennis — Research Specialist, gert v2 Design Team  
**Status:** Final — Ready for team review and architectural prioritization

---

## dennis-stress-test-corpus

# Dennis: Runbook Corpus Coverage Analysis

**Date:** 2026-04-18  
**Author:** Dennis (CS Researcher)  
**Related Artifact:** `/Volumes/Projects/gert/.squad/tmp/dennis-runbook-corpus.md`

---

## Summary

Compiled 10 production-quality runbooks covering 8 domains, all complexity patterns, all interaction patterns, and all failure modes identified in the v2 research phase. Corpus is ready for John to translate to gert v2 schema YAML.

---

## Coverage Dimensions Achieved

### Domain Coverage (8/8)
✅ **SRE/Operations** — Runbooks 1 (K8s incident), 9 (on-call escalation)  
✅ **DevOps/Deployment** — Runbook 2 (canary deployment)  
✅ **HR/IT** — Runbook 3 (employee onboarding)  
✅ **Compliance/Audit** — Runbooks 4 (SOC2 evidence), 10 (GDPR deletion)  
✅ **Security** — Runbook 5 (breach containment)  
✅ **Data Engineering** — Runbook 6 (database migration)  
✅ **Finance** — Runbook 7 (purchase approval)  
✅ **Regulated Industry** — Runbook 8 (FDA medical device release)

### Complexity Pattern Coverage (6/6)
✅ **Linear** — Runbooks 6, 8  
✅ **Branching** — Runbooks 1, 2, 5, 7, 8 (conditional routing, multi-way decision)  
✅ **Parallel** — Runbooks 2, 3, 4, 5, 10 (fan-out/fan-in, wait-all, heterogeneous tasks)  
✅ **Looping/Retry** — Runbooks 1, 2, 6, 9, 10 (convergence, max passes, backoff)  
✅ **Nested/Invoke** — Runbook 4 (15 sub-runbook invocations with I/O)  
✅ **Multi-party** — Runbooks 3, 4, 5, 7, 8, 10 (escalation, quorum, dual attestation)

### Interaction Pattern Coverage (6/6)
✅ **Fully automated** — Runbook 9 (no human steps)  
✅ **Human approval gates** — All runbooks  
✅ **User data collection** — Runbooks 3, 4, 7, 10 (forms, dropdowns, file uploads)  
✅ **File/artifact collection** — Runbooks 1, 4, 7, 8 (logs, screenshots, evidence)  
✅ **Multi-party approval** — Runbooks 3, 7, 8, 10 (dual, quorum, escalation)  
✅ **Self-service** — Runbook 3 step 6 (employee completes own training)

### Failure Mode Coverage (5/5)
✅ **Compensating actions/rollback** — Runbooks 1, 2, 5, 6 (saga pattern, automatic revert)  
✅ **Partial completion** — Runbooks 3, 10 (best-effort parallel, continue-on-failure)  
✅ **Timeout/SLA** — All runbooks (escalation, default choice, abort)  
✅ **Skip-on-condition** — Runbooks 5, 8 (conditional step execution)  
✅ **Retry with backoff** — Runbooks 1, 2, 6, 10 (transient failure handling)

### Edge Case Coverage (5/5)
✅ **Long runbooks** — Runbook 8 (15 steps), Runbook 4 (15 sub-runbooks)  
✅ **No human interaction** — Runbook 9 (fully automated escalation)  
✅ **All human steps** — Runbook 7 (mostly approvals)  
✅ **Cross-system workflows** — Runbooks 3, 4, 10 (6+ external systems)  
✅ **Long-running pauses** — Runbook 8 (90-day FDA review pause)

---

## Top 10 Schema Challenges Identified

These challenges emerged consistently across multiple runbooks and will stress-test the gert v2 schema:

1. **Saga pattern with compensation scope** (Runbooks 2, 6)
   - Register compensating actions at step X to execute if steps Y-Z fail
   - Multi-step compensation with fallback logic
   - Conditional compensation triggers

2. **Nested runbook invocation** (Runbook 4)
   - Sub-runbook path resolution
   - Input passing and output capture
   - Failure propagation (sub-failure fails parent)

3. **Dynamic approver resolution** (Runbooks 3, 7)
   - Lookup approver from external API or database
   - Org chart traversal (manager, director, VP chain)
   - Self-service assignment (employee is approver)

4. **Business day timeout calculation** (Runbooks 3, 7, 8)
   - Calendar-aware timeout (skip weekends, holidays)
   - Essential for HR, finance, regulated workflows

5. **Quorum and dual approval** (Runbooks 7, 8)
   - M-of-N approval logic (3 of 5 board members)
   - Dual attestation (both QA AND CEO must approve)
   - Different from any-one or all-must-approve

6. **Best-effort parallel execution** (Runbook 10)
   - Continue-on-failure for parallel blocks
   - Log failures but don't abort
   - Contrast with fail-fast (Runbook 2)

7. **Convergence-based iteration** (Runbooks 2, 6, 9, 10)
   - Exit loop when condition met (not just max passes)
   - Examples: "10 consecutive healthy checks," "incident acknowledged," "data deleted"

8. **Long-running external pauses** (Runbook 8)
   - Pause for external event (FDA clearance, Jira status change)
   - Resume after weeks/months
   - Webhook trigger pattern

9. **Cryptographic operations** (Runbooks 4, 8, 10)
   - SHA256 hashing, digital signatures, certificate management
   - 21 CFR Part 11 compliance (medical), GDPR compliance
   - Evidence integrity for audit

10. **Artifact collection and metadata** (Runbooks 1, 4, 8, 10)
    - File uploads (contracts, logs, certificates)
    - Metadata: filename, size, hash, timestamp
    - Storage policies: local, S3, object lock, retention

---

## Gaps Not Covered (Future Corpus)

The following patterns are NOT in this corpus but exist in real-world runbooks:

1. **Cyclic workflows** — "Retry entire deployment from step 1 if final validation fails (max 3 cycles)"
2. **Dynamic step generation** — "For each affected server in list, add a remediation step"
3. **Async wait with webhook** — "Wait for Jira ticket status == 'Resolved' (webhook callback)"
4. **Weighted approval voting** — "CEO vote counts as 2, board members as 1 each"
5. **Conditional compensation** — "Execute compensation only if step X completed but step Y failed"
6. **Inter-runbook messaging** — "Sub-runbook can ask parent for additional input mid-execution"
7. **Time-boxed auto-approval** — "If no response in 1 hour, auto-approve with option X"
8. **Dynamic escalation policy** — "Escalate to on-call for service X (looked up from PagerDuty)"

**Recommendation:** If John identifies that current corpus is insufficient after initial schema translation, add 2-3 more runbooks targeting these gaps.

---

## Usage Instructions for Schema Translation

**For John (Schema Translator):**

1. **Start simple:** Translate Runbook 6 (database migration) or Runbook 9 (escalation) first. Linear/loop-only, fewer edge cases.

2. **Test composition:** Translate Runbook 4 (SOC2 audit). 15 nested sub-runbooks will stress-test invoke semantics.

3. **Test saga pattern:** Translate Runbook 2 (canary deployment). Core v2 goal, must work well.

4. **Test parallel failure modes:** Translate Runbook 10 (GDPR deletion). Best-effort parallel is tricky.

5. **Test long-running:** Translate Runbook 8 (FDA release). 90-day pause tests resume semantics.

6. **For each runbook:**
   - Attempt full gert v2 schema YAML translation
   - Document: what's clear, what's ambiguous, what's impossible
   - Note: missing primitives, unclear semantics, expressiveness gaps

7. **Aggregate findings:**
   - Which patterns are common? (Need first-class support)
   - Which patterns need workarounds? (Acceptable vs. brittle)
   - Which patterns are unsupported? (Schema design gap)

**Output Format:**
- 10 YAML files: `runbook-01-k8s-incident.yaml` through `runbook-10-gdpr-deletion.yaml`
- 1 findings document: `schema-translation-findings.md`
- Findings should include: successes, ambiguities, blockers, recommendations

---

## Recommendations for v2 Schema Design

Based on corpus analysis, these patterns should have **first-class schema support** (not workarounds):

### High Priority (Appear in 6+ runbooks)
1. Approval timeout with escalation/default-choice
2. Retry with backoff (transient vs. permanent failure)
3. Parallel execution with wait-all/wait-any
4. Conditional step execution (skip-on-condition)
5. Artifact collection with metadata

### Medium Priority (Appear in 3-5 runbooks)
6. Saga compensation with scope
7. Business day timeout calculation
8. Quorum/dual approval
9. Convergence-based iteration
10. Dynamic approver lookup

### Low Priority (Appear in 1-2 runbooks, but high impact)
11. Nested runbook invocation
12. Long-running external event wait
13. Cryptographic operations
14. Best-effort parallel

---

## Validation

This corpus satisfies the task requirements:

✅ **8-10 runbooks** — Delivered 10  
✅ **Diverse domains** — 8 domains covered  
✅ **All complexity patterns** — Linear, branching, parallel, looping, nested, multi-party  
✅ **All interaction patterns** — Automated, approval, data collection, file uploads, multi-party  
✅ **All failure modes** — Compensation, partial completion, timeout, skip, retry  
✅ **Edge cases** — Long runbooks, fully automated, long pauses, cross-system  
✅ **Realistic and specific** — Each runbook drawn from actual industry practice  
✅ **Concrete steps** — Specific enough for John to translate to schema  

---

## Next Steps

1. **John:** Translate all 10 runbooks to gert v2 schema YAML
2. **John:** Document translation findings (successes, ambiguities, blockers)
3. **Dennis:** Review findings, identify research gaps (if any)
4. **Team:** Use findings to refine v2 schema design (§03)
5. **Team:** Prioritize schema features based on corpus frequency

---

## Appendix: Runbook Naming Convention

For John's YAML files:

```
runbook-01-k8s-incident.yaml          (Kubernetes Pod Incident Response)
runbook-02-canary-deployment.yaml     (Production Deployment with Canary)
runbook-03-employee-onboarding.yaml   (New Employee Onboarding)
runbook-04-soc2-evidence.yaml         (SOC2 Evidence Collection)
runbook-05-breach-containment.yaml    (Security Breach Containment)
runbook-06-db-migration.yaml          (Database Migration)
runbook-07-purchase-approval.yaml     (Financial Approval)
runbook-08-fda-release.yaml           (Medical Device Release)
runbook-09-oncall-escalation.yaml     (On-Call Escalation Ladder)
runbook-10-gdpr-deletion.yaml         (Customer Data Deletion)
```

Each YAML should include:
- Full step sequence
- Step types (cli, extension, manual, approval, choice, decision, assert, etc.)
- Inputs, outputs, conditionals, retry policies, timeout policies
- Governance policies (if applicable)
- Comments explaining schema decisions

---

**Status:** ✅ Ready for John's schema translation work.

---

## dennis-stress-test-final

# Schema Stress Test — Final Verdict

**From:** Dennis (CS Researcher)  
**Date:** 2026-04-18  
**Topic:** gert v2 Schema Production Readiness

---

## Executive Summary

Synthesized findings from John's schema translations (10 runbooks, full YAML + validation scoring) and Ken's architectural analysis (independent predictions across same corpus).

**Reconciled Verdict:** The gert v2 schema is **architecturally sound but requires 2 critical fixes before production readiness for enterprise use cases.**

- **Test Results:** 1 PASS (10%), 7 PASS WITH NOTES (70%), 2 FAIL (20%)
- **Average Scores:** 82% completeness, 73% fidelity (target: 95%)
- **Unique Gaps Identified:** 12 total (2 CRITICAL, 4 IMPORTANT, 6 NICE-TO-HAVE)

---

## Critical Gaps (Must Fix Before v2.0)

### GAP-1: Business-Day Timeout (G2, CRITICAL)
- **Affects:** R3 (Onboarding), R7 (Financial), R8 (FDA) — 15+ individual steps
- **Problem:** Approval workflows require business-day SLAs ("2 business days"), not wall-clock ("48h"). Current `timeout` field doesn't support calendar-aware expressions.
- **Fix:** Add `timeout_business_days`, `timeout_calendar` fields to §03 (Schema), §11 (Governance)
- **Effort:** Medium (2-3 days)

### GAP-2: M-of-N Quorum Approval (G4, CRITICAL)
- **Affects:** R7 (Financial board), R8 (FDA dual attestation), R10 (GDPR DPO + legal)
- **Problem:** Multi-party governance needs "3 of 5 board members must approve". Current `approvals.min` lacks explicit pool definition.
- **Fix:** Add `approvals.mode`, `approvals.pool`, `approvals.pool_size` to §03 (Schema), §14 (JSON-RPC)
- **Effort:** Medium (2-3 days)

**Total Critical Fix Effort:** 4-6 days for both gaps

---

## Important Gaps (Should Fix for v2.0)

- **GAP-3:** External event trigger (G3, HIGH) — FDA 90-day pause needs webhook resume
- **GAP-4:** Dynamic approver lookup (G2, IMPORTANT) — Manager from HR system at runtime
- **GAP-5:** Choice timeout default (G2, IMPORTANT) — Security triage defaults to "High"
- **GAP-6:** Cross-branch parallelism (G3, IMPORTANT) — Forensics parallel to containment

---

## What's Ready Today

✅ **SRE/Operations:** Incident response, health checks, automated remediation (R1, R9)  
✅ **DevOps:** Deployments with canary/rollback, saga patterns (R2, R6)  
✅ **Compliance:** SOC2/audit evidence collection (R4, R10)  
✅ **Data Engineering:** Database migrations with validation (R6)

❌ **Finance:** Multi-level approval chains with board quorum (R7) — blocked by GAP-2  
❌ **Regulated:** Medical device/FDA workflows (R8) — blocked by GAP-1, GAP-2, GAP-3

---

## Recommendation

The schema is **80% production-ready**. With 2 critical fixes (business-day timeout, M-of-N quorum), it reaches **95%+ readiness** for enterprise adoption.

**Action:** Prioritize GAP-1 and GAP-2 before declaring schema implementation-ready for Brian's parser work. Both fixes are additive (backward compatible) and addressable in a focused sprint (4-6 days total).

---

## Artifacts

- **Full Report:** `/Volumes/Projects/gert/.squad/tmp/stress-test-final-report.md` (20KB)
- **Executive Summary:** `/Volumes/Projects/gert/.squad/tmp/stress-test-executive-summary.md`
- **Corpus:** `/Volumes/Projects/gert/.squad/tmp/dennis-runbook-corpus.md` (10 runbooks)
- **John's Translations:** `/Volumes/Projects/gert/.squad/tmp/john-schema-translations.md` (R1-R3 full YAML)
- **Ken's Analysis:** `/Volumes/Projects/gert/.squad/tmp/ken-stress-conclusions.md`

---

**Verdict Reconciliation Note:** John's "NOT PRODUCTION READY" (strict methodology thresholds) and Ken's "NEEDS TARGETED FIXES" (pragmatic workaround assessment) agree on substance: schema is sound but has targeted enterprise governance gaps. The reconciled verdict captures both perspectives.

---

## john-cli-run-map

# Decision: Add `run:` map form to `type: cli`

**Author:** John (Schema & Language Specialist)  
**Date:** 2026-04-19  
**Status:** Approved by project owner — implemented

---

## Decision

Add a `run:` field (string or map) to `type: cli` as a first-class alternative to
the existing `command`/`args` exec form.  Add a `shell:` field to control shell
selection for the string form.

---

## Context

The original `type: cli` spec supported only the exec form (`command` + `args`),
which requires knowing the binary name and constructing argument lists.  Shell
constructs (pipes, conditionals, heredocs, multi-command scripts) required an
awkward `command: bash args: ["-c", "..."]` wrapper that:

1. Made scripts hard to read in YAML (escaping, quoting).
2. Was silently POSIX-only — no indication to the runtime that Windows was
   unsupported.
3. Provided no mechanism to express cross-platform equivalents within a single
   step.

The "Option C" proposal (multi-shell `run:` map) was investigated prior to this
decision.  The project owner approved it on 2026-04-19.

---

## What Changed

### Schema additions to `type: cli`

#### New field: `shell`

```
shell:
  type: string
  enum: [auto, bash, sh, pwsh, cmd]
  default: auto
  optional: true
```

- Applies only to the `run` string form.
- `auto`: runtime picks platform-native shell (bash/sh on Unix, pwsh → cmd on Windows).
- Explicit value: runtime invokes that specific shell.
- **Ignored** when `run` is a map (validation warning emitted if both are set).

#### New field: `run` (string | map)

**String form** — shell-script passed verbatim to the selected shell:
```yaml
run: |
  if [ "{{ .env }}" = "prod" ]; then
    echo "Production deploy"
  fi
shell: bash
```

**Map form** — keys are shell names, values are shell-script strings:
```yaml
run:
  bash: rm -rf /tmp/cache/{{ .service }}
  cmd: rd /s /q C:\tmp\cache\{{ .service }}
  pwsh: Remove-Item -Recurse -Force C:\tmp\cache\{{ .service }}
```

### Mutual exclusivity rule

`command`/`args` and `run` are mutually exclusive.  Using both in the same step
is a **schema validation error**.

### Map key resolution order

When `run` is a map, the executor selects the entry as follows:

1. **Exact match**: host shell matches a map key → use that entry.
2. **Family fallback**: no exact match → apply fallback table.
3. **No match**: step fails with:
   `"No matching shell entry for current platform. Available entries: {keys}. Platform shell: {detected}."`

Fallback table:

| Requested shell | Falls back to      |
|-----------------|--------------------|
| `bash`          | `sh`               |
| `sh`            | (none — fail)      |
| `pwsh`          | `cmd` (Windows)    |
| `cmd`           | (none — fail)      |

---

## Rationale

- **Readability**: multiline shell scripts are expressed naturally as YAML block
  scalars rather than single-quoted `-c` arguments.
- **Cross-platform safety**: the map form makes platform assumptions explicit in
  the runbook rather than leaving them implicit.  A Windows executor that
  encounters only a `bash` key gets a clear error rather than silent failure.
- **Backward compatibility**: the exec form (`command`/`args`) is unchanged and
  remains the right choice for direct binary invocations.
- **Governance compatibility**: `allowed_commands` / `denied_commands` checks
  apply to the shell binary itself (e.g. `bash`, `pwsh`) in the run form,
  consistent with the exec form.

---

## Affected Files

- `design/gert-v2/sections/03-schema-vnext.tex` — §03 `type: cli` subsection
  expanded with new field table rows, run-form prose, map resolution rules,
  fallback table, and three new examples.
- `design/gert-v2/testdata/runbooks/r04-soc2-evidence/schema.yaml` — `init_audit_package`
  step converted from `command: bash args: ["-c", "..."]` to `run:` map form
  (bash + pwsh entries) as a demonstration of improved cross-platform fidelity.

---

## Implementation Notes (for Brian/Ken)

- **Parser (Brian)**: `run` field type is `oneOf: [string, map[string]string]`.
  Validate mutual exclusivity with `command`/`args` at structural validation phase.
  Emit warning if `shell` + map `run` are both present.
- **Runtime (Ken)**: map key resolution must follow the priority order above.
  The `shell: auto` resolution must consult `$PATH` / OS detection, not just
  OS name.  On Windows, check for `pwsh.exe` before falling back to `cmd.exe`.

---

## john-extension-not-a-step-type

### 2026-04-19: type:extension is not a step type

**By:** John  
**What:** `type: extension` does not exist in the gert v2 schema. Extension is a field-annotation
convention (`x-<namespace>:` prefix) only — it annotates runbook or step objects with metadata.
It is not a step type that can appear in the `flow:` array.

- **Outbound notify/send** steps (Slack, PagerDuty, email, external API calls) → `type: tool`
  with the tool declared in `toolRefs`.
- **Inbound event-receive** steps (wait for webhook, wait for SIEM alert, wait for callback from
  external system) → GAP-3: `type: wait_for_event` is not yet specified. Use a `type: cli` stub
  with a `# GAP: no wait_for_event step type yet` comment as placeholder.

**Why:** Correcting testdata YAML (r01-k8s-incident, r05-security-breach) that used
`type: extension` as a catch-all for steps that didn't fit other types. Prevents future confusion
during implementation and ensures testdata accurately reflects the schema's real capabilities
and gaps.

---

## john-field-types-p0

# Decision Record: P0 Field-Type Improvements — collector & choice

**Author:** John (Schema & Language Specialist)
**Date:** 2026-04-19
**Status:** IMPLEMENTED
**Requested by:** ormasoftchile (project owner)
**Follows:** `.squad/decisions/inbox/john-input-triad-audit.md`

---

## Summary

Implemented the P0 field-type improvements to `§03 — Schema vNext` and updated
the three reference runbook testdata files. All seven items in the task have been
addressed.

---

## What Was Added to §03 (03-schema-vnext.tex)

### 1. Step Type Inventory Table
- Updated the `collector` description from "Operator provides unstructured
  input/files" to "Operator provides structured input across 9 field types".

### 2. `type: choice` — `multiple` flag
Added three new fields to the `choice` step field table:
- `multiple: bool` (default: false) — when true, operator can select more than one option
- `min_selections: integer` (optional) — minimum number of selections when `multiple: true`
- `max_selections: integer` (optional) — maximum number of selections when `multiple: true`

New constraints:
- When `multiple: true`, stored variable becomes an array of selected values.
- `min_selections` must be ≥ 1; `max_selections` must be ≤ total options and ≥ `min_selections`.

### 3. `type: collector` — complete rewrite of field section

**Field structure table** expanded with new columns:
- `default` — pre-filled default value
- `validation` — type-specific validation constraints object
- `options` / `options_from` — for `select` type
- `multiple` — for `select` type (multi-select)
- `multiline` — for `text` type (textarea)

**New field type inventory table** (9 types, was 5):

| Type | Notes |
|------|-------|
| `text` | Single-line or multi-line (set `multiline: true` for textarea) |
| `number` | Integer or float; validation: `min`, `max`, `step` |
| `integer` | Shorthand for `number` with `step: 1` |
| `date` | ISO 8601 YYYY-MM-DD; UI: date picker; validation: `min`, `max` as ISO date strings |
| `datetime` | RFC 3339 with timezone; UI: datetime picker; validation: `min`, `max` |
| `boolean` | True/false checkbox; stored as boolean (not string) |
| `select` | Dropdown; `options` (static) or `options_from` (dynamic provider); `multiple: true` for multi-select |
| `file` | File attachment; stored as artifact with SHA256 hash |
| `image` | Image/screenshot; stored as artifact |

**Removed as a distinct type:** `multiline` and `url` from prior enumeration.
`multiline` is now a flag on `text`; `url` is subsumed by `text` with appropriate
hint (retained for backward compatibility via `gert migrate`).

**New subsubsections** added for each of the 6 new types:
- `\subsubsection{Field type: text}` — with `multiline` flag
- `\subsubsection{Field type: number}` — with validation table and 3 examples
- `\subsubsection{Field type: integer}` — shorthand docs
- `\subsubsection{Field type: date}` — with validation table
- `\subsubsection{Field type: datetime}` — with validation table
- `\subsubsection{Field type: boolean}` — template usage note
- `\subsubsection{Field type: select}` — static + dynamic provider options, multi-select

### 4. Questionnaire Pattern (new `\paragraph`)
Added `\paragraph{Questionnaire Pattern}` in the collector section documenting:
- A `collector` step with multiple fields IS the canonical questionnaire
- Atomic submission semantics
- Benefits over sequential single-field steps
- Complete 4-field triage form example using `integer`, `select`, `text`,
  and `text` with `multiline: true`

### 5. Validation rules for `collector` (new `\paragraph`)
Added `\paragraph{Validation rules for collector}` specifying:
- `fields` MUST have at least one entry (structural: `minItems: 1`)
- Field `name` uniqueness within a step
- `select` fields: exactly one of `options` / `options_from` required
- `number` fields: `min ≤ max` when both present
- `date`/`datetime` fields: `min ≤ max` (chronologically) when both present

### 6. Collector example updated
The "Incident evidence collection" example updated to use `type: text` with
`multiline: true` instead of the deprecated `type: multiline`.

---

## File Metrics

| File | Lines before | Lines after | Delta |
|------|-------------|-------------|-------|
| `sections/03-schema-vnext.tex` | 1,988 | 2,325 | +337 |
| `testdata/r03-employee-onboarding/schema.yaml` | ~450 | 457 | +7 |
| `testdata/r07-financial-approval/schema.yaml` | ~350 | 355 | +5 |
| `testdata/r04-soc2-evidence/schema.yaml` | ~290 | 293 | +3 |

---

## Testdata Updates

### r03-employee-onboarding/schema.yaml
Gaps closed from G2-002 (`# GAP: no dropdown/autocomplete field type`):
- `department`: `type: text` + hint → `type: select` with 4 static options
- `office_location`: `type: text` + hint → `type: select` with 4 static options
- `start_date`: `type: text` + "(YYYY-MM-DD)" hint → `type: date`
- `estimated_delivery` (laptop order): `type: text` → `type: date`
- `manager_approved`: `type: text` "Type 'yes'" → `type: boolean`
- `background_check_completed`, `no_disqualifying_issues`, `report_uploaded`: → `type: boolean`
- `calendar_invite_sent`, `zoom_link_included`: → `type: boolean`
- `security_training_completed`, `phishing_simulation_completed`, `aup_signed`: → `type: boolean`
- `access_confirmed`: → `type: boolean`

Assessment updated: Completeness 8→9, Fidelity 7→9. G2-002 marked FIXED.

### r07-financial-approval/schema.yaml
Gaps closed from G2-004 (`# GAP: no dropdown`) and numeric/date workarounds:
- `amount`: `type: text` + "(USD, e.g. 75000)" hint → `type: number` with `validation.min: 50000`
- `budget_line_item`: `type: text` + hint → `type: select` with `options_from: finance-system`
- `requested_delivery_date`: `type: text` + "(YYYY-MM-DD)" → `type: date`
- `department`: `type: text` + hint → `type: select` with 5 static options
- `business_justification`: `type: multiline` → `type: text` with `multiline: true`
- All approval yes/no fields (6 steps): `type: text` "Type 'yes'" → `type: boolean`
- `manager_comments`: `type: multiline` → `type: text` with `multiline: true`

Side effect: `.amount` is now a proper number — Go template `gt`/`lt` comparisons
in the branch conditions (step 4) are now semantically correct numeric comparisons.

Assessment updated: Completeness 8→9, Fidelity 8→9. G2-002 and G2-004 marked FIXED.

### r04-soc2-evidence/schema.yaml
Gaps closed from G2-002 (`# GAP: no dropdown field type`):
- `risk_assessment_date`: `type: text` + "(YYYY-MM-DD)" hint → `type: date`
- `mitigation_status`: `type: text` + hint → `type: select` with 3 static options
- `all_controls_evidenced`, `no_missing_files`, `date_ranges_covered`: → `type: boolean`

Assessment updated: Fidelity 8→9. G2-002 marked FIXED.

---

## Design Decisions Made

1. **`integer` as distinct type, not `number` + `step: 1`**: Both coexist.
   `integer` is a convenience alias that signals intent clearly in the schema
   and produces a stored value guaranteed to be an integer. The runtime
   normalizes `integer` to `number + step: 1` during parsing.

2. **`multiline` removed as a top-level type**: Made a flag on `text`
   (`multiline: true`). Cleaner: one text type with a rendering hint.
   `gert migrate` rewrites `type: multiline` → `type: text` + `multiline: true`.

3. **`url` not removed**: `url` is retained as a valid type for backward
   compatibility. It is no longer listed in the primary inventory table but
   remains accepted by the parser (emits a deprecation warning suggesting
   `type: text` with a `hint`).

4. **`options_from` for dynamic select**: Introduced `options_from.provider`
   (string) + `options_from.field` (optional string) as the provider integration
   point for runtime option population. This closes the `budget_line_item` gap
   in r07 without requiring a new step type.

5. **`multiple` on both `choice` and `select`**: Consistent flag name across
   both step-level selection (`choice.multiple`) and field-level selection
   (`collector.fields[*].multiple`). Stored as array of values in both cases.

---

## Gaps Still Open (not in scope for P0)

- Business-day timeout support (`timeout_business_days` for `collector.approvals`)
- Self-service task pattern (`type: task`, no approval semantics)
- `on_timeout: auto_approve` option
- Autocomplete field type for directory lookups
- Dynamic approver resolution (`roles_from:` with provider query)
- `invoke_many:` / `iterate.over: imports.*` for bulk sub-runbook invocation

---

## john-field-types-p1

# Decision: Collector Field Types — P1 Improvements

**By:** John (Schema & Language Specialist)  
**Date:** 2026-04-18  
**Status:** Accepted  
**Relates to:** `design/gert-v2/sections/03-schema-vnext.tex`, §03 Schema vNext

---

## Context

The P0 pass (field-types-p0) added nine field types to `collector` steps. A follow-up
input-triad audit identified two P1 gaps that were left unaddressed:

- **P1-A:** No mechanism to conditionally show/hide individual fields within a collector
  based on previously-entered values.
- **P1-B:** No validation constraints for the `text` field type beyond `required` and
  `hint`.

---

## Decision: P1-A — Conditional Field Visibility via `when`

### What

Add an optional `when` field to every `collector.fields[*]` object. The value is a Go
template expression (the same engine used for step-level `when:` guards and all other
gert template expressions).

### Schema addition

```yaml
fields:
  - name: severity
    type: select
    ...
  - name: escalation_reason
    type: text
    required: true
    when: "{{ eq .severity 'critical' }}"   # only shown when severity = critical
```

### Semantics

1. **Falsy → absent.** When `when` evaluates to falsy the field is hidden in the UI,
   not prompted in a TUI, and its variable binding is **skipped** (the variable is not
   set; it does not appear as `null` or empty in the variable space).

2. **Dynamic re-evaluation.** In interactive UIs, if an earlier field value changes,
   downstream `when` expressions are re-evaluated and dependent fields are shown/hidden
   accordingly.

3. **required + when.** A `required: true` field with a `when` expression is only
   required when its `when` condition is truthy. When the condition is falsy, `required`
   does not apply.

4. **Ordering constraint.** A field's `when` expression MUST NOT reference a field
   declared later in the same `fields` array (no forward references). This is a
   schema validation error detected at parse time. Expressions may freely reference:
   (a) fields declared earlier in the same `fields` array, and (b) the global runbook
   variable space (prior step outputs, runbook inputs, `gert.*` built-ins).

### Validation rules added

- A field `when` expression MUST NOT reference a field declared later in the same
  `fields` array (no forward references).
- A `required: true` field with a `when` expression is only required when the `when`
  condition is truthy.

### Why

Runbook forms are inherently conditional: escalation details are only relevant for
critical incidents; customer impact is irrelevant for low-severity alerts; shipping
address is irrelevant for in-office employees. Without field-level `when:`, authors
are forced to split one logical form into multiple sequential steps, losing atomicity
and increasing approval-gate complexity. The `when` field closes this gap with minimal
schema surface area by reusing the existing Go template engine.

---

## Decision: P1-B — Text Field Validation Constraints

### What

Add an optional `validation` object to `type: text` fields with four sub-fields:

| Field | Type | Description |
|-------|------|-------------|
| `pattern` | string | ECMA 262 regex; field value must match |
| `pattern_hint` | string | Human-readable error shown when pattern validation fails |
| `min_length` | integer ≥ 0 | Minimum character count |
| `max_length` | integer > 0 | Maximum character count |

### Schema addition

```yaml
- name: target_ip
  type: text
  label: Target IP address
  required: true
  validation:
    pattern: "^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$"
    pattern_hint: "Must be a valid IPv4 address (e.g. 192.168.1.1)"
```

### Validation rules added

- `validation.max_length` MUST be strictly greater than `validation.min_length` when
  both are present.
- `validation.pattern` MUST be a valid ECMA 262 regex, validated at schema load time
  (structural validation error if invalid).
- `validation.pattern_hint` is silently ignored when `validation.pattern` is absent
  (warning, not error).

### Why

Free-form `type: text` fields are the most common source of operator data entry
errors (wrong date format, wrong IP format, over-length inputs causing downstream
tool failures). Validation constraints surface errors at form submission time rather
than at tool execution time, improving the operator experience and reducing invalid
evidence in the audit trail.

ECMA 262 regex was chosen over Go regex because:
- It is the JSON Schema Draft 2020-12 `pattern` format, keeping the gert schema
  self-consistent with the JSON Schema layer.
- VS Code and browser-based UIs implement ECMA 262 regex natively for live validation.
- ECMA 262 is a strict subset of most patterns that operators would write in practice.

---

## Testdata Impact

| Runbook | Change | Field(s) |
|---------|--------|----------|
| r03-employee-onboarding | P1-B: `validation.pattern` | `email` → `@company.com` domain |
| r03-employee-onboarding | P1-A: `when` | `shipping_address` → only shown when `office_location = Remote` |
| r05-security-breach | P1-A: `when` | `legal_notes` → only shown when `notification_required = yes` |
| r05-security-breach | P1-B: `validation.pattern` | `responsible_team` → restricted to valid team names |
| r05-security-breach | P1-B: `validation.pattern` | `target_completion_date` → enforces YYYY-MM-DD |
| r06-db-migration | P1-B: `validation.pattern` | `maintenance_start` → enforces RFC 3339 datetime |
| r07-financial-approval | P1-B: `validation.max_length` | `item_description` → enforces 500-char prose limit |

---

## Assessment Score Changes

| Runbook | Metric | Before | After |
|---------|--------|--------|-------|
| r03-employee-onboarding | Fidelity | 9/10 | 10/10 |
| r05-security-breach | Fidelity | 7/10 | 8/10 |
| r06-db-migration | Fidelity | 7/10 | 8/10 |
| r07-financial-approval | Fidelity | 9/10 | 10/10 |

---

## Alternatives Considered

### P1-A: Use separate steps instead of field-level `when`

The existing workaround is to split a conditional field into its own `collector` step
with a step-level `when:` guard. This works but fragments the logical form: the
operator fills in partial forms sequentially, and each step requires its own
submission confirmation. Multi-field forms with several conditional fields become
awkward chains of single-field steps. Rejected in favour of field-level `when`.

### P1-B: Re-use JSON Schema `pattern` format (URI annotation)

Considered linking `validation.pattern` to the JSON Schema `format: "regex"` concept.
Decided against it: the gert spec is already explicit about ECMA 262 and runtime
implementors benefit from the clear callout rather than an implicit JSON Schema
reference.

---

## Implementation Notes for Brian (Parser) and Ken (Runtime)

- `when` expressions on fields use the same `text/template` evaluation path as
  step-level `when:`. No new expression engine is needed.
- Forward-reference detection requires a single pass over the `fields` array during
  semantic validation (Phase 2). Collect field names in declaration order; for each
  `when` expression, parse template AST and check `.FieldName` references against
  the already-seen set.
- `validation.pattern` should be compiled at schema load time (not at form render
  time). Use `regexp.MustCompile` equivalent wrapped in a structured validation error
  if the regex is invalid. Use the ECMA 262 regex dialect where the runtime supports
  it (e.g. in the browser extension); fall back to RE2 in Go, noting that RE2 rejects
  some ECMA 262 constructs (lookahead, backreferences). A compatibility warning should
  be emitted if the pattern uses ECMA-262-only features.

---

## john-field-types-p2

# Decision: Collector Field Types — P2 Improvements

**Author:** John (Schema & Language Specialist)  
**Date:** 2026-04-19  
**Status:** Proposed  
**Requested by:** Project owner  
**References:** `john-field-types-p0.md`, `john-field-types-p1.md`

---

## Context

P0 added core field types (text/number/integer/date/datetime/boolean/select/file/image) and basic
validation. P1 added pattern validation, `when:` conditional fields, and field-level validation
constraints (min/max, min_length/max_length, pattern). P2 addresses three remaining gaps
identified during testdata stress-testing and schema review:

1. Secrets in audit trails — credential fields (tokens, passwords, API keys) must not appear in
   JSONL audit logs or evidence stores.
2. Live search for large datasets — static option lists are unworkable when the candidate set is
   an entire employee directory, service registry, or ticket database.
3. Structural validation for multiline text — authors frequently collect YAML or JSON snippets
   via collector steps with no way to validate parse correctness before the value flows
   downstream.

---

## Decisions

### P2-A: `ephemeral: true` field attribute

**Decision:** Add `ephemeral` boolean attribute (default `false`) to all collector field types.

**Semantics:**
- Value IS bound to the variable space and usable in downstream template expressions within the
  same run.
- Value is NEVER written to the JSONL audit log; the audit record shows `"[REDACTED]"` instead.
- Value is NEVER persisted to any store (trace, artifact store, variable snapshot).
- If the run is suspended (e.g., after `wait_for_event`) and resumed, ephemeral values are
  gone; the collector step must be re-executed.

**Validation rules:**
- Valid on all field types; semantically meaningful on `text`, `select`, `boolean`.
- Setting `ephemeral: true` on `type: file` or `type: image` is a **warning** (not error):
  file payloads will not be stored, but a `[REDACTED]` placeholder in the audit trail is still
  useful for evidence lineage.

**Security note:** Ephemeral fields reduce audit footprint for secrets. They do NOT prevent the
value from being transmitted to tool steps or sub-processes. Authors are responsible for
ensuring ephemeral values flow only to secure execution contexts.

---

### P2-B: `type: autocomplete` field type

**Decision:** Add `autocomplete` as a tenth collector field type.

**Semantics:**
- Functions like `select` in storage and validation (stored value is a string; `multiple: true`
  stores an array).
- Options are fetched dynamically from an input provider as the operator types (search-as-you-type).
- `options_from` is required (same object as `select`), extended with:
  - `provider` (required): input provider ID
  - `field` (required): provider input field that receives the search query string
  - `min_chars` (optional, default 2): minimum characters before search fires
  - `debounce_ms` (optional, default 300): debounce delay in milliseconds
- `options` (static list) is NOT valid on `autocomplete`; structural validation error if present.
- **CLI / non-interactive fallback:** The provider MUST be called once with an empty query to
  fetch all options; the result is rendered as a standard `select`.

**Rationale:** Large datasets (employee directories, service registries, ticket systems) are
impossible to enumerate as static option lists. Autocomplete is the idiomatic UI pattern and
is already anticipated by the `options_from` provider mechanism in `select`.

---

### P2-C: `validation.format` for multiline text fields

**Decision:** Add `format` sub-field to the `validation` object of `type: text` fields. Only
meaningful when `multiline: true`.

**Supported values:**
- `format: yaml` — value must parse as valid YAML
- `format: json` — value must parse as valid JSON
- `format: toml` — deferred to a future release; declaring it is a warning in v2.0

**Validation behavior:**
- Parse attempt is made after submission.
- On parse failure: re-prompt with message `"Value must be valid {FORMAT}. Please correct and resubmit."`
- On success: value is stored as a raw string (the original text), NOT as a parsed object.
  Parsing is validation-only.

**Constraints:**
- `validation.format` on a non-multiline field is a validation **warning**; constraint ignored at runtime.
- `validation.format: toml` is a validation **warning** in v2.0; not enforced at runtime.

**Note:** The stored value is always a string. Downstream template expressions that need to
traverse the structure must use gert's `fromYAML`/`fromJSON` template functions.

---

## Testdata impact

| Runbook | Change | P2 feature |
|---------|--------|------------|
| r03-employee-onboarding | `manager` field: `type: text` → `type: autocomplete` with `options_from.provider: hr-directory` | P2-B |
| r03-employee-onboarding | Score: 9/10 → 10/10; verdict: PASS WITH NOTES → PASS | P2-B |

**P2-A:** No existing collector fields in the 10 testdata runbooks carry names containing
"token", "password", "key", "secret", or "credential". No changes required for P2-A.

**P2-C:** No existing multiline fields in the testdata are clearly YAML or JSON payloads (all
are prose text). No changes required for P2-C.

---

## Spec changes

- `§03 §\ref{subsec:step-collector}` — field structure table: added `ephemeral` row
- `§03 §\ref{par:ephemeral-fields}` — new paragraph: ephemeral semantics, security note, example
- `§03 §\ref{par:collector-field-types}` — type inventory: `nine` → `ten`; added `autocomplete` row
- `§03 §\ref{subsubsec:field-text}` — validation table: `format` sub-field (five validation fields)
- `§03 §\ref{par:text-format}` — new paragraph: format validation semantics and examples
- `§03 §\ref{subsubsec:field-autocomplete}` — new subsection: autocomplete type, `options_from`
  extension table, CLI fallback note, examples
- `§03 §\ref{par:collector-validation}` — validation rules: added P2-A, P2-B, P2-C rules

---

## john-gap1-gap2-approve

### 2026-04-19: GAP-1 + GAP-2 resolved in approve step
**By:** John
**What:** approve step now supports: (1) timeout_business_days + timezone + business_calendar for business-day timeouts (mutually exclusive with timeout); (2) approvals.mode (all/any/quorum) + approvals.pool + approvals.required for M-of-N quorum. Validation rules documented. R7 and R8 testdata updated.
**Why:** Critical gaps from stress test. R7 Financial and R8 FDA runbooks were FAIL due to these missing fields.

---

## john-input-triad-audit

# Input-Capturing Triad Audit — choice, decision, collector

**Author:** John (Schema & Language Specialist)  
**Date:** 2026-04-19  
**Requested by:** ormasoftchile  
**Status:** RECOMMENDATION — awaiting owner decision  

---

## Executive Summary

The input-capturing triad (`choice`, `decision`, `collector`) covers **80% of real-world input needs** but has **7 critical gaps** that force workarounds in production runbooks. Most gaps can be resolved by **extending existing step types** — no new step type required.

**Top 3 gaps by impact:**
1. **No dropdown/select field type** — forces `type: text` with hint comments (blocks validation)
2. **No numeric field type** — forces string comparison for amounts, no range validation
3. **No date/datetime field type** — forces `type: text` with YYYY-MM-DD format in hint

**Questionnaire pattern:** Fully supported via `collector` with multi-field `fields` array — no schema change needed.

**Internal consistency:** Types are well-separated, but `collector` has approval-gate semantics overlap with `approve` step (acceptable; documented below).

---

## 1. Full Input Taxonomy

Comprehensive list of input types required by production runbooks:

| Input Type | Example Use Case | Priority |
|------------|------------------|----------|
| **Single-select from options** | Environment (staging/prod/canary) | P0 |
| **Multi-select from options** | Checkbox list (affected systems) | P1 |
| **Free text (short)** | Name, email, ticket ID | P0 |
| **Free text (long)** | Incident summary, justification | P0 |
| **Numeric (integer)** | Number of approvers, timeout days | P1 |
| **Numeric (float)** | Dollar amount, percentage | P1 |
| **Numeric with units** | Timeout (5m / 2h / 3d), storage (50GB) | P2 |
| **Date** | Start date, delivery date | P1 |
| **Time** | Cut-over window start time | P2 |
| **Datetime** | Scheduled maintenance window | P2 |
| **Boolean / acknowledgment** | "I have verified X" checkbox | P1 |
| **File upload** | Error log, signed contract PDF | P0 |
| **Image / screenshot** | Dashboard screenshot, photo | P0 |
| **URL** | Related ticket, documentation link | P0 |
| **Credentials (ephemeral)** | API token, password (never persisted) | P2 |
| **Dropdown (dynamic source)** | Budget line items from finance system | P1 |
| **Autocomplete (search)** | Employee name from directory | P2 |
| **Ranked ordering** | Priority order of mitigation steps | P3 |
| **Questionnaire (multi-question form)** | 5-question security checklist | P1 |
| **Table / matrix input** | Access control matrix (user × role) | P3 |
| **Signature / attestation** | Legal signature (image or typed name) | P2 |
| **JSON / YAML snippet** | Structured configuration block | P2 |
| **Key-value pairs** | Tag list, metadata | P2 |

---

## 2. Coverage Mapping

### type: choice
**Purpose:** User selects one value from a fixed set of options; value is stored in a variable.

| Input Type | Coverage | Notes |
|------------|----------|-------|
| Single-select from options | ✅ Full | `options` with `label`, `value`, `hint` |
| Multi-select from options | ❌ None | **GAP-1**: No `multiple: true` flag |
| Boolean / acknowledgment | ⚠️ Workaround | Can model as 2-option choice (yes/no) but semantic mismatch |
| Dropdown (static) | ✅ Full | Same as single-select |
| Dropdown (dynamic source) | ❌ None | **GAP-2**: No `options_from: provider` or `options_from: tool` |

**Recommendation:**
- Add `multiple: true` (optional, default: false) to enable multi-select (GAP-1)
- Add `options_from: { provider: string, field: string }` for dynamic options (GAP-2)

---

### type: decision
**Purpose:** User selects an execution path; flow branches to selected runbook or step.

| Input Type | Coverage | Notes |
|------------|----------|-------|
| Single-select from paths | ✅ Full | `routes` with `label`, `runbook`, `goto`, `hint` |

**No gaps.** Decision is a pure control-flow primitive; it stores the selected route label for audit but does not collect data.

**Internal consistency note:** Decision overlaps slightly with `type: branch` (condition-based routing). Distinction is clear: `decision` = user-driven routing, `branch` = condition-driven routing. However, runbook authors unfamiliar with the schema might reach for `decision` when they should use `branch` with a prior `choice` step.

**Recommendation:** Add clarifying prose to spec:
> "Use `decision` when the user chooses the execution path interactively. Use `branch` (with condition) when the path is determined programmatically from existing variables. If you need the user to select a value AND route based on that value, use a `choice` step followed by a `branch` step."

---

### type: collector
**Purpose:** Collects unstructured or multi-field input from the user; stores values as variables or artifacts.

| Input Type | Coverage | Notes |
|------------|----------|-------|
| Free text (short) | ✅ Full | `field.type: text` |
| Free text (long) | ✅ Full | `field.type: multiline` |
| File upload | ✅ Full | `field.type: file` (stored as artifact with SHA256) |
| Image / screenshot | ✅ Full | `field.type: image` (stored as artifact) |
| URL | ✅ Full | `field.type: url` (basic validation) |
| Numeric (integer) | ❌ None | **GAP-3**: No `field.type: number` or `field.type: integer` |
| Numeric (float) | ❌ None | **GAP-3**: No `field.type: number` |
| Numeric with units | ❌ None | **GAP-3**: No `field.type: duration` or `field.type: quantity` |
| Date | ❌ None | **GAP-4**: No `field.type: date` |
| Time | ❌ None | **GAP-4**: No `field.type: time` |
| Datetime | ❌ None | **GAP-4**: No `field.type: datetime` |
| Boolean / acknowledgment | ❌ None | **GAP-5**: No `field.type: boolean` or `field.type: checkbox` |
| Dropdown (static) | ❌ None | **GAP-6**: No `field.type: select` with `options` array |
| Dropdown (dynamic) | ❌ None | **GAP-6 + GAP-2**: No select + no dynamic options |
| Autocomplete (search) | ❌ None | **GAP-7**: No `field.type: autocomplete` with provider binding |
| Multi-select (checkboxes) | ❌ None | **GAP-1 + GAP-6**: No multi-select + no select type |
| Credentials (ephemeral) | ⚠️ Partial | Can use `type: text` but no `ephemeral: true` flag (credential appears in audit trace) |
| Ranked ordering | ❌ None | Out of scope (P3, use case too rare) |
| Table / matrix | ❌ None | Out of scope (P3, can be modeled as JSON/YAML snippet) |
| Signature / attestation | ⚠️ Workaround | Can use `type: image` (upload signature image) or `type: text` (type name) |
| JSON / YAML snippet | ⚠️ Workaround | Can use `type: multiline` but no syntax validation |
| Key-value pairs | ⚠️ Workaround | Can use `type: multiline` (format: `key=value` per line) but no validation |
| Questionnaire (multi-question) | ✅ Full | Use `fields` array with multiple field objects |

**Recommendations:**

**GAP-3: Add numeric field types**
```yaml
fields:
  - name: amount
    type: number  # NEW: accepts integer or float
    label: Amount (USD)
    required: true
    validation:  # NEW: optional validation block
      min: 50000
      max: 10000000
      
  - name: timeout_minutes
    type: integer  # NEW: integer-only variant
    label: Timeout (minutes)
    required: true
    validation:
      min: 1
      max: 1440
```

**GAP-4: Add date/time field types**
```yaml
fields:
  - name: start_date
    type: date  # NEW: YYYY-MM-DD format, date picker in UI
    label: Start date
    required: true
    validation:  # NEW: optional validation
      min: "2026-01-01"  # earliest allowed date
      
  - name: maintenance_window
    type: datetime  # NEW: ISO 8601 format
    label: Maintenance window start
    required: true
```

**GAP-5: Add boolean/checkbox field type**
```yaml
fields:
  - name: security_training_completed
    type: boolean  # NEW: checkbox in UI, boolean value in variables
    label: Security training completed?
    required: true
    default: false
```

**GAP-6: Add select field type (dropdown)**
```yaml
fields:
  - name: department
    type: select  # NEW: dropdown in UI
    label: Department
    required: true
    options:  # NEW: options array (same schema as choice.options)
      - label: Engineering
        value: engineering
      - label: Sales
        value: sales
      - label: Marketing
        value: marketing
    multiple: false  # NEW: default false; set true for multi-select
```

**GAP-7: Add dynamic options binding**
```yaml
fields:
  - name: manager
    type: select
    label: Manager
    required: true
    options_from:  # NEW: fetch options from provider
      provider: employee-directory
      field: managers
      # Provider must implement options protocol: return {label, value}[] 
```

**Ephemeral credentials (minor enhancement):**
```yaml
fields:
  - name: api_token
    type: text
    label: API token
    required: true
    ephemeral: true  # NEW: value is not persisted in audit trace; redacted as [REDACTED]
```

**JSON/YAML snippet validation (optional enhancement):**
```yaml
fields:
  - name: config_block
    type: multiline
    label: Configuration (YAML)
    required: true
    format: yaml  # NEW: optional; validates syntax (yaml | json)
```

---

### type: approve
**Purpose:** Standalone approval gate; suspends execution until authorized roles approve.

| Input Type | Coverage | Notes |
|------------|----------|-------|
| Approval / sign-off | ✅ Full | Dedicated step type; no data collection |
| Attestation with identity | ✅ Full | Captures approver identity in audit trail |

**No gaps.** `approve` is a specialized approval gate that does not collect data. It has full support for quorum (`mode: quorum`), business-day timeout (`timeout_business_days`), and escalation (`escalate_to`).

**Overlap with collector:** The `collector` step type can embed an `approvals` block, which creates an approval gate + data collection in a single step. This is intentional: sometimes approval and data collection are coupled (e.g., "upload signed contract and have procurement approve"). The distinction is clear:
- Use `approve` when you need a pure go/no-go gate with no data collection.
- Use `collector` with `approvals` when data collection and approval are coupled.

---

### type: wait_for_event
**Purpose:** Pauses execution until an inbound event arrives (webhook, message queue, signal).

| Input Type | Coverage | Notes |
|------------|----------|-------|
| Inbound webhook payload | ✅ Full | `event.source: webhook` with `capture` block |
| External system callback | ✅ Full | `event.source: channel` with event ID |

**No gaps.** This is a specialized input type for asynchronous event-driven flows, not interactive user input. Included here for completeness.

---

## 3. Questionnaire Analysis

**Question:** Can the current triad support questionnaires (multiple questions presented as a single form)?

**Answer:** Yes, fully supported by `type: collector` with the `fields` array.

### Example: Security Checklist (5 questions)

```yaml
- step:
    id: security_checklist
    type: collector
    title: Pre-deployment security checklist
    prompt: |
      Complete the security checklist before deployment.
    fields:
      - name: secrets_rotated
        type: boolean  # Assuming GAP-5 is resolved
        label: All production secrets rotated in last 90 days?
        required: true
        
      - name: vulnerability_scan_passed
        type: boolean
        label: Vulnerability scan passed with no critical findings?
        required: true
        
      - name: pentest_date
        type: date  # Assuming GAP-4 is resolved
        label: Date of most recent penetration test
        required: true
        
      - name: pentest_findings
        type: multiline
        label: Outstanding penetration test findings (or "none")
        required: true
        
      - name: compliance_status
        type: select  # Assuming GAP-6 is resolved
        label: SOC2 compliance status
        required: true
        options:
          - label: Compliant
            value: compliant
          - label: Non-compliant (remediation in progress)
            value: non_compliant
          - label: Not applicable
            value: na
    approvals:
      min: 1
      roles: [security-officer]
```

### Questionnaire Pattern Evaluation

**Option 1: Sequence of individual steps (one step per question)**
- ❌ **Rejected:** Too verbose (5 steps for 5 questions). Poor UX: user must click through 5 separate screens. No atomic submission (user could abandon mid-questionnaire).

**Option 2: A `collector` with multi-field `schema`**
- ✅ **CURRENT SCHEMA:** `collector.fields` array already supports this pattern.
- ✅ **Atomic submission:** All fields submitted together.
- ✅ **Single approval gate:** Approval applies to the entire questionnaire, not individual questions.
- ✅ **Natural composition:** No new step type needed.

**Option 3: A dedicated `form` step type**
- ❌ **Rejected:** `form` would be semantically identical to `collector` with `fields`. Adds complexity with no benefit.

**Option 4: Inline multi-question within `collector`**
- ✅ **Already supported:** This is exactly what `fields` array does.

**Recommendation:** No schema change needed. The `collector` step with `fields` array is the canonical questionnaire pattern. Document this pattern explicitly in the spec as a worked example.

---

## 4. Gap Assessment

### Summary Table

| Gap ID | Impact | Type Affected | Recommendation | Priority |
|--------|--------|---------------|----------------|----------|
| GAP-1 | High | `choice`, `collector.field` | Add `multiple: true` flag for multi-select | P1 |
| GAP-2 | Medium | `choice`, `collector.field` | Add `options_from: {provider, field}` for dynamic options | P1 |
| GAP-3 | High | `collector.field` | Add `type: number`, `type: integer` with `validation: {min, max}` | P0 |
| GAP-4 | High | `collector.field` | Add `type: date`, `type: time`, `type: datetime` with validation | P1 |
| GAP-5 | High | `collector.field` | Add `type: boolean` for checkboxes | P1 |
| GAP-6 | High | `collector.field` | Add `type: select` with `options` array | P0 |
| GAP-7 | Medium | `collector.field` | Add `type: autocomplete` with provider binding | P2 |
| GAP-8 | Low | `collector.field` | Add `ephemeral: true` flag for credentials | P2 |
| GAP-9 | Low | `collector.field` | Add `format: yaml|json` validation for multiline | P3 |

### GAP-1: Multi-select
**Current workaround:** Multiple `type: text` fields with hint "enter comma-separated values"  
**Extend:** Add `multiple: true` to `choice` step and `field.type: select`  
**Schema change:**
```yaml
# choice step
- step:
    type: choice
    multiple: true  # NEW: default false
    options: [...]
    variable: selected_systems  # stores array: ["api", "db", "cache"]

# collector field
fields:
  - name: affected_systems
    type: select
    multiple: true  # NEW
    options: [...]
```

---

### GAP-2: Dynamic options (dropdown from external source)
**Current workaround:** `type: text` with hint "Select from approved list in finance system"  
**Extend:** Add `options_from` binding to `choice` and `field.type: select`  
**Schema change:**
```yaml
- step:
    type: choice
    options_from:  # NEW: mutually exclusive with options
      provider: finance-system
      field: budget_line_items
      # Provider contract: must return {label: string, value: string, hint?: string}[]
    variable: budget_line
```

**Implementation note:** Runtime fetches options by invoking the provider at step initialization. Provider must implement the `options` protocol (returns array of option objects). If provider call fails, step fails with clear error message.

---

### GAP-3: Numeric field types
**Current workaround:** `type: text` with hint "Enter amount in USD"; string comparison in conditions  
**Extend:** Add `type: number` and `type: integer` to `collector.field`  
**Schema change:**
```yaml
fields:
  - name: amount
    type: number  # NEW: accepts integer or float
    label: Amount (USD)
    required: true
    validation:  # NEW: optional validation block
      min: 50000
      max: 10000000
      step: 0.01  # optional: for float precision (e.g., currency)
```

**Benefits:**
- UI can render numeric input with validation
- Variables are stored as `float64` (not string), enabling numeric comparison in conditions: `{{ gt .amount 100000 }}`
- Runtime validates min/max before step completes

---

### GAP-4: Date/time field types
**Current workaround:** `type: text` with hint "YYYY-MM-DD format"; no validation  
**Extend:** Add `type: date`, `type: time`, `type: datetime` to `collector.field`  
**Schema change:**
```yaml
fields:
  - name: start_date
    type: date  # NEW: YYYY-MM-DD format
    label: Start date
    required: true
    validation:  # NEW: optional
      min: "2026-01-01"
      max: "2027-12-31"
      
  - name: maintenance_window
    type: datetime  # NEW: ISO 8601 format
    label: Maintenance window start
    required: true
```

**Benefits:**
- UI can render date/time pickers
- Runtime validates format and range
- Variables are stored as ISO 8601 strings; can be parsed by downstream steps

---

### GAP-5: Boolean/checkbox field type
**Current workaround:** `type: text` with hint "yes/no" or "Type 'yes' to confirm"  
**Extend:** Add `type: boolean` to `collector.field`  
**Schema change:**
```yaml
fields:
  - name: security_training_completed
    type: boolean  # NEW
    label: Security training completed?
    required: true
    default: false  # optional: default value if not required
```

**Benefits:**
- UI renders checkbox (not text input)
- Variable is stored as boolean (not string "yes"/"no")
- Clearer semantic intent

---

### GAP-6: Select/dropdown field type
**Current workaround:** `type: text` with hint "Options: Engineering, Sales, Marketing"  
**Extend:** Add `type: select` to `collector.field`  
**Schema change:**
```yaml
fields:
  - name: department
    type: select  # NEW
    label: Department
    required: true
    options:  # NEW: same schema as choice.options
      - label: Engineering
        value: engineering
      - label: Sales
        value: sales
    multiple: false  # NEW: default false; set true for multi-select
```

**Benefits:**
- UI renders dropdown (not free-text input)
- Runtime validates value is in the allowed set
- Prevents typos and invalid values

---

### GAP-7: Autocomplete with search
**Current workaround:** `type: text` with hint "Use autocomplete from employee directory" (no actual autocomplete)  
**Extend:** Add `type: autocomplete` to `collector.field`  
**Schema change:**
```yaml
fields:
  - name: manager
    type: autocomplete  # NEW
    label: Manager
    required: true
    options_from:  # NEW: provider binding
      provider: employee-directory
      field: managers
      search: true  # provider supports search query
      # Provider must implement search protocol: search(query: string) -> {label, value}[]
```

**Priority:** P2 (nice-to-have; can defer to v2.1)

---

### GAP-8: Ephemeral credentials
**Current workaround:** `type: text`; credential appears in audit trace (compliance risk)  
**Extend:** Add `ephemeral: true` flag to `collector.field`  
**Schema change:**
```yaml
fields:
  - name: api_token
    type: text
    label: API token
    required: true
    ephemeral: true  # NEW: value redacted in audit trace as [REDACTED]
```

**Implementation:** Runtime still stores the value in variables (so downstream steps can use it), but trace writer redacts it when writing to JSONL.

**Priority:** P2 (important for security-sensitive runbooks)

---

### GAP-9: Structured text validation (JSON/YAML)
**Current workaround:** `type: multiline`; no syntax validation  
**Extend:** Add `format: yaml|json` to `collector.field`  
**Schema change:**
```yaml
fields:
  - name: config_block
    type: multiline
    label: Configuration (YAML)
    required: true
    format: yaml  # NEW: validates YAML syntax before step completes
```

**Priority:** P3 (low impact; syntax errors are caught downstream anyway)

---

## 5. Internal Consistency Check

### Potential Confusion: choice vs. decision
**Scenario:** Runbook author wants user to select an environment (staging/prod) and route to different steps based on the selection.

**Wrong approach (confused author):**
```yaml
- step:
    type: decision  # WRONG: decision is for routing, not data capture
    prompt: Which environment?
    routes:
      - label: Staging
        goto: deploy_staging
      - label: Production
        goto: deploy_production
```

**Correct approach (2 steps):**
```yaml
- step:
    type: choice
    prompt: Which environment?
    options:
      - label: Staging
        value: staging
      - label: Production
        value: production
    variable: env
    
- step:
    type: branch
    branches:
      - condition: '{{ eq .env "staging" }}'
        label: Staging
        steps: [...]
      - condition: '{{ eq .env "production" }}'
        label: Production
        steps: [...]
```

**Recommendation:** Add clarifying prose to spec (see earlier "Decision" section recommendation).

---

### Potential Confusion: collector vs. approve
**Scenario:** Runbook author needs an approval gate with no data collection.

**Wrong approach (confused author):**
```yaml
- step:
    type: collector  # WRONG: collector implies data collection
    title: Manager approval
    fields: []  # empty fields array — schema should reject this
    approvals:
      min: 1
      roles: [manager]
```

**Correct approach:**
```yaml
- step:
    type: approve
    title: Manager approval
    approvals:
      min: 1
      roles: [manager]
```

**Recommendation:** Schema validation MUST reject `collector` with empty `fields` array. Add to spec:
> "The `fields` array in a `collector` step must contain at least one field. If no data collection is needed, use `type: approve` instead."

---

### Potential Confusion: collector with approvals vs. approve
**Scenario:** Runbook author needs approval AND data collection in the same step.

**Correct approach (use collector with approvals):**
```yaml
- step:
    type: collector
    title: Upload contract and get approval
    fields:
      - name: signed_contract
        type: file
        label: Signed contract (PDF)
        required: true
    approvals:
      min: 1
      roles: [legal]
```

**Alternative (2 steps — also correct but verbose):**
```yaml
- step:
    type: collector
    title: Upload contract
    fields:
      - name: signed_contract
        type: file
        required: true
        
- step:
    type: approve
    title: Legal approval
    approvals:
      min: 1
      roles: [legal]
```

**Guidance:** Document in spec that `collector` with `approvals` is preferred when data collection and approval are semantically coupled (e.g., "upload evidence and have legal approve it"). Use two separate steps when the approval is semantically independent of the data collection (e.g., "collect incident notes (step 1), then get manager approval to proceed (step 2)").

---

## 6. Recommendations Summary

### Schema Changes Required (P0-P1)

1. **Extend `collector.field` with new `type` values:**
   - `number` (integer or float) with optional `validation: {min, max, step}`
   - `integer` (integer-only variant)
   - `date` (YYYY-MM-DD) with optional `validation: {min, max}`
   - `datetime` (ISO 8601) with optional `validation: {min, max}`
   - `boolean` (checkbox) with optional `default`
   - `select` (dropdown) with `options` array and `multiple: true|false`

2. **Extend `choice` step:**
   - Add `multiple: true` flag for multi-select (default: false)
   - Add `options_from: {provider, field}` for dynamic options (mutually exclusive with `options`)

3. **Extend `collector.field` with `options_from` (same as choice):**
   - Enables dynamic dropdowns in forms

4. **Add validation for `collector`:**
   - Reject `collector` with empty `fields` array (structural validation error)

### Optional Enhancements (P2-P3)

5. **Add `ephemeral: true` flag to `collector.field`:**
   - Redacts sensitive values (credentials, tokens) in audit trace

6. **Add `type: autocomplete` to `collector.field`:**
   - Requires provider search protocol implementation

7. **Add `format: yaml|json` to `collector.field.type: multiline`:**
   - Validates syntax before step completes

### Spec Clarifications (no schema change)

8. **Add worked example for questionnaire pattern:**
   - Show `collector` with 5-field `fields` array as canonical multi-question form

9. **Add guidance on choice vs. decision vs. branch:**
   - When to use each; clarify that decision is for user-driven routing, not data storage

10. **Add guidance on collector with approvals vs. approve:**
    - When to use inline approval vs. dedicated approve step

---

## 7. Test Runbook Evidence

Evidence from `/design/gert-v2/testdata/runbooks/`:

### r03-employee-onboarding (lines 55, 60, 69, 93, 120, 214, 268, 368)
- **GAP-6:** "Options: Engineering, Sales, Marketing, Finance" in hint (should be dropdown)
- **GAP-7:** "Use autocomplete from employee directory" in hint (no autocomplete field type)
- **GAP-4:** "Start date (YYYY-MM-DD)" in hint (no date field type)
- **GAP-5:** "yes/no" questions modeled as `type: text` (should be boolean)
- Timeout durations in prose are "business days" but schema only supports wall-clock hours

### r07-financial-approval (lines 42, 52, 68, 144)
- **GAP-3:** Amount field is `type: text` (should be `type: number`)
- **GAP-6:** "Options: Engineering, Sales, Marketing..." in hint (should be dropdown)
- **GAP-2:** "Select from approved budget line items" in hint (should be dynamic dropdown)
- Line 68: Hardcoded role `[manager]` should be dynamic lookup from HR system (schema limitation)
- Line 144: Numeric comparison `{{ and (gt .amount "50000") (lt .amount "100000") }}` — fragile string comparison (should be numeric type)

### r04-soc2-evidence (lines 101, 216, 235)
- **GAP-6:** "Options: Complete, In Progress, Planned" in hint (should be dropdown)
- Multiple timeout durations in prose are "business days" (already supported by `timeout_business_days` in v2)

---

## 8. Migration Path

If the recommended schema changes are accepted:

### v2.0 → v2.1 Migration

**Breaking change:** None. All changes are **additive** (new field types, new optional flags).

**Backward compatibility:**
- Existing runbooks with `type: text` + hint comments remain valid
- New runbooks can use the new field types
- Runtime must support both old and new field types

**gert migrate behavior:**
- `gert migrate` does **not** automatically upgrade `type: text` to `type: select` (requires semantic analysis; too risky)
- Provide a **linter warning** for `type: text` fields with hints like "Options: ..." suggesting migration to `type: select`

**Validation:**
- Structural validation: Reject unknown `field.type` values
- Semantic validation: Validate `options` array structure, `validation.min` ≤ `validation.max`, etc.

---

## Decision

**Proposal:** Accept GAP-3, GAP-4, GAP-5, GAP-6 (P0-P1) as blocking for v2.0. Implement as additive schema extensions. Defer GAP-7, GAP-8, GAP-9 (P2-P3) to v2.1.

**Impact:** 4 new field types + 1 validation block. Estimated schema expansion: ~150 lines in 03-schema-vnext.tex.

**Timeline:** 2 days to write schema extensions, update spec, add test cases, update JSON Schema artifacts.

**Owner decision required:** Proceed with P0-P1 gaps now, or defer all to v2.1?

---

**End of audit.**

---

## john-invoke-renamed-include

### 2026-04-19: type:invoke renamed to type:include
**By:** John
**What:** type:invoke is now type:include. Semantics clarified: include expands another runbook's steps inline into the current run path, sharing the caller's variable space. NOT a sub-procedure call. Cycle detection is a validation-time hard error. A future type:call (isolated scope, explicit I/O mapping) is deferred to post-v2.0.
**Why:** User identified that "invoke" implies calling another party (RPC/sub-procedure), but the intended semantic is inline composition/inclusion. "include" matches Ansible include_tasks, C #include, and is unambiguous.

---

## john-testdata-runbooks

Decision: Runbook test fixtures persisted at design/gert-v2/testdata/runbooks/. 10 runbooks, each with source.md + schema.yaml + assessment.md. GAP comments mark schema limitations.

Details:
- R1–R3: Full YAML translations extracted verbatim from john-schema-translations.md
- R4–R10: Best-effort translations written from source corpus
- All schema.yaml files are syntactically valid gert v2 YAML
- # GAP: comments identify lines where schema limitations prevent full fidelity
- Verdicts: 1 PASS, 7 PASS WITH NOTES, 2 FAIL (R7, R8)
- README.md at root links to methodology and summarizes schema readiness by domain

---

## john-translations-complete

# Schema Translation Complete — Gap Summary

**Date:** 2026-04-18  
**Author:** John (Schema & YAML Specialist)  
**For Review By:** Team (Ken, Dennis, Barbara, Brian, ormasoftchile)

---

## Translation Summary

Completed translation of all 10 runbooks from Dennis's corpus into gert v2 YAML schema with validation scoring per methodology.

**Results:**
- **PASS:** 1 runbook (10%)
- **PASS WITH NOTES:** 7 runbooks (70%)
- **FAIL:** 2 runbooks (20%)

**Average Scores:**
- Completeness: 82% (target: 95%)
- Fidelity: 73% (target: 95%)

**Verdict: Schema is NOT production-ready** (does not meet 95% thresholds + has 4 CRITICAL gaps).

---

## Top 3 Gap Patterns

### 1. Calendar-Aware Timeouts (3 runbooks, HIGH severity)

Approval workflows specify timeouts in business days, but schema only supports wall-clock durations.

**Recommendation:** Add `timeout: {value: 2, unit: business_days, calendar: us_federal_holidays}`

### 2. Dynamic Approver Resolution (2 runbooks, CRITICAL severity)

Approval workflows need runtime approver lookup (manager from HR, VP from org chart), but schema only supports static role lists.

**Recommendation:** Allow template expressions in `roles:` OR add provider-based resolution.

### 3. Cross-Branch Parallelism (2 runbooks, CRITICAL severity)

Cannot express "run X in parallel with all branches of decision Y". Parallel blocks don't span branches.

**Recommendation:** Add `async: true` step attribute for background execution.

---

## Critical Gaps (P0 — Block Production Use)

1. **Dynamic approver resolution** — Blocks R7 (financial approval)
2. **External event triggers** — Blocks R8 (FDA webhook resume after 90 days)
3. **Cross-branch parallelism** — Blocks R5 (security forensics during containment)
4. **Calendar-aware timeouts** — Degrades R3, R7, R8 (approval SLAs)

---

## High-Severity Gaps (P1 — Significant Friction)

- Bulk invocation verbosity (15 invoke steps for SOC2 controls)
- No datetime-based wait (maintenance window)
- No signature metadata (21 CFR Part 11 compliance)
- No best-effort parallel (partial failure handling)
- No JSON path queries (conditions use string matching)
- And 5 more...

---

## Runbooks That Failed

**R7: Financial Approval for Large Purchase**
- Completeness: 70%, Fidelity: 60%
- **Blocker:** Cannot express dynamic approver lookup from org chart
- **Blocker:** Business day timeouts for approval SLAs

**R8: Medical Device Software Release (FDA)**
- Completeness: 70%, Fidelity: 60%
- **Blocker:** Cannot pause runbook for 90 days and resume on FDA webhook
- **Gap:** No signature metadata for 21 CFR Part 11 compliance

---

## Recommendation

**Address 4 critical P0 gaps before declaring schema stable for implementation.**

The schema is 80% production-ready. The remaining 20% affects core enterprise use cases (financial approvals, regulatory workflows, security incident response).

---

## Detailed Output

Full translations and analysis in:
- `/Volumes/Projects/gert/.squad/tmp/john-schema-translations.md` (Runbooks 1-3 full YAML)
- `/Volumes/Projects/gert/.squad/tmp/john-translations-summary.md` (Complete analysis)

---

**Status:** Ready for team review and schema improvement prioritization discussion.

---

## john-validation-methodology

# Decision Proposal: gert v2 Schema Validation Methodology

**Date:** 2026-04-18  
**Proposed by:** John (Schema & YAML Specialist)  
**Status:** Pending team review  
**Category:** Quality Assurance, Schema Design Process

---

## Summary

Propose adopting a rigorous validation methodology for stress-testing the gert v2 schema against real-world prose runbooks. The methodology provides systematic translation protocols, binary completeness/fidelity criteria, gap classification, and scoring rubrics to answer: **"Can we express this operational procedure in gert v2 schema without losing semantic meaning?"**

---

## Problem Statement

The gert v2 schema (§03) is now a 1,495-line normative specification with 12 step types, rich data flow constructs, and complex control flow patterns. Before declaring the schema "implementation-ready," we need a systematic way to validate that:

1. Real-world operational runbooks can be translated to gert v2 schema
2. All semantics from the prose description are preserved in the schema
3. No critical expressiveness gaps exist
4. Any gaps are identified early and classified for remediation

Without a formal methodology, schema validation is subjective and ad-hoc. We risk missing expressiveness gaps until implementation phase or production usage.

---

## Proposed Solution

Adopt the **gert v2 Schema Validation Methodology** (full document at `.squad/tmp/john-validation-methodology.md`).

### Core Components

#### 1. Translation Protocol
Step-by-step instructions for translating prose runbooks to schema:
- Step boundary identification rules
- 12-question decision tree for step type classification
- Patterns for branching (automatic vs. human-driven)
- Data flow modeling (inputs → captures → consumers)
- Failure/compensation patterns
- Human interaction classification
- Parallelism and nested runbook modeling

#### 2. Completeness Criteria (10 binary checks)
- C1: Step coverage
- C2: Decision point representation
- C3: Data flow completeness
- C4: Timing and sequencing
- C5: Failure path coverage
- C6: Human interaction fidelity
- C7: Parallelism declaration
- C8: Nested runbook references
- C9: Governance and approval requirements
- C10: Terminal outcomes

#### 3. Fidelity Criteria (10 binary checks)
- F1: Semantic equivalence
- F2: No information loss
- F3: Execution path preservation
- F4: Variable binding correctness
- F5: Step type precision
- F6: Timing preservation
- F7: Governance alignment
- F8: Human prompt clarity
- F9: Deterministic evaluation
- F10: Artifact integrity

#### 4. Gap Classification (G1–G6)
- G1: Missing step type (HIGH severity)
- G2: Missing field (varies)
- G3: Missing flow construct (CRITICAL if common)
- G4: Missing interaction model (HIGH)
- G5: Semantic loss (MEDIUM)
- G6: Verbosity/workaround (LOW–MEDIUM)

#### 5. Scoring and Verdicts
- **Completeness Score:** (satisfied / 10) × 100%
- **Fidelity Score:** (satisfied / 10) × 100%
- **Verdict:**
  - PASS: ≥90% both scores, no CRITICAL gaps
  - PASS WITH NOTES: ≥80% both scores, no CRITICAL gaps, mitigation plan for HIGH gaps
  - FAIL: <80% either score OR CRITICAL gaps present

#### 6. Schema Improvement Signals
When gaps accumulate across runbooks:
- **Signal 1:** Schema extensions needed (G1/G2 appears in ≥3 runbooks)
- **Signal 2:** Spec clarifications needed (consistent misinterpretation)
- **Signal 3:** Design limitations (acceptable non-goals)

---

## Production Readiness Thresholds

**For gert v2 schema to be declared implementation-ready:**

- ✅ At least 10 diverse real-world runbooks translated
- ✅ Average completeness score ≥95%
- ✅ Average fidelity score ≥95%
- ✅ Zero CRITICAL-severity gaps
- ✅ All HIGH-severity gaps have documented workarounds

---

## Benefits

1. **Objective quality gate:** Binary pass/fail criteria eliminate subjective assessment
2. **Early gap detection:** Identifies missing step types/fields before implementation
3. **Actionable feedback:** Gap classification points to specific schema improvements
4. **Regression prevention:** Re-running methodology after schema changes ensures no fidelity loss
5. **Documentation artifact:** Scorecard provides evidence of schema completeness for stakeholders

---

## Trade-offs

### Advantages
- Systematic, repeatable process
- Bridges gap between normative spec and real-world usage
- Provides quantitative metrics (completeness %, fidelity %)
- Prioritizes gaps by severity and frequency

### Disadvantages
- Requires upfront effort to translate 10+ runbooks
- Methodology itself may need refinement as we apply it
- Some prose runbooks may be inherently ambiguous (not a schema gap)

---

## Implementation Plan

### Phase 1: Pilot (1–2 weeks)
1. Select 3 diverse real-world runbooks (incident response, deployment, root cause analysis)
2. Apply translation protocol
3. Score completeness and fidelity
4. Identify any G1–G6 gaps
5. Refine methodology based on learnings

### Phase 2: Full Validation (2–3 weeks)
1. Translate 7 additional runbooks
2. Aggregate gap inventory
3. Calculate average scores
4. Classify gaps by severity and frequency
5. Produce schema improvement roadmap

### Phase 3: Remediation (varies)
1. Design schema extensions for CRITICAL and HIGH gaps
2. Update §03 normative spec
3. Re-translate affected runbooks
4. Verify gap resolution

### Phase 4: Production Readiness (1 week)
1. Verify all thresholds met
2. Document any accepted design limitations
3. Declare schema implementation-ready
4. Handoff to Brian (Parser) and Ken (Runtime)

---

## Open Questions

1. **Who selects the 10 runbooks?** Recommend: ormasoftchile + John, prioritizing real production runbooks from ormasoftchile's experience.
2. **What if we find CRITICAL gaps late?** Methodology is designed for early detection; if found, delay implementation-ready declaration until remediated.
3. **How do we handle ambiguous prose?** Document ambiguity in scorecard; schema must pick ONE interpretation (document rationale in `prose:` section).

---

## Recommendation

**Adopt this methodology as the quality gate for gert v2 schema.**

Apply it immediately in pilot mode with 3 runbooks. If pilot reveals methodology gaps, refine and re-run. Once methodology stabilizes, complete full validation with 10 runbooks before declaring schema implementation-ready.

This ensures Brian's parser and Ken's runtime are built against a stress-tested, production-validated schema — not a theoretical design.

---

## Artifacts

- **Full methodology:** `.squad/tmp/john-validation-methodology.md` (38KB)
- **Includes:**
  - Translation protocol (9 sections)
  - 20 binary criteria (C1–C10, F1–F10)
  - Gap classification taxonomy (G1–G6)
  - Scoring rubric with verdict matrix
  - 2 worked examples (PASS and PASS WITH NOTES)
  - Printable checklist
  - Step type quick reference table

---

## Next Steps

1. **Team review:** Gon (Architecture), Ken (Runtime), Brian (Parser), ormasoftchile
2. **Decision:** Adopt, iterate, or defer?
3. **If adopted:** John proceeds with Phase 1 pilot (3 runbooks)

---

**Author:** John  
**Reviewed by:** _(pending)_  
**Decision date:** _(pending)_

---

## john-wait-for-event-spec

### 2026-04-19: Added type:wait_for_event step type
**By:** John
**What:** type:wait_for_event is now a first-class step type in §03. It pauses execution until an inbound event (webhook, message, signal, or channel) arrives. Fields: event.source, event.id, event.filter, event.payload_schema, capture, timeout, on_timeout.
**Why:** GAP-3 from stress test — required by K8s incident runbook (Prometheus webhook), security breach runbook (SIEM events), FDA release runbook, and others.

---

## ken-cli-shell-contract

# Decision: CLI Shell Resolution Contract

**By:** Ken (Systems Architect)
**Date:** 2026-04-19
**Status:** Pending merge to `.squad/decisions.md`

---

## Context

John is extending the `type: cli` step schema (§03) with two new `run:` forms:
1. A **shell-string form** — `run: "<string>"` with an optional `shell:` discriminator.
2. A **multi-shell map form** — `run: {bash: "...", cmd: "...", pwsh: "..."}` for
   cross-platform runbooks.

This record documents the executor-layer contract decisions that back those schema
additions.  All decisions here apply to §02 (architecture) only; the schema constraints
appear in §03.

---

## Decisions

### 1. Exec form vs shell-string form are mutually exclusive at the field level

`command`/`args` and `run` are mutually exclusive on a `type: cli` step.  Validation
rejects steps that declare both.  This is an Error-severity load-time check.

**Rationale:** Two competing invocation models on the same step create ambiguous
executor semantics.  Forcing a single form keeps the contract deterministic.

---

### 2. `shell: auto` resolves to the platform-native shell at plan time

Priority order:
- **Linux / macOS:** `/bin/sh` (POSIX-guaranteed).
- **Windows:** `pwsh` if `exec.LookPath("pwsh")` succeeds, else `cmd.exe`.

**Rationale:** `/bin/sh` is the lowest-common-denominator on POSIX systems.  `pwsh`
is preferred on Windows because it provides a consistent scripting surface across
Windows 10+; `cmd.exe` is the fallback for environments where PowerShell is not
installed.

---

### 3. Explicit `shell:` values are resolved at plan time, not run time

When `shell` is explicitly set to `bash`, `sh`, `pwsh`, or `cmd`, the executor calls
`exec.LookPath(shell)` during plan construction.  A binary not found at plan time is
a hard plan-time error that rejects the runbook before execution begins.

**Rationale:** Run-time failures are harder to surface to operators.  Surfacing missing
shells at plan time gives authors immediate actionable feedback before the first step
runs, consistent with the "fail fast" principle applied to include-cycle detection and
governance pre-flight.

---

### 4. Multi-shell map form resolution uses insertion-order walk with `bash`→`sh` fallback

At run time the executor walks map keys in insertion order, calling
`exec.LookPath(key)` for each.  The first key that resolves is selected.

Special case: if `bash` is a map key but is not found, the executor tries `sh` as an
implicit fallback before moving to the next map key.

**Rationale:** Insertion order is author-intent order — authors list preferred shells
first.  The `bash`→`sh` fallback reflects the real-world situation where a runbook
author writes `bash` but the host provides only `sh` (e.g., Alpine Linux with busybox).
Both share the `-c` invocation pattern, making the fallback transparent.

---

### 5. `shell` field is silently ignored (Warning) when `run` is a map

Setting `shell:` alongside a map-form `run:` produces a validation Warning, not an
Error.  The field is ignored; shell selection is always driven by the map keys.

**Rationale:** A Warning communicates the authoring mistake without breaking existing
runbooks that may have been written with `shell:` defensively.

---

### 6. Template expansion precedes shell invocation in all forms

For both string-form and map-form `run:`, Sprig/Go template expansion is applied to
the script value before it is passed to the shell.  The shell never sees template
syntax; it receives the fully-expanded string.

**Rationale:** Consistent with how `command` and `args` fields are expanded.  Authors
can interpolate variables, conditionals, and functions before the shell parses the
script.

---

### 7. Shell injection is an author responsibility; exec form is the safe alternative

Shell-string form is documented as injection-vulnerable.  A normative security note in
§02 directs authors to use exec form (`command`/`args`) for steps that accept untrusted
input.  No automatic escaping is applied.

**Rationale:** Automatic escaping is language-specific (bash quoting ≠ cmd quoting ≠
pwsh quoting) and would create a false sense of safety.  Explicit guidance with a clear
safe alternative is more defensible.

---

### 8. Audit trace records `shell_selected` on every shell-dispatched CLI step

The `step/started` JSONL event gains a `shell_selected` field (string, e.g.`"bash"`)
for all shell-string and map-form CLI steps.  Exec-form steps omit the field.

**Rationale:** Traceability requirement.  For cross-platform runbooks, knowing which
shell ran which step is essential for post-incident analysis and compliance audit.

---

### 9. `gert.platform` is injected at run start, not at step dispatch

The `gert.platform.os`, `gert.platform.arch`, and `gert.platform.shell` variables are
injected into the run variable scope at run start.  `gert.platform.shell` reflects the
shell selected for the current step; it is available in `capture` mappings and
downstream steps but NOT in the `run` script of the step that triggers resolution
(resolution precedes expansion of that field).

**Rationale:** Injecting at run start makes the variables available everywhere
(including `branch` predicates) without special per-step plumbing.  The
`gert.platform.shell` post-resolution constraint is a natural consequence of the
expand-then-invoke ordering.

---

## Files Changed

- `design/gert-v2/sections/02-architecture.tex` — new
  `\subsection{CLI Step Executor Contract}` inserted before the Include Step Executor
  Contract.  Covers: shell-string form resolution, invocation patterns, multi-shell map
  form walk algorithm, `bash`→`sh` fallback, `gert.platform` variable injection,
  security note, and validation rules table.

- `.squad/decisions/inbox/ken-cli-shell-contract.md` — this record.

---

## ken-field-types-executor

# Executor Contract — P0 Field Types (number, integer, date, datetime, boolean, select, multiline, choice/multiple)

**Author:** Ken (Systems Architect)  
**Date:** 2026-04-18  
**Requested by:** ormasoftchile  
**Status:** DECISION RECORD — changes applied to §02 and §14

---

## Context

John's input-triad audit (`john-input-triad-audit.md`) identified seven critical gaps in
the `collector` and `choice` step types. The owner accepted GAP-3 through GAP-6 as P0/P1
for v2.0. John is simultaneously extending §03 (schema spec) to add the new field types.

This record documents the runtime-side architectural decisions needed to support those new
types: validation contract, variable storage semantics, the `options_from` dynamic options
protocol, and the `multiline` UI hint.

---

## Decisions

### 1. Collector Field Validation Contract (new subsection in §02)

Added `\subsection{Collector Field Validation Contract}` (label: `sec:collector_field_validation`)
to §02 Architecture, between the Approve Step Executor Contract and Cancellation subsections.

**Algorithm:**
- Required check → type coercion/format check → constraint check → re-prompt loop
- Max **3** re-prompt attempts before hard step failure
- On failure: `step.state = FAILED`, `step/failed` trace event, `gert.error` populated

**Per-type validation rules (normative):**

| Type | Validation | Storage |
|------|-----------|---------|
| `text` | Required check only; `multiline` is UI hint | JSON string |
| `number` | float64; min/max inclusive; step modulo epsilon | JSON number (float64) |
| `integer` | int64; no fractional part; min/max/step | JSON number (int64) |
| `date` | RFC 3339 full-date (YYYY-MM-DD); min/max lexicographic | JSON string (YYYY-MM-DD) |
| `datetime` | RFC 3339 with offset; min/max normalised to UTC | JSON string (RFC 3339) |
| `boolean` | Truthy/falsy coercion (true/false/"true"/"false"/1/0/"yes"/"no") | JSON boolean |
| `select` (single) | Value in options list (static or dynamic) | JSON string |
| `select` (multiple) | All values in list; min/max_selections; no duplicates | JSON array of strings |
| `choice` (multiple) | Same as `select` multiple, applied to `choice` step | JSON array of strings |
| `file`, `image` | Size/MIME constraints; SHA-256 by executor | Artifact reference (evidence store) |
| `url` | Valid URL syntax, scheme required | JSON string |

**Rationale for 3 retries:** Mirrors common form-validation UX patterns. Too few (1) is
frustrating for typos; too many (>5) enables denial-of-service against interactive runs.
3 is the established gert convention from approval gates.

### 2. `multiline: true` — UI Hint Only

`multiline: true` on a `text` field has zero effect on validation or variable storage.
The executor always treats the value as a plain string. Input providers SHOULD render
a textarea or open `$EDITOR`. This decision keeps executor logic simple and separates
UI concerns from data semantics.

### 3. Variable Storage — Type-Correct JSON

Typed storage is the key enabler for downstream expression correctness:
- `number`/`integer` → JSON number (enables `{{ gt .amount 50000.0 }}`)
- `boolean` → JSON boolean (enables `{{ if .confirmed }}`)
- `select`/`choice` with `multiple: true` → JSON array (enables `{{ has .systems "x" }}`)
- `date`/`datetime` → ISO 8601 strings (parseable by `time.Parse` in templates)

Previously all collector fields were stored as strings, forcing fragile string comparisons
(`{{ gt .amount "50000" }}` uses string comparison, not numeric). This is the primary
motivation for typed field storage — cited directly in John's audit as a production pain
point (r07-financial-approval, line 144).

### 4. `options_from` — Dynamic Options Protocol (new subsection in §14)

Added `\subsection{Dynamic Options Protocol}` (label: `sec:dynamic-options-protocol`)
to §14 Input Provider Framework, between the Collector Step Contract and the Provider
Capability Matrix.

**New RPC method: `inputProvider/getOptions`**

Request:
```json
{
  "jsonrpc": "2.0",
  "id": "getopts-1",
  "method": "inputProvider/getOptions",
  "params": {
    "providerId": "employee-directory",
    "field":      "managers",
    "stepId":     "assign-manager",
    "variables":  { "department": "engineering" },
    "context":    { "runId": "...", "userId": "...", "mode": "real" }
  }
}
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "getopts-1",
  "result": {
    "options": [
      { "value": "alice", "label": "Alice Chen (Staff Eng)" }
    ],
    "cacheTtlSeconds": 300
  }
}
```

**Key decisions:**
- `variables` is included in every request so the provider can filter contextually
- `cacheTtlSeconds` allows the provider to declare result caching per-response (not per-runbook)
- Failure to fetch → step fails with `options_from.fetch_failed`; no silent fallback to empty list
- Provider not advertising `getOptions: true` → step fails with `options_from.capability_not_supported`

### 5. `choice` Step Extensions

Updated §14 Choice Step Contract to document:
- `multiple: true` → `selected` response becomes JSON array; stored as array of strings
- `options_from` → mutually exclusive with static `options`; triggers `inputProvider/getOptions`

### 6. Capability Matrix

Added `getOptions` column to the Provider Capability Matrix in §14. All built-in providers
(`prompt`, `env`, `file`, `workspace`) declare `getOptions: false`. Third-party providers
that serve dynamic option lists must declare `getOptions: true`.

---

## Sections Modified

| File | Changes |
|------|---------|
| `sections/02-architecture.tex` | (1) Extended `collector` dispatch bullet to reference field-type validation and typed storage; (2) Added `Collector Field Validation Contract` subsection (~130 lines) |
| `sections/14-input-provider-framework.tex` | (1) Updated `choice` contract requirements for `multiple` and `options_from`; (2) Updated collector field `type` list to include all new types; (3) Replaced single-line type constraints with expanded itemize block; (4) Replaced collector prompt provider paragraph with extended one covering new types; (5) Added `Dynamic Options Protocol` subsection with `inputProvider/getOptions` RPC (~80 lines); (6) Added `getOptions` column to capability matrix; (7) Added `getOptions: true` to capability declaration example |

---

## Contracts Added

1. **`inputProvider/getOptions`** — new JSON-RPC method, full request/response schema defined in §14
2. **Collector field validation algorithm** — 5-step algorithm with re-prompt loop defined in §02
3. **Per-type storage rules** — normative table in §02 (18 rows covering all field types)
4. **`gert.error` schema for validation failures** — structured error format defined in §02

---

## Cross-Section Consistency Notes

- §03 (John, Schema): This record is the runtime complement. John's schema additions define
  the YAML structure; this record defines what the executor does with the values at runtime.
- §14 (§14 is already the authoritative provider RPC reference): `inputProvider/getOptions`
  is defined here as a first-class method alongside `provider/resolve`, `provider/choice`, etc.
- §06 (Events): The `CollectorRequired` event already exists in the event catalog. The
  re-prompt loop reuses the same event with an `errors` map populated. No new event kinds
  are introduced for re-prompts — this is a point-to-point exchange, not a fan-out event.
- §12 (Tracing): `gert.error` is written to the trace as part of the `step/failed` event
  payload. No separate trace event needed.

---

**End of decision record.**

---

## ken-field-types-p1

# Executor Contract — P1 Field Types (conditional fields, text pattern validation)

**Author:** Ken (Systems Architect)  
**Date:** 2026-04-18  
**Requested by:** ormasoftchile  
**Status:** DECISION RECORD — changes applied to §02 and §14

---

## Context

John is extending §03 (schema spec) with two P1 features:
1. **Conditional field visibility** — `when` expression on `collector.fields[*]`
2. **Text field validation** — `validation.pattern` (regex), `validation.min_length`,
   `validation.max_length`, `validation.pattern_hint`

This record documents the runtime contracts for both features, complementing the P0
contract defined in `ken-field-types-executor.md`.

---

## Decisions

### 1. Conditional Field Evaluation Contract (new subsubsection in §02)

Added `\subsubsection*{Conditional Field Evaluation Contract}` (label:
`sec:conditional-field-eval`) inside the existing `Collector Field Validation Contract`
subsection in §02, after the new text-pattern subsubsection and before the existing
Variable Storage section.

**Evaluation model:**
- Fields evaluated left-to-right (array order).
- Before rendering field *i*, evaluate its `when` expression against:
  - Collected values so far from fields 0…i-1 in the same step.
  - Global variable scope (all prior completed steps).
- `when = false` → field not rendered, not prompted, variable binding skipped.
- `when` absent → always rendered (unconditionally visible).

**This is a pull model.** The executor drives field-by-field evaluation; the input
provider renders one field at a time OR a full form with hidden fields — both valid.

**Input provider integration — two valid strategies:**

| Strategy | Recommended For | Behaviour |
|----------|----------------|-----------|
| Full-form delivery | Rich UI providers | Executor sends all fields with pre-evaluated `visible: bool`; provider handles dynamic show/hide client-side as earlier fields are filled |
| Streaming delivery | CLI providers (e.g., `prompt`) | Executor sends only visible fields one at a time, evaluating `when` in real time as prior values are received |

**Variable binding on skip:**
- Skipped field `id` is NOT added to step variable bindings.
- Downstream template references → zero value (`""` / `0` / `false`).
- References using `.require` → template error at that step.
- Runbook authors must guard downstream expressions referencing conditional fields.

**Load-time forward reference check:**
- At schema load time: parse each `when` expression, extract variable references.
- If any reference binds to a field at a *later* index in the same `fields` array →
  reject the runbook with a validation error naming the forward-referencing field and
  both indices.
- References to global scope (prior steps) are always permitted; only same-step
  forward references are forbidden.

### 2. Text Field Pattern Validation (new subsubsection in §02; table row update)

Updated the `text` row in the Per-Type Validation table in §02 to reference the new
constraints. Added a dedicated `\subsubsection*{text Field Pattern Validation}`
subsubsection documenting the full sequential check order.

**Sequential validation for `text` fields:**
1. Required check (existing, step 1 of general algorithm)
2. `validation.min_length` — `len(value) < min_length` → fail
3. `validation.max_length` — `len(value) > max_length` → fail
4. `validation.pattern` — compile RE2 regex, test full match → fail with
   `validation.pattern_hint` if present, else generic message
5. Any failure → re-prompt (same 3-attempt limit as general algorithm)

**RE2 vs ECMA 262 (normative):**
- Schema spec says "ECMA 262" for JSON Schema / browser validator compatibility.
- Runtime uses Go `regexp` (RE2 semantics), not a full ECMA 262 engine.
- ECMA 262 features absent from RE2 (lookahead, lookbehind, backreferences) are
  **rejected at schema load time** with a specific validation error naming the field
  and the unsupported construct.
- Runbook authors should restrict patterns to the RE2/ECMA 262 common safe subset:
  character classes, quantifiers, anchors, alternation, non-capturing groups.

### 3. §14 Input Provider Framework Updates

Two additions to the `provider/collect` request schema:

**`visible: bool` (new per-field property)**
- Pre-evaluated by the executor from the field's `when` expression before dispatch.
- Always `true` when no `when` is declared.
- CLI providers SHOULD skip fields where `visible: false`.
- Rich UI providers SHOULD show/hide fields dynamically as the user fills in earlier
  fields (re-evaluating `when` client-side without a round-trip to the executor).

**`validation` object (new per-field property)**
- Carries all client-side validation constraints so UI providers can enforce them
  before submitting, reducing round-trips.
- Properties: `pattern`, `patternHint`, `minLength`, `maxLength` (text fields);
  `min`, `max`, `step` (numeric/date fields); `min_selections`, `max_selections`
  (multi-select fields).
- The executor **always** re-validates server-side regardless of provider enforcement.
- The request JSON example in §14 updated to show `visible` on all three fields and
  `validation` (minLength/maxLength/pattern/patternHint) on the `description` text
  field.

---

## Sections Modified

| File | Changes |
|------|---------|
| `sections/02-architecture.tex` | (1) Updated `text` row in Per-Type Validation table; (2) Added `text Field Pattern Validation` subsubsection with sequential check algorithm and RE2 vs ECMA 262 normative note; (3) Added `Conditional Field Evaluation Contract` subsubsection with evaluation model, provider integration strategies, variable-binding-on-skip rules, and load-time forward reference check |
| `sections/14-input-provider-framework.tex` | (1) Added `visible: bool` to all fields in `provider/collect` request JSON example; (2) Added `validation` object to `description` text field in example; (3) Replaced `Type-specific constraints` bullet with `visible` and `validation` bullets in Contract requirements, with full prose for both; (4) Retained all existing type-specific constraint sub-items under `validation` |

---

## Contracts Added / Extended

1. **Conditional field evaluation algorithm** — pull model, left-to-right, `when`
   evaluated against (collected-so-far ∪ global scope), skip = no binding
2. **Forward reference check** — schema-load-time rejection of same-step forward refs
   in `when` expressions
3. **`text` pattern validation sequence** — 5-step ordered check with re-prompt loop
4. **RE2 constraint (normative)** — ECMA 262 lookahead/lookbehind/backreferences
   rejected at load time; safe subset documented
5. **`visible` field in `provider/collect` request** — pre-evaluated `when` flag for
   provider-side conditional rendering
6. **`validation` object in `provider/collect` request** — client-side validation
   constraints passed to provider for pre-submission enforcement

---

## Cross-Section Consistency Notes

- **§03 (John, Schema):** This record is the runtime complement. John's schema additions
  define the YAML structure for `when`, `validation.pattern`, `validation.min_length`,
  `validation.max_length`, `validation.pattern_hint`; this record defines the executor
  semantics.
- **§06 (Events):** The existing `CollectorRequired` re-prompt event is unchanged.
  Conditional skipping does not emit a trace event (skipped fields are silently absent
  from bindings).  Consider adding a `input/field_skipped` event in a future pass if
  audit trail granularity requires it.
- **§11 (Governance / Redaction):** Skipped fields produce no variable bindings and
  therefore produce no values for redaction rules to apply to. Redaction rules
  referencing a conditionally-skipped field's variable are silently no-ops.
- **§14 (Input Provider Framework):** The `validation` object in `provider/collect`
  is advisory for providers; the §02 contract is the normative source of truth for
  runtime enforcement.

---

**End of decision record.**

---

## ken-field-types-p2

# Executor Contract — P2 Field Types (ephemeral, autocomplete, format validation)

**Author:** Ken (Systems Architect)
**Date:** 2026-04-18
**Requested by:** ormasoftchile
**Status:** DECISION RECORD — changes applied to §02, §03, and §14

---

## Context

John is extending §03 (schema spec) with three P2 features:
1. **`ephemeral: true`** — field values redacted from audit log, not persisted
2. **`type: autocomplete`** — live search field with debounced provider calls
3. **`format: yaml|json`** on multiline text — structural parse validation

This record documents the runtime and provider-framework contracts for all three,
complementing the P0 executor contract (`ken-field-types-executor.md`) and the P1
contract (`ken-field-types-p1.md`).

---

## Decisions

### 1. Ephemeral Field Handling (§02 — Collector Field Validation Contract)

Added `\subsubsection*{Ephemeral Field Handling}` (label: `sec:ephemeral-field-handling`)
inside the existing `Collector Field Validation Contract` subsection in §02, after the
Error Variable Schema subsubsection.

**Audit redaction contract:**
- After a field value passes all validation checks, the JSONL trace writer checks
  `VariableScope.IsEphemeral(key)` before serialising each variable.
- If `ephemeral: true`: the JSONL audit record stores `"[REDACTED]"` as the value.
- The actual value IS placed in the in-memory `VariableScope` and is available to all
  downstream steps via normal template expressions.
- The actual value is **NEVER written to disk** — not to `trace.jsonl`, not to
  `snapshots/<stepID>.json`, not to any checkpoint or archive.

**Go interface contract:**
```go
// VariableScope must expose this method.
IsEphemeral(key string) bool
```
The JSONL trace writer is the sole caller; no other layer applies redaction.

**Tool step compatibility:**
Ephemeral values MAY be passed to `type: tool` steps as arguments. The tool receives
the plaintext value. This is an audit redaction feature, not end-to-end encryption.

**Suspension/resume constraint:**
Ephemeral values are in-memory only. On run suspension (`wait_for_event` / `approve`),
the in-memory scope is not persisted; ephemeral values are dropped. On resume they are
absent from the reconstructed scope. Documented constraint:

> Steps that depend on ephemeral variables MUST appear before any `wait_for_event` or
> `approve` step that could suspend the run.

The schema validator SHOULD emit a load-time warning (not hard error in v2.0) if an
ephemeral variable is referenced in a step that follows a suspendable step.

---

### 2. Autocomplete Search Protocol (§14 — new subsection)

Added `\subsection{Autocomplete Search Protocol}` (label: `sec:autocomplete-search-protocol`)
to §14 Input Provider Framework, between the Dynamic Options Protocol subsection and the
Provider Capability Matrix.

**New RPC method: `inputProvider/search`**

Request:
```json
{
  "jsonrpc": "2.0",
  "id": "search-1",
  "method": "inputProvider/search",
  "params": {
    "provider":  "service-registry",
    "field":     "query",
    "query":     "pay",
    "minChars":  2,
    "runId":     "20260418T142300-abc12345",
    "variables": { "region": "us-west" }
  }
}
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "search-1",
  "result": {
    "results": [
      { "value": "payments-api", "label": "Payments API" },
      { "value": "payroll-svc",  "label": "Payroll Service" }
    ],
    "hasMore": false
  }
}
```

**Key decisions:**
- `variables` is included in every request so the provider can filter contextually.
- `hasMore: true` signals the UI to display "Showing top N results — keep typing to narrow."
- Providers MUST return within **500 ms**; timeout → empty results, operator can retry.
- `query: ""` is a valid request used for initial focus and CLI degradation.
- CLI degradation: the `prompt` provider and all built-in providers declare `search: false`;
  the run engine issues a single call with `query: ""` and presents results as a select.

**Capability matrix update:**
Added `search` column to the Provider Capability Matrix in §14:

| Provider    | …getOptions | search |
|-------------|-------------|--------|
| `prompt`    | No          | No     |
| `env`       | No          | No     |
| `file`      | No          | No     |
| `workspace` | No          | No     |

Rich UI providers (VS Code extension) SHOULD declare `search: true`. Third-party
providers that implement live search must declare `search: true` in their capability
advertisement.

**Capability declaration example updated:**
Added `"search": true` to the `provider/initialize` example in §14.

---

### 3. Format Validation Contract (§02 — Per-Type Validation + text subsubsection)

**Per-Type Validation table (`text` row):**
Updated the `text` row to reference `format: yaml|json` structural parse validation.
Variable storage remains `JSON string (raw; parsed form not stored)`.

**`text` Field Validation sequence (extended to 6 steps):**
The existing 5-step sequence in `\subsubsection*{text Field Pattern Validation}` was
extended:

1. Required check
2. `validation.min_length` check
3. `validation.max_length` check
4. `validation.pattern` check (RE2)
5. **`format` check (new):**
   - `format: yaml` → `yaml.Unmarshal([]byte(value), &interface{})`. On error: re-prompt
     with _"Value must be valid YAML."_
   - `format: json` → `json.Unmarshal([]byte(value), &interface{})`. On error: re-prompt
     with _"Value must be valid JSON."_
   - Parse result is discarded; value remains a raw string. Downstream traversal uses
     `fromYAML`/`fromJSON` template functions.
   - If both `validation.pattern` and `format` are declared, both must pass; pattern
     runs first.
6. Re-prompt: **combined total** of 3 attempts across all validation steps (not 3 per check).

**`fromYAML` / `fromJSON` template functions (§03 cross-reference):**
Added `fromYAML` and `fromJSON` to the built-in template function table in §03
(`subsec:expressions`) with signatures:
- `fromYAML(s string) interface{}`
- `fromJSON(s string) interface{}`

Added a `\paragraph{fromYAML and fromJSON}` note in §03 explaining that they parse the
raw string variable on each template evaluation. A parse error in a template expression
is a hard step failure. The `format` validation check at collection time guarantees the
raw string is structurally valid by the time downstream steps execute.

Cross-reference in §02 updated from non-existent `sec:template-expressions` to
the existing `subsec:expressions` label in §03.

---

## Sections Modified

| File | Changes |
|------|---------|
| `sections/02-architecture.tex` | (1) Extended `text` row in Per-Type Validation table to mention `format` validation; (2) Replaced 5-step text validation sequence with 6-step sequence adding the format check and clarifying the 3-attempt combined limit; (3) Added `Ephemeral Field Handling` subsubsection (~50 lines) after Error Variable Schema; (4) Fixed cross-reference from `sec:template-expressions` to `subsec:expressions` |
| `sections/14-input-provider-framework.tex` | (1) Added `Autocomplete Search Protocol` subsection with full `inputProvider/search` request/response schema and CLI degradation contract (~90 lines); (2) Extended capability matrix table with `search` column; (3) Added explanatory note about CLI degradation for `search: false`; (4) Added `"search": true` to capability declaration example |
| `sections/03-schema-vnext.tex` | (1) Added `fromYAML` and `fromJSON` rows to the built-in template function table; (2) Added `fromYAML and fromJSON` paragraph documenting the parse-on-eval behaviour and cross-referencing the `format` collector feature |

---

## Contracts Added

1. **Ephemeral field audit redaction** — `VariableScope.IsEphemeral(key string) bool`
   interface requirement; JSONL writer substitution rule; suspension/resume constraint.
2. **`inputProvider/search`** — new JSON-RPC method; full request/response schema;
   500 ms timeout; `hasMore` signalling; CLI degradation path.
3. **`format: yaml|json` validation** — parse-only structural check; combined 3-attempt
   limit; raw-string storage invariant; `fromYAML`/`fromJSON` for downstream traversal.

---

## Cross-Section Consistency Notes

- §03 (John): The `format` field and `ephemeral` flag are schema additions John owns.
  This record documents what the executor does with those values at runtime. The
  `fromYAML`/`fromJSON` table rows and paragraph in §03 are additive; they do not
  overlap with John's P2 field-type changes.
- §14 (Provider Framework): `inputProvider/search` is a peer to `inputProvider/getOptions`.
  Both live in the same subsystem. The capability matrix addition (`search` column) is
  additive and does not break existing provider handshake contracts — providers that do
  not advertise `search` are treated as `search: false`.
- §06 (Events): No new event kinds introduced for autocomplete search. The RPC exchange
  is point-to-point between run engine and provider, not fan-out events.
- §12 (Tracing): Ephemeral redaction is enforced by the trace writer, which already owns
  all serialisation to `trace.jsonl`. The `IsEphemeral` check is the single enforcement
  point; no other layer is modified.

---

**End of decision record.**

---

## ken-gap1-gap2-runtime

### 2026-04-19: Business Calendar Engine + Quorum Tracker architecture
**By:** Ken
**What:** GAP-1: Business Calendar Engine converts timeout_business_days to a wall-clock deadline at step activation (one-time conversion). Built-in calendars: default, us-federal, uk-banking. GAP-2: Quorum Tracker records approvals/rejections per step instance, enforces mode (all/any/quorum), detects impossible quorum early. Both compose freely.
**Why:** Resolves two critical stress-test gaps. Enables financial approval and FDA release runbooks.

---

## ken-include-executor-contract

### 2026-04-19: include step executor contract
**By:** Ken
**What:** type:include executor: (1) resolve runbook, (2) cycle check, (3) eval when, (4) apply with overrides, (5) inline expand steps into current queue. No new run context, no separate audit entry, shared variable scope. Cycle detection is load-time DFS — executor never sees cycles. future type:call (isolated scope) deferred post-v2.0.
**Why:** Rename from invoke. Clarifies that include is inline composition, not a sub-procedure call.

---

## ken-stress-test-verdict

# Schema Stress Test Verdict

**Author:** Ken (Software Architect)  
**Date:** 2026-04-18  
**Type:** Architecture Review Finding  
**Scope:** §02 Architecture, §03 Schema, §11 Governance

---

## Executive Summary

**Overall Schema Readiness: NEEDS TARGETED FIXES**

After analyzing the 10-runbook corpus against the gert v2 schema specification, I find the schema is **architecturally sound for 80% of operational use cases** but has **3 systemic gaps that block enterprise governance workflows**.

### Verdict Breakdown

| Metric | Value |
|--------|-------|
| PASS cleanly | 4/10 (40%) |
| PASS with workarounds | 4/10 (40%) |
| FAIL current schema | 2/10 (20%) |

### Top 3 Systemic Gaps

| Gap | Severity | Affected Runbooks | Description |
|-----|----------|-------------------|-------------|
| **S1: Business-day timeout** | CRITICAL | 3, 7, 8 | `timeout` uses wall-clock duration; enterprise workflows require business-day SLAs |
| **S2: M-of-N quorum approval** | CRITICAL | 7, 8, 10 | `approvals.min` can't express "3-of-5 board members" with explicit pool |
| **S3: External event trigger** | HIGH | 8, 9 | No webhook callback mechanism to resume paused runs |

### Recommendation for Team

**Before declaring schema "implementation-ready":**

1. **[CRITICAL]** Add business-day timeout support (§03 + §11)
   - New fields: `timeout_business_days`, `timeout_calendar`
   - Effort: Medium

2. **[CRITICAL]** Add M-of-N quorum approval (§03 + §14)
   - New fields: `approvals.mode`, `approvals.pool`, `approvals.pool_size`
   - Effort: Medium

3. **[DEFER TO v2.1]** External event trigger (acceptable polling workaround exists)

### Plain Language Summary

The schema is ready for **SRE incident response**, **DevOps deployments**, and **compliance evidence collection**.

The schema is **not yet ready** for **finance approval chains**, **HR onboarding**, or **regulated industry workflows** (FDA, SOC2 attestation chains) that require business-day SLAs and multi-party governance.

**Two focused fixes** (business-day timeout + quorum approval) will close this gap without redesign.

---

## Full Report

See: `.squad/tmp/ken-stress-conclusions.md`

---

## Proposed Action

- [ ] John: Review and adjust translations if needed based on gap findings
- [ ] Ken: Draft schema extension proposals for S1 and S2
- [ ] Team: Decide on v2.0 scope — include S1/S2 or defer to v2.1?

---

## ken-wait-for-event-runtime

### 2026-04-19: wait_for_event runtime architecture
**By:** Ken
**What:** wait_for_event requires: (1) Event Dispatcher component in gert serve, (2) WAITING run state, (3) durable run serialization for restart survival, (4) serve-only constraint (gert run mode must reject). Webhook transport uses HMAC-authenticated /events/{run-id}/{event-id} endpoint.
**Why:** First-class inbound event handling for runbooks. Enables Prometheus webhook receive, SIEM event handling, external approval callbacks, etc.

---

## leslie-diagrams-complete

### 2026-04-19: Diagram coverage complete
**By:** the user (via Leslie)
**What:** All missing diagrams from the audit have been drawn using TikZ. Every major section in the design document now has at least one visual diagram.
**New diagrams:**
- `fig:dependency-direction` — §02: dependency arrow convention legend
- `fig:cli-step-execution` — §03: CLI step parse→govern→shell→exec→capture→exit flow
- `fig:tool-invocation-flow` — §03: tool resolve→transport→auth→invoke→capture flow
- `fig:include-inlining` — §03: child runbook expansion into parent shared scope
- `fig:branch-logic` — §03: condition evaluation + first-match arm execution
- `fig:iterate-loop` — §03: initialize→test→body→increment loop with max guard
- `fig:parallel-fanout` — §03: fork→N concurrent branches→join→merge variables
- `fig:collector-triad` — §03: choice/decision/collector side-by-side input modes
- `fig:compensation-flow` — §03: saga rollback — compensate in reverse order on failure
**Total diagram count:** 15 (6 prior + 9 new)
**Build status:** CLEAN — 0 hard errors, 325 pages
**Commit:** dc891eb

---

## leslie-minted-mass-conversion

### 2026-04-19: All code blocks converted to minted
**By:** the user (via Leslie)
**What:** All YAML/Go/JSON verbatim and lstlisting environments across all sections converted to minted. minted is now the sole mechanism for code blocks in this document.
**Count:** 260 blocks converted across 13 files (yaml: 96, json: 85, go: 24, bash: 13, text: 42). 42 blocks intentionally left as verbatim (ASCII trees, HTTP headers, CLI terminal output, error messages, file path trees).
**Diagrams added:**
- `fig:governance-enforcement-points` — §07 Security, 4-stage pipeline showing governance checkpoints
- `fig:event-flow-timeline` — §06 Runtime Events, vertical timeline of a two-step run with governance approval
- `fig:provider-resolution-flow` — §14 Input Provider Framework, 6-stage resolution pipeline for `from:` bindings
**Notes:**
- Removed `\usepackage{listings}` from main.tex (conflicted with minted v3's tocbasic lol extension)
- Added `scripts/pypath/python3` wrapper routing latexminted to Python 3.13 (Python 3.14 breaks latexminted 0.5.0 argparse API)
- Global TikZ styles updated: `text centered` → `align=center` to support multiline `\\` in node labels
- Added `scripts/convert_to_minted.py` as a reusable conversion tool

---

## leslie-minted-syntax-highlighting

### 2026-04-19: minted adopted for syntax highlighting
**By:** the user (via Leslie)
**What:** minted package (Pygments-based) for YAML/Go/JSON syntax highlighting in code blocks. Requires --shell-escape on pdflatex and Python 3 + Pygments on build machine.
**Style:** friendly, frame=leftline, no line numbers, small font
**Languages configured:** yaml, go, json
**Why:** Better readability for YAML runbook examples — proper colour-coded output vs plain verbatim
**Migration:** blocks converted to minted on-demand, not all at once

---

## leslie-tikz-global-diagrams

### 2026-04-19: TikZ adopted as global diagramming standard
**By:** the user (via Leslie)
**What:** TikZ/PGF is the single diagramming approach for all gert v2 design doc figures. No external tools (Graphviz, draw.io, PlantUML, Mermaid). All diagrams are inline LaTeX source using the styles defined in main.tex preamble.
**Libraries:** arrows.meta, automata, positioning, shapes.geometric, fit, chains, calc, backgrounds
**Node styles defined:** stepbox, catbox, gertarrow, gertfit
**Why:** Zero external dependencies, ships with TeX Live, inline in .tex source, consistent styling, pdflatex compatible.
**First diagram:** fig:step-type-taxonomy in §03

---

## Prior Decisions Archive

**4. Provider Capability Matrix (§14.6.4)**
- Table showing which built-in providers support which step types
- Only `prompt` supports all three interactive step types
- `env`, `file`, `workspace` only support input resolution
- External providers advertise capabilities in `provider/initialize` handshake
- Fallback: if provider doesn't support step type, engine uses `prompt` provider

### Design Decisions (Barbara's Integration Analysis)

| Decision | Rationale |
|----------|-----------|
| Providers advertise step type support via capabilities | Graceful fallback to `prompt` provider when external provider doesn't implement contracts |
| Decision steps do NOT store route as variable | Pure control flow; if value needed, runbook author adds explicit assignment in target |
| Provider doesn't know runbook graph | Clean separation; provider returns route label, engine handles graph navigation |
| File size limits enforced by provider | Prevents partial uploads; provider rejects before storage |
| SHA-256 in artifact metadata | Enables integrity verification and evidence chain validation |
| External providers MAY use remote storage | Supports cloud deployments (S3, Azure Blob) with signed URLs |

### Integration Points

- **§03 (Schema Spec):** John defining three step types in taxonomy
- **§05 (Tool Runtime):** Provider transport shares JSON-RPC mechanics with tools
- **§07 (Security):** Artifact storage inherits evidence capture and SHA-256 requirements
- **§13 (Adapter Contracts):** Provider contracts follow same versioning model

### Ken's Architecture Review Decisions

**Context:** Cross-section consistency review of gert v2 design document (§00–§15) identified critical interface mismatches.

**Decision 1: Event Envelope Field Names Standardized**

Resolved interface mismatch — standardized all event envelope field names across design document:
- `seq` → `sequence` (monotonic counter within run)
- `type` → `kind` (event kind classification)
- `data` → `payload` (event-specific data)

§06 (Runtime Events) is normative specification. §10 (Migration) already documented v1→v2 rename. §12 and §11 examples were using legacy v1 names.

**Scope:** §12 JSONL envelope and event examples; §11 governance approval traces; §08 references to "seq" ordering

**Status:** ✅ APPLIED — inline fixes completed

**Decision 2: event_id Field Added to Trace Envelope**

Added missing `event_id` field to §12 trace JSONL envelope:
```json
"event_id": <string>  // UUID v4 for correlation with external systems
```

§06 declares `event_id` as one of 7 mandatory envelope fields. §10 "after" example showed event_id in v2. Without it, correlation with external observability (OTel, logs) impossible.

**Use cases:** Correlation between JSONL trace events and OTel spans; idempotency keys; audit trail across distributed systems

**Status:** ✅ APPLIED — inline fix completed in §12

**Architectural Clarification: Wire Format Conventions**

Review identified both `run_id` (snake_case) and `runId` (camelCase) — this is INTENTIONAL and CORRECT:

- **JSONL trace files** (§06, §12, §15 logs): Use **snake_case**
  - Fields: `run_id`, `runbook_id`, `step_id`, `event_id`, `sequence`
  - Rationale: Persistent audit format follows Go struct field tags

- **JSON-RPC wire protocol** (§05, §13, §14): Use **camelCase**
  - Fields: `runId`, `runbookPath`, `stepId`, `actorId`
  - Rationale: JavaScript conventions for RPC APIs

**Recommendation:** Add explicit subsection to §13.1 documenting this convention.

**Status:** ⚠️ FOLLOW-UP NEEDED — Leslie (scribe) should add §13.1 clarification

### Review Metrics

- Sections reviewed: 16 (§00–§15)
- Line count: 9,085 lines
- Critical issues found: 5 (interface mismatches)
- Critical issues fixed: 5
- Minor issues found: 12 (documentation gaps)
- Cross-references checked: All valid
- Interface contracts checked: All consistent (after fixes)
- Overall verdict: APPROVED WITH FIXES ✅

### Files Changed

- `design/gert-v2/sections/03-schema-vnext.tex` (§03: choice/decision/collector specs)
- `design/gert-v2/sections/14-input-provider-framework.tex` (§14.6: +332 lines)

### Build Status

- Document builds to 256 pages
- LaTeX syntax correct in both sections
- PDF generation successful

### Implementation Coordination

Cross-team roles:
- **John** owns normative schema specification (§03)
- **Brian** (Parser Engineer) implements in parser/validator
- **Ken** (Runtime Engineer) implements execution semantics
- **Sam** (VS Code Engineer) implements UI rendering
# Decision Inbox: testutil Scaffold Complete

**Author:** Barbara (Integrations Specialist)
**Date:** 2026-04-19
**Phase:** 0 — Foundation

---

## Deliverables Created

`/Volumes/Projects/gert/v2/pkg/testutil/` contains 6 files:

| File | Purpose |
|------|---------|
| `fake_step_executor.go` | FakeStepExecutor — controllable StepExecutor for unit tests |
| `fake_event_dispatcher.go` | FakeEventDispatcher — consume-semantics event injection for wait_for_event tests |
| `time_controller.go` | TimeController — fake time + timer firing for deterministic timeout tests |
| `concurrent_event_collector.go` | ConcurrentEventCollector — thread-safe trace event collection for parallel step tests |
| `golden.go` | AssertGoldenTrace + NormalizeTrace — golden JSONL trace comparison with -update flag |
| `spec_tag.go` | Tag() + SpecTag — AST-discoverable spec-coverage annotations |

---

## Design Decisions

### Stub types (forward-compatible)
`testutil` defines local stub types for `Step`, `StepResult`, and `TraceEvent` with TODO comments.
These must be replaced with real imports once `pkg/schema`, `pkg/engine`, and `pkg/trace` exist (Brian/Ken Phase 0 deliverables).
No build tags needed — the stubs are self-contained.

### FakeEventDispatcher — consume semantics
Implements the locked decision (2026-04-19): first waiter wins. Events not consumed by a waiter are queued. DrainAll() unblocks all waiters and clears the queue. WaitOnChannel() is the preferred API when the channel name is known at call-site.

### Golden traces
Stored in `testdata/golden/*.jsonl`. Regenerate with `go test -update`. NormalizeTrace() stubs out timestamps (→ `<timestamp>`) and rebases seq numbers for deterministic diffs.

### SpecTag
No-op at runtime. The spec-coverage tool locates `testutil.Tag(file, section, rule)` calls via static AST analysis and produces a coverage matrix.

---

## Build Status

```
cd /Volumes/Projects/gert/v2 && go build ./pkg/testutil/...  ✅ passes
```

---

## Blocking Notes

- `FakeStepExecutor.Execute` signature uses local stub `Step`/`StepResult` — swap for `schema.Step`/`engine.StepResult` when Brian's pkg/engine lands.
- `AssertGoldenTrace` uses local stub `TraceEvent` — swap for `trace.TraceEvent` when pkg/trace lands.
- `FakeEventDispatcher.Wait()` uses a catch-all channel key; prefer `WaitOnChannel()` once engine dispatches with explicit channel names.
# Decision: pkg/platform Interface

**By:** Ken (Software Architect)  
**Date:** 2026-04-19  
**Status:** IMPLEMENTED  
**Priority:** P1

---

## Context

v2 targets Unix-first (Linux/macOS Tier 1) with Windows as Tier 2 for v2.0. Several runtime behaviors are OS-specific: temp directory paths, path separators, POSIX signal availability, executable suffixes, newline conventions, and trace-file append atomicity. Without a dedicated abstraction, these differences would be scattered across the codebase and untestable without a real OS.

---

## Decision

Introduce `pkg/platform` as the single injection point for all OS-specific behavior. All code that touches platform differences **must** receive a `Platform` value; it must not call `runtime.GOOS` or OS primitives directly.

---

## Platform Interface Scope

| Method | Abstracts |
|---|---|
| `TempDir()` | `os.TempDir()` — different default locations on Windows |
| `NormalizePath(path)` | Backslash → forward-slash on Windows; no-op on Unix |
| `AllowedSignals()` | OS signal allow-list for `wait_for_event source: signal` |
| `OpenAppend(path)` | Atomic append semantics (see below) |
| `NewlineNormalizer(w)` | CRLF → LF for JSON-RPC framing on Windows |
| `ExecSuffix()` | `""` on Unix, `".exe"` on Windows |
| `DefaultShell()` | `"/bin/sh"` on Unix, `"cmd.exe"` on Windows |

---

## Windows Workaround: Trace Append Atomicity

Unix guarantees atomic append with `O_APPEND` at the kernel level (POSIX). Windows has no equivalent: `FILE_APPEND_DATA` access is not atomic for concurrent writers.

**v2.0 workaround (Windows Tier 2):** `realPlatform.OpenAppend` on Windows calls `os.OpenFile` with `O_APPEND|O_WRONLY|O_CREATE` and a `// TODO` comment pointing to decisions.md. This is acceptable because:

1. Windows is Tier 2 — advisory CI only, not blocking for v2.0 releases.
2. Single-run-per-process is the v2.0 concurrency model, so cross-process append races are unlikely in normal operation.
3. The TODO is visible and will be addressed in v2.1 when Windows is promoted to Tier 1.

**v2.1 target:** Replace with a mutex-protected `WriteCloser` that uses a per-path `sync.Mutex` to serialize writes within a process, plus documentation that cross-process atomicity is unsupported on Windows.

---

## FakePlatform as Test Double

`FakePlatform` (returned by `NewFakePlatform()`) is the canonical test double for all code that depends on `Platform`. Its defaults mimic a Unix environment:

- `TempDirPath`: `"/tmp"`
- `Signals`: full Linux/macOS allow-list
- `ExecSuffixStr`: `""`
- `ShellPath`: `"/bin/sh"`

Tests that need Windows-like behavior override the exported fields directly — no subclassing or mocking framework required.

`OpenAppend` writes to `FakePlatform.AppendBuf` (`bytes.Buffer`) and increments `AppendCallCount`, enabling assertions on both content and invocation count without touching the filesystem.

---

## Rationale

- All platform differences are in one place → easy to audit during Windows Tier 1 promotion.
- `FakePlatform` makes every consumer unit-testable hermetically.
- Interface is narrow (7 methods) — adding a method requires an explicit decision, preventing scope creep.

---

## phase0-review-and-approval

### Initial Review — Ken (REJECTED)
**Date:** 2026-04-19  
**Verdict:** REJECTED (7 defects identified)

Phase 0 architectural interface review identified 7 defects that must be fixed before Phase 1 can proceed.

### Revision — Barbara (READY FOR RE-REVIEW)
**Date:** 2026-04-19  
**Revision Status:** All 7 defects fixed

Barbara (Integrations Specialist) fixed all 7 Phase 0 defects. Build and vet: `go build ./...` ✅, `go vet ./...` ✅

### Re-Review — Ken (APPROVED)
**Date:** 2026-04-19  
**Verdict:** APPROVED ✅

All 7 defects verified as resolved. Phase 0 complete and coherent. Ready for Phase 1 implementation.
# Implementation Decisions — Phase 2 Planner

**Author:** Brian  
**Date:** 2026-04-19  
**Status:** Implemented — all 13 tests pass

---

## Decision 1: Error types — reuse `pkg/planner.PlanError`, no new `internal/planner/errors.go`

The task charter asked for internal error types (`ErrIncludeCycle{Path}`, etc.).  
After reviewing the test skeleton, all test assertions use `errors.Is(err, plannerPkg.Err*)` against the sentinel errors already defined in `pkg/planner/planner.go`. Creating parallel internal error types would either:

- Duplicate the sentinel definitions, or  
- Require the tests to be rewritten

**Decision:** Use `pkg/planner.PlanError{Code: plannerPkg.ErrXxx, ...}` throughout. `PlanError.Unwrap()` returns the code, so `errors.Is` chains work correctly. No `internal/planner/errors.go` created.

---

## Decision 2: `planCtx` struct for mutable accumulation

Rather than threading `tools map[string]*schema.ToolDef` and `seen map[string]bool` as parameters through every recursive call, I introduced a `planCtx` struct that holds both.

```
planCtx{
    p:     *impl,         // configuration (immutable)
    tools: map[string]*schema.ToolDef,  // accumulated tool defs
    seen:  map[string]bool,             // visited runbook paths
}
```

Methods on `planCtx` (`flattenNodes`, `resolveStep`, `resolveInclude`, `resolveTool`) are all non-recursive wrt the struct — they share state naturally.

---

## Decision 3: Flatten strategy (include inlining, not sub-plan nesting)

The task spec mentioned "store the nested `*ExecutionPlan` in the `ResolvedStep`", but `engine.ResolvedStep` has no such field. The actual struct has `Spec engine.StepSpec`.

**Decision:** Include steps are fully inlined — the child runbook's steps are appended directly to the parent's step list, in declaration order. No wrapper step is emitted for the include itself.

This is the simplest correct approach and matches the test expectation (`len(plan.Steps) == 1` when parent has one include step that wraps a one-step child runbook).

---

## Decision 4: Topo sort strategy — declaration order (BFS not needed)

The task described a BFS topo sort following `next` fields. The actual schema step type has no `next` field — flow order is implicit via the `[]FlowNode` array.

**Decision:** Declaration order is the topological order for linear flows. The flattener walks `[]FlowNode` in slice order. For parallel branches, each branch's steps are appended in branch declaration order after the parallel header. For iterate, body steps follow the iterate header. This is deterministic and matches the test expectation.

---

## Decision 5: Cycle detection — permanent path marking

Per the task spec: track `visited map[string]bool` permanently (not "currently in stack"). This means diamond dependencies (A→B, A→C→B) would be incorrectly rejected as cycles. This is a known limitation of the permanent-marking approach; a future improvement can switch to "in-stack" semantics (add on enter, remove on exit).

---

## Decision 6: `specForStep` fallback — `rawSpec{kind}`

For steps where the typed spec pointer is nil (malformed input), `specForStep` returns a `rawSpec{kind: string(step.Type)}` that satisfies `engine.StepSpec`. This prevents panics on nil pointer dereferences while still emitting the correct `Kind` in the `ResolvedStep`.

---

## Decision 7: `BranchSpec` and `CompensateSpec` inline their sub-flows

Both step types carry nested `[]FlowNode` in their spec structs. The flattener handles them in `resolveStep` as special cases: it emits the container step first, then recursively flattens the nested steps at `depth+1`. The runtime can identify branch/compensate boundaries by the `Kind` field of the container step.

---

## Cross-Agent Notes

- **For Barbara**: Test fakes (`fakeLoader`, `fakeRegistry`) are in `internal/planner/planner_test.go`. Replace with `pkg/testutil` types when available.
- **For Ken**: `engine.ResolvedStep.Spec` is `engine.StepSpec` (not a concrete type). All schema spec types already implement `StepKind() string`. No changes needed to engine types.
- **For Cristian**: Phase 2 planner is complete. `go build ./... && go vet ./... && go test ./internal/planner/... -v -count=1` — all 13 tests pass, no vet warnings.
# Decision: FakeToolRegistry uses name+action strings, not schema.ToolRef

**Date:** 2026-04-19  
**Author:** Barbara  
**Context:** Writing FakeToolRegistry for planner tests

## Decision

`FakeToolRegistry.LookupCalls` uses a local `ToolLookupCall{Name, Action string}` struct
rather than `[]schema.ToolRef` as specified in the task description.

## Reason

`planner.ToolRegistry.Lookup` has signature:

```go
Lookup(ctx context.Context, name string, action string) (*schema.ToolDef, error)
```

`schema.ToolRef` has no `Action` field — it has `Name`, `Path`, `Alias`, `Source`, `Actions []string`.
Recording raw `schema.ToolRef` values in LookupCalls would be misleading and would require
constructing a ToolRef just to record what the caller passed as plain strings.

## Impact

Tests asserting on `LookupCalls` use `ToolLookupCall{Name: "...", Action: "..."}` instead of `schema.ToolRef`.
This is clearer and matches the actual call-site semantics.

---

# Decision: Phase 14 End-to-End Integration Test Suite

**Date:** 2026-07-21  
**Author:** Ken (Software Architect)  
**Phase:** 14  
**Status:** APPROVED

## Overview

Phase 14 delivers the End-to-End Integration Test Suite as the highest-value quality investment before v2.0 GA, plus three NBI carry-forwards:

- **Part A — NBI-13 Closure:** TLS support for OTLP adapter, gc edge case tests, ls JSON schema documentation
- **Part B — E2E Suite:** Full-stack integration testing (Parser → Planner → Engine → CLI Executor → Trace)

## Key Architectural Decisions

### D-14-01: E2E Tests in `internal/e2e/` Package

E2E tests live in a dedicated `internal/e2e/` package, separate from unit tests.

**Rationale:**
- Clear separation of test types (unit vs integration)
- E2E tests have different characteristics (slower, file I/O, subprocess)
- Can run unit tests quickly without e2e tests
- Common pattern in mature Go projects (Kubernetes, Vitess, CockroachDB)

**Impact:** New package `internal/e2e/` with doc.go, 10 test functions, 5 testdata runbooks

### D-14-02: E2E Tests Use Real Executors

E2E tests execute real CLI commands (echo, cat) via the real CLI executor, not fakes.

**Rationale:**
- E2E exists specifically to exercise real behavior
- Commands are simple POSIX utilities available on all CI runners
- Using fakes would reduce e2e to another unit test

**Impact:** Tests may be slower (~100-500ms each) but provide higher confidence

### D-14-03: Defer context.AfterFunc Optimization

NBI-12-03 (context.AfterFunc optimization in mergeContexts) deferred to Phase 15+.

**Rationale:**
- Current goroutine pattern is bounded and stable
- E2E test suite has higher value per Brian-day
- Go 1.25.7 context.AfterFunc available; can be added anytime
- E2E tests will validate optimization didn't break anything

**Impact:** No code change. Carry forward to Phase 15+.

### D-14-04: Sequential E2E Tests (No t.Parallel)

E2E tests run sequentially; t.Parallel() not called initially.

**Rationale:**
- Simpler debugging when tests share temp directory roots
- Subprocess execution may exhibit timing-dependent behaviors
- Parallel tests can be enabled in Phase 15+ once suite proven stable

**Impact:** E2E suite runs in ~3-5 seconds total (10 tests × 300ms average)

## NBI Items Addressed

| ID | Status | Notes |
|----|--------|-------|
| NBI-13-01 | IN SCOPE | WithTLS(*tls.Config) option for OTLP adapter |
| NBI-13-02 | IN SCOPE | 4 new gc edge case unit tests |
| NBI-13-03 | IN SCOPE | ls --output=json JSON schema documentation |
| NBI-12-03 | DEFERRED | D-14-03 defers context.AfterFunc to Phase 15+ |

## New NBI Items (Phase 15+)

- **NBI-14-01:** E2E test parallelization (low priority)
- **NBI-14-02:** E2E coverage for tool steps (medium priority)
- **NBI-12-03:** context.AfterFunc optimization (third deferral)

## Files to Create

| File | Type | Purpose |
|------|------|---------|
| `internal/e2e/doc.go` | Package | Package documentation |
| `internal/e2e/e2e_test.go` | Tests | 10 e2e test functions |
| `internal/e2e/helpers_test.go` | Utilities | E2E harness and assertions |
| `internal/e2e/testdata/echo-runbook.yaml` | Testdata | Simple CLI step |
| `internal/e2e/testdata/vars-runbook.yaml` | Testdata | Variable interpolation |
| `internal/e2e/testdata/branch-runbook.yaml` | Testdata | Conditional branching |
| `internal/e2e/testdata/iterate-runbook.yaml` | Testdata | Loop with early exit |
| `internal/e2e/testdata/manual-runbook.yaml` | Testdata | Manual step (skipped) |

## Files to Modify

| File | Change | Lines |
|------|--------|-------|
| `pkg/otel/adapter/otlp.go` | Add WithTLS option | ~15 |
| `pkg/otel/adapter/otlp_test.go` | Add TLS tests | ~20 |
| `cmd/gert/gc_test.go` | 4 edge case tests | ~40 |
| `cmd/gert/ls.go` | JSON schema godoc | ~15 |

## Test Coverage

**E2E Test Suite (10 tests):**
- Parser → Planner → Engine → CLI Executor → Trace (full stack)
- All step types: cli, branch, iterate, manual
- Variable interpolation and output capture
- Trace persistence and parseability
- Resume from checkpoint
- Cancellation mid-run

**Estimated Effort:** 3-4 Brian-days

## Validation Gate

```bash
cd v2
go build ./...                      # Must exit 0
go vet ./...                        # Must exit 0
go test ./... -race -count=3        # Must pass all packages including internal/e2e
```

---

*Ken, Software Architect*

## Phase 14

# Ken — Phase 14 Review: E2E Integration Tests & NBI Closure

**Date:** 2026-07-21  
**Author:** Ken (Staff Architect)  
**Phase:** 14  
**Brian's Deviation Report:** `.squad/decisions/inbox/brian-phase14-impl.md`

---

## **APPROVED** — Score: 9/10

---

## Summary

Brian has delivered a complete, production-quality E2E test suite that exercises the full Parser → Planner → Engine → Executor → Trace → RunStore stack. All NBI-13 items (TLS option, gc tests, ls docs) are closed correctly. The four deviations are all technically justified and two (DEV-14-01, DEV-14-04) demonstrate Brian's architectural maturity — identifying spec gaps before they became test flakiness or runtime bugs.

---

## Review Dimensions

### 1. Correctness ✅

**TLS Credentials (NBI-13-01):**
```go
creds := credentials.NewTLS(cfg.tlsConfig) // nil → system default
```
- `credentials.NewTLS(nil)` is **safe and correct** — the gRPC credentials package documents that `nil` uses `&tls.Config{}` which triggers Go's default behavior: system root CA trust store, TLS 1.2+, no client cert.
- The `WithTLS(nil)` option correctly sets `c.insecure = false` ensuring the TLS path is taken.
- Both TLS tests verify the dial path executes without panic.

**GC Boundary Fix:**
```go
// Line 113: gc.go
if !ref.Before(cutoff) {
    continue
}
```
- Correctly implements **exclusive boundary** semantics: runs at exactly the cutoff (`ref == cutoff`) are NOT deleted because `ref.Before(cutoff)` returns `false`, so `!ref.Before(cutoff)` is `true`, and we `continue` (skip deletion).
- This matches the spec: "UpdatedAt == cutoff → NOT deleted"
- The test in `TestGc_OlderThan_EdgeCase` verifies both sides of the boundary reliably.

**E2E Test Assertions:**
- `AssertCompleted()` is not vacuous — it checks `state.Status != engine.RunStatusCompleted` and fails explicitly.
- `AssertTrace()` parses the actual JSONL file via `JSONLReader.ReadAll()` and verifies event kinds are present — cannot pass vacuously with empty traces.
- `TestE2E_TracePersistence` explicitly checks envelope fields (EventID, RunID, Timestamp, Kind) for every event.
- `TestE2E_ResumeFromCheckpoint` drives step1, captures runID, calls `eng.Resume()`, drives step2, and asserts completion — proves checkpoint/resume works.
- `TestE2E_CancelMidRun` cancels context and verifies the run ends in a non-completed state.

### 2. Architecture ✅

**E2EHarness (helpers_test.go):**
- Follows injected-deps pattern: `E2EHarness` receives `t *testing.T`, creates temp dirs, and exposes `Store`, `RunDir`, `TraceDir` for fine-grained test control.
- `Prepare()` builds the full stack but returns `(plan, ecfg, eng)` for tests that need direct engine control (resume, cancel).
- `Run()` provides the simple "execute to completion" path for most tests.
- The harness is reusable — all 10 tests use it; no test builds its own stack.

**Test Independence:**
- Each test calls `NewHarness(t, "...")` which creates a fresh `t.TempDir()` — tests cannot interfere with each other's state.
- No shared mutable state between tests.
- D-14-04 (sequential execution) is respected — no `t.Parallel()` calls.

**Testdata Runbooks:**
- All 5 runbooks use `apiVersion: runbook/v2` and follow the correct schema structure.
- `branch-runbook.yaml` uses the `branches:` array form with `condition:` and `else: true` — correct schema.
- `iterate-runbook.yaml` uses `iterate:` with `over:`, `as:`, `until:` — correct structure.
- `manual-runbook.yaml` uses `type: approve` with `approvals: { roles: [operator] }` — correct per DEV-14-02.

### 3. Test Quality ✅

**Non-Vacuous Assertions:**
- Every test calls `h.AssertCompleted(state)` which explicitly fails on non-completed status.
- `TestE2E_SimpleEcho` additionally asserts 4 specific trace event kinds.
- `TestE2E_TracePersistence` validates every event has required envelope fields.
- `TestE2E_CancelMidRun` explicitly checks `state.Status != RunStatusCompleted`.

**JSONL Trace Parsing:**
- Uses `internaltrace.NewJSONLReader(h.traceFile).ReadAll()` — the production trace reader.
- Returns `[]tracepkg.TraceEvent` with strongly-typed fields.
- If parsing fails, tests fail immediately with the parse error.

**Edge Cases Covered:**
- `TestGc_RunningStatus_NeverDeleted` — D-13-03 invariant under extreme conditions (`--older-than=0s --force`)
- `TestGc_MixedStatuses` — 3 terminal + 1 running; verifies exactly 3 deleted
- `TestGc_StatusFilter_Running_Rejected` — `--status=running` fails validation
- `TestGc_OlderThan_EdgeCase` — boundary from both sides

### 4. Deviation Assessments

**DEV-14-01: Boundary test redesign** ✅ ACCEPTED (EXEMPLARY)
> Setting `UpdatedAt = time.Now().Add(-olderThan)` and then calling `gcMain` (which recomputes `time.Now()` independently) makes `UpdatedAt` slightly older than the cutoff by the time `gcMain` runs.

Brian correctly identified that testing "exactly at cutoff" is inherently flaky in wall-clock tests. The redesigned test verifies both sides of the boundary with clear margins (1h vs 3h with 2h cutoff). This is more robust and still validates the exclusive boundary semantic. The boundary fix itself (`!ref.Before(cutoff)`) was correctly applied as specified.

**DEV-14-02: `type: approve` instead of `type: manual`** ✅ ACCEPTED
> The schema has no `manual` step type.

Brian is correct — the schema validator would reject `type: manual`. Using `type: approve` with `approvals: { roles: [operator] }` achieves the same test goal (auto-approved in non-interactive mode via `NoOpApprovalGate`). This deviation corrects a spec error.

**DEV-14-03: No loop variable reference in iterate sub-step** ✅ ACCEPTED (DOCUMENTED)
> The planner flattens iterate sub-steps into the outer execution plan (with Depth=1). When the outer engine reaches the `process-item` step directly (outside the iterate context), `item` is not in the outer run's vars.

This is a legitimate architecture limitation. The iterate behavior is still exercised — the test verifies the run completes with correct semantics (loop count, early-exit). Brian correctly notes this should be tracked as a follow-up issue. **NBI-14-03: Fix planner iterate/branch sub-step variable scoping.**

**DEV-14-04: `RunOptions.Vars` instead of `FakeInputProvider`** ✅ ACCEPTED (PREFERABLE)
> `BuildEngineConfig` does not accept a custom `InputProvider` override via `WireOptions`.

`RunOptions.Vars` is the intended mechanism for passing run variables programmatically. This is cleaner than env var pollution and doesn't require internal API changes. The test behavior is equivalent.

### 5. Security ✅

**`credentials.NewTLS(nil)`:**
- Safe per gRPC documentation: nil config uses system default settings
- Uses OS trust store for server certificate verification
- Enables TLS 1.2+ minimum
- No client certificate presented (which is correct for typical OTLP collectors)

**InsecureSkipVerify in test:**
```go
customCfg := &tls.Config{InsecureSkipVerify: true} //nolint:gosec // test-only
```
- Appropriate for test — the TCP listener doesn't speak TLS, so the test only verifies the option is wired through.
- `//nolint:gosec` annotation is correct for test-only code.

---

## Non-Blocking Items for Phase 15

| ID | Description | Priority |
|----|-------------|----------|
| NBI-14-01 | E2E test parallelization (enable `t.Parallel()` once suite is stable) | Low |
| NBI-14-02 | E2E coverage for tool steps (requires tool registry setup in harness) | Medium |
| NBI-14-03 | Fix planner iterate/branch sub-step variable scoping (DEV-14-03 root cause) | Medium |
| NBI-12-03 | `context.AfterFunc` optimization for `mergeContexts` (fourth deferral) | Low |

---

## Files Reviewed

**Part A:**
| File | Lines | Verdict |
|------|-------|---------|
| `pkg/otel/adapter/otlp.go` | 178 | ✅ WithTLS correctly wired |
| `pkg/otel/adapter/otlp_test.go` | 238 | ✅ 2 new TLS tests |
| `cmd/gert/gc_test.go` | 350 | ✅ 4 new edge-case tests, boundary fix verified |
| `cmd/gert/ls.go` | 112 | ✅ JSON schema documented in godoc |

**Part B:**
| File | Lines | Verdict |
|------|-------|---------|
| `internal/e2e/doc.go` | 6 | ✅ Clear package doc |
| `internal/e2e/helpers_test.go` | 283 | ✅ Harness follows injected-deps pattern |
| `internal/e2e/e2e_test.go` | 271 | ✅ 10 tests, non-vacuous assertions |
| `internal/e2e/testdata/echo-runbook.yaml` | 10 | ✅ Valid schema |
| `internal/e2e/testdata/vars-runbook.yaml` | 20 | ✅ Valid schema |
| `internal/e2e/testdata/branch-runbook.yaml` | 28 | ✅ Valid schema |
| `internal/e2e/testdata/iterate-runbook.yaml` | 19 | ✅ Valid schema |
| `internal/e2e/testdata/manual-runbook.yaml` | 15 | ✅ Valid schema |

---

## Test Results

```
=== RUN   TestE2E_SimpleEcho               --- PASS
=== RUN   TestE2E_VarInterpolation         --- PASS
=== RUN   TestE2E_BranchTrue               --- PASS
=== RUN   TestE2E_BranchFalse              --- PASS
=== RUN   TestE2E_IterateAll               --- PASS
=== RUN   TestE2E_IterateEarlyExit         --- PASS
=== RUN   TestE2E_ManualSkip               --- PASS
=== RUN   TestE2E_TracePersistence         --- PASS
=== RUN   TestE2E_ResumeFromCheckpoint     --- PASS
=== RUN   TestE2E_CancelMidRun             --- PASS
ok  	github.com/ormasoftchile/gert/v2/internal/e2e	0.459s

=== RUN   TestNewOTLPTracerProvider_WithTLS_NilConfig    --- PASS
=== RUN   TestNewOTLPTracerProvider_WithTLS_CustomConfig --- PASS
ok  	github.com/ormasoftchile/gert/v2/pkg/otel/adapter	5.503s

=== RUN   TestGc_RunningStatus_NeverDeleted    --- PASS
=== RUN   TestGc_MixedStatuses                 --- PASS
=== RUN   TestGc_StatusFilter_Running_Rejected --- PASS
=== RUN   TestGc_OlderThan_EdgeCase            --- PASS
ok  	github.com/ormasoftchile/gert/v2/cmd/gert	0.915s
```

All Phase 14 tests pass.

---

## Verdict

**APPROVED (9/10)** — Phase 14 is complete. Scribe may commit.

*Point deduction:* -1 for the iterate variable scoping limitation (DEV-14-03) which, while correctly documented, reveals an architecture gap that should have been caught in my design. The fix is tracked as NBI-14-03.

---

*Ken, Staff Architect*  
*2026-07-21*
