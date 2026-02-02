---
phase: 12-visual-indicators-documentation
plan: 02
subsystem: infrastructure
tags: [path-auditing, bash-scripts, git-root-detection, documentation]

# Dependency graph
requires:
  - phase: 11
    provides: Event polling integration with test suite
provides:
  - Verified and corrected path references in all utility scripts
  - Fixed bash syntax errors in command prompt code examples
  - Consistent path patterns throughout codebase
affects: [13-real-llm-testing]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Git root detection pattern for subdirectory robustness
    - Source-then-call pattern for bash function invocation
    - Relative path sourcing for utility scripts

key-files:
  modified:
    - .aether/utils/atomic-write.sh
    - .aether/utils/file-lock.sh
    - .aether/utils/memory-compress.sh
    - .aether/utils/spawn-tracker.sh
    - .claude/commands/ant/build.md
    - .claude/commands/ant/feedback.md
    - .claude/commands/ant/init.md
    - .claude/commands/ant/redirect.md
    - .claude/commands/ant/execute.md
    - .claude/commands/ant/pause-colony.md
    - .claude/commands/ant/plan.md
    - .claude/commands/ant/resume-colony.md
    - .claude/commands/ant/review.md

key-decisions:
  - "Use git root detection for paths that must work from subdirectories"
  - "Fix incorrect bash syntax: .aether/utils/script.sh function_name → source script then call function"
  - "Standardize on .aether/data/ for all data file paths"

patterns-established:
  - "Git root detection: AETHER_ROOT=$(git rev-parse --show-toplevel || echo $PWD)"
  - "Utility sourcing: source .aether/utils/script.sh then call function_name"
  - "Data files: .aether/data/COLONY_STATE.json not .aether/COLONY_STATE.json"

# Metrics
duration: 7min
completed: 2026-02-02
---

# Phase 12 Plan 02: Path Reference Audit Summary

**Verified and corrected all path references in utility scripts and command prompts, eliminating "file not found" errors from incorrect paths**

## Performance

- **Duration:** 7 min
- **Started:** 2026-02-02T16:37:50Z
- **Completed:** 2026-02-02T16:45:26Z
- **Tasks:** 2
- **Files modified:** 13

## Accomplishments

- **Utility script path robustness:** Added git root detection to atomic-write.sh, file-lock.sh for subdirectory execution
- **Bash syntax corrections:** Fixed incorrect `.aether/utils/script.sh function_name` pattern in command prompts
- **Data path standardization:** Corrected all `.aether/COLONY_STATE.json` references to `.aether/data/COLONY_STATE.json`
- **Export statement cleanup:** Removed non-existent function export from spawn-tracker.sh

## Task Commits

Each task was committed atomically:

1. **Task 1: Audit and fix path references in utility scripts** - `59283ae` (fix)
2. **Task 2: Audit and fix path references in command prompts** - `bda2b8b` (fix)

**Plan metadata:** (will be committed separately)

## Files Created/Modified

### Utility Scripts

- `.aether/utils/atomic-write.sh` - Added git root detection for TEMP_DIR and BACKUP_DIR paths
- `.aether/utils/file-lock.sh` - Added git root detection for LOCK_DIR path
- `.aether/utils/memory-compress.sh` - Fixed pheromones.json path reference with fallback
- `.aether/utils/spawn-tracker.sh` - Removed get_specialist_confidence from exports (not defined here)

### Command Prompts

- `.claude/commands/ant/build.md` - Added source statements before atomic_write_from_file calls (2 locations)
- `.claude/commands/ant/feedback.md` - Added source statement before atomic_write_from_file call
- `.claude/commands/ant/init.md` - Added source statements before atomic_write_from_file calls (4 locations)
- `.claude/commands/ant/redirect.md` - Added source statement before atomic_write_from_file call
- `.claude/commands/ant/execute.md` - Fixed .aether/COLONY_STATE.json → .aether/data/COLONY_STATE.json (Python code)
- `.claude/commands/ant/pause-colony.md` - Fixed .aether/COLONY_STATE.json → .aether/data/COLONY_STATE.json
- `.claude/commands/ant/plan.md` - Fixed .aether/COLONY_STATE.json → .aether/data/COLONY_STATE.json
- `.claude/commands/ant/resume-colony.md` - Fixed .aether/COLONY_STATE.json → .aether/data/COLONY_STATE.json
- `.claude/commands/ant/review.md` - Fixed .aether/COLONY_STATE.json → .aether/data/COLONY_STATE.json

## Decisions Made

**Git root detection for subdirectory robustness:** Scripts that may be executed from subdirectories now use git root detection pattern instead of relative paths. This ensures TEMP_DIR, BACKUP_DIR, and LOCK_DIR are always found.

**Bash function calling pattern:** The pattern `.aether/utils/script.sh function_name` is incorrect bash syntax. The correct pattern is to source the script first (`source .aether/utils/script.sh`) then call the function (`function_name args`). This was fixed in 4 command prompt files.

**Data file path standardization:** All references to `.aether/COLONY_STATE.json` (old path) were corrected to `.aether/data/COLONY_STATE.json` (current path). A legacy file exists at the old location but the bash scripts use the new location.

## Deviations from Plan

None - plan executed exactly as written. All identified path issues were in the plan's scope and were fixed as specified.

## Issues Encountered

**File modification during editing:** During command prompt fixes, the build.md file was modified (likely by a linter or external process) which caused edit operations to fail. This was resolved by re-reading the file and re-applying edits.

**Function export mismatch:** The spawn-tracker.sh script was exporting `get_specialist_confidence` function which doesn't exist in that file (it's defined in spawn-outcome-tracker.sh). This was fixed by removing it from the exports.

## Verification

All verification checks passed:

1. **Utility Script Path Test:** All utility scripts can be sourced successfully
2. **Source Statement Verification:** No incorrect `.aether/utils/script.sh function_name` patterns remain
3. **Path Reference Verification:** All `.aether/COLONY_STATE.json` paths now correctly use `.aether/data/COLONY_STATE.json`
4. **Data File Existence:** All referenced data files exist:
   - .aether/data/COLONY_STATE.json
   - .aether/data/events.json
   - .aether/data/memory.json
   - .aether/data/pheromones.json
   - .aether/data/watcher_weights.json
   - .aether/data/worker_ants.json

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- All path references are now verified and correct
- Scripts will work correctly when executed from subdirectories
- Command prompt code examples use correct bash syntax
- Ready for Phase 12-03 (final plan of this phase) or Phase 13

---
*Phase: 12-visual-indicators-documentation*
*Plan: 02*
*Completed: 2026-02-02*
