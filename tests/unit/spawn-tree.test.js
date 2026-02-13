const test = require('ava');
const { execSync } = require('child_process');
const path = require('path');
const fs = require('fs');

const AETHER_ROOT = path.join(__dirname, '../..');
const AETHER_UTILS = path.join(AETHER_ROOT, '.aether/aether-utils.sh');
const SPAWN_TREE_FILE = path.join(AETHER_ROOT, '.aether/data/spawn-tree.txt');

/**
 * Helper to run aether-utils.sh commands and parse JSON output
 */
function runUtilsCommand(command, args = []) {
  const cmd = `bash "${AETHER_UTILS}" ${command} ${args.join(' ')}`;
  const output = execSync(cmd, {
    cwd: AETHER_ROOT,
    encoding: 'utf8',
    timeout: 10000
  });
  return JSON.parse(output);
}

// Test: spawn-tree-load returns valid tree JSON
test('spawn-tree-load returns valid tree JSON', t => {
  const result = runUtilsCommand('spawn-tree-load');

  t.true(result.ok, 'Should return ok: true');
  t.truthy(result.result, 'Should have result');
  t.truthy(result.result.spawns, 'Should have spawns array');
  t.truthy(result.result.metadata, 'Should have metadata object');
  t.is(typeof result.result.metadata.total_count, 'number', 'metadata.total_count should be a number');
  t.is(typeof result.result.metadata.active_count, 'number', 'metadata.active_count should be a number');
  t.is(typeof result.result.metadata.completed_count, 'number', 'metadata.completed_count should be a number');
});

// Test: spawn-tree-active returns only active spawns
test('spawn-tree-active returns only active spawns', t => {
  const result = runUtilsCommand('spawn-tree-active');

  t.true(result.ok, 'Should return ok: true');
  t.true(Array.isArray(result.result), 'Result should be an array');

  // All returned spawns should be active (not completed/failed)
  for (const spawn of result.result) {
    t.truthy(spawn.name, 'Active spawn should have name');
    t.truthy(spawn.caste, 'Active spawn should have caste');
    t.truthy(spawn.parent, 'Active spawn should have parent');
    t.truthy(spawn.task, 'Active spawn should have task');
    t.truthy(spawn.spawned_at, 'Active spawn should have spawned_at');
  }
});

// Test: spawn-tree-depth returns correct depth for Queen
test('spawn-tree-depth returns 0 for Queen', t => {
  const result = runUtilsCommand('spawn-tree-depth', ['Queen']);

  t.true(result.ok, 'Should return ok: true');
  t.is(result.result.ant, 'Queen', 'Should return ant name Queen');
  t.is(result.result.depth, 0, 'Queen should have depth 0');
});

// Test: spawn-tree-depth returns correct depth for known spawn
test('spawn-tree-depth returns correct depth for known spawn', t => {
  // First get a known spawn from the tree
  const treeResult = runUtilsCommand('spawn-tree-load');

  if (treeResult.result.spawns.length === 0) {
    t.pass('No spawns to test');
    return;
  }

  const knownSpawn = treeResult.result.spawns[0];
  const result = runUtilsCommand('spawn-tree-depth', [knownSpawn.name]);

  t.true(result.ok, 'Should return ok: true');
  t.is(result.result.ant, knownSpawn.name, 'Should return correct ant name');
  t.is(typeof result.result.depth, 'number', 'Depth should be a number');
  t.true(result.result.depth >= 0, 'Depth should be non-negative');
  t.is(result.result.found, true, 'Should indicate spawn was found');
});

// Test: spawn-tree-depth returns depth 1 for unknown ant
test('spawn-tree-depth returns depth 1 for unknown ant', t => {
  const result = runUtilsCommand('spawn-tree-depth', ['NonExistentAnt-999']);

  t.true(result.ok, 'Should return ok: true');
  t.is(result.result.ant, 'NonExistentAnt-999', 'Should return requested ant name');
  t.is(result.result.depth, 1, 'Unknown ant should have default depth 1');
  t.is(result.result.found, false, 'Should indicate spawn was not found');
});

// Test: spawn-tree-load handles missing file gracefully
test('spawn-tree-load handles missing file gracefully', t => {
  // Create a temporary directory without spawn-tree.txt
  const tempDir = fs.mkdtempSync('/tmp/spawn-tree-test-');
  const tempDataDir = path.join(tempDir, 'data');
  fs.mkdirSync(tempDataDir);

  // Run command with custom data dir
  const cmd = `SPAWN_TREE_FILE="${tempDataDir}/nonexistent.txt" bash "${AETHER_UTILS}" spawn-tree-load`;
  const output = execSync(cmd, {
    cwd: AETHER_ROOT,
    encoding: 'utf8',
    timeout: 10000
  });
  const result = JSON.parse(output);

  t.true(result.ok, 'Should return ok: true');
  t.deepEqual(result.result.spawns, [], 'Should return empty spawns array');
  t.is(result.result.metadata.total_count, 0, 'Should show total_count 0');
  t.is(result.result.metadata.file_exists, false, 'Should indicate file does not exist');

  // Cleanup
  fs.rmSync(tempDir, { recursive: true });
});

