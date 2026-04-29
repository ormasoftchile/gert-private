# TUI Runner Architecture — Ken's Brainstorm
**Date**: 2026-04-28 (revised 2026-05-xx — confirmed v0 requirements incorporated)  
**Context**: Revisiting gert v1's TUI runbook runner as a single-binary distribution model for gert v2  
**Architect**: Ken  
**Status**: AUTHORITATIVE — source of truth for TUI exe design

---

## Executive Summary

A standalone TUI runbook runner that bundles the gert v2 engine, runbook definitions, tools, and optionally a domain kit into a **single distributable executable**. Target use case: guided diagnostics, prerequisite validation, machine health checks, and interactive step-by-step runbooks where an engineer needs to run a standardized workflow on a target machine with zero installation overhead.

Key insight: This is **not** a general-purpose gert replacement — it's a specialized deployment model for **offline, sequential, diagnostic runbooks** (which may include interactive decision branches) that need to run in constrained environments (air-gapped machines, customer sites, CI nodes).

**v0 is confirmed interactive.** The original assumption that prompts and branching were out of scope was overturned. Interactive mid-run prompts, branching, and sub-runbook composition via `include:` are all **v0 scope**.

---

## 1. Packaging Model

### Binary Embedding Strategy

**Recommended approach**: `go:embed` + virtual filesystem

```go
//go:embed runbooks/*.yaml
//go:embed runbooks/**/*.yaml
//go:embed domain-kit.yaml
//go:embed tools/*
var embeddedFS embed.FS
```

**What gets embedded**:
- **Entry runbook YAML**: The top-level runbook that drives execution
- **Sub-runbooks**: All YAMLs reachable via `include:` clauses in branch steps (arbitrary depth)
- **Domain kit YAML** (optional): If the runbook references domain-specific tool aliases or configurations
- **Tool binaries**: See "Tool Binary Boundary" below

The `go:embed` glob captures all YAML files under `runbooks/` so sub-runbooks referenced via `include:` are automatically present. The engine resolves `include:` paths against the embedded FS at runtime — no filesystem access required.

### Tool Binary Boundary — Three Options

#### Option A: Embed Everything (Self-Contained Binary)
- **How**: Cross-compile tools for target platform(s), embed as `tools/linux-amd64/curl`, `tools/darwin-arm64/jq`, etc.
- **Pros**: 
  - True zero-dependency execution
  - Works in air-gapped environments
  - Predictable tool versions
- **Cons**:
  - Binary size balloons (100MB+ for a few tools)
  - Multi-platform support requires separate builds per OS/arch
  - License complexity (redistributing third-party binaries)
- **Best for**: Highly constrained environments (customer sites, compliance zones)

#### Option B: System Tool Discovery (Lightweight Binary)
- **How**: Binary shells out to `curl`, `jq`, `nc`, `ping` on $PATH
- **Pros**:
  - Small binary size (5-10MB)
  - Leverages existing system tools
- **Cons**:
  - Requires tools pre-installed on target machine (defeats "zero dependency" goal)
  - Version inconsistencies across machines
  - Fails unpredictably if tools missing
- **Best for**: Internal environments where tool availability is guaranteed (CI runners, standard corp images)

#### Option C: Hybrid — Inline Tools Only (Recommended for v0)
- **How**: 
  - Embed **Go-native inline tools** that gert v2 already has (HTTP checks, JSON parsing, file operations)
  - Explicitly **disallow** out-of-process tool delegation in bundled mode
  - Runbooks designed for this mode must use inline tools exclusively
- **Pros**:
  - Moderate binary size (20-30MB including TUI + engine)
  - No external dependencies
  - No licensing issues
  - Forces runbook authors to think about portability
