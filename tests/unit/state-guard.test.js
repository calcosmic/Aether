/**
 * Unit Tests for StateGuard
 *
 * Tests the Iron Law enforcement, idempotency checks, and file locking
 * using sinon stubs and proxyquire for module mocking.
 *
 * @module tests/unit/state-guard
 */

const test = require('ava');
const sinon = require('sinon');
const proxyquire = require('proxyquire');
const { createMockFs, setupMockFiles, resetMockFs } = require('./helpers/mock-fs');

// Test context for shared mocks
let mockFs;
let StateGuard;
let StateGuardError;
let StateGuardErrorCodes;

test.before(() => {
  // Create mock fs with sinon stubs
  mockFs = createMockFs();

  // Load state-guard.js with mocked fs
  const module = proxyquire('../../bin/lib/state-guard.js', {
    fs: mockFs
  });

  StateGuard = module.StateGuard;
  StateGuardError = module.StateGuardError;
  StateGuardErrorCodes = module.StateGuardErrorCodes;
});

test.afterEach(() => {
  // Reset stubs between tests
  resetMockFs(mockFs);
});

test.after(() => {
  sinon.restore();
});

/**
 * Helper: Create a valid state fixture
 */
function createValidState(overrides = {}) {
  return {
    version: '3.0',
    current_phase: overrides.current_phase ?? 5,
    initialized_at: overrides.initialized_at ?? '2026-02-14T10:00:00Z',
    last_updated: '2026-02-14T10:00:00Z',
    goal: 'Test goal',
    state: 'ACTIVE',
    memory: {
      phase_learnings: [],
      decisions: [],
      instincts: []
    },
    errors: { records: [], flagged_patterns: [] },
    signals: [],
    graveyards: [],
    events: overrides.events ?? [],
    ...overrides
  };
}

/**
 * Helper: Create valid evidence
 */
function createValidEvidence(overrides = {}) {
  return {
    checkpoint_hash: 'sha256:abc123def456',
    test_results: { passed: true, count: 10 },
    timestamp: overrides.timestamp ?? '2026-02-14T12:00:00Z',
    ...overrides
  };
}

/**
 * Helper: Setup FileLock to succeed
 */
function setupLockSuccess() {
  // Mock openSync for lock acquisition (wx flag creates exclusively)
  mockFs.openSync.callsFake((path, flags) => {
    if (flags === 'wx') {
      return 1; // Return file descriptor
    }
    const error = new Error(`Unexpected openSync call: ${path}, ${flags}`);
    throw error;
  });

  mockFs.writeFileSync.callsFake(() => {});
  mockFs.closeSync.callsFake(() => {});
  mockFs.mkdirSync.callsFake(() => {});
  mockFs.unlinkSync.callsFake(() => {});
}

// ============================================================================
// Test 1: advancePhase succeeds with valid evidence
// ============================================================================
test.serial('advancePhase succeeds with valid evidence', async t => {
  const state = createValidState({ current_phase: 5 });
  const evidence = createValidEvidence();

  setupMockFiles(mockFs, {
    '/test/COLONY_STATE.json': JSON.stringify(state)
  });
  setupLockSuccess();

  const guard = new StateGuard('/test/COLONY_STATE.json', {
    lockDir: '/test/locks',
    lock: { acquire: () => Promise.resolve(true), release: () => {} }
  });

  // Mock the lock methods directly
  guard.locked = true;

  const result = await guard.advancePhase(5, 6, evidence);

  t.is(result.status, 'transitioned');
  t.is(result.from, 5);
  t.is(result.to, 6);
});

// ============================================================================
// Test 2: advancePhase throws without evidence (Iron Law)
// ============================================================================
test.serial('advancePhase throws without evidence (Iron Law)', async t => {
  const state = createValidState({ current_phase: 5 });

  setupMockFiles(mockFs, {
    '/test/COLONY_STATE.json': JSON.stringify(state)
  });

  const guard = new StateGuard('/test/COLONY_STATE.json', {
    lockDir: '/test/locks',
    lock: { acquire: () => Promise.resolve(true), release: () => {} }
  });
  guard.locked = true;

  const error = await t.throwsAsync(
    async () => await guard.advancePhase(5, 6, null),
    { instanceOf: StateGuardError }
  );

  t.is(error.code, StateGuardErrorCodes.E_IRON_LAW_VIOLATION);
  t.true(error.message.includes('requires fresh verification evidence'));
  t.truthy(error.recovery);
});

