const test = require('ava');
const { execSync } = require('child_process');
const path = require('path');
const fs = require('fs');
const os = require('os');

const ORACLE_SH = path.join(__dirname, '../../.aether/utils/oracle/oracle.sh');
const PROJECT_ROOT = path.join(__dirname, '../..');

/**
 * Create a temp directory for test fixtures.
 * @returns {string}
 */
function createTmpDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), 'aether-oracle-steering-'));
}

/**
 * Write state.json with defaults merged with overrides.
 * @param {string} dir - Target directory
 * @param {object} overrides - Fields to override
 */
function writeState(dir, overrides = {}) {
  const defaults = {
    version: '1.1',
    topic: 'Test topic',
    scope: 'both',
    phase: 'survey',
    iteration: 0,
    max_iterations: 15,
    target_confidence: 95,
    overall_confidence: 0,
    started_at: '2026-03-13T00:00:00Z',
    last_updated: '2026-03-13T00:00:00Z',
    status: 'active',
    strategy: 'adaptive',
    focus_areas: []
  };
  const state = Object.assign({}, defaults, overrides);
  fs.writeFileSync(path.join(dir, 'state.json'), JSON.stringify(state, null, 2));
}

/**
 * Write a plan.json with structured findings and sources registry.
 * @param {string} dir - Target directory
 * @param {Array} questions - Array of {confidence, touched, status, findings} objects
 * @param {object} sources - Sources registry
 */
function writePlan(dir, questions, sources = {}) {
  const plan = {
    version: '1.1',
    sources,
    questions: questions.map((q, i) => ({
      id: `q${i + 1}`,
      text: `Question ${i + 1}?`,
      status: q.status || 'open',
      confidence: q.confidence,
      key_findings: q.findings || [],
      iterations_touched: q.touched || []
    })),
    created_at: '2026-03-13T00:00:00Z',
    last_updated: '2026-03-13T00:00:00Z'
  };
  fs.writeFileSync(path.join(dir, 'plan.json'), JSON.stringify(plan, null, 2));
}

/**
 * Write a mock aether-utils.sh that returns known pheromone-read JSON.
 * @param {string} dir - Root dir (will create .aether/ subdir)
 * @param {Array} signals - Array of signal objects to return
 */
function writePheromones(dir, signals) {
  const aetherDir = path.join(dir, '.aether');
  const dataDir = path.join(aetherDir, 'data');
  fs.mkdirSync(dataDir, { recursive: true });

  // Write a mock aether-utils.sh that returns the signals via pheromone-read
  const signalsJson = JSON.stringify({ ok: true, result: { signals } });
  const mockScript = `#!/bin/bash
case "$1" in
  pheromone-read) echo '${signalsJson.replace(/'/g, "'\\''")}';;
esac
`;
  fs.writeFileSync(path.join(aetherDir, 'aether-utils.sh'), mockScript, { mode: 0o755 });
}

/**
 * Run read_steering_signals by extracting it from oracle.sh and calling with aetherRoot.
 * @param {string} aetherRoot - Root dir containing .aether/aether-utils.sh mock
 * @returns {string} - Output from read_steering_signals
 */
function runReadSteeringSignals(aetherRoot) {
  const cmd = `bash -c 'set +e
eval "$(sed -n "/^read_steering_signals()/,/^}/p" "${ORACLE_SH}")"
read_steering_signals "${aetherRoot}"
'`;
  try {
    return execSync(cmd, { encoding: 'utf8', cwd: PROJECT_ROOT }).trimEnd();
  } catch (e) {
    return (e.stdout || '').trimEnd();
  }
}

/**
 * Run build_oracle_prompt by extracting it from oracle.sh.
 * @param {string} stateFile - Path to state.json
 * @param {string} oracleMd - Path to oracle.md
 * @param {string} steeringDirective - Optional steering directive text
 * @returns {string} - The prompt output
 */
function runBuildOraclePrompt(stateFile, oracleMd, steeringDirective = '') {
  const cmd = `bash -c 'set +e
eval "$(sed -n "/^build_oracle_prompt()/,/^}/p" "${ORACLE_SH}")"
build_oracle_prompt "${stateFile}" "${oracleMd}" "${steeringDirective}"
'`;
  return execSync(cmd, { encoding: 'utf8', cwd: PROJECT_ROOT }).trimEnd();
}

/**
 * Run validate-oracle-state for state.json validation.
 * @param {string} dir - Directory containing state.json
 * @returns {object} - Parsed JSON result
 */
function runValidateOracleState(dir) {
  const utilsPath = path.join(PROJECT_ROOT, '.aether/aether-utils.sh');
  const cmd = `ORACLE_DIR="${dir}" bash "${utilsPath}" validate-oracle-state state`;
  const output = execSync(cmd, { encoding: 'utf8', cwd: PROJECT_ROOT }).trim();
  return JSON.parse(output);
}


// ==== read_steering_signals tests ====

