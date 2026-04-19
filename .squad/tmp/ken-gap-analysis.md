# Ken — Gap Analysis: gert v2 Design Sections
**Date:** 2026-04-18  
**Author:** Ken (Software Architect)  
**Audience:** Leslie (LaTeX), Dennis (Research), John (Schema), Barbara (Integrations), Brian (Go)

---

## Preamble

All ten sections have been reviewed against: (a) the v1 feature set in `README.md`, (b) the v1 package structure in `ext/`, and (c) the team decisions log in `.squad/decisions.md`. The verdict is uniform and stark: **every section is a skeleton placeholder, not a buildable specification.** The document currently captures intent, not design. No section contains enough detail to drive implementation. The following analysis is intended to close that gap.

---

## Section-by-Section Review

---

### §00 — Overview

**Current state:** Five sentences and three bullet points. States that v2 is a "clean-break architecture" and lists three design priorities. No problem statement, no motivation, no scope boundary, no relationship to v1.

**Gaps:**
- No problem statement — *why* is v2 needed? What failed in v1 that warrants a clean break?
- No explicit scope: what is gert v2 (a new binary? a new schema version? a fully separate system)?
- No relationship to v1: migration story is entirely absent
- No document roadmap: how do the nine sections relate to each other?
- No success definition: what does "done" look like for v2?
- No target audience statement

**Severity: Critical**

---

### §01 — Goals and Non-Goals

**Current state:** Four goals and three non-goals, stated as abstract principles. No measurable outcomes, no rationale linking goals to v1 pain points.

**Gaps:**
- Goals are not measurable — "deterministic core runtime" is aspiration, not a criterion
- No coverage of v1 capabilities that must be *preserved* (governance, evidence capture, tracing, replay, TUI, VS Code extension, `gert test`)
- No coverage of v1 pain points that drove this redesign (coupling between runtime and serve layer, schema versioning brittleness, etc.)
- Missing goals: traceability/audit, migration compatibility, observability, operator UX
- Non-goals are process-oriented, not feature-oriented — "preserve legacy constraints" is not a useful non-goal for builders
- No performance or scale goals

**Severity: Critical**

---

### §02 — Architecture

**Current state:** Five numbered components and one dependency rule. No diagrams, no interfaces, no data flow, no lifecycle description.

**Gaps:**
- No component interface definitions — what methods/contracts does each component expose?
- No data flow description — how does a runbook go from YAML → parse → plan → execute → emit events → write trace?
- No package/module boundaries — how do the five components map to Go packages?
- No lifecycle model — how does a run start, pause, resume, and terminate?
- No concurrency model — is the runtime single-threaded per run? goroutine-per-step?
- No description of the "planner" — what does it produce? A plan DAG? A flat step list with resolved branches?
- "API Layer" is mentioned but never defined — what is its shape? JSON-RPC? gRPC? Go interfaces?
- No component diagram or architecture illustration
- No description of how adapters (TUI, Web, VS Code) attach to the core — via events? direct calls? both?
- Missing: how `gert serve` fits the new architecture (it was a critical v1 coupling point)

**Severity: Critical**

---

### §03 — Schema vNext

**Current state:** Three bullets and one rule. States that core fields are minimal, extension fields are namespaced, and validation is split into phases. No field definitions.

