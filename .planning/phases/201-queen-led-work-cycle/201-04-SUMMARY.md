---
phase: 201-queen-led-work-cycle
plan: "04"
subsystem: queen-orchestration
tags: [go, work-outcome, closeout, ceremony, D-05]

# Dependency graph
requires:
  - phase: 201-queen-led-work-cycle (plan 01)
    provides: "SYN-201-06's cited synthesis decision -- extend the current outcome vocabulary rather than reverting to Classic's binary pass/fail model, via a total mapping that never widens the existing closed OutcomeKind enum"
provides:
  - "pkg/colony/work_outcome.go: colony.WorkOutcome, the closed six-verdict work-outcome vocabulary (success, no_change, partial, blocker, timeout, interrupted) with Valid/MarshalJSON/UnmarshalJSON mirroring OutcomeKind's own discipline, a total LifecycleOutcome() mapping onto the existing closed OutcomeKind vocabulary, and WorkOutcomeLabels() as the single authority for each verdict's owner-facing wording"
  - "cmd/lifecycle_closeout.go: LifecycleCloseoutDetails.WorkOutcome / LifecycleCloseout.WorkOutcome carry the verdict through buildLifecycleCloseout; when present, every canonical closeout slot renders (none dropped for emptiness) so a non-success verdict gets the identical full ceremony a success verdict gets"
  - "cmd/lifecycle_closeout_work_outcome_test.go: TestCloseoutCeremonyIsEqualAcrossVerdicts, an invariant over the runtime's own declared verdict and slot sets that fails by name when a future slot is ever wired up for only some verdicts, or a verdict ships with no label"
affects: [201-06, 201-07, 201-08, 201-09, 201-10, 201-12, 201-15]

