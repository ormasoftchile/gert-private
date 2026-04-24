# Section 3: Authoring Model / DSL

## Overview

The gert-domain-home v0 DSL is a YAML-based authoring interface for household maintenance orchestration. It enables homeowners to declare their property structure, maintenance routines, incident response templates, and delegation rules in a clean, readable format that compiles deterministically to GERT execution graph nodes.

**Design Principles:**
- **Readability First:** Non-technical household owners should understand the structure
- **Deterministic Compilation:** Each DSL construct maps to a well-defined GERT primitive
- **Forward-Compatible:** v0 fields establish extension points for v1 features
- **Convention-Aligned:** Consistent with GERT's existing YAML conventions and JSON Schema 2020-12

---

## 3.1 Property Definition

The `property:` block is the root container. It names the household and declares zones — the spatial partitions used for organizing assets and routines.

### Schema

```yaml
property:
  name: <string>                    # Display name for the household
  zones:                            # List of zone definitions
    - id: <identifier>              # Machine-readable zone identifier (kebab-case)
      label: <string>               # Human-readable zone name
      description: <string>         # Optional description
      metadata:                     # Optional extensible metadata
        <key>: <value>
```

### Field Definitions

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Display name for the household (e.g., "Casa Santiago") |
| `zones` | array | Yes | List of zone objects (minimum 1) |
| `zones[].id` | string | Yes | Unique zone identifier, kebab-case (e.g., "pool", "front_lawn") |
| `zones[].label` | string | No | Human-readable zone name (defaults to id with underscores replaced) |
| `zones[].description` | string | No | Optional description for the zone |
| `zones[].metadata` | object | No | Optional extension point for zone-specific metadata (v1: square footage, indoor/outdoor classification) |

### Constraints

- Zone IDs must be unique within the property
- Zone IDs must match `^[a-z][a-z0-9_]*$` (lowercase letters, numbers, underscores; must start with letter)
- Minimum 1 zone required

### Example

```yaml
property:
  name: Casa Santiago
  zones:
    - id: pool
      label: Swimming Pool
      description: Saltwater pool with heat pump
      metadata:
        area_sqm: 32
        type: outdoor

    - id: front_lawn
      label: Front Lawn
      
    - id: backyard
      label: Backyard Garden
      description: Vegetable garden and fruit trees
      
    - id: garage
      label: Garage
```

---

## 3.2 Asset Definition

Assets are physical things tracked for maintenance (appliances, vehicles, equipment, fixtures). Each asset belongs to exactly one zone and may have associated maintenance routines.

### Schema

```yaml
assets:
  - id: <identifier>                # Unique asset identifier
    zone: <zone_id>                 # Zone where the asset is located
    label: <string>                 # Human-readable asset name
    description: <string>           # Optional description
    installed: <date>               # Optional installation date (YYYY-MM-DD)
    warranty_expires: <date>        # Optional warranty expiration (YYYY-MM-DD)
    metadata:                       # Optional extensible metadata
      <key>: <value>
```

### Field Definitions

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique asset identifier (kebab-case) |
| `zone` | string | Yes | Must reference a declared zone ID |
| `label` | string | Yes | Human-readable asset name |
| `description` | string | No | Optional detailed description |
| `installed` | date | No | Installation date (ISO 8601 date: YYYY-MM-DD) |
| `warranty_expires` | date | No | Warranty expiration date (ISO 8601 date: YYYY-MM-DD) |
| `metadata` | object | No | Extension point for asset-specific data (v1: model, serial number, purchase price) |

### Constraints

- Asset IDs must be unique across all assets
- Asset IDs must match `^[a-z][a-z0-9_]*$`
- `zone` must reference a valid zone ID
- Dates must follow ISO 8601 format (YYYY-MM-DD)

### Example

```yaml
assets:
  - id: pool_pump
    zone: pool
    label: Hayward SuperPump
    description: 1.5 HP variable speed pump
    installed: 2023-06-15
    warranty_expires: 2025-06-15
    metadata:
      model: VS-100
      serial: HW23-45678

  - id: riding_mower
    zone: garage
    label: John Deere X350
    installed: 2022-04-10
    metadata:
      hours_at_install: 0

  - id: front_gate
    zone: front_lawn
    label: Driveway gate motor
    installed: 2020-11-03
```

