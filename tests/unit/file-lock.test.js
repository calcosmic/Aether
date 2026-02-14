/**
 * FileLock Unit Tests
 *
 * Comprehensive tests for the FileLock class using sinon + proxyquire pattern.
 * Tests cover: acquire, release, stale detection, error handling, and idempotency.
 *
 * Uses test.serial() to avoid sinon stub conflicts between tests.
 */

const test = require('ava');
const sinon = require('sinon');
const proxyquire = require('proxyquire');

// Create a single sandbox for all tests
let sandbox;

// Create mock fs for testing
function createMockFs() {
  return {
    existsSync: sandbox.stub(),
    readFileSync: sandbox.stub(),
    writeFileSync: sandbox.stub(),
    openSync: sandbox.stub(),
    closeSync: sandbox.stub(),
    unlinkSync: sandbox.stub(),
    mkdirSync: sandbox.stub(),
    readdirSync: sandbox.stub(),
  };
}

// Load FileLock with mocked fs
function loadFileLock(mockFs) {
  return proxyquire('../../bin/lib/file-lock.js', {
    fs: mockFs,
  });
}

test.before(() => {
  // Create sandbox once
  sandbox = sinon.createSandbox();
});

test.beforeEach((t) => {
  // Reset sandbox before each test
  sandbox.restore();

  // Create fresh mock fs for each test
  t.context.mockFs = createMockFs();

  // Default: lock directory exists
  t.context.mockFs.existsSync.withArgs('.aether/locks').returns(true);

  // Load FileLock with mocks
  const { FileLock } = loadFileLock(t.context.mockFs);
  t.context.FileLock = FileLock;

  // Create instance with short timeout for faster tests
  t.context.fileLock = new FileLock({
    lockDir: '.aether/locks',
    timeout: 100,
    retryInterval: 10,
    maxRetries: 5,
  });

  // Stub process.kill for PID checking (default: only current process exists)
  sandbox.stub(process, 'kill').callsFake((pid, signal) => {
    if (signal === 0) {
      if (pid === process.pid) {
        return true;
      }
      const error = new Error('ESRCH');
      error.code = 'ESRCH';
      throw error;
    }
    return true;
  });
});

test.afterEach(() => {
  // Restore all stubs
  sandbox.restore();
});

test.after(() => {
  // Clean up sandbox
  sandbox.restore();
});

// Test 1: acquire creates lock file atomically
test.serial('acquire creates lock file atomically', (t) => {
  const { mockFs, fileLock } = t.context;

  // Setup: no existing lock
  mockFs.existsSync.withArgs('.aether/locks/state.json.lock').returns(false);
  mockFs.openSync.withArgs('.aether/locks/state.json.lock', 'wx').returns(1);

  // Execute
  const result = fileLock.acquire('/test/state.json');

  // Assert
  t.true(result);
  t.true(mockFs.openSync.calledWith('.aether/locks/state.json.lock', 'wx'));
  t.true(mockFs.writeFileSync.calledWith(1, process.pid.toString(), 'utf8'));
  t.true(mockFs.closeSync.calledWith(1));
  t.true(mockFs.writeFileSync.calledWith('.aether/locks/state.json.lock.pid', process.pid.toString(), 'utf8'));
});

// Test 2: acquire detects and cleans stale locks
test.serial('acquire detects and cleans stale locks', (t) => {
  const { mockFs, fileLock } = t.context;

  // Setup: existing lock with stale PID
  mockFs.existsSync.withArgs('.aether/locks/state.json.lock').returns(true);
  mockFs.existsSync.withArgs('.aether/locks/state.json.lock.pid').returns(true);
  mockFs.readFileSync.withArgs('.aether/locks/state.json.lock.pid', 'utf8').returns('12345');

  // Stale lock cleanup should succeed
  mockFs.unlinkSync.withArgs('.aether/locks/state.json.lock').returns();
  mockFs.unlinkSync.withArgs('.aether/locks/state.json.lock.pid').returns();

  // Then new lock acquisition succeeds
  mockFs.openSync.withArgs('.aether/locks/state.json.lock', 'wx').returns(1);

  // Execute
  const result = fileLock.acquire('/test/state.json');

  // Assert
  t.true(result);
  t.true(mockFs.unlinkSync.calledWith('.aether/locks/state.json.lock'));
  t.true(mockFs.unlinkSync.calledWith('.aether/locks/state.json.lock.pid'));
  t.true(mockFs.openSync.calledWith('.aether/locks/state.json.lock', 'wx'));
});

