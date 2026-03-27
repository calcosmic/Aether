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
  return fs.mkdtempSync(path.join(os.tmpdir(), 'aether-oracle-trust-'));
}

/**
 * Write a v1.1 plan.json with structured findings and sources registry.
 * @param {string} dir - Target directory
 * @param {Array} questions - Array of {confidence, touched, status, findings} objects
 *   where findings is an array of {text, source_ids, iteration} objects
 * @param {object} sources - Sources object keyed by ID (e.g. {S1: {url, title, date_accessed, type}})
 */
function writePlanWithSources(dir, questions, sources = {}) {
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
 * Write a v1.0 plan.json with string key_findings (legacy format).
 * @param {string} dir - Target directory
 * @param {Array} questions - Array of {confidence, touched, status, findings} objects
 *   where findings is an array of strings
 */
function writePlanLegacy(dir, questions) {
  const plan = {
    version: '1.0',
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
 * Run compute_trust_scores by extracting the function from oracle.sh.
 * Returns the trust_summary object from the updated plan.json.
 * @param {string} planFile - Path to plan.json
 * @returns {object|null} - Parsed trust_summary object, or null if not written
 */
function runComputeTrustScores(planFile) {
  const cmd = `bash -c 'set +e
eval "$(sed -n "/^compute_trust_scores()/,/^}/p" "${ORACLE_SH}")"
compute_trust_scores "${planFile}"
'`;
  execSync(cmd, { encoding: 'utf8', cwd: PROJECT_ROOT });
  const plan = JSON.parse(fs.readFileSync(planFile, 'utf8'));
  return plan.trust_summary || null;
}

/**
 * Run build_synthesis_prompt by extracting the function from oracle.sh.
 * @param {string} reason - The synthesis reason string
 * @returns {string} - The prompt output
 */
function runBuildSynthesisPrompt(reason) {
  const oracleDir = path.dirname(ORACLE_SH);
  const cmd = `bash -c 'set +e
SCRIPT_DIR="${oracleDir}"
eval "$(sed -n "/^build_synthesis_prompt()/,/^}/p" "${ORACLE_SH}")"
build_synthesis_prompt "${reason}"
'`;
  return execSync(cmd, { encoding: 'utf8', cwd: PROJECT_ROOT }).trim();
}


// ==== compute_trust_scores tests ====

test('compute_trust_scores: mixed sources produces correct counts', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  const sources = {
    S1: { url: 'https://example.com/a', title: 'Source A', date_accessed: '2026-03-13', type: 'documentation' },
    S2: { url: 'https://example.com/b', title: 'Source B', date_accessed: '2026-03-13', type: 'blog' }
  };

  writePlanWithSources(dir, [
    {
      confidence: 80, touched: [1], status: 'partial',
      findings: [
        { text: 'Multi-source finding', source_ids: ['S1', 'S2'], iteration: 1 },
        { text: 'Single-source finding', source_ids: ['S1'], iteration: 1 },
        { text: 'No-source finding', source_ids: [], iteration: 1 }
      ]
    }
  ], sources);

  const trust = runComputeTrustScores(path.join(dir, 'plan.json'));
  t.truthy(trust, 'trust_summary should exist');
  t.is(trust.total_findings, 3, 'total_findings should be 3');
  t.is(trust.multi_source, 1, 'multi_source should be 1');
  t.is(trust.single_source, 1, 'single_source should be 1');
  t.is(trust.no_source, 1, 'no_source should be 1');
  t.is(trust.trust_ratio, 33, 'trust_ratio should be 33 (1/3 multi-source)');
});

test('compute_trust_scores: all multi-source returns 100% trust ratio', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writePlanWithSources(dir, [
    {
      confidence: 90, touched: [1, 2], status: 'answered',
      findings: [
        { text: 'Finding 1', source_ids: ['S1', 'S2'], iteration: 1 },
        { text: 'Finding 2', source_ids: ['S1', 'S2', 'S3'], iteration: 2 }
      ]
    }
  ], {
    S1: { url: 'https://a.com', title: 'A', date_accessed: '2026-03-13', type: 'documentation' },
    S2: { url: 'https://b.com', title: 'B', date_accessed: '2026-03-13', type: 'blog' },
    S3: { url: 'https://c.com', title: 'C', date_accessed: '2026-03-13', type: 'codebase' }
  });

  const trust = runComputeTrustScores(path.join(dir, 'plan.json'));
  t.is(trust.trust_ratio, 100, 'trust_ratio should be 100 when all findings have 2+ sources');
  t.is(trust.multi_source, 2, 'multi_source should be 2');
  t.is(trust.single_source, 0, 'single_source should be 0');
  t.is(trust.no_source, 0, 'no_source should be 0');
});

test('compute_trust_scores: all single-source returns 0% trust ratio', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writePlanWithSources(dir, [
    {
      confidence: 60, touched: [1], status: 'partial',
      findings: [
        { text: 'Finding A', source_ids: ['S1'], iteration: 1 },
        { text: 'Finding B', source_ids: ['S2'], iteration: 1 },
        { text: 'Finding C', source_ids: ['S1'], iteration: 1 }
      ]
    }
  ], {
    S1: { url: 'https://a.com', title: 'A', date_accessed: '2026-03-13', type: 'documentation' },
    S2: { url: 'https://b.com', title: 'B', date_accessed: '2026-03-13', type: 'blog' }
  });

  const trust = runComputeTrustScores(path.join(dir, 'plan.json'));
  t.is(trust.trust_ratio, 0, 'trust_ratio should be 0 when all findings have exactly 1 source');
  t.is(trust.single_source, 3, 'single_source should be 3');
  t.is(trust.multi_source, 0, 'multi_source should be 0');
});

