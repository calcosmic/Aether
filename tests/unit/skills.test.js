const test = require('ava');
const { execSync } = require('child_process');
const path = require('path');
const fs = require('fs');
const os = require('os');

const AETHER_ROOT = path.join(__dirname, '../..');
const AETHER_UTILS = path.join(AETHER_ROOT, '.aether/aether-utils.sh');

/**
 * Helper to run aether-utils.sh commands and parse JSON output
 */
function runSkillCommand(command, args = [], env = {}) {
  const cmd = `bash "${AETHER_UTILS}" ${command} ${args.map(a => `"${a}"`).join(' ')}`;
  const output = execSync(cmd, {
    cwd: AETHER_ROOT,
    encoding: 'utf8',
    timeout: 15000,
    env: { ...process.env, ...env }
  });
  return JSON.parse(output);
}

/**
 * Helper to create an isolated skills test environment
 * Returns { tempDir, skillsDir } with sample SKILL.md files
 */
function createSkillsEnv() {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'skills-test-'));
  const skillsDir = path.join(tempDir, 'skills');

  // Create colony skill directory
  fs.mkdirSync(path.join(skillsDir, 'colony', 'test-skill'), { recursive: true });
  fs.writeFileSync(
    path.join(skillsDir, 'colony', 'test-skill', 'SKILL.md'),
    [
      '---',
      'name: test-skill',
      'description: Use when testing the skills system',
      'type: colony',
      'domains: [testing, quality]',
      'agent_roles: [builder, watcher]',
      'priority: normal',
      'version: "1.0"',
      '---',
      '',
      'Test skill content here.'
    ].join('\n')
  );

  // Create domain skill directory with detection patterns
  fs.mkdirSync(path.join(skillsDir, 'domain', 'test-domain'), { recursive: true });
  fs.writeFileSync(
    path.join(skillsDir, 'domain', 'test-domain', 'SKILL.md'),
    [
      '---',
      'name: test-domain',
      'description: Use when working with test frameworks',
      'type: domain',
      'domains: [testing, frontend]',
      'agent_roles: [builder]',
      'detect_files: ["*.test.js", "jest.config.*"]',
      'detect_packages: ["jest", "vitest"]',
      'priority: normal',
      'version: "1.0"',
      '---',
      '',
      'Domain skill content for testing.'
    ].join('\n')
  );

  return { tempDir, skillsDir };
}

/**
 * Helper to clean up temp directories
 */
function cleanupEnv(tempDir) {
  fs.rmSync(tempDir, { recursive: true, force: true });
}

// ==========================================================================
// Test 1: skill-list returns valid JSON with skill_count field
// ==========================================================================
test('skill-list returns valid JSON with skill_count field', t => {
  const { tempDir, skillsDir } = createSkillsEnv();

  try {
    // Build the index first so skill-list has data
    runSkillCommand('skill-index', [skillsDir], { AETHER_SKILLS_DIR: skillsDir });

    // Now call skill-list
    const result = runSkillCommand('skill-list', [skillsDir], { AETHER_SKILLS_DIR: skillsDir });

    t.true(result.ok, 'Should return ok: true');
    t.truthy(result.result, 'Should have result');
    t.is(typeof result.result.skill_count, 'number', 'skill_count should be a number');
    t.is(result.result.skill_count, 2, 'Should find 2 skills (colony + domain)');
    t.true(Array.isArray(result.result.skills), 'skills should be an array');
  } finally {
    cleanupEnv(tempDir);
  }
});

// ==========================================================================
// Test 2: skill-parse-frontmatter extracts name field from a SKILL.md
// ==========================================================================
test('skill-parse-frontmatter extracts name field from SKILL.md', t => {
  const { tempDir, skillsDir } = createSkillsEnv();
  const skillFile = path.join(skillsDir, 'colony', 'test-skill', 'SKILL.md');

  try {
    const result = runSkillCommand('skill-parse-frontmatter', [skillFile]);

    t.true(result.ok, 'Should return ok: true');
    t.truthy(result.result, 'Should have result');
    t.is(result.result.name, 'test-skill', 'Should extract name field');
    t.is(result.result.type, 'colony', 'Should extract type field');
    t.is(result.result.description, 'Use when testing the skills system', 'Should extract description');
    t.is(result.result.priority, 'normal', 'Should extract priority');
    t.true(Array.isArray(result.result.domains), 'domains should be an array');
    t.is(result.result.domains.length, 2, 'Should have 2 domains');
  } finally {
    cleanupEnv(tempDir);
  }
});

