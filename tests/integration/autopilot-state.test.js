/**
 * Autopilot State Machine Integration Tests
 *
 * Tests for Phase 4 autopilot state machine transitions:
 * - State initialization (autopilot-init creates valid run-state.json)
 * - Phase tracking (autopilot-update -> autopilot-status round-trip)
 * - Pause conditions (autopilot-stop preserves phase_results)
 * - Replan triggers (autopilot-check-replan at correct intervals)
 * - Full lifecycle (init -> update -> check-replan -> update -> stop)
 *
 * Also verifies two bug fixes found by Chaos Ant:
 * - FIX 1: autopilot-check-replan rejects --interval 0
 * - FIX 2: phases_completed_in_run only increments on action=="advance"
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

async function createTempDir() {
  return fs.promises.mkdtemp(path.join(os.tmpdir(), 'aether-autopilot-'));
}

async function cleanupTempDir(tmpDir) {
  try {
    await fs.promises.rm(tmpDir, { recursive: true, force: true });
  } catch {
    // Ignore cleanup errors
  }
}

function runAetherUtil(tmpDir, command, args = []) {
  const scriptPath = path.join(process.cwd(), '.aether', 'aether-utils.sh');
  const env = {
    ...process.env,
    AETHER_ROOT: tmpDir,
    DATA_DIR: path.join(tmpDir, '.aether', 'data')
  };
  const escapedArgs = args.map(a => `"${a}"`).join(' ');
  const cmd = `bash "${scriptPath}" ${command} ${escapedArgs} 2>/dev/null`;
  return execSync(cmd, { encoding: 'utf8', env, cwd: tmpDir });
}

function runAetherUtilSafe(tmpDir, command, args = []) {
  const scriptPath = path.join(process.cwd(), '.aether', 'aether-utils.sh');
  const env = {
    ...process.env,
    AETHER_ROOT: tmpDir,
    DATA_DIR: path.join(tmpDir, '.aether', 'data')
  };
  const escapedArgs = args.map(a => `"${a}"`).join(' ');
  // Redirect stderr to stdout so we capture json_err output (which goes to stderr)
  const cmd = `bash "${scriptPath}" ${command} ${escapedArgs} 2>&1`;
  try {
    const output = execSync(cmd, { encoding: 'utf8', env, cwd: tmpDir });
    return { output, exitCode: 0 };
  } catch (err) {
    return { output: err.stdout || err.stderr || '', exitCode: err.status || 1 };
  }
}

async function setupTestEnv(tmpDir) {
  const dataDir = path.join(tmpDir, '.aether', 'data');
  await fs.promises.mkdir(dataDir, { recursive: true });

  // Create minimal COLONY_STATE.json (needed for check-replan learnings count)
  const colonyState = {
    goal: 'autopilot integration test',
    state: 'active',
    current_phase: 1,
    plan: { id: 'test-plan', phases: [] },
    memory: {
      decisions: [],
      instincts: [],
      phase_learnings: []
    },
    errors: { records: [] },
    events: [],
    session_id: 'test-session',
    initialized_at: new Date().toISOString()
  };
  await fs.promises.writeFile(
    path.join(dataDir, 'COLONY_STATE.json'),
    JSON.stringify(colonyState, null, 2)
  );

  return dataDir;
}

async function setupTestEnvWithLearnings(tmpDir, learningsCount) {
  const dataDir = await setupTestEnv(tmpDir);

  if (learningsCount > 0) {
    const phaseLearnings = [];
    for (let i = 1; i <= learningsCount; i++) {
      phaseLearnings.push({
        id: `learning_${i}`,
        phase: i,
        phase_name: `Phase ${i}`,
        learnings: [{ claim: `test claim ${i}`, status: 'validated' }]
      });
    }

    const stateFile = path.join(dataDir, 'COLONY_STATE.json');
    const state = JSON.parse(await fs.promises.readFile(stateFile, 'utf8'));
    state.memory.phase_learnings = phaseLearnings;
    await fs.promises.writeFile(stateFile, JSON.stringify(state, null, 2));
  }

  return dataDir;
}

function readRunState(tmpDir) {
  const stateFile = path.join(tmpDir, '.aether', 'data', 'run-state.json');
  return JSON.parse(fs.readFileSync(stateFile, 'utf8'));
}

// ---------------------------------------------------------------------------
// Test 1: autopilot-init creates valid run-state.json with correct schema
// ---------------------------------------------------------------------------

test.serial('1. autopilot-init creates valid run-state.json with correct schema', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestEnv(tmpDir);

    const output = runAetherUtil(tmpDir, 'autopilot-init', [
      '--total-phases', '6', '--start-phase', '1'
    ]);
    const parsed = JSON.parse(output);
    t.true(parsed.ok, 'init should return ok:true');

    // Verify run-state.json exists and has correct schema
    const state = readRunState(tmpDir);
    t.is(state.version, '1.0', 'version should be 1.0');
    t.is(state.status, 'running', 'status should be running');
    t.is(state.total_phases, 6, 'total_phases should be 6');
    t.is(state.start_phase, 1, 'start_phase should be 1');
    t.is(state.current_phase, 1, 'current_phase should equal start_phase');
    t.is(state.max_phases, null, 'max_phases should be null by default');
    t.is(state.pause_reason, null, 'pause_reason should be null');
    t.is(state.last_action, null, 'last_action should be null');
    t.is(state.phases_completed_in_run, 0, 'phases_completed_in_run should be 0');
    t.is(state.total_auto_advanced, 0, 'total_auto_advanced should be 0');
    t.true(Array.isArray(state.phase_results), 'phase_results should be an array');
    t.is(state.phase_results.length, 0, 'phase_results should be empty');
    t.regex(state.started_at, /^\d{4}-\d{2}-\d{2}T/, 'started_at should be ISO-8601');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

// ---------------------------------------------------------------------------
// Test 2: autopilot-update -> autopilot-status returns updated state
// ---------------------------------------------------------------------------

test.serial('2. autopilot-update then autopilot-status returns updated state', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestEnv(tmpDir);

    // Init
    runAetherUtil(tmpDir, 'autopilot-init', [
      '--total-phases', '6', '--start-phase', '1'
    ]);

    // Update with advance action
    runAetherUtil(tmpDir, 'autopilot-update', [
      '--action', 'advance', '--phase', '2', '--result', 'success'
    ]);

    // Status should reflect the update
    const statusOutput = runAetherUtil(tmpDir, 'autopilot-status');
    const parsed = JSON.parse(statusOutput);
    t.true(parsed.ok, 'status should return ok:true');
    t.is(parsed.result.status, 'running', 'status should still be running');
    t.is(parsed.result.current_phase, 2, 'current_phase should be 2');
    t.is(parsed.result.last_action, 'advance', 'last_action should be advance');
    t.is(parsed.result.total_auto_advanced, 1, 'total_auto_advanced should be 1');
    t.is(parsed.result.phase_results.length, 1, 'should have 1 phase result');
    t.is(parsed.result.phase_results[0].result, 'success', 'result should be success');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

// ---------------------------------------------------------------------------
// Test 3: autopilot-check-replan triggers at correct intervals
// ---------------------------------------------------------------------------

test.serial('3. autopilot-check-replan triggers at correct intervals in multi-phase run', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestEnvWithLearnings(tmpDir, 3);

    // Init with 8 phases
    runAetherUtil(tmpDir, 'autopilot-init', [
      '--total-phases', '8', '--start-phase', '1'
    ]);

    // 1 advance: should NOT trigger (under default interval of 2)
    runAetherUtil(tmpDir, 'autopilot-update', [
      '--action', 'advance', '--phase', '2'
    ]);
    let output = JSON.parse(runAetherUtil(tmpDir, 'autopilot-check-replan'));
    t.false(output.result.should_replan, 'should_replan should be false after 1 advance');

    // 2nd advance: SHOULD trigger (hits interval of 2)
    runAetherUtil(tmpDir, 'autopilot-update', [
      '--action', 'advance', '--phase', '3'
    ]);
    output = JSON.parse(runAetherUtil(tmpDir, 'autopilot-check-replan'));
    t.true(output.result.should_replan, 'should_replan should be true after 2 advances');
    t.truthy(output.result.reason, 'reason should be non-empty');

    // 3rd advance: should NOT trigger (3 is not a multiple of 2)
    runAetherUtil(tmpDir, 'autopilot-update', [
      '--action', 'advance', '--phase', '4'
    ]);
    output = JSON.parse(runAetherUtil(tmpDir, 'autopilot-check-replan'));
    t.false(output.result.should_replan, 'should_replan should be false after 3 advances');

    // 4th advance: SHOULD trigger (4 is a multiple of 2)
    runAetherUtil(tmpDir, 'autopilot-update', [
      '--action', 'advance', '--phase', '5'
    ]);
    output = JSON.parse(runAetherUtil(tmpDir, 'autopilot-check-replan'));
    t.true(output.result.should_replan, 'should_replan should be true after 4 advances');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

// ---------------------------------------------------------------------------
// Test 4: autopilot-stop sets status and preserves phase_results
// ---------------------------------------------------------------------------

test.serial('4. autopilot-stop sets status and preserves phase_results', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestEnv(tmpDir);

    // Init
    runAetherUtil(tmpDir, 'autopilot-init', [
      '--total-phases', '6', '--start-phase', '1'
    ]);

    // Add some phase results
    runAetherUtil(tmpDir, 'autopilot-update', [
      '--action', 'build', '--phase', '1', '--result', 'success'
    ]);
    runAetherUtil(tmpDir, 'autopilot-update', [
      '--action', 'advance', '--phase', '2', '--result', 'success'
    ]);
    runAetherUtil(tmpDir, 'autopilot-update', [
      '--action', 'build', '--phase', '2', '--result', 'failure'
    ]);

    // Stop
    const stopOutput = JSON.parse(runAetherUtil(tmpDir, 'autopilot-stop', [
      '--reason', 'build failure in phase 2'
    ]));
    t.true(stopOutput.ok, 'stop should return ok:true');

    // Verify state after stop
    const state = readRunState(tmpDir);
    t.is(state.status, 'stopped', 'status should be stopped');
    t.is(state.pause_reason, 'build failure in phase 2', 'pause_reason should match');
    t.is(state.phase_results.length, 3, 'all 3 phase_results should be preserved');
    t.is(state.phase_results[0].action, 'build', 'first result action should be build');
    t.is(state.phase_results[1].action, 'advance', 'second result action should be advance');
    t.is(state.phase_results[2].result, 'failure', 'third result should be failure');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

// ---------------------------------------------------------------------------
// Test 5: Full lifecycle: init -> update(advance) -> check-replan -> update(advance) -> stop
// ---------------------------------------------------------------------------

test.serial('5. Full lifecycle: init -> advance -> check-replan -> advance -> stop', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestEnvWithLearnings(tmpDir, 2);

    // Step 1: Init
    const initOutput = JSON.parse(runAetherUtil(tmpDir, 'autopilot-init', [
      '--total-phases', '4', '--start-phase', '1', '--max-phases', '3'
    ]));
    t.true(initOutput.ok, 'init should succeed');

    let state = readRunState(tmpDir);
    t.is(state.status, 'running', 'initial status should be running');
    t.is(state.max_phases, 3, 'max_phases should be 3');

    // Step 2: First advance (build phase 1, then advance to 2)
    runAetherUtil(tmpDir, 'autopilot-update', [
      '--action', 'build', '--phase', '1', '--result', 'success'
    ]);
    runAetherUtil(tmpDir, 'autopilot-update', [
      '--action', 'advance', '--phase', '2', '--result', 'success'
    ]);

    state = readRunState(tmpDir);
    t.is(state.current_phase, 2, 'current_phase should be 2 after advance');
    t.is(state.total_auto_advanced, 1, 'total_auto_advanced should be 1');

    // Step 3: Check replan (1 advance, under interval of 2 -> no replan)
    let replanOutput = JSON.parse(runAetherUtil(tmpDir, 'autopilot-check-replan'));
    t.false(replanOutput.result.should_replan, 'no replan after 1 advance');

    // Step 4: Second advance (build phase 2, then advance to 3)
    runAetherUtil(tmpDir, 'autopilot-update', [
      '--action', 'build', '--phase', '2', '--result', 'success'
    ]);
    runAetherUtil(tmpDir, 'autopilot-update', [
      '--action', 'advance', '--phase', '3', '--result', 'success'
    ]);

    state = readRunState(tmpDir);
    t.is(state.total_auto_advanced, 2, 'total_auto_advanced should be 2');

    // Step 5: Check replan (2 advances, at interval of 2 -> replan)
    replanOutput = JSON.parse(runAetherUtil(tmpDir, 'autopilot-check-replan'));
    t.true(replanOutput.result.should_replan, 'replan should trigger after 2 advances');
    t.is(replanOutput.result.learnings_since_last, 2, 'should report 2 learnings');

    // Step 6: Stop with completed status
    const stopOutput = JSON.parse(runAetherUtil(tmpDir, 'autopilot-stop', [
      '--reason', 'all target phases done', '--status', 'completed'
    ]));
    t.true(stopOutput.ok, 'stop should succeed');

    state = readRunState(tmpDir);
    t.is(state.status, 'completed', 'final status should be completed');
    t.is(state.pause_reason, 'all target phases done', 'pause_reason should match');
    t.is(state.phase_results.length, 4, 'should have 4 phase results total');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

// ---------------------------------------------------------------------------
// Test 6 (Bug Fix 1): autopilot-check-replan rejects --interval 0
// ---------------------------------------------------------------------------

test.serial('6. BUG FIX: autopilot-check-replan rejects --interval 0', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestEnv(tmpDir);

    // Init and do one advance
    runAetherUtil(tmpDir, 'autopilot-init', [
      '--total-phases', '6', '--start-phase', '1'
    ]);
    runAetherUtil(tmpDir, 'autopilot-update', [
      '--action', 'advance', '--phase', '2'
    ]);

    // Interval 0 should error, not divide-by-zero
    const { output, exitCode } = runAetherUtilSafe(tmpDir, 'autopilot-check-replan', [
      '--interval', '0'
    ]);
    t.not(exitCode, 0, 'exit code should be non-zero for interval=0');
    // Extract the JSON line from output (may contain warning text before it)
    const jsonLine = output.split('\n').find(line => line.startsWith('{'));
    t.truthy(jsonLine, 'should contain JSON error output');
    const parsed = JSON.parse(jsonLine);
    t.false(parsed.ok, 'ok should be false for interval=0');
    t.truthy(parsed.error, 'should have error object');
    t.truthy(parsed.error.message.includes('interval'), 'error message should mention interval');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

// ---------------------------------------------------------------------------
// Test 7 (Bug Fix 2): phases_completed_in_run only increments on advance
// ---------------------------------------------------------------------------

test.serial('7. BUG FIX: phases_completed_in_run only increments on action==advance', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestEnv(tmpDir);

    // Init
    runAetherUtil(tmpDir, 'autopilot-init', [
      '--total-phases', '6', '--start-phase', '1'
    ]);

    // Do a "build" action — should NOT increment phases_completed_in_run
    runAetherUtil(tmpDir, 'autopilot-update', [
      '--action', 'build', '--phase', '1', '--result', 'success'
    ]);
    let state = readRunState(tmpDir);
    t.is(state.phases_completed_in_run, 0,
      'phases_completed_in_run should be 0 after build (not advance)');

    // Do a "continue" action — should NOT increment either
    runAetherUtil(tmpDir, 'autopilot-update', [
      '--action', 'continue', '--phase', '1', '--result', 'success'
    ]);
    state = readRunState(tmpDir);
    t.is(state.phases_completed_in_run, 0,
      'phases_completed_in_run should still be 0 after continue (not advance)');

    // Do an "advance" action — SHOULD increment
    runAetherUtil(tmpDir, 'autopilot-update', [
      '--action', 'advance', '--phase', '2', '--result', 'success'
    ]);
    state = readRunState(tmpDir);
    t.is(state.phases_completed_in_run, 1,
      'phases_completed_in_run should be 1 after advance');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});
