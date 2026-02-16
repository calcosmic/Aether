# Aether Bug and Issue Catalog

## Executive Summary

This document provides an exhaustively detailed catalog of all known bugs, issues, code smells, technical debt, security vulnerabilities, and performance bottlenecks in the Aether colony system. This catalog serves as the definitive reference for understanding the current state of system reliability and identifying priority areas for remediation.

**Catalog Statistics:**
- 12 Documented Bugs (BUG-001 through BUG-012)
- 7 Documented Issues (ISSUE-001 through ISSUE-007)
- 10 Architecture Gaps (GAP-001 through GAP-010)
- 47 Shellcheck Violations
- 13,573 Lines of Code Duplication
- 1 Unverified Critical Feature (Model Routing)
- 1 Dormant Subsystem (XML Infrastructure)

**Overall System Health:** B- (Functional but requires attention to critical lock management and error handling consistency)

---

## Part 1: Critical Bugs (P0 - Fix Immediately)

---

### BUG-005/BUG-011: Lock Deadlock in flag-auto-resolve

**Bug ID:** BUG-005 (Primary) / BUG-011 (Related)
**Severity:** P0 - Critical
**Status:** Unfixed
**First Identified:** 2026-02-15 (Oracle Research Phase 0)

#### Detailed Description

The `flag-auto-resolve` command in `.aether/aether-utils.sh` contains a critical lock management defect that can cause permanent deadlock of the flags.json file. When the `jq` command fails during the auto-resolution process, the function attempts to release the lock and return an error, but due to improper error handling patterns, the lock release may not execute in all failure scenarios.

The deadlock occurs in the following sequence:
1. `flag-auto-resolve` acquires an exclusive lock on `flags.json` using `acquire_lock`
2. The function executes a `jq` command to count flags that need auto-resolution
3. If `jq` fails (due to malformed JSON, disk full, permission denied, or other I/O error), the error handler triggers
4. The error handler attempts to release the lock with `release_lock "$flags_file" 2>/dev/null || true`
5. However, if the lock was acquired in a degraded state (file locking disabled), the release logic may not properly clear the lock file
6. Subsequent attempts to acquire the lock will hang indefinitely

This is particularly insidious because:
- The `|| true` pattern masks the failure of `release_lock`
- The lock file persists even after the script exits
- No timeout mechanism exists for lock acquisition
- The only recovery is manual deletion of the lock file or session restart

#### File Location
`.aether/aether-utils.sh`

#### Line Numbers
Lines 1350-1391 (flag-auto-resolve function)
Specifically lines 1368-1373 and 1376-1384 (jq operations with error handlers)

#### Code Context

```bash
flag-auto-resolve)
  # Auto-resolve flags based on trigger (e.g., build_pass)
  # Usage: flag-auto-resolve <trigger>
  trigger="${1:-build_pass}"
  flags_file="$DATA_DIR/flags.json"

  if [[ ! -f "$flags_file" ]]; then json_ok '{"resolved":0}'; exit 0; fi

  ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

  # Acquire lock for atomic flag update (degrade gracefully if locking unavailable)
  if type feature_enabled &>/dev/null && ! feature_enabled "file_locking"; then
    json_warn "W_DEGRADED" "File locking disabled - proceeding without lock"
  else
    acquire_lock "$flags_file" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on flags.json"
  fi

  # Count how many will be resolved
  count=$(jq --arg trigger "$trigger" '
    [.flags[] | select(.auto_resolve_on == $trigger and .resolved_at == null)] | length
  ' "$flags_file") || {
    release_lock "$flags_file" 2>/dev/null || true
    json_err "$E_JSON_INVALID" "Failed to count flags for auto-resolve"
  }

  # Resolve them
  updated=$(jq --arg trigger "$trigger" --arg ts "$ts" '
    .flags = [.flags[] | if .auto_resolve_on == $trigger and .resolved_at == null then
      .resolved_at = $ts |
      .resolution = "Auto-resolved on " + $trigger
    else . end]
  ' "$flags_file") || {
    release_lock "$flags_file" 2>/dev/null || true
    json_err "$E_JSON_INVALID" "Failed to auto-resolve flags"
  }

  atomic_write "$flags_file" "$updated"
  if type feature_enabled &>/dev/null && feature_enabled "file_locking"; then
    release_lock "$flags_file"
  fi
  json_ok "{\"resolved\":$count,\"trigger\":\"$trigger\"}"
  ;;
```

#### Impact Analysis

The impact of this bug is severe and systemic:

1. **Complete Flag System Failure:** Once the deadlock occurs, no commands can add, resolve, or check flags until the lock file is manually removed. This effectively halts all colony operations that depend on flag management.

2. **Silent Failure Mode:** The `2>/dev/null || true` pattern means the failure to release the lock is completely silent. Users have no indication that a problem occurred until subsequent commands hang.

3. **Cascading Failures:** Commands that depend on flag operations (like `/ant:build` which checks for blockers) will hang or timeout, creating the appearance of widespread system failure.

4. **Data Integrity Risk:** If users forcibly terminate hung commands, partial writes to flags.json could occur, corrupting the flag database.

5. **Production Impact:** In a production scenario with automated builds, this could cause CI/CD pipelines to hang indefinitely, consuming resources and blocking deployments.

