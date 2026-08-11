# David — Project History

## Learnings

**2026-06-03 — Webhook Delivery Patterns Review**
- Analyzed 4 dimensions of event delivery (webhook models, subscription types, payload designs, security)
- Key insight: retry + exponential backoff is the sweet spot for v1 — simple, survives transient failures, unblocks shipping
- Defensive design: always plan for "customer endpoint is down" before writing code; DLQ is non-negotiable in v1
- Per-execution webhooks scale well initially; tenant-level webhooks reduce blast radius for v1.1; Service Bus moves responsibility to Azure in v2
- HMAC-SHA256 + HTTPS is table-stakes; mTLS and Managed Identity defer to v2
- Summary events (not full stream) in v1 keeps payload small and integrations simple; full audit trace available on-demand via polling endpoint
- Baseline: 5 retries over 24h with exponential backoff; customer admin can replay from DLQ indefinitely

**2026-06-04:** Brainstorm output merged to decisions.md. Webhook & event delivery baseline documented with v1–v2 roadmap. Rollout plan recorded. Security checklist included. Orchestration log created.

**2026-06-04T12:28:39.293-04:00 — Campaign Identity + Completion Integration Design**
- Identity validation at magic-link click must be fail-closed, fast, and tenant-scoped: 8s absolute timeout, per-endpoint circuit breaker, no run start unless the customer endpoint returns a valid approved contract.
- One-time magic links should be opaque tokens with Cosmos DB conditional state transitions (`issued -> redeeming -> redeemed|denied`) so concurrent clicks cannot double-start runs while transient failures can still recover safely.
- Customer notifications should stay summary-first and at-least-once: Service Bus-backed delivery, stable idempotency key per semantic event, scheduled retries, DLQ + replay for prolonged endpoint outages.
- Main integration risks in this scenario are contract drift at the customer identity API, webhook auth/config breakage, and ambiguous retry windows; these surfaced ratification items for timeout, retry schedule, webhook scope, and Cosmos storage shape.

---

## Project Context

GERT is a governed, executable, traceable runbook engine (Go binary, local-first). It has a VSCode extension, TUI runner, and mobile SDKs. The team focus is building a web execution platform on Azure — white-labeled customer portals where end-users run runbooks as part of business processes (applications, authorizations, compliance flows). Azure infra only (no k8s). Key concerns: queue-driven execution, webhook delivery, tenant isolation, audit traceability.

**User:** ormasoftchile
**Session start:** 2026-06-03

---

## Session: 2026-06-04T02:50:37Z — Interactive Waiting Patterns & User Input Gate

**Scribe consolidated 8 inbox items.**

**Key outcomes for David:**
- **New webhook events:** `user_input_requested`, `user_input_received`, `user_input_timed_out`
- **Approval events:** `approval.requested`, `approval.approved`, `approval.rejected`, `approval.timed_out`
- **Event payload:** Summary events only (no full trace changes)
- **Scope:** Same retry/backoff model (5 attempts over 24h)
- **Status:** Event schema updated in decisions.md for integration planning


## Learnings from Declaration & Consent Integration Analysis (2026-06-03)

### Receiver Landscape: Five Industry Archetypes

Declarations are not generic events; they are legal commitments with distinct downstream receivers:

1. **Healthcare:** EHR, surgical scheduling, insurance pre-auth, medical archive. Require signed PDFs, witness verification, 7-year retention, immutable records. Cannot revoke post-procedure start.

2. **Insurance:** Policy admin, underwriting, billing, agent portal. Require evidence of intent, strict ordering (no charge before consent), settlement finality. Billing system is particularly sensitive (recurring charge linkage).

3. **Government:** Registry of acts, archival system, citizen portal. Strictest: registry capacity limits, 30-year archival, citizen notification within 5 min, reproducible audit trails.

4. **Finance/KYC:** AML system, account opening, compliance archive. Compliance-first: silent drops = regulatory fines. Every KYC declaration version traceable. 10-year hold.

5. **Employment:** HRIS, payroll, benefits enrollment. Tight batch windows (payroll deadline is immovable). Tax withholding election cannot be silent-dropped.

**Implication:** Cannot use one-size-fits-all event schema. Must support evidence package depth (trace excerpt, hash chain, audit receipt) that allows receivers to independently verify immutability.

