/**
 * Unit Tests for syncDirWithCleanup function
 *
 * Comprehensive tests for directory synchronization with cleanup,
 * hash-based skipping, and idempotency guarantees.
 *
 * Uses a test helper approach where we load the module once
 * and use a shared mock state that can be reset between tests.
 *
 * @module tests/unit/cli-sync.test
 */

const test = require('ava');
const proxyquire = require('proxyquire');
const crypto = require('crypto');

// Helper to compute SHA256 hash
function computeHash(content) {
  return 'sha256:' + crypto.createHash('sha256').update(content).digest('hex');
}

// Helper to create mock Dirent object
function createMockDirent(name, isDirectory = false) {
  return {
    name,
    isDirectory: () => isDirectory,
    isFile: () => !isDirectory,
    isBlockDevice: () => false,
    isCharacterDevice: () => false,
    isFIFO: () => false,
    isSocket: () => false,
    isSymbolicLink: () => false
  };
}

// Create a single shared mock state that will be reset between tests
const sharedMockState = {
  files: {}, // Map of path -> content (Buffer)
  directories: new Set(), // Set of directory paths
  copyCalls: [],
  chmodCalls: [],
  unlinkCalls: [],
  copyErrorPath: null // Path that should throw on copy
};

// Create mock implementations that use shared state
const mockFs = {
  existsSync: (p) => {
    return sharedMockState.files.hasOwnProperty(p) || sharedMockState.directories.has(p);
  },

  readFileSync: (p, options) => {
    if (!sharedMockState.files.hasOwnProperty(p)) {
      const error = new Error(`ENOENT: no such file or directory, open '${p}'`);
      error.code = 'ENOENT';
      throw error;
    }
    const content = sharedMockState.files[p];
    if (typeof options === 'string' && options === 'utf8') {
      return content.toString('utf8');
    }
    return content;
  },

  writeFileSync: (p, data) => {
    sharedMockState.files[p] = Buffer.isBuffer(data) ? data : Buffer.from(data);
  },

  mkdirSync: (p, options) => {
    sharedMockState.directories.add(p);
    // If recursive, also add all parent directories
    if (options?.recursive) {
      let parent = p;
      while (parent !== '/' && parent !== '') {
        parent = parent.substring(0, parent.lastIndexOf('/')) || '/';
        if (parent !== '/' && parent !== '') {
          sharedMockState.directories.add(parent);
        }
      }
    }
  },

  readdirSync: (p, options) => {
    if (!sharedMockState.directories.has(p)) {
      const error = new Error(`ENOTDIR: not a directory, scandir '${p}'`);
      error.code = 'ENOTDIR';
      throw error;
    }

    const entries = [];

    // Find all files in this directory
    for (const filePath of Object.keys(sharedMockState.files)) {
      const dir = filePath.substring(0, filePath.lastIndexOf('/')) || '/';
      if (dir === p || (p === '/' && dir === '')) {
        const name = filePath.substring(filePath.lastIndexOf('/') + 1);
        if (options?.withFileTypes) {
          entries.push(createMockDirent(name, false));
        } else {
          entries.push(name);
        }
      }
    }

    // Find all subdirectories
    for (const dirPath of sharedMockState.directories) {
      if (dirPath !== p) {
        const parent = dirPath.substring(0, dirPath.lastIndexOf('/')) || '/';
        if (parent === p) {
          const name = dirPath.substring(dirPath.lastIndexOf('/') + 1);
          if (options?.withFileTypes) {
            entries.push(createMockDirent(name, true));
          } else {
            entries.push(name);
          }
        }
      }
    }

    return entries;
  },

  rmdirSync: (p) => {
    sharedMockState.directories.delete(p);
  },

  copyFileSync: (src, dest) => {
    sharedMockState.copyCalls.push({ src, dest });
    if (sharedMockState.copyErrorPath && src.includes(sharedMockState.copyErrorPath)) {
      const error = new Error('Permission denied');
      error.code = 'EACCES';
      throw error;
    }
    if (!sharedMockState.files.hasOwnProperty(src)) {
      const error = new Error(`ENOENT: no such file or directory, copyfile '${src}'`);
      error.code = 'ENOENT';
      throw error;
    }
    sharedMockState.files[dest] = Buffer.from(sharedMockState.files[src]);
  },

  unlinkSync: (p) => {
    sharedMockState.unlinkCalls.push(p);
    if (!sharedMockState.files.hasOwnProperty(p)) {
      const error = new Error(`ENOENT: no such file or directory, unlink '${p}'`);
      error.code = 'ENOENT';
      throw error;
    }
    delete sharedMockState.files[p];
  },

  chmodSync: (p, mode) => {
    sharedMockState.chmodCalls.push({ path: p, mode });
  },

  constants: {
    F_OK: 0,
    R_OK: 4,
    W_OK: 2,
    X_OK: 1
  }
};

