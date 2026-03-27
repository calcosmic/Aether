#!/usr/bin/env node
/**
 * CLI Override Integration Tests
 *
 * Tests for the --model flag parsing and model-profile select/validate commands.
 */

const test = require('ava');
const path = require('path');
const fs = require('fs');
const os = require('os');
const { execSync } = require('child_process');

const { getModelNames } = require('../helpers/mock-profiles');

// Module-level constants derived from YAML via helper
const MODELS = getModelNames(); // ['glm-5', 'glm-5-turbo', 'glm-4.5-air']

// Test utilities
function createTempDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), 'aether-cli-test-'));
}

function cleanupTempDir(tempDir) {
  fs.rmSync(tempDir, { recursive: true, force: true });
}

function createMockModelProfiles(tempDir) {
  const aetherDir = path.join(tempDir, '.aether');
  fs.mkdirSync(aetherDir, { recursive: true });

  // Copy aether-utils.sh from the actual repo
  const repoUtilsPath = path.join(__dirname, '..', '..', '.aether', 'aether-utils.sh');
  const tempUtilsPath = path.join(aetherDir, 'aether-utils.sh');
  if (fs.existsSync(repoUtilsPath)) {
    fs.copyFileSync(repoUtilsPath, tempUtilsPath);
  }

  // Copy bin/lib for Node.js dependencies
  const repoBinPath = path.join(__dirname, '..', '..', 'bin');
  const tempBinPath = path.join(tempDir, 'bin');
  if (fs.existsSync(repoBinPath)) {
    fs.mkdirSync(tempBinPath, { recursive: true });
    const libFiles = fs.readdirSync(path.join(repoBinPath, 'lib'));
    fs.mkdirSync(path.join(tempBinPath, 'lib'), { recursive: true });
    for (const file of libFiles) {
      const src = path.join(repoBinPath, 'lib', file);
      const dest = path.join(tempBinPath, 'lib', file);
      if (fs.statSync(src).isFile()) {
        fs.copyFileSync(src, dest);
      }
    }
  }

  // Copy node_modules for dependencies
  const repoNodeModules = path.join(__dirname, '..', '..', 'node_modules');
  const tempNodeModules = path.join(tempDir, 'node_modules');
  if (fs.existsSync(repoNodeModules)) {
    // Create symlink for js-yaml and other deps
    fs.mkdirSync(tempNodeModules, { recursive: true });
    const deps = ['js-yaml', 'argparse', 'sprintf-js'];
    for (const dep of deps) {
      const src = path.join(repoNodeModules, dep);
      const dest = path.join(tempNodeModules, dep);
      if (fs.existsSync(src) && !fs.existsSync(dest)) {
        fs.cpSync(src, dest, { recursive: true });
      }
    }
  }

  // Use index-based access so YAML changes flow through automatically
  // MODELS = ['glm-5', 'glm-5-turbo', 'glm-4.5-air'] from YAML
  const yamlContent = `worker_models:
  prime: ${MODELS[0]}
  builder: ${MODELS[1]}
  watcher: ${MODELS[2]}
  scout: ${MODELS[1]}
  chaos: ${MODELS[2]}
  oracle: ${MODELS[0]}

model_metadata:
  ${MODELS[0]}:
    provider: openrouter
    description: Complex reasoning and architecture
  ${MODELS[1]}:
    provider: openrouter
    description: Fast implementation and coding
  ${MODELS[2]}:
    provider: openrouter
    description: Validation and research

task_routing:
  default_model: ${MODELS[1]}
  complexity_indicators:
    complex:
      keywords: ["design", "architect", "plan", "complex", "refactor"]
      model: ${MODELS[0]}
    simple:
      keywords: ["implement", "code", "write", "create", "build"]
      model: ${MODELS[1]}
    validate:
      keywords: ["test", "verify", "check", "validate", "review"]
      model: ${MODELS[2]}
`;

  fs.writeFileSync(path.join(aetherDir, 'model-profiles.yaml'), yamlContent);
}

// ============================================
// Tests for model-profile select command
// ============================================