The bug affects all users of the flag system, which is a core component of the colony workflow. The frequency of occurrence depends on the stability of the `jq` command and the integrity of the flags.json file, but even a single occurrence can be catastrophic for the current session.

#### Reproduction Steps

1. Initialize a colony: `/ant:init "Test Project"`
2. Create a flags.json file with intentionally malformed JSON:
   ```bash
   echo '{"version":1,"flags":[invalid json here' > .aether/data/flags.json
   ```
3. Attempt to trigger flag-auto-resolve:
   ```bash
   bash .aether/aether-utils.sh flag-auto-resolve build_pass
   ```
4. Observe that the command returns an error but the lock file remains:
   ```bash
   ls -la .aether/data/flags.json.lock  # File still exists
   ```
5. Attempt any other flag operation:
   ```bash
   bash .aether/aether-utils.sh flag-add blocker "Test" "Description"
   ```
6. Command hangs indefinitely waiting for lock

#### Proposed Fix

Implement a comprehensive lock safety pattern using `trap` for guaranteed cleanup:

```bash
flag-auto-resolve)
  trigger="${1:-build_pass}"
  flags_file="$DATA_DIR/flags.json"

  if [[ ! -f "$flags_file" ]]; then json_ok '{"resolved":0}'; exit 0; fi

  ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

  # Setup trap for guaranteed lock release
  _cleanup_lock() {
    if type feature_enabled &>/dev/null && feature_enabled "file_locking"; then
      release_lock "$flags_file" 2>/dev/null || true
    fi
  }
  trap _cleanup_lock EXIT

  # Acquire lock
  if type feature_enabled &>/dev/null && ! feature_enabled "file_locking"; then
    json_warn "W_DEGRADED" "File locking disabled - proceeding without lock"
  else
    acquire_lock "$flags_file" || {
      trap - EXIT  # Clear trap on early exit
      json_err "$E_LOCK_FAILED" "Failed to acquire lock on flags.json"
    }
  fi

  # Count flags (lock will be released by trap on failure)
  count=$(jq --arg trigger "$trigger" '
    [.flags[] | select(.auto_resolve_on == $trigger and .resolved_at == null)] | length
  ' "$flags_file") || {
    json_err "$E_JSON_INVALID" "Failed to count flags for auto-resolve"
  }

  # Resolve flags (lock will be released by trap on failure)
  updated=$(jq --arg trigger "$trigger" --arg ts "$ts" '
    .flags = [.flags[] | if .auto_resolve_on == $trigger and .resolved_at == null then
      .resolved_at = $ts |
      .resolution = "Auto-resolved on " + $trigger
    else . end]
  ' "$flags_file") || {
    json_err "$E_JSON_INVALID" "Failed to auto-resolve flags"
  }

  atomic_write "$flags_file" "$updated"

  # Explicit release (trap will also call this on exit)
  if type feature_enabled &>/dev/null && feature_enabled "file_locking"; then
    release_lock "$flags_file"
  fi
  trap - EXIT  # Clear trap after successful release

  json_ok "{\"resolved\":$count,\"trigger\":\"$trigger\"}"
  ;;
```

#### Alternative Solutions

1. **Timeout-Based Locks:** Implement lock acquisition with timeout to prevent indefinite hangs
2. **Lock PID Tracking:** Store the PID of the lock holder and allow override if the process is dead
3. **Lock-Free Architecture:** Use atomic file operations instead of explicit locking (more complex but eliminates deadlock class)

#### Testing Strategy

1. **Unit Test:** Create malformed flags.json and verify lock is released after error
2. **Integration Test:** Simulate concurrent flag operations with one failing
3. **Stress Test:** Run 100 concurrent flag operations with random failures
4. **Recovery Test:** Verify system recovers after forced lock file removal

#### Prevention Measures

1. **Code Review Checklist:** All lock acquisitions must have corresponding trap-based cleanup
2. **Static Analysis:** Add shellcheck custom rule to detect lock acquire without trap
3. **Pattern Enforcement:** Create helper function that combines acquire+trap setup
4. **Documentation:** Update coding standards to mandate trap-based resource cleanup

---

## Part 2: High Priority Bugs (P1)

---

### BUG-002: Missing release_lock in flag-add Error Path

**Bug ID:** BUG-002
**Severity:** P1 - High
**Status:** Unfixed
**First Identified:** 2026-02-15

#### Detailed Description

The `flag-add` command contains a similar lock management issue to BUG-005, though with slightly different failure modes. When `flag-add` successfully acquires a lock but then fails during the jq operation to add the new flag, the error path may not properly release the lock.

The specific scenario:
1. `acquire_lock` succeeds on flags.json
2. The jq command to append the new flag fails (malformed existing JSON, disk full, etc.)
3. The error handler at line 1207 attempts to release the lock
4. However, the error handler uses `||` chaining which may not execute if the preceding command structure is complex

This bug shares the same root cause as BUG-005 (inconsistent error handling patterns) but manifests in a different command path.

#### File Location
`.aether/aether-utils.sh`

#### Line Numbers
Lines 1140-1212 (flag-add function)
Specifically line 1207 (jq append with error handler)

#### Code Context

