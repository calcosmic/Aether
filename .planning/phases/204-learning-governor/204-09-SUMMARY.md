---
phase: 204-learning-governor
plan: "09"
subsystem: learning
tags: [promotion-gate, canary, rollback, quarantine, skill-proposal, owner-approval, LEARN-07]

# Dependency graph
requires:
  - phase: 204-01
    provides: "204-CLASSIC-SYNTHESIS.md's SYN-204-10/SYN-204-11 dispositions and ruling (d), naming this plan as the owner of the promotion gate, the canary rollback and the skill-proposal closure"
  - phase: 204-08
    provides: "pkg/shadow's Candidate/Comparison/Verdict/FrozenEvaluator -- the sealed comparison result the promotion gate consumes directly, never re-running or re-grading it"
provides:
  - "cmd/promotion_gate.go: canaryPromotableScopes (project knowledge, routing) and canaryRetainedAuthorityScopes (the nine retained categories, each mapped to the owner or an independent reviewer); admitCandidateToCanary, the gate whose own body derives its admissible set from the promotable list alone and never names a retained-authority identifier"
  - "cmd/rollback.go: startCanary (checkpoint-before-change, refused by name without a restore point), rollbackCanary (restore + quarantine in one atomic store update, replay-safe per candidate), completeCanary, canaryRegressionQuarantineThreshold, and releaseCanaryQuarantine -- the one function proven (AST scan + synthetic fixture) able to clear a quarantine"
  - "pkg/learn/difficulty.go: AutoCreateSkillIfDifficult no longer constructs a skill service or creates a skill in any mode -- it hands a SkillProposal to a caller-supplied SkillProposalSink; propose and auto modes now raise the identical proposal"
  - "cmd/suggest_approve.go: colonySkillProposalSink and approvePendingItem route skill proposals and quarantined canary candidates through the SAME owner tick-to-approve queue every other pending decision already uses"
affects: [204-10, 204-11]

actuals:
  tokens: 26890
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Structural authority refusal: the admitting function's own AST body is scanned (with a non-vacuous synthetic fixture) to prove it names zero identifiers from the retained-authority list -- widening the retained list can never silently widen what the gate admits, because the gate never reads that list at all."
    - "Atomic restore-then-quarantine: the file restore (reused saveRepairCheckpoint/restoreRepairCheckpoint) completes before the ONE store.UpdateJSONAtomically call that writes both the rolled-back status and the quarantine flag together, so no reader can observe one fact without the other."
    - "Owner-approval fan-in: every new kind of owner decision (skill proposal, canary quarantine release) is added as a new Origin value on the EXISTING colony.PendingSuggestion queue and a new branch in a single dispatch function (approvePendingItem), never a second CLI command."

key-files:
  created:
    - cmd/promotion_gate.go
    - cmd/promotion_gate_test.go
    - cmd/rollback.go
    - cmd/rollback_test.go
  modified:
    - pkg/colony/colony.go
    - pkg/learn/difficulty.go
    - pkg/learn/difficulty_test.go
    - cmd/suggest_approve.go
    - cmd/codex_continue_finalize.go

