# Implementation Gate Review — GERT Tool Packages MVP (runtime)

**From:** Barbara (Lead / Architect, formal implementation gate)
**Date:** 2026-08-09T17:58-07:00
**Requested by:** Cristián Ormazábal Ortega
**Under review:** the complete uncommitted working tree of `C:\One\OpenSource\gert`
(25 modified files, 20 new files/packages), authored by Don (runtime core + final
integration pass) and Ken (CLI wiring), against the ratified ruling
`.squad/decisions/archive/barbara-tool-packages-architecture-ruling.md` (AR-TP-1..10),
the binding ruling `.squad/decisions/inbox/barbara-pkg-path-002-workspace-escape-ruling.md`,
the gate-3-approved normative spec (`design/gert/sections/06`, `07`, `08`, `12`, `13`,
`03d`), `design/gert/grammar/gcp.ebnf` §3.4a, and Tess's 85-vector
`conformance/tv-pkg-resolve.yaml`.
**Method:** production wiring read directly (`cmd/gert`, `internal/adapter`,
`internal/executor`, `internal/engine`, `pkg/pkgcatalog`, `pkg/pkgsubst`, `pkg/pkgdrift`,
`pkg/pkgpath`, `pkg/semver`, `pkg/gcp/parser`). Agent summaries were used only to locate
code, never as evidence. `go build ./...` clean; `go test ./...` green repo-wide, zero
failures, zero regressions — verified by me, not taken on trust.

**VERDICT: REJECTED.**

**Revision owner: David (Integration Engineer).** Don and Ken are locked out: both
authored the runtime changes under review. David is the eligible remaining engineer and
the residual work is overwhelmingly integration/wiring — plan-phase sequencing, digest
closure traversal, include-closure threading, event-payload provenance, and one capture
grammar production — which is squarely his competence. Edith and Tess remain locked out
of runtime code by role (spec/conformance ownership).

This is a rejection of a genuinely strong body of work. `pkg/semver`, `pkg/pkgpath`,
`pkg/pkgcatalog`, and `pkg/pkgsubst` are correct, tested, and faithful to the ruling on
the great majority of surfaces. The rejection is narrow and rests on five substantive
spec violations, all of which defeat guarantees the ruling declared non-negotiable.

---

## 1. What I verified as CORRECT (not re-litigable in the revision)

