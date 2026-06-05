# Don — Project History (Summarized)

## Overview
Don is the runtime/tooling engineer. Work spans fixture normalization (Stream D), the removed Stream E migrator, declaration runtime gaps, C# governance parity, and execution adapter validation.

## Key Milestones (Phase 1)

### Stream E — Migration Tool & Idempotency (Days 1-2 — REMOVED BY USER DIRECTIVE)
- **Day 1:** gert migrate-expr scaffolded with 11 translation rules (E-001..011), 31 tests passing
  - CLI via cobra, position-aware YAML traversal (gopkg.in/yaml.v3), expression-position detection (when/condition/until/iterate.over)
  - Rules: GIS templates (E-001..003), GXL operators (E-004..006), contains → stdlib (E-007), jq-style paths (E-008), legacy escapes (E-009), template pipes warn (E-010), now() fix (E-011)
  - Flags: --dry-run, --diff, --report JSON, --strict
- **Day 2 (Dogfood):** 22 runbooks verified with 0 translations post-fix
  - **Real idempotency bug fixed:** migrated $${amount} was being reclassified as E-009 legacy on second pass. Now preserves $${symbol} when symbol known from context (inputs/captures/collector fields)
  - **E-007 contains ambiguity:** String-like → str.contains(), list-like → list.contains(), ambiguous → warning (heuristic only, no auto-rewrite)
  - **E-006 extended:** Quote-aware negation scanner covers !X, !(X), !(X && Y), !!(X || (Y && Z)), !!X
  - **Validation:** gert migrate-expr --dry-run on testdata: 0 translations, go test ./... passing
- **Stream E reversal (2026-06-04T20:14:36.949-07:00):** Removed by ormasoftchile directive after Day 2 shipped because GERT has no production runbooks in the wild, so there is no legacy syntax target to migrate from. Keeping the tool would imply a legacy mode and create a contract surface future runtimes would have to reason about.
- **Phase 1 status:** Stream E is removed from the plan. Phase 1 closes around the remaining A/B/C/D/F work; grammar, conformance, Stream D fixtures, and the independent `now()` stdlib addition remain intact.

### Stream D — Fixture Migration (COMPLETE)
- **Audit:** 22 runbooks, 12 tool files, 1 extension. 388 violations identified (368 templates, 6 and/or, 9 !, 3 contains, 1 jq, 1 pipe)
- **Migration:** All 21 runbooks (r01–r11, r13–r22) migrated in 4 batches
  - ~368 {{ .var }} → ${var} substitutions
  - All and/or → and/or keywords, all ! → not, all infix contains → str.contains(...)
  - r11: bare GDP identifier (no jq-style $.)
  - r20: added inputs.env default, replaced {{ .env | default "dev" }} with ${env}
  - r07: preserved $${amount} as literal $ + ${amount} interpolation
- **Exit Criteria (P1–P5):** P1 ({{ now }}) deferred to Barbara's stdlib decision ✓, P2–P5 all zero ✓
- **Deferred-001:** {{ now }} in r04:285 — resolved by now() stdlib addition. Stream D final: ✅ COMPLETE

### Stream A — EBNF Grammars (COMPLETE)
- Delivered: gxl.ebnf, gis.ebnf, gcp.ebnf (75 KB total)
- Critical decisions locked: short-circuit and/or (OQ1), capture.default: scalars only (OQ2), Portable JSON Value Model (OQ3), list.indexOf stdlib (OQ4)
- Ready for Streams C (corpus) and D (migration design)

### C# Governance Parity & Runtime Architecture (COMPLETE)
- **Governance boundary:** IRunHandle.NextAsync() is identical to Go's RunHandle.Next() — full parity layer defined
- **Enforcement:** IStepRunner + runners marked internal sealed. No InternalsVisibleTo grants. Roslyn analyzers (GERT0001/GERT0002) enforce at build time
- **Redaction:** Google.Re2 (never System.Text.RegularExpressions). Named group syntax difference flagged (OQ-5)
- **Template gap:** No Go text/template equivalent in C#. Option A (minimal port), Option B (WASM-compiled Go). 10 canonical test vectors (TV-TMPL-001..010)
- **Trace:** Synchronous write inside IRunHandle.NextAsync() before dispatch. Fire-and-forget forbidden
- **Validation gates (G-01..G-10):** Go gert replay on C# traces (G-06) is strongest parity check. 60 minimum test vectors across 12 categories

