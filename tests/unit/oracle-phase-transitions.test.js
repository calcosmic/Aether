const test = require('ava');
const { execSync } = require('child_process');
const path = require('path');
const fs = require('fs');
const os = require('os');

const ORACLE_SH = path.join(__dirname, '../../.aether/oracle/oracle.sh');
const ORACLE_MD = path.join(__dirname, '../../.aether/oracle/oracle.md');
const PROJECT_ROOT = path.join(__dirname, '../..');

/**
 * Create a temp directory for test fixtures.
 * @returns {string}
 */
function createTmpDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), 'aether-oracle-phase-'));
}

/**
 * Run determine_phase by extracting the function from oracle.sh and sourcing it.
 * @param {string} stateFile - Path to state.json
 * @param {string} planFile - Path to plan.json
 * @returns {string} - The determined phase
 */
function runDeterminePhase(stateFile, planFile) {
  // Extract determine_phase function and call it in isolation
  const cmd = `bash -c 'set +e
# Extract the function from oracle.sh
eval "$(sed -n "/^determine_phase()/,/^}/p" "${ORACLE_SH}")"
determine_phase "${stateFile}" "${planFile}"
'`;
  return execSync(cmd, { encoding: 'utf8', cwd: PROJECT_ROOT }).trim();
}

/**
 * Run build_oracle_prompt by extracting the function from oracle.sh.
 * @param {string} stateFile - Path to state.json
 * @param {string} oracleMd - Path to oracle.md
 * @returns {string} - The built prompt
 */
function runBuildOraclePrompt(stateFile, oracleMd) {
  const cmd = `bash -c 'set +e
eval "$(sed -n "/^build_oracle_prompt()/,/^}/p" "${ORACLE_SH}")"
build_oracle_prompt "${stateFile}" "${oracleMd}"
'`;
  return execSync(cmd, { encoding: 'utf8', cwd: PROJECT_ROOT }).trim();
}

/**
 * Write a plan.json with specified question configurations.
 * @param {string} dir - Target directory
 * @param {Array} questions - Array of {confidence, touched, status} objects
 */
function writePlan(dir, questions) {
  const plan = {
    version: '1.0',
    questions: questions.map((q, i) => ({
      id: `q${i + 1}`,
      text: `Question ${i + 1}?`,
      status: q.status || 'open',
      confidence: q.confidence,
      key_findings: q.confidence > 0 ? ['some finding'] : [],
      iterations_touched: q.touched || []
    })),
    created_at: '2026-03-13T00:00:00Z',
    last_updated: '2026-03-13T00:00:00Z'
  };
  fs.writeFileSync(path.join(dir, 'plan.json'), JSON.stringify(plan, null, 2));
}

/**
 * Write a state.json with specified phase.
 * @param {string} dir - Target directory
 * @param {string} phase - Current phase
 * @param {object} overrides - Additional fields to override
 */
function writeState(dir, phase, overrides = {}) {
  const state = {
    version: '1.0',
    topic: 'Test topic',
    scope: 'codebase',
    phase,
    iteration: 0,
    max_iterations: 15,
    target_confidence: 95,
    overall_confidence: 0,
    started_at: '2026-03-13T00:00:00Z',
    last_updated: '2026-03-13T00:00:00Z',
    status: 'active',
    ...overrides
  };
  fs.writeFileSync(path.join(dir, 'state.json'), JSON.stringify(state, null, 2));
}


// ---- determine_phase tests ----

test('determine_phase: survey stays survey when no questions touched', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, 'survey');
  writePlan(dir, [
    { confidence: 0, touched: [] },
    { confidence: 0, touched: [] },
    { confidence: 0, touched: [] }
  ]);

  const phase = runDeterminePhase(
    path.join(dir, 'state.json'),
    path.join(dir, 'plan.json')
  );
  t.is(phase, 'survey', 'Should stay in survey when nothing is touched and avg confidence is 0');
});

test('determine_phase: survey -> investigate when all questions touched', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, 'survey');
  // All questions have been touched but low confidence (below 25% avg)
  writePlan(dir, [
    { confidence: 10, touched: [1] },
    { confidence: 15, touched: [1] },
    { confidence: 5, touched: [1] }
  ]);

  const phase = runDeterminePhase(
    path.join(dir, 'state.json'),
    path.join(dir, 'plan.json')
  );
  t.is(phase, 'investigate', 'Should transition to investigate when all questions are touched');
});

test('determine_phase: survey -> investigate when avg confidence >= 25', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, 'survey');
  // Not all touched, but avg confidence is 30%
  writePlan(dir, [
    { confidence: 40, touched: [1] },
    { confidence: 30, touched: [1] },
    { confidence: 20, touched: [] }
  ]);

  const phase = runDeterminePhase(
    path.join(dir, 'state.json'),
    path.join(dir, 'plan.json')
  );
  t.is(phase, 'investigate', 'Should transition to investigate when avg confidence >= 25%');
});

test('determine_phase: investigate stays when avg confidence < 60 and 2+ below 50', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, 'investigate');
  // Avg confidence ~45%, 3 questions below 50%
  writePlan(dir, [
    { confidence: 40, touched: [1, 2] },
    { confidence: 45, touched: [1, 2] },
    { confidence: 30, touched: [1] },
    { confidence: 65, touched: [1, 2] }
  ]);

  const phase = runDeterminePhase(
    path.join(dir, 'state.json'),
    path.join(dir, 'plan.json')
  );
  t.is(phase, 'investigate', 'Should stay in investigate when avg < 60 and 2+ questions below 50');
});

