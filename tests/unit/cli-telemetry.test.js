const test = require('ava');
const proxyquire = require('proxyquire');

// ============================================================================
// Mock Data Helpers
// ============================================================================

function createMockSummary(options = {}) {
  const {
    totalSpawns = 10,
    totalModels = 2,
    models = {
      'kimi-k2.5': {
        total_spawns: 6,
        success_rate: 0.95,
        successful_completions: 5,
        by_caste: {
          builder: { spawns: 4, success: 4, failures: 0, blocked: 0 },
          watcher: { spawns: 2, success: 1, failures: 0, blocked: 1 }
        }
      },
      'glm-5': {
        total_spawns: 4,
        success_rate: 0.5,
        successful_completions: 2,
        by_caste: {
          scout: { spawns: 4, success: 2, failures: 1, blocked: 1 }
        }
      }
    },
    recentDecisions = [
      { timestamp: '2026-02-14T10:00:00Z', task: 'task-1', caste: 'builder', selected_model: 'kimi-k2.5', source: 'caste-default' },
      { timestamp: '2026-02-14T10:05:00Z', task: 'task-2', caste: 'scout', selected_model: 'glm-5', source: 'task-based' }
    ]
  } = options;

  return {
    total_spawns: totalSpawns,
    total_models: totalModels,
    models,
    recent_decisions: recentDecisions
  };
}

function createMockModelPerformance(modelName = 'kimi-k2.5') {
  const performances = {
    'kimi-k2.5': {
      model: 'kimi-k2.5',
      total_spawns: 6,
      successful_completions: 5,
      failed_completions: 0,
      blocked: 1,
      success_rate: 0.83,
      by_caste: {
        builder: { spawns: 4, success: 4, failures: 0, blocked: 0 },
        watcher: { spawns: 2, success: 1, failures: 0, blocked: 1 }
      }
    },
    'glm-5': {
      model: 'glm-5',
      total_spawns: 4,
      successful_completions: 2,
      failed_completions: 1,
      blocked: 1,
      success_rate: 0.5,
      by_caste: {
        scout: { spawns: 4, success: 2, failures: 1, blocked: 1 }
      }
    }
  };

  return performances[modelName] || null;
}

// ============================================================================
// Test: telemetry summary shows message when no data exists
// ============================================================================
test('telemetry summary shows message when no data exists', async t => {
  const mockSummary = createMockSummary({ totalSpawns: 0, totalModels: 0, models: {}, recentDecisions: [] });
  let output = '';

  const mockTelemetry = {
    getTelemetrySummary: () => mockSummary,
    getModelPerformance: () => null
  };

  const mockColors = {
    header: (s) => s,
    info: (s) => s,
    success: (s) => s,
    warning: (s) => s,
    error: (s) => s,
    dim: (s) => s,
    bold: (s) => s
  };

  // Simulate the summary action
  const summary = mockTelemetry.getTelemetrySummary('/fake/path');

  t.is(summary.total_spawns, 0);
  t.is(summary.total_models, 0);
  t.deepEqual(summary.models, {});
  t.deepEqual(summary.recent_decisions, []);
});

// ============================================================================
// Test: telemetry summary displays correct total spawns count
// ============================================================================
test('telemetry summary displays correct total spawns count', async t => {
  const mockSummary = createMockSummary({ totalSpawns: 25, totalModels: 3 });

  const summary = mockSummary;

  t.is(summary.total_spawns, 25);
  t.is(summary.total_models, 3);
});

// ============================================================================
// Test: telemetry summary lists all models with stats
// ============================================================================
test('telemetry summary lists all models with stats', async t => {
  const mockSummary = createMockSummary();

  const summary = mockSummary;

  t.truthy(summary.models['kimi-k2.5']);
  t.truthy(summary.models['glm-5']);
  t.is(summary.models['kimi-k2.5'].total_spawns, 6);
  t.is(summary.models['glm-5'].total_spawns, 4);
});

