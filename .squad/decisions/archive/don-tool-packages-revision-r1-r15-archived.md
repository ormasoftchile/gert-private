# Don — Tool Packages Revision: R1–R15 Resolution (independent, Edith locked out)

**By:** Don (Backend Developer, independent revision owner)
**Date:** 2026-08-09
**Subject:** Revision of the rejected GERT Tool Packages MVP spec/schemas/fixtures per
`.squad/decisions/inbox/barbara-tool-packages-gate-review.md` (R1–R15). Authored
independently; Edith (original author) was locked out per Cristián's directive and did
not advise or contribute. The architecture ruling
(`.squad/decisions/archive/barbara-tool-packages-architecture-ruling.md`, AR-TP-1
through AR-TP-10) is NOT reopened — every change below is a wiring/contradiction fix
within that ruling's bounds, not a re-litigation of it.

## Scope discipline

Only the artifacts implicated by Barbara's R1–R15 (plus the two explicitly-optional
R16/warnings item and non-blocking notes she raised) were touched. Files that were
already correct per Barbara's "what's right" list (governance composition table,
lexical tool-name scoping, event catalog additions, security/digest sections, resumption
contract, open-questions closure) were re-read for consistency with renamed vocabulary
but intentionally NOT modified beyond two small cross-reference additions (see R11
below) — they did not need it.

## R1–R15 resolution matrix

