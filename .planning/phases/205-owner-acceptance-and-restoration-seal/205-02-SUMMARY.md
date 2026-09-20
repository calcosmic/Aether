---
phase: 205-owner-acceptance-and-restoration-seal
plan: 02
subsystem: seal
tags: [seal, queen, prompt-integrity, handoff-contract, worktree]

requires:
  - phase: 198.1
    provides: worker-authored text sanitization pattern used elsewhere in the memory pipeline (198.1, CR-02)
provides:
  - Seal review briefs state the handoff contract the finalizer enforces
  - Seal-promoted lessons are filtered as untrusted input before reaching QUEEN.md
affects: [205-owner-acceptance-and-restoration-seal, seal, entomb]

actuals:
  tokens: 4365
  tasks: 3
  commits: 4

tech-stack:
  added: []
  patterns:
    - "External-lane worker briefs append the handoff schema sentence at the dispatch call site, not inside the shared brief renderer, to avoid double-delivery to native-lane workers (continue/build/seal now all follow this shape)"
    - "Worker-reported free text destined for an owner-facing store is run through pkg/colony's shared content-integrity detector (DetectPromptIntegrityFindings + SanitizeSignalContent) rather than a private per-caller pattern list"

key-files:
  created:
    - cmd/seal_final_review_contract_test.go
  modified:
    - cmd/seal_final_review.go
    - cmd/queen.go
    - pkg/colony/prompt_integrity.go

key-decisions:
  - "Re-verified both field-report defects against current source before touching anything (Task 1's action instruction); both were confirmed still open, so both tasks were real fixes, not verification-only"
  - "Added one new content-integrity rule (secretsPathRuleSpecs) to pkg/colony's shared detector rather than a private regex list in cmd/queen.go, because the field report's exact unsafe lesson (a .env.local-copying shell command) matched none of the four existing shellInjectionRuleSpecs — extending the shared detector protects every caller (pheromone signals too), matching the plan's explicit prohibition on a private list"
  - "sanitizeQueenPromotedLesson is a new, additional function alongside sanitizeQueenInline rather than a modification to it, because sanitizeQueenInline's other existing callers (promoteInstinctLocal) promote runtime-computed text, not raw worker prose, and must keep their whitespace-collapse-only behaviour unchanged (verified by TestQueenSanitizeInlineCallersUnchanged and the required TestQueen* regression pass)"

requirements-completed: []  # PROOF-05 is shared with a sibling plan in this phase still in flight; requirements.ready-ids reported it blocked, so it is left for the last declaring plan to mark complete

coverage:
  - id: D1
    description: "A seal reviewer's brief states the exact handoff result-shape the finalizer enforces, from the same single-source constants (codex.HandoffFieldsSummary, codex.HandoffOpenDecisionsGuidance) the continue and build lanes already use"
    requirement: "PROOF-05"
    verification:
      - kind: unit
        ref: "cmd/seal_final_review_contract_test.go#TestSealFinalReviewBriefCarriesHandoffSchema"
        status: pass
      - kind: unit
        ref: "cmd/seal_final_review_contract_test.go#TestSealFinalReviewBriefHandoffSchemaCoversEveryQueenSelectedCaste"
        status: pass
    human_judgment: false
  - id: D2
    description: "An unsafe worker-reported seal lesson (shell command, prompt-injection phrase, XML tag, over-length text) is silently dropped before it can reach QUEEN.md, while an ordinary lesson is kept with whitespace collapsed, and a fully-refused batch still returns a successful seal result"
    requirement: "PROOF-05"
    verification:
      - kind: unit
        ref: "cmd/seal_final_review_contract_test.go#TestSealPromotedLessonsAreFiltered"
        status: pass
      - kind: unit
        ref: "cmd/seal_final_review_contract_test.go#TestSealSucceedsWhenEveryLessonIsRefused"
        status: pass
      - kind: unit
        ref: "cmd/seal_final_review_contract_test.go#TestQueenSanitizeInlineCallersUnchanged"
        status: pass
    human_judgment: false

duration: 20min
completed: 2026-09-15
status: complete
---

# Phase 205 Plan 02: Seal Handoff Contract and Lesson Sanitization Summary

**Seal review briefs now state the finalizer's handoff contract, and seal-promoted "lessons" are filtered through the shared prompt-integrity detector (plus a new secrets-file-path rule) before they can reach the owner-facing QUEEN.md.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-09-15T08:36:00Z (approx.)
- **Completed:** 2026-09-15T08:54:46Z
- **Tasks:** 3
- **Files modified:** 4 (1 created, 3 modified)

