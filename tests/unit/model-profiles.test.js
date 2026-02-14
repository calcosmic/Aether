const fs = require('fs');
const path = require('path');
const test = require('ava');
const proxyquire = require('proxyquire');
const sinon = require('sinon');

const MODEL_PROFILES_PATH = path.join(__dirname, '../../bin/lib/model-profiles.js');
const YAML_PATH = path.join(__dirname, '../../.aether/model-profiles.yaml');

/**
 * Helper to create a mock model profiles object
 * @returns {object} Mock profiles object
 */
function createMockProfiles() {
  return {
    version: '1.0',
    description: 'Test profiles',
    worker_models: {
      builder: 'kimi-k2.5',
      watcher: 'kimi-k2.5',
      scout: 'kimi-k2.5',
      chaos: 'kimi-k2.5',
      architect: 'glm-5',
      oracle: 'minimax-2.5',
      prime: 'glm-5',
      colonizer: 'kimi-k2.5',
      route_setter: 'kimi-k2.5',
      archaeologist: 'glm-5',
    },
    model_metadata: {
      'glm-5': {
        description: 'Test GLM-5',
        provider: 'z_ai',
        capabilities: ['planning'],
        context_window: 200000,
        speed: 'medium',
        cost_tier: 'high',
      },
      'minimax-2.5': {
        description: 'Test MiniMax',
        provider: 'minimax',
        capabilities: ['browse', 'search'],
        context_window: 200000,
        speed: 'fast',
        cost_tier: 'medium',
      },
      'kimi-k2.5': {
        description: 'Test Kimi',
        provider: 'kimi',
        capabilities: ['coding'],
        context_window: 256000,
        speed: 'fast',
        cost_tier: 'low',
      },
    },
    proxy: {
      endpoint: 'http://localhost:4000',
      auth_token: 'sk-litellm-local',
      health_check: 'http://localhost:4000/health',
    },
  };
}

// ============================================
// loadModelProfiles tests
// ============================================

test('loadModelProfiles successfully loads valid YAML', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = modelProfiles.loadModelProfiles(path.join(__dirname, '../..'));

  t.truthy(profiles, 'Should return profiles object');
  t.is(profiles.version, '1.0', 'Should have correct version');
  t.truthy(profiles.worker_models, 'Should have worker_models');
  t.truthy(profiles.model_metadata, 'Should have model_metadata');
});

test('loadModelProfiles throws ConfigurationError for missing file', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);

  const error = t.throws(() => {
    modelProfiles.loadModelProfiles('/nonexistent/path');
  });

  t.is(error.name, 'ConfigurationError');
  t.true(error.message.includes('Model profiles file not found'));
});

test('loadModelProfiles throws ConfigurationError for invalid YAML', t => {
  const fsMock = {
    existsSync: () => true,
    readFileSync: () => 'invalid: yaml: content: [',
  };

  const yamlMock = {
    load: () => {
      const error = new Error('YAML parse error');
      throw error;
    },
  };

  const modelProfiles = proxyquire(MODEL_PROFILES_PATH, {
    fs: fsMock,
    'js-yaml': yamlMock,
  });

  const error = t.throws(() => {
    modelProfiles.loadModelProfiles('/fake/path');
  });

  t.is(error.name, 'ConfigurationError');
  t.true(error.message.includes('Invalid YAML'));
});

test('loadModelProfiles throws ConfigurationError for read errors', t => {
  const fsMock = {
    existsSync: () => true,
    readFileSync: () => {
      throw new Error('Permission denied');
    },
  };

  const modelProfiles = proxyquire(MODEL_PROFILES_PATH, {
    fs: fsMock,
  });

  const error = t.throws(() => {
    modelProfiles.loadModelProfiles('/fake/path');
  });

  t.is(error.name, 'ConfigurationError');
  t.true(error.message.includes('Failed to read'));
});

// ============================================
// getModelForCaste tests
// ============================================

test('getModelForCaste returns correct model for known castes', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = createMockProfiles();

  t.is(modelProfiles.getModelForCaste(profiles, 'builder'), 'kimi-k2.5');
  t.is(modelProfiles.getModelForCaste(profiles, 'architect'), 'glm-5');
  t.is(modelProfiles.getModelForCaste(profiles, 'oracle'), 'minimax-2.5');
});

test('getModelForCaste returns default for unknown caste', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = createMockProfiles();

  const consoleStub = sinon.stub(console, 'warn');

  const result = modelProfiles.getModelForCaste(profiles, 'unknown_caste');

  t.is(result, modelProfiles.DEFAULT_MODEL);
  t.true(consoleStub.calledOnce);
  t.true(consoleStub.firstCall.args[0].includes('Unknown caste'));

  consoleStub.restore();
});

