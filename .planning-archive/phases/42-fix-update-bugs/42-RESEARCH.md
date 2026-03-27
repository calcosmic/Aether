# Phase 42: Fix Update Bugs - Research

**Researched:** 2026-02-22
**Domain:** File sync operations, atomic writes, counter accuracy, directory cleanup
**Confidence:** HIGH

## Summary

This phase fixes three specific bugs in the `/ant:update` command's sync functionality:

1. **UPDATE-01 (Atomic Writes):** The `syncAetherToRepo` function uses direct `fs.copyFileSync()` which can leave partial files if interrupted. The `syncDirWithCleanup` function has the same issue. Both need to use atomic write pattern (write to temp file, then rename).

2. **UPDATE-02 (Counter Bug):** In dry-run mode, `syncAetherToRepo` increments `copied++` OUTSIDE the `!dryRun` block, causing the counter to report all source files as "copied" even when nothing was actually copied. The `syncDirWithCleanup` function has the correct pattern (increment inside the copy operation), but `syncAetherToRepo` has the bug.

3. **UPDATE-03 (Stale Directories):** The `cleanupStaleAetherDirs` function already removes `.aether/agents/` and `.aether/commands/` (from pre-3.0.0), but the user reports "double documentation" -- likely stale directories under `.aether/` that aren't in the cleanup list.

**Primary recommendation:** Fix atomic writes using the existing `atomic-write.sh` pattern (temp file + rename), fix counter by moving `copied++` inside the actual copy block, extend stale cleanup to cover more directories.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Sync Model
- **Dry-run first, then tick-to-approve** — Show what would change, user approves via tick-to-approve UI
- **Hub required** — Update only syncs from `~/.aether/` (hub), must be installed
- **Full mirror sync** — Target `.aether/` becomes authoritative mirror of source (except protected)

#### Protected Files
- **Always preserve:**
  - `data/` — Colony state
  - `dreams/` — Session notes
  - `oracle/` — Research progress
  - `midden/` — Failure tracking
  - `QUEEN.md` — User's wisdom file (CRITICAL — never touch)
- **Clean everything else** — System dirs, docs, utils, templates get synced

#### Trash Safety
- **Move to trash, don't delete** — Removed files go to `.aether/.trash/`
- **Manual cleanup** — Never auto-purge trash, user cleans when ready
- **One session per trash** — Trash folder timestamped for easy identification

#### Approval UI (Tick-to-Approve)
- **Group by action type** — Show additions first, then removals, then updates
- **All pre-selected** — All changes ticked by default, user unticks to skip
- **File-by-file paths** — List each file with full path

#### Conflict Handling
- **Ask per conflict** — If local file was modified from source, show diff and let user choose
- **Preserve user intent** — Don't blindly overwrite customized files

#### Cleanup Behavior
- **Remove empty directories** — After sync, clean up any empty directories left behind
- **Entire .aether/ scope** — Sync covers all of `.aether/` except protected dirs

#### Reporting
- **File-by-file after sync** — Show each file added, removed, updated with paths
- **Clear summary** — "Added X, Updated Y, Removed Z, Skipped N"

### Claude's Discretion

None explicitly stated — all decisions were locked.

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.

</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| UPDATE-01 | Fix atomic writes in syncDirWithCleanup - Use atomic write pattern (write to temp, then rename) | See "Atomic Write Pattern" section below — pattern exists in `.aether/utils/atomic-write.sh` |
| UPDATE-02 | Fix counter bug in dry-run mode - Counter only increments when files are actually copied | See "Counter Bug Analysis" section below — bug identified at line 723 in `update-transaction.js` |
| UPDATE-03 | Clean old directories from user repos - Remove directories that are no longer in source | See "Stale Directory Cleanup" section below — `cleanupStaleAetherDirs` exists but may need extension |
</phase_requirements>

## Standard Stack

### Core
| Library/Pattern | Version | Purpose | Why Standard |
|-----------------|---------|---------|--------------|
| `fs.copyFileSync` | Node built-in | File copying | Standard Node.js API |
| `fs.renameSync` | Node built-in | Atomic rename | POSIX atomic rename guarantee |
| `fs.mkdtempSync` | Node built-in | Temp directory creation | Unique temp file names |
| `crypto.createHash` | Node built-in | SHA-256 hashing | Already used for skip logic |

