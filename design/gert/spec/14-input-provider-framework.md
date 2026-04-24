# Input Provider Framework

The input provider framework governs how gert resolves dynamic input values at run time. When a runbook declares an input with a `from:` binding, the provider framework locates the appropriate provider, dispatches a resolution request (batched by provider prefix), and returns the concrete value before the first step that needs it begins. Providers share the same transport layer as tools (`stdio`, `stdio-jsonrpc`, `mcp`). Every resolved value is recorded in the run trace.

---

## Provider Definition Schema

Each provider is declared in a `.provider.yaml` file. By convention, project-level providers live in `providers/<name>.provider.yaml`.

```yaml
# providers/pagerduty.provider.yaml
apiVersion: provider/v2

meta:
  name: pd                         # Prefix used in from: bindings (e.g. pd.*)
  version: "1.0.0"
  description: "Resolves inputs from PagerDuty incidents and services"
  binary: pd-provider              # Executable name (looked up in PATH)

transport:
  mode: stdio-jsonrpc              # stdio | stdio-jsonrpc | mcp
  startup:
    argv: ["--config", "pd.yaml"]
    ready-pattern: "provider ready"
    timeout: "10s"
    shutdown-method: "shutdown"

resolution:
  method: provider/resolve         # JSON-RPC method the host calls
  prefixes:
    - "pd."                        # All from: values starting with pd. dispatched here

  input-schema:
    type: object
    properties:
      bindings:
        type: object
        description: "Map of input name to from: binding string"
      context:
        type: object
        description: "Execution context (runId, userId, mode)"
    required: ["bindings"]

  output-schema:
    type: object
    properties:
      resolved:
        type: object
        description: "Map of input name to resolved string value"
      warnings:
        type: array
        items:
          type: string
    required: ["resolved"]

  cache:
    enabled: true
    scope: run                     # run | step | none
    ttl: "60s"

governance:
  requires-capabilities:
    - capability/network
  env-read-patterns:
    - "PD_API_KEY"
    - "PD_*"
```

---

## Resolution Protocol

### Resolution Flow

When the run engine encounters `from: pd.incident.title`:

1. **Prefix match** — framework scans registered providers for the one whose declared prefix matches the start of the binding string. Longest matching prefix wins.
2. **Provider start** — if the matched provider's process is not running, the host spawns it and waits for the ready signal (same mechanics as `stdio-jsonrpc` tools).
3. **Batch collection** — all inputs whose binding prefix maps to the same provider are grouped into a single resolution request.
4. **JSON-RPC call** — host sends a `provider/resolve` request.
5. **Result merge** — resolved values merged into the run's variable map.
6. **Fallback** — any binding not resolved (absent from `resolved` map, or provider returned a warning) falls back according to the input's `fallback:` field (default: interactive prompt).

[DIAGRAM: Resolution pipeline — `from:` binding → Prefix Match → Start Provider → Batch Collect → JSON-RPC Resolve → Merge Vars → Value Available; fallback paths from Prefix Match (no match) and JSON-RPC (unresolved) to Value Available via Interactive Prompt / fallback field value.]

### Resolution Request Envelope

```json
{
  "jsonrpc": "2.0",
  "id": "prov-1",
  "method": "provider/resolve",
  "params": {
    "bindings": {
      "incident_title":    "pd.incident.title",
      "incident_severity": "pd.incident.severity",
      "service_name":      "pd.service.name"
    },
    "context": {
      "runId":   "run-abc123",
      "stepId":  "__pre-flight__",
      "userId":  "ops@acme.example",
      "mode":    "real",
      "traceId": "trace-xyz"
    }
  }
}
```

### Resolution Response Envelope

```json
{
  "jsonrpc": "2.0",
  "id": "prov-1",
  "result": {
    "resolved": {
      "incident_title":    "Database connection pool exhausted",
      "incident_severity": "P1",
      "service_name":      "payments-api"
    },
    "warnings": [],
    "telemetry": {
      "startedAt":  "2026-04-18T10:00:00Z",
      "finishedAt": "2026-04-18T10:00:00.080Z",
      "durationMs": 80
    }
  }
}
```

