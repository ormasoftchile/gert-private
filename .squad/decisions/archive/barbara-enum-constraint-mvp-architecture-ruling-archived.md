# Architecture Ruling — `enum` String Constraint MVP (AR-ENUM-1..15)

**From:** Barbara — Lead / Architect
**Date:** 2026-08-10T13:25:10-07:00
**Requested by:** Cristián Ormazábal Ortega
**Status:** RATIFIED — binding. No open questions. Deferrals below are ratified non-goals, not TBDs.
**Repo scope:** `gert-private` is DESIGN ONLY. Nothing here authorises a change to the Go runtime.
**Supersedes/extends:** `.squad/decisions/archive/barbara-tool-packages-architecture-ruling.md` (AR-TP-5, PKG-013),
`.squad/decisions.md` § Runbook v1 Schema Canonical Source.

---

## 0. Verdict

**APPROVED with scope corrections.** The proposal as stated is architecturally sound and lands
inside an MVP boundary I can defend. Three corrections are binding conditions of approval:

| # | Correction | Why |
|---|---|---|
| C1 | `enum` is **forbidden** on `type: secret` and its member list is **redacted** on any declaration carrying `redact: true` / `sensitive_inputs`. | An enum on a redacted field is a value oracle. "Expected one of [tokenA, tokenB]" in a trace or error message defeats the redaction it sits next to. This is an audit-trail product; that hole is not acceptable. |
| C2 | "Exact enum match across **package mocks**" is discharged as a **conformance obligation**, not a runtime check. | Only one binding is loaded per run. There is no in-run comparand for a mock-vs-real signature diff, and inventing lock fields to create one is scope creep. Making this a required vector class is enforceable and honest; a runtime check would be theatre. |
| C3 | **No** `tool.v1.schema.json` is created in this MVP. | `06-tool-runtime.tex` §Tool Definition Schema remains the sole normative source for `.tool.yaml`. Authoring a full tool schema to host one keyword is a workstream, not a rider. Deferred ticket, §9. |

---

## 1. AR-ENUM-1 — Scope: exactly four declaration sites

`enum` is a **per-declaration value constraint keyword**. It is valid at exactly four sites and
nowhere else:

| Site | Artifact | Governing section |
|---|---|---|
| S1 | tool action `args.<name>` | `06-tool-runtime.tex` §Tool Definition Schema |
| S2 | tool action `outputs.<name>` (substituted actions) | `06-tool-runtime.tex` §Action Substitution §Output contract |
| S3 | runbook `inputs.<name>` | `03-schema-vnext.tex` §Input Declarations |
| S4 | runbook `outputs.<name>` | `03-schema-vnext.tex` §Output Declarations |

**Not valid** on: `capture:` / `capture_defaults:` entries, `vars:`, step-level fields, `collector`
fields, `toolRefs`, `requires`, package manifests, extension manifests, provider `fields:`.
Presence of `enum` at any other key is `additionalProperties: false` / unknown-key rejection under
existing rules — no new code.

**Collector `options:` is explicitly NOT unified with `enum` in this MVP.** `options:` carries
`{value, label}` pairs and drives UI rendering; `enum` is values-only and drives contract
validation. They overlap conceptually and differ operationally. Unifying them is a separate
proposal with its own UX blast radius (AR-ENUM-15).

Provider `fields:` in `03-schema-vnext.tex` §Provider Definition Schema already shows
`enum: [P1, P2, P3, P4]` in an example. That example is **informative prose about a provider's own
schema**, not a GERT-validated construct. Edith MUST add one sentence saying so, or the corpus
will read it as normative by accident.

## 2. AR-ENUM-2 — Type restriction

`enum` is valid **only** when the declaration resolves to `type: string`.

- `enum` on `number`, `integer`, `boolean`, `array`, `object` → **ENUM-001**.
- `enum` on `secret` → **ENUM-001** (C1). A secret's value domain must not be enumerable.
- `enum` where `type` is absent: `03-schema-vnext.tex` §Input Declarations makes `string` the
  default type, so this is legal at S3. At S1/S2/S4 `type` is required; absence is the existing
  schema error, not an enum error.

