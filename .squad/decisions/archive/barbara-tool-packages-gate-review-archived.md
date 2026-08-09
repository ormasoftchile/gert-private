# Architecture Gate Review — GERT Tool Packages MVP (Edith's implementation of AR-TP-1..10)

**From:** Barbara (Lead / Architect)
**Date:** 2026-08-09T14:47-07:00
**Requested by:** Cristián Ormazábal Ortega
**Reviews:** working-tree diff against `.squad/decisions/archive/barbara-tool-packages-architecture-ruling.md`
**Verdict:** **REJECTED** — revision required before the phase gate.
**Reviser designated:** **Don** (Backend Dev / runtime semantics). **Edith is locked out** of this
revision cycle under the reviewer lockout protocol; she authored the artifacts under review.
**Blocked-on-Don:** Tess's `tv-pkg-resolve.yaml` (≥46 vectors) — do not author until R1–R9 land, since
several corrections change the observable contract the vectors encode.

---

## 0. What is right

Credit where due; these are accepted as-authored and must not be re-litigated:

- Two-phase (C/B) model, five tiers, tier-3 no-bare-name rule, and the uniform
  "higher tier wins, ties are errors" direction — correctly propagated into §05 and §06, with the
  old "first match wins" / "later entries win" text actually deleted, not merely contradicted.
- Determinism of enumeration (§06 sorted NFC POSIX paths, declaration order, MCP name sort).
- SemVer grammar: EBNF, AND-only conjunction, 0.x caret narrowing, prerelease and build-metadata
  rules, and "checking not solving" are reproduced faithfully; the `PackageMeta.version` regex is a
  correct strict SemVer 2.0.0 pattern; a leading `v` is rejected, not stripped.
- Governance composition table in §12 is verbatim-correct (OR / union / union / **intersection** /
  union / rules-order / capability closure) with the plan-time-only widening check.
- Digest construction (raw bytes, `sha256sum`-style two-space blob, closure limited to the API
  surface, catalog digest over `qualifiedName + digest + tier`).
- Resume refusal + `--allow-package-drift` + `governance/packageDriftAccepted`; replay drift
  non-fatal via `replay/packageDrift`. Event catalog table updated with REQUIRED/OPTIONAL rows.
- Lexical tool-name scoping in §02, including the explicit asymmetry rationale vs variable scope,
  and plan-time package analysis of `expand: lazy` includes with `PKG-017`.
- Full `PKG-001..028` + `PLAN-010` + `PKG-W001..003` catalog registered in §03d with the reserved
  `PKG-019` hole preserved; `location.file` extended to `<packageRoot>/<relpath>`.
- `toolPackages:` correctly absent from every schema; `alias` removed; `source`/`actions` retained as
  numbered deprecations D-001/D-002 with a removal target; fixtures r01/r05 annotated.
- Corpus verifier passes (280/280) and the `schema.json` → `vector.schema.json` path fix in
  `verify_corpus.py` is a correct incidental repair.

The rejection below is about wiring and contradiction, not about direction.

---

## 1. Required corrections (blocking)

### R1 — `.tool.yaml` action shape is self-contradictory (highest severity)

`06-tool-runtime.tex` §Tool Definition Schema (unchanged, ~line 275) declares `actions:` as a
**mapping** keyed by action name, with `args:` (typed argument map) and `output: {format}`. The new
§Action Substitution and the reference `schemas/examples/acme-incident-tools/tools/kubectl.tool.yaml`
use `actions:` as a **list** of `- name:` entries with `inputs:` / `outputs:`. Both are normative,
both are in the same chapter, and there is no `.tool.yaml` JSON Schema to arbitrate.

Required: pick one shape and make §Tool Definition Schema, §Action Substitution, and the reference
example agree. Explicitly define the relationship between `args:`/`output:` (process actions) and
`inputs:`/`outputs:` (substitution contract) — either they are the same fields under one vocabulary,
or the mapping between `step.tool.args` and each is stated normatively. Today `step.tool.args` binds
to `args` for a process action and to `inputs` for a substituted action with nothing saying so.

