# Ken — Tool Packages Revision: S1/S2 Sweep (independent, Edith and Don locked out)

**By:** Ken (Backend Developer, third independent revision owner)
**Date:** 2026-08-09
**Subject:** Resolution of Barbara's second-gate blockers S1/S2 per
`.squad/decisions/inbox/barbara-tool-packages-gate-review-2.md`. Authored independently;
Edith (original author) and Don (first revision author) were both locked out per Cristián's
directive and did not advise or contribute. AR-TP-1..10 and R1–R15 are **not** reopened; this
is a documentation/mechanical sweep only, exactly as scoped by Barbara's gate-2 review.

## S1/S2 resolution matrix

| # | Barbara's blocking item | Resolution | Files |
|---|---|---|---|
| S1 | §03 `\subsection{toolRefs}` taught `alias:` (now hard `PKG-021`) and a deleted default `tools/<name>.tool.yaml` discovery rule | Rewrote `\subsection{toolRefs}` (`\label{subsec:toolrefs}`): dropped the `alias:` example/paragraph (replaced with a one-line `PKG-021` note), dropped the default-discovery-path sentence, and cross-referenced `\S\ref{sec:tool-bindings}` (canonical `requires:`, `toolRefs` bind/narrow semantics, `package`/`version`/`path`/deprecated `source`/`actions`) and `\S\ref{sec:tool-name-resolution}` (frozen-catalog lookup, five-tier model) instead of restating tier rules locally. | `design/gert/sections/03-schema-vnext.tex` |
| S2 | Three `.tool.yaml` listings still used mapping-shaped `actions:` (`03-schema-vnext.tex` `get-pods`, `check`/`check-timeout`; `08-security-and-trust.tex` `deploy` with `sensitive_inputs`) | Converted all three to the canonical list form (`- name: <action>`), preserving all fields including `sensitive_inputs` under the converted `deploy` entry in §08. | `design/gert/sections/03-schema-vnext.tex`, `design/gert/sections/08-security-and-trust.tex` |

## Optional cleanup (mechanically safe, authorized by Barbara's non-blocking notes)

1. **Stale ruling citations.** Swept all remaining `inbox/barbara-tool-packages-architecture-ruling.md`
   references to `archive/barbara-tool-packages-architecture-ruling.md` across the schemas
   (`tool-package.v1`, `package-lock.v1`, `project-config.v1`), `conformance/vector.schema.json`,
   `schemas/README.md`, sections `02`, `03d`, `06`, `07`, `08`, `10`, `12`, `13`, and the r23 fixture
   description. Verified zero remaining `inbox/...architecture-ruling.md` hits repo-wide.
2. **`acme.`/`com.acme.` package-name drift.** Normalized the shipped example
   (`gert-package.yaml`, `drain-node.yaml`) and the r23 fixture to `acme.incident-tools`, matching the
   majority spec-prose usage (§06 ×6, §13, §08). Directory name (`acme-incident-tools/`) intentionally
   left as-is (hyphenated, filesystem-safe; not a package identifier).
3. **`LocalStructured` mis-citation.** §06 §Caller-visible capture namespace cited
   "gcp.ebnf §3.4 Local Step Capture, `LocalStructured`" for the substituted-action output surface;
   corrected to "§3.4a Substituted Tool Action Output Capture, `LocalOutputs`" — the actual production
   added for R3.
4. **r23 dry-run narrative overreach.** The fixture's mode-acceptance description claimed
   `assert_drained` "only reaches on non-dry-run modes," implying a `when:` guard that does not exist
   in the flow. Rewrote to state plainly that the step has no guard, is still reached in dry-run, and
   would evaluate an unproduced `outputs.drained` capture — noting this is the pre-existing,
   spec-wide, under-specified question of downstream behavior after an upstream mode-skip, not a
   defect in this fixture, and that real/replay (the modes R15 actually turned on) are what the
   fixture demonstrates end-to-end.
5. **Invented cross-step `step.{id}.outputs.*` form.** Confirmed no such form was actually invented
   anywhere in the corpus (grep across `.tex`/`.ebnf`/schemas: none found) — Barbara's note was that
   the *absence* was unstated, not that an invented form existed. Added one clarifying sentence to §06
   §Caller-visible capture namespace stating `outputs.<name>` is capturable only in the same step's own
   `capture:` block, that `StepCapture` gains no `step.{id}.outputs.{name}` form in this MVP, and that
   this is a deliberate scope choice, not an oversight.

No other artifact was touched. AR-TP-1..10, R1–R15, and S1/S2 as scoped are the entirety of the
diff's design surface; the cleanup items above are text-only corrections to already-decided facts
(citation paths, a naming choice already made elsewhere, a grammar production name, and a fixture
comment's accuracy) and change no schema, no error code, and no runtime semantics.

## Validations performed

- `python design/gert/scripts/verify_corpus.py` → **280/280 conformance vectors validate** (baseline
  preserved).
- JSON-load of all 5 touched/added JSON Schema files → valid JSON.
- YAML-load of `drain-node.yaml`, `gert-package.yaml`, `r23-tool-package/schema.yaml` → valid YAML.
- `jsonschema.validate` of `drain-node.yaml` against `runbook.v1.schema.json` → **0 errors**.
- `jsonschema.validate` of `gert-package.yaml` against `tool-package.v1.schema.json` → **0 errors**.
- `jsonschema.validate` of r23 `schema.yaml` against `runbook.v1.schema.json` → 1 error,
  `apiVersion: runbook/v2` — the pre-existing, corpus-wide, explicitly out-of-scope issue noted at
  gate 1 (note #5) and gate 2 (item 6); not introduced or affected by this sweep.
- `python design/gert/scripts/latex.py build` (tectonic, pygmentize installed for `minted`) — **full
  document builds to `main.pdf` (~983 KB) with no new errors.** The same two pre-existing
  `LaTeX Warning: ... undefined` references (`sec:gis:portable-json`, `sec:gcp`) remain, in files this
  sweep did not touch (`03b-interpolation-syntax.tex`, `03c-capture-paths.tex`); no new undefined
  references were introduced by the §03/§08 edits.

## Remaining blocker

None from this sweep's scope. Pre-existing, explicitly-deferred, non-blocking items (unchanged,
out of scope per both gate reviews and this task's instructions):
- Corpus-wide `apiVersion: runbook/v2` vs `runbook.v1.schema.json` mismatch (separate ticket).
- `07-runtime-events.tex` vs `13-evidence-tracing-resumption.tex` `mode: replay` terminology overload
  (flagged by Don, not part of R1–R15/S1/S2, own ticket).
- r23 `assessment.md`/`README.md` companion — still optional, not added.
- Tess's `tv-pkg-resolve.yaml` (≥46 vectors) — unblocked per gate 2, not this revision's task.

**Recommendation:** ready for Barbara's third gate pass (diff-only check of S1/S2 plus the
corpus/schema re-runs she specified).

— Ken (Backend Developer)
