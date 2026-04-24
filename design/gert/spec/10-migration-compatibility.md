# Migration and Compatibility

Gert v2 is a breaking redesign from v1. This section specifies the migration path from `runbook/v0` and `runbook/v1` to the v2 schema, defines what `gert migrate` (v2 version) must accomplish, and establishes compatibility guarantees for existing runbooks, tools, providers, and integrations. Migration is a launch blocker for adoption: every operator with a v1 runbook must understand what breaks, what is automatically migrated, and what requires manual effort.

---

## v1 → v2 Compatibility Matrix

| v1 Feature/Interface | Status | v2 Change | Migration Notes |
|---|---|---|---|
| `runbook/v0` schema | BREAKING | Removed | Migrate to v1 first, then v2 |
| `runbook/v1` schema | BREAKING | Promoted fields | `gert migrate` auto-rewrites |
| `provider/v0` schema | BREAKING | New namespace | Manual rewrite required |
| `tool/v0` schema | BREAKING | New fields | `gert migrate` rewrites |
| CLI: `gert run` | COMPATIBLE | New flags added | All v1 flags preserved |
| CLI: `gert test` | COMPATIBLE | Unchanged | |
| CLI: `gert serve` | COMPATIBLE | New RPC methods | v1 methods preserved |
| CLI: `gert validate` | COMPATIBLE | v2 schema aware | |
| CLI: `gert migrate` | COMPATIBLE | v0→v1→v2 path | New `--to-v2` flag |
| JSON-RPC: `exec/*` | COMPATIBLE | Additive | New methods; old methods unchanged |
| JSON-RPC: `runbook/*` | COMPATIBLE | Additive | |
| JSONL trace format | BREAKING | New envelope | See Trace Format Migration |
| Extension handshake | BREAKING | New capability fields | See Extension Migration |
| Event stream schema | BREAKING | New event kinds | Old events preserved |
| VS Code extension protocol | COMPATIBLE | Backward compat | v2 extension reads v1 traces |
| TUI adapter | COMPATIBLE | Event-based | |

**Key takeaways:**
- All **CLI commands and flags** are backward compatible. Existing scripts using `gert run` continue to work.
- All **JSON-RPC methods** are backward compatible. The v2 `gert serve` daemon accepts v1 client requests.
- The **runbook YAML schema** has breaking changes (field promotions, renames). Automated migration provided via `gert migrate`.
- The **tool and provider schemas** have breaking changes. Manual updates required.
- The **JSONL trace format** has breaking changes (new envelope fields). Historical trace readers must be updated.
- The **extension handshake** has breaking changes. Extension authors must adopt the v2 protocol.

---

## Schema Migration: Runbook YAML

### Breaking Changes Summary

The v1 → v2 runbook schema migration involves three categories of change:

1. **Field promotions:** Seven `meta.*` fields are promoted to top-level.
2. **Step type renames:** `collector` → `manual`, `router` → `end`.
3. **New required fields:** `$schema`, `id` (derived from `name` if absent).

All other runbook semantics are preserved. Step execution behavior, governance primitives, evidence capture, and input/output contracts remain unchanged.

### Before and After Example

**v1 runbook (`simple.runbook.yaml`):**

```yaml
apiVersion: runbook/v1

meta:
  name: service-health-check
  kind: operational
  description: Check service health and restart if needed
  vars:
    service_name: "api-server"
  governance:
    allowed_commands: [systemctl, curl, jq]

flow:
  - step:
      id: check_status
      type: cli
      title: Check service status
      with:
        argv: ["systemctl", "status", "{{ .service_name }}"]
      capture:
        exit_code: exit_status

  - step:
      id: prompt_restart
      type: collector
      title: Service is down. Restart?
      evidence:
        - kind: text
          name: decision

  - step:
      id: restart_service
      type: cli
      when: "{{ eq .decision \"yes\" }}"
      with:
        argv: ["systemctl", "restart", "{{ .service_name }}"]
```

**v2 runbook (migrated):**

```yaml
$schema: "https://schemas.gert.dev/runbook/v2.json"
apiVersion: runbook/v2

id: service-health-check
name: Service Health Check
kind: operational
description: Check service health and restart if needed

vars:
  service_name: "api-server"

governance:
  allowed_commands: [systemctl, curl, jq]

flow:
  - step:
      id: check_status
      type: cli
      title: Check service status
      with:
        argv: ["systemctl", "status", "{{ .service_name }}"]
      capture:
        exit_code: exit_status

  - step:
      id: prompt_restart
      type: manual
      title: Service is down. Restart?
      evidence:
        - kind: text
          name: decision

  - step:
      id: restart_service
      type: cli
      when: "{{ eq .decision \"yes\" }}"
      with:
        argv: ["systemctl", "restart", "{{ .service_name }}"]
```

