---
phase: 02-command-audit-data-tooling
plan: 02
subsystem: data
tags: [data-clean, colony-maintenance, artifact-removal, slash-command]

# Dependency graph
requires:
  - phase: 01-data-purge
    provides: "Clean baseline data files and knowledge of test artifact patterns"
provides:
  - "Repeatable /ant:data-clean command for ongoing artifact removal"
  - "data-clean subcommand in aether-utils.sh scanning 6 data file types"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Scan-then-confirm pattern: dry-run default, --confirm required for destructive changes"
    - "grep -c exit code handling: use `var=$(...) || var=0` instead of `var=$(... || echo 0)` to avoid multiline values"

key-files:
  created:
    - ".claude/commands/ant/data-clean.md"
  modified:
    - ".aether/aether-utils.sh"
    - ".claude/commands/ant/help.md"

key-decisions:
  - "Placed data-clean subcommand at end of case statement (before wildcard) for minimal diff and clear separation"
  - "Used atomic_write for file modifications when available, with direct write fallback"

patterns-established:
  - "Maintenance subcommand pattern: --dry-run default, --confirm for destructive ops, --json for machine output"

requirements-completed: [DATA-07]

# Metrics
duration: 5min
completed: 2026-03-19
---

# Phase 02 Plan 02: Data Clean Command Summary

**New /ant:data-clean slash command backed by aether-utils.sh subcommand that scans 6 colony data files for test artifacts with dry-run default and confirm-to-clean safety pattern**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-19T16:52:21Z
- **Completed:** 2026-03-19T16:58:02Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Implemented data-clean subcommand in aether-utils.sh (248 lines) scanning pheromones.json, QUEEN.md, learning-observations.json, midden.json, spawn-tree.txt, and constraints.json for test artifacts
- Created /ant:data-clean slash command with scan-confirm-clean workflow and user decision gate before any destructive changes
- Added data-clean to help.md MAINTENANCE section, bringing total command count to 38

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement data-clean subcommand in aether-utils.sh** - `da33fa0` (feat)
2. **Task 2: Create /ant:data-clean slash command and update help** - `8b1e452` (feat)

## Files Created/Modified
- `.aether/aether-utils.sh` - Added data-clean subcommand with --dry-run, --confirm, --json flags
- `.claude/commands/ant/data-clean.md` - New slash command with 5-step workflow (scan, decide, clean, summarize, next-up)
- `.claude/commands/ant/help.md` - Added data-clean entry to MAINTENANCE section

## Decisions Made
- Placed subcommand before wildcard case at end of the 10K-line file for minimal diff impact
- Used `var=$(...) || var=0` pattern instead of `var=$(... || echo 0)` to avoid grep exit-code-1 creating multiline values in bash arithmetic

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed grep -c exit code handling for bash arithmetic**
- **Found during:** Task 1 (data-clean subcommand implementation)
- **Issue:** `grep -cE pattern file || echo 0` produces "0\n0" when grep finds no matches (exit 1 triggers echo 0, but the original 0 count is also output)
- **Fix:** Changed to `var=$(...) || var=0` which properly handles the exit code without creating multiline values
- **Files modified:** .aether/aether-utils.sh
- **Verification:** data-clean --dry-run exits 0 with correct counts
- **Committed in:** da33fa0 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Bug fix was necessary for correct arithmetic. No scope creep.

## Issues Encountered
None beyond the grep exit code fix documented above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- /ant:data-clean is fully operational for ongoing artifact removal
- Combined with Phase 1 manual purge, colony data maintenance is now automated
- 537 tests pass with no breakage

## Self-Check: PASSED

All artifacts verified:
- .claude/commands/ant/data-clean.md: FOUND
- .aether/aether-utils.sh (data-clean subcommand): FOUND
- .claude/commands/ant/help.md (data-clean entry): FOUND
- Commit da33fa0: FOUND
- Commit 8b1e452: FOUND

---
*Phase: 02-command-audit-data-tooling*
*Completed: 2026-03-19*
