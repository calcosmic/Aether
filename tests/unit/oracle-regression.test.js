const fs = require('fs');
const path = require('path');
const test = require('ava');

const COLONY_STATE_PATH = path.join(__dirname, '../../.aether/data/COLONY_STATE.json');

/**
 * Helper function to detect duplicate keys in JSON string
 * Standard JSON.parse allows duplicates (last one wins), so we need custom detection
 * This version properly handles arrays that contain objects with potential duplicates
 * @param {string} jsonString
 * @returns {object} - { hasDuplicates: boolean, duplicates: Array }
 */
function detectDuplicateKeys(jsonString) {
  const duplicates = [];

  // Track all keys at each object level
  const keyStack = [];
  let currentKeys = new Set();
  let inString = false;
  let escapeNext = false;
  let currentKey = '';
  let expectingKey = true;

  for (let i = 0; i < jsonString.length; i++) {
    const char = jsonString[i];

    if (escapeNext) {
      escapeNext = false;
      if (inString) {
        currentKey += char;
      }
      continue;
    }

    if (char === '\\') {
      escapeNext = true;
      if (inString) {
        currentKey += char;
      }
      continue;
    }

    if (char === '"' && !inString) {
      inString = true;
      currentKey = '';
      continue;
    }

    if (char === '"' && inString) {
      inString = false;
      if (expectingKey) {
        // This was a key - check for duplicates at current object level
        if (currentKeys.has(currentKey)) {
          duplicates.push(currentKey);
        } else {
          currentKeys.add(currentKey);
        }
        expectingKey = false;
      }
      continue;
    }

    if (inString) {
      currentKey += char;
      continue;
    }

    if (char === '{') {
      keyStack.push(currentKeys);
      currentKeys = new Set();
      expectingKey = true;
      continue;
    }

    if (char === '}') {
      currentKeys = keyStack.pop() || new Set();
      expectingKey = false;
      continue;
    }

    if (char === '[') {
      // Arrays don't have keys, but their contents might
      // We don't change state here, just continue
      continue;
    }

    if (char === ']') {
      // End of array - we may have been in a value context
      expectingKey = false;
      continue;
    }

    if (char === ':' && !expectingKey) {
      // Skip over the value - after colon we're in value context
      continue;
    }

    if (char === ',' && !inString) {
      // After comma in object context, we expect a key
      // After comma in array context, we expect a value (which could be an object with keys)
      expectingKey = true;
      continue;
    }
  }

  return {
    hasDuplicates: duplicates.length > 0,
    duplicates: [...new Set(duplicates)] // Remove duplicates from the list itself
  };
}

/**
 * Helper to verify events are in chronological order
 * @param {Array} events
 * @returns {object} - { inOrder: boolean, firstOutOfOrder: object }
 */
function verifyChronologicalOrder(events) {
  for (let i = 1; i < events.length; i++) {
    const prevTime = new Date(events[i - 1].timestamp).getTime();
    const currTime = new Date(events[i].timestamp).getTime();

    if (currTime < prevTime) {
      return {
        inOrder: false,
        firstOutOfOrder: {
          index: i,
          current: events[i],
          previous: events[i - 1]
        }
      };
    }
  }

  return { inOrder: true };
}

// ============================================================================
// ORACLE REGRESSION TESTS
// These tests verify that the detection functions catch the specific bugs
// that Oracle discovered in archived versions of COLONY_STATE.json
// ============================================================================

// Test: Detection function catches duplicate keys (intentional failure test)
test('detectDuplicateKeys catches duplicate status keys', t => {
  const jsonWithDuplicateStatus = `{
    "id": "task_1.1",
    "description": "Test task",
    "status": "in_progress",
    "status": "completed"
  }`;

  const result = detectDuplicateKeys(jsonWithDuplicateStatus);

  t.true(result.hasDuplicates, 'Should detect duplicate keys');
  t.true(result.duplicates.includes('status'), 'Should detect duplicate "status" key');
});

// Test: Detection function catches multiple duplicate keys
test('detectDuplicateKeys catches multiple duplicate keys', t => {
  const jsonWithMultipleDuplicates = `{
    "id": "task_1",
    "id": "task_1_duplicate",
    "status": "pending",
    "status": "completed",
    "priority": "high"
  }`;

  const result = detectDuplicateKeys(jsonWithMultipleDuplicates);

  t.true(result.hasDuplicates, 'Should detect duplicate keys');
  t.is(result.duplicates.length, 2, 'Should detect 2 unique duplicate key names');
  t.true(result.duplicates.includes('id'), 'Should detect duplicate "id" key');
  t.true(result.duplicates.includes('status'), 'Should detect duplicate "status" key');
});

// Test: Detection function works with nested objects
test('detectDuplicateKeys catches duplicates in nested objects', t => {
  const jsonWithNestedDuplicates = `{
    "outer": {
      "key1": "value1",
      "key1": "value2"
    },
    "outer2": "value"
  }`;

  const result = detectDuplicateKeys(jsonWithNestedDuplicates);

  t.true(result.hasDuplicates, 'Should detect duplicate keys in nested objects');
  t.true(result.duplicates.includes('key1'), 'Should detect duplicate "key1" in nested object');
});

// Test: Detection function does not flag valid JSON
test('detectDuplicateKeys does not flag valid JSON without duplicates', t => {
  const validJson = `{
    "id": "task_1",
    "status": "completed",
    "priority": "high"
  }`;

  const result = detectDuplicateKeys(validJson);

  t.false(result.hasDuplicates, 'Should not flag valid JSON without duplicates');
  t.is(result.duplicates.length, 0, 'Should have empty duplicates array');
});

