# Gert Core — Phase 2 Rev 3 Status Report

**Date:** 2026-08-18  
**Prepared by:** Don (Gert Core / Go), on behalf of the Gert engineering team  
**For:** SQL Live-Site Operations

---

## Verdict

All five gaps from Phase 2 Rev 3 have code on `main` in both repos. The ICM
binding is complete: explicit package-map selection wired into `serve`, the
`vscode_input` mapping block implemented and validated fail-closed, the real
provider tool name `mcp_icm_mcp_serve_get_incident_details_by_id` in place.
Two limitations must be understood before you take this to production: `serve`
cannot honour package-map `requires:` bindings and does not silently ignore
them — it fails at startup; and the TSG recommendation binding has no `args:`
because we were never given that contract. Both are described in the
**Limitations** section below. No real end-to-end run through a live VS Code
session has occurred; all evidence is from tests and the stub bridge.

---

## Gap Status

### Gap 1 — Explicit binding selection (`serve --package-map`)

**Status: Shipped.** `gert serve` now accepts `--package-map <path>`, honours
`tool-paths:` bindings with the same precedence semantics as `gert run` and
`gert plan` (the explicit map wins over filesystem walk order), and **rejects
at startup** any package-map that contains a non-empty `requires:` section.
`requires:` is structurally unsupported in `serve`; see the Limitations section
for the exact error, exit class, and reasoning. A package-map that uses only
`tool-paths:` works without restriction.

- The flag is validated at startup, before the listener binds. A missing or
  malformed file exits immediately with `exitValidation`; `gert serve` does not
  reach the accept loop.
- Extra tool paths from the map override the base registry via
  `OverlayRegistry.Override`, so the explicit map always wins over filesystem
  walk order. `TestServe_PackageMap_DeterminesToolBinding` proves this: with two
  contract-identical tool definitions present, the map-selected definition
  binds; removing the override call causes the test to fail with:

  > `got tool version "2.0.0", want "1.0.0": scan order is deciding the binding
  > instead of the explicit package-map path`

- Extension side (for completeness): `848e720` on `gert-vscode main`.

**Core SHAs:** `4da94b2` (flag + binding), `81e8c35` (requires: guard — see
Limitations).

**Limitation — `requires:` is not honoured.** See Limitations section.

---

### Gap 2 — Loopback launch address

**Status: Shipped by the extension team.** `localServerAddress()` in the
extension now binds `127.0.0.1:<port>` rather than the previous form.

**Extension SHA:** `848e720` on `gert-vscode main`.  
*Core has no change for this gap.*

---

### Gap 3 — Active-project bridge registry

**Status: Shipped by the extension team.** The bridge registry is now derived
from the active runbook's resolved project via `pickServerRoot`, replacing the
prior `workspaceFolders[0]`. A folder-order-reversal test proves that the
selection does not depend on workspace enumeration order.

**Extension SHA:** `848e720` on `gert-vscode main`.  
*Core has no change for this gap.*

---

### Gap 4 — Declarative provider input adaptation (`vscode_input`)

**Status: Shipped.** The `vscode_input` block is now a first-class schema
field on `ToolAction`. It maps logical argument names to provider (MCP)
parameter names, with optional type coercion and required-ness.

**Contract shape (as specified by you; we did not invent it):**

```yaml
vscode_input:
  incidentId:               # KEY = provider (MCP) parameter name
    from: incident_id       # logical argument name from the action's args:
    coerce: integer         # optional
    required: true          # optional, default false
```

**Thread:** The field is live through all five required chain points:
`pkg/schema/tool.go` → `pkg/tool/tool.go` →
`internal/tool/validate_transport.go` → `internal/tool/scan.go` →
`internal/tool/runtime.go`. Core applies the mapping and coercion *before* the
adapted arguments are placed on the bridge wire. The extension validates the
adapted arguments against the live registered `inputSchema` after receipt; that
is the extension's job and we do not duplicate it.

**Fail-closed behaviour — each is tested and has a mutation:**

| Failure mode | Detected at | Test |
|---|---|---|
| `from:` names an arg the action does not declare | scan / plan time | `TestValidateVSCodeInputActions_FromNonExistentArg` |
| `vscode_input` on a non-`vscode-mcp` transport | scan / plan time (VSCODE-002) | `TestValidateVSCodeInputActions_WrongTransport` |
| `required: true` and value absent at invocation | before bridge call | `TestVSCodeInputAdaptation_RequiredAbsent` |
| `coerce: integer` and value is not losslessly an integer | before bridge call | `TestVSCodeInputAdaptation_CoerceIntegerLossy` (`"12.5"` → error), `_CoerceIntegerLossyFloat` (`float64(12.5)` → error), `_CoerceIntegerEmptyString` (`""` → error) |
| A logical argument supplied but not referenced by any mapping's `from:` | before bridge call | `TestVSCodeInputAdaptation_UnmappedArg` |
| Unsupported `coerce` value | scan / plan time | `TestValidateVSCodeInputActions_BadCoerce` |
| `vscode_input` absent | — | Backward compatible; args pass through unchanged (`TestVSCodeInputAdaptation_PassThrough`) |
| Optional arg absent | — | No error, key absent from adapted args (`TestVSCodeInputAdaptation_OptionalAbsent`) |

