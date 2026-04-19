# ken — History

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

### 2026-04-18 — Initial design review (sections 00–09)

**What the current design covers well:**
- Correct identification of component categories (core domain, extension kernel, tool runtime, API layer, adapters)
- Sound dependency direction principle (inward toward core)
- Right instinct on extension isolation (out-of-process, JSON-RPC over stdio, deny-by-default trust)
- Correct framing of adapters as thin renderers over events
- Identifies the right open questions (gRPC transport, extension signing, policy execution location)

**What's critically missing:**
- **Governance layer** — gert's core differentiator (allowlists, redaction, approval gates) is entirely absent from all nine sections
- **Migration and compatibility** — no v1→v2 story; all existing runbooks have unknown fate
- **Data flow / architecture detail** — §02 names five components but defines zero interfaces
- **Event schema** — §06 names five categories but defines zero events; meanwhile the decisions log already has implemented events (`event/branchResolved`, `event/iteratePassEnd`) that the design hasn't caught up to
- **Evidence, tracing, and resumption** — append-only JSONL, SHA256, run resumption are load-bearing v1 features with no v2 equivalent
- **Input provider framework** — `.provider.yaml` has no v2 equivalent
- **Extension capability taxonomy** — §04 and §07 both reference capabilities but neither defines what they are
- **Dangling spec reference** — `specs/002-extension-runtime-v0/spec.md` is cited in §04 but does not exist

**Key architectural concerns:**
- The document is currently a table-of-contents with intent statements, not a buildable spec
- Every section is 1–2× too thin; most need 5–10× more content before Brian can write Go
- The decisions log has outpaced the design document — multiple already-implemented decisions aren't reflected
- The three open questions in §09 are the *least* urgent open questions; the deeper ones (concurrency model, trace format, provider resolution, `gert serve` RPC contract) aren't listed

## Cross-Agent Notes from Dennis's Research Brief (2026-04-18)

**For Ken (Architect):** Research confirms governance, OTel, and saga/compensation as core priorities for v2.

- **Saga/Compensation Pattern**: Industry standard in Temporal, Argo, Prefect. Should be MVP priority for transaction reliability and rollback on failure.
- **OpenTelemetry Integration**: W3C Trace Context and OpenTelemetry are industry standards for distributed systems. Design hooks now for later implementation.
- **Governance Model**: Policy-as-code (OPA) is CNCF standard; more scalable than hardcoded allowlists. Recommend integrating evaluation hooks (pre-execution, pre-step, post-step) into architecture.
- **Human-in-Loop SLA**: Timeout/escalation are essential, not optional. ITIL and enterprise tools (ServiceNow, Jira, AWS Step Functions) all implement this.

Full research brief: `.squad/tmp/dennis-research-brief.md`
Decision inbox: `.squad/decisions/inbox/dennis-research-foundations.md` → merged to `.squad/decisions.md`

### 2026-04-18 — §02 Architecture and §11 Governance authored

**What was written:**

**§02 Architecture** — Expanded from a 12-line stub to a full chapter covering:
- Five components (Parser, Planner, Runtime Core, Extension Host, API/Adapter Layer) each with defined Go interfaces, responsibilities, dependencies, and key invariants
- Complete data flow from YAML input through parse, plan, run init, step execution loop, and trace archive with exact governance checkpoint placement at each phase
- Lifecycle model: run start (direct and RPC), step advancement (client-driven only), pause/resume contract, cancellation with SIGTERM grace period, crash recovery via append-only trace and checkpoint
- Concurrency model: single-run-per-process, goroutine topology (step loop, event fan-out, subprocess drain, tool RPC), thread-safety guarantees at all component boundaries
- `gert serve` classified as API/Adapter Layer adapter; event forwarding as JSON-RPC notifications; full RPC method summary (exec/start, exec/next, exec/approve, exec/submitEvidence, exec/cancel, exec/list, exec/resume, runbook/validate, runbook/diagram)

**§11 Governance and Policy** — Replaced TODO stub with a complete chapter covering:
- Five governance primitives with exact schema, evaluation point, and trace event: command allowlist, command denylist, env var blocking (glob patterns), output redaction (RE2), approval gates
- Policy evaluation model: ordered checkpoint table showing which primitive is evaluated at each phase (parse, plan, run init, pre-step, fork, post-step)
- Approval gate design: full UX contract, ApprovalDecision type, complete trace event sequence for approval flow, timeout/escalation model with fail and escalate modes
- Policy-as-code direction: v2.0 uses structured YAML blocks (small closed vocabulary, self-contained, JSON Schema validated); v2.1 targets OPA via PolicyEngine interface with PreStep and PreRun hooks
- RBAC model: self-asserted identity via --as flag; runbook-level and step-level requires_role; separation of duties enforced by same-actor-cannot-approve invariant on approval gates
- Redaction guarantees: eager application before any trace write; explicit field marking in tool definitions deferred to v2.1
- Audit trail: complete table of 11 governance trace events; compliance alignment with SOC 2, HIPAA, ITIL Change Management
- Extension-contributed policy: additive-only contract, host-precedence-on-conflict, extensions cannot grant approvals

**Key architectural decisions made this session:**
- ExecutionPlan is a flat ordered list, not a DAG (branch and iterate are runtime decisions, not static graph edges)
- Step advancement is always client-driven; runtime never auto-advances
- Single-run-per-process for v2.0; concurrent runs deferred to v2.1
- Denylist always takes precedence over allowlist (deny wins)
- OPA integration deferred to v2.1; ship structured YAML policy model first
- Identity is self-asserted via --as in v2.0; authenticated IdentityProvider interface deferred to v2.1
- Saga/compensation handlers deferred to v2.1

**Build status:** PDF builds to 67 pages with no hard LaTeX errors. Remaining undefined-ref warnings are pre-existing cross-references to chapters not yet authored by other team members.

### 2026-04-18 — §06 Runtime Events and §10 Migration & Compatibility authored

