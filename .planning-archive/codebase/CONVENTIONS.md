# Coding Conventions

**Analysis Date:** 2026-02-17

## Language Overview

This codebase uses two primary languages:
- **JavaScript** (Node.js) - CLI logic, state management, testing infrastructure
- **Bash** - Shell utilities, colony operations, build automation

## JavaScript Conventions

### File Organization

**Location:** `bin/lib/` for CLI modules, `tests/unit/` for unit tests

**Naming:**
- Modules: `camelCase.js` (e.g., `state-guard.js`, `errors.js`)
- Test files: `*.test.js` (e.g., `state-guard.test.js`)
- Helpers: `camelCase.js` (e.g., `mock-fs.js`)

### Code Style

**Variables:**
- Use `const` by default, `let` only when reassignment is required
- Never use `var`

```javascript
// Good
const CONFIG_PATH = '/path/to/config';
let currentPhase = 5;

// Bad
var config = 'value';
```

**Functions:**
- Prefer `async/await` over raw Promises or callback patterns
- Use named function declarations for clarity
- Document public APIs with JSDoc-style comments

```javascript
/**
 * Load and parse state file
 * @returns {object} Parsed state object
 * @throws {StateGuardError} If file missing or invalid
 */
async function loadState(filePath) {
  // Implementation
}
```

**String Interpolation:**
- Use template literals for string interpolation

```javascript
// Good
const message = `Phase ${current} transitioned to ${next}`;

// Bad
const message = 'Phase ' + current + ' transitioned to ' + next;
```

**Error Handling:**
- Use structured error classes from `bin/lib/errors.js`
- All errors extend `AetherError` base class
- Include recovery suggestions in error messages

```javascript
const { AetherError, ValidationError, ErrorCodes } = require('./errors');

throw new ValidationError(
  'Invalid state file structure',
  { path: stateFile, missingFields: ['version', 'current_phase'] },
  'Restore from backup or reinitialize'
);
```

### Import Organization

**Order:**
1. Node.js built-ins (`fs`, `path`, `crypto`)
2. External packages (`commander`, `sinon`, `proxyquire`)
3. Local modules (`./errors`, `../lib/state-guard`)

```javascript
const fs = require('fs');
const path = require('path');
const { program } = require('commander');
const sinon = require('sinon');
const proxyquire = require('proxyquire');
const { AetherError } = require('./errors');
const { StateGuard } = require('./state-guard');
```

### Class Patterns

**Structure:**
- JSDoc comment with class purpose
- Constructor with explicit parameter documentation
- Methods with JSDoc including @returns and @throws
- Use `Error.captureStackTrace` for proper stack traces

```javascript
/**
 * StateGuard - Enforces Iron Law and manages phase transitions
 */
class StateGuard {
  /**
   * @param {string} stateFilePath - Path to COLONY_STATE.json
   * @param {object} options - Configuration options
   */
  constructor(stateFilePath, options = {}) {
    this.stateFile = stateFilePath;
    this.locked = false;

    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, StateGuard);
    }
  }
}
```

## Shell Script Conventions

### File Organization

**Location:** `.aether/` for utilities, `.aether/utils/` for helper scripts

**Naming:**
- Scripts: `kebab-case.sh` (e.g., `aether-utils.sh`, `file-lock.sh`)
- Functions: `snake_case` (e.g., `get_caste_emoji`, `json_ok`)

### Code Style

**Shebang and Strict Mode:**
```bash
#!/bin/bash
set -euo pipefail
```

**Variable Handling:**
- Always quote variables: `"$var"` not `$var`
- Use `${var}` for clarity in complex expressions
- Declare local variables with `local`

```bash
# Good
local target_file="$1"
if [[ -f "$target_file" ]]; then
  echo "File exists: $target_file"
fi

# Bad
if [ -f $1 ]; then
  echo File exists: $1
fi
```

**Conditionals:**
- Use `[[ ]]` for tests, never `[ ]`
- Prefer `[[ -z "$var" ]]` over `[[ "$var" == "" ]]`

```bash
# Good
if [[ -z "$input" ]]; then
  echo "No input provided"
fi

# Bad
if [ "$input" == "" ]; then
  echo "No input provided"
fi
```

**Tool Checking:**
- Check for required tools before using
- Use `command -v` for portability

```bash
command -v jq >/dev/null || exit 1
command -v git >/dev/null || echo "Warning: git not available"
```

**Error Handling:**
- Use trap for error context
- Define error constants

```bash
trap 'if type error_handler &>/dev/null; then error_handler ${LINENO} "$BASH_COMMAND" $?; fi' ERR

: "${E_UNKNOWN:=E_UNKNOWN}"
: "${E_FILE_NOT_FOUND:=E_FILE_NOT_FOUND}"
```

### Function Design

**Patterns:**
- Use `local` for all variables within functions
- Return JSON via `printf` for structured output
- Use `json_err` for error output

```bash
json_ok() { printf '{"ok":true,"result":%s}\n' "$1"; }

json_err() {
  local message="${2:-$1}"
  printf '{"ok":false,"error":"%s"}\n' "$message" >&2
  exit 1
}
```

## Naming Conventions Summary

| Type | Pattern | Example |
|------|---------|---------|
| JavaScript modules | camelCase.js | `state-guard.js` |
| Shell scripts | kebab-case.sh | `file-lock.sh` |
| Test files | *.test.js | `state-guard.test.js` |
| Markdown docs | kebab-case.md | `coding-standards.md` |
| JavaScript classes | PascalCase | `StateGuard` |
| JavaScript functions | camelCase | `loadState()` |
| Shell functions | snake_case | `get_caste_emoji()` |
| Constants | UPPER_SNAKE_CASE | `MAX_RETRIES` |
| Variables | camelCase (JS), snake_case (Shell) | `currentPhase`, `target_file` |

## Error Handling Strategy

**JavaScript:**
- All errors use structured `AetherError` hierarchy
- Each error includes: code, message, details, recovery
- Errors export `toJSON()` for structured logging
- Exit codes mapped to sysexits.h standards

**Shell:**
- Use `set -euo pipefail` for strict error propagation
- Define error constants (`E_*`)
- Output JSON errors to stderr
- Return success/failure via JSON `ok` field

## Documentation Patterns

**JSDoc Requirements:**
- All exported functions require JSDoc
- Include @param with type and description
- Include @returns with type
- Include @throws for documented exceptions

```javascript
/**
 * Advance phase with full guard enforcement
 * @param {number} fromPhase - Current phase number
 * @param {number} toPhase - Target phase number
 * @param {object} evidence - Verification evidence
 * @returns {Promise<object>} Result object with status
 * @throws {StateGuardError} If Iron Law is violated
 */
async function advancePhase(fromPhase, toPhase, evidence) {
  // Implementation
}
```

---

*Convention analysis: 2026-02-17*