### Resolution Error Handling

If the provider returns a JSON-RPC error, or a binding key is absent from `resolved`, the host applies the input's fallback strategy:

```yaml
inputs:
  incident_title:
    from: pd.incident.title
    description: "Active incident title"
    fallback: prompt              # prompt | fail | default
    default: "Unknown incident"   # Used when fallback: default
```

- **`fallback: prompt`** (default) — host issues an interactive prompt. In non-interactive modes (dry-run, CI) this causes the run to fail.
- **`fallback: default`** — declared `default:` value is used without prompting; a warning is written to the trace.
- **`fallback: fail`** — run aborts immediately with a structured error.

### Resolution Caching

Providers declare cache policy in their definition; the host respects it:

- **`scope: run`** — resolved value reused for all steps in the same run referencing the same binding (default for external providers).
- **`scope: step`** — value resolved fresh on each step.
- **`scope: none`** — caching disabled; every binding triggers a new RPC call.

Cache entries keyed by `(runId, bindingString)`. Cached values stored in-memory only; **not** persisted across run resumptions.

---

## Built-in Providers

Always available without a `.provider.yaml` file.

### `env`

Reads values from environment variables. Fails if the variable is not set; fallback strategy applied.

```yaml
inputs:
  api_key:
    from: env.ACME_API_KEY
```

### `file`

Reads values from local files. An optional JSON Pointer fragment (`#/path/to/field`) extracts a nested value from JSON/YAML. Without a fragment, the entire file contents are returned as a string.

```yaml
inputs:
  ssh_key:
    from: "file.~/.ssh/id_rsa"
  config_value:
    from: "file./etc/gert/config.json#/database/host"
```

### `prompt`

Default resolution strategy and fallback for any failed provider resolution (unless overridden). In `real`/`debug` modes, presents a terminal prompt. In `dry-run`/`replay` modes, reads from the scenario's `inputs.yaml`. In non-interactive environments without a scenario, fails with an informative error directing the operator to supply the value via `--var` or an explicit provider.

```yaml
inputs:
  server_name:
    from: prompt
    description: "Server to check"
```

### `workspace`

Reads values from `.gert/config.yaml` using dot-notation keys.

```yaml
inputs:
  default_region:
    from: workspace.defaults.region
```

---

## Interactive Step Type Contracts

Input providers also power three interactive step types: `choice`, `decision`, and `collector`.

### Choice Step Contract

Presents a fixed set of labeled options; stores the selected value as a variable.

**Request (`provider/choice`):**

```json
{
  "jsonrpc": "2.0",
  "id": "choice-1",
  "method": "provider/choice",
  "params": {
    "stepId": "select-region",
    "prompt": "Select the target AWS region",
    "options": [
      { "value": "us-east-1", "label": "US East (N. Virginia)" },
      { "value": "us-west-2", "label": "US West (Oregon)" },
      { "value": "eu-west-1", "label": "Europe (Ireland)" }
    ],
    "default": "us-east-1",
    "context": { "runId": "run-abc123", "stepId": "select-region",
                 "userId": "ops@acme.example", "mode": "real" }
  }
}
```

**Response:**

```json
{
  "jsonrpc": "2.0",
  "id": "choice-1",
  "result": {
    "selected": "us-west-2"
  }
}
```

**Contract requirements:**

- `options` array MUST contain at least two elements.
- Each option MUST have both `value` (stored in variable) and `label` (displayed to operator).
- `default` (optional) MUST match one of the `value` fields if present.
- Provider's `selected` MUST match one of the option values exactly; otherwise run engine fails step with `choice.invalid_selection`.
- Providers MAY render options in declaration order or sorted alphabetically by label.
- `multiple: true` (optional, default `false`) — when set, provider MUST allow selecting one or more values; `selected` MUST be a JSON array of strings.
- `options_from` (optional) — mutually exclusive with static `options`; when present, run engine calls `inputProvider/getOptions` before dispatching the request (see Dynamic Options Protocol).

