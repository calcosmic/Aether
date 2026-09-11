---
phase: 202-swarm-oracle-and-live-colony
plan: "13"
subsystem: infra
tags: [oracle, swarm, standing-vocabulary, honesty-labelling, go]

# Dependency graph
requires:
  - phase: 202-09
    provides: watch replay branch and lifecycle facts reading conventions
  - phase: 202-10
    provides: durable, replay-safe Swarm episode record (cmd/swarm_episode.go)
  - phase: 202-12
    provides: Oracle's recommendation-first synthesis and its existing oracleResearchPartialLabel/oracleResearchStandingLabel discipline
provides:
  - The one three-value standing vocabulary (verified / useful notes / standing unknown) in cmd/partial_work_label.go
  - workStandingLabel, the single renderer both Oracle and Swarm resolve a standing phrase through
  - collectUnverifiedWork, a read-only inventory of Oracle research, Swarm repair ideas/interrupted episodes, plan research, and local reflections
affects: [202-14, status, history, watch]

# Actuals (#2632)
actuals:
  tokens: 8839
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "One shared standing vocabulary (workStanding: verified/useful notes/standing unknown) resolved by every subsystem through a single renderer (workStandingLabel), never rendered locally"
    - "Read-only cross-subsystem inventory built by direct filesystem reads (no storage.NewStore, which creates directories on construction) to preserve a strict no-mutation guarantee"

key-files:
  created:
    - cmd/partial_work_label.go
    - cmd/partial_work_label_test.go
  modified:
    - cmd/oracle_research_doc.go
    - cmd/swarm_episode.go

key-decisions:
  - "Oracle's existing rendered text (front-matter 'standing:' key and the body's 'partial — stopped after N rounds' first line) is left byte-identical; only the internal branch decision is rewired through the shared vocabulary (oracleResearchStanding), because existing tests (TestSavedSynthesisFrontMatterCarriesStandingAndConfidence, TestStoppedResearchIsFiledAndLabelledPartial, etc.) assert that exact text and are not part of this plan's scope to change."
  - "collectUnverifiedSwarmWork reads swarms/*/episode.json directly off disk rather than via storage.NewStore, because NewStore calls os.MkdirAll on construction — using it would violate the read-only guarantee (digest-unchanged acceptance criterion) and would create a directory on a project where none existed."
  - "The inventory only lists non-verified items (an item whose standing resolves to verified is skipped) — the task's own name is 'inventory every piece of unproven work,' so a clean, finished item is not something the inventory needs to carry."
  - "Plan research (.aether/data/planning/) and local reflections (.aether/dreams/) have no built-in verification concept, so every file found there is always useful notes; only Oracle research and Swarm episodes carry a determinable verified/not-verified split."

requirements-completed: [LIVE-05, LIVE-07]

coverage:
  - id: D1
    description: "One standing vocabulary (verified, useful notes, standing unknown) shared by Swarm and Oracle, rendered through a single function"
    requirement: "LIVE-07"
    verification:
      - kind: unit
        ref: "cmd/partial_work_label_test.go#TestOneStandingVocabularyAcrossSubsystems"
        status: pass
      - kind: unit
        ref: "cmd/partial_work_label_test.go#TestUsefulNotesLabelNamesWhatWouldVerifyIt"
        status: pass
      - kind: unit
        ref: "cmd/partial_work_label_test.go#TestUnknownStandingIsNeverVerified"
        status: pass
    human_judgment: false
  - id: D2
    description: "Oracle's existing partial-research trigger condition is unchanged after being rewired through the shared vocabulary"
    requirement: "LIVE-07"
    verification:
      - kind: unit
        ref: "cmd/partial_work_label_test.go#TestOraclePartialTriggerConditionIsUnchanged"
        status: pass
      - kind: unit
        ref: "cmd/oracle_research_doc_test.go#TestOracleResearchDocumentRecordsWhatWasAsked"
        status: pass
      - kind: unit
        ref: "cmd/oracle_synthesis_test.go#TestSavedSynthesisFrontMatterCarriesStandingAndConfidence"
        status: pass
    human_judgment: false
  - id: D3
    description: "Swarm's unapplied/rolled-back repair ideas and interrupted episodes resolve to useful notes with distinct, stated reasons"
    requirement: "LIVE-05"
    verification:
      - kind: unit
        ref: "cmd/partial_work_label_test.go#TestSwarmRepairIdeaAndInterruptedRunHaveDistinctReasons"
        status: pass
    human_judgment: false
  - id: D4
    description: "collectUnverifiedWork inventories all four sources (Oracle research, Swarm episodes/repair ideas, plan research, reflections), read-only, deterministic, never dropping a malformed item"
    requirement: "LIVE-05"
    verification:
      - kind: unit
        ref: "cmd/partial_work_label_test.go#TestUnverifiedWorkInventoryCoversFourSources"
        status: pass
      - kind: unit
        ref: "cmd/partial_work_label_test.go#TestUnverifiedWorkInventoryIsReadOnly"
        status: pass
      - kind: unit
        ref: "cmd/partial_work_label_test.go#TestMalformedItemIsListedAsUnknownNotDropped"
        status: pass
      - kind: unit
        ref: "cmd/partial_work_label_test.go#TestUnverifiedWorkInventoryOrderingIsDeterministic"
        status: pass
      - kind: unit
        ref: "cmd/partial_work_label_test.go#TestUnverifiedWorkInventoryAllSourcesAbsentYieldsNoEntries"
        status: pass
    human_judgment: false

