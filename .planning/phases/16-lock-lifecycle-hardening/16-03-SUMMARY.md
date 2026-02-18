---
phase: 16-lock-lifecycle-hardening
plan: 03
subsystem: testing
tags: [bash, file-locking, atomic-write, regression-tests, known-issues]

# Dependency graph
requires:
  - phase: 16-01-lock-lifecycle-hardening
    provides: EXIT trap pattern across all 4 flag commands (LOCK-01/02), atomic_write_from_file backup ordering fix (LOCK-03)
  - phase: 16-02-lock-lifecycle-hardening
    provides: context-update acquire/release lock pair (LOCK-04), force-unlock subcommand

provides:
  - Regression test suite (tests/bash/test-lock-lifecycle.sh) with 12 tests covering LOCK-01 through LOCK-04
  - Documented fix status for BUG-002, BUG-003, BUG-005, BUG-011, GAP-009 in known-issues.md

affects: [phase-17, future-lock-changes, distribution-testing]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Isolated test env via fake 'git' binary prepended to PATH — prevents atomic-write.sh git detection from pointing to real repo root"
    - "LOCK-01 tests: corrupt flags.json to force jq failure, assert 0 .lock files remain"
    - "LOCK-02 tests: background process with signal file, kill then wait, assert no locks"
    - "LOCK-03 tests: driver subprocess with AETHER_ROOT isolation, check backup content = original"

key-files:
  created:
    - tests/bash/test-lock-lifecycle.sh
  modified:
    - .aether/docs/known-issues.md

key-decisions:
  - "Isolated atomic-write tests using fake git binary in PATH to prevent AETHER_ROOT resolving to real repo — cleaner than patching atomic-write.sh"
  - "LOCK-02 signal tests use lock_signal file (not sleep duration) to detect lock acquisition — avoids timing races"
  - "known-issues.md entries updated in-place with FIXED status, regression test references added — no restructuring"

patterns-established:
  - "Fake git binary pattern: create $tmp_dir/bin/git that exits 1, prepend to PATH before sourcing scripts with git-based root detection"

requirements-completed:
  - LOCK-01
  - LOCK-02
  - LOCK-03
  - LOCK-04

# Metrics
duration: 15min
completed: 2026-02-18
---

# Phase 16 Plan 03: Lock Lifecycle Tests and Documentation Summary

**12 bash regression tests verifying lock release on all exit paths (LOCK-01 through LOCK-04), plus known-issues.md updated to mark BUG-002/003/005/011 and GAP-009 as fixed with Phase 16 attribution**

## Performance

- **Duration:** 15 min
- **Started:** 2026-02-18T17:35:32Z
- **Completed:** 2026-02-18T17:50:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Created `tests/bash/test-lock-lifecycle.sh` (730 lines, 12 tests) covering all four LOCK requirements
- All 12 tests pass, including 4 LOCK-01 jq-failure tests, 2 LOCK-02 signal tests, 4 LOCK-03 atomic-write tests, and 2 LOCK-04 tests
- Updated `known-issues.md` to mark BUG-002, BUG-003, BUG-005, BUG-011, and GAP-009 as FIXED with Phase 16 attribution and regression test references

## Task Commits

Each task was committed atomically:

1. **Task 1: Create lock lifecycle test suite** - `7b0dcc4` (test)
2. **Task 2: Update known-issues.md with fix status** - `341cafb` (docs)

## Files Created/Modified

- `tests/bash/test-lock-lifecycle.sh` — 12-test bash regression suite for lock lifecycle correctness
- `.aether/docs/known-issues.md` — BUG-002, BUG-003, BUG-005, BUG-011, GAP-009 marked FIXED with Phase 16 notes

## Decisions Made

- **Fake git binary for atomic-write isolation:** `atomic-write.sh` uses `git rev-parse --show-toplevel` to set `AETHER_ROOT`. Tests inject a fake `git` binary that exits 1 so the script falls back to `pwd` (our isolated tmp_dir). Cleaner than patching the script under test.
- **Signal file pattern for LOCK-02:** Background lock-holding scripts write a signal file after acquiring the lock. Parent waits for the signal file before sending SIGTERM/SIGINT. Avoids timing races from fixed sleep durations.
- **In-place known-issues.md updates:** Bug entries updated with FIXED status, fix notes, and regression test references. File structure preserved — no restructuring to avoid merge conflicts with other updates.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed AETHER_ROOT isolation in atomic-write tests**
- **Found during:** Task 1, tests 7-10 (atomic_write and atomic_write_from_file tests)
- **Issue:** `atomic-write.sh` overwrites any exported `AETHER_ROOT` by calling `git rev-parse --show-toplevel` at source time, pointing backups to the real repo rather than the isolated tmp_dir. Tests 7 and 9 initially failed with "Expected at least 1 backup in backups/, found 0".
- **Fix:** Created a fake `git` binary in `$tmp_dir/bin/` that exits 1, prepended `$tmp_dir/bin` to PATH in driver scripts, and ran drivers from `cd "$tmp_dir"` so the fallback `AETHER_ROOT=$(pwd)` resolves to the isolated directory.
- **Files modified:** `tests/bash/test-lock-lifecycle.sh` (tests 7-10 driver scripts)
- **Verification:** All 4 atomic-write tests now pass; backup files appear in `$tmp_dir/.aether/data/backups/`
- **Committed in:** `7b0dcc4` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 — bug in test isolation approach)
**Impact on plan:** Fix was necessary for test correctness. No scope creep.

## Issues Encountered

None beyond the AETHER_ROOT isolation issue documented above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 16 complete: all four LOCK requirements implemented (16-01, 16-02) and regression-tested (16-03)
- `known-issues.md` accurately reflects fix status — no open lock-related bugs remain
- Ready for Phase 17 (error code standardisation) or Phase 18 (npm publish cycle)

## Self-Check: PASSED

- FOUND: tests/bash/test-lock-lifecycle.sh
- FOUND: .aether/docs/known-issues.md
- FOUND: 16-03-SUMMARY.md
- FOUND commit: 7b0dcc4 (test: lock lifecycle test suite)
- FOUND commit: 341cafb (docs: known-issues.md fix status)

---
*Phase: 16-lock-lifecycle-hardening*
*Completed: 2026-02-18*
