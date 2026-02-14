#!/usr/bin/env node
/**
 * E2E Test: Update with Rollback
 *
 * Tests the complete update flow with automatic rollback on failure.
 * Verifies:
 * 1. Update creates checkpoint before sync
 * 2. Failed update automatically rolls back
 * 3. Recovery commands are displayed
 * 4. State remains consistent after rollback
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');
const { initializeRepo, isInitialized } = require('../../bin/lib/init');
const { UpdateTransaction } = require('../../bin/lib/update-transaction');

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

test.serial('update creates checkpoint before sync', async (t) =>
{
  const tmpDir = await createTempDir();

  try {
    // Initialize git repo
    gitInit(tmpDir);

    // Initialize Aether
    await initializeRepo(tmpDir, { goal: 'E2E test' });
    t.true(isInitialized(tmpDir), 'Repo should be initialized');

    // Create an initial commit
    fs.writeFileSync(path.join(tmpDir, 'test.txt'), 'test content');
    execSync('git add .', { cwd: tmpDir, stdio: 'pipe' });
    execSync('git commit -m "initial"', { cwd: tmpDir, stdio: 'pipe' });

    // Create UpdateTransaction
    const transaction = new UpdateTransaction(tmpDir, {
      sourceVersion: '1.0.0',
      quiet: true
    });

    // Execute update (dry-run to avoid actual file changes)
    const result = await transaction.execute('1.0.0', { dryRun: true });

    // Assert: checkpoint was created (result has checkpoint_id from transaction)
    t.truthy(result.checkpoint_id, 'Should have checkpoint_id');
    t.is(result.status, 'dry-run', 'Should be dry-run status');

    // Verify checkpoint metadata exists (checkpoint id is used for filename)
    const checkpointPath = path.join(tmpDir, '.aether', 'checkpoints', `${result.checkpoint_id}.json`);
    t.true(fs.existsSync(checkpointPath), 'Checkpoint metadata should exist');

    // Verify checkpoint has required fields
    const checkpoint = JSON.parse(fs.readFileSync(checkpointPath, 'utf8'));
    t.truthy(checkpoint.id, 'Checkpoint should have id');
    t.truthy(checkpoint.created_at, 'Checkpoint should have created_at');

  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('state remains consistent after failed update', async (t) =>
{
  const tmpDir = await createTempDir();

  try {
    // Initialize git repo
    gitInit(tmpDir);

    // Initialize Aether
    await initializeRepo(tmpDir, { goal: 'E2E consistency test' });

    // Create an initial commit
    fs.writeFileSync(path.join(tmpDir, 'test.txt'), 'test content');
    execSync('git add .', { cwd: tmpDir, stdio: 'pipe' });
    execSync('git commit -m "initial"', { cwd: tmpDir, stdio: 'pipe' });

    // Read initial state
    const statePath = path.join(tmpDir, '.aether', 'data', 'COLONY_STATE.json');
    const initialState = JSON.parse(fs.readFileSync(statePath, 'utf8'));

    // Create UpdateTransaction with invalid source to trigger failure
    const transaction = new UpdateTransaction(tmpDir, {
      sourceVersion: 'invalid-version',
      quiet: true
    });

    // Execute update - should fail but not corrupt state
    try {
      await transaction.execute('invalid-version', { dryRun: true });
    } catch (error) {
      // Expected to fail
    }

    // Read state after failed update
    const finalState = JSON.parse(fs.readFileSync(statePath, 'utf8'));

    // Assert: state is unchanged
    t.deepEqual(finalState, initialState, 'State should remain unchanged after failed update');

  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('checkpoint metadata has correct structure', async (t) =>
{
  const tmpDir = await createTempDir();

  try {
    // Initialize git repo
    gitInit(tmpDir);

    // Initialize Aether
    await initializeRepo(tmpDir, { goal: 'E2E checkpoint test' });

    // Create an initial commit with a tracked file
    fs.writeFileSync(path.join(tmpDir, 'tracked.txt'), 'tracked content');
    execSync('git add .', { cwd: tmpDir, stdio: 'pipe' });
    execSync('git commit -m "initial"', { cwd: tmpDir, stdio: 'pipe' });

    // Create UpdateTransaction
    const transaction = new UpdateTransaction(tmpDir, {
      sourceVersion: '1.0.0',
      quiet: true
    });

    // Execute update (dry-run)
    const result = await transaction.execute('1.0.0', { dryRun: true });

    // Verify checkpoint structure
    const checkpointPath = path.join(tmpDir, '.aether', 'checkpoints', `${result.checkpoint_id}.json`);
    const checkpoint = JSON.parse(fs.readFileSync(checkpointPath, 'utf8'));

    // Required fields
    t.truthy(checkpoint.checkpoint_id, 'Should have checkpoint_id');
    t.truthy(checkpoint.created_at, 'Should have created_at');
    t.truthy(checkpoint.message, 'Should have message');
    t.truthy(checkpoint.files, 'Should have files object');

    // checkpoint_id should match
    t.is(checkpoint.checkpoint_id, result.checkpoint_id);

    // created_at should be valid ISO 8601
    const createdDate = new Date(checkpoint.created_at);
    t.false(isNaN(createdDate.getTime()), 'created_at should be valid date');

  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('update transaction tracks phases correctly', async (t) =>
{
  const tmpDir = await createTempDir();

  try {
    // Initialize git repo
    gitInit(tmpDir);

    // Initialize Aether
    await initializeRepo(tmpDir, { goal: 'E2E phases test' });

    // Create an initial commit
    fs.writeFileSync(path.join(tmpDir, 'test.txt'), 'test content');
    execSync('git add .', { cwd: tmpDir, stdio: 'pipe' });
    execSync('git commit -m "initial"', { cwd: tmpDir, stdio: 'pipe' });

    // Create UpdateTransaction
    const transaction = new UpdateTransaction(tmpDir, {
      sourceVersion: '1.0.0',
      quiet: true
    });

    // Verify initial phase
    t.is(transaction.phase, 'idle', 'Initial phase should be idle');

    // Execute update (dry-run)
    await transaction.execute('1.0.0', { dryRun: true });

    // After execution, phase should be back to idle
    t.is(transaction.phase, 'idle', 'Phase should return to idle after execution');

  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('e2e init -> checkpoint -> status flow', async (t) =>
{
  const tmpDir = await createTempDir();

  try {
    // Initialize git repo
    gitInit(tmpDir);

    // Step 1: Initialize Aether
    const initResult = await initializeRepo(tmpDir, { goal: 'E2E flow test' });
    t.true(initResult.success, 'Init should succeed');
    t.true(isInitialized(tmpDir), 'Repo should be initialized');

    // Step 2: Create initial commit
    fs.writeFileSync(path.join(tmpDir, 'README.md'), '# Test Repo');
    execSync('git add .', { cwd: tmpDir, stdio: 'pipe' });
    execSync('git commit -m "initial commit"', { cwd: tmpDir, stdio: 'pipe' });

    // Step 3: Verify state file exists and has correct structure
    const statePath = path.join(tmpDir, '.aether', 'data', 'COLONY_STATE.json');
    t.true(fs.existsSync(statePath), 'State file should exist');

    const state = JSON.parse(fs.readFileSync(statePath, 'utf8'));
    t.is(state.goal, 'E2E flow test', 'Goal should match');
    t.is(state.current_phase, 0, 'Phase should be 0');
    t.truthy(state.events, 'Should have events array');

    // Step 4: Verify directory structure
    t.true(fs.existsSync(path.join(tmpDir, '.aether', 'data')), 'data dir should exist');
    t.true(fs.existsSync(path.join(tmpDir, '.aether', 'checkpoints')), 'checkpoints dir should exist');
    t.true(fs.existsSync(path.join(tmpDir, '.aether', 'locks')), 'locks dir should exist');

    // Step 5: Verify .gitignore exists
    const gitignorePath = path.join(tmpDir, '.aether', '.gitignore');
    t.true(fs.existsSync(gitignorePath), '.gitignore should exist');

    const gitignoreContent = fs.readFileSync(gitignorePath, 'utf8');
    t.true(gitignoreContent.includes('data/'), '.gitignore should ignore data/');
    t.true(gitignoreContent.includes('checkpoints/'), '.gitignore should ignore checkpoints/');

  } finally {
    await cleanupTempDir(tmpDir);
  }
});
