---
phase: 42-fix-update-bugs
plan: 02
type: execute
subsystem: update-system
tags: [update, safety, trash, protection]
dependency_graph:
  requires: [42-RESEARCH, 42-CONTEXT]
  provides: [42-03]
  affects: [update-transaction.js]
tech_stack:
  added: []
  patterns:
    - "Trash-safe removal with timestamped folders"
    - "File-level protection via EXCLUDE_FILES"
    - "Cross-device move fallback (copy+delete)"
key_files:
  created: []
  modified:
    - bin/lib/update-transaction.js
    - tests/unit/update-transaction.test.js
decisions:
  - "Move to trash instead of delete for user safety"
  - "Timestamped trash folders for easy identification"
  - "Add oracle, midden, exchange to EXCLUDE_DIRS"
  - "Add EXCLUDE_FILES array for QUEEN.md protection"
metrics:
  duration_minutes: 35
  tasks_completed: 3
  test_changes: 3
  lines_added: ~80
  lines_modified: ~30
---

# Phase 42 Plan 02: Stale Directory Cleanup with Trash Safety

## Summary

Extended stale directory cleanup in the update system to protect user data and move removed files to trash instead of deleting them permanently. Added missing protected directories (oracle, midden, exchange) and file-level protection for QUEEN.md.

## One-Liner

Trash-safe cleanup with protected directories (oracle, midden, exchange) and QUEEN.md file protection.

## What Was Built

### Changes to `bin/lib/update-transaction.js`

1. **Extended EXCLUDE_DIRS** (line 176)
   - Added 'oracle', 'midden', 'exchange' to protected directories
   - These directories contain user data that must never be synced or cleaned

2. **Added EXCLUDE_FILES array** (line 179)
   - New array for file-level protection
   - Contains 'QUEEN.md' (user's wisdom file)

3. **Extended shouldExclude() method** (lines 689-703)
   - Now checks both EXCLUDE_DIRS (directory-level) and EXCLUDE_FILES (file-level)
   - QUEEN.md is protected regardless of its location

4. **Added moveToTrash() method** (lines 822-848)
   - Moves files/directories to `.aether/.trash/{timestamp}/`
   - Uses atomic rename for same-device moves
   - Falls back to copy+delete for cross-device moves
   - Returns boolean success status

5. **Updated cleanupStaleAetherDirs() method** (lines 850-922)
   - Added comprehensive documentation about trash behavior
   - Added safety check logging for protected directories
   - Now uses moveToTrash() instead of direct deletion
   - Returns trashDir path in result for user reference

### Changes to `tests/unit/update-transaction.test.js`

1. **Added new mocks** (lines 17-20)
   - `renameSync`, `statSync`, `cpSync` for trash operations

2. **Updated cleanup tests** (lines 971-1041)
   - Tests now verify trash behavior instead of direct deletion
   - Verify renameSync is called for trash moves
   - Verify trashDir is returned in result
   - Updated error handling test for new fallback logic

## Verification Results

All 422 tests pass (9 skipped):

```
✔ cleanupStaleAetherDirs moves existing stale directories and files to trash
✔ cleanupStaleAetherDirs is idempotent — returns empty when nothing to clean
✔ cleanupStaleAetherDirs handles trash move errors gracefully
```

Manual verification:
- EXCLUDE_DIRS includes: oracle, midden, exchange ✅
- EXCLUDE_FILES array exists with QUEEN.md ✅
- shouldExclude() checks both arrays ✅
- moveToTrash() method exists ✅
- cleanupStaleAetherDirs uses moveToTrash ✅
- Protected paths documented in comments ✅
- trashDir returned in result ✅

## Deviations from Plan

None - plan executed exactly as written.

## Auth Gates

None encountered.

## Commits

| Hash | Message |
|------|---------|
| ada931d | feat(42-02): add oracle, midden, exchange to EXCLUDE_DIRS and create EXCLUDE_FILES |
| 09cb2d2 | feat(42-02): add trash-safe removal to cleanupStaleAetherDirs |
| f1dd796 | test(42-02): update tests for trash-safe cleanup |

## Self-Check

- [x] All files modified exist
- [x] All commits exist in git history
- [x] Tests pass
- [x] No syntax errors
- [x] Documentation updated

## Self-Check: PASSED

## Notes

The trash system provides users with a safety net - removed files go to `.aether/.trash/2026-02-22T04-30-00/` style folders rather than being permanently deleted. Users can inspect and manually clean trash when ready. This aligns with the CONTEXT.md decision "Move to trash, don't delete".

Protected paths are now comprehensive:
- **Directories**: data, dreams, oracle, midden, checkpoints, locks, temp, agents, commands, rules, archive, chambers, exchange
- **Files**: QUEEN.md
