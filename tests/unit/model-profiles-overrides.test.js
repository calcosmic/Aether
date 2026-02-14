const fs = require('fs');
const path = require('path');
const test = require('ava');
const proxyquire = require('proxyquire');
const sinon = require('sinon');

const MODEL_PROFILES_PATH = path.join(__dirname, '../../bin/lib/model-profiles.js');

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
// setModelOverride tests
// ============================================

test('setModelOverride successfully sets override for valid caste/model', t => {
  const profiles = createMockProfiles();
  const yamlContent = require('js-yaml').dump(profiles);

  const fsMock = {
    existsSync: sinon.stub().returns(true),
    readFileSync: sinon.stub().returns(yamlContent),
    writeFileSync: sinon.stub(),
  };

  const modelProfiles = proxyquire(MODEL_PROFILES_PATH, {
    fs: fsMock,
  });

  const result = modelProfiles.setModelOverride('/fake/path', 'builder', 'glm-5');

  t.true(result.success);
  t.is(result.previous, null);
  t.true(fsMock.writeFileSync.calledOnce);

  // Verify the written content includes user_overrides
  const writtenContent = fsMock.writeFileSync.firstCall.args[1];
  t.true(writtenContent.includes('user_overrides'));
  t.true(writtenContent.includes('builder: glm-5'));
});

test('setModelOverride returns previous value when updating existing override', t => {
  const profiles = createMockProfiles();
  profiles.user_overrides = { builder: 'kimi-k2.5' };
  const yamlContent = require('js-yaml').dump(profiles);

  const fsMock = {
    existsSync: sinon.stub().returns(true),
    readFileSync: sinon.stub().returns(yamlContent),
    writeFileSync: sinon.stub(),
  };

  const modelProfiles = proxyquire(MODEL_PROFILES_PATH, {
    fs: fsMock,
  });

  const result = modelProfiles.setModelOverride('/fake/path', 'builder', 'glm-5');

  t.true(result.success);
  t.is(result.previous, 'kimi-k2.5');
});

test('setModelOverride throws ValidationError for invalid caste', t => {
  const profiles = createMockProfiles();
  const yamlContent = require('js-yaml').dump(profiles);

  const fsMock = {
    existsSync: sinon.stub().returns(true),
    readFileSync: sinon.stub().returns(yamlContent),
    writeFileSync: sinon.stub(),
  };

  const modelProfiles = proxyquire(MODEL_PROFILES_PATH, {
    fs: fsMock,
  });

  const error = t.throws(() => {
    modelProfiles.setModelOverride('/fake/path', 'invalid_caste', 'glm-5');
  });

  t.is(error.name, 'ValidationError');
  t.true(error.message.includes('Invalid caste'));
  t.true(error.details.validCastes.includes('builder'));
});

test('setModelOverride throws ValidationError for invalid model', t => {
  const profiles = createMockProfiles();
  const yamlContent = require('js-yaml').dump(profiles);

  const fsMock = {
    existsSync: sinon.stub().returns(true),
    readFileSync: sinon.stub().returns(yamlContent),
    writeFileSync: sinon.stub(),
  };

  const modelProfiles = proxyquire(MODEL_PROFILES_PATH, {
    fs: fsMock,
  });

  const error = t.throws(() => {
    modelProfiles.setModelOverride('/fake/path', 'builder', 'invalid_model');
  });

  t.is(error.name, 'ValidationError');
  t.true(error.message.includes('Invalid model'));
  t.true(error.details.validModels.includes('glm-5'));
});

test('setModelOverride creates user_overrides section if not exists', t => {
  const profiles = createMockProfiles();
  delete profiles.user_overrides;
  const yamlContent = require('js-yaml').dump(profiles);

  const fsMock = {
    existsSync: sinon.stub().returns(true),
    readFileSync: sinon.stub().returns(yamlContent),
    writeFileSync: sinon.stub(),
  };

  const modelProfiles = proxyquire(MODEL_PROFILES_PATH, {
    fs: fsMock,
  });

  modelProfiles.setModelOverride('/fake/path', 'builder', 'glm-5');

  const writtenContent = fsMock.writeFileSync.firstCall.args[1];
  const writtenData = require('js-yaml').load(writtenContent);

  t.truthy(writtenData.user_overrides);
  t.is(writtenData.user_overrides.builder, 'glm-5');
});

