---
phase: 194-the-queen-decides-the-team
plan: 08
subsystem: orchestration
tags: [queen, spawn-budget, continue, build-manifest, proof-test, windows-ledger, D-11]

# Dependency graph
requires:
  - phase: 194-01
    provides: "queenForcedReviewersForPhase / queenRiskSignalTable — the forced-reviewer derivation this plan's proof tests dispatch against at the continue boundary"
  - phase: 194-02
    provides: "the shrunken build floor (builder alone / nil on discovery) this plan measures on the build side"
  - phase: 194-05
    provides: "queenFallbackTeam — the no-proposal team this plan's fallback-path assertion measures"
provides:
  - "countWorkersAcrossBothBoundaries — the one function every future proof of \"the number the owner feels\" should call, so build and continue are never counted by two disagreeing walkers"
  - "TestOneTaskBugFixIsOneWorkerPlusChecks — the measured 2026-08-22 eight-worker baseline, now proven at exactly one worker on both the proposal path and the no-proposal fallback path, with a heavy-request guard row"
  - "TestNoCasteIsDispatchedAtBothBoundaries — the real-dispatch-list proof that closes .planning/WINDOWS.md #1"
  - "the folded todo 2026-08-21-weight-classes-pipeline-fits-the-task.md is resolved (its own resolution condition, per 194-CONTEXT.md, was this test passing)"
affects: [196-cost-line, 197-closing-card, gsd-ship]

# Actuals (#2632)
actuals:
  tokens: 3425
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "One shared counter (countWorkersAcrossBothBoundaries) for both the total-dispatch-count claim and the double-dispatch claim: an overlap surfaces as total != len(union), so the two proofs cannot silently disagree"
    - "A fixture that could pass vacuously (an inert pipeline, or a fixture matching no signal) gets its own guard assertion in the same test, not a separate follow-up — the heavy-request row in task 1 and the non-empty-continue-set check in task 2"

key-files:
  created:
    - cmd/one_task_bug_fix_test.go
    - cmd/boundary_double_dispatch_test.go
  modified:
    - .planning/WINDOWS.md

key-decisions:
  - "countWorkersAcrossBothBoundaries takes reasons as a single map[string]string, not the variadic map[string]string the underlying functions accept — the plan's own signature (one map argument) is simpler for every call site in both this plan's files, and the underlying variadic parameter accepts a single map value identically to a 1-element variadic slice."
  - "Task 2's intersection-emptiness assertion is backed by a second, independent form of the same claim reusing the SAME helper (total dispatch count vs. distinct-caste union count) rather than a second hand-rolled walker, per the plan's own instruction that two counters which could disagree prove nothing."
  - "Chose production-mode phases naming the credentials/auth and database-migration signals (matching wording already used in 194-05's TestOtherFlowsKeepTheirRequiredSets for the migration case) as the two fixtures, because these are the exact conditions .planning/WINDOWS.md #1's own description names: a production/security phase where build and continue used to independently derive the same required caste."

patterns-established:
  - "A broken-window ledger entry is marked fixed via `gsd-tools windows fixed <id>` only in the same task, strictly after its proving test has been run and shown green — never in a separate step, never on the strength of a SUMMARY claim."

requirements-completed: [TEAM-01, TEAM-04, TEAM-05]

coverage:
  - id: D1
    description: "A one-task bug-fix phase is sent one worker plus the program's own checks, counting build and continue together, on the proposal path AND on the no-proposal fallback path."
    requirement: TEAM-04
    verification:
      - kind: unit
        ref: "cmd/one_task_bug_fix_test.go#TestOneTaskBugFixIsOneWorkerPlusChecks"
        status: pass
    human_judgment: false
  - id: D2
    description: "The same fixture with an explicit heavy request produces more than one worker, so the proof cannot pass by the pipeline having become inert."
    requirement: TEAM-05
    verification:
      - kind: unit
        ref: "cmd/one_task_bug_fix_test.go#TestOneTaskBugFixIsOneWorkerPlusChecks"
        status: pass
    human_judgment: false
  - id: D3
    description: "For a production-mode phase with a security signal and no proposal, and separately for a production-mode phase with a migration signal and no proposal, no caste appears in both the build dispatch list and the continue dispatch list — closing .planning/WINDOWS.md #1."
    requirement: TEAM-01
    verification:
      - kind: unit
        ref: "cmd/boundary_double_dispatch_test.go#TestNoCasteIsDispatchedAtBothBoundaries"
        status: pass
    human_judgment: false
  - id: D4
    description: "Broken-window entry 1 is marked fixed only after the proving test passed, never on the strength of a summary."
    verification:
      - kind: other
        ref: ".planning/WINDOWS.md frontmatter open_count: 0, entry 1 status: fixed, resolved_at set after both new tests passed locally (see Task Commits ordering below)"
        status: pass
    human_judgment: false
  - id: D5
    description: "The measured worker count for the sample one-task bug fix is recorded against the 2026-08-22 baseline of eight."
    verification:
      - kind: other
        ref: "See 'The Measured Number' section below — fallback path 1, proposal path 1, heavy-request guard row 4 (build 1 + continue 3), all against a measured baseline of 8"
        status: pass
    human_judgment: false

