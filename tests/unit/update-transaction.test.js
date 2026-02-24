const test = require('ava');
const sinon = require('sinon');
const proxyquire = require('proxyquire');

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
  rmSync: sinon.stub(),
  chmodSync: sinon.stub(),
  accessSync: sinon.stub(),
  renameSync: sinon.stub(),
  statSync: sinon.stub(),
  cpSync: sinon.stub(),
  constants: { R_OK: 4 },
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
  t.context.TransactionStates = t.context.module.TransactionStates;
});

test.afterEach((t) => {
  sinon.restore();
});

// Test 1: UpdateError class structure and methods
test('UpdateError has correct structure and methods', (t) => {
  const { UpdateError, UpdateErrorCodes } = t.context;

  const error = new UpdateError(
    UpdateErrorCodes.E_UPDATE_FAILED,
    'Test error message',
    { detail: 'test' },
    ['cmd1', 'cmd2']
  );

  t.is(error.name, 'UpdateError');
  t.is(error.code, UpdateErrorCodes.E_UPDATE_FAILED);
  t.is(error.message, 'Test error message');
  t.deepEqual(error.details, { detail: 'test' });
  t.deepEqual(error.recoveryCommands, ['cmd1', 'cmd2']);
  t.truthy(error.timestamp);
  t.truthy(error.stack);
});

test('UpdateError.toJSON() returns structured object', (t) => {
  const { UpdateError, UpdateErrorCodes } = t.context;

  const error = new UpdateError(
    UpdateErrorCodes.E_SYNC_FAILED,
    'Sync failed',
    { file: 'test.txt' },
    ['git stash pop']
  );

  const json = error.toJSON();
  t.is(json.error.name, 'UpdateError');
  t.is(json.error.code, UpdateErrorCodes.E_SYNC_FAILED);
  t.is(json.error.message, 'Sync failed');
  t.deepEqual(json.error.details, { file: 'test.txt' });
  t.deepEqual(json.error.recoveryCommands, ['git stash pop']);
  t.truthy(json.error.timestamp);
  t.truthy(json.error.stack);
});

test('UpdateError.toString() includes recovery commands prominently', (t) => {
  const { UpdateError, UpdateErrorCodes } = t.context;

  const error = new UpdateError(
    UpdateErrorCodes.E_VERIFY_FAILED,
    'Verification failed',
    { errors: ['hash mismatch'] },
    ['cd /repo && git stash pop', 'aether checkpoint restore chk_123']
  );

  const str = error.toString();
  t.true(str.includes('UPDATE FAILED - RECOVERY REQUIRED'));
  t.true(str.includes('cd /repo && git stash pop'));
  t.true(str.includes('aether checkpoint restore chk_123'));
  t.true(str.includes('Verification failed'));
});

// Test 2: UpdateTransaction initialization
test('UpdateTransaction initializes with correct defaults', (t) => {
  const { UpdateTransaction, TransactionStates } = t.context;

  const tx = new UpdateTransaction('/test/repo');

  t.is(tx.repoPath, '/test/repo');
  t.is(tx.state, TransactionStates.PENDING);
  t.is(tx.checkpoint, null);
  t.is(tx.syncResult, null);
  t.deepEqual(tx.errors, []);
  t.is(tx.quiet, false);
});

test('UpdateTransaction accepts options', (t) => {
  const { UpdateTransaction } = t.context;

  const tx = new UpdateTransaction('/test/repo', {
    sourceVersion: '1.1.0',
    quiet: true,
  });

  t.is(tx.sourceVersion, '1.1.0');
  t.is(tx.quiet, true);
});

// Test 3: createCheckpoint success
test.serial('createCheckpoint creates checkpoint with stash', async (t) => {
  const { UpdateTransaction } = t.context;
  const { mockFs, mockCp } = t.context;

  // Setup git repo check
  mockCp.execSync
    .withArgs('git rev-parse --git-dir', sinon.match.any)
    .returns('.git');

  // Setup git status (no dirty files)
  mockCp.execSync
    .withArgs(sinon.match(/git status --porcelain/), sinon.match.any)
    .returns('');

  const tx = new UpdateTransaction('/test/repo');
  tx.HOME = '/home/test';

  const checkpoint = await tx.createCheckpoint();

  t.truthy(checkpoint.id);
  t.true(checkpoint.id.startsWith('chk_'));
  t.truthy(checkpoint.timestamp);
  t.is(tx.checkpoint, checkpoint);
  t.is(tx.checkpoint.stashRef, null); // No dirty files, no stash
});

test.serial('createCheckpoint stashes dirty files', async (t) => {
  const { UpdateTransaction } = t.context;
  const { mockFs, mockCp } = t.context;

  // Setup targetDirs exist
  mockFs.existsSync.callsFake((p) => {
    if (p.includes('.aether') || p.includes('.claude') || p.includes('.opencode')) return true;
    return false;
  });

  // Setup git commands with callsFake to handle all commands
  // Git porcelain format: XY filename (X=index status, Y=worktree status, space=unmodified)
  // " M file1.txt" means: index=unmodified, worktree=modified, filename=file1.txt
  mockCp.execSync.callsFake((cmd) => {
    if (cmd === 'git rev-parse --git-dir') return '.git';
    if (cmd.includes('git status --porcelain')) {
      // Return in format: "XY filename" where positions 0,1 are status, position 2 is space
      return ' M .aether/aether-utils.sh\n M .claude/commands/ant/build.md';
    }
    if (cmd.includes('git stash push')) return '';
    if (cmd === 'git stash list') return 'stash@{0}: On main: aether-update-backup';
    return '';
  });

  const tx = new UpdateTransaction('/test/repo');
  tx.HOME = '/home/test';

  const checkpoint = await tx.createCheckpoint();

  t.truthy(checkpoint.stashRef);
  t.is(checkpoint.stashRef, 'stash@{0}');
  t.true(checkpoint.dirtyFiles.length === 2);
  t.true(checkpoint.dirtyFiles.includes('.aether/aether-utils.sh'));
  t.true(checkpoint.dirtyFiles.includes('.claude/commands/ant/build.md'));
});

