# Skill: JSON Schema Authoring for Grammar-Backed Document Languages

**Slug:** `json-schema-grammar-backed`  
**Author:** Edith (Spec Editor)  
**Extracted:** 2026-06-07T19:14:39-07:00 from runbook v1 schema task  
**Applicable to:** Any GERT document format where the spec is defined by EBNF grammars + LaTeX prose and the canonical Go struct surface is a *reference implementation*, not the source of truth.

---

## When to Apply This Skill

Use this skill when authoring a JSON Schema (draft 2020-12) for a YAML document format that:
- Has a formal grammar (EBNF) defining its expression language
- Has a LaTeX spec that is the normative authority
- Has Go (or other language) structs that implement the spec but are not the source of truth
- Will be consumed by multiple language runtimes (Go, C#, TypeScript)
- Needs to distinguish structural validity from expression/semantic validity

---

## Checklist

### 1. Read Before Writing

- [ ] Read ALL grammar files (`*.ebnf`) relevant to the document format
- [ ] Read the spec sections (`.tex`) for field definitions, required/optional status, and enum values
- [ ] Read the Go struct surface as a **field inventory checklist** (not as normative authority)
- [ ] Read all existing examples (`.yaml` files) — these are your positive validation set
- [ ] Read existing sibling schemas for style/convention consistency (e.g., `conformance/schema.json`)
- [ ] Read the architectural decision that scopes your task (where is canonical home, what is the boundary)

### 2. Schema Header

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "<canonical-url-from-decision>",
  "title": "<Human-readable title>",
  "description": "Structural contract for <format>. Validates structure and types only — <LANG> expression semantics are enforced at runtime, not by this schema."
}
```

### 3. Discriminated Unions (Step Types, Flow Nodes)

For a type-discriminated object (e.g., a step with a `type` field), use:

```json
{
  "type": "object",
  "required": ["type"],
  "properties": {
    "type": { "enum": [...] },
    ... (all common fields)
  },
  "allOf": [
    {
      "if": { "required": ["type"], "properties": { "type": { "const": "foo" } } },
      "then": {
        "required": ["foo_payload"],
        "properties": { "foo_payload": { "$ref": "#/$defs/FooPayload" } }
      }
    }
    ... (one arm per type value)
  ],
  "unevaluatedProperties": false
}
```

**Why `unevaluatedProperties: false` and not `additionalProperties: false`:**  
`additionalProperties` is annotation-aware only in the same schema object. `unevaluatedProperties` is annotation-aware across all applied subschemas including `if/then/else` and `$ref`. Only draft 2020-12 supports this correctly.

**Why `required: ["type"]` in the `if`:**  
Prevents the condition from matching vacuously when `type` is absent (e.g., during partial validation).

### 4. Exclusive One-of-N Keys (Flow Nodes)

For a "one of these keys must be present and only one" pattern:

```json
{
  "type": "object",
  "properties": {
    "a": { "$ref": "#/$defs/A" },
    "b": { "$ref": "#/$defs/B" },
    "c": { "$ref": "#/$defs/C" }
  },
  "additionalProperties": false,
  "oneOf": [
    { "required": ["a"] },
    { "required": ["b"] },
    { "required": ["c"] }
  ]
}
```

`oneOf` fails if 0 or 2+ alternatives match — which correctly enforces "exactly one key present".

### 5. Opaque Expression Strings

For any string field that holds a GIS template, GXL expression, or GCP capture path:

```json
"when": {
  "type": "string",
  "description": "GXL boolean condition. Evaluated as GXL at runtime; structural validation only at parse time."
}
```

Do NOT attempt to regex-validate the expression syntax. Expression syntax belongs to the runtime evaluator.

### 6. `additionalProperties` Policy

| Case | Use |
|---|---|
| Structural object with known fields | `"additionalProperties": false` |
| Free-form map (`map[string]any` in Go) | `"additionalProperties": {}` |
| Free-form map with string values (`map[string]string`) | `"additionalProperties": { "type": "string" }` |
| Object being extended via `unevaluatedProperties` | Omit `additionalProperties`; use `unevaluatedProperties: false` at the parent |

### 7. Validation Before Declaring Done

- [ ] Install AJV (`npm install ajv ajv-formats js-yaml`)
- [ ] Write a validation script that finds ALL `.yaml` files in your positive set
- [ ] Run the script; schema must pass ALL examples
- [ ] If a failure occurs: diagnose whether the schema is too strict (fix schema) or the example is wrong (file inbox note to Don/Ken)
- [ ] Never loosen the schema to accept structurally invalid examples — diagnose first

### 8. Open Questions Protocol

When you encounter an ambiguous value, enum, or pattern:
1. Do NOT guess and implement silently
2. Choose the **safe conservative default** (e.g., include the observed value, flag it)
3. Write an open question to `.squad/decisions/inbox/<your>-<slug>-questions.md` with:
   - The observed behavior
   - The Go struct surface
   - What action was taken in the schema
   - The specific question for Barbara
   - What it blocks

---

## Anti-Patterns

| Anti-pattern | Why it's wrong |
|---|---|
| Generate schema from Go structs (`invopop/jsonschema`) | Go-biased output; can't serve C#/TS ports; ignores YAML-specific constraints |
| Use `additionalProperties: false` with `if/then` | Annotations from `if/then` don't propagate into `additionalProperties` in draft 2020-12; use `unevaluatedProperties` |
| Validate expression strings with a regex in the schema | Expression semantics are runtime-only; schema sees them as opaque strings |
| Skip validation against examples | Schema correctness is only proven by the positive set |
| Guess at enum values the spec doesn't explicitly list | File open question; use the observed set with a `SQ-NNN` note |

---

## References

- JSON Schema draft 2020-12 spec: https://json-schema.org/draft/2020-12
- AJV (draft 2020-12): `ajv/dist/2020.js`
- GERT runbook schema: `design/gert/schemas/runbook.v1.schema.json`
- GERT schema decision: `.squad/decisions/inbox/barbara-runbook-v1-schema-canonical-source.md`
- GERT schema open questions: `.squad/decisions/inbox/edith-runbook-schema-questions.md`
