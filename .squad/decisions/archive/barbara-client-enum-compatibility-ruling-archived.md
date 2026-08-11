# Architecture Ruling — Client Compatibility for `runbook/v1` Input String Enums (AR-CE-1..10)

**From:** Barbara (Lead / Architect)
**Date:** 2026-08-11T09:45-07:00
**Requested by:** Cristián Ormazábal Ortega
**Status:** RATIFIED — binding on all GERT client surfaces
**Governs:** `C:\One\OpenSource\gert` (engine + `serve` web GUI), `C:\One\OpenSource\gert-tui`,
`C:\One\OpenSource\gert-vscode`
**Upstream, not reopened:** `barbara-enum-constraint-mvp-architecture-ruling-archived.md`
(AR-ENUM-1..15, C1/C2/C3) and `barbara-enum-mvp-final-gate-approval-archived.md` (APPROVED).

---

## 0. Finding first: the drift report is half wrong, and the wrong half matters

I read all three client trees before ruling. **No client rejects `enum:`.** The reported
"reject" symptom does not exist in code today, and any fix aimed at it would be aimed at nothing.

| Claim | Verdict | Evidence |
|---|---|---|
| GUI **rejects** enum | **False.** `internal/serve` owns no schema and no field allow-list; it delegates to `internal/parser`, which compiles the one embedded `schemas.RunbookSchema` (`gert/schemas/schema.go:6-9`, `internal/parser/validate_structural.go:19-35`) that already permits `enum` (`schemas/runbook.schema.json:122-141`, `:146-169`). YAML decode is non-strict (`internal/parser/unmarshal.go:22-42`). | read |
| TUI **rejects** enum | **False.** `gert-tui` contains no YAML decoder, no schema copy, no `Input` struct. It calls engine `run.Start` (`cmd/gert-tui/main.go:75-89`) and depends on the engine by `replace ... => ../gert` (`go.mod:5-8,53`). | read |
| VS Code **rejects** enum | **False.** No `contributes.jsonValidation`/`yamlValidation`, no bundled `*.schema.json`, no YAML parser dependency (`package.json:16-84`). It spawns `gert preview` / `gert serve` and iframes the server page (`src/extension.ts:57-60,92-116`; `src/serverManager.ts:55-71`). | read |
| Clients **lack support** for enum | **TRUE, and it is a transport gap, not a validation gap.** | below |

**The real defect is that an enum declaration never reaches any client.** Two DTO holes and one
error-mapping bug:

1. **D-1 — the input-declaration DTO does not exist.** The preview document carries no `inputs`
   at all (`pkg/preview/graphdoc/doc.go:66-75`). The web GUI therefore offers a single free-text
   box into which the operator hand-types the entire inputs object as JSON
   (`internal/serve/static/preview.html:543-544,621-630,912-915`). There is nothing to attach a
   selector to.
2. **D-2 — the interaction DTO carries `options` but not `enum`.** `InteractionField`
   (`internal/serve/broker.go:58-68`, mapped at `:458-468`) has `Options []InteractionOption` for
   collector fields and no enum member carriage. Both the GUI (`preview.html:1402-1417`) and the
   TUI (`internal/interaction/multiform.go:39-57,318-329`) already own working select widgets —
   they are starved of data, not of UI.
3. **D-3 — ENUM-008 is reported to RPC clients as "Parse error".** `mapRunbookError`
   (`internal/serve/rpc.go:876-894`) classifies by lowercased message substring; the ENUM-008
   text (`pkg/schema/enum.go:225-227`) matches neither the parse nor the `validation`/`[` branch
   and falls through to `rpcRunbookParseErr`. The operator is told their **file** is broken when
   their **value** was rejected. `errkit.Coder` already exists (`pkg/errkit/errors.go:15,55`), so
   this is a structural mapping that was simply never wired.

Two adjacent gaps, both engine-side, both blocking client parity:

4. **D-4 — `pkg/run.Start` drops `parsed.Warnings`** (`pkg/run/run.go:96-101`, never read again).
   `cmd/gert/run.go:179-184` prints them; the library entry point the TUI uses does not. **ENUM-W001
   is unreachable in the TUI by construction.**
