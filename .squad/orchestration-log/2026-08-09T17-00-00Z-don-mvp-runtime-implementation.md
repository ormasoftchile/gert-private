# Don — Tool Packages MVP: deferred runtime surfaces implemented

Session continuing prior Don work in `gert` worktree (non-worktree mode). Implemented the
deferred runtime surfaces against Tess's `design/gert/conformance/tv-pkg-resolve.yaml` and
the ratified spec sections (06/07/12/13). No commits made in `gert`; protected files
(`examples/simple-health-check/simple-health-check.runbook.yaml`,
`internal/tool/native.go`, `internal/tool/native_test.go`,
`examples/simple-health-check/examples.code-workspace`) left byte-for-byte untouched.

## Environment note
Ken was actively editing `cmd/gert/run.go`, `internal/adapter/wire.go`,
`internal/tool/overlay_registry.go`, `cmd/gert/packagemap.go` concurrently in the shared
`gert` worktree during this session (fresh timestamps, non-Don diffs). A transient
`undefined: newOverlayToolRegistry` build failure and one `internal/adapter` test failure
were confirmed (via `git status`/timestamps) to be his own in-progress WIP, not caused by
Don's changes; both had resolved by the time of the final full-repo build/test pass
(`go build ./...` and `go test ./...` both green, one pre-existing flaky
`internal/serve.TestSSE_FilterByRunID` timing test aside — passes in isolation, file
untouched by Don).

## What was implemented

1. **Substitution planning** — new `pkg/pkgsubst` package (`Plan` function): resolves
   `execute.kind: runbook` via the exact `execute.path` kind, enforces exact
   input/output signature match (PKG-013), `from: context`-only substitute inputs
   (PKG-026), unproducible-output detection via empty `value:` (PKG-027), cycle
   detection via `(package, tool, action)` frame stack (PKG-015), max depth 4 (PKG-028),
   and full governance composition per §12 (OR/union/union/intersect) with PKG-014
   widening detection on `allow_commands`. 17 unit tests directly derived from
   TV-PKG-SUBST-001–011.

2. **Trace event scaffolding** — added `package/resolved`, `catalog/frozen`,
   `tool/substituted`, `governance/packageDriftAccepted`, `replay/packageDrift`
   `EventKind` constants plus exact-shape payload structs to `pkg/trace/event.go`, per
   §07. **Not wired into a live emitter** — see blockers below.

3. **Resume/replay drift decision** — new `pkg/pkgdrift` package (`Evaluate` function):
   pure function over recorded-vs-actual digest pairs + mode (resume/replay) + override
   flag, implementing the exact Package Resumption Contract (§13): resume = hard PKG-009
   refusal unless `allowDrift`, which then requires an auditable drift-accepted event
   (both digests + operator); replay = always advisory (`replay/packageDrift`, non-fatal).
   5 unit tests. **Not wired into `internal/resume/resume.go`** — see blockers below.

4. **Exact path-kind bases** — `pkg/pkgpath/kinds.go` (already begun before this log
   entry) plus new `pkg/pkgpath/kinds_test.go` (9 new tests) directly exercising
   TV-PKG-PATH-001/003/004/005/006 and the toolRefs package-internal-override rule.
   **Corpus/spec discrepancy found and NOT silently resolved**: TV-PKG-PATH-002 expects
   `PKG-007` for a project-scope `requires[].path` escaping the workspace root, but
   06-tool-runtime.tex §Resolution base per path kind states workspace-level kinds
   (including project-scope `requires[].path`) "MAY legitimately resolve outside
   [the workspace] ... never rejected merely for escaping the workspace" (only
   PKG-W003 external-report). Implementation follows the normative prose (tests assert
   external=true, no error) rather than the vector, since the task instructs preserving
   normative behavior; **flagging this vector for Tess/Barbara to reconcile** — either
   the vector's `expected.error.code` is wrong, or there's an unstated additional rule
   (e.g. "escape leaves the conformance sandbox root entirely" is distinct from "escapes
   the workspace root") that the prose doesn't yet capture.

5. **`ToolDef.Actions` reconciliation** — done in a prior sub-session already reflected in
   `pkg/schema/tool.go` (list-form canonical / map-form legacy-compat dispatch,
   `ActionsMapForm` flag, `pkgcatalog` PKG-004 rejection of map-form for package-exported
   tool files). This session additionally added `ToolGovernance.AllowCommands`
   (`allow-commands`, hyphenated, matching the corpus's tool-level governance block) and
   a `GovernanceConfig.UnmarshalYAML` on the runbook-level type that accepts both the
   pre-existing snake_case field spellings (`allow_commands` etc., used by
   `examples/multi-region-rollout`) and the corpus's hyphenated spellings
   (`allow-commands` etc., used in substitute-runbook governance blocks) without breaking
   either.

6. **Tests**: `pkg/pkgsubst` (17), `pkg/pkgdrift` (5), `pkg/pkgpath/kinds_test.go` (9,
   new in this pass), plus the `pkg/pkgcatalog`/`internal/adapter` fixture updates needed
   to keep `TestBuildPackageCatalogAndResolveToolRefsViaCatalog` and
   `catalog_test.go`'s `BindFile` call sites correct against the `packageRoot` parameter
   and PKG-004 map-form rejection added by the earlier sub-session.

## Explicitly identified blockers (not implemented; exact reason)

- **Live engine wiring for `execute.kind: runbook` execution.** `pkgsubst.Plan` is a
  complete, tested *planning* function, but nothing in `internal/engine`/`internal/executor`
  calls it or executes a planned substitute's nested steps as part of a real run. This
  requires touching the live step-execution loop (tool-step dispatch) and the GCP
  `outputs.<name>` capture-root plumbing end-to-end, which is squarely in the area Ken is
  concurrently, actively editing (`cmd/gert/run.go`, `internal/adapter/wire.go`,
  `internal/tool/overlay_registry.go`) this same session. Wiring it now risks a direct
  edit collision with his in-flight work and was explicitly out of the "don't alter Ken's
  concurrent CLI/config files" instruction's spirit even where not the literal same
  lines.
- **`pkg/trace` event emission wiring.** The event kinds/payloads exist and are
  unit-testable in isolation, but no call site in the run loop actually emits
  `package/resolved`/`catalog/frozen`/`tool/substituted`/`governance/packageDriftAccepted`/
  `replay/packageDrift` yet — that requires a trace-writer handle threaded through
  `pkgcatalog.Build`/`pkgsubst.Plan` call sites inside the live run loop, which doesn't
  yet exist as a call site (same reason as above).
- **`pkgdrift.Evaluate` wiring into `internal/resume/resume.go`.** The run manifest
  schema has no package/catalog digest fields yet to compare against on resume; adding
  them is a manifest-schema change with its own backward-compatibility concerns for
  already-written manifests, judged out of scope for this pass.
- **Full YAML conformance-harness integration of `tv-pkg-resolve.yaml`.** The generic
  `internal/conformance` GXL/GIS/GCP harness's `Vector`/`Expected` types lack
  `catalog`/`error.code`/`warnings` fields the PKG vectors use; direct Go unit tests
  derived from representative vectors were written instead (see above), consistent with
  the effort-budget scope-management decision already logged in the prior session.

## Validation
`go build ./...` — clean. `go test ./...` — all green except one pre-existing,
Don-untouched, timing-flaky `internal/serve.TestSSE_FilterByRunID` (passes in isolation).
No commits, no staging, no branch changes made.
