# RunGraph Normalized Provenance Design

**Author:** Ken (Software Architect)  
**Date:** 2025-01-21  
**Status:** Proposed

---

## Problem Statement

The current `RunNode` structure lacks provenance metadata: you cannot determine which runbook file and step specification produced a given runtime node. The rejected `SourceRef` approach embedded full runbook metadata on every node, causing wasteful repetition when 100 nodes come from the same runbook.

**Requirements:**
1. Store runbook metadata once per unique runbook (not per node)
2. Each RunNode holds only a foreign key back to runbook metadata
3. Support include chains without duplication
4. Handle iterate: multiple nodes map to same source step
5. Enable GUI renderers to draw structural runbook graphs

---

## Design: Normalized Provenance Model

### 1. Data Structures

```go
// RunbookRef — stored once per unique runbook file in the graph
type RunbookRef struct {
    ID          string   // unique key: "main" | "include:<alias>" | "include:<alias>/<nested>"
    FilePath    string   // absolute or relative path to .runbook.yaml
    RunbookName string   // from meta.name in YAML
    ParentRefID string   // empty for root; references parent RunbookRef.ID for includes
    ImportAlias string   // empty for root; the alias used in parent's include statement
}

// StepRef — stored once per unique (runbook, step) pair
type StepRef struct {
    ID         string   // unique key: "<runbook-ref-id>:<step-id>"
    RunbookID  string   // foreign key → RunbookRef.ID
    StepID     string   // step identifier from YAML (e.g., "check_host")
    StepName   string   // human-readable name (e.g., "Check if host is reachable")
    StepKind   NodeKind // cli, tool, display, manual, approve, branch, iterate, etc.
}

// RunNode — runtime execution node (existing fields + new provenance FK)
type RunNode struct {
    // Existing fields (unchanged)
    ID       string
    Name     string
    Kind     NodeKind
    Status   NodeStatus
    Depth    int
    ParentID string
    ChildIDs []string
    Output   string
    
    // NEW: Provenance foreign key
    StepRefID  string   // foreign key → StepRef.ID
    Iteration  int      // 0 for non-iterate steps; N for Nth iteration of iterate
}

// RunGraph — execution graph with normalized provenance tables
type RunGraph struct {
    // Existing fields (unchanged)
    nodes     map[string]*RunNode
    order     []string
    focusedID string
    listeners []func(*RunGraph)
    
    // NEW: Provenance lookup tables
    runbooks  map[string]*RunbookRef   // key = RunbookRef.ID
    steps     map[string]*StepRef      // key = StepRef.ID
}
```

---

### 2. Lookup API

```go
// GetRunbookRef returns the runbook metadata for a given RunbookRef.ID.
// Returns nil if not found.
func (g *RunGraph) GetRunbookRef(runbookID string) *RunbookRef {
    return g.runbooks[runbookID]
}

// GetStepRef returns the step metadata for a given StepRef.ID.
// Returns nil if not found.
func (g *RunGraph) GetStepRef(stepRefID string) *StepRef {
    return g.steps[stepRefID]
}

// GetNodeProvenance returns the full provenance chain for a given runtime node.
// Returns (stepRef, runbookRef, includeChain) or (nil, nil, nil) if node not found.
//
// includeChain is ordered from root → leaf (e.g., ["main", "include:check", "include:check/nested"]).
func (g *RunGraph) GetNodeProvenance(nodeID string) (stepRef *StepRef, runbookRef *RunbookRef, includeChain []*RunbookRef) {
    node := g.nodes[nodeID]
    if node == nil {
        return nil, nil, nil
    }
    
    stepRef = g.steps[node.StepRefID]
    if stepRef == nil {
        return nil, nil, nil
    }
    
    runbookRef = g.runbooks[stepRef.RunbookID]
    if runbookRef == nil {
        return stepRef, nil, nil
    }
    
    // Walk up the include chain
    includeChain = []*RunbookRef{runbookRef}
    current := runbookRef
    for current.ParentRefID != "" {
        parent := g.runbooks[current.ParentRefID]
        if parent == nil {
            break // orphaned reference (shouldn't happen)
        }
        includeChain = append([]*RunbookRef{parent}, includeChain...)
        current = parent
    }
    
    return stepRef, runbookRef, includeChain
}

// RegisterRunbook adds a runbook to the provenance table.
// Panics if a runbook with the same ID already exists.
func (g *RunGraph) RegisterRunbook(ref RunbookRef) {
    if _, exists := g.runbooks[ref.ID]; exists {
        panic("session: RunGraph.RegisterRunbook: duplicate runbook ID: " + ref.ID)
    }
    r := ref
    g.runbooks[r.ID] = &r
}

// RegisterStep adds a step reference to the provenance table.
// Panics if a step with the same ID already exists.
func (g *RunGraph) RegisterStep(ref StepRef) {
    if _, exists := g.steps[ref.ID]; exists {
        panic("session: RunGraph.RegisterStep: duplicate step ID: " + ref.ID)
    }
    s := ref
    g.steps[s.ID] = &s
}
```

