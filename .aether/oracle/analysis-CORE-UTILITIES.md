# Core Utility Layer Analysis

## Executive Summary

This document provides a comprehensive technical analysis of the Aether colony system's core utility layer, covering all functions, bugs, and improvement opportunities across the entire codebase.

## File Statistics

| File | Lines | Functions | Purpose |
|------|-------|-----------|---------|
| `aether-utils.sh` | 3,593 | 80+ commands | Main utility dispatcher |
| `utils/file-lock.sh` | 123 | 7 functions | File locking mechanism |
| `utils/atomic-write.sh` | 218 | 8 functions | Atomic file operations |
| `utils/error-handler.sh` | 201 | 12 functions | Error handling & feature flags |
| `utils/chamber-utils.sh` | 286 | 5 functions | Chamber (archive) management |
| `utils/spawn-tree.sh` | 429 | 6 functions | Spawn tree tracking |
| `utils/xml-utils.sh` | 2,162 | 35+ functions | XML processing & pheromones |
| `utils/xml-compose.sh` | 248 | 4 functions | XInclude composition |
| `utils/state-loader.sh` | 216 | 4 functions | State loading with locks |
| `utils/swarm-display.sh` | 269 | 10 functions | Real-time swarm visualization |
| `utils/watch-spawn-tree.sh` | 254 | 5 functions | Live spawn tree view |
| `utils/colorize-log.sh` | 133 | 3 functions | Colorized log streaming |
| `utils/spawn-with-model.sh` | 57 | 1 function | Model-aware spawning |
| `utils/chamber-compare.sh` | 181 | 3 functions | Chamber comparison |
| **TOTAL** | **8,298** | **~190 functions** | |

---

## Function Catalog: aether-utils.sh

### JSON Output Helpers

#### `json_ok()` (line 62)
- **Purpose**: Output successful JSON response
- **Inputs**: JSON string to wrap
- **Outputs**: `{"ok":true,"result":<input>}`
- **Dependencies**: None
- **Issues**: None

#### `json_err()` (line 66-73 fallback, enhanced in error-handler.sh)
- **Purpose**: Output error JSON and exit
- **Inputs**: code, message, details, recovery
- **Outputs**: JSON to stderr, exits with code 1
- **Dependencies**: error-handler.sh (if available)
- **Issues**: Fallback version doesn't use error constants

### Caste System

#### `get_caste_emoji()` (line 76-102)
- **Purpose**: Return emoji for caste identification
- **Inputs**: Caste name or worker name
- **Outputs**: Emoji string (e.g., "🔨🐜" for builder)
- **Dependencies**: None
- **Issues**: Duplicate pattern matching (line 82-83 both match Chart/Plot)

### Context Management

#### `_cmd_context_update()` (line 108-449)
- **Purpose**: Multi-action context file updater
- **Inputs**: action, various args depending on action
- **Outputs**: JSON status
- **Dependencies**: jq, sed
- **Issues**:
  - Line 446: Uses `$E_VALIDATION_FAILED` before it's defined (defined in error-handler.sh which is sourced later)
  - Heavy use of sed for JSON manipulation (fragile)

**Sub-functions:**
- `ensure_context_dir()` (line 115-119): Creates context directory
- `read_colony_state()` (line 121-132): Reads COLONY_STATE.json

### Command Handlers (case statement entries)

#### `help` (line 456-460)
- Lists all available commands
- Hardcoded command list (maintenance risk)

#### `version` (line 461-462)
- Returns "1.0.0"

#### `validate-state` (line 464-507)
- **Purpose**: Validate colony state files
- **Sub-commands**: colony, constraints, all
- **Issues**: None significant

#### `error-add` (line 509-526)
- **Purpose**: Add error record to COLONY_STATE.json
- **Inputs**: category, severity, description, [phase]
- **Issues**: None

#### `error-pattern-check` (line 527-535)
- **Purpose**: Check for recurring error patterns
- **Issues**: None

