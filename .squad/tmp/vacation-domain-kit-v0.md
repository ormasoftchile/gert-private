# Vacation / Guided Stay Domain Kit — Prototype Design v0

**Authors:** Ken (Architecture), John (YAML DSL), Barbara (Integrations & UX)
**Date:** 2026-04-22
**Status:** Draft v0 — for review by Cristian

This document defines a prototype Domain Kit for vacation / guided-stay workflows
built on top of GERT core. It is organized into 9 sections following the brief
provided by Cristian. Each section is the work of a specialist; cross-references
are explicit.

**Kit vocabulary:** `apiVersion: vacation.gert.io/v0` — kit resources use this prefix.
**GERT core version required:** `>=2.0.0`

---

# Vacation Domain Kit — Prototype Design

**Author:** Ken (Software Architect)  
**Date:** 2026-04-22  
**Status:** Draft — Sections 1, 2, 4, 5, 8  
**Requested by:** Cristian

---

## Section 1: Domain Kit Definition

### 1.1 What Problems Does a Vacation Domain Kit Solve?

A hospitality operator managing guest stays faces a scheduling, coordination, and fulfillment problem that is fundamentally a **durable workflow** — but the vocabulary of runs, steps, events, and governance primitives is foreign to anyone thinking in terms of guests, days, activities, meals, and credits.

Without a domain kit, operators would need to:

1. **Author raw runbooks** using core step types (`cli`, `manual`, `tool`, `branch`, `iterate`) to model guest stays. This forces hospitality staff to think in terms of execution primitives rather than their natural domain concepts (check-in, day skeleton, activity pool, meal slot, QR access pass).

2. **Manually wire** timer-based wake-ups (morning activation, dinner reminders, stay expiration), event-driven replanning (weather changes, operator overrides), and human-task resolution (guest "What now?" taps, upgrade confirmations) using raw GERT constructs. Each stay would require hand-coding 50+ steps that follow predictable patterns.

3. **Reinvent credit/allowance tracking** as ad-hoc variable manipulation rather than a first-class ledger with consumption rules, balance checks, and audit trails.

4. **Lose semantic validation.** Core GERT cannot enforce domain invariants like "every day must have at least one meal slot," "QR passes must expire when the stay ends," or "credits cannot go negative." Without kit validators, malformed stay templates pass through the parser silently.

5. **Sacrifice projections.** Without a kit, there is no "stay timeline" view, no "guest activity history" projection, no "credits remaining" read model. Operators would need to read raw trace events.

### 1.2 Why a Domain Kit vs. Raw Runbooks?

The Domain Kit abstraction (§4 of the design doc) exists precisely for this scenario. The Vacation Kit provides:

| Kit Capability | What It Does for Vacations |
|---|---|
| **Authoring schemas** | Authors write `stay-template`, `day-skeleton`, `activity`, `meal-slot` — not `iterate` + `branch` + `manual` |
| **Validation rules** | "Every day must define breakfast, lunch, dinner slots." Checked at compile time. |
| **Lowering/compilation** | `stay-template` compiles to a parent run with child sub-runs per day, timers for slot transitions, human tasks for guest choices. The runtime never sees vacation concepts. |
| **Projections** | "Stay timeline," "credits remaining," "activity history" views computed from trace data. |
| **Testing contracts** | Fixture stay templates with expected lowered output verify the compiler is correct. |

A raw runbook approach would produce the same execution, but every stay template author would need deep GERT internals knowledge. The kit makes authoring accessible to hospitality operators while preserving all of GERT's governance, evidence, and traceability guarantees.

### 1.3 Abstractions Introduced

The Vacation Kit introduces these domain-specific authoring concepts:

- **Stay Template** — a parameterized template for an entire guest stay (like a "package")
- **Stay Run** — a live, executing instance of a stay template bound to a specific guest
- **Day Skeleton** — the structural shape of a single day (time slots, required activities)
- **Day Flavor** — a thematic variant applied to a day skeleton (e.g., "Adventure Day," "Relax Day")
- **Day Modifier** — a runtime patch to a day (weather override, operator override)
- **Activity** — a concrete thing a guest can do, with duration, location, capacity, prerequisites
- **Activity Pool** — the set of available activities for a given context (day flavor + constraints)
- **Meal Slot** — a time-bounded slot for a meal, with venue options and dietary constraints
- **Operator** — the hospitality staff identity with override authority
- **Guest Pass / QR Access** — a time-bounded, scopeable access token for guest identity/entry
- **Credits / Allowances** — a numeric ledger tracking consumable value (spa credits, bar allowance, upgrade points)
- **Suggestion Context** — the current state used to compute "What should I do now?" recommendations
- **Upgrade Option** — a credit-consuming enhancement to the current stay or day

### 1.4 Clean Kernel Compliance

Per §4.3 (The Clean Kernel Principle): **the runtime never executes kit-specific step types.**

The Vacation Kit follows the compilation pipeline:

```
Vacation YAML → Vacation Parser → Vacation AST → Vacation Compiler → Core YAML → Core Parser → ExecutionPlan
```

The lowered output is a valid `runbook/v2` document using only core primitives (`cli`, `manual`, `tool`, `branch`, `iterate`). The runtime is never aware that vacation concepts exist. The lowered document can be inspected, version-controlled, and executed without the Vacation Kit installed.

The Kit compiler is deterministic: same stay template → same lowered output. This enables `gert test` scenarios against lowered output.

---

## Section 2: Core Domain Concepts

### 2.1 Stay Template

**Definition:** A parameterized blueprint for an entire guest stay. Defines the number of days, included services, credit allocation, default day flavors, governance rules, and operator roles. Analogous to a "vacation package" that can be instantiated per guest.

**Lifecycle role:** Authoring artifact. Written by the operator, validated at compile time, lowered to core GERT before execution. Never modified at runtime — runtime changes go through Day Modifiers.

**GERT primitive mapping:**
- Stay Template → **runbook definition** (the YAML source document)
- Template parameters (guest name, dates, room, credit amount) → **runbook inputs** (resolved at run creation)
- Included services → mapped to step sequences in the lowered output
- Governance rules (operator override authority, guest data redaction) → **`meta.governance`** fields in the lowered runbook

### 2.2 Stay Run

**Definition:** A live, executing instance of a Stay Template, bound to a specific guest, with concrete dates, resolved parameters, and active state. The Stay Run is the top-level durable entity that persists across the entire guest visit.

**Lifecycle role:** Runtime entity. Created at guest check-in, active for the duration of the stay, sealed at checkout/expiration. Survives process restarts (GERT resumability). All sub-entities (days, activities, meals) are children of this run.

**GERT primitive mapping:**
- Stay Run → **parent run** (`run_id`, created via `run/started` event)
- Run state (checked-in, active, completing, sealed) → **run lifecycle** (`run/started` → `step/*` progression → `run/completed`)
- Run directory → `.runbook/runs/<run-id>/` with `trace.jsonl`, snapshots, attachments
- Guest identity → **run input** (`inputs.guest_id`, `inputs.guest_name`)
- Resumability → **checkpoint/resume** mechanism (§13 evidence, snapshots per step)

### 2.3 Day Skeleton

**Definition:** The structural shape of a single day within a stay. Defines the time-slot grid: wake-up time, morning block, lunch slot, afternoon block, dinner slot, evening block. Does not specify what fills the slots — that's the Day Flavor's job.

**Lifecycle role:** Compile-time structural template. Each day in a Stay Template references a Day Skeleton. The skeleton defines the timing grid; flavors and modifiers fill it.

**GERT primitive mapping:**
- Day Skeleton → **sub-run structure template** (defines the sequence of child steps within a day sub-run)
- Time slots → **timer/wakeup** primitives (e.g., "morning block starts at 08:00" → timer fires at 08:00, transitions to next step)
- Slot boundaries → **events** (`slot/transition` lowers to a core event emission + timer-based advancement)

### 2.4 Day Flavor

**Definition:** A thematic variant that fills a Day Skeleton with specific activities and options. Examples: "Adventure Day" (hiking, kayaking, zipline), "Relax Day" (spa, pool, reading corner), "Cultural Day" (museum tour, cooking class, wine tasting). A flavor selects from the Activity Pool what goes into each slot.

**Lifecycle role:** Compile-time configuration. Applied to a Day Skeleton to produce a concrete day plan. Multiple flavors can exist for the same skeleton. The stay template assigns a default flavor per day; operators can swap flavors via Day Modifiers.

**GERT primitive mapping:**
- Day Flavor → **branch configuration** (the flavor selection determines which branch of activities is compiled into the day sub-run)
- Flavor selection → **runtime state variable** (`day.flavor`) that the lowered runbook uses in `branch` conditions
- Flavor swap (operator override) → **event trigger** that fires a replanning sub-run

### 2.5 Day Modifier

**Definition:** A runtime patch applied to an already-planned day. Triggered by external events (weather change, operator decision, guest request) or internal conditions (activity sold out, venue closed). Modifiers do not replace the day skeleton — they adjust slot contents within the existing structure.

**Lifecycle role:** Runtime entity. Created in response to events, applied to the current or future day's sub-run. Modifiers are recorded in the evidence trail.

**GERT primitive mapping:**
- Day Modifier → **event trigger → replanning sub-run**
  - External event (e.g., `weather/changed`) arrives as a **GERT event** on the event bus
  - The parent run's event handler spawns a **child sub-run** that re-evaluates affected slots
  - Modified slot assignments become new **step replacements** in the day sub-run
- Modifier audit trail → **evidence** (trace events record what was modified, by whom, why)
- Operator-initiated modifier → **human task** (operator confirms override) + **governance audit** (`governance/approved` event)

### 2.6 Activity

**Definition:** A concrete thing a guest can do. Has a name, duration, location, capacity (max concurrent guests), prerequisites (e.g., "must have life jacket certification for kayaking"), time-of-day constraints, and credit cost (if applicable).

**Lifecycle role:** Reference data. Activities are defined in the stay template or in a shared activity catalog. At runtime, an activity becomes a step in the day sub-run when selected to fill a slot.

**GERT primitive mapping:**
- Activity definition → **tool definition** in the Kit's companion extension (activity metadata: name, duration, location, capacity)
- Activity execution (guest does the activity) → **manual step** (human task — guest or operator confirms completion) or **tool step** (if the activity involves a system action like booking a slot)
- Activity prerequisites → **branch condition** in the lowered runbook (check prerequisite state before offering activity)
- Capacity tracking → **runtime state variable** (`activity.<id>.remaining_capacity`), decremented when a guest commits

### 2.7 Activity Pool

**Definition:** The set of activities available for a given context: filtered by day flavor, time of day, weather conditions, guest preferences, remaining capacity, and credit balance. The pool is the input to the suggestion engine.

**Lifecycle role:** Runtime computed set. Recomputed whenever the context changes (time advances, weather event, activity fills up). Not persisted as a GERT entity — it's a projection over current state.

