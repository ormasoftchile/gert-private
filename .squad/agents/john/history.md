# john — History

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

- `type: extension` does not exist as a step type in gert v2. Extension is a field-annotation
  convention only (`x-<namespace>:` prefix on runbook or step objects). Using `type: extension`
  as a catch-all step type was incorrect.
- **Outbound notify/send** steps (Slack, PagerDuty, email, external API calls) belong as
  `type: tool` with the tool declared in `toolRefs`. This already works in the v2 schema.
- **Inbound event receive** steps (wait for webhook, wait for SIEM alert, wait for callback)
  are now covered by `type: wait_for_event` (GAP-3 resolved 2026-04-19). The step pauses
  execution, opens an inbound endpoint, captures payload fields into run variables, and
  resumes when the event arrives. Timeout + on_timeout control fallback behavior.
  Key fields: `event.source` (webhook/message/signal/channel), `event.id`, `event.filter`,
  `event.payload_schema`, `capture`, `timeout`, `on_timeout`.

## Cross-Agent Notes from Ken's Architectural Review (2026-04-18)

**For John (Schema):** §03 Schema vNext has critical gaps. Needs concrete starting point with:

1. **Field inventory** — What are the core language fields in v2? How do they differ from v1? (retained, removed, breaking-changed, new)
2. **Namespace convention** — For extension fields, what is the exact convention? (just "namespaced" is not enough)
3. **Versioning strategy** — What is `apiVersion: runbook/v2`? What breaks from v1?
4. **Migration path** — Must specify how `gert migrate` handles `runbook/v0` and `runbook/v1`
5. **Validation split** — Define what "structural" vs. "semantic" validation means concretely
6. **Step types** — Do cli, tool, manual, invoke, branch, iterate survive unchanged? Any new types? Removed types?
7. **Expression language** — Go templates currently — staying in v2?
8. **Tool schema vNext** — Does tool definition format change in v2?
9. **Provider schema** — How are providers defined in v2?
10. **Extension registration** — How are namespaced schema extensions registered and validated?

Ken's recommendation: Treat §03 as "needs expansion 3–5×". This is a blocker for Brian's parser implementation and validation engine. See full gap analysis at `.squad/tmp/ken-gap-analysis.md` (§03 section, lines 70–90). Decisions documented at `.squad/decisions.md` (search "Ken's Gap Analysis Findings").

## Cross-Agent Notes from Dennis's Research Brief (2026-04-18)

**For John (Schema):** Research confirms schema self-description and semantic versioning as industry standards.

- **Self-Describing Schemas**: JSON Schema best practices recommend `$schema` field. Used in GitHub Actions, Argo, Kubernetes CRDs. Should be added to all runbook/tool/provider YAML files in v2.
- **Semantic Versioning**: Clear compatibility guarantees (major.minor.patch) are industry standard. Enables automated compatibility testing (old runbooks on new runtime).
- **Schema Embedding**: $schema field enables IDE tooling, validation, and version negotiation without external configuration.
- **Migration Path**: Automated compatibility testing needed for v1→v2 migration.

Full research brief: `.squad/tmp/dennis-research-brief.md`
Decision inbox: `.squad/decisions/inbox/dennis-research-foundations.md` → merged to `.squad/decisions.md`

### §03 Schema vNext — Complete Rewrite (2026-04-18)

Performed a full rewrite of design/gert-v2/sections/03-schema-vnext.tex,
expanding the 3-bullet stub to a 1,495-line normative specification.

Work done:
- Audited v1 schema: read pkg/schema/schema.go (all 823 lines), v1 runbook
  examples, tool definition examples, Ken gap analysis, Dennis research brief.
- Reviewed all decisions in .squad/decisions.md related to schema, versioning,
  and migration.

Key design decisions made:
1. apiVersion: runbook/v2 / tool/v2 / provider/v2 as canonical version strings.
2. Promote meta.* to top-level fields - breaking change from v1.
3. Schema field added to all document kinds; URLs at https://schemas.gert.dev/
4. v2 parser reads runbook/v1 in compatibility mode; gert migrate rewrites to v2.
5. toolRefs replaces flat tools string array with typed path/alias overrides.
6. type: manual replaces type: collector; type: end replaces type: router.
7. type: branch is now a first-class step type.
8. New step type: type: compensate - saga/compensation pattern.
9. New step type: type: parallel - fan-out/fan-in with join semantics.
10. Expression language: Go templates retained; standard function library added.
11. Extension namespace: x-namespace prefix pattern.
12. Validation split: Phase 1 = JSON Schema structural; Phase 2 = 10 semantic rules.
13. gert schema export --kind bundle exports all three schemas.
14. Tool definition v2 adds capture.format, per-action contract, per-action governance.
15. Provider definition v2 adds typed fields map for binding resolution.
16. Input declaration gains type field and richer from bindings.
17. Output declarations are now first-class top-level field.
18. Approval gates gain timeout, on_timeout, and escalate_to fields.

Gaps resolved from Ken analysis: All 10 gaps in ken-gap-analysis.md section 03 addressed.

