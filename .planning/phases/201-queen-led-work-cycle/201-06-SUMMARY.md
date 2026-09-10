---
phase: 201-queen-led-work-cycle
plan: "06"
subsystem: queen-orchestration
tags: [go, spend-ledger, build-attempt, closeout, status, work-outcome, cost-line]

# Dependency graph
requires:
  - phase: 201-queen-led-work-cycle (plan 04)
    provides: "pkg/colony/work_outcome.go: colony.WorkOutcome (six-verdict vocabulary), plus cmd/lifecycle_closeout.go's LifecycleCloseout.WorkOutcome field and full-ceremony slot gating this plan's Task 2 hangs the cost-and-time block off of"
provides:
  - "cmd/spend_cost_line.go: spendElapsedFigure(startedAt, completedAt) and renderSpendCostLine(phase) now join an attempt-bound elapsed-time line to the existing per-phase cost block, sourced only from the attempt's own StartedAt/CompletedAt; renderSpendCostLineFromLedgers(ledgers) keeps its old signature and behavior unchanged via the new shared renderSpendCostLineBlock(ledgers, elapsedFigure) body"
  - "cmd/lifecycle_closeout.go: appendLifecycleCloseoutSpendLine appends the one cost-and-time block (via the existing appendSpendCostLine helper) as the final element of appendLifecycleCloseoutVisual's output, for every closeout that carries a colony.WorkOutcome verdict (build/check/autopilot work-cycle results) -- one placement rule, never a second one, and never rendered for a closeout with no verdict (colonize/plan/init/entomb/pause/resume/seal)"
  - "cmd/status.go: computeColonyRunningSpendTotal(state) sums elapsed time and reported cost across every planned phase through the same durable loaders the per-phase block already uses (loadSpendLedgersForPhase, loadLatestBuildAttempt), and renderColonyRunningSpendTotal renders both totals plus the unreported/unmeasured counts on the status dashboard and in its JSON envelope"
affects: [201-12, 201-07, 201-10, 201-15]

# Actuals (#2632)
actuals:
  tokens: 7406
  tasks: 3
  commits: 6

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Attempt-bound sentinel reuse across two figures: spendElapsedFigure mirrors spendCostLineFigure's own discipline exactly -- a timestamp/usage that could not be measured renders the identical dash sentinel (spendNotReportedFigure), never a zero and never a computed default, so a reader cannot mistake 'nothing was measured' for 'measurement was zero'"
    - "Absent-vs-incomplete distinction, carried through three layers: a phase/attempt genuinely absent from disk renders NOTHING about that fact (no line, no sentinel) at every layer this plan touches -- renderSpendCostLine (no attempt -> no Elapsed line), the colony total (no attempt -> contributes to neither the sum nor the unmeasured count), and the closeout gate (no WorkOutcome -> no cost block at all). Only a fact that WAS attempted but came back incomplete (an attempt exists but is missing a timestamp; a row exists but carries no usage) renders the sentinel or increments the unreported/unmeasured count. Collapsing this distinction anywhere would either fabricate silence into a false zero or bury a real gap inside a total that looks complete."
    - "One placement rule instead of two: appendLifecycleCloseoutSpendLine reuses the exact same appendSpendCostLine helper cmd/ceremony_cmd.go's build/continue direct lane already called since Phase 196, rather than a second rendering path invented for the lifecycle_closeout.go system -- gated purely on closeout.WorkOutcome != nil so the two call sites can never both fire for the same screen"
    - "Existing-caller preservation via a shared elapsedFigure=\"\" default: renderSpendCostLineFromLedgers(ledgers) is now a one-line wrapper around the new renderSpendCostLineBlock(ledgers, elapsedFigure), always passed \"\", so every one of the 9 pre-existing tests that call it directly (and the many that call renderSpendCostLine(phase) against a store with no attempt saved) is provably unmodified -- confirmed by running them unchanged after this plan's implementation, not merely by inspection"

key-files:
  created:
    - cmd/status_running_total_test.go
  modified:
    - cmd/spend_cost_line.go
    - cmd/spend_cost_line_test.go
    - cmd/lifecycle_closeout.go
    - cmd/status.go