const mockPath = {
  join: (...parts) => {
    const filtered = parts.filter(p => p !== '');
    if (filtered.length === 0) return '/';
    const joined = filtered.join('/').replace(/\/+/g, '/');
    return joined.startsWith('/') ? joined : '/' + joined;
  },

  dirname: (p) => {
    const parts = p.split('/').filter(Boolean);
    parts.pop();
    return parts.length === 0 ? '/' : '/' + parts.join('/');
  },

  relative: (base, full) => {
    const basePath = base.replace(/\/$/, '');
    const fullPath = full.replace(/\/$/, '');
    if (fullPath.startsWith(basePath + '/')) {
      return fullPath.substring(basePath.length + 1);
    }
    return fullPath.startsWith('/') ? fullPath.substring(1) : fullPath;
  },

  resolve: (...parts) => {
    const joined = parts.join('/').replace(/\/+/g, '/');
    return joined.startsWith('/') ? joined : '/' + joined;
  },

  sep: '/'
};

const mockChildProcess = {
  execSync: () => {
    throw new Error('Not a git repo');
  }
};

// Load CLI module once with mocks
const cli = proxyquire('../../bin/cli.js', {
  fs: mockFs,
  path: mockPath,
  child_process: mockChildProcess
});

// Reset function to clear state between tests
function resetMockState() {
  sharedMockState.files = {};
  sharedMockState.directories.clear();
  sharedMockState.copyCalls = [];
  sharedMockState.chmodCalls = [];
  sharedMockState.unlinkCalls = [];
  sharedMockState.copyErrorPath = null;
}

test.beforeEach(() => {
  resetMockState();
});

// Test 1: syncDirWithCleanup copies all files from source to destination
test.serial('syncDirWithCleanup copies all files from source to destination', t => {
  // Setup: source has 2 files
  sharedMockState.directories.add('/source');
  sharedMockState.files['/source/file1.txt'] = Buffer.from('content of file1');
  sharedMockState.files['/source/file2.txt'] = Buffer.from('content of file2');

  // Run sync
  const result = cli.syncDirWithCleanup('/source', '/dest');

  // Verify files were copied
  t.true(sharedMockState.files.hasOwnProperty('/dest/file1.txt'));
  t.true(sharedMockState.files.hasOwnProperty('/dest/file2.txt'));
  t.is(sharedMockState.files['/dest/file1.txt'].toString(), 'content of file1');
  t.is(sharedMockState.files['/dest/file2.txt'].toString(), 'content of file2');

  // Verify result
  t.is(result.copied, 2);
  t.is(result.skipped, 0);
});

// Test 2: syncDirWithCleanup creates destination directory if missing
test.serial('syncDirWithCleanup creates destination directory if missing', t => {
  // Setup: empty source
  sharedMockState.directories.add('/source');

  // Run sync
  cli.syncDirWithCleanup('/source', '/dest');

  // Verify dest directory was created
  t.true(sharedMockState.directories.has('/dest'));
});

// Test 3: syncDirWithCleanup skips files with matching hashes (idempotent)
test.serial('syncDirWithCleanup skips files with matching hashes (idempotent)', t => {
  const fileContent = 'same content';

  // Setup: source and dest both have same file with same content
  sharedMockState.directories.add('/source');
  sharedMockState.directories.add('/dest');
  sharedMockState.files['/source/file.txt'] = Buffer.from(fileContent);
  sharedMockState.files['/dest/file.txt'] = Buffer.from(fileContent);

  // Run sync
  const result = cli.syncDirWithCleanup('/source', '/dest');

  // Verify file was NOT copied (skipped due to hash match)
  t.is(sharedMockState.copyCalls.length, 0);

  // Verify result
  t.is(result.copied, 0);
  t.is(result.skipped, 1);
});

// Test 4: syncDirWithCleanup copies files with different hashes
test.serial('syncDirWithCleanup copies files with different hashes', t => {
  // Setup: source and dest both have file with different content
  sharedMockState.directories.add('/source');
  sharedMockState.directories.add('/dest');
  sharedMockState.files['/source/file.txt'] = Buffer.from('source content');
  sharedMockState.files['/dest/file.txt'] = Buffer.from('different content');

  // Run sync
  const result = cli.syncDirWithCleanup('/source', '/dest');

  // Verify file was updated
  t.is(sharedMockState.files['/dest/file.txt'].toString(), 'source content');

  // Verify result
  t.is(result.copied, 1);
  t.is(result.skipped, 0);
});

