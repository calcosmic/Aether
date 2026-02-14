/**
 * Initialization Module Unit Tests
 *
 * Tests for bin/lib/init.js - repo initialization with local state
 */

const test = require('ava');
const sinon = require('sinon');
const proxyquire = require('proxyquire');
const { createMockFs, resetMockFs, setupMockFiles } = require('./helpers/mock-fs');

// Mock fs module for testing
let mockFs;
let init;

test.before(() => {
  mockFs = createMockFs();
  init = proxyquire('../../bin/lib/init', {
    fs: mockFs
  });
});

test.beforeEach(() => {
  resetMockFs(mockFs);
});

test.serial('initializeRepo creates directory structure', async (t) => {
  const repoPath = '/test/repo';

  // Setup: hub exists, repo doesn't
  setupMockFiles(mockFs, {
    [require('os').homedir() + '/.aether']: null,
    [require('os').homedir() + '/.aether/system']: null,
    [require('os').homedir() + '/.aether/version.json']: JSON.stringify({ version: '1.1.0' }),
    [require('os').homedir() + '/.aether/registry.json']: JSON.stringify({ schema_version: 1, repos: [] })
  });

  // Call initializeRepo
  const result = await init.initializeRepo(repoPath, { goal: 'Test', quiet: true });

  // Assert: mkdirSync called for required directories
  t.true(mockFs.mkdirSync.calledWith('/test/repo/.aether', { recursive: true }));
  t.true(mockFs.mkdirSync.calledWith('/test/repo/.aether/data', { recursive: true }));
  t.true(mockFs.mkdirSync.calledWith('/test/repo/.aether/checkpoints', { recursive: true }));
  t.true(mockFs.mkdirSync.calledWith('/test/repo/.aether/locks', { recursive: true }));

  // Assert: success result
  t.true(result.success);
  t.is(result.stateFile, '/test/repo/.aether/data/COLONY_STATE.json');
});

test.serial('initializeRepo creates COLONY_STATE.json with correct structure', async (t) => {
  const repoPath = '/test/repo';

  // Setup: hub exists, repo doesn't
  setupMockFiles(mockFs, {
    [require('os').homedir() + '/.aether']: null,
    [require('os').homedir() + '/.aether/system']: null,
    [require('os').homedir() + '/.aether/version.json']: JSON.stringify({ version: '1.1.0' }),
    [require('os').homedir() + '/.aether/registry.json']: JSON.stringify({ schema_version: 1, repos: [] })
  });

  // Call initializeRepo
  await init.initializeRepo(repoPath, { goal: 'Test goal', quiet: true });

  // Assert: writeFileSync called with COLONY_STATE.json path
  const writeCall = mockFs.writeFileSync.getCalls().find(
    call => call.args[0] === '/test/repo/.aether/data/COLONY_STATE.json'
  );
  t.truthy(writeCall, 'writeFileSync should be called for COLONY_STATE.json');

  // Parse the written content
  const writtenContent = writeCall.args[1];
  const state = JSON.parse(writtenContent);

  // Assert: state has correct structure
  t.is(state.version, '3.0');
  t.is(state.goal, 'Test goal');
  t.is(state.state, 'INITIALIZING');
  t.is(state.current_phase, 0);
  t.truthy(state.session_id, 'Should have session_id');
  t.truthy(state.session_id.startsWith('session_'), 'session_id should start with session_');
  t.truthy(state.initialized_at, 'Should have initialized_at');
  t.truthy(state.created_at, 'Should have created_at');
  t.truthy(state.last_updated, 'Should have last_updated');

  // Assert: plan structure
  t.truthy(state.plan, 'Should have plan');
  t.deepEqual(state.plan.phases, []);

  // Assert: memory structure
  t.truthy(state.memory, 'Should have memory');
  t.deepEqual(state.memory.phase_learnings, []);
  t.deepEqual(state.memory.decisions, []);
  t.deepEqual(state.memory.instincts, []);

  // Assert: errors structure
  t.truthy(state.errors, 'Should have errors');
  t.deepEqual(state.errors.records, []);
  t.deepEqual(state.errors.flagged_patterns, []);

  // Assert: events array with colony_initialized event
  t.true(Array.isArray(state.events), 'events should be an array');
  t.is(state.events.length, 1);
  t.is(state.events[0].type, 'colony_initialized');
  t.is(state.events[0].worker, 'init');
  t.true(state.events[0].details.includes('Test goal'));
});

test.serial('isInitialized returns true when state exists', (t) => {
  const repoPath = '/test/repo';

  // Setup: all required files exist
  setupMockFiles(mockFs, {
    '/test/repo/.aether': null,
    '/test/repo/.aether/data': null,
    '/test/repo/.aether/checkpoints': null,
    '/test/repo/.aether/locks': null,
    '/test/repo/.aether/data/COLONY_STATE.json': JSON.stringify({ version: '3.0' })
  });

  // Assert: isInitialized returns true
  const result = init.isInitialized(repoPath);
  t.true(result);
});

test.serial('isInitialized returns false when state missing', (t) => {
  const repoPath = '/test/repo';

  // Setup: nothing exists
  mockFs.existsSync.returns(false);

  // Assert: isInitialized returns false
  const result = init.isInitialized(repoPath);
  t.false(result);
});

test.serial('isInitialized returns false when directories missing', (t) => {
  const repoPath = '/test/repo';

  // Setup: state file exists but directories missing
  let callCount = 0;
  mockFs.existsSync.callsFake((path) => {
    // Only state file exists
    return path === '/test/repo/.aether/data/COLONY_STATE.json';
  });

  // Assert: isInitialized returns false
  const result = init.isInitialized(repoPath);
  t.false(result);
});

