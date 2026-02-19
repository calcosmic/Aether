#!/usr/bin/env node
// Test file for user modification detection feature
// Tests Task 4.2: Handle user modification detection

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

// NEW syncDirWithCleanup implementation (WITH user modification detection)
function syncDirWithCleanup(src, dest, opts) {
  opts = opts || {};
  const dryRun = opts.dryRun || false;
  const backupDir = opts.backupDir || null;
  const manifest = opts.manifest || null;
  fs.mkdirSync(dest, { recursive: true });

  // Phase 1: Detect user modifications (files that differ from both source AND manifest)
  const userModifications = [];
  if (manifest && manifest.files) {
    const srcFiles = listFilesRecursive(src);
    for (const relPath of srcFiles) {
      const srcPath = path.join(src, relPath);
      const destPath = path.join(dest, relPath);
      if (fs.existsSync(destPath)) {
        const srcHash = hashFileSync(srcPath);
        const destHash = hashFileSync(destPath);
        const manifestHash = manifest.files[relPath];
        // User modified: dest differs from source AND dest differs from manifest
        if (srcHash !== destHash && manifestHash && destHash !== manifestHash) {
          userModifications.push(relPath);
        }
      }
    }
  }

  // Phase 2: Backup user-modified files if backupDir specified
  const backedUp = [];
  if (backupDir && userModifications.length > 0 && !dryRun) {
    fs.mkdirSync(backupDir, { recursive: true });
    for (const relPath of userModifications) {
      const destPath = path.join(dest, relPath);
      const backupPath = path.join(backupDir, relPath);
      if (fs.existsSync(destPath)) {
        fs.mkdirSync(path.dirname(backupPath), { recursive: true });
        fs.copyFileSync(destPath, backupPath);
        backedUp.push(relPath);
      }
    }
  }

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

  // Only include userModifications and backedUp when manifest is provided (backward compatible)
  if (manifest && manifest.files) {
    return { copied, removed, skipped, userModifications, backedUp };
  } else {
    return { copied, removed, skipped };
  }
}

// Test 1: Detect user modification - dest differs from both source and manifest
test('detect user modification (dest differs from source AND manifest)', t => {
  const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'aether-umod-'));
  const srcDir = path.join(testDir, 'src');
  const destDir = path.join(testDir, 'dest');
  fs.mkdirSync(srcDir, { recursive: true });
  fs.mkdirSync(destDir, { recursive: true });
  t.teardown(() => fs.rmSync(testDir, { recursive: true, force: true }));

  // Source file with content "v1"
  fs.writeFileSync(path.join(srcDir, 'config.txt'), 'v1');

  // Dest file with user modification "user-changes"
  fs.writeFileSync(path.join(destDir, 'config.txt'), 'user-changes');

  // Manifest expects v1
  const manifest = {
    generated_at: '2026-01-01T00:00:00Z',
    files: {
      'config.txt': hashFileSync(path.join(srcDir, 'config.txt'))
    }
  };

  const result = syncDirWithCleanup(srcDir, destDir, { manifest });

  // Should detect user modification
  t.true(result.userModifications !== undefined && result.userModifications.length === 1);
  t.is(result.userModifications[0], 'config.txt');
});

// Test 2: No false positive - source changed, user didn't modify
test('no false positive when source changed (user kept original)', t => {
  const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'aether-umod-'));
  const srcDir = path.join(testDir, 'src');
  const destDir = path.join(testDir, 'dest');
  fs.mkdirSync(srcDir, { recursive: true });
  fs.mkdirSync(destDir, { recursive: true });
  t.teardown(() => fs.rmSync(testDir, { recursive: true, force: true }));

  // Source has "v1" originally
  fs.writeFileSync(path.join(srcDir, 'config.txt'), 'v1');

  // Compute manifest hash for v1 BEFORE updating source
  const v1Hash = hashFileSync(path.join(srcDir, 'config.txt'));

  // Now update source to v2
  fs.writeFileSync(path.join(srcDir, 'config.txt'), 'v2');

  // Dest still has "v1" (original from manifest)
  fs.writeFileSync(path.join(destDir, 'config.txt'), 'v1');

  // Manifest expects v1 (the original hash)
  const manifest = {
    generated_at: '2026-01-01T00:00:00Z',
    files: {
      'config.txt': v1Hash
    }
  };

  const result = syncDirWithCleanup(srcDir, destDir, { manifest });

  // Should NOT detect user modification (dest matches manifest, not source)
  // This is a source update, not a user modification
  t.true(!result.userModifications || result.userModifications.length === 0);
});

// Test 3: Backup user-modified files
test('backup user-modified files when --backup specified', t => {
  const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'aether-umod-'));
  const srcDir = path.join(testDir, 'src');
  const destDir = path.join(testDir, 'dest');
  const backupDir = path.join(testDir, 'backup');
  fs.mkdirSync(srcDir, { recursive: true });
  fs.mkdirSync(destDir, { recursive: true });
  fs.mkdirSync(backupDir, { recursive: true });
  t.teardown(() => fs.rmSync(testDir, { recursive: true, force: true }));

  // Source file with content "v1"
  fs.writeFileSync(path.join(srcDir, 'config.txt'), 'v1');

  // Dest file with user modification "user-changes"
  fs.writeFileSync(path.join(destDir, 'config.txt'), 'user-changes');

  // Manifest expects v1
  const manifest = {
    generated_at: '2026-01-01T00:00:00Z',
    files: {
      'config.txt': hashFileSync(path.join(srcDir, 'config.txt'))
    }
  };

  const result = syncDirWithCleanup(srcDir, destDir, { manifest, backupDir });

  // Should backup the user-modified file
  const backedUpFile = path.join(backupDir, 'config.txt');
  t.true(result.backedUp !== undefined && result.backedUp.length === 1);
  t.true(fs.existsSync(backedUpFile));
  const backedContent = fs.readFileSync(backedUpFile, 'utf8');
  t.is(backedContent, 'user-changes');
});