// Test 5: syncDirWithCleanup removes files in destination not in source
test.serial('syncDirWithCleanup removes files in destination not in source', t => {
  // Setup: source has file1, dest has file1 and file2
  sharedMockState.directories.add('/source');
  sharedMockState.directories.add('/dest');
  sharedMockState.files['/source/file1.txt'] = Buffer.from('content');
  sharedMockState.files['/dest/file1.txt'] = Buffer.from('content');
  sharedMockState.files['/dest/file2.txt'] = Buffer.from('old content');

  // Run sync
  const result = cli.syncDirWithCleanup('/source', '/dest');

  // Verify file2 was removed
  t.false(sharedMockState.files.hasOwnProperty('/dest/file2.txt'));
  t.true(sharedMockState.files.hasOwnProperty('/dest/file1.txt'));

  // Verify result
  t.deepEqual(result.removed, ['file2.txt']);
});

// Test 6: syncDirWithCleanup handles dry-run mode
test.serial('syncDirWithCleanup handles dry-run mode', t => {
  // Setup: source has 2 files
  sharedMockState.directories.add('/source');
  sharedMockState.files['/source/file1.txt'] = Buffer.from('content1');
  sharedMockState.files['/source/file2.txt'] = Buffer.from('content2');

  // Run sync with dryRun option
  const result = cli.syncDirWithCleanup('/source', '/dest', { dryRun: true });

  // Verify files were NOT copied
  t.false(sharedMockState.files.hasOwnProperty('/dest/file1.txt'));
  t.false(sharedMockState.files.hasOwnProperty('/dest/file2.txt'));

  // Verify result reports files that would be copied
  t.is(result.copied, 2);
});

// Test 7: syncDirWithCleanup sets executable permissions on .sh files
test.serial('syncDirWithCleanup sets executable permissions on .sh files', t => {
  // Setup: source has script.sh
  sharedMockState.directories.add('/source');
  sharedMockState.files['/source/script.sh'] = Buffer.from('#!/bin/bash');

  // Run sync
  cli.syncDirWithCleanup('/source', '/dest');

  // Verify chmod was called with 0o755
  t.is(sharedMockState.chmodCalls.length, 1);
  t.is(sharedMockState.chmodCalls[0].path, '/dest/script.sh');
  t.is(sharedMockState.chmodCalls[0].mode, 0o755);
});

// Test 8: syncDirWithCleanup creates nested directories
test.serial('syncDirWithCleanup creates nested directories', t => {
  // Setup: source has nested structure
  sharedMockState.directories.add('/source');
  sharedMockState.directories.add('/source/dir');
  sharedMockState.directories.add('/source/dir/subdir');
  sharedMockState.files['/source/dir/subdir/file.txt'] = Buffer.from('deep content');

  // Run sync
  cli.syncDirWithCleanup('/source', '/dest');

  // Verify nested directories were created
  t.true(sharedMockState.directories.has('/dest'));
  t.true(sharedMockState.directories.has('/dest/dir'));
  t.true(sharedMockState.directories.has('/dest/dir/subdir'));

  // Verify file was copied
  t.true(sharedMockState.files.hasOwnProperty('/dest/dir/subdir/file.txt'));
});

// Test 9: syncDirWithCleanup handles copy errors gracefully
test.serial('syncDirWithCleanup handles copy errors gracefully', t => {
  // Setup: source has 2 files
  sharedMockState.directories.add('/source');
  sharedMockState.files['/source/file1.txt'] = Buffer.from('content1');
  sharedMockState.files['/source/file2.txt'] = Buffer.from('content2');

  // Set up copy to throw for file1
  sharedMockState.copyErrorPath = 'file1.txt';

  // Run sync
  const result = cli.syncDirWithCleanup('/source', '/dest');

  // Verify file2 was copied despite file1 error
  t.true(sharedMockState.files.hasOwnProperty('/dest/file2.txt'));

  // Verify result - file1 skipped due to error, file2 copied
  t.is(result.copied, 1);
  t.is(result.skipped, 1);
});

