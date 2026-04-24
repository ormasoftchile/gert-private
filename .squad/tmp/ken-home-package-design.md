# gert-domain-home Go Package Structure Design

**Author:** Ken (Software Architect)  
**Date:** 2024-04-24  
**Status:** v0 Design — Ready for Implementation  
**Context:** Package layout for gert-domain-home domain kit v0

---

## 1. Module Placement Decision

**Decision:** `domains/home/` as a separate Go module at repository root

**Rationale:**

1. **Architectural Separation** — Domain kits are compilation layers, not core runtime. They should live outside `v2/` to maintain a clear conceptual boundary. The v2 module is the GERT runtime; domain kits are consumers of that runtime.

2. **Independent Versioning Path** — While v0 lives in the monorepo, the separate module structure enables future extraction to `github.com/ormasoftchile/gert-domain-home` without refactoring. Domain kits should version independently from core runtime.

3. **Dependency Direction** — Home kit depends ON gert v2 (imports `pkg/schema`, `pkg/engine`, etc.), never the reverse. A separate module enforces this dependency direction at compile time.

4. **go.work Integration** — The repo already uses `go.work` for multi-module coordination. Adding `./domains/home` as a workspace member is trivial and consistent with existing `./ext/*` modules.

5. **Monorepo Benefits Retained** — Using `replace` directives in `go.work`, the home kit can reference the local v2 module during development without publishing intermediate versions.

**Alternative Considered:** `v2/pkg/domains/home/`  
**Why Rejected:** Blurs the core/kit boundary. Domain kits should not live inside the runtime module — they are separate compilation units that happen to share a repository for v0 convenience.

---

## 2. Directory Tree

```
domains/home/
├── go.mod                          # Module: github.com/ormasoftchile/gert/domains/home
├── go.sum
├── README.md                        # Kit overview: purpose, usage, examples
├── pkg/
│   ├── model/                       # Domain types (Property, Zone, Asset, Routine, Incident)
│   │   ├── doc.go                   # Package documentation
│   │   ├── property.go              # Property, Zone types
│   │   ├── asset.go                 # Asset type
│   │   ├── routine.go               # Routine, Cadence types
│   │   ├── incident.go              # Incident, RepairRun types
│   │   ├── evidence.go              # Evidence wrapper (thin over GERT evidence)
│   │   ├── delegation.go            # Delegation, AwayMode types
│   │   └── executor.go              # Executor type
│   │
│   ├── loader/                      # YAML DSL parser (.home.yaml → model types)
│   │   ├── doc.go                   # Package documentation
│   │   ├── loader.go                # PropertyLoader interface + implementation
│   │   ├── property.go              # Parse property + zones from YAML
│   │   ├── assets.go                # Parse assets array
│   │   ├── routines.go              # Parse routines array
│   │   ├── validation.go            # Zone ID uniqueness, cadence rules
│   │   └── loader_test.go           # Parse golden files from ../../testdata/
│   │
│   ├── compiler/                    # Domain model → GERT execution graph
│   │   ├── doc.go                   # Package documentation
│   │   ├── compiler.go              # DomainCompiler interface + implementation
│   │   ├── routine.go               # Routine → timer-backed run + human task
│   │   ├── incident.go              # Incident → ad-hoc run + repair sub-run
│   │   ├── cadence.go               # Cadence → timer policy (simple interval only)
│   │   ├── evidence.go              # Evidence requirements → GERT evidence spec
│   │   └── compiler_test.go         # Compile model → ExecutionPlan, verify structure
│   │
│   └── delegation/                  # Away mode + delegate routing logic
│       ├── doc.go                   # Package documentation
│       ├── policy.go                # DelegationPolicy interface + implementation
│       ├── routing.go               # Evaluate delegation window + scoped tasks filter
│       ├── projection.go            # Delegate-scoped projection (filter to assigned tasks)
│       └── delegation_test.go       # Time-bounded routing, filter correctness
│
├── cmd/
│   └── home-validate/               # CLI tool: validate a .home.yaml file
│       └── main.go                  # Load + validate property file, print errors
│
└── testdata/
    ├── property-simple.yaml         # Minimal valid property (1 zone, 1 routine)
    ├── property-full.yaml           # Full example from spec §3.7
    ├── property-invalid-zone.yaml   # Invalid zone ID (starts with digit)
    └── routine-seasonal.yaml        # Seasonal cadence (v1 feature, loader rejects)
```

**Key Design Points:**

