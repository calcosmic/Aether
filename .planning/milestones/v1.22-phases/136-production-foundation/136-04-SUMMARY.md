---
phase: 136-production-foundation
plan: 04
subsystem: ts-host
tags: [typescript, oracle, dispatch, ceremony, dry-run, testing]

# Dependency graph
requires: [136-01, 136-03]
provides:
  - "Oracle lifecycle dispatches real workers with ceremony rendering (HOST-05)"
  - "Go-computed confidence from finalize drives Oracle loop termination"
  - "--dry-run flag on all host commands renders ceremony without spawning workers (HOST-07)"
  - "DRY RUN badge visible in stderr output (D-06)"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns: [oracle-ceremony-rendering, dry-run-flag, ceremony-badge]

key-files:
  created: []
  modified:
    - .aether/ts-host/src/oracle-lifecycle.ts
    - .aether/ts-host/src/host.ts
    - .aether/ts-host/src/command-registry.ts
    - .aether/ts-host/src/ceremony-adapter.ts
    - .aether/ts-host/test/oracle-lifecycle.test.ts
    - .aether/ts-host/test/host-integration.test.ts

key-decisions:
  - "Oracle ceremony uses 'build' CeremonyWorkflow since Oracle does not have its own workflow enum value"
  - "Ceremony helpers (emitCeremonyOutput) duplicated locally in oracle-lifecycle.ts to match host.ts pattern"
  - "DRY RUN badge rendered via renderDryRunBadge() in ceremony-adapter.ts, shared by all runner types"
  - "Dry-run path for dispatched runner fetches manifest (read-only) and renders ceremony before badge, no dispatch/finalizer"
  - "Dry-run path for oracle-lifecycle fetches first iteration manifest only"
  - "Dry-run path for go-json adds badge before manifest JSON output"

patterns-established:
  - "Dry-run ceremony preview: fetch manifest -> render ceremony -> badge -> exit without dispatch/finalizer"
  - "Ceremony adapter mock injection pattern in oracle-lifecycle.ts mirrors host.ts mock injection"
  - "Oracle task_brief forwarded through BuildDispatch.task_brief field for real dispatch"

requirements-completed: [HOST-05, HOST-07, D-06]

# Metrics
duration: 8min
completed: 2026-05-18
---

# Phase 136 Plan 04: Oracle Real Dispatch and --dry-run Ceremony Preview Summary

**Wired Oracle lifecycle for real dispatch with ceremony rendering and added --dry-run ceremony preview with DRY RUN badge -- 11 new tests across oracle-lifecycle.test.ts and host-integration.test.ts.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-05-18T12:00:00Z
- **Completed:** 2026-05-18T12:08:00Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- Wired Oracle lifecycle for real dispatch: Oracle already defaults to real dispatch (simulateWorkers defaults to false from host.ts)
- Added ceremony rendering to Oracle RALF loop: spawn-plan, wave-start, worker-complete, closeout for each iteration (HOST-05)
- Added task_brief field to Oracle BuildDispatch for real worker dispatch
- Added Go-computed confidence from finalize result driving loop termination (verified in tests)
- Added --dry-run flag to parseArgs and ParsedHostArgs (HOST-07)
- Implemented dry-run path for all runner types: dispatched, oracle-lifecycle, go-json
- Added renderDryRunBadge() to ceremony-adapter.ts with visible "--- DRY RUN ---" output (D-06)
- Dry-run fetches manifest (read-only), renders ceremony, shows badge, exits without dispatching workers or calling finalizers
- All 70 tests pass across oracle-lifecycle (17), host (28), host-integration (20), command-registry (5)

## Task Commits

Each task was committed atomically:

1. **Task 1: Wire Oracle lifecycle for real dispatch with ceremony (HOST-05)** - `329e5e0b`
2. **Task 2: Add --dry-run flag with ceremony preview (HOST-07, D-06)** - `fcc480f8`