test('compute_trust_scores: zero findings returns trust_ratio 0', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  // Questions exist but have no findings
  writePlanWithSources(dir, [
    { confidence: 0, touched: [], status: 'open', findings: [] },
    { confidence: 0, touched: [], status: 'open', findings: [] }
  ], {});

  const trust = runComputeTrustScores(path.join(dir, 'plan.json'));
  // With no findings at all, the jq check for structured findings returns false
  // (no objects to detect), so compute_trust_scores should return early
  // and trust_summary should NOT be written
  t.is(trust, null, 'trust_summary should not be written when there are no findings (format detection returns false)');
});

test('compute_trust_scores: skips computation for legacy string findings', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writePlanLegacy(dir, [
    { confidence: 80, touched: [1, 2], status: 'partial', findings: ['finding 1', 'finding 2'] },
    { confidence: 60, touched: [1], status: 'open', findings: ['finding 3'] }
  ]);

  const trust = runComputeTrustScores(path.join(dir, 'plan.json'));
  t.is(trust, null, 'trust_summary should NOT be written for legacy string findings');
});

test('compute_trust_scores: handles multiple questions with mixed findings', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writePlanWithSources(dir, [
    {
      confidence: 85, touched: [1, 2], status: 'answered',
      findings: [
        { text: 'Q1 multi', source_ids: ['S1', 'S2'], iteration: 1 },
        { text: 'Q1 single', source_ids: ['S1'], iteration: 2 }
      ]
    },
    {
      confidence: 70, touched: [1], status: 'partial',
      findings: [
        { text: 'Q2 multi', source_ids: ['S2', 'S3'], iteration: 1 },
        { text: 'Q2 none', source_ids: [], iteration: 1 }
      ]
    },
    {
      confidence: 50, touched: [1], status: 'open',
      findings: [
        { text: 'Q3 single', source_ids: ['S1'], iteration: 1 }
      ]
    }
  ], {
    S1: { url: 'https://a.com', title: 'A', date_accessed: '2026-03-13', type: 'documentation' },
    S2: { url: 'https://b.com', title: 'B', date_accessed: '2026-03-13', type: 'blog' },
    S3: { url: 'https://c.com', title: 'C', date_accessed: '2026-03-13', type: 'codebase' }
  });

  const trust = runComputeTrustScores(path.join(dir, 'plan.json'));
  // 5 total: 2 multi, 2 single, 1 no-source
  t.is(trust.total_findings, 5, 'total across 3 questions should be 5');
  t.is(trust.multi_source, 2, 'multi_source across questions should be 2');
  t.is(trust.single_source, 2, 'single_source across questions should be 2');
  t.is(trust.no_source, 1, 'no_source across questions should be 1');
  t.is(trust.trust_ratio, 40, 'trust_ratio should be 40 (2/5 multi-source)');
});


// ==== build_synthesis_prompt tests (citation requirements) ====

test('build_synthesis_prompt: includes Sources section requirement', t => {
  const output = runBuildSynthesisPrompt('converged');
  t.true(output.includes('Sources'), 'Synthesis prompt should mention Sources section');
  t.true(output.includes('inline citations') || output.includes('inline citation'),
    'Synthesis prompt should mention inline citations');
  t.true(output.includes('[S1]'), 'Synthesis prompt should reference [S1] citation format');
});

test('build_synthesis_prompt: includes single source flagging instruction', t => {
  const output = runBuildSynthesisPrompt('converged');
  t.true(
    output.includes('single source') || output.includes('single-source'),
    'Synthesis prompt should instruct flagging single-source findings'
  );
});


// ==== validate-oracle-state tests ====

test('validate-oracle-state: passes for v1.1 plan with sources', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writePlanWithSources(dir, [
    {
      confidence: 80, touched: [1], status: 'partial',
      findings: [
        { text: 'Finding 1', source_ids: ['S1'], iteration: 1 }
      ]
    }
  ], {
    S1: { url: 'https://example.com', title: 'Example', date_accessed: '2026-03-13', type: 'documentation' }
  });

  const cmd = `ORACLE_DIR="${dir}" bash "${path.join(PROJECT_ROOT, '.aether/aether-utils.sh')}" validate-oracle-state plan`;
  const output = execSync(cmd, { encoding: 'utf8', cwd: PROJECT_ROOT }).trim();
  const result = JSON.parse(output);
  t.true(result.result.pass, 'v1.1 plan.json with sources and structured findings should pass validation');
});

test('validate-oracle-state: passes for v1.0 plan without sources (backward compat)', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writePlanLegacy(dir, [
    { confidence: 50, touched: [1], status: 'open', findings: ['finding 1', 'finding 2'] },
    { confidence: 70, touched: [1, 2], status: 'partial', findings: ['finding 3'] }
  ]);

  const cmd = `ORACLE_DIR="${dir}" bash "${path.join(PROJECT_ROOT, '.aether/aether-utils.sh')}" validate-oracle-state plan`;
  const output = execSync(cmd, { encoding: 'utf8', cwd: PROJECT_ROOT }).trim();
  const result = JSON.parse(output);
  t.true(result.result.pass, 'v1.0 plan.json with string findings and no sources should pass validation (backward compat)');
});
