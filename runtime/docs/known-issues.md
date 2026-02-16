# Known Issues and Workarounds

Documented issues from Oracle research findings. These are known limitations and bugs in the Aether system.

---

## Fixed Issues

### Checkpoint Allowlist System (Fixed 2026-02-15)

**Issue:** Build checkpoint could stash user work (TO-DOs.md, dreams, Oracle specs)

**Root Cause:** `git stash` touched files outside system allowlist, stashing 1,145 lines of user work

**Fix:** Explicit allowlist system implemented
- Created `.aether/data/checkpoint-allowlist.json` defining safe system files
- Added `checkpoint-check` helper to `.aether/aether-utils.sh`
- Updated `build.md` (Claude and OpenCode) to use allowlist
- User data (`.aether/data/`, `.aether/dreams/`, `TO-DOs.md`) is never touched
- Warning displayed if user files are present during checkpoint

**System Files (Safe):**
- `.aether/aether-utils.sh`, `.aether/workers.md`, `.aether/docs/**/*.md`
- `.claude/commands/ant/**/*.md`, `.claude/commands/st/**/*.md`
- `.opencode/commands/ant/**/*.md`, `.opencode/agents/**/*.md`
- `runtime/**/*`, `bin/**/*`

**User Data (Never Touch):**
- `.aether/data/`, `.aether/dreams/`, `.aether/oracle/`
- `TO-DOs.md`, `COLONY_STATE.json`, `.env`, `*.log`

---

## Critical Issues (Fix Immediately)

### BUG-005: Missing lock release in flag-auto-resolve
**Location:** `.aether/aether-utils.sh:1022`
**Severity:** HIGH
**Symptom:** If jq command fails during flag resolution, lock is never released
**Impact:** Deadlock on flags.json if jq fails (malformed JSON, disk full, etc.)
**Workaround:** Restart the colony session if commands hang on flag operations
**Fix:** Add error handling with lock release before json_err

### BUG-011: Missing error handling in flag-auto-resolve jq
**Location:** `.aether/aether-utils.sh:1022`
**Severity:** HIGH
**Symptom:** jq failure during auto-resolve not handled
**Impact:** Combined with BUG-005, causes deadlock
**Fix:** Add `|| { release_lock; json_err ... }` pattern

---

## Medium Priority Issues

### BUG-002: Missing release_lock in flag-add error path
**Location:** `.aether/aether-utils.sh:814`
**Severity:** MEDIUM
**Symptom:** If acquire_lock succeeds but jq fails, lock is never released
**Impact:** Potential deadlock on file operations
**Fix:** Use trap-based cleanup or ensure release_lock in all exit paths

### BUG-003: Race condition in backup creation
**Location:** `.aether/utils/atomic-write.sh:75`
**Severity:** MEDIUM
**Symptom:** Backup created AFTER temp file validation but BEFORE atomic move
**Impact:** If process crashes between validation and backup, inconsistent state
**Fix:** Create backup BEFORE validation, or use transactional approach

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

### ISSUE-002: Missing exec error handling
**Location:** `.aether/aether-utils.sh:2132-2144`
**Severity:** LOW
**Description:** `model-get` and `model-list` use `exec` without fallback
**Impact:** If exec fails, script continues to unknown command handler

### ISSUE-003: Incomplete help command
**Location:** `.aether/aether-utils.sh:106-111`
**Severity:** LOW
**Description:** Help command missing newer commands like queen-*, view-state-*, swarm-timing-*
**Impact:** Users cannot discover all available commands

### ISSUE-004: Template path hardcoded to runtime/
**Location:** `.aether/aether-utils.sh:2689`
**Severity:** MEDIUM
**Description:** queen-init uses runtime/ directory which may not exist in npm installs
**Impact:** queen-init will fail when Aether is installed as npm package
**Workaround:** Use git clone instead of npm install

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

### ISSUE-007: Feature detection race condition
**Location:** `.aether/aether-utils.sh:33-45`
**Severity:** LOW
**Description:** Feature detection runs before error handler fully sourced

---

## Architecture Gaps

### GAP-001: No schema version validation
**Description:** Commands assume state structure without validating version
**Impact:** Silent failures when state structure changes

### GAP-002: No cleanup for stale spawn-tree entries
**Description:** spawn-tree.txt grows indefinitely
**Impact:** File could grow very large over many sessions

### GAP-003: No retry logic for failed spawns
**Description:** Task tool calls don't have retry logic
**Impact:** Transient failures cause build failures

### GAP-004: Missing queen-* documentation
**Description:** No docs for queen-init, queen-read, queen-promote
**Impact:** Users cannot discover wisdom feedback loop

### GAP-005: No validation of queen-read JSON output
**Description:** queen-read builds JSON but doesn't validate before returning
**Impact:** Could return malformed response

### GAP-006: Missing queen-* command documentation
**Description:** Duplicate of GAP-004 - no documentation exists
**Impact:** Commands are undiscoverable

### GAP-007: No error code standards documentation
**Description:** Error codes exist but aren't documented
**Impact:** Developers don't know which codes to use

### GAP-008: Missing error path test coverage
**Description:** Error handling paths not tested
**Impact:** Bugs in error handling go undetected

### GAP-009: context-update has no file locking
**Description:** Race condition possible during concurrent context updates
**Impact:** Potential data corruption

### GAP-010: Missing error code standards documentation
**Description:** Duplicate of GAP-007

---

## Workarounds Summary

| Issue | Workaround |
|-------|------------|
| Lock-related deadlocks (BUG-005, BUG-002) | Restart colony session |
| Template path issue (ISSUE-004) | Use git clone instead of npm |
| Missing command docs (GAP-004) | Read source code directly |

---

*Generated from Oracle Research findings - 2026-02-15*