test.serial('createCheckpoint throws UpdateError when not in git repo', async (t) => {
  const { UpdateTransaction, UpdateError, UpdateErrorCodes } = t.context;
  const { mockCp } = t.context;

  // Setup git repo check to fail
  mockCp.execSync
    .withArgs('git rev-parse --git-dir', sinon.match.any)
    .throws(new Error('Not a git repository'));

  const tx = new UpdateTransaction('/test/repo');
  tx.HOME = '/home/test';

  const error = await t.throwsAsync(tx.createCheckpoint());

  t.true(error instanceof UpdateError);
  t.is(error.code, UpdateErrorCodes.E_CHECKPOINT_FAILED);
  t.true(error.message.includes('Not in a git repository'));
});

// Test 4: syncFiles operation
test('syncFiles updates state to syncing', (t) => {
  const { UpdateTransaction, TransactionStates } = t.context;
  const { mockFs } = t.context;

  // Setup hub directories exist
  mockFs.existsSync.callsFake((p) => {
    if (p.includes('.aether')) return true;
    if (p.includes('.aether/system')) return true;
    return false;
  });

  const tx = new UpdateTransaction('/test/repo');
  tx.HOME = '/home/test';

  // Mock listFilesRecursive to return empty
  tx.listFilesRecursive = () => [];

  tx.syncFiles('1.0.0', false);

  t.is(tx.state, TransactionStates.SYNCING);
});

// Test 5: verifyIntegrity operation
test('verifyIntegrity updates state to verifying', (t) => {
  const { UpdateTransaction, TransactionStates } = t.context;
  const { mockFs } = t.context;

  mockFs.existsSync.returns(false);

  const tx = new UpdateTransaction('/test/repo');
  tx.HOME = '/home/test';

  tx.listFilesRecursive = () => [];

  const result = tx.verifyIntegrity();

  t.is(tx.state, TransactionStates.VERIFYING);
  t.true(result.valid);
  t.deepEqual(result.errors, []);
});

test('verifyIntegrity detects missing files', (t) => {
  const { UpdateTransaction } = t.context;
  const { mockFs } = t.context;

  const tx = new UpdateTransaction('/test/repo');
  tx.HOME = '/home/test';

  // Mock listFilesRecursive to return a file
  tx.listFilesRecursive = (dir) => {
    if (dir === tx.HUB_SYSTEM_DIR) return ['test.txt'];
    return [];
  };

  // Only HUB_SYSTEM_DIR exists, and destination file is missing.
  mockFs.existsSync.callsFake((p) => {
    if (p === tx.HUB_SYSTEM_DIR) return true;
    if (p === '/test/repo/.aether/test.txt') return false;
    return false;
  });

  const result = tx.verifyIntegrity();

  t.false(result.valid);
  t.is(result.errors.length, 1);
  t.true(result.errors[0].includes('Missing file'));
});

// Test 6: rollback operation
test.serial('rollback restores stash and cleans up', async (t) => {
  const { UpdateTransaction, TransactionStates } = t.context;
  const { mockFs, mockCp } = t.context;

  const tx = new UpdateTransaction('/test/repo');
  tx.HOME = '/home/test';
  tx.checkpoint = {
    id: 'chk_20260214_120000',
    stashRef: 'stash@{0}',
    timestamp: new Date().toISOString(),
  };

  mockCp.execSync.returns('');
  mockFs.existsSync.returns(true);
  mockFs.unlinkSync.returns();

  const result = await tx.rollback();

  t.true(result);
  t.is(tx.state, TransactionStates.ROLLED_BACK);

  // Verify stash pop was called
  t.true(mockCp.execSync.calledWith(
    'git stash pop stash@{0}',
    sinon.match.any
  ));

  // Verify checkpoint file was deleted
  t.true(mockFs.unlinkSync.calledWith(
    sinon.match(/chk_20260214_120000\.json/)
  ));
});

test.serial('rollback handles missing checkpoint gracefully', async (t) => {
  const { UpdateTransaction, TransactionStates } = t.context;

  const tx = new UpdateTransaction('/test/repo');
  tx.HOME = '/home/test';
  tx.checkpoint = null;

  const result = await tx.rollback();

  t.false(result);
  t.is(tx.state, TransactionStates.ROLLED_BACK);
});

