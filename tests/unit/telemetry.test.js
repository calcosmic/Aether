const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');

// Import the module under test
const {
  recordSpawnTelemetry,
  updateSpawnOutcome,
  getTelemetrySummary,
  getModelPerformance,
  getRoutingStats,
  loadTelemetry,
  saveTelemetry,
  TELEMETRY_VERSION,
  MAX_ROUTING_DECISIONS
} = require('../../bin/lib/telemetry');

/**
 * Helper to create a temporary directory for test isolation
 */
function createTempDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), 'telemetry-test-'));
}

/**
 * Helper to cleanup temp directory
 */
function cleanupTempDir(tempDir) {
  try {
    fs.rmSync(tempDir, { recursive: true, force: true });
  } catch (error) {
    // Ignore cleanup errors
  }
}

// ============================================================================
// Test: loadTelemetry creates default structure if file doesn't exist
// ============================================================================
test('loadTelemetry creates default structure if file does not exist', t => {
  const tempDir = createTempDir();

  const data = loadTelemetry(tempDir);

  t.is(data.version, TELEMETRY_VERSION);
  t.truthy(data.last_updated);
  t.deepEqual(data.models, {});
  t.deepEqual(data.routing_decisions, []);

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: loadTelemetry handles corrupted telemetry.json gracefully
// ============================================================================
test('loadTelemetry handles corrupted telemetry.json gracefully', t => {
  const tempDir = createTempDir();
  const dataDir = path.join(tempDir, '.aether', 'data');
  fs.mkdirSync(dataDir, { recursive: true });

  // Write invalid JSON
  fs.writeFileSync(path.join(dataDir, 'telemetry.json'), 'not valid json', 'utf8');

  const data = loadTelemetry(tempDir);

  t.is(data.version, TELEMETRY_VERSION);
  t.deepEqual(data.models, {});
  t.deepEqual(data.routing_decisions, []);

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: loadTelemetry handles missing required fields
// ============================================================================
test('loadTelemetry handles missing required fields', t => {
  const tempDir = createTempDir();
  const dataDir = path.join(tempDir, '.aether', 'data');
  fs.mkdirSync(dataDir, { recursive: true });

  // Write JSON with missing fields
  fs.writeFileSync(
    path.join(dataDir, 'telemetry.json'),
    JSON.stringify({ version: '1.0' }),
    'utf8'
  );

  const data = loadTelemetry(tempDir);

  t.is(data.version, TELEMETRY_VERSION);
  t.deepEqual(data.models, {});
  t.deepEqual(data.routing_decisions, []);

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: recordSpawnTelemetry creates telemetry.json if it doesn't exist
// ============================================================================
test('recordSpawnTelemetry creates telemetry.json if it does not exist', t => {
  const tempDir = createTempDir();

  const result = recordSpawnTelemetry(tempDir, {
    task: 'test-task',
    caste: 'builder',
    model: 'kimi-k2.5',
    source: 'test'
  });

  t.true(result.success);
  t.truthy(result.decision_id);

  // Verify file was created
  const telemetryPath = path.join(tempDir, '.aether', 'data', 'telemetry.json');
  t.true(fs.existsSync(telemetryPath));

  const data = JSON.parse(fs.readFileSync(telemetryPath, 'utf8'));
  t.is(data.version, TELEMETRY_VERSION);
  t.is(data.routing_decisions.length, 1);
  t.is(data.models['kimi-k2.5'].total_spawns, 1);

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: recordSpawnTelemetry increments total_spawns for the model
// ============================================================================
test('recordSpawnTelemetry increments total_spawns for the model', t => {
  const tempDir = createTempDir();

  // Record multiple spawns for same model
  recordSpawnTelemetry(tempDir, {
    task: 'task-1',
    caste: 'builder',
    model: 'kimi-k2.5',
    source: 'test'
  });

  recordSpawnTelemetry(tempDir, {
    task: 'task-2',
    caste: 'watcher',
    model: 'kimi-k2.5',
    source: 'test'
  });

  const data = loadTelemetry(tempDir);
  t.is(data.models['kimi-k2.5'].total_spawns, 2);

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: recordSpawnTelemetry creates by_caste entry if new caste
// ============================================================================
test('recordSpawnTelemetry creates by_caste entry if new caste', t => {
  const tempDir = createTempDir();

  recordSpawnTelemetry(tempDir, {
    task: 'task-1',
    caste: 'builder',
    model: 'kimi-k2.5',
    source: 'test'
  });

  recordSpawnTelemetry(tempDir, {
    task: 'task-2',
    caste: 'watcher',
    model: 'kimi-k2.5',
    source: 'test'
  });

  const data = loadTelemetry(tempDir);
  const modelStats = data.models['kimi-k2.5'];

  t.truthy(modelStats.by_caste.builder);
  t.truthy(modelStats.by_caste.watcher);
  t.is(modelStats.by_caste.builder.spawns, 1);
  t.is(modelStats.by_caste.watcher.spawns, 1);

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: recordSpawnTelemetry appends to routing_decisions
// ============================================================================
test('recordSpawnTelemetry appends to routing_decisions', t => {
  const tempDir = createTempDir();

  recordSpawnTelemetry(tempDir, {
    task: 'task-1',
    caste: 'builder',
    model: 'kimi-k2.5',
    source: 'caste-default'
  });

  recordSpawnTelemetry(tempDir, {
    task: 'task-2',
    caste: 'watcher',
    model: 'glm-5',
    source: 'task-based'
  });

  const data = loadTelemetry(tempDir);
  t.is(data.routing_decisions.length, 2);
  t.is(data.routing_decisions[0].task, 'task-1');
  t.is(data.routing_decisions[1].task, 'task-2');
  t.is(data.routing_decisions[0].source, 'caste-default');
  t.is(data.routing_decisions[1].source, 'task-based');

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: recordSpawnTelemetry rotates routing_decisions at 1000 entries
// ============================================================================
test('recordSpawnTelemetry rotates routing_decisions at 1000 entries', t => {
  const tempDir = createTempDir();

  // Create initial data with 999 decisions
  const data = loadTelemetry(tempDir);
  data.routing_decisions = Array(999).fill(null).map((_, i) => ({
    timestamp: `2026-01-${String(i % 30 + 1).padStart(2, '0')}T00:00:00Z`,
    task: `old-task-${i}`,
    caste: 'builder',
    selected_model: 'old-model',
    source: 'test'
  }));
  saveTelemetry(tempDir, data);

  // Add one more - should not rotate yet
  recordSpawnTelemetry(tempDir, {
    task: 'task-1000',
    caste: 'builder',
    model: 'kimi-k2.5',
    source: 'test'
  });

  let updatedData = loadTelemetry(tempDir);
  t.is(updatedData.routing_decisions.length, 1000);

  // Add one more - should rotate to keep last 1000
  recordSpawnTelemetry(tempDir, {
    task: 'task-1001',
    caste: 'builder',
    model: 'kimi-k2.5',
    source: 'test'
  });

  updatedData = loadTelemetry(tempDir);
  t.is(updatedData.routing_decisions.length, 1000);

  // First entry should now be old-task-1 (old-task-0 was rotated out)
  t.is(updatedData.routing_decisions[0].task, 'old-task-1');
  // Last entry should be task-1001
  t.is(updatedData.routing_decisions[999].task, 'task-1001');

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: recordSpawnTelemetry uses atomic writes
// ============================================================================
test('recordSpawnTelemetry uses atomic writes', t => {
  const tempDir = createTempDir();

  recordSpawnTelemetry(tempDir, {
    task: 'atomic-test',
    caste: 'builder',
    model: 'kimi-k2.5',
    source: 'test'
  });

  // Check that temp file doesn't exist (was renamed)
  const tempPath = path.join(tempDir, '.aether', 'data', 'telemetry.json.tmp');
  t.false(fs.existsSync(tempPath));

  // Check that actual file exists
  const telemetryPath = path.join(tempDir, '.aether', 'data', 'telemetry.json');
  t.true(fs.existsSync(telemetryPath));

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: updateSpawnOutcome updates successful_completions counter
// ============================================================================
test('updateSpawnOutcome updates successful_completions counter', t => {
  const tempDir = createTempDir();

  const result = recordSpawnTelemetry(tempDir, {
    task: 'test-task',
    caste: 'builder',
    model: 'kimi-k2.5',
    source: 'test'
  });

  const updateResult = updateSpawnOutcome(tempDir, result.decision_id, 'completed');
  t.true(updateResult);

  const data = loadTelemetry(tempDir);
  t.is(data.models['kimi-k2.5'].successful_completions, 1);
  t.is(data.models['kimi-k2.5'].failed_completions, 0);
  t.is(data.models['kimi-k2.5'].blocked, 0);

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: updateSpawnOutcome updates failed_completions counter
// ============================================================================
test('updateSpawnOutcome updates failed_completions counter', t => {
  const tempDir = createTempDir();

  const result = recordSpawnTelemetry(tempDir, {
    task: 'test-task',
    caste: 'builder',
    model: 'kimi-k2.5',
    source: 'test'
  });

  updateSpawnOutcome(tempDir, result.decision_id, 'failed');

  const data = loadTelemetry(tempDir);
  t.is(data.models['kimi-k2.5'].successful_completions, 0);
  t.is(data.models['kimi-k2.5'].failed_completions, 1);
  t.is(data.models['kimi-k2.5'].blocked, 0);

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: updateSpawnOutcome updates blocked counter
// ============================================================================
test('updateSpawnOutcome updates blocked counter', t => {
  const tempDir = createTempDir();

  const result = recordSpawnTelemetry(tempDir, {
    task: 'test-task',
    caste: 'builder',
    model: 'kimi-k2.5',
    source: 'test'
  });

  updateSpawnOutcome(tempDir, result.decision_id, 'blocked');

  const data = loadTelemetry(tempDir);
  t.is(data.models['kimi-k2.5'].successful_completions, 0);
  t.is(data.models['kimi-k2.5'].failed_completions, 0);
  t.is(data.models['kimi-k2.5'].blocked, 1);

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: updateSpawnOutcome updates by_caste counters correctly
// ============================================================================
test('updateSpawnOutcome updates by_caste counters correctly', t => {
  const tempDir = createTempDir();

  const result = recordSpawnTelemetry(tempDir, {
    task: 'test-task',
    caste: 'builder',
    model: 'kimi-k2.5',
    source: 'test'
  });

  updateSpawnOutcome(tempDir, result.decision_id, 'completed');

  const data = loadTelemetry(tempDir);
  const casteStats = data.models['kimi-k2.5'].by_caste.builder;

  t.is(casteStats.spawns, 1);
  t.is(casteStats.success, 1);
  t.is(casteStats.failures, 0);
  t.is(casteStats.blocked, 0);

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: updateSpawnOutcome returns false for non-existent spawn
// ============================================================================
test('updateSpawnOutcome returns false for non-existent spawn', t => {
  const tempDir = createTempDir();

  const result = updateSpawnOutcome(tempDir, 'non-existent-id', 'completed');
  t.false(result);

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: updateSpawnOutcome returns false for non-existent model
// ============================================================================
test('updateSpawnOutcome returns false for non-existent model', t => {
  const tempDir = createTempDir();

  // Manually create a decision with a model that doesn't exist in models
  const data = loadTelemetry(tempDir);
  data.routing_decisions.push({
    timestamp: 'orphan-timestamp',
    task: 'orphan-task',
    caste: 'builder',
    selected_model: 'non-existent-model',
    source: 'test'
  });
  saveTelemetry(tempDir, data);

  const result = updateSpawnOutcome(tempDir, 'orphan-timestamp', 'completed');
  t.false(result);

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: updateSpawnOutcome returns false for invalid outcome
// ============================================================================
test('updateSpawnOutcome returns false for invalid outcome', t => {
  const tempDir = createTempDir();

  const result = recordSpawnTelemetry(tempDir, {
    task: 'test-task',
    caste: 'builder',
    model: 'kimi-k2.5',
    source: 'test'
  });

  const updateResult = updateSpawnOutcome(tempDir, result.decision_id, 'invalid-outcome');
  t.false(updateResult);

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: getTelemetrySummary returns correct structure
// ============================================================================
test('getTelemetrySummary returns correct structure', t => {
  const tempDir = createTempDir();

  // Record some spawns
  recordSpawnTelemetry(tempDir, {
    task: 'task-1',
    caste: 'builder',
    model: 'kimi-k2.5',
    source: 'test'
  });

  recordSpawnTelemetry(tempDir, {
    task: 'task-2',
    caste: 'watcher',
    model: 'glm-5',
    source: 'test'
  });

  const summary = getTelemetrySummary(tempDir);

  t.is(summary.total_spawns, 2);
  t.is(summary.total_models, 2);
  t.truthy(summary.models['kimi-k2.5']);
  t.truthy(summary.models['glm-5']);
  t.is(summary.recent_decisions.length, 2);

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: getTelemetrySummary calculates success_rate correctly
// ============================================================================
test('getTelemetrySummary calculates success_rate correctly', t => {
  const tempDir = createTempDir();

  // Record spawns and outcomes
  const result1 = recordSpawnTelemetry(tempDir, {
    task: 'task-1',
    caste: 'builder',
    model: 'kimi-k2.5',
    source: 'test'
  });
  updateSpawnOutcome(tempDir, result1.decision_id, 'completed');

  const result2 = recordSpawnTelemetry(tempDir, {
    task: 'task-2',
    caste: 'builder',
    model: 'kimi-k2.5',
    source: 'test'
  });
  updateSpawnOutcome(tempDir, result2.decision_id, 'failed');

  recordSpawnTelemetry(tempDir, {
    task: 'task-3',
    caste: 'builder',
    model: 'kimi-k2.5',
    source: 'test'
  });

  const summary = getTelemetrySummary(tempDir);
  const modelStats = summary.models['kimi-k2.5'];

  t.is(modelStats.total_spawns, 3);
  t.is(modelStats.success_rate, 0.33); // 1 success / 3 spawns, rounded to 2 decimals

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: getTelemetrySummary returns last 10 routing decisions
// ============================================================================
test('getTelemetrySummary returns last 10 routing decisions', t => {
  const tempDir = createTempDir();

  // Record 15 spawns
  for (let i = 0; i < 15; i++) {
    recordSpawnTelemetry(tempDir, {
      task: `task-${i}`,
      caste: 'builder',
      model: 'kimi-k2.5',
      source: 'test'
    });
  }

  const summary = getTelemetrySummary(tempDir);

  t.is(summary.recent_decisions.length, 10);
  // Should be the last 10 (task-5 through task-14)
  t.is(summary.recent_decisions[0].task, 'task-5');
  t.is(summary.recent_decisions[9].task, 'task-14');

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: getModelPerformance returns correct stats for a model
// ============================================================================
test('getModelPerformance returns correct stats for a model', t => {
  const tempDir = createTempDir();

  // Record spawns and outcomes
  const result1 = recordSpawnTelemetry(tempDir, {
    task: 'task-1',
    caste: 'builder',
    model: 'kimi-k2.5',
    source: 'test'
  });
  updateSpawnOutcome(tempDir, result1.decision_id, 'completed');

  const result2 = recordSpawnTelemetry(tempDir, {
    task: 'task-2',
    caste: 'watcher',
    model: 'kimi-k2.5',
    source: 'test'
  });
  updateSpawnOutcome(tempDir, result2.decision_id, 'failed');

  const result3 = recordSpawnTelemetry(tempDir, {
    task: 'task-3',
    caste: 'builder',
    model: 'kimi-k2.5',
    source: 'test'
  });
  updateSpawnOutcome(tempDir, result3.decision_id, 'blocked');

  const perf = getModelPerformance(tempDir, 'kimi-k2.5');

  t.truthy(perf);
  t.is(perf.model, 'kimi-k2.5');
  t.is(perf.total_spawns, 3);
  t.is(perf.successful_completions, 1);
  t.is(perf.failed_completions, 1);
  t.is(perf.blocked, 1);
  t.is(perf.success_rate, 0.33);
  t.truthy(perf.by_caste.builder);
  t.truthy(perf.by_caste.watcher);

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: getModelPerformance returns null for non-existent model
// ============================================================================
test('getModelPerformance returns null for non-existent model', t => {
  const tempDir = createTempDir();

  const perf = getModelPerformance(tempDir, 'non-existent-model');
  t.is(perf, null);

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: getRoutingStats returns all stats when no filters
// ============================================================================
test('getRoutingStats returns all stats when no filters', t => {
  const tempDir = createTempDir();

  recordSpawnTelemetry(tempDir, {
    task: 'task-1',
    caste: 'builder',
    model: 'kimi-k2.5',
    source: 'caste-default'
  });

  recordSpawnTelemetry(tempDir, {
    task: 'task-2',
    caste: 'watcher',
    model: 'glm-5',
    source: 'task-based'
  });

  const stats = getRoutingStats(tempDir);

  t.is(stats.total_decisions, 2);
  t.is(stats.by_source['caste-default'], 1);
  t.is(stats.by_source['task-based'], 1);
  t.is(stats.by_model['kimi-k2.5'], 1);
  t.is(stats.by_model['glm-5'], 1);
  t.truthy(stats.date_range);

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: getRoutingStats filters by caste correctly
// ============================================================================
test('getRoutingStats filters by caste correctly', t => {
  const tempDir = createTempDir();

  recordSpawnTelemetry(tempDir, {
    task: 'task-1',
    caste: 'builder',
    model: 'kimi-k2.5',
    source: 'test'
  });

  recordSpawnTelemetry(tempDir, {
    task: 'task-2',
    caste: 'watcher',
    model: 'glm-5',
    source: 'test'
  });

  recordSpawnTelemetry(tempDir, {
    task: 'task-3',
    caste: 'builder',
    model: 'kimi-k2.5',
    source: 'test'
  });

  const stats = getRoutingStats(tempDir, { caste: 'builder' });

  t.is(stats.total_decisions, 2);
  t.is(stats.by_model['kimi-k2.5'], 2);

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: getRoutingStats filters by days correctly
// ============================================================================
test('getRoutingStats filters by days correctly', t => {
  const tempDir = createTempDir();

  // Create data with old and new decisions
  const data = loadTelemetry(tempDir);

  // Old decision (10 days ago)
  const oldDate = new Date();
  oldDate.setDate(oldDate.getDate() - 10);
  data.routing_decisions.push({
    timestamp: oldDate.toISOString(),
    task: 'old-task',
    caste: 'builder',
    selected_model: 'old-model',
    source: 'test'
  });

  // Recent decision (1 day ago)
  const recentDate = new Date();
  recentDate.setDate(recentDate.getDate() - 1);
  data.routing_decisions.push({
    timestamp: recentDate.toISOString(),
    task: 'recent-task',
    caste: 'builder',
    selected_model: 'recent-model',
    source: 'test'
  });

  saveTelemetry(tempDir, data);

  // Filter to last 5 days
  const stats = getRoutingStats(tempDir, { days: 5 });

  t.is(stats.total_decisions, 1);
  t.is(stats.by_model['recent-model'], 1);

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: getRoutingStats returns empty stats when no decisions match
// ============================================================================
test('getRoutingStats returns empty stats when no decisions match', t => {
  const tempDir = createTempDir();

  recordSpawnTelemetry(tempDir, {
    task: 'task-1',
    caste: 'builder',
    model: 'kimi-k2.5',
    source: 'test'
  });

  const stats = getRoutingStats(tempDir, { caste: 'non-existent-caste' });

  t.is(stats.total_decisions, 0);
  t.deepEqual(stats.by_source, {});
  t.deepEqual(stats.by_model, {});
  t.is(stats.date_range, null);

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: saveTelemetry creates directory if it doesn't exist
// ============================================================================
test('saveTelemetry creates directory if it does not exist', t => {
  const tempDir = createTempDir();
  const data = createDefaultTelemetry();

  const result = saveTelemetry(tempDir, data);
  t.true(result);

  // Verify directory was created
  const dataDir = path.join(tempDir, '.aether', 'data');
  t.true(fs.existsSync(dataDir));

  // Verify file was created
  const telemetryPath = path.join(dataDir, 'telemetry.json');
  t.true(fs.existsSync(telemetryPath));

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: saveTelemetry returns false on error
// ============================================================================
test('saveTelemetry returns false on error', t => {
  // Use a path that can't be written to
  const invalidPath = '/non-existent-path-xyz';
  const data = createDefaultTelemetry();

  const result = saveTelemetry(invalidPath, data);
  t.false(result);
});

// ============================================================================
// Test: recordSpawnTelemetry handles missing parameters gracefully
// ============================================================================
test('recordSpawnTelemetry handles missing parameters gracefully', t => {
  const tempDir = createTempDir();

  const result = recordSpawnTelemetry(tempDir, {
    task: null,
    caste: null,
    model: null,
    source: null
  });

  t.true(result.success);

  const data = loadTelemetry(tempDir);
  t.is(data.routing_decisions[0].task, 'unknown');
  t.is(data.routing_decisions[0].caste, 'unknown');
  t.is(data.routing_decisions[0].selected_model, 'default'); // null defaults to 'default'
  t.is(data.routing_decisions[0].source, 'unknown');

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: recordSpawnTelemetry uses provided timestamp
// ============================================================================
test('recordSpawnTelemetry uses provided timestamp', t => {
  const tempDir = createTempDir();
  const customTimestamp = '2026-01-15T12:00:00Z';

  const result = recordSpawnTelemetry(tempDir, {
    task: 'test-task',
    caste: 'builder',
    model: 'kimi-k2.5',
    source: 'test',
    timestamp: customTimestamp
  });

  t.is(result.decision_id, customTimestamp);

  const data = loadTelemetry(tempDir);
  t.is(data.routing_decisions[0].timestamp, customTimestamp);

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: Multiple models tracked independently
// ============================================================================
test('Multiple models tracked independently', t => {
  const tempDir = createTempDir();

  // Record spawns for different models
  recordSpawnTelemetry(tempDir, {
    task: 'task-1',
    caste: 'builder',
    model: 'kimi-k2.5',
    source: 'test'
  });

  recordSpawnTelemetry(tempDir, {
    task: 'task-2',
    caste: 'builder',
    model: 'kimi-k2.5',
    source: 'test'
  });

  recordSpawnTelemetry(tempDir, {
    task: 'task-3',
    caste: 'watcher',
    model: 'glm-5',
    source: 'test'
  });

  const data = loadTelemetry(tempDir);

  t.is(data.models['kimi-k2.5'].total_spawns, 2);
  t.is(data.models['glm-5'].total_spawns, 1);

  cleanupTempDir(tempDir);
});

// ============================================================================
// Test: getTelemetrySummary handles empty telemetry
// ============================================================================
test('getTelemetrySummary handles empty telemetry', t => {
  const tempDir = createTempDir();

  const summary = getTelemetrySummary(tempDir);

  t.is(summary.total_spawns, 0);
  t.is(summary.total_models, 0);
  t.deepEqual(summary.models, {});
  t.deepEqual(summary.recent_decisions, []);

  cleanupTempDir(tempDir);
});

// Helper function (not exported from module)
function createDefaultTelemetry() {
  return {
    version: TELEMETRY_VERSION,
    last_updated: new Date().toISOString(),
    models: {},
    routing_decisions: []
  };
}
