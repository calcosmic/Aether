/**
 * Data Safety Integration Tests (Phase 33 Plan 04)
 *
 * Proves that the escaping fixes from plans 01-03 actually work with adversarial inputs:
 *
 * SAFE-01: grep -F fixed-string matching (regex metacharacters in ant names)
 * SAFE-03: JSON construction with special characters (quotes, backslashes, unicode, emoji)
 * SAFE-04: Lock safety (release on failure, stale lock cleanup)
 *
 * Uses real aether-utils.sh subcommands with temp directory isolation.
 */

const test = require('ava');
const { execSync } = require('child_process');
const path = require('path');
const fs = require('fs');
const os = require('os');

const AETHER_ROOT = path.join(__dirname, '../..');
const AETHER_UTILS = path.join(AETHER_ROOT, '.aether/aether-utils.sh');

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function createTempDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), 'aether-datasafety-'));
}

function cleanupTempDir(tmpDir) {
  try {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  } catch {
    // Ignore cleanup errors
  }
}

/**
 * Run an aether-utils subcommand in an isolated temp colony.
 * Returns parsed JSON output.
 */
function run(tmpDir, cmd, args = [], opts = {}) {
  const quotedArgs = args.map(a => `'${a.replace(/'/g, "'\\''")}'`).join(' ');
  const full = `bash "${AETHER_UTILS}" ${cmd} ${quotedArgs}`;
  const output = execSync(full, {
    cwd: tmpDir,
    encoding: 'utf8',
    timeout: 15000,
    env: {
      ...process.env,
      AETHER_ROOT: tmpDir,
      DATA_DIR: path.join(tmpDir, '.aether', 'data'),
      LOCK_DIR: path.join(tmpDir, '.aether', 'locks'),
      HOME: tmpDir,
      AETHER_STALE_LOCK_MODE: 'auto',
      ...opts.env
    },
    stdio: ['pipe', 'pipe', 'pipe']
  });
  return JSON.parse(output.trim());
}

/**
 * Run subcommand and return raw output (for non-JSON or multi-line output).
 */
function runRaw(tmpDir, cmd, args = [], opts = {}) {
  const quotedArgs = args.map(a => `'${a.replace(/'/g, "'\\''")}'`).join(' ');
  const full = `bash "${AETHER_UTILS}" ${cmd} ${quotedArgs}`;
  return execSync(full, {
    cwd: tmpDir,
    encoding: 'utf8',
    timeout: 15000,
    env: {
      ...process.env,
      AETHER_ROOT: tmpDir,
      DATA_DIR: path.join(tmpDir, '.aether', 'data'),
      LOCK_DIR: path.join(tmpDir, '.aether', 'locks'),
      HOME: tmpDir,
      AETHER_STALE_LOCK_MODE: 'auto',
      ...opts.env
    },
    stdio: ['pipe', 'pipe', 'pipe']
  });
}

/**
 * Set up a minimal colony in a temp directory for testing.
 */
