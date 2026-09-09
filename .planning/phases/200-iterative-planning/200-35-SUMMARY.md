---
phase: 200-iterative-planning
plan: 35
subsystem: build-lifecycle-tests
tags: [go, tdd, test-fixtures, transactions, authority-binding, ast-ratchet]

requires:
  - phase: 200-33
    provides: Canonical receipt-last atomic build-start transaction
  - phase: 200-34
    provides: All production build-start caller families migrated to the canonical transaction
provides:
  - Transaction-only accepted-authority build-start fixture with explicit conditional effects
  - First bounded migration of core attempt, preflight, autopilot, floor-fix, and codex-build tests
  - Six-file AST ratchet preventing legacy or reconstructed partial build starts
affects: [plan-36-helper-retirement, build-attempt-tests, preflight, autopilot, floor-fix, codex-build]

tech-stack:
  added: []
  patterns: [accepted-authority test fixture, canonical transition seeding, AST migration ratchet]

key-files:
  created:
    - cmd/build_start_test_helpers_200_test.go
  modified:
    - cmd/build_attempt_test.go
    - cmd/preflight_phase_198_3_test.go
    - cmd/autopilot_checkpoints_test.go
    - cmd/floor_fix_attempt_test.go
    - cmd/codex_build_test.go

key-decisions:
  - "The fixture exposes every conditional and nondeterministic build-start input explicitly and returns the durable receipt, attempt, latest pointer, state, and manifest used by assertions."
  - "Tests that need an arbitrary attempt status must commit a canonical start and then use transitionBuildAttempt; no fixture may write attempt or latest-pointer JSON directly."
  - "Public-build fixtures seed their required goal through the repository mutation session, preserving the accepted-authority fixture while reaching the intended preflight boundary."
  - "Attempt-history assertions temporarily filter for valid attempt IDs because Plan 36 owns the production receipt-enumeration correction."

patterns-established:
  - "Canonical test start: seed accepted prerequisites, assemble an explicit buildStartRequest, invoke commitBuildStart once, and read all durable evidence back from its receipt."
  - "Fixture migration ratchet: AST inspection rejects legacy start helpers and any local function that reconstructs the attempt-plus-latest-pointer pair."

requirements-completed: [PLAN-06]

duration: 24min
completed: 2026-09-09
---

# Phase 200 Plan 35: Canonical Build-Start Fixture Migration Summary

**Core build lifecycle tests now create accepted-authority attempts through the real atomic build-start transaction, with a local AST ratchet preventing the old partial shortcuts from returning.**

## Performance

- **Duration:** 24 min
- **Started:** 2026-09-09T02:46:40Z
- **Completed:** 2026-09-09T03:10:09Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- Added `commitTestBuildStart`, a transaction-only fixture that constructs accepted authority in a physically contained temporary repository and delegates every start target to `commitBuildStart`.
- Migrated all core build-attempt tests while preserving record-field, latest-pointer, transition, recovery, replay, and history assertions.
- Migrated preflight, autopilot checkpoint, floor-fix, and codex-build attempt seeders, including arbitrary-status setup through the production transition API.
- Preserved selected-task and matching-dispatch coverage in the autopilot fixture rather than weakening the original behavioral scenario.
- Added an AST ratchet across the six owned files that rejects legacy helper calls and reconstructed direct attempt-plus-pointer seeders.

## Task Commits

Each task followed a fail-first RED/GREEN boundary and was committed atomically:

1. **Task 1 RED: Require the canonical build-start fixture** - `1be1f601` (test)
2. **Task 1 GREEN: Add the transaction-only fixture and migrate core attempts** - `7729625e` (test)
3. **Task 2 RED: Reject legacy and reconstructed partial fixture starts** - `3e867a3e` (test)
4. **Task 2 GREEN: Migrate preflight, autopilot, floor-fix, and codex-build fixtures** - `d76088df` (test)
5. **Coverage retention: Preserve checkpoint selected-task coverage** - `d1588fd4` (test)

## Files Created/Modified

- `cmd/build_start_test_helpers_200_test.go` - Canonical accepted-authority start options/results, durable receipt readback, prerequisite goal seeding, and six-file AST ratchet.
- `cmd/build_attempt_test.go` - Core attempt, pointer, transition, recovery, replay, and history tests migrated from `beginBuildAttempt` to the transaction fixture.
- `cmd/preflight_phase_198_3_test.go` - Active-build preflight setup now uses a canonical start followed by the production status transition.
- `cmd/autopilot_checkpoints_test.go` - Journal-bound checkpoint manifest now comes from the transaction, with explicit selected-task and dispatch inputs retained.
- `cmd/floor_fix_attempt_test.go` - Original and automatic fix attempts now use accepted-authority canonical starts while preserving append-only behavior assertions.
- `cmd/codex_build_test.go` - Arbitrary-status worker-name fixtures now use canonical start plus `transitionBuildAttempt`, with no direct start-target writes.

