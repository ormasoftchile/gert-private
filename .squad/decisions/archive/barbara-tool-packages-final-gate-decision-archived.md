# Final Implementation Gate Decision — GERT Tool Packages MVP (runtime)

**From:** Barbara (Lead / Architect, formal implementation gate — second pass)
**Date:** 2026-08-09T19:24-07:00
**Requested by:** Cristián Ormazábal Ortega
**Under review:** the complete uncommitted working tree of `C:\One\OpenSource\gert`
after David's independent revision
(`.squad/decisions/archive/david-tool-packages-revision-archived.md`), against my
rejecting first-pass gate entry
(`.squad/decisions/archive/barbara-tool-packages-implementation-gate-review-archived.md`),
the ratified ruling `barbara-tool-packages-architecture-ruling.md` (AR-TP-1..10),
the `TV-PKG-PATH-002` workspace-escape ruling, and the gate-3-approved spec
(`design/gert/sections/06`, `07`, `08`, `12`, `13`, `03d`, `grammar/gcp.ebnf` §3.4a).
**Lockout honoured:** Don and Ken made no contribution; the revision is David's alone.
**Method:** production code read directly; every blocker and every §3 item verified
against the code that actually runs, not against the revision summary. Tests executed
by me. One item (§3.7) was verified by me *outside* the test suite because its tests
skip on Windows — see below.

**VERDICT: APPROVED.**

---

## 1. Blocker-by-blocker verification

