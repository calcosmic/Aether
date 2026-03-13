const test = require('ava');
const { execSync } = require('child_process');
const path = require('path');
const fs = require('fs');
const os = require('os');

const AETHER_UTILS_PATH = path.join(__dirname, '../../.aether/aether-utils.sh');
const PROJECT_ROOT = path.join(__dirname, '../..');

/**
 * Helper to run validate-oracle-state with an isolated ORACLE_DIR.
 * @param {string} subTarget - 'state', 'plan', or 'all'
 * @param {string} oracleDir - Path to temp oracle directory
 * @returns {object} - Parsed JSON result
 */
function runValidate(subTarget, oracleDir) {
  const cmd = `bash "${AETHER_UTILS_PATH}" validate-oracle-state ${subTarget}`;
  const output = execSync(cmd, {
    encoding: 'utf8',
    cwd: PROJECT_ROOT,
    env: { ...process.env, ORACLE_DIR: oracleDir }
  });
  return JSON.parse(output);
}

/**
 * Helper to run validate-oracle-state expecting failure.
 * @param {string} subTarget - 'state', 'plan', or 'all'
 * @param {string} oracleDir - Path to temp oracle directory
 * @returns {object} - Parsed JSON error result
 */
function runValidateExpectError(subTarget, oracleDir) {
  const cmd = `bash "${AETHER_UTILS_PATH}" validate-oracle-state ${subTarget}`;
  try {
    execSync(cmd, {
      encoding: 'utf8',
      cwd: PROJECT_ROOT,
      env: { ...process.env, ORACLE_DIR: oracleDir }
    });
    throw new Error('Expected command to fail');
  } catch (error) {
    if (error.status !== 0) {
      const sources = [error.stdout || '', error.stderr || ''];
      for (const source of sources) {
        const lines = source.trim().split('\n');
        for (let i = lines.length - 1; i >= 0; i--) {
          try { return JSON.parse(lines[i]); } catch (e) { continue; }
        }
      }
      return { ok: false, error: { message: error.stderr || error.message } };
    }
    throw error;
  }
}

/**
 * Create a temp directory for oracle state tests.
 * @returns {string} - Path to temp directory
 */
function createTmpDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), 'aether-oracle-'));
}

// Minimal valid state.json
const VALID_STATE = {
  version: '1.0',
  topic: 'Test topic',
  scope: 'codebase',
  phase: 'survey',
  iteration: 0,
  max_iterations: 15,
  target_confidence: 95,
  overall_confidence: 0,
  started_at: '2026-03-13T00:00:00Z',
  last_updated: '2026-03-13T00:00:00Z',
  status: 'active'
};

// Minimal valid plan.json
const VALID_PLAN = {
  version: '1.0',
  questions: [
    { id: 'q1', text: 'Test question?', status: 'open', confidence: 0, key_findings: [], iterations_touched: [] }
  ],
  created_at: '2026-03-13T00:00:00Z',
  last_updated: '2026-03-13T00:00:00Z'
};


// ---- Tests for validate-oracle-state state ----

test('validate-oracle-state state: valid state.json passes validation', t => {
  const tmpDir = createTmpDir();
  t.teardown(() => fs.rmSync(tmpDir, { recursive: true, force: true }));

  fs.writeFileSync(path.join(tmpDir, 'state.json'), JSON.stringify(VALID_STATE, null, 2));
  const result = runValidate('state', tmpDir);

  t.true(result.ok, 'ok should be true');
  t.true(result.result.pass, 'pass should be true for valid state');
  t.is(result.result.file, 'state.json');
  t.true(result.result.checks.every(c => c === 'pass'), 'all checks should pass');
});

