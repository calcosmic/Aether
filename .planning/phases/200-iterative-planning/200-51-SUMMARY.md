---
phase: 200-iterative-planning
plan: 51
subsystem: build-result-presentation
tags: [go, partial-recovery, blocker-advisory, json, terminal, accepted-plan, tdd]

# Dependency graph
requires:
  - phase: 200-48
    provides: durable receipt-backed partial recovery attempts and exact redispatch commands
  - phase: 200-49
    provides: approved-specification and accepted-plan fixture authority
provides:
  - one typed partial-recovery projection shared by native and external build presentation
  - one typed blocker-advisory projection shared by plan-only and direct build lanes
  - exact-once terminal and valid JSON coverage for interactive and non-interactive advisory output
affects: [build, build-finalize, partial-replay, wrapper-output, direct-output, phase-200-final-gates]

# Tech tracking
tech-stack:
  added: []
  patterns: [typed result projection, compatibility fields derived from canonical facts, pre-transition read-only advisory snapshot]

key-files:
  created: []
  modified:
    - cmd/codex_build.go
    - cmd/codex_visuals.go
    - cmd/codex_build_finalize.go
    - cmd/codex_workflow_cmds.go
    - cmd/build_finalize_partial_screen_test.go
    - cmd/codex_build_native_partial_test.go
    - cmd/build_blocker_advisory_test.go

key-decisions:
  - "The nested partial_recovery value is the renderer's canonical fact; established top-level recovery fields remain compatibility projections of that value."
  - "The nested build_advisory value is computed once by each runtime lane and drives both terminal rendering and JSON instead of re-reading mutable post-build state."
  - "Direct build checks boundary questions read-only against accepted pre-build authority before the start transaction; it does not materialize a question, rewrite the accepted plan, or change dispatch policy."
  - "A missing durable recovery projection renders an explicit recovery-evidence error and can never fall through to the ordinary build-complete screen."

patterns-established:
  - "Canonical fact before presentation: terminal text and compatibility JSON fields project from one typed runtime result."
  - "Advisory is not authority: blocker visibility may ask the owner a question but cannot approve, mutate, or replace the accepted plan."

requirements-completed: [SYNTH-02, CEC-03, PLAN-05]

# Metrics
duration: 35min
completed: 2026-09-09
---

# Phase 200 Plan 51: Truthful Partial and Blocker Presentation Summary

**Partial builds now name every unfinished task and the exact durable recovery command, while plan-only and direct builds expose one consistent blocker advisory in terminal and JSON output.**

## Performance

- **Duration:** 35 min
- **Started:** 2026-09-09T14:03:18Z
- **Completed:** 2026-09-09T14:38:12Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments

- Added a typed `partial_recovery` result sourced directly from Plan 48's durable recovery child, while preserving existing top-level JSON fields as projections for wrapper compatibility.
- Routed native build and external finalize/replay partial screens through that typed result, so neither lane can reconstruct a command from prose or show the ordinary finished-build screen.
- Added a typed `build_advisory` result shared by plan-only and direct build lanes, including named signals, the interactivity decision, and the one owner question when applicable.
- Moved the direct lane's boundary-question check to the accepted pre-build snapshot and removed its post-build re-derivation, without mutating candidate acceptance, the accepted plan, or dispatch/finalize policy.
- Replaced pre-D-16 shortcut advisory fixtures with approved-and-accepted plan fixtures, then proved exact-once visual output and valid semantic JSON for both interactive and `--no-checkin` runs.

## Task Commits

Each TDD gate and implementation unit was committed atomically:

1. **Task 1 RED: Require typed partial recovery projection** - `0e5c03e0` (test)
2. **Task 1 GREEN: Project durable partial recovery** - `dd1407ea` (fix)
3. **Task 2 RED: Require shared blocker advisory projection** - `108f0d3a` (test)
4. **Task 2 GREEN: Share typed build blocker advisory** - `3745d82b` (fix)

## Files Created/Modified

- `cmd/codex_build.go` - Carries typed partial recovery and blocker advisory facts, preserves legacy projections, rejects a missing durable recovery child, and computes direct advisory truth before lifecycle transition.
- `cmd/codex_visuals.go` - Renders partial recovery and blocker output only from typed build results.
- `cmd/codex_build_finalize.go` - Projects the external finalizer and idempotent replay outcomes through the shared typed recovery helper.
- `cmd/codex_workflow_cmds.go` - Makes native, plan-only, direct, and partial command screens consume typed result projections exactly once.
- `cmd/build_finalize_partial_screen_test.go` - Requires external partial terminal/JSON parity, every unfinished ID, the exact recovery command, and no ordinary success markers.
- `cmd/codex_build_native_partial_test.go` - Requires the same contract on the native direct lane.
- `cmd/build_blocker_advisory_test.go` - Uses accepted-plan authority and verifies both lanes across visual, JSON, boundary-question, and non-interactive cases.

## Decisions Made

