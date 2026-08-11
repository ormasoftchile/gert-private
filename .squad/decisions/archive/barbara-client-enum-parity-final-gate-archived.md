# Final Gate — Client Enum Parity (B-1..B-6 / F-1..F-11) — **APPROVED**

**From:** Barbara (Lead / Architect), final reviewer
**Date:** 2026-08-11T11:42-07:00
**Requested by:** Cristián Ormazábal Ortega
**Governs:** `C:\One\OpenSource\gert`, `C:\One\OpenSource\gert-tui`, `C:\One\OpenSource\gert-vscode`
**Gate authority:** `barbara-client-enum-compatibility-ruling.md` (AR-CE-1..10, RATIFIED);
upstream `AR-ENUM-1..15` + C1/C2/C3 (not reopened).
**Supersedes:** `barbara-client-enum-parity-gate-review.md` (REJECTED, B-1..B-6 / F-1..F-11).
**Reviser under review:** David (independent, under author lockout).
**Method:** read the ruling, the prior gate, David's revision record and the current diffs; then
**built and executed the real binaries** in all three repos and observed behaviour directly —
`gert preview --format graphjson`, a live `gert serve` on `127.0.0.1:7911` (`/preview/document`
and `POST /runs`), `gert dry-run`, `gert run`, the real `gert-tui` binary driven interactively,
and `go test` / `npm test`. No product artifact was modified; every pre-existing dirty/untracked
file in all four repos was preserved; all my scratch fixtures and verification binaries were
removed at the end of the session.

---

## Verdict: **APPROVED**

Every blocking defect B-1..B-6 is fixed, and each fix is verified by execution rather than by
reading a test name. The three gaps David self-reported are **acceptable**, not blocking: none of
them is required by the original ask or by AR-CE-1..10. One additional coverage gap I found myself
(§4) is likewise non-blocking but carries a mandatory follow-up.

---

## 1. Direct verification of each B/F item