## Accomplishments

- `sealExternalBriefWithHandoffSchema` (`cmd/seal_final_review.go`) appends `codex.HandoffFieldsSummary` and `codex.HandoffOpenDecisionsGuidance` to every seal review dispatch's brief, at the same structural call site (`plannedSealFinalReviewDispatches`) the continue lane (`continueExternalBriefWithHandoffSchema`) and build lane (`composeBuildManifestBrief`) already use for their own external dispatches — so a seal reviewer that follows its brief exactly now returns a handoff the finalizer (`mergeExternalSealReviewResults` → `mergeExternalContinueResults` → `ValidateWorkerHandoff`) accepts on the first attempt.
- `sanitizeQueenPromotedLesson` (`cmd/queen.go`) treats a worker-reported seal lesson as untrusted input: whitespace-collapse first, then a pass through `pkg/colony`'s shared content-integrity detector (`DetectPromptIntegrityFindings`) and the shared sanitizer's length ceiling (`SanitizeSignalContent`). `writeSealReusableLessonsToQueen` now calls it instead of `sanitizeQueenInline` alone, skipping a refused lesson with the same silent shape the pre-existing empty-string case already used.
- `secretsPathRuleSpecs` (`pkg/colony/prompt_integrity.go`) closes a real gap in the shared detector: the field report's exact unsafe lesson (`cd .../dashboard && cp .../dashboard/.env.local . 2>/dev/null; npm install ...`) has neither a pipe/semicolon `rm`, a backtick, nor a `$()` substitution — none of the four pre-existing `shellInjectionRuleSpecs` entries matched it. The new rule is added to the one shared detector every caller (pheromone signals via `SanitizeSignalContent`, and now seal-promoted lessons) already runs through, per the plan's explicit instruction not to add a private pattern list in `cmd/queen.go`.

## Task Commits

Each task was committed with a RED/GREEN pair (tdd="true"):

1. **Task 1: Seal reviewer brief carries the handoff contract**
   - `a1f170e2` — `test(205-02): add failing test for seal handoff schema` (RED — compile failure: `sealExternalBriefWithHandoffSchema` undefined)
   - `123424d2` — `feat(205-02): seal review brief carries handoff schema` (GREEN)
2. **Task 2: Worker-reported lessons are untrusted input**
   - `fec31873` — `test(205-02): add failing test for seal lesson sanitization` (RED — observed failing: "expected exactly 1 lesson promoted from a mixed batch, got 5" and "expected zero promoted count for a fully-refused batch, got 3")
   - `827ef554` — `feat(205-02): treat seal-promoted lessons as untrusted input` (GREEN)
3. **Task 3: Regression scope check** — verification-only, no code changes; no commit (see Deviations)

**Plan metadata:** this commit (docs)

## Files Created/Modified

- `cmd/seal_final_review_contract_test.go` — new file; `TestSealFinalReviewBriefCarriesHandoffSchema`, `TestSealFinalReviewBriefHandoffSchemaCoversEveryQueenSelectedCaste`, `TestSealPromotedLessonsAreFiltered`, `TestSealSucceedsWhenEveryLessonIsRefused`, `TestQueenSanitizeInlineCallersUnchanged`
- `cmd/seal_final_review.go` — `sealExternalBriefWithHandoffSchema` added; `plannedSealFinalReviewDispatches` wraps `TaskBrief` with it; `writeSealReusableLessonsToQueen` calls `sanitizeQueenPromotedLesson` instead of `sanitizeQueenInline`
- `cmd/queen.go` — `sanitizeQueenPromotedLesson` added (new `pkg/colony` import); `sanitizeQueenInline` and its other callers unchanged
- `pkg/colony/prompt_integrity.go` — `secretsPathRuleSpecs`, `secretsPathRules`, and a third finding loop added to `DetectPromptIntegrityFindings`

## Decisions Made

See `key-decisions` in frontmatter.

## Deviations from Plan

### Auto-fixed Issues

