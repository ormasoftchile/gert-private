# barbara — History

## Project Context

**Project:** gert — Governed Executable Runbook Engine
**Owner:** ormasoftchile
**Mission:** Redesign gert from scratch as v2, applying learnings from v1.
**Stack:** Go, YAML, JSON Schema (Draft 2020-12), LaTeX (design docs), TypeScript (VS Code extension)

## Current Focus

The team is working on the **v2 design document** located at `design/gert-v2/`.
This is a LaTeX document using the MastersThesis class. Sections are in `design/gert-v2/sections/`.

### v2 Goals
1. Extensive research: runbooks, workflows, governance, traceability
2. Apply research + project learnings to define the new version
3. Document the new design as a usable reference to build v2

### v1 Key Features (for reference)
- Validate/execute/debug runbooks in YAML
- Governance: approval gates, allowlists, output redaction
- Evidence capture with SHA256, append-only JSONL traces
- VS Code extension + TUI + JSON-RPC server
- Tool definitions (.tool.yaml) and input providers (.provider.yaml)
- Step types: cli, manual, tool, invoke, branch, iterate

## Key Files
- `design/gert-v2/main.tex` — root LaTeX document
- `design/gert-v2/sections/` — all section .tex files
- `design/gert-v2/MastersThesis.cls` — document class
- `ext/` — Go source (core engine)
- `vscode/` — VS Code extension (TypeScript)

## Learnings


## Cross-Agent Notes from Ken's Architectural Review (2026-04-18)

**For Barbara (Integrations):** §05 Tool Runtime section has important gaps. Needs detailed specifications for:

1. **Tool discovery algorithm** — How are tool definitions found in v2? Still by convention `tools/<name>.tool.yaml`? Or registry-based?
2. **Transport layer specification** — stdio/jsonrpc/mcp exist in v1. Do all three survive in v2? Any new transports?
3. **Invocation envelope** — What fields are in the "run context and telemetry" mentioned? Full spec needed.
4. **Error envelope schema** — "Structured and machine-readable" is not a spec. Define the exact error shape.
5. **Capability gates** — What checks happen during tool invocation? Who enforces them?
6. **Tool versioning** — Can a runbook pin a tool version? How?
7. **Timeout/cancellation** — What is the contract for timeouts? How does cancellation propagate to tools?
8. **Output capture** — How does stdout/stderr map to captured variables in v2?
9. **Built-in vs. extension-contributed tools** — What's the difference in §05's treatment?
10. **MCP tool integration** — MCP tool discovery is dynamic in v1. How does this interact with static governance policies in v2?

Ken's assessment: §05 is "Important" severity with 10 major gaps. This is a direct blocker for Barbara's tool/provider contract design and for Brian's tool runtime implementation. See full gap analysis at `.squad/tmp/ken-gap-analysis.md` (§05 section, lines 112–130). Decisions documented at `.squad/decisions.md` (search "Ken's Gap Analysis Findings").

## Cross-Agent Notes from Dennis's Research Brief (2026-04-18)

**For Barbara (Integrations):** Research identifies OPA integration and structured error envelopes as key integration findings.

- **OPA Integration**: OPA is CNCF standard, cloud-agnostic, and enables centralized governance policy across tools. Integration hooks: pre-execution, pre-step, post-step. OPA server mode for policy evaluation.
- **Structured Error Envelopes**: Industry tools (AWS, Temporal, Argo) use structured error formats with error codes, context, and metadata. Enables consistent error handling across tool/provider integrations.
- **OpenTelemetry Integration Points**: Emit spans per step (start, end, attributes, status). Trace context propagation across invoked runbooks and tools. Correlation IDs in all log entries.
- **Policy Examples**: "Commands including kubectl delete require 2 approvals", "Production runbooks require change-manager role"

Full research brief: `.squad/tmp/dennis-research-brief.md`
Decision inbox: `.squad/decisions/inbox/dennis-research-foundations.md` → merged to `.squad/decisions.md`

## Learnings

### 2026-04-18 — §04, §05, §14 authored

**What was done:**
Wrote three complete integration sections for the gert v2 design document:
- **§04 Extension Runtime**: replaced stub and dangling spec reference with full spec covering capability taxonomy (10 capabilities), `gert-extension.yaml` manifest format, discovery model (4 sources: built-in, workspace config, runbook declaration, CLI override), full JSON-RPC handshake protocol (initialize/contributions/list/ping/shutdown), versioning/compatibility rules, and sandboxing (process isolation, capability enforcement, timeout/crash handling).
- **§05 Tool Runtime**: replaced stub with full tool discovery algorithm (4-source resolution order with name collision handling), v2 `.tool.yaml` schema, all three transports (stdio/stdio-jsonrpc/mcp) with invocation and response envelopes, invocation context model, capability gate, output capture with redaction, MCP tool integration, cancellation/timeout protocol, and tool versioning.
- **§14 Input Provider Framework**: written from scratch — `provider/v2` definition schema, complete resolution protocol (prefix match → batch → JSON-RPC → fallback), built-in providers (env/file/prompt/workspace), provider composition chains, and provider lifecycle (persistent vs per-resolution, startup/shutdown/crash handling).

