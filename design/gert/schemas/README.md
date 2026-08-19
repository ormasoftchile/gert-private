# GERT Schemas — Canonical Design Artifacts

**Location:** `design/gert/schemas/`  
**Owner:** Edith (Spec Editor) — day-to-day authoring; Barbara (Architect) — review gate.  
**Added:** 2026-06-07T19:14:39-07:00 per Barbara's ruling in `decisions/inbox/barbara-runbook-v1-schema-canonical-source.md`.

---

## What Lives Here and Why

This directory is the canonical home for **normative JSON Schemas** that describe the GERT document formats. Schemas here are language-neutral design artifacts — they serve Go, C#, TypeScript, and any future runtime equally. They are **not** generated from Go structs; they are hand-authored from the GERT specification sections and grammar files.

The architectural decision to place schemas here (and not in `gert/pkg/schema/` or in individual consumer repos) is recorded in full at `.squad/decisions/inbox/barbara-runbook-v1-schema-canonical-source.md`.

---

## Schema Inventory

| File | Version | Validates | Status |
|---|---|---|---|
| `runbook.v1.schema.json` | runbook/v1 | `*.runbook.yaml` runbook documents | ✅ Active |
| `tool-package.v1.schema.json` | tool-package/v1 | `gert-package.yaml` tool package manifests | ✅ Active |
| `project-config.v1.schema.json` | config/v1 | `.gert/config.yaml` project configuration | ✅ Active |
| `package-lock.v1.schema.json` | package-lock/v1 | `.gert/packages.lock.json` generated lock files | ✅ Active |

The three tool-package schemas were added per Barbara's ruling
`.squad/decisions/archive/barbara-tool-packages-architecture-ruling.md` (AR-TP-3, ratified
2026-08-09) and are normatively described in
`design/gert/sections/06-tool-runtime.tex` §Tool Packages. `runbook.v1.schema.json` gained
the top-level `requires` field and the `ToolRef.package`/`ToolRef.version` fields from the same
ruling — it remains schema version `v1` (additive-only change, AR-TP-10 §5).

