---
phase: 141-survey-noise-filter-source-anchors
plan: 02
subsystem: survey-planning
tags: [source-anchors, survey, planning, go, codegraph]

# Dependency graph
requires:
  - phase: 141-01
    provides: pkg/codegraph/scan_filter.go with ShouldSkipDir, ShouldSkipFile
provides:
  - extractSourceAnchors function producing up to 50 repo-owned source file paths
  - SourceAnchors field on codexWorkspaceFacts and codexSurveyContext
  - anchors.json written to .aether/data/survey/ during survey output
  - anchors.json read by loadCodexSurveyContext for planner consumption
affects: [codex_plan, codex_colonize, grounded-planning]

# Tech tracking
tech-stack:
  added: []
  patterns: [source-anchor-extraction, survey-to-planner-data-pipeline]

key-files:
  created: []
  modified:
    - cmd/codex_colonize.go
    - cmd/codex_colonize_test.go
    - cmd/codex_plan.go

key-decisions:
  - "Source anchors sorted by path depth then alphabetically -- shallowest files first gives planners the most stable targets"
  - "Capped at 50 anchors to avoid token budget bloat in planner context"
  - "Minified files (.min.js, .min.css) excluded via shared ShouldSkipFile"

patterns-established:
  - "Source anchors flow: surveyWorkspace -> extractSourceAnchors -> facts.SourceAnchors -> writeSurveyCompatibilityJSON -> anchors.json -> loadCodexSurveyContext -> ctx.SourceAnchors"
  - "Shared ScanFilter (ShouldSkipDir, ShouldSkipFile) used at every directory-walking boundary"

requirements-completed: [GROUND-03, GROUND-04]

# Metrics
duration: 13min
completed: 2026-05-18
---

# Phase 141 Plan 2: Source Anchors Summary

**extractSourceAnchors produces top 50 repo-owned source file paths sorted by depth, flowing from survey output through anchors.json into planner context**

## Performance

- **Duration:** 13 min
- **Started:** 2026-05-18T19:25:50Z
- **Completed:** 2026-05-18T19:38:29Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- extractSourceAnchors walks repo, filters noise/test/config, produces up to 50 paths sorted by depth then alphabetically
- Survey output writes anchors.json to .aether/data/survey/ with source_anchors, anchor_count, summary
- loadCodexSurveyContext reads anchors.json into codexSurveyContext.SourceAnchors for planner consumption
- Full data pipeline: survey -> anchors.json -> planner context

## Task Commits

Each task was committed atomically with TDD (test -> feat):

1. **Task 1: Add extractSourceAnchors, wire SourceAnchors through survey output**
   - `ee5051ab` (test) - 7 failing tests for anchor extraction
   - `6c3d6780` (feat) - Implementation: extractSourceAnchors, SourceAnchors field, anchors.json output

2. **Task 2: Wire SourceAnchors into planner context via loadCodexSurveyContext**
   - `e6cbb7fc` (test) - 2 failing tests for planner context anchor reading
   - `39e2e3e6` (feat) - SourceAnchors field, anchors.json read, uniqueSortedStrings dedup

## Files Created/Modified
- `cmd/codex_colonize.go` - Added SourceAnchors field to codexWorkspaceFacts, extractSourceAnchors function, anchors.json in writeSurveyCompatibilityJSON, surveyWorkspace wiring
- `cmd/codex_colonize_test.go` - Added 9 new tests: 7 for extractSourceAnchors (basic, cap, noise exclusion, test exclusion, minified exclusion, depth+alpha sort, survey JSON output), 2 for loadCodexSurveyContext (includes anchors, empty when no file)
- `cmd/codex_plan.go` - Added SourceAnchors field to codexSurveyContext, anchors.json reading in loadCodexSurveyContext, uniqueSortedStrings dedup

## Decisions Made
- Anchors sorted by path depth (shallowest first) then alphabetically -- gives planners the most stable, top-level file targets
- Capped at 50 anchors to stay within planner token budget
- Minified files excluded via shared ShouldSkipFile (already handles .min.js, .min.css)
- SourceAnchors initialized as `[]string{}` (not nil) in codexSurveyContext -- nil-safe for downstream consumers

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None - all planned work completed cleanly.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Source anchor pipeline complete: survey -> anchors.json -> planner context
- Grounded planning infrastructure ready: planners now receive concrete file paths
- All existing tests pass, no regressions
- Go vet clean on modified packages

## Self-Check: PASSED

All modified files verified present. All 4 commits verified in git log. Zero uncommitted changes.

---
*Phase: 141-survey-noise-filter-source-anchors*
*Completed: 2026-05-18*
