#!/usr/bin/env node
/**
 * FileLock - PID-based file locking with stale detection
 *
 * Implements exclusive atomic locks for safe concurrent access to shared resources.
 * Based on the pattern from .aether/utils/file-lock.sh for use in the CLI.
 *
 * @module bin/lib/file-lock
 */

const fs = require('fs');
const path = require('path');
const { FileSystemError, ConfigurationError } = require('./errors');

/**
 * Default configuration for lock behavior
 */
const DEFAULT_OPTIONS = {
  lockDir: '.aether/locks',
  timeout: 5000,        // 5 seconds total timeout
  retryInterval: 50,    // 50ms between retries
  maxRetries: 100,      // Total 5 seconds max wait
  maxLockAge: 5 * 60 * 1000,  // 5 minutes - configurable stale lock threshold
};

/**
 * Module-level set of registered cleanup functions (PLAN-006 fix #4)
 * Prevents duplicate cleanup handler registration across multiple FileLock instances
 */
const registeredCleanups = new Set();

/**
 * FileLock class for exclusive file locking with stale detection
 *
 * Enables safe concurrent access to COLONY_STATE.json by multiple processes,
 * preventing race conditions during phase transitions.
 */
class FileLock {
  /**
   * Create a FileLock instance
   *
   * @param {Object} options - Configuration options
   * @param {string} options.lockDir - Directory for lock files (default: '.aether/locks')
   * @param {number} options.timeout - Total timeout in milliseconds (default: 5000)
   * @param {number} options.retryInterval - Milliseconds between retries (default: 50)
   * @param {number} options.maxRetries - Maximum retry attempts (default: 100)
   * @param {number} options.maxLockAge - Maximum lock age in ms before considered stale (default: 300000)
   */
  constructor(options = {}) {
    this.options = { ...DEFAULT_OPTIONS, ...options };

    // Validate lockDir (PLAN-006 fix #2)
    if (!this.options.lockDir || typeof this.options.lockDir !== 'string') {
      throw new ConfigurationError(
        'lockDir must be a non-empty string',
        { lockDir: this.options.lockDir }
      );
    }

    // Validate timeout (PLAN-006 fix #5)
    if (typeof this.options.timeout !== 'number' || this.options.timeout < 0) {
      throw new ConfigurationError(
        'timeout must be a non-negative number',
        { timeout: this.options.timeout }
      );
    }

    // Validate retryInterval
    if (typeof this.options.retryInterval !== 'number' || this.options.retryInterval < 0) {
      throw new ConfigurationError(
        'retryInterval must be a non-negative number',
        { retryInterval: this.options.retryInterval }
      );
    }

    // Validate maxRetries
    if (typeof this.options.maxRetries !== 'number' || this.options.maxRetries < 0) {
      throw new ConfigurationError(
        'maxRetries must be a non-negative number',
        { maxRetries: this.options.maxRetries }
      );
    }

    // Validate maxLockAge (PLAN-007 Fix 1)
    if (typeof this.options.maxLockAge !== 'number' || this.options.maxLockAge < 0) {
      throw new ConfigurationError(
        'maxLockAge must be a non-negative number',
        { maxLockAge: this.options.maxLockAge }
      );
    }

    this.currentLock = null;
    this.currentPidFile = null;

    // Ensure lock directory exists
    this._ensureLockDir();

    // Register cleanup handlers
    this._registerCleanupHandlers();
  }

  /**
   * Ensure the lock directory exists
   * @private
   */
  _ensureLockDir() {
    try {
      if (!fs.existsSync(this.options.lockDir)) {
        fs.mkdirSync(this.options.lockDir, { recursive: true });
      }
    } catch (error) {
      throw new FileSystemError(
        `Failed to create lock directory: ${this.options.lockDir}`,
        { error: error.message, code: error.code }
      );
    }
  }

