---
phase: 204-learning-governor
plan: "02"
subsystem: learning
tags: [credit-ledger, recruitment-credit, hypothesis-labeling, colony-prime, autopilot-lessons, learning-status-vocabulary]

# Dependency graph
requires:
  - phase: 204-01
    provides: "204-CLASSIC-SYNTHESIS.md rulings (a) and (b), naming plan 204-02 as the owner of both fixes this plan makes, plus the phase's own tracer designation"
provides:
  - "recordPhaseApplicationCredit: the first real production writer into cmd/recruitment_credit.go's evidence-gated credit ledger, reached from both check lanes via runPhaseEndConsolidation"
  - "learningVerifiedEntries/learningUnverifiedEntries: the one shared, status-aware vocabulary every verified-labelled render path now calls"
affects: [204-03, 204-04, 204-05, 204-06, 204-07, 204-08, 204-09, 204-10, 204-11]

actuals:
  tokens: 14810
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "AST call-graph reachability guard (buildCmdFuncGraph/reachableFrom reused from cmd/worktree_destruction_reachability_test.go) applied to a NEW property: a writer function reachable from both check-lane entry functions, not a destruction-safety gate"
    - "AST-based 'no second predicate' structural guard (learningStatusPredicateViolationsInFile) that recognizes both a direct == comparison and a call-argument comparison shape, permitting exactly two named functions"
    - "Deterministic content-derived identifiers (phaseApplicationDecisionID/phaseApplicationEffectEvidenceID) following recruitmentCreditRecordID's own idiom -- attempt ID + index, never a fresh UUID or timestamp"

key-files:
  created:
    - cmd/application_evidence.go
    - cmd/application_evidence_test.go
    - cmd/learning_status_vocabulary.go
    - cmd/learning_status_vocabulary_test.go
  modified:
    - cmd/consolidation_lifecycle.go
    - cmd/recruitment_credit.go
    - cmd/colony_prime_context.go
    - cmd/autopilot_lessons.go
    - cmd/autopilot_lessons_test.go
    - cmd/capsule_writer_invariant_198_2_test.go
    - cmd/colony_prime_context_test.go

key-decisions:
  - "recordPhaseApplicationCredit derives outcome from a cross-product of every delivered-instinct contribution against every decision-kind knowledge delta on the phase's latest attempt (per the plan's own 'for each (contribution, decision) pair' instruction), not a 1:1 pairing -- with 1 contribution and 1 decision in every current call site, this is observationally identical to 1:1 but matches the plan's literal derivation rule for phases with multiple decisions."
  - "TestTunerSeesGenuinelyProducedCredit asserts on tuneNoteStrengthFromOutcomes's Ran=true plus the shared reader (pheromoneOutcomeReadCreditRecords) returning non-empty, rather than on tuneNoteStrengthFromOutcomes's own RecordsConsidered field -- that field is scoped to note-kind contributions only (cmd/pheromone_outcome.go, outside this plan's declared files_modified), and this tracer writes memory-item kind. The assertion still proves the previously-always-empty store is now genuinely fed by production data, without touching a file outside the plan's declared scope."
  - "colony_prime_context.go's two learned-memory sections are built via a closure that RETURNS the computed content/scores rather than appending to `sections` itself, so each call site still writes its own colonyPrimeSection{name: \"<literal>\", ...} composite literal -- cmd/capsule_writer_invariant_198_2_test.go's memoryPackPartNames mechanically discovers every memory-pack part by scanning for exactly that literal-string name: field shape, so the two part names could not be routed through a shared variable."

patterns-established:
  - "A tracer plan's two tasks each land as their own atomic commit (feat: writer wiring, feat: status vocabulary), both verified end-to-end before the next begins, per this plan's own tracer-feedback-gate instruction."