---

## 3.3 Recurring Routines

Routines are scheduled maintenance tasks that recur on a fixed or seasonal cadence. Each routine generates a scheduled run at runtime.

### Schema

```yaml
routines:
  - id: <identifier>                # Unique routine identifier
    label: <string>                 # Human-readable routine name
    description: <string>           # Optional description
    zone: <zone_id>                 # Optional zone scope
    asset: <asset_id>               # Optional asset scope
    cadence:                        # Scheduling configuration (exactly one of every/seasonal)
      every: <duration>             # Fixed interval (e.g., "7d", "2w", "3M")
      seasonal:                     # Season-dependent intervals
        summer: <duration>
        autumn: <duration>
        winter: <duration>
        spring: <duration>
    evidence:                       # Evidence requirement
      type: <enum>                  # none | note | photo | checklist
      prompt: <string>              # Optional custom prompt for evidence
    executor:                       # Optional executor hint
      role: <string>                # Suggested executor role (v1: can require role)
    notifications:                  # Optional notification rules
      remind_hours_before: <int>    # Send reminder N hours before due
      escalate_hours_overdue: <int> # Escalate if not done N hours after due
```

### Field Definitions

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique routine identifier (kebab-case) |
| `label` | string | Yes | Human-readable routine name |
| `description` | string | No | Optional detailed description |
| `zone` | string | No | Zone scope (mutually exclusive with `asset`) |
| `asset` | string | No | Asset scope (mutually exclusive with `zone`) |
| `cadence.every` | duration | Conditional | Fixed interval (mutually exclusive with `seasonal`) |
| `cadence.seasonal` | object | Conditional | Season-specific intervals (mutually exclusive with `every`) |
| `evidence.type` | enum | No | Evidence requirement: `none`, `note`, `photo`, `checklist` (default: `none`) |
| `evidence.prompt` | string | No | Custom prompt for evidence capture |
| `executor.role` | string | No | Suggested executor role (v1: enforcement) |
| `notifications.remind_hours_before` | integer | No | Reminder lead time in hours |
| `notifications.escalate_hours_overdue` | integer | No | Escalation threshold in hours after due |

### Duration Format

Durations use compact notation:
- `7d` = 7 days
- `2w` = 2 weeks (14 days)
- `3M` = 3 months (90 days average; runtime adjusts to calendar month)
- `1y` = 1 year

### Seasonal Resolution

The runtime evaluates `seasonal` cadence based on:
- Current date
- Property hemisphere (inferred from locale or explicit configuration)
- Astronomical season boundaries (solstices/equinoxes)

Seasons:
- **Northern Hemisphere:** summer (Jun-Aug), autumn (Sep-Nov), winter (Dec-Feb), spring (Mar-May)
- **Southern Hemisphere:** summer (Dec-Feb), autumn (Mar-May), winter (Jun-Aug), spring (Sep-Nov)

### Constraints

- Routine IDs must be unique across all routines
- Exactly one of `zone` or `asset` may be specified (neither is also valid for property-wide routines)
- Exactly one of `cadence.every` or `cadence.seasonal` must be specified
- If `seasonal` is used, all four seasons must be defined
- Evidence type must be one of: `none`, `note`, `photo`, `checklist`

### Example

```yaml
routines:
  - id: pool_clean
    label: Pool cleaning
    description: Vacuum pool floor, clean walls, check chemical levels
    zone: pool
    cadence:
      every: 7d
    evidence:
      type: photo
      prompt: "Take photo of clear water and chemical test strip"
    notifications:
      remind_hours_before: 4

  - id: lawn_mow
    label: Lawn mowing
    zone: front_lawn
    cadence:
      seasonal:
        summer: 7d
        autumn: 10d
        winter: 21d
        spring: 10d
    evidence:
      type: photo

  - id: water_plants
    label: Water backyard plants
    zone: backyard
    cadence:
      every: 3d
    evidence:
      type: none
    notifications:
      remind_hours_before: 2

  - id: change_pool_filter
    label: Replace pool sand filter
    asset: pool_pump
    cadence:
      every: 6M
    evidence:
      type: note
      prompt: "Record filter brand and installation date"
    executor:
      role: homeowner

  - id: trash_day
    label: Take trash bins to curb
    description: Both recycling and general waste
    cadence:
      every: 7d
    evidence:
      type: none
    notifications:
      remind_hours_before: 12
      escalate_hours_overdue: 2
```

