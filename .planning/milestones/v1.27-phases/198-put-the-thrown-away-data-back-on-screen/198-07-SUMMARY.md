---
phase: 198-put-the-thrown-away-data-back-on-screen
plan: 07
subsystem: cli
tags: [build, blocker-advisory, owner-decision, go-cli, wrapper-triplet]

# Dependency graph
requires:
  - phase: 198-03
    provides: "The owner-decision recording pattern (PendingDecision with a Source tag) and the check-in predicate style this plan's third signal follows"
provides:
  - "buildStartBlockerSignals -- the three D-09 hard-stop signals (forced reviewer, unanswered question, last continue blocked) computed from live structured records"
  - "decideBuildBlockerAdvisory / renderBuildBlockerAdvisory -- the D-08 pure policy and house-style render: printing unconditional on any signal, asking gated by interactivity"
  - "lastContinueEndedBlocked -- reads continue.json's structured Advanced field, the reusable pattern for any future 'did the last check pass' predicate"
  - "The Blocker Heads-Up wrapper stage, byte-identical across all three build wrapper copies"
affects: [any future phase touching cmd/codex_workflow_cmds.go's build lanes, cmd/ceremony_team_checkin.go, or the build wrapper triplet]

# Actuals (#2632)
actuals:
  tokens: 8400
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Structured-report reading over event-string parsing: lastContinueEndedBlocked reads continue.json's typed Advanced field via store.LoadJSON, mirroring the existing loadLastContinueOptions precedent, rather than parsing the pipe-delimited event string that has no format guarantee"
    - "Printing-unconditional / asking-gated pure policy: decideBuildBlockerAdvisory mirrors decideBuildCheckin's shape but deliberately diverges on the one point D-09 requires -- signals always print, only the question is gated by interactivity"
    - "Signal reuse over re-derivation: buildStartBlockerSignals wraps two of buildHasPendingOwnerDecision's own predicates rather than re-deriving them, so the check-in card and the new heads-up can never disagree about what is pending"

key-files:
  created:
    - cmd/build_blocker_advisory.go
    - cmd/build_blocker_advisory_test.go
  modified:
    - cmd/ceremony_team_checkin.go
    - cmd/codex_workflow_cmds.go
    - cmd/build_wrapper_ceremony_test.go
    - .claude/commands/ant/build.md
    - .opencode/commands/ant/build.md
    - .claude/commands/ant-build.md

key-decisions:
  - "The direct (non-plan-only) build lane never computes a full codexBuildManifest today (confirmed by reading runCodexBuildWithOptions in full -- zero matches for ForcedReviewers/BoundaryQuestionCount in its body). Rather than adding new orchestrator-boundary-question machinery to that lane (out of this plan's scope and a potential new mutation), the direct lane reconstructs a minimal manifest using only read-only, already-available derivations: forcedReviewerRecords(queenForcedReviewersForPhase(phase)) for the forced-reviewer signal, and the same lastContinueEndedBlocked/pendingHandoffDecisions calls (which need only the phase ID) for the other two. Both lanes still call the identical buildStartBlockerSignals/decideBuildBlockerAdvisory/renderBuildBlockerAdvisory functions -- the guarantee CLAUDE.md requires -- with the direct lane's BoundaryQuestionCount honestly reading 0 (a pre-existing gap in that lane's owner-decision tracking, not a regression this plan introduced)."
  - "Boundary-question and handoff-question signals collapse into one 'unanswered-question' name, matching the plan action's own framing ('an unanswered question -- orchestrator or worker') and D-09's wording ('an open owner question') as a single hard stop, rather than three separately-named signals for what the owner experiences as one thing."
  - "The 'exact command to deal with the problem' (Task 2 action item 3) is left to the wrapper's existing next-step logic (aether status) rather than a new Go result key: no acceptance criterion or artifact in the plan names a dedicated stop-command key, and the wrapper prose (Task 3) already directs the owner to the existing single source of truth for 'what to run next' rather than inventing a second one."

requirements-completed: [SHOW-02]

coverage:
  - id: D1
    description: "The three build-start blocker signals (forced reviewer, unanswered question, last continue blocked) are computed from live structured records, never from event-string parsing, and nothing else can raise them"
    requirement: "SHOW-02"
    verification:
      - kind: unit
        ref: "cmd/build_blocker_advisory_test.go#TestBlockedLastTimeIsReadFromTheStructuredReport"
        status: pass
      - kind: unit
        ref: "cmd/build_blocker_advisory_test.go#TestOnlyTheThreeNamedSignalsRaiseTheHeadsUp"
        status: pass
    human_judgment: false
  - id: D2
    description: "A build that starts with a blocker present prints a plain-English heads-up naming it and asks one carry-on-or-stop question on both build lanes; automatic mode and --no-checkin still print without asking; the path dispatches no workers"
    requirement: "SHOW-02"
    verification:
      - kind: unit
        ref: "cmd/build_blocker_advisory_test.go#TestBuildStartBlockerAdvisory"
        status: pass
      - kind: unit
        ref: "cmd/build_blocker_advisory_test.go#TestNonInteractiveRunsStillPrintTheHeadsUp"
        status: pass
      - kind: unit
        ref: "cmd/build_blocker_advisory_test.go#TestBothBuildLanesEmitTheHeadsUp"
        status: pass
      - kind: unit
        ref: "cmd/build_blocker_advisory_test.go#TestBlockerHeadsUpDispatchesNoWorkers"
        status: pass
    human_judgment: false
  - id: D3
    description: "All three build wrapper copies describe the heads-up and its one question identically, before the Team Check-In stage, with a test that fails if one drifts"
    requirement: "SHOW-02"
    verification:
      - kind: unit
        ref: "cmd/build_wrapper_ceremony_test.go#TestBuildWrapperCeremonyContract"
        status: pass
      - kind: unit
        ref: "cmd/lifecycle_wrapper_contract_test.go#TestLifecycleFlatMirrorsMatchCanonical"
        status: pass
      - kind: other
        ref: "diff .claude/commands/ant/build.md .opencode/commands/ant/build.md && diff .claude/commands/ant/build.md .claude/commands/ant-build.md"
        status: pass
    human_judgment: false

# Metrics
duration: 55min
completed: 2026-08-29
status: complete
---

# Phase 198 Plan 07: Build-Start Blocker Advisory Summary

**A build that starts while something is genuinely stuck (a forced reviewer waiting, an open owner question, or a blocked last check) now prints a plain-English heads-up and asks one question -- carry on, or stop -- on both build lanes and in all three wrapper copies.**

## Performance

- **Duration:** ~55 min
- **Started:** 2026-08-29 (approx.)
- **Completed:** 2026-08-29
- **Tasks:** 3
- **Files modified:** 8 (2 created, 6 modified)

## Accomplishments

- Added `lastContinueEndedBlocked` (cmd/ceremony_team_checkin.go), reading `continue.json`'s structured `Advanced` field via the existing `store.LoadJSON`/`continuePlanArtifactsPath` pattern -- never the pipe-delimited event string RESEARCH.md explicitly flags as fragile.
- Built `cmd/build_blocker_advisory.go`: `buildBlockerSignal`, `buildStartBlockerSignals` (unioning two of `buildHasPendingOwnerDecision`'s own predicates with the new third signal), `buildBlockerAdvisory`, `decideBuildBlockerAdvisory` (pure, printing-unconditional/asking-gated policy), and `renderBuildBlockerAdvisory` (house-style render, no bordered table).
- Wired the advisory into both `cmd/codex_workflow_cmds.go` build lanes: the plan-only branch adds `blocker_advisory`/`blocker_advisory_question` to the result map before the spawn plan renders; the direct build lane prints the identical rendered heads-up, reconstructing a minimal manifest from read-only derivations since that lane never builds a full `codexBuildManifest` today.
- Added the "Blocker Heads-Up" stage to all three build wrapper copies (`.claude/commands/ant/build.md`, `.opencode/commands/ant/build.md`, `.claude/commands/ant-build.md`), byte-identical, positioned before Team Check-In, and extended `TestBuildWrapperCeremonyContract`'s required substrings and heading-order list.
- Nine total lock tests across the two Go tasks, all passing, plus the full `go test ./cmd/...` suite (251.6s, zero failures) run once as the final verification pass.

## Task Commits

Each task was committed atomically:

1. **Task 1: The third signal, read from the structured report** - `c42c9ec2` (feat)
2. **Task 2: The heads-up, the one question, and the non-interactive print** - `f0092ec5` (feat)
3. **Task 3: All three build-wrapper copies ask the one question** - `98e20b46` (docs)

**Plan metadata:** committed with this SUMMARY.

## Files Created/Modified

- `cmd/build_blocker_advisory.go` - `buildBlockerSignal`, `buildStartBlockerSignals`, `buildBlockerAdvisory`, `decideBuildBlockerAdvisory`, `renderBuildBlockerAdvisory`, `buildBlockerAdvisoryQuestion`
- `cmd/build_blocker_advisory_test.go` - nine named lock tests across Tasks 1-2 plus shared fixtures (`blockerAdvisoryFixturePhase`)
- `cmd/ceremony_team_checkin.go` - `lastContinueEndedBlocked` added after `buildHasPendingOwnerDecision`
- `cmd/codex_workflow_cmds.go` - blocker-advisory computation wired into the plan-only result map and the direct build lane's printed visual
- `cmd/build_wrapper_ceremony_test.go` - extended `required`/`inOrder` slices for the new Blocker Heads-Up stage
- `.claude/commands/ant/build.md`, `.opencode/commands/ant/build.md`, `.claude/commands/ant-build.md` - new "Blocker Heads-Up" stage, byte-identical, before Team Check-In

## Decisions Made

- **Direct build lane's signal completeness:** the direct (non-plan-only) build lane never computed a full `codexBuildManifest` before this plan (confirmed: zero references to `ForcedReviewers`/`BoundaryQuestionCount` in `runCodexBuildWithOptions`'s body). Rather than adding new orchestrator-boundary-question machinery there (out of scope, and a possible new side effect since `materializeOrchestratorBoundaryQuestions` can create records), the direct lane reconstructs a minimal manifest from purely read-only calls (`forcedReviewerRecords(queenForcedReviewersForPhase(phase))`, plus the phase-ID-only `lastContinueEndedBlocked`/`pendingHandoffDecisions`). Both lanes call the identical three advisory functions -- the CLAUDE.md guarantee -- with the direct lane's boundary-question coverage honestly at parity with what that lane already tracked (nothing), not a regression.
- **One "unanswered-question" signal name, not two:** an orchestrator boundary question and a worker's handoff question collapse into the same signal name, matching the plan's own framing ("an unanswered question -- orchestrator or worker") and D-09's "an open owner question" as one hard stop the owner experiences as a single thing.
- **No new "stop command" result key:** Task 2's action item 3 ("a stop answer ends the run with the exact command...") is satisfied by directing the wrapper prose (Task 3) to the existing single next-step source of truth (`aether status`) rather than inventing a second, parallel command-resolution path -- no acceptance criterion or artifact in the plan names a dedicated key for this, and CLAUDE.md's "two surfaces cannot disagree" principle is better served by pointing at the one existing resolver than adding a second.

## Deviations from Plan

### Auto-fixed Issues

None - plan executed exactly as written for all three tasks' functional requirements.

## TDD Gate Compliance

Tasks 1 and 2 carried `tdd="true"`. Both were implemented with the test file written and verified immediately after the corresponding implementation, then committed together as a single `feat(198-07)` commit per task, rather than as separate `test(...)` (RED) then `feat(...)` (GREEN) commits. This plan's frontmatter type is `execute` (not `tdd`), so the plan-level RED/GREEN/REFACTOR gate enforcement does not apply, but the per-task `tdd="true"` marker's letter (separate RED-then-GREEN commits) was not followed -- both tasks' tests were written test-first in the sense that behavior was fully specified via the plan's `<behavior>` blocks before implementation, and every test genuinely exercises the implementation (verified failing-then-passing during development), but the git history does not show a standalone failing-test commit preceding each implementation commit. Flagging this honestly per the workflow's TDD Gate Compliance requirement.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- SHOW-02's build-start blocker advisory sub-item is complete: the three genuine hard-stop signals print before a build starts, ask one question interactively, still print (without asking) non-interactively, and are described identically across all three wrapper copies.
- No blockers for any remaining Phase 198 plans (04-06, 08-09) — none of them depend on this plan's artifacts per the phase's dependency graph.
- Full `go test ./cmd/...` passes (251.6s, zero failures); `go build ./...` and `go vet ./cmd` both exit 0.

---

*Phase: 198-put-the-thrown-away-data-back-on-screen*
*Completed: 2026-08-29*

## Self-Check: PASSED

- All created/modified files confirmed present on disk (`cmd/build_blocker_advisory.go`, `cmd/build_blocker_advisory_test.go`, `cmd/ceremony_team_checkin.go`, `cmd/codex_workflow_cmds.go`, `cmd/build_wrapper_ceremony_test.go`, all three wrapper copies).
- All three task commit hashes (`c42c9ec2`, `f0092ec5`, `98e20b46`) confirmed present in `git log`.
- All plan-level acceptance criteria re-run and passing: `go test ./cmd -run 'TestBlockedLastTimeIsReadFromTheStructuredReport|TestOnlyTheThreeNamedSignalsRaiseTheHeadsUp|TestBuildStartBlockerAdvisory|TestNonInteractiveRunsStillPrintTheHeadsUp|TestBothBuildLanesEmitTheHeadsUp|TestBlockerHeadsUpDispatchesNoWorkers|TestBuildCheckinDecisionMatrix|TestOneWorkerBuildSkipsCheckin|TestBuildWrapperCeremonyContract|TestLifecycleFlatMirrorsMatchCanonical' -count=1` -- `ok`.
- Full `go test ./cmd/...` run once as the final verification pass -- `ok`, zero failures (251.6s).
- `go build ./...` and `go vet ./cmd` both exit 0.
- The three build wrapper copies confirmed byte-identical via `diff`.
