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

// ============================================================================
// PLAN-006: Additional Resilience Tests
// ============================================================================

// Test 11: Phase 0 returns PLANNING not INITIALIZING (PLAN-006 fix #7)
test.serial('determineColonyState returns PLANNING for Phase 0', (t) => {
  const { stateSync } = t.context;

  // Phase 0 with no status should return PLANNING (not INITIALIZING)
  t.is(stateSync.determineColonyState(null, 0), 'PLANNING',
    'Phase 0 should return PLANNING, not INITIALIZING');

  // Phase 0 with status should still work
  t.is(stateSync.determineColonyState('READY', 0), 'PLANNING');
  t.is(stateSync.determineColonyState('BUILDING', 0), 'BUILDING');
});

// Test 12: syncStateFromPlanning distinguishes EACCES from ENOENT (PLAN-006 fix #9)
test.serial('syncStateFromPlanning reports permission denied for STATE.md', (t) => {
  const { mockFs, mockFileLock, stateSync, repoPath } = t.context;

  // Setup: STATE.md exists but is not readable
  const accessError = new Error('Permission denied');
  accessError.code = 'EACCES';
  mockFs.readFileSync.withArgs(path.join(repoPath, '.planning', 'STATE.md'), 'utf8')
    .throws(accessError);

  // Execute
  const result = stateSync.syncStateFromPlanning(repoPath);

  // Assert: Should return permission error, not generic error
  t.false(result.synced);
  t.true(result.error.includes('permission') || result.error.includes('denied'),
    `Error should mention permission: ${result.error}`);

  // Lock should have been released
  t.true(mockFileLock.release.calledOnce);
});

// Test 13: syncStateFromPlanning distinguishes EACCES for COLONY_STATE.json (PLAN-006 fix #9)
test.serial('syncStateFromPlanning reports permission denied for COLONY_STATE.json', (t) => {
  const { mockFs, mockFileLock, stateSync, repoPath } = t.context;

  // Setup: COLONY_STATE.json not accessible
  const accessError = new Error('Permission denied');
  accessError.code = 'EACCES';
  mockFs.existsSync.withArgs(path.join(repoPath, '.aether', 'data', 'COLONY_STATE.json'))
    .throws(accessError);

  // Execute
  const result = stateSync.syncStateFromPlanning(repoPath);

  // Assert: Should return permission error
  t.false(result.synced);
  t.true(result.error.includes('permission') || result.error.includes('accessible'),
    `Error should mention permission: ${result.error}`);
});

// ============================================================================
// PLAN-007: State Schema Validation Tests
// ============================================================================

// Test 14: validateStateSchema accepts valid state (PLAN-007 Fix 3)
test.serial('validateStateSchema accepts valid state', (t) => {
  const { stateSync } = t.context;

  const validState = {
    version: '3.0',
    current_phase: 1,
    events: [
      { timestamp: '2026-01-01T00:00:00Z', type: 'test', worker: 'test-worker' }
    ],
    goal: 'Test Goal',
    state: 'BUILDING'
  };

  const result = stateSync.validateStateSchema(validState);

  t.true(result.valid);
  t.is(result.errors.length, 0);
});

// Test 15: validateStateSchema rejects missing required fields (PLAN-007 Fix 3)
test.serial('validateStateSchema rejects missing required fields', (t) => {
  const { stateSync } = t.context;

  const invalidState = {
    goal: 'Test Goal'
    // Missing version, current_phase, events
  };

  const result = stateSync.validateStateSchema(invalidState);

  t.false(result.valid);
  t.true(result.errors.some(e => e.includes('version')));
  t.true(result.errors.some(e => e.includes('current_phase')));
  t.true(result.errors.some(e => e.includes('events')));
});

