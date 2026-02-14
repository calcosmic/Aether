# Architecture Research: v1.1 Bug Fixes Integration

**Domain:** Aether Colony System - CLI-based multi-agent orchestration framework
**Researched:** 2026-02-14
**Confidence:** HIGH

## Executive Summary

This research addresses how four critical bug fixes integrate with the existing Aether colony architecture:
1. Phase advancement / state tracking bugs causing loops
2. Update system with proper version tracking and rollback
3. Safe checkpoint system that doesn't stash user data
4. Testing strategy for file sync operations

The fixes must work within the established four-layer architecture (Queen, Constraints, Worker, Utility) while maintaining compatibility with existing state management patterns in COLONY_STATE.json.

## Current Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        QUEEN LAYER                           │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐        │
│  │  init   │  │  plan   │  │  build  │  │continue │        │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘        │
│       │            │            │            │              │
├───────┴────────────┴────────────┴────────────┴──────────────┤
│                      CONSTRAINTS LAYER                       │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              focus/avoid rules (pheromones)          │    │
│  └─────────────────────────────────────────────────────┘    │
├─────────────────────────────────────────────────────────────┤
│                        WORKER LAYER                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │ Builder  │  │ Watcher  │  │  Scout   │  │  Oracle  │    │
│  │  🔨      │  │  👁️      │  │  🔍      │  │  🔮      │    │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘    │
├─────────────────────────────────────────────────────────────┤
│                        UTILITY LAYER                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │aether-   │  │  file-   │  │ atomic-  │  │  state-  │    │
│  │utils.sh  │  │  lock.sh │  │ write.sh │  │ loader.sh│    │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘    │
├─────────────────────────────────────────────────────────────┤
│                        DATA LAYER                            │
│  ┌──────────────┐ ┌──────────────┐ ┌────────────────────┐  │
│  │COLONY_STATE  │ │ spawn-tree   │ │     flags.json     │  │
│  │   .json      │ │   .txt       │ │                    │  │
│  └──────────────┘ └──────────────┘ └────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Current Implementation |
|-----------|----------------|------------------------|
| Queen Layer | Orchestration, phase management, worker spawning | `.opencode/agents/aether-queen.md` |
| Constraints Layer | Declarative focus/avoid rules | `.aether/data/constraints.json` |
| Worker Layer | Task execution with depth limits | Spawned via `task` tool, tracked in `spawn-tree.txt` |
| Utility Layer | Deterministic operations, state management | `.aether/aether-utils.sh` |
| Data Layer | Persistent state, activity logs | `.aether/data/` directory |

## Fix 1: Phase Advancement / State Tracking Loops

### Problem Analysis

Phase advancement loops occur when:
1. State validation passes but phase transition logic is flawed
2. Worker completion doesn't properly update COLONY_STATE.json
3. Race conditions between concurrent workers updating state
4. Missing idempotency checks in phase advancement

### Recommended Architecture

```
Phase Advancement Flow:
┌─────────────────┐
│ Worker Complete │
└────────┬────────┘
         │
         ▼
┌─────────────────┐     ┌─────────────────┐
│  Acquire Lock   │────▶│  Lock Timeout   │
└────────┬────────┘     │  (fail fast)    │
         │              └─────────────────┘
         ▼
┌─────────────────┐
│ Read Current    │
│ State (validate)│
└────────┬────────┘
         │
         ▼
┌─────────────────┐     ┌─────────────────┐
│ Check Phase     │────▶│ Already at      │
│ Already Complete│────▶│ Target? Exit    │
└────────┬────────┘     └─────────────────┘
         │ No
         ▼
┌─────────────────┐
│ Verify All      │
│ Workers Done    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐     ┌─────────────────┐
│ Update State    │────▶│ Atomic Write    │
│ (phase++)       │     │ (with backup)   │
└────────┬────────┘     └─────────────────┘
         │
         ▼
┌─────────────────┐
│ Release Lock    │
└─────────────────┘
```

