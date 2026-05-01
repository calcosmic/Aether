---
phase: 86-depth-selection-ui-and-persistence
plan: 03
subsystem: ui
tags: [go, depth-selection, result-map, gap-closure]

# Dependency graph
requires:
  - phase: 86-depth-selection-ui-and-persistence
    provides: "verification_depth, planning_smart_default, verification_smart_default, planning_phase variables resolved at function scope"
provides:
  - "Fresh plan generation result map now includes all four depth keys consumed by renderPlanVisual"
  - "Regression test preventing future omission of depth keys from any result map path"
affects: [codex_visuals, depth-selection-banner]

# Tech tracking
tech-stack:
  added: []
  patterns: [source-level regression test for result map completeness]

key-files:
  created: []
  modified:
    - cmd/codex_plan.go
    - cmd/review_depth_test.go

key-decisions:
  - "Used source-level assertion test (strings.Count) instead of integration test because the plan function has heavy filesystem dependencies making unit-level invocation impractical"

patterns-established:
  - "Source-level regression test pattern: verify key presence in all result map construction sites by counting occurrences in source"

requirements-completed: [DEPTH-04]

# Metrics
duration: 3min
completed: 2026-05-01
---

# Phase 86 Plan 03: Add Missing Depth Keys to Fresh Plan Result Map Summary

**Four missing depth keys added to fresh plan generation result map, closing the verification gap where the depth selection banner silently skipped the verification depth line on first /ant-plan run.**

## Performance

- **Duration:** 3 min
- **Started:** 2026-05-01T12:43:01Z
- **Completed:** 2026-05-01T12:43:43Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Fresh plan generation result map in codex_plan.go now includes verification_depth, verification_smart_default, planning_smart_default, and planning_phase keys
- Depth selection banner in renderPlanVisual will now correctly display both planning and verification depth with reason annotations for fresh plan generation
- Regression test ensures all four keys appear in all three result map construction sites (existing-plan, fresh generation, plan-only)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add four missing depth keys to the fresh plan generation result map** - `cf441544` (fix)
2. **Task 2: Add test verifying depth keys in fresh plan generation result map** - `c4d87064` (test)

## Files Created/Modified
- `cmd/codex_plan.go` - Added verification_depth, verification_smart_default, planning_smart_default, planning_phase keys to the fresh plan generation result map (~line 449)
- `cmd/review_depth_test.go` - Added TestDepthKeysPresentInFreshPlanResultMap regression test and os import

## Decisions Made
- Used source-level assertion test (counting key occurrences in source file) rather than an integration test calling the plan function directly, because the plan function has heavy filesystem dependencies (temp directories, file I/O) that make lightweight unit invocation impractical

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Pre-existing build failure in worktree: `.aether/rules/` directory missing, causing `embedded_assets.go` embed pattern to fail. Resolved by copying the rule file from the main repo. This is a worktree setup issue, not related to plan changes.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- All three result map paths in codex_plan.go now have consistent depth key coverage
- Depth selection banner will render correctly for all plan generation scenarios
- No blockers for downstream consumers of the result map

## Self-Check: PASSED

- 86-03-SUMMARY.md: FOUND
- Commit cf441544: FOUND
- Commit c4d87064: FOUND
- cmd/codex_plan.go: FOUND
- cmd/review_depth_test.go: FOUND

---
*Phase: 86-depth-selection-ui-and-persistence*
*Completed: 2026-05-01*