### Evidence Package Shape

Declarations require layered evidence beyond what transient events carry:

- **Signed Document (PDF):** Time-limited SAS URL (7 days default) before seal, permanent after seal. Hash included for immutability proof.
- **Trace Excerpt (NDJSON):** Not full execution trace; only form input + signature steps. Reduces payload; receiver can fetch full trace via declaration_id if audited.
- **Hash Chain (Merkle Proof):** Daily batch Merkle tree proving declaration is in immutable archive (no post-hoc edits by system operator).
- **Audit Receipt (PDF + QR):** Human-readable receipt + permanent QR link for declarant to print/file as proof.

**Implication:** Evidence package is not inline in webhook. Webhook carries URLs; receiver pulls evidence on demand. Scales to large audiences without massive payloads.

### Delivery Guarantee: At-Least-Once + Immutability

V1.0 retry model is necessary but not sufficient. Declarations demand idempotency at the receiver level:

- Receiver deduplicates on `idempotency_key` (not just event_id). Safe to retry declaration.completed for same declaration_id multiple times.
- Replay endpoint (`GET /api/v1/declarations/{declaration_id}/replay?since=T`) for reconstruction if receiver crashes mid-processing.
- Sealed archive notification (`declaration.sealed` event) signals: "Declaration is now in permanent immutable archive; all SAS URLs permanent; hash chain finalized."
- Long-term retrieval SLA: Healthcare 7yr, Finance 10yr, Government 10–30yr. No expiration after seal.

**Implication:** Webhook infrastructure must support replay + permanent SAS (or fallback retrieval mechanism) for receivers to comply with legal holds.

### Edge Cases: Nine Declaration Event Types

Declarations are not binary (completed or failed). Each path requires an event:

1. **submitted:** Form filled, signature pending. Receiver audits; optional queue.
2. **verified:** KYC/ID verification passed. Continue; optional status update.
3. **signed:** Declarant signature captured (witness may still be pending).
4. **completed:** FINAL; all signatures, all verification done. Trigger downstream: schedule surgery, bind policy, create compliance record.
5. **declined:** Declarant actively rejected (clicked "Decline"). Cancel downstream; notify participant.
6. **abandoned:** User didn't finish (inactivity, closed tab). Audit-log; may retry.
7. **expired_unwitnessed:** Witness window passed. Retry or escalate.
8. **sealed:** N days post-completion; immutable archive. Archive-ready; receiver downloads evidence package to long-term storage.
9. **revoked:** Declarant/authorized party revoked consent post-completion. **CRITICAL:** Cancel downstream, reverse charges, halt procedure.

**Implication:** V1.0 event family (execution lifecycle + user input) is too coarse. Declaration workflow needs a second event family with finer granularity.

### Revocation Semantics: The Hardest Edge Case

Post-hoc revocation is non-trivial. Declarant signed surgical consent, was placed on pre-op protocol, then calls to withdraw consent hours before surgery.

- Receiver gets `declaration.revoked` event.
- Receiver must be idempotent: revoking twice = safe no-op.
- Receiver must cascade: cancel surgery, notify anesthesia team, halt pre-op protocol.
- Audit trail critical: compliance must know who revoked, when, what cascade impact.

**Implication:** Revocation is not handled at webhook registration time; it's a post-completion async action requiring playbook execution at receiver.

### Retention SLA Complexity

Industries have different retention requirements:
- Healthcare: 7 years minimum (varies by jurisdiction).
- Insurance: 5–10 years (product-dependent).
- Government: 10–30 years (registry-dependent).
- Finance/KYC: 10 years (regulatory).
- Employment: 7 years (tax, labor law).

**Implication:** Cannot have one DLQ retention policy. Declarations must be tagged with industry + retention tier. Compliance dashboard should show: "Healthcare declarations from 2020 still held; can purge finance declarations from 2019."

### Open Questions for Next Sprint

1. Idempotency key scope: `declaration_id` only, or `declaration_id + version`? (If revocation bumps version, idempotency_key changes?)
2. Cascade notifications: When revocation happens, does GERT notify declarant/witness/surgeon directly, or only via webhook?
3. Audit receipt: PDF with QR only, or support text QR body in email?
4. SAS URL expiration before seal: 7 days? Configurable?
5. Merkle tree scope: Daily batch per-tenant, or global daily batch?
6. Revocation window: Can revoke indefinitely, or only within N days?

