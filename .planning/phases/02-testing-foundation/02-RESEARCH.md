# Phase 2: Testing Foundation - Research

**Researched:** 2026-02-13
**Domain:** AVA test framework, bash testing, JSON validation, Oracle bug fixes
**Confidence:** HIGH

## Summary

This research covers the implementation of Phase 2: Testing Foundation, which involves setting up AVA test framework for Node.js utilities, creating bash integration tests for aether-utils.sh, fixing Oracle-discovered bugs (duplicate keys and timestamp ordering), and ensuring existing tests continue to pass.

The standard approach is:
1. **AVA v6.x** for Node.js unit tests (CommonJS compatible with existing codebase)
2. **Simple bash assertions** (no external framework) for aether-utils.sh integration tests
3. **Custom JSON parsing** for duplicate key detection (native JSON.parse allows duplicates)
4. **Native Date comparison** for chronological timestamp validation

**Primary recommendation:** Use AVA 6.x with CommonJS configuration, simple bash test patterns matching existing e2e tests, and custom validation logic for Oracle bug detection.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| ava | ^6.0.0 | Node.js test runner | Fast parallel execution, minimal API, native ESM/CommonJS support |
| jq | system | JSON validation in bash | Industry standard for CLI JSON processing |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| child_process | native | Execute bash from Node.js | Testing aether-utils.sh subcommands |
| fs | native | File system operations | Reading COLONY_STATE.json, test fixtures |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| AVA | Jest, Mocha | AVA is lighter, faster, and has simpler parallel execution |
| Simple bash assertions | BATS framework | BATS adds dependency; simple assertions match existing e2e pattern |
| Custom duplicate detection | json5 library | Custom solution avoids extra dependency for single use case |

**Installation:**
```bash
npm install --save-dev ava@^6.0.0
```

## Architecture Patterns

### Recommended Project Structure
```
tests/
├── unit/                    # AVA unit tests for Node.js utilities
│   ├── colony-state.test.js
│   └── validate-state.test.js
├── bash/                    # Bash integration tests
│   ├── test-helpers.sh
│   └── test-aether-utils.sh
└── e2e/                     # Existing end-to-end tests
    ├── test-install.sh
    └── run-all.sh
test/                        # Existing tests (keep separate)
├── sync-dir-hash.test.js
├── user-modification-detection.test.js
└── namespace-isolation.test.js
```

### Pattern 1: AVA Test Structure
**What:** Standard AVA test file structure for CommonJS
**When to use:** All Node.js unit tests
**Example:**
```javascript
// Source: AVA documentation patterns
const test = require('ava');
const fs = require('fs');
const path = require('path');

const COLONY_STATE_PATH = path.join(__dirname, '../../.aether/data/COLONY_STATE.json');

test('COLONY_STATE.json is valid JSON', t => {
  const content = fs.readFileSync(COLONY_STATE_PATH, 'utf8');
  const state = JSON.parse(content);
  t.truthy(state.version);
  t.truthy(state.goal);
});

test('events are in chronological order', t => {
  const state = JSON.parse(fs.readFileSync(COLONY_STATE_PATH, 'utf8'));
  for (let i = 1; i < state.events.length; i++) {
    const prev = new Date(state.events[i-1].timestamp);
    const curr = new Date(state.events[i].timestamp);
    t.true(curr >= prev, `Event ${i} timestamp out of order`);
  }
});
```

