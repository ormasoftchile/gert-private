# Integration Contracts: Campaign Identity Validation + Completion Notifications

**Author:** David (Integration Engineer)  
**Date:** 2026-06-04T12:28:39.293-04:00  
**Status:** Draft — working design for implementation and ratification  
**Requested by:** ormasoftchile (Gerardo)

---

## Purpose

Define the customer-facing integration contract for campaign-based runs where:
1. GERT emails a magic link to the end user.
2. The user lands in a white-labeled portal.
3. GERT validates identity against the customer company's endpoint.
4. GERT runs the assigned runbook.
5. GERT notifies the customer when the user and/or campaign reaches a terminal state.

This design is defensive by default: no silent drops, every outbound event is retried, every token redemption is atomic, and every external integration has an idempotency strategy.

## Non-Negotiable Integration Rules

- **At-least-once outbound delivery:** customer webhooks can be delivered more than once; the receiver must deduplicate.
- **No synchronous dependency after run completion:** final customer notification is queued before the run is considered durably finished.
- **Opaque magic links:** links never expose user PII in the URL.
- **Versioned contracts:** request/response and webhook schemas carry explicit versions.
- **Per-tenant isolation:** timeouts, circuit breakers, signing secrets, auth config, and dead-letter handling are scoped to a single customer endpoint registration.
- **Poison-message handling always:** malformed responses and permanent auth/configuration failures go to DLQ with operator visibility.

---

## 1. Customer Identity Endpoint Contract

### 1.1 Contract Summary

| Item | Design |
|---|---|
| Protocol | HTTPS only |
| Method | `POST` |
| Content type | `application/json` |
| Invocation point | After magic link redemption lease is acquired, before runbook start |
| Caller | GERT API/App Service |
| Idempotency | GERT sends stable `X-GERT-Idempotency-Key` per link redemption attempt |
| Correlation | GERT sends `X-GERT-Correlation-Id`; customer echoes it in logs/response |
| Response model | Explicit decision: `approved`, `denied`, or `error` |
| Success criteria | HTTP `200` with valid schema and terminal decision |
| Retry policy | No background retry in click path; user-visible failure if request exceeds timeout budget |

### 1.2 Request GERT Sends

**Headers**

| Header | Required | Notes |
|---|---|---|
| `Content-Type: application/json` | Yes | Fixed |
| `Accept: application/json` | Yes | Fixed |
| `User-Agent: GERT-IdentityValidator/1.0` | Yes | Fixed |
| `X-GERT-Contract-Version: 2026-06-04` | Yes | Request/response contract version |
| `X-GERT-Tenant-Id` | Yes | GERT tenant/customer id |
| `X-GERT-Campaign-Id` | Yes | Campaign being executed |
| `X-GERT-Link-Id` | Yes | One magic-link record id |
| `X-GERT-Correlation-Id` | Yes | Traceable per HTTP call |
| `X-GERT-Idempotency-Key` | Yes | Stable for the same redemption lease |
| `X-GERT-Timestamp` | Yes | RFC3339 timestamp of outbound call |
| `Authorization` | Conditional | Bearer/API key, if configured |
| `X-GERT-Signature` | Optional | HMAC signature when customer wants signed requests |

**Request body**

```json
{
  "contract_version": "2026-06-04",
  "tenant_id": "tenant_acme",
  "campaign": {
    "campaign_id": "camp_2026_q2_kyc",
    "campaign_external_ref": "acme-onboarding-q2",
    "runbook_id": "rb_identity_onboarding_v3"
  },
  "link": {
    "link_id": "lnk_01JX8YF5MS5Y6N7X8Q2A0B1C2D",
    "issued_at": "2026-06-04T12:28:39.293-04:00",
    "expires_at": "2026-06-04T12:28:39.293-04:00",
    "nonce": "e4b8f7b5b7d84c1e9df3b7f5b6a2a9d1"
  },
  "subject_hint": {
    "external_user_ref": "usr_49017",
    "email": "user@example.com",
    "email_hash": "sha256:ec3f...",
    "locale": "es-CL"
  },
  "portal_context": {
    "portal_base_url": "https://portal.customer.example",
    "ip_address": "198.51.100.10",
    "user_agent": "Mozilla/5.0",
    "clicked_at": "2026-06-04T12:28:39.293-04:00"
  },
  "requested_claims": [
    "customer_user_id",
    "identity_status",
    "display_name",
    "segment",
    "risk_tier"
  ]
}
```

