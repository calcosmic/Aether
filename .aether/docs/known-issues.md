# Known Issues and Workarounds

Documented issues from Oracle research findings. These are known limitations and bugs in the Aether system.

---

## Medium Priority Issues

### BUG-004: Missing error code in flag-acknowledge
**Location:** `flag-acknowledge` subcommand in `.aether/utils/flag.sh`
**Severity:** MEDIUM
**Status:** [FIXED in v2.1 -- Phase 10 error triage + Phase 13 modularization]
`flag-acknowledge` now uses `$E_VALIDATION_FAILED`, `$E_FILE_NOT_FOUND`, `$E_LOCK_FAILED`, and `$E_JSON_INVALID` appropriately.

### BUG-006: No lock release on JSON validation failure
**Location:** `atomic_write` in `.aether/utils/atomic-write.sh`
**Severity:** MEDIUM
**Symptom:** If JSON validation fails in `atomic_write`, temp file is cleaned but any lock held by the caller is not released
**Impact:** Lock remains held if caller had acquired it before calling `atomic_write`
**Fix:** Document lock ownership contract clearly -- callers must use trap-based cleanup
**Status:** Open -- `atomic_write` itself does not manage locks; callers are responsible for lock release via EXIT traps

### BUG-007: 17+ instances of missing error codes
**Location:** Various subcommands across `.aether/aether-utils.sh` and domain modules
**Severity:** MEDIUM
**Status:** [Mostly FIXED in v2.1 -- Phase 10 error triage]
Phase 10 replaced ~110 lazy error suppressions with proper fallbacks and added `$E_*` constants to ~48 dangerous paths. A small number of uncommented `2>/dev/null` idioms remain (SUPPRESS:OK annotated).

### BUG-008: Missing error code in flag-add jq failure
**Location:** `flag-add` subcommand in `.aether/utils/flag.sh`
**Severity:** HIGH
**Status:** [FIXED in v2.1 -- Phase 10 error triage + Phase 13 modularization]
`flag-add` now uses `$E_VALIDATION_FAILED`, `$E_JSON_INVALID`, and proper error handling throughout.

### BUG-009: Missing error codes in file checks
**Location:** `flag-acknowledge` and related subcommands in `.aether/utils/flag.sh`
**Severity:** MEDIUM
**Status:** [FIXED in v2.1 -- Phase 10 error triage + Phase 13 modularization]
File-not-found errors now use `json_err "$E_FILE_NOT_FOUND" "..."` consistently.

### BUG-010: Missing error codes in context-update
**Location:** `context-update` subcommand in `.aether/aether-utils.sh`
**Severity:** MEDIUM
**Status:** [FIXED in v2.1 -- Phase 10 error triage]
All error paths now use `$E_FILE_NOT_FOUND`, `$E_LOCK_FAILED`, and `$E_VALIDATION_FAILED` consistently.

### BUG-012: Missing error code in unknown command
**Location:** Default case (`*`) in `.aether/aether-utils.sh` dispatch
**Severity:** LOW
**Status:** [FIXED in v2.1 -- Phase 10 error triage]
Unknown command handler now uses `json_err "$E_VALIDATION_FAILED" "Unknown command: $cmd"`.

---

## Architecture Issues

### ISSUE-001: Inconsistent error code usage
**Location:** Multiple locations
**Severity:** MEDIUM
**Description:** Some `json_err` calls use hardcoded strings instead of constants
**Status:** [Mostly FIXED in v2.1 -- Phase 10 error triage]
The majority of error paths now use `$E_*` constants. Remaining intentional suppressions are annotated with SUPPRESS:OK comments.

### ISSUE-005: Potential infinite loop in spawn-tree
**Location:** `spawn-tree-depth` in `.aether/utils/spawn.sh`
**Severity:** LOW
**Description:** Edge case with circular parent chain could cause issues
**Mitigation:** Safety limit of 5 exists
**Status:** Open -- low risk due to safety limit; code extracted to `spawn.sh` during Phase 13 modularization

### ISSUE-006: Fallback json_err incompatible
**Location:** `.aether/aether-utils.sh` (lines ~65-72, fallback error handler)
**Severity:** LOW
**Description:** Fallback json_err doesn't accept error code parameter
**Impact:** If error-handler.sh fails to load, error codes are lost
**Status:** Open -- low risk since error-handler.sh is a stable infrastructure module

---

## Architecture Gaps

### GAP-007: No error code standards documentation
**Description:** Error codes exist but aren't documented for external consumers
**Impact:** Developers don't know which codes to use
**Status:** Partially addressed -- `_aether_log_error` infrastructure added in Phase 10, but no standalone error code reference doc exists yet

### GAP-008: Missing error path test coverage
**Description:** Error handling paths not fully tested
**Impact:** Bugs in error handling go undetected
**Status:** Partially addressed -- Phase 12 added state-api tests; 580+ tests now pass, but error-specific path coverage remains incomplete

---

*Generated from Oracle Research findings -- Updated 2026-03-24 during v2.1 documentation accuracy phase*