// Test 16: validateStateSchema rejects wrong types (PLAN-007 Fix 3)
test.serial('validateStateSchema rejects wrong types', (t) => {
  const { stateSync } = t.context;

  const invalidState = {
    version: 3,  // Should be string
    current_phase: '1',  // Should be number
    events: {}  // Should be array
  };

  const result = stateSync.validateStateSchema(invalidState);

  t.false(result.valid);
  t.true(result.errors.some(e => e.includes('version') && e.includes('string')));
  t.true(result.errors.some(e => e.includes('current_phase') && e.includes('number')));
  t.true(result.errors.some(e => e.includes('events') && e.includes('array')));
});

// Test 17: validateStateSchema validates event structure (PLAN-007 Fix 3)
test.serial('validateStateSchema validates event structure', (t) => {
  const { stateSync } = t.context;

  const invalidState = {
    version: '3.0',
    current_phase: 1,
    events: [
      { timestamp: '2026-01-01T00:00:00Z' }, // Missing type and worker
      { type: 'test', worker: 'test' }, // Missing timestamp
      { timestamp: '2026-01-01T00:00:00Z', type: 'test' } // Missing worker
    ]
  };

  const result = stateSync.validateStateSchema(invalidState);

  t.false(result.valid);
  t.true(result.errors.some(e => e.includes('events[0]') && e.includes('type')));
  t.true(result.errors.some(e => e.includes('events[0]') && e.includes('worker')));
  t.true(result.errors.some(e => e.includes('events[1]') && e.includes('timestamp')));
  t.true(result.errors.some(e => e.includes('events[2]') && e.includes('worker')));
});

// Test 18: validateStateSchema handles null/undefined (PLAN-007 Fix 3)
test.serial('validateStateSchema handles null/undefined', (t) => {
  const { stateSync } = t.context;

  t.false(stateSync.validateStateSchema(null).valid);
  t.false(stateSync.validateStateSchema(undefined).valid);
  t.false(stateSync.validateStateSchema('string').valid);
  t.false(stateSync.validateStateSchema(123).valid);
  t.false(stateSync.validateStateSchema([]).valid);
});

// Test 19: syncStateFromPlanning rejects invalid state before write (PLAN-007 Fix 3)
test.serial('syncStateFromPlanning rejects invalid state before write', (t) => {
  const { mockFs, mockFileLock, stateSync, repoPath } = t.context;

  // Setup: COLONY_STATE.json has invalid schema (events as object instead of array)
  mockFs.readFileSync.withArgs(path.join(repoPath, '.aether', 'data', 'COLONY_STATE.json'), 'utf8')
    .returns(JSON.stringify({
      version: '3.0',
      goal: 'Test Milestone',
      state: 'BUILDING',
      current_phase: 1,
      plan: { phases: [] },
      events: {} // Invalid - should be array
    }));

  // Make state change to trigger write
  mockFs.readFileSync.withArgs(path.join(repoPath, '.planning', 'STATE.md'), 'utf8')
    .returns('Phase 2\nMilestone: Different Milestone\nStatus: BUILDING');

  // Execute
  const result = stateSync.syncStateFromPlanning(repoPath);

  // Assert: Should return schema validation error
  t.false(result.synced);
  t.true(result.error.includes('schema validation') || result.error.includes('events'),
    `Error should mention schema validation: ${result.error}`);

  // Check recovery field exists
  t.truthy(result.recovery, `Recovery should exist: ${result.recovery}`);

  // Should not have written to temp file (validation happens before write)
  const writeCalls = mockFs.writeFileSync.getCalls();
  const wroteTmp = writeCalls.some(call => call.args[0] && call.args[0].includes('.tmp'));
  t.false(wroteTmp, 'Should not write .tmp file when validation fails');

  // Lock should have been released
  t.true(mockFileLock.release.calledOnce);
});

// ============================================================================
// PLAN-007: Events Pruning Tests
// ============================================================================

// Test 20: pruneEvents returns array unchanged if under limit (PLAN-007 Fix 2)
test.serial('pruneEvents returns array unchanged if under limit', (t) => {
  const { stateSync } = t.context;

  const events = [
    { timestamp: '2026-01-01T00:00:00Z', type: 'test', worker: 'test' }
  ];

  const result = stateSync.pruneEvents(events);

  t.is(result.length, 1);
  t.is(result[0].timestamp, '2026-01-01T00:00:00Z');
});

