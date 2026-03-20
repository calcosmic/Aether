/**
 * Pheromone Expire: Eternal Promotion with Effective Strength Tests
 *
 * Tests that pheromone-expire uses decayed effective_strength (not raw .strength)
 * when deciding whether to promote expired signals to eternal memory.
 *
 * The decay formula is: effective_strength = strength * (1 - elapsed_days / decay_days)
 * Decay days by type: FOCUS=30, REDIRECT=60, FEEDBACK=90
 * Promotion threshold: effective_strength > 0.80 (i.e., > 80 when scaled to int)
 *
 * Bug fix: Previously used raw .strength, so REDIRECT at 0.9 always qualified.
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

// Helper to create temp directory
async function createTempDir() {
  const tmpDir = await fs.promises.mkdtemp(path.join(os.tmpdir(), 'aether-expire-eternal-'));
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
    DATA_DIR: path.join(tmpDir, '.aether', 'data'),
    HOME: tmpDir  // Redirect eternal memory to temp dir
  };
  const cmd = `bash "${scriptPath}" ${command} ${args.map(a => `"${a}"`).join(' ')} 2>/dev/null`;
  return execSync(cmd, { encoding: 'utf8', env, cwd: tmpDir, timeout: 15000 });
}

// Helper to setup test colony structure
async function setupTestColony(tmpDir) {
  const aetherDir = path.join(tmpDir, '.aether');
  const dataDir = path.join(aetherDir, 'data');

  await fs.promises.mkdir(dataDir, { recursive: true });

  // Create QUEEN.md
  const isoDate = new Date().toISOString();
  const queenTemplate = `# QUEEN.md --- Colony Wisdom

> Last evolved: ${isoDate}
> Colonies contributed: 0
> Wisdom version: 1.0.0

---

## Philosophies

*No philosophies recorded yet*

---

## Patterns

*No patterns recorded yet*

---

## Redirects

*No redirects recorded yet*

---

## Stack Wisdom

*No stack wisdom recorded yet*

---

## Decrees

*No decrees recorded yet*

---

## Evolution Log

| Date | Colony | Change | Details |
|------|--------|--------|---------|

---

<!-- METADATA {"version":"1.0.0","last_evolved":"${isoDate}","colonies_contributed":[],"promotion_thresholds":{"philosophy":1,"pattern":1,"redirect":1,"stack":1,"decree":0},"stats":{"total_philosophies":0,"total_patterns":0,"total_redirects":0,"total_stack_entries":0,"total_decrees":0}} -->`;

  await fs.promises.writeFile(path.join(aetherDir, 'QUEEN.md'), queenTemplate);

  // Create COLONY_STATE.json
  const colonyState = {
    session_id: 'colony_expire_test',
    goal: 'test eternal promotion decay',
    state: 'BUILDING',
    current_phase: 1,
    plan: { phases: [] },
    memory: { instincts: [], phase_learnings: [], decisions: [] },
    errors: { flagged_patterns: [] },
    events: []
  };
  await fs.promises.writeFile(
    path.join(dataDir, 'COLONY_STATE.json'),
    JSON.stringify(colonyState, null, 2)
  );

  // Create midden directory and file (needed by pheromone-expire)
  const middenDir = path.join(dataDir, 'midden');
  await fs.promises.mkdir(middenDir, { recursive: true });
  await fs.promises.writeFile(
    path.join(middenDir, 'midden.json'),
    JSON.stringify({ signals: [], version: '1.0.0' }, null, 2)
  );

  // Initialize eternal memory directory (at HOME/.aether/eternal/)
  const eternalDir = path.join(tmpDir, '.aether', 'eternal');
  await fs.promises.mkdir(eternalDir, { recursive: true });
  await fs.promises.writeFile(
    path.join(eternalDir, 'memory.json'),
    JSON.stringify({ version: '1.0.0', entries: [], stats: { total_entries: 0, total_promotions: 0 } }, null, 2)
  );

  return { aetherDir, dataDir };
}

// Generate ISO timestamp for N days ago
function daysAgo(days) {
  return new Date(Date.now() - days * 86400000).toISOString();
}

// Helper to create a pheromones.json with specific signals
async function writePheromones(tmpDir, signals) {
  const dataDir = path.join(tmpDir, '.aether', 'data');
  const pherFile = path.join(dataDir, 'pheromones.json');
  await fs.promises.writeFile(
    pherFile,
    JSON.stringify({ signals, version: '1.0.0' }, null, 2)
  );
}

// Helper to read eternal memory high-value signals
async function readEternalSignals(tmpDir) {
  const memFile = path.join(tmpDir, '.aether', 'eternal', 'memory.json');
  try {
    const data = JSON.parse(await fs.promises.readFile(memFile, 'utf8'));
    return data.high_value_signals || [];
  } catch {
    return [];
  }
}


// =============================================================================
// Eternal Promotion with Effective Strength Tests
// =============================================================================

test.serial('1. Heavily-decayed REDIRECT (effective ~0.45) should NOT be eternally promoted', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestColony(tmpDir);

    // REDIRECT with strength 0.9, created 30 days ago
    // effective = 0.9 * (1 - 30/60) = 0.9 * 0.5 = 0.45
    // Already expired (expires_at in the past)
    await writePheromones(tmpDir, [{
      id: 'sig_decayed_redirect',
      type: 'REDIRECT',
      priority: 'high',
      source: 'test',
      created_at: daysAgo(30),
      expires_at: daysAgo(1),  // expired yesterday
      active: true,
      reason: 'Test decayed redirect',
      content: { text: 'Avoid using global state' },
      strength: 0.9
    }]);

    // Run pheromone-expire
    runAetherUtil(tmpDir, 'pheromone-expire');

    // Check eternal memory -- should NOT be promoted
    const entries = await readEternalSignals(tmpDir);
    const promoted = entries.find(e =>
      (e.content && e.content.includes('global state')) ||
      (e.text && e.text.includes('global state')) ||
      (e.signal_id === 'sig_decayed_redirect')
    );
    t.falsy(promoted,
      'Heavily-decayed REDIRECT (effective ~0.45) should NOT be promoted to eternal memory');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('2. Fresh REDIRECT (effective ~0.885) SHOULD be eternally promoted', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestColony(tmpDir);

    // REDIRECT with strength 0.9, created only ~1 day ago
    // effective = 0.9 * (1 - 1/60) ~= 0.9 * 0.983 ~= 0.885
    // Expired (forced via expires_at in the past)
    // We set created 1 day ago but set expires_at to just before now
    const oneDayAgo = daysAgo(1);
    const justExpired = daysAgo(0.001); // just barely past

    await writePheromones(tmpDir, [{
      id: 'sig_fresh_redirect',
      type: 'REDIRECT',
      priority: 'high',
      source: 'test',
      created_at: oneDayAgo,
      expires_at: justExpired,
      active: true,
      reason: 'Test fresh redirect',
      content: { text: 'Never store passwords in plaintext' },
      strength: 0.9
    }]);

    // Run pheromone-expire
    runAetherUtil(tmpDir, 'pheromone-expire');

    // Check eternal memory -- SHOULD be promoted (effective > 0.8)
    const entries = await readEternalSignals(tmpDir);
    const promoted = entries.find(e =>
      (e.content && e.content.includes('passwords')) ||
      (e.text && e.text.includes('passwords')) ||
      (e.signal_id === 'sig_fresh_redirect')
    );
    t.truthy(promoted,
      'Fresh REDIRECT (effective ~0.885) SHOULD be promoted to eternal memory');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('3. REDIRECT at exact boundary (effective ~0.80) should NOT be promoted (threshold is >80)', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestColony(tmpDir);

    // REDIRECT with strength 0.9, we need effective ~= 0.80
    // 0.9 * (1 - x/60) = 0.80 => 1 - x/60 = 0.889 => x = 6.67 days
    // At 6.67 days: effective = 0.9 * (1 - 6.67/60) = 0.9 * 0.889 = 0.8001
    // At 7 days: effective = 0.9 * (1 - 7/60) = 0.9 * 0.883 = 0.795
    // Use 7 days to get just below 0.80
    await writePheromones(tmpDir, [{
      id: 'sig_boundary_redirect',
      type: 'REDIRECT',
      priority: 'high',
      source: 'test',
      created_at: daysAgo(7),
      expires_at: daysAgo(1),
      active: true,
      reason: 'Test boundary redirect',
      content: { text: 'Boundary threshold test signal' },
      strength: 0.9
    }]);

    // Run pheromone-expire
    runAetherUtil(tmpDir, 'pheromone-expire');

    // Check eternal memory -- should NOT be promoted (effective ~0.795 < 0.80)
    const entries = await readEternalSignals(tmpDir);
    const promoted = entries.find(e =>
      (e.content && e.content.includes('Boundary threshold')) ||
      (e.text && e.text.includes('Boundary threshold')) ||
      (e.signal_id === 'sig_boundary_redirect')
    );
    t.falsy(promoted,
      'REDIRECT at boundary (effective ~0.795) should NOT be promoted to eternal memory');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('4. FOCUS with strength 0.9 decayed to ~0.45 should NOT be promoted', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestColony(tmpDir);

    // FOCUS decays over 30 days, created 15 days ago
    // effective = 0.9 * (1 - 15/30) = 0.9 * 0.5 = 0.45
    await writePheromones(tmpDir, [{
      id: 'sig_decayed_focus',
      type: 'FOCUS',
      priority: 'normal',
      source: 'test',
      created_at: daysAgo(15),
      expires_at: daysAgo(1),
      active: true,
      reason: 'Test decayed focus',
      content: { text: 'Focus on authentication module' },
      strength: 0.9
    }]);

    // Run pheromone-expire
    runAetherUtil(tmpDir, 'pheromone-expire');

    // Should NOT be promoted
    const entries = await readEternalSignals(tmpDir);
    const promoted = entries.find(e =>
      (e.content && e.content.includes('authentication')) ||
      (e.text && e.text.includes('authentication')) ||
      (e.signal_id === 'sig_decayed_focus')
    );
    t.falsy(promoted,
      'Decayed FOCUS (effective ~0.45) should NOT be promoted to eternal memory');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});
