---
phase: 05-wisdom-promotion
plan: 02
subsystem: testing
tags: [wisdom, queen, auto-promotion, learning-promote-auto, colony-prime, integration-tests]

# Dependency graph
requires:
  - phase: 05-wisdom-promotion
    plan: 01
    provides: "Batch wisdom auto-promotion wiring in continue-finalize and seal playbooks"
  - phase: 04-pheromone-auto-emission
    provides: "Auto-emission infrastructure and learning-observations.json population"
provides:
  - "8 integration tests proving wisdom promotion pipeline works end-to-end"
  - "Regression protection for QUEEN-01 (auto-promotion), QUEEN-02 (batch sweep), QUEEN-03 (colony-prime visibility)"
  - "parseLastJson pattern for multi-line subcommand output"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "parseLastJson helper: takes last JSON line from multi-line subcommand output"
    - "Section-scoped assertions: count occurrences within specific QUEEN.md sections, not whole file (Evolution Log also contains content)"
    - "setupTestColony with COLONY_STATE.json for tests requiring colony-prime or instinct-create"

key-files:
  created:
    - "tests/integration/wisdom-promotion.test.js"
  modified:
    - ".aether/aether-utils.sh"

key-decisions:
  - "Used parseLastJson helper instead of JSON.parse for learning-promote-auto and memory-capture output -- these subcommands call instinct-create which also outputs JSON to stdout"
  - "Fixed memory-capture tail -1 bug in aether-utils.sh -- multi-line learning-promote-auto output corrupted auto_promoted and promotion_reason fields"
  - "QUEEN.md content assertions scope to section headers (e.g. Patterns) not whole file, because queen-promote also writes to Evolution Log"
  - "Test 8 verifies absence of promoted-format entries rather than absence of QUEEN WISDOM header, because colony-prime includes placeholder text as non-empty wisdom"

patterns-established:
  - "parseLastJson: when aether-utils subcommands call other subcommands that also use json_ok, take the last line as authoritative result"
  - "Section-scoped QUEEN.md assertions: split on section header and next ## to isolate content"

requirements-completed: [QUEEN-01, QUEEN-02, QUEEN-03]

# Metrics
duration: 11min
completed: 2026-03-07
---

# Phase 5 Plan 2: Wisdom Promotion Integration Tests Summary

**8 integration tests proving end-to-end wisdom promotion: auto-threshold promotion via learning-promote-auto, batch sweep idempotency, and colony-prime prompt_section visibility**

## Performance

- **Duration:** 11 min
- **Started:** 2026-03-07T00:22:43Z
- **Completed:** 2026-03-07T00:33:53Z
- **Tasks:** 1
- **Files modified:** 2

## Accomplishments
- Created wisdom-promotion.test.js with 8 integration tests covering all three QUEEN requirements
- Fixed pre-existing bug in memory-capture where multi-line learning-promote-auto output corrupted JSON response fields
- Established parseLastJson pattern for safely parsing subcommand output that contains internal subprocess JSON

## Task Commits

Each task was committed atomically:

1. **Task 1: Create wisdom-promotion.test.js with end-to-end tests** - `01c3a54` (test)

## Files Created/Modified
- `tests/integration/wisdom-promotion.test.js` - 8 integration tests for wisdom promotion pipeline (472 lines)
- `.aether/aether-utils.sh` - Fixed memory-capture to use `tail -1` on learning-promote-auto output, preventing multi-line JSON corruption

## Decisions Made
- Used `parseLastJson` helper that extracts the last JSON line from multi-line output, handling the case where learning-promote-auto calls instinct-create which also outputs JSON to stdout
- Fixed memory-capture's handling of learning-promote-auto subprocess output (Rule 1 - Bug: multi-line JSON corruption)
- Scoped QUEEN.md content assertions to specific sections (Patterns, Philosophies) rather than whole file, since queen-promote writes both section entry AND Evolution Log entry containing the same content
- Adapted test 8 ("empty QUEEN.md") to verify absence of promoted-format entries rather than QUEEN WISDOM header, since colony-prime includes placeholder text (e.g. "*No patterns recorded yet*") as non-empty wisdom

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed memory-capture multi-line JSON output corruption**
- **Found during:** Task 1 (test execution)
- **Issue:** memory-capture calls learning-promote-auto which internally calls instinct-create. When COLONY_STATE.json exists, instinct-create outputs JSON to stdout before learning-promote-auto's own output, causing `auto_promoted` and `promotion_reason` fields to contain multi-line values (e.g. "false\ntrue") that corrupt the final JSON response
- **Fix:** Added `tail -1` to capture only the last line (authoritative result) from learning-promote-auto subprocess output
- **Files modified:** `.aether/aether-utils.sh` (line 5398-5399)
- **Verification:** memory-capture integration test passes with valid single-line JSON output
- **Committed in:** `01c3a54` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Bug fix necessary for memory-capture test correctness. No scope creep.

## Issues Encountered
- QUEEN.md content appears in both section body AND Evolution Log, requiring section-scoped assertions for idempotency checks
- Colony-prime includes QUEEN WISDOM header even when QUEEN.md has only placeholder text, requiring test 8 to verify absence of promoted-format content instead

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All 3 QUEEN requirements fully tested and verified
- Phase 5 (wisdom promotion) is complete
- 443 total tests pass across the entire suite

## Self-Check: PASSED

All files exist, all commits verified.

---
*Phase: 05-wisdom-promotion*
*Completed: 2026-03-07*
