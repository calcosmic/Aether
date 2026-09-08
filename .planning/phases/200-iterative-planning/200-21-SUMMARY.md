---
phase: 200-iterative-planning
plan: 21
subsystem: planning
tags: [planning, wrappers, scout, route-setter, presets, candidates, claude, opencode]

requires:
  - phase: 200-13
    provides: explicit four-preset selection and Scout-first stage authorization
  - phase: 200-14
    provides: Scout receipt, first-pass decision batch, and exact Route-Setter authorization
  - phase: 200-15
    provides: Route-Setter receipt, iteration card, continuation, and candidate boundaries
  - phase: 200-16
    provides: reviewable non-active candidates and exact receipt-bound acceptance
  - phase: 200-19
    provides: canonical planning terminal and JSON projections
  - phase: 200-20
    provides: Discuss-to-draft-Specification handoff and material-decision presentation contract
provides:
  - Canonical plan wrapper contract for approved-Specification preflight and unbiased four-preset selection
  - Byte-synchronized Claude and OpenCode Scout-to-Route-Setter stage projections
  - Wrapper regression coverage for causal cards, inactive candidates, and exact acceptance
affects: [200-22, 200-23, planning, command-wrappers, claude, opencode]

tech-stack:
  added: []
  patterns: [runtime-authorized stage dispatch, render-only wrapper projection, append-only iteration cards, exact candidate acceptance]

key-files:
  created: []
  modified:
    - .aether/commands/plan.yaml
    - .claude/commands/ant-plan.md
    - .claude/commands/ant/plan.md
    - .opencode/commands/ant/plan.md
    - cmd/plan_wrapper_cards_test.go
    - cmd/plan_wrapper_ceremony_test.go
    - cmd/platform_doc_hygiene_test.go

key-decisions:
  - "Wrappers inspect the approved Specification first, then present exactly Fast 80/4, Balanced 90/6, Deep 95/8, and Exhaustive 99/12 without a default."
  - "Every wrapper dispatches only the single stage currently authorized by Go: Scout first, Route-Setter only after the Scout receipt, and later Scouts only after a completed card."
  - "Every completed pass renders its Go-issued causal card before continuation, a later material pause, or candidate review."
  - "A reasoned stop creates a NOT ACTIVE candidate; wrappers execute the review result's full acceptance_command verbatim before exposing build or Autopilot."

patterns-established:
  - "Stage projection: runtime manifest -> one visible worker -> strict result -> finalizer receipt -> runtime-selected next boundary."
  - "Authority separation: Specification approval, preset choice, stop, candidate creation, and plan acceptance remain distinct transitions."

requirements-completed: [SYNTH-02, CEC-03, PLAN-01, PLAN-03, PLAN-04, PLAN-05, PLAN-06]

duration: 15 min
completed: 2026-09-08
---

# Phase 200 Plan 21: Staged Planning Wrapper Projection Summary

**Claude and OpenCode now project the Go-owned Scout → Route-Setter loop one receipt-bound stage at a time, with causal pass cards and an inactive candidate that becomes buildable only through exact acceptance.**

## Performance

- **Duration:** 15 min
- **Started:** 2026-09-08T00:55:22Z
- **Completed:** 2026-09-08T01:09:29Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments

- Replaced the three-knob opening with an approved-Specification preflight and an unbiased choice among the exact Fast 80/4, Balanced 90/6, Deep 95/8, and Exhaustive 99/12 policies.
- Recast all managed planning wrappers as a strict staged adapter: one Scout manifest and receipt, an optional complete first-pass owner batch, one separately authorized Route-Setter, then a Go-issued iteration card and next boundary.
- Made the wrapper chronology explicit for weakest-gap continuation, post-card later material decisions, four reasoned stop outcomes, full timeline/candidate review, and exact candidate acceptance.
- Kept routine phase research automatic inside the selected preset and removed every old research-approval command/result expectation from canonical and managed planning surfaces.
- Migrated focused parity and documentation tests so Claude's two projections and OpenCode must carry identical stage order, exact preset facts, structured result fields, candidate inactivity, and acceptance bindings.
- Tightened init/discuss documentation hygiene to enforce Init → Discuss → draft Specification → explicit Specification approval → Plan, with no settled-Discuss shortcut directly into planning.

