/**
 * interactive-setup.js — Interactive menu for npx aether-colony
 *
 * Displays an environment-aware menu with three options:
 *   [1] Full setup    — Install globally + set up this repo
 *   [2] Global only   — Install hub, commands, and agents (~/.aether/)
 *   [3] Repo only     — Set up Aether in this directory (.aether/)
 *
 * Zero npm dependencies — uses built-in Node readline.
 */

const fs = require('fs');
const path = require('path');
const os = require('os');
const readline = require('readline');

const { BANNER } = require('./banner');

const VERSION = require('../../package.json').version;
const HOME_DIR = os.homedir();
const HUB_VERSION_PATH = path.join(HOME_DIR, '.aether', 'version.json');

/**
 * Detect the current environment state.
 * @returns {{ hubInstalled: boolean, hasAether: boolean, isProjectDir: boolean }}
 */
function detectEnvironment() {
  const cwd = process.cwd();

  const hubInstalled = fs.existsSync(HUB_VERSION_PATH);

  const hasAether = fs.existsSync(path.join(cwd, '.aether', 'aether-utils.sh'));

  const isProjectDir =
    fs.existsSync(path.join(cwd, '.git')) ||
    fs.existsSync(path.join(cwd, 'package.json')) ||
    fs.existsSync(path.join(cwd, 'Makefile')) ||
    fs.existsSync(path.join(cwd, 'pyproject.toml')) ||
    fs.existsSync(path.join(cwd, 'Cargo.toml'));

  return { hubInstalled, hasAether, isProjectDir };
}

/**
 * Choose the context-sensitive default menu option.
 * @param {{ hubInstalled: boolean, hasAether: boolean, isProjectDir: boolean }} env
 * @returns {1|2|3}
 */
function getDefaultOption(env) {
  if (!env.hubInstalled && env.isProjectDir) return 1;
  if (!env.hubInstalled) return 2;
  if (env.hubInstalled && !env.hasAether) return 3;
  return 1;
}

/**
 * Readline promise helper.
 * @param {readline.Interface} rl
 * @param {string} question
 * @returns {Promise<string>}
 */
function prompt(rl, question) {
  return new Promise(resolve => rl.question(question, resolve));
}

/**
 * Log a message with 🐜 ant prefix.
 * @param {string} msg
 */
function log(msg) {
  console.log(`🐜 ${msg}`);
}

/**
 * Main interactive setup entry point.
 * Handles --global, --repo, --yes flags and non-TTY environments.
 */