# Metrics
duration: 40min
completed: 2026-08-23
status: complete
---

# Phase 194 Plan 8: The Number The Owner Feels, Proven And Closed Summary

**A one-task bug fix now costs exactly one AI worker — on the proposal path and the no-proposal fallback path alike, counted across build and continue together — down from the eight measured on 2026-08-22, and the double-dispatch defect recorded as `.planning/WINDOWS.md` #1 is closed by a test that walks the real build and continue dispatch lists for a security-signal phase and a migration-signal phase.**

## Performance

- **Duration:** ~40 min
- **Started:** 2026-08-23T15:31:00Z (approx)
- **Completed:** 2026-08-23T16:11:08Z
- **Tasks:** 2 completed
- **Files modified:** 3 (2 new: `cmd/one_task_bug_fix_test.go`, `cmd/boundary_double_dispatch_test.go`; 1 modified: `.planning/WINDOWS.md`)

## The Measured Number

| Scenario | Measured 2026-08-22 baseline | This plan's proof |
|---|---|---|
| One-task bug fix, no proposal (autopilot / unattended path) | 8 workers (5 build: builder, watcher, auditor, probe, tracker; 3 continue: watcher, auditor, probe) | **1 worker** (builder only) — `TestOneTaskBugFixIsOneWorkerPlusChecks` |
| Same phase, proposal naming the implementation worker with a per-worker reason | not separately measured (the judged path was always cheaper) | **1 worker** (builder only) |
| Same phase, explicit `--heavy` request (guard row — proves the pipeline is not simply inert) | n/a | **4 workers** (1 build + 3 continue: gatekeeper, auditor, probe) |

Both the fallback (no-proposal) path and the proposal path land on exactly one worker, closing the last gap this milestone was built to remove: before Phase 194, the automatic (no-proposal) path was the MORE expensive of the two.

## Accomplishments

- `countWorkersAcrossBothBoundaries(phase, state, reviewDepth, proposedCastes, casteReason, reasons) (total int, castes []string)` (`cmd/one_task_bug_fix_test.go`) — walks the real manifest-facing dispatch constructors, `plannedBuildDispatchesWithJudgement` and `plannedContinueReviewDispatches`, for the same phase and state, and returns the summed dispatch count plus the sorted union of castes across both boundaries. This is the ONE counting function both new tests in this plan call — no second, independently-written walker exists that could silently disagree with it.
- `TestOneTaskBugFixIsOneWorkerPlusChecks` — the measured field case (an ordinary one-task defect fix, no named-risk signal, no perf/security/research vocabulary) proven at exactly 1 total dispatch on the fallback path (`nil` proposal, `nil` reasons) AND on the proposal path (`["builder"]` proposed with a per-worker reason). A guard row on the same phase with an explicit heavy request proves the pipeline still produces the full review panel (4 total) rather than the proof having passed by the system going quiet — the v1.26 lesson this repo already paid for once. Every failure message names the 2026-08-22 baseline of eight.
- `TestNoCasteIsDispatchedAtBothBoundaries` (`cmd/boundary_double_dispatch_test.go`) — walks the same two real dispatch-list constructors for a production-mode phase naming the credentials/auth signal ("Password reset") and, separately, a production-mode phase naming the database-migration signal ("Migrate user data to new schema"), both with no proposal. Asserts the intersection of the build caste set and the continue caste set is empty for both fixtures, with a vacuous-pass guard (the continue set must actually be non-empty, i.e. the fixture must genuinely exercise a forced reviewer) and a second, independent confirmation via `countWorkersAcrossBothBoundaries`'s `total == len(union)` identity — an overlap would make the summed total exceed the distinct-caste count, so the two checks cannot silently disagree.
- `.planning/WINDOWS.md` entry #1 marked `fixed` (`gsd-tools windows fixed 1`), `open_count: 0`, `fixed_count: 1` — run strictly AFTER both new tests were verified green locally, never before. This reopens `/gsd-ship`'s windows gate.
- The folded todo `2026-08-21-weight-classes-pipeline-fits-the-task.md` ("small jobs should get the small machine automatically") is resolved: its own resolution condition, stated in 194-CONTEXT.md, was `TestOneTaskBugFixIsOneWorkerPlusChecks` passing.

