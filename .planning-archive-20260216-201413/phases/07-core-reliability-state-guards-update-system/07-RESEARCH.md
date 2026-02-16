# Phase 7: Core Reliability — State Guards & Update System - Research

**Researched:** 2026-02-14
**Domain:** State machine guards, file locking, two-phase commit, automatic rollback
**Confidence:** HIGH

## Summary

This research covers the implementation of state guards to prevent phase advancement loops (Iron Law enforcement), reliable cross-repo synchronization with automatic rollback, and integration with the existing checkpoint system from Phase 6.

**Key Findings:**

1. **Existing file locking infrastructure** - `file-lock.sh` already implements PID-based locking with stale lock detection (lines 30-66), timeout handling, and cleanup traps. This should be extended for Node.js usage.

2. **State loading with lock acquisition** - `state-loader.sh` already demonstrates the pattern: acquire lock → validate → load → release on cleanup (lines 41-121). This pattern should be replicated for phase transitions.

3. **Checkpoint system from Phase 6** - The `aether checkpoint` command with create/list/restore/verify subcommands is fully implemented in `bin/cli.js` (lines 1160-1348). It uses git stash for backup and SHA-256 hash verification for integrity.

4. **COLONY_STATE.json structure** - Already contains `events` array for audit trail (verified in colony-state.test.js lines 274-293). Events have required fields: timestamp, type, worker, details.

5. **Update flow exists but lacks rollback** - `updateRepo()` function (cli.js lines 783-885) handles dirty file detection and git stash, but has no automatic rollback on failure.

**Primary recommendation:** Build on existing file-lock.sh and checkpoint patterns to create a state guard system that enforces the Iron Law (verification evidence required) and implements two-phase commit for updates with automatic rollback.

## Standard Stack

The established libraries/tools for this domain:

### Core (Already in Use)
| Library/Tool | Version | Purpose | Why Standard |
|--------------|---------|---------|--------------|
| `flock` (bash) | system | File locking | Native, battle-tested, handles stale locks |
| `git stash` | system | Backup/restore | Already used in update flow |
| `crypto.createHash` | Node built-in | SHA-256 hashing | Hardware-accelerated, standard |
| `proxyquire` | ^2.1.3 | Module mocking | Phase 6 testing infrastructure |
| `sinon` | ^19.0.0 | Stubbing/spying | Phase 6 testing infrastructure |

### Supporting
| Tool | Purpose | When to Use |
|------|---------|-------------|
| `fs.mkdtempSync` | Secure temp directories | Test isolation |
| `child_process.execSync` | Git operations | Already used throughout |

### Integration Points with Phase 6
| Phase 6 Component | Phase 7 Usage |
|-------------------|---------------|
| `file-lock.sh` | Extend for Node.js state guards |
| `checkpoint` command | Pre-update checkpoint creation |
| `CHECKPOINT_ALLOWLIST` | Verify only safe files modified |
| `hashFileSync()` | Integrity verification in two-phase commit |
| `syncDirWithCleanup()` | Update sync operations |

## Architecture Patterns

### Recommended Project Structure
```
.aether/
├── data/
│   ├── COLONY_STATE.json       # Phase state + events array
│   ├── flags.json              # Blockers/issues tracking
│   └── checkpoints/            # Checkpoint metadata (Phase 6)
├── locks/                      # File lock directory (Phase 6)
│   └── COLONY_STATE.json.lock  # State lock file
└── utils/
    ├── file-lock.sh            # Existing (Phase 6)
    ├── state-loader.sh         # Existing (Phase 6)
    └── state-guard.js          # NEW: Node.js state guards
```

### Pattern 1: State Guard with Iron Law Enforcement
**What:** Before any phase advancement, verify fresh verification evidence exists
**When to use:** All phase transitions (build, complete, resume)