// ============================================================================
// Test 3: advancePhase throws with stale evidence
// ============================================================================
test.serial('advancePhase throws with stale evidence', async t => {
  // State initialized at 10:00:00
  const state = createValidState({
    current_phase: 5,
    initialized_at: '2026-02-14T10:00:00Z'
  });

  // Evidence from 09:00:00 (before initialization)
  const evidence = createValidEvidence({
    timestamp: '2026-02-13T09:00:00Z'
  });

  setupMockFiles(mockFs, {
    '/test/COLONY_STATE.json': JSON.stringify(state)
  });

  const guard = new StateGuard('/test/COLONY_STATE.json', {
    lockDir: '/test/locks',
    lock: { acquire: () => Promise.resolve(true), release: () => {} }
  });
  guard.locked = true;

  const error = await t.throwsAsync(
    async () => await guard.advancePhase(5, 6, evidence),
    { instanceOf: StateGuardError }
  );

  t.is(error.code, StateGuardErrorCodes.E_IRON_LAW_VIOLATION);
});

// ============================================================================
// Test 4: idempotency prevents rebuilding completed phase
// ============================================================================
test.serial('idempotency prevents rebuilding completed phase', async t => {
  // State already at phase 6
  const state = createValidState({ current_phase: 6 });
  const evidence = createValidEvidence();

  setupMockFiles(mockFs, {
    '/test/COLONY_STATE.json': JSON.stringify(state)
  });

  const guard = new StateGuard('/test/COLONY_STATE.json', {
    lockDir: '/test/locks',
    lock: { acquire: () => Promise.resolve(true), release: () => {} }
  });
  guard.locked = true;

  const result = await guard.advancePhase(5, 6, evidence);

  t.is(result.status, 'already_complete');
  t.is(result.currentPhase, 6);
});

// ============================================================================
// Test 5: idempotency prevents skipping phases
// ============================================================================
test.serial('idempotency prevents skipping phases', async t => {
  // State at phase 4, trying to advance from 5 to 6
  const state = createValidState({ current_phase: 4 });
  const evidence = createValidEvidence();

  setupMockFiles(mockFs, {
    '/test/COLONY_STATE.json': JSON.stringify(state)
  });

  const guard = new StateGuard('/test/COLONY_STATE.json', {
    lockDir: '/test/locks',
    lock: { acquire: () => Promise.resolve(true), release: () => {} }
  });
  guard.locked = true;

  const error = await t.throwsAsync(
    async () => await guard.advancePhase(5, 6, evidence),
    { instanceOf: StateGuardError }
  );

  t.is(error.code, StateGuardErrorCodes.E_IDEMPOTENCY_CHECK);
  t.is(error.details.reason, 'previous_incomplete');
});

// ============================================================================
// Test 6: validates sequential transitions only
// ============================================================================
test.serial('validates sequential transitions only', async t => {
  const state = createValidState({ current_phase: 5 });
  const evidence = createValidEvidence();

  setupMockFiles(mockFs, {
    '/test/COLONY_STATE.json': JSON.stringify(state)
  });

  const guard = new StateGuard('/test/COLONY_STATE.json', {
    lockDir: '/test/locks',
    lock: { acquire: () => Promise.resolve(true), release: () => {} }
  });
  guard.locked = true;

  // Try to skip from 5 to 7
  const error = await t.throwsAsync(
    async () => await guard.advancePhase(5, 7, evidence),
    { instanceOf: StateGuardError }
  );

  t.is(error.code, StateGuardErrorCodes.E_INVALID_TRANSITION);
  t.is(error.details.from, 5);
  t.is(error.details.to, 7);
  t.is(error.details.expected, 6);
});