test('validate-oracle-state state: missing required field fails', t => {
  const tmpDir = createTmpDir();
  t.teardown(() => fs.rmSync(tmpDir, { recursive: true, force: true }));

  const invalidState = { ...VALID_STATE };
  delete invalidState.topic;
  fs.writeFileSync(path.join(tmpDir, 'state.json'), JSON.stringify(invalidState, null, 2));
  const result = runValidate('state', tmpDir);

  t.true(result.ok, 'ok should be true (command succeeded)');
  t.false(result.result.pass, 'pass should be false for missing topic');
  t.true(result.result.checks.some(c => typeof c === 'string' && c.includes('topic')), 'should mention topic in failure');
});

test('validate-oracle-state state: wrong type fails', t => {
  const tmpDir = createTmpDir();
  t.teardown(() => fs.rmSync(tmpDir, { recursive: true, force: true }));

  const invalidState = { ...VALID_STATE, iteration: 'not-a-number' };
  fs.writeFileSync(path.join(tmpDir, 'state.json'), JSON.stringify(invalidState, null, 2));
  const result = runValidate('state', tmpDir);

  t.true(result.ok, 'ok should be true (command succeeded)');
  t.false(result.result.pass, 'pass should be false for wrong type');
  t.true(result.result.checks.some(c => typeof c === 'string' && c.includes('iteration')), 'should mention iteration in failure');
});

test('validate-oracle-state state: missing file returns error', t => {
  const tmpDir = createTmpDir();
  t.teardown(() => fs.rmSync(tmpDir, { recursive: true, force: true }));

  const result = runValidateExpectError('state', tmpDir);

  t.false(result.ok, 'ok should be false for missing file');
  t.truthy(result.error, 'should have error field');
});


// ---- Tests for validate-oracle-state plan ----

test('validate-oracle-state plan: valid plan.json passes', t => {
  const tmpDir = createTmpDir();
  t.teardown(() => fs.rmSync(tmpDir, { recursive: true, force: true }));

  const planWith3 = {
    ...VALID_PLAN,
    questions: [
      { id: 'q1', text: 'Question 1?', status: 'open', confidence: 0, key_findings: [], iterations_touched: [] },
      { id: 'q2', text: 'Question 2?', status: 'partial', confidence: 50, key_findings: ['finding1'], iterations_touched: [1] },
      { id: 'q3', text: 'Question 3?', status: 'answered', confidence: 100, key_findings: ['finding2'], iterations_touched: [1, 2] }
    ]
  };
  fs.writeFileSync(path.join(tmpDir, 'plan.json'), JSON.stringify(planWith3, null, 2));
  const result = runValidate('plan', tmpDir);

  t.true(result.ok, 'ok should be true');
  t.true(result.result.pass, 'pass should be true for valid plan');
  t.is(result.result.file, 'plan.json');
});

test('validate-oracle-state plan: question missing required fields fails', t => {
  const tmpDir = createTmpDir();
  t.teardown(() => fs.rmSync(tmpDir, { recursive: true, force: true }));

  const invalidPlan = {
    ...VALID_PLAN,
    questions: [
      { id: 'q1', text: 'Test?', confidence: 0, key_findings: [], iterations_touched: [] }
      // missing 'status'
    ]
  };
  fs.writeFileSync(path.join(tmpDir, 'plan.json'), JSON.stringify(invalidPlan, null, 2));
  const result = runValidate('plan', tmpDir);

  t.true(result.ok, 'ok should be true (command succeeded)');
  t.false(result.result.pass, 'pass should be false for missing status');
});

test('validate-oracle-state plan: invalid status value fails', t => {
  const tmpDir = createTmpDir();
  t.teardown(() => fs.rmSync(tmpDir, { recursive: true, force: true }));

  const invalidPlan = {
    ...VALID_PLAN,
    questions: [
      { id: 'q1', text: 'Test?', status: 'skipped', confidence: 0, key_findings: [], iterations_touched: [] }
    ]
  };
  fs.writeFileSync(path.join(tmpDir, 'plan.json'), JSON.stringify(invalidPlan, null, 2));
  const result = runValidate('plan', tmpDir);

  t.true(result.ok, 'ok should be true (command succeeded)');
  t.false(result.result.pass, 'pass should be false for invalid status');
  t.true(result.result.checks.some(c => typeof c === 'string' && c.includes('status')), 'should mention status in failure');
});