### R2 — `impl:` is overloaded with two unrelated meanings

Top-level `impl:` already exists in `.tool.yaml` as the per-platform mobile dispatch block
(`impl.ios`, `impl.android`, `transport: native-sdk`, `handler:`). The substitution contract
introduces a per-action `impl: {kind, path}`. Same key, two meanings, no cross-reference, and the
interaction (a substituted action on a platform-dispatched tool) is undefined.

Required: disambiguate. Either rename the action-level key, or normatively scope both (`impl:` at
tool level = platform dispatch; `actions[].impl:` = implementation kind) **and** state what happens
when both are present.

### R3 — Substituted-action outputs have no defined capture path

§Action Substitution says only declared `outputs:` cross the boundary, but never says under which
GCP path the caller sees them. The r23 fixture guesses `capture: node_drained: json.drained` — a
process/stdout-shaped path for an action that never produces stdout. This is exactly the class of
under-specification the ruling exists to remove.

Required: specify the capture namespace for substituted-action outputs normatively, and make r23 use
it. State whether it is identical to the process-action capture surface or distinct.

### R4 — The reference substitute example does not satisfy its own contract

`schemas/examples/acme-incident-tools/runbooks/drain-node.yaml` declares
`outputs.drained: {type: boolean, value: "${drain_ok}"}` and produces `drain_ok` via
`capture: {drain_ok: "true"}` on an `end` step. Per `runbook.v1.schema.json`, `capture` values are
**GCP capture paths** ("stdout", "exit_code", "local.varname"), not literals — so `drain_ok` is a
capture from a non-existent path `true`, and the declared output is produced on **no** terminal path.
The flagship demonstration of substitution is itself `PKG-027` (and `PKG-013` on the boolean/string
type mismatch) as written. Note it validates structurally against `runbook.v1.schema.json` — this is
a semantic break the schema cannot catch, which is precisely why the example matters.

Required: rewrite the substitute so the declared output is genuinely produced on every terminal path
by a real mechanism, and make r23's assertion type-correct (it currently compares a boolean output
against the string `"true"`).

### R5 — Path containment base directories are unstated, and the reference example escapes

§Containment says `realpath(join(root, p))` MUST be inside `realpath(root)`, while §Declaration says
`impl.path` resolves relative to the declaring `.tool.yaml`'s directory. The reference tool uses
`path: ../runbooks/drain-node.yaml`. Under the literal containment formula
(`join(packageRoot, "../runbooks/...")`) that escapes the package root and is `PKG-007`; under the
intended reading it is fine. Separately, `requires[].path` at **runbook** scope has no stated
resolution base at all — r23 relies on runbook-file-relative resolution
(`../../../schemas/examples/acme-incident-tools`) while the config example implies workspace-relative.

Required: state the resolution base for every path kind (`requires[].path` at project scope,
`requires[].path` at runbook scope, `exports.tools[].path`, `impl.path`, `toolRefs[].path`,
`tool-paths[]`) and state that containment is evaluated as
`realpath(join(dirname(referencing_file), p))` inside `realpath(containmentRoot)` with the
containment root named per kind. This is a security rule; ambiguity here is not acceptable.

### R6 — `PKG-020` and `PKG-021` are unreachable as specified

`runbook.v1.schema.json` is `additionalProperties: false` at the top level and on `ToolRef`. A
document containing `toolPackages:` or `toolRefs[].alias` therefore fails with the generic
unknown-key schema error, not with the diagnosable codes the spec promises. Nothing in §03d or §06
instructs the parse gate to detect these two keys ahead of generic schema validation. The entire
stated rationale for these codes ("the mistake is diagnosable rather than silently ignored") is lost.

Required: add normative parse-gate text (§03d) that these two keys MUST be checked before generic
structural validation and MUST emit `PKG-020` / `PKG-021`.

### R7 — `dependencies:` contradiction makes `PKG-016` unreachable