### Integration Points

| Integration | How | File/Component |
|-------------|-----|----------------|
| State validation | Reuse `validate-state colony` command | `aether-utils.sh` |
| Lock acquisition | Use existing `file-lock.sh` | `utils/file-lock.sh` |
| Atomic writes | Use `atomic_write` function | `utils/atomic-write.sh` |
| Idempotency key | Add `phase_transition_id` to events | `COLONY_STATE.json` |

### Data Flow Changes

**Current:**
```javascript
// Direct state mutation (risky)
state.current_phase++;
fs.writeFileSync('COLONY_STATE.json', JSON.stringify(state));
```

**Recommended:**
```javascript
// Pattern from aether-utils.sh error-add command
const idempotencyKey = `phase_adv_${Date.now()}_${randomBytes(2).toString('hex')}`;

// Check if already advanced (idempotency)
const existing = state.events.find(e =>
  e.type === 'phase_advance' &&
  e.from_phase === currentPhase &&
  e.idempotency_key === idempotencyKey
);
if (existing) {
  return { alreadyAdvanced: true };
}

// Atomic update with backup
const updated = jqCommand(/* phase++ and add event */);
atomicWrite('COLONY_STATE.json', updated);
```

## Fix 2: Update System with Version Tracking and Rollback

### Problem Analysis

Current update system issues:
1. No proper version comparison before syncing files
2. Registry updates happen before file sync success
3. No rollback mechanism if sync fails mid-operation
4. Version.json not updated atomically with file changes

### Recommended Architecture

```
Update Flow with Rollback:
┌─────────────────────────────────────────────────────────────┐
│                     PREPARATION PHASE                        │
├─────────────────────────────────────────────────────────────┤
│  1. Read hub version.json                                    │
│  2. Read local version.json                                  │
│  3. Compare versions (semver)                                │
│  4. If up-to-date: exit 0                                    │
└────────────────────────┬────────────────────────────────────┘
                         │ Update needed
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                     BACKUP PHASE                             │
├─────────────────────────────────────────────────────────────┤
│  1. Create backup manifest (current state)                   │
│  2. Copy files to .aether/backup/<timestamp>/                │
│  3. Store manifest in backup dir                             │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                     SYNC PHASE                               │
├─────────────────────────────────────────────────────────────┤
│  1. Sync system files (allowlist-based)                      │
│  2. Sync commands (claude/opencode)                          │
│  3. Sync agents                                              │
│  4. Validate each file with hash check                       │
│  5. On failure: jump to ROLLBACK                             │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                     COMMIT PHASE                             │
├─────────────────────────────────────────────────────────────┤
│  1. Update version.json (atomic)                             │
│  2. Update registry.json (atomic)                            │
│  3. Clean up backup (async, after delay)                     │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                     ROLLBACK PHASE (on failure)              │
├─────────────────────────────────────────────────────────────┤
│  1. Read backup manifest                                     │
│  2. Restore files from backup                                │
│  3. Verify restoration                                       │
│  4. Log rollback event                                       │
└─────────────────────────────────────────────────────────────┘
```

### Integration Points

| Integration | How | Component |
|-------------|-----|-----------|
| Version comparison | Use semver library | `package.json` dependency |
| File hashing | Use existing `hashFileSync` | `cli.js` |
| Atomic manifest | Extend manifest format | `validateManifest()` in `cli.js` |
| Registry updates | Use `registry-add` command | `aether-utils.sh` |

### Manifest Format Extension

```javascript
// backup-manifest.json
{
  "backup_id": "bkp_20260214_120000",
  "created_at": "2026-02-14T12:00:00Z",
  "from_version": "1.0.0",
  "to_version": "1.1.0",
  "files": [
    {
      "path": ".aether/aether-utils.sh",
      "hash_before": "sha256:abc...",
      "hash_after": "sha256:def...",
      "action": "modified"
    },
    {
      "path": ".claude/commands/ant/new-cmd.md",
      "hash_before": null,
      "hash_after": "sha256:ghi...",
      "action": "added"
    }
  ],
  "rollback_point": {
    "version_json": "sha256:...",
    "registry_entry": { /* ... */ }
  }
}
```

