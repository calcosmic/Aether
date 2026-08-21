---
phase: 128-swarm-watch-host-bridge
plan: 02
subsystem: ts-host
tags: [typescript, node-test, mock-injection, go-bridge, host-router]

requires:
  - phase: 128-swarm-watch-host-bridge
    provides: watch-display.ts and swarm-display.ts modules with runWatchDisplay and runSwarmDisplay
  - phase: 127-swarm-watch-host-bridge
    provides: Oracle lifecycle mock injection pattern (__setCallGoJSON / __restoreCallGoJSON)

provides:
  - watch and swarm command routing in host.ts
  - Usage text documenting watch and swarm commands
  - Unit tests for watch-display module (success, error, rendering)
  - Unit tests for swarm-display module (success, empty manifest, rendering)

affects:
  - Phase 129 (TS Host Command Surface Completion)

tech-stack:
  added: []
  patterns:
    - "Mutable ref wrapper for callGoJSON to enable test injection without changing public API"
    - "Dashboard disabled in tests to avoid TTY-dependent output"

key-files:
  created:
    - .aether/ts-host/test/watch-display.test.ts
    - .aether/ts-host/test/swarm-display.test.ts
  modified:
    - .aether/ts-host/src/host.ts
    - .aether/ts-host/src/watch-display.ts
    - .aether/ts-host/src/swarm-display.ts

key-decisions:
  - "Reused oracle-lifecycle mock injection pattern (__setCallGoJSON / __restoreCallGoJSON) for consistency"
  - "Tests run with dashboard=false to avoid TTY dependencies in CI"

patterns-established:
  - "Display module test pattern: mock callGoJSON via mutable ref, test success + error + rendering helpers"

requirements-completed:
  - SWB-03

metrics:
  duration: 18min
  completed: 2026-05-15
---

# Phase 128 Plan 02: Swarm/Watch Host Bridge — Wiring and Tests Summary

**Watch and swarm commands wired into host.ts router with full unit test coverage using mock-injected Go bridge**

## Performance

- **Duration:** 18 min
- **Started:** 2026-05-15T09:38:00Z
- **Completed:** 2026-05-15T09:56:18Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Added `watch` and `swarm` command cases to `host.ts` switch statement
- `watch` command calls `runWatchDisplay` with `dashboard: !noDashboard`
- `swarm` command reads target from positional args, calls `runSwarmDisplay` with `planOnly: true`
- Updated `printUsage()` to document both new commands
- Added `__setCallGoJSON` / `__restoreCallGoJSON` to `watch-display.ts` and `swarm-display.ts`
- Created `watch-display.test.ts` with 6 tests covering success path, error path, `renderStatusText`, and `renderStatusFrame`
- Created `swarm-display.test.ts` with 6 tests covering success path, empty manifest path, `renderSwarmText`, and `renderSwarmFrame`
- All 189 tests pass (32 suites), zero failures
- `npm run typecheck` passes with no errors
- `go test ./...` passes with no regressions

## Task Commits

Each task was committed atomically:

1. **Task 1: Wire watch and swarm commands into host.ts** — `49f305fa` (feat)
2. **Mock injection helpers** — `d66b7ddb` (feat)
3. **Task 2: Write tests for watch and swarm display** — `751cfa31` (test)

**Plan metadata:** `TBD` (docs: complete plan)

## Files Created/Modified

- `.aether/ts-host/src/host.ts` — Added watch and swarm command routing, updated usage text
- `.aether/ts-host/src/watch-display.ts` — Added `__setCallGoJSON` / `__restoreCallGoJSON` helpers
- `.aether/ts-host/src/swarm-display.ts` — Added `__setCallGoJSON` / `__restoreCallGoJSON` helpers
- `.aether/ts-host/test/watch-display.test.ts` — Unit tests for watch display module
- `.aether/ts-host/test/swarm-display.test.ts` — Unit tests for swarm display module

## Decisions Made

- Reused the oracle-lifecycle mock injection pattern for consistency across display modules
- Tests disable dashboard mode to avoid TTY-dependent behavior in test environment
- Kept `planOnly: true` as default for swarm in host.ts (matches Go `swarm --plan-only` contract)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed incorrect dispatches_count assertion in swarm-display test**
- **Found during:** Task 2 (swarm-display test writing)
- **Issue:** Test asserted `dispatches_count === 2` but mock manifest has 4 dispatches (2 waves x 2 workers)
- **Fix:** Changed assertion to `dispatches_count === 4`
- **Files modified:** `.aether/ts-host/test/swarm-display.test.ts`
- **Verification:** `npm test` passes
- **Committed in:** `751cfa31` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Minor test assertion correction. No scope creep.

## Issues Encountered

- None beyond the single assertion fix above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- host.ts now supports 7 commands: plan, build, continue, oracle, lifecycle, watch, swarm
- Display modules are tested and mock-injectable
- Ready for Phase 129: Final Integration / TS Host Command Surface Completion
- No blockers

---
*Phase: 128-swarm-watch-host-bridge*
*Completed: 2026-05-15*
