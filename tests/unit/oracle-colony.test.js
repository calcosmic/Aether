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
  return fs.mkdtempSync(path.join(os.tmpdir(), 'aether-oracle-colony-'));
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
    phase: 'verify',
    iteration: 5,
    max_iterations: 15,
    target_confidence: 95,
    overall_confidence: 85,
    started_at: '2026-03-13T00:00:00Z',
    last_updated: '2026-03-13T00:00:00Z',
    status: 'complete',
    strategy: 'adaptive',
    focus_areas: [],
    template: 'custom'
  };
  const state = Object.assign({}, defaults, overrides);
  fs.writeFileSync(path.join(dir, 'state.json'), JSON.stringify(state, null, 2));
}

/**
 * Write plan.json with structured findings (v1.1 format).
 * @param {string} dir - Target directory
 * @param {Array} questions - Array of {confidence, status, findings} objects
 * @param {object} sources - Sources registry
 */
function writePlanWithFindings(dir, questions, sources = {}) {
  const plan = {
    version: '1.1',
    sources,
    questions: questions.map((q, i) => ({
      id: `q${i + 1}`,
      text: q.text || `Question ${i + 1}?`,
      status: q.status || 'open',
      confidence: q.confidence || 0,
      key_findings: q.findings || [],
      iterations_touched: q.touched || [1]
    })),
    created_at: '2026-03-13T00:00:00Z',
    last_updated: '2026-03-13T00:00:00Z'
  };
  fs.writeFileSync(path.join(dir, 'plan.json'), JSON.stringify(plan, null, 2));
}

/**
 * Create .aether/data/COLONY_STATE.json with minimal valid state.
 * @param {string} dir - Root directory (will create .aether/data/ subdir)
 */
function writeColonyState(dir) {
  const dataDir = path.join(dir, '.aether', 'data');
  fs.mkdirSync(dataDir, { recursive: true });
  const state = {
    goal: 'test colony',
    state: 'active',
    current_phase: 1,
    plan: { id: 'test-plan', tasks: [] },
    memory: {},
    errors: { records: [] },
    events: [],
    session_id: 'test-session',
    initialized_at: '2026-03-13T00:00:00Z'
  };
  fs.writeFileSync(path.join(dataDir, 'COLONY_STATE.json'), JSON.stringify(state, null, 2));
}

/**
 * Create a mock aether-utils.sh that logs calls instead of executing real subcommands.
 * @param {string} dir - Root directory (will create .aether/ subdir)
 */
function mockUtils(dir) {
  const aetherDir = path.join(dir, '.aether');
  fs.mkdirSync(aetherDir, { recursive: true });
  const mockScript = `#!/bin/bash
# Mock aether-utils.sh that logs calls
echo "$@" >> "${dir}/promotion-log.txt"
echo '{"ok":true}'
`;
  fs.writeFileSync(path.join(aetherDir, 'aether-utils.sh'), mockScript, { mode: 0o755 });
}

/**
 * Run promote_to_colony by extracting it from oracle.sh.
 * @param {string} planFile - Path to plan.json
 * @param {string} stateFile - Path to state.json
 * @param {string} aetherRoot - Path containing .aether/ with mock utils
 * @returns {{ stdout: string, exitCode: number }}
 */
function runPromoteToColony(planFile, stateFile, aetherRoot) {
  const cmd = `bash -c 'set +e
eval "$(sed -n "/^promote_to_colony()/,/^}/p" "${ORACLE_SH}")"
promote_to_colony "${planFile}" "${stateFile}" "${aetherRoot}"
echo "EXIT_CODE:$?"
'`;
  try {
    const output = execSync(cmd, { encoding: 'utf8', cwd: PROJECT_ROOT, timeout: 10000 }).trimEnd();
    const lines = output.split('\n');
    const exitLine = lines.pop();
    const exitCode = parseInt(exitLine.replace('EXIT_CODE:', ''), 10);
    return { stdout: lines.join('\n'), exitCode };
  } catch (e) {
    const output = (e.stdout || '').trimEnd();
    const lines = output.split('\n');
    const exitLine = lines.find(l => l.startsWith('EXIT_CODE:'));
    const exitCode = exitLine ? parseInt(exitLine.replace('EXIT_CODE:', ''), 10) : 1;
    return { stdout: lines.filter(l => !l.startsWith('EXIT_CODE:')).join('\n'), exitCode };
  }
}

/**
 * Run build_synthesis_prompt by extracting it from oracle.sh.
 * Sets STATE_FILE and SCRIPT_DIR for the function's internal reads.
 * @param {string} stateFile - Path to state.json
 * @param {string} reason - Synthesis reason (e.g., "converged")
 * @returns {string} - The prompt output
 */
function runBuildSynthesisPrompt(stateFile, reason = 'converged') {
  const scriptDir = path.join(PROJECT_ROOT, '.aether/utils/oracle');
  const cmd = `bash -c 'set +e
STATE_FILE="${stateFile}"
SCRIPT_DIR="${scriptDir}"
eval "$(sed -n "/^build_synthesis_prompt()/,/^}/p" "${ORACLE_SH}")"
build_synthesis_prompt "${reason}"
'`;
  return execSync(cmd, { encoding: 'utf8', cwd: PROJECT_ROOT, timeout: 10000 }).trimEnd();
}