### Field Mapping Table

| v1 Path | v2 Path | Notes |
|---|---|---|
| `meta.name` | `name` | Promoted to top-level; now capitalized (human-readable) |
| `meta.kind` | `kind` | Promoted |
| `meta.description` | `description` | Promoted |
| `meta.vars` | `vars` | Promoted |
| `meta.inputs` | `inputs` | Promoted |
| `meta.governance` | `governance` | Promoted |
| `meta.prose` | `prose` | Promoted |
| `tree` | `flow` | Renamed (tree was v0 legacy) |
| `step.type: collector` | `step.type: manual` | Renamed |
| `step.type: router` | `step.type: end` | Renamed |
| `tools: [string]` | `toolRefs: [object]` | Structured (§3.5) |

---

## Automated Migration Tool

### Command Specification

```bash
# Preview migration (dry-run)
gert migrate --to-v2 --dry-run runbook.yaml

# Migrate a single runbook in-place
gert migrate --to-v2 runbook.yaml

# Migrate all runbooks in a directory tree
gert migrate --to-v2 ./runbooks/

# Output migrated YAML to stdout (for review)
gert migrate --to-v2 --output - runbook.yaml

# Check if migration is needed (exit 1 if needed, 0 if already v2)
gert migrate --to-v2 --check runbook.yaml
```

### Flags

| Flag | Type | Description |
|---|---|---|
| `--to-v2` | bool | Enable v1→v2 migration mode |
| `--dry-run` | bool | Preview changes without writing |
| `--output PATH` | string | Write to file or `-` for stdout |
| `--check` | bool | Exit 1 if migration needed, 0 otherwise |
| `--force` | bool | Overwrite existing v2 files |
| `--backup` | bool | Create `.v1.bak` backup before rewrite (default: true) |

### Migration Algorithm

The migration tool performs the following transformations in order:

1. **Parse input:** Read YAML with v1 parser. If `apiVersion` is already `runbook/v2`, exit 0 (no-op).
2. **Version check:** If `apiVersion` is `runbook/v0`, emit error: "v0 must be migrated to v1 first".
3. **Add `$schema`:** Insert `$schema: "https://schemas.gert.dev/runbook/v2.json"` at document top.
4. **Update `apiVersion`:** Change `runbook/v1` → `runbook/v2`.
5. **Derive `id`:** If no `meta.name`, derive from filename. Convert to kebab-case (e.g., `service_health` → `service-health`).
6. **Promote `meta.*` fields:** Extract `meta.name`, `meta.kind`, etc. and move to top-level. Capitalize `name` for human readability.
7. **Rename step types:** `collector` → `manual`, `router` → `end`.
8. **Rename `tree`:** If `tree:` is present, rename to `flow:`.
9. **Rewrite `tools`:** Convert shorthand `tools: [name]` to structured `toolRefs: [{name, version?}]`.
10. **Validate:** Parse the migrated YAML with the v2 parser. If validation fails, rollback and report error.
11. **Write output:** If `--output` is specified, write to that path; otherwise rewrite in-place.

### Diff Report

```
Migration report for service-health-check.runbook.yaml:
  [CHANGE] apiVersion: runbook/v1 → runbook/v2
  [ADD]    $schema: "https://schemas.gert.dev/runbook/v2.json"
  [ADD]    id: "service-health-check"
  [MOVE]   meta.name → name
  [MOVE]   meta.kind → kind
  [MOVE]   meta.description → description
  [MOVE]   meta.vars → vars
  [MOVE]   meta.governance → governance
  [RENAME] step "prompt_restart": type collector → manual
  [OK]     3 steps validated successfully

Summary: 8 changes applied. No errors.
```

### Error Handling

If migration fails, the tool:
1. Rolls back any partial changes (if in-place mode).
2. Emits a structured error message with file, line, and remediation advice.
3. Exits with status 1.

Example error:

```yaml
ERROR: Migration failed for service-health-check.runbook.yaml
  Line 14: step "check_status" has invalid v2 field "with.argv"
  Suggestion: v2 uses "with.command" for CLI steps. Update manually.
```

---

## Provider and Tool Migration

### Tool Definitions (`.tool.yaml`)

The v1 `tool/v0` schema is replaced by `tool/v2`. Key changes:

1. **New required field:** `transport` (one of `stdio`, `jsonrpc`, `mcp`).
2. **Renamed field:** `executable` → `command` (aligns with runbook YAML).
3. **New field:** `capabilities` (list of required capability grants; see §4).

**v1 tool definition:**

```
apiVersion: tool/v0
name: kubectl
executable: /usr/local/bin/kubectl
args_template: "{{ .args }}"
```