**Design notes**
- `subject_hint` is a hint, not proof. The customer's endpoint is the authority.
- GERT should omit `email` when the customer asks for opaque-ID-only mode and can validate via `external_user_ref` alone.
- GERT must never send more PII than the campaign contract allows.

### 1.3 Response GERT Expects

#### Approved

```json
{
  "contract_version": "2026-06-04",
  "decision": "approved",
  "customer_request_id": "idv-784512",
  "subject": {
    "customer_user_id": "cust_u_8821",
    "external_user_ref": "usr_49017",
    "display_name": "Jane Doe"
  },
  "identity": {
    "status": "verified",
    "verified_at": "2026-06-04T12:28:39.293-04:00",
    "method": "customer_system_lookup"
  },
  "run_context": {
    "segment": "enterprise",
    "risk_tier": "low",
    "country": "CL"
  }
}
```

#### Denied (business denial, terminal)

```json
{
  "contract_version": "2026-06-04",
  "decision": "denied",
  "customer_request_id": "idv-784513",
  "reason": {
    "code": "USER_NOT_ELIGIBLE",
    "message": "User is not eligible for this campaign"
  }
}
```

#### Error (non-terminal, retryable by user or operator depending on cause)

```json
{
  "contract_version": "2026-06-04",
  "decision": "error",
  "customer_request_id": "idv-784514",
  "error": {
    "code": "DEPENDENCY_TIMEOUT",
    "message": "Upstream identity provider timeout",
    "retryable": true
  }
}
```

### 1.4 Response Handling Rules

| Condition | GERT behavior |
|---|---|
| `200` + valid `approved` | Mark identity check passed; start runbook |
| `200` + valid `denied` | Mark link terminal-denied; do not start runbook |
| `200` + valid `error` + `retryable=true` | Release redemption lease after cool-off; show retryable failure page |
| `200` + invalid schema | Treat as integration failure; do not start runbook |
| `4xx` auth/config errors | Treat as permanent configuration failure; open incident/DLQ audit record |
| `408`, `429`, `5xx`, network/TLS failure | Treat as transient integration failure |
| Response contract version mismatch | Fail closed; do not guess field mapping |

### 1.5 Authentication Options Supported

| Customer auth mode | How GERT calls it | Notes |
|---|---|---|
| API key | `Authorization: ApiKey <secret>` or customer-defined header | Simplest MVP; rotate per tenant |
| OAuth 2.0 client credentials | GERT fetches token from customer token endpoint; caches until expiry minus safety window | Best for enterprise APIs |
| mTLS | GERT presents tenant-specific client cert from Key Vault-backed secret store | Strong mutual auth; heavier onboarding |
| IP allowlist | Customer allowlists fixed outbound NAT IPs for GERT | Add alongside API key/OAuth; not sufficient alone |
| HMAC request signing | GERT signs canonical request with shared secret | Optional additional integrity layer |

**Recommended order**
1. OAuth 2.0 client credentials for mature enterprise customers.
2. API key + IP allowlist for fast onboarding.
3. mTLS only when required by customer security policy.

### 1.6 Versioning Strategy

| Concern | Design |
|---|---|
| Request schema version | `X-GERT-Contract-Version` header + `contract_version` body field |
| Response schema version | Customer must echo `contract_version` |
| Breaking customer API change | Register a **new endpoint version** in GERT; old and new versions run side-by-side during cutover |
| Backward-compatible change | Additive fields allowed; GERT ignores unknown fields |
| Cutover safety | `ping` + `schema validation` + `signed test call` before campaign activation |

**Rule:** GERT does not auto-adapt to breaking changes. A breaking change requires a new endpoint registration record and explicit re-validation.

### 1.7 Timeout Budget

| Phase | Budget |
|---|---|
| DNS + TCP + TLS connect | 2 seconds |
| Request write | 1 second |
| Time to first byte | 2 seconds |
| Response body read + validation | 3 seconds |
| **Absolute max per identity check** | **8 seconds** |

If 8 seconds is exceeded, GERT fails the identity check for that click and shows a retriable "customer identity service unavailable" page. The runbook does not start.

