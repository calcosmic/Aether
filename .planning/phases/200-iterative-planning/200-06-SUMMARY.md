---
phase: 200-iterative-planning
plan: 06
subsystem: planning-persistence
tags: [go, content-addressing, lifecycle-transactions, replay, immutable-history, tdd]

# Dependency graph
requires:
  - phase: 200-iterative-planning
    plan: 01
    provides: Typed planning iteration cards, timeline bindings, plan-revision bindings, semantic deltas, and closed stop decisions
provides:
  - Atomic repository-scoped persistence for immutable content-addressed planning iteration cards and their ordered index
  - Exact append replay and deterministic crash recovery backed by lifecycle transaction evidence
  - Full-chain loading, tamper detection, accepted-revision protection, and explicit legacy-unbound classification
affects: [planning-finalizer, plan-candidates, plan-acceptance, retention-cleanup, planning-renderers]

# Tech tracking
tech-stack:
  added: []
  patterns: [content-addressed append log, lifecycle-transaction two-target commit, receipt-bound exact replay, immutable accepted-history guard]

key-files:
  created:
    - cmd/planning_timeline.go
    - cmd/planning_timeline_test.go
  modified: []

key-decisions:
  - "An append identity binds receipt ID, run ID, ordinal, card ID/hash, and the exact predecessor hash; reusing the receipt for any other payload is a typed conflict."
  - "Historical replay validates the immutable lifecycle journal and returns the digest of that append's original prefix, while full-chain validation permits the shared index to advance through later appends."
  - "Only an exact PlanRevision timeline ID and digest pair upgrades a candidate-only timeline to accepted, after which both prune and reorder operations fail with typed protection errors."
  - "Unindexed legacy iteration JSON is readable only as legacy_unbound evidence and never receives a current timeline binding."

patterns-established:
  - "Timeline append pattern: canonicalize and validate the complete card, bind it to the prior hash and receipt, validate staged card/index bytes, then commit both targets through one lifecycle transaction."
  - "Replay pattern: identical durable intent resumes, identical completed receipts return their original proof without writes, and every divergent reuse fails closed before mutation."
  - "Accepted-history pattern: revalidate card addresses, locators, request identities, predecessor chain, index address, binding address, and whole-chain digest before applying retention policy."

requirements-completed: [CEC-03, PLAN-01, PLAN-03, PLAN-05, PLAN-06]

# Metrics
duration: 37 min
completed: 2026-09-07
---

# Phase 200 Plan 06: Append-Only Planning Timeline Summary

**Content-addressed planning cards now commit atomically as an ordered evidence chain, resume safely after interruption, replay without mutation, and become permanently protected when an accepted plan revision binds their exact digest.**

## Performance

- **Duration:** 37 min
- **Started:** 2026-09-07T14:01:11Z
- **Completed:** 2026-09-07T14:37:42Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Added canonical repository paths under `.aether/data/planning/{run}` with immutable content-addressed iteration cards and a small ordered, content-addressed timeline index.
- Bound every append to its exact receipt, ordinal, card payload, and predecessor hash, while one lifecycle transaction stages, validates, commits, or rolls back the card and index together.
- Added exact replay for both completed and interrupted appends. A matching staged intent resumes deterministically; a matching completed receipt returns the original card, prefix digest, and lifecycle receipt without rewriting current files; divergent reuse returns a typed conflict.
- Added strict full-chain loading with regular-file and symlink checks, canonical locator enforcement, card/index/binding content-address verification, predecessor validation, deterministic timeline digests, and tamper detection.
- Added cleanup protection that distinguishes candidate-only, accepted, and legacy-unbound history. Candidate-only timelines can follow retention policy, ordering stays append-only, and any timeline referenced by an accepted revision is non-prunable and non-reorderable.

## Task Commits

Each TDD task was committed through a failing-test commit followed by its implementation:

1. **Task 1: Store content-addressed iteration cards atomically** — `76eb1a34` (test/RED), `78c15c7d` (feat/GREEN)
2. **Task 2: Replay timelines and bind immutable accepted history** — `44b31d4f` (test/RED), `b82e9de7` (feat/GREEN), `e958b872` (fix/historical replay regression)

**Plan metadata:** committed separately after state synchronization.

## Files Created/Modified

- `cmd/planning_timeline.go` — Card/index serialization, content addresses, safe repository paths, atomic append transactions, durable-intent recovery, exact historical replay, chain loading, deterministic bindings, legacy classification, and accepted-history protection.
- `cmd/planning_timeline_test.go` — Coverage for round-trip fidelity, atomic rollback, invalid paths/ordinals/predecessors, exact and divergent replay, staged-intent resume, historical-prefix replay, deterministic digesting, tamper/reorder detection, accepted protection, and legacy evidence.

## Decisions Made

