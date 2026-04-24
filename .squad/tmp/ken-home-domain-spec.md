# gert-domain-home v0 — Architectural Specification

**Author:** Ken (Software Architect)  
**Date:** 2024-04-21  
**Status:** Draft v0  
**Purpose:** Architectural design for GERT's first consumer-domain kit

---

## 1. Domain Purpose

**gert-domain-home** is a domain kit that provides household-specific abstractions over GERT's core durable-run runtime. It does NOT re-implement runtime semantics — it compiles a household vocabulary (routine, incident, delegation, zone, asset) down to GERT's generic primitives (timer-backed runs, human tasks, evidence, projections, policies).

### What Problem It Solves

Household maintenance suffers from three failures:
1. **Forgotten routines** — pool cleaning, lawn care, filter replacements slip through the cracks
2. **Reactive chaos** — broken hinge, leaky pipe, noisy pump become weeks-long partial fixes with no tracking
3. **Delegation opacity** — owner travels, family member has no clear task list, owner returns with no verification record

Home ownership generates dozens of recurring tasks (weekly/monthly/seasonal) plus unpredictable reactive incidents. Current tools are either too rigid (calendar reminders with no evidence trail) or too general (issue trackers designed for software teams).

gert-domain-home provides:
- Seasonal-aware recurring maintenance with cadence rules
- Structured reactive incident handling (diagnose → buy parts → fix → verify)
- Time-bounded task delegation with simplified executor views
- Evidence capture (photo/note) proving task completion
- Mobile-first calm interface (no dashboards, no KPIs, just "what to do today")

### Why This Is the Right First Consumer Domain

Three strategic reasons:

**1. Pattern Coverage**  
Home management exercises all three core GERT patterns in one domain:
- **Recurring** — routines with timer-backed runs, seasonal cadence
- **Reactive** — ad-hoc incident runs with no predefined schedule
- **Delegation** — time-bounded policy routing with scoped projections

This validates GERT's runtime more thoroughly than an enterprise kit focused on one pattern (e.g., pure approval workflows).

**2. Real Daily Use**  
Unlike enterprise domains requiring multi-person coordination, home management is:
- Personal (one owner, optional family delegates)
- Daily (tasks appear every week, not quarterly)
- Tangible (mowed lawn, cleaned pool, fixed hinge = immediate visible proof)

This drives daily mobile app usage, surfacing UX friction and runtime assumptions invisible in less-frequent enterprise scenarios.

**3. Mobile Companion Design**  
The home domain demands a calm, practical mobile app — not a web dashboard. This forces GERT's projection and policy systems to support lightweight read-side filtering, pushing architectural decisions toward simplicity:
- Projection must be fast enough for mobile refresh latency
- Event schema must be compact enough for mobile SSE
- Evidence capture must work offline (photo → upload when connected)

### Strategic Value

gert-domain-home validates:
- Domain kit compilation model (authoring DSL → GERT primitives, no runtime reimplementation)
- Timer flexibility (seasonal rules, variable intervals)
- Human task + evidence workflow (photo/note attachment, completion tracking)
- Projection performance (delegate view filtering in <100ms)
- Policy routing (time-bounded executor reassignment)
- Non-technical user experience (no YAML, no runbook jargon)

Success criteria: A non-technical home owner can use the mobile app daily for 30 days without requesting help.

---

## 2. Core Domain Concepts

### property
The top-level ownable unit. Represents a physical residence (house, apartment, estate).  
**Attributes:** name, address, owner_id, created_at  
**Contains:** zones, assets, routines, active incidents, delegates  
**Example:** "Casa Santiago" at "Rua da Fonte 42, Lisboa"

### zone
A named area within a property. Scopes assets and routines to physical locations.  
**Attributes:** zone_id, name, property_id  
**Purpose:** Group related maintenance tasks (pool care, lawn care, garage) for spatial organization  
**Examples:** `pool`, `front_lawn`, `backyard`, `garage`, `kitchen`, `garden`  
**Cardinality:** One property has 3–15 zones (typical)

### asset
A physical thing requiring maintenance or tracking. Lives in a zone.  
**Attributes:** asset_id, name, zone_id, install_date, model, serial_number  
**Purpose:** Track maintenance history for expensive or complex equipment  
**Examples:** pool pump (installed 2023-03-15), car (Volvo XC60 2019), irrigation controller  
**Optional:** Not all zones have assets (front_lawn has no tracked equipment; pool has pump + filter + chlorinator)

### routine
A scheduled recurring maintenance task. Tied to a zone or asset.  
**Attributes:** routine_id, name, zone_id, asset_id (optional), cadence, executor_id, requires_evidence  
**Purpose:** Define maintenance work that repeats on a schedule  
**Examples:**
- "Pool chemical check" — every 3 days, zone=pool, requires_evidence=true
- "Lawn mowing" — every 7 days (summer), zone=front_lawn, executor=owner
- "Car oil change" — every 180 days, asset=car, executor=mechanic_contractor

### cadence
The recurrence rule for a routine. Two types: simple interval or seasonal schedule.

**Simple interval:**  
`every_n_days: 7` → repeats every 7 days from last completion

**Seasonal schedule:**  
```
spring: 10 days
summer: 7 days
autumn: 14 days
winter: 21 days
```
Season determined by current date in owner's timezone.

**Purpose:** Capture domain knowledge that maintenance frequency varies with weather/usage  
**GERT lowering:** Cadence compiles to timer policy. At wakeup, policy evaluates current season (if seasonal) or fixed interval (if simple) to compute next wakeup time.

### seasonal rule
A cadence variant where interval changes by season.  
**Definition:** Four intervals (spring_days, summer_days, autumn_days, winter_days)  
**Season boundaries:** Hardcoded (Northern Hemisphere): Mar 20, Jun 21, Sep 22, Dec 21  
**Evaluation:** At timer wakeup, runtime checks current date → determines season → sets next wakeup to season-specific interval  
**Example:** Lawn mowing: 7d summer (hot, fast growth), 14d spring/autumn (moderate), 21d winter (dormant)

**v0 note:** Seasonal rules deferred to v1. v0 uses simple interval only.

### maintenance task
A specific instance of a routine being executed.  
**Attributes:** task_id, routine_id, due_date, status (pending/in_progress/completed/skipped), executor_id, completed_at, evidence  
**Lifecycle:**
1. Routine timer wakes → creates maintenance task with due_date=today
2. Task appears in executor's "Today" view
3. Executor marks in_progress (optional) → completed (required)
4. Evidence attached (photo/note)
5. Task status → completed, routine timer resets for next cadence

**GERT lowering:** Maintenance task = human task step in routine's durable run. Evidence = GERT evidence primitive.

### incident
An unplanned event requiring reactive work. Not scheduled — reported by user.  
**Attributes:** incident_id, title, description, zone_id, asset_id (optional), reported_by, reported_at, severity  
**Examples:**
- "Leak under kitchen sink" (zone=kitchen, severity=high)
- "Broken hinge on garden gate" (zone=garden, severity=low)
- "Pool pump making loud noise" (zone=pool, asset=pool_pump, severity=medium)

**Trigger:** User taps "Report Incident" in mobile app → selects zone → describes problem → incident created  
**GERT lowering:** Incident spawns a repair run (ad-hoc durable run, no timer)

### repair run
The durable run spawned to handle an incident. Structured workflow: diagnose → buy parts → fix → verify.  
**Steps (all human tasks):**
1. **Diagnose** — identify root cause, determine parts needed  
   Evidence: note describing diagnosis ("needs M8 hinge pin")
2. **Buy parts** — acquire materials  
   Evidence: note with cost + vendor ("€2.40 at Leroy Merlin")
