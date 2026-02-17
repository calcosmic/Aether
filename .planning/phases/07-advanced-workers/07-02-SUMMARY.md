---
phase: 07-advanced-workers
plan: 02
subsystem: commands
tags: [chaos, archaeology, swarm-display, sync-drift]

requires:
  - phase: 03-visual-experience
    provides: "swarm-display-inline and swarm-display-text functions"
provides:
  - "Chaos Claude Code copy synced to match source of truth"
  - "Archaeology Claude Code copy synced to match source of truth"
  - "swarm-display-text vs swarm-display-inline drift resolved"
affects: [09-polish-verify]

tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified:
    - ".claude/commands/ant/chaos.md"
    - ".claude/commands/ant/archaeology.md"

key-decisions:
  - "Claude Code copies must be identical to SoT (swarm-display-inline)"
  - "OpenCode copies correctly use swarm-display-render (not swarm-display-text or swarm-display-inline)"

patterns-established:
  - "Display function convention: SoT/Claude Code use swarm-display-inline, OpenCode uses swarm-display-render"

requirements-completed: [ADV-02, ADV-03]

duration: 8min
completed: 2026-02-18
---

# Plan 07-02: Chaos and Archaeology Sync Drift Fix Summary

**Fixed swarm-display-text vs swarm-display-inline sync drift in Claude Code copies of chaos.md and archaeology.md**

## Performance

- **Duration:** 8 min
- **Completed:** 2026-02-18
- **Tasks:** 2 (chaos sync + archaeology sync)
- **Files modified:** 2

## Accomplishments
- Fixed chaos.md Claude Code copy: replaced `swarm-display-text` with `swarm-display-inline` at line 228
- Fixed archaeology.md Claude Code copy: replaced `swarm-display-text` with `swarm-display-inline` at line 214
- Verified both Claude Code copies now match source of truth exactly
- Verified OpenCode copies have correct adaptations (normalize-args + swarm-display-render)
- Discovered swarm-display-render is a valid function in aether-utils.sh (line 2418), distinct from both swarm-display-inline and swarm-display-text

## Task Commits

1. **Task 1+2: Sync chaos and archaeology** - `782db41` (fix)

## Files Created/Modified
- `.claude/commands/ant/chaos.md` - Fixed swarm-display-text -> swarm-display-inline
- `.claude/commands/ant/archaeology.md` - Fixed swarm-display-text -> swarm-display-inline

## Decisions Made
- Display function convention clarified: SoT and Claude Code use `swarm-display-inline`, OpenCode uses `swarm-display-render` (a different function entirely, not a typo)

## Deviations from Plan
None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Chaos and archaeology commands verified and ready for Phase 9 end-to-end testing

---
*Phase: 07-advanced-workers*
*Completed: 2026-02-18*
