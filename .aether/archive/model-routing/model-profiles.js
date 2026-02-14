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
    const config = yaml.load(content);

    // Substitute environment variables in proxy config
    if (config.proxy) {
      if (config.proxy.auth_token) {
        config.proxy.auth_token = substituteEnvVars(config.proxy.auth_token);
      }
      if (config.proxy.endpoint) {
        config.proxy.endpoint = substituteEnvVars(config.proxy.endpoint);
      }
    }

    return config;
  } catch (error) {
    throw new ConfigurationError(
      `Invalid YAML in model profiles file: ${error.message}`,
      { path: profilePath, originalError: error.message }
    );
  }
}

/**
 * Substitute environment variables in a string
 * Supports ${VAR} and ${VAR:-default} syntax
 * @param {string} str - String with potential env vars
 * @returns {string} String with env vars substituted
 */
function substituteEnvVars(str) {
  if (typeof str !== 'string') return str;

  // Match ${VAR:-default} or ${VAR}
  return str.replace(/\$\{([^}:]+)(?::-([^}]*))?\}/g, (match, varName, defaultValue) => {
    const envValue = process.env[varName];
    if (envValue !== undefined && envValue !== '') {
      return envValue;
    }
    return defaultValue !== undefined ? defaultValue : '';
  });
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

/**
 * Set user override for a caste's model
 * @param {string} repoPath - Path to repository root
 * @param {string} caste - Caste name to override
 * @param {string} model - Model name to assign
 * @returns {object} {success: true, previous: string|null}
 * @throws {ValidationError} If caste or model is invalid
 */
function setModelOverride(repoPath, caste, model) {
  const profiles = loadModelProfiles(repoPath);

  // Validate caste exists
  const casteValidation = validateCaste(profiles, caste);
  if (!casteValidation.valid) {
    const { ValidationError } = require('./errors');
    throw new ValidationError(
      `Invalid caste '${caste}'. Valid castes: ${casteValidation.castes.join(', ')}`,
      { caste, validCastes: casteValidation.castes }
    );
  }

  // Validate model exists
  const modelValidation = validateModel(profiles, model);
  if (!modelValidation.valid) {
    const { ValidationError } = require('./errors');
    throw new ValidationError(
      `Invalid model '${model}'. Valid models: ${modelValidation.models.join(', ')}`,
      { model, validModels: modelValidation.models }
    );
  }

  // Get previous override if exists
  const previous = profiles.user_overrides?.[caste] || null;

  // Read current YAML content
  const profilePath = path.join(repoPath, '.aether', 'model-profiles.yaml');
  const content = fs.readFileSync(profilePath, 'utf8');
  const data = yaml.load(content);

  // Ensure user_overrides section exists
  if (!data.user_overrides) {
    data.user_overrides = {};
  }

  // Set the override
  data.user_overrides[caste] = model;

  // Write back with proper YAML formatting
  const yamlContent = yaml.dump(data, {
    indent: 2,
    lineWidth: -1,
    noRefs: true,
    sortKeys: false,
  });

  fs.writeFileSync(profilePath, yamlContent, 'utf8');

  return { success: true, previous };
}

/**
 * Reset user override for a caste (remove override)
 * @param {string} repoPath - Path to repository root
 * @param {string} caste - Caste name to reset
 * @returns {object} {success: true, hadOverride: boolean}
 * @throws {ValidationError} If caste is invalid
 */
function resetModelOverride(repoPath, caste) {
  const profiles = loadModelProfiles(repoPath);

  // Validate caste exists
  const casteValidation = validateCaste(profiles, caste);
  if (!casteValidation.valid) {
    const { ValidationError } = require('./errors');
    throw new ValidationError(
      `Invalid caste '${caste}'. Valid castes: ${casteValidation.castes.join(', ')}`,
      { caste, validCastes: casteValidation.castes }
    );
  }

  // Check if override exists
  const hadOverride = profiles.user_overrides?.[caste] !== undefined;

  if (hadOverride) {
    // Read current YAML content
    const profilePath = path.join(repoPath, '.aether', 'model-profiles.yaml');
    const content = fs.readFileSync(profilePath, 'utf8');
    const data = yaml.load(content);

    // Remove the override
    if (data.user_overrides) {
      delete data.user_overrides[caste];

      // Clean up empty user_overrides section
      if (Object.keys(data.user_overrides).length === 0) {
        delete data.user_overrides;
      }

      // Write back with proper YAML formatting
      const yamlContent = yaml.dump(data, {
        indent: 2,
        lineWidth: -1,
        noRefs: true,
        sortKeys: false,
      });

      fs.writeFileSync(profilePath, yamlContent, 'utf8');
    }
  }

  return { success: true, hadOverride };
}