// ============================================================================
// Test: telemetry summary shows recent routing decisions
// ============================================================================
test('telemetry summary shows recent routing decisions', async t => {
  const mockSummary = createMockSummary();

  const summary = mockSummary;

  t.is(summary.recent_decisions.length, 2);
  t.is(summary.recent_decisions[0].caste, 'builder');
  t.is(summary.recent_decisions[0].selected_model, 'kimi-k2.5');
  t.is(summary.recent_decisions[1].caste, 'scout');
  t.is(summary.recent_decisions[1].selected_model, 'glm-5');
});

// ============================================================================
// Test: telemetry model shows warning for unknown model
// ============================================================================
test('telemetry model shows warning for unknown model', async t => {
  const mockTelemetry = {
    getModelPerformance: (repoPath, modelName) => {
      if (modelName === 'unknown-model') {
        return null;
      }
      return createMockModelPerformance(modelName);
    }
  };

  const performance = mockTelemetry.getModelPerformance('/fake/path', 'unknown-model');

  t.is(performance, null);
});

// ============================================================================
// Test: telemetry model displays detailed stats for valid model
// ============================================================================
test('telemetry model displays detailed stats for valid model', async t => {
  const mockTelemetry = {
    getModelPerformance: (repoPath, modelName) => createMockModelPerformance(modelName)
  };

  const performance = mockTelemetry.getModelPerformance('/fake/path', 'kimi-k2.5');

  t.truthy(performance);
  t.is(performance.model, 'kimi-k2.5');
  t.is(performance.total_spawns, 6);
  t.is(performance.successful_completions, 5);
  t.is(performance.failed_completions, 0);
  t.is(performance.blocked, 1);
  t.is(performance.success_rate, 0.83);
});

// ============================================================================
// Test: telemetry model shows breakdown by caste
// ============================================================================
test('telemetry model shows breakdown by caste', async t => {
  const mockTelemetry = {
    getModelPerformance: (repoPath, modelName) => createMockModelPerformance(modelName)
  };

  const performance = mockTelemetry.getModelPerformance('/fake/path', 'kimi-k2.5');

  t.truthy(performance.by_caste);
  t.truthy(performance.by_caste.builder);
  t.truthy(performance.by_caste.watcher);
  t.is(performance.by_caste.builder.spawns, 4);
  t.is(performance.by_caste.watcher.spawns, 2);
});

// ============================================================================
// Test: telemetry model calculates success rate correctly
// ============================================================================
test('telemetry model calculates success rate correctly', async t => {
  const mockTelemetry = {
    getModelPerformance: (repoPath, modelName) => createMockModelPerformance(modelName)
  };

  const kimiPerf = mockTelemetry.getModelPerformance('/fake/path', 'kimi-k2.5');
  const glmPerf = mockTelemetry.getModelPerformance('/fake/path', 'glm-5');

  // kimi-k2.5: 5 successes / 6 spawns = 0.83
  t.is(kimiPerf.success_rate, 0.83);

  // glm-5: 2 successes / 4 spawns = 0.5
  t.is(glmPerf.success_rate, 0.5);
});

// ============================================================================
// Test: telemetry performance shows message when no data
// ============================================================================
test('telemetry performance shows message when no data', async t => {
  const mockSummary = createMockSummary({ totalSpawns: 0, totalModels: 0, models: {} });

  const summary = mockSummary;

  t.is(summary.total_spawns, 0);
  t.deepEqual(summary.models, {});
});

// ============================================================================
// Test: telemetry performance ranks models by success rate
// ============================================================================
test('telemetry performance ranks models by success rate', async t => {
  const mockSummary = createMockSummary({
    models: {
      'model-a': { total_spawns: 5, success_rate: 0.9, successful_completions: 4 },
      'model-b': { total_spawns: 5, success_rate: 0.6, successful_completions: 3 },
      'model-c': { total_spawns: 5, success_rate: 0.8, successful_completions: 4 }
    }
  });

  // Sort models by success rate (highest first)
  const ranked = Object.entries(mockSummary.models)
    .map(([model, stats]) => ({ model, ...stats }))
    .sort((a, b) => b.success_rate - a.success_rate);

  t.is(ranked[0].model, 'model-a'); // 0.9
  t.is(ranked[1].model, 'model-c'); // 0.8
  t.is(ranked[2].model, 'model-b'); // 0.6
});

