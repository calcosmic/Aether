---
phase: 26-auto-repair
plan: 01
status: complete
---

# Plan 01: Repair Engine — Summary

## What was built

The `--fix` repair engine for `aether medic`. Repairs fixable colony data issues with backup, trace logging, and post-repair verification.

## Key Files

### Created
- `cmd/medic_repair.go` — Repair engine: RepairResult, RepairRecord, createBackup, cleanupOldBackups, logRepairToTrace, performRepairs, repairStateIssues, repairPheromoneIssues, repairSessionIssues, repairDataIssues
- `cmd/medic_repair_test.go` — 17 tests covering all repair types

### Modified
- `cmd/medic_cmd.go` — Wired repairs between scan and render, added Repair Log section to visual report and JSON output
- `cmd/medic_scanner.go` — Updated fixableIssue() calls to mark issues as repairable
- `cmd/medic_cmd_test.go` — Updated existing tests for new flow

## Repair Capabilities

| Repair | Trigger | Requires --force |
|--------|---------|------------------|
| Session phase/goal mismatch | Mismatch with COLONY_STATE | No |
| Orphaned worktree entries | status="orphaned" in COLONY_STATE | No |
| Expired pheromone signals | active=true but past expires_at | No |
| Legacy state values | PAUSED/PLANNED/SEALED/ENTOMBED strings | No |
| Deprecated signals array | Non-empty signals[] in COLONY_STATE | No |
| Missing pheromone IDs | Signal without id field | No |
| Invalid pheromone types | Type not in FOCUS/REDIRECT/FEEDBACK | No |
| Corrupted JSON recovery | JSON parse failure | Yes |
| Stale spawn state | Run "running" >1 hour | No |
| Ghost constraints | constraints.json has content | No |
| Stale cache files | .cache_ files detected | No |

## Test Results

- 17 repair tests passing
- All existing tests passing (66+ medic-related tests total)
- Integration test covers full scan → repair → re-scan cycle

## Real-World Test

Running `aether medic --fix` on this repo:
- Detected session.json phase/goal mismatch
- Repaired both issues (2/2 succeeded)
- Post-repair scan: 0 warnings (down from 2)
