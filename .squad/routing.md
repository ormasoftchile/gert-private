# Work Routing

How to decide who handles what.

## Routing Table

| Work Type | Route To | Examples |
|-----------|----------|----------|
| Platform architecture, Azure system design, component boundaries | Barbara | "How should we structure the Azure services?", "Design the execution layer" |
| Azure infra, Service Bus, Functions, Container Apps, Entra ID, Bicep | John | "Set up the queue topology", "Configure tenant auth", "Write infra-as-code" |
| Runbook execution service, GERT runtime integration, backend APIs, worker | Don | "Build the execution worker", "Design the API for submitting runs", "SDK contracts" |
| Runtime migration parallel streams (GIS engine, GCP engine), Go backend work paired with Don | Ken | "Implement the GIS interpolation engine", "Build the GCP capture engine", "Pair with Don on multi-stream phases" |
| Web UI, white-label portal, auth UX, customer-facing frontend | Leslie | "Build the customer portal", "Design runbook step UI", "White-label theming" |
| Webhooks, queue consumers, integration events, idempotency, DLQ | David | "Design webhook delivery", "Build the event consumer", "Handle poison messages" |
| Code review | Barbara | Review PRs, enforce API contracts, check quality |
| Scope & priorities | Barbara | What to build next, trade-offs, architectural decisions |
| Session logging | Scribe | Automatic — never needs routing |

## Issue Routing

| Label | Action | Who |
|-------|--------|-----|
| `squad` | Triage: analyze issue, assign `squad:{member}` label | Lead |
| `squad:{name}` | Pick up issue and complete the work | Named member |

### How Issue Assignment Works

1. When a GitHub issue gets the `squad` label, the **Lead** triages it — analyzing content, assigning the right `squad:{member}` label, and commenting with triage notes.
2. When a `squad:{member}` label is applied, that member picks up the issue in their next session.
3. Members can reassign by removing their label and adding another member's label.
4. The `squad` label is the "inbox" — untriaged issues waiting for Lead review.

## Rules

1. **Eager by default** — spawn all agents who could usefully start work, including anticipatory downstream work.
2. **Scribe always runs** after substantial work, always as `mode: "background"`. Never blocks.
3. **Quick facts → coordinator answers directly.** Don't spawn an agent for "what port does the server run on?"
4. **When two agents could handle it**, pick the one whose domain is the primary concern.
5. **"Team, ..." → fan-out.** Spawn all relevant agents in parallel as `mode: "background"`.
6. **Anticipate downstream work.** If a feature is being built, spawn the tester to write test cases from requirements simultaneously.
7. **Issue-labeled work** — when a `squad:{member}` label is applied to an issue, route to that member. The Lead handles all `squad` (base label) triage.
