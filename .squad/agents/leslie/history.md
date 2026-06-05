# Leslie — Project History

## Learnings

### White-Label Deployment & UX Patterns (2026-06-03)

**Deployment Models:**
- Shared multi-tenant SaaS with subdomain/config branding is most operationally efficient but requires strong tenant isolation and config-driven UI
- Separate static deployments per customer give complete control but create per-tenant maintenance burden
- Iframe/web component embedding lets customers own the integration but adds auth complexity (cross-origin flows)
- Customer-hosted shell with component library is most flexible but highest support overhead

**Auth UX:**
- Entra External ID gives branded login but adds dependency on Azure; magic link is best for one-time consent flows
- Customer's own IdP (OIDC) removes vendor lock-in but requires upfront integration work
- Social login reduces friction for new users but may not match regulatory requirements (healthcare, finance)
- **Key insight:** End users don't know which IdP they're using — they care about "Can I use my existing credentials?"

**Runbook Execution UX:**
- Wizard (step-by-step) is most accessible for complex processes; users can't get lost
- Form-first (all inputs visible) is better for quick, simple processes; reduces cognitive load
- Chat-like feels natural but can hide dependencies and parallelizable steps
- Status dashboard (async, return later) is essential for multi-day/approval processes; otherwise users feel stranded

**Progress & Delivery:**
- Real-time SSE is premium UX but requires persistent connections and careful auth scoping
- Polling is good enough for most cases; users don't expect instant feedback on backend work
- Email notification is often the primary way users notice completion; should not be secondary
- In-app notifications alone miss users who close the browser; combine with email
- Process characteristics that drive choices:
- Quick consent (5 min): Form-first + email notification
- Multi-step flow (30+ min, same session): Wizard + real-time or polling + in-app notification
- Approval gate (hours/days): Status dashboard + email + async return link
- Regulated process (healthcare/finance): Customer's own IdP + full audit trail in UI

**2026-06-04:** Brainstorm session in progress (Leslie continuing with white-label frontend research). Barbara's architecture topology proposal, John's Azure constraints, Don's governance red lines, and David's webhook baseline have been merged to decisions.md and orchestrated. Awaiting Leslie's white-label research completion.

### White-Label Portal Hosting & UX Design (2026-06-04T12:28:39.293-04:00)

**Hosting decisions:**
- Recommend one shared Azure Static Web Apps Standard frontend for MVP, with tenant resolution by hostname and runtime config rather than one deployment per customer
- Use explicit per-tenant SWA custom domains (`customer.myservice.com` or customer-owned vanity domains); do not make wildcard domain support a hard dependency for MVP
- Use SWA edge delivery/CDN on day one; bring in Azure Front Door only for WAF, multi-region failover, or advanced traffic management
- Keep backend/API single-region for MVP; frontend assets are edge-distributed already

**Theming approach:**
- Themeing should be runtime-loaded through a `ThemeProvider` using CSS variables and semantic design tokens, not per-tenant builds
- Store mutable portal/theme metadata in Cosmos DB and brand assets (logos, fonts, legal copy packs) in Blob Storage
- Customer-configurable surface includes logo, colors, fonts, portal copy, footer/support links, and step-type helper copy
- Non-themeable surface includes GERT audit controls, accessibility baseline, error/safety semantics, and core step rendering behavior

**UX patterns:**
- Magic link flow should feel passwordless: click email link, brief secure validation, then immediate progression into the campaign
- If Entra External ID is used post-validation, it should feel like a short “Securing your session…” handoff, not a visible login ceremony
- Runbook steps should use non-optimistic advancement for auditable submissions: only move forward after server acknowledgement
- Resume is a first-class behavior: submitted progress survives server-side, same-device drafts can restore locally, and expired sessions resume via a fresh link rather than forced account creation
- Two-device behavior should allow only one active writer session to prevent double-submit confusion

### White-Label Frontend Foundations Close-Out (2026-06-04T17:14:51-07:00)

**Key constraints found:**
- Azure Static Web Apps Standard custom-domain capacity is a small per-app quota, not unlimited. Microsoft docs disagree between 5 and 6; design to 5 until John verifies the active subscription quota.
- Custom domains require explicit public DNS validation and binding. Use TXT validation for zero-downtime/Enterprise Grade Edge flows, CNAME/ALIAS for traffic, and Azure-managed certs unless Front Door/App Service is introduced for certificate control.
- Do not depend on wildcard SWA domains for MVP. Explicit hostname onboarding is cleaner for tenant mapping, offboarding, support, and audit.
- SWA should remain a static React shell: 500 MB-ish per environment, 15,000 files, 30 MB request limit, static `staticwebapp.config.json`, limited managed Functions. Large assets, tenant config, identity validation, run state, and audit belong outside SWA.