// ============================================================================
// Test 7: releases lock even on error
// ============================================================================
test.serial('releases lock even on error', async t => {
  const state = createValidState({ current_phase: 5 });

  setupMockFiles(mockFs, {
    '/test/COLONY_STATE.json': JSON.stringify(state)
  });

  let releaseCalled = false;
  const mockLock = {
    acquire: () => Promise.resolve(true),
    release: () => { releaseCalled = true; }
  };

  const guard = new StateGuard('/test/COLONY_STATE.json', {
    lockDir: '/test/locks',
    lock: mockLock
  });

  try {
    await guard.advancePhase(5, 6, null); // Will throw due to missing evidence
  } catch (e) {
    // Expected
  }

  t.true(releaseCalled, 'Lock should be released even on error');
});

// ============================================================================
// Test 8: hasFreshEvidence validates all required fields
// ============================================================================
test.serial('hasFreshEvidence validates all required fields', t => {
  const state = createValidState();
  const guard = new StateGuard('/test/COLONY_STATE.json');

  // Missing checkpoint_hash
  t.false(guard.hasFreshEvidence(state, 5, {
    test_results: {},
    timestamp: '2026-02-14T12:00:00Z'
  }));

  // Missing test_results
  t.false(guard.hasFreshEvidence(state, 5, {
    checkpoint_hash: 'sha256:abc',
    timestamp: '2026-02-14T12:00:00Z'
  }));

  // Missing timestamp
  t.false(guard.hasFreshEvidence(state, 5, {
    checkpoint_hash: 'sha256:abc',
    test_results: {}
  }));

  // Null evidence
  t.false(guard.hasFreshEvidence(state, 5, null));

  // Undefined evidence
  t.false(guard.hasFreshEvidence(state, 5, undefined));

  // Empty object
  t.false(guard.hasFreshEvidence(state, 5, {}));
});

// ============================================================================
// Test 9: StateGuardError toJSON structure
// ============================================================================
test.serial('StateGuardError toJSON structure', t => {
  const error = new StateGuardError(
    StateGuardErrorCodes.E_IRON_LAW_VIOLATION,
    'Test message',
    { key: 'value' },
    'Test recovery'
  );

  const json = error.toJSON();

  t.is(json.error.name, 'StateGuardError');
  t.is(json.error.code, StateGuardErrorCodes.E_IRON_LAW_VIOLATION);
  t.is(json.error.message, 'Test message');
  t.deepEqual(json.error.details, { key: 'value' });
  t.is(json.error.recovery, 'Test recovery');
  t.truthy(json.error.timestamp);
});

// ============================================================================
// Test 10: StateGuardError toString format
// ============================================================================
test.serial('StateGuardError toString format', t => {
  const error = new StateGuardError(
    StateGuardErrorCodes.E_LOCK_TIMEOUT,
    'Lock timeout',
    {},
    'Check lock file'
  );

  const str = error.toString();

  t.true(str.includes(StateGuardErrorCodes.E_LOCK_TIMEOUT));
  t.true(str.includes('Lock timeout'));
  t.true(str.includes('Recovery:'));
  t.true(str.includes('Check lock file'));
});

// ============================================================================
// Test 11: loadState validates required fields
// ============================================================================
test.serial('loadState validates required fields', t => {
  // Missing version
  setupMockFiles(mockFs, {
    '/test/COLONY_STATE.json': JSON.stringify({
      current_phase: 5,
      events: []
    })
  });

  const guard = new StateGuard('/test/COLONY_STATE.json');

  const error = t.throws(() => guard.loadState(), { instanceOf: StateGuardError });
  t.is(error.code, StateGuardErrorCodes.E_STATE_INVALID);
});

// ============================================================================
// Test 12: loadState throws for missing file
// ============================================================================
test.serial('loadState throws for missing file', t => {
  mockFs.existsSync.returns(false);

  const guard = new StateGuard('/test/COLONY_STATE.json');

  const error = t.throws(() => guard.loadState(), { instanceOf: StateGuardError });
  t.is(error.code, StateGuardErrorCodes.E_STATE_NOT_FOUND);
});