async function interactiveSetup() {
  const args = process.argv.slice(2);
  const flagGlobal = args.includes('--global');
  const flagRepo = args.includes('--repo');
  const flagYes = args.includes('--yes');

  // Import performGlobalInstall lazily to avoid circular require issues
  const { performGlobalInstall } = require('../cli');
  const { initializeRepo } = require('./init');

  const env = detectEnvironment();

  // Non-TTY: auto-pick default without prompting
  if (!process.stdin.isTTY && !flagGlobal && !flagRepo && !flagYes) {
    const choice = getDefaultOption(env);
    await executeChoice(choice, env, performGlobalInstall, initializeRepo);
    return;
  }

  // Flag shortcuts: skip menu entirely
  if (flagGlobal) {
    await executeChoice(2, env, performGlobalInstall, initializeRepo);
    return;
  }
  if (flagRepo) {
    await executeChoice(3, env, performGlobalInstall, initializeRepo);
    return;
  }

  // Already fully set up: offer refresh
  if (env.hubInstalled && env.hasAether) {
    const rl = readline.createInterface({ input: process.stdin, output: process.stdout });
    try {
      console.log(BANNER);
      console.log(`  Aether Colony v${VERSION}\n`);
      log('Aether is already set up in this directory.');

      let answer;
      if (flagYes) {
        answer = 'y';
      } else {
        answer = await prompt(rl, '\n  Already set up. Refresh? (y/n) [n]: ');
      }

      if (answer.trim().toLowerCase() === 'y') {
        await executeChoice(1, env, performGlobalInstall, initializeRepo);
      } else {
        log('Nothing changed. Run /ant:init "your goal" to start a colony.');
      }
    } finally {
      rl.close();
    }
    return;
  }

  // Interactive menu
  const defaultChoice = flagYes ? getDefaultOption(env) : null;

  const rl = readline.createInterface({ input: process.stdin, output: process.stdout });
  try {
    console.log(BANNER);
    console.log(`  Aether Colony v${VERSION}\n`);

    const defaultOption = getDefaultOption(env);
    const option3Disabled = !env.isProjectDir;

    console.log('  What would you like to do?\n');
    console.log(`  [1] Full setup    — Install globally + set up this repo${defaultOption === 1 ? ' (recommended)' : ''}`);
    console.log(`  [2] Global only   — Install hub, commands, and agents (~/.aether/)${defaultOption === 2 ? ' (recommended)' : ''}`);
    if (option3Disabled) {
      console.log('  [3] Repo only     — (not available: no project found in current directory)');
    } else {
      console.log(`  [3] Repo only     — Set up Aether in this directory (.aether/)${defaultOption === 3 ? ' (recommended)' : ''}`);
    }
    console.log('');

    let choice;
    if (flagYes) {
      choice = defaultOption;
      console.log(`  Auto-selected [${choice}] (--yes flag)\n`);
    } else {
      const raw = await prompt(rl, `  Enter choice [${defaultOption}]: `);
      const trimmed = raw.trim();
      choice = trimmed === '' ? defaultOption : parseInt(trimmed, 10);
    }

    if (isNaN(choice) || choice < 1 || choice > 3) {
      console.error('\n  Invalid choice. Please run again and select 1, 2, or 3.\n');
      process.exit(1);
    }

    if (choice === 3 && option3Disabled) {
      console.error('\n  Option 3 is not available outside a project directory.\n');
      process.exit(1);
    }

    await executeChoice(choice, env, performGlobalInstall, initializeRepo);
  } finally {
    rl.close();
  }
}

/**
 * Execute the selected menu option.
 * @param {1|2|3} choice
 * @param {{ hubInstalled: boolean, hasAether: boolean, isProjectDir: boolean }} env
 * @param {Function} performGlobalInstall
 * @param {Function} initializeRepo
 */
async function executeChoice(choice, env, performGlobalInstall, initializeRepo) {
  const cwd = process.cwd();

  if (choice === 1) {
    log('Running full setup...');
    await performGlobalInstall();
    const result = await initializeRepo(cwd, { setupOnly: true });
    printRepoSuccess(result);
  } else if (choice === 2) {
    log('Running global install...');
    await performGlobalInstall();
    printGlobalSuccess();
  } else if (choice === 3) {
    if (!env.hubInstalled) {
      console.error('\n  Aether hub not installed. Run without --repo to install globally first.\n');
      process.exit(1);
    }
    log('Setting up this repository...');
    const result = await initializeRepo(cwd, { setupOnly: true });
    printRepoSuccess(result);
  }
}

/**
 * Print success message after global install.
 */
function printGlobalSuccess() {
  console.log('');
  log('Global install complete.');
  console.log('');
  console.log('  Next steps:');
  console.log('    cd into a project, then run: npx aether-colony --repo');
  console.log('    Or: aether init --goal "your goal"');
  console.log('');
}

/**
 * Print success message after repo setup.
 * @param {{ success: boolean, filesCopied?: number }} result
 */
function printRepoSuccess(result) {
  if (!result || !result.success) {
    console.error('\n  Repo setup failed. Check that the Aether hub is installed.\n');
    return;
  }
  console.log('');
  log('Aether is ready.');
  if (result.filesCopied != null) {
    console.log(`  ${result.filesCopied} system files synced to .aether/`);
  }
  console.log('');
  console.log('  Next steps:');
  console.log('    In Claude Code: /ant:init "your goal"');
  console.log('    Or terminal:    aether init --goal "your goal"');
  console.log('');
}

module.exports = {
  interactiveSetup,
  detectEnvironment,
  getDefaultOption,
  executeChoice,
};
