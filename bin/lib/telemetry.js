#!/usr/bin/env node
/**
 * Telemetry Module
 *
 * Tracks model performance and routing decisions for data-driven model selection.
 * Records every spawn with model, caste, task, and routing source.
 * Tracks success/failure rates per model-caste combination.
 * Rotates at 1000 routing decisions to prevent unbounded growth.
 * Uses atomic writes (temp file + rename) for data integrity.
 */

const fs = require('fs');
const path = require('path');

const TELEMETRY_VERSION = '1.0';
const MAX_ROUTING_DECISIONS = 1000;
const DEFAULT_TELEMETRY_FILE = 'telemetry.json';

/**
 * Get the telemetry file path
 * @param {string} repoPath - Repository root path
 * @returns {string} Full path to telemetry.json
 */
function getTelemetryPath(repoPath) {
  return path.join(repoPath, '.aether', 'data', DEFAULT_TELEMETRY_FILE);
}

/**
 * Load telemetry data from file
 * @param {string} repoPath - Repository root path
 * @returns {Object} Parsed telemetry data or default structure
 */
function loadTelemetry(repoPath) {
  const telemetryPath = getTelemetryPath(repoPath);

  try {
    if (!fs.existsSync(telemetryPath)) {
      return createDefaultTelemetry();
    }

    const content = fs.readFileSync(telemetryPath, 'utf8');
    const data = JSON.parse(content);

    // Validate structure
    if (!data.version || !data.models || !Array.isArray(data.routing_decisions)) {
      return createDefaultTelemetry();
    }

    return data;
  } catch (error) {
    // Return default on any error (corrupted file, permission issues, etc.)
    return createDefaultTelemetry();
  }
}

/**
 * Create default telemetry structure
 * @returns {Object} Default telemetry object
 */
function createDefaultTelemetry() {
  return {
    version: TELEMETRY_VERSION,
    last_updated: new Date().toISOString(),
    models: {},
    routing_decisions: []
  };
}

/**
 * Save telemetry data atomically
 * @param {string} repoPath - Repository root path
 * @param {Object} data - Telemetry data to save
 * @returns {boolean} True if saved successfully
 */
function saveTelemetry(repoPath, data) {
  const telemetryPath = getTelemetryPath(repoPath);

  try {
    // Ensure directory exists
    const dataDir = path.dirname(telemetryPath);
    if (!fs.existsSync(dataDir)) {
      fs.mkdirSync(dataDir, { recursive: true });
    }

    // Update timestamp
    data.last_updated = new Date().toISOString();

    // Atomic write: write to temp file, then rename
    const tempPath = `${telemetryPath}.tmp`;
    fs.writeFileSync(tempPath, JSON.stringify(data, null, 2), 'utf8');
    fs.renameSync(tempPath, telemetryPath);

    return true;
  } catch (error) {
    // Silent fail - don't cascade errors from telemetry
    return false;
  }
}

/**
 * Record spawn telemetry
 * @param {string} repoPath - Repository root path
 * @param {Object} spawnInfo - Spawn details
 * @param {string} spawnInfo.task - Task description
 * @param {string} spawnInfo.caste - Worker caste (e.g., "builder")
 * @param {string} spawnInfo.model - Model used (e.g., "kimi-k2.5")
 * @param {string} spawnInfo.source - Routing source (e.g., "caste-default", "task-based")
 * @param {string} [spawnInfo.timestamp] - Optional timestamp (defaults to now)
 * @returns {Object} Result with success flag and decision_id
 */
function recordSpawnTelemetry(repoPath, { task, caste, model, source, timestamp }) {
  try {
    const data = loadTelemetry(repoPath);
    const decisionTimestamp = timestamp || new Date().toISOString();

    // Initialize model stats if not exists
    if (!data.models[model]) {
      data.models[model] = {
        total_spawns: 0,
        successful_completions: 0,
        failed_completions: 0,
        blocked: 0,
        by_caste: {}
      };
    }

    const modelStats = data.models[model];

    // Increment total spawns
    modelStats.total_spawns++;

    // Initialize caste stats if not exists
    if (!modelStats.by_caste[caste]) {
      modelStats.by_caste[caste] = {
        spawns: 0,
        success: 0,
        failures: 0,
        blocked: 0
      };
    }

    // Increment caste spawns
    modelStats.by_caste[caste].spawns++;

    // Create routing decision record
    const decision = {
      timestamp: decisionTimestamp,
      task: task || 'unknown',
      caste: caste || 'unknown',
      selected_model: model || 'default',
      source: source || 'unknown'
    };

    // Append to routing decisions
    data.routing_decisions.push(decision);

    // Rotate if exceeds max
    if (data.routing_decisions.length > MAX_ROUTING_DECISIONS) {
      data.routing_decisions = data.routing_decisions.slice(-MAX_ROUTING_DECISIONS);
    }

    // Save atomically
    const saved = saveTelemetry(repoPath, data);

    return {
      success: saved,
      decision_id: decisionTimestamp
    };
  } catch (error) {
    return {
      success: false,
      decision_id: null,
      error: error.message
    };
  }
}

