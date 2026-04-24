# gert-domain-home v0 Scaffold — Implementation Notes

**Author:** Brian (Go Programmer)  
**Date:** 2025-01-26  
**Phase:** v0 Scaffold

---

## What Was Created

Created the `domains/home/` Go module with the following structure:

```
domains/home/
├── go.mod                              # Module definition
├── go.sum                              # Dependencies (gopkg.in/yaml.v3)
├── doc.go                              # Package documentation
├── pkg/
│   ├── model/
│   │   ├── duration.go                 # Custom Duration type with YAML marshaling
│   │   └── model.go                    # All domain structs (Property, Zone, Asset, Routine, etc.)
│   ├── loader/
│   │   └── loader.go                   # YAML file loading with validation
│   └── delegation/
│       └── policy.go                   # Away mode policy functions
└── examples/
    └── casa-santiago.home.yaml         # Full example property file
```

### Module Setup

- **Module path:** `github.com/ormasoftchile/gert-domain-home`
- **Go version:** 1.23 (aligned with v2 core)
- **Dependencies:** Only `gopkg.in/yaml.v3` (no gert v2 core imports yet)
- **Workspace:** Added to `go.work` as `./domains/home`

### Model Package

Implemented all domain concepts from the v0 spec (section 2) and DSL spec:

- **Property** — top-level container with zones
- **Zone** — named area within property
- **Asset** — physical equipment with maintenance tracking
- **Routine** — recurring scheduled task (fixed or seasonal cadence)
- **Cadence** — scheduling configuration (Every or Seasonal)
- **SeasonalSchedule** — season-specific intervals
- **Evidence** — evidence requirement (none/note/photo/checklist)
- **IncidentTemplate** — reactive repair workflow
- **IncidentStep** — workflow step (human_task/decision/parallel)
- **Consumable** — item replaced on schedule
- **Delegation** — away mode configuration
- **DelegateConfig** — delegate contact info
- **DelegationAssignment** — routine/zone assignment
- **DelegationPermissions** — delegate capabilities

All structs have:
- Proper YAML struct tags matching DSL field names (snake_case)
- `omitempty` where appropriate
- Doc comments on all exported types and fields
- Zero values that make sense

### Duration Type

Custom `Duration` type with:
- Parsing: "7d", "2w", "3M", "1y" → struct with value + unit
- `UnmarshalYAML` / `MarshalYAML` for YAML integration
- `ToDays()` method (approximate: w=7, M=30, y=365)
- `String()` method for canonical representation
- Validation of units (d/w/M/y only)

### Loader Package

Implements:
- `Load(path string)` — reads and parses .home.yaml file
- `LoadBytes(data []byte)` — parses from bytes
- Comprehensive validation:
  - Required fields: property.name, at least one zone
  - Unique IDs: zones, assets, routines, templates
  - Reference integrity: asset.zone, routine.zone/asset, delegation assignments
  - Cadence constraints: exactly one of every/seasonal
  - Incident step dependencies: no undefined step references
  - Delegation constraints: active dates, assignment types
- Clear error messages with context (e.g., "routine pool_clean: cadence.every or cadence.seasonal is required")

### Delegation Package

Policy functions for away mode:
- `IsAwayModeActive(d, now)` — checks if delegation window is active
- `RoutineIsAssignedToDelegate(d, routineID)` — checks routine assignment
- `GetDelegateForRoutine(d, routineID, now)` — returns delegate if active + assigned
- Added `ZoneIsAssignedToDelegate` and `GetDelegateForZone` helpers

### Example File

`casa-santiago.home.yaml` — realistic example with:
- 5 zones (pool, front_lawn, backyard, garage, driveway)
- 3 assets (pool_pump, riding_mower, front_gate)
- 5 routines (pool_clean, pool_filter_backwash, lawn_mow, water_plants, trash_day)
- 2 consumables (pool_filter_sand, smoke_detector_batteries)
- 2 incident templates (leak, broken_hardware)
- Delegation configuration (Carlos Méndez, 1 week in April)

**Validation:** Loaded and parsed successfully with all validations passing.

---

## What's Deferred to Phase 2

### Compiler Package

The compiler that translates domain models to GERT core primitives is deferred because:

1. **Avoid circular dependencies:** The compiler will need to import `github.com/ormasoftchile/gert/v2` types (Run, Task, Timer, Policy, etc). During early scaffolding, keeping the domain kit independent avoids workspace complexity.

2. **Core API surface unclear:** The v2 package structure exists but the exact constructor APIs and builder patterns for creating runs/tasks/timers are still evolving.

3. **Phase boundary:** v0 scaffold focuses on "can we load and validate the DSL?" Phase 2 will tackle "can we compile it to GERT primitives?"

When implemented, the compiler package will:
- Live at `domains/home/pkg/compiler/`
- Export `Compile(*model.PropertyFile) (*gert.Graph, error)`
- Translate each routine → timer-backed run with human tasks
- Translate each incident template → ad-hoc run graph
- Translate delegation → time-bounded policy

