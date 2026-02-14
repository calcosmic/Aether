# Phase 6: Foundation — Safe Checkpoints & Testing Infrastructure - Research

**Researched:** 2026-02-14
**Domain:** Node.js CLI testing, Git-based checkpointing, File system mocking
**Confidence:** HIGH

## Summary

This research covers the implementation of a safe checkpoint system that never captures user data, and the testing infrastructure needed for deterministic verification of the Aether CLI.

**Key Findings:**

1. **Current checkpoint mechanism exists** - The `update` command already uses `git stash` for dirty files, but there's no explicit checkpoint/restore system for Aether state.

2. **Target functions identified** - `syncDirWithCleanup`, `hashFileSync`, and `generateManifest` are all defined in `bin/cli.js` (lines 283-352, 354-429) and need unit testing.

3. **Testing infrastructure exists** - The project uses `ava` for unit testing with 5 existing test files. No mocking libraries (sinon, proxyquire) are currently installed.

4. **Allowlist approach already implemented** - The `SYSTEM_FILES` array (lines 229-251) in `bin/cli.js` already implements an explicit allowlist pattern for system file syncing.

**Primary recommendation:** Build on the existing `SYSTEM_FILES` allowlist pattern and `git stash` foundation to create an explicit checkpoint system with metadata tracking.

## Current State Analysis

### Existing Checkpoint-Related Code

**Git stash integration (lines 515-527, 640-650):**
```javascript
function gitStashFiles(repoPath, files) {
  try {
    const fileArgs = files.map(f => `"${f}"`).join(' ');
    execSync(`git stash push -m "aether-update-backup" -- ${fileArgs}`, {
      cwd: repoPath,
      stdio: 'pipe',
    });
    return true;
  } catch (err) {
    log(`  Warning: git stash failed (${err.message}). Proceeding without stash.`);
    return false;
  }
}
```

**System files allowlist (lines 229-251):**
```javascript
const SYSTEM_FILES = [
  'aether-utils.sh',
  'coding-standards.md',
  'debugging.md',
  'DISCIPLINES.md',
  'learning.md',
  'planning.md',
  'QUEEN_ANT_ARCHITECTURE.md',
  'tdd.md',
  'verification-loop.md',
  'verification.md',
  'workers.md',
  'docs/constraints.md',
  // ... more files
];
```

### Functions Requiring Tests

| Function | Location | Lines | Purpose |
|----------|----------|-------|---------|
| `syncDirWithCleanup` | bin/cli.js | 354-429 | Syncs directories with hash-based idempotency and cleanup |
| `hashFileSync` | bin/cli.js | 283-291 | Computes SHA-256 hash of file content |
| `generateManifest` | bin/cli.js | 338-352 | Generates manifest with file hashes for integrity |
| `validateManifest` | bin/cli.js | 293-304 | Validates manifest structure |
| `syncSystemFilesWithCleanup` | bin/cli.js | 440-489 | Syncs allowlisted system files |

### Directory Structure Analysis

**What SHOULD be checkpointed (.aether/ allowlist):**
- `.aether/*.md` (coding-standards.md, debugging.md, DISCIPLINES.md, etc.)
- `.aether/docs/` (constraints.md, pathogen-schema.md, etc.)
- `.aether/utils/` (atomic-write.sh, colorize-log.sh, file-lock.sh, etc.)
- `.aether/version.json` (version metadata)

**What should NOT be checkpointed (user data):**
- `.aether/data/` (COLONY_STATE.json, activity.log, backups/, spawn-tree.txt)
- `.aether/dreams/` (user notes and dreams)
- `.aether/oracle/` (oracle outputs, discoveries, progress.md)
- `TO-DOs.md` (if exists in .aether/)

### Existing Test Infrastructure

**Test framework:** AVA (v6.0.0)
**Test location:** `tests/unit/*.test.js`
**Current test files:**
- `colony-state.test.js` - JSON validation tests
- `oracle-regression.test.js` - Oracle-discovered bug regression tests
- `spawn-tree.test.js` - Spawn tree functionality tests
- `state-loader.test.js` - State loading/unloading tests
- `validate-state.test.js` - State validation tests