// Test 3: acquire respects running process locks
test.serial('acquire respects running process locks and returns false on timeout', (t) => {
  const { mockFs, fileLock } = t.context;

  // Setup: existing lock with running PID
  mockFs.existsSync.withArgs('.aether/locks/state.json.lock').returns(true);
  mockFs.existsSync.withArgs('.aether/locks/state.json.lock.pid').returns(true);
  mockFs.readFileSync.withArgs('.aether/locks/state.json.lock.pid', 'utf8').returns('99999');

  // Change stub behavior to simulate running process
  process.kill.callsFake((pid, signal) => {
    if (signal === 0 && pid === 99999) {
      return true; // Process is running
    }
    const error = new Error('ESRCH');
    error.code = 'ESRCH';
    throw error;
  });

  // Lock acquisition fails (EEXIST)
  const error = new Error('EEXIST');
  error.code = 'EEXIST';
  mockFs.openSync.withArgs('.aether/locks/state.json.lock', 'wx').throws(error);

  // Execute
  const result = fileLock.acquire('/test/state.json');

  // Assert
  t.false(result);
});

// Test 4: release cleans up lock files
test.serial('release cleans up lock files', (t) => {
  const { mockFs, fileLock } = t.context;

  // Setup: first acquire a lock
  mockFs.existsSync.withArgs('.aether/locks/state.json.lock').returns(false);
  mockFs.openSync.withArgs('.aether/locks/state.json.lock', 'wx').returns(1);
  fileLock.acquire('/test/state.json');

  // Setup: lock files exist for cleanup
  mockFs.existsSync.withArgs('.aether/locks/state.json.lock').returns(true);
  mockFs.existsSync.withArgs('.aether/locks/state.json.lock.pid').returns(true);

  // Execute
  const result = fileLock.release();

  // Assert
  t.true(result);
  t.true(mockFs.unlinkSync.calledWith('.aether/locks/state.json.lock'));
  t.true(mockFs.unlinkSync.calledWith('.aether/locks/state.json.lock.pid'));
});

// Test 5: isLocked returns correct state
test.serial('isLocked returns true when lock exists', (t) => {
  const { mockFs, fileLock } = t.context;

  // Setup: lock exists
  mockFs.existsSync.withArgs('.aether/locks/state.json.lock').returns(true);

  // Execute & Assert
  t.true(fileLock.isLocked('/test/state.json'));
});

test.serial('isLocked returns false when lock does not exist', (t) => {
  const { mockFs, fileLock } = t.context;

  // Setup: lock does not exist
  mockFs.existsSync.withArgs('.aether/locks/state.json.lock').returns(false);

  // Execute & Assert
  t.false(fileLock.isLocked('/test/state.json'));
});

// Test 6: handles fs errors gracefully
test.serial('acquire throws FileSystemError on unexpected fs errors', (t) => {
  const { mockFs, FileLock } = t.context;

  // Create fresh instance to test error handling
  const fileLock = new FileLock({
    lockDir: '.aether/locks',
    timeout: 100,
    retryInterval: 10,
    maxRetries: 5,
  });

  // Setup: no existing lock
  mockFs.existsSync.withArgs('.aether/locks/state.json.lock').returns(false);

  // openSync throws non-EEXIST error
  const error = new Error('EACCES');
  error.code = 'EACCES';
  mockFs.openSync.withArgs('.aether/locks/state.json.lock', 'wx').throws(error);

  // Execute & Assert
  const err = t.throws(() => fileLock.acquire('/test/state.json'));
  t.is(err.code, 'E_FILE_SYSTEM');
  t.true(err.message.includes('Failed to acquire lock'));
});