#### `error-summary` (line 536-543)
- **Purpose**: Summarize errors by category/severity
- **Issues**: None

#### `activity-log` (line 544-564)
- **Purpose**: Log activity with caste emoji
- **Issues**: None

#### `activity-log-init` (line 565-596)
- **Purpose**: Initialize phase logging
- **Issues**: None

#### `activity-log-read` (line 597-614)
- **Purpose**: Read activity log with optional filter
- **Issues**: None

#### `learning-promote` (line 615-658)
- **Purpose**: Promote learning to global registry
- **Issues**: None

#### `learning-inject` (line 659-679)
- **Purpose**: Inject relevant learnings by keyword
- **Issues**: None

#### `spawn-log` (line 680-700)
- **Purpose**: Log spawn events to activity and spawn-tree.txt
- **Format**: timestamp|parent|caste|child_name|task|model|status
- **Issues**: None

#### `spawn-complete` (line 701-719)
- **Purpose**: Log completion events
- **Issues**: None

#### `spawn-can-spawn` (line 720-752)
- **Purpose**: Check if spawning is allowed at given depth
- **Limits**: depth 1→4 spawns, depth 2→2 spawns, depth 3+→0
- **Global cap**: 10 workers per phase
- **Issues**: None

#### `spawn-get-depth` (line 753-799)
- **Purpose**: Calculate spawn depth for an ant
- **Issues**: None

#### `update-progress` (line 800-850)
- **Purpose**: Generate progress display file
- **Issues**: None

#### `error-flag-pattern` (line 851-901)
- **Purpose**: Track recurring error patterns
- **Issues**: None

#### `error-patterns-check` (line 902-917)
- **Purpose**: Check for known error patterns
- **Issues**: None

#### `check-antipattern` (line 918-989)
- **Purpose**: Scan files for antipatterns
- **Supported**: Swift, TypeScript/JavaScript, Python
- **Issues**: None

#### `signature-scan` (line 990-1041)
- **Purpose**: Scan file for signature pattern
- **Issues**: None

#### `signature-match` (line 1042-1139)
- **Purpose**: Match signatures across directory
- **Issues**: None

#### `flag-add` (line 1140-1212)
- **Purpose**: Add project flag (blocker/issue/note)
- **Issues**:
  - Lines 1161-1173: Lock acquisition with graceful degradation
  - Line 1207: Lock release on jq failure - potential BUG-005/BUG-011

#### `flag-check-blockers` (line 1213-1241)
- **Purpose**: Count unresolved blockers
- **Issues**: None

#### `flag-resolve` (line 1242-1276)
- **Purpose**: Resolve a flag
- **Issues**: Lock handling similar to flag-add

#### `flag-acknowledge` (line 1277-1309)
- **Purpose**: Acknowledge a flag
- **Issues**: Lock handling

#### `flag-list` (line 1310-1349)
- **Purpose**: List flags with filters
- **Issues**: None

#### `flag-auto-resolve` (line 1350-1391)
- **CRITICAL BUG**: BUG-005/BUG-011 Lock Deadlock
- **Location**: Lines 1367-1384
- **Description**: If jq fails after lock acquisition, lock is never released
- **Current code**:
```bash
count=$(jq --arg trigger "$trigger" '[.flags[] | select(.auto_resolve_on == $trigger and .resolved_at == null)] | length' "$flags_file") || {
  release_lock "$flags_file" 2>/dev/null || true  # Line 1371 - releases here
  json_err "$E_JSON_INVALID" "Failed to count flags for auto-resolve"
}
# ... more jq operations without lock release on failure
```
- **Fix**: Add lock release to all error paths or use trap

#### `generate-ant-name` (line 1392-1424)
- **Purpose**: Generate caste-specific ant names
- **Issues**: None

#### `autofix-checkpoint` (line 1430-1467)
- **Purpose**: Create git checkpoint before autofix
- **Issues**: None

