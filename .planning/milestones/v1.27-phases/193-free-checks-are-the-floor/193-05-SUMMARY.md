---
phase: 193-free-checks-are-the-floor
plan: 05
subsystem: verification
tags: [go, continue-pipeline, verification-scope, attempt-journal, self-healing]

# Dependency graph
requires:
  - phase: 193-01
    provides: "runDeterministicFloor(ctx, root, phase, manifest, watcher, timeout) -- the shared deterministic-floor body this plan scopes and extends with one bounded fix attempt"
  - phase: 193-04
    provides: "isLastPhaseOfActivePlan(phaseID) -- reused directly for D-07's final-phase-always-full rule, no new state-loading code needed"
provides:
  - "deriveVerificationScope(root, phase, isFinalPhase, claims, commands) -- scopes the tests command to the Go packages a phase's changed files touched, falling back to the full run whenever that scope cannot be honestly derived, or to mode \"none\" when no tests command resolves at all"
  - "buildCheckFailureIndex(step, claims, phase) checkFailureIndex -- a bounded, deduplicated failure summary (never the whole command log) naming the implicated task(s)"
  - "planCheckFixAttempt / applyAutomaticCheckFixAttempt -- the single bounded automatic builder fix attempt (D-02): dispatches exactly one builder when a free check fails and no reviewer was sent, records it as a new append-only attempt-journal entry, and re-runs the floor once"
  - "check_fix_attempt gate (cmd/codex_continue.go) -- names the failing check and the one exact re-run command when the fix attempt did not resolve it"
affects: [194, 195, 196, 197, 198]

# Actuals (#2632)
actuals:
  tokens: 15610
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Scope-then-run: deriveVerificationScope rewrites only codexVerificationCommands.Test before the existing four-step runVerificationStep loop runs -- the loop shape is untouched, only what commands.Test resolves to changes."
    - "Provider-availability auto-skip reused for a new caller: applyAutomaticCheckFixAttempt gates its real builder dispatch on the exact same \"FakeInvoker or IsAvailable\" check continueWatcherDecision already uses for the reviewer path, so every pre-existing test that never stubs a worker provider is unaffected by the new code path."
    - "New append-only attempt record, not a mutated one: the fix attempt goes through beginBuildAttempt/transitionBuildAttempt like any other attempt, producing a second attempt file with its own CheckFix field -- the original attempt's Dispatches/Claims/Status are never touched."

key-files:
  created:
    - cmd/verification_scope.go
    - cmd/verification_scope_test.go
    - cmd/check_fix_attempt.go
    - cmd/floor_fix_attempt_test.go
  modified:
    - cmd/deterministic_floor.go
    - cmd/build_attempt.go
    - cmd/codex_continue.go

key-decisions:
  - "Only the tests command is ever scoped (D-07 explicit boundary) -- build, types and lint keep running exactly as configured in every mode, because narrowing a compiler or linter changes what it can see."
  - "Scoping only understands the conventional `go test ./...` pattern; a hand-authored test command naming something else is left alone and reported as full (no scoped runner), rather than guessed at -- an honest fallback over a risky rewrite."
  - "isLastPhaseOfActivePlan (193-04) is called directly inside runDeterministicFloor rather than threading a new isFinalPhase parameter through both continue lanes -- keeps the shared floor's signature stable and avoids touching cmd/codex_continue_plan.go (the wrapper lane), which this plan's files_modified never named."
  - "planCheckFixAttempt only fires on a failing shell verification step (build/types/lint/tests); a claims-only or criteria-only failure with every shell step green draws no fix attempt -- there is no command a builder run could plausibly repair in that case, and D-02's language (\"fixing the failed tests check\") is about a check, not a criterion."
  - "The fix-attempt journal entry is a brand-new build attempt (via beginBuildAttempt), not an extension of the original attempt's record -- append-only by construction, the same discipline outOfBandVerificationRecord already established for a differently-caused non-worker closure."
  - "The compact failure index matches file references by full normalized path OR bare filename -- Go's own test failure lines report only the bare filename (\"foo_test.go:42:\"), not the repository-relative path, so filename-only matching is required for ImplicatedTaskIDs to work against real go test output, not just hand-built fixtures."

