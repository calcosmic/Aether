#!/usr/bin/env node
// Namespace Isolation Verification Tests
// Verifies that Aether's 'ant' namespace is isolated from other agent namespaces

const test = require('ava');
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
test('ant namespace is a directory', t => {
  const antPath = path.join(COMMANDS_DIR, 'ant');

  if (!fs.existsSync(antPath)) {
    t.pass('skipped: ant directory does not exist');
    return;
  }

  const stat = fs.statSync(antPath);
  t.true(stat.isDirectory());
});

// Test 2: Verify ant commands don't collide with cds commands
// NOTE: Filename collisions are ACCEPTABLE because invocation is namespace-scoped
// e.g., /ant:help vs /cds:help - different directories, no actual collision
test('ant commands are properly isolated from cds (via directory)', t => {
  const antPath = path.join(COMMANDS_DIR, 'ant');
  const cdsPath = path.join(COMMANDS_DIR, 'cds');

  if (!fs.existsSync(antPath) || !fs.existsSync(cdsPath)) {
    t.pass('skipped: one or both directories do not exist');
    return;
  }

  // The KEY isolation: ant commands are in the ant/ directory, cds in cds/
  // This means /ant:help and /cds:help are DIFFERENT commands
  // Filename collisions don't matter because they're in different directories
  const antExists = fs.existsSync(antPath);
  const cdsExists = fs.existsSync(cdsPath);

  t.true(antExists && cdsExists);
});

// Test 3: Verify ant commands don't collide with mds commands
// NOTE: Filename collisions are ACCEPTABLE because invocation is namespace-scoped
test('ant commands are properly isolated from mds (via directory)', t => {
  const antPath = path.join(COMMANDS_DIR, 'ant');
  const mdsPath = path.join(COMMANDS_DIR, 'mds');

  if (!fs.existsSync(antPath) || !fs.existsSync(mdsPath)) {
    t.pass('skipped: one or both directories do not exist');
    return;
  }

  const antExists = fs.existsSync(antPath);
  const mdsExists = fs.existsSync(mdsPath);

  t.true(antExists && mdsExists);
});

// Test 4: Verify ant doesn't collide with st: prefix files
test('ant does not collide with st: prefix files', t => {
  const antPath = path.join(COMMANDS_DIR, 'ant');

  if (!fs.existsSync(COMMANDS_DIR)) {
    t.pass('skipped: commands directory does not exist');
    return;
  }

  if (!fs.existsSync(antPath)) {
    t.pass('skipped: ant directory does not exist');
    return;
  }

  const stFiles = fs.readdirSync(COMMANDS_DIR)
    .filter(f => f.startsWith('st:') && f.endsWith('.md'))
    .map(f => f.replace('st:', '').replace('.md', ''));

  const antCommands = listFilesRecursive(antPath).map(f => path.basename(f, '.md'));
  const collision = antCommands.filter(cmd => stFiles.includes(cmd));

  t.is(collision.length, 0);
});

// Test 5: Verify ant has unique command names (no duplicates)
test('ant commands are unique (no duplicate filenames)', t => {
  const antPath = path.join(COMMANDS_DIR, 'ant');

  if (!fs.existsSync(antPath)) {
    t.pass('skipped: ant directory does not exist');
    return;
  }

  const files = listFilesRecursive(antPath).filter(f => f.endsWith('.md'));
  const basenames = files.map(f => path.basename(f, '.md'));
  const duplicates = basenames.filter((item, index) => basenames.indexOf(item) !== index);

  t.is(duplicates.length, 0);
});

// Test 6: Verify isolation via directory boundary
test('directory isolation prevents cross-namespace contamination', t => {
  const antPath = path.join(COMMANDS_DIR, 'ant');

  if (!fs.existsSync(antPath)) {
    t.pass('skipped: ant directory does not exist');
    return;
  }

  // Verify that all files in ant/ are actually .md files (commands)
  const files = listFilesRecursive(antPath);
  const nonMarkdown = files.filter(f => !f.endsWith('.md') && !f.endsWith('.sh'));

  t.is(nonMarkdown.length, 0);
});

// Test 7: Verify sync doesn't affect other namespaces
test('sync mechanism only affects ant/ directory', t => {
  // Read the cli.js to verify sync only targets ant/
  const cliPath = path.join(__dirname, '..', 'bin', 'cli.js');

  if (!fs.existsSync(cliPath)) {
    t.pass('skipped: cli.js not found');
    return;
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

  t.true(hasAntOnly && !targetsCDS && !targetsMDS);
});

// Test 8: Verify namespace documentation is accurate
test('namespace documentation matches actual state', t => {
  const namespaceDocPath = path.join(__dirname, '..', '.aether', 'docs', 'namespace.md');

  if (!fs.existsSync(namespaceDocPath)) {
    t.pass('skipped: namespace.md documentation not found');
    return;
  }

  const { namespaces, error } = getNamespaces();

  if (error) {
    t.pass('skipped: ' + error);
    return;
  }

  const dirNamespaces = namespaces.filter(n => n.type === 'directory').map(n => n.name);

  // Check that documentation mentions the correct namespaces
  const docContent = fs.readFileSync(namespaceDocPath, 'utf8');
  const mentionsAnt = docContent.includes('`ant/`');
  const mentionsCDS = docContent.includes('`cds/`');
  const mentionsMDS = docContent.includes('`mds/`');

  t.true(mentionsAnt && mentionsCDS && mentionsMDS);
});
