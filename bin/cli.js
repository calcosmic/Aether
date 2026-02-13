#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

const VERSION = require('../package.json').version;
const PACKAGE_DIR = path.resolve(__dirname, '..');
const HOME = process.env.HOME;

// Claude Code paths (global)
const COMMANDS_SRC = path.join(PACKAGE_DIR, 'commands', 'ant');
const COMMANDS_DEST = path.join(HOME, '.claude', 'commands', 'ant');

// Hub paths
const HUB_DIR = path.join(HOME, '.aether');
const HUB_SYSTEM = path.join(HUB_DIR, 'system');
const HUB_COMMANDS_CLAUDE = path.join(HUB_DIR, 'commands', 'claude');
const HUB_COMMANDS_OPENCODE = path.join(HUB_DIR, 'commands', 'opencode');
const HUB_AGENTS = path.join(HUB_DIR, 'agents');
const HUB_REGISTRY = path.join(HUB_DIR, 'registry.json');
const HUB_VERSION = path.join(HUB_DIR, 'version.json');

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

// System files allowlist — only these are copied during updates (never colony data)
const SYSTEM_FILES = [
  'aether-utils.sh',
  'coding-standards.md',
  'debugging.md',
  'DISCIPLINES.md',
  'learning.md',
  'planning.md',
  'QUEEN_ANT_ARCHITECTURE.md',
  'tdd.md',
  'verification-loop.md',
  'verification.md',
  'workers.md',
  'docs/constraints.md',
  'docs/pathogen-schema-example.json',
  'docs/pathogen-schema.md',
  'docs/pheromones.md',
  'docs/progressive-disclosure.md',
  'utils/atomic-write.sh',
  'utils/colorize-log.sh',
  'utils/file-lock.sh',
  'utils/watch-spawn-tree.sh',
];

function copySystemFiles(srcDir, destDir) {
  let count = 0;
  for (const file of SYSTEM_FILES) {
    const srcPath = path.join(srcDir, file);
    const destPath = path.join(destDir, file);
    if (fs.existsSync(srcPath)) {
      fs.mkdirSync(path.dirname(destPath), { recursive: true });
      fs.copyFileSync(srcPath, destPath);
      if (file.endsWith('.sh')) {
        fs.chmodSync(destPath, 0o755);
      }
      count++;
    }
  }
  return count;
}

function readJsonSafe(filePath) {
  try {
    return JSON.parse(fs.readFileSync(filePath, 'utf8'));
  } catch {
    return null;
  }
}

function writeJsonSync(filePath, data) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, JSON.stringify(data, null, 2) + '\n');
}

function setupHub() {
  // Create ~/.aether/ directory structure and populate from package
  try {
    fs.mkdirSync(HUB_DIR, { recursive: true });

    // Copy runtime/ -> ~/.aether/system/
    const runtimeSrc = path.join(PACKAGE_DIR, 'runtime');
    if (fs.existsSync(runtimeSrc)) {
      const n = copyDirSync(runtimeSrc, HUB_SYSTEM);
      log(`  Hub system: ${n} files -> ${HUB_SYSTEM}`);
    }

    // Copy .claude/commands/ant/ -> ~/.aether/commands/claude/
    const claudeCmdSrc = fs.existsSync(COMMANDS_SRC)
      ? COMMANDS_SRC
      : path.join(PACKAGE_DIR, '.claude', 'commands', 'ant');
    if (fs.existsSync(claudeCmdSrc)) {
      const n = copyDirSync(claudeCmdSrc, HUB_COMMANDS_CLAUDE);
      log(`  Hub commands (claude): ${n} files -> ${HUB_COMMANDS_CLAUDE}`);
    }

    // Copy .opencode/commands/ant/ -> ~/.aether/commands/opencode/
    const opencodeCmdSrc = path.join(PACKAGE_DIR, '.opencode', 'commands', 'ant');
    if (fs.existsSync(opencodeCmdSrc)) {
      const n = copyDirSync(opencodeCmdSrc, HUB_COMMANDS_OPENCODE);
      log(`  Hub commands (opencode): ${n} files -> ${HUB_COMMANDS_OPENCODE}`);
    }

    // Copy .opencode/agents/ -> ~/.aether/agents/
    const agentsSrc = path.join(PACKAGE_DIR, '.opencode', 'agents');
    if (fs.existsSync(agentsSrc)) {
      const n = copyDirSync(agentsSrc, HUB_AGENTS);
      log(`  Hub agents: ${n} files -> ${HUB_AGENTS}`);
    }

    // Create/preserve registry.json
    if (!fs.existsSync(HUB_REGISTRY)) {
      writeJsonSync(HUB_REGISTRY, { schema_version: 1, repos: [] });
      log(`  Registry: initialized ${HUB_REGISTRY}`);
    } else {
      log(`  Registry: preserved existing ${HUB_REGISTRY}`);
    }

    // Write version.json
    writeJsonSync(HUB_VERSION, { version: VERSION, updated_at: new Date().toISOString() });
    log(`  Hub version: ${VERSION}`);
  } catch (err) {
    // Hub setup failure doesn't block install
    log(`  Hub setup warning: ${err.message}`);
  }
}

