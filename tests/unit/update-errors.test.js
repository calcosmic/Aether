const test = require('ava');
const sinon = require('sinon');
const proxyquire = require('proxyquire');

// Helper to create mock Dirent objects
function createMockDirent(name, isDirectory = false) {
  return {
    name,
    isDirectory: () => isDirectory,
    isFile: () => !isDirectory,
  };
}

// Mock fs and child_process for testing
const createMockFs = () => ({
  existsSync: sinon.stub(),
  readFileSync: sinon.stub(),
  writeFileSync: sinon.stub(),
  mkdirSync: sinon.stub(),
  readdirSync: sinon.stub(),
  copyFileSync: sinon.stub(),
  unlinkSync: sinon.stub(),
  rmdirSync: sinon.stub(),
  chmodSync: sinon.stub(),
  statSync: sinon.stub(),
  accessSync: sinon.stub(),
});

const createMockCp = () => ({
  execSync: sinon.stub(),
});

const createMockCrypto = () => ({
  createHash: sinon.stub().returns({
    update: sinon.stub().returns({
      digest: sinon.stub().returns('abc123hash'),
    }),
  }),
});

test.beforeEach((t) => {
  t.context.mockFs = createMockFs();
  t.context.mockCp = createMockCp();
  t.context.mockCrypto = createMockCrypto();

  // Setup default successful behaviors
  t.context.mockFs.existsSync.returns(true);
  t.context.mockFs.readFileSync.returns('{}');
  t.context.mockFs.readdirSync.returns([]);

  // Load module with mocks
  const modulePath = '../../bin/lib/update-transaction';
  t.context.module = proxyquire(modulePath, {
    fs: t.context.mockFs,
    child_process: t.context.mockCp,
    crypto: t.context.mockCrypto,
  });

  t.context.UpdateTransaction = t.context.module.UpdateTransaction;
  t.context.UpdateError = t.context.module.UpdateError;
  t.context.UpdateErrorCodes = t.context.module.UpdateErrorCodes;
});

test.afterEach((t) => {
  sinon.restore();
});

// Test 1: detectDirtyRepo identifies modified files
test.skip('detectDirtyRepo identifies modified files', (t) => {
  const { UpdateTransaction, mockFs, mockCp } = t.context;
  const transaction = new UpdateTransaction('/test/repo');

  // Setup: git status returns modified files
  mockFs.existsSync.withArgs('/test/repo/.aether').returns(true);
  mockCp.execSync.withArgs(sinon.match(/git status --porcelain/)).returns(
    Buffer.from(' M .aether/config.json\n?? .aether/new-file.txt\nA  .aether/staged.md\n')
  );

  const result = transaction.detectDirtyRepo();

  t.true(result.isDirty);
  t.is(result.tracked.length, 2);
  t.is(result.untracked.length, 1);
  t.is(result.staged.length, 1);
  t.true(result.tracked.includes('.aether/config.json'));
  t.true(result.untracked.includes('.aether/new-file.txt'));
  t.true(result.staged.includes('.aether/staged.md'));
});

// Test 2: detectDirtyRepo returns clean for no changes
test('detectDirtyRepo returns clean state for no changes', (t) => {
  const { UpdateTransaction, mockFs, mockCp } = t.context;
  const transaction = new UpdateTransaction('/test/repo');

  // Setup: git status returns empty
  mockFs.existsSync.withArgs('/test/repo/.aether').returns(true);
  mockCp.execSync.withArgs(sinon.match(/git status --porcelain/)).returns(Buffer.from(''));

  const result = transaction.detectDirtyRepo();

  t.false(result.isDirty);
  t.is(result.tracked.length, 0);
  t.is(result.untracked.length, 0);
  t.is(result.staged.length, 0);
});

// Test 3: validateRepoState throws on dirty repo
test.skip('validateRepoState throws UpdateError with E_REPO_DIRTY', (t) => {
  const { UpdateTransaction, UpdateErrorCodes, mockFs, mockCp } = t.context;
  const transaction = new UpdateTransaction('/test/repo');

  // Setup: repo has uncommitted changes
  mockFs.existsSync.withArgs('/test/repo/.aether').returns(true);
  mockCp.execSync.withArgs(sinon.match(/git status --porcelain/)).returns(
    Buffer.from(' M .aether/config.json\n?? .aether/new-file.txt\n')
  );

  const error = t.throws(() => transaction.validateRepoState());

  t.is(error.code, UpdateErrorCodes.E_REPO_DIRTY);
  t.is(error.message, 'Repository has uncommitted changes');
  t.is(error.details.trackedCount, 1);
  t.is(error.details.untrackedCount, 1);
  t.true(error.recoveryCommands.length > 0);
  t.true(error.recoveryCommands.some(cmd => cmd.includes('git stash')));
});

