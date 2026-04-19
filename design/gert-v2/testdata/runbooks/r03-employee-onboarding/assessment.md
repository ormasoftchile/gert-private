# Assessment: New Employee Onboarding

**Completeness:** 8/10 (80%)  
**Fidelity:** 7/10 (70%)  
**Verdict:** PASS WITH NOTES

## Gaps Found

| Gap ID | Class | Severity | Description |
|--------|-------|----------|-------------|
| G2-001 | G2 | HIGH | Timeout field lacks business day support — schema uses wall-clock duration strings only; prose specifies "2 business days", "5 business days", etc. |
| G2-002 | G2 | MEDIUM | Field types lack dropdown/autocomplete — only text, multiline, file, image, url supported; department and location dropdowns reduced to text + hint |
| G4-001 | G4 | HIGH | No self-service task pattern — step 6 (employee self-service training) forced into approval pattern with `roles: [employee]`, which is semantically incorrect |
| G2-003 | G2 | MEDIUM | `on_timeout: approve` (auto-approve on timeout, step 8) not defined in schema spec; workaround uses `on_timeout: skip` |

## Translation Notes

1. **Step 1 (Collect Info):** Used `type: collector` with 7 fields. The prose mentions dropdown
   and autocomplete field types, but the schema only supports basic types. Used `type: text`
   with hints describing the expected input format.

2. **Business Day Timeouts:** The prose specifies "2 business days", "5 business days", etc.
   The schema's `timeout:` field uses duration strings ("30m", "2h"). Written as wall-clock
   equivalents (e.g., "48h" for 2 business days) — this is technically invalid per calendar
   semantics but valid per schema syntax. Marked with `# GAP:` comments in the YAML.

3. **Step 4 (Parallel Provisioning):** Used `type: parallel` with 5 branches (3 automated tool
   calls + 2 manual collector steps). Correctly models heterogeneous parallel tasks.

4. **Step 6 (Security Training):** The prose says "assignee: employee (self-service)". Used
   `roles: [employee]` in the approvals field. This is awkward — it's not really an "approval"
   in the traditional sense. The schema doesn't distinguish self-service tasks from approval
   gates. Marked with `# GAP:` comment.

5. **Step 7 (Access Provisioning):** Used `type: branch` with 4 conditional arms based on
   department. Correctly models conditional logic based on collected data.

6. **Step 8 (Manager Confirm Access):** The prose says "timeout: 1 business day → auto-approve".
   Used `on_timeout: skip` as closest available option. `on_timeout: approve` is not defined in
   the schema spec.

7. **Step 10 (Dual Attestation):** Used `approvals.min: 2` with `roles: [hr-director, it-security-officer]`.
   Correctly models M-of-N approval (both must approve).

## Recommended Schema Improvements

- Add calendar-aware timeout support: `timeout: {value: 2, unit: business_days, calendar: us_federal_holidays}`
- Add `type: task` step for self-service assignments (assignee completes task, not approves)
- Add `on_timeout: auto_approve` option for approval gates
- Extend collector field types: `dropdown`, `autocomplete`, `radio`, `checkbox`
