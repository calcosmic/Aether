---
phase: 11-visual-identity
plan: "04"
subsystem: ui
tags: [commands, banners, visual-language, slash-commands, aether]

requires:
  - phase: 11-03
    provides: Banner and Next Up standardization pattern established for high-complexity commands

provides:
  - Standardized ━━━━ heavy horizontal banners in 10 medium-complexity and special worker commands
  - State-routed print-next-up bash calls at completion of chaos, archaeology, dream, flags, flag, phase, oracle, watch, swarm, colonize

affects:
  - Any future commands added to .claude/commands/ant/
  - Aether visual identity documentation

tech-stack:
  added: []
  patterns:
    - "All command banners use ━━━━ (U+2501) heavy horizontal format with centered emoji+title"
    - "All command completions emit state-based Next Up via print-next-up helper"

key-files:
  created: []
  modified:
    - ".claude/commands/ant/phase.md"
    - ".claude/commands/ant/oracle.md"
    - ".claude/commands/ant/watch.md"
    - ".claude/commands/ant/swarm.md"
    - ".claude/commands/ant/colonize.md"
    - ".claude/commands/ant/chaos.md"
    - ".claude/commands/ant/archaeology.md"
    - ".claude/commands/ant/dream.md"
    - ".claude/commands/ant/flags.md"
    - ".claude/commands/ant/flag.md"

key-decisions:
  - "flag.md uses three separate banner blocks (blocker/issue/note variants) — each replaced individually with ━━━━ format"
  - "flags.md and flag.md lack a log-activity step — print-next-up placed at end of Step 4 display output, before Quick Actions / Flag Lifecycle sections"
  - "lint:sync count mismatch (34 Claude Code vs 33 OpenCode) is pre-existing from before Phase 11; not caused by these changes"

patterns-established:
  - "Banner format: ━━━━ line, emoji+spaced-title, ━━━━ line (50 chars each)"
  - "Next Up pattern: jq reads COLONY_STATE.json for state/current_phase/total_phases, passes to print-next-up"

requirements-completed: []

duration: 25min
completed: 2026-02-18
---

# Phase 11 Plan 04: Visual Identity — Medium and Special Worker Commands Summary

**Standardized ━━━━ banners and state-routed Next Up blocks across 10 medium-complexity and special worker commands, completing the visual identity pass for all Claude Code ant commands**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-02-18T00:00:00Z (continuation from previous context)
- **Completed:** 2026-02-18
- **Tasks:** 2
- **Files modified:** 10

## Accomplishments

- Replaced all legacy `═══` box-drawing equals banners with `━━━━` heavy horizontal format across 10 commands
- Added state-based `print-next-up` bash calls at completion of every command so users always see contextual next steps
- Task 1 (phase, oracle, watch, swarm, colonize) and Task 2 (chaos, archaeology, dream, flags, flag) each committed atomically

## Task Commits

Each task was committed atomically:

1. **Task 1: Medium-complexity commands (phase, oracle, watch, swarm, colonize)** - `df2badb` (feat)
2. **Task 2: Special worker commands (chaos, archaeology, dream, flags, flag)** - `66fdc9a` (feat)

**Plan metadata:** (docs commit — see final_commit step)

## Files Created/Modified

- `.claude/commands/ant/phase.md` - Replaced 2x ═══ banners, replaced hardcoded Next Steps with print-next-up call
- `.claude/commands/ant/oracle.md` - Replaced 2x ═══ banners, added print-next-up before final TMUX_FAIL stop
- `.claude/commands/ant/watch.md` - Added ━━━━ header banner, added print-next-up before Status Update Protocol
- `.claude/commands/ant/swarm.md` - Replaced 2x ═══ banners (SWARM DEPLOYED + SOLUTION RANKING), replaced hardcoded Next steps with print-next-up
- `.claude/commands/ant/colonize.md` - Replaced 2x ═══ banners (COLONIZE + TERRITORY SURVEY COMPLETE), added print-next-up
- `.claude/commands/ant/chaos.md` - Replaced 2x ═══ banners (RESILIENCE TESTER ACTIVE + CHAOS REPORT), added print-next-up
- `.claude/commands/ant/archaeology.md` - Replaced 2x ═══ banners (ARCHAEOLOGIST AWAKENS + ARCHAEOLOGY REPORT), added print-next-up
- `.claude/commands/ant/dream.md` - Replaced 2x ═══ banners (DREAMER AWAKENS + DREAM COMPLETE), added print-next-up
- `.claude/commands/ant/flags.md` - Replaced 1x ═══ banner (PROJECT FLAGS), added print-next-up after display block
- `.claude/commands/ant/flag.md` - Replaced 3x ═══ banner variants (BLOCKER/ISSUE/NOTE CREATED), added print-next-up

## Decisions Made

- For `flag.md`, which has three banner variants (blocker/issue/note), each was replaced individually to maintain the type-specific structure — the print-next-up is placed once after all three output blocks at the end of Step 4
- For `flags.md` and `flag.md`, which lack a "log activity" step, the print-next-up was placed at the end of the Step 4 display section before Quick Actions / Flag Lifecycle documentation
- `oracle.md` has multiple "Stop here" points (TMUX_FAIL, auth gates) — print-next-up added before the final TMUX_FAIL stop which is the natural command completion path

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

- Pre-existing lint:sync mismatch (34 Claude Code commands vs 33 OpenCode commands) reported as warning during commits. This is documented in the 11-03 SUMMARY as a pre-existing issue unrelated to Phase 11 changes. Commits succeed with warning.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Phase 11 visual identity standardization is now complete across all four plans (11-01 through 11-04)
- All Claude Code ant commands now use consistent ━━━━ banner format and state-based Next Up blocks
- Ready for any follow-on visual identity work or other phases

---
*Phase: 11-visual-identity*
*Completed: 2026-02-18*
