### 2026-06-07: Q1 — Missing GDP resolution section (03d-gdp.tex)

**Decision:** Option (B) — Do not create a separate `03d-gdp.tex`. GDP resolution semantics remain where they already live.

**Arbitrated by:** Barbara (Lead/Architect)
**Date:** 2026-06-07T15:06:00-07:00

---

#### Rationale

The "missing 03d-gdp.tex" was a misidentification. The file `03d-parse-time-enforcement.tex` already exists and is correctly referenced by the runtime plan for parse-gate semantics — not for GDP resolution. GDP resolution semantics are already fully specified across two locations:

1. **`03a-expression-language.tex` §subsec:gxl:gdp** — defines the GDP grammar (`GDP = IDENT { "." IDENT | "[" INTEGER "]" }`) and variable-access traversal rules.
2. **`03c-capture-paths.tex` §sec:gcp:traversal** — defines GDP path traversal in the GCP context (source-qualified), including error codes for missing keys (GCP-RESOLVE-002) and out-of-bounds indices (GCP-RESOLVE-003).

Section 03c explicitly states: "GDP itself is defined in §subsec:gxl:gdp and is referenced here; it is not duplicated." GNC built her resolver successfully from these two sections plus the grammar — confirming they are sufficient. Creating a third location would introduce maintenance burden and duplication risk without adding new normative content.

#### Cross-runtime impact

Zero. C#/TS implementers follow the same two sections. No references need updating — the runtime plan's `§03d` citations correctly point at parse-time enforcement.

#### What changed

No spec files created or removed. This decision confirms the status quo and closes the question.
