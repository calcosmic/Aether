#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const { execSync } = require('child_process');
const { program } = require('commander');

// Error handling imports
const {
  AetherError,
  HubError,
  RepoError,
  GitError,
  ValidationError,
  FileSystemError,
  ConfigurationError,
  getExitCode,
  wrapError,
} = require('./lib/errors');
const { logError, logActivity } = require('./lib/logger');
const { UpdateTransaction, UpdateError, UpdateErrorCodes } = require('./lib/update-transaction');
const { initializeRepo, isInitialized } = require('./lib/init');
const { syncStateFromPlanning, reconcileStates } = require('./lib/state-sync');
const { createVerificationReport } = require('./lib/model-verify');
const {
  loadModelProfiles,
  getAllAssignments,
  getProviderForModel,
  validateCaste,
  validateModel,
  setModelOverride,
  resetModelOverride,
  getEffectiveModel,
  getUserOverrides,
  getModelMetadata,
  getProxyConfig,
} = require('./lib/model-profiles');
const {
  checkProxyHealth,
  verifyModelRouting,
  formatProxyStatusColored,
} = require('./lib/proxy-health');
const { findNestmates, formatNestmates, loadNestmateTodos } = require('./lib/nestmate-loader');
const { logSpawn, formatSpawnTree } = require('./lib/spawn-logger');
const {
  getTelemetrySummary,
  getModelPerformance,
} = require('./lib/telemetry');

// Color palette
const c = require('./lib/colors');

const VERSION = require('../package.json').version;
const PACKAGE_DIR = path.resolve(__dirname, '..');
const HOME = process.env.HOME || process.env.USERPROFILE;
if (!HOME) {
  const error = new ConfigurationError(
    'HOME environment variable is not set',
    { env: Object.keys(process.env).filter(k => k.includes('HOME') || k.includes('USER')) },
    'Please ensure HOME or USERPROFILE is defined'
  );
  console.error(JSON.stringify(error.toJSON(), null, 2));
  process.exit(getExitCode(error.code));
}

// Claude Code paths (global)
const COMMANDS_SRC = path.join(PACKAGE_DIR, 'commands', 'ant');
const COMMANDS_DEST = path.join(HOME, '.claude', 'commands', 'ant');

// Hub paths
const HUB_DIR = path.join(HOME, '.aether');
const HUB_SYSTEM_DIR = path.join(HUB_DIR, 'system');
const HUB_COMMANDS_CLAUDE = path.join(HUB_SYSTEM_DIR, 'commands', 'claude');
const HUB_COMMANDS_OPENCODE = path.join(HUB_SYSTEM_DIR, 'commands', 'opencode');
const HUB_AGENTS = path.join(HUB_SYSTEM_DIR, 'agents');
const HUB_RULES = path.join(HUB_SYSTEM_DIR, 'rules');
const HUB_REGISTRY = path.join(HUB_DIR, 'registry.json');
const HUB_VERSION = path.join(HUB_DIR, 'version.json');

// Global quiet flag (set by --quiet option)
let globalQuiet = false;

// Global error handlers
process.on('uncaughtException', (error) => {
  const structuredError = wrapError(error);
  structuredError.code = 'E_UNCAUGHT_EXCEPTION';
  structuredError.recovery = 'Please report this issue with the error details';

  // Log to activity.log
  logError(structuredError);

  // Output structured JSON to stderr
  console.error(JSON.stringify(structuredError.toJSON(), null, 2));

  // Exit with appropriate code
  process.exit(getExitCode(structuredError.code));
});

process.on('unhandledRejection', (reason, promise) => {
  const message = reason instanceof Error ? reason.message : String(reason);
  const details = reason instanceof Error ? { stack: reason.stack, name: reason.name } : {};

  const error = new AetherError(
    'E_UNHANDLED_REJECTION',
    message,
    { ...details, promise: String(promise) },
    'Please report this issue with the error details'
  );

  // Log to activity.log
  logError(error);

  // Output structured JSON to stderr
  console.error(JSON.stringify(error.toJSON(), null, 2));

  // Exit with appropriate code
  process.exit(getExitCode(error.code));
});

/**
 * Feature Flags class for graceful degradation
 * Tracks which features are available vs degraded
 */
class FeatureFlags {
  constructor() {
    this.features = {
      activityLog: true,
      progressDisplay: true,
      gitIntegration: true,
      hashComparison: true,
      manifestTracking: true,
    };
    this.degradedFeatures = new Set();
  }

  /**
   * Disable a feature with a reason
   * @param {string} feature - Feature name
   * @param {string} reason - Why the feature was disabled
   */
  disable(feature, reason) {
    if (this.features.hasOwnProperty(feature)) {
      this.features[feature] = false;
      this.degradedFeatures.add({ feature, reason, timestamp: new Date().toISOString() });

      // Log degradation warning
      console.warn(JSON.stringify({
        warning: {
          type: 'FEATURE_DEGRADED',
          feature,
          reason,
          timestamp: new Date().toISOString(),
        },
      }));
    }
  }

  /**
   * Check if a feature is enabled
   * @param {string} feature - Feature name
   * @returns {boolean} True if enabled
   */
  isEnabled(feature) {
    return this.features[feature] || false;
  }

  /**
   * Get list of degraded features
   * @returns {Array} Array of degraded feature objects
   */
  getDegradedFeatures() {
    return Array.from(this.degradedFeatures);
  }
}

// Global feature flags instance
const features = new FeatureFlags();

/**
 * Wrap a command function with error handling
 * @param {Function} commandFn - Async command function to wrap
 * @param {object} options - Options for error handling
 * @param {boolean} options.logActivity - Whether to log activity (default: true)
 * @returns {Function} Wrapped function
 */
function wrapCommand(commandFn, options = {}) {
  const { logActivity: shouldLog = true } = options;

  return async (...args) => {
    try {
      return await commandFn(...args);
    } catch (error) {
      let structuredError;

      if (error instanceof AetherError) {
        structuredError = error;
      } else {
        structuredError = wrapError(error);
      }

      // Log to activity.log
      if (shouldLog) {
        logError(structuredError);
      }

      // Output structured JSON to stderr
      console.error(JSON.stringify(structuredError.toJSON(), null, 2));

      // Exit with appropriate code
      process.exit(getExitCode(structuredError.code));
    }
  };
}

function log(msg) {
  if (!globalQuiet) console.log(msg);
}

/**
 * Format UpdateError with prominent recovery commands display
 * @param {UpdateError} error - The update error to format
 * @returns {string} Formatted error message with recovery box
 */
function formatUpdateError(error) {
  const lines = [];

  // Header box
  const headerWidth = 62;
  lines.push(c.error('╔' + '═'.repeat(headerWidth) + '╗'));
  lines.push(c.error('║') + '  UPDATE FAILED'.padEnd(headerWidth) + c.error('║'));
  lines.push(c.error('╚' + '═'.repeat(headerWidth) + '╝'));
  lines.push('');

  // Error code and message
  lines.push(`Error: ${c.bold(error.code)} - ${error.message}`);
  lines.push('');

  // Details section
  if (error.details && Object.keys(error.details).length > 0) {
    lines.push('Details:');

    // Handle specific error types with formatted details
    switch (error.code) {
    case UpdateErrorCodes.E_REPO_DIRTY:
      if (error.details.trackedCount > 0) {
        lines.push(`  Modified files: ${error.details.trackedCount}`);
      }
      if (error.details.untrackedCount > 0) {
        lines.push(`  Untracked files: ${error.details.untrackedCount}`);
      }
      if (error.details.stagedCount > 0) {
        lines.push(`  Staged files: ${error.details.stagedCount}`);
      }
      break;

    case UpdateErrorCodes.E_HUB_INACCESSIBLE:
      if (error.details.errors) {
        for (const err of error.details.errors.slice(0, 3)) {
          lines.push(`  - ${err}`);
        }
      }
      break;

    case UpdateErrorCodes.E_PARTIAL_UPDATE:
      lines.push(`  Missing files: ${error.details.missingCount || 0}`);
      lines.push(`  Corrupted files: ${error.details.corruptedCount || 0}`);
      break;

    case UpdateErrorCodes.E_NETWORK_ERROR:
      lines.push(`  Hub directory: ${error.details.hubDir || 'unknown'}`);
      if (error.details.errorCode) {
        lines.push(`  Error code: ${error.details.errorCode}`);
      }
      break;

    default:
      // Generic details display
      for (const [key, value] of Object.entries(error.details)) {
        if (typeof value === 'number' || typeof value === 'string') {
          lines.push(`  ${key}: ${value}`);
        }
      }
    }
    lines.push('');
  }

  // Recovery commands box
  if (error.recoveryCommands && error.recoveryCommands.length > 0) {
    const maxCmdLength = Math.max(...error.recoveryCommands.map(cmd => cmd.length));
    const boxWidth = Math.max(maxCmdLength + 4, 40);

    lines.push(c.warning('╔' + '═'.repeat(boxWidth) + '╗'));
    lines.push(c.warning('║') + '  RECOVERY COMMANDS'.padEnd(boxWidth) + c.warning('║'));
    lines.push(c.warning('║') + ' '.repeat(boxWidth) + c.warning('║'));

    for (const cmd of error.recoveryCommands) {
      lines.push(c.warning('║') + '  ' + c.bold(cmd).padEnd(boxWidth - 2) + c.warning('║'));
    }

    lines.push(c.warning('║') + ' '.repeat(boxWidth) + c.warning('║'));

    // Add specific guidance based on error type
    switch (error.code) {
    case UpdateErrorCodes.E_REPO_DIRTY:
      lines.push(c.warning('║') + '  Or to discard changes (DANGER):'.padEnd(boxWidth) + c.warning('║'));
      lines.push(c.warning('║') + '  ' + c.bold('git checkout -- .').padEnd(boxWidth - 2) + c.warning('║'));
      break;

    case UpdateErrorCodes.E_HUB_INACCESSIBLE:
      lines.push(c.warning('║') + '  Or to reinstall hub:'.padEnd(boxWidth) + c.warning('║'));
      lines.push(c.warning('║') + '  ' + c.bold('aether install').padEnd(boxWidth - 2) + c.warning('║'));
      break;

    case UpdateErrorCodes.E_PARTIAL_UPDATE:
    case UpdateErrorCodes.E_NETWORK_ERROR:
      lines.push(c.warning('║') + '  Then retry:'.padEnd(boxWidth) + c.warning('║'));
      lines.push(c.warning('║') + '  ' + c.bold('aether update').padEnd(boxWidth - 2) + c.warning('║'));
      break;

    case UpdateErrorCodes.E_VERIFY_FAILED:
      lines.push(c.warning('║') + '  Or restore checkpoint:'.padEnd(boxWidth) + c.warning('║'));
      if (error.details?.checkpoint_id) {
        lines.push(c.warning('║') + `  ${c.bold(`aether checkpoint restore ${error.details.checkpoint_id}`)}`.padEnd(boxWidth) + c.warning('║'));
      }
      break;
    }

    lines.push(c.warning('╚' + '═'.repeat(boxWidth) + '╝'));
  }

  // Checkpoint ID if available
  if (error.details?.checkpoint_id) {
    lines.push('');
    lines.push(`Checkpoint ID: ${c.dim(error.details.checkpoint_id)} (for manual restore if needed)`);
  }

  return lines.join('\n');
}

