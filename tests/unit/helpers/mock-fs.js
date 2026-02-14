/**
 * Mock Filesystem Helper
 *
 * Reusable fs mocking utilities for unit testing CLI functions.
 * Uses sinon stubs to mock fs module methods.
 *
 * @module tests/unit/helpers/mock-fs
 */

const sinon = require('sinon');

/**
 * Creates a mock fs object with sinon stubs for all common methods.
 * Each stub can be configured independently in tests.
 *
 * @returns {Object} Mock fs object with sinon stubs
 */
function createMockFs() {
  return {
    // File existence
    existsSync: sinon.stub(),

    // File reading
    readFileSync: sinon.stub(),

    // File writing
    writeFileSync: sinon.stub(),

    // File descriptor operations
    openSync: sinon.stub(),
    closeSync: sinon.stub(),

    // Directory operations
    mkdirSync: sinon.stub(),
    readdirSync: sinon.stub(),
    rmdirSync: sinon.stub(),

    // File operations
    copyFileSync: sinon.stub(),
    unlinkSync: sinon.stub(),
    renameSync: sinon.stub(),

    // File stats
    statSync: sinon.stub(),

    // Permissions
    chmodSync: sinon.stub(),

    // Constants
    constants: {
      F_OK: 0,
      R_OK: 4,
      W_OK: 2,
      X_OK: 1
    }
  };
}

/**
 * Creates a mock Dirent object for readdirSync with withFileTypes option.
 *
 * @param {string} name - File or directory name
 * @param {boolean} isDirectory - Whether this represents a directory
 * @returns {Object} Mock Dirent object
 */
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

/**
 * Resets all stubs on a mock fs object.
 * Call this between tests to ensure clean state.
 *
 * @param {Object} mockFs - The mock fs object from createMockFs()
 */
function resetMockFs(mockFs) {
  Object.values(mockFs).forEach(stub => {
    if (stub && typeof stub.resetBehavior === 'function') {
      stub.resetBehavior();
    }
    if (stub && typeof stub.resetHistory === 'function') {
      stub.resetHistory();
    }
  });
}

/**
 * Convenience function to set up mock file contents.
 * Automatically configures stubs for existsSync, readFileSync, statSync, and readdirSync.
 *
 * @param {Object} mockFs - The mock fs object from createMockFs()
 * @param {Object} fileMap - Map of paths to contents
 *   - File: { '/path/to/file.txt': 'file content' }
 *   - Directory: { '/path/to/dir': null }
 *   - Buffer content: { '/path/to/binary': Buffer.from([0x00, 0x01]) }
 *
 * @example
 * setupMockFiles(mockFs, {
 *   '/test/file.txt': 'content',
 *   '/test/dir': null,  // directory
 *   '/test/data.bin': Buffer.from([0x00, 0x01])
 * });
 */
function setupMockFiles(mockFs, fileMap) {
  const files = Object.entries(fileMap);
  const directories = new Set();

  // Collect all parent directories
  files.forEach(([path, content]) => {
    const parts = path.split('/').filter(Boolean);
    let currentPath = '';
    for (let i = 0; i < parts.length - 1; i++) {
      currentPath += '/' + parts[i];
      directories.add(currentPath);
    }
    if (content === null) {
      directories.add(path);
    }
  });

  // Configure existsSync
  mockFs.existsSync.callsFake((path) => {
    return path in fileMap || directories.has(path);
  });

  // Configure readFileSync
  mockFs.readFileSync.callsFake((path, options) => {
    if (!(path in fileMap)) {
      const error = new Error(`ENOENT: no such file or directory, open '${path}'`);
      error.code = 'ENOENT';
      throw error;
    }

    const content = fileMap[path];

    // Return null for directories
    if (content === null) {
      const error = new Error(`EISDIR: illegal operation on a directory, read '${path}'`);
      error.code = 'EISDIR';
      throw error;
    }

    // Handle encoding options
    const encoding = typeof options === 'string' ? options : options?.encoding;

    if (Buffer.isBuffer(content)) {
      if (encoding) {
        return content.toString(encoding);
      }
      return content;
    }

    if (encoding) {
      return content;
    }

    return Buffer.from(content);
  });

  // Configure statSync
  mockFs.statSync.callsFake((path) => {
    const isDir = directories.has(path) || fileMap[path] === null;

    return {
      isFile: () => !isDir,
      isDirectory: () => isDir,
      isBlockDevice: () => false,
      isCharacterDevice: () => false,
      isFIFO: () => false,
      isSocket: () => false,
      isSymbolicLink: () => false,
      size: isDir ? 0 : (fileMap[path]?.length || 0),
      mtime: new Date(),
      atime: new Date(),
      ctime: new Date(),
      birthtime: new Date()
    };
  });

  // Configure readdirSync
  mockFs.readdirSync.callsFake((path, options) => {
    const isDir = directories.has(path) || fileMap[path] === null;

    if (!isDir) {
      const error = new Error(`ENOTDIR: not a directory, scandir '${path}'`);
      error.code = 'ENOTDIR';
      throw error;
    }

    const entries = files
      .filter(([filePath]) => {
        const parentDir = filePath.substring(0, filePath.lastIndexOf('/')) || '/';
        return parentDir === path;
      })
      .map(([filePath, content]) => {
        const name = filePath.substring(filePath.lastIndexOf('/') + 1);
        if (options?.withFileTypes) {
          return createMockDirent(name, content === null);
        }
        return name;
      });

    return entries;
  });

  // Configure mkdirSync to not throw for existing directories
  mockFs.mkdirSync.callsFake((path, options) => {
    if (directories.has(path) && !options?.recursive) {
      const error = new Error(`EEXIST: file already exists, mkdir '${path}'`);
      error.code = 'EEXIST';
      throw error;
    }
    // Simulate successful creation
    directories.add(path);
  });

  // Configure writeFileSync to simulate writing
  mockFs.writeFileSync.callsFake((path, data, options) => {
    // Simulate successful write - in real tests, verify with sinon assertions
    // e.g., mockFs.writeFileSync.calledWith('/path', 'content')
  });

  // Configure copyFileSync to simulate copying
  mockFs.copyFileSync.callsFake((src, dest) => {
    if (!(src in fileMap)) {
      const error = new Error(`ENOENT: no such file or directory, copyfile '${src}'`);
      error.code = 'ENOENT';
      throw error;
    }
  });

  // Configure unlinkSync to simulate deletion
  mockFs.unlinkSync.callsFake((path) => {
    if (!(path in fileMap)) {
      const error = new Error(`ENOENT: no such file or directory, unlink '${path}'`);
      error.code = 'ENOENT';
      throw error;
    }
  });

  // Configure rmdirSync to simulate directory removal
  mockFs.rmdirSync.callsFake((path) => {
    if (!directories.has(path)) {
      const error = new Error(`ENOENT: no such file or directory, rmdir '${path}'`);
      error.code = 'ENOENT';
      throw error;
    }
  });

  // Configure chmodSync to simulate permission changes
  mockFs.chmodSync.callsFake((path, mode) => {
    if (!(path in fileMap) && !directories.has(path)) {
      const error = new Error(`ENOENT: no such file or directory, chmod '${path}'`);
      error.code = 'ENOENT';
      throw error;
    }
  });
}

module.exports = {
  createMockFs,
  createMockDirent,
  resetMockFs,
  setupMockFiles
};
