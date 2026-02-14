#!/usr/bin/env node
/**
 * State Guard Module
 *
 * Enforces the Iron Law: phase advancement requires fresh verification evidence.
 * Provides StateGuard class with idempotency checks, file locking, and structured errors.
 *
 * @module bin/lib/state-guard
 */

const fs = require('fs');
const path = require('path');

/**
 * Error codes for StateGuard errors
 */
const StateGuardErrorCodes = {
  E_IRON_LAW_VIOLATION: 'E_IRON_LAW_VIOLATION',
  E_IDEMPOTENCY_CHECK: 'E_IDEMPOTENCY_CHECK',
  E_LOCK_TIMEOUT: 'E_LOCK_TIMEOUT',
  E_INVALID_TRANSITION: 'E_INVALID_TRANSITION',
  E_STATE_NOT_FOUND: 'E_STATE_NOT_FOUND',
  E_STATE_INVALID: 'E_STATE_INVALID',
};

/**
 * StateGuardError - Structured error for state guard violations
 */
class StateGuardError extends Error {
  /**
   * @param {string} code - Error code from StateGuardErrorCodes
   * @param {string} message - Human-readable error message
   * @param {object} details - Additional error context
   * @param {string|null} recovery - Recovery suggestion for user
   */
  constructor(code, message, details = {}, recovery = null) {
    super(message);
    this.name = 'StateGuardError';
    this.code = code;
    this.details = details;
    this.recovery = recovery;
    this.timestamp = new Date().toISOString();

    // Maintain proper stack trace in V8 environments
    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, StateGuardError);
    }
  }

  /**
   * Convert error to structured JSON object
   * @returns {object} Structured error representation
   */
  toJSON() {
    return {
      error: {
        name: this.name,
        code: this.code,
        message: this.message,
        details: this.details,
        recovery: this.recovery,
        timestamp: this.timestamp,
      },
    };
  }

  /**
   * Convert error to string for console output
   * @returns {string} Formatted error string
   */
  toString() {
    let str = `${this.code}: ${this.message}`;
    if (this.recovery) {
      str += `\n  Recovery: ${this.recovery}`;
    }
    return str;
  }
}

/**
 * FileLock - PID-based file locking with stale lock detection
 */
class FileLock {
  /**
   * @param {string} lockDir - Directory for lock files (default: .aether/locks)
   * @param {object} options - Lock configuration options
   */
  constructor(lockDir = '.aether/locks', options = {}) {
    this.lockDir = lockDir;
    this.lockTimeout = options.lockTimeout || 300000; // 5 minutes
    this.retryInterval = options.retryInterval || 500;  // 500ms
    this.maxRetries = options.maxRetries || 100;     // 50 seconds max wait
    this.currentLock = null;
    this.currentPidFile = null;
  }

  /**
   * Acquire lock on a file
   * @param {string} filePath - Path to file to lock
   * @returns {Promise<boolean>} True if lock acquired, false otherwise
   */
  async acquire(filePath) {
    const lockFile = path.join(this.lockDir, `${path.basename(filePath)}.lock`);
    const pidFile = `${lockFile}.pid`;

    // Ensure lock directory exists
    if (!fs.existsSync(this.lockDir)) {
      fs.mkdirSync(this.lockDir, { recursive: true });
    }

    // Check for stale lock
    if (fs.existsSync(lockFile)) {
      const lockPid = this.readPidFile(pidFile);
      if (lockPid && !this.isProcessRunning(lockPid)) {
        console.log(`Lock stale (PID ${lockPid} not running), cleaning up...`);
        this.cleanupLock(lockFile, pidFile);
      }
    }

    // Try to acquire with retry
    for (let retry = 0; retry < this.maxRetries; retry++) {
      try {
        // Atomic lock creation using exclusive flag
        const fd = fs.openSync(lockFile, 'wx');
        fs.writeSync(fd, process.pid.toString());
        fs.closeSync(fd);

        // Write PID file
        fs.writeFileSync(pidFile, process.pid.toString());

        this.currentLock = lockFile;
        this.currentPidFile = pidFile;
        return true;
      } catch (err) {
        if (err.code !== 'EEXIST') throw err;

        // Wait before retry
        if (retry < this.maxRetries - 1) {
          await this.sleep(this.retryInterval);
        }
      }
    }

    return false;
  }

  /**
   * Release the current lock
   */
  release() {
    if (this.currentLock) {
      this.cleanupLock(this.currentLock, this.currentPidFile);
      this.currentLock = null;
      this.currentPidFile = null;
    }
  }