key-decisions:
  - "Candidate scope classification is an exact, case-insensitive match against one of the eleven declared canaryScope strings (canaryDeclaredScope), not a keyword-containment search. This keeps the admissible-set proof purely structural (a declared candidate's Scope() text IS the category) and matches the plan's own 'the scope must be in the promotable list' wording literally."
  - "Two independently-testable quarantine concepts, both real: (1) every regression unconditionally quarantines that specific candidate's own run record, atomically with the restore -- satisfying the plan's literal 'a regression... quarantines the candidate'; (2) a separate per-scope regression counter, evaluated by the pure canaryShouldQuarantine(previousCount) helper against canaryRegressionQuarantineThreshold, decides whether THIS regression is the one-time transition worth a fresh owner notification for that scope. TestQuarantineThresholdBoundaries drives (2) directly and purely; TestRegressionRestoresAndQuarantinesAtomically drives (1) end-to-end."
  - "startCanary resolves its checkpoint root via os.Getwd() (matching saveRepairCheckpoint's own root-relative-paths contract) rather than adding a root parameter the plan's own literal two-argument signature does not carry. Tests chdir into an isolated temp directory (t.Chdir) before calling it, so no test ever checkpoints this repository's own working tree."
  - "AutoCreateSkillIfDifficult's signature dropped its store/baseDir parameters entirely (replaced by mode + sink) -- the SQLite skill-existence check and use-count increment that used to gate 'auto' mode's direct creation moved to the approval side (cmd/suggest_approve.go's approveSkillProposal), since no mode constructs a skill service in pkg/learn any more."
  - "The credit-ledger evidence-gated outcome vocabulary (recruitmentCreditOutcome) was deliberately NOT reused for canary regression counting -- a canary regression is a structural rollback-and-restore event this plan directly owns, not a recruitment contribution's later-verified effect; reusing it would have required routing every canary rollback through recordRecruitmentCredit's ChangedDecisionID/EffectEvidenceID contract, which does not fit a scope-level counter."

patterns-established:
  - "A one-way authority boundary (never waivable, unlike the forced-reviewer pattern it borrows its SHAPE from) is proven structurally: the admitting function's source is parsed and scanned for the retained identifiers it must never name, backed by a synthetic violation fixture proving the scanner is not vacuous."

requirements-completed: []

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "The canary scope vocabulary (two promotable, nine retained-authority) and the promotion gate: a promotable, beneficial, unexpired candidate graded by the gate's own evaluator is admitted; every other case is refused by name, and a retained-authority scope's refusal names the authority that retains it with no waiver."
    requirement: LEARN-07
    verification:
      - kind: unit
        ref: "cmd/promotion_gate_test.go#TestOnlyTwoScopesAreCanaryPromotable"
        status: pass
      - kind: unit
        ref: "cmd/promotion_gate_test.go#TestRetainedAuthorityCannotBecomeCanaryPromotable"
        status: pass
      - kind: unit
        ref: "cmd/promotion_gate_test.go#TestEachRetainedScopeRefusalNamesItsAuthority"
        status: pass
      - kind: unit
        ref: "cmd/promotion_gate_test.go#TestNeitherCoordinatorNorAutopilotCanWaiveARetainedRefusal"
        status: pass
      - kind: unit
        ref: "cmd/promotion_gate_test.go#TestNonBeneficialVerdictsAreEachRefusedInTheirOwnWords"
        status: pass
      - kind: unit
        ref: "cmd/promotion_gate_test.go#TestMismatchedGraderDigestIsRefused"
        status: pass
      - kind: unit
        ref: "cmd/promotion_gate_test.go#TestExpiredCandidateIsRefusedAtTheGate"
        status: pass
      - kind: unit
        ref: "cmd/promotion_gate_test.go#TestPromotableCandidateIsAdmitted"
        status: pass
    human_judgment: false
  - id: D2
    description: "A canary cannot change anything before its restore point exists; a regression restores the checkpoint and quarantines the candidate in one atomic store update; a holdout regression rolls back exactly like a hard-gate regression; the quarantine trigger count is a named constant with proven boundary behaviour; no automatic path can release a quarantine; replaying a rollback returns the first receipt; every start/rollback is both inline and durable."
    requirement: LEARN-07
    verification:
      - kind: unit
        ref: "cmd/rollback_test.go#TestCanaryRefusesToChangeWithoutARestorePoint"
        status: pass
      - kind: unit
        ref: "cmd/rollback_test.go#TestRegressionRestoresAndQuarantinesAtomically"
        status: pass
      - kind: unit
        ref: "cmd/rollback_test.go#TestHoldoutRegressionAlsoRollsBack"
        status: pass
      - kind: unit
        ref: "cmd/rollback_test.go#TestQuarantineThresholdBoundaries"
        status: pass
      - kind: unit
        ref: "cmd/rollback_test.go#TestNoAutomaticPathReleasesAQuarantine"
        status: pass
      - kind: unit
        ref: "cmd/rollback_test.go#TestRollbackReplayReturnsTheFirstReceipt"
        status: pass
      - kind: unit
        ref: "cmd/rollback_test.go#TestCanaryEventsAreInlineAndDurable"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_resolver_test.go#TestNoUngovernedQuarantineClear"
        status: pass
    human_judgment: false
  - id: D3
    description: "Neither the previously-silent-no-op 'propose' mode nor the previously-direct-creating 'auto' mode creates an active skill any more -- both raise the identical proposal into the owner's approval queue; approving creates the real skill, dismissing creates nothing; a quarantined canary candidate is releasable only from that same queue; a skill proposal names its source learning entry; the real sink is wired at the one shared continue-lane chokepoint reached by both lanes."
    requirement: LEARN-07
    verification:
      - kind: unit
        ref: "cmd/promotion_gate_test.go#TestAutoDerivedSkillEntersTheApprovalListNotTheActiveSet"
        status: pass
      - kind: unit
        ref: "cmd/promotion_gate_test.go#TestApprovingASkillProposalCreatesTheSkill"
        status: pass
      - kind: unit
        ref: "cmd/promotion_gate_test.go#TestDismissingASkillProposalCreatesNothing"
        status: pass
      - kind: unit
        ref: "cmd/promotion_gate_test.go#TestQuarantinedCandidateIsReleasableOnlyFromTheApprovalSurface"
        status: pass
      - kind: unit
        ref: "cmd/promotion_gate_test.go#TestSkillProposalNamesItsSourceLearningEntry"
        status: pass
      - kind: unit
        ref: "cmd/promotion_gate_test.go#TestSkillProposalSinkIsWiredFromTheCheckPath"
        status: pass
      - kind: unit
        ref: "cmd/suggest_approve_test.go#TestOneApprovalSurface"
        status: pass
      - kind: unit
        ref: "pkg/learn/difficulty_test.go#TestAutoSkillCreation_AutoModeRaisesAProposal"
        status: pass
      - kind: unit
        ref: "pkg/learn/difficulty_test.go#TestAutoSkillCreation_ProposeModeRaisesTheIdenticalProposal"
        status: pass
    human_judgment: false