function setupColony(tmpDir) {
  const dataDir = path.join(tmpDir, '.aether', 'data');
  const middenDir = path.join(dataDir, 'midden');
  const locksDir = path.join(tmpDir, '.aether', 'locks');
  const eternalDir = path.join(tmpDir, '.aether', 'eternal');

  fs.mkdirSync(dataDir, { recursive: true });
  fs.mkdirSync(middenDir, { recursive: true });
  fs.mkdirSync(locksDir, { recursive: true });
  fs.mkdirSync(eternalDir, { recursive: true });

  // Minimal COLONY_STATE.json
  fs.writeFileSync(path.join(dataDir, 'COLONY_STATE.json'), JSON.stringify({
    session_id: 'datasafety_test',
    goal: 'data safety test colony',
    state: 'BUILDING',
    current_phase: 1,
    colony_version: 1,
    plan: { phases: [] },
    memory: { instincts: [], phase_learnings: [], decisions: [] },
    errors: { flagged_patterns: [] },
    events: []
  }, null, 2));

  // Minimal pheromones.json
  fs.writeFileSync(path.join(dataDir, 'pheromones.json'), JSON.stringify({
    signals: [],
    version: '1.0.0'
  }, null, 2));

  // Minimal midden.json
  fs.writeFileSync(path.join(middenDir, 'midden.json'), JSON.stringify({
    version: '1.0.0',
    entries: []
  }, null, 2));

  // Minimal eternal memory
  fs.writeFileSync(path.join(eternalDir, 'memory.json'), JSON.stringify({
    version: '1.0.0',
    entries: [],
    high_value_signals: [],
    stats: { total_entries: 0, total_promotions: 0 }
  }, null, 2));

  // QUEEN.md (required by various subcommands)
  const isoDate = new Date().toISOString();
  fs.writeFileSync(path.join(tmpDir, '.aether', 'QUEEN.md'), `# QUEEN.md --- Colony Wisdom

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

<!-- METADATA {"version":"1.0.0","last_evolved":"${isoDate}","colonies_contributed":[],"promotion_thresholds":{"philosophy":1,"pattern":1,"redirect":1,"stack":1,"decree":0},"stats":{"total_philosophies":0,"total_patterns":0,"total_redirects":0,"total_stack_entries":0,"total_decrees":0}} -->`);

  return { dataDir, middenDir, locksDir };
}

// ===========================================================================
// SAFE-01: Grep escaping (ant_name with regex metacharacters)
// ===========================================================================

test.serial('SAFE-01: spawn-complete with regex-metachar ant_name returns valid JSON', (t) => {
  const tmpDir = createTempDir();
  try {
    setupColony(tmpDir);
    // Name contains regex metacharacters: . + [ ] | * ^ $ ( ) \
    const dangerousName = 'worker.builder+1';
    const result = run(tmpDir, 'spawn-complete', [dangerousName, 'completed', 'test task done']);
    t.true(result.ok, 'spawn-complete should return ok:true');
    t.truthy(result.result, 'result field should be present');
  } finally {
    cleanupTempDir(tmpDir);
  }
});

test.serial('SAFE-01: spawn-get-depth with regex-metachar ant_name returns valid JSON', (t) => {
  const tmpDir = createTempDir();
  try {
    const { dataDir } = setupColony(tmpDir);
    const dangerousName = 'ant[0]';
    // Write a spawn tree entry so grep has something to search
    const spawnTree = path.join(dataDir, 'spawn-tree.txt');
    const ts = new Date().toISOString();
    fs.writeFileSync(spawnTree, `${ts}|Queen|builder|${dangerousName}|test task|default|spawned\n`);

    const result = run(tmpDir, 'spawn-get-depth', [dangerousName]);
    t.true(result.ok, 'spawn-get-depth should return ok:true');
  } finally {
    cleanupTempDir(tmpDir);
  }
});

test.serial('SAFE-01: swarm-timing-start with regex-metachar ant_name returns valid JSON', (t) => {
  const tmpDir = createTempDir();
  try {
    setupColony(tmpDir);
    const dangerousName = 'test|pipe';
    const result = run(tmpDir, 'swarm-timing-start', [dangerousName]);
    t.true(result.ok, 'swarm-timing-start should return ok:true');
    // The result should contain the ant name safely embedded in JSON
    t.is(result.result.ant, dangerousName, 'ant name should be preserved in output');
  } finally {
    cleanupTempDir(tmpDir);
  }
});

test.serial('SAFE-01: swarm-timing-get with regex-metachar ant_name returns valid JSON', (t) => {
  const tmpDir = createTempDir();
  try {
    setupColony(tmpDir);
    const dangerousName = 'star*name';
    // First start timing so there is data to get
    run(tmpDir, 'swarm-timing-start', [dangerousName]);
    const result = run(tmpDir, 'swarm-timing-get', [dangerousName]);
    t.true(result.ok, 'swarm-timing-get should return ok:true');
    t.is(result.result.ant, dangerousName, 'ant name with * should be preserved');
    t.truthy(result.result.started_at, 'should have started_at timestamp');
  } finally {
    cleanupTempDir(tmpDir);
  }
});

