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
- `bin/**/*`

**User Data (Never Touch):**
- `.aether/data/`, `.aether/dreams/`, `.aether/oracle/`
- `TO-DOs.md`, `COLONY_STATE.json`, `.env`, `*.log`

---

## Critical Issues (Fix Immediately)

### BUG-005: Missing lock release in flag-auto-resolve — FIXED (Phase 16)
**Location:** `.aether/aether-utils.sh:1022`
**Severity:** HIGH
**Status:** FIXED — Fixed in Phase 16: unified trap pattern (`trap 'release_lock 2>/dev/null || true' EXIT`) applied across all flag commands ensures lock release on all exit paths including jq failure.
**Symptom:** If jq command fails during flag resolution, lock is never released
**Impact:** Deadlock on flags.json if jq fails (malformed JSON, disk full, etc.)
**Workaround:** ~~Restart the colony session if commands hang on flag operations~~ — no longer needed
**Regression test:** `tests/bash/test-lock-lifecycle.sh` — test_flag_auto_resolve_jq_failure_releases_lock

### BUG-011: Missing error handling in flag-auto-resolve jq — FIXED (Phase 16)
**Location:** `.aether/aether-utils.sh:1022`
**Severity:** HIGH
**Status:** FIXED — Fixed in Phase 16: unified trap pattern across all flag commands. See BUG-005.
**Symptom:** jq failure during auto-resolve not handled
**Impact:** Combined with BUG-005, causes deadlock
**Fix:** ~~Add `|| { release_lock; json_err ... }` pattern~~ — implemented via EXIT trap

---

## Medium Priority Issues

### BUG-002: Missing release_lock in flag-add error path — FIXED (Phase 16)
**Location:** `.aether/aether-utils.sh:814`
**Severity:** MEDIUM
**Status:** FIXED — Fixed in Phase 16: trap-based EXIT cleanup (`trap 'release_lock 2>/dev/null || true' EXIT`) ensures lock release on all exit paths including jq failure. Trap is cleared on the success path.
**Symptom:** If acquire_lock succeeds but jq fails, lock is never released
**Impact:** Potential deadlock on file operations
**Regression test:** `tests/bash/test-lock-lifecycle.sh` — test_flag_add_jq_failure_releases_lock

### BUG-003: Race condition in backup creation — FIXED (Phase 16)
**Location:** `.aether/utils/atomic-write.sh:75`
**Severity:** MEDIUM
**Status:** FIXED — Backup is now created BEFORE JSON validation in both `atomic_write` and `atomic_write_from_file`. Verified in Phase 16 with regression tests confirming backup contains pre-write content.
**Symptom:** Backup created AFTER temp file validation but BEFORE atomic move
**Impact:** If process crashes between validation and backup, inconsistent state
**Regression test:** `tests/bash/test-lock-lifecycle.sh` — test_atomic_write_backup_before_validate, test_atomic_write_from_file_backup_before_validate

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

### ISSUE-002: Missing exec error handling — FIXED (Phase 18-02)
**Location:** `.aether/aether-utils.sh:2132-2144`
**Severity:** LOW
**Description:** `model-get` and `model-list` use `exec` without fallback
**Impact:** If exec fails, script continues to unknown command handler
**Status:** FIXED — Phase 18-02: subprocess error handling added to model-get and model-list with structured E_* error codes on failure.

### ISSUE-003: Incomplete help command — FIXED (Phase 18-03)
**Location:** `.aether/aether-utils.sh:106-111`
**Severity:** LOW
**Description:** Help command missing newer commands like queen-*, view-state-*, swarm-timing-*
**Impact:** Users cannot discover all available commands
**Status:** FIXED — Phase 18-03: help command sections key added with all command groups including Queen Commands, Model Routing, Swarm Operations, and all newer commands.

### ISSUE-004: Template path hardcoded to staging directory — FIXED (Phase 20)
**Location:** `.aether/aether-utils.sh:2689`
**Severity:** MEDIUM
**Status:** FIXED — Phase 20: stale staging template path removed from queen-init lookup array. Template is now found via hub path (`~/.aether/system/templates/`) or dev repo path (`.aether/templates/`) or legacy hub fallback.
**Description:** queen-init used a staging directory path that did not exist in npm installs
**Impact:** ~~queen-init will fail when Aether is installed as npm package~~
**~~Workaround:~~** ~~Use git clone instead of npm install~~ — no longer needed

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

