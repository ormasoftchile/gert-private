# Domain Kit Model — Architectural Draft

**Author:** Ken (Software Architect)
**Date:** 2026-04-20
**Status:** Draft for Leslie (LaTeX integration)
**Target:** New §5 in the v2 design document

---

## Renumbering Note for Leslie

Inserting this chapter requires the following section shifts in `main.tex`:

| Current file                        | Current § | New § |
|-------------------------------------|-----------|-------|
| `sections/04-extension-runtime.tex` | §4        | §6    |
| `sections/05-tool-runtime.tex`      | §5        | §7    |
| `sections/06-runtime-events.tex`    | §6        | §8    |
| `sections/07-security-and-trust.tex`| §7        | §9    |
| `sections/08-testing-and-acceptance.tex` | §8   | §10   |
| `sections/09-open-questions.tex`    | §9        | §11   |
| `sections/10-migration-compatibility.tex` | §10 | §12   |
| `sections/11-governance-policy.tex` | §11       | §13   |
| `sections/12-evidence-tracing-resumption.tex` | §12 | §14 |
| `sections/13-adapter-contracts.tex` | §13       | §15   |
| `sections/14-input-provider-framework.tex` | §14 | §16  |
| `sections/15-observability-diagnostics.tex` | §15 | §17  |

A new §4 (to be drafted separately, likely a "Domain Kit Model — Concepts" or prerequisite section) may also be inserted. Confirm with Cristian before renumbering. The draft below is written to stand alone as a single chapter regardless of final numbering.

All `\ref{}` and `\label{}` cross-references in existing sections that use chapter labels (e.g., `ch:extension`, `ch:tools`) are stable — they reference label names, not section numbers. The `\S\ref{}` references in `sections/00-overview.tex` (Document Roadmap) will need a new entry for Domain Kits and the prose adjusted to reflect the expanded table of contents.

---

## Chapter: Domain Kit Model

*\chapter{Domain Kit Model}*
*\label{ch:domain-kits}*

### §5.1 — Purpose and Scope

The gert v2 core runtime is deliberately domain-agnostic. It executes steps, enforces governance, captures evidence, and writes traces — but it carries no opinion about what kind of work a runbook describes. This is a feature, not an omission: a domain-agnostic kernel is the prerequisite for a stable, long-lived runtime contract.

Domain Kits are the mechanism by which domain-specific semantics enter the system without contaminating the kernel. A Domain Kit is a higher-level semantic layer built atop core runtime primitives. It provides five things:

1. **Authoring schemas.** A Kit defines a domain-specific YAML vocabulary — specialized step types, field structures, validation constraints — that is natural for authors working in that domain. An incident-response Kit might offer a `triage` step type with severity and escalation fields; an operations Kit might offer a `change-request` type with approval-matrix fields.

2. **Validation rules.** A Kit contributes domain-specific semantic validators that run at parse time or plan time. These validators enforce invariants that the core schema cannot express: "a rollback step must follow every deploy step," "every incident runbook must declare a DRI," "escalation timeout must not exceed SLA."

3. **Lowering (compilation into core primitives).** Kit-specific step types do not reach the runtime. A Kit compiler transforms (lowers) its domain-specific AST nodes into sequences of core step types (cli, tool, manual, branch, iterate) before the planner produces an ExecutionPlan. The runtime executes only core primitives; it is never aware that a Kit was involved.

4. **Projections and read models.** A Kit may define read-only projections over run traces. An operations Kit might project a "change log" view from the raw trace events; an incident Kit might project a "timeline" view. Projections are computed after execution; they do not affect runtime behavior.

5. **Testing contracts.** A Kit defines what "correct" means for its domain: fixture runbooks, expected lowered output, expected validation errors, expected projection shapes. These contracts are the Kit's acceptance test suite.

Domain Kits do not contribute runtime capabilities. They do not register tools, subscribe to events, or modify execution behavior. Those are extension concerns (§\ref{ch:extension}). A Kit operates entirely at the authoring and analysis layer — it shapes what authors write and what reviewers see, not what the engine does.

### §5.2 — Architectural Narrative: Why Domain Kits Exist

The v2 architecture draws a hard boundary between three concerns: what the engine executes (core runtime), what capabilities are available at runtime (extensions and tools), and what vocabulary authors use to express intent (Domain Kits). The first two are well-defined in the existing design; the third has been implicit until now.

