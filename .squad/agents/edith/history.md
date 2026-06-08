# Edith — History

## Project Context

- **Project:** GERT — Governed Executable Runbook Technology
- **Owner / User:** ormasoftchile (Germán)
- **Tech Stack:** Go (runtime), Azure (web platform), TypeScript (extensions/web), C# (parallel runtime, planned)
- **Joined:** 2026-06-05 for the GXL/GIS/GCP Phase 1 spec rewrite (Stream B)
- **Specification location:** `design/gert/sections/*.tex`, `design/gert/expression-language-proposal.md`, `design/gert/grammar/*.ebnf`

## Day 1 Context

- The team just replaced Go-style expression evaluation (`expr-lang/expr` + `text/template`) with three GERT-native languages: GXL (boolean), GIS (interpolation), GCP (capture paths).
- Barbara delivered the formal EBNF grammar files in Stream A; I encode them as spec prose in Stream B.
- Phase 1 plan: `.squad/decisions/inbox/barbara-gxl-phase1-plan.md` (now merged to `decisions.md`). Stream B effort = L, ~7 days.
- Ratified semantics binding for spec prose: OQ1 (short-circuit and/or), OQ2 (capture.default scalars only), OI-GIS-01 (`\${` canonical), OI-GCP-02 (bare root capture allowed).

## Learnings

### 2026-06-05 — Stream B Day 1: GXL section (`03a-expression-language.tex`)

**Spec conventions observed:**
- Every file in `design/gert/sections/` is a `\chapter{}` (not `\section{}`).
  Top-level sections within a chapter use `\section{}`.
- Labels use `ch:` prefix for chapter-level cross-references
  (e.g., `\label{ch:security}`, `\label{ch:schema}`).
  The GXL section uses `\label{sec:gxl}` per task specification — note
  the deviation from the `ch:` convention.
- Tables use `\begin{tabular}` with `\hline`; `longtable` is available
  for wide or multi-page tables.
- Code blocks use `\begin{minted}{yaml|go|json|text}`. For grammar
  productions, use `text` lexer.
- Inline code uses `\texttt{}` (not `\texttt` with `\texttt{}`).
- `\yinline{...}` is a convenience macro for inline YAML (`\mintinline{yaml}{...}`).
- `\emph{}` for emphasis; `\textbf{}` for bold.
- `\S\ref{...}` for section cross-references with the § symbol.
- `xcolor`, `minted`, `longtable`, `booktabs`, `tcolorbox` are all loaded.
- No `\TODO` macro in the preamble — defined locally via `\providecommand`.
- RFC 2119 keywords (MUST, SHOULD, MAY) are written in ALL CAPS inline;
  no special macro is used.

**Terminology committed (must remain consistent across all Stream B sections):**
- **GXL evaluator** (not "interpreter") for the runtime component
- **GDP** (GERT Dotted Path) for dotted path access syntax
- **PJVM** (GERT Portable JSON Value Model) for the type system
- **context map** for the runtime variable scope (from grammar)
- **boolean position** for where a bool result is required (from grammar §5.3)
- **parse error** vs. **evaluation error** for timing distinction

**LaTeX macros used (none invented beyond `\TODO`):**
- `\providecommand{\TODO}[1]{\textbf{\textcolor{red}{[TODO:~#1]}}}` — local definition

**Discrepancies flagged (grammar wins in all cases; logged in Day 1 memo):**
1. DISC-1: Proposal §1.3 says no scientific notation; grammar §2.5 includes `EXP`. Grammar wins.
2. DISC-2: Proposal §1.6 lists `len` as a namespace; grammar has it as a top-level builtin. Grammar wins.
3. DISC-3: Proposal §1.6 stdlib table is missing `str.length`, `list.indexOf`, `list.length`. Grammar wins.
4. DISC-4: Proposal §1.8 forbids "member access on identifiers"; grammar §3 defines GDP (dotted path) as a valid expression form. Grammar wins; need Barbara's confirmation.

**Sections identified for Stream B rewrite (not modified in this task):**
- `sections/03-schema-vnext.tex` lines ~412–560 (expr-lang/expr syntax)
- `sections/03-schema-vnext.tex` lines ~2518–2560 (branch condition labeled "Go template")
- `sections/03-schema-vnext.tex` lines ~1250–1354 (collector fields.when conflict)
- `sections/02-architecture.tex` lines ~467–477 (Go template CLI interpolation)
- `sections/02-architecture.tex` lines ~586–587, 1385–1412 (Go template display.content)

