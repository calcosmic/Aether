#!/usr/bin/env node
// Test file for syncDirWithCleanup hash comparison feature

const fs = require('fs');
const path = require('path');
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

// Test utilities
function setupTestDirs() {
  const testDir = path.join(__dirname, 'test-sync-temp');
  const srcDir = path.join(testDir, 'src');
  const destDir = path.join(testDir, 'dest');

  // Clean up any existing test dirs
  if (fs.existsSync(testDir)) {
    fs.rmSync(testDir, { recursive: true });
  }

  fs.mkdirSync(srcDir, { recursive: true });
  fs.mkdirSync(destDir, { recursive: true });

  return { testDir, srcDir, destDir };
}

function cleanupTestDirs(testDir) {
  if (fs.existsSync(testDir)) {
    fs.rmSync(testDir, { recursive: true });
  }
}

// Test cases
function runTests() {
  let passed = 0;
  let failed = 0;

  console.log('=== Testing syncDirWithCleanup hash comparison ===\n');

  // Test 1: Same content - should NOT copy (skipped)
  console.log('Test 1: Same content in src and dest - should skip copy');
  {
    const { testDir, srcDir, destDir } = setupTestDirs();
    try {
      // Create same file in both
      fs.writeFileSync(path.join(srcDir, 'file.txt'), 'hello world');
      fs.writeFileSync(path.join(destDir, 'file.txt'), 'hello world');

      const result = syncDirWithCleanupNew(srcDir, destDir);

      // Should skip (not copy) because hashes match
      if (result.copied === 0 && result.skipped === 1) {
        console.log('  PASS: No copy when hashes match\n');
        passed++;
      } else {
        console.log(`  FAIL: Expected copied=0, skipped=1, got copied=${result.copied}, skipped=${result.skipped}\n`);
        failed++;
      }
    } finally {
      cleanupTestDirs(testDir);
    }
  }

  // Test 2: Different content - should copy
  console.log('Test 2: Different content in src and dest - should copy');
  {
    const { testDir, srcDir, destDir } = setupTestDirs();
    try {
      fs.writeFileSync(path.join(srcDir, 'file.txt'), 'hello world');
      fs.writeFileSync(path.join(destDir, 'file.txt'), 'hello different');

      const result = syncDirWithCleanupNew(srcDir, destDir);

      if (result.copied === 1 && result.skipped === 0) {
        console.log('  PASS: Copied when content differs\n');
        passed++;
      } else {
        console.log(`  FAIL: Expected copied=1, skipped=0, got copied=${result.copied}, skipped=${result.skipped}\n`);
        failed++;
      }
    } finally {
      cleanupTestDirs(testDir);
    }
  }

  // Test 3: File only in dest (cleanup should work)
  console.log('Test 3: File only in dest - should be cleaned up');
  {
    const { testDir, srcDir, destDir } = setupTestDirs();
    try {
      // Only in dest
      fs.writeFileSync(path.join(destDir, 'orphan.txt'), 'orphan');

      const result = syncDirWithCleanupNew(srcDir, destDir);

      if (result.removed.length === 1 && result.removed[0] === 'orphan.txt') {
        console.log('  PASS: Orphan file removed\n');
        passed++;
      } else {
        console.log(`  FAIL: Expected removed=['orphan.txt'], got removed=[${result.removed}]\n`);
        failed++;
      }
    } finally {
      cleanupTestDirs(testDir);
    }
  }

  // Test 4: New file in src - should copy
  console.log('Test 4: New file in src (not in dest) - should copy');
  {
    const { testDir, srcDir, destDir } = setupTestDirs();
    try {
      fs.writeFileSync(path.join(srcDir, 'new.txt'), 'new content');
      // dest is empty

      const result = syncDirWithCleanupNew(srcDir, destDir);

      if (result.copied === 1 && fs.existsSync(path.join(destDir, 'new.txt'))) {
        console.log('  PASS: New file copied\n');
        passed++;
      } else {
        console.log(`  FAIL: Expected copied=1, got copied=${result.copied}\n`);
        failed++;
      }
    } finally {
      cleanupTestDirs(testDir);
    }
  }

  // Test 5: Multiple files - mix of same and different
  console.log('Test 5: Multiple files - mix of same, different, new');
  {
    const { testDir, srcDir, destDir } = setupTestDirs();
    try {
      // src files
      fs.writeFileSync(path.join(srcDir, 'same.txt'), 'same content');
      fs.writeFileSync(path.join(srcDir, 'different.txt'), 'new content');
      fs.writeFileSync(path.join(srcDir, 'new.txt'), 'brand new');

      // dest files (different.txt has old content, same.txt has same)
      fs.writeFileSync(path.join(destDir, 'same.txt'), 'same content');
      fs.writeFileSync(path.join(destDir, 'different.txt'), 'old content');

      const result = syncDirWithCleanupNew(srcDir, destDir);

      // same.txt: skipped, different.txt: copied, new.txt: copied
      if (result.copied === 2 && result.skipped === 1) {
        console.log(`  PASS: Copied=2 (different+new), Skipped=1 (same)\n`);
        passed++;
      } else {
        console.log(`  FAIL: Expected copied=2, skipped=1, got copied=${result.copied}, skipped=${result.skipped}\n`);
        failed++;
      }
    } finally {
      cleanupTestDirs(testDir);
    }
  }

  // Test 6: Dry run mode
  console.log('Test 6: Dry run mode - should report files but not copy');
  {
    const { testDir, srcDir, destDir } = setupTestDirs();
    try {
      fs.writeFileSync(path.join(srcDir, 'file.txt'), 'content');
      // dest is empty

      const result = syncDirWithCleanupNew(srcDir, destDir, { dryRun: true });

      // In dry run, should report all files as "copied" but not actually copy
      if (result.copied === 1 && !fs.existsSync(path.join(destDir, 'file.txt'))) {
        console.log('  PASS: Dry run reports but does not copy\n');
        passed++;
      } else {
        console.log(`  FAIL: Expected copied=1, got copied=${result.copied}\n`);
        failed++;
      }
    } finally {
      cleanupTestDirs(testDir);
    }
  }

  console.log(`\n=== Results: ${passed} passed, ${failed} failed ===`);
  process.exit(failed > 0 ? 1 : 0);
}

runTests();
