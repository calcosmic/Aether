const test = require('ava');
const sinon = require('sinon');
const proxyquire = require('proxyquire');
const path = require('path');

const BINARY_PATH = path.join(
  process.env.HOME || process.env.USERPROFILE,
  '.aether', 'bin', 'aether'
);

const PKG_VERSION = '5.3.3';

/**
 * Create mock fs module
 */
function createMockFs() {
  return {
    existsSync: sinon.stub(),
    accessSync: sinon.stub(),
    constants: { X_OK: 1, R_OK: 4 },
  };
}

/**
 * Create mock child_process module
 */
function createMockCp() {
  return {
    execSync: sinon.stub(),
  };
}

/**
 * Load version-gate with injected mocks
 */
function loadGate(mockFs, mockCp, mockVersion) {
  return proxyquire('../../bin/lib/version-gate', {
    fs: mockFs,
    child_process: mockCp,
    '../../package.json': { version: mockVersion || PKG_VERSION },
  });
}

// --- compareVersions ---

test('compareVersions: equal versions', (t) => {
  const { compareVersions } = loadGate(createMockFs(), createMockCp());
  t.is(compareVersions('1.2.3', '1.2.3'), 0);
});

test('compareVersions: greater patch', (t) => {
  const { compareVersions } = loadGate(createMockFs(), createMockCp());
  t.is(compareVersions('1.2.4', '1.2.3'), 1);
});

test('compareVersions: lesser minor', (t) => {
  const { compareVersions } = loadGate(createMockFs(), createMockCp());
  t.is(compareVersions('1.1.9', '1.2.0'), -1);
});

test('compareVersions: greater major', (t) => {
  const { compareVersions } = loadGate(createMockFs(), createMockCp());
  t.is(compareVersions('2.0.0', '1.9.9'), 1);
});

test('compareVersions: v-prefix on first', (t) => {
  const { compareVersions } = loadGate(createMockFs(), createMockCp());
  t.is(compareVersions('v1.2.3', '1.2.3'), 0);
});

test('compareVersions: v-prefix on both', (t) => {
  const { compareVersions } = loadGate(createMockFs(), createMockCp());
  t.is(compareVersions('v1.2.3', 'v1.2.3'), 0);
});

test('compareVersions: prerelease tag ignored', (t) => {
  const { compareVersions } = loadGate(createMockFs(), createMockCp());
  t.is(compareVersions('1.2.3-alpha.1', '1.2.3'), 0);
});

test('compareVersions: prerelease tag on both same base', (t) => {
  const { compareVersions } = loadGate(createMockFs(), createMockCp());
  t.is(compareVersions('2.0.0-beta.1', '2.0.0-rc.2'), 0);
});

test('compareVersions: different lengths', (t) => {
  const { compareVersions } = loadGate(createMockFs(), createMockCp());
  t.is(compareVersions('1.2', '1.2.0'), 0);
});

test('compareVersions: non-numeric part treated as 0', (t) => {
  const { compareVersions } = loadGate(createMockFs(), createMockCp());
  t.is(compareVersions('1.x.0', '1.0.0'), 0);
});

// --- checkBinary ---

test('checkBinary: missing binary', (t) => {
  const mockFs = createMockFs();
  mockFs.existsSync.returns(false);
  const mockCp = createMockCp();

  const { checkBinary } = loadGate(mockFs, mockCp);
  const result = checkBinary({ binaryPath: '/fake/aether' });

  t.false(result.available);
  t.is(result.reason, 'binary not found');
  t.is(result.version, null);
});

test('checkBinary: binary not executable', (t) => {
  const mockFs = createMockFs();
  mockFs.existsSync.returns(true);
  mockFs.accessSync.throws(new Error('EACCES'));
  const mockCp = createMockCp();

  const { checkBinary } = loadGate(mockFs, mockCp);
  const result = checkBinary({ binaryPath: '/fake/aether' });

  t.false(result.available);
  t.is(result.reason, 'binary not executable');
});

test('checkBinary: version check fails', (t) => {
  const mockFs = createMockFs();
  mockFs.existsSync.returns(true);
  mockFs.accessSync.returns(undefined);
  const mockCp = createMockCp();
  mockCp.execSync.throws(new Error('spawn error'));

  const { checkBinary } = loadGate(mockFs, mockCp);
  const result = checkBinary({ binaryPath: '/fake/aether' });

  t.false(result.available);
  t.is(result.reason, 'binary version check failed');
});

test('checkBinary: version matches', (t) => {
  const mockFs = createMockFs();
  mockFs.existsSync.returns(true);
  mockFs.accessSync.returns(undefined);
  const mockCp = createMockCp();
  mockCp.execSync.returns(PKG_VERSION);

  const { checkBinary } = loadGate(mockFs, mockCp);
  const result = checkBinary({ binaryPath: '/fake/aether' });

  t.true(result.available);
  t.is(result.version, PKG_VERSION);
  t.is(result.reason, null);
});