key-decisions:
  - "Elapsed time is scoped to renderSpendCostLine(phase) only, never renderSpendCostLineFromLedgers(ledgers). The two existing exported-shape functions diverge on purpose: the ledgers-only function has no attempt to read from and is what every pre-Phase-201 test calls directly, so keeping its signature and its always-\"\" elapsedFigure untouched was the only way to satisfy the plan's own 'existing spend and cost-line tests pass unmodified' constraint while still adding elapsed time to the one production entry point (renderSpendCostLine) that ceremony_cmd.go's real build/continue direct lane already calls."
  - "A phase with genuinely no build attempt on disk renders no 'Elapsed: ' line at all -- not the dash sentinel. Verified this was the only design that could keep every pre-existing renderSpendCostLine(phase) test passing unmodified (several assert the dash sentinel NEVER appears anywhere in a fully-measured or no-rows block, and none of those tests seed an attempt), and it is also the more honest claim: 'no attempt was ever recorded' and 'an attempt ran but did not finish measuring' are different facts, and only the second one is what the sentinel means."
  - "The cost-and-time block in appendLifecycleCloseoutVisual is gated strictly on closeout.WorkOutcome != nil, not on command name or workflow. This makes 'every one of the six work verdicts, never for a closeout with no verdict' a single boolean check rather than a workflow allowlist that would need updating every time a new lifecycle command is added -- and it is structurally impossible for it to double up with ceremony_cmd.go's own, separate cost-line call sites (closeoutDirectVisual's build/continue rendering never touches applyLifecycleCloseout/appendLifecycleCloseoutVisual at all -- confirmed by reading cmd/closeout_direct_render.go, this plan's own read_first)."
  - "The colony-wide running total sums COST across ALL rows/attempts recorded for a phase (via loadSpendLedgersForPhase + computeSpendTotals, which already accumulate every attempt's rows per the phase's own ledger-write invariant), but sums ELAPSED from only each phase's LATEST build attempt (loadLatestBuildAttempt) -- mirroring the exact attempt-bound scope Task 1 established for one phase's own block, and the only per-attempt elapsed source available without adding new enumeration machinery outside this task's declared file scope (cmd/status.go, cmd/status_running_total_test.go)."
  - "The running-total dashboard section is omitted entirely when the colony has recorded nothing at all (colonyRunningSpendTotal.hasAnyFacts() is false) -- a freshly initialized colony is not shown a 'Cost: not known' / 'Elapsed: not known' pair about work it has never attempted, matching the same no-fabricated-row discipline the per-phase cost block already holds for zero rows."

requirements-completed: []
# CEC-06 and WORK-05 (this plan's declared requirements) are each also
# declared by sibling 201-* plans (CEC-06: 201-02 [complete]/04 [complete]/
# 06/07/12/15; WORK-05: 201-06/07/10/15) and therefore stay open per the
# shared-ID gate (#2388) until every declaring plan has a SUMMARY.md --
# confirmed via `gsd-tools query requirements.ready-ids`, correct and
# expected, not a gap in this plan's own work.

