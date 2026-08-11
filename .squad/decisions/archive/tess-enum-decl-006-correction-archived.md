# `TV-ENUM-DECL-006` Corrected Per Implementation-Gate Ruling §2a

**From:** Tess (Conformance Tester)
**Date:** 2026-08-10T13:25:10.498-07:00
**Requested by:** Cristián Ormazábal Ortega
**Under:** `barbara-enum-mvp-implementation-gate.md` §2a (REJECTED gate, disposition binding on Tess),
which itself corrects `barbara-enum-constraint-mvp-architecture-ruling.md` AR-ENUM-3 rule 3's
illustrative aside (normative clause unchanged).

**Action taken:** Amended `design/gert/conformance/tv-enum.yaml`, `TV-ENUM-DECL-006` only. No other
vector, no schema, no enum architecture, no runtime code changed. Not committed.

## What was wrong

The vector's `description`/`note` asserted, as fact: *"Unquoted enum members yes/no resolve to YAML
1.2 core-schema booleans, not strings."* That is false. The YAML 1.2 core schema's `tag:yaml.org,2002:bool`
resolves only from `true|True|TRUE|false|False|FALSE`. Unquoted `yes`/`no` resolve to `tag:yaml.org,2002:str`
under the 1.2 core schema (they were booleans under the older, non-core YAML 1.1 schema/PyYAML's legacy
resolver — a different, non-normative resolver). Barbara's own AR-ENUM-3 rule 3 normative clause
("a YAML scalar that resolves to a string **under the YAML 1.2 core schema**") already said the correct
thing; the rule's parenthetical aside contradicted its own normative clause, and the vector codified the
aside. Don's runtime implementation is correct (`enum: [yes, no]` validates and runs clean); the vector's
fixture was defective, not the runtime.

## What changed

- `enum: [yes, no]` → `enum: [true, false]`.
- `description`: now states the correct core-schema fact and gives the correct authoring guidance
  (quote `"true"`/`"false"` if a string member must have that spelling).
- `note`: rewritten to the same effect, referencing AR-ENUM-3(3) as before.
- `id`, `category`, `expected` (`ENUM/ENUM-002`), and `tags` (`yaml-1.2-core-schema`) all **unchanged**.

## Why this preserves coverage intent

The vector's purpose is to pin: *an author who unquotes an enum member that happens to collide with a
YAML 1.2 core-schema keyword scalar gets ENUM-002, not silent coercion.* `true`/`false` are the only
canonical spellings the 1.2 core schema actually resolves to non-string types among common short tokens,
so `[true, false]` is the correct, narrowest fixture for exactly that intent — `[yes, no]` never was.
Verified in Python: `yaml.safe_load("enum: [true, false]")` → `{'enum': [True, False]}` (both `bool`);
the old fixture would have loaded as `['yes', 'no']` (both `str`) and therefore never triggered ENUM-002
under a spec-conformant parser.

## Verification

- `python design/gert/scripts/verify_corpus.py` → `OK 423/423 vectors validate`.
- Vector count in `tv-enum.yaml` unchanged at 58; `ENUM-DECL` category count unchanged at 14.
- No other file touched; `git status` shows only `tv-enum.yaml` modified in this change.

## Handoff

- **Edith** owns the parallel textual fix in `03-schema-vnext.tex` (strike the YAML-1.1 aside from
  AR-ENUM-3 rule 3's rendering, replace the "most likely trap" example with a 1.2-core-schema-accurate
  one) per the same gate ruling §2a. Not done here — out of scope for this correction.
- **Ken** (R2, revision owner for the implementation re-gate): the data-driven harness over
  `tv-enum.yaml` should assert `ENUM-002` for `TV-ENUM-DECL-006`'s corrected fixture
  (`enum: [true, false]`) exactly as for the rest of `ENUM-DECL`; no special-casing needed now that the
  fixture is well-formed-per-spec.
- No further action requested from Barbara; this is the single authorised edit per her gate's
  disposition and it is complete.