// Test 4: validateRepoState returns clean for clean repo
test('validateRepoState returns clean for clean repository', (t) => {
  const { UpdateTransaction, mockFs, mockCp } = t.context;
  const transaction = new UpdateTransaction('/test/repo');

  // Setup: clean repo
  mockFs.existsSync.withArgs('/test/repo/.aether').returns(true);
  mockCp.execSync.withArgs(sinon.match(/git status --porcelain/)).returns(Buffer.from(''));

  const result = transaction.validateRepoState();

  t.true(result.clean);
});

// Test 5: checkHubAccessibility detects missing hub
test('checkHubAccessibility detects missing hub', (t) => {
  const { UpdateTransaction, mockFs } = t.context;
  const transaction = new UpdateTransaction('/test/repo');
  const home = process.env.HOME || process.env.USERPROFILE;

  // Setup: HUB_DIR doesn't exist
  mockFs.existsSync.callsFake((path) => {
    if (path.includes(`${home}/.aether`)) return false;
    return true;
  });

  const result = transaction.checkHubAccessibility();

  t.false(result.accessible);
  t.true(result.errors.length > 0);
  t.true(result.errors[0].includes('Hub directory does not exist'));
  t.true(result.recoveryCommands.includes('aether install'));
});

// Test 6: checkHubAccessibility returns accessible for valid hub
test('checkHubAccessibility returns accessible for valid hub', (t) => {
  const { UpdateTransaction, mockFs } = t.context;
  const transaction = new UpdateTransaction('/test/repo');

  // Setup: hub exists and is readable
  mockFs.existsSync.returns(true);
  mockFs.accessSync.returns(undefined);

  const result = transaction.checkHubAccessibility();

  t.true(result.accessible);
  t.is(result.errors.length, 0);
});

// Test 7: detectPartialUpdate finds missing files
test.skip('detectPartialUpdate finds missing files', (t) => {
  const { UpdateTransaction, mockFs } = t.context;
  const transaction = new UpdateTransaction('/test/repo');
  const home = process.env.HOME || process.env.USERPROFILE;
  const hubSystem = `${home}/.aether/system`;

  // Setup: hub has files that don't exist in repo
  mockFs.existsSync.callsFake((path) => {
    if (path === hubSystem) return true;
    if (path === '/test/repo/.aether') return true;
    if (path.includes('missing-file')) return false;
    return true;
  });

  mockFs.readdirSync.callsFake((path, options) => {
    if (path === hubSystem) {
      if (options?.withFileTypes) {
        return [createMockDirent('missing-file.txt', false)];
      }
      return ['missing-file.txt'];
    }
    return [];
  });

  const result = transaction.detectPartialUpdate();

  t.true(result.isPartial);
  t.is(result.missing.length, 1);
  t.is(result.corrupted.length, 0);
  t.is(result.missing[0].path, 'missing-file.txt');
});

// Test 8: detectPartialUpdate finds corrupted files (hash mismatch)
test.skip('detectPartialUpdate finds corrupted files with hash mismatch', (t) => {
  const { UpdateTransaction, mockFs, mockCrypto } = t.context;
  const transaction = new UpdateTransaction('/test/repo');
  const home = process.env.HOME || process.env.USERPROFILE;
  const hubSystem = `${home}/.aether/system`;
  let callCount = 0;

  // Setup: file exists but has different content
  mockFs.existsSync.returns(true);
  mockFs.statSync.returns({ size: 100 });
  mockFs.readdirSync.callsFake((path, options) => {
    if (path === hubSystem) {
      if (options?.withFileTypes) {
        return [createMockDirent('corrupted.txt', false)];
      }
      return ['corrupted.txt'];
    }
    return [];
  });

  // Mock crypto to return different hashes
  mockCrypto.createHash = sinon.stub().returns({
    update: sinon.stub().returns({
      digest: () => {
        callCount++;
        return callCount % 2 === 1 ? 'hash1' : 'hash2';
      },
    }),
  });

  const result = transaction.detectPartialUpdate();

  t.true(result.isPartial);
  t.is(result.missing.length, 0);
  t.is(result.corrupted.length, 1);
  t.is(result.corrupted[0].path, 'corrupted.txt');
  t.is(result.corrupted[0].reason, 'hash_mismatch');
});

