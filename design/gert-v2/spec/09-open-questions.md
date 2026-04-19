# Open Questions

Twelve unresolved architectural decisions that affect core design choices. Each question states why it matters, the options under consideration, and what information is needed to close it. Decisions MUST be recorded in `.squad/decisions.md` before core contract freeze.

---

## Q1. Can the v2 runtime execute v1 runbooks without prior migration?

**Why it matters:** Existing users have production runbooks in `runbook/v0` and `runbook/v1` format. A compatibility execution mode ("v1 shim") removes the migration barrier but adds permanent maintenance cost to the parser.

**Options:**
- (a) **Full compatibility mode:** v2 runtime transparently executes v1 runbooks by internally upgrading them to v2 schema at load time, with no schema file written.
- (b) **Migration gate:** v2 refuses to run v1 runbooks and provides a clear error pointing to `gert migrate`.
- (c) **Opt-in compatibility flag:** `--compat-v1` enables the shim; production is v2-only.

**Information needed:** A survey of how many v1-format runbooks exist in representative deployments, and whether `gert migrate` can be made fully lossless for all v1 constructs.

---

## Q2. What is the concurrency model for the core runtime?

**Why it matters:** The v1 runtime is single-run-per-process. Whether v2 supports multiple concurrent runs in a single process affects state isolation model, trace writer design, the governance enforcer (per-run or global?), and whether `gert serve` can multiplex concurrent VS Code sessions.

**Options:**
- (a) **Single-run-per-process**, matching v1 semantics; each `gert exec` or `gert tui` is a separate OS process.
- (b) **Multiple concurrent runs in-process**, with run-scoped state isolation via Go context and per-run goroutine groups.
- (c) **Multiple runs via a long-lived daemon** (`gert serve` as the host), with runs as logical sessions.

**Information needed:** Whether the VS Code extension use case requires concurrent runs in a single `gert serve` process (e.g., multiple open runbook panels), and goroutine safety requirements for the trace writer and governance layer.

---

## Q3. Is the v2 trace format backward-compatible with the v1 JSONL trace, or is it a new format?

**Why it matters:** Operators and compliance tools may have parsers, SIEM integrations, or dashboards built against the v1 JSONL event schema. A breaking change requires those integrations to be updated before v2 can be rolled out.

**Options:**
- (a) **New format:** v2 defines a clean JSONL schema with a required `schema_version` field; v1 trace parsers are unsupported.
- (b) **Backward-compatible superset:** v2 trace events include all v1 fields plus new fields; existing parsers continue to work on the subset they understand.
- (c) **Parallel trace output:** v2 writes both a v2 trace and an optional v1 compatibility trace (flag-enabled).

**Information needed:** An inventory of known downstream trace consumers and whether the v1 format has fields that would be structurally incompatible with the v2 event model.

---

## Q4. How do extension-contributed policies compose with host-level governance policies?

**Why it matters:** The host enforces a governance policy (command allowlist, env blocking, redaction). An extension may also wish to contribute policies. The composition model determines whether extensions can weaken host policies (prohibited), extend them (likely desired), or operate in a separate policy namespace.

**Options:**
- (a) **Additive composition only:** extensions may only add restrictions (denylist entries, redaction patterns); they may never remove host-level restrictions.
- (b) **Policy namespacing:** each extension has an isolated policy scope; the host applies all active policies via intersection.
- (c) **Policy delegation:** the host can explicitly delegate a policy domain to an extension (e.g., redaction rules owned by a data-classification extension).

**Information needed:** A threat model analysis of what happens when a malicious or compromised extension contributes a policy that attempts to weaken host restrictions. This feeds into Security and Trust.

---

## Q5. Does the `.provider.yaml` input provider framework survive unchanged in v2?

**Why it matters:** v1's input provider model (`.provider.yaml`, JSON-RPC resolution, `from:` bindings, workspace config) is a complete and working framework. It must either be carried forward as-is, redesigned to fit the v2 component model, or replaced by an extension-based resolution mechanism. The choice affects the schema, the extension runtime, and any operator with a custom provider binary.

**Options:**
- (a) **Preserve as-is:** `.provider.yaml` format and JSON-RPC protocol are unchanged; providers are a stable first-class concept.
- (b) **Redesign as extension capability:** providers become a named capability that extensions may implement; the `.provider.yaml` format is deprecated.
- (c) **Hybrid:** `.provider.yaml` providers continue to work as a built-in extension type, with a migration path to the full extension protocol.

**Information needed:** Whether any v1 provider binaries are in active production use that would be broken by a protocol change, and whether the extension handshake protocol can be made a strict superset of the provider JSON-RPC protocol.

---

## Q6. Is the `gert serve` JSON-RPC contract a versioned breaking change or an evolutionary extension?

**Why it matters:** The VS Code extension consumes the `gert serve` JSON-RPC 2.0 API over stdio. If v2 changes method names, parameter shapes, or notification payloads, the extension must be updated simultaneously. A versioned negotiation protocol allows the extension to evolve independently of the binary.

**Options:**
- (a) **Hard break:** v2 defines a new JSON-RPC method set; the VS Code extension is updated as part of the v2 launch.
- (b) **Versioned negotiation:** the `initialize` handshake includes a protocol version field; the host advertises capabilities and the client selects a compatible version.
- (c) **No change:** the v1 JSON-RPC surface is preserved unchanged in v2.

**Information needed:** A complete method inventory of the v1 JSON-RPC API surface (all methods and notification shapes currently used by the VS Code extension), and which methods require changes to expose v2-specific features (e.g., saga events, OTel trace IDs).

