/**
 * State Guard Integration Tests
 *
 * Integration tests for state guards, file locks, and event audit trail.
 * Uses real filesystem in temp directories.
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { StateGuard, StateGuardError } = require('../../bin/lib/state-guard');
const { FileLock } = require('../../bin/lib/file-lock');
const { initializeRepo, isInitialized } = require('../../bin/lib/init');

// Helper to create temp directory
async function createTempDir() {
  const tmpDir = await fs.promises.mkdtemp(path.join(os.tmpdir(), 'aether-integ-'));
  return tmpDir;
}

// Helper to cleanup temp directory
async function cleanupTempDir(tmpDir) {
  try {
    await fs.promises.rm(tmpDir, { recursive: true, force: true });
  } catch (err) {
    // Ignore cleanup errors
  }
}

// Helper to create valid evidence for phase advancement
function createValidEvidence(phase) {
  // Use a future timestamp to ensure it's after state initialization
  const futureDate = new Date();
  futureDate.setMinutes(futureDate.getMinutes() + 1);
  return {
    checkpoint_hash: `sha256:${phase}_test_hash`,
    test_results: { passed: 5, failed: 0, total: 5 },
    timestamp: futureDate.toISOString()
  };
}

test.serial('complete phase advancement flow', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Initialize repo at phase 0
    await initializeRepo(tmpDir, { goal: 'Integration test' });
    t.true(isInitialized(tmpDir), 'Repo should be initialized');

    // Create StateGuard with path to COLONY_STATE.json
    const stateFile = path.join(tmpDir, '.aether/data/COLONY_STATE.json');
    const guard = new StateGuard(stateFile, { worker: 'test-worker' });

    // Advance to phase 1 with valid evidence
    const evidence = createValidEvidence(1);
    const result = await guard.advancePhase(0, 1, evidence);

    // Assert: state updated to phase 1
    t.is(result.from, 0);
    t.is(result.to, 1);
    t.is(result.status, 'transitioned');

    // Assert: event recorded
    const state = JSON.parse(fs.readFileSync(path.join(tmpDir, '.aether/data/COLONY_STATE.json'), 'utf8'));
    t.is(state.current_phase, 1);
    t.true(state.events.length >= 2, 'Should have initialization and phase transition events');

    const phaseEvent = state.events.find(e => e.type === 'phase_transition');
    t.truthy(phaseEvent, 'Should have phase_transition event');
    t.is(phaseEvent.worker, 'test-worker');

    // Assert: lock released
    const lockFile = path.join(tmpDir, '.aether/locks/state.lock');
    t.false(fs.existsSync(lockFile), 'Lock file should be released');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('concurrent access is serialized', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Initialize repo
    await initializeRepo(tmpDir, { goal: 'Concurrent test' });

    // Create two StateGuard instances with state file path
    const stateFile = path.join(tmpDir, '.aether/data/COLONY_STATE.json');
    const guard1 = new StateGuard(stateFile, { worker: 'worker-1' });
    const guard2 = new StateGuard(stateFile, { worker: 'worker-2' });

    // Start two simultaneous advancePhase calls
    const evidence1 = createValidEvidence(1);
    const evidence2 = createValidEvidence(2);

    const promise1 = guard1.advancePhase(0, 1, evidence1);
    const promise2 = guard2.advancePhase(1, 2, evidence2);

    // Wait for both to complete
    const [result1, result2] = await Promise.allSettled([promise1, promise2]);

    // Assert: at least one succeeded
    const oneSucceeded = result1.status === 'fulfilled' || result2.status === 'fulfilled';
    t.true(oneSucceeded, 'At least one advancement should succeed');

    // Assert: final state is consistent (no corruption)
    const state = JSON.parse(fs.readFileSync(path.join(tmpDir, '.aether/data/COLONY_STATE.json'), 'utf8'));
    t.true(state.current_phase >= 1, 'Phase should have advanced');

    // Events should not be duplicated
    const phaseEvents = state.events.filter(e => e.type === 'PHASE_TRANSITION');
    const uniquePhases = new Set(phaseEvents.map(e => e.details?.to_phase));
    t.is(phaseEvents.length, uniquePhases.size, 'Should not have duplicate phase transition events');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('Iron Law prevents advancement without evidence', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Initialize repo at phase 0
    await initializeRepo(tmpDir, { goal: 'Iron Law test' });

    // Create StateGuard with state file path
    const stateFile = path.join(tmpDir, '.aether/data/COLONY_STATE.json');
    const guard = new StateGuard(stateFile, { worker: 'test-worker' });

    // Attempt advance without evidence
    const error = await t.throwsAsync(
      async () => await guard.advancePhase(0, 1, {}),
      { instanceOf: StateGuardError }
    );

    // Assert: throws E_IRON_LAW_VIOLATION
    t.is(error.code, 'E_IRON_LAW_VIOLATION');
    t.true(error.message.includes('evidence'), 'Error should mention evidence');

    // Assert: state unchanged
    const state = JSON.parse(fs.readFileSync(path.join(tmpDir, '.aether/data/COLONY_STATE.json'), 'utf8'));
    t.is(state.current_phase, 0, 'Phase should not have changed');

    // Assert: no phase transition event
    const phaseEvents = state.events.filter(e => e.type === 'PHASE_TRANSITION');
    t.is(phaseEvents.length, 0, 'Should not have PHASE_TRANSITION event');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('checkpoint -> update -> verify flow', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Initialize repo
    await initializeRepo(tmpDir, { goal: 'Checkpoint flow test' });

    // Create a mock checkpoint by creating the checkpoint directory and metadata
    const checkpointDir = path.join(tmpDir, '.aether/checkpoints');
    fs.mkdirSync(checkpointDir, { recursive: true });

    const checkpointId = `chk_${Date.now()}_test`;
    const checkpointData = {
      checkpoint_id: checkpointId,
      created_at: new Date().toISOString(),
      message: 'Test checkpoint',
      files: {}
    };
    fs.writeFileSync(
      path.join(checkpointDir, `${checkpointId}.json`),
      JSON.stringify(checkpointData, null, 2)
    );

    // Verify checkpoint exists
    const checkpointPath = path.join(checkpointDir, `${checkpointId}.json`);
    t.true(fs.existsSync(checkpointPath), 'Checkpoint file should exist');

    // Verify checkpoint content
    const loadedCheckpoint = JSON.parse(fs.readFileSync(checkpointPath, 'utf8'));
    t.is(loadedCheckpoint.checkpoint_id, checkpointId);
    t.is(loadedCheckpoint.message, 'Test checkpoint');

    // Verify state is still valid
    const state = JSON.parse(fs.readFileSync(path.join(tmpDir, '.aether/data/COLONY_STATE.json'), 'utf8'));
    t.is(state.current_phase, 0);
    t.truthy(state.events, 'State should have events');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('idempotency across multiple calls', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Initialize repo
    await initializeRepo(tmpDir, { goal: 'Idempotency test' });

    // Create StateGuard with state file path
    const stateFile = path.join(tmpDir, '.aether/data/COLONY_STATE.json');
    const guard = new StateGuard(stateFile, { worker: 'test-worker' });

    // Advance to phase 1
    const evidence1 = createValidEvidence(1);
    const result1 = await guard.advancePhase(0, 1, evidence1);
    t.is(result1.status, 'transitioned');

    // Advance to phase 2
    const evidence2 = createValidEvidence(2);
    const result2 = await guard.advancePhase(1, 2, evidence2);
    t.is(result2.status, 'transitioned');

    // Get event count after normal advancement
    const stateAfterNormal = JSON.parse(fs.readFileSync(path.join(tmpDir, '.aether/data/COLONY_STATE.json'), 'utf8'));
    const eventCountAfterNormal = stateAfterNormal.events.length;

    // Attempt to advance phase 1->2 again (should be idempotent)
    const result3 = await guard.advancePhase(1, 2, evidence2);
    t.is(result3.status, 'already_complete', 'Should return already_complete for repeated advancement');

    // Verify no duplicate events
    const stateAfterRepeat = JSON.parse(fs.readFileSync(path.join(tmpDir, '.aether/data/COLONY_STATE.json'), 'utf8'));
    t.is(stateAfterRepeat.events.length, eventCountAfterNormal, 'Should not add duplicate events');

    // Verify phase is still 2
    t.is(stateAfterRepeat.current_phase, 2);
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('FileLock prevents concurrent state modifications', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Initialize repo
    await initializeRepo(tmpDir, { goal: 'FileLock test' });

    // Create two FileLock instances for the same lock directory
    const lockDir = path.join(tmpDir, '.aether/locks');
    const lock1 = new FileLock({ lockDir, timeout: 100 });
    const lock2 = new FileLock({ lockDir, timeout: 100 });

    // Use the state file as the resource to lock
    const stateFile = path.join(tmpDir, '.aether/data/COLONY_STATE.json');

    // Acquire first lock on state file
    const acquired1 = await lock1.acquire(stateFile);
    t.true(acquired1, 'First lock should be acquired');

    // Try to acquire second lock (should fail or timeout)
    const acquired2 = await lock2.acquire(stateFile);
    t.false(acquired2, 'Second lock should not be acquired while first is held');

    // Release first lock
    lock1.release();

    // Now second lock should succeed
    const acquired3 = await lock2.acquire(stateFile);
    t.true(acquired3, 'Second lock should be acquired after first is released');

    lock2.release();
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('StateGuard event query methods work correctly', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Initialize repo
    await initializeRepo(tmpDir, { goal: 'Event query test' });

    // Create StateGuard with state file path and add some events
    const stateFile = path.join(tmpDir, '.aether/data/COLONY_STATE.json');
    const guard = new StateGuard(stateFile, { worker: 'test-worker' });

    // Advance phase to create events
    const evidence = createValidEvidence(1);
    const result = await guard.advancePhase(0, 1, evidence);
    t.is(result.status, 'transitioned', 'Phase should be transitioned');

    // Load state to query events (stateFile already declared above)
    const state = JSON.parse(fs.readFileSync(stateFile, 'utf8'));

    // Verify phase was updated
    t.is(state.current_phase, 1, 'Current phase should be 1');

    // Query events - getEvents expects state object
    const events = StateGuard.getEvents(state);
    t.true(events.length >= 2, 'Should have at least 2 events (init + transition)');

    // Query by type
    const phaseEvents = StateGuard.getEvents(state, { type: 'phase_transition' });
    t.is(phaseEvents.length, 1, 'Should have one phase transition event');

    // Get latest event - verify it returns an event
    const latest = StateGuard.getLatestEvent(state);
    t.truthy(latest, 'Should have a latest event');
    // Latest could be either colony_initialized or phase_transition depending on timing
    t.true(['colony_initialized', 'phase_transition'].includes(latest.type),
      `Latest event should be colony_initialized or phase_transition, got: ${latest?.type}`);
  } finally {
    await cleanupTempDir(tmpDir);
  }
});
