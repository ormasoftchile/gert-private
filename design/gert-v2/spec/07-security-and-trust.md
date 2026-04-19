# Security and Trust

Gert v2 is designed for regulated, high-trust environments where operational actions must be auditable, accountable, and governed by explicit policy. This section defines the threat model, trust boundaries, process isolation mechanisms, credential handling, transport security, RBAC model, audit non-repudiation, and supply chain security — all of which are foundational to the architecture, not add-ons.

---

## Threat Model

| Threat | Mitigation | Section |
|---|---|---|
| Runbook author executes unauthorized commands | Command allowlist/denylist enforced at runtime core | §11 |
| Malicious extension injects code into host process | Out-of-process extension isolation; cryptographic signature verification | Extension Trust, §4 |
| Tool or extension leaks credentials into trace | Mandatory redaction for sensitive fields; `sensitive: true` input marking | Credential Handling |
| Unauthorized user triggers runbook execution | RBAC enforced at `gert serve` ingress | RBAC |
| Man-in-the-middle attack on JSON-RPC transport | TLS required for all non-localhost listeners | Transport Security |
| Replay attack on approval decision | Approval tokens signed with Ed25519; `approvalId` is single-use | RBAC, §11 |
| Compromised tool binary executes untrusted code | Tool binary checksum verification before invocation | Supply Chain |

Derived from NIST SP 800-53 and ISO/IEC 27001 guidance on privileged access management and audit logging.

[DIAGRAM: Governance enforcement pipeline — four stages (Runbook Load → Run Start → Step Dispatch → Execute Step) with per-stage policy checks above and BLOCKED exits below each stage.]

---

## Extension Trust Model

### Trust Levels

**Local trusted.**
Installed by the machine owner (via package manager or explicit installation to a system-wide extension directory such as `/usr/local/lib/gert-extensions/`). Runs as child process of the current user. No signature verification required; trust is conferred by the installation act.

**Project-scoped.**
Declared in the runbook project directory (e.g., `.gert/extensions.yaml`) or inline in the runbook's `extensions:` field. Runs in a restricted sandbox: only capabilities explicitly granted by the workspace governance policy are forwarded. File and network access are constrained to declared paths and hosts. Signature verification is *optional* but recommended.

**Remote/signed.**
Extensions distributed via package repository or downloaded from the internet. MUST carry a cryptographic signature verified against a trust store. The signature covers the extension manifest and binary. If verification fails, the extension is rejected before any subprocess is started.

### Extension Signature Format

Remote/signed extensions must include a `signature:` block in `gert-extension.yaml`:

```yaml
# gert-extension.yaml — signed extension
apiVersion: extension/v2

meta:
  name: acme.incident-pack
  version: "1.2.0"

signature:
  algorithm: ed25519
  publicKeyFingerprint: "sha256:abcd1234..."
  signatureData: "base64-encoded-signature"
  signedFields: ["meta", "capabilities", "entry-point", "transport"]

trust_chain:
  - fingerprint: "sha256:abcd1234..."
    issuer: "Acme Platform Team CA"
    notBefore: "2024-01-01T00:00:00Z"
    notAfter: "2025-01-01T00:00:00Z"
```

**Signature verification algorithm:**

1. Extract the public key fingerprint from the `signature` block.
2. Look up the corresponding public key in the trust store (`$HOME/.gert/trusted-keys/` or `/etc/gert/trusted-keys/`).
3. Reconstruct the canonical JSON of the signed fields (sorted keys, no whitespace).
4. Verify the Ed25519 signature over the canonical JSON bytes.
5. Check that the current time falls within the `notBefore` and `notAfter` window of the trust chain.

If any step fails, the extension is rejected with a `governance/extensionRejected` trace event and reason `signature_verification_failed`.

### Extension Verification Command

```bash
$ gert extension verify ./acme-extension/gert-extension.yaml \
    --trust-store /etc/gert/trusted-keys/
Extension: acme.incident-pack v1.2.0
Signature: valid (ed25519, expires 2025-01-01)
Issuer: Acme Platform Team CA
Capabilities: tool-registration, network
Trust level: remote/signed
Verification: PASSED
```

---

## Process Isolation

