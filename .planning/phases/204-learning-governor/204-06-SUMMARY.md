---
phase: 204-learning-governor
plan: "06"
subsystem: learning-pipeline
tags: [instincts, guidance-application, queen-promotion, credit-ledger, learn-03]

# Dependency graph
requires:
  - phase: 204-learning-governor (plans 01-03)
    provides: the evidence-gated credit ledger (recordRecruitmentCredit), the typed InstinctApplicationEntry.Outcome shape, and recordPhaseApplicationCredit's real credit derivation from a phase's own build attempt
provides:
  - The nine-state guidance application vocabulary (available, rendered, consulted, acted_on, ignored, contradicted, helpful, neutral, harmful) with a declared, closed transition rule
  - Independent corroboration of a worker's own claim to have consulted or acted on a piece of guidance, against durable runtime-derived evidence only
  - A QUEEN.md promotion gate that requires at least one genuinely helpful application, not merely three applications of any kind, with a named reason for every declined instinct
affects: [learning-governor phase closure, any future plan reading InstinctApplicationSummary or the credit store's sibling guidance-application arrays]

# Actuals (#2632)
actuals:
  tokens: 24285
  tasks: 3
  commits: 5

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "guidanceApplicationTransitionRule{Requires, Excludes} declared once in a map, with a generic refusal-check function that never compares a named state constant inline (AST-enforced)"
    - "A new record type lands as a sibling array in an existing evidence-gated store file, permitted into that store's single-writer AST guard by symbol name rather than by file path"
    - "A worker's own free-text handoff fields are read only inside the claim-EXTRACTION function; the claim-CORROBORATION function is scoped by a second AST guard to touch only runtime-derived evidence (ChangedFiles, deriveBuildKnowledgeDeltas output, the phase's durable decision record)"

key-files:
  created: []
  modified:
    - cmd/application_evidence.go (the nine-state vocabulary, its transition rule, the writer, claim extraction/corroboration, the phase-close ignored sweep)
    - cmd/recruitment_credit.go (sibling GuidanceApplications/GuidanceClaims arrays on recruitmentCreditFile)
    - cmd/recruitment_credit_test.go (single-writer guard extended by symbol name)
    - cmd/instinct_application.go (recordInstinctDeliveries now records available/rendered at the same pass it decides eligibility)
    - cmd/memory_feed.go (one new call site: recordGuidanceStatesForWorkerOutcome)
    - pkg/memory/instinct_stats.go (HelpfulApplications/HarmfulApplications/IgnoredApplications counters)
    - pkg/memory/consolidate.go (the three named promotion-floor constants, the helpful-application gate, QueenDeclinedReason)
    - cmd/application_evidence_test.go, cmd/instinct_application_test.go, cmd/seal_ceremony_test.go, cmd/consolidation_lifecycle_test.go, pkg/memory/consolidate_test.go, pkg/memory/pipeline_test.go (new tests plus every existing hand-built instinct fixture that relied on application count alone, updated to also carry a genuinely helpful application)

key-decisions:
  - "The last three guidance states (helpful/neutral/harmful) reuse the credit ledger's own three words as a SEPARATE Go type, never an alias -- a decision-level credit outcome and a delivery-level guidance state answer different questions and can legitimately disagree."
  - "A worker cannot literally cite an instinct's internal ID (colony-prime never renders one into a capsule), so a claim is detected by the same signal delivery already uses: the guidance's own action text appearing in the worker's own retrospective handoff fields (NextWorkerInstructions, DoNotRepeat)."
  - "Corroboration is scoped to exactly three runtime-derived sources: this worker's own recorded changed-file list, deriveBuildKnowledgeDeltas' fresh derivation from this worker's own already-persisted handoff, and the phase's already-durable decision record -- never the worker's own free-text fields directly, enforced by an AST guard."
  - "Contradiction is detected via one explicit, literal marker phrase (guidanceContradictionMarker) immediately followed by the guidance's own action text in a worker's own recorded decision, rather than any prose inference."
  - "Ignored is swept once per phase-end pass, in recordPhaseApplicationCredit itself via defer, so it runs on every return path and correctly waits for a later worker in the same phase who might still consult guidance an earlier one did not."
  - "Every existing hand-built instinct fixture across cmd/ and pkg/memory/ that relied on application count alone to reach QUEEN.md eligibility was updated to also carry a genuinely helpful ApplicationHistory entry, rather than weakening the new gate to keep old fixtures passing unmodified."

patterns-established:
  - "guidanceApplicationTransitionRefusal: a single generic function walking a rule's Requires/Excludes lists, so no future state addition can accidentally reimplement a transition check inline"
  - "A phase-close sweep runs via defer at the top of the phase-end function, guaranteeing it fires on every return path including early-exit branches"

requirements-completed: [LEARN-03]

coverage:
  - id: D1
    description: "The nine-state guidance application vocabulary (available, rendered, consulted, acted_on, ignored, contradicted, helpful, neutral, harmful) with a declared, enforced transition rule"
    requirement: LEARN-03
    verification:
      - kind: unit
        ref: "cmd/application_evidence_test.go#TestGuidanceStateVocabularyIsClosed"
        status: pass
      - kind: unit
        ref: "cmd/application_evidence_test.go#TestGuidanceStateTransitionsRequireTheirPredecessors"
        status: pass
      - kind: unit
        ref: "cmd/application_evidence_test.go#TestAllNineGuidanceStatesAreReachable"
        status: pass
      - kind: unit
        ref: "cmd/application_evidence_test.go#TestIgnoredAndConsultedAreMutuallyExclusive"
        status: pass
      - kind: unit
        ref: "cmd/application_evidence_test.go#TestGuidanceStateRepeatWritesNothing"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_credit_test.go#TestCreditRequiresBothFacts"
        status: pass
    human_judgment: false
  - id: D2
    description: "A worker's own claim to have consulted or acted on a piece of guidance is independently corroborated against durable runtime evidence, never accepted on the worker's own word"
    requirement: LEARN-03
    verification:
      - kind: integration
        ref: "cmd/application_evidence_test.go#TestRenderedAloneIsNotConsulted"
        status: pass
      - kind: integration
        ref: "cmd/application_evidence_test.go#TestCorroboratedClaimBecomesConsulted"
        status: pass
      - kind: integration
        ref: "cmd/application_evidence_test.go#TestUncorroboratedClaimIsRecordedAsUnverified"
        status: pass
      - kind: integration
        ref: "cmd/application_evidence_test.go#TestRenderedAndUnusedBecomesIgnoredAtPhaseClose"
        status: pass
      - kind: integration
        ref: "cmd/application_evidence_test.go#TestContradictedIsRecordedFromARecordedDecision"
        status: pass
      - kind: unit
        ref: "cmd/application_evidence_test.go#TestCorroborationNeverReadsTheWorkersOwnText"
        status: pass
      - kind: unit
        ref: "cmd/memory_feed_test.go#TestEveryBuildLaneFeedsMemoryThroughOneBoundary"
        status: pass
      - kind: unit
        ref: "cmd/application_evidence_test.go#TestPhaseApplicationCreditIsReachedFromBothCheckLanes"
        status: pass
    human_judgment: false
  - id: D3
    description: "QUEEN.md promotion requires at least one genuinely helpful application, with a named reason for every declined instinct"
    requirement: LEARN-03
    verification:
      - kind: integration
        ref: "cmd/application_evidence_test.go#TestPromotionRequiresAHelpfulApplication"
        status: pass
      - kind: unit
        ref: "cmd/application_evidence_test.go#TestDeclinedPromotionNamesItsReason"
        status: pass
      - kind: unit
        ref: "cmd/application_evidence_test.go#TestPromotionThresholdsAreNamedConstants"
        status: pass
      - kind: integration
        ref: "cmd/instinct_application_test.go#TestWorkerLessonBecomesQueenFileWisdom"
        status: pass
      - kind: integration
        ref: "cmd/instinct_application_test.go#TestQueenPromotionNeverHappensWithoutRecordedUse"
        status: pass
    human_judgment: false

# Metrics
duration: 55min
completed: 2026-09-14
status: complete
---

# Phase 204 Plan 06: Learning Governor — Guidance Application Ledger Summary

**Nine declared states (available through harmful) with an enforced transition rule replace the old "a delivered note plus a passing phase is a success" mechanism, and a worker's claim to have used a piece of guidance is independently checked against runtime evidence before it ever counts.**

## Performance
- **Duration:** ~55 min
- **Completed:** 2026-09-14T13:45:18Z
- **Tasks:** 3 (plan) + 1 coverage-completeness addition
- **Files modified:** 13

## Accomplishments
- Declared `guidanceApplicationState` — a closed, nine-member vocabulary (available, rendered, consulted, acted_on, ignored, contradicted, helpful, neutral, harmful) with every transition rule expressed once, in `guidanceApplicationPredecessors`, and enforced by `recordGuidanceApplicationState` — the single writer, storing records as a sibling array inside the credit ledger's own file so one read still shows the whole picture of a contribution.
- Added claim verification: `guidanceClaimFromHandoff` extracts what a worker's own handoff says it consulted; `corroborateGuidanceClaim` checks that claim against durable, runtime-derived evidence only (this worker's own changed files, freshly-derived decision deltas, or the phase's already-durable decision record) — never the worker's raw prose, proven by an AST guard (`TestCorroborationNeverReadsTheWorkersOwnText`).
- An uncorroborated claim is recorded `claimed-but-unverified`, neither accepted nor discarded; guidance rendered and never consulted, acted on, contradicted, or claimed is swept to `ignored` once, at phase close.
- The QUEEN.md promotion gate in `pkg/memory/consolidate.go` now requires at least one genuinely helpful application, not merely three applications of any kind, using three named constants instead of inline literals, and records why every declined instinct fell short.

## Task Commits
1. **Task 1: Declare the nine guidance states and the transitions between them** - `5c096b9e` (feat)
2. **Task 2: Check a worker's claim against what the runtime can see for itself** - `0f472254` (feat)
3. **Task 3: Require a lesson to have actually helped before it reaches the shared instruction file** - `e57d2f72` (feat)
4. **Coverage addition: prove all nine states positively reachable** - `bef335f2` (test)
**Plan metadata:** _pending_ (this commit)

## Files Created/Modified
- `cmd/application_evidence.go` - The nine-state vocabulary, transition rule, writer, claim extraction/corroboration, contradiction detection, and the phase-close ignored sweep.
- `cmd/recruitment_credit.go` - `recruitmentCreditFile` gains `GuidanceApplications`/`GuidanceClaims` sibling arrays.
- `cmd/recruitment_credit_test.go` - The single-writer AST guard now permits the two new writers by symbol name.
- `cmd/instinct_application.go` - `recordInstinctDeliveries` records available/rendered at the same pass it already decides eligibility.
- `cmd/memory_feed.go` - One new call site: `recordGuidanceStatesForWorkerOutcome`, between delivery recording and the existing memory feed.
- `pkg/memory/instinct_stats.go` - `HelpfulApplications`/`HarmfulApplications`/`IgnoredApplications` counters on `InstinctApplicationSummary`.
- `pkg/memory/consolidate.go` - Three named promotion-floor constants, the helpful-application gate, `QueenDeclinedReason`.
- `cmd/application_evidence_test.go` - All new Task 1-3 tests, plus the AST guards.
- `cmd/instinct_application_test.go`, `cmd/seal_ceremony_test.go`, `cmd/consolidation_lifecycle_test.go`, `pkg/memory/consolidate_test.go`, `pkg/memory/pipeline_test.go` - Existing hand-built instinct fixtures updated to carry a genuinely helpful application, matching the new gate.

## Decisions Made
See `key-decisions` in frontmatter above.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - missing critical functionality] Added a positive reachability test for all nine states**
- **Found during:** post-Task-3 review against the plan's own `<verification>` block ("All nine declared states are reachable, and each is refused without its predecessors")
- **Issue:** The Task 1 tests only proved the refusal half of that sentence (each state refused without its predecessors); no test proved the positive half (each state IS reachable once its predecessors exist).
- **Fix:** Added `TestAllNineGuidanceStatesAreReachable`, walking each state's own predecessor chain (with `ignored`'s own special case: requires `rendered` but excludes `consulted`) and asserting the write succeeds for all nine.
- **Files modified:** `cmd/application_evidence_test.go`
- **Verification:** `go test ./cmd -run TestAllNineGuidanceStatesAreReachable` — all nine subtests pass.
- **Committed in:** `bef335f2`

