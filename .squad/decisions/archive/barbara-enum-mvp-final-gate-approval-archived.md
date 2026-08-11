# Final Implementation Gate — Enum-Constrained Tool and Runbook Outputs MVP: **APPROVED**

**From:** Barbara (Lead / Architect, final gate)
**Date:** 2026-08-10T17:45-07:00
**Requested by:** Cristián Ormazábal Ortega
**Under review:** Ken's independent revision of R1–R5 in `C:\One\OpenSource\gert` (working tree,
uncommitted), `.squad/decisions/inbox/ken-enum-mvp-implementation-revision.md`,
Tess's `tess-enum-decl-006-correction.md`, and Edith's AR-ENUM-3(3) prose correction, against
`barbara-enum-mvp-implementation-gate.md` (R1–R5) and
`barbara-enum-constraint-mvp-architecture-ruling.md` (AR-ENUM-1..15, C1/C2/C3).

**Method — verified directly, not on report.** Read the current runtime diff (39 modified,
43 new paths) at `pkg/schema/enum.go`, `cmd/gert/run.go`, `internal/serve/rpc.go` +
`interactions.go`, `internal/planner/enumplan.go`, `internal/executor/tool.go`,
`internal/engine/engine.go`, `internal/replay/executor.go` + `engine.go`,
`internal/adapter/wire.go`, `internal/conformance/enum_harness.go` +
`enum_conformance_test.go`. Ran `go build ./...` (clean) and `go test ./...`
(**61 packages ok, 0 FAIL**). Ran the R2 harness myself
(`go test ./internal/conformance/... -run TestEnumConformance -v -count=1`) →
**58 vectors: 48 pass, 10 named skips, 0 fail**, reproduced independently. Executed **four live
CLI probes** against scratch runbooks built outside both product trees and deleted afterwards.
**No product artifact in either repository was modified by this review**; `git status` in both
repos is byte-identical to its pre-review state (runtime: 82 entries before and after).

**VERDICT: APPROVED.** R1–R5 are all substantively resolved. No blocker remains.

---

## 1. R1–R5 verified

### R1 — ENUM-008 on live caller-binding paths: **RESOLVED, proven by execution.**
`schema.CheckCallerInputBindings` is called at all three live sites, before defaults are seeded so
only genuine caller bindings are judged: `cmd/gert/run.go:507` (`applyInputDefaults`, the real
`gert run` entry point), and `internal/serve/rpc.go:800` (`loadPlanAndSeed`, reached by **both** the
JSON-RPC handler and `internal/serve/interactions.go:77`). My probe — the exact case that failed the
previous gate:

```
> gert run enum-input.runbook.yaml --var env_name=not-a-member
ENUM-008: input "env_name" value "not-a-member" is not a declared enum member    EXIT=2
> gert run enum-input.runbook.yaml --var env_name=staging
✓ show ✓ end  complete: status=completed steps=2                                 EXIT=0
```

The unguarded hole is closed at the product boundary, not merely in `pkg/run`. Materialised tool-arg
bindings (post-GIS-interpolation, including `--var`-sourced values) are separately enforced by
`internal/executor.CheckArgEnums` before dispatch; corpus `TV-ENUM-MOCK-004` and `TV-ENUM-TRACE-003`
exercise exactly that and pass.

### R2 — conformance harness: **RESOLVED, and it is the artifact I asked for.**
`internal/conformance/enum_harness.go` builds the real `gert` CLI, materialises each vector's
`variables` map into an isolated workspace, and runs it as a user would; the three catalog-shaped
`ENUM-SUBST` vectors go through the identical `pkgcatalog.Build` path the CLI itself uses.
`TestLoadEnumSuite` pins the corpus at 58; `TestEnumConformance` **fails any skip lacking an explicit
reason** and logs the per-run report. The vendored `enumdata/tv-enum.yaml` is **byte-identical
(SHA-256 match) to `design/gert/conformance/tv-enum.yaml`**, i.e. it carries Tess's corrected
`TV-ENUM-DECL-006` and nothing else; `enumdata/README.md` documents the re-sync rule and the freeze.
The two semantics I named as untested at the last gate are now covered by *passing* vectors:
NFD-candidate-vs-NFC-member (`TV-ENUM-UNICODE-001`) and NFC-collision/duplicate handling
(`TV-ENUM-DECL-012`/`-015`). `ENUM-MOCK` and `ENUM-TRACE` now execute.