### Pattern 2: Duplicate Key Detection
**What:** Custom detection since JSON.parse allows duplicate keys (last one wins per RFC 8259)
**When to use:** Validating JSON files for Oracle bugs
**Example:**
```javascript
// Source: Research findings - RFC 8259 compliance
function detectDuplicateKeys(jsonString) {
  const duplicates = [];
  const seenAtDepth = new Map();

  // Use reviver to track keys at each depth
  JSON.parse(jsonString, (key, value) => {
    if (typeof key === 'string') {
      const depth = this ? this._depth : 0;
      const path = this ? this._path + '.' + key : key;

      if (seenAtDepth.has(path)) {
        duplicates.push({ key, path });
      } else {
        seenAtDepth.set(path, true);
      }
    }
    return value;
  });

  return duplicates;
}

// Alternative: Regex-based detection for specific patterns
test('no duplicate status keys in tasks', t => {
  const content = fs.readFileSync(COLONY_STATE_PATH, 'utf8');
  // Match task objects and check for duplicate status keys
  const taskPattern = /"id":\s*"[\d.]+"[\s\S]*?"status":\s*"\w+"[\s\S]*?"status":/g;
  const matches = content.match(taskPattern);
  t.is(matches, null, 'Found duplicate status keys in tasks');
});
```

### Pattern 3: Bash Test Structure
**What:** Simple assertion pattern matching existing e2e tests
**When to use:** Testing aether-utils.sh subcommands
**Example:**
```bash
# Source: tests/e2e/test-install.sh (existing pattern)
#!/usr/bin/env bash
set -euo pipefail

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

# Counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

log_test_start() {
    TESTS_RUN=$((TESTS_RUN + 1))
    echo "TEST $TESTS_RUN: $1"
}

log_test_pass() {
    TESTS_PASSED=$((TESTS_PASSED + 1))
    echo -e "${GREEN}✓ PASS${NC}: $1"
}

log_test_fail() {
    TESTS_FAILED=$((TESTS_FAILED + 1))
    echo -e "${RED}✗ FAIL${NC}: $1"
    echo "  Expected: $2"
    echo "  Got: $3"
}

assert_json_valid() {
    local output="$1"
    if echo "$output" | jq -e . >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

assert_json_field_equals() {
    local output="$1"
    local field="$2"
    local expected="$3"
    local actual=$(echo "$output" | jq -r "$field")
    if [[ "$actual" == "$expected" ]]; then
        return 0
    else
        return 1
    fi
}

# Test example
log_test_start "help returns valid JSON"
output=$(bash .aether/aether-utils.sh help)
if assert_json_valid "$output"; then
    log_test_pass "help returns valid JSON"
else
    log_test_fail "help returns valid JSON" "valid JSON" "$output"
fi
```

### Pattern 4: Chronological Timestamp Validation
**What:** Parse ISO 8601 timestamps and verify ascending order
**When to use:** Validating events array in COLONY_STATE.json
**Example:**
```javascript
// Source: Native JavaScript Date parsing
test('events are in chronological order', t => {
  const state = JSON.parse(fs.readFileSync(COLONY_STATE_PATH, 'utf8'));

  for (let i = 1; i < state.events.length; i++) {
    const prev = new Date(state.events[i-1].timestamp);
    const curr = new Date(state.events[i].timestamp);

    // Validate both are valid dates
    t.false(isNaN(prev.getTime()), `Event ${i-1} has invalid timestamp`);
    t.false(isNaN(curr.getTime()), `Event ${i} has invalid timestamp`);

    // Check chronological order
    t.true(curr >= prev, `Event ${i} (${state.events[i].timestamp}) is before event ${i-1} (${state.events[i-1].timestamp})`);
  }
});
```

### Anti-Patterns to Avoid
- **Using JSON.parse() alone for duplicate detection:** Native parser allows duplicates (last one wins)
- **Complex bash test frameworks:** Adds unnecessary dependencies; simple assertions suffice
- **Testing all 59 subcommands:** Out of scope per CONTEXT.md; focus on critical paths only
- **Modifying existing test files:** Keep existing tests in `test/` directory unchanged

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JSON parsing | Custom parser | Native JSON.parse | Native is optimized and standard-compliant |
| Test runner | Custom test harness | AVA | Mature, parallel execution, good reporting |
| JSON validation in bash | Custom regex | jq | Industry standard, handles edge cases |
| Date comparison | String comparison | Date objects | ISO 8601 strings may not sort correctly lexically |

