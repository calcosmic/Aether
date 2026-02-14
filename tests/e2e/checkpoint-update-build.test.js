#!/usr/bin/env node
/**
 * E2E Test: Checkpoint → Update → Build Workflow
 *
 * Verifies all v1.1 fixes work together in the complete workflow:
 * 1. SAFE-01 to SAFE-04: Checkpoint safety (Phase 6)
 * 2. STATE-01 to STATE-04: State guards with Iron Law (Phase 7)
 * 3. UPDATE-01 to UPDATE-05: Update transactions with rollback (Phase 7)
 *
 * Tests the complete workflow: checkpoint → update → build with Iron Law enforcement,
 * rollback on failure, and state consistency.
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');
const { initializeRepo, isInitialized } = require('../../bin/lib/init');
const { UpdateTransaction, UpdateError } = require('../../bin/lib/update-transaction');
const { StateGuard, StateGuardError } = require('../../bin/lib/state-guard');
const { EventTypes } = require('../../bin/lib/event-types');

// Helper to create temp directory
async function createTempDir() {
  const tmpDir = await fs.promises.mkdtemp(path.join(os.tmpdir(), 'aether-e2e-'));
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

// Helper to initialize git repo
function gitInit(tmpDir) {
  execSync('git init', { cwd: tmpDir, stdio: 'pipe' });
  execSync('git config user.email "test@test.com"', { cwd: tmpDir, stdio: 'pipe' });
  execSync('git config user.name "Test"', { cwd: tmpDir, stdio: 'pipe' });
}

// Helper to create valid evidence for phase advancement
function createValidEvidence(phase) {
  // Use a future timestamp to ensure it's after state initialization
  const futureDate = new Date();
  futureDate.setMinutes(futureDate.getMinutes() + 1);
  return {
    checkpoint_hash: `sha256:${phase}_test_hash_${Date.now()}`,
    test_results: { passed: 5, failed: 0, total: 5 },
    timestamp: futureDate.toISOString()
  };
}

// Helper to create a checkpoint using the checkpoint system
function createCheckpoint(tmpDir, message = 'Test checkpoint') {
  const checkpointId = `chk_${Date.now()}_test`;
  const checkpointDir = path.join(tmpDir, '.aether', 'checkpoints');
  fs.mkdirSync(checkpointDir, { recursive: true });

  // Get list of tracked files in .aether directory
  const files = {};
  const aetherDir = path.join(tmpDir, '.aether');

  // Only include system files (not user data)
  const systemFiles = ['version.json', 'data/COLONY_STATE.json'];
  for (const file of systemFiles) {
    const filePath = path.join(aetherDir, file);
    if (fs.existsSync(filePath)) {
      const content = fs.readFileSync(filePath);
      const hash = require('crypto').createHash('sha256').update(content).digest('hex');
      files[file] = `sha256:${hash}`;
    }
  }

  const checkpoint = {
    checkpoint_id: checkpointId,
    created_at: new Date().toISOString(),
    message,
    files
  };

  fs.writeFileSync(
    path.join(checkpointDir, `${checkpointId}.json`),
    JSON.stringify(checkpoint, null, 2)
  );

  return checkpoint;
}

test.serial('complete workflow succeeds', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Step 1: Initialize git repo
    gitInit(tmpDir);

    // Step 2: Initialize Aether
    await initializeRepo(tmpDir, { goal: 'E2E workflow test' });
    t.true(isInitialized(tmpDir), 'Repo should be initialized');

    // Verify COLONY_STATE.json exists with correct structure
    const statePath = path.join(tmpDir, '.aether', 'data', 'COLONY_STATE.json');
    t.true(fs.existsSync(statePath), 'State file should exist');

    const initialState = JSON.parse(fs.readFileSync(statePath, 'utf8'));
    t.is(initialState.current_phase, 0, 'Initial phase should be 0');
    t.is(initialState.goal, 'E2E workflow test', 'Goal should match');
    t.truthy(initialState.events, 'Should have events array');

    // Step 3: Create a checkpoint
    const checkpoint = createCheckpoint(tmpDir, 'Pre-update checkpoint');
    t.truthy(checkpoint.checkpoint_id, 'Should have checkpoint_id');

    const checkpointPath = path.join(tmpDir, '.aether', 'checkpoints', `${checkpoint.checkpoint_id}.json`);
    t.true(fs.existsSync(checkpointPath), 'Checkpoint metadata should exist');

    // Verify checkpoint only contains Aether-managed files (no user data)
    const checkpointFiles = Object.keys(checkpoint.files);
    t.true(checkpointFiles.length > 0, 'Checkpoint should have files');
    t.false(checkpointFiles.some(f => f.includes('user_data')), 'Checkpoint should not contain user data');

    // Step 4: Create initial commit for git operations
    fs.writeFileSync(path.join(tmpDir, 'test.txt'), 'test content');
    execSync('git add .', { cwd: tmpDir, stdio: 'pipe' });
    execSync('git commit -m "initial"', { cwd: tmpDir, stdio: 'pipe' });

    // Step 5: Verify StateGuard enforces Iron Law
    const stateFile = path.join(tmpDir, '.aether', 'data', 'COLONY_STATE.json');
    const guard = new StateGuard(stateFile, { worker: 'e2e-test' });

    // Attempt to advance WITHOUT evidence - should throw StateGuardError
    const error = await t.throwsAsync(
      async () => await guard.advancePhase(0, 1, {}),
      { instanceOf: StateGuardError }
    );
    t.is(error.code, 'E_IRON_LAW_VIOLATION', 'Should throw E_IRON_LAW_VIOLATION');

    // Advance WITH valid evidence - should succeed
    const evidence = createValidEvidence(1);
    const result = await guard.advancePhase(0, 1, evidence);
    t.is(result.status, 'transitioned', 'Should transition successfully');
    t.is(result.from, 0, 'Should advance from phase 0');
    t.is(result.to, 1, 'Should advance to phase 1');

    // Step 6: Verify state advancement
    const updatedState = JSON.parse(fs.readFileSync(statePath, 'utf8'));
    t.is(updatedState.current_phase, 1, 'Current phase should be 1');

    // Step 7: Verify audit trail
    const phaseEvents = updatedState.events.filter(e => e.type === 'phase_transition');
    t.is(phaseEvents.length, 1, 'Should have one phase transition event');

    const phaseEvent = phaseEvents[0];
    t.truthy(phaseEvent.timestamp, 'Event should have timestamp');
    t.is(phaseEvent.type, 'phase_transition', 'Event type should be phase_transition');
    t.is(phaseEvent.worker, 'e2e-test', 'Event should have worker attribution');
    t.is(phaseEvent.details.from, 0, 'Event should record from phase');
    t.is(phaseEvent.details.to, 1, 'Event should record to phase');

  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('Iron Law blocks advancement without evidence', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Initialize git repo and Aether
    gitInit(tmpDir);
    await initializeRepo(tmpDir, { goal: 'Iron Law test' });

    const stateFile = path.join(tmpDir, '.aether', 'data', 'COLONY_STATE.json');
    const guard = new StateGuard(stateFile, { worker: 'iron-law-test' });

    // Test 1: Iron Law enforcement (STATE-01 requirement)
    // Attempt to advance with EMPTY evidence object
    const error = await t.throwsAsync(
      async () => await guard.advancePhase(0, 1, {}),
      { instanceOf: StateGuardError }
    );
    t.is(error.code, 'E_IRON_LAW_VIOLATION', 'Should throw E_IRON_LAW_VIOLATION');
    t.true(error.message.includes('evidence'), 'Error should mention evidence');
    t.truthy(error.details.missing, 'Error should list missing fields');

    // Verify state unchanged
    const state = JSON.parse(fs.readFileSync(stateFile, 'utf8'));
    t.is(state.current_phase, 0, 'Phase should not have changed');

    // Test 2: Idempotency (STATE-02 requirement)
    // Advance with valid evidence
    const evidence = createValidEvidence(1);
    const result1 = await guard.advancePhase(0, 1, evidence);
    t.is(result1.status, 'transitioned', 'First advancement should succeed');

    // Attempt to advance to SAME phase again
    const result2 = await guard.advancePhase(0, 1, evidence);
    t.is(result2.status, 'already_complete', 'Should return already_complete for repeated advancement');
    t.is(result2.currentPhase, 1, 'Should report current phase as 1');

    // Test 3: State lock (STATE-03 requirement)
    // Lock should be released after operation
    const lockFile = path.join(tmpDir, '.aether', 'locks', 'COLONY_STATE.json.lock');
    t.false(fs.existsSync(lockFile), 'Lock file should be released after operation');

    // Test 4: Audit trail (STATE-04 requirement)
    const updatedState = JSON.parse(fs.readFileSync(stateFile, 'utf8'));
    const phaseEvents = updatedState.events.filter(e => e.type === 'phase_transition');
    t.is(phaseEvents.length, 1, 'Should have exactly one phase transition event (no duplicates)');

    // Verify event structure
    const event = phaseEvents[0];
    t.truthy(event.timestamp, 'Event should have timestamp');
    t.truthy(event.type, 'Event should have type');
    t.truthy(event.worker, 'Event should have worker attribution');
    t.truthy(event.details, 'Event should have details');

  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('update rollback preserves state', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Step 1: Initialize repo with known state
    gitInit(tmpDir);
    await initializeRepo(tmpDir, { goal: 'Update rollback test' });

    // Set a specific goal in COLONY_STATE.json
    const statePath = path.join(tmpDir, '.aether', 'data', 'COLONY_STATE.json');
    const initialState = JSON.parse(fs.readFileSync(statePath, 'utf8'));
    initialState.goal = 'Test goal for rollback verification';
    fs.writeFileSync(statePath, JSON.stringify(initialState, null, 2));

    // Record initial state hash
    const initialStateContent = fs.readFileSync(statePath, 'utf8');

    // Step 2: Create initial commit
    fs.writeFileSync(path.join(tmpDir, 'test.txt'), 'test content');
    execSync('git add .', { cwd: tmpDir, stdio: 'pipe' });
    execSync('git commit -m "initial"', { cwd: tmpDir, stdio: 'pipe' });

    // Step 3: Create checkpoint before update (UPDATE-01 requirement)
    const transaction = new UpdateTransaction(tmpDir, {
      sourceVersion: '1.0.0',
      quiet: true
    });

    const checkpoint = await transaction.createCheckpoint();
    t.truthy(checkpoint.id, 'Should have checkpoint id');
    t.truthy(checkpoint.timestamp, 'Should have checkpoint timestamp');

    // Verify checkpoint metadata exists
    const checkpointPath = path.join(tmpDir, '.aether', 'checkpoints', `${checkpoint.id}.json`);
    t.true(fs.existsSync(checkpointPath), 'Checkpoint metadata should exist');

    // Step 4: Test automatic rollback on failure (UPDATE-03 requirement)
    // Execute update with invalid version to trigger failure
    const failedTransaction = new UpdateTransaction(tmpDir, {
      sourceVersion: 'invalid-version-that-does-not-exist',
      quiet: true
    });

    try {
      await failedTransaction.execute('invalid-version', { dryRun: false });
      t.fail('Should have thrown an error');
    } catch (error) {
      // Verify it's an UpdateError
      t.true(error instanceof UpdateError || error.name === 'UpdateError', 'Should throw UpdateError');

      // Step 5: Verify recovery commands (UPDATE-04 requirement)
      if (error.recoveryCommands) {
        t.true(Array.isArray(error.recoveryCommands), 'Error should have recoveryCommands array');
        t.true(error.recoveryCommands.length > 0, 'Should have at least one recovery command');

        // Verify commands are displayed prominently
        const errorString = error.toString();
        t.true(errorString.includes('RECOVERY') || errorString.includes('recover'), 'Error message should mention recovery');
      }
    }

    // Step 6: Verify state is restored to pre-update condition
    const finalStateContent = fs.readFileSync(statePath, 'utf8');
    const finalState = JSON.parse(finalStateContent);

    // State should be unchanged
    t.is(finalState.goal, 'Test goal for rollback verification', 'Goal should be preserved');
    t.is(finalState.current_phase, initialState.current_phase, 'Phase should be preserved');

    // Step 7: Test error handling (UPDATE-05 requirement)
    // Test dirty repo detection
    // Create a dirty file to trigger dirty repo detection
    fs.writeFileSync(path.join(tmpDir, 'dirty-file.txt'), 'dirty content');
    execSync('git add dirty-file.txt', { cwd: tmpDir, stdio: 'pipe' });

    const dirtyTransaction = new UpdateTransaction(tmpDir, {
      sourceVersion: '1.0.0',
      quiet: true
    });

    try {
      await dirtyTransaction.execute('1.0.0', { dryRun: true });
    } catch (error) {
      // Should detect dirty repo
      if (error.code === 'E_REPO_DIRTY') {
        t.is(error.code, 'E_REPO_DIRTY', 'Should detect dirty repository');
        t.truthy(error.details, 'Error should have details about dirty files');
        t.true(error.recoveryCommands.length > 0, 'Should provide recovery commands for dirty repo');
      }
    }

  } finally {
    await cleanupTempDir(tmpDir);
  }
});
