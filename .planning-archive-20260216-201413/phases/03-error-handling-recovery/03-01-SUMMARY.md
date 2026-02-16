---
phase: 03
plan: 01
subsystem: infrastructure
tags: [nodejs, error-handling, cli, logging, sysexits]

requires:
  - 02-testing-foundation

provides:
  - Centralized error handling infrastructure
  - Structured JSON error output
  - Activity.log integration
  - Graceful degradation framework

affects:
  - 03-02-bash-error-handling
  - 03-03-graceful-degradation

key-files:
  created:
    - bin/lib/errors.js
    - bin/lib/logger.js
  modified:
    - bin/cli.js

decisions:
  - Use native Node.js Error classes (no external dependencies)
  - Follow sysexits.h conventions for exit codes
  - Silent fail on logging errors to prevent cascades
  - Feature flags pattern for graceful degradation

metrics:
  duration: "1 hour"
  completed: "2026-02-13"
---

# Phase 3 Plan 1: Centralized Error Handling Summary

## One-Liner

Implemented centralized error handling infrastructure with AetherError class hierarchy, structured JSON output, activity.log integration, and sysexits.h-compliant exit codes for the Node.js CLI.

## What Was Built

### bin/lib/errors.js
A comprehensive error class hierarchy providing:

- **Base AetherError class** with code, message, details, recovery, and timestamp
- **ErrorCodes enum** with categorized error codes:
  - System errors (E_HUB_NOT_FOUND, E_REPO_NOT_INITIALIZED, E_FILE_SYSTEM, E_GIT_ERROR)
  - Validation errors (E_INVALID_STATE, E_MANIFEST_INVALID, E_JSON_PARSE)
  - Runtime errors (E_UPDATE_FAILED, E_LOCK_TIMEOUT, E_ATOMIC_WRITE_FAILED)
  - Unexpected errors (E_UNEXPECTED, E_UNCAUGHT_EXCEPTION, E_UNHANDLED_REJECTION)
  - Configuration errors (E_CONFIG)
- **Specific error subclasses** with auto-set codes and recovery suggestions:
  - HubError: Hub-related errors
  - RepoError: Repository initialization errors
  - GitError: Git operation errors
  - ValidationError: State validation errors
  - FileSystemError: File operation errors
  - ConfigurationError: Environment/configuration errors
- **getExitCode()** mapping to sysexits.h conventions (64-78 range)
- **wrapError()** utility for wrapping plain Errors

### bin/lib/logger.js
Structured logging module providing:

- **logError()**: Log AetherError or plain Error to activity.log
- **logActivity()**: Log caste-based activities with emoji
- **logWarning()**: Log warnings with code and message
- **logInfo()**: Log info messages
- **logSuccess()**: Log success messages
- **getRecentLogs()**: Retrieve recent log entries
- **Silent failure** on logging errors to prevent cascades
- **Sanitization** of log strings (newlines removed, 200 char limit)

### bin/cli.js Integration
Updated CLI with centralized error handling:

- **Global uncaughtException handler**: Catches unexpected errors, logs to activity.log, outputs structured JSON
- **Global unhandledRejection handler**: Catches promise rejections with context
- **FeatureFlags class**: Tracks degraded features for graceful degradation
- **wrapCommand() helper**: Wraps commands with consistent error handling
- **Updated error patterns**: Hub not found, repo not initialized, git dirty files now use structured errors

## Decisions Made

1. **Native Node.js only**: No external dependencies for error handling - using native Error classes and process events
2. **sysexits.h conventions**: Exit codes follow standard Unix conventions:
   - 65 (EX_DATAERR): Data format errors
   - 69 (EX_UNAVAILABLE): Service unavailable
   - 70 (EX_SOFTWARE): Internal software error
   - 73 (EX_CANTCREAT): Can't create file
   - 74 (EX_IOERR): I/O error
   - 78 (EX_CONFIG): Configuration error
3. **Silent logging failures**: All logging operations fail silently to prevent error cascades
4. **Feature flags pattern**: Graceful degradation tracked via FeatureFlags class with disable/isEnabled/getDegradedFeatures methods

## Deviations from Plan

None - plan executed exactly as written.

## Test Results

All verification checks passed:

- bin/lib/errors.js exists with all exports
- bin/lib/logger.js exists with all exports
- AetherError can be instantiated and serialized to JSON
- CLI loads without errors (help, version commands work)
- Activity.log shows test entries with correct format
- Exit codes follow sysexits.h conventions (all 10 tests passed)

## Files Changed

```
bin/lib/errors.js    | 239 +++++++++++++++++++++++++
bin/lib/logger.js    | 242 +++++++++++++++++++++++++
bin/cli.js          | 186 +++++++++++++++++++--
```

## API Usage Examples

### Creating and throwing errors:
```javascript
const { HubError, RepoError, GitError, getExitCode } = require('./lib/errors');
const { logError } = require('./lib/logger');

// Hub not found error
const error = new HubError(
  'No distribution hub found at ~/.aether/',
  { path: HUB_DIR }
);
logError(error);
console.error(JSON.stringify(error.toJSON(), null, 2));
process.exit(getExitCode(error.code)); // exits with 69
```

### Using wrapCommand:
```javascript
const wrappedUpdate = wrapCommand(async (repoPath) => {
  // Command logic here
  if (!fs.existsSync(repoPath)) {
    throw new RepoError('Repository not initialized');
  }
  // ...
});
```

### Feature flags for graceful degradation:
```javascript
const features = new FeatureFlags();

try {
  await riskyOperation();
} catch (err) {
  features.disable('optionalFeature', err.message);
  // Continue with degraded functionality
}

if (features.isEnabled('optionalFeature')) {
  // Use the feature
}
```

## Next Phase Readiness

This plan completes ERROR-01 requirement. Ready for:
- 03-02: Bash Error Handling (extend error handling to aether-utils.sh)
- 03-03: Graceful Degradation (expand feature flags usage)

## Rollback

If needed, revert the changes:
```bash
git revert f7ce282 8d52773 7fafce6
```

Or manually restore the original cli.js and remove bin/lib/ directory.