**2. [Rule 1 - bug] Fixed six existing hand-built instinct fixtures that no longer reached QUEEN.md eligibility**
- **Found during:** Task 3's own regression sweep (`go test ./pkg/memory/...` and targeted `cmd` groups)
- **Issue:** `TestConsolidate_QueenEligible`, `TestConsolidationResult_Fields`, `TestPipeline_InstinctToQueen`, `TestPipeline_Consolidation`, `TestPipeline_FullCycle`, `TestPipeline_RunConsolidation_QueenPromotedTracksActualWrites` (both subtests), `TestPipelineInjectedQueenPromotion200`, `TestPipelineQueenPromotionFailureAccounting200`, `TestSealReportsPipelinePromotedInstinctsBelowLocalBar`, `TestRunSealConsolidationQueenPromotedIDsExcludesFailedWrites`, `TestSealDoesNotDoublePromoteInstincts`, and `TestPhaseEndConsolidationReportsWhatReachedTheQueenFile` all relied on application count alone (via `Provenance.ApplicationCount`, never a typed `ApplicationHistory` entry) to become QUEEN.md-eligible — the exact behavior the new gate is designed to end.
- **Fix:** Added a genuinely helpful `ApplicationHistory` entry to each fixture (directly for hand-typed instincts; via a new `seedOneHelpfulApplicationCreditBuildAttempt` test helper, reusing the real `attachBuildFreeCheckReport`/`attachBuildKnowledgeDeltas` production setters, for the two tests that drive the real phase-end pipeline round by round).
- **Files modified:** `pkg/memory/consolidate_test.go`, `pkg/memory/pipeline_test.go`, `cmd/seal_ceremony_test.go`, `cmd/consolidation_lifecycle_test.go`, `cmd/instinct_application_test.go`
- **Verification:** Full `pkg/memory` suite and targeted `cmd` groups (`Instinct`, `Consolidat`, `Queen`, `Credit`, `Guidance`, `Seal`) all pass.
- **Committed in:** `e57d2f72`

