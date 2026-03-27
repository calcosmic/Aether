const test = require('ava');
const path = require('path');

const MODEL_PROFILES_PATH = path.join(__dirname, '../../bin/lib/model-profiles.js');
const {
  buildMockProfiles,
  getDefaultModelForCaste,
  getModelNames,
} = require('../helpers/mock-profiles');

/**
 * Custom task routing config that assigns different models per complexity tier
 * to test routing logic. Actual YAML has all routes pointing to the same model,
 * so we override task_routing to ensure diversity.
 */
const CUSTOM_TASK_ROUTING = {
  default_model: getDefaultModelForCaste('builder'),
  complexity_indicators: {
    complex: {
      keywords: ['design', 'architecture', 'plan', 'coordinate', 'synthesize', 'strategize', 'optimize'],
      model: getModelNames()[0],  // 'glm-5' from YAML metadata keys
    },
    simple: {
      keywords: ['implement', 'code', 'refactor', 'write', 'create'],
      model: getDefaultModelForCaste('builder'),  // 'glm-5-turbo'
    },
    validate: {
      keywords: ['test', 'validate', 'verify', 'check', 'review', 'audit'],
      model: getModelNames()[2],  // 'glm-4.5-air'
    },
  },
};

// ============================================
// getModelForTask tests
// ============================================

test('getModelForTask returns null when taskRouting is null', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const result = modelProfiles.getModelForTask(null, 'Design new system');
  t.is(result, null);
});

test('getModelForTask returns null when taskRouting is undefined', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const result = modelProfiles.getModelForTask(undefined, 'Design new system');
  t.is(result, null);
});

test('getModelForTask returns null when taskDescription is empty', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles({ taskRouting: CUSTOM_TASK_ROUTING });
  const result = modelProfiles.getModelForTask(profiles.task_routing, '');
  t.is(result, null);
});

test('getModelForTask returns null when taskDescription is null', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles({ taskRouting: CUSTOM_TASK_ROUTING });
  const result = modelProfiles.getModelForTask(profiles.task_routing, null);
  t.is(result, null);
});

test('getModelForTask matches "design" keyword and returns glm-5', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles({ taskRouting: CUSTOM_TASK_ROUTING });
  const result = modelProfiles.getModelForTask(profiles.task_routing, 'Design new system');
  t.is(result, getModelNames()[0]);
});

test('getModelForTask matches "implement" keyword and returns glm-5-turbo', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles({ taskRouting: CUSTOM_TASK_ROUTING });
  const result = modelProfiles.getModelForTask(profiles.task_routing, 'Implement new feature');
  t.is(result, getDefaultModelForCaste('builder'));
});

test('getModelForTask matches "test" keyword and returns glm-4.5-air', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles({ taskRouting: CUSTOM_TASK_ROUTING });
  const result = modelProfiles.getModelForTask(profiles.task_routing, 'Test the validation logic');
  t.is(result, getModelNames()[2]);
});

test('getModelForTask returns default_model when no keywords match', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles({ taskRouting: CUSTOM_TASK_ROUTING });
  const result = modelProfiles.getModelForTask(profiles.task_routing, 'Something completely unrelated');
  t.is(result, getDefaultModelForCaste('builder'));
});

test('getModelForTask returns null when no keywords match and no default_model', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const taskRouting = {
    complexity_indicators: {
      complex: { keywords: ['design'], model: getModelNames()[0] },
    },
    // no default_model
  };
  const result = modelProfiles.getModelForTask(taskRouting, 'Something unrelated');
  t.is(result, null);
});

test('getModelForTask performs case-insensitive matching', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles({ taskRouting: CUSTOM_TASK_ROUTING });

  // Uppercase DESIGN should match
  const result1 = modelProfiles.getModelForTask(profiles.task_routing, 'DESIGN new system');
  t.is(result1, getModelNames()[0]);

  // Mixed case DeSiGn should match
  const result2 = modelProfiles.getModelForTask(profiles.task_routing, 'DeSiGn new system');
  t.is(result2, getModelNames()[0]);
});

test('getModelForTask performs substring matching', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles({ taskRouting: CUSTOM_TASK_ROUTING });

  // "redesign" contains "design" as substring
  const result = modelProfiles.getModelForTask(profiles.task_routing, 'Redesign the interface');
  t.is(result, getModelNames()[0]);

  // "implementation" contains "implement" as substring
  const result2 = modelProfiles.getModelForTask(profiles.task_routing, 'Implementation details');
  t.is(result2, getDefaultModelForCaste('builder'));

  // "testing" contains "test" as substring (but not other keywords from earlier categories)
  const result3 = modelProfiles.getModelForTask(profiles.task_routing, 'Testing the functionality');
  t.is(result3, getModelNames()[2]);
});