duration: 95min
completed: 2026-09-14
status: complete
---

# Phase 204 Plan 09: Learning Governor -- Promotion Gate, Canary Rollback and Owner-Governed Skill Proposals Summary

**A structural authority gate that refuses nine kinds of self-change by name and admits only project-knowledge/routing candidates with a beneficial, unexpired, correctly-graded verdict; a canary runner that never changes anything before a real restore point exists and atomically restores-and-quarantines on regression; and both auto-skill modes now routing through the owner's one existing approval queue instead of one of them silently creating a skill with zero involvement.**

## Performance
- **Duration:** ~95 min
- **Tasks:** 3/3 completed
- **Files created:** 4 (cmd/promotion_gate.go, cmd/promotion_gate_test.go, cmd/rollback.go, cmd/rollback_test.go)
- **Files modified:** 5 (pkg/colony/colony.go, pkg/learn/difficulty.go, pkg/learn/difficulty_test.go, cmd/suggest_approve.go, cmd/codex_continue_finalize.go)

## Accomplishments
- `cmd/promotion_gate.go` -- `canaryPromotableScopes` (project knowledge, routing) and `canaryRetainedAuthorityScopes` (preferences, skills, workflows, source, security, deletion, permission, verification, external actions), each retained scope mapped to the owner or an independent reviewer. `admitCandidateToCanary` checks scope, verdict (beneficial required; overfit/not-beneficial/tied/inconclusive each refused in their own words), expiry, and grader-digest match, in that order -- its own function body carries no reference to the retained list or any of its nine members, proven by an AST scan (`TestRetainedAuthorityCannotBecomeCanaryPromotable`) carrying a synthetic fixture that the scan is demonstrated to catch by symbol name.
- `cmd/rollback.go` -- `startCanary` saves a restore point through the existing `saveRepairCheckpoint` before ever recording a run, refusing by name (and writing nothing) when no restore point could be saved. `rollbackCanary` restores the checkpoint first, then writes the rolled-back status, the quarantine flag, and the scope regression count in ONE atomic store update -- replay-safe per candidate (`TestRollbackReplayReturnsTheFirstReceipt`). `canaryRegressionQuarantineThreshold` is a named constant with a pure, independently-tested boundary rule (`canaryShouldQuarantine`). `releaseCanaryQuarantine` is the one function an AST scan over every non-test file in `cmd/` and `pkg/` (with a synthetic second-releaser fixture) proves can clear a quarantine.
- `pkg/learn/difficulty.go` -- `AutoCreateSkillIfDifficult` constructs no skill service in any mode; `mode` now controls only whether a `SkillProposal` is raised at all, never whether the owner is bypassed. "propose" mode, whose own prior doc comment claimed a candidate was "logged" while doing nothing at all, now raises the identical proposal "auto" mode raises.
- `cmd/suggest_approve.go` -- `colonySkillProposalSink` enqueues a proposal into the same `colony.PendingSuggestion` queue every pending decision already uses; `approvePendingItem` is the one dispatch point that routes a skill proposal to skill creation, a quarantined canary to `releaseCanaryQuarantine`, and everything else to the pre-existing `approvePendingNote` -- no second approval command.
- `pkg/colony/colony.go` -- `PendingSuggestion` gained two new `Origin` values (`skill_proposal`, `canary_candidate`) and the fields each carries (skill name/source-run/learning-entry/confidence; canary candidate id), all `omitempty` so a legacy item round-trips unaffected.