### 1.8 Circuit Breaker Per Customer Endpoint

| Setting | Value |
|---|---|
| Scope | Per registered identity endpoint |
| Open threshold | 5 consecutive transient failures |
| Open duration | 5 minutes |
| Half-open behavior | Permit 1 probe request; if it succeeds, close breaker; if it fails, reopen for 5 minutes |
| Breaker-visible result | Portal shows controlled outage message; no customer API call made while open |
| Alerting | Trigger tenant integration alert when breaker opens |

The breaker only counts transient failures (`408`, `429`, `5xx`, network, TLS, timeout). Configuration/auth failures (`401`, `403`, bad cert, schema mismatch) are handled as permanent misconfiguration until the tenant changes settings.

---

## 2. Magic Link Design

### 2.1 Token Design

**Recommendation:** use an **opaque token**, not a self-describing JWT, in the email URL.

| Aspect | Design |
|---|---|
| URL token | Random 256-bit secret, base64url encoded |
| Stored lookup key | SHA-256 hash of token |
| Token contents visible to user | Nothing meaningful; no user id or email in URL |
| Server-side record contains | `link_id`, `tenant_id`, `campaign_id`, `external_user_ref`, `email/email_hash`, `expires_at`, `status`, `attempt_count`, `nonce` |
| Expiry | Campaign-defined, default 7 days |
| URL example | `https://portal.customer.example/magic-link?t=<opaque-token>` |

Why opaque instead of JWT:
- No PII leakage in browser history, logs, referrers, or screenshots.
- Rotation and revocation are server-side only.
- Contract changes do not require token format migration.

### 2.2 Where Token Is Generated and Stored

Use Cosmos DB **`campaigns` container**, but store each invitation as its own document type rather than embedding user links inside one campaign root document.

**Document shape (same container, separate document)**

```json
{
  "id": "lnk_01JX8YF5MS5Y6N7X8Q2A0B1C2D",
  "document_type": "campaign_link",
  "tenant_id": "tenant_acme",
  "campaign_id": "camp_2026_q2_kyc",
  "external_user_ref": "usr_49017",
  "email": "user@example.com",
  "email_hash": "sha256:ec3f...",
  "token_hash": "sha256:7d3c...",
  "nonce": "e4b8f7b5b7d84c1e9df3b7f5b6a2a9d1",
  "status": "issued",
  "redeem_lease_expires_at": null,
  "redeemed_at": null,
  "run_id": null,
  "attempt_count": 0,
  "expires_at": "2026-06-04T12:28:39.293-04:00",
  "ttl": 604800
}
```

### 2.3 One-Time Use Enforcement

Use a **conditional patch / optimistic concurrency** pattern in Cosmos DB.

**State machine**

| State | Meaning | Allowed next states |
|---|---|---|
| `issued` | Link can be redeemed | `redeeming`, `expired` |
| `redeeming` | Short lease held by one request | `issued`, `redeemed`, `denied`, `expired` |
| `redeemed` | Runbook started successfully | terminal |
| `denied` | Customer identity endpoint returned terminal denial | terminal |
| `expired` | Link lifetime ended | terminal |

**Atomic flow**
1. User clicks link.
2. GERT hashes token and looks up the link document.
3. GERT issues a conditional patch: `status = issued -> redeeming`, sets `redeem_lease_expires_at = now + 30s`, increments `attempt_count`.
4. If the patch fails because the document changed, redemption is rejected as already in progress or already used.
5. GERT calls the customer identity endpoint while it holds the lease.
6. On `approved`, GERT creates the run and conditionally patches `redeeming -> redeemed` with `redeemed_at` and `run_id`.
7. On terminal `denied`, GERT conditionally patches `redeeming -> denied`.
8. On transient infrastructure failure, GERT conditionally patches `redeeming -> issued` (or lets the 30s lease expire) so the user can retry.

This prevents double-starting runs while still allowing recovery from partial failure.

### 2.4 Data GERT Has at Click Time vs. Data From Identity Endpoint

| Available to GERT at click time | Must come from customer identity endpoint |
|---|---|
| `tenant_id` | Authoritative validation decision |
| `campaign_id` | Canonical `customer_user_id` |
| `link_id` | Eligibility/active status |
| `external_user_ref` (if pre-registered) | Current segment/risk flags |
| `email` or `email_hash` | Optional display name |
| Requested runbook id | Any customer-owned identity attributes allowed into run context |
| Portal telemetry (IP, user agent, click timestamp) | Reason code for denial/error |

