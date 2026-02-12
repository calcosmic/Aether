#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

const VERSION = require('../package.json').version;
const PACKAGE_DIR = path.resolve(__dirname, '..');

// Claude Code paths (global)
const COMMANDS_SRC = path.join(PACKAGE_DIR, 'commands', 'ant');
const COMMANDS_DEST = path.join(process.env.HOME, '.claude', 'commands', 'ant');

const command = process.argv[2] || 'help';
const flags = process.argv.slice(3);
const quiet = flags.includes('--quiet');

function log(msg) {
  if (!quiet) console.log(msg);
}

function copyDirSync(src, dest) {
  fs.mkdirSync(dest, { recursive: true });
  const entries = fs.readdirSync(src, { withFileTypes: true });
  let count = 0;
  for (const entry of entries) {
    const srcPath = path.join(src, entry.name);
    const destPath = path.join(dest, entry.name);
    if (entry.isDirectory()) {
      count += copyDirSync(srcPath, destPath);
    } else {
      fs.copyFileSync(srcPath, destPath);
      // Preserve executable bit for shell scripts
      if (entry.name.endsWith('.sh')) {
        fs.chmodSync(destPath, 0o755);
      }
      count++;
    }
  }
  return count;
}

function removeDirSync(dir) {
  if (!fs.existsSync(dir)) return 0;
  let count = 0;
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      count += removeDirSync(fullPath);
    } else {
      fs.unlinkSync(fullPath);
      count++;
    }
  }
  fs.rmdirSync(dir);
  return count;
}

switch (command) {
  case 'install': {
    log(`aether-colony v${VERSION} — installing...`);

    // Copy commands to ~/.claude/commands/ant/
    if (!fs.existsSync(COMMANDS_SRC)) {
      // Running from source repo — commands are in .claude/commands/ant/
      const repoCommands = path.join(PACKAGE_DIR, '.claude', 'commands', 'ant');
      if (fs.existsSync(repoCommands)) {
        const n = copyDirSync(repoCommands, COMMANDS_DEST);
        log(`  Commands: ${n} files -> ${COMMANDS_DEST}`);
      } else {
        console.error('  Commands source not found. Skipping.');
      }
    } else {
      const n = copyDirSync(COMMANDS_SRC, COMMANDS_DEST);
      log(`  Commands: ${n} files -> ${COMMANDS_DEST}`);
    }

    log('');
    log('Install complete.');
    log('  Claude Code: run /ant to get started');
    break;
  }

  case 'version': {
    console.log(`aether-colony v${VERSION}`);
    break;
  }

  case 'uninstall': {
    log(`aether-colony v${VERSION} — uninstalling...`);

    // Remove Claude Code commands
    if (fs.existsSync(COMMANDS_DEST)) {
      const n = removeDirSync(COMMANDS_DEST);
      log(`  Removed: ${n} command files from ${COMMANDS_DEST}`);
    } else {
      log('  Claude Code commands already removed.');
    }

    log('');
    log('Uninstall complete. Per-project .aether/data/ directories are untouched.');
    break;
  }

  case 'help':
  default: {
    console.log(`
aether-colony v${VERSION}

Usage: aether <command>

Commands:
  install      Install Claude Code slash-commands to ~/.claude/commands/ant/
  version      Show installed version
  uninstall    Remove Claude Code slash-commands (preserves project state)
  help         Show this help message

Install location:
  Claude Code: ~/.claude/commands/ant/

After install, run /ant in Claude Code to get started.
Project state lives in each repo's .aether/data/ directory.
`.trim());
    break;
  }
}
