#!/usr/bin/env node
/**
 * Aether Error Class Hierarchy
 *
 * Centralized error handling for the Aether Colony CLI.
 * Provides structured JSON output, error codes, and recovery suggestions.
 */

/**
 * Error codes enum - categorized by error type
 */
const ErrorCodes = {
  // System errors (1-99)
  E_HUB_NOT_FOUND: 'E_HUB_NOT_FOUND',
  E_REPO_NOT_INITIALIZED: 'E_REPO_NOT_INITIALIZED',
  E_FILE_SYSTEM: 'E_FILE_SYSTEM',
  E_GIT_ERROR: 'E_GIT_ERROR',

  // Validation errors (100-199)
  E_INVALID_STATE: 'E_INVALID_STATE',
  E_MANIFEST_INVALID: 'E_MANIFEST_INVALID',
  E_JSON_PARSE: 'E_JSON_PARSE',

  // Runtime errors (200-299)
  E_UPDATE_FAILED: 'E_UPDATE_FAILED',
  E_LOCK_TIMEOUT: 'E_LOCK_TIMEOUT',
  E_ATOMIC_WRITE_FAILED: 'E_ATOMIC_WRITE_FAILED',

  // Unexpected errors (300-399)
  E_UNEXPECTED: 'E_UNEXPECTED',
  E_UNCAUGHT_EXCEPTION: 'E_UNCAUGHT_EXCEPTION',
  E_UNHANDLED_REJECTION: 'E_UNHANDLED_REJECTION',

  // Configuration errors (400-499)
  E_CONFIG: 'E_CONFIG',
};

/**
 * Base AetherError class
 * All application errors extend this class for consistent handling
 */
class AetherError extends Error {
  /**
   * @param {string} code - Error code from ErrorCodes
   * @param {string} message - Human-readable error message
   * @param {object} details - Additional error context
   * @param {string|null} recovery - Recovery suggestion for user
   */
  constructor(code, message, details = {}, recovery = null) {
    super(message);
    this.name = 'AetherError';
    this.code = code;
    this.details = details;
    this.recovery = recovery;
    this.timestamp = new Date().toISOString();

    // Maintain proper stack trace in V8 environments
    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, AetherError);
    }
  }

  /**
   * Convert error to structured JSON object
   * @returns {object} Structured error representation
   */
  toJSON() {
    return {
      error: {
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
 * HubError - Hub-related errors (distribution hub not found, corrupted, etc.)
 */
class HubError extends AetherError {
  constructor(message, details = {}) {
    super(
      ErrorCodes.E_HUB_NOT_FOUND,
      message,
      details,
      'Run: aether install'
    );
    this.name = 'HubError';
  }
}

/**
 * RepoError - Repository initialization errors
 */
class RepoError extends AetherError {
  constructor(message, details = {}) {
    super(
      ErrorCodes.E_REPO_NOT_INITIALIZED,
      message,
      details,
      'Run /ant:init in this repo first'
    );
    this.name = 'RepoError';
  }
}

/**
 * GitError - Git operation errors
 */
class GitError extends AetherError {
  constructor(message, details = {}) {
    super(
      ErrorCodes.E_GIT_ERROR,
      message,
      details,
      'Check git status and resolve conflicts'
    );
    this.name = 'GitError';
  }
}

/**
 * ValidationError - State validation errors
 */
class ValidationError extends AetherError {
  constructor(message, details = {}) {
    super(
      ErrorCodes.E_INVALID_STATE,
      message,
      details,
      'Check the state file and fix validation errors'
    );
    this.name = 'ValidationError';
  }
}

/**
 * FileSystemError - File operation errors
 */
class FileSystemError extends AetherError {
  constructor(message, details = {}) {
    super(
      ErrorCodes.E_FILE_SYSTEM,
      message,
      details,
      'Check file permissions and available disk space'
    );
    this.name = 'FileSystemError';
  }
}

/**
 * ConfigurationError - Environment/configuration errors
 */
class ConfigurationError extends AetherError {
  constructor(message, details = {}) {
    super(
      ErrorCodes.E_CONFIG,
      message,
      details,
      'Check environment variables and configuration'
    );
    this.name = 'ConfigurationError';
  }
}

/**
 * Map error codes to sysexits.h exit codes
 * @param {string} code - Error code
 * @returns {number} Exit code (0-255)
 */
function getExitCode(code) {
  switch (code) {
    case ErrorCodes.E_HUB_NOT_FOUND:
      return 69; // EX_UNAVAILABLE - service unavailable
    case ErrorCodes.E_REPO_NOT_INITIALIZED:
      return 78; // EX_CONFIG - configuration error
    case ErrorCodes.E_INVALID_STATE:
    case ErrorCodes.E_MANIFEST_INVALID:
    case ErrorCodes.E_JSON_PARSE:
      return 65; // EX_DATAERR - data format error
    case ErrorCodes.E_FILE_SYSTEM:
    case ErrorCodes.E_ATOMIC_WRITE_FAILED:
      return 74; // EX_IOERR - I/O error
    case ErrorCodes.E_GIT_ERROR:
      return 70; // EX_SOFTWARE - internal software error
    case ErrorCodes.E_LOCK_TIMEOUT:
      return 73; // EX_CANTCREAT - can't create (lock) file
    case ErrorCodes.E_CONFIG:
      return 78; // EX_CONFIG - configuration error
    default:
      return 1; // Generic error
  }
}

/**
 * Wrap a plain Error in an AetherError
 * @param {Error} error - Plain error to wrap
 * @returns {AetherError} Wrapped error
 */
function wrapError(error) {
  if (error instanceof AetherError) {
    return error;
  }
  return new AetherError(
    ErrorCodes.E_UNEXPECTED,
    error.message,
    { stack: error.stack, name: error.name },
    'Please report this issue with the error details'
  );
}

module.exports = {
  AetherError,
  HubError,
  RepoError,
  GitError,
  ValidationError,
  FileSystemError,
  ConfigurationError,
  ErrorCodes,
  getExitCode,
  wrapError,
};