```javascript
// Source: Pattern based on state-loader.sh lines 41-121
class StateGuard {
  constructor(stateFile) {
    this.stateFile = stateFile;
    this.lockFile = `${stateFile}.lock`;
    this.locked = false;
  }

  // STATE-01: Phase advancement requires fresh verification evidence
  async advancePhase(fromPhase, toPhase, evidence) {
    // Acquire lock (STATE-03)
    if (!await this.acquireLock()) {
      throw new StateGuardError('E_LOCK_TIMEOUT', 'Could not acquire state lock');
    }

    try {
      // Load and validate current state
      const state = await this.loadState();

      // STATE-02: Idempotency check
      if (state.current_phase > fromPhase) {
        return { status: 'already_complete', phase: state.current_phase };
      }

      // STATE-01: Iron Law - verify evidence exists
      if (!this.hasFreshEvidence(state, fromPhase, evidence)) {
        throw new StateGuardError('E_IRON_LAW_VIOLATION',
          `Phase ${fromPhase} advancement requires fresh verification evidence`,
          { required: ['test_results', 'verification_log', 'checkpoint_hash'] }
        );
      }

      // Perform transition
      const updated = this.transitionState(state, fromPhase, toPhase);

      // STATE-04: Add audit trail event
      updated.events.push({
        timestamp: new Date().toISOString(),
        type: 'phase_transition',
        worker: process.env.WORKER_NAME || 'unknown',
        details: { from: fromPhase, to: toPhase, evidence_id: evidence.id }
      });

      await this.saveState(updated);
      return { status: 'transitioned', from: fromPhase, to: toPhase };

    } finally {
      this.releaseLock();
    }
  }

  hasFreshEvidence(state, phase, evidence) {
    // Evidence must be from current session, not inherited
    const phaseLearning = state.memory.phase_learnings.find(
      pl => pl.phase === phase && !pl.source?.includes('inherited')
    );
    return phaseLearning && evidence && evidence.timestamp > state.initialized_at;
  }
}
```

### Pattern 2: Two-Phase Commit for Updates
**What:** Backup → Sync → Verify → Update Version with rollback capability
**When to use:** `aether update` command (UPDATE-01 through UPDATE-05)

```javascript
// Source: Pattern extending updateRepo() in cli.js lines 783-885
async function updateWithTwoPhaseCommit(repoPath, sourceVersion, opts = {}) {
  const dryRun = opts.dryRun || false;
  const transaction = new UpdateTransaction(repoPath);

  try {
    // Phase 1: Prepare
    // UPDATE-01: Create checkpoint before file sync
    const checkpoint = await transaction.createCheckpoint();

    // Phase 2: Sync
    const syncResult = await transaction.syncFiles(sourceVersion, dryRun);

    // Phase 3: Verify
    if (!dryRun) {
      const verification = await transaction.verifyIntegrity();
      if (!verification.valid) {
        // UPDATE-03: Automatic rollback on sync failure
        await transaction.rollback(checkpoint);
        throw new UpdateError('E_UPDATE_FAILED', 'Verification failed, rolled back', {
          checkpoint_id: checkpoint.id,
          // UPDATE-04: Stash recovery commands displayed prominently
          recovery_commands: [
            `cd ${repoPath}`,
            `git stash pop`  // If stash was created
          ]
        });
      }
    }

    // Phase 4: Commit
    if (!dryRun) {
      await transaction.updateVersion(sourceVersion);
    }

    return {
      status: dryRun ? 'dry-run' : 'updated',
      checkpoint_id: checkpoint.id,
      files_synced: syncResult.copied,
      files_removed: syncResult.removed
    };

  } catch (error) {
    // UPDATE-03: Automatic rollback on any failure
    await transaction.rollback();
    throw error;
  }
}
```

### Pattern 3: File Lock with Stale Detection
**What:** PID-based locking with automatic cleanup of stale locks
**When to use:** Any state modification operation

```javascript
// Source: file-lock.sh lines 30-66 adapted for Node.js
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

class FileLock {
  constructor(lockDir = '.aether/locks') {
    this.lockDir = lockDir;
    this.lockTimeout = 300000; // 5 minutes
    this.retryInterval = 500;  // 500ms
    this.maxRetries = 100;     // 50 seconds max wait
  }

  async acquire(filePath) {
    const lockFile = path.join(this.lockDir, `${path.basename(filePath)}.lock`);
    const pidFile = `${lockFile}.pid`;

    // Check for stale lock
    if (fs.existsSync(lockFile)) {
      const lockPid = this.readPidFile(pidFile);
      if (lockPid && !this.isProcessRunning(lockPid)) {
        console.log(`Lock stale (PID ${lockPid} not running), cleaning up...`);
        this.cleanupLock(lockFile, pidFile);
      }
    }

    // Try to acquire with retry
    for (let retry = 0; retry < this.maxRetries; retry++) {
      try {
        // Atomic lock creation using exclusive flag
        const fd = fs.openSync(lockFile, 'wx');
        fs.writeSync(fd, process.pid.toString());
        fs.closeSync(fd);

        // Write PID file
        fs.writeFileSync(pidFile, process.pid.toString());

        this.currentLock = lockFile;
        this.currentPidFile = pidFile;
        return true;
      } catch (err) {
        if (err.code !== 'EEXIST') throw err;

        // Wait before retry
        if (retry < this.maxRetries - 1) {
          await this.sleep(this.retryInterval);
        }
      }
    }

    return false;
  }

  release() {
    if (this.currentLock) {
      this.cleanupLock(this.currentLock, this.currentPidFile);
      this.currentLock = null;
      this.currentPidFile = null;
    }
  }

  isProcessRunning(pid) {
    try {
      process.kill(parseInt(pid), 0);
      return true;
    } catch {
      return false;
    }
  }

  cleanupLock(lockFile, pidFile) {
    try { fs.unlinkSync(lockFile); } catch {}
    try { fs.unlinkSync(pidFile); } catch {}
  }

  sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}
```

