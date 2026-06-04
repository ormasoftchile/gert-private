# White-Label Portal UX & Hosting Design

**Author:** Leslie (Frontend Dev)  
**Date:** 2026-06-04T12:28:39.293-04:00  
**Status:** Proposed design reference  
**Requested by:** ormasoftchile (Gerardo)

---

## Executive Summary

This design recommends a **shared Azure Static Web Apps (SWA) Standard frontend** for MVP, with **tenant resolution by hostname** and **runtime-loaded theme/config**.

Default posture:
- **One shared SWA** for the portal shell, not one build per tenant
- **One branded hostname per tenant** (for example `acme.myservice.com` or `portal.acmehealth.com`)
- **SWA built-in edge/CDN** for day one; **Azure Front Door only when we need WAF, active-active routing, or advanced traffic control**
- **Tenant branding/config in Cosmos DB + Blob Storage**, loaded at runtime by hostname
- **Single-region backend for MVP**, globally cached frontend assets via SWA edge
- **No password/account creation required** for the base magic-link flow

This keeps branding flexible, reduces deployment overhead, and preserves a low-friction customer journey.

---

## 1. Where the White-Label Portal Lives (Hosting Architecture)

### 1.1 Recommendation: Shared SWA with Tenant Routing

**Recommendation:** Start with **one shared SWA Standard application** serving all customer portals, with tenant detection based on the incoming hostname.

Why this is the best MVP tradeoff:
- Rebranding is **runtime config**, not a code deployment
- One React app shell is easier to secure, test, and evolve
- Unlimited-ish white-label growth is operationally simpler than one SWA per customer
- Shared shell still supports tenant isolation at the **config/data/auth** level
- Dedicated SWAs remain available later for enterprise contracts that require them

**Do not** default to one SWA per tenant for MVP. That creates unnecessary CI/CD sprawl, certificate lifecycle overhead, and version drift.

### 1.2 Hosting Pattern

```text
Email magic link
   ↓
https://acme.myservice.com/c/{campaign_token}
   ↓
Azure Static Web Apps Standard
   • Serves React/TypeScript SPA
   • Resolves tenant by Host header
   • Loads tenant theme/config at runtime
   ↓
App Service API
   • Validates magic link
   • Creates secure session
   • Resolves campaign/run state
   • Streams progress via SignalR (fallback: polling)
   ↓
Cosmos DB + Blob Storage
   • Portal config / tenant settings / mutable run state
   • Logos, font files, legal assets, uploaded files, audit artifacts
```

### 1.3 Custom Domain Support

#### Customer subdomains under our domain
Example: `acme.myservice.com`

Recommended model:
- Create an SWA custom domain entry per tenant hostname
- DNS uses **CNAME** from `acme.myservice.com` to the SWA default hostname
- TLS is managed by Azure Static Web Apps

#### Customer-owned vanity domains
Example: `portal.acmehealth.com`

Recommended model:
- Customer creates the DNS record we specify
- We bind the hostname to SWA as a custom domain
- Tenant config maps that hostname to the correct tenant record

#### Wildcard domains
**Recommendation:** Do **not** make wildcard hostnames a hard dependency for MVP.

Reasoning:
- Explicit hostname binding is easier to reason about for support and tenant offboarding
- It keeps certificate and tenant ownership clean
- Most customers only need one or two branded hostnames

### 1.4 CDN Strategy

**Day one:** Use the **built-in Azure Static Web Apps edge delivery/CDN behavior**.

That is enough for:
- Static asset caching
- Global low-latency delivery of JS/CSS/images
- Standard branded portal workloads

**Introduce Azure Front Door later only when needed**, for example:
- WAF requirement
- Advanced bot protection
- Multi-region failover/origin routing
- Blue/green or canary across multiple SWAs
- Centralized wildcard/apex domain strategy across many brands

**Decision:** SWA edge first, Front Door only on clear need.

### 1.5 Per-Tenant Config Storage