`tool-package.v1.schema.json` sets `dependencies: {maxItems: 0}` while its own `description` says a
non-empty array is "a schema-valid but semantically rejected document: the resolver MUST raise
PKG-016". As shipped, non-empty `dependencies` is a schema failure (`PKG-004`) and `PKG-016` can
never fire.

Required: remove `maxItems: 0` and let the resolver raise `PKG-016` (preferred — it gives the better
diagnostic), or delete `PKG-016` from the catalog. Do not ship both.

### R8 — `apiVersion` contradiction between spec example and schema

`06-tool-runtime.tex` §Package manifest shows `apiVersion: package/v1  # REQUIRED, const
"package/v1"`, while `tool-package.v1.schema.json`, `schemas/README.md`, and the reference example
all use `tool-package/v1`. A manifest authored from the normative spec snippet is rejected by the
normative schema. (The ruling itself was internally inconsistent here — §3.1 vs §11.5. `tool-package/v1`
is the correct value; the spec snippet is the artifact that must change.)

### R9 — Tier-3 tools cannot be pinned by the mechanism the spec prescribes

`05-extension-runtime.tex` instructs binding an extension/MCP tool "via `toolRefs[].package` using
the extension's identifier as the package component", but `ToolRef.package` is patterned as
reverse-DNS requiring at least one dot. §06's own MCP example declares `mcp-servers[].name:
acme-mcp` — a single label. The prescribed mechanism is unusable for MCP servers and for any
single-label extension id.

Required: reconcile. Either relax/split the pattern, add a distinct `origin`/`server` field, or state
normatively that tier-3 entries are addressed only by qualified `toolRefs[].name` and strike the
`toolRefs[].package` advice from §05.

---

## 2. Required corrections (blocking, lower severity)

### R10 — §05 extension collision text contradicts the uniform rule

The new text calls two distinct extensions sharing a name "a same-tier collision under the uniform
precedence rule" and then resolves it by precedence with a warning. Under the uniform rule, ties are
**errors**; if the two come from different sources it is not a tie. Additionally, a genuine tie (two
entries with the same `meta.name` in one `.gert/extensions.yaml`) is unspecified. Restate correctly
and assign an outcome to the true-tie case.

### R11 — Replay semantics for substitution are undefined

§13 defines replay behaviour for `cli`, `tool`, and `invoke` steps. Substitution introduces nested
runbook execution that is none of these. The spec does not say whether the substitute executes in
replay mode against the same scenario (as `invoke` does) or whether the tool action's recorded
response short-circuits the substitute entirely. Given §7.6 makes package drift non-fatal in replay,
this gap is directly load-bearing on the drift semantics. Specify it.

### R12 — Step-level `tool.version` is unreconciled with the new constraint sites

`ToolInvocation.version` already exists ("Runtime enforces semver compatibility with the declared
version"). That is a third constraint site and it is **runtime**-enforced, contradicting §4.4's
plan-time-predicate model and duplicating `toolRefs[].version`. Reconcile: deprecate it, or state
that it is evaluated at plan time under the same §Version Constraint Grammar and how it composes with
`toolRefs[].version` and `requires[].version`.

### R13 — Lock-file `root` pattern does not enforce its own invariant

`package-lock.v1.schema.json` `LockedPackage.root` uses
`^(([^\x00]+)|external:sha256:[0-9a-f]{64})$`. The first alternative matches everything — including
`C:\Users\cristian\...` — so the "never a raw absolute path" privacy invariant is unenforced and the
`external:` branch is inert (it is already matched by branch one). Tighten to a workspace-relative
POSIX pattern alternated with the `external:sha256:` form. Also add the strict SemVer pattern to
`LockedPackage.version` for consistency with `PackageMeta.version`.

### R14 — Error catalog is incomplete for two conditions the schemas create

Both `runbook.v1.schema.json` and §06 describe `toolRefs[].package` and `toolRefs[].path` as
"mutually exclusive… a schema-valid but semantically rejected document" — with **no error code**.
Likewise there is no code for a `toolRefs` entry whose `name` resolves to nothing in the frozen
catalog (`PKG-011` is specifically "not exported by the package"). Assign codes (new numbers, do not
reuse `PKG-019`) or fold these into existing codes explicitly.

### R15 — The real/mock unchanged-runbook acceptance scenario is not demonstrated

This was called out in the ruling's review scope and is not met. `kubectl.tool.yaml` declares
`allowed-environments: ["real", "dry-run"]`, omitting `replay`, while §12 enumerates run modes as
(real, dry-run, replay) and §06 says a tool not listed for the current mode is skipped. Combined with
R11, nothing in the shipped examples or r23 shows the same runbook executing unchanged in real and
mock/replay modes, and no scenario fixture accompanies r23. Required: make the reference tool and r23
executable unchanged across modes (or state normatively why substitution is real-mode-only), and add
the mode dimension to the fixture.

---

## 3. Non-blocking follow-ups (do not gate the merge)

1. **Stale citation path.** Every new normative artifact cites
   `.squad/decisions/inbox/barbara-tool-packages-architecture-ruling.md`; the ruling now lives at
   `.squad/decisions/archive/…`. Sweep the paths (schemas ×3, §02, §03d, §05, §07, §08, §10, §12,
   §13, `vector.schema.json`, README).
2. **Example package name drift.** Spec prose uses `acme.incident-tools`; the shipped example and
   r23 use `com.acme.incident-tools`; the directory is `acme-incident-tools`. Harmless but sloppy for
   a reference artifact — pick one.
3. **r23 lacks the `assessment.md` / `README.md` companion** that r01/r21 carry. Optional, but this
   is the fixture reviewers will read first.
4. **`PKG-W*` warnings are not expressible in `vector.schema.json`.** `PackageExpected` supports only
   `catalog` or `error{code}`. The deprecation warnings (`PKG-W001`/`W002`) and the external-root
   warning (`PKG-W003`) are therefore unverifiable by the corpus. Add a `warnings: [<code>]` member
   before Tess authors.
5. **Corpus-wide `apiVersion: runbook/v2` vs `runbook.v1.schema.json`.** r23 follows the existing
   corpus convention, so this is *not* Edith's defect — but the whole corpus currently fails
   `runbook.v1` validation on `apiVersion`, `inputs[].from`, and several step shapes. Separate
   ticket; do not fold into this revision.

### Tess's vectors — direction (still blocked)

The category enum, `TV-PKG-*` id pattern, and `PackageExpected` shape are accepted; Tess may treat
them as stable. Author `tv-pkg-resolve.yaml` **after** R1–R9 land, and add to the ≥46 floor:
- `PKG-007` vectors that fix the resolution base per path kind (R5), one per kind, including a
  runbook-scope `requires[].path` with `..` and a package-internal `impl.path` with `..`;
- `toolPackages:` and `alias:` vectors asserting `PKG-020`/`PKG-021` specifically, not a generic
  schema error (R6);
- a non-empty `dependencies:` vector asserting whichever single code survives R7;
- a tier-3 pinning vector matching whatever R9 resolves to;
- a substitution vector whose declared output is produced on one branch and not another
  (`PKG-027`), plus one asserting the caller-visible capture path from R3;
- a replay-mode drift vector once R11 is specified.

---

## 4. Gate disposition

**REJECTED.** Fifteen required corrections (R1–R15), of which R1–R5 are semantic contradictions or
under-specification in the substitution and path-safety contracts — the two highest-risk areas the
ruling explicitly reserved for this review pass — and R6–R9 are wiring failures that make ruled
error codes unreachable or ruled mechanisms unusable. The architecture direction is sound and is
**not** re-opened; the defects are contradiction and incomplete wiring, all locally fixable.

**Reviser:** Don. **Edith: locked out** for this cycle. Re-submit to me for a second gate pass; I
will re-review R1–R5 and R11 in particular. Deviations from this review require a new inbox entry,
not an inline edit.

— Barbara (Lead / Architect)
