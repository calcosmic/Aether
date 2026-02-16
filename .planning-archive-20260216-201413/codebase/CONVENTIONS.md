# Coding Conventions

**Analysis Date:** 2026-02-13

## Overview

Aether is a multi-agent orchestration system built primarily in Bash and Node.js. The codebase follows strict quality standards documented in `runtime/coding-standards.md` and enforced through linting rules in `package.json`.

## Languages

**Primary:**
- Bash - Core orchestration, utility functions, E2E tests
- Node.js - CLI tool, installation/update logic

**Configuration:**
- JSON - State management, manifests, registry
- Markdown - Documentation, slash commands

## Naming Patterns

**Shell Scripts:**
- Use kebab-case: `aether-utils.sh`, `file-lock.sh`, `atomic-write.sh`
- Utility functions: snake_case (e.g., `acquire_lock`, `atomic_write`, `json_ok`)
- Local variables: snake_case (e.g., `local file_path`, `local target_dir`)
- Constants: UPPER_SNAKE_CASE (e.g., `LOCK_TIMEOUT`, `MAX_BACKUPS`)

**Node.js:**
- Functions: camelCase (e.g., `copyDirSync`, `hashFileSync`, `setupHub`)
- Variables: camelCase (e.g., `hubVersion`, `sourceVersion`)
- Constants: UPPER_SNAKE_CASE or camelCase for module-level (e.g., `COMMANDS_SRC`, `HUB_DIR`)

**Files:**
- Shell scripts: `*.sh` with kebab-case names
- Node.js: `*.js` with kebab-case or camelCase
- Tests: `*.test.js` or `test-*.sh`
- Markdown docs: `UPPERCASE.md` or `kebab-case.md`

## Code Style

**Shell Formatting:**
- Enforced via `shellcheck` with severity=error
- Run: `npm run lint:shell`
- Key rules from `runtime/coding-standards.md`:
  - Functions < 50 lines
  - No deep nesting (use early returns)
  - No magic numbers (use named constants)

**JavaScript Formatting:**
- Standard Node.js conventions
- No explicit formatter configured (follow existing patterns)

**Linting:**
```bash
npm run lint:shell  # ShellCheck for shell scripts
npm run lint:json   # JSON validation
npm run lint:sync   # Command sync verification
npm run lint        # All linters
```

## Error Handling

**Shell Scripts:**
- Use `set -euo pipefail` at script start for strict mode
- JSON errors via `json_err()` function:
  ```bash
  json_err() { printf '{"ok":false,"error":"%s"}\n' "$1" >&2; exit 1; }
  ```
- JSON success via `json_ok()`:
  ```bash
  json_ok() { printf '{"ok":true,"result":%s}\n' "$1"; }
  ```
- All subcommands return JSON to stdout (success) or stderr (error)
- Non-zero exit on error

**Node.js:**
- Try-catch for file operations
- Graceful degradation with warning messages
- Exit code 1 on critical failures

**Atomic Operations:**
- File writes use atomic pattern (temp file + rename)
- See `runtime/utils/atomic-write.sh` for implementation
- JSON validation before atomic rename for .json files

## Logging

**Shell:**
- Activity logging via `activity-log` subcommand
- Format: `[HH:MM:SS] <emoji> <action> <caste>: <description>`
- Colorized output using ANSI codes:
  ```bash
  GREEN='\033[0;32m'
  RED='\033[0;31m'
  YELLOW='\033[1;33m'
  NC='\033[0m'  # No Color
  ```

**Node.js:**
- Console output with quiet mode support (`--quiet` flag)
- Structured messages for user feedback

## Comments

**When to Comment:**
- Explain WHY, not WHAT (per `runtime/coding-standards.md`)
- Document complex algorithms or business logic
- JSDoc-style for public functions is acceptable but not required

**Examples:**
```bash
# GOOD: Explain WHY
# Use exponential backoff to avoid overwhelming the API

# BAD: Stating the obvious
# Increment counter by 1
```

**Anti-Pattern Detection:**
The `check-antipattern` subcommand flags TODO/FIXME comments as warnings.

## Function Design

**Size Guidelines (from `runtime/coding-standards.md`):**
- Functions should be < 50 lines
- Use early returns to avoid deep nesting
- One responsibility per function

**Parameters:**
- Validate required arguments at function start
- Use default values for optional parameters
- Return structured JSON from utility functions

**Return Values:**
- Shell utilities return JSON to stdout
- Exit codes: 0 for success, non-zero for failure
- Errors go to stderr as JSON

## Module Design

**Exports:**
- Shell functions exported via `export -f function_name`
- Node.js uses CommonJS (`module.exports` implicit via function definitions)

**Barrel Files:**
- `.aether/aether-utils.sh` is the main entry point (source of truth)
- Sources utilities from `runtime/utils/*.sh`
- Subcommand dispatch via case statement

**Import Pattern (Shell):**
```bash
[[ -f "$SCRIPT_DIR/utils/file-lock.sh" ]] && source "$SCRIPT_DIR/utils/file-lock.sh"
[[ -f "$SCRIPT_DIR/utils/atomic-write.sh" ]] && source "$SCRIPT_DIR/utils/atomic-write.sh"
```

## File Organization

**Key Locations:**
- Main utilities: `.aether/aether-utils.sh` (source of truth), auto-synced to `runtime/`
- Helper utilities: `runtime/utils/*.sh`
- CLI: `bin/cli.js`
- Tests: `test/*.js` and `tests/e2e/*.sh`
- Documentation: `runtime/*.md`

**State Files:**
- `.aether/data/COLONY_STATE.json` - Colony state
- `.aether/data/flags.json` - Issue tracking
- `.aether/data/learnings.json` - Cross-session learnings

## Security Patterns

**Secrets Detection:**
- `check-antipattern` subcommand flags exposed secrets
- Pattern: `(api_key|apikey|secret|password|token)\s*=\s*['\"][^'\"]+['\"]`

**File Permissions:**
- Shell scripts get executable bit (0o755) during copy
- Lock files use PID tracking for stale lock detection

## Immutability

Per `runtime/coding-standards.md`:
- Always use spread operator for updates
- Never mutate objects directly
- Example:
  ```javascript
  // GOOD
  const updated = { ...obj, field: 'new' };

  // BAD
  obj.field = 'new';
  ```

## Type Safety

From `runtime/coding-standards.md`:
- Avoid `any` type in TypeScript contexts
- Use proper type annotations
- Function signatures should specify return types

## Testing Standards

See `runtime/tdd.md` and `TESTING.md` for testing conventions.

## Key Files for Reference

- `runtime/coding-standards.md` - Primary coding standards document
- `.aether/aether-utils.sh` - Reference implementation for shell patterns (source of truth)
- `bin/cli.js` - Reference implementation for Node.js patterns
- `package.json` - Lint scripts and configuration

---

*Convention analysis: 2026-02-13*
