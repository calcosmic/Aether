---
phase: 19-milestone-polish
plan: 02
subsystem: testing
tags: [ava, validate-state, parallel-tests, data-isolation, bash-tests]

# Dependency graph
requires:
  - phase: 17-error-handling-standardization
    provides: "Structured error format {code,message,details,recovery,timestamp} that replaced bare strings"
  - phase: 18-reliability-architecture-gaps
    provides: "_migrate_colony_state in aether-utils.sh that validate-state tests exercise"
provides:
  - "All 11 validate-state.test.js tests pass with zero failures"
  - "DATA_DIR conditional override in aether-utils.sh (line 19)"
  - "Module-level snapshot isolation pattern for tests that read colony data files"
affects: [any future test files that read .aether/data/ files]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Module-level snapshot dir created at require() time to guard against concurrent test file corruption"
    - "DATA_DIR=${DATA_DIR:-...} conditional assignment to allow env-var override in tests"
    - "Per-test isolated tmpDir populated from module snapshot, cleaned up via t.teardown()"

key-files:
  created: []
  modified:
    - tests/unit/validate-state.test.js
    - .aether/aether-utils.sh

key-decisions:
  - "Module-level snapshot (not per-test copy) is the correct isolation boundary — snapshot created at require() before AVA starts running any tests, so state-loader concurrent corruption window is eliminated"
  - "DATA_DIR conditional assignment uses ${DATA_DIR:-default} not export — bash inherits env vars from parent process, so subprocess spawned by validate-state all also picks up the override"
  - "Error tests (validate-state invalid-target, validate-state without-arg) kept unisolated — they fail before reading any data files, so no race condition exists"
  - "error.error.message.includes() used instead of error.error.includes() — Phase 17 changed error.error from string to {code,message,details,recovery,timestamp} object"

patterns-established:
  - "Snapshot-then-copy pattern: create file snapshot at module init time, copy from snapshot into per-test tmpDir. Prevents concurrent test files from corrupting in-flight copies."
  - "DATA_DIR env override for aether-utils.sh subcommands: pass env: { ...process.env, DATA_DIR: tmpDir } to execSync"

requirements-completed: [ERR-02]

# Metrics
duration: 15min
completed: 2026-02-19
---

# Phase 19 Plan 02: Validate-State Test Fix Summary

**Fixed 2 failing assertions (error.error -> error.error.message) and eliminated AVA parallel-execution race via module-level snapshot isolation and DATA_DIR env override**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-02-19T16:28:00Z
- **Completed:** 2026-02-19T16:34:47Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Fixed both `error.error.includes()` calls — Phase 17 changed `error.error` from a string to a structured `{code, message, details, recovery, timestamp}` object; assertions now use `error.error.message.includes()`
- Added E_VALIDATION_FAILED code assertions to both error tests for stronger coverage
- Made `DATA_DIR` overridable in `aether-utils.sh` line 19 via `${DATA_DIR:-...}` conditional — enables test isolation without changing runtime behavior
- Implemented module-level snapshot isolation: data files copied at `require()` time (before AVA starts any tests) so concurrent `state-loader.test.js` mutations cannot corrupt in-flight copies
- All 11 validate-state tests pass reliably across 3 consecutive concurrent runs with state-loader.test.js

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix error.error assertion type mismatch** - `a236d9f` (fix)
2. **Task 2: Isolate validate-state colony tests with DATA_DIR temp directories** - `a5759b3` (fix)

## Files Created/Modified
- `tests/unit/validate-state.test.js` - Added fs/os imports, module-level snapshot dir, runUtilsCommandIsolated helper, createIsolatedDataDir helper; all colony/constraints/all tests use isolated temp dirs; error assertions updated to .message.includes()
- `.aether/aether-utils.sh` (line 19) - Changed `DATA_DIR="..."` to `DATA_DIR="${DATA_DIR:-...}"` to allow env-var override

## Decisions Made
- **Snapshot-at-module-init, not per-test**: The key insight is that `createIsolatedDataDir` called inside a test function runs concurrently with state-loader tests. Moving the snapshot creation to module level (before `test()` calls) ensures the snapshot is always taken from the clean, valid state before any concurrent mutation.
- **DATA_DIR conditional assignment only**: No changes to how DATA_DIR is used throughout aether-utils.sh — only the initial assignment becomes conditional, which is backward compatible.
- **Error tests stay unisolated**: `validate-state invalid-target` and `validate-state` (no arg) both error before touching any data files — the E_VALIDATION_FAILED is returned at the argument-parsing step, so no isolation needed.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] runUtilsCommandExpectError used direct JSON.parse(error.stderr) — fragile if stderr has multiple lines**
- **Found during:** Task 1 (fixing .includes() calls)
- **Issue:** The original `runUtilsCommandExpectError` did `JSON.parse(error.stderr)` directly, which would throw if stderr contains multiple lines (non-JSON preamble). The new version scans lines from the end to find the last valid JSON, matching the pattern used in other test files.
- **Fix:** Updated `runUtilsCommandExpectError` to scan lines in reverse and parse the last valid JSON line
- **Files modified:** tests/unit/validate-state.test.js
- **Verification:** Both error tests pass with correct JSON parsed
- **Committed in:** a5759b3 (Task 2 commit)

**2. [Rule 1 - Bug] Concurrent run failure traced to snapshot timing, not DATA_DIR inheritance**
- **Found during:** Task 2 verification
- **Issue:** First isolation approach (createIsolatedDataDir inside test function) still failed because state-loader.test.js writes `{"invalid json` to the real COLONY_STATE.json, and a concurrent test's `createIsolatedDataDir` copies the corrupted file into its tmpDir before state-loader restores it
- **Fix:** Move snapshot creation to module level — executed synchronously at require() before AVA launches any test workers, guaranteeing a clean copy
- **Files modified:** tests/unit/validate-state.test.js
- **Verification:** 3 consecutive concurrent runs of validate-state + state-loader = 26 tests passed each time
- **Committed in:** a5759b3 (same commit)

---

**Total deviations:** 2 auto-fixed (both Rule 1 — bugs discovered during verification)
**Impact on plan:** Both fixes necessary for correctness. No scope creep.

## Issues Encountered
- Initial isolation approach (per-test copyFileSync inside test body) failed because AVA runs test functions concurrently even within the same worker, and the state-loader test corrupts the real COLONY_STATE.json in a brief window between our copy and test execution. Solution was to hoist snapshot creation to module-init time, before any test code runs.

## Next Phase Readiness
- validate-state.test.js is fully reliable under AVA parallel execution
- DATA_DIR override pattern available for any future test needing to isolate colony data access

---
*Phase: 19-milestone-polish*
*Completed: 2026-02-19*

## Self-Check: PASSED

- tests/unit/validate-state.test.js: FOUND
- .aether/aether-utils.sh: FOUND
- .planning/phases/19-milestone-polish/19-02-SUMMARY.md: FOUND
- Commit a236d9f: FOUND
- Commit a5759b3: FOUND
- Conditional DATA_DIR pattern in aether-utils.sh: FOUND
- error.error.message.includes() in test file: FOUND
