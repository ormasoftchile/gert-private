RESOLVED 2026-06-07 — see barbara-gdp-section-03d.md and barbara-gcp-optional-chaining.md

### 2026-06-07T19-40-00Z: Two GCP spec questions surfaced by GNC (P3 parallel work)

**By:** Coordinator (relaying GNC from gert#26)

**Context:** GNC built the GCP parser + GDP resolver and ran the 41 tv-gcp-path vectors. Result: 35 runnable, 6 explicitly P7-skipped. During implementation, two spec ambiguities surfaced.

---

#### Question 1 — Missing spec section: `design/gert/sections/03d-gdp.tex`

The runtime implementation plan and Barbara's section 09 contract both reference section 03d (GDP — GERT Dotted Path resolution semantics) as the source of truth for path resolution rules. GNC searched and **the file does not exist** in the repo. She inferred GDP semantics from `gcp.ebnf`, `03c-paths.tex`, and the conformance vectors themselves.

**Action needed:** Barbara (or Edith if spec prose) to either:
- Write `03d-gdp.tex` formalizing the GDP resolution rules GNC inferred (and that the 36/36 tv-gxl-path passes prove correct), OR
- Confirm that GDP semantics live entirely in `03c-paths.tex` and update the plan + section 09 contract to stop referencing `03d`.

Either way, the gap should be closed — a runtime implementer reading the plan today would hit the same wall.

---

#### Question 2 — GCP optional chaining (`?.`) conflicts with spec/vectors

GNC found that GCP optional chaining is undefined in the current spec/grammar, and several vectors that LOOK like they want `?.` GCP behavior actually trigger inconsistent results. She **rejected `?.` in the GCP parser** as the safe default — vectors using GCP `?.` now fail (or fall into the 6 P7-skipped bucket if they require capture service).

**Three options for Barbara to arbitrate:**

- (a) **No `?.` in GCP.** GIS keeps `?.` (Brady's earlier ratification), GCP doesn't. Capture defaults handled via `capture.default:` in the runbook, not via path syntax. Document that GIS `?.` and GCP path semantics are intentionally different.
- (b) **Add `?.` to GCP**, mirror GIS semantics (miss → empty-string, JS-style short-circuit). Update grammar, update spec section 03c (or 03d), update affected vectors. GNC would need a follow-up PR to extend her parser + resolver.
- (c) **Add `?.` to GCP but with GCP-specific semantics** (miss → typed Miss sentinel that capture machinery interprets in P7). More work but possibly cleaner for capture defaults.

**Recommendation from GNC (paraphrased):** Option (a) is the safest current default and aligns with how the existing vectors actually behave. If we want `?.` in GCP, it needs an explicit normative decision and grammar update.

**Impact if option (b) or (c):** A few vectors currently in the "35 passing" or "6 P7-skipped" buckets may move; runtime needs a follow-up extension.

---

**Neither question blocks P3 GXL evaluator** (FAO is being dispatched now). They block the next round of GCP corpus completion and the `03d-gdp.tex` cleanup.

**Action items:** Barbara to arbitrate Q2 and decide on Q1's path (write the missing section or amend the plan references) when next active.
