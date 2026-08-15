# ken — History Archive

**Archived:** 2026-08-15T16:27:17.3698403-07:00
**Size:** 24755 bytes

---

# Ken — History

## Seed Context (2026-06-05)

- **Project:** GERT — Governed Executable Runbook Technology
- **Owner / User:** ormasoftchile (Germán)
- **Squad home:** `gert-private` (this repo). The squad is single and serves all repos. Runtime work happens in `ormasoftchile/gert`, but squad memory stays here.
- **My role:** Second Backend Dev, paired with Don
- **Why I was hired:** OQ-M5 of the Runtime Migration Plan (ratified 2026-06-05, commit 541d194) called for a pair to run Streams E (GIS) and F (GCP) in parallel with Don's critical path on Streams A–D, G, H. Target wallclock: 10–15 days vs. 12–18 solo.

## Project Snapshot at Hire

- **Tech Stack:** Go (runtime), Azure (Functions, Service Bus, Container Apps, Static Web Apps, Entra ID), TypeScript (web/extensions)
- **Phase 1 of GXL/GIS/GCP design:** complete in `gert-private`. 208 conformance vectors, full spec sections 03a–03d, parse-gate, three EBNF grammars, fixture migration done, `gert migrate-expr` tool scaffolded.
- **Phase 2 sketch:** Don wrote a working GXL evaluator + parser + stdlib (commits `97ce48b..5c550c0`) in this repo. Stripped per scope correction; designated as cherry-pick source for the runtime migration (OQ-M2).
- **My first assignment (likely):** Phase A of the Runtime Migration in `ormasoftchile/gert` — pair with Don on PJVM + Clock + harness + DRIFT-DETECTION-001 (vector sync script + `make verify-vectors` CI gate). Then split: Don takes Streams B–D (critical path), I take Streams E and F in parallel.

## Key Decisions to Respect

Read `.squad/decisions.md` in full at first spawn. Highlights:
- **OQ-M1:** Vendored vectors + DRIFT-DETECTION-001 sync infrastructure (NEW Phase A scope item)
- **OQ-M2:** Cherry-pick sketch `97ce48b..5c550c0`, treat as unreviewed
- **OQ-M3:** Build tag `//go:build gxl` for the entire migration window
- **OQ-M4:** Hard cutover at Phase H, CHANGELOG only (pre-1.0)
- **OQ-M5:** Pair model — Ken (me) + Don, both in this squad, both working in gert repo
- **Single-squad directive (2026-06-05):** Never create a parallel squad in the runtime repo. Squad memory lives in `gert-private`.

## Learnings

### 2026-06-05 — gert repo conventions (from Phase A / DRIFT-DETECTION-001)

- **No Makefile existed** in `ormasoftchile/gert` — created from scratch. Pattern: `.PHONY` targets, `help` as default goal, `SHELL := /usr/bin/env bash`.
- **No `scripts/` directory existed** — created. Convention: executable bash scripts go here.
- **No `testdata/` directory existed** — created. `testdata/vectors/` is the first entry; this is the canonical path for vendored conformance vectors per OQ-M1.
- **CI framework:** GitHub Actions. Existing workflow at `.github/workflows/e2e.yml` uses `actions/checkout@v4`, `ubuntu-latest`, `actions/setup-go@v5` with `go-version: '1.21'`.
- **Temp dirs in Makefile:** `.gitignore` lists `tmp/` and `.temp/`. I used `.verify-vectors-tmp/` as verify scratch — always cleaned up by the target, no .gitignore entry needed.
- **Both repos in same org:** `ormasoftchile/gert` + `ormasoftchile/gert-private`. `GITHUB_TOKEN` likely covers cross-repo reads within the org — relevant for the CI Option A/B decision.
- **gert-private is read-only** from the runtime repo perspective — never commit to it from a gert worktree task.
- **Worktree pattern:** Phase A uses dedicated worktrees (`gert-phase-a-drift`, `gert-phase-a-pjvm`) branched from main. Don't `cd` to main repo for edits; the worktree IS the working directory.

## 2026-06-05 Phase 1 Complete