`"7"` → `int64(7)` (lossless string-to-integer, `TestVSCodeInputAdaptation_CoerceIntegerFromString`) and `float64(7.0)` → `int64(7)` (lossless float-to-integer, `_CoerceIntegerFromFloat64Lossless`) are permitted. Lossy conversion is an error, not a truncation.

No ICM-specific code exists in `pkg/` or `internal/`. The ICM tool definition
is in `testdata/`; the mechanism is general.

**ICM fixture** (`internal/tool/testdata/icm.vscode-mcp.tool.yaml`,
`gert-private/deliverables/phase2/icm.vscode-mcp.tool.yaml`):

- `vscode_tool: mcp_icm_mcp_serve_get_incident_details_by_id` — the real
  registered MCP tool name, as provided by you.
- Maps logical `incident_id: string` → provider `incidentId: integer`.

**Core SHAs:** `4da94b2`, `5249733` (testdata sync).

---

### Gap 5 — Failure semantics (fail-stop)

**Status: Shipped.** `ba9d6d4` is the fail-stop remediation delivered during
this engagement in response to the fail-open defect you reported. It was
accepted as the named baseline after three rounds of review and carried
unmodified into `19f31cf`'s ancestry. The revision was not disturbed by this
work.

---

## Evidence and Reproduction

All verification was performed from **detached worktrees** of the exact commit
SHAs, not from working trees. Commands run were:

```
git worktree add --detach <path> <sha>
cd <path>
git status --porcelain   # confirmed empty
go build ./...           # exit 0
go test ./...            # exit 0
git worktree remove --force <path>
```

**`gert` at `d0d8aba` (the branch HEAD, merged into `19f31cf`):**
- `git status --porcelain`: empty
- `go build ./...`: exit 0
- `go test ./...`: exit 0 — **63 ok / 23 no test files / 0 FAIL**
- Verified independently by the lead engineer before merge.

**`gert-vscode` at `848e720`:**
- `git status --porcelain`: empty
- `npm ci` / `npm run compile` / `npm test`: all exit 0 — **118 / 118 pass**
- Verified independently by the lead engineer before merge.

Both merged trees (`19f31cf` on `gert`, `f1e5c51` on `gert-vscode`) were
confirmed byte-identical to the verified branch heads before push.

**Deliverable fixture parity** is verified by
`gert-private/scripts/check-deliverable-parity.js` at `6c90524`. Invocation:

```
node scripts/check-deliverable-parity.js <path-to-gert-checkout>
# or
GERT_CHECKOUT=<path> node scripts/check-deliverable-parity.js
```

Measured exit codes: 1 when the path is absent, 1 with a bad path, 0 against a
real checkout. It does not skip when its input is missing.

---

## Limitations

### `serve --package-map` does not honour `requires:` bindings

`gert run` and `gert plan` resolve `requires:` entries via
`BuildPackageCatalog(catOpts, runbookPath, parsed.Runbook.Requires)`, which is
inherently per-runbook: it knows which runbook it is executing. `gert serve`
pre-builds one engine at startup with no runbook in scope; `requires:` cannot
be resolved without a runbook, and we cannot resolve it per-request without an
architectural change that was out of scope for this engagement.

We did **not** silently discard `requires:`. Silently discarding it would allow
`gert plan --package-map X` to report one binding while `gert serve
--package-map X` executed a different one — a plan/execute mismatch at runtime,
precisely the failure class this engagement exists to eliminate.

Instead, `serve` refuses to start if the package-map contains a non-empty
`requires:` section:

```
error: serve does not support package-map "requires:" bindings
  serve builds one engine at startup and cannot resolve per-runbook package
  requirements; only "tool-paths:" bindings are applied
  fix: use tool-paths: in the package-map passed to serve, or use
       `gert run`/`gert plan` for requires:-based resolution
```

Exit class: `exitValidation`. Exit occurs before the TCP listener binds.

The `--package-map` flag help text states this up front.

Test: `TestServe_PackageMap_RequiresRejected` (non-zero exit, error message
asserted). The `tool-paths:`-only path is confirmed working by
`TestServe_PackageMap_ToolPathsOnlyAccepted` (exit 0, no error output).

**Impact on ICM:** The ICM fixture uses only `tool-paths:`. This limitation
does not affect the ICM binding as specified.

### No real end-to-end run has occurred