None — plan executed exactly as written; both defects were re-verified as still open before any change (per Task 1's action instruction), so no deviation from the plan's own instructions was needed.

### Process note (not a Rule 1-4 deviation)

Task 3 is a verification-only task by design ("no edit unless a regression is found") — no regression was found, so `cmd/queen.go` was read and re-verified but not edited a third time, and no additional commit was made for Task 3. This matches the plan's own file annotation: `cmd/queen.go (read and verified only — no edit unless a regression is found)`.

---

**Total deviations:** 0
**Impact on plan:** None. Both fixes landed exactly as scoped; no scope creep, no architectural changes.

## Issues Encountered

The scoped `go test ./cmd/ -run 'Seal|Queen|Sanitize|Handoff' -count=1 -timeout 90m` run took 562s (vs. typically much faster for a scoped run) because five sibling worktree executors were running their own test suites concurrently on this machine at the same time, consistent with the known machine-level contention noted in project memory ("Suite ceiling measured"). It completed cleanly (exit 0, no failures) — this was a wall-clock/contention issue, not a test failure.

## Task 3: Regression Scope Check Results

Commands run, per the plan's `<verify>`:

```
go test ./cmd/ -run 'Seal|Queen|Sanitize|Handoff' -count=1 -timeout 90m   → ok  github.com/calcosmic/Aether/cmd       562.828s  (0 failures)
go test ./pkg/colony/ -run 'Seal|Queen|Sanitize|Handoff' -count=1 -v -timeout 90m → ok  github.com/calcosmic/Aether/pkg/colony  0.312s  (0 failures, includes existing TestSanitizeSignalContent_* suite, TestRepoPlaceholderSurvivesSanitizer, TestLifecyclePauseHandoff, TestLifecycleSealOutcome, TestPlanningQueenRecommendationIsTypedAdviceOnly)
go test ./cmd/ -run 'TestQueen' -count=1 -timeout 90m                     → ok  (0 failures) -- acceptance criterion for Task 2 (existing sanitizeQueenInline callers unchanged)
go build ./cmd/aether                                                     → exit 0
go vet ./cmd/... ./pkg/colony/...                                         → exit 0
```

No failures observed in any scoped run — nothing to classify against the 2026-09-14 known-red baseline (`TestAuditCatalogGolden`, `TestBuildStartLegacyHelpersRetired200`, `TestCodexBuildPlanOnlySpawnBudgetSeparatesCasteBudgetFromWorkerCount`, `TestCompletionPacketSchemaMatchesStructs`, `TestCurrentVocabulary199`, `TestFailedCheckSendsExactlyOneBuilderFixAttempt`, `TestFixAttemptIsCountedSeparately`, `TestFixAttemptNeverOverwritesTheFirstResult`, `TestGoldenBuildVisualOutput`, `TestGoldenContinueVisualOutput`, `TestGoSourceHintsMatchCobraContracts`, `TestHumanFacingOutputGoesThroughWriteVisualOutput`, `TestNoSecondAutomaticFixAttempt`, `TestPhase199GateReceipt`, `TestPlanningAdversarial200`, `TestPlanningPublicPaths200`, `TestResolveTestCommand_GoProject`) — none of those names match the `Seal|Queen|Sanitize|Handoff` pattern, and the scoped runs found zero failures regardless.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Both of the 2026-09-14 field report's seal-lane defects (finding 1: seal reviewers not told the handoff contract; finding 6: unsafe lessons promoted unfiltered into QUEEN.md) are closed and locked by tests. Sibling plans in this wave/phase cover the rest of the 205 owner-acceptance walkthrough. `PROOF-05` stays unmarked in REQUIREMENTS.md pending its sibling declaring plan(s) finishing (shared-ID gate, `requirements.ready-ids` reported it `blocked`, not `ready`) — no action needed here, the orchestrator's next `update_requirements` pass over this phase will pick it up once all declaring plans are done.

## Self-Check: PASSED

- `cmd/seal_final_review_contract_test.go` exists: FOUND
- `cmd/seal_final_review.go` contains `sealExternalBriefWithHandoffSchema`: FOUND
- `cmd/queen.go` contains `sanitizeQueenPromotedLesson`: FOUND
- `pkg/colony/prompt_integrity.go` contains `secretsPathRuleSpecs`: FOUND
- Commits `a1f170e2`, `123424d2`, `fec31873`, `827ef554` all present in `git log --oneline --all`: FOUND
- `go test ./cmd/ -run 'TestSealFinalReviewBriefCarriesHandoffSchema|TestSealPromotedLessonsAreFiltered|TestSealSucceedsWhenEveryLessonIsRefused' -count=1 -timeout 90m` exits 0: PASSED
- `go build ./cmd/aether` exits 0: PASSED
- `go vet ./cmd/... ./pkg/colony/...` exits 0: PASSED

---
*Phase: 205-owner-acceptance-and-restoration-seal*
*Completed: 2026-09-15*
