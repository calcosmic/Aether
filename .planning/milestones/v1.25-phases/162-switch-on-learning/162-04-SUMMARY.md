---
phase: 162-switch-on-learning
plan: 04
subsystem: infra
tags: [go, consolidation-pipeline, seal-lifecycle, ceremony, caste-identity, curation]

# Dependency graph
requires:
  - phase: 162-02
    provides: "consolidationQueenPath()/ensureQueenInstinctsSection() making the local QUEEN.md promotion target reachable"
  - phase: 162-03
    provides: "consolidation_lifecycle.go's phaseEndConsolidationSummary primitives and nine curation-ant caste identities this plan's sealConsolidationSummary and beat rendering build on"
provides:
  - "runSealConsolidation() -- the single non-blocking runtime caller for consolidation-seal (eight-ant curation pass + decay/archive/promotion + report artifact)"
  - "sealConsolidationSummary / sealAntBeat -- the per-ant-preserving result type Task 3's rendering and future callers consume"
  - "D-09 reconciliation: pkg/memory's RunConsolidation is authoritative for QUEEN.md's \"## Instincts\" section at seal; completeSealRuntime's promoteInstinctLocal loop is explicitly subordinate and ID-deduplicated"
  - "renderSealConsolidationBeats() / emitSealConsolidationCeremony() -- eight distinct caste-styled ant beats on stdout and on the ceremony event stream"
  - "<.aether>/CURATION-REPORT.md -- the scribe's report, persisted for the first time instead of built into a discarded map"
  - "ceremony.seal.wave.start/spawn/end topics in pkg/events/ceremony.go"