**v2 tool definition:**

```
$schema: "https://schemas.gert.dev/tool/v2.json"
apiVersion: tool/v2

id: kubectl
name: kubectl
transport: stdio
command: /usr/local/bin/kubectl
args_template: "{{ .args }}"
capabilities:
  - exec:command
  - net:egress
```

`gert migrate --to-v2` rewrites tool definitions automatically but **requires manual review** for the `capabilities` field (the tool cannot infer required capabilities statically).

### Provider Definitions (`.provider.yaml`)

The v1 `provider/v0` schema is replaced by `provider/v2`. Key changes:

1. **Namespace convention:** Provider fields now use a `provider/*` namespace to avoid name collisions with core schema fields.
2. **New resolution protocol:** v2 providers declare a `resolve_method` (one of `static`, `env`, `prompt`, `jsonrpc`).

**v1 provider definition:**

```yaml
apiVersion: provider/v0
name: incident-id
source: env
env_var: INCIDENT_ID
```

**v2 provider definition:**

```yaml
$schema: "https://schemas.gert.dev/provider/v2.json"
apiVersion: provider/v2

id: incident-id
name: Incident ID
resolve_method: env
provider/env:
  var_name: INCIDENT_ID
  required: true
```

`gert migrate --to-v2` partially rewrites provider definitions but **requires manual namespace updates** for custom provider fields.

---

## Extension Migration

### Handshake Protocol Changes

The v2 extension handshake introduces new capability negotiation. Extensions MUST update their initialization sequence to:

1. Declare supported `protocol_version` (set to `"2.0"`).
2. Declare required `capabilities` (list of capability names).
3. Handle `CapabilityDenied` responses gracefully.

**v1 handshake (JSON-RPC):**

```
→ {"jsonrpc":"2.0","method":"initialize","params":{"version":"1.0"}}
← {"jsonrpc":"2.0","result":{"ready":true}}
```

**v2 handshake (JSON-RPC):**

```
→ {"jsonrpc":"2.0","method":"initialize","params":{
    "protocol_version":"2.0",
    "capabilities":["fs:read","net:egress"]
  }}
← {"jsonrpc":"2.0","result":{
    "granted_capabilities":["fs:read"],
    "denied_capabilities":["net:egress"]
  }}
```

### Extension Compatibility Shim

The v2 Extension Host provides a **v1 compatibility shim**:
- Extensions that declare `protocol_version: "1.0"` are accepted.
- The shim grants all capabilities by default (insecure, but backward-compatible).
- A deprecation warning is logged: *"Extension 'foo' uses deprecated v1 protocol. Update to v2 by YYYY-MM-DD."*
- The shim will be removed in v2.1 (approximately 6 months after v2.0 release).

### Extension Author Checklist

Extension authors MUST:
1. Update `protocol_version` to `"2.0"` in the `initialize` method.
2. Declare all required capabilities in the handshake.
3. Test against the v2 Extension Host in capability-denied scenarios.
4. Update extension manifest to declare minimum gert version (`min_gert_version: "2.0.0"`).
5. Review the capability taxonomy (§4.4) and minimize requested capabilities.

---

## Trace Format Migration

### Breaking Changes in JSONL Trace

**v1 trace format:**

```
{"seq":1,"type":"run/started","timestamp":"...","data":{...}}
```

**v2 trace format:**

```
{"event_id":"...","run_id":"...","runbook_id":"...","timestamp":"...",
 "kind":"run/started","sequence":1,"payload":{...}}
```

**Field mapping:**

| v1 Field | v2 Field | Notes |
|---|---|---|
| `seq` | `sequence` | Renamed |
| `type` | `kind` | Renamed |
| `data` | `payload` | Renamed |
| (none) | `event_id` | New (UUID v4) |
| (none) | `run_id` | New (extracted from file path in v1) |
| (none) | `runbook_id` | New (from `meta.name`) |

### Trace Replay Compatibility

The v2 `gert replay` command can read v1 traces in **compatibility mode**:

1. Detect v1 format by presence of `type` field (v2 uses `kind`).
2. Synthesize missing fields: `event_id` (new UUID), `run_id` (from file path), `runbook_id` (from first event payload).
3. Map `seq` → `sequence`, `type` → `kind`, `data` → `payload`.
4. Replay as normal.

Compatibility mode will be removed in v2.1.

### Trace Archive Migration Tool

```bash
gert trace migrate --to-v2 .runbook/runs/*/trace.jsonl
```

Rewrites all trace files in-place, adding the required v2 envelope fields.

---

## Rollout Strategy

### Recommended Adoption Path