/**
 * Update spawn outcome
 * @param {string} repoPath - Repository root path
 * @param {string} spawnId - Spawn identifier (timestamp from recordSpawnTelemetry)
 * @param {string} outcome - Outcome: 'completed' | 'failed' | 'blocked'
 * @returns {boolean} True if updated successfully
 */
function updateSpawnOutcome(repoPath, spawnId, outcome) {
  try {
    const data = loadTelemetry(repoPath);

    // Find the routing decision by timestamp
    const decision = data.routing_decisions.find(d => d.timestamp === spawnId);

    if (!decision) {
      return false;
    }

    const { selected_model: model, caste } = decision;

    // Ensure model exists
    if (!data.models[model]) {
      return false;
    }

    const modelStats = data.models[model];

    // Update model-level counters
    switch (outcome) {
      case 'completed':
        modelStats.successful_completions++;
        break;
      case 'failed':
        modelStats.failed_completions++;
        break;
      case 'blocked':
        modelStats.blocked++;
        break;
      default:
        return false;
    }

    // Update caste-level counters
    if (modelStats.by_caste[caste]) {
      switch (outcome) {
        case 'completed':
          modelStats.by_caste[caste].success++;
          break;
        case 'failed':
          modelStats.by_caste[caste].failures++;
          break;
        case 'blocked':
          modelStats.by_caste[caste].blocked++;
          break;
      }
    }

    // Save atomically
    return saveTelemetry(repoPath, data);
  } catch (error) {
    return false;
  }
}

/**
 * Get telemetry summary
 * @param {string} repoPath - Repository root path
 * @returns {Object} Summary of telemetry data
 */
function getTelemetrySummary(repoPath) {
  const data = loadTelemetry(repoPath);

  const totalSpawns = Object.values(data.models).reduce(
    (sum, model) => sum + (model.total_spawns || 0),
    0
  );

  const models = {};
  for (const [modelName, stats] of Object.entries(data.models)) {
    const successRate = stats.total_spawns > 0
      ? (stats.successful_completions / stats.total_spawns)
      : 0;

    models[modelName] = {
      total_spawns: stats.total_spawns,
      success_rate: Math.round(successRate * 100) / 100,
      by_caste: stats.by_caste || {}
    };
  }

  // Get last 10 routing decisions
  const recentDecisions = data.routing_decisions.slice(-10);

  return {
    total_spawns: totalSpawns,
    total_models: Object.keys(data.models).length,
    models,
    recent_decisions: recentDecisions
  };
}

/**
 * Get detailed performance for a specific model
 * @param {string} repoPath - Repository root path
 * @param {string} model - Model name
 * @returns {Object|null} Model performance data or null if not found
 */
function getModelPerformance(repoPath, model) {
  const data = loadTelemetry(repoPath);

  if (!data.models[model]) {
    return null;
  }

  const stats = data.models[model];
  const successRate = stats.total_spawns > 0
    ? (stats.successful_completions / stats.total_spawns)
    : 0;

  return {
    model,
    total_spawns: stats.total_spawns,
    successful_completions: stats.successful_completions || 0,
    failed_completions: stats.failed_completions || 0,
    blocked: stats.blocked || 0,
    success_rate: Math.round(successRate * 100) / 100,
    by_caste: stats.by_caste || {}
  };
}

/**
 * Get routing statistics with optional filtering
 * @param {string} repoPath - Repository root path
 * @param {Object} options - Filter options
 * @param {number} [options.days] - Filter to last N days
 * @param {string} [options.caste] - Filter by caste
 * @returns {Object} Routing statistics
 */
function getRoutingStats(repoPath, options = {}) {
  const data = loadTelemetry(repoPath);
  const { days, caste } = options;

  let decisions = [...data.routing_decisions];

  // Filter by days
  if (days && days > 0) {
    const cutoffDate = new Date();
    cutoffDate.setDate(cutoffDate.getDate() - days);

    decisions = decisions.filter(d => new Date(d.timestamp) >= cutoffDate);
  }

  // Filter by caste
  if (caste) {
    decisions = decisions.filter(d => d.caste === caste);
  }

  // Calculate stats
  const bySource = {};
  const byModel = {};

  for (const decision of decisions) {
    // Count by source
    bySource[decision.source] = (bySource[decision.source] || 0) + 1;

    // Count by model
    byModel[decision.selected_model] = (byModel[decision.selected_model] || 0) + 1;
  }

  return {
    total_decisions: decisions.length,
    by_source: bySource,
    by_model: byModel,
    date_range: decisions.length > 0 ? {
      earliest: decisions[0].timestamp,
      latest: decisions[decisions.length - 1].timestamp
    } : null
  };
}

module.exports = {
  recordSpawnTelemetry,
  updateSpawnOutcome,
  getTelemetrySummary,
  getModelPerformance,
  getRoutingStats,
  loadTelemetry,
  saveTelemetry,
  TELEMETRY_VERSION,
  MAX_ROUTING_DECISIONS
};
