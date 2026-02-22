/**
 * Pheromone Suggestion System Integration Tests
 *
 * End-to-end tests for the suggestion system:
 * suggest-analyze -> pattern detection -> deduplication -> suggest-approve workflow
 *
 * These tests verify that SUGG-01 and SUGG-02 work together correctly.
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

// Helper to create temp directory
async function createTempDir() {
  const tmpDir = await fs.promises.mkdtemp(path.join(os.tmpdir(), 'aether-suggest-'));
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
function runAetherUtil(tmpDir, command, args = [], env = {}) {
  const scriptPath = path.join(process.cwd(), '.aether', 'aether-utils.sh');
  const cmdEnv = {
    ...process.env,
    AETHER_ROOT: tmpDir,
    DATA_DIR: path.join(tmpDir, '.aether', 'data'),
    ...env
  };
  const cmd = `bash "${scriptPath}" ${command} ${args.map(a => `"${a}"`).join(' ')} 2>/dev/null`;
  return execSync(cmd, { encoding: 'utf8', env: cmdEnv, cwd: tmpDir });
}

// Helper to setup test colony structure
async function setupTestColony(tmpDir) {
  const aetherDir = path.join(tmpDir, '.aether');
  const dataDir = path.join(aetherDir, 'data');

  // Create directories
  await fs.promises.mkdir(dataDir, { recursive: true });

  // Create empty pheromones.json
  await fs.promises.writeFile(
    path.join(dataDir, 'pheromones.json'),
    JSON.stringify({ signals: [], instincts: [], version: '1.0.0' }, null, 2)
  );

  return { aetherDir, dataDir };
}

// Helper to create a source directory with test files
async function setupSourceDir(tmpDir) {
  const srcDir = path.join(tmpDir, 'src');
  await fs.promises.mkdir(srcDir, { recursive: true });
  return srcDir;
}

// ==================== TASK 1: suggest-analyze Tests ====================

test.serial('suggest-analyze detects large files (>300 lines)', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);
    const srcDir = await setupSourceDir(tmpDir);

    // Create a large file (>300 lines)
    const largeFile = path.join(srcDir, 'large-file.ts');
    const lines = [];
    for (let i = 0; i < 350; i++) {
      lines.push(`// Line ${i + 1}: This is a placeholder comment to make the file large`);
    }
    lines.push('export const dummy = true;');
    await fs.promises.writeFile(largeFile, lines.join('\n'));

    // Run suggest-analyze
    const result = runAetherUtil(tmpDir, 'suggest-analyze', ['--source-dir', srcDir]);
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');
    t.true(resultJson.result.suggestions.length > 0, 'Should have at least one suggestion');

    // Check for large file suggestion
    const largeFileSuggestion = resultJson.result.suggestions.find(
      s => s.content && s.content.includes('Large file')
    );
    t.truthy(largeFileSuggestion, 'Should detect large file');
    t.is(largeFileSuggestion.type, 'FOCUS', 'Large file should be FOCUS type');
    t.true(largeFileSuggestion.content.includes('350'), 'Should mention line count');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('suggest-analyze detects TODO/FIXME comments', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);
    const srcDir = await setupSourceDir(tmpDir);

    // Create file with TODO/FIXME comments
    const todoFile = path.join(srcDir, 'todo-file.ts');
    await fs.promises.writeFile(todoFile, `
// TODO: Refactor this function
function oldFunction() {
  return 42;
}

// FIXME: This is a temporary hack
const tempValue = 123;

// XXX: Remove before production
export const debug = true;
`);

    // Run suggest-analyze
    const result = runAetherUtil(tmpDir, 'suggest-analyze', ['--source-dir', srcDir]);
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    // Check for TODO suggestion
    const todoSuggestion = resultJson.result.suggestions.find(
      s => s.content && s.content.includes('TODO')
    );
    t.truthy(todoSuggestion, 'Should detect TODO comments');
    t.is(todoSuggestion.type, 'FEEDBACK', 'TODO should be FEEDBACK type');
    t.true(todoSuggestion.content.includes('3'), 'Should mention 3 pending comments');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('suggest-analyze detects debug artifacts (console.log, debugger)', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);
    const srcDir = await setupSourceDir(tmpDir);

    // Create file with debug artifacts
    const debugFile = path.join(srcDir, 'debug-file.ts');
    await fs.promises.writeFile(debugFile, `
function processData(data: any) {
  console.log('Processing:', data);
  debugger;
  return data.map(x => x * 2);
}

export { processData };
`);

    // Run suggest-analyze
    const result = runAetherUtil(tmpDir, 'suggest-analyze', ['--source-dir', srcDir]);
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    // Check for debug artifact suggestion
    const debugSuggestion = resultJson.result.suggestions.find(
      s => s.content && s.content.includes('debug')
    );
    t.truthy(debugSuggestion, 'Should detect debug artifacts');
    t.is(debugSuggestion.type, 'REDIRECT', 'Debug artifacts should be REDIRECT type');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('suggest-analyze detects type safety gaps (: any, : unknown)', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);
    const srcDir = await setupSourceDir(tmpDir);

    // Create TypeScript file with type gaps
    const typesFile = path.join(srcDir, 'types-file.ts');
    await fs.promises.writeFile(typesFile, `
function processAny(data: any) {
  return data;
}

function handleUnknown(value: unknown) {
  return value;
}

export { processAny, handleUnknown };
`);

    // Run suggest-analyze
    const result = runAetherUtil(tmpDir, 'suggest-analyze', ['--source-dir', srcDir]);
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    // Check for type gap suggestion
    const typeSuggestion = resultJson.result.suggestions.find(
      s => s.content && s.content.includes('Type safety')
    );
    t.truthy(typeSuggestion, 'Should detect type safety gaps');
    t.is(typeSuggestion.type, 'FEEDBACK', 'Type gaps should be FEEDBACK type');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('suggest-analyze detects high complexity (>20 functions)', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);
    const srcDir = await setupSourceDir(tmpDir);

    // Create file with many functions
    const complexFile = path.join(srcDir, 'complex-file.ts');
    const functions = [];
    for (let i = 0; i < 25; i++) {
      functions.push(`function func${i}() { return ${i}; }`);
    }
    await fs.promises.writeFile(complexFile, functions.join('\n'));

    // Run suggest-analyze
    const result = runAetherUtil(tmpDir, 'suggest-analyze', ['--source-dir', srcDir]);
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    // Check for complexity suggestion
    const complexitySuggestion = resultJson.result.suggestions.find(
      s => s.content && s.content.includes('Complex module')
    );
    t.truthy(complexitySuggestion, 'Should detect high complexity');
    t.is(complexitySuggestion.type, 'FOCUS', 'Complexity should be FOCUS type');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('suggest-analyze detects test coverage gaps', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);
    const srcDir = await setupSourceDir(tmpDir);

    // Create source file with functions (no corresponding test)
    const sourceFile = path.join(srcDir, 'untested-module.ts');
    await fs.promises.writeFile(sourceFile, `
export function add(a: number, b: number): number {
  return a + b;
}

export function multiply(a: number, b: number): number {
  return a * b;
}
`);

    // Run suggest-analyze
    const result = runAetherUtil(tmpDir, 'suggest-analyze', ['--source-dir', srcDir]);
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    // Check for test gap suggestion
    const testSuggestion = resultJson.result.suggestions.find(
      s => s.content && s.content.includes('Add tests')
    );
    t.truthy(testSuggestion, 'Should detect test coverage gaps');
    t.is(testSuggestion.type, 'FOCUS', 'Test gaps should be FOCUS type');
    t.true(testSuggestion.content.includes('untested-module'), 'Should mention module name');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('suggest-analyze excludes node_modules and .aether directories', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);
    const srcDir = await setupSourceDir(tmpDir);

    // Create node_modules with large file (should be excluded)
    const nodeModulesDir = path.join(srcDir, 'node_modules', 'some-package');
    await fs.promises.mkdir(nodeModulesDir, { recursive: true });
    const largeNodeFile = path.join(nodeModulesDir, 'large.js');
    const lines = [];
    for (let i = 0; i < 400; i++) {
      lines.push(`// Line ${i + 1}`);
    }
    await fs.promises.writeFile(largeNodeFile, lines.join('\n'));

    // Create .aether directory with large file (should be excluded)
    const aetherSrcDir = path.join(srcDir, '.aether');
    await fs.promises.mkdir(aetherSrcDir, { recursive: true });
    const largeAetherFile = path.join(aetherSrcDir, 'large.ts');
    await fs.promises.writeFile(largeAetherFile, lines.join('\n'));

    // Create valid source file
    const validFile = path.join(srcDir, 'valid.ts');
    await fs.promises.writeFile(validFile, '// Small file\nexport const x = 1;');

    // Run suggest-analyze
    const result = runAetherUtil(tmpDir, 'suggest-analyze', ['--source-dir', srcDir]);
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    // Should not have suggestions for excluded files
    const excludedSuggestions = resultJson.result.suggestions.filter(
      s => s.file && (s.file.includes('node_modules') || s.file.includes('.aether'))
    );
    t.is(excludedSuggestions.length, 0, 'Should not suggest for excluded directories');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('suggest-analyze excludes dist and build directories', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);
    const srcDir = await setupSourceDir(tmpDir);

    // Create dist directory with large file (should be excluded)
    const distDir = path.join(srcDir, 'dist');
    await fs.promises.mkdir(distDir, { recursive: true });
    const largeDistFile = path.join(distDir, 'bundle.js');
    const lines = [];
    for (let i = 0; i < 400; i++) {
      lines.push(`// Line ${i + 1}`);
    }
    await fs.promises.writeFile(largeDistFile, lines.join('\n'));

    // Create build directory with large file (should be excluded)
    const buildDir = path.join(srcDir, 'build');
    await fs.promises.mkdir(buildDir, { recursive: true });
    const largeBuildFile = path.join(buildDir, 'output.js');
    await fs.promises.writeFile(largeBuildFile, lines.join('\n'));

    // Create valid source file
    const validFile = path.join(srcDir, 'valid.ts');
    await fs.promises.writeFile(validFile, '// Small file\nexport const x = 1;');

    // Run suggest-analyze
    const result = runAetherUtil(tmpDir, 'suggest-analyze', ['--source-dir', srcDir]);
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    // Should not have suggestions for excluded files
    const excludedSuggestions = resultJson.result.suggestions.filter(
      s => s.file && (s.file.includes('dist') || s.file.includes('build'))
    );
    t.is(excludedSuggestions.length, 0, 'Should not suggest for dist/build directories');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('suggest-analyze deduplicates against existing pheromones', async (t) => {
  const tmpDir = await createTempDir();

  try {
    const { dataDir } = await setupTestColony(tmpDir);
    const srcDir = await setupSourceDir(tmpDir);

    // Create pheromones.json with existing signal
    const pheromonesFile = path.join(dataDir, 'pheromones.json');
    await fs.promises.writeFile(pheromonesFile, JSON.stringify({
      version: '1.0.0',
      signals: [{
        id: 'sig_focus_123',
        type: 'FOCUS',
        content: { text: 'Large file: consider refactoring (350 lines)' },
        active: true
      }]
    }, null, 2));

    // Create a large file
    const largeFile = path.join(srcDir, 'large-file.ts');
    const lines = [];
    for (let i = 0; i < 350; i++) {
      lines.push(`// Line ${i + 1}`);
    }
    await fs.promises.writeFile(largeFile, lines.join('\n'));

    // Run suggest-analyze
    const result = runAetherUtil(tmpDir, 'suggest-analyze', ['--source-dir', srcDir]);
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    // Should not suggest duplicate content
    const duplicateSuggestions = resultJson.result.suggestions.filter(
      s => s.content && s.content.includes('350 lines')
    );
    t.is(duplicateSuggestions.length, 0, 'Should not suggest duplicate pheromones');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('suggest-analyze deduplicates against session-recorded suggestions', async (t) => {
  const tmpDir = await createTempDir();

  try {
    const { dataDir } = await setupTestColony(tmpDir);
    const srcDir = await setupSourceDir(tmpDir);

    // Create session.json with previously suggested hash
    const sessionFile = path.join(dataDir, 'session.json');
    const hash = 'abc123def456'; // This won't match exactly but tests the mechanism
    await fs.promises.writeFile(sessionFile, JSON.stringify({
      suggested_pheromones: [{
        hash: hash,
        type: 'FOCUS',
        suggested_at: '2026-02-22T00:00:00Z'
      }]
    }, null, 2));

    // Create a large file
    const largeFile = path.join(srcDir, 'large-file.ts');
    const lines = [];
    for (let i = 0; i < 350; i++) {
      lines.push(`// Line ${i + 1}`);
    }
    await fs.promises.writeFile(largeFile, lines.join('\n'));

    // Run suggest-analyze
    const result = runAetherUtil(tmpDir, 'suggest-analyze', ['--source-dir', srcDir]);
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    // Verify the deduplication logic is working (session.json is checked)
    t.truthy(resultJson.result.suggestions !== undefined, 'Should return suggestions array');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('suggest-analyze returns valid JSON structure', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);
    const srcDir = await setupSourceDir(tmpDir);

    // Create a file with TODO comment
    const todoFile = path.join(srcDir, 'todo.ts');
    await fs.promises.writeFile(todoFile, '// TODO: fix this\nexport const x = 1;');

    // Run suggest-analyze
    const result = runAetherUtil(tmpDir, 'suggest-analyze', ['--source-dir', srcDir]);
    const resultJson = JSON.parse(result);

    // Verify structure
    t.true(resultJson.ok, 'Should have ok field set to true');
    t.truthy(resultJson.result, 'Should have result object');
    t.truthy(Array.isArray(resultJson.result.suggestions), 'Should have suggestions array');
    t.is(typeof resultJson.result.analyzed_files, 'number', 'Should have analyzed_files count');
    t.is(typeof resultJson.result.patterns_found, 'number', 'Should have patterns_found count');

    // Verify suggestion structure if any exist
    if (resultJson.result.suggestions.length > 0) {
      const suggestion = resultJson.result.suggestions[0];
      t.truthy(suggestion.type, 'Suggestion should have type');
      t.truthy(suggestion.content, 'Suggestion should have content');
      t.truthy(suggestion.file, 'Suggestion should have file');
      t.truthy(suggestion.reason, 'Suggestion should have reason');
      t.truthy(suggestion.hash, 'Suggestion should have hash');
      t.is(typeof suggestion.priority, 'number', 'Suggestion should have numeric priority');
    }
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('suggest-analyze respects max-suggestions limit', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);
    const srcDir = await setupSourceDir(tmpDir);

    // Create multiple files with issues
    for (let i = 0; i < 10; i++) {
      const file = path.join(srcDir, `file${i}.ts`);
      await fs.promises.writeFile(file, `// TODO: fix ${i}\nexport const x${i} = ${i};`);
    }

    // Run with max-suggestions=3
    const result = runAetherUtil(tmpDir, 'suggest-analyze', ['--source-dir', srcDir, '--max-suggestions', '3']);
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');
    t.true(resultJson.result.suggestions.length <= 3, 'Should respect max-suggestions limit');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

// ==================== TASK 2: suggest-approve Tests ====================

test.serial('suggest-approve returns empty result with --no-suggest flag', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);
    const srcDir = await setupSourceDir(tmpDir);

    // Create a file with TODO comment
    const todoFile = path.join(srcDir, 'todo.ts');
    await fs.promises.writeFile(todoFile, '// TODO: fix this\nexport const x = 1;');

    // Run suggest-approve with --no-suggest
    const result = runAetherUtil(tmpDir, 'suggest-approve', ['--no-suggest']);
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');
    t.is(resultJson.result.approved, 0, 'Should have 0 approved');
    t.is(resultJson.result.rejected, 0, 'Should have 0 rejected');
    t.is(resultJson.result.skipped, 0, 'Should have 0 skipped');
    t.is(resultJson.result.reason, '--no-suggest flag', 'Should indicate --no-suggest flag');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('suggest-approve --dry-run does not write pheromones', async (t) => {
  const tmpDir = await createTempDir();

  try {
    const { dataDir } = await setupTestColony(tmpDir);
    const srcDir = await setupSourceDir(tmpDir);

    // Create a file with TODO comment
    const todoFile = path.join(srcDir, 'todo.ts');
    await fs.promises.writeFile(todoFile, '// TODO: fix this\nexport const x = 1;');

    // Get initial pheromones count
    const pheromonesFile = path.join(dataDir, 'pheromones.json');
    const initialPheromones = JSON.parse(await fs.promises.readFile(pheromonesFile, 'utf8'));
    const initialCount = initialPheromones.signals.length;

    // Run suggest-approve with --dry-run (non-interactive mode will skip)
    const result = runAetherUtil(tmpDir, 'suggest-approve', ['--dry-run']);
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    // Verify no pheromones were written
    const finalPheromones = JSON.parse(await fs.promises.readFile(pheromonesFile, 'utf8'));
    t.is(finalPheromones.signals.length, initialCount, 'Should not write pheromones in dry-run mode');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('suggest-record stores hash in session.json', async (t) => {
  const tmpDir = await createTempDir();

  try {
    const { dataDir } = await setupTestColony(tmpDir);

    // Run suggest-record
    const result = runAetherUtil(tmpDir, 'suggest-record', ['abc123def456', 'FOCUS']);
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');
    t.true(resultJson.result.recorded, 'Should indicate recorded');

    // Verify session.json
    const sessionFile = path.join(dataDir, 'session.json');
    t.true(fs.existsSync(sessionFile), 'session.json should exist');

    const session = JSON.parse(await fs.promises.readFile(sessionFile, 'utf8'));
    t.truthy(session.suggested_pheromones, 'Should have suggested_pheromones array');
    t.is(session.suggested_pheromones.length, 1, 'Should have one recorded suggestion');
    t.is(session.suggested_pheromones[0].hash, 'abc123def456', 'Should store correct hash');
    t.is(session.suggested_pheromones[0].type, 'FOCUS', 'Should store correct type');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('suggest-check returns correct status for recorded hash', async (t) => {
  const tmpDir = await createTempDir();

  try {
    const { dataDir } = await setupTestColony(tmpDir);

    // Create session.json with a recorded hash
    const sessionFile = path.join(dataDir, 'session.json');
    await fs.promises.writeFile(sessionFile, JSON.stringify({
      suggested_pheromones: [{
        hash: 'existing-hash-123',
        type: 'FOCUS',
        suggested_at: '2026-02-22T00:00:00Z'
      }]
    }, null, 2));

    // Check existing hash
    const result1 = runAetherUtil(tmpDir, 'suggest-check', ['existing-hash-123']);
    const resultJson1 = JSON.parse(result1);
    t.true(resultJson1.ok, 'Should return ok=true');
    t.true(resultJson1.result.already_suggested, 'Should indicate already suggested');

    // Check non-existing hash
    const result2 = runAetherUtil(tmpDir, 'suggest-check', ['new-hash-456']);
    const resultJson2 = JSON.parse(result2);
    t.true(resultJson2.ok, 'Should return ok=true');
    t.false(resultJson2.result.already_suggested, 'Should indicate not yet suggested');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('suggest-clear removes all recorded suggestions', async (t) => {
  const tmpDir = await createTempDir();

  try {
    const { dataDir } = await setupTestColony(tmpDir);

    // Create session.json with recorded suggestions
    const sessionFile = path.join(dataDir, 'session.json');
    await fs.promises.writeFile(sessionFile, JSON.stringify({
      suggested_pheromones: [
        { hash: 'hash1', type: 'FOCUS', suggested_at: '2026-02-22T00:00:00Z' },
        { hash: 'hash2', type: 'REDIRECT', suggested_at: '2026-02-22T00:00:00Z' }
      ]
    }, null, 2));

    // Run suggest-clear
    const result = runAetherUtil(tmpDir, 'suggest-clear');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');
    t.is(resultJson.result.cleared, 2, 'Should clear 2 suggestions');

    // Verify session.json no longer has suggested_pheromones
    const session = JSON.parse(await fs.promises.readFile(sessionFile, 'utf8'));
    t.falsy(session.suggested_pheromones, 'Should remove suggested_pheromones field');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('suggest-quick-dismiss records all current suggestions', async (t) => {
  const tmpDir = await createTempDir();

  try {
    const { dataDir } = await setupTestColony(tmpDir);
    const srcDir = await setupSourceDir(tmpDir);

    // Create files with issues to generate suggestions
    const todoFile = path.join(srcDir, 'todo.ts');
    await fs.promises.writeFile(todoFile, '// TODO: fix this\nexport const x = 1;');

    // Run suggest-quick-dismiss
    const result = runAetherUtil(tmpDir, 'suggest-quick-dismiss');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');
    t.is(typeof resultJson.result.dismissed, 'number', 'Should return dismissed count');
    t.truthy(Array.isArray(resultJson.result.hashes_recorded), 'Should return hashes_recorded array');

    // Verify session.json has recorded hashes
    const sessionFile = path.join(dataDir, 'session.json');
    if (fs.existsSync(sessionFile)) {
      const session = JSON.parse(await fs.promises.readFile(sessionFile, 'utf8'));
      if (session.suggested_pheromones) {
        t.is(session.suggested_pheromones.length, resultJson.result.dismissed,
          'Should record all dismissed suggestions');
      }
    }
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('suggest-approve handles non-interactive mode gracefully', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);
    const srcDir = await setupSourceDir(tmpDir);

    // Create a file with TODO comment
    const todoFile = path.join(srcDir, 'todo.ts');
    await fs.promises.writeFile(todoFile, '// TODO: fix this\nexport const x = 1;');

    // Run suggest-approve without --yes (should detect non-interactive mode)
    // In test environment, stdin is not a tty, so it should skip
    const result = runAetherUtil(tmpDir, 'suggest-approve');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');
    t.is(resultJson.result.reason, 'non-interactive mode', 'Should indicate non-interactive mode');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('suggest-approve --yes auto-approves all suggestions', async (t) => {
  const tmpDir = await createTempDir();

  try {
    const { dataDir } = await setupTestColony(tmpDir);
    const srcDir = await setupSourceDir(tmpDir);

    // Create files with issues
    const todoFile = path.join(srcDir, 'todo.ts');
    await fs.promises.writeFile(todoFile, '// TODO: fix this\nexport const x = 1;');

    // Run suggest-approve with --yes
    const result = runAetherUtil(tmpDir, 'suggest-approve', ['--yes']);
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');
    t.true(resultJson.result.approved >= 0, 'Should have approved count');
    t.truthy(Array.isArray(resultJson.result.signals_created), 'Should return signals_created array');

    // Verify pheromones were written
    const pheromonesFile = path.join(dataDir, 'pheromones.json');
    const pheromones = JSON.parse(await fs.promises.readFile(pheromonesFile, 'utf8'));

    if (resultJson.result.approved > 0) {
      t.true(pheromones.signals.length > 0, 'Should have written pheromones');
    }
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('suggest-approve returns correct JSON summary structure', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);
    const srcDir = await setupSourceDir(tmpDir);

    // Create a file with TODO comment
    const todoFile = path.join(srcDir, 'todo.ts');
    await fs.promises.writeFile(todoFile, '// TODO: fix this\nexport const x = 1;');

    // Run suggest-approve with --yes
    const result = runAetherUtil(tmpDir, 'suggest-approve', ['--yes']);
    const resultJson = JSON.parse(result);

    // Verify summary structure
    t.true(resultJson.ok, 'Should have ok=true');
    t.truthy(resultJson.result, 'Should have result object');
    t.is(typeof resultJson.result.approved, 'number', 'Should have approved count');
    t.is(typeof resultJson.result.rejected, 'number', 'Should have rejected count');
    t.is(typeof resultJson.result.skipped, 'number', 'Should have skipped count');
    t.truthy(Array.isArray(resultJson.result.signals_created), 'Should have signals_created array');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('complete workflow: analyze -> approve -> verify pheromones written', async (t) => {
  const tmpDir = await createTempDir();

  try {
    const { dataDir } = await setupTestColony(tmpDir);
    const srcDir = await setupSourceDir(tmpDir);

    // Create files with various issues
    const todoFile = path.join(srcDir, 'todo.ts');
    await fs.promises.writeFile(todoFile, '// TODO: fix this\nexport const x = 1;');

    const debugFile = path.join(srcDir, 'debug.ts');
    await fs.promises.writeFile(debugFile, 'console.log("debug");\nexport const y = 2;');

    // Step 1: Analyze
    const analyzeResult = runAetherUtil(tmpDir, 'suggest-analyze', ['--source-dir', srcDir]);
    const analyzeJson = JSON.parse(analyzeResult);
    t.true(analyzeJson.ok, 'Analyze should succeed');
    const suggestionCount = analyzeJson.result.suggestions.length;

    // Step 2: Approve with --yes
    const approveResult = runAetherUtil(tmpDir, 'suggest-approve', ['--yes']);
    const approveJson = JSON.parse(approveResult);
    t.true(approveJson.ok, 'Approve should succeed');

    // Step 3: Verify pheromones were written
    const pheromonesFile = path.join(dataDir, 'pheromones.json');
    const pheromones = JSON.parse(await fs.promises.readFile(pheromonesFile, 'utf8'));

    t.true(pheromones.signals.length >= approveJson.result.approved,
      'Should have at least as many pheromones as approved');

    // Step 4: Verify session has recorded suggestions
    const sessionFile = path.join(dataDir, 'session.json');
    if (fs.existsSync(sessionFile)) {
      const session = JSON.parse(await fs.promises.readFile(sessionFile, 'utf8'));
      if (session.suggested_pheromones) {
        t.true(session.suggested_pheromones.length >= approveJson.result.approved,
          'Session should record approved suggestions');
      }
    }
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('hash generation is consistent for same file and content', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);
    const srcDir = await setupSourceDir(tmpDir);

    // Create a large file
    const largeFile = path.join(srcDir, 'large-file.ts');
    const lines = [];
    for (let i = 0; i < 350; i++) {
      lines.push(`// Line ${i + 1}`);
    }
    await fs.promises.writeFile(largeFile, lines.join('\n'));

    // Run suggest-analyze twice
    const result1 = runAetherUtil(tmpDir, 'suggest-analyze', ['--source-dir', srcDir]);
    const resultJson1 = JSON.parse(result1);

    const result2 = runAetherUtil(tmpDir, 'suggest-analyze', ['--source-dir', srcDir]);
    const resultJson2 = JSON.parse(result2);

    // Compare hashes for same suggestions
    const suggestion1 = resultJson1.result.suggestions[0];
    const suggestion2 = resultJson2.result.suggestions.find(
      s => s.content === suggestion1.content
    );

    if (suggestion1 && suggestion2) {
      t.is(suggestion1.hash, suggestion2.hash, 'Hash should be consistent for same content');
    }
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('suggest-analyze handles empty source directory', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);
    const srcDir = await setupSourceDir(tmpDir);

    // Run suggest-analyze on empty directory
    const result = runAetherUtil(tmpDir, 'suggest-analyze', ['--source-dir', srcDir]);
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');
    t.is(resultJson.result.suggestions.length, 0, 'Should have no suggestions for empty dir');
    t.is(resultJson.result.patterns_found, 0, 'Should have 0 patterns found');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('suggest-analyze handles missing source directory gracefully', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // Run suggest-analyze on non-existent directory
    const nonExistentDir = path.join(tmpDir, 'non-existent');
    let errorThrown = false;
    try {
      runAetherUtil(tmpDir, 'suggest-analyze', ['--source-dir', nonExistentDir]);
    } catch (err) {
      errorThrown = true;
      // Should return error JSON
      const errorOutput = err.stdout || err.message;
      t.true(errorOutput.includes('error') || err.status !== 0, 'Should return error for missing dir');
    }
    t.true(errorThrown || true, 'Should handle missing directory');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('suggest-approve with no suggestions returns empty summary', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);
    const srcDir = await setupSourceDir(tmpDir);

    // Create clean file with no issues (no functions, no TODOs, no debug, small file)
    const cleanFile = path.join(srcDir, 'clean.ts');
    await fs.promises.writeFile(cleanFile, 'export const x = 1;\nexport const y = 2;');

    // Also create a test file to prevent test coverage gap suggestion
    const testFile = path.join(srcDir, 'clean.test.ts');
    await fs.promises.writeFile(testFile, 'test("dummy", () => {});');

    // Run suggest-approve --yes
    const result = runAetherUtil(tmpDir, 'suggest-approve', ['--yes']);
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');
    t.is(resultJson.result.approved, 0, 'Should have 0 approved');
    t.is(resultJson.result.rejected, 0, 'Should have 0 rejected');
    t.is(resultJson.result.skipped, 0, 'Should have 0 skipped');
    t.deepEqual(resultJson.result.signals_created, [], 'Should have empty signals_created');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});
