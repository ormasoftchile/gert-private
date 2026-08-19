# Client-Parity Acceptance Matrix (CE-*)

Status: independent revision by David (Integration Engineer), resolving Barbara's rejection
`.squad/decisions/inbox/barbara-client-enum-parity-gate-review.md` (B-1..B-6, F-1..F-11) of the
`AR-CE-1..10` ruling in `.squad/decisions/inbox/barbara-client-enum-compatibility-ruling.md`.

This file is **F-9**: a client-parity matrix separate from the frozen conformance corpus
(`tv-enum.yaml`, `tv-pkg-resolve.yaml`, etc. in this same directory). It does not add or modify any
vector; it cross-references AR-CE-9 §9's acceptance-matrix row IDs to the actual test that executes
each one, across all three client repos plus the shared engine. `S` = surface (G = web GUI/serve,
T = TUI, V = VS Code, E = engine/shared), matching AR-CE-9's own column.

Repo roots referenced below: `gert` = engine/runtime repo, `gert-tui` = TUI repo,
`gert-vscode` = VS Code extension repo.

| ID | S | Repo | Owning test | Notes |
|---|---|---|---|---|
| CE-S-01 | E | gert | `internal/conformance.TestEnumConformance` (runs every `tv-enum.yaml` vector, including plain `enum:` declarations, through the real CLI) | No dedicated single-vector test; covered as part of the full corpus run. |
| CE-S-02 | G,T,V | gert / gert-tui / gert-vscode | `internal/conformance.TestEnumConformance` (G/engine); `internal/e2e.TestEnumParity_DeclaredOrderCarriedToPlan` (T); `test/enumRuntimeRegression.test.js` "CE-S-01/CE-V-01 regression" (V) | Each client loads the same fixture and receives no client-originated diagnostic. |
| CE-S-03 | V | gert-vscode | `test/manifest.test.js` "CE-C-02: no bundled schema..." | No schema contribution point exists at all, so no engine-absent squiggle is possible by construction. |
| CE-D-01 | G | gert | `pkg/preview/graphdoc/enum_decl_test.go` (Document level); `pkg/preview/render/graphjson/graphjson_test.go` `TestRender_Inputs_DeclaredOrderSurvives` (renderer level); `cmd/gert/preview_graphjson_inputs_test.go` `TestPreview_GraphJSON_CarriesDeclaredInputs` (real CLI binary level) | F-1/F-2 fix + regression tests (this revision). |
| CE-D-02 | G | gert | Same three tests as CE-D-01 (each asserts non-alphabetical declared order, e.g. `["staging","prod"]`) | |
| CE-D-03 | G,T | gert / gert-tui | `cmd/gert/preview_graphjson_inputs_test.go` `TestPreview_GraphJSON_CarriesDeclaredInputs` (asserts `free_text` carries no `enum` key, G); `cmd/gert-tui/input_prompt_test.go` `TestDeclaredEnumInputFields_SelectorInDeclaredOrder` (asserts `free_text` is never prompted, T) | T-side test added this revision (F-6). |
| CE-D-04 | G,T | gert | `pkg/preview/graphdoc/doc.go` doc comment (normative statement) + `internal/serve/broker.go` (`InteractionField` struct, post-F-3, carries only `Options`, no `Enum*` fields at all) | F-3 removed the unused/never-populated `InteractionField.Enum*` fields rather than building a producer (see decision-inbox revision record) — after that removal, a leak between `options` and `enum` is structurally impossible on the collector-field wire type, though no dedicated regression test asserts this beyond the type definition itself (accepted limitation: no `broker`-level enum/options co-existence test exists, since there is no longer an `Enum` field for one to leak into). |
| CE-R-01 | G,T | gert / gert-tui | `internal/serve/preview_html_inputs_form_test.go` `TestPreviewHTML_DeclarationDrivenInputsForm` (G); `cmd/gert-tui/input_prompt_test.go` `TestAppLevel_EnumFieldRendersAndSubmitsVerbatim` (T, app-level, drives the real `interaction.MultiFieldFormModel`) | F-4 (G) / F-6 (T) fixes, this revision. |
| CE-R-02 | G,T | gert-tui | `internal/enumui/enumui_test.go` `TestBuildEnumField_DefaultPreselectsOnlyWhenMember`; `cmd/gert-tui/input_prompt_test.go` `TestDeclaredEnumInputFields_SelectorInDeclaredOrder` (asserts no preselect when no declared default) | G-side default-preselect is exercised by `preview.html`'s `InputsForm` `useEffect` (manual/static verification only — no headless-browser test harness exists in this repo; accepted limitation, see below). |
| CE-R-03 | G,T | gert / gert-tui | `internal/serve/static/preview.html` `buildFormInputs()` (omits unset `""` keys, G); `cmd/gert-tui/input_prompt.go` `collectDeclaredEnumInputs` (drops empty-string values before merging into `-var` bindings, T) | Same G-side manual/static-only caveat as CE-R-02. |
| CE-R-04 | G,T | gert-tui | `internal/enumui/enumui_test.go` `TestBuildEnumField_CaseOnlyDistinctMembersPreserved` | |
| CE-R-05 | G,T | gert / gert-tui | `internal/serve/preview_html_inputs_form_test.go` (redaction branch present, G); `internal/enumui/enumui_test.go` `TestBuildEnumField_RedactedNeverCarriesMembers` (T); `cmd/gert-tui/input_prompt_test.go` `TestDeclaredEnumInputFields_SelectorInDeclaredOrder` (real-doc redacted-field assertion, T) | |
| CE-R-06 | G,T | — | none found | **Accepted limitation**: no 40-member-enum fixture/test exists in either client repo. Nothing in F-4/F-6's rendering logic truncates (`InputsForm`/`declaredEnumInputFields` iterate the full `Enum` slice with no length cap), but this is not proven by an executed test. |
| CE-V-01 | E,G,T | gert / gert-tui | `cmd/gert/enum_r1_r4_regression_test.go` `TestRun_Enum008_CallerVarAccepted` (E, accept path); `internal/serve/rpc_enum_test.go` (G); `internal/e2e/enum_parity_test.go` `TestEnumParity_CLIAndTUIVarFlagAgreeOnRejection` (T) | Pre-existing (Don/Ken), unaffected by this revision. |
| CE-V-02 | G | gert | `internal/serve/rpc_enum_test.go` `TestMapRunbookError_Enum008_CodedNotSubstring` | Pre-existing. |
| CE-V-03 | G | gert | `internal/serve/interactions.go` `writeRunbookError` (implementation) + `internal/serve` HTTP handler tests (pre-existing) | |
| CE-V-04 | V | gert-vscode | `test/enumRuntimeRegression.test.js` "CE-V-04/D-3 regression: ENUM-008 ... survives verbatim through deriveFailureMessage" | **Known gap, documented below**: this real-CLI-driven test currently fails against the present (independently dirty, locked-out-author-owned) engine error text, which prefixes the code with `input "<name>": ` before `ENUM-008:`. Not modified this revision (see Remaining Limitations). |
| CE-V-05 | G,T | gert / gert-tui | `internal/serve/static/preview.html` (raw-JSON fallback always wins outright — advisory only, F-4); `cmd/gert-tui/input_prompt.go` `collectDeclaredEnumInputs` (fail-open on any prompting error, never blocks the run, F-6) | |
| CE-V-06 | E | gert-tui | `cmd/gert-tui/input_prompt_test.go` `TestAppLevel_EnumFieldRendersAndSubmitsVerbatim` (submitted value round-trips byte-identical to the declared member) | Added this revision (F-6). |
| CE-U-01 | E,G,T | gert | `internal/serve/rpc_enum_test.go` (comment references CE-U-01) | Pre-existing. |
| CE-U-02 | G,T | gert-vscode | `test/enumInputs.test.js` "CE-U-02: parseVarPairs preserves leading/trailing whitespace in a value verbatim" | Pre-existing. |
| CE-U-03 | G,T | gert-vscode | `test/enumInputs.test.js` "CE-U-03: nfcEquals is case-sensitive" | Pre-existing. |
| CE-U-04 | T | gert-tui | `internal/interaction/multiform_test.go` (NFC-comparison-only preselect regression, pre-existing) | |
| CE-U-05 | G,T | gert-tui | `internal/enumui/enumui_test.go` `TestBuildEnumField_NonASCIIMembersIntact` | **G-side gap**: no equivalent non-ASCII-member test exists for `preview.html`'s `InputsForm`; the rendering path (plain `<option>` text nodes) does not transform strings, but this is not proven by an executed test (same class of gap as CE-R-02/CE-R-03). |
| CE-T-01 | G,T,V | gert | `internal/engine/enum_trace_test.go` `TestEngine_PlanValidated_CarriesEnumConstraintsOnce` | Pre-existing (Don); engine-level, exercised identically regardless of which client drives it. |
| CE-T-02 | G,T,V | gert | Covered implicitly by every DTO/JSON round-trip test using `encoding/json` (Go structs silently ignore unknown fields on decode) and `gert-vscode`'s `extractInputDecls` (tolerates unknown sibling keys, see `test/enumInputs.test.js` "extractInputDecls tolerates unknown sibling keys ... (AR-CE-7)") | |
| CE-T-03 | G | gert | `cmd/gert/preview_graphjson_inputs_test.go` `TestPreview_GraphJSON_RedactedInputs_NoMemberLeak` (CLI/graphjson analogue) | Added this revision (F-2). The exact `plan.validated` trace-event rendering path is exercised by pre-existing `internal/engine` tests; this revision's test covers the equivalent guarantee on the preview-document DTO. |
| CE-T-04 | E | gert | `cmd/gert/substitution_enum_integration_test.go` (asserts `--output=json` envelope shape with enum errors present) | Pre-existing. |
| CE-W-01 | T | gert-tui | `internal/session/live_test.go` `TestLiveSession_EmitWarnings_TranslatesCodedWarnings`; `internal/tui` warning-rendering wiring in `app.go`'s `Update`/`renderStatusBar` (F-5, this revision) | No app-level rendered-view assertion exists for the warning banner text itself (only the `session.WarningMsg` translation is asserted); the `internal/tui` package's own test suite has the pre-existing environmental failures documented in F-11 below, which block adding one in this session. |
| CE-W-02 | G | gert | `internal/serve/rpc_warnings_test.go` `TestRPC_RunStart_SurfacesParseWarnings` | Pre-existing. |
| CE-C-01 | T | gert-tui | `internal/enumui/enumui.go` doc comment (static: no YAML decode, no runbook struct, no schema file in this package) | Pre-existing assertion; no automated static-analysis test enforces it beyond code review. |
| CE-C-02 | V | gert-vscode | `test/manifest.test.js` "CE-C-02: no bundled schema..." and "CE-C-02: no vendored *.schema.json..." | Pre-existing. |
| CE-C-03 | E | gert | `internal/conformance/enumdata/README.md` (manual "copied as of <date>" pin) | **Accepted limitation**: no automated SHA-256 equality test between `internal/conformance/enumdata/tv-enum.yaml` and `design/gert/conformance/tv-enum.yaml` was found. The corpus is manually synced per the README; a drift would not be caught by CI today. Out of this revision's assigned scope (B-1..B-6/F-1..F-11 do not name this gap), flagged here for visibility per F-9's "one row per CE-* ID" requirement. |