`enum` is **not a type**. It is a constraint evaluated **after** type resolution. There is no
"enum type", no enum-valued PJVM member, no change to the PJVM type set
(`03b-interpolation-syntax.tex` §Portable JSON Value Model is untouched).

## 3. AR-ENUM-3 — Well-formedness of the member list

Checked at parse/plan time, unconditionally, on every declaration in the include and package
closure — reachable or not (existing rule: "validity is not conditional on execution path").

1. `enum` MUST be a YAML sequence. A mapping, scalar, or null → **ENUM-002**.
2. `minItems: 1`. An empty sequence → **ENUM-002**. A zero-member domain makes the declaration
   unsatisfiable; it is always an authoring mistake, never a way to say "no values allowed".
3. Every item MUST be a **YAML scalar that resolves to a string under the YAML 1.2 core schema**.
   A plain scalar resolving to boolean/int/float/null (`yes` under a 1.1-ish resolver, `true`,
   `1.0`, `~`, `null`) → **ENUM-002**. **No coercion.** Authors quote. This is the single most
   likely real-world trap (`enum: [yes, no]`) and it fails loudly.
4. Nested sequences/mappings and explicit YAML tags on items → **ENUM-002**.
5. No item may be the empty string, whitespace-only, or carry leading/trailing whitespace →
   **ENUM-003**. Rationale: `""` is indistinguishable from "unset" across `from: env`,
   `from: prompt`, and CLI sourcing; ` prod` vs `prod` is an invisible distinction in an artifact
   humans review during an incident.
6. Members MUST be pairwise distinct after NFC normalisation → **ENUM-004**.
7. Members MUST be valid UTF-8, already in NFC, and free of Unicode control (U+0000–U+001F,
   U+007F) and bidi/format controls (U+200E, U+200F, U+202A–U+202E, U+2066–U+2069) →
   **ENUM-005**.

## 4. AR-ENUM-4 — Unicode, normalisation, equality

- **Comparison is codepoint-wise equality on NFC forms.** No case folding, no locale collation,
  no confusable/skeleton analysis, no trimming at comparison time.
- **Asymmetric normalisation, deliberately:** declared members MUST already be NFC (ENUM-005 if
  not); candidate values from outside the file (env, prompt, provider, tool stdout, caller
  bindings) ARE NFC-normalised before comparison. The author controls their own file and a silent
  rewrite there hides the diff; the author does not control what a Kubernetes API returns.
- **Case-only-distinct members are legal.** `["Prod", "prod"]` validates and matches
  case-sensitively. I am **not** importing the PKG-018 case-only-collision rule: that rule exists
  because filesystems are case-insensitive, and enum members are pure data with no such substrate.
  A validator MUST emit **ENUM-W001** for case-only-distinct member pairs — legal, suspicious,
  worth a line of output.
- Confusable-script detection is a ratified non-goal (AR-ENUM-15).

## 5. AR-ENUM-5 — Two orderings, stated separately

This is the seam people get wrong, so both are normative and neither is optional:

| Ordering | Definition | Used for |
|---|---|---|
| **Declared order** | The sequence order as authored, preserved verbatim | UI/adapter rendering, `--output=json` metadata, diagnostic "permitted values" lists, plan carriage |
| **Canonical order** | Ascending codepoint order of the NFC forms | Any textual/byte comparison of two enums, deterministic diagnostics, cross-runtime parity assertions |

**Enum identity for contract comparison is SET equality, order-insensitive.** Reordering members
is **not** a contract break and MUST NOT raise PKG-013. Order is presentation.

**No new digest machinery.** Package and export digests are digests of file bytes
(`06-tool-runtime.tex` §File digest); an enum edit already changes them and already produces
PKG-009 on drift. Adding an enum to the package lock is a ratified non-goal.

## 6. AR-ENUM-6 — Defaults, optionality, absence

- `default` present with `enum` present → `default` MUST be a member (exact, NFC) → **ENUM-006**
  at plan time. Applies at S1, S3, S4 and to a substituted action's declared output default.
