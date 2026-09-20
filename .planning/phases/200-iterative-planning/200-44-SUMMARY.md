---
phase: 200-iterative-planning
plan: 44
subsystem: testing
tags: [go, ast, worktrees, repository-containment, test-isolation]

# Dependency graph
requires:
  - phase: 200-41
    provides: repository-authorized command-test binding and authority-first cleanup
provides:
  - repository-bound shelf and worktree command fixtures with exact cleanup
  - an existing-root form of the shared command-test binder for real Git fixtures
  - an AST-aware zero-tolerance ratchet for self-restoring AETHER_ROOT defers
affects: [200-40, final-gate, command-tests, worktree-safety]

# Tech tracking
tech-stack:
  added: []
  patterns: [repository-authorized test stores, testing.T environment restoration, AST source ratchets]

key-files:
  created: []
  modified:
    - cmd/shelf_test.go
    - cmd/worktree_merge_gate_test.go
    - cmd/worktree_operator_destruction_test.go
    - cmd/worktree_test.go
    - cmd/testing_main_test.go

key-decisions:
  - "Real Git fixtures reuse the shared command-test authority contract against their existing temporary repository root."
  - "The cleanup ratchet matches executable Go AST nodes rather than source text, so comments and strings cannot trigger or evade it."
  - "Worktree cleanup and operator-destruction behavior remains unchanged; only test-owned repository authority and teardown changed."

patterns-established:
  - "Existing-root command fixtures: bindCommandTestRepositoryAt aligns root, data, store, tracer, Cobra state, and authority closure."
  - "Cleanup ratchets: report exact AST source positions and prove the matcher with an in-memory positive sample."

requirements-completed: [CEC-03, PLAN-06]

# Metrics
duration: 17min
completed: 2026-09-09
---

# Phase 200 Plan 44: Shelf and Worktree Cleanup Ratchet Summary

**Shelf and worktree fixtures now release deleted repository authority exactly, with a syntax-aware ratchet preventing all 120 migrated cleanup defects from returning.**

## Performance

- **Duration:** 17 min
- **Started:** 2026-09-09T09:40:11Z
- **Completed:** 2026-09-09T09:57:25Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Migrated the remaining shelf, merge-gate, operator-destruction, and worktree fixtures away from self-restoring environment defers.
- Extended the Plan 41 binder to existing temporary Git repositories while retaining one physical root/data/store/tracer authority and closing its directory handle before temporary-root deletion.
- Kept positive test-owned worktree registration and every dirty, unmerged, and unrecorded-work refusal assertion intact.
- Added an AST-aware ratchet that detects the exact executable anti-pattern, ignores matching comments and strings, and reports every offending file and line.

## Task Commits

Each task was committed atomically through its RED and GREEN TDD gates:

1. **Task 1 RED: Reproduce worktree fixture root leakage** - `e286a5d5` (test)
2. **Task 1 GREEN: Isolate shelf and worktree fixtures** - `b2196a1b` (test)
3. **Task 2 RED: Define the cleanup ratchet contract** - `09ad2017` (test)
4. **Task 2 GREEN: Enforce the AST-aware cleanup ratchet** - `5bb1ab92` (test)

## Files Created/Modified

- `cmd/shelf_test.go` - Routes all shelf command fixtures through the repository-authorized test binder.
- `cmd/worktree_merge_gate_test.go` - Binds real Git merge-gate fixtures to their owning temporary repository.
- `cmd/worktree_operator_destruction_test.go` - Isolates destructive-operation fixtures without changing their fail-closed assertions.
- `cmd/worktree_test.go` - Gives shared worktree fixtures exact environment cleanup and carries the deleted-root regression proof.
- `cmd/testing_main_test.go` - Adds existing-root repository binding, authority closure, and the AST cleanup ratchet.

## Verification

- `go test ./cmd -run '^TestWorktreeFixtureRestoresRepositoryAuthority200$' -count=1` - RED before migration, then PASS.
- `go test ./cmd -run '^Test(Shelf|Worktree)' -count=2` - PASS.
- `go test ./cmd -run '^TestNoSelfRestoringAetherRootCleanup200$' -count=1` - RED with the matcher stub, then PASS with one injected executable sample and zero repository sites.
- `go test ./cmd -run '^Test(NoSelfRestoringAetherRootCleanup200|Shelf|Worktree)' -count=2` - PASS.
- `git worktree list --porcelain` plus test-branch inventory after repeated execution - PASS; only the owner checkout remained and no test branch survived.

## Decisions Made

- Reused one repository-authority binder for both fresh command tests and existing real-Git fixtures instead of duplicating store/root setup.
- Kept the ratchet deliberately narrow: it recognizes only `defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))` as executable syntax.
- Left production cleanup, worktree ownership, merge, abandon, recover, and destruction policy byte-for-byte unchanged.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- The original broad Task 1 selector was already green because each affected test selected its own temporary root before exercising behavior; it did not observe the deleted root left for the next unrelated test. The RED regression therefore nested one legacy fixture and inspected authority after its teardown, reproducing the actual cross-test leak before migration.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plans 42-44 have removed all 120 known active self-restoring cleanup sites and the AST ratchet now keeps that count at zero.
- Plan 45 can proceed against clean shelf/worktree fixture authority; the final Plan 40 receipt remains untouched until all later repairs and seven gates pass on one unchanged tree.

## TDD Gate Compliance

- Both tasks have a failing RED commit followed by a passing GREEN commit.

## Self-Check: PASSED

- All five modified test files and this summary exist.
- All four RED/GREEN task commits are present in repository history.
- The final combined selector passed twice with zero active AST matches and no surviving registered test worktree or branch.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-09*