  /**
   * Register process cleanup handlers to ensure locks are released on exit
   * Uses module-level tracking to prevent duplicate registrations (PLAN-006 fix #4)
   * @private
   */
  _registerCleanupHandlers() {
    // Create unique identifier for this cleanup based on lock directory
    const cleanupId = `filelock-${this.options.lockDir}`;

    // Only register if not already registered
    if (registeredCleanups.has(cleanupId)) {
      return;
    }

    registeredCleanups.add(cleanupId);

    const cleanup = () => {
      this.release();
    };

    // Register for various exit signals
    process.on('exit', cleanup);
    process.on('SIGINT', () => {
      cleanup();
      process.exit(130);
    });
    process.on('SIGTERM', () => {
      cleanup();
      process.exit(143);
    });

    // Handle uncaught exceptions
    process.on('uncaughtException', (error) => {
      cleanup();
      // Re-throw to allow default handling
      throw error;
    });

    // Handle unhandled promise rejections
    process.on('unhandledRejection', () => {
      cleanup();
    });
  }

  /**
   * Generate lock file paths for a given resource
   *
   * @param {string} filePath - Path to the resource to lock
   * @returns {Object} Object containing lockFile and pidFile paths
   * @private
   */
  _getLockPaths(filePath) {
    const baseName = path.basename(filePath);
    const lockFile = path.join(this.options.lockDir, `${baseName}.lock`);
    const pidFile = `${lockFile}.pid`;
    return { lockFile, pidFile };
  }

  /**
   * Check if a process with the given PID is running
   *
   * @param {number} pid - Process ID to check
   * @returns {boolean} True if process is running, false otherwise
   * @private
   */
  _isProcessRunning(pid) {
    try {
      // process.kill(pid, 0) checks if process exists without sending signal
      process.kill(pid, 0);
      return true;
    } catch {
      return false;
    }
  }

  /**
   * Safely unlink a file, ignoring ENOENT errors
   *
   * @param {string} filePath - Path to file to unlink
   * @private
   */
  _safeUnlink(filePath) {
    try {
      fs.unlinkSync(filePath);
    } catch (error) {
      if (error.code !== 'ENOENT') {
        // Log but don't throw - we're cleaning up
        console.warn(`Warning: Failed to clean up ${filePath}: ${error.message}`);
      }
    }
  }

  /**
   * Clean up stale lock files
   *
   * Checks both PID file and lock file for PID (handles crash scenarios
   * where only one file was created). Also checks lock age to handle
   * PID reuse race condition.
   *
   * @param {string} lockFile - Path to lock file
   * @param {string} pidFile - Path to PID file
   * @returns {boolean} True if lock was cleaned up, false if still held
   * @private
   */
  _cleanupStaleLock(lockFile, pidFile) {
    // Maximum lock age before considering stale (configurable, default 5 minutes)
    // This handles PID reuse race condition
    const maxLockAgeMs = this.options.maxLockAge;

    try {
      // Check lock file age first (handles PID reuse)
      if (fs.existsSync(lockFile)) {
        try {
          const stat = fs.statSync(lockFile);
          const lockAge = Date.now() - stat.mtimeMs;

          if (lockAge > maxLockAgeMs) {
            // Lock is old enough to be considered stale regardless of PID
            this._safeUnlink(lockFile);
            this._safeUnlink(pidFile);
            return true;
          }
        } catch {
          // Cannot stat, proceed with PID check
        }
      }

      let pid = null;

      // Try to read PID from PID file first
      if (fs.existsSync(pidFile)) {
        try {
          const pidData = fs.readFileSync(pidFile, 'utf8').trim();

          // Validate PID is a positive integer (PLAN-006 fix #3)
          if (!/^\d+$/.test(pidData)) {
            // PID file contains invalid data - clean it up
            this._safeUnlink(lockFile);
            this._safeUnlink(pidFile);
            return true;
          }

          pid = parseInt(pidData, 10);
        } catch (readError) {
          // PID file unreadable - will clean up
        }
      }

      // If no valid PID from PID file, try lock file itself
      if ((pid === null || isNaN(pid)) && fs.existsSync(lockFile)) {
        try {
          const lockData = fs.readFileSync(lockFile, 'utf8').trim();

          // Validate PID is a positive integer
          if (!/^\d+$/.test(lockData)) {
            // Lock file contains invalid data - clean it up
            this._safeUnlink(lockFile);
            this._safeUnlink(pidFile);
            return true;
          }

          pid = parseInt(lockData, 10);
        } catch (readError) {
          // Lock file unreadable - will clean up
        }
      }

      // Check if process is running
      if (!isNaN(pid) && this._isProcessRunning(pid)) {
        // Process is running, lock is valid
        return false;
      }

      // Either no valid PID or process not running - clean up stale lock
      this._safeUnlink(lockFile);
      this._safeUnlink(pidFile);

      return true;
    } catch (error) {
      if (error.code === 'ENOENT') {
        return true;
      }
      throw new FileSystemError(
        `Failed to clean up stale lock: ${lockFile}`,
        { error: error.message, code: error.code }
      );
    }
  }