---

## 3.4 Incident Templates

Incident templates define the structure of reactive repair runs. When an unexpected problem occurs (leak, equipment failure, damage), the homeowner triggers an incident, which instantiates the template steps.

### Schema

```yaml
incident_templates:
  - id: <identifier>                # Unique template identifier
    label: <string>                 # Human-readable incident type
    description: <string>           # Optional description
    zones: [<zone_id>, ...]         # Optional applicable zones filter
    assets: [<asset_id>, ...]       # Optional applicable assets filter
    steps:                          # Incident workflow steps
      - id: <identifier>            # Step identifier (unique within template)
        type: <step_type>           # Step type (human_task | decision | parallel)
        label: <string>             # Human-readable step name
        description: <string>       # Optional step instructions
        evidence:                   # Evidence requirement
          type: <enum>              # none | note | photo | checklist
          prompt: <string>          # Optional evidence prompt
        depends_on: [<step_id>]     # Optional dependencies (explicit ordering)
        # type-specific fields
```

### Field Definitions

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique template identifier (kebab-case) |
| `label` | string | Yes | Human-readable incident type name |
| `description` | string | No | Optional detailed description |
| `zones` | array[string] | No | Filter: applicable zones (empty = all zones) |
| `assets` | array[string] | No | Filter: applicable assets (empty = all assets) |
| `steps` | array | Yes | Incident workflow steps (minimum 1) |
| `steps[].id` | string | Yes | Step identifier (unique within template, kebab-case) |
| `steps[].type` | enum | Yes | Step type: `human_task`, `decision`, `parallel` |
| `steps[].label` | string | Yes | Human-readable step name |
| `steps[].description` | string | No | Step instructions or guidance |
| `steps[].evidence.type` | enum | No | Evidence requirement (default: `none`) |
| `steps[].evidence.prompt` | string | No | Custom evidence prompt |
| `steps[].depends_on` | array[string] | No | Step IDs that must complete before this step starts |

### Step Types

**human_task:** Manual action required (the default for v0)
- User completes the task and submits evidence (if required)

**decision:** Binary or multiple-choice decision point
- Additional field: `choices` (array of choice options)
- Runtime presents choices and branches based on selection

**parallel:** Fan-out multiple sub-steps in parallel
- Additional field: `parallel_steps` (array of step definitions)
- All sub-steps execute concurrently; incident waits for all to complete

### Step Ordering

- If no `depends_on` is specified, steps execute in declaration order
- If `depends_on` is specified, step waits for all dependencies to complete
- Circular dependencies are rejected at compile time

### Incident Triggering

Incidents are triggered by:
1. **From YAML:** Declare an `incidents:` block in a run file with `template` reference
2. **From Mobile App:** User selects incident template and provides initial context
3. **From Scheduled Run:** A routine can trigger an incident if it detects a problem (v1 feature)

### Example