---

## Go-Specific Design Decisions

### 1. Zero Values Are Valid

All structs have sensible zero values:
- `Evidence{Type: ""}` defaults to "none" in practice
- `Delegation == nil` means no delegation (not an error)
- Empty slices (Zones, Assets, Routines) are valid during construction (validated at load time)

This aligns with Go idiom: invalid states are caught by validation, not by impossible construction.

### 2. Pointer vs Value Semantics

Used pointers for:
- **Optional structs:** `*Delegation`, `*Executor`, `*Notifications` (nil = not present)
- **Return values:** `GetDelegateForRoutine` returns `*DelegateConfig` (nil = not assigned)

Used values for:
- **Required structs:** `Property`, `Cadence`, `Evidence` (always present, even if empty)
- **Slices:** Always use value types (nil slice == empty slice in Go)

### 3. Validation in Loader, Not Model

The model package has NO validation logic (except Duration parsing). All validation lives in the loader. This keeps model types simple and testable (you can construct invalid states for testing).

### 4. No Global State

No global registries, no init() functions, no mutable package-level variables. All functions are pure or take explicit parameters.

### 5. Error Messages Include Context

Validation errors include the field path:
- `"routine pool_clean: cadence.every or cadence.seasonal is required"`
- `"delegation.assigns[0]: routine or zone is required"`

This makes debugging DSL files easier for non-technical users.

### 6. YAML Tags Match DSL Exactly

All struct tags use snake_case matching the DSL spec:
- `replace_every` (not `replaceEvery`)
- `remind_hours_before` (not `remindHoursBefore`)
- `can_report_incidents` (not `canReportIncidents`)

This ensures zero friction between DSL and Go types.

---

## Issues and Open Questions

### 1. Seasonal Cadence Hemisphere Detection

The DSL spec says:
> "Property hemisphere (inferred from locale or explicit configuration)"

**Issue:** The `Property` struct has no `locale` or `hemisphere` field yet.

**Options:**
- Add `Property.Metadata["hemisphere"]` (v0)
- Add explicit `Property.Hemisphere` field (requires schema update)
- Infer from timezone/address during compilation (requires external geocoding)

**Decision:** Deferred to compiler phase. For v0, we have the schema to represent seasonal schedules, but hemisphere resolution is a runtime concern.

### 2. Date Validation

Currently, dates are stored as strings (`"2023-06-15"`). The loader does NOT validate date formats.

**Options:**
- Add date format validation in loader (regex or time.Parse)
- Use custom Date type with UnmarshalYAML (more type safety)
- Defer to compiler (dates only matter at runtime)

**Decision:** Deferred. The model accepts strings; the compiler will parse them when creating timers.

### 3. Phone Number Validation

The DSL spec requires E.164 format for `delegate.contact` (e.g., "+56 9 8765 4321").

**Current:** No validation.

**Decision:** Deferred. Phone validation is non-trivial (country codes, formatting variations). The compiler or runtime can validate before sending notifications.

### 4. Circular Dependency Detection in Steps

The loader validates that `depends_on` references valid step IDs, but does NOT detect circular dependencies.

**Example (invalid):**
```yaml
steps:
  - id: a
    depends_on: [b]
  - id: b
    depends_on: [a]
```

**Decision:** Deferred to compiler. Detecting cycles requires graph traversal; the compiler will do this when building the execution graph.

### 5. Decision Step Choice Validation

The DSL shows decision steps with `choices[]` containing `next_step`, but the model doesn't validate that `next_step` references a valid step ID.

**Decision:** Deferred to compiler (same as circular dependency detection).

---

## Next Steps for Phase 2

1. **Design compiler API:**
   - Sketch out `compiler.Compile(*model.PropertyFile) (*v2.SomethingTBD, error)`
   - Determine what the compilation target is (Graph? Run? Builder?)

2. **Import v2 core types:**
   - Update `go.mod` to require `github.com/ormasoftchile/gert/v2`
   - Study existing builder/constructor patterns in v2

3. **Implement routine compilation:**
   - Translate `Routine` → timer-backed run
   - Handle fixed vs seasonal cadence
   - Generate human task nodes with evidence

4. **Implement incident compilation:**
   - Translate `IncidentTemplate` → ad-hoc run graph
   - Handle step dependencies (topological sort)
   - Wire up decision/parallel step types

5. **Implement delegation compilation:**
   - Translate `Delegation` → time-bounded policy
   - Generate delegate-scoped projections

6. **Write compiler tests:**
   - Load casa-santiago.home.yaml
   - Compile to v2 graph
   - Assert expected structure

---

## Summary

The v0 scaffold is complete and functional:
- ✓ Module structure created
- ✓ All domain types modeled
- ✓ YAML loading with validation
- ✓ Delegation policy helpers
- ✓ Example file loads successfully
- ✓ No circular dependencies with v2 core

The domain kit can now load and validate `.home.yaml` files. The next phase will implement the compiler to translate these files into GERT runtime primitives.