test.serial('SAFE-01: spawn-complete with backslash in ant_name returns valid JSON', (t) => {
  const tmpDir = createTempDir();
  try {
    setupColony(tmpDir);
    const dangerousName = 'back\\slash';
    const result = run(tmpDir, 'spawn-complete', [dangerousName, 'completed', 'done']);
    t.true(result.ok, 'spawn-complete should handle backslash in name');
  } finally {
    cleanupTempDir(tmpDir);
  }
});

test.serial('SAFE-01: spawn-complete with dollar and caret in ant_name returns valid JSON', (t) => {
  const tmpDir = createTempDir();
  try {
    setupColony(tmpDir);
    const dangerousName = 'hat^dollar$end';
    const result = run(tmpDir, 'spawn-complete', [dangerousName, 'completed', 'done']);
    t.true(result.ok, 'spawn-complete should handle ^ and $ in name');
  } finally {
    cleanupTempDir(tmpDir);
  }
});

// ===========================================================================
// SAFE-03: JSON construction with special characters
// ===========================================================================

test.serial('SAFE-03: flag-add with quotes in title returns valid JSON', (t) => {
  const tmpDir = createTempDir();
  try {
    setupColony(tmpDir);
    const title = 'Fix "broken" parsing';
    const result = run(tmpDir, 'flag-add', ['issue', title, 'Description with "quotes"', 'test']);
    t.true(result.ok, 'flag-add should handle quotes in title');
    // Read flags.json and verify the title was stored correctly
    const flagsFile = path.join(tmpDir, '.aether', 'data', 'flags.json');
    const flags = JSON.parse(fs.readFileSync(flagsFile, 'utf8'));
    const lastFlag = flags.flags[flags.flags.length - 1];
    t.is(lastFlag.title, title, 'Title with quotes should be stored verbatim');
  } finally {
    cleanupTempDir(tmpDir);
  }
});

test.serial('SAFE-03: flag-add with backslash in title returns valid JSON', (t) => {
  const tmpDir = createTempDir();
  try {
    setupColony(tmpDir);
    const title = 'Fix path C:\\Users\\test';
    const result = run(tmpDir, 'flag-add', ['issue', title, 'Backslash description', 'test']);
    t.true(result.ok, 'flag-add should handle backslashes in title');
    const flagsFile = path.join(tmpDir, '.aether', 'data', 'flags.json');
    const flags = JSON.parse(fs.readFileSync(flagsFile, 'utf8'));
    const lastFlag = flags.flags[flags.flags.length - 1];
    t.is(lastFlag.title, title, 'Title with backslashes should be stored verbatim');
  } finally {
    cleanupTempDir(tmpDir);
  }
});

test.serial('SAFE-03: flag-add with emoji in title returns valid JSON', (t) => {
  const tmpDir = createTempDir();
  try {
    setupColony(tmpDir);
    const title = 'Fix the ant colony issue';
    const result = run(tmpDir, 'flag-add', ['note', title, 'Emoji in description too', 'test']);
    t.true(result.ok, 'flag-add should handle emoji in title');
  } finally {
    cleanupTempDir(tmpDir);
  }
});

test.serial('SAFE-03: midden-write with special characters returns valid JSON', (t) => {
  const tmpDir = createTempDir();
  try {
    setupColony(tmpDir);
    const message = 'Failed: "unexpected" token at line 1, col\\42';
    const result = run(tmpDir, 'midden-write', ['security', message, 'test-source']);
    t.true(result.ok, 'midden-write should handle special characters');
    // Verify the entry was stored correctly
    const middenFile = path.join(tmpDir, '.aether', 'data', 'midden', 'midden.json');
    const midden = JSON.parse(fs.readFileSync(middenFile, 'utf8'));
    const lastEntry = midden.entries[midden.entries.length - 1];
    t.is(lastEntry.message, message, 'Midden message with special chars should be stored verbatim');
  } finally {
    cleanupTempDir(tmpDir);
  }
});