test('determine_phase: investigate -> synthesize when avg confidence >= 60', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, 'investigate');
  // Avg confidence 65%
  writePlan(dir, [
    { confidence: 60, touched: [1, 2] },
    { confidence: 70, touched: [1, 2] },
    { confidence: 65, touched: [1, 2] }
  ]);

  const phase = runDeterminePhase(
    path.join(dir, 'state.json'),
    path.join(dir, 'plan.json')
  );
  t.is(phase, 'synthesize', 'Should transition to synthesize when avg confidence >= 60%');
});

test('determine_phase: investigate -> synthesize when fewer than 2 below 50', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, 'investigate');
  // Only 1 question below 50%, avg is 55%
  writePlan(dir, [
    { confidence: 55, touched: [1, 2] },
    { confidence: 70, touched: [1, 2] },
    { confidence: 40, touched: [1] }
  ]);

  const phase = runDeterminePhase(
    path.join(dir, 'state.json'),
    path.join(dir, 'plan.json')
  );
  t.is(phase, 'synthesize', 'Should transition to synthesize when fewer than 2 questions below 50%');
});

test('determine_phase: synthesize -> verify when avg confidence >= 80', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, 'synthesize');
  // Avg confidence 82%
  writePlan(dir, [
    { confidence: 80, touched: [1, 2, 3], status: 'partial' },
    { confidence: 85, touched: [1, 2, 3], status: 'partial' },
    { confidence: 82, touched: [1, 2, 3], status: 'partial' }
  ]);

  const phase = runDeterminePhase(
    path.join(dir, 'state.json'),
    path.join(dir, 'plan.json')
  );
  t.is(phase, 'verify', 'Should transition to verify when avg confidence >= 80%');
});

test('determine_phase: synthesize stays when avg confidence < 80', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, 'synthesize');
  // Avg confidence 70%
  writePlan(dir, [
    { confidence: 65, touched: [1, 2, 3], status: 'partial' },
    { confidence: 75, touched: [1, 2, 3], status: 'partial' },
    { confidence: 70, touched: [1, 2, 3], status: 'partial' }
  ]);

  const phase = runDeterminePhase(
    path.join(dir, 'state.json'),
    path.join(dir, 'plan.json')
  );
  t.is(phase, 'synthesize', 'Should stay in synthesize when avg confidence < 80%');
});


// ---- build_oracle_prompt tests ----

test('build_oracle_prompt: survey prompt contains SURVEY directive', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, 'survey');

  const prompt = runBuildOraclePrompt(
    path.join(dir, 'state.json'),
    ORACLE_MD
  );
  t.true(prompt.includes('SURVEY'), 'Survey prompt should contain SURVEY directive');
  t.true(prompt.includes('Cast a wide net'), 'Survey prompt should include breadth instruction');
});

test('build_oracle_prompt: investigate prompt contains INVESTIGATE directive', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, 'investigate');

  const prompt = runBuildOraclePrompt(
    path.join(dir, 'state.json'),
    ORACLE_MD
  );
  t.true(prompt.includes('INVESTIGATE'), 'Investigate prompt should contain INVESTIGATE directive');
  t.true(prompt.includes('go DEEP'), 'Investigate prompt should include depth instruction');
});

test('build_oracle_prompt: prompt includes oracle.md content', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, 'survey');

  const prompt = runBuildOraclePrompt(
    path.join(dir, 'state.json'),
    ORACLE_MD
  );
  t.true(prompt.includes('Oracle Ant'), 'Prompt should include oracle.md content with "Oracle Ant"');
});


// ---- Edge case tests ----

test('determine_phase: zero questions returns survey', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, 'survey');
  // Empty questions array
  const plan = {
    version: '1.0',
    questions: [],
    created_at: '2026-03-13T00:00:00Z',
    last_updated: '2026-03-13T00:00:00Z'
  };
  fs.writeFileSync(path.join(dir, 'plan.json'), JSON.stringify(plan, null, 2));

  const phase = runDeterminePhase(
    path.join(dir, 'state.json'),
    path.join(dir, 'plan.json')
  );
  t.is(phase, 'survey', 'Zero questions should default to survey');
});

test('determine_phase: all answered at 100% goes to verify', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, 'synthesize');
  writePlan(dir, [
    { confidence: 100, touched: [1, 2, 3], status: 'answered' },
    { confidence: 100, touched: [1, 2, 3], status: 'answered' },
    { confidence: 100, touched: [1, 2, 3], status: 'answered' }
  ]);

  const phase = runDeterminePhase(
    path.join(dir, 'state.json'),
    path.join(dir, 'plan.json')
  );
  t.is(phase, 'verify', 'All questions at 100% should transition to verify');
});

test('determine_phase: boundary confidence 25 triggers investigate', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, 'survey');
  // Exactly 25% avg confidence, not all touched
  writePlan(dir, [
    { confidence: 25, touched: [1] },
    { confidence: 25, touched: [] },
    { confidence: 25, touched: [] }
  ]);

  const phase = runDeterminePhase(
    path.join(dir, 'state.json'),
    path.join(dir, 'plan.json')
  );
  t.is(phase, 'investigate', 'Boundary confidence of exactly 25 should trigger investigate');
});
