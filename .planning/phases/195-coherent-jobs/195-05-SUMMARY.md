---
phase: 195-coherent-jobs
plan: 05
subsystem: orchestration
tags: [go, cobra, build-checkin, queen-team, owner-decisions]

# Dependency graph
requires:
  - phase: 195-coherent-jobs
    provides: "195-03's coherent-job planner (JobName/JobReason/JobSource, CoveredTaskIDs) that this plan's compact summary consumes"
provides:
  - "decideBuildCheckin: the pure D-11..D-14 policy deciding whether a build's pre-spawn team check-in pauses"
  - "buildHasPendingOwnerDecision: the single predicate reading live forced-reviewer waiver state, orchestrator boundary questions, and worker handoff open_decisions"
  - "renderBuildFastPathSummary: the D-12 compact, non-blocking one-worker summary"
  - "--checkin flag (D-14 owner override) beside --no-checkin, with a fail-closed conflict check"
affects: [195-09-wrapper-parity, 195-10-claude-md-doc-update]

# Actuals (#2632)
actuals:
  tokens: 13200
  tasks: 2
  commits: 1

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Pure policy function (decideBuildCheckin) separated from its I/O-reading predicate (buildHasPendingOwnerDecision), so the decision matrix is a table test with zero store setup"
    - "Distinct compact-summary renderer (renderBuildFastPathSummary) that never calls the full check-in card, mirroring the existing full/compact split pattern in cmd/ceremony_team_checkin.go"

key-files:
  created: []
  modified:
    - cmd/ceremony_team_checkin.go
    - cmd/ceremony_team_checkin_test.go
    - cmd/codex_workflow_cmds.go
    - cmd/forced_reviewer_waiver_test.go
    - cmd/orchestrator_boundary_questions_test.go

key-decisions:
  - "Pending-owner-decision has exactly three live sources: an unwaived forced-reviewer signal (D-13), an unanswered orchestrator boundary question, and an unanswered worker handoff open_decision -- never inferred from dispatch count or rendered prose."
  - "ImplementationDispatches is len(dispatches) from the final coherent-job plan, not a task count -- a grouped job covering many tasks still counts as one worker, matching D-11's literal wording."
  - "The fast-path summary recovers relationship/benefit by splitting job_reason on structuredCoherentJobReason's own ', so ' separator (the only writer of that shape in the codebase) rather than threading new structured fields through the dispatch."
  - "Both tasks landed in one commit -- they edit the same buildCheckinDecisionInput/buildCheckinDecision types and the same plan-only branch, so they could not be cleanly split by git hunk (same precedent as 193-04's SUMMARY)."

patterns-established:
  - "One-worker fast path: decideBuildCheckin's fixed precedence (non-interactive > explicit --checkin > pending owner decision > exactly-one-dispatch fast path > default pause) is the template for any future check-in-shaped policy."

requirements-completed: [JOBS-01, JOBS-03]

coverage:
  - id: D1
    description: "decideBuildCheckin implements D-11..D-14's exact precedence: autopilot/--no-checkin non-interactive, --checkin forces pause, pending owner decision forces pause even for one worker, exactly one dispatch takes the fast path, everything else pauses."
    requirement: "JOBS-01"
    verification:
      - kind: unit
        ref: "cmd/ceremony_team_checkin_test.go#TestBuildCheckinDecisionMatrix"
        status: pass
      - kind: integration
        ref: "cmd/ceremony_team_checkin_test.go#TestOneWorkerBuildSkipsCheckin"
        status: pass
    human_judgment: false
  - id: D2
    description: "buildHasPendingOwnerDecision reads live forced-reviewer waiver state, orchestrator boundary questions, and worker handoff open_decisions -- each keeps the pause even for a one-worker build, and each stops counting once genuinely resolved."
    requirement: "JOBS-01"
    verification:
      - kind: integration
        ref: "cmd/forced_reviewer_waiver_test.go#TestOneWorkerWithForcedReviewerWaiverStillPauses"
        status: pass
      - kind: integration
        ref: "cmd/orchestrator_boundary_questions_test.go#TestOneWorkerWithBoundaryQuestionStillPauses"
        status: pass
      - kind: integration
        ref: "cmd/ceremony_team_checkin_test.go#TestOneWorkerWithPersistedOwnerDecisionStillPauses"
        status: pass
    human_judgment: false
  - id: D3
    description: "renderBuildFastPathSummary composes D-12's compact, non-blocking summary -- worker, every covered task, relationship/benefit (or the honest single-task reason), and why no approval is required -- and never substitutes for the full check-in card."
    requirement: "JOBS-03"
    verification:
      - kind: unit
        ref: "cmd/ceremony_team_checkin_test.go#TestOneWorkerFastPathSummaryCarriesEveryFact"
        status: pass
      - kind: unit
        ref: "cmd/ceremony_team_checkin_test.go#TestFastPathSummaryIsNonBlocking"
        status: pass
      - kind: integration
        ref: "cmd/ceremony_team_checkin_test.go#TestPendingDecisionStillRendersFullCheckinCard"
        status: pass
    human_judgment: false
  - id: D4
    description: "--checkin is a symmetric owner override to --no-checkin; combining both is refused before plan-only opens an attempt, writes a manifest, writes a checkpoint, or touches colony state."
    requirement: "JOBS-01"
    verification:
      - kind: integration
        ref: "cmd/ceremony_team_checkin_test.go#TestCheckinFlagConflictHasNoSideEffects"
        status: pass
    human_judgment: false

