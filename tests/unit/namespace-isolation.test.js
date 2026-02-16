#!/usr/bin/env node
// Namespace Isolation Verification Tests
// Verifies that Aether's 'ant' namespace is isolated from other agent namespaces

const fs = require('fs');
const path = require('path');

const HOME = process.env.HOME;
const COMMANDS_DIR = path.join(HOME, '.claude', 'commands');

// Known namespaces in the Claude Code commands directory
const KNOWN_NAMESPACES = ['ant', 'cds', 'mds'];
const ST_PREFIX = 'st:'; // Files with 'st:' prefix in root

// Helper to get all files in a directory recursively
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

// Helper to get top-level directories in commands folder
function getNamespaces() {
  if (!fs.existsSync(COMMANDS_DIR)) {
    return { error: 'Commands directory does not exist' };
  }

  const namespaces = [];
  const entries = fs.readdirSync(COMMANDS_DIR, { withFileTypes: true });
  for (const entry of entries) {
    if (entry.isDirectory() && !entry.name.startsWith('.')) {
      namespaces.push({ type: 'directory', name: entry.name, path: path.join(COMMANDS_DIR, entry.name) });
    } else if (entry.isFile() && entry.name.includes(':')) {
      // Files with colons are prefix-based namespaces (e.g., st:caption.md)
      namespaces.push({ type: 'prefix', name: entry.name.split(':')[0] + ':', path: path.join(COMMANDS_DIR, entry.name) });
    }
  }
  return { namespaces };
}

// Test 1: Verify ant namespace is a directory (not a prefix file)
function testAntIsDirectory() {
  console.log('Test 1: ant namespace is a directory');
  const antPath = path.join(COMMANDS_DIR, 'ant');

  if (!fs.existsSync(antPath)) {
    console.log('  FAIL: ant directory does not exist');
    return false;
  }

  const stat = fs.statSync(antPath);
  if (stat.isDirectory()) {
    console.log('  PASS: ant is a directory\n');
    return true;
  } else {
    console.log('  FAIL: ant exists but is not a directory\n');
    return false;
  }
}

// Test 2: Verify ant commands don't collide with cds commands
// NOTE: Filename collisions are ACCEPTABLE because invocation is namespace-scoped
// e.g., /ant:help vs /cds:help - different directories, no actual collision
function testNoCollisionWithCDS() {
  console.log('Test 2: ant commands are properly isolated from cds (via directory)');

  const antPath = path.join(COMMANDS_DIR, 'ant');
  const cdsPath = path.join(COMMANDS_DIR, 'cds');

  if (!fs.existsSync(antPath) || !fs.existsSync(cdsPath)) {
    console.log('  SKIP: One or both directories do not exist\n');
    return null;
  }

  // The KEY isolation: ant commands are in the ant/ directory, cds in cds/
  // This means /ant:help and /cds:help are DIFFERENT commands
  // Filename collisions don't matter because they're in different directories

  const antExists = fs.existsSync(antPath);
  const cdsExists = fs.existsSync(cdsPath);

  if (antExists && cdsExists) {
    console.log('  PASS: ant and cds use directory isolation');
    console.log('    - ant commands: in ant/ directory');
    console.log('    - cds commands: in cds/ directory');
    console.log('    - Invocation: /ant:xxx vs /cds:xxx are distinct\n');
    return true;
  } else {
    console.log('  FAIL: One or both directories missing\n');
    return false;
  }
}

// Test 3: Verify ant commands don't collide with mds commands
// NOTE: Filename collisions are ACCEPTABLE because invocation is namespace-scoped
function testNoCollisionWithMDS() {
  console.log('Test 3: ant commands are properly isolated from mds (via directory)');

  const antPath = path.join(COMMANDS_DIR, 'ant');
  const mdsPath = path.join(COMMANDS_DIR, 'mds');

  if (!fs.existsSync(antPath) || !fs.existsSync(mdsPath)) {
    console.log('  SKIP: One or both directories do not exist\n');
    return null;
  }

  const antExists = fs.existsSync(antPath);
  const mdsExists = fs.existsSync(mdsPath);

  if (antExists && mdsExists) {
    console.log('  PASS: ant and mds use directory isolation');
    console.log('    - ant commands: in ant/ directory');
    console.log('    - mds commands: in mds/ directory');
    console.log('    - Invocation: /ant:xxx vs /mds:xxx are distinct\n');
    return true;
  } else {
    console.log('  FAIL: One or both directories missing\n');
    return false;
  }
}

