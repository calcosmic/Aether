---
phase: 31-architecture-evolution
plan: 01
subsystem: learning
tags: [bash, jq, global-learnings, two-tier, promotion-ux]

# Dependency graph
requires:
  - phase: 26-auto-learning
    provides: Phase learnings extraction and memory.json schema
  - phase: 30-automation
    provides: Tech debt report in continue.md Step 2.5
provides:
  - learning-promote subcommand in aether-utils.sh for global learnings management
  - learning-inject subcommand in aether-utils.sh for tag-based filtering
  - Learning promotion UX in continue.md Step 2.5b at project completion
affects: [31-02 (learning injection in colonize.md), 31-03 (spawn tree engine)]

# Tech tracking
tech-stack:
  added: []
  patterns: [two-tier-learning-system, global-file-at-home-directory]

key-files:
  created: []
  modified:
    - .aether/aether-utils.sh
    - .claude/commands/ant/continue.md

key-decisions:
  - "Forced curation overflow strategy when 50-entry cap reached (display all, user removes one)"
  - "Auto-continue mode skips promotion entirely with informative message"
  - "Tags inferred from colonization decision in memory.json at promotion time"

patterns-established:
  - "Global state in ~/.aether/ directory (cross-project persistence)"
  - "Subcommand-based global file management (learning-promote/learning-inject)"

# Metrics
duration: 2min
completed: 2026-02-05
---

# Phase 31 Plan 01: Global Learnings Infrastructure Summary

**learning-promote and learning-inject subcommands in aether-utils.sh with interactive promotion UX in continue.md Step 2.5b**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-05T15:04:23Z
- **Completed:** 2026-02-05T15:06:12Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Added `learning-promote` subcommand that creates ~/.aether/ directory and learnings.json on first use, appends entries with full schema (id, content, source_project, source_phase, tags, promoted_at), enforces 50-entry cap
- Added `learning-inject` subcommand that filters global learnings by tag keyword match (case-insensitive, CSV input), returns empty gracefully when file doesn't exist
- Added Step 2.5b in continue.md for interactive learning promotion at project completion with categorized display (candidates vs project-specific), cap enforcement with curation flow, and auto-continue guard

## Task Commits

Each task was committed atomically:

1. **Task 1: Add learning-promote and learning-inject subcommands to aether-utils.sh** - `1f7eb21` (feat)
2. **Task 2: Add learning promotion UX to continue.md Step 2.5** - `f37fd48` (feat)

## Files Created/Modified
- `.aether/aether-utils.sh` - Added learning-promote (creates ~/.aether/learnings.json, appends entries, enforces 50-cap) and learning-inject (filters by tag keyword match) subcommands; updated help command
- `.claude/commands/ant/continue.md` - Added Step 2.5b (promote learnings to global tier with categorized display, user selection, cap enforcement) and Step 2.5c (updated completion message with global learnings reference)

## Decisions Made
- Forced curation overflow strategy: when 50-entry cap is reached, display all existing learnings and ask user to remove one before adding new -- preserves highest-value learnings
- Auto-continue mode (--all) skips promotion entirely with "Run /ant:continue (without --all) to promote learnings" message -- prevents blocking on interactive input
- Tags inferred from colonization decision in memory.json at promotion time -- reuses existing project context data

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Global learnings file infrastructure complete, ready for Plan 02 (learning injection in colonize.md)
- learning-inject subcommand ready for use by colonize.md Step 5.5
- learning-promote subcommand ready for use by continue.md Step 2.5b

---
*Phase: 31-architecture-evolution*
*Completed: 2026-02-05*
