/**
 * Learning Pipeline End-to-End Integration Tests
 *
 * Validates the complete memory-capture -> learning-observe -> learning-promote-auto
 * -> queen-promote + instinct-create pipeline using realistic, non-synthetic data
 * derived from actual Aether development patterns.
 *
 * Requirements covered:
 * LRNG-01: Pipeline validated end-to-end with real (non-test) data
 * LRNG-03: Integration test covers full pipeline path through colony-prime
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

// Helper to create temp directory
async function createTempDir() {
  const tmpDir = await fs.promises.mkdtemp(path.join(os.tmpdir(), 'aether-e2e-'));
  return tmpDir;
}

// Helper to cleanup temp directory
async function cleanupTempDir(tmpDir) {
  try {
    await fs.promises.rm(tmpDir, { recursive: true, force: true });
  } catch (err) {
    // Ignore cleanup errors
  }
}

// Helper to run aether-utils.sh commands
// Returns raw output string. Some subcommands (e.g. memory-capture, learning-promote-auto)
// emit multiple JSON lines when they call other subcommands internally
// (like instinct-create). Use parseLastJson() to safely parse the final result.
function runAetherUtil(tmpDir, command, args = []) {
  const scriptPath = path.join(process.cwd(), '.aether', 'aether-utils.sh');
  const env = {
    ...process.env,
    AETHER_ROOT: tmpDir,
    DATA_DIR: path.join(tmpDir, '.aether', 'data')
  };
  const cmd = `bash "${scriptPath}" ${command} ${args.map(a => `"${a}"`).join(' ')} 2>/dev/null`;
  return execSync(cmd, { encoding: 'utf8', env, cwd: tmpDir });
}

// Helper to parse the last JSON line from multi-line output.
// Some aether-utils subcommands call other subcommands that also output JSON
// to stdout (e.g. memory-capture calls learning-promote-auto which calls
// instinct-create), producing multiple JSON objects. The authoritative result
// is always the last line.
function parseLastJson(output) {
  const lines = output.trim().split('\n');
  return JSON.parse(lines[lines.length - 1]);
}

// Helper to setup test colony structure with all required files
async function setupTestColony(tmpDir, opts = {}) {
  const aetherDir = path.join(tmpDir, '.aether');
  const dataDir = path.join(aetherDir, 'data');

  // Create directories
  await fs.promises.mkdir(dataDir, { recursive: true });

  // Create QUEEN.md with emoji section headers (required by _extract_wisdom_sections in colony-prime)
  const isoDate = new Date().toISOString();
  const queenTemplate = `# QUEEN.md \u2014 Colony Wisdom

> Last evolved: ${isoDate}
> Colonies contributed: 0
> Wisdom version: 1.0.0

---

## \u{1F4DC} Philosophies

Core beliefs that guide all colony work.

*No philosophies recorded yet*

---

## \u{1F9ED} Patterns

Validated approaches that consistently work.

*No patterns recorded yet*

---

## \u{26A0}\u{FE0F} Redirects

Anti-patterns to avoid.

*No redirects recorded yet*

---

## \u{1F527} Stack Wisdom

Technology-specific insights.

*No stack wisdom recorded yet*

---

## \u{1F3DB}\u{FE0F} Decrees

User-mandated rules.

*No decrees recorded yet*

---

## \u{1F4CA} Evolution Log

| Date | Colony | Change | Details |
|------|--------|--------|---------|

---

<!-- METADATA {"version":"1.0.0","last_evolved":"${isoDate}","colonies_contributed":[],"promotion_thresholds":{"philosophy":1,"pattern":1,"redirect":1,"stack":1,"decree":0},"stats":{"total_philosophies":0,"total_patterns":0,"total_redirects":0,"total_stack_entries":0,"total_decrees":0}} -->`;

  await fs.promises.writeFile(path.join(aetherDir, 'QUEEN.md'), queenTemplate);

  // Create empty learning-observations.json
  await fs.promises.writeFile(
    path.join(dataDir, 'learning-observations.json'),
    JSON.stringify({ observations: [] }, null, 2)
  );

  // Create COLONY_STATE.json with memory.instincts array
  const colonyState = {
    session_id: opts.sessionId || 'colony_e2e_test',
    goal: opts.goal || 'validate learning pipeline end-to-end',
    state: 'BUILDING',
    current_phase: opts.currentPhase !== undefined ? opts.currentPhase : 1,
    plan: { phases: opts.completedPhases || [] },
    memory: {
      instincts: [],
      phase_learnings: [],
      decisions: []
    },
    errors: { flagged_patterns: [] },
    events: []
  };
  await fs.promises.writeFile(
    path.join(dataDir, 'COLONY_STATE.json'),
    JSON.stringify(colonyState, null, 2)
  );

  // Create pheromones.json (required by colony-prime -> pheromone-prime)
  const signals = opts.pheromoneSignals || [];
  await fs.promises.writeFile(
    path.join(dataDir, 'pheromones.json'),
    JSON.stringify({ signals: signals, version: '1.0.0' }, null, 2)
  );

  return { aetherDir, dataDir };
}


// =============================================================================
// Realistic content derived from actual Aether development patterns
// =============================================================================

// Pattern: discovered in Phase 3 (03-01) -- jq boolean handling
const REALISTIC_PATTERN = 'Use explicit jq if/elif chains instead of the // operator when checking fields that can legitimately be false';

// Failure: recurring build issue from data corruption
const REALISTIC_FAILURE = 'Validate JSON structure before atomic_write to prevent corrupted state files';

// Redirect: discovered during Phase 1 data purge
const REALISTIC_REDIRECT = 'Never commit test artifacts to colony state files that ship to users';

// Philosophy: emerged from multi-phase development
const REALISTIC_PHILOSOPHY = 'Clean state data before integrating new features to avoid false positives from stale artifacts';


// =============================================================================
// Test 1: First memory-capture records observation without promoting
// =============================================================================

test.serial('memory-capture records realistic observation on first call without promoting', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // Call memory-capture once with realistic pattern content
    const result = parseLastJson(runAetherUtil(tmpDir, 'memory-capture', [
      'learning', REALISTIC_PATTERN, 'pattern', 'worker:builder'
    ]));

    // Verify: observation recorded, not promoted
    t.true(result.ok, 'Should return ok=true');
    t.false(result.result.auto_promoted, 'First call should NOT auto-promote');
    t.is(result.result.observation_count, 1, 'Should have observation_count=1');

    // Verify learning-observations.json has 1 observation with the content
    const obsFile = path.join(tmpDir, '.aether', 'data', 'learning-observations.json');
    const observations = JSON.parse(fs.readFileSync(obsFile, 'utf8'));
    t.is(observations.observations.length, 1, 'Should have 1 observation');
    t.is(observations.observations[0].content, REALISTIC_PATTERN, 'Observation content should match');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// =============================================================================
// Test 2: memory-capture auto-promotes realistic pattern after second observation
// =============================================================================

test.serial('memory-capture auto-promotes realistic pattern after second observation (threshold=2)', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // First call: records observation, not promoted
    const firstCapture = parseLastJson(runAetherUtil(tmpDir, 'memory-capture', [
      'learning', REALISTIC_PATTERN, 'pattern', 'worker:builder'
    ]));
    t.false(firstCapture.result.auto_promoted, 'First capture should NOT auto-promote');
    t.is(firstCapture.result.observation_count, 1, 'First capture: observation_count=1');

    // Second call: threshold met, auto-promoted
    const secondCapture = parseLastJson(runAetherUtil(tmpDir, 'memory-capture', [
      'learning', REALISTIC_PATTERN, 'pattern', 'worker:builder'
    ]));
    t.true(secondCapture.result.auto_promoted, 'Second capture SHOULD auto-promote');

    // Verify QUEEN.md contains the content in Patterns section
    const queenContent = fs.readFileSync(path.join(tmpDir, '.aether', 'QUEEN.md'), 'utf8');
    t.true(queenContent.includes(REALISTIC_PATTERN), 'QUEEN.md should contain promoted content');
    const patternsSection = queenContent.split('Patterns')[1]?.split('##')[0];
    t.truthy(patternsSection, 'Should have Patterns section');
    t.true(patternsSection.includes(REALISTIC_PATTERN), 'Content should be in Patterns section');

    // Verify COLONY_STATE.json has instinct with matching action, domain, and source
    const state = JSON.parse(fs.readFileSync(
      path.join(tmpDir, '.aether', 'data', 'COLONY_STATE.json'), 'utf8'
    ));
    t.true(state.memory.instincts.length >= 1, 'Should have at least 1 instinct');
    const instinct = state.memory.instincts.find(i => i.action === REALISTIC_PATTERN);
    t.truthy(instinct, 'Instinct action should match promoted content');
    t.is(instinct.domain, 'pattern', 'Instinct domain should be pattern');
    t.is(instinct.source, 'promoted_from_learning', 'Instinct source should be promoted_from_learning');
    t.true(instinct.confidence >= 0.7, 'Instinct confidence should be >= 0.7');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// =============================================================================
// Test 3: memory-capture auto-promotes realistic failure content
// =============================================================================

test.serial('memory-capture auto-promotes realistic failure content after second observation (threshold=2)', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // First call: not promoted
    const firstCapture = parseLastJson(runAetherUtil(tmpDir, 'memory-capture', [
      'failure', REALISTIC_FAILURE, 'failure', 'worker:builder'
    ]));
    t.false(firstCapture.result.auto_promoted, 'First capture should NOT auto-promote');

    // Second call: auto-promoted
    const secondCapture = parseLastJson(runAetherUtil(tmpDir, 'memory-capture', [
      'failure', REALISTIC_FAILURE, 'failure', 'worker:builder'
    ]));
    t.true(secondCapture.result.auto_promoted, 'Second capture SHOULD auto-promote');

    // Verify QUEEN.md has content in Patterns section (failure maps to Patterns)
    const queenContent = fs.readFileSync(path.join(tmpDir, '.aether', 'QUEEN.md'), 'utf8');
    t.true(queenContent.includes(REALISTIC_FAILURE), 'QUEEN.md should contain promoted failure content');
    const patternsSection = queenContent.split('Patterns')[1]?.split('##')[0];
    t.true(patternsSection.includes(REALISTIC_FAILURE), 'Failure content should be in Patterns section');

    // Verify instinct created with domain=failure
    const state = JSON.parse(fs.readFileSync(
      path.join(tmpDir, '.aether', 'data', 'COLONY_STATE.json'), 'utf8'
    ));
    const instinct = state.memory.instincts.find(i => i.action === REALISTIC_FAILURE);
    t.truthy(instinct, 'Should have instinct with failure content');
    t.is(instinct.domain, 'failure', 'Instinct domain should be failure');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// =============================================================================
// Test 4: memory-capture auto-promotes philosophy only after third observation
// =============================================================================

test.serial('memory-capture auto-promotes philosophy only after third observation (threshold=3)', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // First call: not promoted
    const firstCapture = parseLastJson(runAetherUtil(tmpDir, 'memory-capture', [
      'learning', REALISTIC_PHILOSOPHY, 'philosophy', 'worker:builder'
    ]));
    t.false(firstCapture.result.auto_promoted, 'First call should NOT auto-promote');

    // Second call: still not promoted (philosophy threshold=3)
    const secondCapture = parseLastJson(runAetherUtil(tmpDir, 'memory-capture', [
      'learning', REALISTIC_PHILOSOPHY, 'philosophy', 'worker:builder'
    ]));
    t.false(secondCapture.result.auto_promoted, 'Second call should NOT auto-promote (philosophy threshold=3)');

    // Third call: now promoted
    const thirdCapture = parseLastJson(runAetherUtil(tmpDir, 'memory-capture', [
      'learning', REALISTIC_PHILOSOPHY, 'philosophy', 'worker:builder'
    ]));
    t.true(thirdCapture.result.auto_promoted, 'Third call SHOULD auto-promote (philosophy threshold=3 met)');

    // Verify QUEEN.md has content in Philosophies section
    const queenContent = fs.readFileSync(path.join(tmpDir, '.aether', 'QUEEN.md'), 'utf8');
    t.true(queenContent.includes(REALISTIC_PHILOSOPHY), 'QUEEN.md should contain promoted philosophy');
    const philosophiesSection = queenContent.split('Philosophies')[1]?.split('##')[0];
    t.truthy(philosophiesSection, 'Should have Philosophies section');
    t.true(philosophiesSection.includes(REALISTIC_PHILOSOPHY), 'Content should be in Philosophies section');

    // Verify instinct created in COLONY_STATE.json
    const state = JSON.parse(fs.readFileSync(
      path.join(tmpDir, '.aether', 'data', 'COLONY_STATE.json'), 'utf8'
    ));
    const instinct = state.memory.instincts.find(i => i.action === REALISTIC_PHILOSOPHY);
    t.truthy(instinct, 'Should have instinct for promoted philosophy');
    t.is(instinct.domain, 'philosophy', 'Instinct domain should be philosophy');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// =============================================================================
// Test 5: Full pipeline: memory-capture -> instinct in colony-prime prompt_section
// =============================================================================

test.serial('full pipeline: memory-capture -> instinct in colony-prime prompt_section', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // Call memory-capture twice with pattern content to trigger promotion
    parseLastJson(runAetherUtil(tmpDir, 'memory-capture', [
      'learning', REALISTIC_PATTERN, 'pattern', 'worker:builder'
    ]));
    const secondCapture = parseLastJson(runAetherUtil(tmpDir, 'memory-capture', [
      'learning', REALISTIC_PATTERN, 'pattern', 'worker:builder'
    ]));
    t.true(secondCapture.result.auto_promoted, 'Should auto-promote after second call');

    // Call colony-prime and verify instinct appears in prompt_section
    const primeResult = JSON.parse(runAetherUtil(tmpDir, 'colony-prime'));

    t.true(primeResult.ok, 'colony-prime should return ok=true');
    t.truthy(primeResult.result.prompt_section, 'Should have prompt_section');
    t.true(primeResult.result.prompt_section.includes('INSTINCTS'),
      'prompt_section should include INSTINCTS header');
    t.true(primeResult.result.prompt_section.includes(REALISTIC_PATTERN),
      'prompt_section should include the promoted content text');
    t.true(primeResult.result.prompt_section.includes('Pattern:'),
      'prompt_section should include domain grouping (Pattern:)');

    // Also verify QUEEN.md has the content
    const queenContent = fs.readFileSync(path.join(tmpDir, '.aether', 'QUEEN.md'), 'utf8');
    t.true(queenContent.includes(REALISTIC_PATTERN), 'QUEEN.md should contain the content');

    // And COLONY_STATE.json has the instinct
    const state = JSON.parse(fs.readFileSync(
      path.join(tmpDir, '.aether', 'data', 'COLONY_STATE.json'), 'utf8'
    ));
    const instinct = state.memory.instincts.find(i => i.action === REALISTIC_PATTERN);
    t.truthy(instinct, 'COLONY_STATE.json should have the instinct');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// =============================================================================
// Test 6: Idempotent: third memory-capture does not create duplicate instinct
// =============================================================================

test.serial('idempotent: third memory-capture with same content does not create duplicate instinct', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // First call: observation recorded
    parseLastJson(runAetherUtil(tmpDir, 'memory-capture', [
      'learning', REALISTIC_REDIRECT, 'pattern', 'worker:builder'
    ]));

    // Second call: auto-promoted
    const secondCapture = parseLastJson(runAetherUtil(tmpDir, 'memory-capture', [
      'learning', REALISTIC_REDIRECT, 'pattern', 'worker:builder'
    ]));
    t.true(secondCapture.result.auto_promoted, 'Second call should auto-promote');

    // Third call: should NOT create a second instinct
    const thirdCapture = parseLastJson(runAetherUtil(tmpDir, 'memory-capture', [
      'learning', REALISTIC_REDIRECT, 'pattern', 'worker:builder'
    ]));

    // Verify: COLONY_STATE.json has exactly 1 instinct with matching action
    const state = JSON.parse(fs.readFileSync(
      path.join(tmpDir, '.aether', 'data', 'COLONY_STATE.json'), 'utf8'
    ));
    const matchingInstincts = state.memory.instincts.filter(i => i.action === REALISTIC_REDIRECT);
    t.is(matchingInstincts.length, 1, 'Should have exactly 1 instinct (not duplicated)');

    // The instinct confidence should remain at 0.75 (from promotion at observation_count=2).
    // The third call returns already_promoted and does NOT re-invoke instinct-create,
    // so confidence is NOT boosted -- this IS correct idempotent behavior.
    t.true(Math.abs(matchingInstincts[0].confidence - 0.75) < 0.01,
      `Confidence should be 0.75 from original promotion (got ${matchingInstincts[0].confidence})`);
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// =============================================================================
// Test 7: Confidence formula: promoted instinct has recurrence-calibrated confidence
// =============================================================================

test.serial('confidence formula: promoted instinct has recurrence-calibrated confidence', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // Call memory-capture twice with pattern content (threshold=2)
    parseLastJson(runAetherUtil(tmpDir, 'memory-capture', [
      'learning', REALISTIC_FAILURE, 'pattern', 'worker:builder'
    ]));
    const secondCapture = parseLastJson(runAetherUtil(tmpDir, 'memory-capture', [
      'learning', REALISTIC_FAILURE, 'pattern', 'worker:builder'
    ]));
    t.true(secondCapture.result.auto_promoted, 'Should auto-promote after second call');

    // Read the instinct from COLONY_STATE.json
    const state = JSON.parse(fs.readFileSync(
      path.join(tmpDir, '.aether', 'data', 'COLONY_STATE.json'), 'utf8'
    ));
    const instinct = state.memory.instincts.find(i => i.action === REALISTIC_FAILURE);
    t.truthy(instinct, 'Should have instinct');

    // Verify confidence is approximately 0.75
    // Formula: min(0.7 + (observation_count - 1) * 0.05, 0.9) = min(0.7 + 1*0.05, 0.9) = 0.75
    t.true(Math.abs(instinct.confidence - 0.75) < 0.01,
      `Confidence should be ~0.75 (got ${instinct.confidence}), formula: min(0.7 + (2-1)*0.05, 0.9)`);
  } finally {
    await cleanupTempDir(tmpDir);
  }
});
