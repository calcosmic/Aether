---
phase: 200-iterative-planning
plan: 49
subsystem: build-lifecycle-testing
tags: [go, plan-authority, build-start, fixtures, process-liveness, tdd]

# Dependency graph
requires:
  - phase: 200-47
    provides: canonical receipt-backed build-start paths and attempt bindings
  - phase: 200-48
    provides: durable partial-finalize and recovery evidence semantics
provides:
  - explicit approved-specification and accepted-plan fixtures for build command tests
  - exact phase, execution owner, dispatch mode, and process-liveness fixture controls
  - canonical start receipts for core build, finalizer, check-in, and recovery scenarios
affects: [build-fixtures, build-finalize, check-in, resume-recovery, phase-200-final-gates]

# Tech tracking
tech-stack:
  added: []
  patterns: [canonical fixture transactions, exact accepted-plan projection, explicit process liveness]

key-files:
  created: []
  modified:
    - cmd/build_start_test_helpers_200_test.go
    - cmd/codex_build_test.go
    - cmd/ceremony_team_checkin_test.go
    - cmd/orchestrator_boundary_guidance_test.go
    - cmd/build_attempt_test.go

key-decisions:
  - "Accepted build fixtures use the real specification approval, candidate coordination, owner acceptance, and commitBuildStart boundaries; mutable execution status is synchronized without replacing immutable authority bindings."
  - "New execution fixtures name phase, owner, mode, and live/dead process state explicitly; the fixed dead PID remains available only through an explicit dead-process choice."
  - "Tests that need a trusted completed manifest obtain it from commitBuildStart and derive authority fields inside the transaction helper instead of authoring those bindings or journal files directly."

patterns-established:
  - "Authority before behavior: command fixtures establish an approved specification and exact accepted phase before exercising build behavior."
  - "Honest liveness: use the current test process for live recovery and an explicitly verified-dead identity for abandoned attempts."

requirements-completed: [CEC-03, PLAN-05, PLAN-06]

# Metrics
duration: 48min
completed: 2026-09-09
---

# Phase 200 Plan 49: Canonical Build Fixture Authority Summary

**Core build tests now cross the same approved-specification, accepted-plan, and receipt-backed start boundary as production, with exact phase identity and honest live/dead process semantics.**

## Performance

- **Duration:** 48 min
- **Started:** 2026-09-09T12:26:42Z
- **Completed:** 2026-09-09T13:14:23Z
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments

- Added an accepted-build fixture that runs the real draft, specification approval, route coordination, explicit owner acceptance, and canonical build-start transaction.
- Migrated ten core build cases without weakening D-16, including verification-depth forwarding, task-scoped redispatch, trusted prior-task repair, job proposals, and external completion.
- Corrected the check-in phase-5/phase-1 mismatch, bound the finalizer manifest to an exact plan-only start receipt, and made live recovery use the current process ID.
- Added source ratchets that keep the migrated core and boundary fixtures on the canonical authority path.

## Task Commits

Each TDD gate and implementation unit was committed atomically:

1. **Task 1 RED: Define explicit phase, authority, and liveness expectations** - `90a589eb` (test)
2. **Task 1 GREEN A: Enforce explicit fixture identity and live/dead process choices** - `cb95f9ed` (test)
3. **Task 1 GREEN B: Build exact approved and accepted test plans** - `12ae64e1` (test)
4. **Task 2 RED: Require canonical accepted authority in all ten core cases** - `14156d96` (test)
5. **Task 2 GREEN: Migrate core build cases and trusted manifest setup** - `5f154a29` (test)
6. **Task 3 RED: Require canonical boundary and liveness markers** - `3f776192` (test)
7. **Task 3 GREEN: Migrate check-in, finalizer, and recovery fixtures** - `27855fe0` (test)

## Files Created/Modified

- `cmd/build_start_test_helpers_200_test.go` - Provides exact accepted-plan setup, explicit phase/owner/mode/liveness options, receipt assertions, and synchronized execution-fact projection.
- `cmd/codex_build_test.go` - Routes the ten owned core build scenarios through approved and accepted authority, including a canonical trusted prior-phase manifest.
- `cmd/ceremony_team_checkin_test.go` - Represents phase 5 with four honestly completed prerequisites and requests the same accepted phase.
- `cmd/orchestrator_boundary_guidance_test.go` - Supplies build-finalize with the plan-only manifest bound by canonical build start.
- `cmd/build_attempt_test.go` - Pins canonical fixture usage and injects the current process for live-attempt recovery.

## Decisions Made

- Kept D-16 strict. Fixture convenience never grants production authority, adds a legacy bypass, or auto-accepts arbitrary runtime state.
- Treated plan definition and execution status separately: the accepted revision remains immutable while the helper synchronizes only lifecycle status required by task-scoped and prior-phase scenarios.
- Derived plan authority, revision ID, and plan hash inside the build-start helper for custom semantic manifests. Tests can describe completed task evidence, but cannot forge the accepted binding.
- Preserved phase 5 in the pending-decision check-in test because light-mode sampling depends on that identity; four completed prerequisite phases make phase 5 genuinely reachable.

