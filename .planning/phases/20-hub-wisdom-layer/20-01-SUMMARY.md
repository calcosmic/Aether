---
phase: 20-hub-wisdom-layer
plan: 01
subsystem: wisdom
tags: [queen-md, colony-prime, prompt-assembly, budget-enforcement, migration]

requires:
  - phase: 18-local-wisdom-injection
    provides: "_filter_wisdom_entries and QUEEN WISDOM prompt section in colony-prime"
  - phase: 19-cross-colony-hive
    provides: "hive-read, queen-seed-from-hive, domain-detect for cross-colony wisdom flow"
provides:
  - "Split QUEEN WISDOM prompt sections: Global vs Colony-Specific with distinct headers"
  - "Source-labeled USER PREFERENCES with [global] and [local] tags"
  - "queen-migrate subcommand for v1-to-v2 QUEEN.md format conversion"
  - "Auto-migration of global QUEEN.md during colony-prime"
  - "Budget enforcement with separate trim priorities for global/local wisdom"
affects: [colony-prime, queen-commands, budget-enforcement]

tech-stack:
  added: []
  patterns:
    - "Same-path detection to avoid double-loading when HOME == AETHER_ROOT"
    - "Independent extraction and filtering of global/local wisdom streams"
    - "Source labeling of user preferences with sed prefix injection"

key-files:
  created:
    - "tests/bash/test-hub-wisdom-layer.sh"
  modified:
    - ".aether/utils/pheromone.sh"
    - ".aether/utils/queen.sh"
    - ".aether/aether-utils.sh"
    - "tests/bash/test-wisdom-injection.sh"
    - "tests/bash/test-colony-prime-budget.sh"

key-decisions:
  - "Global QUEEN wisdom trimmed before local in budget enforcement (local is more relevant to current colony)"
  - "Same-path edge case handled: when HOME == AETHER_ROOT, treat as local-only to avoid double content"
  - "v1 migration preserves entry lines only (strips description paragraphs via grep)"
  - "Global QUEEN.md should NOT have Build Learnings (colony-specific content)"

patterns-established:
  - "Split prompt sections: cp_sec_queen_global and cp_sec_queen_local replace cp_sec_queen"
  - "Auto-migration pattern: detect v1 format and migrate in-place before extraction"
  - "Source labeling: [global]/[local] prefix on user preference entries"

requirements-completed: [HUB-01, HUB-02]

duration: 16min
completed: 2026-03-25
---

# Phase 20 Plan 01: Hub Wisdom Layer Summary

**Split colony-prime QUEEN WISDOM into global/local sections with v1 migration, source-labeled preferences, and tiered budget enforcement**

## Performance

- **Duration:** 16 min
- **Started:** 2026-03-25T02:36:32Z
- **Completed:** 2026-03-25T02:53:01Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- Colony-prime now produces separate "QUEEN WISDOM (Global -- All Colonies)" and "QUEEN WISDOM (Colony-Specific)" prompt sections with distinct headers
- USER PREFERENCES entries labeled with [global] or [local] source tags so workers know provenance
- Budget enforcement trims global wisdom before local (9-step trim order, up from 8)
- queen-migrate subcommand converts v1 QUEEN.md (emoji headers) to v2 format preserving all entries
- Auto-migration triggers on first colony-prime run when global QUEEN.md is v1
- 7 new tests cover all hub wisdom layer behaviors

## Task Commits

Each task was committed atomically:

1. **Task 1: Split colony-prime prompt into global/local QUEEN WISDOM sections** - `4f457fb` (feat)
2. **Task 2: Add hub wisdom layer tests** - `7f8f47d` (test)

## Files Created/Modified
- `.aether/utils/pheromone.sh` - Split wisdom extraction, separate prompt sections, updated budget enforcement
- `.aether/utils/queen.sh` - Added _queen_migrate function for v1-to-v2 conversion
- `.aether/aether-utils.sh` - Registered queen-migrate in help JSON and dispatch
- `tests/bash/test-hub-wisdom-layer.sh` - 7 tests proving global/local distinction, empty gating, budget trim, migration
- `tests/bash/test-wisdom-injection.sh` - Updated header expectation from "Colony Experience" to "Colony-Specific"
- `tests/bash/test-colony-prime-budget.sh` - Increased learning entry text to account for description stripping after migration

## Decisions Made
- Global QUEEN wisdom trimmed before local in budget enforcement -- local wisdom is more relevant to the current colony's work
- When HOME == AETHER_ROOT (common in tests), only load as local to prevent double-counting
- v1 migration extracts entry lines only (grep '^- '), stripping description paragraphs for clean v2 output
- Global QUEEN.md should NOT have Build Learnings content (those are colony-specific by nature)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed v1 migration not preserving entries due to double-encoded JSON strings**
- **Found during:** Task 1 (queen-migrate implementation)
- **Issue:** _extract_wisdom_sections for v1 format returns double-encoded JSON strings. Grepping for entries on the raw output found nothing.
- **Fix:** Added sed to strip inner quotes and unescape newlines before grepping for entry lines
- **Files modified:** .aether/utils/queen.sh
- **Verification:** queen-migrate now preserves all v1 entries (philosophy, pattern, decree lines)
- **Committed in:** 4f457fb (Task 1 commit)

**2. [Rule 1 - Bug] Fixed same-path double-loading when HOME == AETHER_ROOT**
- **Found during:** Task 1 (budget test failures)
- **Issue:** When global and local QUEEN.md paths resolve to the same file, content was loaded twice (once for global, once for local), inflating the prompt
- **Fix:** Added same-path detection using realpath comparison; when paths match, treat as local only
- **Files modified:** .aether/utils/pheromone.sh
- **Verification:** Budget tests pass, no double-content in prompt
- **Committed in:** 4f457fb (Task 1 commit)

**3. [Rule 1 - Bug] Updated existing test expectations for renamed headers**
- **Found during:** Task 1 (test verification)
- **Issue:** test-wisdom-injection.sh expected "QUEEN WISDOM (Colony Experience)" header which was renamed to "Colony-Specific"
- **Fix:** Updated assertion to match new header name
- **Files modified:** tests/bash/test-wisdom-injection.sh
- **Committed in:** 4f457fb (Task 1 commit)

**4. [Rule 1 - Bug] Fixed budget test data insufficient after filtering change**
- **Found during:** Task 1 (budget test verification)
- **Issue:** v1 QUEEN.md descriptions were previously included in prompt content (pushing over 8000 chars). After Phase 20, descriptions are filtered out, reducing content below the budget threshold
- **Fix:** Increased learning entry text length in test data to compensate for stripped descriptions
- **Files modified:** tests/bash/test-colony-prime-budget.sh
- **Committed in:** 4f457fb (Task 1 commit)

---

**Total deviations:** 4 auto-fixed (4 Rule 1 bugs)
**Impact on plan:** All auto-fixes necessary for correctness. No scope creep.

## Issues Encountered
None beyond the auto-fixed deviations above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase 20 is the final phase of v2.2 Living Wisdom
- All wisdom systems now connected: local QUEEN.md, global QUEEN.md, hive brain, and colony-prime prompt assembly
- Workers can distinguish universal patterns (Global) from colony-specific patterns (Colony-Specific)
- v2.2 milestone complete

## Self-Check: PASSED

All 7 key files verified present on disk. Both task commits (4f457fb, 7f8f47d) verified in git log. 584 tests pass in full suite.

---
*Phase: 20-hub-wisdom-layer*
*Completed: 2026-03-25*