// Test: spawn-tree-depth handles deep chains correctly
test('spawn-tree-depth handles deep chains correctly', t => {
  // Create a test file with a deep chain
  const tempDir = fs.mkdtempSync('/tmp/spawn-tree-depth-test-');
  const tempSpawnTree = path.join(tempDir, 'spawn-tree.txt');

  // Create a spawn chain
  // Queen -> Level1 -> Level2 -> Level3 (3 levels deep from Queen)
  const testData = [
    '2026-02-13T10:00:00Z|Queen|builder|Level1|Task 1|spawned',
    '2026-02-13T10:01:00Z|Level1|builder|Level2|Task 2|spawned',
    '2026-02-13T10:02:00Z|Level2|builder|Level3|Task 3|spawned'
  ].join('\n');

  fs.writeFileSync(tempSpawnTree, testData);

  // Test depth calculation for each level
  const tests = [
    { ant: 'Level1', expectedDepth: 1 },
    { ant: 'Level2', expectedDepth: 2 },
    { ant: 'Level3', expectedDepth: 3 }
  ];

  for (const testCase of tests) {
    const cmd = `SPAWN_TREE_FILE="${tempSpawnTree}" bash "${AETHER_UTILS}" spawn-tree-depth ${testCase.ant}`;
    const output = execSync(cmd, {
      cwd: AETHER_ROOT,
      encoding: 'utf8',
      timeout: 10000
    });
    const result = JSON.parse(output);

    t.true(result.ok, `Should return ok: true for ${testCase.ant}`);
    t.is(result.result.ant, testCase.ant, `Should return correct ant name for ${testCase.ant}`);
    t.is(result.result.depth, testCase.expectedDepth, `${testCase.ant} should have depth ${testCase.expectedDepth}`);
    t.is(result.result.found, true, `${testCase.ant} should be found`);
  }

  // Cleanup
  fs.rmSync(tempDir, { recursive: true });
});

// Test: spawn-tree-load includes parent-child relationships
test('spawn-tree-load includes parent-child relationships', t => {
  const result = runUtilsCommand('spawn-tree-load');

  t.true(result.ok, 'Should return ok: true');
  t.true(Array.isArray(result.result.spawns), 'Spawns should be an array');

  // Check that children arrays are present
  for (const spawn of result.result.spawns) {
    t.true(Array.isArray(spawn.children), `Spawn ${spawn.name} should have children array`);

    // If this spawn has children, verify they reference valid spawns
    for (const childName of spawn.children) {
      const childExists = result.result.spawns.some(s => s.name === childName);
      t.true(childExists, `Child ${childName} of ${spawn.name} should exist in spawns list`);
    }
  }

  // Verify parent-child relationship consistency
  for (const spawn of result.result.spawns) {
    if (spawn.parent !== 'Queen') {
      const parent = result.result.spawns.find(s => s.name === spawn.parent);
      if (parent) {
        t.true(
          parent.children.includes(spawn.name),
          `Parent ${parent.name} should list ${spawn.name} in its children`
        );
      }
    }
  }
});

// Test: spawn-tree-active returns empty array when no active spawns
test('spawn-tree-active returns empty array when no active spawns', t => {
  // Create a test file with only completed spawns
  const tempDir = fs.mkdtempSync('/tmp/spawn-tree-active-test-');
  const tempSpawnTree = path.join(tempDir, 'spawn-tree.txt');

  const testData = [
    '2026-02-13T10:00:00Z|Queen|builder|Done1|Task 1|spawned',
    '2026-02-13T10:01:00Z|Done1|completed|Completed task',
    '2026-02-13T10:02:00Z|Queen|builder|Done2|Task 2|spawned',
    '2026-02-13T10:03:00Z|Done2|completed|Completed task'
  ].join('\n');

  fs.writeFileSync(tempSpawnTree, testData);

  const cmd = `SPAWN_TREE_FILE="${tempSpawnTree}" bash "${AETHER_UTILS}" spawn-tree-active`;
  const output = execSync(cmd, {
    cwd: AETHER_ROOT,
    encoding: 'utf8',
    timeout: 10000
  });
  const result = JSON.parse(output);

  t.true(result.ok, 'Should return ok: true');
  t.deepEqual(result.result, [], 'Should return empty array when no active spawns');

  // Cleanup
  fs.rmSync(tempDir, { recursive: true });
});
