/**
 * Instinct Pipeline Integration Tests
 *
 * End-to-end tests for the instinct pipeline:
 * instinct-create -> instinct-read -> pheromone-prime -> colony-prime
 *
 * These tests verify that LEARN-02 and LEARN-03 work together correctly.
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

// Helper to create temp directory
async function createTempDir() {
  const tmpDir = await fs.promises.mkdtemp(path.join(os.tmpdir(), 'aether-instinct-'));
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

// Helper to setup test colony structure with COLONY_STATE.json and pheromones.json
async function setupTestColony(tmpDir, opts = {}) {
  const aetherDir = path.join(tmpDir, '.aether');
  const dataDir = path.join(aetherDir, 'data');

  // Create directories
  await fs.promises.mkdir(dataDir, { recursive: true });

  // Create QUEEN.md from template (METADATA on single line to avoid awk issues)
  const isoDate = new Date().toISOString();
  const queenTemplate = `# QUEEN.md — Colony Wisdom

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

  // Create COLONY_STATE.json
  const instincts = opts.instincts || [];
  const colonyState = {
    session_id: 'colony_test',
    goal: 'test',
    state: 'BUILDING',
    current_phase: 1,
    plan: { phases: [] },
    memory: {
      instincts: instincts,
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

  // Create pheromones.json
  await fs.promises.writeFile(
    path.join(dataDir, 'pheromones.json'),
    JSON.stringify({ signals: [], version: '1.0.0' }, null, 2)
  );

  return { aetherDir, dataDir };
}


test.serial('instinct-create creates a new instinct in COLONY_STATE.json', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // Run instinct-create
    const result = runAetherUtil(tmpDir, 'instinct-create', [
      '--trigger', 'when tests timeout',
      '--action', 'increase timeout to 30s',
      '--confidence', '0.7',
      '--domain', 'testing',
      '--source', 'phase-1',
      '--evidence', '3 timeout failures'
    ]);

    const resultJson = JSON.parse(result);
    t.true(resultJson.ok, 'Should return ok=true');
    t.is(resultJson.result.action, 'created', 'Should report action as created');
    t.is(resultJson.result.confidence, 0.7, 'Should have confidence 0.7');

    // Read COLONY_STATE.json and verify
    const stateFile = path.join(tmpDir, '.aether', 'data', 'COLONY_STATE.json');
    const state = JSON.parse(fs.readFileSync(stateFile, 'utf8'));
    t.is(state.memory.instincts.length, 1, 'Should have 1 instinct');
    t.is(state.memory.instincts[0].trigger, 'when tests timeout');
    t.is(state.memory.instincts[0].action, 'increase timeout to 30s');
    t.is(state.memory.instincts[0].domain, 'testing');
    t.is(state.memory.instincts[0].confidence, 0.7);
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('instinct-create boosts confidence for duplicate trigger+action', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Setup with one existing instinct at confidence 0.7
    await setupTestColony(tmpDir, {
      instincts: [{
        id: 'instinct_existing',
        trigger: 'when builds fail',
        action: 'check dependency versions',
        confidence: 0.7,
        status: 'hypothesis',
        domain: 'architecture',
        source: 'phase-1',
        evidence: ['first observation'],
        tested: false,
        created_at: new Date().toISOString(),
        last_applied: null,
        applications: 0,
        successes: 0,
        failures: 0
      }]
    });

    // Run instinct-create with same trigger+action
    const result = runAetherUtil(tmpDir, 'instinct-create', [
      '--trigger', 'when builds fail',
      '--action', 'check dependency versions',
      '--confidence', '0.7',
      '--domain', 'architecture',
      '--source', 'phase-2',
      '--evidence', 'second observation'
    ]);

    const resultJson = JSON.parse(result);
    t.true(resultJson.ok, 'Should return ok=true');
    t.is(resultJson.result.action, 'updated', 'Should report action as updated');
    // Use approximate comparison for floating point (0.7 + 0.1 = 0.7999... in IEEE 754)
    t.true(Math.abs(resultJson.result.confidence - 0.8) < 0.001,
      'Should boost confidence to ~0.8 (0.7 + 0.1)');

    // Read COLONY_STATE.json and verify no duplication
    const stateFile = path.join(tmpDir, '.aether', 'data', 'COLONY_STATE.json');
    const state = JSON.parse(fs.readFileSync(stateFile, 'utf8'));
    t.is(state.memory.instincts.length, 1, 'Should still have only 1 instinct (not duplicated)');
    t.true(Math.abs(state.memory.instincts[0].confidence - 0.8) < 0.001,
      'Confidence should be ~0.8');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('instinct-read returns empty array with single JSON line when no instincts', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // Run instinct-read with empty instincts
    const result = runAetherUtil(tmpDir, 'instinct-read');

    // Validate single line output (not double JSON -- validates fallthrough bug fix)
    const lines = result.trim().split('\n');
    t.is(lines.length, 1, 'Should output exactly 1 line (not 2 -- validates fallthrough fix)');

    // Parse the single line
    const resultJson = JSON.parse(lines[0]);
    t.true(resultJson.ok, 'Should return ok=true');
    t.deepEqual(resultJson.result.instincts, [], 'Should have empty instincts array');
    t.is(resultJson.result.total, 0, 'Should have total of 0');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('instinct-read filters by min-confidence and domain', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Setup with 3 instincts at different confidence/domain combos
    await setupTestColony(tmpDir, {
      instincts: [
        {
          id: 'instinct_low',
          trigger: 'when linting fails',
          action: 'check eslint config',
          confidence: 0.5,
          status: 'hypothesis',
          domain: 'testing',
          source: 'phase-1',
          evidence: [],
          tested: false,
          created_at: new Date().toISOString(),
          last_applied: null,
          applications: 0,
          successes: 0,
          failures: 0
        },
        {
          id: 'instinct_med',
          trigger: 'when modules circular',
          action: 'refactor to barrel exports',
          confidence: 0.7,
          status: 'hypothesis',
          domain: 'architecture',
          source: 'phase-1',
          evidence: [],
          tested: false,
          created_at: new Date().toISOString(),
          last_applied: null,
          applications: 0,
          successes: 0,
          failures: 0
        },
        {
          id: 'instinct_high',
          trigger: 'when tests timeout',
          action: 'increase timeout to 30s',
          confidence: 0.9,
          status: 'hypothesis',
          domain: 'testing',
          source: 'phase-1',
          evidence: [],
          tested: false,
          created_at: new Date().toISOString(),
          last_applied: null,
          applications: 0,
          successes: 0,
          failures: 0
        }
      ]
    });

    // Test min-confidence filtering
    const confResult = runAetherUtil(tmpDir, 'instinct-read', ['--min-confidence', '0.7']);
    const confJson = JSON.parse(confResult);
    t.is(confJson.result.filtered, 2, 'Should return 2 instincts at >= 0.7 confidence');
    t.true(confJson.result.instincts.every(i => i.confidence >= 0.7), 'All should have confidence >= 0.7');

    // Test domain filtering
    const domainResult = runAetherUtil(tmpDir, 'instinct-read', ['--domain', 'testing']);
    const domainJson = JSON.parse(domainResult);
    t.true(domainJson.result.instincts.every(i => i.domain === 'testing'), 'All should be in testing domain');
    t.is(domainJson.result.filtered, 2, 'Should return 2 testing domain instincts');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('pheromone-prime groups instincts by domain', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Setup with instincts in two domains
    await setupTestColony(tmpDir, {
      instincts: [
        {
          id: 'instinct_test1',
          trigger: 'when tests timeout',
          action: 'increase timeout',
          confidence: 0.8,
          status: 'hypothesis',
          domain: 'testing',
          source: 'phase-1',
          evidence: [],
          tested: false,
          created_at: new Date().toISOString(),
          last_applied: null,
          applications: 0,
          successes: 0,
          failures: 0
        },
        {
          id: 'instinct_arch1',
          trigger: 'when modules circular',
          action: 'use barrel exports',
          confidence: 0.7,
          status: 'hypothesis',
          domain: 'architecture',
          source: 'phase-1',
          evidence: [],
          tested: false,
          created_at: new Date().toISOString(),
          last_applied: null,
          applications: 0,
          successes: 0,
          failures: 0
        }
      ]
    });

    // Run pheromone-prime
    const result = runAetherUtil(tmpDir, 'pheromone-prime');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');
    t.is(resultJson.result.instinct_count, 2, 'Should have 2 instincts');

    // Check domain grouping in prompt_section
    const section = resultJson.result.prompt_section;
    t.true(section.includes('Testing:'), 'Should have Testing: domain header');
    t.true(section.includes('Architecture:'), 'Should have Architecture: domain header');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('colony-prime includes instincts in prompt_section', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Setup with instincts
    await setupTestColony(tmpDir, {
      instincts: [
        {
          id: 'instinct_1',
          trigger: 'when API calls fail',
          action: 'add retry with backoff',
          confidence: 0.8,
          status: 'hypothesis',
          domain: 'resilience',
          source: 'phase-1',
          evidence: [],
          tested: false,
          created_at: new Date().toISOString(),
          last_applied: null,
          applications: 0,
          successes: 0,
          failures: 0
        }
      ]
    });

    // Run colony-prime --compact
    const result = runAetherUtil(tmpDir, 'colony-prime', ['--compact']);
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');
    t.true(resultJson.result.prompt_section.includes('INSTINCTS (Learned Behaviors)'),
      'Should contain INSTINCTS header');
    t.true(resultJson.result.prompt_section.includes('When API calls fail'),
      'Should contain instinct trigger text (display adds When prefix)');
    t.true(resultJson.result.prompt_section.includes('add retry with backoff'),
      'Should contain instinct action text');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('colony-prime omits instincts section when none exist', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Setup with empty instincts
    await setupTestColony(tmpDir);

    // Run colony-prime --compact
    const result = runAetherUtil(tmpDir, 'colony-prime', ['--compact']);
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');
    t.false(resultJson.result.prompt_section.includes('INSTINCTS'),
      'Should NOT contain INSTINCTS when none exist');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('complete pipeline: create -> read -> prime', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Start with clean colony
    await setupTestColony(tmpDir);

    // Create two instincts in different domains
    const create1 = runAetherUtil(tmpDir, 'instinct-create', [
      '--trigger', 'when tests fail intermittently',
      '--action', 'add retry logic to flaky tests',
      '--confidence', '0.8',
      '--domain', 'testing',
      '--source', 'phase-1',
      '--evidence', 'observed 5 flaky test runs'
    ]);
    t.true(JSON.parse(create1).ok, 'First instinct-create should succeed');

    const create2 = runAetherUtil(tmpDir, 'instinct-create', [
      '--trigger', 'when file imports are circular',
      '--action', 'introduce interface layer',
      '--confidence', '0.7',
      '--domain', 'architecture',
      '--source', 'phase-1',
      '--evidence', 'circular dependency detected twice'
    ]);
    t.true(JSON.parse(create2).ok, 'Second instinct-create should succeed');

    // Verify instinct-read sees both
    const readResult = runAetherUtil(tmpDir, 'instinct-read');
    const readJson = JSON.parse(readResult);
    t.is(readJson.result.total, 2, 'instinct-read should see 2 instincts');

    // Verify colony-prime includes both instincts grouped by domain
    const primeResult = runAetherUtil(tmpDir, 'colony-prime', ['--compact']);
    const primeJson = JSON.parse(primeResult);

    t.true(primeJson.ok, 'colony-prime should succeed');
    const section = primeJson.result.prompt_section;
    t.true(section.includes('Testing:'), 'Should have Testing domain group');
    t.true(section.includes('Architecture:'), 'Should have Architecture domain group');
    t.true(section.includes('When tests fail intermittently'), 'Should contain first instinct trigger (display adds When prefix)');
    t.true(section.includes('When file imports are circular'), 'Should contain second instinct trigger (display adds When prefix)');
    t.true(section.includes('add retry logic to flaky tests'), 'Should contain first instinct action');
    t.true(section.includes('introduce interface layer'), 'Should contain second instinct action');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('colony-prime deduplicates "When" prefix in instinct display (no When-When)', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Setup with instincts that already have "when"/"When" prefixes in triggers
    await setupTestColony(tmpDir, {
      instincts: [
        {
          id: 'instinct_lower',
          trigger: 'when API calls fail',
          action: 'add retry with backoff',
          confidence: 0.8,
          status: 'hypothesis',
          domain: 'resilience',
          source: 'phase-1',
          evidence: [],
          tested: false,
          created_at: new Date().toISOString(),
          last_applied: null,
          applications: 0,
          successes: 0,
          failures: 0
        },
        {
          id: 'instinct_upper',
          trigger: 'When deploying to production',
          action: 'run smoke tests first',
          confidence: 0.9,
          status: 'hypothesis',
          domain: 'deployment',
          source: 'phase-2',
          evidence: [],
          tested: false,
          created_at: new Date().toISOString(),
          last_applied: null,
          applications: 0,
          successes: 0,
          failures: 0
        },
        {
          id: 'instinct_noprefix',
          trigger: 'working on database migrations',
          action: 'backup first',
          confidence: 0.7,
          status: 'hypothesis',
          domain: 'database',
          source: 'phase-1',
          evidence: [],
          tested: false,
          created_at: new Date().toISOString(),
          last_applied: null,
          applications: 0,
          successes: 0,
          failures: 0
        }
      ]
    });

    const result = runAetherUtil(tmpDir, 'colony-prime', ['--compact']);
    const resultJson = JSON.parse(result);
    t.true(resultJson.ok, 'Should return ok=true');

    const section = resultJson.result.prompt_section;

    // The display should NOT contain "When when" or "When When" (doubled prefix)
    t.false(section.includes('When when'), 'Should NOT have "When when" (doubled prefix)');
    t.false(section.includes('When When'), 'Should NOT have "When When" (doubled prefix)');

    // Should have exactly one "When " prefix for each instinct
    t.true(section.includes('When API calls fail'), 'lowercase "when" trigger should display as "When API calls fail"');
    t.true(section.includes('When deploying to production'), 'uppercase "When" trigger should display as "When deploying to production"');
    t.true(section.includes('When working on database migrations'), 'no-prefix trigger should display as "When working on database migrations"');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// ============================================================================
// Fuzzy Dedup Tests (27-01)
// ============================================================================

test.serial('fuzzy dedup: similar trigger+action merges into single instinct', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // Create instinct A -- both trigger and action use base forms
    const create1 = runAetherUtil(tmpDir, 'instinct-create', [
      '--trigger', 'when writing tests',
      '--action', 'write tests for all new code',
      '--confidence', '0.7',
      '--domain', 'testing',
      '--source', 'phase-1',
      '--evidence', 'observed testing benefit'
    ]);
    const result1 = JSON.parse(create1);
    t.true(result1.ok, 'First instinct-create should succeed');
    t.is(result1.result.action, 'created', 'First should be created');

    // Create instinct B with similar trigger+action via synonym substitution
    // "when implementing tests" -> "writing testing" (implementing->writing, when->stop, tests->testing)
    // "create tests for all new code" -> "writing testing for all new code" (create->writing, tests->testing)
    // Both normalize to same form => Jaccard 1.00 for both fields
    const create2 = runAetherUtil(tmpDir, 'instinct-create', [
      '--trigger', 'when implementing tests',
      '--action', 'create tests for all new code',
      '--confidence', '0.9',
      '--domain', 'testing',
      '--source', 'phase-2',
      '--evidence', 'confirmed testing value'
    ]);
    const result2 = JSON.parse(create2);
    t.true(result2.ok, 'Second instinct-create should succeed');
    t.is(result2.result.action, 'merged', 'Second should be merged (not created)');
    t.is(result2.result.instinct_id, result1.result.instinct_id, 'Should merge into existing instinct');

    // Verify only 1 instinct exists with averaged confidence
    const stateFile = path.join(tmpDir, '.aether', 'data', 'COLONY_STATE.json');
    const state = JSON.parse(fs.readFileSync(stateFile, 'utf8'));
    t.is(state.memory.instincts.length, 1, 'Should have exactly 1 instinct after merge');

    // Confidence should be averaged: (0.7 + 0.9) / 2 = 0.8
    t.true(Math.abs(state.memory.instincts[0].confidence - 0.8) < 0.01,
      'Confidence should be averaged to ~0.8');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('fuzzy dedup: below 80% threshold creates separate instincts', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // Create instinct A
    const create1 = runAetherUtil(tmpDir, 'instinct-create', [
      '--trigger', 'when writing tests',
      '--action', 'write tests for all new code',
      '--confidence', '0.7',
      '--domain', 'testing',
      '--source', 'phase-1',
      '--evidence', 'observed testing'
    ]);
    t.true(JSON.parse(create1).ok, 'First should succeed');

    // Create instinct B with completely different trigger+action
    const create2 = runAetherUtil(tmpDir, 'instinct-create', [
      '--trigger', 'when deploying to production',
      '--action', 'use blue-green deployment strategy',
      '--confidence', '0.8',
      '--domain', 'deployment',
      '--source', 'phase-2',
      '--evidence', 'deploy issue'
    ]);
    const result2 = JSON.parse(create2);
    t.true(result2.ok, 'Second should succeed');
    t.is(result2.result.action, 'created', 'Should create new instinct (no false merge)');

    // Verify 2 separate instincts
    const stateFile = path.join(tmpDir, '.aether', 'data', 'COLONY_STATE.json');
    const state = JSON.parse(fs.readFileSync(stateFile, 'utf8'));
    t.is(state.memory.instincts.length, 2, 'Should have 2 separate instincts');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('fuzzy dedup: only trigger matches does not merge', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // Create instinct A
    const create1 = runAetherUtil(tmpDir, 'instinct-create', [
      '--trigger', 'when writing tests',
      '--action', 'write tests for all new code',
      '--confidence', '0.7',
      '--domain', 'testing',
      '--source', 'phase-1',
      '--evidence', 'testing'
    ]);
    t.true(JSON.parse(create1).ok, 'First should succeed');

    // Create instinct B: same trigger pattern but completely different action
    const create2 = runAetherUtil(tmpDir, 'instinct-create', [
      '--trigger', 'when implementing tests',
      '--action', 'deploy to staging server immediately',
      '--confidence', '0.8',
      '--domain', 'deployment',
      '--source', 'phase-2',
      '--evidence', 'staging'
    ]);
    const result2 = JSON.parse(create2);
    t.true(result2.ok, 'Second should succeed');
    // Trigger matches (both normalize to "writing testing") but action is completely different
    t.is(result2.result.action, 'created', 'Should NOT merge when only trigger matches');

    // Verify 2 separate instincts
    const stateFile = path.join(tmpDir, '.aether', 'data', 'COLONY_STATE.json');
    const state = JSON.parse(fs.readFileSync(stateFile, 'utf8'));
    t.is(state.memory.instincts.length, 2, 'Should have 2 instincts (action too different to merge)');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('normalize_text: casing and punctuation normalized', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // Create instinct with mixed case and punctuation
    const create1 = runAetherUtil(tmpDir, 'instinct-create', [
      '--trigger', 'When Implementing Tests!',
      '--action', 'Create tests for all new code.',
      '--confidence', '0.7',
      '--domain', 'testing',
      '--source', 'phase-1',
      '--evidence', 'punctuation test'
    ]);
    t.true(JSON.parse(create1).ok, 'First should succeed');

    // Create instinct with same meaning, different casing/punctuation
    const create2 = runAetherUtil(tmpDir, 'instinct-create', [
      '--trigger', 'when writing tests',
      '--action', 'write tests for all new code',
      '--confidence', '0.9',
      '--domain', 'testing',
      '--source', 'phase-2',
      '--evidence', 'case test'
    ]);
    const result2 = JSON.parse(create2);
    t.true(result2.ok, 'Second should succeed');
    t.is(result2.result.action, 'merged', 'Should merge despite casing/punctuation differences');

    // Verify only 1 instinct
    const stateFile = path.join(tmpDir, '.aether', 'data', 'COLONY_STATE.json');
    const state = JSON.parse(fs.readFileSync(stateFile, 'utf8'));
    t.is(state.memory.instincts.length, 1, 'Should have 1 instinct (punctuation/casing normalized)');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('fuzzy dedup: keeps longer text on merge', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // Create instinct A with short trigger and action
    const create1 = runAetherUtil(tmpDir, 'instinct-create', [
      '--trigger', 'when writing tests',
      '--action', 'write tests for new code',
      '--confidence', '0.7',
      '--domain', 'testing',
      '--source', 'phase-1',
      '--evidence', 'short text'
    ]);
    t.true(JSON.parse(create1).ok, 'First should succeed');

    // Create instinct B with longer trigger and action (same meaning via synonyms)
    // Both normalize identically: trigger -> "writing testing", action -> "writing testing for new code"
    // But B uses longer words: "implementing" > "writing", "create" > "write"
    const create2 = runAetherUtil(tmpDir, 'instinct-create', [
      '--trigger', 'when implementing tests',
      '--action', 'create tests for new code',
      '--confidence', '0.9',
      '--domain', 'testing',
      '--source', 'phase-2',
      '--evidence', 'long text'
    ]);
    const result2 = JSON.parse(create2);
    t.true(result2.ok, 'Second should succeed');
    t.is(result2.result.action, 'merged', 'Should merge');

    // Verify merged instinct keeps the longer text
    const stateFile = path.join(tmpDir, '.aether', 'data', 'COLONY_STATE.json');
    const state = JSON.parse(fs.readFileSync(stateFile, 'utf8'));
    t.is(state.memory.instincts.length, 1, 'Should have 1 instinct');
    t.is(state.memory.instincts[0].trigger, 'when implementing tests',
      'Should keep longer trigger (implementing vs writing)');
    t.is(state.memory.instincts[0].action, 'create tests for new code',
      'Should keep longer action (create vs write)');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});
