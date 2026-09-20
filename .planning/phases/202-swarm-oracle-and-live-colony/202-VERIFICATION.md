---
phase: 202-swarm-oracle-and-live-colony
verified: 2026-09-11T23:30:00Z
status: passed
score: 5/5 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 3/5
  gaps_closed:
    - "Watch shows real active workers, lineages, waves, workspaces, questions, confidence, contradictions, signals, findings, elapsed time, cost, and recovery state from replayable typed events (CR-02: recovery no longer hijacks the live view)."
    - "A multi-round Oracle question is watchable through aether watch (CR-01: a genuinely-in-flight Oracle round now classifies live)."
    - "Every locked decision traces to at least one SYN-202 row and implementation plan (this truth was only listed because the two truths above had failed; it required no independent work and is now resolved along with them)."
  gaps_remaining: []
  regressions: []
deferred:
  - "All three human_verification items deferred by owner on 2026-09-12 during /gsd-verify-work 202: tests 1-2 skipped (owner does not use the second-terminal watch dashboard; wants inline colony visibility), test 3 deferred to later. See 202-UAT.md."
human_verification:
  - test: "Open aether watch during a live build/continue run, then trigger a recovery decision (e.g. force a worker failure) and observe whether the dashboard keeps showing the running build's active workers or switches to a near-empty recovery view."
    expected: "The build/continue episode and its active workers should remain visible, with recovery state layered on top -- not replaced by it."
    why_human: "The CR-02 fix is now proven at the unit level -- TestRecoveryDecisionKeepsTheBuildEpisodeLive drives orchestrateRecovery, the live carriers, and the projection end-to-end, and I additionally reverted the fix and confirmed 3 of its 4 subtests fail exactly as the SUMMARY claims, then restored it and confirmed all 4 pass. What remains outside grep/test reach is the actual on-screen timing and visual experience of a real `aether watch` session during a live run -- an eyes-on check is still the right way to close this out."
  - test: "Start a long-running Oracle round (`aether oracle iterate` or equivalent) and, from a second terminal, run `aether watch` while it is in progress."
    expected: "The dashboard should show the Oracle round live (current question, confidence, contradictions) rather than falling back to a replay-labelled summary."
    why_human: "The CR-01 fix is now proven at the unit level -- TestOracleRoundIsLiveWhileItRuns and TestOracleEpisodeBoundaryIsWiredIntoTheLoop pass, and I independently reverted the boundary call and confirmed the wiring guard fails by name (exact message: \"runOracleLoop does not call openOracleLiveEpisode\"), then restored it and confirmed both tests pass again. TestAbandonedOracleRoundIsNotLive also passes, covering the second half of the gap (a finished earlier build's spawn run no longer wrongly demotes a genuinely live Oracle round). What remains is the real-world visual confirmation of `aether watch` during an actual research run."
  - test: "Look at the rendered `aether watch` live dashboard, Swarm's end-of-investigation comparison card, and Oracle's synthesis document for the Classic Feb-April colony character (caste emoji, ant glyphs, ceremony framing, Queen voice) named in D-04."
    expected: "The screens read as the Classic colony's own voice, not a generic status printout."
    why_human: "Visual/tonal character cannot be verified by grep or automated test; unchanged since the prior verification round -- the gap-closure plans did not touch rendering voice or tone."
---

# Phase 202: Swarm, Oracle, and Live Colony Verification Report (Gap-Closure Re-Verification)

**Phase Goal:** Restore substantive Swarm diagnosis, iterative Oracle research, and a real live colony cockpit driven by typed runtime events.
**Verified:** 2026-09-11
**Status:** human_needed
**Re-verification:** Yes — after gap closure (plans 202-16, 202-17)

## Summary

The prior verification round (`gaps_found`, 3/5) found two blocker-level wiring gaps, both confirmed directly against running code rather than inferred from SUMMARY.md:

1. **CR-02** — a recovery decision fired mid-build/check minted a synthetic `recovery-phase-N` episode, hijacking `aether watch` away from the real running episode and its active workers.
2. **CR-01** — Oracle's research loop never opened a `live.episode.*` boundary, so a genuinely-in-progress research round could never satisfy `resolveWatchMode`'s live classification.

Gap-closure plans 202-16 and 202-17 were executed to close exactly these two gaps. I did not take their SUMMARY.md claims as evidence. I independently:

- Read the actual diffs in `cmd/watch_live.go`, `cmd/live_projection.go`, `cmd/watch_replay.go`, `cmd/live_events.go`, `cmd/recovery_orchestrator.go`, `cmd/oracle_live.go`, `cmd/oracle_loop.go`, `cmd/swarm_cmd.go`, `cmd/swarm_lens.go`, `pkg/codex/dispatch.go`, and `CLAUDE.md`, and confirmed every function, call site, and doc-comment claim made in the two SUMMARYs is genuinely present in the code, matching the two PLAN files' `must_haves` verbatim.
- Ran `go build ./...` and `go vet ./cmd ./pkg/codex` clean.
- Ran every named test cited by both SUMMARYs (14 tests, several with multiple subtests) directly, myself, in my own process — all pass.
- Performed my own independent break-it-to-prove-it verification on both fixes (not trusting the SUMMARYs' own claimed break-it-to-prove-it results): reverted `cmd/recovery_orchestrator.go`'s emission call to the old synthetic-ID form and confirmed `TestRecoveryDecisionKeepsTheBuildEpisodeLive` fails 3 of 4 subtests exactly as claimed; reverted `cmd/oracle_loop.go`'s `runOracleLoop` to drop the `openOracleLiveEpisode` call and confirmed `TestOracleEpisodeBoundaryIsWiredIntoTheLoop` fails by name with the exact message the plan specifies. Restored both fixes afterward and reconfirmed green; working tree left clean.
- Re-ran the previously-VERIFIED truths' own tests (Swarm four-lens diagnosis, checkpointed repair/rollback, durable episode, standing vocabulary, episode index, classic-contract traceability, CLAUDE.md test-citation guard) to confirm no regression.
- Cross-checked all 9 requirement IDs (SYNTH-04, CEC-05, LIVE-01..07) against REQUIREMENTS.md and the code.

Both gaps are now genuinely closed in the codebase, not merely claimed closed. One documentation-bookkeeping issue was found (see Requirements Coverage) and is noted as informational, not a gap.

## Goal Achievement

### Observable Truths (mapped to Roadmap Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A cited mechanism study reconstructs the exact Classic Swarm, Oracle, and live-display control loops and compares them with current issuance, research, event, and renderer paths before selecting the modern synthesis. | ✓ VERIFIED (unchanged) | `TestClassicContractPhase202Cases` re-run, all 3 subtests pass; no gap-closure plan touched `202-CLASSIC-SYNTHESIS.md` or the corpus registry. |
| 2 | Watch shows real active workers, lineages, waves, workspaces, questions, confidence, contradictions, signals, findings, elapsed time, cost, and recovery state from replayable typed events. | ✓ VERIFIED (CR-02 closed) | `currentLiveRecoveryEpisode` (`cmd/live_events.go:363-371`) now resolves recovery onto the real open build/continue carrier before falling back to a synthetic ID; `latestLiveEpisodeID` (`cmd/watch_live.go:243-283`) now selects the most recently started *still-open* episode via the shared `openColonyLiveEpisodeIDs`/`latestStartedLiveEpisodeAmong`, not whichever episode owns the newest single event. `TestRecoveryDecisionKeepsTheBuildEpisodeLive` (4/4 subtests), `TestWatchFollowsTheMostRecentlyStartedOpenEpisode`, `TestClosedEpisodesPickTheSameEpisodeTheReplaySummaryNames`, `TestOneOpenBalanceRule` all pass; break-it-to-prove-it independently confirmed on both fixes. |
| 3 | A stubborn defect runs four genuinely distinct Swarm lenses, compares hypotheses, ranks and checkpoints a repair, verifies or rolls back, preserves strike escalation, and stores one issuance-bound replay-safe episode. | ✓ VERIFIED (unchanged) | `TestFourSwarmLensesProduceDistinctEvidence`, `TestSwarmRepairRollsBackOnFailedVerification`, `TestSwarmRunProducesOneReplaySafeEpisode` re-run, all pass. Swarm's six live-event payload literals now read `events.EpisodeKindSwarm` instead of a hand-typed string (WR-04 closed) -- confirmed by direct grep, not a behaviour change. |
| 4 | A multi-round Oracle question visibly targets uncertainty, reports confidence and contradictions, detects diminishing returns, and produces a source-grounded final synthesis, watchable through aether watch. | ✓ VERIFIED (CR-01 closed) | `openOracleLiveEpisode`/`oracleLiveTerminalStatus` (`cmd/oracle_live.go`) give Oracle the same emit-started/defer-emit-ended boundary pair build/continue already use; `runOracleLoop` (`cmd/oracle_loop.go:934-941`) wraps the renamed `runOracleLoopRounds` in that boundary, opened only when the loaded state has a non-empty `StartedAt`; `stopOracleCompatibility` closes the episode for a controller it just killed. `colonyLiveEpisodeAbandoned`/`oracleLiveEpisodeAbandoned` (`cmd/watch_live.go`) judge Oracle's abandonment from its own durable state and controller PID rather than another lane's spawn-run record, so a finished earlier build no longer wrongly demotes a genuinely live Oracle round. `TestOracleRoundIsLiveWhileItRuns`, `TestOracleManualStopClosesTheLiveEpisode`, `TestOracleEpisodeBoundaryIsWiredIntoTheLoop`, `TestAbandonedOracleRoundIsNotLive` (6/6 subtests), and the amended `TestLiveDashboardShowsTheResearchRound` (now asserts the mode it previously discarded) all pass; break-it-to-prove-it independently confirmed on the wiring guard. |
| 5 | Useful partial Swarm/Oracle work, activity, learning, plan research, and local Dreams remain durable, discoverable, and accurately labelled rather than discarded or called verified. | ✓ VERIFIED (unchanged) | `TestOneStandingVocabularyAcrossSubsystems`, `TestStatusHistoryAndWatchShareOneLineage` re-run, both pass. No gap-closure plan touched `cmd/partial_work_label.go` or `cmd/episode_index.go`. |

**Score:** 5/5 roadmap success criteria verified; 0 present-but-behavior-unverified; 0 failed.

### Gap-Closure Plan Must-Haves (202-16, 202-17)

| Must-have | Status | Evidence |
|-----------|--------|----------|
| `cmd/watch_live.go` contains `latestLiveEpisodeID` selecting the most recently started still-open episode | ✓ VERIFIED | Read in full; matches plan text exactly; `TestWatchFollowsTheMostRecentlyStartedOpenEpisode` passes |
| `cmd/live_events.go` contains `currentLiveRecoveryEpisode` | ✓ VERIFIED | Read in full at lines 363-371; matches plan text exactly |
| `cmd/live_projection.go` contains `colonyLiveBoundaryDelta` (one shared open/close rule used by both the reducer and the selector) | ✓ VERIFIED | `foldColonyLiveEvents` derives its delta from `colonyLiveBoundaryDelta` (line ~407); `openColonyLiveEpisodeIDs` derives its balance from the same function (line ~356); floor-at-zero preserved in the reducer |
| `cmd/live_recovery_episode_test.go` contains `TestRecoveryDecisionKeepsTheBuildEpisodeLive` | ✓ VERIFIED | File exists, test runs, 4/4 subtests pass |
| `cmd/oracle_live.go` contains `openOracleLiveEpisode` | ✓ VERIFIED | Read in full at lines 136-141; matches plan text exactly |
| `cmd/oracle_loop.go` contains `runOracleLoopRounds` | ✓ VERIFIED | Confirmed via grep: exactly one `func runOracleLoopRounds` and exactly one `func runOracleLoop(` |
| `cmd/watch_live.go` contains `colonyLiveEpisodeAbandoned` | ✓ VERIFIED | Read in full at lines 88-93; dispatches on `snapshot.EpisodeKind`, delegates to `oracleLiveEpisodeAbandoned` for Oracle, `colonyLiveEpisodeRunHasTerminated` unchanged for every other lane |
| `cmd/oracle_live_test.go` contains `TestOracleRoundIsLiveWhileItRuns` | ✓ VERIFIED | Test exists and passes |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/live_projection.go` | Shared boundary-balance rule for reducer and selector | ✓ VERIFIED | `colonyLiveBoundaryDelta`, `openColonyLiveEpisodeIDs` added; `foldColonyLiveEvents` refactored to consume the shared rule; floor-at-zero behaviour byte-for-byte preserved |
| `cmd/watch_live.go` | Live/replay/idle mode resolution, now episode-kind-aware for abandonment | ✓ VERIFIED | `resolveWatchMode`, `latestLiveEpisodeID`, `colonyLiveEpisodeAbandoned`, `oracleLiveEpisodeAbandoned` all present and correct; previously ⚠️ HOLLOW for Oracle/recovery cases, now fully wired for both |
| `cmd/watch_replay.go` | Shared "latest started episode" selection | ✓ VERIFIED | `latestStartedLiveEpisodeAmong` extracted; `mostRecentlyStartedLiveEpisode` calls it; exactly one implementation in the package |
| `cmd/live_events.go` | Recovery's owning-episode resolution | ✓ VERIFIED | `currentLiveRecoveryEpisode` added, consumed by `cmd/recovery_orchestrator.go` |
| `cmd/recovery_orchestrator.go` | Recovery emits on the owning episode | ✓ VERIFIED | Single emission call site (line 218-219) resolves through `currentLiveRecoveryEpisode`, no longer always mints `recovery-phase-N` |
| `cmd/oracle_live.go` | Oracle's episode boundary open/close pair | ✓ VERIFIED | `openOracleLiveEpisode`, `oracleLiveTerminalStatus` added; previously ⚠️ ORPHANED for watch's live-mode purposes, now genuinely wired |
| `cmd/oracle_loop.go` | The research loop wrapped in its own live episode | ✓ VERIFIED | `runOracleLoop` wraps `runOracleLoopRounds` in the boundary, guarded on non-empty `StartedAt`; `stopOracleCompatibility` closes the episode for a killed controller |
| `cmd/swarm_cmd.go`, `cmd/swarm_lens.go` | Episode kind from shared constant | ✓ VERIFIED | 6/6 literals now use `events.EpisodeKindSwarm`; 0 hand-typed string assignments remain |
| `pkg/codex/dispatch.go` | `ParentWorkerID` doc comment states no production dispatch sets it | ✓ VERIFIED | Comment present, states this plainly |
| `CLAUDE.md` | Live Colony section names tests for both fixes | ✓ VERIFIED | Both new claims present, each naming the exact test symbols, which resolve in source; `TestEveryLiveColonyClaimInCLAUDEMDNamesALiveTest` passes |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `cmd/recovery_orchestrator.go` | `cmd/live_events.go` | recovery resolves the owning open episode instead of minting its own | ✓ WIRED (was ⚠️ WIRED BUT MISROUTED) | `currentLiveRecoveryEpisode` call confirmed at the single emission site; break-it-to-prove-it confirms the fix is load-bearing |
| `cmd/watch_live.go` | `cmd/watch_replay.go` | live and replay share one "latest started episode" selection | ✓ WIRED | `latestStartedLiveEpisodeAmong` is the single implementation, called from both `latestLiveEpisodeID` and `mostRecentlyStartedLiveEpisode` |
| `cmd/oracle_loop.go` | `cmd/oracle_live.go` | the round-based run opens and closes its own live episode | ✓ WIRED (was ⚠️ WIRED BUT INCOMPLETE) | `runOracleLoop` calls `openOracleLiveEpisode` and defers the close; `TestOracleEpisodeBoundaryIsWiredIntoTheLoop` (an AST-walk guard, not a text search) proves the call is genuinely present, and fails by name when removed |
| `cmd/watch_live.go` | `cmd/oracle_loop.go` | an Oracle episode's liveness is judged from Oracle's own durable state and controller process | ✓ WIRED (new) | `oracleLiveEpisodeAbandoned` reads Oracle's state file and controller PID directly; `TestWatchResolvesThreeBranchesFromEvidenceAlone`'s 5 subtests (build/continue/plan/Swarm) pass unchanged, confirming the dispatch did not disturb any other lane |

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|----------------|--------------|--------|----------|
| SYNTH-04 | 202-01 | Swarm/Oracle/live mechanism study | ✓ SATISFIED (code) / ⚠️ checkbox stale | Code unchanged since original verification's SATISFIED finding; `TestClassicContractPhase202Cases` passes. REQUIREMENTS.md's checkbox is unchecked (`[ ]`) -- see note below. |
| CEC-05 | 202-02, 202-03, 202-06, 202-11, 202-15, 202-16, 202-17 | Typed live events across planning/build/Swarm/Oracle/recovery/verification | ✓ SATISFIED | Both gap-closure plans declared `CEC-05` in frontmatter and closed the deeper wiring deliverable; checkbox is `[x]` in REQUIREMENTS.md |
| LIVE-01 | 202-02, 202-04 | One typed event bridge | ✓ SATISFIED (code) / ⚠️ checkbox stale | Code unchanged since original verification's SATISFIED finding. Checkbox unchecked. |
| LIVE-02 | 202-02 -> 202-16 | Real live cockpit | ✓ SATISFIED | 202-16 declared `LIVE-02` in frontmatter and closed CR-02; checkbox is `[x]` |
| LIVE-03 | 202-05 | Substantive Swarm diagnosis | ✓ SATISFIED (code) / ⚠️ checkbox stale | Code unchanged; `TestFourSwarmLensesProduceDistinctEvidence` re-confirmed passing. Checkbox unchecked. |
| LIVE-04 | 202-07 | Safe Swarm repair | ✓ SATISFIED (code) / ⚠️ checkbox stale | Code unchanged; `TestSwarmRepairRollsBackOnFailedVerification` re-confirmed passing. Checkbox unchecked. |
| LIVE-05 | 202-10, 202-13, 202-14 | Durable Swarm outcome | ✓ SATISFIED (code) / ⚠️ checkbox stale | Code unchanged; `TestSwarmRunProducesOneReplaySafeEpisode` re-confirmed passing. Checkbox unchecked. |
| LIVE-06 | 202-08 -> 202-17 | Iterative Oracle | ✓ SATISFIED | 202-17 declared `LIVE-06` in frontmatter and closed CR-01's watchability half; checkbox is `[x]` |
| LIVE-07 | 202-12, 202-13, 202-14 | Durable Oracle synthesis | ✓ SATISFIED (code) / ⚠️ checkbox stale | Code unchanged; `TestSynthesisLeadsWithTheRecommendation` re-confirmed passing. Checkbox unchecked. |

**Documentation-bookkeeping note (not a code gap):** `.planning/REQUIREMENTS.md`'s checkboxes for `SYNTH-04`, `LIVE-01`, `LIVE-03`, `LIVE-04`, `LIVE-05`, `LIVE-07` currently read `[ ]` (unchecked). Git history shows these were correctly reverted from `[x]` to `[ ]` by commit `88fa44ea` ("revert premature Complete requirements after gaps found") when the initial verification found `gaps_found`. The two gap-closure plans (202-16, 202-17) each only declared the specific requirement IDs their own must-haves addressed (`LIVE-02`/`CEC-05` and `CEC-05`/`LIVE-06`), so their `requirements.mark-complete` step correctly left the other six IDs untouched -- but nothing in this phase's process re-checked them once the phase as a whole returned to a fully-satisfied state. This is a leftover bookkeeping gap from the revert, not a code defect: the code satisfies all six exactly as the original verification found (re-confirmed above), and no gap-closure plan touched the underlying implementation for any of them. Recommend the six boxes be restored to `[x]` as part of closing this phase.

No orphaned requirements — every ID in the phase's requirement list is claimed by at least one plan's frontmatter and cross-referenced in REQUIREMENTS.md's Phase 202 row.

### Anti-Patterns Found

No `TODO`/`FIXME`/`XXX`/`TBD` debt markers found in any file touched by plans 202-16 or 202-17.

Findings from the phase's own code-review pass (`202-REVIEW.md`, gap-closure round, `3dfc9a79`) — 0 critical, 2 warnings, 1 info, all non-blocking:

| File | Finding | Severity | Impact |
|------|---------|----------|--------|
| `cmd/live_events.go:363-371` | `currentLiveRecoveryEpisode` only knows the build and continue carriers; a future third lane (e.g. Swarm/Oracle recovery) could silently regress into CR-02's exact bug if `orchestrateRecovery` is ever called from a new call site (WR-01) | ⚠️ Warning | Latent-fragility note; no live regression today, since `orchestrateRecovery`'s only three call sites are build/continue finalize paths |
| `cmd/oracle_loop.go:567-608` vs `934-942` | A killed background Oracle run can theoretically emit two `episode.ended` events for the same episode (a SIGTERM race between `stopOracleCompatibility` and the loop's own `context.Done()` branch) (WR-02) | ⚠️ Warning | Harmless today only because `foldColonyLiveEvents`'s balance floor never goes negative; worth a guard before that floor is ever changed |
| `cmd/live_projection.go:338-370` | `openColonyLiveEpisodeIDs` duplicates the schema-version-skip predicate inline rather than sharing it with `foldColonyLiveEvents` (IN-01) | ℹ️ Info | Proven consistent today by `TestOneOpenBalanceRule`; a future edit to the skip rule in one place could miss the other |

I independently verified this review's own summary claim ("no blockers, both fixes proven end-to-end") is accurate by running the exact named tests and my own break-it-to-prove-it checks above, rather than accepting the review document's status field at face value.

### Build/Test Verification

- `go build ./...` — clean (independently re-run)
- `go vet ./cmd ./pkg/codex` — clean (independently re-run)
- All 14+ named must-have tests from both gap-closure plans — **pass**, re-run directly by the verifier: `TestRecoveryDecisionKeepsTheBuildEpisodeLive` (4 subtests), `TestWatchFollowsTheMostRecentlyStartedOpenEpisode`, `TestClosedEpisodesPickTheSameEpisodeTheReplaySummaryNames`, `TestOneOpenBalanceRule`, `TestOracleRoundIsLiveWhileItRuns`, `TestOracleManualStopClosesTheLiveEpisode`, `TestOracleEpisodeBoundaryIsWiredIntoTheLoop`, `TestAbandonedOracleRoundIsNotLive` (6 subtests), `TestLiveDashboardShowsTheResearchRound`, `TestEveryLiveColonyClaimInCLAUDEMDNamesALiveTest`
- Regression set re-run directly by the verifier, all pass: `TestWatchResolvesThreeBranchesFromEvidenceAlone` (5 subtests), `TestWatchIsReadOnlyInEveryBranch` (3 branches), `TestSwarmInvestigationWaveReachesTheLiveWatchScreen`, `TestLiveProjectionIsPureReplay`, `TestWatchReplaysTheMostRecentEpisode`, `TestReplaySummaryOfInterruptedEpisodeSaysInterrupted`, `TestLiveProjectionResumesWithoutDoubleCounting`, `TestWatchIdle199*` (4 subtests), `TestEveryLifecycleLaneEmitsLiveEvents` (7 subtests), `TestEveryLiveEventGoesThroughOneBoundary`, `TestFailedLiveEmitNeverChangesLaneOutcome` (4 subtests), `TestFourSwarmLensesProduceDistinctEvidence`, `TestSwarmRunProducesOneReplaySafeEpisode`, `TestSwarmRepairRollsBackOnFailedVerification`, `TestOracleRoundsReachTheLiveStream`, `TestOraclePresetLabelsMatchPlanningVocabulary`, `TestSynthesisLeadsWithTheRecommendation`, `TestLiveDashboardShowsCurrentWaveInDepth`, `TestOneStandingVocabularyAcrossSubsystems`, `TestStatusHistoryAndWatchShareOneLineage`, `TestClassicContractPhase202Cases` (3 subtests)
- **Independent break-it-to-prove-it (not from SUMMARY.md, performed by the verifier in this session):**
  - Reverted `cmd/recovery_orchestrator.go`'s emission call to the old `recovery-phase-N` synthetic-ID form -> `TestRecoveryDecisionKeepsTheBuildEpisodeLive` failed on 3 of 4 subtests with exactly the errors the SUMMARY reports (`snapshot.RecoveryState = "", want ... "retry"`; 2 distinct episode IDs found instead of 1). Restored the fix -> all 4 subtests pass again. Working tree confirmed clean afterward.
  - Removed the `openOracleLiveEpisode`/`defer closeEpisode(...)` wrapper from `cmd/oracle_loop.go`'s `runOracleLoop` -> `TestOracleEpisodeBoundaryIsWiredIntoTheLoop` failed by name with the exact message the plan specifies ("runOracleLoop does not call openOracleLiveEpisode -- the live episode boundary is not wired into the loop"). Restored the fix -> test passes again. Working tree confirmed clean afterward.
- `TestCurrentVocabulary199` and `TestPhase199GateReceipt` — independently re-run and confirmed **still fail**, with the same pre-existing symptoms recorded before this phase's gap-closure work began (a vocabulary-inventory count mismatch referencing `199-PATTERNS.md`'s `legacy_pause`/`legacy_resume` entries, and a stale ownership-fingerprint receipt). Neither touches any file modified by plans 202-16/202-17; not a regression, not caused by this phase's code.
- Full unscoped `go test ./cmd` was not run in full (documented ~20-minute machine ceiling); scoped `-run` filters covering every named test cited by both gap-closure SUMMARYs, the full previously-passing regression set, and the two known pre-existing failures were run directly by the verifier instead.

## Gaps Summary

No gaps remain. Both blocker-level findings from the prior verification round (CR-01, CR-02) are genuinely fixed in the codebase — verified by direct code reading, independent test execution, and independent break-it-to-prove-it reversion, not by trusting the gap-closure plans' own SUMMARY.md claims. The phase's own code-review pass on this gap-closure round found zero blockers and only two latent-fragility warnings plus one duplication-risk info note, none of which affect the phase goal's achievement today.

The phase does not resolve to `passed` because three human-verification items remain, all carried forward from the prior round: two are eyes-on confirmations of the now-code-level-proven CR-01/CR-02 fixes during a real `aether watch` session (recommended, not required, given the strength of the unit-level proof — but genuinely outside what grep/test can see, namely on-screen timing and visual presentation), and one (D-04's Classic colony tonal character) is unchanged from before since no gap-closure plan touched rendering voice.

One documentation-bookkeeping issue was found: six requirement checkboxes in REQUIREMENTS.md (SYNTH-04, LIVE-01, LIVE-03, LIVE-04, LIVE-05, LIVE-07) remain unchecked from the prior gaps_found revert, even though the underlying code was never part of the gap and is independently re-confirmed satisfied here. This is not a code gap — it should be corrected as part of closing this phase.

---

_Verified: 2026-09-11_
_Verifier: Claude (gsd-verifier)_
