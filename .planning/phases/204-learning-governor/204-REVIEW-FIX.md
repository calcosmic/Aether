---
phase: 204-learning-governor
fixed_at: 2026-09-15T02:11:15Z
review_path: .planning/phases/204-learning-governor/204-REVIEW.md
iteration: 1
findings_in_scope: 7
fixed: 6
skipped: 1
status: partial
---

# Phase 204: Code Review Fix Report

**Fixed at:** 2026-09-15T02:11:15Z
**Source review:** .planning/phases/204-learning-governor/204-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 7 (CR-01..CR-04, WR-01..WR-03)
- Fixed: 6
- Recorded, not functionally fixed (see below): 1 (CR-01)

CR-01 is counted as "skipped" in the frontmatter's schema sense (no functional
fix was applied), but it was fully investigated and honestly recorded rather
than left as a stale claim — see its section below for the full account of
why no fix was attempted.

## Fixed Issues

### CR-01: The automatic hypothesis-to-validated promoter can never promote anything in production

**Files modified:** `cmd/learning_validator.go`, `cmd/learning_validator_test.go`, `.planning/WINDOWS.md`, `CLAUDE.md`, `.claude/rules/aether-colony.md`, `.aether/rules/aether-colony.md`, `cmd/testdata/fixture-bank/v1/bank.json`
**Commit:** `6bf03852`
**Status:** **recorded, not fixed** — this is the one finding that received no functional code fix, by design, per an explicit instruction in this task's fix rules to never fabricate a connection that no real production writer produces.