// Test 7: getRecoveryCommands
test('getRecoveryCommands returns commands based on state', (t) => {
  const { UpdateTransaction } = t.context;

  const tx = new UpdateTransaction('/test/repo');
  tx.HOME = '/home/test';

  // No checkpoint
  let commands = tx.getRecoveryCommands();
  t.is(commands.length, 1);
  t.true(commands[0].includes('git reset --hard HEAD'));

  // With checkpoint
  tx.checkpoint = {
    id: 'chk_123',
    stashRef: 'stash@{0}',
  };

  commands = tx.getRecoveryCommands();
  t.is(commands.length, 3);
  t.true(commands[0].includes('git stash pop'));
  t.true(commands[1].includes('aether checkpoint restore chk_123'));
  t.true(commands[2].includes('git reset --hard HEAD'));
});

// Test 8: execute with full two-phase commit - success
test.serial('execute completes full two-phase commit on success', async (t) => {
  const { UpdateTransaction, TransactionStates } = t.context;
  const { mockFs, mockCp } = t.context;

  // Setup all git operations to succeed
  mockCp.execSync.callsFake((cmd) => {
    if (cmd === 'git rev-parse --git-dir') return '.git';
    if (cmd.includes('git status')) return '';
    if (cmd === 'git stash list') return '';
    return '';
  });

  mockFs.existsSync.returns(true);
  mockFs.readdirSync.returns([]);

  const tx = new UpdateTransaction('/test/repo');
  tx.HOME = '/home/test';

  // Mock syncFiles to set syncResult and return
  tx.syncFiles = function() {
    this.syncResult = {
      system: { copied: 5, removed: [], skipped: 0 },
      commands: { copied: 10, removed: [], skipped: 0 },
      agents: { copied: 2, removed: [], skipped: 0 },
    };
    return this.syncResult;
  };

  // Mock verifyIntegrity to pass
  tx.verifyIntegrity = () => ({ valid: true, errors: [] });

  // Mock updateVersion
  tx.updateVersion = () => {};

  const result = await tx.execute('1.1.0', { dryRun: false });

  t.true(result.success);
  t.is(result.status, 'updated');
  t.truthy(result.checkpoint_id);
  t.is(result.files_synced, 17); // 5 + 10 + 2
  t.is(tx.state, TransactionStates.COMMITTED);
});

// Test 9: execute with dry-run
test.serial('execute performs dry-run without modifying files', async (t) => {
  const { UpdateTransaction, TransactionStates } = t.context;
  const { mockCp } = t.context;

  // Setup git operations
  mockCp.execSync.callsFake((cmd) => {
    if (cmd === 'git rev-parse --git-dir') return '.git';
    if (cmd.includes('git status')) return '';
    return '';
  });

  const tx = new UpdateTransaction('/test/repo');
  tx.HOME = '/home/test';

  // Mock syncFiles
  tx.syncFiles = function() {
    this.syncResult = {
      system: { copied: 5, removed: [], skipped: 0 },
      commands: { copied: 10, removed: [], skipped: 0 },
      agents: { copied: 2, removed: [], skipped: 0 },
    };
    return this.syncResult;
  };

  // updateVersion should NOT be called in dry-run
  let versionUpdated = false;
  tx.updateVersion = () => { versionUpdated = true; };

  const result = await tx.execute('1.1.0', { dryRun: true });

  t.true(result.success);
  t.is(result.status, 'dry-run');
  t.false(versionUpdated);
});

// Test 10: execute with verification failure triggers rollback
test.serial('execute rolls back on verification failure', async (t) => {
  const { UpdateTransaction, UpdateError, UpdateErrorCodes, TransactionStates } = t.context;
  const { mockCp } = t.context;

  // Setup git operations
  mockCp.execSync.callsFake((cmd) => {
    if (cmd === 'git rev-parse --git-dir') return '.git';
    if (cmd.includes('git status')) return '';
    return '';
  });

  const tx = new UpdateTransaction('/test/repo');
  tx.HOME = '/home/test';

  // Mock syncFiles
  tx.syncFiles = function() {
    this.syncResult = {
      system: { copied: 5, removed: [], skipped: 0 },
      commands: { copied: 10, removed: [], skipped: 0 },
      agents: { copied: 2, removed: [], skipped: 0 },
    };
    return this.syncResult;
  };

  // Mock verifyIntegrity to fail
  tx.verifyIntegrity = () => ({
    valid: false,
    errors: ['Hash mismatch: file.txt'],
  });

  // Track rollback
  let rollbackCalled = false;
  tx.rollback = async () => {
    rollbackCalled = true;
    tx.state = TransactionStates.ROLLED_BACK;
    return true;
  };

  const error = await t.throwsAsync(tx.execute('1.1.0'));

  t.true(error instanceof UpdateError);
  t.is(error.code, UpdateErrorCodes.E_VERIFY_FAILED);
  t.true(rollbackCalled);
  t.true(error.message.includes('Verification failed'));
  t.true(error.recoveryCommands.length > 0);
});

// Test 11: execute with sync failure triggers rollback
test.serial('execute rolls back on sync failure', async (t) => {
  const { UpdateTransaction, UpdateError, UpdateErrorCodes, TransactionStates } = t.context;
  const { mockCp } = t.context;

  // Setup git operations
  mockCp.execSync.callsFake((cmd) => {
    if (cmd === 'git rev-parse --git-dir') return '.git';
    if (cmd.includes('git status')) return '';
    return '';
  });

  const tx = new UpdateTransaction('/test/repo');
  tx.HOME = '/home/test';

  // Mock syncFiles to throw
  tx.syncFiles = () => {
    throw new Error('Disk full');
  };

  // Track rollback
  let rollbackCalled = false;
  tx.rollback = async () => {
    rollbackCalled = true;
    tx.state = TransactionStates.ROLLED_BACK;
    return true;
  };

  const error = await t.throwsAsync(tx.execute('1.1.0'));

  t.true(error instanceof UpdateError);
  t.is(error.code, UpdateErrorCodes.E_UPDATE_FAILED);
  t.true(rollbackCalled);
  t.true(error.recoveryCommands.length > 0);
});

