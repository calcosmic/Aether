---
phase: 194-the-queen-decides-the-team
plan: 06
subsystem: orchestration
tags: [queen, continue, changed-files, forced-reviewer, two-lane-parity, D-02, D-13]

# Dependency graph
requires:
  - phase: 194-01
    provides: "queenRiskSignalTable / riskSignal.PathPatterns (declared empty) / queenForcedContinueReviewers (changedFiles parameter accepted, ignored) -- this plan fills both"
  - phase: 194-03
    provides: "the per-worker reason / RefusedNoReason machinery this plan proves reaches the fast continue path, not just the plan-only path"
  - phase: 194-05
    provides: "queenFallbackTeam (the retuned no-proposal team) and the D-13 depth table this plan's autopilot and heavy-panel tests assert against"
provides:
  - "queenRiskSignalHitsFromPaths: the file-detected half of D-02 -- matches the builder's own reported changed_files against the five-signal table's now-filled PathPatterns"
  - "queenForcedContinueReviewers unions plan-wording hits (recorded or re-derived) with changed-file hits through the SAME collapseToForcedReviewers merge -- one merge function, add-only by construction"
  - "Both continue boundaries (plannedContinueReviewDispatches, plannedExternalContinueDispatches) supply phaseChangedFilesFromHandoffs(phase.ID) into the shared queenContinueReviewSpecsWithJudgement/queenContinueDispatchesWithJudgement chain -- the two lanes can no longer derive a different forced-reviewer set for the same phase"
  - "Proof that the owner's --castes proposal, per-worker reasons, an explicit --heavy request, and autopilot's no-proposal fallback all behave identically on the default (fast) continue path, not only the heavy plan-only path"
  - "Proof that the wrapper lane's planned-subset check already holds with a changed-files-forced reviewer, with no correction needed to the check or the planned-set recording"
affects: [194-07-waive-choice, 194-08-close-windows-1, 194-09-claudemd-rewrite, 196-cost-line]

# Actuals (#2632)
actuals:
  tokens: 10088
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A second, independent detector (file paths) unions into the same collapse function a first detector (plan wording) already used, rather than a parallel merge path that could disagree -- riskSignalHitsFromRecords reconstructs riskSignalHit values from the durable record precisely so both sources feed collapseToForcedReviewers identically"
    - "A structural relay (the changed-files union) is computed once, at the SAME shared function both continue lanes call, rather than twice at each lane's own boundary -- the boundary's job is only to supply the input (phaseChangedFilesFromHandoffs), never to re-derive the union itself"
    - "A nil global (store) reached by a newly-unconditional call path is a call-site bug waiting to happen the moment a lightweight test exercises that path without setting one up -- guard the shared read function once, not every caller"

key-files:
  created: []
  modified:
    - cmd/queen_risk_signals.go
    - cmd/codex_continue.go
    - cmd/codex_continue_plan.go
    - cmd/codex_dispatch_contract.go
    - cmd/queen_forced_reviewer_test.go
    - cmd/continue_fastpath_castes_test.go
    - cmd/owner_dials_test.go
    - cmd/claudemd_verification_depth_test.go
    - cmd/floor_unskippable_test.go
    - cmd/queen_judgement_test.go
    - cmd/queen_orchestration_regression_test.go

key-decisions:
  - "PathPatterns are lowercased path-segment substrings, matched with the same 'longest match wins' discipline queenRiskSignalHits already used for phrases -- one shared style across both detectors rather than a second matching algorithm."
  - "release sign-off gets no PathPatterns at all (nil, by construction of the signal table entry), exactly as the plan specified: no file path reliably means a release gate, and a pattern firing on the literal word 'release' appearing anywhere in a path would be a false alarm with no upside."
  - "forcedReviewerReason's clause construction is now source-aware (forcedReviewerReasonClause): a plan-wording hit reads 'the plan mentions \"...\"', a changed-file hit reads 'the files changed touched \"...\"' -- two clauses, not two sentence templates, so a caste forced by both a wording hit and a file hit in the same run states both without duplicating the caste's plain-English name."
  - "riskSignalHitsFromRecords reconstructs full riskSignalHit values (including the signal's PlainEnglish) from the durable codexForcedReviewerRecord by name lookup against queenRiskSignalTable, rather than converting records to forcedReviewer directly as the old code did -- this lets the recorded set and the newly-detected changed-file hits merge through the exact same collapseToForcedReviewers path the original build-time derivation used, instead of two independent merge implementations that could silently diverge."
  - "Chose to move TestForcedReviewerAddedByChangedFilesSatisfiesThePlannedSubsetCheck into cmd/owner_dials_test.go (task 2's own file) rather than leaving it in cmd/queen_forced_reviewer_test.go (task 1's file) purely so the two tasks' commits split cleanly by file -- the test itself proves a task 2 acceptance criterion (the wrapper lane's subset check) using task 1's machinery (the changed-files detector)."
  - "[Rule 3 - Blocking] loadWorkerHandoffRecords now returns (nil, nil) when the global store is nil, instead of panicking. phaseChangedFilesFromHandoffs became unconditional on every continue dispatch construction call as part of this plan's own wiring; several existing tests (TestPhaseVerifiedOnce, TestContinueFastPathHonoursCasteProposal) never set up a store because they never needed file I/O before. Guarding the one shared read function is simpler and safer than auditing every caller for a store setup it does not otherwise need."

