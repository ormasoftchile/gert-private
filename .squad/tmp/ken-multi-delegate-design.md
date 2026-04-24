# Multi-Delegate Support Design for Home Domain Kit

**Author:** Ken (Software Architect)  
**Date:** 2026-04-24  
**Status:** Design Review  
**Target:** Brian (Go Implementation)

---

## 1. Executive Summary

This design extends the Home Domain Kit to support **multiple concurrent delegates**, enabling task distribution across different people during the same time period (e.g., son handles garage tasks, daughter handles garden tasks).

**Current limitation:** Single `delegation:` block assigns all tasks to one delegate.  
**New capability:** Multiple `delegations:` list assigns different task sets to different delegates.

---

## 2. YAML DSL Change

### 2.1 Current DSL (Single Delegate)

```yaml
delegation:
  delegate:
    name: Carlos
    contact: "+56 9 8765 4321"
  active:
    from: 2025-04-25
    to: 2025-05-02
  assigns:
    - routine: trash_day
    - zone: pool
  permissions:
    can_report_incidents: true
```

### 2.2 New DSL (Multiple Delegates)

```yaml
delegations:
  - delegate:
      name: Son
      contact: "+1 555 0100"
      email: son@example.com
    active:
      from: 2025-04-25
      to: 2025-05-02
    assigns:
      - routine: trash_day
      - zone: garage
    permissions:
      can_report_incidents: true
      can_modify_routines: false
    notifications:
      remind_delegate_hours_before: 12
      notify_owner_on_completion: true
  
  - delegate:
      name: Daughter
      contact: "+1 555 0101"
      email: daughter@example.com
    active:
      from: 2025-04-25
      to: 2025-05-02
    assigns:
      - routine: water_plants
      - zone: garden
    permissions:
      can_report_incidents: true
      can_modify_routines: false
    notifications:
      remind_delegate_hours_before: 12
```

**Key properties:**
- `delegations:` is a list (plural)
- Each item is a complete delegation configuration
- Each delegation has independent: delegate info, time window, assignments, permissions
- Multiple delegates can have overlapping time windows

---

## 3. Go Model Changes

### 3.1 Modified Structs

**File:** `domains/home/pkg/model/model.go`

#### Change 1: Add `Delegations` field to `PropertyFile`

```go
// PropertyFile represents the complete .home.yaml file structure.
type PropertyFile struct {
	// Property is the top-level household definition.
	Property Property `yaml:"property"`

	// Assets are the tracked physical items.
	Assets []Asset `yaml:"assets,omitempty"`

	// Routines are the recurring maintenance tasks.
	Routines []Routine `yaml:"routines,omitempty"`

	// IncidentTemplates define reactive repair workflows.
	IncidentTemplates []IncidentTemplate `yaml:"incident_templates,omitempty"`

	// Consumables are items replaced on a schedule.
	Consumables []Consumable `yaml:"consumables,omitempty"`

	// Delegation activates away mode (DEPRECATED: use Delegations).
	Delegation *Delegation `yaml:"delegation,omitempty"`

	// Delegations activates multi-delegate away mode.
	Delegations []Delegation `yaml:"delegations,omitempty"`
}
```

**Notes:**
- Keep `Delegation` for backward compatibility (deprecated)
- Add `Delegations` as new field (plural)
- Both can coexist during transition
- Validation rule: EITHER `delegation:` OR `delegations:`, NOT both

#### Change 2: No changes to `Delegation` struct

The existing `Delegation` struct is already perfect for multi-delegate use:

```go
// Delegation represents a single delegation configuration.
type Delegation struct {
	// Delegate is the delegate information.
	Delegate DelegateConfig `yaml:"delegate"`

	// Active is the delegation time window.
	Active ActiveWindow `yaml:"active"`

	// Assigns lists the routine/zone assignments.
	Assigns []DelegationAssignment `yaml:"assigns"`

	// Permissions defines what the delegate can/cannot do.
	Permissions DelegationPermissions `yaml:"permissions,omitempty"`

	// Notifications defines notification preferences.
	Notifications *DelegationNotifications `yaml:"notifications,omitempty"`
}
```

No structural changes needed — this design already supports individual delegation configs.