patterns-established:
  - "checkFailureIndex as the shape any future bounded-fix-attempt caller reuses if the pattern extends beyond the tests check."

requirements-completed: [FLOOR-01, FLOOR-02]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "Verification runs the tests command scoped to the Go packages a phase's changed files touched, and always falls back to the full run on the plan's final phase, wired through runDeterministicFloor (D-07)."
    requirement: FLOOR-01
    verification:
      - kind: unit
        ref: "cmd/verification_scope_test.go#TestScopeIsDerivedFromChangedFiles"
        status: pass
      - kind: unit
        ref: "cmd/verification_scope_test.go#TestFinalPhaseAlwaysRunsFull"
        status: pass
    human_judgment: false
  - id: D2
    description: "Scoping falls back to the full, unchanged command whenever a scope cannot be honestly derived: no derivable package, an empty claims set, or an ecosystem with no scoped runner."
    requirement: FLOOR-01
    verification:
      - kind: unit
        ref: "cmd/verification_scope_test.go#TestScopeFallsBackToFullWhenUndecidable"
        status: pass
    human_judgment: false
  - id: D3
    description: "A targeted run's package set is structurally a subset of the full run's coverage -- proven as an invariant, not a fixed-string comparison."
    verification:
      - kind: unit
        ref: "cmd/verification_scope_test.go#TestScopedCommandNeverBroadensTheRun"
        status: pass
    human_judgment: false
  - id: D4
    description: "A timed-out verification step reads as failed (not skipped), carries the timeout ErrorClass, and contributes a blocking issue naming the check and the timeout duration; a step with no resolved command stays skipped and is not eligible for the fix path."
    requirement: FLOOR-02
    verification:
      - kind: unit
        ref: "cmd/floor_fix_attempt_test.go#TestTimedOutCheckIsAFailedCheck"
        status: pass
    human_judgment: false
  - id: D5
    description: "A failing check's output becomes a bounded, deduplicated index (a small proportion of the raw output, never the whole log or its tail) naming the implicated task."
    verification:
      - kind: unit
        ref: "cmd/floor_fix_attempt_test.go#TestFailureIndexIsCompactAndNamesTheImplicatedTasks"
        status: pass
      - kind: unit
        ref: "cmd/floor_fix_attempt_test.go#TestFailureIndexNeverCarriesTheWholeLog"
        status: pass
    human_judgment: false
  - id: D6
    description: "A free check failing with no reviewer dispatched draws exactly one automatic builder fix attempt naming the failing check, with no reviewer dispatch of any kind."
    requirement: FLOOR-02
    verification:
      - kind: unit
        ref: "cmd/floor_fix_attempt_test.go#TestFailedCheckSendsExactlyOneBuilderFixAttempt"
        status: pass
    human_judgment: false
  - id: D7
    description: "If the re-run still fails, continue blocks and the recovery plan / check_fix_attempt gate carry exactly one command to re-run the builder by hand."
    requirement: FLOOR-02
    verification:
      - kind: unit
        ref: "cmd/floor_fix_attempt_test.go#TestSecondFailureBlocksAndNamesTheCommand"
        status: pass
    human_judgment: false
  - id: D8
    description: "The fix attempt is a new, append-only attempt-journal entry: the original attempt's dispatches, claims and status are byte-identical before and after, and the journal holds both as separate records."
    requirement: FLOOR-02
    verification:
      - kind: unit
        ref: "cmd/floor_fix_attempt_test.go#TestFixAttemptNeverOverwritesTheFirstResult"
        status: pass
    human_judgment: false
  - id: D9
    description: "A fix attempt already recorded for a phase and check blocks a second automatic attempt on a later continue run."
    requirement: FLOOR-02
    verification:
      - kind: unit
        ref: "cmd/floor_fix_attempt_test.go#TestNoSecondAutomaticFixAttempt"
        status: pass
    human_judgment: false
  - id: D10
    description: "The fix attempt's own dispatch count is recorded on its own separate journal entry, distinguishable from the original build's workers -- the data the team card and cost line (Phase 196) will read."
    requirement: FLOOR-02
    verification:
      - kind: unit
        ref: "cmd/floor_fix_attempt_test.go#TestFixAttemptIsCountedSeparately"
        status: pass
    human_judgment: false
  - id: D11
    description: "No regression introduced across the whole cmd package by scoping verification and adding the bounded fix attempt; every earlier plan's own named regression tests still hold."
    verification:
      - kind: unit
        ref: "go test ./cmd -count=1 (full package, 0 failures, 390.9s)"
        status: pass
      - kind: unit
        ref: "go test ./cmd -race -count=1 -run <12 tests this plan added> (0 failures, 4.4s)"
        status: pass
      - kind: unit
        ref: "go test ./cmd -run 'TestDeterministicChecksCannotBeSkipped$|TestPhaseVerifiedOnce$|TestGateAcceptsDeterministicEvidenceWithoutReviewer$|TestZeroReviewerPhaseAdvancesWhenFreeChecksPass$' -count=1"
        status: pass
      - kind: other
        ref: "go build ./... && go vet ./..."
        status: pass
    human_judgment: false

