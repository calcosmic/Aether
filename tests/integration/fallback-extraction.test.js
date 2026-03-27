/**
 * Fallback Learning Extraction Integration Tests
 *
 * Tests for the learning-extract-fallback subcommand:
 * - Fires when no learnings exist, produces structured learnings from git diff
 * - Skips trivial changes (whitespace-only, package-lock.json, .aether/data/)
 * - Respects 5-learning cap
 * - Does not fire when learnings already exist
 *
 * Phase 27 Plan 02: PIPE-03
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

// Helper to create temp directory
async function createTempDir() {
  const tmpDir = await fs.promises.mkdtemp(path.join(os.tmpdir(), 'aether-fallback-'));
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

// Helper to setup test colony with COLONY_STATE.json
async function setupTestColony(tmpDir, opts = {}) {
  const aetherDir = path.join(tmpDir, '.aether');
  const dataDir = path.join(aetherDir, 'data');

  // Create directories
  await fs.promises.mkdir(dataDir, { recursive: true });

  // Create COLONY_STATE.json
  const colonyState = {
    session_id: 'colony_test',
    goal: 'test',
    state: 'BUILDING',
    current_phase: opts.currentPhase || 1,
    plan: { phases: [] },
    memory: {
      instincts: opts.instincts || [],
      phase_learnings: opts.phaseLearnings || [],
      decisions: []
    },
    errors: { flagged_patterns: [] },
    events: []
  };
  await fs.promises.writeFile(
    path.join(dataDir, 'COLONY_STATE.json'),
    JSON.stringify(colonyState, null, 2)
  );
}

// Helper to initialize git repo and create commits
function gitInit(tmpDir) {
  const env = {
    ...process.env,
    GIT_AUTHOR_NAME: 'Test',
    GIT_AUTHOR_EMAIL: 'test@test.com',
    GIT_COMMITTER_NAME: 'Test',
    GIT_COMMITTER_EMAIL: 'test@test.com'
  };

  execSync('git init', { cwd: tmpDir, env });
  execSync('git checkout -b main', { cwd: tmpDir, env });
}

function gitCommit(tmpDir, message) {
  const env = {
    ...process.env,
    GIT_AUTHOR_NAME: 'Test',
    GIT_AUTHOR_EMAIL: 'test@test.com',
    GIT_COMMITTER_NAME: 'Test',
    GIT_COMMITTER_EMAIL: 'test@test.com'
  };

  execSync('git add -A', { cwd: tmpDir, env });
  execSync(`git commit -m "${message}" --allow-empty`, { cwd: tmpDir, env });
}

// Helper to write a file in the temp dir
async function writeFile(tmpDir, relativePath, content) {
  const fullPath = path.join(tmpDir, relativePath);
  await fs.promises.mkdir(path.dirname(fullPath), { recursive: true });
  await fs.promises.writeFile(fullPath, content);
}


test.serial('fallback fires when no learnings exist', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Setup: git repo with colony, initial commit, then source changes
    await setupTestColony(tmpDir);
    gitInit(tmpDir);

    // Initial commit with a file
    await writeFile(tmpDir, 'src/index.js', 'console.log("hello");');
    gitCommit(tmpDir, 'initial commit');

    // Second commit with meaningful source changes
    await writeFile(tmpDir, 'src/index.js', 'console.log("hello");\nconsole.log("world");\nfunction add(a, b) { return a + b; }\n');
    await writeFile(tmpDir, 'src/utils.js', 'export function helper() { return 42; }\n');
    gitCommit(tmpDir, 'add source files');

    // Run fallback
    const result = runAetherUtil(tmpDir, 'learning-extract-fallback');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');
    t.true(resultJson.result.count > 0, 'Should produce at least 1 learning');
    t.true(resultJson.result.count <= 5, 'Should cap at 5 learnings');

    // Verify each learning has required fields
    for (const learning of resultJson.result.learnings) {
      t.truthy(learning.trigger, 'Learning should have trigger');
      t.truthy(learning.action, 'Learning should have action');
      t.truthy(learning.fact, 'Learning should have fact');
      t.truthy(learning.interpretation, 'Learning should have interpretation');
    }

    // Verify instincts were created in COLONY_STATE.json
    const stateFile = path.join(tmpDir, '.aether', 'data', 'COLONY_STATE.json');
    const state = JSON.parse(fs.readFileSync(stateFile, 'utf8'));
    t.true(state.memory.instincts.length > 0, 'Should have created instincts in colony state');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('fallback skips trivial changes', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Setup: git repo with colony, initial commit, then only trivial changes
    await setupTestColony(tmpDir);
    gitInit(tmpDir);

    // Initial commit
    await writeFile(tmpDir, 'src/index.js', 'console.log("hello");');
    gitCommit(tmpDir, 'initial commit');

    // Second commit: only whitespace changes and package-lock.json
    await writeFile(tmpDir, 'src/index.js', 'console.log("hello"); \n');  // whitespace only
    await writeFile(tmpDir, 'package-lock.json', JSON.stringify({ name: 'test', lockfileVersion: 3 }));
    await writeFile(tmpDir, '.aether/data/COLONY_STATE.json', '{}');  // internal state
    gitCommit(tmpDir, 'trivial changes');

    // Run fallback
    const result = runAetherUtil(tmpDir, 'learning-extract-fallback');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');
    t.is(resultJson.result.count, 0, 'Should produce 0 learnings for trivial changes');
    t.deepEqual(resultJson.result.learnings, [], 'Learnings array should be empty');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('fallback respects 5-learning cap', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Setup: git repo with colony, initial commit, then changes in many categories
    await setupTestColony(tmpDir);
    gitInit(tmpDir);

    // Initial commit
    await writeFile(tmpDir, 'src/index.js', 'old');
    gitCommit(tmpDir, 'initial commit');

    // Second commit: changes across 10+ files in different categories
    await writeFile(tmpDir, 'src/index.js', 'new source code here\n'.repeat(50));
    await writeFile(tmpDir, 'src/utils.js', 'new utility code\n'.repeat(30));
    await writeFile(tmpDir, 'src/api.js', 'new api code\n'.repeat(20));
    await writeFile(tmpDir, 'tests/unit/foo.test.js', 'new test code\n'.repeat(15));
    await writeFile(tmpDir, 'tests/integration/bar.test.js', 'more test code\n'.repeat(10));
    await writeFile(tmpDir, 'docs/guide.md', '# New documentation\n\nLots of docs\n'.repeat(5));
    await writeFile(tmpDir, 'docs/api.md', '# API docs\n\nMore docs\n'.repeat(5));
    await writeFile(tmpDir, 'config.json', '{"key": "new value"}');
    await writeFile(tmpDir, '.env.example', 'NEW_VAR=value\n');
    await writeFile(tmpDir, 'README.md', '# Updated readme\n'.repeat(10));
    gitCommit(tmpDir, 'massive changes');

    // Run fallback
    const result = runAetherUtil(tmpDir, 'learning-extract-fallback');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');
    t.true(resultJson.result.count <= 5, 'Should cap at 5 learnings (got ' + resultJson.result.count + ')');
    t.is(resultJson.result.learnings.length, resultJson.result.count, 'Learnings array length should match count');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('fallback always includes test file additions', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Setup: git repo with colony, initial commit, then only small test file addition
    await setupTestColony(tmpDir);
    gitInit(tmpDir);

    // Initial commit
    await writeFile(tmpDir, 'src/index.js', 'console.log("hello");');
    gitCommit(tmpDir, 'initial commit');

    // Second commit: only a small test file (under 3 lines, normally filtered)
    await writeFile(tmpDir, 'tests/unit/small.test.js', 'const t = require("ava");\nt.pass("hello");\n');
    gitCommit(tmpDir, 'add small test');

    // Run fallback
    const result = runAetherUtil(tmpDir, 'learning-extract-fallback');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');
    t.true(resultJson.result.count > 0, 'Should include test file additions even if small');

    // Verify the learning is categorized as testing
    const hasTesting = resultJson.result.learnings.some(l => l.trigger.includes('testing'));
    t.true(hasTesting, 'Should have a testing-category learning');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('fallback returns empty when no git history', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Setup: colony but no git repo (single commit only)
    await setupTestColony(tmpDir);
    gitInit(tmpDir);

    // Only initial commit -- no HEAD~1 to diff against
    await writeFile(tmpDir, 'src/index.js', 'console.log("hello");');
    gitCommit(tmpDir, 'initial commit');

    // This should return empty because HEAD~1 doesn't exist
    // Actually, we need a second scenario: no git repo at all
    // Let's test with a fresh colony that has no git repo
    const tmpDir2 = await createTempDir();
    try {
      await setupTestColony(tmpDir2);
      // No git init at all

      const result2 = runAetherUtil(tmpDir2, 'learning-extract-fallback');
      const resultJson2 = JSON.parse(result2);

      t.true(resultJson2.ok, 'Should return ok=true even without git');
      t.is(resultJson2.result.count, 0, 'Should produce 0 learnings without git history');
    } finally {
      await cleanupTempDir(tmpDir2);
    }
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('fallback returns empty without colony state', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Setup: git repo but no colony state
    gitInit(tmpDir);
    await writeFile(tmpDir, 'src/index.js', 'old');
    gitCommit(tmpDir, 'initial');
    await writeFile(tmpDir, 'src/index.js', 'new source code\n'.repeat(50));
    gitCommit(tmpDir, 'changes');

    const result = runAetherUtil(tmpDir, 'learning-extract-fallback');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');
    t.is(resultJson.result.count, 0, 'Should produce 0 learnings without colony state');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});