---

## 4. Conflict Resolution Rules

### 4.1 The Problem

What happens when two delegates are assigned the same routine or zone?

**Example conflict:**

```yaml
delegations:
  - delegate:
      name: Son
    assigns:
      - routine: trash_day
  
  - delegate:
      name: Daughter
    assigns:
      - routine: trash_day  # CONFLICT: same routine assigned twice
```

### 4.2 Resolution Rule: **Last-Wins (Array Order)**

**Decision:** Use **deterministic array order** to resolve conflicts.

- **Rule:** If multiple delegations assign the same routine ID, the **last delegation in the array wins**.
- **Rationale:**
  - Simple to implement: single pass through array builds assignment map
  - Predictable: no hidden priority logic
  - Debuggable: user can see order in YAML file
  - Common YAML convention (ConfigMaps, Helm values)

**Implementation logic (pseudocode):**

```go
// Build final assignment map (routine → delegate)
assignments := make(map[string]string) // routine_id → delegate_name

for _, delegation := range propertyFile.Delegations {
    for _, assign := range delegation.Assigns {
        if assign.Routine != "" {
            assignments[assign.Routine] = delegation.Delegate.Name  // Overwrites if exists
        }
        if assign.Zone != "" {
            // Resolve zone to all routines in that zone
            for _, routine := range propertyFile.Routines {
                if routine.Zone == assign.Zone {
                    assignments[routine.ID] = delegation.Delegate.Name
                }
            }
        }
    }
}
```

**Result:** The last delegation to claim a routine becomes its sole owner.

### 4.3 Error Detection vs. Silent Overwrite

**Design choice:** **Silent overwrite** (no error on conflict)

**Why:**
- Zone assignments can legitimately overlap (e.g., `zone: garage` + `routine: garage_sweep` both valid)
- Detecting conflicts requires complex pre-flight analysis
- User can easily fix ordering if delegation goes to wrong person

**Alternative (rejected):** Explicit `priority` field
```yaml
delegation:
  priority: 100  # Higher number wins
```
**Why rejected:** Adds complexity; ordering is simpler and sufficient.

### 4.4 User-Facing Documentation

**SUMMARY.md must document:**

> **Multi-Delegate Conflict Resolution:**  
> If multiple delegations assign the same routine (either explicitly or via zone assignment), the **last delegation in the array** takes precedence. To change which delegate receives a routine, reorder the `delegations:` list.

---

## 5. Compiler Changes

### 5.1 Current Compiler Behavior

**File:** `domains/home/pkg/compiler/compiler.go`

**Current implementation (lines 116-122):**

```go
// Compile delegation if present
if prop.Delegation != nil {
    policyDef, err := c.CompileDelegation(prop, prop.Delegation)
    if err != nil {
        return nil, fmt.Errorf("compile delegation: %w", err)
    }
    compiled.Delegation = policyDef
}
```

**Output:** Single `PolicyDefinition` struct in `CompiledProperty.Delegation`

### 5.2 New Compiler Behavior

**Change 1:** `CompiledProperty` struct gains `Delegations` field

```go
// CompiledProperty represents the full compiled output for a property.
type CompiledProperty struct {
	// PropertyID is the property identifier.
	PropertyID string

	// Routines contains one RunDefinition per routine.
	Routines []*RunDefinition

	// IncidentTemplates contains one RunDefinition per incident template.
	IncidentTemplates []*RunDefinition

	// Delegation contains the compiled policy definition (DEPRECATED: use Delegations).
	Delegation *PolicyDefinition

	// Delegations contains one PolicyDefinition per delegation (plural).
	Delegations []*PolicyDefinition
}
```

**Change 2:** `CompileProperty()` loops over `Delegations` slice