**Built-in `prompt` provider:** presents a numbered list; waits for operator to enter number or value. In `dry-run` mode reads from scenario `inputs.yaml` or uses `default`.

---

### Decision (Router) Step Contract

Presents routing choices; selected route controls execution flow (not stored as a variable).

**Request (`provider/decision`):**

```json
{
  "jsonrpc": "2.0",
  "id": "decision-1",
  "method": "provider/decision",
  "params": {
    "stepId": "triage-severity",
    "prompt": "Select the incident severity level",
    "routes": [
      { "route": "p1-critical", "label": "P1: Critical (immediate)" },
      { "route": "p2-high",     "label": "P2: High (1 hour SLA)" },
      { "route": "p3-normal",   "label": "P3: Normal (24 hour SLA)" }
    ],
    "context": { "runId": "run-abc123", "stepId": "triage-severity",
                 "userId": "ops@acme.example", "mode": "real" }
  }
}
```

**Response:**

```json
{
  "jsonrpc": "2.0",
  "id": "decision-1",
  "result": {
    "route": "p2-high"
  }
}
```

**Contract requirements:**

- `routes` MUST contain at least two elements with both `route` (target node label/runbook ID) and `label`.
- Provider does **not** need to know the runbook graph structure; it only returns the selected route label.
- If provider returns a route label not in `routes`, run engine fails step with `decision.invalid_route`.
- Decision result is **not** stored as a variable; it only controls execution flow.

**Built-in `prompt` provider:** displays routes as a numbered list. In `dry-run` mode reads from scenario file or fails if no scenario is provided.

---

### Collector Step Contract

Gathers unstructured input: free-form text, multi-line notes, file attachments, screenshots, URLs, or other evidence artifacts.

**Request (`provider/collect`):**

```json
{
  "jsonrpc": "2.0",
  "id": "collect-1",
  "method": "provider/collect",
  "params": {
    "stepId": "gather-evidence",
    "instructions": "Upload a screenshot of the error and describe the issue",
    "fields": [
      {
        "name": "screenshot",
        "type": "file",
        "label": "Error screenshot",
        "required": true,
        "visible": true,
        "accept": ["image/png", "image/jpeg"],
        "maxSizeBytes": 10485760
      },
      {
        "name": "description",
        "type": "text",
        "label": "Describe what you observed",
        "required": true,
        "visible": true,
        "multiline": true,
        "validation": {
          "minLength": 10,
          "maxLength": 2000,
          "pattern": "^[\\s\\S]+$",
          "patternHint": "Description must not be empty"
        }
      },
      {
        "name": "log_url",
        "type": "url",
        "label": "Link to related logs (optional)",
        "required": false,
        "visible": true
      }
    ],
    "context": { "runId": "run-abc123", "stepId": "gather-evidence",
                 "userId": "ops@acme.example", "mode": "real" }
  }
}
```

**Response:**

```json
{
  "jsonrpc": "2.0",
  "id": "collect-1",
  "result": {
    "values": {
      "description": "Database connection pool exhausted...",
      "log_url": "https://logs.acme.example/trace/xyz"
    },
    "artifacts": [
      {
        "field": "screenshot",
        "filename": "error-2026-04-18.png",
        "contentType": "image/png",
        "sizeBytes": 245631,
        "sha256": "a3c7d1f...",
        "storagePath": "artifacts/run-abc123/screenshot.png"
      }
    ]
  }
}
```

**Field types:** `text`, `number`, `integer`, `date`, `datetime`, `boolean`, `select`, `file`, `image`, `url`.

**Field contract requirements:**

