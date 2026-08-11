# Revision Response — Client Enum Parity Gate Review (B-1..B-6 / F-1..F-11)

**From:** David (Integration Engineer, independent revision owner)
**Date:** 2026-08-11T09:45:19-07:00
**Requested by:** Cristián Ormazábal Ortega
**Responds to:** `.squad/decisions/inbox/barbara-client-enum-parity-gate-review.md` (REJECTED,
B-1..B-6, F-1..F-11)
**Governs:** `gert`, `gert-tui`, `gert-vscode` (paths per environment: `C:\One\OpenSource\gert`,
`gert-tui`, `gert-vscode`)
**Constraint honored throughout:** Don, Ken, and Leslie (original authors of the rejected work) were
not consulted, messaged, or asked for input at any point in this revision. No branch was reverted,
staged, committed, or switched in any repo. Every pre-existing dirty/untracked file in all four
repos (including protected runtime and in-flight TUI/VS Code edits) was left exactly as found;
only additive changes were made on top.

---

## 1. B/F resolution matrix

| Finding | Fix(es) | Status | Repo(s) |
|---|---|---|---|
| **B-1** — DTO never reaches any client (renderer drops `Inputs`) | F-1, F-2 | **Resolved** | `gert` |
| **B-2** — `InteractionField.Enum` has no live producer | F-3 | **Resolved (by removal — see §3)** | `gert` |
| **B-3** — no declaration-driven input form in the web GUI | F-4 | **Resolved** | `gert` |
| **B-4** — TUI enum selector is a disconnected helper | F-6 | **Resolved** | `gert-tui` |
| **B-5** — `StartWithWarnings` exists but unused by the TUI | F-5 | **Resolved** | `gert-tui` |
| **B-6** — no client-parity matrix artifact; CE-T-01..04 not demonstrated per-client | F-9 | **Resolved (with two documented gaps — see §5)** | `gert-private` |
| F-7 — VS Code test against the real graphjson binary; stale comments | F-7 | **Resolved** | `gert-vscode` |
| F-8 — `enumui` doc comment update | F-8 | **Resolved** | `gert-tui` |
| F-10 — normative comment on `buildInputDecls`' name-sort | F-10 | **Already satisfied** (pre-existing comment found correct on inspection; no change needed) | `gert` |
| F-11 — confirm pre-existing `gert-tui` failures are environmental | F-11 | **Confirmed environmental, documented — see §4** | `gert-tui` |

---

## 2. Files and tests changed, per repo

### `gert`

| File | Change |
|---|---|
| `pkg/preview/render/graphjson/graphjson.go` | Added `Inputs []graphdoc.InputDecl` field to `Document`; copied `doc.Inputs` in `RenderWithState`; updated package doc comment. (F-1) |
| `pkg/preview/render/graphjson/graphjson_test.go` | Added `TestRender_Inputs_DeclaredOrderSurvives` + helpers. (F-2) |
| `cmd/gert/preview_graphjson_inputs_test.go` (new) | `TestPreview_GraphJSON_CarriesDeclaredInputs`, `TestPreview_GraphJSON_RedactedInputs_NoMemberLeak` — exercise the real `gert preview --format graphjson` binary path. (F-2, CE-D-01/02/03) |
| `internal/serve/broker.go` | Removed the never-populated `Enum`/`EnumRedacted`/`EnumMemberCount` fields from `InteractionField`, the `NewEnumInteractionField` constructor, and their propagation in `toFields`. (F-3) |
| `internal/serve/broker_enum_test.go` | **Deleted** — tested only the removed symbols. (F-3) |
| `pkg/input/prompt.go` | Removed the same dead `Enum*` fields from `FormField`. (F-3) |
| `internal/serve/static/preview.html` | Added `formValues` state, `buildFormInputs()` (omits the `""` "unset" sentinel), `InputsForm` React component (closed `<select>` in declared order for non-redacted enums, free text for `enumRedacted`/unconstrained inputs, no auto-select), wired into `startRun`; raw-JSON box retained as the outright-wins fallback. (F-4) |
| `internal/serve/preview_html_inputs_form_test.go` (new) | `TestPreviewHTML_DeclarationDrivenInputsForm` — static assertions the form/selector/redaction logic is present and wired. (F-4) |
| `internal/serve/preview_test.go` | Added `TestPreviewDocument_GraphJSON_CarriesDeclaredInputs` (real HTTP handler test, same endpoint the GUI consumes) + `writeTestFile` helper + `os` import. (F-4) |

