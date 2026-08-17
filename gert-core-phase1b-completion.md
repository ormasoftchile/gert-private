# Gert Core → SQL Live-Site Operations
## Phase 1B Completion

**Date:** 2026-08-17  
**From:** Barbara (Lead/Architect, Gert Core)  
**Final commit:** `05fd13b` (pushed to `origin/main`)  
**Status:** Phase 1B code-complete. All items delivered. Reproducibility remediation done.

---

## 1. Reproducibility Remediation

You gave us seven conditions for a reproducible release. Here is where each stands against commit `05fd13b`:

| # | Condition | Status | Evidence |
|---|-----------|--------|----------|
| 1 | All required source and test files committed | **Satisfied** | 0 untracked files in clean checkout of `05fd13b` |
| 2 | Clean checkout with no untracked build inputs | **Satisfied** | Detached worktree of `05fd13b`, tracked files only, 0 untracked |
| 3 | `go build ./...` passing from clean checkout | **Satisfied** | Exit 0 |
| 4 | `go test ./...` passing from clean checkout | **Satisfied** | Exit 0, 63 packages, 0 FAIL |
| 5 | CI evidence produced from the clean checkout | **Not yet observed by us** | `.github/workflows/go-test.yml` is tracked and triggers on push across an OS matrix. GitHub Actions checks out only tracked files, so a passing run constitutes clean-checkout evidence by construction. However: our `gh` CLI is authenticated to an account that cannot read this repository's Actions runs. We have not seen the result. We will confirm it rather than assert it. |
| 6 | Exact commit containing the complete implementation | **Satisfied** | `05fd13b` |
| 7 | Every cited capability reachable from that commit alone | **Satisfied** | `TestCLI_ReachabilityGate` passes; `TestEnumConformance` 62 PASS / 10 SKIP / 0 FAIL / 72 total |

The feature-reachability gate is itself committed (`56b1bc4`) and demonstrated from the clean checkout (item 7 above).

---

## 2. Requested Deliverables

### 2.1 Reusable contract-parity test helper

**Path:** `pkg/contractparity`  
**Commit:** `a94f35a`

Exported package, importable by any Go module:

```go
import "github.com/<org>/gert/pkg/contractparity"
```

A foreign repo adds `gert` as a module dependency, defines its contract fixture and bindings, and calls the harness API. No Gert-specific test code needs to be copied. 30 tests internally; 11 negative controls verify the harness is non-vacuous (detects injected violations).

### 2.2 Worked example: `--package-map` plus `--profile` during execution

**Commit:** `d53a45f` (made load-bearing in CLI proof), `05fd13b` (parity proof via public harness API)

`TestSyntheticContract_CLI_PackageMap_Profile_MCPHTTPBinding` composes both flags in a single `runRun()` execution (not `gert plan`). The profile supplies an endpoint and `provider: managed-identity`; a mock IMDS issues a token; the mock MCP server asserts it received exactly `Bearer SYNTHETIC_MI_TOKEN_CLI_PROOF`.

### 2.3 Documentation for contract-identical native, stdio, and HTTP bindings

**Path:** `docs/contract-identical-bindings.md`  
**Commit:** `e6521f6` (README-linked)

---

## 3. Acceptance Criterion 4 — Mechanism Proof

Criterion 4 requires "the exact SQL Live-Site contract passes native-mock and managed-identity HTTP bindings." This is jointly owned per our agreed split:

| Owner | Proves | Status |
|---|---|---|
| **Gert core** | The mechanism works on a synthetic contract (`ops-synthetic`) | **Done** |
| **SQL Live-Site** | Their exact contract passes under both bindings | Theirs to execute using the exported harness |

### What we proved

**Synthetic contract `ops-synthetic`:**
- Action A `status-query`: typed 5-field output (`id`, `name`, `status`, `region`, `health_score`)
- Action B `pattern-search`: two outcome shapes (match-found with `pattern` field; no-match without)
- Both classified `read-only`
- Bindings: native (`cmd/tools/ops-mock`) and `mcp-http` (mock MCP server)

