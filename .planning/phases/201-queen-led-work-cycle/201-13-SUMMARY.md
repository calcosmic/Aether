---
phase: 201-queen-led-work-cycle
plan: "13"
subsystem: queen-orchestration
tags: [go, telemetry, turnaround, tdd, owner-decision, WORK-08]

# Dependency graph
requires:
  - phase: 201-queen-led-work-cycle (plan 12)
    provides: "cmd/job_telemetry.go: jobTelemetryRecord, the eight-segment attempt-bound timing record this plan's comparison reads, and jobTelemetrySegmentLabels/jobTelemetrySegmentOrder, the vocabulary a target's Segment is validated against"
provides:
  - ".planning/phases/201-queen-led-work-cycle/201-TURNAROUND-BASELINE.md: a real, owner-approved single-run measurement of a one-plan job (408s/6m48s build-dispatch window), an owner-recorded caveat that the measured job is lighter than his typical request (a floor-setting benchmark for small jobs, not the typical case), and the owner-ratified target (4m30s / 270s, targeting the work segment)"
  - "cmd/turnaround_target.go: turnaroundTarget (an exact total duration plus the single named segment it targets), ratifiedTurnaroundTarget (the 4m30s/work figure copied verbatim from the baseline document), and compareTurnaroundTarget's three-state (met/missed-with-exact-shortfall/unmeasurable) read against a jobTelemetryRecord"
  - "An owner-decision record (aether decision-answer, id pd_1789065675116599000, .aether/data/pending-decisions.json) durably binding the two ratification answers to the phase, independent of this document's prose"
affects: [201-14, 201-15]

# Actuals (#2632)
actuals:
  tokens: 7283
  tasks: 3
  commits: 4

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Ratification travels through the runtime's own owner-decision path, not just prose: `aether decision-answer` recorded both of the owner's answers (representativeness + target figure) as a durable pending-decision entry before the baseline document's prose was updated to match, so the ratification is a checkable fact rather than something only a human reading the markdown would trust."
    - "A target's Segment field is validated at construction against the exact same vocabulary map (jobTelemetrySegmentLabels) job_telemetry.go already defines, rather than a second parallel list of segment names -- there is exactly one place in the codebase that knows the eight valid segment names."
    - "Comparison abstains rather than guesses: compareTurnaroundTarget reads the record's own Measured flag for the targeted segment and returns turnaroundComparisonUnmeasurable, carrying the segment's own stated reason, before it will ever compute a met/missed verdict -- the same honesty discipline jobTelemetryRecord's unmeasured segments and verificationScope's mode selection already hold one layer down."
    - "Fixture records in tests are built through the production capture path (newJobTelemetryCapture/capture.measure/markUnmeasured/newJobTelemetryRecord), never a hand-written struct literal, so the tests exercise the same code real instrumentation uses to produce a record."

key-files:
  created:
    - cmd/turnaround_target.go
    - cmd/turnaround_target_test.go
  modified:
    - .planning/phases/201-queen-led-work-cycle/201-TURNAROUND-BASELINE.md

key-decisions:
  - "Owner ratified the proposed target as-is: 4 minutes 30 seconds (270s) for a small, single-task job, down from the measured 408s (6m48s), reached only via a slimmer worker brief and the project's own already-proven test-suite speedups -- never by cutting testing or adding parallel workers."
  - "Owner named the measured job as lighter than his typical request. The baseline document and the ratified target are recorded explicitly as a floor-setting benchmark for small, single-task jobs, not as a claim that every job he runs should finish in 4m30s."

requirements-completed: [WORK-08]

coverage:
  - id: D1
    description: "A real one-plan job was measured end to end (owner-approved single-run deviation) and recorded in the baseline document with all eight segments accounted for or honestly named unmeasured."
    verification:
      - kind: other
        ref: "grep segment/median/proposed-target checks against 201-TURNAROUND-BASELINE.md (Task 1's own <verify> command)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The owner confirmed the measured job's representativeness and ratified the exact target figure, and that ratification is recorded both in the baseline document and through the runtime's own owner-decision path."
    verification:
      - kind: other
        ref: "grep 'ratified target'/'representative' checks against 201-TURNAROUND-BASELINE.md, plus aether decision-answer id pd_1789065675116599000 in .aether/data/pending-decisions.json"
        status: pass
    human_judgment: false
  - id: D3
    description: "The ratified target is stored in code as turnaroundTarget/ratifiedTurnaroundTarget and its three-state comparison (met/missed-with-exact-shortfall/unmeasurable) against a real jobTelemetryRecord, built RED-GREEN via TDD."
    verification:
      - kind: unit
        ref: "cmd/turnaround_target_test.go#TestRatifiedTurnaroundTargetMatchesBaseline"
        status: pass
      - kind: unit
        ref: "cmd/turnaround_target_test.go#TestTurnaroundTargetComparison"
        status: pass
      - kind: unit
        ref: "cmd/turnaround_target_test.go#TestUnmeasurableNeverReportsMetOrMissed"
        status: pass
      - kind: unit
        ref: "cmd/turnaround_target_test.go#TestTurnaroundTargetRefusesOutOfVocabularySegment"
        status: pass
      - kind: unit
        ref: "cmd/turnaround_target_test.go#TestTurnaroundTargetReadWritesNothing"
        status: pass
    human_judgment: false

