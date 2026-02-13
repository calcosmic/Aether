---
phase: "03"
plan: "02"
subsystem: "error-handling"
tags: ["bash", "error-handling", "json", "trap", "graceful-degradation"]
dependencies:
  requires: ["02-03"]
  provides: ["structured-bash-errors", "trap-err-handler", "feature-flags"]
  affects: ["03-03", "03-04"]
tech-stack:
  added: []
  patterns: ["structured-json-errors", "trap-err-handler", "graceful-degradation"]
key-files:
  created:
    - .aether/utils/error-handler.sh
  modified:
    - .aether/aether-utils.sh
decisions:
  - id: "D031"
    text: "Use error code constants matching Node.js format (E_HUB_NOT_FOUND, etc.)"
  - id: "D032"
    text: "Implement trap ERR handler for unexpected bash failures with context"
  - id: "D033"
    text: "Use simple variables for feature flags (bash 3.2+ compatibility)"
  - id: "D034"
    text: "Log errors to activity.log with ERROR prefix for traceability"
metrics:
  duration: "215s"
  completed: "2026-02-13"
---

# Phase 3 Plan 2: Bash Error Handler Enhancement Summary

## One-Liner
Enhanced bash utilities with structured JSON error handling, trap ERR for unexpected failures, and graceful degradation for optional features.

## What Was Built

### 1. Shared Error Handler Module (`.aether/utils/error-handler.sh`)

Created a comprehensive bash error handling module with:

- **Error Code Constants**: E_UNKNOWN, E_HUB_NOT_FOUND, E_REPO_NOT_INITIALIZED, E_FILE_NOT_FOUND, E_JSON_INVALID, E_LOCK_FAILED, E_GIT_ERROR, E_VALIDATION_FAILED, E_FEATURE_UNAVAILABLE, E_BASH_ERROR

- **json_err() Function**: Enhanced error output with code, message, details, recovery suggestion, and ISO timestamp. Outputs structured JSON to stderr and logs to activity.log.

- **json_warn() Function**: Non-fatal warnings that output JSON to stdout (not stderr) and log to activity.log without exiting.

- **error_handler() Function**: For use with `trap ERR`. Captures line number, command, and exit code when unexpected failures occur.

- **Feature Flag Functions**: feature_enable(), feature_disable(), feature_enabled(), feature_log_degradation() for graceful degradation of optional features. Uses colon-separated string for bash 3.2+ compatibility.

- **Recovery Suggestions**: Internal functions that provide actionable recovery hints based on error codes.

### 2. Integration with aether-utils.sh

Updated the main utility script to:

- Source the error-handler.sh module
- Set up trap ERR for structured error output on unexpected failures
- Update existing json_err calls to use enhanced format with error codes
- Maintain backward compatibility with fallback json_err if source fails

### 3. Graceful Degradation Implementation

Added feature detection at initialization:

- **activity_log**: Disabled if DATA_DIR not writable
- **git_integration**: Disabled if git not installed
- **json_processing**: Disabled if jq not installed
- **file_locking**: Disabled if lock utilities not available

Updated subcommands to handle degradation:

- activity-log, activity-log-init, activity-log-read: Return warning instead of error when logging disabled
- flag-add, flag-resolve, flag-acknowledge, flag-auto-resolve: Proceed without locks when locking disabled (with warning)

## Files Changed

| File | Change | Lines |
|------|--------|-------|
| `.aether/utils/error-handler.sh` | Created | 200 |
| `.aether/aether-utils.sh` | Modified | +96/-13 |

## Commits

| Hash | Message |
|------|---------|
| 272396b | feat(03-02): create shared bash error handler module |
| 13ae5ff | feat(03-02): integrate error handler into aether-utils.sh |
| 645c455 | feat(03-02): add graceful degradation to aether-utils.sh |

## Verification Results

- [x] `.aether/utils/error-handler.sh` exists with all error functions
- [x] `bash -n .aether/utils/error-handler.sh` passes syntax check
- [x] `bash .aether/aether-utils.sh help` returns valid JSON
- [x] `json_err "E_TEST" "test message"` outputs structured JSON with code, message, recovery, timestamp
- [x] trap ERR is set up in aether-utils.sh
- [x] Feature flag functions work (feature_enable, feature_disable, feature_enabled)
- [x] Activity.log shows test error entries with correct format
- [x] All ERROR-02 and ERROR-03 requirements satisfied

## Error Format Example

```json
{
  "ok": false,
  "error": {
    "code": "E_FILE_NOT_FOUND",
    "message": "COLONY_STATE.json not found",
    "details": {"file": "COLONY_STATE.json"},
    "recovery": "Check file path and permissions",
    "timestamp": "2026-02-13T22:23:52Z"
  }
}
```

## Warning Format Example

```json
{
  "ok": true,
  "warning": {
    "code": "W_DEGRADED",
    "message": "Activity logging disabled: DATA_DIR not writable",
    "timestamp": "2026-02-13T22:23:52Z"
  }
}
```

## Deviations from Plan

None - plan executed exactly as written.

## Decisions Made

1. **Error Code Consistency**: Used same error codes as Node.js CLI (E_HUB_NOT_FOUND, etc.) for consistency across the codebase.

2. **Bash 3.2+ Compatibility**: Used colon-separated string for feature flags instead of associative arrays to support older bash versions.

3. **Trap ERR Integration**: Set up trap to call error_handler only if it's defined (from sourced file), providing graceful fallback.

4. **Activity Log Best Effort**: Error logging to activity.log uses `2>/dev/null || true` to prevent failures if logging itself fails.

5. **Backward Compatibility**: Maintained fallback json_err function in aether-utils.sh in case error-handler.sh fails to source.

## Next Phase Readiness

- ERROR-02: Structured JSON errors from bash - COMPLETE
- ERROR-03: Graceful degradation for optional features - COMPLETE
- Ready for Phase 3 Plan 3: Recovery Strategies
