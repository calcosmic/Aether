/**
 * State Sync Unit Tests
 *
 * Tests for the state-sync module covering:
 * - JSON parse error handling (PLAN-002)
 * - File locking behavior
 * - Atomic write pattern
 *
 * Uses sinon + proxyquire pattern consistent with other tests.
 */

const test = require('ava');
const sinon = require('sinon');
const proxyquire = require('proxyquire');
const path = require('path');

// Create a single sandbox for all tests
let sandbox;

// Create mock fs for testing
function createMockFs() {
  return {
    existsSync: sandbox.stub(),
    readFileSync: sandbox.stub(),
    writeFileSync: sandbox.stub(),
    renameSync: sandbox.stub(),
    mkdirSync: sandbox.stub(),
  };
}

// Create mock FileLock
function createMockFileLock() {
  return {
    acquire: sandbox.stub().returns(true),
    release: sandbox.stub().returns(true),
  };
}

// Load state-sync with mocked dependencies
function loadStateSync(mockFs, mockFileLock) {
  return proxyquire('../../bin/lib/state-sync.js', {
    fs: mockFs,
    './file-lock': {
      FileLock: function() {
        return mockFileLock;
      }
    }
  });
}

test.before(() => {
  sandbox = sinon.createSandbox();
});

test.beforeEach((t) => {
  sandbox.restore();

  t.context.mockFs = createMockFs();
  t.context.mockFileLock = createMockFileLock();

  // Load state-sync with mocks
  const stateSync = loadStateSync(t.context.mockFs, t.context.mockFileLock);
  t.context.stateSync = stateSync;

  // Setup default: files exist (using full paths that will be constructed)
  const repoPath = '/test';
  t.context.repoPath = repoPath;

  t.context.mockFs.existsSync.withArgs(path.join(repoPath, '.planning', 'STATE.md')).returns(true);
  t.context.mockFs.existsSync.withArgs(path.join(repoPath, '.planning', 'ROADMAP.md')).returns(true);
  t.context.mockFs.existsSync.withArgs(path.join(repoPath, '.aether', 'data', 'COLONY_STATE.json')).returns(true);
  t.context.mockFs.existsSync.withArgs(path.join(repoPath, '.aether', 'locks')).returns(true);

  // Default file contents
  t.context.mockFs.readFileSync.withArgs(path.join(repoPath, '.planning', 'STATE.md'), 'utf8')
    .returns('Phase 1\nMilestone: Test Milestone\nStatus: BUILDING');
  t.context.mockFs.readFileSync.withArgs(path.join(repoPath, '.planning', 'ROADMAP.md'), 'utf8')
    .returns('## Phase 1: Test Phase');
  t.context.mockFs.readFileSync.withArgs(path.join(repoPath, '.aether', 'data', 'COLONY_STATE.json'), 'utf8')
    .returns(JSON.stringify({
      version: '3.0',
      goal: 'Test Milestone',
      state: 'BUILDING',
      current_phase: 1,
      plan: { phases: [] },
      events: []
    }));
});

test.afterEach(() => {
  sandbox.restore();
});

test.after(() => {
  sandbox.restore();
});

// Test 1: syncStateFromPlanning handles corrupted JSON gracefully (PLAN-002)
test.serial('syncStateFromPlanning handles corrupted JSON gracefully', (t) => {
  const { mockFs, mockFileLock, stateSync, repoPath } = t.context;

  // Setup: COLONY_STATE.json exists but is corrupted
  mockFs.readFileSync.withArgs(path.join(repoPath, '.aether', 'data', 'COLONY_STATE.json'), 'utf8')
    .returns('{"broken": not valid json');

  // Execute
  const result = stateSync.syncStateFromPlanning(repoPath);

  // Assert: Should return error, not throw
  t.false(result.synced);
  t.truthy(result.error);
  t.true(result.error.includes('invalid JSON') || result.error.includes('parse'));
  t.truthy(result.recovery);
  t.true(result.recovery.includes('fix or delete'));

  // Lock should have been acquired and released
  t.true(mockFileLock.acquire.calledOnce);
  t.true(mockFileLock.release.calledOnce);
});

// Test 2: syncStateFromPlanning acquires lock before modifying state (PLAN-002)
test.serial('syncStateFromPlanning acquires lock before modifying state', (t) => {
  const { mockFs, mockFileLock, stateSync, repoPath } = t.context;

  // Execute
  stateSync.syncStateFromPlanning(repoPath);

  // Assert: Lock was acquired
  t.true(mockFileLock.acquire.calledOnce);
  t.true(mockFileLock.acquire.calledBefore(mockFs.writeFileSync));
});