**Tiering recommendation:**
- Standard tenants: shared SWA + shared A6 App Service/data plane with strict `tenantId` enforcement.
- Higher-risk/public campaign tenants: shared SWA behind Azure Front Door for WAF, rate controls, TLS/domain scaling, and multiple-origin routing.
- Regulated/high-volume/sovereign tenants: dedicated SWA/Front Door/App Service/data resources when isolation, private networking, release cadence, or residency requires it.

**Open questions:**
- John must confirm the live SWA custom-domain quota in the target subscription because docs conflict.
- Product/team must decide whether enterprise tenants require BYO certificates or accept Azure-managed certs.
- Barbara/John need to define which tier gets Front Door from day one and what threshold triggers a dedicated tenant stack.

---

## Project Context

GERT is a governed, executable, traceable runbook engine (Go binary, local-first). It has a VSCode extension, TUI runner, and mobile SDKs. The team focus is building a web execution platform on Azure — white-labeled customer portals where end-users run runbooks as part of business processes (applications, authorizations, compliance flows). Azure infra only (no k8s). Key concerns: queue-driven execution, webhook delivery, tenant isolation, audit traceability.

**User:** ormasoftchile
**Session start:** 2026-06-03

## 2026-06-04 — Declaration/Consent UX Analysis

### Learnings

**Presentation Enforcement Requires Frontend-Only Tracking**
Declaration disclosure patterns (scroll-to-bottom, time minimums, section expansion, TTS) are not gate-driven. They're 100% client-side validation; gate fires only when enforcement requirements are met. No server-side gate evaluation needed. This unblocks fast iteration—presentation rules can change per runbook metadata without backend changes.

**Signature Requires New Gate Kind**
Current system lacks signature-specific IUserInputGate.Kind. Proposed "Signature" gate with sub-types (typed | drawn | cert | otp) to model full spectrum of signing patterns. Drawn signatures need pressure data capture (PointerEvents API + SVG paths). Critical decision: Should typed signatures be legal-valid, or drawn-only? Medical/legal should be drawn; informational can allow typed.

**Same-Session Witness Handoff is Distinct from Email Witness**
Different UX semantics:
- **Email-based** (different-session): Use existing IApprovalGate; witness gets email link + reviews asynchronously
- **Same-session** (quick & device-handed): Requires new WitnessLinking gate or compound gate pattern; uses QR + verbal code linking + real-time side-by-side confirmation
Current system doesn't model same-session handoff. IApprovalGate is sufficient for email workflow (low effort, high value); defer WitnessLinking gate pending customer demand.

**Attachments Should Use FileUpload Gate with Optional Server-Side OCR**
No new gate kind needed. FileUpload gate carries attachment metadata. Client-side UX decision: camera-first on mobile (native device camera, auto-edge-detection, perspective warp), file-picker on desktop. Server-side optional OCR validation (runbook metadata flag) validates extracted text against required fields (e.g., expiration date match). Post-signature attachment failures should allow "supplemental attachment" step (track attachment_phase metadata).

**Declaration Timeout Should Extend Beyond 15 Minutes**
Current 15-min user input timeout is too aggressive for declarations. Recommended extension: **60 minutes** for read-heavy flows. Rationale: Declarations require active reading; interruptions are common; 15 min fails too fast. Implementation: Add optional `timeoutSeconds` override to IUserInputGate (gate-level, not global). Keep 15 min for fast-turnaround fields (time-sensitive appointments, etc.).

**Save-Draft Pattern Requires Checkpoint Events, Not Gate Firing**
Forms can be resumed after interruption. Proposed pattern: User clicks "Continue Later" → client posts resumable_checkpoint → server saves form state + issues resume_token (JWT, 24h expiry). On resume: token validated, form state hydrated, user completes + re-signs. Implementation: New "CheckpointRequest" event type (separate from gate.NextAsync()). Can use server DB + optional client localStorage for redundancy. No new gate kind; existing Form gate carries the checkpoint state.