**Recommendation:**
- **Cosmos DB** stores mutable per-tenant portal config metadata
- **Blob Storage** stores larger assets referenced by config

Use Cosmos DB for:
- tenant id
- company name
- domain bindings
- theme tokens
- portal copy overrides
- feature flags
- support contact details
- auth/identity policy flags

Use Blob Storage for:
- logo files
- favicon
- optional webfont CSS/files
- long-form legal copy packs
- downloadable completion PDFs/receipts

This matches the platform split already used elsewhere: **Cosmos DB for mutable operational state, Blob for assets/artifacts**.

### 1.6 Deployment Model

**Recommendation:** One shared frontend deployment; **tenant branding is runtime configuration**.

That means:
- React bundle is deployed once
- Tenant onboarding creates/configures records, not a new build
- Theme updates do **not** require redeploying the SPA
- Emergency UX fixes roll out once for every tenant

**Exception path:** dedicated SWA only for enterprise/regulatory customers that explicitly buy isolation or custom extension points.

### 1.7 Multi-Region Strategy

**Recommendation:**
- **Frontend assets:** effectively geo-distributed from day one via SWA edge delivery
- **Backend/API + run state:** single region for MVP
- **Multi-region active-active:** only when latency, residency, or uptime requirements justify the complexity

This avoids premature multi-region complexity while still giving end users fast static asset delivery globally.

---

## 2. The Theming System

### 2.1 What a Customer Can Configure

Customers can configure:
- Logo URL
- Favicon URL
- Company name
- Portal title
- Primary color
- Secondary color
- Accent/surface/background colors
- Font family (from approved/supported list or hosted asset)
- Footer links
- Support email / support phone
- Welcome copy
- Completion copy
- Custom helper text per step type
- Optional hero illustration/image
- Whether to show customer legal/privacy links
- Locale defaults

Recommended configurable copy buckets:
- `magicLinkLoading`
- `identityChecking`
- `campaignIntro`
- `stepTypes.information`
- `stepTypes.choice`
- `stepTypes.text`
- `stepTypes.confirmation`
- `stepTypes.fileUpload`
- `stepTypes.form`
- `completion`
- `errors`

### 2.2 What Cannot Be Customized

These remain controlled by the platform:
- GERT audit/governance controls
- Step renderer behavior and supported step types
- Accessibility baseline (WCAG contrast floor, focus states, minimum sizing)
- Safety/error semantics (error red, warning semantics, disabled states)
- Session/security messaging structure
- Trace/event/audit metadata
- File upload security constraints
- Required legal/audit footer elements if ratified by the team

In plain terms: customers can change the **brand skin and copy**, but not the **governed behavior**.

### 2.3 How the Theme Is Applied

**Recommendation:** Runtime-loaded config + CSS variables, with React semantic tokens.

Implementation pattern:
1. App boots on tenant hostname
2. Frontend requests `/api/portal-context` with hostname + route context
3. API returns tenant config JSON
4. Frontend writes CSS variables to `:root` before app hydration finishes
5. Tailwind/utility classes reference semantic tokens like:
   - `--color-primary`
   - `--color-secondary`
   - `--color-surface`
   - `--font-body`
   - `--font-heading`
6. React components use semantic classes, not raw tenant values

**Important:** Do not build one Tailwind bundle per tenant. Keep one semantic design system and inject tenant values at runtime.

### 2.4 Theme Configuration JSON Schema

