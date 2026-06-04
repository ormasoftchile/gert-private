# White-Label Contract Acceptance Campaign — Reference Architecture

**Author:** Barbara (Lead / Architect)  
**Date:** 2026-06-04T12:28:39.293-04:00  
**Status:** Reference design for scenario family  
**Architecture baseline:** A6 is adopted (`App Service + BackgroundService`, C#). This document specializes A6 for contract acceptance campaigns.

---

## Executive Position

Start from the failure mode, not the happy path:

1. **A magic link is not identity proof.** It is invitation possession proof only.
2. **Customer identity validation must be a server-to-server integration.** Do not call the customer endpoint from the browser.
3. **The contract acceptance record is a legal/audit artifact.** Blob JSONL trace and immutable contract versioning matter more than raw UX convenience.
4. **Shared white-label delivery is the default.** Per-tenant deployments are an exception for isolation, networking, or regulatory reasons.
5. **Do not let the portal invent a second execution model.** The portal collects user input; GERT runtime still owns step progression, audit, and completion.

**Recommended baseline for this scenario:**
- **White-label website:** shared **Azure Static Web Apps Standard** SPA, fronted by **Azure Front Door Standard/Premium** for per-tenant custom domains, WAF, and managed TLS.
- **Execution/API layer:** shared **Azure App Service P1v3** hosting ASP.NET Core APIs plus `BackgroundService` worker (A6).
- **Campaign state:** **Azure Cosmos DB** for mutable state; **Azure Blob Storage** for immutable JSONL trace and contract artifacts.
- **Queueing:** **Azure Service Bus Standard** with sessions for run dispatch and separate queue for webhook delivery.
- **Realtime UX:** **Azure SignalR Service** for live run updates; never authoritative.
- **Email:** customer-sent by default; platform-sent only if the customer delegates sender reputation, domain verification, and template ownership.
- **Identity:** invitation token + customer identity validation endpoint; optionally add OTP / federated login for high-assurance tenants.

---

## 1. Full Component Map

### 1.1 Logical architecture

```text
Customer Admin / Ops
        |
        v
+--------------------------+      +--------------------------------+
| Campaign Admin Portal    |----->| Campaign API / Tenant Config   |
| (internal/customer admin)|      | App Service (A6 shared)        |
+--------------------------+      +--------------------------------+
                                             |
                                             +--> Cosmos DB
                                             |    - tenants
                                             |    - campaigns
                                             |    - invitations
                                             |    - runs
                                             |    - user_inputs
                                             |    - webhooks
                                             |
                                             +--> Blob Storage
                                             |    - runbook versions
                                             |    - contract PDFs/HTML
                                             |    - JSONL traces
                                             |    - receipts
                                             |
                                             +--> Service Bus
                                             |    - campaign-send
                                             |    - runs (sessions)
                                             |    - webhooks
                                             |
                                             +--> Key Vault
                                             |    - customer API secrets
                                             |    - webhook secrets
                                             |    - email credentials
                                             |
                                             +--> Azure SignalR Service
                                             |
                                             +--> Customer identity API
                                             |
                                             +--> Customer webhook endpoint

End User
  |
  v
+---------------------------------------------------------------+
| Azure Front Door Std/Premium + WAF + managed TLS              |
| custom domains: contracts.customer.com, accept.customer.com   |
+---------------------------------------------------------------+
                              |
                              v
+---------------------------------------------------------------+
| Azure Static Web Apps Standard (shared SPA codebase)          |
| - tenant branding loaded at runtime                           |
| - same build, tenant-specific config                          |
+---------------------------------------------------------------+
                              |
                              v
+---------------------------------------------------------------+
| App Service P1v3 (shared A6 runtime/API)                      |
| - invitation redemption                                       |
| - portal session issuance                                     |
| - run creation / resume                                       |
| - BackgroundService GERT runtime                              |
+---------------------------------------------------------------+
```

### 1.2 Where the white-label website lives

**Default answer:** the white-label website lives in a **shared Azure Static Web Apps Standard** deployment, not one Static Web App per tenant.

Why:
- one SPA codebase,
- one release train,
- cheaper operationally,
- brand and copy can be data-driven,
- custom domains terminate at Front Door, not by cloning the frontend per customer.

**Use per-tenant website deployment only when one of these is true:**
- customer contract requires isolated deployment artifacts,
- tenant needs private network pathing that diverges from the shared platform,
- tenant requires independent release cadence,
- tenant requires dedicated WAF policy, analytics boundary, or sovereign region.

### 1.3 Shared platform vs per-tenant resources

| Resource | Default model | Notes |
|---|---|---|
| Front Door | Shared | Separate frontend hostnames and WAF rules per custom domain |
| Static Web App | Shared | One SPA, tenant config loaded by hostname |
| App Service A6 API/runtime | Shared by environment | Tenant isolation is logical unless premium isolation tier is purchased |
| Azure SignalR Service | Shared | Channel names and auth scoped by tenant/run |
| Service Bus namespace | Shared | Use tenant-scoped metadata and run/session IDs |
| Cosmos DB account | Shared | Partition by `tenantId` and `runId`; use strict data access layer |
| Blob Storage account | Shared by environment | Separate containers/prefixes per tenant; immutable trace policies |
| Key Vault | Shared or split by environment | Secrets namespaced per tenant; dedicated vault for premium tenants |
| Campaign definitions | Per tenant logical objects | Immutable version per campaign launch |
| Invitation records | Per tenant logical objects | One record per user invite |
| Branding config | Per tenant | Logo, palette, copy blocks, domain mapping |
| Customer API credentials | Per tenant | Prefer per-tenant secret/certificate |
| Webhook secret | Per tenant | Rotate independently |
| Custom domain | Per tenant | Customer-controlled DNS delegation/CNAME |
| Private connectivity to customer | Usually per tenant | VPN/private endpoint path is inherently tenant-specific |
| Dedicated App Service / data plane | Optional premium tier | Needed for stricter isolation, throughput, or networking |

### 1.4 Campaign management layer

This scenario needs a real campaign layer. Without it, you will reinvent state in ad hoc tables.

**Core entities:**

| Entity | Purpose | Example fields |
|---|---|---|
| `Tenant` | Customer company configuration | `tenantId`, `displayName`, `domains`, `branding`, `localePolicy`, `webhookConfig` |
| `IdentityIntegration` | How to validate the person | `mode`, `endpointUrl`, `authMode`, `timeoutMs`, `allowedAttempts`, `stepUpRequired` |
| `Campaign` | One outreach effort for one contract/version | `campaignId`, `tenantId`, `contractVersion`, `runbookVersion`, `launchWindow`, `inviteTtl`, `sendMode` |
| `AudienceMember` | One target user in the campaign | `externalUserId`, `email`, `locale`, `identityHints`, `status` |
| `Invitation` | One redeemable invitation | `invitationId`, `tokenHash`, `expiresAt`, `sentAt`, `consumedAt`, `deliveryStatus` |
| `Run` | One runbook execution instance | `runId`, `invitationId`, `status`, `currentStep`, `outcome`, `traceBlobUri` |
| `ContractArtifact` | Immutable document shown to the user | `artifactId`, `sha256`, `language`, `blobUri`, `effectiveDate` |
| `NotificationSubscription` | Customer callback settings | `endpoint`, `secretRef`, `eventTypes`, `retryPolicy` |

**Operational services around campaigns:**
- Campaign Admin UI for internal ops or customer administrators.
- Audience import/export job.
- Invitation generation service.
- Email delivery adapter.
- Identity verification adapter.
- Completion notification adapter.
- Reconciliation/reporting API.

### 1.5 Where the runbook definition lives

**Recommended pattern:**
- **Runbook source** authored as versioned YAML in the product repository or controlled artifact pipeline.
- **Published runbook artifact** stored in **Blob Storage** with immutable version ID.
- **Campaign record** in **Cosmos DB** stores the exact `runbookVersion` and `contractArtifactVersion` pinned for that campaign.

That gives you two things you need legally:
1. the exact workflow version used for this campaign, and
2. the exact contract content/hash shown to the user.

Do **not** let an active campaign point at “latest”. Pin versions.

### 1.6 Recommended deployment tiers

| Tier | Website | Compute/Data | Use when |
|---|---|---|---|
| Standard shared | Shared SWA + shared Front Door | Shared A6 App Service, shared Cosmos/Blob/Bus | Most customers |
| Premium isolated networking | Shared SPA codebase, tenant-specific custom domain | Shared A6 plus tenant-specific private connectivity/NAT/webhook secreting | Customer API lives behind network controls |
| Dedicated tenant stack | Dedicated SWA/Front Door/App Service/data accounts | Per-tenant infra | Regulated or high-volume customers |

---

## 2. Complete Flow — Step by Step

### 2.1 End-to-end sequence

```text
1. Customer onboarded
2. Tenant config + branding + domain + webhook + identity integration configured
3. Contract artifact and runbook version published
4. Campaign created and audience loaded
5. Invitation records + one-time tokens generated
6. Email sent (customer-owned by default, platform optional)
7. User clicks link -> landing page resolves campaign/invitation
8. User performs human gesture to redeem link
9. Portal collects identity challenge -> App Service calls customer identity endpoint
10. If verified, App Service creates portal session + run instance
11. GERT runtime executes contract acceptance runbook
12. User accepts or rejects
13. Completion written to trace + state store
14. Customer notified by signed webhook / reconciliation API
15. Campaign reporting updated until all invites terminal
```

### 2.2 Customer onboarding and campaign setup

1. **Tenant creation**
   - Create tenant record.
   - Assign tenant admin users.
   - Record region/data residency policy.
   - Decide shared vs dedicated deployment tier.

2. **Branding and domain setup**
   - Customer provides logo, colors, legal display name, support email/phone, default locale.
   - Customer creates DNS CNAME for `contracts.customer.com` (recommended) to Azure Front Door endpoint.
   - Managed TLS certificate issued/validated.

3. **Identity integration setup**
   - Customer provides endpoint contract and auth mechanism.
   - Connectivity tested from App Service path, not a developer laptop.
   - Retry/timeouts agreed.
   - Fail-open is forbidden. Identity endpoint errors must not silently pass users through.

4. **Notification setup**
   - Customer provides webhook endpoint and HMAC secret exchange method.
   - Optional pull API access for reconciliation.

5. **Contract artifact and runbook publication**
   - Contract PDF/HTML uploaded.
   - Hash computed and stored.
   - Runbook version published and pinned.

6. **Campaign definition**
   - Campaign name, launch window, invitation TTL, reminder policy, languages, identity proof rules, acceptance vs rejection notification rules.
   - Audience file/API loaded.

### 2.3 Email generation and sending

There are two viable models. Pick one explicitly.

#### Model A — Customer sends the emails (**recommended default**)

Why this is the default:
- customer already owns recipient list and consent basis,
- customer owns sender reputation,
- customer can align wording with their legal/compliance team,
- less platform exposure to bulk email deliverability problems.

**Platform responsibility in Model A:**
- generate invitation links,
- expose export/API for campaign audience with generated links,
- track redemption and completion states,
- optionally provide template variables.

**Customer responsibility in Model A:**
- email content,
- sender domain,
- opt-out/legal footer where required,
- reminders.

#### Model B — GERT sends the emails (optional service)

Use **Azure Communication Services Email** or **SendGrid** only if the customer delegates this.

Required prerequisites:
- verified sender domain,
- approved templates,
- SPF/DKIM/DMARC alignment,
- rate limits sized for campaign volume,
- bounce/complaint handling policy.

#### Example template variables

| Variable | Example |
|---|---|
| `{{customerName}}` | `Contoso Health` |
| `{{campaignName}}` | `2026 Service Agreement Acceptance` |
| `{{linkUrl}}` | `https://contracts.contoso.com/i/01J...?...` |
| `{{expiryDate}}` | `2026-06-11T12:28:39.293-04:00` |
| `{{supportEmail}}` | `support@contoso.com` |

### 2.4 Magic link design

**Do not use a naive JWT in the URL as the only control.** Use an **opaque random token**.

**Recommended link shape:**

```text
https://contracts.customer.com/i/{invitationId}?t={opaqueToken}
```

**Token design:**
- 256-bit random value.
- Store **hash only** in Cosmos DB.
- Bound to `tenantId`, `campaignId`, `audienceMemberId`, `invitationId`.
- Default TTL: **7 days**; configurable per campaign.
- Single-use for session issuance.
- Never contains raw PII.

**Critical anti-failure detail:** email security scanners will prefetch links.

So redemption must be **two-stage**:
1. **GET** renders a landing page and stores only a short-lived browser nonce.
2. **POST /invitations/{id}/redeem** with the opaque token and nonce performs the real redemption after a user click/tap.

That avoids burning invitations due to bot prefetch.

**Portal session after redemption:**
- On successful invitation validation, issue a short-lived portal session cookie/token (for example 15-30 minutes idle, 2 hours absolute).
- If the user abandons and returns with the same link before completion, allow re-entry if the invitation is still valid and the prior session is expired.
- After terminal acceptance/rejection, subsequent link opens must show terminal status, not create a second run.

### 2.5 Portal identity validation

**Baseline rule:** the browser submits challenge answers to GERT API; **App Service** performs the customer endpoint call server-to-server.

#### Example identity flow

1. User arrives with valid invitation.
2. Portal shows minimal challenge fields, for example:
   - employee number + date of birth,
   - document number + last name,
   - customer account number + OTP.
3. User submits form.
4. App Service calls customer identity endpoint.
5. Endpoint returns normalized match result.
6. If matched, App Service binds `customerSubjectId` to the run and lets the runbook start.

#### Example request/response contract

**Request from GERT to customer API:**

```json
{
  "campaignId": "camp_2026_contract_01",
  "invitationId": "inv_01J...",
  "externalUserId": "cust_user_4421",
  "challenge": {
    "employeeNumber": "E-100245",
    "dateOfBirth": "1989-06-21"
  },
  "requestId": "req_01J...",
  "timestamp": "2026-06-04T12:28:39.293-04:00"
}
```

**Response from customer API:**

```json
{
  "status": "matched",
  "customerSubjectId": "emp_99811",
  "displayName": "Jane Doe",
  "assuranceLevel": "knowledge_based",
  "correlationId": "cust-5f3a7c"
}
```

**Normalized outcomes:**
- `matched`
- `not_matched`
- `manual_review_required`
- `temporarily_unavailable`
- `invalid_request`

Do not let every tenant invent a different semantic contract. Normalize it.

### 2.6 What if identity validation fails?

**Recommended policy:**
- Allow **3 to 5 attempts** per invitation, configurable by tenant risk policy.
- Add short cool-down after repeated failures.
- After max attempts, mark invitation `locked_identity_failed` and require customer support/manual recovery.
- Trace every failed attempt with reason code.

**Failure branches:**
- `not_matched`: user can retry up to limit.
- `manual_review_required`: pause campaign completion and route to manual helpdesk workflow.
- `temporarily_unavailable`: do not burn attempts; allow retry later.
- `invalid_request`: platform/config bug; alert ops.

### 2.7 “Something else maybe?” — step-up assurance options

If the customer says “validate identity using our endpoint and something else maybe,” the realistic options are:

| Option | When to use | Comment |
|---|---|---|
| Invitation + customer endpoint only | Low/medium risk contract | Lowest friction |
| Invitation + customer endpoint + OTP | Contract has higher legal sensitivity | OTP can be customer-owned or platform-sent |
| Invitation + federated login (OIDC/SAML/Entra) | Customer has workforce/customer IdP | Strongest existing identity proof |
| Invitation + endpoint + document/biometric check | High-risk onboarding | Usually outside GERT scope; integrate specialist vendor |

My recommendation: **do not promise biometric/KYC inside GERT**. Keep that as an external integration seam.

### 2.8 Runbook design for contract acceptance

The runbook for this scenario is not arbitrary. It should be boring and auditable.

**Recommended step pattern:**

1. **Context step** — explain what this process is and why identity is required.
2. **Identity challenge step** — collect challenge fields or launch federated login.
3. **Contract presentation step** — show exact contract artifact version.
4. **Required acknowledgements** — checkboxes for mandatory statements.
5. **Optional data capture** — typed full name, job title, locale confirmation.
6. **Acceptance decision step** — accept or reject.
7. **Receipt step** — show confirmation/reference and delivery status.

#### Example runbook skeleton

```yaml
steps:
  - id: intro
    kind: message
    title: Review service agreement

  - id: identity
    kind: form
    purpose: identity-proof

  - id: contract_view
    kind: document
    artifact: contract-v2026-06

  - id: acknowledge
    kind: confirmation
    statements:
      - I confirm I have reviewed the agreement.
      - I am authorized to respond on behalf of myself.

  - id: acceptance
    kind: choice
    options: [accept, reject]

  - id: finalize
    kind: message
    title: Submission recorded
```

**Important scope boundary:** the portal renders the steps; GERT runtime still controls progression and audit.

### 2.9 User progression through steps

- Each step submission is written to run state and trace.
- The authoritative current step is stored in runtime/Cosmos, not in browser memory.
- SignalR pushes updates for responsiveness.
- On refresh or reconnect, portal asks `GET /runs/{id}` and `GET /runs/{id}/pending-input`.
- Step submissions use idempotency keys so resubmission from flaky mobile networks does not double-record.

### 2.10 Completion event and what gets stored

**Terminal outcomes:**
- `accepted`
- `rejected`
- `expired`
- `identity_failed`
- `cancelled`
- `manual_review`

**Minimum acceptance record:**

| Field | Why it matters |
|---|---|
| `tenantId`, `campaignId`, `runId`, `invitationId` | Traceability |
| `customerSubjectId` / `externalUserId` | Person linkage |
| `contractArtifactId` + `sha256` | Exact content shown |
| `runbookVersion` | Exact workflow version |
| `acceptedAt` / `rejectedAt` | Legal timestamp |
| `displayLanguage` | Evidence of what was shown |
| `identityAssuranceLevel` | How identity was checked |
| `ipAddress`, `userAgent` | Forensic evidence |
| `actorAssertions` | Required acknowledgement answers |
| `traceBlobUri` | Authoritative event history |
| `receiptId` | Customer-facing reference |

**Storage split:**
- **Blob JSONL trace:** authoritative event log.
- **Cosmos DB:** queryable run summary and campaign reporting state.
- **Blob receipt artifact:** optional PDF/JSON receipt for download and external sharing.

### 2.11 How the customer gets notified

**Primary mechanism:** signed webhook delivered asynchronously.

**Recommended payload example:**

```json
{
  "eventType": "campaign.contract.completed",
  "tenantId": "tenant_contoso",
  "campaignId": "camp_2026_contract_01",
  "runId": "run_01J...",
  "invitationId": "inv_01J...",
  "externalUserId": "cust_user_4421",
  "customerSubjectId": "emp_99811",
  "outcome": "accepted",
  "completedAt": "2026-06-04T12:28:39.293-04:00",
  "contractArtifactId": "contract-v2026-06",
  "contractSha256": "...",
  "traceUri": "https://storage/.../run_01J....jsonl"
}
```

**Delivery pattern:**
- enqueue webhook message,
- POST with HMAC signature,
- retry with exponential backoff,
- dead-letter after policy exhausted,
- expose replay tooling.

**Secondary mechanism:** reconciliation pull API.

This is non-negotiable. Webhooks fail. Customers need a way to query final state by campaign/user/run.

---

## 3. Network and Infrastructure Considerations

### 3.1 Must the customer identity endpoint be publicly reachable from Azure?

**From the platform’s point of view, yes: reachable from Azure.** Public internet is only one way to achieve that.

Three patterns:

| Pattern | Good for | Notes |
|---|---|---|
| Public HTTPS endpoint with IP allowlist + mTLS | Fastest onboarding | Usually enough for MVP |
| Private endpoint / VNet path via VPN or ExpressRoute | Customer intranet systems | More setup, cleaner security story |
| Customer-hosted API gateway in DMZ | Legacy internal systems | Often the practical compromise |

### 3.2 If the customer endpoint is private

Then somebody must fund and own the network seam. The usual options are:

1. **App Service VNet Integration** into a customer-connected VNet.
2. **Site-to-Site VPN** between Azure VNet and customer network.
3. **ExpressRoute** for larger enterprise customers.
4. Customer exposes a **DMZ/API gateway** that proxies to the internal identity system.

**Operational implication:** if customer requires outbound IP allowlisting, use **NAT Gateway** with VNet integration to give the App Service stable egress IPs. Do not promise “just allowlist the App Service” without checking outbound IP stability and scaling behavior.

### 3.3 CORS considerations

Best pattern: **avoid browser-to-customer API calls entirely**. Then CORS to the customer identity endpoint disappears.

For the portal itself:
- Prefer same-origin routing through Front Door (`contracts.customer.com` serving SPA and proxying API routes) so CORS is minimized.
- If SPA and API are on separate origins, allow only the tenant’s configured frontend origins.
- No wildcard CORS on authenticated endpoints.

### 3.4 Custom domain and DNS strategy

**Recommended:** `contracts.customer.com` or `accept.customer.com`.

Why this is better than `myservice.com/customer`:
- cleaner white-labeling,
- easier cookie isolation,
- clearer trust boundary for end users,
- simpler tenant-specific origin rules,
- better long-term path to dedicated stacks.

**DNS setup:**
- customer creates CNAME to Azure Front Door endpoint,
- validate domain ownership,
- issue managed certificate,
- map hostname to tenant config.

### 3.5 TLS requirements

- TLS 1.2+ minimum, preferably 1.3 where supported.
- Managed certificate at Front Door for portal hostname.
- TLS for all platform-to-customer calls.
- Prefer **mTLS** for customer identity endpoint if customer can support it.
- Rotate certificates/secrets without campaign downtime.

### 3.6 Firewall / NSG / private access notes

| Surface | Recommendation |
|---|---|
| Front Door | WAF enabled, rate limiting, bot protection |
| App Service inbound | Restrict direct access where possible; prefer Front Door origin lock-down |
| Storage/Cosmos/Service Bus/Key Vault | Private endpoints for higher-security tiers |
| Customer identity endpoint | IP allowlist and/or mTLS; explicit timeout and retry budget |
| Customer webhook endpoint | Same: IP allowlist or public HTTPS + HMAC verification |

### 3.7 Azure service list for this scenario

| Concern | Azure service |
|---|---|
| White-label SPA hosting | Azure Static Web Apps Standard |
| Global entry, WAF, custom domains, TLS | Azure Front Door Standard/Premium |
| API + worker runtime | Azure App Service P1v3 |
| Queue dispatch | Azure Service Bus Standard |
| Mutable state | Azure Cosmos DB |
| Audit trace + contract artifacts | Azure Blob Storage |
| Live UI updates | Azure SignalR Service |
| Secrets/certs | Azure Key Vault |
| Email (if platform-owned) | Azure Communication Services Email |
| Monitoring | Azure Monitor + Application Insights + Log Analytics |

---

## 4. Where This Can Fail — Exhaustive Failure Analysis

### 4.1 Failure table

| Failure point | What breaks | Required system behavior |
|---|---|---|
| Magic link expired | User cannot enter flow | Show branded expiry screen, allow resend/new invite path, no partial run creation |
| Magic link already used | Duplicate access attempt | Show terminal or resume state; never create second terminal run for same invitation |
| Email scanner prefetched link | Invite could be consumed accidentally | Two-stage redemption prevents consumption on GET |
| Identity endpoint down | User blocked before runbook | Return retry-later state, do not consume remaining attempts, alert tenant ops if threshold exceeded |
| Identity endpoint slow | Bad UX / timeout | Hard timeout, retry budget, user-facing “try again later”, record latency metrics |
| Identity validation response malformed | Integration contract drift | Treat as platform/customer integration failure, do not pass user through |
| Identity validation returns no match | Possible impersonation or user error | Increment attempt count, show generic retry message, avoid revealing sensitive data |
| User abandons mid-runbook | Run stays incomplete | Persist run state, allow resume until invitation/run expiry, send reminders based on campaign policy |
| User switches mobile to desktop | Session continuity issue | Rehydrate from authoritative run state after portal session renewal |
| Webhook to customer fails | Customer misses completion | Retry asynchronously, then DLQ, expose replay/reconciliation API |
| Duplicate completion submit | Same step posted twice | Idempotency key + step ETag returns prior result, no double webhook |
| Two browser tabs for same user | Competing submissions | First successful mutation wins, second gets stale-step conflict and refresh prompt |
| Multiple users from same company hitting same invite | Wrong person could try same run | Invitation bound to one audience member; customer identity proof must match that person |
| Cosmos DB write failure during step completion | Queryable state lags or conflicts | Do not lose authority: trace write remains decisive; resume/rebuild state from trace if needed |
| JSONL trace write failure | Audit gap risk | Block step progression immediately; fail-safe over fail-open |
| SignalR disconnect mid-step | User stops seeing live updates | Polling/resume API path restores view; SignalR is convenience only |
| Browser refresh mid-runbook | Local state lost | Reload from `GET /runs/{id}` and pending input endpoint |
| User uses multiple devices | Inconsistent UI state | Allow viewing from many devices, but serialize mutations per run/step |
| Service Bus duplicate delivery after lock loss | Run may reprocess | Session lock + run status/idempotent dispatch guard prevents second active executor |
| App Service instance crash | Active execution interrupted | Worker recovers from persisted run state/checkpoint; no audit loss beyond last durable trace line |
| Blob Storage transient outage | Cannot append trace | Suspend/block step until storage available or fail run with retryable infrastructure status |
| Contract artifact missing or wrong hash | Legal evidence compromised | Do not start run; campaign misconfiguration alert |
| Reminder email bounced | User never receives invite | Mark delivery status, expose reporting, allow alternate channel/resend |
| Customer webhook endpoint returns 200 but ignores payload | False sense of delivery | Recommend webhook idempotency key + reconciliation pull; delivery success != business ingestion success |
| Customer changes identity API contract mid-campaign | Validation breaks | Version the contract; do not auto-adapt; require explicit campaign integration version |
| Tenant branding missing/broken | Portal trust drops | Fall back to safe default theme, never blank page |
| WAF blocks legitimate traffic | Users cannot access portal | Observe per-tenant rules and override workflow |
| Portal session expires mid-flow | User forced out | Allow safe re-entry using invite + existing run state if still eligible |
| User rejects contract | Customer expects completion semantics | `rejected` is terminal and must notify/report distinctly from technical failure |

### 4.2 Specific recovery rules by storage layer

| Layer | Authority | Recovery rule |
|---|---|---|
| Blob JSONL trace | Highest | If trace append failed, the step did not durably happen |
| Cosmos DB run summary | Secondary | Rebuild from trace if disagreement exists |
| SignalR connection state | None | Drop and rehydrate |
| Browser memory | None | Drop and rehydrate |

### 4.3 Concurrency rules you need now, not later

1. **One invitation -> one terminal contract outcome.**
2. **One run -> one active executor.**
3. **One step input row -> optimistic concurrency enforced.**
4. **All customer notifications -> idempotent by `eventId` / `runId` + outcome.**

If you do not define these rules now, you will debug ghosts later.

---

## 5. Security and Compliance Considerations

### 5.1 Magic link security

Required controls:
- 256-bit entropy.
- Hash at rest.
- TTL enforced server-side.
- Single-use for session issuance.
- Invitation is bound to audience member/campaign.
- Redemption only after human interaction.
- Rate limit redemption and challenge endpoints.
- Never log raw token.

**Do not bind magic link directly to IP or device fingerprint as the primary rule.** Too fragile for real users. Bind the resulting portal session to the verified run context instead.

### 5.2 PII handling

Principle: **collect the least possible data**.

- Store only the challenge values needed to prove what was checked, and even then only when legally/operationally justified.
- Prefer storing normalized match outcome + subject IDs over raw identity documents.
- Encrypt sensitive fields at rest.
- Separate customer secrets in Key Vault.
- Do not echo customer identity response bodies into browser logs.

### 5.3 Audit trail for legal enforceability

For contract acceptance, the audit record should answer:
- Who was invited?
- How was identity checked?
- Exactly what document/version/hash was shown?
- In what language?
- What acknowledgements did the user make?
- What was the final decision?
- When, from where, and via what client?
- Can the event stream be shown without ambiguity?

**Recommended evidence set:**
- immutable contract artifact hash,
- trace JSONL,
- acceptance/rejection timestamp,
- invitation ID,
- verified subject ID,
- locale displayed,
- IP/user agent,
- acknowledgement values,
- receipt artifact.

### 5.4 Tenant isolation

The answer to “Can user A from Company X ever see user B’s data?” must be **no, by design**.

Controls:
- every record includes `tenantId`,
- data access layer enforces tenant filter centrally,
- partition strategy supports tenant isolation,
- signed URLs/artifact access are tenant-scoped and time-limited,
- hostname-to-tenant mapping verified server-side,
- webhook secrets and API credentials isolated per tenant,
- admin roles separated per tenant.

### 5.5 GDPR / residency / retention

Questions that must be answered before go-live:
- In which Azure region is each tenant’s data stored?
- Are traces replicated outside that jurisdiction?
- What retention period applies to acceptance records?
- What is erasable vs non-erasable due to legal retention obligations?
- Who is controller vs processor for campaign PII?

**Practical recommendation:**
- region-pin tenants where contractually required,
- keep acceptance evidence for the customer-defined legal retention period,
- pseudonymize reporting tables where possible,
- document lawful basis and DPA responsibilities.

### 5.6 Scope boundary on compliance-heavy features

GERT should own:
- contract presentation workflow,
- user acknowledgements,
- acceptance/rejection capture,
- immutable trace emission,
- completion notification.

GERT should **not** natively own unless explicitly expanded:
- KYC/AML,
- biometric verification,
- qualified electronic signature authority,
- sanctions screening,
- document authenticity analysis.

These stay as integration seams.

---

## 6. The White-Label Design

### 6.1 Design principles

This portal is for ordinary people, not operators.

Use:
- one decision per screen,
- minimal fields,
- strong trust cues (brand, domain, support contact),
- obvious progress,
- plain-language reasons for each step,
- mobile-first layout.

### 6.2 Key screens

#### 1. Landing screen (from email link)

Purpose:
- confirm brand trust,
- explain what the invitation is,
- confirm expiry window,
- obtain human click before redemption.

Contents:
- customer logo and service name,
- short explanation: “You have been invited to review and respond to a service agreement,”
- estimated time,
- support contact,
- primary CTA: **Continue**.

#### 2. Identity validation screen

Purpose:
- prove the person is the intended audience member.

Contents:
- 1-2 challenge fields only,
- explanation of why identity information is needed,
- privacy notice link,
- retry-safe error messaging,
- optional OTP field if step-up required.

#### 3. Runbook/contract steps screen

Purpose:
- walk the user through the contract process without overwhelming them.

Layout:
- header with customer brand,
- progress indicator,
- main card with current step,
- sticky footer with support and legal links,
- explicit primary action and secondary back/cancel where allowed.

For the contract display step:
- embedded PDF/HTML or structured document view,
- document version/date,
- download option,
- continue button gated behind acknowledgement where legally appropriate.

#### 4. Completion screen

Variants:
- accepted,
- rejected,
- pending manual review,
- expired.

Contents:
- clear status,
- reference/receipt ID,
- timestamp,
- optional download receipt,
- what happens next.

### 6.3 What the customer can brand

| Customizable | Examples |
|---|---|
| Domain | `contracts.customer.com` |
| Logo and favicon | Company brand assets |
| Primary/secondary colors | Within accessibility guardrails |
| Intro/outro copy | Welcome, completion, support messaging |
| Support contact | Email, phone, help URL |
| Email template copy | Subject/body/footer |
| Locale defaults | English, Spanish, etc. |

### 6.4 What the customer cannot customize

| Not customizable | Why |
|---|---|
| Core GERT step semantics | Runtime consistency and audit integrity |
| Mandatory audit capture fields | Legal evidence requirement |
| Trace emission behavior | Platform integrity |
| Security controls (token TTL enforcement, rate limits, HMAC) | Not negotiable per tenant |
| Notification event schema core fields | Downstream contract stability |
| Outcome model (`accepted`, `rejected`, etc.) | Reporting consistency |

### 6.5 Progressive disclosure for non-technical users

Do not dump the whole process on screen 1.

Recommended pattern:
1. explain the overall purpose,
2. collect identity proof,
3. show one contractual unit at a time,
4. ask for explicit acknowledgement,
5. record final decision,
6. show receipt.

Users should never wonder:
- where am I,
- why am I being asked this,
- what happens if I stop now,
- whether my response was recorded.

---

## 7. What Needs to Be in Place Before Going Live

### 7.1 Go-live checklist

| Area | Requirement | Owner |
|---|---|---|
| Tenant onboarding | Tenant record, admins, region policy, support contacts created | Platform + customer |
| Branding | Logo, colors, copy, locale assets approved | Customer |
| Domain | DNS CNAME configured, domain validated, TLS issued | Customer + platform |
| Identity API contract | Request/response schema versioned and signed off | Customer + platform |
| Identity connectivity | Public/mTLS or private connectivity tested from Azure path | Customer + platform |
| Secrets | API creds, certificates, webhook secrets in Key Vault | Platform |
| Contract artifact | Final document uploaded, hashed, approved, language variants ready | Customer + platform |
| Runbook | Exact runbook version published and pinned to campaign | Platform |
| Audience | Recipient list/API loaded, deduplicated, status validated | Customer + platform |
| Email model | Explicit decision: customer-send or platform-send | Customer + platform |
| Email deliverability | SPF/DKIM/DMARC and bounce handling ready if platform sends | Customer + platform |
| Reminder policy | Expiry, reminders, resend rules approved | Customer |
| Notification | Webhook endpoint tested, HMAC verified, reconciliation API agreed | Customer + platform |
| Legal review | Acceptance evidence fields and retention signed off | Customer legal + platform |
| Monitoring | Dashboards, alerts, DLQ monitoring, trace-write alerts enabled | Platform |
| Support runbook | Identity failure, resend, manual review, webhook replay procedures documented | Platform + customer |
| Load test | Invitation redemption, identity burst, webhook throughput tested | Platform |
| Security review | Threat model, WAF, token handling, rate limits reviewed | Platform |

### 7.2 Pre-launch uncomfortable questions

These are the questions to force before anyone says “ship it”:

1. Is the customer asking for **invitation possession** or real **identity assurance**?
2. Who legally owns the outgoing email content and consent basis?
3. What is the fallback when the customer identity API is down during a live campaign?
4. Does rejection count as successful completion in downstream systems?
5. Must the acceptance record be admissible as legal evidence in a dispute?
6. What retention period applies to traces and contract artifacts?
7. Can a user resume on another device without re-proving identity?
8. What happens when a customer changes contract text after invitations are already sent?
9. Does this tenant require shared platform or dedicated stack?
10. Who handles manual review and user support when identity does not match?

---

## Recommended Decisions for This Scenario

1. **Shared white-label frontend by default.** Do not create one frontend deployment per tenant unless there is a real isolation requirement.
2. **Invitation token is not enough.** Require customer identity verification before the contract step opens.
3. **Server-to-server customer validation only.** No browser-direct calls into customer identity infrastructure.
4. **Customer-sent email by default.** Platform-sent email is an opt-in managed service.
5. **Signed webhook plus pull reconciliation.** Never rely on webhook delivery alone.
6. **Pinned runbook version and pinned contract artifact hash per campaign.** No “latest” pointers.

That is the seam: the white-label portal owns user experience, the campaign layer owns invitation/state, and the A6 runtime still owns execution truth.