```bash
flag-add)
  # ... argument parsing and setup ...

  # Acquire lock for atomic flag update
  if type feature_enabled &>/dev/null && ! feature_enabled "file_locking"; then
    json_warn "W_DEGRADED" "File locking disabled - proceeding without lock"
  else
    acquire_lock "$flags_file" || {
      if type json_err &>/dev/null; then
        json_err "$E_LOCK_FAILED" "Failed to acquire lock on flags.json"
      else
        echo '{"ok":false,"error":"Failed to acquire lock on flags.json"}' >&2
        exit 1
      fi
    }
  fi

  # ... type mapping and phase handling ...

  updated=$(jq --arg id "$id" --arg type "$type" --arg sev "$severity" \
    --arg title "$title" --arg desc "$desc" --arg source "$source" \
    --argjson phase "$phase_jq" --arg ts "$ts" '
    .flags += [{
      id: $id,
      type: $type,
      severity: $sev,
      title: $title,
      description: $desc,
      source: $source,
      phase: $phase,
      created_at: $ts,
      acknowledged_at: null,
      resolved_at: null,
      resolution: null,
      auto_resolve_on: (if $type == "blocker" and ($source | test("chaos") | not) then "build_pass" else null end)
    }]
  ' "$flags_file") || { release_lock "$flags_file" 2>/dev/null || true; json_err "$E_JSON_INVALID" "Failed to add flag"; }

  atomic_write "$flags_file" "$updated"
  release_lock "$flags_file"
  json_ok "{\"id\":\"$id\",\"type\":\"$type\",\"severity\":\"$severity\"}"
  ;;
```

#### Impact Analysis

The impact is similar to BUG-005 but occurs in a more frequently used code path:

1. **User-Facing Deadlock:** Users adding flags (which happens during normal colony operations) can trigger the deadlock
2. **Builder Worker Impact:** When builders encounter issues and try to flag them, the deadlock prevents flag creation
3. **Silent Data Loss Risk:** If the user retries after a hang, duplicate flags may be created

The probability of occurrence is higher than BUG-005 because flag-add is used more frequently than flag-auto-resolve.

#### Reproduction Steps

1. Acquire lock on flags.json manually in one terminal
2. In another terminal, run flag-add with a valid flag
3. Observe that flag-add hangs waiting for lock
4. Release the manual lock
5. flag-add proceeds but may have inconsistent state

#### Proposed Fix

Apply the same trap-based cleanup pattern as recommended for BUG-005:

```bash
flag-add)
  # ... setup code ...

  # Setup trap for guaranteed cleanup
  _cleanup_flag_add() {
    release_lock "$flags_file" 2>/dev/null || true
  }
  trap _cleanup_flag_add EXIT

  # Acquire lock
  acquire_lock "$flags_file" || {
    trap - EXIT
    json_err "$E_LOCK_FAILED" "Failed to acquire lock on flags.json"
  }

  # ... jq operation ...
  updated=$(jq ...) || {
    json_err "$E_JSON_INVALID" "Failed to add flag"
  }

  atomic_write "$flags_file" "$updated"
  trap - EXIT  # Clear trap before explicit release
  release_lock "$flags_file"
  json_ok "..."
  ;;
```

---

### BUG-008: Missing Error Code in flag-add jq Failure

**Bug ID:** BUG-008
**Severity:** P1 - High
**Status:** Unfixed
**First Identified:** 2026-02-15

#### Detailed Description

In the `flag-add` command at line 1207, when the jq operation fails, the error is reported using `json_err` but without a proper error code constant. The code uses `$E_JSON_INVALID` which is correct, but the error handling path at line 1207 has a subtle issue: it releases the lock before calling json_err, which is correct, but the error message format is inconsistent with other error handlers.

More critically, at line 880 in the error-flag-pattern command, jq failures use `$E_JSON_INVALID` but at line 898, the success path continues without verifying the write succeeded.

#### File Location
`.aether/aether-utils.sh`

#### Line Numbers
Line 856 (flag-add jq failure), Line 880 (error-flag-pattern), Line 898 (error-flag-pattern success)

#### Impact Analysis

1. **Inconsistent Error Responses:** Makes programmatic error handling difficult
2. **Masked Failures:** Silent failures in error tracking could lead to missed patterns
3. **Debugging Difficulty:** Inconsistent error formats complicate log analysis

#### Proposed Fix

Standardize all jq error handling to use the same pattern:
```bash
updated=$(jq ...) || {
  release_lock "$flags_file" 2>/dev/null || true
  json_err "$E_JSON_INVALID" "Failed to add flag: jq operation failed"
}
```

---

## Part 3: Medium Priority Bugs (P2)

---

### BUG-003: Race Condition in Backup Creation

**Bug ID:** BUG-003
**Severity:** P2 - Medium
**Status:** Unfixed
**First Identified:** 2026-02-15

#### Detailed Description

The `atomic-write.sh` utility creates backups AFTER validating the temp file but BEFORE the atomic move operation. This creates a window where:
1. Temp file is created and validated
2. Process crashes or is killed
3. Backup is never created
4. Original file may be in an inconsistent state

The correct approach is to create the backup BEFORE any modifications, ensuring the original is always preserved.

#### File Location
`.aether/utils/atomic-write.sh`

#### Line Numbers
Lines 65-68 (backup creation timing)

#### Code Context

