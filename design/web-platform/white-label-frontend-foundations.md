# White-Label Frontend Foundations

**Author:** Leslie (Frontend Dev)  
**Date:** 2026-06-04T17:14:51-07:00  
**Status:** Consolidated close-out for original web platform brainstorm

This closes the frontend foundation thread left open after Barbara's topology, John's Azure stack, Don's governance, and David's webhook decisions. It builds on `design/web-platform/white-label-portal-ux.md` and `design/web-platform/white-label-campaign-scenario.md`; it does not restate the theming, magic-link, or campaign UX decisions already captured there.

---

## 1. Custom Domain Strategy

### Hard constraints

- Azure Static Web Apps (SWA) Standard is production-capable but not an unlimited domain platform. Microsoft's current public docs are inconsistent: the hosting-plan page says **5 custom domains per app**, while the quotas page says **6 custom domains per app**. Treat the safe design limit as **5 tenant hostnames per shared SWA** unless John verifies the active quota in the target subscription.
- Custom domains must be explicitly added and publicly DNS-resolvable for validation and managed certificate renewal.
- New Enterprise Grade Edge custom domain validation requires the TXT token method; zero-downtime migration also uses `_dnsauth...` TXT records before traffic moves.
- Normal subdomain routing uses DNS CNAME to the SWA default hostname. Apex/root domains need ALIAS/ANAME/CNAME flattening or an A record fallback, but A records lose some global distribution advantages and should not be the default.
- SWA automatically provisions and renews free TLS certificates for bound custom domains. SWA is not the right place to demand tenant-managed certificate lifecycle or advanced certificate policy; use Azure Front Door or App Service if BYO certificate control becomes contractual.
- Do not rely on wildcard custom-domain binding for MVP. Onboard each hostname explicitly so ownership, offboarding, support, and tenant mapping stay clear.

### Recommended pattern

1. **Default hostname shape:** `portal.{customer}.com` or `contracts.{customer}.com` for customer-owned domains; `{customer}.gert.example` only for lower-touch tenants.
2. **Onboarding flow:**
   - Create tenant record with expected hostname.
   - Ask customer to add TXT validation record.
   - Bind custom domain to the SWA or Front Door profile.
   - Ask customer to add CNAME/ALIAS traffic record.
   - Wait for managed certificate issuance.
   - Mark hostname active in tenant config.
3. **MVP layering:** use SWA direct custom domains only while tenant count is low and WAF is not required.
4. **Front Door layering:** introduce Azure Front Door Standard/Premium when any of these becomes true:
   - more than the SWA custom-domain quota is needed on one shared portal,
   - WAF/bot/rate-limit policy is required,
   - centralized TLS/custom certificate policy is required,
   - wildcard or many-hostname onboarding becomes important,
   - multiple SWAs/dedicated tenant stacks need one global entry point.

### Open questions

- John should confirm the live SWA custom-domain limit in the target subscription because Azure docs disagree between 5 and 6.
- Do enterprise contracts require BYO certificates, or are Azure-managed certs acceptable?
- Which tenants require Front Door from day one because of WAF or certificate governance?

### Links to locked decisions

- Aligns with Leslie's decision in `.squad/decisions.md`: **Domain Strategy = Explicit Per-Tenant Custom Domains**.
- Refines John's `.squad/decisions.md` Azure stack note that SWA Standard is the frontend recommendation; the earlier "unlimited custom domains" assumption must be corrected to a small per-app quota.
- Aligns with Barbara's campaign reference in `design/web-platform/white-label-campaign-scenario.md`: Front Door is appropriate for campaign portals that need WAF, managed TLS at the global edge, or many custom domains.

---

## 2. Multi-Tenant Isolation Patterns

### Hard constraints

- The browser is not an isolation boundary. Hostname-to-tenant resolution must be verified server-side before returning tenant config, run state, or assets.
- Every data record and artifact lookup must carry `tenantId`; the portal may display theme and copy, but the API must enforce tenant filtering centrally.
- Shared SWA means shared frontend code, shared cache behavior, and one release train. It does **not** mean shared tenant data access.
- Runtime-loaded theming must be constrained: CSS variables, vetted asset URLs, copy length limits, font source allowlists, WCAG contrast checks, and no tenant-provided script execution.
- Noisy or compromised tenants can still affect shared surfaces: traffic spikes, WAF false positives, asset abuse, support incidents, and release rollback pressure.

### Recommended pattern

Use tiered isolation:

| Tier | Frontend | Data/API posture | Who gets it |
|---|---|---|---|
| Shared standard | Shared SWA, explicit hostname, runtime theme config | Shared A6 App Service/Cosmos/Blob with strict `tenantId` enforcement | Most MVP and standard customers |
| Shared plus edge controls | Shared SWA behind shared Front Door with per-host WAF/rate rules | Shared A6 plus stronger per-tenant secrets, egress controls, monitoring | Public campaigns, higher traffic, customer API allowlisting |
| Dedicated tenant stack | Dedicated SWA/Front Door/App Service/data accounts where needed | Per-tenant infra and release controls | Regulated, sovereign, high-volume, or contractually isolated tenants |

Routing rule: **Host header resolves tenant; invitation/run token resolves campaign; API cross-checks both.** If hostname tenant and token tenant disagree, fail closed.

Theming rule: load `/api/portal-context` from the resolved hostname, inject only semantic CSS variables/config, and never ship per-tenant bundles unless the tenant is on a dedicated stack.