test.serial('validateInitialization returns valid for complete setup', (t) => {
  const repoPath = '/test/repo';

  // Setup: complete valid state
  setupMockFiles(mockFs, {
    '/test/repo/.aether': null,
    '/test/repo/.aether/data': null,
    '/test/repo/.aether/checkpoints': null,
    '/test/repo/.aether/locks': null,
    '/test/repo/.aether/data/COLONY_STATE.json': JSON.stringify({
      version: '3.0',
      goal: 'Test',
      state: 'INITIALIZING',
      current_phase: 0,
      session_id: 'session_test',
      initialized_at: '2026-01-01T00:00:00Z',
      events: []
    })
  });

  // Call validateInitialization
  const result = init.validateInitialization(repoPath);

  // Assert: valid
  t.true(result.valid);
  t.deepEqual(result.errors, []);
});

test.serial('validateInitialization returns errors for missing fields', (t) => {
  const repoPath = '/test/repo';

  // Setup: state file with missing fields
  setupMockFiles(mockFs, {
    '/test/repo/.aether': null,
    '/test/repo/.aether/data': null,
    '/test/repo/.aether/checkpoints': null,
    '/test/repo/.aether/locks': null,
    '/test/repo/.aether/data/COLONY_STATE.json': JSON.stringify({
      version: '3.0'
      // Missing: goal, state, current_phase, session_id, initialized_at
    })
  });

  // Call validateInitialization
  const result = init.validateInitialization(repoPath);

  // Assert: invalid with errors
  t.false(result.valid);
  t.true(result.errors.length > 0);
  t.true(result.errors.some(e => e.includes('goal')));
  t.true(result.errors.some(e => e.includes('state')));
  t.true(result.errors.some(e => e.includes('current_phase')));
  t.true(result.errors.some(e => e.includes('session_id')));
  t.true(result.errors.some(e => e.includes('initialized_at')));
});

test.serial('validateInitialization returns error for invalid JSON', (t) => {
  const repoPath = '/test/repo';

  // Setup: invalid JSON in state file
  setupMockFiles(mockFs, {
    '/test/repo/.aether': null,
    '/test/repo/.aether/data': null,
    '/test/repo/.aether/checkpoints': null,
    '/test/repo/.aether/locks': null,
    '/test/repo/.aether/data/COLONY_STATE.json': 'invalid json {'
  });

  // Call validateInitialization
  const result = init.validateInitialization(repoPath);

  // Assert: invalid with JSON error
  t.false(result.valid);
  t.true(result.errors.some(e => e.includes('Invalid JSON')));
});

test.serial('validateInitialization returns errors for missing directories', (t) => {
  const repoPath = '/test/repo';

  // Setup: only state file exists (no directories)
  // Use direct stub configuration instead of setupMockFiles to avoid
  // auto-detection of parent directories
  mockFs.existsSync.callsFake((path) => {
    // Only the state file exists, no directories
    return path === '/test/repo/.aether/data/COLONY_STATE.json';
  });

  mockFs.readFileSync.callsFake((path) => {
    if (path === '/test/repo/.aether/data/COLONY_STATE.json') {
      return JSON.stringify({
        version: '3.0',
        goal: 'Test',
        state: 'INITIALIZING',
        current_phase: 0,
        session_id: 'session_test',
        initialized_at: '2026-01-01T00:00:00Z',
        events: []
      });
    }
    throw new Error('ENOENT');
  });

  // Call validateInitialization
  const result = init.validateInitialization(repoPath);

  // Assert: invalid with directory errors
  t.false(result.valid);
  t.true(result.errors.some(e => e === 'Missing directory: .aether/'));
  t.true(result.errors.some(e => e === 'Missing directory: .aether/data/'));
  t.true(result.errors.some(e => e === 'Missing directory: .aether/checkpoints/'));
  t.true(result.errors.some(e => e === 'Missing directory: .aether/locks/'));
});

test.serial('initializeRepo respects skipIfExists when already initialized', async (t) => {
  const repoPath = '/test/repo';

  // Setup: already initialized
  setupMockFiles(mockFs, {
    '/test/repo/.aether': null,
    '/test/repo/.aether/data': null,
    '/test/repo/.aether/checkpoints': null,
    '/test/repo/.aether/locks': null,
    '/test/repo/.aether/data/COLONY_STATE.json': JSON.stringify({ version: '3.0' })
  });

  // Call initializeRepo with skipIfExists: true
  const result = await init.initializeRepo(repoPath, { skipIfExists: true });

  // Assert: success but no files written
  t.true(result.success);
  t.is(result.message, 'Repository already initialized, skipping');

  // Verify no writeFileSync calls for state
  const stateWriteCalls = mockFs.writeFileSync.getCalls().filter(
    call => call.args[0] && call.args[0].includes('COLONY_STATE.json')
  );
  t.is(stateWriteCalls.length, 0, 'Should not write state file when skipping');
});

test.serial('createInitialState generates unique session IDs', (t) => {
  const state1 = init.createInitialState('Goal 1');
  const state2 = init.createInitialState('Goal 2');

  // Assert: different session IDs
  t.not(state1.session_id, state2.session_id);
  t.true(state1.session_id.startsWith('session_'));
  t.true(state2.session_id.startsWith('session_'));
});

test.serial('generateSessionId creates valid session ID format', (t) => {
  const sessionId = init.generateSessionId();

  // Assert: format is session_{timestamp}_{random}
  t.true(sessionId.startsWith('session_'));
  const parts = sessionId.split('_');
  t.is(parts.length, 3);
  t.true(/\d+/.test(parts[1]), 'Timestamp should be numeric');
  t.true(/[a-z0-9]+/.test(parts[2]), 'Random part should be alphanumeric');
});