// Test 3: syncStateFromPlanning releases lock after completion (PLAN-002)
test.serial('syncStateFromPlanning releases lock after completion', (t) => {
  const { mockFs, mockFileLock, stateSync, repoPath } = t.context;

  // Execute
  stateSync.syncStateFromPlanning(repoPath);

  // Assert: Lock was released
  t.true(mockFileLock.release.calledOnce);
});

// Test 4: syncStateFromPlanning releases lock on error (PLAN-002)
test.serial('syncStateFromPlanning releases lock on error', (t) => {
  const { mockFs, mockFileLock, stateSync, repoPath } = t.context;

  // Setup: Force an error during processing
  mockFs.readFileSync.withArgs(path.join(repoPath, '.planning', 'STATE.md'), 'utf8')
    .throws(new Error('Read error'));

  // Execute
  const result = stateSync.syncStateFromPlanning(repoPath);

  // Assert: Error returned and lock released
  t.false(result.synced);
  t.true(mockFileLock.release.calledOnce);
});

// Test 5: syncStateFromPlanning uses atomic write pattern (PLAN-002)
test.serial('syncStateFromPlanning uses atomic write pattern', (t) => {
  const { mockFs, mockFileLock, stateSync, repoPath } = t.context;

  // Setup: Make state actually change (different milestone)
  mockFs.readFileSync.withArgs(path.join(repoPath, '.planning', 'STATE.md'), 'utf8')
    .returns('Phase 1\nMilestone: NEW Milestone\nStatus: BUILDING');

  // Execute
  const result = stateSync.syncStateFromPlanning(repoPath);

  const statePath = path.join(repoPath, '.aether', 'data', 'COLONY_STATE.json');

  // Assert: Write to temp file, then rename
  t.true(result.synced);
  t.true(result.changed);
  t.true(mockFs.writeFileSync.calledWith(`${statePath}.tmp`));
  t.true(mockFs.renameSync.calledWith(`${statePath}.tmp`, statePath));
});

// Test 6: syncStateFromPlanning returns lock error when cannot acquire
test.serial('syncStateFromPlanning returns lock error when cannot acquire', (t) => {
  const { mockFileLock, stateSync, repoPath } = t.context;

  // Setup: Lock acquisition fails
  mockFileLock.acquire.returns(false);

  // Execute
  const result = stateSync.syncStateFromPlanning(repoPath);

  // Assert
  t.false(result.synced);
  t.true(result.error.includes('lock'));
  t.false(mockFileLock.release.called); // Never acquired, so no release
});

// Test 7: reconcileStates handles corrupted JSON gracefully
test.serial('reconcileStates handles corrupted JSON gracefully', (t) => {
  const { mockFs, stateSync, repoPath } = t.context;

  // Setup: COLONY_STATE.json exists but is corrupted
  mockFs.readFileSync.withArgs(path.join(repoPath, '.aether', 'data', 'COLONY_STATE.json'), 'utf8')
    .returns('{"broken": not valid json');

  // Execute
  const result = stateSync.reconcileStates(repoPath);

  // Assert: Should return error, not throw
  t.false(result.consistent);
  t.true(result.mismatches.length > 0);
  t.true(result.mismatches[0].includes('invalid JSON'));
  t.truthy(result.resolution);
});

// Test 8: parseStateMd handles empty content
test.serial('parseStateMd handles empty content', (t) => {
  const { stateSync } = t.context;

  const result = stateSync.parseStateMd('');

  t.is(result.phase, null);
  t.is(result.milestone, null);
  t.is(result.status, null);
});

// Test 9: parseStateMd extracts phase correctly
test.serial('parseStateMd extracts phase correctly', (t) => {
  const { stateSync } = t.context;

  const result = stateSync.parseStateMd('Phase 5\nMilestone: Test');

  t.is(result.phase, 5);
});

// Test 10: determineColonyState returns correct states
test.serial('determineColonyState returns correct states', (t) => {
  const { stateSync } = t.context;

  t.is(stateSync.determineColonyState('BUILDING', 1), 'BUILDING');
  t.is(stateSync.determineColonyState('PLANNING', 1), 'PLANNING');
  t.is(stateSync.determineColonyState('COMPLETE', 1), 'COMPLETED');
  t.is(stateSync.determineColonyState(null, null), 'INITIALIZING');
});
