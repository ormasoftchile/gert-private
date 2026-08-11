# Question for Barbara — String-only `enum:` on tool-action args/outputs and runbook inputs/outputs

**From:** Edith (Spec Editor)
**Date:** 2026-08-10T13:25-07:00
**Status:** OPEN — analysis only, no artifacts changed
**Requestor of underlying feature:** Cristián Ormazábal Ortega

## Context

Assessing a request to add a string-only `enum:` constraint to:
1. Runbook-level `inputs.<name>` (`$defs/Input`, `runbook.v1.schema.json`)
2. Runbook-level `outputs.<name>` (`$defs/Output`, `runbook.v1.schema.json`)
3. Tool-action `args:` entries (`.tool.yaml`, prose-only today — no `tool.v1.schema.json` exists; §`args:` vs `inputs:`, `06-tool-runtime.tex` L295)
4. Tool-action `outputs:` entries for substituted actions (same section, L318)

This is architecturally load-bearing enough (new error-code family, parse-time vs. runtime split, cross-artifact schema changes) that I want your ruling before authoring, per my charter ("I do not invent semantics").

## Q1 — Where does enum enforcement happen: parse time, runtime, or both?

Unlike `pattern:` (a regex the schema/spec already describes for string `Input`, `03-schema-vnext.tex` L363 — itself never actually reached the hand-authored `$defs/Input` in `runbook.v1.schema.json`, so it is dead prose today, see Q4), an `enum:` on `args:`/`inputs:` constrains the **resolved value**, not the GIS-templated expression text bound to it. A literal default or literal caller-supplied value can be checked at parse time (mirrors `capture.default:` scalar-type checking, `GCP-TYPE-001`/`PLAN-003`, `03c-capture-paths.tex` §`capture.default:` Policy); a value arriving via `${...}` interpolation, `from: prompt`, `from: env`, or a caller binding can only be checked after evaluation.

Two prior-art patterns exist in the spec and I need to know which one enum follows:
- **`GCP-TYPE-001`/`PLAN-003` pattern**: static check when the value is a literal, silently deferred (no runtime check at all) otherwise.
- **`GXL-TYPE-003` pattern**: a pure runtime/eval-time check, no static component.

**Recommendation:** enum should be checked both ways — reject a literal (non-templated) value or `default:` at parse time if it is not in the enum set (new `PLAN-0xx`/`SEM-0xx` code), and additionally enforce at runtime, after GIS evaluation, immediately before the value is bound into the execution context (new `SEM-0xx` or a new prefix — see Q3). This matches how `Input.type` is effectively enforced today (structural for literals, semantic/runtime for the rest) and avoids a silent gap where every non-trivial input would bypass validation entirely.

## Q2 — Runbook `Output` schema vs. prose disagree today; which is canonical for adding `enum:`?

Pre-existing conflict, unrelated to enum, but I must pick a side before I can place `enum:` on `Output`:

| | `03-schema-vnext.tex` §Output Declarations (L403–420) | `runbook.v1.schema.json` `$defs/Output` (schema, hand-authored) |
|---|---|---|
| `type` enum | `string \| number \| boolean \| secret` | `string, number, integer, boolean, object, array` |
| value source | `from: captures.<captureName>` (required) | `value: <GCP path>`, evaluated as GCP at runtime |

These are two different vocabularies for the same concept. The schema is newer (post-runbook-v1 authoring) and matches what the 23 example runbooks actually use; the prose predates it and was never reconciled. I will treat the **schema as canonical** per the same rule my charter applies to grammar-vs-prose conflicts, and rewrite the L403–420 prose to `value:`/GCP-path form as a byproduct of this task — flagging here rather than silently fixing, since it is a substantive prose rewrite outside the literal scope of "add enum." **Confirm this is the correct resolution, or tell me to leave `Output` out of the enum feature until that conflict is resolved separately.**

## Q3 — New error-code family: `PKG-03x` (extends Tool Packages catalog) or a new prefix?

Tool-action `args:`/`outputs:` violations currently live in the `PKG-0xx` catalog (`03d-parse-time-enforcement.tex`, e.g. `PKG-013` type/key-set mismatch, `PKG-026` `from:` prohibition, `PKG-027` unproduced output) even though they are not about package *resolution* per se — they're about the action contract. Runbook-level `Input`/`Output` enum violations have no natural PKG-prefixed home (Inputs/Outputs are not package-scoped).

**Recommendation:** two codes, not one prefix reused awkwardly:
- `SEM-0xx` (next free `SEM-` number) for runbook `inputs.*`/`outputs.*` enum violations (parse-time literal check) and its runtime counterpart.
- `PKG-030` for tool-action `args:`/`outputs:` enum violations (continues the existing sequence; `PKG-029` is the last allocated code as of the tool-packages ruling).

## Q4 — `pattern:`/`example:` on `Input` are prose-only; should enum land in schema-only, prose-only, or both, and should the dead `pattern`/`example` fields be fixed in the same pass?

`03-schema-vnext.tex` L358–363 documents `pattern:` (regex) and `example:` as `Input` fields; `runbook.v1.schema.json` `$defs/Input` has `additionalProperties: false` and does **not** include either field, so a runbook that supplies `pattern:` or `example:` today fails schema validation — the prose has been unreachable since the hand-authored schema was written (2026-06-07) and was not caught by any of the 23 conformance examples (none use `pattern:`/`example:`). This is not part of my task, but `enum:` is the same kind of field (string-only, sits beside `type`) and I don't want to add a fourth orphaned field to the same paragraph.

**Recommendation:** fix `pattern:`/`example:` in the same PR that adds `enum:` (add both to `$defs/Input` in the schema, since the prose already promised them and no example contradicts it), rather than leave a third generation of prose/schema drift. Flagging for your sign-off since it technically expands scope beyond "add enum."

## Not blocking — proceeding on these defaults unless you say otherwise

- `enum:` values MUST be `type: string` in this task (per the request); non-string `type` + `enum:` present is a schema-level error (`additionalProperties`/`oneOf` gate), not deferred to runtime.
- `enum:` is OPTIONAL and mutually compatible with `pattern:` (both narrow the string space; the runtime applies both when present — enum is checked first since it's the tighter, order-independent check).
- `default:`, when present alongside `enum:`, MUST be a member of the `enum:` list; violation is a parse-time error (literal-vs-literal, always checkable), not deferred.
- Tool-action `args:`/`outputs:` follow the exact same `enum:` shape as `Input`/`Output` (single vocabulary principle, `06-tool-runtime.tex` L295) — I will not invent a second enum grammar for tools.
- No JSON Schema exists yet for `.tool.yaml` (`tool.v1.schema.json` is prose-only per `06-tool-runtime.tex` L328–331, contradicting the "shipped, codegen'd" claim in `03-schema-vnext.tex` §Schema Artifacts, L3691–3709 — separate pre-existing inconsistency, not blocking, flagged for awareness only). I will encode tool-action `enum:` as normative prose in §`args:` vs `inputs:` alongside the existing `{type, required, description, redact}` shape, matching how the rest of the tool-action contract is specified today.