**Not in this directory:**  
`design/gert/conformance/vector.schema.json` — validates `tv-*.yaml` conformance test vectors (Tess's artifact). It is a sibling, not a runbook schema. Do not confuse the two.

---

## `runbook.v1.schema.json`

### Scope

Validates the **structure and types** of any `*.runbook.yaml` document whose `apiVersion` is `runbook/v1`. Covers:

- Required and optional root-level fields (`apiVersion`, `id`, `name`, `kind`, `flow`, etc.)
- All step types: `cli`, `tool`, `include`, `choice`, `decision`, `collector`, `branch`, `approve`, `assert`, `compensate`, `wait_for_event`, `end`, `noop`, `display`, `extension`
- Flow-level node shapes: `step`, `iterate`, `parallel`
- Capture blocks, `capture_defaults`, `when` conditions, governance policy, region manifests
- Pattern constraint on step and flow-node `id` fields: `^[a-z][a-z0-9_-]*$`

### `$id` and `$schema`

```
$schema: https://json-schema.org/draft/2020-12/schema
$id:     https://gert.dev/schemas/runbook/v1
```

The `$id` URL is a stable placeholder. Resolution via HTTP is not currently wired; consumers reference the file by path or via vendor copy. Submission to [SchemaStore.org](https://www.schemastore.org/) is planned post-v1.0 stabilization.

### Boundary Statement

The schema is a **structural gatekeeper only**. It validates what a YAML parser can see at parse time.

| **In scope — schema validates** | **Out of scope — runtime-only enforcement** |
|---|---|
| Required/optional fields per step type | GXL expression semantics (type correctness, short-circuit) |
| Field value types (string, number, boolean, object, array) | GIS interpolation resolution (path existence, variable binding) |
| `enum` values for `type`, `kind`, `apiVersion`, `source` | GCP capture path validity (requires runtime context map) |
| Pattern constraints on step IDs (`^[a-z][a-z0-9_-]*$`) | Cross-step reference integrity (step ID exists in flow) |
| `capture` / `capture_defaults` key shape | Expression evaluation results |
| Structural nesting (flow → steps → sub-steps) | Governance policy enforcement (approval policies) |
| `$schema` and `apiVersion` presence | Timeout/duration parsing beyond string format |

String fields that hold GIS templates (e.g., `when`, `title`, `prompt`, content values) are **opaque strings** in this schema. Their description fields note "Evaluated as GIS at runtime; structural validation only at parse time." Expression syntax inside `${...}`, GCP path syntax, and condition truthiness are runtime-only concerns.

---

## Consumer Rules

### Vendor-with-CI-Drift-Check Pattern (DRIFT-DETECTION-001)

Consumers do **not** reference this file via a live URL at runtime. Instead:

1. **Vendor a copy** of the schema at a specific git SHA or tagged release from `gert-private`.
2. **Run a CI drift-check** that fetches the current canonical schema and diffs it against the vendored copy. Fail the build on any diff.

This is the same pattern used for conformance vectors (DRIFT-DETECTION-001 in `.squad/decisions.md`).

| Consumer | Vendored path | Mechanism |
|---|---|---|
| `gert-vscode` | `schemas/runbook.v1.schema.json` | `contributes.yamlValidation` in `package.json`; `scripts/sync-schema.sh` + CI check |
| `gert-tui` | `schemas/runbook.v1.schema.json` | Same sync script pattern |
| `gert` (Go runtime) | CI fixture validation only | CI validates `testdata/fixtures/*.runbook.yaml` against this schema; NOT a runtime import |
| Future C#/TS ports | Consumer-defined path | Fetch schema; optionally generate types with `quicktype` |
| SchemaStore.org | (future submission) | After v1.0 stabilization |

---

## Generation Policy

> **This schema is HAND-AUTHORED. Do NOT auto-generate it from Go structs.**

Rationale (from Barbara's ruling):

- `invopop/jsonschema` produces a Go-biased artifact (Go type names, Go `omitempty` semantics, Go struct embedding). It cannot express YAML-specific constraints or conditional sub-schemas per step type.
- The schema must serve C#, TypeScript, and future ports equally. A Go-generated schema privileges one runtime.
- The GERT spec sections (`.tex`) and grammar files (`.ebnf`) are the normative source. Edith authors the schema from those, using the Go structs as a reference implementation checklist.

When the Go structs change, the **spec sections are updated first** (in `gert-private`), then Edith updates this schema in the same PR, then Don/Ken update the Go structs.

---

## Maintenance — When to Update This Schema

| Trigger | Action |
|---|---|
| Grammar change in `design/gert/grammar/*.ebnf` | Review for new structural elements; Barbara gates the update |
| New step type added to spec | Add `if/then` arm in Step.$defs, update type `enum` |
| New top-level runbook field added | Add to root `properties`; check `required` list |
| Field renamed or removed | Update `properties`; bump consumers' drift check |
| New `kind`, `source`, or other enum value | Add to the relevant `enum` array |
| New value-constraint keyword | Structural array shape only (e.g. `minItems`, `uniqueItems`, per-item `pattern`); NFC normalisation, Unicode control/bidi rejection, and type-linkage rules are semantic/runtime, not schema-enforced — see the `enum` keyword's own split (`design/gert/sections/03-schema-vnext.tex` §Input/§Output Declarations, `design/gert/sections/03d-parse-time-enforcement.tex` §Enum Constraint Error Codes) as the worked example |
| New spec section (03d, 04a, etc.) | Review for structural implications; Edith assesses |

**Review gate:** Any PR touching `design/gert/schemas/` MUST have Barbara's approval.

**Cross-references:**  
- Grammar files that trigger schema review: `gxl.ebnf`, `gis.ebnf`, `gcp.ebnf`  
- Spec sections that define schema-visible structure: `sections/03-schema-vnext.tex` (§Step, §Runbook), `sections/05-*.tex` through `sections/08-*.tex` (step types)

---

## Open Questions

See `design/gert/schemas/` open questions in `.squad/decisions/inbox/edith-runbook-schema-questions.md`.

