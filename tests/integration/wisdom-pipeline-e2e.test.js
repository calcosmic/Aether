/**
 * Wisdom Pipeline End-to-End Integration Tests
 *
 * Validates the complete Aether wisdom pipeline from observation capture
 * through hive brain storage. This is the final validation gate for v2.4
 * -- proving that colony work produces real wisdom that flows through
 * the entire system.
 *
 * Pipeline chain:
 * 1. memory-capture "learning" -> records observation, attempts auto-promotion
 * 2. learning-promote-auto (internal) -> queen-promote + instinct-create
 * 3. instinct-create (internal) -> stores instinct in COLONY_STATE.json
 * 4. queen-promote (internal) -> writes to QUEEN.md Patterns/Philosophies section
 * 5. colony-prime -> reads QUEEN.md + instincts -> prompt_section with both
 * 6. hive-promote -> abstracts instinct -> stores in hive wisdom.json
 * 7. hive-read -> retrieves from wisdom.json
 *
 * Requirements covered: VAL-01
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

const SCRIPT_PATH = path.join(process.cwd(), '.aether', 'aether-utils.sh');

// =============================================================================
// Helpers (following existing integration test patterns)
// =============================================================================

// Helper to create temp directory
async function createTempDir() {
  const tmpDir = await fs.promises.mkdtemp(path.join(os.tmpdir(), 'aether-wisdom-e2e-'));
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
// Returns raw output string. Some subcommands emit multiple JSON lines
// when they call other subcommands internally. Use parseLastJson() to
// safely parse the final result.
function runAetherUtil(tmpDir, command, args = []) {
  const env = {
    ...process.env,
    HOME: tmpDir,
    AETHER_ROOT: tmpDir,
    DATA_DIR: path.join(tmpDir, '.aether', 'data')
  };
  const cmd = `bash "${SCRIPT_PATH}" ${command} ${args.map(a => `"${a}"`).join(' ')} 2>/dev/null`;
  try {
    return execSync(cmd, { encoding: 'utf8', env, cwd: tmpDir, timeout: 15000 });
  } catch (err) {
    // Some commands may return JSON on stderr or non-zero exit
    if (err.stdout) return err.stdout;
    throw err;
  }
}

// Helper to parse the last JSON line from multi-line output.
// Some aether-utils subcommands emit multiple JSON objects on separate lines.
// The authoritative result is always the last complete JSON object.
function parseLastJson(output) {
  const lines = output.trim().split('\n');
  // Try the last line first (single-line JSON)
  try {
    return JSON.parse(lines[lines.length - 1]);
  } catch (_) {
    // Fall through
  }
  // Try parsing the entire output as a single JSON (pretty-printed multi-line)
  try {
    return JSON.parse(output.trim());
  } catch (_) {
    // Fall through
  }
  // Walk backwards to find a line that starts with { and parse from there
  for (let i = lines.length - 1; i >= 0; i--) {
    if (lines[i].trim().startsWith('{')) {
      try {
        return JSON.parse(lines.slice(i).join('\n'));
      } catch (_) {
        continue;
      }
    }
  }
  throw new Error('No JSON found in output: ' + output);
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
    session_id: opts.sessionId || 'wisdom_e2e_test',
    goal: opts.goal || 'validate wisdom pipeline end-to-end',
    state: 'BUILDING',
    current_phase: opts.currentPhase !== undefined ? opts.currentPhase : 1,
    plan: { phases: opts.completedPhases || [] },
    memory: {
      instincts: opts.instincts || [],
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

// Initialize hive in the temp dir (HOME must be tmpDir for hive isolation)
function initHive(tmpDir) {
  runAetherUtil(tmpDir, 'hive-init');
}


// =============================================================================
// Test 1: memory-capture records observation
// =============================================================================

test.serial('memory-capture records observation on first call', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // Call memory-capture with realistic content
    const result = parseLastJson(runAetherUtil(tmpDir, 'memory-capture', [
      'learning', 'jq boolean handling requires explicit if/elif chains', 'pattern', 'worker:builder'
    ]));

    // Verify: observation recorded, not promoted (first call)
    t.true(result.ok, 'Should return ok=true');
    t.false(result.result.auto_promoted, 'First call should NOT auto-promote');
    t.is(result.result.observation_count, 1, 'Should have observation_count=1');

    // Verify learning-observations.json has 1 observation with the content
    const obsFile = path.join(tmpDir, '.aether', 'data', 'learning-observations.json');
    const observations = JSON.parse(fs.readFileSync(obsFile, 'utf8'));
    t.is(observations.observations.length, 1, 'Should have 1 observation');
    t.true(observations.observations[0].content.includes('jq boolean'),
      'Observation content should contain "jq boolean"');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// =============================================================================
// Test 2: auto-promotion writes to QUEEN.md after threshold
// =============================================================================

test.serial('auto-promotion writes to QUEEN.md after threshold', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    const content = 'Always use explicit if/elif chains in jq boolean expressions';

    // First call: records observation, not promoted
    const firstCapture = parseLastJson(runAetherUtil(tmpDir, 'memory-capture', [
      'learning', content, 'pattern', 'worker:builder'
    ]));
    t.false(firstCapture.result.auto_promoted, 'First capture should NOT auto-promote');

    // Second call: threshold met, auto-promoted (pattern type threshold = 2)
    const secondCapture = parseLastJson(runAetherUtil(tmpDir, 'memory-capture', [
      'learning', content, 'pattern', 'worker:builder'
    ]));
    t.true(secondCapture.result.auto_promoted, 'Second capture SHOULD auto-promote');

    // Verify QUEEN.md contains the content
    const queenContent = fs.readFileSync(path.join(tmpDir, '.aether', 'QUEEN.md'), 'utf8');
    t.true(queenContent.includes('if/elif') || queenContent.includes('jq'),
      'QUEEN.md should contain promoted content about jq/if-elif');

    // Verify COLONY_STATE.json has instinct with matching content and confidence >= 0.7
    const state = JSON.parse(fs.readFileSync(
      path.join(tmpDir, '.aether', 'data', 'COLONY_STATE.json'), 'utf8'
    ));
    t.true(state.memory.instincts.length >= 1, 'Should have at least 1 instinct');
    const instinct = state.memory.instincts.find(i => i.action === content);
    t.truthy(instinct, 'Should find instinct with matching action');
    t.true(instinct.confidence >= 0.7,
      `Instinct confidence should be >= 0.7 (got ${instinct.confidence})`);
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// =============================================================================
// Test 3: colony-prime includes QUEEN wisdom and instincts
// =============================================================================

test.serial('colony-prime includes QUEEN wisdom and instincts in prompt_section', async (t) => {
  const tmpDir = await createTempDir();

  try {
    const isoDate = new Date().toISOString();

    // Setup with pre-seeded instinct
    await setupTestColony(tmpDir, {
      instincts: [
        {
          id: 'instinct_preseed', trigger: 'jq boolean expressions',
          action: 'use explicit if/elif chains', confidence: 0.85,
          status: 'hypothesis', domain: 'pattern',
          source: 'promoted_from_learning', evidence: ['Phase 3'],
          tested: false, created_at: isoDate, last_applied: null,
          applications: 3, successes: 0, failures: 0
        }
      ]
    });

    // Write a real entry to QUEEN.md Patterns section
    const queenPath = path.join(tmpDir, '.aether', 'QUEEN.md');
    let queenContent = fs.readFileSync(queenPath, 'utf8');
    queenContent = queenContent.replace(
      '*No patterns recorded yet*',
      '- Always use explicit if/elif chains in jq\n\n*No patterns recorded yet*'
    );
    fs.writeFileSync(queenPath, queenContent);

    // Call colony-prime (output may be multi-line, use parseLastJson)
    const primeResult = parseLastJson(runAetherUtil(tmpDir, 'colony-prime'));

    // Verify prompt_section contains both QUEEN WISDOM and instinct content
    t.true(primeResult.ok, 'colony-prime should return ok=true');
    t.truthy(primeResult.result.prompt_section, 'Should have prompt_section');

    const section = primeResult.result.prompt_section;

    // Verify QUEEN WISDOM section header
    t.true(section.includes('QUEEN WISDOM'),
      'prompt_section should contain QUEEN WISDOM header');

    // Verify instinct content appears
    t.true(section.includes('if/elif') || section.includes('jq'),
      'prompt_section should contain instinct content about if/elif or jq');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// =============================================================================
// Test 4: hive-promote stores and hive-read retrieves wisdom
// =============================================================================

test.serial('hive-promote stores and hive-read retrieves wisdom', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);
    initHive(tmpDir);

    // Create a high-confidence instinct directly
    const createResult = parseLastJson(runAetherUtil(tmpDir, 'instinct-create', [
      '--trigger', 'jq boolean handling',
      '--action', 'use explicit if/elif chains',
      '--confidence', '0.85',
      '--domain', 'pattern',
      '--source', 'promoted_from_learning',
      '--evidence', 'Phase 3'
    ]));
    t.true(createResult.ok, 'instinct-create should succeed');

    // Run hive-promote on the instinct
    const promoteResult = parseLastJson(runAetherUtil(tmpDir, 'hive-promote', [
      '--text', 'use explicit if/elif chains',
      '--source-repo', '/tmp/test-repo',
      '--confidence', '0.85',
      '--category', 'pattern'
    ]));
    t.true(promoteResult.ok, 'hive-promote should succeed');
    t.is(promoteResult.result.action, 'promoted',
      'hive-promote action should be "promoted"');

    // Run hive-read to verify
    const readResult = parseLastJson(runAetherUtil(tmpDir, 'hive-read', [
      '--confidence', '0.5'
    ]));
    t.true(readResult.ok, 'hive-read should succeed');

    // Verify entries exist (result.entries is an array, result.total_matched gives count)
    const entries = readResult.result.entries || [];
    t.true(Array.isArray(entries), 'hive-read should return entries array');
    t.true(entries.length >= 1, 'hive-read should return at least 1 entry');

    // Verify wisdom.json file exists with the promoted text
    const wisdomPath = path.join(tmpDir, '.aether', 'hive', 'wisdom.json');
    t.true(fs.existsSync(wisdomPath), 'wisdom.json should exist');

    const wisdom = JSON.parse(fs.readFileSync(wisdomPath, 'utf8'));
    t.is(wisdom.entries.length, 1, 'Should have 1 entry in wisdom.json');
    t.true(wisdom.entries[0].text.includes('if/elif') || wisdom.entries[0].text.includes('explicit'),
      'Wisdom entry should contain the promoted text');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});