### Step Type Refactor: manual → choice/decision/collector (2026-04-18)

Replaced the overly-generic `manual` step type with three precise types based on
user directive identifying conceptual error in the design. This resolves a
semantic clarity gap in the step type inventory.

**Changes made:**
1. Removed `type: manual` entirely from the spec.
2. Added `type: choice` — user selects from predefined options, stores result in variable.
   - Fields: prompt, options (label/value/hint), variable, default
   - Example: environment selection (staging/production/canary)
3. Added `type: decision` (also accepts `router` as alias) — user picks execution path.
   - Fields: prompt, routes (label + runbook or goto), variable (for audit)
   - Example: incident response fork to different runbooks
4. Redefined `type: collector` — user provides unstructured input (text/files/images).
   - Fields: prompt, fields (name/type/label/required/hint), approvals
   - Field types: text, multiline, file, image, url
   - Example: incident evidence collection with approval gates
5. Updated migration table (lines 139-142): documented the split.
6. Updated gert migrate behavior (line 179): replaced manual rename with precise split.

**Rationale:**
- `manual` conflated three distinct concepts: value capture, control flow, and data collection.
- Each new type has a single, well-defined responsibility.
- Implementation clarity: parsers/runtimes know exactly what UI to render and what semantics to enforce.
- Better validation: each type has specific field constraints (e.g., decision routes must have runbook XOR goto).

**No interaction with saga pattern:** The three new types work with `compensate` the same
as any other step type - compensation is orthogonal to user interaction.

**Step type inventory after refactor:**
1. cli
2. choice (new)
3. decision (new)
4. collector (redefined)
5. tool
6. invoke
7. branch
8. iterate
9. parallel
10. assert
11. compensate
12. end

---

## 2026-04-18 — Team Sync: Step Type Refactor Complete

**Status:** ✅ Merged to decisions.md

**Cross-team coordination:**
- **Barbara** (Integrations): Updated §14 Input Provider Framework with JSON-RPC contracts for choice/decision/collector
  - Provider capability matrix: only `prompt` supports all three types
  - Fallback behavior: engine uses `prompt` if configured provider doesn't support step type
  - File upload specs: MIME types, size limits, SHA-256 verification
  - Artifact storage contract finalized

- **Ken** (Architecture): Completed comprehensive cross-section review (§00–§15)
  - Event envelope field names standardized (seq→sequence, type→kind, data→payload)
  - Added missing event_id field to §12 trace envelope
  - Wire format convention documented: snake_case (JSONL traces) vs camelCase (JSON-RPC)
  - 5 critical issues fixed; 12 minor documentation gaps identified

**Document status:** Builds to 256 pages; PDF ready for implementation phase

**Next: Implementation begins**
- Brian (Parser): Implement parser/validator for choice/decision/collector
- Ken (Runtime): Implement execution semantics for three step types
- Sam (VS Code): Implement UI rendering for interactive steps

---

## 2026-04-18 — Validation Methodology for Schema Stress Testing

**Requested by:** ormasoftchile  
**Output:** `.squad/tmp/john-validation-methodology.md` (38KB, 600 lines)

Created a rigorous methodology for translating prose runbooks to gert v2 schema and validating completeness and fidelity. This provides a systematic framework for stress-testing the schema against real-world operational procedures.

**Methodology components:**

1. **Translation Protocol** — Step-by-step instructions for prose→schema translation:
   - Step boundary identification (when does a new step begin?)
   - Step type classification (12-question decision tree)
   - Branching patterns (automatic vs. human-driven)
   - Data flow modeling (inputs, captures, variables, scope)
   - Failure/compensation paths (saga pattern, retry, continue-on-fail)
   - Human interaction patterns (choice/decision/collector/approvals)
   - Parallelism (fan-out/fan-in with join semantics)
   - Nested runbooks (invoke with imports/outputs)

2. **Completeness Criteria (C1–C10)** — Binary yes/no checks:
   - C1: Step coverage (every action has a step)
   - C2: Decision points (all branches/choices represented)
   - C3: Data flow (all variables traced source→consumer)
   - C4: Timing constraints (delays/timeouts/retries/polling)
   - C5: Failure paths (error handling and compensation)
   - C6: Human interaction (correct step types used)
   - C7: Parallelism (explicitly declared where prose says "concurrent")
   - C8: Nested runbooks (imports/invoke)
   - C9: Governance (approval gates, effects, roles)
   - C10: Terminal outcomes (all end states declared)