// Test 21: pruneEvents keeps most recent events (PLAN-007 Fix 2)
test.serial('pruneEvents keeps most recent events', (t) => {
  const { stateSync } = t.context;

  const events = [];
  // Create 150 events with incrementing timestamps in 2024
  for (let i = 0; i < 150; i++) {
    // Use months to avoid day overflow - spread across the year
    const month = Math.floor(i / 31);
    const day = (i % 31) + 1;
    events.push({
      timestamp: new Date(2024, month, day).toISOString(),
      type: 'test',
      worker: 'test'
    });
  }

  const result = stateSync.pruneEvents(events, 100);

  t.is(result.length, 100);
  // Most recent should be first (sorted descending)
  // The last event added has the highest timestamp (month 4, day 27)
  const lastMonth = Math.floor(149 / 31);
  const lastDay = (149 % 31) + 1;
  t.is(result[0].timestamp, new Date(2024, lastMonth, lastDay).toISOString());
  // Oldest kept should be event 50 (month 1, day 20)
  const oldestMonth = Math.floor(50 / 31);
  const oldestDay = (50 % 31) + 1;
  t.is(result[99].timestamp, new Date(2024, oldestMonth, oldestDay).toISOString());
});

// Test 22: pruneEvents handles non-array input (PLAN-007 Fix 2)
test.serial('pruneEvents handles non-array input', (t) => {
  const { stateSync } = t.context;

  t.is(stateSync.pruneEvents(null), null);
  t.is(stateSync.pruneEvents(undefined), undefined);
  t.is(stateSync.pruneEvents('string'), 'string');
  t.deepEqual(stateSync.pruneEvents({}), {});
});

// Test 23: pruneEvents handles empty array (PLAN-007 Fix 2)
test.serial('pruneEvents handles empty array', (t) => {
  const { stateSync } = t.context;

  const result = stateSync.pruneEvents([]);

  t.deepEqual(result, []);
});

// Test 24: syncStateFromPlanning prunes events after adding (PLAN-007 Fix 2)
test.serial('syncStateFromPlanning prunes events after adding', (t) => {
  const { mockFs, mockFileLock, stateSync, repoPath } = t.context;

  // Setup: Create state with 150 events to trigger actual pruning
  const existingEvents = [];
  for (let i = 0; i < 150; i++) {
    // Use months to avoid day overflow - spread across multiple years
    const month = Math.floor(i / 31);
    const day = (i % 31) + 1;
    existingEvents.push({
      timestamp: new Date(2024, month, day).toISOString(),
      type: 'old_event',
      worker: 'test'
    });
  }

  mockFs.readFileSync.withArgs(path.join(repoPath, '.aether', 'data', 'COLONY_STATE.json'), 'utf8')
    .returns(JSON.stringify({
      version: '3.0',
      goal: 'Test Milestone',
      state: 'BUILDING',
      current_phase: 1,
      plan: { phases: [] },
      events: existingEvents
    }));

  // Make state change to trigger sync event
  mockFs.readFileSync.withArgs(path.join(repoPath, '.planning', 'STATE.md'), 'utf8')
    .returns('Phase 2\nMilestone: Different Milestone\nStatus: BUILDING');

  // Execute
  const result = stateSync.syncStateFromPlanning(repoPath);

  // Assert
  t.true(result.synced);

  // Check that writeFileSync was called
  t.true(mockFs.writeFileSync.called);

  // Get the written state
  const writeCall = mockFs.writeFileSync.getCalls().find(c => c.args[0].includes('.tmp'));
  const writtenState = JSON.parse(writeCall.args[1]);

  // Should have exactly 100 events after pruning (150 + 1 sync, pruned to 100)
  t.is(writtenState.events.length, 100, 'Events should be pruned to 100');

  // Most recent event should be the sync event (has current timestamp in 2026)
  // All old events are from 2024, so the sync event (2026) should be first
  t.is(writtenState.events[0].type, 'state_synced_from_planning',
    'First event should be the sync event (most recent)');
});
