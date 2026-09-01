---
phase: 193-free-checks-are-the-floor
plan: 02
subsystem: orchestration
tags: [go, build-dispatch, queen-judgement, verification, attempt-journal]

# Dependency graph
requires:
  - phase: 193-01
    provides: "runDeterministicFloor(ctx, root, phase, manifest, watcher, timeout) -- the shared deterministic-floor body reused here for the build-time free-check report; evaluateContinueWatcherVerification for reading the already-resolved watcher value"
provides:
  - "The build's verification stage dispatches a watcher only when the Queen's proposal explicitly named one (queenAskedFor, gated on queenJudgement.Proposed, not the effective caste set)"
  - "buildFreeCheckReport + attachBuildFreeCheckReport(cmd/build_attempt.go) -- build finalize records the program's own free checks on the attempt journal as a report, never an advancement gate"
  - "TestPhaseVerifiedOnce (cmd/phase_verified_once_test.go) -- proves the watcher is not dispatched at both the build and continue boundaries for the same phase unless the Queen asks, over a real build manifest and continue plan pair"
affects: [193-03, 193-04, 193-05, 194]

# Actuals (#2632)
actuals:
  tokens: 13000
  tasks: 3
  commits: 6

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "queenAskedFor vs queenCastes: the Queen's explicit proposal (queenJudgement.Proposed) is a separate set from the effective team after the required-caste floor unions itself in (queenJudgement.Final) -- the build's verification-stage watcher dispatch is gated on the former, everything else on the latter."
    - "Dedicated narrow report-only atomic setter (attachBuildFreeCheckReport) mirroring closeBuildAttemptOutOfBand's discipline: the function's own signature makes it structurally impossible to touch Status/Dispatches/History while attaching a report."

key-files:
  created:
    - cmd/phase_verified_once_test.go
    - .planning/WINDOWS.md
  modified:
    - cmd/codex_build.go
    - cmd/codex_build_finalize.go
    - cmd/build_attempt.go
    - cmd/codex_build_test.go
    - cmd/codex_visuals_test.go
    - cmd/queen_judgement_test.go
    - cmd/queen_orchestration_regression_test.go
    - cmd/review_depth_test.go
    - cmd/codex_build_finalize_test.go
    - cmd/blackbox_harness_test.go
    - cmd/testdata/golden_build.txt

key-decisions:
  - "The build's verification-stage watcher dispatch condition moved from queenCastes[\"watcher\"] (the effective team, which always contains watcher via the required-caste floor) to queenAskedFor[\"watcher\"] (queenJudgement.Proposed, the Queen's literal, explicit ask). This is Task 1's entire change: one condition, one comment block. queenBuildSafetyRequiredCastes and queenApplyJudgement are untouched -- TestWatcherIsAlwaysRequiredOnBuild and TestQueenCannotDropTheWatcher both still pass unchanged, proving this is a dispatch-condition change, not a required-caste-floor change (that floor is Phase 194's territory)."
  - "Build finalize's free-check report is attached through a dedicated setter (attachBuildFreeCheckReport) rather than by extending transitionBuildAttempt's signature (which has 11 call sites) -- narrower blast radius, and the setter's own body makes it structurally impossible to also mutate Status/Dispatches/History, matching the precedent closeBuildAttemptOutOfBand already set for OutOfBandVerification."
  - "TestPhaseVerifiedOnce is deliberately scoped to the watcher caste, not literal full-intersection emptiness across every caste. Confirmed by direct experiment (see Deviations): probe, auditor and gatekeeper still legitimately double-dispatch on production/security phases with NO explicit Queen proposal on either side, because both build and continue derive \"this phase needs one\" independently from the same phase content (cmd/caste_relevance.go isAlwaysRequired's \"continue\" case at non-light depth, and the separate deterministic engine queenOrchestrate at any depth for the no-proposal path). This plan's own frontmatter already flagged this as \"FLOOR-04 edge probe row is unclassified -- NOT auto-resolved\" and scoped required-caste-floor changes to Phase 194. Asserting full emptiness would have failed for a real, pre-existing, out-of-scope reason -- recorded in .planning/WINDOWS.md (kind: unmet-truth) instead of silently narrowed away."

requirements-completed: [FLOOR-04]