```yaml
incident_templates:
  - id: leak
    label: Water leak
    description: Emergency response for any water leak
    zones: [pool, backyard, garage]
    steps:
      - id: assess
        type: human_task
        label: Locate and assess the leak
        description: |
          1. Find the source of the leak
          2. Estimate severity (drip, stream, gush)
          3. Check if shut-off valve is accessible
        evidence:
          type: photo
          prompt: Take photo of leak source and affected area

      - id: shutoff_decision
        type: decision
        label: Can you safely shut off water?
        depends_on: [assess]
        choices:
          - value: "yes"
            label: "Yes, I can shut it off"
            next_step: shutoff
          - value: "no"
            label: "No, call plumber immediately"
            next_step: call_plumber

      - id: shutoff
        type: human_task
        label: Shut off water supply
        description: Turn the shut-off valve clockwise until fully closed
        evidence:
          type: photo
          prompt: Photo of closed valve

      - id: call_plumber
        type: human_task
        label: Call emergency plumber
        evidence:
          type: note
          prompt: Record plumber name, ETA, and job number

      - id: buy_parts
        type: human_task
        label: Purchase replacement parts
        description: Based on assessment, buy necessary parts (pipe, fittings, sealant)
        depends_on: [assess]
        evidence:
          type: note
          prompt: List parts purchased and cost

      - id: fix
        type: human_task
        label: Perform repair
        depends_on: [shutoff, buy_parts]
        evidence:
          type: photo
          prompt: Photo of completed repair

      - id: verify
        type: human_task
        label: Verify fix
        description: Turn water back on and monitor for 10 minutes
        depends_on: [fix]
        evidence:
          type: photo
          prompt: Photo showing no leak after 10 minutes

  - id: electrical_fault
    label: Electrical problem
    description: Power outage, breaker trip, or sparking outlet
    steps:
      - id: safety_check
        type: human_task
        label: Safety assessment
        description: |
          DO NOT TOUCH if sparking or smoking.
          Check if the issue is localized (one circuit) or whole-house.
        evidence:
          type: note

      - id: breaker_reset
        type: human_task
        label: Attempt breaker reset
        description: If breaker is tripped, reset once. If it trips again, STOP.
        depends_on: [safety_check]
        evidence:
          type: note
          prompt: Record which breaker and outcome

      - id: call_electrician
        type: human_task
        label: Call licensed electrician
        depends_on: [breaker_reset]
        evidence:
          type: note
          prompt: Electrician contact and appointment time

  - id: broken_hardware
    label: Broken hardware or fixture
    description: Gate hinge, door handle, fence post, etc.
    steps:
      - id: document_damage
        type: human_task
        label: Document the damage
        evidence:
          type: photo

      - id: parts_research
        type: human_task
        label: Research replacement part
        description: Find part number, supplier, and cost
        depends_on: [document_damage]
        evidence:
          type: note
          prompt: Part number, supplier, and estimated cost

      - id: purchase
        type: human_task
        label: Purchase replacement part
        depends_on: [parts_research]
        evidence:
          type: note
          prompt: Purchase receipt or confirmation

      - id: install
        type: human_task
        label: Install replacement part
        depends_on: [purchase]
        evidence:
          type: photo
          prompt: Photo of installed part

      - id: test
        type: human_task
        label: Test functionality
        depends_on: [install]
        evidence:
          type: note
          prompt: Confirm part works as expected
```

---

## 3.5 Consumables

Consumables are items that are regularly replaced on a schedule (filters, batteries, chemicals). They differ from routines in that they track the *thing* being consumed, not just the *action* of replacing it.

**v0 Status:** Schema defined, execution deferred to v1.

### Schema

```yaml
consumables:
  - id: <identifier>                # Unique consumable identifier
    label: <string>                 # Human-readable consumable name
    description: <string>           # Optional description
    zone: <zone_id>                 # Optional zone association
    asset: <asset_id>               # Optional asset association
    replace_every: <duration>       # Replacement interval
    last_replaced: <date>           # Last replacement date (YYYY-MM-DD)
    stock:                          # Optional inventory tracking (v1)
      current: <int>                # Current quantity on hand
      reorder_at: <int>             # Reorder threshold
    triggers_routine: <routine_id>  # Optional linked routine that performs replacement
```

### Field Definitions

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique consumable identifier (kebab-case) |
| `label` | string | Yes | Human-readable consumable name |
| `description` | string | No | Optional detailed description |
| `zone` | string | No | Associated zone |
| `asset` | string | No | Associated asset |
| `replace_every` | duration | Yes | Replacement interval (e.g., "90d", "6M") |
| `last_replaced` | date | Yes | Date of last replacement (ISO 8601: YYYY-MM-DD) |
| `stock.current` | integer | No | Current quantity on hand (v1) |
| `stock.reorder_at` | integer | No | Reorder threshold (v1) |
| `triggers_routine` | string | No | Routine ID that performs replacement |

### Constraints

- Consumable IDs must be unique
- `last_replaced` must not be in the future
- If `triggers_routine` is specified, it must reference a valid routine ID
- v0: Consumables are tracked but not enforced; reminders sent but no blocking

### Example

```yaml
consumables:
  - id: pool_filter_sand
    label: Pool sand filter media
    description: 50kg bag of pool filter sand
    zone: pool
    asset: pool_pump
    replace_every: 180d
    last_replaced: 2024-11-01
    triggers_routine: change_pool_filter

  - id: smoke_detector_batteries
    label: Smoke detector 9V batteries
    description: Pack of 4 batteries for all detectors
    replace_every: 180d
    last_replaced: 2025-01-15
    stock:
      current: 4
      reorder_at: 2

  - id: hvac_filter
    label: HVAC air filter
    zone: garage
    replace_every: 90d
    last_replaced: 2025-03-01
```

