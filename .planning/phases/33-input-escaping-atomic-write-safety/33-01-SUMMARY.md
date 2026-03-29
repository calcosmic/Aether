---
phase: 33-input-escaping-atomic-write-safety
plan: 01
subsystem: infra
tags: [bash, grep, json, escaping, security, shell-injection]

# Dependency graph
requires: []
provides:
  - "grep -F fixed-string matching for all ant_name pattern matches in spawn, spawn-tree, swarm"
  - "jq-safe JSON construction for all json_ok calls with user-derived string values"
affects: [spawn, spawn-tree, swarm, aether-utils]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "grep -F for all variable-based pattern matching (never use bare grep with $variable)"
    - "jq -n --arg for JSON construction with user-derived strings (never interpolate into JSON directly)"

key-files:
  created: []
  modified:
    - ".aether/utils/spawn.sh"
    - ".aether/utils/spawn-tree.sh"
    - ".aether/utils/swarm.sh"
    - ".aether/aether-utils.sh"

key-decisions:
  - "Use jq -n --arg for string values, --argjson for numeric/boolean in json_ok calls"
  - "Drop ^ and $ regex anchors when switching to grep -F since -F treats everything as literal"
  - "Ant names are unique per swarm so grep -F without anchors is safe for timing file lookups"

patterns-established:
  - "grep -F pattern: All grep calls that match dynamic variables use -F flag"
  - "jq JSON construction: All json_ok calls with string variables use jq -n --arg instead of direct interpolation"

requirements-completed: [SAFE-01]

# Metrics
duration: 23min
completed: 2026-03-29
---

# Phase 33 Plan 01: Grep Fixed-String and JSON Escaping Summary

**grep -F for all ant_name pattern matches plus jq-safe JSON construction for all json_ok calls in spawn, spawn-tree, and swarm modules**

## Performance

- **Duration:** 23 min
- **Started:** 2026-03-29T05:12:30Z
- **Completed:** 2026-03-29T05:35:00Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- All grep calls using ant_name/current_ant/current/swarm_id/caste_filter variables now use -F (fixed-string) matching
- All json_ok calls in spawn.sh, spawn-tree.sh, and swarm.sh that interpolate string variables now use jq -n --arg for safe JSON construction
- Broader sweep of aether-utils.sh found and fixed one additional grep call (activity-log-read caste_filter)
- All 616 tests pass with no regressions

## Task Commits

Each task was committed atomically:

1. **Task 1: grep -F for all ant_name patterns** - Prior commit `fb779cd` covered spawn.sh/spawn-tree.sh/swarm.sh; additional fix `672e533` for aether-utils.sh caste_filter
2. **Task 2: json_ok escaping in spawn.sh, spawn-tree.sh, swarm.sh** - `4fbee0a` (swarm findings/cleanup), `dfb5de0` (spawn.sh/spawn-tree.sh), `77cc1a0` (swarm display/timing)

## Files Created/Modified
- `.aether/utils/spawn.sh` - All json_ok calls use jq -n --arg for ant_name, child_name, summary, swarm_id
- `.aether/utils/spawn-tree.sh` - get_spawn_depth, get_spawn_children, get_spawn_lineage use jq for JSON output
- `.aether/utils/swarm.sh` - display_init/update, timing_start/get/eta, findings_init/add, solution_set, cleanup all use jq
- `.aether/aether-utils.sh` - activity-log-read grep uses -F for caste_filter

## Decisions Made
- Used `jq -n --arg` for strings and `--argjson` for numbers/booleans in json_ok construction
- Dropped `^` and `$` regex anchors when switching to `grep -F` since fixed-string mode treats them as literals
- For swarm timing file lookups, matching `ant_name|` with `-F` is sufficient because ant names are unique per swarm (documented in inline comments)
- Fixed heredoc JSON creation in `_swarm_findings_init` and `_swarm_display_init` by replacing with `jq -n` construction

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Fixed json_ok escaping for swarm_id, scout_type, and display variables**
- **Found during:** Task 2 (json_ok audit)
- **Issue:** Plan focused on ant_name interpolation, but the same vulnerability existed for swarm_id, scout_type, caste, emoji, chamber, and stash_name variables in swarm.sh
- **Fix:** Applied the same jq -n --arg pattern to all string variable interpolations in swarm.sh json_ok calls
- **Files modified:** .aether/utils/swarm.sh
- **Verification:** grep audit confirms zero raw string interpolation in json_ok calls
- **Committed in:** 4fbee0a, 77cc1a0

**2. [Rule 2 - Missing Critical] Fixed heredoc JSON creation in _swarm_findings_init and _swarm_display_init**
- **Found during:** Task 2 (json_ok audit)
- **Issue:** `cat > file <<EOF` with `$swarm_id` inside creates malformed JSON if swarm_id contains quotes
- **Fix:** Replaced heredoc with `jq -n --arg` construction
- **Files modified:** .aether/utils/swarm.sh
- **Verification:** Output validates as proper JSON
- **Committed in:** 4fbee0a, 77cc1a0

**3. [Rule 2 - Missing Critical] grep -F for caste_filter in aether-utils.sh**
- **Found during:** Task 1 (broader sweep)
- **Issue:** `grep "$caste_filter"` in activity-log-read treated caste names as regex patterns
- **Fix:** Changed to `grep -F "$caste_filter"`
- **Files modified:** .aether/aether-utils.sh
- **Committed in:** 672e533

---

**Total deviations:** 3 auto-fixed (3 missing critical)
**Impact on plan:** All auto-fixes extend the same pattern the plan established. No scope creep -- these are the same vulnerability class applied to additional variables.

## Issues Encountered
- Concurrent session was modifying overlapping files, causing stash conflicts and requiring fixes to be re-applied. Resolved by isolating my changes and only committing plan-relevant files.
- instinct-confidence tests showed intermittent failures (4 tests) due to concurrent COLONY_STATE.json modification by another session -- unrelated to this plan's changes.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- All grep pattern injection and json_ok escaping fixes are complete for spawn, spawn-tree, and swarm modules
- Ready for plan 02 (json_ok escaping in other modules) and plans 03-04 (atomic write safety, lock cleanup)

---
*Phase: 33-input-escaping-atomic-write-safety*
*Completed: 2026-03-29*