## Fix 3: Safe Checkpoint System

### Problem Analysis

Current `autofix-checkpoint` command has issues:
1. Uses `git stash` which can stash user work, not just Aether files
2. No verification that stash contains only Aether-managed files
3. Rollback can restore stale state if user made commits

### Recommended Architecture

```
Safe Checkpoint Flow:
┌─────────────────────────────────────────────────────────────┐
│                     CHECKPOINT CREATION                      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. Identify Aether-managed files (allowlist)               │
│     - .aether/ directory (excluding data/)                  │
│     - .claude/commands/ant/                                 │
│     - .opencode/commands/ant/                               │
│     - .opencode/agents/                                     │
│     - bin/ (if exists)                                      │
│                                                              │
│  2. Check for uncommitted changes in managed files          │
│     ├─ No changes: Record commit hash only                  │
│     └─ Changes exist: Create checkpoint                     │
│                                                              │
│  3. Create checkpoint (NEVER use git stash)                 │
│     ├─ Copy files to .aether/checkpoints/<id>/              │
│     ├─ Store metadata (commit hash, timestamp, files)       │
│     └─ Verify checkpoint integrity                          │
│                                                              │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                     ROLLBACK EXECUTION                       │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. Verify checkpoint exists and is valid                   │
│                                                              │
│  2. Check current state                                     │
│     ├─ If clean: Restore from checkpoint                    │
│     └─ If dirty: Warn user, require --force                 │
│                                                              │
│  3. Restore files (copy from checkpoint, not stash pop)     │
│                                                              │
│  4. Verify restoration                                      │
│                                                              │
│  5. Clean up checkpoint (or keep for audit trail)           │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Integration Points

| Integration | How | Component |
|-------------|-----|-----------|
| File allowlist | Extend SYSTEM_FILES in cli.js | `bin/cli.js` |
| Checkpoint storage | Use .aether/checkpoints/ | New directory |
| Metadata format | JSON with file hashes | New format |
| Exclusion | Never touch .aether/data/ | Hardcoded exclusion |

### Checkpoint Metadata Format

```json
{
  "checkpoint_id": "chk_1707912000_a1b2",
  "created_at": "2026-02-14T12:00:00Z",
  "commit_hash": "abc123def456",
  "trigger": "autofix",
  "files": [
    {
      "path": ".aether/aether-utils.sh",
      "hash": "sha256:def...",
      "size": 12345
    }
  ],
  "excluded_files": [
    "src/user-code.js",
    "README.md"
  ],
  "integrity": {
    "manifest_hash": "sha256:ghi...",
    "file_count": 5
  }
}
```

## Fix 4: Testing Strategy for File Sync Operations

### Problem Analysis

Current E2E tests exist but need:
1. Unit tests for individual sync functions
2. Mock filesystem for isolated testing
3. Hash comparison verification
4. Rollback scenario testing

### Recommended Architecture

```
Test Architecture:
┌─────────────────────────────────────────────────────────────┐
│                     TEST LAYERS                              │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Unit Tests (tests/unit/sync-*.test.js)                     │
│  ├─ hashFileSync() - various file types                     │
│  ├─ compareVersions() - semver edge cases                   │
│  ├─ validateManifest() - schema validation                  │
│  ├─ shouldSyncFile() - allowlist logic                      │
│  └─ createCheckpoint() / restoreCheckpoint()                │
│                                                              │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Integration Tests (tests/integration/)                     │
│  ├─ File sync with temp directories                         │
│  ├─ Registry updates                                        │
│  ├─ Version comparison scenarios                            │
│  └─ Checkpoint create/restore cycles                        │
│                                                              │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  E2E Tests (tests/e2e/) - EXISTING                          │
│  ├─ test-update.sh - Single repo update                     │
│  ├─ test-update-all.sh - Multi-repo update                  │
│  ├─ test-install.sh - Fresh install                         │
│  └─ run-all.sh - Test suite orchestration                   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Integration Points