**What I found:** the review's diagnosis was confirmed and found to be even more complete than described. I traced every non-test call site of `recordGuidanceApplicationState` in `cmd/` and confirmed:
1. Every real writer keys a `GuidanceApplications` record by an Instinct's own ID (`cmd/instinct_application.go:69,75`), never by a `learn.Entry`'s ID. `learn.Entry.ParentID` exists only for hypothesis-revision lineage (one `learn.Entry` to another, per `pkg/learn/hypothesis_test.go`), never entry-to-instinct. No production writer of a `learn.Entry` (`cmd/learning_cmds.go`'s hand-run `learning-propose`, `cmd/codex_continue_finalize.go`'s worker-lesson capture, or `cmd/oracle_promote.go`'s `promoteOracleFindingAsLearning`) ever records a lineage link back to an Instinct.
2. Independently of (1), **no non-test call site anywhere in `cmd/` ever writes the `guidanceApplicationStateHelpful`/`…Neutral`/`…Harmful` terminal states at all.** The "helpful" signal QUEEN promotion actually reads (`recruitmentCreditOutcomeHelpful`) lives in a different list (`recruitmentCreditFile.Entries`) than the one `learningEntryHasHelpfulApplication` reads (`recruitmentCreditFile.GuidanceApplications`), and the two are never bridged. So `promoteHelpfulHypotheses` cannot promote anything today for two independent, compounding reasons, not just the ID mismatch the review named.

**What I did, since no honest identifier bridge exists within reach of this fix pass:**
1. Rewrote `promoteHelpfulHypotheses`'s and `learningEntryHasHelpfulApplication`'s doc comments in `cmd/learning_validator.go` to state the limitation and both root causes in full, with the exact grep evidence.
2. Reopened WINDOWS.md entry 44 (status `fixed` → `open`) in **both** the markdown table row and the JSON ledger block, with byte-identical description text, naming the ID-space mismatch and the missing Helpful/Neutral/Harmful writer as the reason.
3. Checked REQUIREMENTS.md: no checkbox line cites the promoter's own tests (`TestHelpfulHypothesisIsPromotedAutomatically` etc. — none appear anywhere in REQUIREMENTS.md). LEARN-01's claim ("hypotheses... are never labelled or injected as verified") is about the render/injection layer, which is genuinely unaffected and still true, so nothing needed unticking there.
4. Corrected CLAUDE.md's "Learning Governor" section: the paragraph claiming "a lesson recorded only as a guess is now promoted to genuinely verified automatically... once the program's own records show it truly helped" was false as written. Rewrote it to say the gating rule is real and tested, but the promotion itself cannot happen in the running program yet, citing the new honest test. Mirrored the same correction byte-for-byte into `.claude/rules/aether-colony.md` and `.aether/rules/aether-colony.md` (verified identical with `diff`).
5. Fixed the false-certificate test: `cmd/learning_validator_test.go` no longer seeds a `GuidanceApplications` record using a hand-typed `learn.Entry` ID as the `guidanceID`. Replaced the positive-path tests (`TestHelpfulHypothesisIsPromotedAutomatically`, `TestPromotionIsIdempotent`) with `TestHypothesisPromotionNeverCrossesTheIdentifierGap` (drives a genuinely helpful guidance record through the real writer under an Instinct-shaped ID, and a genuine hypothesis with its own store-assigned ID, and asserts NO promotion occurs) and `TestPromotionPassIsIdempotentAndLeavesAlreadyValidatedEntriesAlone` (an honest idempotency test that doesn't fabricate the impossible scenario). The remaining internal-logic-only tests (Tests 2–5, 7) were re-labelled in their doc comments as exercising the internal gating rule under a hypothetical future identifier pairing, never claiming production reachability.
6. Regenerated the fixture bank (`go test ./cmd -run '^TestSeededBankUpdate$' -update-bank`): entry 44's old "FIXED by 204-16..." fixture (`fixture-cd29310f9456`) disappeared as expected (only `status: fixed` entries become regression fixtures), and its guard (`TestHypothesisPromoterIsReachedFromBothCheckLanes`) was reported orphaned by `TestSeededBankUpdate` — exactly the expected, documented behavior per this task's fix rules. `TestSeededBankIsReproducible`, `TestEveryFixtureNamesItsGuardOrIsCountedUnguarded`, and `TestSeedBankUnguardedFloorIsTheRealCount` all still pass against the regenerated bank.

**Verification:** `TestHypothesisPromotionNeverCrossesTheIdentifierGap` was watched FAILING (mutated `learningEntryHasHelpfulApplication` to `return true` unconditionally, confirming the test genuinely catches a false-positive promotion) and restored. `TestCLAUDEMDLearningGovernorClaimsCiteLiveTests` and `TestCLAUDEMDLearningGovernorSectionIsPlainEnglish` pass against the corrected CLAUDE.md section. `TestCurrentVocabulary199`'s `d17-status-primary-generated-parity` subtest confirms the two rules mirrors stayed byte-identical.

---

### CR-02: Recovery-episode collision silently drops durable close records on the primary interactive build-finalize path

**Files modified:** `cmd/recovery_orchestrator.go`, `cmd/recovery_orchestrator_test.go`, `cmd/live_recovery_episode_test.go`
**Commit:** `53ce0398`
**Applied fix:** `orchestrateRecovery`'s standalone-episode fallback (`currentLiveRecoveryEpisode`'s phase-only ID, `recovery-phase-%d`) is now disambiguated per call via a new `standaloneRecoveryEpisodeID(base, workerName, taskID)` helper, using the worker name and task ID already present on `RecoveryContext`. This is scoped narrowly to `orchestrateRecovery`'s own open/changed/close triple — `currentLiveRecoveryEpisode` itself is unchanged, so its other three callers (`forced_reviewer_waiver.go`, `handoff_decisions_cmd.go`, `rollback.go`, none of which open or close an episode) are unaffected. Updated `TestRecoveryDecisionKeepsTheBuildEpisodeLive`'s "no live episode open" subtest to expect the new discriminated ID instead of the bare fallback.

**Verification:** New test `TestBuildFinalize_MultipleFailedDispatches_EachGetsADistinctDurableEpisode` drives the real production chain (`runCodexBuildFinalize` → `buildExternalBuildRecoveryInstructions` → `orchestrateRecovery`, once per failed dispatch) with two failed dispatches and no open build/continue episode, asserting two independent durable open+closed records exist. Watched FAILING against the reverted fix (only one `recovery-phase-1` closed record existed, carrying only the first dispatch's outcome — exactly CR-02's described collision) and restored passing. Full recovery/episode-ledger test suite re-run clean.

---

### CR-03: Swarm's delegate/external finalize lane never records a durable episode at all

**Files modified:** `cmd/swarm_cmd.go`, `cmd/swarm_cmd_test.go`
**Commit:** `08ccd675`
**Applied fix:** `runSwarmFinalize` now opens `emitColonyLiveEpisodeStarted(swarmID, events.EpisodeKindSwarm)` immediately after `initializeSwarmRun` succeeds, accumulates usage via the existing `addSwarmRunsUsage`/`episodeCloseFacts` helpers, and closes with `emitColonyLiveOutcomeRecorded`/`emitColonyLiveEpisodeEndedEventOnly` reading the same `runStatus` variable `finishRuntimeSpawnRun`'s own deferred call reads — the identical shape `runSwarmDestroy` already uses for swarm's native lane.

**Verification:** New test `TestSwarmFinalizeLaneOpensAndClosesADurableEpisode` drives `runSwarmFinalize` directly and asserts a durable open record and a closed record with `TerminalResult: "completed"` for the swarm episode. Watched FAILING against the reverted fix ("delegate swarm-finalize lane wrote no durable open record... durable records: []") and restored passing. I deliberately did not extend the shared `liveLaneEntryPoints` map (which asserts one representative lane per declared episode kind, not "every lane") — instead I followed this codebase's own established pattern for proving a delegate lane's coverage independently of a native lane's (`TestCheckEpisodeRecordsItsOwnFacts` + `TestDelegateCheckEpisodeRecordsItsOwnFacts`), adding a standalone test alongside the existing `TestSwarmLaneOpensAndClosesADurableEpisode`.

---

### CR-04: The automatic improvement pass's repeated-intervention trigger performs unattended `git checkout` on the live tree

**Files modified:** `cmd/source_proposal.go`, `cmd/source_proposal_test.go`
**Commit:** `8b4d93a6`
**Applied fix:** `proposeSourceImprovement` no longer runs `git checkout` in `root` (the live process working directory) at all. Branch creation and the change-set commit now happen entirely inside a new, temporary, isolated `git worktree` (`sourceProposalIsolatedWorktree`, created via `os.MkdirTemp` outside the repo tree), so the live checkout's branch is never switched, whether the function succeeds or fails partway through. `sourceProposalApplyChangeSet` needed no changes — it already operated generically on whatever root it was given. The isolated worktree checkout is removed after use (`git worktree remove --force`); its branch is never deleted by that cleanup, since a source proposal's branch is the whole point and must survive for a human to review (unlike a build worker's throwaway worktree branch).

**Verification:** Two new tests. `TestSourceProposalFailureNeverTouchesTheLiveCheckout` forces a mid-pipeline failure (an absolute-path change-set refusal, which fires after the isolated worktree already exists) and asserts the live checkout's branch, full `git status --porcelain` output, HEAD commit, and worktree count are all byte-for-byte unchanged, and that no proposal branch survives the failure. `TestSourceProposalNeverChecksOutInTheLiveRepository` is a structural test asserting the literal string `"checkout"` no longer appears anywhere in `cmd/source_proposal.go` — watched FAILING against the pre-fix file (checked out at commit `3e5c2837`, the last commit to touch this file before this fix) and passing against the fixed file. All 11 pre-existing tests in `cmd/source_proposal_test.go` still pass unmodified, including `TestProposalCreatesABranchAndNothingElse`'s existing assertion that the original branch stays checked out.

Note on the literal "force the restore step to fail" instruction: the new design has no restore step left to sabotage (that is the point — the failure class is eliminated structurally, not retried), so I could not construct a test that fails specifically at "the final `git checkout` back to main." Instead I proved the stronger, more general property: no failure anywhere in the pipeline can touch the live checkout, plus the structural absence of any `checkout` call against `root`.

---

### WR-01: Refused improvement candidates are re-compared and re-reported on every single check, forever

**Files modified:** `cmd/improvement_pass.go`, `cmd/improvement_pass_test.go`
**Commit:** `8a9e567d`
**Applied fix:** Added a small, dedicated durable store (`shadow/refusals.json`, `improvementRefusalRecord`/`improvementRefusalFile`) recording a refused declaration's own `ContentDigest` (the same digest `declareShadowCandidate` already computes). `processNewImprovementCandidate` now skips comparison and reporting entirely when a matching refusal marker already exists for the candidate's current declaration, and clears the marker the moment that candidate id is genuinely admitted. Since `declareShadowCandidate` itself refuses a genuine content edit under the same ID ("a stored candidate is never edited"), the digest check is defense-in-depth rather than exercisable via the normal declare path today, but it is the correct, future-proof key.

**Verification:** New test `TestRefusedCandidateIsNotReCompareOrReReportedOnANextCheck` declares a candidate guaranteed to be refused, runs the pass three times, and asserts the first run reports one comparison/refusal/event while the second and third runs report zero of each. Watched FAILING against the reverted fix (second pass still reported 1 comparison) and restored passing.

---

### WR-03: The shadow classifier can never credit a candidate for addressing a fixture whose Title/Invariant has no words of 5+ letters

**Files modified:** `cmd/shadow_cmds.go`, `cmd/shadow_grader_test.go`
**Commit:** `1aba2fbb`
**Applied fix:** `shadowFixtureSubjectWords` now falls back to every non-stop-word (regardless of length) when the stricter length-and-stopword-filtered pass finds zero words, so a fixture whose Title/Invariant happen to consist entirely of short/common words is never permanently unaddressable. The stricter pass is still preferred whenever it finds anything, so this is a pure safety net for the degenerate case.

**Verification:** New test `TestTerseFixtureStillHasAQualifyingSubjectWord` builds a fixture whose Title/Invariant are verified (in the test itself) to consist entirely of sub-5-letter or stop words, and asserts a non-empty fallback word list that `shadowTextNamesFixtureSubject` can actually match against. Watched FAILING against the reverted fix and restored passing. Also added the review's suggested startup/test-time check, `TestEveryBankFixtureHasAQualifyingSubjectWord`, which iterates the real committed fixture bank and asserts every fixture has at least one qualifying subject word — currently passing against all 51 committed fixtures.

## Warnings Recorded as Design Choices

### WR-02: Canary completion/rollback decisions use the current phase's own free-check pass/fail as a proxy for the canary's own health

**Files modified:** `cmd/improvement_pass.go`
**Commit:** `f43ed451`
**Status:** documented, not changed — per this task's explicit instruction, this is a design choice, and an honest doc comment naming the proxy is the accepted resolution. Added a doc comment to `improvementPassPhaseGatesPassed` and `processRunningImprovementCanary` explaining exactly why `gatesPassed` is a whole-phase, not scope-specific, signal, and that it will almost always read "passed" regardless of what a given canary's own scope was actually implicated by — mirroring `shadowEvaluator`'s own placeholder-history doc comment style. No behavior changed; purely documentation.

## Skipped Issues

None in the strict "attempted and gave up" sense — CR-01 above is the one finding where the correct outcome was to record the gap honestly rather than apply a functional fix, per explicit instruction in this task's fix rules to never fabricate an identifier connection no real production writer produces.

## Verification Summary

All work was done directly on branch `oracle-reinstate` in the main checkout (per this task's explicit instruction — no worktree isolation for this run, since a full-suite gate was already running in a separate detached worktree concurrently). Every fix was verified with:
- `go build ./cmd/aether` — clean after every change.
- `go vet ./cmd/...` — clean after every change.
- `gofmt -l` on every touched file — no diffs (three pre-existing gofmt-flagged files elsewhere in `cmd/` were not touched by this session and are unrelated).
- Scoped `go test ./cmd -run '^(...)$'` for every touched area, never the full suite.
- For every Critical finding: the locking test was watched FAILING with the fix reverted (recorded above per finding) and PASSING after restoring the fix.
- Register consistency after `.planning/WINDOWS.md` edits: `TestSeededBankUpdate -update-bank` regenerated the fixture bank; `TestSeededBankIsReproducible`, `TestEveryFixtureNamesItsGuardOrIsCountedUnguarded`, `TestSeedBankUnguardedFloorIsTheRealCount`, `TestCLAUDEMDLearningGovernorClaimsCiteLiveTests`, and `TestCLAUDEMDLearningGovernorSectionIsPlainEnglish` all pass.
- `TestCurrentVocabulary199` run in full: only its `tracked-occurrences-are-exhaustively-classified` subtest is red (pre-existing, listed as known-red in this task's fix rules, unrelated to any file this session touched); every other subtest, including the `.claude`/`.aether` rules-mirror parity subtest, is green.

Seven commits were made, one per finding (six `fix(204):`, one `docs(204):` for the WR-02 documentation-only change), in the order: CR-01, CR-02, CR-03, CR-04, WR-02, WR-01, WR-03.

---

_Fixed: 2026-09-15T02:11:15Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
