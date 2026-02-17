---
phase: 06-colony-lifecycle
plan: 02
subsystem: colony-lifecycle
tags: [entomb, archive, chamber, seal-first, eternal-memory, queen-promote]

# Dependency graph
requires:
  - phase: 06-colony-lifecycle
    provides: "CROWNED-ANTHILL.md from seal command (plan 06-01)"
provides:
  - "Seal-first entomb command with full archive and date-first naming"
  - "Eternal memory recording in ~/.aether/eternal/memory.json"
  - "State reset including instincts and learnings after promotion"
affects: [06-03-tunnels]

# Tech tracking
tech-stack:
  added: []
  patterns: ["seal-first enforcement gate", "date-first chamber naming YYYY-MM-DD-goal", "full colony data archive"]

key-files:
  created: []
  modified:
    - ".aether/commands/claude/entomb.md"
    - ".claude/commands/ant/entomb.md"
    - ".aether/commands/opencode/entomb.md"

key-decisions:
  - "Seal-first enforcement replaces old all-phases-complete + no-critical-errors + not-executing gates"
  - "Date-first naming (YYYY-MM-DD-goal) replaces old goal-timestamp format"
  - "Full archive copies all data files plus CROWNED-ANTHILL.md and dreams to chamber"
  - "State reset now clears memory.instincts, memory.phase_learnings, memory.decisions (promoted to QUEEN.md)"

patterns-established:
  - "Seal-first: entomb refuses if milestone != Crowned Anthill"
  - "Belt-and-suspenders: check both milestone field and CROWNED-ANTHILL.md file existence"

requirements-completed: [LIF-02]

# Metrics
duration: 5min
completed: 2026-02-18
---

# Phase 06 Plan 02: Rewrite /ant:entomb with Seal-First Enforcement Summary

**Full-archive entomb command with seal-first gate, date-first chamber naming, eternal memory, and complete state reset**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-18T00:03:00Z
- **Completed:** 2026-02-18T00:08:00Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Rewrote entomb to enforce seal-first (must be Crowned Anthill milestone)
- Added full archive of all colony data files plus CROWNED-ANTHILL.md and dreams
- Implemented date-first chamber naming (YYYY-MM-DD-sanitized-goal)
- Added eternal memory recording via eternal-init + jq append
- State reset now clears instincts, phase_learnings, and decisions (already promoted to QUEEN.md)
- Synced all three command locations with platform-specific differences

## Task Commits

Each task was committed atomically:

1. **Task 1: Rewrite entomb.md** — `0e7d44e` (feat)
2. **Task 2: Sync to Claude Code and OpenCode** — included in `0e7d44e`

## Files Created/Modified
- `.aether/commands/claude/entomb.md` — Rewritten with seal-first, full archive, date-first naming
- `.claude/commands/ant/entomb.md` — Synced copy (identical to source of truth)
- `.aether/commands/opencode/entomb.md` — Synced with normalize-args and swarm-display-render

## Decisions Made
- Seal-first enforcement is cleaner than the old 3-precondition gate (all-phases, not-executing, no-critical-errors)
- Belt-and-suspenders check: verify both milestone field AND CROWNED-ANTHILL.md file existence
- Archives ALL data files including pheromones.json, session.json, constraints.json, timing.log, view-state.json
- Excludes: backups/, locks/, midden/, survey/ directories

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Entomb command ready to work with seal's CROWNED-ANTHILL.md
- Chamber archives now include seal documents for tunnels to display
- Plan 06-03 (tunnels) can reference CROWNED-ANTHILL.md in chambers

---
*Phase: 06-colony-lifecycle*
*Completed: 2026-02-18*
