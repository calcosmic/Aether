---
phase: 195-coherent-jobs
plan: 03
subsystem: build-orchestration
tags: [go, dispatch-planning, coherent-jobs, worker-briefs, worktrees, tdd]

requires:
  - phase: 195-coherent-jobs
    provides: the pure coherent-job planner (195-01) and the additive task-receipt wire contract (195-02)
provides:
  - Coherent jobs planned before waves and worktree ownership on every build lane
  - Repeatable --job-proposal transport with validated, visible accept/refuse/repair decisions
  - Grouped worker briefs that carry every covered task's own contract exactly once
  - Covered-task-aware continue recovery and assessment
  - CalVault six-batch in-repo regression proving one Builder with six ordered covered IDs
affects: [195-04, 195-06, 195-08, 195-10]

actuals:
  tokens: 4300
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns: [pre-wave canonical grouping, additive grouped-brief section, covered-task iteration at every dispatch consumer]

key-files:
  created: []
  modified:
    - cmd/codex_build.go
    - cmd/codex_workflow_cmds.go
    - cmd/codex_continue.go
    - cmd/codex_visuals.go
    - cmd/dispatch_coalesce_test.go
    - cmd/codex_build_worktree_test.go
    - cmd/build_print_brief_test.go
    - cmd/testdata/golden_build.txt
    - .aether/schemas/completion-packet.schema.json

key-decisions:
  - "Grouping runs before task waves and before declared-worktree-ownership validation, so declared overlap becomes one coherent job instead of a pre-dispatch conflict error."
  - "Grouped briefs get an additive `## Covered Task Contracts` section carrying only what has no legacy single-task home (evidence requirements and otherwise-unlisted declared paths); a one-task brief stays byte-for-byte unchanged."
  - "Every consumer of a dispatch iterates `dispatchCoveredTaskIDs`, never the bare `TaskID` — abandoned-build recovery and continue assessment both previously lost every task after the first."
  - "Owner-facing wording moved from `dependent tasks` to `covered tasks`; automatic grouping is no longer dependency-only, so the old phrase had become false."
  - "`coalesceSequentialDispatches` is retained as a definition with no live production caller; the planner is the sole grouping authority."

patterns-established:
  - "Pre-wave canonical grouping: the planner is called once, before waves, worktree ownership, and deterministic naming, so one job plan is shared by in-repo and worktree modes."
  - "Additive brief sections: new grouped-job content never re-renders values an existing single-task section already emits, keeping legacy brief output stable and paths unduplicated."

requirements-completed: [JOBS-01, JOBS-02, JOBS-03, JOBS-04]

coverage:
  - id: D1
    description: "Six chained CalVault copy steps plan as one in-repo Builder covering all six task IDs in order"
    requirement: JOBS-01
    verification:
      - kind: integration
        ref: "cmd/dispatch_coalesce_test.go#TestCalVaultSixBatchesBecomeOneInRepoJob"
        status: pass
    human_judgment: false
  - id: D2
    description: "A grouped brief carries every covered task's goal, constraints, hints, criteria, evidence and declared paths, each exactly once"
    requirement: JOBS-03
    verification:
      - kind: integration
        ref: "cmd/dispatch_coalesce_test.go#TestGroupedJobBriefCarriesEveryTaskContract"
        status: pass
    human_judgment: false
  - id: D3
    description: "Independent work still fans out and a cross-caste chain stays split"
    requirement: JOBS-01
    verification:
      - kind: integration
        ref: "cmd/dispatch_coalesce_test.go#TestIndependentTasksStillFanOut, #TestChainAcrossDifferentCastesDoesNotMerge"
        status: pass
    human_judgment: false
  - id: D4
    description: "The same job plan, identity and wave are produced in in-repo and worktree parallel modes, and declared worktree overlap groups instead of erroring"
    requirement: JOBS-04
    verification:
      - kind: integration
        ref: "cmd/dispatch_coalesce_test.go#TestCoherentJobsMatchAcrossParallelModes, cmd/codex_build_worktree_test.go#TestBuildWorktreeModeGroupsDeclaredOverlapBeforeDispatch"
        status: pass
    human_judgment: false
  - id: D5
    description: "Grouped worker names seed from ordered covered IDs while single-task names keep their pre-195 seed; every covered task keeps credit at continue"
    requirement: JOBS-02
    verification:
      - kind: unit
        ref: "cmd/dispatch_coalesce_test.go#TestGroupedWorkerNameUsesOrderedCoveredIDs, #TestSingleTaskWorkerNameIsStable, #TestGroupedJobCreditsEveryCoveredTaskDuringContinue, cmd/merged_dispatch_task_credit_test.go#TestMergedDispatchCreditsEveryCoveredTask"
        status: pass
    human_judgment: false
  - id: D6
    description: "Selected-task builds never pull unselected or completed work into a job, and an invalid or cyclic proposal creates no attempt or manifest"
    requirement: JOBS-01
    verification:
      - kind: integration
        ref: "cmd/dispatch_coalesce_test.go#TestSelectedTaskGroupingStaysInScope, cmd/codex_build_test.go#TestBuildRejectsInvalidJobProposalBeforeAttempt"
        status: pass
    human_judgment: false