test('read_steering_signals: returns empty when no signals', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writePheromones(dir, []);
  const output = runReadSteeringSignals(dir);
  t.is(output, '', 'should return empty string when no signals exist');
});

test('read_steering_signals: formats FOCUS signals', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writePheromones(dir, [
    { id: 'sig_1', type: 'FOCUS', content: { text: 'Security implications' }, effective_strength: 0.9, created_at: '2026-03-13T00:00:00Z', expires_at: '2026-03-14T00:00:00Z' },
    { id: 'sig_2', type: 'FOCUS', content: { text: 'Performance analysis' }, effective_strength: 0.7, created_at: '2026-03-13T00:00:00Z', expires_at: '2026-03-14T00:00:00Z' }
  ]);

  const output = runReadSteeringSignals(dir);
  t.regex(output, /FOCUS \(Prioritize these areas\)/, 'should contain FOCUS header');
  t.true(output.includes('Security implications'), 'should contain first FOCUS signal text');
  t.true(output.includes('Performance analysis'), 'should contain second FOCUS signal text');
});

test('read_steering_signals: formats REDIRECT signals', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writePheromones(dir, [
    { id: 'sig_r1', type: 'REDIRECT', content: { text: 'Do not use deprecated APIs' }, effective_strength: 0.95, created_at: '2026-03-13T00:00:00Z', expires_at: '2026-03-14T00:00:00Z' }
  ]);

  const output = runReadSteeringSignals(dir);
  t.regex(output, /REDIRECT \(Hard constraints -- MUST follow\)/, 'should contain REDIRECT header');
  t.true(output.includes('Do not use deprecated APIs'), 'should contain REDIRECT signal text');
});

test('read_steering_signals: formats FEEDBACK signals', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writePheromones(dir, [
    { id: 'sig_f1', type: 'FEEDBACK', content: { text: 'Include more code examples' }, effective_strength: 0.6, created_at: '2026-03-13T00:00:00Z', expires_at: '2026-03-14T00:00:00Z' }
  ]);

  const output = runReadSteeringSignals(dir);
  t.regex(output, /FEEDBACK \(Adjust approach\)/, 'should contain FEEDBACK header');
  t.true(output.includes('Include more code examples'), 'should contain FEEDBACK signal text');
});

test('read_steering_signals: respects signal limits (max 3 FOCUS)', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writePheromones(dir, [
    { id: 'sig_a', type: 'FOCUS', content: { text: 'Focus area A' }, effective_strength: 0.9, created_at: '2026-03-13T00:00:00Z', expires_at: '2026-03-14T00:00:00Z' },
    { id: 'sig_b', type: 'FOCUS', content: { text: 'Focus area B' }, effective_strength: 0.8, created_at: '2026-03-13T00:00:00Z', expires_at: '2026-03-14T00:00:00Z' },
    { id: 'sig_c', type: 'FOCUS', content: { text: 'Focus area C' }, effective_strength: 0.7, created_at: '2026-03-13T00:00:00Z', expires_at: '2026-03-14T00:00:00Z' },
    { id: 'sig_d', type: 'FOCUS', content: { text: 'Focus area D' }, effective_strength: 0.6, created_at: '2026-03-13T00:00:00Z', expires_at: '2026-03-14T00:00:00Z' },
    { id: 'sig_e', type: 'FOCUS', content: { text: 'Focus area E' }, effective_strength: 0.5, created_at: '2026-03-13T00:00:00Z', expires_at: '2026-03-14T00:00:00Z' }
  ]);

  const output = runReadSteeringSignals(dir);
  // Should contain top 3 (A, B, C) but not D or E
  t.true(output.includes('Focus area A'), 'should contain highest-strength FOCUS');
  t.true(output.includes('Focus area B'), 'should contain second-highest FOCUS');
  t.true(output.includes('Focus area C'), 'should contain third-highest FOCUS');
  t.false(output.includes('Focus area D'), 'should NOT contain fourth FOCUS (over limit)');
  t.false(output.includes('Focus area E'), 'should NOT contain fifth FOCUS (over limit)');
});

test('read_steering_signals: handles mixed signal types in correct order', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writePheromones(dir, [
    { id: 'sig_f', type: 'FOCUS', content: { text: 'Focus signal' }, effective_strength: 0.8, created_at: '2026-03-13T00:00:00Z', expires_at: '2026-03-14T00:00:00Z' },
    { id: 'sig_r', type: 'REDIRECT', content: { text: 'Redirect signal' }, effective_strength: 0.9, created_at: '2026-03-13T00:00:00Z', expires_at: '2026-03-14T00:00:00Z' },
    { id: 'sig_fb', type: 'FEEDBACK', content: { text: 'Feedback signal' }, effective_strength: 0.7, created_at: '2026-03-13T00:00:00Z', expires_at: '2026-03-14T00:00:00Z' }
  ]);

  const output = runReadSteeringSignals(dir);
  const redirectIdx = output.indexOf('REDIRECT');
  const focusIdx = output.indexOf('FOCUS');
  const feedbackIdx = output.indexOf('FEEDBACK');

  // All three headers should appear
  t.true(redirectIdx >= 0, 'should contain REDIRECT header');
  t.true(focusIdx >= 0, 'should contain FOCUS header');
  t.true(feedbackIdx >= 0, 'should contain FEEDBACK header');

  // REDIRECT should appear before FOCUS, FOCUS before FEEDBACK
  t.true(redirectIdx < focusIdx, 'REDIRECT should appear before FOCUS');
  t.true(focusIdx < feedbackIdx, 'FOCUS should appear before FEEDBACK');
});