```json
{
  "$schema": "https://gert.dev/schemas/portal-theme-v1.json",
  "tenantId": "acme-health",
  "version": "2026-06-04T12:28:39.293-04:00",
  "hostnames": [
    "acme.myservice.com",
    "portal.acmehealth.com"
  ],
  "branding": {
    "companyName": "Acme Health",
    "portalTitle": "Acme Health Secure Tasks",
    "logoUrl": "https://assets.example.com/acme/logo.svg",
    "faviconUrl": "https://assets.example.com/acme/favicon.ico"
  },
  "palette": {
    "primary": "#0057B8",
    "secondary": "#00A3A3",
    "accent": "#F4B400",
    "surface": "#FFFFFF",
    "background": "#F7F9FC",
    "text": "#1F2937"
  },
  "typography": {
    "bodyFontFamily": "Inter, system-ui, sans-serif",
    "headingFontFamily": "Inter, system-ui, sans-serif",
    "fontCssUrl": "https://assets.example.com/acme/fonts.css",
    "baseFontScale": 1.0
  },
  "copy": {
    "magicLinkLoading": "Verifying your secure link…",
    "identityChecking": "Confirming your identity with Acme Health…",
    "campaignIntro": {
      "titlePrefix": "You've been invited to complete",
      "estimatedTimeLabel": "Estimated time",
      "whatYouNeedLabel": "What you'll need"
    },
    "stepTypes": {
      "information": {
        "continueLabel": "Continue"
      },
      "choice": {
        "continueLabel": "Save and continue"
      },
      "text": {
        "placeholder": "Type your answer"
      },
      "confirmation": {
        "confirmLabel": "Confirm and continue"
      },
      "fileUpload": {
        "uploadLabel": "Upload file"
      },
      "form": {
        "submitLabel": "Save form"
      }
    },
    "completion": {
      "title": "You're all set",
      "body": "We've recorded your response and notified Acme Health."
    },
    "errors": {
      "generic": "We couldn't complete that request. Please try again.",
      "expiredLink": "This secure link has expired.",
      "usedLink": "This secure link has already been used."
    }
  },
  "footer": {
    "links": [
      {
        "label": "Privacy",
        "url": "https://acmehealth.com/privacy"
      },
      {
        "label": "Support",
        "url": "mailto:support@acmehealth.com"
      }
    ]
  },
  "behavior": {
    "showPoweredByGert": true,
    "allowResume": true,
    "supportEmail": "support@acmehealth.com",
    "localeDefault": "en-US"
  }
}
```

### 2.5 Validation Rules

Required:
- `tenantId`
- at least one hostname
- `branding.companyName`
- `branding.portalTitle`
- `branding.logoUrl`
- `palette.primary`
- `palette.surface`
- `palette.text`
- `behavior.allowResume`

Validated by platform:
- contrast ratios
- allowed font sources
- link safety
- image size/type limits
- copy length limits for key layouts

### 2.6 How a New Customer Sets Up Their Theme

**Recommendation:**
- **Phase 1:** internal onboarding workflow or tenant-setup API used by ops/admins
- **Phase 2:** customer-facing admin UI with preview/publish workflow

Preferred MVP flow:
1. Ops creates tenant record
2. Ops uploads logo/font assets
3. Ops configures hostname(s)
4. Ops saves theme JSON through an internal admin surface/API
5. Customer previews portal on staging hostname
6. Ops/publisher marks config version as live

Do **not** make customers edit files in source control for standard onboarding.

---

## 3. Screen Designs — Full User Journey

### Shared Layout Principles

- Single-column, focused task layout
- Strong brand presence in header, but the task card stays clean and readable
- Progressive disclosure: only show what is needed for the current moment
- Sticky footer CTA on mobile
- Consistent support/help affordance
- Progress visible, but not intimidating

Generic shell:

```text
┌─────────────────────────────────────────────────────────────┐
│ [Logo]                              Need help?             │
├─────────────────────────────────────────────────────────────┤
│ Campaign title / context                                     │
│ [Progress bar] Step N of M                                   │
│                                                               │
│ ┌───────────────────────────────────────────────────────────┐ │
│ │ Current step card                                         │ │
│ │ Title                                                     │ │
│ │ Description / instructions                                │ │
│ │ Input control(s)                                          │ │
│ │ Validation / helper text                                  │ │
│ └───────────────────────────────────────────────────────────┘ │
│                                                               │
│ [Back]                                  [Save and continue]  │
├─────────────────────────────────────────────────────────────┤
│ Privacy • Support • Terms • Powered by GERT?                │
└─────────────────────────────────────────────────────────────┘
```