```go
// CompileProperty compiles the full property into a set of GERT run definitions.
func (c *Compiler) CompileProperty(prop *model.PropertyFile) (*CompiledProperty, error) {
	compiled := &CompiledProperty{
		PropertyID:        c.PropertyID,
		Routines:          make([]*RunDefinition, 0, len(prop.Routines)),
		IncidentTemplates: make([]*RunDefinition, 0, len(prop.IncidentTemplates)),
		Delegations:       make([]*PolicyDefinition, 0),
	}

	// Compile all routines
	for i := range prop.Routines {
		runDef, err := c.CompileRoutine(prop, &prop.Routines[i])
		if err != nil {
			return nil, fmt.Errorf("compile routine %q: %w", prop.Routines[i].ID, err)
		}
		compiled.Routines = append(compiled.Routines, runDef)
	}

	// Compile all incident templates
	for i := range prop.IncidentTemplates {
		runDef, err := c.CompileIncidentTemplate(prop, &prop.IncidentTemplates[i])
		if err != nil {
			return nil, fmt.Errorf("compile incident template %q: %w", prop.IncidentTemplates[i].ID, err)
		}
		compiled.IncidentTemplates = append(compiled.IncidentTemplates, runDef)
	}

	// Backward compatibility: compile single delegation (deprecated)
	if prop.Delegation != nil {
		if len(prop.Delegations) > 0 {
			return nil, fmt.Errorf("cannot specify both 'delegation' and 'delegations' (use 'delegations' only)")
		}
		policyDef, err := c.CompileDelegation(prop, prop.Delegation)
		if err != nil {
			return nil, fmt.Errorf("compile delegation: %w", err)
		}
		compiled.Delegation = policyDef
		compiled.Delegations = []*PolicyDefinition{policyDef}  // Mirror to new field
	}

	// Compile all delegations (plural)
	for i := range prop.Delegations {
		policyDef, err := c.CompileDelegation(prop, &prop.Delegations[i], i)
		if err != nil {
			return nil, fmt.Errorf("compile delegation %d: %w", i, err)
		}
		compiled.Delegations = append(compiled.Delegations, policyDef)
	}

	return compiled, nil
}
```

**Change 3:** `CompileDelegation()` gains index parameter for unique IDs

```go
// CompileDelegation compiles Delegation config into a GERT policy definition.
// Index parameter is used to create unique policy IDs when multiple delegations exist.
func (c *Compiler) CompileDelegation(prop *model.PropertyFile, d *model.Delegation, index int) (*PolicyDefinition, error) {
	var policyID string
	if index == -1 {
		// Backward compat: single delegation (deprecated path)
		policyID = fmt.Sprintf("%s.delegation", c.PropertyID)
	} else {
		// Multi-delegate: indexed IDs
		policyID = fmt.Sprintf("%s.delegation.%d", c.PropertyID, index)
	}

	// Parse dates
	from, err := time.Parse("2006-01-02", d.Active.From)
	if err != nil {
		return nil, fmt.Errorf("parse active.from: %w", err)
	}

	to, err := time.Parse("2006-01-02", d.Active.To)
	if err != nil {
		return nil, fmt.Errorf("parse active.to: %w", err)
	}

	// Resolve all assigned routines
	assignedRoutines := make([]string, 0)
	for _, assign := range d.Assigns {
		if assign.Routine != "" {
			assignedRoutines = append(assignedRoutines, assign.Routine)
		} else if assign.Zone != "" {
			for i := range prop.Routines {
				if prop.Routines[i].Zone == assign.Zone {
					assignedRoutines = append(assignedRoutines, prop.Routines[i].ID)
				}
			}
		}
	}

	return &PolicyDefinition{
		ID:               policyID,
		ActiveFrom:       from,
		ActiveTo:         to,
		DelegateName:     d.Delegate.Name,
		DelegateContact:  d.Delegate.Contact,
		AssignedRoutines: assignedRoutines,
		Permissions:      d.Permissions,
	}, nil
}
```

**Backward compat caller:**

```go
// For backward compat, existing call sites can use index = -1
policyDef, err := c.CompileDelegation(prop, prop.Delegation, -1)
```

### 5.3 Output: Multiple `PolicyDefinition` Structs

**Before (single delegation):**

```go
CompiledProperty{
    Delegation: &PolicyDefinition{
        ID: "casa-santiago.delegation",
        DelegateName: "Carlos",
        AssignedRoutines: ["trash_day", "pool_clean"],
    },
}
```

**After (multiple delegations):**

