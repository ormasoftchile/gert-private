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
