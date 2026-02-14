/**
 * Unit tests for generateManifest and validateManifest functions
 *
 * Tests manifest generation and validation behavior including:
 * - File discovery and hash inclusion
 * - Exclusions (registry.json, version.json, manifest.json)
 * - Nested directory handling
 * - Validation rules and error messages
 */

const test = require('ava');
const sinon = require('sinon');
const proxyquire = require('proxyquire');

// Prevent proxyquire from caching the module
proxyquire.noPreserveCache();

/**
 * Creates a mock Dirent object for readdirSync with withFileTypes option
 * @param {string} name - File or directory name
 * @param {boolean} isDir - Whether this represents a directory
 * @returns {Object} Mock Dirent object
 */
function createMockDirent(name, isDir = false) {
  return {
    name,
    isDirectory: () => isDir,
    isFile: () => !isDir,
    isBlockDevice: () => false,
    isCharacterDevice: () => false,
    isFIFO: () => false,
    isSocket: () => false,
    isSymbolicLink: () => false
  };
}

/**
 * Creates a mock commander program that supports the chained API
 * @returns {Object} Mock commander program
 */
function createMockCommander() {
  const mockSubCommand = {
    description: sinon.stub().returnsThis(),
    argument: sinon.stub().returnsThis(),
    option: sinon.stub().returnsThis(),
    action: sinon.stub().returnsThis()
  };

  const mockCommand = {
    description: sinon.stub().returnsThis(),
    option: sinon.stub().returnsThis(),
    action: sinon.stub().returnsThis(),
    addCommand: sinon.stub().returnsThis()
  };

  const mockProgram = {
    name: sinon.stub().returnsThis(),
    description: sinon.stub().returnsThis(),
    version: sinon.stub().returnsThis(),
    option: sinon.stub().returnsThis(),
    helpOption: sinon.stub().returnsThis(),
    addCommand: sinon.stub().returnsThis(),
    createCommand: sinon.stub().returns(mockSubCommand),
    command: sinon.stub().returns(mockCommand),
    on: sinon.stub().returnsThis(),
    configureOutput: sinon.stub().returnsThis(),
    parse: sinon.stub().returnsThis()
  };

  return { program: mockProgram };
}

/**
 * Creates a fresh mock fs object for each test
 * @returns {Object} Mock fs object with sinon stubs
 */
function createMockFs() {
  return {
    existsSync: sinon.stub(),
    readdirSync: sinon.stub(),
    readFileSync: sinon.stub(),
    mkdirSync: sinon.stub(),
    writeFileSync: sinon.stub(),
    copyFileSync: sinon.stub(),
    unlinkSync: sinon.stub(),
    rmdirSync: sinon.stub(),
    statSync: sinon.stub(),
    chmodSync: sinon.stub()
  };
}

// Use serial tests to avoid module caching issues with commander.js
test.beforeEach(t => {
  // Create fresh mock fs for each test
  t.context.mockFs = createMockFs();

  // Create mock commander
  t.context.mockCommander = createMockCommander();

  // Load cli.js with mocked dependencies
  t.context.cli = proxyquire('../../bin/cli.js', {
    fs: t.context.mockFs,
    commander: t.context.mockCommander
  });
});

test.afterEach(() => {
  // Restore all stubs after each test
  sinon.restore();
  // Delete the module from require cache to prevent commander.js conflicts
  delete require.cache[require.resolve('../../bin/cli.js')];
});

// ============================================================================
// Tests for generateManifest
// ============================================================================

test.serial('generateManifest returns object with generated_at timestamp', t => {
  const { mockFs, cli } = t.context;
  const hubDir = '/test/hub';

  // Mock empty directory
  mockFs.existsSync.withArgs(hubDir).returns(true);
  mockFs.readdirSync.withArgs(hubDir, { withFileTypes: true }).returns([]);

  const result = cli.generateManifest(hubDir);

  t.truthy(result.generated_at);
  t.is(typeof result.generated_at, 'string');
  // Verify ISO 8601 format (rough check)
  t.true(/\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/.test(result.generated_at));
});