**Parity tests (all PASS):**
- `TestSyntheticContractParity_ActionA_StatusQuery`
- `TestSyntheticContractParity_ActionB_MatchFound`
- `TestSyntheticContractParity_ActionB_NoMatch`
- `TestSyntheticContractParity_NegativeControl` — correctly detects an injected extra output key

**Composition proof:**
`TestSyntheticContract_CLI_PackageMap_Profile_MCPHTTPBinding` — `--package-map` and `--profile` in a single execution; profile supplies endpoint + `provider: managed-identity`; mock IMDS issues token; mock MCP server asserts `Bearer SYNTHETIC_MI_TOKEN_CLI_PROOF`.

**Mutation resistance (measured):**
- Remove `--profile` → exit 1 (endpoint falls back to placeholder, dial fails)
- Remove `--package-map` → exit 2 (PKG-011, package unbound at planning)

These are the strongest evidence that the composition is load-bearing. A vacuous test would still pass with either flag removed.

**Invariants enforced:**
- PROF-001: profile must not rewrite transport mode
- Rule A: `ProfileToolOverride` has no `Scope` or `AllowedHosts` fields (structurally impossible to clobber TokenGate)
- PLAN-013: endpoint outside `allowed_hosts` fails at planning, not mid-execution
- Credential non-leakage: extended to the synthetic `mcp-http` + managed-identity path

No production credentials are required by any of this.

---

## 4. Acceptance Criteria

| # | Criterion | Status | Evidence |
|---|-----------|--------|----------|
| 1 | Managed identity and workload identity are not ambiguously combined | **Met** | `0ce2054` — IMDS-only provider; `workload-identity` returns MCP-002 by design. Explicit test asserts WI still fails. This is an intentional design outcome of your instruction, not a gap. |
| 2 | Profile/provider/endpoint compatibility fails closed | **Met** | `85bfa4a` — PLAN-013 rejects override endpoint whose host ∉ `allowed_hosts` at plan time |
| 3 | Endpoint overrides respect `allowed_hosts` | **Met** | Same commit; TokenGate constructed exclusively from tool definition's `AllowedHosts`; Rule A enforced structurally |
| 4 | The exact SQL Live-Site contract passes native-mock and managed-identity HTTP bindings | **Jointly owned** | Gert half done (§3 above). Your half: run your contract through the exported `pkg/contractparity` harness. We hand you: the harness, the worked example, and the bindings guide. |
| 5 | Package-map/profile composition demonstrated | **Met** | `d53a45f`, `05fd13b` — `TestSyntheticContract_CLI_PackageMap_Profile_MCPHTTPBinding`; mutation resistance measured |
| 6 | INDETERMINATE state and acknowledgment tested | **Met** | `22ad3e7` — `*IndeterminateRecord` with 7 evidence fields; classification-aware timeout divergence (INDET-005); `--acknowledge-indeterminate` required for resume |
| 7 | Profileless non-interactive execution fails immediately | **Met** | `d1314cc` — fails at startup before approval gate construction; `os.Stdin.Stat()` check fails closed |
| 8 | No credential appears in runbook state, results, traces, or errors | **Met** | `c7decfd` — 13-surface sweep with sentinel token; 8-case negative control ensures the sweep cannot pass vacuously; extended to synthetic mcp-http path in `05fd13b` |

---

## 5. What Remains Outside Gert Core's Ownership

| Item | Owner | Notes |
|---|---|---|
| CI evidence confirmation (reproducibility item 5) | **Us** | We will confirm the Actions run result once we have read access. We do not ask you to wait on this — the local clean-checkout evidence is independently verifiable. |
| Consumer contract proof (criterion 4, your half) | **You** | Import `pkg/contractparity`, define your contract fixture and bindings, run the harness. The guide is at `docs/contract-identical-bindings.md`. |
| Workload identity | **Deferred** | Returns MCP-002. Will remain so until its agreed later phase. This is your instruction implemented, not a missing feature. |
| Live/production ICM validation | **You** | Separate integration gate under your control. |
| VS Code host bridge / Phase 2 | **Future** | Not in scope. |

We need nothing from you to close Phase 1B on our side. If you need anything from us to run your half of criterion 4, ask.