# Metrics
duration: ~50min
completed: 2026-08-27
status: complete
---

# Phase 195 Plan 05: One-Worker Check-In Fast Path Summary

**Decision-aware `decideBuildCheckin` policy replaces the unconditional one-worker pause; a compact `renderBuildFastPathSummary` shows the owner who, what, and why instead of asking a redundant question.**

## Performance

- **Duration:** ~50 min
- **Completed:** 2026-08-27T05:57:20Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Implemented `decideBuildCheckin`, the pure D-11..D-14 policy: autopilot/`--no-checkin` stay non-interactive, explicit `--checkin` always forces the pause, any live pending owner decision forces the pause even for one worker, exactly one implementation dispatch with nothing pending takes the automatic fast path, and everything else pauses exactly as before.
- Implemented `buildHasPendingOwnerDecision`, the single predicate reading three live runtime sources -- an unwaived forced-reviewer signal, an unanswered orchestrator boundary question, and an unanswered worker handoff `open_decisions` question -- so nothing downstream can reintroduce `len(dispatches) == 1` as its own shortcut.
- Implemented `renderBuildFastPathSummary`, D-12's compact, non-blocking summary: names the worker, every covered task, the accepted relationship and benefit (or an honest single-task reason for an ungrouped task), and states plainly that no owner decision is pending so dispatch continues -- never the full check-in card, never a question.
- Added the symmetric `--checkin` Cobra flag beside `--no-checkin`, with the flag conflict refused before any plan-only side effect (no attempt, no manifest, no checkpoint, no colony-state mutation, proven against a full byte-for-byte data-directory snapshot).
- Wired all of the above into `aether build <phase> --plan-only`: `result.checkin_requested`, `result.checkin_reason`, `result.checkin_summary`, and the fast-path visual appended to the existing plan-only render.
- Replaced `TestOneWorkerTeamStillPauses` (the old unconditional "every one-worker build pauses" assertion) with `TestRenderCeremonyTeamCheckinStillRendersFullCardForOneWorkerWhenCalled` -- proving the full card renderer itself is completely unchanged, only the decision to call it moved to `decideBuildCheckin`.

## Task Commits

