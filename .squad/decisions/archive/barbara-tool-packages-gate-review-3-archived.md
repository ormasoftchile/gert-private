# Architecture Gate Review #3 (final, narrow) — GERT Tool Packages MVP (Ken's S1/S2 sweep)

**From:** Barbara (Lead / Architect)
**Date:** 2026-08-09T15:45-07:00
**Requested by:** Cristián Ormazábal Ortega
**Scope:** Diff-only re-check of S1/S2 per
`.squad/decisions/inbox/barbara-tool-packages-gate-review-2.md`, plus confirmation that Ken's
optional §2 cleanups did not alter accepted semantics. AR-TP-1..10 and R1–R15 remain closed and
were not re-litigated.
**Revision under review:** `.squad/decisions/inbox/ken-tool-packages-revision-s1-s2.md` (Ken,
authored under Edith + Don lockout)
**Verdict:** **APPROVED**

---

## 1. S1 — §03 `\subsection{toolRefs}` — RESOLVED

Verified in `design/gert/sections/03-schema-vnext.tex` lines 170–195, not from Ken's matrix:

- The `alias: kubectl-wrapper  # optional local alias` line and its explanatory paragraph are
  **gone**. Replaced by an explicit statement that `alias` is **not** part of the schema and its
  presence is `PKG-021`, a hard parse-gate error detected before generic schema validation.
- The `tools/foo.tool.yaml` default-discovery sentence is **gone**, and replaced by its negation:
  "There is no default, name-derived discovery path such as `tools/<name>.tool.yaml`: a bare
  `name` is resolved by lookup against the frozen catalog produced by Phase C." That is the
  ratified AR-TP-2 model, stated by cross-reference rather than restated locally — exactly the
  instruction given.
- `toolRefs` is now correctly framed as bind/narrow-only, with `requires:` named as the canonical
  catalog-supplying key.
- Cross-references resolve: `sec:tool-bindings` (§06:119), `sec:tool-name-resolution` (§06:206),
  `sec:tool-secure-paths` (§06:1089) all exist. I re-ran a full `\label`/`\ref` closure across
  `sections/*.tex` + `main.tex`: **293 labels, 121 refs, 0 unresolved.** The §03 text matches the
  §06 `ToolRef` field-disposition table (`package`/`version`/`path`/deprecated `source`/deprecated
  `actions`, `alias` REMOVED → `PKG-021`) with no drift.

Repo-wide scan for surviving alias/default-discovery **guidance**: the only remaining hit is the
negation above. Every other `alias` occurrence is either a prohibition (§03d `PKG-021` rows, §06
field table, §10 deferral rationale, `runbook.v1.schema.json` `PKG-020` description), the unrelated
and still-valid `imports:` alias→path map for child runbooks, or unrelated GXL/GCP function-alias
prose. r23's description mentions `alias` only to state the fixture does **not** use it. No
normative example instructs a reader to author `alias:` or to rely on name-derived discovery.

## 2. S2 — mapping-shaped `actions:` — RESOLVED

All three listings converted to the canonical list form, `sensitive_inputs` preserved:

- `03-schema-vnext.tex:3182` → `- name: get-pods`
- `03-schema-vnext.tex:3303` → `- name: check`, `- name: check-timeout`
- `08-security-and-trust.tex:320` → `- name: deploy`, with `args.api_key` /
  `sensitive_inputs` intact underneath — §08 remains the normative home of `sensitive_inputs`
  placement, now in a shape §06 permits.

Exhaustive re-scan of every `actions:` block in the corpus (`.tex`, `.yaml`, `.md`): **zero
mapping-shaped occurrences remain.** §06:271 and §06:873 already used the list form; all 12
`testdata/tools/*.tool.yaml` and the reference `kubectl.tool.yaml` use `- name:`.

## 3. Optional §2 cleanups — semantics unchanged

Each checked for semantic drift against the accepted state; all four are text-only.