coverage:
  - id: D1
    description: "Every attempt-bound cost block (renderSpendCostLine) joins elapsed time to the reported cost, sourced only from the attempt's own StartedAt/CompletedAt; missing or unparseable timestamps render the identical dash sentinel the unreported-cost figure uses, and a phase with no attempt at all renders no elapsed line"
    requirement: CEC-06
    verification:
      - kind: unit
        ref: "cmd/spend_cost_line_test.go#TestElapsedTimeComesFromTheAttemptTimestamps"
        status: pass
      - kind: unit
        ref: "cmd/spend_cost_line_test.go#TestMissingTimestampRendersTheUnmeasuredSentinel"
        status: pass
      - kind: unit
        ref: "cmd/spend_cost_line_test.go#TestNoAttemptMeansNoElapsedLineAtAll"
        status: pass
      - kind: integration
        ref: "cmd/spend_pipeline_e2e_test.go#TestSpendPipelineIsReachableEndToEnd"
        status: pass
    human_judgment: false
  - id: D2
    description: "The cost-and-time block renders exactly once, last on the screen, for every one of the six work verdicts (success, no_change, partial, blocker, timeout, interrupted) -- including a blocked run that recorded figures before blocking -- and never for a closeout with no work verdict"
    requirement: CEC-06
    verification:
      - kind: unit
        ref: "cmd/spend_cost_line_test.go#TestEveryVerdictEndsWithOneCostAndTimeBlock"
        status: pass
      - kind: unit
        ref: "cmd/spend_cost_line_test.go#TestBlockedRunStillReportsWhatItSpent"
        status: pass
      - kind: unit
        ref: "cmd/spend_cost_line_test.go#TestNoWorkVerdictMeansNoCostAndTimeBlock"
        status: pass
      - kind: integration
        ref: "cmd/ceremony_closeout_spend_test.go#TestBuildEndsWithOneCostLine"
        status: pass
      - kind: integration
        ref: "cmd/ceremony_closeout_spend_test.go#TestNoLaneRendersTwoCostLines"
        status: pass
    human_judgment: true
    rationale: "The gate (closeout.WorkOutcome != nil) is fully proven against fixture closeouts, but no production call site sets LifecycleCloseoutDetails.WorkOutcome yet anywhere in this tree (confirmed by 201-05-SUMMARY.md's own 'Next Phase Readiness' -- buildUnverifiedCloseoutDetails/buildVerifiedCloseoutDetails are built and tested but not wired to a live aether build call site). ceremony_cmd.go's separate build/continue direct lane already carried the cost line since Phase 196 and now also carries elapsed time via D1, so today's real build/continue screens are unaffected either way -- but this plan's own Task 2 addition (the lifecycle_closeout.go system's block) will not visibly render for any real user until a later plan wires a WorkOutcome into a live call site. A human should confirm that gap is intentional scope for this plan rather than a missed wiring step."
  - id: D3
    description: "The status dashboard shows a colony-wide running total for elapsed time and for reported cost, each the exact sum of the per-phase facts it reads, with the count of unreported/unmeasured rows or attempts named on screen and in the JSON envelope; reading status writes nothing"
    requirement: WORK-05
    verification:
      - kind: unit
        ref: "cmd/status_running_total_test.go#TestStatusRunningTotalEqualsTheSumOfItsRows"
        status: pass
      - kind: unit
        ref: "cmd/status_running_total_test.go#TestStatusRunningTotalWritesNothing"
        status: pass
    human_judgment: false

duration: 45min
completed: 2026-09-10
status: complete
---

# Phase 201 Plan 06: Elapsed Time and the Colony-Wide Running Total Summary

**Every attempt-bound cost block now also states how long that attempt took (sourced only from its own recorded start/end, sentinel-marked when incomplete), the same block is guaranteed to be the last thing on every work-verdict closeout screen, and the status dashboard shows a colony-wide running total that provably equals the sum of the per-phase rows and attempts it was built from.**

## Performance

- **Duration:** 45 min (approx.)
- **Started:** 2026-09-10T14:22:00Z (approx.)
- **Completed:** 2026-09-10T14:22:00Z (see Task Commits for the real commit-time span, 16:08:41–16:19:59 local)
- **Tasks:** 3
- **Files modified:** 5 (1 created, 4 modified)

## Accomplishments