### Pattern 4: Audit Trail Event Sourcing
**What:** Immutable event log for all phase transitions
**When to use:** Every state modification (STATE-04)

```javascript
// Source: COLONY_STATE.json structure (verified in colony-state.test.js)
const eventSchema = {
  timestamp: '2026-02-14T14:30:22Z',  // ISO 8601 required
  type: 'phase_transition',            // event type
  worker: 'Builder-42',                // ant/worker name
  details: {                           // event-specific data
    from: 6,
    to: 7,
    evidence_id: 'ev_20260214_143022',
    checkpoint_id: 'chk_20260214_143015'
  }
};

// Event types for phase transitions
const EventTypes = {
  PHASE_TRANSITION: 'phase_transition',
  PHASE_BUILD_STARTED: 'phase_build_started',
  PHASE_BUILD_COMPLETED: 'phase_build_completed',
  PHASE_ROLLED_BACK: 'phase_rolled_back',
  CHECKPOINT_CREATED: 'checkpoint_created',
  CHECKPOINT_RESTORED: 'checkpoint_restored',
  UPDATE_STARTED: 'update_started',
  UPDATE_COMPLETED: 'update_completed',
  UPDATE_FAILED: 'update_failed'
};
```

### Anti-Patterns to Avoid

**Anti-Pattern 1: Lock Without Timeout**
- **Why it's bad:** Process crash leaves permanent lock, blocking all operations
- **What to do instead:** Always implement stale lock detection (PID checking) and timeout

**Anti-Pattern 2: State Modification Without Validation**
- **Why it's bad:** Corrupted state files, invalid transitions
- **What to do instead:** Validate state schema before every write (see validate-state in aether-utils.sh lines 103-146)

**Anti-Pattern 3: Silent Rollback Failures**
- **Why it's bad:** User doesn't know system is in inconsistent state
- **What to do instead:** Display prominent recovery commands on any failure (UPDATE-04)

**Anti-Pattern 4: Nested Locks**
- **Why it's bad:** Deadlocks when operations call each other
- **What to do instead:** Single lock per resource, acquire at entry point only

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| File locking | Custom lock files without PID tracking | `file-lock.sh` pattern with stale detection | Already tested, handles crashes |
| State validation | Manual JSON checks | `validate-state colony` command | Comprehensive schema validation |
| Checkpointing | Custom tar archives | `aether checkpoint` + git stash | Deduplication, integrity verification |
| Hash verification | Custom hash comparison | `hashFileSync()` from cli.js | SHA-256, consistent format |
| Atomic writes | Direct file writes | `atomic_write()` from atomic-write.sh | Temp file + mv pattern |
| Error recovery | Silent catch blocks | Structured error with recovery commands | User knows how to recover |

**Key insight:** The Phase 6 checkpoint system already solves the hard problems (allowlist safety, hash verification, git integration). Phase 7 should orchestrate these existing tools, not rebuild them.

## Common Pitfalls

### Pitfall 1: Race Condition in Phase Advancement
**What goes wrong:** Two workers simultaneously advance to same phase, both pass idempotency check, both write state
**Why it happens:** Idempotency check and state write are not atomic
**How to avoid:** Always acquire lock BEFORE reading state, hold until after write
**Warning signs:** Duplicate events in COLONY_STATE.json with same timestamp

### Pitfall 2: Zombie Locks After Crash
**What goes wrong:** Lock file remains after process crash, blocking all future operations
**Why it happens:** No stale lock detection or cleanup on startup
**How to avoid:** Check PID in lock file on every acquire attempt, clean up if process dead
**Warning signs:** "Failed to acquire lock" errors when no processes running

### Pitfall 3: Partial Update Corruption
**What goes wrong:** Update fails mid-sync, leaving some files updated and others not
**Why it happens:** No checkpoint before sync, no rollback on failure
**How to avoid:** Two-phase commit pattern: checkpoint → sync → verify → commit
**Warning signs:** Version mismatch between files, missing files, hash mismatches

