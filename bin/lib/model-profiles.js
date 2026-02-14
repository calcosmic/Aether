#!/usr/bin/env node
/**
 * Model Profiles Library
 *
 * Reads and validates caste-to-model assignments from model-profiles.yaml.
 * Provides utilities for model routing and profile management.
 */

const fs = require('fs');
const path = require('path');
const yaml = require('js-yaml');
const { ConfigurationError } = require('./errors');

/**
 * Default model to use when caste is not found
 */
const DEFAULT_MODEL = 'kimi-k2.5';

/**
 * Load and parse model profiles from YAML file
 * @param {string} repoPath - Path to repository root
 * @returns {object} Parsed model profiles
 * @throws {ConfigurationError} If file not found or invalid YAML
 */
function loadModelProfiles(repoPath) {
  const profilePath = path.join(repoPath, '.aether', 'model-profiles.yaml');

  if (!fs.existsSync(profilePath)) {
    throw new ConfigurationError(
      `Model profiles file not found: ${profilePath}`,
      { path: profilePath }
    );
  }

  let content;
  try {
    content = fs.readFileSync(profilePath, 'utf8');
  } catch (error) {
    throw new ConfigurationError(
      `Failed to read model profiles file: ${error.message}`,
      { path: profilePath, originalError: error.message }
    );
  }

  try {
    return yaml.load(content);
  } catch (error) {
    throw new ConfigurationError(
      `Invalid YAML in model profiles file: ${error.message}`,
      { path: profilePath, originalError: error.message }
    );
  }
}

/**
 * Get the assigned model for a specific caste
 * @param {object} profiles - Parsed model profiles
 * @param {string} caste - Caste name (e.g., 'builder', 'watcher')
 * @returns {string} Model name for the caste, or default if not found
 */
function getModelForCaste(profiles, caste) {
  if (!profiles || typeof profiles !== 'object') {
    console.warn(`[WARN] Invalid profiles object, using default model: ${DEFAULT_MODEL}`);
    return DEFAULT_MODEL;
  }

  const model = profiles.worker_models?.[caste];

  if (!model) {
    console.warn(`[WARN] Unknown caste '${caste}', using default model: ${DEFAULT_MODEL}`);
    return DEFAULT_MODEL;
  }

  return model;
}

/**
 * Validate if a caste name is valid
 * @param {object} profiles - Parsed model profiles
 * @param {string} caste - Caste name to validate
 * @returns {object} { valid: boolean, castes: string[] }
 */
function validateCaste(profiles, caste) {
  if (!profiles || typeof profiles !== 'object') {
    return { valid: false, castes: [] };
  }

  const validCastes = Object.keys(profiles.worker_models || {});
  const valid = validCastes.includes(caste);

  return { valid, castes: validCastes };
}

/**
 * Validate if a model name is valid
 * @param {object} profiles - Parsed model profiles
 * @param {string} model - Model name to validate
 * @returns {object} { valid: boolean, models: string[] }
 */
function validateModel(profiles, model) {
  if (!profiles || typeof profiles !== 'object') {
    return { valid: false, models: [] };
  }

  const validModels = Object.keys(profiles.model_metadata || {});
  const valid = validModels.includes(model);

  return { valid, models: validModels };
}

/**
 * Get the provider for a specific model
 * @param {object} profiles - Parsed model profiles
 * @param {string} model - Model name
 * @returns {string|null} Provider name, or null if not found
 */
function getProviderForModel(profiles, model) {
  if (!profiles || typeof profiles !== 'object') {
    return null;
  }

  return profiles.model_metadata?.[model]?.provider || null;
}

/**
 * Get all caste-to-model assignments with provider info
 * @param {object} profiles - Parsed model profiles
 * @returns {Array<{caste: string, model: string, provider: string|null}>} Array of assignments
 */
function getAllAssignments(profiles) {
  if (!profiles || typeof profiles !== 'object') {
    return [];
  }

  const workerModels = profiles.worker_models || {};

  return Object.entries(workerModels).map(([caste, model]) => ({
    caste,
    model,
    provider: getProviderForModel(profiles, model),
  }));
}

/**
 * Get model metadata for a specific model
 * @param {object} profiles - Parsed model profiles
 * @param {string} model - Model name
 * @returns {object|null} Model metadata, or null if not found
 */
function getModelMetadata(profiles, model) {
  if (!profiles || typeof profiles !== 'object') {
    return null;
  }

  return profiles.model_metadata?.[model] || null;
}

/**
 * Get proxy configuration from profiles
 * @param {object} profiles - Parsed model profiles
 * @returns {object|null} Proxy configuration, or null if not found
 */
function getProxyConfig(profiles) {
  if (!profiles || typeof profiles !== 'object') {
    return null;
  }

  return profiles.proxy || null;
}

module.exports = {
  loadModelProfiles,
  getModelForCaste,
  validateCaste,
  validateModel,
  getProviderForModel,
  getAllAssignments,
  getModelMetadata,
  getProxyConfig,
  DEFAULT_MODEL,
};
