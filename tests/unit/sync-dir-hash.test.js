#!/usr/bin/env node
// Test file for syncDirWithCleanup hash comparison feature

const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const crypto = require('crypto');

// Mock hashFileSync (same as in cli.js)
function hashFileSync(filePath) {
  const content = fs.readFileSync(filePath);
  return 'sha256:' + crypto.createHash('sha256').update(content).digest('hex');
}

// Copy the listFilesRecursive function from cli.js
function listFilesRecursive(dir, base) {
  base = base || dir;
  const results = [];
  if (!fs.existsSync(dir)) return results;
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  for (const entry of entries) {
    if (entry.name.startsWith('.')) continue;
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      results.push(...listFilesRecursive(fullPath, base));
    } else {
      results.push(path.relative(base, fullPath));
    }
  }
  return results;
}

// Current syncDirWithCleanup implementation (BEFORE fix)
function syncDirWithCleanupOld(src, dest, opts) {
  opts = opts || {};
  const dryRun = opts.dryRun || false;
  fs.mkdirSync(dest, { recursive: true });

  // Copy phase
  let copied = 0;
  const srcFiles = listFilesRecursive(src);
  if (!dryRun) {
    for (const relPath of srcFiles) {
      const srcPath = path.join(src, relPath);
      const destPath = path.join(dest, relPath);
      fs.mkdirSync(path.dirname(destPath), { recursive: true });
      fs.copyFileSync(srcPath, destPath);
      if (relPath.endsWith('.sh')) {
        fs.chmodSync(destPath, 0o755);
      }
      copied++;
    }
  } else {
    copied = srcFiles.length;
  }

  // Cleanup phase
  const destFiles = listFilesRecursive(dest);
  const srcSet = new Set(srcFiles);
  const removed = [];
  for (const relPath of destFiles) {
    if (!srcSet.has(relPath)) {
      removed.push(relPath);
      if (!dryRun) {
        fs.unlinkSync(path.join(dest, relPath));
      }
    }
  }

  return { copied, removed };
}

// NEW syncDirWithCleanup implementation (WITH hash comparison)
function syncDirWithCleanupNew(src, dest, opts) {
  opts = opts || {};
  const dryRun = opts.dryRun || false;
  fs.mkdirSync(dest, { recursive: true });

  // Copy phase with hash comparison
  let copied = 0;
  let skipped = 0;
  const srcFiles = listFilesRecursive(src);
  if (!dryRun) {
    for (const relPath of srcFiles) {
      const srcPath = path.join(src, relPath);
      const destPath = path.join(dest, relPath);
      fs.mkdirSync(path.dirname(destPath), { recursive: true });

      // Hash comparison: only copy if file doesn't exist or hash differs
      let shouldCopy = true;
      if (fs.existsSync(destPath)) {
        const srcHash = hashFileSync(srcPath);
        const destHash = hashFileSync(destPath);
        if (srcHash === destHash) {
          shouldCopy = false;
          skipped++;
        }
      }

      if (shouldCopy) {
        fs.copyFileSync(srcPath, destPath);
        if (relPath.endsWith('.sh')) {
          fs.chmodSync(destPath, 0o755);
        }
        copied++;
      }
    }
  } else {
    copied = srcFiles.length;
  }

  // Cleanup phase — remove files in dest that aren't in src
  const destFiles = listFilesRecursive(dest);
  const srcSet = new Set(srcFiles);
  const removed = [];
  for (const relPath of destFiles) {
    if (!srcSet.has(relPath)) {
      removed.push(relPath);
      if (!dryRun) {
        fs.unlinkSync(path.join(dest, relPath));
      }
    }
  }

  return { copied, removed, skipped };
}

// Test 1: Same content - should NOT copy (skipped)
test('same content in src and dest - should skip copy', t => {
  const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'aether-sync-'));
  const srcDir = path.join(testDir, 'src');
  const destDir = path.join(testDir, 'dest');
  fs.mkdirSync(srcDir, { recursive: true });
  fs.mkdirSync(destDir, { recursive: true });
  t.teardown(() => fs.rmSync(testDir, { recursive: true, force: true }));

  // Create same file in both
  fs.writeFileSync(path.join(srcDir, 'file.txt'), 'hello world');
  fs.writeFileSync(path.join(destDir, 'file.txt'), 'hello world');

  const result = syncDirWithCleanupNew(srcDir, destDir);

  // Should skip (not copy) because hashes match
  t.is(result.copied, 0);
  t.is(result.skipped, 1);
});

