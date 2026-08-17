# Gert Core → SQL Live-Site Operations
## Phase 1B Status and Revised Ownership (Rev 3)

---

## 1. Summary

Phase 1B is code-complete except for Item 4, which is blocked on one answer from you.

We are also withdrawing the artifact request from our previous message, including the request for access to `gert-sqllivesite`. The reasoning is in §4.

---

## 2. Delivered and verified

All commits are in the `gert` repository. `go build ./...` and `go test ./...` both exit 0 on the full suite.

| Item | Deliverable | Commit |
|---|---|---|
| 1 | `managed-identity` via IMDS | `0ce2054` |
| 2 | Profile-selected endpoint/auth reaches execution + PLAN-013 | `85bfa4a` |
| 3 | `INDETERMINATE` semantics and classification-aware timeout | `22ad3e7` |
| 5 | Profileless non-interactive execution fails fast | `d1314cc` |
| 6 | Credential non-leakage assertions | `c7decfd` |
| — | Feature-reachability merge gate | `56b1bc4` |

### Against your corrections

**Correction 1 — managed identity and workload identity are not combined.**
`managed-identity` is IMDS only. `workload-identity` continues to return MCP-002 and is deferred to its agreed later phase. The provider does not inspect ambient state and does not choose between mechanisms. There is an explicit test asserting `workload-identity` still fails, so the separation cannot regress silently. No Azure SDK dependency was added; IMDS is stdlib HTTP.

**Correction 4 — endpoint override safety fails closed at plan time.**
PLAN-013 rejects a profile endpoint override whose host is not in the tool definition's `allowed_hosts`, during planning, before step 1 executes.

Ratified rule, now structurally enforced: a profile may supply `Provider`. It may **not** supply `Scope` or `AllowedHosts`. The `TokenGate` is always constructed from the tool definition. A profile substitutes *who acquires* the token — never what it is scoped for, nor where it may be sent. A profile also cannot rewrite transport mode; that is tested (PROF-001).

We want to flag the sequencing explicitly, because it was the highest-risk step in this item: the override wiring and PLAN-013 landed in a **single commit**. Had they shipped separately, the interval between them would have executed endpoint overrides without validation.

**Correction 5 — INDETERMINATE.**
`StepStatusIndeterminate` and an `IndeterminateRecord` carrying all seven evidence fields you specified: run and step IDs, logical tool and action, classification, endpoint host (host only, never credentials), attempt number, deadline and failure time, and transport error category.

The record is a pointer. Non-nil means completion is unknown. This matters because `Output == nil` is ambiguous — a tool that legitimately returns nothing also yields nil. Where a timed-out tool provides no evidence, the system records that completion is unknown rather than fabricating output, per your instruction.

Timeout behavior is now classification-aware. Previously a **destructive** action timing out was handled identically to a **read-only** one. Test INDET-005 asserts the two paths now diverge and fails if they ever converge again.

No automatic retry occurs for mutating, destructive, or unspecified actions. Resume is refused without `--acknowledge-indeterminate`.

**Correction 6 — profileless CI/headless execution fails fast.**
You rejected our proposal to defer this to documentation and a warning. You were right, and it shipped in Phase 1B. Non-interactive execution with no profile now fails at startup, before any approval gate is constructed:

```
error: non-interactive execution requires an unattended runtime profile
fix: pass --profile <profile>
```

Detection is stdlib `os.Stdin.Stat()` and **fails closed** — if the check errors, the run is treated as non-interactive and a profile is required.

This changed the conformance harness, which previously invoked `gert run` profileless. We added an unattended test profile. Vector results are unchanged: **72 vectors, 62 pass, 10 skip, 0 fail**, identical before and after. We verified the counts directly rather than relying on the harness summary, because a changed vector would have meant the profile altered semantics.

**Your required validation — approval state never changes execution semantics.**
Orthogonality tests assert that flipping `RequiresApproval` between unspecified and explicit-false changes nothing about classification, retry, or late-result handling. `RequiresApproval: false` routes approval only. It never assigns, implies, or coerces `classification: read-only`.

**Acceptance criterion 8 — no credential leakage.**
A sweep runs a real managed-identity execution against a mocked IMDS returning a sentinel token, then searches trace events, tool results, error strings, and indeterminate records for it.

The sweep has a negative control: a permanent test deliberately injects the sentinel into eight surface types and asserts the sweep catches each one. An absence-assertion that cannot fail proves nothing, so we made ours fail on demand.

