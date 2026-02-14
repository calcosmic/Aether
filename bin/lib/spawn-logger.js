#!/usr/bin/env node
/**
 * Spawn Logger Module
 *
 * Provides utilities for logging worker spawn events with model information
 * and formatting spawn trees for display. Used by both CLI and programmatic
 * interfaces to track which models are used for each worker spawn.
 */

const fs = require('fs');
const path = require('path');
const { logActivity } = require('./logger');

/**
 * Caste emoji mapping for display
 */
const CASTE_EMOJIS = {
  prime: '🏛️',
  builder: '🔨',
  watcher: '👁️',
  oracle: '🔮',
  scout: '🔍',
  chaos: '🎲',
  architect: '📐',
  archaeologist: '🏺',
  colonizer: '🌱',
  route_setter: '🎯',
};

/**
 * Status emoji mapping
 */
const STATUS_EMOJIS = {
  spawned: '○',
  completed: '✓',
  failed: '✗',
  blocked: '🚫',
};

/**
 * Log a worker spawn event with model information
 * @param {string} repoPath - Repository root path
 * @param {Object} spawnInfo - Spawn details
 * @param {string} spawnInfo.parent - Parent ant name (e.g., "Queen")
 * @param {string} spawnInfo.caste - Worker caste (e.g., "builder")
 * @param {string} spawnInfo.child - Child ant name (e.g., "Builder-1")
 * @param {string} spawnInfo.task - Task description
 * @param {string} spawnInfo.model - Model used (e.g., "kimi-k2.5")
 * @param {string} spawnInfo.status - Spawn status (default: "spawned")
 * @returns {Promise<boolean>} True if logged successfully
 */
async function logSpawn(repoPath, { parent, caste, child, task, model, status = 'spawned' }) {
  try {
    const timestamp = new Date().toISOString();
    const logLine = `${timestamp}|${parent}|${caste}|${child}|${task}|${model || 'default'}|${status}\n`;

    const spawnTreePath = path.join(repoPath, '.aether', 'data', 'spawn-tree.txt');

    // Ensure logs directory exists
    const logsDir = path.dirname(spawnTreePath);
    if (!fs.existsSync(logsDir)) {
      fs.mkdirSync(logsDir, { recursive: true });
    }

    // Append to spawn tree log
    fs.appendFileSync(spawnTreePath, logLine);

    // Also log to activity log via logger module
    const casteForLog = caste || 'ant';
    const description = `${child} (${caste}): ${task} [model: ${model || 'default'}]`;
    logActivity('SPAWN', casteForLog, description);

    return true;
  } catch (error) {
    // Silent fail - don't cascade errors from logging
    return false;
  }
}

/**
 * Parse a spawn tree line into its components
 * @param {string} line - Raw line from spawn-tree.txt
 * @returns {Object|null} Parsed spawn record or null if invalid
 */
function parseSpawnLine(line) {
  const parts = line.split('|');

  // Support multiple formats:
  // New format (7 parts): timestamp|parent|caste|child|task|model|status
  // Old format (6 parts): timestamp|parent|caste|child|task|status
  // Complete format (3-4 parts): timestamp|ant_name|status|summary (optional)

  if (parts.length === 7) {
    // New format with model
    return {
      timestamp: parts[0],
      parent: parts[1],
      caste: parts[2],
      child: parts[3],
      task: parts[4],
      model: parts[5],
      status: parts[6],
    };
  } else if (parts.length === 6) {
    // Old format without model - default to 'unknown'
    return {
      timestamp: parts[0],
      parent: parts[1],
      caste: parts[2],
      child: parts[3],
      task: parts[4],
      model: 'unknown',
      status: parts[5],
    };
  } else if (parts.length >= 3 && parts.length <= 4) {
    // spawn-complete format: timestamp|ant_name|status|summary
    // This is a completion record, treat as special case
    return {
      timestamp: parts[0],
      parent: null,
      caste: 'complete',
      child: parts[1],
      task: parts[3] || '',
      model: 'n/a',
      status: parts[2],
      isCompletion: true,
    };
  }

  // Unrecognized format
  return null;
}