- **`name`** — variable name where the value is stored.
- **`type`** — one of the types listed above.
- **`label`** — human-readable prompt text.
- **`required`** — boolean. Providers MUST return all required fields in `values`; missing required field → `collector.missing_required_field` error.
- **`visible`** — pre-evaluated by the executor from the field's `when` expression before the request is sent. CLI providers SHOULD skip `visible: false` fields; rich UI providers SHOULD show/hide dynamically.
- **`validation`** (optional) — client-side constraints that providers MAY enforce; executor always re-validates server-side:
  - `pattern` — RE2 regex for full match (`text` fields only)
  - `patternHint` — human-readable message when pattern fails
  - `minLength` / `maxLength` — character bounds for `text`
  - `min` / `max` / `step` — numeric bounds for `number`/`integer`
  - `min` / `max` — date bounds for `date`/`datetime` (ISO 8601)
  - `multiple: true|false` — enables multi-select for `select` fields
  - `min_selections` / `max_selections` — array length bounds for multi-select
  - `accept` (MIME types) and `maxSizeBytes` — for `file`/`image` fields
- For `type: file` fields, providers MUST upload to artifact store and return metadata (`field`, `filename`, `contentType`, `sizeBytes`, `sha256`, `storagePath`).
- File size limits enforced by provider. File exceeding `maxSizeBytes` → `collector.file_too_large` error before storing.
- Supported MIME types: images (`image/png`, `image/jpeg`, `image/gif`), documents (`application/pdf`, `text/plain`, `text/markdown`), archives (`application/zip`, `application/x-tar`, `application/gzip`), logs (`text/plain`, `application/json`).
- Providers MAY support additional MIME types but MUST respect the `accept` constraint.

---

## Dynamic Options Protocol

When a `select` field in `collector` or a `choice` step declares `options_from.provider`, the run engine fetches options dynamically before rendering.

**Fetch sequence:**

1. **Provider lookup** — resolve provider ID from `options_from.provider`; if not registered → `options_from.unknown_provider`.
2. **Provider start** — spawn provider if not running (normal `stdio-jsonrpc` startup).
3. **Options request** — send `inputProvider/getOptions` to provider.
4. **Option list received** — treat returned `options` as if declared statically; validation and rendering proceed identically.

Failure to fetch options fails the step immediately with `options_from.fetch_failed`. **No silent fallback to empty options.**

### `inputProvider/getOptions` Request

```json
{
  "jsonrpc": "2.0",
  "id": "getopts-1",
  "method": "inputProvider/getOptions",
  "params": {
    "providerId": "employee-directory",
    "field":      "managers",
    "stepId":     "assign-manager",
    "variables": {
      "department": "engineering",
      "region":     "us-west"
    },
    "context": { "runId": "run-abc123", "stepId": "assign-manager",
                 "userId": "ops@acme.example", "mode": "real" }
  }
}
```

**Request fields:** `providerId`, `field` (allows one provider to serve multiple option lists), `stepId`, `variables` (current run variable scope for contextual filtering), `context`.

### `inputProvider/getOptions` Response

```json
{
  "jsonrpc": "2.0",
  "id": "getopts-1",
  "result": {
    "options": [
      { "value": "alice",   "label": "Alice Chen (Staff Eng)" },
      { "value": "bob",     "label": "Bob Okafor (Principal)" },
      { "value": "charlie", "label": "Charlie Park (Director)",
                             "hint": "Manages infrastructure" }
    ],
    "cacheTtlSeconds": 300
  }
}
```

**`cacheTtlSeconds`** (optional) — if non-zero, run engine caches the option list for the remainder of the run (capped at declared TTL). `0` disables caching.

Providers that implement `inputProvider/getOptions` MUST declare the capability in their handshake. If the provider does not advertise `getOptions: true`, step fails with `options_from.capability_not_supported`.

---

## Autocomplete Search Protocol

The `type: autocomplete` field in `collector` requires live-search support. Run engine issues `inputProvider/search` RPC calls as the operator types; debouncing handled by the UI layer.

### `inputProvider/search` Request