| Area | Finding |
|---|---|
| SemVer 2.0.0 + constraint grammar | Strict parse, leading `v` rejected (PKG-025), `^1.4.2`/`^0.4.2`/`^0.0.3`/`~1.4.2` bounds exact, §11 ordering, prerelease same-tuple rule exact, space-AND conjunction, `\|\|`/`*`/`x`/hyphen/partial/`!=` all rejected as PKG-003. |
| Secure path resolution | Syntax gate precedes filesystem access; backslash/UNC/extended-length/device/drive-relative/`~`/env-expansion/URI/control chars → PKG-007; reserved DOS names and trailing dot/space → PKG-018; containment checked on the **resolved** path after link resolution. |
| TV-PKG-PATH-002 ruling | `pkgpath.Class`/`Resolve`/`ResolveKind` implement my binding ruling **exactly**: `PackageInternal` escape → PKG-007; `WorkspaceLevel` escape → `(resolved, external=true, nil)` + PKG-W003 advisory, never a rejection. Confirmed live in `catalog_trace_integration_test.go` (`External == true`, no failure). |
| Two-phase model | `Build` is a pure function of options+filesystem with no step context; `BindFile` copies `byBare` into a file-local map and never mutates the frozen catalog. |
| Five tiers | Tier order correct; project `requires:` processed before runbook `requires:`; `tool-paths` in declaration order then `<workspace>/tools/`; tier-3 entries unconditionally stripped of bare names (`catalog.go:189`, `e.Bare = ""`). |
| Precedence/collision | Same-tier bare collision = hard PKG-006, no warn-and-continue. Undeclared cross-tier shadowing = **PKG-022, genuinely enforced** (`bind.go:210-213`, reached only when neither `package` nor `path` is declared). Qualified names never collide. No surviving "first match wins"/"later wins" logic. |
| Binding contract | `package`/`path` mutual exclusion → PKG-029; `toolRefs[].version` checked against the **resolved tool definition's** version (the no-op bug Don found and fixed is correctly fixed); `actions` documentation-only + PKG-W001, never gating; `source` non-participating + PKG-W002; `alias` → PKG-021 and `toolPackages:` → PKG-020 via a genuine **Phase-0 raw-YAML scan ahead of structural validation** (`internal/parser/parser.go:58-75`); unresolved name → PKG-030; unexported tool → PKG-011; PKG-016 on non-empty `dependencies`. |
| Warning discipline | PKG-W001/W002/W003 are non-fatal, split via `errkit.SplitWarnings`, surfaced on stderr, and — critically — no longer discard otherwise-valid bindings. The regression guard test for this is real. |
| Project binding + `--package-map` | `.gert/config.yaml` (`config/v1`) is genuinely loaded on the production `gert run` path (`run.go:181`). `mergePackageBindings` merges **per package name**, so a partial override leaves unmentioned packages resolved normally. `TestRun_PackageMap_RealVsMockBinding` runs the **byte-identical runbook** through the real `runRun()` entry point twice — real binding and overridden binding — with no runbook edit. The unchanged-runbook requirement is met. |
| Substitution declaration & scope | `execute.path` resolves against the declaring `.tool.yaml`'s directory with the **package root** as containment root — not the caller, not the workspace, not cwd. Caller scope is genuinely isolated: the nested engine receives only the action's rendered arg values (`childVars`), never the caller's `vars`. Values, never expressions. |
| Substitution signatures | `validateInputs`/`validateOutputs` perform true bidirectional key-set equality, exact type identity, and required-implication → PKG-013. PKG-026 (`from: prompt`/`env`) genuinely enforced. |
| Governance arithmetic | `composeGovernance` is exact: `require_approval` OR, `deny_commands`/`deny_env_vars`/`redact` UNION, and `allow_commands` a **true INTERSECTION** with the widened remainder raised as PKG-014. `MaxDepth = 4`. |
| Trace events | `package/resolved`, `catalog/frozen` emitted from the real CLI at the Phase-C boundary before `eng.Start`, with a pre-generated `runID` stamped onto `plan.RunID` so pre-run and run events share one `run_id`; `tool/substituted` emitted from the executor with the composed governance recorded. Proven end-to-end by CLI-level integration tests, not unit shims. |
| Dry-run | The `"tool"` executor is replaced wholesale by `DryRunExecutor`, so a substitute's side-effecting steps are never reached. |
| Resume refusal mechanics | `checkResumePackageDrift` is called from the real `--resume` branch, recomputes the catalog exactly as a fresh run would, returns PKG-009 without the flag, and emits `governance/packageDriftAccepted` **with both digests and the operator identity** under `--allow-package-drift`. Drift is never silently bypassed. |
| Protected user edits | `examples/simple-health-check/simple-health-check.runbook.yaml` (GIS var migration), `internal/tool/native.go` + `native_test.go` (Windows `ping -c`→`-n` argv normalizer), and untracked `examples.code-workspace` are **byte-identical to their pre-session state**. Verified by diff inspection, not by assertion. Nothing staged, nothing committed, dirty tree preserved. |
| Regressions | None. Full `go test ./...` green including all 22 pre-existing runbook parser fixtures. |

---

## 2. BLOCKERS

### B1 — Substitution is validated only when a step actually executes; there is no plan phase

`pkgsubst.Plan` has exactly **one** call site in the entire repository:
`internal/executor/tool.go:166`, inside `ToolExecutor.Execute`. Nothing in `cmd/gert`,
`internal/adapter`, or the planner invokes it. Consequently PKG-013, PKG-014, PKG-015,
PKG-026, PKG-027 and PKG-028 are all **runtime** errors raised lazily on the live call
stack, per dispatched step.

This contradicts the ruling in three places, in its own words:

- §5.4: "PKG-014, raised at plan time by static comparison — **not at runtime**.
  Widening detectable only at runtime is a design defect; this is why `allow_commands`
  is intersected rather than merged."
- §5.5: "**Unreachable steps are still validated: validity is not conditional on
  execution path.**"
- §5.6: "A cycle is PKG-015, detected **at plan time by DFS**."

Concretely, today: a governance-widening substitute, a substitution cycle, a depth-5
chain, or a signature mismatch sitting behind a `when:` guard or an untaken `branch`
arm passes `gert run` silently. Worse, because dry-run swaps out the whole `"tool"`
executor, `gert run --dry-run` validates **nothing** about substitution — the mode an
operator would use precisely to check a runbook before executing it. Cycle/depth state
is carried in `context.Value` (`substitution_context.go`), i.e. a live call stack, not
a static frame graph.

