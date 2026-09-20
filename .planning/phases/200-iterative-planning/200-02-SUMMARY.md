---
phase: 200-iterative-planning
plan: 02
subsystem: planning-state
tags: [go, migration, validation, sha256, replay-safety]

# Dependency graph
requires:
  - phase: 200-iterative-planning
    plan: 01
    provides: Immutable specification lineage, planning candidates, exact acceptance receipts, and additive legacy-compatible bindings
provides:
  - Deterministic validation for current specification, plan revision, candidate, timeline, and acceptance chains
  - Explicit legacy_unbound classification that preserves existing executable phase and task state without inventing authority
  - Contained SHA-256 indexing of pre-Phase-200 planning evidence and timeline artifacts
  - Replay-stable plan hashes across absent and explicit legacy classification markers
affects: [planning-loop, plan-finalize, timeline-replay, accepted-plan-authority, lifecycle-state-load]

# Tech tracking
tech-stack:
  added: []
  patterns: [pure load-time migration, validate-before-read path containment, authority-neutral legacy indexing, canonical compatibility hashes]

key-files:
  created:
    - cmd/planning_state.go
    - cmd/planning_state_test.go
    - cmd/planning_migration.go
    - cmd/planning_migration_test.go
  modified:
    - cmd/state_load.go
    - cmd/state_load_test.go
    - cmd/plan_revision.go

key-decisions:
  - "Classify only executable pre-Phase-200 plans as legacy_unbound; never synthesize specification approval, plan candidates, acceptance receipts, semantic IDs, or freshness."
  - "Re-index legacy planning files deterministically on load as evidence or timeline material after validating every path before any read."
  - "Keep state loads observational and let the next ordinary safe state transaction persist the in-memory legacy marker."
  - "Canonicalize the additive legacy_unbound marker out of stale-packet hashes because it does not change executable plan content."

patterns-established:
  - "Authority-preserving migration: additive compatibility metadata cannot grant modern owner authority."
  - "Legacy artifact boundary: only regular files contained by repository .aether/data/planning are readable, and JSON must be well formed before hashing into the index."
  - "Replay-compatible hashing: compatibility classification does not invalidate an otherwise unchanged planning packet."

requirements-completed: [PLAN-05, PLAN-06]

# Metrics
duration: 40 min
completed: 2026-09-07
---

# Phase 200 Plan 02: Planning State Migration Summary

**Current planning authority now fails closed on corrupt lineage or bindings, while older executable colonies load as explicitly legacy-unbound with contained, content-addressed historical evidence.**

## Performance

- **Duration:** 40 min
- **Started:** 2026-09-07T11:42:08Z
- **Completed:** 2026-09-07T12:22:00Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments

- Added deterministic structural validation for current specification revisions, plan revisions, semantic/proof links, five-dimension iteration cards, timeline digests, candidates, and exact acceptance bindings.
- Added an idempotent migration that preserves legacy phases, task statuses, active revision identity, and build eligibility while assigning only the explicit `legacy_unbound` classification.
- Added SHA-256 indexing for old planning artifacts as authority-neutral evidence or timeline material, with malformed JSON, absolute paths, traversal, symlinks, and out-of-bound files rejected before content is read.
- Preserved observational command behavior and pre-upgrade packet replay by avoiding implicit state writes and canonicalizing the additive legacy marker out of plan-state hashes.

## Task Commits

Each TDD task was committed through a failing-test commit followed by its implementation:

1. **Task 1: Validate current specification and planning state** — `549ebe82` (test/RED), `8c5bc7a4` (feat/GREEN)
2. **Task 2: Classify and migrate legacy planning artifacts on load** — `6b783fb7` (test/RED), `ac4d7845` (feat/GREEN), `56d86292` and `0081199d` (integration fixes)

**Plan metadata:** committed separately after state synchronization.

## Files Created/Modified

- `cmd/planning_state.go` — Pure validation and normalization for current specification/planning authority and legacy-compatible absence.
- `cmd/planning_state_test.go` — Valid current round-trip plus corrupt lineage, duplicate dimension, wrong digest, future-schema, partial-current, and empty-legacy fixtures.
- `cmd/planning_migration.go` — Idempotent legacy classifier and safe evidence/timeline artifact discovery, containment, JSON validation, and hashing.
- `cmd/planning_migration_test.go` — Migration preservation, authority-negative, idempotence, material classification, malformed-artifact, path-escape, and replay-hash coverage.
- `cmd/state_load.go` — Load-time planning validation/migration for both normal and explicitly read-only state paths without implicit migration writes.
- `cmd/state_load_test.go` — Build-eligibility preservation, safe-write persistence, read-only behavior, and escaped-symlink refusal coverage.
- `cmd/plan_revision.go` — Compatibility-stable plan-state hashing for missing versus explicit legacy classification.

