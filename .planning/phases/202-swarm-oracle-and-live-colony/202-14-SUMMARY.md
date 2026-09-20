---
phase: 202-swarm-oracle-and-live-colony
plan: "14"
subsystem: live-events
tags: [status, history, watch, swarm, oracle, build-attempt, spend, go]

# Dependency graph
requires:
  - phase: 202-10
    provides: "cmd/swarm_episode.go's swarmEpisodeRecord -- the durable, replay-safe Swarm episode this plan folds into the shared lineage without redefining it."
  - phase: 202-12
    provides: "cmd/oracle_research_doc.go's parseOracleResearchFrontMatter and oracleResearchStanding -- the exact parser and standing resolver this plan's Oracle source reuses rather than re-deriving."
  - phase: 202-13
    provides: "cmd/partial_work_label.go's workStanding vocabulary, workStandingLabel, resolveMalformedItemStanding and collectUnverifiedWork -- the shared standing authority every entry in this plan resolves through, and the unverified-work inventory history now lists alongside the lineage."
provides:
  - "cmd/episode_index.go: loadColonyEpisodeIndex -- the one read-only projection over the three durable record sources that already exist (Swarm episodes, saved Oracle research, build/check attempts), ordered newest first with a stable tiebreak, naming any source that is missing or unreadable rather than inventing an outcome for it."
  - "cmd/status.go: a Most Recent Episode section (renderMostRecentEpisodeStatusSection) naming what last ran, its outcome, its standing, and its cost through the existing spend ledger authority -- omitted entirely when the index has no entries."
  - "cmd/history.go: every episode and every piece of unverified work in the same listing, each carrying its outcome or standing, a cost block where one applies, and a path to its full write-up, plus a --kind filter that narrows the listing without altering any surviving row's content."
  - "A cross-surface proof (TestStatusHistoryAndWatchShareOneLineage) that status, history and the replay-backed watch summary describe one shared episode with the same outcome word and the same byte-identical cost block."
affects: [status, history, watch]

# Actuals (#2632)
actuals:
  tokens: 15931
  tasks: 3
  commits: 4

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "One projection, three renderers: loadColonyEpisodeIndex is the single read over Swarm/Oracle/build-attempt records; status's section, history's rows, and the cross-surface test all consume its entries rather than re-deriving outcome, standing, or cost locally."
    - "Directory-existence-is-availability: a source is 'unavailable' only when its own directory does not exist or cannot be listed (loadSwarmEpisodeIndexEntries, loadOracleResearchIndexEntries, loadBuildAttemptIndexEntries) -- an existing-but-empty directory is available with zero entries, so 'this colony never ran Swarm' stays distinguishable from 'Swarm ran and recorded nothing'."
    - "Cost is always a reference (colonyEpisodeCostRef{Phase}), never a copied figure -- resolved at render time through the one existing ledger authority (loadSpendLedgersForPhase, renderSpendCostLineFromLedgers). History renders the SAME full cost block status renders (colonyEpisodeCostBlock), not a compact summary, specifically so the two surfaces can be compared byte-for-byte rather than merely 'agree in spirit'."
    - "root/store resolution follows watch's own precedent (resolveAetherRoot() + the passed *storage.Store), not skillWorkspaceRoot()'s cwd-dependent guess -- so status, history and watch can never silently disagree about which colony they are describing."

key-files:
  created:
    - cmd/episode_index.go
    - cmd/episode_index_test.go
  modified:
    - cmd/status.go
    - cmd/history.go
    - cmd/compatibility_cmds.go
    - cmd/orientation_agreement_199_test.go
    - cmd/lifecycle_history_199_test.go
    - .planning/REQUIREMENTS.md

key-decisions:
  - "A build attempt whose CheckFix field is set is reported as kind 'check-attempt' rather than 'build-attempt', even though both live in the same durable buildAttemptRecord store -- this is the distinction an owner actually cares about, and it does not require a fourth durable record."
  - "Standing for build/check attempts is resolved locally in episode_index.go (buildAttemptIndexStanding) rather than adding a new resolver to cmd/partial_work_label.go, which Phase 202-13 scoped to Swarm and Oracle only -- every branch still routes its final phrase through the shared workStandingLabel, so no second vocabulary was introduced, only a third caller of the existing one."
  - "History's per-episode Cost field renders through renderSpendCostLineFromLedgers directly (the same function status calls), not a compact one-line figure -- chosen specifically so TestStatusHistoryAndWatchShareOneLineage can assert byte-identical cost text across status, history and watch rather than merely 'the same number, formatted differently.'"
  - "collectUnverifiedWork's existing entries (Oracle partial research, Swarm repair ideas/interrupted episodes) and loadColonyEpisodeIndex's episode entries are listed as separate rows in history, never merged or deduplicated against each other -- the plan's own 'unverified work appears in the same listing' language describes concatenation, not reconciliation, and the two answer different questions (what ran vs. what is still unproven)."