# Actuals (#2632)
actuals:
  tokens: 8646
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Additive typed verdict over a closed durable enum (mirrors OutcomeKind's own Valid/MarshalJSON/UnmarshalJSON discipline one layer up): WorkOutcome never widens OutcomeKind -- it carries a total mapping (LifecycleOutcome()) onto the existing values, so a record written by this version still decodes cleanly in the version before it, and the zero value decodes as absent rather than defaulting to success"
    - "Full-ceremony slot gating keyed on verdict presence (cmd/lifecycle_closeout.go lifecycleCloseoutSlots): a closeout with no work verdict keeps its exact pre-existing empty-slot-skip behavior (byte-identical JSON); a closeout carrying a verdict renders the full canonical slot set unconditionally, with each render helper saying plainly 'Nothing recorded' rather than a slot silently disappearing"
    - "Pure invariant-violation functions over runtime-declared sets (mirrors the AST call-graph guards in 201-02/201-03, but comparing rendered content instead of parsed syntax): lifecycleCeremonySlotViolations and lifecycleWorkOutcomeLabelCoverageViolations take no *testing.T, so the same check runs unchanged against real production output and a synthetic broken fixture, and the violation strings the test asserts on ARE what a human would read, not a boolean"

key-files:
  created:
    - pkg/colony/work_outcome.go
    - pkg/colony/work_outcome_test.go
    - cmd/lifecycle_closeout_work_outcome_test.go
  modified:
    - cmd/lifecycle_closeout.go

key-decisions:
  - "WorkOutcome is never folded into OutcomeKind, per the plan's explicit must_haves and despite SYN-201-06 leaving 'extend OutcomeKind directly' as one option on the table. OutcomeKind is validated strictly by validateLifecycleHeader and is the wire vocabulary an older binary decodes; widening it would make a record written by this version undecodable by the version before it. LifecycleOutcome() gives the required total mapping instead, and `grep -c 'OutcomeKind = \"' pkg/colony/lifecycle.go` is unchanged by this plan (still 11)."
  - "The default closeout summary text (buildLifecycleCloseout, used when the caller supplies no LifecycleCloseoutDetails.Summary) now reads from WorkOutcomeLabels() when a verdict is present, instead of the generic 'X finished with outcome Y.' phrasing. This was forced by TestNonSuccessVerdictBorrowsNoSuccessWording: the generic fallback's own word 'finished' collided with the success verdict's label wording for every non-success verdict, which is exactly the borrowed-success-wording failure D-05 exists to prevent. Fixing the shared fallback (rather than picking success wording that dodges 'finished') is the correct fix -- it also means every verdict's default summary now actually says something specific to that verdict, not generic prose."
  - "lifecycleCloseoutSlots' full-ceremony behavior is gated on `closeout.WorkOutcome != nil`, not on any other signal (e.g. outcome kind). A closeout built without a work verdict -- the vast majority of existing call sites, none of which pass this plan's new field -- keeps its pre-existing empty-slot-skip logic exactly as written, proven byte-identical by TestCloseoutWithoutAVerdictIsUnchanged. Only a caller that explicitly supplies a verdict opts into the full eight-slot ceremony."
  - "The equal-ceremony invariant (Task 3) is implemented as two small pure functions (lifecycleCeremonySlotViolations, lifecycleWorkOutcomeLabelCoverageViolations) rather than inline *testing.T assertions, so the exact same violation-detection logic runs against both real production output (must find zero violations) and a synthetic broken fixture (must find and name the specific violation) -- avoiding the need to fake t.Errorf to prove the guard actually fires."

requirements-completed: []
# WORK-05 and CEC-06 (this plan's declared requirements) are each also
# declared by sibling 201-* plans (WORK-05: 201-06/07/10/15; CEC-06:
# 201-02 [complete]/06/07/12/15) and therefore stay open per the shared-ID
# gate (#2388) until every declaring plan has a SUMMARY.md -- confirmed via
# `gsd-tools query requirements.ready-ids`, correct and expected, not a gap
# in this plan's own work.

coverage:
  - id: D1
    description: "A closed six-verdict work-outcome vocabulary (success, no_change, partial, blocker, timeout, interrupted) with a total mapping onto the existing closed OutcomeKind vocabulary, never widening it; an absent verdict decodes and reports as absent rather than defaulting to success"
    requirement: WORK-05
    verification:
      - kind: unit
        ref: "pkg/colony/work_outcome_test.go#TestWorkOutcomeVocabularyIsClosed"
        status: pass
      - kind: unit
        ref: "pkg/colony/work_outcome_test.go#TestWorkOutcomeLifecycleMappingIsTotal"
        status: pass
      - kind: unit
        ref: "pkg/colony/work_outcome_test.go#TestAbsentWorkOutcomeIsNotSuccess"
        status: pass
      - kind: unit
        ref: "pkg/colony/work_outcome_test.go#TestWorkOutcomeLabelsAreComplete"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every one of the six work verdicts renders the identical full closeout ceremony (colony identity, participants, what happened, evidence, state changes, standing instructions, unresolved items, next choices) -- only the verdict's own wording differs, and a non-success verdict never borrows the success verdict's language"
    requirement: CEC-06
    verification:
      - kind: unit
        ref: "cmd/lifecycle_closeout_work_outcome_test.go#TestEveryWorkVerdictRendersTheSameCeremonySlots"
        status: pass
      - kind: unit
        ref: "cmd/lifecycle_closeout_work_outcome_test.go#TestNonSuccessVerdictBorrowsNoSuccessWording"
        status: pass
      - kind: unit
        ref: "cmd/lifecycle_closeout_work_outcome_test.go#TestCloseoutWithoutAVerdictIsUnchanged"
        status: pass
      - kind: unit
        ref: "cmd/lifecycle_closeout_199_test.go#TestLifecycleCloseout199Contract"
        status: pass
    human_judgment: false
  - id: D3
    description: "The equal-ceremony property is held by a structural invariant over the runtime's own declared verdict and slot sets, not by hand-maintained lists -- it fails by name (slot + affected verdicts) when a future slot is ever wired up for only some verdicts, and fails by name when a declared verdict ships with no label"
    requirement: CEC-06
    verification:
      - kind: unit
        ref: "cmd/lifecycle_closeout_work_outcome_test.go#TestCloseoutCeremonyIsEqualAcrossVerdicts"
        status: pass
    human_judgment: false

duration: 45min
completed: 2026-09-10
status: complete
---

# Phase 201 Plan 04: Work-Outcome Vocabulary and Equal-Ceremony Closeout Summary

**A closed six-verdict work-outcome vocabulary (`colony.WorkOutcome`) with a total, non-widening mapping onto the existing lifecycle outcome kinds, plus a closeout ceremony that renders every slot identically for all six verdicts and is held equal by a structural invariant, not by care.**

## Performance

- **Duration:** 45 min
- **Started:** 2026-09-10T12:55:00Z (approx.)
- **Completed:** 2026-09-10T13:09:26Z
- **Tasks:** 3
- **Files modified:** 4 (3 created, 1 modified)

## Accomplishments

- Created `pkg/colony/work_outcome.go` defining `WorkOutcome` (six constants: `success`, `no_change`, `partial`, `blocker`, `timeout`, `interrupted`), with `Valid()`, `MarshalJSON`/`UnmarshalJSON` following `OutcomeKind`'s exact fail-closed discipline, `LifecycleOutcome()` giving a total mapping onto the existing closed `OutcomeKind` vocabulary (success→completed, no_change→no_change, partial→in_progress, blocker→failed, timeout→failed, interrupted→recovery_required), `IsSuccess()`, and `WorkOutcomeLabels()` as the single owner-facing wording authority.
- Wired the verdict through `cmd/lifecycle_closeout.go`: `LifecycleCloseoutDetails.WorkOutcome` (input) and `LifecycleCloseout.WorkOutcome` (output, `omitempty`); `buildLifecycleCloseout` derives the lifecycle outcome from the verdict rather than a second inference, and every canonical slot renders when a verdict is present -- an empty slot says "Nothing recorded" / "No participants recorded" / etc. rather than disappearing.
- Fixed a real wording collision the equal-ceremony test caught: the closeout's generic default summary ("X finished with outcome Y.") shared the word "finished" with the success verdict's label, which would have quietly borrowed success-shaped language onto every non-success card. The default summary now reads from `WorkOutcomeLabels()` when a verdict is present.
- Added `cmd/lifecycle_closeout_work_outcome_test.go` with `TestEveryWorkVerdictRendersTheSameCeremonySlots` (slot-set equality + label presence across all six verdicts, including a sparse fixture proving empty slots still render), `TestNonSuccessVerdictBorrowsNoSuccessWording` (token-level proof, tokens derived from `WorkOutcomeLabels()` inside the test), and `TestCloseoutWithoutAVerdictIsUnchanged` (a closeout built with no verdict stays byte-identical to its pre-existing behavior).
- Added `TestCloseoutCeremonyIsEqualAcrossVerdicts`, a structural invariant built from two pure, `*testing.T`-free functions (`lifecycleCeremonySlotViolations`, `lifecycleWorkOutcomeLabelCoverageViolations`) that enumerate `colony.AllWorkOutcomes()` and `lifecycleCloseoutCanonicalSlots` from the runtime's own declarations, run clean against real production output, and -- proven against synthetic broken fixtures in the same test -- fail by name (slot + verdicts, or verdict) the moment a future slot or a future verdict breaks the invariant.

## Task Commits

1. **Task 1: Define the closed six-verdict work-outcome vocabulary** - `6ef5ec6a` (feat)
2. **Task 2: Carry the verdict through the closeout ceremony** - `4472188e` (feat)
3. **Task 3: Hold the equal-ceremony invariant against future slots** - `771e4888` (test)

**Plan metadata:** committed alongside this summary.

## Files Created/Modified

- `pkg/colony/work_outcome.go` - `WorkOutcome` type, six constants, `AllWorkOutcomes()`, `Valid`, `MarshalJSON`/`UnmarshalJSON`, `LifecycleOutcome()`, `IsSuccess()`, `WorkOutcomeLabels()`
- `pkg/colony/work_outcome_test.go` - Closed-vocabulary, total-mapping, absent-is-not-success, and label-completeness coverage
- `cmd/lifecycle_closeout.go` - `LifecycleCloseoutDetails.WorkOutcome`, `LifecycleCloseout.WorkOutcome`, `LifecycleCloseoutEvent.Verdict`, verdict-aware default summary, full-ceremony `lifecycleCloseoutSlots`, "nothing recorded" render fallbacks for Colony/Participants/Evidence/StandingInstructions/Unresolved
- `cmd/lifecycle_closeout_work_outcome_test.go` - Equal-slot-set, no-borrowed-wording, unchanged-without-a-verdict, and the runtime-declared-set equal-ceremony invariant (with synthetic-failure sub-tests)

## Decisions Made

- `WorkOutcome` is never folded into `OutcomeKind` -- a total `LifecycleOutcome()` mapping instead, per the plan's explicit prohibition and confirmed unchanged by `grep -c 'OutcomeKind = "' pkg/colony/lifecycle.go` (still 11).
- The closeout's generic default summary text now reads from `WorkOutcomeLabels()` when a verdict is present, fixing a real wording collision the equal-ceremony test surfaced (the word "finished" was shared between the generic fallback and the success label).
- Full-ceremony slot rendering is gated strictly on `closeout.WorkOutcome != nil` so every existing call site (none of which sets the new field) keeps byte-identical output.
- The equal-ceremony invariant is two pure violation-reporting functions rather than inline assertions, so the identical logic proves both "the real package holds the invariant" and "a synthetic regression is caught and named" without faking `*testing.T`.

## Deviations from Plan

None - plan executed exactly as written. (The default-summary wording fix described above was implementation work needed to satisfy the plan's own acceptance criterion -- "the rendered card shares no token with the success verdict's label" -- not a deviation from what the plan specified.)

## Issues Encountered

- `TestEveryLifecycleCommandEndsWithNextAction` failed once when run as part of a broad `-run 'Lifecycle|Closeout|WorkOutcome'` sweep (`storage: repository containment refused: data root path must not be empty` inside an isolated child process), then passed cleanly in isolation both before and after this plan's changes (confirmed via `git stash`). This matches the project's own documented "Suite ceiling measured" / isolated-child resource-contention finding from `201-03-SUMMARY.md`'s own Issues Encountered section -- a pre-existing environment limitation under parallel load, not a defect this plan introduced.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plans 201-06 through 201-10 and 201-12/201-15 (each declaring `WORK-05` and/or `CEC-06`) now have `colony.WorkOutcome`, `WorkOutcomeLabels()`, and a proven equal-ceremony closeout to attach cost, files, next action, and repair evidence to -- a card shape that is already equal for every verdict, per the plan's own stated success criterion.
- No blockers. `go build ./...` and `go vet ./...` are clean; targeted `go test ./cmd ./pkg/colony` runs for all seven new test functions plus the full pre-existing `TestLifecycleCloseout*` suite pass.

---
*Phase: 201-queen-led-work-cycle*
*Completed: 2026-09-10*

## Self-Check: PASSED

- Created files verified present on disk: `pkg/colony/work_outcome.go`, `pkg/colony/work_outcome_test.go`, `cmd/lifecycle_closeout_work_outcome_test.go`.
- Task commit hashes verified present in `git log`: `6ef5ec6a`, `4472188e`, `771e4888`.
- Re-ran plan-level `<verification>`: `go test ./pkg/colony -run '^(TestWorkOutcomeVocabularyIsClosed|TestWorkOutcomeLifecycleMappingIsTotal|TestAbsentWorkOutcomeIsNotSuccess)$'` and `go test ./cmd -run '^(TestEveryWorkVerdictRendersTheSameCeremonySlots|TestNonSuccessVerdictBorrowsNoSuccessWording|TestCloseoutWithoutAVerdictIsUnchanged|TestCloseoutCeremonyIsEqualAcrossVerdicts|TestLifecycleCloseout)'` — all PASS. `go build ./...` and `go vet ./cmd ./pkg/colony` clean.
