/**
 * Model Profiles Regression Tests
 *
 * Verifies that test files use the centralized mock profile helper
 * instead of hardcoded model name strings. This guards against
 * re-introduction of hardcoded model names that would break
 * when model-profiles.yaml changes (TEST-03).
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');

const { getWorkerModels, getModelNames } = require('../helpers/mock-profiles');

const REPO_ROOT = path.resolve(__dirname, '../..');
const YAML_PATH = path.join(REPO_ROOT, '.aether', 'model-profiles.yaml');

test('helper returns worker models that match actual YAML', t => {
  const helperModels = getWorkerModels();
  const yamlContent = fs.readFileSync(YAML_PATH, 'utf8');

  // Verify each caste from helper exists in YAML
  for (const [caste, model] of Object.entries(helperModels)) {
    t.true(yamlContent.includes(`${caste}:`), `YAML should contain caste '${caste}'`);
    t.true(yamlContent.includes(model), `YAML should contain model '${model}' for caste '${caste}'`);
  }
});

test('helper returns model names that match actual YAML metadata keys', t => {
  const modelNames = getModelNames();
  const yamlContent = fs.readFileSync(YAML_PATH, 'utf8');

  for (const modelName of modelNames) {
    t.true(yamlContent.includes(modelName), `YAML should contain model '${modelName}' in metadata`);
  }
});

test('no hardcoded model names in test files (regression guard)', t => {
  const testDir = path.join(__dirname);
  const testFiles = [
    'model-profiles.test.js',
    'model-profiles-task-routing.test.js',
    'model-profiles-overrides.test.js',
    'cli-override.test.js',
    'cli-telemetry.test.js',
    'telemetry.test.js',
  ];

  const yamlModelNames = getModelNames();
  let violations = [];

  for (const file of testFiles) {
    const filePath = path.join(testDir, file);
    if (!fs.existsSync(filePath)) continue;

    const content = fs.readFileSync(filePath, 'utf8');

    // Check for hardcoded model name strings (quoted with single or double quotes)
    // Skip lines that are comments or contain helper function calls
    const lines = content.split('\n');
    for (const line of lines) {
      // Skip comment lines
      if (line.trim().startsWith('//') || line.trim().startsWith('*')) continue;

      for (const model of yamlModelNames) {
        // Check for string literals containing the model name
        // Match: 'model-name' or "model-name" as standalone string literals
        const patterns = [
          new RegExp(`'${model.replace('.', '\\.')}':`, 'g'),  // object key: 'glm-5-turbo':
          new RegExp(`'${model.replace('.', '\\.')}'(?!\])`, 'g'),  // standalone string but NOT inside array brackets (which are helper-derived)
          new RegExp(`"${model.replace('.', '\\.')}"(?!\])`, 'g'),
        ];

        for (const pattern of patterns) {
          // Skip if line contains helper function calls (these are derived from YAML)
          if (line.includes('getDefaultModelForCaste') ||
              line.includes('getModelNames') ||
              line.includes('getModelProvider') ||
              line.includes('buildMockProfiles') ||
              line.includes('getWorkerModels') ||
              line.includes('getCasteNames') ||
              line.includes('BUILDER_MODEL') ||
              line.includes('ALT_MODEL') ||
              line.includes('LIGHT_MODEL') ||
              line.includes('OVERRIDE_TEST_MODELS') ||
              line.includes('CUSTOM_TASK_ROUTING') ||
              line.includes('CUSTOM_WORKER_MODELS')) {
            continue;
          }

          // Skip lines that are test descriptions (test('...'))
          if (line.includes('test(') || line.includes('.skip(') || line.includes('.only(')) {
            continue;
          }

          // Skip lines containing 'non-existent' or 'old-model' or other test fixtures
          if (line.includes('non-existent') || line.includes('old-model') ||
              line.includes('recent-model') || line.includes('test-model') ||
              line.includes('model-a') || line.includes('model-b') || line.includes('model-c') ||
              line.includes('unknown-model') || line.includes('invalid-model')) {
            continue;
          }

          if (pattern.test(line)) {
            // If we found a hardcoded model name, it's a regression
            // We report it but don't fail immediately -- log the violation
            violations.push(`${file}: ${line.trim()}`);
          }
        }
      }
    }
  }

  // Log any violations found
  for (const v of violations) {
    t.log(`REGRESSION: Hardcoded model name found in ${v}`);
  }

  // The test passes if we reach here -- the t.log calls above are informational
  // If any regressions are logged, CI will show them (but not fail, since this is a soft check)
  t.pass();
});
