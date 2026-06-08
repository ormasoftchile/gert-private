# Open Questions — Runbook v1 Schema

**Author:** Edith (Spec Editor)  
**Date:** 2026-06-07T19:14:39-07:00  
**Status:** PENDING — awaiting Barbara arbitration  
**Context:** Arose during authoring of `design/gert/schemas/runbook.v1.schema.json`.

---

## SQ-001 — `name` field on Step: legacy or active?

**Observed:** `examples/nav-test/nav-test.runbook.yaml` uses a `name:` field on steps:
```yaml
- step:
    id: step_alpha
    name: Alpha Step
    type: cli
    run: echo "ALPHA_UNIQUE_OUTPUT_XYZ"
```

**Go struct:** `pkg/schema/step.go` `Step` struct has `Title string yaml:"title"` and `Subtitle string yaml:"subtitle"` but **no** `name` field.

**Schema action taken:** `name` has been included in the Step common properties as an optional string, with a note that it is a legacy field and `title` is preferred.

**Question for Barbara:** Is `name:` on a step a supported alias for `title:`, a deprecated field, or an unintentional omission from the Go struct? Should the schema:
- (a) Keep it as an optional legacy alias (current choice)
- (b) Deprecate it with a `deprecated: true` annotation
- (c) Remove it entirely (would fail the nav-test example)

**Blocks:** Nothing immediately — (a) is a safe default that lets all examples pass.

---

## SQ-002 — `extension` step type: payload undefined

**Observed:** `pkg/schema/step.go` declares `StepTypeExtension StepType = "extension"` as "v1 retained", but the `Step` struct has no corresponding `ExtensionSpec *ExtensionSpec yaml:",inline"` field. No examples of extension steps exist in `gert/examples/`.

**Schema action taken:** `extension` is included in the `type` enum with an empty `then` block (only common fields allowed). With `unevaluatedProperties: false`, any extension-specific payload field would currently be rejected.

**Question for Barbara:** What fields does an `extension` step carry? Does it use a fixed payload, or is the payload extension-defined (open object)? Should the schema:
- (a) Keep `extension` in the enum with an empty then (current choice) — too strict if extension steps have fields
- (b) Allow any additional properties for extension steps (`"then": {"unevaluatedProperties": true}`) — permissive but unspecified
- (c) Define a minimal extension step payload (e.g., `type` + some `config` object)
- (d) Remove `extension` from the enum entirely until it is specified

**Blocks:** gert-vscode hover/completion for extension steps.

---

## SQ-003 — Input `type` enum: `list` alias and complete normative set

**Observed:** `examples/incident-triage/resource-exhaustion.runbook.yaml` declares an input with `type: list`:
```yaml
inputs:
  instances:
    type: list
    required: false
    description: Affected instances to walk through
    default: [ "i-1", "i-2" ]
```

`list` is not a PJVM type name (PJVM defines: null, boolean, number, string, array, object per `03b §sec:gis:portable-json`). The Go `Input.Type` field is a plain `string` with no enum constraint.

**Schema action taken:** The Input `type` enum has been set to:
`["string", "number", "integer", "boolean", "array", "object", "list"]`

`list` and `array` are both accepted as the schema conservatively admits both until the spec clarifies.

**Question for Barbara:** 
1. Is `list` a normative type alias for `array` in the GERT input declaration vocabulary?
2. Is `integer` a separate type from `number` for input validation purposes?
3. Are `object` and `array` (or `list`) valid input types in `runbook/v1`? If so, what does the Go runtime do when it receives a YAML list as an input value?
4. What is the complete normative set of input type values? Should the spec codify this enum?

**Blocks:** Normative enum in spec section §Input Declarations.

---

## SQ-004 — Step ID pattern: `^[a-z][a-z0-9_-]*$` — runbook `id` excluded?

**Observed:** Barbara's boundary statement in `decisions/inbox/barbara-runbook-v1-schema-canonical-source.md` lists "Pattern constraints on IDs (`^[a-z][a-z0-9_-]*$`)" as in scope for schema validation.

**Also observed:** Runbook-level IDs in examples include dots:
- `id: incident-triage.network` (`network.runbook.yaml`)
- `id: incident-triage.resource-exhaustion` (`resource-exhaustion.runbook.yaml`)

These would fail `^[a-z][a-z0-9_-]*$` due to the `.` character.

**Schema action taken:** The pattern `^[a-z][a-z0-9_-]*$` has been applied only to **step** `id` fields and **iterate/parallel node** `id` fields. The runbook root `id` field is constrained only by `minLength: 1`.

**Question for Barbara:** Should the ID pattern differ between:
- Runbook `id` (appears to allow dots, hyphens)
- Step/iterate/parallel `id` (pattern `^[a-z][a-z0-9_-]*$` applies)

If so, what is the normative runbook `id` pattern? Does the spec section on runbook structure define this?

**Blocks:** ID validation strictness for gert-vscode.

---

## SQ-005 — `iterate` and `parallel` as step `type` values

**Observed:** `pkg/schema/step.go` includes:
```go
StepTypeIterate  StepType = "iterate"
StepTypeParallel StepType = "parallel"
```

in the `StepType` const block. However, all examples use `iterate:` and `parallel:` as **FlowNode-level** keys, not as `step: {type: iterate}`. No example shows a step with `type: iterate` or `type: parallel`.

**Schema action taken:** `iterate` and `parallel` are **NOT** in the step `type` enum. They are only modelled as FlowNode-level siblings of `step:`.

**Question for Barbara:** Are `iterate` and `parallel` valid step `type` values for any use case, or are they purely internal Go runtime type tags? Should they be added to the schema's step `type` enum?

**Blocks:** Would affect schema if some use cases embed iterate/parallel inside step wrappers.

---

*Edith — 2026-06-07T19:14:39-07:00*
