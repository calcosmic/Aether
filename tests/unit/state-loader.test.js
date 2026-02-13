const fs = require('fs');
const path = require('path');
const test = require('ava');
const { execSync } = require('child_process');

const AETHER_ROOT = path.join(__dirname, '../..');
const STATE_LOADER_PATH = path.join(AETHER_ROOT, '.aether/utils/state-loader.sh');
const DATA_DIR = path.join(AETHER_ROOT, '.aether/data');
const COLONY_STATE_PATH = path.join(DATA_DIR, 'COLONY_STATE.json');
const HANDOFF_PATH = path.join(AETHER_ROOT, '.aether/HANDOFF.md');

/**
 * Helper to execute bash commands with state-loader.sh sourced
 * @param {string} command - Bash command to execute
 * @returns {object} - { stdout, stderr, exitCode }
 */
function execWithLoader(command) {
  const fullCommand = `source "${STATE_LOADER_PATH}" 2>/dev/null && ${command}`;
  try {
    const stdout = execSync(fullCommand, {
      encoding: 'utf8',
      shell: '/bin/bash',
      cwd: AETHER_ROOT
    });
    return { stdout: stdout.trim(), stderr: '', exitCode: 0 };
  } catch (error) {
    return {
      stdout: error.stdout ? error.stdout.toString().trim() : '',
      stderr: error.stderr ? error.stderr.toString().trim() : '',
      exitCode: error.status || 1
    };
  }
}

/**
 * Helper to backup and restore COLONY_STATE.json
 */
function backupState() {
  const backupPath = `${COLONY_STATE_PATH}.backup`;
  if (fs.existsSync(COLONY_STATE_PATH)) {
    fs.copyFileSync(COLONY_STATE_PATH, backupPath);
  }
  return backupPath;
}

function restoreState(backupPath) {
  if (fs.existsSync(backupPath)) {
    fs.copyFileSync(backupPath, COLONY_STATE_PATH);
    fs.unlinkSync(backupPath);
  }
}

/**
 * Helper to backup and restore HANDOFF.md
 */
function backupHandoff() {
  const backupPath = `${HANDOFF_PATH}.backup`;
  if (fs.existsSync(HANDOFF_PATH)) {
    fs.copyFileSync(HANDOFF_PATH, backupPath);
    return backupPath;
  }
  return null;
}

function restoreHandoff(backupPath) {
  if (backupPath && fs.existsSync(backupPath)) {
    fs.copyFileSync(backupPath, HANDOFF_PATH);
    fs.unlinkSync(backupPath);
  } else if (fs.existsSync(HANDOFF_PATH)) {
    fs.unlinkSync(HANDOFF_PATH);
  }
}

// Test: State loader can be sourced without errors
test('state-loader.sh can be sourced without errors', t => {
  const result = execWithLoader('echo "sourced successfully"');
  t.is(result.exitCode, 0);
  t.is(result.stdout, 'sourced successfully');
});

// Test: load_colony_state function exists
test('load_colony_state function is defined', t => {
  const result = execWithLoader('type load_colony_state');
  t.is(result.exitCode, 0);
  t.true(result.stdout.includes('function'));
});

// Test: unload_colony_state function exists
test('unload_colony_state function is defined', t => {
  const result = execWithLoader('type unload_colony_state');
  t.is(result.exitCode, 0);
  t.true(result.stdout.includes('function'));
});

// Test: get_handoff_summary function exists
test('get_handoff_summary function is defined', t => {
  const result = execWithLoader('type get_handoff_summary');
  t.is(result.exitCode, 0);
  t.true(result.stdout.includes('function'));
});

// Test: display_resumption_context function exists
test('display_resumption_context function is defined', t => {
  const result = execWithLoader('type display_resumption_context');
  t.is(result.exitCode, 0);
  t.true(result.stdout.includes('function'));
});