## Task Commits

Each task was committed atomically, and the ledger fix was verified before being marked, per the plan's own ordering requirement:

1. **Task 1: The one-task bug fix, counted end to end, on both paths** - `fb409e95` (test) — verified green (`go test ./cmd -run 'TestOneTaskBugFixIsOneWorkerPlusChecks' -count=1` and the full `go test ./cmd -count=1`, 426.8s, 0 failures) BEFORE this commit.
2. **Task 2: No worker is summoned at both boundaries, and the broken window closes on the test** - `4a4fe736` (test) — `TestNoCasteIsDispatchedAtBothBoundaries` verified green locally FIRST, THEN `gsd-tools windows fixed 1` was run, THEN this single commit carries both the new test file and the resulting `.planning/WINDOWS.md` diff together (the plan's own `<files>` list names both files for this task).

**Plan metadata:** (this commit, following)

## Files Created/Modified

- `cmd/one_task_bug_fix_test.go` — `oneTaskBugFixPhase`, `countWorkersAcrossBothBoundaries`, `TestOneTaskBugFixIsOneWorkerPlusChecks`.
- `cmd/boundary_double_dispatch_test.go` — `boundaryDoubleDispatchPhase`, `TestNoCasteIsDispatchedAtBothBoundaries`.
- `.planning/WINDOWS.md` — entry #1 status `open` → `fixed`, `resolved_at` timestamp set, `open_count: 1` → `0`, `fixed_count: 0` → `1`.

## Decisions Made

See `key-decisions` in frontmatter. The two most consequential: (1) `countWorkersAcrossBothBoundaries` takes a single `reasons map[string]string` parameter rather than mirroring the underlying functions' variadic signature, since every call site in this plan only ever supplies zero or one reasons map; (2) task 2's intersection check is backed by a second, independent confirmation reusing the SAME helper's `total == len(union)` identity, so an overlap cannot pass one check while failing the other silently.

## Deviations from Plan

None — plan executed exactly as written. Both tests passed on the first implementation attempt with no fix-up cycles; no production code was touched (this plan's own scope explicitly excludes it — it adds only new test files and a ledger edit).

## Issues Encountered

None. Both `go test ./cmd -count=1` full-suite runs (after task 1's commit and again after task 2's commit) passed with zero failures — no timing-flaky tests surfaced in this session, unlike several prior plans in this phase that hit the documented pre-existing flaky tests under load.

## Known Stubs

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `.planning/WINDOWS.md` is fully closed (`open_count: 0`) — `/gsd-ship`'s windows gate no longer blocks on this defect.
- `countWorkersAcrossBothBoundaries` is available for plan 196 (the cost line) if a future proof needs the same "one number across both boundaries" measurement.
- Phase 194's remaining plan, 194-09 (the CLAUDE.md rewrite), is unaffected by this plan and can proceed — this plan touched no documentation.
- No blockers for 194-09.

---
*Phase: 194-the-queen-decides-the-team*
*Completed: 2026-08-23*

## Self-Check: PASSED

- FOUND: commit `fb409e95` (Task 1) in `git log --oneline --all`
- FOUND: commit `4a4fe736` (Task 2) in `git log --oneline --all`
- FOUND: `cmd/one_task_bug_fix_test.go`
- FOUND: `cmd/boundary_double_dispatch_test.go`
- `go build ./...` — pass
- `go vet ./...` — pass
- `go test ./cmd -run 'TestOneTaskBugFixIsOneWorkerPlusChecks|TestNoCasteIsDispatchedAtBothBoundaries' -count=1` — pass
- `go test ./cmd -count=1` — pass (two full runs, 426.8s and 386.7s, zero failures both times)
- `grep -c 'countWorkersAcrossBothBoundaries' cmd/boundary_double_dispatch_test.go` — 3 (≥ 1)
- `.planning/WINDOWS.md` — `open_count: 0`, entry 1 `status: fixed` with `resolved_at` set