## Task Commits
Each task was committed atomically:
1. **Task 1: Refuse the nine retained-authority categories structurally, not by rule** - `531d0ac5` (feat)
2. **Task 2: Run the canary behind a restore point, and undo it in one operation when it goes wrong** - `3da592fe` (feat)
3. **Task 3: Route automatically derived skills and canary candidates into the one approval list** - `21698edb` (feat)

**Plan metadata:** committed alongside this SUMMARY (worktree mode -- STATE.md/ROADMAP.md excluded, handled by the orchestrator).

## Files Created/Modified
- `cmd/promotion_gate.go` - `canaryScope`, `canaryPromotableScopes`, `canaryRetainedAuthorityScopes`, `canaryRetainedAuthority`, `canaryRetainedAuthorityFor`, `canaryDeclaredScope`, `canaryAdmission`, `canaryBoundExceeded`, `canaryGateResolvesEvaluatorDigest`, `admitCandidateToCanary`
- `cmd/promotion_gate_test.go` - all 7 Task 1 tests, `TestPromotableCandidateIsAdmitted` (positive-path proof), and all 6 Task 3 tests
- `cmd/rollback.go` - `canaryRun`, `canaryRunStatus`, `canaryRollbackReceipt`, `canaryRunFile`, `canaryRegressionQuarantineThreshold`, `canaryShouldQuarantine`, `canaryCheckpointIdentity`, `canaryEpisodeID`, `startCanary`, `completeCanary`, `rollbackCanary`, `releaseCanaryQuarantine`, `enqueueCanaryQuarantineApproval`, `emitCanaryStarted`/`emitCanaryCompleted`/`emitCanaryRolledBack`
- `cmd/rollback_test.go` - all 7 Task 2 named tests plus `TestCompleteCanaryMarksTheChangeAsKept` and `TestReleaseCanaryQuarantineThroughApprovedPath`
- `pkg/colony/colony.go` - `PendingOriginSkillProposal`, `PendingOriginCanaryCandidate`, four new `PendingSuggestion` fields for skill provenance, `CanaryCandidateID`
- `pkg/learn/difficulty.go` - `SkillProposalSink`, `SkillProposal`, retyped `AutoCreateSkillIfDifficult(entry, mode, sink)`
- `pkg/learn/difficulty_test.go` - rewrote the auto-skill-creation test section against a `fakeSkillProposalSink`, added `TestAutoSkillCreation_NoModeConstructsASkillService`
- `cmd/suggest_approve.go` - `approvePendingItem`, `colonySkillProposalSink`, `approveSkillProposal`, `approveCanaryQuarantineRelease`
- `cmd/codex_continue_finalize.go` - `captureContinueLearning` pre-assigns `entry.ID` (Add's pass-by-value copy never returned it) and calls the new `AutoCreateSkillIfDifficult(entry, mode, colonySkillProposalSink{})` signature

## Decisions Made
See `key-decisions` in frontmatter for the full account. Summary: candidate scope classification is exact-match against the eleven declared scope strings, not keyword search; every regression unconditionally quarantines its own candidate atomically with the restore, while a separate pure per-scope threshold counter governs a one-time owner-notification transition; `startCanary` resolves its checkpoint root via `os.Getwd()` under the plan's literal two-argument signature, with tests `t.Chdir`-isolated; `AutoCreateSkillIfDifficult` dropped its store/baseDir parameters entirely in favor of `mode, sink`; canary regression counting was deliberately kept separate from the recruitment credit ledger's evidence-gated vocabulary, which does not fit a scope-level structural rollback event.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `AutoCreateSkillIfDifficult`'s signature change broke its sole production call site and every existing test**
- **Found during:** Task 3, immediately after retyping the function
- **Issue:** `cmd/codex_continue_finalize.go`'s `captureContinueLearning` called the old four-argument signature (`entry, store, baseDir, mode`), and `pkg/learn/difficulty_test.go` had ten call sites against the same old signature, none in this plan's own declared `files_modified`.
- **Fix:** Updated `captureContinueLearning` to call the new `(entry, mode, colonySkillProposalSink{})` signature, removing the now-dead `NewSQLiteColonyStore`/`ResolveAetherRoot` setup and its now-unused `context`/`pkg/storage` imports. Rewrote the affected test section in `pkg/learn/difficulty_test.go` against a `fakeSkillProposalSink` test double, preserving every test's original intent (auto/off/propose mode behaviour, easy-task skip, non-blocking error handling) while asserting on the proposal raised rather than a directly-created skill.
- **Files modified:** `cmd/codex_continue_finalize.go`, `pkg/learn/difficulty_test.go`
- **Verification:** `go build ./...` and `go vet ./cmd/... ./pkg/...` both clean; `go test ./pkg/learn -count=1` passes in full (28 tests, including the 9 rewritten auto-skill tests).
- **Committed in:** `21698edb`

**2. [Rule 1 - Bug] `learn.ColonyStore.Add` never returns an assigned entry ID to its caller (pass-by-value)**
- **Found during:** Task 3, wiring `SkillProposal.LearningEntryID`
- **Issue:** `Add(entry Entry) error` takes `entry` by value; when `entry.ID` is empty it generates one on its own local copy and never communicates it back (no return value, no pointer receiver). `captureContinueLearning`'s pre-existing code relied on this and would have handed every skill proposal an empty `LearningEntryID`, defeating this plan's own "so its provenance is nameable" requirement.
- **Fix:** Pre-assign `entry.ID` (via the existing `generateSignalID()` helper) before calling `Add`, documented inline so a future reader does not reintroduce the same silent loss.
- **Files modified:** `cmd/codex_continue_finalize.go`
- **Verification:** `TestSkillProposalNamesItsSourceLearningEntry` passes and would fail without this fix (the assertion directly compares the queued item's `SkillLearningEntryID` against the entry's own pre-assigned `ID`).
- **Committed in:** `21698edb`

---
**Total deviations:** 2 auto-fixed (Rule 3 blocking, Rule 1 bug) -- both direct, unavoidable consequences of this plan's own mandated signature change. **Impact on plan:** None on any machine-checkable gate -- every `<acceptance_criteria>` line and every task's `<verify>` command for Tasks 1-3 pass; a broader regression sweep (`TestEpisode*`, `TestEvalGate*`/`TestSeedBank*`/`TestFixtureBank*`/`TestSeededBank*`, `TestRecruitmentCredit*`, `TestPheromone*`, `TestSkill*`, `TestContinueLearning*`/`TestCaptureContinue*`/`TestMemoryFeed*`/`TestColonyPrime*`/`TestAutopilotLessons*`/`TestConfirmedAutopilot*`, `TestCLIFlagAudit`, `TestVoicedScreensSpeakPlainEnglish`, `TestVoiceGlyphsHaveOneTable`, `TestNoUngovernedQuarantineClear`, `pkg/shadow` in full) passes with zero regressions.

## Issues Encountered
None beyond the deviations documented above. `go build ./...`, `go vet ./cmd/... ./pkg/...` are clean; `gofmt -l` reports nothing for any file this plan touched.

`TestRegressionRestoresAndQuarantinesAtomically`'s "demonstrated able to fail" requirement (CLAUDE.md's own Definition of Done) was verified manually during implementation: temporarily splitting `rollbackCanary`'s single `store.UpdateJSONAtomically` call into two separate atomic writes (one for `Status`, a second for `Quarantined`) and re-running the test's own "status rolled_back implies Quarantined true" assertion produced a genuine, real failure when the read was constructed between the two writes, before the split was reverted to the real, atomic implementation.

