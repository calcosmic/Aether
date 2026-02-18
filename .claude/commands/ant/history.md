---
name: ant:history
description: "📜🐜📜 Show colony event history"
---

You are the **Queen**. Show colony history.

## Instructions

### Step 1: Read State

Read `.aether/data/COLONY_STATE.json`.

If file missing or `goal: null`:
```
No colony initialized. Run /ant:init first.
```
Stop here.

### Step 2: Parse Events

Extract the `events` array from the state.

Each event is in format: `timestamp|type|source|description`

Parse each event into components:
- timestamp: before first `|`
- type: between first and second `|`
- source: between second and third `|`
- description: after third `|`

### Step 3: Parse Filter Arguments

Parse optional filter arguments:

**Event Type Filter:**
- `--type TYPE` or `-t TYPE`: Filter by event type
- Multiple types can be specified (comma-separated)
- Valid types: `colony_initialized`, `phase_started`, `phase_advanced`, `plan_generated`, `milestone_reached`, `state_upgraded`

**Date Range Filter:**
- `--since DATE`: Show events since DATE
  - ISO format: `2026-02-13`, `2026-02-13T14:30:00`
  - Relative: `1d` (1 day ago), `2h` (2 hours ago), `30m` (30 minutes ago)
- `--until DATE`: Show events until DATE
  - Same format options as `--since`

### Step 4: Apply Limit

Parse optional `--limit N` argument (default: 10).

If `--limit` is provided, only show N events.

### Step 5: Filter Events

Apply filters to the events array:

**Type Filtering:**
- If `--type` is provided, split by comma to get list of types
- Filter events to only include those where type matches any of the specified types (case-insensitive)

**Date Range Filtering:**
- For `--since`: Parse the date value
  - If relative (e.g., "1d", "2h", "30m"), calculate timestamp by subtracting from current time
  - If ISO format, parse directly
  - Filter events with timestamp >= since date
- For `--until`: Same parsing logic
  - Filter events with timestamp <= until date

**Track Active Filters:**
- Record which filters are active for display indicator

### Step 6: Sort and Display

Sort events in reverse chronological order (most recent first).

If events array is empty:
```
No colony events recorded yet.
```

**Filter Indicators:**
- If any filters are active, show at the top:
  - Type filter: `Filters: type=<TYPE>[,<TYPE>...]`
  - Since filter: `Filters: since=<DATE>`
  - Until filter: `Filters: until=<DATE>`
  - Multiple filters: Combine on same line (e.g., `Filters: type=phase_started,phase_advanced since=1d`)
- Show filtered count: "Showing X of Y events (filtered from Z total)"

Otherwise, display in format:
```
━━━ Colony History (most recent first) ━━━

[TIMESTAMP] [TYPE] from [SOURCE]
  Description: [description]

[TIMESTAMP] [TYPE] from [SOURCE]
  Description: [description]

... (up to limit)
```

**Format:**
- Timestamp: Show in readable format (e.g., "2026-02-13 14:30:00")
- Type: Uppercase event type
- Source: Italics for source
- Description: Plain text

**Event type icons:**
- `colony_initialized`: 🏠
- `phase_started`: 🚀
- `phase_advanced`: ➡️
- `plan_generated`: 📋
- `milestone_reached`: 🏆
- `state_upgraded`: 🔄
- `default`: 📌

**Limit display:**
- If filtering is active, show: "Showing X of Y events (filtered from Z total)"
- If only limit is applied (no filters), show: "Showing X of Y events"

### Step 7: Display Summary

Show total event count at the end:
```
Total events recorded: <count>
```

### Step 8: Next Up

Generate the state-based Next Up block by running using the Bash tool with description "Generating Next Up suggestions...":
```bash
state=$(jq -r '.state // "IDLE"' .aether/data/COLONY_STATE.json)
current_phase=$(jq -r '.current_phase // 0' .aether/data/COLONY_STATE.json)
total_phases=$(jq -r '.plan.phases | length' .aether/data/COLONY_STATE.json)
bash .aether/aether-utils.sh print-next-up "$state" "$current_phase" "$total_phases"
```
