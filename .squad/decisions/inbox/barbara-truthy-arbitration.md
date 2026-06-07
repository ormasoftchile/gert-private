## Truthy() Arbitration for Empty Collections — Decision Record

**Date:** 2026-06-07
**Author:** Barbara (Lead/Architect)
**Requested by:** Coordinator (relaying Booster, P1 PJVM)
**Status:** RATIFIED

---

### Question

PJVM `Truthy()` is called by GXL boolean coercion in contexts like `if x:` or `${x ? a : b}`. Spec section 03a says "no implicit truthiness" but does not explicitly pin down the behavior for every PJVM type (especially empty collections `[]` and `{}`) when they end up in a boolean context. Booster's P1 PJVM picked JS-style truthiness to make tests pass; without a normative answer, P3's GXL evaluator would lock in a de-facto behavior that C#/TS runtimes would have to reverse-engineer.

### Options Considered

- **(a) JS-style:** empty array/object → truthy. Only canonical falsy set (`false`, `null`, `0`, `""`) is falsy.
- **(b) Python-style:** empty array/object → falsy. Empty collections join the falsy set.
- **(c) Strict:** non-bool value in boolean context → `GXL-TYPE-002` error. No implicit coercion at all.

### Decision

**Option (c) — Strict.** A value in boolean context MUST have PJVM type `bool`. Any non-bool value raises `GXL-TYPE-002`. There is no `Truthy()` coercion function.

### Rationale

The spec already mandates this. Section 03a states "Authors MUST NOT rely on implicit truthiness" and `gxl.ebnf` §5.2 states "A non-bool value in boolean position → GXL-TYPE-002." Choosing (a) or (b) would contradict existing normative text and require a spec *change*, not a clarification. Option (c) confirms what the spec already says.

From a cross-runtime parity perspective, strict typing is trivially portable: every runtime checks `type == bool` and raises `GXL-TYPE-002` otherwise. Options (a) and (b) each require every runtime to implement an identical truthiness table — a needless divergence surface. The blast radius of (c) is zero: it changes no existing behavior, adds no new coercion paths, and leaves the type system closed. Users who want emptiness checks use explicit idioms (`len(x) > 0`, `x != null`, `n != 0`, `s != ""`), which are self-documenting and unambiguous.

The "is this list empty?" idiom (`if myList:`) is common in Python/JS, but GXL is not those languages. GXL's value proposition is deterministic, auditable evaluation — implicit coercion undermines that. The explicit `len(myList) == 0` form is four characters longer and infinitely clearer in an audit trail.

### Cross-Runtime Implication

All GXL runtime implementations (Go, C#, TypeScript) MUST enforce the strict boolean-context gate identically: only `bool` values pass; all other PJVM types trigger `GXL-TYPE-002` — no runtime may introduce its own truthy/falsy table.

### Spec Section Updated

`design/gert/sections/03a-expression-language.tex` — new subsection "Truthy Coercion in Boolean Context" (§`sec:gxl:semantics:truthy`) with normative rule, worked examples for all PJVM types, recommended idioms, and cross-runtime note.

### Grammar

`design/gert/grammar/gxl.ebnf` — no change needed. Boolean context is a runtime evaluation concept, not a grammar-level production. The EBNF already documents the rule in §5.2 commentary ("A non-bool value in boolean position → GXL-TYPE-002"). The grammar defines where boolean expressions appear syntactically (after `not`, both sides of `and`/`or`); the type check is enforced at eval time, not parse time.

### Corpus Vectors Added

See [barbara-truthy-corpus-vectors.md](barbara-truthy-corpus-vectors.md) — ~14 new conformance vectors for Tess to add to `tv-gxl-eval.yaml`.

### Impact on Booster's P1 PJVM

If Booster's P1 PJVM implements a `Truthy()` function with JS-style semantics, it must be corrected: the function should accept only `bool` and return the bool value directly, or (preferably) the evaluator should perform the type check inline without a separate `Truthy()` method. Either way, the runtime MUST raise `GXL-TYPE-002` for non-bool inputs in boolean context.