```json
{
  "jsonrpc": "2.0",
  "id": "search-1",
  "method": "inputProvider/search",
  "params": {
    "provider":  "service-registry",
    "field":     "query",
    "query":     "pay",
    "minChars":  2,
    "runId":     "20260418T142300-abc12345",
    "variables": { "region": "us-west" }
  }
}
```

**`query`** — current text typed. Run engine does not issue the RPC until `len(query) >= minChars`.

### `inputProvider/search` Response

```json
{
  "jsonrpc": "2.0",
  "id": "search-1",
  "result": {
    "results": [
      { "value": "payments-api", "label": "Payments API" },
      { "value": "payroll-svc",  "label": "Payroll Service" }
    ],
    "hasMore": false
  }
}
```

**`hasMore: true`** — UI SHOULD display "Showing top N results — keep typing to narrow."

**Timeout contract:** Providers MUST return within **500 ms**. Timeout → empty results list; operator may retry by continuing to type.

**CLI degradation:** Built-in `prompt` provider does not support real-time search. For `autocomplete` fields dispatched to a CLI provider, run engine issues a single search with `query: ""` to retrieve all options, then presents as a standard `select` list.

Providers implementing `inputProvider/search` MUST declare `search: true` in capability advertisement. Providers that do not advertise `search: true` have autocomplete fields automatically degraded to CLI path.

---

## Provider Capability Matrix

| Provider    | Input Resolution | `choice` | `decision` | `collector` | `getOptions` | `search` |
|-------------|:----------------:|:--------:|:----------:|:-----------:|:------------:|:--------:|
| `prompt`    | Yes              | Yes      | Yes        | Yes         | No           | No       |
| `env`       | Yes              | No       | No         | No          | No           | No       |
| `file`      | Yes              | No       | No         | No          | No           | No       |
| `workspace` | Yes              | No       | No         | No          | No           | No       |

Rich UI providers (e.g. VS Code extension's built-in provider) SHOULD declare `search: true` to enable real-time autocomplete.

External providers MAY implement any combination. Providers MUST declare capabilities in the handshake:

```
{
  "jsonrpc": "2.0",
  "method": "provider/initialize",
  "params": {
    "capabilities": {
      "resolution": true,
      "choice": true,
      "decision": false,
      "collector": true,
      "getOptions": true,
      "search": true
    }
  }
}
```

If the run engine encounters a `choice`, `decision`, or `collector` step and the configured provider does not advertise the corresponding capability, the engine falls back to the built-in `prompt` provider.

---

## Provider Composition

A `from:` binding may declare a priority-ordered chain of providers. The host attempts each in order and uses the first successful result.

```yaml
inputs:
  api_token:
    from:
      - env.ACME_API_TOKEN          # Try env var first
      - vault.secret/acme/api-token # Fall back to Vault
      - prompt                      # Final fallback: ask the operator
    description: "API token for Acme service"
```

A provider is considered to have failed if it returns an error, if the binding key is absent from the resolved map, or if the resolved value is an empty string (unless `allow-empty: true` is declared).

---

## Provider Lifecycle

**Process model:**
- `stdio-jsonrpc` and `mcp` transport providers are **persistent** — spawned once at pre-flight resolution phase, kept alive for the entire run (avoids repeated startup latency for providers with session state like Vault token renewal or PagerDuty OAuth).
- `stdio` transport providers are **per-resolution** — new process per resolution request.

**Startup:** Lazy on first use; same `ready-pattern` / timeout mechanism as `stdio-jsonrpc` tools.

**Shutdown:** At run completion (success, failure, or cancellation), host iterates over all running provider processes and sends the method declared in `transport.startup.shutdown-method` (default: `shutdown`). Providers have a 5-second grace period, then SIGTERM, then SIGKILL after a further 5 seconds.

**Crash handling:** If a provider crashes during a resolution call, host logs a `provider.crashed` trace event and applies the input's `fallback:` strategy for all in-flight bindings.
