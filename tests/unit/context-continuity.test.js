#!/usr/bin/env node
/**
 * Context continuity runtime tests
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

function createTempDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), 'aether-context-'));
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
  fs.cpSync(path.join(srcAetherDir, 'utils'), path.join(dstAetherDir, 'utils'), { recursive: true });

  const srcExchangeDir = path.join(srcAetherDir, 'exchange');
  if (fs.existsSync(srcExchangeDir)) {
    fs.cpSync(srcExchangeDir, path.join(dstAetherDir, 'exchange'), { recursive: true });
  }
  const srcSchemasDir = path.join(srcAetherDir, 'schemas');
  if (fs.existsSync(srcSchemasDir)) {
    fs.cpSync(srcSchemasDir, path.join(dstAetherDir, 'schemas'), { recursive: true });
  }
}

function runUtil(tempDir, subcommand, args = []) {
  const env = {
    ...process.env,
    AETHER_ROOT: tempDir,
    DATA_DIR: path.join(tempDir, '.aether', 'data')
  };
  const quoted = args.map((a) => `"${String(a).replace(/"/g, '\\"')}"`).join(' ');
  const cmd = `bash .aether/aether-utils.sh ${subcommand} ${quoted}`;
  const out = execSync(cmd, {
    cwd: tempDir,
    env,
    encoding: 'utf8',
    stdio: ['pipe', 'pipe', 'pipe']
  });
  return JSON.parse(out);
}

function seedState(tempDir) {
  const state = {
    version: '3.0',
    goal: 'Harden auth and retries',
    state: 'READY',
    current_phase: 1,
    session_id: 'session_123_test',
    initialized_at: '2026-02-22T00:00:00Z',
    build_started_at: null,
    plan: {
      generated_at: '2026-02-22T00:00:00Z',
      confidence: 90,
      phases: [
        { id: 1, name: 'Auth API', status: 'in_progress', tasks: [], success_criteria: [] },
        { id: 2, name: 'Retries', status: 'pending', tasks: [], success_criteria: [] }
      ]
    },
    memory: {
      phase_learnings: [],
      instincts: [],
      decisions: [
        { decision: 'Use JWT', timestamp: '2026-02-22T01:00:00Z' },
        { decision: 'Use retries with backoff', timestamp: '2026-02-22T02:00:00Z' }
      ]
    },
    errors: { records: [], flagged_patterns: [] },
    events: [],
    signals: [],
    graveyards: []
  };

  const pheromones = {
    signals: [
      {
        signal_id: 'sig_1',
        type: 'REDIRECT',
        content: { text: 'No synchronous file I/O' },
        strength: 0.9,
        created_at: '2026-02-22T00:00:00Z',
        expires_at: '2026-03-22T00:00:00Z',
        active: true
      },
      {
        signal_id: 'sig_2',
        type: 'FOCUS',
        content: { text: 'Error handling paths' },
        strength: 0.8,
        created_at: '2026-02-22T00:00:00Z',
        expires_at: '2026-03-22T00:00:00Z',
        active: true
      },
      {
        signal_id: 'sig_3',
        type: 'FEEDBACK',
        content: { text: 'Prefer small, testable functions' },
        strength: 0.6,
        created_at: '2026-02-22T00:00:00Z',
        expires_at: '2026-03-22T00:00:00Z',
        active: true
      }
    ]
  };

  const flags = {
    flags: [
      {
        id: 'f1',
        type: 'blocker',
        title: 'Flaky auth integration test',
        resolved: false
      }
    ]
  };

  fs.writeFileSync(path.join(tempDir, '.aether', 'data', 'COLONY_STATE.json'), JSON.stringify(state, null, 2));
  fs.writeFileSync(path.join(tempDir, '.aether', 'data', 'pheromones.json'), JSON.stringify(pheromones, null, 2));
  fs.writeFileSync(path.join(tempDir, '.aether', 'data', 'flags.json'), JSON.stringify(flags, null, 2));
}

test('context-capsule returns compact prompt section with next action', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);
    seedState(tempDir);

    const out = runUtil(tempDir, 'context-capsule', ['--compact', '--json']);
    t.true(out.ok);
    t.true(out.result.exists);
    t.truthy(out.result.prompt_section);
    t.true(out.result.prompt_section.includes('CONTEXT CAPSULE'));
    t.true(out.result.prompt_section.includes('Goal: Harden auth and retries'));
    t.true(out.result.next_action.startsWith('/ant:'));
    t.true(out.result.word_count > 0);
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('pheromone-prime --compact respects max signal limit', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);
    seedState(tempDir);

    const out = runUtil(tempDir, 'pheromone-prime', ['--compact', '--max-signals', '2', '--max-instincts', '1']);
    t.true(out.ok);
    t.true(out.result.signal_count <= 2);
    t.true(out.result.prompt_section.includes('COMPACT SIGNALS'));
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('rolling-summary add/read keeps only last 15 entries', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);

    for (let i = 0; i < 20; i += 1) {
      const addOut = runUtil(tempDir, 'rolling-summary', ['add', 'learning', `entry-${i}`, 'test']);
      t.true(addOut.ok);
    }

    const out = runUtil(tempDir, 'rolling-summary', ['read', '--json']);
    t.true(out.ok);
    t.is(out.result.count, 15);
    t.is(out.result.entries[0].summary, 'entry-5');
    t.is(out.result.entries[14].summary, 'entry-19');
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('memory-capture appends rolling-summary entry', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);
    seedState(tempDir);
    fs.writeFileSync(path.join(tempDir, '.aether', 'QUEEN.md'), '# QUEEN.md\n\n## 📜 Philosophies\n\nx\n\n## 🧭 Patterns\n\nx\n\n## ⚠️ Redirects\n\nx\n\n## 🔧 Stack Wisdom\n\nx\n\n## 🏛️ Decrees\n\nx\n\n## 📊 Evolution Log\n\nx\n<!-- METADATA {\"version\":\"1.0.0\"} -->\n');
    fs.writeFileSync(path.join(tempDir, '.aether', 'data', 'learning-observations.json'), JSON.stringify({ observations: [] }, null, 2));

    const out = runUtil(tempDir, 'memory-capture', ['learning', 'Validated retry strategy', 'pattern', 'worker:test']);
    t.true(out.ok);

    const roll = runUtil(tempDir, 'rolling-summary', ['read', '--json']);
    t.true(roll.ok);
    t.true(roll.result.count >= 1);
    t.true(roll.result.entries.some((e) => e.summary.includes('Validated retry strategy')));
  } finally {
    cleanupTempDir(tempDir);
  }
});