### Next Session: Implementation Roadmap

- **Phase 1 (MVP):** 4 core events (submitted, signed, completed, declined); evidence package URLs; idempotency implementation.
- **Phase 2 (Robustness):** 5 edge case events; DLQ compliance-critical alerting; replay endpoint.
---

## Team Integration — White-Label Campaign Platform (2026-06-04)

**Cross-agent coordination:**
- **Barbara:** Integration contracts ratify webhook delivery model (signed webhook + pull reconciliation) and identity validation requirements (server-to-server only); campaign layer defines notification subscription entities
- **John:** Aspire local dev supports testing integration contracts in hybrid dev loop
- **Leslie:** Portal UX determines when webhooks are triggered (completion, abandonment, identity denial); affects event schema design
- **Scribe:** 5 integration contract ratification items merged to decisions.md for team review

**Key decisions:** Identity endpoint timeout 8s absolute; circuit breaker opens after 5 failures (5 min); campaign links stored as separate Cosmos docs; webhook registration at tenant/campaign scope; 5 retries over 24h; default events (campaign.closed, user.started, user.completed, user.failed).

---

## Expression Language Alignment — GXL/GIS/GCP Approved (2026-06-05)

**Action for David (Integration/Webhooks):** Webhook payload shapes must support new portable interpolation syntax for variable substitution. No more Go template evaluation in webhook metadata.

**What changes:**
- Webhook metadata fields (event.id, custom headers): Migrate from Go `text/template` to GIS interpolation (`${...}` syntax)
- Webhook payload variable substitution: Use GIS dotted paths (`${capture.customer_id}`) instead of template ranges
- Assertion operands in webhook verification logic: Use GIS for dynamic values instead of template syntax
- `capture.default:` provides fallback for optional capture paths (replaces text/template silent-empty behavior)

**Integration implications:**
- Webhook payloads remain JSON; payload *structure* doesn't change, only how variables are interpolated
- Integration tests should verify new GIS syntax interpolation in webhook metadata
- Backward compatibility: Any Go-template syntax in current fixtures will be rejected at parse time; migration is enforced before webhook dispatch
- Multi-receiver scenarios (declaration events, at-least-once webhooks): Variables are interpolated consistently across all runtimes; no drift risk

**References:** `.squad/decisions/decisions.md` (GXL/GIS/GCP open questions resolved + architecture proposal)

---

## Team Directive — 2026-06-04T17:15:45-07:00

**Leslie uses he/him pronouns.** All team members and the coordinator must refer to Leslie with he/him going forward.

---

## Learnings — 2026-06-07T19-07-00-07-00 — gert-tui NTFS Colon Recovery

**What was bad:** 19 historical Scribe log files in `.squad/log/` and `.squad/orchestration-log/` had raw ISO-8601 colons in their filenames (e.g., `2026-04-29T01:00:25Z-scaffold-session.md`). Windows NTFS forbids `:` in path components, causing `git checkout` to fail silently mid-clone and leave the entire working tree empty (index also wiped to match — all 205 tracked files appeared as staged deletions in `git status`).

**Diagnostic fingerprint:**
- `git status --short` shows 205 `D ` (staged deletions) entries
- `git ls-files` returns nothing (index is empty)
- `git ls-tree -r HEAD --name-only | Select-String ":"` reveals the offending paths
- `git checkout -- .` fails with `error: pathspec '.' did not match any file(s) known to git`
- `core.protectNTFS = true` is the guard that prevents writing the bad paths

**Rename pattern applied:** `T(\d{2}):(\d{2}):(\d{2})Z` → `T$1-$2-$3Z`

**Fix method (bypasses NTFS entirely):**
1. Fetch full recursive tree via `gh api repos/{owner}/{repo}/git/trees/HEAD?recursive=1`
2. Build new blob-only tree array with colon paths renamed (skip `type=tree` entries)
3. `POST /git/trees` → new tree SHA
4. `POST /git/commits` with new tree + HEAD as parent
5. `PATCH /git/refs/heads/main` to advance the ref
6. `git fetch origin && git reset --hard origin/main` locally — clean restore in one shot