  /**
   * Attempt to acquire a lock atomically
   *
   * Uses PID-file-first ordering for crash recovery:
   * 1. Write PID file first (if this fails, no lock file created)
   * 2. Create lock file atomically
   * 3. On failure, clean up both files
   *
   * @param {string} lockFile - Path to lock file
   * @param {string} pidFile - Path to PID file
   * @returns {boolean} True if lock acquired, false otherwise
   * @throws {FileSystemError} On unexpected filesystem errors
   * @private
   */
  _tryAcquire(lockFile, pidFile) {
    let pidFileCreated = false;
    let lockFileCreated = false;

    try {
      // Step 1: Write PID file first (easier to clean up if lock fails)
      try {
        fs.writeFileSync(pidFile, process.pid.toString(), 'utf8');
        pidFileCreated = true;
      } catch (pidError) {
        // Cannot write PID file - cannot proceed
        throw new FileSystemError(
          `Failed to write PID file: ${pidFile}`,
          { error: pidError.message, code: pidError.code }
        );
      }

      // Step 2: Create lock file atomically
      try {
        const fd = fs.openSync(lockFile, 'wx');
        lockFileCreated = true;

        try {
          // Write PID to lock file as well (for redundancy)
          fs.writeFileSync(fd, process.pid.toString(), 'utf8');
        } finally {
          fs.closeSync(fd);
        }
      } catch (lockError) {
        if (lockError.code === 'EEXIST') {
          // Lock file exists - clean up our PID file
          this._safeUnlink(pidFile);
          return false;
        }
        throw lockError;
      }

      // Step 3: Track current lock (only after both files created)
      this.currentLock = lockFile;
      this.currentPidFile = pidFile;

      return true;

    } catch (error) {
      // Clean up on any failure
      if (lockFileCreated) {
        this._safeUnlink(lockFile);
      }
      if (pidFileCreated) {
        this._safeUnlink(pidFile);
      }

      if (error instanceof FileSystemError) {
        throw error;
      }

      throw new FileSystemError(
        `Failed to acquire lock: ${lockFile}`,
        { error: error.message, code: error.code }
      );
    }
  }

