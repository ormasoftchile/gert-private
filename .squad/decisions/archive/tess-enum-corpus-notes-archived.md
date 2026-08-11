# Tess — Notes from Enum-Constrained Tool/Runbook Outputs MVP corpus (`tv-enum.yaml`)

**From:** Tess (Conformance Tester)
**To:** Edith / Barbara
**Re:** `design/gert/conformance/tv-enum.yaml`, implementing
`.squad/decisions/inbox/barbara-enum-constraint-mvp-architecture-ruling.md` (AR-ENUM-1..15)
**Status:** Corpus complete and passing; one schema gap found that made one AR-ENUM-6
sub-case untestable at its literal site; flagging per task instructions ("append
decision inbox only if actual ambiguity is found").

## Finding: `$defs.Output` in `runbook.v1.schema.json` has no `default` property

AR-ENUM-6 states the "default must be a declared member" rule "Applies at S1, S3, S4
and to a substituted action's declared output default." S4 is the runbook-level
`outputs.<name>` site. However, `runbook.v1.schema.json`'s `$defs.Output` currently
only defines `type` / `description` / `value` / `enum` — there is no `default`
property at all (unlike `$defs.Input`, which does have one). This is not one of the
already-known/ticketed D1–D4 drifts in the ruling's §16; it appears to be a
plain oversight (Input has it, Output doesn't), not an enum-semantics ambiguity.

**Impact on the corpus:** I could not write a literal `outputs.<name>.default: <non-member>`
fixture at the runbook-Output (S4) site — it isn't schema-representable today (the
document itself wouldn't be `runbook/v1`-schema-valid for reasons unrelated to enum
membership, so a harness couldn't distinguish "rejected for ENUM-006" from "rejected
because `default` isn't even a legal key here").

**What I did instead (no invented fields, per task instructions):**
- `TV-ENUM-DEFAULT-003` now tests the S2 case (a substituted tool action's own
  `outputs.<name>.default` in `tool.yaml`, which has no governing schema, C3) instead
  of S4 — this is schema-safe and is explicitly one of AR-ENUM-6's enumerated sites.
- `TV-ENUM-DEFAULT-006` is a documentation-only contract vector: it exercises the S4
  site's ordinary enum production-time check on a path that doesn't require a
  `default` literal, and its `note` records that the literal-default-at-S4 case is
  currently untestable in this corpus/schema and needs either (a) adding `default` to
  `$defs.Output`, or (b) an explicit ruling that S4 defaults are intentionally schema
  free-form like S1/S2. Tagged `untestable-in-corpus-format` / `schema-gap`.

## Non-finding (no action needed): drift D3 reference in `TV-ENUM-RUNTIME-003`

`TV-ENUM-RUNTIME-003` (provider-binding `from: myprovider.region`) is spec-legal per
`03-schema-vnext.tex` but not yet schema-legal per `$defs.Input.from`'s narrower enum —
this is the already-known, already-ticketed drift D3 from the ruling's §16. I left the
vector as originally authored (asserting the spec-normative ENUM-008 behavior) and
added an explanatory `note` cross-referencing D3, rather than rewriting the fixture or
re-litigating an already-settled drift. No decision needed from this note; included
here only for completeness alongside the Output.default finding above.

## No other ambiguities found

Everything else in the ruling (AR-ENUM-1..15) mapped cleanly onto the existing vector
shapes with no invented fields. `Input.redact` / `GovernanceConfig.sensitive_inputs`
absence from the schema (vs. `08-security-and-trust.tex`'s prose example) was worked
around by anchoring `TV-ENUM-TRACE-002`/`003` at the S1 tool-arg `redact: true` site,
which is the one enum/redaction combination that is genuinely unconstrained by schema
today (C3) — this did not require a workaround note since S1 is explicitly one of
AR-ENUM-6/7/8's enumerated sites, not a substitution.
