#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const VERSION = require('../package.json').version;
const PACKAGE_DIR = path.resolve(__dirname, '..');

// Claude Code paths (global)
const COMMANDS_SRC = path.join(PACKAGE_DIR, 'commands', 'ant');
const RUNTIME_SRC = path.join(PACKAGE_DIR, 'runtime');
const COMMANDS_DEST = path.join(process.env.HOME, '.claude', 'commands', 'ant');
const RUNTIME_DEST = path.join(process.env.HOME, '.aether');

// OpenCode paths (global — ~/.config/opencode/)
const OPENCODE_COMMANDS_SRC = path.join(PACKAGE_DIR, 'opencode', 'commands', 'ant');
const OPENCODE_AGENTS_SRC = path.join(PACKAGE_DIR, 'opencode', 'agents');
const OPENCODE_GLOBAL_COMMANDS_DEST = path.join(process.env.HOME, '.config', 'opencode', 'commands', 'ant');
const OPENCODE_GLOBAL_AGENTS_DEST = path.join(process.env.HOME, '.config', 'opencode', 'agents');

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

    // Copy runtime to ~/.aether/
    if (!fs.existsSync(RUNTIME_SRC)) {
      // Running from source repo — runtime is in .aether/
      const repoRuntime = path.join(PACKAGE_DIR, '.aether');
      if (fs.existsSync(repoRuntime)) {
        // Only copy system files, not data/
        const runtimeFiles = [
          'aether-utils.sh',
          'QUEEN_ANT_ARCHITECTURE.md',
          'workers.md',
          'DISCIPLINES.md',
          'verification.md',
          'verification-loop.md',
          'debugging.md',
          'tdd.md',
          'learning.md',
          'coding-standards.md',
          'planning.md'
        ];
        const runtimeDirs = ['workers', 'utils', 'docs'];

        fs.mkdirSync(RUNTIME_DEST, { recursive: true });
        let count = 0;
        for (const f of runtimeFiles) {
          const src = path.join(repoRuntime, f);
          if (fs.existsSync(src)) {
            fs.copyFileSync(src, path.join(RUNTIME_DEST, f));
            if (f.endsWith('.sh')) fs.chmodSync(path.join(RUNTIME_DEST, f), 0o755);
            count++;
          }
        }
        for (const d of runtimeDirs) {
          const src = path.join(repoRuntime, d);
          if (fs.existsSync(src)) {
            count += copyDirSync(src, path.join(RUNTIME_DEST, d));
          }
        }
        log(`  Runtime: ${count} files -> ${RUNTIME_DEST}`);
      } else {
        console.error('  Runtime source not found. Skipping.');
      }
    } else {
      const n = copyDirSync(RUNTIME_SRC, RUNTIME_DEST);
      log(`  Runtime: ${n} files -> ${RUNTIME_DEST}`);
    }

    // Ensure learnings.json exists
    const learningsFile = path.join(RUNTIME_DEST, 'learnings.json');
    if (!fs.existsSync(learningsFile)) {
      fs.writeFileSync(learningsFile, '{"learnings":[],"version":1}\n');
      log('  Created: ~/.aether/learnings.json');
    }

    // Install OpenCode commands globally to ~/.config/opencode/
    {
      let opencodeCommandsSrc = OPENCODE_COMMANDS_SRC;
      let opencodeAgentsSrc = OPENCODE_AGENTS_SRC;

      // Running from source repo
      if (!fs.existsSync(opencodeCommandsSrc)) {
        opencodeCommandsSrc = path.join(PACKAGE_DIR, '.opencode', 'commands', 'ant');
        opencodeAgentsSrc = path.join(PACKAGE_DIR, '.opencode', 'agents');
      }

      if (fs.existsSync(opencodeCommandsSrc)) {
        const cmdCount = copyDirSync(opencodeCommandsSrc, OPENCODE_GLOBAL_COMMANDS_DEST);
        log(`  OpenCode Commands: ${cmdCount} files -> ${OPENCODE_GLOBAL_COMMANDS_DEST}`);
      }

      if (fs.existsSync(opencodeAgentsSrc)) {
        const agentCount = copyDirSync(opencodeAgentsSrc, OPENCODE_GLOBAL_AGENTS_DEST);
        log(`  OpenCode Agents: ${agentCount} files -> ${OPENCODE_GLOBAL_AGENTS_DEST}`);
      }
    }

    log('');
    log('Install complete.');
    log('  Claude Code: run /ant to get started');
    log('  OpenCode:    run /ant to get started');
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

    // Remove runtime (but preserve learnings.json)
    if (fs.existsSync(RUNTIME_DEST)) {
      const learningsFile = path.join(RUNTIME_DEST, 'learnings.json');
      let learningsBackup = null;
      if (fs.existsSync(learningsFile)) {
        learningsBackup = fs.readFileSync(learningsFile, 'utf8');
      }

      const n = removeDirSync(RUNTIME_DEST);
      log(`  Removed: ${n} runtime files from ${RUNTIME_DEST}`);

      // Restore learnings
      if (learningsBackup) {
        fs.mkdirSync(RUNTIME_DEST, { recursive: true });
        fs.writeFileSync(learningsFile, learningsBackup);
        log('  Preserved: ~/.aether/learnings.json (cross-project learnings)');
      }
    } else {
      log('  Runtime already removed.');
    }

    // Remove OpenCode global commands
    let opencodeRemoved = 0;
    if (fs.existsSync(OPENCODE_GLOBAL_COMMANDS_DEST)) {
      opencodeRemoved += removeDirSync(OPENCODE_GLOBAL_COMMANDS_DEST);
    }
    // Only remove Aether-owned agent files, not the entire agents directory
    if (fs.existsSync(OPENCODE_GLOBAL_AGENTS_DEST)) {
      const aetherAgents = ['aether-queen.md', 'aether-builder.md', 'aether-scout.md', 'aether-watcher.md'];
      for (const agent of aetherAgents) {
        const agentPath = path.join(OPENCODE_GLOBAL_AGENTS_DEST, agent);
        if (fs.existsSync(agentPath)) {
          fs.unlinkSync(agentPath);
          opencodeRemoved++;
        }
      }
    }
    if (opencodeRemoved > 0) {
      log(`  Removed: ${opencodeRemoved} OpenCode files from ~/.config/opencode/`);
    } else {
      log('  OpenCode commands already removed.');
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
  install      Set up commands and runtime for both Claude Code and OpenCode
  version      Show installed version
  uninstall    Remove commands and runtime (preserves learnings and project state)
  help         Show this help message

Install locations:
  Claude Code: ~/.claude/commands/ant/ (global)
  OpenCode:    ~/.config/opencode/commands/ant/ (global)
  Runtime:     ~/.aether/ (global utilities)

After install, run /ant in either tool to get started.
Both tools share state in .aether/data/ for seamless switching.
`.trim());
    break;
  }
}