  /**
   * Acquire an exclusive lock on a file (SYNCHRONOUS - may block event loop)
   *
   * WARNING: This method uses a busy-wait loop that blocks the Node.js event loop
   * during retry intervals. For non-blocking operation, use acquireAsync() instead.
   *
   * @param {string} filePath - Path to the file to lock
   * @returns {boolean} True if lock acquired, false on timeout
   * @throws {FileSystemError} On unexpected filesystem errors
   */
  acquire(filePath) {
    const { lockFile, pidFile } = this._getLockPaths(filePath);

    // Check for existing lock and handle stale locks
    if (fs.existsSync(lockFile)) {
      const cleaned = this._cleanupStaleLock(lockFile, pidFile);

      if (!cleaned) {
        // Lock is held by a running process, need to retry
      }
    }

    // Try to acquire lock with retries
    let retryCount = 0;
    const startTime = Date.now();

    while (retryCount < this.options.maxRetries) {
      // Try to acquire the lock
      if (this._tryAcquire(lockFile, pidFile)) {
        return true;
      }

      // Check timeout
      if (Date.now() - startTime >= this.options.timeout) {
        return false;
      }

      // Wait before retry
      retryCount++;
      if (retryCount < this.options.maxRetries) {
        // Simple synchronous delay using busy-wait
        // In practice, this should be rare as locks are short-lived
        const delayStart = Date.now();
        while (Date.now() - delayStart < this.options.retryInterval) {
          // Busy wait for precise timing
        }
      }
    }

    return false;
  }

  /**
   * Release the current lock
   *
   * This method is idempotent - safe to call multiple times.
   *
   * @returns {boolean} True if lock was fully released, false if no lock was held
   *                    or if deletion failed (check console.warn for details)
   */
  release() {
    if (!this.currentLock) {
      return false;
    }

    let success = true;

    // Delete lock file
    try {
      if (fs.existsSync(this.currentLock)) {
        fs.unlinkSync(this.currentLock);
      }
    } catch (error) {
      if (error.code !== 'ENOENT') {
        // Log but don't throw - we're cleaning up
        console.warn(`Warning: Failed to remove lock file: ${error.message}`);
        success = false;
      }
    }

    // Delete PID file
    if (this.currentPidFile) {
      try {
        if (fs.existsSync(this.currentPidFile)) {
          fs.unlinkSync(this.currentPidFile);
        }
      } catch (error) {
        if (error.code !== 'ENOENT') {
          console.warn(`Warning: Failed to remove PID file: ${error.message}`);
          success = false;
        }
      }
    }

    // Clear state
    this.currentLock = null;
    this.currentPidFile = null;

    return success;
  }

  /**
   * Check if a file is currently locked
   *
   * @param {string} filePath - Path to check
   * @returns {boolean} True if locked, false otherwise
   */
  isLocked(filePath) {
    const { lockFile } = this._getLockPaths(filePath);
    return fs.existsSync(lockFile);
  }

  /**
   * Get the PID of the process holding a lock
   *
   * @param {string} filePath - Path to check
   * @returns {number|null} PID of lock holder, or null if not locked
   */
  getLockHolder(filePath) {
    const { pidFile } = this._getLockPaths(filePath);

    try {
      if (!fs.existsSync(pidFile)) {
        return null;
      }

      const pidData = fs.readFileSync(pidFile, 'utf8').trim();
      const pid = parseInt(pidData, 10);

      return isNaN(pid) ? null : pid;
    } catch {
      return null;
    }
  }

  /**
   * Wait for a lock to be released (SYNCHRONOUS - may block event loop)
   *
   * WARNING: This method uses a busy-wait loop that blocks the Node.js event loop.
   * For non-blocking operation, use waitForLockAsync() instead.
   *
   * @param {string} filePath - Path to wait for
   * @param {number} maxWait - Maximum milliseconds to wait (default: timeout option)
   * @returns {boolean} True if lock was released, false on timeout
   */
  waitForLock(filePath, maxWait = null) {
    const waitTime = maxWait || this.options.timeout;
    const startTime = Date.now();

    while (this.isLocked(filePath)) {
      if (Date.now() - startTime >= waitTime) {
        return false;
      }

      // Small delay between checks
      const delayStart = Date.now();
      while (Date.now() - delayStart < 10) {
        // 10ms busy-wait
      }
    }

    return true;
  }

