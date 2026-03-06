/**
 * Context Expansion Integration Tests
 *
 * End-to-end tests for the context expansion pipeline:
 * CONTEXT.md decisions -> colony-prime -> prompt_section  (CTX-01)
 * flags.json blockers  -> colony-prime -> prompt_section  (CTX-02)
 *
 * These tests verify that CTX-01 and CTX-02 work together correctly:
 * key decisions from CONTEXT.md and unresolved blocker flags reach builder
 * prompts, while missing files produce no errors, empty data produces no
 * sections, resolved/wrong-phase blockers are excluded, and compact mode
 * caps are enforced.
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

// Helper to create temp directory
async function createTempDir() {
  const tmpDir = await fs.promises.mkdtemp(path.join(os.tmpdir(), 'aether-ctx-'));
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
// Extended for context expansion: accepts contextDecisions and blockerFlags options
async function setupTestColony(tmpDir, opts = {}) {
  const aetherDir = path.join(tmpDir, '.aether');
  const dataDir = path.join(aetherDir, 'data');

  // Create directories
  await fs.promises.mkdir(dataDir, { recursive: true });

  // Create QUEEN.md from template (METADATA on single line to avoid awk issues)
  const isoDate = new Date().toISOString();
  const queenTemplate = `# QUEEN.md --- Colony Wisdom

> Last evolved: ${isoDate}
> Colonies contributed: 0
> Wisdom version: 1.0.0

---

## Philosophies

Core beliefs that guide all colony work.

*No philosophies recorded yet*

---

## Patterns

Validated approaches that consistently work.

*No patterns recorded yet*

---

## Redirects

Anti-patterns to avoid.

*No redirects recorded yet*

---

## Stack Wisdom

Technology-specific insights.

*No stack wisdom recorded yet*

---

## Decrees

User-mandated rules.

*No decrees recorded yet*

---

## Evolution Log

| Date | Colony | Change | Details |
|------|--------|--------|---------|

---

<!-- METADATA {"version":"1.0.0","last_evolved":"${isoDate}","colonies_contributed":[],"promotion_thresholds":{"philosophy":1,"pattern":1,"redirect":1,"stack":1,"decree":0},"stats":{"total_philosophies":0,"total_patterns":0,"total_redirects":0,"total_stack_entries":0,"total_decrees":0}} -->`;

  await fs.promises.writeFile(path.join(aetherDir, 'QUEEN.md'), queenTemplate);

  // Create COLONY_STATE.json
  const instincts = opts.instincts || [];
  const phaseLearnings = opts.phaseLearnings || [];
  const currentPhase = opts.currentPhase !== undefined ? opts.currentPhase : 1;

  const colonyState = {
    session_id: 'colony_test',
    goal: 'test',
    state: 'BUILDING',
    current_phase: currentPhase,
    plan: { phases: [] },
    memory: {
      instincts: instincts,
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

  // Create pheromones.json (optionally with signals)
  const signals = opts.pheromoneSignals || [];
  await fs.promises.writeFile(
    path.join(dataDir, 'pheromones.json'),
    JSON.stringify({ signals: signals, version: '1.0.0' }, null, 2)
  );

  // Write CONTEXT.md if contextDecisions provided
  if (opts.contextDecisions !== undefined) {
    let contextMd = `# Aether Colony -- Current Context

## Recent Decisions

| Date | Decision | Rationale | Made By |
|------|----------|-----------|---------|
`;
    for (const d of opts.contextDecisions) {
      contextMd += `| ${d.date} | ${d.decision} | ${d.rationale} | ${d.madeBy} |\n`;
    }
    contextMd += `
---

## Recent Activity

*No recent activity*
`;
    await fs.promises.writeFile(path.join(aetherDir, 'CONTEXT.md'), contextMd);
  }

  // Write flags.json if blockerFlags provided
  if (opts.blockerFlags !== undefined) {
    const flagsData = {
      version: 1,
      flags: opts.blockerFlags
    };
    await fs.promises.writeFile(
      path.join(dataDir, 'flags.json'),
      JSON.stringify(flagsData, null, 2)
    );
  }

  return { aetherDir, dataDir };
}


test.serial('colony-prime includes CONTEXT.md decisions in prompt', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir, {
      currentPhase: 2,
      contextDecisions: [
        { date: '2026-03-06', decision: 'Use awk for parsing', rationale: 'Simpler than regex', madeBy: 'Queen' },
        { date: '2026-03-06', decision: 'Cap at 5 decisions', rationale: 'Prompt budget', madeBy: 'Colony' }
      ]
    });

    const result = runAetherUtil(tmpDir, 'colony-prime');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    const section = resultJson.result.prompt_section;

    // Section header should be present
    t.true(section.includes('KEY DECISIONS'),
      'Should contain KEY DECISIONS header');

    // Both decision texts should appear
    t.true(section.includes('Use awk for parsing'),
      'Should include first decision text');
    t.true(section.includes('Cap at 5 decisions'),
      'Should include second decision text');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('colony-prime includes blocker warnings from flags.json', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir, {
      currentPhase: 2,
      blockerFlags: [{
        id: 'flag_test_001',
        type: 'blocker',
        severity: 'critical',
        title: 'Tests failing on module X',
        description: 'Integration tests for module X return timeout errors',
        source: 'verification',
        phase: 2,
        created_at: '2026-03-06T12:00:00Z',
        resolved_at: null,
        auto_resolve_on: 'build_pass'
      }]
    });

    const result = runAetherUtil(tmpDir, 'colony-prime');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    const section = resultJson.result.prompt_section;

    // Blocker section should be present
    t.true(section.includes('BLOCKER WARNINGS'),
      'Should contain BLOCKER WARNINGS header');

    // Blocker title should appear
    t.true(section.includes('Tests failing on module X'),
      'Should include blocker title');

    // Source prefix should appear
    t.true(section.includes('[source: verification]'),
      'Should include [source: verification] prefix');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('blocker warnings are distinguishable from REDIRECT pheromones', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir, {
      currentPhase: 1,
      pheromoneSignals: [{
        type: 'REDIRECT',
        message: 'Never modify COLONY_STATE.json directly',
        strength: 0.9,
        source: 'user',
        created_at: new Date().toISOString(),
        expires_at: null,
        phase: null,
        auto_decay: false
      }],
      blockerFlags: [{
        id: 'flag_test_002',
        type: 'blocker',
        severity: 'critical',
        title: 'XML utils not integrated',
        description: 'The XML utilities exist as standalone scripts',
        source: 'verification',
        phase: 1,
        created_at: '2026-03-06T12:00:00Z',
        resolved_at: null,
        auto_resolve_on: 'build_pass'
      }]
    });

    const result = runAetherUtil(tmpDir, 'colony-prime');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    const section = resultJson.result.prompt_section;

    // Both sections should exist
    t.true(section.includes('BLOCKER WARNINGS'),
      'Should contain BLOCKER WARNINGS section');
    t.true(section.includes('REDIRECT (HARD CONSTRAINTS'),
      'Should contain REDIRECT section from pheromones');

    // They should be in different positions (separate sections)
    const blockerIdx = section.indexOf('BLOCKER WARNINGS');
    const redirectIdx = section.indexOf('REDIRECT (HARD CONSTRAINTS');
    t.not(blockerIdx, redirectIdx,
      'BLOCKER WARNINGS and REDIRECT should be at different positions');

    // BLOCKER WARNINGS should appear BEFORE ACTIVE SIGNALS section
    const activeSignalsIdx = section.indexOf('ACTIVE SIGNALS');
    t.true(blockerIdx < activeSignalsIdx,
      'BLOCKER WARNINGS should appear before ACTIVE SIGNALS section');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('missing CONTEXT.md produces no error and no section', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Setup colony WITHOUT contextDecisions (no CONTEXT.md file created)
    await setupTestColony(tmpDir, {
      currentPhase: 2
    });

    const result = runAetherUtil(tmpDir, 'colony-prime');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true (no error)');

    const section = resultJson.result.prompt_section;

    // No KEY DECISIONS section should appear
    t.false(section.includes('KEY DECISIONS'),
      'Should NOT contain KEY DECISIONS when CONTEXT.md is missing');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('missing flags.json produces no error and no section', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Setup colony WITHOUT blockerFlags (no flags.json file created)
    await setupTestColony(tmpDir, {
      currentPhase: 2
    });

    const result = runAetherUtil(tmpDir, 'colony-prime');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true (no error)');

    const section = resultJson.result.prompt_section;

    // No BLOCKER WARNINGS section should appear
    t.false(section.includes('BLOCKER WARNINGS'),
      'Should NOT contain BLOCKER WARNINGS when flags.json is missing');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('empty decisions table produces no section', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Setup colony with empty contextDecisions array (CONTEXT.md has table header but no rows)
    await setupTestColony(tmpDir, {
      currentPhase: 2,
      contextDecisions: []
    });

    const result = runAetherUtil(tmpDir, 'colony-prime');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    const section = resultJson.result.prompt_section;

    // No KEY DECISIONS section should appear (table exists but has no data rows)
    t.false(section.includes('KEY DECISIONS'),
      'Should NOT contain KEY DECISIONS when decision table is empty');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('resolved blockers are excluded', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir, {
      currentPhase: 2,
      blockerFlags: [
        {
          id: 'flag_resolved_001',
          type: 'blocker',
          severity: 'critical',
          title: 'Old resolved blocker',
          description: 'This was already fixed',
          source: 'verification',
          phase: 2,
          created_at: '2026-03-05T12:00:00Z',
          resolved_at: '2026-03-06T10:00:00Z',
          auto_resolve_on: 'build_pass'
        },
        {
          id: 'flag_unresolved_001',
          type: 'blocker',
          severity: 'critical',
          title: 'Active unresolved blocker',
          description: 'This still needs fixing',
          source: 'verification',
          phase: 2,
          created_at: '2026-03-06T12:00:00Z',
          resolved_at: null,
          auto_resolve_on: 'build_pass'
        }
      ]
    });

    const result = runAetherUtil(tmpDir, 'colony-prime');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    const section = resultJson.result.prompt_section;

    // Extract just the BLOCKER WARNINGS section to avoid false positives
    // from the context capsule's "Open risks" list (which shows all flags)
    const blockerStart = section.indexOf('BLOCKER WARNINGS');
    const blockerEnd = section.indexOf('END BLOCKER WARNINGS');
    t.true(blockerStart !== -1, 'Should have BLOCKER WARNINGS section');
    t.true(blockerEnd !== -1, 'Should have END BLOCKER WARNINGS marker');

    const blockerSection = section.substring(blockerStart, blockerEnd);

    // Only unresolved blocker should appear in BLOCKER WARNINGS
    t.true(blockerSection.includes('Active unresolved blocker'),
      'BLOCKER WARNINGS should include unresolved blocker title');
    t.false(blockerSection.includes('Old resolved blocker'),
      'BLOCKER WARNINGS should NOT include resolved blocker title');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('blockers from wrong phase are excluded', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir, {
      currentPhase: 2,
      blockerFlags: [
        {
          id: 'flag_wrong_phase',
          type: 'blocker',
          severity: 'critical',
          title: 'Phase 5 blocker should not appear',
          description: 'This is for a different phase',
          source: 'verification',
          phase: 5,
          created_at: '2026-03-06T12:00:00Z',
          resolved_at: null,
          auto_resolve_on: 'build_pass'
        },
        {
          id: 'flag_correct_phase',
          type: 'blocker',
          severity: 'critical',
          title: 'Phase 2 blocker should appear',
          description: 'This is for the current phase',
          source: 'verification',
          phase: 2,
          created_at: '2026-03-06T12:00:00Z',
          resolved_at: null,
          auto_resolve_on: 'build_pass'
        },
        {
          id: 'flag_global',
          type: 'blocker',
          severity: 'critical',
          title: 'Global blocker should appear',
          description: 'This has no phase (global)',
          source: 'chaos',
          phase: null,
          created_at: '2026-03-06T12:00:00Z',
          resolved_at: null,
          auto_resolve_on: null
        }
      ]
    });

    const result = runAetherUtil(tmpDir, 'colony-prime');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    const section = resultJson.result.prompt_section;

    // Extract just the BLOCKER WARNINGS section to avoid false positives
    // from the context capsule's "Open risks" list (which shows all flags)
    const blockerStart = section.indexOf('BLOCKER WARNINGS');
    const blockerEnd = section.indexOf('END BLOCKER WARNINGS');
    t.true(blockerStart !== -1, 'Should have BLOCKER WARNINGS section');
    t.true(blockerEnd !== -1, 'Should have END BLOCKER WARNINGS marker');

    const blockerSection = section.substring(blockerStart, blockerEnd);

    // Phase 2 blocker should appear in BLOCKER WARNINGS
    t.true(blockerSection.includes('Phase 2 blocker should appear'),
      'BLOCKER WARNINGS should include current phase blocker');

    // Phase 5 blocker should NOT appear in BLOCKER WARNINGS
    t.false(blockerSection.includes('Phase 5 blocker should not appear'),
      'BLOCKER WARNINGS should NOT include wrong phase blocker');

    // Global (null phase) blocker should appear in BLOCKER WARNINGS
    t.true(blockerSection.includes('Global blocker should appear'),
      'BLOCKER WARNINGS should include global (null phase) blocker');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('compact mode caps decisions', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Create 6 decisions (more than the compact cap of 3)
    await setupTestColony(tmpDir, {
      currentPhase: 2,
      contextDecisions: [
        { date: '2026-03-01', decision: 'Decision Alpha', rationale: 'Reason A', madeBy: 'Queen' },
        { date: '2026-03-02', decision: 'Decision Beta', rationale: 'Reason B', madeBy: 'Queen' },
        { date: '2026-03-03', decision: 'Decision Gamma', rationale: 'Reason C', madeBy: 'Queen' },
        { date: '2026-03-04', decision: 'Decision Delta', rationale: 'Reason D', madeBy: 'Queen' },
        { date: '2026-03-05', decision: 'Decision Epsilon', rationale: 'Reason E', madeBy: 'Queen' },
        { date: '2026-03-06', decision: 'Decision Zeta', rationale: 'Reason F', madeBy: 'Queen' }
      ]
    });

    // Run with --compact flag
    const result = runAetherUtil(tmpDir, 'colony-prime', ['--compact']);
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    const section = resultJson.result.prompt_section;

    // Extract the KEY DECISIONS section
    const decisionsStart = section.indexOf('KEY DECISIONS');
    const decisionsEnd = section.indexOf('END KEY DECISIONS');
    t.true(decisionsStart !== -1, 'Should have KEY DECISIONS section');
    t.true(decisionsEnd !== -1, 'Should have END KEY DECISIONS marker');

    const decisionsSection = section.substring(decisionsStart, decisionsEnd);

    // Count decision bullet lines (lines starting with "- ")
    const bulletLines = decisionsSection.split('\n').filter(line => line.trimStart().startsWith('- '));
    t.true(bulletLines.length <= 3,
      `Compact mode should cap at 3 decisions, found ${bulletLines.length}`);
    t.true(bulletLines.length > 0,
      'Should have at least 1 decision in compact mode');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('log_line includes decision and blocker counts', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir, {
      currentPhase: 2,
      contextDecisions: [
        { date: '2026-03-06', decision: 'Some decision', rationale: 'Some reason', madeBy: 'Queen' }
      ],
      blockerFlags: [{
        id: 'flag_log_001',
        type: 'blocker',
        severity: 'critical',
        title: 'A blocker for log test',
        description: 'Testing log line counts',
        source: 'verification',
        phase: 2,
        created_at: '2026-03-06T12:00:00Z',
        resolved_at: null,
        auto_resolve_on: 'build_pass'
      }]
    });

    const result = runAetherUtil(tmpDir, 'colony-prime');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    const logLine = resultJson.result.log_line;

    t.true(logLine.includes('decisions'),
      'Log line should mention decisions');
    t.true(logLine.includes('blockers'),
      'Log line should mention blockers');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});
