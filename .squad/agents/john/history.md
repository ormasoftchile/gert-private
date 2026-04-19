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