## Files Created/Modified
- `.aether/ts-host/src/oracle-lifecycle.ts` - Imported createCeremonyAdapter and CeremonyWorkflow; added _createCeremonyAdapterRef mock injection; added emitCeremonyOutput helper; added ceremony rendering calls (spawn-plan, wave-start, worker-complete, closeout) in runOracleLifecycle loop; added task_brief to Oracle BuildDispatch
- `.aether/ts-host/src/host.ts` - Added dryRun boolean to parseArgs; added --dry-run flag parsing; added dryRun to ParsedHostArgs return; added --dry-run to usage text; added runDryRunDispatchedCommand helper; modified dispatched runner to check dryRun first; modified oracle-lifecycle runner to handle dryRun; modified go-json runner to show badge on dryRun; imported renderDryRunBadge from ceremony-adapter
- `.aether/ts-host/src/command-registry.ts` - Added dryRun: boolean to ParsedHostArgs interface
- `.aether/ts-host/src/ceremony-adapter.ts` - Added renderDryRunBadge() function exporting DRY RUN badge to stderr
- `.aether/ts-host/test/oracle-lifecycle.test.ts` - Added 5 new tests: real dispatch (simulateWorkers=false), simulated dispatch (simulateWorkers=true), finalize should_continue=false termination, max_iterations ceiling, ceremony rendering verification; added __setCreateCeremonyAdapter/__restoreCreateCeremonyAdapter imports and mock
- `.aether/ts-host/test/host-integration.test.ts` - Added 6 new tests in "dry-run ceremony preview" describe block: build dry-run, plan dry-run, continue dry-run, oracle dry-run, DRY RUN badge function verification, dispatchWorkers not called verification

## Decisions Made
- Oracle ceremony uses "build" CeremonyWorkflow because ceremony-adapter.ts does not define an "oracle" workflow value -- this is correct per D-04/D-05 (ceremony output is identical across workflows)
- The emitCeremonyOutput helper is duplicated locally in oracle-lifecycle.ts matching the same pattern in host.ts rather than creating a shared module, keeping the change minimal
- The dry-run path for the dispatched runner extracts manifest data generically (checks dispatch_manifest, plan_manifest, planning_manifest, continue_manifest) to work for all dispatched workflows
- The oracle-lifecycle dry-run fetches only the first iteration manifest to keep the preview lightweight

## Deviations from Plan

None -- plan executed exactly as written.

## Issues Encountered
- Initial Oracle dispatch tests used max_iterations=1 with current_iteration=1, causing checkStopConditions to halt before dispatch. Fixed by using max_iterations=3.
- DRY RUN badge test initially used `require()` which fails in ESM context. Fixed by using dynamic `import()`.

## Next Phase Readiness
- Phase 136 is now complete (all 4 plans executed)
- Phase 137 (Hive Wisdom Injection) can proceed -- the production dispatch pipeline is fully wired
- All 70 tests across all test files pass
- Phase 136 requirements completed: HOST-01, HOST-02, HOST-03, HOST-04, HOST-05, HOST-06, HOST-07, HOST-08, HOST-09, SKILL-01, SKILL-02, SKILL-03

## Self-Check: PASSED

- FOUND: .aether/ts-host/src/oracle-lifecycle.ts (createCeremonyAdapter import, ceremony rendering in runOracleLifecycle, task_brief on dispatch)
- FOUND: .aether/ts-host/src/host.ts (dryRun in parseArgs, runDryRunDispatchedCommand, dryRun checks in all runner cases)
- FOUND: .aether/ts-host/src/command-registry.ts (dryRun: boolean in ParsedHostArgs)
- FOUND: .aether/ts-host/src/ceremony-adapter.ts (renderDryRunBadge function)
- FOUND: .aether/ts-host/test/oracle-lifecycle.test.ts (5 new tests in HOST-05 section)
- FOUND: .aether/ts-host/test/host-integration.test.ts (6 new tests in dry-run ceremony preview block)
- FOUND: commit 329e5e0b (Task 1)
- FOUND: commit fcc480f8 (Task 2)
- VERIFIED: 70 tests pass across oracle-lifecycle (17), host (28), host-integration (20), command-registry (5)

---
*Phase: 136-production-foundation*
*Completed: 2026-05-18*