**Gaps:**
- No schema field inventory — what are the core language fields in v2? How do they differ from v1?
- No versioning strategy — what is `apiVersion: runbook/v2`? What breaks from v1?
- No namespace convention for extension fields — just "namespaced" is not enough
- No migration path from `runbook/v0` and `runbook/v1` — must `gert migrate` be updated?
- No definition of what "structural" vs. "semantic" validation means concretely
- No schema contract artifact referenced or defined (the section says it must exist but doesn't point to it)
- No coverage of step type changes — do `cli`, `tool`, `manual`, `invoke` survive unchanged?
- No coverage of new step types or removed step types
- No expression/template language specification (Go templates currently — staying in v2?)
- No tool schema vNext (`tool/v1`?) — does the tool definition format change?
- No provider schema definition
- Missing: how schema extensions (namespaced fields) are registered and validated

**Severity: Critical**

---

### §04 — Extension Runtime

**Current state:** Four key decisions (transport, packaging, trust, safety) and a reference to `specs/002-extension-runtime-v0/spec.md`. This is the most developed section but still lacks a protocol definition.

**Gaps:**
- The normative spec reference (`specs/002-extension-runtime-v0/spec.md`) does not exist in the repository — this is a dangling reference
- No capability taxonomy — what capabilities can be granted? File access? Network? Tool registration? Schema registration? Event subscription?
- No handshake/lifecycle protocol — what JSON-RPC methods are used? What is the initialization sequence?
- No versioning strategy for extension contracts — how does a v1 extension run in v2 host?
- No extension manifest format defined — what fields are required?
- No extension discovery model — where does the host look for extensions? How are they declared?
- No extension compatibility matrix — semver? Custom?
- No sandboxing specifics — OS-level isolation? Process isolation only?
- No coverage of built-in extensions vs. third-party extensions
- No extension upgrade/downgrade policy

**Severity: Critical**

---

### §05 — Tool Runtime

**Current state:** Four requirements stated as bullet points. No actual tool invocation protocol, no transport details, no discovery model.

**Gaps:**
- No tool discovery algorithm — how are tool definitions found in v2? Still by convention `tools/<name>.tool.yaml`?
- No transport layer specification — stdio/jsonrpc/mcp exist in v1; do all three survive in v2? Any new transports?
- No invocation envelope definition — what fields are in the "run context and telemetry" mentioned?
- No error envelope schema — "structured and machine-readable" is not a spec
- No capability gate implementation — what checks happen? Who enforces them?
- No tool versioning model — can a runbook pin a tool version?
- No timeout/cancellation contract for tools
- No tool output capture contract — how does stdout/stderr map to captured variables in v2?
- No difference from extension-contributed tools vs. built-in tools vs. project tools
- No coverage of MCP tool discovery changes (MCP tool registration is dynamic in v1; how does this interact with static governance policies?)

**Severity: Important**

---

### §06 — Runtime Events and Determinism

**Current state:** Five event category names and one constraint (adapters must not implement private execution logic). No event schema, no ordering guarantees, no payload definitions.

**Gaps:**
- No event schema — what fields does each event have? (type, runId, stepId, timestamp, payload?)
- No event ordering guarantees — total order? per-step order? eventual?
- No event IDs or correlation model — how do consumers link events to runs and steps?
- No replay semantics — how are events used to deterministically replay execution?
- No event bus/delivery model — push vs. pull? buffered? guaranteed delivery?
- The five categories are names only — no events listed within each category
- No coverage of `event/branchResolved` and `event/iteratePassEnd` (already implemented and decided in the decisions log — this design is already behind the codebase)
- No consumer contract — how do adapters subscribe to events?
- No back-pressure or slow-consumer handling
- No persistence model for events — are they written to the trace file? Separately?
- Missing: how the event model relates to the existing append-only JSONL trace format

**Severity: Critical**

---

### §07 — Security and Trust

**Current state:** Three trust model principles and three runtime hardening rules. Correct at the philosophy level but provides no implementation guidance.

**Gaps:**
- No capability taxonomy (must be defined here or cross-referenced from §04)
- No threat model — what attack vectors are considered? Malicious extension? Compromised tool? Env var exfiltration?
- No policy DSL or policy schema — how is "policy" defined for extension admission and operation?
- No audit trail specification — how are security events recorded? Format? Retention?
- No authentication model — who runs a runbook? `--as engineer@company.com` in v1 — what changes in v2?
- No authorization model — can certain users run certain runbooks? Can governance be role-scoped?
- No supply chain model — how are extensions verified/signed? (§09 asks this as an open question but §07 should frame the problem)
- Missing: governance layer design — this is a first-class v1 feature (allowlists, denylists, env blocking, redaction) and is entirely absent from v2 design

**Severity: Critical**

---

### §08 — Testing and Acceptance

**Current state:** Four contract test categories and four acceptance criteria. The criteria are correct but insufficiently specific to drive a test plan.

**Gaps:**
- No test framework specified — Go `testing`? testify? Custom harness?
- No coverage percentage requirements
- No integration test strategy — how are end-to-end run scenarios tested?
- No regression testing strategy — how are v1 runbooks used as v2 test fixtures?
- No performance acceptance criteria — what latency is acceptable for tool invocation, event delivery?
- Acceptance criteria #3 ("Capability violations fail fast with structured errors") needs specific test cases
- No contract for extension test harness — how does an extension developer test their extension?
- No CI/CD integration guidance
- Missing: scenario replay testing (v1's `gert test` feature) — does it survive in v2 and how?

**Severity: Important**

---

### §09 — Open Questions

**Current state:** Three questions. Correct in identifying some uncertainty but dramatically under-populated given the gaps above.

**Gaps:**
- The three questions are implementation-level choices; deeper architectural questions are not listed
- Missing questions: How does v2 handle v1 runbooks at runtime? Is there a migration mode? What is the concurrency model for the core runtime? How are provider inputs resolved in v2? What is the trace format in v2 (new format or backward-compatible)? What is the deprecation plan for `gert serve`'s existing JSON-RPC contract? How does the policy engine compose with extension-contributed policies?
- No resolution timeline or owner assigned to existing questions
- Question about gRPC transport should be higher-priority given the existing VS Code extension dependency on JSON-RPC stdio

**Severity: Nice-to-have** (the questions are fine; the list just needs expansion)

---

## Cross-Cutting Gaps

### Entirely missing topics

1. **Migration and Compatibility** — v1 has three schema versions (`runbook/v0`, `runbook/v1`, `tool/v0`). The design says nothing about how v2 handles existing runbooks. `gert migrate` exists in v1 — is there a `gert migrate-v2`? This is a launch blocker for any real-world adoption.

2. **Governance Layer** — This is arguably gert's most distinctive feature: allowlists, denylists, env var blocking, output redaction, approval gates. The word "governance" does not appear in any v2 design section. This is a critical omission.

3. **Evidence Capture and Traceability** — v1's append-only JSONL trace, per-step state snapshots, SHA256 evidence hashing, and run resumption are entirely absent from the v2 design. These are load-bearing operational features.

4. **Input Provider Design** — v1 has a complete input provider framework (`.provider.yaml`, JSON-RPC resolution, workspace config). No v2 equivalent is described.

5. **Runbook Lifecycle** — There is no description of the full execution lifecycle: plan → validate → execute → pause/resume → complete → archive. Specific gaps: How does `--resume` work in v2? What state is checkpointed?

6. **Adapter Contracts** — §02 names TUI, Web, and VS Code as adapters but defines no contract for how they attach to the core. No event subscription API, no render-cycle contract, no initialization handshake.

7. **Observability and Telemetry** — Structured logging, metrics, and tracing (OpenTelemetry?) are unaddressed. §07 mentions "structured diagnostics" but this is not a design.

8. **Deployment and Packaging** — How is gert v2 distributed? Single binary? Plugin directory? Extension registry? This matters for §04 (extension discovery) and §05 (tool discovery).

9. **Concurrency Model** — The core runtime's concurrency contract is never stated. Single-run-per-process? Multiple concurrent runs? Goroutine safety guarantees?

10. **`gert serve` / RPC Contract** — v1's JSON-RPC server is the backbone of the VS Code extension. The decisions log has multiple entries about the serve layer. v2 design never mentions it.

### Inconsistencies between sections

- **§04 vs §05**: Extension Runtime and Tool Runtime overlap — tools in v1 are registered by extensions (MCP tools are dynamically discovered). The boundary between "extension contributes a tool" and "tool is a first-class citizen" is undefined.
- **§02 vs §06**: Architecture names an "API Layer" but Events defines adapters as pure consumers. It's unclear whether adapters call the API Layer or subscribe to events or both.
- **§04 vs §07**: Security says "deny by default with capability grants" but §04 doesn't define *what* capabilities exist. These sections must be authored together.
- **§06 vs decisions log**: The decisions log already documents `event/branchResolved` and `event/iteratePassEnd` as implemented events, but §06 doesn't list a single concrete event. The design is behind the codebase.
- **§03 vs §08**: §03 says validation is split into structural and semantic phases but §08's contract tests don't test this split specifically.

---

## Priority Recommendations

The following seven items must be addressed before this document is usable as a build reference:

**1. Write a Migration and Compatibility section (new §10)**  
This is a launch blocker. Every operator with a v1 runbook needs to know what breaks, what's automatically migrated, and what requires manual effort. Without this, v2 is unusable by existing users.

**2. Define the governance layer design (expand §07 or new §11)**  
Governance is gert's core differentiator. Allowlists, denylists, env blocking, output redaction, and approval gates must be specified in v2 — whether they are preserved as-is, redesigned, or moved to an extension. Currently the design reads as if governance doesn't exist.

**3. Define the complete architecture data flow (expand §02)**  
The five components named in §02 need: (a) interface definitions, (b) a data flow walkthrough (YAML → execution → events → trace), (c) a package boundary map. Without this, Brian (Go) cannot build and John (Schema) cannot validate their section against runtime semantics.

**4. Define the event schema (expand §06)**  
Every adapter is event-driven by design. The event schema (fields, payloads, ordering guarantees, consumer API) must be specified before any adapter work begins. The decisions log already has real events — they need to be promoted into the design.

**5. Write the extension capability taxonomy (expand §04 + §07)**  
"Deny by default with capability grants" means nothing without a list of what capabilities exist. This must be a named, versioned list that Brian can implement and extension authors can reference.

**6. Define the schema vNext field inventory (expand §03)**  
John needs a concrete starting point. The section must list: (a) core fields retained from v1, (b) fields removed or breaking-changed, (c) new fields, (d) the namespace convention for extension fields, (e) the structural/semantic validation split definition.

**7. Resolve the dangling spec reference in §04**  
`specs/002-extension-runtime-v0/spec.md` is referenced as the normative contract but does not exist. Either write it or remove the reference. A section that points to a nonexistent normative spec cannot be implemented.

---

## New Section Proposals

The following sections should be added to the document:

### §10 — Migration and Compatibility
Covers the migration path from `runbook/v0` and `runbook/v1` to the v2 schema. Specifies which v1 fields are renamed, removed, or restructured, and what `gert migrate` (v2 version) must do. Includes a compatibility matrix and a definition of whether v2 can execute v1 runbooks in a compatibility mode.

### §11 — Governance and Policy
A first-class section for gert's core differentiator. Specifies the v2 governance model: what policy fields survive from v1, how policy is evaluated (host-enforced vs. extension-contributed), how approval gates work in the event model, and how redaction is applied to captured outputs and trace events.

### §12 — Evidence, Tracing, and Resumption
Covers the append-only trace format (JSONL v2 schema), per-step state snapshot model, SHA256 evidence capture, and the run-resumption contract. Specifies how trace events relate to runtime events from §06, and what the trace file guarantees (append-only, crash-safe, tamper-evident).

### §13 — Adapter Contracts
Defines the concrete interface that TUI, Web, and VS Code adapters implement against the core. Covers: event subscription API, render lifecycle, initialization handshake, and the constraint that adapters own no execution logic. This is needed before any adapter work begins.

### §14 — Input Provider Framework
Covers the v2 input provider model: how `from:` bindings are resolved, how providers are declared and configured, the JSON-RPC resolution protocol, and how provider failures are handled. Should address whether v1 `.provider.yaml` format is preserved.

### §15 — Observability and Diagnostics
Covers structured logging format, metric points, distributed tracing integration (OpenTelemetry?), and the structured diagnostics model for extension interactions. Needed to make §07's hardening claims implementable.

---

## Summary Assessment

The current document is a valid **table of contents with design intent statements**. It correctly identifies all the right component categories. It is not a design document. Every section needs at least 3–5× more content before Brian can write Go or John can write schema. The most severe omissions are: governance (entirely missing), migration (entirely missing), data flow (entirely missing), event schema (named but undefined), and the dangling spec reference. Leslie should treat every section as "needs author pass" and Dennis should prioritize research on governance models and trace formats to fill the evidence capture gap.