// ============================================
// resetModelOverride tests
// ============================================

test('resetModelOverride successfully removes override', t => {
  const profiles = createMockProfiles();
  profiles.user_overrides = { builder: 'glm-5', watcher: 'minimax-2.5' };
  const yamlContent = require('js-yaml').dump(profiles);

  const fsMock = {
    existsSync: sinon.stub().returns(true),
    readFileSync: sinon.stub().returns(yamlContent),
    writeFileSync: sinon.stub(),
  };

  const modelProfiles = proxyquire(MODEL_PROFILES_PATH, {
    fs: fsMock,
  });

  const result = modelProfiles.resetModelOverride('/fake/path', 'builder');

  t.true(result.success);
  t.true(result.hadOverride);
  t.true(fsMock.writeFileSync.calledOnce);

  // Verify the written content
  const writtenContent = fsMock.writeFileSync.firstCall.args[1];
  const writtenData = require('js-yaml').load(writtenContent);

  t.is(writtenData.user_overrides.builder, undefined);
  t.is(writtenData.user_overrides.watcher, 'minimax-2.5');
});

test('resetModelOverride returns hadOverride: false if no override existed', t => {
  const profiles = createMockProfiles();
  const yamlContent = require('js-yaml').dump(profiles);

  const fsMock = {
    existsSync: sinon.stub().returns(true),
    readFileSync: sinon.stub().returns(yamlContent),
    writeFileSync: sinon.stub(),
  };

  const modelProfiles = proxyquire(MODEL_PROFILES_PATH, {
    fs: fsMock,
  });

  const result = modelProfiles.resetModelOverride('/fake/path', 'builder');

  t.true(result.success);
  t.false(result.hadOverride);
  t.false(fsMock.writeFileSync.called);
});

test('resetModelOverride removes user_overrides section when empty', t => {
  const profiles = createMockProfiles();
  profiles.user_overrides = { builder: 'glm-5' };
  const yamlContent = require('js-yaml').dump(profiles);

  const fsMock = {
    existsSync: sinon.stub().returns(true),
    readFileSync: sinon.stub().returns(yamlContent),
    writeFileSync: sinon.stub(),
  };

  const modelProfiles = proxyquire(MODEL_PROFILES_PATH, {
    fs: fsMock,
  });

  modelProfiles.resetModelOverride('/fake/path', 'builder');

  const writtenContent = fsMock.writeFileSync.firstCall.args[1];
  const writtenData = require('js-yaml').load(writtenContent);

  t.is(writtenData.user_overrides, undefined);
});

test('resetModelOverride throws ValidationError for invalid caste', t => {
  const profiles = createMockProfiles();
  const yamlContent = require('js-yaml').dump(profiles);

  const fsMock = {
    existsSync: sinon.stub().returns(true),
    readFileSync: sinon.stub().returns(yamlContent),
    writeFileSync: sinon.stub(),
  };

  const modelProfiles = proxyquire(MODEL_PROFILES_PATH, {
    fs: fsMock,
  });

  const error = t.throws(() => {
    modelProfiles.resetModelOverride('/fake/path', 'invalid_caste');
  });

  t.is(error.name, 'ValidationError');
  t.true(error.message.includes('Invalid caste'));
});

// ============================================
// getEffectiveModel tests
// ============================================

test('getEffectiveModel returns override model when user_override exists', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = createMockProfiles();
  profiles.user_overrides = { builder: 'glm-5' };

  const result = modelProfiles.getEffectiveModel(profiles, 'builder');

  t.is(result.model, 'glm-5');
  t.is(result.source, 'override');
});

test('getEffectiveModel returns default model when no override', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = createMockProfiles();

  const result = modelProfiles.getEffectiveModel(profiles, 'builder');

  t.is(result.model, 'kimi-k2.5');
  t.is(result.source, 'default');
});

