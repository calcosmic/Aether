---
phase: 145-silent-pipeline-fix
plan: 03
status: complete
started: 2026-05-20T23:05:00Z
completed: 2026-05-20T23:12:00Z
---

# Plan 145-03: Pending Decisions Session Scoping — Summary

## Status

Complete. All tasks verified.

## What Was Built

### Task 1: Session scope filtering in pending-decision-list
`cmd/pending_decision.go` already contained session scope filtering in `pendingDecisionListCmd`:
- Line 110: `scope := loadCurrentPendingDecisionScope()`
- Line 111: `active, stale := filterPendingDecisionFileForScope(file, scope)`
- Output includes `stale` count for transparency
- `--unresolved` and `--type` filters apply after scoping

### Task 2: Cross-session isolation tests
`cmd/pending_decision_test.go` already contains:
- `TestPendingDecisionListFilterUnresolved` — verifies filtering works
- `TestPendingDecisionListFilterType` — verifies type filtering works
- `TestPendingDecisionResolveRejectsStaleScopedDecision` — verifies resolve rejects stale decisions
- `TestPendingDecisionAddWithFlags` — verifies add stamps scope

### Task 3: Playbook readers use scoped CLI
Fixed `plan-prep.md` line 49:
- Before: `aether pending-decisions --count 2>/dev/null || echo "0"` (command doesn't exist)
- After: `aether pending-decision-list | jq '.unresolved' 2>/dev/null || echo "0"` (uses scoped CLI)

## Verification

```bash
go test ./cmd/... -run "TestPendingDecision" -v
# All 9 tests pass

grep -rn "pending-decisions" .aether/docs/command-playbooks/ | grep -v "pending-decision-list" | grep -v "pending-decision-add" | grep -v "pending-decision-resolve"
# Zero matches — no direct file reads
```

## Acceptance Criteria

- [x] `pending-decision-list` returns only session-scoped decisions
- [x] Output includes `stale` count
- [x] `--unresolved` and `--type` filters work on scoped subset
- [x] Tests prove list filtering, resolve rejection, and add stamping
- [x] Playbook uses scoped CLI, no direct JSON reads

## Key Decisions

- `pending-decision-list` automatically scopes — callers don't need to pass session IDs
- Stale decisions are hidden but counted (`stale` field) for transparency
- Non-existent `pending-decisions` command was replaced with `pending-decision-list | jq '.unresolved'`