test.serial('generateManifest includes files with hashes', t => {
  const { mockFs, cli } = t.context;
  const hubDir = '/test/hub';

  // Mock directory with files
  mockFs.existsSync.callsFake((path) => {
    return path === hubDir || path === '/test/hub/file1.txt' || path === '/test/hub/file2.txt';
  });

  mockFs.readdirSync.callsFake((path, options) => {
    if (path === hubDir) {
      return [
        createMockDirent('file1.txt', false),
        createMockDirent('file2.txt', false)
      ];
    }
    return [];
  });

  mockFs.readFileSync.callsFake((path) => {
    if (path === '/test/hub/file1.txt') return Buffer.from('content1');
    if (path === '/test/hub/file2.txt') return Buffer.from('content2');
    throw new Error('File not found');
  });

  const result = cli.generateManifest(hubDir);

  t.truthy(result.files);
  t.truthy(result.files['file1.txt']);
  t.truthy(result.files['file2.txt']);
  // Verify hash format (sha256: prefix)
  t.true(result.files['file1.txt'].startsWith('sha256:'));
  t.true(result.files['file2.txt'].startsWith('sha256:'));
});

test.serial('generateManifest excludes registry.json, version.json, manifest.json', t => {
  const { mockFs, cli } = t.context;
  const hubDir = '/test/hub';

  // Mock directory with excluded files
  mockFs.existsSync.callsFake((path) => {
    const files = [hubDir, '/test/hub/registry.json', '/test/hub/version.json', '/test/hub/manifest.json', '/test/hub/keep.txt'];
    return files.includes(path);
  });

  mockFs.readdirSync.callsFake((path, options) => {
    if (path === hubDir) {
      return [
        createMockDirent('registry.json', false),
        createMockDirent('version.json', false),
        createMockDirent('manifest.json', false),
        createMockDirent('keep.txt', false)
      ];
    }
    return [];
  });

  mockFs.readFileSync.callsFake((path) => {
    if (path === '/test/hub/keep.txt') return Buffer.from('keep me');
    if (path === '/test/hub/registry.json') return Buffer.from('registry');
    if (path === '/test/hub/version.json') return Buffer.from('version');
    if (path === '/test/hub/manifest.json') return Buffer.from('manifest');
    throw new Error('File not found');
  });

  const result = cli.generateManifest(hubDir);

  // Excluded files should not be in result
  t.falsy(result.files['registry.json']);
  t.falsy(result.files['version.json']);
  t.falsy(result.files['manifest.json']);

  // Other files should be included
  t.truthy(result.files['keep.txt']);
});

test.serial('generateManifest skips files that cannot be hashed', t => {
  const { mockFs, cli } = t.context;
  const hubDir = '/test/hub';

  // Mock directory with one readable and one unreadable file
  mockFs.existsSync.callsFake((path) => {
    return path === hubDir || path === '/test/hub/readable.txt' || path === '/test/hub/unreadable.txt';
  });

  mockFs.readdirSync.callsFake((path, options) => {
    if (path === hubDir) {
      return [
        createMockDirent('readable.txt', false),
        createMockDirent('unreadable.txt', false)
      ];
    }
    return [];
  });

  mockFs.readFileSync.callsFake((path) => {
    if (path === '/test/hub/readable.txt') return Buffer.from('readable content');
    if (path === '/test/hub/unreadable.txt') {
      const error = new Error('EACCES: permission denied');
      error.code = 'EACCES';
      throw error;
    }
    throw new Error('File not found');
  });

  const result = cli.generateManifest(hubDir);

  // Readable file should be included
  t.truthy(result.files['readable.txt']);

  // Unreadable file should be skipped (not in result)
  t.falsy(result.files['unreadable.txt']);
});

test.serial('generateManifest handles nested directories', t => {
  const { mockFs, cli } = t.context;
  const hubDir = '/test/hub';

  // Mock nested directory structure
  mockFs.existsSync.callsFake((path) => {
    const paths = [
      hubDir,
      '/test/hub/dir1',
      '/test/hub/dir2',
      '/test/hub/dir1/file.txt',
      '/test/hub/dir2/sub',
      '/test/hub/dir2/sub/file.txt'
    ];
    return paths.includes(path);
  });

  mockFs.readdirSync.callsFake((path, options) => {
    if (path === hubDir) {
      return [
        createMockDirent('dir1', true),
        createMockDirent('dir2', true)
      ];
    }
    if (path === '/test/hub/dir1') {
      return [createMockDirent('file.txt', false)];
    }
    if (path === '/test/hub/dir2') {
      return [createMockDirent('sub', true)];
    }
    if (path === '/test/hub/dir2/sub') {
      return [createMockDirent('file.txt', false)];
    }
    return [];
  });

  mockFs.readFileSync.callsFake((path) => {
    if (path === '/test/hub/dir1/file.txt') return Buffer.from('dir1 file');
    if (path === '/test/hub/dir2/sub/file.txt') return Buffer.from('nested file');
    throw new Error('File not found');
  });

  const result = cli.generateManifest(hubDir);

  // All files should be included with correct relative paths
  t.truthy(result.files['dir1/file.txt']);
  t.truthy(result.files['dir2/sub/file.txt']);
  t.is(Object.keys(result.files).length, 2);
});