**Test patterns observed:**
- Tests use `execSync` to run bash commands and parse JSON output
- Helper functions for backup/restore of test fixtures
- Temporary directory creation for isolated tests
- Error testing via try/catch with expected failures

## Checkpoint System Design Options

### Option 1: Git-Based Checkpoints (Recommended)

**Approach:** Use git commits in a dedicated branch or orphan commits

**Pros:**
- Leverages existing git infrastructure
- Built-in compression and deduplication
- Natural rollback capability
- Already used for `git stash` in update flow

**Cons:**
- Requires git repository
- Checkpoints are local to git history

**Implementation:**
```javascript
// Create checkpoint branch if not exists
// Create orphan commit with only allowlisted files
// Store checkpoint metadata in JSON
```

### Option 2: Tar Archive with Metadata

**Approach:** Create compressed tar archives with manifest

**Pros:**
- Works without git
- Portable archives
- Explicit control over contents

**Cons:**
- Manual cleanup needed
- No built-in deduplication

### Option 3: Hard Link Snapshots

**Approach:** Use hard links for copy-on-write snapshots

**Pros:**
- Space efficient
- Fast creation

**Cons:**
- Platform-specific behavior
- Complex to manage

**Recommendation:** Option 1 (Git-based) aligns with existing patterns and provides the best safety/UX tradeoff.

## Testing Strategy

### Mocking Approach

**Libraries needed:**
- `sinon` (^19.0.0) - For stubbing/spying on functions
- `proxyquire` (^2.1.3) - For mocking `fs` and `crypto` modules

**Installation:**
```bash
npm install --save-dev sinon proxyquire
```

### Test Patterns for Target Functions

**Pattern 1: Mocking fs for syncDirWithCleanup**
```javascript
const proxyquire = require('proxyquire');
const sinon = require('sinon');

// Create mock fs
const mockFs = {
  existsSync: sinon.stub(),
  readdirSync: sinon.stub(),
  mkdirSync: sinon.stub(),
  copyFileSync: sinon.stub(),
  readFileSync: sinon.stub(),
  unlinkSync: sinon.stub(),
  rmdirSync: sinon.stub(),
};

// Load cli.js with mocked fs
const cli = proxyquire('../bin/cli.js', { fs: mockFs });
```

**Pattern 2: Idempotency Property Tests**
```javascript
test('syncDirWithCleanup is idempotent', t => {
  // First sync
  const result1 = syncDirWithCleanup(src, dest);

  // Second sync should copy nothing
  const result2 = syncDirWithCleanup(src, dest);

  t.is(result2.copied, 0, 'Second sync should copy no files');
  t.is(result2.removed.length, 0, 'Second sync should remove no files');
});
```

**Pattern 3: Hash Verification Tests**
```javascript
test('hashFileSync returns consistent SHA-256 hash', t => {
  const testContent = Buffer.from('test content');
  mockFs.readFileSync.returns(testContent);

  const hash = hashFileSync('/test/file.txt');

  t.is(hash, 'sha256:' + expectedSha256);
});
```

## Implementation Approach Recommendations

### 1. Safe Checkpoint System

**Architecture:**
```
.aether/checkpoints/
  checkpoint-YYYY-MM-DD-HHMMSS.json  # Metadata
  # Git tags point to checkpoint commits
```

**Checkpoint metadata format:**
```json
{
  "checkpoint_id": "chk_20260214_143022",
  "created_at": "2026-02-14T14:30:22Z",
  "git_ref": "refs/checkpoints/chk_20260214_143022",
  "files": {
    "aether-utils.sh": "sha256:abc123...",
    "coding-standards.md": "sha256:def456..."
  },
  "excluded": [
    "data/COLONY_STATE.json",
    "dreams/",
    "oracle/"
  ]
}
```

