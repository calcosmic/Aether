---
phase: 194-the-queen-decides-the-team
plan: 05
subsystem: orchestration
tags: [queen, caste-relevance, spawn-budget, verification-depth, judgement, D-06, D-11, D-13]

# Dependency graph
requires:
  - phase: 194-01
    provides: "queenForcedReviewersForPhase / queenRiskSignalTable -- the forced-reviewer set queenFallbackTeam folds in for continue"
  - phase: 194-02
    provides: "the shrunken build floor (builder alone / nil on discovery) this plan's queenFallbackTeam builds its required-caste dispatches from"
  - phase: 194-03
    provides: "queenRuntimeReasonForCaste / mergeCasteReasons -- the per-caste reason machinery queenFallbackTeam reuses for its own dispatches"
provides:
  - "queenFallbackTeam / queenSelectorIsGatedForFlow: the no-proposal team for build and continue is the required-caste floor only (builder, plus any forced reviewer at continue, plus one scout on a discovery build) -- the keyword/relevance engine no longer selects on either flow"
  - "isAlwaysRequired's continue branch matches D-13 exactly: light/standard require nothing unconditionally; heavy requires the security and quality reviewer always, the coverage reviewer only where the phase produces testable code"
  - "resolveSmartVerificationDepth no longer raises depth from phase position -- the last phase of a plan gets the same automatic depth as a middle phase with identical mode and risk"
  - "queenApplyJudgement refuses a proposed coverage reviewer by name on a phase with no testable code, closing the skipped subtest 194-02 left for this plan"
  - "the external/wrapper continue lane's watcher relay is gated on skipWatchers alone, not on watcher's caste selection -- a bug this plan's own change would otherwise have introduced (a depth flag silently dropping the deterministic floor's relay)"
affects: [194-06-cost-line, 194-07-waive-choice, 194-08-close-windows-1, 194-09-claudemd-rewrite]

# Actuals (#2632)
actuals:
  tokens: 25139
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "queenFallbackTeam delegates to queenCandidateDispatches unchanged for every flow D-11 does not name (plan, colonize, swarm, seal) -- one gate function (queenSelectorIsGatedForFlow) decides scope, not a duplicated switch"
    - "Tests that exercise the SCORING REGISTRY (which caste a phase's wording earns) now call queenCandidateDispatches directly; tests that exercise the NO-PROPOSAL ENTRY POINT call queenOrchestrate -- conflating the two after this plan silently asserts a mechanism that no longer runs"
    - "A structural dispatch (the continue watcher relay, which reports the ALREADY-COMPUTED deterministic floor to an external worker) must never be gated on caste-relevance selection -- it is a skipWatchers/owner-controlled decision, not a judgement outcome"

key-files:
  created:
    - cmd/queen_fallback_team_test.go
    - cmd/owner_dials_test.go
  modified:
    - cmd/caste_relevance.go
    - cmd/queen_judgement.go
    - cmd/review_depth.go
    - cmd/codex_dispatch_contract.go
    - cmd/codex_continue_plan.go
    - cmd/queen_probe_gating_test.go
    - cmd/caste_relevance_test.go
    - cmd/caste_relevance_doc_test.go
    - cmd/continue_depth_spawn_test.go
    - cmd/continue_fastpath_castes_test.go
    - cmd/queen_judgement_test.go
    - cmd/queen_orchestration_regression_test.go
    - cmd/queen_relevance_floor_test.go
    - cmd/reclaim_wiring_test.go
    - cmd/review_depth_test.go
    - cmd/claudemd_verification_depth_test.go
    - cmd/codex_build_test.go
    - cmd/codex_build_finalize_test.go
    - cmd/codex_continue_test.go
    - cmd/codex_continue_plan_test.go
    - cmd/codex_visuals_test.go
    - cmd/build_attempt_external_test.go
    - cmd/testdata/golden_build.txt
    - cmd/testdata/golden_continue.txt
    - cmd/testdata/golden_plan.txt
    - .aether/docs/retired-tests-ledger.md