**B1 — plan-time substitution validation. RESOLVED.**
`cmd/gert/substitution_plan.go` adds a `flowwalk.Walker`-based visitor invoked from
`runWithMode` (`cmd/gert/run.go`) unconditionally, after catalog build/binding and
**before** `plannerImpl.Plan` and `eng.Start`. Because `dryrun.go` is nothing but
`runWithMode(args, RunModeDryRun)`, dry-run is covered by construction — the mode that
previously validated nothing. The visitor is structural: `flowwalk` never evaluates
`when:`, and it descends branch arms, iterate/parallel bodies and compensate bodies, so
unreachable steps are validated exactly like reachable ones ("validity is not
conditional on execution path" — satisfied literally). `BeforeInclude`/`EnterInclude`
both return true unconditionally, so lazily-expanded includes are still walked eagerly
at plan time. `planSubstitutionChain` calls `pkgsubst.Plan` per frame and recurses into
the substitute's own flow with an accumulated `[]pkgsubst.Frame`, so PKG-015 (cycle)
and PKG-028 (depth) are decided statically by DFS rather than from a live
`context.Value` stack; the executor's runtime check remains as defence in depth, as I
required. Verified live: `TestRun_Substitution_PlanTimeContractViolation_DryRun` and
`..._UnreachableStep` both pass through the real `runRun`/`runDryRun` entry points.

**B2 — digest closure. RESOLVED.**
`pkgcatalog.closureDigestLines`/`closeIncludeFile` resolve each exported tool's
`execute.path` substitute via `pkgpath.ResolveKind(KindExecutePath, ...)` and recurse
through package-internal includes via `KindPackageInternalInclude`, cycle-guarded by a
`closureVisited` set shared across all exports of the package, with actions iterated in
sorted order and all `digestLines` sorted before hashing. Include extraction is done by
walking raw `*yaml.Node`s (`collectIncludePathsRaw`), descending branch/iterate/parallel/
compensate — the right call, since the typed `schema.Step` decode collides on inline
tags and a substitute need not be a fully valid runbook at digest time. Closure files
are hashed as `hex  relpath_posix_nfc`, per §7.2. Files outside the closure remain
unhashed, as specified.

**B3 — catalog digest from the package digest. RESOLVED.**
Export entries are now deferred through a `pending` slice and added only after
`pkgDigest` is computed, with `Entry.Digest = pkgDigest` for tier-1; tiers 0/2/4 keep
their file digests, and `CatalogDigest()` (unchanged) therefore now proves what §7.3
says it proves. `TestBuild_DigestClosure_SubstituteRunbookByteChange_ChangesPackageDigest`
asserts exactly the property I asked for: a byte edit to a substitute runbook changes
both the package digest and `CatalogDigest()` while the exported tool file's own digest
is unchanged.

**B4 — removed/added package on resume. RESOLVED.**
`checkResumePackageDrift` now treats `Plan.Metadata.PackageDigests` as authoritative: a
recorded name absent from the freshly-resolved catalog yields `PKG-001` naming the
package(s), sorted, **before** any digest comparison; a resolved package absent from the
manifest is folded in as a forced `PKG-009` drift pair. Pairs are sorted for stable
diagnostics. `TestCheckResumePackageDrift_RemovedPackage_PKG001` and `..._AddedPackage_PKG009`
pass.

**B5 — `outputs.<name>` capture root. RESOLVED end-to-end.**
`gdp.SourceOutputs` added; `pkg/gcp/parser` accepts `outputs.<name>` and rejects any GDP
suffix (`GCP-PARSE-002`); `pkg/capture.resolveOutputs` resolves against the **current**
step's `Output` map only with `GCP-RESOLVE-002` for an unknown name; no `step.*.outputs.*`
form was added. The step-context restriction lives where it belongs — plan validation
(`internal/planner/validate.go:stepBindsSubstitutionAction` → `GCP-PARSE-001`), not the
bare grammar. David also found and fixed the latent defect that would have silently
defeated both B1 and B5: `schemaToolDefFromRuntime` was dropping `Execute`/`Outputs`, so
every catalog-bound substitution action read as a plain action to the planner. That was
a genuine find, not bookkeeping. Both B5 tests pass.

**§5 — include-closure semantics. RESOLVED via sanctioned fallback (b).**
`cmd/gert/include_closure.go` walks the entire include closure eagerly and fails closed
with a typed `PKG-017` for any *included* runbook declaring `requires:` or `toolRefs:`,
with a message that names the file and states the remedy. This is the fallback I
explicitly sanctioned; option (a) (full per-file lexical `BindFile`) is deferred and is
recorded below as a follow-up, not as a silent gap. Critically, the outcome I said I
would not accept — silent dynamic scoping — no longer occurs. Both §5 tests pass.

## 2. The nine §3 fixes — all verified

1. `ConstraintSources` — real. `mergeRequirements` returns an ordered per-package site
   list ("project requires:" / "runbook requires: (<path>)"), threaded through
   `loadPackage` → `LockedPackageInfo` → the `package/resolved` payload.
2. PKG-002 provenance — both `catalog.go` and `bind.go` now name the package and the
   contributing declaration site(s).
3. Build metadata in constraints — `+` added to `unsupportedPatterns` → PKG-003;
   versions still retain build metadata verbatim.
4. PKG-006 ordering — bare names and tiers sorted before iteration; collision member
   names sorted. Deterministic.
5. PLAN-010 — new `errkit.ErrPLAN010`; `resolveTool` wraps the registry sentinel as the
   cause, so the typed code is emitted *and* `errors.Is(err, ErrToolNotFound)` still
   matches through the unwrap chain. `internal/planner` green.
6. PKG-018 normalisation — export id/path uniqueness now also compares
   `NormalizeForComparison(..., true)`, raising PKG-018 on case/NFC-only collisions.
7. `maxLinkHops` — `checkLinkHopBound` counts hops explicitly ahead of `EvalSymlinks`.
   **Its tests skip on Windows, so I verified it myself**: a 10-link chain returns
   `PKG-008: ... exceeds the maximum of 8 hops`, and an 8-link chain resolves cleanly
   with no false positive, exercised through the exported `pkgpath.Resolve` on real
   Windows symlinks from a scratch module outside the product tree (since deleted). The
   fix is real, not merely declared.
8. Output coercion — `coerceOutputValue` returns an error and fails the step on a
   declared-type mismatch; no silent degradation.
9. `--package-map` provenance — `origin` added to the `package/resolved` payload
   (`omitempty`), sourced from the existing merge provenance.
   `TestRun_PackageMap_TraceRecordsConstraintSourcesAndOrigin` asserts both §3.1 and
   §3.9 on a real emitted trace event.

## 3. Constraints and hygiene

- **Protected files untouched.** All four are byte-identical to their pre-revision
  state; their mtimes (16:44, and 8/1 for the untracked workspace file) predate the
  entire revision window (18:20–19:20). The pre-existing `ping`/`curl` argv fix and the
  GXL var rewrite in the dirty tree are the user's own and were neither re-derived nor
  disturbed.
- **Spec not reopened.** No file under `design/` was modified during the revision
  window. Runtime was changed to match the ratified spec, as directed.
- **Tree state.** Nothing staged, nothing committed, dirty tree preserved.
- **Build/tests.** `go build ./...` clean. `go test ./...` green except a single
  timing-flaky `internal/serve/TestSSE_FilterByRunID`, which passed 5/5 on re-run, lives
  in a package this change does not touch, and is unrelated to tool packages. I re-ran
  `internal/planner`, `pkg/pkgsubst`, `pkg/gcp/parser`, `pkg/capture` and
  `internal/conformance` (Tess's vectors) individually: all green.

## 4. Ready for Cristián

**Yes.** The Tool Packages MVP runtime is ready to hand to Cristián as a working,
uncommitted tree. Every guarantee I declared non-negotiable now holds at plan time and
is evidenced in the trace: substitution contracts are static, the digest closure covers
the files that actually carry behaviour, the catalog digest proves the tool universe,
resume names a missing package, the ratified capture root exists, and no resolution path
is silently dynamically scoped. The one caveat he must be told plainly: **cross-process
`gert run --resume` still cannot work**, for a pre-existing reason unrelated to this
change (no `ExecutionPlan` persistence in `DirRunStore`), so `--allow-package-drift` and
PKG-009 are in-process-only today.

## 5. Accepted non-goals (true, and re-verified)

- Deep governance `deny_commands`/`allow_commands` enforcement inside substitute bodies:
  the evaluator is orphaned repo-wide, so substitution confers **no relative** privilege
  escalation; the static PKG-014 non-widening gate now genuinely fires at plan time, and
  the composed `EffectiveGovernance` is still recorded on `tool/substituted`.
- Cross-process resume plan persistence (pre-existing).
- Replay mode: does not exist in this runtime; the drift plumbing awaits it.
- `§5` option (a), full per-file lexical `toolRefs` binding: deferred behind a hard,
  typed PKG-017 refusal.

## 6. Follow-ups (tickets, none blocking)

1. Implement §5 option (a): include-closure global package set + per-file `BindFile`
   lexical binding; retire the PKG-017 fail-closed refusal when it lands.
2. Document the single-file `requires:`/`toolRefs:` restriction in user-facing docs —
   the README's new "Tool packages" section covers `--package-map` but not this
   restriction. The runtime error is self-explanatory, so this is documentation debt,
   not a defect.
3. Wire `GovernanceEvaluator` into the top-level and nested engines (pre-existing).
4. Persist `ExecutionPlan` in `DirRunStore` to make cross-process resume — and therefore
   end-to-end package-drift refusal — actually reachable (pre-existing).
5. Enable the `pkgpath` hop-limit tests on Windows where symlink privileges exist (they
   pass there; I verified the behaviour manually).
6. Extend `checkLinkHopBound` to count hops across intermediate path components, not only
   the terminal path's own chain. Containment is still enforced on the fully resolved
   path, so this is a hardening refinement.
7. Investigate the flaky `internal/serve/TestSSE_FilterByRunID` (unrelated, pre-existing).
8. Previously ticketed and still open: `ToolDef.Actions` map-vs-array divergence; the
   corpus-wide `apiVersion: runbook/v2` mismatch; the §07/§13 `replay` terminology
   overload; `rules:`/`requires-capabilities` composition in §5.4.

## 7. Disposition

**APPROVED.** B1–B5, the §5 include-closure item, and all nine §3 required fixes are
resolved in the code that runs, not merely in the summary. David's revision is accepted
in full, including his explicit, correctly-flagged scope reduction on §5 and his honest
reporting of the skipped Windows link tests — which is precisely why I could target my
own verification where it mattered. Credit is also due to Don and Ken, whose underlying
packages I found correct at the first gate and did not need to re-litigate here.

No third gate pass is required. The tree is ready for Cristián's review and commit.

— Barbara (Lead / Architect)