- **`enum` constrains values, not presence.** An enum does NOT imply `required: true`. An optional,
  defaultless, unsupplied declaration is *absent*; absence is not an enum violation.
- **`null` is never a member** and cannot be added as one (ENUM-002 rejects the `null` scalar). An
  explicit null on an optional declaration means absent, not "member violation".
- Existing `required`/`default` interaction rules (`03-schema-vnext.tex`: `default` valid only when
  `required: false`) are unchanged and are evaluated first.

## 7. AR-ENUM-7 — Plan-time vs runtime, stated exhaustively

**Plan time** (static, zero execution — the `ValidatedPlan` gate of `03d-parse-time-enforcement.tex`):

- P1. All of §3 (well-formedness) and §4 (Unicode) on every declaration in the closure.
- P2. §6 default-in-enum → ENUM-006.
- P3. Substitution signature enum set-equality → PKG-013 (§8).
- P4. Any **statically-known literal** bound to an enum-constrained declaration: a `step.tool.args`
  value that is a constant with no GIS interpolation, and a runbook input `default`.
  Non-member → **ENUM-007**.

**Explicitly NOT plan time:** any value containing GIS interpolation. There is **no** partial
evaluation, prefix analysis, or constant folding of `${...}` against an enum in the MVP. A
runtime-only enum failure is an accepted, designed outcome — not a defect. (Contrast PKG-014,
where runtime-only detection *is* a design defect, because governance widening must never be
discoverable late. Value-domain violations are ordinary input errors; policy widening is an
escalation. The two get different treatment on purpose.)

**Runtime** — enum membership is checked at the **moment of binding**, in this exact order:

1. Existing type resolution / GIS coercion to string (`03b` §Type Coercion to String).
2. Existing type check.
3. NFC normalisation of the candidate.
4. Enum membership.

Consequences, all normative:

- A non-string value bound to an enum-constrained string declaration is reported as the **type**
  error, not ENUM-008. **Single-cause reporting**: an author gets one true root cause, not two.
- `${count}` coercing to `"2"` and matching a member `"2"` is a **pass**. Enum operates on the
  post-coercion PJVM string. This does not reopen GXL's no-implicit-coercion rule
  (`03a` §874/§901) — GIS coercion-to-string is an existing, separately ratified mechanism.
- On failure: **ENUM-008**, the binding does not occur, the value never enters the execution
  context, is never passed to a tool, and is never written to the trace as an accepted value.
  **No coercion, no nearest-match auto-correction, no fallback to `default`.**
- Sites: tool action arg binding (S1), runbook input materialisation (S3), incl. `from: prompt`
  (the operator is re-prompted per the existing collector retry contract where one applies;
  otherwise the run fails), `from: env`, `from: <provider>.<field>`, and caller/context bindings.

**Outputs** (S2, S4) are checked at production time — when the substituted action's or runbook's
declared output value resolves at completion, **before** the value crosses the caller boundary,
enters `outputs.<name>` GCP capture, is written to the trace, or reaches `--output=json`.
Non-member → **ENUM-009**. An off-enum output MUST NOT be emitted with a warning; the contract
that says "this output is one of these" is worthless if it can be violated silently.

**Replay mode MUST NOT bypass enum validation.** Values answered from a scenario file are bound
values and are validated identically. This parallels the ratified "replay does not bypass
governance" rule. A scenario that records an off-enum value fails — that is the scenario's bug.

**Dry-run** validates everything plan-time plus any binding it actually performs; it does not
execute side-effecting steps, so runtime checks fire only for bindings that occur.

## 8. AR-ENUM-8 — Substitution and package contract compatibility

**PKG-013 is extended, not replaced.** For every `args.<name>` and `outputs.<name>` shared between
a tool action and its `execute.kind: runbook` substitute:

- Both sides declare an enum whose member **sets** are equal (§5) → OK.
- Neither side declares an enum → OK.
- One side declares an enum and the other does not → **PKG-013**.
- Both declare enums with unequal sets → **PKG-013**.

