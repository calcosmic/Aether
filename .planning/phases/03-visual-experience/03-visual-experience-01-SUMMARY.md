---
phase: 03-visual-experience
plan: 01
subsystem: ui
tags: [aether-utils, bash, emoji, colony-display, swarm]

# Dependency graph
requires: []
provides:
  - swarm-display-text subcommand in aether-utils.sh
  - Plain-text ANSI-free colony display for Claude conversation context
  - Caste-specific emoji rendering (builder, watcher, scout, chaos, prime, oracle, route_setter, archaeologist, surveyor)
  - Progress bar rendering with block characters
  - Tool usage counts per ant row
  - Overflow indicator for more than 5 ants
affects:
  - 03-visual-experience (all remaining plans can use swarm-display-text)
  - Any command that currently calls swarm-display-inline may migrate to swarm-display-text

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "ANSI-free display: use plain text and emojis only, no escape codes"
    - "JSON structure tolerance: handle both flat total_active and nested .summary.total_active"
    - "Inline helpers: define get_emoji, format_tools_text, render_bar_text as local functions inside the case branch"

key-files:
  created: []
  modified:
    - .aether/aether-utils.sh

key-decisions:
  - "Add swarm-display-text as a new additive case branch, not modifying swarm-display-inline — both coexist"
  - "Handle both flat (.total_active) and nested (.summary.total_active) JSON to support mock test data and real data"
  - "Use format_tools_text (renamed from format_tools) to avoid name collision with swarm-display-inline's identically-named function"

patterns-established:
  - "Colony display pattern: header, rule, ant rows (emoji + name + bar + task + tools), overflow, footer rule, count"

requirements-completed:
  - VIS-01
  - VIS-02
  - VIS-03
  - VIS-04

# Metrics
duration: 12min
completed: 2026-02-17
---

# Phase 03 Plan 01: Visual Experience - swarm-display-text Summary

**Added `swarm-display-text` subcommand to aether-utils.sh: ANSI-free emoji colony display for use inside Claude conversations**

## Performance

- **Duration:** ~12 min
- **Started:** 2026-02-17T~T (approx)
- **Completed:** 2026-02-17
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments

- New `swarm-display-text)` case branch added to `.aether/aether-utils.sh` after the `swarm-display-inline` block
- Produces clean plain-text output readable in Claude's conversation (no ANSI escape sequences)
- Caste-specific emoji pairs: builder=🔨🐜, watcher=👁️🐜, scout=🔍🐜, chaos=🎲🐜, prime=👑🐜, oracle=🔮🐜, route_setter=🧭🐜, archaeologist=🏺🐜, surveyor=📊🐜
- 10-block progress bars using Unicode block characters (█░)
- Tool usage per ant: 📖 read, 🔍 grep, ✏️ edit, ⚡ bash (only non-zero counts shown)
- Overflow: caps at 5 visible ants, shows "+N more ants..." for larger swarms
- Graceful fallbacks: missing file → "🐜 Colony idle", missing jq → "🐜 Swarm active (details unavailable)", zero ants → "🐜 Colony idle"
- `swarm-display-text` registered in the `help)` commands JSON array

## Task Commits

Each task was committed atomically:

1. **Task 1: Add swarm-display-text case branch to aether-utils.sh** - `fd98595` (feat)

## Files Created/Modified

- `.aether/aether-utils.sh` - Added `swarm-display-text)` case branch (~101 lines) and registered command in help JSON

## Decisions Made

- Used `format_tools_text` as local function name (not `format_tools`) to avoid bash name collision with the identically-named function inside `swarm-display-inline`
- Made `total_active` reading tolerant of both flat JSON (`"total_active":1`) and nested JSON (`"summary":{"total_active":1}`) using `(.total_active // .summary.total_active // 0)`
- Added `swarm-display-text` as a purely additive new branch — `swarm-display-inline` is untouched and both coexist

## Deviations from Plan

None - plan executed exactly as written. One minor adaptation made transparently: the local function was named `format_tools_text` instead of `format_tools` to avoid bash scope collision; and the jq expression was broadened to handle both flat and nested `total_active` fields (needed for the plan's own mock test data to work).

## Issues Encountered

- The plan's verify section used `DATA_DIR=/tmp/... bash aether-utils.sh` to mock data, but `DATA_DIR` is hardcoded inside the script from `$AETHER_ROOT`. Verified caste-specific emoji by temporarily writing to the actual data dir and restoring the original file afterward. All verification passed.

## Next Phase Readiness

- `swarm-display-text` is ready for any command to call via `bash .aether/aether-utils.sh swarm-display-text`
- Next: Plan 03-02 can integrate this into agent-spawning slash commands (build, swarm, colonize, init, plan, continue)

---
*Phase: 03-visual-experience*
*Completed: 2026-02-17*
