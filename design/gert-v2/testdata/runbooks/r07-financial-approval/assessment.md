# Assessment: Financial Approval for Large Purchase

**Completeness:** 7/10 (70%)  
**Fidelity:** 6/10 (60%)  
**Verdict:** FAIL

## Gaps Found

| Gap ID | Class | Severity | Description |
|--------|-------|----------|-------------|
| G2-001 | G2 | CRITICAL | Dynamic approver resolution not supported — step 2 requires looking up manager from HR system; `approvals.roles:` is a static string list; cannot use template expression or provider query |
| G2-002 | G2 | HIGH | Business-day timeouts not supported — all 6 approval steps use business days; expressed as approximate wall-clock hours with accuracy loss |
| G2-003 | G2 | HIGH | Quorum approval (M-of-N) — step 7b requires 3 of 5 board members; `approvals.min: 3` with `roles: [board-members]` works as a workaround but doesn't enforce exactly 5 eligible approvers |
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

4. **Step 7b (Board Approval):** Used `approvals.min: 3` with `roles: [board-member]`. This
   correctly enforces 3 approvals but doesn't restrict to exactly 5 eligible board members.
   Any 3 users with the `board-member` role can approve.

5. **Business Day Timeouts:** All timeouts approximated as wall-clock hours. This means
   escalation would trigger at 2am on a Sunday rather than after 2 business days. Marked with
   `# GAP:` comments throughout.

6. **Step 9 (Contract Execution):** Used `type: collector` with `type: file` fields for PDF
   uploads. File capture works correctly; this is one of the functional areas of the schema.

## Why FAIL

This runbook FAILS because:
1. The core workflow depends on dynamic approver resolution (manager from HR, VP from org chart)
   which is fundamentally not expressible in the current schema
2. Business-day timeouts affect all 6 approval steps, making SLA enforcement incorrect
3. These are not edge cases — they are the core business logic of a financial approval workflow

## Recommended Schema Improvements

- Add `roles_from:` field with provider-based resolution: `roles_from: {provider: hr-system, query: manager_for_email, params: {email: "{{ .requester_email }}"}}`
- Add calendar-aware timeout: `timeout: {value: 2, unit: business_days}`
- Add `approvers_pool:` with exact eligible set for quorum approval
