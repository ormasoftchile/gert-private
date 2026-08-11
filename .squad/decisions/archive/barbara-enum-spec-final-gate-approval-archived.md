# Final Gate (Gate 3) — `enum` String Constraint MVP: **APPROVED**

**From:** Barbara (Lead / Architect, final gate)
**Date:** 2026-08-10T14:50-07:00
**Requested by:** Cristián Ormazábal Ortega
**Under review:** Don's B6 fix (`.squad/decisions/inbox/don-enum-spec-b6-fix.md`), authored
under author lockout (Edith and David both locked out), against my re-gate
(`.squad/decisions/inbox/barbara-enum-spec-regate-review.md`, sole blocker B6) and the
architecture ruling (`barbara-enum-constraint-mvp-architecture-ruling.md`, AR-ENUM-1..15,
C1/C2/C3, D1).
**Method:** working-tree diff read directly; the `outputs.<name>.value` sweep re-run
independently by structural YAML/TeX block walk (not by grep on the value literal, which
was the check that failed to catch B6 the first time); all five JSON Schemas parsed and
`$defs.Input`/`$defs.Output` extracted programmatically; full `\label`/`\ref` closure and
inline-`\paragraph{` sweep re-run across `design/gert/sections/`; `tv-enum.yaml` re-parsed
and re-counted; corpus verifier run. No product/design artifact was modified by this review.

**VERDICT: APPROVED.**

---

## 1. B6 — verified fixed, exactly and only as specified

`design/gert/sections/03-schema-vnext.tex:586`, the canonical `outputs:` shape block:

```
    value: "captures.<captureName>"      # GCP path, resolved at runbook completion
```

The GIS interpolation `"${captured_value}"` is gone; the placeholder is a bare GCP path;
the (already-correct) comment is preserved verbatim. This is the exact remedy I specified,
in the form I offered as the first of two acceptable options.

**Scope containment verified, not taken on report:** `03-schema-vnext.tex` is the only file
under `design/` with a modification timestamp after the re-gate (14:48 vs. the 13:45–13:47
batch that predates it); every other tracked file in the diff is untouched since Gate 2.
`git grep captured_value -- design/` returns **zero hits** tree-wide. No `*.ebnf` change
beyond the pre-existing tool-packages §3.4a `LocalOutputs` block, no `tool.v1.schema.json`,
no `apiVersion` bump, `tv-enum.yaml` byte-identical.

## 2. Independent sweep for remaining identical canonical contradictions

I walked every YAML/JSON/TeX file under `design/gert/` structurally, entered each `outputs:`
mapping, and enumerated every `value:` key declared inside one. Complete result — **11 sites,
zero contradictions**:

| Site | Form |
|---|---|
| `03-schema-vnext.tex:268` (annotated example, B1) | `"captures.http_result"` — bare GCP |
| `03-schema-vnext.tex:586` (canonical shape block, B6) | `"captures.<captureName>"` — bare GCP |
| `schemas/examples/acme-incident-tools/runbooks/drain-node.yaml:58` (reference package) | `"step.drain.json.drained"` — bare GCP |
| `conformance/tv-enum.yaml` ×4 (1109, 2015, 2042, 2073) | `"step.drain.json.drain_state"` — bare GCP |
| `runbook.v1.schema.json:48`, `14-adapter-contracts.tex:303`, `testdata/tools/{jsonrpc,mcp}-server.tool.yaml` | inline/empty `outputs`, no `value` literal |

`$defs.Output.value.description` is prose-only ("GCP path… **evaluated as GCP at runtime**")
with no example literal to drift. **One shape, one field, tree-wide.** The D1/B1/B6
pathology — two shapes for one field — is now closed at every surface.

Every other `value:` in the tree (219 total occurrences) was classified and belongs to a
**different** field: conformance `expected.value` assertions, `evidence[].value`
(`r22-evidence-replay/schema.yaml:188`, `"${env_snapshot}"` — a GIS template, and correct
there), and collector `options[].value` (`03:1361-1367`, `r05:115-121`, `r08:200-204`).
These are correct as written and were correctly left alone, as I directed.

