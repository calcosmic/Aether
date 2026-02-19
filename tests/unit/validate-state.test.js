const test = require('ava');
const { execSync } = require('child_process');
const path = require('path');
const fs = require('fs');
const os = require('os');

const AETHER_UTILS_PATH = path.join(__dirname, '../../.aether/aether-utils.sh');
const REAL_DATA_DIR = path.join(__dirname, '../../.aether/data');

/**
 * Module-level snapshot of colony data files.
 *
 * Created at require() time (before AVA runs any tests) so concurrent test
 * files (e.g. state-loader.test.js) cannot corrupt the snapshot by temporarily
 * writing invalid JSON to .aether/data/COLONY_STATE.json during their own tests.
 *
 * Each test that needs data files creates its own tmpDir and COPIES from this
 * snapshot — isolating it completely from any parallel file mutations.
 */
const SNAPSHOT_DIR = fs.mkdtempSync(path.join(os.tmpdir(), 'aether-vs-snapshot-'));
fs.copyFileSync(
  path.join(REAL_DATA_DIR, 'COLONY_STATE.json'),
  path.join(SNAPSHOT_DIR, 'COLONY_STATE.json')
);
fs.copyFileSync(
  path.join(REAL_DATA_DIR, 'constraints.json'),
  path.join(SNAPSHOT_DIR, 'constraints.json')
);

/**
 * Helper to execute aether-utils.sh subcommand and parse JSON output
 * @param {string} args - Arguments to pass to the script
 * @returns {object} - Parsed JSON result
 */
function runUtilsCommand(args) {
  const cmd = `bash "${AETHER_UTILS_PATH}" ${args}`;
  const output = execSync(cmd, { encoding: 'utf8', cwd: path.join(__dirname, '../..') });
  return JSON.parse(output);
}

/**
 * Helper to execute aether-utils.sh subcommand with an isolated DATA_DIR.
 * Prevents AVA parallel-execution races where other tests delete/corrupt COLONY_STATE.json.
 * @param {string} args - Arguments to pass to the script
 * @param {string} dataDir - Path to the isolated temp directory to use as DATA_DIR
 * @returns {object} - Parsed JSON result
 */
function runUtilsCommandIsolated(args, dataDir) {
  const cmd = `bash "${AETHER_UTILS_PATH}" ${args}`;
  const output = execSync(cmd, {
    encoding: 'utf8',
    cwd: path.join(__dirname, '../..'),
    env: { ...process.env, DATA_DIR: dataDir }
  });
  return JSON.parse(output);
}

/**
 * Helper to execute aether-utils.sh subcommand that expects failure
 * @param {string} args - Arguments to pass to the script
 * @returns {object} - Parsed JSON error result
 */
function runUtilsCommandExpectError(args) {
  const cmd = `bash "${AETHER_UTILS_PATH}" ${args}`;
  try {
    execSync(cmd, { encoding: 'utf8', cwd: path.join(__dirname, '../..') });
    throw new Error('Expected command to fail');
  } catch (error) {
    if (error.status !== 0) {
      // Command failed as expected — parse last JSON line from stderr
      const lines = (error.stderr || '').trim().split('\n');
      for (let i = lines.length - 1; i >= 0; i--) {
        try { return JSON.parse(lines[i]); } catch (e) { continue; }
      }
      return { ok: false, error: { message: error.stderr || error.message } };
    }
    throw error;
  }
}

/**
 * Create a per-test isolated temp directory pre-populated from the module-level snapshot.
 * @param {string[]} files - File names to copy (e.g. ['COLONY_STATE.json', 'constraints.json'])
 * @returns {string} - Path to the temp directory
 */
function createIsolatedDataDir(files) {
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'aether-vs-'));
  for (const file of files) {
    fs.copyFileSync(path.join(SNAPSHOT_DIR, file), path.join(tmpDir, file));
  }
  return tmpDir;
}