# Metrics
duration: 43min
completed: 2026-08-22
status: complete
---

# Phase 193 Plan 05: Targeted Verification and the One Bounded Fix Attempt Summary

**Verification now runs the tests command scoped to the Go packages a phase's changed files actually touched (falling back to the full suite whenever that scope cannot be honestly derived, or on the plan's final phase), and a failing free check with no reviewer dispatched draws exactly one automatic builder fix attempt -- recorded as its own append-only attempt-journal entry -- before continue blocks and names the single exact command to redispatch by hand.**

## Performance

- **Duration:** ~43 min
- **Started:** 2026-08-22T15:52:00Z (approx, immediately following 193-04's completion)
- **Completed:** 2026-08-22T16:35:00Z (approx)
- **Tasks:** 3 completed
- **Files modified:** 7 (4 created, 3 modified)

## Accomplishments

- `deriveVerificationScope` (D-07) rewrites the tests command to cover only the Go package directories a phase's claimed changed files touched, and falls back to the full, unchanged command whenever that scope cannot be honestly derived: no changed files, files with no derivable package (docs, config), an ecosystem with no scoped runner, or the plan's final phase, which always runs everything as the last honest check before the plan closes. Only the tests command is ever scoped -- build, types and lint keep running exactly as configured, since narrowing a compiler or linter changes what it can see.
- The subset property is proven as an invariant, not a string comparison: every targeted package pattern is structurally a recursive restriction (`./dir/...`) of the full run's own `./...` pattern.
- A timed-out verification step already read as failed (not skipped), carried the timeout `ErrorClass`, and contributed a blocking issue naming the check and the duration in the pre-existing code from 193-01 -- this plan adds the regression tests locking that distinction in and distinguishing it from a step with no resolved command, which stays skipped and ineligible for the fix path.
- `buildCheckFailureIndex` turns a failing check's raw output into a small, bounded index: the first few deduplicated failure lines (capped in count and per-line length), never the whole log or its tail, plus the task(s) whose reported changed files the failure implicates (matched by full path or bare filename, since Go's own test failures report only the filename).
- When a free check fails and no reviewer was dispatched, `applyAutomaticCheckFixAttempt` sends exactly one builder carrying the compact failure index, records the attempt as a brand-new, append-only entry in the build attempt journal (never mutating the original attempt's dispatches, claims or status), and re-runs the floor once. A fix attempt already recorded for a phase and check blocks any further automatic attempt. If the re-run still fails, continue blocks: the recovery plan and the new `check_fix_attempt` gate both carry the single exact command to redispatch the builder by hand.
- The real builder dispatch is gated on the same worker-provider-availability check `continueWatcherDecision` already uses for the reviewer path (`FakeInvoker` or `invoker.IsAvailable`), so every pre-existing test across the package that never stubs a worker provider draws no fix attempt and is unaffected by the new code path -- confirmed by the full, unmodified `go test ./cmd` sweep.

## Task Commits

Each task was committed atomically:

1. **Task 1: Targeted per phase, full at the end** - `b3625a18` (feat)
2. **Task 2: A timed-out check is a failed check, and a failure becomes a compact index** - `587ee81c` (feat)
3. **Task 3: Exactly one automatic builder fix attempt, recorded append-only** - `c3504c17` (feat)

**Plan metadata:** (this commit, following)

_Note: all three tasks were marked `tdd="true"`. RED was confirmed for every named test by temporarily stubbing the relevant function to a no-op/naive implementation, re-running the exact test set to observe genuine failures, then restoring the real implementation and re-running to green -- see "TDD Discipline" below. This RED/restore cycle was not preserved as a separate git commit; each task landed as one GREEN commit, matching the precedent set by 193-04's SUMMARY for the same reason (the test file and its production code were designed and iterated together before either was committed)._

## Files Created/Modified

- `cmd/verification_scope.go` (new) -- `verificationScope` type, `deriveVerificationScope`, `loadRawBuildClaimsForScope`, `changedFilesFromBuildClaims`, `goPackagePathsForChangedFiles`, `scopedGoTestCommand`.
- `cmd/verification_scope_test.go` (new) -- the four named Task 1 tests plus the `wired through runDeterministicFloor` integration subtest.
- `cmd/deterministic_floor.go` -- `runDeterministicFloor` now calls `deriveVerificationScope` before the four `runVerificationStep` calls and carries the resulting `verificationScope` onto `deterministicFloorResult.Scope`.
- `cmd/check_fix_attempt.go` (new) -- `checkFailureIndex`, `buildCheckFailureIndex`, `compactFailureExcerpts`, `implicatedTaskIDsFromExcerpts` (Task 2); `maxAutomaticCheckFixAttempts`, `planCheckFixAttempt`, `firstFailingVerificationStep`, `countCheckFixAttempts`, `applyAutomaticCheckFixAttempt`, `plannedCheckFixBuilderDispatch`, `renderCheckFixAttemptBrief` (Task 3).
- `cmd/build_attempt.go` -- `checkFixAttemptRecord` type, `buildAttemptRecord.CheckFix` field, `attachCheckFixAttempt` (a narrow setter mirroring `attachBuildFreeCheckReport`'s discipline), `listBuildAttemptsForPhase` (the append-only-journal scan `planCheckFixAttempt`'s dedup guard needs).
- `cmd/codex_continue.go` -- `codexContinueVerificationReport.CheckFixAttempt`, `codexContinueRecoveryPlan.CheckFixCommand`; `runCodexContinueVerification` wires `applyAutomaticCheckFixAttempt` in between resolving the reviewer decision and computing `checksPassed`; `assessCodexContinue` populates `Recovery.CheckFixCommand`; `runCodexContinueGates` gains the `check_fix_attempt` gate.
- `cmd/floor_fix_attempt_test.go` (new) -- all eight named Task 2/Task 3 tests plus fixture helpers.

## Decisions Made

See `key-decisions` in the frontmatter for the six load-bearing ones (scope boundary, conservative pattern matching, reusing `isLastPhaseOfActivePlan` internally, the shell-step-only fix-attempt trigger, the brand-new-attempt append-only design, and filename-fallback matching for implicated tasks).

## TDD Discipline

Every test this plan adds was confirmed RED before the corresponding implementation existed, using a temporary-stub-then-restore cycle rather than separate `test(193-05)`/`feat(193-05)` commits (each task landed as one GREEN commit, its test file and production code designed together):

- **Task 1:** stubbed `scopedGoTestCommand` to always return `("", false)` -- `TestScopeIsDerivedFromChangedFiles` and `TestScopedCommandNeverBroadensTheRun` genuinely failed (every case fell back to full instead of targeting), confirming the tests exercise real scoping logic, not a tautology.
- **Task 2:** stubbed `compactFailureExcerpts` to return the raw output unbounded -- `TestFailureIndexIsCompactAndNamesTheImplicatedTasks` and `TestFailureIndexNeverCarriesTheWholeLog` genuinely failed (`index=33449 bytes, output=33450 bytes`, and the index literally contained the raw output's tail), confirming the bound and the never-carries-the-tail invariant are real, not vacuous.
- **Task 3:** stubbed `planCheckFixAttempt` to always return `(checkFixAttemptRecord{}, false)` -- all five named tests genuinely failed ("expected a check fix attempt to have run" / "expected the first run to draw a fix attempt"), confirming the fix-attempt wiring is load-bearing, not dead code the tests happened to pass around.

## Deviations from Plan

None -- plan executed exactly as written. Task 2's action item 1 ("make a timed-out step unambiguous") required no implementation change: 193-01's `runVerificationStep` already set `Passed: false`, `Skipped: false`, and `ErrorClass: ErrorClassTimeout` correctly for a timed-out step, and `failureSummaryForStep` already named the timeout duration in the resulting blocking-issue text. This plan added the regression tests (`TestTimedOutCheckIsAFailedCheck`) locking that pre-existing correctness in, following the same "no implementation change needed, tests kept as regression guards" precedent 193-03's SUMMARY recorded for an analogous finding.

## Issues Encountered

None that affected the outcome. One environment detail worth recording: `writeAgentsVerificationCommands`'s markdown-command parser (`looksLikeVerificationCommand`, `cmd/codex_continue.go`) only accepts a bare (non-backtick-wrapped) command whose first word is a small fixed whitelist (`printf`, `echo`, `true`, `false`, `make`, `sh`, `bash`) or a recognized ecosystem pattern (`go build`, `go test`, etc.) -- a fixture command of `sleep 2` is silently rejected and resolves to no command at all. `TestTimedOutCheckIsAFailedCheck`'s integration subtest uses `sh -c 'sleep 2'` instead (fields[0] == "sh" is whitelisted), which the shell executes identically since `runShellCommandContext` already wraps every configured command in `sh -c` itself.

The full, unmodified `go test ./cmd -count=1` sweep (390.886s, matching the ~384-392s baseline 193-01/193-02/193-04 recorded) reported zero failures across the whole ~5,900-test package, confirming the new automatic-fix-attempt code path is correctly gated off for every pre-existing test that does not stub a worker provider.

## User Setup Required

None -- no external service configuration required. This plan is entirely local Go code and tests.

## Next Phase Readiness

FLOOR-01 and FLOOR-02 now hold with runnable commands proving them (see `coverage` above), completing both requirements first declared by 193-01. This is the last plan of Phase 193 -- all four requirements this phase declares (FLOOR-01 through FLOOR-04) are now satisfied and covered by tests, closing the phase's stated boundary: the program's own deterministic checks (build, types, lint, tests, claimed-files-exist, criterion evidence) are the verification floor on every phase, can never be skipped, and a phase with zero reviewer workers advances when they pass and is blocked -- with one bounded, honest recovery path -- when they fail.

Flagged for Phase 194 (already recorded in `.planning/WINDOWS.md` by 193-02, unchanged by this plan): probe, auditor and gatekeeper still legitimately double-dispatch on production/security phases with no explicit Queen proposal, because build and continue independently derive the same required caste from the same phase content. This plan's scoping and fix-attempt work does not touch that required-caste floor.

---
*Phase: 193-free-checks-are-the-floor*
*Completed: 2026-08-22*

## Self-Check: PASSED

Recorded by the execute-phase orchestrator after the plan returned without this section (every other plan in the phase carried one):

- Key files present on disk: `cmd/verification_scope.go`, `cmd/check_fix_attempt.go`, `cmd/verification_scope_test.go`, `cmd/floor_fix_attempt_test.go` — all four found.
- Commits: `git log --oneline --grep="193-05"` returns 5 commits (`b3625a18`, `587ee81c`, `c3504c17`, `911f91a3`, `4e5c9adb`).
- Post-wave gate on the final tree (2026-08-22): `go build ./...` exit 0, `go vet ./...` exit 0, `go test ./...` exit 0 — 18 packages ok, `cmd` in 387.9s.
- Working tree clean apart from the untracked, gitignored `.gsd/` sentinel and `.planning/milestone.lock`.