**Key insight:** The duplicate key detection is the one area requiring custom logic since standard parsers intentionally allow duplicates per RFC 8259.

## Common Pitfalls

### Pitfall 1: Duplicate Key Blindness
**What goes wrong:** Assuming JSON.parse() will throw on duplicate keys
**Why it happens:** RFC 8259 says names "SHOULD be unique" but parsers aren't required to reject duplicates
**How to avoid:** Use custom detection with reviver function or regex patterns
**Warning signs:** Tests pass even when JSON has duplicates; silent data loss

### Pitfall 2: Timestamp String Comparison
**What goes wrong:** Comparing ISO 8601 strings lexically instead of as dates
**Why it happens:** "2026-02-13T11:00:00Z" > "2026-02-13T09:00:00Z" works but "2026-02-13T11:00:00Z" < "2026-02-13T2:00:00Z" fails
**How to avoid:** Always parse to Date objects before comparison
**Warning signs:** Tests pass/fail inconsistently with different timestamp formats

### Pitfall 3: Bash Test Isolation Failures
**What goes wrong:** Tests pollute each other's state (files, environment variables)
**Why it happens:** No automatic isolation in simple bash tests
**How to avoid:** Use setup/teardown functions, temp directories, unset variables
**Warning signs:** Tests pass individually but fail in suite; flaky tests

### Pitfall 4: AVA ESM/CommonJS Mismatch
**What goes wrong:** Tests fail with module format errors
**Why it happens:** AVA 6 defaults to ESM but project uses CommonJS
**How to avoid:** Configure AVA for CommonJS or add "type": "module" to package.json
**Warning signs:** "Cannot use import statement outside a module" errors

## Code Examples

### AVA Configuration (package.json)
```json
{
  "scripts": {
    "test": "ava",
    "test:unit": "ava",
    "test:bash": "bash tests/bash/test-aether-utils.sh"
  },
  "devDependencies": {
    "ava": "^6.0.0"
  },
  "ava": {
    "files": [
      "tests/unit/**/*.test.js"
    ],
    "timeout": "60s"
  }
}
```

### Duplicate Key Detection Test
```javascript
const test = require('ava');
const fs = require('fs');
const path = require('path');

const COLONY_STATE_PATH = path.join(__dirname, '../../.aether/data/COLONY_STATE.json');

// Detect duplicates using line-by-line parsing
function findDuplicateKeys(jsonString) {
  const lines = jsonString.split('\n');
  const duplicates = [];
  const stack = []; // Track object keys at each nesting level

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const keyMatch = line.match(/^\s*"([^"]+)"\s*:/);

    if (keyMatch) {
      const key = keyMatch[1];
      const indent = line.match(/^\s*/)[0].length;

      // Find current object level
      while (stack.length > 0 && stack[stack.length - 1].indent >= indent) {
        stack.pop();
      }

      // Check for duplicate at this level
      const currentObj = stack[stack.length - 1];
      if (currentObj && currentObj.keys.has(key)) {
        duplicates.push({ key, line: i + 1 });
      } else if (currentObj) {
        currentObj.keys.add(key);
      }
    }

    // Track object start/end
    if (line.includes('{')) {
      stack.push({ indent: line.match(/^\s*/)[0].length, keys: new Set() });
    }
    if (line.includes('}')) {
      stack.pop();
    }
  }

  return duplicates;
}

test('COLONY_STATE.json has no duplicate keys', t => {
  const content = fs.readFileSync(COLONY_STATE_PATH, 'utf8');
  const duplicates = findDuplicateKeys(content);
  t.deepEqual(duplicates, [], `Found duplicate keys at lines: ${duplicates.map(d => d.line).join(', ')}`);
});
```

