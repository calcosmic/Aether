#!/usr/bin/env node

/**
 * npx-install.js — Professional installer for Aether Colony
 *
 * Usage: npx aether-colony install
 *
 * Creates the global hub at ~/.aether/ with all system files,
 * slash commands, and agent definitions.
 */

const fs = require('fs');
const path = require('path');
const os = require('os');

const BANNER = `
      █████╗ ███████╗████████╗██╗  ██╗███████╗██████╗
     ██╔══██╗██╔════╝╚══██╔══╝██║  ██║██╔════╝██╔══██╗
     ███████║█████╗     ██║   ███████║█████╗  ██████╔╝
     ██╔══██║██╔══╝     ██║   ██╔══██║██╔══╝  ██╔══██╗
     ██║  ██║███████╗   ██║   ██║  ██║███████╗██║  ██║
     ╚═╝  ╚═╝╚══════╝   ╚═╝   ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝
`;

const AETHER_VERSION = require('../package.json').version;
const HOME_DIR = os.homedir();
const HUB_DIR = path.join(HOME_DIR, '.aether');
const CLAUDE_COMMANDS_DIR = path.join(HOME_DIR, '.claude', 'commands', 'ant');
const CLAUDE_AGENTS_DIR = path.join(HOME_DIR, '.claude', 'agents', 'ant');

// Get the package root (where this script is located)
const PACKAGE_ROOT = path.resolve(__dirname, '..');
const AETHER_SRC = path.join(PACKAGE_ROOT, '.aether');
const CLAUDE_COMMANDS_SRC = path.join(PACKAGE_ROOT, '.claude', 'commands', 'ant');
const OPENCODE_AGENTS_SRC = path.join(PACKAGE_ROOT, '.opencode', 'agents');
const OPENCODE_COMMANDS_SRC = path.join(PACKAGE_ROOT, '.opencode', 'commands', 'ant');

function log(message, type = 'info') {
  const icons = {
    info: 'ℹ',
    success: '✓',
    warning: '⚠',
    error: '✗',
    ant: '🐜'
  };
  console.log(`${icons[type] || '•'} ${message}`);
}

function ensureDir(dir) {
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
    return true;
  }
  return false;
}

function copyDir(src, dest, options = {}) {
  const { exclude = [] } = options;
  ensureDir(dest);

  const entries = fs.readdirSync(src, { withFileTypes: true });
  let copied = 0;

  for (const entry of entries) {
    const srcPath = path.join(src, entry.name);
    const destPath = path.join(dest, entry.name);

    // Skip excluded directories
    if (exclude.includes(entry.name)) {
      continue;
    }

    if (entry.isDirectory()) {
      copied += copyDir(srcPath, destPath, options);
    } else {
      fs.copyFileSync(srcPath, destPath);
      copied++;
    }
  }

  return copied;
}

function copyFile(src, dest) {
  const destDir = path.dirname(dest);
  ensureDir(destDir);
  fs.copyFileSync(src, dest);
}

function install() {
  console.log(BANNER);
  console.log('\n');

  let filesCopied = 0;

  // Step 1: Create hub directory structure
  log('Creating hub directory structure...', 'ant');
  const hubDirs = [
    path.join(HUB_DIR, 'system'),
    path.join(HUB_DIR, 'system', 'docs'),
    path.join(HUB_DIR, 'system', 'utils'),
    path.join(HUB_DIR, 'system', 'templates'),
    path.join(HUB_DIR, 'system', 'schemas'),
    path.join(HUB_DIR, 'system', 'exchange'),
    path.join(HUB_DIR, 'system', 'rules'),
    path.join(HUB_DIR, 'data'),
    path.join(HUB_DIR, 'chambers')
  ];

  for (const dir of hubDirs) {
    if (ensureDir(dir)) {
      log(`  Created ${path.relative(HOME_DIR, dir)}`, 'info');
    }
  }

  // Step 2: Copy system files from .aether/
  log('Copying system files to hub...', 'ant');
  if (fs.existsSync(AETHER_SRC)) {
    // Private directories to exclude
    const excludeDirs = ['data', 'dreams', 'oracle', 'checkpoints', 'locks', 'temp', 'archive', 'chambers'];
    filesCopied += copyDir(AETHER_SRC, path.join(HUB_DIR, 'system'), { exclude: excludeDirs });
    log(`  Copied ${filesCopied} files from .aether/`, 'success');
  } else {
    log('  Warning: .aether/ source not found', 'warning');
  }

  // Step 3: Copy Claude Code commands
  log('Installing Claude Code commands...', 'ant');
  if (fs.existsSync(CLAUDE_COMMANDS_SRC)) {
    const cmdCount = copyDir(CLAUDE_COMMANDS_SRC, CLAUDE_COMMANDS_DIR);
    log(`  Installed ${cmdCount} slash commands to ~/.claude/commands/ant/`, 'success');
    filesCopied += cmdCount;
  }

  // Step 4: Copy Claude Code agents (from OpenCode agents)
  log('Installing Claude Code agents...', 'ant');
  if (fs.existsSync(OPENCODE_AGENTS_SRC)) {
    const agentCount = copyDir(OPENCODE_AGENTS_SRC, CLAUDE_AGENTS_DIR);
    log(`  Installed ${agentCount} agents to ~/.claude/agents/ant/`, 'success');
    filesCopied += agentCount;
  }

  // Step 5: Write version file
  const versionData = {
    version: AETHER_VERSION,
    installed_at: new Date().toISOString(),
    package_root: PACKAGE_ROOT
  };
  fs.writeFileSync(
    path.join(HUB_DIR, 'version.json'),
    JSON.stringify(versionData, null, 2)
  );
  log('  Version info written', 'success');

  // Step 6: Create global QUEEN.md if missing
  const globalQueen = path.join(HUB_DIR, 'QUEEN.md');
  if (!fs.existsSync(globalQueen)) {
    const queenTemplate = path.join(HUB_DIR, 'system', 'templates', 'QUEEN.md.template');
    if (fs.existsSync(queenTemplate)) {
      let content = fs.readFileSync(queenTemplate, 'utf8');
      content = content.replace(/{TIMESTAMP}/g, new Date().toISOString());
      fs.writeFileSync(globalQueen, content);
      log('  Created global QUEEN.md', 'success');
    }
  }

  // Summary
  console.log('\n  ─────────────────────────────────────────────\n');
  log(`Installation complete! ${filesCopied} files installed.`, 'success');
  console.log('\n  Next steps:\n');
  console.log('    1. Run /ant:init "your goal" in any project');
  console.log('    2. Use /ant:build to execute phases');
  console.log('    3. Run /ant:help for command reference\n');
  console.log('  ─────────────────────────────────────────────\n');
}

// Run installer
install();