- `spendElapsedFigure(startedAt, completedAt)` in `cmd/spend_cost_line.go` parses an attempt's own `StartedAt`/`CompletedAt` (RFC3339Nano, the format every production `buildAttemptRecord` writer already uses) and renders elapsed time; either timestamp being absent, unparseable, or `CompletedAt` preceding `StartedAt` renders the identical dash sentinel the unreported-cost figure uses. `renderSpendCostLine(phase)` -- the exact function `cmd/ceremony_cmd.go`'s real build/continue closing screen already calls via `appendSpendCostLine` -- now looks up the phase's latest build attempt (`loadLatestBuildAttempt`) and joins this figure onto the existing cost block as a new `Elapsed: ` line, still exactly one block per screen.
- `renderSpendCostLineFromLedgers(ledgers)` keeps its pre-existing signature and behavior byte-for-byte: it is now a thin wrapper around the new shared `renderSpendCostLineBlock(ledgers, elapsedFigure)` body, always called with `elapsedFigure=""`. Every one of the file's 9 pre-existing tests (several of which explicitly assert the dash sentinel appears NOWHERE in a fully-measured block) passes unmodified, proving the split.
- `cmd/lifecycle_closeout.go`'s `appendLifecycleCloseoutVisual` now appends the one cost-and-time block as the final element of its rendered output for every closeout carrying a `colony.WorkOutcome` verdict (the six-verdict vocabulary from plan 201-04) -- reusing the existing `appendSpendCostLine` helper rather than inventing a second placement rule, so "exactly one, last on screen" is structural. A closeout with no verdict (colonize, plan, init, entomb, pause, resume, seal) is unaffected; a blocked run still shows the figures it recorded before blocking, since the block is read fresh from the durable ledgers rather than derived from the verdict itself.
- `cmd/status.go` gained `computeColonyRunningSpendTotal(state)`, which sums elapsed time and reported cost across every phase in the colony's plan through the same durable loaders (`loadSpendLedgersForPhase` for cost -- accumulating every attempt's rows exactly as the per-phase block already does; `loadLatestBuildAttempt` for elapsed, matching Task 1's own attempt-bound scope), and `renderColonyRunningSpendTotal`, which renders both totals plus the unreported/unmeasured counts, named as such, wired into both the visual dashboard (`renderDashboard`) and the JSON envelope (`buildStatusResult`'s new `colony_running_total` key). The section is omitted entirely for a colony that has recorded nothing at all.
- All three tasks followed RED-GREEN TDD: each is two commits (a failing `test(...)` commit proving the assertion could not pass beforehand, followed by the `feat(...)` commit that makes it pass), for 6 commits total.

## Task Commits

1. **Task 1: Measure and render elapsed time for the exact attempt** - `84016ff5` (test, RED) + `b44868d8` (feat, GREEN)
2. **Task 2: Attach the block to every closeout lane** - `b2a28841` (test, RED) + `97b4143d` (feat, GREEN)
3. **Task 3: Show the running colony total in status** - `f1c5f597` (test, RED) + `72ada873` (feat, GREEN)

**Plan metadata:** committed alongside this summary.

## Files Created/Modified

- `cmd/spend_cost_line.go` - `spendElapsedFigure`, `renderSpendCostLineBlock`, elapsed-aware `renderSpendCostLine(phase)`
- `cmd/spend_cost_line_test.go` - elapsed-time tests (Task 1), equal-cost-and-time-block-across-verdicts tests (Task 2)
- `cmd/lifecycle_closeout.go` - `appendLifecycleCloseoutSpendLine`, wired into `appendLifecycleCloseoutVisual`
- `cmd/status.go` - `colonyRunningSpendTotal`, `computeColonyRunningSpendTotal`, `renderColonyRunningSpendTotal`, wired into `renderDashboard` and `buildStatusResult`
- `cmd/status_running_total_test.go` (new) - `TestStatusRunningTotalEqualsTheSumOfItsRows`, `TestStatusRunningTotalWritesNothing`

## Decisions Made

See `key-decisions` in the frontmatter for the full reasoning; in short: elapsed time is scoped to the phase-aware `renderSpendCostLine(phase)` entry point only (never the ledgers-only function every pre-existing test calls), a genuinely absent attempt renders no elapsed line at all (distinct from an incomplete one, which renders the sentinel), the closeout block gate is a single `WorkOutcome != nil` check rather than a workflow allowlist, and the colony total sums cost across every attempt but elapsed from only each phase's latest attempt.

## Deviations from Plan

None - plan executed exactly as written. Both `<action>` narratives and the task-level `<acceptance_criteria>` were followed directly; no Rule 1-3 auto-fixes were needed.

---

**Total deviations:** 0. **Impact on plan:** None.

## Issues Encountered

