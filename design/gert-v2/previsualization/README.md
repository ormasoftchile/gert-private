# Gert Runbook Previsualization

This sandbox explores how speculative runbook schema constructs could map to graph visuals using React Flow.

## What this is

- A design playground for visual language and interaction primitives.
- Not wired to parser/runtime yet.
- Built from archetypes under `design/gert-v2/testdata/runbooks/*/schema.yaml`.

## Included scenarios

- K8s Incident Response
- Employee Onboarding
- Canary Deployment + Compensation
- SOC2 Evidence Collection

## Visual stereotypes

- Trigger: event wait / start points
- Human: collector, choice, decision, approval gates
- Control: branch, parallel, iterate, joins, assertions
- Automation: cli, tool, extension, include
- Compensation: rollback and compensating action zones
- Terminal: end nodes

## Commands

```sh
npm install
npm run dev
npm run build
npm run preview
```

## Next iteration ideas

- Add parsed schema importer and automatic graph generation
- Add grouped subflows for include and parallel branch containers
- Add edge styling by semantics (success/failure/timeout/approval)
- Add execution playback mode for runtime traces
