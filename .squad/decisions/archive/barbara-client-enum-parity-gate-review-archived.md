# Gate Review — Client Enum Parity (AR-CE-1..10) — **REJECTED**

**From:** Barbara (Lead / Architect), formal reviewer
**Date:** 2026-08-11T10:34-07:00
**Requested by:** Cristián Ormazábal Ortega
**Governs:** `C:\One\OpenSource\gert`, `C:\One\OpenSource\gert-tui`, `C:\One\OpenSource\gert-vscode`
**Gate authority:** `barbara-client-enum-compatibility-ruling.md` (AR-CE-1..10, RATIFIED),
upstream `AR-ENUM-1..15` + C1/C2/C3 (not reopened).
**Method:** read the ruling, all three working trees' full diffs, the new tests, and **executed**
the real binaries (`gert preview --format graphjson`, `gert dry-run`, `gert-tui -var`,
`go test`, `npm test`). No product artifact modified; all dirty files preserved.

---

## Verdict: **REJECTED**

Real progress landed — Don's runtime work is largely correct and the coded-error path is a genuine
structural fix — but **the feature does not work end-to-end on any GUI/VS Code surface, and the TUI
selector remains a disconnected helper exactly as Ken reported.** Three of the five gate conditions
fail on executed evidence, not on opinion.

---

## 1. What passes

| Area | Evidence |
|---|---|
| **AR-CE-1 — no client schema fork** | `gert-tui` still has no YAML decoder / `Input` struct / schema copy; `enumui` consumes `pkg/schema.Input` + `pkg/engine.EnumMeta` from the `replace ../gert` build. `gert-vscode` ships no `*.schema.json`, no `jsonValidation`/`yamlValidation`, no YAML parser — asserted by two green tests (CE-C-02). **Preserved.** |
| **AR-CE-4 §5 / D-3 — coded errors** | `mapRunbookError` now classifies via `errors.As(err, &errkit.Coder)` before any substring branch; `error.data.code` carries `ENUM-008`; `POST /runs` returns a JSON body with `code`. `safeMessageForCode` deliberately never echoes the rejected value or a member list — **no leakage**. Regression tests `rpc_enum_test.go`, `rpc_warnings_test.go` present and green. |
| **D-4 — warnings in the library entry point** | `pkg/run.StartWithWarnings` added additively; `Start` preserved for the `replace` build. RPC `run.start` and `POST /runs` both return `warnings[]`. |
| **DTO shape (as a Go type)** | `graphdoc.InputDecl` matches §2 normatively: declared order preserved, `enum` omitted when redacted, `enumMemberCount` only when redacted, `default` dropped under redaction, `enum` never folded into `options`. `InteractionField` likewise. |
| **CLI/TUI `--var` parity is live** | Executed: `gert dry-run --var env_name=nope` → `ENUM-008 …` exit 2; `gert-tui -var env_name=nope` → `Error: ENUM-008: input "env_name": …` exit 1. `parseVarFlags` is syntax-only; membership is decided solely by `schema.CheckCallerInputBindings`. **No client-side normalisation, no client-invented code.** |
| **No auto-selection / C1 / verbatim submission (unit level)** | `NoAutoSelect` sentinel `-1` in `multiform.go`; `nfcEqual` is comparison-only and submission uses the option's own `Value`; `enumui` redacted path carries zero members. All green in `internal/enumui`, `internal/interaction`. |

## 2. Blocking defects (executed evidence)

