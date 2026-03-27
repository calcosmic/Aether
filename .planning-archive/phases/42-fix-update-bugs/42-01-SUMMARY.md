---
phase: 42-fix-update-bugs
plan: 01
type: execute
subsystem: update-system
tags: [atomic-writes, counter-fix, bug-fix]
dependency_graph:
  requires: []
  provides: [UPDATE-01, UPDATE-02]
  affects: [bin/lib/update-transaction.js]
tech_stack:
  added: []
  patterns: [atomic-write-pattern, temp-file-rename]
key_files:
  created: []
  modified:
    - bin/lib/update-transaction.js
decisions:
  - Use temp file + rename pattern for atomic file writes
  - Consolidate .sh permission handling inside copyFileAtomic
  - Move hash comparison outside dry-run block for accurate counting
metrics:
  duration: 8 minutes
  completed_date: 2026-02-22
---

# Phase 42 Plan 01: Fix Update Bugs - Summary

**One-liner:** Fixed atomic writes and counter accuracy in the update transaction system to prevent file corruption during interrupted updates and ensure dry-run mode reports accurate counts.

---

## What Was Built

### 1. copyFileAtomic() Helper Method
Added a new private method to the `UpdateTransaction` class that implements atomic file writes using the temp file + rename pattern:

- Creates unique temp file: `${destPath}.tmp.${process.pid}.${Date.now()}`
- Copies source to temp file using `fs.copyFileSync()`
- Atomically renames temp to destination using `fs.renameSync()` (POSIX guarantees atomicity)
- Sets executable permission (0o755) if destination ends with `.sh`
- Cleans up temp file on any failure
- Throws the original error after cleanup on failure

### 2. syncDirWithCleanup() Atomic Writes
Modified the `syncDirWithCleanup()` method to use the new `copyFileAtomic()` method:

- Replaced direct `fs.copyFileSync(srcPath, destPath)` with `this.copyFileAtomic(srcPath, destPath)`
- Removed redundant chmod block since `copyFileAtomic` handles `.sh` permissions internally
- Interrupted writes now leave the original file intact

### 3. syncAetherToRepo() Counter Fix and Atomic Writes
Fixed two issues in the `syncAetherToRepo()` method:

**Counter Bug Fix:**
- Moved hash comparison logic outside the `!dryRun` block so it runs for both dry-run and actual copy
- `copied++` now only increments when `shouldCopy` is true
- In dry-run mode, only counts files that would actually be copied (after hash check)
- Previously, `copied++` was outside the conditional block, causing dry-run to report all source files as "copied" even when nothing was written

**Atomic Writes:**
- Replaced direct `fs.copyFileSync(srcPath, destPath)` with `this.copyFileAtomic(srcPath, destPath)`
- Removed redundant chmod block since `copyFileAtomic` handles `.sh` permissions

---

## Commits

| Hash | Message | Files |
|------|---------|-------|
| 2941a32 | feat(42-01): add copyFileAtomic helper method for atomic file writes | bin/lib/update-transaction.js |
| c05657c | feat(42-01): use copyFileAtomic in syncDirWithCleanup | bin/lib/update-transaction.js |
| 0555270 | feat(42-01): fix counter bug and use atomic writes in syncAetherToRepo | bin/lib/update-transaction.js |

---

## Verification

All existing tests pass:
- 422 tests passed
- 9 tests skipped

Manual verification:
- `grep -c "copyFileAtomic" bin/lib/update-transaction.js` returns 3 (method definition + 2 usages)
- `grep -n "copied++" bin/lib/update-transaction.js` shows counter only inside conditional blocks
- No direct `fs.copyFileSync` calls remain in sync methods (only in `copyFileAtomic` itself)

---

## Deviations from Plan

None - plan executed exactly as written.

---

## Key Technical Details

### Atomic Write Pattern
The atomic write pattern used is:
1. Write to a temp file in the same directory as the target
2. Use `fs.renameSync()` to atomically move temp file to target
3. POSIX guarantees that `rename()` is atomic - readers see either the old file or the new file, never a partial file

### Counter Logic Fix
The previous bug:
```javascript
if (!dryRun) {
  // ... copy logic ...
}
copied++;  // BUG: This ran for every file, even in dry-run mode
```

The fixed logic:
```javascript
// Hash comparison runs for both dry-run and actual copy
let shouldCopy = true;
if (fs.existsSync(destPath)) {
  // ... hash comparison ...
  if (srcHash === destHash) {
    shouldCopy = false;
    skipped++;
  }
}

if (!dryRun) {
  if (shouldCopy) {
    this.copyFileAtomic(srcPath, destPath);
    copied++;  // Only count when actually copied
  }
} else {
  if (shouldCopy) {
    copied++;  // In dry-run, only count files that would be copied
  }
}
```

---

## Self-Check: PASSED

- [x] copyFileAtomic() method exists with temp file + rename pattern
- [x] syncDirWithCleanup() uses copyFileAtomic() instead of fs.copyFileSync()
- [x] syncAetherToRepo() uses copyFileAtomic() instead of fs.copyFileSync()
- [x] syncAetherToRepo() counter only increments when files are actually copied
- [x] All existing tests pass (422 passed, 9 skipped)
- [x] SUMMARY.md created
- [x] STATE.md will be updated

---

*Completed: 2026-02-22*
