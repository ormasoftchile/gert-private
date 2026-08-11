# Don — Enum-Constrained Tool and Runbook Outputs MVP: Implementation Report

**From:** Don (Backend Developer)
**To:** Barbara (Architect), squad record
**Date:** 2026-08-10
**Re:** Implementation of the ratified Enum-Constrained Tool and Runbook Outputs MVP
(`barbara-enum-constraint-mvp-architecture-ruling.md`, gated by
`barbara-enum-spec-final-gate-approval.md`) in `gert` (runtime repo)

## Summary

Implemented AR-ENUM-1..15 end to end in the runtime, at exactly the four ratified
declaration sites (tool action `args`/`outputs`, runbook `inputs`/`outputs`), enforcing
ENUM-001..009 and ENUM-W001, extending PKG-013 for substitution enum-set equality, and
carrying enum metadata through `ValidatedPlan` with C1 redaction. No apiVersion or
grammar (GXL/GIS/GCP) changes. No commits made in either repository; no branch
switches. All four dirty-tree files you flagged as protected
(`examples/simple-health-check/simple-health-check.runbook.yaml`,
`internal/tool/native.go`, `internal/tool/native_test.go`,
`examples/simple-health-check/examples.code-workspace`) were verified untouched before
and after this pass via `git diff`.

Full detail (files changed, exact behavior per code, test list) is recorded in
`.squad/agents/don/history.md` under "Enum-Constrained Tool and Runbook Outputs MVP
(2026-08-10)". This entry summarizes the two items that require your attention as new
gates, per your own stated escalation process.

## Finding 1 — `TV-ENUM-DECL-006` conflicts with this repo's actual YAML resolver

This repo pins `gopkg.in/yaml.v3` v3.0.1, whose scalar-tag resolver follows the YAML
1.2 core schema: bare `yes`/`no` resolve to `!!str`, not `!!bool`. `tv-enum.yaml`'s
`TV-ENUM-DECL-006` vector expects `enum: [yes, no]` to fail as ENUM-002, which is only
true under a YAML-1.1-style resolver (where `yes`/`no` coerce to booleans, tripping the
"every item must be a plain string-resolving scalar" rule).

Verified empirically against this library:
- `enum: [yes, no]` → two valid, distinct, plain string scalars. **Does NOT fail.**
- `enum: [true, false]`, `enum: [1.0, 2.0]`, `enum: [~, null]` → correctly rejected as
  ENUM-002 (these resolve to `!!bool`/`!!float`/`!!null` under yaml.v3 too).
- `enum: [!!int "1", "2"]` (explicit non-str tag) → correctly rejected as ENUM-002.

I implemented against the actual, verified library behavior rather than force a fake
failure to match the vector's YAML-1.1 assumption. This is a genuine
vector/implementation-library conflict, not a coding error — requesting your ruling on
whether to (a) amend `TV-ENUM-DECL-006` to an unambiguous non-boolean-coercing
counter-example (e.g. an explicit `!!bool` tag, which already fails correctly), or
(b) something else.

## Finding 2 — No root (non-substituted) runbook output-materialization path exists

Independent of enum work: `pkg/run/run.go` (the real `gert run` entry point) never
evaluates a directly-invoked runbook's own top-level `outputs:` block anywhere. The
only live output-production code in the whole engine is
`internal/executor/tool.go`'s `executeSubstitution`, which evaluates a *substituted*
action's runbook's `Outputs`. ENUM-009 (output production check) is fully implemented
and verified at that one real site (new CLI integration test
`cmd/gert/substitution_enum_integration_test.go` proves it end to end, including that
the CLI's `--output=json` error envelope surfaces `ENUM-009` without echoing the enum
member list). But a root runbook's own `outputs.<name>.enum` contract has no runtime
moment to be checked against, since nothing ever materializes it. This is a pre-existing
engine gap outside this MVP's authorized scope (closing it would mean inventing new
root-output-materialization engine behavior, not "wiring enum checks"). Flagging as a
candidate for a future, separately-scoped ticket.

## Other implementation notes (not requiring your ruling, informational)

- C1 redaction (enum member lists redacted in `ValidatedPlan`/traces/CLI for sensitive
  declarations) was implemented against the only redaction primitive this schema
  actually has today: `governance.redact[].pattern` (a value-scrubbing regex), matched
  against the declaration's own name as a best-effort proxy. Neither a per-arg
  `redact: true` field nor a `governance.sensitive_inputs` list exists in the schema; if
  either is wanted as a first-class concept, that's a schema addition beyond this MVP's
  scope.
- `ExecutionPlan.Governance` was previously always `nil` on the real run path (a latent,
  pre-existing wiring gap, unrelated to enum) — populated it via the existing
  `internal/governance.BuildPolicy` so the enum-redaction lookup (and the already-present
  but previously dead `internal/engine/engine.go:625` redaction-pattern hook) has real
  data. Purely additive; full `go test ./...` shows no regressions.
- ENUM-MOCK (package mock vs. real binding enum-set equality) was correctly left as a
  conformance-vector-only concern per your C2 ruling — no runtime mock-vs-real
  comparison mechanism was built.

## Validation

- `go build ./...`: clean.
- `go vet ./...`: only the same 2 pre-existing, unrelated `internal/serve` issues,
  untouched by this pass.
- `go test ./...`: full repo green, no regressions (including a self-caught and fixed
  `pkg/errkit` `ClassForCode` prefix-ordering bug introduced and immediately caught by
  its own existing `TestDeclaredClassesCovered` test).
- Targeted new tests across parser (declaration well-formedness), planner (ENUM-006/007
  + metadata carriage/redaction), executor (ENUM-008 arg binding, ENUM-009 output
  production), pkgsubst (PKG-013 enum-set equality), pkg/run (ENUM-008 caller input),
  and a live CLI integration test (ENUM-009 + JSON error envelope + redaction spot-check)
  — all passing.

## Remaining, honestly-scoped-out gaps

- Root-runbook output materialization (Finding 2) — no engine hook exists to attach
  ENUM-009 to for that specific case.
- No dedicated ENUM-MOCK-shaped conformance test was added this pass (correct per C2
  that it's conformance-only, but no vector-shaped test was written given the effort
  budget); the existing `TestRun_PackageMap_RealVsMockBinding` pattern in `cmd/gert`
  is the right shape for a follow-up.
- Trace-event-level enum metadata emission: the metadata now exists on `ValidatedPlan`
  (redaction-safe, once per run) but no explicit trace event was added to surface it in
  `trace.jsonl` this pass.
- The full 58-vector `tv-enum.yaml` corpus was not mechanized as a single data-driven
  harness; targeted hand-written tests cover representative cases per category rather
  than every vector individually.