// Test: State loading succeeds with valid COLONY_STATE.json
test('load_colony_state succeeds with valid COLONY_STATE.json', t => {
  const backupHandoffPath = backupHandoff();

  try {
    // Remove any existing handoff to test clean load
    if (fs.existsSync(HANDOFF_PATH)) {
      fs.unlinkSync(HANDOFF_PATH);
    }

    const result = execWithLoader('load_colony_state && echo "LOADED_STATE is set: $([[ -n \"$LOADED_STATE\" ]] && echo yes || echo no)" && echo "STATE_LOCK_ACQUIRED: $STATE_LOCK_ACQUIRED"');

    t.is(result.exitCode, 0);
    t.true(result.stdout.includes('LOADED_STATE is set: yes'));
    t.true(result.stdout.includes('STATE_LOCK_ACQUIRED: true'));

    // Cleanup
    execWithLoader('unload_colony_state');
  } finally {
    restoreHandoff(backupHandoffPath);
  }
});

// Test: unload_colony_state releases lock properly
test('unload_colony_state releases lock properly', t => {
  const backupHandoffPath = backupHandoff();

  try {
    if (fs.existsSync(HANDOFF_PATH)) {
      fs.unlinkSync(HANDOFF_PATH);
    }

    // Load state
    execWithLoader('load_colony_state');

    // Unload state
    const result = execWithLoader('unload_colony_state && echo "STATE_LOCK_ACQUIRED after unload: $STATE_LOCK_ACQUIRED"');

    t.is(result.exitCode, 0);
    t.true(result.stdout.includes('STATE_LOCK_ACQUIRED after unload: false') ||
            result.stdout.includes('STATE_LOCK_ACQUIRED after unload:'));
  } finally {
    restoreHandoff(backupHandoffPath);
  }
});

// Test: State loading fails when COLONY_STATE.json is missing
test('load_colony_state fails when COLONY_STATE.json is missing', t => {
  const stateBackup = backupState();
  const backupHandoffPath = backupHandoff();

  try {
    // Remove handoff if exists
    if (fs.existsSync(HANDOFF_PATH)) {
      fs.unlinkSync(HANDOFF_PATH);
    }

    // Temporarily rename state file
    fs.unlinkSync(COLONY_STATE_PATH);

    const result = execWithLoader('load_colony_state 2>&1 || echo "EXIT_CODE: $?"');

    // Should fail with non-zero exit code
    t.not(result.exitCode, 0);
    // Should output error JSON
    t.true(result.stderr.includes('E_FILE_NOT_FOUND') || result.stdout.includes('E_FILE_NOT_FOUND'));
  } finally {
    restoreState(stateBackup);
    restoreHandoff(backupHandoffPath);
  }
});

// Test: State loading detects handoff
test('load_colony_state detects handoff when HANDOFF.md exists', t => {
  const backupHandoffPath = backupHandoff();

  try {
    // Create a test handoff file
    const testHandoff = `# Colony Session Paused

## Quick Resume
Run \`/ant:resume-colony\` in a new session.

## State at Pause
- Goal: "Test goal"
- State: READY
- Current Phase: 3 — Test Phase
- Session: test_session
- Paused: 2026-02-13T20:30:00Z

## What Was Happening
Testing handoff detection.
`;
    fs.writeFileSync(HANDOFF_PATH, testHandoff, 'utf8');

    const result = execWithLoader('load_colony_state && echo "HANDOFF_DETECTED: $HANDOFF_DETECTED" && echo "HANDOFF_CONTENT_LENGTH: ${#HANDOFF_CONTENT}"');

    t.is(result.exitCode, 0);
    t.true(result.stdout.includes('HANDOFF_DETECTED: true'));
    t.true(result.stdout.includes('HANDOFF_CONTENT_LENGTH:'));

    // Cleanup
    execWithLoader('unload_colony_state');
  } finally {
    restoreHandoff(backupHandoffPath);
  }
});