// Test 12: state transitions through transaction lifecycle
test.serial('execute transitions through correct states', async (t) => {
  const { UpdateTransaction, TransactionStates } = t.context;
  const { mockCp } = t.context;

  // Setup git operations
  mockCp.execSync.callsFake((cmd) => {
    if (cmd === 'git rev-parse --git-dir') return '.git';
    if (cmd.includes('git status')) return '';
    return '';
  });

  const tx = new UpdateTransaction('/test/repo');
  tx.HOME = '/home/test';

  const states = [];

  // Wrap methods to capture state changes
  const originalCreateCheckpoint = tx.createCheckpoint.bind(tx);
  tx.createCheckpoint = async () => {
    states.push(tx.state);
    return originalCreateCheckpoint();
  };

  tx.syncFiles = function() {
    states.push(tx.state);
    this.syncResult = { system: { copied: 1 }, commands: { copied: 1 }, agents: { copied: 1 } };
    return this.syncResult;
  };

  tx.verifyIntegrity = () => {
    states.push(tx.state);
    return { valid: true, errors: [] };
  };

  tx.updateVersion = () => {
    states.push(tx.state);
  };

  await tx.execute('1.1.0');

  states.push(tx.state); // Final state

  t.is(states[0], TransactionStates.PREPARING);
  t.is(states[1], TransactionStates.SYNCING);
  t.is(states[2], TransactionStates.VERIFYING);
  t.is(states[3], TransactionStates.COMMITTING);
  t.is(states[4], TransactionStates.COMMITTED);
});

// Test 13: hashFileSync helper
test('hashFileSync computes SHA-256 hash', (t) => {
  const { UpdateTransaction } = t.context;
  const { mockFs, mockCrypto } = t.context;

  mockFs.readFileSync.returns(Buffer.from('test content'));

  const tx = new UpdateTransaction('/test/repo');
  const hash = tx.hashFileSync('/test/file.txt');

  t.is(hash, 'sha256:abc123hash');
  t.true(mockCrypto.createHash.calledWith('sha256'));
});

test('hashFileSync returns null on error', (t) => {
  const { UpdateTransaction } = t.context;
  const { mockFs } = t.context;

  mockFs.readFileSync.throws(new Error('Permission denied'));

  const tx = new UpdateTransaction('/test/repo');
  const hash = tx.hashFileSync('/test/file.txt');

  t.is(hash, null);
});

// Test 14: isGitRepo helper
test('isGitRepo returns true for git repository', (t) => {
  const { UpdateTransaction } = t.context;
  const { mockCp } = t.context;

  mockCp.execSync.returns('.git');

  const tx = new UpdateTransaction('/test/repo');
  const result = tx.isGitRepo();

  t.true(result);
});

test('isGitRepo returns false for non-git directory', (t) => {
  const { UpdateTransaction } = t.context;
  const { mockCp } = t.context;

  mockCp.execSync.throws(new Error('Not a git repository'));

  const tx = new UpdateTransaction('/test/repo');
  const result = tx.isGitRepo();

  t.false(result);
});

// Test 15: readJsonSafe helper
test('readJsonSafe parses valid JSON', (t) => {
  const { UpdateTransaction } = t.context;
  const { mockFs } = t.context;

  mockFs.readFileSync.returns('{"version": "1.0.0"}');

  const tx = new UpdateTransaction('/test/repo');
  const result = tx.readJsonSafe('/test/version.json');

  t.deepEqual(result, { version: '1.0.0' });
});

test('readJsonSafe returns null for invalid JSON', (t) => {
  const { UpdateTransaction } = t.context;
  const { mockFs } = t.context;

  mockFs.readFileSync.returns('not valid json');

  const tx = new UpdateTransaction('/test/repo');
  const result = tx.readJsonSafe('/test/version.json');

  t.is(result, null);
});

// Test 16: writeJsonSync helper (atomic: write to .tmp then rename)
test('writeJsonSync writes formatted JSON atomically', (t) => {
  const { UpdateTransaction } = t.context;
  const { mockFs } = t.context;

  const tx = new UpdateTransaction('/test/repo');
  tx.writeJsonSync('/test/output.json', { test: 'data' });

  t.true(mockFs.mkdirSync.calledWith('/test', { recursive: true }));
  t.true(mockFs.writeFileSync.calledWith(
    '/test/output.json.tmp',
    '{\n  "test": "data"\n}\n'
  ));
  t.true(mockFs.renameSync.calledWith(
    '/test/output.json.tmp',
    '/test/output.json'
  ));
});