test('getModelForCaste handles null/undefined profiles gracefully', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);

  const consoleStub = sinon.stub(console, 'warn');

  t.is(modelProfiles.getModelForCaste(null, 'builder'), modelProfiles.DEFAULT_MODEL);
  t.is(modelProfiles.getModelForCaste(undefined, 'builder'), modelProfiles.DEFAULT_MODEL);
  t.is(modelProfiles.getModelForCaste('not an object', 'builder'), modelProfiles.DEFAULT_MODEL);

  t.is(consoleStub.callCount, 3);

  consoleStub.restore();
});

// ============================================
// validateCaste tests
// ============================================

test('validateCaste returns valid=true for known castes', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = createMockProfiles();

  t.true(modelProfiles.validateCaste(profiles, 'builder').valid);
  t.true(modelProfiles.validateCaste(profiles, 'watcher').valid);
  t.true(modelProfiles.validateCaste(profiles, 'architect').valid);
  t.true(modelProfiles.validateCaste(profiles, 'prime').valid);
});

test('validateCaste returns valid=false for unknown castes', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = createMockProfiles();

  t.false(modelProfiles.validateCaste(profiles, 'unknown').valid);
  t.false(modelProfiles.validateCaste(profiles, '').valid);
  t.false(modelProfiles.validateCaste(profiles, null).valid);
});

test('validateCaste returns complete list of valid castes', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = createMockProfiles();

  const result = modelProfiles.validateCaste(profiles, 'builder');

  t.true(Array.isArray(result.castes));
  t.true(result.castes.includes('builder'));
  t.true(result.castes.includes('architect'));
  t.true(result.castes.includes('oracle'));
  t.is(result.castes.length, 10);
});

test('validateCaste handles null/undefined profiles gracefully', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);

  const result1 = modelProfiles.validateCaste(null, 'builder');
  t.false(result1.valid);
  t.deepEqual(result1.castes, []);

  const result2 = modelProfiles.validateCaste(undefined, 'builder');
  t.false(result2.valid);
  t.deepEqual(result2.castes, []);
});

// ============================================
// validateModel tests
// ============================================

test('validateModel returns valid=true for known models', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = createMockProfiles();

  t.true(modelProfiles.validateModel(profiles, 'glm-5').valid);
  t.true(modelProfiles.validateModel(profiles, 'minimax-2.5').valid);
  t.true(modelProfiles.validateModel(profiles, 'kimi-k2.5').valid);
});

test('validateModel returns valid=false for unknown models', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = createMockProfiles();

  t.false(modelProfiles.validateModel(profiles, 'gpt-4').valid);
  t.false(modelProfiles.validateModel(profiles, 'unknown-model').valid);
  t.false(modelProfiles.validateModel(profiles, '').valid);
});

test('validateModel returns complete list of valid models', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = createMockProfiles();

  const result = modelProfiles.validateModel(profiles, 'glm-5');

  t.true(Array.isArray(result.models));
  t.true(result.models.includes('glm-5'));
  t.true(result.models.includes('minimax-2.5'));
  t.true(result.models.includes('kimi-k2.5'));
  t.is(result.models.length, 3);
});

test('validateModel handles null/undefined profiles gracefully', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);

  const result1 = modelProfiles.validateModel(null, 'glm-5');
  t.false(result1.valid);
  t.deepEqual(result1.models, []);

  const result2 = modelProfiles.validateModel(undefined, 'glm-5');
  t.false(result2.valid);
  t.deepEqual(result2.models, []);
});

// ============================================
// getProviderForModel tests
// ============================================

test('getProviderForModel returns correct provider for each model', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = createMockProfiles();

  t.is(modelProfiles.getProviderForModel(profiles, 'glm-5'), 'z_ai');
  t.is(modelProfiles.getProviderForModel(profiles, 'minimax-2.5'), 'minimax');
  t.is(modelProfiles.getProviderForModel(profiles, 'kimi-k2.5'), 'kimi');
});

test('getProviderForModel returns null for unknown model', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = createMockProfiles();

  t.is(modelProfiles.getProviderForModel(profiles, 'unknown-model'), null);
  t.is(modelProfiles.getProviderForModel(profiles, 'gpt-4'), null);
});

test('getProviderForModel handles null/undefined profiles gracefully', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);

  t.is(modelProfiles.getProviderForModel(null, 'glm-5'), null);
  t.is(modelProfiles.getProviderForModel(undefined, 'glm-5'), null);
  t.is(modelProfiles.getProviderForModel('not an object', 'glm-5'), null);
});

// ============================================
// getAllAssignments tests
// ============================================