## Task Commits

Each task was committed atomically after its verification gate passed:

1. **Task 1: Replace and synchronize the planning wrapper with staged preset and candidate semantics** — `0d282b40` (feat)
2. **Task 2: Migrate planning wrapper and documentation parity tests** — `a8b52460` (test)

## Files Created/Modified

- `.aether/commands/plan.yaml` — Canonical approved-Specification, preset, staged worker, iteration-card, candidate-review, and exact-acceptance projection contract.
- `.claude/commands/ant-plan.md` — Flat Claude managed projection, synchronized byte-for-byte with the canonical nested Claude wrapper.
- `.claude/commands/ant/plan.md` — Canonical Claude wrapper for one runtime-authorized planning stage at a time.
- `.opencode/commands/ant/plan.md` — OpenCode projection with the same headings, fields, order, and safety boundaries.
- `cmd/plan_wrapper_cards_test.go` — Exact preset, structured stage/card field, causal timeline, inactive candidate, exact acceptance, and retired-contract absence checks.
- `cmd/plan_wrapper_ceremony_test.go` — Approved-Specification-to-acceptance ceremony order, thin-host safety, stage skeleton, and reasoned-stop checks.
- `cmd/platform_doc_hygiene_test.go` — Updated Claude/OpenCode init, discuss, and plan lifecycle documentation expectations.

## Decisions Made

- Wrapper intelligence stops at display, host-native owner interaction, and dispatching the exact current authorization. Go remains the only authority for evidence admission, scores, semantic deltas, materiality, receipts, stage transitions, stops, candidates, acceptance, persistence, and next-action truth.
- The preset is the only routine planning-budget decision. Once chosen, read-only phase research and gap-directed passes proceed automatically up to the runtime-enforced cap.
- First-pass material choices pause before Route-Setter authority; later material choices pause only after the complete Scout → Route-Setter card explains their impact.
- Candidate review is a separate read-only operation, and only the returned full `acceptance_command` may activate the exact candidate against its Specification, base plan, proposal, and timeline bindings.

## Verification

- `rtk go run ./cmd/aether source-check` — passed; 16 canonical surfaces, 5 retired mirrors, and 124 managed wrappers checked.
- `rtk go test ./cmd -run 'TestPlanCommand.*Preset|TestPlanCandidate' -count=1` — 22 tests passed.
- `rtk go test ./cmd -run 'TestPlanWrapperCardsParity|TestPlanWrapperCeremonyContract|TestPlanWrapperStageSkeleton|TestLifecycleCommandDocsPreferRuntimeCLI' -count=1` — 57 tests passed.
- `rtk go test ./cmd -run 'Test.*Plan.*Wrapper|TestLifecycleCommandDocsPreferRuntimeCLI|TestPlanCandidate' -count=1` — 82 tests passed.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- The first migrated test run exposed a case-sensitive expectation mismatch for the displayed `Target sufficiency` label. The assertion was corrected before the task commit; no production contract or runtime behavior changed.
- `state.update-progress` found no legacy body-level `Progress:` field; `state.advance-plan` still updated the authoritative frontmatter to 56 completed plans, and the roadmap handler recorded Phase 200 at 22/25 summaries.
- `requirements.mark-complete` does not parse this milestone's bold-ID checkbox format. All seven Plan 21 requirement IDs were already checked complete, so no manual requirement mutation was needed.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 200-22 can now update the public plan/host contracts, Codex command guide, and build-cycle skill against a stable synchronized primary-platform projection.
- Plan 200-23 can use the migrated wrappers as the public-path basis for the end-to-end Classic contract corpus and final gate receipt.
- No Plan 200-21 blocker remains.

## Self-Check: PASSED

- All seven task-owned files and this summary exist.
- Task commits `0d282b40` and `a8b52460` are present in repository history.