**White-Label Themeing is Feasible; "Powered by GERT" Audit Decision Pending**
Logo, color, font, signature style are themeable via Static Web Apps Standard + ThemeProvider component. Typography, color scheme, button styles all customizable. Non-themeable: error states (red, safety), accessibility floor (WCAG AA contrast non-negotiable), camera UI (native device, consistent). Pending decision: Does "Powered by GERT" watermark remain visible in all white-label declarations? Current assumption: Yes (audit/compliance), but needs team ratification.

**Receipt/Confirmation Should Issue QR-Verified PDF**
Post-signature receipt flow: Server generates confirmation_id, creates QR code (encodes confirmation_id + document hash + issuer_id). On-screen: confirmation ID + QR + download/email CTAs. PDF generation server-side; include embedded QR, optional digital signature (X.509), watermark. QR verification endpoint (/verify?qr=ABC123) returns public info (name, timestamp) only; no PII. Immediate implementation possible; low risk. Digital signature (enterprise tier) deferred.

### System Gaps Identified
1. IUserInputGate.Kind missing "Signature" variant (typed, drawn, cert, otp sub-types)
2. Same-session witness linking not modeled (WitnessLinking gate deferred; use IApprovalGate for now)
3. Declaration timeout hardcoded to 15 min (should extend to 60+ min; gate-level override needed)
4. Checkpoint event type missing (required for save-draft pattern)
5. Presentation enforcement rules not gate-driven (100% frontend; acceptable per Leslie's charter)
6. OCR validation for attachments not yet integrated with FileUpload gate (optional server logic)
7. "Powered by GERT" watermark visibility unresolved (audit vs white-label aesthetic—team decision needed)

### Design Decisions Requiring Team Ratification
1. Should "Powered by GERT" remain visible in all white-label declarations?
2. Is same-session witness (QR) immediate priority, or defer to Phase 2?
3. Is typed signature legal-valid for medical/legal flows?
4. Should save-draft be Phase 1, or defer pending customer demand?
5. Should declarations get 60-min timeout, or keep 15-min default?

### Immediate Next Steps for Frontend
- Implement DeclarationPresentation component (scroll-to-bottom + time enforcement)
- Create Signature gate kind + SignatureCanvas (drawn) + TypedSignature variants
- Extend IUserInputGate.timeoutSeconds for gate-level override
- Build ConfirmationScreen + QR receipt flow
- Integrate camera capture with FileUpload gate (auto-edge-detection)

### Reflection
Declaration flows are high-stakes UX (legal signatures, medical consent, financial liability). Small UX friction = abandonment; good UX = completion + regulatory compliance. Key insights: (1) Presentation enforcement is purely frontend—no gate logic needed; frees backend iteration. (2) Signature requires new gate kind to span typed/drawn/cert/otp spectrum. (3) Witness flows have two architectures (email async vs same-session real-time); email is low-effort starting point. (4) Attachments reuse FileUpload gate; optional OCR is value-add. (5) 60-min timeout is necessary for long docs; 15-min fails too fast. (6) Receipt/confirmation QR flow is immediate win for audit + customer trust. (7) White-label themeing is feasible except for audit watermark decision.

Confident we can ship Phase 1 (presentation + signature + 60-min timeout + receipt) in 2-3 sprints. Phase 2 (same-session witness + save-draft + OCR) pending customer demand + team ratification on audit decisions.

## Team Integration — White-Label Campaign Platform (2026-06-04)

**Cross-agent coordination:**
- **Barbara:** Campaign architecture drives portal topology; white-label platform needs shared SWA + Front Door with per-tenant domain binding; campaign layer entities (tenant, campaign, audience, invitation, contract artifact) are the data model Leslie's UX navigates
- **John:** Aspire local dev enables Leslie to test portal locally with emulated service bus and blob storage; dev auth bypass supports fast UX iteration
- **David:** Integration contracts define webhook/notification UX (when users receive completion emails); identity validation timeouts affect portal form submission UX
- **Scribe:** 7 portal UX ratification items merged to decisions.md for team review

**Key decisions:** Shared SWA MVP (dedicated SWA only for enterprise); explicit per-tenant custom domains; Static Web Apps built-in edge (Front Door later); Cosmos DB + Blob theme storage; magic-link-first auth (no password); resumable progress with single-writer concurrency; audit branding boundary decision pending.