## Deviations from Plan

None - the helper expansion and all fixture migrations stayed within the five planned test files and retained current product gates.

## Issues Encountered

- CLI-based fixtures initially resolved accepted candidate artifacts relative to the source checkout. Binding their working directory to the temporary accepted repository made physical repository authority explicit.
- The prior-phase repair case could not keep its old direct manifest and claims writes because they left the accepted state baseline stale. A direct `commitBuildStart` receipt now writes the trusted manifest before the test deliberately introduces only the stale task-status facts it is meant to repair.
- The plan-only finalizer fixture previously carried contradictory empty-owner/plan-only metadata. The canonical transaction now derives its binding with `host-queen` / `plan-only` before finalization.
- The installed progress updater found no Markdown-body progress field, and the requirement marker does not parse this milestone's bold-ID checkbox format. State position/progress metadata was preserved manually; `CEC-03`, `PLAN-05`, and `PLAN-06` were already checked complete.

## TDD Gate Compliance

- **Task 1 RED (`90a589eb`):** the helper lacked explicit phase/liveness controls and mismatch guards.
- **Task 1 GREEN (`cb95f9ed`, `12ae64e1`):** exact accepted authority, real live PID, explicit dead PID, and mismatch refusals pass.
- **Task 2 RED (`14156d96`):** all ten named core fixtures failed the canonical-authority source ratchet.
- **Task 2 GREEN (`5f154a29`):** all ten reach their original assertions through accepted authority and canonical start artifacts.
- **Task 3 RED (`3f776192`):** check-in, boundary-finalize, and recovery fixtures lacked exact authority/liveness markers and their focused behaviors failed.
- **Task 3 GREEN (`27855fe0`):** phase identity, manifest binding, and live recovery are explicit and the three focused behaviors pass.

## Verification

- `go test ./cmd -run '^TestCanonicalBuildStartFixture(Authority|LiveProcess|RefusesMismatch)200$' -count=1` - PASS (`ok`, 12.969s).
- `go test ./cmd -run '^(TestBuildWritesDispatchArtifactsAndUpdatesState|TestDispatchEntryCarriesBriefPath|TestBuildPlanOnlyCLIForwardsVerificationDepth|TestBuildPlanOnlyHeavyReviewAllowsPolicyMeasurerAndChaos|TestBuildPlanOnlyKeepsRoutineUIQueenSelectionLean|TestBuildCLIForwardsVerificationDepth|TestBuildFinalizeRecordsExternalTaskResultsForContinue|TestBuildSupportsTaskScopedRedispatch|TestBuildRepairsCompletedPriorPhaseTasksFromTrustedManifest|TestBuildJobProposalRoundTrip)$' -count=1` - PASS (`ok`, 47.776s).
- `go test ./cmd -run '^Test(PendingDecisionStillRendersFullCheckinCard|BuildFinalizeAddsOrchestratorBoundaryGuidance|ResumeDashboardDoesNotRedispatchLiveBuildProcess)$' -count=1` - PASS (`ok`, 18.733s).
- Combined exact Plan 49 regex - PASS (`ok`, 79.698s).
- `gsd-sdk query verify.key-links .planning/phases/200-iterative-planning/200-49-PLAN.md --raw` - PASS (`1/1 valid`).
- `git diff --check` - PASS for all Plan 49 implementation and test commits.

## Known Stubs

None. Empty slices and zero values in the changed files are assertion inputs or canonical serialization fixtures, not user-facing placeholders.

## Threat Flags

None. All changed files are tests; the fixture-to-execution and process-spoofing boundaries are the two threats named in the plan and are now guarded by accepted authority, transaction receipts, and explicit liveness choices. No endpoint, schema, dependency, credential path, or new production filesystem authority was introduced.

## User Setup Required

None - no dependency, credential, service, or configuration change is required.

## Next Phase Readiness

- Plans 50-55 can rely on core build failures representing product behavior rather than pre-D-16 fixture setup.
- Final Phase 200 gates can distinguish genuinely live attempts from dead recovery cases without PID folklore.
- No known blocker remains from Plan 49; the intentionally deferred full-repository failures remain owned by later gap-closure plans.

## Self-Check: PASSED

- All five modified test files and this summary exist.
- TDD commits `90a589eb`, `cb95f9ed`, `12ae64e1`, `14156d96`, `5f154a29`, `3f776192`, and `27855fe0` exist in repository history.
- All three exact task gates, the combined Plan 49 gate, both source ratchets, and the declared key link pass.
- `.planning/config.json`, `.gsd/`, and untracked Phase 199 `199-PATTERNS.md` were neither staged nor changed by Plan 49.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-09*