/**
 * Run validate-oracle-state for state.json validation.
 * @param {string} dir - Directory containing state.json
 * @returns {object} - Parsed JSON result
 */
function runValidateOracleState(dir) {
  const utilsPath = path.join(PROJECT_ROOT, '.aether/aether-utils.sh');
  const cmd = `ORACLE_DIR="${dir}" bash "${utilsPath}" validate-oracle-state state`;
  const output = execSync(cmd, { encoding: 'utf8', cwd: PROJECT_ROOT, timeout: 10000 }).trim();
  return JSON.parse(output);
}


// ==== promote_to_colony tests (COLN-01) ====

test('promote_to_colony: extracts only 80%+ confidence answered findings', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, { status: 'complete' });
  writePlanWithFindings(dir, [
    { confidence: 90, status: 'answered', findings: [{ text: 'High confidence finding', source_ids: ['s1'], iteration: 3 }] },
    { confidence: 70, status: 'partial', findings: [{ text: 'Medium finding', source_ids: ['s2'], iteration: 2 }] },
    { confidence: 50, status: 'open', findings: [{ text: 'Low finding', source_ids: [], iteration: 1 }] }
  ]);
  writeColonyState(dir);
  mockUtils(dir);

  const result = runPromoteToColony(
    path.join(dir, 'plan.json'),
    path.join(dir, 'state.json'),
    dir
  );

  t.true(result.stdout.includes('Promoting 1 high-confidence findings'), 'should promote only the 90% finding');
  t.is(result.exitCode, 0, 'should succeed');

  // Check the mock log for API calls
  const log = fs.readFileSync(path.join(dir, 'promotion-log.txt'), 'utf8');
  t.true(log.includes('instinct-create'), 'should call instinct-create');
  t.true(log.includes('learning-promote'), 'should call learning-promote');
  t.true(log.includes('memory-capture'), 'should call memory-capture');
});

test('promote_to_colony: rejects promotion when status is active', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, { status: 'active' });
  writePlanWithFindings(dir, [
    { confidence: 90, status: 'answered', findings: [{ text: 'Finding', source_ids: ['s1'], iteration: 1 }] }
  ]);
  writeColonyState(dir);
  mockUtils(dir);

  const result = runPromoteToColony(
    path.join(dir, 'plan.json'),
    path.join(dir, 'state.json'),
    dir
  );

  t.true(result.stdout.includes('ERROR'), 'should output error message');
  t.true(result.stdout.includes('still active'), 'should mention research is still active');
  t.is(result.exitCode, 1, 'should return error exit code');
});

test('promote_to_colony: allows promotion when status is complete', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, { status: 'complete' });
  writePlanWithFindings(dir, [
    { confidence: 85, status: 'answered', findings: [{ text: 'Complete finding', source_ids: ['s1'], iteration: 2 }] }
  ]);
  writeColonyState(dir);
  mockUtils(dir);

  const result = runPromoteToColony(
    path.join(dir, 'plan.json'),
    path.join(dir, 'state.json'),
    dir
  );

  t.is(result.exitCode, 0, 'should succeed with complete status');
  t.true(result.stdout.includes('Promoting 1'), 'should promote findings');
});

test('promote_to_colony: allows promotion when status is stopped', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, { status: 'stopped' });
  writePlanWithFindings(dir, [
    { confidence: 95, status: 'answered', findings: [{ text: 'Stopped finding', source_ids: ['s1', 's2'], iteration: 4 }] }
  ]);
  writeColonyState(dir);
  mockUtils(dir);

  const result = runPromoteToColony(
    path.join(dir, 'plan.json'),
    path.join(dir, 'state.json'),
    dir
  );

  t.is(result.exitCode, 0, 'should succeed with stopped status');
  t.true(result.stdout.includes('Promoting 1'), 'should promote findings');
});

test('promote_to_colony: returns 0 with message when no findings meet threshold', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, { status: 'complete' });
  writePlanWithFindings(dir, [
    { confidence: 60, status: 'partial', findings: [{ text: 'Below threshold', source_ids: ['s1'], iteration: 1 }] },
    { confidence: 40, status: 'open', findings: [{ text: 'Way below', source_ids: [], iteration: 1 }] }
  ]);
  writeColonyState(dir);
  mockUtils(dir);

  const result = runPromoteToColony(
    path.join(dir, 'plan.json'),
    path.join(dir, 'state.json'),
    dir
  );

  t.is(result.exitCode, 0, 'should exit 0 even when no findings qualify');
  t.true(result.stdout.includes('No findings meet promotion threshold'), 'should explain no findings qualify');
});