// Test: validate-state colony returns valid JSON
test('validate-state colony returns valid JSON with correct structure', t => {
  const tmpDir = createIsolatedDataDir(['COLONY_STATE.json']);
  t.teardown(() => fs.rmSync(tmpDir, { recursive: true, force: true }));

  const result = runUtilsCommandIsolated('validate-state colony', tmpDir);

  t.true('ok' in result, 'Result should have ok field');
  t.true('result' in result, 'Result should have result field');
  t.true(result.ok, 'ok should be true for valid state');

  const validation = result.result;
  t.is(typeof validation, 'object', 'result should be an object');
  t.is(validation.file, 'COLONY_STATE.json', 'file should be COLONY_STATE.json');
  t.true('checks' in validation, 'result should have checks array');
  t.true(Array.isArray(validation.checks), 'checks should be an array');
  t.true('pass' in validation, 'result should have pass field');
  t.is(typeof validation.pass, 'boolean', 'pass should be a boolean');
});

// Test: validate-state colony checks include pass/fail status
test('validate-state colony checks have pass/fail status', t => {
  const tmpDir = createIsolatedDataDir(['COLONY_STATE.json']);
  t.teardown(() => fs.rmSync(tmpDir, { recursive: true, force: true }));

  const result = runUtilsCommandIsolated('validate-state colony', tmpDir);
  const validation = result.result;

  t.true(validation.checks.length > 0, 'Should have at least one check');

  for (const check of validation.checks) {
    t.true(
      check === 'pass' || typeof check === 'string' && check.startsWith('fail:'),
      `Each check should be 'pass' or start with 'fail:', got: ${check}`
    );
  }
});

// Test: validate-state colony checks specific fields
test('validate-state colony validates required fields', t => {
  const tmpDir = createIsolatedDataDir(['COLONY_STATE.json']);
  t.teardown(() => fs.rmSync(tmpDir, { recursive: true, force: true }));

  const result = runUtilsCommandIsolated('validate-state colony', tmpDir);
  const validation = result.result;

  // Count passes and fails
  const passCount = validation.checks.filter(c => c === 'pass').length;
  const failCount = validation.checks.filter(c => c.startsWith && c.startsWith('fail:')).length;

  t.true(passCount > 0, 'Should have at least some passing checks');
  t.is(validation.pass, failCount === 0, 'pass should be true only when no failures');
});

// Test: validate-state constraints returns valid JSON
test('validate-state constraints returns valid JSON with correct structure', t => {
  const tmpDir = createIsolatedDataDir(['constraints.json']);
  t.teardown(() => fs.rmSync(tmpDir, { recursive: true, force: true }));

  const result = runUtilsCommandIsolated('validate-state constraints', tmpDir);

  t.true('ok' in result, 'Result should have ok field');
  t.true('result' in result, 'Result should have result field');
  t.true(result.ok, 'ok should be true for valid constraints');

  const validation = result.result;
  t.is(typeof validation, 'object', 'result should be an object');
  t.is(validation.file, 'constraints.json', 'file should be constraints.json');
  t.true('checks' in validation, 'result should have checks array');
  t.true(Array.isArray(validation.checks), 'checks should be an array');
  t.true('pass' in validation, 'result should have pass field');
  t.is(typeof validation.pass, 'boolean', 'pass should be a boolean');
});

// Test: validate-state constraints validates array fields
test('validate-state constraints validates array fields', t => {
  const tmpDir = createIsolatedDataDir(['constraints.json']);
  t.teardown(() => fs.rmSync(tmpDir, { recursive: true, force: true }));

  const result = runUtilsCommandIsolated('validate-state constraints', tmpDir);
  const validation = result.result;

  t.true(validation.checks.length >= 2, 'Should check at least focus and constraints arrays');

  for (const check of validation.checks) {
    t.true(
      check === 'pass' || typeof check === 'string' && check.startsWith('fail:'),
      `Each check should be 'pass' or start with 'fail:', got: ${check}`
    );
  }
});

// Test: validate-state all returns combined results
test('validate-state all returns combined validation results', t => {
  const tmpDir = createIsolatedDataDir(['COLONY_STATE.json', 'constraints.json']);
  t.teardown(() => fs.rmSync(tmpDir, { recursive: true, force: true }));

  const result = runUtilsCommandIsolated('validate-state all', tmpDir);

  t.true('ok' in result, 'Result should have ok field');
  t.true('result' in result, 'Result should have result field');
  t.true(result.ok, 'ok should be true when all validations pass');

  const validation = result.result;
  t.true('pass' in validation, 'result should have pass field');
  t.is(typeof validation.pass, 'boolean', 'pass should be a boolean');
  t.true('files' in validation, 'result should have files array');
  t.true(Array.isArray(validation.files), 'files should be an array');
  t.is(validation.files.length, 2, 'Should have results for 2 files (colony and constraints)');
});

