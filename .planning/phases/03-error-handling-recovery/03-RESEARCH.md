# Phase 3: Error Handling & Recovery - Research

**Researched:** 2026-02-13
**Domain:** Node.js CLI error handling, Bash error handling, Structured logging, Graceful degradation
**Confidence:** HIGH

## Summary

This research covers the implementation of Phase 3: Error Handling & Recovery, which involves implementing centralized error handling with graceful degradation for the Aether Colony CLI system.

The Aether system currently has ad-hoc error handling scattered across `bin/cli.js` (Node.js) and `.aether/aether-utils.sh` (Bash). This phase will consolidate error handling into centralized handlers that provide:
1. Structured error formats with codes, messages, details, and recovery suggestions
2. Consistent JSON error output from both Node.js and Bash components
3. Graceful degradation when optional features fail
4. Error logging to `activity.log`
5. User-friendly error messages with actionable recovery steps

**Primary recommendation:** Implement a centralized error class hierarchy in Node.js with a wrapper pattern, extend the existing `json_err` helper in Bash with structured error codes, and use feature flags for graceful degradation.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Native Node.js | >=16.0.0 | Error handling, process events | No dependencies, built-in error classes |
| Bash 3.2+ | system | Error handling via trap ERR | Universal POSIX support |
| jq | system | JSON error formatting in Bash | Industry standard for CLI JSON processing |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| process.on('uncaughtException') | native | Global error catch-all | Last-resort error handling |
| process.on('unhandledRejection') | native | Promise rejection handling | Async error catching |
| trap ERR in Bash | native | Command failure interception | Bash function error handling |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Native Error classes | Custom error library (verror, boom) | Adds dependency; native sufficient for CLI |
| console.error | Winston, Pino | Overkill for CLI tool; native console with JSON is sufficient |
| Bash trap ERR | set -e with wrappers | trap ERR provides more control and context |

## Architecture Patterns

### Recommended Project Structure
```
bin/
├── cli.js                    # Main CLI with centralized error handler
└── lib/
    ├── errors.js             # Error class hierarchy
    └── logger.js             # Structured logging utility
.aether/
├── aether-utils.sh           # Bash utilities with error handler
└── utils/
    └── error-handler.sh      # Shared bash error handling functions
```

### Pattern 1: Centralized Error Handler (Node.js)
**What:** A wrapper around command execution that catches and formats all errors consistently
**When to use:** All CLI commands in bin/cli.js
**Example:**
```javascript
// Source: Node.js best practices for CLI tools
class AetherError extends Error {
  constructor(code, message, details = {}, recovery = null) {
    super(message);
    this.name = 'AetherError';
    this.code = code;
    this.details = details;
    this.recovery = recovery;
    this.timestamp = new Date().toISOString();
  }

  toJSON() {
    return {
      error: {
        code: this.code,
        message: this.message,
        details: this.details,
        recovery: this.recovery,
        timestamp: this.timestamp
      }
    };
  }
}

// Error codes enum
const ErrorCodes = {
  // System errors (1-99)
  E_HUB_NOT_FOUND: 'E_HUB_NOT_FOUND',
  E_REPO_NOT_INITIALIZED: 'E_REPO_NOT_INITIALIZED',
  E_FILE_SYSTEM: 'E_FILE_SYSTEM',
  E_GIT_ERROR: 'E_GIT_ERROR',

  // Validation errors (100-199)
  E_INVALID_STATE: 'E_INVALID_STATE',
  E_MANIFEST_INVALID: 'E_MANIFEST_INVALID',

  // Runtime errors (200-299)
  E_UPDATE_FAILED: 'E_UPDATE_FAILED',
  E_LOCK_TIMEOUT: 'E_LOCK_TIMEOUT',
  E_ATOMIC_WRITE_FAILED: 'E_ATOMIC_WRITE_FAILED'
};

// Centralized error handler
function handleError(error, { logActivity = true } = {}) {
  const structuredError = error instanceof AetherError
    ? error
    : new AetherError(
        'E_UNEXPECTED',
        error.message,
        { stack: error.stack },
        'Please report this issue with the error details'
      );

  // Log to activity.log if enabled
  if (logActivity) {
    logErrorToActivity(structuredError);
  }

  // Output structured error
  console.error(JSON.stringify(structuredError.toJSON(), null, 2));

  // Exit with appropriate code
  process.exit(getExitCode(structuredError.code));
}

// Wrapper for command execution
function withErrorHandling(fn, options = {}) {
  return async (...args) => {
    try {
      return await fn(...args);
    } catch (error) {
      handleError(error, options);
    }
  };
}
```

