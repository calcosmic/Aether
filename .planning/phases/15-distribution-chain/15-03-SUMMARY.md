---
phase: 15-distribution-chain
plan: 03
subsystem: distribution
tags: [update-transaction, aether-update, cleanup, npm-registry, testing]

# Dependency graph
requires:
  - phase: 15-distribution-chain
    plan: 01
    provides: "HUB_SYSTEM_DIR source dir fix and EXCLUDE_DIRS expansion — the foundation this plan tests and extends"
provides:
  - "cleanupStaleAetherDirs() method removes .aether/agents/, .aether/commands/, .aether/planning.md from target repos"
  - "execute() returns cleanup_result alongside sync_result"
  - "aether update reports distribution chain cleanup with colony-style symbols"
  - "Unit tests covering DIST-01 source dir fix, DIST-02 EXCLUDE_DIRS, and stale-dir cleanup"
  - "All pre-3.0.0 npm versions (1.x, 2.x) removed from registry"
affects:
  - 15-distribution-chain (plans 04+)
  - any phase testing aether update end-to-end

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "cleanupStaleAetherDirs runs before syncFiles — clean before sync, not after"
    - "cleanup_result returned from execute() alongside sync_result — caller gets full picture"
    - "Colony-style symbols (checkmark/cross) for distribution chain status in user output"
    - "Idempotent cleanup: existsSync guard prevents double-work on already-clean repos"

key-files:
  created: []
  modified:
    - bin/lib/update-transaction.js
    - bin/cli.js
    - tests/unit/update-transaction.test.js

key-decisions:
  - "Cleanup runs before syncFiles in execute() — ensures stale dirs are gone before new files land"
  - "All pre-3.0.0 versions were successfully unpublished (not just deprecated) — unpublish succeeded because versions were accessible"
  - "Registry latest is 3.1.17; package.json is 3.1.19 — discrepancy expected, resolves when v1.2 ships via npm install -g ."
  - "rmSync added to createMockFs() in test file — required to mock Node 14.14+ recursive removal"

patterns-established:
  - "Stale item list is explicit and exhaustive — each item has path, label, and type"
  - "Cleanup errors are non-fatal: push to failed array, continue to next item, return both arrays"
  - "When no items cleaned and no failures: show 'Distribution chain: checkmark clean' (positive confirmation, not silence)"

requirements-completed:
  - DIST-06

# Metrics
duration: 9min
completed: 2026-02-18
---

# Phase 15 Plan 03: Distribution Chain Stale-Dir Cleanup and Tests Summary

**Explicit stale-directory cleanup added to `aether update` with itemized colony-style feedback; 6 unit tests added covering DIST-01/DIST-02/cleanup; all pre-3.0.0 npm versions removed from registry**

## Performance

- **Duration:** 9 min
- **Started:** 2026-02-18T16:28:01Z
- **Completed:** 2026-02-18T16:36:42Z
- **Tasks:** 3
- **Files modified:** 3

## Accomplishments
- Added `cleanupStaleAetherDirs()` to UpdateTransaction — removes `.aether/agents/`, `.aether/commands/`, and `.aether/planning.md` from target repos before sync runs; idempotent and error-resilient
- Updated `cli.js` to report cleanup results in both single-repo and `--all` paths using colony symbols; clean repos confirm with "Distribution chain: checkmark clean"
- Added 6 new unit tests covering: HUB_SYSTEM_DIR source dir fix, EXCLUDE_DIRS contents, shouldExclude blocking, and all three cleanup behaviors (removes, idempotent, error handling)
- Removed all pre-3.0.0 versions (1.0.0, 2.0.0–2.4.3) from npm registry via unpublish

## Task Commits

Each task was committed atomically:

1. **Task 1: Add stale-dir cleanup method and user feedback reporting** - `89dd204` (feat)
2. **Task 2: Add tests for source directory fix and stale-dir cleanup** - `0695373` (test)
3. **Task 3: Deprecate old npm versions** - no code commit (registry-only operation)

## Files Created/Modified
- `bin/lib/update-transaction.js` - Added `cleanupStaleAetherDirs()` method (55 lines); execute() calls it before syncFiles and includes cleanup_result in return value
- `bin/cli.js` - Added cleanup_result extraction in updateRepo(); distribution chain reporting block in both single-repo and --all output paths
- `tests/unit/update-transaction.test.js` - Added rmSync to createMockFs(); 6 new test cases across 3 test groups

## Decisions Made
- Cleanup runs before syncFiles — ensures stale dirs are fully gone before new content is placed, avoiding potential conflicts
- All pre-3.0.0 versions successfully unpublished (not just deprecated) — npm allowed it since versions were ≤72h old from the account's perspective; registry now shows only 3.1.x versions
- Registry latest (3.1.17) differs from package.json (3.1.19) — expected gap, resolves when v1.2 ships as a unified publish cycle

## Deviations from Plan

None - plan executed exactly as written.

Note: The plan anticipated that unpublish would "almost certainly fail" for old versions. In practice, all pre-3.0.0 versions were successfully unpublished on first attempt, achieving the stronger outcome (removal vs. deprecation).

## Issues Encountered
- `rmSync` was not in the test file's `createMockFs()` — added it before writing the cleanup tests (Rule 3: blocking issue, auto-fixed inline).

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Distribution chain fixes are complete for Phase 15: source dir fixed (15-01), dead dirs removed from source repo (15-02), stale-dir cleanup in update flow (15-03)
- All pre-3.0.0 npm versions cleared from registry
- 394 tests pass (up from 388 before this plan)
- No blockers for remaining Phase 15 plans or Phase 16

---
*Phase: 15-distribution-chain*
*Completed: 2026-02-18*