duration: 6min
completed: 2026-09-10
status: complete
---

# Phase 201 Plan 13: Turnaround Baseline and Ratified Target Summary

**Owner ratified a 4m30s turnaround target for small single-task jobs (down from a measured 408s), and the target now lives in code as a checkable met/missed/unmeasurable comparison against real job telemetry.**

## Performance

- **Duration:** 6 min (Tasks 2-3, this session; Task 1's real measurement was completed and committed in a prior session — commit `002af506`)
- **Started:** 2026-09-10T18:41:45Z
- **Completed:** 2026-09-10T18:44:14Z
- **Tasks:** 3 (Task 1 previously complete; Tasks 2 and 3 completed this session)
- **Files modified:** 3 (1 modified, 2 created)

## Accomplishments
- Owner ratified the turnaround target at exactly 4 minutes 30 seconds for a small, single-task job, and named the measured job as lighter than his typical request — recorded in the baseline document as a floor-setting benchmark, not the typical case.
- Ratification recorded through the runtime's own owner-decision path (`aether decision-answer`), not only as prose in a markdown file, so it is a durable, auditable fact.
- The ratified target and its three-state comparison are now real, tested Go code (`cmd/turnaround_target.go`), built RED-GREEN through TDD, that a later plan or phase can read without guessing.

## Task Commits

Each task was committed atomically:

1. **Task 1: Measure a real one-plan job and record the baseline** - `002af506` (docs) — completed in a prior session
2. **Task 2: Decision — the owner ratifies the turnaround target** - `ad5e3f8c` (docs)
3. **Task 3: Store the ratified target where a check can read it** - `d9e424c0` (test), `269a0adb` (feat)

_Note: Task 3 was TDD — RED (failing test, undefined symbols) then GREEN (implementation), no REFACTOR commit needed (code was clean on first pass, confirmed via `gofmt -l`)._

## Files Created/Modified
- `.planning/phases/201-queen-led-work-cycle/201-TURNAROUND-BASELINE.md` - Added the owner's verbatim ratification answers and the ratified target section (4m30s, work segment, arithmetic, approved lever, explicit exclusions)
- `cmd/turnaround_target.go` - `turnaroundTarget`, `ratifiedTurnaroundTarget` (4m30s/work), `compareTurnaroundTarget` (met/missed-with-shortfall/unmeasurable)
- `cmd/turnaround_target_test.go` - Five tests covering the ratified figure, met/exact-match/missed-with-shortfall, unmeasurable-never-met-or-missed, out-of-vocabulary refusal, and read-writes-nothing (via fixture-directory digest)

## Decisions Made
- **Target ratified as proposed, no adjustment:** the owner took the exact arithmetic Task 1 proposed (408s × ⅔ ≈ 270s = 4m30s) rather than asking for a different bar.
- **Scope caveat carried forward explicitly:** because the owner named the measured job as lighter than typical, the ratified target is documented as applying only to small, single-task jobs — this prevents a later reader from mistaking 4m30s as the bar for every job size.
- **Comparison targets the `work` segment specifically**, not the record's overall total, because that is the one segment Task 1's real measurement actually captured, and the segment the project's own prior evidence (the worker-turnaround-too-slow todo) already names as the dominant cost.

## Deviations from Plan

None for Tasks 2-3 — executed exactly as written. (Task 1's single-run-instead-of-three-run deviation was owner-approved and is documented in that task's own commit and in the baseline document itself; it is not repeated here as this session's deviation, since it predates this session.)

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Plan 201-14's levers (slimmer worker brief, proven suite speedups) now have an exact, owner-ratified number to aim at: cut the `work` segment from 408s to 270s (4m30s) for a comparably small job.
- `compareTurnaroundTarget` is available for a later plan to wire into a status/closing-line read — this plan deliberately left it unwired from any gate that could block a build or check, per its own scope boundary.
- The baseline document also names two real, reproducible environmental defects discovered during Task 1's measurement (a sandbox denying all worker writes under `.aether/data/`, and a `aether discuss` semantic-lineage length bug) — recorded for the project's attention, not fixed in this plan, and worth a look before they surface again in a future real run.

---
*Phase: 201-queen-led-work-cycle*
*Completed: 2026-09-10*