### Pattern 2: Structured Bash Error Handler
**What:** Consistent JSON error output from Bash scripts using an enhanced json_err function
**When to use:** All aether-utils.sh subcommands
**Example:**
```bash
# Source: Existing aether-utils.sh pattern (lines 36-41)
# Current implementation:
# json_err() { printf '{"ok":false,"error":"%s"}\n' "$1" >&2; exit 1; }

# Enhanced implementation with structured errors:
json_err() {
  local code="${1:-E_UNKNOWN}"
  local message="${2:-Unknown error}"
  local details="${3:-{}}"
  local recovery="${4:-null}"
  local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

  printf '{"ok":false,"error":{"code":"%s","message":"%s","details":%s,"recovery":%s,"timestamp":"%s"}}\n' \
    "$code" "$message" "$details" "$recovery" "$timestamp" >&2

  # Log to activity.log if available
  if [[ -d "$DATA_DIR" ]]; then
    echo "[$timestamp] ❌ ERROR $code: $message" >> "$DATA_DIR/activity.log"
  fi

  exit 1
}

# Usage examples:
# json_err "E_HUB_NOT_FOUND" "Hub not found at ~/.aether" \
#   '{"path":"'"$HUB_DIR"'"}' \
#   '"Run: aether install"'
#
# json_err "E_REPO_NOT_INITIALIZED" "No .aether directory found" \
#   '{"path":"'"$repoPath"'"}' \
#   '"Run /ant:init in this repo first"'
```

### Pattern 3: Graceful Degradation with Feature Flags
**What:** Continue operation when optional features fail, with degraded functionality
**When to use:** Non-critical features like activity logging, progress display, optional git operations
**Example:**
```javascript
// Source: Existing cli.js pattern (lines 465-468) - graceful hub setup failure
// Current: try/catch with console.error warning
// Enhanced: Structured degradation with feature flags

class FeatureFlags {
  constructor() {
    this.features = {
      activityLog: true,
      progressDisplay: true,
      gitIntegration: true,
      hashComparison: true,
      manifestTracking: true
    };
    this.degradedFeatures = new Set();
  }

  disable(feature, reason) {
    this.features[feature] = false;
    this.degradedFeatures.add({ feature, reason, timestamp: new Date().toISOString() });

    // Log degradation
    console.warn(JSON.stringify({
      warning: {
        type: 'FEATURE_DEGRADED',
        feature,
        reason,
        timestamp: new Date().toISOString()
      }
    }));
  }

  isEnabled(feature) {
    return this.features[feature];
  }

  getDegradedFeatures() {
    return Array.from(this.degradedFeatures);
  }
}

// Usage in commands
const features = new FeatureFlags();

function setupHub() {
  try {
    // ... hub setup code ...
  } catch (err) {
    // Graceful degradation: disable hub-related features
    features.disable('manifestTracking', err.message);
    features.disable('hashComparison', err.message);

    // Continue with limited functionality
    return { status: 'degraded', reason: err.message };
  }
}

function updateRepo(repoPath, sourceVersion, opts) {
  // Check if git integration is available
  if (!isGitRepo(repoPath)) {
    features.disable('gitIntegration', 'Not a git repository');
  }

  // Skip git operations if disabled
  if (features.isEnabled('gitIntegration')) {
    // ... git safety checks ...
  }

  // Continue with file operations regardless
  // ...
}
```

### Pattern 4: Activity Log Integration
**What:** Structured error logging to activity.log with consistent format
**When to use:** All error conditions that should be persisted
**Example:**
```javascript
// Source: Existing activity.log format analysis
// Current format: [HH:MM:SS] $emoji $action $caste: $description
// Error format: [HH:MM:SS] ❌ ERROR $code: $message

function logErrorToActivity(error) {
  const logFile = path.join(DATA_DIR, 'activity.log');
  const timestamp = new Date().toISOString().split('T')[1].slice(0, 8); // HH:MM:SS
  const emoji = '❌';
  const code = error.code || 'E_UNKNOWN';
  const message = error.message.replace(/\n/g, ' '); // Single line

  const logLine = `[${timestamp}] ${emoji} ERROR ${code}: ${message}`;

  try {
    fs.appendFileSync(logFile, logLine + '\n');
  } catch (err) {
    // Silent fail - can't log logging errors
  }
}
```