// Test 4: No backup when not requested
test('no backup when --backup not specified', t => {
  const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'aether-umod-'));
  const srcDir = path.join(testDir, 'src');
  const destDir = path.join(testDir, 'dest');
  fs.mkdirSync(srcDir, { recursive: true });
  fs.mkdirSync(destDir, { recursive: true });
  t.teardown(() => fs.rmSync(testDir, { recursive: true, force: true }));

  // Source file with content "v1"
  fs.writeFileSync(path.join(srcDir, 'config.txt'), 'v1');

  // Dest file with user modification "user-changes"
  fs.writeFileSync(path.join(destDir, 'config.txt'), 'user-changes');

  // Manifest expects v1
  const manifest = {
    generated_at: '2026-01-01T00:00:00Z',
    files: {
      'config.txt': hashFileSync(path.join(srcDir, 'config.txt'))
    }
  };

  const result = syncDirWithCleanup(srcDir, destDir, { manifest });

  // Should NOT backup (no backupDir)
  t.true(!result.backedUp || result.backedUp.length === 0);
});

// Test 5: Multiple user-modified files
test('detect multiple user-modified files', t => {
  const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'aether-umod-'));
  const srcDir = path.join(testDir, 'src');
  const destDir = path.join(testDir, 'dest');
  fs.mkdirSync(srcDir, { recursive: true });
  fs.mkdirSync(destDir, { recursive: true });
  t.teardown(() => fs.rmSync(testDir, { recursive: true, force: true }));

  // Source files
  fs.writeFileSync(path.join(srcDir, 'file1.txt'), 'v1');
  fs.writeFileSync(path.join(srcDir, 'file2.txt'), 'v2');
  fs.writeFileSync(path.join(srcDir, 'file3.txt'), 'v3');

  // User-modified dest files
  fs.writeFileSync(path.join(destDir, 'file1.txt'), 'user1');
  fs.writeFileSync(path.join(destDir, 'file2.txt'), 'v2'); // same as source - not modified
  fs.writeFileSync(path.join(destDir, 'file3.txt'), 'user3');

  // Manifest with original hashes
  const manifest = {
    generated_at: '2026-01-01T00:00:00Z',
    files: {
      'file1.txt': hashFileSync(path.join(srcDir, 'file1.txt')),
      'file2.txt': hashFileSync(path.join(srcDir, 'file2.txt')),
      'file3.txt': hashFileSync(path.join(srcDir, 'file3.txt'))
    }
  };

  const result = syncDirWithCleanup(srcDir, destDir, { manifest });

  // Should detect 2 user modifications (file1 and file3)
  t.true(result.userModifications !== undefined && result.userModifications.length === 2);
  t.true(result.userModifications.includes('file1.txt'));
  t.true(result.userModifications.includes('file3.txt'));
});

// Test 6: Dry-run mode - no actual backup
test('dry-run mode - detect but do not backup', t => {
  const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'aether-umod-'));
  const srcDir = path.join(testDir, 'src');
  const destDir = path.join(testDir, 'dest');
  const backupDir = path.join(testDir, 'backup');
  fs.mkdirSync(srcDir, { recursive: true });
  fs.mkdirSync(destDir, { recursive: true });
  fs.mkdirSync(backupDir, { recursive: true });
  t.teardown(() => fs.rmSync(testDir, { recursive: true, force: true }));

  // Source file with content "v1"
  fs.writeFileSync(path.join(srcDir, 'config.txt'), 'v1');

  // Dest file with user modification "user-changes"
  fs.writeFileSync(path.join(destDir, 'config.txt'), 'user-changes');

  // Manifest expects v1
  const manifest = {
    generated_at: '2026-01-01T00:00:00Z',
    files: {
      'config.txt': hashFileSync(path.join(srcDir, 'config.txt'))
    }
  };

  const result = syncDirWithCleanup(srcDir, destDir, { manifest, backupDir, dryRun: true });

  // Should detect user modification but NOT backup in dry-run
  const backedUpFile = path.join(backupDir, 'config.txt');
  t.true(result.userModifications !== undefined && result.userModifications.length === 1);
  t.true(!result.backedUp || result.backedUp.length === 0);
  t.false(fs.existsSync(backedUpFile));
});

// Test 7: Without manifest - no detection (backward compatible)
test('without manifest - backward compatible behavior', t => {
  const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'aether-umod-'));
  const srcDir = path.join(testDir, 'src');
  const destDir = path.join(testDir, 'dest');
  fs.mkdirSync(srcDir, { recursive: true });
  fs.mkdirSync(destDir, { recursive: true });
  t.teardown(() => fs.rmSync(testDir, { recursive: true, force: true }));

  // Source file with content "v1"
  fs.writeFileSync(path.join(srcDir, 'config.txt'), 'v1');

  // Dest file with different content
  fs.writeFileSync(path.join(destDir, 'config.txt'), 'different');

  // No manifest - result should not have userModifications key at all
  const result = syncDirWithCleanup(srcDir, destDir);

  // Should NOT have userModifications key (backward compatible - undefined check)
  t.is(result.userModifications, undefined);
});
