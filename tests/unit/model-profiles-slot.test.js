const test = require('ava');
const {
  getModelSlotForCaste,
  validateSlot,
} = require('../../bin/lib/model-profiles');
const { buildMockProfiles, getCasteNames } = require('../helpers/mock-profiles');

// ============================================
// getModelSlotForCaste tests
// ============================================

test('getModelSlotForCaste returns "sonnet" for builder caste', t => {
  const profiles = buildMockProfiles();
  t.is(getModelSlotForCaste(profiles, 'builder'), 'sonnet');
});

test('getModelSlotForCaste returns "opus" for queen caste', t => {
  const profiles = buildMockProfiles();
  t.is(getModelSlotForCaste(profiles, 'queen'), 'opus');
});

test('getModelSlotForCaste returns "inherit" for chronicler caste', t => {
  const profiles = buildMockProfiles();
  t.is(getModelSlotForCaste(profiles, 'chronicler'), 'inherit');
});

test('getModelSlotForCaste returns "inherit" for unknown caste', t => {
  const profiles = buildMockProfiles();
  t.is(getModelSlotForCaste(profiles, 'unknown_caste'), 'inherit');
});

test('getModelSlotForCaste returns "inherit" when profiles is null', t => {
  t.is(getModelSlotForCaste(null, 'builder'), 'inherit');
});

test('getModelSlotForCaste returns "inherit" when profiles is undefined', t => {
  t.is(getModelSlotForCaste(undefined, 'builder'), 'inherit');
});

test('getModelSlotForCaste returns "inherit" when worker_models is missing', t => {
  t.is(getModelSlotForCaste({}, 'builder'), 'inherit');
});

test('getModelSlotForCaste returns correct slot for all 24 castes', t => {
  const profiles = buildMockProfiles();
  const casteNames = getCasteNames();

  // All castes defined in worker_models should return a valid slot
  for (const caste of casteNames) {
    const slot = getModelSlotForCaste(profiles, caste);
    t.true(
      ['opus', 'sonnet', 'inherit'].includes(slot),
      `Caste ${caste} should return a valid slot, got: ${slot}`
    );
  }

  // Verify specific caste counts per tier
  const opusCastes = casteNames.filter(c => getModelSlotForCaste(profiles, c) === 'opus');
  const sonnetCastes = casteNames.filter(c => getModelSlotForCaste(profiles, c) === 'sonnet');
  const inheritCastes = casteNames.filter(c => getModelSlotForCaste(profiles, c) === 'inherit');

  t.is(opusCastes.length, 10, 'Should have 10 opus-tier castes');
  t.is(sonnetCastes.length, 11, 'Should have 11 sonnet-tier castes');
  t.is(inheritCastes.length, 3, 'Should have 3 inherit-tier castes');
});

// ============================================
// validateSlot tests
// ============================================

test('validateSlot returns valid for "opus"', t => {
  const result = validateSlot('opus');
  t.deepEqual(result, { valid: true, error: null });
});

test('validateSlot returns valid for "sonnet"', t => {
  const result = validateSlot('sonnet');
  t.deepEqual(result, { valid: true, error: null });
});

test('validateSlot returns valid for "haiku"', t => {
  const result = validateSlot('haiku');
  t.deepEqual(result, { valid: true, error: null });
});

test('validateSlot returns valid for "inherit"', t => {
  const result = validateSlot('inherit');
  t.deepEqual(result, { valid: true, error: null });
});

test('validateSlot returns invalid for "gpt-4" with valid options in error', t => {
  const result = validateSlot('gpt-4');
  t.false(result.valid);
  t.truthy(result.error);
  t.true(result.error.includes('Invalid slot'));
  t.true(result.error.includes('gpt-4'));
  t.true(result.error.includes('opus'));
  t.true(result.error.includes('sonnet'));
  t.true(result.error.includes('haiku'));
  t.true(result.error.includes('inherit'));
});

test('validateSlot returns invalid for empty string', t => {
  const result = validateSlot('');
  t.false(result.valid);
  t.truthy(result.error);
  t.true(result.error.includes('Invalid slot'));
});

test('validateSlot returns invalid for null', t => {
  const result = validateSlot(null);
  t.false(result.valid);
  t.truthy(result.error);
  t.true(result.error.includes('Invalid slot'));
});

test('validateSlot returns invalid for undefined', t => {
  const result = validateSlot(undefined);
  t.false(result.valid);
  t.truthy(result.error);
  t.true(result.error.includes('Invalid slot'));
});