test('getEffectiveModel returns fallback when caste not found', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = createMockProfiles();

  const result = modelProfiles.getEffectiveModel(profiles, 'unknown_caste');

  t.is(result.model, modelProfiles.DEFAULT_MODEL);
  t.is(result.source, 'fallback');
});

test('getEffectiveModel returns fallback for null/undefined profiles', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);

  const result1 = modelProfiles.getEffectiveModel(null, 'builder');
  t.is(result1.model, modelProfiles.DEFAULT_MODEL);
  t.is(result1.source, 'fallback');

  const result2 = modelProfiles.getEffectiveModel(undefined, 'builder');
  t.is(result2.model, modelProfiles.DEFAULT_MODEL);
  t.is(result2.source, 'fallback');
});

test('getEffectiveModel override takes precedence over default', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = createMockProfiles();

  // Default is kimi-k2.5, override to glm-5
  profiles.user_overrides = { builder: 'glm-5' };

  const result = modelProfiles.getEffectiveModel(profiles, 'builder');

  t.is(result.model, 'glm-5');
  t.is(result.source, 'override');
  t.not(result.model, profiles.worker_models.builder);
});

// ============================================
// getUserOverrides tests
// ============================================

test('getUserOverrides returns empty object when no overrides', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = createMockProfiles();

  const result = modelProfiles.getUserOverrides(profiles);

  t.deepEqual(result, {});
});

test('getUserOverrides returns all overrides when present', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = createMockProfiles();
  profiles.user_overrides = {
    builder: 'glm-5',
    watcher: 'minimax-2.5',
  };

  const result = modelProfiles.getUserOverrides(profiles);

  t.is(result.builder, 'glm-5');
  t.is(result.watcher, 'minimax-2.5');
  t.is(Object.keys(result).length, 2);
});

test('getUserOverrides returns empty object for null/undefined profiles', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);

  t.deepEqual(modelProfiles.getUserOverrides(null), {});
  t.deepEqual(modelProfiles.getUserOverrides(undefined), {});
  t.deepEqual(modelProfiles.getUserOverrides('not an object'), {});
});

// ============================================
// Integration: Full override workflow
// ============================================

test('integration: full set/reset workflow', t => {
  const profiles = createMockProfiles();
  const yamlContent = require('js-yaml').dump(profiles);

  let currentContent = yamlContent;
  const fsMock = {
    existsSync: sinon.stub().returns(true),
    readFileSync: () => currentContent,
    writeFileSync: (path, content) => {
      currentContent = content;
    },
  };

  const modelProfiles = proxyquire(MODEL_PROFILES_PATH, {
    fs: fsMock,
  });

  // Step 1: Set override
  const setResult = modelProfiles.setModelOverride('/fake/path', 'builder', 'glm-5');
  t.true(setResult.success);
  t.is(setResult.previous, null);

  // Step 2: Verify effective model shows override
  let currentProfiles = require('js-yaml').load(currentContent);
  let effective = modelProfiles.getEffectiveModel(currentProfiles, 'builder');
  t.is(effective.model, 'glm-5');
  t.is(effective.source, 'override');

  // Step 3: Update override
  const updateResult = modelProfiles.setModelOverride('/fake/path', 'builder', 'minimax-2.5');
  t.true(updateResult.success);
  t.is(updateResult.previous, 'glm-5');

  // Step 4: Verify new effective model
  currentProfiles = require('js-yaml').load(currentContent);
  effective = modelProfiles.getEffectiveModel(currentProfiles, 'builder');
  t.is(effective.model, 'minimax-2.5');
  t.is(effective.source, 'override');

  // Step 5: Reset override
  const resetResult = modelProfiles.resetModelOverride('/fake/path', 'builder');
  t.true(resetResult.success);
  t.true(resetResult.hadOverride);

  // Step 6: Verify back to default
  currentProfiles = require('js-yaml').load(currentContent);
  effective = modelProfiles.getEffectiveModel(currentProfiles, 'builder');
  t.is(effective.model, 'kimi-k2.5');
  t.is(effective.source, 'default');
});
