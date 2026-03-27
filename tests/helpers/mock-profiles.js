/**
 * Centralized Mock Profile Helper
 *
 * Provides test utilities for model-profiles that read from the actual YAML
 * at runtime. This is the single source of truth for test fixture data --
 * if a caste's model changes in YAML, tests using these helpers update
 * automatically (TEST-01, TEST-02).
 *
 * Each function call reads fresh YAML -- no caching at module load time.
 * This is intentional: if YAML structure changes, tests break intentionally.
 */

const path = require('path');
const {
  loadModelProfiles,
  getModelForCaste,
  getModelMetadata,
  getProviderForModel,
} = require('../../bin/lib/model-profiles');

/** Path to repository root (from tests/helpers/) */
const REPO_ROOT = path.resolve(__dirname, '../..');

/**
 * Load and parse the actual model profiles from YAML.
 * @returns {object} Full parsed model profiles config
 */
function loadActualProfiles() {
  return loadModelProfiles(REPO_ROOT);
}

/**
 * Get a shallow copy of the worker_models map (caste -> model name).
 * @returns {object} e.g. { builder: 'glm-5-turbo', watcher: 'glm-5-turbo', ... }
 */
function getWorkerModels() {
  const profiles = loadActualProfiles();
  return { ...profiles.worker_models };
}

/**
 * Get all valid model names defined in model_metadata.
 * @returns {string[]} e.g. ['glm-5', 'glm-5-turbo', 'glm-4.5-air']
 */
function getModelNames() {
  const profiles = loadActualProfiles();
  return Object.keys(profiles.model_metadata || {});
}

/**
 * Get the assigned model for a specific caste from YAML.
 * @param {string} caste - Caste name (e.g., 'builder', 'watcher')
 * @returns {string} Model name assigned to the caste
 */
function getDefaultModelForCaste(caste) {
  const profiles = loadActualProfiles();
  return getModelForCaste(profiles, caste);
}

/**
 * Get all caste names defined in worker_models.
 * @returns {string[]} e.g. ['builder', 'watcher', 'scout', ...]
 */
function getCasteNames() {
  const profiles = loadActualProfiles();
  return Object.keys(profiles.worker_models || {});
}

/**
 * Get metadata for a specific model from YAML.
 * @param {string} modelName - Model name to look up
 * @returns {object|null} Model metadata object, or null if not found
 */
function getModelMeta(modelName) {
  const profiles = loadActualProfiles();
  return getModelMetadata(profiles, modelName);
}

/**
 * Get the provider for a specific model from YAML.
 * @param {string} modelName - Model name to look up
 * @returns {string|null} Provider name, or null if not found
 */
function getModelProvider(modelName) {
  const profiles = loadActualProfiles();
  return getProviderForModel(profiles, modelName);
}

/**
 * Build a complete mock profiles object starting from actual YAML,
 * with optional overrides for testing.
 *
 * @param {object} [overrides={}]
 * @param {object} [overrides.workerModels] - Caste-to-model overrides (merged over YAML)
 * @param {object} [overrides.modelMetadata] - Model metadata overrides (merged over YAML)
 * @param {object} [overrides.proxy] - Proxy config override. Pass null to omit proxy entirely.
 * @param {object} [overrides.taskRouting] - Full task routing replacement (not merged)
 * @returns {object} Complete mock profiles object suitable for testing
 */
function buildMockProfiles(overrides = {}) {
  const profiles = loadActualProfiles();

  const result = {
    version: profiles.version || '1.0',
    description: profiles.description || 'Test profiles',
    worker_models: {
      ...(profiles.worker_models || {}),
      ...(overrides.workerModels || {}),
    },
    model_metadata: {
      ...(profiles.model_metadata || {}),
      ...(overrides.modelMetadata || {}),
    },
  };

  // Handle proxy: omit if override is explicitly null or not present in YAML
  if (overrides.proxy === null) {
    // omit proxy key entirely
  } else if (profiles.proxy || overrides.proxy) {
    result.proxy = {
      ...(profiles.proxy || {}),
      ...(overrides.proxy || {}),
    };
  }

  // Handle task_routing: use override if provided, otherwise use YAML value
  if (overrides.taskRouting !== undefined) {
    result.task_routing = overrides.taskRouting;
  } else if (profiles.task_routing) {
    result.task_routing = profiles.task_routing;
  }

  return result;
}

module.exports = {
  loadActualProfiles,
  getWorkerModels,
  getModelNames,
  getDefaultModelForCaste,
  getCasteNames,
  getModelMeta,
  getModelProvider,
  buildMockProfiles,
};