### Pattern 5: Bash Trap ERR Pattern
**What:** Centralized error handling for Bash scripts using trap
**When to use:** aether-utils.sh and utility scripts
**Example:**
```bash
# Source: Bash best practices for error handling
# Add to aether-utils.sh after set -euo pipefail (line 10)

# Error handler function
error_handler() {
  local line=$1
  local command=$2
  local code=$3
  local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

  # Build structured error
  local error_json=$(printf '{"ok":false,"error":{"code":"E_BASH_ERROR","message":"Command failed","details":{"line":%d,"command":"%s","exit_code":%d},"timestamp":"%s"}}' \
    "$line" "$command" "$code" "$timestamp")

  echo "$error_json" >&2

  # Log to activity.log
  if [[ -d "$DATA_DIR" ]]; then
    echo "[$timestamp] ❌ ERROR E_BASH_ERROR: Command failed at line $line" >> "$DATA_DIR/activity.log"
  fi

  exit 1
}

# Set up trap (after existing set -euo pipefail)
trap 'error_handler ${LINENO} "$BASH_COMMAND" $?' ERR

# Note: This works alongside set -e but provides structured output
# Disable with: trap - ERR
```

### Anti-Patterns to Avoid
- **Swallowing errors silently:** Always log errors, even if continuing operation
- **Inconsistent error formats:** Mixing plain text and JSON errors breaks consumers
- **Exposing stack traces to users:** Include in details for debugging, not main message
- **Hardcoded error messages:** Use error codes for programmatic handling
- **Immediate exit on non-critical errors:** Use graceful degradation for optional features

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Error code mapping | Custom exit code logic | Error code enum with getExitCode() function | Consistent exit codes across CLI |
| JSON error formatting | String concatenation | JSON.stringify / jq | Proper escaping, valid JSON |
| Error logging rotation | Custom log rotation | Simple append with archive | Complexity not needed for activity.log |
| Process error handling | try/catch everywhere | Centralized handleError wrapper | DRY principle, consistent handling |
| Bash error context | Manual $?, $LINENO tracking | trap ERR with BASH_LINENO | Bash provides this automatically |

**Key insight:** The existing `json_ok`/`json_err` pattern in aether-utils.sh is the right approach - just needs enhancement for structured errors and recovery suggestions.

## Common Pitfalls

### Pitfall 1: Inconsistent Error Formats Between Node.js and Bash
**What goes wrong:** Node.js outputs `{error: {code, message, details}}` while Bash outputs `{ok: false, error: "string"}`
**Why it happens:** Different authors, no shared specification
**How to avoid:** Define shared error schema, implement in both environments
**Warning signs:** Consumers need conditional parsing based on error source

### Pitfall 2: Activity.log Write Failures Breaking Operations
**What goes wrong:** Error logging fails (disk full, permissions) and causes cascade failure
**Why it happens:** Error handling code throws errors
**How to avoid:** Wrap all logging in try/catch, silent fail on logging errors
**Warning signs:** Operations fail mysteriously when log directory has issues

### Pitfall 3: Bash ERR Trap Interfering with Expected Failures
**What goes wrong:** `trap ERR` catches expected failures (like `grep` not finding matches)
**Why it happens:** ERR trap fires on any command returning non-zero
**How to avoid:** Use `|| true` for expected failures, or check specific exit codes
**Warning signs:** Commands that should continue instead exit with error

### Pitfall 4: Graceful Degradation Hiding Critical Errors
**What goes wrong:** Feature is disabled silently, user doesn't know functionality is missing
**Why it happens:** Degradation warnings not visible or logged
**How to avoid:** Always log degradation, include in status output, require explicit acknowledgment for critical features
**Warning signs:** "Why didn't X happen?" - feature was silently disabled

### Pitfall 5: Error Message Injection Vulnerabilities
**What goes wrong:** User input in error messages allows JSON injection or log forging
**Why it happens:** Unescaped strings in JSON output or log files
**How to avoid:** Always escape strings for JSON context, sanitize log output
**Warning signs:** Error messages containing user input break JSON parsing

## Code Examples

