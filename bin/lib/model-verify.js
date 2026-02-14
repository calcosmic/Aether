#!/usr/bin/env node
/**
 * Model Routing Verification Module
 *
 * Verifies that model routing configuration is actually working.
 * Addresses the gap between "configuration exists" and "execution verified".
 *
 * @module bin/lib/model-verify
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

/**
 * Check if LiteLLM proxy is running
 * @returns {Promise<object>} Proxy status: { running: boolean, latency: number|null }
 */
async function checkLiteLLMProxy() {
  const startTime = Date.now();

  try {
    // Try to connect to LiteLLM proxy health endpoint
    const response = await fetch('http://localhost:4000/health', {
      method: 'GET',
      signal: AbortSignal.timeout(5000)
    });

    const latency = Date.now() - startTime;

    if (response.ok) {
      return {
        running: true,
        latency,
        status: response.status
      };
    }

    return {
      running: false,
      latency: null,
      status: response.status
    };
  } catch (error) {
    return {
      running: false,
      latency: null,
      error: error.message
    };
  }
}

/**
 * Verify model assignment for a caste using aether-utils.sh
 * @param {string} caste - Caste name (prime, builder, oracle, etc.)
 * @returns {object} Model assignment: { assigned: boolean, model: string|null, via: string }
 */
function verifyModelAssignment(caste) {
  try {
    // Check if aether-utils.sh exists
    const utilsPath = path.join(process.cwd(), '.aether', 'aether-utils.sh');
    if (!fs.existsSync(utilsPath)) {
      return {
        assigned: false,
        model: null,
        via: 'not_found',
        error: 'aether-utils.sh not found'
      };
    }

    // Try to get model profile
    const result = execSync(`bash "${utilsPath}" model-profile get ${caste}`, {
      encoding: 'utf8',
      stdio: ['pipe', 'pipe', 'pipe']
    });

    // Parse result - should be JSON
    try {
      const profile = JSON.parse(result.trim());
      return {
        assigned: !!profile.model,
        model: profile.model || null,
        via: profile.source || 'unknown',
        profile
      };
    } catch (parseError) {
      return {
        assigned: false,
        model: null,
        via: 'parse_error',
        error: `Failed to parse profile: ${parseError.message}`,
        raw: result
      };
    }
  } catch (error) {
    return {
      assigned: false,
      model: null,
      via: 'error',
      error: error.message
    };
  }
}

/**
 * Check ANTHROPIC_MODEL and ANTHROPIC_BASE_URL environment variables
 * @returns {object} Environment status: { model: string|null, baseUrl: string|null, routingActive: boolean }
 */
function checkAnthropicModelEnv() {
  const model = process.env.ANTHROPIC_MODEL || null;
  const baseUrl = process.env.ANTHROPIC_BASE_URL || null;

  // Routing is active if both are set and baseUrl points to LiteLLM proxy
  const routingActive = !!model && !!baseUrl && baseUrl.includes('localhost:4000');

  return {
    model,
    baseUrl,
    routingActive
  };
}

/**
 * Verify worker spawn environment
 * Simulates what build.md does before spawning workers
 * @returns {object} Spawn verification: { wouldRoute: boolean, model: string, issues: string[] }
 */
function verifyWorkerSpawnEnv() {
  const issues = [];

  // Check ANTHROPIC_MODEL
  const model = process.env.ANTHROPIC_MODEL;
  if (!model) {
    issues.push('ANTHROPIC_MODEL not set');
  }

  // Check ANTHROPIC_BASE_URL
  const baseUrl = process.env.ANTHROPIC_BASE_URL;
  if (!baseUrl) {
    issues.push('ANTHROPIC_BASE_URL not set');
  } else if (!baseUrl.includes('localhost:4000')) {
    issues.push('ANTHROPIC_BASE_URL does not point to LiteLLM proxy (localhost:4000)');
  }

  // Check WORKER_NAME
  const workerName = process.env.WORKER_NAME;
  if (!workerName) {
    issues.push('WORKER_NAME not set (optional but recommended)');
  }

  // Check CASTE
  const caste = process.env.CASTE;
  if (!caste) {
    issues.push('CASTE not set (optional but recommended)');
  }

  // Would routing work?
  const wouldRoute = !!model && !!baseUrl && baseUrl.includes('localhost:4000');

  return {
    wouldRoute,
    model: model || 'default (claude-opus-4-6)',
    caste: caste || 'unknown',
    workerName: workerName || 'unknown',
    issues
  };
}