**B-1 — the DTO never reaches any client. The sole renderer drops it.**
`graphdoc.Document.Inputs` was added, but `pkg/preview/render/graphjson.Document` (the shape used by
BOTH `gert preview --format graphjson` and serve's `/preview/document`) has no `Inputs` field and
does not copy it in `RenderWithState`. Executed against the real binary on
`gert-vscode/test/fixtures/enum.runbook.yaml`: the emitted JSON contains **no `inputs` key at all**.
Consequences: **CE-D-01, CE-D-02, CE-D-03 fail**; `gert-vscode`'s `extractInputDecls` always returns
`undefined`, so `gert.validateInputs` **always** falls through to the comma-separated free-text box
and its QuickPick selector is unreachable dead code. "VS Code selector behavior works once DTO
arrives" is **not** satisfied — the DTO does not arrive.

**B-2 — `InteractionField.Enum` has no live producer.**
`NewEnumInteractionField` is referenced only by `broker_enum_test.go`. Nothing anywhere in `gert`
ever assigns `input.FormField.Enum` / `.EnumRedacted` / `.EnumMemberCount`, so `toFields` copies a
field that is always empty. D-2's carriage exists as a type, not as behaviour. **CE-D-04, CE-R-01..06
unverifiable on any live interaction.**

**B-3 — no declaration-driven input form in the web GUI.**
`internal/serve/static/preview.html` is **unmodified**: the only inputs surface is still the raw JSON
box at `:912-915`. T-GUI-INPUT-FORM is not started. AR-CE-3 §6 permits the box as a *fallback*, not
as the only surface once the DTO exists. **CE-R-01..R-06 (G) fail.**

**B-4 — TUI enum selector is a disconnected helper. Ken's statement is CONFIRMED and remains true after Don's landing.**
`internal/enumui.BuildEnumField` is called only from `internal/enumui/enumui_test.go` and
`internal/e2e/enum_parity_test.go`; **no production path in `cmd/gert-tui` or `internal/tui`
constructs a form from runbook input declarations.** The TUI never prompts for top-level inputs at
all. `enum_parity_test.go` calls the helper directly, so it proves the helper, not the surface — it
is precisely the "assertion about intent" I said at the ruling I would refuse. The gate requirement
"TUI actually permits/selects only declared enum values … live rather than a disconnected helper" is
**met only for `-var`** (B-4 does not affect that path) and **fails for the interactive surface**.

**B-5 — Leslie's D-4 dependency is now resolved but not consumed; the stale-blocked comment is wrong.**
`session/live.go`'s `engineWarning` adapter documents itself as "the shared runtime does not
implement it yet… always a no-op". Don has since shipped `run.StartWithWarnings`, yet
`cmd/gert-tui/main.go:91` still calls `run.Start` and discards warnings. Executed on a
case-only-distinct fixture: CLI prints `gert: warning: inputs.env_name: [ENUM-W001] …`; the TUI
prints **nothing**. **CE-W-01 — an explicitly gating row — fails.** ENUM-W001 is still invisible in
the TUI, which is the exact condition AR-CE-4 §6 forbids.

**B-6 — anti-drift coverage incomplete.**
No client-parity matrix file exists (`design/gert/conformance/` holds only `tv-enum.yaml`,
`tv-pkg-resolve.yaml`). §9's rows are scattered across three repos as ad-hoc tests with no
vector-ID cross-reference, so there is no single artifact that shows which of CE-* is covered.
CE-C-03 (vendored corpus SHA) is in place; CE-T-01..T-04 are not demonstrated on any client.

## 3. Non-blocking observations

- `gert-tui` `go test ./...` fails in `internal/tui` (`TestPreviewMode_*` reference a macOS
  `/Volumes/Projects/...` fixture path) and `internal/e2e` (`TestE2E_CollectHealth*`, network-
  dependent). Both look pre-existing/environmental and unrelated to enum work, but the reviser MUST
  confirm rather than assume. `gert` (`pkg/preview/...`, `internal/serve/...`, `pkg/run/...`) and
  `gert-vscode` (26/26) are green.
- `gert-vscode/src/enumInputs.ts` and `gert-tui/internal/enumui/enumui.go` both carry doc comments
  asserting the DTO "does not exist yet". After B-1 is fixed these become actively misleading.
- `graphdoc.buildInputDecls` name-sorts the declaration list. Acceptable for hash determinism (Go
  map order is undefined and `schema.Runbook.Inputs` is a map), and member order within a
  declaration is untouched — but it must be stated normatively, since AR-CE-3 §1 says "declared
  order" and a reader will otherwise assume it applies to the list too.
- `writeRunbookError` still concatenates `err.Error()` for uncoded errors. Fine today; re-check if a
  coded error ever loses its code.

## 4. Ruling on the two questions put to me

1. **Ken's stated lack of live TUI integration:** **upheld, and it is blocking.** The helper +
   helper-level tests do not constitute integration. B-4.
2. **Leslie's dependency on Don's DTO:** **discharged on the runtime side, not consumed on the
   client side.** `StartWithWarnings` exists; the TUI does not call it (B-5), and the DTO the GUI/VS
   Code work waits on is silently dropped by the renderer (B-1). Leslie is no longer blocked; the
   work is simply unfinished.

## 5. Reviser assignment

**David** — independent of every rejected portion (he authored neither the runtime DTO/serve work
(Don), nor the TUI (Ken), nor the VS Code changes). Assigned as sole reviser.

### Exact fixes required (all must be green before re-gate)

| # | Repo | Fix |
|---|---|---|
| **F-1** | `gert` | Add `Inputs []graphdoc.InputDecl \`json:"inputs,omitempty"\`` to `pkg/preview/render/graphjson.Document` and copy `doc.Inputs` in `RenderWithState`. Verbatim pass-through: no re-sorting, no member reordering, `enum` omitted (not `[]`) when absent or redacted. |
| **F-2** | `gert` | Regression test in `pkg/preview/render/graphjson`: declared order `["staging","prod"]` survives to rendered JSON; unconstrained input emits no `enum` key; redacted input emits `enumRedacted`+`enumMemberCount` and **no** members. Plus a `cmd/gert` test asserting `gert preview --format graphjson` output contains `inputs[]` for an enum runbook (CE-D-01/02/03). |
| **F-3** | `gert` | Give `InteractionField.Enum` a live producer, or delete `NewEnumInteractionField` and the unused `FormField.Enum*` fields. Do **not** leave a wire field no code path populates. If retained, add a test that drives a real interaction and observes `enum` on the SSE frame with `options` untouched (CE-D-04). |
| **F-4** | `gert` | `internal/serve/static/preview.html`: declaration-driven input form from the DTO — closed `<select>` in declared order, nothing preselected when required-and-defaultless, explicit "unset" distinct from `""`, free-text for `enumRedacted` with count only, raw-JSON box retained as fallback. No JS membership check, no sorting/dedupe/case-fold (CE-R-01..R-06, CE-U-05). |
| **F-5** | `gert-tui` | `cmd/gert-tui/main.go`: call `run.StartWithWarnings`, feed the returned `[]parser.ParseWarning` into `session.WarningMsg`, render them in `TUIApp.Update`. Delete the `engineWarning` no-op adapter and its now-false comment. Gating regression test: case-only-distinct fixture ⇒ `ENUM-W001` visible in the rendered view, run not failed (CE-W-01). |
| **F-6** | `gert-tui` | Wire `enumui.BuildEnumField` into a real path: prompt for declared top-level inputs (from `handle.State().Plan.Inputs` + `Plan.Validation.EnumConstraints`) through the existing multiform before/at run start, for inputs not already bound by `-var`. Test must drive the **app**, not the helper: assert the rendered view shows members in declared order, nothing preselected, and that the submitted value is byte-verbatim (CE-R-01/R-02/R-03, CE-V-06). |
| **F-7** | `gert-vscode` | Once F-1 lands, add a test that runs the **real** `gert preview --format graphjson` on the enum fixture and asserts `extractInputDecls` returns a decl with `enum: ["prod","staging"]` in order — i.e. the selector path is actually reached. Update the stale "DTO does not exist yet" comments in `enumInputs.ts`. |
| **F-8** | `gert-tui` | Update `internal/enumui/enumui.go`'s package doc once the DTO is real; keep consuming engine types (no client DTO struct). |
| **F-9** | `gert-private` | Client-parity matrix file under `design/gert/conformance/` (separate from the frozen `tv-enum.yaml`), one row per CE-* ID with its owning test's repo+name, covering CE-T-01..T-04. Do not extend `tv-enum.yaml`. |
| **F-10** | `gert` | One-line normative comment on `buildInputDecls` stating that the name-sort applies to the declaration **list** only and never to enum members. |
| **F-11** | all | Confirm the pre-existing `gert-tui` `internal/tui` / `internal/e2e` failures are environmental and unrelated; if not, fix or file them explicitly. |

**Sequencing:** F-1/F-2 first (everything else is blocked on the DTO actually shipping), then F-3/F-4
(`gert`), then F-5/F-6 (`gert-tui`), then F-7/F-8, then F-9/F-10/F-11. Engine changes merge before
client changes — the `replace ../gert` path makes concurrent edits a real trap.

**Standing constraints for the reviser:** no client schema/struct/decoder/validator; no auto-select;
no client-side trim/case-fold/NFC before submission (NFC on the comparison path only); no member
ever rendered, logged, cached, or autocompleted under `enumRedacted`; no `ENUM-*` code invented
client-side; `enum` never merged into `options`.

## 6. Re-gate criteria

I will re-review when F-1..F-11 are complete. I will specifically execute, not read: the real
`gert preview --format graphjson` output for `inputs[]`; the TUI rendering ENUM-W001; the TUI
selector driven at the app level; and the VS Code selector reached against a real engine build.

**No architecture reopened.** AR-CE-1..10 stand as ratified.
