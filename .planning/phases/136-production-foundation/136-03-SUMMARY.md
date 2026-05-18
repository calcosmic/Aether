---
phase: 136-production-foundation
plan: 03
subsystem: ts-host
tags: [typescript, dispatch, ceremony, build, plan, continue, finalizer]

# Dependency graph
requires: [136-01, 136-02]
provides:
  - "Dispatched runner type routes build/plan/continue through real dispatch pipeline"
  - "Build dispatch fetches manifest, dispatches workers, writes completion file, calls build-finalize"
  - "Plan dispatch fetches manifest, dispatches Scout/Route-Setter workers, calls plan-finalize"
  - "Continue dispatch fetches manifest, dispatches review workers, calls continue-finalize"
  - "Ceremony renders spawn-plan, wave-start, worker-complete, closeout for all three workflows"
  - "Skill injection summary line when dispatches have skill_section (D-07)"
affects: [136-04]

# Tech tracking
tech-stack:
  added: []
  patterns: [dispatched-runner-pipeline, ceremony-helpers-in-host, mock-injection-for-dispatch]

key-files:
  created: []
  modified:
    - .aether/ts-host/src/command-registry.ts
    - .aether/ts-host/src/host.ts
    - .aether/ts-host/test/host.test.ts
    - .aether/ts-host/test/host-integration.test.ts

key-decisions:
  - "New 'dispatched' HostCommandRunner type handles build/plan/continue with ceremony and finalizer integration"
  - "Plan and continue dispatch pipelines follow same pattern as build: manifest -> ceremony -> dispatch -> completion file -> finalizer"
  - "Fallback in dispatched runner for unrecognized workflows (colonize, seal) uses go-json passthrough"
  - "Mock injection points (__setDispatchWorkers, __setDetectAvailablePlatforms, __restoreAllMocks) added to host.ts for testability"
  - "Skill injection summary (D-07) counts dispatches with non-empty skill_section and writes one-line message to stderr"

patterns-established:
  - "Dispatched runner pipeline: buildHostGoArgs -> callGoJSON manifest -> ceremony render -> detectAvailablePlatforms -> dispatchWorkers -> toWorkerResults -> ceremony render -> writeCompletionFile -> callGoJSON finalizer -> closeout"
  - "Ceremony helpers extracted to host.ts (emitCeremonyOutput, renderManifestCeremony, renderWorkerCeremony, ceremonyExecutionWaves)"
  - "Mock injection pattern: mutable references for external dependencies with set/restore/restoreAll test helpers"

requirements-completed: [HOST-02, HOST-03, HOST-04, HOST-06, D-07]

# Metrics
duration: 10min
completed: 2026-05-18
---

# Phase 136 Plan 03: Real Dispatch Pipelines for Build, Plan, and Continue Summary

**Added dispatched runner with full build/plan/continue pipelines and ceremony -- 16 new tests across host.test.ts and host-integration.test.ts.**

## Performance

- **Duration:** 10 min
- **Started:** 2026-05-18T11:30:00Z
- **Completed:** 2026-05-18T12:00:00Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Added "dispatched" to HostCommandRunner union type and changed build/plan/continue to use it
- Implemented build dispatch pipeline: Go manifest fetch, platform check, ceremony render, worker dispatch, completion file write, build-finalize call, closeout (HOST-02, HOST-06)
- Implemented plan dispatch pipeline: Go plan manifest fetch, Scout/Route-Setter dispatch, plan-finalize call with ceremony (HOST-03)
- Implemented continue dispatch pipeline: Go continue manifest fetch, review worker dispatch, continue-finalize call with ceremony (HOST-04)
- Added skill injection summary line counting dispatches with skill_section (D-07)
- Added mock injection points for dispatchWorkers and detectAvailablePlatforms for testability
- All 47 tests pass across host.test.ts (28), host-integration.test.ts (14), and command-registry.test.ts (5)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add "dispatched" runner and build real dispatch pipeline (HOST-02, HOST-06)** - `d5ed7d2a`
2. **Task 2: Add plan and continue real dispatch pipelines (HOST-03, HOST-04)** - `27514d1b`

## Files Created/Modified
- `.aether/ts-host/src/command-registry.ts` - Added "dispatched" to HostCommandRunner union; changed build, plan, and continue from runner "go-json" to "dispatched"; updated descriptions
- `.aether/ts-host/src/host.ts` - Added ceremony helpers (emitCeremonyOutput, renderManifestCeremony, renderWorkerCeremony, ceremonyExecutionWaves), manifest result types (BuildManifestResult, PlanManifestResult, ContinueManifestResult), runDispatchedBuildCommand, runDispatchedPlanCommand, runDispatchedContinueCommand, mock injection points (__setDispatchWorkers, __setDetectAvailablePlatforms, __restoreAllMocks), dispatched case in main switch
- `.aether/ts-host/test/host.test.ts` - Added 7 new tests in "dispatched build runner" describe block: dispatched runner type verification, build --plan-only args, --simulate passthrough, plan/continue runner types, mock injection/restore
- `.aether/ts-host/test/host-integration.test.ts` - Added 9 new tests in "dispatched plan and continue runners" describe block: plan/continue dispatched runner types, plan/continue --plan-only args, plan-finalize/continue-finalize configuration, --simulate passthrough, colonize/seal still go-json

## Decisions Made
- The dispatched runner pattern follows the lifecycle.ts build pipeline: manifest -> ceremony -> dispatch -> completion file -> finalizer, replicated for each workflow
- Plan manifest uses `plan_manifest` or `planning_manifest` field (checks both for compatibility)
- Continue manifest uses `continue_manifest` field
- Skill injection summary counts dispatches with non-empty skill_section strings and writes to stderr before dispatch
- Mock injection uses mutable references (same pattern as __setCallGoJSON) for dispatchWorkers and detectAvailablePlatforms
- The dispatched runner switch case handles unrecognized workflows (colonize, seal, swarm) by falling back to go-json passthrough

## Deviations from Plan

None -- plan executed exactly as written.

## Issues Encountered
- None -- implementation matched the lifecycle.ts reference patterns closely

## Next Phase Readiness
- Plan 04 (Oracle real dispatch and --dry-run ceremony preview) can proceed -- the dispatched runner framework is in place
- The dispatched runner's fallback for unknown workflows handles oracle (via oracle-lifecycle runner) and dry-run (to be added in Plan 04)
- All existing tests (47 combined) continue to pass

## Self-Check: PASSED

- FOUND: .aether/ts-host/src/command-registry.ts (dispatched in HostCommandRunner, build/plan/continue use it)
- FOUND: .aether/ts-host/src/host.ts (runBuildDispatch, runDispatchedPlanCommand, runDispatchedContinueCommand, dispatched case)
- FOUND: .aether/ts-host/test/host.test.ts (7 new tests in dispatched build runner block)
- FOUND: .aether/ts-host/test/host-integration.test.ts (9 new tests in dispatched plan/continue runners block)
- FOUND: commit d5ed7d2a (Task 1)
- FOUND: commit 27514d1b (Task 2)
- VERIFIED: 47 tests pass across host.test.ts, host-integration.test.ts, command-registry.test.ts

---
*Phase: 136-production-foundation*
*Completed: 2026-05-18*