test('getModelForTask returns first matching complexity indicator', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles({ taskRouting: CUSTOM_TASK_ROUTING });

  // "design" is listed before "implement" in complexity_indicators order
  // If task contains both, complex (design) should win since it's first in iteration
  // But actually the order depends on Object.entries() which follows insertion order
  const result = modelProfiles.getModelForTask(profiles.task_routing, 'Design and implement');
  // complex indicator with 'design' comes first in YAML order
  t.is(result, getModelNames()[0]);
});

test('getModelForTask handles missing complexity_indicators gracefully', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const taskRouting = {
    default_model: getDefaultModelForCaste('builder'),
    // no complexity_indicators
  };
  const result = modelProfiles.getModelForTask(taskRouting, 'Design something');
  t.is(result, getDefaultModelForCaste('builder'));
});

test('getModelForTask handles empty complexity_indicators', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const taskRouting = {
    default_model: getDefaultModelForCaste('builder'),
    complexity_indicators: {},
  };
  const result = modelProfiles.getModelForTask(taskRouting, 'Design something');
  t.is(result, getDefaultModelForCaste('builder'));
});


// ============================================
// selectModelForTask tests
// ============================================

test('selectModelForTask CLI override takes precedence over everything', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles({ taskRouting: CUSTOM_TASK_ROUTING });

  // Even with user override and task routing matching, CLI should win
  profiles.user_overrides = { builder: getModelNames()[0] };

  const result = modelProfiles.selectModelForTask(profiles, 'builder', 'Design new system', getModelNames()[2]);
  t.is(result.model, getModelNames()[2]);
  t.is(result.source, 'cli-override');
});

test('selectModelForTask CLI override validates model before using', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles({ taskRouting: CUSTOM_TASK_ROUTING });

  // Invalid CLI override should fall through to next precedence level
  const result = modelProfiles.selectModelForTask(profiles, 'builder', 'Design new system', 'invalid-model');
  // Should fall through to task routing since CLI was invalid
  t.is(result.model, getModelNames()[0]);
  t.is(result.source, 'task-routing');
});

test('selectModelForTask user override takes precedence over task routing', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles({ taskRouting: CUSTOM_TASK_ROUTING });

  profiles.user_overrides = { builder: getModelNames()[2] };

  // Even though "Design" would route to glm-5, user override should win
  const result = modelProfiles.selectModelForTask(profiles, 'builder', 'Design new system');
  t.is(result.model, getModelNames()[2]);
  t.is(result.source, 'user-override');
});

test('selectModelForTask task routing takes precedence over caste default', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles({ taskRouting: CUSTOM_TASK_ROUTING });

  // No user override, but task contains "design"
  const result = modelProfiles.selectModelForTask(profiles, 'builder', 'Design new system');
  // builder default is glm-5-turbo, but "design" routes to glm-5
  t.is(result.model, getModelNames()[0]);
  t.is(result.source, 'task-routing');
});

test('selectModelForTask task routing default is used when no keyword matches', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles({ taskRouting: CUSTOM_TASK_ROUTING });

  // No user override, no keyword match - should use task_routing.default_model
  const result = modelProfiles.selectModelForTask(profiles, 'builder', 'Something unrelated');
  t.is(result.model, getDefaultModelForCaste('builder'));
  t.is(result.source, 'task-routing');
});

test('selectModelForTask caste default is used when no task_routing config', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles({ taskRouting: CUSTOM_TASK_ROUTING });
  delete profiles.task_routing;

  // No task routing config - should fall back to caste default
  const result = modelProfiles.selectModelForTask(profiles, 'builder', 'Something unrelated');
  t.is(result.model, getDefaultModelForCaste('builder'));
  t.is(result.source, 'caste-default');
});

test('selectModelForTask fallback is used when nothing matches', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);

  // Empty profiles - no worker_models, no task_routing, no user_overrides
  const emptyProfiles = {
    model_metadata: {
      [getDefaultModelForCaste('builder')]: {},
    },
  };

  const result = modelProfiles.selectModelForTask(emptyProfiles, 'unknown-caste', 'Something');
  t.is(result.model, getDefaultModelForCaste('builder')); // DEFAULT_MODEL
  t.is(result.source, 'fallback');
});