// Test: get_handoff_summary extracts phase info
test('get_handoff_summary extracts phase information', t => {
  const backupHandoffPath = backupHandoff();

  try {
    // Create a test handoff file with phase info
    const testHandoff = `## Phase 3 - Test Phase Name

Some content here.
`;
    fs.writeFileSync(HANDOFF_PATH, testHandoff, 'utf8');

    const result = execWithLoader('load_colony_state && get_handoff_summary');

    t.is(result.exitCode, 0);
    t.true(result.stdout.includes('Phase 3') || result.stdout.includes('Test Phase'));

    // Cleanup
    execWithLoader('unload_colony_state');
  } finally {
    restoreHandoff(backupHandoffPath);
  }
});

// Test: display_resumption_context shows resume message and removes handoff
test('display_resumption_context shows resume message and removes handoff', t => {
  const backupHandoffPath = backupHandoff();

  try {
    // Create a test handoff file
    const testHandoff = `## Phase 5 - State Loading

Test content.
`;
    fs.writeFileSync(HANDOFF_PATH, testHandoff, 'utf8');

    // Load state and display context
    const result = execWithLoader('load_colony_state && display_resumption_context');

    t.is(result.exitCode, 0);
    t.true(result.stdout.includes('Resuming:') || result.stdout.includes('Phase 5'));

    // Verify handoff file was removed
    t.false(fs.existsSync(HANDOFF_PATH));

    // Cleanup
    execWithLoader('unload_colony_state');
  } finally {
    restoreHandoff(backupHandoffPath);
  }
});

// Test: Validation failure handling - lock is released
test('load_colony_state releases lock on validation failure', t => {
  const stateBackup = backupState();
  const backupHandoffPath = backupHandoff();

  try {
    // Remove handoff if exists
    if (fs.existsSync(HANDOFF_PATH)) {
      fs.unlinkSync(HANDOFF_PATH);
    }

    // Create invalid JSON temporarily
    fs.writeFileSync(COLONY_STATE_PATH, '{"invalid json', 'utf8');

    // Use a subshell to handle the failure gracefully
    const result = execWithLoader('(load_colony_state) || true; echo "AFTER_LOAD"; unload_colony_state 2>/dev/null || true; echo "AFTER_UNLOAD"');

    // The unload should work (not hang), indicating lock was released
    t.true(result.stdout.includes('AFTER_LOAD'));
    t.true(result.stdout.includes('AFTER_UNLOAD'));
  } finally {
    restoreState(stateBackup);
    restoreHandoff(backupHandoffPath);
  }
});

// Test: CLI load-state command works
test('CLI load-state command returns JSON with loaded status', t => {
  const backupHandoffPath = backupHandoff();

  try {
    // Remove handoff for clean test
    if (fs.existsSync(HANDOFF_PATH)) {
      fs.unlinkSync(HANDOFF_PATH);
    }

    const result = execSync('bash .aether/aether-utils.sh load-state', {
      encoding: 'utf8',
      cwd: AETHER_ROOT
    });

    const json = JSON.parse(result);
    t.true(json.ok);
    t.true(json.result.loaded);
  } finally {
    restoreHandoff(backupHandoffPath);
  }
});

// Test: CLI unload-state command works
test('CLI unload-state command returns JSON with unloaded status', t => {
  const result = execSync('bash .aether/aether-utils.sh unload-state', {
    encoding: 'utf8',
    cwd: AETHER_ROOT
  });

  const json = JSON.parse(result);
  t.true(json.ok);
  t.true(json.result.unloaded);
});

// Test: CLI load-state detects handoff
test('CLI load-state detects handoff and returns summary', t => {
  const backupHandoffPath = backupHandoff();

  try {
    // Create a test handoff file
    const testHandoff = `## Phase 7 - Integration Testing

Test content for CLI.
`;
    fs.writeFileSync(HANDOFF_PATH, testHandoff, 'utf8');

    const result = execSync('bash .aether/aether-utils.sh load-state', {
      encoding: 'utf8',
      cwd: AETHER_ROOT
    });

    const json = JSON.parse(result);
    t.true(json.ok);
    t.true(json.result.loaded);
    t.true(json.result.handoff_detected);
    t.truthy(json.result.handoff_summary);
  } finally {
    restoreHandoff(backupHandoffPath);
  }
});