**Key lesson:** Never attempt `git mv` or `git update-index` for colon-path recovery on Windows — the index itself rejects those entries. The GitHub API is the only reliable path on a Windows-only machine.

**New git config that helped:** `core.protectNTFS = true` was already set (default on Windows git). `core.longpaths` was not set (blank = disabled) — not relevant here but worth enabling for repos with deep paths.

---

## Scribe Session — 2026-06-07T19:07:00-07:00

**Summary:** Scribe consolidated David's recovery work into decisions.md and created orchestration logs. David's recovery memo (`david-ntfs-safe-filenames-policy.md`) is now part of the ratified NTFS-safe filenames policy for all GERT-family repos.

**Key update:** Pre-commit hook rollout needed across gert-tui, gert-vscode, gert (runtime), and all future GERT-family repos. Owner: Coordinator.

**Artifacts created:**
- `.squad/orchestration-log/2026-06-07T19-07-00Z-david.md` (recovery summary)
- Updated `.squad/decisions/decisions.md` (policy now ratified)
- `.squad/log/2026-06-07T19-07-00Z-gert-tui-checkout-and-vscode-audit.md` (session log)



---

## Revision — 2026-08-09T15:59:36-07:00 — Tool Packages MVP gate-review resolution (independent revision owner)

**Context:** Barbara rejected the Tool Packages MVP runtime implementation (Don/Ken authored, now locked out). Cristián assigned me as sole independent revision owner to resolve every blocking item and required lower-severity fix without consulting either locked-out author.

**What I did:** Resolved all five blockers (B1 plan-time substitution validation, B2 digest closure over substitute runbooks/package-internal includes, B3 catalog digest from package digest not file digest, B4 resume drift PKG-001/PKG-009 for removed/added packages, B5 `outputs.<name>` GCP capture root) plus the sixth (§5 include closure) via Barbara's own explicitly sanctioned fail-closed fallback (chose (b) over full lexical `toolRefs` binding (a), given revision size). Also completed all nine §3 required lower-severity fixes (`ConstraintSources` provenance, PKG-002 messages, PKG-003 build-metadata rejection, deterministic PKG-006 ordering, typed PLAN-010, PKG-018 NFC/case wiring, explicit `maxLinkHops` enforcement, output-coercion fail-fast, `--package-map` origin in trace).