test('selectModelForTask returns correct source for each precedence level', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles({ taskRouting: CUSTOM_TASK_ROUTING });

  // Test CLI override source
  const cliResult = modelProfiles.selectModelForTask(profiles, 'builder', 'Design', getModelNames()[0]);
  t.is(cliResult.source, 'cli-override');

  // Test user override source
  profiles.user_overrides = { builder: getModelNames()[0] };
  const userResult = modelProfiles.selectModelForTask(profiles, 'builder', 'Implement');
  t.is(userResult.source, 'user-override');
  delete profiles.user_overrides;

  // Test task routing source (keyword match)
  const taskResult = modelProfiles.selectModelForTask(profiles, 'builder', 'Design');
  t.is(taskResult.source, 'task-routing');

  // Test task routing default source (no keyword match, but default_model exists)
  const taskDefaultResult = modelProfiles.selectModelForTask(profiles, 'builder', 'Something');
  t.is(taskDefaultResult.source, 'task-routing');

  // Test caste default source (no task_routing config)
  const profilesNoRouting = buildMockProfiles({ taskRouting: CUSTOM_TASK_ROUTING });
  delete profilesNoRouting.task_routing;
  const casteResult = modelProfiles.selectModelForTask(profilesNoRouting, 'builder', 'Something');
  t.is(casteResult.source, 'caste-default');

  // Test fallback source
  const fallbackResult = modelProfiles.selectModelForTask({}, 'unknown', 'Something');
  t.is(fallbackResult.source, 'fallback');
});

test('selectModelForTask handles null profiles gracefully', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);

  const result = modelProfiles.selectModelForTask(null, 'builder', 'Design');
  t.is(result.model, getDefaultModelForCaste('builder'));
  t.is(result.source, 'fallback');
});

test('selectModelForTask handles undefined profiles gracefully', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);

  const result = modelProfiles.selectModelForTask(undefined, 'builder', 'Design');
  t.is(result.model, getDefaultModelForCaste('builder'));
  t.is(result.source, 'fallback');
});

test('selectModelForTask handles null taskDescription gracefully', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles({ taskRouting: CUSTOM_TASK_ROUTING });

  // Null taskDescription should skip task routing and go to caste default
  const result = modelProfiles.selectModelForTask(profiles, 'builder', null);
  t.is(result.model, getDefaultModelForCaste('builder'));
  t.is(result.source, 'caste-default');
});

test('selectModelForTask handles empty string taskDescription gracefully', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles({ taskRouting: CUSTOM_TASK_ROUTING });

  // Empty taskDescription should skip task routing (getModelForTask returns null)
  // and go to caste default
  const result = modelProfiles.selectModelForTask(profiles, 'builder', '');
  t.is(result.model, getDefaultModelForCaste('builder'));
  t.is(result.source, 'caste-default');
});

test('selectModelForTask works with different castes', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles({ taskRouting: CUSTOM_TASK_ROUTING });

  // Remove task_routing to test caste defaults directly
  delete profiles.task_routing;

  // architect default is glm-5-turbo
  const architectResult = modelProfiles.selectModelForTask(profiles, 'architect', 'Something');
  t.is(architectResult.model, getDefaultModelForCaste('builder'));
  t.is(architectResult.source, 'caste-default');

  // oracle default is glm-5-turbo
  const oracleResult = modelProfiles.selectModelForTask(profiles, 'oracle', 'Something');
  t.is(oracleResult.model, getDefaultModelForCaste('builder'));
  t.is(oracleResult.source, 'caste-default');

  // builder default is glm-5-turbo
  const builderResult = modelProfiles.selectModelForTask(profiles, 'builder', 'Something');
  t.is(builderResult.model, getDefaultModelForCaste('builder'));
  t.is(builderResult.source, 'caste-default');
});

test('selectModelForTask task routing works for all complexity levels', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles({ taskRouting: CUSTOM_TASK_ROUTING });

  // Complex tasks -> glm-5
  const complexResult = modelProfiles.selectModelForTask(profiles, 'builder', 'Design the architecture');
  t.is(complexResult.model, getModelNames()[0]);
  t.is(complexResult.source, 'task-routing');

  // Simple tasks -> glm-5-turbo
  const simpleResult = modelProfiles.selectModelForTask(profiles, 'builder', 'Implement the function');
  t.is(simpleResult.model, getDefaultModelForCaste('builder'));
  t.is(simpleResult.source, 'task-routing');

  // Validate tasks -> glm-4.5-air
  const validateResult = modelProfiles.selectModelForTask(profiles, 'builder', 'Test the validation');
  t.is(validateResult.model, getModelNames()[2]);
  t.is(validateResult.source, 'task-routing');
});

test('selectModelForTask cliOverride defaults to null', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = buildMockProfiles({ taskRouting: CUSTOM_TASK_ROUTING });

  // Call without cliOverride parameter - should work without error
  const result = modelProfiles.selectModelForTask(profiles, 'builder', 'Design');
  t.is(result.source, 'task-routing');
});
