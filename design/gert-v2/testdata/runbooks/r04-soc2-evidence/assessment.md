# Assessment: SOC2 Evidence Collection for Audit

**Completeness:** 9/10 (90%)  
**Fidelity:** 8/10 (80%)  
**Verdict:** PASS WITH NOTES

## Gaps Found

| Gap ID | Class | Severity | Description |
|--------|-------|----------|-------------|
| G6-001 | G6 | HIGH | Bulk invocation verbosity — 15 sequential invoke steps for 15 controls; schema lacks `invoke_many:` or `iterate.over: imports.*` syntax |
| G2-001 | G2 | HIGH | Business-day timeout not supported — compliance officer review (5 days), security officer (3 days) expressed as wall-clock hours |
| G2-002 | G2 | MEDIUM | No dropdown field type in collector — mitigation status forced to text + hint |
| G2-003 | G2 | MEDIUM | WORM/object-lock storage policy not expressible — S3 object lock requires separate API call outside schema |
| G2-004 | G2 | MEDIUM | Presigned URL generation not natively supported — would require an additional cli step or extension |

## Translation Notes

1. **Sub-runbook invocations (steps 2-8):** Used `type: invoke` for each control sub-runbook.
   This is the correct v2 pattern for nested runbook composition. Input propagation (audit_id,
   period_start, period_end) works correctly. The 15-step verbosity is a limitation but valid.

2. **CC6.1-CC6.8 consolidation:** Bundled the 8 CC6 controls into a single invoke step pointing
   to a hypothetical `cc6-access-controls-bundle.yaml` to reduce verbosity. In a real
   implementation, these would each be separate invoke steps.

3. **Compliance officer review (step 11):** Used `type: collector` with attestation fields and
   `approvals:` block. Business-day timeouts approximated as wall-clock hours with `# GAP:`
   comments.

4. **Evidence aggregation (step 9):** Used a bash cli step combining tar and sha256sum. The
   schema doesn't have a native artifact aggregation primitive.

5. **WORM storage:** The `aws s3 cp` step with `--sse` handles encryption but not object lock.
   Object lock would require a separate `aws s3api put-object-legal-hold` call. Marked with
   `# GAP:` comment.

## Recommended Schema Improvements

- Add `invoke_many:` syntax for bulk sub-runbook invocation with shared inputs
- Add calendar-aware timeout syntax
- Add `storage_policy:` field to artifact capture for retention, versioning, and encryption
- Add `generate_presigned_url:` action or `type: artifact` step
