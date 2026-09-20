---
phase: 200-iterative-planning
plan: 28
subsystem: planning-persistence
tags: [go, cross-process-locking, transactions, timeline, receipts, tdd]

requires:
  - phase: 200-27
    provides: "Open physical repository authority, no-follow repository storage, and full-path cross-process locks"
provides:
  - "One repository mutation session spanning authoritative reads, derivation, exact baseline checks, commit, rollback, and receipt publication"
  - "Session-bound append-only planning timeline writes that serialize concurrent processes without losing cards"
  - "Session-bound planning stage manifest, output, state, receipt, card, and index persistence"
affects: [phase-200-verification, planning-timeline, stage-receipts, lifecycle-transactions, concurrent-writers]

tech-stack:
  added: []
  patterns:
    - "Repository mutation callback owns one cross-process lock for the complete read-derive-commit boundary."
    - "Lifecycle transaction declarations consume session-captured present-or-absent baselines and refuse uncaptured targets."
    - "Mutation paths have session-aware readers while read-only callers retain their established loaders."

key-files:
  created:
    - cmd/planning_repository_session.go
    - cmd/planning_mutation_session_200_test.go
  modified:
    - cmd/lifecycle_transaction.go
    - cmd/planning_timeline.go
    - cmd/planning_stage_receipt.go

key-decisions:
  - "Use one repository-authorized synthetic data target as the shared mutation-lock identity, held with Store.UpdateFile for the entire callback."
  - "Require every session-backed transaction target to have been read as an exact SHA-256 baseline or explicit absence before declaration."
  - "Keep timeline and stage read-only APIs compatible, but route their mutating entrypoints through session-only helpers that reuse the held roots."

patterns-established:
  - "Session boundary: acquire before the first authoritative planning read and release only after commit, rollback, replay receipt publication, or error return."
  - "Explicit target baseline: no session transaction may silently recapture an undeclared or newer filesystem state."

requirements-completed: [CEC-03, PLAN-02, PLAN-06]

coverage:
  - id: D1
    description: "A repository mutation session serializes separate processes, captures exact present/absent baselines, rejects stale deletion without mutation, survives process termination, and supports multiple lifecycle transactions beneath one held lock."
    requirement: CEC-03
    verification:
      - kind: integration
        ref: "cmd/planning_mutation_session_200_test.go#TestPlanningMutationSession200"
        status: pass
      - kind: other
        ref: "go test -race ./cmd -run '^(TestPlanningMutationSession200|TestPlanningTimelineConcurrentProcesses200)$' -count=1"
        status: pass
    human_judgment: false
  - id: D2
    description: "Two independently derived timeline appends serialize beneath the repository session and persist both compact, content-addressed cards in deterministic predecessor order."
    requirement: PLAN-02
    verification:
      - kind: integration
        ref: "cmd/planning_mutation_session_200_test.go#TestPlanningTimelineConcurrentProcesses200"
        status: pass
      - kind: other
        ref: "go test ./cmd -run '^TestPlanningTimeline' -count=1"
        status: pass
    human_judgment: false
  - id: D3
    description: "Timeline and planning-stage mutation entrypoints load indexes, cards, stage state, manifests, outputs, and receipts through the session before atomically publishing their declared targets."
    requirement: PLAN-06
    verification:
      - kind: other
        ref: "go test ./cmd -run '^(TestPlanningMutationSession200|TestPlanningTimelineConcurrentProcesses200|TestPlanningTimeline.*|TestPlanningStage.*Receipt.*)$' -count=1"
        status: pass
      - kind: other
        ref: "go test -race ./cmd -run '^(TestLifecycleTransaction|TestPlanningTimeline|TestPlanningStage)' -count=1"
        status: pass
    human_judgment: false

duration: 24 min
completed: 2026-09-08
status: complete
---

# Phase 200 Plan 28: Repository Mutation Session Summary

**Cross-process repository sessions now hold one verified lock from planning reads through atomic timeline and stage-receipt publication, preventing stale writers from losing committed history.**

## Performance

- **Duration:** 24 min
- **Started:** 2026-09-08T20:45:59Z
- **Completed:** 2026-09-08T21:09:28Z
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments

- Added a reusable repository mutation session that holds Plan 27's repository-authorized cross-process lock across authoritative reads, derivation, transaction commit, rollback, and callback completion.
- Made lifecycle transactions consume exact session-captured SHA-256-or-absent baselines, reject uncaptured targets, and retain existing recovery and fault-injection behavior.
- Moved timeline append and stage manifest/output/receipt persistence beneath that session, with a deterministic two-process proof that both cards survive in correct hashed order.

## Task Commits

1. **Task 1: Specify session locking, stale baselines, and process races** — `6df6eece` (test)
2. **Task 2: Implement the shared repository mutation session** — `8f4d756a` (feat)
3. **Task 3: Move timeline and stage receipt persistence inside the session** — `3e9e8d57` (feat)

## Files Created/Modified

