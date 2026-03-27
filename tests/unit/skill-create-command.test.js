/**
 * Skill-Create Command Quality Tests
 *
 * Validates that .claude/commands/ant/skill-create.md exists,
 * has proper YAML frontmatter, and contains all required sections
 * for the skill creation wizard flow.
 */

'use strict';

const test = require('ava');
const yaml = require('js-yaml');
const fs = require('fs');
const path = require('path');

const COMMAND_PATH = path.join(__dirname, '../../.claude/commands/ant/skill-create.md');

// ---------------------------------------------------------------------------
// Helper: parse command file
// ---------------------------------------------------------------------------

function parseCommandFile() {
  const content = fs.readFileSync(COMMAND_PATH, 'utf8');
  const parts = content.split(/^---\s*$/m);
  if (parts.length < 3) return null;
  try {
    const frontmatter = yaml.load(parts[1]);
    const body = parts.slice(2).join('---');
    return { frontmatter, body, content };
  } catch (e) {
    return null;
  }
}

// ---------------------------------------------------------------------------
// TEST-01: File exists
// ---------------------------------------------------------------------------

test('TEST-01: skill-create.md exists', t => {
  t.true(fs.existsSync(COMMAND_PATH), 'Command file should exist at .claude/commands/ant/skill-create.md');
});

// ---------------------------------------------------------------------------
// TEST-02: Valid YAML frontmatter
// ---------------------------------------------------------------------------

test('TEST-02: has valid YAML frontmatter with name and description', t => {
  const parsed = parseCommandFile();
  t.truthy(parsed, 'Command file should have valid frontmatter delimiters');
  t.truthy(parsed.frontmatter, 'Frontmatter should parse as valid YAML');
  t.is(parsed.frontmatter.name, 'ant:skill-create', 'name should be ant:skill-create');
  t.truthy(parsed.frontmatter.description, 'description should be present');
  t.true(parsed.frontmatter.description.length > 10, 'description should be descriptive');
});

// ---------------------------------------------------------------------------
// TEST-03: Contains argument parsing (Step 0)
// ---------------------------------------------------------------------------

test('TEST-03: includes argument parsing step', t => {
  const parsed = parseCommandFile();
  t.truthy(parsed, 'Command file should parse');
  t.true(parsed.body.includes('$ARGUMENTS'), 'Should reference $ARGUMENTS');
  t.regex(parsed.body, /Step 0|Parse Argument/i, 'Should have argument parsing step');
});

// ---------------------------------------------------------------------------
// TEST-04: Contains Oracle mini-research (Step 1)
// ---------------------------------------------------------------------------

test('TEST-04: includes Oracle mini-research step', t => {
  const parsed = parseCommandFile();
  t.truthy(parsed, 'Command file should parse');
  t.regex(parsed.body, /Oracle|research|web\s*search/i, 'Should reference Oracle research');
  t.regex(parsed.body, /5\s*iteration|quick\s*research|mini.?research/i, 'Should specify 5-iteration research');
});

// ---------------------------------------------------------------------------
// TEST-05: Contains wizard questions using AskUserQuestion (Step 2)
// ---------------------------------------------------------------------------

test('TEST-05: includes wizard questions with AskUserQuestion', t => {
  const parsed = parseCommandFile();
  t.truthy(parsed, 'Command file should parse');
  t.true(parsed.body.includes('AskUserQuestion'), 'Should use AskUserQuestion for wizard');
  t.regex(parsed.body, /experience\s*level|beginner|intermediate|advanced/i, 'Should ask about experience level');
  t.regex(parsed.body, /focus|aspect/i, 'Should ask about focus area');
  t.regex(parsed.body, /rules|constraints/i, 'Should ask about specific rules or constraints');
});

// ---------------------------------------------------------------------------
// TEST-06: Contains SKILL.md generation (Step 3)
// ---------------------------------------------------------------------------

test('TEST-06: includes SKILL.md generation with correct frontmatter fields', t => {
  const parsed = parseCommandFile();
  t.truthy(parsed, 'Command file should parse');
  // All required frontmatter fields for the generated skill
  const requiredFields = ['name', 'description', 'type: domain', 'domains', 'agent_roles', 'detect_files', 'detect_packages', 'priority', 'version'];
  for (const field of requiredFields) {
    t.true(parsed.body.includes(field), `Should mention frontmatter field: ${field}`);
  }
});

// ---------------------------------------------------------------------------
// TEST-07: Contains write and verify (Step 4)
// ---------------------------------------------------------------------------

test('TEST-07: includes write, verify, and cache rebuild', t => {
  const parsed = parseCommandFile();
  t.truthy(parsed, 'Command file should parse');
  t.true(parsed.body.includes('~/.aether/skills/domain/'), 'Should write to user skills directory');
  t.true(parsed.body.includes('skill-parse-frontmatter'), 'Should verify with skill-parse-frontmatter');
  t.true(parsed.body.includes('skill-cache-rebuild'), 'Should rebuild skill cache');
});

// ---------------------------------------------------------------------------
// TEST-08: Does not reference protected paths for write
// ---------------------------------------------------------------------------

test('TEST-08: does not write to protected colony paths', t => {
  const parsed = parseCommandFile();
  t.truthy(parsed, 'Command file should parse');
  // Should not write to .aether/data/ or .aether/dreams/
  t.false(parsed.body.includes('Write tool to create `.aether/data/'), 'Should not write to .aether/data/');
  t.false(parsed.body.includes('Write tool to create `.aether/dreams/'), 'Should not write to .aether/dreams/');
});