- **4 core packages:** `model`, `loader`, `compiler`, `delegation` — each with clear responsibility
- **No internal/** — All packages are public API for this kit (v0 simplicity)
- **testdata/** — Shared across packages; loader tests golden files, compiler tests use model fixtures
- **cmd/home-validate** — Proves the loader works; useful for manual property file authoring

---

## 3. Key Interfaces

### 3.1 PropertyLoader (pkg/loader)

```go
package loader

import (
	"context"
	"io"
	
	"github.com/ormasoftchile/gert/domains/home/pkg/model"
)

// PropertyLoader parses a .home.yaml file into a Property model.
// Validates zone ID uniqueness, cadence rules, and executor references.
type PropertyLoader interface {
	// Load reads a property definition from r and returns the parsed model.
	// Returns error if YAML is malformed, validation fails, or v1-only features are used.
	Load(ctx context.Context, r io.Reader) (*model.Property, error)
}

// New returns the default PropertyLoader implementation.
func New() PropertyLoader {
	return &loader{}
}
```

**Validation Rules:**
- Zone IDs must match `^[a-z][a-z0-9_]*$` and be unique within property
- Cadence: only `type: simple` + `every_n_days` allowed (reject seasonal in v0)
- Routine must reference a valid zone ID
- Asset must reference a valid zone ID

**Error Handling:** Return structured validation errors (not just "parse failed")

---

### 3.2 DomainCompiler (pkg/compiler)

```go
package compiler

import (
	"context"
	
	"github.com/ormasoftchile/gert/domains/home/pkg/model"
	"github.com/ormasoftchile/gert/v2/pkg/engine"
)

// DomainCompiler compiles a home Property into a GERT ExecutionPlan.
// Each Routine becomes a timer-backed run. Incidents create ad-hoc runs.
type DomainCompiler interface {
	// CompileProperty converts a Property model into a GERT ExecutionPlan.
	// The plan contains one durable run per routine (with timer + human task).
	// Returns error if model contains unsupported features or compilation fails.
	CompileProperty(ctx context.Context, p *model.Property) (*engine.ExecutionPlan, error)
	
	// CompileIncident creates an ad-hoc ExecutionPlan for a reported incident.
	// The plan contains one run with a repair sub-run (4 sequential human tasks).
	CompileIncident(ctx context.Context, inc *model.Incident) (*engine.ExecutionPlan, error)
}

// New returns the default DomainCompiler implementation.
func New() DomainCompiler {
	return &compiler{}
}
```

**Compilation Semantics:**

1. **Routine → Run:**
   - Run ID: `routine:{routine_id}`
   - Timer: `initial_wakeup = now + cadence.every_n_days * 86400`
   - Step: Human task with description = routine.name, executor = routine.executor_id
   - Evidence: Attach requirement if `requires_evidence: true`
   - On step completion, timer resets to `last_completion + interval`

2. **Incident → Run + Sub-Run:**
   - Parent run ID: `incident:{incident_id}`
   - Sub-run ID: `repair:{incident_id}`
   - Sub-run steps: `diagnose`, `buy_parts`, `fix`, `verify` (all human tasks, sequential)
   - Each step requires evidence (note or photo)
   - Parent run completes when sub-run completes

**GERT Dependencies:**
- Import `github.com/ormasoftchile/gert/v2/pkg/engine` for `ExecutionPlan`, `Run`, `Step` types
- Import `github.com/ormasoftchile/gert/v2/pkg/schema` for step type definitions (HumanTask, Timer, etc.)

---

### 3.3 DelegationPolicy (pkg/delegation)

```go
package delegation

import (
	"context"
	"time"
	
	"github.com/ormasoftchile/gert/domains/home/pkg/model"
)

// DelegationPolicy evaluates whether a task should route to a delegate.
// Used by GERT runtime when creating human tasks during delegation window.
type DelegationPolicy interface {
	// ShouldDelegate returns true if the task should route to delegate (not owner).
	// Evaluates current time against delegation window + task name against scoped filter.
	ShouldDelegate(ctx context.Context, d *model.Delegation, taskName string, now time.Time) bool
	
	// GetDelegateExecutorID returns the executor_id to assign if delegation is active.
	// Returns empty string if delegation is inactive or task is out of scope.
	GetDelegateExecutorID(ctx context.Context, d *model.Delegation, taskName string, now time.Time) string
}

// TaskFilter defines delegate-scoped projection (read-side filter).
type TaskFilter interface {
	// FilterForDelegate returns only tasks visible to the delegate.
	// Filters by executor_id, due_date within delegation window, status in [pending, in_progress].
	FilterForDelegate(ctx context.Context, d *model.Delegation, allTasks []*model.Task) []*model.Task
}

// New returns default implementations.
func NewPolicy() DelegationPolicy { return &policy{} }
func NewFilter() TaskFilter { return &filter{} }
```

**Evaluation Logic (ShouldDelegate):**
```
if now < d.StartDate || now > d.EndDate:
    return false  # delegation expired
if taskName not in d.ScopedTasks:
    return false  # task not assigned to delegate
return true       # route to delegate
```

**Projection Logic (FilterForDelegate):**
```
filtered := []
for task in allTasks:
    if task.executor_id == d.DelegateID:
        if task.due_date >= d.StartDate && task.due_date <= d.EndDate:
            if task.status in [pending, in_progress]:
                filtered.append(task)
return filtered
```

---

### 3.4 Domain Model Types (pkg/model)

**Core types:**

```go
package model

import "time"

// Property is the top-level container for a household.
type Property struct {
	ID      string
	Name    string
	Address string
	OwnerID string
	Zones   []*Zone
	Assets  []*Asset
	Routines []*Routine
	Delegations []*Delegation
	CreatedAt time.Time
}

// Zone is a named area within a property.
type Zone struct {
	ID          string
	Label       string
	Description string
	PropertyID  string
	Metadata    map[string]interface{}
}

// Routine is a scheduled recurring maintenance task.
type Routine struct {
	ID               string
	Name             string
	ZoneID           string
	AssetID          string  // optional
	Cadence          Cadence
	ExecutorID       string
	RequiresEvidence bool
}

// Cadence defines recurrence interval (v0: simple only).
type Cadence struct {
	Type        string  // "simple" only in v0
	EveryNDays  int     // e.g., 7 for weekly
}

// Incident is a user-reported reactive event.
type Incident struct {
	ID          string
	Title       string
	Description string
	ZoneID      string
	AssetID     string  // optional
	ReportedBy  string
	ReportedAt  time.Time
	Severity    string  // "low", "medium", "high"
}

// Delegation is a time-bounded transfer of task execution authority.
type Delegation struct {
	ID           string
	DelegateName string
	DelegateContact string
	DelegateID   string  // executor_id for the delegate
	StartDate    time.Time
	EndDate      time.Time
	ScopedTasks  []string  // routine names assigned to delegate
}

// Task represents a specific maintenance task instance (read-side projection).
type Task struct {
	ID          string
	RoutineID   string
	Name        string
	DueDate     time.Time
	Status      string  // "pending", "in_progress", "completed", "skipped"
	ExecutorID  string
	CompletedAt *time.Time
	Evidence    []Evidence
}

// Evidence is a photo or note attached to a completed task.
type Evidence struct {
	Type      string    // "photo", "note"
	Content   string    // base64 photo or text note
	Timestamp time.Time
	SHA256    string
}
```

**Design Notes:**
- All types are exported (public API of the kit)
- No GERT types leak into model package (it's pure domain vocabulary)
- Delegation uses `time.Time` for start/end (not strings) — validated at load time

---

## 4. Dependency Graph

**Home kit imports from gert v2:**

```
domains/home/pkg/compiler
  ├─ import "github.com/ormasoftchile/gert/v2/pkg/engine"
  │   └─ Uses: ExecutionPlan, Run, Step types
  ├─ import "github.com/ormasoftchile/gert/v2/pkg/schema"
  │   └─ Uses: HumanTaskStep, TimerStep, SubRunStep type definitions
  └─ import "github.com/ormasoftchile/gert/v2/pkg/evidence"
      └─ Uses: Evidence primitive (SHA256, append-only log)

domains/home/pkg/delegation
  └─ (No GERT imports — operates on model types only)

domains/home/pkg/loader
  ├─ import "gopkg.in/yaml.v3"  (already in v2 go.mod)
  └─ (No GERT imports — outputs model types)

domains/home/pkg/model
  └─ (No GERT imports — pure domain vocabulary)
```

**Key Invariant:** Only `compiler` package imports GERT runtime types. Model, loader, and delegation operate entirely in domain vocabulary.

**Version Coupling:** Home kit v0 depends on gert v2.0 (or later). Specified in go.mod:
```go
require github.com/ormasoftchile/gert/v2 v2.0.0
```

During development, use `replace` in `go.work`:
```
use ./v2
use ./domains/home
```

---

## 5. go.mod Stub

```go
module github.com/ormasoftchile/gert/domains/home

go 1.25.7

require (
	github.com/ormasoftchile/gert/v2 v2.0.0
	gopkg.in/yaml.v3 v3.0.1
)

// Development: reference local v2 module via go.work replace directive
```

**Notes:**
- Module path: `github.com/ormasoftchile/gert/domains/home` (not `/domains/home/v0`)
- Go version: 1.25.7 (matches v2 go.mod)
- Dependencies: Only v2 core and YAML parser (no other external deps in v0)
- No `replace` directives in this go.mod — handled by go.work at repo root

---

## 6. go.work Addition

Add this line to `/Users/cristianormazabal/Projects/gert/go.work`:

```diff
 use (
 	.
 	./ext/debug
 	./ext/diagram
 	./ext/mcp
 	./ext/render
 	./ext/serve
 	./ext/tui
 	./v2
+	./domains/home
 )
```

**Placement:** After `./v2`, before closing paren — maintains alphabetical order within workspace.

**Effect:** Enables local development workflow:
- `cd domains/home && go test ./...` works without publishing v2
- `go work sync` keeps module dependencies aligned
- IDE (VS Code) recognizes home kit as part of workspace

---

## 7. v0 Implementation Order

**Phase 1: Model + Loader (Week 1)**

1. **Scaffold directory structure:**
   ```bash
   mkdir -p domains/home/{pkg/{model,loader,compiler,delegation},cmd/home-validate,testdata}
   cd domains/home
   go mod init github.com/ormasoftchile/gert/domains/home
   ```

2. **Implement `pkg/model/`:**
   - `property.go`: Property, Zone types
   - `routine.go`: Routine, Cadence types
   - `asset.go`: Asset type (stub — not used in v0 compilation)
   - `incident.go`: Incident type
   - `delegation.go`: Delegation type
   - `executor.go`: Executor type (may merge into property.go)
   - All types exported, fully documented with godoc comments

3. **Implement `pkg/loader/`:**
   - `loader.go`: PropertyLoader interface + `New()`
   - `property.go`: Parse property + zones from YAML
   - `routines.go`: Parse routines array
   - `assets.go`: Parse assets array (minimal — just name + zone)
   - `validation.go`: Zone ID regex, uniqueness check, cadence type check
   - `loader_test.go`: Parse `testdata/property-simple.yaml`, verify model populated

4. **Implement `cmd/home-validate/`:**
   - `main.go`: Accept `.home.yaml` file path, call loader, print errors or "valid"
   - Test manually: `go run ./cmd/home-validate testdata/property-simple.yaml`

5. **Add golden test files to `testdata/`:**
   - `property-simple.yaml`: 1 zone, 1 routine (minimal valid)
   - `property-full.yaml`: Copy from spec §3.7 (pool, lawn, garage zones)
   - `property-invalid-zone.yaml`: Zone ID "2pool" (starts with digit — should reject)

**Exit Criteria Phase 1:**  
- [ ] `pkg/model` compiles, all types documented  
- [ ] `pkg/loader` parses `property-simple.yaml` without errors  
- [ ] `home-validate` CLI tool runs successfully  
- [ ] All tests pass: `go test ./pkg/loader`

---

**Phase 2: Compiler (Week 2)**

1. **Implement `pkg/compiler/`:**
   - `compiler.go`: DomainCompiler interface + `New()`
   - `routine.go`: Routine → ExecutionPlan with timer-backed run + human task
   - `cadence.go`: Cadence → timer interval (simple only: `every_n_days * 86400`)
   - `incident.go`: Incident → ad-hoc run + repair sub-run (4 steps)
   - `evidence.go`: Map `requires_evidence: true` to GERT evidence requirement
   - `compiler_test.go`: Compile simple routine, verify ExecutionPlan structure

2. **Wire GERT dependencies:**
   - Import `github.com/ormasoftchile/gert/v2/pkg/engine`
   - Import `github.com/ormasoftchile/gert/v2/pkg/schema`
   - Verify types match: `engine.ExecutionPlan`, `schema.HumanTaskStep`, etc.

3. **Test compilation:**
   - Load `property-simple.yaml` → compile → verify ExecutionPlan has 1 run
   - Verify run has timer with correct interval (7 days → 604800 seconds)
   - Verify run has 1 human task step with description = routine name

**Exit Criteria Phase 2:**  
- [ ] `pkg/compiler` compiles routine to valid ExecutionPlan  
- [ ] ExecutionPlan structure matches spec §4 (timer + human task)  
- [ ] All tests pass: `go test ./pkg/compiler`

---

**Phase 3: Delegation (Week 3)**

1. **Implement `pkg/delegation/`:**
   - `policy.go`: DelegationPolicy interface + `NewPolicy()`
   - `routing.go`: `ShouldDelegate()` logic (time window + scoped tasks filter)
   - `projection.go`: TaskFilter interface + `FilterForDelegate()`
   - `delegation_test.go`: Test time-bounded routing, verify filter correctness

2. **Test scenarios:**
   - Delegation active (now between start/end) + task in scope → delegate
   - Delegation active + task NOT in scope → owner
   - Delegation expired (now > end_date) → owner
   - Projection: filter 10 tasks to 3 delegate-assigned tasks

**Exit Criteria Phase 3:**  
- [ ] `pkg/delegation` correctly routes tasks based on time + scope  
- [ ] Projection filters tasks to delegate-visible subset  
- [ ] All tests pass: `go test ./pkg/delegation`

---

**Phase 4: Integration (Week 4)**

1. **End-to-End test:**
   - Load `property-full.yaml` → compile → verify 3 routines become 3 runs
   - Create delegation → compile routine with delegation active → verify executor_id = delegate
   - Create incident → compile → verify repair run has 4 sequential steps

2. **Documentation:**
   - `domains/home/README.md`: Kit overview, usage example, compilation semantics
   - Godoc comments for all exported types and interfaces

3. **Hand off to Barbara:**
   - Barbara writes integration test: load → compile → execute with v2 runtime
   - Verify timer wakes, human task created, evidence attached, timer resets

**Exit Criteria Phase 4:**  
- [ ] E2E test: property → ExecutionPlan → runtime execution (1 routine completes)  
- [ ] Documentation complete  
- [ ] Ready for v0 demo

---

## 8. Open Design Questions

### 8.1 Timer Policy Interface

**Issue:** GERT v2.0 timer primitive may not yet support variable intervals (cadence rules).  

**Current assumption:** Timer has `next_wakeup` field that can be updated on completion.

**If assumption is wrong:** Compiler may need to generate timer policy code (Rego or Go function) to calculate next wakeup.

**Resolution needed:** Review `v2/pkg/schema/steps.go` — does TimerStep support dynamic interval? If not, defer seasonal cadence AND simple cadence reset logic to v2.1.

**Assigned to Brian:** Verify timer primitive capabilities before Phase 2.

---

### 8.2 Sub-Run Support

**Issue:** Incident repair run is a sub-run of incident run. Does GERT v2.0 support sub-runs?

**Current assumption:** `engine.ExecutionPlan` has a `SubRuns` field or `Steps` array can contain sub-run references.

**If assumption is wrong:** Incident compilation will fail. Fallback: flatten repair steps into incident run (no parent-child relationship).

**Resolution needed:** Review `v2/pkg/engine/run.go` and `v2/pkg/schema/step.go` — is sub-run a first-class primitive?

**Assigned to Brian:** Verify sub-run support before Phase 2.

---

### 8.3 Evidence Attachment API

**Issue:** How does evidence attach to a human task step in GERT v2?

**Current assumption:** `engine.Run` has method like `AttachEvidence(stepID, evidenceBlob)`.

**If assumption is wrong:** Home kit may need to call separate evidence API (`pkg/evidence`) directly.

**Resolution needed:** Review `v2/pkg/evidence/evidence.go` — is evidence attached at step level or run level?

**Assigned to Brian:** Document evidence API before Phase 3.

---

## 9. Success Criteria for v0

Package design is successful if Brian can scaffold all 4 packages (`model`, `loader`, `compiler`, `delegation`) in Week 1 and pass Phase 1 exit criteria.

Compilation design is successful if Barbara's integration test (Phase 4) executes a routine from `property-full.yaml` end-to-end: load → compile → run → timer wakes → human task created → evidence attached → timer resets.

**Architectural validation:** This design proves the domain kit model — thin authoring DSL over GERT primitives, zero runtime reimplementation. If this works, future kits (manufacturing, deployment, compliance) follow the same pattern.

---

## 10. Architectural Invariants

These must hold true across all implementation phases:

1. **No GERT types in model package** — `pkg/model` is pure domain vocabulary. ExecutionPlan, Run, Step types appear ONLY in `pkg/compiler`.

2. **Loader is GERT-agnostic** — `pkg/loader` outputs model types, never GERT types. It does not know what an ExecutionPlan is.

3. **Compiler is deterministic** — Same Property input → same ExecutionPlan output. No randomness, no external API calls, no current-time evaluation during compilation (time evaluation happens at runtime, not compile time).

4. **Delegation is policy, not state** — `pkg/delegation` evaluates policy ("should this task route to delegate?") but does not mutate Property or Delegation models. It's a pure function over time + delegation config.

5. **Evidence is append-only** — Home kit never deletes or edits evidence. Wrapper types in `pkg/model/evidence.go` are read-only views over GERT's append-only evidence log.

---

**END OF DESIGN**

Next step: Brian scaffolds directory structure and implements Phase 1 (model + loader).
