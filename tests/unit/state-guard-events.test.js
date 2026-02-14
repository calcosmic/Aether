/**
 * Unit Tests for StateGuard Event Audit Trail
 *
 * Tests event recording, validation, and querying functionality
 * using sinon stubs and proxyquire for module mocking.
 *
 * @module tests/unit/state-guard-events
 */

const test = require('ava');
const sinon = require('sinon');
const proxyquire = require('proxyquire');
const { createMockFs, setupMockFiles, resetMockFs } = require('./helpers/mock-fs');

// Test context for shared mocks
let mockFs;
let StateGuard;
let EventTypes;
let validateEvent;
let createEvent;

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
  mockFs.openSync.callsFake((path, flags) => {
    if (flags === 'wx') {
      return 1;
    }
    const error = new Error(`Unexpected openSync call: ${path}, ${flags}`);
    throw error;
  });

  mockFs.writeFileSync.callsFake(() => {});
  mockFs.closeSync.callsFake(() => {});
  mockFs.mkdirSync.callsFake(() => {});
  mockFs.unlinkSync.callsFake(() => {});
}

test.before(() => {
  mockFs = createMockFs();

  // Load event-types module with mocked fs
  const eventTypesModule = proxyquire('../../bin/lib/event-types.js', {
    fs: mockFs
  });
  EventTypes = eventTypesModule.EventTypes;
  validateEvent = eventTypesModule.validateEvent;
  createEvent = eventTypesModule.createEvent;

  // Load state-guard module with mocked fs and event-types
  const stateGuardModule = proxyquire('../../bin/lib/state-guard.js', {
    fs: mockFs,
    './event-types': eventTypesModule
  });
  StateGuard = stateGuardModule.StateGuard;
});

test.afterEach(() => {
  resetMockFs(mockFs);
});

test.after(() => {
  sinon.restore();
});

// ============================================================================
// Test 1: advancePhase creates phase_transition event
// ============================================================================
test.serial('advancePhase creates phase_transition event', async t => {
  const state = createValidState({ current_phase: 5, events: [] });
  const evidence = createValidEvidence({ checkpoint_hash: 'sha256:test789' });

  setupMockFiles(mockFs, {
    '/test/COLONY_STATE.json': JSON.stringify(state)
  });
  setupLockSuccess();

  const guard = new StateGuard('/test/COLONY_STATE.json', {
    lockDir: '/test/locks',
    lock: { acquire: () => Promise.resolve(true), release: () => {} },
    worker: 'TestWorker'
  });
  guard.locked = true;

  const result = await guard.advancePhase(5, 6, evidence);

  t.is(result.status, 'transitioned');
  t.is(result.from, 5);
  t.is(result.to, 6);

  // Verify the saved state has the event
  const savedState = JSON.parse(mockFs.writeFileSync.lastCall.args[1]);
  t.is(savedState.events.length, 1);

  const event = savedState.events[0];
  t.is(event.type, EventTypes.PHASE_TRANSITION);
  t.is(event.details.from, 5);
  t.is(event.details.to, 6);
  t.is(event.details.checkpoint_hash, 'sha256:test789');
  t.truthy(event.timestamp);
  t.is(event.worker, 'TestWorker');
});

// ============================================================================
// Test 2: validateEvent accepts valid events
// ============================================================================
test.serial('validateEvent accepts valid events', t => {
  const validEvent = {
    timestamp: '2026-02-14T14:30:22.123Z',
    type: EventTypes.PHASE_TRANSITION,
    worker: 'TestWorker',
    details: { from: 5, to: 6 }
  };

  const result = validateEvent(validEvent);
  t.true(result.valid);
  t.is(result.errors.length, 0);
});

// ============================================================================
// Test 3: validateEvent rejects invalid events
// ============================================================================
test.serial('validateEvent rejects events with missing timestamp', t => {
  const event = {
    type: EventTypes.PHASE_TRANSITION,
    worker: 'TestWorker',
    details: {}
  };

  const result = validateEvent(event);
  t.false(result.valid);
  t.true(result.errors.includes('Missing required field: timestamp'));
});

test.serial('validateEvent rejects events with invalid type', t => {
  const event = {
    timestamp: '2026-02-14T14:30:22Z',
    type: 'invalid_type',
    worker: 'TestWorker',
    details: {}
  };

  const result = validateEvent(event);
  t.false(result.valid);
  t.true(result.errors.some(e => e.includes('type must be a valid EventType')));
});

test.serial('validateEvent rejects events with missing worker', t => {
  const event = {
    timestamp: '2026-02-14T14:30:22Z',
    type: EventTypes.PHASE_TRANSITION,
    details: {}
  };

  const result = validateEvent(event);
  t.false(result.valid);
  t.true(result.errors.includes('Missing required field: worker'));
});

