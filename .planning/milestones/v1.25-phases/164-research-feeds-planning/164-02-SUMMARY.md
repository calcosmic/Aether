---
phase: 164-research-feeds-planning
plan: 02
subsystem: planning
tags: [go, queen-decision, pending-decision, research-proposal]

requires:
  - phase: 163-context-reaches-workers
    provides: codexSurveyContext plumbing and territory survey data available at plan time
provides:
  - computePhaseResearchProposal (per-phase research/skip recommendation with grounded, non-empty reason)
  - phaseResearchProposal / phaseResearchRecommendation / phaseResearchHint data structures
  - renderPhaseResearchProposalBlock (D-01 one-approve-all, per-phase-flip tick-to-approve batch)
  - newPhaseResearchDecision / phaseResearchDecisionResolution / resolvePhaseResearchDecisions (decision-record builders on the existing PendingDecision store)
affects: [164-05, research-dispatch-wiring]

tech-stack:
  added: []
  patterns:
    - "Runtime hint computation grounded in deterministic signals (external tech mentions, survey-gap detection, phase mode), never in prose keyword lists"
    - "Fixed reason-string lookup table with only the matched signal token interpolated, mirroring getSmartDefaultReason's shape, to prevent prompt/text injection via crafted phase descriptions"
    - "Decisions expressed as the existing PendingDecision type -- no new struct, no new JSON store"

key-files:
  created:
    - cmd/phase_research_decision.go
    - cmd/phase_research_decision_test.go
  modified: []

key-decisions:
  - "phaseResearchExternalSignals is a new, independent signal list -- deliberately not reusing researchPhaseKeywords/isResearchPhase (cmd/plan_grounding.go), which is a Phase 167 removal target for a different gate (grounding exemption)"
  - "DomainGaps derived by checking each ExternalTech token against every individual lowercased survey entry (not a single concatenated string), matching the plan's literal per-entry substring spec"
  - "extractResearchDirection checks 'skip' before 'research' in the resolution tail, because a flip-to-skip resolution's text also names 'research' as the noun being skipped (e.g. 'skip research on phase N') -- checking skip first yields the direction that actually took effect"

patterns-established:
  - "phaseResearchReasons lookup table: reason text is built from a fixed template with only the matched signal token interpolated, never the raw phase description, closing the reason-line injection surface (T-164-04)"

requirements-completed: [RESEARCH-01, RESEARCH-02]

duration: 20min
completed: 2026-08-02
---

# Phase 164 Plan 02: Grounded Research Decision + Batch Card Summary

**New `cmd/phase_research_decision.go`: `computePhaseResearchProposal` grounds a research/skip recommendation per phase in runtime-computed signals (not keyword lists), and a set of pure builder functions render the D-01 one-approve-all tick-to-approve batch and express decisions/overrides as existing `PendingDecision` records.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-08-02T13:01Z (plan commit)
- **Completed:** 2026-08-02T13:19Z (last task commit)
- **Tasks:** 2 completed
- **Files modified:** 2 (both newly created)

## Accomplishments

- Every candidate phase now receives a research/skip recommendation with a non-empty, plain-English, grounded reason -- proven independent of `researchPhaseKeywords`/`isResearchPhase` by an explicit test case
- A fast run recommends skip for every phase while still listing every phase, satisfying D-15's "flip it on" contract
- The batch renders as exactly one `--approve-all` command with one `--flip N` line per phase, mirroring `renderPendingSuggestionsBlock`'s existing UX shape
- A user flip (either direction) is expressible entirely through the existing `PendingDecision` type -- no new struct, no new store file name introduced (RESEARCH-06)

## Task Commits

Each task was committed atomically:

1. **Task 1: Grounded per-phase research recommendation with a reason** - `e18a01e6` (feat)
2. **Task 2: Batch card rendering and decision-record builders** - `1adcbfed` (feat)

_Note: Both tasks were TDD-tagged in the plan; tests were written alongside implementation in the same commit per task rather than as separate RED/GREEN commits, since each task's `<action>` and `<behavior>` were implemented together and verified against the specified test names before committing. All specified test functions (`TestQueenResearchDecision`, `TestResearchProposalBatchRendersTickToApprove`, `TestResearchDecisionRecordsUseExistingStore`) exist and pass._

## Files Created/Modified

- `cmd/phase_research_decision.go` - `computePhaseResearchProposal`, hint computation (`computePhaseResearchHint`, `surveyContainsSignal`), the `phaseResearchReasons` lookup table, `renderPhaseResearchProposalBlock`, and the `PendingDecision` builder/resolver trio (`newPhaseResearchDecision`, `phaseResearchDecisionResolution`, `resolvePhaseResearchDecisions`)
- `cmd/phase_research_decision_test.go` - `TestQueenResearchDecision` (6 subtests covering grounded recommendation behavior), `TestResearchProposalBatchRendersTickToApprove`, `TestResearchDecisionRecordsUseExistingStore` (7 subtests covering decision-record and override behavior)

## Decisions Made

- Kept `phaseResearchExternalSignals` and `phaseResearchReasons` as new package-level vars entirely separate from `cmd/plan_grounding.go`'s keyword surface, per the plan's explicit instruction and the anti-pattern warning in 164-RESEARCH.md
- Chose to check "skip" before "research" when deriving direction from a resolution string's tail text, since a flip-to-skip resolution's literal text also contains the word "research" (as in "skip research on phase N") -- checking skip first is what makes `resolvePhaseResearchDecisions` return the direction that actually took effect
- `renderPhaseResearchProposalBlock` returns the empty string for a zero-phase proposal per the plan's explicit spec, rather than a header-only string

## Deviations from Plan

None - plan executed exactly as written. Both tasks' `<action>` sections were followed literally: types, function signatures, the `phaseResearchExternalSignals` list contents, the `phaseResearchReasons` map keys, and all four Task 2 function signatures match the plan text exactly.

## Issues Encountered

None. `gofmt -w` was run once to fix struct field alignment after the initial write (routine formatting, not a deviation from plan content).

## User Setup Required

None - no external service configuration required. This plan produces pure Go functions with no cobra registration and no store I/O; Plan 05 wires this into the manifest and CLI.

## Next Phase Readiness

- `computePhaseResearchProposal`, `renderPhaseResearchProposalBlock`, and the `PendingDecision` builder/resolver functions are ready for Plan 05 to wire into `cmd/codex_plan.go`'s manifest construction and a new `aether plan-research-approve` CLI command
- No blockers. The full `cmd` package test suite (`go test ./cmd/... -count=1`) passes, `go vet ./...` is clean across the whole repo, and `cmd/phase_research_decision.go` compiles without importing anything from `cmd/plan_grounding.go`'s keyword surface (verified via grep in acceptance criteria)

---
*Phase: 164-research-feeds-planning*
*Completed: 2026-08-02*

## Self-Check: PASSED

- FOUND: cmd/phase_research_decision.go
- FOUND: cmd/phase_research_decision_test.go
- FOUND: .planning/phases/164-research-feeds-planning/164-02-SUMMARY.md
- FOUND: commit e18a01e6 (Task 1)
- FOUND: commit 1adcbfed (Task 2)
- FOUND: commit 0c3e7164 (SUMMARY)