---

## 3.6 Delegation / Away Mode

The `delegation:` block activates away mode, assigning specified routines to a delegate for a time-bounded period. The homeowner remains the owner; the delegate receives notifications and submits evidence on behalf of the owner.

### Schema

```yaml
delegation:
  delegate:                         # Delegate information
    name: <string>                  # Delegate's name
    contact: <phone>                # Delegate's phone (E.164 format)
    email: <email>                  # Optional email
  active:                           # Delegation time window
    from: <date>                    # Start date (YYYY-MM-DD)
    to: <date>                      # End date (YYYY-MM-DD)
  assigns:                          # Routine assignments
    - routine: <routine_id>         # Explicit routine ID
    - zone: <zone_id>               # All routines in zone
  permissions:                      # Delegate permissions
    can_report_incidents: <bool>    # Can the delegate trigger incidents?
    can_modify_routines: <bool>     # Can the delegate reschedule routines?
    can_view_history: <bool>        # Can the delegate view run history?
  notifications:                    # Notification preferences
    remind_delegate_hours_before: <int>   # Delegate reminder lead time
    notify_owner_on_completion: <bool>    # Notify owner when delegate completes task
    notify_owner_if_overdue: <bool>       # Notify owner if delegate misses task
```

### Field Definitions

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `delegate.name` | string | Yes | Delegate's full name |
| `delegate.contact` | phone | Yes | Delegate's phone (E.164 format: +56 9 1234 5678) |
| `delegate.email` | email | No | Delegate's email address |
| `active.from` | date | Yes | Delegation start date (ISO 8601: YYYY-MM-DD) |
| `active.to` | date | Yes | Delegation end date (ISO 8601: YYYY-MM-DD) |
| `assigns` | array | Yes | Routine/zone assignments (minimum 1) |
| `assigns[].routine` | string | Conditional | Explicit routine ID (mutually exclusive with `zone`) |
| `assigns[].zone` | string | Conditional | Zone ID (assigns all routines in zone; mutually exclusive with `routine`) |
| `permissions.can_report_incidents` | boolean | No | Can delegate trigger incidents? (default: false) |
| `permissions.can_modify_routines` | boolean | No | Can delegate reschedule routines? (default: false) |
| `permissions.can_view_history` | boolean | No | Can delegate view run history? (default: true) |
| `notifications.remind_delegate_hours_before` | integer | No | Delegate reminder lead time in hours (default: 24) |
| `notifications.notify_owner_on_completion` | boolean | No | Notify owner on completion (default: true) |
| `notifications.notify_owner_if_overdue` | boolean | No | Notify owner if overdue (default: true) |

### Constraints

- `active.to` must be after `active.from`
- Each assignment must specify exactly one of `routine` or `zone`
- Assigned routines/zones must reference valid IDs
- Phone numbers must follow E.164 format (international format with country code)

### Delegation Semantics

**During active window:**
- Delegate receives notifications for assigned routines
- Delegate submits evidence via mobile app (linked by delegation token)
- Evidence is recorded with delegate attribution
- Owner receives summary notifications (based on `notify_owner_*` settings)

**Unassigned routines:**
- Routines not explicitly assigned remain the owner's responsibility
- Owner receives normal notifications
- Delegate does not see unassigned routines

**After delegation expires:**
- Delegate loses access immediately at midnight on `active.to` + 1 day
- All routines revert to owner
- Delegation record persists in run history for audit

### Example

```yaml
delegation:
  delegate:
    name: Carlos Méndez
    contact: "+56 9 8765 4321"
    email: carlos.mendez@example.com
  active:
    from: 2025-04-25
    to: 2025-05-02
  assigns:
    - routine: water_plants
    - routine: trash_day
    - zone: pool                    # Assigns pool_clean and any other pool routines
  permissions:
    can_report_incidents: true      # Carlos can report if something breaks
    can_modify_routines: false      # But cannot reschedule
    can_view_history: true          # Can see past run history for context
  notifications:
    remind_delegate_hours_before: 12
    notify_owner_on_completion: true
    notify_owner_if_overdue: true
```

---