function updateRepo(repoPath, sourceVersion) {
  const repoAether = path.join(repoPath, '.aether');
  const repoVersionFile = path.join(repoAether, 'version.json');

  if (!fs.existsSync(repoAether)) {
    return { status: 'skipped', reason: 'no .aether directory' };
  }

  const currentVersion = readJsonSafe(repoVersionFile);
  const currentVer = currentVersion ? currentVersion.version : 'unknown';

  // Copy system files from hub
  const systemCount = copySystemFiles(HUB_SYSTEM, repoAether);

  // Copy commands from hub
  let commandCount = 0;
  const repoClaudeCmds = path.join(repoPath, '.claude', 'commands', 'ant');
  if (fs.existsSync(HUB_COMMANDS_CLAUDE)) {
    commandCount += copyDirSync(HUB_COMMANDS_CLAUDE, repoClaudeCmds);
  }

  const repoOpencodeCmds = path.join(repoPath, '.opencode', 'commands', 'ant');
  if (fs.existsSync(HUB_COMMANDS_OPENCODE)) {
    commandCount += copyDirSync(HUB_COMMANDS_OPENCODE, repoOpencodeCmds);
  }

  // Copy agents from hub
  let agentCount = 0;
  const repoAgents = path.join(repoPath, '.opencode', 'agents');
  if (fs.existsSync(HUB_AGENTS)) {
    agentCount += copyDirSync(HUB_AGENTS, repoAgents);
  }

  // Write version.json
  writeJsonSync(repoVersionFile, { version: sourceVersion, updated_at: new Date().toISOString() });

  // Update registry entry
  const registry = readJsonSafe(HUB_REGISTRY);
  if (registry) {
    const ts = new Date().toISOString();
    const existing = registry.repos.find(r => r.path === repoPath);
    if (existing) {
      existing.version = sourceVersion;
      existing.updated_at = ts;
    } else {
      registry.repos.push({ path: repoPath, version: sourceVersion, registered_at: ts, updated_at: ts });
    }
    writeJsonSync(HUB_REGISTRY, registry);
  }

  return { status: 'updated', from: currentVer, to: sourceVersion, system: systemCount, commands: commandCount, agents: agentCount };
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

    // Set up distribution hub at ~/.aether/
    log('');
    log('Setting up distribution hub...');
    setupHub();

    log('');
    log('Install complete.');
    log('  Claude Code: run /ant to get started');
    log('  Hub: ~/.aether/ (for coordinated updates across repos)');
    break;
  }

  case 'update': {
    const forceFlag = flags.includes('--force');
    const allFlag = flags.includes('--all');
    const listFlag = flags.includes('--list');

    // Check hub exists
    if (!fs.existsSync(HUB_VERSION)) {
      console.error('No distribution hub found at ~/.aether/');
      console.error('Run `aether install` first to set up the hub.');
      process.exit(1);
    }

    const hubVersion = readJsonSafe(HUB_VERSION);
    const sourceVersion = hubVersion ? hubVersion.version : VERSION;

    if (listFlag) {
      // Show registered repos
      const registry = readJsonSafe(HUB_REGISTRY);
      if (!registry || registry.repos.length === 0) {
        console.log('No repos registered. Run /ant:init in a repo to register it.');
        break;
      }
      console.log(`Registered repos (hub v${sourceVersion}):\n`);
      for (const repo of registry.repos) {
        const exists = fs.existsSync(repo.path);
        const status = exists ? `v${repo.version}` : 'NOT FOUND';
        const marker = exists ? (repo.version === sourceVersion ? '  ' : '* ') : 'x ';
        console.log(`${marker}${repo.path}  (${status})`);
      }
      console.log('');
      console.log('* = update available, x = path no longer exists');
      break;
    }

    if (allFlag) {
      // Update all registered repos
      const registry = readJsonSafe(HUB_REGISTRY);
      if (!registry || registry.repos.length === 0) {
        console.log('No repos registered. Run /ant:init in a repo to register it.');
        break;
      }

      let updated = 0;
      let upToDate = 0;
      let pruned = 0;
      const survivingRepos = [];

      for (const repo of registry.repos) {
        if (!fs.existsSync(repo.path)) {
          log(`  Pruned: ${repo.path} (no longer exists)`);
          pruned++;
          continue;
        }

        survivingRepos.push(repo);

        if (!forceFlag && repo.version === sourceVersion) {
          log(`  Up-to-date: ${repo.path} (v${repo.version})`);
          upToDate++;
          continue;
        }

        const result = updateRepo(repo.path, sourceVersion);
        if (result.status === 'updated') {
          log(`  Updated: ${repo.path} (${result.from} -> ${result.to}) [${result.system} system, ${result.commands} commands, ${result.agents} agents]`);
          updated++;
        } else {
          log(`  Skipped: ${repo.path} (${result.reason})`);
        }
      }

      // Save pruned registry
      if (pruned > 0) {
        registry.repos = survivingRepos;
        writeJsonSync(HUB_REGISTRY, registry);
      }

      console.log(`\nSummary: ${updated} updated, ${upToDate} up-to-date, ${pruned} pruned`);
    } else {
      // Update current repo
      const repoPath = process.cwd();
      const repoAether = path.join(repoPath, '.aether');

      if (!fs.existsSync(repoAether)) {
        console.error('No .aether/ directory found in current repo.');
        console.error('Run /ant:init in this repo first.');
        process.exit(1);
      }

      const currentVersion = readJsonSafe(path.join(repoAether, 'version.json'));
      const currentVer = currentVersion ? currentVersion.version : 'unknown';

      if (!forceFlag && currentVer === sourceVersion) {
        console.log(`Already up-to-date (v${sourceVersion}).`);
        break;
      }

      const result = updateRepo(repoPath, sourceVersion);
      console.log(`Updated: ${result.from} -> ${result.to}`);
      console.log(`  ${result.system} system files, ${result.commands} command files, ${result.agents} agent files`);
      console.log('  Colony data (.aether/data/) untouched.');
    }
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
    log('Hub at ~/.aether/ preserved (remove manually if desired).');
    break;
  }

  case 'help':
  default: {
    console.log(`
aether-colony v${VERSION}

Usage: aether <command> [options]

Commands:
  install              Install slash-commands and set up distribution hub
  update               Update current repo from hub
  update --all         Update all registered repos
  update --all --force Force update all (even if versions match)
  update --list        Show registered repos and versions
  version              Show installed version
  uninstall            Remove slash-commands (preserves project state and hub)
  help                 Show this help message

Locations:
  Commands:  ~/.claude/commands/ant/
  Hub:       ~/.aether/ (distribution hub for coordinated updates)

After install, run /ant in Claude Code to get started.
Project state lives in each repo's .aether/data/ directory.
`.trim());
    break;
  }
}