Without Domain Kits, domain semantics have two bad places to live. They can leak into the core schema, which makes the kernel unstable — every new domain (incident response, change management, compliance audits, database operations) adds fields and step types to the core, bloating it into an ever-expanding union type. Or they can be pushed into extensions, which conflates authoring concerns with runtime concerns — an extension that contributes domain-specific validators, authoring schemas, and projections is doing fundamentally different work than an extension that contributes tools or event subscriptions.

Domain Kits solve this by providing a clean, well-defined home for domain semantics. The design principle is:

- **Core owns execution.** The core runtime (§\ref{ch:architecture}), governance layer (§\ref{ch:governance}), evidence/tracing system (§\ref{ch:evidence}), and event model (§\ref{ch:events}) are stable, domain-agnostic infrastructure.
- **Kits own vocabulary.** Authoring schemas, domain validation rules, lowering/compilation, projections, and testing contracts belong to Kits. A Kit is the right place to encode "what does a well-formed incident runbook look like?" or "what fields does a change-request step require?"
- **Extensions own capabilities.** Tool registration, event subscription, schema extension (namespaced `x-*` fields), policy contribution, and provider registration belong to extensions (§\ref{ch:extension}).
- **Tools own effects.** Executable actions — invoking binaries, calling APIs, performing side-effecting work — belong to the tool runtime (§\ref{ch:tools}).

This separation preserves the kernel's stability guarantee: the core schema and runtime contract do not change when a new domain is added. The Kit compiles domain-specific vocabulary down to core primitives; the runtime never sees Kit-specific nodes.

### §5.3 — How Kits Preserve a Clean Kernel

The key architectural invariant is: **the runtime never executes Kit-specific step types.** All domain-specific authoring nodes are lowered (compiled) into core primitives before the planner sees them.

The compilation pipeline is:

```
Kit YAML → Kit Parser → Kit AST → Kit Compiler → Core YAML → Core Parser → ExecutionPlan
```

Or equivalently, when integrated into the gert pipeline:

1. The user authors a runbook using Kit vocabulary (e.g., `apiVersion: runbook/v2+ops/v1`).
2. The Kit's parser validates domain-specific fields and produces a Kit-level AST.
3. The Kit's compiler lowers the AST into a standard `runbook/v2` document using only core step types.
4. The core parser and planner process the lowered document normally.
5. The runtime executes the plan with no knowledge that a Kit was involved.

This is a compile-time transformation, not a runtime interception. The lowered output is a valid `runbook/v2` document that can be inspected, version-controlled, and executed without the Kit installed. This property is critical for portability: a Kit-authored runbook can always be reduced to its core equivalent.

The Kit compiler must be deterministic: the same Kit input must always produce the same core output. This ensures that `gert test` scenarios written against lowered output remain stable across Kit versions.

### §5.4 — Layer Relationship Clarification

The v2 system has four distinct layers. Each layer has a defined responsibility boundary and a defined relationship to the others.

| Layer              | Owns                                                        | Does NOT own                           | Runs at         |
|--------------------|-------------------------------------------------------------|----------------------------------------|-----------------|
| **Core Runtime**   | Execution, state machine, governance enforcement, evidence, event emission, trace writing | Domain vocabulary, tool implementations, UI rendering | Run time |
| **Domain Kits**    | Authoring schemas, domain validation, lowering/compilation, projections, testing contracts | Execution, runtime capabilities, side effects | Author time / compile time |
| **Extension Runtime** | Runtime capabilities: tool registration, event subscription, schema extension (`x-*`), policy contribution, provider registration | Authoring vocabulary, domain semantics, execution control | Run time (out-of-process) |
| **Tool Runtime**   | Side-effecting actions: binary execution, API calls, MCP tool invocation | Governance, evidence, schema, authoring vocabulary | Run time (per-invocation or persistent process) |

Key distinctions:

- **Domain Kits are NOT extensions.** Extensions contribute runtime capabilities via the capability grant model (§\ref{sec:ext-capabilities}). Kits contribute authoring semantics via compilation. An extension runs as a child process during execution; a Kit runs as a compiler before execution. They serve fundamentally different purposes and operate at different phases.

- **Domain Kits are NOT tools.** Tools perform side-effecting work. Kits are pure transformations with no side effects. A Kit compiler reads YAML and writes YAML; it never invokes binaries, calls APIs, or modifies system state.

- **Extensions may accompany a Kit but are distinct.** A Kit distribution (e.g., an "incident response pack") may ship both a Domain Kit (authoring schemas, lowering rules) and an extension (incident-specific tools, PagerDuty providers). These are bundled for convenience but are architecturally separate: the Kit operates at compile time, the extension operates at run time.

### §5.5 — Kit Manifest and Compatibility