I also swept the diff's *added* lines for any other GIS template appearing where a GCP path
is normative: exactly one hit, `\texttt{\${count}}` in the AR-ENUM-7 coercion prose, which
is deliberately a GIS interpolation and is the correct illustration there.

## 3. Corpus and schema integrity

- `python design/gert/scripts/verify_corpus.py` → **`OK 423/423 vectors validate`**.
- All five JSON Schemas parse: `runbook.v1`, `tool-package.v1`, `project-config.v1`,
  `package-lock.v1`, `conformance/vector.schema.json`.
- `$defs.Input`: `required: ["type"]`, `additionalProperties: false`,
  `type.enum = [string, number, boolean, secret, array, object]` — element-for-element and
  in-order identical to the `03:370` shape block (L1 holds).
- `$defs.Output`: `required: ["type"]`, `additionalProperties: false`,
  `type.enum = [string, number, integer, boolean, object, array]` — identical to the
  `03:584` shape block (B2 holds); **no `default` property**, which is what makes the B3
  replacement text true rather than merely asserted.
- Both `enum` subschemas carry `minItems`/`uniqueItems`/`items` (AR-ENUM-3 shape).
- `tv-enum.yaml`: **58 vectors**, no duplicate ids, `ENUM-MOCK` present (5), all eight
  categories populated. Above the 54 minimum. Unmodified since Gate 2.
- LaTeX closure across `design/gert/sections/`: **zero duplicate labels, zero dangling
  `\ref`s, zero inline `\paragraph{`**. `para:type-field` 1 label / 1 ref;
  `para:enum-constraint` 1 label / 9 refs (B5 holds).

## 4. Standing gate question — which untouched text did the fix falsify?

**Nothing.** The fix moves one placeholder literal into agreement with four artifacts that
were already correct and normative (`$defs.Output.value`, `03:268`, the reference fixture,
all 58 vectors). It asserts nothing new, so it creates no new obligation elsewhere. I
confirmed this rather than accepting Don's assertion of it: the only text keyed to that
line is the `value:` paragraph immediately below it (`03:589-602`), which already described
the GCP form and now agrees with the block above it for the first time.

## 5. Disposition

- **B1–B6, L1–L3: all CLOSED.** No blocker, no required lower-severity fix, remains open.
- **D2, D3, D4, D5 remain open tickets**, unchanged and explicitly out of scope: `$defs.Input`
  missing `pattern`/`example` (D2 — note this is a *different* class from B6: a canonical
  block declaring keys the schema forbids under `additionalProperties: false`, ruled a
  separate ticket in the architecture ruling §16 and not to ride along on this PR),
  `$defs.Input.from` vs. §03's binding forms (D3), the `\S\ref` inside a `minted` comment
  at `03:184` (D4), and the `06` §Output contract "declare a default" drift (D5).
- **AR-ENUM-1..15, C1/C2/C3, D1: all discharged.** No design is reopened by this approval.

## 6. Authorisation

**Runtime implementation of the `enum` string constraint MVP is hereby AUTHORISED** in
`C:\One\OpenSource\gert`, against this design as it now stands in the working tree.
Implementation is bounded by: ENUM-001..009 + ENUM-W001 with no renumbering and no
ENUM-010; the four declaration sites only; the four-step binding order with type-error-wins
single-cause reporting; the asymmetric NFC rule; SET-equality contract identity under
PKG-013; `enum_constraints` carried once in the ValidatedPlan with C1 redaction
(`"<redacted>"` + `member_count`) and never in per-step event payloads; and no grammar or
`apiVersion` change. `tv-enum.yaml`'s 58 vectors are the acceptance gate for the
implementation — they are frozen as authored and are to be run, not edited, by the
implementer. Any implementation finding that appears to require a design change comes back
to me as a new gate; it is not resolved in code.

**Tess remains unblocked. Edith's and David's lockouts are lifted.**

**Conformance status: green — 423/423, 58 enum vectors incl. `ENUM-MOCK`. Design CLEARED
for implementation.**