/**
 * Get effective model for a caste (respecting overrides)
 * @param {object} profiles - Parsed model profiles
 * @param {string} caste - Caste name
 * @returns {object} {model: string, source: 'override'|'default'|'fallback'}
 */
function getEffectiveModel(profiles, caste) {
  if (!profiles || typeof profiles !== 'object') {
    return { model: DEFAULT_MODEL, source: 'fallback' };
  }

  // Check user overrides first
  const override = profiles.user_overrides?.[caste];
  if (override) {
    return { model: override, source: 'override' };
  }

  // Fall back to worker_models default
  const defaultModel = profiles.worker_models?.[caste];
  if (defaultModel) {
    return { model: defaultModel, source: 'default' };
  }

  // Final fallback
  return { model: DEFAULT_MODEL, source: 'fallback' };
}

/**
 * Get current user overrides
 * @param {object} profiles - Parsed model profiles
 * @returns {object} User overrides object (empty if none)
 */
function getUserOverrides(profiles) {
  if (!profiles || typeof profiles !== 'object') {
    return {};
  }

  return profiles.user_overrides || {};
}

/**
 * Get the appropriate model for a task based on keyword matching
 * @param {object} taskRouting - Task routing configuration from profiles
 * @param {string} taskDescription - Description of the task to route
 * @returns {string|null} Model name for the task, or null if no match
 */
function getModelForTask(taskRouting, taskDescription) {
  if (!taskRouting || !taskDescription) {
    return null;
  }

  const normalizedTask = taskDescription.toLowerCase();
  const complexityIndicators = taskRouting.complexity_indicators || {};

  // Iterate through complexity indicators (complex, simple, validate)
  for (const [complexity, config] of Object.entries(complexityIndicators)) {
    if (!config || !config.keywords || !Array.isArray(config.keywords)) {
      continue;
    }

    // Check if any keyword is a substring of the task description
    const hasMatch = config.keywords.some(keyword =>
      normalizedTask.includes(keyword.toLowerCase())
    );

    if (hasMatch) {
      return config.model;
    }
  }

  // Return default model if no keywords match
  return taskRouting.default_model || null;
}

/**
 * Select the appropriate model for a task with full precedence chain
 * Precedence: CLI override > user override > task routing > caste default > fallback
 * @param {object} profiles - Parsed model profiles
 * @param {string} caste - Caste name (e.g., 'builder', 'watcher')
 * @param {string} taskDescription - Description of the task
 * @param {string|null} cliOverride - Optional CLI-provided model override
 * @returns {object} { model: string, source: string } with source tracking
 */
function selectModelForTask(profiles, caste, taskDescription, cliOverride = null) {
  // 1. CLI override (highest precedence)
  if (cliOverride) {
    const validation = validateModel(profiles, cliOverride);
    if (validation.valid) {
      return { model: cliOverride, source: 'cli-override' };
    }
  }

  // 2. User override
  if (profiles && profiles.user_overrides && profiles.user_overrides[caste]) {
    return { model: profiles.user_overrides[caste], source: 'user-override' };
  }

  // 3. Task-based routing
  if (taskDescription && profiles && profiles.task_routing) {
    const taskModel = getModelForTask(profiles.task_routing, taskDescription);
    if (taskModel) {
      return { model: taskModel, source: 'task-routing' };
    }
  }

  // 4. Caste default
  if (profiles && profiles.worker_models && profiles.worker_models[caste]) {
    return { model: profiles.worker_models[caste], source: 'caste-default' };
  }

  // 5. Fallback (lowest precedence)
  return { model: DEFAULT_MODEL, source: 'fallback' };
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
  setModelOverride,
  resetModelOverride,
  getEffectiveModel,
  getUserOverrides,
  getModelForTask,
  selectModelForTask,
  DEFAULT_MODEL,
};
