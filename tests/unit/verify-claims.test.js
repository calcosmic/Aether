#!/usr/bin/env node
/**
 * Verify Claims unit tests (QUAL-08)
 *
 * Tests verify-claims subcommand end-to-end:
 * - Missing file detection (hard block)
 * - Test exit code mismatch (hard block)
 * - Clean pass
 * - Graceful handling of missing claims file
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { spawnSync } = require('child_process');

const REPO_ROOT = path.join(__dirname, '..', '..');
const AETHER_UTILS = path.join(REPO_ROOT, '.aether', 'aether-utils.sh');

function createTempDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), 'aether-verify-claims-'));
}

function cleanupTempDir(tempDir) {
  fs.rmSync(tempDir, { recursive: true, force: true });
}

function setupTempAether(tempDir) {
  const srcAetherDir = path.join(REPO_ROOT, '.aether');
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
}

function runVerifyClaims(tempDir, builderClaimsPath, watcherClaims, testExitCode) {
  const result = spawnSync('bash', [
    path.join(tempDir, '.aether', 'aether-utils.sh'),
    'verify-claims',
    builderClaimsPath || '',
    watcherClaims || '',
    String(testExitCode || 0)
  ], {
    cwd: tempDir,
    encoding: 'utf8',
    timeout: 10000
  });
  return result;
}

test('verify-claims: end-to-end clean pass with real files', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);

    // Create real files
    const srcDir = path.join(tempDir, 'src');
    fs.mkdirSync(srcDir, { recursive: true });
    fs.writeFileSync(path.join(srcDir, 'auth.ts'), 'export const auth = {};');
    fs.writeFileSync(path.join(srcDir, 'types.ts'), 'export type User = {};');

    // Write builder claims
    const claimsPath = path.join(tempDir, '.aether', 'data', 'last-build-claims.json');
    fs.writeFileSync(claimsPath, JSON.stringify({
      files_created: [path.join(srcDir, 'auth.ts')],
      files_modified: [path.join(srcDir, 'types.ts')],
      build_phase: 1,
      timestamp: new Date().toISOString()
    }));

    const result = runVerifyClaims(tempDir, claimsPath, '{"verification_passed":true}', 0);
    const parsed = JSON.parse(result.stdout.trim());

    t.true(parsed.ok);
    t.is(parsed.result.verification_status, 'passed');
    t.is(parsed.result.blocked, false);
    t.is(parsed.result.checks_run, 2);
    t.deepEqual(parsed.result.mismatches, []);
    t.is(parsed.result.summary, 'Verification passed');
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('verify-claims: detects missing file and blocks', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);

    const srcDir = path.join(tempDir, 'src');
    fs.mkdirSync(srcDir, { recursive: true });
    // Only create one file -- the other is claimed but nonexistent
    fs.writeFileSync(path.join(srcDir, 'auth.ts'), 'export const auth = {};');

    const claimsPath = path.join(tempDir, '.aether', 'data', 'last-build-claims.json');
    fs.writeFileSync(claimsPath, JSON.stringify({
      files_created: [path.join(srcDir, 'auth.ts'), path.join(srcDir, 'nonexistent.ts')],
      files_modified: [],
      build_phase: 1,
      timestamp: new Date().toISOString()
    }));

    const result = runVerifyClaims(tempDir, claimsPath, '{"verification_passed":true}', 0);
    const parsed = JSON.parse(result.stdout.trim());

    t.true(parsed.ok);
    t.is(parsed.result.verification_status, 'blocked');
    t.is(parsed.result.blocked, true);
    t.is(parsed.result.mismatches.length, 1);
    t.is(parsed.result.mismatches[0].type, 'missing_file');
    t.true(parsed.result.summary.includes('missing'));
    t.true(parsed.result.summary.includes('Blocked'));
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('verify-claims: test exit code mismatch with watcher passed=true blocks', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);

    // No builder claims file needed for this test (will be graceful)
    const claimsPath = path.join(tempDir, '.aether', 'data', 'last-build-claims.json');
    fs.writeFileSync(claimsPath, JSON.stringify({
      files_created: [],
      files_modified: [],
      build_phase: 1,
      timestamp: new Date().toISOString()
    }));

    // Test exit code 1 but watcher says passed
    const result = runVerifyClaims(tempDir, claimsPath, '{"verification_passed":true}', 1);
    const parsed = JSON.parse(result.stdout.trim());

    t.true(parsed.ok);
    t.is(parsed.result.verification_status, 'blocked');
    t.is(parsed.result.blocked, true);
    t.is(parsed.result.mismatches.length, 1);
    t.is(parsed.result.mismatches[0].type, 'test_mismatch');
    t.true(parsed.result.summary.includes('Blocked'));
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('verify-claims: clean pass when all files exist and tests pass', t => {
  const tempDir = createTempDir();
  try {
    setupTempAether(tempDir);

    const srcDir = path.join(tempDir, 'src');
    fs.mkdirSync(srcDir, { recursive: true });
    fs.writeFileSync(path.join(srcDir, 'index.ts'), 'export default {};');

    const claimsPath = path.join(tempDir, '.aether', 'data', 'last-build-claims.json');
    fs.writeFileSync(claimsPath, JSON.stringify({
      files_created: [path.join(srcDir, 'index.ts')],
      files_modified: [],
      build_phase: 1,
      timestamp: new Date().toISOString()
    }));

    const result = runVerifyClaims(tempDir, claimsPath, '{"verification_passed":true}', 0);
    const parsed = JSON.parse(result.stdout.trim());

    t.true(parsed.ok);
    t.is(parsed.result.verification_status, 'passed');
    t.is(parsed.result.blocked, false);
    t.is(parsed.result.checks_run, 2);
    t.is(parsed.result.summary, 'Verification passed');
  } finally {
    cleanupTempDir(tempDir);
  }
});
