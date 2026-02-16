---
phase: 12
plan: 02
name: Real-time Swarm Display
subsystem: visualization
tags: [swarm, display, real-time, caste-colors, chamber-activity]

dependency_graph:
  requires: [12-01]
  provides: [swarm-display, caste-colors, chamber-tracking]
  affects: [12-03, 12-04]

tech-stack:
  added: [picocolors]
  patterns: [ANSI colors, file watching, JSON state]

key-files:
  created:
    - bin/lib/caste-colors.js
    - .aether/utils/swarm-display.sh
  modified:
    - .claude/commands/ant/swarm.md
    - .aether/aether-utils.sh

decisions:
  - id: D12-02-001
    text: "Use both color AND emoji together for caste display (not replacing each other)"
    rationale: "PROJECT.md explicitly requires both for immersive experience"
  - id: D12-02-002
    text: "ANSI codes in bash must match picocolors in Node.js"
    rationale: "Single source of truth via caste-colors.js exports both formats"
  - id: D12-02-003
    text: "Chamber activity tracked via optional 8th parameter in swarm-display-update"
    rationale: "Backward compatible - existing calls work without chamber"
  - id: D12-02-004
    text: "Fix brace expansion bug in default JSON parameter"
    rationale: "Bash interprets {} as brace expansion, causing malformed JSON"

metrics:
  duration: 12m
  completed: 2026-02-14
---

# Phase 12 Plan 02: Real-time Swarm Display Summary

## One-Liner

Implemented `/ant:swarm` real-time display with caste colors, emojis, tool stats, trophallaxis metrics, and chamber activity map - delivering the core "3 foragers excavating..." immersive experience.

## What Was Built

### 1. Caste Color System (`bin/lib/caste-colors.js`)

Centralized caste styling module providing:

| Caste   | Color   | Emoji | ANSI Code | Use Case          |
|---------|---------|-------|-----------|-------------------|
| builder | blue    | 🔨    | \033[34m  | Construction      |
| watcher | green   | 👁️    | \033[32m  | Observation       |
| scout   | yellow  | 🔍    | \033[33m  | Exploration       |
| chaos   | red     | 🎲    | \033[31m  | Testing/Disruption|
| prime   | magenta | 👑    | \033[35m  | Coordination      |

Exports:
- `CASTE_STYLES` - Raw style definitions
- `getCasteStyle(caste)` - Case-insensitive lookup
- `formatAnt(name, caste)` - Returns "🔨 Builder" with color
- `formatAntAnsi(name, caste)` - ANSI codes for bash
- `getCastes()` - Array of all caste names

### 2. Real-time Display Script (`.aether/utils/swarm-display.sh`)

Live-updating swarm visualization with:

**Header:**
```
       .-.
      (o o)  AETHER COLONY
      | O |  Swarm Activity
       `-`
```

**Per-Ant Display:**
- Caste emoji + colored name
- Animated status phrases ("excavating...", "observing...")
- Tool usage: 📖5 🔍3 ✏️2 ⚡1
- Elapsed time: (2m3s)
- Trophallaxis: 🍯1250 tokens

**Chamber Activity Map (VIZ-07):**
```
Chamber Activity:
  🍄 fungus garden 🔥🔥 (2 ants)
```

**Update Mechanisms:**
- `fswatch` (macOS) - native file watching
- `inotifywait` (Linux) - native file watching
- Polling fallback (2s) - universal compatibility

### 3. Enhanced `/ant:swarm` Command

Two modes supported:

**Quick View Mode** (no arguments):
```bash
/ant:swarm
# Runs: bash .aether/utils/swarm-display.sh
# Shows real-time colony activity
```

**Bug Destruction Mode** (with arguments):
- Preserved existing 4-scout parallel investigation
- Added swarm-display-init/update calls throughout
- Tracks each scout spawn with caste/tools/tokens
- Updates status to "completed" when scouts finish

### 4. Chamber Activity Tracking

Extended `swarm-display-update` command:

```bash
# New 8th parameter: chamber
bash .aether/aether-utils.sh swarm-display-update \
  "ant-name" "builder" "excavating" "task" "Queen" \
  '{"read":1}' 100 "fungus_garden"
```

Features:
- Auto-increments chamber activity count
- Auto-decrements old chamber when ant moves
- Validates chamber exists before updating
- Returns chamber in JSON response

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed brace expansion in default JSON parameter**

- **Found during:** Task 4 testing
- **Issue:** `tools_json="${6:-{}}"` causes bash to expand `{}` as brace expansion, resulting in `'{} }'` (extra space and brace)
- **Fix:** Changed to `tools_json="${6:-}"` with explicit empty check
- **Files modified:** `.aether/aether-utils.sh`
- **Commit:** `06db022`

## Verification Results

| Check | Status | Evidence |
|-------|--------|----------|
| caste-colors.js exports | PASS | `formatAnt('Test','builder')` returns "🔨 Test" |
| Color alignment | PASS | Both files use `\033[34m` for builder blue |
| swarm.md Quick View | PASS | Section "Quick View Mode (No Arguments)" exists |
| Chamber activity | PASS | `fungus_garden.activity = 2` with 2 ants assigned |
| Tool stats format | PASS | 📖5 🔍3 ✏️2 ⚡1 pattern implemented |
| Trophallaxis | PASS | 🍯 token indicator in display |

## Files Changed

```
bin/lib/caste-colors.js                    | 57 +++++++++++++
.aether/utils/swarm-display.sh             | 215 ++++++++++++++++++++++++++++++
.claude/commands/ant/swarm.md              | 57 +++++++++
.aether/aether-utils.sh                    | 36 ++++++-
```

## Commits

1. `6b1f9c3` - feat(12-02): add caste-colors.js with color and emoji definitions
2. `190994f` - feat(12-02): add swarm-display.sh real-time rendering script
3. `470c8af` - feat(12-02): enhance ant:swarm command with real-time display
4. `06db022` - feat(12-02): add chamber activity tracking to swarm-display-update

## Next Phase Readiness

**Ready for Plan 12-03:** Tunnel View with collapsible nested spawns
- swarm-display.json has `parent` field for hierarchy
- render_swarm function can be extended for tree view
- Activity tracking infrastructure complete

**Ready for Plan 12-04:** Chamber Activity Map
- Chamber activity counts working (VIZ-07)
- Fire intensity display implemented
- Just needs chamber comparison feature

## Success Criteria Verification

- [x] caste-colors.js exists with builder=blue, watcher=green, scout=yellow, chaos=red, prime=magenta + emojis
- [x] swarm-display.sh renders real-time display with ANSI colors, caste emojis, tool stats, tokens, timing
- [x] swarm.md command has Quick View mode (real-time) and Bug Destruction mode (existing)
- [x] Activity tracking integrated into bug destruction flow
- [x] All caste color definitions are consistent between JS and bash
- [x] VIZ-07 chamber activity map shows active chambers with fire intensity based on ant count