// Test 7: multiple release calls are idempotent
test.serial('multiple release calls are idempotent', (t) => {
  const { mockFs, fileLock } = t.context;

  // Setup: acquire lock first
  mockFs.existsSync.withArgs('.aether/locks/state.json.lock').returns(false);
  mockFs.openSync.withArgs('.aether/locks/state.json.lock', 'wx').returns(1);
  fileLock.acquire('/test/state.json');

  // Setup: lock files exist
  mockFs.existsSync.withArgs('.aether/locks/state.json.lock').returns(true);
  mockFs.existsSync.withArgs('.aether/locks/state.json.lock.pid').returns(true);

  // First release should succeed
  const result1 = fileLock.release();
  t.true(result1);

  // Second release should return false (no lock held)
  const result2 = fileLock.release();
  t.false(result2);

  // unlinkSync should only be called for the first release
  t.is(mockFs.unlinkSync.callCount, 2); // Once for .lock, once for .pid
});

// Test 8: getLockHolder returns correct PID
test.serial('getLockHolder returns PID of lock holder', (t) => {
  const { mockFs, fileLock } = t.context;

  // Setup: lock exists with PID
  mockFs.existsSync.withArgs('.aether/locks/state.json.lock.pid').returns(true);
  mockFs.readFileSync.withArgs('.aether/locks/state.json.lock.pid', 'utf8').returns('12345');

  // Execute
  const pid = fileLock.getLockHolder('/test/state.json');

  // Assert
  t.is(pid, 12345);
});

test.serial('getLockHolder returns null when no lock', (t) => {
  const { mockFs, fileLock } = t.context;

  // Setup: no lock
  mockFs.existsSync.withArgs('.aether/locks/state.json.lock.pid').returns(false);

  // Execute
  const pid = fileLock.getLockHolder('/test/state.json');

  // Assert
  t.is(pid, null);
});

// Test 9: constructor creates lock directory if it does not exist
test.serial('constructor creates lock directory if it does not exist', (t) => {
  const { mockFs, FileLock } = t.context;

  // Setup: lock directory does not exist
  mockFs.existsSync.withArgs('.aether/locks').returns(false);

  // Execute
  new FileLock({ lockDir: '.aether/locks' });

  // Assert
  t.true(mockFs.mkdirSync.calledWith('.aether/locks', { recursive: true }));
});

// Test 10: cleanupAll removes stale locks
test.serial('cleanupAll removes stale locks', (t) => {
  const { mockFs, fileLock } = t.context;

  // Setup: lock directory exists with files
  mockFs.existsSync.withArgs('.aether/locks').returns(true);
  mockFs.readdirSync.withArgs('.aether/locks').returns([
    'state.json.lock',
    'state.json.lock.pid',
    'other.json.lock',
    'other.json.lock.pid',
  ]);

  // Setup: stale lock (PID not running)
  mockFs.readFileSync.withArgs('.aether/locks/state.json.lock.pid', 'utf8').returns('12345');
  mockFs.readFileSync.withArgs('.aether/locks/other.json.lock.pid', 'utf8').returns('67890');

  // Execute
  const cleaned = fileLock.cleanupAll();

  // Assert
  t.is(cleaned, 4); // All 4 files cleaned (both stale)
  t.true(mockFs.unlinkSync.calledWith('.aether/locks/state.json.lock'));
  t.true(mockFs.unlinkSync.calledWith('.aether/locks/state.json.lock.pid'));
});