// ==========================================================================
// Test 3: skill-match returns colony_skills and domain_skills arrays
// ==========================================================================
test('skill-match returns colony_skills and domain_skills arrays', t => {
  const { tempDir, skillsDir } = createSkillsEnv();

  try {
    // Build the index first
    runSkillCommand('skill-index', [skillsDir], { AETHER_SKILLS_DIR: skillsDir });

    // Match as "builder" role with task description containing "testing" to boost domain score
    // Domain skills start at score 0 and need domain overlap to reach minimum threshold (20)
    const result = runSkillCommand('skill-match', ['builder', 'testing frontend work', skillsDir], {
      AETHER_SKILLS_DIR: skillsDir
    });

    t.true(result.ok, 'Should return ok: true');
    t.truthy(result.result, 'Should have result');
    t.true(Array.isArray(result.result.colony_skills), 'colony_skills should be an array');
    t.true(Array.isArray(result.result.domain_skills), 'domain_skills should be an array');
    t.true(result.result.colony_skills.length >= 1, 'Builder should match at least 1 colony skill');
    t.true(result.result.domain_skills.length >= 1, 'Builder should match at least 1 domain skill (with task context)');

    // Verify the matched colony skill has expected fields
    const colonySkill = result.result.colony_skills[0];
    t.truthy(colonySkill.name, 'Colony skill should have name');
    t.is(typeof colonySkill.match_score, 'number', 'Colony skill should have match_score');
  } finally {
    cleanupEnv(tempDir);
  }
});

// ==========================================================================
// Test 4: skill-parse-frontmatter returns error for missing file
// ==========================================================================
test('skill-parse-frontmatter returns error for missing file', t => {
  // The command exits non-zero for missing files, so execSync throws.
  // Parse the error JSON from stderr to verify the structured error response.
  const cmd = `bash "${AETHER_UTILS}" skill-parse-frontmatter "/nonexistent/SKILL.md"`;
  try {
    execSync(cmd, { cwd: AETHER_ROOT, encoding: 'utf8', timeout: 15000 });
    t.fail('Should have thrown for missing file');
  } catch (err) {
    const errorOutput = err.stderr || err.stdout || '';
    const result = JSON.parse(errorOutput.trim());
    t.is(result.ok, false, 'Should return ok: false for missing file');
    t.truthy(result.error, 'Should have error field');
    t.is(result.error.code, 'E_FILE_NOT_FOUND', 'Error code should be E_FILE_NOT_FOUND');
  }
});

// ==========================================================================
// Test 5a: skill-match selects at most 2 colony + 2 domain skills
// ==========================================================================
test('skill-match selects at most 2 colony + 2 domain skills', t => {
  // Create environment with 4 colony + 4 domain skills to test the limit
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'skills-limit-'));
  const skillsDir = path.join(tempDir, 'skills');

  const colonyNames = ['alpha', 'beta', 'gamma', 'delta'];
  const domainNames = ['dom-a', 'dom-b', 'dom-c', 'dom-d'];

  for (const name of colonyNames) {
    fs.mkdirSync(path.join(skillsDir, 'colony', name), { recursive: true });
    fs.writeFileSync(
      path.join(skillsDir, 'colony', name, 'SKILL.md'),
      [
        '---',
        `name: ${name}`,
        `description: Colony skill ${name}`,
        'type: colony',
        'domains: [testing, quality]',
        'agent_roles: [builder, watcher]',
        'priority: normal',
        'version: "1.0"',
        '---',
        '',
        `Content for colony skill ${name}.`
      ].join('\n')
    );
  }

  for (const name of domainNames) {
    fs.mkdirSync(path.join(skillsDir, 'domain', name), { recursive: true });
    fs.writeFileSync(
      path.join(skillsDir, 'domain', name, 'SKILL.md'),
      [
        '---',
        `name: ${name}`,
        `description: Domain skill ${name}`,
        'type: domain',
        'domains: [testing, quality]',
        'agent_roles: [builder]',
        'detect_files: ["*.test.js"]',
        'priority: normal',
        'version: "1.0"',
        '---',
        '',
        `Content for domain skill ${name}.`
      ].join('\n')
    );
  }

  try {
    runSkillCommand('skill-index', [skillsDir], { AETHER_SKILLS_DIR: skillsDir });

    const result = runSkillCommand('skill-match', ['builder', '', skillsDir], {
      AETHER_SKILLS_DIR: skillsDir
    });

    t.true(result.ok, 'Should return ok: true');
    t.true(result.result.colony_skills.length <= 2,
      `Colony skills should be at most 2, got ${result.result.colony_skills.length}`);
    t.true(result.result.domain_skills.length <= 2,
      `Domain skills should be at most 2, got ${result.result.domain_skills.length}`);
  } finally {
    fs.rmSync(tempDir, { recursive: true, force: true });
  }
});