## Decisions Made

- All inputs whose defaults could hide test behavior—authority, clocks, IDs, process/platform data, selected tasks, dispatches, ownership, manifest/checkpoint/completion/claims, lifecycle promotion, provenance, reviewer effects, stale paths, and the pre-root callback—are explicit fixture options.
- The helper returns durable evidence read from the repository rather than returning only its request, so existing assertions continue to test persisted behavior.
- The accepted-authority fixture remains minimal; tests that exercise public build preflight add the required goal through the repository mutation API instead of editing lifecycle state directly.
- Attempt-history checks filter out invalid IDs until Plan 36 corrects production enumeration of receipt siblings.

## Verification

- Task 1 focused `TestBuildAttempt` suite - passed (6/6).
- Final Plan 35 focused suite - passed (17/17).
- Final focused suite repeated three times - passed (51/51).
- Broader directly touched test set - passed (15/15).
- Broader directly touched test set under `go test -race` - passed (15/15).
- Plan 35 focused suite under `go test -race` - passed (17/17).
- AST fixture ratchet - passed (1/1).
- `go vet ./cmd`, implementation-range `git diff --check`, exact six-file ownership, and no-deletion checks - passed.
- The known repository-wide `go test ./...` command was not run because the execution brief explicitly excludes its approximately 11-minute runtime.

## TDD Gate Compliance

- Task 1 RED failed because `commitTestBuildStart` and its explicit option/result contract did not exist; GREEN added it around the real transaction and made all six core attempt tests pass.
- Task 2 RED failed on four legacy helper-call sites and the codex-build direct attempt-plus-pointer seeder; GREEN migrated each family and made the AST ratchet pass.
- The focused checkpoint regression then exposed that selected-task setup had become implicit; `d1588fd4` restored an explicit selected task and matching dispatch, with the affected test passing afterward.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Seeded the active-colony goal through the repository mutation session**
- **Found during:** Task 2 public preflight verification
- **Issue:** The reused accepted-candidate prerequisite intentionally carries a zero-value goal, so public build preflight stopped at `No colony initialized` before reaching the provider failure asserted by the test.
- **Fix:** Added a test-only goal seeder using `withPlanningMutationSession` and the existing canonical state writer, preserving repository containment and avoiding all build-start targets.
- **Files modified:** `cmd/build_start_test_helpers_200_test.go`
- **Commit:** `d76088df`

## Deferred Issues

- `listBuildAttemptsForPhase` currently decodes `.start-receipt.json` siblings as empty attempt records. Migrated history assertions therefore filter for `validBuildAttemptID` while still requiring exactly the original and fix journals. This is a production compatibility issue already owned by Plan 36; no out-of-scope production file was changed here.
- The installed GSD state synchronizer again stripped Phase 200 provenance fields and temporarily advanced the display to numeric Plan 36. The scoped provenance closeout restores those fields against the normal metadata commit and points Current Position to Plan 38, the actual next incomplete plan in Wave 22; Plan 36 remains correctly blocked on that wave.

## Known Stubs

None.

## Threat Flags

None - all changes are test-only and introduce no new endpoint, authentication, filesystem trust boundary, or schema surface beyond the plan's fixture-to-transaction threat model.

## User Setup Required

None - no packages, credentials, or external services were added.

## Next Phase Readiness

- Plan 36 can use the AST ratchet as its migration checklist, move remaining fixtures to `commitTestBuildStart`, correct receipt enumeration, and retire the production compatibility helpers.
- No Plan 35 behavior, race, verification, ownership, or scope blocker remains.

## Self-Check: PASSED

- All six declared test files and this summary exist.
- Commits `1be1f601`, `7729625e`, `3e867a3e`, `d76088df`, and `d1588fd4` exist in repository history.
- The five implementation commits change exactly the six owned test files, delete no tracked file, and pass focused, repeated, race, vet, and diff-hygiene gates.
- Summary commit `273d430c` and normal planning-metadata commit `45886462` exist; STATE provenance names that exact metadata boundary and preserves Wave 22 ordering.
- Protected `.planning/config.json`, `.gsd/`, and Phase 199 PATTERNS dirt remains present and unstaged.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-09*
