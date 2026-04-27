---
phase: 62-lifecycle-ceremony-seal-and-init
plan: 01
subsystem: seal-ceremony
tags: [seal, blockers, promotion, pheromones, enrichment]
dependency_graph:
  requires: []
  provides: [62-02]
  affects: []
tech_stack:
  added: []
  patterns: [ceremony-steps, local-only-promotion, bulk-signal-expiry]
key_files:
  created:
    - cmd/seal_ceremony_test.go
  modified:
    - cmd/codex_workflow_cmds.go
    - cmd/pheromone_write.go
    - cmd/queen.go
decisions: []
metrics:
  duration: PT5M
  completed_date: 2026-04-27
---

# Phase 62 Plan 01: Seal Ceremony Hardening Summary

Seal becomes a real ceremony: blocker checking with --force override, local instinct promotion, FOCUS pheromone expiry, and enriched CROWNED-ANTHILL.md with colony statistics.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Seal blocker checking and --force flag | 2f3f8d6d | cmd/codex_workflow_cmds.go, cmd/seal_ceremony_test.go |
| 2 | Local instinct promotion, FOCUS expiry, CROWNED-ANTHILL enrichment | 3b546deb | cmd/codex_workflow_cmds.go, cmd/pheromone_write.go, cmd/queen.go, cmd/seal_ceremony_test.go |

## Deviations from Plan

None -- plan executed exactly as written.

## Key Changes

### cmd/codex_workflow_cmds.go
- Added `--force` bool flag to sealCmd
- Added `checkSealBlockers()` -- loads flags, splits unresolved into blockers and issues
- Added `renderBlockerSummary()` -- go-pretty table with resolution hints per blocker
- Added `countResolvedFlags()` -- counts resolved entries for enrichment
- Added `sealEnrichment` struct for ceremony metrics
- Modified `buildSealSummary()` to accept enrichment parameter and output Colony Statistics table
- Inserted 3 ceremony steps in sealCmd: instinct promotion, hive suggestion log, FOCUS expiry

### cmd/pheromone_write.go
- Added `expireSignalsByType()` -- bulk deactivates active signals of a given type using `deactivateSignal()`
- Added `storage` import

### cmd/queen.go
- Added `promoteInstinctLocal()` -- writes a single instinct to local QUEEN.md Wisdom section only (no global hub write per D-08)

### cmd/seal_ceremony_test.go (new)
- 14 new test functions covering all seal ceremony paths
- Tests use COLONY_DATA_DIR env var for PersistentPreRunE compatibility
- Tests verify: blocker blocking, --force override, issue warning, JSON output, instinct promotion, hive suggestion, FOCUS/REDIRECT preservation, CROWNED-ANTHILL enrichment

## Threat Flags

None -- all changes operate on local colony data within the established trust boundary.

## Self-Check: PASSED