### 2.5 Pre-Registered Users vs Endpoint-Only Lookup

| Model | How it works | Pros | Risks |
|---|---|---|---|
| Pre-registered roster | Customer uploads users/emails before campaign | GERT can send emails directly; easiest audience control | Customer data can drift before click |
| Endpoint-only lookup | GERT knows little or no user data until click | Minimal stored PII | Hard for GERT to send email unless customer sends links |
| Hybrid (**recommended**) | Customer pre-registers minimal audience (`external_user_ref` + email or customer sends links), endpoint confirms current eligibility at click | Best UX + current validation | Requires two contracts |

**Recommendation:** Hybrid. GERT should know enough to target the campaign audience, but the customer endpoint remains the source of truth for real-time eligibility.

---

## 3. Completion Notification to Customer

### 3.1 Delivery Topology

**Recommended production path**
1. Run reaches a publishable state.
2. GERT writes a summary event record.
3. GERT enqueues an outbound message to **Service Bus `webhooks` queue**.
4. Webhook worker dequeues and performs HTTPS `POST` to the customer endpoint.
5. On success, the event is marked delivered.
6. On failure, the message is rescheduled with exponential backoff.
7. On retry exhaustion or permanent failure, the message goes to DLQ and alerts are raised.

**Decision:** internal delivery is queue-backed; external delivery is HTTPS. This gives at-least-once guarantees without making the customer operate Azure messaging infrastructure.

### 3.2 Event Families Customers Can Subscribe To

| Event type | Default | Notes |
|---|---|---|
| `campaign.started` | Optional | Campaign opened for processing |
| `campaign.closed` | Yes | Final close because completed, deadline reached, or cancelled |
| `user.started` | Yes | User redeemed link and run started |
| `user.completed` | Yes | Final success |
| `user.failed` | Yes | Final failure |
| `user.abandoned` | Optional | User timed out/never finished |
| `user.identity_denied` | Optional | Link redeemed but customer endpoint denied eligibility |
| `user.identity_error` | Optional | Identity check failed due to customer integration issue |
| `milestone.reached` | Optional | Named business milestones only; not raw step stream |

**Default rule:** expose **summary lifecycle events**, not raw step-level events. Raw step events over-couple the customer to runbook internals and create unnecessary webhook volume.

### 3.3 Event Envelope

```json
{
  "event_id": "evt_01JX8ZB7A3R3P0W5W8BZ1P2M9A",
  "event_type": "user.completed",
  "event_version": "2026-06-04",
  "occurred_at": "2026-06-04T12:28:39.293-04:00",
  "tenant_id": "tenant_acme",
  "campaign_id": "camp_2026_q2_kyc",
  "campaign_user_id": "cmpusr_8821",
  "run_id": "run_01JX8Z93A5M1K9Q6T0Z8E4R7N1",
  "delivery": {
    "attempt": 1,
    "idempotency_key": "tenant_acme:camp_2026_q2_kyc:cmpusr_8821:user.completed:v1"
  },
  "data": {
    "external_user_ref": "usr_49017",
    "customer_user_id": "cust_u_8821",
    "status": "completed",
    "identity": {
      "status": "verified",
      "verified_at": "2026-06-04T12:28:39.293-04:00",
      "customer_request_id": "idv-784512"
    },
    "runbook": {
      "runbook_id": "rb_identity_onboarding_v3",
      "runbook_version": "3.0.0"
    },
    "result": {
      "completed_at": "2026-06-04T12:28:39.293-04:00",
      "duration_seconds": 418,
      "outcome_code": "SUCCESS"
    },
    "artifacts": {
      "summary_url": "https://gert.example/api/v1/runs/run_01JX8Z93A5M1K9Q6T0Z8E4R7N1/summary",
      "receipt_url": "https://gert.example/api/v1/runs/run_01JX8Z93A5M1K9Q6T0Z8E4R7N1/receipt"
    },
    "metadata": {
      "campaign_external_ref": "acme-onboarding-q2",
      "locale": "es-CL"
    }
  }
}
```

