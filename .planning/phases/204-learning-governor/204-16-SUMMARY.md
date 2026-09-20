---
phase: 204-learning-governor
plan: 16
subsystem: learning-governor
tags: [source-proposal, canary, learning-validator, requirements-truth, windows-ledger, go]

# Dependency graph
requires:
  - phase: 204-learning-governor
    provides: "runAutomaticImprovementPass and the real shadow grader (204-12), the closed episodeInterventionKind vocabulary and swarm/recovery episode boundaries (204-13), the seedBankUnguardedFloor=30 two-sided ratchet (204-14), the nine writerless episode fields and the live two-figure report (204-15)"
provides:
  - "triggerRepeatedInterventionProposal (cmd/improvement_pass.go) -- proposeSourceImprovement's first real production caller, gated on 3+ distinct episodes carrying the same declared intervention kind -- closes SC5d"
  - "promoteHelpfulHypotheses (cmd/learning_validator.go) -- the automatic hypothesis-to-validated promoter WINDOWS entry 44 named, gated on a corroborated helpful guidance-application record -- closes WINDOWS entry 44"
  - "A truthful REQUIREMENTS.md: SYNTH-06 and LEARN-01/02/03/04/06/07/08 ticked on named passing tests; LEARN-05 left unticked with the real remaining number (30/47 unguarded)"
  - "A truthful WINDOWS.md: entries 38/41/44/45 closed, entries 40/42 narrowed back to open with their remaining limits named, entry 43 unchanged, table and JSON ledger in agreement on all 46 entries"
  - "CLAUDE.md's and .claude/rules/aether-colony.md's Learning Governor sections corrected to cite the tests that now prove the automatic grader/canary pass/hypothesis promotion/source-proposal trigger, plus the real slash-command counts (64/64)"
affects: [204-VERIFICATION.md SC5d, WINDOWS.md entries 38 and 40-45, REQUIREMENTS.md SYNTH-06 and LEARN-01..08]

# Actuals (#2632)
actuals:
  tokens: 25627
  tasks: 3
  commits: 4

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Deterministic candidate identity derived from the repeated category alone (sourceProposalRepeatedInterventionCandidateID), never from the evidence list or a timestamp, so proposeSourceImprovement's existing replay rule collapses a second run over unchanged evidence to no second branch."
    - "Proposal file directory chosen for hub-sync AND git-tracking safety together: .aether/reviews-archive/proposals/ is excluded from every aether publish/install hub sync (an existing hubExcludeDirs entry, confirmed by reading listFilesRecursiveWithExclusion/pathHasExcludedComponent) and is NOT listed in .gitignore, unlike the plan's originally-suggested .aether/proposals/ (unprotected, would leak into the hub) or .aether/data/proposals/ (gitignored, git add would silently refuse to stage it)."
    - "promoteHelpfulHypotheses selects candidates entirely through learningUnverifiedEntries (the one shared predicate TestOneLearningStatusVocabulary polices) rather than a second direct comparison against learn.StatusHypothesis -- ColonyStore.loadEntries already normalizes a genuinely empty on-disk status to StatusHypothesis before any reader sees it, so there is no distinguishable 'truly empty legacy' case left to exclude at this layer."

key-files:
  created:
    - cmd/learning_validator.go
    - cmd/learning_validator_test.go
  modified:
    - cmd/source_proposal.go
    - cmd/source_proposal_test.go
    - cmd/improvement_pass.go
    - cmd/improvement_pass_test.go
    - cmd/codex_visuals.go
    - cmd/consolidation_lifecycle.go
    - .planning/REQUIREMENTS.md
    - .planning/WINDOWS.md
    - CLAUDE.md
    - .claude/rules/aether-colony.md
    - .aether/rules/aether-colony.md

