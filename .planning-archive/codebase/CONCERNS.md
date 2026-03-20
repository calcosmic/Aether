# Codebase Concerns

**Analysis Date:** 2026-02-17

---

## Tech Debt

### Lock Handling Deadlock Risk

**Issue:** Missing lock release on error paths in flag operations

**Files:** `.aether/aether-utils.sh:814`, `.aether/aether-utils.sh:1022`

**Impact:** If jq command fails during flag resolution, lock is never released causing deadlock on `flags.json`

**Fix approach:** Add trap-based cleanup or ensure `release_lock` in all exit paths. Add error handling with lock release before `json_err` calls

---

### Error Code Inconsistency

**Issue:** 17+ locations use hardcoded strings instead of `$E_*` constants

**Files:** `.aether/aether-utils.sh` various lines (lines 856, 899, 930, 933, 1758+, 2947)

**Impact:** Inconsistent error handling, harder programmatic processing

**Fix approach:** Standardize all to use `json_err "$E_*" "message"` pattern

---

### Template Path Hardcoded

**Issue:** `queen-init` uses `runtime/` directory which may not exist in npm installs

**Files:** `.aether/aether-utils.sh:2689`

**Impact:** `queen-init` fails when Aether is installed as npm package

**Fix approach:** Use `$HOME/.aether/` or detect installation method

---

### Large Monolithic Files

**Issue:** Two files exceed 2000 lines each

**Files:**
- `.aether/aether-utils.sh` - 4050 lines
- `bin/cli.js` - 2295 lines

**Impact:** Hard to navigate, understand, and test. High risk of introducing bugs during modifications

**Fix approach:** Modularize into smaller focused modules. Extract command handlers into separate files

---

### Spawn-Tree Growth

**Issue:** `spawn-tree.txt` grows indefinitely with no cleanup

**Files:** `.aether/aether-utils.sh:402-448`

**Impact:** File could grow very large over many sessions

**Fix approach:** Add periodic cleanup or age-based pruning

---

## Known Bugs

### BUG-005/BUG-011: Lock Deadlock in Flag-Auto-Resolve

**Location:** `.aether/aether-utils.sh:1022`

**Severity:** HIGH

**Symptom:** If jq command fails during flag resolution, lock is never released

**Trigger:** Malformed JSON in flags.json, disk full, or any jq failure

**Workaround:** Restart colony session if commands hang on flag operations

---

### BUG-002: Missing Release_Lock in Flag-Add Error Path

**Location:** `.aether/aether-utils.sh:814`

**Severity:** MEDIUM

**Symptom:** If acquire_lock succeeds but jq fails, lock is never released

**Trigger:** jq command fails after acquiring lock

---

### BUG-003: Race Condition in Backup Creation

**Location:** `.aether/utils/atomic-write.sh:75`

**Severity:** MEDIUM

**Symptom:** Backup created AFTER temp file validation but BEFORE atomic move

**Trigger:** Process crashes between validation and backup

---

### BUG-007: Error Code Standardization

**Location:** `.aether/aether-utils.sh` multiple locations

**Severity:** MEDIUM

**Symptom:** Commands use hardcoded strings instead of error constants

**Impact:** Inconsistent error handling across commands

---

### ISSUE-004: Queen-Init Fails via NPM

**Location:** `.aether/aether-utils.sh:2689`

**Severity:** MEDIUM

**Symptom:** Template path hardcoded to `runtime/` which doesn't exist in npm installs

**Workaround:** Use git clone instead of npm install

---

## Security Considerations

### Dangerous Command Execution

**Risk:** Scripts execute `rm -rf` and other destructive operations

**Files:** `.aether/aether-utils.sh:3818`, `.aether/utils/xml-utils.sh:414`, `.aether/utils/spawn-tree.sh:217`

**Current mitigation:** Block destructive operations hook exists in `.aether/docs/` but not actively enforced

**Recommendations:**
- Add explicit user confirmation for all destructive operations
- Implement dry-run modes for dangerous commands
- Add audit logging for file deletions

---

### Shell Injection Risk

**Risk:** Variable interpolation without proper quoting

**Files:** `.aether/aether-utils.sh` (variable quoting patterns)

**Current mitigation:** Uses `set -u` and quotes most variables

**Recommendations:**
- Audit all variable interpolations for edge cases
- Add shellcheck to CI pipeline

---

### Git Stash Data Loss Risk

**Risk:** Build checkpoint could stash user work (historical issue)

**Files:** Commands that use `git stash`

**Current mitigation:** Checkpoint allowlist system implemented - only allowlisted system files are stashed

**Recommendations:**
- Add pre-stash validation to warn about any modified user files
- Consider using `git stash push --staged` instead of full stash

---

## Performance Bottlenecks

### JSON Parsing Overhead

**Problem:** Multiple full-file reads and parses for state operations

**Files:** `.aether/aether-utils.sh` - state management functions

