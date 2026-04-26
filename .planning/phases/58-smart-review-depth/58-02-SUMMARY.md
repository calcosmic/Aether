---
phase: 58-smart-review-depth
plan: 02
subsystem: review-depth
tags: [tdd, dispatch-filtering, visual-rendering, colony-prime]
dependency_graph:
  requires: [58-01]
  provides: [depth-aware-build-dispatch, depth-aware-continue-review, renderReviewDepthLine, colony-prime-review-depth]
  affects: [cmd/codex_build.go, cmd/codex_continue.go, cmd/codex_visuals.go, cmd/colony_prime_context.go]
tech_stack:
  added: []
  patterns: [parameter-threading, result-map-depth-propagation, visual-depth-line]
key_files:
  created: []
  modified:
    - cmd/codex_build.go
    - cmd/codex_continue.go
    - cmd/codex_visuals.go
    - cmd/colony_prime_context.go
    - cmd/codex_workflow_cmds.go
    - cmd/codex_continue_finalize.go
    - cmd/continue_wrapper_ceremony_test.go
    - cmd/review_depth_test.go
decisions:
  - Review depth computed once in runCodexBuildWithOptions/runCodexContinue and threaded through to dispatch and visual functions
  - Light-mode continue review returns empty specs slice (not nil), producing Passed=true with zero dispatches
  - Visual depth line uses exact format from plan: "Review depth: light (Phase N of M -- final phase gets full review)"
  - Colony-prime review depth section at priority 6 (after blockers/pheromones, before lower-priority sections)
metrics:
  duration: 8m
  completed: "2026-04-27"
  tasks: 2
  files: 8
  tests_added: 7
---

# Phase 58 Plan 02: Dispatch Wiring and Visual Output Summary

Wire review depth into build dispatch planner, continue review dispatcher, visual renderers, and colony-prime context injection so intermediate phases get fast review and final/security phases get full review.

## What Was Built

Two wiring layers that connect the pure depth resolution logic from Plan 01 to every place that needs it:

1. **Build dispatch filtering** -- `plannedBuildDispatchesForSelection` now accepts a `ReviewDepth` parameter. Light-mode builds skip Measurer entirely and include Chaos only for 30% deterministically-sampled phase IDs. Heavy-mode builds include both Measurer (when build depth is deep/full) and Chaos (when build depth is full). The `--light` and `--heavy` flags flow from the CLI through `codexBuildOptions` into the dispatch planner.

2. **Continue review filtering** -- `plannedContinueReviewDispatches` accepts `ReviewDepth` and returns an empty specs slice in light mode, causing the review wave to produce zero dispatches and report `Passed=true`. Heavy mode spawns all 3 review agents (gatekeeper, auditor, probe). Flags flow through `codexContinueOptions`.

3. **Visual depth display** -- `renderReviewDepthLine` produces formatted strings showing the review depth with phase position context. All build and continue visual renderers include this line after the phase name.

4. **Colony-prime context** -- `buildColonyPrimeOutput` includes a `review_depth` section at priority 6 when the colony has a valid current phase, informing workers of their review depth.

## TDD Gate Compliance

| Gate | Commit | Hash |
|------|--------|------|
| RED (dispatch + visual tests) | test(58-02): add failing tests for depth-aware dispatch filtering | 0ae6efc5 |
| GREEN (dispatch wiring) | feat(58-02): wire review depth into build and continue dispatch planning | 6ec3cb15 |
| GREEN (visual + context) | feat(58-02): add review depth display and colony-prime context injection | 3e7c10c4 |

All three gate commits present and verified in git log.

## Commits

| Hash | Message |
|------|---------|
| 0ae6efc5 | test(58-02): add failing tests for depth-aware dispatch filtering |
| 6ec3cb15 | feat(58-02): wire review depth into build and continue dispatch planning |
| 3e7c10c4 | feat(58-02): add review depth display and colony-prime context injection |

## Deviations from Plan

None - plan executed exactly as written.

## Self-Check

- cmd/codex_build.go: modified (review depth threading, result map)
- cmd/codex_continue.go: modified (review depth threading, result maps)
- cmd/codex_visuals.go: modified (renderReviewDepthLine, depth display)
- cmd/colony_prime_context.go: modified (review depth section)
- cmd/codex_workflow_cmds.go: modified (flag passing, visual callers)
- cmd/codex_continue_finalize.go: modified (visual caller update)
- cmd/review_depth_test.go: modified (7 new tests)
- cmd/continue_wrapper_ceremony_test.go: modified (signature update)
- All 3 gate commits verified in git log
- All 43 review-depth tests passing
- Binary builds cleanly

## Self-Check: PASSED

## Known Stubs

None.

## Threat Flags

None. No new network endpoints, auth paths, or trust boundaries beyond those documented in the plan threat model.
