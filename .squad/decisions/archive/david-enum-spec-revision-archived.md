# Enum Spec Revision — Blocker Resolution (B1–B5, L1–L3)

**From:** David (Integration Engineer, revision owner)
**Date:** 2026-08-10T13:25-07:00
**Requested by:** Cristián Ormazábal Ortega
**Against:** `.squad/decisions/inbox/barbara-enum-spec-authoring-gate-review.md`
**Ruling reference:** `.squad/decisions/inbox/barbara-enum-constraint-mvp-architecture-ruling.md` (AR-ENUM-1..15)
**Status:** All five blockers and all three lower-severity items resolved. No design reopened.
No file outside the enum diff's already-affected set touched. Not committed (per instructions).

## Resolution matrix

| # | Item | Fix applied | File(s) |
|---|---|---|---|
| **B1** | Flagship annotated example still authored `from: captures.http_result` | Rewrote to `value: "captures.http_result"` (GCP path, no `from:`). Grepped entire `design/gert/` post-edit: zero authored `from: captures.*` instances remain; the two remaining textual occurrences are the prose passage that explicitly states the form MUST NOT be authored/is rejected, and a schema description documenting the historical drift (D1) — both intentional. | `sections/03-schema-vnext.tex` (`outputs.health_status` block) |
| **B2** | Output `type` prose disagreed with `$defs.Output` (added `secret`, dropped `integer`) | Changed prose line to `string \| number \| integer \| boolean \| object \| array`, exact field-for-field match with `$defs.Output.type.enum`. | `sections/03-schema-vnext.tex` (§Output Declarations shape block) |
| **B3** | §Output enum paragraph asserted an unauthorable `default` on `value:` | Removed the claim. `$defs.Output` genuinely has no `default` property and none was added — see "Tess's finding" resolution below. Replaced with an explicit statement that no default-membership obligation attaches to S4, and a cross-reference to `06-tool-runtime.tex` §Action Substitution §Output contract for the S2 (substituted tool action) default rule, which already states ENUM-006/PKG-027 independence correctly. D5 (06's own schema-gap ticket) left untouched, exactly as instructed. | `sections/03-schema-vnext.tex` (§Output enum paragraph, both occurrences of the `\paragraph{enum...}` intro sentence) |
| **B4** | `$defs.Input.enum` description asserted `type` may be absent; `$defs.Input.required` includes `type` | Struck "(or absent, since string is the default type)" from the schema description; added a sentence stating `type` is REQUIRED so an omitted `type` is a structural schema error, never `ENUM-001`. Mirrored the correction in `03` §Input Declarations prose (the `enum:` paragraph and the `type field` paragraph both now state `type` is required and that omission is a structural error, not an enum error). `Input.required` list unchanged (`["type"]`). | `schemas/runbook.v1.schema.json` (`$defs.Input.properties.enum.description`); `sections/03-schema-vnext.tex` |
| **B5** | `\paragraph{type field}` used inline, mid-sentence, breaking the paragraph | Added `\label{para:type-field}` immediately after the (unchanged, standalone) `\paragraph{type field}` heading, and replaced the inline `\paragraph{...}` invocation with a proper `\S\ref{para:type-field}`. Swept the full enum diff (`grep` for `\paragraph{...}` used outside its own heading line) — this was the only offender; all ~140 other `\paragraph{...}` occurrences in the affected files are standalone heading uses, not inline. | `sections/03-schema-vnext.tex` |
| **L1** | Input YAML shape block listed `type: string \| number \| boolean \| secret`, missing `array`/`object` present in the bullets and in `$defs.Input.type` | Updated shape block to `type: string \| number \| boolean \| secret \| array \| object`. | `sections/03-schema-vnext.tex` (§Input Declarations shape block) |
| **L2** | `03d` Enum Constraint Error Codes table doesn't restate the §10/§08 redaction rule (unlike `03`/`08`) | Added one sentence after the table cross-referencing `08-security-and-trust.tex` §Secret Resolution and Redaction, stating the `"<redacted>"` + `member_count` form applies to all `ENUM-*` output. | `sections/03d-parse-time-enforcement.tex` |
| **L3** | `13` and `03d`/`03` described the redacted form inconsistently ("replaced with a member count" vs. the exact two-field `"<redacted>"` + `member_count` shape) | Unified all three sites on `03d`'s exact wording: literal string `"<redacted>"` plus a `member_count` integer. | `sections/13-evidence-tracing-resumption.tex`, `sections/03-schema-vnext.tex` (both now match `sections/03d-parse-time-enforcement.tex` §The `ValidatedPlan` Type Contract, which was already correct and untouched) |

## Tess's finding — `$defs.Output` has no `default` — resolution (part of B3)

**Question:** does the ratified four-site MVP require a `default` field on runbook-level
`Output` (S4)?

**Finding, stated without guessing:** No. Reading AR-ENUM-6 and AR-ENUM-9 together:

- AR-ENUM-6's summary line ("Applies at S1, S3, S4 and to a substituted action's declared
  output default") is a loosely-written enumeration, not four independent default-bearing
  fields. AR-ENUM-9 §"PKG-027 interaction" — the ruling's only detailed treatment of an
  output-default scenario — ties the default-membership rule specifically to **"a
  substituted action's declared output"** (S2, tool-side, `06` §Action Substitution §Output
  contract, which is prose-only per C3 and already carries `default` informally). It never
  describes an S4 (runbook `Output`) mechanism by which a value is "otherwise unproduced" and
  falls back to a literal default — S4's only production path is `value:` (a GCP path)
  resolving at runbook completion; there is no unproduced/fallback case for a GCP-path field
  the way there is for a tool action's own output.