// Test 17: Error codes are defined correctly
test('UpdateErrorCodes contains all expected codes', (t) => {
  const { UpdateErrorCodes } = t.context;

  t.is(UpdateErrorCodes.E_UPDATE_FAILED, 'E_UPDATE_FAILED');
  t.is(UpdateErrorCodes.E_CHECKPOINT_FAILED, 'E_CHECKPOINT_FAILED');
  t.is(UpdateErrorCodes.E_SYNC_FAILED, 'E_SYNC_FAILED');
  t.is(UpdateErrorCodes.E_VERIFY_FAILED, 'E_VERIFY_FAILED');
  t.is(UpdateErrorCodes.E_ROLLBACK_FAILED, 'E_ROLLBACK_FAILED');
});

// Test 18: TransactionStates are defined correctly
test('TransactionStates contains all expected states', (t) => {
  const { TransactionStates } = t.context;

  t.is(TransactionStates.PENDING, 'pending');
  t.is(TransactionStates.PREPARING, 'preparing');
  t.is(TransactionStates.SYNCING, 'syncing');
  t.is(TransactionStates.VERIFYING, 'verifying');
  t.is(TransactionStates.COMMITTING, 'committing');
  t.is(TransactionStates.COMMITTED, 'committed');
  t.is(TransactionStates.ROLLING_BACK, 'rolling_back');
  t.is(TransactionStates.ROLLED_BACK, 'rolled_back');
});

// Test 19: execute() writes .update-pending before sync starts
test.serial('execute writes .update-pending sentinel before sync starts', async (t) => {
  const { UpdateTransaction } = t.context;
  const { mockFs, mockCp } = t.context;

  // Setup git operations to succeed
  mockCp.execSync.callsFake((cmd) => {
    if (cmd === 'git rev-parse --git-dir') return '.git';
    if (cmd.includes('git status')) return '';
    if (cmd === 'git stash list') return '';
    return '';
  });

  mockFs.existsSync.returns(true);
  mockFs.readdirSync.returns([]);

  const tx = new UpdateTransaction('/test/repo');
  tx.HOME = '/home/test';

  const writeOrder = [];

  // Capture write calls to track order
  mockFs.writeFileSync.callsFake((filePath) => {
    writeOrder.push(filePath);
  });

  // Mock syncFiles to record that it was called after pending write
  let pendingWrittenBeforeSync = false;
  tx.syncFiles = function() {
    // Check if .update-pending was written before sync was called
    pendingWrittenBeforeSync = writeOrder.some(p => p.includes('.update-pending'));
    this.syncResult = {
      system: { copied: 1, removed: [], skipped: 0 },
      commands: { copied: 1, removed: [], skipped: 0 },
      agents: { copied: 0, removed: [], skipped: 0 },
      rules: { copied: 0, removed: [], skipped: 0 },
    };
    return this.syncResult;
  };

  tx.verifyIntegrity = () => ({ valid: true, errors: [] });
  tx.updateVersion = () => {};

  await tx.execute('2.0.0', { dryRun: false });

  // .update-pending must have been written before sync
  t.true(pendingWrittenBeforeSync, '.update-pending should be written before sync starts');
  // Confirm the path ends in .update-pending
  t.true(writeOrder.some(p => p.includes('.update-pending')));
});

// Test 20: execute() writes .update-pending with correct target_version
test.serial('execute writes .update-pending with correct target_version', async (t) => {
  const { UpdateTransaction } = t.context;
  const { mockFs, mockCp } = t.context;

  mockCp.execSync.callsFake((cmd) => {
    if (cmd === 'git rev-parse --git-dir') return '.git';
    if (cmd.includes('git status')) return '';
    return '';
  });

  mockFs.existsSync.returns(true);
  mockFs.readdirSync.returns([]);

  const tx = new UpdateTransaction('/test/repo');
  tx.HOME = '/home/test';

  let capturedContent = null;
  mockFs.writeFileSync.callsFake((filePath, content) => {
    if (filePath.includes('.update-pending')) {
      capturedContent = content;
    }
  });

  tx.syncFiles = function() {
    this.syncResult = { system: { copied: 0, removed: [], skipped: 0 }, commands: { copied: 0, removed: [], skipped: 0 }, agents: { copied: 0, removed: [], skipped: 0 }, rules: { copied: 0, removed: [], skipped: 0 } };
    return this.syncResult;
  };
  tx.verifyIntegrity = () => ({ valid: true, errors: [] });
  tx.updateVersion = () => {};

  await tx.execute('2.0.0', { dryRun: false });

  t.truthy(capturedContent, '.update-pending content should have been written');
  const parsed = JSON.parse(capturedContent);
  t.is(parsed.target_version, '2.0.0');
  t.truthy(parsed.started_at);
});

