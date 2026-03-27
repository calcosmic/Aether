const test = require('ava');
const fs = require('fs');
const path = require('path');

const AETHER_ROOT = path.join(__dirname, '../..');

/**
 * Verify that build-verify.md contains skill injection for the Watcher role.
 * Task 3.3: Add skill injection to the Watcher prompt in build-verify.md
 */

const BUILD_VERIFY_PATH = path.join(AETHER_ROOT, '.aether/docs/command-playbooks/build-verify.md');

test('build-verify.md exists', t => {
  t.true(fs.existsSync(BUILD_VERIFY_PATH), 'build-verify.md should exist');
});

test('build-verify.md contains skill-match call for watcher role', t => {
  const content = fs.readFileSync(BUILD_VERIFY_PATH, 'utf8');
  t.true(
    content.includes('skill-match "watcher"'),
    'Should contain skill-match call with "watcher" role'
  );
});

test('build-verify.md contains skill-inject call', t => {
  const content = fs.readFileSync(BUILD_VERIFY_PATH, 'utf8');
  t.true(
    content.includes('skill-inject'),
    'Should contain skill-inject call'
  );
});

test('build-verify.md Watcher prompt includes skill_section', t => {
  const content = fs.readFileSync(BUILD_VERIFY_PATH, 'utf8');
  t.true(
    content.includes('{ skill_section }'),
    'Watcher prompt should include { skill_section }'
  );
});

test('build-verify.md skill_section appears after prompt_section', t => {
  const content = fs.readFileSync(BUILD_VERIFY_PATH, 'utf8');
  const promptIdx = content.indexOf('{ prompt_section }');
  const skillIdx = content.indexOf('{ skill_section }');
  t.true(promptIdx >= 0, 'prompt_section must exist');
  t.true(skillIdx >= 0, 'skill_section must exist');
  t.true(skillIdx > promptIdx, 'skill_section should appear after prompt_section');
});

test('build-verify.md contains skill loading display message', t => {
  const content = fs.readFileSync(BUILD_VERIFY_PATH, 'utf8');
  t.true(
    content.includes('Skills loaded for watcher verification'),
    'Should display skill loading message'
  );
});

test('build-verify.md skill-match appears before Watcher prompt template', t => {
  const content = fs.readFileSync(BUILD_VERIFY_PATH, 'utf8');
  const skillMatchIdx = content.indexOf('skill-match "watcher"');
  const watcherPromptIdx = content.indexOf('**Watcher Worker Prompt');
  t.true(skillMatchIdx >= 0, 'skill-match must exist');
  t.true(watcherPromptIdx >= 0, 'Watcher Worker Prompt must exist');
  t.true(skillMatchIdx < watcherPromptIdx, 'skill-match should appear before Watcher Worker Prompt');
});

test('build-verify.md does not replace prompt_section with skill_section', t => {
  const content = fs.readFileSync(BUILD_VERIFY_PATH, 'utf8');
  t.true(
    content.includes('{ prompt_section }'),
    'prompt_section must still be present (not replaced)'
  );
  t.true(
    content.includes('{ skill_section }'),
    'skill_section must be present alongside prompt_section'
  );
});
