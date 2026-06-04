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

---

## Project Context

GERT is a governed, executable, traceable runbook engine (Go binary, local-first). It has a VSCode extension, TUI runner, and mobile SDKs. The team focus is building a web execution platform on Azure — white-labeled customer portals where end-users run runbooks as part of business processes (applications, authorizations, compliance flows). Azure infra only (no k8s). Key concerns: queue-driven execution, webhook delivery, tenant isolation, audit traceability.

**User:** ormasoftchile
**Session start:** 2026-06-03