| # | Verdict | Executed evidence I observed myself |
|---|---|---|
| **B-1 / F-1, F-2** | **Fixed** | `graphjson.Document` now carries `Inputs []graphdoc.InputDecl` and `RenderWithState` copies it. Real binary on an enum fixture emits `"inputs": [{"name":"env_name","type":"string","required":true,"enum":["prod","staging"]}]`. On my own multi-input fixture, member order survives **verbatim** (`["staging","prod","dev"]` — non-alphabetical, unsorted), while only the declaration *list* is name-sorted. An unconstrained input emits **no** `enum` key (not `[]`). |
| **B-1 (redaction)** | **Fixed, redaction-safe** | With `governance.redact: [{pattern: "^inputs\\.secret_env$"}]`, the live output is `{"name":"secret_env","required":true,"enumRedacted":true,"enumMemberCount":3}` — **zero members, no `default`, no `enum` key**. C1 holds on the wire. |
| **B-1 (serve)** | **Fixed** | Same DTO retrieved over live HTTP from `GET /preview/document?...&format=graphjson` — the exact endpoint the web GUI consumes. |
| **B-2 / F-3** | **Fixed by sanctioned removal** | `InteractionField.Enum/EnumRedacted/EnumMemberCount`, `NewEnumInteractionField` and `FormField.Enum*` are gone; `broker.go` and `prompt.go` carry normative comments forbidding re-adding them without a live producer. No wire field lacks a producer. This was the alternative my own F-3 explicitly offered, and the rationale (no S1-S4 enum site links collector fields; a producer would require `Input.From` sourcing, out of scope under `T-ENUM-FROM-SOURCING`) is correct. |
| **B-3 / F-4** | **Fixed** | `preview.html` is no longer raw-JSON-only. `InputsForm` is mounted in the live render tree (`e(InputsForm, {decls: (doc && doc.inputs) || [], ...})`), rendering a **closed `<select>` in declared order** for non-redacted enums, **free text with count only** for `enumRedacted`, and free text for unconstrained inputs. `buildFormInputs()` submits values byte-verbatim and **omits** any field left at `""` — "unset" is key-absent, never a submitted empty string. No JS membership check, no sort/dedupe/case-fold. The raw-JSON box is retained as a fallback that wins outright when non-empty. |
| **B-4 / F-6** | **Fixed, production-connected** | I ran the real `gert-tui` binary with no `-var`. The production entry point (`cmd/gert-tui/main.go` → `collectDeclaredEnumInputs`, before `run.StartWithWarnings`) rendered: `env_name *` as a selector listing `staging`, `prod` **in declared order with nothing preselected**, and `secret_env *` as a free-text box hinted `one of 3 permitted values` **with no member shown**. This is the live surface, not a helper. `NoAutoSelect` (sentinel `-1`) is enforced in `multiform.go` with its own regression tests, and a required unselected field cannot be submitted. |
| **B-5 / F-5** | **Fixed** | I ran the real `gert-tui` binary on a case-only-distinct fixture. The rendered view showed `⚠ ENUM-W001: enum members "Prod" and "prod" are case-only-distinct` above the status bar, and the run **completed** (`Status: completed`) rather than failing. `run.StartWithWarnings` is wired, `SetParseWarnings` feeds `LiveSession`, `WarningMsg` never touches `m.status`/`m.err`, and the `engineWarning` no-op adapter with its false comment is deleted. CE-W-01 passes. |
| **B-6 / F-9** | **Fixed** | `design/gert/conformance/client-parity-matrix.md` exists, one row per CE-* ID with owning repo + test name, covering CE-T-01..T-04, and **explicitly marks its own gaps** rather than overclaiming. `tv-enum.yaml` untouched. |
| **F-7** | **Fixed** | `npm test` with `GERT_BIN` pointed at a freshly built engine: the CE-D-01/CE-D-02 test executes the **real** `gert preview --format graphjson` and recovers `enum: ["prod","staging"]` in file order — the VS Code selector path is genuinely reached. `validateInputs` calls the real CLI, `extractInputDecls` feeds a closed `QuickPick` in declared order that highlights but never auto-submits a default, and returns the picked value verbatim. Stale "DTO does not exist yet" comments are gone. |
| **F-8, F-10** | **Satisfied** | `enumui` package doc updated and still consumes engine types only. `buildInputDecls` carries the normative comment that the name-sort applies to the declaration list only and never to members. |
| **F-11** | **Confirmed** | `internal/tui/preview_test.go` hardcodes `/Volumes/Projects/...` macOS paths and is **committed and untouched** (`git status --porcelain` empty for it). `internal/e2e.TestE2E_CollectHealth` fails on unreachable hosts. Both environmental, neither caused by enum work. |

### Cross-cutting gate conditions

- **Typed ENUM errors, no leak:** live `POST /runs` with a non-member returns HTTP 400 and
  `{"code":"ENUM-008","error":"The submitted value is not a declared enum member."}` — the code is
  structured, and **neither the rejected value nor any member is echoed**, including for the
  redacted input. `gert dry-run` exits 2 with `ENUM-008`; `gert-tui -var` exits 1 with `ENUM-008`.
- **No client-invented codes / no client normalisation:** `parseVarFlags` remains syntax-only;
  NFC is used on the comparison path only; submission is byte-verbatim on all three surfaces.
- **No schema fork (AR-CE-1):** `gert-tui` has no YAML decoder or runbook struct and consumes the
  engine via `replace ../gert`; `gert-vscode` ships no `*.schema.json`, no `yamlValidation`/
  `jsonValidation`, no YAML parser. Both re-verified this session.
- **Test state:** `gert` — `go build ./...` and `go test ./...` **fully green**. `gert-tui` —
  builds clean; `cmd/gert-tui`, `internal/session`, `internal/enumui`, `internal/interaction` green;
  only the two confirmed environmental clusters fail. `gert-vscode` — compiles clean, 26/27, the one
  failure being the CE-V-04 conflict adjudicated in §3.

---

## 2. Ruling on the two questions carried over from the prior gate

