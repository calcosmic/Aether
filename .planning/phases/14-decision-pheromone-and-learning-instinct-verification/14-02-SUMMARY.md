---
phase: 14-decision-pheromone-and-learning-instinct-verification
plan: 02
subsystem: colony-learning
tags: [instinct, confidence, learning, recurrence, awk, aether-utils]

requires:
  - phase: 14-decision-pheromone-and-learning-instinct-verification
    provides: "Plan 01 aligned decision pheromone format for dedup"
provides:
  - "Recurrence-calibrated confidence formula in learning-promote-auto"
  - "Instinct confidence scales with observation_count evidence"
  - "Integration tests verifying LRN-01 formula at 4 data points"
affects: [learning-promote-auto, instinct-create, continue-advance, continue-full]

tech-stack:
  added: []
  patterns: ["awk-computed confidence from observation_count", "min(0.7 + (count-1)*0.05, 0.9) formula"]

key-files:
  created:
    - tests/unit/instinct-confidence.test.js
  modified:
    - .aether/aether-utils.sh
    - .aether/docs/command-playbooks/continue-advance.md
    - .aether/docs/command-playbooks/continue-full.md

key-decisions:
  - "Used decree wisdom_type for observation_count=1 test since pattern auto threshold is 2"

patterns-established:
  - "Instinct confidence is evidence-proportional: single observations start at 0.70, growing with recurrence"

requirements-completed: [LRN-01]

duration: 5min
completed: 2026-03-14
---

# Phase 14 Plan 02: Learning-to-Instinct Confidence Calibration Summary

**Recurrence-calibrated instinct confidence via awk formula min(0.7 + (count-1)*0.05, 0.9) replacing hardcoded 0.6 in learning-promote-auto**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-14T08:18:31Z
- **Completed:** 2026-03-14T08:24:06Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Replaced hardcoded `--confidence 0.6` in learning-promote-auto with awk-computed formula based on observation_count
- Updated confidence guidelines in both continue-advance.md and continue-full.md playbooks
- Added 4 integration tests verifying formula at observation_count=1/3/5/10
- Full test suite passes (537 tests, up from 533)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add recurrence-calibrated confidence to learning-promote-auto** - `ca26ccb` (feat)
2. **Task 2: Add instinct confidence calibration integration tests** - `639b164` (test)

## Files Created/Modified
- `.aether/aether-utils.sh` - Added LRN-01 awk confidence computation and replaced hardcoded 0.6 with `$lp_confidence`
- `.aether/docs/command-playbooks/continue-advance.md` - Updated Steps 3/3b confidence guidelines to reference observation-based formula
- `.aether/docs/command-playbooks/continue-full.md` - Updated Step 3 confidence table from 0.4/0.5/0.7 to 0.7/0.8/0.9 with formula note
- `tests/unit/instinct-confidence.test.js` - 4 tests verifying confidence at observation_count=1 (0.70), 3 (0.80), 5 (0.90), 10 (0.90 cap)

## Decisions Made
- Used "decree" wisdom_type (auto threshold=0) for the observation_count=1 test, since "pattern" has auto threshold=2 which would reject count=1 before reaching the confidence computation. The confidence formula is wisdom_type-agnostic, so this still validates the formula correctly.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed QUEEN.md template missing emoji section headers**
- **Found during:** Task 2 (test creation)
- **Issue:** Test QUEEN.md template used plain headers (`## Patterns`) but queen-promote expects emoji headers (`## 🧭 Patterns`)
- **Fix:** Updated QUEEN.md template in test to use Unicode escape sequences matching instinct-pipeline.test.js pattern
- **Files modified:** tests/unit/instinct-confidence.test.js
- **Committed in:** 639b164 (Task 2 commit)

**2. [Rule 1 - Bug] Fixed multi-line JSON output parsing**
- **Found during:** Task 2 (test creation)
- **Issue:** learning-promote-auto outputs two JSON lines (one from nested instinct-create, one from final json_ok); JSON.parse fails on multi-line output
- **Fix:** Updated runAetherUtil helper to return only the last non-empty line of output
- **Files modified:** tests/unit/instinct-confidence.test.js
- **Committed in:** 639b164 (Task 2 commit)

**3. [Rule 3 - Blocking] Used decree type for observation_count=1 test**
- **Found during:** Task 2 (test design)
- **Issue:** Plan specified wisdom-thresholds.json override file, but no such override mechanism exists in get_wisdom_threshold
- **Fix:** Used "decree" wisdom_type (auto threshold=0) instead of "pattern" (auto threshold=2) for the count=1 test case
- **Files modified:** tests/unit/instinct-confidence.test.js
- **Committed in:** 639b164 (Task 2 commit)

---

**Total deviations:** 3 auto-fixed (2 bugs, 1 blocking)
**Impact on plan:** All fixes necessary for test correctness. No scope creep.

## Issues Encountered
None beyond the deviations documented above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- LRN-01 requirement is complete: instinct confidence is now evidence-proportional
- DEC-01 was completed in plan 14-01
- Phase 14 is fully complete -- all verification requirements satisfied
- Ready for phase completion

## Self-Check: PASSED

All files exist, all commits verified.

---
*Phase: 14-decision-pheromone-and-learning-instinct-verification*
*Completed: 2026-03-14*
