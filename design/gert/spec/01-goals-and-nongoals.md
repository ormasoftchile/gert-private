# Goals and Non-Goals

This section defines the binding commitments of the v2 design (verifiable criteria, not aspirations) and the explicit out-of-scope boundaries. Implementors MUST satisfy all goals; the non-goals define where scope ends and prevent feature creep.

## Goals

### G1. Preserve and Formalize the Governance Layer

Every capability from v1 — command allowlists/denylists, environment variable blocking, output redaction with regex patterns, per-step role-based approval gates — MUST be present in v2 with identical or stronger semantics. v2 additionally MUST support RBAC scoping on runbook execution: which identities may run which runbooks, and under what conditions.

### G2. Preserve the Append-Only, Crash-Safe, Tamper-Evident Trace

The v2 execution trace MUST be an append-only JSONL file. Each line is a self-describing JSON object with:
- Unique event ID (UUID)
- ISO 8601 UTC timestamp
- Actor identity
- Step ID
- Structured payload

The file MUST be crash-safe (partial writes MUST NOT corrupt prior entries) and MUST support optional cryptographic signing to satisfy SOC 2 and ISO 27001 (A.12.4) audit requirements. **This is a launch blocker: no trace, no compliance.**

### G3. Preserve Evidence Capture, Deterministic Replay, TUI, VS Code Extension, and `gert test`

The following v1 capabilities MUST be functionally preserved:
- Evidence capture with text, checklists, and SHA256-hashed attachments
- Scenario-based deterministic replay (`--mode replay`)
- Bubble Tea terminal UI
- VS Code extension powered by JSON-RPC server
- `gert test` scenario runner with assertion engine

Their absence in v2 is a **regression**, not a simplification.

### G4. Establish Stable, Versioned Extension Contracts

- Tool definitions (`.tool.yaml`), input providers (`.provider.yaml`), and out-of-process extensions MUST bind to named, versioned contracts.
- A host version bump that breaks an extension contract requires a **major version increment** and a migration path.
- The capability taxonomy (what an extension may register, observe, and invoke) MUST be a published, enumerated list — not an implicit set of permissions inferred from code.

### G5. Introduce Saga and Compensation for Multi-Step Rollback

v2 MUST support explicit compensation registration: when a forward step completes, the runbook MAY declare a compensating action to execute in reverse order on failure. This is the industry-standard saga pattern. Without compensation, partially-executed runbooks that modify infrastructure leave systems in undefined intermediate states.

### G6. Integrate OpenTelemetry for Distributed Tracing

Every runbook execution MUST emit OpenTelemetry spans:
- One root span per run
- One child span per step
- W3C Trace Context (`traceparent` / `tracestate`) propagated to tool invocations

Correlation IDs MUST appear in both the JSONL trace and OTel span attributes, enabling run traces to be joined with infrastructure observability data in Jaeger, Zipkin, or any OTLP-compatible backend.

### G7. Decouple Adapters from Core via a Typed Event API

- The TUI, VS Code extension, and any future web adapter MUST attach to the core exclusively via a typed event subscription interface.
- No adapter MAY import internal runtime packages or call runtime methods directly.
- The event schema MUST be versioned: adapters declare which schema version they consume; the host rejects incompatible connections.
- This constraint enables concurrent adapters (e.g., TUI and VS Code running simultaneously against a single run).

### G8. Establish Migration Compatibility for v1 Runbooks

- `gert migrate` MUST produce a valid `runbook/v2` document from any well-formed `runbook/v0` or `runbook/v1` input with **no manual fixup required** for standard runbooks.
- The v2 runtime SHOULD additionally support a compatibility execution mode that accepts v1 runbooks without prior migration.

## Non-Goals

### NG1. No Managed Cloud Execution Service

No gert-as-a-service, no central execution registry, no SaaS offering. Gert v2 is a local-first binary. Cloud scheduling, multi-tenant isolation, and remote execution orchestration are out of scope.

### NG2. No General-Purpose Workflow Orchestrator

Gert is not Temporal, Argo, or Prefect. No distributed workers, no fan-out parallelism across remote nodes, no arbitrary DAG execution. Parallel execution within a single local run is a stretch goal; cross-host parallelism is explicitly out of scope.

### NG3. No Built-In Policy Server

v2 MAY integrate policy-as-code hooks, but does not bundle or operate an OPA server. Policy evaluation happens at the host process level. Central policy distribution, policy bundles over HTTP, and multi-host policy synchronization are out of scope.

### NG4. No Extension Marketplace or Signing Authority

v2 defines an extension signing *mechanism* (a verifiable public key model), but does not operate a registry, curate extensions, or act as a certificate authority. Extension distribution is the responsibility of the extension author.

### NG5. No Binary Backward Compatibility with v1 Internal Packages

v2 is a clean-break architecture. Only the documented public contracts carry compatibility commitments:
- Runbook schema
- Tool schema
- JSON-RPC API surface
- Extension protocol
- JSONL trace schema
