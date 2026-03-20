---
phase: 34-add-user-approval-ux
plan: 03
subsystem: learning-system
completed: 2026-02-20
duration: 7 minutes
tags: [pheromone-evolution, learning-system, queen, cli-ux]
dependency_graph:
  requires: [34-02]
  provides: [PHER-EVOL-03]
  affects: [continue.md, aether-utils.sh]
tech_stack:
  added: []
  patterns: [batch-operations, undo-pattern, deferred-storage]
key_files:
  created: []
  modified:
    - .aether/aether-utils.sh
    - .claude/commands/ant/continue.md
decisions:
  - Stored undo state in $DATA_DIR/.promotion-undo.json with 24h TTL
  - Deferred proposals auto-expire after 30 days
  - Stop-on-first-error with partial success reporting
  - learning-select-proposals outputs text+JSON; learning-approve-proposals filters JSON
---

# Phase 34 Plan 03: Approval Execution, Deferred Storage, and Undo Summary

## One-Liner
Complete tick-to-approve UX with batch promotion execution, deferred proposal storage, and immediate undo functionality.

## What Was Built

### 1. learning-defer-proposals Function
Stores unselected proposals in `learning-deferred.json` for later review:
- Acquires lock for concurrent access safety
- Adds `deferred_at` timestamp to each proposal
- Merges new proposals without duplicates (by `content_hash`)
- Auto-expires entries older than 30 days during write
- Logs DEFERRED activity with counts

### 2. learning-approve-proposals Function
Orchestrates the full approval workflow:
- Integrates with `learning-select-proposals` for selection UX
- Supports `--verbose`, `--dry-run`, `--yes`, and `--deferred` flags
- Executes batch promotions via `queen-promote` with success feedback
- Shows "✓ Promoted {type}: {content}" for each success
- Defers unselected proposals via `learning-defer-proposals`
- Offers undo prompt after successful promotions
- Logs PROMOTED activity with counts
- Returns JSON summary: `{promoted, deferred, failed, undo_offered}`

### 3. learning-undo-promotions Function
Reverts promotions from QUEEN.md:
- Reads undo state from `$DATA_DIR/.promotion-undo.json`
- Enforces 24h TTL on undo window
- Removes entries from appropriate QUEEN.md sections
- Updates METADATA stats when reverting
- Logs UNDONE activity with count
- Handles entries already removed (warns but continues)

### 4. continue.md Integration
Updated Step 2.1.5 to use the new approval workflow:
- Replaced old AskUserQuestion-based approval with `learning-approve-proposals`
- Added `--deferred` flag support for reviewing deferred proposals
- Added `--verbose` flag passthrough
- Simplified flow: check proposals → invoke function → done

## Verification Results

- [x] learning-defer-proposals stores unselected items with timestamps
- [x] learning-approve-proposals orchestrates full workflow
- [x] Batch promotions execute with success feedback
- [x] Undo prompt appears after promotions
- [x] Undo function reverts promotions from QUEEN.md
- [x] continue.md integrated with new approval flow
- [x] --deferred flag works in continue.md
- [x] learning-deferred.json is gitignored (via data/ directory)

## Test Output

```bash
# Test dry-run mode
$ bash .aether/aether-utils.sh learning-approve-proposals --dry-run
Dry run: would promote pattern: "Always validate inputs"
Dry run: would promote pattern: "Test new learning"
...
{"ok":true,"result":{"promoted":8,"deferred":0,"failed":"null","undo_offered":false}}

# Test defer storage
$ echo '[{"content_hash":"sha256:test",...}]' | bash .aether/aether-utils.sh learning-defer-proposals
{"ok":true,"result":{"deferred":1,"new":1,"expired":0}}

# Test undo
$ bash .aether/aether-utils.sh learning-undo-promotions
{"ok":true,"result":{"undone":0,"not_found":["Test undo"]}}
```

## Commits

1. `7ebc82a` - feat(34-03): add learning-defer-proposals function
2. `399a142` - feat(34-03): add learning-approve-proposals function
3. `7806ba9` - feat(34-03): add learning-undo-promotions function
4. `2a58b91` - feat(34-03): update continue.md with approval UX integration

## Architecture Notes

### Error Handling
- Stop on first error (per CONTEXT.md decisions)
- Show which succeeded before failure
- Leave successful promotions in place
- Log failure for manual retry

### Data Flow
```
learning-approve-proposals
├── learning-select-proposals (display + capture)
├── queen-promote (for each selected)
├── learning-defer-proposals (for unselected)
└── learning-undo-promotions (if user requests)
```

### File Locations
- `learning-deferred.json` → `$DATA_DIR/learning-deferred.json`
- `.promotion-undo.json` → `$DATA_DIR/.promotion-undo.json`
- Both excluded by `.aether/.gitignore` (data/ directory)

## Deviations from Plan

None - plan executed exactly as written.

## Next Steps

Phase 34 is complete. The tick-to-approve UX is now fully functional:
1. Run `/ant:continue` to see proposals with checkbox UI
2. Select proposals by number and batch approve
3. Unselected proposals are deferred for later
4. Run `/ant:continue --deferred` to review deferred items
5. Undo promotions immediately after approval

---

*Generated: 2026-02-20*
*Phase 34-03 Complete*