- `cmd/planning_repository_session.go` — Owns the verified repository authority, root lock, and immutable per-target baselines.
- `cmd/lifecycle_transaction.go` — Accepts a session and validates declarations against its captured roots and baselines.
- `cmd/planning_timeline.go` — Performs append/replay loading and card/index publication inside one mutation session.
- `cmd/planning_stage_receipt.go` — Performs stage dispatch, output, finalization, receipt-chain, and Route-card persistence inside one mutation session.
- `cmd/planning_mutation_session_200_test.go` — Exercises process barriers, exact baselines, stale refusal, rollback prefixes, lock release after termination, nested transaction reuse, containment, and concurrent timeline appends.

## Decisions Made

- The root lock uses a dedicated repository-authorized data target and `Store.UpdateFile`; returning a sentinel error retains the lock without rewriting the target.
- Session-backed transaction declarations fail closed unless the target was already loaded as a full digest or explicit absence beneath the held lock.
- Existing read-only loaders and compatibility helpers remain available for out-of-scope callers, while the timeline/stage mutating entrypoints use separate session-aware helpers.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Made simultaneous first-use lock initialization deterministic**
- **Found during:** Task 2 focused process verification
- **Issue:** Two fresh processes could both observe the lock target and lock directory as absent, causing one repository-store initialization to see a transient not-found error.
- **Fix:** Reopened the store through the same retained repository authority and retried the lock-target write only for `os.ErrNotExist`; no path-based directory fallback was introduced.
- **Files modified:** `cmd/planning_repository_session.go`
- **Verification:** `TestPlanningMutationSession200` passed repeatedly and under the race detector.
- **Committed in:** `8f4d756a`

**2. [Rule 3 - Blocking] Preserved helper compatibility for writers outside Plan 28 scope**
- **Found during:** Task 3 compile verification
- **Issue:** Other planning commands outside the five owned paths still call `planningStageRoots` and the non-session `planningStageWriteConfig` signature.
- **Fix:** Retained those compatibility helpers for untouched callers and introduced distinct session-only config helpers for the three migrated stage mutation paths.
- **Files modified:** `cmd/planning_stage_receipt.go`
- **Verification:** `go test` compiled the complete `cmd` package, all focused suites passed, and no out-of-scope source file changed.
- **Committed in:** `3e9e8d57`

---

**Total deviations:** 2 auto-fixed (1 bug, 1 blocking issue)
**Impact on plan:** Both fixes preserve the required repository authority and strict file scope; no dependency or architectural expansion was introduced.

## Issues Encountered

- The RED test commit failed to compile because the mutation-session type, callback, and transaction field did not exist yet, establishing the intended TDD gate.
- Before Task 3 integration, the two-process timeline test reproduced the stale-writer conflict; after moving the load/derive/commit boundary under the session it passes repeatedly with both cards retained.
- The known unrelated repository-wide `go test ./...` command-package timeout was not rerun. Plan 28's focused, repeated process, lifecycle, timeline, stage, race, and vet gates all completed without assertion failures.

## TDD Gate Compliance

- **RED:** `6df6eece` added deterministic process-pipe tests; the focused command failed on the intentionally missing mutation-session API.
- **GREEN:** `8f4d756a` implemented the session and transaction baseline contract; `TestPlanningMutationSession200` passed three consecutive runs.
- **INTEGRATION GREEN:** `3e9e8d57` moved timeline and stage persistence under the session; the previously failing process timeline race and all existing focused suites passed.
- **REFACTOR:** No separate behavior-neutral refactor commit was needed.

## Verification

- Exact Plan 28 gate — 35 passed on the committed tree.
- Process suite repeated three times — 36 passed; the same suite under `-race` — 12 passed.
- Existing lifecycle transaction suite — 43 passed; timeline suite — 18 passed; full planning-stage suite — 29 passed.
- Combined lifecycle/timeline/stage race suite — 90 passed.
- `go vet ./cmd` and `git diff --check 6df6eece^..3e9e8d57` — passed.
- The implementation range contains exactly the five plan-owned paths; protected `.planning/config.json`, `.gsd` content, Phase 199 patterns, and Phase 200 verification evidence remain untouched.

## Known Stubs

None. Empty-string assignments found by the scan are canonical hash-field clearing or optional-state comparisons, not runtime placeholders.

## User Setup Required

None - no dependency, credential, service, or local configuration change is required.

## Next Phase Readiness

- Plan 28's repository mutation session is ready for the remaining Phase 200 writers and verification plans to adopt.
- Plan 29 is the next incomplete plan; the unrelated repository-wide command-package timeout remains an integration-runner condition rather than a Plan 28 assertion failure.

## Self-Check: PASSED

- The summary and all five plan-owned source/test files exist.
- Task commits `6df6eece`, `8f4d756a`, and `3e9e8d57` are present in repository history.
- The implementation range contains exactly the five declared paths and passes whitespace validation.
- Focused, repeated process, race, lifecycle, timeline, stage, vet, stub, containment, and protected-file checks all pass.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-08*