- `TestEveryLifecycleCommandEndsWithNextAction` (pre-existing, not touched by this plan) failed once inside a broad `-run 'Spend|Cost|Status|Lifecycle|Closeout|WorkOutcome'` sweep with `storage: repository containment refused: data root path must not be empty`, then passed cleanly in isolation (`go test ./cmd -run '^TestEveryLifecycleCommandEndsWithNextAction$'`). This matches the project's own documented isolated-child resource-contention finding under parallel load, already recorded in `201-04-SUMMARY.md` and `201-05-SUMMARY.md`'s own Issues Encountered sections and in this repo's "Suite ceiling measured" project memory -- a pre-existing environment limitation, not a regression introduced here.
- Per the executor's verification policy (targeted runs preferred over the ~20-minute full `go test ./cmd` sweep, which hits the same documented environment ceiling), verification for this plan ran: every task-level `<verify>` command individually, the full `cmd/spend_cost_line_test.go` file, the Phase 196 `cmd/ceremony_closeout_spend_test.go` regression suite, the Phase 201-04 `TestEveryWorkVerdictRendersTheSameCeremonySlots`/`TestCloseoutCeremonyIsEqualAcrossVerdicts` family, the full `Status`-matched test set, and a combined `-run 'Spend|Cost|Status|Lifecycle|Closeout|WorkOutcome'` sweep -- all clean except the one documented pre-existing flake above. `go build ./...` and `go vet ./cmd` are both clean.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 201-12 ("append the largest timing segment to this same line") has a stable `Elapsed: ` line to extend: `renderSpendCostLineBlock`'s doc comment explicitly reserves that extension point, and no segment logic was implemented here.
- Plans 201-07/201-10/201-15 (also declaring `WORK-05`, and 201-07/201-12/201-15 also declaring `CEC-06`) inherit a proven, tested cost-and-time/running-total foundation to build on.
- **Open gap, explicitly out of this plan's own declared file scope (see D2's `human_judgment: true` rationale above):** no production call site anywhere in the tree yet sets `LifecycleCloseoutDetails.WorkOutcome`, so the `appendLifecycleCloseoutVisual` cost-and-time block this plan adds is fully built and tested but will not visibly render on any real `aether run` (autopilot) screen until a later plan wires a real verdict into a live call site -- the same open gap `201-05-SUMMARY.md` already documented one layer down (the verdict-carrying closeout builders themselves were built-but-unwired there; here, the block that would render for them is built-but-unwired too). `aether build`/`aether continue`'s own direct lane (`cmd/ceremony_cmd.go`) is unaffected by this gap -- it already had the cost line since Phase 196 and now also carries elapsed time live, via `renderSpendCostLine(phase)` directly.
- `go build ./...` and `go vet ./cmd` are clean. Every task-level `<verify>` command from `201-06-PLAN.md` passes individually.
- Ready for `201-07-PLAN.md`.

---
*Phase: 201-queen-led-work-cycle*
*Completed: 2026-09-10*

## Self-Check: PASSED

- Created file verified present on disk: `cmd/status_running_total_test.go`.
- Task commit hashes verified present in `git log`: `84016ff5`, `b44868d8`, `b2a28841`, `97b4143d`, `f1c5f597`, `72ada873`.
- Re-ran plan-level `<verification>`: `go test ./cmd -run '^(TestElapsedTimeComesFromTheAttemptTimestamps|TestMissingTimestampRendersTheUnmeasuredSentinel|TestCostBlockSaysWhatItCounts|TestNoAttemptMeansNoElapsedLineAtAll)$'`, `go test ./cmd -run '^TestSpendPipelineIsReachableEndToEnd$'`, `go test ./cmd -run '^(TestEveryVerdictEndsWithOneCostAndTimeBlock|TestBlockedRunStillReportsWhatItSpent|TestNoWorkVerdictMeansNoCostAndTimeBlock)$'`, `go test ./cmd -run '^(TestBuildEndsWithOneCostLine|TestContinueEndsWithOneCostLine|TestCloseoutCostLineCountIsAssertedNotAssumed|TestNoLaneRendersTwoCostLines)$'`, `go test ./cmd -run '^(TestStatusRunningTotalEqualsTheSumOfItsRows|TestStatusRunningTotalWritesNothing)$'` -- all PASS. `go build ./...` and `go vet ./cmd` clean.