// Test 11: cleanupAll skips locks held by running processes
test.serial('cleanupAll skips locks held by running processes', (t) => {
  const { mockFs, fileLock } = t.context;

  // Setup: lock directory exists with files
  mockFs.existsSync.withArgs('.aether/locks').returns(true);
  mockFs.readdirSync.withArgs('.aether/locks').returns([
    'state.json.lock',
    'state.json.lock.pid',
  ]);

  // Change stub behavior to simulate running process (all PIDs exist)
  process.kill.callsFake((pid, signal) => {
    if (signal === 0) {
      return true; // All processes exist
    }
    return true;
  });

  // Setup: lock held by running process
  mockFs.existsSync.withArgs('.aether/locks/state.json.lock.pid').returns(true);
  mockFs.readFileSync.withArgs('.aether/locks/state.json.lock.pid', 'utf8').returns('12345');

  // Execute
  const cleaned = fileLock.cleanupAll();

  // Assert
  t.is(cleaned, 0); // No files cleaned (process is running)
  t.false(mockFs.unlinkSync.called);
});

// Test 12: waitForLock returns true when lock is released
test.serial('waitForLock returns true when lock is released quickly', (t) => {
  const { mockFs, fileLock } = t.context;

  // Setup: lock exists initially, then released
  let checkCount = 0;
  mockFs.existsSync.withArgs('.aether/locks/state.json.lock').callsFake(() => {
    checkCount++;
    return checkCount < 3; // Released after 2 checks
  });

  // Execute
  const result = fileLock.waitForLock('/test/state.json', 100);

  // Assert
  t.true(result);
});

// Test 13: waitForLock returns false on timeout
test.serial('waitForLock returns false on timeout', (t) => {
  const { mockFs, fileLock } = t.context;

  // Setup: lock never released
  mockFs.existsSync.withArgs('.aether/locks/state.json.lock').returns(true);

  // Execute with very short timeout
  const result = fileLock.waitForLock('/test/state.json', 5);

  // Assert
  t.false(result);
});

// Test 14: acquire returns false when retries exhausted
test.serial('acquire returns false when max retries exhausted', (t) => {
  const { mockFs, fileLock } = t.context;

  // Setup: lock held by running process
  mockFs.existsSync.withArgs('.aether/locks/state.json.lock').returns(true);
  mockFs.existsSync.withArgs('.aether/locks/state.json.lock.pid').returns(true);
  mockFs.readFileSync.withArgs('.aether/locks/state.json.lock.pid', 'utf8').returns('99999');

  // Change stub behavior to simulate running process
  process.kill.callsFake((pid, signal) => {
    if (signal === 0 && pid === 99999) {
      return true;
    }
    const error = new Error('ESRCH');
    error.code = 'ESRCH';
    throw error;
  });

  // Lock acquisition always fails
  const error = new Error('EEXIST');
  error.code = 'EEXIST';
  mockFs.openSync.withArgs('.aether/locks/state.json.lock', 'wx').throws(error);

  // Execute
  const result = fileLock.acquire('/test/state.json');

  // Assert
  t.false(result);
});

// Test 15: handles malformed PID files gracefully
test.serial('acquire handles malformed PID files as stale', (t) => {
  const { mockFs, fileLock } = t.context;

  // Setup: existing lock with malformed PID
  mockFs.existsSync.withArgs('.aether/locks/state.json.lock').returns(true);
  mockFs.existsSync.withArgs('.aether/locks/state.json.lock.pid').returns(true);
  mockFs.readFileSync.withArgs('.aether/locks/state.json.lock.pid', 'utf8').returns('not-a-number');

  // Cleanup succeeds
  mockFs.unlinkSync.withArgs('.aether/locks/state.json.lock').returns();
  mockFs.unlinkSync.withArgs('.aether/locks/state.json.lock.pid').returns();

  // New lock acquisition succeeds
  mockFs.openSync.withArgs('.aether/locks/state.json.lock', 'wx').returns(1);

  // Execute
  const result = fileLock.acquire('/test/state.json');

  // Assert
  t.true(result);
  t.true(mockFs.unlinkSync.calledWith('.aether/locks/state.json.lock'));
});
