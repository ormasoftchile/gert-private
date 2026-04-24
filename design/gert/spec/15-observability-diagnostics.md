# Observability and Diagnostics

Gert v2 exposes operational signals via three pillars: metrics (Prometheus), distributed tracing (OpenTelemetry), and structured logging (JSON). This is distinct from §12 (Evidence, Tracing, and Resumption): §12 specifies **immutable audit records for compliance and resumption**; this chapter specifies **operational signals** to answer: *is the system healthy?*, *what is slow?*, *where did this run fail?*, *what is the resource utilization?* All instrumentation uses open standards (OpenTelemetry, W3C Trace Context, RFC 3339) for interoperability.

---

## Observability Model

### Metrics

Aggregated performance indicators. Counter, Gauge, and Histogram types. Exposition format: Prometheus OpenMetrics via `GET /metrics` when running `gert serve --http`.

### Distributed Tracing

Hierarchical execution tree with timing and outcome for every run, step, and tool invocation. OpenTelemetry SDK with OTLP exporter. Backends: Jaeger, Zipkin, Datadog, New Relic, AWS X-Ray (all OTLP-compatible).

**Span hierarchy:**

```
gert.run (run_id=...)
├─ gert.step (step_id=backup-db, step_type=cli)
│  ├─ gert.tool.invoke (tool=ssh, exit_code=0)
│  └─ gert.governance.check (policy=allow-ssh)
├─ gert.step (step_id=deploy, step_type=tool)
│  └─ gert.tool.invoke (tool=kubectl, exit_code=0)
└─ gert.step (step_id=verify, step_type=manual)
   └─ gert.input.resolve (provider=approval-gate)
```

### Structured Logging

Machine-parseable JSON log lines with run and step context. Emitted to `stderr` by default. Aggregation: Loki, Elasticsearch, Splunk, CloudWatch (any JSON-capable log platform).

---

## OpenTelemetry Integration

### Span Hierarchy and Attributes

#### Root Span: `gert.run`

**Span Name:** `gert.run`

**Required Attributes:**

```text
runbook.id       = "deploy-app-v3"
runbook.version  = "1.2.3"
actor.id         = "alice@example.com"
run.id           = "20260418T143022-a3f9c1"
run.mode         = "real"               // "real" | "replay" | "dry-run"
client.type      = "cli"                // "cli" | "tui" | "web" | "mcp"
```

**Span Status:**
- `OK` if run outcome is `success`
- `ERROR` if run outcome is `failure`, `cancelled`, or `governance-blocked`

#### Child Span: `gert.step`

**Span Name:** `gert.step`

**Required Attributes:**

```text
step.id          = "backup-db"
step.type        = "cli"                // "cli" | "manual" | "tool" | "invoke"
step.index       = 2                    // zero-based position in sequence
run.id           = "20260418T143022-a3f9c1"
step.name        = "Backup production database"
```

**Optional Attributes:**

```text
step.outcome     = "success"            // "success" | "failure" | "skipped"
step.retries     = 2
```

**Span Events:**
- `governance.check.started` — governance evaluation started
- `governance.check.passed` — governance check passed
- `approval.requested` — human approval requested
- `approval.granted` — approval decision logged

**Span Status:**
- `OK` if step outcome is `success` or `skipped`
- `ERROR` if step outcome is `failure`

#### Grandchild Span: `gert.tool.invoke`

**Required Attributes:**

```text
tool.name        = "kubectl"
tool.transport   = "exec"               // "exec" | "ssh" | "http"
tool.exit_code   = 0
run.id           = "20260418T143022-a3f9c1"
step.id          = "deploy"
```

**Optional Attributes:**

```text
tool.args        = ["apply", "-f", "deployment.yaml"]  // redacted args
tool.duration_ms = 1234
```

**Span Status:** `OK` if `exit_code == 0`; `ERROR` if `exit_code != 0` or invocation crashes.

#### Grandchild Span: `gert.input.resolve`

**Required Attributes:**

```text
input.name       = "db_password"
provider.type    = "vault"
provider.address = "https://vault:8200"
run.id           = "20260418T143022-a3f9c1"
```

#### Grandchild Span: `gert.governance.check`

**Required Attributes:**

```text
policy.name      = "allow-ssh"
policy.outcome   = "allow"              // "allow" | "deny"
run.id           = "20260418T143022-a3f9c1"
step.id          = "backup-db"
```

### Span Error Recording

When a span is marked `ERROR`, the runtime MUST record the error:

```go
span.RecordError(err, trace.WithAttributes(
  attribute.String("error.type", "tool_invocation_failed"),
  attribute.String("error.message", err.Error()),
))
```

### Context Propagation

