---
phase: 144-regression-execution-path-cleanup
plan: 02
subsystem: testing
tags: [go-tests, regression, yaml, audit, cross-platform, tdd]

# Dependency graph
requires:
  - phase: 143-build-plan-ceremony-restore
    provides: "YAML orchestration stripped from build.yaml/plan.yaml; plan playbooks created"
  - phase: 144-regression-execution-path-cleanup
    provides: "Phase 144 plan 01 restored anchor strings and fixed CLI flag audit"
provides:
  - "M4L regression test proving .venv fixture -> survey -> anchors -> grounding pipeline"
  - "Codex command-guide smoke tests (build, plan) verifying output without playbooks"
  - "Execution path audit test proving one conductor per workflow per platform"
affects: [future phases running cmd/ tests, v1.22 milestone closeout]

# Tech tracking
tech-stack:
  added: []
  patterns: [end-to-end-regression, execution-path-audit, smoke-test]

key-files:
  created: []
  modified:
    - cmd/codex_colonize_test.go
    - cmd/command_guide_test.go

key-decisions:
  - "Tests exercise existing implementation directly -- Phase 141-142 already delivered the pipeline"
  - "Execution path audit uses YAML metadata (runtime.command field) as programmatic source of truth"
  - "Colonize orchestration block documented as-is (intentionally not stripped in Phase 143)"
  - "CLEAN-06 coverage delegated to existing TS cross-platform-parity.test.ts (8 passing tests)"

requirements-completed: [CLEAN-01, CLEAN-02, CLEAN-05, CLEAN-06]

# Metrics
duration: 4min
completed: 2026-05-18
---

# Phase 144 Plan 02: M4L Regression, Codex Smoke Tests, and Execution Path Audit Summary

**End-to-end M4L regression test, Codex command-guide smoke tests, and programmatic execution path audit proving one conductor per workflow per platform.**

## Performance

- **Duration:** 4 min
- **Started:** 2026-05-18T22:53:17Z
- **Completed:** 2026-05-18T22:57:15Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- TestM4LRegression_VenvProducesGroundedPlan proves the full survey-to-grounding pipeline composes correctly (Phase 141 noise filtering + Phase 142 grounding gate)
- TestCommandGuideBuildSmoke and TestCommandGuidePlanSmoke verify Codex command-guide produces correct output without playbook loading
- TestExecutionPathAudit_OneConductorPerWorkflow programmatically verifies one declared execution path per workflow per platform for all 5 workflows
- Cross-platform parity confirmed: all 8 TS cross-platform-parity tests pass including ceremony alignment checks

## Task Commits

Each task was committed atomically:

1. **Task 1: Create M4L regression test and Codex command-guide smoke tests** - `dfd164f7` (test)
2. **Task 2: Document execution path audit (CLEAN-02) and verify cross-platform parity (CLEAN-06)** - `c86c40ca` (feat)

## Files Created/Modified
- `cmd/codex_colonize_test.go` - Added TestM4LRegression_VenvProducesGroundedPlan (4-step pipeline verification: survey noise exclusion, anchor extraction, grounded task validation, ungrounded task warning detection)
- `cmd/command_guide_test.go` - Added TestCommandGuideBuildSmoke, TestCommandGuidePlanSmoke (Codex output verification), and TestExecutionPathAudit_OneConductorPerWorkflow (5-subtest YAML audit)

## Decisions Made
- **TDD RED phase passed immediately:** The implementation already existed from Phases 141-142. This is expected for a capstone verification plan -- the tests prove composition, not new behavior.
- **YAML runtime.command field as audit source:** The execution path audit reads the structured `runtime.command` field from YAML metadata rather than grepping for strings in the full file. This is more robust against text changes in other sections.
- **Colonize orchestration acknowledged as-is:** The audit test verifies colonize has a valid execution path via `aether host colonize` in its orchestration block, documenting that this is intentional (Phase 143 did not strip colonize).
- **CLEAN-06 delegated to TS tests:** The existing cross-platform-parity.test.ts already comprehensively checks build ceremony alignment (8 tests covering file names, agent counts, ceremony invocations, wrapper patterns). Added a comment in the Go test noting this coverage rather than duplicating it.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All 4 requirements (CLEAN-01, CLEAN-02, CLEAN-05, CLEAN-06) satisfied
- M4L regression test provides permanent guardrail against survey noise regression
- Execution path audit test will fail if someone introduces a dual conductor
- Phase 144 is now complete; v1.22 milestone closeout can proceed

## TDD Gate Compliance

The plan frontmatter declares `tdd="true"` for Task 1. Note: the RED phase produced passing tests immediately because the implementation was already delivered in Phases 141-142. This is expected behavior for a regression/capstone verification plan. The tests prove end-to-end composition of existing features.

- `test(...)` commit: `dfd164f7` (RED -- tests added and pass against existing implementation)
- `feat(...)` commit: `c86c40ca` (GREEN -- execution path audit test added)

No separate RED failure commit was needed because the feature being tested already exists.

---
*Phase: 144-regression-execution-path-cleanup*
*Completed: 2026-05-18*
