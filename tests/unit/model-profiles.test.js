const path = require('path');
const test = require('ava');
const proxyquire = require('proxyquire');
const sinon = require('sinon');

const MODEL_PROFILES_PATH = path.join(__dirname, '../../bin/lib/model-profiles.js');
const YAML_PATH = path.join(__dirname, '../../.aether/model-profiles.yaml');
const {
  buildMockProfiles,
  getDefaultModelForCaste,
  getModelNames,
  getCasteNames,
  getModelMeta,
  getModelProvider,
} = require('../helpers/mock-profiles');

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
  const profiles = buildMockProfiles();

  t.is(modelProfiles.getModelForCaste(profiles, 'builder'), getDefaultModelForCaste('builder'));
  t.is(modelProfiles.getModelForCaste(profiles, 'architect'), getDefaultModelForCaste('architect'));
  t.is(modelProfiles.getModelForCaste(profiles, 'oracle'), getDefaultModelForCaste('oracle'));
});

test('getModelForCaste returns default for unknown caste', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles();

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
  const profiles = buildMockProfiles();

  t.true(modelProfiles.validateCaste(profiles, 'builder').valid);
  t.true(modelProfiles.validateCaste(profiles, 'watcher').valid);
  t.true(modelProfiles.validateCaste(profiles, 'architect').valid);
  t.true(modelProfiles.validateCaste(profiles, 'prime').valid);
});

test('validateCaste returns valid=false for unknown castes', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles();

  t.false(modelProfiles.validateCaste(profiles, 'unknown').valid);
  t.false(modelProfiles.validateCaste(profiles, '').valid);
  t.false(modelProfiles.validateCaste(profiles, null).valid);
});

test('validateCaste returns complete list of valid castes', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles();

  const result = modelProfiles.validateCaste(profiles, 'builder');

  t.true(Array.isArray(result.castes));
  t.true(result.castes.includes('builder'));
  t.true(result.castes.includes('architect'));
  t.true(result.castes.includes('oracle'));
  t.is(result.castes.length, getCasteNames().length);
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
  const profiles = buildMockProfiles();

  t.true(modelProfiles.validateModel(profiles, 'glm-5').valid);
  t.true(modelProfiles.validateModel(profiles, 'glm-5-turbo').valid);
  t.true(modelProfiles.validateModel(profiles, 'glm-4.5-air').valid);
});

test('validateModel returns valid=false for unknown models', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles();

  t.false(modelProfiles.validateModel(profiles, 'gpt-4').valid);
  t.false(modelProfiles.validateModel(profiles, 'unknown-model').valid);
  t.false(modelProfiles.validateModel(profiles, '').valid);
});

test('validateModel returns complete list of valid models', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles();

  const result = modelProfiles.validateModel(profiles, 'glm-5');

  t.true(Array.isArray(result.models));
  t.true(result.models.includes('glm-5'));
  t.true(result.models.includes('glm-5-turbo'));
  t.true(result.models.includes('glm-4.5-air'));
  t.is(result.models.length, getModelNames().length);
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
  const profiles = buildMockProfiles();

  t.is(modelProfiles.getProviderForModel(profiles, 'glm-5'), getModelProvider('glm-5'));
  t.is(modelProfiles.getProviderForModel(profiles, 'glm-5-turbo'), getModelProvider('glm-5-turbo'));
  t.is(modelProfiles.getProviderForModel(profiles, 'glm-4.5-air'), getModelProvider('glm-4.5-air'));
});

test('getProviderForModel returns null for unknown model', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles();

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
  const profiles = buildMockProfiles();

  const assignments = modelProfiles.getAllAssignments(profiles);

  t.true(Array.isArray(assignments));
  t.is(assignments.length, getCasteNames().length);

  const casteNames = assignments.map(a => a.caste);
  t.true(casteNames.includes('builder'));
  t.true(casteNames.includes('architect'));
  t.true(casteNames.includes('oracle'));
});

test('getAllAssignments each entry has caste, model, provider fields', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles();

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
  const profiles = buildMockProfiles();

  const assignments = modelProfiles.getAllAssignments(profiles);

  const builder = assignments.find(a => a.caste === 'builder');
  t.is(builder.model, getDefaultModelForCaste('builder'));
  t.is(builder.provider, getModelProvider(getDefaultModelForCaste('builder')));

  const architect = assignments.find(a => a.caste === 'architect');
  t.is(architect.model, getDefaultModelForCaste('architect'));
  t.is(architect.provider, getModelProvider(getDefaultModelForCaste('architect')));

  const oracle = assignments.find(a => a.caste === 'oracle');
  t.is(oracle.model, getDefaultModelForCaste('oracle'));
  t.is(oracle.provider, getModelProvider(getDefaultModelForCaste('oracle')));
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
  const profiles = buildMockProfiles();

  const metadata = modelProfiles.getModelMetadata(profiles, 'glm-5');

  t.truthy(metadata);
  t.is(metadata.provider, getModelProvider('glm-5'));
  t.is(metadata.context_window, 200000);
});

test('getModelMetadata returns null for unknown model', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles();

  t.is(modelProfiles.getModelMetadata(profiles, 'unknown'), null);
});

// ============================================
// getProxyConfig tests (bonus function)
// ============================================

test('getProxyConfig returns proxy configuration', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles();

  const config = modelProfiles.getProxyConfig(profiles);

  t.truthy(config);
  t.is(config.endpoint, 'http://localhost:4000');
  t.is(config.auth_token, 'sk-litellm-local');
});

test('getProxyConfig returns null when proxy not configured', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles();
  delete profiles.proxy;

  t.is(modelProfiles.getProxyConfig(profiles), null);
});

// ============================================
// DEFAULT_MODEL constant test
// ============================================

test('DEFAULT_MODEL is exported and has expected value', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);

  t.is(modelProfiles.DEFAULT_MODEL, 'glm-5-turbo');
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
  const expectedModels = getModelNames();

  for (const model of expectedModels) {
    const result = modelProfiles.validateModel(profiles, model);
    t.true(result.valid, `Model '${model}' should be valid`);
  }

  // Verify assignments work
  const assignments = modelProfiles.getAllAssignments(profiles);
  t.is(assignments.length, 10);

  // Verify all castes use the expected default model
  t.is(modelProfiles.getModelForCaste(profiles, 'builder'), getDefaultModelForCaste('builder'));
  t.is(modelProfiles.getModelForCaste(profiles, 'architect'), getDefaultModelForCaste('architect'));
  t.is(modelProfiles.getModelForCaste(profiles, 'oracle'), getDefaultModelForCaste('oracle'));

  // Verify providers for all models
  for (const model of getModelNames()) {
    t.is(modelProfiles.getProviderForModel(profiles, model), getModelProvider(model), `Provider for ${model} should match`);
  }
});