### Node.js Error Class Hierarchy
```javascript
// bin/lib/errors.js
class AetherError extends Error {
  constructor(code, message, details = {}, recovery = null) {
    super(message);
    this.name = 'AetherError';
    this.code = code;
    this.details = details;
    this.recovery = recovery;
    this.timestamp = new Date().toISOString();
  }

  toJSON() {
    return {
      error: {
        code: this.code,
        message: this.message,
        details: this.details,
        recovery: this.recovery,
        timestamp: this.timestamp
      }
    };
  }
}

// Specific error types
class HubError extends AetherError {
  constructor(message, details = {}) {
    super('E_HUB_NOT_FOUND', message, details, 'Run: aether install');
  }
}

class RepoError extends AetherError {
  constructor(message, details = {}) {
    super('E_REPO_NOT_INITIALIZED', message, details, 'Run /ant:init in this repo first');
  }
}

class GitError extends AetherError {
  constructor(message, details = {}) {
    super('E_GIT_ERROR', message, details, 'Check git status and resolve conflicts');
  }
}

module.exports = { AetherError, HubError, RepoError, GitError };
```

### Enhanced Bash Error Handler
```bash
# .aether/utils/error-handler.sh

# Error codes
E_UNKNOWN="E_UNKNOWN"
E_HUB_NOT_FOUND="E_HUB_NOT_FOUND"
E_REPO_NOT_INITIALIZED="E_REPO_NOT_INITIALIZED"
E_FILE_NOT_FOUND="E_FILE_NOT_FOUND"
E_JSON_INVALID="E_JSON_INVALID"
E_LOCK_FAILED="E_LOCK_FAILED"
E_GIT_ERROR="E_GIT_ERROR"

# Recovery suggestions
_recovery_hub_not_found() { echo '"Run: aether install"'; }
_recovery_repo_not_init() { echo '"Run /ant:init in this repo first"'; }
_recovery_file_not_found() { echo '"Check file path and permissions"'; }
_recovery_json_invalid() { echo '"Validate JSON syntax"'; }
_recovery_lock_failed() { echo '"Wait for other operations to complete"'; }
_recovery_git_error() { echo '"Check git status and resolve conflicts"'; }
_recovery_default() { echo 'null'; }

# Main error function
json_err() {
  local code="${1:-$E_UNKNOWN}"
  local message="${2:-An unknown error occurred}"
  local details="${3:-{}}"
  local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

  # Get recovery suggestion based on code
  local recovery
  case "$code" in
    "$E_HUB_NOT_FOUND") recovery=$(_recovery_hub_not_found) ;;
    "$E_REPO_NOT_INITIALIZED") recovery=$(_recovery_repo_not_init) ;;
    "$E_FILE_NOT_FOUND") recovery=$(_recovery_file_not_found) ;;
    "$E_JSON_INVALID") recovery=$(_recovery_json_invalid) ;;
    "$E_LOCK_FAILED") recovery=$(_recovery_lock_failed) ;;
    "$E_GIT_ERROR") recovery=$(_recovery_git_error) ;;
    *) recovery=$(_recovery_default) ;;
  esac

  # Escape message for JSON
  local escaped_message=$(echo "$message" | sed 's/"/\\"/g' | tr '\n' ' ')

  # Output structured error
  printf '{"ok":false,"error":{"code":"%s","message":"%s","details":%s,"recovery":%s,"timestamp":"%s"}}\n' \
    "$code" "$escaped_message" "$details" "$recovery" "$timestamp" >&2

  # Log to activity.log (best effort)
  if [[ -n "${DATA_DIR:-}" && -d "$DATA_DIR" ]]; then
    echo "[$timestamp] ❌ ERROR $code: $escaped_message" >> "$DATA_DIR/activity.log" 2>/dev/null || true
  fi

  exit 1
}

# Warning function (non-fatal)
json_warn() {
  local code="${1:-W_UNKNOWN}"
  local message="${2:-Warning}"
  local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

  local escaped_message=$(echo "$message" | sed 's/"/\\"/g')

  printf '{"ok":true,"warning":{"code":"%s","message":"%s","timestamp":"%s"}}\n' \
    "$code" "$escaped_message" "$timestamp"

  # Log warning
  if [[ -n "${DATA_DIR:-}" && -d "$DATA_DIR" ]]; then
    echo "[$timestamp] ⚠️ WARN $code: $escaped_message" >> "$DATA_DIR/activity.log" 2>/dev/null || true
  fi
}

export -f json_err json_warn
export E_UNKNOWN E_HUB_NOT_FOUND E_REPO_NOT_INITIALIZED E_FILE_NOT_FOUND E_JSON_INVALID E_LOCK_FAILED E_GIT_ERROR
```

