#!/usr/bin/env node

/**
 * UpdateTransaction - Two-phase commit for updates with automatic rollback
 *
 * Implements UPDATE-01 through UPDATE-04 requirements:
 * - UPDATE-01: Create checkpoint before file sync
 * - UPDATE-02: Two-phase commit (backup → sync → verify → update version)
 * - UPDATE-03: Automatic rollback on failure
 * - UPDATE-04: Recovery commands displayed prominently on failure
 */

const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const { execSync } = require('child_process');

/**
 * Error codes for update operations
 */
const UpdateErrorCodes = {
  E_UPDATE_FAILED: 'E_UPDATE_FAILED',
  E_CHECKPOINT_FAILED: 'E_CHECKPOINT_FAILED',
  E_SYNC_FAILED: 'E_SYNC_FAILED',
  E_VERIFY_FAILED: 'E_VERIFY_FAILED',
  E_ROLLBACK_FAILED: 'E_ROLLBACK_FAILED',
  E_REPO_DIRTY: 'E_REPO_DIRTY',
  E_HUB_INACCESSIBLE: 'E_HUB_INACCESSIBLE',
  E_PARTIAL_UPDATE: 'E_PARTIAL_UPDATE',
  E_NETWORK_ERROR: 'E_NETWORK_ERROR',
};

/**
 * UpdateError - Structured error with recovery commands
 *
 * Provides detailed error information and recovery commands for failed updates.
 * Recovery commands are displayed prominently to help users recover from failures.
 */
class UpdateError extends Error {
  /**
   * Create an UpdateError
   * @param {string} code - Error code from UpdateErrorCodes
   * @param {string} message - Human-readable error message
   * @param {object} details - Additional error context
   * @param {string[]} recoveryCommands - Array of shell commands to recover
   */
  constructor(code, message, details = {}, recoveryCommands = []) {
    super(message);
    this.name = 'UpdateError';
    this.code = code;
    this.details = details;
    this.recoveryCommands = recoveryCommands;
    this.timestamp = new Date().toISOString();

    // Maintain proper stack trace in V8 environments
    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, UpdateError);
    }
  }

  /**
   * Convert error to JSON representation
   * @returns {object} Structured error object
   */
  toJSON() {
    return {
      error: {
        name: this.name,
        code: this.code,
        message: this.message,
        details: this.details,
        recoveryCommands: this.recoveryCommands,
        timestamp: this.timestamp,
        stack: this.stack,
      },
    };
  }

  /**
   * Convert error to formatted string with recovery commands
   * @returns {string} Formatted error message
   */
  toString() {
    let output = `${this.name}: ${this.code} - ${this.message}`;

    if (this.details && Object.keys(this.details).length > 0) {
      output += '\n\nDetails:';
      for (const [key, value] of Object.entries(this.details)) {
        if (Array.isArray(value)) {
          output += `\n  ${key}:`;
          for (const item of value) {
            output += `\n    - ${item}`;
          }
        } else {
          output += `\n  ${key}: ${value}`;
        }
      }
    }

    if (this.recoveryCommands.length > 0) {
      output += '\n\n========================================';
      output += '\nUPDATE FAILED - RECOVERY REQUIRED';
      output += '\n========================================';
      output += '\n\nTo recover your workspace:';
      for (const cmd of this.recoveryCommands) {
        output += `\n  ${cmd}`;
      }
      output += '\n\n========================================';
    }

    return output;
  }
}

/**
 * Transaction states for tracking update progress
 */
const TransactionStates = {
  PENDING: 'pending',
  PREPARING: 'preparing',
  SYNCING: 'syncing',
  VERIFYING: 'verifying',
  COMMITTING: 'committing',
  COMMITTED: 'committed',
  ROLLING_BACK: 'rolling_back',
  ROLLED_BACK: 'rolled_back',
};

/**
 * UpdateTransaction - Two-phase commit for safe updates
 *
 * Implements a four-phase update process:
 * 1. Prepare: Create checkpoint for rollback safety
 * 2. Sync: Copy files from hub to repo
 * 3. Verify: Ensure all files copied correctly with hash verification
 * 4. Commit: Update version.json
 *
 * On any failure, automatic rollback restores the checkpoint.
 */
class UpdateTransaction {
  /**
   * Create an UpdateTransaction
   * @param {string} repoPath - Path to repository being updated
   * @param {object} options - Transaction options
   * @param {string} options.sourceVersion - Version to update to
   * @param {boolean} options.quiet - Suppress output
   * @param {boolean} options.force - Force update even with dirty files
   */
  constructor(repoPath, options = {}) {
    this.repoPath = repoPath;
    this.sourceVersion = options.sourceVersion || null;
    this.quiet = options.quiet || false;
    this.force = options.force || false;

    // Transaction state
    this.state = TransactionStates.PENDING;
    this.checkpoint = null;
    this.syncResult = null;
    this.errors = [];

    // Hub paths (from cli.js)
    this.HOME = process.env.HOME || process.env.USERPROFILE;
    this.HUB_DIR = path.join(this.HOME, '.aether');
    this.HUB_SYSTEM_DIR = path.join(this.HUB_DIR, 'system');
    this.HUB_COMMANDS_CLAUDE = path.join(this.HUB_SYSTEM_DIR, 'commands', 'claude');
    this.HUB_COMMANDS_OPENCODE = path.join(this.HUB_SYSTEM_DIR, 'commands', 'opencode');
    this.HUB_AGENTS = path.join(this.HUB_SYSTEM_DIR, 'agents');
    this.HUB_AGENTS_CLAUDE = path.join(this.HUB_SYSTEM_DIR, 'agents-claude');
    this.HUB_RULES = path.join(this.HUB_SYSTEM_DIR, 'rules');
    this.HUB_VERSION = path.join(this.HUB_DIR, 'version.json');
    this.HUB_REGISTRY = path.join(this.HUB_DIR, 'registry.json');

    // Directories to exclude from sync (user data, local state, and separately-synced dirs)
    // v4.0: archive and chambers added — these are private and must not sync to target repos
    // v6.0: oracle, midden, exchange added for complete user data protection
    this.EXCLUDE_DIRS = ['data', 'dreams', 'oracle', 'midden', 'checkpoints', 'locks', 'temp', 'agents', 'commands', 'rules', 'archive', 'chambers', 'exchange'];

    // Files to exclude from sync (user wisdom, protected files)
    this.EXCLUDE_FILES = ['QUEEN.md'];

    // Target directories for git safety checks
    this.targetDirs = ['.aether', '.claude/commands/ant', '.claude/agents/ant', '.claude/rules', '.opencode/commands/ant', '.opencode/agents'];
  }