**Fix:** add a plan-phase substitution validator invoked from `cmd/gert/run.go` after
`BuildPackageCatalog`/binding and **before** `eng.Start` (and unconditionally in
dry-run). It must walk every `step.tool` node — reachable **and** unreachable — in
every runbook of the include closure, resolve the bound action, and for
`execute.kind: runbook` call `pkgsubst.Plan` recursively, accumulating `(package, tool,
action)` frames by DFS so PKG-015 and PKG-028 are decided statically. The runtime
context-frame check may remain as defence in depth; it may not be the only check.

### B2 — Package digest closure omits substitute runbooks and package-internal includes

`loadPackage` (`catalog.go:384-458`) builds `digestLines` from exactly two sources:
`gert-package.yaml` (line 386) and each `exports.tools[].path` file (line 434). There
is no traversal of `execute.path` targets or package-internal includes.

§7.2 is explicit: the digest set is "`gert-package.yaml`, every `exports.tools[].path`,
**and every file transitively referenced by an exported tool via `impl.path` and
package-internal includes**."

The consequence is not academic and it is not cosmetic: the substitute runbook — the
file that contains the *actual executable logic* of a `kind: runbook` action, newly
shipped in this very change — can be rewritten arbitrarily without altering the package
digest, the lock, or the catalog digest. Resume drift (§7.5) therefore passes against a
package whose behaviour has changed. That defeats the stated purpose of the digest: "any
normalisation is a second parser" was the rationale for hashing raw bytes; hashing the
*wrong set of files* is a strictly larger hole than any normalisation bug.

**Fix:** in `loadPackage`'s export loop, after parsing each `ToolDef`, resolve each
action's `execute.path` and every package-internal include transitively (via
`pkgpath.ResolveKind` with `KindExecutePath`/package-internal containment, cycle-guarded)
and add each closure file to `digestLines` as `hex  relpath_posix_nfc`. Files outside the
closure remain unhashed. Add a conformance-aligned test asserting the PKG-DIGEST vector
property "digest changes on content change" for a substitute runbook, and
"digest unchanged when a non-exported file changes".

### B3 — Catalog digest uses the file digest where the ruling requires the package digest

§7.3 defines the catalog-digest line as `qualifiedName + "  " + packageDigestOrFileDigest
+ "  " + tierNumber`. For tier-1 (package-sourced) entries the correct value is the
**package** digest. `catalog.go:453` sets `Entry.Digest = fileDigest`; `pkgDigest`
(line 458) is written only to `LockedPackageInfo.Digest` (line 472) and never reaches
`CatalogDigest()` (line 569, which uses `e.Digest` for every tier).

Net effect: a package manifest or version bump that does not touch the exported tool
file's bytes produces an **identical** catalog digest. Combined with B2, the catalog
digest — the single value the ruling says "proves two runs saw the same tool universe"
— does not prove that.

**Fix:** compute `pkgDigest` before entries are appended (or backfill it onto the
entries after the export loop) and use it as `Entry.Digest` for tier-1 entries. Tiers
0/2/4 keep the file digest.

### B4 — Resume drift cannot detect a removed or added package

`resume_drift.go:101-104` iterates over **the packages present now** and skips any name
not tracked in the manifest (`if !tracked { continue }`). Therefore: a package recorded
in the run manifest but **absent** on resume is never noticed, and a package newly
appearing is silently accepted. §7.5 rule 5 is unambiguous: "Missing packages on resume
are PKG-001, never a fallback to a different tier." PKG-001 is never raised on this path.

The catalog-digest comparison does catch most such cases incidentally, but incidental
coverage is not the specified behaviour, and it produces PKG-009 where the ruling
requires PKG-001 with the missing package named — a materially worse diagnostic for the
operator.

**Fix:** iterate the manifest's `PackageDigests` as the authoritative set: a name in the
manifest with no corresponding resolved package on resume → PKG-001 naming the package;
a resolved package absent from the manifest → PKG-009 drift. Add tests for both.

### B5 — The ratified `outputs.<name>` capture root is not implemented