## 3.7 Full Property File Example

The following is a complete `.home.yaml` file for "Casa Santiago" — a realistic example demonstrating all DSL features.

```yaml
# Casa Santiago — Home Maintenance Configuration
# gert-domain-home v0
# Owner: Santiago Family

property:
  name: Casa Santiago
  zones:
    - id: pool
      label: Swimming Pool
      description: Saltwater pool with heat pump and LED lighting
      metadata:
        area_sqm: 32
        type: outdoor

    - id: front_lawn
      label: Front Lawn
      metadata:
        area_sqm: 150
        type: outdoor

    - id: backyard
      label: Backyard Garden
      description: Raised bed vegetable garden and fruit trees

    - id: garage
      label: Garage
      description: Two-car garage with workshop area

    - id: driveway
      label: Driveway
      metadata:
        type: outdoor

assets:
  - id: pool_pump
    zone: pool
    label: Hayward SuperPump VS
    description: 1.5 HP variable speed pump
    installed: 2023-06-15
    warranty_expires: 2025-06-15
    metadata:
      model: VS-100
      serial: HW23-45678

  - id: riding_mower
    zone: garage
    label: John Deere X350
    installed: 2022-04-10
    metadata:
      hours_at_install: 0
      current_hours: 87

  - id: front_gate
    zone: driveway
    label: Driveway gate motor
    installed: 2020-11-03
    warranty_expires: 2023-11-03

routines:
  - id: pool_clean
    label: Pool cleaning
    description: Vacuum pool floor, brush walls, clean skimmer, check chemical levels
    zone: pool
    cadence:
      every: 7d
    evidence:
      type: photo
      prompt: Take photo of clear water and chemical test strip
    notifications:
      remind_hours_before: 4
      escalate_hours_overdue: 24

  - id: pool_filter_backwash
    label: Backwash pool filter
    asset: pool_pump
    cadence:
      every: 14d
    evidence:
      type: note
      prompt: Record backwash duration and final pressure reading
    notifications:
      remind_hours_before: 12

  - id: lawn_mow
    label: Mow front lawn
    zone: front_lawn
    cadence:
      seasonal:
        summer: 7d
        autumn: 10d
        winter: 21d
        spring: 10d
    evidence:
      type: photo
      prompt: Photo of mowed lawn
    executor:
      role: gardener

  - id: water_plants
    label: Water backyard plants
    description: Water raised beds and fruit trees
    zone: backyard
    cadence:
      every: 3d
    evidence:
      type: none
    notifications:
      remind_hours_before: 2

  - id: trash_day
    label: Take trash bins to curb
    description: Both recycling (blue) and general waste (black) bins
    cadence:
      every: 7d
    evidence:
      type: none
    notifications:
      remind_hours_before: 12
      escalate_hours_overdue: 2

consumables:
  - id: pool_filter_sand
    label: Pool sand filter media
    description: 50kg bag of pool filter sand
    zone: pool
    asset: pool_pump
    replace_every: 180d
    last_replaced: 2024-11-01
    triggers_routine: pool_filter_replace

  - id: smoke_detector_batteries
    label: Smoke detector 9V batteries
    description: 6-pack for all household detectors
    replace_every: 180d
    last_replaced: 2025-01-15
    stock:
      current: 6
      reorder_at: 2

incident_templates:
  - id: leak
    label: Water leak
    description: Emergency response for any water leak
    zones: [pool, backyard, garage]
    steps:
      - id: assess
        type: human_task
        label: Locate and assess the leak
        description: |
          1. Find the source of the leak
          2. Estimate severity (drip, stream, gush)
          3. Check if shut-off valve is accessible
        evidence:
          type: photo
          prompt: Take photo of leak source and affected area

      - id: shutoff_decision
        type: decision
        label: Can you safely shut off water?
        depends_on: [assess]
        choices:
          - value: "yes"
            label: "Yes, I can shut it off"
            next_step: shutoff
          - value: "no"
            label: "No, call plumber immediately"
            next_step: call_plumber

      - id: shutoff
        type: human_task
        label: Shut off water supply
        description: Turn the shut-off valve clockwise until fully closed
        evidence:
          type: photo
          prompt: Photo of closed valve

      - id: call_plumber
        type: human_task
        label: Call emergency plumber
        evidence:
          type: note
          prompt: Record plumber name, ETA, and job number

      - id: buy_parts
        type: human_task
        label: Purchase replacement parts
        description: Based on assessment, buy necessary parts
        depends_on: [assess]
        evidence:
          type: note
          prompt: List parts purchased and cost

      - id: fix
        type: human_task
        label: Perform repair
        depends_on: [shutoff, buy_parts]
        evidence:
          type: photo
          prompt: Photo of completed repair

      - id: verify
        type: human_task
        label: Verify fix
        description: Turn water back on and monitor for 10 minutes
        depends_on: [fix]
        evidence:
          type: photo
          prompt: Photo showing no leak after 10 minutes

  - id: broken_hardware
    label: Broken hardware or fixture
    description: Gate hinge, door handle, fence post, etc.
    steps:
      - id: document_damage
        type: human_task
        label: Document the damage
        evidence:
          type: photo

      - id: parts_research
        type: human_task
        label: Research replacement part
        description: Find part number, supplier, and cost
        depends_on: [document_damage]
        evidence:
          type: note
          prompt: Part number, supplier, and estimated cost

      - id: purchase
        type: human_task
        label: Purchase replacement part
        depends_on: [parts_research]
        evidence:
          type: note
          prompt: Purchase receipt or confirmation

      - id: install
        type: human_task
        label: Install replacement part
        depends_on: [purchase]
        evidence:
          type: photo
          prompt: Photo of installed part

      - id: test
        type: human_task
        label: Test functionality
        depends_on: [install]
        evidence:
          type: note
          prompt: Confirm part works as expected

delegation:
  delegate:
    name: Carlos Méndez
    contact: "+56 9 8765 4321"
    email: carlos.mendez@example.com
  active:
    from: 2025-04-25
    to: 2025-05-02
  assigns:
    - routine: water_plants
    - routine: trash_day
    - zone: pool                    # Assigns pool_clean and pool_filter_backwash
  permissions:
    can_report_incidents: true      # Carlos can report if something breaks
    can_modify_routines: false      # But cannot reschedule
    can_view_history: true          # Can see past run history for context
  notifications:
    remind_delegate_hours_before: 12
    notify_owner_on_completion: true
    notify_owner_if_overdue: true
```

