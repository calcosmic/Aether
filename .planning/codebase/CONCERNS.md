# Codebase Concerns

**Analysis Date:** 2026-02-17

---

## Critical Issues (Fix Immediately)

### BUG-005/BUG-011: Lock Deadlock in Flag-Auto-Resolve
- **Issue:** If `jq` command fails during flag resolution, lock is never released
- **Files:** `.aether/aether-utils.sh:1022`, `.aether/aether-utils.sh:1385-1407`
- **Impact:** Deadlock on `flags.json` if jq fails (malformed JSON, disk full, etc.)
- **Fix approach:** Add error handling with lock release before `json_err`. Use trap-based cleanup to ensure locks are released in all exit paths.

### BUG-002: Missing Lock Release in Flag-Add
- **Issue:** If `acquire_lock` succeeds but `jq` fails, lock is never released
- **Files:** `.aether/aether-utils.sh:814`, `.aether/aether-utils.sh:1188`, `.aether/aether-utils.sh:1227`
- **Impact:** Potential deadlock on file operations
- **Fix approach:** Use trap-based cleanup or ensure `release_lock` in all exit paths

---

## Known Bugs

### BUG-003: Race Condition in Backup Creation
- **Symptom:** Backup created AFTER temp file validation but BEFORE atomic move
- **Files:** `.aether/utils/atomic-write.sh:75`
- **Trigger:** Process crashes between validation and backup
- **Fix approach:** Create backup BEFORE validation, or use transactional approach

### BUG-004: Missing Error Code in Flag-Acknowledge
- **Issue:** Uses hardcoded string instead of `$E_VALIDATION_FAILED`
- **Files:** `.aether/aether-utils.sh:930`
- **Fix approach:** Change to `json_err "$E_VALIDATION_FAILED" "Usage: ..."`

### BUG-006: No Lock Release on JSON Validation Failure
- **Issue:** If JSON validation fails, temp file cleaned but lock not released
- **Files:** `.aether/utils/atomic-write.sh:66`
- **Fix approach:** Document lock ownership contract clearly

### BUG-007: 17+ Instances of Missing Error Codes
- **Issue:** Commands use hardcoded strings instead of error constants
- **Files:** `.aether/aether-utils.sh` various lines
- **Impact:** Inconsistent error handling, harder programmatic processing
- **Fix approach:** Standardize all to use `json_err "$E_*" "message"` pattern

### BUG-008: Missing Error Code in Flag-Add jq Failure
- **Issue:** Lock released but error code missing on jq failure
- **Files:** `.aether/aether-utils.sh:856`
- **Fix approach:** Change to `json_err "$E_JSON_INVALID" "Failed to add flag"`

### BUG-009: Missing Error Codes in File Checks
- **Issue:** File not found errors use hardcoded strings
- **Files:** `.aether/aether-utils.sh:899`, `.aether/aether-utils.sh:933`
- **Fix approach:** Use `json_err "$E_FILE_NOT_FOUND" "..."`

### BUG-010: Missing Error Codes in Context-Update
- **Issue:** Various error paths lack error code constants
- **Files:** `.aether/aether-utils.sh:1758+`

### BUG-012: Missing Error Code in Unknown Command
- **Issue:** Unknown command handler uses bare string
- **Files:** `.aether/aether-utils.sh:2947`

---

## Architecture Issues

### ISSUE-001: Inconsistent Error Code Usage
- **Description:** Some `json_err` calls use hardcoded strings instead of constants
- **Pattern:** Commands added early use strings; later commands use constants
- **Files:** Multiple locations

### ISSUE-004: Template Path Hardcoded to Runtime/
- **Issue:** `queen-init` uses `runtime/` directory which may not exist in npm installs
- **Files:** `.aether/aether-utils.sh:2689`
- **Impact:** `queen-init` will fail when Aether is installed as npm package
- **Workaround:** Use git clone instead of npm install

---

## Security Considerations

### Hardcoded Secret Detection
- **Current:** `aether-utils.sh:976-979` has exposed secrets check
- **Pattern:** Scans for `api_key`, `apikey`, `secret`, `password`, `token` patterns
- **Concern:** Detection is basic - could miss encoded/obfuscated secrets
- **Recommendation:** Consider adding entropy detection for harder-to-spot secrets