test.serial('validateEvent rejects events with empty worker', t => {
  const event = {
    timestamp: '2026-02-14T14:30:22Z',
    type: EventTypes.PHASE_TRANSITION,
    worker: '   ',
    details: {}
  };

  const result = validateEvent(event);
  t.false(result.valid);
  t.true(result.errors.includes('worker must not be empty'));
});

test.serial('validateEvent rejects events with invalid timestamp format', t => {
  const event = {
    timestamp: 'not-a-valid-timestamp',
    type: EventTypes.PHASE_TRANSITION,
    worker: 'TestWorker',
    details: {}
  };

  const result = validateEvent(event);
  t.false(result.valid);
  t.true(result.errors.some(e => e.includes('timestamp must be valid ISO 8601')));
});

test.serial('validateEvent rejects events with array details', t => {
  const event = {
    timestamp: '2026-02-14T14:30:22Z',
    type: EventTypes.PHASE_TRANSITION,
    worker: 'TestWorker',
    details: []
  };

  const result = validateEvent(event);
  t.false(result.valid);
  t.true(result.errors.some(e => e.includes('details must be an object, not an array')));
});

// ============================================================================
// Test 4: createEvent generates correct structure
// ============================================================================
test.serial('createEvent generates correct structure', t => {
  const beforeTime = Date.now();
  const event = createEvent(EventTypes.CHECKPOINT_CREATED, 'TestWorker', { id: '123' });
  const afterTime = Date.now();

  t.is(event.type, EventTypes.CHECKPOINT_CREATED);
  t.is(event.worker, 'TestWorker');
  t.deepEqual(event.details, { id: '123' });

  // Verify timestamp is recent (within 1 second)
  const eventTime = new Date(event.timestamp).getTime();
  t.true(eventTime >= beforeTime - 1000 && eventTime <= afterTime + 1000);

  // Verify timestamp format
  t.regex(event.timestamp, /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d{3})?Z$/);
});

test.serial('createEvent throws on invalid type', t => {
  const error = t.throws(() => {
    createEvent('invalid_type', 'TestWorker', {});
  });

  t.true(error.message.includes('Invalid event type'));
});

test.serial('createEvent uses environment worker as fallback', t => {
  const originalWorker = process.env.WORKER_NAME;
  process.env.WORKER_NAME = 'EnvWorker';

  try {
    const event = createEvent(EventTypes.PHASE_TRANSITION, null, {});
    t.is(event.worker, 'EnvWorker');
  } finally {
    if (originalWorker) {
      process.env.WORKER_NAME = originalWorker;
    } else {
      delete process.env.WORKER_NAME;
    }
  }
});

// ============================================================================
// Test 5: getEvents filters correctly
// ============================================================================
test.serial('getEvents filters by type', t => {
  const state = createValidState({
    events: [
      { timestamp: '2026-02-14T10:00:00Z', type: EventTypes.PHASE_TRANSITION, worker: 'w1', details: {} },
      { timestamp: '2026-02-14T10:01:00Z', type: EventTypes.CHECKPOINT_CREATED, worker: 'w2', details: {} },
      { timestamp: '2026-02-14T10:02:00Z', type: EventTypes.PHASE_TRANSITION, worker: 'w3', details: {} }
    ]
  });

  const transitions = StateGuard.getEvents(state, { type: EventTypes.PHASE_TRANSITION });
  t.is(transitions.length, 2);
  t.is(transitions[0].type, EventTypes.PHASE_TRANSITION);
  t.is(transitions[1].type, EventTypes.PHASE_TRANSITION);
});

test.serial('getEvents filters by since timestamp', t => {
  const state = createValidState({
    events: [
      { timestamp: '2026-02-14T10:00:00Z', type: EventTypes.PHASE_TRANSITION, worker: 'w1', details: {} },
      { timestamp: '2026-02-14T10:30:00Z', type: EventTypes.PHASE_TRANSITION, worker: 'w2', details: {} },
      { timestamp: '2026-02-14T11:00:00Z', type: EventTypes.PHASE_TRANSITION, worker: 'w3', details: {} }
    ]
  });

  const recent = StateGuard.getEvents(state, { since: '2026-02-14T10:15:00Z' });
  t.is(recent.length, 2);
  t.is(recent[0].worker, 'w3'); // Most recent first
  t.is(recent[1].worker, 'w2');
});

test.serial('getEvents respects limit option', t => {
  const state = createValidState({
    events: [
      { timestamp: '2026-02-14T10:00:00Z', type: EventTypes.PHASE_TRANSITION, worker: 'w1', details: {} },
      { timestamp: '2026-02-14T10:01:00Z', type: EventTypes.PHASE_TRANSITION, worker: 'w2', details: {} },
      { timestamp: '2026-02-14T10:02:00Z', type: EventTypes.PHASE_TRANSITION, worker: 'w3', details: {} }
    ]
  });

  const limited = StateGuard.getEvents(state, { limit: 2 });
  t.is(limited.length, 2);
  t.is(limited[0].worker, 'w3'); // Most recent
  t.is(limited[1].worker, 'w2');
});

