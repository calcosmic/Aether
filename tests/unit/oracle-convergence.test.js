const test = require('ava');
const { execSync } = require('child_process');
const path = require('path');
const fs = require('fs');
const os = require('os');

const ORACLE_SH = path.join(__dirname, '../../.aether/utils/oracle/oracle.sh');
const ORACLE_MD = path.join(__dirname, '../../.aether/utils/oracle/oracle.md');
const PROJECT_ROOT = path.join(__dirname, '../..');

/**
 * Create a temp directory for test fixtures.
 * @returns {string}
 */
function createTmpDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), 'aether-oracle-conv-'));
}

/**
 * Write a plan.json with specified question configurations.
 * Extends the Phase 7 pattern with key_findings arrays for novelty testing.
 * @param {string} dir - Target directory
 * @param {Array} questions - Array of {confidence, touched, status, findings} objects
 */
function writePlan(dir, questions) {
  const plan = {
    version: '1.0',
    questions: questions.map((q, i) => ({
      id: `q${i + 1}`,
      text: `Question ${i + 1}?`,
      status: q.status || 'open',
      confidence: q.confidence,
      key_findings: q.findings || (q.confidence > 0 ? ['some finding'] : []),
      iterations_touched: q.touched || []
    })),
    created_at: '2026-03-13T00:00:00Z',
    last_updated: '2026-03-13T00:00:00Z'
  };
  fs.writeFileSync(path.join(dir, 'plan.json'), JSON.stringify(plan, null, 2));
}

/**
 * Write a state.json with specified phase and optional overrides.
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

/**
 * Write a state.json with a convergence object included.
 * @param {string} dir - Target directory
 * @param {string} phase - Current phase
 * @param {object} convergence - Convergence object (history, prev_findings_count, composite_score, etc.)
 * @param {object} overrides - Additional state-level overrides
 */
function writeStateWithConvergence(dir, phase, convergence, overrides = {}) {
  writeState(dir, phase, { convergence, ...overrides });
}

/**
 * Run compute_convergence by extracting the function from oracle.sh.
 * @param {string} planFile - Path to plan.json
 * @param {string} stateFile - Path to state.json
 * @returns {object} - Parsed JSON with gap_resolution_pct, coverage_pct, novelty_delta, total_findings
 */
function runComputeConvergence(planFile, stateFile) {
  const cmd = `bash -c 'set +e
eval "$(sed -n "/^compute_convergence()/,/^}/p" "${ORACLE_SH}")"
compute_convergence "${planFile}" "${stateFile}"
'`;
  const output = execSync(cmd, { encoding: 'utf8', cwd: PROJECT_ROOT }).trim();
  return JSON.parse(output);
}

/**
 * Run detect_diminishing_returns by extracting the function from oracle.sh.
 * @param {string} stateFile - Path to state.json
 * @returns {string} - "strategy_change", "synthesize_now", or "continue"
 */
function runDetectDiminishingReturns(stateFile) {
  const cmd = `bash -c 'set +e
eval "$(sed -n "/^detect_diminishing_returns()/,/^}/p" "${ORACLE_SH}")"
detect_diminishing_returns "${stateFile}"
'`;
  return execSync(cmd, { encoding: 'utf8', cwd: PROJECT_ROOT }).trim();
}

/**
 * Run check_convergence by extracting the function from oracle.sh.
 * @param {string} stateFile - Path to state.json
 * @returns {number} - Exit code (0 = converged, 1 = not converged)
 */
function runCheckConvergence(stateFile) {
  const cmd = `bash -c 'set +e
eval "$(sed -n "/^check_convergence()/,/^}/p" "${ORACLE_SH}")"
check_convergence "${stateFile}"
echo $?
'`;
  const output = execSync(cmd, { encoding: 'utf8', cwd: PROJECT_ROOT }).trim();
  return parseInt(output.split('\n').pop(), 10);
}

/**
 * Run update_convergence_metrics by extracting both it and compute_convergence.
 * Returns the updated state.json content.
 * @param {string} stateFile - Path to state.json
 * @param {string} planFile - Path to plan.json
 * @returns {object} - Parsed state.json after update
 */