### Pitfall 4: Lost Evidence in State Transitions
**What goes wrong:** Phase advances but verification evidence is stale or missing
**Why it happens:** Evidence check doesn't verify freshness or completeness
**How to avoid:** Iron Law enforcement - require evidence from current session with all required fields
**Warning signs:** Phase shows COMPLETED but no test results, no checkpoint hash

### Pitfall 5: Unclear Recovery Path on Failure
**What goes wrong:** Update fails with generic error, user doesn't know how to recover
**Why it happens:** Error messages don't include specific recovery commands
**How to avoid:** UPDATE-04 - always display stash recovery commands prominently
**Warning signs:** Users asking "what do I do now?" after errors

## Code Examples

### Example 1: Iron Law Enforcement
```javascript
// Source: Pattern for STATE-01 requirement
function enforceIronLaw(state, phase, evidence) {
  // Check for fresh verification evidence
  const requiredEvidence = ['checkpoint_hash', 'test_results'];
  const missing = requiredEvidence.filter(field => !evidence[field]);

  if (missing.length > 0) {
    throw new ValidationError(
      `Iron Law violation: Phase ${phase} advancement missing required evidence`,
      { missing, provided: Object.keys(evidence) },
      `Provide verification evidence: ${missing.join(', ')}`
    );
  }

  // Verify evidence is from current session
  const evidenceAge = Date.now() - new Date(evidence.timestamp).getTime();
  const sessionAge = Date.now() - new Date(state.initialized_at).getTime();

  if (evidenceAge > sessionAge) {
    throw new ValidationError(
      `Iron Law violation: Evidence is stale (from previous session)`,
      { evidence_timestamp: evidence.timestamp, session_start: state.initialized_at }
    );
  }

  return true;
}
```

### Example 2: Two-Phase Commit Implementation
```javascript
// Source: Pattern for UPDATE-01 through UPDATE-05
class UpdateTransaction {
  constructor(repoPath) {
    this.repoPath = repoPath;
    this.checkpoint = null;
    this.state = 'pending'; // pending, prepared, committed, rolled_back
  }

  async execute(sourceVersion) {
    try {
      // Phase 1: Prepare
      this.state = 'preparing';
      this.checkpoint = await this.createCheckpoint();

      // Phase 2: Sync
      this.state = 'syncing';
      const syncResult = await this.syncFiles(sourceVersion);

      // Phase 3: Verify
      this.state = 'verifying';
      const verification = await this.verifySync(syncResult);

      if (!verification.valid) {
        throw new Error(`Verification failed: ${verification.errors.join(', ')}`);
      }

      // Phase 4: Commit
      this.state = 'committing';
      await this.updateVersion(sourceVersion);
      this.state = 'committed';

      return { success: true, checkpoint: this.checkpoint };

    } catch (error) {
      // Automatic rollback
      this.state = 'rolling_back';
      await this.rollback();
      this.state = 'rolled_back';

      // Enhance error with recovery info
      error.recovery = this.getRecoveryCommands();
      throw error;
    }
  }

  getRecoveryCommands() {
    const commands = [];
    if (this.checkpoint?.stashRef) {
      commands.push(`cd ${this.repoPath} && git stash pop ${this.checkpoint.stashRef}`);
    }
    if (this.checkpoint?.id) {
      commands.push(`aether checkpoint restore ${this.checkpoint.id}`);
    }
    return commands;
  }
}
```