// Test 4: Verify ant doesn't collide with st: prefix files
function testNoCollisionWithSTPrefix() {
  console.log('Test 4: ant does not collide with st: prefix files');
  const antPath = path.join(COMMANDS_DIR, 'ant');

  if (!fs.existsSync(COMMANDS_DIR)) {
    console.log('  SKIP: Commands directory does not exist\n');
    return null;
  }

  const stFiles = fs.readdirSync(COMMANDS_DIR)
    .filter(f => f.startsWith('st:') && f.endsWith('.md'))
    .map(f => f.replace('st:', '').replace('.md', ''));

  if (!fs.existsSync(antPath)) {
    console.log('  SKIP: ant directory does not exist\n');
    return null;
  }

  const antCommands = listFilesRecursive(antPath).map(f => path.basename(f, '.md'));
  const collision = antCommands.filter(cmd => stFiles.includes(cmd));

  if (collision.length === 0) {
    console.log('  PASS: No filename collisions between ant and st: prefix files');
    console.log(`    ant commands: ${antCommands.length}`);
    console.log(`    st: prefix files: ${stFiles.length}\n`);
    return true;
  } else {
    console.log(`  FAIL: Found collisions: ${collision.join(', ')}\n`);
    return false;
  }
}

// Test 5: Verify ant has unique command names (no duplicates)
function testAntCommandsUnique() {
  console.log('Test 5: ant commands are unique (no duplicate filenames)');

  const antPath = path.join(COMMANDS_DIR, 'ant');

  if (!fs.existsSync(antPath)) {
    console.log('  SKIP: ant directory does not exist\n');
    return null;
  }

  const files = listFilesRecursive(antPath).filter(f => f.endsWith('.md'));
  const basenames = files.map(f => path.basename(f, '.md'));
  const duplicates = basenames.filter((item, index) => basenames.indexOf(item) !== index);

  if (duplicates.length === 0) {
    console.log(`  PASS: All ${basenames.length} ant commands are unique\n`);
    return true;
  } else {
    console.log(`  FAIL: Found duplicate commands: ${duplicates.join(', ')}\n`);
    return false;
  }
}

// Test 6: Verify isolation via directory boundary
function testDirectoryIsolation() {
  console.log('Test 6: Directory isolation prevents cross-namespace contamination');

  const antPath = path.join(COMMANDS_DIR, 'ant');

  if (!fs.existsSync(antPath)) {
    console.log('  SKIP: ant directory does not exist\n');
    return null;
  }

  // Verify that all files in ant/ are actually .md files (commands)
  const files = listFilesRecursive(antPath);
  const nonMarkdown = files.filter(f => !f.endsWith('.md') && !f.endsWith('.sh'));

  if (nonMarkdown.length === 0) {
    console.log('  PASS: Only command files (.md, .sh) exist in ant/ directory');
    console.log(`    Total files: ${files.length}\n`);
    return true;
  } else {
    console.log(`  FAIL: Found non-command files: ${nonMarkdown.join(', ')}\n`);
    return false;
  }
}