Phase 1 closed before my first session. TESS-AMBIG-3..6 resolved:
- **GXL-TYPE-005:** Boolean ordered comparison forbidden
- **GXL-TYPE-001:** Extended to forbid list/object equality (scalars+null only)
- **GXL-PATH-004:** Field access on non-object/null
- **Corpus:** 211 GXL vectors, 0 TBD (83 parse + 92 eval + 36 path)
- **Dogfood:** All 22 runbook fixtures clean (P1–P5 all zero)

Don + I cleared to start Phase A in `ormasoftchile/gert`. PJVM + Clock + harness + DRIFT-DETECTION-001 are first deliverables.

## 2026-06-05 — Phase A: DRIFT-DETECTION-001 Shipped (PR #8, First Shipment)

**Worktree:** `gert-phase-a-drift`  
**PR:** https://github.com/ormasoftchile/gert/pull/8 (draft, awaiting merge)  
**Status:** First ever shipment (pre-Phase-B)

**Shipped:**
- `scripts/sync-vectors.sh` — syncs tv-*.yaml + schema.json from gert-private; writes `VECTORS_SHA` (pinned to 3ce53431)
- `Makefile` — `sync-vectors` + `verify-vectors` targets (no prior Makefile in repo)
- `.github/workflows/verify-vectors.yml` — CI gate (PR + main + weekly cron)
- `testdata/vectors/README.md` — runbook (sync flow, drift detection, upgrade path)

**Option Chosen: A (best-effort).** GITHUB_TOKEN + sentinel-SHA fallback. Rationale: immediate within org; upgradeable to Option B (deploy-key) if token insufficient. Local `make verify-vectors` always fully enforced (dev gate during active dev).

**Test Loop Verified:** sync ✅ → verify-clean ✅ → hand-edit + verify-drift ✅ → restore + verify-clean ✅

**Env Contract:** `GERT_PRIVATE_PATH` (default: `../gert-private`)

**Learnings

## 2026-06-06 — Phase E + F Shipped (PR #13 + #14) — 56/56 PASS

**Status:** Two parallel shipments — GIS (Phase E, 15 vectors) + GCP (Phase F, 41 vectors) — both 100% conformance.

**PRs:** ormasoftchile/gert#13 (phase-e-gis) + #14 (phase-f-gcp), both draft, stacked on phase-a-pjvm (PR #9).

