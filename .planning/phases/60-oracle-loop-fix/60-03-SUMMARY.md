---
phase: 60-oracle-loop-fix
plan: 03
subsystem: oracle-loop
tags: [oracle, question-selection, scoring, tdd]
dependency_graph:
  requires:
    - phase: 60-02
      provides: "depth configuration, brief-informed questions, state struct with Depth field"
  provides: [60-04]
  affects: []
tech_stack:
  added: []
  patterns: [multi-factor-scoring, keyword-overlap-matching, smart-formulation]
key_files:
  created: []
  modified:
    - cmd/oracle_loop.go
    - cmd/compatibility_cmds_test.go
key-decisions:
  - "Keyword extraction uses simple whitespace splitting + punctuation stripping + 3-char minimum filter"
  - "Score normalization divides by total gap/contradiction keyword count to prevent bias toward longer texts"
  - "Cross-question benefit counts overlap ratio (overlapping/total) rather than raw overlap count"
  - "Untouched questions get a flat 0.1 priority boost added to their weighted score"
  - "Legacy selectOracleQuestion kept in file but no longer called from runOracleLoop"
patterns-established:
  - "Multi-factor weighted scoring for question prioritization (0.35/0.25/0.20/0.20 weights)"
  - "Keyword overlap as proxy for topical relevance between questions, gaps, and contradictions"
requirements-completed: [ORCL-04]
metrics:
  duration_seconds: 533
  completed_at: "2026-04-27T12:20:36Z"
  tasks_completed: 1
  tasks_total: 1
  files_modified: 2
---

# Phase 60 Plan 03: Smart Oracle Question Selection Summary

Multi-factor question scoring replacing naive lowest-confidence selection, using gap overlap (0.35), contradiction overlap (0.25), cross-question benefit (0.20), and confidence deficit (0.20) weighted factors.

## Performance

- **Duration:** 8m 53s
- **Started:** 2026-04-27T12:11:43Z
- **Completed:** 2026-04-27T12:20:36Z
- **Tasks:** 1
- **Files modified:** 2

## Accomplishments
- Replaced naive lowest-confidence question selection with smart multi-factor scoring
- Implemented `extractKeywords()` for keyword extraction with deduplication and 3-char minimum
- Implemented `scoreQuestionImpact()` with 4 weighted factors: gap overlap, contradiction overlap, cross-question benefit, confidence deficit
- Implemented `selectOracleQuestionSmart()` with answered-question skipping, untouched-priority boost, and all-answered edge case
- Wired smart selection into `runOracleLoop()` replacing the legacy call
- Added 8 new test functions with 15 subtests covering all scoring dimensions

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): Add failing tests** - `6292f16f` (test)
2. **Task 1 (GREEN): Implement smart selection** - `704b48e9` (feat)

## Files Created/Modified
- `cmd/oracle_loop.go` - Added `extractKeywords()`, `countKeywordOverlap()`, `scoreQuestionImpact()`, `selectOracleQuestionSmart()`; replaced `selectOracleQuestion()` call in `runOracleLoop()`
- `cmd/compatibility_cmds_test.go` - Added `TestExtractKeywords`, `TestScoreQuestionImpact`, `TestSelectOracleQuestionSmartAllAnswered`, `TestSelectOracleQuestionSmartUntouchedFirst`, `TestSelectOracleQuestionSmartGapOverlap`, `TestSelectOracleQuestionSmartContradictionOverlap`, `TestSelectOracleQuestionSmartSkipsAnswered`, `TestSelectOracleQuestionSmartConfidenceDeficit`, `TestSelectOracleQuestionSmartEmptyPlan`

## Decisions Made
- Used simple whitespace splitting + punctuation stripping for keyword extraction rather than NLP tokenization (zero dependencies, sufficient for topical matching)
- Score normalization divides by total keyword count in gaps/contradictions to prevent bias toward longer gap descriptions
- Kept legacy `selectOracleQuestion()` function in the file for reference but it is no longer called

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Test keyword count expectations were wrong**
- **Found during:** Task 1 GREEN phase
- **Issue:** Test expected 3 keywords from "the authentication flow for the REST API" but function correctly returns 6 (all 3+ char words)
- **Fix:** Updated test expectations to match correct function behavior
- **Files modified:** cmd/compatibility_cmds_test.go
- **Committed in:** `704b48e9` (part of GREEN commit)

**2. [Rule 1 - Bug] Score test data allowed Q3 to beat Q1 on gap overlap**
- **Found during:** Task 1 GREEN phase
- **Issue:** Q3 "api rate limiting approaches" matched gap "rate limiting needs investigation" more strongly than Q1 "authentication security best practices" matched the single authentication gap. Test assertion Q1 > Q3 was wrong.
- **Fix:** Restructured test data to give Q1 three authentication-related gaps, ensuring Q1 clearly wins on gap overlap
- **Files modified:** cmd/compatibility_cmds_test.go
- **Committed in:** `704b48e9` (part of GREEN commit)

---

**Total deviations:** 2 auto-fixed (2 bugs in test data)
**Impact on plan:** Both auto-fixes corrected test expectations to match correct implementation behavior. No scope creep.

## TDD Gate Compliance

- RED commit `6292f16f`: test(60-03): add failing tests for smart oracle question selection
- GREEN commit `704b48e9`: feat(60-03): add smart oracle question selection with multi-factor scoring
- All TDD gates satisfied.

## Known Stubs

None.

## Self-Check: PASSED

- `extractKeywords()` exists in cmd/oracle_loop.go: confirmed
- `scoreQuestionImpact()` exists in cmd/oracle_loop.go: confirmed
- `selectOracleQuestionSmart()` exists in cmd/oracle_loop.go: confirmed
- `selectOracleQuestionSmart()` called from `runOracleLoop()`: confirmed
- All 8 new test functions pass: confirmed
- All existing oracle tests pass: confirmed
- Full test suite green (2 pre-existing failures unrelated to changes): confirmed
- RED commit `6292f16f` exists: confirmed
- GREEN commit `704b48e9` exists: confirmed

---
*Phase: 60-oracle-loop-fix*
*Completed: 2026-04-27*
