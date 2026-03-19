# Known Issues and Workarounds

Documented issues from Oracle research findings. These are known limitations and bugs in the Aether system.

---

## Medium Priority Issues

### BUG-004: Missing error code in flag-acknowledge
**Location:** `.aether/aether-utils.sh:930`
**Severity:** MEDIUM
**Symptom:** Uses hardcoded string instead of `$E_VALIDATION_FAILED`
**Impact:** Inconsistent error handling
**Fix:** Change to `json_err "$E_VALIDATION_FAILED" "Usage: ..."`

### BUG-006: No lock release on JSON validation failure
**Location:** `.aether/utils/atomic-write.sh:66`
**Severity:** MEDIUM
**Symptom:** If JSON validation fails, temp file cleaned but lock not released
**Impact:** Lock remains held if caller had acquired it
**Fix:** Document lock ownership contract clearly

### BUG-007: 17+ instances of missing error codes
**Location:** `.aether/aether-utils.sh` various lines
**Severity:** MEDIUM
**Symptom:** Commands use hardcoded strings instead of error constants
**Impact:** Inconsistent error handling, harder programmatic processing
**Fix:** Standardize all to use `json_err "$E_*" "message"` pattern

### BUG-008: Missing error code in flag-add jq failure
**Location:** `.aether/aether-utils.sh:856`
**Severity:** HIGH
**Symptom:** Lock released but error code missing on jq failure
**Impact:** Error response lacks proper error code
**Fix:** Change to `json_err "$E_JSON_INVALID" "Failed to add flag"`

### BUG-009: Missing error codes in file checks
**Location:** `.aether/aether-utils.sh:899, 933`
**Severity:** MEDIUM
**Symptom:** File not found errors use hardcoded strings
**Impact:** Inconsistent with other file not found errors
**Fix:** Use `json_err "$E_FILE_NOT_FOUND" "..."`

### BUG-010: Missing error codes in context-update
**Location:** `.aether/aether-utils.sh:1758+`
**Severity:** MEDIUM
**Symptom:** Various error paths lack error code constants
**Impact:** Inconsistent error handling

### BUG-012: Missing error code in unknown command
**Location:** `.aether/aether-utils.sh:2947`
**Severity:** LOW
**Symptom:** Unknown command handler uses bare string
**Impact:** Inconsistent error response

---

## Architecture Issues

### ISSUE-001: Inconsistent error code usage
**Location:** Multiple locations
**Severity:** MEDIUM
**Description:** Some `json_err` calls use hardcoded strings instead of constants
**Pattern:** Commands added early use strings; later commands use constants

### ISSUE-005: Potential infinite loop in spawn-tree
**Location:** `.aether/aether-utils.sh:402-448`, `spawn-tree.sh:222-263`
**Severity:** LOW
**Description:** Edge case with circular parent chain could cause issues
**Mitigation:** Safety limit of 5 exists

### ISSUE-006: Fallback json_err incompatible
**Location:** `.aether/aether-utils.sh:65-72`
**Severity:** LOW
**Description:** Fallback json_err doesn't accept error code parameter
**Impact:** If error-handler.sh fails to load, error codes are lost

---

## Architecture Gaps

### GAP-007: No error code standards documentation
**Description:** Error codes exist but aren't documented
**Impact:** Developers don't know which codes to use

### GAP-008: Missing error path test coverage
**Description:** Error handling paths not tested
**Impact:** Bugs in error handling go undetected

---

*Generated from Oracle Research findings - Updated 2026-03-19 during v1.3 milestone documentation phase*