**CLI integration:**
```bash
aether checkpoint create [--message "Before major change"]
aether checkpoint list
aether checkpoint restore <checkpoint-id>
aether checkpoint verify <checkpoint-id>  # Verify integrity
```

### 2. Testing Infrastructure

**New test files to create:**
- `tests/unit/cli-sync.test.js` - Tests for syncDirWithCleanup
- `tests/unit/cli-hash.test.js` - Tests for hashFileSync
- `tests/unit/cli-manifest.test.js` - Tests for generateManifest
- `tests/unit/checkpoint.test.js` - Tests for checkpoint system

**Test utilities:**
- `tests/unit/helpers/mock-fs.js` - Reusable fs mocking helpers
- `tests/unit/helpers/temp-dir.js` - Temporary directory management

### 3. Package Lock Commit

**Current state:** `package-lock.json` exists and is 24KB
**Action:** Ensure it's committed to git for deterministic builds

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| File hashing | Custom hash function | `crypto.createHash('sha256')` | Standard, well-tested, hardware-accelerated |
| Git operations | `child_process.exec('git ...')` | `child_process.execSync` with proper error handling | Already used in codebase, simple enough |
| Test mocking | Manual dependency injection | `proxyquire` + `sinon` | Industry standard, proven patterns |
| Temp directories | `mkdirSync('/tmp/...')` | `fs.mkdtempSync` | Secure, race-condition free |
| JSON validation | Manual parsing | `JSON.parse` with try/catch | Native, fast, standard |

## Common Pitfalls

### Pitfall 1: Accidentally Including User Data
**What goes wrong:** Checkpoint system captures `.aether/data/COLONY_STATE.json` or other user data
**Why it happens:** Using glob patterns instead of explicit allowlists
**How to avoid:** Strict allowlist validation - reject any file not in SYSTEM_FILES
**Warning signs:** Tests that write to actual `.aether/data/` instead of temp directories

### Pitfall 2: Test Pollution
**What goes wrong:** Tests create/modify files in the actual project directory
**Why it happens:** Forgetting to use temp directories or mock fs
**How to avoid:** Always use `fs.mkdtempSync()` for test file operations
**Warning signs:** Tests that pass individually but fail in sequence

### Pitfall 3: Hash Comparison Edge Cases
**What goes wrong:** Hash comparison fails for empty files or binary files
**Why it happens:** Assuming all files have content, or encoding issues
**How to avoid:** Test with empty files, binary files, and files with special characters
**Warning signs:** Intermittent test failures on different platforms

### Pitfall 4: Git State Assumptions
**What goes wrong:** Checkpoint fails in repos without git or with unusual git state
**Why it happens:** Assuming git is always available and clean
**How to avoid:** Check `isGitRepo()` before git operations, handle errors gracefully
**Warning signs:** Tests fail when run outside git repo

## Code Examples

### Example 1: Safe Checkpoint Creation
```javascript
// Source: bin/cli.js patterns
function createCheckpoint(repoPath, message) {
  const checkpointId = `chk_${formatDate(new Date())}`;
  const allowlistedFiles = getAllowlistedFiles(repoPath);

  // Verify no user data in allowlist
  for (const file of allowlistedFiles) {
    if (isUserData(file)) {
      throw new Error(`Refusing to checkpoint user data: ${file}`);
    }
  }

  // Create git checkpoint
  const filesArg = allowlistedFiles.map(f => `"${f}"`).join(' ');
  execSync(`git stash push -m "aether-checkpoint-${checkpointId}" -- ${filesArg}`, {
    cwd: repoPath,
    stdio: 'pipe',
  });

  // Generate metadata
  const manifest = generateManifest(path.join(repoPath, '.aether'));
  const metadata = {
    checkpoint_id: checkpointId,
    message,
    created_at: new Date().toISOString(),
    files: manifest.files,
  };

  return metadata;
}
```

