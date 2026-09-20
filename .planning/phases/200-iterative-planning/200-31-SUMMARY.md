---
phase: 200-iterative-planning
plan: 31
subsystem: planning-authority
tags: [go, tdd, content-addressing, transactions, concurrency, impact-closure]

requires:
  - phase: 200-28
    provides: Repository-scoped planning mutation sessions and lifecycle transactions
  - phase: 200-29
    provides: Canonical specification and planning record identities
  - phase: 200-30
    provides: Complete immutable candidate review payloads and acceptance tokens
provides:
  - Session-bound candidate acceptance with authority reads taken after locking
  - Independent base/specification/proposal semantic and impact derivation
  - Build/run revalidation of accepted candidate impact and node scope
  - Deterministic rollback, replay, and separate-process acceptance proofs
affects: [planning, build-authority, run-authority, lifecycle-facts, phase-insertion]

tech-stack:
  added: []
  patterns: [derive-then-compare authority, session-baseline reads, atomic authority activation, canonical production fixtures]

key-files:
  created:
    - cmd/plan_candidate_acceptance_200_test.go
  modified:
    - cmd/plan_revision.go
    - cmd/plan_impact.go
    - cmd/planning_state.go
    - cmd/planning_state_test.go
    - cmd/plan_authority.go

key-decisions:
  - "Acquire the Plan 28 repository mutation session before reading any candidate, state, specification, base, timeline, or receipt authority."
  - "Treat candidate semantic delta, authority impacts, and affected markers only as claims to compare with a pure base/specification/proposal derivation."
  - "Use the verified final iteration only for evidence and generation-only classifications that cannot be reconstructed from PlanRevision, while requiring every stable semantic identity to exist in the independently derived tuple."
  - "Preserve completed work using the independently verified activation scope, not the broader proof-reachability graph."

patterns-established:
  - "Authority derivation: immutable stored hashes prove bytes, while a separate pure derivation proves meaning."
  - "Execution recheck: build and run share one evaluator that repeats canonical candidate, timeline, receipt, and impact validation."

requirements-completed: [PLAN-04, PLAN-05, PLAN-06]

duration: 57min
completed: 2026-09-09
---

# Phase 200 Plan 31: Independent Acceptance Authority Summary

**Candidate acceptance and every later execution attempt now rederive semantic impact from locked base/specification/proposal facts, with atomic activation and exact replay under concurrent processes.**

## Performance

- **Duration:** 57 min
- **Started:** 2026-09-08T22:40:53Z
- **Completed:** 2026-09-08T23:38:10Z
- **Tasks:** 3
- **Files modified:** 6

## Accomplishments

- Moved every acceptance authority read beneath one repository mutation session and committed candidate, receipt, planning stage, and colony state through one rollback-safe transaction.
- Added a pure derivation of the full base-to-proposal semantic delta and transitive plan/specification impact closure, then compared candidate claims against the derived result.
- Removed structural fallback validation for modern accepted authority; build and run now repeat canonical candidate, proposal, timeline, receipt, node-scope, and impact checks.
- Proved exact replay is read-only, transaction faults restore the complete authority inventory, and simultaneous OS processes yield one activation plus one byte-identical replay.
- Rebuilt the shared modern planning-state fixture by generating and accepting production-addressed records rather than treating digest-shaped placeholders as authority.

## Task Commits

Each TDD gate and implementation was committed atomically:

1. **Task 1 RED: Define forged-coverage, replay, rollback, and acceptance races** - `1d532259` (test)
2. **Task 2 GREEN: Re-derive exact delta and impact at acceptance** - `ea4fdbee` (feat)
3. **Task 3 RED: Expose accepted build/run impact trust** - `abec08bf` (test)
4. **Task 3 GREEN: Revalidate accepted impact before build and run** - `5d16c800` (feat)

## Files Created/Modified

- `cmd/plan_candidate_acceptance_200_test.go` - Forgery, rollback, exact replay, process serialization, execution recheck, and accepted phase-insertion proofs.
- `cmd/plan_revision.go` - Session-first acceptance, exact baseline loading, independent derivation, and atomic activation/rollback.
- `cmd/plan_impact.go` - Pure semantic/impact derivation and independently justified living-plan impact reconciliation.
- `cmd/planning_state.go` - Strict session-baseline state loading and canonical modern candidate validation.
- `cmd/planning_state_test.go` - Production-generated canonical fixture with isolated immutable clones and retained corruption coverage.
- `cmd/plan_authority.go` - Shared build/run validator that independently repeats accepted-lineage and node-scope checks.

## Decisions Made