  /**
   * Acquire an exclusive lock asynchronously (yields to event loop)
   *
   * This is the non-blocking version of acquire(). It uses setTimeout for
   * delays, allowing other async operations to run during wait periods.
   *
   * @param {string} filePath - Path to the file to lock
   * @returns {Promise<boolean>} True if lock acquired, false on timeout
   * @throws {FileSystemError} On unexpected filesystem errors
   */
  async acquireAsync(filePath) {
    const { lockFile, pidFile } = this._getLockPaths(filePath);

    // Check for existing lock and handle stale locks
    if (fs.existsSync(lockFile)) {
      this._cleanupStaleLock(lockFile, pidFile);
      // Lock is held by running process if not cleaned
    }

    // Try to acquire lock with retries
    let retryCount = 0;
    const startTime = Date.now();

    while (retryCount < this.options.maxRetries) {
      // Try to acquire the lock
      if (this._tryAcquire(lockFile, pidFile)) {
        return true;
      }

      // Check timeout
      if (Date.now() - startTime >= this.options.timeout) {
        return false;
      }

      // Wait before retry (ASYNC - yields to event loop)
      retryCount++;
      if (retryCount < this.options.maxRetries) {
        await new Promise(resolve =>
          setTimeout(resolve, this.options.retryInterval)
        );
      }
    }

    return false;
  }

  /**
   * Wait for a lock to be released asynchronously (yields to event loop)
   *
   * This is the non-blocking version of waitForLock(). It uses setTimeout
   * for delays, allowing other async operations to run during wait periods.
   *
   * @param {string} filePath - Path to wait for
   * @param {number} maxWait - Maximum milliseconds to wait (default: timeout option)
   * @returns {Promise<boolean>} True if lock was released, false on timeout
   */
  async waitForLockAsync(filePath, maxWait = null) {
    const waitTime = maxWait || this.options.timeout;
    const startTime = Date.now();

    while (this.isLocked(filePath)) {
      if (Date.now() - startTime >= waitTime) {
        return false;
      }

      // Small async delay (yields to event loop)
      await new Promise(resolve => setTimeout(resolve, 10));
    }

    return true;
  }

  /**
   * Force cleanup of all locks in the lock directory
   *
   * Use with caution - only for emergency cleanup.
   *
   * @returns {number} Number of locks cleaned up
   */
  cleanupAll() {
    let cleaned = 0;

    try {
      if (!fs.existsSync(this.options.lockDir)) {
        return 0;
      }

      const files = fs.readdirSync(this.options.lockDir);

      // First pass: identify which locks are held by running processes
      const runningLocks = new Set();
      for (const file of files) {
        if (file.endsWith('.lock')) {
          const filePath = path.join(this.options.lockDir, file);
          const pidFile = `${filePath}.pid`;

          try {
            if (fs.existsSync(pidFile)) {
              const pidData = fs.readFileSync(pidFile, 'utf8').trim();
              const pid = parseInt(pidData, 10);

              if (!isNaN(pid) && this._isProcessRunning(pid)) {
                // Process is still running, mark both files to skip
                runningLocks.add(file);
                runningLocks.add(`${file}.pid`);
              }
            }
          } catch {
            // Error reading PID file, treat as stale
          }
        }
      }

      // Second pass: clean up stale locks
      for (const file of files) {
        if (file.endsWith('.lock') || file.endsWith('.lock.pid')) {
          // Skip if held by running process
          if (runningLocks.has(file)) {
            continue;
          }

          const filePath = path.join(this.options.lockDir, file);

          try {
            fs.unlinkSync(filePath);
            cleaned++;
          } catch (error) {
            if (error.code !== 'ENOENT') {
              console.warn(`Warning: Failed to clean up ${filePath}: ${error.message}`);
            }
          }
        }
      }
    } catch (error) {
      if (error.code !== 'ENOENT') {
        throw new FileSystemError(
          `Failed to cleanup locks in ${this.options.lockDir}`,
          { error: error.message, code: error.code }
        );
      }
    }

    return cleaned;
  }
}

module.exports = { FileLock };
