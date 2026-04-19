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
  are GAP-3: `type: wait_for_event` does not yet exist in the schema. Use a `type: cli` stub
  with a `# GAP: no wait_for_event step type yet` comment as placeholder.

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