Full `go build ./...` and `go test ./...` clean (all packages green) after these changes.

### `gert-tui`

| File | Change |
|---|---|
| `internal/session/live.go` | Removed the permanently-false `engineWarning` no-op interface; added `warnings []parserpkg.ParseWarning` field and `SetParseWarnings` setter; rewrote `emitWarnings()` to translate real parser warnings into `WarningMsg`. (F-5) |
| `cmd/gert-tui/main.go` | Switched `run.Start` → `run.StartWithWarnings`; wired `sess.SetParseWarnings(res.Warnings)` before `sess.Start(ctx)`. Also wired F-6's pre-run-start input collection (see below). (F-5, F-6) |
| `internal/tui/app.go` | Added `warnings []session.WarningMsg` field; `case session.WarningMsg` in `Update()` (never touches `m.status`/`m.err` — nonfatal per AR-ENUM-12); `renderStatusBar()` prepends a warning line when present. (F-5) |
| `internal/tui/styles.go` | Added `WarningStyle` (yellow, distinct from the red failure style). (F-5) |
| `internal/session/live_test.go` | Rewritten: `fakeHandle` (matching the real `engine.RunHandle` shape, no `Warnings()` method); `TestLiveSession_EmitWarnings_TranslatesCodedWarnings`, `TestLiveSession_EmitWarnings_NoOpWithoutWarnings`. (F-5) |
| `internal/enumui/enumui.go` | Added `BuildFieldFromDecl(d graphdoc.InputDecl) session.FormField` — the same rendering rules as `BuildEnumField`, but driven directly from the now-real preview-document DTO (`pkg/preview/graphdoc.InputDecl`) rather than a full `engine.ValidatedPlan`. Updated the package doc comment to state the DTO now exists (no more "does not exist yet"). (F-6, F-8) |
| `cmd/gert-tui/input_prompt.go` (new) | `collectDeclaredEnumInputs` — parses the runbook via the real `pkg/preview.BuildDocument` before `run.StartWithWarnings` is called, derives the missing enum-constrained input fields (`declaredEnumInputFields`), and hosts them in a standalone `tea.Program` wrapping the **existing** `interaction.MultiFieldFormModel` widget (`inputFormWrapper`/`runInputForm`) — the same widget collector-step forms already use inside `TUIApp`. Fail-open on any parse/hosting error or embedded-runbook path (no filesystem source). (F-6) |
| `cmd/gert-tui/input_prompt_test.go` (new) | `TestDeclaredEnumInputFields_SelectorInDeclaredOrder`, `TestDeclaredEnumInputFields_SkipsAlreadyBoundVar`, `TestDeclaredEnumInputFields_NoEnumInputs_NoPrompt`, `TestCollectDeclaredEnumInputs_NoRunbookPath_FailsOpen`, and — the app-level evidence F-6 explicitly requires — `TestAppLevel_EnumFieldRendersAndSubmitsVerbatim`, which drives the real `interaction.MultiFieldFormModel` built from a real parsed declaration, asserts the rendered `View()` shows both declared members in order with nothing preselected, then simulates navigation + Enter and asserts the resulting `interaction.DoneMsg` carries the selected member back byte-verbatim. (F-6) |

`go build ./...` and `go vet ./...` clean. `go test ./...`: `cmd/gert-tui`, `internal/session`,
`internal/enumui`, `internal/interaction` all pass (including the new tests above);
`internal/tui`/`internal/e2e` have the pre-existing environmental failures documented in §4.

### `gert-vscode`