function runUpdateConvergenceMetrics(stateFile, planFile) {
  const cmd = `bash -c 'set +e
eval "$(sed -n "/^compute_convergence()/,/^}/p" "${ORACLE_SH}")"
eval "$(sed -n "/^update_convergence_metrics()/,/^}/p" "${ORACLE_SH}")"
update_convergence_metrics "${stateFile}" "${planFile}"
'`;
  execSync(cmd, { encoding: 'utf8', cwd: PROJECT_ROOT });
  return JSON.parse(fs.readFileSync(stateFile, 'utf8'));
}

/**
 * Run build_synthesis_prompt by extracting the function from oracle.sh.
 * @param {string} reason - The synthesis reason string
 * @returns {string} - The prompt output
 */
function runBuildSynthesisPrompt(reason) {
  // build_synthesis_prompt references $SCRIPT_DIR/oracle.md via cat
  // We need SCRIPT_DIR set so the cat works
  const oracleDir = path.dirname(ORACLE_SH);
  const cmd = `bash -c 'set +e
SCRIPT_DIR="${oracleDir}"
eval "$(sed -n "/^build_synthesis_prompt()/,/^}/p" "${ORACLE_SH}")"
build_synthesis_prompt "${reason}"
'`;
  return execSync(cmd, { encoding: 'utf8', cwd: PROJECT_ROOT }).trim();
}


// ==== compute_convergence tests ====

test('compute_convergence: all questions answered returns 100% gap resolution', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writePlan(dir, [
    { confidence: 90, touched: [1, 2], status: 'answered', findings: ['f1', 'f2'] },
    { confidence: 85, touched: [1, 2], status: 'answered', findings: ['f3'] },
    { confidence: 95, touched: [1, 2, 3], status: 'answered', findings: ['f4', 'f5'] }
  ]);
  writeState(dir, 'verify');

  const result = runComputeConvergence(
    path.join(dir, 'plan.json'),
    path.join(dir, 'state.json')
  );
  t.is(result.gap_resolution_pct, 100, 'All answered questions should give 100% gap resolution');
});

test('compute_convergence: no questions answered returns 0% gap resolution', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writePlan(dir, [
    { confidence: 10, touched: [1], status: 'open', findings: [] },
    { confidence: 15, touched: [], status: 'open', findings: [] },
    { confidence: 20, touched: [1], status: 'open', findings: [] }
  ]);
  writeState(dir, 'survey');

  const result = runComputeConvergence(
    path.join(dir, 'plan.json'),
    path.join(dir, 'state.json')
  );
  t.is(result.gap_resolution_pct, 0, 'No answered or high-partial questions should give 0% gap resolution');
});

test('compute_convergence: partial with 70%+ confidence counts as resolved', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  // 2 partial at 75% (count as resolved) + 1 open at 20% (not resolved) = 2/3 = 66%
  writePlan(dir, [
    { confidence: 75, touched: [1, 2], status: 'partial', findings: ['f1'] },
    { confidence: 75, touched: [1, 2], status: 'partial', findings: ['f2'] },
    { confidence: 20, touched: [1], status: 'open', findings: [] }
  ]);
  writeState(dir, 'investigate');

  const result = runComputeConvergence(
    path.join(dir, 'plan.json'),
    path.join(dir, 'state.json')
  );
  t.is(result.gap_resolution_pct, 66, 'Partial questions with 70%+ confidence should count as resolved (2/3 = 66%)');
});

test('compute_convergence: coverage counts questions with any iterations_touched', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  // 2 touched, 1 not touched = 66% coverage
  writePlan(dir, [
    { confidence: 40, touched: [1], status: 'open', findings: ['f1'] },
    { confidence: 30, touched: [1, 2], status: 'open', findings: ['f2'] },
    { confidence: 0, touched: [], status: 'open', findings: [] }
  ]);
  writeState(dir, 'survey');

  const result = runComputeConvergence(
    path.join(dir, 'plan.json'),
    path.join(dir, 'state.json')
  );
  t.is(result.coverage_pct, 66, 'Coverage should be 66% when 2 of 3 questions have been touched');
});