# Metrics
duration: 25min
completed: 2026-09-11
status: complete
---

# Phase 202 Plan 13: One Shared Standing Vocabulary for Unproven Work Summary

**A three-value standing vocabulary (verified / useful notes / standing unknown) shared by Swarm and Oracle, plus a read-only inventory (`collectUnverifiedWork`) that lists every unproven piece of research, repair idea, interrupted run, plan artifact, and reflection in one place, honestly labelled.**

## Performance

- **Duration:** 25 min
- **Started:** 2026-09-11T13:20:00Z (approximate)
- **Completed:** 2026-09-11T13:45:10Z
- **Tasks:** 2
- **Files modified:** 4 (2 created, 2 modified)

## Accomplishments
- Declared the one standing vocabulary (`workStanding`: `verified`, `useful notes`, `standing unknown`) and the single renderer `workStandingLabel` that both Swarm and Oracle now resolve a standing phrase through — no subsystem renders a standing phrase locally.
- Rewired Oracle's `oracleResearchPartialLabel` to resolve its complete/not-complete branch through the new `oracleResearchStanding`, keeping its rendered text (front matter and body) byte-identical to before.
- Gave Swarm's proposed repair ideas and interrupted episodes their own standing (`swarmRepairIdeaStanding`, `swarmEpisodeStanding` in `cmd/swarm_episode.go`): an unapplied idea, a rolled-back repair, and an interrupted run each resolve to useful notes with their own distinct reason.
- Built `collectUnverifiedWork`, a strictly read-only inventory covering four sources — Oracle's durable research documents, Swarm's episode store, the planning flow's sanctioned scratch directory, and the local Dreams reflections directory — each entry carrying its kind, subject, standing, the sentence naming what would verify it, its recorded time, and a path to read it in full.

## Task Commits

Each task was committed atomically:

1. **Task 1: One standing vocabulary, shared by Swarm and Oracle** - `4e2a490a` (feat)
2. **Task 2: Inventory every piece of unproven work in one read-only listing** - `34239e71` (feat)

**Plan metadata:** committed together with this SUMMARY (see below).

## Files Created/Modified
- `cmd/partial_work_label.go` - The shared standing vocabulary, `workStandingLabel`, `resolveMalformedItemStanding`, and `collectUnverifiedWork` with its four per-source collectors
- `cmd/partial_work_label_test.go` - Tests proving one vocabulary across subsystems, the useful-notes label contract, the never-verified-when-unknown guarantee, Oracle's unchanged trigger condition, and the inventory's four-source coverage, read-only guarantee, malformed-item handling, and deterministic ordering
- `cmd/oracle_research_doc.go` - Added `oracleResearchStanding`; rewired `oracleResearchPartialLabel` to branch through it (text output unchanged)
- `cmd/swarm_episode.go` - Added `swarmEpisodeStanding` and `swarmRepairIdeaStanding`, plus the `swarmEpisodeVerificationCompleted` constant they share with the existing learning-proposal check

## Decisions Made
- Oracle's rendered text stays byte-identical (front matter `standing:` key and the body's "partial — stopped after N rounds" line); only the internal branch decision moved to the shared vocabulary. Existing tests (`TestSavedSynthesisFrontMatterCarriesStandingAndConfidence`, `TestStoppedResearchIsFiledAndLabelledPartial`, `TestOracleResearchDocumentRecordsWhatWasAsked`, and others) assert that exact wording and are outside this plan's scope to change.
- `collectUnverifiedSwarmWork` reads `swarms/*/episode.json` directly off disk with `os.ReadDir`/`os.ReadFile` rather than through `storage.NewStore`, because `NewStore` calls `os.MkdirAll` on construction — using it would both violate the read-only/digest-unchanged acceptance criterion and create a directory that may not otherwise exist.
- The inventory lists only non-verified items; a clean, finished item is skipped, matching the task's own scope ("inventory every piece of unproven work").
- Plan research and local reflections have no built-in verification concept of their own, so every file found in those two directories is always `useful notes`; only Oracle research and Swarm episodes carry a determinable verified/not-verified split.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `collectUnverifiedWork` and the shared standing vocabulary are ready for 202-14 to bind into status/history (the read-only lineage index) without any storage-shape rework.
- `go build ./cmd/...`, `go vet ./cmd`, and both tasks' exact plan-specified `<verify>` commands pass, along with the full Swarm and Oracle regression suites (`go test ./cmd -run 'Swarm'`, `go test ./cmd -run 'Oracle'`) and every test asserting Oracle's exact pre-existing wording.
- No blockers.

---
*Phase: 202-swarm-oracle-and-live-colony*
*Completed: 2026-09-11*