---

## Compilation Semantics (Normative)

The gert-domain-home compiler transforms the DSL into GERT execution primitives:

### Routines → Scheduled Runs

Each routine compiles to:
1. **Schedule rule** (cron-like or calendar-based, depending on `cadence.every` vs `cadence.seasonal`)
2. **Run template** with single `type: human_task` step
3. **Evidence requirement** attached to the step
4. **Notification rules** as engine-level triggers

### Incident Templates → Run Templates

Each incident template compiles to:
1. **Run template** with `steps` as GERT step tree
2. **Step types** map to GERT primitives:
   - `human_task` → `type: manual` (v2) with evidence config
   - `decision` → `type: branch` with choice branches
   - `parallel` → `type: parallel` (v2)
3. **Dependencies** (`depends_on`) → step-level `depends_on` in GERT execution graph

### Delegation → Access Control + Notification Override

During active window:
1. **Access control rule** grants delegate read/write on assigned runs
2. **Notification routing override** redirects assigned routine notifications to delegate contact
3. **Attribution metadata** tags delegate submissions with delegate ID

---

## Forward Compatibility

### v1 Extension Points

The following features are deferred to v1 but have schema extension points reserved:

1. **Asset metadata standardization:** `metadata.model`, `metadata.serial`, `metadata.warranty_url`
2. **Routine executor enforcement:** `executor.role` becomes required field with RBAC enforcement
3. **Consumable inventory tracking:** `stock.current` and `stock.reorder_at` become operational
4. **Seasonal cadence customization:** Allow user to override hemisphere or define custom season boundaries
5. **Incident template versioning:** Allow templates to be versioned for iterative improvement
6. **Delegation permission granularity:** Per-routine permission overrides
7. **Multi-delegate assignment:** Allow multiple delegates with disjoint routine sets

### Schema Versioning

All `.home.yaml` files will include:
```yaml
apiVersion: home/v0
```

When v1 is released, the compiler will support both `home/v0` and `home/v1` in compatibility mode, with `gert migrate` to rewrite v0 files to v1 format.

---

## End of Section 3
