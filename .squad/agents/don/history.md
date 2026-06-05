# Don — Project History (Summarized)

## Overview
Don is the migration/tooling and runtime engineer. Work spans fixture migration (Stream D), migration tool construction (Stream E), declaration runtime gaps, C# governance parity, and execution adapter validation.

## Key Milestones (Phase 1)

### Stream E — Migration Tool & Idempotency (Days 1-2 — COMPLETE)
- **Day 1:** gert migrate-expr scaffolded with 11 translation rules (E-001..011), 31 tests passing
  - CLI via cobra, position-aware YAML traversal (gopkg.in/yaml.v3), expression-position detection (when/condition/until/iterate.over)
  - Rules: GIS templates (E-001..003), GXL operators (E-004..006), contains → stdlib (E-007), jq-style paths (E-008), legacy escapes (E-009), template pipes warn (E-010), now() fix (E-011)
  - Flags: --dry-run, --diff, --report JSON, --strict
- **Day 2 (Dogfood):** 22 runbooks verified with 0 translations post-fix
  - **Real idempotency bug fixed:** migrated $${amount} was being reclassified as E-009 legacy on second pass. Now preserves $${symbol} when symbol known from context (inputs/captures/collector fields)
  - **E-007 contains ambiguity:** String-like → str.contains(), list-like → list.contains(), ambiguous → warning (heuristic only, no auto-rewrite)
  - **E-006 extended:** Quote-aware negation scanner covers !X, !(X), !(X && Y), !!(X || (Y && Z)), !!X
  - **Validation:** gert migrate-expr --dry-run on testdata: 0 translations, go test ./... passing
- **Stream E criteria:** Idempotent ✓, ambiguous rewrites warning-only ✓, strict mode coverage TBD, docs/help match behavior TBD

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