key-decisions:
  - "Proposal directory: .aether/reviews-archive/proposals/, not the plan's own suggested .aether/proposals/ -- see tech-stack.patterns above and the 'Directory safety check' section below for the full evidence."
  - "runAutomaticImprovementPass's early-return ('nothing declared and nothing running: return summary') was removed and replaced with unconditional fall-through to the new trigger step, because the repeated-intervention signal (the durable episode ledger) is independent of the shadow-candidate/canary stores that early return was guarding -- TestCheckWithNoDeclaredCandidateCostsNothing still passes unchanged since the trigger is a pure read that writes nothing when no category has reached the threshold."
  - "promoteHelpfulHypotheses's phaseID parameter is accepted but not used to scope the helpful-application lookup: a hypothesis's proof of helpfulness is not itself scoped to the phase that happens to be ending when the automatic pass runs -- a lesson that helped once, on any phase, has stopped being merely a guess. The parameter exists for call-site signature parity with runAutomaticImprovementPass and to leave room for phase-scoping if a future plan finds a reason to add it."
  - "WINDOWS entry 40 was reopened from 'fixed' (incorrectly set by plan 204-12) back to 'open', per the plan's own explicit instruction and D-11: the grader is now genuinely real, but the entry's own second sentence (comparison never covers a code-valued candidate) is still true today, and marking an entry fixed while its own described limit still applies is exactly the dishonesty D-11 exists to prevent."
  - "Did not run gsd-tools requirements mark-complete for this plan: Task 3 is itself a manual, evidence-based edit of REQUIREMENTS.md's checkbox state (ticking SYNTH-06 and LEARN-01/02/03/04/06/07/08, explicitly leaving LEARN-05 unticked) -- running the generic mark-complete tool afterward over this plan's declared `requirements: [LEARN-08, LEARN-02, LEARN-05, LEARN-06, LEARN-07]` would either duplicate the already-correct edit or, worse, blindly tick LEARN-05 against this plan's own explicit finding that it remains partial."

requirements-completed: [LEARN-02, LEARN-06, LEARN-07, LEARN-08]
# LEARN-05 is deliberately NOT included: Task 3's own evidence-based review
# found it still partial (30 of 47 fixture-bank incidents remain unguarded)
# and REQUIREMENTS.md itself was left unticked for LEARN-05, per D-11. See
# "Requirements Truth Table" below for every one of the nine reviewed IDs,
# including the six ticked outside this plan's own declared requirements
# list (SYNTH-06, LEARN-01, LEARN-03, LEARN-04) which Task 3 also reviewed
# and re-ticked on evidence, per its own instruction to decide the state of
# "SYNTH-06 and LEARN-01 through LEARN-08" as a whole.