### R3 — trace carriage: **RESOLVED, verified in a real trace file.**
`internal/engine/engine.go:339` adds `enum_constraints` to the existing `plan.validated` payload via
`enumConstraintsTracePayload` (redacted declarations emit `"<redacted>"` + `member_count`; JSON map
key order is deterministic). From my probe's trace (8 events):

```
plan.validated  →  "enum_constraints":{"inputs.env_name":{"member_count":2,"members":["Prod","prod"]}}
catalog/frozen, run/started, step/started ×2, step/completed ×2, run/completed  →  key absent
```

Emitted **once per run**, and the other half of AR-ENUM-10 — absence from step/tool events — holds in
practice, not only in a unit test. `--output=json` unchanged, as required.

### R4 — ENUM-W001 reaches a human: **RESOLVED, probed.**
`cmd/gert/run.go:183` prints every `ParsedRunbook.Warnings` entry to stderr and continues:

```
gert: warning: inputs.env_name: [ENUM-W001] enum members "Prod" and "prod" are case-only-distinct
✓ show ✓ end  complete: status=completed steps=2    EXIT=0
```

Warning surfaced; run not failed. AR-ENUM-12 satisfied.

### R5 — replay does not bypass enum validation: **RESOLVED at the live replay path.**
`ReplayExecutor.executeTool` calls `checkToolCallEnums` **before consulting any recorded fixture**;
it renders args through the same evaluator and delegates to the *same exported*
`internalexecutor.CheckArgEnums` the real path uses. `internal/replay/engine.go:81` wires
`NewReplayExecutorRegistryWithEnumChecks` with `plan.Tools` — the live `ReplayFromTrace` path. A
scenario can no longer launder a value the real path would reject.

One residual, and Ken handled it the way I asked: `internal/adapter/wire.go:137`'s `mode: "replay"`
branch still constructs the plain registry, because at wire time no plan exists to check against. I
verified the reachability claim myself — **nothing in non-test code ever assigns
`adapter.Options.ScenarioFile`**, and that branch hard-errors when it is empty, so it is dead in the
product today. It is documented in code at the site with the correct remedy. Recorded as ticket
**T-ENUM-REPLAY-WIRE** (wire-time replay construction must use
`NewReplayExecutorRegistryWithEnumChecks` if it ever becomes reachable), not a blocker.

---

## 2. The 48 pass / 10 skip corpus result — each skip adjudicated

I required that no skip be an evasion of an MVP behaviour. I checked all ten against the fixtures
themselves. **None is an evasion.** Every one is either a ticketed non-existent runtime site or a
demonstrable corpus-authoring defect, and in each of the latter cases the vector's *stated intent* is
already covered by a different, passing vector or by direct probe.