3. **Fix** — perform repair  
   Evidence: photo of completed repair
4. **Verify** — confirm fix works, no regression  
   Evidence: note ("gate opens/closes smoothly, no squeak")

**Status tracking:** Each step has status (pending/completed). Run remains active until all steps completed.  
**GERT lowering:** Repair run = durable run with 4 sequential human task steps. Each step has evidence requirement.

### consumable
A tracked item with known replacement interval. Triggers a routine when due.  
**Attributes:** consumable_id, name, asset_id, install_date, replacement_interval_days, next_replacement_date  
**Examples:**
- Pool filter (install_date=2024-01-10, interval=90 days, next=2024-04-10)
- Car air filter (install_date=2023-06-15, interval=365 days, next=2024-06-15)
- Water heater anode rod (install_date=2022-11-01, interval=730 days, next=2024-11-01)

**Behavior:**  
When next_replacement_date approaches (within 7 days), a routine is auto-created: "Replace {consumable_name}".  
On completion, install_date updates to today, next_replacement_date recalculates.

**v0 note:** Consumable tracking deferred to v1. v0 routines are manually authored, not consumable-triggered.

### delegation
The temporary transfer of task execution authority to another person.  
**Attributes:** delegation_id, delegate_name, delegate_contact (email/phone), start_date, end_date, scoped_tasks (filter)  
**Purpose:** Owner travels Apr 25–May 2 → son Carlos handles daily tasks (watering, dog, pool basket check)  
**Scope:** Delegate sees ONLY tasks assigned to them, ONLY for dates within delegation window  
**Permissions:** Delegate can mark tasks complete, attach evidence. Delegate CANNOT create incidents, modify routines, or change property.

**GERT lowering:**
- Delegation = time-bounded policy (active between start_date and end_date)
- Policy routes human tasks matching scoped_tasks filter to delegate's executor_id
- Policy expires automatically at end_date; tasks revert to owner

### away mode
The runtime state when owner is absent and a delegate is active.  
**Activation:** Owner sets departure/return dates + assigns delegate → away mode activates  
**Behavior:**
- Delegate receives daily notification with today's task list (simplified view)
- Tasks not completed by threshold (e.g., 8pm) trigger reminder notification to delegate
- Owner view switches to "Away" dashboard showing delegation status + delegate's completion rate
- At return date, away mode auto-deactivates, tasks revert to owner, owner reviews evidence

**GERT lowering:**
- Away mode = projection that filters property run graph to delegate-scoped tasks
- Reminder = GERT timer with policy: if task status != completed by 20:00, emit notification event

### executor
The person assigned to complete a task. Can be owner, family member, or contractor.  
**Attributes:** executor_id, name, contact (email/phone), role (owner/delegate/contractor)  
**Purpose:** Track WHO did WHAT, enabling delegation and contractor management  
**Task assignment:** Each task has executor_id. Mobile app shows tasks for current user's executor_id.  
**Example:** Owner (executor_id=owner_1), Son Carlos (executor_id=delegate_carlos), Pool Service Inc (executor_id=contractor_pool)

### evidence
A photo, note, or timestamp attached to a completed task. Proves task was done.  
**Types:**
- **Photo** — JPG/PNG image, max 5MB, uploaded from mobile camera
- **Note** — text (max 500 chars), typed in completion flow
- **Timestamp** — GPS coordinates + capture time (optional, for location-sensitive tasks)