patterns-established:
  - "queen_risk_signals.go remains the single file owning the entire named-risk vocabulary, both its matchers (phrase-based and path-based), the collapse rule, and the cross-boundary record conversion in both directions (forcedReviewerRecords for build->manifest, riskSignalHitsFromRecords for manifest->union) -- a future third detector has one file to extend, not several."

requirements-completed: [TEAM-02, TEAM-05]

coverage:
  - id: D1
    description: "What the builder actually touched can raise a reviewer the plan's wording missed, and can never lower one: the changed-files detector is add-only."
    requirement: TEAM-02
    verification:
      - kind: unit
        ref: "cmd/queen_forced_reviewer_test.go#TestChangedFilesCanOnlyAddAForcedReviewer"
        status: pass
    human_judgment: false
  - id: D2
    description: "Both continue lanes -- the default in-process lane and the wrapper plan-only lane -- force the same reviewer set for the same phase, the same recorded set and the same changed files."
    requirement: TEAM-02
    verification:
      - kind: unit
        ref: "cmd/queen_forced_reviewer_test.go#TestBothContinueLanesForceTheSameReviewers"
        status: pass
    human_judgment: false
  - id: D3
    description: "--castes and the per-worker reason flag take effect on the default (fast) continue path as well as the heavy plan-only path; a proposal with no reason is refused by name on the fast path too."
    requirement: TEAM-05
    verification:
      - kind: unit
        ref: "cmd/continue_fastpath_castes_test.go#TestContinueFastPathHonoursCasteProposal"
        status: pass
    human_judgment: false
  - id: D4
    description: "An explicit heavy request produces the full review panel (security + quality + coverage-where-testable) on both continue lanes."
    requirement: TEAM-05
    verification:
      - kind: unit
        ref: "cmd/owner_dials_test.go#TestHeavyGivesTheFullReviewPanel"
        status: pass
    human_judgment: false
  - id: D5
    description: "Autopilot, which passes no proposal, reaches the retuned fallback team (queenFallbackTeam) rather than the old scored team (queenCandidateDispatches)."
    verification:
      - kind: unit
        ref: "cmd/owner_dials_test.go#TestAutopilotPathSendsTheFallbackTeam"
        status: pass
    human_judgment: false
  - id: D6
    description: "A forced reviewer added at continue time by the changed-files detector still satisfies the wrapper lane's planned-subset check, because the planned set already records it -- not because the subset check was weakened."
    verification:
      - kind: unit
        ref: "cmd/owner_dials_test.go#TestForcedReviewerAddedByChangedFilesSatisfiesThePlannedSubsetCheck"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-08-23
status: complete
---

# Phase 194 Plan 6: Both Checking-Step Lanes Carry the Same Team Summary

**A second forced-reviewer detector reads the builder's own reported changed files (not just the plan's wording) and can only ADD a reviewer, wired identically into both continue lanes; the owner's `--castes`, per-worker reasons, `--heavy`, and autopilot's no-proposal fallback are all proven to behave the same on the default continue path as on the heavy plan-only path.**

## Performance

- **Duration:** ~55 min
- **Tasks:** 2 completed
- **Files modified:** 11 (0 new)

## Accomplishments