// ============================================================================
// Test 13: loadState throws for invalid JSON
// ============================================================================
test.serial('loadState throws for invalid JSON', t => {
  setupMockFiles(mockFs, {
    '/test/COLONY_STATE.json': 'not valid json'
  });

  const guard = new StateGuard('/test/COLONY_STATE.json');

  const error = t.throws(() => guard.loadState(), { instanceOf: StateGuardError });
  t.is(error.code, StateGuardErrorCodes.E_STATE_INVALID);
});

// ============================================================================
// Test 14: acquireLock throws on timeout
// ============================================================================
test.serial('acquireLock throws on timeout', async t => {
  const guard = new StateGuard('/test/COLONY_STATE.json', {
    lockDir: '/test/locks',
    lock: {
      acquire: () => Promise.resolve(false),
      release: () => {}
    }
  });

  const error = await t.throwsAsync(
    async () => await guard.acquireLock(),
    { instanceOf: StateGuardError }
  );

  t.is(error.code, StateGuardErrorCodes.E_LOCK_TIMEOUT);
});

// ============================================================================
// Test 15: saveState updates last_updated and writes atomically
// ============================================================================
test.serial('saveState updates last_updated and writes atomically', t => {
  const state = createValidState({ current_phase: 5 });

  setupMockFiles(mockFs, {
    '/test/COLONY_STATE.json': JSON.stringify(state)
  });

  let writtenToTemp = false;
  let renamed = false;

  mockFs.writeFileSync.callsFake((path, data) => {
    if (path === '/test/COLONY_STATE.json.tmp') {
      writtenToTemp = true;
      const parsed = JSON.parse(data);
      t.is(parsed.current_phase, 6); // Should have updated phase
      t.truthy(parsed.last_updated); // Should have updated timestamp
    }
  });

  mockFs.renameSync.callsFake((from, to) => {
    if (from === '/test/COLONY_STATE.json.tmp' && to === '/test/COLONY_STATE.json') {
      renamed = true;
    }
  });

  const guard = new StateGuard('/test/COLONY_STATE.json');
  const updated = guard.transitionState(state, 5, 6, createValidEvidence());
  guard.saveState(updated);

  t.true(writtenToTemp, 'Should write to temp file');
  t.true(renamed, 'Should rename atomically');
});

// ============================================================================
// Test 16: transitionState adds audit event
// ============================================================================
test.serial('transitionState adds audit event', t => {
  const state = createValidState({ current_phase: 5, events: [] });
  const evidence = createValidEvidence({ checkpoint_hash: 'sha256:test123' });

  const guard = new StateGuard('/test/COLONY_STATE.json');
  const updated = guard.transitionState(state, 5, 6, evidence);

  t.is(updated.current_phase, 6);
  t.is(updated.events.length, 1);

  const event = updated.events[0];
  t.is(event.type, 'phase_transition');
  t.is(event.details.from, 5);
  t.is(event.details.to, 6);
  t.is(event.details.evidence_id, 'sha256:test123');
  t.truthy(event.timestamp);
  t.truthy(event.worker);
});

// ============================================================================
// Test 17: hasFreshEvidence rejects invalid timestamp format
// ============================================================================
test.serial('hasFreshEvidence rejects invalid timestamp format', t => {
  const state = createValidState();
  const guard = new StateGuard('/test/COLONY_STATE.json');

  const evidence = {
    checkpoint_hash: 'sha256:abc',
    test_results: {},
    timestamp: 'not-a-valid-timestamp'
  };

  t.false(guard.hasFreshEvidence(state, 5, evidence));
});

// ============================================================================
// Test 18: releaseLock is safe to call when not locked
// ============================================================================
test.serial('releaseLock is safe to call when not locked', t => {
  const guard = new StateGuard('/test/COLONY_STATE.json', {
    lockDir: '/test/locks',
    lock: { release: () => {} }
  });

  // Should not throw
  t.notThrows(() => guard.releaseLock());
  t.false(guard.locked);
});