### 3.4 HTTP Headers GERT Sends to Customer Webhook Endpoint

| Header | Purpose |
|---|---|
| `Content-Type: application/json` | Payload type |
| `User-Agent: GERT-Webhooks/1.0` | Client identity |
| `X-GERT-Event-Id` | Stable across retries |
| `X-GERT-Event-Type` | Fast routing |
| `X-GERT-Event-Version` | Schema version |
| `X-GERT-Delivery-Attempt` | Current attempt number |
| `Idempotency-Key` | Receiver deduplication key |
| `X-GERT-Timestamp` | Signature input |
| `X-GERT-Signature` | HMAC-SHA256 signature over canonical payload |

### 3.5 Delivery Guarantees, Retry Policy, and Dead-Letter Behavior

| Condition | Action |
|---|---|
| `2xx` | Success; mark delivered |
| `408`, `429`, `5xx`, network/TLS failure | Retry with exponential backoff + jitter |
| `400`, `401`, `403`, `404`, `410`, `422` | Permanent delivery failure; move to DLQ immediately |
| Malformed non-JSON response body | Ignore body, classify by status code |
| Duplicate send caused by worker crash | Same `event_id` and `Idempotency-Key` are reused |

**Retry schedule (recommended)**
- Initial delivery: immediate
- Retry 1: +1 minute
- Retry 2: +5 minutes
- Retry 3: +30 minutes
- Retry 4: +4 hours
- Retry 5: +18 hours
- Jitter: ±20%

**Dead-letter policy**
- Active queue TTL: 7 days
- Retry window: ~24 hours from first failure
- DLQ retention: indefinite until replay/purge by operator policy
- Alerting: first alert at retry 3, second alert on DLQ, dashboard remains red until resolved
- Replay: operator or tenant admin can replay one event or a filtered range after endpoint recovery

### 3.6 Idempotency Expectations for the Customer Receiver

The customer's receiving endpoint must deduplicate on **`Idempotency-Key`**, not only `event_id`.

| Key | Why |
|---|---|
| `event_id` | Identifies this physical event record |
| `Idempotency-Key` | Identifies the semantic effect to apply exactly once |

Example receiver rule:
- If `Idempotency-Key` already processed: return `200 OK` and do nothing.
- If body differs for the same processed key: log conflict and return `409 Conflict` or `200` with no-op, based on customer policy.

### 3.7 If the Customer Webhook Endpoint Is Down for Hours

1. Events remain durable in Service Bus and scheduled retries continue.
2. If recovery happens within the retry window, normal delivery resumes automatically.
3. If the outage lasts longer, the event lands in DLQ and remains replayable.
4. Tenant alerts stay open until either replay succeeds or the customer explicitly waives the event.
5. No event is silently dropped because the customer endpoint was unavailable.

### 3.8 Partial Completions and Event Granularity

| Option | Supported? | Recommendation |
|---|---|---|
| Raw step-level events | Not by default | Avoid in customer-facing contract |
| Named milestone events | Yes | Use only for business milestones defined in campaign config |
| Final-only events | Yes | Minimum viable subscription set |

**Recommendation:** send `user.started`, optional `milestone.reached`, and one terminal event: `user.completed`, `user.failed`, `user.abandoned`, or `user.identity_denied`.

---

## 4. Customer Onboarding Integration Flow

### 4.1 Tenant Registration

| Required data | Purpose |
|---|---|
| Tenant/company legal name | Contract and audit |
| Tenant technical contact | Integration alerts |
| Tenant operations contact | DLQ/replay coordination |
| Data residency / region | Storage placement |
| White-label portal domain(s) | Link generation and branding |
| Outbound notification email settings | Magic-link delivery, if GERT sends emails |
| Customer identifiers | `tenant_id`, external account reference |

### 4.2 Identity Endpoint Registration

| Field | Required | Notes |
|---|---|---|
| Endpoint URL | Yes | HTTPS only |
| Auth method | Yes | API key, OAuth2 CC, mTLS, or combo |
| Auth material | Yes | Secret, token endpoint, cert reference |
| Allowed contract version | Yes | E.g. `2026-06-04` |
| Timeout override | Optional | Can only decrease from global max if customer wants stricter UX |
| Customer request header mapping | Optional | For legacy APIs |
| Ping/test endpoint | Strongly recommended | Validates DNS/TLS/auth/schema before activation |
| Sandbox example response | Strongly recommended | Used for onboarding validation |