| File | Change |
|---|---|
| `src/enumInputs.ts` | Updated the module-level doc comment and `extractInputDecls`'s doc comment to state the DTO now ships from `gert preview --format graphjson` (removed the stale "does not exist yet" language), while keeping the absence-tolerant behaviour unchanged (forward compatibility, AR-CE-7). (F-7) |
| `src/extension.ts` | Updated `validateInputs`'s doc comment to match (DTO is real, not pending). No behavioural change — `validateInputs` already called the real CLI and `extractInputDecls` correctly. (F-7) |
| `test/enumRuntimeRegression.test.js` | Added `CE-D-01/CE-D-02 regression: extractInputDecls reads the real gert preview --format graphjson output, declared order preserved` — runs the actual `gert preview --format graphjson` binary against `test/fixtures/enum.runbook.yaml` and asserts the returned `env_name` declaration carries `enum: ["prod","staging"]` in file order, `required: true`, and no `enumRedacted`. (F-7) |

`npm run compile` clean. `npm test`: 26/27 pass (see §5 for the one pre-existing failure, unrelated
to this fix and not modified).

### `gert-private`

| File | Change |
|---|---|
| `design/gert/conformance/client-parity-matrix.md` (new) | The F-9 client-parity matrix: one row per CE-* ID from AR-CE-9 §9, citing the owning test's repo + test name, plus an explicit "Remaining limitations" section for the handful of CE-* rows this revision could not fully close (§5). Does not modify `tv-enum.yaml` or any other frozen corpus file. |
| `.squad/decisions/inbox/david-client-enum-parity-revision.md` (this file) | Revision record. |
| `.squad/agents/david/history.md` | Appended a session entry (see below). |

---

## 3. Decision rationale: F-3 (deletion instead of a new producer)

B-2 asked either for a live producer of `InteractionField.Enum`/`FormField.Enum*`, or — Barbara's own
explicitly offered alternative — deletion: *"or delete `NewEnumInteractionField` and the unused
`FormField.Enum*` fields... Do not leave a wire field no code path populates."*

I investigated a live producer first. The only type that ever constructs `input.FormField` is
`internal/executor/collector.go` (collector steps), which have no schema linkage to top-level
runbook `inputs.<name>` enum declarations — collector fields are not one of the S1-S4 enum sites
(tool-action `args`/`outputs`, runbook `inputs`/`outputs`) defined in the archived
`barbara-enum-constraint-mvp-architecture-ruling-archived.md`. Building a real link would require
either a new collector-to-input schema binding (forbidden: no schema forks) or wiring
`Input.From: prompt` interactive sourcing — explicitly out of scope per this task's own instructions
("no `Input.From`/root-output work") and per the ruling's own ticketing (`T-ENUM-FROM-SOURCING`,
tracked separately). I therefore took Barbara's sanctioned alternative: deleted the dead fields and
constructor rather than inventing an out-of-scope producer. This is a deliberate, accepted scope
boundary, not an oversight.

---

## 4. F-11: confirmation of pre-existing `gert-tui` failures

Both flagged failure clusters were independently re-executed and confirmed environmental/pre-existing,
unrelated to enum-parity work:

- **`internal/tui.TestPreviewMode_TogglesAndShowsTree` / `TestPreviewMode_ShowsRegions`** — both
  hardcode absolute macOS paths (`/Volumes/Projects/gert/examples/...`,
  `/Volumes/Projects/gert-domain-dri/...`) that do not exist on this Windows development machine.
  `git status --porcelain` on `internal/tui/preview_test.go` shows **no modification** — it is fully
  committed, untouched by this revision or any dirty work. This is a pre-existing
  environment-portability defect in the test itself.
- **`internal/e2e.TestE2E_CollectHealth` / `TestE2E_CollectHealthParallel`** — fail because the
  `collect-health` example runbook pings/checks real hostnames (`api.east.contoso.com`, literal IPv4
  and IPv6 literals) that are not reachable/resolvable in this environment (`check_host`/`ping_host`
  steps report `degraded`/`failed`, cascading to the accumulate/report assertions). This is a genuine
  network/DNS dependency, not a code defect; the one file in this cluster that shows as modified
  (`internal/e2e/tui_e2e_test.go`) only has a 7-line **addition** to an unrelated test
  (`TestE2E_SimpleHealthCheck`, pre-existing dirty work, left untouched) — the `CollectHealth` tests
  themselves are not touched by any dirty diff.

Neither failure is caused by, or related to, any F-1..F-10 change in this revision. Filed here per
F-11's "confirm rather than assume" instruction rather than silently assumed.

---

## 5. Remaining accepted limitations

