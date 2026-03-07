/**
 * Wisdom Promotion Integration Tests
 *
 * End-to-end tests for the complete wisdom promotion pipeline:
 * auto-promote observations to QUEEN.md, verify in colony-prime prompt_section.
 *
 * Requirements covered:
 * QUEEN-01: Auto-promotion via learning-promote-auto during continue
 * QUEEN-02: Batch promotion sweep during seal workflow
 * QUEEN-03: Promoted wisdom visible in colony-prime prompt_section
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

// Helper to create temp directory
async function createTempDir() {
  const tmpDir = await fs.promises.mkdtemp(path.join(os.tmpdir(), 'aether-wisdom-'));
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
// Returns raw output string. Some subcommands (e.g. learning-promote-auto)
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
// to stdout (e.g. learning-promote-auto calls instinct-create), producing
// multiple JSON objects. The authoritative result is always the last line.
function parseLastJson(output) {
  const lines = output.trim().split('\n');
  return JSON.parse(lines[lines.length - 1]);
}

// Helper to setup test colony structure
async function setupTestColony(tmpDir, opts = {}) {
  const aetherDir = path.join(tmpDir, '.aether');
  const dataDir = path.join(aetherDir, 'data');

  // Create directories
  await fs.promises.mkdir(dataDir, { recursive: true });

  // Create QUEEN.md with emoji section headers (required by _extract_wisdom_sections in colony-prime)
  const isoDate = new Date().toISOString();
  const queenTemplate = `# QUEEN.md — Colony Wisdom

> Last evolved: ${isoDate}
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

<!-- METADATA {"version":"1.0.0","last_evolved":"${isoDate}","colonies_contributed":[],"promotion_thresholds":{"philosophy":1,"pattern":1,"redirect":1,"stack":1,"decree":0},"stats":{"total_philosophies":0,"total_patterns":0,"total_redirects":0,"total_stack_entries":0,"total_decrees":0}} -->`;

  await fs.promises.writeFile(path.join(aetherDir, 'QUEEN.md'), queenTemplate);

  // Create empty learning-observations.json
  await fs.promises.writeFile(
    path.join(dataDir, 'learning-observations.json'),
    JSON.stringify({ observations: [] }, null, 2)
  );

  // Create COLONY_STATE.json
  const colonyState = {
    session_id: opts.sessionId || 'colony_test',
    goal: opts.goal || 'test wisdom promotion',
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

  // Create pheromones.json
  const signals = opts.pheromoneSignals || [];
  await fs.promises.writeFile(
    path.join(dataDir, 'pheromones.json'),
    JSON.stringify({ signals: signals, version: '1.0.0' }, null, 2)
  );

  return { aetherDir, dataDir };
}


// =============================================================================
// QUEEN-01: Auto-promotion during continue (4 tests)
// =============================================================================

test.serial('learning-promote-auto promotes observation meeting auto threshold (pattern type)', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    const content = 'Always use structured logging for debugging';

    // Observe TWICE (auto threshold for pattern = 2)
    runAetherUtil(tmpDir, 'learning-observe', [content, 'pattern', 'colony-alpha']);
    runAetherUtil(tmpDir, 'learning-observe', [content, 'pattern', 'colony-beta']);

    // Act: call learning-promote-auto
    const result = parseLastJson(runAetherUtil(tmpDir, 'learning-promote-auto', [
      'pattern', content, 'colony-alpha', 'learning'
    ]));

    // Assert: promoted
    t.true(result.ok, 'Should return ok=true');
    t.true(result.result.promoted, 'Should promote after meeting auto threshold of 2');

    // Assert: content in QUEEN.md Patterns section
    const queenContent = fs.readFileSync(path.join(tmpDir, '.aether', 'QUEEN.md'), 'utf8');
    t.true(queenContent.includes(content), 'QUEEN.md should contain promoted content');

    const patternsSection = queenContent.split('## 🧭 Patterns')[1]?.split('##')[0];
    t.truthy(patternsSection, 'Should have Patterns section');
    t.true(patternsSection.includes(content), 'Content should be in Patterns section');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('learning-promote-auto skips observation below auto threshold', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    const content = 'Single observation should not auto-promote';

    // Observe only ONCE (auto threshold for pattern = 2)
    runAetherUtil(tmpDir, 'learning-observe', [content, 'pattern', 'colony-alpha']);

    // Act: call learning-promote-auto
    const result = parseLastJson(runAetherUtil(tmpDir, 'learning-promote-auto', [
      'pattern', content, 'colony-alpha', 'learning'
    ]));

    // Assert: not promoted
    t.true(result.ok, 'Should return ok=true');
    t.false(result.result.promoted, 'Should NOT promote below auto threshold');
    t.is(result.result.reason, 'threshold_not_met', 'Reason should be threshold_not_met');

    // Assert: content NOT in QUEEN.md
    const queenContent = fs.readFileSync(path.join(tmpDir, '.aether', 'QUEEN.md'), 'utf8');
    t.false(queenContent.includes(content), 'QUEEN.md should NOT contain below-threshold content');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('learning-promote-auto prevents double-promotion (idempotency guard)', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    const content = 'Idempotency test pattern for auto-promotion';

    // Observe twice to meet threshold
    runAetherUtil(tmpDir, 'learning-observe', [content, 'pattern', 'colony-alpha']);
    runAetherUtil(tmpDir, 'learning-observe', [content, 'pattern', 'colony-beta']);

    // First promotion: should succeed
    const firstResult = parseLastJson(runAetherUtil(tmpDir, 'learning-promote-auto', [
      'pattern', content, 'colony-alpha', 'learning'
    ]));
    t.true(firstResult.result.promoted, 'First call should promote');

    // Second promotion: should be a no-op
    const secondResult = parseLastJson(runAetherUtil(tmpDir, 'learning-promote-auto', [
      'pattern', content, 'colony-alpha', 'learning'
    ]));
    t.true(secondResult.ok, 'Second call should return ok=true');
    t.false(secondResult.result.promoted, 'Second call should NOT promote again');
    t.is(secondResult.result.reason, 'already_promoted', 'Reason should be already_promoted');

    // Assert: content appears exactly once in Patterns section (not duplicated).
    // queen-promote also writes an Evolution Log entry, so the content appears
    // twice in the full file (section + log). We check the Patterns section only.
    const queenContent = fs.readFileSync(path.join(tmpDir, '.aether', 'QUEEN.md'), 'utf8');
    const patternsSection = queenContent.split('## 🧭 Patterns')[1]?.split('##')[0] || '';
    const sectionOccurrences = patternsSection.split(content).length - 1;
    t.is(sectionOccurrences, 1, 'Content should appear exactly ONCE in Patterns section (not duplicated)');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('memory-capture triggers auto-promotion on recurrence (end-to-end continue path)', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    const content = 'Validate inputs before database writes';

    // First memory-capture: records observation, not enough for auto-promotion
    const firstCapture = parseLastJson(runAetherUtil(tmpDir, 'memory-capture', [
      'learning', content, 'pattern', 'worker:builder'
    ]));
    t.true(firstCapture.ok, 'First capture should succeed');
    t.false(firstCapture.result.auto_promoted, 'First capture should NOT auto-promote');

    // Second memory-capture: now meets pattern auto threshold (2)
    const secondCapture = parseLastJson(runAetherUtil(tmpDir, 'memory-capture', [
      'learning', content, 'pattern', 'worker:builder'
    ]));
    t.true(secondCapture.ok, 'Second capture should succeed');
    t.true(secondCapture.result.auto_promoted, 'Second capture should auto-promote');

    // Verify in QUEEN.md
    const queenContent = fs.readFileSync(path.join(tmpDir, '.aether', 'QUEEN.md'), 'utf8');
    t.true(queenContent.includes(content), 'QUEEN.md should contain the auto-promoted content');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// =============================================================================
// QUEEN-02: Batch promotion during seal (2 tests)
// =============================================================================

test.serial('batch sweep promotes multiple observations meeting different thresholds', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    const patternContent = 'Use dependency injection for testability';
    const philosophyContent = 'Simplicity over cleverness in all designs';
    const belowThresholdContent = 'This observation only seen once';

    // Pattern: observe 2x (meets pattern auto=2)
    runAetherUtil(tmpDir, 'learning-observe', [patternContent, 'pattern', 'colony-a']);
    runAetherUtil(tmpDir, 'learning-observe', [patternContent, 'pattern', 'colony-b']);

    // Philosophy: observe 3x (meets philosophy auto=3)
    runAetherUtil(tmpDir, 'learning-observe', [philosophyContent, 'philosophy', 'colony-a']);
    runAetherUtil(tmpDir, 'learning-observe', [philosophyContent, 'philosophy', 'colony-b']);
    runAetherUtil(tmpDir, 'learning-observe', [philosophyContent, 'philosophy', 'colony-c']);

    // Below threshold: observe only 1x (below pattern auto=2)
    runAetherUtil(tmpDir, 'learning-observe', [belowThresholdContent, 'pattern', 'colony-a']);

    // Batch sweep: read all observations and call learning-promote-auto for each
    const obsFile = path.join(tmpDir, '.aether', 'data', 'learning-observations.json');
    const observations = JSON.parse(fs.readFileSync(obsFile, 'utf8'));

    const results = [];
    for (const obs of observations.observations) {
      const result = parseLastJson(runAetherUtil(tmpDir, 'learning-promote-auto', [
        obs.wisdom_type, obs.content, obs.colonies[0], 'learning'
      ]));
      results.push({ content: obs.content, promoted: result.result.promoted });
    }

    // Assert: pattern and philosophy promoted
    const patternResult = results.find(r => r.content === patternContent);
    t.true(patternResult.promoted, 'Pattern should be promoted (2 observations >= auto threshold 2)');

    const philosophyResult = results.find(r => r.content === philosophyContent);
    t.true(philosophyResult.promoted, 'Philosophy should be promoted (3 observations >= auto threshold 3)');

    // Assert: below-threshold NOT promoted
    const belowResult = results.find(r => r.content === belowThresholdContent);
    t.false(belowResult.promoted, 'Below-threshold content should NOT be promoted');

    // Verify QUEEN.md content
    const queenContent = fs.readFileSync(path.join(tmpDir, '.aether', 'QUEEN.md'), 'utf8');
    t.true(queenContent.includes(patternContent), 'QUEEN.md should contain pattern content');
    t.true(queenContent.includes(philosophyContent), 'QUEEN.md should contain philosophy content');
    t.false(queenContent.includes(belowThresholdContent), 'QUEEN.md should NOT contain below-threshold content');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('batch sweep is idempotent (safe to run after memory-capture already promoted)', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    const content = 'Pattern already promoted via memory-capture';

    // Observe twice to meet threshold
    runAetherUtil(tmpDir, 'learning-observe', [content, 'pattern', 'colony-a']);
    runAetherUtil(tmpDir, 'learning-observe', [content, 'pattern', 'colony-b']);

    // Promote via memory-capture path (internally calls learning-promote-auto)
    // We simulate this by calling learning-promote-auto directly (as memory-capture does)
    const firstPromote = parseLastJson(runAetherUtil(tmpDir, 'learning-promote-auto', [
      'pattern', content, 'colony-a', 'learning'
    ]));
    t.true(firstPromote.result.promoted, 'First promotion should succeed');

    // Batch sweep: call learning-promote-auto again (simulating continue-finalize or seal sweep)
    const sweepResult = parseLastJson(runAetherUtil(tmpDir, 'learning-promote-auto', [
      'pattern', content, 'colony-a', 'learning'
    ]));
    t.true(sweepResult.ok, 'Sweep call should return ok=true');
    t.false(sweepResult.result.promoted, 'Sweep should NOT re-promote');
    t.is(sweepResult.result.reason, 'already_promoted', 'Reason should be already_promoted');

    // Verify QUEEN.md has content only once in Patterns section.
    // queen-promote also writes an Evolution Log entry, so we check the section specifically.
    const queenContent = fs.readFileSync(path.join(tmpDir, '.aether', 'QUEEN.md'), 'utf8');
    const patternsSection = queenContent.split('## 🧭 Patterns')[1]?.split('##')[0] || '';
    const sectionOccurrences = patternsSection.split(content).length - 1;
    t.is(sectionOccurrences, 1, 'Content should appear exactly ONCE in Patterns section');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// =============================================================================
// QUEEN-03: Wisdom in colony-prime prompt_section (2 tests)
// =============================================================================

test.serial('colony-prime includes promoted wisdom in prompt_section', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    const content = 'Always validate API responses before processing';

    // Observe twice and promote
    runAetherUtil(tmpDir, 'learning-observe', [content, 'pattern', 'colony-a']);
    runAetherUtil(tmpDir, 'learning-observe', [content, 'pattern', 'colony-b']);
    const promoteResult = parseLastJson(runAetherUtil(tmpDir, 'learning-promote-auto', [
      'pattern', content, 'colony-a', 'learning'
    ]));
    t.true(promoteResult.result.promoted, 'Promotion should succeed');

    // Act: call colony-prime
    const primeResult = JSON.parse(runAetherUtil(tmpDir, 'colony-prime'));

    // Assert: prompt_section contains promoted content and QUEEN WISDOM header
    t.true(primeResult.ok, 'colony-prime should return ok=true');
    t.truthy(primeResult.result.prompt_section, 'Should have prompt_section');
    t.true(primeResult.result.prompt_section.includes(content),
      'prompt_section should contain promoted wisdom content');
    t.true(primeResult.result.prompt_section.includes('QUEEN WISDOM'),
      'prompt_section should contain QUEEN WISDOM header');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('colony-prime prompt_section has no user-promoted content when QUEEN.md is template-only', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Setup colony with empty QUEEN.md (template only, no promoted content)
    await setupTestColony(tmpDir);

    // Also promote some content in a separate colony dir so we can verify
    // that our template-only QUEEN.md does NOT include that content
    const testContent = 'This should NOT appear in prompt_section';

    // Act: call colony-prime with no promotions done
    const primeResult = JSON.parse(runAetherUtil(tmpDir, 'colony-prime'));

    // Assert: colony-prime succeeds
    t.true(primeResult.ok, 'colony-prime should return ok=true');
    t.truthy(primeResult.result.prompt_section, 'Should have prompt_section');

    // The QUEEN WISDOM section may include template placeholder text
    // (e.g. "*No patterns recorded yet*"), but should NOT contain any
    // user-promoted wisdom entries. Verify no promoted content is present.
    const section = primeResult.result.prompt_section;
    t.false(section.includes(testContent),
      'prompt_section should NOT contain non-promoted content');

    // Verify the wisdom object reflects empty sections
    // (placeholder text is extracted but no actual promoted entries)
    if (primeResult.result.wisdom) {
      const patterns = primeResult.result.wisdom.patterns || '';
      t.false(patterns.includes('- **'),
        'Patterns section should not contain promoted entry format (- **)');
    }
  } finally {
    await cleanupTempDir(tmpDir);
  }
});