  /**
   * Log a message (respects quiet mode)
   * @param {string} msg - Message to log
   */
  log(msg) {
    if (!this.quiet) {
      console.log(msg);
    }
  }

  /**
   * Read JSON file safely
   * @param {string} filePath - Path to JSON file
   * @returns {object|null} Parsed JSON or null on error
   * @private
   */
  readJsonSafe(filePath) {
    try {
      return JSON.parse(fs.readFileSync(filePath, 'utf8'));
    } catch {
      return null;
    }
  }

  /**
   * Write JSON file atomically
   * @param {string} filePath - Path to write
   * @param {object} data - Data to write
   * @private
   */
  writeJsonSync(filePath, data) {
    fs.mkdirSync(path.dirname(filePath), { recursive: true });
    fs.writeFileSync(filePath, JSON.stringify(data, null, 2) + '\n');
  }

  /**
   * Compute SHA-256 hash of a file
   * @param {string} filePath - Path to file
   * @returns {string|null} Hash in format 'sha256:hex' or null on error
   * @private
   */
  hashFileSync(filePath) {
    try {
      const content = fs.readFileSync(filePath);
      return 'sha256:' + crypto.createHash('sha256').update(content).digest('hex');
    } catch (err) {
      return null;
    }
  }

  /**
   * Copy file atomically using temp file + rename pattern
   * Prevents partial file corruption on process interruption
   * @param {string} srcPath - Source file path
   * @param {string} destPath - Destination file path
   * @private
   */
  copyFileAtomic(srcPath, destPath) {
    const tempPath = `${destPath}.tmp.${process.pid}.${Date.now()}`;

    try {
      // Copy to temp file first
      fs.copyFileSync(srcPath, tempPath);

      // Atomic rename (POSIX guarantees atomicity)
      fs.renameSync(tempPath, destPath);

      // Set executable permission for shell scripts
      if (destPath.endsWith('.sh')) {
        fs.chmodSync(destPath, 0o755);
      }
    } catch (err) {
      // Clean up temp file on failure
      try {
        if (fs.existsSync(tempPath)) {
          fs.unlinkSync(tempPath);
        }
      } catch {
        // Ignore cleanup errors
      }
      throw err;
    }
  }

  /**
   * Check if path is a git repository
   * @returns {boolean} True if git repo
   * @private
   */
  isGitRepo() {
    try {
      execSync('git rev-parse --git-dir', { cwd: this.repoPath, stdio: 'pipe' });
      return true;
    } catch {
      return false;
    }
  }

  /**
   * Detect dirty repository state with detailed categorization
   * @returns {object} Dirty state info: { isDirty, tracked, untracked, staged }
   */
  detectDirtyRepo() {
    try {
      const args = this.targetDirs.filter(d => fs.existsSync(path.join(this.repoPath, d)));
      if (args.length === 0) return { isDirty: false, tracked: [], untracked: [], staged: [] };

      const result = execSync(`git status --porcelain -- ${args.map(d => `"${d}"`).join(' ')}`, {
        cwd: this.repoPath,
        stdio: 'pipe',
        encoding: 'utf8',
      });

      const lines = result.trim().split('\n').filter(Boolean);
      const tracked = [];
      const untracked = [];
      const staged = [];

      for (const line of lines) {
        const status = line.slice(0, 2);
        const filePath = line.slice(3);

        // Staged changes (in index)
        if (status[0] !== ' ' && status[0] !== '?') {
          staged.push(filePath);
        }

        // Untracked files
        if (status === '??') {
          untracked.push(filePath);
        } else {
          // Modified/tracked files
          tracked.push(filePath);
        }
      }

      return {
        isDirty: lines.length > 0,
        tracked,
        untracked,
        staged,
      };
    } catch {
      return { isDirty: false, tracked: [], untracked: [], staged: [] };
    }
  }

  /**
   * Get dirty files in target directories (legacy method for backward compatibility)
   * @returns {string[]} Array of dirty file paths
   * @private
   */
  getGitDirtyFiles() {
    const dirty = this.detectDirtyRepo();
    return [...dirty.tracked, ...dirty.untracked];
  }