/**
 * Check model profile configuration file
 * @param {string} repoPath - Path to repository root
 * @returns {object} Profile file status: { exists: boolean, path: string, profiles: object }
 */
function checkModelProfilesFile(repoPath) {
  const profilesPath = path.join(repoPath, '.aether', 'model-profiles.yaml');

  if (!fs.existsSync(profilesPath)) {
    return {
      exists: false,
      path: profilesPath,
      profiles: {}
    };
  }

  try {
    const content = fs.readFileSync(profilesPath, 'utf8');

    // Simple YAML parsing for model profiles
    // Looks for patterns like "prime: glm-5" or "builder: kimi-k2.5"
    const profiles = {};
    const lines = content.split('\n');

    for (const line of lines) {
      const match = line.match(/^(\w+):\s*(.+)$/);
      if (match) {
        const [, caste, model] = match;
        profiles[caste] = model.trim();
      }
    }

    return {
      exists: true,
      path: profilesPath,
      profiles
    };
  } catch (error) {
    return {
      exists: true,
      path: profilesPath,
      error: error.message,
      profiles: {}
    };
  }
}

/**
 * Create comprehensive verification report
 * @param {string} repoPath - Path to repository root
 * @returns {Promise<object>} Complete verification report
 */
async function createVerificationReport(repoPath) {
  const issues = [];

  // Check LiteLLM proxy
  const proxy = await checkLiteLLMProxy();
  if (!proxy.running) {
    issues.push('LiteLLM proxy is not running on localhost:4000');
  }

  // Check environment
  const env = checkAnthropicModelEnv();
  if (!env.routingActive) {
    issues.push('ANTHROPIC_MODEL/ANTHROPIC_BASE_URL not configured for proxy routing');
  }

  // Check model profiles file
  const profilesFile = checkModelProfilesFile(repoPath);
  if (!profilesFile.exists) {
    issues.push('Model profiles file not found (.aether/model-profiles.yaml)');
  }

  // Verify caste assignments
  const castes = {};
  const casteNames = ['prime', 'builder', 'oracle', 'scout'];

  for (const caste of casteNames) {
    const assignment = verifyModelAssignment(caste);
    castes[caste] = assignment;

    if (!assignment.assigned) {
      issues.push(`Model not assigned for caste: ${caste}`);
    }
  }

  // Check worker spawn environment
  const spawnEnv = verifyWorkerSpawnEnv();

  // Generate recommendation
  let recommendation;
  if (issues.length === 0) {
    recommendation = 'All checks passed. Model routing is properly configured and verified.';
  } else if (!proxy.running) {
    recommendation = 'Start LiteLLM proxy: litellm --config /path/to/config.yaml';
  } else if (!env.routingActive) {
    recommendation = 'Set environment variables: export ANTHROPIC_MODEL=your-model && export ANTHROPIC_BASE_URL=http://localhost:4000';
  } else {
    recommendation = 'Review configuration files and environment variables';
  }

  return {
    proxy,
    env,
    profilesFile,
    castes,
    spawnEnv,
    issues,
    recommendation
  };
}

module.exports = {
  checkLiteLLMProxy,
  verifyModelAssignment,
  checkAnthropicModelEnv,
  verifyWorkerSpawnEnv,
  checkModelProfilesFile,
  createVerificationReport
};
