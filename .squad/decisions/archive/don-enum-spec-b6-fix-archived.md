# B6 Fix — Output `value:` canonical shape block corrected to bare GCP path

**From:** Don (Backend Dev / runtime-contracts)
**Date:** 2026-08-10T13:25-07:00
**Requested by:** Cristián Ormazábal Ortega
**Under:** Barbara's re-gate review (`.squad/decisions/inbox/barbara-enum-spec-regate-review.md`),
Blocker B6.

## Fix

`design/gert/sections/03-schema-vnext.tex:586`, the canonical `outputs:` shape block:

```diff
-    value: "${captured_value}"      # GCP path, resolved at runbook completion
+    value: "captures.<captureName>"      # GCP path, resolved at runbook completion
```

Only the value literal changed, from a GIS `${...}` interpolation template to a bare GCP
path placeholder. The comment (already correct) is untouched. No other line in the file
was modified.

## Verification

1. **Outputs-`value:` sweep.** Every `value:` occurrence in `design/gert/` was enumerated
   and each classified by field:
   - `03-schema-vnext.tex:268` (annotated example, B1) — `"captures.http_result"`. Bare
     GCP path. Consistent.
   - `03-schema-vnext.tex:586` (canonical shape block, this fix) — now
     `"captures.<captureName>"`. Bare GCP path. Consistent.
   - `runbook.v1.schema.json` `$defs.Output.value` (line 784–790) — description states
     "GCP path", "Checked at production time... when 'value' resolves at runbook
     completion (ENUM-009)". No literal example value to drift.
   - `schemas/examples/acme-incident-tools/runbooks/drain-node.yaml:58` (reference
     fixture) — `value: "step.drain.json.drained"`, with the in-line comment "never a
     literal and never a GIS `${...}` template". Bare GCP path. Consistent.
   - `conformance/tv-enum.yaml` — the two remaining `value:` sites under `expected:`
     (line ~1432, `expected.value: ${strategy}`) and no other `outputs.<name>.value`
     fixture entries; the shape used inside `outputs:` blocks throughout the corpus
     (e.g. `outputs: {drained: {type: boolean, value: "step.drain.json.drained"}}`) is
     the bare GCP path form. `expected.value` is a test-vector assertion field, not an
     `Output.value` declaration, and is explicitly modeling the AR-ENUM-7 "no plan-time
     GIS evaluation" case — out of scope per Barbara's note and untouched.
   - `testdata/runbooks/r22-evidence-replay/schema.yaml:188` — `evidence[].value:
     "${env_snapshot}"`. Different field (`evidence[]`, not `outputs.<name>`), explicitly
     named by Barbara as out of scope; correct as written; untouched.

   **Result: after this fix, every `outputs.<name>.value` site in `design/gert/`
   (sections, schema, reference example, corpus) uses the bare GCP-path form. No second
   shape for this field remains anywhere in the tree.**

2. **Corpus.** `python design/gert/scripts/verify_corpus.py` → `OK 423/423 vectors
   validate`. No corpus file was edited (`tv-enum.yaml` untouched; its 58 enum vectors,
   including `ENUM-MOCK`, stand as authored).

3. **`\label`/`\ref` closure.** Unaffected by this fix — line 586 carries no label. No
   `\paragraph{`/label/reference lines were touched.

4. **Scope check.** No `*.ebnf` file touched. No `tool.v1.schema.json` created. No
   `apiVersion` bumped. No enum architecture, AR-ENUM-1..15, or D1–D5 text revisited.
   Only the single YAML literal at `03-schema-vnext.tex:586` changed.

## Standing gate question

Which untouched text did the fix falsify? **Nothing.** The fix brings the canonical shape
block into agreement with text that was already correct (`03:268`, `$defs.Output.value`,
the reference fixture, all 58 vectors); it does not introduce a new claim anywhere else in
the corpus that now needs re-checking.

## Status

B6 resolved. Ready for re-gate. No other findings from B1–B5/L1–L3 reopened; D2–D5 remain
open tickets and are out of scope for this fix.