  /**
   * Check if a process is running
   * @param {string} pid - Process ID to check
   * @returns {boolean} True if process is running
   */
  isProcessRunning(pid) {
    try {
      process.kill(parseInt(pid), 0);
      return true;
    } catch {
      return false;
    }
  }

  /**
   * Read PID from PID file
   * @param {string} pidFile - Path to PID file
   * @returns {string|null} PID or null if file doesn't exist
   */
  readPidFile(pidFile) {
    try {
      return fs.readFileSync(pidFile, 'utf8').trim();
    } catch {
      return null;
    }
  }

  /**
   * Clean up lock files
   * @param {string} lockFile - Path to lock file
   * @param {string} pidFile - Path to PID file
   */
  cleanupLock(lockFile, pidFile) {
    try { fs.unlinkSync(lockFile); } catch {}
    try { fs.unlinkSync(pidFile); } catch {}
  }

  /**
   * Sleep for specified milliseconds
   * @param {number} ms - Milliseconds to sleep
   * @returns {Promise<void>}
   */
  sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}

/**
 * StateGuard - Enforces Iron Law and manages phase transitions
 */
class StateGuard {
  /**
   * @param {string} stateFilePath - Path to COLONY_STATE.json
   * @param {object} options - Configuration options
   */
  constructor(stateFilePath, options = {}) {
    this.stateFile = stateFilePath;
    this.lock = options.lock || new FileLock(options.lockDir);
    this.locked = false;
  }

  /**
   * Acquire lock on state file
   * @returns {Promise<void>}
   * @throws {StateGuardError} If lock cannot be acquired
   */
  async acquireLock() {
    const acquired = await this.lock.acquire(this.stateFile);
    if (!acquired) {
      throw new StateGuardError(
        StateGuardErrorCodes.E_LOCK_TIMEOUT,
        `Could not acquire lock on ${this.stateFile} after ${this.lock.maxRetries} retries`,
        { lockFile: `${this.stateFile}.lock`, maxRetries: this.lock.maxRetries },
        'Check for stuck processes or manually remove stale lock file'
      );
    }
    this.locked = true;
  }

  /**
   * Release lock on state file
   * Safe to call even if not locked
   */
  releaseLock() {
    this.lock.release();
    this.locked = false;
  }

  /**
   * Load and parse state file
   * @returns {object} Parsed state object
   * @throws {StateGuardError} If file missing or invalid
   */
  loadState() {
    // Check if file exists
    if (!fs.existsSync(this.stateFile)) {
      throw new StateGuardError(
        StateGuardErrorCodes.E_STATE_NOT_FOUND,
        `State file not found: ${this.stateFile}`,
        { path: this.stateFile },
        'Run: aether init'
      );
    }

    // Read and parse
    let content;
    try {
      content = fs.readFileSync(this.stateFile, 'utf8');
    } catch (err) {
      throw new StateGuardError(
        StateGuardErrorCodes.E_STATE_INVALID,
        `Failed to read state file: ${err.message}`,
        { path: this.stateFile, error: err.message },
        'Check file permissions and disk space'
      );
    }

    let state;
    try {
      state = JSON.parse(content);
    } catch (err) {
      throw new StateGuardError(
        StateGuardErrorCodes.E_STATE_INVALID,
        `Invalid JSON in state file: ${err.message}`,
        { path: this.stateFile, error: err.message },
        'Restore from backup or reinitialize'
      );
    }

    // Validate basic structure
    if (!state.version || typeof state.current_phase !== 'number' || !Array.isArray(state.events)) {
      throw new StateGuardError(
        StateGuardErrorCodes.E_STATE_INVALID,
        'State file missing required fields: version, current_phase, events',
        { path: this.stateFile, hasVersion: !!state.version, hasPhase: typeof state.current_phase === 'number', hasEvents: Array.isArray(state.events) },
        'Restore from backup or reinitialize'
      );
    }

    return state;
  }

  /**
   * Save state atomically (write to temp, then rename)
   * @param {object} state - State object to save
   */
  saveState(state) {
    // Update timestamp
    state.last_updated = new Date().toISOString();

    // Write to temp file first (atomic write)
    const tempFile = `${this.stateFile}.tmp`;
    fs.writeFileSync(tempFile, JSON.stringify(state, null, 2), 'utf8');

    // Atomic rename
    fs.renameSync(tempFile, this.stateFile);
  }
}

module.exports = {
  StateGuard,
  StateGuardError,
  StateGuardErrorCodes,
  FileLock,
};