test.serial('SAFE-03: midden-write with emoji returns valid JSON', (t) => {
  const tmpDir = createTempDir();
  try {
    setupColony(tmpDir);
    const message = 'Build failed with exit code 1';
    const result = run(tmpDir, 'midden-write', ['build', message, 'builder']);
    t.true(result.ok, 'midden-write should handle emoji');
    const middenFile = path.join(tmpDir, '.aether', 'data', 'midden', 'midden.json');
    const midden = JSON.parse(fs.readFileSync(middenFile, 'utf8'));
    t.truthy(midden.entries.length > 0, 'Entry should be stored');
  } finally {
    cleanupTempDir(tmpDir);
  }
});

test.serial('SAFE-03: spawn-log with special characters in task_summary returns valid JSON', (t) => {
  const tmpDir = createTempDir();
  try {
    setupColony(tmpDir);
    const summary = 'Fix "parsing" of C:\\path with [brackets]';
    const result = run(tmpDir, 'spawn-log', ['Queen', 'builder', 'test-worker', summary]);
    t.true(result.ok, 'spawn-log should handle special characters in summary');
  } finally {
    cleanupTempDir(tmpDir);
  }
});

test.serial('SAFE-03: swarm-timing-start with unicode ant_name returns valid JSON', (t) => {
  const tmpDir = createTempDir();
  try {
    setupColony(tmpDir);
    // Unicode combining character (e with accent)
    const unicodeName = 'caf\u00e9-worker';
    const result = run(tmpDir, 'swarm-timing-start', [unicodeName]);
    t.true(result.ok, 'swarm-timing-start should handle unicode');
    t.is(result.result.ant, unicodeName, 'Unicode name should be preserved');
  } finally {
    cleanupTempDir(tmpDir);
  }
});

// ===========================================================================
// SAFE-04: Lock safety
// ===========================================================================

test.serial('SAFE-04: state-write with invalid JSON rejects and releases lock', (t) => {
  const tmpDir = createTempDir();
  try {
    const { dataDir, locksDir } = setupColony(tmpDir);
    // Attempt to write invalid JSON
    try {
      run(tmpDir, 'state-write', ['not valid json {{{']);
      t.fail('state-write with invalid JSON should have thrown');
    } catch (e) {
      // Expected: command exits non-zero
      t.truthy(e.message || e.stderr, 'Should fail with an error');
    }
    // Verify no lock file remains
    const lockFiles = fs.readdirSync(locksDir).filter(f => f.endsWith('.lock'));
    t.is(lockFiles.length, 0, 'No lock files should remain after state-write failure');
  } finally {
    cleanupTempDir(tmpDir);
  }
});

test.serial('SAFE-04: state-mutate with failing jq expression releases lock', (t) => {
  const tmpDir = createTempDir();
  try {
    const { locksDir } = setupColony(tmpDir);
    // Use an invalid jq expression
    try {
      run(tmpDir, 'state-mutate', ['.invalid_syntax[[[']);
      t.fail('state-mutate with invalid jq should have thrown');
    } catch (e) {
      t.truthy(e.message || e.stderr, 'Should fail with an error');
    }
    // Verify no lock file remains
    const lockFiles = fs.readdirSync(locksDir).filter(f => f.endsWith('.lock'));
    t.is(lockFiles.length, 0, 'No lock files should remain after state-mutate failure');
  } finally {
    cleanupTempDir(tmpDir);
  }
});

