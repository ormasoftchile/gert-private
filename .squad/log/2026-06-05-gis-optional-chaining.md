# 2026-06-05 — GIS Optional-Chaining Ratification & Merge

**Status:** Complete

## Summary
Successfully ratified and merged GIS optional-chaining (`?.`) extension across all four implementation streams:
- **Design:** Proposal drafted, 4 open questions answered by user directive
- **Grammar:** EBNF delta applied to `design/gert/grammar/gis.ebnf` (optional-dot and optional-bracket productions)
- **Specification:** Normative section appended to `design/gert/sections/03b-interpolation-syntax.tex`
- **Conformance:** 15 test vectors created in `design/gert/conformance/tv-gis-path.yaml`

## Key Decisions
- Default missing value: **empty string `""`**
- Short-circuit semantics: **JS/TS-compatible full-tail** (once any segment misses, entire remaining chain → `""`)
- Scope: **GIS only** (does not extend to GXL or GCP)
- `?.[N]` optional bracket indexing: **IN**
- `${?.root}` optional root: **Illegal** (root always mandatory)
- `??` nullish-coalescing: **Deferred**

## All Deliverables Committed
Coordinator committed all changes in commit e5db6b8 and pushed.

## Next Phase
Runtime implementations (Go/C#) can begin parser and evaluator work for `?.` and `?.[N]` operators.