key-decisions:
  - "queenFallbackTeam builds its dispatches directly from queenRequiredCastesForBudget rather than filtering queenCandidateDispatches' output, because the point of D-11 is that no scoring happens at all on the no-proposal build/continue path -- filtering a scored list would still compute scores nobody asked for."
  - "The continue watcher relay's gate (plannedExternalContinueDispatches) was decoupled from queenContinueHasCaste entirely, not just re-derived. It used to be a redundant check (watcher was always in queenContinueDispatches); after D-13 it became an active bug (the relay silently vanished at light/standard depth). Since the relay reports the deterministic floor that already ran, not a caste-selection outcome, coupling it to caste selection was wrong even before this plan, and CLAUDE.md's 'a depth flag must never remove a program check' makes the fix mandatory rather than optional."
  - "Reconciliation tests distinguish two questions with two different call sites: 'does the scoring registry still score this caste correctly' (queenCandidateDispatches, ~15 tests) versus 'does the no-proposal entry point still send it' (queenOrchestrate, only the plan's own new tests and the ones asserting the gated floor). Conflating them would have hidden the actual behavior change behind a passing test."
  - "TestQueenTrimsContinueReviewersAndKeepsTheWatcher was retired rather than adjusted: its whole premise (judgement trims a keyword-over-selecting deterministic engine) is now impossible to observe, since the no-proposal engine no longer over-selects on continue at all -- there is no smaller-than-empty team to trim to."
  - "TestProbeStillRequiredWhenTheRepositoryHasCode was retired rather than adjusted: D-13 removed standard depth's unconditional Probe membership entirely, and its actual protected claim (the code-detection gate itself still distinguishes real code from documentation) survives as a REFUSAL rule instead, already covered by this plan's own extension to TestProbeIsRequiredOnlyWhereItCanFindSomething."

patterns-established:
  - "One derivation, one boundary extends to structural (non-caste) dispatches too: a relay of an already-computed result must be gated on the owner's own flag, never on whether a caste-selection mechanism happens to also want that caste this run."

requirements-completed: [TEAM-04, TEAM-05]

coverage:
  - id: D1
    description: "With no proposal, a build sends the implementation worker plus any forced reviewer and nothing else; a one-task bug-fix phase gets exactly the builder at build and no reviewer at continue."
    requirement: TEAM-04
    verification:
      - kind: unit
        ref: "cmd/queen_fallback_team_test.go#TestNoProposalSendsBuilderPlusForcedOnly"
        status: pass
    human_judgment: false
  - id: D2
    description: "A discovery-mode build with no proposal sends exactly one research worker, because the implementation caste is suppressed there and findings are the deliverable."
    requirement: TEAM-04
    verification:
      - kind: unit
        ref: "cmd/queen_fallback_team_test.go#TestDiscoveryFallbackSendsOneResearcher"
        status: pass
    human_judgment: false
  - id: D3
    description: "The keyword/relevance engine no longer selects an optional specialist on build or continue's no-proposal path, even when the phase's own wording would clear its score threshold -- the relevance registry itself is proven unchanged (still scores the same phase the same way through queenCandidateDispatches) so the change is isolated to the entry point."
    requirement: TEAM-04
    verification:
      - kind: unit
        ref: "cmd/queen_fallback_team_test.go#TestKeywordEngineNoLongerSelectsOnBuildOrContinue"
        status: pass
    human_judgment: false
  - id: D4
    description: "Plan, colonize, swarm and seal produce the identical caste set they produced before this plan -- proven by comparing queenOrchestrate's output against the pre-plan computation for a representative phase on each flow, not merely asserting the gate function's own scope."
    requirement: TEAM-04
    verification:
      - kind: unit
        ref: "cmd/queen_fallback_team_test.go#TestOtherFlowsKeepTheirRequiredSets"
        status: pass
    human_judgment: false
  - id: D5
    description: "Continue's required set by depth is exactly: light -> none, standard -> none, heavy -> security reviewer + quality reviewer always, coverage reviewer only where the phase produces testable code."
    requirement: TEAM-05
    verification:
      - kind: unit
        ref: "cmd/owner_dials_test.go#TestContinueRequiredSetByDepth"
        status: pass
    human_judgment: false
  - id: D6
    description: "A light request can never drop a forced reviewer -- the forced reviewer bypasses the budget/depth trim entirely because it is unioned into the required-caste set regardless of depth."
    requirement: TEAM-05
    verification:
      - kind: unit
        ref: "cmd/owner_dials_test.go#TestLightCannotDropAForcedReviewer"
        status: pass
    human_judgment: false
  - id: D7
    description: "An explicitly named, reasoned caste survives a light request -- the more specific instruction (--castes) outranks the general one (--light)."
    requirement: TEAM-05
    verification:
      - kind: unit
        ref: "cmd/owner_dials_test.go#TestExplicitCastesOutrankLight"
        status: pass
    human_judgment: false
  - id: D8
    description: "Phase position no longer raises verification depth: the last phase of a five-phase plan gets the same automatic depth as a middle phase with identical name, mode and risk; an explicit --heavy request still raises depth on that same middle phase."
    requirement: TEAM-05
    verification:
      - kind: unit
        ref: "cmd/owner_dials_test.go#TestPositionNoLongerRaisesVerificationDepth"
        status: pass
    human_judgment: false
  - id: D9
    description: "A proposed coverage reviewer (Probe) is refused by name on a documentation-only phase and reaches Final on a phase that produces testable code -- the refusal gate 194-02 left as a skipped subtest for this plan."
    verification:
      - kind: unit
        ref: "cmd/queen_probe_gating_test.go#TestProbeIsRequiredOnlyWhereItCanFindSomething"
        status: pass
      - kind: other
        ref: "grep -c 't.Skip' cmd/queen_probe_gating_test.go = 0"
        status: pass
    human_judgment: false
  - id: D10
    description: "The full package and whole-repo test suites pass with the change, including the ~25 existing tests and 3 golden fixtures this plan reconciled to the new floor."
    verification:
      - kind: unit
        ref: "go test ./cmd -count=1"
        status: pass
      - kind: unit
        ref: "go test ./... -count=1 (whole repo, excluding the network-dependent TestPackedNPMReleaseCandidateContract)"
        status: pass
      - kind: other
        ref: "go build ./cmd/aether && go vet ./..."
        status: pass
    human_judgment: false