// Test 2: Different content - should copy
test('different content in src and dest - should copy', t => {
  const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'aether-sync-'));
  const srcDir = path.join(testDir, 'src');
  const destDir = path.join(testDir, 'dest');
  fs.mkdirSync(srcDir, { recursive: true });
  fs.mkdirSync(destDir, { recursive: true });
  t.teardown(() => fs.rmSync(testDir, { recursive: true, force: true }));

  fs.writeFileSync(path.join(srcDir, 'file.txt'), 'hello world');
  fs.writeFileSync(path.join(destDir, 'file.txt'), 'hello different');

  const result = syncDirWithCleanupNew(srcDir, destDir);

  t.is(result.copied, 1);
  t.is(result.skipped, 0);
});

// Test 3: File only in dest (cleanup should work)
test('file only in dest - should be cleaned up', t => {
  const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'aether-sync-'));
  const srcDir = path.join(testDir, 'src');
  const destDir = path.join(testDir, 'dest');
  fs.mkdirSync(srcDir, { recursive: true });
  fs.mkdirSync(destDir, { recursive: true });
  t.teardown(() => fs.rmSync(testDir, { recursive: true, force: true }));

  // Only in dest
  fs.writeFileSync(path.join(destDir, 'orphan.txt'), 'orphan');

  const result = syncDirWithCleanupNew(srcDir, destDir);

  t.is(result.removed.length, 1);
  t.is(result.removed[0], 'orphan.txt');
});

// Test 4: New file in src - should copy
test('new file in src (not in dest) - should copy', t => {
  const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'aether-sync-'));
  const srcDir = path.join(testDir, 'src');
  const destDir = path.join(testDir, 'dest');
  fs.mkdirSync(srcDir, { recursive: true });
  fs.mkdirSync(destDir, { recursive: true });
  t.teardown(() => fs.rmSync(testDir, { recursive: true, force: true }));

  fs.writeFileSync(path.join(srcDir, 'new.txt'), 'new content');
  // dest is empty

  const result = syncDirWithCleanupNew(srcDir, destDir);

  t.is(result.copied, 1);
  t.true(fs.existsSync(path.join(destDir, 'new.txt')));
});

// Test 5: Multiple files - mix of same and different
test('multiple files - mix of same, different, new', t => {
  const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'aether-sync-'));
  const srcDir = path.join(testDir, 'src');
  const destDir = path.join(testDir, 'dest');
  fs.mkdirSync(srcDir, { recursive: true });
  fs.mkdirSync(destDir, { recursive: true });
  t.teardown(() => fs.rmSync(testDir, { recursive: true, force: true }));

  // src files
  fs.writeFileSync(path.join(srcDir, 'same.txt'), 'same content');
  fs.writeFileSync(path.join(srcDir, 'different.txt'), 'new content');
  fs.writeFileSync(path.join(srcDir, 'new.txt'), 'brand new');

  // dest files (different.txt has old content, same.txt has same)
  fs.writeFileSync(path.join(destDir, 'same.txt'), 'same content');
  fs.writeFileSync(path.join(destDir, 'different.txt'), 'old content');

  const result = syncDirWithCleanupNew(srcDir, destDir);

  // same.txt: skipped, different.txt: copied, new.txt: copied
  t.is(result.copied, 2);
  t.is(result.skipped, 1);
});

// Test 6: Dry run mode
test('dry run mode - should report files but not copy', t => {
  const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'aether-sync-'));
  const srcDir = path.join(testDir, 'src');
  const destDir = path.join(testDir, 'dest');
  fs.mkdirSync(srcDir, { recursive: true });
  fs.mkdirSync(destDir, { recursive: true });
  t.teardown(() => fs.rmSync(testDir, { recursive: true, force: true }));

  fs.writeFileSync(path.join(srcDir, 'file.txt'), 'content');
  // dest is empty

  const result = syncDirWithCleanupNew(srcDir, destDir, { dryRun: true });

  // In dry run, should report all files as "copied" but not actually copy
  t.is(result.copied, 1);
  t.false(fs.existsSync(path.join(destDir, 'file.txt')));
});
