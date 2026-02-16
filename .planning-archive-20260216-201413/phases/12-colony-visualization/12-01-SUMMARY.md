---
phase: 12
plan: 01
name: Activity Tracking Infrastructure
subsystem: visualization
tags: [telemetry, swarm, tracking, tools, tokens, timing]

dependency_graph:
  requires: [11-04]
  provides: [12-02, 12-03, 12-04, 12-05]
  affects: [VIZ-03, VIZ-04, VIZ-05]

tech_stack:
  added: []
  patterns: [atomic-writes, json-state, cumulative-metrics]

key_files:
  created:
    - .aether/data/swarm-display.json
  modified:
    - bin/lib/telemetry.js
    - .aether/aether-utils.sh

decisions:
  - tool_tracking_in_routing_decisions: Store per-tool counters in routing_decisions array for complete spawn history
  - cumulative_token_counting: Tokens accumulate per spawn for trophallaxis metrics
  - timing_log_simple_format: Use pipe-delimited log file for timing data (ant|timestamp|ISO) for efficiency
  - chamber_structure_in_display: Pre-define chambers (fungus_garden, nursery, etc.) with icons in swarm-display.json
  - atomic_writes_everywhere: All state writes use temp file + rename pattern

metrics:
  duration: "21 minutes"
  completed: "2026-02-14"
---

# Phase 12 Plan 01: Activity Tracking Infrastructure Summary

## One-Liner
Extended telemetry.js with tool/token tracking and added swarm activity/timing commands to aether-utils.sh, creating the data foundation for real-time colony visualization.

## What Was Built

### telemetry.js Extensions
- **`updateToolUsage(repoPath, spawnId, toolType, count)`**: Increment Read/Grep/Edit/Bash counters per spawn
- **`updateTokenUsage(repoPath, spawnId, tokens)`**: Cumulative token consumption tracking (trophallaxis metrics)
- **Enhanced `recordSpawnTelemetry`**: Now initializes `tools` object and `tokens` field for all new routing decisions

### aether-utils.sh Commands

**Activity Tracking:**
- `swarm-activity-log <ant_name> <action> <details>` - Append timestamped activity to log
- `swarm-display-init <swarm_id>` - Create swarm-display.json with chamber structure
- `swarm-display-update <ant_name> <caste> <status> <task> [parent] [tools] [tokens]` - Update ant activity, recalculate summary stats
- `swarm-display-get` - Retrieve current swarm display state as JSON

**Timing Utilities:**
- `swarm-timing-start <ant_name>` - Record start timestamp for duration tracking
- `swarm-timing-get <ant_name>` - Calculate elapsed seconds with MM:SS formatting
- `swarm-timing-eta <ant_name> <percent_complete>` - Calculate ETA based on progress percentage

### Data Structures

**swarm-display.json:**
```json
{
  "swarm_id": "...",
  "timestamp": "ISO",
  "active_ants": [...],
  "summary": {
    "total_active": N,
    "by_caste": {},
    "by_zone": {}
  },
  "chambers": {
    "fungus_garden": {"activity": 0, "icon": "🍄"},
    "nursery": {"activity": 0, "icon": "🥚"},
    "refuse_pile": {"activity": 0, "icon": "🗑️"},
    "throne_room": {"activity": 0, "icon": "👑"},
    "foraging_trail": {"activity": 0, "icon": "🌿"}
  }
}
```

## Verification Results

All success criteria met:
- [x] Tool usage tracking functions exist in telemetry.js and are exported
- [x] Token consumption tracking functions exist in telemetry.js and are exported
- [x] aether-utils.sh has swarm-display-init, swarm-display-update, swarm-display-get commands
- [x] aether-utils.sh has swarm-timing-start, swarm-timing-get, swarm-timing-eta commands
- [x] swarm-display.json created with proper structure including chambers, summary, active_ants
- [x] All commands return valid JSON and use atomic writes

## Deviations from Plan

None - plan executed exactly as written.

## Key Design Decisions

1. **Cumulative Token Tracking**: Tokens add up over time per spawn, enabling trophallaxis metrics that show total resource consumption
2. **Simple Timing Log Format**: Used pipe-delimited text file for timing data rather than JSON - more efficient for append-only operations
3. **Automatic Summary Recalculation**: swarm-display-update automatically recalculates total_active, by_caste, and by_zone counts
4. **Pre-defined Chambers**: Chambers structure includes ant-themed zones (fungus_garden, nursery, etc.) with emoji icons

## Next Phase Readiness

This plan provides the data foundation for:
- Plan 12-02: Real-time swarm display (`/ant:swarm` command)
- Plan 12-03: Tunnel view with collapsible nested spawns
- Plan 12-04: Chamber activity map
- Plan 12-05: ASCII art anthill maturity visualization

All visualization features can now consume real data from swarm-display.json and telemetry.json.

## Files Changed

| File | Changes |
|------|---------|
| `bin/lib/telemetry.js` | +75 lines: Added updateToolUsage, updateTokenUsage, enhanced recordSpawnTelemetry |
| `.aether/aether-utils.sh` | +224 lines: Added 7 new swarm/timing commands, updated help |
| `.aether/data/swarm-display.json` | Created dynamically by swarm-display-init |

## Commits

- `ec88460`: feat(12-01): add tool and token tracking to telemetry.js
- `bf5e86a`: feat(12-01): add swarm activity tracking and timing utilities