test('compute_convergence: zero questions returns 100 for all metrics', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  // Empty questions array -- edge case
  const plan = {
    version: '1.0',
    questions: [],
    created_at: '2026-03-13T00:00:00Z',
    last_updated: '2026-03-13T00:00:00Z'
  };
  fs.writeFileSync(path.join(dir, 'plan.json'), JSON.stringify(plan, null, 2));
  writeState(dir, 'survey');

  const result = runComputeConvergence(
    path.join(dir, 'plan.json'),
    path.join(dir, 'state.json')
  );
  t.is(result.gap_resolution_pct, 100, 'Zero questions should default to 100% gap resolution');
  t.is(result.coverage_pct, 100, 'Zero questions should default to 100% coverage');
});

test('compute_convergence: novelty delta computed from prev_findings_count', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  // State has prev_findings_count=3, plan has 5 total findings across questions
  writePlan(dir, [
    { confidence: 50, touched: [1], status: 'partial', findings: ['f1', 'f2', 'f3'] },
    { confidence: 40, touched: [1], status: 'open', findings: ['f4', 'f5'] }
  ]);
  writeStateWithConvergence(dir, 'investigate', {
    prev_findings_count: 3,
    history: []
  });

  const result = runComputeConvergence(
    path.join(dir, 'plan.json'),
    path.join(dir, 'state.json')
  );
  t.is(result.novelty_delta, 2, 'Novelty delta should be current findings (5) minus prev (3) = 2');
  t.is(result.total_findings, 5, 'Total findings should count all key_findings across questions');
});


// ==== detect_diminishing_returns tests ====

test('detect_diminishing_returns: returns continue when history has fewer than 3 entries', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeStateWithConvergence(dir, 'survey', {
    history: [
      { iteration: 1, novelty_delta: 0, confidence_delta: 0, gap_resolution_pct: 30, coverage_pct: 50, phase: 'survey' },
      { iteration: 2, novelty_delta: 0, confidence_delta: 0, gap_resolution_pct: 30, coverage_pct: 50, phase: 'survey' }
    ],
    prev_findings_count: 0
  });

  const result = runDetectDiminishingReturns(path.join(dir, 'state.json'));
  t.is(result, 'continue', 'Should return continue when fewer than 3 history entries');
});

test('detect_diminishing_returns: returns strategy_change when 3 low-change iterations in survey phase', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeStateWithConvergence(dir, 'survey', {
    history: [
      { iteration: 1, novelty_delta: 0, confidence_delta: 0, gap_resolution_pct: 30, coverage_pct: 50, phase: 'survey' },
      { iteration: 2, novelty_delta: 0, confidence_delta: 0, gap_resolution_pct: 30, coverage_pct: 50, phase: 'survey' },
      { iteration: 3, novelty_delta: 1, confidence_delta: 0, gap_resolution_pct: 30, coverage_pct: 50, phase: 'survey' }
    ],
    prev_findings_count: 0
  });

  const result = runDetectDiminishingReturns(path.join(dir, 'state.json'));
  t.is(result, 'strategy_change', 'Should return strategy_change when 3 consecutive low-novelty in survey');
});

test('detect_diminishing_returns: returns synthesize_now when 3 low-change iterations in synthesize phase', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeStateWithConvergence(dir, 'synthesize', {
    history: [
      { iteration: 1, novelty_delta: 0, confidence_delta: 0, gap_resolution_pct: 80, coverage_pct: 100, phase: 'synthesize' },
      { iteration: 2, novelty_delta: 1, confidence_delta: 0, gap_resolution_pct: 80, coverage_pct: 100, phase: 'synthesize' },
      { iteration: 3, novelty_delta: 0, confidence_delta: 0, gap_resolution_pct: 80, coverage_pct: 100, phase: 'synthesize' }
    ],
    prev_findings_count: 10
  });

  const result = runDetectDiminishingReturns(path.join(dir, 'state.json'));
  t.is(result, 'synthesize_now', 'Should return synthesize_now when 3 consecutive low-novelty in synthesize');
});