coverage:
  - id: D1
    description: "triggerRepeatedInterventionProposal gives proposeSourceImprovement its first real production caller: 3+ distinct episodes carrying the same declared intervention kind trigger exactly one source proposal on an isolated branch, with the working tree and checked-out branch unchanged afterwards -- closes SC5d's automatic-trigger half"
    requirement: LEARN-08
    verification:
      - kind: unit
        ref: "cmd/improvement_pass_test.go#TestRepeatedInterventionProposesExactlyOneSourceChange"
        status: pass
      - kind: unit
        ref: "cmd/improvement_pass_test.go#TestAutomaticProposalReplayCreatesNoSecondBranch"
        status: pass
      - kind: unit
        ref: "cmd/improvement_pass_test.go#TestTwoOccurrencesProposeNothing"
        status: pass
      - kind: unit
        ref: "cmd/improvement_pass_test.go#TestThresholdIsPerCategoryNotATotal"
        status: pass
      - kind: unit
        ref: "cmd/improvement_pass_test.go#TestDirtyTreeAutomaticProposalIsRefusedAndNeverBlocks"
        status: pass
    human_judgment: false
  - id: D2
    description: "The new automatic entry point is inside sourceProposalReachabilityEntryPoints in the same change that introduces it; the cannot-merge/publish/deploy call-graph walk covers it and still finds no path to a forbidden operation"
    requirement: LEARN-08
    verification:
      - kind: unit
        ref: "cmd/source_proposal_test.go#TestSourceProposalCannotMergePublishOrDeploy"
        status: pass
      - kind: unit
        ref: "cmd/source_proposal_test.go#TestForbiddenOperationListIsReadFromTheSource"
        status: pass
    human_judgment: false
  - id: D3
    description: "promoteHelpfulHypotheses promotes a StatusHypothesis learning entry to StatusValidated only when a corroborated guidance-application record for its own identifier has reached the helpful state -- never merely rendered/consulted/acted-on, never a worker's own claim, never a disproven entry -- closes WINDOWS entry 44"
    requirement: LEARN-08
    verification:
      - kind: unit
        ref: "cmd/learning_validator_test.go#TestHelpfulHypothesisIsPromotedAutomatically"
        status: pass
      - kind: unit
        ref: "cmd/learning_validator_test.go#TestActedOnIsNotEnoughToValidate"
        status: pass
      - kind: unit
        ref: "cmd/learning_validator_test.go#TestNeutralOrHarmfulIsNeverValidated"
        status: pass
      - kind: unit
        ref: "cmd/learning_validator_test.go#TestUncorroboratedClaimIsNeverValidated"
        status: pass
      - kind: unit
        ref: "cmd/learning_validator_test.go#TestHypothesisWithNoRecordStaysAHypothesis"
        status: pass
      - kind: unit
        ref: "cmd/learning_validator_test.go#TestPromotionIsIdempotent"
        status: pass
      - kind: unit
        ref: "cmd/learning_validator_test.go#TestDisprovenEntryIsNeverPromoted"
        status: pass
      - kind: unit
        ref: "cmd/learning_validator_test.go#TestHypothesisPromoterIsReachedFromBothCheckLanes"
        status: pass
    human_judgment: false
  - id: D4
    description: "REQUIREMENTS.md and WINDOWS.md say what the tests actually prove today: nine requirement lines re-evaluated from evidence, WINDOWS entries 38-45 all reconciled between the markdown table and the JSON ledger"
    requirement: null
    verification:
      - kind: other
        ref: "python3 script comparing every WINDOWS.md table row against its JSON ledger entry for ids 38-45 (and totals across all 46) -- see 'WINDOWS.md consistency check' below for the exact command and output"
        status: pass
    human_judgment: true
    rationale: "Whether a given requirement's own wording is 'true of the running program' is a judgment call informed by, but not mechanically reducible to, a passing test list -- a human should spot-check the reasoning in the Requirements Truth Table below, even though every cited test is independently confirmed passing."

# Metrics
duration: ~150min
completed: 2026-09-15
status: complete
---

# Phase 204 Plan 16: Automatic Source Proposals, Hypothesis Promotion, and Truthful Registers Summary

**Gave `proposeSourceImprovement` its first real caller (a repeated-intervention trigger) and `learn.StatusValidated` its first automatic writer (a corroborated-helpful-application promoter), then rewrote REQUIREMENTS.md and WINDOWS.md to say only what the tests in this session actually prove -- closing SC5d, WINDOWS entries 38/41/44/45, and narrowing 40/42 back to open with their remaining limits named.**

## Performance

- **Duration:** ~150 min (no wall-clock start captured at spawn; estimated from session activity)
- **Tasks:** 3
- **Files modified:** 13 (2 created, 11 modified)

## Accomplishments