| Integration | How | Component |
|-------------|-----|-----------|
| Mock filesystem | Use `mock-fs` or `memfs` | Test dependencies |
| Hash verification | Reuse `hashFileSync` | `bin/cli.js` |
| Temp directories | Use `tmp` package or `mktemp` | Test setup |
| Assertion helpers | Extend existing test-helpers | `tests/bash/test-helpers.sh` |

### Test Data Patterns

```javascript
// tests/unit/sync-hash.test.js
const { hashFileSync } = require('../../bin/cli.js');
const fs = require('fs');
const path = require('path');
const os = require('os');

describe('hashFileSync', () => {
  let tmpDir;

  beforeEach(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'aether-test-'));
  });

  afterEach(() => {
    fs.rmSync(tmpDir, { recursive: true });
  });

  test('hashes text file correctly', () => {
    const file = path.join(tmpDir, 'test.txt');
    fs.writeFileSync(file, 'hello world');

    const hash = hashFileSync(file);

    expect(hash).toMatch(/^sha256:[a-f0-9]{64}$/);
  });

  test('returns null for non-existent file', () => {
    const hash = hashFileSync('/does/not/exist');
    expect(hash).toBeNull();
  });

  test('different content produces different hash', () => {
    const file1 = path.join(tmpDir, 'a.txt');
    const file2 = path.join(tmpDir, 'b.txt');
    fs.writeFileSync(file1, 'content A');
    fs.writeFileSync(file2, 'content B');

    expect(hashFileSync(file1)).not.toBe(hashFileSync(file2));
  });
});
```

## Component Boundaries

### What Talks to What

```
┌─────────────────────────────────────────────────────────────┐
│                    COMMUNICATION MATRIX                      │
├──────────────────┬──────────────────┬───────────────────────┤
│ Source           │ Target           │ Method                │
├──────────────────┼──────────────────┼───────────────────────┤
│ Queen (agents)   │ Utility Layer    │ bash aether-utils.sh  │
│ Queen (agents)   │ State            │ validate-state cmd    │
│ CLI (cli.js)     │ Utility Layer    │ registry-add cmd      │
│ CLI (cli.js)     │ Hub              │ filesystem (copy)     │
│ CLI (cli.js)     │ Registry         │ JSON read/write       │
│ Utility Layer    │ State            │ file-lock.sh          │
│ Utility Layer    │ State            │ atomic-write.sh       │
│ Tests            │ CLI functions    │ require() exports     │
│ Tests            │ Utility Layer    │ bash execution        │
└──────────────────┴──────────────────┴───────────────────────┘
```

### Data Flow

```
State Update Flow:
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Worker     │────▶│  aether-     │────▶│  file-       │
│  Completion  │     │  utils.sh    │     │  lock.sh     │
└──────────────┘     └──────────────┘     └──────┬───────┘
                                                 │
                                                 ▼
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   COLONY_    │◀────│ atomic-write │◀────│   Validate   │
│   STATE.json │     │     .sh      │     │   State      │
└──────────────┘     └──────────────┘     └──────────────┘

Update Flow:
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│    User      │────▶│   cli.js     │────▶│  Hub Files   │
│   Command    │     │  (update)    │     │  (~/.aether) │
└──────────────┘     └──────┬───────┘     └──────────────┘
                            │
              ┌─────────────┼─────────────┐
              ▼             ▼             ▼
        ┌──────────┐  ┌──────────┐  ┌──────────┐
        │  Local   │  │ Registry │  │ Version  │
        │  .aether │  │  Update  │  │  Check   │
        └──────────┘  └──────────┘  └──────────┘
```