// Test 21: execute() deletes .update-pending after successful version stamp
test.serial('execute deletes .update-pending after successful version stamp', async (t) => {
  const { UpdateTransaction } = t.context;
  const { mockFs, mockCp } = t.context;

  mockCp.execSync.callsFake((cmd) => {
    if (cmd === 'git rev-parse --git-dir') return '.git';
    if (cmd.includes('git status')) return '';
    return '';
  });

  mockFs.existsSync.returns(true);
  mockFs.readdirSync.returns([]);

  const tx = new UpdateTransaction('/test/repo');
  tx.HOME = '/home/test';

  const versionUpdatedAt = [];
  const unlinkedAfterVersion = [];
  let versionUpdateDone = false;

  tx.syncFiles = function() {
    this.syncResult = { system: { copied: 0, removed: [], skipped: 0 }, commands: { copied: 0, removed: [], skipped: 0 }, agents: { copied: 0, removed: [], skipped: 0 }, rules: { copied: 0, removed: [], skipped: 0 } };
    return this.syncResult;
  };
  tx.verifyIntegrity = () => ({ valid: true, errors: [] });
  tx.updateVersion = () => {
    versionUpdateDone = true;
    versionUpdatedAt.push(Date.now());
  };

  mockFs.unlinkSync.callsFake((filePath) => {
    if (versionUpdateDone && filePath.includes('.update-pending')) {
      unlinkedAfterVersion.push(filePath);
    }
  });

  await tx.execute('1.1.0', { dryRun: false });

  // unlinkSync was called on .update-pending after updateVersion
  t.true(unlinkedAfterVersion.length > 0, '.update-pending should be deleted after version stamp');
  t.true(unlinkedAfterVersion[0].includes('.update-pending'));
});

// Test 22: rollback() deletes .update-pending
test.serial('rollback deletes .update-pending', async (t) => {
  const { UpdateTransaction, TransactionStates } = t.context;
  const { mockFs, mockCp } = t.context;

  const tx = new UpdateTransaction('/test/repo');
  tx.HOME = '/home/test';
  tx.checkpoint = {
    id: 'chk_20260218_120000',
    stashRef: null,
    timestamp: new Date().toISOString(),
  };

  mockCp.execSync.returns('');
  mockFs.existsSync.returns(true);

  const unlinkedPaths = [];
  mockFs.unlinkSync.callsFake((filePath) => {
    unlinkedPaths.push(filePath);
  });

  await tx.rollback();

  t.is(tx.state, TransactionStates.ROLLED_BACK);
  t.true(unlinkedPaths.some(p => p.includes('.update-pending')), 'rollback should delete .update-pending');
});

// Test 23: execute() continues successfully if pending file delete fails
test.serial('execute succeeds even if pending sentinel delete fails', async (t) => {
  const { UpdateTransaction, UpdateError } = t.context;
  const { mockFs, mockCp } = t.context;

  mockCp.execSync.callsFake((cmd) => {
    if (cmd === 'git rev-parse --git-dir') return '.git';
    if (cmd.includes('git status')) return '';
    return '';
  });

  // existsSync returns true for pending path
  mockFs.existsSync.callsFake(() => true);
  mockFs.readdirSync.returns([]);

  // Make unlinkSync throw when called on the pending path
  mockFs.unlinkSync.callsFake((filePath) => {
    if (filePath.includes('.update-pending')) {
      throw new Error('Permission denied');
    }
  });

  const tx = new UpdateTransaction('/test/repo');
  tx.HOME = '/home/test';

  tx.syncFiles = function() {
    this.syncResult = { system: { copied: 0, removed: [], skipped: 0 }, commands: { copied: 0, removed: [], skipped: 0 }, agents: { copied: 0, removed: [], skipped: 0 }, rules: { copied: 0, removed: [], skipped: 0 } };
    return this.syncResult;
  };
  tx.verifyIntegrity = () => ({ valid: true, errors: [] });
  tx.updateVersion = () => {};

  // Should not throw even though unlinkSync fails on pending path
  const result = await tx.execute('1.1.0', { dryRun: false });

  t.true(result.success, 'execute should succeed even if sentinel delete fails');
  t.false(result instanceof UpdateError);
});

// ---------------------------------------------------------------------------
// Test group: Source directory fix (DIST-01)
// ---------------------------------------------------------------------------

test('syncFiles uses HUB_SYSTEM_DIR not HUB_DIR for system sync', (t) => {
  const { UpdateTransaction } = t.context;
  const { mockFs } = t.context;

  // Only return true for paths containing 'system' (HUB_SYSTEM_DIR)
  mockFs.existsSync.callsFake((p) => {
    return p.includes('/system');
  });
  mockFs.readdirSync.returns([]);

  const tx = new UpdateTransaction('/test/repo');
  tx.HOME = '/home/test';
  tx.listFilesRecursive = () => [];

  tx.syncFiles('3.1.19', false);

  // existsSync should have been called with the HUB_SYSTEM_DIR path (ends in /system)
  const systemDirChecked = mockFs.existsSync.args.some(([p]) => p.endsWith('/system'));
  t.true(systemDirChecked, 'existsSync should be called with HUB_SYSTEM_DIR path ending in /system');

  // existsSync should NOT have been called with the bare hub dir as first arg for system sync
  // (HUB_DIR = ~/.aether/, HUB_SYSTEM_DIR = ~/.aether/system/)
  // The HUB_DIR itself would end in /.aether/ but the system sync path ends in /system
  const hubDirWithoutSystem = mockFs.existsSync.args.filter(([p]) =>
    p.endsWith('/home/test/.aether') || p.endsWith('/home/test/.aether/')
  );
  t.is(hubDirWithoutSystem.length, 0, 'system sync should not use bare HUB_DIR');
});

// ---------------------------------------------------------------------------
// Test group: EXCLUDE_DIRS (DIST-02)
// ---------------------------------------------------------------------------

