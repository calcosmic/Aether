#!/usr/bin/env node
/**
 * Proxy Health Library
 *
 * Health checking utilities for LiteLLM proxy.
 * Provides functions to verify proxy status, model availability, and routing.
 */

const { ConfigurationError } = require('./errors');

/**
 * Default timeout for health check requests (ms)
 */
const DEFAULT_TIMEOUT = 5000;

/**
 * Check proxy health endpoint
 * @param {string} endpoint - Proxy endpoint URL (e.g., http://localhost:4000)
 * @param {number} timeoutMs - Timeout in milliseconds (default: 5000)
 * @returns {Promise<object>} Health status object
 *   - healthy: boolean - Whether proxy is healthy
 *   - status: number - HTTP status code
 *   - latency: number - Response time in milliseconds
 *   - error: string|null - Error message if unhealthy
 *   - models: string[]|null - Available model IDs if healthy
 */
async function checkProxyHealth(endpoint, timeoutMs = DEFAULT_TIMEOUT) {
  const startTime = Date.now();
  const healthUrl = `${endpoint.replace(/\/$/, '')}/health`;

  try {
    // Use native fetch with AbortSignal for timeout (Node 18+)
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), timeoutMs);

    const response = await fetch(healthUrl, {
      method: 'GET',
      signal: controller.signal,
      headers: {
        'Accept': 'application/json',
      },
    });

    clearTimeout(timeoutId);
    const latency = Date.now() - startTime;

    if (!response.ok) {
      return {
        healthy: false,
        status: response.status,
        latency,
        error: `HTTP ${response.status}: ${response.statusText}`,
        models: null,
      };
    }

    // Try to get models list
    let models = null;
    try {
      models = await getProxyModels(endpoint, timeoutMs);
    } catch {
      // Models endpoint may fail even if health passes
      models = null;
    }

    return {
      healthy: true,
      status: response.status,
      latency,
      error: null,
      models,
    };
  } catch (error) {
    const latency = Date.now() - startTime;

    if (error.name === 'AbortError') {
      return {
        healthy: false,
        status: 0,
        latency,
        error: `Timeout after ${timeoutMs}ms`,
        models: null,
      };
    }

    // Handle network errors
    const errorMessage = error.message || 'Unknown error';
    return {
      healthy: false,
      status: 0,
      latency,
      error: errorMessage,
      models: null,
    };
  }
}

/**
 * Verify a specific model is routable through the proxy
 * @param {string} endpoint - Proxy endpoint URL
 * @param {string} model - Model name to verify
 * @param {number} timeoutMs - Timeout in milliseconds
 * @returns {Promise<object>} Verification result
 *   - available: boolean - Whether model is available on proxy
 *   - found: boolean - Whether model was found in proxy's model list
 *   - model: string - The model name that was checked
 */
async function verifyModelRouting(endpoint, model, timeoutMs = DEFAULT_TIMEOUT) {
  try {
    const models = await getProxyModels(endpoint, timeoutMs);

    if (!models) {
      return {
        available: false,
        found: false,
        model,
        error: 'Could not fetch models from proxy',
      };
    }

    const found = models.includes(model);

    return {
      available: found,
      found,
      model,
      error: found ? null : `Model '${model}' not found in proxy model list`,
    };
  } catch (error) {
    return {
      available: false,
      found: false,
      model,
      error: error.message || 'Failed to verify model routing',
    };
  }
}

/**
 * Fetch available models from proxy
 * @param {string} endpoint - Proxy endpoint URL
 * @param {number} timeoutMs - Timeout in milliseconds
 * @returns {Promise<string[]|null>} Array of model IDs or null on error
 */
async function getProxyModels(endpoint, timeoutMs = DEFAULT_TIMEOUT) {
  const modelsUrl = `${endpoint.replace(/\/$/, '')}/models`;

  try {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), timeoutMs);

    const response = await fetch(modelsUrl, {
      method: 'GET',
      signal: controller.signal,
      headers: {
        'Accept': 'application/json',
      },
    });

    clearTimeout(timeoutId);

    if (!response.ok) {
      return null;
    }

    const data = await response.json();

    // LiteLLM returns models in OpenAI format: { data: [{ id: 'model-name' }, ...] }
    if (data && Array.isArray(data.data)) {
      return data.data.map(m => m.id).filter(Boolean);
    }

    // Fallback: try to extract models from various formats
    if (Array.isArray(data)) {
      return data.map(m => m.id || m.name || m).filter(Boolean);
    }

    return null;
  } catch {
    return null;
  }
}

/**
 * Format proxy health status for display
 * @param {object} health - Health status object from checkProxyHealth
 * @returns {string} Formatted status string with colors/emoji
 */
function formatProxyStatus(health) {
  if (!health) {
    return 'Unknown';
  }

  if (health.healthy) {
    const latencyStr = health.latency ? `(${health.latency}ms)` : '';
    return `✓ Healthy ${latencyStr}`.trim();
  }

  const errorStr = health.error || 'Unknown error';
  return `✗ Unhealthy: ${errorStr}`;
}

/**
 * Format proxy health status with ANSI colors
 * @param {object} health - Health status object
 * @param {object} colors - Color functions from colors.js
 * @returns {string} Colored status string
 */
function formatProxyStatusColored(health, colors) {
  if (!health) {
    return colors.dim('Unknown');
  }

  if (health.healthy) {
    const latencyStr = health.latency ? `(${health.latency}ms)` : '';
    return `${colors.success('✓')} ${colors.success('Healthy')} ${colors.dim(latencyStr)}`.trim();
  }

  const errorStr = health.error || 'Unknown error';
  return `${colors.error('✗')} ${colors.error('Unhealthy')}: ${errorStr}`;
}

/**
 * Verify caste model assignments against proxy
 * @param {string} endpoint - Proxy endpoint URL
 * @param {object} profiles - Model profiles object
 * @param {number} timeoutMs - Timeout in milliseconds
 * @returns {Promise<object>} Verification results for all castes
 */
async function verifyCasteModels(endpoint, profiles, timeoutMs = DEFAULT_TIMEOUT) {
  const results = {};
  const workerModels = profiles?.worker_models || {};

  for (const [caste, model] of Object.entries(workerModels)) {
    const verification = await verifyModelRouting(endpoint, model, timeoutMs);
    results[caste] = {
      model,
      ...verification,
    };
  }

  return results;
}

module.exports = {
  checkProxyHealth,
  verifyModelRouting,
  getProxyModels,
  formatProxyStatus,
  formatProxyStatusColored,
  verifyCasteModels,
  DEFAULT_TIMEOUT,
};