- Derived card identity from the complete canonical card with only `id` and `content_hash` cleared. Caller-supplied identity can therefore never override stored content.
- Stored the append request digest and deterministic lifecycle transaction ID in every index entry so both the ordered chain and its recovery journal can be cross-validated during load.
- Used the existing lifecycle transaction journal as immutable replay evidence. Historical receipt loading validates journal, manifest, staged bytes, progress, receipt digest, command, and declared changes without requiring an older append's index bytes to equal the legitimately advanced live index.
- Derived a replayed append's timeline digest from its validated card prefix, preserving the original receipt contract even after later passes exist.
- Classified unindexed legacy JSON by safe locator only. It stays inspectable evidence but cannot be exposed as a current `PlanningTimelineBinding` or inferred to have owner acceptance.
- Made reordering invalid for every current timeline, while pruning is permitted only for unaccepted retention candidates. An exact accepted revision reference overrides retention and locks the whole history.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Followed the renamed lifecycle transaction implementation**

- **Found during:** Task 1 (Store content-addressed iteration cards atomically)
- **Issue:** The plan referenced `cmd/lifecycle_tx.go`, which does not exist in the current tree after prior lifecycle work landed.
- **Fix:** Used the repository's actual `cmd/lifecycle_transaction.go` primitives and preserved their durable intent, recovery, allowlist, and receipt contracts unchanged.
- **Files modified:** None outside the plan-owned files.
- **Verification:** Both atomic-failure fixtures pass and the final 17-test timeline suite passes under the race detector.
- **Committed in:** `78c15c7d` and `b82e9de7`

**2. [Rule 1 - Bug] Preserved exact replay after later timeline appends**

- **Found during:** Task 2 final self-review (Replay timelines and bind immutable accepted history)
- **Issue:** The generic lifecycle receipt loader correctly expects current targets to equal one transaction's result, but a shared append-only index legitimately changes after later cards. Replaying an earlier completed receipt would therefore fail or return the latest whole-chain digest rather than that append's original prefix digest.
- **Fix:** Added strict historical journal-receipt validation plus prefix-digest reconstruction, keeping later index bytes untouched while returning the earlier append's exact receipt contract.
- **Files modified:** `cmd/planning_timeline.go`, `cmd/planning_timeline_test.go`
- **Verification:** The new regression failed before the fix, then passed; all 17 timeline tests pass normally and with `-race`.
- **Committed in:** `e958b872`

---

**Total deviations:** 2 auto-fixed (1 blocking reference drift, 1 replay correctness bug)
**Impact on plan:** Both changes were required to execute the intended current-tree lifecycle contract and make exact replay true for the complete append-only history. No feature scope or non-plan production file changed.

## Issues Encountered

- Repository-wide `go test ./... -count=1` completed with 9,269 passes, 11 skips, and the same four protected Phase 199 bookkeeping failures seen before this plan: two unclassified `199-UAT.md` vocabulary occurrences plus stale historical gate-receipt timestamps.
- Repository-wide `go test ./... -race -count=1` reproduced those four failures and also timed out one isolated lifecycle next-action child under full-suite race contention. The exact test passed immediately when rerun alone with `-race`, and the final timeline race suite passed all 17 tests. Per execution scope, `.planning/phases/199-front-door-and-classic-contract/199-PATTERNS.md` and other Phase 199 artifacts were left untouched.
- The installed requirements helper did not recognize this milestone's legacy `**ID — title:**` checkbox shape. CEC-03, PLAN-03, PLAN-05, and PLAN-06 were already complete; PLAN-01 was checked directly using the established Phase 200 tracking fallback.
- The protected unrelated changes in `.planning/config.json`, `.gsd/`, and the Phase 199 `199-PATTERNS.md` remained unstaged and uncommitted throughout execution.

## TDD Gate Compliance

- Task 1 RED (`76eb1a34`) failed on the absent append/storage API before GREEN (`78c15c7d`) made all atomicity, path, ordinal, predecessor, and round-trip fixtures pass.
- Task 2 RED (`44b31d4f`) failed on the absent replay/load/protection APIs before GREEN (`b82e9de7`) made all replay, digest, tamper, binding, and legacy fixtures pass.
- The historical-prefix regression added during self-review failed before `e958b872`, then passed with the full normal and race-enabled timeline suites.
- Every RED commit precedes its corresponding production implementation in repository history.

## Known Stubs

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The planning finalizer can append each completed Scout/Route-Setter pass and safely resume the same receipt after interruption.
- Candidate and acceptance work can consume one deterministic validated timeline binding, while cleanup can query the protection classifier before retention changes.
- Renderers can load the same typed cards and timeline binding without becoming a second persistence authority.
- No Plan 200-06 blocker remains.

## Self-Check: PASSED

- Both plan-owned Go files and this summary exist.
- All five implementation/TDD commits exist in the required RED→GREEN order.
- All 17 `TestPlanningTimeline` tests pass normally and under the race detector; `go vet ./cmd` and whitespace checks pass.
- Exact append, rollback, crash resume, immediate replay, earlier-prefix replay, divergent conflict, tamper, reordered index, accepted protection, and legacy classification each have direct regression coverage.
- Stub and threat-surface scans found no incomplete UI data path, placeholder implementation, new endpoint, authentication path, external network boundary, or unplanned schema trust boundary.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-07*
