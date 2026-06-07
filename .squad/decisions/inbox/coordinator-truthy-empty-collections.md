**RESOLVED — see [barbara-truthy-arbitration.md](barbara-truthy-arbitration.md) (2026-06-07, Option c: Strict).**

### 2026-06-07T19-00-00Z — Truthy() semantics for empty arrays/objects (needs arbitration)

**By:** Coordinator (relaying Booster from gert#24 P1 work)

**What:** PJVM `Truthy()` is called by GXL boolean coercion (e.g., `if x:`). Section 03a says no implicit truthiness, but empty array `[]` and empty object `{}` need a deterministic answer because the GXL evaluator (lands in P3) will call `Truthy()` on whatever the user passes to a boolean context. Booster's P1 implementation picked one in code to keep PJVM testable, but the choice is not pinned down in the spec.

**Options:**
- (a) **JS-style:** empty array/object → truthy. Only the canonical falsy set (`false`, `null`, `0`, `""`) is falsy.
- (b) **Python-style:** empty array/object → falsy. Falsy set expands to include empty collections.
- (c) **Strict:** any non-bool value passed to a boolean context → parse-gate error OR runtime error. No implicit coercion at all.

**Why it matters now:** P3 GXL evaluator cannot land without this answer. Cross-runtime parity for C# / TS later depends on a single answer that ships in the corpus.

**Recommendation from Booster:** Option (a) — JS-style. Predictable, matches what most users will expect from `${someList ? "yes" : "no"}` style expressions, and is the dominant convention in adjacent expression languages.

**Action items:**
1. Barbara (or Brady) arbitrate Option (a) / (b) / (c)
2. Tess add conformance vectors covering empty `[]` and `{}` in boolean context once decided
3. Update section 03a (or equivalent) to make the rule normative
4. Block P3 (GXL evaluator) until resolved — without this, the evaluator has no defensible answer for `Truthy([])`