#### `autofix-rollback` (line 1469-1506)
- **Purpose**: Rollback from checkpoint
- **Issues**: None

#### `spawn-can-spawn-swarm` (line 1508-1529)
- **Purpose**: Check swarm spawn capacity
- **Issues**: None

#### `swarm-findings-init` (line 1531-1548)
- **Purpose**: Initialize swarm findings file
- **Issues**: None

#### `swarm-findings-add` (line 1550-1578)
- **Purpose**: Add finding from scout
- **Issues**: None

#### `swarm-findings-read` (line 1580-1590)
- **Purpose**: Read swarm findings
- **Issues**: None

#### `swarm-solution-set` (line 1592-1613)
- **Purpose**: Set chosen solution
- **Issues**: None

#### `swarm-cleanup` (line 1615-1637)
- **Purpose**: Clean up swarm files
- **Issues**: None

#### `grave-add` (line 1639-1687)
- **Purpose**: Record failure marker
- **Issues**: None

#### `grave-check` (line 1689-1709)
- **Purpose**: Query grave markers
- **Issues**: None

#### `generate-commit-message` (line 1715-1797)
- **Purpose**: Generate commit messages
- **Issues**: None

#### `context-update` (line 1803-1823)
- **Purpose**: Dispatch to _cmd_context_update
- **Issues**: None

#### `version-check` (line 1829-1850)
- **Purpose**: Check for updates
- **Issues**: None

#### `registry-add` (line 1852-1893)
- **Purpose**: Add repo to registry
- **Issues**: None

#### `bootstrap-system` (line 1895-1943)
- **Purpose**: Copy system files from hub
- **Issues**: None

#### `load-state` / `unload-state` (line 1945-1969)
- **Purpose**: State loading wrappers
- **Issues**: None

#### `spawn-tree-load` / `spawn-tree-active` / `spawn-tree-depth` (line 1971-1998)
- **Purpose**: Spawn tree wrappers
- **Issues**: None

#### `model-profile` (line 2001-2120)
- **Purpose**: Model profile management
- **Sub-commands**: get, list, verify, select, validate
- **Issues**:
  - Line 2117: Uses `$E_VALIDATION_FAILED` in error path

#### `model-get` / `model-list` (line 2122-2134)
- **Purpose**: Shortcuts for model-profile
- **Issues**: None

#### `chamber-create` / `chamber-verify` / `chamber-list` (line 2136-2175)
- **Purpose**: Chamber management wrappers
- **Issues**: None

#### `milestone-detect` (line 2177-2238)
- **Purpose**: Detect colony milestone from state
- **Issues**: None

#### `swarm-activity-log` (line 2244-2257)
- **Purpose**: Log swarm activity
- **Issues**: None

#### `swarm-display-init` (line 2259-2282)
- **Purpose**: Initialize swarm display
- **Issues**: None

#### `swarm-display-update` (line 2284-2378)
- **Purpose**: Update ant activity in display
- **Issues**: None

#### `swarm-display-get` (line 2380-2390)
- **Purpose**: Get display state
- **Issues**: None

#### `swarm-display-render` (line 2392-2406)
- **Purpose**: Render display to terminal
- **Issues**: None

#### `swarm-timing-start` (line 2408-2427)
- **Purpose**: Record start time
- **Issues**: None

#### `swarm-timing-get` (line 2429-2456)
- **Purpose**: Get elapsed time
- **Issues**: None

#### `swarm-timing-eta` (line 2458-2505)
- **Purpose**: Calculate ETA
- **Issues**: None

#### `view-state-init` / `view-state-get` / `view-state-set` / `view-state-toggle` / `view-state-expand` / `view-state-collapse` (line 2511-2673)
- **Purpose**: View state management
- **Issues**: None

#### `queen-init` (line 2675-2717)
- **Purpose**: Initialize QUEEN.md from template
- **Issues**:
  - Line 2689: BUG-004 - Template path hardcoded to runtime/
  - Line 2712: Hardcoded source path in success message