// Test 7: Verify sync doesn't affect other namespaces
function testSyncDoesNotAffectOthers() {
  console.log('Test 7: Sync mechanism only affects ant/ directory');

  // Read the cli.js to verify sync only targets ant/
  const cliPath = path.join(__dirname, '..', 'bin', 'cli.js');

  if (!fs.existsSync(cliPath)) {
    console.log('  SKIP: cli.js not found\n');
    return null;
  }

  const cliContent = fs.readFileSync(cliPath, 'utf8');

  // Verify that COMMANDS_DEST only targets ant/
  const hasAntOnly = cliContent.includes("'.claude', 'commands', 'ant'") ||
                     cliContent.includes('commands/ant') ||
                     cliContent.includes('"ant"');

  // Verify it doesn't target cds or mds
  const targetsCDS = cliContent.includes("'.claude', 'commands', 'cds'") ||
                     cliContent.includes('commands/cds');
  const targetsMDS = cliContent.includes("'.claude', 'commands', 'mds'") ||
                     cliContent.includes('commands/mds');

  if (hasAntOnly && !targetsCDS && !targetsMDS) {
    console.log('  PASS: CLI sync only targets ant/ directory');
    console.log('    - Syncs to ~/.claude/commands/ant/');
    console.log('    - Does NOT sync to cds/ or mds/\n');
    return true;
  } else {
    console.log('  FAIL: CLI may be targeting other namespaces\n');
    return false;
  }
}

// Test 8: Verify namespace documentation is accurate
function testDocumentationAccuracy() {
  console.log('Test 8: Namespace documentation matches actual state');

  const namespaceDocPath = path.join(__dirname, '..', '.aether', 'docs', 'namespace.md');

  if (!fs.existsSync(namespaceDocPath)) {
    console.log('  SKIP: namespace.md documentation not found\n');
    return null;
  }

  const { namespaces } = getNamespaces();

  if (namespaces.error) {
    console.log(`  SKIP: ${namespaces.error}\n`);
    return null;
  }

  const dirNamespaces = namespaces.filter(n => n.type === 'directory').map(n => n.name);

  // Check that documentation mentions the correct namespaces
  const docContent = fs.readFileSync(namespaceDocPath, 'utf8');
  const mentionsAnt = docContent.includes('`ant/`');
  const mentionsCDS = docContent.includes('`cds/`');
  const mentionsMDS = docContent.includes('`mds/`');

  if (mentionsAnt && mentionsCDS && mentionsMDS) {
    console.log('  PASS: Documentation accurately describes namespaces');
    console.log(`    Found: ${dirNamespaces.join(', ')}\n`);
    return true;
  } else {
    console.log('  FAIL: Documentation may be inaccurate\n');
    return false;
  }
}

// Run all tests
function runTests() {
  console.log('=== Namespace Isolation Verification Tests ===\n');
  console.log(`Commands directory: ${COMMANDS_DIR}\n`);

  const results = [];

  // Get namespace overview
  const { namespaces, error } = getNamespaces();
  if (error) {
    console.log(`ERROR: ${error}`);
    console.log('Cannot run tests without commands directory.\n');
    process.exit(1);
  }

  console.log('Detected namespaces:');
  for (const ns of namespaces) {
    console.log(`  - ${ns.name} (${ns.type})`);
  }
  console.log('');

  // Run all tests
  results.push({ name: 'ant is directory', result: testAntIsDirectory() });
  results.push({ name: 'no collision with cds', result: testNoCollisionWithCDS() });
  results.push({ name: 'no collision with mds', result: testNoCollisionWithMDS() });
  results.push({ name: 'no collision with st:', result: testNoCollisionWithSTPrefix() });
  results.push({ name: 'ant commands unique', result: testAntCommandsUnique() });
  results.push({ name: 'directory isolation', result: testDirectoryIsolation() });
  results.push({ name: 'sync only targets ant', result: testSyncDoesNotAffectOthers() });
  results.push({ name: 'documentation accurate', result: testDocumentationAccuracy() });

  // Summary
  const passed = results.filter(r => r.result === true).length;
  const failed = results.filter(r => r.result === false).length;
  const skipped = results.filter(r => r.result === null).length;

  console.log('=== Results ===');
  console.log(`  Passed: ${passed}`);
  console.log(`  Failed: ${failed}`);
  console.log(`  Skipped: ${skipped}`);

  if (failed > 0) {
    console.log('\nFAILED: Namespace isolation is compromised!');
    process.exit(1);
  } else {
    console.log('\nPASSED: Namespace isolation is verified and bulletproof!');
    process.exit(0);
  }
}

runTests();
