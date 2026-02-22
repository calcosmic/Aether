/**
 * Learning Pipeline Integration Tests
 *
 * End-to-end tests for the learning pipeline:
 * observe -> check promotion -> approve -> promote -> read back via colony-prime
 *
 * These tests verify that FLOW-01 and FLOW-02 work together correctly.
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

// Helper to create temp directory
async function createTempDir() {
  const tmpDir = await fs.promises.mkdtemp(path.join(os.tmpdir(), 'aether-learning-'));
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
  const cmd = `bash "${scriptPath}" ${command} ${args.map(a => `"${a}"`).join(' ')}`;
  return execSync(cmd, { encoding: 'utf8', env, cwd: tmpDir });
}

// Helper to setup test colony structure
async function setupTestColony(tmpDir) {
  const aetherDir = path.join(tmpDir, '.aether');
  const dataDir = path.join(aetherDir, 'data');

  // Create directories
  await fs.promises.mkdir(dataDir, { recursive: true });

  // Create QUEEN.md from template
  const queenTemplate = `# QUEEN.md — Colony Wisdom

> Last evolved: ${new Date().toISOString()}
> Colonies contributed: 0
> Wisdom version: 1.0.0

---

## 📜 Philosophies

Core beliefs that guide all colony work.

*No philosophies recorded yet*

---

## 🧭 Patterns

Validated approaches that consistently work.

*No patterns recorded yet*

---

## ⚠️ Redirects

Anti-patterns to avoid.

*No redirects recorded yet*

---

## 🔧 Stack Wisdom

Technology-specific insights.

*No stack wisdom recorded yet*

---

## 🏛️ Decrees

User-mandated rules.

*No decrees recorded yet*

---

## 📊 Evolution Log

| Date | Colony | Change | Details |
|------|--------|--------|---------|

---

<!-- METADATA
{
  "version": "1.0.0",
  "last_evolved": "${new Date().toISOString()}",
  "colonies_contributed": [],
  "promotion_thresholds": {
    "philosophy": 1,
    "pattern": 1,
    "redirect": 1,
    "stack": 1,
    "decree": 0
  },
  "stats": {
    "total_philosophies": 0,
    "total_patterns": 0,
    "total_redirects": 0,
    "total_stack_entries": 0,
    "total_decrees": 0
  }
}
-->`;

  await fs.promises.writeFile(path.join(aetherDir, 'QUEEN.md'), queenTemplate);

  // Create empty learning-observations.json
  await fs.promises.writeFile(
    path.join(dataDir, 'learning-observations.json'),
    JSON.stringify({ observations: [] }, null, 2)
  );

  return { aetherDir, dataDir };
}

test.serial('learning-observe records a new observation', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // Record an observation
    const result = runAetherUtil(tmpDir, 'learning-observe', [
      'Test observation for pipeline verification',
      'pattern',
      'test-colony'
    ]);

    // Parse result
    const resultJson = JSON.parse(result);
    t.true(resultJson.success, 'Should return success');
    t.is(resultJson.data.observation_count, 1, 'Should have count of 1');
    t.is(resultJson.data.wisdom_type, 'pattern', 'Should be pattern type');
    t.true(resultJson.data.threshold_met, 'Should meet threshold (threshold=1)');
    t.true(resultJson.data.is_new, 'Should be a new observation');

    // Verify file was created
    const obsFile = path.join(tmpDir, '.aether', 'data', 'learning-observations.json');
    t.true(fs.existsSync(obsFile), 'Observations file should exist');

    // Verify content
    const obsContent = JSON.parse(fs.readFileSync(obsFile, 'utf8'));
    t.is(obsContent.observations.length, 1, 'Should have one observation');
    t.is(obsContent.observations[0].content, 'Test observation for pipeline verification');
    t.is(obsContent.observations[0].wisdom_type, 'pattern');
    t.deepEqual(obsContent.observations[0].colonies, ['test-colony']);
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('learning-observe increments count for duplicate content', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // Record same observation twice from different colonies
    runAetherUtil(tmpDir, 'learning-observe', [
      'Duplicate content test',
      'pattern',
      'colony-a'
    ]);

    const result2 = runAetherUtil(tmpDir, 'learning-observe', [
      'Duplicate content test',
      'pattern',
      'colony-b'
    ]);

    const resultJson = JSON.parse(result2);
    t.is(resultJson.data.observation_count, 2, 'Should have count of 2');
    t.false(resultJson.data.is_new, 'Should not be new (existing)');
    t.deepEqual(resultJson.data.colonies.sort(), ['colony-a', 'colony-b'], 'Should have both colonies');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('learning-check-promotion finds threshold-meeting observations', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // Record observations that meet threshold
    runAetherUtil(tmpDir, 'learning-observe', ['Meets threshold', 'pattern', 'test-colony']);
    runAetherUtil(tmpDir, 'learning-observe', ['Also meets threshold', 'philosophy', 'test-colony']);

    // Check for proposals
    const result = runAetherUtil(tmpDir, 'learning-check-promotion');
    const resultJson = JSON.parse(result);

    t.true(resultJson.success, 'Should return success');
    t.is(resultJson.data.proposals.length, 2, 'Should have 2 proposals');

    // Verify proposal structure
    const proposal = resultJson.data.proposals[0];
    t.truthy(proposal.content, 'Proposal should have content');
    t.truthy(proposal.wisdom_type, 'Proposal should have wisdom_type');
    t.truthy(proposal.observation_count, 'Proposal should have observation_count');
    t.truthy(proposal.threshold, 'Proposal should have threshold');
    t.true(proposal.ready, 'Proposal should be ready');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('queen-promote writes wisdom to QUEEN.md', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // First record an observation so threshold is met
    runAetherUtil(tmpDir, 'learning-observe', ['Test pattern for promotion', 'pattern', 'test-colony']);

    // Promote the wisdom
    const result = runAetherUtil(tmpDir, 'queen-promote', [
      'pattern',
      'Test pattern for promotion',
      'test-colony'
    ]);

    const resultJson = JSON.parse(result);
    t.true(resultJson.success, 'Should return success');

    // Verify QUEEN.md was updated
    const queenFile = path.join(tmpDir, '.aether', 'QUEEN.md');
    const queenContent = fs.readFileSync(queenFile, 'utf8');

    t.true(queenContent.includes('Test pattern for promotion'), 'QUEEN.md should contain the promoted wisdom');
    t.true(queenContent.includes('test-colony'), 'QUEEN.md should contain the colony name');

    // Verify it's in the Patterns section
    const patternsSection = queenContent.split('## 🧭 Patterns')[1]?.split('##')[0];
    t.truthy(patternsSection, 'Should have Patterns section');
    t.true(patternsSection.includes('Test pattern for promotion'), 'Pattern should be in Patterns section');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('colony-prime reads promoted wisdom', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // Record and promote wisdom
    runAetherUtil(tmpDir, 'learning-observe', ['Wisdom to be primed', 'pattern', 'test-colony']);
    runAetherUtil(tmpDir, 'queen-promote', ['pattern', 'Wisdom to be primed', 'test-colony']);

    // Create pheromones.json for colony-prime
    const pheromonesFile = path.join(tmpDir, '.aether', 'data', 'pheromones.json');
    await fs.promises.writeFile(
      pheromonesFile,
      JSON.stringify({ signals: [], instincts: [], version: '1.0.0' }, null, 2)
    );

    // Prime the colony
    const result = runAetherUtil(tmpDir, 'colony-prime');
    const resultJson = JSON.parse(result);

    t.true(resultJson.success, 'Should return success');
    t.truthy(resultJson.data.wisdom, 'Should have wisdom');
    t.truthy(resultJson.data.wisdom.patterns, 'Should have patterns section');
    t.true(resultJson.data.wisdom.patterns.includes('Wisdom to be primed'), 'Patterns should include promoted wisdom');
    t.truthy(resultJson.data.prompt_section, 'Should have prompt_section');
    t.true(resultJson.data.prompt_section.includes('Wisdom to be primed'), 'Prompt section should include wisdom');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('complete pipeline: observe -> check -> promote -> prime', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // Step 1: Record observations
    runAetherUtil(tmpDir, 'learning-observe', ['End-to-end test observation 1', 'pattern', 'colony-1']);
    runAetherUtil(tmpDir, 'learning-observe', ['End-to-end test observation 1', 'pattern', 'colony-2']);
    runAetherUtil(tmpDir, 'learning-observe', ['End-to-end test observation 2', 'philosophy', 'colony-1']);

    // Step 2: Check for proposals
    const checkResult = runAetherUtil(tmpDir, 'learning-check-promotion');
    const checkJson = JSON.parse(checkResult);
    t.is(checkJson.data.proposals.length, 2, 'Should find 2 proposals');

    // Step 3: Promote each proposal
    for (const proposal of checkJson.data.proposals) {
      const promoteResult = runAetherUtil(tmpDir, 'queen-promote', [
        proposal.wisdom_type,
        proposal.content,
        proposal.colonies[0]
      ]);
      const promoteJson = JSON.parse(promoteResult);
      t.true(promoteJson.success, `Should promote ${proposal.wisdom_type}`);
    }

    // Step 4: Create pheromones and prime
    const pheromonesFile = path.join(tmpDir, '.aether', 'data', 'pheromones.json');
    await fs.promises.writeFile(
      pheromonesFile,
      JSON.stringify({ signals: [], instincts: [], version: '1.0.0' }, null, 2)
    );

    const primeResult = runAetherUtil(tmpDir, 'colony-prime');
    const primeJson = JSON.parse(primeResult);

    t.true(primeJson.success, 'colony-prime should succeed');
    t.true(primeJson.data.wisdom.patterns.includes('End-to-end test observation 1'), 'Should have pattern');
    t.true(primeJson.data.wisdom.philosophies.includes('End-to-end test observation 2'), 'Should have philosophy');

    // Step 5: Verify QUEEN.md has both entries
    const queenFile = path.join(tmpDir, '.aether', 'QUEEN.md');
    const queenContent = fs.readFileSync(queenFile, 'utf8');
    t.true(queenContent.includes('End-to-end test observation 1'), 'QUEEN.md should have observation 1');
    t.true(queenContent.includes('End-to-end test observation 2'), 'QUEEN.md should have observation 2');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('decree type promotes immediately with threshold 0', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // Record a decree (threshold=0)
    const observeResult = runAetherUtil(tmpDir, 'learning-observe', [
      'Immediate decree test',
      'decree',
      'test-colony'
    ]);
    const observeJson = JSON.parse(observeResult);

    t.true(observeJson.data.threshold_met, 'Decree should meet threshold immediately');

    // Promote it
    runAetherUtil(tmpDir, 'queen-promote', ['decree', 'Immediate decree test', 'test-colony']);

    // Verify in QUEEN.md
    const queenFile = path.join(tmpDir, '.aether', 'QUEEN.md');
    const queenContent = fs.readFileSync(queenFile, 'utf8');
    t.true(queenContent.includes('Immediate decree test'), 'Decree should be in QUEEN.md');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('failure type maps to patterns section when promoted', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // Record a failure observation
    runAetherUtil(tmpDir, 'learning-observe', ['Failure pattern to learn', 'failure', 'test-colony']);

    // Promote it (failure maps to pattern section)
    runAetherUtil(tmpDir, 'queen-promote', ['failure', 'Failure pattern to learn', 'test-colony']);

    // Verify it's in Patterns section
    const queenFile = path.join(tmpDir, '.aether', 'QUEEN.md');
    const queenContent = fs.readFileSync(queenFile, 'utf8');

    t.true(queenContent.includes('Failure pattern to learn'), 'Failure should be promoted');

    const patternsSection = queenContent.split('## 🧭 Patterns')[1]?.split('##')[0];
    t.true(patternsSection.includes('Failure pattern to learn'), 'Failure should be in Patterns section');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});