**Key design decisions made:**
- The spec at `specs/002-extension-runtime-v0/spec.md` exists; the design section now explicitly references it as the normative spec rather than claiming it doesn't exist.
- Provider definitions use `apiVersion: provider/v2` (not tool/v2) to preserve distinct identity.
- MCP transport is supported for providers as well as tools (consistency across the integration surface).
- Provider composition chains declared as YAML lists on `from:`.
- All three transports share the same invocation context envelope fields.
- Resolution caching scoped to `run | step | none` with per-provider declaration.

**Files changed:**
- `design/gert-v2/sections/04-extension-runtime.tex`
- `design/gert-v2/sections/05-tool-runtime.tex`
- `design/gert-v2/sections/14-input-provider-framework.tex`

**Build status:** Clean rebuild, 142 pages, no multiply-defined labels, no new errors.

### 2026-04-18 — §07, §13 authored (Wave 2)

**What was done:**
Wrote two complete security and integration sections for the gert v2 design document:
- **§07 Security and Trust**: replaced stub with comprehensive security specification covering threat model (7 threats with mitigations mapped to NIST/ISO controls), extension trust model (3 trust levels: local trusted, project-scoped, remote/signed with Ed25519 signature verification), process isolation (filesystem/network/environment isolation, crash isolation), credential and secret handling (sensitive marking, mandatory redaction, no secret storage), transport security (TLS for non-localhost, bearer token and mTLS authentication, WebSocket security), RBAC for `gert serve` (3 roles: viewer/operator/admin, role assignment, approval authorizer flow with signed approval tokens), audit non-repudiation (governance decision recording with Ed25519 signatures, HMAC trace chaining, SHA256 evidence capture), and supply chain security (tool binary checksums, extension signature verification, `gert verify` command).
- **§13 Adapter Contracts**: replaced stub with complete contract specification defining what an adapter contract is, four contract categories (JSON-RPC execution contract `exec/v2` with 6 methods, WebSocket event stream contract `events/v2` with subscribe/reconnect, tool invocation contract `tool/v2` with invoke/cancel/MCP adapter, extension handshake contract `extension/v2` with initialize/ping), contract versioning (version format `{surface}/{major}`, breaking vs non-breaking changes, 12-month deprecation policy, forward/backward compatibility), canonical error code registry (5xxx categories: execution, governance, schema, integration, server errors), contract test suite (YAML test cases in `testdata/contracts/`, `gert contract test` command, CI integration), reference implementations (VS Code extension for exec/events, built-in tools for tool contract, MCP adapter), and contract documentation (OpenAPI 3.1 specs, JSON Schema validation, client library generation).

**Key design decisions made:**

§07 Security and Trust:
- Extension trust levels: local trusted (no signature), project-scoped (optional signature), remote/signed (mandatory Ed25519 signature with trust chain).
- Signature verification: Ed25519 over canonical JSON of signed fields, public key fingerprint lookup in trust store, notBefore/notAfter validation.
- Process isolation enforced via unveil (OpenBSD), seccomp (Linux), and host-side validation on all platforms.
- Environment variable isolation: only `governance.allowed_env` forwarded, `deny_env_vars` takes precedence.
- Secret handling: `sensitive: true` marking on provider outputs, `sensitive_inputs: [...]` on tool actions, mandatory redaction before trace write.
- RBAC roles: viewer (read-only), operator (execute/approve/cancel), admin (manage governance/roles).
- Approval tokens signed with Ed25519; replay protection via single-use approvalId; 5-minute clock skew window.
- HMAC trace chaining: H_i = HMAC-SHA256(key, H_{i-1} || E_i) for cryptographic integrity.
- Tool binary checksum verification: SHA256 hash in `.tool.yaml`, verified before invocation.
- `gert extension verify` and `gert verify` commands for manual/CI validation.