// Test: validate-state all files have required structure
test('validate-state all files have required structure', t => {
  const tmpDir = createIsolatedDataDir(['COLONY_STATE.json', 'constraints.json']);
  t.teardown(() => fs.rmSync(tmpDir, { recursive: true, force: true }));

  const result = runUtilsCommandIsolated('validate-state all', tmpDir);
  const validation = result.result;

  for (const file of validation.files) {
    t.true('file' in file, 'Each file result should have file field');
    t.true('pass' in file, 'Each file result should have pass field');
    t.is(typeof file.pass, 'boolean', 'file.pass should be a boolean');
  }

  // Verify both expected files are present
  const fileNames = validation.files.map(f => f.file);
  t.true(fileNames.includes('COLONY_STATE.json'), 'Should include COLONY_STATE.json');
  t.true(fileNames.includes('constraints.json'), 'Should include constraints.json');
});

// Test: validate-state with invalid target returns error
// No colony files needed — error fires before any file access
test('validate-state with invalid target returns error', t => {
  const error = runUtilsCommandExpectError('validate-state invalid-target');

  t.false(error.ok, 'ok should be false for error');
  t.true('error' in error, 'Error should have error field');
  t.true(error.error.message.includes('Usage:'), 'Error should include usage information');
  t.is(error.error.code, 'E_VALIDATION_FAILED', 'Error code should be E_VALIDATION_FAILED');
});

// Test: validate-state without argument returns error
// No colony files needed — error fires before any file access
test('validate-state without argument returns error', t => {
  const error = runUtilsCommandExpectError('validate-state');

  t.false(error.ok, 'ok should be false for error');
  t.true('error' in error, 'Error should have error field');
  t.true(error.error.message.includes('Usage:'), 'Error should include usage information');
  t.is(error.error.code, 'E_VALIDATION_FAILED', 'Error code should be E_VALIDATION_FAILED');
});

// Test: All validate-state subcommands return consistent JSON format
test('all validate-state subcommands return consistent JSON format', t => {
  const tmpDir = createIsolatedDataDir(['COLONY_STATE.json', 'constraints.json']);
  t.teardown(() => fs.rmSync(tmpDir, { recursive: true, force: true }));

  const colonyResult = runUtilsCommandIsolated('validate-state colony', tmpDir);
  const constraintsResult = runUtilsCommandIsolated('validate-state constraints', tmpDir);
  const allResult = runUtilsCommandIsolated('validate-state all', tmpDir);

  // All should have ok and result fields
  t.true('ok' in colonyResult);
  t.true('ok' in constraintsResult);
  t.true('ok' in allResult);
  t.true('result' in colonyResult);
  t.true('result' in constraintsResult);
  t.true('result' in allResult);

  // All ok values should be true (since files are valid)
  t.true(colonyResult.ok);
  t.true(constraintsResult.ok);
  t.true(allResult.ok);
});

// Test: validate-state colony handles optional fields correctly
test('validate-state colony handles optional fields', t => {
  const tmpDir = createIsolatedDataDir(['COLONY_STATE.json']);
  t.teardown(() => fs.rmSync(tmpDir, { recursive: true, force: true }));

  const result = runUtilsCommandIsolated('validate-state colony', tmpDir);
  const validation = result.result;

  // Check that optional fields are validated (they should be 'pass' if present or not required)
  const checks = validation.checks;
  const optionalChecks = checks.filter(c =>
    typeof c === 'string' && (
      c.includes('session_id') ||
      c.includes('initialized_at') ||
      c.includes('build_started_at')
    )
  );

  // Optional fields should either pass or not be checked
  t.true(
    optionalChecks.every(c => c === 'pass'),
    'Optional field checks should pass if present'
  );
});