- Exact content addressing remains necessary but is no longer sufficient: stored impact claims never grant authority without a separately derived match.
- The repository lock is acquired before the first authority read, preventing a mixed candidate/specification/base/timeline moment.
- Modern accepted plans always receive strict canonical validation; only explicitly classified `legacy_unbound` plans retain legacy compatibility.
- Phase/task generation metadata that is not retained in `PlanRevision` stays bound by the canonical final-card bytes, while stable IDs and reproducible proof/dependency hashes must match the independently derived proposal.
- Completed phases and tasks are preserved according to the verified candidate activation scope; shared proof reachability alone cannot erase earned completion.

## Verification

- `go test ./cmd -run '^(TestPlanningState.*|TestCandidateAcceptanceBase.*|TestPlanCandidateAcceptanceIntegrity200|TestPlanCandidateAcceptanceConcurrentProcesses200|TestPlanCandidateAccept.*)$' -count=1` - passed.
- `go test ./cmd -run '^(TestPlanningState.*|TestCandidateAcceptanceBase.*|TestPlanCandidateAcceptanceIntegrity200|Test.*PlanAuthority.*|Test.*Build.*Authority.*|Test.*Run.*Authority.*|TestSpecProjection.*|TestLifecycleProjection.*)' -count=1` - passed.
- Plan-impact, lifecycle, next-action, seal, Plan 30, and phase-insertion compatibility sweep - passed.
- Acceptance integrity and separate-process concurrency suite with `-count=5` - passed.
- The same race/rollback/authority suite under `go test -race` with `-count=2` - passed.
- `go vet ./...` - passed.
- `go test ./... -count=1` - executed exactly once; non-`cmd` packages passed, while `cmd` reached its pre-existing 11-minute default timeout and also reported unrelated build-presentation assertions. The exact assertions reproduce unchanged at pre-plan commit `1c42ba9e`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Cleared an exhausted generated Go cache**

- **Found during:** Task 1 RED
- **Issue:** The 66 GB generated Go cache exhausted the filesystem and prevented compilation.
- **Fix:** Ran `go clean -cache -testcache`, recovering approximately 67 GiB without changing repository files.
- **Files modified:** None
- **Verification:** Focused and race-enabled Go suites compiled and passed afterward.
- **Committed in:** Not applicable (generated cache only)

**2. [Rule 1 - Bug] Preserved phase-insertion compatibility in derived scope**

- **Found during:** Task 3 compatibility verification
- **Issue:** Phase insertion legitimately records root-level proof aggregates that normal Route output omits, and the first derived universe excluded them.
- **Fix:** Included aggregate nodes only when the immutable final card classifies them.
- **Files modified:** `cmd/plan_impact.go`
- **Verification:** Existing phase-insertion acceptance and independently revalidated phase-insertion authority tests passed.
- **Committed in:** `5d16c800`

**3. [Rule 1 - Bug] Kept unaffected completed work credited**

- **Found during:** Task 3 compatibility verification
- **Issue:** The broad proof-reachability graph could reopen a completed phase merely because a new corrective phase shared one proof link.
- **Fix:** Applied completion preservation from the independently verified activation scope while retaining the broader graph for reachability proof.
- **Files modified:** `cmd/plan_revision.go`
- **Verification:** `TestInsertPhaseImmutableStableIDsAndCompletedStatus` passed.
- **Committed in:** `5d16c800`

**4. [Rule 1 - Bug] Restored canonical fixture lifecycle context**

- **Found during:** Task 3 compatibility verification
- **Issue:** Replacing the fake-hash fixture with production output omitted the helper's goal/session and progress-mutation contract.
- **Fix:** Retained production canonical identities while restoring goal/session context and isolated active-phase mutation behavior expected by lifecycle tests.
- **Files modified:** `cmd/planning_state_test.go`
- **Verification:** Next-action, seal, lifecycle-facts, and lifecycle-projection suites passed.
- **Committed in:** `5d16c800`

---

**Total deviations:** 4 auto-fixed (3 Rule 1 bugs, 1 Rule 3 blocker)
**Impact on plan:** Every deviation was required to compile or preserve existing correctness; no new command, dependency, or out-of-scope source file was introduced.

## Issues Encountered

- The exact broad repository command reached the known `cmd` package timeout after 11 minutes. Several unrelated build-presentation assertions also appeared, so they were rerun as one bounded group and reproduced against the untouched pre-Plan-31 commit `1c42ba9e`; they are not caused by this plan.

## Deferred Issues

- The pre-existing `cmd` default-timeout and unrelated build-presentation fixture failures remain outside Plan 31's six-file scope. Plan 31's focused correctness, repeated process, race-detector, compatibility, and vet gates are green.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Acceptance and execution authority now share independently derived impact truth and are ready for later temporal-standing work.
- No Plan 31 correctness blocker remains; only the baseline-wide `cmd` test debt described above is deferred.

## Self-Check: PASSED

- Summary file exists.
- Task commits `1d532259`, `ea4fdbee`, `abec08bf`, and `5d16c800` exist in repository history.
- All six declared source/test paths are the only code paths changed from the Plan 30 baseline.
- Protected user/orchestrator files retain their starting hashes.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-09*