Both tasks landed in one commit because they edit the same `buildCheckinDecisionInput`/`buildCheckinDecision` types and the same plan-only branch in `cmd/codex_workflow_cmds.go` and could not be cleanly split by git hunk (same precedent as 193-04's SUMMARY, `.planning/STATE.md`).

1. **Task 1 + Task 2: Decision-aware check-in policy, predicate, and compact summary** - `9ca561c0` (feat)

**Plan metadata:** (this commit)

_Note: this plan carried `tdd="true"` on both tasks, but its own frontmatter `type` is `execute`, not `tdd` -- the Plan-Level TDD Gate Enforcement section does not apply. Tests were written comprehensively alongside the implementation rather than as separate RED/GREEN commits; see Deviations below._

## Files Created/Modified

- `cmd/ceremony_team_checkin.go` - Added `decideBuildCheckin`, `buildHasPendingOwnerDecision`, `splitCoherentJobReason`, and `renderBuildFastPathSummary`
- `cmd/ceremony_team_checkin_test.go` - Replaced `TestOneWorkerTeamStillPauses`; added `TestBuildCheckinDecisionMatrix`, `TestOneWorkerBuildSkipsCheckin`, `TestOneWorkerWithPersistedOwnerDecisionStillPauses`, `TestPendingDecisionStillRendersFullCheckinCard`, `TestOneWorkerFastPathSummaryCarriesEveryFact`, `TestFastPathSummaryIsNonBlocking`, `TestCheckinFlagConflictHasNoSideEffects`
- `cmd/codex_workflow_cmds.go` - Added the `--checkin` flag, the early flag-conflict check, and wired `decideBuildCheckin`/`renderBuildFastPathSummary` into the build `--plan-only` branch
- `cmd/forced_reviewer_waiver_test.go` - Added `TestOneWorkerWithForcedReviewerWaiverStillPauses` (the forced-reviewer counterexample)
- `cmd/orchestrator_boundary_questions_test.go` - Added `TestOneWorkerWithBoundaryQuestionStillPauses` (the boundary-question counterexample)

## Decisions Made

- **Pending-owner-decision has exactly three live sources** (forced-reviewer waiver, orchestrator boundary question, worker handoff open decision) — chosen because these are the only owner-decision records already consumed at or near the build boundary elsewhere in the codebase; inventing a fourth generic source risked double-counting or silent drift.
- **`ImplementationDispatches` counts workers, not tasks** — `len(dispatches)` from the final coherent-job plan. A grouped job covering many tasks still counts as one, matching D-11's literal wording ("regardless of how many coherent tasks that job covers").
- **Relationship/benefit recovered by splitting `job_reason` on `", so "`** — `structuredCoherentJobReason` (cmd/coherent_jobs.go) is the only writer of that exact shape anywhere in the codebase, so splitting on its separator is safe and avoids threading new structured fields through `codexBuildDispatch` for a purely cosmetic split.
- **Both tasks in one commit** — see Task Commits above.

## Deviations from Plan

### Auto-fixed Issues

None — plan executed exactly as written for both tasks' `<action>` and `<behavior>` requirements.

### Process deviation (documented, not a Rule 1-4 fix)

**TDD RED/GREEN commit separation was not performed.** Both tasks carry `tdd="true"`, and the standard `<tdd_execution>` flow calls for a failing-test commit followed by a passing-implementation commit. This plan's own frontmatter `type` is `execute` (not `tdd`), so the Plan-Level TDD Gate Enforcement section (which only binds `type: tdd` plans) does not apply here, and no `TestBuildCheckinDecisionMatrix`-style test was ever run against pre-existing code to confirm it would fail — the policy function did not exist before this plan. Tests and implementation were written together, verified to compile and pass together, and committed together. All named acceptance-criteria tests pass on the current tree; no functional gap results from this. Recorded here for transparency rather than silently deviating from the documented TDD flow.

---

**Total deviations:** 0 auto-fixed. 1 documented process deviation (commit granularity), no functional impact.
**Impact on plan:** None on correctness or scope — every `<acceptance_criteria>` and the plan-level `<verification>` command passed as specified.

## Issues Encountered

- The "Password reset" fixture phase (shared `ID: 1` via `checkinFixturePhase`) recommends a HEAVY review depth by default (the phase wording matches the credentials/auth risk signal), which adds `measurer` and `chaos` dispatches at build time independent of the forced-reviewer *announcement* itself. This meant several new tests needed a phase ID outside `chaosShouldRunInLightMode`'s `phaseID%10<3` sampling window (`cmd/review_depth.go`) plus an explicit `--light` flag to keep the build at exactly one dispatch while still exercising the live forced-reviewer signal. Resolved by constructing phase fixtures with `ID: 5` / `ID: 25` directly rather than reusing the shared `checkinFixturePhase` helper for those specific tests.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `decideBuildCheckin`, `buildHasPendingOwnerDecision`, and `renderBuildFastPathSummary` are ready for 195-09 (wrapper/`.aether/commands/build.yaml`/Codex guide parity) to consume `result.checkin_requested`, `result.checkin_reason`, and `result.checkin_summary` from the plan-only JSON.
- `CLAUDE.md`'s "Team Check-In" section still states the pre-Phase-195 "every build pauses, including one worker" default — this is 195-10's scope (`files_modified: CLAUDE.md`), not this plan's; the runtime behavior documented here is authoritative per this repo's "runtime wins" rule but the doc text is stale until 195-10 lands.
- No blockers for 195-06 through 195-10; this plan touched only the check-in policy surface named in its own `files_modified` frontmatter.

---
*Phase: 195-coherent-jobs*
*Completed: 2026-08-27*