**Cause:** No caching, each operation reads and parses entire JSON file

**Improvement path:** Implement in-memory caching with write-through strategy

---

### Command Generation Duplication

**Problem:** 13,573 lines duplicated between `.claude/commands/` and `.opencode/commands/`

**Files:** Both command directories

**Cause:** Manual duplication instead of single-source generation

**Improvement path:** Build YAML-based command generator (noted in TO-DOs)

---

## Fragile Areas

### Error Handling in json_err Fallback

**Location:** `.aether/aether-utils.sh:65-72`

**Why fragile:** Fallback `json_err` doesn't accept error code parameter - if error-handler.sh fails to load, error codes are lost

**Safe modification:** Always test error paths when modifying command handlers

---

### Feature Detection Race Condition

**Location:** `.aether/aether-utils.sh:33-45`

**Why fragile:** Feature detection runs before error handler fully sourced - timing dependencies

**Safe modification:** Move feature detection after error handler initialization

---

### Queen-Read JSON Validation

**Location:** `.aether/aether-utils.sh` - queen-read command

**Why fragile:** Builds JSON but doesn't validate before returning - could return malformed response

**Safe modification:** Add JSON validation before returning response

---

### Context-Update No Locking

**Location:** `.aether/aether-utils.sh:1758+`

**Why fragile:** No file locking on context-update - race condition possible during concurrent updates

**Safe modification:** Add file locking before write operations

---

## Scaling Limits

### File Lock Contention

**Resource:** `.aether/locks/` directory

**Current capacity:** Single lock per operation

**Limit:** Multiple concurrent operations will block

**Scaling path:** Implement lock timeout and retry mechanism

---

### Session State File Size

**Resource:** `COLONY_STATE.json`

**Current capacity:** Unbounded JSON growth

**Limit:** Performance degradation with large state files

**Scaling path:** Implement state archiving and pagination

---

### Worker Spawn Limits

**Resource:** Claude Code/OpenCode worker processes

**Current capacity:** Max depth 3, max 4 at depth 1, max 2 at depth 2

**Limit:** Cannot spawn beyond depth 3

**Scaling path:** Worker pool system for horizontal scaling

---

## Dependencies at Risk

### Minimal Production Dependencies

**Packages:** `commander`, `js-yaml`, `picocolors`

**Risk:** LOW - these are stable, well-maintained packages

**Migration plan:** None needed

---

### Dev Dependencies

**Packages:** `ava`, `proxyquire`, `sinon`

**Risk:** LOW - standard testing tools

**Note:** These are only used in development, not shipped with package

---

## Missing Critical Features

### Model Routing Verification

**Feature gap:** Model-per-caste routing exists in configuration but unproven in execution

**Problem:** Workers may not inherit `ANTHROPIC_MODEL` environment variable

**Blocks:** Proper worker specialization by capability

**Priority:** HIGH

---

### XML Deep Integration

**Feature gap:** XML utilities exist but not integrated into workflow

**Problem:** Pheromones, queen wisdom, and cross-colony sharing not using XML

**Blocks:** Eternal storage and colony-to-colony transfer

**Priority:** HIGH

---

### Oracle Timestamp Verification

**Feature gap:** Oracle spawns long-running agents without session freshness checks

**Problem:** Stale progress files from interrupted Oracle sessions

**Blocks:** Reliable deep research workflow

**Priority:** MEDIUM

---

## Test Coverage Gaps

### Error Path Coverage

**What's not tested:** Error handling paths in commands

**Files:** Most command handlers

**Risk:** Bugs in error handling go undetected until production

**Priority:** HIGH

---

### Concurrent Operation Testing

**What's not tested:** Multiple ants modifying state simultaneously

**Files:** State management functions

**Risk:** Race conditions not caught before production

**Priority:** HIGH

---

### Integration Tests for Flag System

**What's not tested:** Flag auto-resolve with various failure scenarios

**Files:** Flag operations in `.aether/aether-utils.sh`

**Risk:** Lock deadlocks only discovered when they happen

**Priority:** HIGH

---

### Shell Script Tests

**What's not tested:** Full coverage of `aether-utils.sh` functions

**Location:** `tests/bash/`

**Risk:** Edge cases in shell scripts may fail in production

**Priority:** MEDIUM

---

## Deferred Technical Debt

| Debt | Why Deferred | Impact |
|------|--------------|--------|
| YAML command generator | Works manually, not broken | 13,573 lines duplicated |
| Test coverage audit | Tests pass, purpose unclear | May have false confidence |
| Pheromone evolution | Feature exists but unused | Telemetry collected but not consumed |

---

## Workarounds Summary

| Issue | Workaround |
|-------|------------|
| Lock deadlock (BUG-005) | Restart colony session |
| Template path (ISSUE-004) | Use git clone instead of npm |
| Missing command docs | Read source code directly |
| Model routing unverified | Use default model for all workers |

---

*Concerns audit: 2026-02-17*