```go
CompiledProperty{
    Delegations: []*PolicyDefinition{
        {
            ID: "casa-santiago.delegation.0",
            DelegateName: "Son",
            AssignedRoutines: ["trash_day", "garage_sweep"],
        },
        {
            ID: "casa-santiago.delegation.1",
            DelegateName: "Daughter",
            AssignedRoutines: ["water_plants", "garden_weed"],
        },
    },
}
```

**Each delegation becomes an independent policy** — GERT runtime can route tasks based on policy ID + time window.

---

## 6. Validation Rules

Add to loader validation (future work for Brian):

1. **Mutual exclusion:** Reject if BOTH `delegation:` and `delegations:` are present

   ```go
   if prop.Delegation != nil && len(prop.Delegations) > 0 {
       return fmt.Errorf("cannot specify both 'delegation' (singular) and 'delegations' (plural); use 'delegations' only")
   }
   ```

2. **Delegate name uniqueness:** Warn (or error) if two delegations have the same delegate name within overlapping time windows

   ```go
   // Implementation: build map[delegateName][]ActiveWindow and check for overlaps
   // Decision: WARN only (not error) — same person might legitimately get two different assignment sets
   ```

3. **Assignment referential integrity:** Validate that all `routine:` IDs exist, all `zone:` IDs exist (already implemented in current loader)

4. **Time window validation:** Ensure `from` ≤ `to` (already implemented)

---

## 7. SUMMARY.md Updates

### 7.1 Section 3 — Pattern 6 Update

**File:** `specs/gert-domain-home/SUMMARY.md` (lines 206-236)

**Change:** Replace single delegation example with multi-delegate example

**New Pattern 6 content:**

```markdown
### Pattern 6: Multiple Delegations (Away Mode)

Assign different tasks to multiple delegates during the same period:

```yaml
delegations:
  - delegate:
      name: Son
      contact: "+1 555 0100"
      email: son@example.com
    active:
      from: 2025-04-25
      to: 2025-05-02
    assigns:
      - routine: trash_day
      - zone: garage          # All garage-scoped routines
    permissions:
      can_report_incidents: true
      can_modify_routines: false
    notifications:
      remind_delegate_hours_before: 12
      notify_owner_on_completion: true
  
  - delegate:
      name: Daughter
      contact: "+1 555 0101"
    active:
      from: 2025-04-25
      to: 2025-05-02
    assigns:
      - routine: water_plants
      - zone: garden
    permissions:
      can_report_incidents: true
    notifications:
      remind_delegate_hours_before: 12
```

- Each delegation is a complete assignment: delegate info, time window, tasks, permissions
- Multiple delegates can have overlapping time windows
- **Conflict resolution:** If multiple delegations assign the same routine, the **last delegation in the array wins**
- Each assignment is either `routine: <id>` or `zone: <id>` (mutually exclusive within one assignment)
- `zone: pool` assigns ALL routines scoped to that zone

**Backward compatibility:** The singular `delegation:` field is deprecated but still supported:

```yaml
delegation:  # DEPRECATED: use delegations: instead
  delegate:
    name: Carlos
  assigns:
    - routine: trash_day
```
```

### 7.2 Section 4.3 Update — Compilation Model

**File:** `specs/gert-domain-home/SUMMARY.md` (lines 365-395)

**Change:** Update to show multiple `PolicyDefinition` outputs

**New §4.3 content:**