patterns-established:
  - "Shared-lineage cross-surface test: TestStatusHistoryAndWatchShareOneLineage drives one fixture (a build attempt, a matching live-event episode, and one spend ledger) through all three commands' own entry points and asserts agreement on the rendered text, not on an intermediate struct -- so a future change to any one renderer's wording is caught by the others disagreeing, not merely by a stale mock."

requirements-completed: [LIVE-05, LIVE-07, LIVE-02]

coverage:
  - id: D1
    description: "One read-only lineage (loadColonyEpisodeIndex) covers Swarm episodes, saved Oracle research, and build/check attempts, ordered newest first with a stable tiebreak, never inventing an outcome for a record that has none, and never writing anything."
    requirement: "LIVE-05"
    verification:
      - kind: unit
        ref: "cmd/episode_index_test.go#TestEpisodeIndexCoversThreeRecordSources"
        status: pass
      - kind: unit
        ref: "cmd/episode_index_test.go#TestEpisodeIndexOrdersNewestFirstWithStableTiebreak"
        status: pass
      - kind: unit
        ref: "cmd/episode_index_test.go#TestEpisodeIndexMarksMissingOutcomeUnknown"
        status: pass
      - kind: unit
        ref: "cmd/episode_index_test.go#TestEpisodeIndexIsReadOnly"
        status: pass
    human_judgment: false
  - id: D2
    description: "The status dashboard names the most recent episode -- what it was, how it ended, its standing, and what it cost -- omitted entirely rather than shown empty when nothing has been recorded, without disturbing any other section."
    requirement: "LIVE-05"
    verification:
      - kind: unit
        ref: "cmd/episode_index_test.go#TestStatusShowsTheMostRecentEpisode"
        status: pass
      - kind: unit
        ref: "cmd/episode_index_test.go#TestStatusOmitsTheEpisodeSectionWhenThereAreNone"
        status: pass
      - kind: unit
        ref: "cmd/episode_index_test.go#TestStatusEpisodeSectionDoesNotDisturbOtherSections"
        status: pass
    human_judgment: false
  - id: D3
    description: "History lists every episode and every piece of unverified work in one listing, each carrying its outcome/standing, cost where applicable, and a path to its full write-up; a kind filter narrows the listing without changing row content; and status, history and the replay-backed watch summary agree on one fixture's episode, outcome and cost block."
    requirement: "LIVE-07"
    verification:
      - kind: unit
        ref: "cmd/episode_index_test.go#TestHistoryListsEveryEpisodeWithOutcomeAndCost"
        status: pass
      - kind: unit
        ref: "cmd/episode_index_test.go#TestHistoryListsUnverifiedWorkWithItsStanding"
        status: pass
      - kind: unit
        ref: "cmd/episode_index_test.go#TestHistoryVerifiedAndUnverifiedAreDistinguishable"
        status: pass
      - kind: unit
        ref: "cmd/episode_index_test.go#TestHistoryFilterNarrowsWithoutChangingRows"
        status: pass
      - kind: unit
        ref: "cmd/episode_index_test.go#TestStatusHistoryAndWatchShareOneLineage"
        status: pass
      - kind: unit
        ref: "cmd/lifecycle_history_199_test.go#TestLifecycleHistory199Order"
        status: pass
    human_judgment: false

# Metrics
duration: 55min
completed: 2026-09-11
status: complete
---

# Phase 202 Plan 14: One Read-Only Lineage for Status, History and Watch Summary

**A single projection (`loadColonyEpisodeIndex`) over the Swarm, Oracle, and build/check attempt records that already exist, surfaced on the status dashboard's "Most Recent Episode" section and in history's full episode-and-unverified-work listing, with a fixture-driven proof that status, history and watch cannot disagree about what ran.**

## Performance

- **Duration:** 55 min
- **Started:** 2026-09-11T14:38:00Z
- **Completed:** 2026-09-11T15:33:00Z
- **Tasks:** 3
- **Files modified:** 8

## Accomplishments
- `cmd/episode_index.go` reads three durable sources -- Swarm episodes, saved Oracle research, build/check attempts -- into one ordered, read-only lineage; a missing source is named `unavailable` rather than silently contributing nothing.
- `cmd/status.go` gains a "Most Recent Episode" section: what last ran, its outcome, its standing (through the shared vocabulary), and its cost via the existing spend ledger authority -- omitted entirely with nothing recorded, proved not to disturb any other section.
- `cmd/history.go` lists every episode and every piece of unverified work (Oracle partial research, Swarm repair ideas, interrupted episodes, plan research, reflections) in the same listing, each with a path to its full write-up, plus a `--kind` filter.
- A cross-surface test drives one fixture through status, history and watch's own entry points and asserts the same episode, outcome word, and byte-identical cost block across all three.

## Task Commits

Each task was committed atomically:

1. **Task 1: One read-only lineage over the records that already exist** - `03fdd059` (feat)
2. **Task 2: Put the most recent episode on the status dashboard** - `f27f726d` (feat)
3. **Task 3: List every episode, and every unproven note, in history** - `f7ab656d` (feat)
4. **Read-only proof hardening + REQUIREMENTS.md checkbox** - `9b81d8c6` (test)