5. **D-5 — `gert-tui` never populates `run.Config.Variables`** (`cmd/gert-tui/main.go:76-88`), so
   `CheckCallerInputBindings` (`pkg/run/run.go:179-189`) has nothing to check there. The TUI has no
   caller-binding path for declared inputs at all today.

Every requirement below follows from this: **clients are starved, not hostile.** The fix is DTO
carriage plus rendering, and it must not become a second validator.

---

## 1. AR-CE-1 — Schema acceptance: one schema, zero client forks

**Normative:**

1. There is exactly one structural authority for `runbook/v1`: `gert/schemas/runbook.schema.json`,
   embedded in the engine binary, kept byte-faithful to the design copy
   `design/gert/schemas/runbook.v1.schema.json`. A client MUST NOT fork, vendor, hand-copy,
   paraphrase, or partially re-encode it.
2. A client MUST NOT maintain a private allow-list of runbook keys, a private `Input`/`Output`
   struct, or a strict decoder (`KnownFields`, `DisallowUnknownFields`, `additionalProperties:false`
   in a client-owned schema) applied to runbook source. Any such construct is, by definition, a
   drift generator: the enum keyword would have been rejected by it, and the next keyword will be.
3. A client MUST NOT gate execution on its own structural verdict. If a client wants a verdict, it
   asks the engine (`gert preview`, `gert validate`, `serve`'s parse path) and renders the answer.
4. **Editor affordances are the sole exception, and are advisory.** If `gert-vscode` later wants
   `yamlValidation` squiggles, it MUST obtain the schema as a build-time artifact from the engine
   repo with a recorded SHA-256, MUST fail its own build on digest mismatch, and MUST mark the
   resulting diagnostics as advisory (severity Information/Warning), never as an execution gate.
   Absent that pipeline, it ships no schema. **Today it ships none: correct, keep it that way.**

**Consequence for the report:** "the schema rejects enum in the clients" was never true because no
client has a schema. That is the property to preserve, not an accident to fix.

## 2. AR-CE-2 — Declaration carriage: the enum must reach the client, in declared order

`enum` is contract metadata attached to a declaration (AR-ENUM-10). Clients render declarations.
So the declaration DTO MUST carry it.

**Normative — the input-declaration DTO (new, D-1):**

The engine MUST expose runbook input declarations to clients, at minimum through the preview
document and the run-start precondition path, with these fields per input:

| Field | Type | Rule |
|---|---|---|
| `name` | string | declared key |
| `type` | string | as declared; absent means `string` (AR-ENUM-2) |
| `required` | bool | as declared |
| `default` | any, omitempty | as declared; **omitted when redacted** |
| `description` | string, omitempty | as declared |
| `enum` | `[]string`, omitempty | **declared order, verbatim, unsorted, undeduplicated** (AR-ENUM-5) |
| `enumRedacted` | bool, omitempty | true ⇒ `enum` MUST be omitted entirely |
| `enumMemberCount` | int, omitempty | present **only** when `enumRedacted` |

**Normative — `InteractionField` (D-2):** add `enum []string`, `enumRedacted bool`,
`enumMemberCount int` with the identical rules. **`enum` MUST NOT be flattened into `options`.**
AR-ENUM-1 kept collector `options:` (`{value,label}`, UI-driven) and `enum` (values-only,
contract-driven) deliberately separate; merging them at the wire layer re-unifies at the DTO what
the ruling separated at the schema, and would let a UI-only `options` edit silently look like a
contract change. Two fields, distinct names, distinct meanings.

**Absence rule (AR-ENUM-11 restated for the wire):** an unconstrained string carries **no** `enum`
key. It MUST NOT be encoded as `[]`, `null`, `["*"]`, or a wildcard. `"enum": []` on the wire is a
protocol error, not "any string".

## 3. AR-CE-3 — String selector vs. validation fallback

**Ruling: selector when a member list is available, validation fallback in every other case. The
selector is an affordance; the server check is the contract.**

1. When an input declaration carries a non-redacted `enum`, a client that renders a per-input form
   MUST render a **closed selector** (dropdown / list / radio group) over the members, **in declared
   order**, with no client-side sorting, case-folding, deduplication, relabelling, translation, or
   truncation of the member strings.
2. The selector MUST be closed: it MUST NOT accept free text that is not a member, and it MUST NOT
   offer "other…". A nearest-match/"did you mean" suggestion MAY be *displayed* after a server
   rejection; a client MUST NEVER substitute it (AR-ENUM-15 §10).
3. **Validation fallback is mandatory, not optional**, in each of these cases, and in each the
   client submits a free-text (or programmatic) value and lets the server adjudicate:
   - `enumRedacted: true` (C1 — see §6);
   - the surface is non-declarative (JSON-RPC `run.start` `inputs` map, `--var`, HTTP `POST /runs`);
   - the client's engine build is older than the DTO and the field is absent;
   - the client cannot render a selector for that input for any other reason.
4. **A client MUST NOT refuse to submit** a value the server has not yet judged. Client-side
   pre-validation is advisory UX only (§4).
5. **No implicit selection.** A required input with no `default` MUST open with nothing selected.
   Auto-selecting the first member fabricates a binding the operator never made — during an
   incident, silently. Prohibited.
6. The GUI's current raw-JSON inputs box (`preview.html:912-915`) is the fallback surface and MAY
   remain, but it MUST NOT be the only surface once the DTO exists.

## 4. AR-CE-4 — Validation timing and error surfacing

**Ordering, normative and unchanged from AR-ENUM-7:** coercion → type check → NFC → membership.
Clients participate in **none** of these authoritatively.

1. **The engine is the sole enum authority at every binding moment.** A client MUST NOT treat its
   own check as sufficient, and MUST NOT skip submission because it believes the value is valid.
2. A client MAY pre-validate for UX. If it does, it MUST use exactly the ruled algorithm — NFC
   normalise the candidate, compare codepoint-wise against the declared member, case-sensitively —
   or it MUST NOT pre-validate at all. A near-miss client algorithm is worse than none: it produces
   a disagreement between two surfaces about the same runbook, which is the failure mode this whole
   feature exists to eliminate.
3. **Client pre-validation is non-blocking.** It renders an inline hint. It never converts to a
   fatal, never invents an error code, and never reuses an `ENUM-0NN` code for a client-originated
   message. `ENUM-*` codes are emitted by the engine only.
4. **Server errors are rendered with their code.** A client MUST surface the engine's `code` and
   `message` (and `remedy` where the envelope carries it) rather than a paraphrase.
5. **D-3 is a defect and MUST be fixed structurally:** the JSON-RPC/HTTP error path MUST classify
   via `errkit.Coder` (`pkg/errkit/errors.go:15,55`), not by lowercased message substring
   (`internal/serve/rpc.go:876-894`), and MUST carry the code to the client — for JSON-RPC, in
   `error.data.code`. An ENUM-008 MUST map to an invalid-input class, **never** to
   `rpcRunbookParseErr`/"Parse error". Message-substring classification of coded errors is banned
   going forward; it is how D-3 happened.
6. **ENUM-W001 MUST reach a human on every surface that parses a runbook.** `pkg/run.Start` MUST
   return `parsed.Warnings` to its caller (D-4), and `gert-tui` MUST display them. AR-ENUM-12 makes
   the warning non-fatal, not invisible.
7. Plan-time codes (ENUM-001..007) surface as validation failures of the *file*; runtime codes
   (ENUM-008/009) surface as failures of the *value/run*. A client MUST NOT merge these two
   presentations — that conflation is exactly what D-3 does today.

## 5. AR-CE-5 — Strings, NFC, and case on the client

1. **Free-text values are transmitted verbatim.** A client MUST NOT trim, collapse whitespace,
   case-fold, or NFC-normalise a user-entered value before submission. The engine normalises the
   candidate (`pkg/schema/enum.go:162-179`); a client that pre-normalises hides the operator's
   actual input from the trace and from the diagnostic.
2. **Selector values are transmitted byte-verbatim as received in the DTO.** No re-encoding, no
   round-trip through a display transform.
3. **Preselection/echo matching MUST compare NFC-normalised codepoints**, not raw bytes. The TUI's
   exact `opt.Value == v` preselect (`internal/interaction/multiform.go:43-49`) is the wrong shape
   for enum data arriving from outside the file and MUST be normalised on the comparison path only.
4. **Rendering MAY normalise for display; storage and submission MUST NOT.**
5. **Case-only-distinct members are legal** (AR-ENUM-4) and MUST remain visually distinguishable.
   Clients MUST NOT case-insensitively sort, dedupe, or filter the member list; a type-ahead filter,
   if any, MUST be case-sensitive or MUST be purely additive (never removes a member from reach).
6. Clients MUST NOT strip or "sanitise" control/bidi characters out of a member for display in a way
   that makes two distinct members render identically without an indication. Such members cannot
   occur in a valid runbook (ENUM-005), so this is a defence-in-depth rule for untrusted files.

## 6. AR-CE-6 — Required, default, absence, redaction

1. **`enum` does not imply `required`** (AR-ENUM-6). An optional, defaultless input MUST have an
   affordance for *absent* that is distinct from the empty string. Empty string is never a member
   (ENUM-003), so "cleared selector" MUST NOT be submitted as `""` for a required input.
2. **`default` preselects, and only if it is a member.** ENUM-006 guarantees it is at plan time; a
   client that receives a non-member default MUST render nothing preselected and MUST NOT repair it.
3. A required input with no default: nothing preselected, submission blocked by the *client form's
   own required rule* (not by an enum rule), or submitted absent and rejected by the engine.
4. **Redaction (C1, non-negotiable):**
   - `type: secret` never carries an enum (ENUM-001); a client that sees one MUST treat the payload
     as malformed and MUST NOT render members.
   - When `enumRedacted` is set, the client MUST render a **free-text** control, MAY state
     "one of N permitted values" using `enumMemberCount`, and MUST NOT display, log, cache,
     autocomplete, or place in browser history any member.
   - A client MUST NOT render the literal `"<redacted>"` as if it were a member. It is a marker.
   - Rejection messages for a redacted declaration MUST NOT be enriched client-side with a member
     list or a nearest-match suggestion.
   - `EnumMeta.Redacted` remains a best-effort proxy (ticket **T-ENUM-SENSITIVE-DECL**) and MUST NOT
     be documented to operators as a guarantee.

## 7. AR-CE-7 — Trace and JSON implications

1. **Forward compatibility is mandatory.** Clients MUST tolerate unknown keys in event payloads,
   plan records, and DTOs. The current transports already do (`Payload map[string]any`,
   `pkg/serve/serve.go:84-90`; `runstate.Apply` switches on known kinds only,
   `pkg/preview/runstate/state.go:177-257`); no client may regress this by adopting a strict decoder.
2. `enum_constraints` appears **once per run**, inside the `plan.validated` payload
   (`internal/engine/engine.go:339`). Clients MUST NOT expect enum metadata on `step/*` or `tool/*`
   events and MUST NOT synthesise it there.
3. A client that wants declaration metadata for a *live* run SHOULD read it from `plan.validated`
   rather than re-reading and re-parsing the runbook file, which may have drifted since planning.
4. **`--output=json` run-summary shape is unchanged** (AR-ENUM-10). No client may depend on enum
   metadata appearing there; enum failures arrive as ENUM-00x error objects in the existing envelope.
5. Redacted declarations appear in the trace as `"<redacted>"` plus `member_count`. Any client that
   renders trace/plan metadata MUST honour §6's marker rule there as well.
6. Trace rendering MUST NOT reorder members. Declared order is the only order a human sees.

## 8. AR-CE-8 — Shared schema or parity tests? Per-repo, by topology

The question is answered differently for each client because the repos differ, and pretending
otherwise is how a monorepo-shaped answer gets imposed on a three-repo world.

| Client | Topology today | **Ruling** |
|---|---|---|
| **Web GUI** (`gert/internal/serve/static/preview.html`) | Same repo, same binary as the engine; assets are `go:embed`ed | **Shared by construction.** The DTO *is* the contract. No schema in the page, no JS-side member validation, no second copy of anything. Guarded by Go tests in-repo. |
| **gert-tui** | Separate repo, but `replace github.com/ormasoftchile/gert => ../gert` (`go.mod:5-8,53`), no vendor dir; compiles the engine's own `pkg/schema` | **Shared by construction — and this MUST be preserved.** A TUI-local runbook struct, YAML decode, or schema copy is forbidden. **One parity test** in `gert-tui` asserting it renders a selector from the engine's declared `Enum` field and preselects by NFC comparison. Nothing more; a second corpus here would be a fork with extra steps. |
| **gert-vscode** | Separate repo, TypeScript, no Go, no schema, pure pass-through to the CLI/server | **Neither vendor nor duplicate. Parity tests only, and thin ones.** It has no enum semantics to get wrong. Its obligations are: forward the file, surface engine diagnostics with codes, do not parse. Two node tests suffice (§9 CE-V-01/02). |

**Rejected alternatives, with reasons:**

- **A published shared schema package (npm/Go module) consumed by all three.** Rejected for this
  cycle. There is no publish pipeline, `gert-private` is design-only, and a package introduces a
  *version skew* axis that does not exist today — the TUI would then have a schema version and an
  engine version that can disagree. Revisit only if a client ever needs to validate without the
  engine present.
- **A git submodule of `gert/schemas`.** Rejected: same skew problem, plus submodules across three
  repos with different release cadences are a well-known drift source.
- **Re-implementing enum checking in TS/JS for editor UX.** Rejected outright (AR-CE-4 §2). Two
  implementations of NFC + codepoint equality is precisely the divergence the corpus exists to
  prevent, and the payoff is a squiggle.

**Anti-drift strategy — four mechanisms, in force order:**

1. **Structural (strongest): no client owns a schema, a struct, or a validator.** Drift is
   impossible where a copy is impossible. This is the primary defence and everything else is backup.
2. **DTO additivity:** new declaration metadata is added as optional fields; clients tolerate
   unknown keys; absence means "older engine", never "unconstrained".
3. **Executable parity, not mirrored logic:** the acceptance matrix (§9) is run against the **real**
   engine/CLI/server, exactly as `internal/conformance/enum_harness.go` does. Clients assert
   *rendering and transport*; the engine asserts *semantics*. No client re-asserts semantics.
4. **Corpus stays single-sourced:** `design/gert/conformance/tv-enum.yaml` remains frozen and
   authoritative; the vendored engine copy stays SHA-256 identical. Client matrices reference vector
   IDs; they do not copy vectors.

## 9. AR-CE-9 — Acceptance test matrix

Every row is executed against a real engine build. `S` = surface (G = web GUI/serve, T = TUI,
V = VS Code, E = engine/shared).

| ID | S | Scenario | Expected |
|---|---|---|---|
| CE-S-01 | E | Runbook with `inputs.x.enum: ["prod","staging"]` parses | Accepted; no error, no warning |
| CE-S-02 | G,T,V | Same runbook loaded in each client | Accepted; no client-originated diagnostic |
| CE-S-03 | V | Extension opens the file with no engine present | No schema squiggle (no bundled schema) |
| CE-D-01 | G | Preview document for the runbook | Carries `inputs[]` with `enum` in **declared order** `["prod","staging"]` |
| CE-D-02 | G | Reordered members `["staging","prod"]` | DTO order matches file order; no sorting |
| CE-D-03 | G,T | Input with no `enum` | `enum` key **absent** (not `[]`, not null) |
| CE-D-04 | G,T | `InteractionField` for a collector with `options` and an enum-constrained input in the same runbook | `options` and `enum` are distinct fields; neither leaks into the other |
| CE-R-01 | G,T | Enum input rendered | Closed selector, members in declared order, nothing preselected when required + no default |
| CE-R-02 | G,T | Enum input with `default: staging` | `staging` preselected |
| CE-R-03 | G,T | Optional enum input, no default | Explicit "unset" affordance, distinct from `""`; submitting unset sends no value |
| CE-R-04 | G,T | Members `["Prod","prod"]` (ENUM-W001) | Both rendered, visually distinct, no dedupe; run proceeds |
| CE-R-05 | G,T | `enumRedacted: true`, `enumMemberCount: 3` | Free-text control; no member displayed anywhere; count MAY be shown; `"<redacted>"` never rendered as a member |
| CE-R-06 | G,T | 40-member enum | All members reachable; no truncation of the reachable set |
| CE-V-01 | E,G,T | Submit `"not-a-member"` for an enum input | ENUM-008; binding does not occur; value absent from trace as accepted |
| CE-V-02 | G | Same, over JSON-RPC `run.start` | Error carries `ENUM-008` in `error.data.code`; class is invalid-input, **not** "Parse error" (D-3 regression test) |
| CE-V-03 | G | Same, over `POST /runs` | Non-2xx with the code present in the body |
| CE-V-04 | V | Engine emits ENUM-008 while the extension is driving the server | Code and message reach the output channel verbatim |
| CE-V-05 | G,T | Client pre-validation present | Advisory only; submission still occurs; no client-invented code |
| CE-V-06 | E | Selector value submitted verbatim | Round-trips byte-identical to the declared member |
| CE-U-01 | E,G,T | NFD candidate vs NFC member (cf. `TV-ENUM-UNICODE-001`) | Accepted; client did **not** pre-normalise before sending |
| CE-U-02 | G,T | Free-text `" prod"` (leading space) | Sent verbatim; engine rejects ENUM-008; client did not trim |
| CE-U-03 | G,T | Free-text `"PROD"` against member `"prod"` | ENUM-008; no case-folding anywhere |
| CE-U-04 | T | Preselect a member supplied in NFD form | Matches by NFC comparison, not raw bytes (`multiform.go:43-49` regression test) |
| CE-U-05 | G,T | Non-ASCII members (`"café"`, CJK) | Rendered and submitted intact |
| CE-T-01 | G,T,V | Run with an enum input | `enum_constraints` present exactly once, in `plan.validated`; absent from `step/*` and `tool/*` |
| CE-T-02 | G,T,V | Event payload gains an unknown key | No client error; unknown key ignored |
| CE-T-03 | G | Redacted enum in `plan.validated` | Renders as redacted marker + count; never as a member named `<redacted>` |
| CE-T-04 | E | `--output=json` run summary | Byte-shape unchanged vs. pre-enum baseline |
| CE-W-01 | T | Runbook with case-only-distinct members | ENUM-W001 visible to the operator; run not failed (D-4 regression test) |
| CE-W-02 | G | Same via serve | Warning reaches the client surface |
| CE-C-01 | T | Static assertion | No YAML decode, no runbook struct, no schema file in `gert-tui` |
| CE-C-02 | V | Static assertion | No `*.schema.json`, no `jsonValidation`/`yamlValidation`, no YAML parser dependency in `gert-vscode` — unless the §1.4 digest-checked pipeline exists, in which case the digest test replaces this row |
| CE-C-03 | E | Static assertion | Vendored `enumdata/tv-enum.yaml` SHA-256 == `design/gert/conformance/tv-enum.yaml` |

**Gating:** CE-V-02, CE-U-04, CE-W-01, CE-C-01, CE-C-02, CE-C-03 are regression tests for defects
found in this review and MUST be red before the fix and green after.

## 10. AR-CE-10 — Ratified non-goals for client enum support

1. Client-side enum validation as a gate. Advisory only, or absent.
2. Any client-owned runbook schema, struct, or YAML decoder (excepting §1.4's advisory,
   digest-checked editor schema).
3. Unifying `enum` with collector `options:` at the DTO or UI layer (AR-ENUM-15 §3, restated).
4. Member labels, descriptions, icons, grouping, i18n in any client (AR-ENUM-15 §4).
5. Client-side "did you mean" auto-correction or auto-selection (AR-ENUM-15 §10).
6. Enum-driven selectors for `capture:`, `vars:`, `toolRefs`, provider fields, or any site outside
   AR-ENUM-1's four (S1–S4).
7. A published cross-repo schema package or submodule this cycle (§8).
8. Client rendering of tool-action arg enums (S1) beyond what the interaction DTO already carries;
   the caller-facing MVP surface is S3 (runbook inputs).

---

## 11. Tickets opened by this ruling

| Ticket | Repo | Summary |
|---|---|---|
| **T-CLIENT-ENUM-DTO** | `gert` | Add the input-declaration DTO (§2) to the preview document and add `enum`/`enumRedacted`/`enumMemberCount` to `InteractionField` |
| **T-SERVE-ENUM-ERRCODE** | `gert` | Classify RPC/HTTP errors via `errkit.Coder`, carry `error.data.code`; ENUM-008 ≠ "Parse error" (D-3) |
| **T-RUN-WARNINGS** | `gert` | `pkg/run.Start` must return `parsed.Warnings` (D-4) |
| **T-GUI-INPUT-FORM** | `gert` | Declaration-driven input form in `preview.html`; keep raw-JSON box as fallback (D-1) |
| **T-TUI-ENUM-SELECT** | `gert-tui` | Render enum inputs via the existing multiform select; NFC-safe preselect (D-5, §5.3) |
| **T-TUI-WARNINGS** | `gert-tui` | Display `Warnings` once `T-RUN-WARNINGS` lands (CE-W-01) |
| **T-VSCODE-DIAG-PASSTHROUGH** | `gert-vscode` | Ensure engine codes reach the output channel verbatim; add CE-C-02 static test |

Pre-existing tickets unchanged and still open: `T-ENUM-ROOT-OUTPUTS`, `T-ENUM-FROM-SOURCING`
(note: this one is *why* the TUI has no caller-binding path — D-5 is its downstream symptom, not a
separate cause), `T-ENUM-SENSITIVE-DECL`, `T-ENUM-REPLAY-WIRE`.

## 12. Routing recommendation

| Owner | Work | Sequencing |
|---|---|---|
| **Don or Ken** (Backend, `gert`) | T-CLIENT-ENUM-DTO, T-SERVE-ENUM-ERRCODE, T-RUN-WARNINGS | **First, and strictly first.** Every client task is blocked on the DTO and the error code. One backend dev, not two — the three touch the same serve/run seam. |
| **Leslie** (Frontend) | T-GUI-INPUT-FORM, T-TUI-ENUM-SELECT, T-TUI-WARNINGS, T-VSCODE-DIAG-PASSTHROUGH | After the DTO lands. Web GUI first (it defines the rendering precedent), TUI second (reuses the ruling, different widget), VS Code last and smallest. |
| **Tess** (Conformance) | Encode §9 as an executable matrix in the existing harness style; do **not** extend `tv-enum.yaml` (frozen) — client rows are a separate client-parity file referencing vector IDs | Can start in parallel with the backend work; CE-C-* rows are runnable today. |
| **Edith** (Spec) | Normative prose in `14-adapter-contracts.tex` (§JSON-RPC `exec/v1` error codes, §Contract Versioning, §Forward and Backward Compatibility) for AR-CE-1..7; one cross-ref line in `16-observability-diagnostics.tex` | After the backend DTO shape is final; specifying an unbuilt DTO invites a third source of truth. |
| **Barbara** | Gate review before any client merge | Two things I will check specifically: that no client acquired a schema/struct/validator, and that CE-V-02 and CE-U-04 exist as real regression tests rather than assertions about intent. |

**Cross-repo note:** `gert-tui` carries its own `.squad`. Work spanning `gert` and `gert-tui` MUST
be coordinated from this root (`gert-private`) with the engine change merged first, because the
`replace ../gert` path means a TUI build silently picks up whatever is in the engine working tree —
convenient during development, and a genuine trap if the two are edited concurrently.

## 13. Handoff

**→ Don/Ken.** Build the DTO and the error mapping before anyone touches a pixel. The DTO shape in
§2 is normative: declared order, omit-when-absent, redaction fields separate from members, `enum`
never folded into `options`. Kill message-substring error classification while you are in
`mapRunbookError` — it is the mechanism that produced D-3, and it will produce D-6 otherwise.

**→ Leslie.** You are not writing a validator. You are rendering a list and displaying a server's
verdict. If you find yourself comparing strings for membership anywhere except selector preselection
(NFC, §5.3), stop and write to my inbox. The widgets you need already exist in both surfaces —
`preview.html:1402-1417` and `internal/interaction/multiform.go:39-57`.

**→ Tess.** §9 rows only, against the real engine, in the harness style Ken established. The
enum corpus stays frozen; client parity lives in its own file and cites vector IDs.

**→ Edith.** Nothing to write until the DTO is real. Then `14-adapter-contracts.tex`, and state the
redaction rules (§6.4) as MUST language rather than a note — same instruction, same reason, as at
the AR-ENUM gate.

**No architecture is reopened.** AR-ENUM-1..15 and C1/C2/C3 stand; AR-CE-1..10 extend them to the
client boundary and add nothing to the engine's semantics.