### Integration in cli.js
```javascript
// bin/cli.js - error handling integration

const { AetherError, HubError, RepoError, GitError } = require('./lib/errors');

// Global error handlers
process.on('uncaughtException', (error) => {
  console.error(JSON.stringify({
    error: {
      code: 'E_UNCAUGHT_EXCEPTION',
      message: error.message,
      details: { stack: error.stack },
      recovery: 'Please report this issue',
      timestamp: new Date().toISOString()
    }
  }, null, 2));
  process.exit(1);
});

process.on('unhandledRejection', (reason, promise) => {
  console.error(JSON.stringify({
    error: {
      code: 'E_UNHANDLED_REJECTION',
      message: String(reason),
      details: {},
      recovery: 'Please report this issue',
      timestamp: new Date().toISOString()
    }
  }, null, 2));
  process.exit(1);
});

// Command wrapper with error handling
function wrapCommand(commandFn) {
  return async (...args) => {
    try {
      return await commandFn(...args);
    } catch (error) {
      if (error instanceof AetherError) {
        console.error(JSON.stringify(error.toJSON(), null, 2));

        // Log to activity.log
        const logLine = `[${new Date().toISOString().split('T')[1].slice(0, 8)}] ❌ ERROR ${error.code}: ${error.message}\n`;
        try {
          fs.appendFileSync(path.join(DATA_DIR, 'activity.log'), logLine);
        } catch {}

        process.exit(1);
      }
      throw error; // Re-throw for global handler
    }
  };
}

// Usage in switch statement
switch (command) {
  case 'update': {
    await wrapCommand(async () => {
      if (!fs.existsSync(HUB_VERSION)) {
        throw new HubError('No distribution hub found', { path: HUB_DIR });
      }
      // ... rest of update logic
    })();
    break;
  }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| console.error with plain text | Structured JSON errors | Phase 3 | Machine-parseable errors, consistent format |
| set -e with generic exit | trap ERR with context | Phase 3 | Better debugging, structured output |
| Immediate exit on any error | Graceful degradation | Phase 3 | Better UX, continued operation |
| Ad-hoc error messages | Error code hierarchy | Phase 3 | Programmatic error handling |
| Silent failures | Activity.log integration | Phase 3 | Audit trail, debugging support |

**Deprecated/outdated:**
- String-based error matching: Use error codes instead
- process.exit() scattered in code: Use centralized handler
- Generic exit code 1: Use specific exit codes per error type

## Open Questions

1. **Exit Code Strategy**
   - What we know: Different error types should have different exit codes
   - What's unclear: Whether to use custom exit codes (64-113) or simple 0/1
   - Recommendation: Use sysexits.h conventions (64-78) for common errors, 1 for generic

2. **Activity.log Rotation**
   - What we know: activity.log grows indefinitely currently
   - What's unclear: Whether rotation is needed or if per-phase archives suffice
   - Recommendation: Keep current per-phase archive pattern, add size-based rotation if needed

3. **Error Recovery Automation**
   - What we know: Recovery suggestions are manual currently
   - What's unclear: Whether to implement automatic recovery for certain errors
   - Recommendation: Start with suggestions, add auto-recovery in future phase

4. **Bash ERR Trap Compatibility**
   - What we know: trap ERR can interfere with expected failures
   - What's unclear: Full scope of scripts that need adjustment
   - Recommendation: Add trap ERR to aether-utils.sh only, test all subcommands

## Sources

### Primary (HIGH confidence)
- Existing codebase: bin/cli.js (current error handling patterns)
- Existing codebase: .aether/aether-utils.sh (json_ok/json_err pattern, lines 36-41)
- Existing codebase: .aether/data/activity.log (established log format)
- Node.js documentation: process events, Error class
- Bash manual: trap command, special parameters ($?, $LINENO, $BASH_COMMAND)

### Secondary (MEDIUM confidence)
- POSIX exit codes (sysexits.h conventions)
- JSON-RPC 2.0 error object specification (inspiration for structure)
- CLI best practices from npm, yarn, gh CLI patterns

### Tertiary (LOW confidence)
- Web search: "Node.js CLI error handling patterns 2025" (search unavailable)
- Web search: "bash error handling trap ERR JSON output" (search unavailable)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Native Node.js and Bash features only
- Architecture: HIGH - Based on existing codebase patterns
- Pitfalls: MEDIUM - Based on common CLI development experience
- Code examples: HIGH - Derived from existing patterns and documentation

**Research date:** 2026-02-13
**Valid until:** 90 days (error handling patterns are stable)
