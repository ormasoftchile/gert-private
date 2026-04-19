# Assessment: Financial Approval for Large Purchase

**Completeness:** 8/10 (80%)
**Fidelity:** 8/10 (80%)
**Verdict:** PASS WITH NOTES

## Gaps Fixed (this revision)

| Gap ID | Class | Severity | Resolution |
|--------|-------|----------|------------|
| G2-002 | G2 | ~~HIGH~~ | **FIXED** — All 6 approval steps now use `timeout_business_days` + `timezone: "America/New_York"` instead of approximate wall-clock hours |
| G2-003 | G2 | ~~HIGH~~ | **FIXED** — Board approval (step 7b) now uses `approvals.mode: quorum` + `pool: [board-member-1 … board-member-5]` + `required: 3` |

## Remaining Gaps

| Gap ID | Class | Severity | Description |
|--------|-------|----------|-------------|
| G2-001 | G2 | CRITICAL | Dynamic approver resolution not supported — step 2 requires looking up manager from HR system; `approvals.roles:` is a static string list; cannot use template expression or provider query |
| G2-004 | G2 | MEDIUM | No dropdown field type for budget line item and department fields |
| G2-005 | G2 | MEDIUM | Dynamic approver at each tier (director/VP from org chart) falls back to static hardcoded role names |

## Translation Notes

1. **Step 2 (Manager Approval):** The prose says approver = "requester's manager (lookup from
   HR system)". This is a CRITICAL gap — the schema cannot resolve approvers dynamically.
   Workaround: hardcoded `roles: [manager]` which assumes a static role name rather than a
   specific person resolved from HR data. This means ALL managers can approve, not just the
   requester's manager.

2. **Step 4 (Approval Tier Decision):** Used `type: branch` with numeric range conditions.
   Go template comparisons use string comparison, which may fail for numeric values. A proper
   `gt`, `lt` comparison on parsed numbers is needed.

3. **Step 5-7 (Tier Branches):** Director/VP approvers are hardcoded role names rather than
   dynamic lookups from org chart. In a real multi-tenant scenario, this would require separate
   runbooks per department.

4. **Step 7b (Board Approval):** Now correctly uses `mode: quorum` with `pool` of 5 named
   board-member roles and `required: 3`. This enforces exactly 3-of-5 semantics with a
   14-business-day timeout.

5. **Business Day Timeouts:** All 6 approval steps now use `timeout_business_days` with
   `timezone: "America/New_York"`. Escalation will fire after the correct number of working
   days, not at arbitrary wall-clock intervals on weekends.

6. **Step 9 (Contract Execution):** Used `type: collector` with `type: file` fields for PDF
   uploads. File capture works correctly.

## Why PASS WITH NOTES (previously FAIL)

GAP-2 (business-day timeouts) and GAP-3 (quorum approval) are now resolved by the new
`timeout_business_days`/`timezone`/`business_calendar` fields and `approvals.mode: quorum`.

This runbook remains PASS WITH NOTES (not full PASS) because GAP-2-001 — dynamic approver
resolution from HR system — is still a CRITICAL unresolved gap that degrades the approval
chain's correctness in a real multi-tenant deployment.

## Recommended Schema Improvements (remaining)

- Add `roles_from:` field with provider-based resolution: `roles_from: {provider: hr-system, query: manager_for_email, params: {email: "{{ .requester_email }}"}}`
- Add dropdown/select field type for `collector` fields