**No variance. Not subset, not superset.** Rationale, and it is symmetric:

- A **narrower** substitute fails at runtime on values the caller's declared contract permits —
  converting a static contract into a runtime landmine, which is exactly what AR-TP-5's
  "substitution is invisible to the caller" invariant forbids.
- A **wider** substitute accepts values the published contract forbids — silent contract erosion,
  and the caller's own plan-time ENUM-007 check would reject a literal the implementation would
  happily have taken. Two disagreeing truths about the same action.

I keep this inside PKG-013 rather than minting an ENUM-01x code because it *is* a signature
mismatch, existing tooling and the `PKG-SUBST` vector category already key on it, and splitting one
concept across two catalogs is how error taxonomies rot. Edith extends PKG-013's catalog
description to name enum set-equality explicitly.

**Package mocks (C2).** A "mock package" is not a spec concept — it is an ordinary package bound
via `.gert/config.yaml` `requires[].path` or a `--package-map` override. Ratified consequences:

1. The runbook is **unchanged** between real and mock binding; the same enum-constrained usage is
   validated against whichever tool definition is bound. Enum validation is **binding-independent**
   and MUST NOT be weakened, skipped, or short-circuited under an override or in any non-`real`
   environment.
2. A mock that declares a different enum set than the real package is an **authoring error**
   detectable only by running both bindings. It is therefore enforced by a **required conformance
   vector class** (`ENUM-MOCK`, §10), not by a runtime comparison. I will not invent a lock field
   or a per-run signature registry to fake in-run detectability.
3. When a package lock entry is present, any enum edit inside the package already changes the
   export digest and is already **PKG-009**. That is the whole integrity story for enums. **No new
   lock schema field.**

## 9. AR-ENUM-9 — Includes and runbook output semantics

- **Enums are declaration-local and lexically scoped.** They are never inherited, merged,
  intersected, or unioned across an include boundary. This mirrors the already-ratified lexical
  tool-name scoping across includes.
- `include` shares the caller's variable space. If the parent and an included runbook both declare
  the same variable name with different enums, each declaration constrains its **own** binding
  site. This is **not** a conflict and raises no error. Attempting intersection semantics would
  make a benign refactor (adding an input declaration to a child) able to invalidate a parent —
  action at a distance across files, which this design has consistently refused.
- **Lazy includes:** enum well-formedness (§3/§4) of a lazily materialised file is validated when
  that file is materialised, consistent with existing lazy-include semantics. This creates **no**
  late-binding hole analogous to PKG-017, because no catalog, package set, or tier is involved —
  an enum is inert file-local data with no global namespace.
- **Runbook output semantics (S4):** validated at runbook completion against the resolved value of
  `outputs.<name>.value` (the GCP path form in `runbook.v1.schema.json`), before the value is
  exposed to a caller, to `outputs.<name>` capture at a substituting call site, to the trace, or to
  `--output=json`.
- **PKG-027 interaction:** a substituted action's declared output must already be produced on every
  terminal path or carry a default. If it carries a default, §6 applies (default must be a member).
  The two checks are independent and both fire.

## 10. AR-ENUM-10 — Metadata carriage, visibility, redaction

**`enum` is contract metadata, not a value.** It travels with the *declaration*, never with the
*datum*.

- **ValidatedPlan** (`03d` §The `ValidatedPlan` Type Contract) MUST carry, in **declared order**,
  the enum of every enum-constrained declaration it validated. A stored plan must be
  self-describing; otherwise re-validation on resume can only re-derive it by re-reading files
  that may have drifted.
- **Trace events:** enum metadata is emitted **once per run**, in the existing plan/validation
  record required by `03d` §"Validation result recorded in trace". It is **NOT** repeated in
  `tool/invoked`, `tool/completed`, `step/started`, or `step/completed` payloads. Per-event
  payloads carry the **value** only. Repeating a static member list on every invocation inflates
  every trace file for zero information gain.
- **`--output=json`** (`16-observability-diagnostics.tex` §JSON Output Mode): the run summary
  object is unchanged — no enum metadata. Enum failures surface as ENUM-007/008/009 error objects
  in the existing `03d` §Error Message Format shape.
