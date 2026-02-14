#!/usr/bin/env node
/**
 * Event Types Module
 *
 * Defines standardized event types for audit trail in COLONY_STATE.json.
 * Provides validation and creation helpers for event sourcing.
 *
 * @module bin/lib/event-types
 */

/**
 * Standardized event types for colony audit trail
 * @readonly
 * @enum {string}
 */
const EventTypes = {
  /** Phase transition from one phase to another */
  PHASE_TRANSITION: 'phase_transition',
  /** Phase build process started */
  PHASE_BUILD_STARTED: 'phase_build_started',
  /** Phase build process completed successfully */
  PHASE_BUILD_COMPLETED: 'phase_build_completed',
  /** Phase was rolled back to previous state */
  PHASE_ROLLED_BACK: 'phase_rolled_back',
  /** Checkpoint was created */
  CHECKPOINT_CREATED: 'checkpoint_created',
  /** Checkpoint was restored */
  CHECKPOINT_RESTORED: 'checkpoint_restored',
  /** Update process started */
  UPDATE_STARTED: 'update_started',
  /** Update process completed successfully */
  UPDATE_COMPLETED: 'update_completed',
  /** Update process failed */
  UPDATE_FAILED: 'update_failed',
  /** Iron Law violation detected */
  IRON_LAW_VIOLATION: 'iron_law_violation',
};

/**
 * Valid event type values for quick lookup
 * @type {string[]}
 */
const VALID_EVENT_TYPES = Object.values(EventTypes);

/**
 * ISO 8601 timestamp regex for validation
 * Matches: 2026-02-14T14:30:22.123Z or 2026-02-14T14:30:22Z
 * @type {RegExp}
 */
const ISO8601_REGEX = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d{3})?Z$/;

/**
 * Validate an event object has all required fields and valid values
 *
 * @param {object} event - Event object to validate
 * @returns {object} Validation result with valid flag and errors array
 * @returns {boolean} result.valid - True if event is valid
 * @returns {string[]} result.errors - Array of error messages if invalid
 */
function validateEvent(event) {
  const errors = [];

  // Check event is an object
  if (!event || typeof event !== 'object' || Array.isArray(event)) {
    return {
      valid: false,
      errors: ['Event must be an object']
    };
  }

  // Required fields
  const requiredFields = ['timestamp', 'type', 'worker', 'details'];
  for (const field of requiredFields) {
    if (!(field in event)) {
      errors.push(`Missing required field: ${field}`);
    }
  }

  // If missing required fields, return early
  if (errors.length > 0) {
    return { valid: false, errors };
  }

  // Validate timestamp format (ISO 8601)
  if (typeof event.timestamp !== 'string') {
    errors.push('timestamp must be a string');
  } else if (!ISO8601_REGEX.test(event.timestamp)) {
    errors.push('timestamp must be valid ISO 8601 format (e.g., 2026-02-14T14:30:22Z)');
  } else {
    // Also validate it's a valid date
    const date = new Date(event.timestamp);
    if (isNaN(date.getTime())) {
      errors.push('timestamp must be a valid date');
    }
  }

  // Validate type is a valid EventType
  if (typeof event.type !== 'string') {
    errors.push('type must be a string');
  } else if (!VALID_EVENT_TYPES.includes(event.type)) {
    errors.push(`type must be a valid EventType: ${VALID_EVENT_TYPES.join(', ')}`);
  }

  // Validate worker is a non-empty string
  if (typeof event.worker !== 'string') {
    errors.push('worker must be a string');
  } else if (event.worker.trim().length === 0) {
    errors.push('worker must not be empty');
  }

  // Validate details is an object
  if (typeof event.details !== 'object' || event.details === null) {
    errors.push('details must be an object');
  } else if (Array.isArray(event.details)) {
    errors.push('details must be an object, not an array');
  }

  return {
    valid: errors.length === 0,
    errors
  };
}

/**
 * Create a new event object with validation
 *
 * @param {string} type - Event type from EventTypes
 * @param {string} worker - Worker/agent name
 * @param {object} [details={}] - Event-specific details
 * @returns {object} Created event object
 * @returns {string} result.timestamp - ISO 8601 timestamp
 * @returns {string} result.type - Event type
 * @returns {string} result.worker - Worker name
 * @returns {object} result.details - Event details
 * @throws {Error} If type is not a valid EventType
 */
function createEvent(type, worker, details = {}) {
  // Validate type
  if (!VALID_EVENT_TYPES.includes(type)) {
    throw new Error(
      `Invalid event type: "${type}". Must be one of: ${VALID_EVENT_TYPES.join(', ')}`
    );
  }

  // Get worker name from parameter, environment, or default
  const workerName = worker || process.env.WORKER_NAME || 'unknown';

  // Create event object
  const event = {
    timestamp: new Date().toISOString(),
    type,
    worker: workerName,
    details: details || {}
  };

  // Validate the created event
  const validation = validateEvent(event);
  if (!validation.valid) {
    throw new Error(`Created event failed validation: ${validation.errors.join(', ')}`);
  }

  return event;
}

/**
 * Check if a string is a valid event type
 *
 * @param {string} type - Type to check
 * @returns {boolean} True if valid event type
 */
function isValidEventType(type) {
  return typeof type === 'string' && VALID_EVENT_TYPES.includes(type);
}

/**
 * Get all valid event types
 *
 * @returns {string[]} Array of valid event type strings
 */
function getValidEventTypes() {
  return [...VALID_EVENT_TYPES];
}

module.exports = {
  EventTypes,
  validateEvent,
  createEvent,
  isValidEventType,
  getValidEventTypes,
};