| Vector | Skip class | My adjudication |
|---|---|---|
| RUNTIME-001/002/003 | No site — `Input.From` | **Upheld.** I re-grepped: `.From` is read in exactly one place in non-test runtime code (`pkg/pkgsubst` rejecting non-`context` sourcing). `from: env/prompt/<provider>` does not exist in this runtime. Ticket **T-ENUM-FROM-SOURCING**. |
| MOCK-005 | No site — same `from: prompt` gap | **Upheld.** The fixture's binding moment is a prompt answer, which this runtime never performs; there is nothing for replay to intercept. R5's actual contract (tool-arg binding in replay) is covered by code + `internal/replay/enum_r5_test.go`. Not an evasion of R5. |
| RUNTIME-006 | Root-output S4 | **Upheld** — the limitation I knowingly accepted at the previous gate. **T-ENUM-ROOT-OUTPUTS**. |
| RUNTIME-007 | Root-output S4 | **Upheld.** Its expected `GCP-TYPE-001` sits behind the same missing S4 materialisation site. Same ticket. |
| UNICODE-005 | Corpus defect | **Upheld, verified in the fixture:** `default: graceful` is declared alongside `enum: ["á","force"]`. AR-ENUM-6 is unconditional, so ENUM-006 is *mandatory* here and the vector's `expected: success` contradicts the rule it exists to test. Its stated intent (combining-acute candidate matches precomposed member) is covered by **UNICODE-001, which passes**. |
| PLAN-003 | Methodology | **Upheld.** `expected.value: ${strategy}` asserts a *plan-time unevaluated literal*, which a run-to-completion harness cannot observe; the fixture never binds `strategy`, so the run cannot complete regardless. Its intent (no constant folding of `${...}` at plan time) is positively proven by **MOCK-004 and TRACE-003**, which pass plan time with interpolated args and fail only at runtime. Not a design-only hole. |
| PLAN-005 | Corpus defect (`yes`/`no`) | **Upheld — and I verified the underlying behaviour directly rather than accept the skip.** The fixture cannot fire ENUM-002 under YAML 1.2, exactly as my own §2a ruling established. So I probed the behaviour the vector actually exists to pin — lazy-include materialisation validating the child's enum: a `expand: lazy` parent including a child with `enum: [true, false]` produces `[ENUM-002] inputs.env_name.enum.0/1: got boolean, want string`, exit 2. **The MVP behaviour is present and enforced; only the fixture is defective.** |
| RUNTIME-004 | Corpus defect | **Upheld, verified in the fixture:** substitute declares `["graceful","force","aggressive"]`, action arg declares `["graceful","force"]` — unequal sets, which AR-ENUM-8 makes PKG-013 *unconditionally*, ahead of any runtime binding. The vector's own note ("not a PKG-013 vector") contradicts the ratified rule. Identical shape passes as **SUBST-004**; the ENUM-008-at-binding intent is covered by **MOCK-004** and by my R1 probe. |

Ken did not edit the frozen corpus, correctly, and flagged all five defects for Tess instead. That is
the right behaviour.

---

## 3. Other required confirmations

- **PKG-013 no-variance matching:** unchanged from the ratified implementation and now covered end to
  end — `SUBST-003` (one side only), `SUBST-004` (unequal), `SUBST-005` (narrower), `SUBST-006`
  (wider) all raise PKG-013; `SUBST-001` (identical), `SUBST-002` (both absent), `SUBST-007`
  (reordered) all pass. No subset/superset tolerance, no new error code.
- **YAML 1.2 corrections:** Tess's `TV-ENUM-DECL-006` amendment is in place (`enum: [true, false]`,
  same id/category/`ENUM-002` expectation, count still 58, `ENUM-DECL` still 14) and the vendored copy
  matches it byte for byte. Edith's prose fix is in place in **both** rendering sites —
  `03-schema-vnext.tex` §441-452 and `03d-parse-time-enforcement.tex:502` now state that unquoted
  `yes`/`no` resolve to strings under the 1.2 core schema and that the real trap is
  `enum: [true, false]`. The normative clause is unchanged, as ruled.
- **Warning consumers:** S1/S2 via `internal/tool/scan.go` (stderr), S3/S4 via `cmd/gert/run.go`
  (stderr) — verified live.
- **Substitution captures / outputs:** ENUM-009 remains at the one real S4 site
  (`executeSubstitution`), correctly ordered coerce-then-enum, with the offending value withheld from
  `result.Output`. Ken's capture-after-failure gating (`result.Status != StepStatusFailed`, at both
  the executor and engine sites) is a genuine bug fix: it stopped an unrelated `GCP-RESOLVE-002` from
  masking the real ENUM-009 and, at the engine site, from aborting the run. Ratified.