- **Cons**:
  - Limits expressiveness (no arbitrary shell commands, no system-specific tools)
  - May not fit all diagnostic scenarios (e.g., can't run `ethtool` or `smartctl`)
- **Best for**: **v0 — start here**, expand later if needed

**Decision for v0**: **Option C** — inline tools only. Embed the gert v2 engine with its built-in tool registry, but restrict bundled runbooks to inline-only execution. This keeps binaries distributable, predictable, and license-safe.

### Extensions / Out-of-Process Tools in Bundled Mode

**Short answer**: Not for v0.

**Why**: Out-of-process tools (delegated executables) require either:
1. Embedding third-party binaries (licensing + size explosion)
2. Assuming system tools exist (breaks zero-dependency promise)

**Future consideration**: If there's strong demand, add a **"tool manifest"** mode where the binary checks for required system tools at startup and fails fast with clear instructions if missing. This is cleaner than silent failures mid-runbook.

---

## 2. TUI Design

### Library Choice: Bubble Tea + Lipgloss

**Recommendation**: [`charmbracelet/bubbletea`](https://github.com/charmbracelet/bubbletea) + [`charmbracelet/lipgloss`](https://github.com/charmbracelet/lipgloss)

**Why**:
- Industry-standard Go TUI framework (used by GitHub CLI, k9s alternatives, etc.)
- Elm-inspired model-view-update architecture (clean state management)
- Lipgloss provides styled layout primitives (panels, borders, colors)
- Active maintenance, excellent docs
- Bubbletea's input model handles keyboard events cleanly — required for interactive prompts

**Alternatives considered**:
- `tview`: More widget-heavy, less flexible for custom layouts
- `termui`: Abandoned, outdated
- Raw `termbox-go`: Too low-level, reinventing layout logic

### Layout Specification

```
┌─────────────────────────────────┬──────────────────────────────────────────┐
│ Runbook: network-diagnostics    │ Step 2/5: Check DNS Resolution           │
├─────────────────────────────────┼──────────────────────────────────────────┤
│                                 │                                          │
│ ○ 1. Verify internet conn       │ $ nslookup google.com                    │
│ ⏳ 2. Check DNS resolution       │ Server:   192.168.1.1                    │
│ ○ 3. Test HTTPS endpoints        │ Address:  192.168.1.1#53                 │
│ ○ 4. Validate proxy config       │                                          │
│ ○ 5. Generate report             │ Non-authoritative answer:                │
│                                 │ Name:     google.com                     │
│                                 │ Address:  142.250.80.46                  │
│                                 │                                          │
│                                 │ ✅ DNS resolution successful              │
│                                 │                                          │
│                                 │ [auto-scroll ↓]                          │
│                                 │                                          │
├─────────────────────────────────┴──────────────────────────────────────────┤
│ Progress: ██████████░░░░░░░░░░ 40% │ Elapsed: 12s │ [q] quit [r] retry    │
└──────────────────────────────────────────────────────────────────────────────┘
```

When a prompt step is active, the right panel is replaced by the prompt UI (see §3).

When a branch activates, new steps appear dynamically in the left panel beneath the prompt step (see §4).

### Panel Details

#### Left Panel: Step List (30% width)
- **Status icons**:
  - `○` Pending (gray)
  - `⏳` Running (yellow, animated spinner)
  - `✅` Done (green)
  - `❌` Failed (red)
  - `⏭️` Skipped (blue)
  - `?` Awaiting user input (cyan, blinking)
- **Scrollable** if >10 steps
- **Highlight** current step
- Show step number + title (truncate long titles)
- **Dynamic**: Branch steps are inserted at runtime when a branch activates (see §4)

#### Right Panel: Output Stream / Prompt Area (70% width)
- For **execution steps**: Live streaming from current step's tool output
  - **Auto-scroll** to bottom as new lines arrive (with scroll-lock toggle on user scroll-up)
  - **Syntax highlighting** (optional for v0.1): Detect JSON/YAML output and colorize
  - **Scrollable** with ↑/↓ or PgUp/PgDn
  - Clear on each new step (or configurable: cumulative vs. per-step)
- For **prompt steps**: Replaced by interactive prompt UI (see §3)

#### Bottom Bar: Status + Controls
- **Progress bar**: X/N steps complete
- **Elapsed time**: Total runbook runtime
- **Keybindings**:
  - `q`: Quit (with confirmation if mid-run)
  - `r`: Retry current step (if failed)
  - `s`: Skip current step (if failed)
  - `a`: Abort runbook

### Failure Handling

**When a step fails**:
1. Mark step as `❌` in left panel
2. Show error output in right panel (stderr + exit code)
3. **Pause execution** — don't auto-continue
4. Display modal/footer prompt:
   ```
   ❌ Step 3 failed (exit code 1)
   [r] Retry  [s] Skip  [a] Abort
   ```
5. User choice:
   - **Retry**: Re-run the same step (preserves idempotency if step is designed for it)
   - **Skip**: Mark step as `⏭️ Skipped` and continue to next
   - **Abort**: Stop runbook, show final summary

**Retry limit**: After 3 retries of the same step, force choice between Skip or Abort (prevent infinite retry loops).

---

## 3. Interactive Prompt Model

**v0 confirmed**: Interactive mid-run prompts are **in scope**. A prompt step pauses execution and presents the user with a question in the right panel. The left panel marks the step as `?` (awaiting input).

### Prompt Step Types

#### A. Multiple-Choice (Radio List)
```yaml
- id: diagnose-failure
  type: prompt
  question: "What symptom are you observing?"
  choices:
    - id: ssl_fail
      label: "SSL certificate error in browser"
    - id: unreachable
      label: "Site is completely unreachable"
    - id: slow
      label: "Site loads but is very slow"
```

UI appearance (right panel):
```
? What symptom are you observing?

  ● SSL certificate error in browser
  ○ Site is completely unreachable
  ○ Site loads but is very slow

[↑/↓] Navigate  [Enter] Confirm
```

Navigation: ↑/↓ to move selection, Enter to confirm. No typing needed.

#### B. Free-Text Evidence Field
```yaml
- id: record-expiry
  type: prompt
  question: "Open the site in your browser. What is the SSL certificate expiry date?"
  evidence:
    label: "Certificate expiry date (e.g. 2026-09-15)"
    required: true
```

UI appearance (right panel):
```
? Open the site in your browser.
  What is the SSL certificate expiry date?

  Certificate expiry date (e.g. 2026-09-15):
  > [2026-07-01_____________]

[Enter] Submit
```

- `required: true` disables submit until the field is non-empty
- Evidence is captured in the run record and included in the final report
- If a `open_url:` field is present on the step, the TUI invokes `open`/`xdg-open` to launch the URL before showing the prompt

#### C. Combined (Choice + Evidence)
```yaml
- id: cert-check
  type: prompt
  question: "What does the SSL certificate show?"
  choices:
    - id: expired
      label: "Certificate is expired"
    - id: wrong_domain
      label: "Certificate is for wrong domain"
    - id: self_signed
      label: "Self-signed / untrusted CA"
  evidence:
    label: "Certificate expiry date if visible"
    required: false
```

UI appearance:
```
? What does the SSL certificate show?

  ● Certificate is expired
  ○ Certificate is for wrong domain
  ○ Self-signed / untrusted CA

  Certificate expiry date if visible (optional):
  > [_____________________]

[↑/↓] Navigate  [Tab] Switch focus  [Enter] Confirm
```

Tab switches focus between radio list and text field. Enter submits once a choice is selected (evidence optional unless `required: true`).

#### D. Browser-Confirm Step
```yaml
- id: verify-portal
  type: prompt
  open_url: "https://status.corp.example.com"
  question: "Does the portal show 'All Systems Operational'?"
  choices:
    - id: yes
      label: "Yes"
    - id: no
      label: "No — it shows an incident"
```

The TUI calls `open`/`xdg-open` with the URL before rendering the prompt. The user views the page in their default browser and reports back. Evidence field can be added here too.

### Prompt Step in Run Record

All prompt answers (choice ID + label + evidence text) are captured in the run record:
```json
{
  "step": "diagnose-failure",
  "type": "prompt",
  "answer": {
    "choice": "ssl_fail",
    "choice_label": "SSL certificate error in browser",
    "evidence": "2026-07-01"
  },
  "timestamp": "2026-05-01T10:23:44Z"
}
```

This flows into the final report so the audit trail includes what the user saw and reported.

---

## 4. Branching Model

### How Branching Works

A prompt step's choices activate **different downstream step sequences** — the branch. When the user confirms a choice, the TUI:

1. Looks up the branch for that choice ID in the runbook definition
2. Inserts the branch steps into the left panel **immediately below** the prompt step
3. Executes those steps in sequence
4. On branch completion, returns to the next top-level step (unless the branch ends the runbook)

### YAML Syntax

```yaml
- id: diagnose-failure
  type: prompt
  question: "What symptom are you observing?"
  choices:
    - id: ssl_fail
      label: "SSL certificate error in browser"
    - id: unreachable
      label: "Site is completely unreachable"
  branch:
    ssl_fail:
      steps:
        - id: check-cert-expiry
          run: check-ssl-cert
          args: { host: "{{ target }}" }
        - id: record-expiry
          type: prompt
          question: "What is the expiry date shown?"
          evidence:
            label: "Expiry date"
            required: true
    unreachable:
      include: runbooks/network-diag.yaml
```

### Dynamic Left Panel

Before the prompt is answered:
```
○ 1. Verify connectivity
? 2. Diagnose failure       ← current step (awaiting input)
○ 3. Generate report
```

After user selects "SSL certificate error":
```
○ 1. Verify connectivity
✅ 2. Diagnose failure
  ⏳ 2a. check-cert-expiry  ← branch steps injected
  ○ 2b. record-expiry
○ 3. Generate report
```

Branch steps use sub-numbering (`2a`, `2b`, ...) to show they are children of the prompt step. After the branch completes, execution resumes at step 3.

### Nested Branching

A branch step can itself be a prompt with further branches. There is no depth limit in the schema. The left panel nests sub-numbering (`2a`, `2a-i`, `2a-i-α`, ...) though in practice runbooks should stay shallow (≤3 levels) for readability.

After a nested branch completes, execution returns up the call stack to the parent branch's next step, not necessarily the top-level runbook's next step.

### Branch Termination

A branch can include a terminal step that ends the runbook:
```yaml
branch:
  catastrophic_failure:
    steps:
      - id: collect-logs
        run: collect-logs
      - id: end
        type: terminal
        message: "Critical failure detected. Escalate to on-call. Run ID: {{ run_id }}"
```

`type: terminal` stops execution and shows the final report without attempting remaining top-level steps.

---

## 5. include: Composition Model

### What include: Does

`include:` in a branch block replaces inline steps with the steps from an external runbook YAML file. This is **not** a launch selector menu — there is one entry runbook, and `include:` is called from within branch steps to compose sub-runbooks.

```yaml
branch:
  ssl_fail:
    include: runbooks/ssl-check.yaml
  unreachable:
    include: runbooks/network-diag.yaml
```

At runtime, when the branch activates, the engine:
1. Loads `runbooks/ssl-check.yaml` from the embedded FS
2. Substitutes its steps for the branch path
3. Executes those steps as if they were inline

### Nesting / Recursion

Sub-runbooks can themselves contain `include:` clauses. The engine resolves them recursively. The embedded FS must contain all transitively-referenced runbook files.

Example tree:
```
entry.yaml
  └─ branch: ssl_fail
       └─ include: runbooks/ssl-check.yaml
              └─ branch: chain_issue
                   └─ include: runbooks/chain-validator.yaml
```

All three files must be embedded in the binary.

### include: vs Inline Steps — When to Use Each

| Use inline steps | Use include: |
|---|---|
| Branch logic is specific to this runbook | Branch logic is shared across multiple runbooks |
| Steps are few and simple | Steps form a coherent independent diagnostic |
| No nesting needed | The sub-flow could stand alone as its own runbook |

### Binary Embeds All Referenced Runbooks

The `go:embed` directive must cover all sub-runbook paths. Recommended convention:

```go
//go:embed runbooks/*.yaml runbooks/**/*.yaml
var embeddedFS embed.FS
```

Build tooling (`gert bundle create`) should statically analyze the entry runbook's `include:` graph and verify all referenced files are present before embedding.

### No Launch Menu

There is **no runbook selector UI at launch**. The binary has one entry point runbook. Sub-runbooks are private implementation details — they are reached only through branch steps. The binary name communicates its purpose to the user.

```bash
./ssl-diagnostic          # runs ssl-diagnostic/entry.yaml
./network-health-check    # runs network-health-check/entry.yaml
```

---

## 6. Runbook Suitability

### ✅ Appropriate Use Cases

1. **Sequential diagnostics**:
   - Network connectivity checks (ping, DNS, HTTP, TLS handshake)
   - Disk health (space, SMART status, mount points)
   - Process checks (is service X running? listening on port Y?)
   - **Why**: Fixed sequence, predictable output, no external state

2. **Prerequisite validators**:
   - Pre-install checks (OS version, disk space, CPU, RAM)
   - Dependency verification (is Docker installed? Python 3.9+? etc.)
   - Config validation (are required env vars set? files readable?)
   - **Why**: Gate-keeping step before expensive operations

3. **Machine health certifications**:
   - "Is this CI runner ready for builds?"
   - "Can this server join the k8s cluster?"
   - Generate a pass/fail report with evidence
   - **Why**: Auditable, repeatable, standardized checks

4. **Simple remediation scripts** (with caution):
   - Restart a service if it's not running
   - Clear a temp directory if disk is >90% full
   - Apply a config file from embedded template
   - **Why**: Low-risk, idempotent fixes
   - ⚠️ **Caution**: Avoid destructive operations (no `rm -rf /data`, no schema migrations). Prefer "report the problem" over "fix it automatically" unless fix is trivial + safe.

5. **Interactive diagnostic runbooks** ✅ (confirmed v0):
   - "What symptom are you seeing?" → branches into SSL check, network check, or slow-response path
   - Browser-assisted checks ("Open this URL and tell me what you see")
   - Evidence collection (record certificate dates, error codes, version strings observed on screen)
   - **Why**: Users often know more than the machine can detect automatically; prompts capture that knowledge

6. **Composed multi-path diagnostics** ✅ (confirmed v0):
   - A single binary that routes to different sub-runbooks based on user input
   - Shared sub-runbook logic reused across multiple entry-point runbooks
   - **Why**: include: composition allows modular, maintainable runbook libraries without a menu

### ❌ NOT Appropriate

1. **Long-running background processes**:
   - Servers, daemons, watch loops
   - **Why**: TUI blocks on foreground execution; background tasks require different orchestration

2. **Runbooks requiring GUI/browser interaction as automation**:
   - OAuth flows driven by the binary, web dashboards, Electron apps
   - **Note**: A step can *open* a URL for the user to inspect manually — but the binary does not automate browser interaction

3. **Runbooks requiring secret input** (passwords, API keys) mid-run:
   - **Why**: Use CLI args (`--token`) or env vars for secrets; don't prompt for them interactively (display security)
   - Evidence fields are for observable facts, not credentials

4. **Runbooks with evolving domain state**:
   - "Deploy service X, wait for it to be healthy, then deploy Y"
   - Stateful workflows that depend on previous run history
   - **Why**: Bundled binary has no persistent state; domain kits in this model are static snapshots

5. **Collaborative runbooks**:
   - "User A approves, then User B executes"
   - Workflows requiring distributed coordination
   - **Why**: Single-user, single-machine execution model

---

## 7. Distribution Model

### One Binary, One Entry Runbook

A bundle has **one entry runbook** that drives the entire flow. Sub-runbooks are embedded implementation details reached via `include:` in branch steps. The binary name communicates its purpose.

```
ssl-diagnostic           (binary: gert-engine + TUI + ssl-diagnostic/entry.yaml
                                + runbooks/ssl-check.yaml
                                + runbooks/chain-validator.yaml
                                + company-infra.yaml domain kit)
prereq-check             (binary: gert-engine + TUI + prereq-check.yaml)
cluster-join-check       (binary: gert-engine + TUI + cluster-join/entry.yaml
                                + runbooks/dns-check.yaml
                                + runbooks/registry-check.yaml)
```

**Build process**:
```bash
gert bundle create ssl-diagnostic/entry.yaml \
  --kit company-infra.yaml \
  --output ssl-diagnostic \
  --platform linux-amd64
```

The bundler statically resolves the full `include:` graph and embeds all referenced runbook files.

**Pros**:
- Self-documenting (binary name = runbook purpose)
- Single file to transfer (email, Slack, USB)
- No confusion about "which runbook to run" (there is no choice at launch)
- One binary can cover an entire diagnostic tree via include: composition

**Cons**:
- Duplication of gert engine across binaries (if org has 50 runbooks, 50 copies of engine)
- Update overhead (new engine version = rebuild all binaries)

### Distribution Channels

**Where users get the binary**:
1. **Internal artifact registry** (Artifactory, Nexus, S3 bucket):
   - Automated builds on runbook updates
   - Versioned releases (ssl-diagnostic-v1.2.3)
   - Download via authenticated HTTP/S3 CLI
2. **Email/Slack**:
   - Attach binary to support ticket
   - "Hey, run this on the failing machine and send us the output"
3. **USB/physical media**:
   - Air-gapped environments, field support
4. **Shared network drive**:
   - Corporate file shares (if target machines have network)

**Checksums + signing**: Always provide SHA256 checksum (or sign binaries if org has code signing infrastructure). Prevents tampering, verifies download integrity.

### Version Pinning

**Problem**: Embedded runbook references gert v2 engine features. If engine evolves (new tools, changed semantics), old runbooks might break.

**Solution**: Embed **version metadata** in binary:
```go
type EmbeddedManifest struct {
    EngineVersion   string // "2.3.1"
    EntryRunbook    string // "ssl-diagnostic/entry.yaml"
    RunbookVersion  string // "ssl-diagnostic-v4"
    BuildDate       string // "2026-04-28T14:32:00Z"
    EmbeddedFiles   []string // all embedded runbook paths
    Checksum        string // SHA256 of all embedded YAMLs
}
```

**Display at startup**:
```
┌─────────────────────────────────────────────────┐
│ SSL Diagnostic Runner                          │
│ Engine: gert v2.3.1                             │
│ Runbook: ssl-diagnostic-v4 (2026-04-28)        │
│ Press [Enter] to start...                       │
└─────────────────────────────────────────────────┘
```

**Update policy**: When gert v2 engine makes breaking changes, increment runbook version and rebuild binaries. Old binaries continue to work with their pinned engine version (Go static linking = no runtime dependency drift).

---

## 8. Domain Kit Integration

### Can a Domain Kit Be Embedded?

**Yes**, and it unlocks significant value for **org-specific diagnostic runbooks**.

### Use Case Example: Company Infrastructure Kit

**Scenario**: Your org has standardized infra (Kubernetes, Datadog, internal DNS). You want a "cluster join readiness" runbook that checks:
- Can reach internal DNS (`.corp.company.com`)
- Can authenticate to internal Docker registry (`registry.company.com`)
- Has required labels/taints for k8s node pool
- Datadog agent installed and configured

**Domain kit** (`company-infra.yaml`):
```yaml
domain: company-infra
version: 2.1.0

tools:
  check-dns:
    inline: dns-lookup
    config:
      nameservers: ["10.0.0.53", "10.0.0.54"]
  
  check-registry:
    inline: http-get
    config:
      base-url: https://registry.company.com
      headers:
        Authorization: "Bearer ${REGISTRY_TOKEN}"

  check-labels:
    inline: file-grep
    # ... etc
```

**Bundled binary**: `cluster-join-check`
- Embeds: `cluster-join/entry.yaml` (runbook) + sub-runbooks + `company-infra.yaml` (domain kit)
- Runbook references `check-dns`, `check-registry` tools from domain kit
- Engineer runs on new node: `./cluster-join-check`
- No need to install gert, no need to fetch kit from network

### What to Embed

- **Do embed**:
  - Tool aliases (shortcuts to common inline tools)
  - Default configurations (URLs, timeouts, retry policies)
  - Environment-specific constants (endpoint addresses, resource IDs)
- **Don't embed**:
  - Secrets (API keys, passwords) — these must be passed at runtime via env vars or CLI args
  - User-specific state (history, preferences)
  - Large data blobs (keep embedded YAML <1MB)

---

## 9. V0 Scope Table

| Feature | v0 | Notes |
|---|---|---|
| 3-panel TUI layout | ✅ | Steps / Output / Status bar |
| Sequential step execution | ✅ | |
| Live output streaming | ✅ | |
| Failure handling (retry/skip/abort) | ✅ | |
| go:embed packaging | ✅ | |
| Inline tools only | ✅ | No out-of-process delegation |
| Domain kit embedding | ✅ | Optional |
| Version metadata in binary | ✅ | |
| Final report (txt + log) | ✅ | |
| **Interactive prompt steps** | ✅ | Radio, free-text, combined, browser-confirm |
| **Branching on user answers** | ✅ | Dynamic left panel update |
| **include: sub-runbook composition** | ✅ | Arbitrary nesting |
| **Multiple runbooks in one binary** | ✅ | Via include: — no launch menu |
| Out-of-process tool delegation | ❌ | v0.1+ |
| Parallel step execution | ❌ | v0.1+ |
| Remote reporting / telemetry | ❌ | v0.1+ |
| Networked kit catalog | ❌ | v0.1+ |
| Step-level auto-retry with backoff | ❌ | v0.1+ |
| Secret/password input | ❌ | Never in TUI — use CLI args/env vars |
| Runbook selector menu at launch | ❌ | By design — one entry runbook per binary |
| Runbook editing in TUI | ❌ | Never — edit in IDE, rebuild |

---

## 10. Implementation Sketch (v0 Scope)

### What v0 Delivers

1. **Bundler CLI**:
   ```bash
   gert bundle create <entry-runbook.yaml> \
     [--kit <domain-kit.yaml>] \
     --output <binary-name> \
     --platform linux-amd64
   ```
   - Resolves full `include:` graph, verifies all referenced files exist
   - Embeds entry runbook + all sub-runbooks + optional kit + gert engine + TUI
   - Produces single self-contained executable

2. **TUI Runtime**:
   - 3-panel layout (steps, output, status bar)
   - Step status tracking (pending → running → done/failed/awaiting-input)
   - Live output streaming from current step
   - Failure handling (retry/skip/abort)
   - **Interactive prompt handling** (radio, free-text, combined, browser-confirm)
   - **Dynamic branch activation** (left panel updates when branch chosen)
   - **include: resolution** at runtime against embedded FS
   - Final summary report (copy-pasteable or saved to file, includes prompt evidence)

3. **Constraints**:
   - **Inline tools only** (no out-of-process delegation)
   - **One entry runbook per binary** (no launch menu)
   - **Sequential execution** (no parallelism)
   - **No secret prompts** (evidence fields for observable facts only)

### Build Artifacts

- **Bundler** integrated into main gert CLI: `gert bundle` subcommand
- **TUI package**: `github.com/company/gert/tui` (reusable for other runners)
- **Example runbooks**: `examples/bundles/ssl-diagnostic/entry.yaml`, `prereq-check.yaml`
- **Example sub-runbooks**: `examples/bundles/runbooks/ssl-check.yaml`, `network-diag.yaml`
- **Example domain kit**: `examples/bundles/company-infra.yaml`

### Success Criteria

- [ ] Engineer receives `ssl-diagnostic` binary via Slack
- [ ] Runs on target machine (Linux/macOS, no dependencies)
- [ ] Startup screen shows runbook name + engine version, press Enter to begin
- [ ] Sees TUI with initial diagnostic steps running
- [ ] Step reaches a prompt: "What symptom are you seeing?" → user selects "SSL certificate error"
- [ ] Left panel dynamically shows branch steps below the prompt step
- [ ] Branch includes a browser-confirm step: TUI opens URL, user confirms what they see
- [ ] Evidence field records certificate expiry date, appears in final report
- [ ] Branch calls `include: runbooks/ssl-check.yaml` for deeper cert chain analysis
- [ ] All steps complete, final report shows summary with timestamps and all captured evidence
- [ ] Report saved to `ssl-diagnostic-report-20260428-143022.txt`
- [ ] Total time: <5 minutes from download to report

---

## Open Questions for Team Review

1. **Binary size tolerance**: Is 20-30MB per bundled binary acceptable? (For reference: k9s is ~50MB, lazygit is ~15MB)
   - If "no", we need to revisit Option B (system tool discovery) or Option 2 (separate engine + runbook archive)

2. **Cross-platform builds**: Do we auto-build for linux-amd64, darwin-amd64, darwin-arm64, windows-amd64? Or one platform at a time based on demand?
   - Affects CI/build matrix complexity

3. **Domain kit versioning**: If embedded kit is v2.1.0, but org updates to v2.2.0, what happens?
   - Option A: Binary continues to use v2.1.0 (pinned, predictable)
   - Option B: Binary checks for updates at runtime (requires network, defeats offline model)
   - Recommendation: **Option A** for v0, document update process

4. **Failure semantics**: Should "skip" mark the runbook as "partial success" or "degraded"?
   - Affects final report pass/fail status
   - Recommendation: Three outcomes: SUCCESS (all steps passed), DEGRADED (some skipped), FAILURE (aborted or critical step failed)

5. **Output capture**: Should TUI save full step output to report file, or just summary + errors?
   - Full output = verbose but complete audit trail
   - Summary = concise but loses diagnostic detail
   - Recommendation: **Both** — save full output to `.log` file, human-readable summary to `.txt` report

6. **include: cycle detection**: Two sub-runbooks that include each other would deadlock. The bundler must detect cycles in the include graph and reject them at build time.

7. **Branch variable scoping**: Can a branch step reference `{{ evidence }}` from a parent prompt? Define scoping rules now before implementation.

---

## Next Steps

1. **Prototype**: Build minimal TUI in bubbletea with hardcoded 3-step runbook including one prompt step (no gert engine integration yet)
2. **Prompt widget**: Implement radio + free-text prompt widget in bubbletea; confirm keyboard model works cleanly
3. **Bundler design**: Spec out `gert bundle create` command API, embedded manifest format, include: graph resolver
4. **Engine integration**: Wire bubbletea TUI to gert v2 execution engine (observer pattern for step events); define prompt step event contract
5. **Example runbook**: Write `ssl-diagnostic/entry.yaml` using inline tools + prompt steps + include: branches
6. **Team review**: Walk through this doc in sync meeting, get buy-in on scope + constraints

---

**Author**: Ken (Software Architect)  
**Version**: 2.0 (Revised — interactive prompts, branching, include: composition confirmed v0)  
**Status**: AUTHORITATIVE — source of truth for TUI exe design
