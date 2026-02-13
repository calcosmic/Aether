const fs = require('fs');
const path = require('path');
const test = require('ava');

const COLONY_STATE_PATH = path.join(__dirname, '../../.aether/data/COLONY_STATE.json');

/**
 * Helper function to detect duplicate keys in JSON string
 * Standard JSON.parse allows duplicates (last one wins), so we need custom detection
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
  let braceDepth = 0;

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
        // This was a key
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
      braceDepth++;
      keyStack.push(currentKeys);
      currentKeys = new Set();
      expectingKey = true;
      continue;
    }

    if (char === '}') {
      braceDepth--;
      currentKeys = keyStack.pop() || new Set();
      expectingKey = false;
      continue;
    }

    if (char === ':' && !expectingKey) {
      // Skip over the value
      continue;
    }

    if (char === ',' && !inString) {
      expectingKey = true;
      continue;
    }

    if (char === '[') {
      // Skip array content - arrays don't have keys
      let depth = 1;
      i++;
      while (i < jsonString.length && depth > 0) {
        if (jsonString[i] === '"' && jsonString[i - 1] !== '\\') {
          // Skip string content inside array
          i++;
          while (i < jsonString.length && !(jsonString[i] === '"' && jsonString[i - 1] !== '\\')) {
            i++;
          }
        } else if (jsonString[i] === '[') {
          depth++;
        } else if (jsonString[i] === ']') {
          depth--;
        }
        i++;
      }
      i--; // Adjust for loop increment
      continue;
    }
  }

  return {
    hasDuplicates: duplicates.length > 0,
    duplicates: [...new Set(duplicates)] // Remove duplicates from the list itself
  };
}

/**
 * Helper to check for duplicate keys in specific object paths
 * Uses a regex-based approach to find duplicate "status" keys in task objects
 * @param {string} jsonString
 * @returns {Array} - Array of paths where duplicates were found
 */
