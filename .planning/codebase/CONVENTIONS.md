# Coding Conventions

**Analysis Date:** 2026-02-17

## Naming Patterns

**Files:**
- Shell scripts: `kebab-case.sh` (e.g., `aether-utils.sh`, `file-lock.sh`)
- JavaScript modules: `camelCase.js` (e.g., `state-guard.js`, `model-profiles.js`)
- Markdown docs: `kebab-case.md` (e.g., `coding-standards.md`)
- Test files: `{module-name}.test.js` (e.g., `cli-hash.test.js`)

**Functions:**
- JavaScript: `camelCase` (e.g., `hashFileSync`, `validateManifest`)
- Shell: `snake_case` (e.g., `get_caste_emoji`, `json_ok`)
- Private methods: prefix with underscore `_privateMethod()`

**Variables:**
- JavaScript: `camelCase` (e.g., `currentVersion`, `globalQuiet`)
- Shell: `snake_case` (e.g., `aether_root`, `lock_acquired`)
- Constants: `UPPER_SNAKE_CASE` for compile-time constants (e.g., `ErrorCodes`)
- Private variables: prefix with underscore `_privateVar`

**Types:**
- Classes: `PascalCase` (e.g., `AetherError`, `StateGuard`, `FeatureFlags`)
- Error classes: suffix with `Error` (e.g., `HubError`, `ValidationError`)
- Interfaces: Not explicitly used; objects serve as implicit contracts

## Code Style

**Formatting:**
- No explicit formatter configured (no Prettier, ESLint found)
- Manual formatting; consistent 2-space indentation
- JavaScript: no semicolons at statement ends (modern style)
- Shell: `set -euo pipefail` at script top

**Linting:**
- Shell scripts: `shellcheck` with `--severity=error`
- JSON: validated via `node -e "JSON.parse(...)"`
- Sync: `bash bin/generate-commands.sh check`
- Run lint: `npm run lint`

**Key shellcheck rules enforced:**
- `set -euo pipefail` required
- Variables must be quoted: `"$var"` not `$var`
- Use `[[ ]]` for tests, not `[ ]`
- Check for required tools before using: `command -v foo >/dev/null || exit 1`

## Import Organization

**Order in JavaScript files:**
1. External dependencies (e.g., `require('commander')`)
2. Internal lib imports (e.g., `require('./lib/errors')`)
3. Module exports (at end of file)

**Path aliases:**
- None configured; relative paths used throughout
- Example: `const { logError } = require('./lib/logger');`

**Example from `/Users/callumcowie/repos/Aether/bin/cli.js`:**
```javascript
const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const { execSync } = require('child_process');
const { program } = require('commander');

// Error handling imports
const {
  AetherError,
  HubError,
  // ...
} = require('./lib/errors');
const { logError, logActivity } = require('./lib/logger');
// ... more imports
```

## Error Handling

**JavaScript Error Pattern:**
- Hierarchical error classes extending `AetherError` (defined in `/Users/callumcowie/repos/Aether/bin/lib/errors.js`)
- Each error has: `code`, `message`, `details`, `recovery`, `timestamp`
- Use `wrapError()` to convert plain errors to structured format
- Exit codes mapped via `getExitCode()` (uses sysexits.h conventions)

**Error Class Hierarchy:**
```javascript
AetherError (base)
├── HubError
├── RepoError
├── GitError
├── ValidationError
├── FileSystemError
├── ConfigurationError
└── StateSchemaError
```

**Shell Error Pattern:**
- Use `json_err()` for JSON error output (exits 1)
- Use `json_ok()` for success output
- Constants defined in error-handler.sh: `$E_UNKNOWN`, `$E_HUB_NOT_FOUND`, etc.
- Trap ERR for structured error context: `trap 'error_handler ...' ERR`

**Example from `/Users/callumcowie/repos/Aether/bin/lib/errors.js`:**
```javascript
class ValidationError extends AetherError {
  constructor(message, details = {}) {
    super(
      ErrorCodes.E_INVALID_STATE,
      message,
      details,
      'Check the state file and fix validation errors'
    );
    this.name = 'ValidationError';
  }
}
```

## Logging

**Framework:** Custom logger in `/Users/callumcowie/repos/Aether/bin/lib/logger.js`

**Log Functions:**
- `logError(structuredError)` - Logs to activity.log
- `logActivity(message)` - Logs activity events

**Shell Logging:**
- Use `echo` with color codes from `colors.js`
- Prefix with `log_` for logging functions: `log_error()`, `log_info()`

**Patterns:**
- JSON structured logging for errors (machine-parseable)
- Human-readable for CLI output (with color codes)

## Comments

**When to Comment:**
- Document public API functions with JSDoc-style comments
- Explain non-obvious logic or workarounds
- Note WHY something is done, not just WHAT
- TODO comments for incomplete work

**JSDoc/TSDoc:**
- Used for public functions and classes
- Example from `/Users/callumcowie/repos/Aether/bin/lib/errors.js`:
```javascript
/**
 * Base AetherError class
 * All application errors extend this class for consistent handling
 * @param {string} code - Error code from ErrorCodes
 * @param {string} message - Human-readable error message
 * @param {object} details - Additional error context
 * @param {string|null} recovery - Recovery suggestion for user
 */
```

**Shell Comments:**
- Header comment with purpose at script top
- Inline comments for complex logic: `# Explanation of what follows`

## Function Design

**Size:**
- Keep functions under 50 lines
- Extract helpers for repeated patterns
- One function per logical unit

**Parameters:**
- JavaScript: Destructure objects for clarity: `function({ foo, bar })`
- Shell: Use local variables: `local var_name="$1"`

**Return Values:**
- JavaScript: Return structured objects for complex results
- Shell: JSON via stdout for structured data, exit codes for status

## Module Design

**Exports:**
- Use CommonJS: `module.exports = { ... }` or `module.exports = ClassName`
- Named exports for utilities: `module.exports = { func1, func2, ClassName }`

**Barrel Files:**
- Not used; imports directly from modules

**JavaScript Module Structure:**
```javascript
// Constants at top
const VERSION = require('../package.json').version;

// Error class definitions
class MyClass { ... }

// Helper functions
function helper() { ... }

// Main export
module.exports = {
  MyClass,
  helper,
};
```

---

*Convention analysis: 2026-02-17*