test('detect_diminishing_returns: returns continue when iterations have high novelty', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeStateWithConvergence(dir, 'survey', {
    history: [
      { iteration: 1, novelty_delta: 5, confidence_delta: 10, gap_resolution_pct: 30, coverage_pct: 50, phase: 'survey' },
      { iteration: 2, novelty_delta: 5, confidence_delta: 8, gap_resolution_pct: 40, coverage_pct: 60, phase: 'survey' },
      { iteration: 3, novelty_delta: 5, confidence_delta: 5, gap_resolution_pct: 50, coverage_pct: 70, phase: 'survey' }
    ],
    prev_findings_count: 15
  });

  const result = runDetectDiminishingReturns(path.join(dir, 'state.json'));
  t.is(result, 'continue', 'Should return continue when iterations have high novelty (delta=5)');
});

test('detect_diminishing_returns: investigate phase uses threshold 0 (novelty_delta=1 is progress)', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  // In investigate phase, threshold is 0, so novelty_delta=1 counts as progress
  writeStateWithConvergence(dir, 'investigate', {
    history: [
      { iteration: 1, novelty_delta: 1, confidence_delta: 5, gap_resolution_pct: 40, coverage_pct: 80, phase: 'investigate' },
      { iteration: 2, novelty_delta: 1, confidence_delta: 5, gap_resolution_pct: 45, coverage_pct: 80, phase: 'investigate' },
      { iteration: 3, novelty_delta: 1, confidence_delta: 5, gap_resolution_pct: 50, coverage_pct: 80, phase: 'investigate' }
    ],
    prev_findings_count: 5
  });

  const result = runDetectDiminishingReturns(path.join(dir, 'state.json'));
  t.is(result, 'continue', 'Investigate phase with novelty_delta=1 should continue (threshold is 0)');
});


// ==== check_convergence tests ====

test('check_convergence: returns true when composite >= 85 and 2+ low novelty iterations', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeStateWithConvergence(dir, 'verify', {
    composite_score: 90,
    history: [
      { iteration: 1, novelty_delta: 5, confidence_delta: 10, gap_resolution_pct: 60, coverage_pct: 80, phase: 'investigate' },
      { iteration: 2, novelty_delta: 0, confidence_delta: 2, gap_resolution_pct: 90, coverage_pct: 100, phase: 'verify' },
      { iteration: 3, novelty_delta: 1, confidence_delta: 0, gap_resolution_pct: 90, coverage_pct: 100, phase: 'verify' }
    ],
    prev_findings_count: 10
  });

  const exitCode = runCheckConvergence(path.join(dir, 'state.json'));
  t.is(exitCode, 0, 'Should return 0 (converged) when composite >= 85 and last 2 entries have low novelty');
});

test('check_convergence: returns false when composite < 85', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeStateWithConvergence(dir, 'investigate', {
    composite_score: 70,
    history: [
      { iteration: 1, novelty_delta: 0, confidence_delta: 0, gap_resolution_pct: 50, coverage_pct: 70, phase: 'investigate' },
      { iteration: 2, novelty_delta: 0, confidence_delta: 0, gap_resolution_pct: 50, coverage_pct: 70, phase: 'investigate' }
    ],
    prev_findings_count: 5
  });

  const exitCode = runCheckConvergence(path.join(dir, 'state.json'));
  t.is(exitCode, 1, 'Should return 1 (not converged) when composite score is below 85');
});

test('check_convergence: returns false when history has fewer than 2 entries', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeStateWithConvergence(dir, 'verify', {
    composite_score: 90,
    history: [
      { iteration: 1, novelty_delta: 0, confidence_delta: 0, gap_resolution_pct: 90, coverage_pct: 100, phase: 'verify' }
    ],
    prev_findings_count: 10
  });

  const exitCode = runCheckConvergence(path.join(dir, 'state.json'));
  t.is(exitCode, 1, 'Should return 1 (not converged) when fewer than 2 history entries');
});


// ==== build_synthesis_prompt tests ====