// ==========================================================================
// Test 5b: skill-match filters out skills with match_score below 20
// ==========================================================================
test('skill-match filters out skills with match_score below 20', t => {
  // Create a domain skill that has no pheromone overlap and no task overlap
  // Domain skills start at score 0, so with no domain matches it stays below 20
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'skills-threshold-'));
  const skillsDir = path.join(tempDir, 'skills');

  // Create a domain skill with obscure domains that won't match anything
  fs.mkdirSync(path.join(skillsDir, 'domain', 'obscure-skill'), { recursive: true });
  fs.writeFileSync(
    path.join(skillsDir, 'domain', 'obscure-skill', 'SKILL.md'),
    [
      '---',
      'name: obscure-skill',
      'description: A very niche skill',
      'type: domain',
      'domains: [zzz-nonexistent-domain]',
      'agent_roles: [builder]',
      'priority: normal',
      'version: "1.0"',
      '---',
      '',
      'Obscure content.'
    ].join('\n')
  );

  try {
    runSkillCommand('skill-index', [skillsDir], { AETHER_SKILLS_DIR: skillsDir });

    // No task description, no pheromones -> domain skill scores 0 -> below threshold
    const result = runSkillCommand('skill-match', ['builder', '', skillsDir], {
      AETHER_SKILLS_DIR: skillsDir
    });

    t.true(result.ok, 'Should return ok: true');
    // The obscure domain skill should be filtered out (score 0 < 20)
    t.is(result.result.domain_skills.length, 0,
      'Domain skill with score below 20 should be filtered out');
  } finally {
    fs.rmSync(tempDir, { recursive: true, force: true });
  }
});

// ==========================================================================
// Test 5c: skill-inject respects 8K char budget (not 12K)
// ==========================================================================
test('skill-inject respects 8K char budget', t => {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'skills-budget-'));
  const skillsDir = path.join(tempDir, 'skills');

  // Create a colony skill with 5000 chars of content
  fs.mkdirSync(path.join(skillsDir, 'colony', 'medium-skill'), { recursive: true });
  fs.writeFileSync(
    path.join(skillsDir, 'colony', 'medium-skill', 'SKILL.md'),
    [
      '---',
      'name: medium-skill',
      'description: Medium sized skill',
      'type: colony',
      'domains: [testing]',
      'agent_roles: [builder]',
      'priority: normal',
      'version: "1.0"',
      '---',
      '',
      'A'.repeat(5000)
    ].join('\n')
  );

  // Create a domain skill with 5000 chars of content
  fs.mkdirSync(path.join(skillsDir, 'domain', 'another-medium'), { recursive: true });
  fs.writeFileSync(
    path.join(skillsDir, 'domain', 'another-medium', 'SKILL.md'),
    [
      '---',
      'name: another-medium',
      'description: Another medium skill',
      'type: domain',
      'domains: [testing]',
      'agent_roles: [builder]',
      'priority: normal',
      'version: "1.0"',
      '---',
      '',
      'B'.repeat(5000)
    ].join('\n')
  );

  try {
    const matchJson = JSON.stringify({
      colony_skills: [{ name: 'medium-skill', file_path: path.join(skillsDir, 'colony', 'medium-skill', 'SKILL.md') }],
      domain_skills: [{ name: 'another-medium', file_path: path.join(skillsDir, 'domain', 'another-medium', 'SKILL.md') }]
    });

    const cmd = `bash "${AETHER_UTILS}" skill-inject '${matchJson.replace(/'/g, "'\\''")}'`;
    const output = execSync(cmd, { cwd: AETHER_ROOT, encoding: 'utf8', timeout: 15000 });
    const result = JSON.parse(output);

    t.true(result.ok, 'Should return ok: true');
    // Both skills together = ~10000 chars, which exceeds 8K budget
    // So total_chars should be capped at 8000 or below
    t.true(result.result.total_chars <= 8000,
      `Total chars should be <= 8000, got ${result.result.total_chars}`);
  } finally {
    fs.rmSync(tempDir, { recursive: true, force: true });
  }
});

// ==========================================================================
// Test 5: skill-match excludes skills for non-matching role
// ==========================================================================
test('skill-match excludes skills for non-matching role', t => {
  const { tempDir, skillsDir } = createSkillsEnv();

  try {
    // Build the index first
    runSkillCommand('skill-index', [skillsDir], { AETHER_SKILLS_DIR: skillsDir });

    // Match as "chronicler" -- neither skill has chronicler in agent_roles
    const result = runSkillCommand('skill-match', ['chronicler', '', skillsDir], {
      AETHER_SKILLS_DIR: skillsDir
    });

    t.true(result.ok, 'Should return ok: true');
    t.is(result.result.colony_skills.length, 0, 'Chronicler should match 0 colony skills');
    t.is(result.result.domain_skills.length, 0, 'Chronicler should match 0 domain skills');
  } finally {
    cleanupEnv(tempDir);
  }
});