#### `queen-read` (line 2719-2773)
- **Purpose**: Read QUEEN.md as JSON
- **Issues**: None

#### `queen-promote` (line 2775-2948)
- **Purpose**: Promote learning to QUEEN.md
- **Issues**: Complex sed/awk manipulation - fragile

#### `survey-load` / `survey-verify` (line 2950-3009)
- **Purpose**: Survey document management
- **Issues**: None

#### `checkpoint-check` (line 3011-3080)
- **Purpose**: Check which files are system vs user
- **Issues**: None

#### `normalize-args` (line 3082-3114)
- **Purpose**: Normalize Claude Code vs OpenCode args
- **Issues**: None

#### `survey-verify-fresh` / `survey-clear` (line 3117-3160)
- **Purpose**: Backward compatibility wrappers
- **Issues**: None

#### `session-verify-fresh` (line 3162-3277)
- **Purpose**: Generic session freshness verification
- **Issues**:
  - Line 3181: Uses `$E_VALIDATION_FAILED` constant
  - Lines 3241: Cross-platform stat command (macOS vs Linux)

#### `session-clear` (line 3279-3360)
- **Purpose**: Generic session file clearing
- **Issues**: None

#### `pheromone-export` (line 3362-3382)
- **Purpose**: Export pheromones to XML
- **Issues**:
  - Line 3380: Uses undefined `$E_DEPENDENCY_MISSING`

#### `session-init` / `session-update` / `session-read` / `session-is-stale` / `session-clear` / `session-mark-resumed` / `session-summary` (line 3388-3587)
- **Purpose**: Session continuity management
- **Issues**:
  - Line 3430: Uses `local` outside function (in main case handler)
  - Line 3490: macOS-specific date command (`date -j -f`)
  - Line 3520: Same macOS date issue
  - Line 3556: Uses undefined `$E_RESOURCE_NOT_FOUND`

---

## Function Catalog: utils/file-lock.sh

#### `acquire_lock()` (line 30-66)
- **Purpose**: Acquire file lock using noclobber
- **Issues**: None significant

#### `release_lock()` (line 68-78)
- **Purpose**: Release acquired lock
- **Issues**: None

#### `cleanup_locks()` (line 80-85)
- **Purpose**: Cleanup on exit
- **Issues**: None

#### `is_locked()` (line 91-95)
- **Purpose**: Check if file is locked
- **Issues**: None

#### `get_lock_holder()` (line 97-102)
- **Purpose**: Get PID of lock holder
- **Issues**: None

#### `wait_for_lock()` (line 104-119)
- **Purpose**: Wait for lock release
- **Issues**: None

---

## Function Catalog: utils/atomic-write.sh

#### `atomic_write()` (line 47-92)
- **Purpose**: Atomic file write via temp+rename
- **Issues**: None

#### `atomic_write_from_file()` (line 94-147)
- **Purpose**: Atomic copy from source file
- **Issues**: None

#### `create_backup()` (line 149-163)
- **Purpose**: Create backup before write
- **Issues**: None

#### `rotate_backups()` (line 165-174)
- **Purpose**: Rotate old backups
- **Issues**: None

#### `restore_backup()` (line 176-198)
- **Purpose**: Restore from backup
- **Issues**: None

#### `list_backups()` (line 200-208)
- **Purpose**: List available backups
- **Issues**: None

#### `cleanup_temp_files()` (line 210-213)
- **Purpose**: Clean old temp files
- **Issues**: None

---

## Function Catalog: utils/error-handler.sh

#### `json_err()` (line 47-86)
- **Purpose**: Enhanced error output
- **Issues**: None

#### `json_warn()` (line 88-108)
- **Purpose**: Non-fatal warning output
- **Issues**: None

#### `error_handler()` (line 113-138)
- **Purpose**: Trap ERR handler
- **Issues**: None