- **Adapters/UI** MAY render the declared-order member list for input prompts and select surfaces.
  This is the one place the declared order is user-visible and is why §5 preserves it.

**Redaction (C1), normative:**

| Declaration | `enum` allowed? | Member list in trace/JSON/errors |
|---|---|---|
| `type: secret` (S3) | **No** — ENUM-001 | n/a |
| tool arg with `redact: true` (S1) | Yes | **Redacted.** Emit `"enum": "<redacted>"`; diagnostics report the declaration name and member **count** only. |
| listed in `governance.sensitive_inputs` (S1/S3) | Yes | **Redacted**, as above. |
| matched by a `governance.redact` regex rule | Yes | **Redacted**, as above. |
| ordinary string | Yes | Visible |

An ENUM-008 message for a redacted declaration MUST NOT enumerate permitted values and MUST NOT
echo the rejected value. Values themselves are redacted per existing rules regardless of enum.
This is C1 and it is not negotiable: the diagnostic quality loss on secret-adjacent fields is the
correct trade against handing an attacker the value domain.

## 11. AR-ENUM-11 — Compatibility and versioning

- **Unconstrained strings are unaffected.** Absence of `enum` means "any PJVM string". Absence MUST
  NOT be modelled as an empty enum, a wildcard enum, or a null enum anywhere in a schema, plan,
  trace, or runtime type.
- **Schema versions do not bump.** `runbook/v1`, `tool/v1`, `tool-package/v1`, `config/v1`,
  `package-lock/v1` all stay. Adding an optional keyword is additive per
  `03-schema-vnext.tex` §compatibility policy ("Adding new optional fields... are compatible").
- **Author-facing, however, adding an `enum` to a previously unconstrained string IS a breaking
  change to that contract** — it is the limiting case of the already-documented
  "narrowing an existing enum (removing a valid value)". Edith MUST state this explicitly at
  `03-schema-vnext.tex:105`, because the current text only names removal from an existing enum and
  a reader will otherwise conclude that introducing one is safe.
- Old runbooks and old `.tool.yaml` files, containing no `enum`, validate and execute unchanged.
  There is no migration, no shim, no flag.

## 12. AR-ENUM-12 — Error taxonomy

**One new family, `ENUM-0NN`, in `03d-parse-time-enforcement.tex` §Error Code Catalog.** Not split
across PKG-* and PLAN-*: the constraint is a single concept applied at four sites, and forcing an
implementer to reassemble it from two catalogs guarantees divergent implementations.

| Code | Raised by | Condition | Remedy |
|---|---|---|---|
| `ENUM-001` | Parser/Schema | `enum` on a non-string declaration, incl. `secret` | Remove `enum`, or declare `type: string` |
| `ENUM-002` | Parser/Schema | `enum` not a sequence, empty, or contains a non-string-resolving scalar / collection / tagged item | Quote members; use a non-empty list of strings |
| `ENUM-003` | Parser/Schema | Member is empty, whitespace-only, or has leading/trailing whitespace | Remove the member or trim it |
| `ENUM-004` | Parser/Schema | Duplicate members after NFC normalisation | Remove the duplicate |
| `ENUM-005` | Parser/Schema | Member is invalid UTF-8, not NFC, or contains control/bidi-format characters | Normalise to NFC; remove control characters |
| `ENUM-006` | Planner | `default` is not a member | Use a declared member as the default |
| `ENUM-007` | Planner | Statically-known literal bound value is not a member (plan time) | Use a declared member |
| `ENUM-008` | Runtime | Materialised bound value is not a member | Supply a declared member |
| `ENUM-009` | Runtime | Declared output value is not a member at production time | Fix the producing step or widen the contract deliberately |

**Warning** (non-fatal, MUST appear in the validation report, MUST NOT fail the run):

| Code | Condition |
|---|---|
| `ENUM-W001` | Enum contains case-only-distinct members (`Prod` / `prod`) |

`PKG-013`'s description is **extended** to name enum set-equality (§8). No `ENUM-010`. No renumbering
of any existing code. `PKG-019` remains reserved and unused.