### Example 3: Test Pattern for State Guards
```javascript
// Source: Pattern based on state-loader.test.js
const test = require('ava');
const sinon = require('sinon');
const proxyquire = require('proxyquire');

test.beforeEach(t => {
  t.context.mockFs = {
    existsSync: sinon.stub(),
    readFileSync: sinon.stub(),
    writeFileSync: sinon.stub(),
    mkdirSync: sinon.stub(),
    openSync: sinon.stub(),
    closeSync: sinon.stub(),
    unlinkSync: sinon.stub()
  };

  t.context.StateGuard = proxyquire('../lib/state-guard', {
    fs: t.context.mockFs
  }).StateGuard;
});

test.afterEach(t => {
  sinon.restore();
});

test('StateGuard prevents advancement without evidence', async t => {
  const { mockFs, StateGuard } = t.context;

  // Setup valid state
  const state = {
    version: '3.0',
    current_phase: 5,
    initialized_at: '2026-02-14T10:00:00Z',
    memory: { phase_learnings: [] },
    events: []
  };

  mockFs.existsSync.returns(true);
  mockFs.readFileSync.returns(JSON.stringify(state));
  mockFs.openSync.returns(1); // Lock acquired

  const guard = new StateGuard('/test/COLONY_STATE.json');

  // Attempt advancement without evidence
  const error = await t.throwsAsync(
    guard.advancePhase(5, 6, null)
  );

  t.is(error.code, 'E_IRON_LAW_VIOLATION');
});

test('StateGuard enforces idempotency', async t => {
  const { mockFs, StateGuard } = t.context;

  // State already at phase 6
  const state = {
    current_phase: 6,
    initialized_at: '2026-02-14T10:00:00Z',
    memory: { phase_learnings: [] },
    events: []
  };

  mockFs.existsSync.returns(true);
  mockFs.readFileSync.returns(JSON.stringify(state));
  mockFs.openSync.returns(1);

  const guard = new StateGuard('/test/COLONY_STATE.json');

  // Attempt to rebuild phase 5
  const result = await guard.advancePhase(5, 6, { test: 'evidence' });

  t.is(result.status, 'already_complete');
});
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual phase advancement | State guards with Iron Law enforcement | Phase 7 | Cannot advance without verification |
| Direct file updates | Two-phase commit with checkpoint | Phase 7 | Automatic rollback on failure |
| Best-effort locking | PID-based locks with stale detection | Phase 6 | No zombie locks |
| Silent failures | Prominent recovery commands | Phase 7 | Users know how to recover |
| Ad-hoc state validation | `validate-state colony` command | Phase 6 | Consistent validation |

**Deprecated/outdated:**
- Direct COLONY_STATE.json modification without locking (use StateGuard)
- Update without checkpoint (use two-phase commit)
- Manual git stash for recovery (use `aether checkpoint`)

## Integration with Phase 6 Checkpoint System

### Checkpoint Creation Flow (UPDATE-01)
```
1. User runs: aether update
2. System calls: checkpoint create "Pre-update backup"
3. Checkpoint metadata saved to .aether/checkpoints/chk_YYYYMMDD_HHMMSS.json
4. Git stash created with allowlisted files
5. Update proceeds only after checkpoint verified
```

### Rollback Flow (UPDATE-03)
```
1. Update fails during sync or verification
2. System calls: checkpoint restore <checkpoint-id>
3. Git stash pop restores files
4. Version.json reverted
5. Error displayed with recovery commands
```

### Verification Evidence Flow (STATE-01)
```
1. Phase build completes
2. Tests run and generate results
3. Checkpoint created with hash verification
4. Evidence recorded: { checkpoint_id, test_results, timestamp }
5. Phase advancement checks for evidence
6. Event added to COLONY_STATE.json events array
```

## Open Questions

1. **Checkpoint retention during updates**
   - What we know: Checkpoints are created before updates
   - What's unclear: How many checkpoints to keep, cleanup policy
   - Recommendation: Keep last 10 checkpoints, auto-remove old ones after successful update

2. **Network failure handling during update**
   - What we know: Update syncs from hub at ~/.aether/
   - What's unclear: Behavior when hub is inaccessible mid-update
   - Recommendation: Verify hub accessibility before starting transaction

3. **Concurrent update and phase build**
   - What we know: File locking prevents concurrent state modification
   - What's unclear: Whether update should be blocked during active phase build
   - Recommendation: Check state.status === 'BUILDING' and reject update with clear message

## Sources

### Primary (HIGH confidence)
- `/Users/callumcowie/repos/Aether/.aether/utils/file-lock.sh` - Lock implementation with stale detection
- `/Users/callumcowie/repos/Aether/.aether/utils/state-loader.sh` - State loading with lock acquisition pattern
- `/Users/callumcowie/repos/Aether/bin/cli.js` - Checkpoint commands (lines 1160-1348), update flow (lines 783-885)
- `/Users/callumcowie/repos/Aether/tests/unit/state-loader.test.js` - Lock testing patterns
- `/Users/callumcowie/repos/Aether/tests/unit/colony-state.test.js` - Event validation patterns

### Secondary (MEDIUM confidence)
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` - validate-state implementation (lines 103-146)
- `/Users/callumcowie/repos/Aether/bin/lib/errors.js` - Error class hierarchy for structured errors

### Tertiary (LOW confidence)
- Two-phase commit patterns from distributed systems literature
- State machine guard patterns from functional programming

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All tools already in use in codebase
- Architecture: HIGH - Based on existing file-lock.sh and state-loader.sh patterns
- Pitfalls: MEDIUM - Inferred from common distributed systems issues

**Research date:** 2026-02-14
**Valid until:** 2026-03-14 (30 days for stable patterns)