## Decisions Made

- Legacy compatibility is an explicit classification, not a fabricated modern approval: old phases remain executable, but no specification or acceptance record is created.
- Artifact indexing remains a deterministic derived view rather than mutating historical `PlanRevision` snapshots; later timeline readers can reproduce it from the same contained bytes.
- State inspection and preview surfaces remain byte-for-byte observational. A command that truly changes state persists the already migrated in-memory value through the existing safe store transaction.
- The `legacy_unbound` marker is excluded only from legacy plan-state stale-packet hashing because adding that marker changes classification, not executable phase content.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Preserved no-write preview and report modes**

- **Found during:** Task 2 broad command-package verification
- **Issue:** The first migration implementation persisted `legacy_unbound` directly from the generic state loader, causing `plan --plan-only`, Seal preview, and report-only verification to mutate `COLONY_STATE.json`.
- **Fix:** Kept load-time classification in memory and deferred persistence to the next ordinary state-changing command's existing safe writer.
- **Files modified:** `cmd/state_load.go`, `cmd/state_load_test.go`
- **Verification:** The plan gate plus `TestPlanOnlyUnchanged`, `TestSealPlanOnlyDoesNotMutate`, and both report-only verification tests pass.
- **Committed in:** `56d86292`

**2. [Rule 2 - Missing Critical] Kept pre-upgrade planning packets replayable**

- **Found during:** Task 2 broad command-package verification
- **Issue:** Adding an explicit legacy marker changed `planStateHash`, so an unchanged packet created before migration was incorrectly rejected as stale after loading.
- **Fix:** Canonicalized `legacy_unbound` to its absent pre-migration representation only while hashing plan state, and added a regression test.
- **Files modified:** `cmd/plan_revision.go`, `cmd/planning_migration_test.go`
- **Verification:** `TestPlanningMigrationLegacyClassificationPreservesPlanStateHash` and `TestValidateExternalPlanStateAllowsExistingPlanWhenManifestAcknowledges` pass.
- **Committed in:** `0081199d`

---

**Total deviations:** 2 auto-fixed (1 bug, 1 missing critical replay guarantee)
**Impact on plan:** Both fixes preserve established safety and replay contracts; neither expands user-facing scope.

## Issues Encountered

- The plan's read-first list named `cmd/lifecycle_tx.go`; the repository's actual safe transaction implementation is `cmd/lifecycle_transaction.go`, which was used as the intended reference.
- The additional full `cmd` suite exposed two reproducible, out-of-scope Phase 199 tracking failures: the already documented vocabulary inventory mismatch and stale checked-in gate-receipt timestamps. Both are recorded in `deferred-items.md`. A broad-run lifecycle next-action verdict passed three consecutive isolated reruns and was treated as transient shared-test-state interference.
- The installed `requirements.mark-complete` handler returned `not_found` for the milestone's legacy bold-heading format; the exact `PLAN-05` and `PLAN-06` checkboxes were already checked by Plan 01, so no requirement file edit was needed.

## TDD Gate Compliance

- Task 1 RED (`549ebe82`) preceded GREEN (`8c5bc7a4`).
- Task 2 RED (`6b783fb7`) preceded GREEN (`ac4d7845`).
- `go test ./cmd -run 'TestPlanningState|TestPlanningMigration|TestStateLoad.*Planning' -count=1` passes.
- Adjacent preview, report-only, and external-packet replay regressions pass, and `go vet ./cmd` passes.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Later Phase 200 planning-loop, timeline replay, lifecycle-facts, and accepted-plan authority work can consume a validated current state or an explicit legacy compatibility classification.
- No Phase 200 blocker remains. Phase 199 tracking receipt/inventory maintenance is deferred and does not affect the scoped planning-state gates.

## Self-Check: PASSED

- All four created Go files and the plan summary exist.
- All six implementation/test/fix commits are present in order.
- The exact plan verification, adjacent integration regressions, vet, and whitespace checks pass.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-07*