test('promote_to_colony: handles v1.0 string findings gracefully', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, { status: 'complete' });
  // v1.0 format: findings are plain strings, not objects
  const plan = {
    version: '1.0',
    sources: {},
    questions: [
      {
        id: 'q1',
        text: 'What is X?',
        status: 'answered',
        confidence: 90,
        key_findings: ['Plain string finding 1', 'Plain string finding 2'],
        iterations_touched: [1, 2]
      }
    ],
    created_at: '2026-03-13T00:00:00Z',
    last_updated: '2026-03-13T00:00:00Z'
  };
  fs.writeFileSync(path.join(dir, 'plan.json'), JSON.stringify(plan, null, 2));
  writeColonyState(dir);
  mockUtils(dir);

  const result = runPromoteToColony(
    path.join(dir, 'plan.json'),
    path.join(dir, 'state.json'),
    dir
  );

  t.is(result.exitCode, 0, 'should handle v1.0 string findings without error');
  t.true(result.stdout.includes('Promoting 1'), 'should promote the qualifying question');
});

test('promote_to_colony: rejects when COLONY_STATE.json missing', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, { status: 'complete' });
  writePlanWithFindings(dir, [
    { confidence: 90, status: 'answered', findings: [{ text: 'Finding', source_ids: ['s1'], iteration: 1 }] }
  ]);
  // Don't create colony state or .aether/data dir
  mockUtils(dir);

  const result = runPromoteToColony(
    path.join(dir, 'plan.json'),
    path.join(dir, 'state.json'),
    dir
  );

  t.true(result.stdout.includes('ERROR'), 'should output error');
  t.true(result.stdout.includes('No active colony'), 'should mention no active colony');
  t.is(result.exitCode, 1, 'should return error exit code');
});


// ==== build_synthesis_prompt template tests (COLN-02, OUTP-01, OUTP-03) ====

test('build_synthesis_prompt: tech-eval template produces Comparison Matrix section', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, { template: 'tech-eval' });
  const output = runBuildSynthesisPrompt(path.join(dir, 'state.json'));

  t.true(output.includes('Comparison Matrix'), 'should contain Comparison Matrix section');
  t.true(output.includes('Adoption Assessment'), 'should contain Adoption Assessment section');
});

test('build_synthesis_prompt: architecture-review template produces Component Map section', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, { template: 'architecture-review' });
  const output = runBuildSynthesisPrompt(path.join(dir, 'state.json'));

  t.true(output.includes('Component Map'), 'should contain Component Map section');
  t.true(output.includes('Risk Assessment'), 'should contain Risk Assessment section');
});

test('build_synthesis_prompt: bug-investigation template produces Root Cause Analysis section', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, { template: 'bug-investigation' });
  const output = runBuildSynthesisPrompt(path.join(dir, 'state.json'));

  t.true(output.includes('Root Cause Analysis'), 'should contain Root Cause Analysis section');
  t.true(output.includes('Reproduction Steps'), 'should contain Reproduction Steps section');
});

test('build_synthesis_prompt: best-practices template produces Gap Analysis section', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, { template: 'best-practices' });
  const output = runBuildSynthesisPrompt(path.join(dir, 'state.json'));

  t.true(output.includes('Gap Analysis'), 'should contain Gap Analysis section');
  t.true(output.includes('Best Practice Benchmark'), 'should contain Best Practice Benchmark section');
});

test('build_synthesis_prompt: custom template produces Findings by Question section', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, { template: 'custom' });
  const output = runBuildSynthesisPrompt(path.join(dir, 'state.json'));

  t.true(output.includes('Findings by Question'), 'should contain Findings by Question section');
  t.true(output.includes('Methodology Notes'), 'should contain Methodology Notes section');
});

test('build_synthesis_prompt: all templates include confidence grouping', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  const templates = ['tech-eval', 'architecture-review', 'bug-investigation', 'best-practices', 'custom'];

  for (const template of templates) {
    writeState(dir, { template });
    const output = runBuildSynthesisPrompt(path.join(dir, 'state.json'));
    t.true(output.includes('Confidence Grouping'), `${template} should include Confidence Grouping`);
  }
});

test('build_synthesis_prompt: unknown template falls through to custom', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, { template: 'unknown-value' });
  const output = runBuildSynthesisPrompt(path.join(dir, 'state.json'));

  t.true(output.includes('Findings by Question'), 'unknown template should fall through to custom structure');
});


// ==== validate-oracle-state template tests ====

test('validate-oracle-state: accepts valid template values', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  const validTemplates = ['tech-eval', 'architecture-review', 'bug-investigation', 'best-practices', 'custom'];

  for (const template of validTemplates) {
    writeState(dir, { template });
    const result = runValidateOracleState(dir);
    t.true(result.result.pass, `template "${template}" should pass validation`);
  }
});

test('validate-oracle-state: rejects invalid template value', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, { template: 'invalid-template' });
  const result = runValidateOracleState(dir);

  t.false(result.result.pass, 'invalid template value should fail validation');
  const failChecks = result.result.checks.filter(c => typeof c === 'string' && c.includes('fail') && c.includes('template'));
  t.true(failChecks.length > 0, 'should have a failing check mentioning template');
});

test('validate-oracle-state: accepts state without template field (backward compat)', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  // Write state without template field
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
  t.true(result.result.pass, 'state without template field should pass (backward compat)');
});