All ENUM-* messages use the existing `03d` §Error Message Format envelope (`code`, `message`,
`file`, `line`, `column`, `remedy`), subject to §10 redaction.

## 13. AR-ENUM-13 — Relation to existing type, coercion, and expression rules

- **Grammar files are untouched.** `gxl.ebnf`, `gis.ebnf`, `gcp.ebnf` gain no production, no
  keyword, no error class. `enum` is a schema-level declaration constraint, invisible to all three
  grammars. Any proposal to teach a grammar about enums is rejected in advance for this MVP.
- **GXL is unaffected.** `x == "prod"` compares strings. Enum membership is not exposed to GXL,
  there is no `enum.contains()` stdlib addition, and enum-constrained variables have no special
  type in comparisons. GXL's no-implicit-coercion rule (`03a`) stands untouched.
- **GIS is unaffected.** `${var}` coerces per `03b` §Type Coercion to String; enum checking happens
  on the resulting string at the binding site, downstream of GIS.
- **GCP is unaffected.** `outputs.<name>` resolution (`gcp.ebnf` §3.4a) is unchanged; enum
  validation on S2/S4 happens at production time, before the value is available to a capture path.
- **Collector field validation** (`02-architecture.tex` §Collector Field Validation Contract) is
  unchanged. Its `options`-membership check is a separate, pre-existing mechanism. It MUST NOT be
  re-specified in terms of `enum`, and no ENUM-* code fires for a collector field.

## 14. AR-ENUM-14 — Exact normative artifacts to modify

**Spec sections** (`design/gert/sections/`) — Edith:

| File | Change |
|---|---|
| `03-schema-vnext.tex` §Input Declarations (`subsec:inputs`, ~L349) | Add `enum:` to the YAML shape; normative rules §2–§7; ENUM-* cross-refs |
| `03-schema-vnext.tex` §Output Declarations (`subsec:outputs`, ~L416) | Add `enum:`; production-time validation (§7, §9); **and** correct the stale `from: captures.<name>` prose to the `value:` GCP-path form the schema and §06 actually use (see §16) |
| `03-schema-vnext.tex` ~L105 compatibility policy | State that *introducing* an enum on a previously unconstrained string is narrowing (§11) |
| `03-schema-vnext.tex` §Provider Definition Schema (~L3476) | One sentence: the `enum: [P1..P4]` in the provider example is the provider's own schema, not a GERT-validated construct (§1) |
| `06-tool-runtime.tex` §Tool Definition Schema (~L222–330) | `enum` in the `args:` shape and the `args:`/`outputs:` vocabulary paragraph |
| `06-tool-runtime.tex` §Action Substitution §Input contract / §Output contract (~L908–950) | Enum set-equality under PKG-013 (§8); no variance |
| `06-tool-runtime.tex` §Action Substitution §Replay semantics (~L989) | Replay does not bypass enum validation (§7) |
| `06-tool-runtime.tex` §Tool Packages / §Non-Goals | Mocks are ordinary bindings; no lock/digest change (§8) |
| `03d-parse-time-enforcement.tex` §Error Code Catalog (~L204–442) | New ENUM-* table; ENUM-W001 in the warnings list; extend PKG-013's description |
| `03d-parse-time-enforcement.tex` §`ValidatedPlan` Type Contract (~L150) | Plan carries enum metadata in declared order (§10) |
| `07-runtime-events.tex` §tool/invoked, §tool/completed | One sentence: payloads carry values, never enum metadata (§10) |
| `13-evidence-tracing-resumption.tex` §Event Type Catalogue + §Replay Semantics | Enum metadata emitted once per run in the validation record; replay no-bypass |
| `08-security-and-trust.tex` §Secret Resolution and Redaction (~L296–340) | The redacted-enum oracle rule (C1, §10) |
| `16-observability-diagnostics.tex` §JSON Output Mode (~L910) | Run summary unchanged; ENUM-* errors use the `03d` error envelope |

**Schemas** (`design/gert/schemas/`) — Edith, Barbara review gate:

| File | Change |
|---|---|
| `runbook.v1.schema.json` `$defs.Input` | Add `enum`: `{type: array, minItems: 1, uniqueItems: true, items: {type: string, minLength: 1, pattern: "^\\S(.*\\S)?$"}}`. Structural layer only — NFC, control-character, and type-linkage rules are runtime/semantic (ENUM-001/004/005), matching the schema's ratified structural-gatekeeper boundary |
| `runbook.v1.schema.json` `$defs.Output` | Same `enum` keyword |
| `schemas/README.md` | Maintenance-trigger row: "New value-constraint keyword"; note the structural/semantic split for `enum` |
| `tool-package.v1.schema.json`, `project-config.v1.schema.json`, `package-lock.v1.schema.json` | **NO CHANGE** — ratified |
| `design/gert/schemas/tool.v1.schema.json` | **NOT CREATED** (C3) |

**Grammar** (`design/gert/grammar/*.ebnf`): **NO CHANGE** — ratified (§13).

**Conformance** (`design/gert/conformance/`) — Tess: see §10/§15.

**Fixtures** (`design/gert/testdata/`, `design/gert/schemas/examples/`) — Edith/Tess:
`examples/acme-incident-tools/tools/kubectl.tool.yaml` gains an enum-constrained arg on
`drain-node`; `testdata/runbooks/r23-tool-package/schema.yaml` exercises it end-to-end through a
substituted action. One reference package, one reference runbook — not a corpus-wide sweep.

## 15. AR-ENUM-15 — Ratified non-goals

Deferred with prejudice for this MVP. These are **decided**, not open:

1. `enum` on `number`, `integer`, `boolean`, `array`, `object` — the stated deferral, confirmed.
2. Object/structural validation of any kind (nested schemas, `properties`, `items`).
3. Unification of `enum` with collector `options:` / `options_from:`.
4. Member labels, descriptions, i18n, deprecation markers on individual members.
5. Variance (subset/superset) compatibility across substitution or package bindings.
6. Enum inheritance, extension, aliasing, or reference (`$ref`-style member reuse).
7. Case-insensitive or locale-aware matching; Unicode confusable/skeleton detection.
8. Partial/constant-fold evaluation of GIS-interpolated values against an enum at plan time.
9. Enum members recorded in `packages.lock.json` or in any digest beyond file bytes.
10. Normative "did you mean" auto-correction. Diagnostics **MAY** suggest a nearest member (subject
    to §10 redaction); a runtime **MUST NOT** substitute it.
11. `enum` on captures, vars, step fields, `toolRefs`, or extension manifests.
12. A `tool.v1.schema.json` (C3).

## 16. Adjacent pre-existing drift found (tickets, not blockers)

Found while reading the objects Edith is about to edit. Disposition is ratified:

| # | Drift | Disposition |
|---|---|---|
| D1 | `03-schema-vnext.tex` §Output Declarations documents `from: captures.<name>`, but `runbook.v1.schema.json` `$defs.Output` uses `value:` (a GCP path), and `06-tool-runtime.tex` §Output contract depends on `outputs.<name>.value`. The schema and §06 are right; the §03 prose is stale. | **Fix in the same PR** (§14). The enum output rules reference this exact object; leaving two contradictory shapes in the file being edited guarantees a wrong enum implementation. |
| D2 | `runbook.v1.schema.json` `$defs.Input` omits `pattern` and `example`, which `03-schema-vnext.tex` §Input Declarations declares. With `additionalProperties: false`, a spec-conformant runbook is schema-invalid. | **Separate ticket.** Real bug, adjacent object, unrelated cause. Do not let it ride along on the enum PR. |
| D3 | `runbook.v1.schema.json` `$defs.Input.from` enumerates `[prompt, env, context]`; §03 documents `env.<NAME>`, `file.<path>`, `<provider>.<field>`, `var.<name>`. | **Separate ticket**, same reasoning as D2. Note the irony: the fix is itself an enum-vs-pattern decision. |
| D4 | `03-schema-vnext.tex:184` — a `\S\ref` leaking verbatim inside a `minted` comment (carried from Tool Packages gate review 3). | Unchanged: typographic nit, fix opportunistically. |