`gcp.ebnf` §3.4a defines `LocalOutputs = "outputs" "." OutputName` as a first-class
local-capture root, and §06 `Caller-visible capture namespace` states that a substituted
action's declared outputs cross the boundary under `outputs.<name>`, capturable in the
owning step's own `capture:` block (and deliberately with **no** `step.{id}.outputs.{name}`
cross-step form). This production was added specifically for R3 and survived three gate
passes.

The runtime GCP capture parser (`pkg/gcp/parser/parser.go:51-74`) recognises only
`step`, `http`, `event`, `exit_code`, `stdout`, `stderr`, `json`, `yaml`. `outputs` falls
to `default:` → `GCP-PARSE-001 unknown source prefix "outputs"`. A caller's only access
to a substituted action's outputs is `StepResult.Output[name]` via the CLI's JSON
summary.

I record Don's framing accurately — the capture vocabulary is a fixed set — but that is a
description of the defect, not a justification for it: the vocabulary is fixed *by a
grammar we ratified and then did not implement*. Tess also authored PKG-SUBST vectors
against this root. A ratified grammar production absent from the runtime is a spec/runtime
divergence, and this one severs the declared data path out of a substituted action.

**Fix:** add an `"outputs"` prefix case to `capturePath` resolving `outputs.<name>`
against the **current step's** `Output` map only, valid only for a `tool` step whose bound
action is `execute.kind: runbook` (else `GCP-PARSE-001`), unknown name →
`GCP-RESOLVE-002`, no GDP suffix permitted; wire the corresponding source into the capture
resolver. Do **not** add any `step.*.outputs.*` form.

---

## 3. REQUIRED, LOWER-SEVERITY FIXES (fold into the same revision)

1. **`ConstraintSources` is permanently empty.** `trace_events.go:39` emits
   `ConstraintSources: []string{}` unconditionally. See §4, item 4 — this is a required
   fix, not an accepted gap.
2. **PKG-002 messages omit declaration-site provenance.** §4.4 requires the error to
   report "package name, resolved version, effective constraint, **and the list of
   declaration sites that contributed to the intersection**." `bind.go:295` reports
   neither package name nor sites; `catalog.go:376` omits the sites. `mergeRequirements`
   already computes `scopeByName` — thread it through.
3. **Build metadata in constraints is wrongly accepted.** §4.2 lists "build metadata in
   constraints" as PKG-003. `unsupportedPatterns` (`constraint.go:36-42`) never checks
   `+`, and `ParseVersion`'s regex tolerates `+BUILD`, so `>=1.4.2+build.5` parses as a
   valid constraint. Reject `+` in a comparator before parsing. (Build metadata remains
   ignored for ordering/satisfaction and retained verbatim in lock/trace — that part is
   correct.)
4. **PKG-006 error ordering is non-deterministic.** `detectSameTierCollisions`
   (`catalog.go:210-229`) iterates `range c.byBare`, an unsorted Go map, and the ordering
   leaks into the returned error slice. The catalog and its digest are unaffected
   (entries live in an append-ordered slice), so §2.4's digest guarantee holds — but a
   governance-bearing plan report must not vary run to run. Sort the keys.
5. **PLAN-010 is never raised.** An unbound `step.tool.name` yields the generic,
   untyped `planner: tool not found`. §5.5 rule 1 and the §9 catalog require PLAN-010
   with the structured error format. Emit the typed code on the toolRefs-binding-aware
   path.
6. **PKG-018's normalisation half is dead.** Export uniqueness in `catalog.go:389-395`
   compares raw strings; `pkgpath.NormalizeForComparison` exists but is never called from
   `pkgcatalog`. Two exports differing only by ASCII case or NFC form are not caught as
   PKG-018 (§6.1 rule 5) — which is exactly the "works on Linux, breaks on Windows"
   class the rule exists to fail at authoring time, on every platform.
7. **`maxLinkHops = 8` is declared but never enforced** (`pkgpath.go:39`); link-chain
   bounding relies entirely on OS `ELOOP`. §6.3 rule 3 requires an explicit bound with
   PKG-008. Count hops explicitly.
8. **Silent output type fallback.** `coerceOutputValue` (`internal/executor/tool.go`)
   returns the raw string when `strconv` conversion fails, so a substitute promising
   `int` can hand the caller a `string` at runtime with no error. The *contract* check is
   correctly exact and this is only value materialisation — but it must fail the step,
   not degrade silently.