```markdown
### 4.3 Delegations → Multiple Policy Definitions

**Input:** Two delegations with different assignments

**Output:** List of `PolicyDefinition` structs (one per delegation):

```go
[]*PolicyDefinition{
    {
        ID:               "casa-santiago.delegation.0",
        ActiveFrom:       time.Parse("2025-04-25"),
        ActiveTo:         time.Parse("2025-05-02"),
        DelegateName:     "Son",
        DelegateContact:  "+1 555 0100",
        AssignedRoutines: []string{
            "trash_day",
            "garage_sweep",       // zone: garage
            "garage_organize",    // zone: garage
        },
        Permissions: DelegationPermissions{
            CanReportIncidents: true,
            CanModifyRoutines:  false,
        },
    },
    {
        ID:               "casa-santiago.delegation.1",
        ActiveFrom:       time.Parse("2025-04-25"),
        ActiveTo:         time.Parse("2025-05-02"),
        DelegateName:     "Daughter",
        DelegateContact:  "+1 555 0101",
        AssignedRoutines: []string{
            "water_plants",
            "garden_weed",        // zone: garden
            "garden_fertilize",   // zone: garden
        },
        Permissions: DelegationPermissions{
            CanReportIncidents: true,
        },
    },
}
```

**Key behaviors:**
- Compiler resolves `zone: pool` to ALL routines with `zone: pool`
- Each delegation becomes an independent policy with unique ID (`delegation.0`, `delegation.1`, etc.)
- If multiple delegations assign the same routine, **last delegation in array wins** (conflict resolution via deterministic ordering)
- During `[ActiveFrom, ActiveTo]` window, GERT policy routes each routine to its assigned delegate
- Projection filters property run graph to show delegate-scoped tasks
```

### 7.3 Section 7 Update — Status

**File:** `specs/gert-domain-home/SUMMARY.md` (line 599)

**Change:** Add multi-delegate to implemented features

```markdown
**Domain Model:**
- Full DSL types in `pkg/model/model.go`
- Duration parsing (`7d`, `2w`, `3M`, `1y`)
- Evidence types (none, note, photo, checklist)
- Seasonal cadence (all four seasons required)
- Delegation with zone/routine assignment resolution
- **Multi-delegate support (multiple concurrent delegations)**  ← ADD THIS LINE
```

---

## 8. Backward Compatibility Strategy

### 8.1 Deprecation Path

**Phase 1 (v0.2):** Add `delegations:` support, keep `delegation:` working
- Both fields work
- Validation error if BOTH specified
- Documentation marks `delegation:` as deprecated

**Phase 2 (v1.0):** Remove `delegation:` field
- Breaking change
- Migration guide: rename `delegation:` → `delegations:` + wrap in array

### 8.2 Recommended: Keep `delegation:` as Syntactic Sugar

**Decision:** **YES — keep `delegation:` indefinitely as sugar for single-item `delegations:` list**

**Rationale:**
- Most households have only one delegate at a time
- `delegation:` is cleaner UX for 90% case
- No runtime cost: compiler treats both uniformly
- Loader can normalize: if `delegation:` present → convert to `delegations: [...]` internally

**Implementation:**

```go
// In loader (or compiler init):
if prop.Delegation != nil {
    if len(prop.Delegations) > 0 {
        return fmt.Errorf("cannot use both 'delegation' and 'delegations'")
    }
    // Normalize: convert singular to plural
    prop.Delegations = []Delegation{*prop.Delegation}
    prop.Delegation = nil  // Clear deprecated field
}
```

**User-facing:** Both syntaxes documented, `delegation:` marked as convenience shorthand.

---

## 9. Testing Strategy (for Brian)

### 9.1 Unit Tests (compiler_test.go)

**New test cases:**

1. `TestCompileMultipleDelegations` — 2 delegations with non-overlapping assignments
2. `TestDelegationConflictLastWins` — 2 delegations assign same routine, verify last wins
3. `TestDelegationZoneResolutionMultiple` — Each delegate gets different zone
4. `TestDelegationPolicyIDsUnique` — Verify `delegation.0`, `delegation.1`, etc.
5. `TestBackwardCompatSingleDelegation` — Old `delegation:` field still works
6. `TestMutualExclusionError` — Error if both `delegation:` and `delegations:` present

### 9.2 Integration Tests

**File:** `domains/home/integration_test.go`

**New test:**

```go
func TestCompileMultiDelegateCasaSantiago(t *testing.T) {
    // Load .home.yaml with 2 delegations
    // Compile to GERT
    // Verify CompiledProperty.Delegations has 2 PolicyDefinitions
    // Verify each has correct AssignedRoutines list
    // Verify unique IDs (delegation.0, delegation.1)
}
```

### 9.3 Golden File Tests

**New example file:** `testdata/multi-delegate.home.yaml`

