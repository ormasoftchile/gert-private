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