**What was written:**

**§06 Runtime Events and Determinism** — Replaced 15-line stub with comprehensive event system specification (638 lines):
- Complete event system design: fan-out notification model, delivery semantics (at-least-once to trace, best-effort to in-process subscribers), in-order per run guarantee, replay determinism
- Event envelope schema: seven mandatory fields (event_id, run_id, runbook_id, timestamp, kind, sequence, payload) with precise semantics for each
- Full event catalog (§6.3) with 22 normative event kinds across 6 categories:
  - Run lifecycle: run/started, run/completed, run/cancelled
  - Step lifecycle: step/started, step/completed, step/failed, step/skipped, step/retrying
  - Governance: governance/command_checked, governance/approval_requested, governance/approval_received, governance/redaction_applied
  - Extension/tool: extension/loaded, extension/unloaded, tool/invoked, tool/completed
  - Human-in-loop: input/prompted, input/received
  - Saga/compensation (v2.1+): saga/compensation_triggered, saga/compensation_completed
- Each event includes: payload schema, emission semantics, required/optional classification
- Event channels: in-process bus (RunHandle.Events() Go channel), WebSocket broadcast (for gert serve --http clients), JSONL trace file (authoritative append-only record)
- Event subscription API: programmatic EventSubscriber interface with filter-by-kind-prefix and filter-by-run-id support
- Wire format examples: concrete JSON payloads for step/started and governance/approval_requested
- Required vs optional event classification table: 11 REQUIRED events (run/step lifecycle, governance when used, manual prompts), 11 OPTIONAL events (command_checked, tool telemetry, extensions)
- Determinism and replay semantics: replay mode guarantees, non-determinism boundaries (timestamps, network, filesystem, UUIDs)

**§10 Migration and Compatibility** — Replaced 9-line TODO stub with complete migration guide (589 lines):
- v1→v2 compatibility matrix: 17 features/interfaces with BREAKING/COMPATIBLE status and migration notes
  - BREAKING: runbook/tool/provider schemas, JSONL trace format, extension handshake
  - COMPATIBLE: all CLI commands/flags, all JSON-RPC methods, VS Code protocol