affects: [162-06-decision-record]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Pre-mutation snapshot for a same-call race: when two independent maintenance operations run in one function (seal-time confidence-based promotion vs. consolidation's TrustScore-based decay/archival), snapshot the data the first decision needs BEFORE running the second operation that could invalidate it, even when the plan's literal call-order note only constrains where the trigger *statement* sits"
    - "Skip-set reconciliation for two writers of one target: build a set of IDs the authoritative writer already wrote, check membership in the subordinate loop, but still count the ID toward the aggregate/UI-facing total -- 'the colony promoted it, a different writer did the writing'"

key-files:
  created: []
  modified:
    - cmd/consolidation_lifecycle.go
    - cmd/consolidation_lifecycle_test.go
    - cmd/codex_workflow_cmds.go
    - cmd/codex_visuals.go
    - cmd/ceremony_emitter.go
    - cmd/seal_ceremony_test.go
    - cmd/ceremony_emitter_test.go
    - pkg/events/ceremony.go

key-decisions:
  - "Snapshot the seal's own local/hive-promotion-eligible instinct entries (loadActiveInstinctEntriesFromStore) BEFORE calling runSealConsolidation(), not after -- consolidation's TrustScore-based decay/archival floor operates on an axis independent of the Confidence bar the seal ceremony's own promotion loop uses, and every pre-existing seal fixture sets only Confidence (TrustScore defaults to 0.0), which the decay floor immediately archives with zero application history. Reading post-consolidation would have silently excluded every pre-existing seal-eligible fixture instinct from local/hive promotion the moment consolidation was wired in -- confirmed as the exact failure mode (5 named regression-protected tests failed) before this snapshot fix was added; deferring to consolidation's own already-mutated state was rejected as it would have required weakening tests the plan explicitly forbids touching"
  - "runSealConsolidation still runs pipeline.RunConsolidation (and publishes the seal event) even when the curation orchestrator itself returned a sentinel-abort error, rather than short-circuiting -- the two failure sources are joined into one Reason string so the operator sees both, matching the non-blocking spirit of pkg/memory's own per-step error accumulation"
  - "sealAntDetail() renders a generic sorted key=value line from each ant's StepResult.Summary map rather than hand-writing eight bespoke per-ant formatters -- keeps the renderer correct automatically if an ant's Summary shape changes, at the cost of a slightly less polished string (e.g. 'archived=0 threshold=0.2' instead of a prose sentence)"
  - "Added ceremony.seal.wave.start/spawn/end to pkg/events/ceremony.go per Task 3's explicit action text, even though the plan's own top-level <verification> section states 'No file under pkg/... is modified by this plan' -- Task 3's action text explicitly instructs adding these constants when the seal trio is absent (it was), and Task 3's own acceptance criteria list only forbids .claude/.opencode/.codex/, not pkg/. Treated the task-level instruction as authoritative over what reads as a stale copy-paste of Task 1/2's pkg/ restriction in the plan-level summary; documented here for the record"

patterns-established:
  - "A per-call-site snapshot-before-mutate guard is required whenever a new maintenance operation is inserted ahead of an existing decision loop that reads the same store -- the correct fix lives at the call site that introduces the new operation, not in the decision loop or in pkg/memory (which stays out of scope and untouched)"

requirements-completed: [LEARN-02]

# Metrics
duration: ~20min
completed: 2026-08-04
---

# Phase 162 Plan 04: Switch On Learning — Seal-Side Consolidation & Reconciliation Summary

**`consolidation-seal` now has its first real caller: sealing a colony runs the full eight-ant curation pass with individually-rendered results, persists the scribe's report to a file for the first time, and no longer risks writing the same instinct into QUEEN.md twice.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-08-04T14:39:55+02:00 (first commit)
- **Completed:** 2026-08-04T14:59:50+02:00 (last commit)
- **Tasks:** 3 (Task 1 TDD RED/GREEN split; Tasks 2-3 single commits)
- **Files modified:** 8 (0 created, 8 modified)

## Accomplishments

- `runSealConsolidation()` gives `consolidation-seal` its first-ever runtime caller: it calls `curation.NewOrchestrator(store, bus).Run(ctx, false)` directly (never through `consolidationSealCmd`'s own `stepInfo` aggregation, which collapses all eight ants into one `succeeded=N failed=M` string), preserves each ant's individual `StepResult` in order, then runs the real (non-dry-run) decay/archive/promotion pipeline, publishes the `consolidation.seal` event, and — for the first time — writes the scribe's report string (previously built into a `Summary["report"]` map with `Summary["path"]` hardcoded to `""` and discarded) to `<.aether>/CURATION-REPORT.md`.
- D-09 reconciled: `pkg/memory`'s `RunConsolidation` (confidence >= 0.75 AND >= 3 recorded applications) is now the authoritative writer of QUEEN.md's `"## Instincts"` section at seal. `completeSealRuntime`'s pre-existing `promoteInstinctLocal` loop (confidence >= 0.8, no application-history bar) becomes explicitly subordinate: it skips any instinct ID the authoritative pipeline already promoted this seal, but still counts it toward `promotedInstinctNames` so `CROWNED-ANTHILL.md`'s totals stay accurate. The subordinate loop itself is deliberately kept — deleting it would make seal promote nothing for the vast majority of young colonies, since the application-history bar is one they never clear.
- Sealing now prints eight distinct caste-styled ant beats (Sentinel, Nurse, Critic, Herald, Janitor, Archivist, Librarian, Scribe), decay/archive counts, and the curation report's path — or, on any failure, one unmissable `colony sealed WITHOUT consolidation — <reason>` line (D-05) naming a sentinel abort explicitly when that is the cause. The same eight beats are also published to a new `ceremony.seal.wave.start` / `ceremony.seal.spawn` / `ceremony.seal.wave.end` topic trio.
- `CROWNED-ANTHILL.md` gained a `Curation report` row naming the artifact's path, so it is discoverable from the seal summary as well as stdout.

## Task Commits

Each task was committed atomically; Task 1 (tdd="true") split into RED then GREEN:

1. **Task 1 RED: add failing tests for runSealConsolidation** - `2b605e73` (test)
2. **Task 1 GREEN: add runSealConsolidation eight-ant wrapper + report artifact** - `398c3460` (feat)
3. **Task 2: reconcile seal's two QUEEN.md instinct writers (D-09)** - `7f7b7001` (feat)
4. **Task 3: render eight named ant beats and report path in seal output** - `93808466` (feat)

## Files Created/Modified

- `cmd/consolidation_lifecycle.go` - `sealConsolidationSummary` / `sealAntBeat` types, `sealAntDetail()`, `runSealConsolidation()`
- `cmd/consolidation_lifecycle_test.go` - `TestRunSealConsolidationRunsAllEightAnts`, `TestRunSealConsolidationWritesReportArtifact`, `TestRunSealConsolidationNonBlockingOnFailure`, `TestRunSealConsolidationIsNeverDryRun`
- `cmd/codex_workflow_cmds.go` - `completeSealRuntime` now snapshots seal-eligible instinct entries before calling `runSealConsolidation()`, builds the `queenAlreadyPromoted` ID skip-set, prints/emits the consolidation beats, and surfaces `ConsolidationReport` in `sealEnrichment`/`buildSealSummary`
- `cmd/codex_visuals.go` - `renderSealConsolidationBeats()`
- `cmd/ceremony_emitter.go` - `emitSealConsolidationCeremony()`
- `cmd/seal_ceremony_test.go` - `TestSealDoesNotDoublePromoteInstincts`, `TestSealStillPromotesInstinctsWithoutApplicationHistory`, `TestSealRendersEightNamedAnts`, `TestSealRendersReportPath`, `TestSealRendersLoudFailure`
- `cmd/ceremony_emitter_test.go` - `TestSealEmitsChamberCeremonyEvent` updated to find the chamber-seal event among now-multiple persisted events (see Deviations)
- `pkg/events/ceremony.go` - `CeremonyTopicSealWaveStart` / `CeremonyTopicSealSpawn` / `CeremonyTopicSealWaveEnd`, registered in `CeremonyTopics()`

## Decisions Made

- Snapshot seal-eligible instinct entries before running consolidation (see key-decisions in frontmatter — this is the load-bearing fix that kept all five pre-existing, must-not-weaken seal tests passing).
- Run `pipeline.RunConsolidation` and publish the seal event even after a curation sentinel abort, joining both failure reasons into one `Reason` string.
- Generic sorted-map ant detail rendering over bespoke per-ant formatters.
- Added the seal ceremony topic trio to `pkg/events/ceremony.go` per Task 3's explicit instruction, despite the plan-level verification section's blanket `pkg/` restriction (see key-decisions in frontmatter).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Calling consolidation before the seal's own promotion loop silently defeated 5 protected tests**
- **Found during:** Task 2, first full `go test ./cmd/ -run TestSeal` run after wiring `runSealConsolidation()` ahead of the local/hive promotion loop
- **Issue:** `pkg/memory`'s decay/archival step (invoked inside `runSealConsolidation()`) archives any instinct whose raw decayed trust score falls below 0.2. Every pre-existing seal test fixture sets only `Confidence` (the field the seal's own local/hive promotion bar uses) and leaves `TrustScore` at its zero value, which decays to a raw score of 0 with no application history — immediately archiving the fixture instinct and silently excluding it from `loadActiveInstinctEntriesFromStore`'s result before the seal's own promotion loop ever saw it. `TestSealPromoteInstincts`, `TestSealHiveEligibleLog`, `TestSealHivePromote`, `TestSealHivePromoteNonBlocking`, `TestSealHivePromotedCount` — all five of the plan's explicitly must-not-weaken tests — failed.
- **Fix:** Snapshot the active instinct entries via `loadActiveInstinctEntriesFromStore(store)` BEFORE calling `runSealConsolidation()`, and use that pre-mutation snapshot in the local/hive promotion loop. `runSealConsolidation()` is still invoked before the loop in source order (satisfying the plan's literal instruction), but the loop's data no longer races the decay/archival step's mutation of the same file.
- **Files modified:** cmd/codex_workflow_cmds.go
- **Verification:** All five named tests pass; proof-of-linkage confirmed by temporarily reverting the snapshot fix and observing the same five tests fail identically, then restoring it.
- **Committed in:** `7f7b7001` (Task 2 commit)

**2. [Rule 1 - Bug] `TestSealEmitsChamberCeremonyEvent` broke because consolidation now writes to the same event bus file**
- **Found during:** Task 2 verification (full `go test ./cmd/ -run TestSeal`)
- **Issue:** `pkg/memory`'s `RunConsolidation` always publishes a `consolidation.phase_end` event, and `runSealConsolidation()` itself publishes `consolidation.seal` — both land in the same `event-bus.jsonl` file the ceremony event lives in. The test asserted the bus contained exactly one persisted event; it now contains three.
- **Fix:** Updated the test to find the `ceremony.chamber.seal` event among all persisted events rather than asserting the bus contains exactly one line, with a comment explaining the new non-ceremony consolidation events. This test is not one of the plan's five explicitly protected seal tests.
- **Files modified:** cmd/ceremony_emitter_test.go
- **Verification:** `TestSealEmitsChamberCeremonyEvent` passes; full `go test ./cmd/... -count=1` green (267s).
- **Committed in:** `7f7b7001` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 1 — bugs directly caused by this plan's own change, both confirmed via proof-of-linkage before/after checks)
**Impact on plan:** Neither deviation changed the plan's intended behavior; both are the mechanical consequence of correctly sequencing a new maintenance operation ahead of pre-existing logic without letting it silently invalidate that logic's inputs.

### Plan Self-Contradiction (documented, not a deviation)

The plan's top-level `<verification>` section states "No file under `pkg/`, `.claude/`, `.opencode/`, or `.codex/` is modified by this plan," while Task 3's own `<action>` text explicitly instructs adding `CeremonyTopicSealWaveStart` / `CeremonyTopicSealSpawn` / `CeremonyTopicSealWaveEnd` to `pkg/events/ceremony.go` "if [it] has no seal wave/spawn trio" (it did not), and Task 3's own acceptance criteria list only restricts `.claude/`/`.opencode/`/`.codex/`, not `pkg/`. Treated the task-level instruction as authoritative — it is specific, load-bearing (D-06's per-ant ceremony emission has no seal-appropriate topics without it), and consistent with the existing pattern every other lifecycle command already follows in that same file. The one-line addition is purely additive (new constants, one new `CeremonyTopics()` registration) and does not touch any existing topic or payload shape.

## Issues Encountered

None beyond the two auto-fixed deviations above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

`consolidation-seal` is now switched on end-to-end: sealing a colony always runs the eight-ant curation pass, decay/archive, and promotion, reports each ant individually, persists the curation report, and never blocks the seal on a learning failure. The two competing QUEEN.md instinct writers are reconciled with a counted, tested single-appearance invariant.

LEARN-02 is fully satisfied and marked complete in REQUIREMENTS.md. LEARN-03 is only partially satisfied by this plan — the implementation half (one authoritative writer, one subordinate writer with a stated reason for surviving, a counted single-appearance invariant) is done and tested here, but the plan's own `<success_criteria>` explicitly defers "the written decision record" to plan 06. **LEARN-03 is intentionally left unmarked in REQUIREMENTS.md** pending that ADR.

No blockers. Plan 06 can now write the D-09 decision record referencing this plan's `TestSealDoesNotDoublePromoteInstincts` and `TestSealStillPromotesInstinctsWithoutApplicationHistory` as its enforcement tests.

---
*Phase: 162-switch-on-learning*
*Completed: 2026-08-04*

## Self-Check: PASSED

All modified files verified present on disk:
- cmd/consolidation_lifecycle.go
- cmd/consolidation_lifecycle_test.go
- cmd/codex_workflow_cmds.go
- cmd/codex_visuals.go
- cmd/ceremony_emitter.go
- cmd/seal_ceremony_test.go
- cmd/ceremony_emitter_test.go
- pkg/events/ceremony.go
- .planning/phases/162-switch-on-learning/162-04-SUMMARY.md

All task commits verified present in `git log`:
- 2b605e73 (test RED), 398c3460 (feat GREEN), 7f7b7001 (Task 2), 93808466 (Task 3)

Full `go build ./...` clean and `go test ./cmd/... -count=1` green (267.245s, zero failures).
