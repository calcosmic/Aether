#!/usr/bin/env node
/**
 * Pending Decisions Queue unit tests
 *
 * Tests pending-decision-add, pending-decision-list, pending-decision-resolve
 * and autopilot-headless-check, autopilot-set-headless subcommands
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
  return fs.mkdtempSync(path.join(os.tmpdir(), 'aether-pending-decisions-'));
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

function runUtilRaw(tempDir, subcommand, args = []) {
  const env = {
    ...process.env,
    AETHER_ROOT: tempDir,
    DATA_DIR: path.join(tempDir, '.aether', 'data')
  };
  const quoted = args.map((a) => `"${String(a).replace(/"/g, '\\"')}"`).join(' ');
  const cmd = `bash .aether/aether-utils.sh ${subcommand} ${quoted}`;
  return spawnSync('bash', ['-c', cmd], {
    cwd: tempDir,
    env,
    encoding: 'utf8'
  });
}

function runUtilExpectError(tempDir, subcommand, args = []) {
  const result = runUtilRaw(tempDir, subcommand, args);
  const lines = (result.stderr || '').trim().split('\n');
  for (let i = lines.length - 1; i >= 0; i--) {
    try { return JSON.parse(lines[i]); } catch (e) { continue; }
  }
  return { ok: false, error: { message: result.stderr || 'unknown error' } };
}

// ============================================================================
// pending-decision-add tests
// ============================================================================

test('pending-decision-add creates pending-decisions.json with first entry', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);

    const out = runUtil(tempDir, 'pending-decision-add', [
      '--type', 'replan',
      '--description', 'Need to re-evaluate phase plan'
    ]);

    t.true(out.ok);
    t.truthy(out.result.id);
    t.regex(out.result.id, /^pd_/);
    t.is(out.result.decision_count, 1);

    // Verify file was created with correct structure
    const dataDir = path.join(tempDir, '.aether', 'data');
    const filePath = path.join(dataDir, 'pending-decisions.json');
    t.true(fs.existsSync(filePath));

    const fileContent = JSON.parse(fs.readFileSync(filePath, 'utf8'));
    t.is(fileContent.version, '1.0');
    t.is(fileContent.decisions.length, 1);
    t.is(fileContent.decisions[0].type, 'replan');
    t.is(fileContent.decisions[0].description, 'Need to re-evaluate phase plan');
    t.false(fileContent.decisions[0].resolved);
    t.truthy(fileContent.decisions[0].created_at);
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('pending-decision-add appends to existing decisions', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);

    runUtil(tempDir, 'pending-decision-add', [
      '--type', 'replan',
      '--description', 'First decision'
    ]);

    const out = runUtil(tempDir, 'pending-decision-add', [
      '--type', 'escalation',
      '--description', 'Second decision',
      '--phase', '3',
      '--source', 'chaos-agent'
    ]);

    t.true(out.ok);
    t.is(out.result.decision_count, 2);

    const dataDir = path.join(tempDir, '.aether', 'data');
    const fileContent = JSON.parse(fs.readFileSync(
      path.join(dataDir, 'pending-decisions.json'), 'utf8'
    ));
    t.is(fileContent.decisions.length, 2);
    t.is(fileContent.decisions[1].type, 'escalation');
    t.is(fileContent.decisions[1].phase, 3);
    t.is(fileContent.decisions[1].source, 'chaos-agent');
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('pending-decision-add accepts all valid types', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);
    const validTypes = ['visual_checkpoint', 'replan', 'escalation', 'runtime_verification', 'user_input'];

    for (const type of validTypes) {
      const out = runUtil(tempDir, 'pending-decision-add', [
        '--type', type,
        '--description', `Decision of type ${type}`
      ]);
      t.true(out.ok, `Should accept type: ${type}`);
    }
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('pending-decision-add requires --type', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);

    const out = runUtilExpectError(tempDir, 'pending-decision-add', [
      '--description', 'Missing type'
    ]);

    t.false(out.ok);
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('pending-decision-add requires --description', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);

    const out = runUtilExpectError(tempDir, 'pending-decision-add', [
      '--type', 'replan'
    ]);

    t.false(out.ok);
  } finally {
    cleanupTempDir(tempDir);
  }
});

// ============================================================================
// pending-decision-list tests
// ============================================================================

test('pending-decision-list returns empty result when no file exists', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);

    const out = runUtil(tempDir, 'pending-decision-list');

    t.true(out.ok);
    t.is(out.result.total, 0);
    t.is(out.result.unresolved, 0);
    t.deepEqual(out.result.decisions, []);
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('pending-decision-list shows only unresolved by default', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);

    const add1 = runUtil(tempDir, 'pending-decision-add', [
      '--type', 'replan', '--description', 'Decision 1'
    ]);
    runUtil(tempDir, 'pending-decision-add', [
      '--type', 'escalation', '--description', 'Decision 2'
    ]);

    // Resolve the first one
    runUtil(tempDir, 'pending-decision-resolve', [
      '--id', add1.result.id,
      '--resolution', 'Resolved manually'
    ]);

    const out = runUtil(tempDir, 'pending-decision-list');

    t.true(out.ok);
    t.is(out.result.unresolved, 1);
    t.is(out.result.decisions.length, 1);
    t.is(out.result.decisions[0].description, 'Decision 2');
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('pending-decision-list --unresolved filters correctly', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);

    runUtil(tempDir, 'pending-decision-add', [
      '--type', 'replan', '--description', 'Unresolved'
    ]);

    const out = runUtil(tempDir, 'pending-decision-list', ['--unresolved']);

    t.true(out.ok);
    t.is(out.result.decisions.length, 1);
    t.false(out.result.decisions[0].resolved);
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('pending-decision-list --type filters by type', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);

    runUtil(tempDir, 'pending-decision-add', ['--type', 'replan', '--description', 'A replan']);
    runUtil(tempDir, 'pending-decision-add', ['--type', 'escalation', '--description', 'An escalation']);
    runUtil(tempDir, 'pending-decision-add', ['--type', 'replan', '--description', 'Another replan']);

    const out = runUtil(tempDir, 'pending-decision-list', ['--type', 'replan']);

    t.true(out.ok);
    t.is(out.result.decisions.length, 2);
    t.true(out.result.decisions.every(d => d.type === 'replan'));
  } finally {
    cleanupTempDir(tempDir);
  }
});

// ============================================================================
// pending-decision-resolve tests
// ============================================================================

test('pending-decision-resolve marks decision resolved with timestamp', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);

    const addOut = runUtil(tempDir, 'pending-decision-add', [
      '--type', 'replan',
      '--description', 'Need replanning'
    ]);
    const id = addOut.result.id;

    const resolveOut = runUtil(tempDir, 'pending-decision-resolve', [
      '--id', id,
      '--resolution', 'Decided to continue as-is'
    ]);

    t.true(resolveOut.ok);
    t.true(resolveOut.result.resolved);
    t.is(resolveOut.result.id, id);

    // Verify in file
    const dataDir = path.join(tempDir, '.aether', 'data');
    const fileContent = JSON.parse(fs.readFileSync(
      path.join(dataDir, 'pending-decisions.json'), 'utf8'
    ));
    const decision = fileContent.decisions.find(d => d.id === id);
    t.true(decision.resolved);
    t.is(decision.resolution, 'Decided to continue as-is');
    t.truthy(decision.resolved_at);
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('pending-decision-resolve requires --id', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);

    const out = runUtilExpectError(tempDir, 'pending-decision-resolve', [
      '--resolution', 'some resolution'
    ]);

    t.false(out.ok);
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('pending-decision-resolve requires --resolution', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);

    const out = runUtilExpectError(tempDir, 'pending-decision-resolve', [
      '--id', 'pd_12345_99'
    ]);

    t.false(out.ok);
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('pending-decision-resolve returns error for unknown id', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);

    runUtil(tempDir, 'pending-decision-add', [
      '--type', 'replan', '--description', 'Exists'
    ]);

    const out = runUtilExpectError(tempDir, 'pending-decision-resolve', [
      '--id', 'pd_999_nonexistent',
      '--resolution', 'Does not exist'
    ]);

    t.false(out.ok);
  } finally {
    cleanupTempDir(tempDir);
  }
});

// ============================================================================
// autopilot-headless-check tests
// ============================================================================

test('autopilot-headless-check returns headless false when no run-state.json', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);

    const out = runUtil(tempDir, 'autopilot-headless-check');

    t.true(out.ok);
    t.false(out.result.headless);
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('autopilot-headless-check returns headless false when run-state.json has no headless field', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);

    // Initialize autopilot without headless flag
    runUtil(tempDir, 'autopilot-init', [
      '--total-phases', '5',
      '--start-phase', '1'
    ]);

    const out = runUtil(tempDir, 'autopilot-headless-check');

    t.true(out.ok);
    t.false(out.result.headless);
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('autopilot-headless-check returns headless true after set-headless true', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);

    runUtil(tempDir, 'autopilot-init', [
      '--total-phases', '5',
      '--start-phase', '1'
    ]);
    runUtil(tempDir, 'autopilot-set-headless', ['true']);

    const out = runUtil(tempDir, 'autopilot-headless-check');

    t.true(out.ok);
    t.true(out.result.headless);
  } finally {
    cleanupTempDir(tempDir);
  }
});

// ============================================================================
// autopilot-set-headless tests
// ============================================================================

test('autopilot-set-headless creates run-state.json with headless flag', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);

    runUtil(tempDir, 'autopilot-init', [
      '--total-phases', '3',
      '--start-phase', '1'
    ]);

    const out = runUtil(tempDir, 'autopilot-set-headless', ['true']);

    t.true(out.ok);
    t.true(out.result.headless);
    t.true(out.result.updated);

    // Verify in file
    const dataDir = path.join(tempDir, '.aether', 'data');
    const runState = JSON.parse(fs.readFileSync(
      path.join(dataDir, 'run-state.json'), 'utf8'
    ));
    t.true(runState.headless);
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('autopilot-set-headless false clears headless flag', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);

    runUtil(tempDir, 'autopilot-init', [
      '--total-phases', '3',
      '--start-phase', '1'
    ]);
    runUtil(tempDir, 'autopilot-set-headless', ['true']);
    const out = runUtil(tempDir, 'autopilot-set-headless', ['false']);

    t.true(out.ok);
    t.false(out.result.headless);
    t.true(out.result.updated);

    const dataDir = path.join(tempDir, '.aether', 'data');
    const runState = JSON.parse(fs.readFileSync(
      path.join(dataDir, 'run-state.json'), 'utf8'
    ));
    t.false(runState.headless);
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('autopilot-set-headless requires true or false argument', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);

    runUtil(tempDir, 'autopilot-init', [
      '--total-phases', '3',
      '--start-phase', '1'
    ]);

    const out = runUtilExpectError(tempDir, 'autopilot-set-headless', ['maybe']);

    t.false(out.ok);
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('autopilot-set-headless requires run-state.json to exist', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);
    // No autopilot-init called

    const out = runUtilExpectError(tempDir, 'autopilot-set-headless', ['true']);

    t.false(out.ok);
  } finally {
    cleanupTempDir(tempDir);
  }
});