### Supporting
| Utility | Location | Purpose | When to Use |
|---------|----------|---------|-------------|
| `atomic-write.sh` | `.aether/utils/atomic-write.sh` | Atomic write pattern for shell | Bash scripts needing safe writes |
| `file-lock.sh` | `.aether/utils/file-lock.sh` | File locking | Concurrent access prevention |
| `UpdateTransaction` | `bin/lib/update-transaction.js` | Two-phase commit | CLI update operations |

### Existing Patterns
| Pattern | Location | Use For |
|---------|----------|---------|
| `hashFileSync()` | `update-transaction.js:221-229` | Skip unchanged files |
| `listFilesRecursive()` | `update-transaction.js:516-532` | Directory traversal |
| `cleanEmptyDirs()` | `update-transaction.js:633-646` | Remove empty dirs after cleanup |

## Architecture Patterns

### Atomic Write Pattern (UPDATE-01 Fix)

**What:** Write to a temp file, then atomically rename to target. Guarantees either complete file or no file — never partial.

**When to use:** Any file write that must not be corrupted by crash/interruption.

**Example from `.aether/utils/atomic-write.sh`:**
```bash
# Lines 47-91 (simplified)
atomic_write() {
    local target_file="$1"
    local content="$2"
    local target_dir=$(dirname "$target_file")
    mkdir -p "$target_dir"

    # Create unique temp file
    local temp_file="${TEMP_DIR}/$(basename "$target_file").$$.$(date +%s%N).tmp"

    # Write content to temp file
    echo "$content" > "$temp_file" || { rm -f "$temp_file"; return 1; }

    # Validate JSON if applicable (optional but recommended)
    if [[ "$target_file" == *.json ]]; then
        python3 -c "import json; json.load(open('$temp_file'))" || { rm -f "$temp_file"; return 1; }
    fi

    # Atomic rename (overwrites target if exists)
    mv "$temp_file" "$target_file" || { rm -f "$temp_file"; return 1; }
}
```

**JavaScript equivalent for `update-transaction.js`:**
```javascript
// New helper method to add to UpdateTransaction class
copyFileAtomic(srcPath, destPath) {
  const tempPath = `${destPath}.tmp.${process.pid}.${Date.now()}`;

  // Write to temp first
  fs.copyFileSync(srcPath, tempPath);

  // Atomic rename
  fs.renameSync(tempPath, destPath);

  // Set permissions if needed
  if (destPath.endsWith('.sh')) {
    fs.chmodSync(destPath, 0o755);
  }
}
```

### Counter Fix Pattern (UPDATE-02 Fix)

**Bug location:** `bin/lib/update-transaction.js:723`

**Current (buggy) code in `syncAetherToRepo`:**
```javascript
// Lines 698-724
for (const relPath of srcFiles) {
  const srcPath = path.join(srcDir, relPath);
  const destPath = path.join(destDir, relPath);

  if (!dryRun) {
    fs.mkdirSync(path.dirname(destPath), { recursive: true });
    // Hash comparison...
    if (shouldCopy) {
      fs.copyFileSync(srcPath, destPath);
      // chmod if needed
    }
  }
  copied++;  // BUG: This runs even in dry-run mode!
}
```

**Correct pattern (already in `syncDirWithCleanup` lines 560-600):**
```javascript
if (!dryRun) {
  for (const relPath of srcFiles) {
    // ... hash comparison ...
    if (shouldCopy) {
      fs.copyFileSync(srcPath, destPath);
      if (relPath.endsWith('.sh')) {
        fs.chmodSync(destPath, 0o755);
      }
      copied++;  // CORRECT: Only count when actually copied
    }
  }
} else {
  copied = srcFiles.length;  // dry-run: report what WOULD be copied
}
```

### Stale Directory Cleanup (UPDATE-03)

**Current `cleanupStaleAetherDirs()` at lines 780-821:**
```javascript
cleanupStaleAetherDirs(repoPath) {
  const staleItems = [
    {
      path: path.join(repoPath, '.aether', 'agents'),
      label: '.aether/agents/ (stale duplicate)',
      type: 'dir',
    },
    {
      path: path.join(repoPath, '.aether', 'commands'),
      label: '.aether/commands/ (stale duplicate)',
      type: 'dir',
    },
    {
      path: path.join(repoPath, '.aether', 'planning.md'),
      label: '.aether/planning.md (phantom file)',
      type: 'file',
    },
  ];
  // ... removal logic ...
}
```