- `triggerRepeatedInterventionProposal` (cmd/improvement_pass.go), called unconditionally at the end of `runAutomaticImprovementPass`, counts DISTINCT episodes per declared `episodeInterventionKind` category and, once any category reaches `sourceProposalRepeatedInterventionThreshold` (3), calls `proposeSourceImprovement` with a deterministic candidate id, the real episode identifiers as evidence, and a one-file plain-English case written to `.aether/reviews-archive/proposals/<candidate-id>.md` on an isolated branch.
- `sourceProposalReachabilityEntryPoints` gained the new trigger's name in the SAME change, and `TestSourceProposalCannotMergePublishOrDeploy`'s call-graph walk now starts from three entry points instead of two, still finding no path to merge/push/publish/deploy.
- `renderImprovementPassBeat` gained a fourth, plain-English closing-card line for a written proposal (`improvementPassEventProposalWritten`), completed in a follow-up commit after being missed in Task 1's initial commit (see Deviations).
- `promoteHelpfulHypotheses` (new file, cmd/learning_validator.go), called from `runPhaseEndConsolidation` alongside the improvement pass, promotes a hypothesis to validated only when a helper's guidance-application record for that entry's own id has independently reached `guidanceApplicationStateHelpful` -- through the exact same `learnStore.Replace` call the hand-run `learning-validate` command already uses.
- REQUIREMENTS.md's SYNTH-06 and LEARN-01 through LEARN-08 were each re-evaluated from evidence gathered in this session (see the Requirements Truth Table below); 8 of 9 are now ticked with the specific passing test(s) named inline, and LEARN-05 is left unticked with the real remaining number (30 of 47 fixture-bank incidents still unguarded).
- WINDOWS.md's entries 38, 41, 44, 45 are marked fixed; entries 40 and 42 are narrowed back to (or kept at) `open` with their remaining limits named; entry 43 is unchanged. The markdown table and the JSON ledger agree on the status of all 46 entries (verified by script, see below).
- CLAUDE.md's and `.claude/rules/aether-colony.md`'s (and its canonical source, `.aether/rules/aether-colony.md`'s) Learning Governor sections were corrected to cite the tests that now prove the automatic grader, the automatic canary pass, the automatic hypothesis promotion, and the automatic source-proposal trigger -- replacing prose that previously described these as merely structurally possible. The Quick Reference slash-command count was corrected from 60/60 to the real 64/64.

## Task Commits

Each task was committed atomically:

1. **Task 1: A repeated preventable intervention automatically becomes a source-improvement proposal** - `3e5c2837` (feat)
2. **Task 2: A hypothesis becomes validated only once the program's own records show it helped** - `3a1f386a` (feat)
3. **Task 1 follow-up: the missed closing-card line** - `7b15c725` (feat) -- see Deviations
4. **Task 3: Make REQUIREMENTS.md, WINDOWS.md and the shipped documents tell the truth** - `9cacfa39` (docs)

## Files Created/Modified

- `cmd/source_proposal.go` - new `sourceProposalRepeatedInterventionThreshold` constant (3)
- `cmd/source_proposal_test.go` - `sourceProposalReachabilityEntryPoints` gains `triggerRepeatedInterventionProposal`, comment rewritten
- `cmd/improvement_pass.go` - `triggerRepeatedInterventionProposal`, `sourceProposalRepeatedInterventionCandidateID`, `sourceProposalRepeatedInterventionChangeSet`, `sourceProposalRepeatedInterventionProposalsDir`; `runAutomaticImprovementPass`'s early return removed in favour of unconditional fall-through
- `cmd/improvement_pass_test.go` - the five Task-1 behavior tests plus `TestProposalWrittenClosingLineIsPlainEnglish`
- `cmd/codex_visuals.go` - `improvementPassEventSentence` gains the `improvementPassEventProposalWritten` case
- `cmd/learning_validator.go` (new) - `promoteHelpfulHypotheses`, `learningValidationSummary`, `learningEntryHasHelpfulApplication`
- `cmd/learning_validator_test.go` (new) - the eight Task-2 behavior tests
- `cmd/consolidation_lifecycle.go` - `phaseEndConsolidationSummary.LearningValidation` field; one call to `promoteHelpfulHypotheses` at the end of `runPhaseEndConsolidation`
- `.planning/REQUIREMENTS.md` - SYNTH-06, LEARN-01..08 re-ticked from evidence; Phase 204 traceability row annotated
- `.planning/WINDOWS.md` - header counts updated (open 34->31, fixed 12->15); entries 38/39/40/41/42/44/45 descriptions and statuses corrected in both the markdown table and the JSON ledger
- `CLAUDE.md` - Learning Governor section prose corrected and extended with new test citations; Quick Reference slash-command counts corrected
- `.claude/rules/aether-colony.md` / `.aether/rules/aether-colony.md` - mirrored prose corrections; `/ant-improve` added to the Advanced command table (see Deviations for why the canonical source file was also touched)

## Decisions Made

See `key-decisions` in the frontmatter above for the full reasoning on: the proposal directory choice, the early-return removal, `promoteHelpfulHypotheses`'s phase-scoping non-decision, reopening WINDOWS entry 40, and not running `requirements mark-complete`.

## Directory safety check (Task 1, plan's own required evidence)

The plan's own action text suggested `.aether/proposals/<proposal-id>.md`. Before using it, I read `cmd/install_cmd.go`'s `hubExcludeDirs` map and `listFilesRecursiveWithExclusion`/`pathHasExcludedComponent` (the functions that actually walk `.aether/` during `aether publish`/`install`'s hub sync): `hubExcludeDirs` does NOT contain an entry for `proposals`, so a bare `.aether/proposals/` directory would be synced into every downstream colony's hub install on the next publish -- leaking this colony's own generated case files into the shared, cross-project hub. I also checked `.aether/data/proposals/` as an alternative: `data` IS excluded from hub sync, but `.aether/data/` is also listed in this repository's own `.gitignore` (confirmed: `grep -n "\.aether" .gitignore` shows `.aether/data/` on its own line), so `git add` would silently refuse to stage a file there, defeating the whole point of committing the proposal onto its own branch.

I chose `.aether/reviews-archive/proposals/` instead: `reviews-archive` IS one of `hubExcludeDirs`'s entries (confirmed: `grep -n "reviews-archive" cmd/install_cmd.go`), so anything under it -- including a new `proposals/` subdirectory -- is skipped by `pathHasExcludedComponent`'s any-path-component match, and `.aether/reviews-archive/` is a real, already git-tracked directory (`git ls-files .aether/reviews-archive` lists its existing category subdirectories: bugs, history, performance, quality, resilience, security, testing) that carries no `.gitignore` entry of its own. `proposals/` sits alongside those existing categories as one more kind of archived finding.

## Requirements Truth Table (Task 3)

| ID | State | Justification |
|---|---|---|
| SYNTH-06 | Ticked | `TestClassicContractPhase204MechanismRegistry`, `TestClassicContractPhase204Cases`, `TestClassicContractRegistryHasNoRuntimeWriter` all pass; `204-CLASSIC-SYNTHESIS.md`/`204-CLASSIC-HISTORICAL-EVIDENCE.md` exist. FAILS-WHEN-UNWIRED for this one is inherited from 204-01's own plan, not re-performed here. |
| LEARN-01 | Ticked | `TestEveryMemoryStoreFieldHasALiveWriter`, `TestLegacyRecordsReadAsLegacy`, `TestNewRecordsCarryVersionAndLineage`, `TestHypothesisIsNeverRenderedAsVerified`, `TestOneLearningStatusVocabulary`, `TestRetiredFieldWithoutOwnerAgreementIsRefused` all pass. |
| LEARN-02 | Ticked, with a named remaining limit | `TestEveryLifecycleLaneWritesADurableOutcome` (zero skips), `TestSwarmLaneOpensAndClosesADurableEpisode`, `TestRecoveryLaneOpensADurableEpisodeOnlyWhenItOwnsOne`, `TestBuildEpisodeRecordsItsOwnFacts`, `TestCheckEpisodeRecordsItsOwnFacts`, `TestDelegateCheckEpisodeRecordsItsOwnFacts`, `TestEpisodeOutcomeIsRecordedFromBothCheckLanes` all pass. Remaining limit: `usage`/`reported_cost_usd` stay absent on both check lanes (204-15's own documented gap). |
| LEARN-03 | Ticked | `TestPhaseApplicationCreditTracerEndToEnd`, `TestPhaseApplicationCreditIsReachedFromBothCheckLanes`, `TestCreditRequiresBothFacts` all pass. |
| LEARN-04 | Ticked | `TestSeededBankFixturesAllCiteRealProvenance` passes; the bank's own structure/provenance/dedup rules are real regardless of guard completeness (LEARN-05's own concern). |
| LEARN-05 | **Left unticked** | `TestEvalGateVocabularyMatchesTheManifest`, `TestTruncatedGateRunFails`, `TestEverySentinelStillExists` pass (the gate architecture is real), but `seedBankUnguardedFloor = 30` (confirmed against `cmd/eval_gates.go` in this session) -- 30 of 47 confirmed incidents still carry no guarding test, so "the seed bank... preserve[s] coverage" is not yet true of the running program. |
| LEARN-06 | Ticked, with a named remaining limit | `TestShadowGraderDistinguishesBeneficialFromHarmful`, `TestOverfitCandidateIsRefusedAtTheGate`, `TestEvaluatorDigestIsStableAndChanged` all pass. Remaining limit: comparison covers this project's own settings/routing fixtures only, never a code-valued candidate -- LEARN-06's own scoping. |
| LEARN-07 | Ticked | `TestAutomaticImprovementPassTracerEndToEnd`, `TestAutomaticImprovementPassIsReachedFromBothCheckLanes`, `TestAutomaticPassNeverPromotesOutsideTheTwoScopes`, `TestAutomaticPassCarriesNoBypassParameter` all pass. |
| LEARN-08 | Ticked | `TestTwoFiguresAreRealOnALiveColony`, `TestReportBoundariesOnRealRecords`, `TestRenderedLiveReportKeepsTheTwoFiguresApart`, `TestRepeatedInterventionProposesExactlyOneSourceChange`, `TestAutomaticProposalReplayCreatesNoSecondBranch`, `TestSourceProposalCannotMergePublishOrDeploy` all pass. |

FAILS-WHEN-UNWIRED mutations recorded by the plans this table cites (by plan and test name, per the acceptance criteria): 204-12 (`TestAutomaticImprovementPassTracerEndToEnd` fails when `runAutomaticImprovementPass`'s call site is commented out of `runPhaseEndConsolidation`), 204-13 (`TestSwarmLaneOpensAndClosesADurableEpisode` / `TestEveryLifecycleLaneWritesADurableOutcome` fail when the swarm/recovery episode-boundary calls are commented out), 204-14 (`TestSeedBankUnguardedFloorIsTheRealCount` fails when the floor is inflated above 30), 204-15 (`TestBuildEpisodeRecordsItsOwnFacts` / `TestCheckEpisodeRecordsItsOwnFacts` / `TestDelegateCheckEpisodeRecordsItsOwnFacts` / `TestTwoFiguresAreRealOnALiveColony` each fail when their respective production writer is commented out), 204-16 (this plan's own two mutations, quoted in full below).

## WINDOWS.md consistency check

Ran a script parsing both the markdown table and the JSON ledger block, comparing `status` for every id and totaling all 46:

```
38 table= fixed json= fixed MATCH
39 table= fixed json= fixed MATCH
40 table= open json= open MATCH
41 table= fixed json= fixed MATCH
42 table= open json= open MATCH
43 table= open json= open MATCH
44 table= fixed json= fixed MATCH
45 table= fixed json= fixed MATCH
Total table rows: 46 Total json entries: 46
open: 31 fixed: 15 waived: 0 total: 46
mismatches: []
```

Header block (`open_count`/`fixed_count`/`waived_count`/`total_count`) updated to match: 31/15/0/46 (was 34/0/12/46 before this plan -- 38, 41, 44, 45 moved open->fixed; 40 moved fixed->open per its reopening).

## FAILS-WHEN-UNWIRED (performed, observed, reverted) -- this plan's own two mutations

**Task 1 -- commented out `triggerRepeatedInterventionProposal(&summary)` in `runAutomaticImprovementPass`:**
```
--- FAIL: TestRepeatedInterventionProposesExactlyOneSourceChange (0.11s)
    improvement_pass_test.go:571: expected exactly one proposal_written event, got []
```
Reverted; `go test ./cmd -run '^TestRepeatedInterventionProposesExactlyOneSourceChange$'` passes again. `git status --porcelain` / checked-out branch confirmed identical before and after the mutation-and-revert cycle.

**Task 2 -- commented out `summary.LearningValidation = promoteHelpfulHypotheses(phaseID)` in `runPhaseEndConsolidation`:**
```
--- FAIL: TestHypothesisPromoterIsReachedFromBothCheckLanes (0.26s)
    learning_validator_test.go:313: promoteHelpfulHypotheses is not transitively reachable from: [runCodexContinue runCodexContinueFinalize] -- both check lanes must reach the automatic hypothesis promoter
```
Reverted; the test passes again. Note: the plan's acceptance criteria named `TestHelpfulHypothesisIsPromotedAutomatically` for this mutation, but that test calls `promoteHelpfulHypotheses` directly rather than through `runPhaseEndConsolidation`, so it would NOT have caught this specific wiring removal. `TestHypothesisPromoterIsReachedFromBothCheckLanes` (the plan's own Test 8, an AST call-graph guard in the same shape as `TestAutomaticImprovementPassIsReachedFromBothCheckLanes`) is the test that actually depends on this call site, so it was used for this mutation instead -- documented here as a deliberate substitution, not a skipped check.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] `.claude/rules/aether-colony.md` is a generated file; editing only the checked-in copy broke ClaudeRuleGenerationParity**
- **Found during:** Task 3, running the plan's own verify command (`TestCurrentVocabulary199`)
- **Issue:** `.claude/rules/aether-colony.md` is not hand-authored independently -- it is a byte-for-byte production-generated copy of `.aether/rules/aether-colony.md` (the canonical source), enforced by `cmd/current_vocabulary_docs_199_test.go`'s `TestCurrentVocabularyDocs199/ClaudeRuleGenerationParity` subtest (itself invoked from `TestCurrentVocabulary199/executed-runtime-routes-and-compatibility`). Editing only the checked-in `.claude/` copy (as the plan's own files_modified list, which does not include `.aether/rules/aether-colony.md`, would suggest) made the two diverge and failed that subtest.
- **Fix:** Copied the corrected `.claude/rules/aether-colony.md` over `.aether/rules/aether-colony.md` so the two are byte-identical again, restoring the generation invariant. This touches one file outside the plan's declared `files_modified` (`.aether/rules/aether-colony.md`), which I judged a Rule 3 blocking-issue auto-fix (a genuine parity contract this project's own test enforces) rather than a scope change.
- **Files modified:** `.aether/rules/aether-colony.md`
- **Verification:** `go test ./cmd -run '^TestCurrentVocabulary199$'` -- the `executed-runtime-routes-and-compatibility` subtest (which runs `TestCurrentVocabularyDocs199` as a subprocess) now passes; only the pre-existing, unrelated `tracked-occurrences-are-exhaustively-classified` subtest still fails (see Known-Red Baseline below).
- **Committed in:** `9cacfa39` (Task 3 commit)