1. **CE-V-04 (`gert-vscode`) real-CLI test failure, not fixed in this revision.** The pre-existing,
   untracked test `test/enumRuntimeRegression.test.js`'s `"CE-V-04/D-3 regression"` case expects the
   real `gert dry-run` CLI's rejection message to lead with `ENUM-008:` (matching Barbara's own
   "what passes" quote in the rejection: `Error: ENUM-008: input "env_name": …`). The currently
   running binary instead emits `input "env_name": ENUM-008: value is not a declared enum member` —
   the code and the input-name prefix have swapped order relative to when Barbara wrote her review.
   This formatting is owned by `pkg/schema` (dirty/untracked files such as `pkg/schema/enum.go`,
   authored by a locked-out original author and part of the protected, concurrent, in-flight work this
   task instructs me not to touch, revert, or resolve unilaterally). Per this task's explicit
   instruction ("if direct conflict makes a required live seam impossible, report exact conflict"),
   I report this exact conflict rather than editing another author's protected in-flight file or the
   test's own expectation. This is not one of the B-1..B-6/F-1..F-11 items assigned to this revision.
2. **CE-R-06 (40-member enum) and CE-U-05 (non-ASCII members) on the web GUI, plus CE-R-02/CE-R-03 GUI
   default/unset behaviour** have no executed DOM-level test — `internal/serve/static/preview.html`
   has no headless-browser harness in this repo, only static source assertions
   (`preview_html_inputs_form_test.go`). The rendering code itself does not truncate or transform
   member values (verified by manual source inspection), but this is unproven by an executed test.
3. **CE-C-03 (vendored `tv-enum.yaml` SHA-256 parity)** has no automated equality check between
   `internal/conformance/enumdata/tv-enum.yaml` (gert) and `design/gert/conformance/tv-enum.yaml`
   (gert-private) — synced manually per `internal/conformance/enumdata/README.md`. Not named in
   B-1..B-6/F-1..F-11; flagged for visibility per F-9's "one row per CE-* ID" requirement, not
   fixed in this revision (out of assigned scope).

None of these three gaps block re-gate on B-1..B-6/F-1..F-11 as literally specified; they are
flagged so Barbara/Cristián can decide whether any warrants a follow-up ticket.

---

## 6. Protected-file verification

`git status --porcelain` was captured in all four repos before and after this revision's edits. No
file that was already modified/untracked at session start was reverted, staged, committed, or had
its working-tree content replaced by this revision — every touched file in the tables above is
either a new file this revision added, or an existing file where this revision's edits are additive
on top of whatever state the file was already in (verified via targeted `git diff`/`git status` per
file during the session). No `git checkout`, `git reset`, `git stash`, `git commit`, or branch
switch was executed in any repo at any point in this revision. Dirty file counts at the end of the
session: `gert` 96 (was already dirty at session start, plus this revision's additive changes),
`gert-tui` 19 (unchanged file count from adding new files + the additive edits above),
`gert-vscode` 10, `gert-private` 38 (net new: this record, David's history entry, and the F-9 matrix
file).

---

## 7. Executed verification summary

- `gert`: `go build ./...` — clean. `go test ./...` — **all packages green**.
- `gert-tui`: `go build ./...`, `go vet ./...` — clean. `go test ./...` — `cmd/gert-tui`,
  `internal/session`, `internal/enumui`, `internal/interaction` green (including all new F-5/F-6
  tests); `internal/tui`/`internal/e2e` fail only on the confirmed-environmental cases in §4.
- `gert-vscode`: `npm run compile` — clean. `npm test` — 26/27 green; the one failure is the
  pre-existing, out-of-scope CE-V-04 conflict documented in §5(1).
- Real-binary evidence executed and observed directly (not merely read): `gert preview --format
  graphjson` against an enum fixture emits `inputs[]` with declared-order `enum`, `enumRedacted`,
  `enumMemberCount`; `gert-tui`'s new pre-run-start form (driven at the app level in
  `TestAppLevel_EnumFieldRendersAndSubmitsVerbatim`) renders both declared members with nothing
  preselected and submits the selected member byte-verbatim; `gert-vscode`'s `extractInputDecls`
  reads the real CLI's `graphjson` output and recovers the declared enum in order.
