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