**2. [Rule 1 - Bug] `renderImprovementPassBeat` was not updated for the new `proposal_written` event in Task 1's initial commit**
- **Found during:** Reviewing Task 1's action text against my own diff before moving to Task 2
- **Issue:** The plan's Task 1 action text explicitly requires "`renderImprovementPassBeat` gains one more plain-English line for 'a proposal was written', per D-03" -- this was missed in the first commit (`3e5c2837`), which added the `improvementPassEventProposalWritten` event kind but never taught `improvementPassEventSentence` (cmd/codex_visuals.go) to render it, so a written proposal would have produced a silent (empty-string) closing-card line.
- **Fix:** Added the missing `case improvementPassEventProposalWritten:` branch, plus a new test (`TestProposalWrittenClosingLineIsPlainEnglish`) proving both the in-process struct and the JSON-round-tripped map shape render a non-empty line that never leaks the raw `proposal_written` token.
- **Files modified:** `cmd/codex_visuals.go`, `cmd/improvement_pass_test.go`
- **Verification:** `TestProposalWrittenClosingLineIsPlainEnglish` passes; full Task 1 test list re-run clean.
- **Committed in:** `7b15c725` (a follow-up commit, before the Task 3 commit)

---

**Total deviations:** 2 auto-fixed (1 Rule 3 blocking-issue, 1 Rule 1 bug caught before it shipped). **Impact on plan:** Neither widens scope beyond what the plan's own text already required (D-03's closing-card line was explicitly asked for; the generation-parity fix restores an existing project invariant this plan's own edit accidentally broke). No boundary was widened by either fix.

