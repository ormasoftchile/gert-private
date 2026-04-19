# Assessment: Security Breach Containment and Forensics

**Completeness:** 8/10 (80%)  
**Fidelity:** 7/10 (70%)  
**Verdict:** PASS WITH NOTES

## Gaps Found

| Gap ID | Class | Severity | Description |
|--------|-------|----------|-------------|
| G3-001 | G3 | CRITICAL | Cross-branch parallelism not expressible — forensics (step 8) should run concurrently with containment branches (4, 5, 6); schema's `type: parallel` cannot span branch arms |
| G3-002 | G3 | HIGH | Mid-runbook cross-branch goto not supported — step 6b (Medium Containment) needs to jump to "High Containment" branch; workaround inlines duplicate steps |
| G2-001 | G2 | HIGH | `choice` step default on timeout — prose specifies "default to High on 10-min timeout"; workaround uses `on_timeout: skip` which proceeds with pre-set default variable |
| G2-002 | G2 | MEDIUM | No dropdown field type — remediation planning responsible_team field reduced to text + hint |

## Translation Notes

1. **Step 2 (Triage Severity):** Used `type: choice` correctly for human severity selection.
   The "default to High on timeout" is approximated via `on_timeout: skip` with the default
   variable value set to `high`. This is a partial workaround — the skip behavior with default
   depends on runtime behavior not fully specified in the schema.

2. **Step 3-7 (Branching):** Used `type: branch` with four conditions. Critical containment
   includes nested parallel steps for isolation, credential revocation, and forensic snapshots.
   This correctly models nested parallel within a branch.

3. **Cross-Branch Parallelism (step 8):** Prose says forensics runs IN PARALLEL WITH
   containment. This is the most significant gap — the schema's `type: parallel` cannot wrap
   branch arms. Workaround: forensics runs sequentially AFTER containment. Marked with
   `# GAP:` comment.

4. **Step 6b (Medium → High escalation):** Prose says "if compromise confirmed, route to High
   Containment". Cross-branch goto is not expressible. Workaround: duplicated the essential High
   Containment steps inline within the Medium branch. Marked with `# GAP:` comment.

5. **Compensation (step 4b):** Used `type: compensate` to register rollback. The `on: failure`
   trigger is correct but note this triggers on ANY subsequent step failure, not specifically
   step 4f failure. This is a known limitation.

6. **Step 9 (Stakeholder notification):** Used `type: extension` with hypothetical
   `email-sender`. If severity == Critical, additional board notification would require a
   `when:` guard on a separate step.

## Recommended Schema Improvements

- Add `async: true` step attribute for background execution (enables forensics alongside containment)
- Add cross-branch goto or `type: goto` step for mid-runbook routing
- Add `on: step_id_failure` to `type: compensate` for targeted compensation triggers
- Extend `type: choice` to support `default:` value when timeout occurs without explicit skip