- `cmd/queen_risk_signals.go` — `PathPatterns` filled in on four of the five `queenRiskSignalTable` entries (release sign-off deliberately gets none) and `queenRiskSignalHitsFromPaths(paths []string) []riskSignalHit` implemented: matches the builder's OWN reported changed files against those patterns, "longest match wins" per signal, source tagged `"changed files"`.
- `queenForcedContinueReviewers` now unions the plan-wording hits (recorded from the build manifest, or re-derived when no manifest exists) with the changed-file hits through the SAME `collapseToForcedReviewers` merge the original derivation used — `riskSignalHitsFromRecords` reconstructs full `riskSignalHit` values from the durable record so both sources feed one merge function, never two that could disagree. Proven add-only by `TestChangedFilesCanOnlyAddAForcedReviewer`: a changed file matching no pattern leaves the recorded set byte-identical; a `migrations/...sql` file adds `auditor` while the recorded `gatekeeper` and its reason survive unchanged.
- Both continue boundaries — `plannedContinueReviewDispatches` (in-process lane, `cmd/codex_continue.go`) and `plannedExternalContinueDispatches` (wrapper lane, `cmd/codex_continue_plan.go`) — now supply `phaseChangedFilesFromHandoffs(phase.ID)` into the shared `queenContinueReviewSpecsWithJudgement`/`queenContinueDispatchesWithJudgement`/`unionForcedContinueReviewers` chain (threaded as a new `changedFiles []string` parameter, along with every call site the signature change touched). `TestBothContinueLanesForceTheSameReviewers` proves the two lanes return the identical caste set for the same phase, the same recorded set, and the same changed files.
- `TestContinueFastPathHonoursCasteProposal` extended: the same proposal, with no per-worker reason, is refused by name on the fast continue path (D-08) — closing the same class of asymmetry the test was originally written to catch, this time for the reason/refusal machinery rather than caste selection.
- `TestHeavyGivesTheFullReviewPanel` (new, `cmd/owner_dials_test.go`): an explicit heavy request with no proposal produces `gatekeeper` + `auditor` + `probe` on both continue lanes.
- `TestAutopilotPathSendsTheFallbackTeam` (new): proves the exact function autopilot's CLI path reaches (`queenOrchestrate` → `queenFallbackTeam`) does not carry a caste the old scored engine (`queenCandidateDispatches`) would have selected for the same wording — autopilot.go itself holds no dispatch call, so this asserts the function its unproposed `aether build`/`aether continue` invocations actually reach, per the task's own fallback instruction.
- `TestForcedReviewerAddedByChangedFilesSatisfiesThePlannedSubsetCheck` (new): proves `continueReviewCastesArePlannedSubset` already holds once a changed-files-forced reviewer is added — the file detector fires *inside* `plannedExternalContinueDispatches`, the exact call that produces the persisted "planned" manifest, so the forced reviewer is already part of "planned" before any worker dispatches. No correction to the planned-set recording or the subset check was needed.
- Audited every remaining nil-proposal call site: `queenContinueDispatches` and `queenContinueReviewSpecs` (the bare, no-`WithJudgement` wrappers) are called from exactly one place in the whole repo — a single test (`queen_judgement_test.go:326`, `TestContinueWithNoProposalIsUnchanged`, which deliberately compares the wrapper's output against an explicit nil-proposal call) — and nowhere in production code. No call site had the options struct in scope and was left passing nil.

## Task Commits

1. **Task 1: The changed-files detector adds a reviewer at the checking step, and can never remove one** - `f0c582bf` (feat)
2. **Task 2: The owner's caste list and reasons work on the path people actually run** - `0ec2c3f0` (feat)

**Plan metadata:** (this commit, following)

## Files Created/Modified

- `cmd/queen_risk_signals.go` — `PathPatterns` filled; `queenRiskSignalHitsFromPaths`, `riskSignalHitsFromRecords`, `forcedReviewerReasonClause` added; `forcedReviewerReason` and `queenForcedContinueReviewers` rewritten to union both detectors through one merge.
- `cmd/codex_continue.go` — `queenContinueDispatchesWithJudgement`, `unionForcedContinueReviewers`, `queenContinueReviewSpecsWithJudgement` gained a `changedFiles []string` parameter, threaded through to `queenForcedContinueReviewers`; `plannedContinueReviewDispatches` now computes `phaseChangedFilesFromHandoffs(phase.ID)` and passes it in; the two nil-proposal wrappers (`queenContinueDispatches`, `queenContinueReviewSpecs`) pass `nil`.
- `cmd/codex_continue_plan.go` — `plannedExternalContinueDispatches` computes and passes the same `phaseChangedFilesFromHandoffs(phase.ID)` into `queenContinueReviewSpecsWithJudgement`.
- `cmd/codex_dispatch_contract.go` — `loadWorkerHandoffRecords` guards against a nil global `store` (see Deviations).
- `cmd/queen_forced_reviewer_test.go` — `TestChangedFilesCanOnlyAddAForcedReviewer`, `TestBothContinueLanesForceTheSameReviewers` added; existing calls to the now-6-argument `queenContinueDispatchesWithJudgement` updated.
- `cmd/continue_fastpath_castes_test.go` — `TestContinueFastPathHonoursCasteProposal` extended with the no-reason refusal case.
- `cmd/owner_dials_test.go` — `TestHeavyGivesTheFullReviewPanel`, `TestAutopilotPathSendsTheFallbackTeam`, `TestForcedReviewerAddedByChangedFilesSatisfiesThePlannedSubsetCheck` added.
- `cmd/claudemd_verification_depth_test.go`, `cmd/floor_unskippable_test.go`, `cmd/queen_judgement_test.go`, `cmd/queen_orchestration_regression_test.go` — call-site arity fixes only (the new `changedFiles` parameter), no behavior change to what these tests assert.

## Decisions Made

See `key-decisions` in frontmatter. The two most consequential: (1) both detectors — plan wording and changed files — now feed the SAME `collapseToForcedReviewers` merge via `riskSignalHitsFromRecords`, rather than two independently-derived sets that could disagree; (2) the wrapper lane's planned-subset check needed no correction because the file detector fires at the SAME call that produces the persisted "planned" manifest, before any worker dispatches — proven, not assumed.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `loadWorkerHandoffRecords` panicked on a nil global `store`**
- **Found during:** Task 2, running `TestContinueFastPathHonoursCasteProposal` after Task 1's wiring made `phaseChangedFilesFromHandoffs` unconditional on every continue dispatch construction call
- **Issue:** `phaseChangedFilesFromHandoffs` → `loadWorkerHandoffRecords` → `store.ReadFile(...)` dereferences the package-global `store` variable, which several existing lightweight tests (`TestContinueFastPathHonoursCasteProposal`, `TestPhaseVerifiedOnce`, and others surfaced by the full-suite run) never initialize, because they never previously needed file I/O on this path. A full `go test ./cmd -count=1` run before the fix showed one panic (`TestPhaseVerifiedOnce`) that crashed the whole test binary.
- **Fix:** `loadWorkerHandoffRecords` now returns `(nil, nil)` when `store == nil` — "no handoffs recorded yet" is the correct answer for a caller with no store, not a crash.
- **Files modified:** `cmd/codex_dispatch_contract.go`
- **Verification:** `go test ./cmd -count=1` passes cleanly (439s, 0 failures); `go test ./... -count=1` passes except one confirmed pre-existing timing-flaky test in an unrelated package (see Issues Encountered).
- **Committed in:** `f0c582bf` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (blocking — required for the test suite to stay green; a bug this plan's own wiring change would otherwise have introduced into every lightweight caller of the newly-unconditional path).
**Impact on plan:** No scope creep — the fix is scoped to the one function this plan made unconditional.

## Issues Encountered

- `go test ./... -count=1` (whole repo) showed one failure: `TestAvailabilityProbeRetriesOnlyTimeouts` in `pkg/codex`. This is the documented pre-existing timing-flaky test named in this plan's own project notes ("failed once in the orchestrator's own full run after wave 5, passed 3/3 in isolation"). Reran in isolation: 3/3 clean (`go test ./pkg/codex -run TestAvailabilityProbeRetriesOnlyTimeouts -count=3`). Not a regression from this plan; not touched.

## Known Stubs

None. Every `must_haves.truths` deliverable and every `coverage` entry above has a passing test asserted on the real dispatch list or the real planned-subset check, never an intermediate struct.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- The changed-files detector (`queenRiskSignalHitsFromPaths`) and its union into `queenForcedContinueReviewers` are in place for 194-07 (the owner-only waiver, which needs to know which signal fired and from which detector) and 194-08 (closing `.planning/WINDOWS.md` #1's end-to-end proof).
- Both continue lanes are now proven to force the same reviewer set for the same phase, the same recorded set, and the same changed files — the two-lane parity discipline the 2026-08-21 review gate found broken is now test-locked on this new surface too.
- No blockers for 194-07.

---
*Phase: 194-the-queen-decides-the-team*
*Completed: 2026-08-23*

## Self-Check: PASSED

- FOUND: commit `f0c582bf` (Task 1) in `git log --oneline --all`
- FOUND: commit `0ec2c3f0` (Task 2) in `git log --oneline --all`
- FOUND: `cmd/queen_risk_signals.go` (queenRiskSignalHitsFromPaths, riskSignalHitsFromRecords, PathPatterns filled)
- FOUND: `cmd/codex_continue.go` (changedFiles threaded through the shared chain)
- FOUND: `cmd/codex_continue_plan.go` (changedFiles supplied at the wrapper boundary)
- FOUND: `cmd/codex_dispatch_contract.go` (nil-store guard)
- FOUND: `cmd/queen_forced_reviewer_test.go`, `cmd/continue_fastpath_castes_test.go`, `cmd/owner_dials_test.go`
- `go build ./...` — pass
- `go vet ./...` — pass
- `go test ./cmd -count=1` — pass (439s, zero failures)
- `go test ./... -count=1` (whole repo) — pass except one confirmed pre-existing timing-flaky test in an untouched package (`TestAvailabilityProbeRetriesOnlyTimeouts`, `pkg/codex`), verified clean in 3/3 isolated reruns
- `grep -c 'phaseChangedFilesFromHandoffs' cmd/codex_continue.go cmd/codex_continue_plan.go` — 3 and 2 respectively (both ≥ 1)