**Plan metadata:** (this commit)

## Files Created/Modified
- `cmd/episode_index.go` - `loadColonyEpisodeIndex` and its three per-source loaders (Swarm, Oracle research, build/check attempts)
- `cmd/episode_index_test.go` - all 12 new tests across the three tasks
- `cmd/status.go` - the Most Recent Episode section (loader + renderer)
- `cmd/history.go` - episode and unverified-work rows, `Kind`/`Standing`/`Cost`/`Path` fields on `LifecycleHistoryRow`, the `--kind` filter
- `cmd/compatibility_cmds.go`, `cmd/orientation_agreement_199_test.go` - updated call sites for `buildLifecycleHistoryProjection`'s new parameters
- `cmd/lifecycle_history_199_test.go` - `TestLifecycleHistory199Order` now separates the pre-existing event/actor/receipt rows from the newly-additive episode/unverified rows before checking its original count and tie-break assertions
- `.planning/REQUIREMENTS.md` - `LIVE-07` marked complete

## Decisions Made
- Kind `check-attempt` vs `build-attempt` is decided by whether `buildAttemptRecord.CheckFix` is set -- no fourth durable record was introduced.
- Standing for build/check attempts is resolved in `episode_index.go` itself (not added to `partial_work_label.go`, which Phase 202-13 scoped to Swarm/Oracle), but still renders exclusively through the shared `workStandingLabel`.
- History's episode cost renders the full `renderSpendCostLineFromLedgers` block (identical to status's), specifically so the cross-surface test can assert byte-identical text rather than merely equivalent figures.
- The episode lineage and the unverified-work inventory are listed as separate rows in history, never deduplicated against each other -- they answer different questions (what ran vs. what remains unproven), and the plan's own language describes one listing, not one reconciled record.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `TestLifecycleHistory199Order`'s hard-coded row count went stale under this plan's own new, correct behavior**
- **Found during:** Task 3 verification (`go test ./cmd -run '^TestLifecycleHistory'`)
- **Issue:** The shared "valid" lifecycle-facts fixture (`seedLifecycleFactsFixture`, used by many unrelated tests) already seeds `.aether/research/front-door.md` and `.aether/dreams/orientation.md` for other tests' purposes. Once history started listing episodes and unverified work, these pre-existing fixture files legitimately produced three additional rows, breaking the test's exact `len(rows) != 7` assertion.
- **Fix:** Filtered the "episode" and "unverified_work" categories out of the row set before applying the test's original count/order/tie-break assertions -- proving the plan's own stated invariant ("the existing history rows for lifecycle events and receipts are unchanged in shape and order relative to each other") rather than the stronger, unintended claim that no other row may ever appear.
- **Files modified:** cmd/lifecycle_history_199_test.go
- **Verification:** `go test ./cmd -run '^TestLifecycleHistory'` passes; `TestLifecycleHistory199Order`'s DeepEqual determinism check and receipt/tie-break assertions are unchanged.
- **Committed in:** f7ab656d (Task 3 commit)

**2. [Rule 3 - Blocking] `requirements mark-complete`'s checkbox regex does not match this milestone's REQUIREMENTS.md bold convention**
- **Found during:** Post-task requirement marking
- **Issue:** `gsd-tools query requirements.mark-complete` expects an isolated `**REQ-ID**` bold span; this milestone's file bolds the ID and its description together (`**LIVE-07 — Durable Oracle synthesis:**`), so the tool reported `not_found` for both `LIVE-05` (already checked) and `LIVE-07`.
- **Fix:** Edited the `LIVE-07` checkbox by hand to match the file's existing convention (the same style every already-checked requirement in this file uses).
- **Files modified:** .planning/REQUIREMENTS.md
- **Verification:** Visual diff confirms only the checkbox character changed; the file's per-phase traceability table has no per-requirement Status column, so no second surface needed reconciling.
- **Committed in:** 9b81d8c6

---

**Total deviations:** 2 auto-fixed (1 bug, 1 blocking tooling mismatch)
**Impact on plan:** Both fixes were necessary to reach a genuinely green test suite and an honest REQUIREMENTS.md; neither changed this plan's own scope or behavior.

## Issues Encountered
- `LIVE-02` (real live cockpit) is declared in this plan's `requirements` frontmatter but was NOT checked in REQUIREMENTS.md: `gsd-tools query requirements.ready-ids` reports it blocked by a sibling plan in this phase that has not yet produced a `*-SUMMARY.md`. It will become ready and get marked automatically once that sibling plan finishes (per the shared-ID gate, `execute-plan.md`'s `update_requirements` step).

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Status, history and watch now share one read-only lineage over Swarm episodes, Oracle research, and build/check attempts -- ready for any later plan (202-15) that needs to present "what the colony has done" without inventing a fourth record type.
- `LIVE-02` remains open, blocked on a sibling plan in this phase finishing.

## Self-Check: PASSED

---
*Phase: 202-swarm-oracle-and-live-colony*
*Completed: 2026-09-11*