3. **Fidelity Criteria (F1–F10)** — Semantic preservation checks:
   - F1: Semantic equivalence (execution produces same outcomes)
   - F2: No information loss (all behavioral details preserved)
   - F3: Execution path preservation (all possible paths present)
   - F4: Variable binding correctness (no unbound reads)
   - F5: Step type precision (most specific type used)
   - F6: Timing preservation (values match prose)
   - F7: Governance alignment (approval/risk settings match)
   - F8: Human prompt clarity (interactive steps are unambiguous)
   - F9: Deterministic evaluation (conditions don't depend on hidden state)
   - F10: Artifact integrity (evidence capture uses proper mechanisms)

4. **Gap Classification (G1–G6):**
   - G1: Missing step type (action has no direct step type)
   - G2: Missing field (step type exists but lacks required field)
   - G3: Missing flow construct (control flow pattern not expressible)
   - G4: Missing interaction model (human pattern not supported)
   - G5: Semantic loss (expressible but meaning degraded)
   - G6: Verbosity/workaround (expressible but awkwardly)

5. **Scoring Rubric:**
   - Completeness Score = satisfied criteria / 10 × 100%
   - Fidelity Score = satisfied criteria / 10 × 100%
   - Gap inventory with severity ratings
   - Composite verdict: PASS / PASS WITH NOTES / FAIL
   - PASS requires: ≥90% on both scores + no CRITICAL gaps

6. **Schema Improvement Signals:**
   - Gap aggregation across multiple runbooks
   - Threshold-based triggers for schema extensions
   - Distinction between fixable gaps vs. acceptable design limitations
   - Workflow for: triage → design → prototype → validate → document

**Production readiness criteria defined:**
- 10+ diverse runbooks translated
- Average completeness ≥95%
- Average fidelity ≥95%
- Zero CRITICAL gaps
- All HIGH gaps have workarounds

**Includes:**
- Two worked examples (one PASS, one PASS WITH NOTES)
- Step type quick reference table
- Printable validation checklist
- Scorecard template (markdown table format)

**Purpose:** This methodology gives us a binary answer to "can gert v2 express this runbook?" and provides actionable feedback for schema improvements. It bridges the gap between normative spec and real-world stress testing.

**Decision inbox:** `.squad/decisions/inbox/john-validation-methodology.md` created for team review.


---

## 2026-04-18 — Schema Validation: 10-Runbook Translation Corpus

Translated all 10 runbooks from Dennis corpus into gert v2 YAML with rigorous validation.
Results: 1 PASS, 7 PASS WITH NOTES, 2 FAIL. Average 82% completeness, 73% fidelity.
Schema NOT production-ready (4 CRITICAL gaps).

Top 3 gaps: calendar-aware timeouts, dynamic approver resolution, cross-branch parallelism.
Most problematic runbooks: R7 (financial approval), R8 (FDA release) — both FAIL.

Output files:
- /Volumes/Projects/gert/.squad/tmp/john-schema-translations.md
- /Volumes/Projects/gert/.squad/tmp/john-translations-summary.md


---

## 2026-04-18 — Persist 10 Stress-Test Runbooks as Dev Fixtures

**Requested by:** ormasoftchile  
**Output:** design/gert-v2/testdata/runbooks/ (31 files)

Persisted all 10 runbooks from Dennis corpus as permanent dev fixtures.
Each directory: source.md, schema.yaml, assessment.md.
R1-R3: verbatim from john-schema-translations.md. R4-R10: new best-effort translations.
Verdicts: 1 PASS, 7 PASS WITH NOTES, 2 FAIL (R7, R8).

---

## 2026-04-19 — Added type:wait_for_event Step Type (GAP-3)

**Requested by:** ormasoftchile
**Output:** sections/03-schema-vnext.tex (new subsection), testdata r01 + r05 updated

Resolved GAP-3 from the schema stress test. `type: wait_for_event` is now a
first-class step type in §03, positioned between `tool` and `invoke`.

**Spec added:**
- Pause/resume model: runtime opens inbound endpoint, suspends run record,
  resumes on matching event, captures payload fields into variables.
- 7 fields: event.source, event.id, event.filter, event.payload_schema,
  capture, timeout, on_timeout.
- 2 YAML examples: Prometheus alert webhook (r01-k8s-incident context) and
  named channel approval callback (generic pattern).
- Transport Note: 4 source types (webhook/message/signal/channel), endpoint
  available as `{{ .gert.event.<id>.endpoint }}`.
- Security Note: one-time HMAC token per run, available as
  `{{ .gert.event.<id>.token }}`.
- Step type summary table added at top of Section 6 (Step Types v2 Inventory)
  listing all 13 types with section refs.

**Testdata updated:**
- r01-k8s-incident/schema.yaml: detect_alert step converted from cli stub to
  wait_for_event with Prometheus webhook source.
- r05-security-breach/schema.yaml: receive_alert step converted from cli stub to
  wait_for_event with SIEM webhook source.

**Decision inbox:** .squad/decisions/inbox/john-wait-for-event-spec.md


---

## 2026-04-19 — GAP-1 + GAP-2 Resolved: Business-Day Timeouts and M-of-N Quorum

**Requested by:** ormasoftchile
**Output:** sections/03-schema-vnext.tex (new subsection), testdata r07 + r08 updated

Resolved GAP-1 (business-day timeouts) and GAP-2 (M-of-N quorum approval) from
the schema stress test. Both gaps were blocking R7 (Financial Approval) and R8
(FDA Release) from passing validation.

### GAP-1: Business-Day Timeout

Added three new fields to the approve step type:
- timeout_business_days (integer) — timeout in business days; mutually exclusive with timeout
- timezone (string) — IANA timezone name; required when timeout_business_days is set
- business_calendar (string) — named calendar ID (e.g. us-federal, uk-banking); defaults to Mon-Fri no holidays

Runtime counting rules: clock starts when step enters waiting state; weekends + holidays per
calendar do not count; day boundaries determined by timezone.

### GAP-2: M-of-N Quorum Approval

Extended the approvals block with three new fields:
- approvals.mode (string) — all (default), any, or quorum
- approvals.pool (string[]) — eligible roles for quorum; mutually exclusive with roles
- approvals.required (integer) — approvals needed from pool; must be >= 1 and <= len(pool)

Quorum semantics: runtime accepts approvals from any pool member; step proceeds once required
is reached; rejections are recorded and may trigger automatic failure if quorum becomes
unreachable.

### New Step Type: type: approve

Created a new first-class subsection{Step Type: approve} in Section 03 (between collector and tool).
The approve step is a standalone approval gate with no data-collection fields.
Step type inventory updated from 13 to 14 types.

### Validation Rules Documented (6 rules)
1. timeout and timeout_business_days mutually exclusive
2. timeout_business_days requires timezone
3. mode: quorum requires both pool and required
4. required must be >= 1 and <= len(pool)
5. roles and pool mutually exclusive
6. mode: all / any requires roles (not pool)

### Testdata Updated
- R7 Financial Approval: 7 approval steps converted from wall-clock hours to timeout_business_days;
  board approval (step 7b) converted from min:3/roles:[board-member] to mode:quorum/pool:[5 named members]/required:3;
  assessment updated FAIL => PASS WITH NOTES
- R8 FDA Release: 7 approval steps converted from wall-clock hours to timeout_business_days;
  assessment updated FAIL => PASS WITH NOTES (CRITICAL gaps G4-001 and G2-001 remain)

**Decision inbox:** .squad/decisions/inbox/john-gap1-gap2-approve.md

---

## 2026-04-19 — type:invoke Renamed to type:include

**Requested by:** ormasoftchile
**Output:** sections/03-schema-vnext.tex, testdata r04 + r09 updated, decision inbox written

Renamed the step type `invoke` to `include` throughout the schema spec and all testdata runbooks.

### Semantic Clarifications Added to §03

- **Inline expansion semantics**: `type: include` expands another runbook's steps directly into the caller's run path — not a sub-procedure call.
- **Shared variable space**: The included runbook shares the caller's variable scope; no input/output mapping needed or supported.
- **Audit trace**: Included steps appear inline in the run trace and audit log, with no sub-run or separate audit entry.
- **Cycle detection**: The runtime builds a runbook dependency graph at load time; cyclic includes (direct or transitive) are a hard validation error with the full chain reported (`cyclic include detected: A → B → A`).
- **Future type:call**: Isolated scope with explicit I/O mapping is deferred to a future `type: call` step type (post-v2.0).

### Field Table Changes

Old `invoke` fields removed: `invoke.inputs`, `invoke.outputs`, `invoke.gate`.
New `include` fields: `include.runbook` (required), `include.with` (variable overrides, optional), `include.when` (skip condition, optional).

### Files Changed

- `sections/03-schema-vnext.tex`: subsection renamed, prose rewritten, field table updated, YAML example replaced, two new paragraphs (Cycle Detection, Difference from Sub-Procedure Call), inventory table updated, validation rules updated, Output Declarations section updated, runbook `id` field prose updated, `composable` kind description updated.
- `testdata/runbooks/r04-soc2-evidence/schema.yaml`: 7x `type: invoke` to `type: include`, `invoke:` to `include:`, `inputs:` to `with:`, added NOTE comment.
- `testdata/runbooks/r09-oncall-escalation/schema.yaml`: 1x `type: invoke` to `type: include`, `invoke:` to `include:`, `inputs:` to `with:`, added NOTE comment.

**Decision inbox:** .squad/decisions/inbox/john-invoke-renamed-include.md

---

## 2026-04-19 — Collector Field Types P2: ephemeral, autocomplete, format

**Requested by:** ormasoftchile
**Output:** sections/03-schema-vnext.tex, testdata r03 updated, decision inbox written

Implemented all three P2 field-type improvements to §03 Schema vNext and updated
testdata runbooks where applicable.

### P2-A: `ephemeral: true` attribute

- Added `ephemeral` row to the field structure table (default: false).
- Added new paragraph "Ephemeral fields" with full semantics:
  - Value IS bound to variable space and usable in downstream templates.
  - Value is NEVER written to JSONL audit log (`[REDACTED]` placeholder).
  - Value is NEVER persisted to any store; in-memory only for run duration.
  - If run is suspended and resumed, ephemeral values are gone; re-collect required.
- Added ephemeral + type: file/image validation warning rule.
- Added security note: ephemeral fields reduce audit footprint but do not prevent
  downstream transmission; authors must ensure secure contexts.
- Added YAML example with deploy credentials collector step.
- Testdata: no existing collector fields matched secret-name criteria (token/password/
  key/secret/credential); no testdata changes for P2-A.

### P2-B: `type: autocomplete` field type

- Updated `options_from` description in field structure table to cover both `select`
  and `autocomplete`.
- Updated `multiple` description to cover both `select` and `autocomplete`.
- Updated field type inventory: nine → ten field types; added `autocomplete` row.
- Added subsection "Field type: autocomplete" with:
  - `options_from` extended table (provider, field, min_chars, debounce_ms).
  - CLI/non-interactive graceful degradation note.
  - YAML examples (live service search, multi-select user lookup).
- Added validation rules: `options_from` required (provider + field); `options` not
  valid on autocomplete; min_chars and debounce_ms range constraints.
- Testdata r03: `manager` field converted from `type: text` to `type: autocomplete`
  with `options_from.provider: hr-directory` (previously had hint "Use autocomplete
  from employee directory"). Score 9/10 → 10/10; verdict PASS.

### P2-C: `format` validation for multiline text

- Updated text validation table from 4 to 5 sub-fields; added `format` enum row.
- Added paragraph "Structured multiline format validation" with:
  - format: yaml / json / toml (toml deferred with warning note).
  - Re-prompt behavior on parse failure.
  - Stored value is always raw string (parsing is validation-only).
  - Note on fromYAML/fromJSON template functions for downstream traversal.
  - YAML examples (config_patch, event_payload).
  - Warning: format ignored when multiline is false.
- Added validation rules: format on non-multiline is warning; toml is warning in v2.0.
- Testdata: no existing multiline fields are clearly YAML/JSON (all are prose);
  no testdata changes for P2-C.

**Decision inbox:** .squad/decisions/inbox/john-field-types-p2.md

---

## Task: Fix D2 — Cycle detection uses permanent marking
**Requested by:** Cristian

### What was done
- Fixed `resolveInclude` in `v2/internal/planner/planner.go`: replaced permanent marking with DFS backtracking via `defer delete(pc.seen, inclPath)` after `pc.seen[inclPath] = true`. Diamond deps (A→B→D, A→C→D) now resolve correctly; true cycles still caught.
- Added `TestPlanner_DiamondDependency` to `v2/internal/planner/planner_test.go`: A includes B and C, both include D — no false cycle error, D step inlined twice.
- All 14 planner tests pass.

---

## 2026-04-19T22:57:27Z — Phase 2 Defect D2: Cycle Detection Fix (Completed)

**Status:** ✅ COMPLETE

### Phase Context

Phase 2 architectural review (Ken) identified two critical defects blocking approval:
1. **D1 (Barbara):** Missing compile-time interface guard in planner.go
2. **D2 (John):** Cycle detection incorrectly rejects diamond dependencies

Both defects have been resolved. Phase 2 ready for re-review.

### Orchestration Log

- `.squad/orchestration-log/2026-04-19T22:57:27Z-barbara-phase2-d1.md` — D1 fix (interface guard)
- `.squad/orchestration-log/2026-04-19T22:57:27Z-john-phase2-d2.md` — D2 fix (cycle detection backtracking)
- `.squad/log/2026-04-19T22:57:27Z-phase2-fixes.md` — Session summary

### Status

- Build: Clean
- Tests: 14/14 pass
- Vet: No warnings
- Next gate: Ken's re-review
## 2026-04-19: D2 Verification Complete

**Status:** ✅ APPROVED

Backtracking fix verified by Ken re-review. Lines 270-271 contain correct DFS backtracking:
```go
pc.seen[inclPath] = true
defer delete(pc.seen, inclPath)
```

New diamond-dependency test passes. All 14 tests pass. True cycle detection unchanged. Ready for Phase 3.

---

## 2026-04-21 — gert-domain-home v0 DSL Specification

**Requested by:** Cristian  
**Output:** `.squad/tmp/john-home-domain-dsl.md` (33KB, 700+ lines)

Designed the complete authoring model (Section 3) for the **gert-domain-home** domain kit — a YAML-based DSL for household maintenance orchestration.

### Design Context

GERT's domain kit architecture allows specialized DSLs to compile to core GERT execution primitives. The home domain kit targets non-technical homeowners managing property maintenance, routines, and incident response.

### DSL Sections Authored

**3.1 Property Definition** — Root container declaring property name and zones (spatial partitions)
- Zones are structured objects (id/label/description/metadata), not inline strings
- Forward-compatible metadata extension point for v1 (area, indoor/outdoor classification)

**3.2 Asset Definition** — Physical things tracked for maintenance (appliances, vehicles, fixtures)
- Each asset belongs to exactly one zone
- Optional installation/warranty dates for future notification triggers
- Metadata extension point for model/serial/purchase price (v1)

**3.3 Recurring Routines** — Scheduled maintenance tasks
- Two cadence models: fixed interval (`every: 7d`) or seasonal (`summer: 7d, winter: 21d`)
- Seasonal cadence resolves at runtime using hemisphere + astronomical season boundaries
- Evidence requirement: none/note/photo/checklist
- Optional executor hint (v1: enforcement) and notification rules (remind/escalate)
- Scope: zone, asset, or property-wide

**3.4 Incident Templates** — Reactive repair run structures
- Step types: human_task, decision, parallel
- Explicit dependency ordering via `depends_on` (implicit = declaration order)
- Triggering: from YAML run file, mobile app, or scheduled run (v1)
- Steps share parent variable scope (no isolated I/O)

**3.5 Consumables** — Replacement-tracked items (filters, batteries, chemicals)
- v0 schema defined but execution deferred to v1
- Tracks replacement cadence and last-replaced date
- Optional stock tracking (current quantity + reorder threshold)
- Can trigger a routine when replacement is due

**3.6 Delegation / Away Mode** — Temporary assignment to delegate
- Time-bounded delegation window (from/to dates)
- Assigns routines by explicit ID or by zone (all routines in zone)
- Permissions: can_report_incidents, can_modify_routines, can_view_history
- Notification routing: remind delegate, notify owner on completion/overdue
- Delegate contact (E.164 phone + optional email)

**3.7 Full Property File Example** — "Casa Santiago" capstone example
- 4 zones, 3 assets, 4 routines (1 seasonal), 2 incident templates, 1 consumable, 1 delegation
- Realistic 7-day trip delegation scenario
- 180+ lines of production-quality YAML

### Key Design Decisions

1. **Readability first**: Non-technical homeowners should understand the structure
   - Zones are structured objects, not opaque strings (future-proof)
   - Evidence is explicit enum (none/note/photo/checklist), not boolean
   - Duration format is compact (`7d`, `3M`) not verbose ISO 8601

2. **Deterministic compilation**: Each DSL construct maps to GERT primitives
   - Routines → scheduled runs with single `type: human_task` step
   - Incident templates → run templates with step tree
   - Delegation → access control rules + notification routing overrides

3. **Forward compatibility**: v0 fields establish extension points
   - Metadata blocks on zones/assets for future structured data
   - Evidence.prompt allows custom prompts (v1: templates)
   - Executor.role is a hint in v0, becomes enforced RBAC in v1
   - Consumables defined but not enforced until v1

4. **Convention alignment**: Consistent with GERT v2 schema patterns
   - kebab-case IDs (pool_pump, lawn_mow)
   - ISO 8601 dates (YYYY-MM-DD)
   - Duration notation matches GERT core (`7d`, `2w`, `3M`, `1y`)
   - Step types map to GERT step types (human_task → manual, decision → branch)

### Compilation Semantics (Normative)

**Routines → Scheduled Runs:**
- Schedule rule (cron-like for fixed, calendar-based for seasonal)
- Run template with single step (type: manual with evidence config)
- Notification rules as engine-level triggers

**Incident Templates → Run Templates:**
- Step tree with dependencies (depends_on → GERT execution graph)
- Step type mapping: human_task → manual, decision → branch, parallel → parallel
- Evidence requirements attached to step config

**Delegation → Access Control + Notifications:**
- Access control rule grants delegate read/write on assigned runs
- Notification routing override redirects to delegate contact
- Attribution metadata tags submissions with delegate ID

### v1 Extension Points Reserved

Deferred features with schema extension points:
1. Asset metadata standardization (model/serial/warranty_url)
2. Routine executor enforcement (role becomes required with RBAC)
3. Consumable inventory tracking (stock becomes operational)
4. Seasonal cadence customization (override hemisphere or custom boundaries)
5. Incident template versioning (iterate templates)
6. Multi-delegate assignment (disjoint routine sets)
7. Per-routine delegation permission overrides

### Schema Versioning

All `.home.yaml` files will include `apiVersion: home/v0`. When v1 ships, compiler supports both in compatibility mode with `gert migrate` to rewrite.

### Learnings

- **Domain kits are DSL compilers**: The home domain kit is NOT a GERT runtime extension — it's a compiler from home-domain YAML to core GERT primitives. This clarifies the architecture: domain kits sit *above* GERT, not inside it.

- **Seasonal scheduling is runtime-evaluated**: Seasonal cadence (`summer: 7d`) requires runtime to know current date, property hemisphere, and astronomical season boundaries. This is a calendar-aware schedule rule, not a static cron expression.

- **Evidence types should be explicit enums**: Using `none/note/photo/checklist` is clearer than boolean flags (`photo: true, note: false`). It's self-documenting and allows future extension (e.g., `video` or `signature`).

- **Delegation is time-bounded access control + notification routing**: It's NOT role-based access control (delegate doesn't have a persistent role). It's a temporary override with automatic expiration and explicit permission grants.

- **Incident templates define structure, not instances**: The template is the *class*, the triggered incident is the *instance*. The runtime instantiates the template steps into a run when the user triggers an incident.

- **v0 consumables are tracked but not enforced**: Defining the schema in v0 establishes the data model, but reminders are informational only. v1 enforcement requires blocking logic (prevent routine completion without consumable replacement).

**Implications for v2 schema:**
- Domain kits may need a `domain:` top-level field in GERT runbooks to declare which kit compiled them (for reverse-engineering and debugging)
- Calendar-aware scheduling (seasonal cadence) might be useful in core GERT, not just domain kits
- Evidence types (photo/note/checklist) could be promoted to core GERT manual/collector step types

---

### gert-domain-home v0 JSON Schema (2025-04-21)

Produced the complete JSON Schema (Draft 2020-12) for gert-domain-home `.home.yaml` files.

**Deliverables:**
1. `specs/gert-domain-home/schema.json` — Complete JSON Schema (26,650 chars, 10 reusable definitions)
2. `specs/gert-domain-home/schema-notes.md` — Implementation notes and validation caveats (8,459 chars)

**Schema Coverage:**
- **Property definition**: name, address, hemisphere, timezone, owner, zones (min 1)
- **Zones**: id (kebab-case pattern), label, description, metadata (extensible)
- **Assets**: id, zone reference, label, installed date, warranty expiration, metadata
- **Routines**: id, label, zone/asset scope, cadence (every XOR seasonal), evidence, executor, notifications
- **Incident templates**: id, label, applicable zones/assets, steps (min 1)
- **Incident steps**: id, type (human_task/decision/parallel), label, evidence, depends_on, choices, parallel_steps
- **Consumables**: id, label, zone/asset reference, replace_every, last_replaced, stock tracking, triggers_routine
- **Delegation**: delegate info, active time window, task assignments (routine/zone), permissions, notifications

**Reusable Definitions ($defs):**
1. `zone` — Zone object with id pattern validation
2. `asset` — Asset object with zone reference
3. `routine` — Routine object with cadence oneOf constraint
4. `incident_template` — Template object with steps array
5. `incident_step` — Step object with type-specific fields
6. `consumable` — Consumable object with duration validation
7. `delegation` — Delegation object with time window and assigns array
8. `duration` — Duration string pattern (e.g., "7d", "2w", "3M", "1y")
9. `seasonal_cadence` — Object with spring/summer/autumn/winter fields (all required)
10. `evidence_config` — Evidence type enum + optional prompt

**Validation Constraints:**
- Pattern validation: kebab-case IDs (`^[a-z][a-z0-9_]*$`), duration strings (`^\d+[dwMy]$`)
- Enum constraints: hemisphere, evidence type, step type
- OneOf constraints: cadence (every XOR seasonal), delegation assigns (routine XOR zone)
- Required fields: property.name, property.zones, routine.cadence, etc.
- MinItems constraints: zones (min 1), incident steps (min 1), delegation assigns (min 1)
- AdditionalProperties: false on all objects (strict validation)

**Known Limitations (documented in schema-notes.md):**
1. **Referential integrity**: Cannot enforce zone/asset/routine ID references (runtime validation required)
2. **Temporal constraints**: Cannot enforce date ordering (e.g., delegation.to > delegation.from)
3. **Uniqueness**: Cannot enforce unique IDs across arrays (runtime validation required)
4. **Circular dependencies**: Cannot detect cycles in incident_step.depends_on (topological sort required)
5. **Mutual exclusivity**: routine.zone + routine.asset both optional (runtime should warn if both set)

**Extension Points for v1:**
- Asset metadata standardization (model, serial, warranty_url)
- Routine executor enforcement (RBAC)
- Consumable inventory tracking (operational reordering)
- Seasonal boundary customization (override hemisphere defaults)
- Incident template versioning
- Multi-delegate assignment

**Schema Quality:**
- Uses `$schema: "https://json-schema.org/draft/2020-12/schema"`
- Uses `$id: "https://gert.run/schemas/domains/home/v0.json"`
- All properties have `title` and `description`
- Enum values documented with descriptions where helpful
- Examples provided for string patterns and common values
- Strict validation with `additionalProperties: false` throughout

**Validation Approach:**
The schema enforces syntactic correctness. Runtime compilation must perform:
1. ID registry building (collect all zone/asset/routine/template IDs)
2. Reference validation (all ID references point to declared entities)
3. Uniqueness validation (no duplicate IDs)
4. Temporal validation (date ordering constraints)
5. Dependency graph validation (no circular dependencies in incident steps)

**Key Design Decisions:**
- Duration compact notation (`7d` not `"7 days"`) for brevity
- Evidence type as enum (not boolean flags) for extensibility
- Seasonal cadence requires all four seasons (explicit > implicit)
- Delegation assigns uses oneOf (routine XOR zone) for clarity
- Metadata objects are free-form in v0 (standardized in v1)

This schema can be used with standard JSON Schema validators (ajv-cli, Go jsonschema library) to validate `.home.yaml` files before compilation.

---

## 2026-04-25 — Phase 4 Option 5: Apartment Minimal Example Property

**Requested by:** Cristian  
**Completed:** apartment-minimal.home.yaml created, validated, committed (9ac0a0f)

Created a second example `.home.yaml` property file to prove the authoring model handles minimal configurations gracefully. Contrasts with the complex casa-santiago.home.yaml house.

**Property design:** Apartment 4B — small city apartment
- 3 zones: kitchen, living_room, bathroom (no pool, lawn, or garden)
- 1 asset: AC unit filter cartridge
- 2 routines: simple quarterly filter replacement + monthly bathroom deep clean
- 1 incident template: plumbing issue with decision branching (DIY vs. professional)
- 1 consumable: AC filter cartridge with stock tracking
- NO delegation block (proves optional)

**Validation:**
- YAML passes JSON Schema validation against `specs/gert-domain-home/schema.json`
- `go run ./cmd/home-validate` compiles successfully
- Generates 2 routine references (replace_ac_filter, deep_clean_bathroom)
- Generates 1 incident mitigation template (plumbing_issue)
- All YAML field names match model.go YAML tags (snake_case)
- Duration notation correct (90d, 30d)

**Schema gap observations (no blockers):**
1. Schema allows both routine.zone and routine.asset as optional simultaneously — should enforce XOR at compile time (documented as known limitation)
2. Decision step choice.next_step currently optional — incident template requires explicit step routing for clarity
3. No explicit order enforcement in incident steps — dependency graph order unclear if multiple steps depend on same predecessor

**Authoring model assessment:**
✅ YAML is intuitive and reads naturally  
✅ Field names self-document (no lookup needed)  
✅ Duration notation compact and unambiguous  
✅ Evidence prompts are clear and specific  
✅ Decision branching with depends_on provides good clarity for multi-path incidents

The minimal example validates the core authoring experience without complexity noise. Both casa-santiago.home.yaml (complex property) and apartment-minimal.home.yaml (minimal property) now demonstrate graceful handling of the full spectrum of configurations.

---

## Phase 4: Apartment Minimal Example Property (2026-04-24)

**Status:** ✅ Complete

**Mission:** Create a minimal .home.yaml example property to validate authoring model at opposite end of complexity spectrum from casa-santiago.

### Artifact Created

**File:** `domains/home/examples/apartment-minimal.home.yaml`

**Property:** Apartment 4B (urban, limited space)
- 3 zones: kitchen, living_room, bathroom
- 1 asset: AC filter
- 2 routines: filter replacement, bathroom cleaning
- 1 incident template: plumbing decision tree
- 1 consumable: AC filter cartridge
- NO delegation (proves it's optional)

**Validation:** ✅ Passes schema, compiles cleanly

### Authoring Model Assessment

#### What Works Well ✅

1. Field naming — all snake_case, self-documenting
2. YAML structure — natural hierarchy (property → zones → assets → routines → incidents)
3. Duration notation — compact `90d`, `30d` is intuitive
4. Evidence capture — prompts specific and actionable
5. Decision trees — `depends_on` + `next_step` make workflows explicit
6. Optional fields — delegation, executor, notifications appropriately optional
7. Minimal configurations — apartment example proves model handles simplicity well

#### Schema Gaps Identified (v1 Enhancement Candidates) 🔧

1. **Routine Scope Ambiguity (Minor)**
   - Issue: Routine can have optional `zone` and optional `asset` simultaneously
   - Gap: Semantic requirement is exactly ONE scope (zone XOR asset XOR property-wide)
   - JSON Schema cannot enforce XOR across independent fields
   - Recommendation: Add runtime validation; consider schema enhancement

2. **Decision Step Routing Clarity (Medium)**
   - Issue: `incident_step.choice.next_step` is optional, but workflow requires explicit routing
   - Gap: Schema doesn't enforce that decision steps have routable choices
   - Question: If choice lacks `next_step`, does execution continue or stall?
   - Recommendation: Add schema constraint: `type: decision` → all choices MUST have `next_step`

3. **Incident Step Dependency Order (Low)**
   - Issue: Schema allows arbitrary `depends_on` lists with no ordering enforcement
   - Gap: Execution is DAG-based, but YAML order affects readability
   - Authors may be confused if steps appear after their dependents
   - Recommendation: Best practice docs: "Write incident steps in topological order"

### Validation & Compilation

```bash
cd domains/home
go run ./cmd/home-validate examples/apartment-minimal.home.yaml ✅
```

Output: Valid GERT runbook YAML, cleanly compiled

### Decisions Generated

1. **Apartment example validates minimal authoring model**
2. **Three schema gaps identified for v1 enhancement prioritization**
3. **Optional fields (delegation) work correctly**

### Learnings

1. Minimal authoring is achievable — not all properties need full complexity
2. Optional fields work as designed — flexibility is correct
3. Schema is sound with 3 minor gaps — all addressable in v1 enhancement
4. Topological ordering matters for readability (execution is DAG-based)

### Model Scaling

The authoring model now demonstrates range from simple to complex:
- **Minimal:** apartment-minimal.home.yaml
- **Complex:** casa-santiago.home.yaml

Both validate successfully, confirming model flexibility.

### Next Steps

1. Promote 3 schema gaps to decisions.md "Schema Enhancements" section
2. Team sync to prioritize gap fixes before v1 release
3. Use apartment example in authoring documentation as "minimal property" guide