#### `feature_enable()` / `feature_disable()` / `feature_enabled()` / `feature_log_degradation()` (line 147-189)
- **Purpose**: Feature flag management
- **Issues**: None

---

## Function Catalog: utils/chamber-utils.sh

#### `chamber_sanitize_goal()` (line 40-51)
- **Purpose**: Sanitize goal for directory name
- **Issues**: None

#### `chamber_compute_hash()` (line 53-71)
- **Purpose**: Compute SHA256 hash
- **Issues**: None

#### `chamber_create()` (line 74-162)
- **Purpose**: Create chamber archive
- **Issues**: None

#### `chamber_verify()` (line 164-216)
- **Purpose**: Verify chamber integrity
- **Issues**: None

#### `chamber_list()` (line 218-282)
- **Purpose**: List all chambers
- **Issues**: None

---

## Function Catalog: utils/spawn-tree.sh

#### `parse_spawn_tree()` (line 15-218)
- **Purpose**: Parse spawn-tree.txt to JSON
- **Issues**: None

#### `get_spawn_depth()` (line 220-264)
- **Purpose**: Calculate spawn depth
- **Issues**: None

#### `get_active_spawns()` (line 266-326)
- **Purpose**: Get active spawns
- **Issues**: None

#### `get_spawn_children()` (line 328-368)
- **Purpose**: Get children of spawn
- **Issues**: None

#### `get_spawn_lineage()` (line 370-410)
- **Purpose**: Get full lineage
- **Issues**: None

#### `reconstruct_tree_json()` (line 412-418)
- **Purpose**: Reconstruct full tree
- **Issues**: None

---

## Function Catalog: utils/xml-utils.sh

This is a large file with many functions. Key ones:

#### `xml-validate()` (line 60-88)
#### `xml-well-formed()` (line 90-114)
#### `xml-to-json()` (line 116-213)
#### `json-to-xml()` (line 215-262)
#### `xml-query()` (line 264-294)
#### `xml-query-attr()` (line 296-326)
#### `xml-merge()` (line 328-384)
#### `xml-format()` (line 386-416)
#### `xml-escape()` / `xml-unescape()` (line 418-446)
#### `xml-detect-tools()` (line 448-453)
#### `pheromone-to-xml()` (line 459-913)
#### `pheromone-from-xml()` (line 915-950)
#### `queen-wisdom-to-xml()` (line 956-996)
#### `queen-wisdom-from-xml()` (line 998-1032)
#### `registry-to-xml()` / `registry-from-xml()` (line 1038-1115)
#### `pheromone-export()` (line 1121-1383)
#### `generate-colony-namespace()` (line 1395-1415)
#### `generate-cross-colony-prefix()` (line 1417-1451)
#### `prefix-pheromone-id()` (line 1453-1471)
#### `extract-session-from-namespace()` (line 1473-1490)
#### `validate-colony-namespace()` (line 1492-1509)
#### `queen-wisdom-to-markdown()` (line 1515-1590)
#### `queen-wisdom-validate-entry()` (line 1611-1700)
#### `queen-wisdom-promote()` (line 1702-1770)
#### `queen-wisdom-import()` (line 1772-1864)
#### `prompt-to-xml()` (line 1870-1981)
#### `prompt-from-xml()` (line 1983-2111)
#### `prompt-validate()` (line 2113-2128)

---

## Bug Inventory

### BUG-005/BUG-011: Lock Deadlock in flag-auto-resolve
- **File**: `.aether/aether-utils.sh`
- **Line**: 1367-1384
- **Severity**: P0 (Critical)
- **Description**: If jq fails after lock acquisition, lock is never released causing deadlock
- **Fix**: Add trap-based lock release or ensure release on all error paths