Blast-radius rule: start shared; move tenants to dedicated when they require independent release cadence, private networking, region/data residency isolation, dedicated WAF policy, or sustained volume that can harm other tenants.

### Open questions

- What exact traffic or incident threshold moves a tenant from shared Front Door to dedicated stack?
- Does the product need a self-serve tenant onboarding path, or can ops-mediated onboarding remain acceptable for MVP?
- Should premium tenants get separate Key Vault and storage accounts before they get separate SWA?

### Links to locked decisions

- Aligns with Barbara's A6/A8 segmentation: shared A6 is MVP; dedicated/premium segmentation preserves isolation when customer needs justify it.
- Aligns with Barbara's campaign scenario: shared white-label delivery is the default; per-tenant deployments are exceptions for isolation, networking, regulatory, or release-cadence reasons.
- Aligns with Leslie's prior theming/auth decisions: runtime config in Cosmos/Blob, magic-link-first UX, resumable progress, and single active writer.

---

## 3. Static Web Apps Deployment Constraints

### Hard constraints

SWA Standard is a good SPA host, not a general frontend compute platform.

Current constraints to design around:

- **Storage/app size:** Standard allows about 500 MB per environment, 2 GB total across environments, and 15,000 files. Large tenant-specific asset packs belong in Blob Storage/CDN, not inside the SPA bundle.
- **Custom domains:** small per-app quota; do not model one shared SWA as the permanent home for dozens of vanity hostnames without Front Door or sharding.
- **Preview environments:** limited; do not promise preview-per-tenant.
- **Request size:** 30 MB. Large uploads should use backend-issued Blob upload flows, not SWA-managed API payloads.
- **Routes/config:** `staticwebapp.config.json` handles routing/auth/header rules, but it is static, first-match, and not a tenant policy engine. There is no useful published route-count budget to build product policy around; keep the file small and generic. Client-side routes are not secured by SWA route rules; APIs must enforce authorization.
- **Environment variables/app settings:** useful for app/API configuration, with no tenant-friendly published count to rely on. Managed Functions reserve many prefixes and are not a tenant config store. Tenant config belongs in Cosmos DB and Blob Storage.
- **Auth providers:** built-in SWA auth supports GitHub and Microsoft Entra ID by default; Standard supports custom provider registrations. Our default portal flow remains invitation/magic-link plus server-side identity validation. Entra External ID/B2C is a step-up UX, not the only entry path.
- **API runtime:** managed Functions are HTTP-only, Consumption-hosted, no managed identity, no Key Vault references, no Durable Functions. Public docs list managed runtime support across Node.js 12/14/16/18/20 preview, .NET Core 3.1/.NET 6/7/8, and Python 3.8/3.9/3.10, subject to Azure Functions language support policy. Bring-your-own Functions is Standard-only and still maps through `/api`. Our adopted A6 App Service API remains the main backend.
- **Region availability/residency:** SWA serves static assets globally, but resource creation and backend/data residency are separate concerns. If a tenant requires strict region-pinned frontend origin/control that SWA cannot provide in the target geography, use App Service or Container Apps for that tenant's frontend tier.
- **Enterprise-grade edge:** helpful, but it adds cost and cannot be combined with SWA private endpoint. Private frontend access and Front Door edge features must be chosen deliberately.

### Recommended pattern

Keep SWA Standard as the **shared React shell** for MVP:

- static SPA only,
- runtime tenant config,
- App Service API for authoritative state/auth/run progression,
- Blob Storage for large brand/legal assets,
- SignalR or polling for progress,
- Front Door added only when edge policy/domain scale requires it.

Move off SWA for the frontend tier when we need:

- SSR or server-side personalization at request time,
- tenant-specific server middleware at the web edge,
- large dynamic file generation from the frontend host,
- dozens of vanity domains on one shared entry point without sharding,
- BYO certificate or strict TLS policy not supported by SWA direct bindings,
- private-only frontend access combined with edge/WAF needs,
- region/data residency guarantees SWA cannot meet,
- deployment artifacts larger than SWA limits,
- API/runtime requirements that make the SWA-managed Functions model tempting but insufficient.

When pushed off SWA, prefer **App Service** for the frontend if we want simple Node/SSR hosting aligned with A6 operations; prefer **Container Apps** only when packaged edge/frontend services need container-specific behavior or scale-to-zero economics are worth the extra operations.

### Open questions

- Does any planned customer portal require SSR, or is SPA plus API enough for the first paid campaigns?
- Will customer-uploaded brand assets ever need to be bundled for offline/static delivery, or can Blob/CDN remain authoritative?
- Which Azure regions must be supported for residency before the first regulated tenant signs?

### Links to locked decisions

- Supports John's decision to use SWA Standard for the frontend and App Service P1v3 for A6 compute.
- Supports Barbara's A6 architecture: the portal renders and collects input; App Service/GERT runtime owns progression and audit.
- Respects Don's governance line: the frontend never becomes a second execution engine and never writes authoritative trace.

---

## Close-Out Position

For customers, the experience should feel like a trusted branded portal at `portal.customer.com`. Under the curtain, MVP should be one shared SWA shell, explicit hostname onboarding, server-enforced tenant isolation, runtime theme config, and a clear premium path to Front Door or dedicated stacks when domain scale, WAF, certificate control, networking, or regulation requires it.