test.serial('getEvents combines filters', t => {
  const state = createValidState({
    events: [
      { timestamp: '2026-02-14T10:00:00Z', type: EventTypes.PHASE_TRANSITION, worker: 'w1', details: {} },
      { timestamp: '2026-02-14T10:30:00Z', type: EventTypes.CHECKPOINT_CREATED, worker: 'w2', details: {} },
      { timestamp: '2026-02-14T11:00:00Z', type: EventTypes.PHASE_TRANSITION, worker: 'w3', details: {} },
      { timestamp: '2026-02-14T11:30:00Z', type: EventTypes.PHASE_TRANSITION, worker: 'w4', details: {} }
    ]
  });

  const filtered = StateGuard.getEvents(state, {
    type: EventTypes.PHASE_TRANSITION,
    since: '2026-02-14T10:15:00Z',
    limit: 1
  });

  t.is(filtered.length, 1);
  t.is(filtered[0].worker, 'w4'); // Most recent phase_transition after 10:15
});

test.serial('getEvents returns empty array for invalid state', t => {
  t.deepEqual(StateGuard.getEvents(null), []);
  t.deepEqual(StateGuard.getEvents({}), []);
  t.deepEqual(StateGuard.getEvents({ events: 'not-an-array' }), []);
});

// ============================================================================
// Test 6: getLatestEvent returns most recent
// ============================================================================
test.serial('getLatestEvent returns most recent event of given type', t => {
  const state = createValidState({
    events: [
      { timestamp: '2026-02-14T10:00:00Z', type: EventTypes.PHASE_TRANSITION, worker: 'w1', details: { from: 1, to: 2 } },
      { timestamp: '2026-02-14T10:30:00Z', type: EventTypes.CHECKPOINT_CREATED, worker: 'w2', details: {} },
      { timestamp: '2026-02-14T11:00:00Z', type: EventTypes.PHASE_TRANSITION, worker: 'w3', details: { from: 2, to: 3 } },
      { timestamp: '2026-02-14T11:30:00Z', type: EventTypes.PHASE_TRANSITION, worker: 'w4', details: { from: 3, to: 4 } }
    ]
  });

  const latest = StateGuard.getLatestEvent(state, EventTypes.PHASE_TRANSITION);
  t.truthy(latest);
  t.is(latest.worker, 'w4');
  t.is(latest.details.from, 3);
  t.is(latest.details.to, 4);
});

test.serial('getLatestEvent returns null when no events of type', t => {
  const state = createValidState({
    events: [
      { timestamp: '2026-02-14T10:00:00Z', type: EventTypes.CHECKPOINT_CREATED, worker: 'w1', details: {} }
    ]
  });

  const latest = StateGuard.getLatestEvent(state, EventTypes.PHASE_TRANSITION);
  t.is(latest, null);
});

test.serial('getLatestEvent returns most recent of any type when no type specified', t => {
  const state = createValidState({
    events: [
      { timestamp: '2026-02-14T10:00:00Z', type: EventTypes.PHASE_TRANSITION, worker: 'w1', details: {} },
      { timestamp: '2026-02-14T11:00:00Z', type: EventTypes.CHECKPOINT_CREATED, worker: 'w2', details: {} }
    ]
  });

  const latest = StateGuard.getLatestEvent(state);
  t.truthy(latest);
  t.is(latest.worker, 'w2');
});

test.serial('getLatestEvent returns null for empty state', t => {
  t.is(StateGuard.getLatestEvent(null), null);
  t.is(StateGuard.getLatestEvent({}), null);
  t.is(StateGuard.getLatestEvent({ events: [] }), null);
});

// ============================================================================
// Test 7: addEvent method works correctly
// ============================================================================
test.serial('addEvent creates and adds event to state', t => {
  const state = createValidState({ events: [] });

  const guard = new StateGuard('/test/COLONY_STATE.json', {
    worker: 'TestWorker'
  });

  const event = guard.addEvent(state, EventTypes.UPDATE_STARTED, { version: '1.2.0' });

  t.is(state.events.length, 1);
  t.is(state.events[0].type, EventTypes.UPDATE_STARTED);
  t.is(state.events[0].worker, 'TestWorker');
  t.deepEqual(state.events[0].details, { version: '1.2.0' });
  t.is(event.type, EventTypes.UPDATE_STARTED);
});

test.serial('addEvent creates events array if missing', t => {
  const state = createValidState();
  delete state.events;

  const guard = new StateGuard('/test/COLONY_STATE.json');
  guard.addEvent(state, EventTypes.PHASE_BUILD_STARTED, {});

  t.truthy(state.events);
  t.is(state.events.length, 1);
});