### ISSUE-007: Feature detection race condition — FIXED (Phase 18-01)
**Location:** `.aether/aether-utils.sh:33-45`
**Severity:** LOW
**Description:** Feature detection runs before error handler fully sourced
**Status:** FIXED — Phase 18-01 (ARCH-09): feature detection block moved after fallback json_err definition (line 68 -> 81) so all fallback infrastructure available when feature detection runs.

---

## Architecture Gaps

### GAP-001: No schema version validation — FIXED (Phase 18-04)
**Description:** Commands assume state structure without validating version
**Impact:** Silent failures when state structure changes
**Status:** FIXED — Phase 18-04: `_migrate_colony_state` added to validate-state colony; auto-migrates pre-3.0 state files to v3.0 (additive only), notifies via W_MIGRATED warning; corrupt state files backed up before error.

### GAP-002: No cleanup for stale spawn-tree entries — FIXED (Phase 18-01)
**Description:** spawn-tree.txt grows indefinitely
**Impact:** File could grow very large over many sessions
**Status:** FIXED — Phase 18-01: `_rotate_spawn_tree` added to session-init; rotates spawn-tree.txt on each session start with timestamped archives; 5-archive cap; in-place truncation preserves tail -f file handles.

### GAP-003: No retry logic for failed spawns — RESOLVED (Phase 18-02)
**Description:** Task tool calls don't have retry logic
**Impact:** Transient failures cause build failures
**Status:** RESOLVED — User decision: fail-fast with rich error context (Phase 18-02). Retry logic adds complexity without clear benefit; subprocess errors now emit structured E_* codes with actionable Try: suggestions, allowing callers to decide on retry strategy.

### GAP-004: Missing queen-* documentation — FIXED (Phase 18-03)
**Description:** No docs for queen-init, queen-read, queen-promote
**Impact:** Users cannot discover wisdom feedback loop
**Status:** FIXED — Phase 18-03: queen-commands.md created in .aether/docs/; help command sections key added with Queen Commands section listing all three commands with descriptions.

### GAP-005: No validation of queen-read JSON output — FIXED (Phase 18-04)
**Description:** queen-read builds JSON but doesn't validate before returning
**Impact:** Could return malformed response
**Status:** FIXED — Phase 18-04: Two validation gates added to queen-read: Gate 1 validates METADATA JSON before --argjson use; Gate 2 validates assembled result before json_ok. Both emit E_JSON_INVALID with actionable Try: suggestion.

### GAP-006: Missing queen-* command documentation — FIXED (Phase 18-03)
**Description:** Duplicate of GAP-004 - no documentation exists
**Impact:** Commands are undiscoverable
**Status:** FIXED — Phase 18-03: See GAP-004.

### GAP-007: No error code standards documentation
**Description:** Error codes exist but aren't documented
**Impact:** Developers don't know which codes to use

### GAP-008: Missing error path test coverage
**Description:** Error handling paths not tested
**Impact:** Bugs in error handling go undetected

### GAP-009: context-update has no file locking — FIXED (Phase 16)
**Description:** Race condition possible during concurrent context updates
**Status:** FIXED — Fixed in Phase 16: context-update wraps all 11 action handlers in a single acquire_lock/release_lock pair. Lock is held for the duration of the update and released on both success and error paths via EXIT trap. force-unlock subcommand added for emergency recovery.
**Impact:** Potential data corruption
**Regression test:** `tests/bash/test-lock-lifecycle.sh` — test_context_update_acquires_lock, test_force_unlock_clears_locks

### GAP-010: Missing error code standards documentation
**Description:** Duplicate of GAP-007

---

## Workarounds Summary

| Issue | Workaround |
|-------|------------|
| ~~Lock-related deadlocks (BUG-005, BUG-002)~~ | ~~Restart colony session~~ — FIXED in Phase 16 |
| ~~Template path issue (ISSUE-004)~~ | ~~Use git clone instead of npm~~ — FIXED in Phase 20 |
| Missing command docs (GAP-004) | Read source code directly |

---

*Generated from Oracle Research findings - 2026-02-15*