duration: 87min
completed: 2026-08-23
status: complete
---

# Phase 194 Plan 5: The Automatic Path Costs What the Judgement Path Costs Summary

**With no proposal, build now sends the implementation worker plus any forced reviewer and nothing else, continue's required reviewers by depth are light/standard: none and heavy: the full panel, and a phase's position in the plan no longer buys it a heavier review — closing the last gap between the automatic (autopilot) path and the judgement path this milestone was built to align.**

## Performance

- **Duration:** ~87 min
- **Started:** 2026-08-23T12:43:38Z (approx)
- **Completed:** 2026-08-23T14:10:14Z (approx)
- **Tasks:** 2 completed
- **Files modified:** 28 (2 new: `cmd/queen_fallback_team_test.go`, `cmd/owner_dials_test.go`)

## Accomplishments

- `queenFallbackTeam` (`cmd/caste_relevance.go`) is the new no-proposal answer for build and continue: the required-caste floor only (`builder` on a non-discovery build; any forced reviewer at continue; one `scout` on a discovery build), computed directly from `queenRequiredCastesForBudget` rather than filtering a scored candidate list. `queenSelectorIsGatedForFlow` is the single gate function naming exactly the two flows D-11 changes; every other flow (plan, colonize, swarm, seal) is proven byte-for-byte unchanged.
- `isAlwaysRequired`'s continue branch now matches D-13's table exactly: light and standard require nothing unconditionally (Watcher's and Probe's last unconditional continue membership is gone); heavy requires the security and quality reviewer always, and the coverage reviewer only where the phase produces testable code.
- `resolveSmartVerificationDepth` no longer escalates to heavy from phase position alone — the "final phase" clause is removed from both mode branches (production, and prototype/maintenance); the last phase of a plan is treated exactly like a middle phase with the same risk, and an explicit `--heavy` request still raises depth on any phase.
- `queenApplyJudgement`'s refusal loop now refuses a proposed coverage reviewer by name when the phase produces no testable code — landing the refusal gate plan 194-02 deliberately left as a skipped subtest (`TestProbeIsRequiredOnlyWhereItCanFindSomething`), now un-skipped with two new assertions (refused on a doc-only phase, kept on a testable-code phase).
- Found and fixed a real bug this plan's own change would otherwise have introduced: the external/wrapper continue lane's watcher relay (`plannedExternalContinueDispatches`) was gated on `queenContinueHasCaste(queenDispatches, "watcher")` in addition to `!skipWatchers` — a redundant check while watcher was unconditionally selected, but an active bug once D-13 removed that unconditional membership, since it would have silently dropped the relay of the already-computed deterministic verification report at light/standard depth. Decoupled entirely: `skipWatchers` alone is now the gate, matching CLAUDE.md's "a depth flag must never be able to remove a program check."
- `buildQueenSpawnBudgetContract`'s pruned-caste reason now says the true thing for a gated flow ("nobody asked for it and nothing in this phase forces it") instead of citing a budget number that played no part in the decision.
- Reconciled ~25 existing tests and 3 golden fixtures (`golden_build.txt`, `golden_continue.txt`, `golden_plan.txt`) to the new floor, following the "assert the spawn list, not the decision record" discipline: tests that exercise the scoring registry itself now call `queenCandidateDispatches` directly (unchanged by this plan); tests that exercise the no-proposal entry point call `queenOrchestrate` and assert the smaller team. Retired two tests whose entire premise no longer exists (`TestQueenTrimsContinueReviewersAndKeepsTheWatcher`, `TestProbeStillRequiredWhenTheRepositoryHasCode`), both cited in the retired-tests ledger with their replacement coverage named.