## Suggested Build Order

Based on dependencies between fixes:

### Phase 1: Foundation (Week 1)
1. **Safe Checkpoint System** (Fix 3)
   - No dependencies on other fixes
   - Provides rollback capability for other fixes
   - Changes: `aether-utils.sh` autofix-checkpoint/rollback commands

2. **Testing Infrastructure** (Fix 4 - partial)
   - Unit test framework for sync operations
   - Mock filesystem setup
   - Hash verification tests

### Phase 2: Core Fixes (Week 2)
3. **Update System Rollback** (Fix 2)
   - Depends on: Safe checkpoint system
   - Changes: `cli.js` update command, manifest format
   - Tests: E2E tests for rollback scenarios

4. **Phase Advancement Idempotency** (Fix 1)
   - Depends on: Testing infrastructure
   - Changes: State advancement logic, event format
   - Tests: Unit tests for idempotency keys

### Phase 3: Integration (Week 3)
5. **Complete Testing** (Fix 4 - completion)
   - Integration tests for all fixes
   - Full E2E test suite
   - Performance benchmarks

### Dependency Graph

```
                    ┌─────────────────────┐
                    │   Testing Infra     │
                    │     (Fix 4a)        │
                    └──────────┬──────────┘
                               │
         ┌─────────────────────┼─────────────────────┐
         │                     │                     │
         ▼                     ▼                     ▼
┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐
│ Safe Checkpoint │   │ Phase Advance   │   │ Update System   │
│   (Fix 3)       │◀──│   (Fix 1)       │   │  (Fix 2)        │
└────────┬────────┘   └─────────────────┘   └────────┬────────┘
         │                                           │
         │              ┌─────────────────┐          │
         └─────────────▶│  Complete Tests │◀─────────┘
                        │    (Fix 4b)     │
                        └─────────────────┘
```

## Anti-Patterns to Avoid

### Anti-Pattern 1: Direct State Mutation
**What people do:** Read COLONY_STATE.json, modify in memory, write directly
**Why it's wrong:** Race conditions, partial writes, no validation
**Do this instead:** Always use `atomic_write` with validation

### Anti-Pattern 2: Git Stash for Checkpoints
**What people do:** Use `git stash` to save Aether state
**Why it's wrong:** Stashes user work, pollutes git history
**Do this instead:** Copy files to `.aether/checkpoints/` directory

### Anti-Pattern 3: Registry Before Files
**What people do:** Update registry.json before confirming file sync success
**Why it's wrong:** Registry out of sync with actual files
**Do this instead:** Two-phase commit - files first, registry last

### Anti-Pattern 4: Missing Idempotency
**What people do:** Phase advancement without checking if already done
**Why it's wrong:** Double-advancement, phase loops
**Do this instead:** Generate idempotency key, check events before advancing

## Scalability Considerations

| Concern | At 1 repo | At 10 repos | At 100 repos |
|---------|-----------|-------------|--------------|
| Update --all | Sequential OK | Parallel with limit | Batch with progress |
| Registry size | JSON file OK | JSON file OK | Consider SQLite |
| Checkpoint storage | Keep all | Rotate after 30 days | Archive to compressed store |
| Spawn tree | In-memory OK | In-memory OK | Paginated queries |

## Sources

- `.aether/aether-utils.sh` - State management commands
- `.aether/utils/state-loader.sh` - State loading patterns
- `.aether/utils/file-lock.sh` - Lock acquisition patterns
- `.aether/utils/atomic-write.sh` - Safe write patterns
- `bin/cli.js` - Update system implementation
- `tests/e2e/test-update.sh` - E2E test patterns
- `tests/e2e/test-update-all.sh` - Multi-repo test patterns
- `.opencode/agents/aether-queen.md` - Queen orchestration patterns

---
*Architecture research for: Aether Colony v1.1 Bug Fixes*
*Researched: 2026-02-14*