**My Track Record:**
- **Phase A (PR #8):** DRIFT-DETECTION-001 (sync-vectors.sh + verify-vectors CI gate) ✅
- **Phase E (PR #13):** GIS path resolver (internal/eval/gis/, 15/15 vectors) ✅
- **Phase F (PR #14):** GCP capture engine (internal/eval/gcp/, 41/41 vectors) ✅

Total: 3 shipments, 56 conformance vectors, zero defects.

### Key Design Pattern: Three Separate Packages, Single Shared Foundation

**Architecture:** gxl, gis, gcp are three independent evaluators, each in its own package:
- `internal/eval/gxl/` — variable bindings + operator precedence + stdlib
- `internal/eval/gis/` — optional chaining + null-as-miss semantics
- `internal/eval/gcp/` — four source prefixes + GDP traversal + §6 default policy

**Why separate?** Each has fundamentally different resolution semantics. Sharing would require abstraction layers that obfuscate the logic. Attempting to fork a "generic resolver" across all three creates coupling and refactor overhead without benefit. Keep each self-contained.

**Only shared layer:** `internal/eval/core` (PJVM value model, `FromYAML`, `core.Value` interface). This is the right level of abstraction — the common representation, not the traversal logic.

**Implication for future work:** Don't expect GIS/GCP to reuse GXL path logic. When Phase G integrates all three into the request/response flow, each stays independent, each gets its own harness runner entry point.

### Implementation Notes

**GIS (Phase E):**
- Two-level parser: outer template string layer, inner GIS expression tokenizer
- Lexer uses longest-match for `?.` and `?.[` to avoid ambiguity
- Eval implements null coercion (null-as-miss) + short-circuit semantics per optional-chaining design
- Falsy values (`""`, `false`, `0`, `[]`) correctly treated as **present** values, not misses
- Stdlib integration: str functions (toLower, toUpper, trim, etc.) receive optional results and propagate `""` on miss

**GCP (Phase F):**
- Four source prefixes (local/http/event/step) parsed uniformly but resolved differently per source
- HTTP/event headers resolve to soft-null (absent header → null, not error) per RFC 7230 case-insensitivity
- YAML timestamp handling: yaml.v3 tags dates as `!!timestamp`, but spec (OI-GCP-06) requires YAML 1.2 strings. Implemented local YAML converter (`gcpFromYAML`) to map timestamp tags back to strings
- §6 default policy: post-resolution check (`checkDefaultPolicy`) handles both GCP-DEFAULT-SUBTREE (object/array capture with default) and GCP-TYPE-001 (scalar type mismatch with default)
- GDP traversal implemented locally in `traverseGDP` — different semantics than GXL (GCP uses single error code for both non-array and out-of-bounds)

### Learnings for Phase G

1. **Separate-package pattern scales.** When Don integrates all three engines into the request/response flow, he won't be juggling a monolithic resolver or a tangle of conditional branches. Three independent entry points, three independent error code sets, three independent unit test suites.

2. **Build tag discipline holds.** All files `//go:build gxl`. The tag stays until Phase H cutover. Once old engine is deleted, tags come off and we run both systems side-by-side in CI to verify behavior parity.

3. **Error code pre-ratification critical.** All 56 vectors passed without corpus bugs or spec gaps. This happened because Barbara's Phase 1 arbitrations (TESS-AMBIG-3..6) locked down all error codes upfront. Phase G can proceed without returning to Spec.

4. **No handoff surprises.** Barbara: no action. Tess: no action. Each phase just ships. This is what pre-ratification looks like.

### Next: Phase G + H (Don)

Don takes integration (Phase G) and cutover (Phase H). My assignment (Phases E–F) closes once #9 merges and my PRs (#13/#14) pass final review. Wallclock on Phases A–F: ~6 days (2026-06-01 to 2026-06-06). Target was 10–15 days total with parallel streams; we're on track.

## 2026-08-10 — Enum-Constrained Runtime MVP: Independent Revision of Don's Rejected Gate

**Context:** Barbara rejected Don's implementation report (`don-enum-mvp-implementation-report.md`)
against the ratified `barbara-enum-mvp-implementation-gate.md` — five runtime blockers (R1–R5).
Don was locked out as the original author; I was brought in independently to re-resolve all five
from scratch, without reusing Don's rejected work. Full report:
`.squad/decisions/inbox/ken-enum-mvp-implementation-revision.md`.

**Outcome:** All five blockers resolved and regression-tested. R1 (ENUM-008 on every
caller-binding path incl. `--var`), R3 (enum metadata once in `plan/validated`, declared order,
C1-safe), R4 (ENUM-W001 surfaced non-fatally), and R5 (replay retains enum checks at the stable
boundary) were the more contained fixes. R2 — a faithful, data-driven conformance harness over all
58 `tv-enum.yaml` vectors — was the bulk of the work and the highest-value output: building a
harness that actually runs the real CLI against real fixtures (no dry-run shortcuts, no mocked
executors) surfaced **ten distinct genuine runtime bugs**, several of which were silently
defeating enum enforcement for entire fixture families (not edge cases). Final report: 58/58
vectors accounted for — 48 passed, 10 named/cited skips (5 pre-existing ticketed gaps, 5 newly
diagnosed corpus-fixture inconsistencies verified against the ratified architecture ruling), 0
failed, 0 silently dropped. Full `go build ./... && go test ./...`: clean across all 61 tested
packages.

### Key Learning: A Synthetic/Mocked Conformance Harness Hides Bugs a Real One Finds

The single biggest lesson from this session: earlier attempts (implicitly, Don's rejected report)
likely leaned on dry-run mode or a subset of vectors to claim conformance. Dry-run
(`internal/adapter.DryRunExecutorRegistry`) fakes success for every step — it never exercises real
tool-arg binding, real GCP capture resolution, or real ENUM-007/008/009 evaluation. Building the
harness to actually shell out to a real `gert.exe` against materialized fixtures (Git Bash
subprocess steps, real package/catalog resolution, real substitution) is what surfaced:
- a tool-arg output resolution bug (`executeSubstitution` using the wrong evaluator for GCP Capture
  Paths vs. `${...}` templates — two genuinely different grammars, `pkg/gcp/parser` vs.
  `internal/expr`),
- a **silent field-drop bug** in the catalog→planner tool-def conversion that defeated ENUM-006/007
  for every `requires:`/`toolRefs:`-resolved tool (the corpus's dominant shape) — the kind of bug
  that "passes" a shallow/synthetic conformance run because the checks it defeats never get a
  chance to even fire, so there's no crash, just silent under-enforcement,
- a missing plan-time check (S2 output defaults) that AR-ENUM-6 explicitly required,
- two separate "capture attempted after a failed result masks the real error" bugs (one in the
  executor, one — more consequential — in the engine's own generic post-execution capture path,
  which aborted the *entire run* via a different code path and hid the original failure behind an
  unrelated one),
- a schema/struct drift bug (`expand:` supported in Go structs and a real feature `pkg/expand`, but
  never added to the JSON Schema used for structural validation — so any runbook using it was
  rejected outright),
- an `imports:` alias-resolution gap (a documented include-aliasing idiom that the include-walking
  code never actually implemented — treated the alias name as a literal file path),
- and a genuine validation-ordering defect (`cmd/gert/run.go` short-circuited on B1's substitution
  check before `Plan()` ever ran, so a real finding on one declaration could mask a real,
  unrelated finding on another).

**Pattern for future conformance work:** when asked to build/validate a "faithful" harness against
a frozen vector corpus, resist any temptation to special-case or mock around real CLI execution —
that convenience is exactly what hides silent, high-blast-radius bugs. Also: not every vector
"failure" is a runtime defect. Five of the ten skips this session were genuine corpus-fixture
authoring inconsistencies (verified by re-deriving the expected behavior directly from the ratified
architecture ruling's unconditional rules, e.g. AR-ENUM-6's default-must-be-member and AR-ENUM-8's
"no variance" PKG-013 rule) — the discipline of checking the *ruling*, not just the vector's own
prose `note:`, before "fixing" the runtime to match a vector is what separates a real fix from
weakening a correctly-enforced rule to force a pass.



## Learnings: 2026-08-15 — Dynamic Include Stream 1 (Schema + Parser + Errkit)

### errkit structure
- **Sentinel pattern:** every code gets a module-level `var Err<Name> = &Error{code, class}`. Class is derived by `ClassForCode(code)` at call site via `Wrap()`/`New()`, so the sentinel var's class must match `ClassForCode`'s output — enforced by `TestSentinelsExposeClassAndCode`.
- **Warning classes use `-W` suffix** (e.g. `PKG-W`, `DINC-W`). `IsWarning` was hard-coded to `class == "PKG-W"` — I generalized it to `strings.HasSuffix(class, "-W")` to cover all warning classes.
- **Three registries to update:** `codeOrder` (for `Codes()`), `sentinels` map (for `Sentinel(code)`), `classSentinels` map (for `ClassSentinel(class)`). Miss one and the existing `TestSentinelsExposeClassAndCode` / `TestDeclaredClassesCovered` tests fail.

### JSON Schema oneOf with additionalProperties
- The `oneOf` + `additionalProperties: false` pattern correctly enforces mutual exclusion between static (`runbook`) and dynamic (`runbook_ref`) include arms.
- Each arm's `additionalProperties: false` rejects the OTHER arm's fields, so both arms fail when both fields are present — `oneOf` fails cleanly.
- `expand` intentionally absent from the dynamic arm per Barbara's contract.

### parser/validate_semantic.go conventions
- Step-type-specific semantic validation lives in a `switch s.Type` block inside `validateStep`. Adding a new `case schema.StepTypeInclude` there is the correct extension point.
- The JSON Schema structural pass runs first and rejects most structural errors; the semantic validator adds belt-and-suspenders cross-field checks with developer-friendly error messages.
- `verr(code, field, message)` is the internal constructor for `*ValidationError`.

### ValidateRenderedRef placement decision
- Belongs in `pkg/pkgcatalog/` (next to catalog lookup methods) not in `internal/parser/` — it is catalog-reference syntax, not runbook-document syntax.
- Stream 3 (executor) can import it from `pkg/pkgcatalog` without any new import cycles.

### DINC-007 class inconsistency in Barbara's contract
- Barbara's §11 table says DINC-007 is a "Warning" but her explicit Go code block says `class: "DINC"`.
- Implemented as the code block specifies to avoid breaking `TestSentinelsExposeClassAndCode`.
- Documented the discrepancy in `ken-dynamic-include-stream1.md` for Barbara's arbitration.

### 2026-08-15 Addendum — Barbara's B-6/B-14 Delta

- **DINC-013** added: catalog lookup succeeded but file vanished from disk. Distinct from DINC-002 (identity not in catalog). `on_not_found: continue` does NOT suppress DINC-013 — it is an infrastructure failure, not a missing identity.
- **B-14 (expand):** The dynamic arm's `additionalProperties: false` already excluded `expand` without an explicit schema change. Belt-and-suspenders semantic check added to catch it on unmarshal-only paths. Comment-only edit to `pkg/schema/steps.go` per scope constraints.
- **B-5 (on_not_found scope):** Only DINC-002 is suppressible by `on_not_found: continue`. DINC-003 (ambiguous), DINC-013 (file missing), and all others always fatal — noted for Stream 3 executor implementation.

## Learnings: 2026-08-15 — DEF-003 Governance Enforcement

### Root cause pattern: composed-but-not-enforced
`ComposeGovernance` in `dynamic_resolver.go` correctly computes the effective governance — but the sub-engine created by `runSubSteps` in `pkg/run/run.go` has no `GovernanceEvaluator`. The context value (`dynGovKey`) was always there; nothing read it for enforcement. This is a recurring "the composition is right but the enforcement is missing" pattern — watch for it in other composite executors (branch, iterate, parallel).

### Static includes have the same gap
The static include path (eager + lazy) runs child steps through the same `runSubSteps` sub-engine without a `GovernanceEvaluator`. Governance is not enforced for any child steps today. This was reported (not fixed silently) in `ken-dynamic-include-governance.md`.

### Where to enforce without touching pkg/run
`pkg/run/run.go` was out of scope. The fix stays in `internal/executor/`:
1. **include.go**: enforce `require_approval` before calling the runner; build a compiled `governance.GovernancePolicy` from `effectiveGov` and store it in `childCtx` via `withGovPolicy`
2. **cli.go**: read `govPolicyFromCtx(ctx)` and call `pol.CheckCommand(command)` before `platform.Exec`
This is a "push policy into context, check at leaf executor" pattern — works without touching the sub-engine or the runner.

### Conversion: EffectiveGovernancePayload → schema.GovernanceConfig
`internal/governance.BuildPolicy` takes `*schema.GovernanceConfig`. `EffectiveGovernancePayload` (trace pkg) is the composed result. Added `effectiveGovToSchemaConfig` in `dynamic_resolver.go` to project the common fields. `Capabilities` is NOT in `GovernanceConfig` — this blocks TV-DYN-GOV-015.

### ApprovalGate wiring
Added `approvalGate` field to `IncludeExecutor` with a fluent `WithApprovalGate` setter (avoids changing `newIncludeExecutorFull` signature and the 20 existing test call sites). `NewDefaultRegistry` calls `.WithApprovalGate(cfg.ApprovalGate)` — the gate is already in `RegistryConfig`.

### TV-DYN-GOV-005 New Failure (Tess's Concurrent Work)

Tess removed TV-DYN-GOV-005 from the conformance skip list. This vector tests `run: "rm -rf ..."` with `deny_commands: ["rm -rf *"]`. My deny_commands enforcement checks `argv[0]` (the shell binary, e.g. `sh`), not script content — documented limitation in this file. TV-DYN-GOV-005 requires governance enforcement on `run:` script content, which is a separate gap. This failure is Tess's concurrent change, not a DEF-005 regression.

## Learnings: 2026-08-15 — DEF-005 Empty Ref / DINC-002 Fix

**Problem pattern**: `hasDynamic := inc.RunbookRef != ""` is the wrong test when the user explicitly writes `runbook_ref: ""`. An empty string and an absent field are indistinguishable in a Go struct without pointer types. The fix uses companion fields (`resolve_from`, `on_not_found`) as dynamic-intent signals.

**`ValidateRenderedRef` was orphaned**: Written in Stream 1 for Stream 3 to call, but `executeDynamic` (Stream 3's executor) never wired it in. Always check that shared validators are actually called at the right call site.

**conformance test unskipping race**: Tess can unskip vectors concurrently. When full-suite tests show a new failure in `internal/conformance/`, check whether the vector was recently unskipped before investigating whether my changes caused it.
`CheckCommand` sees `argv[0]`. For `run:` specs, `argv[0]` is the shell (`sh`, `pwsh`). A deny pattern like `"rm -rf *"` doesn't match `"sh"`. TV-DYN-GOV-005 uses `run:` and will not be satisfied by this enforcement. Documented in governance findings file.

### TV-DYN-GOV-015 conflict
`schema.GovernanceConfig` has no `Capabilities` field — capabilities declared in child YAML are silently dropped. `ComposeGovernance` already hardcodes `nil` for child capabilities. The vector expects `DYN-015` (fatal) but Barbara's §6.4 says non-fatal warning. Both the schema gap and the Barbara/vector conflict are documented in `ken-dynamic-include-governance.md`.

## Learnings: 2026-08-15 — `run:` Script Governance Fix

**`path.Match`'s `*` does not cross `/`**: This is the core gotcha for deny-command patterns. Operators write `"rm -rf *"` expecting it to match anything starting with `"rm -rf "`. `path.Match` returns false for `"rm -rf /tmp/scratch"` because `*` stops at the first `/`. The fix: `scriptMatchPattern` tries `path.Match` first, then falls back to `strings.HasPrefix(script, prefix)` when the pattern ends in `*`. This is sound for command patterns — operators don't use `/`-sensitive path patterns in deny lists.

**Optional interface pattern for governance extension**: The public `GovernancePolicy` interface (`pkg/governance/policy.go`) must not be changed — other packages implement it and breaking that interface would cascade. Adding `CheckScript` only to the concrete `*policy` struct (in `internal/governance/builder.go`) and using a local `ScriptChecker` interface in `cli.go` for a type-assertion is the right way to extend without breaking. This pattern is reusable whenever you need to add optional capabilities to a sealed interface.

**Incomplete DEF-003 fix**: DEF-003 established enforcement for `command:` specs only. `run:` scripts were visible in the same `CLIExecutor.Execute` function but overlooked. The lesson: when adding a governance check at a seam, enumerate ALL execution paths through that seam, not just the one the failing test exercises.

**DEF-006 is a distinct root cause**: Even with correct `CheckScript` for `run:` scripts, TV-DYN-GOV-005 remains SKIPPED in the conformance harness because the entry runbook's static governance block is never seeded into `dynGov` context. The executor can only enforce a policy it was given; if the planner/engine never calls `withDynGov` with the top-level runbook's governance, `dynGovFromCtx` returns nil and `ComposeGovernance` gets an empty parent. This is a seeding gap, not an enforcement gap.

**TV-DYN-COMPAT-001 failure is pre-existing FlowNode schema gap**: `FlowNode.oneOf` in `schemas/runbook.schema.json` only lists `step`, `iterate`, `parallel`. Tess added a vector using `branch:` at the flow level per B-19 ruling, but the FlowNode schema was never updated. This has nothing to do with IncludeConfig changes.


---

## Session: Dynamic Runbook Includes — Phase 4 Wrap-up (2026-08-15)

**Agents:** Barbara (arch), Tess (test), Ken (stream1), Don (streams2-3), David (stream4)

**Outcome:** Feature complete, approved, ready for merge (APPROVE WITH CONDITIONS, all conditions satisfied)

**Key achievements:**
- 5 defects found and fixed (DEF-001..006 via Tess real-CLI exercise + implementation streams)
- 21 architecture rulings (B-1..B-21) issued and enforced
- 3 deferred items formally captured (B-18, B-19/B-20, B-21)
- 26/29 conformance vectors pass; 3 skips permanent with ruling
- Cross-repo coordination: design repo (spec authority) ↔ runtime repo (implementation)

**Coordination patterns that worked:**
- Conformance corpus authored before implementation as specification
- Real CLI exercise (not hand-reading) revealed every gap
- Plausible descriptions can be materially wrong; rely on code review
- Formal deferral of gaps better than hidden work

**Status:** This session's work fully merged into .squad/decisions.md. Five orchestration logs recorded. Session log captures defining lesson: real verification beats narrative.


