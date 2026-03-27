const test = require('ava');
const fs = require('fs');
const path = require('path');

const AETHER_ROOT = path.join(__dirname, '../..');

/**
 * Verify that build-wave.md and build-context.md enforce context layer budget caps.
 *
 * - archaeology_context: 4000 char cap with "[archaeology truncated]" marker
 * - midden_context: 2000 char cap, last 3 failures (not 5)
 * - grave_context: 2000 char cap per worker
 * - research budget: 8000 chars (reduced from 16000)
 */

const BUILD_WAVE_PATH = path.join(AETHER_ROOT, '.aether/docs/command-playbooks/build-wave.md');
const BUILD_CONTEXT_PATH = path.join(AETHER_ROOT, '.aether/docs/command-playbooks/build-context.md');

// --- build-wave.md: archaeology_context budget ---

test('build-wave.md enforces 4000 char cap on archaeology_context', t => {
  const content = fs.readFileSync(BUILD_WAVE_PATH, 'utf8');
  t.true(
    content.includes('4000'),
    'Should mention 4000 character budget for archaeology context'
  );
});

test('build-wave.md uses "[archaeology truncated]" marker', t => {
  const content = fs.readFileSync(BUILD_WAVE_PATH, 'utf8');
  t.true(
    content.includes('[archaeology truncated]'),
    'Should include truncation marker "[archaeology truncated]"'
  );
});

// --- build-wave.md: midden_context budget ---

test('build-wave.md enforces 2000 char cap on midden_context', t => {
  const content = fs.readFileSync(BUILD_WAVE_PATH, 'utf8');
  // Check for 2000 char budget mention near midden context
  const middenIdx = content.indexOf('midden_context');
  const relevantSection = content.substring(middenIdx, middenIdx + 2000);
  t.true(
    relevantSection.includes('2000') || relevantSection.includes('2,000'),
    'Should mention 2000 character budget for midden context'
  );
});

test('build-wave.md limits midden to last 3 failures', t => {
  const content = fs.readFileSync(BUILD_WAVE_PATH, 'utf8');
  // The midden-recent-failures call for builder context should use 3 instead of 5
  t.true(
    content.includes('midden-recent-failures 3'),
    'Should call midden-recent-failures with limit of 3 (not 5)'
  );
  // Ensure there is no "midden-recent-failures 5" (but allow "midden-recent-failures 50" for threshold checks)
  const matches = content.match(/midden-recent-failures 5(?!\d)/g);
  t.is(
    matches,
    null,
    'Should NOT call midden-recent-failures with limit of exactly 5'
  );
});

test('build-wave.md uses "[midden truncated]" marker', t => {
  const content = fs.readFileSync(BUILD_WAVE_PATH, 'utf8');
  t.true(
    content.includes('[midden truncated]'),
    'Should include truncation marker "[midden truncated]"'
  );
});

// --- build-wave.md: grave_context budget ---

test('build-wave.md enforces 2000 char cap on grave_context per worker', t => {
  const content = fs.readFileSync(BUILD_WAVE_PATH, 'utf8');
  // Look for the 2000 cap near grave_context
  const graveIdx = content.indexOf('grave_context');
  const relevantSection = content.substring(graveIdx, graveIdx + 2000);
  t.true(
    relevantSection.includes('2000') || relevantSection.includes('2,000'),
    'Should mention 2000 character budget for grave context per worker'
  );
});

test('build-wave.md uses "[graveyard truncated]" marker', t => {
  const content = fs.readFileSync(BUILD_WAVE_PATH, 'utf8');
  t.true(
    content.includes('[graveyard truncated]'),
    'Should include truncation marker "[graveyard truncated]"'
  );
});

// --- build-context.md: research budget ---

test('build-context.md uses 8000 char research budget', t => {
  const content = fs.readFileSync(BUILD_CONTEXT_PATH, 'utf8');
  t.true(
    content.includes('research_budget=8000'),
    'Should set research_budget to 8000'
  );
});

test('build-context.md does not use 16000 char research budget', t => {
  const content = fs.readFileSync(BUILD_CONTEXT_PATH, 'utf8');
  t.false(
    content.includes('research_budget=16000'),
    'Should NOT have the old 16000 research budget'
  );
});

test('build-context.md research budget comment reflects colony-prime parity', t => {
  const content = fs.readFileSync(BUILD_CONTEXT_PATH, 'utf8');
  t.true(
    content.includes('8K') || content.includes('8000') || content.includes('8,000'),
    'Should reference the 8K research budget in documentation text'
  );
});