1. **Ken's stated lack of live TUI integration — now DISCHARGED.** The selector is reached from
   `cmd/gert-tui/main.go` on the real binary. My prior objection ("a helper plus helper-level tests
   is not integration") is answered by execution, not by assertion.
2. **Leslie's D-4 dependency — DISCHARGED and consumed.** `StartWithWarnings` is called by the TUI
   and its warnings are rendered.

---

## 3. Adjudication of David's three claimed remaining gaps

| Gap | Classification | Reasoning |
|---|---|---|
| **CE-V-04 conflict** (`gert-vscode` real-CLI test asserts `/^ENUM-008:/`; engine now emits `input "env_name": ENUM-008: ...`) | **ACCEPTABLE — not a blocker** | AR-CE-9 §9's CE-V-04 row requires *"Code and message reach the output channel verbatim."* They do: `ENUM-008` and the full message are present and unaltered. The **test over-specifies** the ruling by additionally anchoring the code to the start of the string. The message text is produced by `pkg/schema` (untracked `enum.go`, protected in-flight work by a locked-out author), and the failing test is itself an untracked file by a locked-out author. David was correct to report the conflict rather than edit either party's protected file. Requires reconciliation, not rejection. |
| **No headless-browser tests for the GUI** (CE-R-06 40-member, CE-U-05 non-ASCII, CE-R-02/R-03 GUI default/unset proven by source inspection only) | **ACCEPTABLE — not a blocker** | AR-CE-1..10 never mandated a DOM harness, and introducing one would mean adding a browser test toolchain to `gert` — a new testing tool this task does not need. I independently confirmed the live endpoint feeds `doc.inputs` to the mounted `InputsForm`, and that the rendering path iterates the full `Enum` slice with no length cap and no string transformation. Coverage debt, not a defect. |
| **No automated corpus SHA check** (CE-C-03, `internal/conformance/enumdata/tv-enum.yaml` vs `design/gert/conformance/tv-enum.yaml`) | **ACCEPTABLE — not a blocker** | Pre-existing, unrelated to client parity, and named in none of B-1..B-6 / F-1..F-11. The corpus is manually pinned per its README. David flagged it for visibility rather than silently expanding scope, which is the right call. |

---

## 4. One additional gap I found (non-blocking, mandatory follow-up)

F-5 literally required an app-level regression test asserting `ENUM-W001` is visible **in the
rendered view**. Only the `session`-level translation is asserted (`live_test.go`); no
`internal/tui` test drives `Update`/`renderStatusBar` for a warning. David's stated justification —
that `internal/tui`'s environmental failures "block adding one" — is **factually incorrect**: I ran
`go test ./internal/tui/ -run TestApp` and fifteen app-level tests pass; only the two macOS-path
`TestPreviewMode_*` cases fail. Nothing prevented the test.

I am not rejecting on this, because the **behaviour itself is proven** — I executed the real binary
and watched the warning render on a completed run, which is stronger evidence than any unit test.
This is anti-drift debt, and it is ticketed below as mandatory.

---

## 5. Exact user-visible supported behaviour (as of this approval)

An operator working with a runbook that declares `inputs.<name>.enum` gets, on **all four**
surfaces:

- **CLI (`gert run` / `gert dry-run`)** — bind values with repeatable `--var name=value`. A
  non-member is rejected with `ENUM-008` and a non-zero exit (2 on `dry-run`). Case-only-distinct
  members produce a non-fatal `gert: warning: ... [ENUM-W001] ...` and the run proceeds.
- **Web GUI (`gert serve` → `preview.html`)** — after **Load**, a declaration-driven input bar
  appears above the graph: a closed dropdown listing the declared members **in the order written in
  the YAML** for each enum input, a free-text box for unconstrained inputs, and a free-text box
  labelled *"one of N permitted values"* for redacted enums (no member is ever shown). Nothing is
  preselected for a required input without a declared default; leaving a field untouched omits it
  from the submission entirely (it is never sent as `""`). The raw-JSON inputs box remains available
  and, when non-empty, takes precedence over the form. Rejections return HTTP 400 with a structured
  `{"code":"ENUM-008", ...}` body.
- **TUI (`gert-tui`)** — `-var name=value` works as on the CLI. When enum-constrained inputs are
  **not** supplied via `-var`, the TUI now prompts **before the run starts** with an arrow-key
  selector showing members in declared order with nothing preselected, and a free-text field with a
  count-only hint for redacted enums. `ENUM-W001` and other non-fatal coded warnings render as a
  yellow banner above the status bar without failing the run.
- **VS Code (`gert.validateInputs`)** — runs the real engine, reads the live `inputs[]` DTO, and
  prompts per declared input: a closed QuickPick over declared members in declared order (a declared
  default is *highlighted*, never auto-accepted; "Leave unset" is offered for optional inputs), or a
  free-text box for redacted/unconstrained inputs. Engine codes and warnings reach the output
  channel verbatim.

Across every surface: the selected value is submitted **byte-verbatim** (no trim, no case-folding,
no NFC normalisation before submission); membership is decided **only** by the engine; no client
invents an `ENUM-*` code; and under redaction no member is ever rendered, logged, cached, or
autocompleted.

## 6. True limitations the user should know

1. **Interactive prompting covers enum-constrained top-level inputs only.** Non-enum inputs are
   still `-var`/JSON-box only in the TUI; the TUI prompt is skipped for embedded/FS-backed runbooks
   and fails open (the run proceeds) when there is no TTY or the runbook cannot be pre-parsed.
2. **Enum declarations are not yet interactively *sourced* mid-run.** `Input.From: prompt` remains
   out of scope under the still-open `T-ENUM-FROM-SOURCING`; collector-step fields carry no enum
   constraint and deliberately no longer have `Enum*` wire fields.
3. **Error text ordering is cosmetically inconsistent.** The engine currently emits
   `input "x": ENUM-008: ...`, and the TUI's own `formatRunError` re-prefixes the code, producing a
   visible duplication (`ENUM-008: input "x": ENUM-008: ...`); the TUI warning banner duplicates
   similarly (`⚠ ENUM-W001: ENUM-W001: ...`). Cosmetic only — the code is present and correct, and
   nothing leaks — but it is user-visible and is the root of the CE-V-04 test conflict.
4. **GUI enum rendering is not proven by an executed browser test** (§3), and **the TUI warning
   banner is not proven by an app-level test** (§4). Both behaviours were verified by hand this
   session; neither is protected against regression by CI.
5. **Conformance-corpus drift between the two `tv-enum.yaml` copies is not detected automatically.**
6. **Pre-existing, unrelated red tests remain in `gert-tui`**: `internal/tui/preview_test.go`'s
   hardcoded macOS paths and `internal/e2e`'s network-dependent `CollectHealth` cases.

## 7. Follow-up tickets (none gating this approval)

| Ticket | Repo | Scope |
|---|---|---|
| **T-TUI-WARN-VIEWTEST** *(mandatory)* | `gert-tui` | App-level test driving `TUIApp.Update` with a `session.WarningMsg` and asserting `ENUM-W001` appears in the rendered view with `m.status`/`m.err` untouched. Closes §4. |
| **T-ENUM-MSG-FORMAT** | `gert` + `gert-vscode` | Once the `pkg/schema` lockout lifts, settle one canonical ENUM-008 message ordering, remove the duplicated code prefixes in `gert-tui`'s `formatRunError` and the warning banner, and reconcile the CE-V-04 test to AR-CE-9's actual wording. Closes §3 row 1 and §6 item 3. |
| **T-GUI-DOM-TESTS** | `gert` | Optional browser-level coverage for CE-R-06 / CE-U-05 / CE-R-02 / CE-R-03 on `preview.html`. Only if a browser harness is wanted in this repo for other reasons. |
| **T-CORPUS-SHA-CHECK** | `gert` | Automated SHA-256 equality check between the two `tv-enum.yaml` copies. |
| **T-TUI-TESTPORT** | `gert-tui` | Replace `internal/tui/preview_test.go`'s hardcoded `/Volumes/...` paths; gate or stub `internal/e2e`'s network-dependent cases. |

**No architecture reopened.** AR-CE-1..10 stand as ratified. No reviser is assigned — this gate is
closed as approved.