/**
 * Format spawn tree for display
 * @param {string} repoPath - Repository root path
 * @returns {string} Formatted tree
 */
function formatSpawnTree(repoPath) {
  const spawnTreePath = path.join(repoPath, '.aether', 'data', 'spawn-tree.txt');

  if (!fs.existsSync(spawnTreePath)) {
    return 'No spawn history found.';
  }

  const content = fs.readFileSync(spawnTreePath, 'utf8').trim();
  if (!content) {
    return 'No spawn history found.';
  }

  const lines = content.split('\n');

  // Parse and format as tree
  const formatted = lines.map((line) => {
    const record = parseSpawnLine(line);
    if (!record) {
      return `  ? Invalid line: ${line.slice(0, 50)}`;
    }

    // Handle completion records differently
    if (record.isCompletion) {
      const statusEmoji = STATUS_EMOJIS[record.status] || STATUS_EMOJIS.spawned;
      return `${statusEmoji} ${record.child}: ${record.status}${record.task ? ' - ' + record.task.slice(0, 40) : ''}`;
    }

    const casteEmoji = getCasteEmoji(record.caste);
    const statusEmoji = STATUS_EMOJIS[record.status] || STATUS_EMOJIS.spawned;
    const modelDisplay = record.model || 'default';

    return `${statusEmoji} ${record.parent} → ${casteEmoji} ${record.child} [${modelDisplay}] - ${record.task}`;
  });

  return formatted.join('\n');
}

/**
 * Get spawn tree as structured data
 * @param {string} repoPath - Repository root path
 * @returns {Array<Object>} Array of spawn records
 */
function getSpawnTreeData(repoPath) {
  const spawnTreePath = path.join(repoPath, '.aether', 'data', 'spawn-tree.txt');

  if (!fs.existsSync(spawnTreePath)) {
    return [];
  }

  const content = fs.readFileSync(spawnTreePath, 'utf8').trim();
  if (!content) {
    return [];
  }

  const lines = content.split('\n');
  return lines.map(parseSpawnLine).filter(Boolean);
}

/**
 * Get emoji for caste
 * @param {string} caste - Caste name
 * @returns {string} Emoji for the caste
 */
function getCasteEmoji(caste) {
  return CASTE_EMOJIS[caste] || '🐜';
}

/**
 * Get spawns by parent
 * @param {string} repoPath - Repository root path
 * @param {string} parentName - Parent ant name to filter by
 * @returns {Array<Object>} Array of spawn records for the parent
 */
function getSpawnsByParent(repoPath, parentName) {
  const allSpawns = getSpawnTreeData(repoPath);
  return allSpawns.filter((spawn) => spawn.parent === parentName);
}

/**
 * Get spawns by caste
 * @param {string} repoPath - Repository root path
 * @param {string} caste - Caste to filter by
 * @returns {Array<Object>} Array of spawn records for the caste
 */
function getSpawnsByCaste(repoPath, caste) {
  const allSpawns = getSpawnTreeData(repoPath);
  return allSpawns.filter((spawn) => spawn.caste === caste);
}

/**
 * Get spawns by model
 * @param {string} repoPath - Repository root path
 * @param {string} model - Model to filter by
 * @returns {Array<Object>} Array of spawn records using the model
 */
function getSpawnsByModel(repoPath, model) {
  const allSpawns = getSpawnTreeData(repoPath);
  return allSpawns.filter((spawn) => spawn.model === model);
}

module.exports = {
  logSpawn,
  formatSpawnTree,
  getSpawnTreeData,
  getCasteEmoji,
  parseSpawnLine,
  getSpawnsByParent,
  getSpawnsByCaste,
  getSpawnsByModel,
  CASTE_EMOJIS,
  STATUS_EMOJIS,
};