test('getAllAssignments returns array with all castes', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = createMockProfiles();

  const assignments = modelProfiles.getAllAssignments(profiles);

  t.true(Array.isArray(assignments));
  t.is(assignments.length, 10);

  const casteNames = assignments.map(a => a.caste);
  t.true(casteNames.includes('builder'));
  t.true(casteNames.includes('architect'));
  t.true(casteNames.includes('oracle'));
});

test('getAllAssignments each entry has caste, model, provider fields', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = createMockProfiles();

  const assignments = modelProfiles.getAllAssignments(profiles);

  for (const assignment of assignments) {
    t.true('caste' in assignment, 'Assignment should have caste field');
    t.true('model' in assignment, 'Assignment should have model field');
    t.true('provider' in assignment, 'Assignment should have provider field');
    t.is(typeof assignment.caste, 'string');
    t.is(typeof assignment.model, 'string');
    t.true(assignment.provider === null || typeof assignment.provider === 'string');
  }
});

test('getAllAssignments includes correct provider for each caste', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = createMockProfiles();

  const assignments = modelProfiles.getAllAssignments(profiles);

  const builder = assignments.find(a => a.caste === 'builder');
  t.is(builder.model, 'kimi-k2.5');
  t.is(builder.provider, 'kimi');

  const architect = assignments.find(a => a.caste === 'architect');
  t.is(architect.model, 'glm-5');
  t.is(architect.provider, 'z_ai');

  const oracle = assignments.find(a => a.caste === 'oracle');
  t.is(oracle.model, 'minimax-2.5');
  t.is(oracle.provider, 'minimax');
});

test('getAllAssignments handles null/undefined profiles gracefully', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);

  t.deepEqual(modelProfiles.getAllAssignments(null), []);
  t.deepEqual(modelProfiles.getAllAssignments(undefined), []);
  t.deepEqual(modelProfiles.getAllAssignments('not an object'), []);
});

// ============================================
// getModelMetadata tests (bonus function)
// ============================================

test('getModelMetadata returns metadata for known models', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = createMockProfiles();

  const metadata = modelProfiles.getModelMetadata(profiles, 'glm-5');

  t.truthy(metadata);
  t.is(metadata.provider, 'z_ai');
  t.is(metadata.context_window, 200000);
});

test('getModelMetadata returns null for unknown model', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = createMockProfiles();

  t.is(modelProfiles.getModelMetadata(profiles, 'unknown'), null);
});

// ============================================
// getProxyConfig tests (bonus function)
// ============================================

test('getProxyConfig returns proxy configuration', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = createMockProfiles();

  const config = modelProfiles.getProxyConfig(profiles);

  t.truthy(config);
  t.is(config.endpoint, 'http://localhost:4000');
  t.is(config.auth_token, 'sk-litellm-local');
});

test('getProxyConfig returns null when proxy not configured', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = createMockProfiles();
  delete profiles.proxy;

  t.is(modelProfiles.getProxyConfig(profiles), null);
});

// ============================================
// DEFAULT_MODEL constant test
// ============================================

test('DEFAULT_MODEL is exported and has expected value', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);

  t.is(modelProfiles.DEFAULT_MODEL, 'kimi-k2.5');
});

// ============================================
// Integration test with actual YAML file
// ============================================

test('integration: load actual YAML and verify all castes', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const repoPath = path.join(__dirname, '../..');

  const profiles = modelProfiles.loadModelProfiles(repoPath);

  // Verify all expected castes exist
  const expectedCastes = [
    'prime', 'archaeologist', 'architect', 'oracle', 'route_setter',
    'builder', 'watcher', 'scout', 'chaos', 'colonizer'
  ];

  for (const caste of expectedCastes) {
    const result = modelProfiles.validateCaste(profiles, caste);
    t.true(result.valid, `Caste '${caste}' should be valid`);
  }

  // Verify all expected models exist
  const expectedModels = ['glm-5', 'minimax-2.5', 'kimi-k2.5'];

  for (const model of expectedModels) {
    const result = modelProfiles.validateModel(profiles, model);
    t.true(result.valid, `Model '${model}' should be valid`);
  }

  // Verify assignments work
  const assignments = modelProfiles.getAllAssignments(profiles);
  t.is(assignments.length, 10);

  // Verify specific mappings
  t.is(modelProfiles.getModelForCaste(profiles, 'builder'), 'kimi-k2.5');
  t.is(modelProfiles.getModelForCaste(profiles, 'architect'), 'glm-5');
  t.is(modelProfiles.getModelForCaste(profiles, 'oracle'), 'minimax-2.5');

  // Verify providers
  t.is(modelProfiles.getProviderForModel(profiles, 'glm-5'), 'z_ai');
  t.is(modelProfiles.getProviderForModel(profiles, 'minimax-2.5'), 'minimax');
  t.is(modelProfiles.getProviderForModel(profiles, 'kimi-k2.5'), 'kimi');
});
