# SKILL: JSON Schema Discriminated Union Patterns (Draft 2020-12)

**Slug:** `json-schema-discriminated-union`  
**Author:** Edith  
**Date:** 2026-06-07T19:28:42-07:00  
**Applies to:** Any task authoring or updating `design/gert/schemas/*.schema.json`

---

## Problem

GERT runbook steps are a discriminated union: a `type` field selects which
payload fields are required. JSON Schema draft 2020-12 has no native `discriminator`
keyword; the correct encoding is `allOf[if/then]` with `unevaluatedProperties: false`.

---

## Pattern 1: allOf[if/then] for Step-Type Discrimination

```json
"Step": {
  "type": "object",
  "required": ["id", "type"],
  "properties": {
    "type": { "type": "string", "enum": ["cli", "tool", "…"] },
    "id":   { "type": "string" }
    // … common fields …
  },
  "allOf": [
    {
      "if":   { "required": ["type"], "properties": { "type": { "const": "cli" } } },
      "then": { "required": ["…"], "properties": { /* cli-specific fields */ } }
    },
    {
      "if":   { "required": ["type"], "properties": { "type": { "const": "extension" } } },
      "then": {
        "required": ["extension"],
        "properties": {
          "extension": {
            "type": "object",
            "required": ["name", "action"],
            "additionalProperties": false,
            "properties": {
              "name":   { "type": "string" },
              "action": { "type": "string" },
              "args":   { "type": "object", "additionalProperties": true }
            }
          }
        }
      }
    }
  ],
  "unevaluatedProperties": false
}
```

**Key rules:**
- `unevaluatedProperties: false` at the Step root rejects fields not declared in any matching `then`.
- The `if` condition MUST include `"required": ["type"]` alongside the `properties` check; without it, steps with no `type` field would silently satisfy every `if`.
- Each `then` branch only needs to list the type-specific fields; common fields live in the outer `properties`.
- Use `additionalProperties: false` on payload sub-objects (like `extension`); use `additionalProperties: true` on open key-value maps (like `args`).

---

## Pattern 2: oneOf for Dual-Form Modeling

When a construct can appear in two structurally distinct forms (e.g., `iterate`
as a step type vs. as a standalone TreeNode), use `oneOf` at the container level:

```json
"FlowNode": {
  "type": "object",
  "properties": {
    "step":     { "$ref": "#/$defs/Step" },
    "iterate":  { "$ref": "#/$defs/IterateNode" },
    "parallel": { "$ref": "#/$defs/ParallelNode" }
  },
  "additionalProperties": false,
  "oneOf": [
    { "required": ["step"] },
    { "required": ["iterate"] },
    { "required": ["parallel"] }
  ]
}
```

Both forms are independently modeled. The step-type form (`- step: { type: iterate }`)
is handled by the Step's `allOf[if/then]`. The standalone TreeNode form (`- iterate: {}`)
is handled by the FlowNode `oneOf`. No shared `$ref` is needed between them; they
share the same field *names* but not the same schema branch.

---

## Pattern 3: Closed Container, Open Leaf

For extension payloads where the container is known but the args are open:

```json
"extension": {
  "type": "object",
  "required": ["name", "action"],
  "additionalProperties": false,     ← closed: only name, action, args
  "properties": {
    "name":   { "type": "string" },
    "action": { "type": "string" },
    "args":   { "type": "object", "additionalProperties": true }  ← open leaf
  }
}
```

---

## Validation Pipeline

```bash
python -c "
import jsonschema, json, glob, yaml
from jsonschema import Draft202012Validator
with open('design/gert/schemas/runbook.v1.schema.json', encoding='utf-8') as f:
    schema = json.load(f)
validator = Draft202012Validator(schema)
for fpath in glob.glob('../gert/examples/**/*.runbook.yaml', recursive=True):
    with open(fpath, encoding='utf-8') as f:
        doc = yaml.safe_load(f)
    errors = list(validator.iter_errors(doc))
    print('PASS' if not errors else 'FAIL', fpath)
"
```

Requires: `pip install jsonschema pyyaml`  
Run from: `P:\Projects\gert-private`

---

## Pitfalls

1. **Stash-on-branch-switch conflicts**: If you edit a file on main and then
   switch to a branch where that file doesn't exist, git stash pop will conflict.
   Resolution: `git add <file>` to accept the stash version, or re-apply edits manually.
2. **`list` is not `array`**: Barbara ruled `list` is not a normative alias for `array`
   in GERT input declarations. The normative enum is `[string, number, boolean, secret, array, object]`.
3. **`integer` is not normative**: PJVM does not have a separate integer type.
   Use `number` for all numeric inputs.
