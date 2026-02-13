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
const HUB_SYSTEM = path.join(HUB_DIR, 'system');
const HUB_COMMANDS_CLAUDE = path.join(HUB_DIR, 'commands', 'claude');
const HUB_COMMANDS_OPENCODE = path.join(HUB_DIR, 'commands', 'opencode');
const HUB_AGENTS = path.join(HUB_DIR, 'agents');
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

// System files allowlist — only these are copied during updates (never colony data)
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
  'docs/constraints.md',
  'docs/pathogen-schema-example.json',
  'docs/pathogen-schema.md',
  'docs/pheromones.md',
  'docs/progressive-disclosure.md',
  'utils/atomic-write.sh',
  'utils/colorize-log.sh',
  'utils/file-lock.sh',
  'utils/watch-spawn-tree.sh',
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
  if (!manifest.files || typeof manifest.files !== 'object') {
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

function setupHub() {
  // Create ~/.aether/ directory structure and populate from package
  try {
    fs.mkdirSync(HUB_DIR, { recursive: true });

    // Read previous manifest for delta reporting
    const prevManifestRaw = readJsonSafe(path.join(HUB_DIR, 'manifest.json'));
    const prevManifest = prevManifestRaw && validateManifest(prevManifestRaw).valid ? prevManifestRaw : null;
    if (prevManifestRaw && !prevManifest) {
      log(`  Warning: previous manifest is invalid, regenerating`);
    }

    // Sync runtime/ -> ~/.aether/system/
    const runtimeSrc = path.join(PACKAGE_DIR, 'runtime');
    if (fs.existsSync(runtimeSrc)) {
      const result = syncDirWithCleanup(runtimeSrc, HUB_SYSTEM);
      log(`  Hub system: ${result.copied} files -> ${HUB_SYSTEM}`);
      if (result.removed.length > 0) {
        log(`  Hub system: removed ${result.removed.length} stale files`);
        for (const f of result.removed) log(`    - ${f}`);
      }
    }

    // Sync .claude/commands/ant/ -> ~/.aether/commands/claude/
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

    // Sync .opencode/commands/ant/ -> ~/.aether/commands/opencode/
    const opencodeCmdSrc = path.join(PACKAGE_DIR, '.opencode', 'commands', 'ant');
    if (fs.existsSync(opencodeCmdSrc)) {
      const result = syncDirWithCleanup(opencodeCmdSrc, HUB_COMMANDS_OPENCODE);
      log(`  Hub commands (opencode): ${result.copied} files -> ${HUB_COMMANDS_OPENCODE}`);
      if (result.removed.length > 0) {
        log(`  Hub commands (opencode): removed ${result.removed.length} stale files`);
        for (const f of result.removed) log(`    - ${f}`);
      }
    }

    // Sync .opencode/agents/ -> ~/.aether/agents/
    const agentsSrc = path.join(PACKAGE_DIR, '.opencode', 'agents');
    if (fs.existsSync(agentsSrc)) {
      const result = syncDirWithCleanup(agentsSrc, HUB_AGENTS);
      log(`  Hub agents: ${result.copied} files -> ${HUB_AGENTS}`);
      if (result.removed.length > 0) {
        log(`  Hub agents: removed ${result.removed.length} stale files`);
        for (const f of result.removed) log(`    - ${f}`);
      }
    }

    // Create/preserve registry.json
    if (!fs.existsSync(HUB_REGISTRY)) {
      writeJsonSync(HUB_REGISTRY, { schema_version: 1, repos: [] });
      log(`  Registry: initialized ${HUB_REGISTRY}`);
    } else {
      log(`  Registry: preserved existing ${HUB_REGISTRY}`);
    }

    // Generate and write manifest
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

    // Write version.json
    writeJsonSync(HUB_VERSION, { version: VERSION, updated_at: new Date().toISOString() });
    log(`  Hub version: ${VERSION}`);
  } catch (err) {
    // Hub setup failure doesn't block install
    log(`  Hub setup warning: ${err.message}`);
  }
}

function updateRepo(repoPath, sourceVersion, opts) {
  opts = opts || {};
  const dryRun = opts.dryRun || false;
  const force = opts.force || false;

  const repoAether = path.join(repoPath, '.aether');
  const repoVersionFile = path.join(repoAether, 'version.json');

  if (!fs.existsSync(repoAether)) {
    return { status: 'skipped', reason: 'no .aether directory' };
  }

  const currentVersion = readJsonSafe(repoVersionFile);
  const currentVer = currentVersion ? currentVersion.version : 'unknown';

  // Target directories for git safety checks
  const targetDirs = ['.aether', '.claude/commands/ant', '.opencode/commands/ant', '.opencode/agents'];

  // Git safety: check for dirty files in target directories (skip in dry-run mode)
  let stashCreated = false;
  if (!dryRun && isGitRepo(repoPath)) {
    const dirtyFiles = getGitDirtyFiles(repoPath, targetDirs);
    if (dirtyFiles.length > 0) {
      if (!force) {
        return { status: 'dirty', files: dirtyFiles };
      }
      // --force: stash dirty files before proceeding
      stashCreated = gitStashFiles(repoPath, dirtyFiles);
    }
  }

  // Sync system files from hub with cleanup
  const systemResult = syncSystemFilesWithCleanup(HUB_SYSTEM, repoAether, { dryRun });

  // Sync commands from hub with cleanup
  let commandsCopied = 0;
  const allRemovedFiles = [...systemResult.removed];

  const repoClaudeCmds = path.join(repoPath, '.claude', 'commands', 'ant');
  if (fs.existsSync(HUB_COMMANDS_CLAUDE)) {
    const result = syncDirWithCleanup(HUB_COMMANDS_CLAUDE, repoClaudeCmds, { dryRun });
    commandsCopied += result.copied;
    allRemovedFiles.push(...result.removed.map(f => `.claude/commands/ant/${f}`));
  }

  const repoOpencodeCmds = path.join(repoPath, '.opencode', 'commands', 'ant');
  if (fs.existsSync(HUB_COMMANDS_OPENCODE)) {
    const result = syncDirWithCleanup(HUB_COMMANDS_OPENCODE, repoOpencodeCmds, { dryRun });
    commandsCopied += result.copied;
    allRemovedFiles.push(...result.removed.map(f => `.opencode/commands/ant/${f}`));
  }

  // Sync agents from hub with cleanup
  let agentsCopied = 0;
  const repoAgents = path.join(repoPath, '.opencode', 'agents');
  if (fs.existsSync(HUB_AGENTS)) {
    const result = syncDirWithCleanup(HUB_AGENTS, repoAgents, { dryRun });
    agentsCopied = result.copied;
    allRemovedFiles.push(...result.removed.map(f => `.opencode/agents/${f}`));
  }

  if (dryRun) {
    return {
      status: 'dry-run',
      from: currentVer,
      to: sourceVersion,
      system: systemResult.copied,
      commands: commandsCopied,
      agents: agentsCopied,
      removed: allRemovedFiles.length,
      removedFiles: allRemovedFiles,
    };
  }

  // Write version.json
  writeJsonSync(repoVersionFile, { version: sourceVersion, updated_at: new Date().toISOString() });

  // Update registry entry
  const registry = readJsonSafe(HUB_REGISTRY);
  if (registry) {
    const ts = new Date().toISOString();
    const existing = registry.repos.find(r => r.path === repoPath);
    if (existing) {
      existing.version = sourceVersion;
      existing.updated_at = ts;
    } else {
      registry.repos.push({ path: repoPath, version: sourceVersion, registered_at: ts, updated_at: ts });
    }
    writeJsonSync(HUB_REGISTRY, registry);
  }

  return {
    status: 'updated',
    from: currentVer,
    to: sourceVersion,
    system: systemResult.copied,
    commands: commandsCopied,
    agents: agentsCopied,
    removed: allRemovedFiles.length,
    removedFiles: allRemovedFiles,
    stashCreated,
  };
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
  .description('Install slash-commands and set up distribution hub')
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
  .description('Update current repo from hub')
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

        const result = updateRepo(repo.path, sourceVersion, { dryRun, force: forceFlag });
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

      const result = updateRepo(repoPath, sourceVersion, { dryRun, force: forceFlag });

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
      console.log(`  ${result.system} system files, ${result.commands} command files, ${result.agents} agent files`);
      if (result.removed > 0) {
        console.log(`  Removed ${result.removed} stale files:`);
        for (const f of result.removedFiles) console.log(`    - ${f}`);
      }
      if (result.stashCreated) {
        console.log('  Git stash created. Recover with: git stash pop');
      }
      console.log('  Colony data (.aether/data/) untouched.');
    }
  }));

// Version command
program
  .command('version')
  .description('Show installed version')
  .action(() => {
    console.log(c.header(`aether-colony v${VERSION}`));
  });

// Uninstall command
program
  .command('uninstall')
  .description('Remove slash-commands (preserves project state and hub)')
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

// Parse command line arguments
program.parse();