**Open items for next Stream B tasks:**
- PJVM needs a single canonical definition (probably in a GIS or shared section)
- Conformance corpus TV-GXL-EVAL is owned by Tess (not Stream B)
- GCP spec section needed before the `capture.default:` cross-reference resolves
- `main.tex` needs `\input{sections/03a-expression-language}` added
- Migration appendix (deprecation of expr-lang/expr) is a separate Stream B task

---

### 2026-06-05 — Stream B Day 2: GIS and GCP sections (`03b`, `03c`)

**Sections delivered:**
- `design/gert/sections/03b-interpolation-syntax.tex` — Normative GIS spec.
  Labels: `sec:gis`, `sec:gis:normref`, `sec:gis:lexical`,
  `subsec:gis:template-string`, `subsec:gis:raw`, `subsec:gis:escapes`,
  `subsec:gis:interpolation`, `sec:gis:expression`, `sec:gis:portable-json`,
  `subsec:gis:pjvm-types`, `subsec:gis:pjvm-exclusions`,
  `subsec:gis:unresolved`, `sec:gis:coercion`, `subsec:gis:coercion-number`,
  `sec:gis:errors`, `sec:gis:migration`.
- `design/gert/sections/03c-capture-paths.tex` — Normative GCP spec.
  Labels: `sec:gcp`, `sec:gcp:normref`, `sec:gcp:sources`,
  `subsec:gcp:sources:local`, `subsec:gcp:sources:step`,
  `subsec:gcp:sources:http`, `subsec:gcp:sources:event`,
  `subsec:gcp:sources:table`, `sec:gcp:traversal`, `sec:gcp:subtree`,
  `subsec:gcp:subtree:inference`, `subsec:gcp:subtree:iteration`,
  `sec:gcp:default`, `subsec:gcp:default:scalar`,
  `subsec:gcp:default:subtree`, `sec:gcp:errors`, `sec:gcp:migration`.

**New terminology committed (must remain consistent):**
- **GIS evaluator** — runtime component rendering a GIS template string
- **template string** — GIS field value (outer string, not the expression inside)
- **interpolation block** — a `${...}` segment in a template string
- **PJVM** — canonical home is now `§sec:gis:portable-json` (03b); 03a and 03c reference it
- **boolean** — the PJVM type name (grammar term); `bool` is GXL shorthand only
- **bare-root capture** — a `json`/`yaml` path with no GDP suffix (OI-GCP-02)
- **scalar capture** — PJVM captured value of type null/boolean/number/string
- **subtree capture** — PJVM captured value of type array or object
- **source prefix** — the leading segment of a GCP expression
- **capture-then-GXL pattern** — approved alternative to pipe expressions

**Model-defining decisions committed:**
- `§sec:gis:portable-json` is the single authoritative normative home of PJVM.
  All other spec sections reference it; none duplicate it.
- `boolean` is the PJVM type name; `bool` is acceptable GXL shorthand only in
  error-message descriptions and the GXL type table (`03a`). Not interchangeable
  in normative PJVM prose.
- `http.body` without GDP suffix returns **string** (scalar), not subtree.
  Bare-root subtree capability (OI-GCP-02) applies only to `json`/`yaml` sources.
- Pipe expressions (e.g., `json | length`) are explicitly forbidden in GCP.
  The approved pattern is capture-then-GXL.

**Discrepancies flagged (grammar wins in all cases; logged in Day 2 memo):**
1. DISC-B2-1: `boolean` (gis.ebnf) vs `bool` (03a prose). Grammar wins; `boolean` is PJVM canonical.
2. DISC-B2-2: decisions.md OI-GCP-02 example uses `http.body` as bare-root subtree example, but grammar shows `http.body` (no GDP) is a scalar string return. Grammar wins; spec encodes grammar behavior.
3. DISC-B2-3: OQ5/Q5 label from task brief not traceable in accessible decisions. Cited `gcp.ebnf §5` as source; parenthetical "(Q5, resolved)" per task brief.

**Deferred items:**
- Proposal §2.6 amendment (OI-GIS-01 action item) — not done in this task
- `03a §sec:gxl:semantics:types` forward-reference to `§sec:gis:portable-json`
- `main.tex` wiring for 03b and 03c (same as 03a, deferred to integration task)

---

## Integrated to Main (2026-06-05T00:27:12-04:00)
## 2026-06-07T19:14:39-07:00 — Runbook v1 JSON Schema (`design/gert/schemas/runbook.v1.schema.json`)

### Task

Authored the canonical `runbook/v1` JSON Schema per Barbara's ruling (`decisions/inbox/barbara-runbook-v1-schema-canonical-source.md`). This is the front-door artifact for gert-vscode, gert-tui, and future C#/TS runtimes.