9. **`--package-map` override provenance is stderr-only.** It is printed
   (`run.go:220-222`) but does not reach the trace. §7.4 makes package provenance part of
   the evidence record; an override of where a package came from is exactly the fact an
   auditor needs. Add it to the `package/resolved` payload.

---

## 4. Explicit classification of Don's reported gaps

Don reported these honestly and prominently rather than burying them. That is the
correct behaviour and it is why this review could be done at all. Classification is
nonetheless on the merits.

**(1) Deep governance deny/allow-command enforcement inside substitute bodies —
ACCEPTABLE NON-GOAL. Ticketed, not blocking.**

I verified the wider claim myself: `governance.PolicyEvaluator` is orphaned repo-wide.
`internal/governance.BuildEvaluator` has no production caller;
`engine.EngineConfig.GovernanceEvaluator` is set neither by `adapter.BuildEngineConfig`
(top level) nor by `runSubStepsViaEngine` (nested). `deny_commands`/`allow_commands`/
`deny_env_vars` are unenforced for **every** CLI step in this runtime, substituted or
not.

That is decisive in Don's favour. §5.4's invariant is that substitution "MUST NOT be a
privilege-escalation path" — escalation is *relative*. A substitute body is subject to
exactly the same (absent) enforcement as the caller's own steps, so no privilege is
gained by substituting. Demanding that this change build the platform's entire
governance enforcement path would be scope creep of the first order, and I will not
impose it on a revision owner.

Two conditions attach. First, this is acceptable **only because B1 is being fixed**: the
static PKG-014 non-widening gate must actually fire at plan time, independent of
execution, so a widening substitute is rejected before it can rely on the absence of
runtime enforcement. Second, the composed `EffectiveGovernance` must continue to be
recorded on `tool/substituted` — it already is, and that is what keeps the gap visible
rather than silent. File a standalone ticket: "wire `GovernanceEvaluator` into the
top-level and nested engines."

**(2) Inability to re-capture substitution outputs via `capture:` — BLOCKER (B5).**

Not accepted. See B5. The `outputs.<name>` root is a ratified grammar production in
`gcp.ebnf` §3.4a with normative prose in §06 and conformance vectors written against it.
The fixed capture vocabulary is the thing that needs changing, not the reason to decline.
The fix is bounded: one parser case and one resolver source.

**(3) In-memory resume plan limitation — ACCEPTABLE NON-GOAL, but must be recorded.**

Confirmed and confirmed pre-existing. `DirRunStore.plans` is an in-memory map;
`RegisterPlan` is only ever called from `Start`; `adapter/wire.go` constructs a fresh
store per CLI invocation; `internal/engine.Resume` fails at "plan not available for
resume" before drift-checking is reached. A genuine two-process `gert run --resume`
therefore cannot work today **for reasons that predate this change entirely**.

Don is not responsible for that and correctly declined to paper over it, validating
`checkResumePackageDrift` at unit level with a fake handle instead of faking a passing
CLI test. That was the right call and I want it recorded as such. The drift feature is
correct **as wired**; it is unreachable end-to-end because of an unrelated architectural
gap. Two obligations: file a ticket for `ExecutionPlan` persistence in `DirRunStore`
(the plan metadata already serialises, so this is genuinely small), and document
`--allow-package-drift`/PKG-009 as in-process-only until that lands, so nobody ships
believing resume integrity is enforced across restarts.

**(4) Empty `ConstraintSources` — BLOCKER-adjacent; REQUIRED FIX, not an accepted gap.**

Rejected as a non-goal. `mergeRequirements` already computes `scopeByName`; the
provenance exists and was simply not threaded into the payload. §7.4 defines
`constraintSources` as part of the `package/resolved` envelope and §4.4 requires the same
information in PKG-002. Emitting a permanently-empty field in an evidence artefact is
worse than omitting the field: it looks like "no constraints contributed" to every
downstream consumer and to any auditor. This is a wiring fix of modest size, in an
evidence record, in a governance-bearing product. It goes in this revision. I credit Don
for documenting it rather than faking a value — that instinct was right; the conclusion
was not.

**(5) Absent replay mode — ACCEPTABLE NON-GOAL. Confirmed, no ticket beyond a note.**

