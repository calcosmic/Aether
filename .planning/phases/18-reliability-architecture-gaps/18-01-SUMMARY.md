---
phase: 18-reliability-architecture-gaps
plan: 01
subsystem: infra
tags: [shell, startup, trap, cleanup, spawn-tree, orphan-detection]

# Dependency graph
requires:
  - phase: 16-lock-hardening
    provides: cleanup_locks function in file-lock.sh, composed trap pattern
  - phase: 14-foundation-fixes
    provides: fallback json_err definition that feature detection must follow
provides:
  - Correct startup ordering (ARCH-09): feature detection after all fallback definitions
  - Composed EXIT trap (ARCH-10): _aether_exit_cleanup calls both cleanup_locks and cleanup_temp_files
  - Startup orphan temp file cleanup (ARCH-10): _cleanup_orphaned_temp_files removes dead-PID .tmp files
  - Spawn-tree rotation at session-init (ARCH-03): archives previous session's tree, caps at 5 archives
  - Three regression tests in test-aether-utils.sh (ARCH-09, ARCH-10, ARCH-03)
affects:
  - session management
  - exit cleanup behavior
  - spawn-tree growth

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Composed EXIT trap: single _aether_exit_cleanup function overrides individual traps from sourced files"
    - "PID-based orphan detection: kill -0 to check liveness without signaling"
    - "In-place truncation (> file): preserves file handle for tail -f watchers"
    - "5-archive cap: ls -t | tail -n +6 | xargs rm -f"

key-files:
  created: []
  modified:
    - .aether/aether-utils.sh
    - tests/bash/test-aether-utils.sh

key-decisions:
  - "Feature detection block moved after fallback json_err (line 68) to line 81 — correctness over marginal ordering speed"
  - "Composed EXIT trap uses _aether_exit_cleanup placed after file-lock.sh source to override its individual trap"
  - "Orphan cleanup is silent on startup — matches existing lock cleanup behavior"
  - "Spawn-tree rotation uses archive-not-wipe strategy: timestamped archive preserves previous session for debugging"
  - "Archive cap is 5 files to bound disk usage without losing useful history"
  - "Rotation truncates in-place (> file) rather than rm+touch to preserve tail -f file handles"

patterns-established:
  - "Startup ordering: source utils -> E_* constants -> fallback atomic_write -> json_ok/json_err -> feature detection -> composed trap -> orphan cleanup"
  - "Regression tests for startup ordering use line-number comparison (feature_line > fallback_line)"

requirements-completed:
  - ARCH-09
  - ARCH-10
  - ARCH-03

# Metrics
duration: 4min
completed: 2026-02-19
---

# Phase 18 Plan 01: Reliability Architecture Gaps — Startup Ordering, Exit Trap Composition, Spawn-Tree Rotation Summary

**Startup ordering fixed (feature detection after fallbacks), EXIT trap composed to call both lock and temp cleanup, startup orphan scan added, and spawn-tree rotation with 5-archive cap added at session-init**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-19T15:00:00Z
- **Completed:** 2026-02-19T15:04:44Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Fixed ARCH-09: feature detection block now runs after all fallback definitions (atomic_write, json_ok, json_err) — eliminates potential "function not found" startup race
- Fixed ARCH-10: composed `_aether_exit_cleanup` trap overrides file-lock.sh's individual trap so both `cleanup_locks` and `cleanup_temp_files` fire on every exit path; startup orphan scan removes .tmp files from dead PIDs
- Fixed ARCH-03: `_rotate_spawn_tree` at session-init archives previous spawn-tree.txt to timestamped file, truncates in-place, caps at 5 archives to prevent unbounded disk growth
- Added 3 regression tests (tests 24-26 in test-aether-utils.sh); all 26 bash tests pass

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix startup ordering, compose EXIT trap, wire startup orphan cleanup** - `f75490d` (feat)
2. **Task 2: Add spawn-tree rotation at session-init and regression tests** - `09cf157` (feat)

## Files Created/Modified
- `.aether/aether-utils.sh` - Startup section reordered; _aether_exit_cleanup, _cleanup_orphaned_temp_files, and _rotate_spawn_tree added
- `tests/bash/test-aether-utils.sh` - Three regression tests added: ARCH-09, ARCH-10, ARCH-03

## Decisions Made
- Feature detection moved after fallback json_err (not just after E_* constants) — full correctness guarantee: all fallback infrastructure (atomic_write, json_ok, json_err) is available when feature detection runs
- Composed trap uses `|| true` on both calls so neither failure prevents the other from running
- `_rotate_spawn_tree` is defined and called inside the `session-init)` case block (local scope) — no namespace pollution in global scope
- Archive strategy chosen over wipe: previous session's spawn tree preserved for post-mortem debugging
- Orphan cleanup uses `kill -0` (macOS/Linux-compatible) rather than /proc/ file access

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Pre-existing validate-state unit test failures (2-4 flaky failures in the AVA suite) unrelated to this plan's changes — confirmed by running tests with and without changes. Bash test suite shows 0 failures across all 26 tests.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- ARCH-09, ARCH-10, ARCH-03 complete — reliability architecture gap work can continue with remaining gaps
- Regression tests in place to prevent reintroduction of ordering/cleanup bugs

---
*Phase: 18-reliability-architecture-gaps*
*Completed: 2026-02-19*
