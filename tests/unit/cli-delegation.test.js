/**
 * CLI Delegation Integration Tests
 *
 * Tests the delegation shim in bin/cli.js by verifying:
 * 1. When version gate passes, the Go binary would be called
 * 2. Node-only commands (install, update, setup) never delegate
 * 3. Fallback to Node.js when binary unavailable or version mismatch
 *
 * These tests exercise the version-gate module through proxyquire
 * with mocked fs and child_process to simulate binary states.
 */

const test = require('ava');
const sinon = require('sinon');
const proxyquire = require('proxyquire');

const PKG_VERSION = '5.3.3';

/**
 * Create mock fs module for version-gate
 */
function createMockFs(opts) {
  opts = opts || {};
  const stub = {
    existsSync: sinon.stub().returns(opts.exists !== false),
    accessSync: sinon.stub().returns(undefined),
    constants: { X_OK: 1, R_OK: 4 },
  };
  if (opts.notExecutable) {
    stub.accessSync.throws(new Error('EACCES'));
  }
  return stub;
}

/**
 * Create mock child_process
 */
function createMockCp(opts) {
  opts = opts || {};
  const stub = {
    execSync: sinon.stub().returns(opts.version || PKG_VERSION),
    spawnSync: sinon.stub().returns({ status: opts.exitStatus || 0 }),
  };
  if (opts.execFails) {
    stub.execSync.throws(new Error('spawn error'));
  }
  return stub;
}

/**
 * Load version-gate with injected mocks
 */
function loadGate(mockFs, mockCp) {
  return proxyquire('../../bin/lib/version-gate', {
    fs: mockFs,
    child_process: mockCp,
    '../../package.json': { version: PKG_VERSION },
  });
}

// --- Delegation: node-only commands always stay in Node.js ---

test('delegation: install command never delegates even when binary available', (t) => {
  const gate = loadGate(createMockFs(), createMockCp());

  // Binary is available
  t.true(gate.checkBinary({ binaryPath: '/fake/aether' }).available);

  // But install still doesn't delegate
  t.false(gate.shouldDelegate(['node', 'cli.js', 'install'], { binaryPath: '/fake/aether' }));
});

test('delegation: update command never delegates even when binary available', (t) => {
  const gate = loadGate(createMockFs(), createMockCp());

  t.true(gate.checkBinary({ binaryPath: '/fake/aether' }).available);
  t.false(gate.shouldDelegate(['node', 'cli.js', 'update'], { binaryPath: '/fake/aether' }));
});

test('delegation: setup command never delegates even when binary available', (t) => {
  const gate = loadGate(createMockFs(), createMockCp());

  t.true(gate.checkBinary({ binaryPath: '/fake/aether' }).available);
  t.false(gate.shouldDelegate(['node', 'cli.js', 'setup'], { binaryPath: '/fake/aether' }));
});

test('delegation: setup-hub command never delegates even when binary available', (t) => {
  const gate = loadGate(createMockFs(), createMockCp());

  t.true(gate.checkBinary({ binaryPath: '/fake/aether' }).available);
  t.false(gate.shouldDelegate(['node', 'cli.js', 'setup-hub'], { binaryPath: '/fake/aether' }));
});

// --- Delegation: other commands delegate when binary available ---

test('delegation: status command delegates when binary available', (t) => {
  const gate = loadGate(createMockFs(), createMockCp());
  t.true(gate.shouldDelegate(['node', 'cli.js', 'status'], { binaryPath: '/fake/aether' }));
});

test('delegation: -v flag delegates when binary available', (t) => {
  const gate = loadGate(createMockFs(), createMockCp());
  t.true(gate.shouldDelegate(['node', 'cli.js', '-v'], { binaryPath: '/fake/aether' }));
});

test('delegation: version command delegates when binary available', (t) => {
  const gate = loadGate(createMockFs(), createMockCp());
  t.true(gate.shouldDelegate(['node', 'cli.js', 'version'], { binaryPath: '/fake/aether' }));
});

test('delegation: no args delegates to binary for help output', (t) => {
  const gate = loadGate(createMockFs(), createMockCp());
  t.true(gate.shouldDelegate(['node', 'cli.js'], { binaryPath: '/fake/aether' }));
});

// --- Delegation: fallback to Node.js ---

test('delegation: falls back when binary not found', (t) => {
  const gate = loadGate(createMockFs({ exists: false }), createMockCp());
  t.false(gate.shouldDelegate(['node', 'cli.js', 'status'], { binaryPath: '/fake/aether' }));
});

test('delegation: falls back when binary version mismatches', (t) => {
  const gate = loadGate(createMockFs(), createMockCp({ version: '1.0.0' }));
  t.false(gate.shouldDelegate(['node', 'cli.js', 'status'], { binaryPath: '/fake/aether' }));
});

test('delegation: falls back when binary not executable', (t) => {
  const gate = loadGate(createMockFs({ notExecutable: true }), createMockCp());
  t.false(gate.shouldDelegate(['node', 'cli.js', 'status'], { binaryPath: '/fake/aether' }));
});

// --- spawnSync verification ---

test('delegation: spawnSync not called when command does not delegate', (t) => {
  const mockCp = createMockCp();
  const gate = loadGate(createMockFs(), mockCp);

  // install is node-only -- shouldDelegate returns false
  t.false(gate.shouldDelegate(['node', 'cli.js', 'install'], { binaryPath: '/fake/aether' }));

  // spawnSync should not have been called by version-gate itself
  t.is(mockCp.spawnSync.callCount, 0);
});

test('delegation: getBinaryPath returns valid path for spawnSync', (t) => {
  const gate = loadGate(createMockFs(), createMockCp());

  // Verify shouldDelegate says yes for status
  t.true(gate.shouldDelegate(['node', 'cli.js', 'status'], { binaryPath: '/fake/aether' }));

  // The actual spawnSync call happens in cli.js, not in version-gate.
  // We verify the contract: getBinaryPath returns a valid path.
  const binaryPath = gate.getBinaryPath();
  t.truthy(binaryPath);
  t.true(binaryPath.includes('.aether'));
  t.true(binaryPath.includes('bin'));
  t.true(binaryPath.endsWith('aether'));
});