test('EXCLUDE_DIRS includes agents, commands, and rules', (t) => {
  const { UpdateTransaction } = t.context;

  const tx = new UpdateTransaction('/test/repo');

  t.true(tx.EXCLUDE_DIRS.includes('agents'), 'EXCLUDE_DIRS should include agents');
  t.true(tx.EXCLUDE_DIRS.includes('commands'), 'EXCLUDE_DIRS should include commands');
  t.true(tx.EXCLUDE_DIRS.includes('rules'), 'EXCLUDE_DIRS should include rules');

  // Original 5 entries still present
  t.true(tx.EXCLUDE_DIRS.includes('data'), 'EXCLUDE_DIRS should include data');
  t.true(tx.EXCLUDE_DIRS.includes('dreams'), 'EXCLUDE_DIRS should include dreams');
  t.true(tx.EXCLUDE_DIRS.includes('checkpoints'), 'EXCLUDE_DIRS should include checkpoints');
  t.true(tx.EXCLUDE_DIRS.includes('locks'), 'EXCLUDE_DIRS should include locks');
  t.true(tx.EXCLUDE_DIRS.includes('temp'), 'EXCLUDE_DIRS should include temp');
});

test('shouldExclude returns true for agents, commands, and rules paths', (t) => {
  const { UpdateTransaction } = t.context;

  const tx = new UpdateTransaction('/test/repo');

  t.true(tx.shouldExclude('agents/foo.md'), 'agents/foo.md should be excluded');
  t.true(tx.shouldExclude('commands/bar.md'), 'commands/bar.md should be excluded');
  t.true(tx.shouldExclude('rules/baz.md'), 'rules/baz.md should be excluded');
  t.false(tx.shouldExclude('docs/caste-system.md'), 'docs/caste-system.md should not be excluded');
  t.false(tx.shouldExclude('workers.md'), 'workers.md should not be excluded');
});

// ---------------------------------------------------------------------------
// Test group: Stale-dir cleanup (DIST-06)
// ---------------------------------------------------------------------------

test('cleanupStaleAetherDirs moves existing stale directories and files to trash', (t) => {
  const { UpdateTransaction } = t.context;
  const { mockFs } = t.context;

  // All three stale items exist
  mockFs.existsSync.callsFake((p) => {
    return (
      p.endsWith('.aether/agents') ||
      p.endsWith('.aether/commands') ||
      p.endsWith('.aether/planning.md') ||
      p.includes('.trash')  // trash directory creation
    );
  });
  mockFs.renameSync.returns(undefined);
  mockFs.statSync.returns({ isDirectory: () => true });

  const tx = new UpdateTransaction('/test/repo');
  const result = tx.cleanupStaleAetherDirs('/test/repo');

  // mkdirSync called for trash directory
  t.true(mockFs.mkdirSync.calledWith(
    sinon.match((p) => p.includes('.trash')),
    { recursive: true }
  ), 'mkdirSync should be called for trash directory');

  // renameSync called for all items (move to trash)
  t.true(mockFs.renameSync.calledWith(
    sinon.match((p) => p.endsWith('.aether/agents')),
    sinon.match((p) => p.includes('.trash') && p.includes('agents'))
  ), 'renameSync should be called for .aether/agents');

  t.true(mockFs.renameSync.calledWith(
    sinon.match((p) => p.endsWith('.aether/commands')),
    sinon.match((p) => p.includes('.trash') && p.includes('commands'))
  ), 'renameSync should be called for .aether/commands');

  t.true(mockFs.renameSync.calledWith(
    sinon.match((p) => p.endsWith('.aether/planning.md')),
    sinon.match((p) => p.includes('.trash') && p.includes('planning.md'))
  ), 'renameSync should be called for .aether/planning.md');

  // All three should appear in cleaned
  t.is(result.cleaned.length, 3, 'cleaned should have 3 entries');
  t.is(result.failed.length, 0, 'failed should be empty');
  t.true(result.trashDir.includes('.trash'), 'trashDir should be returned');
});

test('cleanupStaleAetherDirs is idempotent — returns empty when nothing to clean', (t) => {
  const { UpdateTransaction } = t.context;
  const { mockFs } = t.context;

  // No stale items exist
  mockFs.existsSync.returns(false);

  const tx = new UpdateTransaction('/test/repo');
  const result = tx.cleanupStaleAetherDirs('/test/repo');

  t.is(result.cleaned.length, 0, 'cleaned should be empty');
  t.is(result.failed.length, 0, 'failed should be empty');
  t.false(mockFs.renameSync.called, 'renameSync should not be called');
  t.false(mockFs.mkdirSync.calledWith(sinon.match((p) => p.includes('.trash'))), 'trash mkdir should not be called');
});

test('cleanupStaleAetherDirs handles trash move errors gracefully', (t) => {
  const { UpdateTransaction } = t.context;
  const { mockFs } = t.context;

  // Only .aether/agents exists and rename fails (both attempts)
  mockFs.existsSync.callsFake((p) => p.endsWith('.aether/agents'));
  mockFs.renameSync.throws(new Error('Permission denied'));
  mockFs.statSync.returns({ isDirectory: () => true });
  mockFs.cpSync.throws(new Error('Copy failed'));

  const tx = new UpdateTransaction('/test/repo');
  const result = tx.cleanupStaleAetherDirs('/test/repo');

  t.is(result.cleaned.length, 0, 'cleaned should be empty when trash move fails');
  t.is(result.failed.length, 1, 'failed should have 1 entry');
  t.is(result.failed[0].error, 'Failed to move to trash', 'error message should indicate trash failure');
  t.true(result.failed[0].label.includes('agents'), 'label should reference agents');
});