requirements-completed: [LEARN-03, LEARN-01]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "recordPhaseApplicationCredit: the first real production writer into the evidence-gated credit ledger, wired into runPhaseEndConsolidation (both check lanes), deriving decision/effect identifiers and outcome from the phase's own real durable build-attempt evidence."
    requirement: "LEARN-03"
    verification:
      - kind: unit
        ref: "cmd/application_evidence_test.go#TestPhaseApplicationCreditTracerEndToEnd (5 subtests: helpful, neutral, harmful, pending, no-record)"
        status: pass
      - kind: unit
        ref: "cmd/application_evidence_test.go#TestPhaseApplicationCreditIsReachedFromBothCheckLanes"
        status: pass
      - kind: unit
        ref: "cmd/application_evidence_test.go#TestPhaseApplicationCreditIsReplaySafe"
        status: pass
      - kind: unit
        ref: "cmd/application_evidence_test.go#TestTunerSeesGenuinelyProducedCredit"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_credit_test.go#TestCreditRequiresBothFacts (still passes, unmodified writer guard)"
        status: pass
    human_judgment: false
  - id: D2
    description: "learningVerifiedEntries/learningUnverifiedEntries: one shared status-aware vocabulary governing both render/admission paths that label learning content verified (cmd/colony_prime_context.go's worker capsule, cmd/autopilot_lessons.go's confirmedAutopilotLessonsSincePlan), with a structural guard forbidding a second predicate."
    requirement: "LEARN-01"
    verification:
      - kind: unit
        ref: "cmd/learning_status_vocabulary_test.go#TestHypothesisIsNeverRenderedAsVerified (all-hypothesis, mixed, disproven subtests)"
        status: pass
      - kind: unit
        ref: "cmd/learning_status_vocabulary_test.go#TestUnverifiedEntriesAreStillShown"
        status: pass
      - kind: unit
        ref: "cmd/learning_status_vocabulary_test.go#TestAutopilotLessonsRequireValidatedStatus"
        status: pass
      - kind: unit
        ref: "cmd/learning_status_vocabulary_test.go#TestOneLearningStatusVocabulary"
        status: pass
    human_judgment: false

duration: 80min
completed: 2026-09-14
status: complete
---

# Phase 204 Plan 02: Learning Governor Tracer Summary

**Wired the first real production writer into the previously-unreached evidence-gated credit ledger, and replaced the single mislabeled "Verified Outcomes" learned-memory section with two status-honest sections so a hypothesis is never shown to a worker or owner as proven.**

## Performance

- **Duration:** ~80 min
- **Started:** ~2026-09-14T11:20:00Z (estimated from session context load)
- **Completed:** 2026-09-14T12:39:41+02:00
- **Tasks:** 2/2 completed
- **Files modified:** 11 (4 created, 7 modified)

## Accomplishments