### BUG-004: Template Path Hardcoded to runtime/
- **File**: `.aether/aether-utils.sh`
- **Line**: 2689
- **Severity**: P1 (High)
- **Description**: queen-init fails when Aether installed via npm because template path hardcoded to runtime/
- **Fix**: Check multiple paths as already done, but fix success message

### BUG-007: Error Code Inconsistency
- **File**: `.aether/aether-utils.sh`
- **Lines**: Multiple (446, 2117, 3181, 3380, 3556)
- **Severity**: P2 (Medium)
- **Description**: Mix of hardcoded strings and `$E_*` constants
- **Fix**: Standardize on error constants throughout

### BUG-008: Undefined Error Constants
- **File**: `.aether/aether-utils.sh`
- **Lines**: 3380, 3556
- **Severity**: P2 (Medium)
- **Description**: Uses `$E_DEPENDENCY_MISSING` and `$E_RESOURCE_NOT_FOUND` which don't exist in error-handler.sh
- **Fix**: Add missing constants or use existing ones

### BUG-009: macOS-Specific Date Commands
- **File**: `.aether/aether-utils.sh`
- **Lines**: 3490, 3520
- **Severity**: P2 (Medium)
- **Description**: Uses `date -j -f` which is macOS-specific
- **Fix**: Add Linux fallback

### BUG-010: local Outside Function
- **File**: `.aether/aether-utils.sh`
- **Line**: 3430
- **Severity**: P3 (Low)
- **Description**: `local` keyword used in main case handler context
- **Fix**: Remove `local` or move to function

---

## Improvement Opportunities

### 1. Error Handling Standardization
- Standardize all error handling to use error-handler.sh constants
- Add trap-based cleanup for locks
- Improve error messages with recovery suggestions

### 2. Cross-Platform Compatibility
- Abstract date commands for macOS/Linux
- Abstract stat commands (already partially done)
- Test on both platforms

### 3. Performance Optimizations
- Cache jq results where possible
- Reduce subshell usage
- Optimize spawn-tree parsing

### 4. Code Organization
- Split aether-utils.sh into smaller modules
- Reduce code duplication (caste emoji functions)
- Centralize configuration

### 5. Security Improvements
- Add more path traversal protections
- Validate all inputs more strictly
- Add schema validation for all JSON files

### 6. Testing Coverage
- Add unit tests for all utility functions
- Add integration tests for command workflows
- Add platform compatibility tests

### 7. Documentation
- Add inline documentation for all functions
- Generate API documentation
- Add usage examples

---

## Security Analysis

### Path Traversal Protection
- `xml-compose.sh` has good path validation (lines 24-73)
- `checkpoint-check` has allowlist-based filtering
- Could be enhanced in more locations

### Input Validation
- Most commands validate inputs
- JSON validation uses jq where available
- Could add more strict validation

### Lock Safety
- File locks have timeout protection
- Stale lock detection exists
- BUG-005 needs fixing for complete safety

---

## Dependencies

### Required
- bash 3.2+
- jq (for JSON processing)

### Optional but Recommended
- xmllint (libxml2) - for XML processing
- xmlstarlet - for XPath queries
- xsltproc - for XSLT transformations
- git - for checkpoint/rollback

### For Visualization
- fswatch or inotifywait - for file watching

---

## Architecture Observations

### Strengths
1. Modular design with clear separation of concerns
2. Comprehensive error handling framework
3. Atomic operations for file safety
4. Feature flag system for graceful degradation
5. Extensive logging and activity tracking

### Weaknesses
1. Main file (aether-utils.sh) is very large (3,593 lines)
2. Some code duplication between utilities
3. Platform-specific code not always abstracted
4. Some functions too long (violate 50-line guideline)

### Recommendations
1. Split aether-utils.sh into thematic modules
2. Create shared library for common patterns
3. Add comprehensive test suite
4. Document all public APIs

---

*Analysis generated: 2026-02-16*
*Files analyzed: 15*
*Total lines: 8,298*
*Functions cataloged: ~190*
*Bugs identified: 6*
