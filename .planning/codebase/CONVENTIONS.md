# Coding Conventions

**Analysis Date:** 2026-03-19

## Naming Patterns

**Files:**
- Kebab-case for all files (e.g., `file-lock.js`, `state-sync.js`, `model-profiles.js`)
- Single responsibility: file name reflects primary class or module (e.g., `errors.js` for error classes, `colors.js` for color utilities)
- Test files follow pattern: `{module-name}.test.js` (e.g., `file-lock.test.js`)

**Functions:**
- camelCase for all function names
- Descriptive verb-first names: `loadModelProfiles()`, `createCheckpoint()`, `getActivityLogPath()`
- Helper functions with clear intent: `sanitizeForLog()`, `formatTimestamp()`, `validateStateSchema()`
- Private helpers follow same naming (JavaScript doesn't enforce privacy, use comments to indicate internal use)

**Variables:**
- camelCase for variables and parameters (e.g., `mockFs`, `testDir`, `staleTimeout`)
- SCREAMING_SNAKE_CASE for constants (e.g., `DEFAULT_MODEL`, `MAX_RETRIES`, `PACKAGE_DIR`, `HOME`)
- Descriptive names: avoid single letters except in loops (e.g., `for (let i = 0; i < events.length; i++)`)

**Types & Classes:**
- PascalCase for class names (e.g., `AetherError`, `FileLock`, `UpdateTransaction`, `ValidationError`)
- Extends pattern: specific error types extend base (e.g., `RepoError extends AetherError`)
- Enum-like objects in camelCase (e.g., `ErrorCodes`, `TransactionStates`, `EMOJI`)

## Code Style

**Formatting:**
- 2-space indentation (enforced in module code)
- Line length: no strict limit, but keep functional blocks readable
- Semicolons required (always present in code)
- No external formatter (eslint or prettier not used)

**Linting:**
- ShellCheck for bash files: `npm run lint:shell` (severity=error)
- JSON validation: `npm run lint:json` for state and constraint files
- Sync validation: `npm run lint:sync` verifies Claude/OpenCode command parity

**Import Organization:**
- Node.js built-in modules first (e.g., `fs`, `path`, `crypto`, `child_process`)
- Third-party modules second (e.g., `commander`, `js-yaml`, `proxyquire`, `sinon`)
- Local modules last (e.g., `require('./lib/errors')`)
- Comments separate import groups (example from `cli.js` lines 3-50)

```javascript
// Built-ins
const fs = require('fs');
const path = require('path');

// Third-party
const { program } = require('commander');

// Local
const { AetherError } = require('./lib/errors');
```

## Error Handling

**Patterns:**
- All errors extend `AetherError` from `bin/lib/errors.js`
- Specific error types for different failure domains:
  - `HubError` — hub distribution issues
  - `RepoError` — repository initialization
  - `GitError` — git operations
  - `ValidationError` — state validation
  - `FileSystemError` — file operations
  - `ConfigurationError` — config/env issues
  - `UpdateError` — package updates

**Structured error format:**
- `code`: ErrorCodes enum value (e.g., `E_UPDATE_FAILED`)
- `message`: user-readable explanation
- `details`: context object with diagnostic info
- `recovery`: suggested recovery command or action
- `timestamp`: ISO string of when error occurred

Example from `bin/lib/errors.js` (lines 49-61):
```javascript
constructor(code, message, details = {}, recovery = null) {
  super(message);
  this.name = 'AetherError';
  this.code = code;
  this.details = details;
  this.recovery = recovery;
  this.timestamp = new Date().toISOString();
  // ...
}
```

**Error output:**
- Always output structured JSON: `JSON.stringify(error.toJSON(), null, 2)`
- Silent failures for logging (fail-safe in `logActivity()`)
- Errors logged to `.aether/data/activity.log` via `logError()`

## Logging

**Framework:** Node.js `console` + custom structured logging via `bin/lib/logger.js`

**Patterns:**
- All logs go to `.aether/data/activity.log` (centralized activity file)
- Log levels: `logError()`, `logActivity()`, `logSpawn()`
- No console output except errors and final results (silent by default)
- `--quiet` flag suppresses non-error output

**Usage:**
```javascript
const { logError, logActivity } = require('./lib/logger');

logActivity('builder spawned for phase 2');
logError(structuredError); // logs + appends to activity.log
```

**Sanitization:**
- All log entries sanitized via `sanitizeForLog()` (bin/lib/logger.js lines 62-71)
- Removes newlines, control characters, limits to 200 chars
- Prevents log injection and corruption

## Comments

**When to Comment:**
- JSDoc blocks for all exported functions and classes (see `bin/lib/errors.js`, `bin/lib/file-lock.js`)
- Complex algorithms explained (e.g., duplicate key detection in `tests/unit/colony-state.test.js`)
- Non-obvious business logic (e.g., lock acquisition retry logic)
- Do NOT comment obvious code (e.g., `const name = 'builder';` doesn't need a comment)

**JSDoc/TSDoc:**
- Format: `/** @param {type} name - description */`
- Always include for public functions
- Always include `@returns {type}` for functions with return values
- Example from `bin/lib/model-profiles.js` (lines 20-23):

```javascript
/**
 * Load and parse model profiles from YAML file
 * @param {string} repoPath - Path to repository root
 * @returns {object} Parsed model profiles
 * @throws {ConfigurationError} If file not found or invalid YAML
 */
function loadModelProfiles(repoPath) {
```

**Special Comments:**
- `@module path/to/file` — identifies module purpose
- `@private` — indicates internal-only functions
- `NOTE:` — design decisions or known limitations
- Section dividers for large files: `// ============================================================================`

## Function Design

**Size:**
- Target 30-50 lines for most functions
- Larger functions (100-200+ lines) acceptable only if:
  - Single responsibility (e.g., transaction state management)
  - Well-structured with section comments
  - Fully tested and stable
- Example: `UpdateTransaction.createCheckpoint()` in `bin/lib/update-transaction.js` is 150+ lines but handles one complete flow

**Parameters:**
- Max 3-4 positional parameters recommended
- Use options object for 4+ parameters: `function init(repoPath, options = {})`
- Always provide sensible defaults (e.g., `{ quiet: false, timeout: 30000 }`)
- Type-check early: validate inputs at function start

**Return Values:**
- Functions return single value or object
- Use objects to return multiple values: `{ success, checkpoint, error }`
- Promise-based: use `async`/`await` (see `bin/lib/file-lock.js` acquireAsync)
- Null/undefined for "not found" (preferred over throwing)

## Module Design

**Exports:**
- Use `module.exports = { function1, function2, Class1 }` (destructurable)
- One class per file when class is primary export
- Helper functions grouped with main export in same file
- Example from `bin/lib/errors.js`: exports error classes + `getExitCode()` helper

**Barrel Files:**
- Not used in this codebase (each module is independent)
- CLI imports directly from `bin/lib/` files

**File Organization:**
- Order within file: constants → helper functions → main class/functions → exports
- Example from `bin/lib/colors.js`:
  1. JSDoc header
  2. Require statements
  3. Helper functions (isColorEnabled)
  4. Constants (color palette)
  5. module.exports

## Async/Promise Patterns

**Synchronous is default:**
- Most codebase uses sync file operations
- Matches Node.js 16+ compat requirement

**Async when needed:**
- Lock acquisition: `FileLock.acquireAsync()` (yields to event loop)
- HTTP calls: `checkLiteLLMProxy()` uses `fetch()` with async/await
- Test lifecycle: `test.beforeEach()`, `test.afterEach()` support async

**Example from `bin/lib/file-lock.js` (lines 304-320):**
```javascript
async acquireAsync(filePath) {
  const startTime = Date.now();
  while (Date.now() - startTime < this.timeout) {
    if (this.acquire(filePath)) {
      return true;
    }
    // Async delay (yields to event loop)
    await new Promise(resolve => setTimeout(resolve, this.retryInterval));
  }
  return false; // Timeout
}
```

## Standard Patterns Used

**State Guard Pattern:**
- Validates state before operations in `bin/lib/state-guard.js`
- Prevents invalid state transitions
- Returns validation result with errors array

**File Lock Pattern:**
- PID-based lock files in `.aether/locks/`
- Atomic lock acquisition with exclusive flag
- Stale lock detection and cleanup

**Transaction Pattern:**
- Checkpoint before destructive operations
- State tracking (PENDING → IN_PROGRESS → COMPLETE)
- Error recovery with stash pop commands

---

*Convention analysis: 2026-03-19*
