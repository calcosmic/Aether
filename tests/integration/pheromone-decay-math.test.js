/**
 * Pheromone Decay Math Integration Tests
 *
 * Tests for the decay math pipeline in pheromone-read:
 * - Epoch conversion consistency (to_epoch function)
 * - Decay formula: effective_strength = strength * (1 - elapsed_days / decay_days)
 * - Threshold: signals with effective_strength < 0.1 are inactive
 * - Expiry: signals past expires_at are inactive
 * - Type-specific decay periods: FOCUS=30d, REDIRECT=60d, FEEDBACK=90d
 *
 * Covers PHER-06 (epoch unification) and PHER-02 (decay correctness).
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

// Helper to create temp directory
async function createTempDir() {
  const tmpDir = await fs.promises.mkdtemp(path.join(os.tmpdir(), 'aether-decay-'));
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

// Helper to setup test colony structure
async function setupTestColony(tmpDir, opts = {}) {
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
    session_id: 'colony_decay_test',
    goal: 'test decay math',
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

  return { aetherDir, dataDir };
}

// Helper to create a test signal directly in pheromones.json
async function createTestSignal(tmpDir, overrides = {}) {
  const dataDir = path.join(tmpDir, '.aether', 'data');
  const pherFile = path.join(dataDir, 'pheromones.json');

  const now = new Date();
  const signal = {
    id: overrides.id || `sig_test_${Date.now()}`,
    type: overrides.type || 'FOCUS',
    priority: overrides.priority || 'normal',
    source: overrides.source || 'test',
    created_at: overrides.created_at || now.toISOString(),
    expires_at: overrides.expires_at || new Date(now.getTime() + 30 * 86400000).toISOString(),
    active: overrides.active !== undefined ? overrides.active : true,
    reason: overrides.reason || 'Test signal',
    content: overrides.content || { text: 'Test decay signal' },
    ...('strength' in overrides ? { strength: overrides.strength } : { strength: 0.8 })
  };

  // If overrides has no_strength_field set, remove strength entirely
  if (overrides.no_strength_field) {
    delete signal.strength;
  }

  let pheromones;
  try {
    pheromones = JSON.parse(await fs.promises.readFile(pherFile, 'utf8'));
  } catch {
    pheromones = { signals: [], version: '1.0.0' };
  }
  pheromones.signals.push(signal);
  await fs.promises.writeFile(pherFile, JSON.stringify(pheromones, null, 2));
  return signal;
}

// Helper to read active signals via pheromone-read
function readActiveSignals(tmpDir) {
  const result = runAetherUtil(tmpDir, 'pheromone-read', ['all']);
  const parsed = JSON.parse(result);
  return parsed.result.signals;
}

// Generate ISO timestamp for N days ago
function daysAgo(days) {
  return new Date(Date.now() - days * 86400000).toISOString();
}

// Generate ISO timestamp for N days from now
function daysFromNow(days) {
  return new Date(Date.now() + days * 86400000).toISOString();
}


// =============================================================================
// Decay Math Edge Cases
// =============================================================================

test.serial('1. Zero time elapsed: effective_strength equals original strength', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestColony(tmpDir);
    await createTestSignal(tmpDir, {
      id: 'sig_zero_elapsed',
      type: 'FOCUS',
      strength: 0.8,
      created_at: new Date().toISOString(),
      expires_at: daysFromNow(30)
    });

    const signals = readActiveSignals(tmpDir);
    const signal = signals.find(s => s.id === 'sig_zero_elapsed');
    t.truthy(signal, 'Signal should be in active results');
    // Allow small tolerance for seconds elapsed during test setup
    t.true(Math.abs(signal.effective_strength - 0.8) <= 0.02,
      `effective_strength should be ~0.8 (got ${signal.effective_strength})`);
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('2. Half-life for FOCUS (15 days): effective_strength ~= 0.4', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestColony(tmpDir);
    await createTestSignal(tmpDir, {
      id: 'sig_half_life',
      type: 'FOCUS',
      strength: 0.8,
      created_at: daysAgo(15),
      expires_at: daysFromNow(30)
    });

    const signals = readActiveSignals(tmpDir);
    const signal = signals.find(s => s.id === 'sig_half_life');
    t.truthy(signal, 'Signal should be in active results');
    // FOCUS decay: 0.8 * (1 - 15/30) = 0.8 * 0.5 = 0.4
    t.true(Math.abs(signal.effective_strength - 0.4) <= 0.05,
      `effective_strength should be ~0.4 (got ${signal.effective_strength})`);
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('3. Full decay for FOCUS (30 days): signal is NOT active', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestColony(tmpDir);
    await createTestSignal(tmpDir, {
      id: 'sig_full_decay',
      type: 'FOCUS',
      strength: 0.8,
      created_at: daysAgo(30),
      expires_at: daysFromNow(30)
    });

    const signals = readActiveSignals(tmpDir);
    const signal = signals.find(s => s.id === 'sig_full_decay');
    // At 30 days: 0.8 * (1 - 30/30) = 0.0, below 0.1 threshold
    t.falsy(signal, 'Signal should NOT be in active results (fully decayed)');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('4. Past full decay (31 days): signal is NOT active (clamped to 0)', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestColony(tmpDir);
    await createTestSignal(tmpDir, {
      id: 'sig_past_decay',
      type: 'FOCUS',
      strength: 0.8,
      created_at: daysAgo(31),
      expires_at: daysFromNow(30)
    });

    const signals = readActiveSignals(tmpDir);
    const signal = signals.find(s => s.id === 'sig_past_decay');
    // At 31 days: 0.8 * (1 - 31/30) = negative, clamped to 0, below 0.1
    t.falsy(signal, 'Signal should NOT be in active results (past full decay)');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('5. Missing strength field (fallback to 0.8): effective_strength ~= 0.8', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestColony(tmpDir);
    await createTestSignal(tmpDir, {
      id: 'sig_no_strength',
      type: 'FOCUS',
      created_at: new Date().toISOString(),
      expires_at: daysFromNow(30),
      no_strength_field: true
    });

    const signals = readActiveSignals(tmpDir);
    const signal = signals.find(s => s.id === 'sig_no_strength');
    t.truthy(signal, 'Signal should be in active results');
    // Missing strength defaults to 0.8 via (.strength // 0.8)
    t.true(Math.abs(signal.effective_strength - 0.8) <= 0.02,
      `effective_strength should be ~0.8 with fallback (got ${signal.effective_strength})`);
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('6. expires_at == "phase_end": signal IS active (not time-expired)', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestColony(tmpDir);
    await createTestSignal(tmpDir, {
      id: 'sig_phase_end',
      type: 'FOCUS',
      strength: 0.8,
      created_at: new Date().toISOString(),
      expires_at: 'phase_end'
    });

    const signals = readActiveSignals(tmpDir);
    const signal = signals.find(s => s.id === 'sig_phase_end');
    t.truthy(signal, 'Signal with expires_at="phase_end" should be in active results');
    t.true(signal.effective_strength > 0.1,
      'Signal should have meaningful effective_strength');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('7. expires_at is ISO timestamp in the past: signal is NOT active', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestColony(tmpDir);
    await createTestSignal(tmpDir, {
      id: 'sig_past_expiry',
      type: 'FOCUS',
      strength: 0.8,
      created_at: new Date().toISOString(),
      expires_at: daysAgo(1) // expired yesterday
    });

    const signals = readActiveSignals(tmpDir);
    const signal = signals.find(s => s.id === 'sig_past_expiry');
    t.falsy(signal, 'Signal with past expires_at should NOT be in active results');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('8. expires_at is ISO timestamp in the future: signal IS active', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestColony(tmpDir);
    await createTestSignal(tmpDir, {
      id: 'sig_future_expiry',
      type: 'FOCUS',
      strength: 0.8,
      created_at: new Date().toISOString(),
      expires_at: daysFromNow(30)
    });

    const signals = readActiveSignals(tmpDir);
    const signal = signals.find(s => s.id === 'sig_future_expiry');
    t.truthy(signal, 'Signal with future expires_at should be in active results');
    t.true(signal.effective_strength > 0.1,
      'Signal should have meaningful effective_strength');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('9. Signal with active: false: NOT in active results regardless of decay', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestColony(tmpDir);
    await createTestSignal(tmpDir, {
      id: 'sig_inactive',
      type: 'FOCUS',
      strength: 0.8,
      created_at: new Date().toISOString(),
      expires_at: daysFromNow(30),
      active: false
    });

    const signals = readActiveSignals(tmpDir);
    const signal = signals.find(s => s.id === 'sig_inactive');
    t.falsy(signal, 'Signal with active=false should NOT be in active results');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('10. REDIRECT half-life (30 days): effective_strength ~= 0.45', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestColony(tmpDir);
    await createTestSignal(tmpDir, {
      id: 'sig_redirect_half',
      type: 'REDIRECT',
      strength: 0.9,
      created_at: daysAgo(30),
      expires_at: daysFromNow(60)
    });

    const signals = readActiveSignals(tmpDir);
    const signal = signals.find(s => s.id === 'sig_redirect_half');
    t.truthy(signal, 'REDIRECT signal at 30 days should still be active (60d decay period)');
    // REDIRECT decay: 0.9 * (1 - 30/60) = 0.9 * 0.5 = 0.45
    t.true(Math.abs(signal.effective_strength - 0.45) <= 0.05,
      `effective_strength should be ~0.45 (got ${signal.effective_strength})`);
  } finally {
    await cleanupTempDir(tmpDir);
  }
});