test('build_synthesis_prompt: includes synthesis directive and reason', t => {
  const output = runBuildSynthesisPrompt('converged');
  t.true(output.includes('SYNTHESIS PASS'), 'Synthesis prompt should contain "SYNTHESIS PASS" header');
  t.true(output.includes('converged'), 'Synthesis prompt should contain the reason string');
});

test('build_synthesis_prompt: includes oracle.md content', t => {
  const output = runBuildSynthesisPrompt('max_iterations');
  t.true(output.includes('Oracle Ant'), 'Synthesis prompt should include oracle.md content with "Oracle Ant"');
});


// ==== validate_and_recover tests ====

test('validate_and_recover: returns 0 for valid JSON', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  // Write valid JSON
  fs.writeFileSync(path.join(dir, 'test.json'), '{"valid": true}');

  // We need AETHER_ROOT set for the fallback path in validate_and_recover
  const cmd = `bash -c 'set +e
AETHER_ROOT="${PROJECT_ROOT}/.aether"
eval "$(sed -n "/^validate_and_recover()/,/^}/p" "${ORACLE_SH}")"
validate_and_recover "${path.join(dir, 'test.json')}"
echo $?
'`;
  const output = execSync(cmd, { encoding: 'utf8', cwd: PROJECT_ROOT }).trim();
  const exitCode = parseInt(output.split('\n').pop(), 10);
  t.is(exitCode, 0, 'Valid JSON should return exit code 0');
});

test('validate_and_recover: recovers from pre-iteration backup', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  const testFile = path.join(dir, 'state.json');

  // Write invalid JSON to main file
  fs.writeFileSync(testFile, '{invalid json broken}');
  // Write valid JSON to pre-iteration backup
  fs.writeFileSync(testFile + '.pre-iteration', '{"recovered": true, "version": "1.0"}');

  const cmd = `bash -c 'set +e
AETHER_ROOT="${PROJECT_ROOT}/.aether"
eval "$(sed -n "/^validate_and_recover()/,/^}/p" "${ORACLE_SH}")"
validate_and_recover "${testFile}" 2>/dev/null
echo $?
'`;
  const output = execSync(cmd, { encoding: 'utf8', cwd: PROJECT_ROOT }).trim();
  const exitCode = parseInt(output.split('\n').pop(), 10);
  t.is(exitCode, 0, 'Should return 0 after recovering from pre-iteration backup');

  // Verify the file was actually restored
  const content = JSON.parse(fs.readFileSync(testFile, 'utf8'));
  t.is(content.recovered, true, 'File should contain the recovered backup content');
});


// ==== update_convergence_metrics tests ====

test('update_convergence_metrics: writes convergence history to state.json', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writePlan(dir, [
    { confidence: 80, touched: [1, 2], status: 'partial', findings: ['f1', 'f2'] },
    { confidence: 90, touched: [1, 2, 3], status: 'answered', findings: ['f3', 'f4', 'f5'] }
  ]);
  writeState(dir, 'synthesize', { iteration: 3, overall_confidence: 50 });

  const updated = runUpdateConvergenceMetrics(
    path.join(dir, 'state.json'),
    path.join(dir, 'plan.json')
  );

  t.truthy(updated.convergence, 'State should have convergence object after update');
  t.truthy(updated.convergence.history, 'Convergence should have history array');
  t.true(updated.convergence.history.length >= 1, 'History should have at least 1 entry');
  t.is(typeof updated.convergence.composite_score, 'number', 'Composite score should be a number');
  t.is(typeof updated.convergence.converged, 'boolean', 'Converged flag should be a boolean');
});

test('update_convergence_metrics: updates prev_findings_count to current total', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  // 3 findings total across questions
  writePlan(dir, [
    { confidence: 50, touched: [1], status: 'partial', findings: ['f1', 'f2'] },
    { confidence: 30, touched: [1], status: 'open', findings: ['f3'] }
  ]);
  writeState(dir, 'investigate', { iteration: 2, overall_confidence: 30 });

  const updated = runUpdateConvergenceMetrics(
    path.join(dir, 'state.json'),
    path.join(dir, 'plan.json')
  );

  t.is(updated.convergence.prev_findings_count, 3, 'prev_findings_count should be updated to current total findings (3)');
});