When a tool invokes an HTTP API, the runtime MUST inject W3C Trace Context headers:

```
traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
tracestate: gert=run_id:20260418T143022-a3f9c1
```

`traceparent` format: `version` (00) / `trace-id` (128-bit) / `parent-id` (64-bit span ID) / `trace-flags` (sampling decision).

`run_id` is also propagated as OpenTelemetry baggage:

```go
baggage.Set("run_id", runID)
```

### OTLP Exporter Configuration

**OTLP/gRPC (default):**

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4317
gert run deploy.yaml
```

Or via flag: `gert run deploy.yaml --otel-endpoint=http://jaeger:4317`

**OTLP/HTTP:**

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318
export OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
```

**Console (development):**

```bash
gert run deploy.yaml --otel-console
```

Prints spans to `stderr` in human-readable format.

### Sampling Strategy

Gert v2 uses **parent-based sampling** with configurable default rate:

- If incoming request has a trace context, **inherit parent's sampling decision**.
- If no parent context exists, sample at configured rate (default: 100%).

```bash
export OTEL_TRACES_SAMPLER=parentbased_traceidratio
export OTEL_TRACES_SAMPLER_ARG=0.1  # 10% sampling
```

For high-throughput production systems, consider `0.01` (1%) or `0.001` (0.1%).

---

## Metrics Catalog

All metrics prefixed with `gert_`.

### Run Metrics

| Metric                        | Type      | Labels                        | Description                                          |
|-------------------------------|-----------|-------------------------------|------------------------------------------------------|
| `gert_runs_total`             | Counter   | `outcome`, `runbook_id`       | Total run completions by outcome (`success`, `failure`, `cancelled`) |
| `gert_run_duration_seconds`   | Histogram | `outcome`, `runbook_id`       | End-to-end run duration                              |
| `gert_active_runs`            | Gauge     | —                             | Currently executing runs                             |

Histogram buckets for `gert_run_duration_seconds`: `0.5, 1, 2, 5, 10, 30, 60, 120, 300, 600, 1800, 3600` seconds.

### Step Metrics

| Metric                        | Type      | Labels                        | Description                                          |
|-------------------------------|-----------|-------------------------------|------------------------------------------------------|
| `gert_steps_total`            | Counter   | `outcome`, `step_type`        | Total step completions by outcome and type           |
| `gert_step_duration_seconds`  | Histogram | `step_type`                   | Per-step duration by type (`cli`, `manual`, `tool`, `invoke`) |

Histogram buckets for `gert_step_duration_seconds`: `0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300` seconds.

### Tool Invocation Metrics

| Metric                            | Type      | Labels                        | Description                                          |
|-----------------------------------|-----------|-------------------------------|------------------------------------------------------|
| `gert_tool_invocations_total`     | Counter   | `tool_name`, `outcome`        | Tool invocation count by tool and outcome            |
| `gert_tool_duration_seconds`      | Histogram | `tool_name`                   | Tool invocation latency                              |

Histogram buckets for `gert_tool_duration_seconds`: `0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10, 30` seconds.

### Approval Metrics

| Metric                        | Type      | Labels                        | Description                                          |
|-------------------------------|-----------|-------------------------------|------------------------------------------------------|
| `gert_approval_wait_seconds`  | Histogram | `step_id`                     | Time from approval request to decision               |
| `gert_approvals_total`        | Counter   | `decision`, `step_id`         | Total approval decisions (`approved`, `rejected`)    |

Histogram buckets for `gert_approval_wait_seconds`: `10, 30, 60, 300, 600, 1800, 3600, 7200, 14400, 28800, 86400` seconds.

### Extension Metrics

| Metric                            | Type      | Labels            | Description                                    |
|-----------------------------------|-----------|-------------------|------------------------------------------------|
| `gert_extension_crashes_total`    | Counter   | `extension_id`    | Extension process crashes                      |
| `gert_extension_loaded`           | Gauge     | `extension_id`    | 1 if extension loaded, 0 otherwise             |

### Metrics Endpoint

```
GET http://localhost:8080/metrics
```

```bash
gert serve --http --addr=:8080
```

Example output:

```
# TYPE gert_runs_total counter
gert_runs_total{outcome="success",runbook_id="deploy-app-v3"} 42
gert_runs_total{outcome="failure",runbook_id="deploy-app-v3"} 3

# TYPE gert_run_duration_seconds histogram
gert_run_duration_seconds_bucket{outcome="success",runbook_id="deploy-app-v3",le="1"} 5
gert_run_duration_seconds_bucket{outcome="success",runbook_id="deploy-app-v3",le="5"} 18
gert_run_duration_seconds_bucket{outcome="success",runbook_id="deploy-app-v3",le="+Inf"} 42
gert_run_duration_seconds_sum{outcome="success",runbook_id="deploy-app-v3"} 187.4
gert_run_duration_seconds_count{outcome="success",runbook_id="deploy-app-v3"} 42