## Remaining limitations discovered while compiling this matrix

1. **CE-V-04 test failure (gert-vscode `test/enumRuntimeRegression.test.js`)**: the real `gert
   dry-run` CLI's ENUM-008 error text is `input "env_name": ENUM-008: value is not a declared enum
   member`, not `ENUM-008: ...` at the very start of the message. The pre-existing test (added by a
   locked-out author, untracked/uncommitted — part of the protected concurrent work this revision
   must not alter) asserts the code leads the message with no prefix. This formatting lives in
   `pkg/schema` (dirty/untracked files owned by other authors, e.g. `pkg/schema/enum.go`), which is
   out of this revision's assigned scope (B-1..B-6/F-1..F-11 do not touch caller-binding
   error-message formatting) and is protected concurrent work per this task's instructions. Reported
   here as an exact conflict rather than silently patched or worked around.
2. **CE-R-02/CE-R-03/CE-U-05 web-GUI rows**: `internal/serve/static/preview.html` has no headless-
   browser test harness in this repo, so its declaration-driven `InputsForm` behaviour (default
   preselection, unset-vs-empty-string handling, non-ASCII member rendering) is verified only by
   static source assertions (`internal/serve/preview_html_inputs_form_test.go`) plus manual
   inspection, not an executed DOM-level test.
3. **CE-R-06 (40-member enum)**: no fixture/test exists for either client; the rendering code paths
   do not truncate, but this is unproven by an executed test.
4. **CE-C-03 (vendored corpus SHA)**: no automated equality check exists; see table row above.

See `.squad/decisions/inbox/david-client-enum-parity-revision.md` for the full B/F resolution
matrix, files/tests changed per repo, and protected-file verification.