- `$defs.Output`'s properties (`type`, `description`, `value`, `enum`) were hand-authored
  post-ruling and deliberately have no `default` — unlike `$defs.Input`, which does. This is
  consistent with S4 having no default concept, not an oversight parallel to Input's.
- Barbara's gate review (B3) explicitly rules the S4 restatement in `03` as inventing an
  unauthorable field and directs deletion with a cross-reference to the S2 (tool-side) rule
  — the same conclusion reached independently above.

**Disposition:** `$defs.Output` is **not** changed to add `default`. This is a genuine schema
limitation that is now made coherent in prose (B3 fix above), not a gap to close.

**Effect on Tess's corpus:** none required. Her `TV-ENUM-DEFAULT-003` (S2 substituted-action
default, schema-free C3 site) and `TV-ENUM-DEFAULT-006` (S4 documentation-only vector, tagged
`untestable-in-corpus-format`/`schema-gap`) were already the correct shape under this reading
— S4 has no literal-default site to test because none exists by design, not because of an
accidental omission. Tess's corpus is unmodified (owner-locked, per instructions); her existing
`note` fields remain accurate and do not need updating, since "untestable" and "schema gap" both
remain true statements about the artifact even though the underlying cause is now understood to
be intentional rather than an oversight. No corpus-owner action requested.

## Verification performed

- **`from: captures` sweep:** `grep -rn "from:\s*captures" design/gert/` → 2 hits, both intentional
  (rejection-statement prose in `03-schema-vnext.tex` and drift-history schema description in
  `runbook.v1.schema.json`); zero authored examples remain.
- **Schema/prose field-for-field reconciliation:** `$defs.Input.required == ["type"]`,
  `$defs.Input.type.enum == [string, number, boolean, secret, array, object]`,
  `$defs.Output.required == ["type"]`, `$defs.Output.type.enum == [string, number, integer,
  boolean, object, array]`, `$defs.Output` has no `default` key — all confirmed via
  `jsonschema.Draft202012Validator.check_schema` + direct property inspection, and all now match
  the corresponding `03-schema-vnext.tex` prose lines verbatim.
- **`design/gert/conformance/tv-*.yaml` corpus:** `python scripts/verify_corpus.py` →
  `OK 423/423 vectors validate` (includes Tess's `tv-enum.yaml`, unmodified).
- **LaTeX `\label`/`\ref` closure:** grep-based closure check across all of
  `design/gert/sections/*.tex` (296 labels, 127 unique `\ref`/`\S\ref` targets) → **0 dangling
  references**, including the newly added `para:type-field` label/ref pair.
- **LaTeX compile:** `pdflatex -shell-escape main.tex` (3 passes) → 271-page PDF produced
  successfully. Two pre-existing warnings not touched by this revision and out of scope
  (`sec:gis:portable-json`/`sec:gcp` undefined — `03b`/`03c` are not `\input` by `main.tex` at
  all, a pre-existing structural fact unrelated to enum semantics; one Unicode-character error
  in `08-security-and-trust.tex:767`, pre-existing, unrelated to any B1–B5/L1–L3 edit). Neither
  is in Barbara's blocker list; neither was introduced or touched by this revision. Build
  artifacts (`main.aux/.log/.pdf/...`, `_minted-main/`) generated during validation were removed
  afterward; none were tracked by git (all gitignored).
- **Scope discipline:** no `tool.v1.schema.json` created; no `*.ebnf` touched; no `apiVersion`
  bumped; `tool-package.v1`/`project-config.v1`/`package-lock.v1` schemas untouched; Edith not
  consulted; Tess's corpus (`tv-enum.yaml`) not modified; runtime code (`C:\One\OpenSource\gert`)
  not touched.

## Files changed

- `design/gert/sections/03-schema-vnext.tex`
- `design/gert/sections/03d-parse-time-enforcement.tex`
- `design/gert/sections/13-evidence-tracing-resumption.tex`
- `design/gert/schemas/runbook.v1.schema.json`

No other files in the enum diff were modified. All other pre-existing uncommitted changes
(tool-packages workstream files, `tv-enum.yaml`, `tv-pkg-resolve.yaml`, example fixtures, etc.)
are untouched and preserved exactly as found.

## Which untouched text did this fix falsify? (standing gate question)

None found. The B1/B2/B4/L1 fixes make previously-contradicting text agree with the canonical
schema; the B3 fix removes a claim rather than asserting a new one, and the cross-reference it
adds (`06` §Action Substitution §Output contract) already stated the S2 rule correctly and is
unchanged. The B5 fix only converts an inline sectioning command to a `\S\ref` and adds the
label that command was missing — no prose meaning changes. L2/L3 add/align redaction wording
that was already normative and correct in `03d`; they do not introduce a new rule.

## Remaining blocker

None from B1–B5 or L1–L3. Two pre-existing, out-of-scope issues remain on record for future
tickets, neither raised by this revision and neither blocking: (1) `main.tex` does not `\input`
`03a`/`03b`/`03c`/`03d` at all, so a subset of `\ref`s resolve only under the fuller
grep-based cross-file closure check, not a `main.tex` compile — a pre-existing build-target
scoping fact; (2) one pre-existing Unicode-character LaTeX compile error at
`08-security-and-trust.tex:767`, unrelated to enum semantics. D2–D5 tickets from Barbara's
review remain open and are explicitly not addressed here, per her instructions.

Re-gating on B1–B5 requested from Barbara.