**Activation gate:** endpoint registration is not active until GERT completes a signed test call and validates the response schema.

### 4.3 Webhook Endpoint Registration

| Field | Required | Notes |
|---|---|---|
| Webhook URL | Yes | HTTPS only |
| Auth method | Yes | HMAC secret, API key, OAuth, mTLS |
| Event subscriptions | Yes | Selected from supported event family |
| Delivery secret/cert ref | Yes | Per tenant endpoint |
| Custom timeout | Optional | Max 10 seconds per delivery attempt |
| Replay contact | Yes | Who gets DLQ alerts |
| Test webhook/ping | Yes | Must return `2xx` before activation |

### 4.4 Campaign Definition

| Field | Required | Purpose |
|---|---|---|
| `campaign_id` | Yes | Durable campaign identifier |
| Campaign external reference | Optional | Customer correlation |
| Runbook id + version | Yes | Which runbook to execute |
| Audience source | Yes | Uploaded roster, API import, or customer-sent links |
| Deadline / closes_at | Yes | Determines campaign close behavior |
| Reminder policy | Optional | Email/SMS resend cadence |
| Identity endpoint version | Yes | Binds campaign to a validated contract |
| Webhook subscription set | Yes | What notifications customer receives |
| PII policy | Yes | What fields GERT may store and emit |
| Locale/branding profile | Optional | White-label portal behavior |

### 4.5 Suggested Onboarding Sequence

1. Register tenant.
2. Register and validate identity endpoint.
3. Register and validate webhook endpoint.
4. Upload audience or request customer-managed link distribution.
5. Bind runbook + campaign deadline + event subscriptions.
6. Execute end-to-end test campaign with one synthetic user.
7. Promote configuration from test to active.

---

## 5. The Data Handoff — What Flows Where

### 5.1 Identity Data Passed into Runbook Context

Only fields explicitly whitelisted in campaign configuration flow from the customer identity endpoint into the runbook context.

| Field | Source | Stored by GERT? | Notes |
|---|---|---|---|
| `customer_user_id` | Identity endpoint | Yes | Required for downstream correlation |
| `external_user_ref` | Campaign roster and/or endpoint | Yes | Opaque preferred |
| `display_name` | Identity endpoint | Optional | Only if needed in UX/receipt |
| `identity.status` | Identity endpoint | Yes | `verified`, `denied`, etc. |
| `identity.verified_at` | Identity endpoint | Yes | Audit evidence |
| `segment`, `risk_tier`, `country` | Identity endpoint | Optional | Only if runbook logic needs them |
| Email address | Campaign roster | Only if GERT sends mail | Otherwise customer should keep it |

### 5.2 What GERT Captures vs What the Customer Retains

| Concern | GERT captures | Customer retains |
|---|---|---|
| Campaign execution | Run id, campaign id, timestamps, status, artifacts, audit trace | Source CRM/ERP state |
| Identity proof | Validation decision, customer request id, allowed claims | Full identity record and evidence |
| User contact data | Minimal delivery data if GERT sends links | Master user profile |
| Legal artifact | Acceptance record, content hash, receipt, trace reference | Business system of record |

### 5.3 PII Minimization

**Preferred model:** GERT stores opaque identifiers and only the minimum contact data required for delivery.

| Data type | Recommendation |
|---|---|
| User ids | Use customer opaque ids, not national IDs/passport numbers |
| Email | Store only when GERT must send the magic link; otherwise customer distributes link |
| Sensitive profile attributes | Do not return from identity endpoint unless runbook strictly requires them |
| Webhook payloads | Use opaque ids and status codes; exclude unnecessary free-text PII |
| Logs | Never log raw tokens, secrets, or full PII payloads |

### 5.4 Contract Acceptance Record (Legally Enforceable Minimum)

For each completed or explicitly denied flow, GERT should persist an acceptance record containing:

| Field | Why it matters |
|---|---|
| `tenant_id`, `campaign_id`, `run_id`, `link_id` | Traceability |
| Legal text / document version id | What the user was shown |
| Rendered content hash | Proves no post-hoc edit |
| `customer_user_id` / `external_user_ref` | Who acted |
| Identity validation result + `customer_request_id` | Why GERT believed the user identity was valid |
| `accepted_at` / `denied_at` | When the act occurred |
| Portal IP address + user agent | Supplementary evidence |
| Signature / affirmation method | Typed, click-through, drawn signature, etc. |
| Receipt / evidence artifact URLs or hashes | Retrieval and audit |

If the workflow is legally significant, the acceptance record must be immutable after completion and linked to the authoritative trace/artifact set.

---

## 6. Failure Inventory (Integration Layer Only)

| Touchpoint | Failure mode | Detection | Required behavior | Recovery |
|---|---|---|---|---|
| Customer identity endpoint | Endpoint down | Connect/TLS/network error | Fail closed; do not start runbook; increment breaker counter; release redemption lease | User retries later; operator alerted if breaker opens |
| Customer identity endpoint | Slow response | 8s timeout exceeded | Abort call; do not start runbook; classify transient | Same as above |
| Customer identity endpoint | Invalid JSON/schema | Schema validation failure | Fail closed; mark integration error; no run starts | Customer fixes contract; onboarding test should catch earlier |
| Customer identity endpoint | Auth failure (`401`/`403`) | HTTP status | Treat as permanent configuration issue; no retries in click path | Tenant updates credentials/cert |
| Magic link store (Cosmos DB) | Write failure at generation | Cosmos write error | Do not send email; campaign user stays unsent; raise operator error | Retry generation job safely with same audience record |
| Magic link store (Cosmos DB) | Read failure at redemption | Cosmos read error | Show retriable error page; no identity call | Retry later |
| Magic link store (Cosmos DB) | Conditional write conflict | ETag / patch precondition failure | Treat as duplicate redemption or in-flight request | Show already-used/in-progress page |
| Runbook execution events | Service Bus enqueue failure before terminal event publish | Send failure returned by SDK | Keep run in "completion_pending" state; background recovery job retries enqueue | Do not mark publish complete until queue write succeeds |
| Completion webhook | Endpoint down | Timeout/network/5xx | Retry on Service Bus schedule; keep delivery record open | Auto-retry then DLQ |
| Completion webhook | Auth rejected | `401`/`403` | Permanent failure; DLQ immediately; alert tenant | Tenant fixes webhook auth and replays |
| Completion webhook | Malformed response | Non-2xx or protocol issue | Ignore response body except for diagnostics; classify by status code | As above |
| Completion webhook | Duplicate delivery | Same event retried or worker crash | Reuse same `event_id` + `Idempotency-Key`; customer dedups | Safe no-op at receiver |
| Campaign close event | All users done | Campaign state evaluator sees terminal states for all targeted users | Emit `campaign.closed` with `close_reason = all_users_terminal` | None |
| Campaign close event | Deadline reached | Scheduler observes `closes_at` passed with remaining non-terminal users | Mark remaining users `expired`/`abandoned` per policy; emit `campaign.closed` with `close_reason = deadline_reached` | Customer may reopen or launch follow-up campaign |
| Campaign close event | Customer cancels | Authenticated cancel command received | Stop outstanding reminders, block new redemptions, emit `campaign.closed` with `close_reason = cancelled_by_customer` | Manual follow-up campaign if needed |

---

## Recommended Defaults

| Area | Default |
|---|---|
| Identity timeout | 8 seconds absolute |
| Identity breaker | 5 transient failures -> open 5 minutes |
| Magic link expiry | 7 days |
| Redemption lease | 30 seconds |
| Webhook delivery | Service Bus-backed, HTTPS outbound |
| Webhook retries | Immediate + 5 retries over ~24h |
| DLQ retention | Indefinite until operator replay/purge |
| External event granularity | Summary lifecycle events |
| PII posture | Opaque ids first; email only when required for delivery |

## Open Decisions Requiring Ratification

1. Exact retry schedule for the adopted "5 retries over 24h" webhook policy.
2. Whether campaign integrations can move from per-execution webhooks to tenant/campaign-level webhook registration in MVP.
3. Whether the `campaigns` container is the long-term home for `campaign_link` documents or whether a dedicated `campaign_links` container is preferred later for scale.
4. Whether `user.identity_denied` and `user.identity_error` are in the default event subscription set or opt-in only.