### Bash Test Helper (test-helpers.sh)
```bash
#!/usr/bin/env bash
# Test helper functions for bash integration tests

set -euo pipefail

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

log() {
    echo -e "${NC}[$(date +'%H:%M:%S')] $1${NC}"
}

log_test_start() {
    TESTS_RUN=$((TESTS_RUN + 1))
    log "${YELLOW}TEST $TESTS_RUN: $1${NC}"
}

log_test_pass() {
    TESTS_PASSED=$((TESTS_PASSED + 1))
    log "${GREEN}✓ PASS${NC}: $1"
}

log_test_fail() {
    TESTS_FAILED=$((TESTS_FAILED + 1))
    log "${RED}✗ FAIL${NC}: $1"
    log "${RED}  Expected: $2${NC}"
    log "${RED}  Got: $3${NC}"
}

assert_json_valid() {
    local output="$1"
    echo "$output" | jq -e . >/dev/null 2>&1
}

assert_json_field_equals() {
    local output="$1"
    local field="$2"
    local expected="$3"
    local actual=$(echo "$output" | jq -r "$field // empty")
    [[ "$actual" == "$expected" ]]
}

assert_exit_code() {
    local actual="$1"
    local expected="$2"
    [[ "$actual" -eq "$expected" ]]
}

test_summary() {
    log ""
    log "${YELLOW}=== Test Summary ===${NC}"
    log "Tests run:    $TESTS_RUN"
    log "${GREEN}Tests passed: $TESTS_PASSED${NC}"
    if [[ "$TESTS_FAILED" -gt 0 ]]; then
        log "${RED}Tests failed: $TESTS_FAILED${NC}"
        return 1
    else
        log "Tests failed: $TESTS_FAILED"
        return 0
    fi
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Custom test harness | AVA 6.x | 2024 | Better parallel execution, ESM native |
| BATS framework | Simple assertions | Project-specific | Fewer dependencies, matches existing pattern |
| Moment.js | Native Date | 2020+ | Moment deprecated, native Date sufficient for ISO 8601 |
| JSON.parse for validation | Custom duplicate detection | N/A | Standard parsers allow duplicates by design |

**Deprecated/outdated:**
- Moment.js for date parsing: Use native Date or date-fns
- BATS for simple tests: Use simple assertions to avoid dependency
- Manual test counting: AVA provides built-in reporting

## Open Questions

1. **AVA Configuration for CommonJS**
   - What we know: Project uses CommonJS (require/module.exports)
   - What's unclear: Whether to configure AVA for CommonJS or migrate to ESM
   - Recommendation: Configure AVA for CommonJS to match existing codebase

2. **Existing Test Compatibility**
   - What we know: Three existing tests in `test/` directory use custom harness
   - What's unclear: Whether to migrate them to AVA or keep as-is
   - Recommendation: Keep existing tests unchanged per "existing tests pass" requirement

3. **Oracle Bug Current Status**
   - What we know: Oracle reported duplicate keys and timestamp issues in archived version
   - What's unclear: Whether current COLONY_STATE.json has these issues
   - Recommendation: Audit current file and fix if needed

## Sources

### Primary (HIGH confidence)
- AVA documentation: https://github.com/avajs/ava (v6.x, CommonJS support)
- RFC 8259: JSON standard specifying duplicate key behavior
- Existing codebase: tests/e2e/test-install.sh (proven bash test pattern)

### Secondary (MEDIUM confidence)
- Web search: Bash testing patterns 2025 (BATS, bashunit, shest comparison)
- Web search: JSON duplicate key detection approaches in JavaScript
- Web search: Node.js timestamp validation techniques

### Tertiary (LOW confidence)
- Oracle findings from 02-CONTEXT.md (specific bug locations may be in archived files)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - AVA is well-documented, bash patterns proven in existing tests
- Architecture: HIGH - Clear separation between unit (AVA) and integration (bash) tests
- Pitfalls: MEDIUM - Based on common JavaScript/JSON behaviors and bash testing experience

**Research date:** 2026-02-13
**Valid until:** 30 days for AVA (stable), 90 days for bash patterns (very stable)