# TYPE gert_active_runs gauge
gert_active_runs 3
```

### Push-Based Metrics (OTLP)

For ephemeral CI jobs where scraping is not feasible:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://collector:4317
export OTEL_METRICS_EXPORTER=otlp
gert run deploy.yaml
```

Runtime pushes metrics at configurable intervals (default: 60 seconds).

---

## Structured Logging

### Log Line Format

```json
{
  "timestamp": "2026-04-18T14:30:22.123456Z",
  "level": "info",
  "msg": "step execution started",
  "run_id": "20260418T143022-a3f9c1",
  "step_id": "backup-db",
  "step_type": "cli",
  "actor": "alice@example.com"
}
```

### Required Fields

- **`timestamp`** — RFC 3339 with microsecond precision, UTC timezone
- **`level`** — one of: `debug`, `info`, `warn`, `error`
- **`msg`** — human-readable message

### Contextual Fields

- `run_id` — present in all logs within run context
- `step_id` — present in all logs within step context
- `step_type` — `cli`, `manual`, `tool`, `invoke`
- `actor` — user or service principal
- `tool_name` — tool identifier (tool invocation logs)
- `error` — error message (`error` level logs)
- `exit_code` — process exit code (CLI step completions)

### Log Levels

- **`debug`** — detailed diagnostic (input resolution, policy evaluation details)
- **`info`** — operational milestones (step started, run completed)
- **`warn`** — recoverable issues (retry attempt, fallback to default)
- **`error`** — failures requiring attention (step failure, governance denial)

Default: `info`. Configure via `GERT_LOG_LEVEL=debug` or `--log-level=debug`.

### Sensitive Data Handling

Log lines MUST NOT contain sensitive input values. Apply same redaction rules as the trace file (§12.4).

**Safe:**
```json
{"msg": "resolved input 'db_password' from provider 'vault'"}
```

**Unsafe:**
```json
{"msg": "resolved input 'db_password' = 'hunter2'"}
```

### Log Output Configuration

Default: `stderr`. Additional file output:

```bash
gert run deploy.yaml --log-output=file:/var/log/gert/gert.log
```

Syslog:

```bash
gert run deploy.yaml --log-output=syslog://localhost:514
```

### Log Aggregation and Correlation

Every log line in run context includes `run_id`.

**Loki:**
```
{job="gert"} | json | run_id="20260418T143022-a3f9c1"
```

**Elasticsearch:**
```
GET /gert-logs-*/_search
{
  "query": { "term": {"run_id": "20260418T143022-a3f9c1"} }
}
```

---

## Health and Readiness Endpoints

Available when running `gert serve --http`. Designed for Kubernetes liveness and readiness probes.

### `GET /health` (Liveness)

```json
{
  "status": "ok",
  "version": "v2.0.0",
  "uptime_seconds": 3600
}
```

### `GET /ready` (Readiness)

**200 OK (ready):**

```json
{
  "status": "ready",
  "extensions_loaded": 3,
  "extensions_failed": 0
}
```

**503 Service Unavailable (not ready):**

```json
{
  "status": "not_ready",
  "reason": "extension 'aws-tools' failed to load",
  "extensions_loaded": 2,
  "extensions_failed": 1
}
```

### Kubernetes Probe Configuration

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: gert-server
spec:
  containers:
  - name: gert
    image: gert:v2.0.0
    command: ["gert", "serve", "--http", "--addr=:8080"]
    livenessProbe:
      httpGet:
        path: /health
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 10
    readinessProbe:
      httpGet:
        path: /ready
        port: 8080
      initialDelaySeconds: 10
      periodSeconds: 5
```

---

## Diagnostics Commands

### `gert diagnose`

Pre-flight checks to verify runtime environment.

```bash
gert diagnose [--runbook=<path>]
```

Checks performed:
1. Runbook Schema Validation
2. Tool Availability (all `.tool.yaml` tools findable in PATH or `.runbook/tools/`)
3. Input Provider Connectivity (providers reachable, API keys valid)
4. Extension Loading (all declared extensions load successfully)
5. Governance Policy Syntax (OPA policies validated, if configured)

Exit code: 0 if all pass, 1 if any fail.

```
✓ Runbook schema validation passed
✓ Tool 'kubectl' found at /usr/local/bin/kubectl
✗ Input provider 'vault' unreachable: connection refused
✗ Extension 'k8s-tools' failed to load: missing dependency

