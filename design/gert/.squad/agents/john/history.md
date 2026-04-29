# John's Work History
**Role**: YAML and Schema Specialist  
**Team**: gert v2 design

---

## 2026-04-28: Domain Kit Model Brainstorm

**Task**: Analyze domain kit embedding model for standalone diagnostic TUI executable  
**Output**: `/Volumes/Projects/gert-domain-home/.squad/tmp/brainstorm-john-kit.md`

### Key Findings

1. **Kit Utility Assessment**
   - Pure diagnostics (ping, DNS, port checks): Kit unnecessary
   - Asset-specific diagnostics: Kit essential (provides inventory, metadata)
   - Prerequisite validators: Kit optionally useful
   - Remediation runbooks: Kit required

2. **Kit Embedding Model**
   - **Recommended**: go:embed for single kit YAML (~100-500KB overhead)
   - Runtime: Parse from memory (no temp file extraction)
   - Override support: `--kit /path/to/custom.yaml` flag
   - Graceful degradation: Engine runs even if kit is nil

3. **Kit-less Runbook Model**
   - Fully self-contained with literal values (no {{ asset.X }} references)
   - Engine changes: Make kit parameter optional, template renderer handles missing context
   - Schema: Add `metadata.requires_kit` field (true/false/null)

4. **Output/Evidence Model**
   - Three formats: JSON (structured), text (console), markdown (reports)
   - Kit-aware reports include asset metadata from kit
   - Evidence submission: POST to API if kit.evidence.endpoint exists, else save local file
   - Hybrid mode: Always save local + attempt API submit

5. **Schema Extensions**
   - New `diagnostic_mode` section in runbook.yaml
   - New `diagnostics` section in domain-kit.yaml
   - Action types: exec, http_get, tcp_connect, snmp_get, ssh_exec, foreach
   - Evidence capture modes: stdout, stderr, response_body, value

### Recommendation

Build **unified binary** supporting both kit-embedded and kit-less modes with graceful degradation. Implement in phases: (1) kit-optional engine, (2) go:embed integration, (3) output formatters, (4) advanced actions.

### Example Schema

Provided complete 5-step network diagnostic runbook (kit-less) demonstrating:
- Step-by-step validation (gateway, DNS, HTTPS, port scan)
- Evidence capture and labeling
- Markdown report generation
- Severity levels and failure handling

---

