/**
 * Hive Intelligence Integration Tests
 *
 * End-to-end tests for Phase 2-3 features:
 * - User preferences in QUEEN.md -> colony-prime prompt_section output
 * - Eternal memory high_value_signals -> colony-prime HIVE WISDOM section
 * - Registry-add with domain tags stores and retrieves correctly
 * - Combined output: user prefs AND hive wisdom present together
 * - Graceful degradation when eternal memory is missing/empty
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

// Helper to create temp directory
async function createTempDir() {
  const tmpDir = await fs.promises.mkdtemp(path.join(os.tmpdir(), 'aether-hive-'));
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

// Helper to run aether-utils.sh commands with colony-prime style env
function runColonyPrime(tmpDir, args = []) {
  const scriptPath = path.join(process.cwd(), '.aether', 'aether-utils.sh');
  const env = {
    ...process.env,
    HOME: tmpDir,
    AETHER_ROOT: tmpDir,
    DATA_DIR: path.join(tmpDir, '.aether', 'data')
  };
  const cmd = `bash "${scriptPath}" colony-prime ${args.map(a => `"${a}"`).join(' ')} 2>/dev/null`;
  return execSync(cmd, { encoding: 'utf8', env, cwd: tmpDir });
}

// Helper to run registry subcommands with isolated HOME
function runRegistryCommand(tmpDir, fakeHome, subcmd, args = []) {
  const scriptPath = path.join(tmpDir, '.aether', 'aether-utils.sh');
  const env = {
    ...process.env,
    HOME: fakeHome
  };
  const cmd = `bash "${scriptPath}" ${subcmd} ${args.map(a => `"${a}"`).join(' ')} 2>/dev/null`;
  return execSync(cmd, { encoding: 'utf8', env, cwd: tmpDir });
}

// Helper to setup test colony with optional user preferences and eternal memory
async function setupTestColony(tmpDir, opts = {}) {
  const aetherDir = path.join(tmpDir, '.aether');
  const dataDir = path.join(aetherDir, 'data');

  await fs.promises.mkdir(dataDir, { recursive: true });

  const isoDate = new Date().toISOString();

  // Build QUEEN.md -- optionally include User Preferences section
  // Section headers must use emoji prefixes to match awk parsing in aether-utils.sh
  let queenMd = `# QUEEN.md --- Colony Wisdom

> Last evolved: ${isoDate}
> Colonies contributed: 0
> Wisdom version: 1.0.0

---

## \u{1F4DC} Philosophies

*No philosophies recorded yet*

---

## \u{1F9ED} Patterns

*No patterns recorded yet*

---

## \u26A0\uFE0F Redirects

*No redirects recorded yet*

---

## \u{1F527} Stack Wisdom

*No stack wisdom recorded yet*

---

## \u{1F3DB}\uFE0F Decrees

*No decrees recorded yet*

---
`;

  if (opts.userPreferences && opts.userPreferences.length > 0) {
    queenMd += `
## \u{1F464} User Preferences

`;
    for (const pref of opts.userPreferences) {
      queenMd += `- ${pref}\n`;
    }
    queenMd += `
---
`;
  }

  queenMd += `
## \u{1F4CA} Evolution Log

| Date | Colony | Change | Details |
|------|--------|--------|---------|

---

<!-- METADATA {"version":"1.0.0","last_evolved":"${isoDate}","colonies_contributed":[],"promotion_thresholds":{"philosophy":1,"pattern":1,"redirect":1,"stack":1,"decree":0},"stats":{"total_philosophies":0,"total_patterns":0,"total_redirects":0,"total_stack_entries":0,"total_decrees":0}} -->`;

  await fs.promises.writeFile(path.join(aetherDir, 'QUEEN.md'), queenMd);

  // Create COLONY_STATE.json
  const currentPhase = opts.currentPhase !== undefined ? opts.currentPhase : 1;
  const phaseLearnings = opts.phaseLearnings || [];

  const colonyState = {
    session_id: 'test_hive_int',
    goal: 'test hive intelligence integration',
    state: 'BUILDING',
    current_phase: currentPhase,
    plan: { phases: [] },
    memory: {
      instincts: [],
      phase_learnings: phaseLearnings,
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

  // Create eternal memory if specified
  if (opts.eternalSignals !== undefined) {
    const eternalDir = path.join(aetherDir, 'eternal');
    await fs.promises.mkdir(eternalDir, { recursive: true });

    const eternalMemory = {
      version: '1.0',
      created_at: '2026-02-17T22:25:34Z',
      colonies: [],
      high_value_signals: opts.eternalSignals,
      cross_session_patterns: []
    };
    await fs.promises.writeFile(
      path.join(eternalDir, 'memory.json'),
      JSON.stringify(eternalMemory, null, 2)
    );
  }

  // Write malformed eternal memory if specified
  if (opts.malformedEternal) {
    const eternalDir = path.join(aetherDir, 'eternal');
    await fs.promises.mkdir(eternalDir, { recursive: true });
    await fs.promises.writeFile(
      path.join(eternalDir, 'memory.json'),
      'NOT VALID JSON {{{'
    );
  }

  return { aetherDir, dataDir };
}

// Helper to setup registry test environment
async function setupRegistryEnv(tmpDir) {
  const aetherDir = path.join(tmpDir, '.aether');
  const fakeHome = path.join(tmpDir, 'fakehome');

  await fs.promises.mkdir(path.join(aetherDir, 'data'), { recursive: true });
  await fs.promises.mkdir(path.join(fakeHome, '.aether'), { recursive: true });

  // Copy aether-utils.sh
  const srcScript = path.join(process.cwd(), '.aether', 'aether-utils.sh');
  await fs.promises.copyFile(srcScript, path.join(aetherDir, 'aether-utils.sh'));

  // Copy utils directory
  const srcUtils = path.join(process.cwd(), '.aether', 'utils');
  if (fs.existsSync(srcUtils)) {
    await fs.promises.cp(srcUtils, path.join(aetherDir, 'utils'), { recursive: true });
  }

  // Copy exchange directory
  const srcExchange = path.join(process.cwd(), '.aether', 'exchange');
  if (fs.existsSync(srcExchange)) {
    await fs.promises.cp(srcExchange, path.join(aetherDir, 'exchange'), { recursive: true });
  }

  // Copy schemas directory
  const srcSchemas = path.join(process.cwd(), '.aether', 'schemas');
  if (fs.existsSync(srcSchemas)) {
    await fs.promises.cp(srcSchemas, path.join(aetherDir, 'schemas'), { recursive: true });
  }

  return fakeHome;
}


// ============================================================================
// Test 1: User preferences in QUEEN.md appear in colony-prime prompt_section
// ============================================================================
test.serial('user preferences in QUEEN.md appear in colony-prime prompt_section', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir, {
      userPreferences: [
        'Communication style: Plain English, no jargon',
        'Expertise level: Non-technical founder',
        'Decision pattern: Prefers quick iteration'
      ]
    });

    const result = runColonyPrime(tmpDir);
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    const section = resultJson.result.prompt_section;

    // USER PREFERENCES block should be present
    t.true(section.includes('USER PREFERENCES'),
      'Should contain USER PREFERENCES header');

    // Preference content should appear
    t.true(section.includes('Plain English'),
      'Should include preference content about communication style');
    t.true(section.includes('Non-technical founder'),
      'Should include preference content about expertise level');
    t.true(section.includes('quick iteration'),
      'Should include preference content about decision pattern');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// ============================================================================
// Test 2: Eternal memory entries injected as HIVE WISDOM section
// ============================================================================
test.serial('eternal memory entries injected into colony-prime as HIVE WISDOM section', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir, {
      eternalSignals: [
        {
          content: 'Use structured logging with correlation IDs',
          type: 'PATTERN',
          strength: 0.9,
          source_colony: 'colony-alpha',
          promoted_at: '2026-03-01T00:00:00Z'
        },
        {
          content: 'Never store secrets in environment variables without encryption',
          type: 'REDIRECT',
          strength: 0.8,
          source_colony: 'colony-beta',
          promoted_at: '2026-03-05T00:00:00Z'
        }
      ]
    });

    const result = runColonyPrime(tmpDir);
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    const section = resultJson.result.prompt_section;

    // HIVE WISDOM section should be present
    t.true(section.includes('HIVE WISDOM'),
      'Should contain HIVE WISDOM header');

    // Signal content should appear
    t.true(section.includes('structured logging'),
      'Should include first signal content');
    t.true(section.includes('secrets'),
      'Should include second signal content');

    // Signal types should appear
    t.true(section.includes('PATTERN'),
      'Should include signal type PATTERN');
    t.true(section.includes('REDIRECT'),
      'Should include signal type REDIRECT');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// ============================================================================
// Test 3: Registry-add with domain tags stores and retrieves correctly
// ============================================================================
test.serial('registry-add with domain tags stores and retrieves correctly', async (t) => {
  const tmpDir = await createTempDir();

  try {
    const fakeHome = await setupRegistryEnv(tmpDir);

    // Add a repo with domain tags, goal, and active status
    const addResult = runRegistryCommand(tmpDir, fakeHome, 'registry-add', [
      '/tmp/test-project', '1.0.0', '--tags', 'node,api,web', '--goal', 'Build REST API', '--active', 'true'
    ]);
    const addJson = JSON.parse(addResult);
    t.true(addJson.ok, 'registry-add should return ok=true');

    // List registry to verify retrieval
    const listResult = runRegistryCommand(tmpDir, fakeHome, 'registry-list');
    const listJson = JSON.parse(listResult);
    t.true(listJson.ok, 'registry-list should return ok=true');

    // Verify repo count
    t.is(listJson.result.repos.length, 1, 'Should have 1 repo');

    // Verify domain tags
    const repo = listJson.result.repos[0];
    t.deepEqual(repo.domain_tags, ['node', 'api', 'web'],
      'domain_tags should be stored as array');

    // Verify goal
    t.is(repo.last_colony_goal, 'Build REST API',
      'last_colony_goal should be stored');

    // Verify active status
    t.is(repo.active_colony, true,
      'active_colony should be true');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// ============================================================================
// Test 4: Colony-prime output contains both user prefs AND hive wisdom
// ============================================================================
test.serial('colony-prime output contains both user prefs AND hive wisdom when both exist', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir, {
      userPreferences: [
        'Prefer dark mode UI',
        'Use concise commit messages'
      ],
      eternalSignals: [
        {
          content: 'Always validate user input before database operations',
          type: 'REDIRECT',
          strength: 0.85,
          source_colony: 'colony-validation',
          promoted_at: '2026-03-01T00:00:00Z'
        }
      ],
      currentPhase: 2,
      phaseLearnings: [{
        phase: '1',
        phase_name: 'Foundation',
        learnings: [{
          claim: 'Test learning for section ordering',
          status: 'validated',
          evidence: ['test'],
          confidence: 0.9
        }]
      }]
    });

    const result = runColonyPrime(tmpDir);
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    const section = resultJson.result.prompt_section;

    // Both sections should exist
    t.true(section.includes('USER PREFERENCES'),
      'Should contain USER PREFERENCES section');
    t.true(section.includes('HIVE WISDOM'),
      'Should contain HIVE WISDOM section');

    // Content from both should be present
    t.true(section.includes('dark mode'),
      'Should include user preference content');
    t.true(section.includes('validate user input'),
      'Should include hive wisdom content');

    // Verify ordering: USER PREFERENCES before HIVE WISDOM before PHASE LEARNINGS
    const prefsIdx = section.indexOf('USER PREFERENCES');
    const hiveIdx = section.indexOf('HIVE WISDOM');
    const learningsIdx = section.indexOf('PHASE LEARNINGS');

    t.true(prefsIdx < hiveIdx,
      'USER PREFERENCES should appear before HIVE WISDOM');
    t.true(hiveIdx < learningsIdx,
      'HIVE WISDOM should appear before PHASE LEARNINGS');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// ============================================================================
// Test 5: Empty/missing eternal memory does not break colony-prime
// ============================================================================
test.serial('empty/missing eternal memory does not break colony-prime (graceful degradation)', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Case A: No eternal memory at all
    await setupTestColony(tmpDir, {
      userPreferences: [
        'Keep things simple'
      ]
    });

    const resultA = runColonyPrime(tmpDir);
    const jsonA = JSON.parse(resultA);

    t.true(jsonA.ok, 'Should return ok=true when eternal memory is missing');

    const sectionA = jsonA.result.prompt_section;

    // USER PREFERENCES should still work
    t.true(sectionA.includes('USER PREFERENCES'),
      'USER PREFERENCES should still appear without eternal memory');
    t.true(sectionA.includes('Keep things simple'),
      'User preference content should still appear');

    // HIVE WISDOM should NOT appear
    t.false(sectionA.includes('HIVE WISDOM'),
      'Should NOT contain HIVE WISDOM when no eternal memory exists');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// ============================================================================
// Test 6: Empty eternal signals array produces no HIVE WISDOM
// ============================================================================
test.serial('empty eternal signals array produces no HIVE WISDOM section', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir, {
      eternalSignals: []
    });

    const result = runColonyPrime(tmpDir);
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    const section = resultJson.result.prompt_section;

    t.false(section.includes('HIVE WISDOM'),
      'Should NOT contain HIVE WISDOM when signals array is empty');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// ============================================================================
// Test 7: Malformed eternal memory JSON handled gracefully
// ============================================================================
test.serial('malformed eternal memory JSON handled gracefully', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir, {
      malformedEternal: true
    });

    const result = runColonyPrime(tmpDir);
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true even with malformed eternal memory');

    const section = resultJson.result.prompt_section;

    t.false(section.includes('HIVE WISDOM'),
      'Should NOT contain HIVE WISDOM when eternal memory is malformed JSON');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// ============================================================================
// Test 8: Log line includes hive wisdom and user prefs counts
// ============================================================================
test.serial('log_line includes hive wisdom and user prefs counts', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir, {
      userPreferences: [
        'Preference A',
        'Preference B'
      ],
      eternalSignals: [
        {
          content: 'Signal for log test',
          type: 'PATTERN',
          strength: 0.7,
          source_colony: 'colony-log',
          promoted_at: '2026-03-01T00:00:00Z'
        }
      ]
    });

    const result = runColonyPrime(tmpDir);
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    const logLine = resultJson.result.log_line;

    // Log line should reference hive wisdom
    t.true(logLine.includes('hive'),
      'Log line should mention hive wisdom entries');

    // Log line should reference user prefs
    t.true(logLine.includes('user_prefs'),
      'Log line should mention user_prefs count');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});
