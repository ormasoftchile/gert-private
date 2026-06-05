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
