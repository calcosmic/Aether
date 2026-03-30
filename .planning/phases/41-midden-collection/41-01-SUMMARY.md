---
phase: 41-midden-collection
plan: 01
subsystem: midden
tags: [cross-branch, idempotency, revert, pattern-detection, auto-redirect, retention]

# Dependency graph
requires:
  - phase: 40-lifecycle-enhancement
    provides: Worktree path resolution pattern
provides:
  - midden-collect subcommand with dual-layer idempotency
  - midden-handle-revert subcommand with tag-based revert tracking
  - midden-cross-pr-analysis subcommand with auto-REDIRECT emission
  - midden-prune subcommand with retention cleanup
affects: [continue, run, status, midden]

# Tech tracking
tech-stack:
  added: []
patterns: [dual-layer-idempotency, tag-based-revert, cross-pr-scoring, auto-redirect-emission, retention-pruning]

key-files:
  created:
    - tests/bash/test-midden-collection.sh
  modified:
    - .aether/utils/midden.sh
    - .aether/aether-utils.sh

key-decisions:
  - "Smart wrapper API: midden-collect accepts --branch/--merge-sha and resolves worktree path internally"
  - "Tag-based revert: entries are tagged reverted:<sha>, never deleted, for audit trail"
  - "Auto-REDIRECT suppressed to /dev/null to prevent stdout contamination"
  - "Prune counts use simple jq .field | length instead of wrapped array expression"

patterns-established:
  - "Dual-layer idempotency: Layer 1 merge fingerprint (fast path), Layer 2 entry ID dedup (safety net)"
  - "Cross-PR scoring: (unique_prs/5)*0.6 + (total_entries/10)*0.4 with systemic/critical/single-pr tiers"
  - "Auto-emit REDIRECT: non-blocking pheromone emission for systemic patterns, stdout suppressed"

requirements-completed:
  - MIDD-01
  - MIDD-02
  - MIDD-03

# Metrics
duration: 10min
completed: 2026-03-31
---

# Phase 41 Plan 01: Midden Collection Subcommands Summary

**Four new midden subcommands (collect, handle-revert, cross-pr-analysis, prune) with dual-layer idempotency, tag-based revert tracking, cross-PR systemic pattern detection with auto-REDIRECT, and retention pruning -- all 13 tests passing.**

## Performance

- **Duration:** 10 min
- **Started:** 2026-03-30T23:12:30Z
- **Completed:** 2026-03-30T23:22:45Z
- **Tasks:** 1
- **Files modified:** 3

## Accomplishments
- Implemented midden-collect with smart wrapper API resolving worktree paths from --branch/--merge-sha
- Implemented dual-layer idempotency (merge fingerprint + per-entry ID dedup) preventing duplicate collection
- Implemented tag-based revert handling that preserves entries for audit trail instead of deleting
- Implemented cross-PR analysis with scoring formula and auto-REDIRECT emission for systemic patterns
- Implemented retention pruning for stale merge records and old reverted entries
- Added dispatch entries in aether-utils.sh for all four new subcommands
- Comprehensive test suite with 13 test cases covering all four subcommands

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement all four midden subcommands with tests** - `1a4b602` (feat)

## Files Created/Modified
- `.aether/utils/midden.sh` - Added four new functions: _midden_collect, _midden_handle_revert, _midden_cross_pr_analysis, _midden_prune
- `.aether/aether-utils.sh` - Added dispatch entries and help metadata for four new subcommands
- `tests/bash/test-midden-collection.sh` - 13 test cases covering collect, revert, analysis, and prune

## Decisions Made
- Smart wrapper API (D-01): midden-collect accepts --branch/--merge-sha and resolves worktree path internally, hiding worktree path complexity from users
- Tag-based revert (D-02): entries get reverted:<sha> tag but are never deleted, maintaining full audit trail
- Auto-REDIRECT stdout suppression: pheromone-write output redirected to /dev/null to prevent JSON output contamination
- Prune count robustness: added explicit zero-default fallbacks to prevent arithmetic syntax errors on empty jq output

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed prune arithmetic syntax error from jq array wrapper**
- **Found during:** Task 1 (test execution)
- **Issue:** jq filter `[.merges // [] | length]` produced `[1]` (JSON array) instead of `1` (number), causing bash arithmetic syntax error
- **Fix:** Changed to `.merges | length` (no array wrapper) and added `${var:-0}` defaults
- **Files modified:** .aether/utils/midden.sh
- **Verification:** test_prune_stale_merges passes
- **Committed in:** 1a4b602 (part of task commit)

**2. [Rule 1 - Bug] Fixed cross-pr analysis stdout contamination from pheromone-write**
- **Found during:** Task 1 (test execution)
- **Issue:** pheromone-write emitted JSON to stdout which mixed with analysis output, causing test JSON parsing failures
- **Fix:** Redirected pheromone-write stdout to /dev/null (`>/dev/null 2>&1`)
- **Files modified:** .aether/utils/midden.sh
- **Verification:** test_cross_pr_detect_systemic passes
- **Committed in:** 1a4b602 (part of task commit)

---

**Total deviations:** 2 auto-fixed (2 bugs)
**Impact on plan:** Both fixes necessary for correct runtime behavior. No scope creep.

## Issues Encountered
None - both deviations were caught and fixed during initial test execution.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All four midden subcommands are functional and tested
- Ready for 41-02 (wiring into /ant:continue and /ant:run workflows)

---
*Phase: 41-midden-collection*
*Completed: 2026-03-31*