### Example 2: Mocking Pattern for Tests
```javascript
// Source: Recommended pattern based on existing tests
const test = require('ava');
const sinon = require('sinon');
const proxyquire = require('proxyquire');

test.beforeEach(t => {
  t.context.mockFs = {
    existsSync: sinon.stub(),
    readFileSync: sinon.stub(),
    writeFileSync: sinon.stub(),
    mkdirSync: sinon.stub(),
    readdirSync: sinon.stub(),
  };

  t.context.cli = proxyquire('../bin/cli.js', {
    fs: t.context.mockFs,
  });
});

test.afterEach(t => {
  sinon.restore();
});

test('hashFileSync computes correct SHA-256', t => {
  const { mockFs, cli } = t.context;
  const testContent = Buffer.from('hello world');
  mockFs.readFileSync.returns(testContent);

  const hash = cli.hashFileSync('/test/file.txt');

  t.true(hash.startsWith('sha256:'));
  t.is(hash.length, 64 + 7); // 'sha256:' + 64 hex chars
});
```

### Example 3: Idempotency Test Pattern
```javascript
// Source: Based on existing spawn-tree.test.js patterns
test('syncDirWithCleanup is idempotent', async t => {
  const tempDir = fs.mkdtempSync('/tmp/sync-test-');
  const srcDir = path.join(tempDir, 'src');
  const destDir = path.join(tempDir, 'dest');

  // Setup source files
  fs.mkdirSync(srcDir, { recursive: true });
  fs.writeFileSync(path.join(srcDir, 'file.txt'), 'content');

  // First sync
  const result1 = syncDirWithCleanup(srcDir, destDir);
  t.is(result1.copied, 1);

  // Second sync should be no-op
  const result2 = syncDirWithCleanup(srcDir, destDir);
  t.is(result2.copied, 0);
  t.is(result2.skipped, 1);

  // Cleanup
  fs.rmSync(tempDir, { recursive: true });
});
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Blocklist for exclusions ("don't copy X") | Allowlist for inclusions ("only copy Y") | v1.0.0 | Never accidentally copy user data |
| Manual file copying | Hash-based idempotency | v1.0.0 | Only copy when content changes |
| No testing of CLI functions | Unit tests with mocking | Phase 6 | Deterministic verification |

**Deprecated/outdated:**
- None identified

## Open Questions

1. **Checkpoint storage location**
   - What we know: Git stash stores in git reflog
   - What's unclear: Whether to also store metadata in `.aether/checkpoints/`
   - Recommendation: Store metadata JSON in `.aether/checkpoints/` for visibility

2. **Checkpoint retention policy**
   - What we know: Git stash entries can accumulate
   - What's unclear: How many checkpoints to keep, auto-cleanup policy
   - Recommendation: Keep last 10 checkpoints, auto-remove old ones

3. **Integration with existing update flow**
   - What we know: Update already uses git stash
   - What's unclear: Whether checkpoint should replace or supplement stash
   - Recommendation: Checkpoint is user-initiated, stash remains for update protection

## Sources

### Primary (HIGH confidence)
- `/Users/callumcowie/repos/Aether/bin/cli.js` - Core functions identified (lines 283-489)
- `/Users/callumcowie/repos/Aether/package.json` - Test framework configuration
- `/Users/callumcowie/repos/Aether/tests/unit/*.test.js` - Existing test patterns

### Secondary (MEDIUM confidence)
- `/Users/callumcowie/repos/Aether/.aether/` directory structure - File classification
- Git documentation for `git stash` and `git commit-tree` for orphan commits

### Tertiary (LOW confidence)
- sinon documentation (https://sinonjs.org/) - Mocking patterns
- proxyquire documentation (https://github.com/thlorenz/proxyquire) - Module mocking

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Ava already in use, sinon/proxyquire are industry standard
- Architecture: HIGH - Based on existing code patterns in cli.js
- Pitfalls: MEDIUM - Inferred from common Node.js testing issues

**Research date:** 2026-02-14
**Valid until:** 2026-03-14 (30 days for stable Node.js ecosystem)
