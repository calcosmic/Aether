/**
 * FileLock Integration Tests (PLAN-007 Fix 4)
 *
 * Tests for concurrent access, crash scenarios, and PID reuse.
 * These tests use real filesystem operations to verify integration behavior.
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');
const { FileLock } = require('../../bin/lib/file-lock');
const { promisify } = require('util');

const setTimeoutPromise = promisify(setTimeout);

// Test directory for integration tests
const TEST_LOCK_DIR = '/tmp/aether-lock-integration-test';

// Helper to create unique test directory
function createTestDir() {
  const testDir = `${TEST_LOCK_DIR}-${Date.now()}-${Math.random().toString(36).slice(2)}`;
  fs.mkdirSync(testDir, { recursive: true });
  return testDir;
}

// Helper to cleanup test directory
function cleanupTestDir(testDir) {
  try {
    fs.rmSync(testDir, { recursive: true, force: true });
  } catch {
    // Ignore cleanup errors
  }
}

// Helper to create a mock lock file (simulating stale lock)
function createStaleLock(lockDir, resourceName, pid, ageMs = 0) {
  // Ensure lock directory exists
  if (!fs.existsSync(lockDir)) {
    fs.mkdirSync(lockDir, { recursive: true });
  }

  const lockFile = path.join(lockDir, `${resourceName}.lock`);
  const pidFile = `${lockFile}.pid`;

  // Create lock file with PID
  fs.writeFileSync(lockFile, pid.toString());

  // Create PID file
  fs.writeFileSync(pidFile, pid.toString());

  // Set mtime to simulate age if specified
  if (ageMs > 0) {
    const oldTime = new Date(Date.now() - ageMs);
    fs.utimesSync(lockFile, oldTime, oldTime);
  }

  return { lockFile, pidFile };
}

test.before(() => {
  // Ensure base test directory exists
  try {
    fs.mkdirSync(TEST_LOCK_DIR, { recursive: true });
  } catch {
    // Ignore if exists
  }
});

test.after.always(() => {
  // Cleanup base test directory
  try {
    fs.rmSync(TEST_LOCK_DIR, { recursive: true, force: true });
  } catch {
    // Ignore cleanup errors
  }
});

// ============================================================================
// Concurrent Access Tests
// ============================================================================

// Test 1: Two processes attempting concurrent state sync - one wins, one waits
test.serial('concurrent lock acquisition - second process waits', async (t) => {
  const testDir = createTestDir();
  const lockDir = path.join(testDir, 'locks');
  const resourcePath = path.join(testDir, 'state.json');

  try {
    const lock1 = new FileLock({
      lockDir,
      timeout: 2000,
      retryInterval: 50,
      maxRetries: 40,
    });

    const lock2 = new FileLock({
      lockDir,
      timeout: 2000,
      retryInterval: 50,
      maxRetries: 40,
    });

    // First lock acquires
    const acquired1 = await lock1.acquireAsync(resourcePath);
    t.true(acquired1, 'First lock should acquire');

    // Track if second lock acquired
    let lock2Acquired = false;
    let lock2Released = false;

    // Start second lock acquisition in background
    const lock2Promise = lock2.acquireAsync(resourcePath).then(acquired => {
      lock2Acquired = acquired;
      if (acquired) {
        // Hold briefly then release
        setTimeoutPromise(50).then(() => {
          lock2.release();
          lock2Released = true;
        });
      }
      return acquired;
    });

    // Wait a bit then release first lock
    await setTimeoutPromise(100);
    lock1.release();

    // Now second lock should be able to acquire
    const acquired2 = await lock2Promise;
    t.true(acquired2, 'Second lock should acquire after first is released');

    // Wait for cleanup
    await setTimeoutPromise(100);

  } finally {
    cleanupTestDir(testDir);
  }
});

// Test 2: Concurrent acquireAsync yields to event loop during wait
test.serial('acquireAsync yields to event loop during wait', async (t) => {
  const testDir = createTestDir();
  const lockDir = path.join(testDir, 'locks');
  const resourcePath = path.join(testDir, 'state.json');

  try {
    // Create a lock held by current process (will NOT be cleaned up as stale)
    const lock1 = new FileLock({
      lockDir,
      timeout: 1000,
    });
    await lock1.acquireAsync(resourcePath);

    const lock2 = new FileLock({
      lockDir,
      timeout: 500,
      retryInterval: 100,
      maxRetries: 5,
    });

    // Track if timer fires during acquireAsync
    let timerFired = false;
    const timer = setTimeout(() => {
      timerFired = true;
    }, 50);

    // Attempt to acquire with lock2 (will retry and timeout since lock1 holds it)
    await lock2.acquireAsync(resourcePath);

    clearTimeout(timer);

    // Release lock1
    lock1.release();

    // Timer should have fired because event loop was not blocked during retries
    t.true(timerFired, 'Event loop should not be blocked during acquireAsync wait');

  } finally {
    cleanupTestDir(testDir);
  }
});

// ============================================================================
// Crash Recovery Tests
// ============================================================================

// Test 3: Stale lock from crashed process is cleaned up
test.serial('stale lock from crashed process is cleaned up', async (t) => {
  const testDir = createTestDir();
  const lockDir = path.join(testDir, 'locks');
  const resourcePath = path.join(testDir, 'state.json');

  try {
    // Create stale lock with non-running PID
    const nonRunningPid = 999999;
    createStaleLock(lockDir, 'state.json', nonRunningPid);

    const lock = new FileLock({ lockDir, timeout: 1000 });

    // Should be able to acquire (stale lock cleaned up)
    const acquired = await lock.acquireAsync(resourcePath);

    t.true(acquired, 'Should acquire lock after cleaning up stale lock');

    lock.release();

  } finally {
    cleanupTestDir(testDir);
  }
});

// Test 4: Lock recovery when PID file exists but lock file missing
test.serial('lock recovery when PID file exists but lock file missing', async (t) => {
  const testDir = createTestDir();
  const lockDir = path.join(testDir, 'locks');
  const resourcePath = path.join(testDir, 'state.json');

  try {
    // Ensure lock directory exists
    fs.mkdirSync(lockDir, { recursive: true });

    // Create only PID file (lock file missing - crash scenario)
    const pidFile = path.join(lockDir, 'state.json.lock.pid');
    fs.writeFileSync(pidFile, '999999');

    const lock = new FileLock({ lockDir, timeout: 1000 });

    // Should be able to acquire (orphaned PID file cleaned up)
    const acquired = await lock.acquireAsync(resourcePath);

    t.true(acquired, 'Should acquire lock despite orphaned PID file');

    lock.release();

  } finally {
    cleanupTestDir(testDir);
  }
});

// Test 5: Lock recovery when lock file exists but PID file missing
test.serial('lock recovery when lock file exists but PID file missing', async (t) => {
  const testDir = createTestDir();
  const lockDir = path.join(testDir, 'locks');
  const resourcePath = path.join(testDir, 'state.json');

  try {
    // Ensure lock directory exists
    fs.mkdirSync(lockDir, { recursive: true });

    // Create only lock file (PID file missing - crash scenario)
    const lockFile = path.join(lockDir, 'state.json.lock');
    fs.writeFileSync(lockFile, '999999');

    const lock = new FileLock({ lockDir, timeout: 1000 });

    // Should be able to acquire (stale lock cleaned up via lock file PID)
    const acquired = await lock.acquireAsync(resourcePath);

    t.true(acquired, 'Should acquire lock after cleaning up stale lock file');

    lock.release();

  } finally {
    cleanupTestDir(testDir);
  }
});

// ============================================================================
// PID Reuse Tests
// ============================================================================

// Test 6: Old lock is cleaned up when maxLockAge exceeded
test.serial('old lock is cleaned up when maxLockAge exceeded', async (t) => {
  const testDir = createTestDir();
  const lockDir = path.join(testDir, 'locks');
  const resourcePath = path.join(testDir, 'state.json');

  try {
    // Create lock that is 10 minutes old with current PID (simulates PID reuse)
    const currentPid = process.pid;
    createStaleLock(lockDir, 'state.json', currentPid, 10 * 60 * 1000);

    const lock = new FileLock({
      lockDir,
      timeout: 1000,
      maxLockAge: 5 * 60 * 1000, // 5 minutes
    });

    // Should be able to acquire (lock is old even though PID is running)
    const acquired = await lock.acquireAsync(resourcePath);

    t.true(acquired, 'Should acquire lock when maxLockAge exceeded');

    lock.release();

  } finally {
    cleanupTestDir(testDir);
  }
});

// Test 7: Lock held by running process is NOT cleaned even if old (within maxLockAge)
test.serial('lock held by running process is NOT cleaned within maxLockAge', async (t) => {
  const testDir = createTestDir();
  const lockDir = path.join(testDir, 'locks');
  const resourcePath = path.join(testDir, 'state.json');

  try {
    // Create lock with current PID and recent timestamp
    const currentPid = process.pid;
    createStaleLock(lockDir, 'state.json', currentPid, 1000); // 1 second old

    const lock = new FileLock({
      lockDir,
      timeout: 200, // Short timeout
      maxLockAge: 5 * 60 * 1000, // 5 minutes
    });

    // Should NOT be able to acquire (lock is valid and held by running process)
    const acquired = await lock.acquireAsync(resourcePath);

    t.false(acquired, 'Should NOT acquire lock held by running process');

  } finally {
    cleanupTestDir(testDir);
  }
});

// ============================================================================
// Edge Cases
// ============================================================================

// Test 8: Handles corrupted PID file gracefully
test.serial('handles corrupted PID file gracefully', async (t) => {
  const testDir = createTestDir();
  const lockDir = path.join(testDir, 'locks');
  const resourcePath = path.join(testDir, 'state.json');

  try {
    // Ensure lock directory exists
    fs.mkdirSync(lockDir, { recursive: true });

    // Create lock files with corrupted PID data
    const lockFile = path.join(lockDir, 'state.json.lock');
    const pidFile = `${lockFile}.pid`;

    fs.writeFileSync(lockFile, 'state'); // Not a PID
    fs.writeFileSync(pidFile, 'not-a-number'); // Corrupted

    const lock = new FileLock({ lockDir, timeout: 1000 });

    // Should be able to acquire (corrupted lock cleaned up)
    const acquired = await lock.acquireAsync(resourcePath);

    t.true(acquired, 'Should acquire lock after cleaning up corrupted lock');

    lock.release();

  } finally {
    cleanupTestDir(testDir);
  }
});

// Test 9: Handles empty PID file gracefully
test.serial('handles empty PID file gracefully', async (t) => {
  const testDir = createTestDir();
  const lockDir = path.join(testDir, 'locks');
  const resourcePath = path.join(testDir, 'state.json');

  try {
    // Ensure lock directory exists
    fs.mkdirSync(lockDir, { recursive: true });

    // Create lock files with empty PID file
    const lockFile = path.join(lockDir, 'state.json.lock');
    const pidFile = `${lockFile}.pid`;

    fs.writeFileSync(lockFile, '123');
    fs.writeFileSync(pidFile, ''); // Empty

    const lock = new FileLock({ lockDir, timeout: 1000 });

    // Should be able to acquire (empty PID file treated as stale)
    const acquired = await lock.acquireAsync(resourcePath);

    t.true(acquired, 'Should acquire lock after cleaning up lock with empty PID file');

    lock.release();

  } finally {
    cleanupTestDir(testDir);
  }
});

// Test 10: Release is idempotent - safe to call multiple times
test.serial('release is idempotent - safe to call multiple times', async (t) => {
  const testDir = createTestDir();
  const lockDir = path.join(testDir, 'locks');
  const resourcePath = path.join(testDir, 'state.json');

  try {
    const lock = new FileLock({ lockDir, timeout: 1000 });

    // Acquire lock
    const acquired = await lock.acquireAsync(resourcePath);
    t.true(acquired);

    // Release multiple times - should not throw
    const result1 = lock.release();
    t.true(result1, 'First release should return true');

    const result2 = lock.release();
    t.false(result2, 'Second release should return false (no lock held)');

    const result3 = lock.release();
    t.false(result3, 'Third release should return false (no lock held)');

  } finally {
    cleanupTestDir(testDir);
  }
});

// Test 11: CleanupAll removes stale locks but not active locks
test.serial('cleanupAll removes stale locks but not active locks', async (t) => {
  const testDir = createTestDir();
  const lockDir = path.join(testDir, 'locks');

  try {
    // Create stale lock (non-running PID)
    createStaleLock(lockDir, 'stale.json', 999999);

    // Create active lock (current PID)
    createStaleLock(lockDir, 'active.json', process.pid);

    const lock = new FileLock({ lockDir });

    // Run cleanup
    const cleaned = lock.cleanupAll();

    // Should clean up stale lock files (2 files: lock + pid)
    // But not active lock files
    t.is(cleaned, 2, 'Should clean up stale lock files only');

    // Verify stale lock is gone
    t.false(fs.existsSync(path.join(lockDir, 'stale.json.lock')));
    t.false(fs.existsSync(path.join(lockDir, 'stale.json.lock.pid')));

    // Verify active lock still exists
    t.true(fs.existsSync(path.join(lockDir, 'active.json.lock')));
    t.true(fs.existsSync(path.join(lockDir, 'active.json.lock.pid')));

  } finally {
    cleanupTestDir(testDir);
  }
});