**Extension needed:** The user reports "double documentation" — potentially stale directories like:
- Old documentation directories that were renamed/reorganized
- Directories from previous architecture versions (pre-v4.0)

**Key insight from CONTEXT.md:** The sync should be "authoritative" — target `.aether/` becomes a mirror of source, so the cleanup should remove ANY directory not in source (except protected).

### Anti-Patterns to Avoid

- **Direct `fs.copyFileSync` without atomic pattern:** Can leave partial files on crash
- **Counter increment outside actual operation:** Inaccurate reporting
- **Hardcoded stale list:** Should derive from source comparison, not hardcoded list

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Atomic writes | Custom temp file logic | `atomic_write()` pattern from utils | Already handles temp dir, cleanup, validation |
| File hashing | Custom hash function | `hashFileSync()` already in codebase | SHA-256, consistent format `sha256:hex` |
| Directory traversal | Recursive readdir | `listFilesRecursive()` | Handles dotfiles, deep nesting |
| Empty dir cleanup | Custom rmdir loop | `cleanEmptyDirs()` | Already handles recursive cleanup |

**Key insight:** The codebase already has battle-tested patterns. The bugs are inconsistencies between two similar functions (`syncDirWithCleanup` vs `syncAetherToRepo`) where one has the correct pattern and the other doesn't.

## Common Pitfalls

### Pitfall 1: Counter Increment Outside Dry-Run Block
**What goes wrong:** In dry-run mode, `copied` counter equals total source files, not files that would actually be copied (after hash comparison).
**Why it happens:** Developer copied code structure but put counter increment outside the conditional.
**How to avoid:** Counter must increment ONLY when actual file operation happens (or would happen after hash check in dry-run).
**Warning signs:** Dry-run output shows all files as "copied" even when most would be skipped.

### Pitfall 2: Non-Atomic Writes During Sync
**What goes wrong:** Process crash during `fs.copyFileSync()` leaves partial/corrupt file at destination.
**Why it happens:** Direct copy doesn't use temp file + rename pattern.
**How to avoid:** Always write to temp file in same directory, then `fs.renameSync()` for atomicity.
**Warning signs:** Corrupted files after interrupted update, partial JSON files.

### Pitfall 3: Hardcoded Stale Directory List
**What goes wrong:** New stale directories from future refactors aren't cleaned up.
**Why it happens:** `cleanupStaleAetherDirs()` uses hardcoded list instead of comparing against source.
**How to avoid:** The `syncAetherToRepo` already has cleanup logic (lines 727-764) that removes files not in source — extend this to directories or use same pattern.
**Warning signs:** "Double documentation" or duplicate directories accumulating in user repos.

### Pitfall 4: Missing Protected Directory Check
**What goes wrong:** Sync removes user data directories that should be preserved.
**Why it happens:** `shouldExclude()` doesn't include all protected paths.
**How to avoid:** Ensure `EXCLUDE_DIRS` (line 175) includes: `data`, `dreams`, `oracle`, `midden`, and any `QUEEN.md` file.
**Warning signs:** User loses colony state or wisdom notes after update.

## Code Examples

### Atomic Copy for JavaScript (UPDATE-01 Implementation)

```javascript
// Add to UpdateTransaction class
/**
 * Copy file atomically using temp file + rename
 * @param {string} srcPath - Source file path
 * @param {string} destPath - Destination file path
 * @private
 */
copyFileAtomic(srcPath, destPath) {
  const tempDir = path.dirname(destPath);
  const tempPath = path.join(tempDir, `.tmp.${process.pid}.${Date.now()}.${path.basename(destPath)}`);

  try {
    // Write to temp file
    fs.copyFileSync(srcPath, tempPath);

    // Atomic rename (POSIX guarantees atomicity)
    fs.renameSync(tempPath, destPath);

    // Set executable permission for shell scripts
    if (destPath.endsWith('.sh')) {
      fs.chmodSync(destPath, 0o755);
    }
  } catch (err) {
    // Clean up temp file on failure
    try { fs.unlinkSync(tempPath); } catch { /* ignore */ }
    throw err;
  }
}
```

### Fixed Counter Logic (UPDATE-02 Implementation)