- **C1 message hygiene:** `CheckArgEnums` and the ENUM-009 site still name the declaration only —
  no member list, no rejected value. `CheckCallerInputBindings` echoes the caller's **own supplied
  value** (`--var`/RPC input) but never the member list. That is not a value oracle (the caller
  already knows what they typed) and `enum` on `type: secret` is ENUM-001, so no secret can reach it.
  Accepted; recorded so it is not later mistaken for a regression of the stronger no-echo property.
- **Protected-file integrity:** `examples/simple-health-check/simple-health-check.runbook.yaml`,
  `internal/tool/native.go`, `internal/tool/native_test.go` (all mtime 2026-08-09) and
  `examples/simple-health-check/examples.code-workspace` (2026-08-01) all predate Ken's session and
  are untouched. No commit, stage, revert, branch or stash in either repository.
- **Collateral fixes** (`imports:` alias resolution for `include.runbook`, the missing `expand`
  property in `schemas/runbook.schema.json`, the GCP output-value resolution fix, the dropped `Enum`
  in `schemaToolDefFromRuntime`, the B1-vs-plan validation-ordering aggregation): all were *required*
  to run the corpus faithfully, all are correct, all are covered by the green full suite. Ratified as
  in-scope consequences of R2. The dropped-`Enum` bug in particular was defeating ENUM-006/007 for
  every `requires:`-resolved tool — finding it is exactly why I demanded a real-CLI harness.

---

## 4. Disposition

- **VERDICT: APPROVED.** The enum runtime implementation is **complete for the MVP**: all four
  declaration sites (AR-ENUM-1), ENUM-001..007 at plan time, ENUM-008 at every caller-binding and
  tool-arg-binding site that exists in this runtime (CLI `--var`, RPC/API inputs, materialised tool
  args, replay), ENUM-009 at the one real output-production site, PKG-013 no-variance, AR-ENUM-10
  trace carriage, AR-ENUM-12 warning delivery. 48/58 vectors execute green against the real CLI with
  zero failures and zero silent drops.
- **True remaining limitations, all ticketed, none blocking:**
  - **T-ENUM-ROOT-OUTPUTS** — a non-substituted root runbook never evaluates its own top-level
    `outputs:`; S4 production (and its ENUM-009/type checks) has no runtime site. Engine scope.
  - **T-ENUM-FROM-SOURCING** — `Input.From` (`from: env|prompt|<provider>.<field>`) is read by no
    runtime path; those ENUM-008 sites do not exist. Pre-existing, unrelated to enum.
  - **T-ENUM-SENSITIVE-DECL** — no first-class per-declaration sensitivity marker; `EnumMeta.Redacted`
    remains a best-effort name-vs-`governance.redact` proxy and MUST NOT be documented as a guarantee.
  - **T-ENUM-REPLAY-WIRE** (new) — `internal/adapter/wire.go`'s replay branch builds an
    enum-unchecked registry; currently unreachable (`ScenarioFile` is never assigned in product code)
    and documented at the site.
  - **Tool-arg runtime type enforcement** does not exist, so `CheckArgEnums` skips non-string values
    rather than misreporting a type error. Pre-existing, honest, recorded.
- **Tess** owns four further corpus amendments under the DECL-006 precedent, and only these:
  `TV-ENUM-UNICODE-005` (default not a member), `TV-ENUM-PLAN-005` (`yes`/`no` → a scalar that
  actually fails under YAML 1.2, preserving the lazy-include intent I verified works),
  `TV-ENUM-RUNTIME-004` (make caller and substitute sets equal so the vector tests binding, not
  PKG-013), and `TV-ENUM-PLAN-003` (either re-express as a plan-only assertion or bind `strategy`).
  Each amendment must keep id, category and expectation intent; count stays 58. The corpus remains
  otherwise frozen. Re-run the harness after; I expect the skip count to fall to five.
- **Ken** is released. His revision found and fixed five real runtime defects the previous
  hand-written suite could not have surfaced; the conformance harness is now the cheap gate I wanted.
- **No architecture is reopened.** AR-ENUM-1..15 and C1/C2/C3 stand as ratified, with the single
  §2a self-correction now propagated to corpus, spec prose and runtime consistently.
