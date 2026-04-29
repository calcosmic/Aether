---
phase: 74-suggest-analyze
plan: 01
subsystem: cli
tags: [cobra, go, pheromones, suggest-analyze, change-detection, dedup]

# Dependency graph
requires: []
provides:
  - "PendingSuggestion schema in ColonyState for persisting unreviewed suggestions"
  - "suggest-analyze CLI command with pattern detection, dedup, and persistence"
  - "Change detection via git diff --stat with configurable threshold"
  - "Build-specific extra patterns: TODO/FIXME density, large files, test gaps, dependency count"
affects: [74-02-suggest-approve, build-playbook]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Threshold-based change detection (git diff --stat) to skip re-analysis"
    - "Content hash deduplication against active pheromone signals"
    - "Non-blocking command pattern: ok:true with empty results on any error"

key-files:
  created:
    - cmd/suggest_analyze.go
    - cmd/suggest_analyze_test.go
  modified:
    - pkg/colony/colony.go

key-decisions:
  - "Change threshold set to 5 files -- small changes don't trigger re-analysis"
  - "Non-blocking design: all errors return ok:true with empty suggestions"
  - "Build-specific patterns scan source files but skip vendor/node_modules/.git"

patterns-established:
  - "suggest-analyze reuses generatePheromoneSuggestions from init_research.go"
  - "Dedup uses same sha256Sum + 'sha256:' prefix as pheromone_write.go"

requirements-completed: [INTEL-01, INTEL-02]

# Metrics
duration: 5min
completed: 2026-04-29
---

# Phase 74 Plan 01: suggest-analyze Command Summary

**suggest-analyze CLI command with threshold-based change detection, 25+ pattern detection, content hash dedup against active pheromones, and colony state persistence**

## Performance

- **Duration:** 5 min
- **Started:** 2026-04-29T13:11:47Z
- **Completed:** 2026-04-29T13:16:47Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- PendingSuggestion schema added to ColonyState with all required fields
- suggest-analyze command detects 25+ codebase patterns plus 4 build-specific extras
- Content hash deduplication filters out suggestions matching active pheromones
- Change detection skips re-analysis when fewer than 5 files changed since last run
- All 9 tests pass, zero regressions in existing pkg/colony tests

## Task Commits

Each task was committed atomically:

1. **Task 1: Add PendingSuggestion schema to ColonyState** - `a5770076` (feat)
2. **Task 2: Implement suggest-analyze command (TDD)** - `c56987eb` (test), `6bc86c6a` (feat)

## Files Created/Modified
- `pkg/colony/colony.go` - Added PendingSuggestion struct and two new ColonyState fields
- `cmd/suggest_analyze.go` - Full suggest-analyze command with change detection, pattern detection, dedup, sanitization, and persistence
- `cmd/suggest_analyze_test.go` - 9 tests covering suggestions, dedup, persistence, non-blocking, build patterns, dry-run, and change detection

## Decisions Made
- Change threshold set to 5 files -- small changes don't trigger re-analysis (per CONTEXT.md discretion)
- Non-blocking design: all errors return ok:true with empty suggestions (per RESEARCH Pitfall 3)
- Build-specific patterns use directory skipping for .git, node_modules, vendor, .aether, etc.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Created missing .aether/rules directory**
- **Found during:** Task 2 (RED phase test compilation)
- **Issue:** embedded_assets.go references `all:.aether/rules` but the directory didn't exist in the worktree, preventing any test compilation
- **Fix:** Created `.aether/rules/` with a `.gitkeep` placeholder
- **Files modified:** `.aether/rules/.gitkeep`
- **Verification:** Test compilation succeeded after directory creation
- **Committed in:** Part of test commit `c56987eb`

**2. [Rule 1 - Bug] Fixed test using wrong state constant**
- **Found during:** Task 2 (RED phase test compilation)
- **Issue:** Tests used `colony.StateActive` which doesn't exist; correct constant is `colony.StateREADY`
- **Fix:** Changed to `colony.StateREADY` in all test setup functions
- **Files modified:** cmd/suggest_analyze_test.go
- **Verification:** Tests compile and pass

**3. [Rule 1 - Bug] Fixed duplicate runGit function and wrong exec.Command usage**
- **Found during:** Task 2 (RED phase test compilation)
- **Issue:** `runGit` already declared in init_cmd_test.go; `execGitRevParse` used wrong function signature
- **Fix:** Removed duplicate `runGit`, rewrote `execGitRevParse` and `initTestGitRepo` to use existing `runGit` helper
- **Files modified:** cmd/suggest_analyze_test.go
- **Verification:** Tests compile without redeclaration errors

**4. [Rule 1 - Bug] Fixed Test 9 creating files with same name**
- **Found during:** Task 2 (GREEN phase)
- **Issue:** Test 9 created 6 files all named "changed.go" so git only tracked one, making diff --stat show 1 file (below threshold)
- **Fix:** Changed to unique filenames `changed_0.go` through `changed_5.go`
- **Files modified:** cmd/suggest_analyze_test.go
- **Verification:** Test 9 passes with 6 changed files exceeding threshold

---

**Total deviations:** 4 auto-fixed (1 blocking, 3 bugs)
**Impact on plan:** All fixes were necessary for compilation and correctness. No scope creep.

## Issues Encountered
- Worktree embedded_assets.go requires `.aether/rules` directory which was missing -- created placeholder to unblock compilation

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- suggest-analyze command fully functional with tests
- Ready for plan 02 (suggest-approve) which will consume PendingSuggestions from colony state
- LastAnalyzeCommit field set by suggest-analyze enables change detection across builds

---
*Phase: 74-suggest-analyze*
*Completed: 2026-04-29*