```yaml
property:
  name: Multi-Delegate Test House
  zones:
    - id: garage
    - id: garden

routines:
  - id: trash_day
    label: Take out trash
    cadence:
      every: 7d
  - id: garage_sweep
    label: Sweep garage
    zone: garage
    cadence:
      every: 14d
  - id: water_plants
    label: Water plants
    zone: garden
    cadence:
      every: 3d

delegations:
  - delegate:
      name: Son
      contact: "+1 555 0100"
    active:
      from: 2025-04-25
      to: 2025-05-02
    assigns:
      - routine: trash_day
      - zone: garage
  - delegate:
      name: Daughter
      contact: "+1 555 0101"
    active:
      from: 2025-04-25
      to: 2025-05-02
    assigns:
      - zone: garden
```

---

## 10. Runtime Integration Notes (Future Work)

**NOT part of v0.2 implementation** — documentation for future GERT runtime team:

### 10.1 Policy Routing

GERT runtime will need to:
1. Load all `PolicyDefinition` structs for a property
2. For each routine execution request:
   - Check current time against all `[ActiveFrom, ActiveTo]` windows
   - Find matching policies (may be multiple if windows overlap)
   - Route to correct delegate based on policy's `AssignedRoutines` list
3. Handle delegation expiry (revert to owner after `ActiveTo`)

### 10.2 Projection API

Mobile app endpoint needs:
- `GET /properties/{id}/tasks?delegate={name}` → filter runs to show only assigned tasks
- `GET /properties/{id}/delegations?active=true` → list current active delegations

### 10.3 Conflict Handling at Runtime

If two delegations with overlapping windows both claim the same routine:
- Use **last-wins** rule (index-based precedence)
- Log warning: "Routine 'trash_day' assigned to both Son (delegation.0) and Daughter (delegation.1); routing to Daughter (last wins)"

---

## 11. Decision Summary

| Aspect | Decision | Rationale |
|--------|----------|-----------|
| **Field name** | `delegations:` (plural) | Array of delegation configs |
| **Struct changes** | Add `[]Delegation` field to `PropertyFile`; reuse existing `Delegation` struct | Minimal model change |
| **Conflict resolution** | Last-wins (array order) | Simple, deterministic, debuggable |
| **Error vs. silent overwrite** | Silent overwrite | Zone assignments can legitimately overlap |
| **Backward compat** | Keep `delegation:` as sugar | Better UX for single-delegate case (90% use) |
| **Policy ID format** | `{property}.delegation.{index}` | Unique ID per delegation |
| **Compiler output** | `[]*PolicyDefinition` (list) | One policy per delegation |

---

## 12. Implementation Checklist for Brian

- [ ] Add `Delegations []Delegation` field to `PropertyFile` struct
- [ ] Add mutual exclusion validation (error if both `delegation` and `delegations` present)
- [ ] Normalize `delegation` → `delegations` in loader (optional: syntactic sugar)
- [ ] Update `CompiledProperty` struct: add `Delegations []*PolicyDefinition`
- [ ] Modify `CompileProperty()` to loop over `prop.Delegations`
- [ ] Add `index` parameter to `CompileDelegation()`
- [ ] Generate unique policy IDs: `{property}.delegation.{index}`
- [ ] Write 6 new unit tests (see §9.1)
- [ ] Add integration test for multi-delegate compilation
- [ ] Create `testdata/multi-delegate.home.yaml` golden file
- [ ] Update SUMMARY.md §3 Pattern 6 (delegation example)
- [ ] Update SUMMARY.md §4.3 (compilation model)
- [ ] Update SUMMARY.md §7 (status: add multi-delegate feature)
- [ ] Update godoc comments for `Delegation`, `PropertyFile`, `CompileDelegation`

---

## 13. Open Questions for Cristian

1. **Notification preferences:** Should we support per-delegate notification settings (already in model), or should notifications always go to owner only?
   - Current design: each delegation has its own `notifications:` block ✅
   
2. **Overlapping windows with same delegate:** Should we allow the same person to have multiple delegations (e.g., different permissions for different zones)?
   - Current design: allows this (validates by name+window overlap with WARNING only)
   
3. **Mobile app UX:** How should UI show "2 active delegations" — separate task lists or merged with delegate badges?
   - Not a compiler concern, but affects PolicyDefinition metadata needs

---

**END OF DESIGN**