### 3.1 Screen 1: Magic Link Landing

#### URL Structure

**Recommended public URL:**
- `https://acme.myservice.com/c/{campaign_token}`

Notes:
- Token should be opaque, short-lived, and single-use for session bootstrap
- After validation, the app should replace the visible URL with a session/run route such as `/run/{runId}` to reduce accidental token reuse and browser-history leakage

#### Loading / Validation State

What the user sees:

```text
┌───────────────────────────────────────────────┐
│ [Acme logo]                                   │
│                                               │
│ Verifying your secure link…                   │
│ This usually takes a few seconds.             │
│                                               │
│ [spinner]                                     │
│                                               │
│ Need help? support@acmehealth.com             │
└───────────────────────────────────────────────┘
```

Behavior:
- Validate token immediately on load
- Resolve tenant and campaign from hostname + token
- Exchange token for secure session
- Transition automatically to next screen when valid

#### Expired Token

Message:
- Title: `This link has expired`
- Body: `For your security, this invitation link is no longer active.`
- CTAs:
  - `Send me a new link`
  - `Contact support`

#### Already Used Token

Message:
- Title: `This link has already been used`
- If same-browser session still exists: show `Resume your task`
- Otherwise: show `Request a new secure link`

#### Invalid Token

Message:
- Title: `We couldn't verify this link`
- Body: `The link may be incomplete or no longer valid.`
- CTAs:
  - `Request a new link`
  - `Contact support`

#### Technical Error

Message:
- Title: `We couldn't connect right now`
- Body: `Please try again in a moment.`
- CTA: `Try again`

### 3.2 Screen 2: Identity Verification (When Required)

#### Base Recommendation

Most customers should experience identity verification as **transparent/automatic** after the magic link click.

That feels like:

```text
┌───────────────────────────────────────────────┐
│ Confirming your identity…                     │
│                                               │
│ We're securely checking your details with     │
│ Acme Health before we begin.                  │
│                                               │
│ [spinner]                                     │
└───────────────────────────────────────────────┘
```

The user should not be forced through a visible login unless the customer requires a stronger identity proofing step.

#### When Step-Up Is Required

If the customer policy requires stronger proofing:
- show a clear interstitial
- explain why extra verification is needed
- use a single CTA, for example `Continue to secure verification`

#### Failure State

If identity check fails:
- Title: `We couldn't verify your identity`
- Body: explain the next best action in plain language
- CTAs:
  - `Try again`
  - `Send a new link`
  - `Contact support`

Avoid blamey language. The user should feel blocked by policy, not accused of fraud.

### 3.3 Screen 3: Runbook Introduction / Campaign Welcome

#### Content

Show:
- Campaign name
- Company name
- Short purpose statement
- Estimated time
- What the user will need
- Privacy or data-use summary
- Primary CTA to begin

Suggested copy pattern:
- `You've been invited to complete [Campaign Name] for [Company Name].`
- `Estimated time: 5–10 minutes`
- `You may need: a photo ID, your policy number, and a document upload`

#### Wireframe

```text
┌────────────────────────────────────────────────────┐
│ [Logo]                                             │
│                                                    │
│ You've been invited to complete                    │
│ Coverage Confirmation                              │
│ for Acme Health                                    │
│                                                    │
│ Why you're here                                    │
│ We need a few details before your appointment.     │
│                                                    │
│ Estimated time: 8 minutes                          │
│ You'll need: ID, insurance card                    │
│                                                    │
│ [Begin]                                            │
└────────────────────────────────────────────────────┘
```

### 3.4 Screen 4: Runbook Step Execution

#### Step Frame

