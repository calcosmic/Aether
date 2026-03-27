#!/usr/bin/env node
/**
 * State API unit tests
 *
 * Tests state-read, state-read-field, state-mutate, state-write subcommands
 * via aether-utils.sh with isolated temp directories.
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync, spawnSync } = require('child_process');

const REPO_ROOT = path.join(__dirname, '..', '..');
const AETHER_UTILS = path.join(REPO_ROOT, '.aether', 'aether-utils.sh');

function createTempDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), 'aether-state-api-'));
}

function cleanupTempDir(tempDir) {
  fs.rmSync(tempDir, { recursive: true, force: true });
}

function setupTempAether(tempDir) {
  const srcAetherDir = path.join(REPO_ROOT, '.aether');
  const dstAetherDir = path.join(tempDir, '.aether');
  const dstDataDir = path.join(dstAetherDir, 'data');

  fs.mkdirSync(dstAetherDir, { recursive: true });
  fs.mkdirSync(dstDataDir, { recursive: true });
  fs.copyFileSync(path.join(srcAetherDir, 'aether-utils.sh'), path.join(dstAetherDir, 'aether-utils.sh'));
  fs.cpSync(path.join(srcAetherDir, 'utils'), path.join(dstAetherDir, 'utils'), { recursive: true });

  const srcExchangeDir = path.join(srcAetherDir, 'exchange');
  if (fs.existsSync(srcExchangeDir)) {
    fs.cpSync(srcExchangeDir, path.join(dstAetherDir, 'exchange'), { recursive: true });
  }
}

function seedState(tempDir) {
  const state = {
    version: '3.0',
    goal: 'Test state API facade',
    state: 'READY',
    current_phase: 1,
    session_id: 'session_api_test',
    initialized_at: '2026-01-01T00:00:00Z',
    build_started_at: null,
    plan: {
      generated_at: null,
      confidence: null,
      phases: [
        { id: 1, name: 'Phase One', status: 'pending', tasks: [], success_criteria: [] }
      ]
    },
    memory: { phase_learnings: [], decisions: [], instincts: [] },
    errors: { records: [], flagged_patterns: [] },
    events: [],
    signals: [],
    graveyards: []
  };

  fs.writeFileSync(
    path.join(tempDir, '.aether', 'data', 'COLONY_STATE.json'),
    JSON.stringify(state, null, 2)
  );
}

function runUtil(tempDir, subcommand, args = []) {
  const env = {
    ...process.env,
    AETHER_ROOT: tempDir,
    DATA_DIR: path.join(tempDir, '.aether', 'data')
  };
  const quoted = args.map((a) => `"${String(a).replace(/"/g, '\\"')}"`).join(' ');
  const cmd = `bash .aether/aether-utils.sh ${subcommand} ${quoted}`;
  const out = execSync(cmd, {
    cwd: tempDir,
    env,
    encoding: 'utf8',
    stdio: ['pipe', 'pipe', 'pipe']
  });
  return JSON.parse(out);
}

function runUtilExpectError(tempDir, subcommand, args = []) {
  const env = {
    ...process.env,
    AETHER_ROOT: tempDir,
    DATA_DIR: path.join(tempDir, '.aether', 'data')
  };
  const quoted = args.map((a) => `"${String(a).replace(/"/g, '\\"')}"`).join(' ');
  const cmd = `bash .aether/aether-utils.sh ${subcommand} ${quoted}`;
  const result = spawnSync('bash', ['-c', cmd], {
    cwd: tempDir,
    env,
    encoding: 'utf8'
  });
  // Parse error JSON from stderr
  const lines = (result.stderr || '').trim().split('\n');
  for (let i = lines.length - 1; i >= 0; i--) {
    try { return JSON.parse(lines[i]); } catch (e) { continue; }
  }
  return { ok: false, error: { message: result.stderr || 'unknown error' } };
}

// ============================================================================
// state-read tests
// ============================================================================

test('state-read returns full state JSON', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);
    seedState(tempDir);

    const out = runUtil(tempDir, 'state-read');
    t.true(out.ok);
    t.is(out.result.goal, 'Test state API facade');
    t.is(out.result.version, '3.0');
    t.is(out.result.current_phase, 1);
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('state-read returns error for missing file', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);
    // No seedState -- COLONY_STATE.json doesn't exist

    const out = runUtilExpectError(tempDir, 'state-read');
    t.false(out.ok);
    t.is(out.error.code, 'E_FILE_NOT_FOUND');
  } finally {
    cleanupTempDir(tempDir);
  }
});

// ============================================================================
// state-read-field tests
// ============================================================================

test('state-read-field returns specific field value', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);
    seedState(tempDir);

    const out = runUtil(tempDir, 'state-read-field', ['.goal']);
    t.true(out.ok);
    t.is(out.result, 'Test state API facade');
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('state-read-field returns numeric field', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);
    seedState(tempDir);

    const out = runUtil(tempDir, 'state-read-field', ['.current_phase']);
    t.true(out.ok);
    t.is(out.result, 1);
  } finally {
    cleanupTempDir(tempDir);
  }
});

// ============================================================================
// state-mutate tests
// ============================================================================

test('state-mutate applies jq expression and persists', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);
    seedState(tempDir);

    const out = runUtil(tempDir, 'state-mutate', ['.state = "EXECUTING"']);
    t.true(out.ok);
    t.true(out.result.mutated);

    // Verify persistence
    const verify = runUtil(tempDir, 'state-read-field', ['.state']);
    t.is(verify.result, 'EXECUTING');
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('state-mutate on missing file returns error', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);
    // No seedState

    const out = runUtilExpectError(tempDir, 'state-mutate', ['.state = "EXECUTING"']);
    t.false(out.ok);
    t.is(out.error.code, 'E_FILE_NOT_FOUND');
  } finally {
    cleanupTempDir(tempDir);
  }
});

// ============================================================================
// state-write backward compatibility tests
// ============================================================================

test('state-write accepts JSON and persists it', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);
    seedState(tempDir);

    const newState = JSON.stringify({
      version: '3.0',
      goal: 'Written by state-write',
      state: 'READY',
      current_phase: 2,
      plan: { phases: [] },
      memory: {},
      errors: { records: [] },
      events: []
    });

    const out = runUtil(tempDir, 'state-write', [newState]);
    t.true(out.ok);
    t.true(out.result.written);

    // Verify
    const verify = runUtil(tempDir, 'state-read-field', ['.goal']);
    t.is(verify.result, 'Written by state-write');
  } finally {
    cleanupTempDir(tempDir);
  }
});