test.serial('generateManifest handles empty directory', t => {
  const { mockFs, cli } = t.context;
  const hubDir = '/test/hub';

  // Mock empty directory
  mockFs.existsSync.withArgs(hubDir).returns(true);
  mockFs.readdirSync.withArgs(hubDir, { withFileTypes: true }).returns([]);

  const result = cli.generateManifest(hubDir);

  t.truthy(result.generated_at);
  t.deepEqual(result.files, {});
  t.is(Object.keys(result.files).length, 0);
});

// ============================================================================
// Tests for validateManifest
// ============================================================================

test.serial('validateManifest returns valid for correct manifest', t => {
  const { cli } = t.context;

  const validManifest = {
    generated_at: '2024-01-15T10:30:00.000Z',
    files: {
      'file1.txt': 'sha256:aabbccdd',
      'file2.txt': 'sha256:eeffgghh'
    }
  };

  const result = cli.validateManifest(validManifest);

  t.true(result.valid);
  t.falsy(result.error);
});

test.serial('validateManifest returns error for null manifest', t => {
  const { cli } = t.context;

  const result = cli.validateManifest(null);

  t.false(result.valid);
  t.is(result.error, 'Manifest must be an object');
});

test.serial('validateManifest returns error for undefined manifest', t => {
  const { cli } = t.context;

  const result = cli.validateManifest(undefined);

  t.false(result.valid);
  t.is(result.error, 'Manifest must be an object');
});

test.serial('validateManifest returns error for string manifest', t => {
  const { cli } = t.context;

  const result = cli.validateManifest('not an object');

  t.false(result.valid);
  t.is(result.error, 'Manifest must be an object');
});

test.serial('validateManifest returns error for missing generated_at', t => {
  const { cli } = t.context;

  const manifest = {
    files: {
      'file1.txt': 'sha256:aabbccdd'
    }
  };

  const result = cli.validateManifest(manifest);

  t.false(result.valid);
  t.is(result.error, 'Manifest missing required field: generated_at');
});

test.serial('validateManifest returns error for non-string generated_at', t => {
  const { cli } = t.context;

  const manifest = {
    generated_at: 12345,
    files: {}
  };

  const result = cli.validateManifest(manifest);

  t.false(result.valid);
  t.is(result.error, 'Manifest missing required field: generated_at');
});

test.serial('validateManifest returns error for missing files', t => {
  const { cli } = t.context;

  const manifest = {
    generated_at: '2024-01-15T10:30:00.000Z'
  };

  const result = cli.validateManifest(manifest);

  t.false(result.valid);
  t.is(result.error, 'Manifest missing required field: files');
});

test.serial('validateManifest returns error for non-object files', t => {
  const { cli } = t.context;

  const manifest = {
    generated_at: '2024-01-15T10:30:00.000Z',
    files: ['file1.txt', 'file2.txt'] // Array instead of object
  };

  const result = cli.validateManifest(manifest);

  t.false(result.valid);
  t.is(result.error, 'Manifest missing required field: files');
});

test.serial('validateManifest accepts empty files object', t => {
  const { cli } = t.context;

  const manifest = {
    generated_at: '2024-01-15T10:30:00.000Z',
    files: {}
  };

  const result = cli.validateManifest(manifest);

  t.true(result.valid);
  t.falsy(result.error);
});

test.serial('validateManifest returns error for null files', t => {
  const { cli } = t.context;

  const manifest = {
    generated_at: '2024-01-15T10:30:00.000Z',
    files: null
  };

  const result = cli.validateManifest(manifest);

  t.false(result.valid);
  t.is(result.error, 'Manifest missing required field: files');
});