```text
┌────────────────────────────────────────────────────┐
│ Coverage Confirmation                              │
│ Step 2 of 6                                        │
│ [████████░░░░░░░░]                                 │
│                                                    │
│ Tell us how you'd like to be contacted             │
│ Choose one option below.                           │
│                                                    │
│ ( ) Email                                          │
│ ( ) SMS                                            │
│ ( ) Phone call                                     │
│                                                    │
│ [Back]                          [Save and continue]│
└────────────────────────────────────────────────────┘
```

#### Progress Indicator

Use:
- step count (`Step N of M`)
- progress bar
- current step title

Do not show a dense sidebar tree on mobile-first flows. Keep the user focused on the current action.

#### Step Type Designs

##### A. Information Display
Use for read-only instructions, disclosures, and context.

UI:
- title
- rich text content
- optional callout box
- CTA: `Continue`

No optimistic audit assumptions are needed; continue is immediate unless the step has explicit acknowledgement rules.

##### B. Choice
Use radio cards, segmented buttons, or stacked options depending on count.

UI:
- question title
- helper text
- one-choice or multi-choice control
- inline validation if required
- CTA: `Save and continue`

##### C. Text Input
Use for short or long free-form answers.

UI:
- label
- helper text
- text field or textarea
- validation hints under the field
- CTA: `Save and continue`

##### D. Confirmation
Use for yes/no, attestations, and user acknowledgements.

UI:
- statement block
- confirmation checkbox or yes/no action
- optional “I understand” copy
- CTA: `Confirm and continue`

##### E. File Upload
Use a camera-first pattern on mobile and file picker on desktop.

UI:
- upload zone/button
- allowed file types and size text
- preview of selected file
- remove/replace action
- CTA: `Upload and continue`

Mobile behavior:
- offer camera capture first when appropriate
- keep the rest of the page stable after returning from camera/gallery

##### F. Form
Use for structured multi-field input.

UI:
- grouped fields with subheadings
- inline validation per field
- summary error block at top on submit failure
- CTA: `Save form`

Progressive disclosure inside forms:
- only reveal dependent fields after the parent answer is known
- avoid overwhelming first paint

#### Submit / Continue Interaction

**Recommendation:** advance only after server acknowledgement for any persisted or auditable step.

Interaction pattern:
1. Client validates locally
2. CTA enters loading state (`Saving your answer…`)
3. Inputs lock to prevent double submit
4. API confirms submission
5. UI advances to next step or shows backend-driven next state

Use **non-optimistic step advancement** for auditable actions. The user should not be shown the next step until the previous answer is actually accepted.

#### Error State on Submit

Show inline, not as a blank page.

Pattern:
- keep the user's entered value in place
- show clear error copy
- allow retry
- if network-related, explain that nothing new was saved until confirmation is received

#### Can the User Go Back?

**Recommendation:**
- Yes for previously completed informational/input steps **until an irreversible step is committed**
- No once a step creates an external side effect or legally meaningful submission that should not be silently edited

Rules:
- Allow back navigation for normal choice/text/form steps when metadata says the step is editable
- Disable back after signed/confirmed declaration steps if changing prior context would invalidate the recorded trace
- If back is disabled, explain why: `This step has already been submitted and can't be changed here.`

#### Mobile Responsiveness

Required behaviors:
- single-column layout
- sticky bottom CTA bar
- large tap targets
- fixed help link in overflow/menu, not crowding the header
- progress collapses to compact bar + step count
- upload and camera flows stay within native device expectations

### 3.5 Screen 5: Completion

#### Success State

What the user sees:
- success title
- short confirmation summary
- confirmation/reference number
- what happens next
- optional download receipt / email receipt
- optional link back to customer portal

#### Wireframe

```text
┌────────────────────────────────────────────────────┐
│ [Success icon]                                     │
│ You're all set                                     │
│                                                    │
│ We've recorded your response and notified          │
│ Acme Health.                                       │
│                                                    │
│ Confirmation number                                │
│ ACM-20260604-1042                                  │
│                                                    │
│ Next steps                                         │
│ • Acme Health will review your submission          │
│ • You'll receive an email if anything else is needed│
│                                                    │
│ [Download receipt]   [Return to Acme portal]       │
└────────────────────────────────────────────────────┘
```

