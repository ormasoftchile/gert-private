# Wave 2 Design Decisions Merger

This document merges decisions from Ken (§06, §10), Barbara (§07, §13), and Dennis (§15).

## §06 Runtime Events — Key Decisions

### Event Role: Fan-Out Notifications, Not Command/Control
Events are one-way notifications from Runtime Core; they don't trigger state transitions. Keeps adapters passive.

### Event Delivery Semantics
- JSONL trace: at-least-once (synchronous)
- In-process subscribers: best-effort (bounded channel, discard-on-full)

### Event Envelope: 7 Mandatory Fields
`event_id`, `run_id`, `runbook_id`, `timestamp` (RFC3339 UTC), `kind`, `sequence` (int64), `payload`

### Event Catalog: 22 Event Kinds
6 categories: Run lifecycle (3), Step lifecycle (6), Governance (4), Extension/tool (4), Human-in-loop (2), Saga/compensation (2)

### Required vs Optional Events
11 REQUIRED (lifecycle + governance), 11 OPTIONAL (observability)

### Event Channels: Three Delivery Paths
1. In-process event bus (bounded Go channel)
2. WebSocket broadcast (`/events` endpoint)
3. JSONL trace file (authoritative log)

### Replay Determinism
Historical traces re-emitted with same sequence/timestamp/kind/payload; new event_id assigned; no step execution.

---

## §10 Migration and Compatibility — Key Decisions

### v1 Compatibility Shim: 6-Month Grace Period
v2 parser accepts v1 runbooks, silently normalizes. Shim removed in v2.1.

### CLI and JSON-RPC Backward Compatibility
All CLI commands and JSON-RPC methods preserved; no breaking changes to public interfaces.

### Runbook Schema Breaking Changes: 3 Categories
1. Field promotions (meta.* → top-level)
2. Step type renames (collector → manual, router → end)
3. New required fields ($schema, id)

### Automated Migration Tool: 10-Step Algorithm
Deterministic, idempotent, fail-safe rollback on validation error.

### Tool and Provider Migration: Manual Review Required
Tool capabilities and provider field namespacing require manual review.

### Extension Handshake: v2 Protocol with v1 Compatibility
v2 requires protocol_version: "2.0", capabilities: [...]. v1 shim grants all capabilities.

### Trace Format Migration: v2 Replay Compatibility Mode
v2 gert replay reads v1 traces; synthesizes missing fields. Removed in v2.1.

### Rollout Strategy: 4-Phase Adoption Over 8 Weeks
Phase 1: validation, Phase 2: schema migration, Phase 3: extension/tool migration, Phase 4: integration migration.

---

## §07 Security & Trust — Key Decisions

### Extension Trust Levels
Three levels: Local trusted (no sig), Project-scoped (optional sig), Remote/signed (mandatory Ed25519)

### Signature Format and Verification
Ed25519 over canonical JSON. Signature block includes algorithm, publicKeyFingerprint, signatureData, signedFields, trust_chain.

### Process Isolation Mechanisms
Out-of-process isolation: unveil (OpenBSD), seccomp-bpf (Linux), host-side validation.

### Credential and Secret Handling
Secrets NEVER in traces. Providers/tools mark outputs as sensitive. Host redacts before trace write.

### RBAC Model for gert serve
Three roles: Viewer (read-only), Operator (execute), Admin (manage governance).

### Approval Token Signing and Replay Protection
Ed25519 signatures over canonical JSON; timestamp within 5-min window; single-use approvalId.

### HMAC Trace Chaining for Integrity
Optional HMAC-SHA256 chain over JSONL events for tamper-proof audit logs.

### Supply Chain Security: Tool Checksums
Tool definitions declare SHA256 checksum. Host verifies before invocation.

---

## §13 Adapter Contracts — Key Decisions

### Contract Version Format
`{surface}/{major}` format (e.g., exec/v2). No minor version; breaking change bumps major.

### Error Code Registry
5 categories: 1xxx (execution), 2xxx (governance), 3xxx (schema), 4xxx (integration), 5xxx (server)

### Contract Deprecation Policy
Previous version supported 12 months after GA. Simultaneous support during transition.

### Forward and Backward Compatibility
Clients ignore unknown fields. Servers accept requests omitting optional fields (with defaults).

### Contract Test Suite
YAML test cases in `testdata/contracts/{name}/`. `gert contract test` validates. CI gate.

### JSON-RPC Execution Contract Methods
6 methods: exec/start, exec/next, exec/cancel, exec/status, run/list, run/get

### WebSocket Event Stream Protocol
Connect to /ws, subscribe (runId, sinceSequence). Server streams JSON events. Reconnect with sinceSequence to resume.

### Tool Invocation Contract (tool/v2)
stdio JSON-RPC: invoke (params: action, inputs, context), cancel (params: invocationId, reason)

### MCP Tool Adapter
MCP servers wrapped to satisfy tool/v2. Adapter translates invoke → tools/call, forwards context as _gert_context.

### OpenAPI and JSON Schema Specifications
All contracts documented as OpenAPI 3.1, schemas as JSON Schema Draft 2020-12. Source of truth for specs.

---

## §15 Observability & Diagnostics — Key Decisions

### OpenTelemetry as the Distributed Tracing Standard
OTel for vendor neutrality, industry adoption, W3C Trace Context support, metrics/logs convergence.

### Prometheus OpenMetrics Format for Metrics
Prometheus pull model via `/metrics` endpoint; OTLP push as alternative via OTEL_METRICS_EXPORTER.

### Three-Level Span Hierarchy
Root (gert.run) → Child (gert.step) → Grandchild (gert.tool.invoke, gert.governance.check, gert.input.resolve)

### W3C Trace Context for Cross-System Correlation
traceparent and tracestate HTTP headers propagate trace context across system boundaries.

### Health Endpoint Paths Follow Kubernetes Conventions
/health (liveness), /ready (readiness), /metrics (Prometheus)

### gert diagnose Command for Pre-Flight Checks
Validates runbook schema, tool availability, provider connectivity, extension loading, governance policies.

### Sensitive Data Redaction in Logs and Traces
Inputs referred by name only; no values in logs/traces. Same redaction rules as §12 Evidence Tracing.

### Metrics Cardinality Management via Label Dimensions
Low-cardinality labels: outcome, runbook_id, step_type, tool_name, step_id (approvals only). High-cardinality data in traces/logs.

### Histogram Bucket Tuning by Operation Type
Tool (sub-sec to 30s), Step (100ms to 5min), Run (500ms to 1hr), Approval (10s to 24hr)

### Parent-Based Sampling Strategy
Inherit parent's sampling decision if trace exists; sample at configurable rate for standalone runs (default 100% dev, 1-10% prod).

### CI/CD Integration via JSON Output Mode
`gert run --output=json` emits machine-readable summary. Exit codes: 0 (success), 1 (failure), 2 (governance-blocked), 3 (cancelled).

### Integration Recipes for Common Platforms
Prometheus+Grafana, Jaeger, Loki+Grafana, Datadog, New Relic with copy-paste configs.

---

**Status:** ✅ Merged from inbox  
**Ready for:** Implementation by Ken (runtime), Barbara (security/adapters), Dennis (observability)