**Purpose:**
- Verify task completion (mowed lawn photo proves mowing happened)
- Delegation accountability (owner reviews son's evidence on return)
- Incident history (repair run photos show before/during/after state)

**Storage:** Evidence stored as GERT evidence primitive (immutable, SHA256-hashed, append-only log)  
**Mobile UX:** Tap "Complete Task" → optional camera icon → tap photo → add note → confirm

---

## 3. Authoring Model (Compilation Surface)

The Home domain kit provides a YAML-based authoring DSL that compiles to GERT core primitives. This section is implicit in the lowering (Section 4), but outlined here for clarity.

**Authoring artifacts:**
1. **property.yaml** — defines zones, assets, owner
2. **routines/*.yaml** — one file per routine, declares cadence + zone + evidence requirements
3. **incidents** — created dynamically via mobile app (no authoring file)

**Example routine authoring (lawn_mowing.yaml):**
```yaml
name: "Lawn mowing"
zone: front_lawn
cadence:
  type: simple
  every_n_days: 7
executor: owner
requires_evidence: true
evidence_type: photo
```

**Compilation:** Home domain compiler reads `routines/*.yaml` → generates GERT durable runs with timer policies + human tasks + evidence requirements.

**v0 authoring:** Manual YAML editing. v1 will add mobile app UI for routine creation.

---

## 4. Lowering into GERT Core Semantics

This section defines how Home kit concepts compile to GERT runtime primitives.

### routine → timer-backed durable run

**Authoring input:**
```yaml
name: "Pool chemical check"
zone: pool
cadence:
  type: simple
  every_n_days: 3
executor: owner
requires_evidence: true
```

**GERT lowering:**
1. Create durable run with ID `routine:{routine_id}`
2. Attach timer with initial wakeup = now + 3 days
3. Add human task step: "Pool chemical check (pool zone)"
4. Attach evidence requirement to step (type=photo_or_note)
5. On step completion, timer resets to now + 3 days

**Runtime behavior:**
- Timer wakes → run resumes → human task appears in executor's task list
- Executor completes task → attaches evidence → step completes
- Run yields (paused) → timer resets → run sleeps until next wakeup

**Key invariant:** Routine run NEVER completes. It pauses after each execution, waiting for next timer wakeup.

### cadence → timer policy with optional seasonal override

**Simple cadence (every_n_days: 7):**
- Timer policy: `next_wakeup = last_completion_time + 7 days`
- No date logic, no external data, deterministic

**Seasonal cadence (deferred to v1):**
- Timer policy evaluates current date at wakeup time
- Determines season (spring/summer/autumn/winter) from date
- Looks up interval for that season (e.g., summer=7d, winter=21d)
- Sets `next_wakeup = now + season_interval`

**GERT primitive:** Timer with policy function. Policy is Rego code (OPA) evaluated at wakeup time.

**v0 limitation:** Only simple cadence supported. Seasonal rules require OPA integration (GERT v2.1).

### seasonal rule → policy evaluated at wakeup time

**Authoring (deferred to v1):**
```yaml
cadence:
  type: seasonal
  spring_days: 10
  summer_days: 7
  autumn_days: 14
  winter_days: 21
```

**GERT lowering (design only, not implemented in v0):**
1. Timer policy function receives current date as input
2. Policy computes season:
   ```rego
   season := "summer" if date >= "2024-06-21" and date < "2024-09-22"
   season := "autumn" if date >= "2024-09-22" and date < "2024-12-21"
   ...
   ```
3. Policy looks up interval for season: `intervals[season]`
4. Returns `next_wakeup = now + intervals[season] * 86400`

**Complexity note:** Requires GERT to pass runtime context (current date, timezone) to policy evaluator. This is NOT yet implemented in GERT v2.0.

### incident → ad-hoc run triggered by user report

**User action:** Tap "Report Incident" → select zone → enter description → submit

**GERT lowering:**
1. Create durable run with ID `incident:{incident_id}`
2. NO timer attached (ad-hoc, starts immediately)
3. Run state = active
4. Spawn repair run as sub-run (see next section)

**Key difference from routine:** Incident run has no recurrence. It runs once, completes, and never wakes again.

### repair run → structured sub-run with ordered steps

**Incident spawns repair run:**
```
incident:leak_kitchen_sink
  └─ repair_run:leak_kitchen_sink_repair
       ├─ step1: diagnose (human task)
       ├─ step2: buy_parts (human task)
       ├─ step3: fix (human task)
       └─ step4: verify (human task)
```

**GERT lowering:**
1. Repair run is a sub-run of incident run
2. Four sequential human task steps (cannot proceed to step N+1 until step N completes)
3. Each step has evidence requirement (note or photo)
4. On step4 completion, repair run completes → incident run completes

**Execution semantics:**
- Step advancement is manual (user taps "Next Step" or step auto-advances on completion)
- Steps can pause indefinitely (e.g., waiting for parts delivery)
- Evidence attached to each step persists in GERT evidence log
- If incident is abandoned, repair run remains in partial state (no automatic cleanup)

**GERT primitives used:**
- Sub-run (repair run is child of incident run)
- Human task (all four steps)
- Evidence (attached to each step)
- Sequential ordering (steps have explicit dependency chain)

### consumable → asset attribute that feeds a routine

**Authoring (deferred to v1):**
```yaml
asset:
  id: pool_filter
  name: "Pool filter cartridge"
  zone: pool
  consumable:
    install_date: 2024-01-10
    replacement_interval_days: 90
```

**GERT lowering (design only, not in v0):**
1. Timer with wakeup = install_date + interval - 7 days (reminder threshold)
2. On wakeup, create routine: "Replace pool filter cartridge"
3. Routine is one-time (not recurring)
4. On routine completion, evidence timestamp becomes new install_date
5. Timer resets to new install_date + interval - 7 days

**Challenge:** This requires dynamic routine creation (not pre-authored). GERT v2.0 supports ad-hoc run creation, but Home kit compiler needs to generate run templates on-the-fly.

**Deferral rationale:** Requires template-based run instantiation, which is not yet designed in GERT core.

### delegation → role assignment scoped to time window + task filter

**Authoring (mobile app UI):**
- Owner sets start_date=2024-04-25, end_date=2024-05-02
- Owner assigns delegate_id=carlos
- Owner selects scoped tasks: [water_plants, feed_dog, pool_basket_check]

**GERT lowering:**
1. Create policy with ID `delegation:{delegation_id}`
2. Policy has time bounds: active_start=2024-04-25T00:00, active_end=2024-05-02T23:59
3. Policy rule (Rego pseudocode):
   ```rego
   executor_id := input.default_executor  # owner
   if now >= active_start and now <= active_end:
     if input.task_name in scoped_tasks:
       executor_id := delegate_id  # carlos
   ```
4. Policy is evaluated when human task is created → routes task to delegate if match

**Runtime behavior:**
- Before Apr 25: all tasks route to owner
- Apr 25–May 2: scoped tasks route to carlos, others route to owner
- After May 2: policy expired, all tasks route to owner

**GERT primitive:** Policy with time-bounded activation + task filter condition.

### away mode → projection that filters to delegate-visible tasks + reminder policy

**Activation:**
- Owner sets delegation (see above)
- Away mode activates automatically when delegation starts

**GERT lowering:**
1. Create projection with ID `away_mode:{delegation_id}`
2. Projection filters run graph:
   ```
   SELECT task FROM property_runs
   WHERE task.executor_id = delegate_id
     AND task.due_date >= active_start
     AND task.due_date <= active_end
     AND task.status IN ['pending', 'in_progress']
   ```
3. Delegate's mobile app queries this projection (NOT the full property run graph)
4. Create reminder policy:
   ```
   FOR EACH task in delegate_tasks:
     IF task.status != 'completed' AND now > task.due_date + 12 hours:
       EMIT notification_event(delegate_id, task.name, "reminder")
   ```

**Projection update frequency:** Every time a task status changes, projection recalculates.  
**Performance requirement:** Projection query must complete in <100ms for mobile app responsiveness.

**GERT primitives used:**
- Projection (read-side filter)
- Policy (reminder logic)
- Notification event (emitted when reminder triggers)

### maintenance task → human task with evidence requirement

**Already covered above.** Maintenance task IS a human task in GERT runtime.

**Attributes mapped to GERT human task:**
- `task.name` → human_task.description
- `task.due_date` → human_task.created_at (or metadata field)
- `task.executor_id` → human_task.assigned_to
- `task.evidence` → GERT evidence attached to step completion

**Completion flow:**
1. Executor taps task in mobile app
2. App calls GERT API: `POST /run/{run_id}/step/{step_id}/complete`
3. Executor attaches evidence (photo/note)
4. GERT marks step complete, appends evidence to log
5. Routine timer resets (if recurring) or run completes (if incident repair)

### evidence → GERT evidence primitive

**GERT evidence API:**
```
POST /run/{run_id}/step/{step_id}/evidence
Content-Type: multipart/form-data

file: <photo.jpg>
note: "pH slightly low, added 200ml acid"
timestamp: 2024-04-21T15:30:00Z
```

**Storage:**
- Photo stored as SHA256-hashed blob in evidence store
- Note + timestamp stored as JSON in evidence log (append-only)
- Evidence is immutable (cannot edit or delete)

**Mobile app UX:**
- Camera icon on task completion screen
- Tap → capture photo → add optional note → submit
- Evidence uploads asynchronously (queued if offline)

**GERT invariant:** Evidence is tied to step completion. Cannot attach evidence to incomplete step.

---

## 5. Delegation / Away Mode Model

### Design Goals

1. **Temporary authority** — delegate has execution power ONLY during absence window
2. **Simplified view** — delegate sees daily checklist, NOT full property state
3. **Accountability** — owner reviews evidence on return
4. **No surprise edits** — delegate cannot modify routines, create incidents, or change property config

### Data Model

**Delegation record:**
```yaml
delegation_id: del_001
delegate_name: "Carlos"
delegate_contact: "carlos@example.com"
start_date: "2024-04-25"
end_date: "2024-05-02"
scoped_tasks:
  - water_plants
  - feed_dog
  - pool_basket_check
created_by: owner
created_at: "2024-04-20T10:00:00Z"
```

**Key fields:**
- `scoped_tasks` — whitelist of routine names delegate can see/execute
- `start_date` / `end_date` — time bounds (inclusive)
- `delegate_contact` — email/phone for notifications

### Runtime Semantics

**Phase 1: Setup (before departure)**  
Owner opens mobile app → taps "Going Away" → selects dates → enters delegate name/contact → selects tasks from list → confirms  
Backend creates delegation record + policy + projection

**Phase 2: Active delegation (Apr 25–May 2)**  
- Each morning at 7am, delegate receives push notification: "3 tasks for today"
- Delegate opens app → sees "Today" tab with simple list:
  ```
  Today (Apr 26)
  1. ☐ Water plants (backyard)
  2. ☐ Feed dog
  3. ☐ Check pool basket
  ```
- Delegate taps task → marks complete → optional photo → done
- If task not completed by 8pm, delegate receives reminder: "Water plants still pending"

**Phase 3: Owner return (May 3)**  
- Away mode auto-expires at end_date
- Tasks revert to owner
- Owner opens app → "Delegation Summary" screen shows:
  - 21 tasks assigned to Carlos (Apr 25–May 2)
  - 19 completed (90%)
  - 2 skipped (dog feeding on Apr 28, Apr 30)
  - Evidence gallery (photos from completed tasks)

### Delegate Permissions

**Can:**
- View tasks assigned to them within delegation window
- Mark tasks complete
- Attach evidence (photo/note)
- Skip tasks (status=skipped, requires reason note)

**Cannot:**
- View full property config (zones, assets, all routines)
- Create new incidents
- Modify routine cadence or settings
- Add/remove other delegates
- Delete evidence
- Access owner's historical data (tasks before delegation start)

**Mobile app enforcement:**  
Delegate login shows ONLY "Today" tab. No access to Property/History/Settings tabs.

### GERT Policy Implementation

**Policy pseudocode (Rego):**
```rego
package home_delegation

# Input: { task_name, default_executor, current_date, delegations }
# Output: { executor_id }

executor_id := input.default_executor  # owner by default

# Check all active delegations
active_delegation := delegations[_]
  active_delegation.start_date <= input.current_date
  active_delegation.end_date >= input.current_date
  input.task_name in active_delegation.scoped_tasks

# If match found, route to delegate
executor_id := active_delegation.delegate_id
```

**Evaluation point:** When human task is created (routine timer wakes), GERT runtime calls policy to determine `executor_id`.

**Time-bound enforcement:** Policy returns default executor if current date outside delegation window. No manual deactivation needed.

### GERT Projection Implementation

**Projection query (SQL pseudocode):**
```sql
SELECT
  task.id,
  task.name,
  task.due_date,
  task.status,
  task.zone
FROM property_tasks task
JOIN delegations del ON task.executor_id = del.delegate_id
WHERE del.delegation_id = :delegation_id
  AND task.due_date >= del.start_date
  AND task.due_date <= del.end_date
  AND task.status IN ('pending', 'in_progress')
ORDER BY task.due_date ASC
```

**Projection API endpoint:**
```
GET /property/{property_id}/delegate/{delegate_id}/tasks?date=2024-04-26
```

**Response (JSON):**
```json
{
  "date": "2024-04-26",
  "tasks": [
    {
      "id": "task_001",
      "name": "Water plants",
      "zone": "backyard",
      "status": "pending",
      "evidence_required": true
    },
    {
      "id": "task_002",
      "name": "Feed dog",
      "zone": "kitchen",
      "status": "pending",
      "evidence_required": false
    }
  ]
}
```

### Concrete Example: Owner Travels Apr 25–May 2

**Setup (Apr 20):**  
Owner Maria opens app → "Going Away" → dates: Apr 25–May 2 → delegate: Carlos (son) → tasks: water plants, feed dog, check pool basket → confirm

**Backend actions:**
1. Create delegation record (ID: del_001)
2. Create policy: route scoped tasks to executor_id=carlos during Apr 25–May 2
3. Create projection: filter tasks for carlos, Apr 25–May 2 only
4. Send confirmation email to carlos@example.com: "You're managing Casa Santiago Apr 25–May 2. 3 daily tasks."

**Day 1 (Apr 26):**  
7:00am — Carlos receives push notification: "3 tasks for today"  
10:30am — Carlos opens app → sees Today tab:
```
Today (Apr 26)
1. ☐ Water plants (backyard)
2. ☐ Feed dog
3. ☐ Check pool basket
```
10:35am — Carlos taps "Water plants" → camera icon → takes photo of watered plants → taps "Complete"  
Backend: task status → completed, evidence uploaded (photo SHA256: a3f2...), routine timer unaffected  
11:00am — Carlos completes "Feed dog" (no photo, just tap Complete)  
8:00pm — Carlos has not completed "Check pool basket" → reminder notification: "Pool basket check still pending"  
8:15pm — Carlos completes pool basket check → attaches note: "basket was clean, no debris"

**Day 2–7 (Apr 27–May 2):**  
Same daily pattern. Carlos completes 19 of 21 tasks. Misses dog feeding on Apr 28 (forgot) and Apr 30 (out all day).

**Return (May 3):**  
Maria returns home, opens app → "Delegation Summary" appears:
```
Delegation Summary (Apr 25–May 2)
Delegate: Carlos
Total tasks: 21
Completed: 19 (90%)
Skipped: 2

Evidence:
[Photo thumbnails from 15 completed tasks with photos]

Tap to review details →
```

Maria taps → sees day-by-day breakdown:
```
Apr 26: 3/3 ✓
Apr 27: 3/3 ✓
Apr 28: 2/3 (skipped: feed dog)
Apr 29: 3/3 ✓
Apr 30: 2/3 (skipped: feed dog)
May 1: 3/3 ✓
May 2: 3/3 ✓
```

Maria taps Apr 28 "feed dog" → sees Carlos left note: "Forgot, sorry. Dog had food in bowl from morning."  
Maria is satisfied — delegation worked, evidence clear, no surprises.

### Reminder Mechanics

**Reminder policy (Rego pseudocode):**
```rego
package home_reminders

# Runs every hour (cron: "0 * * * *")
# For each delegate task due today:
#   If status != completed AND time > due_date + 12 hours:
#     Emit notification

for task in delegate_tasks:
  if task.status != "completed":
    if now > task.due_date + duration("12h"):
      emit_notification(task.executor_id, task.name, "reminder")
```

**Notification format (push):**
```
Title: "Task reminder"
Body: "Water plants (backyard) still pending"
Action: "Open app" → deep link to task
```

**Frequency:** One reminder per task per day. No repeated reminders (to avoid annoyance).

---

## 6. Mobile App Concept

### Design Principles

1. **Calm interface** — no dashboards, no analytics, no clutter
2. **Today-focused** — primary view answers "what do I do today?"
3. **Evidence-first** — photo capture is 2 taps, not buried in menus
4. **Trustworthy** — completed tasks stay visible (with timestamp) for confidence
5. **Non-technical** — no jargon (no "runs", "steps", "policies" — use "tasks", "zones", "routines")

### Tab Structure

**4 tabs (bottom navigation):**

1. **Today** (default landing)
2. **Property**
3. **Tasks**
4. **History**

### Tab 1: Today

**Purpose:** Show tasks due today (or overdue) for current executor.

**Layout:**
```
┌─────────────────────────────────┐
│  Today — Apr 26                 │
├─────────────────────────────────┤
│  🟢 3 tasks                      │
│                                  │
│  ☐ Water plants (backyard)      │
│     Tap to complete              │
│                                  │
│  ☐ Feed dog (kitchen)            │
│     Tap to complete              │
│                                  │
│  ☐ Pool basket check (pool)     │
│     Tap to complete              │
│                                  │
│  ─────────────────               │
│  Upcoming (Apr 27)               │
│  • Lawn mowing (front_lawn)      │
└─────────────────────────────────┘
```

**Interaction:**  
Tap task → task detail screen:
```
┌─────────────────────────────────┐
│  ← Water plants                 │
├─────────────────────────────────┤
│  Zone: backyard                 │
│  Due: Today (Apr 26)             │
│  Routine: Every 2 days           │
│                                  │
│  📷 Add photo (optional)         │
│  📝 Add note (optional)          │
│                                  │
│  [Mark Complete]                 │
└─────────────────────────────────┘
```

Tap "Add photo" → camera opens → capture → preview → confirm → returns to task detail  
Tap "Mark Complete" → task disappears from Today, appears in History, evidence uploaded

**Away mode variant:**  
If current user is a delegate (not owner), Today tab shows ONLY delegate-scoped tasks. No "Upcoming" section. No access to other tabs.

**Empty state:**  
```
┌─────────────────────────────────┐
│  Today — Apr 26                 │
├─────────────────────────────────┤
│  ✅ All done!                    │
│                                  │
│  No tasks due today.             │
│                                  │
│  Next task:                      │
│  Pool chemical check (Apr 28)    │
└─────────────────────────────────┘
```

### Tab 2: Property

**Purpose:** Visual map of property zones, assets, upcoming routines.

**Layout:**
```
┌─────────────────────────────────┐
│  Casa Santiago                  │
│  Rua da Fonte 42, Lisboa         │
├─────────────────────────────────┤
│  🏊 Pool                          │
│     Last: Chemical check (Apr 23)│
│     Next: Chemical check (Apr 26)│
│     Tap for details →            │
│                                  │
│  🌳 Front Lawn                    │
│     Last: Mowing (Apr 20)        │
│     Next: Mowing (Apr 27)        │
│     Tap for details →            │
│                                  │
│  🏡 Backyard                      │
│     Last: Plant watering (Apr 25)│
│     Next: Plant watering (Apr 27)│
│     Tap for details →            │
│                                  │
│  🚗 Garage                        │
│     No active routines           │
│     Tap for details →            │
└─────────────────────────────────┘
```

**Zone detail (tap "Pool"):**
```
┌─────────────────────────────────┐
│  ← Pool                          │
├─────────────────────────────────┤
│  Assets:                         │
│  • Pool pump (Hayward SP2610X15)│
│  • Filter (installed Jan 10)     │
│                                  │
│  Active Routines:                │
│  • Chemical check (every 3 days) │
│     Last: Apr 23, next: Apr 26   │
│  • Basket clean (every 7 days)   │
│     Last: Apr 22, next: Apr 29   │
│                                  │
│  Recent History:                 │
│  • Apr 23: Chemical check ✓      │
│  • Apr 22: Basket clean ✓        │
│  • Apr 19: Chemical check ✓      │
│                                  │
│  [Report Incident]               │
└─────────────────────────────────┘
```

**Tap "Report Incident":**  
Opens incident form → select zone (pre-filled: Pool) → enter description → severity (low/medium/high) → submit → repair run starts

### Tab 3: Tasks

**Purpose:** Show active repair runs (incident-triggered) and upcoming routines (next 7 days).

**Layout:**
```
┌─────────────────────────────────┐
│  Active Repairs                 │
├─────────────────────────────────┤
│  🔧 Broken hinge (garden gate)   │
│     Step 2/4: Buy parts          │
│     Tap to continue →            │
│                                  │
│  ─────────────────               │
│  Upcoming Routines (7 days)      │
├─────────────────────────────────┤
│  Apr 26:                         │
│  • Water plants                  │
│  • Feed dog                      │
│  • Pool basket check             │
│                                  │
│  Apr 27:                         │
│  • Lawn mowing                   │
│  • Plant watering                │
│                                  │
│  Apr 28:                         │
│  • Pool chemical check           │
└─────────────────────────────────┘
```

**Tap repair run:**  
Opens repair run detail → shows all 4 steps, current step highlighted → tap current step → complete → evidence → next step activates

### Tab 4: History

**Purpose:** Browse completed tasks with evidence. Filter by date/zone.

**Layout:**
```
┌─────────────────────────────────┐
│  History                        │
│  [Filter: All zones ▼]  [Apr ▼] │
├─────────────────────────────────┤
│  Apr 25                          │
│  ✅ Plant watering (backyard)    │
│     📷 Photo • 10:30am           │
│                                  │
│  Apr 23                          │
│  ✅ Pool chemical check (pool)   │
│     📝 Note • 3:15pm             │
│     "pH slightly low, added acid"│
│                                  │
│  Apr 22                          │
│  ✅ Pool basket clean (pool)     │
│     📷 Photo • 11:00am           │
│                                  │
│  Apr 20                          │
│  ✅ Lawn mowing (front_lawn)     │
│     📷 Photo • 9:45am            │
│     ✅ Lawn mowing (backyard)    │
│     📷 Photo • 10:15am           │
└─────────────────────────────────┘
```

**Tap task:**  
Opens evidence detail → full-size photo, note text, timestamp, zone, executor name

**Filter by zone:**  
Dropdown: All zones, Pool, Front Lawn, Backyard, Garage, Kitchen → filters to tasks in selected zone

**Filter by month:**  
Dropdown: Apr, Mar, Feb, Jan → filters to tasks completed in selected month

**Delegation use case:**  
Owner returns from travel → opens History → filter: Apr → sees all tasks Carlos completed, with photos/notes as evidence

### UX Tone

**What it feels like:**
- Todoist (clean task lists, calm completion checkmarks)
- Apple Reminders (simple, no clutter, photo attachment is fast)
- Weather app (glanceable, answers "what do I need to know today?")

**What it does NOT feel like:**
- Jira (no sprints, no boards, no story points)
- Asana (no projects, no timelines, no team collaboration features)
- Notion (no databases, no nested pages, no markdown editing)

**Color palette:**
- Green for completed (✅)
- Orange for overdue (⚠️)
- Blue for upcoming (informational)
- No red (too alarming for home tasks)

**Typography:**
- System font (SF Pro on iOS, Roboto on Android)
- Large touch targets (minimum 44pt per Apple HIG)
- High contrast (WCAG AA compliant)

**Animations:**
- Task completion: checkmark fade-in + gentle bounce
- Photo capture: shutter animation
- Screen transitions: slide from right (iOS standard)

**Offline behavior:**
- Today tab works offline (cached task list)
- Evidence upload queued when offline, syncs when reconnected
- No error messages — silent queue + sync badge

---

## 7. AI Assistance Opportunities

### Design Constraints

AI assistance in the Home domain MUST respect these boundaries:

1. **Advisory only** — AI suggests, user approves or dismisses
2. **No autonomous execution** — AI cannot mark tasks complete, modify routines, or create incidents
3. **Grounded in data** — suggestions reference real property data (cadence history, weather, usage patterns)
4. **Dismissible** — every hint has "Don't suggest this again" or "Snooze 30 days" option
5. **Transparent** — hint shows WHY (e.g., "Based on warm weather forecast")

### Hint Scenarios

#### 1. Seasonal Cadence Adjustment

**Trigger:** Weather forecast shows sustained temperature change  
**Hint:**
```
┌─────────────────────────────────┐
│  💡 Suggestion                  │
├─────────────────────────────────┤
│  Warm week ahead (avg 32°C).    │
│  Increase plant watering to      │
│  every 2 days?                   │
│                                  │
│  Current: every 3 days           │
│  Suggested: every 2 days         │
│  (Apr 26–May 3 only)             │
│                                  │
│  [Apply]  [Dismiss]              │
└─────────────────────────────────┘
```

**Data source:** Weather API (OpenWeatherMap, local forecast)  
**Logic:** If 7-day avg temp > 30°C AND current cadence > 2 days, suggest temporary increase  
**Behavior:** If user taps "Apply", cadence changes for 7 days, then reverts to original

#### 2. Pool Usage Spike

**Trigger:** Pool chemical check completed 4 days in a row (unusual frequency)  
**Hint:**
```
┌─────────────────────────────────┐
│  💡 Suggestion                  │
├─────────────────────────────────┤
│  Pool used 4 days in a row.     │
│  Suggest extra cleaning this     │
│  week?                           │
│                                  │
│  Add one-time task:              │
│  "Pool vacuum and skim"          │
│  Due: Apr 28                     │
│                                  │
│  [Add Task]  [Dismiss]           │
└─────────────────────────────────┘
```

**Data source:** Completion history (evidence timestamps)  
**Logic:** If pool-related tasks completed > 3 days consecutively, suggest deep clean  
**Behavior:** If user taps "Add Task", creates one-time task (not recurring routine)

#### 3. Missed Routine

**Trigger:** Routine skipped last execution, now due again  
**Hint:**
```
┌─────────────────────────────────┐
│  💡 Suggestion                  │
├─────────────────────────────────┤
│  Lawn mowing was skipped last    │
│  week (Apr 20).                  │
│  Reschedule for this Saturday?   │
│                                  │
│  Original cadence: every 7 days  │
│  Suggested: Apr 27 (Saturday)    │
│                                  │
│  [Schedule]  [Dismiss]           │
└─────────────────────────────────┘
```

**Data source:** Task status history (skipped vs. completed)  
**Logic:** If task skipped AND next occurrence is weekday, suggest weekend reschedule  
**Behavior:** If user taps "Schedule", due_date shifts to suggested date (one-time override)

#### 4. Consumable Alert

**Trigger:** Consumable replacement date approaching (within 11 days)  
**Hint:**
```
┌─────────────────────────────────┐
│  💡 Suggestion                  │
├─────────────────────────────────┤
│  Pool filter installed 89 days   │
│  ago. Replacement due in ~11     │
│  days.                           │
│                                  │
│  Create task:                    │
│  "Replace pool filter"           │
│  Due: May 5                      │
│                                  │
│  [Create Task]  [Dismiss]        │
└─────────────────────────────────┘
```

**Data source:** Asset install_date + replacement_interval (consumable tracking)  
**Logic:** If next_replacement_date - today < 14 days, suggest task creation  
**Behavior:** If user taps "Create Task", creates one-time task with evidence requirement

#### 5. Delegation Prep

**Trigger:** Owner calendar shows travel event (or user manually sets "Going Away" dates)  
**Hint:**
```
┌─────────────────────────────────┐
│  💡 Suggestion                  │
├─────────────────────────────────┤
│  You're traveling Apr 25–May 2.  │
│  3 routines fall during that     │
│  period.                         │
│                                  │
│  Set up delegation?              │
│                                  │
│  Routines:                       │
│  • Water plants (Apr 26, 28, 30) │
│  • Feed dog (daily)              │
│  • Pool basket (Apr 29)          │
│                                  │
│  [Set Up Delegation]  [Dismiss]  │
└─────────────────────────────────┘
```

**Data source:** Calendar integration (Google Calendar, iCal) OR manual "Going Away" dates  
**Logic:** If travel event detected AND routines scheduled during travel dates, suggest delegation  
**Behavior:** If user taps "Set Up Delegation", opens delegation setup flow (pre-filled with dates + suggested tasks)

### AI Hint Rules

**Frequency limits:**
- Maximum 1 hint per day (to avoid annoyance)
- Hints prioritized by severity: consumable alert > missed routine > usage spike > cadence adjustment

**Dismissal behavior:**
- "Dismiss" — hide this hint permanently (store dismissed_hint_id in user prefs)
- "Snooze 30 days" — hide this hint type for 30 days, then re-evaluate
- "Don't suggest this again" — disable this hint category globally

**Data privacy:**
- All AI logic runs locally (no cloud inference)
- Weather API is the ONLY external data source (no user data leaves device)
- Hints are deterministic (same data → same hint, no ML training)

**v0 scope:**  
AI hints deferred to v1. v0 has no suggestion system.

**v1 implementation approach:**
- Hints are Rego policies evaluated on device
- Policy inputs: property data, weather forecast, completion history
- Policy outputs: hint JSON (title, body, actions)
- Mobile app renders hint as bottom sheet notification

---

## 8. Example Weekly Experience

This section shows what using gert-domain-home feels like across one real week. Owner is home (no delegation). Mid-summer (warm weather). Property: Casa Santiago, Lisboa.

### Monday, Apr 21 — Morning

**7:00am** — Push notification: "2 tasks due today"

**10:00am** — Maria opens app → Today tab:
```
Today — Apr 21
🟢 2 tasks

☐ Lawn mowing (front_lawn + backyard)
   Tap to complete

☐ Pool basket check (pool)
   Tap to complete
```

**10:15am** — Maria finishes mowing front lawn. Taps "Lawn mowing" → task detail screen → taps camera icon → takes photo of mowed lawn → taps "Mark Complete"

Backend: Evidence uploaded (photo SHA256: b4e9...), task status → completed, routine timer resets to Apr 28 (7 days from now)

**10:45am** — Maria finishes mowing backyard. Opens app → notices "Lawn mowing" still shows as incomplete (confused — she just mowed!)

*BUG DISCOVERED:* Routine assumes one task execution, but front_lawn + backyard are two separate tasks. Design needs refinement.

*ARCHITECTURAL NOTE:* In v0, we'll model lawn_mowing as TWO routines (lawn_mowing_front, lawn_mowing_back) with same cadence. v1 can add multi-zone tasks.

**11:00am** — Maria checks pool basket (2 minutes). Returns to app → taps "Pool basket check" → no photo (quick task) → taps "Mark Complete" → adds note: "basket was clean"

Today tab now shows:
```
✅ All done!
No tasks due today.

Next task:
Pool chemical check (Apr 23)
```

### Tuesday, Apr 22 — Quiet Day

**7:00am** — No notification (no tasks due today)

**10:00am** — Maria opens app out of curiosity → Today tab:
```
✅ All done!
No tasks due today.

Next task:
Pool chemical check (Apr 23)
```

**11:00am** — Maria notices garden gate hinge is broken (squeaks loudly, doesn't close fully).  
Opens app → taps Property tab → taps "Garden" zone → taps "Report Incident"

Incident form:
```
Zone: Garden (pre-filled)
Title: [Broken hinge on garden gate]
Description: [Gate squeaks and doesn't close. Hinge looks bent.]
Severity: [Low ▼]
[Submit]
```

Maria taps "Submit" → incident created → repair run starts

**11:05am** — Maria taps Tasks tab → sees:
```
Active Repairs

🔧 Broken hinge on garden gate
   Step 1/4: Diagnose
   Tap to continue →
```

Maria taps repair run → sees step detail:
```
Step 1: Diagnose

Identify the problem and parts needed.

📝 Add note
[Complete Step]
```

Maria inspects gate hinge, types note: "Hinge pin is bent and corroded. Needs M8 replacement pin." → taps "Complete Step"

Repair run advances to step 2:
```
Step 2/4: Buy parts

Acquire materials needed for repair.

📝 Add note (cost + vendor)
[Complete Step]
```

Maria doesn't have time now. Closes app. Repair run remains at step 2 (paused).

### Wednesday, Apr 23 — Pool Chemistry + Parts Shopping

**7:00am** — Push notification: "1 task due today"

**3:00pm** — Maria opens app → Today tab:
```
Today — Apr 23
🟢 1 task

☐ Pool chemical check (pool)
   Tap to complete
```

Maria tests pool water (pH test strip). pH is 7.0 (slightly low, target 7.2–7.6).  
Maria adds 200ml muriatic acid to pool. Returns to app → taps "Pool chemical check" → adds note: "pH was 7.0, added 200ml acid" → taps "Mark Complete"

Today tab clears. Next pool check: Apr 26 (3 days from now).

**4:00pm** — Maria drives to hardware store (Leroy Merlin), buys M8 hinge pin (€2.40).

**4:30pm** — Maria opens app → Tasks tab → taps "Broken hinge" repair run → step 2 (Buy parts) → adds note: "M8 hinge pin, €2.40 at Leroy Merlin" → taps "Complete Step"

Repair run advances to step 3:
```
Step 3/4: Fix

Perform the repair.

📷 Add photo (recommended)
[Complete Step]
```

Maria doesn't have tools in the car. Will fix tomorrow. Closes app. Repair run paused at step 3.

### Thursday, Apr 24 — Weather Hint (fictional AI scenario for illustration)

**7:00am** — No tasks due today. No notification.

**10:00am** — Maria opens app. Today tab shows hint (NOTE: AI hints not in v0, this is illustrative):
```
💡 Suggestion

Temperature forecast 35°C Thu–Sun.
Water plants tomorrow?

Current: every 3 days (next due Apr 26)
Suggested: water tomorrow (Apr 25)

[Accept]  [Dismiss]
```

Maria taps "Accept" → "Water plants" task added to Apr 25 Today list.

**3:00pm** — Maria has time. Opens app → Tasks tab → taps "Broken hinge" repair run → step 3 (Fix).  
Maria goes to garden gate, removes old hinge pin, installs new M8 pin, tests gate (opens/closes smoothly, no squeak).  
Returns to app → taps camera icon → takes photo of repaired hinge → taps "Complete Step"

Repair run advances to step 4:
```
Step 4/4: Verify

Confirm repair works, no regression.

📝 Add note
[Complete Step]
```

Maria tests gate 3 more times (open/close). All good. Adds note: "Gate opens and closes smoothly, no squeak. Fixed!" → taps "Complete Step"

Repair run completes. Incident closed. Tasks tab now shows:
```
Active Repairs

(none)

─────────────────
Upcoming Routines (7 days)
...
```

Maria taps History tab → sees new entry:
```
Apr 24
✅ Broken hinge (garden gate)
   📷 Photo • 3:15pm
   "Gate fixed, no squeak"
```

### Friday, Apr 25 — AI-Suggested Watering

**7:00am** — Push notification: "1 task due today"

**9:00am** — Maria opens app → Today tab:
```
Today — Apr 25
🟢 1 task

☐ Water plants (backyard)
   Tap to complete
```

This task was added via AI suggestion yesterday (illustrated scenario).

**9:30am** — Maria waters backyard plants (10 minutes). Returns to app → taps "Water plants" → taps camera icon → takes photo of watered plants → taps "Mark Complete"

Today tab clears.

### Saturday, Apr 26 — Weekend Routines

**7:00am** — Push notification: "2 tasks due today"

**10:00am** — Maria opens app → Today tab:
```
Today — Apr 26
🟢 2 tasks

☐ Water plants (backyard)
   Tap to complete

☐ Pool basket check (pool)
   Tap to complete
```

Wait — water plants was just done yesterday (Apr 25). Why is it due again today?

Maria taps "Water plants" → task detail:
```
Water plants
Zone: backyard
Due: Today (Apr 26)
Routine: Every 3 days (last: Apr 23)
```

OH — the AI-suggested task on Apr 25 was a ONE-TIME task, not part of the routine. The routine's normal cadence (every 3 days from Apr 23) still triggered today (Apr 26).

Maria realizes plants don't need water two days in a row. Taps task → taps "Skip" (not "Complete") → adds reason: "Already watered yesterday via suggestion."

Task status → skipped. Routine timer resets to Apr 29 (3 days from Apr 26).

**10:15am** — Maria checks pool basket. Returns to app → taps "Pool basket check" → taps "Mark Complete" → no photo, no note (quick task).

Today tab clears.

### Sunday, Apr 27 — No Tasks, Property Review

**7:00am** — No tasks due. No notification.

**11:00am** — Maria opens app, browses Property tab → taps "Pool" zone:
```
Pool

Assets:
• Pool pump (Hayward SP2610X15)
• Filter (installed Jan 10)

Active Routines:
• Chemical check (every 3 days)
   Last: Apr 23, next: Apr 26  ← BUG: should be Apr 29!
• Basket clean (every 7 days)
   Last: Apr 26, next: May 3

Recent History:
• Apr 26: Basket clean ✓
• Apr 23: Chemical check ✓
• Apr 19: Basket clean ✓

[Report Incident]
```

Wait — pool chemical check says "next: Apr 26" but today is Apr 27. That's stale data.

*BUG DISCOVERED:* Property tab projection is not live-updated. Needs refresh on tab focus.

*ARCHITECTURAL NOTE:* v0 will accept this UX flaw (manual refresh). v1 adds SSE live updates.

Maria pulls down to refresh (iOS gesture) → Property tab re-queries backend → "next: Apr 29" now correct.

### Week Summary

**Tasks completed:** 6 (lawn mowing x1, pool basket x2, pool chemical x1, plant watering x1, repair run x4 steps)  
**Tasks skipped:** 1 (plant watering duplicate)  
**Incidents handled:** 1 (garden gate hinge, start to finish in 3 days)  
**Evidence captured:** 5 photos, 4 notes  
**App opens:** ~12 (1–2 per day, avg 30 seconds per session)  
**Bugs discovered:** 2 (multi-zone task model, stale projection data)

**User sentiment:** Positive. Maria feels confident she's not forgetting maintenance. Evidence trail gives peace of mind. Repair run structure helped her not lose track of gate fix across multiple days.

---

## 9. Why This Validates GERT v2

gert-domain-home exercises GERT's core primitives in ways that pure enterprise domains (approval workflows, deployment pipelines) cannot. Here's what it proves:

### 1. Recurring Orchestration

**What Home domain tests:**  
Routines are the simplest valid use case for durable runs with timer-backed wakeups. No complex branching, no external dependencies, no approval gates — just "wake up every N days, create human task, wait for completion, reset timer."

**GERT validation:**
- Timer primitive must support variable intervals (3 days, 7 days, 14 days)
- Timer reset logic must correctly calculate next_wakeup from last_completion_time (not from last_wakeup_time, to avoid drift)
- Human task creation at wakeup must be deterministic (same routine → same task structure every time)
- Run must remain active indefinitely (never complete, always yield after task completion)

**What breaks if this fails:**  
Routines drift out of sync (e.g., "every 7 days" becomes 8 days due to rounding error). Owner loses trust in app.

### 2. Long-Running Runs

**What Home domain tests:**  
Repair runs can span days or weeks. Step 1 (diagnose) completes Monday, step 2 (buy parts) completes Wednesday, step 3 (fix) completes Saturday. Run must persist correctly across 5 days, 3 app sessions, multiple device reboots.

**GERT validation:**
- Run state persists across process restarts
- Step completion is atomic (evidence + status update happen together or not at all)
- Partial completion is a valid stable state (run at step 2/4 can sit idle for weeks)
- No automatic timeouts or cleanup (repair run waits indefinitely for user action)

**What breaks if this fails:**  
User completes step 2, closes app, returns next day → step 2 shows incomplete, evidence lost. User abandons app.

### 3. Projections

**What Home domain tests:**  
Two critical projections:
1. **Today tab** — filter all property tasks to executor_id=current_user, due_date=today, status IN (pending, in_progress)
2. **Away mode** — filter to delegate-scoped tasks within delegation time window

Both must query in <100ms (mobile refresh latency). Both must update immediately when task status changes (SSE or polling).

**GERT validation:**
- Projection query performance at 100+ tasks (typical property has 50–100 routines over 1 year)
- Projection correctness (filtering logic must match policy routing logic)
- Projection update latency (if task completed, Today tab refresh must show updated state within 1 second)

**What breaks if this fails:**  
Today tab shows stale data (task marked complete, still appears in list). Delegate sees owner's tasks (privacy breach). User loses trust.

### 4. Delegation

**What Home domain tests:**  
Time-bounded role transfer. Policy must route tasks to delegate_id only during delegation window. Policy must expire automatically at end_date without manual deactivation.

**GERT validation:**
- Policy evaluation at human task creation (not at run creation, because run is created before delegation starts)
- Time-based policy conditions (active_start <= now <= active_end)
- Task-name-based filtering (scoped_tasks list)
- Policy composition (if multiple delegations overlap, which wins? Home domain assumes no overlap, but GERT must define precedence.)

**What breaks if this fails:**  
Delegate continues to receive tasks after return date (owner gets angry). OR delegate never receives tasks during delegation (owner's trip is ruined).

### 5. Timers

**What Home domain tests:**  
Seasonal cadence rules (deferred to v1) require timer policy to evaluate external data (current date, season) at wakeup time. This tests timer flexibility beyond fixed intervals.

**GERT validation:**
- Timer policy can access runtime context (current date, timezone)
- Timer policy can call external functions (season determination logic)
- Timer policy evaluation happens at wakeup time, not at timer creation time (because season changes over time)

**What breaks if this fails:**  
Seasonal rules evaluate to wrong season (lawn mowing wakes every 7 days in winter when it should be 21 days). Grass gets overgrown or mowed wastefully.

**v0 note:** v0 only tests simple fixed-interval timers. Seasonal rules validate GERT v2.1 (OPA integration required).

### 6. Reactive Incidents

**What Home domain tests:**  
Ad-hoc runs with no predefined schedule. Incident is user-reported → repair run starts immediately, no timer.

**GERT validation:**
- Run creation API must support manual triggering (not just timer-triggered)
- Run must start in active state (not paused waiting for timer)
- Sub-run spawning (repair run is child of incident run)
- Parent-child lifecycle (incident run completes when repair run completes)

**What breaks if this fails:**  
User reports incident, nothing happens (repair run never starts). OR incident completes prematurely while repair run still active.

### 7. Domain Kits

**What Home domain tests:**  
The entire domain kit model: thin authoring DSL (YAML routines) compiles to GERT primitives, with no runtime re-implementation.

**GERT validation:**
- Compiler can generate ExecutionPlan from domain YAML (routine.yaml → GERT ExecutionPlan with timer + human task)
- GERT runtime has no knowledge of "routine" or "zone" concepts (they exist only in domain layer)
- Domain layer can query GERT run state and reinterpret it as domain concepts (GERT run status → routine next_due_date)

**What breaks if this fails:**  
Domain kit compiler generates invalid ExecutionPlan (GERT runtime rejects it). OR compiler generates valid plan but domain semantics are lost (e.g., routine cadence ignored).

---

## 10. Recommended v0 Scope

v0 is a **proof-of-concept prototype**, not a production-ready app. Goal: validate the domain kit model and core GERT primitives in 4–6 weeks of development.

### What to Build (Minimum Useful Set)

#### Data Model (Backend)
- **Property** entity with zones (hardcoded: pool, front_lawn, backyard, garage)
- **Assets** (optional, just name + zone_id, no complex tracking)
- **Routines** (2–3 examples: pool_basket_check every 7d, pool_chemical_check every 3d, lawn_mowing every 7d)
- **Incidents** (user-reported, spawns repair run)

#### GERT Primitives Used
- Timer-backed durable run (for routines)
- Human task (for maintenance tasks)
- Evidence (photo + note, no GPS)
- Ad-hoc run (for incidents)
- Sub-run (repair run as child of incident)

#### Mobile App (MVP)
- **Today tab only** (no Property/Tasks/History tabs in v0)
- Show tasks due today for current user
- Tap task → mark complete → attach photo OR note → submit
- Push notification for tasks due today (7am daily)

#### Delegation (Simplified)
- Owner can set start_date + end_date + delegate_name + delegate_contact
- Delegate receives email/SMS with link to simplified Today view
- Delegate sees ONLY their tasks (no other tabs)
- No reminder notifications in v0 (defer to v1)
- Owner reviews evidence on return (manual, no summary screen in v0)

#### Authoring (Manual)
- Property config: `property.yaml` (manual edit)
- Routine definitions: `routines/*.yaml` (manual edit)
- Incident creation: mobile app UI (tap "Report Incident" in Today tab)

### What to Explicitly Defer (Out of Scope for v0)

#### Deferred to v1:
- **Seasonal cadence rules** (requires OPA integration, GERT v2.1)
- **AI hints** (requires policy evaluator + weather API)
- **Consumable tracking** (requires dynamic routine creation, not designed yet)
- **Property/Tasks/History tabs** (Today tab proves the model, other tabs are UI polish)
- **Reminder notifications** (requires timer with notification policy)
- **GPS evidence** (photo + note are sufficient for v0 validation)
- **Offline evidence upload queue** (assume always-online in v0)
- **Multi-zone tasks** (lawn_mowing as single task for front+back requires task model redesign)
- **Routine editing in mobile app** (v0 uses manual YAML editing)

#### Why Each Deferral:

**Seasonal rules:** Requires GERT v2.1 (OPA integration). v0 validates simple fixed-interval timers only. Seasonal logic can be designed in parallel, tested in v1.

**AI hints:** Adds implementation complexity (weather API, policy evaluator, hint rendering) without validating new GERT primitives. UX is nice-to-have, not architectural validation.

**Consumable tracking:** Requires dynamic routine creation (template-based run instantiation), which is not yet designed in GERT. Deferring this frees v0 from solving two hard problems at once.

**Property/Tasks/History tabs:** Today tab proves projection correctness, human task completion, evidence capture. Other tabs are read-side UX, not architectural validation.

**Reminder notifications:** Requires timer with notification emission (new GERT primitive not yet designed). Away mode works without reminders — delegate just checks app daily.

**GPS evidence:** Photo + note prove evidence primitive works. GPS adds mobile permission complexity, no architectural value.

**Offline queue:** Assume always-online in v0 simplifies mobile app (no local DB, no sync logic). Real users need this, but v0 is prototype for team use only.

**Multi-zone tasks:** Requires task model redesign (one task → multiple completions). Current GERT model is one human task = one completion. Deferring this avoids redesign during v0.

**Routine editing in app:** Mobile-first authoring is UX polish. v0 validates runtime, not authoring UX. YAML editing is acceptable for team prototype.

### v0 Success Criteria

v0 is successful if it proves these claims:

1. **Domain kit compilation works** — routine.yaml compiles to valid GERT ExecutionPlan, runtime executes it correctly
2. **Timer-backed runs work** — routines wake on cadence, create human tasks, reset timer on completion, never drift
3. **Delegation works** — time-bounded policy routes tasks to delegate, expires automatically, owner reviews evidence
4. **Evidence capture works** — photo/note attach to completed tasks, persist in GERT evidence log, retrievable via API
5. **Reactive incidents work** — user-reported incident spawns repair run, multi-step workflow completes over days
6. **Mobile app is usable** — non-technical user can complete daily tasks without help, <1 minute per task

**Validation method:**  
Ken (architect) and 1–2 non-technical users (family members) use the app daily for 14 days. Track:
- Task completion rate (target: >90%)
- Evidence capture rate (target: >70% of tasks have photo or note)
- Bug reports (target: <5 blocking bugs)
- User NPS (target: 7+/10)

**Timeline:**  
- Week 1–2: Backend (property/routine/incident models, GERT compiler)
- Week 3–4: Mobile app (Today tab, evidence capture, delegation UI)
- Week 5: Integration testing (routine cadence, delegation, repair run)
- Week 6: User validation (14-day daily use by 2 non-technical users)

**Deliverables:**
- `.squad/tmp/home-domain-compiler.go` (routine.yaml → GERT ExecutionPlan)
- `.squad/tmp/home-domain-api.md` (REST API spec for mobile app)
- `.squad/tmp/home-mobile-app/` (React Native or Flutter app, Today tab only)
- `.squad/tmp/home-v0-validation-report.md` (14-day usage results, bugs found, GERT primitives validated)

---

## Conclusion

gert-domain-home is the right first consumer domain for GERT because it exercises recurring, reactive, and delegation patterns in a real-world context with daily use. v0 validates the domain kit model (authoring DSL → GERT primitives, no runtime reimplementation) and tests GERT's timer, projection, policy, and evidence primitives under realistic load.

Success in v0 proves:
- GERT's primitives are general enough to support non-enterprise domains
- Domain kits can provide rich abstractions without forking the runtime
- Mobile companion apps can deliver calm, practical UX on top of durable-run orchestration

Failure modes (timer drift, stale projections, policy bugs) surface early in daily usage, guiding GERT v2.1 hardening before enterprise adoption.

The Home domain is not a toy. It's a serious validation of GERT's architectural claim: **one runtime, many domains, zero reimplementation.**
