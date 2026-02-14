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

// Hub paths (for copying system files during init)
const HOME = process.env.HOME || process.env.USERPROFILE;
const HUB_DIR = HOME ? path.join(HOME, '.aether') : null;
const HUB_SYSTEM = HUB_DIR ? path.join(HUB_DIR, 'system') : null;
const HUB_COMMANDS_CLAUDE = HUB_DIR ? path.join(HUB_DIR, 'commands', 'claude') : null;
const HUB_COMMANDS_OPENCODE = HUB_DIR ? path.join(HUB_DIR, 'commands', 'opencode') : null;
const HUB_AGENTS = HUB_DIR ? path.join(HUB_DIR, 'agents') : null;
const HUB_REGISTRY = HUB_DIR ? path.join(HUB_DIR, 'registry.json') : null;
const HUB_VERSION = HUB_DIR ? path.join(HUB_DIR, 'version.json') : null;

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
 * Read JSON file safely
 * @param {string} filePath - Path to JSON file
 * @returns {object|null} Parsed JSON or null if file doesn't exist or is invalid
 */
function readJsonSafe(filePath) {
  try {
    if (!fs.existsSync(filePath)) return null;
    const content = fs.readFileSync(filePath, 'utf8');
    return JSON.parse(content);
  } catch {
    return null;
  }
}

/**
 * Write JSON file atomically
 * @param {string} filePath - Path to write
 * @param {object} data - Data to write
 */
function writeJsonSync(filePath, data) {
  const dir = path.dirname(filePath);
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
  }
  const tempFile = `${filePath}.tmp.${Date.now()}`;
  fs.writeFileSync(tempFile, JSON.stringify(data, null, 2) + '\n');
  fs.renameSync(tempFile, filePath);
}

/**
 * List files recursively in a directory
 * @param {string} dir - Directory to list
 * @param {string} prefix - Prefix for relative paths
 * @returns {string[]} Array of relative file paths
 */
function listFilesRecursive(dir, prefix = '') {
  const files = [];
  if (!fs.existsSync(dir)) return files;

  const entries = fs.readdirSync(dir, { withFileTypes: true });
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    const relPath = prefix ? path.join(prefix, entry.name) : entry.name;

    if (entry.isDirectory()) {
      files.push(...listFilesRecursive(fullPath, relPath));
    } else {
      files.push(relPath);
    }
  }
  return files;
}

/**
 * Sync files from source to destination directory
 * @param {string} src - Source directory
 * @param {string} dest - Destination directory
 * @returns {object} Result: { copied: number, skipped: number }
 */
function syncFiles(src, dest) {
  let copied = 0;
  let skipped = 0;

  if (!fs.existsSync(src)) {
    return { copied, skipped, error: `Source directory not found: ${src}` };
  }

  fs.mkdirSync(dest, { recursive: true });

  const files = listFilesRecursive(src);
  for (const relPath of files) {
    const srcPath = path.join(src, relPath);
    const destPath = path.join(dest, relPath);

    try {
      fs.mkdirSync(path.dirname(destPath), { recursive: true });

      // Check if file exists and compare content
      let shouldCopy = true;
      if (fs.existsSync(destPath)) {
        const srcContent = fs.readFileSync(srcPath);
        const destContent = fs.readFileSync(destPath);
        if (srcContent.equals(destContent)) {
          shouldCopy = false;
          skipped++;
        }
      }

      if (shouldCopy) {
        fs.copyFileSync(srcPath, destPath);
        if (relPath.endsWith('.sh')) {
          fs.chmodSync(destPath, 0o755);
        }
        copied++;
      }
    } catch (err) {
      console.error(`Warning: could not copy ${relPath}: ${err.message}`);
    }
  }

  return { copied, skipped };
}

/**
 * Register a repository in the global registry
 * @param {string} repoPath - Path to repository
 * @param {string} version - Aether version
 * @returns {object} Result: { success: boolean, message: string }
 */
