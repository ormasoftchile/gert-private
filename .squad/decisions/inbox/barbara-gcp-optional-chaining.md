### 2026-06-07: Q2 — GCP optional chaining (`?.`) arbitration

**Decision:** Option (a) — No `?.` in GCP. GIS keeps `?.` (ratified earlier); GCP does not support it.

**Arbitrated by:** Barbara (Lead/Architect)
**Date:** 2026-06-07T15:06:00-07:00

---

#### Rationale

1. **Zero vector impact.** None of the 41 tv-gcp-path conformance vectors use `?.`. The 6 P7-blocked vectors require the capture service and step output model — not optional chaining. Adding `?.` would create new surface area with no existing test demand.

2. **Different concerns, different syntax.** GIS interpolates into strings where empty-on-miss is cosmetically safe (`"Hello, ${user?.name}"` → `"Hello, "`). GCP captures structured data into PJVM where silent empty-string substitution would mask structural errors in step outputs. The existing `capture_defaults:` mechanism provides explicit, type-checked, runbook-declared defaults — superior to implicit path-level coercion.

3. **Blast radius: zero.** No grammar change. No runtime extension. No corpus changes. No conformance vector reclassification. GNC's implementation already rejects `?.` — this decision ratifies her safe default.

4. **Cross-runtime parity.** Clean, unambiguous answer for C#/TS: "GCP paths are strict; missing segments raise GCP-RESOLVE-002/003. Use `capture_defaults:` for optional-value patterns." No ambiguity about whether `?.` means empty-string or Miss sentinel.

5. **User mental model.** Yes, `vars.user?.id` works differently in `${...}` (GIS) vs `capture.path:` (GCP). This is acceptable because they serve different purposes: interpolation vs structured extraction. The runbook schema already separates these syntactically (`string templates` vs `capture: map entries`), so the contexts are unambiguous.

#### Spec changes made

Added normative subsection `§subsec:gcp:traversal:no-optional-chaining` to `03c-capture-paths.tex` documenting:
- `?.` MUST be rejected in GCP paths (GCP-PARSE-005)
- Rationale for the asymmetry with GIS
- Correct pattern using `capture_defaults:`

#### P7-blocked vectors (unchanged)

The 6 P7-blocked vectors depend on capture service infrastructure (plan-time validation of step IDs, default-policy enforcement, type checking) — none require `?.` syntax. They remain P7-blocked regardless of this decision.

#### Follow-up needed

None. No grammar changes, no runtime extension, no corpus modifications.