// Test 10: syncDirWithCleanup is idempotent - second run copies nothing
test.serial('syncDirWithCleanup is idempotent - second run copies nothing', t => {
  const file1Content = 'content1';
  const file2Content = 'content2';
  const file3Content = 'content3';

  // Setup: source has 3 files
  sharedMockState.directories.add('/source');
  sharedMockState.files['/source/file1.txt'] = Buffer.from(file1Content);
  sharedMockState.files['/source/file2.txt'] = Buffer.from(file2Content);
  sharedMockState.files['/source/file3.txt'] = Buffer.from(file3Content);

  // First sync
  const result1 = cli.syncDirWithCleanup('/source', '/dest');
  t.is(result1.copied, 3);
  t.is(result1.skipped, 0);

  // Clear copy calls tracking
  sharedMockState.copyCalls = [];

  // Second sync - should copy nothing
  const result2 = cli.syncDirWithCleanup('/source', '/dest');

  // Verify no files copied on second run
  t.is(sharedMockState.copyCalls.length, 0);
  t.is(result2.copied, 0);
  t.is(result2.skipped, 3);
});

// Test 11: syncDirWithCleanup is idempotent - cleanup removes nothing on second run
test.serial('syncDirWithCleanup is idempotent - cleanup removes nothing on second run', t => {
  const content = 'content';

  // Setup: same files in source and dest
  sharedMockState.directories.add('/source');
  sharedMockState.directories.add('/dest');
  sharedMockState.files['/source/file1.txt'] = Buffer.from(content);
  sharedMockState.files['/source/file2.txt'] = Buffer.from(content);
  sharedMockState.files['/dest/file1.txt'] = Buffer.from(content);
  sharedMockState.files['/dest/file2.txt'] = Buffer.from(content);

  // Run sync
  const result = cli.syncDirWithCleanup('/source', '/dest');

  // Verify no files removed
  t.is(sharedMockState.unlinkCalls.length, 0);
  t.deepEqual(result.removed, []);
});

// Test 12: syncDirWithCleanup cleans up empty directories after removal
test.serial('syncDirWithCleanup cleans up empty directories after removal', t => {
  // Setup: source is empty, dest has dir/file.txt
  sharedMockState.directories.add('/source');
  sharedMockState.directories.add('/dest');
  sharedMockState.directories.add('/dest/dir');
  sharedMockState.files['/dest/dir/file.txt'] = Buffer.from('old content');

  // Run sync
  cli.syncDirWithCleanup('/source', '/dest');

  // Verify file was unlinked
  t.is(sharedMockState.unlinkCalls.length, 1);
  t.is(sharedMockState.unlinkCalls[0], '/dest/dir/file.txt');
  t.false(sharedMockState.files.hasOwnProperty('/dest/dir/file.txt'));
});

// Test 13: syncDirWithCleanup handles nested file paths correctly
test.serial('syncDirWithCleanup handles nested file paths correctly', t => {
  // Setup: nested structure in source
  sharedMockState.directories.add('/source');
  sharedMockState.directories.add('/source/level1');
  sharedMockState.directories.add('/source/level1/level2');
  sharedMockState.files['/source/level1/level2/deep.txt'] = Buffer.from('deep content');

  // Run sync
  const result = cli.syncDirWithCleanup('/source', '/dest');

  // Verify file was copied to correct nested path
  t.true(sharedMockState.files.hasOwnProperty('/dest/level1/level2/deep.txt'));
  t.is(sharedMockState.files['/dest/level1/level2/deep.txt'].toString(), 'deep content');
  t.is(result.copied, 1);
});

// Test 14: syncDirWithCleanup handles empty source directory
test.serial('syncDirWithCleanup handles empty source directory', t => {
  // Setup: empty source, dest has old file
  sharedMockState.directories.add('/source');
  sharedMockState.directories.add('/dest');
  sharedMockState.files['/dest/old.txt'] = Buffer.from('old content');

  // Run sync
  const result = cli.syncDirWithCleanup('/source', '/dest');

  // Verify old file was removed
  t.false(sharedMockState.files.hasOwnProperty('/dest/old.txt'));
  t.deepEqual(result.removed, ['old.txt']);
  t.is(result.copied, 0);
});

// Test 15: syncDirWithCleanup returns accurate counts
test.serial('syncDirWithCleanup returns accurate counts', t => {
  const content = 'content';

  // Setup: mix of new, existing, and removed files
  sharedMockState.directories.add('/source');
  sharedMockState.directories.add('/dest');
  sharedMockState.files['/source/new.txt'] = Buffer.from('new content');
  sharedMockState.files['/source/existing.txt'] = Buffer.from(content);
  sharedMockState.files['/dest/existing.txt'] = Buffer.from(content);
  sharedMockState.files['/dest/removed.txt'] = Buffer.from('old');

  // Run sync
  const result = cli.syncDirWithCleanup('/source', '/dest');

  // Verify counts
  t.is(result.copied, 1);  // new.txt
  t.is(result.skipped, 1); // existing.txt
  t.deepEqual(result.removed, ['removed.txt']);
});
