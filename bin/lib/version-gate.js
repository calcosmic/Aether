#!/usr/bin/env node

/**
 * Version Gate Module
 *
 * Checks whether the Go binary at ~/.aether/bin/aether exists, is executable,
 * and reports a version matching the npm package version. Provides delegation
 * logic so that `aether` commands route to the Go binary when the gate passes
 * and fall back to the Node.js CLI when it does not.
 *
 * Requirements: GATE-01, GATE-02, SHM-01, SHM-02
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const HOME = process.env.HOME || process.env.USERPROFILE;
const BINARY_PATH = HOME ? path.join(HOME, '.aether', 'bin', 'aether') : null;

// Commands that must always run in Node.js regardless of binary availability (SHM-02)
const NODE_ONLY_COMMANDS = ['install', 'update', 'setup', 'setup-hub'];

/**
 * Compare two semver version strings.
 * Handles optional 'v' prefix and compares major.minor.patch numerically.
 * Pre-release tags are ignored for comparison purposes (treated as the base version).
 *
 * @param {string} a - First version string
 * @param {string} b - Second version string
 * @returns {number} -1 if a < b, 0 if a === b, 1 if a > b
 */
function compareVersions(a, b) {
  const stripPrefix = (v) => String(v).replace(/^v/, '');
  const parseParts = (v) => {
    const cleaned = stripPrefix(v);
    // Take only the numeric parts (ignore pre-release tags like -alpha.1)
    const numericPart = cleaned.split('-')[0];
    return numericPart.split('.').map((n) => {
      const parsed = parseInt(n, 10);
      return isNaN(parsed) ? 0 : parsed;
    });
  };

  const aParts = parseParts(a);
  const bParts = parseParts(b);
  const maxLen = Math.max(aParts.length, bParts.length);

  for (let i = 0; i < maxLen; i++) {
    const aVal = aParts[i] || 0;
    const bVal = bParts[i] || 0;
    if (aVal < bVal) return -1;
    if (aVal > bVal) return 1;
  }

  return 0;
}

/**
 * Get the expected path to the Go binary.
 * @returns {string|null} Absolute path to the binary, or null if HOME is unset
 */
function getBinaryPath() {
  return BINARY_PATH;
}

/**
 * Check whether the Go binary is available and its version matches the npm package.
 *
 * Returns an object describing the binary state:
 * - available: boolean - true if binary exists, is executable, and version matches
 * - path: string - the expected binary path
 * - version: string|null - the binary's reported version (null if unavailable)
 * - reason: string|null - why the gate failed, if it did
 *
 * @param {object} [opts] - Options
 * @param {string} [opts.binaryPath] - Override binary path (for testing)
 * @param {string} [opts.packageVersion] - Override package version (for testing)
 * @param {object} [opts.fs] - Override fs module (for testing)
 * @param {object} [opts.childProcess] - Override child_process (for testing)
 * @returns {{ available: boolean, path: string, version: string|null, reason: string|null }}
 */
function checkBinary(opts) {
  opts = opts || {};
  const binaryPath = opts.binaryPath || BINARY_PATH;
  const pkgVersion = opts.packageVersion || require('../../package.json').version;
  const fsMod = opts.fs || fs;
  const cp = opts.childProcess || { execSync };

  if (!binaryPath) {
    return { available: false, path: binaryPath, version: null, reason: 'HOME not set' };
  }

  // Check binary exists
  if (!fsMod.existsSync(binaryPath)) {
    return { available: false, path: binaryPath, version: null, reason: 'binary not found' };
  }

  // Check executable
  try {
    fsMod.accessSync(binaryPath, fsMod.constants.X_OK);
  } catch {
    return { available: false, path: binaryPath, version: null, reason: 'binary not executable' };
  }

  // Get binary version — call `aether version` and strip "aether v" prefix
  let binaryVersion;
  try {
    const output = cp.execSync(`"${binaryPath}" version`, {
      encoding: 'utf8',
      timeout: 5000,
      stdio: ['pipe', 'pipe', 'pipe'],
    }).trim();
    // Strip "aether " or "aether v" prefix (e.g. "aether v5.3.3" → "5.3.3")
    binaryVersion = output.replace(/^aether\s+v?/, '');
  } catch {
    return { available: false, path: binaryPath, version: null, reason: 'binary version check failed' };
  }

  // Compare versions (compareVersions handles v-prefix on both sides)
  if (compareVersions(binaryVersion, pkgVersion) !== 0) {
    return {
      available: false,
      path: binaryPath,
      version: binaryVersion,
      reason: `version mismatch: binary=${binaryVersion}, package=${pkgVersion}`,
    };
  }

  return { available: true, path: binaryPath, version: binaryVersion, reason: null };
}

/**
 * Determine whether the current command should be delegated to the Go binary.
 *
 * @param {string[]} argv - process.argv (or equivalent)
 * @param {object} [opts] - Options (forwarded to checkBinary)
 * @returns {boolean} True if the command should be delegated to Go
 */
function shouldDelegate(argv, opts) {
  opts = opts || {};

  // Extract command name: skip node and script path
  // argv[0] = node, argv[1] = script path, argv[2] = command name
  const args = argv.slice(2);
  const command = args[0];

  // No command means help/version — still delegate if possible
  if (!command) {
    const check = checkBinary(opts);
    return check.available;
  }

  // Strip leading dashes
  const cleanCommand = command.replace(/^--?/, '');

  // Node-only commands never delegate (SHM-02)
  if (NODE_ONLY_COMMANDS.includes(cleanCommand)) {
    return false;
  }

  // Global flags (--help, --version, --no-color, --quiet) — delegate if binary available
  if (command.startsWith('-')) {
    const check = checkBinary(opts);
    return check.available;
  }

  // All other commands: delegate if version gate passes
  const check = checkBinary(opts);
  return check.available;
}

module.exports = {
  compareVersions,
  checkBinary,
  shouldDelegate,
  getBinaryPath,
  NODE_ONLY_COMMANDS,
};