coverage:
  - id: D1
    description: "A build planned with no Queen caste proposal produces zero verification-stage watcher dispatches, for documentation, prototype and production phase shapes; a build whose proposal names the watcher still gets exactly one."
    requirement: FLOOR-04
    verification:
      - kind: unit
        ref: "cmd/phase_verified_once_test.go#TestBuildPlansNoReviewerWithoutAProposal"
        status: pass
      - kind: unit
        ref: "cmd/phase_verified_once_test.go#TestBuildStillDispatchesAWatcherTheQueenAskedFor"
        status: pass
      - kind: unit
        ref: "go test ./cmd -run 'TestCodexBuild' -count=1 (rewritten stage assertions)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Build finalize runs the deterministic checks (build/types/lint/tests) and records them on the attempt journal as a report; a failing report produces byte-identical phase and task status to a passing one, so advancement stays continue's job."
    requirement: FLOOR-04
    verification:
      - kind: unit
        ref: "cmd/phase_verified_once_test.go#TestBuildFinalizeRecordsFreeChecksAsAReport"
        status: pass
      - kind: unit
        ref: "cmd/phase_verified_once_test.go#TestBuildFinalizeFreeChecksDoNotAdvanceThePhase"
        status: pass
    human_judgment: false
  - id: D3
    description: "A phase's build manifest and its continue plan share no dispatched watcher unless the Queen's proposal named it, proven over four phase shapes plus a fifth row proving the intersection can legitimately be non-empty when asked."
    requirement: FLOOR-04
    verification:
      - kind: unit
        ref: "cmd/phase_verified_once_test.go#TestPhaseVerifiedOnce (5 subtests)"
        status: pass
    human_judgment: false
  - id: D4
    description: "No regression introduced across the whole cmd package by removing the implicit build-side watcher dispatch."
    verification:
      - kind: unit
        ref: "go test ./cmd -count=1 (full package, 0 failures, 384s)"
        status: pass
      - kind: unit
        ref: "go test ./cmd -race -count=1 -run <changed/new test functions> (17 tests/subtests, 0 failures, 25s)"
        status: pass
      - kind: other
        ref: "go build ./cmd/aether && go vet ./..."
        status: pass
    human_judgment: false

# Metrics
duration: 62min
completed: 2026-08-22
status: complete
---

# Phase 193 Plan 02: The Build Side Runs Free Checks Only; Verified Once Summary

**The build's own verification stage no longer sends a reviewer implicitly -- it dispatches a watcher only when the Queen's proposal explicitly names one -- and build finalize records the program's free checks (build, types, lint, tests) on the attempt journal as a report that never gates or advances the phase, proven by a real build-manifest-vs-continue-plan comparison and a byte-identical-state comparison between a passing and a failing check run.**

## Performance