Diagnostics: FAIL (2 errors)
```

### `gert trace show`

Pretty-print a completed run's trace.

```bash
gert trace show <run_id>
```

```
Run: 20260418T143022-a3f9c1
Runbook: deploy-app-v3
Actor: alice@example.com
Started: 2026-04-18T14:30:22Z
Completed: 2026-04-18T14:33:15Z
Duration: 2m53s
Outcome: success

Steps:
  [0] backup-db (cli) - 12.3s - success
    └─ tool: ssh - 12.1s - exit_code=0
  [1] deploy (tool) - 45.2s - success
    └─ tool: kubectl - 44.8s - exit_code=0
  [2] verify (manual) - 2m10s - success
    └─ approval: alice@example.com - granted
```

### `gert trace export`

Convert a JSONL trace to OTLP format for import into Jaeger/Zipkin.

```bash
gert trace export <run_id> --format=otlp --output=trace.json
```

Supported formats: `otlp`, `jaeger`, `zipkin`.

### `gert run list`

List recent runs with status.

```bash
gert run list [--limit=N] [--json]
```

JSON output for CI integration:

```json
[
  {
    "run_id": "20260418T143022-a3f9c1",
    "runbook_id": "deploy-app-v3",
    "actor": "alice@example.com",
    "outcome": "success",
    "duration_ms": 173000,
    "started_at": "2026-04-18T14:30:22Z",
    "completed_at": "2026-04-18T14:33:15Z"
  }
]
```

---

## Observability for CI/CD Pipelines

### JSON Output Mode

```bash
gert run deploy.yaml --output=json
```

Emits JSON summary to `stdout` on completion:

```json
{
  "run_id": "20260418T143022-a3f9c1",
  "outcome": "success",
  "step_count": 3,
  "failed_step": null,
  "duration_ms": 173000,
  "started_at": "2026-04-18T14:30:22Z",
  "completed_at": "2026-04-18T14:33:15Z"
}
```

**Exit codes:**
- `0` — `outcome == "success"`
- `1` — `outcome == "failure"`
- `2` — `outcome == "governance-blocked"`
- `3` — `outcome == "cancelled"`

### GitHub Actions Example

```
- name: Run gert runbook
  id: gert_run
  run: |
    OUTPUT=$(gert run deploy.yaml --output=json)
    echo "$OUTPUT" | jq .
    RUN_ID=$(echo "$OUTPUT" | jq -r .run_id)
    OUTCOME=$(echo "$OUTPUT" | jq -r .outcome)
    echo "run_id=$RUN_ID" >> $GITHUB_OUTPUT
    echo "outcome=$OUTCOME" >> $GITHUB_OUTPUT

- name: Upload trace artifact
  if: always()
  uses: actions/upload-artifact@v3
  with:
    name: gert-trace
    path: .runbook/runs/${{ steps.gert_run.outputs.run_id }}/trace.jsonl
```

### GitLab CI Example

```yaml
deploy:
  script:
    - gert run deploy.yaml --output=json | tee run-output.json
    - RUN_ID=$(jq -r .run_id run-output.json)
    - echo "Run ID: $RUN_ID"
  artifacts:
    when: always
    paths:
      - .runbook/runs/
    reports:
      junit: .runbook/runs/*/trace.jsonl
```

---

## Integration Recipes

### Prometheus + Grafana

```bash
gert serve --http --addr=:8080
```

Prometheus `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'gert'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

Reference Grafana dashboard at `.runbook/observability/grafana-dashboard.json`. Visualizes: run throughput, success/failure rate, p50/p95/p99 duration, active runs, step execution time by type, tool invocation latency.

### Jaeger (Distributed Tracing)

```bash
docker run -d --name jaeger -p 16686:16686 -p 4317:4317 \
  jaegertracing/all-in-one:latest

export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
gert run deploy.yaml
```

View traces at `http://localhost:16686` (service: `gert`).

### Loki + Grafana (Log Aggregation)

Promtail config:

```yaml
scrape_configs:
  - job_name: gert
    static_configs:
      - targets: [localhost]
        labels:
          job: gert
          __path__: /var/log/gert/*.log
    pipeline_stages:
      - json:
          expressions:
            timestamp: timestamp
            level: level
            msg: msg
            run_id: run_id
      - labels:
          level:
          run_id:
```

```bash
gert run deploy.yaml --log-output=file:/var/log/gert/gert.log
```

Grafana query: `{job="gert"} | json | run_id="20260418T143022-a3f9c1"`

### Datadog / New Relic (OTLP SaaS)

**Datadog:**

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=https://api.datadoghq.com
export DD_API_KEY=<your-api-key>
gert run deploy.yaml
```

**New Relic:**

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=https://otlp.nr-data.net:4317
export NEW_RELIC_API_KEY=<your-api-key>
gert run deploy.yaml
```