Extensions and tools MUST always run as separate OS processes. They are never loaded as shared libraries or executed in-process. This ensures that a crash, hang, or malicious extension cannot compromise the host process.

### Isolation Mechanisms

**Separate OS process.** Extensions are forked as child processes with their own address space, file descriptor table, and signal handlers.

**Filesystem access control.** Extensions with `capability/file-read` or `capability/file-write` declare accessible paths via `file-read-paths` and `file-write-paths` in the manifest. Enforcement:
- On OpenBSD: via `unveil(2)` to restrict visible paths.
- On Linux: via seccomp-bpf to filter `open(2)` and `openat(2)` syscalls.
- On all platforms: by validating all path arguments in `file-read` and `file-write` RPC methods before dispatch.

Attempts to access undeclared paths return a capability-denied error.

**Network access control.** Extensions with `capability/network` declare allowed hostnames via `network-hosts` in the manifest.
- Before dispatching an extension RPC with a target URL, the host checks the hostname against declared `network-hosts` patterns.
- Extensions MAY NOT listen for inbound connections.
- Wildcard patterns supported: `*.acme.example` permits `api.acme.example`, `prod.acme.example`, etc.

**Environment variable isolation.** Extensions receive ONLY environment variables explicitly forwarded by the runbook's `governance.allowed_env` list. Variables matching `governance.deny_env_vars` patterns are NEVER forwarded, even if they appear in `allowed_env`. The denylist takes precedence.

**Crash isolation.** If an extension process crashes:
1. Emit an `extension.crashed` trace event with exit code and signal number.
2. Mark the extension as unavailable for the remainder of the run.
3. Cancel all in-flight invocations to that extension with error code `-32099` (`extension_crashed`).
4. Continue the run; the host process does NOT terminate.

---

## Credential and Secret Handling

Gert v2 does not store secrets. Secrets are resolved from input providers at runtime, used for a single step invocation, and then discarded. Secrets MUST NOT appear in trace logs or captured output.

### Secret Resolution and Redaction

**Input provider sensitivity marking.** Provider definitions declare sensitive values via `sensitive: true`:

```yaml
# vault.provider.yaml
outputs:
  credentials:
    type: object
    sensitive: true
```

When the host resolves `from: vault.credentials`, it marks the returned value as sensitive in memory.

**Tool sensitive inputs.** Tool action definitions declare sensitive input parameters via `sensitive_inputs`:

```yaml
# my-tool.tool.yaml
actions:
  deploy:
    args:
      api_key:
        type: string
        required: true
    sensitive_inputs: [api_key]
```

The host marks the `api_key` field as sensitive when preparing the tool invocation envelope. The tool runtime redacts this field in all trace events.

**Mandatory redaction.** All values marked `sensitive: true` are redacted before being written to the trace or displayed in any adapter UI. Redaction replaces the value with `<redacted>`.

The runbook's `governance.redact` block defines regex patterns applied to all captured output (stdout, stderr, JSON responses) before storage. See §11 for the full redaction specification.

### Secret Storage Policy