The TSG recommendation MCP tool is not provisioned in any VS Code session
accessible to us. The ICM tool integration has not been exercised through a
live VS Code session. All evidence — binding selection, input adaptation,
coercion, fail-closed paths, bridge dispatch — comes from tests using the stub
bridge. We have not observed a live MCP round-trip.

### TSG recommendation: `args:` contract unknown

`tsg-recommendation.vscode-mcp.tool.yaml` intentionally declares **no
`args:`** for the `recommend` action. We were never given the provider
contract for this tool.

To implement the mapping we need, for the `recommend` action:
- The exact registered MCP tool name in the user's VS Code session.
- The names, types, and required-ness of all provider parameters.
- The corresponding logical argument names and types you want the runbook to
  use.
- Whether `recommend` takes any inputs at all, or is a no-argument call.

We will not guess any of these. Guessing a tool name is what caused the Rev 2
rejection of the ICM binding. Please supply the TSG recommendation tool
contract in the same form you supplied the ICM contract.

### TSG absence guard: test branch, not merged

The branch `squad/rev3-tsg-absence` on `gert-vscode` contains
`test/tsgAbsence.test.js` (at `e8613bd`) proving that an absent TSG tool is
fatal: the bridge returns `tool_unavailable` with an error body and no
`result`, core converts an error body into an `Invoke` error, and `ba9d6d4`
terminates the run. The mutation proves the gap: disabling the registry guard
causes the absent tool to return `{"recommendation_status":"no-suggestion"}` —
exactly the silent-wrong-outcome hazard you described.

**This branch is not merged.** The absence test cannot be merged without the
TSG contract being in place, because `tsg-recommendation.vscode-mcp.tool.yaml`
has no `args:` and therefore no `vscode_input` mapping. The absence-handling
mechanism is correct and tested; it is waiting on the contract.

---

## What We Need From You

1. **TSG recommendation provider contract** — see the TSG limitation above.
   Please supply: registered MCP tool name, all provider parameter names /
   types / required-ness, and the logical argument names you want the runbook
   to use.

2. **ICM live validation** — if possible, a live VS Code session with the ICM
   MCP server running so we can observe an actual bridge round-trip and confirm
   the adapted `incidentId: integer` reaches the provider correctly. We cannot
   provide this evidence ourselves.

---

## Reproducibility

We found and corrected a defect in our own test infrastructure that you did not
report. We are disclosing it here for the same reason we disclosed the earlier
reproducibility failure: you should know the state of the test suite you are
relying on.

**What was broken:** `gert/internal/tool/deliverable_contract_test.go`
resolved deliverable files via a relative path `../gert-private/deliverables/phase2/`
— a test in one repository depending on another repository's **working tree**,
including any uncommitted edits. When a detached worktree of `gert` is
checked out at a specific SHA (our required verification method), that path
resolves to the real `gert-private` working tree, not any committed state.
The test then called `t.Skipf` when the path was absent, so it never ran in
CI. Locally, it was reading uncommitted state and reporting it as test
evidence. This is what caused the `5249733` commit: another engineer updated
the deliverable in `gert-private` while we were working, and our test suite in
`gert` broke as a result — two repositories, no shared commit, coupled through
the filesystem.

This is the **eighth** occurrence of this bug class in this engagement —
parsed-but-unreachable fields and cross-repo test reads among them — and we now
have a committed guard against the latter.

**What is fixed:**
- `TestDeliverableContracts_ByteIdentical` and `deliverableDir` are deleted
  from `gert`. The remaining contract assertions
  (`TestDeliverableContracts_ICM`, `TestDeliverableContracts_TSGRecommendation`)
  read only from `internal/tool/testdata/`, which is committed in `gert`.
- `TestRepoBoundary_NoExternalPaths` (`d0d8aba`) walks every `*_test.go` file
  in the `gert` repository and fails on: (a) any file containing a
  sibling-repo name reference, or (b) any file containing the Go string literal
  `".."` that is not in an explicit allowlist. The allowlist has 11 entries,
  each with a stated reason. An unlisted `".."` use fails the build; the author
  must add an entry with a justification.
- Cross-repo deliverable parity is now checked from `gert-private` by
  `scripts/check-deliverable-parity.js` (`6c90524`). Invocation:

  ```
  node scripts/check-deliverable-parity.js <path-to-gert-checkout>
  # or
  GERT_CHECKOUT=<path> node scripts/check-deliverable-parity.js
  ```

  Measured exit codes: 1 with no path supplied, 1 with a bad path, 0 against a
  real checkout. It does not skip when its input is missing.

The guard has been verified by two mutations: a sibling-repo reference injected
into `cmd/gert/serve_package_map_test.go` causes the guard to fail naming that
file; an un-allowlisted `".."` injected into the same file causes the guard to
fail directing the author to the allowlist.

---

*All SHA references are on the repositories as of the push to `main` on
2026-08-18. Deliverable files: `gert-private` at `6c90524`.*