#### Customer Company Notification

The completion UI should state that the company has been notified **after** the backend has queued the notification event.

Do not block the success screen on downstream webhook delivery completion. From the user's point of view:
- their task is complete
- the handoff to the customer company is in motion

#### Abandonment Recovery

Recommended model:
- submitted progress survives on the server
- refresh/reconnect resumes at the current pending step
- unsaved draft field values can be restored from local browser storage on the same device
- if the secure session is gone, the user resumes via a fresh magic link, not by creating a password account

---

## 4. Authentication UX

### 4.1 Does the Magic Link Require an Account or Password?

**Recommendation:** No. The base magic-link runbook flow should **not** require account creation or password login.

Why:
- lower abandonment
- clearer one-time-task mental model
- better fit for invitation-based campaigns
- avoids forcing casual users into portal-account management they do not want

### 4.2 If Entra External ID Is Used After Link Validation, What Does It Feel Like?

To the user, it should feel like a **brief secure-session handoff**, not a login ceremony.

Good UX:
- short branded interstitial: `Securing your session…`
- maybe a redirect flash
- then immediate progression into the campaign

Bad UX:
- dropping the user onto a generic sign-in page after they already clicked a trusted email link
- asking them to invent a password for a one-time task

### 4.3 Should Users Return Later with a Real Login to See Their Completion Record?

**Recommendation:** optional, not default.

Default experience:
- one-time completion flow
- receipt/reference number
- email copy if needed

Optional enterprise/regulated experience:
- a separate authenticated records portal
- customer-controlled retention and viewing policy
- only enabled when the customer actually needs longitudinal records access

This keeps the campaign flow simple while leaving room for a richer customer portal later.

---

## 5. Error States and Edge Cases

| Condition | What the user sees | System behavior |
|---|---|---|
| Network offline mid-runbook | Persistent offline banner, disabled submit, reassurance that unsaved answers stay on device until connection returns | Retry automatically on reconnect; restore unsaved local draft |
| Session expired mid-runbook | Modal/screen: `Your secure session has expired` with `Resume securely` CTA | Attempt silent refresh if possible; otherwise request new magic link |
| Browser refresh | Same shell reloads and returns user to current pending step | Server-fetched progress resumes; submitted steps persist |
| Mobile app switch / background | Reconnecting state for a few seconds, then normal flow | Rejoin SignalR or fall back to polling |
| Two devices simultaneously | Banner: `This task is open on another device` with takeover option or read-only notice | One active writer session at a time; takeover invalidates the older writer session |
| API timeout on submit | Inline error with retry, entered data preserved | No step advancement until confirmed save |
| File upload too large/invalid | Inline validation and replacement action | Client blocks before submit when possible |
| Customer identity check unavailable | Plain-language interruption screen with retry/support options | Do not start the runbook until identity state is known |

### 5.1 Offline Mid-Runbook

Use a top banner and keep the user anchored in place.

Copy example:
- `You're offline. Reconnect to keep going.`
- `Your latest unsent changes will stay on this device until you're back online.`

### 5.2 Session Expiry Mid-Runbook

Use a focused interruption screen, not a silent failure.

Copy example:
- `For your security, this session has expired.`
- `Resume securely to continue where you left off.`

### 5.3 Browser Refresh

**Recommendation:** yes, progress should survive.

Mechanics:
- completed step submissions are already on the server
- current pending step is rehydrated from API
- unsaved form values restore from local storage only on the same device/browser

### 5.4 Mobile App Switching

Design for interruption as normal behavior.

When the user returns:
- show `Reconnecting…` briefly
- restore them to the same step
- if the session expired in the background, shift to the secure resume state

### 5.5 Two Devices Simultaneously

**Recommendation:** single active writer session.