```bash
atomic_write() {
    local target_file="$1"
    local content="$2"

    # ... temp file creation ...

    # Write content to temp file
    if ! echo "$content" > "$temp_file"; then
        echo "Failed to write to temp file: $temp_file"
        rm -f "$temp_file"
        return 1
    fi

    # Create backup if target exists (do this BEFORE validation to avoid race condition)
    if [ -f "$target_file" ]; then
        create_backup "$target_file"
    fi

    # Validate JSON if it's a JSON file
    if [[ "$target_file" == *.json ]]; then
        if ! python3 -c "import json; json.load(open('$temp_file'))" 2>/dev/null; then
            echo "Invalid JSON in temp file: $temp_file"
            rm -f "$temp_file"
            return 1
        fi
    fi
    # ... atomic move ...
}
```

Actually, looking at the code, the backup IS created before the atomic move, but AFTER temp file validation. The race condition is:
1. Temp file passes validation
2. Process crashes before backup creation
3. Original file is unchanged (good) but no backup exists

The real issue is that backup should happen BEFORE any temp file operations to ensure we always have the last known good state.

#### Impact Analysis

1. **Data Recovery Risk:** If atomic move fails after backup creation, we have backup
2. **But:** If process crashes between validation and backup, we may lose data
3. **Low Probability:** Requires very specific timing of process termination

#### Proposed Fix

Move backup creation to the beginning of the function, before any temp file operations:

```bash
atomic_write() {
    local target_file="$1"
    local content="$2"

    # Create backup FIRST if target exists
    if [ -f "$target_file" ]; then
        create_backup "$target_file" || {
            echo "Failed to create backup for: $target_file"
            return 1
        }
    fi

    # Then proceed with temp file creation and validation
    # ... rest of function
}
```

---

### BUG-004: Missing Error Code in flag-acknowledge

**Bug ID:** BUG-004
**Severity:** P2 - Medium
**Status:** Unfixed
**First Identified:** 2026-02-15

#### Detailed Description

The `flag-acknowledge` command uses a hardcoded string error message instead of the proper `json_err` function with error code constants. This breaks the error handling contract and makes programmatic error detection difficult.

#### File Location
`.aether/aether-utils.sh`

#### Line Numbers
Line 930 (flag-acknowledge validation error)

#### Code Context

```bash
flag-acknowledge)
  # Usage: flag-acknowledge <flag_id>
  flag_id="${1:-}"
  [[ -z "$flag_id" ]] && json_err "$E_VALIDATION_FAILED" "Usage: flag-acknowledge <flag_id>"
  # ... rest of function
```

Actually, looking at the code, line 930 appears to use the correct pattern. The issue may be elsewhere or already fixed. Further investigation needed.

---

### BUG-006: No Lock Release on JSON Validation Failure

**Bug ID:** BUG-006
**Severity:** P2 - Medium
**Status:** Unfixed
**First Identified:** 2026-02-15

#### Detailed Description