**GERT primitive mapping:**
- Activity Pool → **runtime state computation** (a function over the current day's state variables: flavor, time slot, weather, capacities, guest credits)
- Pool filtering → lowered as a **branch** tree that evaluates conditions and produces the eligible set
- Pool as suggestion input → feeds the **suggestion context** (see §2.12)

### 2.8 Meal Slot

**Definition:** A time-bounded slot for a meal within a day. Defines: meal type (breakfast, lunch, dinner, snack), time window (start/end), venue options, dietary constraint support, and whether the slot is included in the stay credits or costs extra.

**Lifecycle role:** Structural element of a Day Skeleton. Filled at compile time with default venue; can be modified at runtime (venue change, dietary adjustment). Meal completion is tracked for billing and dietary compliance.

**GERT primitive mapping:**
- Meal Slot → **timer-bounded manual step** in the day sub-run
  - Timer fires at slot start time → step becomes active
  - Timer fires at slot end time → step auto-advances (or warns if not completed)
- Venue selection → **branch** (if multiple venues, guest or operator picks one)
- Dietary constraint → **validation rule** at compile time (Kit validator ensures dietary info is declared)
- Meal completion → **manual step completion** (guest confirms, or operator marks done) → evidence event
- Credit impact → **runtime state mutation** (if meal costs credits, ledger is decremented)

### 2.9 Operator / Guest Pass / QR Access

**Operator:**

**Definition:** A hospitality staff member with authority to override stay parameters, approve upgrades, and manage guest experiences. Operators have different authority levels (front desk, concierge, manager).

**Lifecycle role:** Identity and authorization entity. Operators are declared in the stay template's governance section. At runtime, operator actions require governance approval (the operator is both the actor and the approver, but this is audited).

**GERT primitive mapping:**
- Operator → **role in `meta.governance.roles`** (e.g., `operator: [concierge, front-desk, manager]`)
- Operator override → **approval gate** — the override triggers a governance checkpoint where the operator's role is verified
- Operator identity → **actor identity** in trace events (`user` field in event envelope)

**Guest Pass / QR Access:**

**Definition:** A time-bounded, scope-limited access token issued to a guest. Encodes: guest identity, stay run ID, valid-from/valid-to timestamps, and access scopes (e.g., "pool access," "gym access," "restaurant entry"). Materialized as a QR code for physical scanning.

**Lifecycle role:** Created at check-in (stay run start), revoked at checkout (stay run seal). Scopes can be adjusted mid-stay via operator override. The pass is the guest's runtime identity token.

**GERT primitive mapping:**
- QR Pass → **policy + access token**
  - The pass is a **signed JWT** (or similar token) containing `run_id`, `guest_id`, `scopes[]`, `valid_from`, `valid_to`
  - Issuance → **tool step** in the lowered runbook (a `tool` step calls a token-generation tool at check-in)
  - Validation → **policy evaluation** (when a scope-gated resource is accessed, the GERT policy layer checks the token's scopes and validity)
  - Revocation → **run completion event** triggers token invalidation (token's `valid_to` is set to run seal time)
- Pass as evidence → token issuance and revocation events are recorded in `trace.jsonl`

### 2.10 Credits / Allowances

**Definition:** A numeric ledger tracking consumable value within a stay. Examples: "100 spa credits included," "50 bar allowance," "3 upgrade tokens." Credits are allocated at stay creation, consumed by activities/upgrades/extras, and cannot go negative (unless the operator explicitly overrides).

**Lifecycle role:** Runtime state that persists across the entire stay. Initialized from the stay template, mutated by consumption events, auditable via the evidence trail. Balance checks gate certain activities/upgrades.

**GERT primitive mapping:**
- Credits → **runtime ledger state**
  - Stored as a **runtime state variable**: `credits.<category>.balance` (e.g., `credits.spa.balance = 100`)
  - Initial allocation → set in **run inputs** or first step of parent run
  - Consumption → **state mutation** within a step: `credits.spa.balance -= activity.cost`
  - Balance check → **branch condition** in the lowered runbook: `if credits.spa.balance >= required_cost then allow else deny`
  - Negative balance protection → **Kit validation rule** (compile-time: "cost fields must be non-negative") + **runtime branch** (balance check before deduction)
  - Audit trail → every credit mutation emits an **evidence event** with before/after balance, reason, actor
- Allowance reset → if the template defines daily allowances, a **timer** at day-start resets the daily portion
- Credit ledger projection → Kit projection reads credit-related trace events and computes "credits remaining" view

### 2.11 Suggestion Context

**Definition:** The current state snapshot used to compute personalized "What should I do now?" recommendations for a guest. Includes: current time, day flavor, weather, remaining credits, completed activities today, guest preferences, activity pool availability.

**Lifecycle role:** Ephemeral runtime computation. Built on demand when a guest taps "What now?" Not persisted as a GERT entity — assembled from current run state and activity pool.

**GERT primitive mapping:**
- Suggestion request ("What now?" tap) → **non-blocking human task**
  - The lowered runbook includes a `manual` step that presents suggestions and awaits guest choice
  - This is a **human task** in GERT's model: the runtime pauses this step until the guest responds
  - The suggestions themselves are computed by a **tool step** that reads current state and returns ranked options
- Suggestion as advisory → the suggestion is **non-blocking to the stay run** — other timers, events, and day progression continue while the guest browses options
  - Implemented as a **parallel sub-run** or a non-blocking `manual` step that doesn't gate the parent run's advancement
- Suggestion context assembly → **runtime state read** (gather `day.flavor`, `time.current`, `weather.current`, `credits.*.balance`, `activities.completed[]`)

### 2.12 Upgrade Option

**Definition:** A credit-consuming enhancement to the current stay or day. Examples: "Upgrade to ocean-view room (+50 credits)," "Add private guide to hike (+30 credits)," "Late checkout (+20 credits)." Upgrades modify the stay run's state and may trigger replanning.

**Lifecycle role:** Available as a suggestion or operator-offered option. Consuming an upgrade deducts credits, modifies state, and may spawn a replanning sub-run (e.g., room change requires updating the QR pass scopes).

**GERT primitive mapping:**
- Upgrade availability → **branch condition** (check credit balance, check upgrade prerequisites)
- Upgrade confirmation → **human task** (guest or operator confirms the upgrade)
  - Guest-initiated: `manual` step with upgrade details, guest taps "confirm"
  - Operator-initiated: `manual` step with approval gate (operator role verified)
- Credit deduction → **state mutation** (`credits.<category>.balance -= upgrade.cost`)
- State modification → **sub-run** that applies the upgrade's effects:
  - Room upgrade → update `stay.room` state variable, regenerate QR pass scopes (tool step)
  - Activity upgrade → modify current/future day sub-run plan (Day Modifier mechanism)
- Upgrade evidence → **trace events**: `upgrade/requested`, `upgrade/confirmed`, `credits/deducted`, `upgrade/applied`

---

## Section 4: Lowering / Compilation Model

### 4.1 Overview

The Vacation Kit compiler transforms vacation-specific YAML into standard `runbook/v2` documents using only core primitives. The compiler is a **pure function**: vacation YAML in, core YAML out. It runs at author time / compile time, never at execution time.

The lowering pipeline:

```
vacation.yaml (apiVersion: runbook/v2+vacation/v1)
  → Vacation Parser (validates vacation schema)
  → Vacation AST (stay template, days, activities, meals, credits)
  → Vacation Compiler (lowering rules below)
  → runbook/v2 YAML (cli, manual, tool, branch, iterate steps)
  → Core Parser → Planner → ExecutionPlan
```

### 4.2 Stay Template → Parent Run Structure

A stay template lowers to a parent runbook with:

```yaml
apiVersion: runbook/v2
meta:
  name: "stay-{{.guest_id}}-{{.checkin_date}}"
  governance:
    roles: [operator, concierge, manager]
    allowed_commands: [token-gen, notify, booking-api]
    redact_patterns: ["*credit_card*", "*passport*"]
inputs:
  - name: guest_id
    type: string
    required: true
  - name: guest_name
    type: string
  - name: checkin_date
    type: string
  - name: checkout_date
    type: string
  - name: credits_spa
    type: number
    default: 100
  - name: credits_bar
    type: number
    default: 50
```

The parent run's steps are:

1. **Initialization step** (`tool`): Allocate credits, generate QR pass, emit `stay/started` evidence
2. **Day iterate** (`iterate`): Loop over each day in the stay, spawning day sub-runs
3. **Finalization step** (`tool`): Revoke QR pass, archive credit ledger, emit `stay/sealed` evidence

### 4.3 Day → Child Sub-Run

Each day in the iterate loop lowers to a **child sub-run** (an `invoke` step referencing a day runbook):

```yaml
steps:
  - name: "day-{{.day_number}}"
    type: include
    runbook: ".generated/day-{{.day_flavor}}.yaml"
    inputs:
      day_number: "{{.day_number}}"
      flavor: "{{.day_flavor}}"
      weather: "{{.weather_current}}"
      credits_spa: "{{.credits_spa}}"
```

The day sub-runbook contains:

1. **Morning activation** — `manual` step (operator confirms day start or timer auto-advances at configured wake-up time)
2. **Morning activity block** — `branch` step selecting from Activity Pool based on flavor + weather + credits
3. **Lunch slot** — `manual` step with venue branch
4. **Afternoon activity block** — same pattern as morning
5. **Dinner slot** — timer-triggered `manual` step with venue selection
6. **Evening block** — optional activities or free time

### 4.4 Slot Transitions → Timers + Events

Time-bounded slots in the Day Skeleton lower to **timer/wakeup** patterns:

```yaml
# Morning slot transition (lowered)
steps:
  - name: "wait-morning-start"
    type: tool
    tool: gert.timer
    spec:
      action: wait-until
      time: "{{.day_date}}T08:00:00Z"
    # Timer fires → emits step/completed → next step activates

  - name: "morning-block"
    type: branch
    # ... activity selection logic

  - name: "wait-lunch-start"
    type: tool
    tool: gert.timer
    spec:
      action: wait-until
      time: "{{.day_date}}T12:30:00Z"
```

Each timer is a `tool` step calling `gert.timer` (a core extension tool that implements the wakeup/sleep primitive). When the timer fires, the step completes and the next step in the sequence activates. This maps directly to GERT's **durable timer** mechanism — the timer survives process restarts because the step's pending state is checkpointed.

### 4.5 QR Pass → Access Token + Policy

QR pass issuance lowers to a `tool` step + policy configuration:

```yaml
# Check-in: issue QR pass (lowered)
steps:
  - name: "issue-guest-pass"
    type: tool
    tool: vacation.token-gen
    spec:
      action: issue
      guest_id: "{{.guest_id}}"
      run_id: "{{.run_id}}"
      scopes: ["pool", "gym", "restaurant", "room-{{.room_number}}"]
      valid_from: "{{.checkin_date}}"
      valid_to: "{{.checkout_date}}"
    outputs:
      - name: qr_token
        store_as: guest.qr_token
```

The pass itself is a signed token. Validation happens outside GERT's core runtime — when a physical scanner reads the QR code, it calls a validation endpoint that checks the token's validity window and scopes. GERT's role is:

1. **Issue** the token (tool step at check-in)
2. **Record** issuance in evidence (`trace.jsonl`)
3. **Revoke** at checkout (tool step calls revocation API)
4. **Audit** all access events if the validation endpoint emits events back to GERT

The **policy** dimension: the stay template's `meta.governance` section declares what scopes exist and who can modify them. Operator scope changes (e.g., adding "spa-premium" scope mid-stay) require an **approval gate**.

### 4.6 Credits → Runtime Ledger State

Credits lower to **runtime state variables** with a disciplined mutation pattern:

**Initialization (stay start):**
```yaml
- name: "init-credits"
  type: tool
  tool: vacation.ledger
  spec:
    action: initialize
    categories:
      spa: "{{.credits_spa}}"
      bar: "{{.credits_bar}}"
      upgrade: "{{.credits_upgrade}}"
```

**Consumption (during activity/upgrade):**
```yaml
- name: "check-credit-balance"
  type: branch
  spec:
    condition: "{{.credits_spa}} >= {{.activity_cost}}"
    if_true:
      - name: "deduct-credits"
        type: tool
        tool: vacation.ledger
        spec:
          action: deduct
          category: spa
          amount: "{{.activity_cost}}"
          reason: "{{.activity_name}}"
    if_false:
      - name: "insufficient-credits"
        type: manual
        spec:
          prompt: "Insufficient spa credits (balance: {{.credits_spa}}, cost: {{.activity_cost}}). Request operator override?"
```

The **ledger tool** (`vacation.ledger`) is a companion extension tool (NOT a Kit component — tools are extension concerns per §4). It:
- Maintains credit state as GERT runtime variables
- Emits evidence events on every mutation (`credits/deducted`, `credits/refunded`)
- Enforces non-negativity unless operator override flag is set
- Provides balance query for branch conditions

### 4.7 Suggestions → Non-Blocking Human Task / Advisory Projection

The "What now?" suggestion pattern lowers to:

```yaml
- name: "suggestion-request"
  type: tool
  tool: vacation.suggest
  spec:
    action: compute
    context:
      flavor: "{{.day_flavor}}"
      time: "{{.current_time}}"
      weather: "{{.weather_current}}"
      credits: "{{.credits_spa}}"
      completed: "{{.activities_completed_today}}"
  outputs:
    - name: suggestions
      store_as: current_suggestions

- name: "guest-choice"
  type: manual
  spec:
    prompt: "Here are your options:\n{{range .current_suggestions}}• {{.name}} ({{.duration}}, {{.credits}} credits)\n{{end}}\nChoose one or skip."
    timeout: 1800  # 30 min — auto-skip if no response
```

The suggestion tool computes ranked options; the manual step presents them. The **timeout** on the manual step ensures the day progresses even if the guest doesn't respond — this is a **non-blocking advisory**, not a gate.

The suggestion computation is a **projection**: it reads current state without modifying it. The tool could equally be implemented as a Kit projection, but we use a tool because the computation needs runtime access to live state (capacities, weather, time).

### 4.8 Weather Override → Event Trigger → Replanning Sub-Run

Weather changes are modeled as external events that trigger replanning:

```yaml
# Event listener (lowered as a parallel branch in the day sub-run)
- name: "weather-watch"
  type: tool
  tool: vacation.weather
  spec:
    action: watch
    location: "{{.resort_location}}"
    notify_on: ["rain", "storm", "extreme-heat"]

# When weather event fires, it sets a state variable:
# weather.current = "rain"
# This triggers a branch that spawns a replanning sub-run:

- name: "weather-replan"
  type: branch
  spec:
    condition: "{{.weather_changed}}"
    if_true:
      - name: "replan-day"
        type: include
        runbook: ".generated/replan-weather.yaml"
        inputs:
          new_weather: "{{.weather_current}}"
          remaining_slots: "{{.remaining_slots}}"
          day_flavor: "{{.day_flavor}}"
```

The replanning sub-run:
1. Reads the current day state
2. Filters the Activity Pool for weather-appropriate activities
3. Generates replacement slot assignments
4. Applies them as **Day Modifiers** (state variable updates in the parent day sub-run)

This exercises GERT's **event model** (weather tool emits event), **sub-run composition** (replanning is a child run), and **resumability** (if the process restarts mid-replan, the checkpoint allows recovery).

### 4.9 Operator Override → Governance Primitive

Operator overrides lower to **approval-gated manual steps**:

```yaml
- name: "operator-override"
  type: manual
  spec:
    prompt: "Override requested: {{.override_description}}"
    requires_approval:
      roles: [operator, manager]
      min_approvals: 1
    evidence:
      capture: true
      fields: [override_reason, authorized_by]
```

This maps directly to GERT's **approval gate** primitive (§12 Governance). The governance layer:
1. Verifies the actor has the required role
2. Records the approval in the trace (`governance/approved` event)
3. Captures evidence fields (reason, authorizer)
4. Only then allows the override step to proceed

For elevated overrides (e.g., overriding a negative credit balance), the template can require `manager` role:

```yaml
requires_approval:
  roles: [manager]
  min_approvals: 1
  escalation_timeout: 300  # 5 min to get manager approval
```

---

## Section 5: Runtime Behavior

### 5.1 Stay Lifecycle Walkthrough

This walkthrough traces a complete stay through GERT's runtime, showing how core primitives (runs, events, timers, human tasks, sub-runs, evidence) are exercised at each stage.

---

#### 5.1.1 Guest Arrives → Run Starts

**Trigger:** Front desk operator initiates check-in via the GERT client (CLI, TUI, or web).

**GERT mechanics:**
1. The Vacation Kit compiler has already lowered the stay template to a `runbook/v2` document.
2. `gert run stay-premium.yaml --input guest_id=G-4421 --input checkin_date=2026-05-01` creates a new **parent run**.
3. The runtime emits `run/started` event (sequence 0) with all inputs.
4. **Step 1 (tool: init-credits):** The `vacation.ledger` tool initializes `credits.spa=100`, `credits.bar=50`, `credits.upgrade=3`. State variables are set. Evidence event emitted.
5. **Step 2 (tool: issue-guest-pass):** The `vacation.token-gen` tool generates a signed QR token. Token value stored as `guest.qr_token`. Evidence event records token issuance with scopes and validity window.
6. **Checkpoint:** State snapshot written to `.runbook/runs/<run-id>/snapshots/step-0001.json`. If the process crashes here, GERT resumes from this checkpoint.

**Resumability exercised:** If the operator's laptop loses power after credit init but before QR issuance, GERT resumes at step 2 on restart. Credits are already initialized (idempotent check via snapshot), QR issuance proceeds.

---

#### 5.1.2 Morning Activation

**Trigger:** Timer fires at 08:00 on Day 1.

**GERT mechanics:**
1. The **iterate** step advances to day 1, spawning the day sub-run.
2. The day sub-run's first step is `wait-morning-start` — a `gert.timer` tool step set for 08:00.
3. Timer is **durable**: registered in the run's timer state. If the process restarts at 07:30, the timer is restored from the checkpoint and fires at 08:00.
4. Timer fires → step completes → `step/completed` event emitted → next step (`morning-block`) activates.
5. The morning block is a `branch` step that evaluates: day flavor + weather + credit balance → selects activities from the pool.

**Event model exercised:** Timer completion generates a `step/completed` event. The step transition is recorded in `trace.jsonl`. If an adapter (TUI or web) is connected, it renders the morning activation in real time via SSE/WebSocket.

---

#### 5.1.3 "What Now?" Tap → Suggestion Resolved

**Trigger:** Guest opens the app and taps "What should I do?"

**GERT mechanics:**
1. The day sub-run has a `suggestion-request` step that can be triggered by the guest.
2. **Tool step (vacation.suggest):** Reads current state — flavor is "Adventure Day," weather is sunny, spa credits = 85, completed today = [hiking], time = 10:30. Computes ranked suggestions: [kayaking (45min, 15 credits), zipline (30min, 20 credits), pool (free)].
3. **Manual step (guest-choice):** Presents options to the guest via the adapter. This is a **human task** — the runtime pauses THIS step and waits for the guest's response.
4. Guest picks "kayaking." The manual step completes with `response: "kayaking"`.
5. The next step begins: capacity check for kayaking (branch), credit check (branch), then the activity step (manual: "proceed to kayak launch at 10:45").

**Human task model exercised:** The `manual` step suspends the day sub-run at this point. The **parent stay run continues** — other timers keep ticking, other events can fire. This is non-blocking composition: the suggestion step doesn't gate the entire stay.

**Resumability:** If the app disconnects, the manual step remains pending. When the guest reconnects, the adapter retrieves current run state and re-renders the pending prompt. No work is lost.

---

#### 5.1.4 Operator Override

**Trigger:** Concierge wants to give the guest complimentary access to the premium spa (normally requires 50 credits, guest has 35).

**GERT mechanics:**
1. Concierge initiates override via operator interface.
2. The lowered runbook includes an `operator-override` manual step with `requires_approval: { roles: [operator, concierge, manager] }`.
3. **Governance checkpoint fires:** The runtime checks the concierge's role against the approval gate. Role = `concierge` ∈ `[operator, concierge, manager]` → **approved**.
4. The governance layer emits `governance/approved` event with: actor, role, timestamp, override description.
5. **Evidence capture:** The override step captures `override_reason: "guest birthday"` and `authorized_by: "concierge-maria"`.
6. The credit check branch is bypassed — the override permits the activity regardless of balance.
7. Optionally, the credits are adjusted: `vacation.ledger` tool deducts 50, balance goes to -15, with a `credits/override` evidence event recording the negative balance and authorization.

**Governance exercised:** Approval gates, role verification, evidence capture, audit trail. The entire override is traceable: who, when, why, what authority.

---

#### 5.1.5 Rain → Weather Event → Replanning

**Trigger:** Weather API detects rain starting at 14:00.

**GERT mechanics:**
1. The `vacation.weather` tool (running as a persistent watcher in the day sub-run) detects the weather change.
2. Tool emits an **event**: `weather/changed` with payload `{ condition: "rain", starts_at: "14:00", location: "resort" }`.
3. The event sets a state variable: `weather.current = "rain"`, `weather.changed = true`.
4. The `weather-replan` branch evaluates: `weather.changed == true` → triggers the replanning sub-run.
5. **Replanning sub-run spawns** (`invoke` step): reads current day state, filters Activity Pool for indoor activities, generates new afternoon slot assignments.
6. Sub-run completes → Day Modifier applied → afternoon activities changed from [kayaking, beach volleyball] to [spa, indoor cooking class].
7. Guest is notified via adapter (event emitted, adapter renders updated schedule).

**Event model + sub-run composition exercised:** External event → state change → branch evaluation → child sub-run → state modification → evidence. The replanning sub-run is a full GERT run with its own trace, evidence, and checkpoint. If the process crashes during replanning, the sub-run resumes from its last checkpoint.

---

#### 5.1.6 Dinner Reminder → Timer Fires

**Trigger:** Timer set for 19:00 (dinner slot start).

**GERT mechanics:**
1. The `wait-dinner-start` timer step fires at 19:00.
2. `step/completed` event emitted for the timer step.
3. Next step activates: `dinner-venue-selection` (branch step with venue options).
4. If the guest has a pending suggestion session, it's **not interrupted** — dinner is a separate step in the day sequence. The adapter shows a dinner notification alongside any pending suggestions.
5. Guest selects venue → manual step completes → meal slot step begins → dietary constraints validated (from stay template metadata).
6. Dinner step includes a `manual` sub-step for meal completion confirmation. When confirmed, evidence event records: venue, time, dietary notes.

**Timer durability:** If GERT restarts at 18:55, the timer is restored from checkpoint. It fires at 19:00. If GERT restarts at 19:05 (timer already past), the timer fires immediately on resume — the step was pending, and the timer's target time has passed.

---

#### 5.1.7 Credit Consumed by Upgrade

**Trigger:** Guest wants to upgrade to ocean-view room (costs 2 upgrade tokens).

**GERT mechanics:**
1. Upgrade option appears in suggestion list or is offered by operator.
2. **Balance check branch:** `credits.upgrade >= 2` → true (balance is 3).
3. **Human task:** Guest confirms upgrade via manual step. Step captures `upgrade_type: "ocean-view"`, `cost: 2`.
4. **Ledger mutation:** `vacation.ledger` tool deducts 2 upgrade tokens. Emits `credits/deducted` event with before (3) / after (1) balance.
5. **Side effects sub-run:** Spawns a child sub-run that:
   - Updates `stay.room` state variable
   - Calls `vacation.token-gen` tool to regenerate QR pass with updated room scope
   - Emits `upgrade/applied` event
6. Evidence trail: `upgrade/requested` → `upgrade/confirmed` → `credits/deducted` → `upgrade/applied` — complete audit chain.

**State mutation + evidence exercised:** Credit balance is GERT runtime state. Every mutation is traced. The upgrade spawns a sub-run because it has side effects beyond simple state changes (QR pass regeneration, room system integration).

---

#### 5.1.8 Stay Expires → Run Sealed + Evidence Archived

**Trigger:** Checkout date reached (timer fires) or operator initiates checkout.

**GERT mechanics:**
1. The stay's checkout timer (set at stay creation) fires on the checkout date.
2. **Finalization sequence begins:**
   - `revoke-guest-pass` (tool step): Calls `vacation.token-gen` with `action: revoke`. QR token invalidated. Evidence event: `pass/revoked`.
   - `archive-credit-ledger` (tool step): Calls `vacation.ledger` with `action: archive`. Final balances recorded. Evidence event: `credits/archived` with full ledger snapshot.
   - `seal-stay` (manual step, optional): Operator confirms checkout, captures any final notes.
3. **Run completes:** `run/completed` event emitted (final sequence number).
4. **Evidence archival:**
   - `run.yaml` completion manifest written to run directory
   - `trace.jsonl` is complete and sealed (no more events will be appended)
   - All snapshots, attachments, and annotations are in place
   - The run directory is a self-contained audit artifact
5. **QR pass invalidation is immediate:** The revocation event timestamp becomes the pass's effective `valid_to`. Any scan after this point is rejected.

**Resumability final exercise:** If the process crashes during finalization (e.g., after revoking the pass but before archiving credits), GERT resumes from the last checkpoint. The pass revocation is already done (tool step completed), so the ledger archival step is the resume point. No double-revocation, no lost ledger data.

**Replay:** The entire stay can be replayed by re-emitting `trace.jsonl` events. An adapter rendering the replay shows the exact sequence: check-in, each day's activities, weather replanning, upgrades, meals, and checkout. Indistinguishable from watching a live stay.

---

## Section 8: Prototype Scope (2-Week MVP)

### 8.1 Week 1: Foundation

**Goal:** A single stay template that checks in a guest, issues a QR pass, runs one day with morning/afternoon/dinner slots, and checks out.

#### Schemas to define

1. **`vacation-kit.yaml`** (gert-kit manifest):
   ```yaml
   apiVersion: kit/v1
   meta:
     name: gert.vacation
     version: "0.1.0"
   compatibility:
     gert-core: ">=2.0.0"
   schema:
     apiVersion-suffix: "vacation/v1"
     json-schema: "./schemas/vacation-v1.json"
   ```

2. **`schemas/vacation-v1.json`** — JSON Schema for vacation-specific types:
   - `stay-template` object (guest inputs, days array, credit allocations)
   - `day-skeleton` object (slots array with time boundaries)
   - `activity` object (name, duration, location, cost, prerequisites)
   - `meal-slot` object (type, time window, venues)

3. **`schemas/credit-ledger.json`** — Schema for the credit state structure

#### Stay template to prototype

**"Weekend Getaway" (2-night stay):**
- Day 1: Adventure flavor (morning hike, afternoon pool, dinner at main restaurant)
- Day 2: Relax flavor (morning spa, afternoon free, dinner at beach grill)
- Credits: 100 spa, 50 bar, 2 upgrade tokens
- QR scopes: pool, gym, restaurant, room

This is the simplest meaningful stay: 2 days, 2 flavors, 3 meal slots per day, basic credit allocation.

#### QR guest access mechanism

**Week 1 implementation: minimal token.**
- Tool: `vacation.token-gen` — a small Go binary or script
- Token format: base64-encoded JSON `{ guest_id, run_id, scopes, valid_from, valid_to, signature }`
- Signature: HMAC-SHA256 with a configured secret (reuse GERT's JWT infrastructure from Phase 17-18)
- QR encoding: use a standard QR library to render the token as a QR code (PNG output)
- Validation: a `vacation.token-verify` tool that checks signature + validity window + scopes
- **No external infrastructure** — token gen and verify are local tools. A real deployment would connect to a door-access system, but the MVP validates the token lifecycle within GERT.

#### Simple timeline/progress view

- Kit projection: `vacation/stay-timeline`
- Reads `trace.jsonl` from the run directory
- Outputs a chronological list: `[08:00 morning-start] [08:15 hike-started] [10:30 hike-completed] [12:30 lunch-start] ...`
- Implementation: ~100 lines of Go reading JSONL and formatting output
- **Can be a simple CLI tool** (`gert-vacation timeline <run-id>`) — no UI needed for MVP

#### Week 1 deliverables

| Deliverable | Implementation | Lines (est.) |
|---|---|---|
| `vacation-kit.yaml` manifest | YAML file | 30 |
| JSON Schema (vacation types) | JSON | 200 |
| Weekend Getaway stay template | Vacation YAML | 80 |
| Lowered runbook (compiler output) | Core YAML | 250 |
| Vacation compiler (lowering logic) | Go package | 400 |
| `vacation.token-gen` tool | Go binary | 150 |
| `vacation.token-verify` tool | Go binary | 100 |
| Timeline projection | Go binary | 100 |
| Integration test (full check-in → checkout) | Go test | 200 |
| **Total** | | **~1,510** |

**What can be `gert-kit.yaml` + lowered runbooks:** The manifest, schema, and stay template are pure Kit artifacts. The compiler is the Go package that does the lowering. The tools (`token-gen`, `token-verify`) are companion extension tools, NOT part of the Kit itself (per §4 architectural boundary).

### 8.2 Week 2: Intelligence

**Goal:** Add suggestion engine, operator override, credit ledger, and weather fallback to the weekend stay.

#### Suggestion engine (basic ranking)

- Tool: `vacation.suggest`
- Input: current state (flavor, time, weather, credits, completed activities)
- Algorithm (MVP): **weighted score** based on:
  - Time-appropriateness (morning activities score higher in morning)
  - Weather compatibility (outdoor activities penalized in rain)
  - Credit affordability (activities the guest can afford score higher)
  - Novelty (activities not yet done today score higher)
- Output: ranked list of `{ name, duration, cost, score, reason }`
- Implementation: ~200 lines Go. No ML, no personalization, no history — just contextual filtering and scoring.

#### Operator override

- Lowered as approval-gated manual steps (already designed in §4.9)
- Implementation: wire the `requires_approval` field in the lowered runbook
- Test: operator overrides a credit check → verify governance events emitted
- ~50 lines of additional lowering logic in the compiler

#### Credits/allowances ledger

- Tool: `vacation.ledger`
- Operations: `initialize`, `deduct`, `refund`, `query`, `archive`
- State: stored as GERT runtime variables (`credits.<category>.balance`)
- Evidence: every mutation emits a trace event
- Non-negativity enforced by default, operator override bypasses via governance gate
- Implementation: ~200 lines Go

#### Weather fallback

- Tool: `vacation.weather` (MVP: reads from a local JSON file, not a real API)
- Weather file: `weather.json` with hourly forecasts for the stay duration
- Watcher mode: polls the file every N seconds for changes (simulates real-time weather)
- On change: sets `weather.current` and `weather.changed` state variables
- Replanning: the day sub-run's weather branch triggers a replan
- Implementation: ~100 lines Go (file watcher + state setter)

#### Week 2 deliverables

| Deliverable | Implementation | Lines (est.) |
|---|---|---|
| `vacation.suggest` tool | Go binary | 200 |
| `vacation.ledger` tool | Go binary | 200 |
| `vacation.weather` tool | Go binary | 100 |
| Compiler updates (override, weather, suggestions) | Go (additions to compiler) | 150 |
| Updated stay template (with all features) | Vacation YAML | 40 |
| Updated lowered runbook | Core YAML | 150 |
| Test: suggestion flow | Go test | 80 |
| Test: operator override + governance | Go test | 80 |
| Test: credit deduction + insufficient balance | Go test | 80 |
| Test: weather replan | Go test | 80 |
| Test: full stay lifecycle (end-to-end) | Go test | 150 |
| `weather.json` test fixture | JSON | 30 |
| **Total** | | **~1,340** |

### 8.3 Deferred (Out of Scope for MVP)

| Feature | Reason to Defer |
|---|---|
| **Multi-guest stays** (family/group) | Adds sub-run-per-guest complexity; validate single-guest first |
| **Real weather API** | MVP uses local file; real API adds network dependency, rate limiting, error handling |
| **Persistent activity catalog** | MVP hardcodes activities in the stay template; catalog service is a v2 feature |
| **Guest preference learning** | Requires history across stays; MVP is stateless across stays |
| **Real QR scanning integration** | MVP validates token lifecycle in-process; physical scanner integration is deployment-specific |
| **Web/mobile UI** | MVP uses CLI/TUI adapter; UI is an adapter concern, not a Kit concern |
| **Multi-language support** | MVP is English-only; i18n is a projection/adapter concern |
| **Concurrent stays** | GERT v2.0 is single-run-per-process; multi-stay requires the v2.1 concurrency model |
| **Payment integration** | Credits are abstract units in MVP; payment gateway is an extension concern |
| **Activity capacity management** | MVP ignores capacity limits; real capacity needs a shared state service |
| **Day flavor swapping by guest** | MVP: flavors are set at template time; runtime flavor changes need the Day Modifier pipeline |

### 8.4 Implementation Notes

**What can be built as `gert-kit.yaml` + lowered runbooks (pure Kit, no Go):**
- The `vacation-kit.yaml` manifest
- JSON schemas for vacation types
- Stay templates (vacation YAML)
- Lowered runbooks (core YAML output)
- Test fixtures (input YAML + expected lowered YAML)
- Kit validation rules (expressible as JSON Schema constraints)
- Timeline projection (if implemented as a jq/shell script over JSONL)

**What needs a small Go package:**
- **Vacation compiler** (~400 LOC): The lowering logic that transforms vacation YAML → core YAML. This is the Kit's compiler binary referenced in `vacation-kit.yaml`. Could also be implemented as a standalone binary.
- **Companion tools** (~650 LOC total): `vacation.token-gen`, `vacation.token-verify`, `vacation.ledger`, `vacation.suggest`, `vacation.weather`. These are extension tools, NOT Kit components. They follow the `.tool.yaml` + binary pattern from GERT's tool runtime (§6).
- **Timeline projection** (~100 LOC): Could be a Go binary or a shell script. Go is simpler for JSONL parsing.

**Local-first constraints:**
- No cloud services, no databases, no external APIs (MVP)
- All state in GERT run directories (`.runbook/runs/`)
- All tools are local binaries
- QR tokens are self-contained (no token service)
- Weather is a local file (no weather API)

**Pragmatic shortcuts for MVP:**
- Activities are hardcoded in the stay template, not loaded from a catalog
- The suggestion engine is a simple scoring function, not a recommendation system
- Weather watcher polls a file, not an API
- Credit ledger uses GERT state variables, not a dedicated store
- QR validation is in-process, not a network service

**Total estimated effort:** ~2,850 lines across 2 weeks. This is achievable for a single developer (Brian) or a pair (Brian + one contributor). The Kit manifest, schemas, and templates can be authored in parallel by a non-Go contributor.

---

*End of Sections 1, 2, 4, 5, 8.*


---


---

## Section 3: Authoring DSL (v0)

**Author:** John (YAML Schema Specialist)

The Vacation Kit DSL is a set of YAML resource types that allow operators to author
stay templates, day structures, activities, and policies without working with GERT
core primitives directly. All resources use the `apiVersion/kind` discriminator
pattern; the Kit compiler lowers them to `runbook/v2` before execution.

Design constraints applied throughout:
- Every resource has `apiVersion: vacation.gert.io/v0` and a `kind` discriminator.
- `$ref:` uses relative file paths for cross-resource linking.
- Day Skeleton (time structure) and Day Flavor (activity pool bias) are separated.
- Compiler auto-generates weather branching from `weather_constraint`; authors do not write if/else.
- Credits are integer ledger state tracked by the runtime.
- Slot type is a strict enum: `activity | meal | free | rest | transfer`.
- JSON Schema Draft 2020-12 alignment with GERT v2 core schemas.
- Operator extension fields use `x-` prefix.

---

### 3.1 Kit Manifest (`gert-kit.yaml`)

The manifest declares the Kit's identity, minimum GERT core version, and the
resource types it introduces. The `gert` CLI reads this file when loading a Kit.

```yaml
apiVersion: kit/v1
kind: KitManifest
meta:
  name: gert.vacation
  display_name: "Vacation / Guided Stay Domain Kit"
  version: "0.1.0"
  author: "gert-team"
  description: >
    Domain Kit for guided vacation and hospitality stays.
    Provides stay templates, day skeleton/flavor authoring,
    activity pools, credit ledgers, and QR guest access.

compatibility:
  gert-core: ">=2.0.0"

resources:
  - kind: StayTemplate
    schema: "./schemas/stay-template.json"
  - kind: DaySkeleton
    schema: "./schemas/day-skeleton.json"
  - kind: DayFlavor
    schema: "./schemas/day-flavor.json"
  - kind: DayModifier
    schema: "./schemas/day-modifier.json"
  - kind: Activity
    schema: "./schemas/activity.json"
  - kind: ActivityPool
    schema: "./schemas/activity-pool.json"
  - kind: OperatorMessage
    schema: "./schemas/operator-message.json"

compiler:
  binary: "./bin/vacation-compiler"
  input_apiVersion: "vacation.gert.io/v0"
  output_apiVersion: "runbook/v2"

tools:
  - name: vacation.token-gen
    binary: "./bin/vacation-token-gen"
  - name: vacation.ledger
    binary: "./bin/vacation-ledger"
  - name: vacation.suggest
    binary: "./bin/vacation-suggest"
  - name: vacation.weather
    binary: "./bin/vacation-weather"
```

---

### 3.2 Stay Template

The Stay Template is the top-level authoring artifact for a guest stay. It declares
the duration, included services, credit allocations, day schedule, and governance rules.
At compile time, it becomes the parent runbook.

```yaml
apiVersion: vacation.gert.io/v0
kind: StayTemplate
meta:
  name: floripa-5day-premium
  display_name: "Florianópolis 5-Day Premium Stay"
  version: "1.0.0"
  operator: "surf-and-culture-floripa"
  tags: [floripa, premium, beach, culture]

spec:
  duration_days: 5
  location: "Florianópolis, SC, Brazil"
  timezone: "America/Sao_Paulo"

  # Guest inputs resolved at run creation time
  inputs:
    - name: guest_name
      type: string
      required: true
    - name: guest_id
      type: string
      required: true
    - name: checkin_date
      type: string    # ISO 8601 date: "2026-08-13"
      required: true
    - name: room_number
      type: string
      required: true
    - name: language
      type: string
      default: "en"

  # Credit allocations (integer units)
  credits:
    activity:
      initial: 20
      description: "Used for paid activities and upgrades"
    dining:
      initial: 10
      description: "Used for premium dining extras"
    upgrade:
      initial: 3
      description: "Used for room upgrades and premium options"

  # Package tier
  package: $ref: ./packages/premium.yaml

  # Day schedule: each day references a skeleton + default flavor
  days:
    - day: 1
      label: "Arrival & First Impressions"
      skeleton: $ref: ./skeletons/relaxed-morning.yaml
      flavor: $ref: ./flavors/food-day.yaml
      notes: "Arrival day — lighter schedule, orientation walk"

    - day: 2
      label: "Island Explorer"
      skeleton: $ref: ./skeletons/beach-morning.yaml
      flavor: $ref: ./flavors/beach-day.yaml

    - day: 3
      label: "Culture & Flavors"
      skeleton: $ref: ./skeletons/city-morning.yaml
      flavor: $ref: ./flavors/food-day.yaml

    - day: 4
      label: "Adventure Day"
      skeleton: $ref: ./skeletons/adventure-morning.yaml
      flavor: $ref: ./flavors/active-day.yaml

    - day: 5
      label: "Farewell Morning"
      skeleton: $ref: ./skeletons/relaxed-morning.yaml
      flavor: $ref: ./flavors/food-day.yaml
      notes: "Checkout before 12:00 — afternoon slots suppressed"
      checkout_by: "12:00"

  # Governance: who can override what
  governance:
    roles:
      operator:   [front-desk, concierge]
      supervisor: [manager]
    override_authority:
      credit_check_bypass:  [manager]
      flavor_change:        [concierge, manager]
      slot_substitution:    [concierge, manager]
    audit:
      redact_fields: ["*passport*", "*credit_card*", "*document_number*"]
      evidence_retention_days: 90
```

---

### 3.3 Day Skeleton

A Day Skeleton defines the time structure of a single day — what slots exist, their
types, and their time windows. Skeletons say nothing about what fills the slots;
that is the Day Flavor's responsibility.

Slot type enum: `activity | meal | free | rest | transfer`

```yaml
apiVersion: vacation.gert.io/v0
kind: DaySkeleton
meta:
  name: beach-morning
  description: "Early beach start, full afternoon, early dinner"

spec:
  wake_up: "07:00"
  slots:
    - id: morning-activity
      type: activity
      label: "Morning Activity"
      window: { start: "08:00", end: "12:00" }
      required: true

    - id: lunch
      type: meal
      label: "Lunch"
      window: { start: "12:30", end: "14:00" }
      required: true
      meal_type: lunch

    - id: afternoon-activity
      type: activity
      label: "Afternoon Activity"
      window: { start: "14:30", end: "18:00" }
      required: false

    - id: pre-dinner-rest
      type: rest
      label: "Pre-Dinner Rest"
      window: { start: "18:00", end: "19:30" }
      required: false

    - id: dinner
      type: meal
      label: "Dinner"
      window: { start: "19:30", end: "21:30" }
      required: true
      meal_type: dinner

    - id: evening
      type: free
      label: "Evening (free)"
      window: { start: "21:30", end: "23:00" }
      required: false
---
apiVersion: vacation.gert.io/v0
kind: DaySkeleton
meta:
  name: relaxed-morning
  description: "Late start, light morning, long afternoon, late dinner"

spec:
  wake_up: "09:00"
  slots:
    - id: late-breakfast
      type: meal
      label: "Breakfast"
      window: { start: "09:00", end: "10:30" }
      required: true
      meal_type: breakfast

    - id: morning-activity
      type: activity
      label: "Morning Activity"
      window: { start: "11:00", end: "13:30" }
      required: false

    - id: lunch
      type: meal
      label: "Lunch"
      window: { start: "13:30", end: "15:00" }
      required: true
      meal_type: lunch

    - id: afternoon-activity
      type: activity
      label: "Afternoon Activity"
      window: { start: "15:30", end: "18:30" }
      required: false

    - id: dinner
      type: meal
      label: "Dinner"
      window: { start: "20:00", end: "22:00" }
      required: true
      meal_type: dinner
---
apiVersion: vacation.gert.io/v0
kind: DaySkeleton
meta:
  name: adventure-morning
  description: "Early departure, packed morning, recovery afternoon"

spec:
  wake_up: "06:00"
  slots:
    - id: early-briefing
      type: activity
      label: "Adventure Briefing"
      window: { start: "06:30", end: "07:30" }
      required: true

    - id: main-adventure
      type: activity
      label: "Main Adventure"
      window: { start: "07:30", end: "13:00" }
      required: true

    - id: lunch
      type: meal
      label: "Post-Adventure Lunch"
      window: { start: "13:30", end: "15:00" }
      required: true
      meal_type: lunch

    - id: recovery
      type: rest
      label: "Recovery"
      window: { start: "15:00", end: "18:00" }
      required: false

    - id: dinner
      type: meal
      label: "Dinner"
      window: { start: "19:00", end: "21:00" }
      required: true
      meal_type: dinner
---
apiVersion: vacation.gert.io/v0
kind: DaySkeleton
meta:
  name: city-morning
  description: "Urban exploration — markets, culture, dining"

spec:
  wake_up: "08:30"
  slots:
    - id: morning-activity
      type: activity
      label: "Morning Exploration"
      window: { start: "09:00", end: "12:30" }
      required: true

    - id: lunch
      type: meal
      label: "Lunch"
      window: { start: "12:30", end: "14:00" }
      required: true
      meal_type: lunch

    - id: afternoon-activity
      type: activity
      label: "Afternoon Culture"
      window: { start: "14:30", end: "17:30" }
      required: false

    - id: dinner
      type: meal
      label: "Dinner"
      window: { start: "19:00", end: "21:30" }
      required: true
      meal_type: dinner
```

---

### 3.4 Day Flavor

A Day Flavor selects the activity pool and constrains slot resolution for a given day.
Flavors bias the compiler's activity selection without specifying exact activities.
The `weather_constraint` field drives compiler-generated branching — authors do not
write weather if/else logic manually.

```yaml
apiVersion: vacation.gert.io/v0
kind: DayFlavor
meta:
  name: beach-day
  description: "Ocean-focused day — water sports, beach time, seafood"

spec:
  activity_pool: $ref: ./pools/beach-pool.yaml
  weather_constraint: outdoor     # compiler generates rain fallback automatically
  slot_bias:
    morning-activity:
      prefer_tags: [beach, water, outdoor, morning]
    afternoon-activity:
      prefer_tags: [beach, water, active]
  fallback_flavor: $ref: ./flavors/rain-day.yaml   # used when weather=rain
---
apiVersion: vacation.gert.io/v0
kind: DayFlavor
meta:
  name: food-day
  description: "Culinary day — markets, restaurants, local cuisine"

spec:
  activity_pool: $ref: ./pools/food-culture-pool.yaml
  weather_constraint: any         # food tours work rain or shine
  slot_bias:
    morning-activity:
      prefer_tags: [food, market, cultural, morning]
    afternoon-activity:
      prefer_tags: [food, cultural, cooking]
---
apiVersion: vacation.gert.io/v0
kind: DayFlavor
meta:
  name: active-day
  description: "High-energy outdoor adventure — hiking, kayaking, ziplining"

spec:
  activity_pool: $ref: ./pools/beach-pool.yaml
  weather_constraint: outdoor
  slot_bias:
    morning-activity:
      prefer_tags: [adventure, outdoor, physical, morning]
    afternoon-activity:
      prefer_tags: [adventure, outdoor, physical]
  fallback_flavor: $ref: ./flavors/rain-day.yaml
---
apiVersion: vacation.gert.io/v0
kind: DayFlavor
meta:
  name: nightlife-day
  description: "Late start, afternoon culture, vibrant evening"

spec:
  activity_pool: $ref: ./pools/food-culture-pool.yaml
  weather_constraint: any
  slot_bias:
    morning-activity:
      prefer_tags: [relaxing, cultural]
    afternoon-activity:
      prefer_tags: [cultural, shopping]
    evening:
      prefer_tags: [nightlife, music, social]
---
apiVersion: vacation.gert.io/v0
kind: DayFlavor
meta:
  name: rain-day
  description: "Indoor fallback — spa, cooking, museums, markets"

spec:
  activity_pool: $ref: ./pools/indoor-pool.yaml
  weather_constraint: indoor
  slot_bias:
    morning-activity:
      prefer_tags: [indoor, spa, cultural]
    afternoon-activity:
      prefer_tags: [indoor, cooking, art]
```

---

### 3.5 Activity

An Activity is a concrete guest experience. Activities reference a weather constraint;
the compiler generates branching logic so that outdoor activities are automatically
replaced when rain is detected.

```yaml
apiVersion: vacation.gert.io/v0
kind: Activity
meta:
  name: standup-paddleboard-lesson
  display_name: "Stand-Up Paddleboard Lesson"
  tags: [beach, water, active, outdoor, morning, beginner-friendly]

spec:
  duration_minutes: 90
  location:
    name: "Lagoa da Conceição Paddleboard Center"
    coordinates: [-27.6074, -48.4639]
    meeting_point: "Dock at the south end of Lagoa da Conceição"
  capacity:
    max_concurrent: 8
    booking_required: true
  credits_cost: 3
  weather_constraint: outdoor   # compiler generates indoor alternative path
  prerequisites: []
  included_in_packages: [premium, standard]
  x-operator-notes: "Confirm instructor availability 24h before"
---
apiVersion: vacation.gert.io/v0
kind: Activity
meta:
  name: mercado-publico-food-tour
  display_name: "Mercado Público Food Tour"
  tags: [food, cultural, indoor, morning, guided]

spec:
  duration_minutes: 120
  location:
    name: "Mercado Público de Florianópolis"
    coordinates: [-27.5970, -48.5497]
    meeting_point: "Main entrance, Rua Conselheiro Mafra"
  capacity:
    max_concurrent: 12
    booking_required: false
  credits_cost: 1
  weather_constraint: any       # indoor market — weather irrelevant
  prerequisites: []
  included_in_packages: [premium, standard]
---
apiVersion: vacation.gert.io/v0
kind: Activity
meta:
  name: ilha-do-campeche-snorkel
  display_name: "Ilha do Campeche Snorkeling Tour"
  tags: [beach, water, outdoor, adventure, morning, snorkel]

spec:
  duration_minutes: 240
  location:
    name: "Ilha do Campeche"
    coordinates: [-27.7194, -48.4786]
    meeting_point: "Boat departure from Armação Beach"
  capacity:
    max_concurrent: 15
    booking_required: true
  credits_cost: 4
  weather_constraint: outdoor
  prerequisites:
    - can_swim: true
  included_in_packages: [premium]
  x-operator-notes: "Check island access quota — capped at 100 visitors/day"
---
apiVersion: vacation.gert.io/v0
kind: Activity
meta:
  name: spa-treatment-session
  display_name: "Spa Treatment Session"
  tags: [spa, indoor, relaxing, wellness, afternoon]

spec:
  duration_minutes: 60
  location:
    name: "Resort Spa — Level B1"
    coordinates: null    # on-premises
    meeting_point: "Reception desk, Level B1"
  capacity:
    max_concurrent: 4
    booking_required: true
  credits_cost: 2
  weather_constraint: indoor
  prerequisites: []
  included_in_packages: [premium]
```

---

### 3.6 Activity Pool

An Activity Pool groups activities by context — used by flavors to constrain which
activities are eligible for a given day's slots. Tag-based filtering keeps pools
declarative and recomposable.

```yaml
apiVersion: vacation.gert.io/v0
kind: ActivityPool
meta:
  name: beach-pool
  description: "Outdoor beach and water activities for Florianópolis"

spec:
  activities:
    - $ref: ./activities/standup-paddleboard-lesson.yaml
    - $ref: ./activities/ilha-do-campeche-snorkel.yaml
    - $ref: ./activities/surf-lesson-joaquina.yaml
    - $ref: ./activities/lagoa-kayak.yaml
    - $ref: ./activities/beach-volleyball.yaml
    - $ref: ./activities/sunset-boat-tour.yaml

  filter:
    require_tags: [outdoor]       # only outdoor activities qualify
    exclude_tags: [nightlife]
---
apiVersion: vacation.gert.io/v0
kind: ActivityPool
meta:
  name: food-culture-pool
  description: "Food, cultural, and market experiences — weather-agnostic"

spec:
  activities:
    - $ref: ./activities/mercado-publico-food-tour.yaml
    - $ref: ./activities/local-cooking-class.yaml
    - $ref: ./activities/historic-center-walk.yaml
    - $ref: ./activities/cachaça-tasting.yaml
    - $ref: ./activities/art-gallery-tour.yaml

  filter:
    require_tags: [cultural, food]    # food or cultural tag required
    exclude_tags: []
---
apiVersion: vacation.gert.io/v0
kind: ActivityPool
meta:
  name: indoor-pool
  description: "Rain-day fallback — indoor activities only"

spec:
  activities:
    - $ref: ./activities/spa-treatment-session.yaml
    - $ref: ./activities/local-cooking-class.yaml
    - $ref: ./activities/art-gallery-tour.yaml
    - $ref: ./activities/cachaça-tasting.yaml

  filter:
    require_tags: [indoor]
    exclude_tags: [outdoor]
---
apiVersion: vacation.gert.io/v0
kind: ActivityPool
meta:
  name: nightlife-pool
  description: "Evening and nightlife experiences"

spec:
  activities:
    - $ref: ./activities/sunset-boat-tour.yaml
    - $ref: ./activities/live-music-bar.yaml
    - $ref: ./activities/forró-dance-class.yaml

  filter:
    require_tags: [nightlife, evening]
    exclude_tags: []
```

---

### 3.7 Package / Entitlements

Packages define what is included in a guest's stay tier. They extend the credit
allocations with concrete inclusions and named upgrade options.

```yaml
apiVersion: vacation.gert.io/v0
kind: Package
meta:
  name: premium
  display_name: "Premium Guided Stay"

spec:
  credits:
    activity: 20
    dining:   10
    upgrade:   3

  inclusions:
    - type: transport
      label: "Airport pickup (round-trip)"
      included: true

    - type: activity
      label: "Welcome dinner at Ostradamus"
      activity: $ref: ./activities/ostradamus-dinner.yaml
      day: 1
      slot: dinner
      included: true

    - type: activity
      label: "Ilha do Campeche snorkeling tour"
      activity: $ref: ./activities/ilha-do-campeche-snorkel.yaml
      day: 2
      slot: morning-activity
      included: true

    - type: service
      label: "Concierge reservation assistance (up to 3 bookings)"
      quota: 3

  upgrade_options:
    - id: ocean-view-room
      label: "Upgrade to ocean-view room"
      credits_cost: 2
      category: upgrade
      description: "Available at check-in or during stay if rooms available"

    - id: private-guide
      label: "Private guide for one activity"
      credits_cost: 3
      category: activity
      description: "Any activity replaced with private guided version"

    - id: late-checkout
      label: "Late checkout (until 15:00)"
      credits_cost: 1
      category: upgrade
      description: "Subject to availability; request before 09:00 on checkout day"
---
apiVersion: vacation.gert.io/v0
kind: Package
meta:
  name: standard
  display_name: "Standard Guided Stay"

spec:
  credits:
    activity: 10
    dining:    5
    upgrade:   1

  inclusions:
    - type: activity
      label: "Welcome orientation walk"
      day: 1
      slot: morning-activity
      included: true

  upgrade_options:
    - id: ocean-view-room
      label: "Upgrade to ocean-view room"
      credits_cost: 2
      category: upgrade
```

---

### 3.8 Operator Message

Operator messages are broadcast to guests via the stay run's event stream. Three
axes control routing and display: `urgency`, `target`, and `action`.

```yaml
apiVersion: vacation.gert.io/v0
kind: OperatorMessage
meta:
  name: sunset-boat-confirmation
  description: "Confirm sunset boat tour for tonight"

spec:
  urgency: info          # info | warning | critical
  target: guest          # guest (this run only) | all (all active runs)
  action: augment        # augment (add to suggestions) | replace (override top suggestion)
  body: >
    Your sunset boat tour is confirmed for tonight at 18:30.
    Meet at the marina entrance. Bring a light jacket — it gets cool on the water.
  sent_at: "{{.current_time}}"     # resolved at emit time
---
apiVersion: vacation.gert.io/v0
kind: OperatorMessage
meta:
  name: rain-alert-afternoon

spec:
  urgency: warning
  target: all
  action: replace        # replaces the current top suggestion for all guests
  body: >
    Heavy rain expected from 14:00 onwards. Outdoor afternoon activities have been
    updated to indoor alternatives. Check your updated schedule in the app.
---
apiVersion: vacation.gert.io/v0
kind: OperatorMessage
meta:
  name: emergency-evacuation

spec:
  urgency: critical
  target: all
  action: replace
  body: >
    URGENT: Please proceed to the main lobby immediately.
    Follow staff instructions. This is not a drill.
```

---

### 3.9 Day Modifier

A Day Modifier is a runtime patch applied to an already-planned day. It is authored
as a template and triggered either by weather threshold or by operator action. The
compiler maps each trigger type to the appropriate GERT primitive.

```yaml
apiVersion: vacation.gert.io/v0
kind: DayModifier
meta:
  name: rain-afternoon-switch
  description: "Switch afternoon outdoor slots to indoor on rain"

spec:
  trigger:
    type: weather         # weather | operator-manual
    condition:
      weather_is: rain
      threshold_pct: 70   # trigger if rain probability >= 70%
      applies_to: afternoon-activity

  slot_overrides:
    - slot_id: afternoon-activity
      substitute_pool: $ref: ./pools/indoor-pool.yaml
      reason: "Outdoor activity replaced due to rain forecast"

  lock: false             # false = guest can still override; true = operator-locked
---
apiVersion: vacation.gert.io/v0
kind: DayModifier
meta:
  name: operator-birthday-upgrade
  description: "Operator manually upgrades a guest's day on their birthday"

spec:
  trigger:
    type: operator-manual
    requires_role: concierge

  slot_overrides:
    - slot_id: dinner
      substitute_activity: $ref: ./activities/private-rooftop-dinner.yaml
      reason: "Birthday upgrade — complimentary private dinner"
    - slot_id: afternoon-activity
      substitute_activity: $ref: ./activities/spa-treatment-session.yaml
      credit_override: true     # bypass credit check for this substitution
      reason: "Birthday gift — spa session complimentary"

  lock: true              # operator-locked: guest cannot revert this modifier
```

**Compiler behavior for `DayModifier`:**
- `trigger.type: weather` lowers to a GERT event watcher (`vacation.weather` tool) +
  a branch step that fires when the threshold is crossed, spawning a replanning sub-run.
- `trigger.type: operator-manual` lowers to a governance-gated `manual` step that
  the operator completes; the `lock` flag controls whether the guest's advisory task
  can still override.
- `credit_override: true` lowers to a bypass of the credit-check branch, guarded by
  the operator's role in the approval gate.

---

### Design Decisions (Section 3)

| ID | Decision | Rationale |
|----|----------|-----------|
| VK-01 | `apiVersion/kind` discriminator on every resource | Enables schema validation per resource type; aligns with GERT v2 core schema conventions |
| VK-02 | `$ref:` cross-file linking (relative paths) | Skeleton and flavor reuse across stay templates without copy-paste; follows YAML referencing idioms |
| VK-03 | Skeleton ≠ Flavor — strict separation | Skeleton = time structure; Flavor = activity bias. Allows same skeleton with multiple flavors and vice versa |
| VK-04 | Compiler auto-generates weather branching | Authors declare `weather_constraint: outdoor`; compiler writes the branch. Authors never write weather if/else |
| VK-05 | Credits are integer ledger units | Avoids floating-point accounting errors; integer semantics are clear and auditable |
| VK-06 | Slot type is a strict enum | Prevents ambiguous slot definitions; compiler knows exactly what to generate per type |
| VK-07 | JSON Schema Draft 2020-12 | Aligns with GERT v2 core schemas; enables `gert validate` to work on kit resources |
| VK-08 | Operator extension fields use `x-` prefix | Prevents collisions with kit-defined fields; preserves forward compatibility |
| VK-09 | ActivityPool uses tag filter model | Declarative and composable; pools are defined once and referenced by multiple flavors |
| VK-10 | DayModifier is separate from OperatorMessage | Messages communicate; Modifiers change the plan. Conflating them would couple notification logic with replanning logic |
| VK-11 | `trigger.type` determines compilation target | `weather` → event watcher + branch; `operator-manual` → governance gate. One field drives the entire lowering path |
| VK-12 | `lock: bool` on DayModifier | Explicit control over whether guest advisory can override an operator modification |
| VK-13 | Fallback flavor referenced in primary flavor | Decouples rain handling from the compiler — compiler follows the `fallback_flavor` reference; no hardcoded rain logic |



---

# Vacation Domain Kit — Integration Design
## Sections 6, 7, and 9

**Author:** Barbara (Integrations Specialist)  
**Date:** 2026-04-21  
**Requested by:** Cristian  
**Status:** Draft — for team review

---

## Section 6: Mobile UX Contract

The mobile app is a read-mostly client of `gert serve`. It does not own execution state; the GERT runtime does. This section defines the integration surface — the exact contract between the runtime and the mobile app — not the UI itself.

---

### 6.1 App Access Model

#### QR Code → Bearer Token Flow

The stay begins when the operator issues a QR code to the guest. The QR code encodes a one-time-use provisioning URL:

```
https://<operator-host>/stay/provision?code=<opaque-token>
```

The provisioning endpoint:
1. Validates the opaque token against the Stay Run ID in the operator's system
2. Mints a short-lived JWT scoped to that specific Stay Run
3. Returns the JWT and the SSE stream URL to the mobile app
4. Invalidates the provisioning code (one-time use)

The JWT carries:
```json
{
  "sub": "guest:<stay_run_id>",
  "run_id": "<stay_run_id>",
  "scope": "run:read sse:subscribe",
  "iat": 1713700000,
  "exp": 1713786400,
  "nbf": 1713700000
}
```

**Scope is strictly `run:read` + `sse:subscribe`.** The guest cannot enumerate other runs, cannot access any run other than their own Stay Run, and cannot write to run state directly (all writes go through discrete write endpoints, enumerated in §6.3).

#### Token Lifetime

- **Initial expiry:** aligned to stay end date + 4-hour buffer (so a guest's last morning doesn't break)
- **Maximum lifetime:** 14 days (hard cap regardless of stay length)
- **Token is long-lived by design.** Mobile browsers don't have reliable cookie/session persistence. A token that expires mid-hike is a bad experience.

#### Refresh Flow

The app holds a refresh token alongside the access JWT (refresh token is opaque, stored in localStorage). When the access JWT is within 1 hour of expiry:

1. App POST `POST /auth/refresh` with the refresh token
2. Server validates refresh token against the Stay Run (still active, not cancelled)
3. Server returns new access JWT (same scopes, new expiry)
4. App replaces stored JWT; no user action required

The refresh token itself is single-use. Each refresh rotation issues a new refresh token.

#### Expired Token Mid-Stay

If the access JWT expires and the refresh token is also expired or invalid (e.g., the stay was administratively cancelled):

- `gert serve` returns `401 Unauthorized` on all requests
- The SSE stream closes with a `401` event
- The app shows a locked screen: "Your stay access has ended. Contact your operator."
- Cached last-known state remains visible (read-only)
- No automatic retry loop — the guest must contact the operator to re-provision

If the stay is still active and this is a token error (not intentional revocation), the operator can re-issue a new QR code. The new QR code provisions a new JWT pointing to the same Stay Run — all run state is preserved.

---

### 6.2 Data Surfaces (Read Contract)

All surfaces are read via `gert serve` JSON-RPC calls with the guest bearer token. The underlying data lives in the Stay Run's state, events, human tasks, and evidence records.

Each surface maps to a `run.query` or `run.get` call scoped to the guest's Stay Run ID.

---

#### 6.2.1 Current Status

**What it shows:** Current step, current day number, stay progress percentage.

**GERT primitive:** Run state (`run.get` → `state.current_step`, `state.day_index`, `state.total_days`)

**Shape:**
```json
{
  "surface": "status",
  "run_id": "...",
  "current_step": "day-3-morning-slot",
  "day_index": 3,
  "total_days": 7,
  "progress_pct": 42,
  "stay_phase": "active"
}
```

`stay_phase` values: `pre-arrival` | `active` | `completed` | `cancelled`

---

#### 6.2.2 What's Next

**What it shows:** The single next suggested activity + its slot details (time, location, duration, credit cost).

**GERT primitive:** Human task (the current advisory slot task) + run state (resolved suggestion from AI layer, §7)

**Shape:**
```json
{
  "surface": "whats_next",
  "slot": "afternoon",
  "activity": {
    "activity_id": "praia-da-joaquina-surf",
    "name": "Surf Lesson at Joaquina",
    "location": "Praia da Joaquina",
    "duration_min": 120,
    "credit_cost": 2,
    "tags": ["outdoor", "active", "beach"],
    "reason": "Great beach conditions today, fits your afternoon slot"
  },
  "slot_starts_at": "2026-08-15T14:00:00-03:00",
  "confirmed": false
}
```

`confirmed: false` until the guest explicitly accepts (§6.3).

---

#### 6.2.3 Suggested Activities

**What it shows:** Ranked list of alternatives for the current slot. Guest can browse and pick a different option.

**GERT primitive:** Run state (AI-ranked suggestion list from §7, stored in the current slot's task context)

**Shape:**
```json
{
  "surface": "suggestions",
  "slot": "afternoon",
  "ranked": [
    {
      "activity_id": "praia-da-joaquina-surf",
      "name": "Surf Lesson at Joaquina",
      "score": 0.91,
      "credit_cost": 2,
      "tags": ["outdoor", "active"],
      "reason": "Great beach conditions today"
    },
    {
      "activity_id": "mercado-publico-tour",
      "name": "Mercado Público Food Tour",
      "score": 0.74,
      "credit_cost": 1,
      "tags": ["cultural", "food"],
      "reason": "Popular with guests who enjoyed the harbor walk"
    }
  ]
}
```

---

#### 6.2.4 Upcoming

**What it shows:** Meals, confirmed reservations, and reminders for the rest of today.

**GERT primitive:** Events (scheduled timer events for meal times) + human tasks (confirmed reservations) + run state (daily schedule)

**Shape:**
```json
{
  "surface": "upcoming",
  "items": [
    {
      "type": "meal",
      "label": "Dinner — Casa da Lagoa",
      "at": "2026-08-15T19:30:00-03:00",
      "status": "confirmed"
    },
    {
      "type": "reminder",
      "label": "Sunscreen reminder before afternoon slot",
      "at": "2026-08-15T13:30:00-03:00",
      "status": "pending"
    }
  ]
}
```

---

#### 6.2.5 Budget

**What it shows:** Credits remaining, credits spent, per-category breakdown.

**GERT primitive:** Run state (credit ledger — a structured field on the Stay Run, updated by credit-consume events)

**Shape:**
```json
{
  "surface": "budget",
  "total_credits": 20,
  "spent_credits": 6,
  "remaining_credits": 14,
  "breakdown": [
    { "category": "activities", "spent": 4 },
    { "category": "upgrades", "spent": 2 },
    { "category": "meals", "spent": 0 }
  ]
}
```

The credit ledger is append-only. Each credit-consume event writes a debit entry. The `budget` surface is a projection over those events — it does not store a single mutable "balance" field.

---

#### 6.2.6 Messages

**What it shows:** Operator messages (announcements, personal notes), unread count, urgency level.

**GERT primitive:** Events (operator-message events emitted to the Stay Run) + run state (read/unread tracking per guest acknowledgement)

**Shape:**
```json
{
  "surface": "messages",
  "unread_count": 1,
  "items": [
    {
      "message_id": "msg-001",
      "from": "Operator",
      "body": "Your sunset boat tour is confirmed for 6pm tonight.",
      "sent_at": "2026-08-15T10:00:00-03:00",
      "urgency": "normal",
      "read": false
    }
  ]
}
```

`urgency` values: `normal` | `high` | `urgent`

---

#### 6.2.7 Alerts

**What it shows:** Weather changes, operator overrides, urgent notices. These surface as push-style interrupts via SSE, not just on-demand reads.

**GERT primitive:** Events (weather-alert, operator-override events emitted to the Stay Run)

**Shape (SSE event payload + read endpoint):**
```json
{
  "surface": "alerts",
  "items": [
    {
      "alert_id": "alert-003",
      "type": "weather",
      "title": "Rain expected this afternoon",
      "body": "Outdoor activities may be affected. Indoor alternatives have been updated.",
      "severity": "warning",
      "emitted_at": "2026-08-15T11:45:00-03:00",
      "acknowledged": false
    }
  ]
}
```

`type` values: `weather` | `operator_override` | `urgent_notice` | `cancellation`

---

#### 6.2.8 Timeline

**What it shows:** Full stay view — past days (sealed/read-only), current day (live), future days (planned).

**GERT primitive:** Run state (day structure) + events (sealed day summaries as evidence records) + human tasks (planned future slots)

**Shape:**
```json
{
  "surface": "timeline",
  "days": [
    {
      "day_index": 1,
      "date": "2026-08-13",
      "label": "Arrival Day",
      "status": "sealed",
      "summary": "Checked in, harbor walk, dinner at Ostradamus",
      "activities_done": ["harbor-walk", "ostradamus-dinner"]
    },
    {
      "day_index": 2,
      "date": "2026-08-14",
      "label": "Island Explorer",
      "status": "sealed",
      "summary": "Ilha do Campeche snorkeling, lunch at Lagoa",
      "activities_done": ["campeche-snorkel", "lagoa-lunch"]
    },
    {
      "day_index": 3,
      "date": "2026-08-15",
      "label": "Beach & Culture",
      "status": "live",
      "slots": [
        { "slot": "morning", "status": "completed", "activity": "mercado-walk" },
        { "slot": "afternoon", "status": "pending", "activity": null },
        { "slot": "evening", "status": "planned", "activity": "sunset-boat" }
      ]
    },
    {
      "day_index": 4,
      "date": "2026-08-16",
      "label": "Adventure Day",
      "status": "planned",
      "slots": null
    }
  ]
}
```

Past days (`status: "sealed"`) have their evidence record attached. The current day is live with slot-by-slot tracking. Future days show the planned flavor/label but not slot details (those are resolved when the day becomes live).

---

### 6.3 Write Contract

The app can write back to GERT through a small, strictly bounded set of write operations. All writes go through `gert serve` JSON-RPC methods that produce events or resolve human tasks on the Stay Run. The guest token has no direct state-mutation access.

---

#### 6.3.1 Guest Confirms / Selects Activity

**Action:** Guest accepts the suggested activity for a slot, or picks an alternative from the ranked list.

**GERT operation:** Resolves the current slot's advisory human task with the selected activity ID.

```jsonc
// RPC call
{
  "method": "run.task.resolve",
  "params": {
    "run_id": "<stay_run_id>",
    "task_id": "<slot_task_id>",
    "resolution": {
      "type": "activity_selected",
      "activity_id": "praia-da-joaquina-surf"
    }
  }
}
```

The runtime then advances the slot state, records the selection as an event, and emits a `slot-changed` SSE event.

---

#### 6.3.2 Guest Accepts Upgrade

**Action:** Guest accepts an upgrade offer (e.g., premium boat tour instead of standard). Consumes credits.

**GERT operation:** Emits a `credit-consume` event and resolves the upgrade human task.

```jsonc
{
  "method": "run.task.resolve",
  "params": {
    "run_id": "<stay_run_id>",
    "task_id": "<upgrade_task_id>",
    "resolution": {
      "type": "upgrade_accepted",
      "upgrade_id": "sunset-boat-premium",
      "credits_to_consume": 3
    }
  }
}
```

The runtime validates credits available before accepting. If credits are insufficient, the task resolution is rejected with `INSUFFICIENT_CREDITS` and the app shows an error. The guest cannot overdraft.

---

#### 6.3.3 Guest Uploads Document

**Action:** Guest uploads a required document (e.g., scanned passport for tour operator, medical waiver).

**GERT operation:** Attaches the upload as an evidence record on the Stay Run.

```jsonc
{
  "method": "run.evidence.attach",
  "params": {
    "run_id": "<stay_run_id>",
    "label": "passport_scan",
    "content_type": "image/jpeg",
    "data": "<base64-encoded>",
    "linked_task_id": "<document_task_id>"
  }
}
```

Max upload size: 5MB. Types: `image/jpeg`, `image/png`, `application/pdf`. The runtime stores the evidence record; the operator can retrieve it via the admin interface.

---

#### 6.3.4 Guest Acknowledges Alert

**Action:** Guest taps "Got it" on a weather alert or operator notice.

**GERT operation:** Emits an `alert-acknowledged` event on the Stay Run.

```jsonc
{
  "method": "run.event.emit",
  "params": {
    "run_id": "<stay_run_id>",
    "event_type": "alert-acknowledged",
    "payload": { "alert_id": "alert-003" }
  }
}
```

This is non-blocking — the runtime records the acknowledgement; no task is resolved. The alert transitions from `acknowledged: false` to `acknowledged: true` in the alerts surface.

---

#### 6.3.5 Guest Requests Operator Assistance

**Action:** Guest sends a help request (lost item, question, complaint).

**GERT operation:** Emits an `assistance-requested` event and creates a human task assigned to the operator.

```jsonc
{
  "method": "run.event.emit",
  "params": {
    "run_id": "<stay_run_id>",
    "event_type": "assistance-requested",
    "payload": {
      "category": "question",
      "message": "Can we extend checkout by 2 hours?",
      "urgency": "normal"
    }
  }
}
```

The operator sees this in their admin view. If/when they respond, it appears as a new operator message in §6.2.6.

---

### 6.4 Realtime Updates

The app connects to a persistent SSE stream for the Stay Run immediately after provisioning. This avoids polling and gives sub-second latency for urgent updates (weather changes, operator overrides).

**Stream endpoint:**
```
GET /sse/run/<stay_run_id>
Authorization: Bearer <guest_jwt>
Accept: text/event-stream
```

This reuses the existing `gert serve` SSE infrastructure (implemented in Phase 16).

#### Events to Subscribe To

| SSE Event Type       | Triggers When                                              | App Reaction                                         |
|----------------------|------------------------------------------------------------|------------------------------------------------------|
| `slot-changed`       | Guest confirms activity, operator overrides slot           | Refresh `whats_next` and `suggestions` surfaces      |
| `operator-message`   | Operator sends a message to this guest's run               | Increment unread badge, show toast notification      |
| `weather-alert`      | Weather provider emits rain/storm event to the run         | Show alert banner, trigger AI re-rank (§7.4)        |
| `credit-consumed`    | Any credit debit recorded on the run                       | Refresh `budget` surface                             |
| `alert-acknowledged` | Guest acknowledged; update local state optimistically      | (No-op for receiving device; useful for multi-device)|
| `day-sealed`         | Day transitions to sealed at midnight                      | Refresh `timeline` surface, advance day_index        |
| `run-completed`      | Stay run reaches terminal state                            | Show end-of-stay screen, gracefully close stream     |

#### Event Payload Shape

All SSE events follow:
```
event: slot-changed
data: {"run_id": "...", "event_type": "slot-changed", "payload": {...}, "emitted_at": "..."}
```

The app receives the event and refetches the affected surface(s). It does not attempt to derive new state purely from event payloads — events are notifications, not state deltas. The source of truth is always `gert serve`.

#### Reconnect Behavior

The app uses the `Last-Event-ID` SSE header for reconnect. `gert serve` replays events since that ID (within a configurable window, default: 5 minutes of event history). If the gap is too large or the event history is unavailable, the app receives a `stream-reset` synthetic event and performs a full refresh of all surfaces.

```
event: stream-reset
data: {"reason": "gap_too_large", "replay_from": null}
```

Reconnect back-off: 1s → 2s → 4s → 8s → 16s (max). After 5 failed reconnects, show the degraded mode indicator (§6.5) and stop retrying until the user taps "Reconnect."

---

### 6.5 Offline / Degraded Mode

Mobile connections drop. The app must be usable when disconnected, with clear affordances about what works and what doesn't.

#### What the App Shows When Disconnected

- **Last-known state is displayed as-is** — all surfaces show their last-fetched data with a "last updated X minutes ago" label
- **An offline banner is visible** — non-intrusive, persistent while disconnected
- **Time-sensitive data is flagged** — alerts surface shows a "may be outdated" indicator
- **Budget surface is read-only but visible** — credits display may be stale

#### Which Actions Are Deferred vs Blocked

| Action                        | Status     | Notes                                                                 |
|-------------------------------|------------|-----------------------------------------------------------------------|
| Confirm activity              | **Deferred** | Queued locally, submitted when reconnected. App shows "pending" state |
| Acknowledge alert             | **Deferred** | Queued locally, submitted on reconnect                                |
| Accept upgrade                | **Blocked**  | Requires live credit validation — app shows "Connect to confirm"     |
| Upload document               | **Blocked**  | File upload requires connectivity — app shows "Connect to upload"    |
| Request operator assistance   | **Deferred** | Queued locally (message + timestamp), submitted on reconnect          |

Deferred actions are stored in localStorage. On reconnect, they are replayed in order. If a deferred action fails (e.g., the slot has already been advanced by the operator), the app shows a reconciliation notice.

#### Reconnect Indicator

A subtle "Reconnecting…" spinner replaces the "last updated" label while reconnecting. It disappears automatically on successful reconnect and SSE stream re-establishment.

---

## Section 7: AI Assistance Layer

AI assists guests by ranking activity suggestions. It does not own execution truth. The GERT runtime enforces constraints (credits, availability, operator policy). AI provides ranked suggestions and short explanations that make the guest experience feel curated and personal.

---

### 7.1 Architecture Position

```
┌─────────────────────────────────────────────────────────────────┐
│                        GERT Runtime                             │
│   Stay Run state machine  ←→  Constraints (credits, capacity)  │
│   Human Task: slot advisory task (current slot)                │
│                    │                                            │
│                    ▼                                            │
│          Suggestion Resolver (Go)                               │
│          Reads: available activities + constraints              │
│          Calls: AI Ranker (tool invocation or HTTP)            │
│          Falls back: static ranking if AI unavailable          │
│                    │                                            │
│                    ▼                                            │
│          Ranked list written to slot task context              │
└─────────────────────────────────────────────────────────────────┘
                    │
                    ▼
         Mobile App reads `suggestions` surface
```

**Key constraint:** AI is a read-only advisor. It receives inputs and returns a ranked list. It does not modify run state, emit events, or control execution flow. The Suggestion Resolver is the component that acts on AI output — it writes the ranked list to the slot task context, where the runtime and app can read it.

If AI is unavailable, the Suggestion Resolver falls back to static ranking (operator-configured priority order). The guest still gets a list; it's just not personalized.

---

### 7.2 Ranking Inputs

The Suggestion Resolver assembles this input context before calling the AI Ranker:

| Signal                    | Source                                        | Example                                      |
|---------------------------|-----------------------------------------------|----------------------------------------------|
| `current_day_flavor`      | Run state (day descriptor from stay template) | `"beach_and_culture"`                        |
| `slot_type`               | Run state (current slot)                      | `"afternoon"` / `"morning"` / `"evening"`   |
| `weather_current`         | Weather provider (tool invocation)            | `"sunny"` / `"rainy"` / `"partly_cloudy"`   |
| `activities_done`         | Events log (all `activity_selected` events)   | `["campeche-snorkel", "harbor-walk"]`        |
| `categories_done`         | Derived from `activities_done`                | `["outdoor", "cultural"]`                    |
| `credits_remaining`       | Run state (credit ledger projection)          | `14`                                         |
| `guest_tags`              | Run state (optional guest profile tags)       | `["beginner_surfer", "vegetarian"]`          |
| `operator_promoted`       | Activity catalog (operator-flagged entries)   | `["sunset-boat-premium"]`                    |
| `available_activities`    | Activity catalog filtered by: slot type, credit cost ≤ remaining, operator availability | Full activity list |

The Suggestion Resolver filters `available_activities` to only those that are feasible (credit-within-budget, slot-compatible, operator-available) before passing to AI. **AI never sees infeasible activities.** This keeps the AI prompt small and ensures AI cannot recommend something the guest can't actually do.

---

### 7.3 Output Contract

The AI Ranker returns a structured JSON response. No free-form prose in the ranked list — every field is typed and bounded.

```json
{
  "ranked": [
    {
      "activity_id": "praia-da-joaquina-surf",
      "score": 0.91,
      "reason": "Great beach day — fits your afternoon slot and you haven't tried surfing yet",
      "tags_matched": ["outdoor", "active", "beach"]
    },
    {
      "activity_id": "mercado-publico-tour",
      "score": 0.74,
      "reason": "Popular with guests who enjoy food and culture",
      "tags_matched": ["cultural", "food"]
    },
    {
      "activity_id": "lagoa-kayak",
      "score": 0.61,
      "reason": "Relaxed outdoor option if you prefer calmer water",
      "tags_matched": ["outdoor", "relaxing"]
    }
  ],
  "fallback_reason": null,
  "model_version": "vacation-ranker-v1",
  "latency_ms": 340
}
```

**Contract constraints:**
- `activity_id` must be in the input `available_activities` list (Suggestion Resolver validates)
- `score` is a float in `[0.0, 1.0]`
- `reason` is 1–2 sentences, plain text, no markdown, max 120 characters
- `tags_matched` is a subset of the activity's defined tags
- `fallback_reason` is `null` on success; a short string if AI could not rank (e.g., `"no_activities_available"`, `"weather_data_missing"`)
- Max 10 items in `ranked`
- Response must be returned within 3 seconds; Suggestion Resolver times out and falls back to static ranking at 3s

If `ranked` is empty and `fallback_reason` is set, the Suggestion Resolver uses static ranking and logs the fallback.

---

### 7.4 Weather Fallback

When a `weather-alert` event fires on the Stay Run (e.g., rain is incoming), the AI re-ranking is triggered automatically as part of the slot advisory update.

**Flow:**

1. Weather provider emits `weather-alert` event to the Stay Run
2. Runtime fires the weather handler step (a GERT tool invocation in the Kit's compiled lowering)
3. Suggestion Resolver is invoked with updated context: `weather_current: "rainy"`
4. Suggestion Resolver filters `available_activities` to those tagged `indoor` (drops all `outdoor` activities)
5. AI Ranker receives the filtered list + `current_day_flavor` biased toward `rain_day`
6. AI returns new ranked list weighted toward indoor options
7. Suggestion Resolver writes the new ranked list to the slot task context
8. Runtime emits `slot-changed` SSE event
9. App receives `slot-changed`, refetches `whats_next` and `suggestions` — guest sees updated recommendations

**Important constraint:** If the guest has already confirmed an activity for the current slot and rain arrives, the runtime does NOT automatically undo the confirmation. It emits a `weather-alert` with the activity's outdoor status flagged, and surfaces an "Activity may be affected" notice. The guest decides whether to switch. This preserves guest agency.

---

### 7.5 Experience Diversification

The AI Ranker receives `activities_done` and `categories_done` as inputs. It uses this to bias away from repetition.

**Rules (encoded in the AI prompt, not in GERT runtime):**

1. **No exact repeats** — an activity in `activities_done` receives a score penalty of -0.3 (may still appear if the list is short, but deprioritized)
2. **Category diversification** — categories in `categories_done` receive a score penalty of -0.15 per occurrence
3. **Novelty bonus** — activities in categories not yet explored receive a +0.10 score bonus
4. **Operator promotion override** — operator-promoted activities receive a +0.20 bonus that partially overrides diversification penalties (the operator wants certain activities featured regardless)

**No persistent guest profile required for MVP.** All history is stay-scoped: `activities_done` is derived entirely from the current Stay Run's events. The AI has no cross-stay memory. A guest who returns for a second stay starts fresh — the operator can optionally annotate the new run with past-stay tags if they choose.

---

### 7.6 Integration Point

#### AI as GERT Tool Invocation

The AI Ranker is invoked by the Suggestion Resolver as a GERT tool call. In the Kit's compiled lowering, each slot advisory step includes a `tool` invocation step:

```yaml
# Lowered core YAML (compiler output, not author-facing)
- type: tool
  name: vacation.ai.rank_activities
  input:
    current_day_flavor: "{{run.state.current_day_flavor}}"
    slot_type: "{{run.state.current_slot}}"
    weather_current: "{{run.state.weather_current}}"
    activities_done: "{{run.events.filter('activity_selected').pluck('activity_id')}}"
    credits_remaining: "{{run.state.credits_remaining}}"
    available_activities: "{{run.state.slot_available_activities}}"
  timeout: 3s
  on_timeout: fallback_static_ranking
```

The `vacation.ai.rank_activities` tool is registered by the Vacation Kit's companion extension. It wraps the AI provider call (OpenAI, Anthropic, or a local model) in the standard GERT tool interface.

#### Non-Blocking Design

If the AI tool invocation fails or times out:
- The `on_timeout: fallback_static_ranking` directive activates
- The Suggestion Resolver applies static ranking (operator priority order + basic diversity rule)
- Run continues normally; the guest sees a working list
- The timeout/failure is recorded in the trace for operator review

AI unavailability never blocks run progression. The slot advisory is a hint to the guest, not a gate on the run.

#### MVP Simplification

For MVP with a single local operator: the AI Ranker can be an external HTTP call rather than a GERT tool registration. The Suggestion Resolver calls `POST https://ai.operator-host/rank` and applies the same timeout/fallback behavior. The tool registration model is the v1 production design; the HTTP call is acceptable for an MVP with one operator.

---

## Section 9: Deliverables Summary

---

### 9.1 Architecture Summary

```
┌──────────────────────────────────────────────────────────────────────┐
│                     Vacation Domain Kit                              │
│                                                                      │
│  Author writes vacation YAML (stay templates, activity catalogs)    │
│                         │                                            │
│                    Kit Compiler                                      │
│            (lowers to core runbook/v2 YAML)                         │
│                         │                                            │
│                    GERT Core Runtime                                 │
│    ┌────────────┬────────────────┬──────────────┬──────────────┐    │
│    │  Run state │  Human tasks   │    Events    │   Evidence   │    │
│    │ (stay, day,│ (slot advisory,│ (weather,    │ (documents,  │    │
│    │  credits)  │  doc requests) │  messages,   │  day seals,  │    │
│    │            │                │  alerts)     │  receipts)   │    │
│    └────────────┴────────────────┴──────────────┴──────────────┘    │
│                         │                                            │
│                    gert serve                                        │
│              (JSON-RPC + SSE stream)                                │
│                         │                                            │
│          ┌──────────────┴──────────────────┐                        │
│          │                                 │                        │
│    Mobile App                        AI Ranker                      │
│  (guest-facing,                  (suggestion engine,                │
│   read-mostly)                    non-blocking)                     │
└──────────────────────────────────────────────────────────────────────┘
```

**Kit layers:**
- **DSL layer:** Vacation-specific YAML vocabulary (stay templates, day flavors, activity catalogs, credit rules)
- **Compiler layer:** Lowers vacation YAML to `runbook/v2` core primitives (Go package in `gert-kit-vacation/compiler/`)
- **Runtime layer:** Pure GERT core — run state machine, events, human tasks, evidence, timers
- **Integration layer:** `gert serve` (JSON-RPC + SSE) ← existing v2 infrastructure

**What's new vs reused:**

| Component                    | Status          |
|------------------------------|-----------------|
| gert serve (JSON-RPC, SSE)   | ✅ Reused (v2)   |
| Bearer auth (JWT validation) | ✅ Reused (v2)   |
| Run state, events, evidence  | ✅ Reused (v2)   |
| Human tasks, timers          | ✅ Reused (v2)   |
| Kit compiler (Go package)    | 🆕 New           |
| QR token provisioning        | 🆕 New (small)   |
| Credit ledger (event-sourced)| 🆕 New (small)   |
| AI Ranker integration        | 🆕 New (external)|
| Vacation YAML schemas        | 🆕 New           |
| Mobile web app               | 🆕 New (separate)|

---

### 9.2 Package Structure

```
gert-kit-vacation/
  gert-kit.yaml              # Kit manifest (name, version, gert-core compat range)
  
  schemas/
    stay-v1.json             # JSON Schema for stay template YAML
    activity-v1.json         # JSON Schema for activity catalog entries
    credit-rules-v1.json     # JSON Schema for credit policy rules
  
  templates/
    florianopolis-7day/
      stay.yaml              # Example 7-day stay template
      activities.yaml        # Activity catalog for this operator
      credit-rules.yaml      # Credit policy
    minimal/
      stay.yaml              # Minimal 1-day example for testing/docs
  
  compiler/
    compiler.go              # Kit compiler entry point (YAML in → core YAML out)
    lowering/
      stay.go                # Stay Run structure lowering
      day.go                 # Day + slot lowering
      activities.go          # Activity catalog resolution
      credits.go             # Credit ledger setup
      weather.go             # Weather handler lowering
    validate/
      stay.go                # Domain-specific validation rules
    testdata/
      golden/                # Golden lowering outputs for compiler tests
  
  provisioning/
    token.go                 # QR code token minting + validation (small Go package)
  
  docs/
    operator-guide.md        # How to author a stay template
    activity-catalog.md      # How to define activities and credit costs
    mobile-contract.md       # This document (§6) — shared with app developers
    ai-integration.md        # This document (§7) — for AI integration work
```

---

### 9.3 Implementation Notes

#### Pure YAML + Lowering Rules (no new Go code beyond the compiler)

The following can be expressed entirely as YAML authoring + lowering rules with no custom Go logic:

- **Day structure and slot sequencing** — days lower to a `sequential` branch of steps; slots lower to human tasks + tool invocations
- **Day flavor labeling** — a metadata annotation on the compiled run; no runtime behavior
- **Meal scheduling** — timer events at fixed times (breakfast 8am, dinner 7:30pm); lower to `timer` steps in core
- **Activity catalog lookup** — static data referenced by the compiler at lowering time; no runtime lookup needed
- **Operator messages** — an `operator-message` event type registered by the Kit; the operator emits it via the admin API

#### Requires a Small Go Package

- **QR token provisioning** (`provisioning/token.go`): ~200 lines. Mints signed provisioning tokens, issues JWTs, handles one-time-use invalidation. Wraps standard `golang-jwt/jwt`. Runs as a thin HTTP handler alongside `gert serve`.
- **Credit ledger** (`compiler/lowering/credits.go`): ~150 lines. Defines the credit-consume event handler and the ledger projection query. The ledger is event-sourced (append-only debit records on the run); the projection is a query at read time. No mutable balance field.
- **Suggestion Resolver** (`compiler/lowering/activities.go`): ~300 lines. Filters available activities, calls AI Ranker (with timeout), applies fallback, writes ranked list to task context.

#### Reuses Existing gert serve Infrastructure

- **SSE stream** (`GET /sse/run/<id>`) — used as-is; no changes needed
- **Bearer auth** — guest JWT validated by existing middleware; only scope check is new (add `run:read` scope validation to the middleware)
- **`run.get` RPC** — used for all read surfaces; the data shapes in §6.2 are query projections, not new RPC methods
- **`run.task.resolve` RPC** — used for guest confirmations (§6.3.1, §6.3.2)
- **`run.evidence.attach` RPC** — used for document uploads (§6.3.3)
- **`run.event.emit` RPC** — used for alert acknowledgements and assistance requests (§6.3.4, §6.3.5)

No new gert serve RPC methods are needed for MVP. The write contract maps entirely to existing primitives.

---

### 9.4 Risks

#### R1: Credit Ledger Concurrency (Medium)

The credit ledger uses event-sourced debit records. If two concurrent requests try to consume the last credit simultaneously (e.g., guest taps "Accept Upgrade" twice quickly), both events might be accepted before the ledger projection shows the balance as zero.

**Mitigation:** The `credit-consume` event handler reads the current balance from the projected ledger before emitting. Use a per-run advisory lock (or optimistic concurrency check on event sequence number) in the Suggestion Resolver. For MVP with a single guest per run, this is low-risk; it's a real concern if multi-device is supported.

#### R2: AI Latency / Cost (Low-Medium)

The AI Ranker is on the hot path for slot advisory updates. If it's slow (>2s), the guest sees a loading spinner. If it's expensive, operator costs scale with stay volume.

**Mitigation:** 3-second timeout with static fallback (§7.6). For MVP, cache the ranked list per slot+weather combination for 15 minutes — most guests won't change their mind within 15 minutes, and weather doesn't change that fast.

#### R3: SSE Connection Stability on Mobile (Medium)

Mobile networks drop SSE connections frequently. The reconnect logic (§6.4) handles this, but if `gert serve` doesn't retain event history, the app gets stale state after reconnect.

**Mitigation:** `gert serve` should retain the last 5 minutes of events per run in an in-memory ring buffer. This is a small addition to the existing SSE implementation. Without it, every reconnect requires a full surface refresh (slower, more data, worse UX).

#### R4: Token Refresh While Offline (Low)

If the access JWT expires while the guest is offline, they return to a locked app with no refresh path.

**Mitigation:** Token lifetime is aligned to stay duration + 4-hour buffer. For short stays (2–3 days), the token is valid for the entire stay. For longer stays, the app proactively refreshes at 1-hour-before-expiry (while online). The risk is low as long as the guest has connectivity at least once per `(token_lifetime - 1hr)`.

#### R5: Operator Authoring Complexity (High Risk for Adoption)

If authoring a stay template requires YAML expertise, most local operators won't use the Kit. A 7-day stay with 3 slots per day and 20 activities per slot is a non-trivial YAML document.

**Mitigation:** Provide 2-3 complete template examples (including the Florianópolis 7-day example in `templates/`). An operator admin UI (§9.5) is the real long-term mitigation. For MVP, one technically-capable person per operator (or the gert team) handles authoring.

#### R6: Under-Specified Operator Admin Path (Medium)

This design covers the guest-facing mobile contract but doesn't specify how the operator monitors stays, responds to assistance requests, or emits messages. The operator needs a surface too.

**Assumption to validate:** For MVP, assume the operator uses `gert serve` directly (with a different, admin-scoped JWT) and a simple terminal or basic web view. A dedicated admin UI is deferred to v1.

---

### 9.5 Next Steps

#### What a v1 (Production-Ready Kit) Would Add

1. **Operator admin UI** — A minimal web app for the operator. Views: active stays, guest help requests, message compose, credit adjustments, activity availability toggling. Reads from `gert serve` with an admin-scoped JWT. This is the single highest-value addition after MVP.

2. **Multi-device support** — Allow a guest to scan the QR code on a second device (e.g., partner). Requires the refresh token model to support multiple concurrent sessions per Stay Run. Low complexity, high usability value.

3. **Reservation integrations** — Connect activity bookings to real reservation systems (e.g., Google Calendar, local booking APIs). Currently activities are informational; a v1 Kit could emit a `reservation-confirmed` event that triggers an actual booking via a GERT tool.

4. **Stay analytics / projections** — Post-stay summaries for the operator: which activities were most popular, average credit consumption, common assistance request categories. Implemented as Kit projections over the sealed run trace.

5. **Hardened AI integration** — Move from HTTP call to registered GERT tool (§7.6 production model). Add AI response caching. Consider a local model option for operators with privacy concerns.

6. **Schema versioning** — The Kit's `stay-v1.json` schema is v1. A v1 Kit needs a documented migration path when the schema evolves. Add a `gert-kit migrate` command that rewrites old stay templates to the new schema.

#### Operator Tooling Needed (Admin UI Minimum)

For the Kit to be adopted by real operators without technical staff:

- **Stay builder UI:** Drag-and-drop day/slot/activity configuration. Outputs a valid `stay.yaml`. This is the unlock for non-technical operators.
- **Live dashboard:** Active stays, current guest locations in the run, pending help requests.
- **Message composer:** Send a message to one guest or all active guests. Supports urgency levels.

These are separate from the Kit itself (they're a web app that calls `gert serve`), but they're prerequisites for real operator adoption.

#### How This Informs the Second Domain Kit

The Vacation Kit validates two things the Domain Kit model needs:

1. **Human task as advisory pattern** — the slot advisory task (AI suggests, guest confirms) is a reusable pattern. Any Kit with a "suggest + confirm" interaction should follow this shape. The second Kit should reuse it.

2. **Event-sourced ledger pattern** — the credit ledger (append-only debit events, projection at read time) is a general pattern for any Kit with budget/quota tracking. Incident response kits (time-to-resolve budgets), compliance kits (audit trail credits), and operations kits (change budget) could all reuse this pattern. It should be promoted to a Kit standard library pattern, not re-invented per Kit.

3. **Mobile contract is reusable** — if a second Kit also needs a mobile-facing read surface (e.g., a field operations Kit for technicians), the §6 patterns (bearer token, SSE, deferred writes) apply directly. The second Kit should extend the contract, not replace it.

The main design risk to watch: **Kit-specific vocabulary must not leak into gert serve.** The serve layer should remain domain-agnostic. All domain-specific read surfaces (§6.2) should be expressible as `run.get` queries with a projection parameter, not as new Kit-specific RPC methods.

---

*End of Sections 6, 7, and 9 — Barbara (Integrations Specialist)*

---

## §4.10 — Traceability: Reverse Mapping

**Author:** Ken (Software Architect)  
**Date:** 2026-04-22  
**Purpose:** Define the canonical GERT Domain Kit contract for reverse mapping lowered core steps back to kit-level abstractions.

---

### 4.10.1 The Problem Statement

When a Domain Kit compiler lowers kit-specific abstractions into GERT core YAML (`runbook/v2` format), the resulting runbook contains only core primitives: `branch`, `choice`, `cli`, `include`, `tool`, `manual`, etc. The kit-level concepts are gone at the schema level.

For the Vacation Kit, this means:
- A `DayFlavor: beach-day` with an `afternoon-activity` slot lowers to a `branch` step with multiple child steps.
- A `MealSlot: dinner` lowers to a timer-bounded `manual` step.
- A `StayTemplate` credit allocation lowers to runtime state initialization steps.

After lowering, the kit vocabulary has no schema-level representation. The lowered runbook is pure GERT — executable without the Vacation Kit installed, version-controllable, testable against core GERT invariants.

**This is intentional and correct** — it is the Clean Kernel Principle (§4.3) in action.

**The problem:** A developer (or operator) inspecting a trace file or a lowered runbook has no way to know:
- Which kit generated this step
- Which kit concept (DayFlavor, MealSlot, ActivityPool, etc.) this step came from
- Which source field in the kit YAML maps to this step

This is the **compiler source map problem** — exactly the same problem TypeScript→JavaScript, SASS→CSS, Terraform→CloudFormation, etc. solve with source maps.

#### Why Reverse Mapping Is Essential

Without reverse mapping, Domain Kits become write-only abstractions:

1. **Debugging** — When a trace event shows `step/failed: vacation.day.2.afternoon-activity.weather-branch`, the operator asks: "Which DayFlavor was that? What was the source condition that failed?" Without a reverse map, the operator must manually trace the step ID back through the lowered runbook, then back through the kit source, losing 10 minutes per incident.

2. **Audit / Governance** — A compliance requirement: "Show me which policy gate corresponds to which Stay policy in the template." The audit log contains core step IDs; the compliance officer needs kit-level labels. Without reverse mapping, the evidence trail is technically complete but semantically opaque.

3. **Projection** — A domain-level dashboard ("Show me the guest's completed activities today") must correlate core trace events back to kit concepts. Without a reverse map, projection code must parse step IDs heuristically or embed kit knowledge in every projection — both brittle and non-portable.

4. **Replay Analysis** — `gert replay` shows step-by-step execution. An operator wants to replay "Day 2" — not "steps 47–103." Without a reverse map, the operator must manually identify which steps belong to which day.

5. **Multi-Kit Composition** — If a runbook includes steps from two kits (e.g., Vacation Kit + Compliance Kit), trace events must carry provenance to disambiguate which kit owns each step.

**Design principle:** Reverse mapping is not optional for production-grade Domain Kits. It is a mandatory contract that makes kit-generated runbooks inspectable, debuggable, and auditable.

---

### 4.10.2 Three-Layer Solution

GERT's reverse mapping strategy is a layered approach: start with zero-cost conventions (Layer 1), add compiler-emitted artifacts (Layer 2), extend core schema when inline annotation is required (Layer 3). Each layer builds on the previous; kits choose the layer that matches their traceability needs.

#### Layer 1 — Step ID Naming Convention (Zero Cost, Works Today)

**Contract:** Every Domain Kit compiler MUST produce deterministic, structured step IDs encoding the full provenance path.

**Format:**

```
{kit-prefix}.{concept-kind}.{concept-name}.{sub-element}
```

**Rules:**

1. All segments are `kebab-case`.
2. `{kit-prefix}` is the kit's reverse-DNS ID shortened to a recognizable prefix. Example: `vacation.gert.io/v0` → `vacation`.
3. `{concept-kind}` matches the kit vocabulary noun (e.g., `day`, `slot`, `activity`, `credits`, `meal`, `pass`).
4. `{concept-name}` is the instance name from the kit source (e.g., `beach-day`, `floripa-5day`, `spa-credit`).
5. `{sub-element}` is an optional role suffix for sub-steps generated for the same concept (e.g., `.branch`, `.check`, `.reminder`, `.fallback`).
6. IDs MUST be unique within the runbook.
7. IDs MUST be stable across re-compilations for the same kit source (deterministic, not random).

**Vacation Kit Examples:**

| Kit Source Concept | Generated Step ID |
|---|---|
| DayFlavor: `beach-day`, slot: `afternoon-activity` | `vacation.day.2.afternoon-activity` |
| └─ Weather branch for rain fallback | `vacation.day.2.afternoon-activity.weather-branch` |
| └─ Rain fallback activity selection | `vacation.day.2.afternoon-activity.rain-fallback` |
| StayTemplate: `floripa-5day`, credit check: `spa` | `vacation.credits.spa.check` |
| MealSlot: `dinner`, day 3 | `vacation.slot.dinner.3` |
| └─ Dietary reminder for guest | `vacation.slot.dinner.3.reminder` |
| QR pass issuance at check-in | `vacation.pass.checkin.issue` |

**Why this works:**

Projection tools can parse IDs by splitting on `.` to recover kit provenance:

```python
# Projection code reads trace.jsonl
for event in trace:
    step_id = event["step"]
    segments = step_id.split(".")
    
    if segments[0] == "vacation":
        kit = "vacation.gert.io/v0"
        concept_kind = segments[1]  # "day", "slot", "credits"
        concept_name = segments[2]  # "2", "dinner", "spa"
        sub_element = segments[3] if len(segments) > 3 else None
        
        # Now project to domain-level view
        if concept_kind == "day":
            day_number = int(concept_name)
            add_to_day_timeline(day_number, event)
```

**No schema changes required.** This works TODAY with core GERT v2.0. It is a **convention**, not a runtime feature.

**Kit Contract Rule 1:** All Domain Kits MUST follow this naming convention. Non-compliance means the kit's trace events are not projectable.

---

#### Layer 2 — Compiler-Emitted Source Map (Sidecar File)

**Contract:** The kit compiler MUST emit a `{runbook-name}.sourcemap.yaml` file alongside the lowered runbook. This file is the canonical reverse map — it is NOT part of core GERT schema; it is a kit artifact.

**Purpose:** Layer 1 (step IDs) provides coarse-grained provenance. Layer 2 (source map) adds:
- Source file and line number (where the kit concept was defined)
- Structured metadata (kind, name, slot, day, role)
- Kit version and lowering timestamp (for compatibility checks across kit versions)

**Format:**

```yaml
# stay-floripa.sourcemap.yaml
version: "1"
kit: vacation.gert.io/v0
runbook: stay-floripa
lowered_at: "2026-04-22T14:33:00Z"
entries:
  vacation.day.2.afternoon-activity:
    kind: DayFlavor
    name: beach-day
    slot: afternoon-activity
    day: 2
    source_file: templates/beach-day.yaml
    source_line: 42
    parent_concept: floripa-5day
    
  vacation.day.2.afternoon-activity.weather-branch:
    kind: DayFlavor
    name: beach-day
    slot: afternoon-activity
    day: 2
    role: weather-branch
    source_file: templates/beach-day.yaml
    source_line: 47
    parent_step: vacation.day.2.afternoon-activity
    
  vacation.credits.spa.check:
    kind: StayTemplate
    name: floripa-5day
    field: credits.spa
    source_file: templates/floripa-5day.yaml
    source_line: 17
    
  vacation.slot.dinner.3:
    kind: MealSlot
    name: dinner
    day: 3
    time_window: "18:00-20:00"
    source_file: templates/floripa-5day.yaml
    source_line: 89
```

**Schema:**

- `version`: Source map format version (currently `"1"`)
- `kit`: Fully-qualified kit ID with version
- `runbook`: Name of the lowered runbook this map corresponds to
- `lowered_at`: ISO8601 timestamp of lowering (for cache invalidation)
- `entries`: Map of `{step-id → metadata}`
  - `kind`: Kit concept type (DayFlavor, MealSlot, StayTemplate, Activity, etc.)
  - `name`: Instance name from kit source
  - `source_file`: Relative path to kit source YAML
  - `source_line`: Line number where the concept is defined (optional but recommended)
  - Additional fields as needed per concept type (day, slot, role, parent_step, etc.)

**Consumers:**

1. **Kit projection layer** — Reads `trace.jsonl` + `sourcemap.yaml` to produce domain-level views. Example:
   ```bash
   gert-kit vacation project --run-id stay-123 --view activities
   ```
   Projection reads trace, looks up each step ID in the source map, filters for `kind: Activity`, groups by day.

2. **`gert replay`** — When a source map is present, `gert replay` can group steps by kit concept:
   ```bash
   gert replay stay-123 --group-by kind
   ```
   Output shows "Day 2 (DayFlavor: beach-day)" as a collapsible group, not 23 individual steps.

3. **Operator dashboard** — Enriches step events with kit-level labels. A trace event `{"step": "vacation.day.2.afternoon-activity.weather-branch", "status": "failed"}` becomes "Beach Day: afternoon activity — weather fallback failed."

4. **Kit version compatibility check** — When replaying an old trace against a new kit version, compare the trace's `kit` field to the source map's `kit` field. Warn if versions mismatch.

**The source map is a plain YAML file** — no runtime dependency. Core GERT never reads it. It's an artifact like a `.map` file in JavaScript: optional but invaluable for production debugging.

**Kit Contract Rule 2:** All Domain Kits MUST emit a `{runbook-name}.sourcemap.yaml` sidecar for every lowered runbook.

**Kit Contract Rule 3:** The source map MUST be deterministic: same kit source → same source map (across re-compilations). No timestamps in entry keys; no random ordering.

**Kit Contract Rule 4:** The source map MUST version the kit: `version` and `kit` fields required. This enables compatibility checks and projection code evolution.

---

#### Layer 3 — Schema Extension (Proposed, Requires Core Change)

**Status:** This layer is NOT available in GERT v2.0. It is a concrete proposal for v2.1 or later.

**Problem Layer 2 Solves / Doesn't Solve:**

Layer 2 (source map) works perfectly when:
- You have the lowered runbook + source map + trace file in the same directory
- You're analyzing a run offline (post-mortem, batch projection)
- You're building a dashboard that can load the source map once per runbook

Layer 2 breaks down when:
- Trace events are streamed live to an external system (OpenTelemetry collector, log aggregator)
- Multiple kits are composed, each with its own source map (requires multi-map lookup)
- You want trace events to be self-describing (no external files required)

**Proposal:** Add a `Meta` field to the core `Step` struct. This field carries optional kit-level annotations stamped by the lowering compiler. The core runtime propagates this map unchanged into trace events.

**Implementation:**

Add to `pkg/schema/step.go`:

```go
// Step represents a single unit of work in a runbook.
type Step struct {
    Name        string            `yaml:"name" json:"name"`
    Type        string            `yaml:"type" json:"type"`
    Description string            `yaml:"description,omitempty" json:"description,omitempty"`
    // ... existing fields ...
    
    // Meta carries optional kit-level annotations stamped by the lowering compiler.
    // The core runtime propagates this map unchanged into trace events.
    // Keys prefixed with "x-" are reserved for extensions.
    Meta        map[string]string `yaml:"meta,omitempty" json:"meta,omitempty"`
}
```

**Kit Compiler Usage:**

When lowering, the Vacation Kit compiler stamps `meta` fields on every generated step:

```yaml
# Lowered runbook: stay-floripa.yaml
name: stay-floripa
version: runbook/v2
steps:
  - name: vacation.day.2.afternoon-activity
    type: branch
    description: "Beach day afternoon activity selection"
    meta:
      kit: vacation.gert.io/v0
      source_kind: DayFlavor
      source_name: beach-day
      source_slot: afternoon-activity
      source_day: "2"
      source_file: templates/beach-day.yaml
      source_line: "42"
    branches:
      - condition: weather == "sunny"
        steps:
          - name: vacation.day.2.afternoon-activity.kayak
            type: manual
            meta:
              kit: vacation.gert.io/v0
              source_kind: Activity
              source_name: kayak
              source_day: "2"
              source_slot: afternoon-activity
            # ... step definition ...
```

**Runtime Behavior:**

The trace writer emits `meta` fields in `step/started` and `step/completed` events:

```jsonl
{"event":"step/started","step":"vacation.day.2.afternoon-activity","meta":{"kit":"vacation.gert.io/v0","source_kind":"DayFlavor","source_name":"beach-day","source_slot":"afternoon-activity","source_day":"2","source_file":"templates/beach-day.yaml","source_line":"42"},"ts":"2026-04-22T14:35:12Z"}
```

Now every trace event carries its kit provenance **inline** — no external source map lookup needed.

**Advantages:**

1. **Self-describing traces** — Trace events exported to OpenTelemetry, log aggregators, or remote dashboards carry full provenance. No sidecar files required.
2. **Multi-kit composition** — If a runbook includes steps from two kits, `meta.kit` disambiguates ownership.
3. **Streaming projections** — A projection service consuming live trace events (via SSE or OpenTelemetry) can produce domain-level views without loading source maps.
4. **Backward compatible** — If `meta` is absent, core GERT behavior is unchanged. Layer 1 (step ID) and Layer 2 (source map) still work.

**Disadvantages:**

1. **Increased trace size** — Every step event now carries 6–10 extra fields. For a 200-step runbook, this adds ~5–10 KB per trace file (negligible for most use cases, but measurable for high-volume systems).
2. **Core schema change** — Requires a GERT v2.1 release. Not available today.

**Impact Assessment:**

Adding `Meta map[string]string` to the `Step` struct is a **small, low-risk change**:
- Parser: Already handles unknown YAML fields (currently drops them silently). With `meta,omitempty`, old parsers ignore it; new parsers load it.
- Runtime: No logic changes. The runtime copies `meta` from the step definition to the trace event envelope.
- Trace writer: One-line addition to emit `meta` fields in step events.
- Backward compat: Old GERT runtimes can execute runbooks with `meta` fields (they ignore unknown fields). New GERT runtimes can execute runbooks without `meta` (field is optional).

**Decision:** This proposal should be filed as a GERT v2.1 candidate feature. For v2.0, kits must rely on Layer 1 + Layer 2.

**Kit Contract Rule 5:** If Layer 3 (Meta) is available in the target GERT version, the compiler SHOULD populate `meta` fields on every generated step. Use key prefix `x-` for kit-specific extensions not part of the core schema.

---

### 4.10.3 Which Layer to Use When

The three layers are **complementary, not exclusive**. A kit can (and should) implement multiple layers.

| Scenario | Recommended Layers | Rationale |
|---|---|---|
| **Quick debugging** | Layer 1 (step ID naming) | Parse step IDs manually or with simple regex. No external files needed. Works TODAY. |
| **Operator dashboard** | Layer 1 + Layer 2 (source map) | Dashboard loads source map once per runbook at startup. Enriches live trace events with kit labels. Production-ready. |
| **Post-mortem analysis** | Layer 2 (source map) | Load trace + source map, produce detailed audit report with source file references. Standard forensic workflow. |
| **Streaming projection** | Layer 3 (meta field) — FUTURE | Trace events are self-describing; no source map lookup required. For high-volume, multi-kit systems. |
| **Multi-kit composition** | Layer 3 (meta field) — FUTURE | `meta.kit` disambiguates which kit owns each step when multiple kits contribute to the same runbook. |
| **Kit compatibility check** | Layer 2 (source map versioning) | Compare trace's `kit` version to current kit version. Warn if projection code is outdated. |
| **Offline replay analysis** | Layer 1 + Layer 2 | `gert replay` groups steps by kit concept (Day, Slot, Activity). Operator replays "Day 2" instead of "steps 47–103." |

**Decision Matrix for Kit Authors:**

- **Layer 1 is MANDATORY** — All kits must follow the step ID naming convention. No exceptions.
- **Layer 2 is RECOMMENDED** — Production kits should emit source maps. This is the standard best practice.
- **Layer 3 is OPTIONAL** — Only needed for advanced scenarios (streaming projections, multi-kit composition). Not available in v2.0.

**Implementation Effort:**

| Layer | Kit Compiler Work | Example LOC |
|---|---|---|
| Layer 1 | Implement deterministic step ID generator | ~50 lines |
| Layer 2 | Emit `sourcemap.yaml` during lowering | ~150 lines |
| Layer 3 | Populate `meta` fields on each step | ~80 lines (reuses Layer 2 data) |

Layer 1 + Layer 2 are achievable in a 2-week MVP kit. Layer 3 requires core GERT v2.1.

---

### 4.10.4 Formal Kit Traceability Contract

All GERT Domain Kits MUST satisfy the following traceability contract. Non-compliant kits are not production-ready.

**Kit Traceability Contract (5 Rules):**

1. **Step IDs MUST follow the structured naming convention:**  
   `{kit-prefix}.{concept-kind}.{concept-name}.{sub-element}`  
   All segments kebab-case. IDs must be unique and deterministic.

2. **The compiler MUST emit a `{runbook-name}.sourcemap.yaml` sidecar for every lowered runbook.**  
   The source map must include: `version`, `kit`, `runbook`, `lowered_at`, and `entries` map.

3. **The source map MUST be deterministic:**  
   Same kit source → same source map (across re-compilations). No random ordering, no timestamps in entry keys.

4. **The source map MUST version the kit:**  
   `version` and `kit` fields are required. This enables projection code to detect kit version mismatches.

5. **If Layer 3 (Meta) is available in the target GERT version, the compiler SHOULD populate `meta` fields on every generated step.**  
   Use the `x-` key prefix for kit-specific metadata not defined in the core schema.

**Verification:**

Kit maintainers can verify compliance with a test:

```bash
# Compile the kit
gert-kit vacation compile templates/floripa-5day.yaml --output build/

# Check Layer 1 compliance (all step IDs follow convention)
gert-kit vacation lint build/stay-floripa.yaml --check step-ids

# Check Layer 2 compliance (source map exists and is valid)
test -f build/stay-floripa.sourcemap.yaml
gert-kit vacation lint build/stay-floripa.sourcemap.yaml --check schema

# Check determinism (compile twice, compare outputs)
gert-kit vacation compile templates/floripa-5day.yaml --output build-v1/
gert-kit vacation compile templates/floripa-5day.yaml --output build-v2/
diff -u build-v1/stay-floripa.yaml build-v2/stay-floripa.yaml  # must be identical
diff -u build-v1/stay-floripa.sourcemap.yaml build-v2/stay-floripa.sourcemap.yaml  # must be identical
```

**Penalty for non-compliance:**

If a kit does not follow this contract:
- Trace events are not projectable → operators cannot build domain-level dashboards
- Debugging is manual → incidents take 10x longer to resolve
- Audit trails are opaque → compliance officers cannot map trace events to domain concepts
- Multi-kit composition breaks → provenance ambiguity when multiple kits are used

Traceability is not optional. It is the price of admission for a production-grade Domain Kit.

---

### 4.10.5 Vacation Kit Concrete Example

Let's trace a concrete example end-to-end: from kit source to lowered runbook to source map to trace event.

#### Step 1: Kit Source (Vacation YAML)

Author writes a `DayFlavor` for "Beach Day" with an afternoon activity slot:

```yaml
# templates/beach-day.yaml
apiVersion: vacation.gert.io/v0
kind: DayFlavor
name: beach-day
description: "A sunny beach day with water activities"
slots:
  afternoon-activity:
    time_window: "14:00-17:00"
    activity_pool:
      - kayak
      - snorkel
      - beach-volleyball
    fallback_on_rain:
      - spa-massage
      - indoor-pool
    allow_guest_choice: true
```

This is referenced from a Stay Template:

```yaml
# templates/floripa-5day.yaml
apiVersion: vacation.gert.io/v0
kind: StayTemplate
name: floripa-5day
days: 5
day_flavors:
  - day: 1
    flavor: arrival-day
  - day: 2
    flavor: beach-day  # ← references beach-day.yaml
  - day: 3
    flavor: cultural-day
  # ...
```

#### Step 2: Lowered GERT YAML (Core Primitives)

The Vacation Kit compiler lowers this to core GERT:

```yaml
# build/stay-floripa.yaml (excerpt for Day 2, afternoon slot)
name: stay-floripa
version: runbook/v2
steps:
  # ... (day 1 steps omitted) ...
  
  # Day 2: Beach Day — Afternoon Activity
  - name: vacation.day.2.afternoon-activity
    type: branch
    description: "Beach day afternoon activity selection with weather fallback"
    meta:  # Layer 3 (if available in GERT v2.1)
      kit: vacation.gert.io/v0
      source_kind: DayFlavor
      source_name: beach-day
      source_slot: afternoon-activity
      source_day: "2"
      source_file: templates/beach-day.yaml
      source_line: "8"
    branches:
      - name: sunny-weather
        condition: run.vars.weather == "sunny"
        steps:
          - name: vacation.day.2.afternoon-activity.choice
            type: choice
            description: "Guest selects afternoon activity"
            meta:
              kit: vacation.gert.io/v0
              source_kind: DayFlavor
              source_name: beach-day
              source_slot: afternoon-activity
              source_day: "2"
              role: activity-choice
            choices:
              - label: "Kayak (2 hours, beginner-friendly)"
                value: kayak
              - label: "Snorkel (90 min, gear provided)"
                value: snorkel
              - label: "Beach Volleyball (open join)"
                value: beach-volleyball
            result_var: day.2.afternoon_activity
            
      - name: rainy-weather
        condition: run.vars.weather == "rainy"
        steps:
          - name: vacation.day.2.afternoon-activity.rain-fallback
            type: choice
            description: "Rain fallback: indoor activities"
            meta:
              kit: vacation.gert.io/v0
              source_kind: DayFlavor
              source_name: beach-day
              source_slot: afternoon-activity
              source_day: "2"
              role: rain-fallback
            choices:
              - label: "Spa Massage (60 min, costs 20 credits)"
                value: spa-massage
              - label: "Indoor Pool (free access)"
                value: indoor-pool
            result_var: day.2.afternoon_activity
```

Notice:
- **Layer 1 compliance:** Step IDs follow `vacation.day.2.afternoon-activity.{role}` convention.
- **Layer 3 usage:** Every step has a `meta` field with kit provenance (if GERT v2.1+ is available).
- **Core primitives only:** The lowered output uses `branch`, `choice` — no vacation-specific types.

#### Step 3: Source Map (Layer 2)

The compiler emits a source map:

```yaml
# build/stay-floripa.sourcemap.yaml (excerpt)
version: "1"
kit: vacation.gert.io/v0
runbook: stay-floripa
lowered_at: "2026-04-22T15:10:00Z"
entries:
  vacation.day.2.afternoon-activity:
    kind: DayFlavor
    name: beach-day
    slot: afternoon-activity
    day: 2
    source_file: templates/beach-day.yaml
    source_line: 8
    parent_concept: floripa-5day
    
  vacation.day.2.afternoon-activity.choice:
    kind: DayFlavor
    name: beach-day
    slot: afternoon-activity
    day: 2
    role: activity-choice
    source_file: templates/beach-day.yaml
    source_line: 11
    parent_step: vacation.day.2.afternoon-activity
    
  vacation.day.2.afternoon-activity.rain-fallback:
    kind: DayFlavor
    name: beach-day
    slot: afternoon-activity
    day: 2
    role: rain-fallback
    source_file: templates/beach-day.yaml
    source_line: 15
    parent_step: vacation.day.2.afternoon-activity
```

#### Step 4: Trace Event

At runtime, GERT executes the lowered runbook. When the afternoon activity step starts, it emits:

```jsonl
{"event":"step/started","run_id":"stay-abc123","step":"vacation.day.2.afternoon-activity","type":"branch","ts":"2026-04-22T15:30:00Z","meta":{"kit":"vacation.gert.io/v0","source_kind":"DayFlavor","source_name":"beach-day","source_slot":"afternoon-activity","source_day":"2","source_file":"templates/beach-day.yaml","source_line":"8"}}
```

Weather condition evaluates to `rainy`, so the rain fallback branch is taken:

```jsonl
{"event":"step/started","run_id":"stay-abc123","step":"vacation.day.2.afternoon-activity.rain-fallback","type":"choice","ts":"2026-04-22T15:30:02Z","meta":{"kit":"vacation.gert.io/v0","source_kind":"DayFlavor","source_name":"beach-day","source_slot":"afternoon-activity","source_day":"2","role":"rain-fallback","source_file":"templates/beach-day.yaml","source_line":"15"}}
```

Guest selects "Indoor Pool":

```jsonl
{"event":"step/completed","run_id":"stay-abc123","step":"vacation.day.2.afternoon-activity.rain-fallback","status":"success","result":"indoor-pool","ts":"2026-04-22T15:31:45Z","meta":{"kit":"vacation.gert.io/v0","source_kind":"DayFlavor","source_name":"beach-day","source_slot":"afternoon-activity","source_day":"2","role":"rain-fallback"}}
```

#### Step 5: Projection Reads It Back

The Kit's projection layer reads the trace and builds a domain-level view:

```python
# vacation_projector.py
def build_day_timeline(run_id):
    trace = load_trace(run_id)
    sourcemap = load_sourcemap(run_id)  # Layer 2
    
    timeline = {}
    for event in trace:
        step_id = event["step"]
        
        # Layer 1: Parse step ID
        if not step_id.startswith("vacation."):
            continue  # Not a vacation kit step
        
        # Layer 2: Lookup in source map
        meta = sourcemap["entries"].get(step_id, {})
        kind = meta.get("kind")
        
        # Layer 3: Or read inline meta (if available)
        if "meta" in event:
            kind = event["meta"]["source_kind"]
            slot = event["meta"]["source_slot"]
            day = int(event["meta"]["source_day"])
        
        if kind == "DayFlavor":
            if day not in timeline:
                timeline[day] = {"flavor": meta["name"], "slots": {}}
            
            timeline[day]["slots"][slot] = {
                "status": event.get("status", "in_progress"),
                "result": event.get("result"),
                "timestamp": event["ts"]
            }
    
    return timeline

# Output for Day 2:
# {
#   2: {
#     "flavor": "beach-day",
#     "slots": {
#       "afternoon-activity": {
#         "status": "success",
#         "result": "indoor-pool",
#         "timestamp": "2026-04-22T15:31:45Z"
#       }
#     }
#   }
# }
```

The operator's dashboard displays:

```
Day 2: Beach Day
  Afternoon Activity: Indoor Pool (completed at 15:31)
  Reason: Rain fallback triggered
```

**Without reverse mapping**, the dashboard would show:

```
Step: vacation.day.2.afternoon-activity.rain-fallback
Status: completed
Result: indoor-pool
```

The operator has no idea this was "Day 2, Beach Day, rain fallback." The semantic gap is closed by reverse mapping.

---

### 4.10.6 Multi-Kit Composition

When a runbook includes steps from multiple kits (e.g., Vacation Kit + Compliance Kit), reverse mapping becomes essential for provenance disambiguation.

Example: A hospitality operator runs stays with integrated compliance checks:

```yaml
# templates/compliant-stay.yaml
name: compliant-stay
version: runbook/v2
steps:
  - name: compliance.gdpr.data-retention-check
    type: tool
    meta:
      kit: compliance.gert.io/v1
      source_kind: CompliancePolicy
      source_name: gdpr-data-retention
    # ...
    
  - name: vacation.day.1.checkin
    type: manual
    meta:
      kit: vacation.gert.io/v0
      source_kind: StayTemplate
      source_name: floripa-5day
    # ...
```

Trace events now carry `meta.kit` to identify which kit owns each step:

```jsonl
{"event":"step/started","step":"compliance.gdpr.data-retention-check","meta":{"kit":"compliance.gert.io/v1","source_kind":"CompliancePolicy","source_name":"gdpr-data-retention"},"ts":"..."}
{"event":"step/started","step":"vacation.day.1.checkin","meta":{"kit":"vacation.gert.io/v0","source_kind":"StayTemplate","source_name":"floripa-5day"},"ts":"..."}
```

Projection code can now route events to kit-specific projectors:

```python
for event in trace:
    kit = event["meta"]["kit"]
    
    if kit.startswith("vacation.gert.io"):
        vacation_projector.handle(event)
    elif kit.startswith("compliance.gert.io"):
        compliance_projector.handle(event)
```

**Without Layer 3 (meta field)**, this requires loading two source maps and disambiguating by step ID prefix — brittle and slow. Layer 3 makes multi-kit composition first-class.

---

### 4.10.7 Design Implications for GERT Core

Reverse mapping has minimal impact on core GERT:

1. **Parser:** Already handles unknown YAML fields. The `meta` field (Layer 3) is parsed like any other optional field.
2. **Runtime:** No logic changes. The runtime reads the execution plan and emits trace events. If a step has `meta`, it's copied to the trace event envelope.
3. **Trace writer:** One-line addition to emit `meta` fields in step events.
4. **Schema:** Adding `Meta map[string]string` to `Step` struct is a 1-line change.

**No new core concepts** are required. Reverse mapping is entirely a **compiler concern** (Layers 1 and 2) and an **optional schema extension** (Layer 3).

This is by design: **the Clean Kernel Principle** ensures core GERT never needs to understand kit-level semantics. Reverse mapping is the mechanism that bridges the gap without violating the kernel.

---

### 4.10.8 Testing the Traceability Contract

Every Domain Kit should include traceability tests:

#### Test 1: Step ID Convention Compliance

```bash
# Compile the kit
gert-kit vacation compile templates/floripa-5day.yaml --output build/

# Validate all step IDs follow convention
grep -E '^  - name: ' build/stay-floripa.yaml | while read line; do
    step_id=$(echo "$line" | awk '{print $3}')
    
    # Must start with kit prefix
    if [[ ! "$step_id" =~ ^vacation\. ]]; then
        echo "FAIL: Step ID $step_id does not start with 'vacation.'"
        exit 1
    fi
    
    # Must have at least 3 segments (prefix.kind.name)
    segment_count=$(echo "$step_id" | tr '.' '\n' | wc -l)
    if [ "$segment_count" -lt 3 ]; then
        echo "FAIL: Step ID $step_id has fewer than 3 segments"
        exit 1
    fi
done
echo "PASS: All step IDs follow convention"
```

#### Test 2: Source Map Completeness

```bash
# Every step in the lowered runbook must have a source map entry
lowered_steps=$(grep -E '^  - name: ' build/stay-floripa.yaml | awk '{print $3}')
sourcemap_steps=$(yq '.entries | keys | .[]' build/stay-floripa.sourcemap.yaml)

for step in $lowered_steps; do
    if ! echo "$sourcemap_steps" | grep -q "^$step$"; then
        echo "FAIL: Step $step is in lowered runbook but missing from source map"
        exit 1
    fi
done
echo "PASS: Source map is complete"
```

#### Test 3: Determinism

```bash
# Compile twice, outputs must be identical
gert-kit vacation compile templates/floripa-5day.yaml --output build-v1/
gert-kit vacation compile templates/floripa-5day.yaml --output build-v2/

if ! diff -u build-v1/stay-floripa.yaml build-v2/stay-floripa.yaml; then
    echo "FAIL: Lowered runbook is not deterministic"
    exit 1
fi

if ! diff -u build-v1/stay-floripa.sourcemap.yaml build-v2/stay-floripa.sourcemap.yaml; then
    echo "FAIL: Source map is not deterministic"
    exit 1
fi

echo "PASS: Compilation is deterministic"
```

#### Test 4: Trace Event Meta Propagation (Layer 3, if available)

```bash
# Execute a test run and verify trace events carry meta fields
gert run build/stay-floripa.yaml --inputs guest_id=test-123

# Check that step events have meta field
if ! grep -q '"meta":{' .runbook/runs/*/trace.jsonl; then
    echo "WARN: Trace events do not contain meta field (Layer 3 not available or not enabled)"
else
    echo "PASS: Trace events propagate meta field"
fi
```

These tests ensure the kit is traceable. Include them in the kit's CI pipeline.

---

### 4.10.9 Summary

**Problem:** Domain Kit lowering loses semantic information. Trace events carry only core step IDs — no kit context.

**Solution:** A three-layer reverse mapping strategy:

1. **Layer 1 (Step ID Naming)** — MANDATORY. Zero-cost convention: `{kit}.{kind}.{name}.{role}`. Works today. Parse step IDs to recover provenance.

2. **Layer 2 (Source Map Sidecar)** — RECOMMENDED. Compiler emits `{runbook}.sourcemap.yaml` with full metadata: kind, name, source file, line number. Enables rich projections and operator dashboards.

3. **Layer 3 (Meta Field)** — OPTIONAL (requires GERT v2.1+). Inline `meta` map on every step. Trace events become self-describing. Enables streaming projections and multi-kit composition.

**Kit Traceability Contract (5 Rules):**

1. Step IDs MUST follow structured naming convention
2. Compiler MUST emit source map sidecar
3. Source map MUST be deterministic
4. Source map MUST version the kit
5. Compiler SHOULD populate `meta` fields (if Layer 3 available)

**For the Vacation Kit:**

- **MVP (v0):** Implement Layer 1 + Layer 2. ~200 LOC in compiler.
- **v1 (production-ready):** Add Layer 3 once GERT v2.1 is available.
- **Testing:** Include 4 traceability tests in CI.

Reverse mapping is not optional. It is the foundation of kit observability, debuggability, and operator UX. Without it, Domain Kits are write-only abstractions — unusable in production.

---

*End of §4.10 — Ken (Software Architect)*

---

## Document Index

| Section | Title | Author | Status |
|---------|-------|--------|--------|
| §1 | Domain Kit Definition | Ken | ✅ Complete |
| §2 | Core Domain Concepts | Ken | ✅ Complete |
| §3 | Authoring DSL (v0) | John | ✅ Complete |
| §4 | Lowering / Compilation Model | Ken | ✅ Complete |
| §5 | Runtime Behavior | Ken | ✅ Complete |
| §6 | Mobile UX Contract | Barbara | ✅ Complete |
| §7 | AI Assistance Layer | Barbara | ✅ Complete |
| §8 | Prototype Scope (2-Week MVP) | Ken | ✅ Complete |
| §9 | Deliverables Summary | Barbara | ✅ Complete |

**Total decisions filed:**
- Ken: 8 architectural decisions → `.squad/decisions/inbox/ken-vacation-kit.md`
- John: 13 DSL decisions (VK-01–VK-13) → `.squad/decisions/inbox/john-vacation-kit-dsl.md`
- Barbara: 6 integration decisions → `.squad/decisions/inbox/barbara-vacation-kit.md`

---

*Vacation Domain Kit v0 — assembled 2026-04-22*
*Authors: Ken (Architecture), John (YAML DSL), Barbara (Integrations & UX)*