---

## 3. One systemic fix worth reporting

This project has repeatedly shipped fields that parsed, validated, and unit-tested cleanly while never being read by production code. `AllowedEnvironments` and `RequiresCapabilities` were both dead when you first raised them.

We have added a reachability gate. A feature must either have a test proving it changes observable CLI behavior — one that fails if the production read is removed — or be explicitly registered as known-dead with a reason. There is no silent third option, and the gate is itself tested to confirm it fires.

`ProfileToolOverride.Endpoint` was registered as known-dead at the start of this phase and has now graduated to reachable under that gate.

---

## 4. Ownership of the contract proof

We are withdrawing the request for access to `gert-sqllivesite` and the artifacts we listed previously.

Gert core should not take a dependency on a consumer repository. This is the boundary established in round one, when `run-gert.ps1` turned out to be your artifact rather than Gert core. Importing your ICM contract into Gert would mean our test suite breaks when your contract changes, our CI depends on your fixtures, and Gert core encodes incident-management semantics it has no business knowing.

Your correction 2 was correct: a test against Gert's sample `icm.tool.yaml` proves nothing about your scenario. But the remedy is a split of ownership, not an import.

| Owner | Proves | Where |
|---|---|---|
| **Gert core** | The mechanism: `--package-map` and `--profile` compose at execution; a native/mock binding and an `mcp-http` binding both resolve for the same logical tool; contract parity holds across them; managed identity attaches to the HTTP binding; the profile never rewrites transport mode. Proven with a synthetic contract-shaped fixture we own — deliberately not named after or copied from your contract. | `gert` |
| **SQL Live-Site** | Your actual contract — `icm.get-incident`, `tsg-recommendation.recommend`, the real `icm-tsg-router.runbook.yaml`, both recommendation outcomes — under both bindings, with Gert as a dependency. | `gert-sqllivesite` |

Neither team guesses a schema, and your real scenario is still proven — by the team that owns its semantics.

This has a consequence we want stated plainly rather than buried: **acceptance criterion 4 is now jointly owned.** Gert proves the mechanism; you prove the exact contract. Neither of us satisfies that criterion alone.

---

## 5. What we need from you

Two confirmations. No files.

**1. The production transport mode for each of `icm.get-incident` and `tsg-recommendation.recommend`** — `mcp-stdio` or `mcp-http`.

This is the one thing we cannot infer. You told us your production ICM definition uses stdio MCP; if that holds for both actions, the HTTP proof requires a separate contract-identical `mcp-http` binding selected through package-map, exactly as you described. We need it per-tool to build the fixture correctly.

**2. Confirmation that you will own the consumer-side contract proof**, and what you need from us to write it — documentation, a worked example, or the contract-parity harness exported as a reusable helper. We are happy to provide any of these.

Item 4 is the only incomplete Phase 1B deliverable. It is otherwise unblocked and depends on nothing else.

---

## 6. Disclosure: repository reproducibility

While verifying this work we found a pre-existing problem we should report rather than let you discover.

The `gert` repository does not currently build from a clean checkout of `main`. Ninety-one files are untracked, including two entire packages (`pkg/pkgcatalog`, `pkg/pkgpath`) that committed code imports. This predates the runtime-portability effort and originates in unrelated work.

Two artifacts we cited to you in the Phase 1A status report are among the untracked files: `internal/tool/auth_gate.go`, which defines the `TokenGate` we described as already shipping, and `cmd/gert/packagemap_integration_test.go`, which we cited as proof that `--package-map` is execution-wired.

To be precise about what this does and does not mean. The code exists, is correct, and is exercised by the passing suite — our technical claims about behavior stand. What does not stand is the implication that those claims were reproducible from the commit history alone. We verified file contents and that cited commits resolved; we did not verify that the commits *contained* the functionality. That was our error, and it is the same failure mode as the dead-field problem in §3: something looked verified without proving the thing it claimed.

Remediation is being scheduled. We will confirm when `main` builds from a clean checkout. We would rather tell you this now than have it surface during your verification.

---

## 7. Not in Phase 1B

Unchanged from the agreed plan: workload identity, reconnect and idempotency hardening, unattended approval evidence, the VS Code host bridge, and enforcement of `RequiresCapabilities` and `AllowedModes` as active controls.

Live ICM validation remains a separate integration gate under your control and does not block code-complete status for the mock proof.