---

### 3. Changes to AddNode

The `AddNode` method signature remains unchanged, but callers must populate the new `StepRefID` field:

```go
// Example: adding a node from step "check_host" in runbook "include:check"
graph.AddNode(session.RunNode{
    ID:        "node-001",
    Name:      "Check if host is reachable",
    Kind:      session.NodeKindCLI,
    Status:    session.NodeStatusPending,
    Depth:     1,
    ParentID:  "node-000",
    ChildIDs:  []string{},
    Output:    "",
    StepRefID: "include:check:check_host",   // NEW: foreign key
    Iteration: 0,                             // NEW: iteration counter
})
```

**Caller responsibilities:**
1. Call `RegisterRunbook` for each unique runbook file encountered during planning
2. Call `RegisterStep` for each unique (runbook, step) pair
3. When creating a `RunNode`, set `StepRefID` to the corresponding `StepRef.ID`

---

### 4. Engine Event Extensions

To support this model, engine events must carry provenance information. Proposed additions:

#### `step/planned` (new event, emitted during planning phase)

Emitted once per unique step specification discovered during planning (before iteration expansion).

```json
{
  "kind": "step/planned",
  "payload": {
    "runbook_id": "include:check",
    "runbook_path": "/home/user/runbooks/check-service.runbook.yaml",
    "runbook_name": "Check Service Health",
    "parent_runbook_id": "main",
    "import_alias": "check",
    "step_id": "check_host",
    "step_name": "Check if host is reachable",
    "step_kind": "cli"
  }
}
```

Adapters listening to this event can call `RegisterRunbook` and `RegisterStep` to populate provenance tables before runtime nodes are created.

#### `step/started` (existing event, extended payload)

Add two fields to existing event:

```json
{
  "kind": "step/started",
  "payload": {
    "node_id": "node-001",
    "step_id": "check_host",
    "name": "Check if host is reachable",
    "kind": "cli",
    "depth": 1,
    
    // NEW:
    "step_ref_id": "include:check:check_host",
    "iteration": 0
  }
}
```

The adapter uses `step_ref_id` to populate `RunNode.StepRefID` when calling `AddNode`.

---

### 5. Include Chain Representation

Include chains are represented by linking `RunbookRef.ParentRefID` to the parent's `RunbookRef.ID`.

**Example:** `collect-health.runbook.yaml` includes `check-service.runbook.yaml` as alias `check`:

```go
// Step 1: Register root runbook
graph.RegisterRunbook(session.RunbookRef{
    ID:          "main",
    FilePath:    "/home/user/runbooks/collect-health.runbook.yaml",
    RunbookName: "Collect Health Metrics",
    ParentRefID: "",
    ImportAlias: "",
})

// Step 2: Register included runbook
graph.RegisterRunbook(session.RunbookRef{
    ID:          "include:check",
    FilePath:    "/home/user/runbooks/check-service.runbook.yaml",
    RunbookName: "Check Service Health",
    ParentRefID: "main",           // links to parent
    ImportAlias: "check",           // alias used in include statement
})

// Step 3: Register step from included runbook
graph.RegisterStep(session.StepRef{
    ID:         "include:check:check_host",
    RunbookID:  "include:check",
    StepID:     "check_host",
    StepName:   "Check if host is reachable",
    StepKind:   session.NodeKindCLI,
})

// Step 4: Add runtime node (iteration 0 of check_loop)
graph.AddNode(session.RunNode{
    ID:        "node-001",
    Name:      "Check if host is reachable",
    Kind:      session.NodeKindCLI,
    Status:    session.NodeStatusPending,
    Depth:     2,
    ParentID:  "node-000-check_loop",
    ChildIDs:  []string{},
    Output:    "",
    StepRefID: "include:check:check_host",
    Iteration: 0,
})

// Step 5: Add runtime node (iteration 1 of check_loop)
graph.AddNode(session.RunNode{
    ID:        "node-002",
    Name:      "Check if host is reachable",
    Kind:      session.NodeKindCLI,
    Status:    session.NodeStatusPending,
    Depth:     2,
    ParentID:  "node-000-check_loop",
    ChildIDs:  []string{},
    Output:    "",
    StepRefID: "include:check:check_host",  // SAME step ref as node-001
    Iteration: 1,                            // DIFFERENT iteration
})
```