A Domain Kit ships a `gert-kit.yaml` manifest alongside its compiler. The manifest declares identity, compatibility, and the Kit's schema contributions.

```yaml
# gert-kit.yaml — Domain Kit Manifest
apiVersion: kit/v1

meta:
  name: gert.ops                     # Reverse-DNS namespaced identifier
  version: "1.0.0"                   # Semver
  description: "Operations runbook authoring kit"
  author: "Gert Project <gert@ormasoft.dev>"

compatibility:
  gert-core: ">=2.0.0 <3.0.0"       # Minimum gert core version (semver range)

schema:
  apiVersion-suffix: "ops/v1"        # Runbooks authored with this Kit use
                                     # apiVersion: runbook/v2+ops/v1
  json-schema: "./schemas/ops-v1.json"  # Kit-specific JSON Schema overlay

compiler:
  executable: "./bin/gert-kit-ops"   # Kit compiler binary
  args: ["compile"]
  input-format: yaml                 # Kit YAML in, core YAML out
  output-format: yaml

step-types:                          # Domain-specific step types this Kit defines
  - name: change-request
    lowers-to: [manual, tool]        # Core types this compiles into
  - name: rollback
    lowers-to: [cli, branch]
  - name: triage
    lowers-to: [manual, branch]

validators:                          # Domain-specific validation rules
  - name: ops/rollback-follows-deploy
    phase: plan                      # parse | plan
    description: "Every deploy step must be followed by a rollback step"
  - name: ops/dri-required
    phase: parse
    description: "Every ops runbook must declare a DRI in meta.roles"

projections:                         # Read models over trace data
  - name: ops/change-log
    description: "Change management timeline derived from trace events"
  - name: ops/evidence-summary
    description: "Audit-ready evidence summary grouped by governance primitive"
```

**Compatibility and versioning rules:**

- A Kit declares a semver range for the minimum gert core version in `compatibility.gert-core`. If the host version falls outside this range, the Kit is rejected at compile time with a structured error.
- Breaking changes to a Kit's authoring schema require a Kit major version bump. The `apiVersion-suffix` encodes the Kit's major version (e.g., `ops/v1` → `ops/v2`). This is independent of the gert core version.
- Additive changes (new optional fields, new step types) within a Kit major version are backward-compatible and do not require a version bump to the suffix.
- The Kit's `json-schema` artifact carries a semver artifact version for minor/patch tracking, following the same pattern as core schema artifacts (§\ref{subsec:versioning-strategy}).

**Local-first constraints:**

Domain Kits are portable semantic layers. They run on any supported gert host — a local laptop, a CI runner, a self-hosted server — without requiring network access, cloud services, or central registries. A Kit is a directory containing a manifest, a compiler binary, and schema artifacts. It can be vendored into a repository, distributed as a tarball, or installed from a package manager. There is no Kit registry service, no Kit telemetry, and no phone-home behavior. This aligns with gert's local-first design philosophy (§\ref{ch:overview}).

### §5.6 — Extension Runtime Audit

The following table audits concerns currently described in the extension runtime chapter (§\ref{ch:extension}) and evaluates whether each is properly an extension concern, a Domain Kit concern, or shared.

| Concern | Current location | Recommendation | Rationale |
|---------|-----------------|----------------|-----------|
| **Domain-specific validators** | Implicit in `capability/schema-extension` | **Move to Kit.** | Validation rules that enforce domain invariants (e.g., "rollback must follow deploy") are authoring-time concerns. They belong in the Kit's validation phase, not in a runtime extension. The extension `capability/schema-extension` should be reserved for namespaced field contributions (`x-*`), not for semantic validation of domain workflows. |
| **Authoring schema registration** | `capability/schema-extension` allows extensions to contribute fields | **Clarify boundary.** Extensions contribute namespaced runtime-observable fields (`x-acme.*`). Kits contribute domain-specific authoring vocabulary (new step types, required field structures). These are different operations. Add a clarifying note to §\ref{sec:ext-capabilities} that `capability/schema-extension` is for runtime-observable namespace extensions, not for authoring-vocabulary contributions. |
| **Projections / read models** | Not currently addressed in extension runtime | **Kit concern.** Projections are pure, read-only transformations over trace data. They are computed after execution and do not require runtime capabilities. They belong in the Kit layer. Extensions that need to observe execution should use `capability/event-subscription`; post-hoc analysis belongs to Kits. |
| **Compilation hooks** | Not currently addressed | **Kit concern.** The lowering/compilation step is a pure transformation that happens before the runtime starts. It has no runtime footprint and requires no capabilities. Compilation is entirely within the Kit layer. |
| **Domain testing fixtures** | Not currently addressed | **Kit concern.** Test fixtures for domain-specific authoring (example runbooks, expected lowered output, expected validation errors) are Kit deliverables. They do not interact with the extension runtime. |
| **Tool registration** | `capability/tool-registration` | **Stays in extension runtime.** Tool registration is a runtime capability. A Kit distribution may ship an accompanying extension that registers domain-specific tools, but tool registration itself remains an extension concern. |
| **Policy contribution** | `capability/policy-contribution` | **Stays in extension runtime.** Policy rules that are evaluated at run time (pre-step, pre-run) are runtime concerns. Domain-specific validation rules that are evaluated at parse/plan time belong to Kits; runtime policy enforcement belongs to extensions. |
| **Event subscription** | `capability/event-subscription` | **Stays in extension runtime.** Event observation is inherently a runtime activity. No change needed. |