| # | Barbara's blocking item | Resolution | Files |
|---|---|---|---|
| R1 | No single canonical `tool/v1` action shape; map vs list, `args/output` vs `inputs/outputs` unreconciled | `actions:` is now always a list (never a map); `args:` is the single canonical parameter vocabulary for both process and substituted actions (`step.tool.args` always binds to it); `output:` (singular, process-only) vs `outputs:` (plural, substitution-only) kept as deliberately distinct keys with a normative paragraph explaining why | `sections/06-tool-runtime.tex` (Tool Definition Schema), `examples/acme-incident-tools/tools/kubectl.tool.yaml` |
| R2 | Per-action `impl:` collides in name with the pre-existing top-level mobile `impl:` | Per-action key renamed `execute:` (`execute.kind: process\|runbook`, `execute.path`); top-level mobile `impl:` (iOS/Android dispatch) untouched; new disambiguation paragraph + interaction rule (tool-level `impl:` wins over `execute.kind: runbook` on a covered platform) | `sections/06-tool-runtime.tex`, `examples/.../kubectl.tool.yaml` |
| R3 | Undefined: how substituted outputs become the caller's capture namespace | New `outputs.<name>` GCP root, valid only for `execute.kind: runbook` tool steps; new grammar production `LocalOutputs`; new normative paragraph and worked example | `grammar/gcp.ebnf` (production + Source Prefix Reference Table row), `sections/06-tool-runtime.tex` (§Caller-visible capture namespace) |
| R4 | Reference substitute didn't produce its declared output on the required path; capture expressions invalid GCP (`capture: {drain_ok: "true"}` is a literal, not a path; `value: "${drain_ok}"` is GIS syntax where a bare GCP path is required) | `drain-node.yaml` rewritten: a real `cli` step emits genuine JSON, captured, then `outputs.drained.value: "step.drain.json.drained"` (valid GCP); caller side in r23 now captures via `outputs.drained` per R3 and asserts against the PJVM-coerced string `"true"` (schema-correct, not a bool) | `examples/.../runbooks/drain-node.yaml`, `testdata/runbooks/r23-tool-package/schema.yaml` |
| R5 | No stated containment bases for manifest exports / execute paths / substitution paths / project bindings / package-map bindings / runbook-scope requires; `../runbooks/...` in the reference example looked like an escape | New "Resolution base per path kind" table naming resolution base vs. containment root per path kind explicitly, including the package-internal-substitute-runbook special case; explains `execute.path`'s base (declaring file's directory) is deliberately not the same as its containment root (package root), so `../runbooks/drain-node.yaml` is valid, not PKG-007 | `sections/06-tool-runtime.tex` (new §Resolution base per path kind, updated Containment rule), fixtures re-authored to match |
| R6 | PKG-020/PKG-021 unreachable behind closed schemas | New normative pre-schema-validation raw-document scan for `toolPackages:`/`toolRefs[].alias`, ordered before generic schema validation | `sections/03d-parse-time-enforcement.tex` |
| R7 | `dependencies` schema (`maxItems: 0`) contradicted its own description and made PKG-016 unreachable | Removed `maxItems: 0`; description now states the resolver (not the schema) raises PKG-016 for any non-empty array, preserving the ruling's MVP non-goal (no transitive deps) while keeping PKG-016 reachable | `schemas/tool-package.v1.schema.json` |
| R8 | `apiVersion` inconsistencies (`package/v1` vs `tool-package/v1`, etc.) | All example/manifest `apiVersion` values corrected to match their schema's `$id` | `sections/06-tool-runtime.tex` (manifest example), `examples/.../gert-package.yaml`, `.../kubectl.tool.yaml` |
| R9 | Tier-3 qualification/pinning identifiers schema-incompatible (reverse-DNS `package` pattern vs. MCP single-label names) | Tier-3 is addressed ONLY via a qualified `toolRefs[].name` (e.g. `mcp:server/tool`); `toolRefs[].package` is explicitly prohibited for tier-3 tools | `sections/05-extension-runtime.tex`, `schemas/runbook.v1.schema.json` (`ToolRef` description) |
| R10 | §05 collision-paragraph contradiction (precedence vs. hard tie error) | Split into two cases: cross-source shadowing (precedence applies, not a tie) vs. genuine same-source name collision (hard error) | `sections/05-extension-runtime.tex` |
| R11 | Replay behavior for substitution undefined; directly load-bearing on non-fatal `replay/packageDrift` | Substitution (`execute.kind: runbook`) replays like an `invoke` step: the substitute executes in replay mode against the caller's scenario file, its own nested steps matched individually by `argv`; distinct from, and not affected by, the separate historical-trace-event replay described in §07 (which performs no step execution at all). Ties explicitly into why package-digest drift is non-fatal in replay (AR-TP §7.6) | `sections/06-tool-runtime.tex` (new §Replay semantics), `sections/13-evidence-tracing-resumption.tex` (new bullet in existing §Replay Semantics), `testdata/runbooks/r23-tool-package/schema.yaml` (worked scenario) |
| R12 | Step-level `tool.version` vs. package version semantics unreconciled | `ToolInvocation.version` description rewritten: plan-time-evaluated, intersected with `toolRefs[].version`/`requires[].version`, deprecated in favor of `toolRefs[].version` | `schemas/runbook.v1.schema.json` |
| R13 | Lock-root no-raw-absolute-path invariant not structurally enforced | `LockedPackage.root` pattern now structurally forbids raw absolute paths (relative-POSIX-only, or `external:sha256:<64hex>`); `version` given a strict SemVer 2.0.0 pattern | `schemas/package-lock.v1.schema.json` |
| R14 | Missing errors for mutually-exclusive package/path binding and unresolved `toolRefs.name` | New `PKG-029` (package+path both present) and `PKG-030` (name resolves to no catalog entry at all, distinct from PKG-011) added at the next available codes (PKG-019 remains reserved, per the ruling's no-renumbering rule) | `sections/03d-parse-time-enforcement.tex`, `schemas/runbook.v1.schema.json` |
| R15 | No concrete unchanged-runbook real/mock acceptance scenario with valid environments + override map | Reference tool's `allowed-environments` extended to `["real", "dry-run", "replay"]`; r23 fixture description now documents all three run modes for the identical, unchanged runbook, including a worked scenario-file override map (`commands:` matched by `argv`, covering both the top-level and the nested-substitute process invocations) | `examples/.../kubectl.tool.yaml`, `testdata/runbooks/r23-tool-package/schema.yaml` |
| R16 (optional, per review) | `warnings:` missing from accepted `PackageExpected` conformance schema surface | Added `warnings` array (`^PKG-W\d{3}$` pattern), valid alongside `catalog` or `error` | `conformance/vector.schema.json` |

## Files changed (full list, this revision only)

- `design/gert/grammar/gcp.ebnf` — `LocalOutputs` production (R3/R4) + reference table row.
- `design/gert/schemas/runbook.v1.schema.json` — `ToolRef` (R9/R14), `ToolInvocation.version` (R12).
- `design/gert/schemas/tool-package.v1.schema.json` — `dependencies` (R7).
- `design/gert/schemas/package-lock.v1.schema.json` — `root`, `version` (R13).
- `design/gert/conformance/vector.schema.json` — `PackageExpected.warnings` (R16).
- `design/gert/sections/03d-parse-time-enforcement.tex` — PKG-029/030 (R14), PKG-020/021 parse-gate ordering (R6).
- `design/gert/sections/05-extension-runtime.tex` — tier-3 addressing (R9), collision split (R10).
- `design/gert/sections/06-tool-runtime.tex` — R1, R2, R3, R5, R8, R11 (largest set of edits; see matrix above).
- `design/gert/sections/13-evidence-tracing-resumption.tex` — one added bullet cross-referencing substitution replay behavior (R11 consistency only; no other change).
- `design/gert/schemas/examples/acme-incident-tools/gert-package.yaml` — verified consistent, no change needed.
- `design/gert/schemas/examples/acme-incident-tools/tools/kubectl.tool.yaml` — `execute:` rename (R2), `allowed-environments` (R15).
- `design/gert/schemas/examples/acme-incident-tools/runbooks/drain-node.yaml` — full rework (R3, R4, R5).
- `design/gert/testdata/runbooks/r23-tool-package/schema.yaml` — capture namespace, assert fix, mode-acceptance scenario (R4, R15).
- `.squad/agents/don/history.md`, `.squad/decisions.md` — this revision recorded.

No changes were made to `design/gert/sections/02-architecture.tex`, `07-runtime-events.tex`,
`08-security-and-trust.tex`, `10-open-questions.tex`, `12-governance-policy.tex`, or the
`r01`/`r05` fixture annotations — these were Edith's or the ruling-implementation's
existing, correct content per Barbara's review and needed no revision for R1–R15.

## Validation performed

- All touched JSON Schema files: `python -c "import json; json.load(...)"` — pass (valid JSON) for all 5 files.
- All touched/rewritten YAML fixtures: `python -c "import yaml; yaml.safe_load(...)"` — pass.
- `python design/gert/scripts/verify_corpus.py` — **280/280 conformance vectors validate** (baseline preserved; `vector.schema.json`'s R16 addition did not break anything).
- `python design/gert/scripts/latex.py build` (tectonic engine, `pygmentize` installed for `minted`) — **full document builds to `main.pdf` (~1 MB) with no new errors or undefined references.** The only two `LaTeX Warning: ... undefined` references (`sec:gis:portable-json`, `sec:gcp`) are pre-existing, in files this revision did not touch (`03-schema-vnext.tex`, `03b-interpolation-syntax.tex`, `03c-capture-paths.tex`), and are out of scope.
- Manual re-trace of R1–R15 against final text: all 15 confirmed resolved (see matrix). R16 (optional) resolved.

## Deviations / things flagged for Barbara's second gate pass, not fixed here (out of scope)

1. **Pre-existing `replay` terminology overload** discovered while resolving R11:
   `07-runtime-events.tex` §Determinism and Replay Semantics describes a *trace-event*
   replay (re-emits a historical event stream for adapter/UI rendering; no step
   execution) while `13-evidence-tracing-resumption.tex` §Replay Semantics describes an
   *execution-mode* replay (steps genuinely execute, I/O intercepted from a scenario
   file). Both use the same `mode: replay` vocabulary. This revision resolves R11 using
   the execution-mode model (consistent with the ruling's own §7.6 wording, "replay
   determinism is defined by the scenario file"), and adds one cross-reference bullet to
   §13 for substitution — but the underlying two-concepts-one-name tension in the
   existing (pre-Edith, pre-this-revision) corpus is NOT resolved and is a good
   candidate for its own ticket; fixing it generally is out of R1–R15's scope and would
   touch runtime-events architecture beyond tool packages.
2. **`apiVersion: runbook/v2` corpus-wide inconsistency** — confirmed still present and
   still NOT caught by `verify_corpus.py` (which only validates conformance vectors, not
   runbook/tool-package/config/lock documents against their JSON Schemas). Per Barbara's
   note #5, this is a separate, pre-existing, non-blocking ticket; left untouched.
3. Optional `assessment.md`/`README.md` companion for r23 (Barbara's non-blocking note
   #3) — not added; the extensive in-file description comments were judged sufficient
   given time/scope constraints. Can be added on request.

## Tess follow-up (deliberately NOT done here, per explicit task instruction)

- The ≥46-vector Tess-owned conformance corpus for tool packages (`tv-pkg-resolve.yaml`
  and siblings) was **not authored** in this revision. Barbara's review text ("Author
  `tv-pkg-resolve.yaml` after R1–R9 land") is guidance for a future pass, not an
  assignment to this revision, and the task instructions explicitly said not to create
  it unless the review assigns a small scenario vector — it does not. Tess should author
  the full corpus against the now-resolved R1–R15 contracts (canonical `outputs.<name>`
  capture root, `execute:` action key, `PKG-029`/`PKG-030`, the resolution-base table,
  and the replay-as-invoke substitution model) once this revision passes gate review.