- `cmd/application_evidence.go`'s `recordPhaseApplicationCredit` earns evidence-gated credit for every instinct a phase's delivery ledger recorded as delivered, against real changed-decision and free-check-comparison evidence read from the phase's own durable build attempt -- never on delivery or a passing phase alone. Wired into `runPhaseEndConsolidation`, the single call site both `cmd/codex_continue.go` and `cmd/codex_continue_finalize.go` already invoke, proved reachable from each by a real AST call-graph guard with a non-vacuous synthetic negative fixture.
- The inline owner-facing "A note changed a decision" line (`cmd/codex_visuals.go`'s `renderInlineDecisionChangedLine`, wired in `cmd/recruitment_credit.go`) now also fires for a memory-item contribution, with no second line shape.
- `cmd/learning_status_vocabulary.go` is now the one place the package decides what counts as verified learning content. `cmd/colony_prime_context.go`'s worker capsule renders two sections -- `## LEARNED MEMORY (Verified Outcomes)` for genuinely validated entries, and a new honestly-worded `## LEARNED MEMORY (Not Yet Verified -- Unchecked Observations)` for hypothesis/empty-status entries -- sharing one combined 20-entry cap so the context budget does not grow. A disproven entry appears under neither.
- `cmd/autopilot_lessons.go`'s `confirmedAutopilotLessonsSincePlan` now requires `learningVerifiedEntries` admission instead of its own inline status exclusion; every other admission rule (phase floors, blocked classification, revision boundary, gates, worker completion) is unchanged.
- A structural AST guard (`TestOneLearningStatusVocabulary`) forbids any function outside the two named helpers from comparing against the `learn.Status*` vocabulary, proven non-vacuous against a synthetic fixture.

## Task Commits

Each task was committed atomically:

1. **Task 1: End-to-end "a remembered lesson earns a real outcome" (tracer)** - `e6b24de1` (feat)
2. **Task 2: One status vocabulary, so an untested guess is never labelled verified** - `e1983cbf` (feat)

**Plan metadata:** committed alongside this SUMMARY (worktree mode -- STATE.md/ROADMAP.md excluded, handled by the orchestrator).

## Files Created/Modified

- `cmd/application_evidence.go` - `recordPhaseApplicationCredit`, `phaseApplicationDecisionID`, `phaseApplicationEffectEvidenceID`, `phaseApplicationEffectAndOutcome`, `phaseApplicationCreditSummary`
- `cmd/application_evidence_test.go` - Tracer end-to-end proof (5 outcome subtests), both-check-lanes call-graph guard, replay-safety proof, tuner-reader proof
- `cmd/consolidation_lifecycle.go` - One call to `recordPhaseApplicationCredit(phaseID)` in `runPhaseEndConsolidation`, after `recordInstinctApplicationsForPhase`
- `cmd/recruitment_credit.go` - Inline decision-changed line widened to fire for `recruitmentContributionMemoryItem` too
- `cmd/learning_status_vocabulary.go` - `learningVerifiedEntries`, `learningUnverifiedEntries`, `learningEntryStatusEquals`, `learnedMemoryVerifiedHeading`, `learnedMemoryUnverifiedHeading`
- `cmd/learning_status_vocabulary_test.go` - Status-placement render tests, autopilot admission test, structural "one vocabulary" guard
- `cmd/colony_prime_context.go` - Split the single learned-memory section into two (verified/unverified), sharing one combined entry cap
- `cmd/autopilot_lessons.go` - `confirmedAutopilotLessonsSincePlan` now calls `learningVerifiedEntries` instead of its own inline `learn.StatusDisproven` comparison
- `cmd/autopilot_lessons_test.go` - Shared `autopilotLessonFixture` now uses `learn.StatusValidated` (a "valid" fixture must genuinely pass every admission rule, including status)
- `cmd/capsule_writer_invariant_198_2_test.go` - Added the `learned_memory_unverified` writer-map entry the new part requires
- `cmd/colony_prime_context_test.go` - Three raw `entries.json` fixtures gained an explicit `"status": "validated"` so template-rendering tests keep exercising the verified section as originally intended

## Decisions Made

See `key-decisions` in frontmatter above for the three substantive ones (cross-product derivation, the tuner-test assertion shape, and the closure-return-value pattern that preserves the AST-discoverable literal `name:` fields).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `cmd/colony_prime_context.go`'s original refactor broke `memoryPackPartNames`'s mechanical AST discovery**
- **Found during:** Task 2, first full-suite verification pass
- **Issue:** A first-draft shared closure took the section name as a parameter and wrote `colonyPrimeSection{name: sectionName, ...}` -- `sectionName` is an `*ast.Ident`, not a string literal, so `cmd/capsule_writer_invariant_198_2_test.go`'s `memoryPackPartNames` (which scans for literal-string `name:` fields only, by design, per its own doc comment) stopped discovering the `learned_memory` part at all, failing `TestEveryMemoryPackPartHasALiveWriter` with an "orphaned writer-map entry" report.
- **Fix:** Restructured so the closure returns computed content/scores by value; each of the two call sites still writes its own `colonyPrimeSection{name: "learned_memory", ...}` / `colonyPrimeSection{name: "learned_memory_unverified", ...}` literal. Added the required `learned_memory_unverified` entry to `memoryPackPartWriters`.
- **Files modified:** `cmd/colony_prime_context.go`, `cmd/capsule_writer_invariant_198_2_test.go`
- **Verification:** `go test ./cmd -run '^TestEveryMemoryPackPartHasALiveWriter$'` passes.
- **Committed in:** `e1983cbf`

**2. [Rule 1 - Bug] Pre-existing test fixtures assumed any non-disproven status was verified-admissible**
- **Found during:** Task 2, full-suite verification pass
- **Issue:** Splitting the single learned-memory section by status broke tests that predate the status distinction: `TestColonyPrimeTemplateLoad`/`TestColonyPrimeLearnedMemoryTemplate` (raw `entries.json` fixtures with no `status` field, expecting content under the custom "learned_memory" template) and `TestConfirmedAutopilotLessonsEligibility`/`Deduplicates.../UsesGeneratedAt...` (the shared `autopilotLessonFixture` used `Status: learn.StatusHypothesis` to represent "admissible", which the new admission rule correctly no longer accepts).
- **Fix:** Added explicit `"status": "validated"` to the three raw fixture literals in `cmd/colony_prime_context_test.go`, and changed `autopilotLessonFixture`'s `Status` to `learn.StatusValidated` -- both fixtures' actual intent (testing template rendering / phase-and-gates admission, not status filtering) is preserved, now representing a genuinely admissible entry rather than a merely non-disproven one.
- **Files modified:** `cmd/colony_prime_context_test.go`, `cmd/autopilot_lessons_test.go`
- **Verification:** `go test ./cmd -run '^TestColonyPrime'` and `go test ./cmd -run '^TestConfirmedAutopilotLessons'` both pass.
- **Committed in:** `e1983cbf`

**3. [Rule 3 - Blocking] A tracer test subtest's instinct content failed admissibility**
- **Found during:** Task 1, first test run
- **Issue:** `TestPhaseApplicationCreditTracerEndToEnd`'s "pending" subtest used instinct content ("run gofmt -l cmd/ before committing...") that `memory.IsAdmissibleInstinctContent` rejected -- it named no recognized file/command/error pattern.
- **Fix:** Reworded to a `go`-prefixed command matching the admissibility regex, consistent with the other subtests.
- **Files modified:** `cmd/application_evidence_test.go`
- **Verification:** `go test ./cmd -run '^TestPhaseApplicationCreditTracerEndToEnd$'` passes.
- **Committed in:** `e6b24de1`

**4. [Rule 3 - Blocking] `cmd/consolidation_lifecycle.go`'s own doc comment double-counted the acceptance-criteria grep**
- **Found during:** Task 1, acceptance-criteria verification
- **Issue:** The plan's `<acceptance_criteria>` requires `grep -c 'recordPhaseApplicationCredit' cmd/consolidation_lifecycle.go` to equal exactly 1 (one line). A first-draft doc comment above the call site repeated the function name a second time, failing the count.
- **Fix:** Reworded the comment to reference the writer's own doc comment in `cmd/application_evidence.go` instead of repeating the identifier.
- **Files modified:** `cmd/consolidation_lifecycle.go`
- **Verification:** `grep -c 'recordPhaseApplicationCredit' cmd/consolidation_lifecycle.go` returns `1`.
- **Committed in:** `e6b24de1`

---

**Total deviations:** 4 auto-fixed (2 Rule 1 bug fixes surfaced by the status-vocabulary change breaking pre-existing test assumptions, 2 Rule 3 blocking fixes to meet acceptance criteria exactly). **Impact on plan:** None on any machine-checkable gate -- every `<acceptance_criteria>` line and both tasks' `<verify>` commands pass; the fixes are entirely in test fixtures and doc-comment wording, with zero production-behavior change beyond what the plan itself specified.

## Issues Encountered

None beyond the deviations documented above. `go build ./...`, `go vet ./cmd/... ./pkg/...` are clean. Full Task 1 + Task 2 verify set (13 named tests, several with subtests) passes together. A broader sweep (`TestColonyPrime*`, `TestAutopilot*`, `TestLearning*`, `TestMemoryPack*`, `TestConfirmed*`, `TestCapsule*`) also passes, confirming no wider regression from the status-vocabulary change.

## User Setup Required

None -- no external service configuration required.

## Next Phase Readiness

Ruling (a) and ruling (b) from `204-CLASSIC-SYNTHESIS.md` are both closed: the credit ledger now has a real production writer reachable from both check lanes, and no render path can show an unverified guess as proven. Plan `204-03` onward can build on `recordPhaseApplicationCredit`'s established pattern (real evidence, never inferred) and `learningVerifiedEntries`'s shared vocabulary (any new "is this verified?" question should call it, not invent a second predicate -- the structural guard will catch a violation by name).

No blockers.

## Self-Check: PASSED

- All 11 key files confirmed present via `git ls-files` (both created and modified files are tracked on disk).
- Both task commits (`e6b24de1`, `e1983cbf`) confirmed present via `git log --oneline --all`.
- All acceptance-criteria grep checks re-run clean: `recordPhaseApplicationCredit` appears exactly once in `cmd/consolidation_lifecycle.go`; `cmd/application_evidence.go` contains no `credit/records.json` literal; `Verified Outcomes` appears in exactly one non-test file (`cmd/learning_status_vocabulary.go`); `cmd/autopilot_lessons.go` contains zero `learn.Status(Disproven|Hypothesis)` references.
- Full Task 1 + Task 2 named `<verify>` test set (13 test functions, several with subtests) passes together: `go test ./cmd -run '^(TestPhaseApplicationCreditTracerEndToEnd|TestPhaseApplicationCreditIsReachedFromBothCheckLanes|TestPhaseApplicationCreditIsReplaySafe|TestTunerSeesGenuinelyProducedCredit|TestCreditRequiresBothFacts|TestNoteStrengthTuningHelpfulNeutralHarmfulMovements|TestNoteStrengthTuningQuarantinesOnHarmfulThreshold|TestPinnedNoteIsNeverTuned|TestTuningPassIsFreeWithoutCredit|TestHypothesisIsNeverRenderedAsVerified|TestUnverifiedEntriesAreStillShown|TestAutopilotLessonsRequireValidatedStatus|TestOneLearningStatusVocabulary)$' -count=1 -timeout 90m` -> `ok`.
- `go build ./...` and `go vet ./cmd/...` both clean.
- Broader regression sweep (`TestColonyPrime*`, `TestAutopilot*`, `TestLearning*`, `TestMemoryPack*`, `TestConfirmed*`, `TestCapsule*`) passes.
- Requirement IDs `LEARN-03`/`LEARN-01` are shared with other plans in this phase not yet complete (`requirements.ready-ids` reports both `blocked`) -- correctly NOT marked complete in `REQUIREMENTS.md` per the shared-ID gate; the phase-final plan will close them once every declaring plan finishes.

---
*Phase: 204-learning-governor*
*Completed: 2026-09-14*
