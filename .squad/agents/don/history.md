# Don — Project History

## Learnings

## Learnings

### 2026-06-06 — Stream E Day 1 Complete: `gert migrate-expr` Tool

**Deliverables:** Subcommand scaffolded, 11 translation rules implemented, 31 tests passing.

**Architecture:**
- CLI via `github.com/spf13/cobra` v1.8.1
- Position-aware YAML traversal with `gopkg.in/yaml.v3` v3.0.1 for comment/style preservation
- Expression-position detection by YAML key name (`when`, `condition`, `until`, `iterate.over`)
- Non-string tag guard (skip `!!int`, `!!bool`, etc.)

**Translation Rules (11):**
- **E-001/002/003:** GIS interpolation templates (`{{ .X }}` → `${X}`, etc.)
- **E-004/005/006:** GXL binary operators (`&&`/`||`/`!` → `and`/`or`/`not`)
- **E-007:** Infix `contains` → `str.contains()` stdlib call
- **E-008:** jq-style root path `$.X` → bare GDP identifier
- **E-009:** Old escape style `$${...}` → OI-GIS-01 canonical `\${...}`
- **E-010:** Template pipes — emit WARN, no auto-translate (e.g., `{{ .env | default "dev" }}`)
- **E-011:** `{{ now }}` → `${now()}` (Barbara's GXL stdlib decision)

**Test Coverage:**
- 26 unit tests (rules, heuristic guards, edge cases)
- 5 integration tests (full-file YAML transformations)
- All passing; `go test ./...` clean

**Flags:**
- `--dry-run`: compute without writing
- `--diff`: show unified diffs
- `--report <file>`: JSON per-file counts + warnings
- `--strict`: exit non-zero if any pattern cannot be auto-translated

**Known Gaps (Day 2 actions):**
- E-007 `contains` heuristic: assumes string; lists need `list.contains()`. Day 2: emit per-occurrence warning in `--report`.
- E-006 parenthesized negation `!(EXPR)`: regex does not match. Day 2: extend to detect and warn.
- YAML style normalization: `gopkg.in/yaml.v3` may normalize indentation. Day 2: document in migration guide.

**Day 2 Plan:** Dogfood on already-migrated fixtures (expect zero diff); enhance E-007 warnings; extend E-006 regex; add `--report` summary + deferred patterns list; validate r04-soc2-evidence with Barbara's `now()` patch via `gert validate`.

**Ratified decisions baked in:**
- OI-GIS-01 (canonical escape `\${`)
- OI-GCP-02 (bare-root GDP captures allowed)
- Barbara's now() spec (E-011 rule)

### 2026-06-04 — Go-Style Expression Evaluation Audit

**Deliverables produced:** `design/gert/expression-evaluation-audit.md`, `.squad/decisions/inbox/don-go-expression-audit.md`, `.squad/skills/expression-evaluation-audit/SKILL.md`

**Key findings:**
- The repository snapshot did not include Go runtime source files or `go.mod`, so implementation-level evaluator calls could not be inspected.
- Spec/docs expose two author-facing evaluator families: `expr-lang/expr` for `when`/`condition`/`until`, and Go `text/template` for interpolation in titles, args, tool argv, display content, and assertion operands.
- Conditional syntax currently exposes Go/C-style operators (`&&`, `||`, `!`, `==`, comparisons). The spec calls the intended vocabulary portable, but fixtures still use non-normative `expr` infix `contains`.
- There are contract conflicts: global expression docs say conditional fields use `expr`, while branch and collector tables still describe some guard fields as Go templates.
- Capture paths are not Go-style but are underspecified (`json.items[0].metadata.name`, `json.items | length`, `stdout.incident.id`), creating cross-runtime parity risk.

**Portability red line:** authored runbook semantics must be rejected or normalized at parse/plan time before `RunHandle.Next`; runtime adapters must not paper over evaluator drift.

---

### 2026-06-03 — Declaration/Consent Scenario Gap Analysis

**Deliverable produced:** `.squad/decisions/inbox/don-declaration-runtime-gaps.md`

**Gap inventory (10 gaps identified):**

| # | Gap | Priority | New Primitive |
|---|---|---|---|
| G-1 | Signature capture | P0 | `UserInputKind.Signature`, `DeclarationCollectedEvent` |
| G-2 | Identity proofing at declaration moment | P0 | `IdentityProofingLevel` on `UserInputRequest` |
| G-3 | Witness / co-signer flow | P0 | `IWitnessGate`, `WitnessAttestedEvent` |
| G-4 | Presented-document version hash | P0 | `document_hash` + `document_version` on trace event |
| G-5 | Multi-language / locale provenance | P1 | `Locale` (BCP-47) on `DeclarationCollectedEvent` |
| G-6 | On-behalf-of / capacity (subject ≠ declarant) | P1 | `DeclarationPrincipal` record |
| G-7 | Time-bounded validity (expiry) | P1 | `valid_until` field + `IDeclarationRegistry.GetEffectiveStateAsync` |
| G-8 | Revocation (inverse event, no rewrite) | P1 | `DeclarationRevokedEvent` + `IDeclarationRegistry` |
| G-9 | Contextual PII in free-text fields | P2 | `PiiZone` policy flag (no new engine) |
| G-10 | QTSP integration (eIDAS/ESIGN/Ley 19.799) | P2 | `UserInputKind.QualifiedSignature`, `QualifiedSignatureReceivedEvent` |

**What already fits well:**
- JSONL append-only trace is the right substrate for declaration registration — synchronous-before-dispatch invariant directly satisfies "must be registered before process proceeds."
- Service Bus at-most-once prevents duplicate declaration collection on worker restart.
- RE2 redaction handles structured PII (RUT, SSN, IBAN) well — gaps only in free-text narrative PII.
- `IApprovalGate` covers genuine downstream authorizers (underwriters, administrators) but NOT witnesses or co-signers.
- `Confirmation` kind is the closest current primitive to a declaration acknowledgment — insufficient for legal weight without G-1 and G-2.
- `client="web"` RunOptions tagging already fits per-channel audit needs.

**Proposed new trace event types:** `DeclarationCollectedEvent`, `WitnessAttestedEvent`, `DeclarationRevokedEvent`, `DeclarationValidityCheckedEvent`, `QualifiedSignatureReceivedEvent`.

**Proposed new interface:** `IWitnessGate` (parallel to `IApprovalGate`; fires inside `IRunHandle.NextAsync()`).

**Proposed extensions:**
- `UserInputKind.Signature` and `UserInputKind.QualifiedSignature` added to enum.
- `DeclarationContext` added to `UserInputRequest` for declaration-type steps.
- `DeclarationPrincipal` record for on-behalf-of flows.
- `IDeclarationRegistry` for effective-state queries (Active | Expired | Revoked).

**Non-goals confirmed:** GERT is NOT a TSP, does NOT store biometrics, does NOT render legal text, does NOT adjudicate legal validity, does NOT enforce per-jurisdiction compliance rules.

**Chilean legal context:** RE2 patterns for RUT format already expressible. Ley 19.799 (Firma Electrónica) compliance requires QTSP integration (G-10) for advanced e-signature tier — GERT records TSP tokens, does not issue signatures.

---

### 2026-06-03 — C# Governance Parity Design

**Deliverable produced:** `design/web-platform/c-sharp-governance-parity.md`

**Key design decisions made:**

- **Pipeline mapping:** Full Go-to-C# interface mapping documented. `IRunHandle.NextAsync()` is the governance boundary — identical to Go's `RunHandle.Next()`. Every stage (Parser → Planner → Runtime → RunHandle) has a named C# equivalent with method signatures.
- **IApprovalGate stubbed for A6/A8 swap:** Designed so `ServiceBusApprovalGate` (A6: held thread, Service Bus reply queue) can be swapped for `DurableApprovalGate` (A8: `WaitForExternalEvent`) without changing `IRunHandle` or the governance layer. The runbook contract sees only `IApprovalGate`.
- **IStepRunner enforcement via `internal sealed`:** `IStepRunner` and all concrete runners (`CliStepRunner`, `ToolStepRunner`, etc.) are `internal` to `Gert.Runtime.Core`. No `InternalsVisibleTo` grants to adapter assemblies. Bypass is a compile error, not a code review finding. Two Roslyn analyzers (`GERT0001`, `GERT0002`) enforce this and the RE2 namespace restriction at build time.
- **RE2 via `Google.Re2` NuGet:** All redaction evaluation uses `Google.Re2.Regex`, never `System.Text.RegularExpressions`. Patterns containing lookahead/lookbehind/possessive quantifiers will throw at startup (fail-fast). Named group syntax difference (`(?P<name>...)`) flagged as open question OQ-5.
- **Template engine gap:** Go `text/template` has no C# equivalent. Option A (minimal port) is recommended for MVP. Option B (WASM compiled Go evaluator) is the fallback. 10 canonical test vectors defined (TV-TMPL-001..010).
- **Trace synchronous write:** `IRunStore.WriteTraceAsync()` must complete before step dispatch. Call order in `RunHandleImpl.NextAsync()` is: governance check → emit event → **synchronous trace write** → step dispatch. Fire-and-forget is explicitly forbidden.
- **Validation gate:** 10 gates (G-01..G-10) defined. Go `gert replay` on C#-generated traces (G-06) is the strongest parity check. No production promotion with any gate failing. Roslyn analyzers enforce G-08 and G-09 at build time.
- **60 minimum test vectors** across 12 categories (allowlist, denylist, env blocking, redaction, approval gates, template evaluation).

**Open questions filed:** OQ-1 (template strategy), OQ-2 (approval reply queue topology), OQ-3 (canonical Go test harness), OQ-4 (sequence counter strategy), OQ-5 (RE2 named group syntax).

---

### 2026-06-03 — Execution Adapter Pattern Review

**Patterns reviewed:** Queue-triggered worker, HTTP-triggered Container App, Durable execution (Azure Durable Functions), Sidecar/agent model.

**Key governance concerns identified:**

- GERT's governance layer (allowlist/denylist, env var filtering, contract risk assessment, approval gates) fires entirely inside the **Runtime Core** during the `RunHandle.Next()` loop. It cannot be preserved by any pattern that bypasses `Runtime.Start → RunHandle.Next`.
- The JSONL trace file is written **synchronously inside the Runtime Core before each action proceeds**. Any pattern that writes trace events from outside the Go binary (e.g., a Function orchestrator writing its own log) would produce a trace that is NOT the authoritative GERT trace and would fail replay determinism.
- The `RunHandle` interface is **stateful and sequential** (`Next()` must not be called concurrently, re-entrant only per run). It cannot be split across multiple processes or Function invocations.
- Splitting execution into per-step Azure Durable Function activities is a **governance red line**: each activity invocation re-enters the binary cold, losing the in-process governance engine state and potentially allowing steps to execute without the full pre-flight chain.
- Approval gates (`RunHandle.Approve()`) and evidence submission (`RunHandle.SubmitEvidence()`) require a **live, running Go process** that holds the `RunHandle`. Any pattern that doesn't keep the worker alive through the pause window cannot support interactive runbooks without an explicit resume mechanism (checkpoint → reload from RunStore).
- The `client: "web"` field in `run/started` must be set correctly. Forgetting to identify the client means audit trail queries filtering by client will miss web-triggered runs.
- Queue-triggered worker is the strongest match: decoupled submission, worker owns the full Runtime Core, governance fires completely, trace goes to Blob Storage, approval gates handled via reply messages or a dedicated interaction queue.

**2026-06-04:** Brainstorm output merged to decisions.md. Governance red lines formalized and documented. Pattern evaluation matrix recorded. Requires team ratification before implementation. Orchestration log created.

---

### 2026-06-03 — C# Runtime Option Analysis

**Trigger:** User confirmed they are NOT attached to the Go binary for web execution. Open to a C# reimplementation of GERT governance for the Azure backend.

**What this unlocks:**
- Durable Functions Orchestrators can now legitimately implement `RunHandle.Next()` semantics — the orchestrator IS the governance engine, not a bypass of it. This was a red line when Go binary was fixed because each activity invocation would cold-start the binary, losing governance state. In C#, governance fires inside the orchestrator before each activity, which is correct.

---

## [Archived Sessions — Pre-2026-06-04]

**Historical summary:** Don completed 7 major analysis cycles (2026-06-03 through 2026-06-04T02:50:37Z):
1. Declaration/consent gap analysis (10 gaps, 5 new primitives proposed)
2. C# governance parity design (10 validation gates, Roslyn analyzer strategy)
3. Execution adapter pattern review (Red Line architecture decisions)
4. C# runtime option analysis (BackgroundService + Durable orchestration design)
5. Choice gate runtime contract (Interactive waiting patterns)
6. User input gate correction (Generalized IUserInputGate)
7. Interactive waiting patterns cross-review

**Key decisions adopted:** IRunHandle.NextAsync as governance boundary, Google.Re2 for redaction, IUserInputGate interface (not IChoiceGate), Roslyn analyzers GERT0001/GERT0002, 10 validation gates before C# production.

**Active open questions:** OQ-1 through OQ-5 (template strategy, approval queue topology, sequence counter, RE2 named group syntax).

---

## 2026-06-05T18:30:00-04:00 — Stream D Complete: Fixture Migration (Day 1 + Day 2)

**Day 1 — Audit & Planning:** Conducted comprehensive fixture migration audit across 22 runbooks, 12 tool files, 1 extension file. Identified **388 total violations** requiring migration:
- 368× `{{ .var }}` template syntax violations
- 6× `&&`/`||` boolean operators in expression-position fields
- 9× `!` prefix negations in `when:` fields
- 3× infix `contains` in expression-position (non-method calls)
- 1× `$.` jq-style root path reference
- 1× template pipe `| func` pattern

**Escalation items:** 5 RISK-* items flagged for team decision; 1 blocking (r20 `env` semantics — resolved by Germán: `env` is runbook input with default "dev").

**TESS-AMBIG survey:** Confirmed no fixture exposure to Tess's 4 unresolved ambiguities (boolean ordering, array equality, scalar field access, null field access).

**Day 2 — Full Migration Execution:** Completed all 21 runbooks (r01–r11, r13–r22; r12 already clean) in 4 batches:
- **Batch 1 (6 files):** r16, r18, r13, r21, r14, r11 — pure template substitution + GDP normalization
- **Batch 2 (5 files):** r17, r15, r19, r06, r09 — medium volume, expression operator normalization
- **Batch 3 (4 files):** r22, r08, r05, r04 — high volume, pure GIS migration
- **Batch 4 (5 files):** r02, r07, r10, r01, r03 — expression violations resolved

**Template Substitutions:** Applied ~368 replacements across all files. All `{{ .var }}` → `${var}` (GIS), `{{ .X.Y }}` → `${X.Y}` (GDP), `{{ .X[N] }}` → `${X[N]}` (array indexing).

**Expression Normalizations:**
- All `&&`/`||` → `and`/`or` (GXL binary operators)
- All `!var` → `not var` (GXL unary operator)
- All infix `contains` → `str.contains(var, "str")` (stdlib method call)

**Structural Changes:**
- r11: `over: "$.services"` → `over: services` (bare GDP identifier per GCP spec §3.5)
- r20: Added `inputs.env: {type: string, default: "dev"}` + replaced `{{ .env | default "dev" }}` with `${env}`
- r07: Verified `${{ .amount }}` → `$${amount}` (literal `$` + interpolation block) is correct

**Exit Criteria Results (P1–P5):**
- P1 `{{ }}` non-comment: 1 remaining (DEFERRED-001: `{{ now }}` in r04:285)
- P2 `&&`/`||` expression-position: ✅ 0
- P3 `!` in `when:`: ✅ 0
- P4 infix `contains`: ✅ 0
- P5 `$.` jq-style: ✅ 0

**Deferred Decision — DEFERRED-001 ({{ now }} in r04):**
- `{{ now }}` is Go Sprig template function (not variable access)
- No `now()` in GXL stdlib v1.0.0-draft
- **Decision ratified:** Add `now()` to GXL stdlib (per Germán call); returns UTC ISO-8601 string
- **Implementation:** Barbara to handle grammar/spec/fixture patch in parallel
- **Impact:** Stream D complete pending now() stdlib availability

**Stream D Final Status:** ✅ **COMPLETE** — 21 runbooks migrated, 20 fully delivered, 1 (r04) deferred on now() stdlib. All P2–P5 exit criteria passed at zero. Tools and extensions untouched.

---

## Learnings from Stream D Migration

## 2026-06-04T23:56:26-04:00 — Stream A (Grammar Files) Complete — Ready for Stream C/D

**Update:** Barbara's Stream A (reference EBNF grammar files) is complete. Delivered:
- `design/gert/grammar/gxl.ebnf` (~25 KB)
- `design/gert/grammar/gis.ebnf` (~16 KB)
- `design/gert/grammar/gcp.ebnf` (~26 KB)

**Critical decisions now binding:**
- OQ1 (short-circuit `and`/`or`) — ratified by ormasoftchile
- OQ2 (`capture.default:` scalars only) — ratified by ormasoftchile
- OQ3 (Portable JSON Value Model) — specified by Barbara
- OQ4 (`list.indexOf` stdlib) — added to grammar

**You can now start:**
1. **Stream C (Conformance Corpus)** — error codes and semantics locked. Begin writing ≥200 test vectors.
2. **Stream D (Fixture Migration)** — grammar constraints are clear. Can begin migrating 22 runbook + 12 tool fixtures.

**Outstanding team ratifications needed:**
- OI-GIS-01: `\${` escape convention (deviation from original `$${` proposal)
- OI-GCP-02: Root JSON capture without path (`json` keyword currently parse error)

Both are documented in grammar files. Consider adding to today's decision record.

**Orchestration log:** `.squad/orchestration-log/2026-06-04T23-56-26-barbara.md`

---

## 2026-06-05T00:14:04-04:00 — Phase 1 Day 1 Streams B, C, F Complete — Stream D Ready for Design

**Update:** Streams B, C, F completed successfully. For Stream D (Fixture Migration), the following are now available:
- **Spec Foundation:** `design/gert/sections/03a-expression-language.tex` (GXL spec) + `design/gert/sections/03d-parse-time-enforcement.tex` (parse gate spec) — Edith & Barbara
- **Conformance Corpus:** `design/gert/conformance/schema.json` + `design/gert/conformance/tv-gxl-parse.yaml` (83 parse vectors) — Tess
- **Grammar Patches:** `design/gert/grammar/gis.ebnf` + `design/gert/grammar/gcp.ebnf` updated per OI-GIS-01, OI-GCP-02 ratifications

**You can now start:** Stream D (Fixture Migration) design phase. Grammar constraints, spec definitions, and conformance corpus all locked. OI-GIS-01 (`\${` canonical) and OI-GCP-02 (bare-root captures allowed) ratified and encoded.

**Outstanding blockers for Phase 2:** OPQ-GATE-01 (Germán decision needed on in-flight grammar version upgrade strategy — blocks OPQ-GATE-05 plan storage). No blockers for Stream D kickoff.

**Orchestration logs:** `.squad/orchestration-log/20260605-042704-{edith,tess,barbara}.md`