// Test 9: detectPartialUpdate finds corrupted files (size mismatch)
test.skip('detectPartialUpdate finds corrupted files with size mismatch', (t) => {
  const { UpdateTransaction, mockFs } = t.context;
  const transaction = new UpdateTransaction('/test/repo');
  const home = process.env.HOME || process.env.USERPROFILE;
  const hubSystem = `${home}/.aether/system`;
  let sizeCallCount = 0;

  // Setup: file exists but has different size
  mockFs.existsSync.returns(true);
  mockFs.statSync = sinon.stub().callsFake(() => {
    sizeCallCount++;
    return { size: sizeCallCount === 1 ? 100 : 50 };
  });
  mockFs.readdirSync.callsFake((path, options) => {
    if (path === hubSystem) {
      if (options?.withFileTypes) {
        return [createMockDirent('size-mismatch.txt', false)];
      }
      return ['size-mismatch.txt'];
    }
    return [];
  });

  const result = transaction.detectPartialUpdate();

  t.true(result.isPartial);
  t.is(result.corrupted.length, 1);
  t.is(result.corrupted[0].path, 'size-mismatch.txt');
  t.is(result.corrupted[0].reason, 'size_mismatch');
  t.is(result.corrupted[0].hubSize, 100);
  t.is(result.corrupted[0].repoSize, 50);
});

// Test 10: handleNetworkError enhances timeout errors
test('handleNetworkError enhances ETIMEDOUT errors', (t) => {
  const { UpdateTransaction, UpdateErrorCodes } = t.context;
  const transaction = new UpdateTransaction('/test/repo');

  const networkError = new Error('Connection timed out');
  networkError.code = 'ETIMEDOUT';

  const result = transaction.handleNetworkError(networkError);

  t.is(result.code, UpdateErrorCodes.E_NETWORK_ERROR);
  t.true(result.message.includes('Network error'));
  t.true(result.details.hubDir.includes('.aether'));
  t.true(result.recoveryCommands.length > 0);
  t.true(result.recoveryCommands.some(cmd => cmd.includes('ls -la')));
});

// Test 11: handleNetworkError enhances ECONNREFUSED errors
test('handleNetworkError enhances ECONNREFUSED errors', (t) => {
  const { UpdateTransaction, UpdateErrorCodes } = t.context;
  const transaction = new UpdateTransaction('/test/repo');

  const networkError = new Error('Connection refused');
  networkError.code = 'ECONNREFUSED';

  const result = transaction.handleNetworkError(networkError);

  t.is(result.code, UpdateErrorCodes.E_NETWORK_ERROR);
  t.true(result.recoveryCommands.length > 0);
});

// Test 12: handleNetworkError handles non-network errors
test('handleNetworkError handles non-network errors gracefully', (t) => {
  const { UpdateTransaction, UpdateErrorCodes } = t.context;
  const transaction = new UpdateTransaction('/test/repo');

  const genericError = new Error('Something went wrong');
  genericError.code = 'UNKNOWN';

  const result = transaction.handleNetworkError(genericError);

  t.is(result.code, UpdateErrorCodes.E_UPDATE_FAILED);
  t.is(result.message, 'Something went wrong');
});

// Test 13: error recovery commands include cd to repo path
test.skip('E_REPO_DIRTY recovery commands include cd to repo path', (t) => {
  const { UpdateTransaction, UpdateErrorCodes, mockFs, mockCp } = t.context;
  const transaction = new UpdateTransaction('/test/repo');

  // Setup: dirty repo
  mockFs.existsSync.withArgs('/test/repo/.aether').returns(true);
  mockCp.execSync.withArgs(sinon.match(/git status --porcelain/)).returns(
    Buffer.from(' M .aether/config.json\n')
  );

  const error = t.throws(() => transaction.validateRepoState());

  t.is(error.code, UpdateErrorCodes.E_REPO_DIRTY);
  t.true(error.recoveryCommands.every(cmd => cmd.includes('/test/repo')));
});