UX pattern:
- second device can detect the run state
- if first session is active, second device sees a conflict banner
- user may choose `Continue on this device`
- old session becomes read-only and is told that the task moved elsewhere

This avoids double submission confusion.

---

## 6. Implementation Notes

### 6.1 SWA Routing Config for Magic Link Paths

The SPA should own portal routes; the API stays under `/api`.

Example `staticwebapp.config.json` pattern:

```json
{
  "navigationFallback": {
    "rewrite": "/index.html",
    "exclude": ["/api/*", "/assets/*", "/*.css", "/*.js", "/*.png", "/*.svg"]
  },
  "routes": [
    { "route": "/c/*", "rewrite": "/index.html" },
    { "route": "/run/*", "rewrite": "/index.html" },
    { "route": "/complete/*", "rewrite": "/index.html" }
  ],
  "responseOverrides": {
    "404": {
      "rewrite": "/index.html"
    }
  }
}
```

Recommendation:
- accept magic links on `/c/*`
- exchange token immediately
- replace route with `/run/{runId}` after validation

### 6.2 How the Frontend Receives Theme Config at Runtime

Recommended boot sequence:
1. Read hostname and path
2. Call `/api/portal-context`
3. Receive:
   - tenant info
   - theme tokens
   - campaign summary (when appropriate)
   - feature flags
   - auth/session state
4. Apply CSS variables immediately
5. Render the React shell

To reduce flash-of-default-theme:
- load a tiny boot script before the main app mounts
- keep a minimal neutral fallback palette for the first paint
- cache config by hostname with ETag/short TTL

### 6.3 Real-Time Feedback: SignalR / SSE

**Recommendation:** use **SignalR as the primary real-time channel** for A6 to match the adopted user-input gate direction.

Use SignalR for:
- step submission acknowledgements
- pending-step updates
- reconnect/resume events
- completion state changes

Fallbacks:
- polling for restricted networks or reconnect recovery
- SSE only if we later want a simpler read-only event stream for specific progress views

For the main runbook UX, one real-time channel is better than mixing interaction semantics.

### 6.4 Reusable Components vs Step-Specific Components

#### Reusable shell components
- `PortalBootstrap`
- `ThemeProvider`
- `PortalLayout`
- `HeaderBrand`
- `ProgressHeader`
- `StepFrame`
- `StickyActionBar`
- `ErrorPanel`
- `OfflineBanner`
- `SessionExpiredDialog`
- `CompletionCard`
- `SupportLinks`

#### Step-specific renderers
- `InformationStep`
- `ChoiceStep`
- `TextStep`
- `ConfirmationStep`
- `FileUploadStep`
- `FormStep`

#### Cross-cutting utilities
- `usePortalContext()`
- `useRunProgress()`
- `useSessionState()`
- `useDraftPersistence()`
- `useRealtimeChannel()`
- `validateThemeContract()`

### 6.5 UX Rules Worth Preserving in Implementation

- Replace magic-link URLs quickly after token exchange
- Do not optimistically advance auditable steps before acknowledgement
- Keep support/help available on every screen
- Use runtime theming, not per-tenant builds
- Prefer automatic/silent identity handling over visible login ceremony
- Treat interruption and resume as standard behavior, not an exception

---

## Final Recommended Decisions

1. **MVP frontend topology:** one shared SWA Standard with hostname-based tenant resolution
2. **Branding model:** runtime-loaded theme/config, not per-tenant frontend deployments
3. **Domain model:** explicit custom domains per tenant hostname; no wildcard dependency for MVP
4. **CDN/edge:** SWA edge now, Front Door only when requirements justify it
5. **Auth UX:** magic link first, no password/account creation for base campaign flow
6. **Real-time UX:** SignalR primary, polling fallback
7. **Resume UX:** server-saved progress + same-device draft restore + fresh-link resume when session expires

This gives us a portal customers can brand deeply without making end users think about the platform underneath.