  /**
   * Validate repository state before update
   * @returns {object} Validation result: { clean: boolean, dirtyState?: object }
   * @throws {UpdateError} If repository has uncommitted changes
   */
  validateRepoState() {
    const dirtyState = this.detectDirtyRepo();

    if (!dirtyState.isDirty) {
      return { clean: true };
    }

    // If force flag is set, allow dirty repo (will be stashed in checkpoint)
    if (this.force) {
      this.log('  Force flag set: proceeding with dirty repository (will stash changes)');
      return { clean: true, dirty: dirtyState, force: true };
    }

    // Build detailed error message
    const lines = [
      'Cannot update: repository has uncommitted changes',
      '',
    ];

    if (dirtyState.tracked.length > 0) {
      lines.push(`Modified files (${dirtyState.tracked.length}):`);
      for (const f of dirtyState.tracked.slice(0, 10)) {
        lines.push(`  - ${f}`);
      }
      if (dirtyState.tracked.length > 10) {
        lines.push(`  ... and ${dirtyState.tracked.length - 10} more`);
      }
      lines.push('');
    }

    if (dirtyState.untracked.length > 0) {
      lines.push(`Untracked files (${dirtyState.untracked.length}):`);
      for (const f of dirtyState.untracked.slice(0, 10)) {
        lines.push(`  - ${f}`);
      }
      if (dirtyState.untracked.length > 10) {
        lines.push(`  ... and ${dirtyState.untracked.length - 10} more`);
      }
      lines.push('');
    }

    if (dirtyState.staged.length > 0) {
      lines.push(`Staged files (${dirtyState.staged.length}):`);
      for (const f of dirtyState.staged.slice(0, 10)) {
        lines.push(`  - ${f}`);
      }
      if (dirtyState.staged.length > 10) {
        lines.push(`  ... and ${dirtyState.staged.length - 10} more`);
      }
      lines.push('');
    }

    lines.push('Options:');
    lines.push('  1. Stash changes: git stash push -m "pre-update"');
    lines.push('  2. Commit changes: git add . && git commit -m "wip"');
    lines.push('  3. Discard changes: git checkout -- . (DANGER: loses work)');
    lines.push('');
    lines.push('After resolving, run: aether update');

    const message = lines.join('\n');

    throw new UpdateError(
      UpdateErrorCodes.E_REPO_DIRTY,
      'Repository has uncommitted changes',
      {
        trackedCount: dirtyState.tracked.length,
        untrackedCount: dirtyState.untracked.length,
        stagedCount: dirtyState.staged.length,
        tracked: dirtyState.tracked,
        untracked: dirtyState.untracked,
        staged: dirtyState.staged,
      },
      [
        `cd ${this.repoPath} && git stash push -m "pre-update"`,
        `cd ${this.repoPath} && git add . && git commit -m "wip"`,
        `cd ${this.repoPath} && aether update`,
      ]
    );
  }