---
**Total deviations:** 2 auto-fixed (1 Rule 2, 1 Rule 1).
**Impact on plan:** No scope change. Both were required to keep the existing suite green and to fully satisfy the plan's own stated verification block; neither weakened the new gate or vocabulary.

## Proof the new gate can actually fail (CLAUDE.md's Definition of Done)
Before finalizing Task 3, the `summary.HelpfulApplications >= queenPromotionHelpfulFloor` condition was temporarily removed from `pkg/memory/consolidate.go`, `TestPromotionRequiresAHelpfulApplication` was re-run and confirmed to FAIL (the "no helpful application" colony became eligible), then the condition was restored and the test re-confirmed passing. This is documented rather than left implicit, per this repo's own rule that a test must be demonstrably able to fail.

## Issues Encountered
None beyond the deviations above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- The nine-state guidance application ledger and the helpful-application promotion gate are both live in production code paths (`recordDispatchWorkerOutcome`, `recordPhaseApplicationCredit`, `pkg/memory.ConsolidationService.Run`), not merely declared.
- No known blockers for phase closure from this plan.

## Self-Check: PASSED
- `cmd/application_evidence.go`, `cmd/application_evidence_test.go`, `pkg/memory/consolidate.go`, `pkg/memory/instinct_stats.go` all present on disk with the declared symbols (verified via grep).
- `git log --oneline --all --grep="204-06"` returns 4 commits (5c096b9e, 0f472254, e57d2f72, bef335f2).
- Every acceptance-criteria `<verify>` command from the plan re-run clean immediately before this summary was written.

---
*Phase: 204-learning-governor*
*Completed: 2026-09-14*
