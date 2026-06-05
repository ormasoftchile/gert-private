# SWA Multi-Tenant Routing

Use this pattern when designing white-label portals on Azure Static Web Apps.

## Pattern

- Resolve tenant from the incoming hostname, but verify it server-side before returning config or data.
- Cross-check hostname tenant with invitation/run tenant; fail closed on mismatch.
- Keep one shared SPA bundle; load tenant theme/copy at runtime through `/api/portal-context`.
- Inject only validated CSS variables, copy, and asset URLs. Never execute tenant-supplied scripts.
- Store tenant metadata in Cosmos DB and large brand/legal assets in Blob Storage.
- Bind customer hostnames explicitly; do not depend on wildcard domains for MVP.
- Treat SWA custom-domain capacity as a small per-app quota. Add Azure Front Door when WAF, many hostnames, wildcard strategy, custom TLS policy, or multiple origins are needed.

## Isolation Tiers

1. Shared SWA for standard tenants.
2. Shared SWA behind Front Door for tenants needing WAF/rate/domain controls.
3. Dedicated SWA/Front Door/App Service/data accounts for regulated, high-volume, sovereign, or independently released tenants.

## Frontend Rule

SWA hosts the shell. App Service/API owns identity validation, tenant authorization, run state, and audit progression.