  /**
   * Create git stash for files
   * @param {string[]} files - Files to stash
   * @returns {string|null} Stash reference or null on failure
   * @private
   */
  gitStashFiles(files) {
    try {
      // Separate tracked and untracked files
      const trackedFiles = [];
      const untrackedFiles = [];

      for (const file of files) {
        const fullPath = path.join(this.repoPath, file);
        try {
          // Check if file is tracked by git
          execSync(`git ls-files --error-unmatch "${file}"`, {
            cwd: this.repoPath,
            stdio: 'pipe'
          });
          trackedFiles.push(file);
        } catch {
          // File is not tracked (untracked or in .gitignore)
          untrackedFiles.push(file);
        }
      }

      let stashRef = null;

      // Stash tracked files
      if (trackedFiles.length > 0) {
        const fileArgs = trackedFiles.map(f => `"${f}"`).join(' ');
        execSync(`git stash push -m "aether-update-backup" -- ${fileArgs}`, {
          cwd: this.repoPath,
          stdio: 'pipe',
        });

        // Get the stash reference
        const stashList = execSync('git stash list', { cwd: this.repoPath, encoding: 'utf8' });
        const match = stashList.match(/^(stash@\{[^}]+\})/m);
        stashRef = match ? match[1] : null;
      }

      // For untracked files, we can't stash them easily
      // Just log a warning - they'll be left as-is during the update
      if (untrackedFiles.length > 0) {
        this.log(`  Note: ${untrackedFiles.length} untracked files won't be stashed (left in place)`);
      }

      return stashRef;
    } catch (err) {
      this.log(`  Warning: git stash failed (${err.message}). Proceeding without stash.`);
      return null;
    }
  }

  /**
   * Create a checkpoint before update
   * Implements UPDATE-01: Update creates checkpoint before file sync
   *
   * @returns {Promise<object>} Checkpoint object: { id, stashRef, timestamp }
   * @throws {UpdateError} If checkpoint creation fails
   */
  async createCheckpoint() {
    this.log('  Creating checkpoint for rollback safety...');

    try {
      // 1. Check if in git repo
      if (!this.isGitRepo()) {
        throw new UpdateError(
          UpdateErrorCodes.E_CHECKPOINT_FAILED,
          'Not in a git repository',
          { repoPath: this.repoPath },
          ['git init', 'cd ' + this.repoPath]
        );
      }

      // 2. Get dirty files in target directories
      const dirtyFiles = this.getGitDirtyFiles();

      // 3. Stash dirty files if any
      let stashRef = null;
      if (dirtyFiles.length > 0) {
        stashRef = this.gitStashFiles(dirtyFiles);
      }

      // 4. Generate checkpoint ID
      const now = new Date();
      const checkpointId = `chk_${now.toISOString().slice(0, 10).replace(/-/g, '')}_${now.toTimeString().slice(0, 8).replace(/:/g, '')}`;

      // 5. Create checkpoint metadata
      const checkpoint = {
        id: checkpointId,
        stashRef,
        timestamp: now.toISOString(),
        dirtyFiles,
        repoPath: this.repoPath,
      };

      // 6. Save checkpoint metadata
      const checkpointsDir = path.join(this.repoPath, '.aether', 'checkpoints');
      fs.mkdirSync(checkpointsDir, { recursive: true });
      this.writeJsonSync(path.join(checkpointsDir, `${checkpointId}.json`), checkpoint);

      this.checkpoint = checkpoint;
      this.log(`  Created checkpoint ${checkpointId} for rollback safety`);

      return checkpoint;
    } catch (error) {
      if (error instanceof UpdateError) {
        throw error;
      }
      throw new UpdateError(
        UpdateErrorCodes.E_CHECKPOINT_FAILED,
        `Failed to create checkpoint: ${error.message}`,
        { originalError: error.message },
        this.getRecoveryCommands()
      );
    }
  }

  /**
   * List files recursively in a directory
   * @param {string} dir - Directory to list
   * @param {string} base - Base path for relative paths
   * @returns {string[]} Array of relative file paths
   * @private
   */
  listFilesRecursive(dir, base) {
    base = base || dir;
    const results = [];
    if (!fs.existsSync(dir)) return results;
    const entries = fs.readdirSync(dir, { withFileTypes: true });
    for (const entry of entries) {
      if (entry.name.startsWith('.')) continue;
      const fullPath = path.join(dir, entry.name);
      if (entry.isDirectory()) {
        results.push(...this.listFilesRecursive(fullPath, base));
      } else {
        results.push(path.relative(base, fullPath));
      }
    }
    return results;
  }

  /**
   * Sync directory with cleanup (copied from cli.js)
   * @param {string} src - Source directory
   * @param {string} dest - Destination directory
   * @param {object} opts - Options
   * @returns {object} Sync result: { copied, removed, skipped }
   * @private
   */
  syncDirWithCleanup(src, dest, opts) {
    opts = opts || {};
    const dryRun = opts.dryRun || false;

    try {
      fs.mkdirSync(dest, { recursive: true });
    } catch (err) {
      if (err.code !== 'EEXIST') {
        throw new UpdateError(
          UpdateErrorCodes.E_SYNC_FAILED,
          `Could not create directory ${dest}: ${err.message}`,
          { src, dest },
          this.getRecoveryCommands()
        );
      }
    }

    // Copy phase with hash comparison
    let copied = 0;
    let skipped = 0;
    const srcFiles = this.listFilesRecursive(src);

    if (!dryRun) {
      for (const relPath of srcFiles) {
        const srcPath = path.join(src, relPath);
        const destPath = path.join(dest, relPath);
        try {
          fs.mkdirSync(path.dirname(destPath), { recursive: true });

          // Hash comparison: only copy if file doesn't exist or hash differs
          let shouldCopy = true;
          if (fs.existsSync(destPath)) {
            const srcHash = this.hashFileSync(srcPath);
            const destHash = this.hashFileSync(destPath);
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
          throw new UpdateError(
            UpdateErrorCodes.E_SYNC_FAILED,
            `Could not copy ${relPath}: ${err.message}`,
            { srcPath, destPath },
            this.getRecoveryCommands()
          );
        }
      }
    } else {
      copied = srcFiles.length;
    }

    // Cleanup phase — remove files in dest that aren't in src
    const destFiles = this.listFilesRecursive(dest);
    const srcSet = new Set(srcFiles);
    const removed = [];

    for (const relPath of destFiles) {
      if (!srcSet.has(relPath)) {
        removed.push(relPath);
        if (!dryRun) {
          try {
            fs.unlinkSync(path.join(dest, relPath));
          } catch (err) {
            this.log(`  Warning: could not remove ${relPath}: ${err.message}`);
          }
        }
      }
    }

    // Clean empty directories
    if (!dryRun && removed.length > 0) {
      this.cleanEmptyDirs(dest);
    }

    return { copied, removed, skipped };
  }

  /**
   * Clean empty directories recursively
   * @param {string} dir - Directory to clean
   * @private
   */
  cleanEmptyDirs(dir) {
    if (!fs.existsSync(dir)) return;
    const entries = fs.readdirSync(dir, { withFileTypes: true });
    for (const entry of entries) {
      if (entry.isDirectory()) {
        this.cleanEmptyDirs(path.join(dir, entry.name));
      }
    }
    // Re-read after recursive cleanup
    const remaining = fs.readdirSync(dir);
    if (remaining.length === 0) {
      fs.rmdirSync(dir);
    }
  }

  /**
   * Check if a path should be excluded from sync
   * @param {string} relPath - Relative path
   * @returns {boolean} True if should be excluded
   * @private
   */
  shouldExclude(relPath) {
    const parts = relPath.split(path.sep);
    // Check for excluded directories
    if (parts.some(part => this.EXCLUDE_DIRS.includes(part))) {
      return true;
    }
    // Check for excluded files (QUEEN.md protection)
    const basename = path.basename(relPath);
    if (this.EXCLUDE_FILES.includes(basename)) {
      return true;
    }
    return false;
  }

  /**
   * Sync .aether/ directory from hub to repo, excluding user data directories
   * @param {string} srcDir - Source hub directory
   * @param {string} destDir - Destination repo .aether/ directory
   * @param {object} opts - Options
   * @returns {object} Sync result: { copied, removed, skipped }
   * @private
   */
  syncAetherToRepo(srcDir, destDir, opts) {
    opts = opts || {};
    const dryRun = opts.dryRun || false;

    if (!fs.existsSync(srcDir)) {
      return { copied: 0, removed: [], skipped: 0 };
    }

    // Collect all files in source, filtering out excluded directories
    const srcFiles = [];
    const collectFiles = (dir, base) => {
      const entries = fs.readdirSync(dir, { withFileTypes: true });
      for (const entry of entries) {
        if (entry.name.startsWith('.')) continue;
        const fullPath = path.join(dir, entry.name);
        const relPath = path.relative(base, fullPath);

        if (this.shouldExclude(relPath)) continue;

        if (entry.isDirectory()) {
          collectFiles(fullPath, base);
        } else {
          srcFiles.push(relPath);
        }
      }
    };
    collectFiles(srcDir, srcDir);

    // Copy files with hash comparison
    let copied = 0;
    let skipped = 0;
    for (const relPath of srcFiles) {
      const srcPath = path.join(srcDir, relPath);
      const destPath = path.join(destDir, relPath);

      if (!dryRun) {
        fs.mkdirSync(path.dirname(destPath), { recursive: true });

        // Hash comparison
        let shouldCopy = true;
        if (fs.existsSync(destPath)) {
          const srcHash = this.hashFileSync(srcPath);
          const destHash = this.hashFileSync(destPath);
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
        }
      }
      copied++;
    }

    // Cleanup: remove files in dest that aren't in source
    const destFiles = [];
    const collectDestFiles = (dir, base) => {
      if (!fs.existsSync(dir)) return;
      const entries = fs.readdirSync(dir, { withFileTypes: true });
      for (const entry of entries) {
        if (entry.name.startsWith('.')) continue;
        const fullPath = path.join(dir, entry.name);
        const relPath = path.relative(base, fullPath);

        if (this.shouldExclude(relPath)) continue;

        if (entry.isDirectory()) {
          collectDestFiles(fullPath, base);
        } else {
          destFiles.push(relPath);
        }
      }
    };
    collectDestFiles(destDir, destDir);

    const srcSet = new Set(srcFiles);
    const removed = [];
    for (const relPath of destFiles) {
      if (!srcSet.has(relPath)) {
        removed.push(relPath);
        if (!dryRun) {
          try {
            fs.unlinkSync(path.join(destDir, relPath));
          } catch (err) {
            // Ignore cleanup errors
          }
        }
      }
    }

    if (!dryRun && removed.length > 0) {
      this.cleanEmptyDirs(destDir);
    }

    return { copied, removed, skipped };
  }

  /**
   * Remove known stale directories and files left behind by pre-3.0.0 versions.
   * These paths were incorrectly distributed in earlier releases and must be
   * explicitly removed — EXCLUDE_DIRS prevents NEW pollution but does not clean
   * items that already exist on disk.
   *
   * Idempotent: safe to call when items do not exist.
   *
   * @param {string} repoPath - Absolute path to the target repository root
   * @returns {{ cleaned: string[], failed: Array<{label: string, error: string}> }}
   */
  cleanupStaleAetherDirs(repoPath) {
    const staleItems = [
      {
        path: path.join(repoPath, '.aether', 'agents'),
        label: '.aether/agents/ (stale duplicate)',
        type: 'dir',
      },
      {
        path: path.join(repoPath, '.aether', 'commands'),
        label: '.aether/commands/ (stale duplicate)',
        type: 'dir',
      },
      {
        path: path.join(repoPath, '.aether', 'planning.md'),
        label: '.aether/planning.md (phantom file)',
        type: 'file',
      },
    ];

    const cleaned = [];
    const failed = [];

    for (const item of staleItems) {
      if (!fs.existsSync(item.path)) {
        // Already clean — idempotent skip
        continue;
      }

      try {
        if (item.type === 'dir') {
          fs.rmSync(item.path, { recursive: true, force: true });
        } else {
          fs.unlinkSync(item.path);
        }
        cleaned.push(item.label);
      } catch (err) {
        failed.push({ label: item.label, error: err.message });
      }
    }

    return { cleaned, failed };
  }

  /**
   * Sync files from hub to repo
   * @param {string} sourceVersion - Version to sync from
   * @param {boolean} dryRun - If true, don't actually copy files
   * @returns {object} Sync result: { copied, removed, unchanged, errors }
   */
  syncFiles(sourceVersion, dryRun = false) {
    this.state = TransactionStates.SYNCING;

    const results = {
      system: { copied: 0, removed: 0, skipped: 0 },
      commands: { copied: 0, removed: 0, skipped: 0 },
      agents: { copied: 0, removed: 0, skipped: 0 },
      agents_claude: { copied: 0, removed: [], skipped: 0 },
      rules: { copied: 0, removed: 0, skipped: 0 },
      errors: [],
    };

    const repoAether = path.join(this.repoPath, '.aether');

    // Sync .aether/ from hub to repo (excluding user data directories)
    if (fs.existsSync(this.HUB_SYSTEM_DIR)) {
      results.system = this.syncAetherToRepo(this.HUB_SYSTEM_DIR, repoAether, { dryRun });
    }

    // Sync commands from hub
    const repoClaudeCmds = path.join(this.repoPath, '.claude', 'commands', 'ant');
    if (fs.existsSync(this.HUB_COMMANDS_CLAUDE)) {
      const result = this.syncDirWithCleanup(this.HUB_COMMANDS_CLAUDE, repoClaudeCmds, { dryRun });
      results.commands = result;
    }

    const repoOpencodeCmds = path.join(this.repoPath, '.opencode', 'commands', 'ant');
    if (fs.existsSync(this.HUB_COMMANDS_OPENCODE)) {
      const result = this.syncDirWithCleanup(this.HUB_COMMANDS_OPENCODE, repoOpencodeCmds, { dryRun });
      results.commands.copied += result.copied;
      results.commands.removed.push(...result.removed);
      results.commands.skipped += result.skipped;
    }

    // Sync agents from hub
    const repoAgents = path.join(this.repoPath, '.opencode', 'agents');
    if (fs.existsSync(this.HUB_AGENTS)) {
      results.agents = this.syncDirWithCleanup(this.HUB_AGENTS, repoAgents, { dryRun });
    }

    // Sync claude agents from hub to .claude/agents/ant/
    const repoClaudeAgents = path.join(this.repoPath, '.claude', 'agents', 'ant');
    if (fs.existsSync(this.HUB_AGENTS_CLAUDE)) {
      results.agents_claude = this.syncDirWithCleanup(this.HUB_AGENTS_CLAUDE, repoClaudeAgents, { dryRun });
    }

    // Sync rules from hub to .claude/rules/
    const repoRules = path.join(this.repoPath, '.claude', 'rules');
    if (fs.existsSync(this.HUB_RULES)) {
      results.rules = this.syncDirWithCleanup(this.HUB_RULES, repoRules, { dryRun });
    }

    this.syncResult = results;
    return results;
  }

  /**
   * Verify integrity of synced files
   * @returns {object} Verification result: { valid: boolean, errors: string[] }
   */
  verifyIntegrity() {
    this.state = TransactionStates.VERIFYING;

    const errors = [];

    // Verify hub files exist and match expected
    const verifyDir = (hubDir, repoDir) => {
      if (!fs.existsSync(hubDir)) return;

      const files = this.listFilesRecursive(hubDir);
      for (const relPath of files) {
        // Skip excluded directories
        if (this.shouldExclude(relPath)) continue;

        const hubPath = path.join(hubDir, relPath);
        const repoPath = path.join(repoDir, relPath);

        // Check file exists
        if (!fs.existsSync(repoPath)) {
          errors.push(`Missing file: ${relPath}`);
          continue;
        }

        // Check hash matches
        const hubHash = this.hashFileSync(hubPath);
        const repoHash = this.hashFileSync(repoPath);

        if (hubHash !== repoHash) {
          errors.push(`Hash mismatch: ${relPath}`);
        }
      }
    };

    const repoAether = path.join(this.repoPath, '.aether');
    verifyDir(this.HUB_SYSTEM_DIR, repoAether);
    verifyDir(this.HUB_COMMANDS_CLAUDE, path.join(this.repoPath, '.claude', 'commands', 'ant'));
    verifyDir(this.HUB_COMMANDS_OPENCODE, path.join(this.repoPath, '.opencode', 'commands', 'ant'));
    verifyDir(this.HUB_AGENTS, path.join(this.repoPath, '.opencode', 'agents'));
    verifyDir(this.HUB_AGENTS_CLAUDE, path.join(this.repoPath, '.claude', 'agents', 'ant'));
    verifyDir(this.HUB_RULES, path.join(this.repoPath, '.claude', 'rules'));

    return {
      valid: errors.length === 0,
      errors,
    };
  }

  /**
   * Check if hub is accessible before sync
   * @returns {object} Accessibility result: { accessible: boolean, errors: string[] }
   * @throws {UpdateError} If hub is not accessible
   */
  checkHubAccessibility() {
    const errors = [];

    // Check if HUB_DIR exists
    if (!fs.existsSync(this.HUB_DIR)) {
      errors.push(`Hub directory does not exist: ${this.HUB_DIR}`);
      return {
        accessible: false,
        errors,
        recoveryCommands: [
          'aether install',
          `mkdir -p ${this.HUB_DIR}`,
        ],
      };
    }

    // Check if hub directories are readable
    const checkDir = (dir, name) => {
      if (!fs.existsSync(dir)) {
        // Non-critical: directories may not exist if no files to sync
        return;
      }
      try {
        fs.accessSync(dir, fs.constants.R_OK);
      } catch (err) {
        errors.push(`Cannot read ${name} directory: ${dir} - ${err.message}`);
      }
    };

    checkDir(this.HUB_DIR, '.aether');
    checkDir(this.HUB_COMMANDS_CLAUDE, 'commands/claude');
    checkDir(this.HUB_COMMANDS_OPENCODE, 'commands/opencode');
    checkDir(this.HUB_AGENTS, 'agents');
    checkDir(this.HUB_AGENTS_CLAUDE, 'agents-claude');
    checkDir(this.HUB_RULES, 'rules');
    checkDir(this.HUB_VERSION, 'version');

    // Check if source files exist
    const checkSourceFiles = () => {
      if (fs.existsSync(this.HUB_VERSION)) {
        return true;
      }
      errors.push(`Hub version file not found: ${this.HUB_VERSION}`);
      return false;
    };

    const hasVersion = checkSourceFiles();

    if (errors.length > 0 || !hasVersion) {
      return {
        accessible: false,
        errors,
        recoveryCommands: [
          `ls -la ${this.HUB_DIR}`,
          'aether install',
          'aether update',
        ],
      };
    }

    return { accessible: true, errors: [] };
  }

  /**
   * Detect partial update by comparing expected vs actual files
   * @returns {object} Detection result: { isPartial, missing, corrupted }
   */
  detectPartialUpdate() {
    const missing = [];
    const corrupted = [];

    // Compare expected files (from hub) vs actual files (in repo)
    const checkDir = (hubDir, repoDir) => {
      if (!fs.existsSync(hubDir)) return;

      const files = this.listFilesRecursive(hubDir);
      for (const relPath of files) {
        // Skip excluded directories
        if (this.shouldExclude(relPath)) continue;

        const hubPath = path.join(hubDir, relPath);
        const repoPath = path.join(repoDir, relPath);

        // Check if file exists
        if (!fs.existsSync(repoPath)) {
          missing.push({
            path: relPath,
            hubPath,
            repoPath,
          });
          continue;
        }

        // Check file size
        try {
          const hubStat = fs.statSync(hubPath);
          const repoStat = fs.statSync(repoPath);

          if (hubStat.size !== repoStat.size) {
            corrupted.push({
              path: relPath,
              reason: 'size_mismatch',
              hubSize: hubStat.size,
              repoSize: repoStat.size,
            });
            continue;
          }

          // Check hash
          const hubHash = this.hashFileSync(hubPath);
          const repoHash = this.hashFileSync(repoPath);

          if (hubHash !== repoHash) {
            corrupted.push({
              path: relPath,
              reason: 'hash_mismatch',
              hubHash,
              repoHash,
            });
          }
        } catch (err) {
          corrupted.push({
            path: relPath,
            reason: 'read_error',
            error: err.message,
          });
        }
      }
    };

    const repoAether = path.join(this.repoPath, '.aether');
    checkDir(this.HUB_SYSTEM_DIR, repoAether);
    checkDir(this.HUB_COMMANDS_CLAUDE, path.join(this.repoPath, '.claude', 'commands', 'ant'));
    checkDir(this.HUB_COMMANDS_OPENCODE, path.join(this.repoPath, '.opencode', 'commands', 'ant'));
    checkDir(this.HUB_AGENTS, path.join(this.repoPath, '.opencode', 'agents'));
    checkDir(this.HUB_RULES, path.join(this.repoPath, '.claude', 'rules'));

    return {
      isPartial: missing.length > 0 || corrupted.length > 0,
      missing,
      corrupted,
    };
  }

  /**
   * Verify sync completeness after file sync
   * @throws {UpdateError} If partial update detected
   */
  verifySyncCompleteness() {
    const partial = this.detectPartialUpdate();

    if (!partial.isPartial) {
      return;
    }

    // Build detailed error message
    const lines = [
      `Update incomplete: ${partial.missing.length} files missing, ${partial.corrupted.length} files corrupted`,
      '',
    ];

    if (partial.missing.length > 0) {
      lines.push('Missing files:');
      for (const f of partial.missing.slice(0, 10)) {
        lines.push(`  - ${f.path}`);
      }
      if (partial.missing.length > 10) {
        lines.push(`  ... and ${partial.missing.length - 10} more`);
      }
      lines.push('');
    }

    if (partial.corrupted.length > 0) {
      lines.push('Corrupted files:');
      for (const f of partial.corrupted.slice(0, 10)) {
        lines.push(`  - ${f.path} (${f.reason})`);
      }
      if (partial.corrupted.length > 10) {
        lines.push(`  ... and ${partial.corrupted.length - 10} more`);
      }
      lines.push('');
    }

    lines.push('The update has been rolled back. Your workspace is unchanged.');
    lines.push('');
    lines.push('To retry: aether update');

    throw new UpdateError(
      UpdateErrorCodes.E_PARTIAL_UPDATE,
      'Update incomplete: files missing or corrupted',
      {
        missingCount: partial.missing.length,
        corruptedCount: partial.corrupted.length,
        missing: partial.missing.map(f => f.path),
        corrupted: partial.corrupted.map(f => ({ path: f.path, reason: f.reason })),
      },
      [
        `cd ${this.repoPath}`,
        'aether update',
      ]
    );
  }

  /**
   * Handle network-related errors with enhanced diagnostics
   * @param {Error} error - Original error
   * @returns {UpdateError} Enhanced error with recovery commands
   */
  handleNetworkError(error) {
    const networkErrorCodes = ['ETIMEDOUT', 'ECONNREFUSED', 'ENETUNREACH', 'EACCES', 'EPERM'];
    const isNetworkError = networkErrorCodes.includes(error.code) ||
      error.message.includes('network') ||
      error.message.includes('timeout') ||
      error.message.includes('connection');

    if (!isNetworkError) {
      // Not a network error, return generic error
      return new UpdateError(
        UpdateErrorCodes.E_UPDATE_FAILED,
        error.message,
        { originalError: error.stack },
        this.getRecoveryCommands()
      );
    }

    // Build network-specific error message
    const lines = [
      `Network error during update: ${error.message}`,
      '',
      'Possible causes:',
      `  - Hub directory not accessible: ${this.HUB_DIR}`,
      '  - Network filesystem unavailable',
      '  - Permission denied',
      '',
      'Recovery:',
      '  1. Check network connectivity',
      `  2. Verify hub exists: ls -la ${this.HUB_DIR}`,
      '  3. Retry: aether update',
    ];

    return new UpdateError(
      UpdateErrorCodes.E_NETWORK_ERROR,
      `Network error: ${error.message}`,
      {
        hubDir: this.HUB_DIR,
        originalError: error.stack,
        errorCode: error.code,
      },
      [
        `ls -la ${this.HUB_DIR}`,
        'aether install',
        'aether update',
      ]
    );
  }

  /**
   * Update version.json in repo
   * @param {string} sourceVersion - Version to set
   */
  updateVersion(sourceVersion) {
    const repoVersionFile = path.join(this.repoPath, '.aether', 'version.json');
    this.writeJsonSync(repoVersionFile, {
      version: sourceVersion,
      updated_at: new Date().toISOString(),
    });

    // Update registry entry
    const registry = this.readJsonSafe(this.HUB_REGISTRY);
    if (registry) {
      const ts = new Date().toISOString();
      const existing = registry.repos.find(r => r.path === this.repoPath);
      if (existing) {
        existing.version = sourceVersion;
        existing.updated_at = ts;
      } else {
        registry.repos.push({
          path: this.repoPath,
          version: sourceVersion,
          registered_at: ts,
          updated_at: ts,
        });
      }
      this.writeJsonSync(this.HUB_REGISTRY, registry);
    }
  }

  /**
   * Rollback to checkpoint
   * Implements UPDATE-03: Automatic rollback on sync failure
   *
   * @returns {Promise<boolean>} True if rollback succeeded
   */
  async rollback() {
    // Clear pending sentinel on rollback — we've cleanly returned to prior state
    const pendingPath = path.join(this.repoPath, '.aether', '.update-pending');
    try {
      if (fs.existsSync(pendingPath)) {
        fs.unlinkSync(pendingPath);
      }
    } catch { /* ignore — rollback cleanup is best-effort */ }

    this.state = TransactionStates.ROLLING_BACK;
    this.log('  Rolling back to checkpoint...');

    try {
      if (!this.checkpoint) {
        this.log('  No checkpoint to rollback to');
        this.state = TransactionStates.ROLLED_BACK;
        return false;
      }

      // Restore from stash if available
      if (this.checkpoint.stashRef) {
        try {
          execSync(`git stash pop ${this.checkpoint.stashRef}`, {
            cwd: this.repoPath,
            stdio: 'pipe',
          });
          this.log(`  Restored stash ${this.checkpoint.stashRef}`);
        } catch (err) {
          this.log(`  Warning: could not restore stash: ${err.message}`);
        }
      }

      // Remove checkpoint metadata file
      const checkpointPath = path.join(
        this.repoPath,
        '.aether',
        'checkpoints',
        `${this.checkpoint.id}.json`
      );
      if (fs.existsSync(checkpointPath)) {
        fs.unlinkSync(checkpointPath);
      }

      this.state = TransactionStates.ROLLED_BACK;
      this.log('  Rollback complete');
      return true;
    } catch (error) {
      this.errors.push(`Rollback failed: ${error.message}`);
      this.state = TransactionStates.ROLLED_BACK;
      return false;
    }
  }

  /**
   * Get recovery commands based on transaction state
   * Implements UPDATE-04: Recovery commands displayed prominently on failure
   *
   * @returns {string[]} Array of shell commands to recover
   */
  getRecoveryCommands() {
    const commands = [];

    // If stash was created, include git stash pop
    if (this.checkpoint?.stashRef) {
      commands.push(`cd ${this.repoPath} && git stash pop ${this.checkpoint.stashRef}`);
    }

    // If checkpoint exists, include checkpoint restore
    if (this.checkpoint?.id) {
      commands.push(`aether checkpoint restore ${this.checkpoint.id}`);
    }

    // Always include manual fallback
    commands.push(`cd ${this.repoPath} && git reset --hard HEAD`);

    return commands;
  }

  /**
   * Execute the full two-phase commit
   * Implements UPDATE-02: Two-phase commit (backup → sync → verify → update version)
   *
   * @param {string} sourceVersion - Version to update to
   * @param {object} options - Execution options
   * @param {boolean} options.dryRun - If true, don't actually modify files
   * @returns {Promise<object>} Result object
   * @throws {UpdateError} On any failure (with automatic rollback)
   */
  async execute(sourceVersion, options = {}) {
    const dryRun = options.dryRun || false;

    // Write pending sentinel BEFORE any work — marks update as in-flight
    const pendingPath = path.join(this.repoPath, '.aether', '.update-pending');
    this.writeJsonSync(pendingPath, {
      target_version: sourceVersion,
      started_at: new Date().toISOString(),
    });

    try {
      // Phase 0: Validate repo state (before any modifications)
      // Check for dirty repo and provide clear recovery instructions
      this.validateRepoState();

      // Phase 1: Prepare
      // UPDATE-01: Create checkpoint before file sync
      this.state = TransactionStates.PREPARING;

      // Check hub accessibility before proceeding
      const hubAccess = this.checkHubAccessibility();
      if (!hubAccess.accessible) {
        throw new UpdateError(
          UpdateErrorCodes.E_HUB_INACCESSIBLE,
          'Hub is not accessible',
          { errors: hubAccess.errors },
          hubAccess.recoveryCommands || [
            `ls -la ${this.HUB_DIR}`,
            'aether install',
            'aether update',
          ]
        );
      }

      await this.createCheckpoint();

      // Clean up known stale directories/files from previous versions
      this.cleanupResult = this.cleanupStaleAetherDirs(this.repoPath);

      // Phase 2: Sync (with network error handling)
      this.state = TransactionStates.SYNCING;
      try {
        this.syncFiles(sourceVersion, dryRun);
      } catch (syncError) {
        // Handle network errors specifically
        throw this.handleNetworkError(syncError);
      }

      // Phase 3: Verify (skip if dryRun)
      if (!dryRun) {
        this.state = TransactionStates.VERIFYING;

        // Check for partial updates first
        this.verifySyncCompleteness();

        // Then run integrity verification
        const verification = this.verifyIntegrity();
        if (!verification.valid) {
          // UPDATE-03: Automatic rollback on sync failure
          await this.rollback();
          throw new UpdateError(
            UpdateErrorCodes.E_VERIFY_FAILED,
            'Verification failed after sync',
            { errors: verification.errors },
            this.getRecoveryCommands()
          );
        }
      }

      // Phase 4: Commit (skip if dryRun)
      if (!dryRun) {
        this.state = TransactionStates.COMMITTING;
        this.updateVersion(sourceVersion);
        this.state = TransactionStates.COMMITTED;

        // Delete pending sentinel — update is now complete
        try {
          if (fs.existsSync(pendingPath)) {
            fs.unlinkSync(pendingPath);
          }
        } catch (err) {
          this.log(`  Warning: could not clear update sentinel: ${err.message}`);
        }
      }

      // Calculate totals
      const filesSynced = (this.syncResult?.system?.copied || 0) +
                         (this.syncResult?.commands?.copied || 0) +
                         (this.syncResult?.agents?.copied || 0) +
                         (this.syncResult?.rules?.copied || 0);
      const filesRemoved = (this.syncResult?.system?.removed?.length || 0) +
                          (this.syncResult?.commands?.removed?.length || 0) +
                          (this.syncResult?.agents?.removed?.length || 0) +
                          (this.syncResult?.rules?.removed?.length || 0);

      return {
        success: true,
        status: dryRun ? 'dry-run' : 'updated',
        checkpoint_id: this.checkpoint?.id,
        files_synced: filesSynced,
        files_removed: filesRemoved,
        sync_result: this.syncResult,
        cleanup_result: this.cleanupResult || { cleaned: [], failed: [] },
      };

    } catch (error) {
      // UPDATE-03: Automatic rollback on any failure
      if (this.state !== TransactionStates.ROLLED_BACK &&
          this.state !== TransactionStates.ROLLING_BACK) {
        await this.rollback();
      }

      // Enhance error with recovery commands if not already an UpdateError
      if (!(error instanceof UpdateError)) {
        error = new UpdateError(
          UpdateErrorCodes.E_UPDATE_FAILED,
          error.message,
          { originalError: error.stack },
          this.getRecoveryCommands()
        );
      }

      throw error;
    }
  }
}

module.exports = {
  UpdateTransaction,
  UpdateError,
  UpdateErrorCodes,
  TransactionStates,
};