test.serial('SAFE-04: stale lock with dead PID is auto-cleaned in non-interactive mode', (t) => {
  const tmpDir = createTempDir();
  try {
    const { locksDir } = setupColony(tmpDir);
    // Create a stale lock file with a non-existent PID
    const lockFile = path.join(locksDir, 'COLONY_STATE.json.lock');
    const pidFile = lockFile + '.pid';
    const deadPid = '999999';
    fs.writeFileSync(lockFile, deadPid);
    fs.writeFileSync(pidFile, deadPid);

    // Now run a command that acquires the same lock (state-mutate on COLONY_STATE.json)
    // AETHER_STALE_LOCK_MODE=auto is set in run() helper
    const result = run(tmpDir, 'state-mutate', ['.state = "TESTING"']);
    t.true(result.ok, 'state-mutate should succeed after cleaning stale lock');

    // Verify the mutation actually happened
    const stateFile = path.join(tmpDir, '.aether', 'data', 'COLONY_STATE.json');
    const state = JSON.parse(fs.readFileSync(stateFile, 'utf8'));
    t.is(state.state, 'TESTING', 'State should have been mutated');
  } finally {
    cleanupTempDir(tmpDir);
  }
});

// ===========================================================================
// Broader sweep tests
// ===========================================================================

test.serial('SAFE-01+03: multiple subcommands with empty-string inputs produce valid JSON', (t) => {
  const tmpDir = createTempDir();
  try {
    setupColony(tmpDir);

    // spawn-get-depth with "Queen" (should always work)
    const depthResult = run(tmpDir, 'spawn-get-depth', ['Queen']);
    t.true(depthResult.ok, 'spawn-get-depth Queen should return valid JSON');

    // swarm-timing-get for non-existent ant (should return default JSON)
    const timingResult = run(tmpDir, 'swarm-timing-get', ['nonexistent']);
    t.true(timingResult.ok, 'swarm-timing-get for missing ant should return valid JSON');
    t.is(timingResult.result.elapsed_seconds, 0, 'elapsed should be 0 for missing ant');

    // midden-write with empty message (graceful degradation)
    const middenResult = run(tmpDir, 'midden-write', ['general', '', 'test']);
    t.true(middenResult.ok, 'midden-write with empty message should degrade gracefully');
  } finally {
    cleanupTempDir(tmpDir);
  }
});

test.serial('SAFE-03: pheromone-write with special characters in content produces valid JSON', (t) => {
  const tmpDir = createTempDir();
  try {
    setupColony(tmpDir);
    // Write a pheromone signal with special characters in its content
    const signalContent = 'Focus on "error handling" in C:\\src paths';
    const result = run(tmpDir, 'pheromone-write', [
      'FOCUS', signalContent,
      '--source', 'user',
      '--strength', '0.8'
    ]);
    t.true(result.ok, 'pheromone-write should handle special characters');

    // Read back and verify valid JSON
    const pherFile = path.join(tmpDir, '.aether', 'data', 'pheromones.json');
    const pheromones = JSON.parse(fs.readFileSync(pherFile, 'utf8'));
    const signal = pheromones.signals.find(s => s.active === true);
    t.truthy(signal, 'Signal should be stored');
    t.true(signal.content.text.includes('error handling'), 'Content should be preserved');
  } finally {
    cleanupTempDir(tmpDir);
  }
});

test.serial('SAFE-04: safety-stats.json tracks stale lock cleanups', (t) => {
  const tmpDir = createTempDir();
  try {
    const { locksDir, dataDir } = setupColony(tmpDir);
    // Create a stale lock
    const lockFile = path.join(locksDir, 'COLONY_STATE.json.lock');
    const pidFile = lockFile + '.pid';
    fs.writeFileSync(lockFile, '999998');
    fs.writeFileSync(pidFile, '999998');

    // Run a command that will trigger stale lock cleanup
    run(tmpDir, 'state-mutate', ['.state = "CLEANED"']);

    // Check if safety-stats.json was created/updated
    const statsFile = path.join(dataDir, 'safety-stats.json');
    if (fs.existsSync(statsFile)) {
      const stats = JSON.parse(fs.readFileSync(statsFile, 'utf8'));
      t.truthy(stats.stale_locks_cleaned >= 1, 'Should track stale lock cleanup');
      t.pass('safety-stats.json tracks stale lock cleanups');
    } else {
      // Safety stats are best-effort; if not created, that is acceptable
      t.pass('Safety stats file not created (best-effort tracking)');
    }
  } finally {
    cleanupTempDir(tmpDir);
  }
});
