---
phase: 18-local-wisdom-injection
plan: 01
subsystem: colony-prime
tags: [wisdom-injection, queen-md, pheromone, prompt-assembly, bash]

# Dependency graph
requires:
  - phase: 17-local-wisdom-accumulation
    provides: "Automatic QUEEN.md writes during builds (codebase patterns, build learnings, instincts)"
provides:
  - "Post-extraction wisdom filtering via _filter_wisdom_entries() in pheromone.sh"
  - "Content-detection gate: QUEEN WISDOM section only built when real entries exist"
  - "Fresh vs accumulated colony behavior tests"
affects: [19-cross-colony-wisdom, colony-prime-budget]

# Tech tracking
tech-stack:
  added: []
  patterns: ["Post-extraction filtering pattern -- filter AFTER extraction to avoid dual-function drift"]

key-files:
  created:
    - tests/bash/test-wisdom-injection.sh
  modified:
    - .aether/utils/pheromone.sh

key-decisions:
  - "Filter AFTER _extract_wisdom() rather than modifying it -- avoids dual-function drift with queen.sh"
  - "Renamed QUEEN WISDOM header from 'Eternal Guidance' to 'Colony Experience' for accuracy"
  - "Entry-only filtering via grep for lines starting with '- ' or '### ' -- simple, reliable"

patterns-established:
  - "Post-extraction filtering: apply _filter_wisdom_entries() to raw extracted text rather than modifying extraction logic"
  - "Content-detection gate: check filtered (not raw) variables before building prompt sections"

requirements-completed: [QUEEN-03]

# Metrics
duration: 5min
completed: 2026-03-25
---

# Phase 18 Plan 01: Local Wisdom Injection Summary

**Post-extraction wisdom filtering strips QUEEN.md description paragraphs so workers receive only real entries, with fresh colonies producing no QUEEN WISDOM section at all**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-25T00:31:56Z
- **Completed:** 2026-03-25T00:36:42Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Colony-prime now filters QUEEN.md wisdom post-extraction, keeping only bullet entries and phase headers
- Fresh colonies with placeholder-only QUEEN.md produce no QUEEN WISDOM section (zero wasted budget chars)
- Accumulated colonies get a clean "QUEEN WISDOM (Colony Experience)" section with real entries only
- Description text overhead (~800 chars) eliminated from prompt_section
- All 584 existing tests pass, plus 5 new wisdom injection tests

## Task Commits

Each task was committed atomically:

1. **Task 1: Add post-extraction wisdom filtering and content-detection gate** - `00e2a3d` (feat)
2. **Task 2: Add wisdom injection tests proving fresh vs accumulated behavior** - `45fb5c7` (test)

## Files Created/Modified
- `.aether/utils/pheromone.sh` - Added `_filter_wisdom_entries()` helper and replaced QUEEN WISDOM section builder to use filtered variables
- `tests/bash/test-wisdom-injection.sh` - 5 tests proving fresh vs accumulated colony behavior

## Decisions Made
- Filter AFTER `_extract_wisdom()` rather than modifying the extraction function -- avoids dual-function drift with `_extract_wisdom_sections()` in queen.sh
- Renamed QUEEN WISDOM header from "Eternal Guidance" to "Colony Experience" per research recommendation -- more accurate label for local wisdom
- Used simple grep-based filtering (`^(- |### )`) to keep only entries and phase headers -- reliable, no false positives

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed Python regex escaping in test helper**
- **Found during:** Task 2 (wisdom injection tests)
- **Issue:** The `get_prompt_section` Python helper had double-escaped backslashes in the regex, causing it to fail to match the prompt_section JSON field
- **Fix:** Matched the exact escaping pattern from the working `test-colony-prime-budget.sh` file
- **Files modified:** `tests/bash/test-wisdom-injection.sh`
- **Verification:** All 5 tests pass after fix
- **Committed in:** 45fb5c7 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Minor escaping fix within the test file itself. No scope creep.

## Issues Encountered
None beyond the auto-fixed escaping issue.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Wisdom injection is now meaningful -- workers receive only real entries, not boilerplate
- Budget enforcement still works (existing budget tests pass)
- Ready for Phase 19 (cross-colony wisdom) which builds on this filtered injection pattern

## Self-Check: PASSED

- [x] `.aether/utils/pheromone.sh` exists
- [x] `tests/bash/test-wisdom-injection.sh` exists (368 lines, min 80)
- [x] `18-01-SUMMARY.md` exists
- [x] Commit `00e2a3d` exists (Task 1)
- [x] Commit `45fb5c7` exists (Task 2)

---
*Phase: 18-local-wisdom-injection*
*Completed: 2026-03-25*
