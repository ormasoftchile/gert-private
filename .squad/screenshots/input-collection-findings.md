# Input Collection Flow Test Results

## Test Date
2026-04-06

## Test Subject
Testing whether runbooks with `from: prompt` inputs show an input collection form before execution, or if they start executing immediately.

## Runbooks with User Inputs (from: prompt)

1. **service-health-branching.runbook.yaml**
   - Inputs: `server_name` (Server to check)

2. **incident-triage.runbook.yaml**
   - Inputs: `service_name` (Affected service name), `incident_id` (Incident tracking ID)

3. **multi-region-rollout.runbook.yaml**
   - Inputs: `deploy_id` (Deployment ID to validate)

4. **incident-triage.app-crash.runbook.yaml**
   - Inputs: `service_name` (Affected service name), `incident_id` (Incident tracking ID)

5. **incident-triage.connectivity-test.runbook.yaml**
   - Inputs: `service_name` (Service endpoint to test), `incident_id` (Incident tracking ID)

6. **incident-triage.resource-exhaustion.runbook.yaml**
   - Inputs: `service_name` (Affected service name), `incident_id` (Incident tracking ID)

7. **incident-triage.network.runbook.yaml**
   - Inputs: `service_name` (Affected service name), `incident_id` (Incident tracking ID)

8. **windows-diagnostic/runbooks/network-health-check.runbook.yaml**
   - Inputs: `primary_host` (Primary hostname to diagnose, e.g. github.com)

9. **windows-diagnostic/runbooks/certificate-audit.runbook.yaml**
   - Inputs: `additional_host` (Additional hostname to check, optional)

10. **windows-diagnostic/runbooks/full-diagnostic.runbook.yaml**
    - Inputs: `primary_host` (Primary hostname for network diagnostics), `additional_host` (Additional hostname for cert checks, optional)

## Test Results

### Test Runbook
`windows-diagnostic/runbooks/network-health-check.runbook.yaml` (has 1 input: `primary_host` from: prompt)

### Findings

**CRITICAL: No input collection form appears!**

The runbook **starts executing immediately** after clicking the Run button, without prompting the user for the required `primary_host` input.

### Screenshot Evidence

1. **input-collection-02-before-run.png**: Shows the Runbook Runner interface with path input and Run button ready
2. **input-collection-03-after-run-click.png** (0.5s after clicking Run): Shows execution already started with "Executing step..." and workflow map visible
3. **input-collection-05-final-state.png** (5.5s after clicking Run): Shows runbook execution in progress at the "Network Health Summary" step

### Test Console Output
```
Has input form fields: false
Has modal dialog: true (execution UI, not input collection)
Has prompt UI: false
Page contains "primary_host": false
Page contains "prompt": false
Page contains "input": true (general term, not input form)
```

### Observed Behavior
- No modal dialog for input collection appeared
- No input fields specific to `primary_host` were shown
- Execution started immediately upon clicking Run
- The runbook proceeded with an empty/undefined value for `primary_host`
- The final report shows "Primary Host:" with no value

## Conclusion

**Input collection is NOT implemented in the current UI.** Runbooks with `from: prompt` inputs execute immediately without collecting those inputs from the user. This results in the runbook executing with undefined/empty values for required inputs.

## Recommendation

The UI needs an input collection step that:
1. Detects when a runbook has inputs with `from: prompt`
2. Shows a modal/form before execution starts
3. Collects all required input values from the user
4. Passes those values to the execution engine
5. Only starts execution after all inputs are provided
