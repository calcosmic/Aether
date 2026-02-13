#!/usr/bin/env node
/**
 * Structured Logging Module
 *
 * Provides consistent logging to activity.log with proper formatting
 * and error handling. All logging operations fail silently to avoid
 * cascading errors.
 */

const fs = require('fs');
const path = require('path');

// Log level emojis
const EMOJI = {
  ERROR: '❌',
  WARN: '⚠️',
  INFO: 'ℹ️',
  SUCCESS: '✓',
};

// Caste emojis (matching cli.js pattern)
const CASTE_EMOJI = {
  queen: '👑',
  scout: '🔍',
  builder: '🔨',
  watcher: '👁️',
  chaos: '🎲',
  ant: '🐜',
};

/**
 * Get the path to activity.log
 * @returns {string} Path to activity.log
 */
function getActivityLogPath() {
  const home = process.env.HOME || process.env.USERPROFILE;
  if (!home) {
    return null;
  }
  return path.join(home, '.aether', 'data', 'activity.log');
}

/**
 * Format a timestamp as HH:MM:SS
 * @param {string|Date} [timestamp] - ISO string or Date object
 * @returns {string} Formatted time string
 */
function formatTimestamp(timestamp) {
  const date = timestamp ? new Date(timestamp) : new Date();
  return date.toISOString().split('T')[1].slice(0, 8); // HH:MM:SS
}

/**
 * Sanitize a string for logging
 * - Removes newlines
 * - Trims whitespace
 * - Limits to 200 characters
 * - Escapes control characters
 * @param {string} str - String to sanitize
 * @returns {string} Sanitized string
 */
function sanitizeForLog(str) {
  if (typeof str !== 'string') {
    str = String(str);
  }
  return str
    .replace(/[\n\r]/g, ' ')
    .replace(/[\x00-\x1F\x7F]/g, '')
    .trim()
    .slice(0, 200);
}

/**
 * Append a line to activity.log
 * Fails silently if logging fails
 * @param {string} line - Line to append
 * @returns {boolean} True if logged successfully, false otherwise
 */
function appendToLog(line) {
  try {
    const logPath = getActivityLogPath();
    if (!logPath) {
      return false;
    }

    // Ensure directory exists
    const logDir = path.dirname(logPath);
    if (!fs.existsSync(logDir)) {
      fs.mkdirSync(logDir, { recursive: true });
    }

    fs.appendFileSync(logPath, line + '\n');
    return true;
  } catch {
    // Silent fail - don't cascade errors from logging
    return false;
  }
}

/**
 * Log an error to activity.log
 * @param {Error|object} error - Error to log (AetherError or plain Error)
 * @returns {boolean} True if logged successfully
 */
function logError(error) {
  try {
    const timestamp = formatTimestamp(error.timestamp);
    const emoji = EMOJI.ERROR;

    // Extract code and message
    let code = 'E_UNKNOWN';
    let message = 'Unknown error';

    if (error.code) {
      code = error.code;
      message = error.message;
    } else if (error.message) {
      message = error.message;
    }

    const sanitizedMessage = sanitizeForLog(message);
    const logLine = `[${timestamp}] ${emoji} ERROR ${code}: ${sanitizedMessage}`;

    return appendToLog(logLine);
  } catch {
    return false;
  }
}

/**
 * Log an activity/event to activity.log
 * @param {string} action - Action name (e.g., 'SPAWN', 'COMPLETED')
 * @param {string} caste - Caste name (e.g., 'queen', 'scout', 'builder')
 * @param {string} description - Activity description
 * @returns {boolean} True if logged successfully
 */
function logActivity(action, caste, description) {
  try {
    const timestamp = formatTimestamp();
    const emoji = CASTE_EMOJI[caste] || CASTE_EMOJI.ant;
    const sanitizedDescription = sanitizeForLog(description);

    const logLine = `[${timestamp}] ${emoji} ${action} ${caste}: ${sanitizedDescription}`;

    return appendToLog(logLine);
  } catch {
    return false;
  }
}

/**
 * Log a warning to activity.log
 * @param {string} code - Warning code (e.g., 'W_CONFIG_MISSING')
 * @param {string} message - Warning message
 * @returns {boolean} True if logged successfully
 */
function logWarning(code, message) {
  try {
    const timestamp = formatTimestamp();
    const emoji = EMOJI.WARN;
    const sanitizedMessage = sanitizeForLog(message);

    const logLine = `[${timestamp}] ${emoji} WARN ${code}: ${sanitizedMessage}`;

    return appendToLog(logLine);
  } catch {
    return false;
  }
}

/**
 * Log an info message to activity.log
 * @param {string} message - Info message
 * @returns {boolean} True if logged successfully
 */
function logInfo(message) {
  try {
    const timestamp = formatTimestamp();
    const emoji = EMOJI.INFO;
    const sanitizedMessage = sanitizeForLog(message);

    const logLine = `[${timestamp}] ${emoji} ${sanitizedMessage}`;

    return appendToLog(logLine);
  } catch {
    return false;
  }
}

/**
 * Log a success message to activity.log
 * @param {string} caste - Caste name
 * @param {string} description - Success description
 * @returns {boolean} True if logged successfully
 */
function logSuccess(caste, description) {
  try {
    const timestamp = formatTimestamp();
    const emoji = EMOJI.SUCCESS;
    const sanitizedDescription = sanitizeForLog(description);

    const logLine = `[${timestamp}] ${emoji} ${caste}: ${sanitizedDescription}`;

    return appendToLog(logLine);
  } catch {
    return false;
  }
}

/**
 * Get recent log entries
 * @param {number} [lines=10] - Number of lines to retrieve
 * @returns {string[]} Array of log lines
 */
function getRecentLogs(lines = 10) {
  try {
    const logPath = getActivityLogPath();
    if (!logPath || !fs.existsSync(logPath)) {
      return [];
    }

    const content = fs.readFileSync(logPath, 'utf8');
    const allLines = content.split('\n').filter(Boolean);
    return allLines.slice(-lines);
  } catch {
    return [];
  }
}

module.exports = {
  EMOJI,
  CASTE_EMOJI,
  getActivityLogPath,
  formatTimestamp,
  sanitizeForLog,
  logError,
  logActivity,
  logWarning,
  logInfo,
  logSuccess,
  getRecentLogs,
};
