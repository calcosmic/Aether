---
phase: 162-switch-on-learning
plan: 03
subsystem: infra
tags: [go, consolidation-pipeline, continue-lifecycle, ceremony, caste-identity]

# Dependency graph
requires:
  - phase: 162-02
    provides: "consolidationQueenPath()/ensureQueenInstinctsSection() making the promotion target reachable, so this plan's runtime callers switch consolidation on into a real destination rather than a void"
provides:
  - "runPhaseEndConsolidation(phaseID int) phaseEndConsolidationSummary — the single non-blocking real-path runtime caller for phase-end consolidation"
  - "attachConsolidationSummary() — snake_case summary attacher matching consolidation-phase-end's own JSON shape"
  - "Two call sites (cmd/codex_continue.go, cmd/codex_continue_finalize.go) invoking consolidation only on durable phase advance"
  - "Nine curation-ant caste identities (sentinel, nurse, critic, herald, janitor, archivist, librarian, scribe, curator) in codex_visuals.go's three caste maps"
  - "renderLearningBeat() / continueLearningFlowStep() — the always-present 🧠 Learning stage beat in four honest states"
affects: [162-04-seal-side-reconciliation, 165-core-lifecycle-commands]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Runtime-owned lifecycle wiring: a new file (consolidation_lifecycle.go) holds the shared non-blocking wrapper; call sites are a one-line addition beside an existing precedent (captureContinueLearning), never a wrapper-markdown edit"
    - "Enrichment vs. gate: a phase-end side effect that can fail is a summary value with Ran/Reason fields, never an error type the caller could propagate into an abort"
    - "Single source of truth for a rendered line: phaseEndConsolidationSummary.LearningBeatLine() is called by both the terminal beat (renderLearningBeat) and the ceremony flow step (continueLearningFlowStep) so the two surfaces cannot drift apart"

key-files:
  created:
    - cmd/consolidation_lifecycle.go
    - cmd/consolidation_lifecycle_test.go
    - cmd/learning_beat_test.go
  modified:
    - cmd/codex_continue.go
    - cmd/codex_continue_finalize.go
    - cmd/codex_visuals.go
    - cmd/golden_workflow_test.go
    - cmd/testdata/golden_continue.txt
    - cmd/testdata/regression_snapshot.json

key-decisions:
  - "runPhaseEndConsolidation treats a non-empty result.Errors the same as a top-level err from pipeline.RunConsolidation — ConsolidationService.Run (pkg/memory) records per-step failures (e.g. corrupt instincts.json) into result.Errors instead of returning a top-level error, so the wrapper had to check both to honor D-05's 'never silently succeed on a partial failure' intent"
  - "External/finalize path emits the learning beat via its own follow-up emitContinueCeremonyFlowSequence call rather than appending into the same batch the default path uses — advanceExternalContinue already emits its own ceremony sequence internally before returning, and D-04 places consolidation strictly after that call returns (PhaseCompleted is written inside advanceExternalContinue), so the two emissions cannot be merged without breaking D-04's placement lock"
  - "TestGoldenContinueVisualOutput now seeds an empty-but-valid instincts.json/learning-observations.json — without this, consolidation would fail against a fresh temp store and embed that test's t.TempDir() path into the checked-in golden fixture, making it mismatch on every future run"
  - "documented_castes regression snapshot updated 26 -> 35 (the only field that changed) after adding 9 curation-ant identities, confirming the count is a pure caste-map cardinality check"

patterns-established:
  - "A rendered ceremony/beat line should have exactly one source of truth (a method on the domain value) consumed by both the stdout renderer and the ceremony event-stream step, not two independently-maintained string templates"

requirements-completed: [LEARN-01]

# Metrics
duration: ~44min
completed: 2026-08-04
---

# Phase 162 Plan 03: Switch On Learning — Consolidation Lifecycle Wiring Summary

**Phase-end consolidation now has its first two runtime callers — wired into both continue paths on durable advance only, reported through nine newly-identified curation-ant castes and an always-present 🧠 learning beat that never goes silent.**