// Test 14: UpdateError toString includes recovery commands
test('UpdateError toString includes recovery commands prominently', (t) => {
  const { UpdateError, UpdateErrorCodes } = t.context;

  const error = new UpdateError(
    UpdateErrorCodes.E_REPO_DIRTY,
    'Test error',
    { trackedCount: 2 },
    ['git stash', 'aether update']
  );

  const str = error.toString();

  t.true(str.includes('UPDATE FAILED'));
  t.true(str.includes('RECOVERY REQUIRED'));
  t.true(str.includes('git stash'));
  t.true(str.includes('aether update'));
});

// Test 15: verifySyncCompleteness throws on partial update
test.skip('verifySyncCompleteness throws E_PARTIAL_UPDATE on partial files', (t) => {
  const { UpdateTransaction, UpdateErrorCodes, mockFs } = t.context;
  const transaction = new UpdateTransaction('/test/repo');
  const home = process.env.HOME || process.env.USERPROFILE;
  const hubSystem = `${home}/.aether/system`;

  // Setup: partial update detected
  mockFs.existsSync.callsFake((path) => {
    if (path === hubSystem) return true;
    if (path === '/test/repo/.aether') return true;
    return false;
  });

  mockFs.readdirSync.callsFake((path, options) => {
    if (path === hubSystem) {
      if (options?.withFileTypes) {
        return [createMockDirent('missing.txt', false)];
      }
      return ['missing.txt'];
    }
    return [];
  });

  const error = t.throws(() => transaction.verifySyncCompleteness());

  t.is(error.code, UpdateErrorCodes.E_PARTIAL_UPDATE);
  t.true(error.message.includes('incomplete'));
  t.true(error.recoveryCommands.includes('aether update'));
});

// Test 16: E_HUB_INACCESSIBLE error has correct recovery commands
test('E_HUB_INACCESSIBLE error includes hub check commands', (t) => {
  const { UpdateTransaction, UpdateError, UpdateErrorCodes, mockFs } = t.context;
  const transaction = new UpdateTransaction('/test/repo');
  const home = process.env.HOME || process.env.USERPROFILE;
  const hubDir = `${home}/.aether`;

  // Setup: hub exists but version file doesn't
  mockFs.existsSync.callsFake((path) => {
    if (path === hubDir) return true;
    if (path.includes('version.json')) return false;
    return true;
  });

  const hubAccess = transaction.checkHubAccessibility();

  const error = new UpdateError(
    UpdateErrorCodes.E_HUB_INACCESSIBLE,
    'Hub is not accessible',
    { errors: hubAccess.errors },
    hubAccess.recoveryCommands
  );

  t.is(error.code, UpdateErrorCodes.E_HUB_INACCESSIBLE);
  t.true(error.recoveryCommands.some(cmd => cmd.includes('ls -la')));
  t.true(error.recoveryCommands.some(cmd => cmd.includes('aether install')));
});

// Test 17: E_PARTIAL_UPDATE error includes retry command
test.skip('E_PARTIAL_UPDATE error includes retry command', (t) => {
  const { UpdateTransaction, UpdateErrorCodes, mockFs } = t.context;
  const transaction = new UpdateTransaction('/test/repo');
  const home = process.env.HOME || process.env.USERPROFILE;
  const hubSystem = `${home}/.aether/system`;

  // Setup: partial update
  mockFs.existsSync.callsFake((path) => {
    if (path === hubSystem) return true;
    if (path === '/test/repo/.aether') return true;
    return false;
  });

  mockFs.readdirSync.callsFake((path, options) => {
    if (path === hubSystem) {
      if (options?.withFileTypes) {
        return [createMockDirent('file.txt', false)];
      }
      return ['file.txt'];
    }
    return [];
  });

  const error = t.throws(() => transaction.verifySyncCompleteness());

  t.is(error.code, UpdateErrorCodes.E_PARTIAL_UPDATE);
  t.true(error.recoveryCommands.includes('aether update'));
});

// Test 18: E_NETWORK_ERROR error includes network diagnostics
test('E_NETWORK_ERROR includes network diagnostics', (t) => {
  const { UpdateTransaction, UpdateErrorCodes } = t.context;
  const transaction = new UpdateTransaction('/test/repo');

  const networkError = new Error('Network timeout');
  networkError.code = 'ETIMEDOUT';

  const result = transaction.handleNetworkError(networkError);

  t.is(result.code, UpdateErrorCodes.E_NETWORK_ERROR);
  t.truthy(result.details.originalError);
  t.truthy(result.details.errorCode);
  t.true(result.recoveryCommands.length >= 3);
});