duration: 95min
completed: 2026-08-27
status: complete
---

# Phase 195 Plan 03: Coherent Jobs Become the Canonical Grouping Pass Summary

**Coherent-job planning now runs once, before waves and worktree ownership, on every build lane — so grouping changes the real dispatch list, the brief, the worker's name and the recovery path, and six chained CalVault copy steps become one Builder that still carries all six task contracts.**

## Performance

- **Duration:** ~95 min across two sessions (interrupted executor + in-place recovery)
- **Completed:** 2026-08-27
- **Tasks:** 2
- **Files modified:** 9 implementation, test, fixture and schema files

## Accomplishments

- Made `planCoherentJobs` the single grouping authority for direct, plan-only, selected-task, in-repo and worktree builds, called before `taskWaves` and before `validateDeclaredWorktreeOwnership`.
- Added repeatable `--job-proposal` transport into `codexBuildOptions.JobProposals`, with all parse and contract errors returned before any checkpoint, manifest or attempt exists.
- Recorded accept / refuse / repair decisions on the manifest (`job_decisions`) and job name, reason and source on each dispatch, while preserving `task_id` and `covered_task_ids` semantics.
- Added a `## Covered Task Contracts` brief section that emits each covered task's evidence requirements and any declared path not already shown, so nothing a task owns is lost to grouping and no path is printed twice.
- Fixed two places that read only a dispatch's primary `TaskID`: abandoned-build recovery (`abandonedBuildTaskIDs`) and continue assessment (`assessCodexContinue`) now iterate every covered task, so a grouped worker's later tasks stay visible.
- Replaced the now-false owner-facing phrase `dependent tasks` with `covered tasks` in the build summary and its golden fixture.
- Turned the old worktree declared-overlap *rejection* test into a *grouping* test: same-wave declared overlap is now one coherent job rather than a pre-dispatch conflict error.

## Task Commits

1. **Task 1 RED: coherent-job wiring regressions** — `1cfa72ee` (test)
2. **Task 1 GREEN: coherent jobs wired into build planning** — `e4b351d2` (feat)
3. **Task 2 RED: grouped-job compatibility regressions** — `c722d43a` (test)
4. **Task 2 GREEN: every covered task contract carried through grouped jobs** — `faafdff5` (feat)

## Files Created/Modified

- `cmd/codex_build.go` — canonical planner wiring, grouped dispatch metadata, job waves, `renderGroupedDispatchTaskContracts`.
- `cmd/codex_workflow_cmds.go` — repeatable `--job-proposal` decoding into runtime options.
- `cmd/codex_continue.go` — covered-task iteration in abandoned-build recovery and continue assessment.
- `cmd/codex_visuals.go` — `covered tasks` summary wording.
- `cmd/dispatch_coalesce_test.go` — CalVault, grouped-brief, parallel-mode, cross-caste and scope regressions.
- `cmd/codex_build_worktree_test.go` — declared overlap now groups instead of erroring.
- `cmd/build_print_brief_test.go` — fixture task hints made distinct so path-uniqueness assertions are meaningful.
- `cmd/testdata/golden_build.txt` — regenerated for the wording change.
- `.aether/schemas/completion-packet.schema.json` — `job_name` / `job_reason` / `job_source` and the `coherentJobDecision` definition.