**Phase 1: Read-Only Validation (Week 1–2)**
1. Install gert v2.0 alongside v1 (both can coexist).
2. Run `gert validate --v2` against existing v1 runbooks.
3. Run `gert migrate --to-v2 --dry-run --check` to identify migration scope.
4. Review migration reports; identify runbooks that require manual fixups.

**Phase 2: Schema Migration (Week 3–4)**
1. Migrate runbooks in a staging directory: `gert migrate --to-v2 ./runbooks/`.
2. Review migrated YAML for correctness (especially governance blocks, tool refs).
3. Run `gert validate` against migrated runbooks to confirm v2 compliance.
4. Commit migrated runbooks to version control (with `.v1.bak` backups retained).

**Phase 3: Extension and Tool Migration (Week 5–6)**
1. Update all `.tool.yaml` files: add `transport` and `capabilities` fields.
2. Update all `.provider.yaml` files: namespace custom fields.
3. Update custom extensions: implement v2 handshake, declare capabilities.
4. Test extensions in v2 runtime with governance policies enabled.

**Phase 4: Integration Migration (Week 7–8)**
1. Update integrations (CI/CD pipelines, monitoring scripts) to use v2 CLI flags.
2. Update JSON-RPC clients (if any) to handle new event kinds.
3. Update trace parsers to handle v2 envelope format.
4. Decommission v1 runtime once all integrations are v2-native.

### Parallel Operation

During the migration period, v1 and v2 runtimes can coexist:
- Install v2 as `/usr/local/bin/gert2` or via version manager (`gert@2`).
- Run v1 runbooks with `gert run` (v1 binary).
- Run v2 runbooks with `gert2 run` or `gert --version 2 run`.
- The v2 runtime rejects v0 runbooks; v1 runbooks are accepted in compatibility mode.

### Timeline

| Milestone | Timeline |
|---|---|
| v2.0 release | T+0 |
| v1 compatibility shim supported | T+0 to T+6 months |
| v1 handshake deprecated warning | T+3 months |
| v1 compatibility shim removed | T+6 months (v2.1 release) |
| v1 runtime end-of-life | T+12 months |

---

## v1 Compatibility Shim

The v2 parser and runtime include a **v1 compatibility mode** that accepts `runbook/v1` documents and silently normalizes them to v2 semantics. Enabled by default in v2.0; removed in v2.1.

### Shim Behavior

1. Accepts `apiVersion: runbook/v1` documents.
2. Silently applies the same transformations as `gert migrate --to-v2`.
3. Emits deprecation warnings to stderr:
   ```
   WARN: runbook 'service-health-check' uses deprecated runbook/v1 schema.
         Run 'gert migrate --to-v2 runbook.yaml' to upgrade.
         v1 support will be removed in gert v2.1 (2026-10-18).
   ```
4. If `--strict` flag is passed, treats deprecation warnings as errors (exit 1).

### Shim Removal

After removal in v2.1:
- The v2 parser rejects `runbook/v1` documents with error: "v1 schema not supported. Migrate to v2."
- All v1 runbooks must be migrated using `gert migrate --to-v2`.
- The v1 compatibility shim code is deleted from the codebase.

---

## Open Questions and Risks

### Open Questions

1. **Provider field namespacing:** Should v2 enforce a strict `provider/*` namespace, or allow custom prefixes? (Recommendation: strict namespace, relaxed in v2.1 if needed.)
2. **Migration of tool discovery paths:** v1 searches `tools/` by default. Should v2 preserve this, or require explicit `--tool-dir` flags? (Recommendation: preserve default path for compatibility.)
3. **Trace archive retention:** Should `gert trace migrate` preserve v1 traces as `.v1.jsonl` backups? (Recommendation: yes, with `--no-backup` opt-out.)

### Migration Risks

1. **Runbooks with custom `meta.*` fields:** If a v1 runbook has custom fields under `meta.*`, migration may fail. Mitigation: emit detailed error messages with field paths.
2. **Extensions relying on v1-specific behaviors:** Some extensions may depend on v1 event ordering or field names. Mitigation: provide comprehensive v2 migration guide for extension authors.
3. **Historical trace parsing:** Teams with custom trace analysis scripts will need updates. Mitigation: publish a trace format migration guide with field mappings.

---

## Summary

Breaking changes: runbook/tool/provider schemas, JSONL trace format, extension handshake protocol. All CLI commands and JSON-RPC methods are backward compatible.

**Migration path:**
- **CLI and JSON-RPC contracts are backward-compatible.**
- **Schema migration is automated** via `gert migrate --to-v2`.
- **v1 compatibility shim** provides a 6-month grace period for runbook and extension migration.
- **Phased rollout** allows teams to adopt v2 incrementally over 8 weeks.

The v1 runtime reaches end-of-life 12 months after v2.0 release.