function copyDirSync(src, dest) {
  fs.mkdirSync(dest, { recursive: true });
  const entries = fs.readdirSync(src, { withFileTypes: true });
  let count = 0;
  for (const entry of entries) {
    const srcPath = path.join(src, entry.name);
    const destPath = path.join(dest, entry.name);
    if (entry.isDirectory()) {
      count += copyDirSync(srcPath, destPath);
    } else {
      fs.copyFileSync(srcPath, destPath);
      // Preserve executable bit for shell scripts
      if (entry.name.endsWith('.sh')) {
        fs.chmodSync(destPath, 0o755);
      }
      count++;
    }
  }
  return count;
}

function removeDirSync(dir) {
  if (!fs.existsSync(dir)) return 0;
  let count = 0;
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      count += removeDirSync(fullPath);
    } else {
      fs.unlinkSync(fullPath);
      count++;
    }
  }
  fs.rmdirSync(dir);
  return count;
}

// System files allowlist — must match bin/sync-to-runtime.sh SYSTEM_FILES exactly
const SYSTEM_FILES = [
  'aether-utils.sh',
  'coding-standards.md',
  'debugging.md',
  'DISCIPLINES.md',
  'learning.md',
  'planning.md',
  'QUEEN_ANT_ARCHITECTURE.md',
  'tdd.md',
  'verification-loop.md',
  'verification.md',
  'workers.md',
  'workers-new-castes.md',
  'CONTEXT.md',
  'model-profiles.yaml',
  'recover.sh',
  'docs/biological-reference.md',
  'docs/command-sync.md',
  'docs/constraints.md',
  'docs/namespace.md',
  'docs/pathogen-schema-example.json',
  'docs/pathogen-schema.md',
  'docs/PHEROMONE-INJECTION.md',
  'docs/PHEROMONE-INTEGRATION.md',
  'docs/PHEROMONE-SYSTEM-DESIGN.md',
  'docs/pheromones.md',
  'docs/progressive-disclosure.md',
  'docs/README.md',
  'docs/VISUAL-OUTPUT-SPEC.md',
  'docs/known-issues.md',
  'docs/implementation-learnings.md',
  'docs/codebase-review.md',
  'docs/planning-discipline.md',
  'utils/atomic-write.sh',
  'utils/chamber-compare.sh',
  'utils/chamber-utils.sh',
  'utils/colorize-log.sh',
  'utils/error-handler.sh',
  'utils/file-lock.sh',
  'utils/spawn-tree.sh',
  'utils/spawn-with-model.sh',
  'utils/state-loader.sh',
  'utils/swarm-display.sh',
  'utils/watch-spawn-tree.sh',
  'utils/xml-utils.sh',
  'utils/xml-core.sh',
  'utils/xml-compose.sh',
  'utils/queen-to-md.xsl',
  'exchange/pheromone-xml.sh',
  'exchange/wisdom-xml.sh',
  'exchange/registry-xml.sh',
  'schemas/aether-types.xsd',
  'schemas/pheromone.xsd',
  'schemas/queen-wisdom.xsd',
  'schemas/colony-registry.xsd',
  'schemas/worker-priming.xsd',
  'schemas/prompt.xsd',
  'templates/QUEEN.md.template',
  'rules/aether-colony.md',
];

function copySystemFiles(srcDir, destDir) {
  let count = 0;
  for (const file of SYSTEM_FILES) {
    const srcPath = path.join(srcDir, file);
    const destPath = path.join(destDir, file);
    if (fs.existsSync(srcPath)) {
      fs.mkdirSync(path.dirname(destPath), { recursive: true });
      fs.copyFileSync(srcPath, destPath);
      if (file.endsWith('.sh')) {
        fs.chmodSync(destPath, 0o755);
      }
      count++;
    }
  }
  return count;
}

function readJsonSafe(filePath) {
  try {
    return JSON.parse(fs.readFileSync(filePath, 'utf8'));
  } catch {
    return null;
  }
}

function writeJsonSync(filePath, data) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, JSON.stringify(data, null, 2) + '\n');
}

function hashFileSync(filePath) {
  try {
    const content = fs.readFileSync(filePath);
    return 'sha256:' + crypto.createHash('sha256').update(content).digest('hex');
  } catch (err) {
    console.error(`Warning: could not hash ${filePath}: ${err.message}`);
    return null;
  }
}

function validateManifest(manifest) {
  if (!manifest || typeof manifest !== 'object') {
    return { valid: false, error: 'Manifest must be an object' };
  }
  if (!manifest.generated_at || typeof manifest.generated_at !== 'string') {
    return { valid: false, error: 'Manifest missing required field: generated_at' };
  }
  if (!manifest.files || typeof manifest.files !== 'object' || Array.isArray(manifest.files)) {
    return { valid: false, error: 'Manifest missing required field: files' };
  }
  return { valid: true };
}

function listFilesRecursive(dir, base) {
  base = base || dir;
  const results = [];
  if (!fs.existsSync(dir)) return results;
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  for (const entry of entries) {
    if (entry.name.startsWith('.')) continue;
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      results.push(...listFilesRecursive(fullPath, base));
    } else {
      results.push(path.relative(base, fullPath));
    }
  }
  return results;
}

function cleanEmptyDirs(dir) {
  if (!fs.existsSync(dir)) return;
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  for (const entry of entries) {
    if (entry.isDirectory()) {
      cleanEmptyDirs(path.join(dir, entry.name));
    }
  }
  // Re-read after recursive cleanup
  const remaining = fs.readdirSync(dir);
  if (remaining.length === 0) {
    fs.rmdirSync(dir);
  }
}

function generateManifest(hubDir) {
  const files = {};
  const allFiles = listFilesRecursive(hubDir);
  for (const relPath of allFiles) {
    // Skip registry, version, and manifest metadata files
    if (relPath === 'registry.json' || relPath === 'version.json' || relPath === 'manifest.json') continue;
    const fullPath = path.join(hubDir, relPath);
    const hash = hashFileSync(fullPath);
    // Skip files that couldn't be hashed (permission issues, etc.)
    if (hash) {
      files[relPath] = hash;
    }
  }
  return { generated_at: new Date().toISOString(), files };
}