function findTaskDuplicateStatusKeys(jsonString) {
  const issues = [];

  // Look for patterns like "status": ... "status": within the same object scope
  // This regex finds objects that have "status" appearing more than once
  const objectPattern = /\{[^{}]*"status"[^{}]*"status"[^{}]*\}/g;
  const matches = jsonString.match(objectPattern);

  if (matches) {
    matches.forEach((match, index) => {
      issues.push(`Potential duplicate "status" key in object #${index + 1}`);
    });
  }

  // More specific check: look for "status" followed by any content then another "status"
  // within what appears to be a single object (no unbalanced braces between them)
  const lines = jsonString.split('\n');
  const statusLineNumbers = [];

  lines.forEach((line, idx) => {
    // Count occurrences of "status" as a key in this line
    const matches = line.match(/"status"\s*:/g);
    if (matches && matches.length > 1) {
      issues.push(`Line ${idx + 1}: Multiple "status" keys in same line: ${line.trim()}`);
    }

    // Track all "status" key occurrences
    if (line.includes('"status"')) {
      statusLineNumbers.push({ line: idx + 1, content: line.trim() });
    }
  });

  return {
    issues,
    allStatusOccurrences: statusLineNumbers
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

// Test: File exists and is readable
test('COLONY_STATE.json exists and is readable', t => {
  t.true(fs.existsSync(COLONY_STATE_PATH), 'COLONY_STATE.json should exist');

  const content = fs.readFileSync(COLONY_STATE_PATH, 'utf8');
  t.truthy(content, 'File should have content');
  t.true(content.length > 0, 'File should not be empty');
});

// Test: JSON is valid and parses correctly
test('COLONY_STATE.json contains valid JSON', t => {
  const content = fs.readFileSync(COLONY_STATE_PATH, 'utf8');

  let parsed;
  t.notThrows(() => {
    parsed = JSON.parse(content);
  }, 'JSON should parse without errors');

  t.truthy(parsed, 'Parsed result should exist');
  t.is(typeof parsed, 'object', 'Parsed result should be an object');
});

// Test: Required top-level fields exist
test('COLONY_STATE.json has all required top-level fields', t => {
  const content = fs.readFileSync(COLONY_STATE_PATH, 'utf8');
  const data = JSON.parse(content);

  const requiredFields = [
    'version',
    'goal',
    'state',
    'current_phase',
    'plan',
    'memory',
    'errors',
    'events'
  ];

  for (const field of requiredFields) {
    t.true(field in data, `Should have required field: ${field}`);
  }
});

// Test: Field types are correct
test('COLONY_STATE.json fields have correct types', t => {
  const content = fs.readFileSync(COLONY_STATE_PATH, 'utf8');
  const data = JSON.parse(content);

  t.is(typeof data.version, 'string', 'version should be a string');
  t.true(data.goal === null || typeof data.goal === 'string', 'goal should be null or string');
  t.is(typeof data.state, 'string', 'state should be a string');
  t.is(typeof data.current_phase, 'number', 'current_phase should be a number');
  t.is(typeof data.plan, 'object', 'plan should be an object');
  t.is(typeof data.memory, 'object', 'memory should be an object');
  t.is(typeof data.errors, 'object', 'errors should be an object');
  t.true(Array.isArray(data.events), 'events should be an array');
});

// Test: No duplicate keys in JSON structure
test('COLONY_STATE.json has no duplicate keys', t => {
  const content = fs.readFileSync(COLONY_STATE_PATH, 'utf8');
  const duplicateCheck = detectDuplicateKeys(content);

  t.false(
    duplicateCheck.hasDuplicates,
    `Found duplicate keys: ${duplicateCheck.duplicates.join(', ')}`
  );
});

// Test: Task objects don't have duplicate "status" keys
test('COLONY_STATE.json task objects have no duplicate status keys', t => {
  const content = fs.readFileSync(COLONY_STATE_PATH, 'utf8');
  const statusCheck = findTaskDuplicateStatusKeys(content);

  t.is(
    statusCheck.issues.length,
    0,
    `Found duplicate status key issues:\n${statusCheck.issues.join('\n')}`
  );
});

// Test: Events array is in chronological order
test('COLONY_STATE.json events are in chronological order', t => {
  const content = fs.readFileSync(COLONY_STATE_PATH, 'utf8');
  const data = JSON.parse(content);

  if (data.events.length < 2) {
    t.pass('Not enough events to check ordering');
    return;
  }

  const orderCheck = verifyChronologicalOrder(data.events);

  t.true(
    orderCheck.inOrder,
    orderCheck.firstOutOfOrder
      ? `Events out of order at index ${orderCheck.firstOutOfOrder.index}: ` +
        `"${orderCheck.firstOutOfOrder.current.type}" (${orderCheck.firstOutOfOrder.current.timestamp}) ` +
        `comes after "${orderCheck.firstOutOfOrder.previous.type}" (${orderCheck.firstOutOfOrder.previous.timestamp})`
      : 'Events should be in chronological order'
  );
});

// Test: Each event has required fields
test('COLONY_STATE.json events have required fields', t => {
  const content = fs.readFileSync(COLONY_STATE_PATH, 'utf8');
  const data = JSON.parse(content);

  if (data.events.length === 0) {
    t.pass('No events to check');
    return;
  }

  for (let i = 0; i < data.events.length; i++) {
    const event = data.events[i];
    const prefix = `Event ${i}`;

    t.true('timestamp' in event, `${prefix} should have timestamp`);
    t.true('type' in event, `${prefix} should have type`);
    t.true('worker' in event, `${prefix} should have worker`);
    t.true('details' in event, `${prefix} should have details`);

    // Validate timestamp format (ISO 8601)
    const timestamp = new Date(event.timestamp);
    t.false(isNaN(timestamp.getTime()), `${prefix} timestamp should be valid ISO 8601`);
  }
});

// Test: Errors object structure
test('COLONY_STATE.json errors object has correct structure', t => {
  const content = fs.readFileSync(COLONY_STATE_PATH, 'utf8');
  const data = JSON.parse(content);

  t.true('records' in data.errors, 'errors should have records field');
  t.true(Array.isArray(data.errors.records), 'errors.records should be an array');

  if (data.errors.records.length > 0) {
    const firstError = data.errors.records[0];
    t.true('id' in firstError, 'error record should have id');
    t.true('category' in firstError, 'error record should have category');
    t.true('severity' in firstError, 'error record should have severity');
    t.true('description' in firstError, 'error record should have description');
    t.true('timestamp' in firstError, 'error record should have timestamp');
  }
});

// Test: Memory object structure
test('COLONY_STATE.json memory object has correct structure', t => {
  const content = fs.readFileSync(COLONY_STATE_PATH, 'utf8');
  const data = JSON.parse(content);

  t.true('phase_learnings' in data.memory, 'memory should have phase_learnings');
  t.true(Array.isArray(data.memory.phase_learnings), 'memory.phase_learnings should be an array');
  t.true('decisions' in data.memory, 'memory should have decisions');
  t.true(Array.isArray(data.memory.decisions), 'memory.decisions should be an array');
  t.true('instincts' in data.memory, 'memory should have instincts');
  t.true(Array.isArray(data.memory.instincts), 'memory.instincts should be an array');
});