test('checkBinary: version mismatch', (t) => {
  const mockFs = createMockFs();
  mockFs.existsSync.returns(true);
  mockFs.accessSync.returns(undefined);
  const mockCp = createMockCp();
  mockCp.execSync.returns('4.0.0');

  const { checkBinary } = loadGate(mockFs, mockCp);
  const result = checkBinary({ binaryPath: '/fake/aether' });

  t.false(result.available);
  t.is(result.version, '4.0.0');
  t.truthy(result.reason);
  t.true(result.reason.includes('version mismatch'));
});

test('checkBinary: v-prefix version match', (t) => {
  const mockFs = createMockFs();
  mockFs.existsSync.returns(true);
  mockFs.accessSync.returns(undefined);
  const mockCp = createMockCp();
  mockCp.execSync.returns('v5.3.3');

  const { checkBinary } = loadGate(mockFs, mockCp);
  const result = checkBinary({ binaryPath: '/fake/aether' });

  t.true(result.available);
  t.is(result.version, 'v5.3.3');
});

// --- shouldDelegate ---

test('shouldDelegate: node-only command "install" never delegates', (t) => {
  const mockFs = createMockFs();
  mockFs.existsSync.returns(true);
  mockFs.accessSync.returns(undefined);
  const mockCp = createMockCp();
  mockCp.execSync.returns(PKG_VERSION);

  const { shouldDelegate } = loadGate(mockFs, mockCp);
  const result = shouldDelegate(['node', 'cli.js', 'install'], {
    binaryPath: '/fake/aether',
  });

  t.false(result);
});

test('shouldDelegate: node-only command "update" never delegates', (t) => {
  const mockFs = createMockFs();
  mockFs.existsSync.returns(true);
  mockFs.accessSync.returns(undefined);
  const mockCp = createMockCp();
  mockCp.execSync.returns(PKG_VERSION);

  const { shouldDelegate } = loadGate(mockFs, mockCp);
  const result = shouldDelegate(['node', 'cli.js', 'update'], {
    binaryPath: '/fake/aether',
  });

  t.false(result);
});

test('shouldDelegate: node-only command "setup" never delegates', (t) => {
  const mockFs = createMockFs();
  mockFs.existsSync.returns(true);
  mockFs.accessSync.returns(undefined);
  const mockCp = createMockCp();
  mockCp.execSync.returns(PKG_VERSION);

  const { shouldDelegate } = loadGate(mockFs, mockCp);
  const result = shouldDelegate(['node', 'cli.js', 'setup'], {
    binaryPath: '/fake/aether',
  });

  t.false(result);
});

test('shouldDelegate: other command delegates when available', (t) => {
  const mockFs = createMockFs();
  mockFs.existsSync.returns(true);
  mockFs.accessSync.returns(undefined);
  const mockCp = createMockCp();
  mockCp.execSync.returns(PKG_VERSION);

  const { shouldDelegate } = loadGate(mockFs, mockCp);
  const result = shouldDelegate(['node', 'cli.js', 'status'], {
    binaryPath: '/fake/aether',
  });

  t.true(result);
});

test('shouldDelegate: other command does not delegate when unavailable', (t) => {
  const mockFs = createMockFs();
  mockFs.existsSync.returns(false);
  const mockCp = createMockCp();

  const { shouldDelegate } = loadGate(mockFs, mockCp);
  const result = shouldDelegate(['node', 'cli.js', 'status'], {
    binaryPath: '/fake/aether',
  });

  t.false(result);
});

test('shouldDelegate: version flag delegates when available', (t) => {
  const mockFs = createMockFs();
  mockFs.existsSync.returns(true);
  mockFs.accessSync.returns(undefined);
  const mockCp = createMockCp();
  mockCp.execSync.returns(PKG_VERSION);

  const { shouldDelegate } = loadGate(mockFs, mockCp);
  const result = shouldDelegate(['node', 'cli.js', '-v'], {
    binaryPath: '/fake/aether',
  });

  t.true(result);
});

test('shouldDelegate: no args delegates to binary for help', (t) => {
  const mockFs = createMockFs();
  mockFs.existsSync.returns(true);
  mockFs.accessSync.returns(undefined);
  const mockCp = createMockCp();
  mockCp.execSync.returns(PKG_VERSION);

  const { shouldDelegate } = loadGate(mockFs, mockCp);
  const result = shouldDelegate(['node', 'cli.js'], {
    binaryPath: '/fake/aether',
  });

  t.true(result);
});

// --- NODE_ONLY_COMMANDS ---

test('NODE_ONLY_COMMANDS contains required entries', (t) => {
  const { NODE_ONLY_COMMANDS } = loadGate(createMockFs(), createMockCp());
  t.true(NODE_ONLY_COMMANDS.includes('install'));
  t.true(NODE_ONLY_COMMANDS.includes('update'));
  t.true(NODE_ONLY_COMMANDS.includes('setup'));
  t.true(NODE_ONLY_COMMANDS.includes('setup-hub'));
});

// --- getBinaryPath ---

test('getBinaryPath returns a path under .aether/bin', (t) => {
  const { getBinaryPath } = loadGate(createMockFs(), createMockCp());
  const p = getBinaryPath();
  t.truthy(p);
  t.true(p.includes('.aether'));
  t.true(p.includes('bin'));
  t.true(p.endsWith('aether'));
});