### Field Inventory Source

The Go runtime structs were used as a **reference implementation checklist** (not as the canonical source):
- `pkg/schema/runbook.go` — top-level `Runbook` struct, `Input`, `Output`, `ToolRef`, `ExtensionRef`, `StepDefaults`, `GovernanceConfig`, `GovernanceRule`, `RedactRule`, `FlowNode`
- `pkg/schema/step.go` — `Step` (common fields), `RetryConfig`, `Contract`
- `pkg/schema/steps.go` — all step-type payloads: `CLISpec`, `ToolCallSpec`, `IncludeSpec`, `ChoiceSpec`, `DecisionSpec`, `CollectorSpec`, `BranchSpec`, `ApproveSpec`, `AssertSpec`, `CompensateSpec`, `WaitForEventSpec`, `EndSpec`, `NoopSpec`, `DisplaySpec`, and nested types
- `pkg/schema/tool.go` — `ToolRef`, `ToolInvocation`
- `pkg/schema/regions/manifest.go` — `RegionsManifest`, `Region`

Canonical authority remains the spec sections (`design/gert/sections/*.tex`) and grammar files (`design/gert/grammar/*.ebnf`).

### Schema Design Decisions

- **Draft:** JSON Schema draft 2020-12 (`$schema: https://json-schema.org/draft/2020-12/schema`)
- **`$id`:** `https://gert.dev/schemas/runbook/v1` (per Barbara's decision document)
- **Step discrimination:** Used `allOf[if/then]` with `unevaluatedProperties: false` at Step level. Each step type has its own `then` branch defining its payload properties. Type-specific fields are rejected for non-matching types. This is the most structurally correct pattern for draft 2020-12.
- **Opaque GIS/GXL/GCP strings:** All string fields that hold expression templates are typed as `"type": "string"` with a description note. The schema does not validate expression syntax — that is runtime-only.
- **`additionalProperties: false`** on every object definition. One exception: `vars` and `ToolInvocation.args` use `additionalProperties: {}` (any value) since Go uses `map[string]any`.
- **Step ID pattern:** `^[a-z][a-z0-9_-]*$` applied to step, iterate, and parallel `id` fields per Barbara's boundary statement. Runbook root `id` is unconstrained (examples use dots: `incident-triage.network`). SQ-004 filed.
- **`name` on Step:** Present in nav-test example, absent from Go struct. Included as optional legacy alias with SQ-001 filed.

### Validation Results

Validated against **23 runbook YAML files** in `gert/examples/` using AJV (draft 2020-12).

| Outcome | Count | Notes |
|---|---|---|
| ✅ PASS | 23 | All examples pass |
| ❌ FAIL | 0 | None |

One schema fix was required during validation:
- `resource-exhaustion.runbook.yaml` uses `type: list` for an input. The Go struct has no enum on `Input.Type`. Schema was updated to include `list` and `array`/`object` in the Input type enum. SQ-003 filed for Barbara to confirm the normative set.

### Ambiguities Parked

Filed `design/gert/schemas/` open questions in `.squad/decisions/inbox/edith-runbook-schema-questions.md`:
- **SQ-001:** `name` field on steps — legacy alias for `title`?
- **SQ-002:** `extension` step type — payload undefined; empty `then` may be too strict
- **SQ-003:** Input `type` enum — `list` alias, normative set not specified in spec
- **SQ-004:** Runbook root `id` pattern — examples use dots, pattern not applied at root level
- **SQ-005:** `iterate`/`parallel` as step `type` values — in Go enum but not used as step types in any example

### Files Delivered

- `design/gert/schemas/runbook.v1.schema.json` — canonical schema (draft 2020-12, hand-authored)
- `design/gert/schemas/README.md` — updated with full boundary statement, consumer rules, maintenance guide, and generation policy
- `design/gert/schemas/examples/smoke-test.runbook.yaml` — CI smoke-test fixture (validated: PASS)
- `.squad/decisions/inbox/edith-runbook-schema-questions.md` — open questions for Barbara

### Conventions Confirmed (JSON Schema for Grammar-Backed Languages)

When authoring a JSON Schema from a grammar-backed spec (EBNF → spec prose → Go structs):
1. Start with the Go struct surface as a field checklist
2. Model discriminated unions with `allOf[if/then]` + `unevaluatedProperties: false` (draft 2020-12)
3. Treat all expression-typed string fields as opaque; note GIS/GXL/GCP in description
4. Use `additionalProperties: false` on all objects except free-form maps (`map[string]any`)
5. Validate against ALL existing examples before declaring done; fix schema if too strict
6. File open questions rather than guess on ambiguous enum values or payload shapes

## 2026-06-07T19:28:42-07:00 — Barbara Rulings Followup (SQ-001–005)

**Task:** Apply all 7 Edith action items from Barbara's schema rulings dispatch
(`.squad/decisions/inbox/barbara-schema-rulings-2026-06-07.md`).

**Schema changes delivered** (`design/gert/schemas/runbook.v1.schema.json`):

| Item | Change |
|---|---|
| SQ-001 | Removed `name` field from Step properties; updated Step description to document the `title`/`name` distinction |
| SQ-002 | Replaced empty extension `then: {}` with full payload sub-schema: `extension.name` (required string), `extension.action` (required string), `extension.args` (optional additionalProperties map) |
| SQ-003 | Tightened Input type enum: removed non-normative `integer` and `list`; added `array` and `object` after authoring spec prose |
| SQ-004 | Set `$id` to `https://schemas.gert.dev/runbook/v1.json` per §Self-Description; prior value was `https://gert.dev/schemas/runbook/v1` |
| SQ-005 | Added `iterate` and `parallel` to step type enum; added corresponding `if/then` payload blocks for step-type usage of both |

**Spec prose changes delivered** (`design/gert/sections/03-schema-vnext.tex`):

| Item | Change |
|---|---|
| §Input Declarations | Added normative `array` and `object` input type paragraphs with RFC 2119 wording, serialization model (JSON array / prompt comma-split), GCP compatibility note, and NOT RECOMMENDED guidance for object prompt-sourcing |
| §Step Types table | Updated count 15 → 16; added `extension` row citing `\S\ref{subsec:step-extension}`; added normative note on `iterate`/`parallel` dual TreeNode form |
| §Step Types taxonomy figure | Updated caption from "15 types" to "16 types" |
| §Step extension subsection | New `\subsection{extension step}\label{subsec:step-extension}` with payload contract table, normative rules list, and YAML example |

**Example cleanup delivered** (in `gert` repo, branch `edith/schema-rulings-followup`):

| File | Change |
|---|---|
| `nav-test/nav-test.runbook.yaml` | Renamed step-level `name:` → `title:` on all 3 steps (SQ-001) |
| `incident-triage/resource-exhaustion.runbook.yaml` | Changed `type: list` → `type: array` on `instances` input (SQ-003; `list` ruled non-normative) |

**Validation results:**
- ✅ All 23 example runbooks in `gert/examples/` pass against updated schema
- LaTeX brace check: no unclosed environments; new subsection follows established `\subsection` → `\paragraph` → `\begin{center}...\end{center}` → `\begin{minted}` pattern
- CI not run — disabled per cost-control directive (gert#35)

**Schema patterns used:**

- **Discriminated step payloads:** `allOf[if/then]` with `if: { required: ["type"], properties: { type: { const: "…" } } }` and `then: { required: [...], properties: {...} }`. `unevaluatedProperties: false` at Step root enforces no spurious fields across all branches.
- **Dual-form iterate modeling:** FlowNode uses `oneOf: [{ required: ["step"] }, { required: ["iterate"] }, { required: ["parallel"] }]` for the standalone TreeNode form. Step.type enum includes `iterate` and `parallel` for the step-wrapper form. Both forms are independently valid; no shared ref is needed — IterateNode and the `iterate` step `then` block share the same field set but are separate schema branches.
- **Extension payload:** `required: ["name","action"]`, `additionalProperties: false` at the `extension` object level, `args` with `additionalProperties: true` for open key-value maps. This pattern (closed container, open leaf) is the right choice when extension authors control the argument space.

**Spec section locations confirmed:**
- `§Input Declarations`: `03-schema-vnext.tex` lines ~360–390 (`\paragraph{type field}`)
- `§Step Types table`: lines ~612–633
- New `§extension step`: inserted before `\section{Tool Definition Schema}` at the end of step subsections
- `§Self-Description`: lines ~50–80 (confirmed `$id` = `https://schemas.gert.dev/runbook/v1.json`)

**Cross-references added:** `\S\ref{subsec:step-extension}` in Step Types table; `\S\ref{sec:gis:portable-json}` and `\S\ref{sec:gcp}` in input type prose.

**Open questions:** None. All 5 SQ rulings were fully actionable. `list` → `array` rename was straightforward (no semantic ambiguity). `object` input NOT RECOMMENDED for prompt/env sourcing encoded inline per logical inference from PJVM model; no Barbara escalation needed.



✅ **Day 2 memo merged to `.squad/decisions.md`** under section "GXL Phase 1 Day 2 — GIS/GCP Specs, Eval+Path Vectors, Conflict Arbitration". All deliverables, ratifications, discrepancies, and follow-ups captured. PJVM canonical home established at `03b §sec:gis:portable-json` per Stream B Day 2 memo.

## 2026-06-04T20:14:36-07:00 — Stream E Removed (Scribe notification)

**Action Item:** §5 Migration Plan (design/gert/expression-language-proposal.md) has been removed per user directive. Appendix A migration examples also removed and precedence appendix renumbered.

**Spec Sections Affected:**
- No changes to 03a/03b/03c/03d content — migration references/language removed, but semantics intact
- All GXL/GIS/GCP spec sections remain authoritative

**Decision:** Merged to `.squad/decisions.md` (timestamp 2026-06-04T20:14:36-07:00)

## 2026-06-05T07:28:56.273-07:00 — GIS Optional Path Chaining

**Task:** Appended normative optional-chaining prose to `design/gert/sections/03b-interpolation-syntax.tex` without touching `design/gert/grammar/gis.ebnf` or conformance vectors.

**Section added:** `Optional Path Chaining` after the GIS error catalog. Labels: `sec:gis:optional-chaining`, `subsec:gis:optional-chaining:syntax`, `subsec:gis:optional-chaining:semantics`, `subsec:gis:optional-chaining:composition`, `subsec:gis:optional-chaining:capture-default`, `subsec:gis:optional-chaining:examples`, `subsec:gis:optional-chaining:out-of-scope`.

**Normative points encoded:** `?.` and `?.[N]`; empty-string default on optional miss; JS/TS-compatible full-tail short-circuiting; missing vs present-value lists; mixed-path hard errors; `${?.root}` parse illegality; stdlib call non-propagation; unchanged hard-error default for plain `${...}`.

**Coordination note:** Dropped `.squad/decisions/inbox/edith-gis-optional-chaining-spec.md` for Scribe/Tess with labels and cross-reference notes.

## 2026-06-07T19:14:39-07:00 — Runbook v1 JSON Schema Authoring

**Task:** Author the canonical `design/gert/schemas/runbook.v1.schema.json` per Barbara's decision that design repo is the normative home.

**Approach:**
- Used Go runtime structs (`pkg/schema/runbook.go`, `step.go`, `steps.go`, `tool.go`) as a reference checklist
- Canonical authority remains spec sections (`design/gert/sections/*.tex`) and grammar (`design/gert/grammar/*.ebnf`)
- Schema: JSON Schema draft 2020-12, `$id: https://gert.dev/schemas/runbook/v1`
- Step discrimination: `allOf[if/then]` with `unevaluatedProperties: false` for type-safe step payloads
- All expression strings (GIS/GXL/GCP) typed as opaque strings with description notes (syntax validation is runtime-only)

**Validation:**
- ✅ All 23 example runbooks in `gert/examples/` pass validation
- ✅ One schema fix during validation: added `list` and `object`/`array` to Input type enum (SQ-003 filed)

**Open Questions Filed (5 spec questions for Barbara arbitration):**
| # | Question | Blocks |
|---|----------|--------|
| SQ-001 | Is `name:` field on steps a legacy alias for `title:`, deprecated, or missing? | Nothing immediately; safe default chosen |
| SQ-002 | What fields does `extension` step type carry? Payload fixed or open? | gert-vscode hover/completion for extension steps |
| SQ-003 | Is `list` a normative type alias for `array` in input declarations? | Normative enum in spec §Input Declarations |
| SQ-004 | Should runbook `id` pattern differ from step `id` pattern? | ID validation strictness for gert-vscode |
| SQ-005 | Are `iterate` and `parallel` valid step `type` values? | Would affect schema if embedded inside step wrappers |

**Files Delivered:**
- `design/gert/schemas/runbook.v1.schema.json` — canonical schema (hand-authored, draft 2020-12)
- `design/gert/schemas/README.md` — boundary statement, consumer rules, maintenance guide
- PR #7 on GitHub: https://github.com/ormasoftchile/gert-private/pull/7
- `.squad/decisions/inbox/edith-runbook-schema-questions.md` — all 5 spec questions documented

**Next Steps:** Awaiting Barbara's arbitration on all 5 spec questions. Once decided, will update schema and spec sections for precision. gert-vscode Phase 2 (YAML schema, IDE integration) unblocked once arbitrations complete.