test('read_steering_signals: graceful on missing utils (nonexistent aether_root)', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  // Don't create .aether/aether-utils.sh -- test graceful degradation
  const output = runReadSteeringSignals(path.join(dir, 'nonexistent'));
  t.is(output, '', 'should return empty string when utils not found');
});


// ==== build_oracle_prompt strategy tests ====

test('build_oracle_prompt: includes strategy modifier for breadth-first', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, { strategy: 'breadth-first' });
  const oracleMd = path.join(dir, 'oracle.md');
  fs.writeFileSync(oracleMd, 'BASE_PROMPT_MARKER');

  const output = runBuildOraclePrompt(path.join(dir, 'state.json'), oracleMd);
  t.true(output.includes('Breadth-first'), 'should contain Breadth-first keyword');
  t.true(output.includes('covering ALL questions') || output.includes('ALL questions'),
    'should mention covering all questions');
  t.true(output.includes('STRATEGY NOTE'), 'should contain STRATEGY NOTE marker');
});

test('build_oracle_prompt: includes strategy modifier for depth-first', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, { strategy: 'depth-first' });
  const oracleMd = path.join(dir, 'oracle.md');
  fs.writeFileSync(oracleMd, 'BASE_PROMPT_MARKER');

  const output = runBuildOraclePrompt(path.join(dir, 'state.json'), oracleMd);
  t.true(output.includes('Depth-first'), 'should contain Depth-first keyword');
  t.true(output.includes('exhaustively'), 'should mention investigating exhaustively');
  t.true(output.includes('STRATEGY NOTE'), 'should contain STRATEGY NOTE marker');
});

test('build_oracle_prompt: omits strategy modifier for adaptive', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, { strategy: 'adaptive' });
  const oracleMd = path.join(dir, 'oracle.md');
  fs.writeFileSync(oracleMd, 'BASE_PROMPT_MARKER');

  const output = runBuildOraclePrompt(path.join(dir, 'state.json'), oracleMd);
  t.false(output.includes('STRATEGY NOTE'), 'should NOT contain STRATEGY NOTE for adaptive');
});

test('build_oracle_prompt: includes steering directive when provided', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, { strategy: 'adaptive' });
  const oracleMd = path.join(dir, 'oracle.md');
  fs.writeFileSync(oracleMd, 'BASE_PROMPT_MARKER');

  const steeringText = 'FOCUS on security and performance';
  const output = runBuildOraclePrompt(path.join(dir, 'state.json'), oracleMd, steeringText);

  // The directive should appear in the output between the phase directive and oracle.md content
  t.true(output.includes(steeringText), 'should include steering directive text');
  t.true(output.includes('BASE_PROMPT_MARKER'), 'should include oracle.md base prompt');

  // Steering should appear before the base prompt
  const steeringIdx = output.indexOf(steeringText);
  const baseIdx = output.indexOf('BASE_PROMPT_MARKER');
  t.true(steeringIdx < baseIdx, 'steering directive should appear before base prompt');
});


// ==== validate-oracle-state tests ====

test('validate-oracle-state: passes with strategy and focus_areas', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, { strategy: 'depth-first', focus_areas: ['security'] });

  const result = runValidateOracleState(dir);
  t.true(result.result.pass, 'state.json with strategy and focus_areas should pass validation');
});

test('validate-oracle-state: passes without strategy and focus_areas (backward compat)', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  // Write state without strategy and focus_areas fields
  const state = {
    version: '1.1',
    topic: 'Test',
    scope: 'both',
    phase: 'survey',
    iteration: 0,
    max_iterations: 15,
    target_confidence: 95,
    overall_confidence: 0,
    started_at: '2026-03-13T00:00:00Z',
    last_updated: '2026-03-13T00:00:00Z',
    status: 'active'
  };
  fs.writeFileSync(path.join(dir, 'state.json'), JSON.stringify(state, null, 2));

  const result = runValidateOracleState(dir);
  t.true(result.result.pass, 'state.json without strategy/focus_areas should pass (backward compat)');
});

test('validate-oracle-state: rejects invalid strategy', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, { strategy: 'random' });

  const result = runValidateOracleState(dir);
  t.false(result.result.pass, 'state.json with invalid strategy should fail validation');
  // Check that at least one check contains "fail" related to strategy
  const failChecks = result.result.checks.filter(c => typeof c === 'string' && c.includes('fail') && c.includes('strategy'));
  t.true(failChecks.length > 0, 'should have a failing check mentioning strategy');
});