function syncDirWithCleanup(src, dest, opts) {
  opts = opts || {};
  const dryRun = opts.dryRun || false;
  try {
    fs.mkdirSync(dest, { recursive: true });
  } catch (err) {
    if (err.code !== 'EEXIST') {
      console.error(`Warning: could not create directory ${dest}: ${err.message}`);
    }
  }

  // Copy phase with hash comparison
  let copied = 0;
  let skipped = 0;
  const srcFiles = listFilesRecursive(src);
  if (!dryRun) {
    for (const relPath of srcFiles) {
      const srcPath = path.join(src, relPath);
      const destPath = path.join(dest, relPath);
      try {
        fs.mkdirSync(path.dirname(destPath), { recursive: true });

        // Hash comparison: only copy if file doesn't exist or hash differs
        let shouldCopy = true;
        if (fs.existsSync(destPath)) {
          const srcHash = hashFileSync(srcPath);
          const destHash = hashFileSync(destPath);
          if (srcHash === destHash) {
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
        skipped++;
      }
    }
  } else {
    copied = srcFiles.length;
  }

  // Cleanup phase — remove files in dest that aren't in src
  const destFiles = listFilesRecursive(dest);
  const srcSet = new Set(srcFiles);
  const removed = [];
  for (const relPath of destFiles) {
    if (!srcSet.has(relPath)) {
      removed.push(relPath);
      if (!dryRun) {
        try {
          fs.unlinkSync(path.join(dest, relPath));
        } catch (err) {
          console.error(`Warning: could not remove ${relPath}: ${err.message}`);
        }
      }
    }
  }

  if (!dryRun && removed.length > 0) {
    try {
      cleanEmptyDirs(dest);
    } catch (err) {
      console.error(`Warning: could not clean directories: ${err.message}`);
    }
  }

  return { copied, removed, skipped };
}

function computeFileHash(filePath) {
  try {
    const content = fs.readFileSync(filePath);
    return crypto.createHash('sha256').update(content).digest('hex');
  } catch {
    return null;
  }
}

function syncSystemFilesWithCleanup(srcDir, destDir, opts) {
  opts = opts || {};
  const dryRun = opts.dryRun || false;

  let copied = 0;
  let skipped = 0;
  for (const file of SYSTEM_FILES) {
    const srcPath = path.join(srcDir, file);
    const destPath = path.join(destDir, file);
    if (fs.existsSync(srcPath)) {
      if (!dryRun) {
        // Compute hashes to determine if copy is needed
        const srcHash = computeFileHash(srcPath);
        const destHash = fs.existsSync(destPath) ? computeFileHash(destPath) : null;

        if (srcHash === destHash) {
          // Files are identical, skip copying
          skipped++;
          continue;
        }

        fs.mkdirSync(path.dirname(destPath), { recursive: true });
        fs.copyFileSync(srcPath, destPath);
        if (file.endsWith('.sh')) {
          fs.chmodSync(destPath, 0o755);
        }
      }
      copied++;
    }
  }

  // Remove allowlisted files that no longer exist in src
  const removed = [];
  for (const file of SYSTEM_FILES) {
    const srcPath = path.join(srcDir, file);
    const destPath = path.join(destDir, file);
    if (!fs.existsSync(srcPath) && fs.existsSync(destPath)) {
      removed.push(file);
      if (!dryRun) {
        fs.unlinkSync(destPath);
      }
    }
  }

  if (!dryRun && removed.length > 0) {
    cleanEmptyDirs(destDir);
  }

  return { copied, removed, skipped };
}

// Checkpoint allowlist - only these files are captured in checkpoints
// NEVER include: data/, dreams/, oracle/, TO-DOs.md (user data)
// Note: runtime/ is generated during publish only, not checkpointed
const CHECKPOINT_ALLOWLIST = [
  '.aether/*.md',                    // All .md files directly in .aether/
  '.claude/commands/ant/**',         // All files in .claude/commands/ant/ recursively
  '.opencode/commands/ant/**',       // All files in .opencode/commands/ant/ recursively
  '.opencode/agents/**',             // All files in .opencode/agents/ recursively
  'bin/cli.js',                      // Specific file: bin/cli.js
];

// Forbidden user data patterns - these are NEVER checkpointed
const USER_DATA_PATTERNS = [
  'data/',
  'dreams/',
  'oracle/',
  'TO-DOs.md',
];

/**
 * Check if a file path matches user data patterns
 * @param {string} filePath - File path to check
 * @returns {boolean} True if file is user data
 */
function isUserData(filePath) {
  for (const pattern of USER_DATA_PATTERNS) {
    if (filePath.includes(pattern)) {
      return true;
    }
  }
  return false;
}

/**
 * Check if a file is tracked by git
 * @param {string} repoPath - Repository root path
 * @param {string} filePath - File path relative to repo root
 * @returns {boolean} True if file is tracked by git
 */
function isGitTracked(repoPath, filePath) {
  try {
    execSync(`git ls-files --error-unmatch "${filePath}"`, {
      cwd: repoPath,
      stdio: 'pipe'
    });
    return true;
  } catch {
    return false;
  }
}

/**
 * Get files matching the checkpoint allowlist
 * @param {string} repoPath - Repository root path
 * @returns {string[]} Array of file paths relative to repo root
 */
function getAllowlistedFiles(repoPath) {
  const files = [];

  for (const pattern of CHECKPOINT_ALLOWLIST) {
    if (pattern === 'bin/cli.js') {
      // Specific file - must be tracked by git for stash to work
      const fullPath = path.join(repoPath, pattern);
      if (fs.existsSync(fullPath) && isGitTracked(repoPath, pattern)) {
        files.push(pattern);
      }
    } else if (pattern === '.aether/*.md') {
      // .md files directly in .aether/ (not subdirs)
      const aetherDir = path.join(repoPath, '.aether');
      if (fs.existsSync(aetherDir)) {
        const entries = fs.readdirSync(aetherDir, { withFileTypes: true });
        for (const entry of entries) {
          if (entry.isFile() && entry.name.endsWith('.md')) {
            const filePath = path.join('.aether', entry.name);
            if (!isUserData(filePath) && isGitTracked(repoPath, filePath)) {
              files.push(filePath);
            }
          }
        }
      }
    } else if (pattern.endsWith('/**')) {
      // Recursive directory pattern
      const dirPath = pattern.slice(0, -3); // Remove '/**'
      const fullDir = path.join(repoPath, dirPath);
      if (fs.existsSync(fullDir)) {
        const dirFiles = listFilesRecursive(fullDir);
        for (const relFile of dirFiles) {
          const filePath = path.join(dirPath, relFile);
          if (!isUserData(filePath) && isGitTracked(repoPath, filePath)) {
            files.push(filePath);
          }
        }
      }
    }
  }

  return files;
}

/**
 * Generate checkpoint metadata with file hashes
 * @param {string} repoPath - Repository root path
 * @param {string} message - Optional checkpoint message
 * @returns {object} Checkpoint metadata object
 */
function generateCheckpointMetadata(repoPath, message) {
  const now = new Date();
  const timestamp = now.toISOString().replace(/[:.]/g, '-').slice(0, 19);
  const checkpointId = `chk_${now.toISOString().slice(0, 10).replace(/-/g, '')}_${now.toTimeString().slice(0, 8).replace(/:/g, '')}`;

  const allowlistedFiles = getAllowlistedFiles(repoPath);
  const files = {};
  const excluded = [];

  for (const filePath of allowlistedFiles) {
    // Double-check for user data (safety)
    if (isUserData(filePath)) {
      excluded.push(filePath);
      continue;
    }

    const fullPath = path.join(repoPath, filePath);
    const hash = hashFileSync(fullPath);
    if (hash) {
      files[filePath] = hash;
    }
  }

  return {
    checkpoint_id: checkpointId,
    created_at: now.toISOString(),
    message: message || 'Checkpoint created',
    files,
    excluded: excluded.length > 0 ? excluded : undefined,
  };
}

/**
 * Save checkpoint metadata to .aether/checkpoints/
 * @param {string} repoPath - Repository root path
 * @param {object} metadata - Checkpoint metadata object
 */
function saveCheckpointMetadata(repoPath, metadata) {
  const checkpointsDir = path.join(repoPath, '.aether', 'checkpoints');
  fs.mkdirSync(checkpointsDir, { recursive: true });
  const metadataPath = path.join(checkpointsDir, `${metadata.checkpoint_id}.json`);
  writeJsonSync(metadataPath, metadata);
}

/**
 * Load checkpoint metadata by ID
 * @param {string} repoPath - Repository root path
 * @param {string} checkpointId - Checkpoint ID
 * @returns {object|null} Checkpoint metadata or null if not found
 */
function loadCheckpointMetadata(repoPath, checkpointId) {
  const metadataPath = path.join(repoPath, '.aether', 'checkpoints', `${checkpointId}.json`);
  return readJsonSafe(metadataPath);
}

function isGitRepo(repoPath) {
  try {
    execSync('git rev-parse --git-dir', { cwd: repoPath, stdio: 'pipe' });
    return true;
  } catch {
    return false;
  }
}

function getGitDirtyFiles(repoPath, targetDirs) {
  try {
    const args = targetDirs.filter(d => fs.existsSync(path.join(repoPath, d)));
    if (args.length === 0) return [];
    const result = execSync(`git status --porcelain -- ${args.map(d => `"${d}"`).join(' ')}`, {
      cwd: repoPath,
      stdio: 'pipe',
      encoding: 'utf8',
    });
    return result.trim().split('\n').filter(Boolean).map(line => line.slice(3));
  } catch {
    return [];
  }
}

function gitStashFiles(repoPath, files) {
  try {
    const fileArgs = files.map(f => `"${f}"`).join(' ');
    execSync(`git stash push -m "aether-update-backup" -- ${fileArgs}`, {
      cwd: repoPath,
      stdio: 'pipe',
    });
    return true;
  } catch (err) {
    log(`  Warning: git stash failed (${err.message}). Proceeding without stash.`);
    return false;
  }
}

// Directories to exclude from hub sync (user data, local state)
const HUB_EXCLUDE_DIRS = ['data', 'dreams', 'checkpoints', 'locks', 'temp'];

/**
 * Check if a path should be excluded from hub sync
 * @param {string} relPath - Relative path from .aether/
 * @returns {boolean} True if should be excluded
 */
function shouldExcludeFromHub(relPath) {
  const parts = relPath.split(path.sep);
  // Exclude if any part of the path is in the exclude list
  return parts.some(part => HUB_EXCLUDE_DIRS.includes(part));
}

/**
 * Sync .aether/ directory to hub, excluding user data directories
 * @param {string} srcDir - Source .aether/ directory
 * @param {string} destDir - Destination hub directory
 * @returns {object} Sync result with copied, removed, skipped counts
 */
function syncAetherToHub(srcDir, destDir) {
  if (!fs.existsSync(srcDir)) {
    return { copied: 0, removed: 0, skipped: 0 };
  }

  fs.mkdirSync(destDir, { recursive: true });

  // Get all files in source, filtering out excluded directories
  const srcFiles = [];
  function collectFiles(dir, base) {
    const entries = fs.readdirSync(dir, { withFileTypes: true });
    for (const entry of entries) {
      if (entry.name.startsWith('.')) continue;
      const fullPath = path.join(dir, entry.name);
      const relPath = path.relative(base, fullPath);

      if (shouldExcludeFromHub(relPath)) continue;

      if (entry.isDirectory()) {
        collectFiles(fullPath, base);
      } else {
        srcFiles.push(relPath);
      }
    }
  }
  collectFiles(srcDir, srcDir);

  // Copy files with hash comparison
  let copied = 0;
  let skipped = 0;
  for (const relPath of srcFiles) {
    const srcPath = path.join(srcDir, relPath);
    const destPath = path.join(destDir, relPath);

    fs.mkdirSync(path.dirname(destPath), { recursive: true });

    // Hash comparison
    let shouldCopy = true;
    if (fs.existsSync(destPath)) {
      const srcHash = hashFileSync(srcPath);
      const destHash = hashFileSync(destPath);
      if (srcHash === destHash) {
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
  }

  // Cleanup: remove files in dest that aren't in source (and aren't excluded)
  const destFiles = [];
  function collectDestFiles(dir, base) {
    if (!fs.existsSync(dir)) return;
    const entries = fs.readdirSync(dir, { withFileTypes: true });
    for (const entry of entries) {
      if (entry.name.startsWith('.') || entry.name === 'registry.json' || entry.name === 'version.json' || entry.name === 'manifest.json') continue;
      const fullPath = path.join(dir, entry.name);
      const relPath = path.relative(base, fullPath);

      if (shouldExcludeFromHub(relPath)) continue;

      if (entry.isDirectory()) {
        collectDestFiles(fullPath, base);
      } else {
        destFiles.push(relPath);
      }
    }
  }
  collectDestFiles(destDir, destDir);

  const srcSet = new Set(srcFiles);
  const removed = [];
  for (const relPath of destFiles) {
    if (!srcSet.has(relPath)) {
      removed.push(relPath);
      try {
        fs.unlinkSync(path.join(destDir, relPath));
      } catch (err) {
        // Ignore cleanup errors
      }
    }
  }

  // Clean up empty directories
  cleanEmptyDirs(destDir);

  return { copied, removed, skipped };
}

function setupHub() {
  // Create ~/.aether/ directory structure and populate from package
  try {
    fs.mkdirSync(HUB_DIR, { recursive: true });

    // MIGRATION: Check for old structure and migrate to system/
    const oldStructureFiles = [
      path.join(HUB_DIR, 'aether-utils.sh'),
      path.join(HUB_DIR, 'workers.md'),
    ];
    const hasOldStructure = oldStructureFiles.some(f => fs.existsSync(f));
    const hasNewStructure = fs.existsSync(HUB_SYSTEM_DIR);

    if (hasOldStructure && !hasNewStructure) {
      log('  Migrating hub to new structure...');
      fs.mkdirSync(HUB_SYSTEM_DIR, { recursive: true });

      // Move system files to system/
      const systemFiles = ['aether-utils.sh', 'workers.md', 'CONTEXT.md', 'model-profiles.yaml',
                          'coding-standards.md', 'debugging.md', 'learning.md', 'planning.md',
                          'tdd.md', 'verification.md', 'verification-loop.md', 'DISCIPLINES.md',
                          'QUEEN_ANT_ARCHITECTURE.md', 'workers-new-castes.md', 'recover.sh'];
      const systemDirs = ['docs', 'utils', 'commands', 'agents', 'schemas', 'exchange', 'templates', 'lib'];

      for (const file of systemFiles) {
        const oldPath = path.join(HUB_DIR, file);
        if (fs.existsSync(oldPath)) {
          fs.renameSync(oldPath, path.join(HUB_SYSTEM_DIR, file));
        }
      }

      for (const dir of systemDirs) {
        const oldPath = path.join(HUB_DIR, dir);
        if (fs.existsSync(oldPath)) {
          fs.renameSync(oldPath, path.join(HUB_SYSTEM_DIR, dir));
        }
      }

      log('  Migration complete: system files moved to ~/.aether/system/');
    }

    // Create system/ directory structure
    fs.mkdirSync(HUB_SYSTEM_DIR, { recursive: true });
    fs.mkdirSync(path.join(HUB_SYSTEM_DIR, 'commands', 'claude'), { recursive: true });
    fs.mkdirSync(path.join(HUB_SYSTEM_DIR, 'commands', 'opencode'), { recursive: true });
    fs.mkdirSync(path.join(HUB_SYSTEM_DIR, 'agents'), { recursive: true });
    fs.mkdirSync(path.join(HUB_SYSTEM_DIR, 'rules'), { recursive: true });

    // Read previous manifest for delta reporting
    const prevManifestRaw = readJsonSafe(path.join(HUB_DIR, 'manifest.json'));
    const prevManifest = prevManifestRaw && validateManifest(prevManifestRaw).valid ? prevManifestRaw : null;
    if (prevManifestRaw && !prevManifest) {
      log(`  Warning: previous manifest is invalid, regenerating`);
    }

    // Sync runtime/ -> ~/.aether/system/ (clean production files)
    // runtime/ is generated during publish - explicit allowlist via sync-to-runtime.sh
    const runtimeSrc = path.join(PACKAGE_DIR, 'runtime');
    if (fs.existsSync(runtimeSrc)) {
      const result = syncAetherToHub(runtimeSrc, HUB_SYSTEM_DIR);
      log(`  Hub system: ${result.copied} files, ${result.skipped} unchanged -> ${HUB_SYSTEM_DIR}`);
      if (result.removed.length > 0) {
        log(`  Hub system: removed ${result.removed.length} stale files`);
        for (const f of result.removed) log(`    - ${f}`);
      }
    }

    // Clean up legacy directories from very old hub structure (pre-system/)
    const legacyDirs = [
      path.join(HUB_DIR, '.aether'),
      path.join(HUB_DIR, 'visualizations'),
    ];
    for (const legacyDir of legacyDirs) {
      if (fs.existsSync(legacyDir)) {
        try {
          removeDirSync(legacyDir);
          log(`  Cleaned up legacy: ${path.basename(legacyDir)}/`);
        } catch (err) {
          // Ignore cleanup errors
        }
      }
    }

    // Sync .claude/commands/ant/ -> ~/.aether/system/commands/claude/
    const claudeCmdSrc = fs.existsSync(COMMANDS_SRC)
      ? COMMANDS_SRC
      : path.join(PACKAGE_DIR, '.claude', 'commands', 'ant');
    if (fs.existsSync(claudeCmdSrc)) {
      const result = syncDirWithCleanup(claudeCmdSrc, HUB_COMMANDS_CLAUDE);
      log(`  Hub commands (claude): ${result.copied} files -> ${HUB_COMMANDS_CLAUDE}`);
      if (result.removed.length > 0) {
        log(`  Hub commands (claude): removed ${result.removed.length} stale files`);
        for (const f of result.removed) log(`    - ${f}`);
      }
    }

    // Sync .opencode/commands/ant/ -> ~/.aether/system/commands/opencode/
    const opencodeCmdSrc = path.join(PACKAGE_DIR, '.opencode', 'commands', 'ant');
    if (fs.existsSync(opencodeCmdSrc)) {
      const result = syncDirWithCleanup(opencodeCmdSrc, HUB_COMMANDS_OPENCODE);
      log(`  Hub commands (opencode): ${result.copied} files -> ${HUB_COMMANDS_OPENCODE}`);
      if (result.removed.length > 0) {
        log(`  Hub commands (opencode): removed ${result.removed.length} stale files`);
        for (const f of result.removed) log(`    - ${f}`);
      }
    }

    // Sync .opencode/agents/ -> ~/.aether/system/agents/
    const agentsSrc = path.join(PACKAGE_DIR, '.opencode', 'agents');
    if (fs.existsSync(agentsSrc)) {
      const result = syncDirWithCleanup(agentsSrc, HUB_AGENTS);
      log(`  Hub agents: ${result.copied} files -> ${HUB_AGENTS}`);
      if (result.removed.length > 0) {
        log(`  Hub agents: removed ${result.removed.length} stale files`);
        for (const f of result.removed) log(`    - ${f}`);
      }
    }

    // Sync rules/ from runtime -> ~/.aether/system/rules/
    const rulesSrc = path.join(PACKAGE_DIR, 'runtime', 'rules');
    if (fs.existsSync(rulesSrc)) {
      const result = syncDirWithCleanup(rulesSrc, HUB_RULES);
      log(`  Hub rules: ${result.copied} files -> ${HUB_RULES}`);
      if (result.removed.length > 0) {
        log(`  Hub rules: removed ${result.removed.length} stale files`);
        for (const f of result.removed) log(`    - ${f}`);
      }
    }

    // Create/preserve registry.json (at root, not in system/)
    if (!fs.existsSync(HUB_REGISTRY)) {
      writeJsonSync(HUB_REGISTRY, { schema_version: 1, repos: [] });
      log(`  Registry: initialized ${HUB_REGISTRY}`);
    } else {
      log(`  Registry: preserved existing ${HUB_REGISTRY}`);
    }

    // Generate and write manifest (at root, tracks everything)
    const manifest = generateManifest(HUB_DIR);
    const manifestPath = path.join(HUB_DIR, 'manifest.json');
    writeJsonSync(manifestPath, manifest);
    const fileCount = Object.keys(manifest.files).length;
    log(`  Manifest: ${fileCount} files tracked`);

    // Report manifest delta
    if (prevManifest && prevManifest.files) {
      const prevKeys = new Set(Object.keys(prevManifest.files));
      const currKeys = new Set(Object.keys(manifest.files));
      const added = [...currKeys].filter(k => !prevKeys.has(k));
      const removed = [...prevKeys].filter(k => !currKeys.has(k));
      const changed = [...currKeys].filter(k => prevKeys.has(k) && prevManifest.files[k] !== manifest.files[k]);
      if (added.length || removed.length || changed.length) {
        log(`  Manifest delta: +${added.length} added, -${removed.length} removed, ~${changed.length} changed`);
      }
    }

    // Write version.json (at root)
    writeJsonSync(HUB_VERSION, { version: VERSION, updated_at: new Date().toISOString() });
    log(`  Hub version: ${VERSION}`);
  } catch (err) {
    // Hub setup failure doesn't block install
    log(`  Hub setup warning: ${err.message}`);
  }
}

async function updateRepo(repoPath, sourceVersion, opts) {
  opts = opts || {};
  const dryRun = opts.dryRun || false;
  const force = opts.force || false;
  const quiet = opts.quiet || false;

  const repoAether = path.join(repoPath, '.aether');
  const repoVersionFile = path.join(repoAether, 'version.json');

  if (!fs.existsSync(repoAether)) {
    return { status: 'skipped', reason: 'no .aether directory' };
  }

  const currentVersion = readJsonSafe(repoVersionFile);
  const currentVer = currentVersion ? currentVersion.version : 'unknown';

  // Target directories for git safety checks
  const targetDirs = ['.aether', '.claude/commands/ant', '.claude/rules', '.opencode/commands/ant', '.opencode/agents'];

  // Git safety: check for dirty files in target directories (skip in dry-run mode)
  let dirtyFiles = [];
  if (isGitRepo(repoPath)) {
    dirtyFiles = getGitDirtyFiles(repoPath, targetDirs);
    if (dirtyFiles.length > 0 && !force) {
      return { status: 'dirty', files: dirtyFiles };
    }
    // Note: --force handling is now done via checkpoint stash in UpdateTransaction
  }

  // Use UpdateTransaction for two-phase commit with automatic rollback
  const transaction = new UpdateTransaction(repoPath, { sourceVersion, quiet, force });

  try {
    const result = await transaction.execute(sourceVersion, { dryRun });

    // Calculate file counts from sync result
    const systemCopied = result.sync_result?.system?.copied || 0;
    const commandsCopied = (result.sync_result?.commands?.copied || 0);
    const agentsCopied = result.sync_result?.agents?.copied || 0;
    const rulesCopied = result.sync_result?.rules?.copied || 0;

    const systemRemoved = result.sync_result?.system?.removed?.length || 0;
    const commandsRemoved = result.sync_result?.commands?.removed?.length || 0;
    const agentsRemoved = result.sync_result?.agents?.removed?.length || 0;
    const rulesRemoved = result.sync_result?.rules?.removed?.length || 0;

    const allRemovedFiles = [
      ...(result.sync_result?.system?.removed || []),
      ...(result.sync_result?.commands?.removed || []).map(f => `.claude/commands/ant/${f}`),
      ...(result.sync_result?.agents?.removed || []).map(f => `.opencode/agents/${f}`),
      ...(result.sync_result?.rules?.removed || []).map(f => `.claude/rules/${f}`),
    ];

    return {
      status: result.status,
      from: currentVer,
      to: sourceVersion,
      system: systemCopied,
      commands: commandsCopied,
      agents: agentsCopied,
      rules: rulesCopied,
      removed: systemRemoved + commandsRemoved + agentsRemoved + rulesRemoved,
      removedFiles: allRemovedFiles,
      stashCreated: !!transaction.checkpoint?.stashRef,
      checkpoint_id: result.checkpoint_id,
    };
  } catch (error) {
    // Handle UpdateError with recovery commands
    if (error instanceof UpdateError) {
      // Re-throw with additional context
      error.details = {
        ...error.details,
        repoPath,
        sourceVersion,
        from: currentVer,
      };
    }
    throw error;
  }
}

// Commander.js program setup
program
  .name('aether')
  .description('Aether Colony - Multi-agent system using ant colony intelligence')
  .version(VERSION, '-v, --version', 'show version')
  .option('--no-color', 'disable colored output')
  .option('-q, --quiet', 'suppress output')
  .helpOption('-h, --help', 'show help');

// Handle --no-color globally
program.on('option:no-color', () => {
  process.env.NO_COLOR = '1';
});

// Handle --quiet globally
program.on('option:quiet', () => {
  globalQuiet = true;
});

// Install command
program
  .command('install')
  .description('Install slash-commands to ~/.claude/commands/ant/ and set up distribution hub')
  .action(wrapCommand(async () => {
    log(c.header(`aether-colony v${VERSION} — installing...`));

    // Sync commands to ~/.claude/commands/ant/ (with orphan cleanup)
    if (!fs.existsSync(COMMANDS_SRC)) {
      // Running from source repo — commands are in .claude/commands/ant/
      const repoCommands = path.join(PACKAGE_DIR, '.claude', 'commands', 'ant');
      if (fs.existsSync(repoCommands)) {
        const result = syncDirWithCleanup(repoCommands, COMMANDS_DEST);
        log(`  Commands: ${result.copied} files -> ${COMMANDS_DEST}`);
        if (result.removed.length > 0) {
          log(`  Commands: removed ${result.removed.length} stale files`);
          for (const f of result.removed) log(`    - ${f}`);
        }
      } else {
        console.error('  Commands source not found. Skipping.');
      }
    } else {
      const result = syncDirWithCleanup(COMMANDS_SRC, COMMANDS_DEST);
      log(`  Commands: ${result.copied} files -> ${COMMANDS_DEST}`);
      if (result.removed.length > 0) {
        log(`  Commands: removed ${result.removed.length} stale files`);
        for (const f of result.removed) log(`    - ${f}`);
      }
    }

    // Set up distribution hub at ~/.aether/
    log('');
    log(c.colony('Setting up distribution hub...'));
    setupHub();

    log('');
    log(c.success('Install complete.'));
    log(`  ${c.queen('Claude Code:')} run /ant to get started`);
    log(`  ${c.colony('Hub:')} ${c.dim('~/.aether/')} (for coordinated updates across repos)`);
  }));

// Update command
program
  .command('update')
  .description('Update current repo from hub (use --all to update all registered repos)')
  .option('-f, --force', 'stash dirty files and force update')
  .option('-a, --all', 'update all registered repos')
  .option('-l, --list', 'show registered repos and versions')
  .option('-d, --dry-run', 'preview what would change without modifying files')
  .action(wrapCommand(async (options) => {
    const forceFlag = options.force || false;
    const allFlag = options.all || false;
    const listFlag = options.list || false;
    const dryRun = options.dryRun || false;

    // Check hub exists
    if (!fs.existsSync(HUB_VERSION)) {
      const error = new HubError(
        'No distribution hub found at ~/.aether/',
        { path: HUB_DIR }
      );
      logError(error);
      console.error(JSON.stringify(error.toJSON(), null, 2));
      process.exit(getExitCode(error.code));
    }

    const hubVersion = readJsonSafe(HUB_VERSION);
    const sourceVersion = hubVersion ? hubVersion.version : VERSION;

    if (listFlag) {
      // Show registered repos
      const registry = readJsonSafe(HUB_REGISTRY);
      if (!registry || registry.repos.length === 0) {
        console.log(c.info('No repos registered. Run the Claude Code slash command /ant:init in a repo to register it.'));
        return;
      }
      console.log(c.header(`Registered repos (hub v${sourceVersion}):\n`));
      for (const repo of registry.repos) {
        const exists = fs.existsSync(repo.path);
        const status = exists ? `v${repo.version}` : 'NOT FOUND';
        const marker = exists ? (repo.version === sourceVersion ? '  ' : '* ') : 'x ';
        console.log(`${marker}${repo.path}  (${status})`);
      }
      console.log('');
      console.log(c.dim('* = update available, x = path no longer exists'));
      return;
    }

    if (allFlag) {
      // Update all registered repos
      const registry = readJsonSafe(HUB_REGISTRY);
      if (!registry || registry.repos.length === 0) {
        console.log(c.info('No repos registered. Run the Claude Code slash command /ant:init in a repo to register it.'));
        return;
      }

      let updated = 0;
      let upToDate = 0;
      let pruned = 0;
      let dirty = 0;
      let totalRemoved = 0;
      const survivingRepos = [];

      if (dryRun) {
        console.log(c.warning('Dry run — no files will be modified.\n'));
      }

      for (const repo of registry.repos) {
        if (!fs.existsSync(repo.path)) {
          log(`  ${c.warning('Pruned:')} ${repo.path} (no longer exists)`);
          pruned++;
          continue;
        }

        survivingRepos.push(repo);

        if (!forceFlag && !dryRun && repo.version === sourceVersion) {
          log(`  Up-to-date: ${repo.path} (v${repo.version})`);
          upToDate++;
          continue;
        }

        try {
          const result = await updateRepo(repo.path, sourceVersion, { dryRun, force: forceFlag, quiet: true });
          if (result.status === 'dirty') {
            console.error(`  ${c.error('Dirty:')} ${repo.path} — uncommitted changes in managed files:`);
            for (const f of result.files) console.error(`    ${f}`);
            console.error(`  Skipping. Use --force to stash and update.`);
            dirty++;
          } else if (result.status === 'dry-run') {
            log(`  Would update: ${repo.path} (${result.from} -> ${result.to}) [${result.system} system, ${result.commands} commands, ${result.agents} agents]`);
            if (result.removed > 0) {
              log(`  Would remove ${result.removed} stale files:`);
              for (const f of result.removedFiles) log(`    - ${f}`);
            }
            updated++;
          } else if (result.status === 'updated') {
            log(`  ${c.success('Updated:')} ${repo.path} (${result.from} -> ${result.to}) [${result.system} system, ${result.commands} commands, ${result.agents} agents]`);
            if (result.removed > 0) {
              log(`  Removed ${result.removed} stale files:`);
              for (const f of result.removedFiles) log(`    - ${f}`);
              totalRemoved += result.removed;
            }
            if (result.stashCreated) {
              log(`  Stash created. Recover with: cd ${repo.path} && git stash pop`);
            }
            updated++;
          } else {
            log(`  Skipped: ${repo.path} (${result.reason})`);
          }
        } catch (error) {
          // Handle UpdateError with formatted recovery commands
          if (error instanceof UpdateError) {
            console.error(formatUpdateError(error));
          }
          throw error;
        }
      }

      // Save pruned registry
      if (pruned > 0 && !dryRun) {
        registry.repos = survivingRepos;
        writeJsonSync(HUB_REGISTRY, registry);
      }

      const label = dryRun ? 'would update' : 'updated';
      let summary = `\nSummary: ${updated} ${label}, ${upToDate} up-to-date, ${pruned} pruned`;
      if (dirty > 0) summary += `, ${dirty} dirty (skipped)`;
      if (totalRemoved > 0) summary += `, ${totalRemoved} stale files removed`;
      console.log(summary);
    } else {
      // Update current repo
      const repoPath = process.cwd();
      const repoAether = path.join(repoPath, '.aether');

      if (!fs.existsSync(repoAether)) {
        const error = new RepoError(
          'No .aether/ directory found in current repo.',
          { path: repoPath }
        );
        logError(error);
        console.error(JSON.stringify(error.toJSON(), null, 2));
        process.exit(getExitCode(error.code));
      }

      const currentVersion = readJsonSafe(path.join(repoAether, 'version.json'));
      const currentVer = currentVersion ? currentVersion.version : 'unknown';

      if (!forceFlag && !dryRun && currentVer === sourceVersion) {
        console.log(c.info(`Already up-to-date (v${sourceVersion}).`));
        return;
      }

      if (dryRun) {
        console.log(c.warning('Dry run — no files will be modified.\n'));
      }

      try {
        const result = await updateRepo(repoPath, sourceVersion, { dryRun, force: forceFlag });

        if (result.status === 'dirty') {
          const error = new GitError(
            'Uncommitted changes in managed files',
            { files: result.files, repo: repoPath }
          );
          logError(error);
          console.error(JSON.stringify(error.toJSON(), null, 2));
          console.error('\nUse --force to stash changes and update, or commit/stash manually first.');
          process.exit(getExitCode(error.code));
        }

        if (result.status === 'dry-run') {
          console.log(`Would update: ${result.from} -> ${result.to}`);
          console.log(`  ${result.system} system files, ${result.commands} command files, ${result.agents} agent files`);
          if (result.removed > 0) {
            console.log(`  Would remove ${result.removed} stale files:`);
            for (const f of result.removedFiles) console.log(`    - ${f}`);
          }
          console.log('  Colony data (.aether/data/) untouched.');
          return;
        }

        console.log(c.success(`Updated: ${result.from} -> ${result.to}`));
        console.log(`  ${result.system} system files, ${result.commands} command files, ${result.agents} agent files, ${result.visualizations} visualization files`);
        if (result.removed > 0) {
          console.log(`  Removed ${result.removed} stale files:`);
          for (const f of result.removedFiles) console.log(`    - ${f}`);
        }
        if (result.stashCreated) {
          console.log('  Git stash created. Recover with: git stash pop');
        }
        if (result.checkpoint_id) {
          console.log(`  Checkpoint: ${result.checkpoint_id}`);
        }
        console.log('  Colony data (.aether/data/) untouched.');
      } catch (error) {
        // Handle UpdateError with prominent recovery commands (UPDATE-04)
        if (error instanceof UpdateError) {
          console.error(`\n${c.error('========================================')}`);
          console.error(c.error('UPDATE FAILED - RECOVERY REQUIRED'));
          console.error(c.error('========================================'));
          console.error(`\n${c.error('Error:')} ${error.message}`);
          if (error.details?.checkpoint_id) {
            console.error(`\nCheckpoint ID: ${error.details.checkpoint_id}`);
          }
          console.error(`\n${c.warning('To recover your workspace:')}`);
          for (const cmd of error.recoveryCommands) {
            console.error(`  ${cmd}`);
          }
          console.error(c.error('========================================'));

          // Log to activity log
          logError(error);

          // Output structured JSON to stderr
          console.error(JSON.stringify(error.toJSON(), null, 2));
          process.exit(getExitCode(error.code) || 1);
        }
        throw error;
      }
    }
  }));

// Version command
program
  .command('version')
  .description('Show installed version and hub status')
  .action(() => {
    console.log(c.header(`aether-colony v${VERSION}`));
  });

// Uninstall command
program
  .command('uninstall')
  .description('Remove slash-commands from ~/.claude/commands/ant/ (preserves project state and hub)')
  .action(wrapCommand(async () => {
    log(c.header(`aether-colony v${VERSION} — uninstalling...`));

    // Remove Claude Code commands
    if (fs.existsSync(COMMANDS_DEST)) {
      const n = removeDirSync(COMMANDS_DEST);
      log(`  Removed: ${n} command files from ${COMMANDS_DEST}`);
    } else {
      log('  Claude Code commands already removed.');
    }

    log('');
    log(c.success('Uninstall complete. Per-project .aether/data/ directories are untouched.'));
    log(`  ${c.colony('Hub:')} ${c.dim('~/.aether/')} preserved (remove manually if desired).`);
  }));

// Checkpoint command
program
  .command('checkpoint')
  .description('Manage Aether checkpoints (safe snapshots of system files)')
  .addCommand(
    program.createCommand('create')
      .description('Create a new checkpoint of Aether system files')
      .argument('[message]', 'optional message describing the checkpoint')
      .action(wrapCommand(async (message) => {
        const repoPath = process.cwd();

        // 1. Check if in git repo
        if (!isGitRepo(repoPath)) {
          console.error(c.error('Error: Not in a git repository'));
          process.exit(1);
        }

        // 2. Get allowlisted files using CHECKPOINT_ALLOWLIST
        const allowlistedFiles = getAllowlistedFiles(repoPath);
        if (allowlistedFiles.length === 0) {
          console.log(c.warning('No allowlisted files found to checkpoint'));
          return;
        }

        // 3. Verify no user data in allowlist (safety check)
        const userDataFiles = allowlistedFiles.filter(f => isUserData(f));
        if (userDataFiles.length > 0) {
          console.error(c.error('Safety check failed: user data detected in allowlist:'));
          for (const f of userDataFiles) console.error(`  - ${f}`);
          process.exit(1);
        }

        // 4. Generate checkpoint metadata with hashes
        const metadata = generateCheckpointMetadata(repoPath, message);

        // 5. Create git stash with allowlisted files
        // Command format: git stash push -m "aether-checkpoint-{timestamp}" -- {files}
        const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
        const stashMessage = `aether-checkpoint-${timestamp}`;
        const fileArgs = allowlistedFiles.map(f => `"${f}"`).join(' ');

        try {
          execSync(`git stash push -m "${stashMessage}" -- ${fileArgs}`, {
            cwd: repoPath,
            stdio: 'pipe'
          });
        } catch (err) {
          console.error(c.error(`Failed to create git stash: ${err.message}`));
          process.exit(1);
        }

        // 6. Save metadata to .aether/checkpoints/
        saveCheckpointMetadata(repoPath, metadata);

        // 7. Output success with checkpoint ID
        console.log(c.success(`Checkpoint created: ${metadata.checkpoint_id}`));
        console.log(`  Files: ${Object.keys(metadata.files).length}`);
        console.log(`  Stash: ${stashMessage}`);
        if (message) console.log(`  Message: ${message}`);
      }))
  )
  .addCommand(
    program.createCommand('list')
      .description('List all checkpoints')
      .action(wrapCommand(async () => {
        const repoPath = process.cwd();
        const checkpointsDir = path.join(repoPath, '.aether', 'checkpoints');

        if (!fs.existsSync(checkpointsDir)) {
          console.log(c.info('No checkpoints found'));
          return;
        }

        const files = fs.readdirSync(checkpointsDir)
          .filter(f => f.endsWith('.json'))
          .sort();

        if (files.length === 0) {
          console.log(c.info('No checkpoints found'));
          return;
        }

        console.log(c.header('Checkpoints:'));
        for (const file of files) {
          const metadata = loadCheckpointMetadata(repoPath, file.replace('.json', ''));
          if (metadata) {
            const fileCount = Object.keys(metadata.files).length;
            const date = new Date(metadata.created_at).toLocaleString();
            console.log(`  ${metadata.checkpoint_id}  ${date}  ${fileCount} files  ${metadata.message || ''}`);
          }
        }
      }))
  )
  .addCommand(
    program.createCommand('restore')
      .description('Restore Aether files from a checkpoint')
      .argument('<checkpoint-id>', 'checkpoint ID to restore from')
      .action(wrapCommand(async (checkpointId) => {
        const repoPath = process.cwd();

        // 1. Load checkpoint metadata
        const metadata = loadCheckpointMetadata(repoPath, checkpointId);
        if (!metadata) {
          console.error(c.error(`Checkpoint not found: ${checkpointId}`));
          process.exit(1);
        }

        // 2. Verify metadata integrity (hashes match current files if they exist)
        let integrityCheck = true;
        for (const [filePath, storedHash] of Object.entries(metadata.files)) {
          const fullPath = path.join(repoPath, filePath);
          if (fs.existsSync(fullPath)) {
            const currentHash = hashFileSync(fullPath);
            if (currentHash !== storedHash) {
              console.warn(c.warning(`File changed since checkpoint: ${filePath}`));
              integrityCheck = false;
            }
          }
        }

        // 3. Use git stash to restore files
        // Find stash by message pattern
        try {
          const stashList = execSync('git stash list', { cwd: repoPath, encoding: 'utf8' });
          const stashMatch = stashList.match(/stash@\{([^}]+)\}.*aether-checkpoint-/);
          if (stashMatch) {
            execSync(`git stash pop stash@{${stashMatch[1]}}`, {
              cwd: repoPath,
              stdio: 'pipe'
            });
            console.log(c.success(`Restored from checkpoint: ${checkpointId}`));
            console.log(`  Files restored: ${Object.keys(metadata.files).length}`);
          } else {
            console.error(c.error('Could not find matching git stash'));
            process.exit(1);
          }
        } catch (err) {
          console.error(c.error(`Failed to restore checkpoint: ${err.message}`));
          process.exit(1);
        }
      }))
  )
  .addCommand(
    program.createCommand('verify')
      .description('Verify checkpoint integrity')
      .argument('<checkpoint-id>', 'checkpoint ID to verify')
      .action(wrapCommand(async (checkpointId) => {
        const repoPath = process.cwd();

        // 1. Load checkpoint metadata
        const metadata = loadCheckpointMetadata(repoPath, checkpointId);
        if (!metadata) {
          console.error(c.error(`Checkpoint not found: ${checkpointId}`));
          process.exit(1);
        }

        // 2. Re-compute hashes for all files in metadata
        // 3. Compare with stored hashes
        let passed = 0;
        let failed = 0;
        let missing = 0;

        for (const [filePath, storedHash] of Object.entries(metadata.files)) {
          const fullPath = path.join(repoPath, filePath);
          if (!fs.existsSync(fullPath)) {
            console.log(c.error(`  MISSING: ${filePath}`));
            missing++;
          } else {
            const currentHash = hashFileSync(fullPath);
            if (currentHash === storedHash) {
              console.log(c.success(`  OK: ${filePath}`));
              passed++;
            } else {
              console.log(c.error(`  MISMATCH: ${filePath}`));
              failed++;
            }
          }
        }

        // 4. Report any mismatches
        console.log('');
        if (failed === 0 && missing === 0) {
          console.log(c.success(`All ${passed} files verified successfully`));
        } else {
          console.log(c.warning(`Verification complete: ${passed} passed, ${failed} mismatched, ${missing} missing`));
          process.exit(1);
        }
      }))
  );

// Sync-state command - Synchronize COLONY_STATE.json with .planning/STATE.md
program
  .command('sync-state')
  .description('Synchronize COLONY_STATE.json with .planning/STATE.md')
  .option('-d, --dry-run', 'Show what would change without applying')
  .action(wrapCommand(async (options) => {
    const repoPath = process.cwd();

    if (!isInitialized(repoPath)) {
      console.error('Aether not initialized. Run: aether init');
      return;
    }

    // Check for mismatches
    const reconciliation = reconcileStates(repoPath);
    if (!reconciliation.consistent) {
      console.log('State mismatch detected:');
      reconciliation.mismatches.forEach(m => console.log(`  - ${m}`));
      console.log('');
    }

    if (options.dryRun) {
      console.log('Dry run - no changes made');
      console.log(`Resolution: ${reconciliation.resolution}`);
      return;
    }

    // Perform sync
    const result = syncStateFromPlanning(repoPath);
    if (result.synced) {
      if (result.changed) {
        console.log('State synchronized successfully');
        console.log('Updates:', result.updates.join(', '));
      } else {
        console.log('State already synchronized - no changes needed');
      }
    } else {
      console.error(`Sync failed: ${result.error}`);
      process.exit(1);
    }
  }));

// Caste emoji mapping for display
const CASTE_EMOJIS = {
  builder: '🔨',
  watcher: '👁️',
  scout: '🔍',
  chaos: '🎲',
  oracle: '🔮',
  architect: '🏗️',
  prime: '🏛️',
  colonizer: '🌱',
  route_setter: '🧭',
  archaeologist: '📜',
};

/**
 * Format context window for display
 * @param {number} contextWindow - Context window size
 * @returns {string} Formatted string (e.g., "256K")
 */
function formatContextWindow(contextWindow) {
  if (!contextWindow) return '-';
  if (contextWindow >= 1000) {
    return `${Math.round(contextWindow / 1000)}K`;
  }
  return String(contextWindow);
}

// Caste-models command - Manage caste-to-model assignments
const casteModelsCmd = program
  .command('caste-models')
  .description('Manage caste-to-model assignments');

// list subcommand
casteModelsCmd
  .command('list')
  .description('List current model assignments per caste')
  .option('--verify', 'Verify model availability on proxy')
  .action(wrapCommand(async (options) => {
    const repoPath = process.cwd();
    const profiles = loadModelProfiles(repoPath);
    const overrides = getUserOverrides(profiles);
    const proxyConfig = getProxyConfig(profiles);

    // Check proxy health
    let proxyHealth = null;
    let proxyModels = null;
    if (proxyConfig?.endpoint) {
      proxyHealth = await checkProxyHealth(proxyConfig.endpoint);
      if (proxyHealth.healthy && proxyHealth.models) {
        proxyModels = proxyHealth.models;
      }
    }

    console.log(c.header('Caste Model Assignments\n'));

    // Display proxy status
    if (proxyConfig?.endpoint) {
      const proxyStatus = formatProxyStatusColored(proxyHealth, c) + c.dim(` @ ${proxyConfig.endpoint}`);
      console.log(`Proxy: ${proxyStatus}`);
      if (!proxyHealth?.healthy) {
        console.log(c.warning('Warning: Using default model (kimi-k2.5) for all castes'));
      }
      console.log('');
    }

    // Table header - add Verify column if --verify flag
    const verifyFlag = options.verify;
    const header = verifyFlag
      ? `${'Caste'.padEnd(14)} ${'Model'.padEnd(14)} ${'Provider'.padEnd(10)} ${'Context'.padEnd(8)} Verify Status`
      : `${'Caste'.padEnd(14)} ${'Model'.padEnd(14)} ${'Provider'.padEnd(10)} ${'Context'.padEnd(8)} Status`;
    console.log(header);
    console.log(verifyFlag ? '─'.repeat(70) : '─'.repeat(60));

    // Get all assignments
    const assignments = getAllAssignments(profiles);

    for (const assignment of assignments) {
      const emoji = CASTE_EMOJIS[assignment.caste] || '•';
      const casteName = assignment.caste.charAt(0).toUpperCase() + assignment.caste.slice(1);
      const casteDisplay = `${emoji} ${casteName}`;

      // Check for override
      const hasOverride = overrides[assignment.caste] !== undefined;
      const effectiveModel = getEffectiveModel(profiles, assignment.caste);
      const modelDisplay = effectiveModel.model + (hasOverride ? ' (override)' : '');

      // Get model metadata
      const metadata = getModelMetadata(profiles, effectiveModel.model);
      const provider = metadata?.provider || assignment.provider || '-';
      const contextWindow = formatContextWindow(metadata?.context_window);

      // Status indicator based on proxy health
      const status = proxyHealth?.healthy ? '✓' : '⚠';

      // Verify flag - check if model is available on proxy
      let verifyStatus = '';
      if (verifyFlag) {
        if (proxyModels) {
          const isAvailable = proxyModels.includes(effectiveModel.model);
          verifyStatus = isAvailable ? '✓' : '✗';
        } else {
          verifyStatus = '?';
        }
        console.log(
          `${casteDisplay.padEnd(14)} ${modelDisplay.padEnd(14)} ${provider.padEnd(10)} ${contextWindow.padEnd(8)} ${verifyStatus.padEnd(7)} ${status}`
        );
      } else {
        console.log(
          `${casteDisplay.padEnd(14)} ${modelDisplay.padEnd(14)} ${provider.padEnd(10)} ${contextWindow.padEnd(8)} ${status}`
        );
      }
    }

    // Show overrides summary if any exist
    const overrideCount = Object.keys(overrides).length;
    if (overrideCount > 0) {
      console.log('');
      console.log(c.info(`Active overrides: ${overrideCount}`));
      for (const [caste, model] of Object.entries(overrides)) {
        console.log(`  ${caste}: ${model}`);
      }
    }
  }));

// set subcommand
casteModelsCmd
  .command('set')
  .description('Set model override for a caste')
  .argument('<assignment>', 'caste=model (e.g., builder=glm-5)')
  .action(wrapCommand(async (assignment) => {
    // Parse caste=model format
    const match = assignment.match(/^([^=]+)=(.+)$/);
    if (!match) {
      const error = new ValidationError(
        `Invalid assignment format: '${assignment}'`,
        { received: assignment },
        'Use format: caste=model (e.g., builder=glm-5)'
      );
      throw error;
    }

    const [, caste, model] = match;

    const repoPath = process.cwd();

    // Validate and set
    try {
      const result = setModelOverride(repoPath, caste, model);

      if (result.previous) {
        console.log(c.success(`Updated ${caste}: ${result.previous} → ${model}`));
      } else {
        console.log(c.success(`Set ${caste} to ${model}`));
      }
    } catch (error) {
      if (error.name === 'ValidationError') {
        // Add helpful suggestions
        if (error.details?.validCastes) {
          console.error(c.error(`Error: ${error.message}`));
          console.error('\nValid castes:');
          for (const casteName of error.details.validCastes) {
            const emoji = CASTE_EMOJIS[casteName] || '•';
            console.error(`  ${emoji} ${casteName}`);
          }
        } else if (error.details?.validModels) {
          console.error(c.error(`Error: ${error.message}`));
          console.error('\nValid models:');
          for (const modelName of error.details.validModels) {
            console.error(`  • ${modelName}`);
          }
        }
        process.exit(1);
      }
      throw error;
    }
  }));

// reset subcommand
casteModelsCmd
  .command('reset')
  .description('Reset caste to default model (remove override)')
  .argument('<caste>', 'caste name (e.g., builder)')
  .action(wrapCommand(async (caste) => {
    const repoPath = process.cwd();

    try {
      const result = resetModelOverride(repoPath, caste);

      if (result.hadOverride) {
        console.log(c.success(`Reset ${caste} to default model`));
      } else {
        console.log(c.info(`${caste} was already using default model`));
      }
    } catch (error) {
      if (error.name === 'ValidationError' && error.details?.validCastes) {
        console.error(c.error(`Error: ${error.message}`));
        console.error('\nValid castes:');
        for (const casteName of error.details.validCastes) {
          const emoji = CASTE_EMOJIS[casteName] || '•';
          console.error(`  ${emoji} ${casteName}`);
        }
        process.exit(1);
      }
      throw error;
    }
  }));

// Verify-models command - Verify model routing configuration
program
  .command('verify-models')
  .description('Verify model routing configuration is active')
  .action(wrapCommand(async () => {
    const repoPath = process.cwd();
    const report = await createVerificationReport(repoPath);

    console.log('=== Model Routing Verification ===\n');

    // Proxy status
    console.log(`LiteLLM Proxy: ${report.proxy.running ? '✓ Running' : '✗ Not running'}`);
    if (report.proxy.running) {
      console.log(`  Latency: ${report.proxy.latency}ms`);
    }

    // Environment
    console.log(`\nEnvironment:`);
    console.log(`  ANTHROPIC_MODEL: ${report.env.model || '(not set)'}`);
    console.log(`  ANTHROPIC_BASE_URL: ${report.env.baseUrl || '(not set)'}`);

    // Caste assignments
    console.log(`\nCaste Model Assignments:`);
    for (const [caste, info] of Object.entries(report.castes)) {
      const status = info.assigned ? '✓' : '✗';
      console.log(`  ${status} ${caste}: ${info.model || 'default'}`);
    }

    // Model profiles file
    console.log(`\nModel Profiles File:`);
    if (report.profilesFile.exists) {
      console.log(`  ✓ Found: ${report.profilesFile.path}`);
      const profileCount = Object.keys(report.profilesFile.profiles).length;
      console.log(`  Profiles: ${profileCount} castes configured`);
    } else {
      console.log(`  ✗ Not found: ${report.profilesFile.path}`);
    }

    // Issues
    if (report.issues.length > 0) {
      console.log(`\nIssues Found:`);
      report.issues.forEach(issue => console.log(`  ⚠ ${issue}`));
    }

    // Recommendation
    console.log(`\n${report.recommendation}`);
  }));

// Spawn-log command - Log a worker spawn event
program
  .command('spawn-log')
  .description('Log a worker spawn event')
  .requiredOption('-p, --parent <parent>', 'Parent ant name')
  .requiredOption('-c, --caste <caste>', 'Worker caste')
  .requiredOption('-n, --name <name>', 'Child ant name')
  .requiredOption('-t, --task <task>', 'Task description')
  .requiredOption('-m, --model <model>', 'Model used')
  .option('-s, --status <status>', 'Spawn status', 'spawned')
  .action(wrapCommand(async (options) => {
    const repoPath = process.cwd();
    await logSpawn(repoPath, {
      parent: options.parent,
      caste: options.caste,
      child: options.name,
      task: options.task,
      model: options.model,
      status: options.status
    });
    console.log(`Logged spawn: ${options.parent} → ${options.name} (${options.caste}) [${options.model}]`);
  }));

// Spawn-tree command - Display worker spawn tree
program
  .command('spawn-tree')
  .description('Display worker spawn tree with model information')
  .action(wrapCommand(async () => {
    const repoPath = process.cwd();
    const tree = formatSpawnTree(repoPath);
    console.log(tree);
  }));

// Status command - Show colony status
program
  .command('status')
  .description('Show colony status')
  .option('-j, --json', 'Output as JSON')
  .action(wrapCommand(async (options) => {
    const repoPath = process.cwd();
    const colonyStatePath = path.join(repoPath, '.aether', 'data', 'COLONY_STATE.json');

    // Check if colony exists
    if (!fs.existsSync(colonyStatePath)) {
      console.log(c.warning('No colony found in current directory.'));
      console.log(c.dim('Run /ant:init to create a new colony, or cd to a project with an existing colony.'));
      return;
    }

    // Load colony state
    let state;
    try {
      state = JSON.parse(fs.readFileSync(colonyStatePath, 'utf8'));
    } catch (err) {
      console.log(c.error('Could not read colony state.'));
      console.log(c.dim(`Error: ${err.message}`));
      console.log(c.dim('The colony state file may be corrupted. Consider running /ant:init to reinitialize.'));
      return;
    }

    // JSON output
    if (options.json) {
      console.log(JSON.stringify(state, null, 2));
      return;
    }

    // Dashboard output
    console.log(c.header('Colony Status\n'));

    // Goal
    if (state.goal) {
      console.log(`${c.queen('Goal:')} ${state.goal}`);
    }

    // State
    if (state.state) {
      const stateDisplay = state.state === 'BUILDING' ? c.success(state.state) :
                          state.state === 'PLANNING' ? c.warning(state.state) :
                          state.state === 'COMPLETED' ? c.success(state.state) :
                          state.state;
      console.log(`${c.colony('State:')} ${stateDisplay}`);
    }

    // Phase
    if (state.current_phase !== undefined) {
      console.log(`${c.info('Phase:')} ${state.current_phase}`);
    }

    // Version
    if (state.version) {
      console.log(`${c.dim('Version:')} ${state.version}`);
    }

    // Last updated
    if (state.last_updated) {
      const lastUpdated = new Date(state.last_updated);
      const now = new Date();
      const hoursAgo = Math.round((now - lastUpdated) / (1000 * 60 * 60));
      const timeDisplay = hoursAgo < 1 ? 'just now' :
                         hoursAgo < 24 ? `${hoursAgo} hours ago` :
                         `${Math.round(hoursAgo / 24)} days ago`;
      console.log(`${c.dim('Last updated:')} ${timeDisplay}`);
    }

    // Event count
    if (state.events && state.events.length > 0) {
      console.log(`${c.dim('Events:')} ${state.events.length}`);
    }

    console.log('');
  }));

// Nestmates command - List sibling colonies
program
  .command('nestmates')
  .description('List sibling colonies (nestmates)')
  .action(wrapCommand(async () => {
    const repoPath = process.cwd();
    const nestmates = findNestmates(repoPath);

    if (nestmates.length === 0) {
      console.log('No nestmates found (sibling directories with .aether/).');
      return;
    }

    console.log(`Found ${nestmates.length} nestmate(s):\n`);
    console.log(formatNestmates(nestmates));
  }));

// Telemetry command - View model performance telemetry
const telemetryCmd = program
  .command('telemetry')
  .description('View model performance telemetry')
  .action(wrapCommand(async () => {
    // Default action: show summary
    const repoPath = process.cwd();
    const summary = getTelemetrySummary(repoPath);

    console.log(c.header('Model Performance Telemetry\n'));
    console.log(`Total Spawns: ${summary.total_spawns}`);
    console.log(`Models Used: ${summary.total_models}\n`);

    if (summary.total_spawns === 0) {
      console.log(c.info('No telemetry data yet. Run some builds to collect data.'));
      return;
    }

    console.log('Model Performance:');
    console.log('─'.repeat(60));
    for (const [model, stats] of Object.entries(summary.models)) {
      const rate = (stats.success_rate * 100).toFixed(1);
      const rateColor = stats.success_rate >= 0.9 ? c.success :
                       stats.success_rate >= 0.7 ? c.warning : c.error;
      console.log(`  ${model.padEnd(15)} ${String(stats.total_spawns).padStart(4)} spawns  ${rateColor(rate + '%')} success`);
    }

    if (summary.recent_decisions.length > 0) {
      console.log('\nRecent Routing Decisions:');
      console.log('─'.repeat(60));
      for (const decision of summary.recent_decisions.slice(-5)) {
        console.log(`  ${decision.caste.padEnd(10)} → ${decision.selected_model.padEnd(12)} (${decision.source})`);
      }
    }
  }));

// summary subcommand (explicit)
telemetryCmd
  .command('summary')
  .description('Show overall telemetry summary')
  .action(wrapCommand(async () => {
    const repoPath = process.cwd();
    const summary = getTelemetrySummary(repoPath);

    console.log(c.header('Model Performance Telemetry\n'));
    console.log(`Total Spawns: ${summary.total_spawns}`);
    console.log(`Models Used: ${summary.total_models}\n`);

    if (summary.total_spawns === 0) {
      console.log(c.info('No telemetry data yet. Run some builds to collect data.'));
      return;
    }

    console.log('Model Performance:');
    console.log('─'.repeat(60));
    for (const [model, stats] of Object.entries(summary.models)) {
      const rate = (stats.success_rate * 100).toFixed(1);
      const rateColor = stats.success_rate >= 0.9 ? c.success :
                       stats.success_rate >= 0.7 ? c.warning : c.error;
      console.log(`  ${model.padEnd(15)} ${String(stats.total_spawns).padStart(4)} spawns  ${rateColor(rate + '%')} success`);
    }

    if (summary.recent_decisions.length > 0) {
      console.log('\nRecent Routing Decisions:');
      console.log('─'.repeat(60));
      for (const decision of summary.recent_decisions.slice(-5)) {
        console.log(`  ${decision.caste.padEnd(10)} → ${decision.selected_model.padEnd(12)} (${decision.source})`);
      }
    }
  }));

// model subcommand
telemetryCmd
  .command('model <model-name>')
  .description('Show detailed performance for a specific model')
  .action(wrapCommand(async (modelName) => {
    const repoPath = process.cwd();
    const performance = getModelPerformance(repoPath, modelName);

    if (!performance) {
      console.log(c.warning(`No data for model: ${modelName}`));
      return;
    }

    console.log(c.header(`Model Performance: ${modelName}\n`));
    console.log(`Total Spawns: ${performance.total_spawns}`);
    console.log(`Success Rate: ${(performance.success_rate * 100).toFixed(1)}%`);
    console.log(`  ✓ Completed: ${performance.successful_completions}`);
    console.log(`  ✗ Failed: ${performance.failed_completions}`);
    console.log(`  🚫 Blocked: ${performance.blocked}`);

    if (Object.keys(performance.by_caste).length > 0) {
      console.log('\nPerformance by Caste:');
      console.log('─'.repeat(50));
      for (const [caste, stats] of Object.entries(performance.by_caste)) {
        const casteRate = stats.spawns > 0 ? (stats.success / stats.spawns * 100).toFixed(1) : '0.0';
        console.log(`  ${caste.padEnd(12)} ${String(stats.spawns).padStart(4)} spawns  ${casteRate}% success`);
      }
    }
  }));

// performance subcommand
telemetryCmd
  .command('performance')
  .description('Show models ranked by performance')
  .action(wrapCommand(async () => {
    const repoPath = process.cwd();
    const summary = getTelemetrySummary(repoPath);

    console.log(c.header('Model Performance Ranking\n'));

    if (summary.total_spawns === 0) {
      console.log(c.info('No telemetry data yet. Run some builds to collect data.'));
      return;
    }

    // Sort models by success rate
    const ranked = Object.entries(summary.models)
      .map(([model, stats]) => ({ model, ...stats }))
      .sort((a, b) => b.success_rate - a.success_rate);

    console.log(`${'Rank'.padEnd(6)} ${'Model'.padEnd(15)} ${'Spawns'.padStart(6)} ${'Success'.padStart(8)} ${'Rate'.padStart(6)}`);
    console.log('─'.repeat(60));

    ranked.forEach((m, i) => {
      const rank = `${i + 1}.`.padEnd(6);
      const rate = (m.success_rate * 100).toFixed(1);
      const rateColor = m.success_rate >= 0.9 ? c.success :
                       m.success_rate >= 0.7 ? c.warning : c.error;
      console.log(`${rank} ${m.model.padEnd(15)} ${String(m.total_spawns).padStart(6)} ${String(m.successful_completions || 0).padStart(8)} ${rateColor(rate.padStart(5) + '%')}`);
    });

    console.log('\n' + c.dim('Tip: Use "aether telemetry model <name>" for detailed stats'));
  }));

// Context command - Show auto-loaded context
program
  .command('context')
  .description('Show auto-loaded context including nestmates')
  .action(wrapCommand(async () => {
    const repoPath = process.cwd();

    // Load nestmates
    const nestmates = findNestmates(repoPath);
    console.log('=== Auto-Loaded Context ===\n');

    // Nestmates
    console.log(`Nestmates: ${nestmates.length} found`);
    if (nestmates.length > 0) {
      console.log(formatNestmates(nestmates));
    }

    // TO-DOs from nestmates
    console.log('\nCross-Project TO-DOs:');
    let hasTodos = false;
    for (const nestmate of nestmates) {
      const todos = loadNestmateTodos(nestmate.path);
      if (todos.length > 0) {
        hasTodos = true;
        console.log(`\n${nestmate.name}:`);
        for (const todo of todos) {
          console.log(`  ${todo.file}:`);
          for (const item of todo.items.slice(0, 5)) {
            console.log(`    ${item}`);
          }
          if (todo.items.length > 5) {
            console.log(`    ... and ${todo.items.length - 5} more`);
          }
        }
      }
    }
    if (!hasTodos) {
      console.log('  No cross-project TO-DOs found.');
    }
  }));

// Init command - Initialize Aether in current repository
program
  .command('init')
  .description('Initialize Aether in current repository')
  .option('-g, --goal <goal>', 'Initial colony goal', 'Aether colony')
  .option('-f, --force', 'Reinitialize even if already initialized')
  .action(wrapCommand(async (options) => {
    const repoPath = process.cwd();

    // Check if already initialized
    if (isInitialized(repoPath) && !options.force) {
      console.log('Aether is already initialized in this repository.');
      console.log('Use --force to reinitialize (WARNING: may overwrite state).');
      return;
    }

    // Initialize
    const result = await initializeRepo(repoPath, {
      goal: options.goal,
      skipIfExists: !options.force
    });

    if (result.success) {
      console.log('Aether initialized successfully!');
      console.log(`State file: ${result.stateFile}`);
      console.log('');
      console.log('Next steps:');
      console.log('  1. Define your colony goal in .aether/data/COLONY_STATE.json');
      console.log('  2. Run: aether sync-state');
      console.log('  3. Run: aether verify-models');
      console.log('  4. Start building: /ant:init');
    }
  }));

// Custom help handler to show CLI vs Slash command distinction
program.on('--help', () => {
  console.log('');
  console.log(c.bold('CLI Commands (Terminal):'));
  console.log('  init                 Initialize Aether in current repository');
  console.log('  install              Install slash-commands and set up distribution hub');
  console.log('  update               Update current repo from hub');
  console.log('  sync-state           Synchronize COLONY_STATE.json with .planning/STATE.md');
  console.log('  verify-models        Verify model routing configuration');
  console.log('  version              Show installed version');
  console.log('  uninstall            Remove slash-commands (preserves project state and hub)');
  console.log('');
  console.log(c.bold('Slash Commands (Claude Code):'));
  console.log('  /ant:init <goal>     Initialize colony in current repo');
  console.log('  /ant:status          Show colony status');
  console.log('  /ant:plan            Generate project plan');
  console.log('  /ant:build <n>       Build phase N');
  console.log('');
  console.log(c.dim('Run these in Claude Code after installing with "aether install"'));
  console.log('');
  console.log(c.bold('Examples:'));
  console.log('  $ aether init --goal "My project"   # Initialize Aether in current repo');
  console.log('  $ aether install                    # Install slash commands');
  console.log('  $ aether update --list              # Show registered repos');
  console.log('  $ aether update --all --force       # Force update all repos');
  console.log('  $ aether --no-color version         # Show version without colors');
});

// Configure error output to use colors
program.configureOutput({
  outputError: (str, write) => write(c.error(str))
});

// Export functions for testing
module.exports = {
  hashFileSync,
  validateManifest,
  generateManifest,
  computeFileHash,
  isGitTracked,
  getAllowlistedFiles,
  generateCheckpointMetadata,
  loadCheckpointMetadata,
  saveCheckpointMetadata,
  isUserData,
  syncDirWithCleanup,
  syncSystemFilesWithCleanup,
  listFilesRecursive,
  cleanEmptyDirs
};

// Parse command line arguments only when run directly (not when required as a module)
if (require.main === module) {
  program.parse();
}
