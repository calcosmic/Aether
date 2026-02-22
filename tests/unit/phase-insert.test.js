#!/usr/bin/env node
/**
 * phase-insert runtime tests
 *
 * Verifies safe phase insertion and renumbering behavior.
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

function createTempDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), 'aether-phase-insert-'));
}

function cleanupTempDir(tempDir) {
  fs.rmSync(tempDir, { recursive: true, force: true });
}

function setupTempAether(tempDir) {
  const repoRoot = path.join(__dirname, '..', '..');
  const srcAetherDir = path.join(repoRoot, '.aether');
  const dstAetherDir = path.join(tempDir, '.aether');
  const dstDataDir = path.join(dstAetherDir, 'data');

  fs.mkdirSync(dstAetherDir, { recursive: true });
  fs.mkdirSync(dstDataDir, { recursive: true });
  fs.copyFileSync(path.join(srcAetherDir, 'aether-utils.sh'), path.join(dstAetherDir, 'aether-utils.sh'));

  const srcUtilsDir = path.join(srcAetherDir, 'utils');
  const dstUtilsDir = path.join(dstAetherDir, 'utils');
  fs.cpSync(srcUtilsDir, dstUtilsDir, { recursive: true });

  const srcExchangeDir = path.join(srcAetherDir, 'exchange');
  const dstExchangeDir = path.join(dstAetherDir, 'exchange');
  if (fs.existsSync(srcExchangeDir)) {
    fs.cpSync(srcExchangeDir, dstExchangeDir, { recursive: true });
  }

  const srcSchemasDir = path.join(srcAetherDir, 'schemas');
  const dstSchemasDir = path.join(dstAetherDir, 'schemas');
  if (fs.existsSync(srcSchemasDir)) {
    fs.cpSync(srcSchemasDir, dstSchemasDir, { recursive: true });
  }
}

function runUtil(tempDir, subcommand, args = []) {
  const env = {
    ...process.env,
    AETHER_ROOT: tempDir,
    DATA_DIR: path.join(tempDir, '.aether', 'data')
  };
  const cmd = `bash .aether/aether-utils.sh ${subcommand} ${args.map((a) => `"${a.replace(/"/g, '\\"')}"`).join(' ')}`;
  return execSync(cmd, {
    cwd: tempDir,
    env,
    encoding: 'utf8',
    stdio: ['pipe', 'pipe', 'pipe']
  });
}

test('phase-insert inserts after current phase and renumbers downstream tasks/dependencies', t => {
  const tempDir = createTempDir();

  try {
    setupTempAether(tempDir);

    const state = {
      version: '3.0',
      goal: 'Build resilient auth flow',
      state: 'READY',
      current_phase: 1,
      session_id: 'session_123_test',
      initialized_at: '2026-02-22T00:00:00Z',
      build_started_at: null,
      plan: {
        generated_at: null,
        confidence: null,
        phases: [
          {
            id: 1,
            name: 'Initial implementation',
            description: 'Build first pass',
            status: 'completed',
            tasks: [
              {
                id: '1.1',
                description: 'Implement auth',
                dependencies: [],
                success_criteria: ['Auth compiles'],
                status: 'completed'
              }
            ],
            success_criteria: ['Phase 1 complete']
          },
          {
            id: 2,
            name: 'Integration',
            description: 'Wire integrations',
            status: 'pending',
            tasks: [
              {
                id: '2.1',
                description: 'Integrate token refresh',
                dependencies: ['2.0'],
                success_criteria: ['Refresh flow works'],
                status: 'pending'
              },
              {
                id: '2.2',
                description: 'Add integration tests',
                depends_on: ['2.1'],
                success_criteria: ['Tests pass'],
                status: 'pending'
              }
            ],
            success_criteria: ['Phase 2 complete']
          }
        ]
      },
      memory: { phase_learnings: [], decisions: [], instincts: [] },
      errors: { records: [], flagged_patterns: [] },
      signals: [],
      graveyards: [],
      events: []
    };

    fs.writeFileSync(path.join(tempDir, '.aether', 'data', 'COLONY_STATE.json'), JSON.stringify(state, null, 2));

    const raw = runUtil(tempDir, 'phase-insert', [
      'Stabilize retry behavior',
      'Fix duplicate retry submissions',
      'Do not change public API signatures'
    ]);
    const out = JSON.parse(raw);

    t.true(out.ok);
    t.true(out.result.inserted);
    t.is(out.result.inserted_phase_id, 2);
    t.is(out.result.after_phase, 1);

    const updated = JSON.parse(fs.readFileSync(path.join(tempDir, '.aether', 'data', 'COLONY_STATE.json'), 'utf8'));
    t.is(updated.plan.phases.length, 3);
    t.deepEqual(updated.plan.phases.map((p) => p.id), [1, 2, 3]);

    const inserted = updated.plan.phases.find((p) => p.id === 2);
    t.truthy(inserted);
    t.is(inserted.name, 'Stabilize retry behavior');
    t.is(inserted.tasks[0].id, '2.1');

    const shifted = updated.plan.phases.find((p) => p.id === 3);
    t.truthy(shifted);
    t.is(shifted.tasks[0].id, '3.1');
    t.deepEqual(shifted.tasks[0].dependencies, ['3.0']);
    t.deepEqual(shifted.tasks[1].depends_on, ['3.1']);

    t.true(updated.events.some((e) => e.includes('phase_inserted')));

    const pherFile = path.join(tempDir, '.aether', 'data', 'pheromones.json');
    t.true(fs.existsSync(pherFile));
    const pher = JSON.parse(fs.readFileSync(pherFile, 'utf8'));
    const types = pher.signals.map((s) => s.type);
    t.true(types.includes('FOCUS'));
    t.true(types.includes('REDIRECT'));
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('phase-insert fails when plan has no phases', t => {
  const tempDir = createTempDir();

  try {
    setupTempAether(tempDir);

    const emptyPlanState = {
      version: '3.0',
      goal: 'Test',
      state: 'READY',
      current_phase: 0,
      session_id: 'session_123_test',
      initialized_at: '2026-02-22T00:00:00Z',
      build_started_at: null,
      plan: { generated_at: null, confidence: null, phases: [] },
      memory: { phase_learnings: [], decisions: [], instincts: [] },
      errors: { records: [], flagged_patterns: [] },
      signals: [],
      graveyards: [],
      events: []
    };

    fs.writeFileSync(path.join(tempDir, '.aether', 'data', 'COLONY_STATE.json'), JSON.stringify(emptyPlanState, null, 2));

    let failed = false;
    try {
      runUtil(tempDir, 'phase-insert', ['Name', 'Goal']);
    } catch (err) {
      failed = true;
      const output = String(err.stdout || err.stderr || '');
      t.true(output.includes('"ok":false'));
    }

    t.true(failed, 'phase-insert should fail when no phases exist');
  } finally {
    cleanupTempDir(tempDir);
  }
});
