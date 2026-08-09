# Architecture Gate Review #2 — GERT Tool Packages MVP (Don's revision of R1–R15)

**From:** Barbara (Lead / Architect)
**Date:** 2026-08-09T15:40-07:00
**Requested by:** Cristián Ormazábal Ortega
**Reviews:** full working-tree diff + untracked artifacts, against
`.squad/decisions/archive/barbara-tool-packages-architecture-ruling.md` (AR-TP-1..10, ratified)
and `.squad/decisions/inbox/barbara-tool-packages-gate-review.md` (R1–R15)
**Revision under review:** `.squad/decisions/inbox/don-tool-packages-revision-r1-r15.md` (Don,
authored under Edith lockout)
**Verdict:** **REJECTED** — narrow editorial/consistency sweep required. The architecture is
**not** reopened and R1–R15 are **not** reopened.
**Reviser designated:** **Ken** (Backend Dev). **Edith and Don are both locked out** of this
cycle — Edith authored the original, Don authored the revision under review.

---

## 0. R1–R15: verified resolved

I re-derived each item from the artifacts, not from Don's matrix. All fifteen are genuinely
resolved, several better than the minimum the review demanded.

| # | Verified how | Status |
|---|---|---|
| R1 | §06 §Tool Definition Schema now states `actions:` is **always an array, never a mapping**, declares itself (with §Action Substitution) the *sole normative source* for `.tool.yaml` structure, and normatively binds `step.tool.args` to `args:` for **both** kinds; `output:` (process, stream shape) vs `outputs:` (substitution, named values) are separated with a stated rationale. `kubectl.tool.yaml` matches. All 12 shipped `testdata/tools/*.tool.yaml` already used the list shape — the correct shape was chosen. | ✅ |
| R2 | Per-action key renamed `execute:` (`kind`, `path`); top-level mobile `impl:` untouched; §06 ¶"Interaction with tool-level `impl:`" assigns a deterministic outcome for the both-present case (native handler on declared mobile platforms, substitute elsewhere). No `impl.kind` remains anywhere. | ✅ |
| R3 | New GCP root `outputs.<name>` — `LocalOutputs` production in `gcp.ebnf` §3.4a, added to `LocalCapture`, added to the Source Prefix Reference Table, restricted to `execute.kind: runbook` tool steps, with process roots (`stdout`/`exit_code`/`json.*`) explicitly forbidden on such steps (`GCP-PARSE-001`) and unknown names `GCP-RESOLVE-002`. r23 uses `capture: {node_drained: outputs.drained}`. | ✅ |
| R4 | `drain-node.yaml` rewritten: a real `cli` step emits genuine JSON, and the runbook-level output is `value: "step.drain.json.drained"` — a valid `StepCapture` form per `gcp.ebnf` §3.1 (`step.{id}.json.{gdp}` is in the reference table). Single terminal path, so the declared output is produced on every terminal path (not `PKG-027`). I validated the file against `runbook.v1.schema.json`: **0 errors**. r23's assertion is now string-vs-string under PJVM coercion, with the reasoning recorded in-file. | ✅ |
| R5 | New §Resolution base per path kind + Table `tab:tool-path-bases` names `referencing_file` **and** `containmentRoot` for all seven path kinds, with the workspace-level vs package-internal class distinction and `PKG-W003` handling. The `execute.path` base≠containment-root asymmetry is stated explicitly and the `../runbooks/drain-node.yaml` case is worked through. Runbook-scope `requires[].path` is declaring-file relative; I re-derived r23's `../../../schemas/examples/acme-incident-tools` by hand — it lands correctly. Containment rule rewritten to reference the table rather than a hardcoded `root`. | ✅ |
| R6 | §03d adds a normative, **ordered** raw-document pre-check for `toolPackages:` and `toolRefs[].alias` that MUST run *before* generic schema validation, with an explicit prohibition on inferring these from schema error messages. | ✅ |
| R7 | `maxItems: 0` removed from `dependencies`; description now says the **resolver** raises `PKG-016`. `PKG-016` is reachable; `PKG-004` no longer pre-empts it. | ✅ |
| R8 | `apiVersion` swept: §06 manifest snippet is `tool-package/v1`; all four schema `const` values match their `$id` (`runbook/v1`, `tool-package/v1`, `config/v1`, `package-lock/v1`); `gert-package.yaml` validates against `tool-package.v1.schema.json` with **0 errors**. | ✅ |
| R9 | Tier-3 is addressable **only** by qualified `toolRefs[].name`; `toolRefs[].package` is normatively prohibited for tier 3 in both §05 and the `ToolRef` schema description, with the reverse-DNS/single-label rationale stated. The unusable advice is struck. | ✅ |
| R10 | §05 splits cross-source shadowing (precedence, warn+skip — *not* a tie) from a same-source tie (hard error at discovery, both paths reported). The contradiction is called out and corrected in place. | ✅ |
| R11 | §06 §Replay semantics + §13 bullet: substitution replays **like `invoke`**, nested steps matched individually by `argv` against the caller's scenario; governance still evaluated; explicitly tied to why `replay/packageDrift` is non-fatal; explicitly distinguished from §07's trace-event replay. `sec:replay-mode` label added so the cross-reference resolves. | ✅ |
| R12 | `ToolInvocation.version` rewritten: plan-time (correcting "runtime enforces"), intersected with `toolRefs[].version` / `requires[].version`, `PKG-002`/`PKG-003`, deprecated in favour of `toolRefs[].version`. Not a fourth independent site. | ✅ |
| R13 | `LockedPackage.root` pattern is now `^(([a-zA-Z0-9_.-]+(/[a-zA-Z0-9_.-]+)*)|external:sha256:[0-9a-f]{64})$` — a raw absolute path (POSIX or `C:\...`) cannot validate under either branch, so the privacy invariant is structural. `LockedPackage.version` gained the strict SemVer 2.0.0 pattern. | ✅ |
| R14 | `PKG-029` (package+path both present) and `PKG-030` (name resolves to no catalog entry) added at the next free numbers; `PKG-019` still reserved and unassigned; both wired into the `ToolRef` schema description; `PKG-030` vs `PKG-011` distinction stated. | ✅ |
| R15 | Reference tool's `allowed-environments` now `["real","dry-run","replay"]`; r23 documents the identical, unchanged runbook across all three modes with a concrete `commands:` override map covering **both** the top-level and the nested-substitute process invocations. Matching a real package (`requires[].path` → the reference package) and a mock (scenario override map) is demonstrated. Shipping the scenario inline in the fixture description rather than as `scenario.yaml` matches the existing corpus convention (r22 also ships only `schema.yaml`) — accepted. | ✅ |
| R16 (opt.) | `PackageExpected.warnings` added (`^PKG-W\d{3}$`), valid alongside `catalog` **or** `error`. I probed the schema directly: `catalog+warnings`, `error+warnings`, `PLAN-010`, and the mutual-exclusion negative case all behave correctly. `PKG-029`/`PKG-030` match the `^PKG-\d{3}$` code pattern. | ✅ |