test('validate-oracle-state plan: confidence out of range fails', t => {
  const tmpDir = createTmpDir();
  t.teardown(() => fs.rmSync(tmpDir, { recursive: true, force: true }));

  const invalidPlan = {
    ...VALID_PLAN,
    questions: [
      { id: 'q1', text: 'Test?', status: 'open', confidence: 150, key_findings: [], iterations_touched: [] }
    ]
  };
  fs.writeFileSync(path.join(tmpDir, 'plan.json'), JSON.stringify(invalidPlan, null, 2));
  const result = runValidate('plan', tmpDir);

  t.true(result.ok, 'ok should be true (command succeeded)');
  t.false(result.result.pass, 'pass should be false for out-of-range confidence');
  t.true(result.result.checks.some(c => typeof c === 'string' && c.includes('confidence')), 'should mention confidence in failure');
});

test('validate-oracle-state plan: too many questions fails', t => {
  const tmpDir = createTmpDir();
  t.teardown(() => fs.rmSync(tmpDir, { recursive: true, force: true }));

  const questions = [];
  for (let i = 1; i <= 9; i++) {
    questions.push({ id: `q${i}`, text: `Question ${i}?`, status: 'open', confidence: 0, key_findings: [], iterations_touched: [] });
  }
  const invalidPlan = { ...VALID_PLAN, questions };
  fs.writeFileSync(path.join(tmpDir, 'plan.json'), JSON.stringify(invalidPlan, null, 2));
  const result = runValidate('plan', tmpDir);

  t.true(result.ok, 'ok should be true (command succeeded)');
  t.false(result.result.pass, 'pass should be false for 9 questions (max 8)');
  t.true(result.result.checks.some(c => typeof c === 'string' && c.includes('questions count')), 'should mention questions count');
});

test('validate-oracle-state plan: zero questions fails', t => {
  const tmpDir = createTmpDir();
  t.teardown(() => fs.rmSync(tmpDir, { recursive: true, force: true }));

  const invalidPlan = { ...VALID_PLAN, questions: [] };
  fs.writeFileSync(path.join(tmpDir, 'plan.json'), JSON.stringify(invalidPlan, null, 2));
  const result = runValidate('plan', tmpDir);

  t.true(result.ok, 'ok should be true (command succeeded)');
  t.false(result.result.pass, 'pass should be false for 0 questions');
  t.true(result.result.checks.some(c => typeof c === 'string' && c.includes('questions count')), 'should mention questions count');
});


// ---- Tests for validate-oracle-state all ----

test('validate-oracle-state all: both valid passes', t => {
  const tmpDir = createTmpDir();
  t.teardown(() => fs.rmSync(tmpDir, { recursive: true, force: true }));

  fs.writeFileSync(path.join(tmpDir, 'state.json'), JSON.stringify(VALID_STATE, null, 2));
  fs.writeFileSync(path.join(tmpDir, 'plan.json'), JSON.stringify(VALID_PLAN, null, 2));
  const result = runValidate('all', tmpDir);

  t.true(result.ok, 'ok should be true');
  t.true(result.result.pass, 'overall pass should be true');
  t.is(result.result.files.length, 2, 'should have results for 2 files');
});

test('validate-oracle-state all: one invalid fails overall', t => {
  const tmpDir = createTmpDir();
  t.teardown(() => fs.rmSync(tmpDir, { recursive: true, force: true }));

  fs.writeFileSync(path.join(tmpDir, 'state.json'), JSON.stringify(VALID_STATE, null, 2));
  const invalidPlan = { ...VALID_PLAN, questions: [] };
  fs.writeFileSync(path.join(tmpDir, 'plan.json'), JSON.stringify(invalidPlan, null, 2));
  const result = runValidate('all', tmpDir);

  t.true(result.ok, 'ok should be true (command succeeded)');
  t.false(result.result.pass, 'overall pass should be false when plan is invalid');
});