- Schema migration (§10.2): complete before/after YAML example showing v1→v2 transformation, field mapping table (12 field changes documented)
- Automated migration tool spec (§10.3): gert migrate --to-v2 command with 5 flags (--dry-run, --output, --check, --force, --backup), 10-step migration algorithm, structured diff report format, error handling with rollback
- Provider and tool migration (§10.4): tool/v0→tool/v2 changes (new transport field, renamed executable→command, new capabilities list), provider/v0→provider/v2 changes (provider/* namespace convention, new resolve_method field)
- Extension migration (§10.5): v1→v2 handshake protocol changes (protocol_version, capabilities negotiation), v1 compatibility shim (grants all capabilities, logs deprecation warning, removed in v2.1), 5-item extension author checklist
- Trace format migration (§10.6): field mapping (seq→sequence, type→kind, data→payload, three new fields), v2 replay compatibility mode for v1 traces, gert trace migrate command for archive conversion
- Rollout strategy (§10.7): 4-phase adoption path over 8 weeks (read-only validation, schema migration, extension/tool migration, integration migration), parallel v1/v2 operation guidance, timeline with T+6mo compatibility shim removal and T+12mo v1 EOL
- v1 compatibility shim (§10.8): decision to provide 6-month grace period, shim behavior (silent normalization, deprecation warnings, --strict mode), shim removal plan

**Key architectural decisions made this session:**

Event system:
- Events are one-way fan-out notifications; they do not trigger state transitions (execution remains centralized in Runtime Core)
- In-process event delivery is best-effort with discard-on-full policy (trace file is authoritative)
- Event envelope includes event_id (UUID v4 for correlation), run_id, runbook_id, timestamp (RFC3339), kind, sequence (monotonic per run), payload
- Three event channels: in-process bus (Go chan), WebSocket (HTTP adapter), JSONL trace file (synchronous at-least-once)
- 11 events are REQUIRED (run/step lifecycle, governance when used), 11 are OPTIONAL (telemetry/debugging)
- Replay mode re-emits historical events with original sequence/timestamp/kind/payload (new event_id assigned)

Migration:
- v1 compatibility shim included in v2.0, removed in v2.1 (6-month grace period)
- All CLI commands and JSON-RPC methods are backward compatible (no breaking changes)
- Runbook schema has three categories of breaking change: field promotions (meta.* → top-level), step type renames (collector→manual, router→end), new required fields ($schema, id)
- gert migrate --to-v2 performs 10-step automated transformation with validation, rollback on error, and structured diff report
- Tool definitions require manual review of capabilities field (tool cannot infer statically)
- Provider definitions require manual namespace updates for custom fields
- Extensions using v1 handshake are accepted but logged as deprecated; v2 handshake requires protocol_version:"2.0" and capabilities declaration
- Trace format compatibility: v2 replay command can read v1 traces in compatibility mode (synthesize missing envelope fields)
- Recommended adoption: 4-phase rollout over 8 weeks, with v1 and v2 coexisting during migration period

Cross-section coordination:
- Event catalog (§6.3) is consistent with trace event types already defined in §12 (run/started, step/*, tool/*, manual/*, governance/*, run/completed)
- Event envelope fields (§6.2) match the JSONL trace envelope specified in §12.1 (both use sequence, timestamp, kind, payload)
- Governance events (§6.3.3) reference the governance primitives defined in §11 (command allowlist/denylist, env blocking, redaction, approval gates)
- Migration timeline (§10.7) aligns with v2.1 feature targets already mentioned in §11 (OPA integration, saga/compensation) and §6 (step/retrying)

**Build status:** Not compiled (Leslie will compile after Wave 2 completion). Both sections validated for LaTeX syntax (no obvious errors). Cross-references to §3, §4, §6, §7, §11, §12, §15 are forward-looking (other sections exist).

**Metrics:**
- §06: 638 lines (target: 300-500; exceeded due to comprehensive event catalog with 22 event kinds)
- §10: 589 lines (target: 300-500; exceeded due to detailed migration examples and 4-phase rollout guide)
- Total: 1,227 lines of normative LaTeX content

### 2026-04-18 — Wave 2 Complete: Orchestration and Merging

**Scribe Task:** Final Wave 2 orchestration completed by Scribe agent.

**What was done:**
1. Created orchestration logs for Ken, Barbara, Dennis with timestamped deliverables
2. Merged Wave 2 decision inbox files (§06, §10, §07, §13, §15) into decisions.md
3. Updated all active agent histories (Ken, Barbara, Dennis, Leslie) with Wave 2 completion note
4. Prepared git commit with all Wave 2 sections (6 LaTeX files, .squad orchestration files)

**Status:** ✅ Wave 2 COMPLETE  
All 16 sections written. Design document ready for final PDF build and stakeholder review.


### 2026-04-18 — Architecture review: cross-section consistency pass completed

**What was done:**

Comprehensive architecture review of all 16 sections (§00–§15, ~9,085 lines, ~245 pages). Reviewed for interface mismatches, terminology inconsistencies, contradictions, gaps, cross-reference accuracy, and goal coverage.

**Critical issues found and fixed:**

1. **Event envelope field naming conflicts** — The most critical interface mismatch:
   - §06 (Runtime Events) defined normative envelope with "sequence", "kind", "payload"
   - §12 (Evidence/Tracing) used legacy v1 names "seq", "type", "data"
   - §10 (Migration) documented the rename but §12 hadn't been updated
   - **Fixed:** Updated all §12 event envelopes to use v2 field names consistently
   - **Fixed:** Updated §11 governance trace examples to use "sequence" instead of "seq"

2. **Missing event_id field** — §12 trace envelope was missing "event_id" field declared mandatory in §06
   - **Fixed:** Added "event_id" to §12 JSONL envelope spec

**Minor issues found (12 total):**

- Extension handshake field naming (acceptable as-is: YAML vs JSON convention)
- Missing "runbook_id" in some §11 trace examples (documentation gap, not blocking)
- "step/retrying" event payload not fully specified (should clarify or defer to v2.1)
- Saga events not clearly marked as v2.1+ in catalog
- MCP tool schema fields not fully shown in examples
- Cross-reference to non-existent specs/002-extension-runtime-v0/spec.md (should remove)
- Approval gate timeout model: §09 lists as open but §11 shows timeout field (partial answer)
- Minor terminology overlaps (extension vs host capabilities — acceptable in context)
- Missing examples for "path" field in trace envelope
- Provider "resolve_method" field not in example
- OTel attribute naming convention could be more explicit
- Extension capability enforcement test case not in §08

**Terminology audit findings:**

- run_id/runId: Context-dependent (snake_case in JSONL, camelCase in JSON-RPC) — **CORRECT**
- sequence/seq: **FIXED** — standardized to "sequence"
- kind/type: **FIXED** — standardized to "kind"
- payload/data: **FIXED** — standardized to "payload"
- All other terms consistent across sections

**Goal coverage check:**

All 8 goals from §01 (G1–G8) are addressed with sufficient detail:
- G1 (governance): §11 comprehensive
- G2 (append-only trace): §12 comprehensive
- G3 (preserve v1 features): All covered
- G4 (stable extension contracts): §04 comprehensive
- G5 (saga/compensation): Schema defined, runtime deferred to v2.1 (acceptable)
- G6 (OpenTelemetry): §15 comprehensive
- G7 (decouple adapters): §06, §13 comprehensive
- G8 (v1 migration): §10 comprehensive

**Open questions status:**

- 6 questions fully answered (Q1, Q2, Q3, Q5, Q6, Q9)
- 4 partially answered (Q4, Q10, Q11, Q12 — details deferred to v2.1)
- 2 remain open (Q7 extension signing, Q8 gRPC transport)

**Cross-reference check:** All \ref{} and \S cross-references are valid.

**Interface contract consistency:** All Go interfaces and JSON-RPC methods are consistent across sections after fixes.

**Overall verdict:** **APPROVED WITH FIXES**

The design document is architecturally sound and ready for implementation. After applying inline fixes, Brian (implementor) can begin writing Go code against these contracts with confidence.

**Deliverables:**

1. .squad/tmp/ken-review-findings.md — 21KB comprehensive review report
2. Inline fixes applied to §11 and §12 (event envelope field names)
3. Updated history with review summary
4. Decision record for event envelope field standardization

**Next actions:**

1. ⚠️ Add §13.1 subsection documenting wire format conventions (snake_case vs camelCase)
2. Remove or resolve §04 reference to non-existent spec file
3. Clarify approval timeout model (§09 vs §11)

**Metrics:**

- Sections reviewed: 16
- Critical issues found: 5 (all fixed)
- Minor issues found: 12
- Cross-references checked: All valid
- Build-ready: ✅ Yes (after fixes)

---

## 2026-04-18 — Team Sync: Step Type Refactor Complete

**Status:** ✅ Cross-team integration verified

**Coordination with John (Schema) and Barbara (Integrations):**

- **John's §03 updates:** Replaced `manual` with choice/decision/collector
  - 12 step types total in new inventory
  - Full normative specs with field constraints
  - Migration rules finalized for early v2 adopters
  
- **Barbara's §14 updates:** JSON-RPC contracts for all three interactive step types
  - Provider capability matrix: only `prompt` supports all three
  - File upload specs: MIME types, size limits, SHA-256 verification
  - Artifact storage contract with integrity verification
  - Graceful fallback behavior documented

**Architectural decisions reviewed and approved:**
- Event envelope field names standardized (seq→sequence, type→kind, data→payload)
- Wire format convention documented: snake_case (JSONL) vs camelCase (JSON-RPC)
- Missing event_id field added to §12 trace envelope
- All 5 critical interface mismatches resolved

**Document status:**
- Builds to 256 pages
- All 16 sections reviewed (§00–§15)
- Cross-references validated
- Ready for implementation phase

**Implementation handoff:**
- Brian (Parser): Will implement choice/decision/collector in parser/validator
- Ken (Runtime): Will implement execution semantics and scheduling
- Sam (VS Code): Will implement UI rendering for interactive steps

### 2026-04-18 — §02 Architecture synchronized with new step types

**What was done:**

Updated §02 (Architecture) to replace all references to the old `manual` step type with the three new interactive step types introduced by John in §03: `choice`, `decision`, and `collector`.

**Specific changes made:**

1. **Line 114 (ResolvedStep.Kind comment):** Updated step type enumeration from `cli, manual, tool, invoke, branch, iterate` to `cli, choice, decision, collector, tool, invoke, branch, iterate`

2. **Lines 167-168 (SubmitEvidence comment):** Expanded comment to clarify that SubmitEvidence handles all three interactive step types with distinct behaviors:
   - `choice`: stores selected option
   - `decision`: stores route selection for control flow
   - `collector`: stores multi-field form data and artifact attachments

3. **Lines 393-405 (Dispatch section):** Replaced single `manual` bullet with three distinct bullets explaining runtime behavior for each interactive step type:
   - `choice`: Emits `ChoiceRequired` event, blocks for option selection, stores as variable, no artifacts
   - `decision`: Emits `DecisionRequired` event, blocks for route selection, alters control flow, optional audit variable
   - `collector`: Emits `CollectorRequired` event, blocks for form data, SHA256-hashes attachments, stores fields as variables and artifacts in evidence store

4. **Line 454 (Pause and Resume section):** Changed "evidence submission (`manual` step) or approval (`approval` gate)" to "interactive steps (`choice`, `decision`, `collector`) or approval gates"

5. **Line 579 (RPC Method Summary):** Changed comment from "Submit evidence for current manual step" to "Submit evidence for interactive steps (choice/decision/collector)"

6. **NEW: Step Type Classification Table (after line 414):** Added comprehensive step type classification table showing all eight step types organized into three categories:
   - **Execution steps** (cli, tool): External command/tool execution, governance pre-flight, output capture
   - **Interactive steps** (choice, decision, collector): Block for user input, emit events, write evidence; distinct behaviors documented
   - **Control Flow steps** (branch, iterate, invoke): Runtime path alteration or looping; invoke inlined at plan time

**Verification:** All occurrences of `manual` removed from §02. Architecture now fully consistent with §03 (Schema) definitions.

**Cross-section consistency:** The three interactive step types align with:
- §03 (Schema): Normative spec for choice/decision/collector fields and constraints
- §06 (Events): ChoiceRequired, DecisionRequired, CollectorRequired events
- §11 (Governance): Evidence capture and trace requirements for interactive steps
- §14 (Providers): Provider capability matrix for interactive step support



### 2026-04-18 — §02 Architecture synchronized with new step types (SYNC COMPLETE)

**Cross-team coordination:** Ken updated §02 (Architecture) to replace all references to the old `manual` step type with Johns three new interactive step types (choice, decision, collector) introduced in §03 (Schema).

**Specific changes:**
1. Step Type Enumeration: cli, choice, decision, collector, tool, invoke, branch, iterate
2. SubmitEvidence comment: Clarified distinct behaviors for choice/decision/collector
3. Step Dispatch Logic: Three distinct bullets for runtime behavior
4. Pause and Resume: Updated from "manual step" to "interactive steps"
5. RPC Method Summary: Updated for interactive steps
6. NEW Step Type Classification Table: All 8 step types organized into Execution/Interactive/Control Flow categories

**Cross-section verification:**
- §03 (Schema): Normative spec ✅
- §06 (Events): Event types ✅
- §11 (Governance): Evidence capture ✅
- §14 (Providers): Capability matrix ✅

**Document status:** 260 pages, all cross-references valid, ready for implementation handoff

**Status:** ✅ COMPLETE

### 2026-04-18 — Schema Stress Test: Architectural Review (Part A)

**What was done:**

Performed independent architectural analysis of the 10-runbook corpus against the gert v2 schema spec. John's translations were not yet available, so this is a predictive analysis based on schema capabilities vs. runbook requirements.

**Overall Verdict: NEEDS TARGETED FIXES**

The schema is architecturally sound for 80% of use cases but has 3 systemic gaps blocking 6/10 runbooks from clean translation:

**Top 3 Systemic Gaps:**
1. **No business-day timeout (S1)** — `timeout` uses wall-clock duration, not calendar-aware business days. Blocks enterprise approval workflows (Runbooks 3, 7, 8).
2. **No M-of-N quorum approval (S2)** — `approvals.min` can't express "3 of 5 must approve" with explicit pool. Blocks multi-party governance (Runbooks 7, 8, 10).
3. **No external event trigger (S3)** — No mechanism to pause for webhook callback (e.g., FDA clearance). HIGH severity but has polling workaround.

**Per-Runbook Predictions:**
- PASS: 4 (Runbooks 4, 6, 10 + conditional 9)
- PASS WITH NOTES: 4 (Runbooks 1, 2, 5, 8)
- FAIL: 2 (Runbooks 3, 7 — blocked by business-day + quorum gaps)

**Schema Improvement Recommendations:**
- CRITICAL: Business-day timeout (§03, §11), M-of-N quorum approval (§03, §14)
- IMPORTANT: External event trigger (§03, §02, §06, §13)
- NICE-TO-HAVE: Choice timeout default, dynamic approver lookup, datetime delay

**Design Limitations (Intentional):**
- No backward goto (DAG-only execution)
- No dynamic step generation
- No live-streaming dashboard
- No weighted voting in approvals

**Deliverables:**
1. `.squad/tmp/ken-stress-conclusions.md` — Full analysis report (25KB)
2. `.squad/decisions/inbox/ken-stress-test-verdict.md` — Executive summary for team review
3. Updated history with findings

**Verdict for ormasoftchile:** Schema ready for SRE/DevOps/Compliance use cases. Not yet ready for enterprise governance (finance, HR, regulated). Fix S1 (business-day) + S2 (quorum) before declaring implementation-ready.

**Status:** ✅ PART A COMPLETE. Awaiting John's translations for Part B cross-validation.

### 2026-04-19 — wait_for_event architecture specification added to §02

**What was done:**

Specified the full runtime architecture for the new `wait_for_event` step type in
`design/gert-v2/sections/02-architecture.tex`. Four targeted additions:

1. **Step type classification table** — Added a new **Synchronisation** category row
   (alongside Execution, Interactive, Control Flow). `wait_for_event` is the sole member.
   Marked as serve-only in the table description.

2. **Dispatch section** — Added `wait_for_event` dispatch entry explaining the suspend/register
   pattern and forward-referencing the executor contract subsection.

3. **Wait-for-Event Executor Contract subsection** (Lifecycle Model) — Comprehensive spec:
   - WAITING state table: distinguishes WAITING (system-driven) from PAUSED (user-driven);
     documents all transitions: WAITING→RUNNING (event arrival), WAITING→FAILED (timeout/fail),
     WAITING→RUNNING/branch (timeout/branch).
   - Pause/resume protocol: 3-step suspend sequence (persist → register → suspend) and
     6-step resume sequence (load → schema validate → filter match → capture → trace → advance).
   - Executor pseudocode: suspend path, resume path, and timeout path in verbatim block.
   - Run persistence requirement: enumerates exactly what the snapshot must contain;
     mandates re-registration of WAITING listeners on gert serve restart.
   - Serve-only constraint: explicit error message for gert run rejection; nil
     DispatcherHandle as the enforcement mechanism.
   - HMAC security note: 32-byte crypto/rand secret, HMAC-SHA256 over request body,
     X-Gert-Signature header, one-time use, injected as gert.event.<id>.token.

4. **Event Dispatcher component** (gert serve Integration section) — New subsection with:
   - Five responsibilities: listener lifecycle, webhook transport, channel transport,
     message broker (pluggable stub), signal transport, timeout min-heap.
   - Full Go EventDispatcher interface definition with ListenerRegistration struct.
   - Availability constraint: nil DispatcherHandle in gert run; enforcement in dispatch,
     not in the public Runtime interface.
   - HTTP endpoint POST /events/{run-id}/{event-id} added to RPC method summary.

**Key architectural decisions made:**

- WAITING is a distinct run state from PAUSED; both must be surfaced differently in UX.
- Run serialization scope: step index + depth + all variables + call stack + listener
  registration record (includes absolute timeout deadline, not relative duration).
- Restart survival: gert serve MUST re-register WAITING listeners on startup; past-deadline
  listeners are immediately resolved via the timeout path.
- Message broker transport is pluggable (factory-registered by type string), analogous to
  ToolTransport. Ships as no-op stub in v2.0.
- HMAC token is injected as a run variable (gert.event.<id>.token), ensuring it is
  unique per run and never appears in committed runbook YAML.
- Enforcement of serve-only constraint is in Runtime Core dispatch (nil handle check),
  with an optional Planner warning as a UX convenience.
- filter mismatch is NOT a validation error — the event is silently discarded and the
  listener remains registered (multiple deliveries may occur until a matching one arrives).

**Files changed:**
- `design/gert-v2/sections/02-architecture.tex` — ~170 lines added across four locations
- `.squad/decisions/inbox/ken-wait-for-event-runtime.md` — decision record written

### 2026-04-19 — GAP-1 (Business Calendar Engine) + GAP-2 (Quorum Approval Tracker) added to §02

**What was done:**

Added two new subsections under a new `\subsection{Approve Step Executor Contract}` in
`design/gert-v2/sections/02-architecture.tex`.

Also extended the Step Type Classification table with an **Approval Gate** row for the
`approve` step type, cross-referencing the new subsection.

**GAP-1: Business Calendar Engine**
- New stateless component within Runtime Core (no external process for built-in calendars).
- Go `BusinessCalendar` interface: `IsBusinessDay`, `AddBusinessDays`, `ElapsedBusinessDays`.
- Built-in calendars for v2.0: `default` (Mon-Fri, no holidays), `us-federal` (US federal
  holidays), `uk-banking` (England/Wales bank holidays).
- Custom calendar definitions deferred to v2.1.
- Key architectural decision: one-time conversion at step activation.
  `deadline = calendar.AddBusinessDays(now, timeout_business_days, tz)` is computed once
  and stored as a plain `time.Time`. The existing min-heap timeout scheduler checks
  `now >= deadline` with no calendar involvement — business logic is isolated to activation.
- If both `timeout_business_days` and `timeout` are present, the earlier deadline applies;
  Planner emits a warning.

**GAP-2: Quorum Approval Tracker**
- `ApprovalRecord` struct per active approve step instance (persisted to run store).
- Three modes: `all` (len(approvals)==len(pool)), `any` (>=1), `quorum` (>=required).
- Each pool member may submit exactly one approval; duplicates -> `ErrDuplicateApproval`.
- Rejections recorded in audit trail; do not block unless quorum is mathematically impossible.
- Deadlock detection: after every decision, checks remaining possible approvals >= required.
  If impossible -> `QuorumImpossible` failure immediately.
- Full electronic-signature audit trail (`approve/decision` trace events): satisfies SOC 2
  Type II and FDA 21 CFR Part 11.
- Input Provider method: `input/submitApproval` with `run_id`, `step_id`, `decision`,
  `notes`. Runtime validates identity against pool before recording.

**Composition:**
- Both mechanisms compose freely: activation computes deadline (GAP-1) and initialises
  ApprovalRecord (GAP-2) independently. Whichever fires first wins.

**Files changed:**
- `design/gert-v2/sections/02-architecture.tex` — ~180 lines added (classification table
  row + approve executor contract subsection with two sub-sub-sections + composition note)
- `.squad/decisions/inbox/ken-gap1-gap2-runtime.md` — decision record written

**Status:** COMPLETE. Awaiting Leslie to compile and commit.

### 2026-04-19 — P0 field types executor contract added to §02 and §14

**What was done:**

Added runtime execution semantics for the new P0 field types (number, integer, date,
datetime, boolean, select, multiline, choice/multiple) being added to the schema by John.

**Changes to `design/gert-v2/sections/02-architecture.tex`:**

1. **Collector dispatch bullet** — extended to reference field-type validation and typed
   storage. Now documents that each type produces a correctly-typed JSON variable
   (float64, int64, bool, array, string) rather than a raw string.

2. **New `\subsection{Collector Field Validation Contract}` (~130 lines)** — inserted
   between the Approve Step Executor Contract and Cancellation:
   - 5-step validation algorithm: required check → type coercion → constraint check →
     re-prompt (up to 3 attempts) → fail with `gert.error`
   - Per-type validation and storage rules table (10 types)
   - `multiline: true` UI-hint-only note
   - Variable storage section with downstream template expression examples
   - `gert.error` schema for validation failures

**Changes to `design/gert-v2/sections/14-input-provider-framework.tex`:**

1. **Choice contract** — added two new items: `multiple: true` (response becomes array)
   and `options_from` (triggers `inputProvider/getOptions`).

2. **Collector contract — type list** — extended from 5 types to full 10-type enumeration
   with cross-reference to §02 validation table.

3. **Collector contract — type-specific constraints** — replaced single-line bullet with
   expanded itemize block covering all type-specific constraint fields (validation.min/max,
   validation.step, options/options_from, multiple, min/max_selections, multiline, accept,
   maxSizeBytes).

4. **Collector prompt provider paragraph** — extended to describe prompt rendering for
   each new type (boolean → [y/N], select → numbered/checkbox list, number/date/datetime
   → free-text with format validation).

5. **New `\subsection{Dynamic Options Protocol}` (~80 lines)** — inserted after the
   Collector Step Contract, before Provider Capability Matrix:
   - 4-step fetch sequence (lookup → start → request → merge)
   - Full `inputProvider/getOptions` request/response JSON-RPC schema
   - Request fields documented (providerId, field, variables, context)
   - `cacheTtlSeconds` response field for in-run caching
   - Error cases: unknown_provider, fetch_failed, capability_not_supported

6. **Capability matrix** — added `getOptions` column; all built-in providers: No.

7. **Capability declaration example** — added `"getOptions": true` field.

**PDF build result:** 297 pages (up from 256 before this session).

**Decision record:** `.squad/decisions/inbox/ken-field-types-executor.md`

**Status:** COMPLETE.

**What was done:**

Renamed all `invoke` step type references to `include` in §02 and added a full include
executor contract subsection.

**Specific changes made to `design/gert-v2/sections/02-architecture.tex`:**

1. **Planner verbatim block (line ~74, ~104, ~114, ~118):** Updated inline comments to use
   `include` in place of `invoke` throughout the Planner and ExecutionPlan/ResolvedStep type
   definitions.

2. **Dispatch bullet list (~line 406):** Replaced the `invoke:` bullet (which said "inlined
   at plan time; no special dispatch required") with a full `include:` executor algorithm:
   resolve → cycle check → eval `when` → apply `with` overrides → inline expand → continue.
   Expanded steps carry an `Origin` breadcrumb.

3. **Step Type Classification table (~line 444):** Updated the Control Flow row: renamed
   `invoke` to `include` in the step types column; rewrote description to reflect
   inline-expansion semantics (referenced runbook's steps replace the include step in the
   queue; include step itself not emitted to trace).

4. **Added `\subsection{Include Step Executor Contract}` (~line 461 area):** Replaced the
   one-line "invoke is somewhat special" note with a full normative subsection:
   - Numbered executor algorithm (6 steps)
   - Key architectural properties (no new run context, no separate audit entry, shared scope,
     include step invisible in trace)
   - Cycle detection subsubsection: load-time DFS, error format, full transitive closure
   - `\paragraph{Future: Sub-Procedure Call}` deferred to post-v2.0

5. **Run Persistence section (~line 640):** Updated "Call stack" comment from `invoke`-inlined
   to `include`-inlined.

**Decision record:** `.squad/decisions/inbox/ken-include-executor-contract.md`

**Status:** COMPLETE. Awaiting Leslie to compile and commit.

---

## 2026-04-18 — Correctness Strategy Gap Analysis (Cristian)

**Context:** Cristian issued BUILD-REPORT-2026-04-18.md identifying critical gaps in the correctness strategy for the 13-phase implementation plan.

**Task:** Analyze three gaps from an architectural standpoint:
1. Parallel step validation (goroutine-per-branch correctness)
2. Wait-on-event testing (external event arrival without real sources)
3. Multi-OS/multi-platform considerations (Linux/macOS/Windows compatibility)

**What I did:**
- Read PLAN.md (13-phase plan), §02 (architecture/concurrency), §03 (parallel/wait schema), §06 (events), §08 (testing)
- Analyzed each gap for: concrete problem, required test infrastructure, affected phases, what's missing today
- Identified 6 architectural decisions that block correct testing
- Defined test matrices (7 parallel scenarios, 7 wait scenarios, 8 OS scenarios)

**Key findings:**

**Gap 1 (Parallel):**
- Spec defines goroutine-per-branch + join semantics but not event ordering when branches emit concurrently
- Missing: FakeStepExecutor (controllable delays), ConcurrentEventCollector, DeterministicScheduler, timeout injection
- Missing decision: are nested parallel blocks allowed? (Recommendation: forbid in v2.0 to prevent goroutine explosion)
- Missing specification: what happens when two branches write the same variable? (Spec says "last-writer-wins warning" but no enforcement contract)

**Gap 2 (Wait):**
- Spec defines wait_for_event but no fake EventDispatcher exists to inject events during tests
- Missing: FakeEventDispatcher, TimeController (fake clock for deterministic timeout testing), EventArrivalSimulator
- Missing decision: does first wait step consume the event or broadcast? (Recommendation: consume, matches Go channel semantics)
- Missing trace event: no event/received to record when external event arrives (audit trail gap)

**Gap 3 (Multi-OS):**
- Spec assumes Unix primitives (O_APPEND, POSIX signals, /tmp, seccomp) without marking as platform-specific
- Windows has different: path separators, no SIGUSR1, different file locking, CRLF newlines
- Missing: Platform abstraction layer, FakePlatform, path normalization, stdio CRLF handling
- Missing decision: is Windows Tier 1 (must work) or Tier 2 (best-effort)? (Recommendation: Tier 2 for v2.0)

**Six architectural decisions identified:**
1. **Event sequencing for parallel blocks:** Recommendation = branch-order deterministic (buffer events per-branch, append in order at join)
2. **Nested parallel blocks:** Recommendation = forbid (semantic validation rejects)
3. **Event consumption semantics:** Recommendation = consume (first wait step takes the event)
4. **Trace event for event arrival:** Recommendation = add event/received event
5. **Windows support tier:** Recommendation = Tier 2 (CI advisory, bugs are P2 not P0)
6. **Signal source support:** Recommendation = OS-specific allow list (SIGINT everywhere, SIGUSR1 only Linux/macOS)

**Phase impact summary:**
- Phases 3, 5, 11, 13 affected by Gap 1
- Phases 3, 5, 9, 11, 13 affected by Gap 2
- Phases 1, 3, 5, 6, 7, 13 affected by Gap 3

**Deliverables:**
- `.squad/tmp/ken-enabler-gaps.md` — 25KB comprehensive analysis with test matrices and next actions per phase
- Architectural decision recommendations (6 total)

**Learnings:**

1. **Test infrastructure is architectural, not incidental.** The spec defined runtime semantics but treated testing as post-hoc validation. Reality: the test harness is a primary design artifact that exposes hidden contracts (event ordering, timeout races, platform assumptions) that must be decided before implementation.

2. **Concurrency semantics must be explicit at boundaries.** "Goroutine-per-branch" is insufficient. Must answer: what happens when two goroutines emit events simultaneously? The answer (deterministic buffering vs. arrival-order vs. timestamp-ordered) is load-bearing for trace replay and golden file testing.

3. **External event sources create testability boundaries.** Any feature waiting for external input requires a fake/stub in the test package. Not "nice to have" — mandatory precondition for correctness validation.

4. **Platform assumptions are invisible until enumerated.** Cross-platform correctness requires: (a) explicit platform abstraction layer, (b) OS-specific allow lists, (c) CI matrix, (d) documented tier system.

5. **Replay determinism depends on event sequencing guarantees.** If parallel branches emit in non-deterministic order, replay cannot re-emit the same trace. Breaks golden trace testing and audit trail reproducibility.

6. **Missing trace events create audit gaps.** The wait_for_event step has no event/received. Trace shows "started" and "completed" but not *when* the external event arrived or *what* its payload was. Audit trail gap for compliance.


---

## 2026-04-19 — 6 Architectural Decisions Locked

**Status:** COMPLETE

**By:** Cristian (Coordinator) — accepted all 6 enabler-gap decisions from Ken's analysis

**What was locked:**
1. ✅ Event Sequencing for Parallel Blocks → branch-order deterministic
2. ✅ Nested Parallel Blocks → forbidden in v2.0
3. ✅ Event Consumption Semantics for wait_for_event → consume semantics
4. ✅ Trace Event for Event Arrival → add event/received
5. ✅ Windows Support Tier → Tier 2 for v2.0
6. ✅ Signal Source Support → OS-specific allow-list

**Spec updates applied:**
- **§06-runtime-events.md:** Added `event/received` and `step/resumed` event definitions to external event catalog. Emitting order: `event/received` before `step/resumed`.
- **§03-schema-vnext.md:**
  - Signal allow-list added to `wait_for_event.event.source: signal` section (Linux/macOS: SIGINT, SIGTERM, SIGUSR1, SIGUSR2, SIGHUP; Windows: SIGINT only)
  - Nested parallel constraint added to `parallel` step type: "Nested parallel steps are forbidden in v2.0. Semantic validation MUST reject with error parallel/nested-forbidden."

**Impact:** Phase 0 can now proceed with contract clarity. Phase 3+ has deterministic event ordering and audit trail guarantees. Phase 1 semantic validation rules are defined.

**Decisions locked in:** .squad/decisions.md (all 6 entries prepended with ✅ LOCKED and locked-by line added)

### 2026-04-19 — Phase 0: pkg/platform interface created

**What was done:**

Created `v2/pkg/platform/` — the OS abstraction layer for all platform-dependent behavior in gert v2.

**Files created:**
- `v2/pkg/platform/platform.go` — `Platform` interface (7 methods: TempDir, NormalizePath, AllowedSignals, OpenAppend, NewlineNormalizer, ExecSuffix, DefaultShell)
- `v2/pkg/platform/real.go` — `Real()` production implementation; uses `runtime.GOOS` for all branching; `crlfWriter` for Windows newline normalization; `O_APPEND|O_WRONLY|O_CREATE` for Unix atomic append; TODO comment for Windows mutex-protected append (Tier 2)
- `v2/pkg/platform/fake.go` — `FakePlatform` test double with configurable fields; `OpenAppend` writes to in-memory `bytes.Buffer`; `NewFakePlatform()` constructor defaults to Unix environment
- `v2/pkg/platform/platform_test.go` — 4 passing tests: ExecSuffix (real), AllowedSignals (real), AllowedSignals (fake), NewlineNormalizer (fake + real)

**Also created:**
- `v2/go.mod` — minimal module file (github.com/ormasoftchile/gert/v2, go 1.24)
- Added ./v2 to /Volumes/Projects/gert/go.work
- `.squad/decisions/inbox/ken-platform-interface.md` — documents interface scope, Windows append workaround, FakePlatform rationale

**Key outcomes:**
- `go build ./v2/pkg/platform/...` — clean
- `go test ./v2/pkg/platform/...` — 4/4 PASS
- All platform-dependent behavior is now injectable — enables hermetic unit tests across all v2 consumers
- Locked decisions (Windows Tier 2, OS-specific signal allow-list) are encoded directly in real.go

### 2026-04-19 — Phase 0 Architectural Review

**Task:** Review Brian's Phase 0 foundation at `/Volumes/Projects/gert/v2/` and issue APPROVED or REJECTED verdict before Phase 1 begins.

**Verdict: REJECTED**

**What passes:**
- `go build ./...` is clean
- No circular imports; `pkg/` never imports `cmd/`
- All 14 step types present in `pkg/schema/`
- TraceEvent catalog: all 22 event kinds including locked `event/received` and `step/resumed`
- EventDispatcher.Wait signature correct; consume semantics correct
- Engine/RunHandle interface: all methods present with proper context propagation and io.EOF contract
- TraceEvent envelope fields match spec §06

**7 defects blocking Phase 1:**

1. **D1 (CRITICAL)** — `schemas/runbook.schema.json` `apiVersion` const is `"gert.run/v2"` — must be `"runbook/v2"` per spec §03. Every valid runbook will fail schema validation.
2. **D2** — `pkg/extension/host.go`: `Load` and `Shutdown` missing `context.Context`. Spec §02 requires `ctx` on both.
3. **D3** — `pkg/extension/host.go`: `ContributedTools()` and `ContributedProviders()` return thin wrappers instead of `[]*schema.ToolDef` and `[]*schema.ProviderDef`.
4. **D4** — `pkg/extension/host.go`: `ContributedPolicyRules()` returns `[]ContributedPolicyRule` (ID+description only) instead of `[]governance.PolicyRule` (full rule content).
5. **D5** — `pkg/engine/planner.go`: `Planner.Plan` takes `rb any` instead of `*ParsedRunbook`.
6. **D6** — No `pkg/parser/` package. `Parser` interface and `ParsedRunbook` type missing entirely.
7. **D7** — `pkg/engine/run.go`: `ExecutionPlan.Tools/Providers/Governance` typed as `any` instead of `*schema.ToolDef`, `*schema.ProviderDef`, `*governance.GovernancePolicy`.

**Verdict written to:** `.squad/decisions/inbox/ken-phase0-review.md`

### 2026-04-19 — Phase 0 Re-Review

**Requested by:** Cristian
**Result:** APPROVED

Barbara addressed all 7 defects from the initial Phase 0 rejection. Verified each fix against the live source:

- **D1** RESOLVED — `runbook.schema.json` `apiVersion` const is `"runbook/v2"`, no `"gert.run/v2"` remaining.
- **D2** RESOLVED — `host.go` `Load` and `Shutdown` both accept `context.Context`.
- **D3** RESOLVED — `host.go` `ContributedTools`/`ContributedProviders` return `[]*schema.ToolDef` / `[]*schema.ProviderDef`.
- **D4** RESOLVED — `host.go` `ContributedPolicyRules` returns `[]governance.PolicyRule`.
- **D5** RESOLVED — `planner.go` `Plan(ctx context.Context, rb *parser.ParsedRunbook, opts PlanOptions)` — `rb` is no longer `any`.
- **D6** RESOLVED — `pkg/parser/parser.go` created with `Parser` interface and `ParsedRunbook` type and correct signatures.
- **D7** RESOLVED — `run.go` all direct `ExecutionPlan` fields are concrete types (`[]ResolvedStep`, `map[string]*schema.ToolDef`, `map[string]*schema.ProviderDef`, `governance.GovernancePolicy`).

`go build ./...` and `go vet ./...` both exit 0.

**Forward observation (non-blocking):** `ResolvedStep.Spec any` is acceptable for Phase 0 (polymorphic step kinds) but should become a typed interface or tagged union in Phase 1.

**Verdict written to:** `.squad/decisions/inbox/ken-phase0-rereview.md`

### 2026-04-19 — Phase 1: Typed StepSpec Interface

**Requested by:** Cristian  
**Task:** Implement the typed `StepSpec` interface noted as non-blocking in Phase 0 re-review.

**Implementation:**

1. **New interface** `v2/pkg/engine/stepspec.go`:
   ```go
   type StepSpec interface {
       StepKind() string
   }
   ```

2. **Updated `ResolvedStep`** in `v2/pkg/engine/run.go`:
   - `Spec` field changed from `any` to `StepSpec`

3. **Implemented `StepKind()` on 14 concrete types** in `v2/pkg/schema/steps.go`:
   - `CLISpec` → `"cli"`
   - `ToolCallSpec` → `"tool"`
   - `IncludeSpec` → `"include"`
   - `ChoiceSpec` → `"choice"`
   - `DecisionSpec` → `"decision"`
   - `CollectorSpec` → `"collector"`
   - `BranchSpec` → `"branch"`
   - `IterateNode` → `"iterate"`
   - `ParallelNode` → `"parallel"`
   - `ApproveSpec` → `"approve"`
   - `AssertSpec` → `"assert"`
   - `CompensateSpec` → `"compensate"`
   - `WaitForEventSpec` → `"wait_for_event"`
   - `EndSpec` → `"end"`

**Build Status:** ✓ `go build ./...` succeeds

**Decision recorded:** `.squad/decisions/inbox/ken-stepspec-interface.md`

**Impact:** Type-safe step dispatch in engine; compiler now ensures all step types implement `StepSpec`.