I verified independently: there is no `gert replay`, no `--mode replay`, no replay entry
point anywhere in `cmd/`. §7.6 specifies drift behaviour *for a replay mode*, and that
mode does not exist in this runtime. `pkgdrift.ModeReplay`, `ReplayDriftEvent`, and
`trace.EventKindReplayPackageDrift` exist and are unit-tested, so the day a replay
command lands the wiring is a single call. Leaving it unwired is correct; inventing a
replay command to have somewhere to call it from would have been worse. No blocker.
Record the dependency in `10-open-questions.tex`'s deferral list so the obligation
travels with the replay feature.

---

## 5. Additional finding not in Don's list — include closure semantics (AR-TP-8)

I am recording this as a **BLOCKER**, resolvable either way.

`cmd/gert/run.go:227,258` builds the catalog from the **root** runbook's `requires:`
only and binds the **root** runbook's `toolRefs:` only. No traversal of the include
closure exists: an included child runbook's `requires:` is never merged into the frozen
set (§8.2's global package set), and an included child's `toolRefs:` is never bound
through `BindFile` (§8.1's lexical scoping). Child steps resolve their tool names against
the process-global runtime registry, which is populated by the workspace directory scan
plus the root's overrides — that is **dynamic scoping**, the model §8.1 explicitly
rejected as "the decisive call," on the grounds that it "makes a child runbook's meaning
depend on who included it, which destroys both composability and static reachability
analysis." PKG-017 is defined in `errkit` and raised nowhere, so §8.3's lazy-include
late-binding rule is likewise unenforced.

**Fix — either is acceptable:**

- **(a) Implement it.** Enumerate the include closure at plan time; union every file's
  `requires:` into the frozen package set with constraints intersected (PKG-002 on empty
  intersection, PKG-005 on intra-list duplicates, PKG-024 on conflicting paths); call
  `BindFile` per file so each file's `toolRefs` bind lexically; raise PKG-017 for any
  package introduced after freeze, including by a lazy include. This pairs naturally
  with B1's closure walk — do them together.
- **(b) Fail closed.** If (a) is too large for this revision, make an included runbook
  that declares `requires:` or `toolRefs:` a hard, typed, diagnosable error, and document
  the single-file restriction. Silently dynamic-scoped resolution is the one outcome I
  will not accept: it is the failure mode the ruling was written to prevent, and it fails
  quietly.

---

## 6. Disposition

**REJECTED.** Blockers: **B1** (no plan-time substitution validation), **B2** (digest
closure omits substitute/include files), **B3** (catalog digest uses file digest for
tier-1), **B4** (resume cannot detect missing/added packages, PKG-001 unreachable),
**B5** (`outputs.<name>` capture root unimplemented), and **§5** (include closure:
global package set and lexical binding absent — implement or fail closed). Plus the nine
required lower-severity fixes in §3, of which `ConstraintSources` (§3.1) is the one
promoted out of Don's accepted-gap list.

**Revision owner: David.** Don and Ken are locked out as authors. David must not consult
either on design; he may read their code and history entries, which are detailed and
accurate. AR-TP-1..10, R1–R15, S1/S2, and the gate-3-approved spec are **closed** and
must not be reopened — this revision changes runtime code to match the ratified spec, not
the spec to match the runtime. The one exception: if implementing B5 or §5 surfaces a
genuine spec ambiguity, David files a decision-inbox entry and stops on that item rather
than choosing for himself.

**Constraints carried into the revision, unchanged:** preserve the dirty tree; do not
touch `examples/simple-health-check/simple-health-check.runbook.yaml`,
`internal/tool/native.go`, `internal/tool/native_test.go`, or
`examples/simple-health-check/examples.code-workspace`; no commits; keep
`go build ./...` and `go test ./...` green with zero regressions.

**Not blocking, ticket separately:** `GovernanceEvaluator` wiring (repo-wide, pre-existing);
`ExecutionPlan` persistence in `DirRunStore` (pre-existing, blocks cross-process resume);
replay mode (does not exist); `ToolDef.Actions` map-vs-array divergence (pre-existing,
flagged by Don, correctly judged out of scope); the corpus-wide `apiVersion: runbook/v2`
mismatch; the §07/§13 `replay` terminology overload; the `PKG-002` intersection-provenance
work is *in* scope per §3.2, but the deeper `rules:`/`requires-capabilities` composition
of §5.4 is deferred with a ticket, since neither field has any enforcement consumer today.

A second gate pass is required, scoped to B1–B5, §5, and §3 only.

— Barbara (Lead / Architect)