## User Setup Required
None -- no external service configuration required.

## Next Phase Readiness
`admitCandidateToCanary`, `startCanary`, `rollbackCanary`, `completeCanary`, `releaseCanaryQuarantine`, `learn.SkillProposalSink`/`learn.SkillProposal`, and `colony.PendingOriginSkillProposal`/`colony.PendingOriginCanaryCandidate` are all established, tested, exported (or cmd-package-visible) symbols any later plan touching the learning governor's self-change surface can build on directly.

`LEARN-07` is shared with plan `204-11` (not yet complete in this worktree) -- `gsd_run query requirements.ready-ids` confirmed `0/1` ready as of this plan's completion, so it correctly stays unmarked in `REQUIREMENTS.md`; `204-11`'s own completion will trip the shared-ID gate once every declaring plan is done.

No blockers.

## Self-Check: PASSED

- All 4 created files and 5 modified files confirmed present via `git status`/`git log`.
- All 3 task commits (`531d0ac5`, `3da592fe`, `21698edb`) confirmed present via `git log --oneline`.
- Acceptance-criteria greps re-run clean: `canaryPromotableScopes`/`canaryRetainedAuthorityScopes` present in `cmd/promotion_gate.go` with 2 and 9 members respectively; `saveRepairCheckpoint`/`canaryRegressionQuarantineThreshold` present in `cmd/rollback.go`; `grep -c 'NewSkillService' pkg/learn/difficulty.go` returns 0; `grep -c 'calcosmic/Aether/cmd' pkg/learn/difficulty.go` returns 0.
- Full combined named `<verify>` test sets for all three tasks pass together (re-run in this session): Task 1's 10-test group (including `TestOnlyTheOwnerCanWaiveAForcedReviewer`/`TestAutopilotNeverWaives`/`TestOneApprovalSurface`), Task 2's 8-test group (including `TestNoUngovernedQuarantineClear`), and Task 3's 7-test group across `cmd`/`pkg/learn`/`pkg/colony` (including `TestOneApprovalSurface`).
- `go build ./...` and `go vet ./cmd/... ./pkg/...` both clean; `gofmt -l` clean on every file this plan created or modified.
- Broader regression sweep passes with zero collateral damage (see Deviations summary above for the full list).
- `LEARN-07` confirmed shared with `204-11` (not yet complete) via `gsd_run query requirements.ready-ids` -- correctly NOT marked complete in `REQUIREMENTS.md`, per the shared-ID gate.

---
*Phase: 204-learning-governor*
*Completed: 2026-09-14*