### Declaration/Consent Runtime Gaps (COMPLETE)
- 10 gaps identified with priority levels:
  - P0: Signature capture (UserInputKind.Signature), identity proofing, witness flow, document versioning
  - P1: Locale/BCP-47 provenance, on-behalf-of (DeclarationPrincipal), validity expiry, revocation
  - P2: Contextual PII policy, QTSP integration
- Proposed events: DeclarationCollectedEvent, WitnessAttestedEvent, DeclarationRevokedEvent, DeclarationValidityCheckedEvent, QualifiedSignatureReceivedEvent
- Proposed interface: IWitnessGate (parallel to IApprovalGate)
- What fits well: JSONL trace, at-most-once Service Bus, RE2 redaction, client="web" tagging
- Non-goals: GERT ≠ TSP, no biometrics, no legal rendering, no jurisdiction enforcement

### Execution Adapter Patterns (COMPLETE)
- **Red lines identified:** 
  - Governance fires inside RunHandle.Next(), cannot be preserved by patterns that bypass it
  - JSONL trace written synchronously inside Runtime Core before action proceeds
  - RunHandle is stateful/sequential (no concurrent calls, no splitting across processes)
  - Per-step Durable Function activities cause governance loss (cold start of binary)
  - Approval gates & evidence submission require live process holding the RunHandle
  - client="web" audit field must be set correctly
- **Strongest match:** Queue-triggered worker (decoupled submission, worker owns full Runtime Core, trace → Blob Storage, approval via reply messages)

### Open Questions (Phase 2)
- OQ-1: Template strategy for C# (minimal port vs. WASM Go)
- OQ-2: Approval reply queue topology
- OQ-3: Canonical Go test harness
- OQ-4: Sequence counter strategy
- OQ-5: RE2 named group syntax difference ((?P<name>...) vs. C# (?<name>...))

## GIS Optional-Chaining (`?.`) — Phase 2 Ready (2026-06-05T07:28:56.273-07:00)

**Status: Spec & Conformance Locked** — Runtime implementation ready to begin

The GIS optional-chaining extension (`?.` and `?.[N]`) has been ratified and all normative specification, grammar, and conformance vectors are committed:
- **Grammar:** `design/gert/grammar/gis.ebnf` (lines 106-187, 285-297) — new GDP productions for optional-dot and optional-bracket
- **Spec:** `design/gert/sections/03b-interpolation-syntax.tex` — normative "Optional Path Chaining" section with semantics and examples
- **Conformance:** `design/gert/conformance/tv-gis-path.yaml` (15 test vectors TV-GIS-PATH-001..015)

### Key Design Decisions (Locked)
| Decision | Value |
|----------|-------|
| Default missing value | **`""` (empty string)** |
| Short-circuit semantics | **JS/TS-compatible full-tail** — once any `?.` segment misses, entire remaining chain → `""` |
| Scope | **GIS `${...}` only** — no extension to GXL or GCP |
| `?.[N]` optional bracket indexing | **IN** — consistency with field-access tolerance |
| `${?.root}` optional root | **Illegal** — root identifier always mandatory |
| No propagation through stdlib calls | **Confirmed** — GDP resolves first; functions receive `""` if chain short-circuits |

### Implementation Scope (Go + C# runtimes)
1. **Lexer/Parser:** Recognize `?.` and `?.[` tokens; update GDP production rules
2. **Evaluator:** Implement short-circuit logic — on first missing optional hop, assign `""` to entire expression
3. **Error handling:** `${a.b.c}` still raises `GIS-PATH-MISSING` on miss (no `?.`); optional paths return `""` instead
4. **Conformance:** All 15 vectors in `tv-gis-path.yaml` must pass

### Phase 2 Dependency
- This is independent of other Phase 2 work; can be implemented in parallel with template/approval/adapter work.

## Learnings
- 2026-06-04T20:14:36.949-07:00 - Stream E was reversed cleanly after Day 2 shipped: no production runbooks means no migration target, and a migrator would imply a legacy mode GERT does not have.
- 2026-06-04T20:14:36.949-07:00 - Dogfooding a tool against the source-of-truth corpus remains a valuable regression pattern; the specific migrator skill was removed, but future agents can re-derive the pattern when a real tool surface exists.
- 2026-06-05T07:28:56.273-07:00 - Ratification meetings lock complex cross-language decisions efficiently when open questions are pre-identified. `?.` coverage across both runtimes can now proceed in parallel without alignment risk.