**Summary of changes needed in existing sections:**

1. Add a clarifying paragraph to `capability/schema-extension` (§\ref{sec:ext-capabilities}) noting the boundary: extensions contribute namespaced runtime fields; Kits contribute authoring vocabulary and domain step types.
2. Add a forward reference from the extension runtime chapter to the Domain Kit chapter for readers who arrive at the extension spec looking for domain-specific validation or projections.
3. No capabilities need to be moved or removed from the extension taxonomy. The taxonomy is correct for runtime concerns; the issue was that domain/authoring concerns had no defined home. Domain Kits provide that home.

### §5.7 — Kit-0: The DRI / Operations Runbook Model

The current DRI / operations runbook model — the step types, governance primitives, evidence capture patterns, and approval workflows defined in the core schema (§\ref{ch:schema}) and governance chapter (§\ref{ch:governance}) — is gert's first Domain Kit. It is Kit-0: the operations domain that motivated gert's existence.

In the current design, Kit-0 is implicit. Its step types (cli, manual, tool, branch, iterate), governance primitives (command allowlists, env blocking, redaction, approval gates), and evidence types (text, choice, attachment with SHA256) are baked into the core schema. This is acceptable for v2.0 because the operations domain is gert's primary use case and the Kit model is being introduced as a roadmap item, not a v2.0 deliverable.

The architectural intent is clear: as the Kit model matures (v2.1+), the operations-specific authoring vocabulary should be extractable into a standalone Kit (`gert.ops`) that compiles into core primitives. The core schema would then contain only the minimal execution primitives (step sequencing, variable binding, governance hooks, evidence slots), and the operations vocabulary (DRI roles, change-request workflows, incident triage patterns) would live in the Kit. This extraction is not required for v2.0 but should be planned for.

Kit-0 grounds the Domain Kit concept in something concrete. When explaining Kits to implementors or users, the answer to "what does a Kit look like?" is: "look at the operations runbook model you already use — that is a Kit."

### §5.8 — Non-Goals and Explicit Boundaries

Domain Kits are a precisely scoped mechanism. The following are explicitly not in scope:

- **Kits are not extensions.** A Kit does not contribute runtime capabilities. It does not register tools, subscribe to events, modify execution behavior, or run as a child process during execution. The extension capability taxonomy (§\ref{sec:ext-capabilities}) does not apply to Kits.

- **Kits are not tools.** A Kit does not perform side-effecting work. It does not invoke binaries, call APIs, or modify system state. The Kit compiler is a pure function: YAML in, YAML out.

- **Kits are not plugins in a generic sense.** Gert does not have a "plugin system." It has three distinct extension points — Domain Kits (authoring semantics), extensions (runtime capabilities), and tools (effectful actions) — each with a different contract, lifecycle, and trust model. Calling any of these "plugins" obscures the architectural boundaries.

- **Kits are not templates or examples.** A Kit is not a collection of example runbooks. It is a semantic layer: schemas, validators, a compiler, projections, and testing contracts. An example runbook may demonstrate a Kit's vocabulary, but the Kit itself is the machinery that makes that vocabulary valid and executable.

- **Kits do not own execution.** The core runtime is the sole authority over step advancement, state management, governance enforcement, evidence capture, and trace writing. A Kit may influence what the runtime executes (via lowering), but it never participates in how execution proceeds.

- **Kits do not replace the core schema.** The core `runbook/v2` schema remains the stable contract between authors, the parser, and the runtime. Kit schemas are overlays that compile down to core; they do not bypass or replace it.

- **Kits are not cloud-native constructs.** Consistent with gert's local-first philosophy, Kits do not require network access, remote registries, or cloud services. A Kit is a local artifact — a directory with a manifest, a compiler, and schemas — that runs wherever gert runs.
