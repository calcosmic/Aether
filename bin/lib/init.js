#!/usr/bin/env node
/**
 * Initialization Module
 *
 * Handles new repo initialization with local state files.
 * Creates COLONY_STATE.json and required directory structure.
 *
 * @module bin/lib/init
 */

const fs = require('fs');
const path = require('path');

/**
 * Generate a unique session ID
 * @returns {string} Session ID in format session_{timestamp}_{random}
 */
function generateSessionId() {
  const timestamp = Date.now();
  const random = Math.random().toString(36).substring(2, 8);
  return `session_${timestamp}_${random}`;
}

/**
 * Create initial state object for new colony
 * @param {string} goal - Colony goal
 * @returns {object} Initial state object
 */
function createInitialState(goal) {
  const now = new Date().toISOString();
  const sessionId = generateSessionId();

  return {
    version: '3.0',
    goal: goal || 'Aether colony initialization',
    state: 'INITIALIZING',
    current_phase: 0,
    session_id: sessionId,
    initialized_at: now,
    build_started_at: null,
    plan: {
      generated_at: null,
      confidence: null,
      phases: []
    },
    memory: {
      phase_learnings: [],
      decisions: [],
      instincts: []
    },
    errors: {
      records: [],
      flagged_patterns: []
    },
    signals: [],
    graveyards: [],
    events: [
      {
        timestamp: now,
        type: 'colony_initialized',
        worker: 'init',
        details: `Colony initialized with goal: ${goal || 'Aether colony initialization'}`
      }
    ],
    created_at: now,
    last_updated: now
  };
}

/**
 * Check if a repository is already initialized
 * @param {string} repoPath - Path to repository root
 * @returns {boolean} True if initialized
 */
function isInitialized(repoPath) {
  const stateFile = path.join(repoPath, '.aether', 'data', 'COLONY_STATE.json');

  // Check if state file exists
  if (!fs.existsSync(stateFile)) {
    return false;
  }

  // Check if required directories exist
  const requiredDirs = [
    path.join(repoPath, '.aether'),
    path.join(repoPath, '.aether', 'data'),
    path.join(repoPath, '.aether', 'checkpoints'),
    path.join(repoPath, '.aether', 'locks')
  ];

  for (const dir of requiredDirs) {
    if (!fs.existsSync(dir)) {
      return false;
    }
  }

  return true;
}

/**
 * Validate initialization of a repository
 * @param {string} repoPath - Path to repository root
 * @returns {object} Validation result: { valid: boolean, errors: string[] }
 */
function validateInitialization(repoPath) {
  const errors = [];

  // Check required directories
  const requiredDirs = [
    { path: path.join(repoPath, '.aether'), name: '.aether/' },
    { path: path.join(repoPath, '.aether', 'data'), name: '.aether/data/' },
    { path: path.join(repoPath, '.aether', 'checkpoints'), name: '.aether/checkpoints/' },
    { path: path.join(repoPath, '.aether', 'locks'), name: '.aether/locks/' }
  ];

  for (const dir of requiredDirs) {
    if (!fs.existsSync(dir.path)) {
      errors.push(`Missing directory: ${dir.name}`);
    }
  }

  // Check state file
  const stateFile = path.join(repoPath, '.aether', 'data', 'COLONY_STATE.json');
  if (!fs.existsSync(stateFile)) {
    errors.push('Missing state file: .aether/data/COLONY_STATE.json');
  } else {
    // Validate JSON structure
    try {
      const content = fs.readFileSync(stateFile, 'utf8');
      const state = JSON.parse(content);

      // Check required fields
      const requiredFields = ['version', 'goal', 'state', 'current_phase', 'session_id', 'initialized_at'];
      for (const field of requiredFields) {
        if (!(field in state)) {
          errors.push(`State file missing required field: ${field}`);
        }
      }

      // Validate events array
      if (!Array.isArray(state.events)) {
        errors.push('State file events field must be an array');
      }

      // Validate current_phase is a number
      if (typeof state.current_phase !== 'number') {
        errors.push('State file current_phase must be a number');
      }

    } catch (err) {
      errors.push(`Invalid JSON in state file: ${err.message}`);
    }
  }

  return {
    valid: errors.length === 0,
    errors
  };
}

/**
 * Initialize a new repository with Aether colony
 * @param {string} repoPath - Path to repository root
 * @param {object} options - Initialization options
 * @param {string} options.goal - Colony goal
 * @param {boolean} options.skipIfExists - Skip if already initialized
 * @returns {object} Result: { success: boolean, stateFile: string|null, message: string }
 */
async function initializeRepo(repoPath, options = {}) {
  const { goal, skipIfExists = false } = options;

  // Check if already initialized
  if (isInitialized(repoPath) && skipIfExists) {
    return {
      success: true,
      stateFile: path.join(repoPath, '.aether', 'data', 'COLONY_STATE.json'),
      message: 'Repository already initialized, skipping'
    };
  }

  // Create directory structure
  const dirs = [
    path.join(repoPath, '.aether'),
    path.join(repoPath, '.aether', 'data'),
    path.join(repoPath, '.aether', 'checkpoints'),
    path.join(repoPath, '.aether', 'locks')
  ];

  for (const dir of dirs) {
    fs.mkdirSync(dir, { recursive: true });
  }

  // Create .gitignore for .aether directory
  const gitignorePath = path.join(repoPath, '.aether', '.gitignore');
  const gitignoreContent = `# Aether local state - not versioned
data/
checkpoints/
locks/
`;
  fs.writeFileSync(gitignorePath, gitignoreContent);

  // Create initial state
  const state = createInitialState(goal);

  // Write state file
  const stateFile = path.join(repoPath, '.aether', 'data', 'COLONY_STATE.json');
  fs.writeFileSync(stateFile, JSON.stringify(state, null, 2) + '\n');

  return {
    success: true,
    stateFile,
    message: 'Repository initialized successfully'
  };
}

module.exports = {
  initializeRepo,
  isInitialized,
  validateInitialization,
  createInitialState,
  generateSessionId
};
