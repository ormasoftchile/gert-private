# Overview

Gert v2 is a governed, executable, debuggable runbook engine that solves the core operational triad: runbooks must be **executable** (not merely documents), **governed** (every infrastructure action bounded by policy), and **traceable** (every execution produces an immutable, auditable record satisfying SOC 2, ISO 27001, and HIPAA requirements by construction). v1 validated the thesis and delivered governance, evidence capture, TUI, VS Code extension, and scenario replay — but accumulated architectural constraints (tight coupling, no stable extension contracts, no saga/compensation, no OpenTelemetry, no parallel fan-out) that require a clean-break v2.

## Problem Statement

- Unstructured runbooks drift from reality, skip governance checks, produce no audit evidence.
- Three required properties: **executable**, **governed**, **traceable**.
- Without all three: audit failures, runbook drift, uncontrolled blast radius.

## Why v2: What v1 Got Right, and What Constrained It

**Preserved from v1:**
- Command allowlists, env-var blocking, output redaction, per-step approval gates
- Append-only JSONL trace, SHA256-hashed evidence attachments
- Deterministic replay (`--mode replay`), `gert test` scenario runner
- Bubble Tea TUI, VS Code extension (JSON-RPC 2.0), interactive step-through REPL

**v1 constraints requiring v2:**
- `gert serve` tightly coupled to runtime — adding adapters required core changes
- Schema versioning brittle (`runbook/v0`, `runbook/v1`) with no formal compatibility contract
- No stable extension contract — third-party tool development fragile
- No saga/compensation pattern for rollback on failure
- No parallel fan-out execution
- No OpenTelemetry support
- No policy-as-code integration

## Scope

**Gert v2 is:** a local-first, governed runbook execution engine with a stable component architecture, versioned extension contracts, a first-class governance and policy layer, full traceability by construction, and defined adapter interfaces for TUI, VS Code, and web clients. Ships as a single Go binary with an optional out-of-process extension runtime.

**Gert v2 is not:** a managed cloud execution service, a general-purpose workflow orchestrator, a CI/CD pipeline system, a SOAR platform, or a replacement for Temporal, Argo, or PagerDuty Runbook Automation. Does not provide a central policy server, a shared execution registry, or multi-tenant execution isolation.

## Design Priorities

- **Core runtime isolated from presentation adapters.** The execution engine (parser, planner, state machine, governance enforcer, trace writer) has no dependency on any presentation layer. TUI, VS Code, and web clients attach via a stable event subscription API; they observe execution but never own it.

- **Explicit and versioned contracts for schema, tools, and events.** Every interface crossing a component boundary is named, versioned, and documented. The runbook schema carries an `apiVersion` field with formal compatibility guarantees. Tool definitions carry a version. The runtime event stream has a defined schema with ordered delivery guarantees. Breaking a contract requires a version increment.

- **Secure extension runtime using out-of-process plugins.** Extensions run in isolated processes and are granted capabilities explicitly. No extension receives ambient authority. The capability model is a named, versioned list that the host enforces at admission time.

## Document Roadmap

| Section | Content |
|---------|---------|
| §Goals | Measurable goals and explicit non-goals |
| §Architecture | Five primary components and dependency rules |
| §Schema | v2 runbook schema, versioning strategy, field inventory, v1 migration |
| §Extension | Extension runtime: capability taxonomy, handshake protocol, trust model |
| §Tools | Tool runtime: discovery, invocation, transport contracts, governance integration |
| §Events | Runtime event model: schema, ordering, delivery semantics, replay contract |
| §Security | Threat model, policy enforcement, supply chain controls, audit requirements |
| §Testing | Contract test categories, coverage requirements, acceptance criteria |
| §Questions | Open architectural questions with options and resolution criteria |

## Success Definition

Gert v2 is complete when:
1. A v1 runbook can be migrated to v2 schema via `gert migrate` and executed without manual fixup.
2. The TUI, VS Code extension, and a third-party adapter can all be built against the published event and API contracts **without importing internal packages**.
3. The governance layer (allowlists, denylists, env blocking, redaction, approval gates) is enforced for **every** execution mode including extension-invoked runs.
4. Every execution produces an append-only, crash-safe JSONL trace that is structurally valid against the v2 trace schema.
5. `gert test` passes a scenario suite covering all step types, governance rules, and evidence capture cases.