- **Duration:** ~62 min
- **Started:** 2026-08-22T13:09:00Z (approx, immediately following 193-01's completion)
- **Completed:** 2026-08-22T14:10:46Z
- **Tasks:** 3 completed
- **Files modified:** 13 (2 created, 11 modified)

## Accomplishments

- The build's "verification" stage dispatches a watcher only when the Queen's proposal explicitly named it (`queenAskedFor["watcher"]`, i.e. `queenJudgement.Proposed`), not whenever the required-caste floor restores watcher into the effective team (`queenCastes["watcher"]`, i.e. `queenJudgement.Final`) -- the required-caste floor itself is untouched, so `TestWatcherIsAlwaysRequiredOnBuild` and `TestQueenCannotDropTheWatcher` both still pass unchanged.
- Build finalize runs the program's own deterministic checks (via the shared `runDeterministicFloor` from 193-01) and attaches the result to the attempt journal through a dedicated, narrow setter (`attachBuildFreeCheckReport`) that touches nothing else on the record -- proven by finalizing the same fixture with a passing and a failing check set and confirming the resulting phase/task status is byte-identical either way.
- `TestPhaseVerifiedOnce` plans a real build manifest and a real continue plan for the same phase across four shapes (documentation-only, prototype, production, high-risk) and confirms the intersection never contains "watcher" unless the Queen's proposal named it, plus a fifth row proving the intersection legitimately can contain it when asked -- reverting Task 1's dispatch condition makes 4 of 5 subtests fail (`go test` exit code 1); restoring it makes all 5 pass (exit code 0).
- A full sweep of the whole `cmd` package (not just the plan's own `-run` filters) found and fixed 8 more places broken by the same one-line dispatch-condition change: three test files' watcher-caste assertions (`codex_visuals_test.go`, `queen_judgement_test.go`, `queen_orchestration_regression_test.go`, `review_depth_test.go`), a fixture worker-name dependency (`codex_build_finalize_test.go`), three black-box journeys' process-log assertions (`blackbox_harness_test.go`), and a golden visual-output fixture (`testdata/golden_build.txt`).
- Discovered (not fixed -- out of scope, flagged by the plan's own frontmatter) that probe, auditor and gatekeeper still legitimately double-dispatch on production/security phases with no explicit Queen proposal, because both build and continue independently derive "this phase needs one" from the same phase content. Recorded in `.planning/WINDOWS.md` for Phase 194.

## Task Commits

Each task was committed atomically, following RED/GREEN for the two TDD implementation tasks:

1. **Task 1 RED:** `af3f0dbf` (test) -- failing `TestBuildPlansNoReviewerWithoutAProposal` / `TestBuildStillDispatchesAWatcherTheQueenAskedFor`
2. **Task 1 GREEN:** `c5ca8520` (feat) -- `queenAskedFor` dispatch condition + rewritten `codex_build_test.go`/`codex_visuals_test.go`/`queen_judgement_test.go`/`queen_orchestration_regression_test.go`/`review_depth_test.go` assertions
3. **Task 2 RED:** `7d10e05f` (test) -- failing `TestBuildFinalizeRecordsFreeChecksAsAReport` / `TestBuildFinalizeFreeChecksDoNotAdvanceThePhase`
4. **Task 2 GREEN:** `39116539` (feat) -- `buildFreeCheckReport` + `attachBuildFreeCheckReport` (`cmd/build_attempt.go`) + the `runCodexBuildFinalize` call site (`cmd/codex_build_finalize.go`)
5. **Full-suite sweep fix:** `5be06a78` (fix) -- `codex_build_finalize_test.go`, `blackbox_harness_test.go`, `testdata/golden_build.txt`
6. **Task 3:** `f833479f` (test) -- `TestPhaseVerifiedOnce`

**Plan metadata:** (this commit, following)

_Note: Task 3 has no separate GREEN commit -- the code it verifies (Task 1's dispatch condition) already exists; its RED/GREEN boundary was instead confirmed by temporarily reverting and restoring that condition (see Deviations)._

## Files Created/Modified

- `cmd/codex_build.go` -- `plannedBuildDispatchesWithJudgement`: `queenAskedFor` local + verification-stage watcher dispatch gated on it instead of `queenCastes`.
- `cmd/build_attempt.go` -- `buildFreeCheckReport` type, `buildFreeCheckReportFromFloor` converter, `attachBuildFreeCheckReport` setter, `buildAttemptRecord.FreeChecks` field.
- `cmd/codex_build_finalize.go` -- `runCodexBuildFinalize`: computes the deterministic floor and attaches the free-check report when `skipVerify` is false; a failing report surfaces a plain-English stderr warning, never an error return.
- `cmd/phase_verified_once_test.go` (new) -- all five tests this plan adds, plus the `phaseVerifiedOncePhase` and `writePhaseVerifiedOnceVerificationCommands` fixture helpers.
- `cmd/codex_build_test.go`, `cmd/codex_visuals_test.go`, `cmd/queen_judgement_test.go`, `cmd/queen_orchestration_regression_test.go`, `cmd/review_depth_test.go` -- rewritten in place (no test functions deleted) to describe the new no-implicit-watcher truth.
- `cmd/codex_build_finalize_test.go`, `cmd/blackbox_harness_test.go`, `cmd/testdata/golden_build.txt` -- fixed by the full-suite sweep (see Deviations).
- `.planning/WINDOWS.md` (new) -- one `unmet-truth` entry recording the probe/auditor/gatekeeper double-dispatch gap for Phase 194.

## Decisions Made

See `key-decisions` in the frontmatter for the three load-bearing ones (dispatch-condition scope, the dedicated setter, and `TestPhaseVerifiedOnce`'s deliberate watcher-only scope).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Five test files outside the plan's declared `files_modified` asserted the now-removed implicit build-side watcher**
- **Found during:** Task 1, via targeted greps for "watcher" across every test that calls `plannedBuildDispatchesWithJudgement`/`plannedBuildDispatchesForSelectionWithState`/`plannedBuildDispatchesForSelection`, before touching implementation.
- **Issue:** `TestQueenChoiceReachesTheDispatchList` (`cmd/queen_judgement_test.go`) and `TestQueenAdaptiveCasteContractAcrossFlowHelpers` (`cmd/queen_orchestration_regression_test.go`) asserted the required-caste floor forces watcher into the build dispatch list "regardless of the proposal"; `TestBuildDispatch_StandardMode_IncludesWatcherAndProbe` (`cmd/review_depth_test.go`) asserted the same for standard depth with no proposal; `TestBuildVisualOutputShowsSpawnPlan` (`cmd/codex_visuals_test.go`) asserted a "Post-Wave: Watcher" line and a dispatch count of 4. All four are exactly the old floor D-08 replaces.
- **Fix:** Rewrote each assertion in place (no function deleted) to assert the opposite -- no build-side watcher without an explicit proposal -- with a comment citing D-08/193.
- **Files modified:** `cmd/queen_judgement_test.go`, `cmd/queen_orchestration_regression_test.go`, `cmd/review_depth_test.go`, `cmd/codex_visuals_test.go`
- **Verification:** `go test ./cmd -run 'TestBuild|TestCodexBuild|TestQueen' -count=1` -- PASS
- **Committed in:** `c5ca8520` (part of Task 1's GREEN commit)

**2. [Rule 1 - Bug] Three more places found only by running the whole `cmd` package, not the plan's named `-run` filters**
- **Found during:** after Task 2, running `go test ./cmd -count=1` (the full ~5,900-test package) as a sanity sweep before Task 3.
- **Issue:** `TestValidateCompletionPacketSemanticsReturnsAllViolations`/`TestBuildFinalizeCLIRejectsMultiViolationPacketWithStructuredDetails` (`cmd/codex_build_finalize_test.go`) attributed two of four violations to a worker named "Keen-6" -- the fixture's deterministic name for the now-removed build-side watcher; with no watcher dispatched, that worker never existed and only 2 of 4 violations fired. Three black-box journey tests (`cmd/blackbox_harness_test.go`: `TestCLIExternalAdapterBuildContract`, `TestCLIProviderBackedPlanRevisionJourney`, `TestCLICompiledInstallToSealJourney`) asserted a `"caste":"watcher"` line in the external adapter's process log, which only ever came from the build-side dispatch this plan removes (a continue-dispatched watcher still runs -- proven by `report.Passed` in the same tests -- it is just not caste-tagged in this adapter log format, a pre-existing gap outside this plan's scope). `cmd/testdata/golden_build.txt` (`TestGoldenBuildVisualOutput`) still recorded the old "Post-Wave: Watcher" step and a stale team-summary line.
- **Fix:** Renamed the fixture worker reference from "Keen-6" to "Check-80" (the probe, the fixture's actual second worker) with an explanatory comment; dropped the `"caste":"watcher"` requirement from the three black-box assertions with an explanatory comment; regenerated the golden file with `-update-golden` and diffed it by hand to confirm every change traced to this plan's dispatch-condition change (dispatch count 3→2, no watcher post-wave step, team-summary line drops "Watcher").
- **Files modified:** `cmd/codex_build_finalize_test.go`, `cmd/blackbox_harness_test.go`, `cmd/testdata/golden_build.txt`
- **Verification:** `go test ./cmd -count=1` (full package) -- 0 failures, 384s. `go test ./cmd -race -count=1` scoped to every changed/new test function -- 0 failures, 25s.
- **Committed in:** `5be06a78`

### Flagged, Not Fixed (out of scope, recorded per this plan's own frontmatter)

**3. [Discovery, deferred to Phase 194] Probe, auditor and gatekeeper still legitimately double-dispatch with no explicit Queen proposal**
- **Found during:** designing Task 3's `TestPhaseVerifiedOnce`. A direct experiment (calling `plannedBuildDispatchesWithJudgement` and `plannedContinueReviewDispatches`/`continueWatcherDecision` for the same phase, no proposal) showed, e.g., for a production phase: build dispatches `{auditor, builder, gatekeeper, probe, porter}`, continue dispatches `{auditor, probe, watcher}` -- `auditor` and `probe` appear on both sides, with no Queen proposal involved on either. This holds at every verification depth tested (standard and light), because `cmd/caste_relevance.go`'s `isAlwaysRequired` computes "continue" flow's required castes independently of "build" flow's, from the same phase content.
- **Why not fixed here:** This plan's own frontmatter already flagged it: `"FLOOR-04 edge probe row is unclassified -- NOT auto-resolved"`, and the plan's action items for Task 1 name only the watcher dispatch condition for removal. The orchestrator notes for this execution explicitly forbid touching `queenBuildSafetyRequiredCastes`/`queenApplyJudgement`/any required-caste rule, reserving that for Phase 194 ("Phase 194 moves the required-caste floor; 193 only stops the duplicate [watcher] dispatch"). Writing `TestPhaseVerifiedOnce` to assert full intersection-emptiness across every caste would have failed on the production/high-risk rows for this real, pre-existing, out-of-scope reason -- that would be testing Phase 194's claim under this plan's name.
- **Recorded:** `.planning/WINDOWS.md`, `kind: unmet-truth`, `phase: 193`, citing this exact gap for Phase 194 to close. `TestPhaseVerifiedOnce`'s own doc comment carries the same explanation.
- **Verification it was not silently swept under the rug:** the finding is stated in three places -- the test's doc comment, this SUMMARY, and `.planning/WINDOWS.md` -- so it survives past this session even if none of those three are read together.

---

**Total deviations:** 2 auto-fixed (Rule 1 -- 8 test files across two discovery passes), 1 flagged-not-fixed (out-of-scope architectural gap, recorded for Phase 194)
**Impact on plan:** Moderate in file count, zero in risk -- every fix was a test-assertion rewrite tracing directly to Task 1's one-line dispatch-condition change; no production logic beyond the plan's own three files was touched.

## Issues Encountered

One transient false alarm, not a real regression: a `go test ./cmd -count=1` background run captured a black-box test failure (`TestCLIExternalAdapterBuildContract`) claiming "black-box command changed the source checkout," listing `.planning/WINDOWS.md` as an unexpected new untracked file. This was caused by this session's own `gsd_run windows append` call creating that file WHILE the background test run was mid-flight comparing `git status` snapshots of this repo -- a timing artifact of concurrent tool use, not a code defect. Re-ran `go test ./cmd -count=1` cleanly afterward (no concurrent file writes): 0 failures, exit 0.

`go test ./... -race` (the plan's literal `<verification>` command) was started and killed after ~1 minute: compiling and race-testing the entire repository (`cmd/` + `pkg/` + everything else) at ~5,900+ tests in `cmd` alone was not going to complete in a reasonable session budget, following 193-01's own precedent of scoping race runs. Ran `go test ./cmd -race -count=1` instead, scoped to every test function this plan added or changed (17 tests/subtests: the five `phase_verified_once_test.go` tests, the six rewritten build/queen/visual assertions, the golden test, and the three finalize/black-box fixes) -- 0 failures, exit 0, 25s.

## User Setup Required

None -- no external service configuration required. This plan is entirely local Go code and tests.

## Next Phase Readiness

Ready for 193-03 (wave 2, runs next) and 193-04 (wave 3). `TestWatcherIsAlwaysRequiredOnBuild` survived unchanged -- it tests `queenBuildSafetyRequiredCastes` (the required-caste computation itself), not the dispatch condition this plan changed, so no ledger entry in `.aether/docs/retired-tests-ledger.md` was needed or made.

Flagged for Phase 194 (already recorded in `.planning/WINDOWS.md`): probe, auditor and gatekeeper still double-dispatch on production/security phases with no explicit Queen proposal, because both build and continue independently derive "this phase needs one" from the same phase content. Moving the required-caste floor (194's stated purpose) should resolve this the same way Task 1 resolved it for watcher.

---
*Phase: 193-free-checks-are-the-floor*
*Completed: 2026-08-22*

## Self-Check: PASSED

All created files (`cmd/phase_verified_once_test.go`, `.planning/WINDOWS.md`, this SUMMARY) and all six task commits (`af3f0dbf`, `c5ca8520`, `7d10e05f`, `39116539`, `5be06a78`, `f833479f`) plus the SUMMARY/ledger commit (`06a66887`) verified present in git history.