**Bug found and fixed along the way (not part of the original review's own list, but blocking correct B1/B5 behavior):** `cmd/gert/packagemap.go`'s `schemaToolDefFromRuntime` was silently dropping `Execute`/`Outputs` when converting a resolved package/toolRefs-bound tool into the planner's schema-typed registry — this made every catalog-bound substitution action invisible to both the new plan-time validator and the new `outputs.<name>` step-context check. Fixed and covered by regression tests.

**Also found and fixed:** a `yaml.v3` panic (duplicated-key) in a naive typed-unmarshal approach to B2's digest closure, caused by `schema.Step`'s many colliding inline-tagged fields — replaced with manual `*yaml.Node` walking (same technique the production parser already uses for this reason).

**Result:** `go build ./...` clean; `go test ./...` all green (every package, including Tess's `internal/conformance` vectors); all four protected files verified byte-for-byte untouched; nothing staged/committed/branch-switched. Full blocker-resolution matrix and revision record filed at `.squad/decisions/inbox/david-tool-packages-revision.md`. One explicit, sanctioned gap remains: §5's full per-file lexical `toolRefs` binding (option (a)) is not implemented, only the fail-closed fallback (b) — flagged for Barbara/Cristián, not silently dropped.

---

## Revision — 2026-08-11T09:45:19-07:00 — Client Enum Parity gate-review resolution (independent revision owner)

**Context:** Barbara rejected the client enum-parity implementation (Don's runtime DTO work, Ken's TUI helper, Leslie's VS Code prep — all now locked out). Cristián assigned me sole independent revision owner for B-1..B-6/F-1..F-11 across `gert`, `gert-tui`, `gert-vscode`, without consulting any of the three original authors.

**What I did, by finding:**
- **B-1/F-1/F-2 (`gert`):** Found the DTO (`graphdoc.InputDecl`, Don's work) was already correct; the bug was purely that `pkg/preview/render/graphjson.Document` never carried `doc.Inputs` through. Added the field + copy, then proved it with a real-binary CLI test (`cmd/gert/preview_graphjson_inputs_test.go`) rather than trusting the Go-type-level fix alone.
- **B-2/F-3 (`gert`):** `InteractionField.Enum`/`FormField.Enum*` had no legitimate live producer without either a new schema binding (forbidden) or wiring `Input.From: prompt` (out of scope). Took Barbara's own sanctioned alternative and deleted the dead fields/constructor instead of inventing scope creep.
- **B-3/F-4 (`gert`):** Replaced the web GUI's raw-JSON-only input box with a real `InputsForm` (closed selector in declared order, no auto-select, redacted free text, unset-vs-empty-string distinction), keeping the raw-JSON box as an outright-wins fallback.
- **B-5/F-5 (`gert-tui`):** The TUI called `run.Start` (discards warnings) with a permanently-false no-op `engineWarning` shim. Switched to `run.StartWithWarnings`, added a real `SetParseWarnings` seam, and rendered warnings nonfatally in the status bar.
- **B-4/F-6 (`gert-tui`, the largest piece):** `enumui.BuildEnumField` was a disconnected helper with only unit tests. Wired a real pre-run-start prompting path: parse the runbook via `pkg/preview.BuildDocument` before `run.StartWithWarnings` locks in `RunOptions.Vars`, derive missing enum-constrained fields, and host them in a standalone `tea.Program` around the *existing* `interaction.MultiFieldFormModel` widget (the same one collector-step forms already use) — never a new UI. Proved this at the app level, not the helper level: a test drives the real widget's `View()`/`Update()` and asserts declared-order rendering, no preselection, and byte-verbatim submission.
- **F-7/F-8 (`gert-vscode`/`gert-tui`):** Updated stale "DTO does not exist yet" comments now that F-1/F-2 shipped it for real, and added a VS Code test that runs the actual `gert preview --format graphjson` binary and asserts `extractInputDecls` recovers the declared enum in order.
- **F-9 (`gert-private`):** Built `design/gert/conformance/client-parity-matrix.md`, one row per CE-* ID citing its owning test, without touching the frozen `tv-enum.yaml`. Documented three honest gaps (a real-CLI VS Code test regression owned by another locked-out author's in-flight `pkg/schema` work, two GUI rows with no headless-browser harness, and an unautomated corpus-SHA check) rather than papering over them.
- **F-11:** Reproduced and confirmed both flagged `gert-tui` failure clusters (`internal/tui`'s hardcoded macOS `/Volumes/Projects/...` paths; `internal/e2e`'s real DNS/ping dependency on `contoso.com` placeholder hosts) are genuinely pre-existing and unrelated — verified via `git status`/`git diff` showing the failing test files uninvolved in any dirty work, rather than assuming Barbara's own suspicion was correct.

**Result:** `gert` full `go test ./...` green. `gert-tui` green except the two confirmed-environmental F-11 clusters. `gert-vscode` 26/27 (one pre-existing, out-of-scope failure caused by a live conflict in another author's protected concurrent `pkg/schema` work — reported explicitly per this task's conflict-reporting instruction, not silently patched). Nothing reverted/staged/committed/branch-switched in any of the four repos; every pre-existing dirty file preserved. Full B/F resolution matrix, per-repo file/test list, protected-file verification, and the three accepted limitations filed at `.squad/decisions/inbox/david-client-enum-parity-revision.md`.

📌 Enum Client Parity Independent Revision (2026-08-11T09:45:19-07:00):
- Assigned for B-1..B-6 blocker resolution (independent, locked-out authors untouched)
- F-1..F-11: DTO delivery to graphjson, GUI form (declaration-driven selects), TUI wiring (pre-run-start form + warning display), error code paths, VS Code test, parity matrix
- Protected concurrent work preserved byte-identical; real-binary execution verified all fixes
- Three accepted non-blocking gaps documented (CE-V-04 conflict, GUI no headless test, corpus SHA drift)
- Approved at final gate (Barbara verified)