Gert v2 MUST NOT:
- Store secrets in the runbook YAML.
- Write secrets to disk (trace files, state snapshots, etc.).
- Cache secrets across steps (unless explicitly declared by the input provider's `cache_scope`).
- Transmit secrets over unencrypted transports.

Secrets MUST be resolved via input providers that integrate with secret management systems (e.g., HashiCorp Vault, AWS Secrets Manager, 1Password).

---

## Transport Security

Two primary transport modes for JSON-RPC communication:

1. **stdio**: JSON-RPC over stdin/stdout of a child process. No network exposure; inherently local. No TLS required.
2. **HTTP/WebSocket**: JSON-RPC over HTTP for `gert serve` mode. TLS and authentication required for non-localhost.

### TLS Requirements

**Localhost exception.** If `gert serve` binds to `127.0.0.1` or `localhost`, TLS is optional.

**Non-localhost listeners.** If `gert serve` binds to `0.0.0.0`, a public IP, or a hostname, TLS is **mandatory**. The host MUST refuse to start without a valid TLS certificate and private key.

```bash
$ gert serve --http 0.0.0.0:8443 \
    --tls-cert /etc/gert/tls/cert.pem \
    --tls-key /etc/gert/tls/key.pem
```

TLS 1.3 is required. TLS 1.2 is supported for compatibility. TLS 1.1 and earlier are rejected.

### Client Authentication

**Bearer token authentication.**
Clients include `Authorization: Bearer <token>` on all HTTP requests. Tokens are validated against a configured token store.

```bash
$ gert serve --http :8443 --auth-mode bearer \
    --bearer-tokens /etc/gert/tokens.yaml
```

```yaml
# tokens.yaml
tokens:
  - token: "abc123..."
    actor: "ops@acme.example"
    roles: ["operator", "viewer"]
```

**Mutual TLS (mTLS).** The host requires clients to present a valid X.509 client certificate signed by a trusted CA. The client certificate's subject CN or SAN is used as the actor identity. Recommended for production deployments.

```bash
$ gert serve --http :8443 --auth-mode mtls \
    --tls-client-ca /etc/gert/tls/client-ca.pem
```

### WebSocket Security

WebSocket connections for the event stream inherit the HTTP server's security context:
- If the HTTP server uses TLS, WebSocket connections are encrypted (`wss://`).
- Bearer tokens or client certificates are validated before the WebSocket upgrade.
- Once upgraded, the WebSocket connection is bound to the authenticated actor identity for its lifetime.

### Extension JSON-RPC Transport

Extensions communicate with the host via stdio JSON-RPC. This transport is not exposed to the network and does not require TLS. The process boundary is the trust boundary.

---

## RBAC for `gert serve`

When running in server mode (`gert serve --http`), gert enforces role-based access control on all API operations.

### Actor Identity

An **actor** is the authenticated identity making a request, derived from:
- The bearer token (mapped to an actor ID via the token store), or
- The client certificate subject CN/SAN (in mTLS mode).

Actor identity is recorded in all trace events where a human or external system initiates an action (run start, approval, cancellation).

### Roles

**Viewer.** Read-only access.
- List runs (`GET /runs`)
- Get run details (`GET /runs/:id`)
- Subscribe to event stream (`GET /ws` or `GET /events`)
- Read trace files

Cannot start runs, approve, or cancel.

**Operator.** Includes all viewer permissions plus:
- Start a run (`POST /exec/start`)
- Cancel a run (`POST /exec/cancel`)
- Submit approvals (`POST /approvals`)

Cannot modify governance policy or manage roles.

**Admin.** Includes all operator permissions plus:
- Modify workspace governance policy
- Manage role assignments (add/remove actors from roles)
- Configure extension trust store

### Role Assignment

Roles are assigned in the workspace config (`gert-project.yaml`):

```yaml
# gert-project.yaml
access:
  roles:
    - role: admin
      actors:
        - "alice@acme.example"
        - "bob@acme.example"
    - role: operator
      actors:
        - "ops-team@acme.example"
        - "oncall@acme.example"
    - role: viewer
      actors:
        - "*@acme.example"  # All Acme employees
```

Actor patterns support wildcards for organization-wide grants.

### Approval Authorizer Role

```yaml
steps:
  - id: delete_database
    cli: "kubectl delete database prod-db"
    approvals:
      min: 1
      roles: ["dba", "admin"]
```

When the run reaches this step, a `approval/required` trace event is emitted and the step is paused. Only actors with the `dba` or `admin` role may submit a valid approval.

**Approval submission:**

```
POST /approvals
{
  "runId": "run-abc123",
  "stepId": "delete_database",
  "decision": "approve",
  "actorId": "alice@acme.example",
  "signature": "<ed25519-signature-of-approval-payload>",
  "timestamp": "2026-04-18T12:00:00Z"
}
```

**Host verification sequence:**
1. Actor is authenticated and has one of the required roles.
2. Signature is valid (Ed25519 over canonical JSON of `runId`, `stepId`, `decision`, `actorId`, `timestamp`).
3. `timestamp` is within a 5-minute clock skew window.
4. `runId` and `stepId` match an active approval gate.
5. This approval has not been submitted before (replay protection).

If all checks pass, the approval is recorded (`approval/received`) and the step proceeds if minimum approval count is reached.

---

## Audit Non-Repudiation

Non-repudiation ensures that governance decisions (approvals and rejections) cannot be denied by the actor who made them. Required for SOC2, ISO27001, HIPAA compliance.

### Governance Decision Recording

Every governance decision event MUST record:
- Actor identity (authenticated user or service principal)
- Timestamp (RFC 3339 with timezone)
- Decision payload (what was approved/rejected)
- Signature (Ed25519 over the canonical decision payload)

```json
{
  "eventType": "governance/approval_received",
  "timestamp": "2026-04-18T12:00:00Z",
  "runId": "run-abc123",
  "stepId": "delete_database",
  "actor": "alice@acme.example",
  "decision": "approve",
  "signature": "ed25519:abcd1234...",
  "signatureAlgorithm": "ed25519",
  "signaturePublicKey": "sha256:pubkey-fingerprint"
}
```

The signature covers the fields `runId`, `stepId`, `actor`, `decision`, and `timestamp`.

### Trace File Integrity (HMAC Chaining)

The JSONL trace file is append-only, but this alone does not prevent tampering. The host optionally computes an HMAC chain over the trace:

1. At run start, initialize `H_0 = HMAC-SHA256(key, runId)`.
2. After writing each trace event `E_i`, compute `H_i = HMAC-SHA256(key, H_{i-1} || E_i)`.
3. Store `H_i` in the event as `integrityHash`.
4. At run completion, emit a final `run.completed` event with `finalHash: H_N`.

Verification: recompute the HMAC chain from `H_0` forward. If any event is modified or deleted, the chain breaks. The HMAC key is derived from the workspace master key or provided via `--integrity-key`.

### SHA256 Evidence Capture

SHA256 hashes of all evidence artifacts (attachments, screenshots, CLI output files) are written to the trace event immediately after capture. Combined with HMAC trace chaining, this provides end-to-end non-repudiation for the entire run.

---

## Supply Chain Security

### Tool Binary Checksums

Tool definitions MAY declare a `checksum:` field in `.tool.yaml`:

```yaml
# kubectl.tool.yaml
meta:
  name: kubectl
  binary: kubectl

checksum:
  algorithm: sha256
  digest: "a3b4c5d6e7f8..."
```

Before invoking the tool, the host:
1. Resolves the binary path (via `PATH` lookup or absolute path).
2. Computes the SHA256 hash of the binary file.
3. Compares the computed hash to `checksum.digest`.
4. If the hashes do not match, blocks the tool invocation with a `governance/toolChecksumMismatch` trace event.

### `gert verify` Command

```bash
$ gert verify --workspace .
Verifying tools...
  ✓ kubectl: checksum verified (sha256:a3b4c5...)
  ✓ curl: no checksum declared (skipped)
  ✗ jq: checksum mismatch (expected sha256:abc123, got sha256:def456)

Verifying extensions...
  ✓ acme.incident-pack: signature valid (ed25519, expires 2025-01-01)
  ✗ debug-tools: no signature (required for remote extensions)

Verification: FAILED (2 errors)
```

SHOULD be run in CI/CD pipelines before deploying runbooks to production.

### Extension Signature Verification

Remote extensions MUST carry a cryptographic signature ensuring:
- The extension binary and manifest have not been tampered with.
- The extension was signed by a trusted authority (public key in the trust store).
- The signature has not expired (`notAfter` check).

If verification fails, the extension is rejected before any subprocess is started.

---

## Security Recommendations

For production deployments:

1. **Enable TLS for all `gert serve` deployments** — even internal networks can be compromised.
2. **Use mTLS for client authentication** — bearer tokens are vulnerable to theft; client certificates provide stronger identity.
3. **Enable HMAC trace chaining** — provides cryptographic integrity for audit trails.
4. **Declare tool binary checksums** — prevents execution of tampered binaries.
5. **Require signatures for all remote extensions** — do not trust unsigned code from the internet.
6. **Run `gert verify` in CI** — catch integrity violations before production.
7. **Use least-privilege capability grants** — extensions should request only the capabilities they need.
8. **Regularly rotate approval keys** — Ed25519 keys used for signing approvals should be rotated every 90 days.

These recommendations align with NIST SP 800-53 controls (AC-2, AC-6, AU-2, AU-9, SI-7).
