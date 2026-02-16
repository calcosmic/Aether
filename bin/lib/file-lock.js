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
const { FileSystemError } = require('./errors');

/**
 * Default configuration for lock behavior
 */
const DEFAULT_OPTIONS = {
  lockDir: '.aether/locks',
  timeout: 5000,        // 5 seconds total timeout
  retryInterval: 50,    // 50ms between retries
  maxRetries: 100,      // Total 5 seconds max wait
};

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
   */
  constructor(options = {}) {
    this.options = { ...DEFAULT_OPTIONS, ...options };
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
   * @private
   */
  _registerCleanupHandlers() {
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
   * Clean up stale lock files
   *
   * @param {string} lockFile - Path to lock file
   * @param {string} pidFile - Path to PID file
   * @private
   */
  _cleanupStaleLock(lockFile, pidFile) {
    try {
      // Try to read the PID from the pid file
      if (fs.existsSync(pidFile)) {
        const pidData = fs.readFileSync(pidFile, 'utf8').trim();
        const pid = parseInt(pidData, 10);

        if (!isNaN(pid)) {
          // Check if the process is still running
          if (this._isProcessRunning(pid)) {
            // Process is running, lock is not stale
            return false;
          }
        }
      }

      // Either no valid PID or process not running - clean up stale lock
      try {
        fs.unlinkSync(lockFile);
      } catch (error) {
        if (error.code !== 'ENOENT') {
          throw error;
        }
      }

      try {
        fs.unlinkSync(pidFile);
      } catch (error) {
        if (error.code !== 'ENOENT') {
          throw error;
        }
      }

      return true;
    } catch (error) {
      if (error.code === 'ENOENT') {
        // Lock file doesn't exist, consider it cleaned
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
   * @param {string} lockFile - Path to lock file
   * @param {string} pidFile - Path to PID file
   * @returns {boolean} True if lock acquired, false otherwise
   * @private
   */
  _tryAcquire(lockFile, pidFile) {
    try {
      // Attempt atomic creation using 'wx' flag (fails if file exists)
      const fd = fs.openSync(lockFile, 'wx');

      try {
        // Write current PID to lock file
        const pid = process.pid;
        fs.writeFileSync(fd, pid.toString(), 'utf8');
      } finally {
        fs.closeSync(fd);
      }

      // Also write to separate PID file for easy reading
      fs.writeFileSync(pidFile, process.pid.toString(), 'utf8');

      // Track current lock
      this.currentLock = lockFile;
      this.currentPidFile = pidFile;

      return true;
    } catch (error) {
      if (error.code === 'EEXIST') {
        // Lock file already exists
        return false;
      }

      // Unexpected error
      throw new FileSystemError(
        `Failed to acquire lock: ${lockFile}`,
        { error: error.message, code: error.code }
      );
    }
  }

  /**
   * Acquire an exclusive lock on a file
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
   * Wait for a lock to be released
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