### Dangerous Permission Flag in Oracle
- **Location:** `.aether/oracle/oracle.sh:41`
- **Pattern:** `claude --dangerously-skip-permissions --print`
- **Concern:** Oracle uses dangerous permission skip flag
- **Risk:** Could execute actions beyond intended scope

---

## Performance Bottlenecks

### Large Single File
- **File:** `.aether/aether-utils.sh` (3,847 lines)
- **Problem:** Monolithic utility file is hard to navigate and maintain
- **Impact:** Slower shell sourcing, harder debugging
- **Improvement path:** Consider modularization by command category

### Spawn-Tree Growth
- **Issue:** `spawn-tree.txt` grows indefinitely
- **Files:** `.aether/aether-utils.sh:402-448`, `.aether/utils/spawn-tree.sh:222-263`
- **Impact:** File could grow very large over many sessions
- **Improvement path:** Add cleanup for stale spawn-tree entries

### No Retry Logic for Failed Spawns
- **Issue:** Task tool calls don't have retry logic
- **Impact:** Transient failures cause build failures
- **Improvement path:** Add retry logic with exponential backoff

---

## Fragile Areas

### Error Handling Without Fallback
- **Files:** `.aether/aether-utils.sh:2132-2144`
- **Why fragile:** `model-get` and `model-list` use `exec` without fallback
- **Safe modification:** Add try-catch pattern or validate exec success

### Feature Detection Race Condition
- **Files:** `.aether/aether-utils.sh:33-45`
- **Why fragile:** Feature detection runs before error handler fully sourced
- **Safe modification:** Source error handler before feature detection

### Incomplete Help Command
- **Files:** `.aether/aether-utils.sh:106-111`
- **Why fragile:** Help command missing newer commands like `queen-*`, `view-state-*`, `swarm-timing-*`
- **Safe modification:** Update help text when adding new commands

---

## Scaling Limits

### Session Freshness Detection
- **Current:** Implemented for 9 commands (colonize, oracle, watch, swarm, init, seal, entomb)
- **Limit:** Not all commands have timestamp verification
- **Improvement path:** Apply pattern to remaining commands

### Model Routing Unverified
- **Status:** Configuration exists (`model-profiles.yaml` maps castes to models)
- **Limit:** Execution unproven - `ANTHROPIC_MODEL` may not be inherited by spawned workers
- **Test:** `/ant:verify-castes` Step 3 spawns test worker
- **Blocked by:** Anthropic token exhaustion (proxy auth fails)

---

## Dependencies at Risk

### Error Handler Dependency
- **Files:** `.aether/aether-utils.sh:65-72`
- **Risk:** Fallback `json_err` doesn't accept error code parameter
- **Impact:** If `error-handler.sh` fails to load, error codes are lost

### jq Dependency (Implicit)
- **Risk:** Several commands fail without explicit handling if jq unavailable
- **Impact:** Silent failures in lock/error operations

---

## Missing Critical Features

### GAP-001: No Schema Version Validation
- **Problem:** Commands assume state structure without validating version
- **Impact:** Silent failures when state structure changes

### GAP-004: Missing Queen-* Documentation
- **Problem:** No docs for `queen-init`, `queen-read`, `queen-promote`
- **Impact:** Users cannot discover wisdom feedback loop

### GAP-005: No Validation of Queen-Read JSON Output
- **Problem:** `queen-read` builds JSON but doesn't validate before returning
- **Impact:** Could return malformed response

### GAP-007: No Error Code Standards Documentation
- **Problem:** Error codes exist but aren't documented
- **Impact:** Developers don't know which codes to use

### GAP-009: Context-Update Has No File Locking
- **Problem:** Race condition possible during concurrent context updates
- **Impact:** Potential data corruption

---

## Test Coverage Gaps

### Untested Error Paths
- **What's not tested:** Error handling paths (17+ error code inconsistencies)
- **Files:** `.aether/aether-utils.sh` various locations
- **Risk:** Bugs in error handling go undetected
- **Priority:** High

### Lock Release Verification
- **What's not tested:** Lock release in all error paths
- **Risk:** Deadlocks only appear under failure conditions
- **Priority:** Critical

---

## Tech Debt Summary

| Category | Count | Priority |
|----------|-------|----------|
| Lock-related bugs | 4 | Critical |
| Missing error codes | 7 | Medium |
| Architecture issues | 4 | Low |
| Security concerns | 2 | Medium |
| Performance issues | 3 | Medium |
| Test coverage gaps | 2 | High |

---

*Concerns audit: 2026-02-17*