---

## Q7. What is the extension supply chain and signing model?

**Why it matters:** Extensions run in separate processes but are granted capabilities that include tool registration, schema field contribution, and event subscription. A malicious or tampered extension binary could exfiltrate captured secrets, inject commands, or suppress governance events. An extension signing model raises the bar for supply chain attacks but adds operational friction for local development.

**Options:**
- (a) **No signing required;** extensions are trusted if present in the declared extension directory.
- (b) **Optional signing:** extensions may carry a detached signature; the host warns but does not block unsigned extensions.
- (c) **Mandatory signing for capability classes above a threshold** (e.g., extensions requesting `governance.policy` capability must be signed).
- (d) **Keyless signing via Sigstore/cosign** for developer convenience.

**Information needed:** A threat model for the extension trust boundary (see Q4), and a survey of whether the target operator community has PKI infrastructure that could issue extension signing keys.

---

## Q8. Is a gRPC transport for the VS Code extension viable for v2.0, or should it be deferred?

**Why it matters:** The current JSON-RPC 2.0 over stdio transport limits the VS Code extension to a single active connection and requires the extension to manage the gert process lifecycle. A gRPC transport would enable streaming (server-push events without polling), multiple concurrent clients, and connection multiplexing. However, it adds protocol complexity and requires a port allocation model.

**Options:**
- (a) **Retain JSON-RPC stdio** as the sole transport in v2; gRPC deferred to a future version.
- (b) **Add gRPC as an opt-in alternative transport** alongside JSON-RPC stdio.
- (c) **Replace JSON-RPC stdio with gRPC** as the primary VS Code transport; provide a backward-compat shim for the transition period.

**Information needed:** Benchmarking of event delivery latency over JSON-RPC stdio versus gRPC under realistic VS Code workloads (step execution at 100ms/step, TUI refresh at 60fps equivalent). Also requires determination of whether VS Code's Node.js extension host supports gRPC client libraries without native module issues.

---

## Q9. What is the parallel execution model for independent steps within a single run?

**Why it matters:** v1 executes all steps sequentially. Many real runbooks contain independent diagnostic steps that could execute in parallel, reducing wall-clock time. However, parallelism complicates trace ordering guarantees, the governance enforcer (can two parallel steps both consume the same approval token?), and TUI rendering.

**Options:**
- (a) **No parallelism in v2;** fan-out deferred to a future version.
- (b) **Explicit parallel blocks** via a new `parallel:` step type that joins on completion of all children.
- (c) **Implicit parallelism** inferred from the step dependency graph (steps with no data dependency on preceding steps execute concurrently).

**Information needed:** Whether any v1 runbooks in active use have independent steps that would benefit from parallelism, and the governance implications for concurrent step execution (e.g., two CLI steps executing simultaneously under the same allowlist).

---

## Q10. How should the saga compensation registry interact with the trace and approval model?

**Why it matters:** A compensation registered for a step must itself be subject to governance (it may execute privileged commands), must appear in the trace (compensations are audit events), and may require its own approval gate if the original step had one.

**Options:**
- (a) **Compensations execute with the same governance context as the original step** (same allowlist, same approver roles).
- (b) **Compensations are exempt from approval gates** (compensations are emergency rollback actions; blocking them on approval would defeat their purpose).
- (c) **Compensation governance is separately configurable** per compensation definition.

**Information needed:** Industry precedent from Temporal (compensation activities run in the same workflow context) and AWS Step Functions (catch/retry blocks have no separate IAM policy). A decision feeds into the schema (compensation field design) and Security and Trust (governance scope).

---

## Q11. What is the schema namespace convention for extension-contributed fields, and how are they validated?

**Why it matters:** The v2 schema states that extension fields are namespaced, but the namespace convention and validation mechanism are undefined. If two extensions contribute fields under the same namespace, or if the host does not validate extension-contributed fields against a schema, runbooks using those fields will have unpredictable runtime behavior.

**Options:**
- (a) **Extension ID as namespace prefix** (`x-myext.fieldName`); validated against a JSON Schema fragment published by the extension at registration time.
- (b) **Dot-notation namespace under a reserved top-level key** (`extensions.myext.fieldName`); host validates at parse time.
- (c) **Extension fields are opaque to the host;** the extension validates its own fields at execution time.

**Information needed:** Whether JSON Schema Draft 2020-12 supports dynamic `$ref` resolution at validation time, which would allow the host to assemble a composite schema from the host schema plus all registered extension schema fragments.

---

## Q12. What is the timeout and escalation model for human approval steps?

**Why it matters:** v1 approval gates block indefinitely. In production, a stalled approval can block an incident response indefinitely. SLA enforcement (timeout after N minutes, escalate to the next role, optionally auto-approve or auto-reject) is a standard pattern in ITIL emergency change processes and is implemented by PagerDuty and AWS Step Functions human approval tasks.

**Options:**
- (a) **No timeout in v2;** approval escalation deferred.
- (b) **Optional `timeout` field on approval steps;** on expiry, run is suspended and operator is notified.
- (c) **Full escalation chain:** `timeout` triggers escalation to a secondary role list; if that also times out, the step enters a configurable terminal state (auto-approve, auto-reject, or suspend-for-manual-review).

**Information needed:** Whether the v2 runtime concurrency model (Q2) supports timer-based step suspension without blocking the process, and what notification mechanism is available to alert escalation targets (stdout, webhook, or an extension-contributed notifier).