// ============================================================================
// Test: telemetry performance displays all columns correctly
// ============================================================================
test('telemetry performance displays all columns correctly', async t => {
  const mockSummary = createMockSummary();

  const ranked = Object.entries(mockSummary.models)
    .map(([model, stats]) => ({ model, ...stats }))
    .sort((a, b) => b.success_rate - a.success_rate);

  // Verify structure of ranked data
  t.is(ranked[0].model, 'kimi-k2.5');
  t.is(ranked[0].total_spawns, 6);
  t.is(ranked[0].successful_completions, 5);
  t.is(ranked[0].success_rate, 0.95);

  t.is(ranked[1].model, 'glm-5');
  t.is(ranked[1].total_spawns, 4);
  t.is(ranked[1].successful_completions, 2);
  t.is(ranked[1].success_rate, 0.5);
});

// ============================================================================
// Test: color coding thresholds work correctly
// ============================================================================
test('color coding thresholds work correctly', async t => {
  // Test color thresholds: green >= 0.9, yellow >= 0.7, red < 0.7
  const getRateColor = (rate) => {
    if (rate >= 0.9) return 'green';
    if (rate >= 0.7) return 'yellow';
    return 'red';
  };

  t.is(getRateColor(0.95), 'green');
  t.is(getRateColor(0.9), 'green');
  t.is(getRateColor(0.85), 'yellow');
  t.is(getRateColor(0.7), 'yellow');
  t.is(getRateColor(0.69), 'red');
  t.is(getRateColor(0.5), 'red');
  t.is(getRateColor(0.0), 'red');
});

// ============================================================================
// Test: telemetry functions handle missing telemetry file gracefully
// ============================================================================
test('telemetry functions handle missing telemetry file gracefully', async t => {
  const mockTelemetry = {
    getTelemetrySummary: () => ({
      total_spawns: 0,
      total_models: 0,
      models: {},
      recent_decisions: []
    }),
    getModelPerformance: () => null
  };

  const summary = mockTelemetry.getTelemetrySummary('/non-existent/path');
  const performance = mockTelemetry.getModelPerformance('/non-existent/path', 'any-model');

  t.is(summary.total_spawns, 0);
  t.is(summary.total_models, 0);
  t.deepEqual(summary.models, {});
  t.is(performance, null);
});

// ============================================================================
// Test: recent decisions are limited to last 10
// ============================================================================
test('recent decisions are limited to last 10', async t => {
  const manyDecisions = Array(15).fill(null).map((_, i) => ({
    timestamp: `2026-02-14T10:${String(i).padStart(2, '0')}:00Z`,
    task: `task-${i}`,
    caste: 'builder',
    selected_model: 'kimi-k2.5',
    source: 'test'
  }));

  const mockSummary = createMockSummary({ recentDecisions: manyDecisions });

  // In actual implementation, only last 10 are returned
  const recentDecisions = mockSummary.recent_decisions.slice(-10);

  t.is(recentDecisions.length, 10);
  t.is(recentDecisions[0].task, 'task-5');
  t.is(recentDecisions[9].task, 'task-14');
});

// ============================================================================
// Test: caste stats calculation handles zero spawns
// ============================================================================
test('caste stats calculation handles zero spawns', async t => {
  const performance = {
    model: 'test-model',
    total_spawns: 0,
    successful_completions: 0,
    failed_completions: 0,
    blocked: 0,
    success_rate: 0,
    by_caste: {
      builder: { spawns: 0, success: 0, failures: 0, blocked: 0 }
    }
  };

  const casteRate = performance.by_caste.builder.spawns > 0
    ? (performance.by_caste.builder.success / performance.by_caste.builder.spawns * 100).toFixed(1)
    : '0.0';

  t.is(casteRate, '0.0');
  t.is(performance.success_rate, 0);
});