§13 Adapter Contracts:
- Contract version format: `{surface}/{major}` (e.g., `exec/v2`, `events/v2`), no minor version.
- Version declaration: `X-Gert-Contract` HTTP header for HTTP/WebSocket, `protocolVersion` field for stdio.
- Error code registry: 1xxx execution, 2xxx governance, 3xxx schema, 4xxx integration, 5xxx server.
- Breaking changes require major version bump; non-breaking changes (add optional fields, add methods, add error codes) are minor.
- Deprecation policy: 12 months support for old major version after new major GA.
- Forward compatibility: clients MUST ignore unknown fields; servers MUST accept omitted optional fields.
- Contract test suite: YAML test cases, `gert contract test --adapter=<name>`, CI integration.
- OpenAPI 3.1 specs for HTTP contracts, JSON Schema (Draft 2020-12) for validation, code generation tooling.
- MCP tool adapter wraps MCP `tools/call` to satisfy `tool/v2` contract, forwards `_gert_context` as MCP extension.

**Files changed:**
- `design/gert-v2/sections/07-security-and-trust.tex` (replaced stub, ~570 lines)
- `design/gert-v2/sections/13-adapter-contracts.tex` (replaced stub, ~620 lines)

**Build status:** Not compiled (Leslie will build after all Wave 2 agents complete).

### 2026-04-18 — Wave 2 Complete: Orchestration and Merging

**Scribe Task:** Final Wave 2 orchestration completed by Scribe agent.

**What was done:**
1. Merged Wave 2 decision inbox files (§07, §13) into decisions.md
2. Created orchestration logs for all Wave 2 agents
3. Updated agent histories with Wave 2 completion note
4. Prepared git commit with all Wave 2 sections

**Status:** ✅ Wave 2 COMPLETE  
All 16 sections written. Design document ready for final PDF build and stakeholder review.


### 2026-04-18 — §14 Enhanced for Three Interactive Step Types

**What was done:**
Enhanced §14 (Input Provider Framework) to define provider contracts for the three new interactive step types that replace `manual`:
- **choice**: User selects from fixed labeled options, result stored as variable
- **decision/router**: User selects a path for control flow routing
- **collector**: User provides unstructured input (text, files, attachments, URLs)

**Key additions to §14:**
- New section "Interactive Step Type Contracts" (§14.6) defining JSON-RPC request/response envelopes for each step type
- **Choice contract**: `provider/choice` method with options array, default selection, validation rules
- **Decision contract**: `provider/decision` method with routes array, no variable storage (flow control only), provider doesn't need graph knowledge
- **Collector contract**: `provider/collect` method with multi-field forms, file upload support, artifact storage contract
- Provider capability matrix showing which built-in providers support which step types (only `prompt` supports all three)
- Capability advertisement during provider handshake via `capabilities` object
- Specified file upload constraints: MIME types (images, documents, archives, logs), size limits, SHA-256 verification
- Artifact storage contract: metadata fields (filename, contentType, sizeBytes, sha256, storagePath)

**Design decisions made:**
- Providers advertise step type support in `provider/initialize` handshake
- If a configured provider doesn't support a step type, engine falls back to built-in `prompt` provider
- Decision steps do NOT store the selected route as a variable (flow control only)
- Collector artifacts include SHA-256 hash for integrity verification
- File size limits enforced by provider before storage (not by engine)
- Built-in `prompt` provider stores artifacts in `.gert/runs/<runId>/artifacts/`
- External providers MAY use remote storage (S3, Azure Blob) and return signed URLs

**Files changed:**
- `design/gert-v2/sections/14-input-provider-framework.tex` (+332 lines, now 670 lines total)

**Build status:** Section 14 syntax correct. Existing build error in different section (line 965-973, unrelated to this change).

**Cross-reference:** John is updating §03 (schema spec) to define these three step types in the step type taxonomy.

---

## 2026-04-18 — Team Sync: Step Type Refactor Complete

**Status:** ✅ Merged to decisions.md

**Cross-team coordination:**
- **John** (Schema): Completed §03 schema spec updates
  - Replaced generic `manual` step type with choice/decision/collector
  - Full normative specs for all three types with field constraints
  - Updated migration rules: early adopters must refactor manual steps based on intent
  - Document now builds to 256 pages

- **Ken** (Architecture): Completed comprehensive cross-section review (§00–§15)
  - Event envelope field names standardized (seq→sequence, type→kind, data→payload)
  - Added missing event_id field to §12 trace envelope
  - Wire format convention documented: snake_case (JSONL traces) vs camelCase (JSON-RPC)
  - 5 critical issues fixed; 12 minor documentation gaps identified

**Provider integration impact:**
- §14 now fully specifies JSON-RPC contracts for all three interactive step types
- Capability matrix enables graceful fallback when provider doesn't support step type
- File upload and artifact storage models finalized
- Artifact integrity verification (SHA-256) embedded in contract

**Next: Implementation begins**
- Brian (Parser): Implement parser/validator for choice/decision/collector
- Ken (Runtime): Implement execution semantics for three step types
- Sam (VS Code): Implement UI rendering for interactive steps