| Cleanup | Check | Result |
|---|---|---|
| Stale ruling citations `inbox/` → `archive/` | Repo-wide grep for `inbox/barbara-tool-packages-architecture-ruling` | **0 hits.** Citations to `inbox/barbara-tool-packages-gate-review*.md` correctly retained — those files are still in `inbox/`. |
| `com.acme.` → `acme.incident-tools` | All 30 occurrences across §06 (×14), §08, §13 (×2), `vector.schema.json`, `gert-package.yaml`, `drain-node.yaml`, r23 now agree. `acme.incident-tools` satisfies the `tool-package.v1` reverse-DNS `name` pattern; the hyphenated directory name is not an identifier and correctly left alone. | No semantic change; drift closed rather than papered over. |
| `LocalStructured` → `LocalOutputs` §3.4a | §06:953 now cites the production that R3 actually added; matches `gcp.ebnf` and r23. | Corrects a mis-citation; grammar untouched. |
| r23 dry-run narrative | Rewritten to state plainly that `assert_drained` has no `when:` guard, *is* reached in dry-run, and would evaluate an unproduced `outputs.drained` — attributing this to the pre-existing spec-wide mode-skip gap. | Accurate; the fixture no longer asserts a guard that does not exist. Real/replay remain the modes demonstrated end-to-end, which is what R15 required. |
| §06 no-cross-step-`outputs` sentence (note #4) | §06:980–987 states `outputs.<name>` is capturable only in the owning step's `capture:` block, that `StepCapture` gains no `step.{id}.outputs.{name}` form, and that this is deliberate MVP scope. | Closes the unstated gap without inventing a new form. I confirmed no `step.*.outputs.*` form exists anywhere in `.tex`/`.ebnf`/schemas. |

No schema, error code, grammar production, or runtime semantic was altered by the cleanups. The
`ToolRef`/`requires` contracts, tier model, `PKG-*` catalog, and substitution/replay model are
byte-compatible with what gate 2 accepted.

## 4. Independent validation (re-run, not taken on trust)

- `python design/gert/scripts/verify_corpus.py` → **OK 280/280 vectors validate.**
- All 4 `schemas/*.json` + `conformance/vector.schema.json` load as valid JSON.
- `Draft202012Validator`: `gert-package.yaml` vs `tool-package.v1` → **0 errors**;
  `drain-node.yaml` vs `runbook.v1` → **0 errors**; `r23/schema.yaml` vs `runbook.v1` → **1
  error**, `apiVersion: runbook/v2`, the known corpus-wide pre-existing issue (gate-1 note #5,
  gate-2 item 6) — not this sweep's defect.
- `\label`/`\ref` closure across all sections → **0 unresolved, 0 duplicates.** The two
  `sec:gis:portable-json` / `sec:gcp` warnings Ken reported are first-pass artifacts; both labels
  exist in `03b`/`03c`, files this sweep did not touch.
- Full diff review of `03-schema-vnext.tex` and `08-security-and-trust.tex`: the only changes are
  the S1 rewrite and the three S2 conversions (plus the §08 package-digest/verify material already
  accepted at gate 2). No collateral edits.

## 5. Disposition

**APPROVED.** The specification is **ready for Cristián's review.**

Nothing outstanding is blocking. What remains is deferred or pre-existing:

**Deferred, Tess-owned (unblocked, not blocking this gate):**
- `design/gert/conformance/tv-pkg-resolve.yaml`, ≥46 vectors, against the now-stable surfaces:
  the `PKG-*` category enum, `TV-PKG-*` id pattern, `PackageExpected` including `warnings:`, the
  `outputs.<name>` capture root, the `execute:` action key, `PKG-029`/`PKG-030`, Table
  `tab:tool-path-bases`, and the replay-as-`invoke` substitution model. The target list in gate-2
  §3 stands unchanged and is now fully concrete.

**Non-blocking pre-existing issues (own tickets, not introduced here):**
1. Corpus-wide `apiVersion: runbook/v2` vs `runbook.v1.schema.json` — separate ticket, agreed at
   gate 1.
2. `mode: replay` terminology overload between §07 (trace-event replay) and §13 (execution-mode
   replay) — flagged by Don, own ticket.
3. Spec-wide under-specification of downstream-step behaviour after an upstream mode-skip —
   surfaced by r23's dry-run path; pre-existing, own ticket.
4. r23 `assessment.md`/`README.md` companion — still optional.

**Editorial nit, non-blocking, not worth a fourth cycle:** `03-schema-vnext.tex:184` carries
`# direct path, resolved per \S\ref{sec:tool-secure-paths}` inside a `minted` block. `minted` is
verbatim and this document sets no `escapeinside`, so the macro renders literally in the PDF
comment. It is typographic only — the guidance is correct, the surrounding prose carries the same
cross-reference properly, and it is the sole such occurrence in the corpus. Fold it into whatever
touches §03 next; it does not justify designating a fourth author or another gate cycle.

**No reviser designated. No further gate pass required.**

— Barbara (Lead / Architect)