// Test: verifyChronologicalOrder catches out-of-order timestamps
test('verifyChronologicalOrder catches out-of-order events', t => {
  const outOfOrderEvents = [
    { timestamp: '2026-02-13T16:00:00Z', type: 'colony_initialized' },
    { timestamp: '2026-02-13T11:18:15Z', type: 'swarm_completed' }, // BEFORE initialization!
    { timestamp: '2026-02-13T20:58:00Z', type: 'work_completed' }
  ];

  const result = verifyChronologicalOrder(outOfOrderEvents);

  t.false(result.inOrder, 'Should detect out-of-order events');
  t.truthy(result.firstOutOfOrder, 'Should provide details about first out-of-order event');
  t.is(result.firstOutOfOrder.index, 1, 'Should identify index 1 as out of order');
  t.is(result.firstOutOfOrder.current.type, 'swarm_completed', 'Should identify swarm_completed as out of order');
});

// Test: verifyChronologicalOrder passes for correctly ordered events
test('verifyChronologicalOrder passes for correctly ordered events', t => {
  const orderedEvents = [
    { timestamp: '2026-02-13T11:18:15Z', type: 'early_event' },
    { timestamp: '2026-02-13T16:00:00Z', type: 'colony_initialized' },
    { timestamp: '2026-02-13T20:58:00Z', type: 'work_completed' }
  ];

  const result = verifyChronologicalOrder(orderedEvents);

  t.true(result.inOrder, 'Should pass for correctly ordered events');
  t.falsy(result.firstOutOfOrder, 'Should not have firstOutOfOrder for ordered events');
});

// Test: verifyChronologicalOrder handles same timestamps (edge case)
test('verifyChronologicalOrder allows same timestamps (simultaneous events)', t => {
  const simultaneousEvents = [
    { timestamp: '2026-02-13T16:00:00Z', type: 'event_a' },
    { timestamp: '2026-02-13T16:00:00Z', type: 'event_b' },
    { timestamp: '2026-02-13T16:00:00Z', type: 'event_c' }
  ];

  const result = verifyChronologicalOrder(simultaneousEvents);

  t.true(result.inOrder, 'Should allow simultaneous events (same timestamp)');
});

// Test: verifyChronologicalOrder handles empty and single-event arrays
test('verifyChronologicalOrder handles edge cases', t => {
  const emptyResult = verifyChronologicalOrder([]);
  t.true(emptyResult.inOrder, 'Should pass for empty events array');

  const singleResult = verifyChronologicalOrder([{ timestamp: '2026-02-13T16:00:00Z', type: 'only_event' }]);
  t.true(singleResult.inOrder, 'Should pass for single event');
});

// ============================================================================
// DOCUMENTATION TESTS
// These tests document the specific Oracle-discovered bugs for future reference
// ============================================================================

// Test: Document the specific Oracle bug - duplicate status in task 1.1
test('Oracle bug documented: duplicate status keys in task objects', t => {
  // This test documents the specific bug Oracle found:
  // Task 1.1 had duplicate "status" keys in the JSON structure
  // Example of what was found (archived version):
  // Note: Using single quotes for the string to avoid escaping issues
  const oracleBugExample = '{' +
    '"plan": {' +
      '"phases": [{' +
        '"tasks": [{' +
          '"id": "1.1",' +
          '"description": "Task description",' +
          '"success_criteria": [],' +
          '"status": "in_progress",' +
          '"status": "completed"' +
        '}]' +
      '}]' +
    '}' +
  '}';

  const result = detectDuplicateKeys(oracleBugExample);

  t.true(result.hasDuplicates, 'Oracle bug: duplicate "status" key should be detected');
  t.true(result.duplicates.includes('status'), 'Should specifically detect duplicate "status"');

  // Verify current COLONY_STATE.json does NOT have this bug
  const currentContent = fs.readFileSync(COLONY_STATE_PATH, 'utf8');
  const currentResult = detectDuplicateKeys(currentContent);
  t.false(currentResult.hasDuplicates, 'Current COLONY_STATE.json should NOT have duplicate keys (bug is fixed)');
});

// Test: Document the specific Oracle bug - out-of-order timestamps
test('Oracle bug documented: events before initialization timestamp', t => {
  // This test documents the specific bug Oracle found:
  // Events at lines 173-175 had timestamps before initialization
  // Example of what was found (archived version):
  const oracleBugExample = [
    { timestamp: '2026-02-13T16:00:00Z', type: 'colony_initialized' },
    { timestamp: '2026-02-13T11:18:15Z', type: 'swarm_completed' },  // BEFORE init!
    { timestamp: '2026-02-13T11:20:00Z', type: 'work_completed' }    // BEFORE init!
  ];

  const result = verifyChronologicalOrder(oracleBugExample);

  t.false(result.inOrder, 'Oracle bug: events before initialization should be detected');
  t.is(result.firstOutOfOrder?.current?.type, 'swarm_completed', 'Should identify swarm_completed as out of order');

  // Verify current COLONY_STATE.json does NOT have this bug
  const currentContent = fs.readFileSync(COLONY_STATE_PATH, 'utf8');
  const currentData = JSON.parse(currentContent);
  const currentResult = verifyChronologicalOrder(currentData.events);
  t.true(currentResult.inOrder, 'Current COLONY_STATE.json events should be in order (bug is fixed)');
});
