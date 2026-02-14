/**
 * Unit Tests for hashFileSync function
 *
 * Tests the SHA-256 hashing functionality in bin/cli.js using
 * sinon stubs and proxyquire for fs module mocking.
 *
 * @module tests/unit/cli-hash
 */

const test = require('ava');
const sinon = require('sinon');
const proxyquire = require('proxyquire');
const crypto = require('crypto');

// Load the module once with mocked fs
let mockFs;
let cli;

test.before(() => {
  // Create mock fs with sinon stubs
  mockFs = {
    readFileSync: sinon.stub()
  };

  // Load cli.js with mocked fs
  cli = proxyquire('../../bin/cli.js', {
    fs: mockFs
  });
});

test.afterEach(() => {
  // Reset stubs between tests
  mockFs.readFileSync.reset();
});

test.after(() => {
  sinon.restore();
});

/**
 * Test: hashFileSync returns correct SHA-256 hash for file content
 */
test('hashFileSync returns correct SHA-256 hash for file content', t => {
  const content = 'hello world';
  const expectedHash = 'sha256:' + crypto.createHash('sha256').update(content).digest('hex');

  // Mock readFileSync to return content
  mockFs.readFileSync.withArgs('/test/file.txt').returns(Buffer.from(content));

  // Call hashFileSync
  const result = cli.hashFileSync('/test/file.txt');

  // Verify result matches expected hash
  t.is(result, expectedHash);
  t.true(mockFs.readFileSync.calledOnceWith('/test/file.txt'));
});

/**
 * Test: hashFileSync returns consistent hash for same content
 */
test('hashFileSync returns consistent hash for same content', t => {
  const content = 'test content for consistency';

  // Mock readFileSync to return same content for both calls
  mockFs.readFileSync.returns(Buffer.from(content));

  // Call hashFileSync twice
  const hash1 = cli.hashFileSync('/test/file1.txt');
  const hash2 = cli.hashFileSync('/test/file2.txt');

  // Verify both calls return identical hashes
  t.is(hash1, hash2);
  t.true(hash1.startsWith('sha256:'));
});

/**
 * Test: hashFileSync returns null for missing file (ENOENT)
 */
test('hashFileSync returns null for missing file', t => {
  // Mock readFileSync to throw ENOENT error
  const error = new Error("ENOENT: no such file or directory, open '/test/missing.txt'");
  error.code = 'ENOENT';
  mockFs.readFileSync.throws(error);

  // Call hashFileSync
  const result = cli.hashFileSync('/test/missing.txt');

  // Verify returns null
  t.is(result, null);
});

/**
 * Test: hashFileSync returns null for permission error (EACCES)
 */
test('hashFileSync returns null for permission error', t => {
  // Mock readFileSync to throw EACCES error
  const error = new Error("EACCES: permission denied, open '/test/protected.txt'");
  error.code = 'EACCES';
  mockFs.readFileSync.throws(error);

  // Call hashFileSync
  const result = cli.hashFileSync('/test/protected.txt');

  // Verify returns null
  t.is(result, null);
});

/**
 * Test: hashFileSync handles empty file
 */
test('hashFileSync handles empty file', t => {
  // Expected hash for empty content
  const expectedHash = 'sha256:' + crypto.createHash('sha256').update('').digest('hex');

  // Mock readFileSync to return empty Buffer
  mockFs.readFileSync.returns(Buffer.from(''));

  // Call hashFileSync
  const result = cli.hashFileSync('/test/empty.txt');

  // Verify returns valid hash of empty content
  t.is(result, expectedHash);
  t.true(result.startsWith('sha256:'));
});

/**
 * Test: hashFileSync handles binary content
 */
test('hashFileSync handles binary content', t => {
  // Create binary content
  const binaryContent = Buffer.from([0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD, 0xFC]);
  const expectedHash = 'sha256:' + crypto.createHash('sha256').update(binaryContent).digest('hex');

  // Mock readFileSync to return binary content
  mockFs.readFileSync.returns(binaryContent);

  // Call hashFileSync
  const result = cli.hashFileSync('/test/binary.bin');

  // Verify returns valid hash
  t.is(result, expectedHash);
  t.true(result.startsWith('sha256:'));
});

/**
 * Test: hashFileSync returns hash in correct format
 */
test('hashFileSync returns hash in correct format', t => {
  const content = 'test content';

  // Mock readFileSync
  mockFs.readFileSync.returns(Buffer.from(content));

  // Call hashFileSync
  const result = cli.hashFileSync('/test/file.txt');

  // Verify format: starts with 'sha256:' and has 64 hex characters
  t.true(result.startsWith('sha256:'), 'Hash should start with "sha256:" prefix');

  const hexPart = result.slice(7); // Remove 'sha256:' prefix
  t.is(hexPart.length, 64, 'Hex portion should be 64 characters');
  t.true(/^[a-f0-9]+$/.test(hexPart), 'Hex portion should contain only lowercase hex characters');
});

/**
 * Test: hashFileSync handles various string contents correctly
 */
test('hashFileSync handles various string contents', t => {
  const testCases = [
    { content: 'simple text', desc: 'simple text' },
    { content: 'Unicode: \u4f60\u597d\u4e16\u754c \ud83c\udf0d', desc: 'unicode text' },
    { content: 'Special chars: !@#$%^&*()_+-=[]{}|;\':",./<>?', desc: 'special characters' },
    { content: 'Multi\nLine\nContent\n', desc: 'multiline content' },
    { content: 'Tabs\tand\tspaces', desc: 'tabs and spaces' }
  ];

  for (const testCase of testCases) {
    mockFs.readFileSync.reset();
    mockFs.readFileSync.returns(Buffer.from(testCase.content));

    const result = cli.hashFileSync('/test/file.txt');

    // Verify format
    t.true(result.startsWith('sha256:'), `${testCase.desc}: should start with 'sha256:'`);

    // Verify it's a valid hash by checking it matches crypto computation
    const expectedHash = 'sha256:' + crypto.createHash('sha256').update(testCase.content).digest('hex');
    t.is(result, expectedHash, `${testCase.desc}: hash should match crypto computation`);
  }
});

/**
 * Test: hashFileSync handles generic errors gracefully
 */
test('hashFileSync handles generic errors gracefully', t => {
  // Mock readFileSync to throw generic error
  const error = new Error('Some unexpected error');
  mockFs.readFileSync.throws(error);

  // Call hashFileSync - should not throw
  const result = cli.hashFileSync('/test/file.txt');

  // Verify returns null instead of throwing
  t.is(result, null);
});