**Query example:**

```go
stepRef, runbookRef, chain := graph.GetNodeProvenance("node-001")
// stepRef.StepID          = "check_host"
// runbookRef.FilePath     = "/home/user/runbooks/check-service.runbook.yaml"
// runbookRef.ImportAlias  = "check"
// chain[0].ID             = "main"
// chain[1].ID             = "include:check"
```

---

### 6. ID Naming Conventions

**RunbookRef.ID:**
- Root runbook: `"main"`
- Direct include: `"include:<alias>"` (e.g., `"include:check"`)
- Nested include: `"include:<parent-alias>/<child-alias>"` (e.g., `"include:check/nested"`)

**StepRef.ID:**
- Format: `"<runbook-ref-id>:<step-id>"`
- Examples:
  - `"main:deploy"` — step `deploy` in root runbook
  - `"include:check:check_host"` — step `check_host` in included runbook `check`
  - `"include:check/nested:ping"` — step `ping` in doubly-nested runbook

**RunNode.ID:**
- Generated by engine at runtime (e.g., UUID or sequential counter)
- No structural meaning; just a unique identifier

---

## Benefits

1. **No duplication:** Runbook metadata stored once, not once per node
2. **Efficient lookup:** O(1) access to provenance via foreign key
3. **Include chain reconstruction:** Walk `ParentRefID` links to build full path
4. **Iteration support:** Multiple nodes share same `StepRefID`, differentiated by `Iteration`
5. **Adapter-agnostic:** TUI, web, VS Code, and harness all use same API
6. **Event-driven population:** Adapters build provenance tables from `step/planned` events

---

## Migration Notes

**Existing code impact:**
- `RunNode` struct gains two fields (`StepRefID`, `Iteration`)
- `RunGraph` struct gains two fields (`runbooks`, `steps`)
- Adapters must listen to `step/planned` events to populate provenance tables
- Engine must emit `step/planned` events during planning phase
- `step/started` events must include `step_ref_id` and `iteration`

**Backward compatibility:**
- Old adapters ignore new fields (graceful degradation)
- New adapters can detect missing provenance and fall back to node name display

---

## Future Extensions (Out of Scope for v2.0)

- **Provenance compression:** For large graphs, serialize provenance tables to JSONL trace as `metadata/*` events
- **Cross-run provenance:** Link runs that invoke sub-runbooks via `invoke` step
- **Visual diff:** Compare two runs of same runbook to highlight structural changes
- **Step usage analytics:** Query which runbooks/steps are most frequently executed

---

## Open Questions

1. **Event timing:** Should `step/planned` be emitted during planning phase, or lazily when first node is created?
   - **Recommendation:** Emit during planning. Adapters can pre-populate provenance tables before execution starts.

2. **Runbook ID conflicts:** What if two included runbooks use the same alias in different branches?
   - **Recommendation:** Use fully-qualified path: `"include:<parent>/<alias>"` to disambiguate.

3. **Trace file size:** Does adding `step/planned` events significantly inflate trace files?
   - **Analysis needed:** Benchmark 100-step runbook with 5 includes. If problematic, defer to v2.1.

---

## Implementation Handoff to Brian

**Deliverables:**
1. Update `internal/session/navigation.go`:
   - Add `RunbookRef` and `StepRef` types
   - Add fields to `RunNode`: `StepRefID`, `Iteration`
   - Add fields to `RunGraph`: `runbooks`, `steps`
   - Implement `RegisterRunbook`, `RegisterStep`, `GetNodeProvenance`
2. Update engine event emitter:
   - Add `step/planned` event type
   - Extend `step/started` payload with `step_ref_id` and `iteration`
3. Update TUI adapter:
   - Subscribe to `step/planned` events
   - Populate provenance tables before execution
4. Add integration test:
   - Verify provenance lookup for included runbook with iteration
   - Assert include chain reconstruction is correct

**Estimated effort:** 2-3 hours (mostly mechanical changes)

---

## Architectural Review Checklist

- [x] No data duplication (runbook metadata stored once)
- [x] O(1) lookup performance (foreign key + hash map)
- [x] Supports all navigation requirements (TUI, web, VS Code, harness)
- [x] Handles include chains without recursion limit
- [x] Iteration semantics correct (multiple nodes → one StepRef)
- [x] Event schema is additive (backward compatible)
- [x] API surface is minimal (3 lookup methods)
- [x] Naming conventions are consistent and unambiguous

---

**Status:** Ready for team review. Awaiting approval from Cristian before handoff to Brian.