## Known-Red Baseline (pre-existing, not this plan's)

`go test ./cmd -run '^TestCurrentVocabulary199$'` still fails its `tracked-occurrences-are-exhaustively-classified` subtest, on the project's own documented known-red list (`TestCurrentVocabulary199` is named explicitly in this plan's project-specific rules as pre-existing on this branch). Observed failure text:
```
tracked occurrence keys = 196, inventory keys = 193; inventory must classify every exact tracked path and family
".planning/phases/199-front-door-and-classic-contract/199-PATTERNS.md\x00legacy_pause" count = 3, inventory = 0
".planning/phases/199-front-door-and-classic-contract/199-PATTERNS.md\x00legacy_resume" count = 4, inventory = 0
"cmd/testdata/fixture-bank/v1/bank.json\x00legacy_pause" count = 6, inventory = 0
```
This concerns `legacy_pause`/`legacy_resume` vocabulary tracking in Phase 199 documentation and the fixture bank, entirely unrelated to this plan's Learning Governor edits (source proposals, hypothesis promotion, REQUIREMENTS.md/WINDOWS.md/CLAUDE.md truth-telling). Not touched, not fixed, per the plan's own instruction not to spend time on it.

## Issues Encountered

None beyond the two documented deviations above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- SC5d is closed: `proposeSourceImprovement` has a real, evidence-gated production caller (`triggerRepeatedInterventionProposal`), and `TestSourceProposalCannotMergePublishOrDeploy`'s call-graph walk now provably covers that caller too.
- WINDOWS entry 44 is closed: a hypothesis is promoted to validated automatically, and only on a corroborated helpful application, reached from both check lanes.
- D-11 is implemented: REQUIREMENTS.md and WINDOWS.md entries say what the tests prove today; entries 40 and 42 are narrowed with their remaining limits named, entry 43 stays open as deferred.
- No boundary was widened: the four forbidden-operation families (`TestSourceProposalCannotMergePublishOrDeploy`), the retained-authority refusals (`TestOnlyTwoScopesAreCanaryPromotable`, `TestRetainedAuthorityCannotBecomeCanaryPromotable`, `TestEachRetainedScopeRefusalNamesItsAuthority`), and the corroboration rule (`TestCorroborationNeverReadsTheWorkersOwnText`, `TestPromotionRequiresAHelpfulApplication`) are all confirmed green in this session.
- Phase 204's own remaining honest gaps, now fully named in REQUIREMENTS.md and WINDOWS.md rather than silently marked done: LEARN-05 (30/47 fixtures unguarded), WINDOWS entry 40 (comparison never covers code-valued candidates), WINDOWS entry 42 (same 30/47 number), WINDOWS entry 43 (three knowingly-empty memory fields, explicitly deferred to a future phase that names a consumer).
- SYNTH-06's five edge probes (adjacency, empty, ordering, idempotency, concurrency) remain surfaced-but-unresolved, exactly as the plan's own flagged assumptions section directs -- SYNTH-06 is a completed documentation artifact this closure does not modify, so no defensible acceptance criterion could be written against those probes here.

## Self-Check: PASSED

- `cmd/learning_validator.go` — FOUND
- `cmd/learning_validator_test.go` — FOUND
- `cmd/source_proposal.go` (new constant) — FOUND
- `cmd/improvement_pass.go` (new trigger) — FOUND
- Commit `3e5c2837` — FOUND in `git log --oneline --all`
- Commit `3a1f386a` — FOUND in `git log --oneline --all`
- Commit `7b15c725` — FOUND in `git log --oneline --all`
- Commit `9cacfa39` — FOUND in `git log --oneline --all`
- All plan-level `<verification>` commands re-run clean in this session except the documented pre-existing `TestCurrentVocabulary199` subtest failure.
- WINDOWS.md table/JSON parity script re-run clean immediately before writing this summary.

---
*Phase: 204-learning-governor*
*Completed: 2026-09-15*