---

## Handoff

**→ Edith (Spec Editor).** You own §14 rows 1–14 plus the two schema edits. Sequence: (1) `03d`
error catalog first — every other section cross-references ENUM-00x and I do not want forward
references to codes that do not exist yet; (2) `03-schema-vnext.tex` §Input/§Output including
drift D1; (3) `06-tool-runtime.tex` (definition shape → substitution contract → replay); (4) the
metadata/redaction trio (`07`, `13`, `08`) and `16`. Two things I will specifically re-review at
the gate: the §10 redaction table rendered as normative MUST language rather than a note, and the
§8 no-variance justification stated in both directions (narrower *and* wider both break) — a
one-directional argument will be read as "subset is probably fine". Do not create
`tool.v1.schema.json`. Do not touch `*.ebnf`. Do not bump any `apiVersion`. If you find a
semantic I did not resolve, stop and write `.squad/decisions/inbox/edith-*.md`; there should not
be one.

**→ Tess (Conformance Tester).** New corpus file `design/gert/conformance/tv-enum.yaml`, validated
by the existing `vector.schema.json`, following the `tv-pkg-resolve.yaml` shape (`vectors[]` with
`id`, `category`, `description`, `input`, `variables` as a virtual file map, `expected`, `tags`).
**Minimum 54 vectors** across eight categories, every ENUM-00x and ENUM-W001 with at least one
negative vector:

| Category | Min | Must cover |
|---|---|---|
| `ENUM-DECL` | 12 | ENUM-001 (incl. `secret`), ENUM-002 (`[yes, no]` unquoted, empty list, mapping, nested seq, tagged item), ENUM-003, ENUM-004, ENUM-W001 |
| `ENUM-UNICODE` | 8 | NFC vs NFD member (ENUM-005); NFD *candidate* matching an NFC member (must PASS — §4 asymmetry); invalid UTF-8; U+202E bidi member; control char; combining-mark equality; case-only pair; codepoint-order canonicalisation |
| `ENUM-DEFAULT` | 5 | ENUM-006 at S1/S3/S4; valid default; optional-absent-is-not-a-violation |
| `ENUM-PLAN` | 7 | ENUM-007 on a literal `step.tool.args`; interpolated value NOT rejected at plan time; unreachable step still validated; lazy-include materialisation timing |
| `ENUM-RUNTIME` | 8 | ENUM-008 from env/prompt/provider/caller binding; ENUM-009 on runbook and substituted output; type-error-wins-over-enum-error precedence; `${count}` → `"2"` matching |
| `ENUM-SUBST` | 6 | PKG-013 on enum-present/absent, unequal sets, narrower, wider; reordered members must PASS; both-absent passes |
| `ENUM-MOCK` | 5 | **The C2 obligation.** Byte-identical runbook under real and mock binding: equal enum sets → both pass; mock with a divergent set → the value that passed under real binding fails under mock with the *same* ENUM-008, proving the check is binding-independent; plus one replay-mode vector proving replay does not bypass (§7) |
| `ENUM-TRACE` | 3 | Enum metadata present once in the validation record and absent from `tool/invoked`/`tool/completed`; redacted member list for a `redact: true` arg; ENUM-008 message for a redacted declaration carries a count, not the values |

Blocked until Edith lands `03d` (you need the code numbers). Determinism rules as always: no
timestamps, no host paths, no locale dependence — and note that this corpus is the one place
where locale dependence would be a genuine hazard, so assert on codepoints, not on rendered text.

**Gate.** I re-review §8 (substitution/mock), §10 (metadata + redaction), and the ENUM-*/PKG-013
catalog text before this workstream closes. Same three-gate pattern as Tool Packages: the
highest-yield question at gate 1 is not "is the new text right" but "which untouched text did the
new text just falsify" — for this ruling the prime suspects are the collector validation contract,
the provider-schema example, and `03-schema-vnext.tex`'s compatibility policy.
