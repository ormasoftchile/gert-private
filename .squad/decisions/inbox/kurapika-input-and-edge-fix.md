# Decision: Input variable mismatch fixed + passthrough-node edge color fix

**Date:** 2026-04-06  
**Author:** Kurapika  
**Commit:** dbb8cb1

---

## Bug 1 — User input was silently ignored in execution

### Root cause
`service-health-branching.runbook.yaml` had two separate declarations:
- `meta.vars.hostname: github.com` — the variable actually used in all `{{ .hostname }}` templates
- `meta.inputs.server_name: { from: prompt }` — collected at runtime but never referenced by any template

When the user entered `"f"` for `server_name`, the value was correctly sent to the server as `vars: { server_name: "f" }` and merged into `rb.Meta.Vars`. But `{{ .hostname }}` was never overridden, so the DNS lookup used `"github.com"` every time.

### Fix
Replaced the split `vars`/`inputs` pattern with a single input:

```yaml
# Before
meta:
  vars:
    hostname: github.com
  inputs:
    server_name:
      from: prompt
      description: Server to check

# After
meta:
  inputs:
    hostname:
      from: prompt
      description: Hostname to check (DNS lookup, HTTP, and ping fallback)
      default: github.com
```

`hostname` is now the user-prompted var. The server's input-resolution logic correctly uses the user's value as `rb.Meta.Vars["hostname"]`, overriding any default.

### Decision
Input variable names in `inputs:` MUST match the `{{ .varName }}` references in step templates. When `inputs.server_name` is collected but templates use `{{ .hostname }}`, the input is a no-op. This convention should be enforced at schema validation time (backlog item for Leorio/backend).

---

## Bug 2 — Passthrough node edges rendered green for untaken branches

### Root cause
In `web/src/shared/treeToGraph.ts`, when `parentDecided && !branchIsTaken`, the branch is collapsed into a narrow passthrough node:

```typescript
const passId = `pass-${id}-${i}-${uid()}`;
nodeStepMap.set(passId, id);  // ← BUG: maps passId → parent branch-step ID
```

`nodeStepId(passId)` resolved to the parent branch-step (`resolve_dns`), which IS taken. So edges involving the passthrough node — including the edge from passthrough → synthetic `end-0` — computed `taken: true` and rendered green, even though the branch was never executed.

### Fix
Removed the `nodeStepMap.set(passId, id)` call. Without a map entry, `nodeStepId(passId)` returns `passId` itself (the fallback `|| nid`). Since `passId` is a synthetic graph ID absent from `stepStates`, `resolveState(passId)` returns `'pending'` and `isTaken` returns `false` → grey edge.

### Affected edges
- Edge from passthrough node → `end-0` (synthetic end node): now correctly grey
- Edge from passthrough node → any downstream sequential node: now correctly grey

### Decision
Passthrough nodes are purely visual placeholders for untaken branches. They must never be registered in `nodeStepMap` to avoid inheriting the parent step's taken state. The invariant is: "a graph node's `nodeStepId` must only resolve to a step that represents the node's own execution" — synthetic/placeholder nodes should resolve to themselves (and thus be absent from `stepStates`).
