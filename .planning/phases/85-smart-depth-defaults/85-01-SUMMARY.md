---
phase: 85-smart-depth-defaults
plan: 01
subsystem: depth-system
tags: [go, planning-depth, verification-depth, risk-classification, smart-defaults]

# Dependency graph
requires:
  - phase: 84-verification-depth-extension
    provides: "resolveVerificationDepth, resolveVerificationDepthFlag, ReviewDepth types, 3-level verification depth system"
provides:
  - "phasePositionLevel: classifies phases as early/intermediate/late/final"
  - "collectPhaseText: extracts all analyzable text from phase fields"
  - "phaseRiskLevel: classifies risk as high/medium/low from keywords"
  - "resolveSmartPlanningDepth: auto-selects planning depth from position + risk"
  - "resolveSmartVerificationDepth: auto-selects verification depth from position + risk"
  - "securityRiskKeywords: 10 security-related keyword substrings"
  - "blastRadiusKeywords: 10 blast-radius keyword substrings"
affects: [85-02-smart-depth-wiring, build-command, continue-command, plan-command]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Safer principle: higher depth wins when position and risk signals disagree"
    - "Keyword risk classification: hardcoded constant lists, no external input"
    - "Text collection: concatenate all phase fields then lowercase for matching"

key-files:
  created: []
  modified:
    - cmd/review_depth.go
    - cmd/review_depth_test.go

key-decisions:
  - "collectPhaseText joins with spaces and lowercases; trailing spaces from empty fields are harmless for substring matching"
  - "Phase position thresholds: 25% for early, 75% for late, exact match for final"
  - "Risk signal overrides position signal (security on early phase gets deep/heavy)"

patterns-established:
  - "Safer principle: when position and risk disagree, use the higher depth"
  - "Two-tier keyword lists: security (high) and blast-radius (medium)"

requirements-completed: [DEPTH-03]

# Metrics
duration: 6min
completed: 2026-04-30
---

# Phase 85 Plan 01: Smart Depth Defaults Summary

**Smart depth resolution functions that auto-select planning and verification depth from phase position and code change risk signals, using keyword-based risk classification with a safer-principle override**

## Performance

- **Duration:** 6 min
- **Started:** 2026-04-30T22:07:05Z
- **Completed:** 2026-04-30T22:12:33Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Seven new pure functions for smart depth resolution in `cmd/review_depth.go`
- Two keyword lists (security + blast-radius) for risk classification
- Ten comprehensive test functions covering all edge cases (44 subtests total)
- Risk signal overrides position signal -- early phase with security keyword gets deep/heavy, not light

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement smart depth resolution functions** - `22b94a5c` (feat)
2. **Task 2: Write comprehensive tests for smart depth functions** - `5ed70233` (test)

## Files Created/Modified
- `cmd/review_depth.go` - Added 7 functions and 2 keyword lists for smart depth defaults
- `cmd/review_depth_test.go` - Added 10 test functions with 44 subtests covering all smart depth logic

## Decisions Made
- `collectPhaseText` joins parts with spaces and lowercases the result. Empty fields (Description, nil slices) produce trailing spaces which are harmless for `strings.Contains` keyword matching. Tests use `Contains` rather than exact match to avoid brittleness.
- Position thresholds use float comparison: `phaseID <= total*0.25` for early, `phaseID >= total*0.75` for late. Boundary cases like `phaseID=2, total=8` where `2.0 == 8*0.25` correctly classify as "early".
- No new dependencies -- uses only `strings` from stdlib plus existing `colony` package types.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Pre-existing `embedded_assets.go` build failure (`all:.aether/rules: no matching files found`) required creating a temporary `.gitkeep` in `.aether/rules/` to unblock `go test ./cmd/`. The directory was cleaned up after testing. This is a pre-existing issue on the base commit, not caused by this plan.
- Two pre-existing test failures (`TestIntegrityDetectSourceContext`, `TestQueenWisdomHygiene`) occur in worktree environments and are unrelated to this plan's changes.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Plan 02 (smart-depth-wiring) can import and call `resolveSmartPlanningDepth` and `resolveSmartVerificationDepth` directly
- All new functions are exported (lowercase package-internal) and tested
- No wiring to command paths was done in this plan -- that is Plan 02's scope

## Self-Check: PASSED

- FOUND: cmd/review_depth.go
- FOUND: cmd/review_depth_test.go
- FOUND: .planning/phases/85-smart-depth-defaults/85-01-SUMMARY.md
- FOUND: 22b94a5c (Task 1 commit)
- FOUND: 5ed70233 (Task 2 commit)

---
*Phase: 85-smart-depth-defaults*
*Completed: 2026-04-30*