// ---------------------------------------------------------------------------
// Test group: writeJsonSync atomicity (ATOMIC-01)
// ---------------------------------------------------------------------------

test('writeJsonSync writes to temp file then renames atomically', (t) => {
  const { UpdateTransaction } = t.context;
  const { mockFs } = t.context;

  const tx = new UpdateTransaction('/test/repo');
  const data = { test: 'data' };
  const filePath = '/test/output.json';
  const expectedContent = JSON.stringify(data, null, 2) + '\n';

  // Track call order
  const callOrder = [];
  mockFs.writeFileSync.callsFake((p) => {
    callOrder.push({ op: 'write', path: p });
  });
  mockFs.renameSync.callsFake((from, to) => {
    callOrder.push({ op: 'rename', from, to });
  });

  tx.writeJsonSync(filePath, data);

  // Should write to a .tmp file first
  t.true(callOrder.length >= 2, 'should have at least write + rename');
  t.is(callOrder[0].op, 'write', 'first operation should be write');
  t.true(callOrder[0].path.endsWith('.tmp'), 'write should target a .tmp file');

  // Then rename temp to final path
  t.is(callOrder[1].op, 'rename', 'second operation should be rename');
  t.is(callOrder[1].to, filePath, 'rename target should be the final file path');
  t.true(callOrder[1].from.endsWith('.tmp'), 'rename source should be the .tmp file');

  // Verify correct content was written
  t.is(mockFs.writeFileSync.firstCall.args[1], expectedContent, 'content should be formatted JSON with trailing newline');
});

test('writeJsonSync cleans up temp file on write failure', (t) => {
  const { UpdateTransaction } = t.context;
  const { mockFs } = t.context;

  const tx = new UpdateTransaction('/test/repo');

  mockFs.writeFileSync.throws(new Error('Disk full'));
  mockFs.existsSync.returns(true);

  const err = t.throws(() => tx.writeJsonSync('/test/output.json', { test: 'data' }));
  t.true(err.message.includes('Disk full'));

  // Should attempt to clean up the temp file
  t.true(mockFs.unlinkSync.called, 'should attempt to clean up temp file');
  t.true(mockFs.unlinkSync.firstCall.args[0].endsWith('.tmp'), 'should clean up the .tmp file');
});

// ---------------------------------------------------------------------------
// Test group: Rollback file restoration (ROLLBACK-01)
// ---------------------------------------------------------------------------

test.serial('rollback restores backed-up managed files before popping stash', async (t) => {
  const { UpdateTransaction, TransactionStates } = t.context;
  const { mockFs, mockCp } = t.context;

  const tx = new UpdateTransaction('/test/repo');
  tx.HOME = '/home/test';
  tx.checkpoint = {
    id: 'chk_20260214_120000',
    stashRef: 'stash@{0}',
    timestamp: new Date().toISOString(),
    backedUpFiles: [
      { relPath: '.aether/workers.md', backupPath: '/test/repo/.aether/checkpoints/chk_20260214_120000_backup/.aether/workers.md' },
      { relPath: '.claude/commands/ant/build.md', backupPath: '/test/repo/.aether/checkpoints/chk_20260214_120000_backup/.claude/commands/ant/build.md' },
    ],
  };

  mockCp.execSync.returns('');
  mockFs.existsSync.returns(true);

  // Track copy operations to ensure backups are restored
  const copyOps = [];
  mockFs.copyFileSync.callsFake((src, dest) => {
    copyOps.push({ src, dest });
  });

  const callOrder = [];
  mockFs.copyFileSync.callsFake((src, dest) => {
    callOrder.push({ op: 'copy', src, dest });
  });
  mockCp.execSync.callsFake((cmd) => {
    callOrder.push({ op: 'exec', cmd });
    return '';
  });

  const result = await tx.rollback();

  t.true(result);
  t.is(tx.state, TransactionStates.ROLLED_BACK);

  // Backup files should be restored
  const restoreOps = callOrder.filter(o => o.op === 'copy');
  t.is(restoreOps.length, 2, 'should restore 2 backed-up files');

  // Stash pop should happen AFTER file restoration
  const stashPopIndex = callOrder.findIndex(o => o.op === 'exec' && o.cmd.includes('git stash pop'));
  const lastCopyIndex = callOrder.reduce((max, o, i) => o.op === 'copy' ? i : max, -1);
  t.true(stashPopIndex > lastCopyIndex, 'stash pop should happen after file restoration');
});

test.serial('rollback skips file restoration when no backedUpFiles in checkpoint', async (t) => {
  const { UpdateTransaction, TransactionStates } = t.context;
  const { mockFs, mockCp } = t.context;

  const tx = new UpdateTransaction('/test/repo');
  tx.HOME = '/home/test';
  tx.checkpoint = {
    id: 'chk_20260214_120000',
    stashRef: 'stash@{0}',
    timestamp: new Date().toISOString(),
    // No backedUpFiles property
  };

  mockCp.execSync.returns('');
  mockFs.existsSync.returns(true);

  const result = await tx.rollback();

  t.true(result);
  t.is(tx.state, TransactionStates.ROLLED_BACK);
  // copyFileSync should NOT be called for backup restoration
  t.false(mockFs.copyFileSync.called, 'should not call copyFileSync when no backedUpFiles');
});