## Task Commits

1. **Task 1: The automatic team becomes the implementation worker plus any forced reviewer** - `ec7a27e7` (feat)
2. **Task 2: The three dials — depth means the review panel, position means nothing, light cannot lower the floor** - `fa4482f4` (feat)

**Plan metadata:** (this commit, following)

## Files Created/Modified

- `cmd/caste_relevance.go` — `queenFallbackTeam`, `queenSelectorIsGatedForFlow`, `queenOrchestrate` rewired to call the fallback team; `isAlwaysRequired`'s continue branch rewritten to D-13's table; `spawnThreshold`'s doc comment updated to state it no longer selects on build or continue.
- `cmd/queen_judgement.go` — the refusal loop's new probe-testable-code gate.
- `cmd/review_depth.go` — `resolveSmartVerificationDepth`'s position-final clause removed from both mode branches; the production branch's now-unused `position` variable removed.
- `cmd/codex_dispatch_contract.go` — `buildQueenSpawnBudgetContract`'s pruned-reason composition made gate-aware.
- `cmd/codex_continue_plan.go` — `plannedExternalContinueDispatches`'s watcher-relay gate decoupled from caste selection.
- `cmd/queen_fallback_team_test.go` (new) — `TestNoProposalSendsBuilderPlusForcedOnly`, `TestDiscoveryFallbackSendsOneResearcher`, `TestKeywordEngineNoLongerSelectsOnBuildOrContinue`, `TestOtherFlowsKeepTheirRequiredSets`.
- `cmd/owner_dials_test.go` (new) — `TestContinueRequiredSetByDepth`, `TestLightCannotDropAForcedReviewer`, `TestExplicitCastesOutrankLight`, `TestPositionNoLongerRaisesVerificationDepth`.
- `cmd/queen_probe_gating_test.go` — the skipped subtest replaced with two real assertions on `queenApplyJudgement`'s `Refused`/`Final` output.
- `cmd/caste_relevance_test.go`, `cmd/caste_relevance_doc_test.go`, `cmd/continue_depth_spawn_test.go`, `cmd/continue_fastpath_castes_test.go`, `cmd/reclaim_wiring_test.go` — fixtures/call sites moved from `queenOrchestrate` to `queenCandidateDispatches` where the test's real subject is the scoring registry, not the gated entry point; assertions corrected where the underlying invariant itself changed (D-13's removal of Watcher/Probe's continue floor).
- `cmd/queen_judgement_test.go`, `cmd/queen_orchestration_regression_test.go`, `cmd/queen_relevance_floor_test.go` — explicit per-worker reasons added to proposals that previously relied on Watcher's now-removed continue exemption; two tests retired (see key-decisions).
- `cmd/review_depth_test.go`, `cmd/claudemd_verification_depth_test.go` — position-final expectations corrected to standard depth (D-06); `TestCLAUDEMDVerificationDepthClaims`'s runtime assertion follows the runtime, not CLAUDE.md's not-yet-rewritten prose (194-09's job).
- `cmd/codex_build_test.go`, `cmd/codex_build_finalize_test.go`, `cmd/build_attempt_external_test.go` — dispatch/wave/execution-plan counts corrected to the shrunken no-proposal team; `prepareExternalBuildCompletionWithProposal` (new helper) added for fixtures that need a genuine second worker in the real build manifest without breaking the durable build-attempt hash.
- `cmd/codex_continue_test.go`, `cmd/codex_continue_plan_test.go`, `cmd/codex_visuals_test.go` — dispatch counts and caste lists corrected; explicit proposals added where a test's own name says "Queen-selected."
- `cmd/testdata/golden_build.txt`, `cmd/testdata/golden_continue.txt`, `cmd/testdata/golden_plan.txt` — regenerated via `-update-golden` and hand-diffed; changes are exactly the position/floor text and the removed Probe review-wave step.
- `.aether/docs/retired-tests-ledger.md` — 2 new entries.

## Decisions Made

See `key-decisions` in frontmatter. The two most consequential: (1) `queenFallbackTeam` computes dispatches directly from the required-caste floor rather than filtering a scored list, so no scoring runs at all on the no-proposal path — the literal reading of D-11; (2) the continue watcher relay's gate was decoupled from caste selection entirely rather than patched, because coupling a structural relay of an already-computed result to a judgement mechanism was the wrong design even before this plan, and CLAUDE.md's Definition of Done makes the fix mandatory once D-13 exposed it.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] The continue watcher relay silently dropped at light/standard depth**
- **Found during:** Task 2, while reconciling `TestContinueFinalizeRejectsCompletedWorkerWithoutHandoff` and `TestContinueExternalDispatchBriefsStateHandoffSchemaOnceNotOnNativePath`
- **Issue:** `plannedExternalContinueDispatches` gated the external/wrapper lane's watcher relay (the worker that carries the already-computed deterministic verification report) on `queenContinueHasCaste(queenDispatches, "watcher")` in addition to `!skipWatchers`. This was a no-op while Watcher was unconditionally selected at every continue depth; once this plan's own Task 2 removed that unconditional membership (D-13), the relay silently stopped appearing at light and standard depth even when the owner never asked to skip it — a depth-driven removal of a program-check relay, which CLAUDE.md explicitly prohibits.
- **Fix:** Removed the `queenContinueHasCaste` condition entirely; `!skipWatchers` alone now gates the relay, matching its actual job (report the deterministic floor's own already-run result, not a caste-selection outcome).
- **Files modified:** `cmd/codex_continue_plan.go`
- **Verification:** Both named tests pass; `go test ./cmd -count=1` green; golden fixtures confirm no unintended change to the in-process lane (unaffected, uses a separate, always-unconditional watcher dispatch mechanism).
- **Committed in:** `fa4482f4` (Task 2 commit)

**2. [Rule 3 - Blocking] Reconciling the suite touched 26 files, not the plan's named 8**
- **Found during:** Task 1 and Task 2, after each `go test ./cmd -count=1` run
- **Issue:** Gating the no-proposal fallback (Task 1) and rewriting `isAlwaysRequired`'s continue branch plus removing position from verification depth (Task 2) broke ~25 tests and 3 golden fixtures across the package — every test that asserted the deleted keyword-selection fallback, the deleted continue floor, or the deleted position-final escalation, through call sites the plan's own `<files>` list did not (and could not) enumerate.
- **Fix:** Applied the same rewrite-or-retire discipline established by plans 194-02/194-03/194-04 to every failure the full suite surfaced: switched tests whose real subject is the scoring registry to call `queenCandidateDispatches` directly; added explicit per-worker proposals where a test's own name says "Queen-selected" and the no-proposal fallback no longer supplies that caste; corrected count/order assertions to the new floor; regenerated and hand-diffed golden fixtures; retired two tests whose premise no longer exists, each with a ledger entry naming the replacement coverage.
- **Files modified:** see `key-files.modified` above (full list).
- **Verification:** `go test ./cmd -count=1` and `go test ./... -count=1` (whole repo) both pass; `go vet ./...` and `go build ./cmd/aether` clean.
- **Committed in:** `ec7a27e7` (Task 1) and `fa4482f4` (Task 2), split by which fixture each edit belonged to.

---

**Total deviations:** 2 auto-fixed (1 bug — a genuine program-check removal the plan's own change would have caused; 1 blocking — required for the suite to stay green as CLAUDE.md's Definition of Done demands). No scope creep beyond what the floor rewrite and the depth-table rewrite genuinely required; no assertion was loosened, only moved to where the claim it protects now lives.

## Issues Encountered

- One full-suite `go test ./... -count=1` run showed `TestAvailabilityProbeRetriesOnlyTimeouts` (`pkg/codex`) failing under disk-pressure conditions (the sandbox's temp filesystem hit 100% during a concurrent `go build` inside the blackbox CLI test suite, which produced a cascade of unrelated `no space left on device` failures across `TestCLI*` and `TestZeroReviewerPhase*`). Ran `go clean -cache` to reclaim ~50GB, then reran every affected test individually and the full suite twice more — all green, including 3/3 clean isolated runs of `TestAvailabilityProbeRetriesOnlyTimeouts`, matching this project's documented pre-existing timing-flaky pattern for that test. Not a regression from this plan; not touched beyond the environment cleanup.

## Known Stubs

None. Every `must_haves.truths` deliverable and every `coverage` entry above has a passing test asserted on the real dispatch list, the real `queenApplyJudgement` output, or the real rendered manifest/golden text — never an intermediate struct.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `queenFallbackTeam`, `queenSelectorIsGatedForFlow`, the rewritten `isAlwaysRequired` continue branch, and the position-free `resolveSmartVerificationDepth` are in place for plan 194-06 (the cost line) and 194-07 (the waive choice) to read the same, now-honest, required-caste and depth surfaces without re-deriving them.
- `.planning/WINDOWS.md` #1's remaining half (the end-to-end manifest+continue test naming both boundaries) is unaffected by this plan and remains 194-08's job.
- CLAUDE.md's "Queen-Owned Orchestration" depth table (`Final phase → heavy` and the old light/standard/heavy continue membership) is now stale prose describing behaviour this plan changed — correcting it is explicitly plan 194-09's job, not this plan's, per this plan's own required-reading note.
- No blockers for 194-06.

---
*Phase: 194-the-queen-decides-the-team*
*Completed: 2026-08-23*

## Self-Check: PASSED

- FOUND: commit `ec7a27e7` (Task 1) in `git log --oneline --all`
- FOUND: commit `fa4482f4` (Task 2) in `git log --oneline --all`
- FOUND: `cmd/queen_fallback_team_test.go`
- FOUND: `cmd/owner_dials_test.go`
- FOUND: `cmd/caste_relevance.go` (queenFallbackTeam, queenSelectorIsGatedForFlow, isAlwaysRequired)
- FOUND: `cmd/queen_judgement.go` (probe testable-code refusal gate)
- FOUND: `cmd/review_depth.go` (position-final clause removed)
- FOUND: `cmd/codex_continue_plan.go` (watcher relay decoupled from caste selection)
- FOUND: `.aether/docs/retired-tests-ledger.md`
- `go build ./cmd/aether` — pass
- `go vet ./...` — pass
- `go test ./cmd -count=1` — pass (436s, zero failures)
- `go test ./... -count=1` (whole repo, excluding the network-dependent `TestPackedNPMReleaseCandidateContract`) — pass (one confirmed non-regression, pre-existing timing-flaky test in an untouched package, verified clean in 3/3 isolated reruns)
- Acceptance criteria greps (`grep -c 't.Skip' cmd/queen_probe_gating_test.go` = 0, pruned-reason string contains no budget number) — all verified pass
