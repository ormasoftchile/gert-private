# Assessment: Medical Device Software Release (FDA/Quality)

**Completeness:** 7/10 (70%)  
**Fidelity:** 6/10 (60%)  
**Verdict:** FAIL

## Gaps Found

| Gap ID | Class | Severity | Description |
|--------|-------|----------|-------------|
| G4-001 | G4 | CRITICAL | No external event trigger — step 8e (wait for FDA clearance, 90+ days) requires pausing and resuming on webhook from FDA portal; schema has no `type: wait_for_event` or pause/resume mechanism |
| G2-001 | G2 | CRITICAL | No 21 CFR Part 11 signature metadata — 7+ approval steps require digital signature with certificate thumbprint, algorithm, and signer identity; schema records role+timestamp only |
| G2-002 | G2 | HIGH | Business-day timeouts not supported — all approval steps use business days; expressed as wall-clock hours with accuracy loss |
| G2-003 | G2 | HIGH | Cryptographic signing of artifacts — step 11 signs SBOM with code signing certificate; schema has no native `type: sign` or private key management |
| G2-004 | G2 | MEDIUM | Document versioning and 10-year retention policy not expressible in schema; handled by external PLM system |

## Translation Notes

1. **Step 3 (Clinical Validation — Conditional):** Used `when: '{{ eq .affects_clinical_functionality "yes" }}'`
   to conditionally skip clinical validation. This is correct per the v2 schema `when:` guard.

2. **Step 5 (Regulatory Affairs Review):** Used `type: choice` correctly for human selection of
   submission type. The rationale capture is approximated via the approval timestamp and role.
   There is no explicit "rationale" field for FDA audit.

3. **Step 8e (FDA Clearance Wait):** The most significant gap. The prose says "pause runbook
   for 90 days, resume on FDA webhook." The schema has no event-driven pause/resume. Workaround:
   modeled as a `type: collector` manual step asking the operator to check status. This means
   the runbook stays "open" and requires human re-entry rather than automated resumption.
   Marked with `# GAP:` comment.

4. **21 CFR Part 11 Signatures:** FDA regulations require digital signatures with certificate
   metadata. The schema's `approvals:` block only captures role name and timestamp. Each
   approval step is marked with `# GAP: No 21 CFR Part 11 signature metadata`.

5. **Step 10 (Manufacturing Release):** Used `approvals.min: 2` with both quality-manager and
   CEO roles. This correctly enforces dual approval.

6. **Step 11 (SBOM Signing):** Used a bash cli step with hypothetical `cosign` command. The
   private key management (`--key` parameter) is not expressible in the schema. Marked with
   `# GAP:` comment.

## Why FAIL

This runbook FAILS because:
1. Step 8e (FDA clearance wait, 90+ days) cannot be modeled correctly — the schema has no
   event-driven resume mechanism
2. All 7+ approval steps lack 21 CFR Part 11 signature metadata required for FDA compliance
3. These are not workaroundable — both gaps represent fundamental FDA compliance requirements

## Recommended Schema Improvements

- Add `type: wait_for_event` step: `{event: webhook, source: "fda-portal", resume_on: {payload.status: cleared}}`
- Add `approvals.signature_policy:` field for 21 CFR Part 11 compliance metadata
- Add `type: sign` step with `key_ref:` for cryptographic artifact signing
- Add `retention:` field to artifact capture for long-term storage policies
