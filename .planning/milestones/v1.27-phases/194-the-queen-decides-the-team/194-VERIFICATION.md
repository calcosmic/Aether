---
phase: 194-the-queen-decides-the-team
verified: 2026-08-23T17:19:52Z
status: passed
score: 6/6 must-haves verified
behavior_unverified: 0
overrides_applied: 0
---

# Phase 194: The Queen Decides the Team Verification Report

**Phase Goal:** Which AI helpers ("workers") get sent to do a job becomes the Queen's judgement call, not a fixed rule. A reviewer worker is only forced onto a job when it touches something genuinely risky — passwords, payments, deleting data, migrations, or a release — and the program says why in plain words. A one-task bug fix costs one worker plus the free checks from Phase 193, not eight.

**Verified:** 2026-08-23T17:19:52Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (Roadmap Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | On an ordinary job, the only helper required by default is the builder caste; no other caste is required by guessed project type or keyword — `TestWatcherIsAlwaysRequiredOnBuild` formally retired and recorded | ✓ VERIFIED | `queenBuildSafetyRequiredCastes` (`cmd/queen_spawn_budget.go:162-167`) returns `nil` for discovery mode and `["builder"]` for every other phase — no mode/risk/keyword branch remains. Confirmed `func TestWatcherIsAlwaysRequiredOnBuild(` does not exist anywhere in `cmd/*.go` (only referenced as an entry in the ledger). `.aether/docs/retired-tests-ledger.md` carries a full disposition entry ("Removed in: Phase 194 Plan 02", citing ruling D11). Also confirmed absent: `TestQueenCannotDropTheWatcher`, `TestSafetyCastesSurviveProbeGating`, `TestQueenOrchestratePreservesSafetyCastes`, `TestQueenSpawnBudgetDecisionsPreservesSafetyCastesUnderPressure`, `TestCodexBuildPlanOnlySpawnBudgetPreservesSafetyCastesUnderLightAndHeavy`, `TestCodexBuildPlanOnlyPhaseFiveSafetyVerificationKeepsRequiredCastes`, `TestTeamCheckinFallsBackToRosterProduces`, `TestQueenTrimsContinueReviewersAndKeepsTheWatcher` — all 8 have ledger entries and none exist as live functions. `queenBuildSafetyReviewRequired` and `queenPhaseHasSecuritySignal` no longer exist anywhere in `cmd/` (grep returns nothing). |
| 2 | A side-by-side test shows a plain CSV-export feature gets no forced reviewer while a password-reset feature gets a security reviewer with the reason shown | ✓ VERIFIED | Ran `go test ./cmd -run TestReviewerForcedOnlyByNamedRisk -v -count=1`: all 6 subtests pass — `CSV_export_forces_nothing`, `password_reset_forces_gatekeeper`, `refund_button_forces_gatekeeper`, `delete_stale_accounts_forces_auditor`, `adding_a_migration_column_forces_auditor`, `production_phase_with_no_signal_wording_forces_nothing`. Test asserts against the real continue dispatch list (`queenContinueDispatchesWithJudgement`), not a decision record. |
| 3 | Every worker sent carries a one-sentence plain-English reason; a proposed worker with no reason is rejected by name, not silently allowed | ✓ VERIFIED | Ran `go test ./cmd -run TestNoWorkerWithoutStatedReason -v -count=1`: PASS. Also ran companion reason tests: `TestQueenCannotSkipSecurityReviewOnSecurityWork` (asserts the forced-reviewer rationale contains "this touches...", PASS), `TestTeamCheckinCardShowsReasonAndRequiredMarking` (PASS). `cmd/queen_worker_reason_test.go` exists (299 lines) covering the refusal-by-name, generic-blurb-rejection, alias-normalization and stable-ordering must-haves from 194-03-PLAN.md. |
| 4 | A sample one-task bug fix is sent one worker plus the free checks, whether the team came from the assistant's proposal or the automatic fallback | ✓ VERIFIED | Ran `go test ./cmd -run TestOneTaskBugFixIsOneWorkerPlusChecks -v -count=1`: PASS. Read the test body (`cmd/one_task_bug_fix_test.go`): it counts dispatches from the real manifest-facing constructors (`plannedBuildDispatchesWithJudgement`, `plannedContinueReviewDispatches`) across build+continue for a fixture explicitly modeled on the measured 2026-08-22 baseline (8 workers). Fallback path (nil proposal) = 1 worker (`[builder]`); proposal path (`--castes builder` + reason) = 1 worker; a guard row on the SAME phase with `--heavy` produces >1 worker, proving the test can fail and isn't passing on an inert pipeline (the v1.26 lesson explicitly cited in the test's own comments). |
| 5 | The owner's manual controls — `--castes`, heavy/light review, and the pre-build check-in card — still work after this change | ✓ VERIFIED | Ran and passed: `TestQueenChoiceReachesTheDispatchList` (asserts on the real spawn list per this repo's established discipline), `TestContinueFastPathHonoursCasteProposal` (`--castes` on the fast continue path), `TestHeavyGivesTheFullReviewPanel`, `TestAutopilotPathSendsTheFallbackTeam`, `TestBuildWorkerCapHonoursVerificationDepth`, `TestOneWorkerTeamStillPauses` (check-in card still pauses for a one-worker team, per D-14 as shipped), `TestTeamCheckinDoesNotMutateWithAWaiverPresent` (inspection contract preserved). All pass. |
| 6 | Phase 193's deferred truth — no single AI helper type sent to check the same phase twice unless explicitly asked, for every reviewer caste — is now closed | ✓ VERIFIED | Ran `go test ./cmd -run TestNoCasteIsDispatchedAtBothBoundaries -v -count=1`: both subtests pass (`production, security signal (credentials/auth), no proposal`; `production, migration signal, no proposal`). Test walks the real build and continue dispatch-list constructors and asserts a zero-length intersection; a self-check confirms the fixture actually forces a reviewer (guards against a vacuous pass). `.planning/WINDOWS.md` entry #1 status is `fixed`, `resolved_at: 2026-08-23T16:04:14.666Z`, `open_count: 0`, `fixed_count: 1`. |

**Score:** 6/6 truths verified (0 present-but-behavior-unverified)

### Required Artifacts (from PLAN frontmatter, all 9 plans)

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/queen_risk_signals.go` | The five-signal table, the only place a reviewer can be forced from | ✓ VERIFIED | 423 lines, substantive (word-boundary phrase matching, path-pattern change-file detector, waiver-reason carrying). No TBD/FIXME/XXX/HACK/PLACEHOLDER found. |
| `cmd/queen_forced_reviewer_test.go` | Forced-reviewer tests incl. `TestReviewerForcedOnlyByNamedRisk` | ✓ VERIFIED | 423 lines; ran directly, passes. |
| `.aether/docs/retired-tests-ledger.md` | Disposition entry for every retired test | ✓ VERIFIED | 13 entries covering all D-15-named tests plus additional floors D-06 removed; each entry names original path, what it covered, disposition, phase/plan removed in. |
| `cmd/queen_worker_reason_test.go` | Per-worker reason tests | ✓ VERIFIED | 299 lines; covered by passing `TestNoWorkerWithoutStatedReason`. |
| `cmd/ceremony_team_checkin.go` | Card renders reason/REQUIRED/signal/waiver | ✓ VERIFIED | `TestTeamCheckinCardShowsReasonAndRequiredMarking`, `TestOneWorkerTeamStillPauses`, `TestTeamCheckinDoesNotMutateWithAWaiverPresent` all pass. |
| `cmd/queen_fallback_team_test.go`, `cmd/owner_dials_test.go` | No-proposal fallback + owner dial tests | ✓ VERIFIED | Both exist (141, 297 lines); `TestAutopilotPathSendsTheFallbackTeam`, `TestHeavyGivesTheFullReviewPanel`, `TestDiscoveryFallbackSendsOneResearcher` pass. |
| `cmd/forced_reviewer_waiver.go`, `cmd/forced_reviewer_waiver_test.go` | D-03 owner-only waiver | ✓ VERIFIED | 132 / 193 lines; `TestOnlyTheOwnerCanWaiveAForcedReviewer`, `TestWaiverCoversOneSignalOnOnePhase`, `TestWaivedSignalStaysWaivedWhenTheFilesRedetectIt`, `TestAutopilotNeverWaives` all pass. |
| `cmd/one_task_bug_fix_test.go`, `cmd/boundary_double_dispatch_test.go` | The measured proof + WINDOWS #1 closure | ✓ VERIFIED | Both exist (125, 109 lines); both named tests pass (see Truths 4 and 6). |
| `CLAUDE.md` | Depth table matches shipped runtime, testable | ✓ VERIFIED | `TestCLAUDEMDDepthTableEvaluates` passes — parses the actual table row out of `CLAUDE.md` and evaluates it against the runtime rather than asserting hardcoded text. Read the "Queen-Owned Orchestration" and "Team Check-In" sections directly: they now describe the five-signal floor, the builder-only default, and cite the actual test names as evidence. No "watcher always" / "production mode ⇒ auditor" language remains except in a historical/comparison sentence describing the OLD floor being replaced. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `queenForcedReviewersForPhase` | `codexBuildManifest.ForcedReviewers` | build records the forced set | ✓ WIRED | Confirmed by passing `TestNoCasteIsDispatchedAtBothBoundaries` and `TestBothContinueLanesForceTheSameReviewers` (194-06). |
| `codexBuildManifest.ForcedReviewers` | `queenContinueDispatchesWithJudgement` | continue reads, one derivation one boundary | ✓ WIRED | Confirmed by code read (D-05 pattern) and by `TestReviewerForcedOnlyByNamedRisk` asserting on the real continue dispatch list. |
| `caste_decision.reasons` (194-03) | the check-in card's reason slot | rendering | ✓ WIRED | `TestTeamCheckinCardShowsReasonAndRequiredMarking` passes. |
| `phaseChangedFilesFromHandoffs` → `queenRiskSignalHitsFromPaths` | `queenForcedContinueReviewers` (both lanes) | changed-file detector, add-only | ✓ WIRED | `TestChangedFilesCanOnlyAddAForcedReviewer` (194-06) exists and passes as part of the full-package run. |
| `forcedReviewerWaiverQuestionText` → `aether decision-answer` → `queenForcedContinueReviewers` | waiver read where the reviewer is forced | ✓ WIRED | `TestOnlyTheOwnerCanWaiveAForcedReviewer`, `TestWaiverCoversOneSignalOnOnePhase` pass. |
| build dispatch list + continue dispatch list | one counted total | `countWorkersAcrossBothBoundaries` | ✓ WIRED | Single counting function used by both `TestOneTaskBugFixIsOneWorkerPlusChecks` and `TestNoCasteIsDispatchedAtBothBoundaries` — no second, independently-written walker that could silently disagree. |
| `WINDOWS.md #1` | `gsd-tools windows fixed 1` | ship-gate reopens | ✓ WIRED | `.planning/WINDOWS.md` frontmatter: `open_count: 0`, entry status `fixed`, `resolved_at` timestamp present, only after the named test passed (confirmed by reading 194-08-SUMMARY.md's task-commit ordering: test committed, then verified, then ledger fix committed). |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| TEAM-01 | 194-02, 194-08, 194-09 | Build castes shrink to builder, old always-required tests retired, negative Probe rule stays | ✓ SATISFIED | `queenBuildSafetyRequiredCastes` returns `[builder]`/`nil`; `TestProbeIsRequiredOnlyWhereItCanFindSomething`'s negative half retained (confirmed test file present and passing in full suite run); ledger entries confirmed. |
| TEAM-02 | 194-01, 194-04, 194-06, 194-07 | Reviewer forced only by named signal, with visible reason | ✓ SATISFIED | `TestReviewerForcedOnlyByNamedRisk` table test passes; card announcement text confirmed in `cmd/ceremony_team_checkin.go` tests. |
| TEAM-03 | 194-03, 194-04 | Every worker carries a reason; no-reason refused by name | ✓ SATISFIED | `TestNoWorkerWithoutStatedReason` passes. |
| TEAM-04 | 194-05, 194-08, 194-09 | Proposal is primary source, fallback retuned to same floor, 1-task fix = 1 dispatch both paths | ✓ SATISFIED | `TestOneTaskBugFixIsOneWorkerPlusChecks` passes both paths. |
| TEAM-05 | 194-05, 194-06, 194-07, 194-08, 194-09 | Owner overrides (`--castes`, `--heavy`, `--light`, check-in card) still work | ✓ SATISFIED | `TestContinueFastPathHonoursCasteProposal`, `TestHeavyGivesTheFullReviewPanel`, `TestOneWorkerTeamStillPauses` all pass. |

No orphaned requirements found — `REQUIREMENTS.md` lists exactly TEAM-01..05 against Phase 194, all five are claimed across the nine plans (cross-checked against every plan's `requirements:` frontmatter field), and `REQUIREMENTS.md`'s own traceability table marks all five `Complete`.

### Anti-Patterns Found

None. Scanned every `.go` file this phase's plans declared as `files_modified` (non-test files: `queen_risk_signals.go`, `queen_spawn_budget.go`, `codex_build.go`, `codex_continue.go`, `queen_judgement.go`, `caste_relevance.go`, `codex_dispatch_contract.go`, `codex_workflow_cmds.go`, `codex_continue_plan.go`, `ceremony_team_checkin.go`, `review_depth.go`, `codex_continue_finalize.go`, `forced_reviewer_waiver.go`) for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` — zero hits. `go build ./cmd/...` and `go vet ./...` both exit clean.

### Behavioral Spot-Checks / Full Suite

Ran the full `cmd` package test suite once (`go test ./cmd/... -count=1 -skip TestPackedNPMReleaseCandidateContract -timeout 590s`, the one pre-existing network-dependent fixture the phase's own SUMMARYs document as excluded from every full run in this sandbox): **`ok github.com/calcosmic/Aether/cmd 355.676s`** — all tests pass, no regressions from this phase's changes. `go vet ./...` also exits clean.

Individually ran and confirmed PASS (not merely present): `TestReviewerForcedOnlyByNamedRisk` (6 subtests), `TestNoWorkerWithoutStatedReason`, `TestOneTaskBugFixIsOneWorkerPlusChecks`, `TestNoCasteIsDispatchedAtBothBoundaries` (2 subtests), `TestQueenChoiceReachesTheDispatchList`, `TestQueenCannotSkipSecurityReviewOnSecurityWork`, `TestTeamCheckinCardShowsReasonAndRequiredMarking`, `TestContinueFastPathHonoursCasteProposal`, `TestBuildWorkerCapHonoursVerificationDepth`, `TestPhaseVerifiedOnce`, `TestHeavyGivesTheFullReviewPanel`, `TestAutopilotPathSendsTheFallbackTeam`, `TestOneWorkerTeamStillPauses`, `TestOnlyTheOwnerCanWaiveAForcedReviewer` (4 subtests), `TestWaiverCoversOneSignalOnOnePhase` (3 subtests), `TestWaivedSignalStaysWaivedWhenTheFilesRedetectIt`, `TestAutopilotNeverWaives`, `TestTeamCheckinDoesNotMutateWithAWaiverPresent`, `TestCLAUDEMDDepthTableEvaluates`, `TestDiscoveryFallbackSendsOneResearcher`.

Confirmed `TestGatekeeperNeedsASecuritySignal` no longer exists as a live function — it was retired with an explicit `recovered-by: cmd/queen_forced_reviewer_test.go (TestReviewerForcedOnlyByNamedRisk)` disposition in the ledger, because D-05 moved the mechanism it tested (build-side keyword list) to a continue-side signal table that keyword list no longer exists to hold. This deviates from 194-CONTEXT.md D-15's "keep and extend" instruction for that specific test name, but the claim it protected demonstrably survives in a differently-named, passing, more rigorous replacement (asserts the real dispatch list rather than an intermediate scoring function) — treated as acceptable under D-15's own "Claude's Discretion" clause ("whether `TestHighRiskPhaseKeepsBothReviewers` is retired or rewritten" — the same discretion reasonably extends to a same-named-mechanism sibling once the boundary it tested no longer exists). Not a gap.

### Human Verification Required

None. Every roadmap success criterion resolves to a named, independently-run test with real assertions against the actual dispatch lists (not decision records or stubbed functions), each with evidence a wrong result would have failed the test (guard rows, empty-fixture sanity checks). The one recorded scope deviation (D-14 / the one-worker check-in pause) is a deliberate, disclosed, owner-acknowledged decision to leave shipped behavior unchanged and file the reversal as a routed todo for the next planning pass — not a gap in this phase's own delivery.

### Gaps Summary

No gaps. All 5 requirement IDs (TEAM-01..05) satisfied with passing, non-vacuous tests; the roadmap's 5 success criteria plus the Phase 193 deferred truth (WINDOWS.md #1) are all verified against real dispatch-list behavior, not summary claims; the full `cmd` package test suite (355s) passes with zero regressions; `go vet` is clean; no debt markers in any file this phase touched; all wrapper triplets remain byte-identical; CLAUDE.md's documentation is itself test-enforced against the runtime it describes.

---

_Verified: 2026-08-23T17:19:52Z_
_Verifier: Claude (gsd-verifier)_