function registerRepo(repoPath, version) {
  if (!HUB_REGISTRY) {
    return { success: false, message: 'HOME environment variable not set' };
  }

  // Initialize registry if it doesn't exist
  let registry = readJsonSafe(HUB_REGISTRY);
  if (!registry) {
    registry = { schema_version: 1, repos: [] };
  }

  // Check if repo already exists
  const existingIndex = registry.repos.findIndex(r => r.path === repoPath);
  const now = new Date().toISOString();

  if (existingIndex >= 0) {
    // Update existing entry
    registry.repos[existingIndex].version = version;
    registry.repos[existingIndex].updated_at = now;
  } else {
    // Add new entry
    registry.repos.push({
      path: repoPath,
      version: version,
      registered_at: now,
      updated_at: now
    });
  }

  writeJsonSync(HUB_REGISTRY, registry);
  return { success: true, message: existingIndex >= 0 ? 'Updated registry entry' : 'Added to registry' };
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
  const { goal, skipIfExists = false, quiet = false } = options;

  // Check if already initialized
  if (isInitialized(repoPath) && skipIfExists) {
    return {
      success: true,
      stateFile: path.join(repoPath, '.aether', 'data', 'COLONY_STATE.json'),
      message: 'Repository already initialized, skipping'
    };
  }

  // Check if hub exists
  if (!HUB_DIR || !fs.existsSync(HUB_DIR)) {
    return {
      success: false,
      stateFile: null,
      message: 'Aether hub not found. Run "aether install" first to set up the distribution hub.'
    };
  }

  const results = {
    system: { copied: 0, skipped: 0 },
    commands: { copied: 0, skipped: 0 },
    agents: { copied: 0, skipped: 0 }
  };

  // Sync system files from hub
  if (HUB_SYSTEM && fs.existsSync(HUB_SYSTEM)) {
    const destSystem = path.join(repoPath, '.aether');
    results.system = syncFiles(HUB_SYSTEM, destSystem);
    if (!quiet && results.system.copied > 0) {
      console.log(`  System files: ${results.system.copied} copied, ${results.system.skipped} skipped`);
    }
  }

  // Sync Claude commands
  if (HUB_COMMANDS_CLAUDE && fs.existsSync(HUB_COMMANDS_CLAUDE)) {
    const destClaude = path.join(repoPath, '.claude', 'commands', 'ant');
    results.commands = syncFiles(HUB_COMMANDS_CLAUDE, destClaude);
    if (!quiet && results.commands.copied > 0) {
      console.log(`  Claude commands: ${results.commands.copied} copied, ${results.commands.skipped} skipped`);
    }
  }

  // Sync OpenCode commands
  if (HUB_COMMANDS_OPENCODE && fs.existsSync(HUB_COMMANDS_OPENCODE)) {
    const destOpencode = path.join(repoPath, '.opencode', 'commands', 'ant');
    const opencodeResult = syncFiles(HUB_COMMANDS_OPENCODE, destOpencode);
    results.commands.copied += opencodeResult.copied;
    results.commands.skipped += opencodeResult.skipped;
  }

  // Sync agents
  if (HUB_AGENTS && fs.existsSync(HUB_AGENTS)) {
    const destAgents = path.join(repoPath, '.opencode', 'agents');
    results.agents = syncFiles(HUB_AGENTS, destAgents);
    if (!quiet && results.agents.copied > 0) {
      console.log(`  Agents: ${results.agents.copied} copied, ${results.agents.skipped} skipped`);
    }
  }

  // Create directory structure (in case some weren't created by sync)
  const dirs = [
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

  // Get hub version
  const hubVersion = readJsonSafe(HUB_VERSION);
  const version = hubVersion ? hubVersion.version : '1.0.0';

  // Register in global registry
  const registerResult = registerRepo(repoPath, version);
  if (!quiet && registerResult.success) {
    console.log(`  ${registerResult.message}`);
  }

  // Write version file
  const versionFile = path.join(repoPath, '.aether', 'version.json');
  writeJsonSync(versionFile, {
    version: version,
    initialized_at: new Date().toISOString()
  });

  return {
    success: true,
    stateFile,
    message: 'Repository initialized successfully',
    version: version,
    filesCopied: results.system.copied + results.commands.copied + results.agents.copied,
    registered: registerResult.success
  };
}

module.exports = {
  initializeRepo,
  isInitialized,
  validateInitialization,
  createInitialState,
  generateSessionId
};