```javascript
// Fixed syncAetherToRepo (lines 698-724)
for (const relPath of srcFiles) {
  const srcPath = path.join(srcDir, relPath);
  const destPath = path.join(destDir, relPath);

  // Hash comparison to determine if copy needed
  let shouldCopy = true;
  if (fs.existsSync(destPath)) {
    const srcHash = this.hashFileSync(srcPath);
    const destHash = this.hashFileSync(destPath);
    if (srcHash === destHash) {
      shouldCopy = false;
      skipped++;
    }
  }

  if (!dryRun) {
    fs.mkdirSync(path.dirname(destPath), { recursive: true });
    if (shouldCopy) {
      this.copyFileAtomic(srcPath, destPath); // Uses atomic pattern
      copied++;  // FIX: Only increment when actually copied
    }
  } else if (shouldCopy) {
    copied++;  // FIX: In dry-run, only count files that WOULD be copied
  }
}
```

### Extended Stale Cleanup (UPDATE-03 Implementation)

```javascript
// Extend EXCLUDE_DIRS to match protected files from CONTEXT.md
this.EXCLUDE_DIRS = ['data', 'dreams', 'oracle', 'midden', 'checkpoints', 'locks', 'temp', 'agents', 'commands', 'rules', 'archive', 'chambers', 'exchange'];

// Special file protection (QUEEN.md)
this.EXCLUDE_FILES = ['QUEEN.md'];

// In shouldExclude(), also check for protected files
shouldExclude(relPath) {
  const parts = relPath.split(path.sep);
  if (parts.some(part => this.EXCLUDE_DIRS.includes(part))) {
    return true;
  }
  // Check for protected files
  const basename = path.basename(relPath);
  if (this.EXCLUDE_FILES.includes(basename)) {
    return true;
  }
  return false;
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Direct `fs.copyFileSync` | Atomic write pattern needed | Bug identified 2026-02 | Prevents corruption |
| Counter in all cases | Counter only on actual copy | Bug identified 2026-02 | Accurate reporting |
| Hardcoded stale list | Source-comparison cleanup needed | v4.0 refactor | Clean repos |

**Deprecated/outdated:**
- `syncAetherToRepo` counter logic: Reports wrong count in dry-run, needs fix

## Open Questions

1. **Should QUEEN.md protection be file-only or directory-level?**
   - What we know: CONTEXT.md says "QUEEN.md — User's wisdom file (CRITICAL — never touch)"
   - What's unclear: Should we also protect `*.md` files user creates in `.aether/` root?
   - Recommendation: Start with just `QUEEN.md` as protected file; can extend later if needed.

2. **Trash directory naming convention?**
   - What we know: CONTEXT.md says "Trash folder timestamped for easy identification"
   - What's unclear: Exact format (`.aether/.trash/2026-02-22T04-30-00/` vs `.aether/.trash/1708578600/`)
   - Recommendation: Use ISO timestamp format for human readability: `.aether/.trash/2026-02-22T04-30-00Z/`

3. **Tick-to-approve UI implementation details?**
   - What we know: CONTEXT.md specifies grouped display, pre-selected, file-by-file paths
   - What's unclear: Is this a CLI interactive prompt or a structured output format?
   - Recommendation: For CLI, use numbered list with confirmation; for slash command, output structured diff that user can approve.

## Sources

### Primary (HIGH confidence)
- `.aether/utils/atomic-write.sh` - Atomic write pattern implementation
- `bin/lib/update-transaction.js` - Current sync implementation, UpdateTransaction class
- `.claude/commands/ant/update.md` - User-facing update command specification
- `.planning/phases/42-fix-update-bugs/42-CONTEXT.md` - Locked user decisions

### Secondary (MEDIUM confidence)
- `tests/unit/cli-sync.test.js` - Test expectations for syncDirWithCleanup
- `tests/unit/sync-dir-hash.test.js` - Hash comparison test patterns

### Tertiary (LOW confidence)
- None — all findings verified from codebase

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Node.js built-ins and existing utilities
- Architecture: HIGH - Pattern already exists in codebase
- Pitfalls: HIGH - Bugs identified in specific code locations
- Atomic writes: HIGH - Pattern exists in atomic-write.sh
- Counter bug: HIGH - Exact bug location identified (line 723)
- Stale cleanup: MEDIUM - Current implementation exists but may need extension

**Research date:** 2026-02-22
**Valid until:** 30 days (stable patterns, but sync logic may change)
