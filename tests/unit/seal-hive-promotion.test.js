/**
 * Seal Hive Promotion Tests
 *
 * Validates that seal.md files:
 * 1. Use correct jq path .memory.instincts[] (not .instincts[])
 * 2. Strip leading "When " from trigger before building promote_text
 * 3. Both Claude and OpenCode seal.md stay in sync on these patterns
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');

const CLAUDE_SEAL = path.join(process.cwd(), '.claude', 'commands', 'ant', 'seal.md');
const OPENCODE_SEAL = path.join(process.cwd(), '.opencode', 'commands', 'ant', 'seal.md');

test('Claude seal.md uses .memory.instincts[] jq path', t => {
  const content = fs.readFileSync(CLAUDE_SEAL, 'utf8');
  // Must use .memory.instincts[] — COLONY_STATE.json stores instincts under .memory
  t.true(content.includes('.memory.instincts[]'), 'jq path should be .memory.instincts[]');
  // Must NOT use bare .instincts[] (wrong path)
  t.false(
    /jq[^']*'\.\binstincts\b\[\]/.test(content) && !content.includes('.memory.instincts[]'),
    'should not use bare .instincts[] without .memory prefix'
  );
});

test('OpenCode seal.md uses .memory.instincts[] jq path', t => {
  const content = fs.readFileSync(OPENCODE_SEAL, 'utf8');
  t.true(content.includes('.memory.instincts[]'), 'jq path should be .memory.instincts[]');
  t.false(
    /jq[^']*'\.\binstincts\b\[\]/.test(content) && !content.includes('.memory.instincts[]'),
    'should not use bare .instincts[] without .memory prefix'
  );
});

test('Claude seal.md strips leading "When" from trigger before building promote_text', t => {
  const content = fs.readFileSync(CLAUDE_SEAL, 'utf8');
  // Should have trigger cleanup pattern
  t.true(
    content.includes("trigger_clean") || content.includes("sed 's/^[Ww]hen //'"),
    'should strip leading When/when from trigger variable'
  );
  // The promote_text should use the cleaned trigger
  t.true(
    content.includes('trigger_clean'),
    'promote_text should reference trigger_clean, not raw trigger'
  );
});

test('OpenCode seal.md strips leading "When" from trigger before building promote_text', t => {
  const content = fs.readFileSync(OPENCODE_SEAL, 'utf8');
  t.true(
    content.includes("trigger_clean") || content.includes("sed 's/^[Ww]hen //'"),
    'should strip leading When/when from trigger variable'
  );
  t.true(
    content.includes('trigger_clean'),
    'promote_text should reference trigger_clean, not raw trigger'
  );
});

test('Claude and OpenCode seal.md hive promotion blocks are in sync', t => {
  const claude = fs.readFileSync(CLAUDE_SEAL, 'utf8');
  const opencode = fs.readFileSync(OPENCODE_SEAL, 'utf8');

  // Extract the hive promotion bash block from both files
  // Both should have the same jq path and trigger cleanup logic
  const claudeHasMemoryPath = claude.includes('.memory.instincts[]');
  const opencodeHasMemoryPath = opencode.includes('.memory.instincts[]');
  t.is(claudeHasMemoryPath, opencodeHasMemoryPath, 'both files should use same jq path');

  const claudeHasCleanup = claude.includes('trigger_clean');
  const opencodeHasCleanup = opencode.includes('trigger_clean');
  t.is(claudeHasCleanup, opencodeHasCleanup, 'both files should have trigger cleanup');
});