**Independent validation I ran** (not taken on trust):

- `runbook.v1.schema.json` vs `drain-node.yaml` → 0 errors; vs `gert-package.yaml`/`tool-package.v1` → 0 errors; vs `r23/schema.yaml` → 1 error, `apiVersion: runbook/v2`, which is the corpus-wide pre-existing issue (gate-1 note #5), **not** this revision's defect.
- `python design/gert/scripts/verify_corpus.py` → **280/280**.
- Direct `jsonschema` probes of `PackageExpected` (above).
- Full `\label`/`\ref` closure across `sections/*.tex`: **293 labels, 223 refs, 0 unresolved, 0 duplicates.**
- Cross-checked that no `impl.kind`, `toolPackages:`, `actions[].inputs:`, or "first match wins"/"later entries win" tool- or extension-resolution text survives in §05/§06.

Also confirmed on the four specific questions in the gate charge: the unchanged runbook resolves
against both the real package and the mock/override map; governance composition is
intersection/OR/union as ratified with `PKG-014` raised at **plan** time (§12 table is verbatim
correct and now cross-linked from §06 and §07); path bases and containment roots are coherent and
per-kind; substitution outputs and replay are both defined.

## 1. Required corrections (blocking) — sweep only

Both items are the same failure mode: `03-schema-vnext.tex` and `08-security-and-trust.tex` were
never swept, so legacy prose now contradicts rules this workstream made normative. Neither
requires a design decision — the correct answer is already fixed and stated elsewhere.

### S1 — §03 `\subsection{toolRefs}` teaches a hard error and a deleted resolution rule

`03-schema-vnext.tex` lines ~170–188 (`\label{subsec:toolrefs}`) still reads:

- `alias: kubectl-wrapper  # optional local alias`, plus a paragraph explaining what `alias` does.
  `alias` was removed from `runbook.v1.schema.json` and its presence is now **`PKG-021`, a hard
  parse-gate error with a dedicated pre-schema check** (§03d). The schema chapter — the first
  place a reader looks for the runbook document shape — instructs the reader to author a key that
  the parse gate rejects. Nothing in §03 marks the text as superseded.
- *"The default discovery path for `name: foo` is `tools/foo.tool.yaml` relative to the runbook
  file. The `path` field overrides this."* This is a per-name, runbook-file-relative discovery
  rule. The ratified model (AR-TP-2, §06 §Tiers / §Precedence Rule / §Name Resolution Order
  (Phase B)) has no such rule: tier 2 is a **workspace**-rooted scan of `tool-paths[]` then
  `<workspace>/tools/`, bare-name binding is a lookup into the **frozen catalog**, an unmatched
  name is `PKG-030`, and an unbound name in the declaring file is `PLAN-010`. This is a survival
  of exactly the "implicit path-derived discovery" behaviour the ruling replaced, sitting
  unqualified in the schema chapter.

Required: rewrite `\subsection{toolRefs}` to match `runbook.v1.schema.json` and §06 — drop
`alias` (with a one-line note that it is `PKG-021`), drop the default-discovery-path sentence,
and point at §06 §Runbook Tool Bindings for `package` / `version` / `path` / deprecated
`source` / deprecated `actions`. Do not restate the tier rules in §03; cross-reference them.

### S2 — three `.tool.yaml` examples still use the mapping-shaped `actions:`

§06 now states normatively that `actions:` is *"always an array … never a mapping keyed by the
action name … the one canonical `tool/v1` action representation."* Three listings still show the
mapping form:

- `03-schema-vnext.tex:3175` (`actions:` → `get-pods:`)
- `03-schema-vnext.tex:3296` (`actions:` → `check:`)
- `08-security-and-trust.tex:320` (`actions:` → `deploy:` → `args:` / `sensitive_inputs:`) — and
  this is the **only** normative statement of where `sensitive_inputs` sits in a tool definition,
  so it cannot simply be deleted.

§06's claim to be the "sole normative source for `.tool.yaml` structure" mitigates this but does
not remove it: a reader authoring from §08 produces a document the canonical rule forbids. All 12
shipped `testdata/tools/*.tool.yaml` use the list form, so the conversion is mechanical.

Required: convert all three listings to the list form (`- name: <action>`), preserving
`sensitive_inputs` in §08 under the converted entry.

## 2. Non-blocking (do not gate; fold into the S1/S2 sweep if convenient)

1. **Stale ruling citations — repeat of gate-1 note #1, still unfixed.** 15 occurrences across
   §02, §03d, §06, §07, §08, §10, §12, §13, three schemas, `vector.schema.json` (×2),
   `schemas/README.md`, and `r23/schema.yaml` still cite
   `.squad/decisions/inbox/barbara-tool-packages-architecture-ruling.md`; the ruling is in
   `archive/`. Purely mechanical.
2. **Package-name drift — repeat of gate-1 note #2.** §06 prose uses `acme.incident-tools`; the
   shipped example and r23 use `com.acme.incident-tools`.
3. **§06 §Output contract mis-cites the new grammar.** It says "gcp.ebnf §3.4 Local Step Capture,
   `LocalStructured`"; the production is `LocalOutputs` in §3.4a. r23 cites §3.4a correctly.
4. **No cross-step form for substituted outputs.** `LocalOutputs` was added to `LocalCapture` but
   `StepCapture` gained no `step.{id}.outputs.{name}` form, so a substituted action's output is
   capturable only in its own step's `capture:` block. That is a coherent MVP scope choice; it is
   simply not stated. One sentence in §06 §Output contract would close it.
5. **r23's dry-run narrative overreaches.** It says the substitute is skipped and therefore
   `assert_drained` "only reaches on non-dry-run modes" — but the flow has no `when:` guard, so
   the assert step *is* reached in dry-run and would evaluate an unproduced capture. The general
   question (what a downstream step does when an upstream tool step was mode-skipped) is
   pre-existing and under-specified spec-wide, so this is not Don's defect and not a gate item —
   but the fixture comment should not assert a guard that isn't there. Real and replay modes,
   which are what R15 actually turned on, are correctly demonstrated.
6. **`apiVersion: runbook/v2` corpus-wide** — unchanged, separate ticket, as agreed at gate 1.
7. **r23 `assessment.md`/`README.md` companion** — still absent; still optional.

## 3. Tess-owned conformance work (deferred, not blocking)

R1–R9 have landed, so Tess is unblocked and may author `tv-pkg-resolve.yaml` (≥46 vectors) now.
The stable surfaces are: the `PKG-*` category enum, the `TV-PKG-*` id pattern, `PackageExpected`
**including the new `warnings:` member**, the `outputs.<name>` capture root, the `execute:` action
key, `PKG-029`/`PKG-030`, Table `tab:tool-path-bases`, and the replay-as-`invoke` substitution
model. The gate-1 vector list stands, with these now-concrete targets:

- one `PKG-007` vector per row of Table `tab:tool-path-bases`, including a runbook-scope
  `requires[].path` with `..` and a package-internal `execute.path` with `..` that stays inside
  the package root (the legal case, which must **not** be `PKG-007`);
- `toolPackages:` / `alias:` vectors asserting `PKG-020` / `PKG-021` specifically, exercising the
  pre-schema ordering rule rather than a generic unknown-key failure;
- a non-empty `dependencies:` vector asserting `PKG-016` (not `PKG-004`);
- a tier-3 vector using a qualified `name` and one asserting that `toolRefs[].package` on a
  tier-3 supplier is rejected;
- `PKG-029` and `PKG-030` vectors, and a `PKG-011` vector alongside `PKG-030` to prove they are
  distinguishable;
- a `PKG-027` vector (output produced on one branch only) and one asserting `outputs.<name>`
  resolution, plus a negative asserting `GCP-PARSE-001` for `json.*`/`stdout` on a substituted
  action's step;
- a replay-mode drift vector (`replay/packageDrift`, non-fatal) and `PKG-W001`/`W002`/`W003`
  vectors using the new `warnings:` member.

## 4. Gate disposition

**REJECTED**, narrowly. This is a materially different rejection from gate 1: all fifteen blocking
corrections are genuinely resolved — verified against the artifacts and by re-running schema and
corpus validation, not accepted on the revision author's word — and several (R5, R11, R13) are
resolved more thoroughly than the minimum. The two remaining blockers are a documentation sweep of
two chapters this workstream changed the meaning of but never edited: §03's `toolRefs` subsection
teaches `alias:` (now a hard `PKG-021` error) and a tool-discovery rule the ratified tier model
deleted, and three `.tool.yaml` listings still use the `actions:` mapping shape that §06 now
forbids in the same document. Both are mechanical; neither requires a design decision; the correct
answer is already written down elsewhere in the spec.

I am rejecting rather than approving-with-conditions because this spec goes to the user next, and
a reader who opens the schema chapter first is currently told to author a key that the parse gate
rejects. That is precisely the class of defect this gate exists to stop, and it is an hour of work
to remove.

**Reviser:** **Ken**. **Edith and Don: both locked out** for this cycle. Scope is limited to §1
(S1, S2) and, at Ken's discretion, the mechanical items in §2 — **no other artifact may be
touched**, and R1–R15 and AR-TP-1..10 are closed and must not be re-litigated. Re-submit to me;
the third pass will be a diff-only check of S1/S2 and a re-run of `verify_corpus.py` plus the
schema validations above.

— Barbara (Lead / Architect)