## Performance

- **Duration:** ~44 min
- **Started:** 2026-08-04T13:41:16+02:00 (first commit)
- **Completed:** 2026-08-04T14:24:36+02:00 (last commit)
- **Tasks:** 3 (Task 1 TDD RED/GREEN split; Tasks 2-3 single commits)
- **Files modified:** 9 (3 created, 6 modified)

## Accomplishments

- `cmd/consolidation_lifecycle.go` gives `consolidation-phase-end` its first-ever runtime caller after four milestones (v1.10, v1.11, v1.13, v1.23) declared the pipeline "restored" without ever wiring one: `runPhaseEndConsolidation` builds the real (non-dry-run) pipeline, self-heals the QUEEN.md Instincts section, bounds the run with a 30s timeout (T-162-12), and never returns an error type a caller could propagate into an abort.
- Both continue paths invoke it only on durable advance: the default path calls it beside `captureContinueLearning`, after the atomic `COLONY_STATE.json` write; the external/finalize path calls it strictly after `advanceExternalContinue` returns `nil` (the stricter-correct placement, since `PhaseCompleted` is written inside that call). A negative test (`TestContinueWithoutAdvanceDoesNotConsolidate`) proves a blocked continue produces zero consolidation side effects.
- The nine curation ants (`sentinel`, `nurse`, `critic`, `herald`, `janitor`, `archivist`, `librarian`, `scribe`, plus `curator` for the aggregate line) now have distinct emoji/color/label identities instead of the generic 🐜/"Ant" fallback every unmapped caste renders as.
- Every phase advance renders a `── Learning ──` stage with a 🧠 Librarian-styled beat in exactly one of four states — populated (counts), zero ("colony observed nothing new this phase"), failed (D-05's exact wording), or absent (no consolidation key recorded) — so silence about learning, the specific failure mode this milestone exists to kill, is not a reachable output.
- Fixed a latent non-determinism risk in the golden continue test: without seeding a valid empty consolidation fixture, the new wiring would have embedded `t.TempDir()`'s absolute path into the checked-in golden file on every regeneration.

## Task Commits

Each task was committed atomically; Task 1 (tdd="true") split into RED then GREEN:

1. **Task 1 RED: add failing test for phase-end consolidation wrapper** - `03bfd1ee` (test)
2. **Task 1 GREEN: add runPhaseEndConsolidation non-blocking wrapper** - `2ea66c85` (feat)
3. **Task 2: wire phase-end consolidation into both continue paths** - `44247c0e` (feat)
4. **Task 3: add curation-ant caste identities and learning beat** - `fa5bc93a` (feat)

## Files Created/Modified

- `cmd/consolidation_lifecycle.go` - `phaseEndConsolidationSummary` type + `ZeroState()`/`LearningBeatLine()` methods, `runPhaseEndConsolidation()`, `attachConsolidationSummary()`
- `cmd/consolidation_lifecycle_test.go` - Success/failure/zero-state unit tests (Task 1) plus three wiring/negative tests (Task 2)
- `cmd/codex_continue.go` - Default-path call site + `continueLearningFlowStep()` (modelled on `continueHousekeepingFlowStep`)
- `cmd/codex_continue_finalize.go` - External/finalize-path call site (post-`advanceExternalContinue`) with its own follow-up ceremony emission
- `cmd/codex_visuals.go` - Nine curation-ant entries in `casteEmojiMap`/`casteColorMap`/`casteLabelMap`; `renderLearningBeat()`; unconditional call from `renderContinueVisual`
- `cmd/learning_beat_test.go` - Caste-distinctness test and four-state learning-beat render test
- `cmd/golden_workflow_test.go` - Seeds a valid empty consolidation fixture before driving the golden continue flow
- `cmd/testdata/golden_continue.txt` - Regenerated to include the new worker-flow line and Learning stage
- `cmd/testdata/regression_snapshot.json` - `documented_castes` updated 26 -> 35 (only field that changed)

## Decisions Made

- Treating `result.Errors` (populated by `ConsolidationService.Run` on a load failure) the same as a top-level `err` inside `runPhaseEndConsolidation` — see key-decisions in frontmatter.
- External path's learning beat is emitted via a second, follow-up `emitContinueCeremonyFlowSequence` call rather than appended into the batch `advanceExternalContinue` already emitted internally — see key-decisions in frontmatter.
- Golden test now seeds `instincts.json`/`learning-observations.json` as empty-but-valid rather than leaving them absent, for deterministic zero-state golden output.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Golden continue fixture and regression caste-count snapshot needed regeneration**
- **Found during:** Task 3 verification (full `go test ./cmd/ -count=1` run)
- **Issue:** `TestGoldenContinueVisualOutput` failed because the new worker-flow line and Learning stage changed the exact stdout text a checked-in golden file pins; separately, `TestRegressionSnapshot` failed because `documented_castes` (a live count of `casteEmojiMap` entries) moved from 26 to 35 after adding the nine curation-ant identities. Both are checked-in snapshots the project's own regression-test convention expects to be regenerated (`-update-golden`) when the underlying behavior legitimately changes.
- **Fix:** Seeded a valid empty consolidation fixture in the golden continue test (so its output is deterministic rather than embedding a temp-dir path), regenerated `cmd/testdata/golden_continue.txt` via `-update-golden`, and regenerated `cmd/testdata/regression_snapshot.json` via `-update-golden` — confirmed only `documented_castes` changed in the latter.
- **Files modified:** cmd/golden_workflow_test.go, cmd/testdata/golden_continue.txt, cmd/testdata/regression_snapshot.json
- **Verification:** `go test ./cmd/ -count=1` green end to end (264s, zero `--- FAIL` lines) after the fix, rerun twice to confirm.
- **Committed in:** fa5bc93a (Task 3 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 - bug/snapshot drift, two checked-in fixtures)
**Impact on plan:** Both fixture updates are the expected, mechanical consequence of Task 3's intentional new output (worker-flow line, Learning stage, nine new castes) — not scope creep. No production behavior was changed to make tests pass; the fixtures were regenerated to match new, intentional behavior.

## Issues Encountered

- `pkg/memory.ConsolidationService.Run` never returns a non-nil top-level error even when a load step fails (e.g. corrupt `instincts.json`) — failures are recorded into `result.Errors` and the function still returns `(result, nil)`. Task 1's `TestRunPhaseEndConsolidationIsNonBlockingOnFailure` (which seeds invalid JSON) initially could not be satisfied by checking `err != nil` alone. Resolved by additionally checking `len(result.Errors) > 0` and joining those errors via `errors.Join` before deciding `Ran: false` — this also meant the zero-state and mutation tests needed to seed empty-but-*parseable* `instincts.json`/`learning-observations.json` files (not leave them absent), since a missing file is itself a load failure.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Phase-end consolidation is switched on end-to-end for `/ant-continue`: both the default and external/finalize paths invoke it exactly on durable advance, a partial or total failure is reported unmissably and never blocks the advance, and the operator sees a 🧠 learning beat in one of four honest states on every single advance. Plan 04 (seal-side reconciliation) can now build the seal-path equivalent (`runSealConsolidation`, the eight-ant ceremony pass) on top of the same `phaseEndConsolidationSummary`/`attachConsolidationSummary` primitives this plan established, and reconcile the two competing QUEEN.md writers (`pkg/memory`'s "Instincts" section vs. the existing seal-path "Wisdom" section) per D-09.

No blockers. `consolidationPhaseEndCmd`/`consolidationSealCmd` (the user-invocable inspection/manual subcommands) were deliberately left untouched, per the plan's explicit scope boundary — they remain available for manual inspection alongside the new automatic runtime path.

---
*Phase: 162-switch-on-learning*
*Completed: 2026-08-04*
