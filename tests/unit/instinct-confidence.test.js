/**
 * Instinct Confidence Calibration Tests (LRN-01)
 *
 * Verifies that learning-promote-auto computes instinct confidence
 * using the recurrence-calibrated formula:
 *   min(0.7 + (observation_count - 1) * 0.05, 0.9)
 *
 * Tests:
 * 1. observation_count=1 -> confidence 0.70
 * 2. observation_count=3 -> confidence 0.80
 * 3. observation_count=5 -> confidence 0.90
 * 4. observation_count=10 -> confidence 0.90 (cap)
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const crypto = require('crypto');
const { execSync } = require('child_process');

// Helper to create temp directory
async function createTempDir() {
  const tmpDir = await fs.promises.mkdtemp(path.join(os.tmpdir(), 'aether-confidence-'));
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
// Returns only the last JSON line (subcommands like learning-promote-auto
// may produce intermediate JSON from nested calls like instinct-create)
function runAetherUtil(tmpDir, command, args = []) {
  const scriptPath = path.join(process.cwd(), '.aether', 'aether-utils.sh');
  const env = {
    ...process.env,
    AETHER_ROOT: tmpDir,
    DATA_DIR: path.join(tmpDir, '.aether', 'data')
  };
  const cmd = `bash "${scriptPath}" ${command} ${args.map(a => `"${a}"`).join(' ')} 2>/dev/null`;
  const output = execSync(cmd, { encoding: 'utf8', env, cwd: tmpDir });
  // Return last non-empty line (the final json_ok result)
  const lines = output.trim().split('\n').filter(l => l.trim());
  return lines[lines.length - 1];
}

// Helper to compute content hash matching aether-utils.sh
function computeContentHash(content) {
  const hash = crypto.createHash('sha256').update(content).digest('hex');
  return `sha256:${hash}`;
}

// Helper to setup test colony structure with QUEEN.md
async function setupTestColony(tmpDir) {
  const aetherDir = path.join(tmpDir, '.aether');
  const dataDir = path.join(aetherDir, 'data');

  // Create directories
  await fs.promises.mkdir(dataDir, { recursive: true });

  // Create QUEEN.md from template (METADATA on single line to avoid awk issues)
  const isoDate = new Date().toISOString();
  const queenTemplate = `# QUEEN.md \u{2014} Colony Wisdom

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

  // Create COLONY_STATE.json with empty instincts array
  const colonyState = {
    session_id: 'colony_test',
    goal: 'test',
    state: 'BUILDING',
    current_phase: 1,
    plan: { phases: [] },
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

  // Create pheromones.json
  await fs.promises.writeFile(
    path.join(dataDir, 'pheromones.json'),
    JSON.stringify({ signals: [], version: '1.0.0' }, null, 2)
  );

  return { aetherDir, dataDir };
}

// Helper to create learning-observations.json with a specific observation_count
async function createObservation(dataDir, content, observationCount, wisdomType) {
  const contentHash = computeContentHash(content);
  const observations = {
    observations: [{
      content_hash: contentHash,
      content: content,
      wisdom_type: wisdomType,
      observation_count: observationCount,
      first_observed: new Date().toISOString(),
      last_observed: new Date().toISOString(),
      colonies: ['test-colony']
    }]
  };
  await fs.promises.writeFile(
    path.join(dataDir, 'learning-observations.json'),
    JSON.stringify(observations, null, 2)
  );
}


// =============================================================================
// Test 1: observation_count=1 -> confidence 0.70
// Uses "decree" wisdom_type (auto threshold=0) to allow promotion with count=1
// =============================================================================

test.serial('learning-promote-auto creates instinct with confidence 0.70 for observation_count=1', async (t) => {
  const tmpDir = await createTempDir();

  try {
    const { dataDir } = await setupTestColony(tmpDir);
    const content = 'Always validate inputs before processing';

    // Create observation with count=1
    // Use "decree" type which has auto threshold=0, allowing count=1 to pass
    await createObservation(dataDir, content, 1, 'decree');

    // Run learning-promote-auto
    const result = runAetherUtil(tmpDir, 'learning-promote-auto', [
      'decree', content, 'test-colony', 'learning'
    ]);

    const resultJson = JSON.parse(result);
    t.true(resultJson.ok, 'Should return ok=true');
    t.is(resultJson.result.promoted, true, 'Should be promoted');

    // Read COLONY_STATE.json and find the instinct
    const stateFile = path.join(dataDir, 'COLONY_STATE.json');
    const state = JSON.parse(fs.readFileSync(stateFile, 'utf8'));
    const instinct = state.memory.instincts.find(i => i.action === content);
    t.truthy(instinct, 'Should find instinct with matching action');
    t.is(parseFloat(instinct.confidence), 0.7, 'Confidence should be 0.70 for observation_count=1');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// =============================================================================
// Test 2: observation_count=3 -> confidence 0.80
// =============================================================================

test.serial('learning-promote-auto creates instinct with confidence 0.80 for observation_count=3', async (t) => {
  const tmpDir = await createTempDir();

  try {
    const { dataDir } = await setupTestColony(tmpDir);
    const content = 'Run tests after every code change';

    // Create observation with count=3 (pattern type, auto threshold=2, passes)
    await createObservation(dataDir, content, 3, 'pattern');

    // Run learning-promote-auto
    const result = runAetherUtil(tmpDir, 'learning-promote-auto', [
      'pattern', content, 'test-colony', 'learning'
    ]);

    const resultJson = JSON.parse(result);
    t.true(resultJson.ok, 'Should return ok=true');
    t.is(resultJson.result.promoted, true, 'Should be promoted');

    // Read COLONY_STATE.json and find the instinct
    const stateFile = path.join(dataDir, 'COLONY_STATE.json');
    const state = JSON.parse(fs.readFileSync(stateFile, 'utf8'));
    const instinct = state.memory.instincts.find(i => i.action === content);
    t.truthy(instinct, 'Should find instinct with matching action');
    t.is(parseFloat(instinct.confidence), 0.8, 'Confidence should be 0.80 for observation_count=3');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// =============================================================================
// Test 3: observation_count=5 -> confidence 0.90
// =============================================================================

test.serial('learning-promote-auto creates instinct with confidence 0.90 for observation_count=5', async (t) => {
  const tmpDir = await createTempDir();

  try {
    const { dataDir } = await setupTestColony(tmpDir);
    const content = 'Always check return codes from shell commands';

    // Create observation with count=5 (pattern type, auto threshold=2, passes)
    await createObservation(dataDir, content, 5, 'pattern');

    // Run learning-promote-auto
    const result = runAetherUtil(tmpDir, 'learning-promote-auto', [
      'pattern', content, 'test-colony', 'learning'
    ]);

    const resultJson = JSON.parse(result);
    t.true(resultJson.ok, 'Should return ok=true');
    t.is(resultJson.result.promoted, true, 'Should be promoted');

    // Read COLONY_STATE.json and find the instinct
    const stateFile = path.join(dataDir, 'COLONY_STATE.json');
    const state = JSON.parse(fs.readFileSync(stateFile, 'utf8'));
    const instinct = state.memory.instincts.find(i => i.action === content);
    t.truthy(instinct, 'Should find instinct with matching action');
    t.is(parseFloat(instinct.confidence), 0.9, 'Confidence should be 0.90 for observation_count=5');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// =============================================================================
// Test 4: observation_count=10 -> confidence 0.90 (cap holds)
// =============================================================================

test.serial('learning-promote-auto caps confidence at 0.90 for observation_count=10', async (t) => {
  const tmpDir = await createTempDir();

  try {
    const { dataDir } = await setupTestColony(tmpDir);
    const content = 'Use descriptive variable names in all scripts';

    // Create observation with count=10 (pattern type, auto threshold=2, passes)
    await createObservation(dataDir, content, 10, 'pattern');

    // Run learning-promote-auto
    const result = runAetherUtil(tmpDir, 'learning-promote-auto', [
      'pattern', content, 'test-colony', 'learning'
    ]);

    const resultJson = JSON.parse(result);
    t.true(resultJson.ok, 'Should return ok=true');
    t.is(resultJson.result.promoted, true, 'Should be promoted');

    // Read COLONY_STATE.json and find the instinct
    const stateFile = path.join(dataDir, 'COLONY_STATE.json');
    const state = JSON.parse(fs.readFileSync(stateFile, 'utf8'));
    const instinct = state.memory.instincts.find(i => i.action === content);
    t.truthy(instinct, 'Should find instinct with matching action');
    t.is(parseFloat(instinct.confidence), 0.9, 'Confidence should be 0.90 (capped) for observation_count=10');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});