- Preserved Plan 48's receipt-backed retry outcome as the authority. The visual layer receives the runtime's exact command and never manufactures `--task` advice.
- Kept legacy `recovery_command`, `unfinished_task_ids`, `blocker_advisory`, and `blocker_advisory_question` keys for compatibility, but made them derivative rather than independent decisions.
- Kept D-16 strict: every end-to-end blocker test now reaches build behavior through specification approval and exact candidate acceptance.
- Kept the advisory informational. It can remain visible or ask whether to continue, but it cannot approve a plan or alter the dispatch manifest.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking Integration] Wired the external finalizer and replay producer to the typed partial projection**
- **Found during:** Task 1
- **Issue:** `cmd/codex_build_finalize.go` was not listed in the task files, but it is the external lane's actual partial result producer and idempotent replay path. Updating only the planned renderer would have left external runtime output on independently assembled map fields.
- **Fix:** Replaced duplicated external/replay field assembly with `addPartialBuildRecoveryResult` and routed the partial finalizer screen through typed-result rendering.
- **Files modified:** `cmd/codex_build_finalize.go`
- **Commit:** `dd1407ea`

**2. [Rule 3 - Blocking Integration] Wired native command handlers to typed partial and advisory rendering**
- **Found during:** Tasks 1 and 2
- **Issue:** `cmd/codex_workflow_cmds.go` owns the actual plan-only/direct terminal branches. Leaving it unchanged would have kept reconstructing loose fields after the runtime created typed facts and would not have delivered the user-visible contract.
- **Fix:** Made partial, plan-only, and direct branches render the typed result once; passed the existing `--no-checkin` state into advisory projection without changing dispatch behavior.
- **Files modified:** `cmd/codex_workflow_cmds.go`
- **Commits:** `dd1407ea`, `3745d82b`

**3. [Rule 1 - Bug] Refused false success when durable partial recovery creation fails**
- **Found during:** Task 1
- **Issue:** The native partial lane could warn after recovery-child persistence failed and still return a successful partial result with a command that had no durable attempt behind it.
- **Fix:** Return an error when recovery persistence fails or yields no child; only emit partial success after the durable recovery outcome exists.
- **Files modified:** `cmd/codex_build.go`
- **Commit:** `dd1407ea`

## Issues Encountered

- The original blocker tests stopped at D-16's accepted-plan gate and captured empty stdout, so they did not reach advisory behavior. The tests now use `createApprovedAcceptedBuildTestColony`; RED failures then narrowed cleanly to the absent typed advisory projection.
- The direct lane formerly re-checked boundary questions after dispatch and lifecycle transition. The check now occurs read-only against the accepted pre-build snapshot, and its result is carried forward rather than recomputed.
- The Task 1 baseline tests passed before RED because they only inspected hand-assembled visual inputs. RED made the runtime result itself carry `partial_recovery` and verified JSON from that same result.
- The installed progress updater found no Markdown-body progress field, and the requirement marker does not parse this milestone's bold-ID checkbox format. ROADMAP advanced to 50/55, all three requirements were already checked, and the updater-dropped STATE provenance fields were restored before commit.

## TDD Gate Compliance

- **Task 1 RED (`0e5c03e0`):** both named tests failed because native and external results had no typed `partial_recovery` projection.
- **Task 1 GREEN (`dd1407ea`):** both named tests passed; the Plan 48 partial journal, replay, forgery, and idempotency regression set also passed 14/14.
- **Task 2 RED (`108f0d3a`):** accepted-authority visual checks reached the intended behavior, while six JSON leaf cases failed because neither lane carried typed `build_advisory` truth.
- **Task 2 GREEN (`3745d82b`):** all 15 task subchecks passed across both lanes, output modes, boundary signaling, and non-interactive behavior.

## Verification

- Task 1 exact selector - PASS (`2/2`).
- Task 2 exact selector - PASS (`15/15` including subtests).
- All five named tests as one regex - PASS (`17/17` including subtests).
- All five named tests individually - PASS (`1`, `1`, `5`, `5`, and `5` checks respectively).
- Blocker/check-in/autopilot focused regression set - PASS (`34/34`).
- Partial/replay, D-16 authority, and direct-preflight focused regression set - PASS (`32/32`).
- `gsd-sdk query verify.key-links .planning/phases/200-iterative-planning/200-51-PLAN.md --raw` - PASS (`2/2 valid`).
- `git diff --check` - PASS.

The full repository suite was intentionally not run because later Phase 200 plans own known remaining failures and the plan explicitly calls for scoped verification.

## Known Stubs

None. No user-facing empty value, placeholder, TODO, FIXME, or mock data path was introduced. Empty/nil checks in the changed code are fail-closed validation or test setup.

## Threat Flags

None. The planned durable-recovery and blocker-advisory trust boundaries are covered by typed receipt-backed facts and accepted pre-build authority. No endpoint, schema, dependency, credential path, or new filesystem authority was introduced.

## User Setup Required

None - no dependency, credential, service, or configuration change is required.

## Next Phase Readiness

- Phase 200's later gap-closure plans can rely on truthful partial and advisory presentation in both terminal and JSON modes.
- Plan 48 recovery evidence and Plan 49 accepted-plan authority remain intact under the focused regression suites.
- No Plan 51 blocker remains; full-suite reconciliation stays with the later plan that owns those known failures.

## Self-Check: PASSED

- All seven modified implementation/test files and this summary exist.
- TDD commits `0e5c03e0`, `dd1407ea`, `108f0d3a`, and `3745d82b` exist in repository history.
- Both exact task gates, all five individual tests, the combined selector, focused regressions, and both declared key links pass.
- STATE points to Plan 52, ROADMAP marks Plan 51 complete at 50/55, and all three plan requirements remain checked complete.
- `.planning/config.json`, `.gsd/`, and untracked Phase 199 `199-PATTERNS.md` remain unstaged and untouched by Plan 51.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-09*