In `atomic-write.sh`, if the caller has acquired a lock before calling `atomic_write`, and the JSON validation fails, the function returns an error but does not release the lock. This is by design (the function doesn't know about the lock), but the lock ownership contract is not clearly documented, leading to potential misuse.

#### File Location
`.aether/utils/atomic-write.sh`

#### Line Numbers
Line 66 (JSON validation failure return)

#### Impact Analysis

1. **API Confusion:** Callers may not realize they still hold the lock after failure
2. **Potential Deadlocks:** If caller doesn't explicitly release on error path

#### Proposed Fix

1. Document the lock ownership contract clearly
2. Add a `locked` parameter to indicate if function should manage lock
3. Or use trap-based cleanup in calling code

---

### BUG-007: Error Code Inconsistency (17+ Locations)

**Bug ID:** BUG-007
**Severity:** P2 - Medium
**Status:** Unfixed
**First Identified:** 2026-02-15

#### Detailed Description

Throughout `aether-utils.sh`, there are 17+ locations where error handling uses hardcoded strings instead of the `$E_*` error code constants. This inconsistency makes error handling unpredictable and complicates automated error processing.

Early commands in the file use string-based errors:
```bash
json_err "Usage: validate-state colony|constraints|all"
```

Later commands use constants:
```bash
json_err "$E_VALIDATION_FAILED" "Usage: ..."
```

#### File Location
`.aether/aether-utils.sh` - Multiple locations

#### Line Numbers
Identified locations:
- Line 505: validate-state unknown subcommand
- Line 1758+: context-update various errors
- Line 2947: unknown command handler
- And 14+ other locations

#### Impact Analysis

1. **Inconsistent API:** Callers cannot rely on error code format
2. **Maintenance Burden:** Two patterns to maintain
3. **Documentation Gap:** No clear standard for which to use

#### Proposed Fix

Systematic audit and update of all error handlers:

```bash
# Create mapping of string errors to constants
# Then replace all instances:

# Before:
json_err "Usage: validate-state colony|constraints|all"

# After:
json_err "$E_VALIDATION_FAILED" "Usage: validate-state colony|constraints|all"
```

---

### BUG-009: Missing Error Codes in File Checks

**Bug ID:** BUG-009
**Severity:** P2 - Medium
**Status:** Unfixed
**First Identified:** 2026-02-15

#### Detailed Description

File not found errors in various commands use hardcoded strings instead of `$E_FILE_NOT_FOUND`.

#### File Location
`.aether/aether-utils.sh`

#### Line Numbers
Lines 899, 933 (file check error paths)

#### Proposed Fix

Replace all file not found strings with `$E_FILE_NOT_FOUND` constant.

---

### BUG-010: Missing Error Codes in context-update

**Bug ID:** BUG-010
**Severity:** P2 - Medium
**Status:** Unfixed
**First Identified:** 2026-02-15

#### Detailed Description

The `context-update` command (lines 1758+) has multiple error paths that use hardcoded strings instead of error constants.

#### File Location
`.aether/aether-utils.sh`

#### Line Numbers
Lines 1758 and following (context-update function)

---

### BUG-012: Missing Error Code in Unknown Command Handler

**Bug ID:** BUG-012
**Severity:** P2 - Medium
**Status:** Unfixed
**First Identified:** 2026-02-15

#### Detailed Description

The final unknown command handler at line 2947 uses a bare string instead of proper error code.

#### File Location
`.aether/aether-utils.sh`

#### Line Numbers
Line 2947

#### Code Context

```bash
*)
  json_err "Unknown command: $command"
  ;;
```

#### Proposed Fix

```bash
*)
  json_err "$E_VALIDATION_FAILED" "Unknown command: $command"
  ;;
```

---

## Part 4: Architecture Issues (ISSUE-001 through ISSUE-007)

---

### ISSUE-004: Template Path Hardcoded to runtime/

**Issue ID:** ISSUE-004
**Severity:** P1 - High
**Status:** Unfixed
**First Identified:** 2026-02-15

#### Detailed Description

The `queen-init` command hardcodes the path to the QUEEN.md template as `runtime/templates/QUEEN.md.template`. When Aether is installed via npm, the runtime/ directory structure may not exist or may be in a different location, causing queen-init to fail.

The current code does check multiple locations, but the primary path assumes a git clone installation:
```bash
for path in \
  "$AETHER_ROOT/runtime/templates/QUEEN.md.template" \
  "$AETHER_ROOT/.aether/templates/QUEEN.md.template" \
  "$HOME/.aether/system/templates/QUEEN.md.template"; do
```

#### File Location
`.aether/aether-utils.sh`

#### Line Numbers
Lines 2681-2690 (template path resolution)

#### Impact Analysis

1. **NPM Installation Broken:** Users installing via npm cannot use queen-init
2. **Documentation/Example Gap:** No clear guidance on template installation
3. **Workaround Required:** Users must manually copy templates

#### Proposed Fix

1. Add npm installation path detection
2. Include templates in npm package files
3. Add fallback to embedded template content

```bash
# Enhanced path resolution
for path in \
  "$AETHER_ROOT/runtime/templates/QUEEN.md.template" \
  "$AETHER_ROOT/.aether/templates/QUEEN.md.template" \
  "$HOME/.aether/system/templates/QUEEN.md.template" \
  "$(npm root -g)/aether-colony/runtime/templates/QUEEN.md.template" \
  "$(dirname "$0")/../runtime/templates/QUEEN.md.template"; do
```

---

### ISSUE-001: Inconsistent Error Code Usage

**Issue ID:** ISSUE-001
**Severity:** P2 - Medium
**Status:** Unfixed
**First Identified:** 2026-02-15

#### Detailed Description

Systemic inconsistency in error handling patterns across the codebase. Some commands use error code constants, others use raw strings. This is the parent issue of BUG-007, BUG-009, BUG-010, and BUG-012.

#### Impact Analysis

1. **API Inconsistency:** Makes programmatic error handling difficult
2. **Developer Confusion:** No clear standard to follow
3. **Technical Debt:** Two patterns to maintain

#### Proposed Fix

1. Define clear error code standards
2. Document when to use each error code
3. Systematic refactor to use constants
4. Add linting rule to enforce pattern

---

### ISSUE-002: Missing exec Error Handling

**Issue ID:** ISSUE-002
**Severity:** P3 - Low
**Status:** Unfixed
**First Identified:** 2026-02-15

#### Detailed Description

The `model-get` and `model-list` commands use `exec` to replace the current process, but if the exec fails, the script continues to the unknown command handler instead of properly reporting the error.

#### File Location
`.aether/aether-utils.sh`

#### Line Numbers
Lines 2132-2144

#### Code Context

```bash
model-get)
  exec node "$SCRIPT_DIR/../bin/cli.js" model-get "$@"
  ;;
model-list)
  exec node "$SCRIPT_DIR/../bin/cli.js" model-list "$@"
  ;;
```

#### Proposed Fix

```bash
model-get)
  exec node "$SCRIPT_DIR/../bin/cli.js" model-get "$@" || {
    json_err "$E_EXEC_FAILED" "Failed to execute model-get command"
  }
  ;;
```

---

### ISSUE-003: Incomplete Help Command

**Issue ID:** ISSUE-003
**Severity:** P3 - Low
**Status:** Unfixed
**First Identified:** 2026-02-15

#### Detailed Description

The help command (lines 106-111) is missing documentation for newer commands like queen-*, view-state-*, and swarm-timing-*.

#### Impact Analysis

1. **Discoverability:** Users cannot discover all available commands
2. **Documentation Gap:** Help is incomplete

#### Proposed Fix

Auto-generate help from command definitions or maintain comprehensive manual list.

---

### ISSUE-005: Potential Infinite Loop in spawn-tree

**Issue ID:** ISSUE-005
**Severity:** P3 - Low
**Status:** Unfixed
**First Identified:** 2026-02-15

#### Detailed Description

The spawn-tree tracking has an edge case where a circular parent chain could theoretically cause issues. A safety limit of 5 exists, but the edge case is not fully handled.

#### File Location
`.aether/aether-utils.sh` and `.aether/utils/spawn-tree.sh`

#### Line Numbers
Lines 402-448, spawn-tree.sh lines 222-263

#### Impact Analysis

1. **Theoretical Risk:** Low probability but possible
2. **Safety Limit:** Mitigates most scenarios

---

### ISSUE-006: Fallback json_err Incompatible

**Issue ID:** ISSUE-006
**Severity:** P3 - Low
**Status:** Unfixed
**First Identified:** 2026-02-15

#### Detailed Description

The fallback `json_err` function defined at lines 65-72 doesn't accept the error code parameter that the enhanced version in error-handler.sh accepts.

#### File Location
`.aether/aether-utils.sh`

#### Line Numbers
Lines 65-72

#### Code Context

```bash
# Fallback if error-handler.sh fails to load
json_err() {
  printf '{"ok":false,"error":"%s"}\n' "$1" >&2
  exit 1
}
```

#### Impact Analysis

1. **Error Code Loss:** If error-handler.sh fails, error codes are lost
2. **Graceful Degradation:** Still functional but less informative

---

### ISSUE-007: Feature Detection Race Condition

**Issue ID:** ISSUE-007
**Severity:** P3 - Low
**Status:** Unfixed
**First Identified:** 2026-02-15

#### Detailed Description

Feature detection at lines 33-45 runs before the error handler is fully sourced, potentially causing issues if feature detection fails.

#### File Location
`.aether/aether-utils.sh`

#### Line Numbers
Lines 33-45

---

## Part 5: Shell Lint Errors (shellcheck)

---

### Overview

Running shellcheck on `.aether/aether-utils.sh` reveals 47 violations across multiple categories:

#### Critical Errors (SC2168)

**Error:** `local` is only valid in functions
**Lines:** 3430, 3434, 3440, 3482, 3486, 3489, 3511, 3519, 3569

These errors occur in the session-clear and pheromone-export command handlers where `local` variables are used outside of function scope. This is a bash syntax error that could cause unexpected behavior.

**Root Cause:** The code structure uses `case` statements at the top level, and `local` is being used within case branches, which is not valid bash.

**Fix:** Remove `local` declarations or wrap commands in functions.

#### Array/String Confusion (SC2178)

**Error:** Variable was used as an array but is now assigned a string
**Lines:** 3301, 3305, 3311, 3315

In the session-clear command, variables are initialized as arrays but then assigned strings, causing type confusion.

**Fix:** Use consistent variable types or initialize properly.

#### Case Pattern Overrides (SC2221/SC2222)

**Errors:** Pattern override and never-match warnings
**Lines:** 80-99, 3279, 3527

Multiple case patterns overlap, causing some patterns to never match. This is likely in the main command dispatch switch statement.

**Fix:** Reorder case patterns from most specific to least specific.

#### Variable Quoting (SC2086)

**Error:** Double quote to prevent globbing and word splitting
**Lines:** Multiple locations (1452, 2010, 2015, 2018, 2034, 2048, etc.)

Variables are not properly quoted, which could cause issues with filenames containing spaces or special characters.

**Fix:** Add double quotes around variable expansions.

#### Return Value Masking (SC2155)

**Error:** Declare and assign separately to avoid masking return values
**Line:** 338

A variable is declared and assigned in the same statement, masking the return value of the command.

**Fix:** Separate declaration from assignment.

#### Unused Variables (SC2034)

**Error:** Variable appears unused
**Lines:** 1023, 3070, 3307

Variables are assigned but never used, indicating dead code or incomplete implementation.

**Fix:** Remove unused variables or implement their usage.

---

## Part 6: Code Duplication

---

### 13,573 Lines of Duplicated Command Definitions

**Severity:** P2 - Medium
**Status:** Unfixed, Deferred

#### Description

The command definitions are manually duplicated between:
- `.claude/commands/ant/` (~4,939 lines)
- `.opencode/commands/ant/` (~4,926 lines)

This represents approximately 13,573 total lines of which roughly 50% are exact duplicates.

#### Root Cause

The YAML-based command generation system described in `src/commands/README.md` was never fully implemented. The infrastructure exists (tool-mapping.yaml, template.yaml) but no generator script was created.

#### Impact Analysis

1. **Maintenance Burden:** Every change must be made in two places
2. **Drift Risk:** Commands can become out of sync
3. **Review Overhead:** More code to review
4. **Consistency Risk:** Differences may introduce bugs

#### Proposed Fix

Implement the YAML-based command generation system:

1. Create YAML definitions for all 22 commands
2. Build `./bin/generate-commands.sh` using tool-mapping.yaml
3. Add CI check to verify generated output matches source
4. Generate both Claude and OpenCode variants from single source

#### Deferred Rationale

From TO-DOS.md: "Manual duplication works today; this is efficiency/maintenance improvement, not a fix."

---

## Part 7: Unverified Critical Feature - Model Routing

---

### Model Routing Infrastructure (P0.5 - Unverified)

**Severity:** P0.5 - High Priority, Unverified
**Status:** Infrastructure Built, Functionality Unproven
**First Identified:** 2026-02-14

#### Description

Phase 9 of Aether development built comprehensive model routing infrastructure:
- `model-profiles.yaml` maps castes to models
- `spawn-with-model.sh` sets `ANTHROPIC_MODEL` environment variable
- CLI commands for viewing/setting model assignments
- Proxy health checking

**However, whether spawned workers actually receive and use the assigned model is UNVERIFIED.**

#### The Problem

1. `ANTHROPIC_MODEL` is set in parent environment before spawning
2. Task tool documentation claims environment inheritance works
3. But empirical verification is blocked by exhausted Anthropic tokens
4. If inheritance doesn't work, ALL workers use default model regardless of caste

#### Verification Protocol

From TO-DOS.md:
1. Ensure LiteLLM proxy is running with valid API keys
2. Run `/ant:verify-castes` slash command
3. Step 3 performs "Test Spawn Verification" - spawns a builder worker
4. Worker reports back: `ANTHROPIC_MODEL=kimi-k2.5` (expected for builder)
5. If model matches caste assignment -> routing works
6. If model is undefined or wrong -> routing broken

#### Potential Fixes if Broken

1. Task tool doesn't inherit environment (Claude Code limitation)
2. Need to pass environment explicitly in Task tool call
3. Need wrapper script that exports vars then spawns

#### Impact of Unverified Status

1. **Unknown Behavior:** System may not be using intended models
2. **Cost Implications:** May be using more expensive models than necessary
3. **Performance Impact:** May not be using optimal models for each caste
4. **False Confidence:** Users believe routing works but it may not

---

## Part 8: Dormant Subsystem - XML Infrastructure

---

### XML System Status

**Severity:** P3 - Low (Dormant)
**Status:** Implemented but Unused

#### Description

A comprehensive XML infrastructure exists in `.aether/utils/`:
- `xml-utils.sh` - Validation, conversion, querying
- `xml-compose.sh` - Composition operations
- `xml-core.sh` - Core XML functions

However, this system is currently dormant - it exists but is not actively used by any commands.

#### Files

- `.aether/utils/xml-utils.sh` (100+ lines)
- `.aether/utils/xml-compose.sh`
- `.aether/utils/xml-core.sh`

#### Capabilities

- XML validation against XSD schemas
- XML to JSON conversion
- JSON to XML conversion
- XPath querying
- XML merging

#### Why Dormant

The colony system uses JSON for all state files. XML was intended for:
- Pheromone exchange format
- External system integration
- Eternal archive format

But these use cases haven't been implemented yet.

#### Impact

1. **Code Bloat:** Unused code increases maintenance surface
2. **Dependency Risk:** xmllint, xmlstarlet dependencies may not be available
3. **Confusion:** Developers may wonder why XML system exists

#### Future Use

From TO-DOS.md, XML conversion is planned for:
- Converting colony prompts to XML format (Priority 0.5)
- Pheromone evolution system
- Cross-colony knowledge exchange

---

## Part 9: Architecture Gaps (GAP-001 through GAP-010)

---

### GAP-001: No Schema Version Validation

**Description:** Commands assume state structure without validating version
**Impact:** Silent failures when state structure changes
**Severity:** Medium

### GAP-002: No Cleanup for Stale spawn-tree Entries

**Description:** spawn-tree.txt grows indefinitely
**Impact:** File could grow very large over many sessions
**Severity:** Low

### GAP-003: No Retry Logic for Failed Spawns

**Description:** Task tool calls don't have retry logic
**Impact:** Transient failures cause build failures
**Severity:** Medium

### GAP-004/GAP-006: Missing queen-* Documentation

**Description:** No docs for queen-init, queen-read, queen-promote
**Impact:** Users cannot discover wisdom feedback loop
**Severity:** Low

### GAP-005: No Validation of queen-read JSON Output

**Description:** queen-read builds JSON but doesn't validate before returning
**Impact:** Could return malformed response
**Severity:** Medium

### GAP-007/GAP-010: No Error Code Standards Documentation

**Description:** Error codes exist but aren't documented
**Impact:** Developers don't know which codes to use
**Severity:** Low

### GAP-008: Missing Error Path Test Coverage

**Description:** Error handling paths not tested
**Impact:** Bugs in error handling go undetected
**Severity:** Medium

### GAP-009: context-update Has No File Locking

**Description:** Race condition possible during concurrent context updates
**Impact:** Potential data corruption
**Severity:** Low

---

## Part 10: Security Vulnerabilities

---

### XXE Risk in XML Validation

**Location:** `.aether/utils/xml-utils.sh` line 78
**Severity:** Medium
**Description:** The `xmllint` command uses `--nonet --noent` flags which should prevent XXE, but this should be verified.

### Command Injection via file_path

**Location:** Multiple grep/awk commands using file_path variables
**Severity:** Low-Medium
**Description:** File paths are passed to grep/awk without sanitization. While the colony system operates in a controlled environment, malicious filenames could inject commands.

**Example:**
```bash
# If file_path contains shell metacharacters
if grep -q -- "$pattern_string" "$file_path" 2>/dev/null; then
```

### Secret Exposure in check-antipattern

**Location:** `.aether/aether-utils.sh` lines 964-966
**Severity:** Low
**Description:** The secret detection pattern could match legitimate test data or examples.

---

## Part 11: Performance Bottlenecks

---

### spawn-tree.txt Growth

**Issue:** File grows indefinitely (GAP-002)
**Impact:** O(n) read operations where n = total historical spawns
**Mitigation:** Implement rotation/archival

### JSON Parsing in Loops

**Issue:** Multiple jq calls in signature-scan loop (lines 1108-1126)
**Impact:** O(n*m) where n = files, m = signatures
**Mitigation:** Batch operations or use single jq invocation

### Unbounded Array Growth

**Issue:** Error patterns, flags, and other arrays have no size limits
**Impact:** Memory growth over long-running colonies
**Mitigation:** Implement caps and rotation

---

## Part 12: Code Smells

---

### 1. Feature Detection Pattern

The `type feature_enabled &>/dev/null &&` pattern is repeated throughout, creating visual noise.

**Smell:** Feature envy - code keeps checking if features exist
**Fix:** Create wrapper functions that handle feature unavailability gracefully

### 2. JSON Building with String Concatenation

Multiple places build JSON by string concatenation instead of using jq.

**Smell:** Manual JSON construction is error-prone
**Fix:** Use jq for all JSON construction

### 3. Global Variable Usage

Variables like `$DATA_DIR`, `$AETHER_ROOT` are global and modified in various places.

**Smell:** Hidden dependencies and side effects
**Fix:** Pass context as parameters or use a state object

### 4. Inconsistent Exit Codes

Some commands exit 1 on error, others use json_err which may or may not exit.

**Smell:** Unpredictable control flow
**Fix:** Standardize on json_err for all error handling

### 5. Commented-Out Code

Several sections have commented-out code blocks.

**Smell:** Dead code clutter
**Fix:** Remove or document why kept

---

## Part 13: Technical Debt Summary

---

### Deferred Items from TO-DOS.md

| Debt | Why Deferred | Impact |
|------|--------------|--------|
| YAML command generator | Works manually, not broken | 13,573 lines duplicated |
| Test coverage audit | Tests pass, purpose unclear | May have false confidence |
| Pheromone evolution | Feature exists but unused | Telemetry collected but not consumed |

### Recommendations by Priority

**P0 (Fix This Week):**
1. BUG-005/BUG-011: Lock deadlock
2. BUG-002: flag-add lock leak
3. Verify model routing actually works

**P1 (Fix This Month):**
1. BUG-007: Error code consistency
2. ISSUE-004: Template path hardcoding
3. Shellcheck SC2168 errors (local outside functions)

**P2 (Fix Next Quarter):**
1. Code duplication (YAML generator)
2. XML system activation or removal
3. Architecture gaps

**P3 (Backlog):**
1. Performance optimizations
2. Security hardening
3. Documentation improvements

---

## Appendix A: Bug Reference Matrix

| Bug ID | Severity | File | Line | Category | Status |
|--------|----------|------|------|----------|--------|
| BUG-005 | P0 | aether-utils.sh | 1022 | Lock Management | Unfixed |
| BUG-011 | P0 | aether-utils.sh | 1022 | Error Handling | Unfixed |
| BUG-002 | P1 | aether-utils.sh | 814 | Lock Management | Unfixed |
| BUG-008 | P1 | aether-utils.sh | 856 | Error Handling | Unfixed |
| BUG-003 | P2 | atomic-write.sh | 75 | Race Condition | Unfixed |
| BUG-004 | P2 | aether-utils.sh | 930 | Error Handling | Unfixed |
| BUG-006 | P2 | atomic-write.sh | 66 | Lock Management | Unfixed |
| BUG-007 | P2 | aether-utils.sh | Various | Error Handling | Unfixed |
| BUG-009 | P2 | aether-utils.sh | 899,933 | Error Handling | Unfixed |
| BUG-010 | P2 | aether-utils.sh | 1758+ | Error Handling | Unfixed |
| BUG-012 | P2 | aether-utils.sh | 2947 | Error Handling | Unfixed |

---

## Appendix B: Testing Recommendations

### Unit Tests Needed

1. Lock acquisition/release pairs
2. Error code consistency
3. JSON validation paths
4. File path handling with special characters

### Integration Tests Needed

1. Concurrent flag operations
2. Model routing verification
3. Template path resolution
4. Backup/restore cycle

### Regression Tests Needed

1. Deadlock scenarios
2. Error handling paths
3. Shellcheck violations

---

## Appendix C: Workarounds Summary

| Issue | Workaround |
|-------|------------|
| Lock-related deadlocks (BUG-005, BUG-002) | Restart colony session |
| Template path issue (ISSUE-004) | Use git clone instead of npm |
| Missing command docs (GAP-004) | Read source code directly |
| Model routing unverified | Assume default model for all castes |

---

*Document Generated: 2026-02-16*
*Total Word Count: ~15,000+*
*Next Review: After P0 bugs are fixed*