## Decisions Made

- Grouping precedes worktree-ownership validation, which converts a former hard failure (same-wave declared overlap) into the intended single job. The old rejection test was rewritten rather than deleted, so the behaviour change is visible in git history.
- The grouped brief section is strictly additive. Goals, constraints, hints and success criteria keep their existing single-task sections, and the new section returns early for a one-task job, so legacy brief output is unchanged byte-for-byte.
- `coalesceSequentialDispatches` was left defined but is called from no production path; removing it was out of scope for this plan and its name still anchors several explanatory comments.

## Deviations from Plan

- **`TestGroupedJobBriefCarriesEveryTaskContract` uses a two-task fixture, not six.** The plan's acceptance criterion says "names all six task criteria/evidence rows". The shipped test names every goal, constraint, hint, criterion, evidence criterion and evidence check for both tasks and additionally asserts each declared path appears exactly once — it fails if any single row is removed, which is the criterion's intent. The six-task shape is separately covered by `TestCalVaultSixBatchesBecomeOneInRepoJob`.
- **`TestCalVaultSixBatchesBecomeOneInRepoJob` was strengthened during recovery.** As left by the interrupted executor it asserted only the worker count. The plan's criterion also requires "exactly one Builder and six ordered CoveredTaskIDs", so caste and ordered-covered-ID assertions were added before the plan was closed.
- **Plan 195-03 was executed across two sessions.** The first (Codex) executor was stopped mid-Task-2 after repeated stall gates, leaving Task 2's implementation uncommitted. Recovery resumed in place per `.planning/HANDOFF.json` rather than redispatching from `c722d43a`.

## Issues Encountered

- The interrupted executor's working set compiled and vetted clean and every Task 2 acceptance test passed once re-run, so it was completed in place rather than discarded.
- `go test ./cmd -count=1 -timeout 25m` hits the timeout on cumulative package runtime (`TestSurveyStaleness` was merely in flight; it passes in 1.9s alone). Recorded in `deferred-items.md`.
- Eleven `cmd` visual-output tests fail under a Claude Code session because they assert the raw `aether <verb>` form that `translateHintCommandsForPlatform` rewrites to `/ant-<verb>` off Codex. All eleven pass with `AETHER_PLATFORM=codex`; pre-existing and unrelated to this plan. Recorded in `deferred-items.md`.
- `pkg/codex`'s `TestAvailabilityProbeRetriesOnlyTimeouts` failed once while the full `cmd` suite ran concurrently and passed 3/3 in isolation — load-sensitive timing, not a regression.

## Known Stubs

None — no placeholder or unwired data path was introduced.

## User Setup Required

None.

## Next Phase Readiness

- Plan 195-04 can admit task receipts against the grouped manifest now that `covered_task_ids` is authoritative assignment scope on real dispatches.
- Plan 195-06 inherits the `job_decisions` manifest surface for external completion.
- Plan 195-08 can prove worktree execution and merge-back against a job plan already proven identical across parallel modes.

## Self-Check: PASSED

- `go build ./...` and `go vet ./cmd/` pass.
- Both tasks' full acceptance commands pass: Task 1's five proposal/wiring tests and Task 2's seven grouped-job tests, plus `TestCoherentJobsMatchAcrossParallelModes`, `TestGroupedBuildSummaryUsesCoveredTasksLanguage` and `TestChainAcrossDifferentCastesDoesNotMerge`.
- Every non-`cmd` package passes; the one failure seen was reproduced as load-sensitive and passes on rerun.
- No production build path calls `coalesceSequentialDispatches`.
- All four RED/GREEN commits are present in git history.

---
*Phase: 195-coherent-jobs*
*Completed: 2026-08-27*