test('model-profile select returns task-routing default when no keyword match', t => {
  const tempDir = createTempDir();
  createMockModelProfiles(tempDir);

  try {
    // Use a task description that doesn't match any keywords
    // Should fall back to default_model with source 'task-routing'
    const result = execSync(
      `bash .aether/aether-utils.sh model-profile select builder "unknown task xyz" ""`,
      { cwd: tempDir, encoding: 'utf8' }
    );

    const parsed = JSON.parse(result);
    t.true(parsed.ok, 'Result should be ok');
    t.is(parsed.result.model, MODELS[1], 'Should return default model from task routing');
    t.is(parsed.result.source, 'task-routing', 'Source should be task-routing when default_model is used as catch-all');
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('model-profile select returns CLI override when provided', t => {
  const tempDir = createTempDir();
  createMockModelProfiles(tempDir);

  try {
    const result = execSync(
      `bash .aether/aether-utils.sh model-profile select builder "implement feature" "${MODELS[0]}"`,
      { cwd: tempDir, encoding: 'utf8' }
    );

    const parsed = JSON.parse(result);
    t.true(parsed.ok, 'Result should be ok');
    t.is(parsed.result.model, MODELS[0], 'Should return CLI override model');
    t.is(parsed.result.source, 'cli-override', 'Source should be cli-override');
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('model-profile select returns task-routing model when no CLI override', t => {
  const tempDir = createTempDir();
  createMockModelProfiles(tempDir);

  try {
    // Task description contains "design" which maps to glm-5
    const result = execSync(
      `bash .aether/aether-utils.sh model-profile select builder "design system" ""`,
      { cwd: tempDir, encoding: 'utf8' }
    );

    const parsed = JSON.parse(result);
    t.true(parsed.ok, 'Result should be ok');
    t.is(parsed.result.model, MODELS[0], `Should return task-routed model (design -> ${MODELS[0]})`);
    t.is(parsed.result.source, 'task-routing', 'Source should be task-routing');
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('model-profile select returns user override when no CLI override', t => {
  const tempDir = createTempDir();
  createMockModelProfiles(tempDir);

  // Add user override
  const profilePath = path.join(tempDir, '.aether', 'model-profiles.yaml');
  let content = fs.readFileSync(profilePath, 'utf8');
  content += `\nuser_overrides:\n  builder: ${MODELS[0]}\n`;
  fs.writeFileSync(profilePath, content);

  try {
    const result = execSync(
      `bash .aether/aether-utils.sh model-profile select builder "implement feature" ""`,
      { cwd: tempDir, encoding: 'utf8' }
    );

    const parsed = JSON.parse(result);
    t.true(parsed.ok, 'Result should be ok');
    t.is(parsed.result.model, MODELS[0], 'Should return user override model');
    t.is(parsed.result.source, 'user-override', 'Source should be user-override');
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('model-profile select CLI override takes precedence over user override', t => {
  const tempDir = createTempDir();
  createMockModelProfiles(tempDir);

  // Add user override
  const profilePath = path.join(tempDir, '.aether', 'model-profiles.yaml');
  let content = fs.readFileSync(profilePath, 'utf8');
  content += `\nuser_overrides:\n  builder: ${MODELS[0]}\n`;
  fs.writeFileSync(profilePath, content);

  try {
    // CLI override should win over user override
    const result = execSync(
      `bash .aether/aether-utils.sh model-profile select builder "implement feature" "${MODELS[2]}"`,
      { cwd: tempDir, encoding: 'utf8' }
    );

    const parsed = JSON.parse(result);
    t.true(parsed.ok, 'Result should be ok');
    t.is(parsed.result.model, MODELS[2], 'CLI override should take precedence');
    t.is(parsed.result.source, 'cli-override', 'Source should be cli-override');
  } finally {
    cleanupTempDir(tempDir);
  }
});

// ============================================
// Tests for model-profile validate command
// ============================================

test('model-profile validate returns valid:true for known models', t => {
  const tempDir = createTempDir();
  createMockModelProfiles(tempDir);

  try {
    for (const model of MODELS) {
      const result = execSync(
        `bash .aether/aether-utils.sh model-profile validate ${model}`,
        { cwd: tempDir, encoding: 'utf8' }
      );

      const parsed = JSON.parse(result);
      t.true(parsed.ok, `Result should be ok for ${model}`);
      t.true(parsed.result.valid, `${model} should be valid`);
    }
  } finally {
    cleanupTempDir(tempDir);
  }
});

test('model-profile validate returns valid:false for unknown models', t => {
  const tempDir = createTempDir();
  createMockModelProfiles(tempDir);

  try {
    const result = execSync(
      `bash .aether/aether-utils.sh model-profile validate unknown-model`,
      { cwd: tempDir, encoding: 'utf8' }
    );

    const parsed = JSON.parse(result);
    t.true(parsed.ok, 'Result should be ok');
    t.false(parsed.result.valid, 'Unknown model should be invalid');
    t.true(Array.isArray(parsed.result.models), 'Should return list of valid models');
    t.true(parsed.result.models.includes(MODELS[0]), `Valid models should include ${MODELS[0]}`);
  } finally {
    cleanupTempDir(tempDir);
  }
});

// ============================================
// Argument parsing pattern tests
// ============================================

test('argument parsing: phase only', t => {
  // Simulate parsing "1"
  const args = '1';
  const parts = args.trim().split(/\s+/);
  const phase = parts[0];
  let cli_model_override = '';

  for (let i = 1; i < parts.length; i++) {
    if ((parts[i] === '--model' || parts[i] === '-m') && parts[i + 1]) {
      cli_model_override = parts[i + 1];
      i++;
    }
  }

  t.is(phase, '1');
  t.is(cli_model_override, '');
});

test('argument parsing: phase with --model flag', t => {
  // Simulate parsing "1 --model {MODELS[0]}"
  const args = `1 --model ${MODELS[0]}`;
  const parts = args.trim().split(/\s+/);
  const phase = parts[0];
  let cli_model_override = '';

  for (let i = 1; i < parts.length; i++) {
    if ((parts[i] === '--model' || parts[i] === '-m') && parts[i + 1]) {
      cli_model_override = parts[i + 1];
      i++;
    }
  }

  t.is(phase, '1');
  t.is(cli_model_override, MODELS[0]);
});

test('argument parsing: phase with -m short flag', t => {
  // Simulate parsing "1 -m {MODELS[0]}"
  const args = `1 -m ${MODELS[0]}`;
  const parts = args.trim().split(/\s+/);
  const phase = parts[0];
  let cli_model_override = '';

  for (let i = 1; i < parts.length; i++) {
    if ((parts[i] === '--model' || parts[i] === '-m') && parts[i + 1]) {
      cli_model_override = parts[i + 1];
      i++;
    }
  }

  t.is(phase, '1');
  t.is(cli_model_override, MODELS[0]);
});

test('argument parsing: phase with verbose and model flags', t => {
  // Simulate parsing "1 --verbose --model {MODELS[0]}"
  const args = `1 --verbose --model ${MODELS[0]}`;
  const parts = args.trim().split(/\s+/);
  const phase = parts[0];
  let verbose_mode = false;
  let cli_model_override = '';

  for (let i = 1; i < parts.length; i++) {
    if (parts[i] === '--verbose' || parts[i] === '-v') {
      verbose_mode = true;
    }
    if ((parts[i] === '--model' || parts[i] === '-m') && parts[i + 1]) {
      cli_model_override = parts[i + 1];
      i++;
    }
  }

  t.is(phase, '1');
  t.true(verbose_mode);
  t.is(cli_model_override, MODELS[0]);
});

// ============================================
// Integration tests
// ============================================

test('integration: end-to-end model selection with all override types', t => {
  const tempDir = createTempDir();
  createMockModelProfiles(tempDir);

  // Add user override
  const profilePath = path.join(tempDir, '.aether', 'model-profiles.yaml');
  let content = fs.readFileSync(profilePath, 'utf8');
  content += `\nuser_overrides:\n  watcher: ${MODELS[0]}\n`;
  fs.writeFileSync(profilePath, content);

  try {
    // Test 1: Task routing default (use task that doesn't match any keywords)
    let result = execSync(
      `bash .aether/aether-utils.sh model-profile select scout "xyz abc" ""`,
      { cwd: tempDir, encoding: 'utf8' }
    );
    let parsed = JSON.parse(result);
    t.is(parsed.result.source, 'task-routing', 'Scout should use task routing default when no keyword match');

    // Test 2: Task routing
    result = execSync(
      `bash .aether/aether-utils.sh model-profile select builder "design architecture" ""`,
      { cwd: tempDir, encoding: 'utf8' }
    );
    parsed = JSON.parse(result);
    t.is(parsed.result.source, 'task-routing', 'Should use task routing for design tasks');
    t.is(parsed.result.model, MODELS[0], `Design tasks should route to ${MODELS[0]}`);

    // Test 3: User override
    result = execSync(
      `bash .aether/aether-utils.sh model-profile select watcher "verify code" ""`,
      { cwd: tempDir, encoding: 'utf8' }
    );
    parsed = JSON.parse(result);
    t.is(parsed.result.source, 'user-override', 'Watcher should use user override');
    t.is(parsed.result.model, MODELS[0], `User override should be ${MODELS[0]}`);

    // Test 4: CLI override
    result = execSync(
      `bash .aether/aether-utils.sh model-profile select builder "any task" "${MODELS[2]}"`,
      { cwd: tempDir, encoding: 'utf8' }
    );
    parsed = JSON.parse(result);
    t.is(parsed.result.source, 'cli-override', 'CLI override should take precedence');
    t.is(parsed.result.model, MODELS[2], `CLI override should be ${MODELS[2]}`);

  } finally {
    cleanupTempDir(tempDir);
  }
});

test('integration: verify JSON output structure', t => {
  const tempDir = createTempDir();
  createMockModelProfiles(tempDir);

  try {
    const result = execSync(
      `bash .aether/aether-utils.sh model-profile select builder "test" ""`,
      { cwd: tempDir, encoding: 'utf8' }
    );

    const parsed = JSON.parse(result);

    // Verify structure
    t.true(parsed.hasOwnProperty('ok'), 'Should have ok property');
    t.true(parsed.hasOwnProperty('result'), 'Should have result property');
    t.true(parsed.result.hasOwnProperty('model'), 'Result should have model property');
    t.true(parsed.result.hasOwnProperty('source'), 'Result should have source property');

    // Verify types
    t.is(typeof parsed.ok, 'boolean', 'ok should be boolean');
    t.is(typeof parsed.result.model, 'string', 'model should be string');
    t.is(typeof parsed.result.source, 'string', 'source should be string');

    // Verify source values
    const validSources = ['cli-override', 'user-override', 'task-routing', 'caste-default', 'fallback'];
    t.true(validSources.includes(parsed.result.source), `Source should be one of ${validSources.join(', ')}`);

  } finally {
    cleanupTempDir(tempDir);
  }
});
