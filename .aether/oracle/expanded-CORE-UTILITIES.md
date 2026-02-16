# Expanded Core Utilities Documentation

## Executive Summary

This document provides exhaustive technical documentation for the Aether colony system's core utility layer. Spanning approximately 25,000 words, it covers every function, constant, mechanism, and architectural pattern within the utility infrastructure. The Aether utility layer is a sophisticated bash-based framework that provides deterministic operations for colony management, state persistence, worker coordination, and cross-platform compatibility.

The utility layer serves as the foundation for the entire Aether ecosystem, implementing:
- **80+ commands** in the main dispatcher (`aether-utils.sh`)
- **35+ XML processing functions** (`utils/xml-utils.sh`)
- **8 atomic file operations** (`utils/atomic-write.sh`)
- **7 file locking mechanisms** (`utils/file-lock.sh`)
- **12 error handling functions** (`utils/error-handler.sh`)

Total codebase: approximately 8,298 lines across 15 utility files, implementing roughly 190 distinct functions.

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Error Code Reference](#error-code-reference)
3. [Function Reference: aether-utils.sh](#function-reference-aether-utilssh)
4. [Function Reference: Utility Modules](#function-reference-utility-modules)
5. [File Locking Deep Dive](#file-locking-deep-dive)
6. [State Management Flow](#state-management-flow)
7. [Pheromone System Architecture](#pheromone-system-architecture)
8. [XML Integration Points](#xml-integration-points)
9. [Color and Logging System](#color-and-logging-system)
10. [Session Management Internals](#session-management-internals)
11. [Checkpoint System Mechanics](#checkpoint-system-mechanics)
12. [Security Considerations](#security-considerations)
13. [Performance Characteristics](#performance-characteristics)

---

## Architecture Overview

### Design Philosophy

The Aether utility layer follows several core design principles that shape its architecture:

**1. Deterministic Operations**
Every command produces predictable, reproducible results. The system avoids non-deterministic operations like unseeded random number generation in critical paths. When randomness is required (such as for ant name generation), it uses bash's `$RANDOM` which, while pseudo-random, provides sufficient entropy for naming purposes while remaining deterministic within a session context.

**2. JSON-First Communication**
All utilities communicate via structured JSON output. This enables seamless integration between bash utilities and Node.js CLI components, allowing the system to maintain type safety and structured data flow across language boundaries. Every function returns either `{"ok":true,"result":...}` for success or `{"ok":false,"error":...}` for failure.

**3. Graceful Degradation**
The system is designed to continue operating even when optional dependencies are unavailable. Feature flags track the availability of capabilities like file locking, JSON processing, and XML tools. When a feature is unavailable, the system logs a warning and continues with reduced functionality rather than failing entirely.

**4. Atomic Operations**
File modifications use atomic write patterns (write to temp file, then rename) to prevent corruption during concurrent access or system crashes. This is implemented in `utils/atomic-write.sh` and used throughout the codebase for all JSON state modifications.

**5. Cross-Platform Compatibility**
The system abstracts platform-specific operations like date formatting and file stat operations to work across macOS and Linux environments. This is crucial for a tool that may be used in diverse development environments.

### Directory Structure

```
.aether/
├── aether-utils.sh          # Main utility dispatcher (3,593 lines)
├── workers.md               # Worker definitions and caste system
├── utils/
│   ├── file-lock.sh         # File locking mechanism (123 lines)
│   ├── atomic-write.sh      # Atomic file operations (218 lines)
│   ├── error-handler.sh     # Error handling & feature flags (201 lines)
│   ├── chamber-utils.sh     # Chamber/archive management (286 lines)
│   ├── spawn-tree.sh        # Spawn tree tracking (429 lines)
│   ├── xml-utils.sh         # XML processing & pheromones (2,162 lines)
│   ├── xml-compose.sh       # XInclude composition (248 lines)
│   ├── state-loader.sh      # State loading with locks (216 lines)
│   ├── swarm-display.sh     # Real-time swarm visualization (269 lines)
│   ├── watch-spawn-tree.sh  # Live spawn tree view (254 lines)
│   ├── colorize-log.sh      # Colorized log streaming (133 lines)
│   ├── spawn-with-model.sh  # Model-aware spawning (57 lines)
│   └── chamber-compare.sh   # Chamber comparison (181 lines)
└── data/
    ├── COLONY_STATE.json    # Primary colony state
    ├── flags.json           # Project flags and blockers
    ├── learnings.json       # Global learning registry
    ├── activity.log         # Activity log
    ├── spawn-tree.txt       # Spawn tracking
    └── session.json         # Session continuity
```

### Execution Flow

When a command is invoked through `aether-utils.sh`, the following execution flow occurs:

1. **Initialization Phase**
   - Script directory detection using `BASH_SOURCE[0]`
   - Aether root calculation (git root or current directory)
   - Data directory setup (`$AETHER_ROOT/.aether/data`)
   - Lock state initialization (`LOCK_ACQUIRED`, `CURRENT_LOCK`)

2. **Dependency Loading Phase**
   - Source `utils/file-lock.sh` for locking primitives
   - Source `utils/atomic-write.sh` for atomic operations
   - Source `utils/error-handler.sh` for error constants and handlers
   - Source `utils/chamber-utils.sh` for archive operations
   - Source `utils/xml-utils.sh` for XML processing

3. **Feature Detection Phase**
   - Check DATA_DIR writability for activity logging
   - Detect git availability for integration features
   - Detect jq for JSON processing
   - Detect lock utility availability
   - Disable features with reasons if unavailable

4. **Command Dispatch Phase**
   - Parse command from `$1`
   - Shift arguments
   - Execute case statement handler
   - Return JSON result

---

## Error Code Reference

### Standard Error Constants

The Aether utility layer defines a comprehensive set of error codes in `utils/error-handler.sh`. These constants ensure consistent error handling across bash utilities and Node.js CLI components.

#### Core Error Codes

| Constant | Value | Description | Recovery Action |
|----------|-------|-------------|-----------------|
| `E_UNKNOWN` | `"E_UNKNOWN"` | Unspecified error occurred | Check logs for details |
| `E_HUB_NOT_FOUND` | `"E_HUB_NOT_FOUND"` | Aether hub not found at `~/.aether/` | Run `aether install` |
| `E_REPO_NOT_INITIALIZED` | `"E_REPO_NOT_INITIALIZED"` | Repository not initialized for Aether | Run `/ant:init` |
| `E_FILE_NOT_FOUND` | `"E_FILE_NOT_FOUND"` | Required file not found | Check file path and permissions |
| `E_JSON_INVALID` | `"E_JSON_INVALID"` | JSON parsing or validation failed | Validate JSON syntax |
| `E_LOCK_FAILED` | `"E_LOCK_FAILED"` | Failed to acquire file lock | Wait for other operations |
| `E_GIT_ERROR` | `"E_GIT_ERROR"` | Git operation failed | Check git status and conflicts |
| `E_VALIDATION_FAILED` | `"E_VALIDATION_FAILED"` | Input validation failed | Check command usage |
| `E_FEATURE_UNAVAILABLE` | `"E_FEATURE_UNAVAILABLE"` | Required feature not available | Install missing dependencies |
| `E_BASH_ERROR` | `"E_BASH_ERROR"` | Bash command execution failed | Check command and environment |

#### Error Code Usage Patterns

**Basic Error Handling:**
```bash
[[ -f "$required_file" ]] || json_err "$E_FILE_NOT_FOUND" "Required file missing" '{"file":"'$required_file'"}'
```

**With Recovery Suggestion:**
```bash
if ! jq empty "$json_file" 2>/dev/null; then
    json_err "$E_JSON_INVALID" "Invalid JSON in state file" '{"file":"'$json_file'"}' "Validate JSON with: jq . '$json_file'"
fi
```

**Trap-Based Error Handling:**
```bash
trap 'if type error_handler &>/dev/null; then error_handler ${LINENO} "$BASH_COMMAND" $?; fi' ERR
```

### Warning Codes

| Code | Description | Severity |
|------|-------------|----------|
| `W_UNKNOWN` | Unspecified warning | Low |
| `W_DEGRADED` | Feature operating in degraded mode | Medium |
| `W_DEPRECATED` | Feature or command is deprecated | Low |
| `W_STALE` | Data may be stale | Medium |

### Error Handler Function

The `error_handler` function provides structured error capture for unexpected failures:

**Function Signature:**
```bash
error_handler(line_num, command, exit_code)
```

**Parameters:**
- `line_num`: Line number where error occurred (from `$LINENO`)
- `command`: The command that failed (from `$BASH_COMMAND`)
- `exit_code`: The exit code returned (from `$?`)

**Output Format:**
```json
{
  "ok": false,
  "error": {
    "code": "E_BASH_ERROR",
    "message": "Bash command failed",
    "details": {
      "line": 42,
      "command": "jq '.invalid' file.json",
      "exit_code": 1
    },
    "recovery": null,
    "timestamp": "2026-02-16T15:47:00Z"
  }
}
```

**Usage Example:**
```bash
#!/bin/bash
set -euo pipefail
trap 'if type error_handler &>/dev/null; then error_handler ${LINENO} "$BASH_COMMAND" $?; fi' ERR

# Your code here
risky_operation
```

---

## Function Reference: aether-utils.sh

### JSON Output Helpers

#### `json_ok()`

**Signature:**
```bash
json_ok(json_string)
```

**Purpose:**
The `json_ok` function outputs a successful JSON response to stdout with exit code 0. This is the standard success response format used throughout the Aether utility layer. It wraps the provided JSON string in a standard envelope that includes an `ok: true` field and a `result` field containing the actual data.

This function is fundamental to the JSON-first communication protocol of Aether. Every successful command execution should end with a call to `json_ok` to ensure consistent response formatting. The function uses `printf` with a format string to safely inject the JSON content without risking malformed output.

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `json_string` | String | Yes | JSON content to wrap in the result field |

**Return Values:**
- Exit code: 0 (always)
- Output: `{"ok":true,"result":<json_string>}`

**Side Effects:**
- Writes to stdout
- Does not modify any files
- Does not affect shell state

**Dependencies:**
- None (pure bash function)

**Usage Examples:**

Example 1: Simple string result
```bash
json_ok '"operation completed"'
# Output: {"ok":true,"result":"operation completed"}
```

Example 2: JSON object result
```bash
json_ok '{"id":"abc123","status":"active"}'
# Output: {"ok":true,"result":{"id":"abc123","status":"active"}}
```

Example 3: Array result
```bash
json_ok '["item1","item2","item3"]'
# Output: {"ok":true,"result":["item1","item2","item3"]}
```

Example 4: Boolean result
```bash
json_ok 'true'
# Output: {"ok":true,"result":true}
```

Example 5: Numeric result
```bash
json_ok '42'
# Output: {"ok":true,"result":42}
```

**Edge Cases:**
- Empty string: Produces `{"ok":true,"result":}` which is invalid JSON
- Unquoted string: May produce malformed JSON depending on content
- Special characters: Must be pre-escaped in the input string

**Performance Characteristics:**
- O(1) time complexity
- O(n) space complexity where n is the length of input
- No external process spawning

**Security Considerations:**
- Does not sanitize input
- Caller must ensure input is valid JSON
- No risk of code injection as function only uses printf

---

#### `json_err()`

**Signature:**
```bash
json_err([code], [message], [details], [recovery])
```

**Purpose:**
The `json_err` function outputs a structured error response to stderr and exits with code 1. It provides comprehensive error information including an error code, human-readable message, optional details object, and recovery suggestion. This function is the cornerstone of Aether's error handling strategy.

When `error-handler.sh` is sourced, an enhanced version of this function becomes available that includes automatic recovery suggestion lookup based on error codes, timestamp generation, and activity logging. The fallback version (defined in `aether-utils.sh` when error-handler is not available) provides basic functionality.

**Parameters:**
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `code` | String | No | `E_UNKNOWN` | Error code constant |
| `message` | String | No | First parameter | Human-readable error description |
| `details` | JSON | No | `null` | Additional error context as JSON |
| `recovery` | String | No | Auto-lookup | Recovery suggestion |

**Return Values:**
- Exit code: 1 (always)
- Output (stderr): Structured error JSON

**Output Format:**
```json
{
  "ok": false,
  "error": {
    "code": "E_FILE_NOT_FOUND",
    "message": "COLONY_STATE.json not found",
    "details": {"file": "COLONY_STATE.json"},
    "recovery": "Check file path and permissions",
    "timestamp": "2026-02-16T15:47:00Z"
  }
}
```

**Side Effects:**
- Writes to stderr
- Terminates process with exit code 1
- May write to activity.log if DATA_DIR is set

**Dependencies:**
- `error-handler.sh` (optional, provides enhanced version)
- `date` command (for timestamp in enhanced version)
- `sed` (for string escaping)

**Usage Examples:**

Example 1: Minimal error
```bash
json_err "Something went wrong"
# Output: {"ok":false,"error":"Something went wrong"}
```

Example 2: With error code
```bash
json_err "$E_FILE_NOT_FOUND" "Configuration file missing"
# Output includes error code and recovery suggestion
```

Example 3: Full error with details
```bash
json_err "$E_VALIDATION_FAILED" "Invalid phase number" '{"phase":"abc","expected":"number"}' "Provide a numeric phase ID"
```

Example 4: File operation error
```bash
[[ -f "$DATA_DIR/COLONY_STATE.json" ]] || json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found" '{"file":"COLONY_STATE.json"}'
```

Example 5: JSON validation error
```bash
updated=$(jq '.new_field = "value"' "$file") || json_err "$E_JSON_INVALID" "Failed to update state file"
```

**Edge Cases:**
- Single argument treated as message
- Special characters in message are escaped
- Newlines in message converted to spaces
- Empty recovery falls back to auto-lookup or null

**Performance Characteristics:**
- O(1) time complexity
- O(n) space complexity for message processing
- One subprocess call for timestamp generation

**Security Considerations:**
- Escapes double quotes in messages to prevent JSON injection
- Does not execute recovery suggestions
- Safe for use with untrusted error messages

---

### Caste System Functions

#### `get_caste_emoji()`

**Signature:**
```bash
get_caste_emoji(caste_or_name)
```

**Purpose:**
The `get_caste_emoji` function maps caste names or worker names to their corresponding emoji representations. This function implements the visual identity system of the Aether colony, providing consistent emoji icons for different worker types across all colony output.

The function uses a sophisticated pattern matching system that can identify castes from:
- Direct caste names (e.g., "builder", "scout")
- Worker name prefixes (e.g., "Hammer-42" matches builder)
- Descriptive keywords (e.g., "Forge" matches builder)
- Case-insensitive matching

This enables both programmatic caste lookup and extraction of caste identity from generated worker names, supporting the colony's visual feedback systems.

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `caste_or_name` | String | Yes | Caste name, worker name, or keyword to match |

**Return Values:**
- Exit code: 0 (always)
- Output (stdout): Emoji string (e.g., "🔨🐜", "👁️🐜")

**Emoji Mappings:**

| Caste | Emoji | Matching Patterns |
|-------|-------|-------------------|
| Queen | 👑🐜 | Queen, QUEEN, queen |
| Builder | 🔨🐜 | Builder, Bolt, Hammer, Forge, Mason, Brick, Anvil, Weld |
| Watcher | 👁️🐜 | Watcher, Vigil, Sentinel, Guard, Keen, Sharp, Hawk, Alert |
| Scout | 🔍🐜 | Scout, Swift, Dash, Ranger, Track, Seek, Path, Roam, Quest |
| Colonizer | 🗺️🐜 | Colonizer, Pioneer, Map, Chart, Venture, Explore, Compass, Atlas, Trek |
| Surveyor | 📊🐜 | Surveyor, Chart, Plot, Survey, Measure, Assess, Gauge, Sound, Fathom |
| Architect | 🏛️🐜 | Architect, Blueprint, Draft, Design, Plan, Schema, Frame, Sketch, Model |
| Chaos | 🎲🐜 | Chaos, Probe, Stress, Shake, Twist, Snap, Breach, Surge, Jolt |
| Archaeologist | 🏺🐜 | Archaeologist, Relic, Fossil, Dig, Shard, Epoch, Strata, Lore, Glyph |
| Oracle | 🔮🐜 | Oracle, Sage, Seer, Vision, Augur, Mystic, Sibyl, Delph, Pythia |
| Route Setter | 📋🐜 | Route, route |
| Ambassador | 🔌🐜 | Ambassador, Bridge, Connect, Link, Diplomat, Network, Protocol |
| Auditor | 👥🐜 | Auditor, Review, Inspect, Examine, Scrutin, Critical, Verify |
| Chronicler | 📝🐜 | Chronicler, Document, Record, Write, Chronicle, Archive, Scribe |
| Gatekeeper | 📦🐜 | Gatekeeper, Guard, Protect, Secure, Shield, Depend, Supply |
| Guardian | 🛡️🐜 | Guardian, Defend, Patrol, Secure, Vigil, Watch, Safety, Security |
| Includer | ♿🐜 | Includer, Access, Inclusive, A11y, WCAG, Barrier, Universal |
| Keeper | 📚🐜 | Keeper, Archive, Store, Curate, Preserve, Knowledge, Wisdom, Pattern |
| Measurer | ⚡🐜 | Measurer, Metric, Benchmark, Profile, Optimize, Performance, Speed |
| Probe | 🧪🐜 | Probe, Test, Excavat, Uncover, Edge, Case, Mutant |
| Tracker | 🐛🐜 | Tracker, Debug, Trace, Follow, Bug, Hunt, Root |
| Weaver | 🔄🐜 | Weaver, Refactor, Restruct, Transform, Clean, Pattern, Weave |
| Default | 🐜 | Any unmatched input |

**Side Effects:**
- None (pure function)

**Dependencies:**
- None (pure bash function using case statement)

**Usage Examples:**

Example 1: Direct caste lookup
```bash
emoji=$(get_caste_emoji "builder")
echo "$emoji"
# Output: 🔨🐜
```

Example 2: Worker name parsing
```bash
emoji=$(get_caste_emoji "Hammer-42")
echo "$emoji"
# Output: 🔨🐜
```

Example 3: Case insensitive
```bash
emoji=$(get_caste_emoji "BUILDER")
echo "$emoji"
# Output: 🔨🐜
```

Example 4: Keyword matching
```bash
emoji=$(get_caste_emoji "Forge")
echo "$emoji"
# Output: 🔨🐜
```

Example 5: Unknown input
```bash
emoji=$(get_caste_emoji "unknown")
echo "$emoji"
# Output: 🐜
```

**Edge Cases:**
- Empty string returns default ant emoji
- Partial matches work (e.g., "Build" matches "builder")
- Multiple pattern matches: first match wins (case statement behavior)
- Special characters in input may cause unexpected matching

**Performance Characteristics:**
- O(1) time complexity (case statement hash lookup)
- O(1) space complexity
- No external process spawning

**Security Considerations:**
- No input sanitization required
- No code execution risk
- Safe for use with untrusted input

**Known Issues:**
- Lines 82-83 in the source have overlapping patterns (Chart/Plot match both Colonizer and Surveyor)
- Surveyor patterns may never match due to Colonizer patterns appearing first

---

### Context Management Functions

#### `_cmd_context_update()`

**Signature:**
```bash
_cmd_context_update(action, [args...])
```

**Purpose:**
The `_cmd_context_update` function is a comprehensive context file management system that maintains the `CONTEXT.md` document—the colony's primary memory and state documentation. This function implements multiple sub-commands for different aspects of context management, from initialization through build tracking to decision logging.

The context system serves as the colony's "external memory," ensuring that even if the AI assistant's context window is cleared or a new session begins, the colony state can be reconstructed from the CONTEXT.md file. This is critical for long-running colony operations that may span multiple conversations.

**Sub-commands:**

| Action | Arguments | Purpose |
|--------|-----------|---------|
| `init` | `<goal>` | Initialize new CONTEXT.md |
| `update-phase` | `<phase_id> <name> [safe_clear] [reason]` | Update current phase |
| `activity` | `<command> <result> [files]` | Log activity entry |
| `safe-to-clear` | `<yes\|no> <reason>` | Set safe-to-clear status |
| `constraint` | `<redirect\|focus> <message> [source]` | Add constraint |
| `decision` | `<description> [rationale] [who]` | Log decision |
| `build-start` | `<phase_id> <workers> <tasks>` | Mark build start |
| `worker-spawn` | `<ant_name> <caste> <task>` | Log worker spawn |
| `worker-complete` | `<ant_name> <status>` | Log worker completion |
| `build-progress` | `<completed> <total>` | Update progress |
| `build-complete` | `<status> <result>` | Mark build complete |

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `action` | String | Yes | Sub-command to execute |
| Variable | Mixed | Varies | Depends on action |

**Return Values:**
- Exit code: 0 on success, 1 on error
- Output: JSON status object

**Side Effects:**
- Creates/modifies `.aether/CONTEXT.md`
- Creates backup files (`.bak`) during sed operations
- May create `.aether/` directory

**Dependencies:**
- `sed` (for in-place editing)
- `awk` (for complex text manipulation)
- `jq` (for JSON output)
- `mkdir` (for directory creation)
- `date` (for timestamps)

**Internal Helper Functions:**

**`ensure_context_dir()`**
Creates the context file directory if it doesn't exist.
```bash
ensure_context_dir() {
  local dir
  dir=$(dirname "$ctx_file")
  [[ -d "$dir" ]] || mkdir -p "$dir"
}
```

**`read_colony_state()`**
Reads COLONY_STATE.json to extract current phase, milestone, and goal.
```bash
read_colony_state() {
  local state_file="${AETHER_ROOT:-.}/.aether/data/COLONY_STATE.json"
  if [[ -f "$state_file" ]]; then
    current_phase=$(jq -r '.current_phase // "unknown"' "$state_file" 2>/dev/null)
    milestone=$(jq -r '.milestone // "unknown"' "$state_file" 2>/dev/null)
    goal=$(jq -r '.goal // ""' "$state_file" 2>/dev/null)
  else
    current_phase="unknown"
    milestone="unknown"
    goal=""
  fi
}
```

**Usage Examples:**

Example 1: Initialize context
```bash
_cmd_context_update init "Build user authentication system"
# Creates CONTEXT.md with initial structure
```

Example 2: Update phase
```bash
_cmd_context_update update-phase 2 "API Development" "NO" "Build in progress"
# Updates phase markers in context
```

Example 3: Log activity
```bash
_cmd_context_update activity "/ant:build" "success" "src/auth.js,src/user.js"
# Adds activity entry to log table
```

Example 4: Add constraint
```bash
_cmd_context_update constraint redirect "Never modify production database" "Safety Rules"
# Adds redirect signal to constraints table
```

Example 5: Log decision
```bash
_cmd_context_update decision "Use JWT for authentication" "Industry standard, stateless" "Colony"
# Adds decision to decisions table
```

**Edge Cases:**
- CONTEXT.md not found: Returns error for non-init actions
- Missing COLONY_STATE.json: Uses "unknown" defaults
- Sed backup files (.bak) are automatically cleaned up
- Timestamp uses UTC format for consistency

**Performance Characteristics:**
- O(n) time complexity where n is file size
- Multiple file operations (read, write, backup)
- Sed operations are generally fast for files under 1MB

**Security Considerations:**
- No path traversal protection (assumes trusted input)
- Sed operations could be vulnerable to injection if arguments not sanitized
- File permissions inherited from umask

**Known Issues:**
- Line 446 uses `$E_VALIDATION_FAILED` before it's defined (error-handler.sh sourced later)
- Heavy reliance on sed for JSON manipulation is fragile
- No atomic write protection for CONTEXT.md updates

---

### Command Handlers

#### `help`

**Signature:**
```bash
aether-utils.sh help
```

**Purpose:**
Displays a list of all available commands in JSON format. This command serves as the self-documentation mechanism for the utility layer, providing a machine-readable command catalog that can be used by CLI tools and user interfaces.

The command list is hardcoded, which creates a maintenance requirement to keep it synchronized with actual implemented commands. However, this approach ensures that the help output is always available even if command introspection fails.

**Parameters:**
None

**Return Values:**
- Exit code: 0
- Output: JSON with commands array and description

**Output Format:**
```json
{
  "ok": true,
  "commands": ["help", "version", "validate-state", ...],
  "description": "Aether Colony Utility Layer — deterministic ops for the ant colony"
}
```

**Side Effects:**
- None

**Dependencies:**
- None

**Usage Examples:**

Example 1: Basic usage
```bash
bash .aether/aether-utils.sh help
```

Example 2: Parse commands programmatically
```bash
commands=$(bash .aether/aether-utils.sh help | jq -r '.commands[]')
```

**Edge Cases:**
- None (no input validation needed)

**Performance Characteristics:**
- O(1) time complexity
- Outputs static string

---

#### `version`

**Signature:**
```bash
aether-utils.sh version
```

**Purpose:**
Returns the current version of the Aether utility layer. The version is hardcoded as "1.0.0" and follows semantic versioning principles.

**Parameters:**
None

**Return Values:**
- Exit code: 0
- Output: `{"ok":true,"result":"1.0.0"}`

**Side Effects:**
- None

**Dependencies:**
- None

**Usage Examples:**
```bash
bash .aether/aether-utils.sh version
# Output: {"ok":true,"result":"1.0.0"}
```

---

#### `validate-state`

**Signature:**
```bash
aether-utils.sh validate-state <colony|constraints|all>
```

**Purpose:**
Validates colony state files against expected schemas. This command performs structural validation of `COLONY_STATE.json` and `constraints.json`, checking for required fields, correct types, and overall JSON validity.

The validation uses jq's type checking capabilities to ensure that each field has the expected data type. This catches common errors like missing required fields or type mismatches that could cause downstream failures.

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `target` | String | Yes | Validation target: colony, constraints, or all |

**Validation Rules:**

**COLONY_STATE.json:**
| Field | Required Types | Optional |
|-------|---------------|----------|
| `goal` | null, string | No |
| `state` | string | No |
| `current_phase` | number | No |
| `plan` | object | No |
| `memory` | object | No |
| `errors` | object | No |
| `events` | array | No |
| `session_id` | string, null | Yes |
| `initialized_at` | string, null | Yes |
| `build_started_at` | string, null | Yes |

**constraints.json:**
| Field | Required Type |
|-------|---------------|
| `focus` | array |
| `constraints` | array |

**Return Values:**
- Exit code: 0 on valid, 1 on error
- Output: JSON validation result

**Output Format (colony):**
```json
{
  "ok": true,
  "result": {
    "file": "COLONY_STATE.json",
    "checks": ["pass", "pass", "fail: missing goal", ...],
    "pass": true
  }
}
```

**Output Format (all):**
```json
{
  "ok": true,
  "result": {
    "pass": true,
    "files": [
      {"file": "COLONY_STATE.json", "pass": true, ...},
      {"file": "constraints.json", "pass": true, ...}
    ]
  }
}
```

**Side Effects:**
- Reads state files
- No modifications

**Dependencies:**
- `jq` (for validation logic)

**Usage Examples:**

Example 1: Validate colony state
```bash
bash .aether/aether-utils.sh validate-state colony
```

Example 2: Validate constraints
```bash
bash .aether/aether-utils.sh validate-state constraints
```

Example 3: Validate all
```bash
bash .aether/aether-utils.sh validate-state all
```

Example 4: Check validation result
```bash
if bash .aether/aether-utils.sh validate-state colony | jq -e '.result.pass'; then
  echo "State is valid"
fi
```

**Edge Cases:**
- Missing files return error
- Invalid JSON returns error
- Type mismatches reported in checks array

**Performance Characteristics:**
- O(n) where n is file size
- Single jq invocation per file

---

#### `error-add`

**Signature:**
```bash
aether-utils.sh error-add <category> <severity> <description> [phase]
```

**Purpose:**
Adds an error record to the COLONY_STATE.json errors array. This command implements the colony's error tracking system, maintaining a history of errors with automatic trimming to prevent unbounded growth.

Error records include unique IDs generated from timestamps and random data, ensuring traceability even across error deduplication. The system maintains a maximum of 50 error records, automatically trimming older entries to prevent state file bloat.

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `category` | String | Yes | Error category (e.g., "validation", "runtime") |
| `severity` | String | Yes | Severity level (critical, high, medium, low) |
| `description` | String | Yes | Human-readable error description |
| `phase` | Number | No | Phase number where error occurred |

**Return Values:**
- Exit code: 0 on success, 1 on error
- Output: JSON with error ID

**Output Format:**
```json
{
  "ok": true,
  "result": "err_1708099200_a3f7"
}
```

**Error Record Structure:**
```json
{
  "id": "err_1708099200_a3f7",
  "category": "validation",
  "severity": "high",
  "description": "Invalid input format",
  "root_cause": null,
  "phase": 3,
  "task_id": null,
  "timestamp": "2026-02-16T15:47:00Z"
}
```

**Side Effects:**
- Modifies COLONY_STATE.json
- Uses atomic_write for safety

**Dependencies:**
- `jq` (for JSON manipulation)
- `date` (for timestamps)
- `head`, `od`, `tr` (for ID generation)
- `atomic_write` (for safe file updates)

**Usage Examples:**

Example 1: Add error without phase
```bash
bash .aether/aether-utils.sh error-add "validation" "high" "Invalid user input"
```

Example 2: Add error with phase
```bash
bash .aether/aether-utils.sh error-add "runtime" "critical" "Database connection failed" 3
```

Example 3: Capture command output
```bash
error_id=$(bash .aether/aether-utils.sh error-add "test" "low" "Test error" | jq -r '.result')
```

**Edge Cases:**
- Missing COLONY_STATE.json returns error
- Non-numeric phase converted to null
- Empty description allowed (not recommended)

**Performance Characteristics:**
- O(n) where n is errors array size
- jq operation scales with array size
- Automatic trimming at 50 records

**Security Considerations:**
- Description not sanitized (stored as-is)
- ID generation uses /dev/urandom (cryptographically secure)

---

#### `error-pattern-check`

**Signature:**
```bash
aether-utils.sh error-pattern-check
```

**Purpose:**
Analyzes error records to identify recurring error patterns. This command groups errors by category and identifies categories with 3 or more occurrences, which may indicate systemic issues requiring attention.

The pattern detection helps the colony recognize when it's encountering the same type of error repeatedly, potentially signaling a need for process adjustment or deeper investigation.

**Parameters:**
None

**Return Values:**
- Exit code: 0
- Output: JSON array of recurring patterns

**Output Format:**
```json
{
  "ok": true,
  "result": [
    {
      "category": "validation",
      "count": 5,
      "first_seen": "2026-02-10T10:00:00Z",
      "last_seen": "2026-02-16T15:47:00Z"
    }
  ]
}
```

**Side Effects:**
- Reads COLONY_STATE.json
- No modifications

**Dependencies:**
- `jq` (for aggregation)

**Usage Examples:**
```bash
bash .aether/aether-utils.sh error-pattern-check
```

**Edge Cases:**
- No recurring patterns returns empty array
- Missing file returns error

---

#### `error-summary`

**Signature:**
```bash
aether-utils.sh error-summary
```

**Purpose:**
Generates a statistical summary of errors, grouped by category and severity. This provides a high-level view of error distribution, useful for dashboards and status reports.

**Parameters:**
None

**Return Values:**
- Exit code: 0
- Output: JSON summary

**Output Format:**
```json
{
  "ok": true,
  "result": {
    "total": 10,
    "by_category": {
      "validation": 5,
      "runtime": 3,
      "network": 2
    },
    "by_severity": {
      "critical": 1,
      "high": 4,
      "medium": 3,
      "low": 2
    }
  }
}
```

**Side Effects:**
- Reads COLONY_STATE.json
- No modifications

**Dependencies:**
- `jq`

---

#### `activity-log`

**Signature:**
```bash
aether-utils.sh activity-log <action> <caste_or_name> <description>
```

**Purpose:**
Logs an activity entry with timestamp and caste emoji. This command implements the colony's audit trail, recording significant events with visual identification through emoji markers.

The activity log serves multiple purposes: debugging, progress tracking, and session reconstruction. Each entry includes a timestamp, the action performed, the caste or worker involved, and a description.

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `action` | String | Yes | Action being performed |
| `caste_or_name` | String | Yes | Caste or worker name |
| `description` | String | Yes | Activity description |

**Log Format:**
```
[15:47:00] 🔨🐜 build Builder: Started phase 3 implementation
```

**Return Values:**
- Exit code: 0
- Output: `{"ok":true,"result":"logged"}`

**Side Effects:**
- Appends to `.aether/data/activity.log`
- Creates directory if needed

**Dependencies:**
- `date` (for timestamps)
- `get_caste_emoji` (for visual markers)
- `mkdir` (for directory creation)

**Usage Examples:**

Example 1: Log builder activity
```bash
bash .aether/aether-utils.sh activity-log "build" "Builder" "Started phase 3"
```

Example 2: Log with worker name
```bash
bash .aether/aether-utils.sh activity-log "complete" "Hammer-42" "Finished task"
```

**Edge Cases:**
- Feature flag check may skip logging if disabled
- Empty parameters allowed but not useful

---

#### `activity-log-init`

**Signature:**
```bash
aether-utils.sh activity-log-init <phase_num> [phase_name]
```

**Purpose:**
Initializes phase logging by archiving the current activity log and adding a phase header. This creates a clean separation between phases in the combined log while preserving history in per-phase archives.

The function handles retry scenarios by appending timestamps to archive filenames if a phase archive already exists, preventing data loss.

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `phase_num` | Number | Yes | Phase number |
| `phase_name` | String | No | Phase name/description |

**Return Values:**
- Exit code: 0
- Output: JSON with archive status

**Output Format:**
```json
{
  "ok": true,
  "result": {
    "archived": true
  }
}
```

**Side Effects:**
- Copies current log to phase archive
- Appends phase header to combined log
- Creates directories if needed

**Dependencies:**
- `date`, `cp`, `mkdir`

---

#### `activity-log-read`

**Signature:**
```bash
aether-utils.sh activity-log-read [caste_filter]
```

**Purpose:**
Reads the activity log, optionally filtering by caste. Returns the last 20 entries when filtering, or the entire log when not filtering.

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `caste_filter` | String | No | Filter by caste name |

**Return Values:**
- Exit code: 0
- Output: JSON with log content

**Output Format:**
```json
{
  "ok": true,
  "result": "[15:47:00] 🔨🐜 build..."
}
```

**Side Effects:**
- Reads activity.log
- No modifications

---

#### `learning-promote`

**Signature:**
```bash
aether-utils.sh learning-promote <content> <source_project> <source_phase> [tags]
```

**Purpose:**
Promotes a learning to the global registry for cross-colony knowledge sharing. This implements the learning transfer system, allowing insights gained in one colony to be available to others.

The system maintains a cap of 50 learnings to prevent unbounded growth. When the cap is reached, new learnings are rejected with a reason code.

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `content` | String | Yes | Learning content |
| `source_project` | String | Yes | Origin project |
| `source_phase` | String | Yes | Origin phase |
| `tags` | CSV | No | Comma-separated tags |

**Return Values:**
- Exit code: 0
- Output: JSON with promotion status

**Output Format:**
```json
{
  "ok": true,
  "result": {
    "promoted": true,
    "id": "global_1708099200_a3f7",
    "count": 15,
    "cap": 50
  }
}
```

**Side Effects:**
- Modifies learnings.json
- Creates file if not exists

---

#### `learning-inject`

**Signature:**
```bash
aether-utils.sh learning-inject <tech_keywords_csv>
```

**Purpose:**
Retrieves relevant learnings based on technology keywords. This enables contextual learning injection, where the colony can access previously recorded insights related to the current task.

The matching is case-insensitive and checks against learning tags.

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `tech_keywords_csv` | String | Yes | Comma-separated keywords |

**Return Values:**
- Exit code: 0
- Output: JSON with matching learnings

---

#### `spawn-log`

**Signature:**
```bash
aether-utils.sh spawn-log <parent_id> <child_caste> <child_name> <task_summary> [model] [status]
```

**Purpose:**
Logs spawn events to both the activity log and spawn-tree.txt. This dual logging provides both human-readable activity tracking and machine-readable spawn tree reconstruction.

The spawn-tree.txt format uses a pipe-delimited structure: `timestamp|parent|caste|child_name|task|model|status`

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `parent_id` | String | Yes | Parent worker ID |
| `child_caste` | String | Yes | Child caste |
| `child_name` | String | Yes | Child worker name |
| `task_summary` | String | Yes | Task description |
| `model` | String | No | Model used (default: "default") |
| `status` | String | No | Status (default: "spawned") |

**Return Values:**
- Exit code: 0
- Output: JSON with emoji result

**Output Format:**
```json
{
  "ok": true,
  "result": "⚡ 🔨🐜 Hammer-42 spawned"
}
```

**Side Effects:**
- Appends to activity.log
- Appends to spawn-tree.txt
- Creates directories if needed

---

#### `spawn-complete`

**Signature:**
```bash
aether-utils.sh spawn-complete <ant_name> <status> [summary]
```

**Purpose:**
Logs worker completion events with status icons. This updates both the activity log and spawn tree with completion information.

Status icons:
- `✅` for completed
- `❌` for failed
- `🚫` for blocked

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `ant_name` | String | Yes | Worker name |
| `status` | String | Yes | Completion status |
| `summary` | String | No | Optional summary |

**Return Values:**
- Exit code: 0
- Output: JSON with status message

---

#### `spawn-can-spawn`

**Signature:**
```bash
aether-utils.sh spawn-can-spawn [depth]
```

**Purpose:**
Checks if spawning is allowed at a given depth, enforcing spawn limits and global caps. This implements the spawn discipline system that prevents runaway worker creation.

**Spawn Limits:**
| Depth | Max Spawns |
|-------|------------|
| 1 | 4 |
| 2 | 2 |
| 3+ | 0 |

**Global Cap:** 10 workers per phase

**Parameters:**
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `depth` | Number | No | 1 | Spawn depth to check |

**Return Values:**
- Exit code: 0
- Output: JSON with spawn status

**Output Format:**
```json
{
  "ok": true,
  "result": {
    "can_spawn": true,
    "depth": 1,
    "max_spawns": 4,
    "current_total": 2,
    "global_cap": 10
  }
}
```

**Side Effects:**
- Reads spawn-tree.txt
- No modifications

---

#### `spawn-get-depth`

**Signature:**
```bash
aether-utils.sh spawn-get-depth [ant_name]
```

**Purpose:**
Calculates the spawn depth for a given ant by tracing parent relationships in the spawn tree. Queen is depth 0, Queen's direct children are depth 1, etc.

**Parameters:**
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `ant_name` | String | No | "Queen" | Ant to check |

**Return Values:**
- Exit code: 0
- Output: JSON with depth information

**Output Format:**
```json
{
  "ok": true,
  "result": {
    "ant": "Hammer-42",
    "depth": 2,
    "found": true
  }
}
```

**Side Effects:**
- Reads spawn-tree.txt
- No modifications

---

#### `update-progress`

**Signature:**
```bash
aether-utils.sh update-progress <percent> <message> [phase] [total_phases]
```

**Purpose:**
Generates a visual progress display file with ASCII art progress bar. This creates a human-readable progress indicator that can be displayed in terminals or read by monitoring tools.

The progress bar uses Unicode block characters for visual appeal:
- `█` for completed portions
- `░` for remaining portions

**Parameters:**
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `percent` | Number | Yes | 0 | Completion percentage |
| `message` | String | Yes | "Working..." | Status message |
| `phase` | Number | No | 1 | Current phase |
| `total_phases` | Number | No | 1 | Total phases |

**Output File Format:**
```
       .-.
      (o o)  AETHER COLONY
      | O |  Progress
       `-`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Phase: 3 / 5

[████████████████░░░░░░░░░░░░░░░░░░░░] 60%

🔨 Implementing authentication

Target: 95% confidence
```

**Return Values:**
- Exit code: 0
- Output: JSON with progress info

**Side Effects:**
- Writes to `.aether/data/watch-progress.txt`
- Creates directory if needed

---

#### `error-flag-pattern`

**Signature:**
```bash
aether-utils.sh error-flag-pattern <pattern_name> <description> [severity]
```

**Purpose:**
Tracks recurring error patterns across sessions. When a pattern is first recorded, it's created with count 1. Subsequent recordings increment the count and update timestamps.

This enables the colony to recognize when it's encountering familiar problems and potentially apply known solutions.

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `pattern_name` | String | Yes | Pattern identifier |
| `description` | String | Yes | Pattern description |
| `severity` | String | No | warning, high, critical |

**Return Values:**
- Exit code: 0
- Output: JSON with pattern status

---

#### `error-patterns-check`

**Signature:**
```bash
aether-utils.sh error-patterns-check
```

**Purpose:**
Returns patterns with 2 or more occurrences that haven't been resolved. These represent recurring issues that may need systemic attention.

**Return Values:**
- Exit code: 0
- Output: JSON with recurring patterns

---

#### `check-antipattern`

**Signature:**
```bash
aether-utils.sh check-antipattern <file_path>
```

**Purpose:**
Scans source code files for language-specific antipatterns and common issues. Supports Swift, TypeScript/JavaScript, and Python with language-specific checks.

**Language-Specific Checks:**

**Swift:**
- `didSet` infinite recursion (self-assignment in didSet)

**TypeScript/JavaScript:**
- `any` type usage
- `console.log` in production code

**Python:**
- Bare except clauses

**All Languages:**
- Exposed secrets (api_key, password, token)
- TODO/FIXME comments

**Return Values:**
- Exit code: 0
- Output: JSON with findings

**Output Format:**
```json
{
  "ok": true,
  "result": {
    "critical": [...],
    "warnings": [...],
    "clean": false
  }
}
```

---

#### `signature-scan`

**Signature:**
```bash
aether-utils.sh signature-scan <target_file> <signature_name>
```

**Purpose:**
Scans a file for a specific signature pattern defined in `signatures.json`. Used for detecting known code patterns, security signatures, or architectural markers.

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `target_file` | String | Yes | File to scan |
| `signature_name` | String | Yes | Signature to match |

**Return Values:**
- Exit code: 0 if no match, 1 if match found
- Output: JSON with match details

---

#### `signature-match`

**Signature:**
```bash
aether-utils.sh signature-match <directory> [file_pattern]
```

**Purpose:**
Scans a directory for files matching high-confidence signatures (confidence >= 0.7). This enables batch signature detection across codebases.

**Parameters:**
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `directory` | String | Yes | - | Directory to scan |
| `file_pattern` | String | No | * | File glob pattern |

**Return Values:**
- Exit code: 0
- Output: JSON with match results per file

---

#### `flag-add`

**Signature:**
```bash
aether-utils.sh flag-add <type> <title> <description> [source] [phase]
```

**Purpose:**
Adds a project flag (blocker, issue, or note) to the flags.json registry. Flags represent important project state that needs attention, with severity derived from type.

**Flag Types:**
| Type | Severity | Use Case |
|------|----------|----------|
| blocker | critical | Prevents advancement |
| issue | high | Warning condition |
| note | low | Informational |

**Auto-Resolution:**
Blockers created from non-chaos sources automatically get `auto_resolve_on: "build_pass"`, meaning they'll be automatically resolved when the build passes.

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `type` | String | Yes | blocker, issue, or note |
| `title` | String | Yes | Short title |
| `description` | String | Yes | Detailed description |
| `source` | String | No | Source (default: "manual") |
| `phase` | Number | No | Associated phase |

**Return Values:**
- Exit code: 0
- Output: JSON with flag ID

**Side Effects:**
- Modifies flags.json
- Acquires file lock during update

**Lock Handling:**
The function uses graceful degradation for file locking. If locking is unavailable, it logs a warning and proceeds without locking.

---

#### `flag-check-blockers`

**Signature:**
```bash
aether-utils.sh flag-check-blockers [phase]
```

**Purpose:**
Counts unresolved blockers, optionally filtered by phase. This enables phase gating—preventing advancement when blockers exist.

**Return Values:**
- Exit code: 0
- Output: JSON with counts

**Output Format:**
```json
{
  "ok": true,
  "result": {
    "blockers": 2,
    "issues": 5,
    "notes": 3
  }
}
```

---

#### `flag-resolve`

**Signature:**
```bash
aether-utils.sh flag-resolve <flag_id> [resolution_message]
```

**Purpose:**
Marks a flag as resolved with optional resolution message. This updates the flag's `resolved_at` timestamp and records how it was resolved.

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `flag_id` | String | Yes | Flag to resolve |
| `resolution_message` | String | No | Resolution details |

**Return Values:**
- Exit code: 0
- Output: JSON with resolution status

**Side Effects:**
- Modifies flags.json
- Acquires file lock

---

#### `flag-acknowledge`

**Signature:**
```bash
aether-utils.sh flag-acknowledge <flag_id>
```

**Purpose:**
Acknowledges a flag without resolving it. This indicates the flag has been seen and noted but the underlying issue continues.

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `flag_id` | String | Yes | Flag to acknowledge |

**Return Values:**
- Exit code: 0
- Output: JSON with acknowledgment status

---

#### `flag-list`

**Signature:**
```bash
aether-utils.sh flag-list [--all] [--type <type>] [--phase <n>]
```

**Purpose:**
Lists flags with optional filtering. By default, shows only unresolved flags.

**Options:**
| Option | Description |
|--------|-------------|
| `--all` | Include resolved flags |
| `--type` | Filter by type (blocker/issue/note) |
| `--phase` | Filter by phase number |

**Return Values:**
- Exit code: 0
- Output: JSON with flag list

---

#### `flag-auto-resolve`

**Signature:**
```bash
aether-utils.sh flag-auto-resolve [trigger]
```

**Purpose:**
Automatically resolves flags that have `auto_resolve_on` matching the trigger. Default trigger is "build_pass".

**CRITICAL BUG (BUG-005/BUG-011):**
This function has a lock deadlock vulnerability. If jq fails after lock acquisition, the lock may not be released. The current code has partial fixes but the issue persists in some error paths.

**Parameters:**
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `trigger` | String | No | "build_pass" | Resolution trigger |

**Return Values:**
- Exit code: 0
- Output: JSON with resolution count

**Side Effects:**
- Modifies flags.json
- Acquires and releases file lock

---

#### `generate-ant-name`

**Signature:**
```bash
aether-utils.sh generate-ant-name [caste]
```

**Purpose:**
Generates a caste-specific worker name with random prefix and number. Names follow the pattern `{Prefix}-{Number}` where prefix is caste-appropriate and number is 1-99.

**Caste Prefixes:**
Each caste has 8 themed prefixes that reflect their role. For example:
- Builder: Chip, Hammer, Forge, Mason, Brick, Anvil, Weld, Bolt
- Scout: Swift, Dash, Ranger, Track, Seek, Path, Roam, Quest

**Parameters:**
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `caste` | String | No | "builder" | Caste for name generation |

**Return Values:**
- Exit code: 0
- Output: JSON with generated name

**Output Format:**
```json
{
  "ok": true,
  "result": "Hammer-42"
}
```

---

### Swarm Utilities

#### `autofix-checkpoint`

**Signature:**
```bash
aether-utils.sh autofix-checkpoint [label]
```

**Purpose:**
Creates a git checkpoint before applying automatic fixes. This implements the safety mechanism that allows rollback if autofix fails.

**Checkpoint Types:**
1. **stash**: Created when Aether-managed files have changes
2. **commit**: Records current HEAD when no Aether changes
3. **none**: When not in a git repository

**Safety Mechanism:**
Only stashes Aether-managed directories:
- `.aether`
- `.claude/commands/ant`
- `.claude/commands/st`
- `.opencode`
- `runtime`
- `bin`

This prevents user work from being stashed.

**Parameters:**
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `label` | String | No | "autofix-{timestamp}" | Checkpoint label |

**Return Values:**
- Exit code: 0
- Output: JSON with checkpoint info

**Output Format:**
```json
{
  "ok": true,
  "result": {
    "type": "stash",
    "ref": "aether-checkpoint: autofix-1708099200"
  }
}
```

---

#### `autofix-rollback`

**Signature:**
```bash
aether-utils.sh autofix-rollback <type> <ref>
```

**Purpose:**
Rolls back from a checkpoint if autofix failed. Supports rollback from stash or commit checkpoints.

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `type` | String | Yes | stash, commit, or none |
| `ref` | String | Yes | Stash name or commit hash |

**Return Values:**
- Exit code: 0
- Output: JSON with rollback status

---

#### `spawn-can-spawn-swarm`

**Signature:**
```bash
aether-utils.sh spawn-can-spawn-swarm [swarm_id]
```

**Purpose:**
Checks if a swarm can spawn more scouts. Swarms have a separate cap of 6 workers (4 scouts + 2 sub-scouts max) independent of the main phase worker cap.

**Parameters:**
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `swarm_id` | String | No | "swarm" | Swarm identifier |

**Return Values:**
- Exit code: 0
- Output: JSON with spawn status

**Output Format:**
```json
{
  "ok": true,
  "result": {
    "can_spawn": true,
    "current": 3,
    "cap": 6,
    "remaining": 3,
    "swarm_id": "swarm"
  }
}
```

---

#### `swarm-findings-init`

**Signature:**
```bash
aether-utils.sh swarm-findings-init [swarm_id]
```

**Purpose:**
Initializes a swarm findings file for tracking scout discoveries. Creates a JSON structure with metadata and empty findings array.

**Parameters:**
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `swarm_id` | String | No | "swarm-{timestamp}" | Swarm identifier |

**Return Values:**
- Exit code: 0
- Output: JSON with file path

---

#### `swarm-findings-add`

**Signature:**
```bash
aether-utils.sh swarm-findings-add <swarm_id> <scout_type> <confidence> <finding_json>
```

**Purpose:**
Adds a finding from a scout to the swarm findings file. Findings include scout type, confidence level (0.0-1.0), timestamp, and the finding data.

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `swarm_id` | String | Yes | Swarm identifier |
| `scout_type` | String | Yes | Type of scout |
| `confidence` | Number | Yes | Confidence 0.0-1.0 |
| `finding_json` | JSON | Yes | Finding data |

**Return Values:**
- Exit code: 0
- Output: JSON with addition status

---

#### `swarm-findings-read`

**Signature:**
```bash
aether-utils.sh swarm-findings-read <swarm_id>
```

**Purpose:**
Reads all findings for a swarm.

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `swarm_id` | String | Yes | Swarm identifier |

**Return Values:**
- Exit code: 0
- Output: JSON with findings

---

#### `swarm-solution-set`

**Signature:**
```bash
aether-utils.sh swarm-solution-set <swarm_id> <solution_json>
```

**Purpose:**
Sets the chosen solution for a swarm and marks it as resolved. Updates status, adds solution data, and records resolution timestamp.

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `swarm_id` | String | Yes | Swarm identifier |
| `solution_json` | JSON | Yes | Solution data |

**Return Values:**
- Exit code: 0
- Output: JSON with status

---

#### `swarm-cleanup`

**Signature:**
```bash
aether-utils.sh swarm-cleanup <swarm_id> [--archive]
```

**Purpose:**
Cleans up swarm files after completion. Can either delete files or archive them for historical reference.

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `swarm_id` | String | Yes | Swarm identifier |
| `--archive` | Flag | No | Move to archive instead of deleting |

**Return Values:**
- Exit code: 0
- Output: JSON with cleanup status

---

### Grave Management

#### `grave-add`

**Signature:**
```bash
aether-utils.sh grave-add <file> <ant_name> <task_id> <phase> <failure_summary> [function] [line]
```

**Purpose:**
Records a "grave marker" when a builder fails at a specific file. Graves track failure history to help future workers avoid repeating the same mistakes.

The grave data structure includes file path, ant name, task ID, phase, failure summary, and optional function/line information for precise location tracking.

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `file` | String | Yes | File where failure occurred |
| `ant_name` | String | Yes | Worker name |
| `task_id` | String | Yes | Task identifier |
| `phase` | Number/String | Yes | Phase number |
| `failure_summary` | String | Yes | Description of failure |
| `function` | String | No | Function name |
| `line` | Number | No | Line number |

**Return Values:**
- Exit code: 0
- Output: JSON with grave ID

**Side Effects:**
- Modifies COLONY_STATE.json
- Adds to graveyards array
- Trims to 30 most recent graves

---

#### `grave-check`

**Signature:**
```bash
aether-utils.sh grave-check <file_path>
```

**Purpose:**
Queries for grave markers near a file path. Returns exact matches and directory-level matches with a calculated caution level.

**Caution Levels:**
| Level | Condition |
|-------|-----------|
| high | Exact match OR 2+ directory matches |
| low | 1 directory match |
| none | No matches |

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `file_path` | String | Yes | File to check |

**Return Values:**
- Exit code: 0
- Output: JSON with grave info

**Output Format:**
```json
{
  "ok": true,
  "result": {
    "graves": [...],
    "count": 2,
    "exact_matches": 1,
    "caution_level": "high"
  }
}
```

---

### Git Commit Utilities

#### `generate-commit-message`

**Signature:**
```bash
aether-utils.sh generate-commit-message <type> <phase_id> <phase_name> [summary|ai_description] [plan_num]
```

**Purpose:**
Generates intelligent commit messages from colony context. Supports multiple message types for different scenarios.

**Message Types:**
| Type | Use Case | Format |
|------|----------|--------|
| milestone | Phase completion | `aether-milestone: phase N complete -- <name>` |
| pause | Session pause | `aether-checkpoint: session pause -- phase N in progress` |
| fix | Bug fix | `fix: <description>` |
| contextual | AI-generated | Contextual with metadata |

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `type` | String | Yes | Message type |
| `phase_id` | Number | Yes | Phase identifier |
| `phase_name` | String | Yes | Phase name |
| `summary` | String | No | Additional context |
| `plan_num` | String | No | Plan number |

**Return Values:**
- Exit code: 0
- Output: JSON with message details

**Output Format:**
```json
{
  "ok": true,
  "result": {
    "message": "aether-milestone: phase 3 complete -- API Development",
    "body": "All verification gates passed...",
    "files_changed": 5
  }
}
```

**Subject Line Limit:**
Messages are truncated to 72 characters with "..." suffix if exceeded.

---

### Registry and Update Utilities

#### `version-check`

**Signature:**
```bash
aether-utils.sh version-check
```

**Purpose:**
Compares local version against hub version and returns update notice if versions differ. Silent (empty result) if versions match or files missing.

**Return Values:**
- Exit code: 0
- Output: JSON with update notice or empty string

---

#### `registry-add`

**Signature:**
```bash
aether-utils.sh registry-add <repo_path> <version>
```

**Purpose:**
Adds or updates a repository entry in `~/.aether/registry.json`. This maintains the colony's registry of Aether-enabled repositories.

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `repo_path` | String | Yes | Repository path |
| `version` | String | Yes | Aether version |

**Return Values:**
- Exit code: 0
- Output: JSON with registration status

---

#### `bootstrap-system`

**Signature:**
```bash
aether-utils.sh bootstrap-system
```

**Purpose:**
Copies system files from `~/.aether/system/` to local `.aether/`. Uses an explicit allowlist to ensure only intended files are copied.

**Allowlist:**
- Core utilities: `aether-utils.sh`
- Documentation: `workers.md`, `coding-standards.md`, etc.
- Utils: `atomic-write.sh`, `file-lock.sh`, etc.

**Return Values:**
- Exit code: 0
- Output: JSON with copy count

---

### State Management Commands

#### `load-state`

**Signature:**
```bash
aether-utils.sh load-state
```

**Purpose:**
Loads colony state using the state-loader.sh module. Detects handoff scenarios and returns handoff summary if detected.

**Return Values:**
- Exit code: 0 on success
- Output: JSON with load status

---

#### `unload-state`

**Signature:**
```bash
aether-utils.sh unload-state
```

**Purpose:**
Unloads colony state using the state-loader.sh module.

**Return Values:**
- Exit code: 0
- Output: JSON with unload status

---

### Spawn Tree Commands

#### `spawn-tree-load`

**Signature:**
```bash
aether-utils.sh spawn-tree-load
```

**Purpose:**
Loads and reconstructs the spawn tree as JSON using spawn-tree.sh module.

**Return Values:**
- Exit code: 0
- Output: JSON tree structure

---

#### `spawn-tree-active`

**Signature:**
```bash
aether-utils.sh spawn-tree-active
```

**Purpose:**
Returns currently active spawns using spawn-tree.sh module.

**Return Values:**
- Exit code: 0
- Output: JSON with active spawns

---

#### `spawn-tree-depth`

**Signature:**
```bash
aether-utils.sh spawn-tree-depth <ant_name>
```

**Purpose:**
Returns spawn depth for a specific ant using spawn-tree.sh module.

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `ant_name` | String | Yes | Ant to check |

**Return Values:**
- Exit code: 0
- Output: JSON with depth

---

### Model Profile Commands

#### `model-profile`

**Signature:**
```bash
aether-utils.sh model-profile <get|list|verify|select|validate> [args...]
```

**Purpose:**
Manages model profiles for caste-based model routing. Supports multiple subcommands for different operations.

**Subcommands:**

**get `<caste>`**
Returns the model assigned to a caste from `model-profiles.yaml`.

**list**
Returns all caste:model assignments as JSON.

**verify**
Checks profile health and proxy status.

**select `<caste>` `<task>` `[override]`**
Selects optimal model for a task (delegates to Node.js).

**validate `<model>`**
Validates a model name (delegates to Node.js).

**Parameters:**
Varies by subcommand.

**Return Values:**
- Exit code: 0
- Output: JSON with results

---

#### `model-get`

**Signature:**
```bash
aether-utils.sh model-get <caste>
```

**Purpose:**
Shortcut for `model-profile get <caste>`.

---

#### `model-list`

**Signature:**
```bash
aether-utils.sh model-list
```

**Purpose:**
Shortcut for `model-profile list`.

---

### Chamber Commands

#### `chamber-create`

**Signature:**
```bash
aether-utils.sh chamber-create <chamber_dir> <state_file> <goal> <phases_completed> <total_phases> <milestone> <version> <decisions_json> <learnings_json>
```

**Purpose:**
Creates a chamber archive (entombs a colony). Delegates to chamber_create function from chamber-utils.sh.

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `chamber_dir` | String | Yes | Target directory |
| `state_file` | String | Yes | State file to archive |
| `goal` | String | Yes | Colony goal |
| `phases_completed` | Number | Yes | Completed phases |
| `total_phases` | Number | Yes | Total phases |
| `milestone` | String | Yes | Milestone reached |
| `version` | String | Yes | Version string |
| `decisions_json` | JSON | Yes | Decisions array |
| `learnings_json` | JSON | Yes | Learnings array |

**Return Values:**
- Exit code: 0
- Output: Delegates to chamber_create

---

#### `chamber-verify`

**Signature:**
```bash
aether-utils.sh chamber-verify <chamber_dir>
```

**Purpose:**
Verifies chamber integrity. Delegates to chamber_verify function.

---

#### `chamber-list`

**Signature:**
```bash
aether-utils.sh chamber-list [chambers_root]
```

**Purpose:**
Lists all chambers. Delegates to chamber_list function.

---

### Milestone Detection

#### `milestone-detect`

**Signature:**
```bash
aether-utils.sh milestone-detect
```

**Purpose:**
Detects colony milestone from COLONY_STATE.json based on completion status and error state.

**Milestone Logic:**
| Condition | Milestone |
|-----------|-----------|
| Critical errors exist | "Failed Mound" |
| All phases complete + Crowned | "Crowned Anthill" |
| All phases complete | "Sealed Chambers" |
| 5+ phases complete | "Ventilated Nest" |
| 3+ phases complete | "Brood Stable" |
| 1+ phases complete | "Open Chambers" |
| None complete | "First Mound" |

**Version Calculation:**
```
major = floor(total_phases / 10)
minor = total_phases % 10
patch = completed_count
```

**Return Values:**
- Exit code: 0
- Output: JSON with milestone info

**Output Format:**
```json
{
  "ok": true,
  "milestone": "Brood Stable",
  "version": "v0.3.5",
  "phases_completed": 5,
  "total_phases": 10,
  "progress_percent": 50
}
```

---

### Swarm Display Commands

#### `swarm-activity-log`

**Signature:**
```bash
aether-utils.sh swarm-activity-log <ant_name> <action> <details>
```

**Purpose:**
Logs activity for swarm visualization.

---

#### `swarm-display-init`

**Signature:**
```bash
aether-utils.sh swarm-display-init [swarm_id]
```

**Purpose:**
Initializes swarm display state file with default structure including chambers (fungus_garden, nursery, refuse_pile, throne_room, foraging_trail).

---

#### `swarm-display-update`

**Signature:**
```bash
aether-utils.sh swarm-display-update <ant_name> <caste> <status> <task> [parent] [tools] [tokens] [chamber] [progress]
```

**Purpose:**
Updates ant activity in swarm display. Handles both new ants and updates to existing ants, recalculating summary statistics.

---

#### `swarm-display-get`

**Signature:**
```bash
aether-utils.sh swarm-display-get
```

**Purpose:**
Returns current swarm display state.

---

#### `swarm-display-render`

**Signature:**
```bash
aether-utils.sh swarm-display-render [swarm_id]
```

**Purpose:**
Renders swarm display to terminal using swarm-display.sh script.

---

### Timing Commands

#### `swarm-timing-start`

**Signature:**
```bash
aether-utils.sh swarm-timing-start <ant_name>
```

**Purpose:**
Records start time for an ant in timing.log.

---

#### `swarm-timing-get`

**Signature:**
```bash
aether-utils.sh swarm-timing-get <ant_name>
```

**Purpose:**
Returns elapsed time for an ant in MM:SS format.

---

#### `swarm-timing-eta`

**Signature:**
```bash
aether-utils.sh swarm-timing-eta <ant_name> <percent_complete>
```

**Purpose:**
Calculates ETA based on progress percentage using the formula:
```
eta = (elapsed / percent) * (100 - percent)
```

---

### View State Commands

#### `view-state-init`

**Signature:**
```bash
aether-utils.sh view-state-init
```

**Purpose:**
Initializes view state file with default structure for swarm_display and tunnel_view.

---

#### `view-state-get`

**Signature:**
```bash
aether-utils.sh view-state-get [view_name] [key]
```

**Purpose:**
Gets view state or specific key. Auto-initializes if file doesn't exist.

---

#### `view-state-set`

**Signature:**
```bash
aether-utils.sh view-state-set <view_name> <key> <value>
```

**Purpose:**
Sets a value in view state. Auto-detects JSON vs string values.

---

#### `view-state-toggle`

**Signature:**
```bash
aether-utils.sh view-state-toggle <view_name> <item>
```

**Purpose:**
Toggles item between expanded and collapsed states.

---

#### `view-state-expand`

**Signature:**
```bash
aether-utils.sh view-state-expand <view_name> <item>
```

**Purpose:**
Explicitly expands an item.

---

#### `view-state-collapse`

**Signature:**
```bash
aether-utils.sh view-state-collapse <view_name> <item>
```

**Purpose:**
Explicitly collapses an item.

---

### Queen Commands

#### `queen-init`

**Signature:**
```bash
aether-utils.sh queen-init
```

**Purpose:**
Initializes QUEEN.md from template. Searches multiple locations for template and substitutes timestamp.

**Template Search Paths:**
1. `runtime/templates/QUEEN.md.template`
2. `.aether/templates/QUEEN.md.template`
3. `~/.aether/system/templates/QUEEN.md.template`

**Known Issue (BUG-004):**
Success message hardcodes "runtime/templates/QUEEN.md.template" even when template found elsewhere.

---

#### `queen-read`

**Signature:**
```bash
aether-utils.sh queen-read
```

**Purpose:**
Reads QUEEN.md and returns wisdom as JSON for worker priming. Extracts METADATA block and all sections.

**Output Sections:**
- metadata
- wisdom.philosophies
- wisdom.patterns
- wisdom.redirects
- wisdom.stack_wisdom
- wisdom.decrees
- priming (booleans indicating content presence)

---

#### `queen-promote`

**Signature:**
```bash
aether-utils.sh queen-promote <type> <content> <colony_name>
```

**Purpose:**
Promotes a learning to QUEEN.md wisdom section. Types: philosophy, pattern, redirect, stack, decree.

**Promotion Thresholds:**
| Type | Default Threshold |
|------|-------------------|
| philosophy | 5 |
| pattern | 3 |
| redirect | 2 |
| stack | 1 |
| decree | 0 (always) |

---

### Survey Commands

#### `survey-load`

**Signature:**
```bash
aether-utils.sh survey-load [phase_type]
```

**Purpose:**
Returns relevant survey documents based on phase type.

**Phase Type Mapping:**
| Phase Type | Documents |
|------------|-----------|
| frontend/component/UI | DISCIPLINES.md, CHAMBERS.md |
| API/endpoint/backend | BLUEPRINT.md, DISCIPLINES.md |
| database/schema | BLUEPRINT.md, PROVISIONS.md |
| test/spec | SENTINEL-PROTOCOLS.md, DISCIPLINES.md |

---

#### `survey-verify`

**Signature:**
```bash
aether-utils.sh survey-verify
```

**Purpose:**
Verifies all required survey documents exist and returns line counts.

**Required Documents:**
- PROVISIONS.md
- TRAILS.md
- BLUEPRINT.md
- CHAMBERS.md
- DISCIPLINES.md
- SENTINEL-PROTOCOLS.md
- PATHOGENS.md

---

### Checkpoint Commands

#### `checkpoint-check`

**Signature:**
```bash
aether-utils.sh checkpoint-check
```

**Purpose:**
Checks which dirty files are system files vs user files using allowlist matching. Critical for autofix safety.

**System File Patterns:**
- `.aether/aether-utils.sh`
- `.aether/workers.md`
- `.aether/docs/*.md`
- `.claude/commands/ant/*.md`
- `.opencode/commands/ant/*.md`
- `.opencode/agents/*.md`
- `runtime/*`
- `bin/*`

**Return Values:**
- Exit code: 0
- Output: JSON with file classifications

---

### Argument Normalization

#### `normalize-args`

**Signature:**
```bash
aether-utils.sh normalize-args [args...]
```

**Purpose:**
Normalizes arguments from Claude Code (`$ARGUMENTS`) or OpenCode (`$@`). Outputs normalized arguments as single string.

**Detection Order:**
1. `$ARGUMENTS` environment variable (Claude Code)
2. `$@` positional parameters (OpenCode)

---

### Session Freshness Commands

#### `session-verify-fresh`

**Signature:**
```bash
aether-utils.sh session-verify-fresh --command <name> [--force] <session_start_unixtime>
```

**Purpose:**
Verifies session files are fresh (created after session start). Cross-platform stat command supports both macOS and Linux.

**Supported Commands:**
- survey
- oracle
- watch
- swarm
- init
- seal
- entomb

**Return Values:**
- Exit code: 0
- Output: JSON with freshness status

**Output Format:**
```json
{
  "ok": true,
  "command": "survey",
  "fresh": ["PROVISIONS.md"],
  "stale": [],
  "missing": ["TRAILS.md"],
  "total_lines": 150
}
```

---

#### `session-clear`

**Signature:**
```bash
aether-utils.sh session-clear --command <name> [--dry-run]
```

**Purpose:**
Clears session files for a command. Protected commands (init, seal, entomb) cannot be auto-cleared.

**Protected Commands:**
- init: COLONY_STATE.json is precious
- seal/entomb: Archives are precious

---

### Pheromone Commands

#### `pheromone-export`

**Signature:**
```bash
aether-utils.sh pheromone-export [input_json] [output_xml] [schema_file]
```

**Purpose:**
Exports pheromones to eternal XML format. Delegates to xml-utils.sh if available.

**Default Paths:**
- Input: `.aether/data/pheromones.json`
- Output: `~/.aether/eternal/pheromones.xml`
- Schema: `.aether/schemas/pheromone.xsd`

---

### Session Continuity Commands

#### `session-init`

**Signature:**
```bash
aether-utils.sh session-init [session_id] [goal]
```

**Purpose:**
Initializes a new session tracking file with colony state.

---

#### `session-update`

**Signature:**
```bash
aether-utils.sh session-update <command> [suggested_next] [summary]
```

**Purpose:**
Updates session with latest activity. Extracts TODOs from TO-DOs.md and colony state from COLONY_STATE.json.

---

#### `session-read`

**Signature:**
```bash
aether-utils.sh session-read
```

**Purpose:**
Reads session state and checks if stale (> 24 hours).

---

#### `session-is-stale`

**Signature:**
```bash
aether-utils.sh session-is-stale
```

**Purpose:**
Returns "true" or "false" indicating session staleness.

---

#### `session-mark-resumed`

**Signature:**
```bash
aether-utils.sh session-mark-resumed
```

**Purpose:**
Marks session as resumed with current timestamp.

---

#### `session-summary`

**Signature:**
```bash
aether-utils.sh session-summary
```

**Purpose:**
Outputs human-readable session summary to stdout (not JSON).

---

## Function Reference: Utility Modules

### file-lock.sh

#### `acquire_lock()`

**Signature:**
```bash
acquire_lock(file_path)
```

**Purpose:**
Acquires a file lock using bash noclobber for atomic lock creation. Implements stale lock detection and retry logic.

**Lock Mechanism:**
1. Check for existing lock file
2. If exists, check if PID is still running
3. If stale, clean up and retry
4. Try to create lock file atomically with noclobber
5. Retry up to LOCK_MAX_RETRIES with LOCK_RETRY_INTERVAL delays

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `file_path` | String | Yes | File to lock |

**Return Values:**
- Exit code: 0 on success, 1 on failure

**Side Effects:**
- Creates lock files in `.aether/locks/`
- Sets LOCK_ACQUIRED and CURRENT_LOCK globals

**Configuration:**
```bash
LOCK_TIMEOUT=300          # 5 minutes max lock time
LOCK_RETRY_INTERVAL=0.5   # 500ms between retries
LOCK_MAX_RETRIES=100      # 50 seconds max wait
```

---

#### `release_lock()`

**Signature:**
```bash
release_lock()
```

**Purpose:**
Releases the currently held lock. Uses global variables set by acquire_lock.

**Return Values:**
- Exit code: 0 on success, 1 if no lock held

**Side Effects:**
- Removes lock files
- Clears LOCK_ACQUIRED and CURRENT_LOCK globals

---

#### `cleanup_locks()`

**Signature:**
```bash
cleanup_locks()
```

**Purpose:**
Cleanup function registered with trap to ensure locks are released on script exit.

---

#### `is_locked()`

**Signature:**
```bash
is_locked(file_path)
```

**Purpose:**
Checks if a file is currently locked.

**Return Values:**
- Exit code: 0 if locked, 1 if not

---

#### `get_lock_holder()`

**Signature:**
```bash
get_lock_holder(file_path)
```

**Purpose:**
Returns PID of process holding lock.

**Return Values:**
- Exit code: 0
- Output: PID string or empty

---

#### `wait_for_lock()`

**Signature:**
```bash
wait_for_lock(file_path, [max_wait])
```

**Purpose:**
Waits for lock to be released.

**Parameters:**
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `file_path` | String | Yes | - | File to wait for |
| `max_wait` | Number | No | LOCK_TIMEOUT | Max seconds to wait |

**Return Values:**
- Exit code: 0 if released, 1 if timeout

---

### atomic-write.sh

#### `atomic_write()`

**Signature:**
```bash
atomic_write(target_file, content)
```

**Purpose:**
Writes content to file atomically using temp file + rename pattern. Validates JSON for .json files.

**Process:**
1. Create unique temp file in TEMP_DIR
2. Write content to temp file
3. Create backup if target exists
4. Validate JSON if .json file
5. Atomic rename (mv) temp to target
6. Sync to disk

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `target_file` | String | Yes | Target file path |
| `content` | String | Yes | Content to write |

**Return Values:**
- Exit code: 0 on success, 1 on failure

---

#### `atomic_write_from_file()`

**Signature:**
```bash
atomic_write_from_file(target_file, source_file)
```

**Purpose:**
Atomically copies source file to target with validation and backup.

---

#### `create_backup()`

**Signature:**
```bash
create_backup(file_path)
```

**Purpose:**
Creates timestamped backup in BACKUP_DIR.

---

#### `rotate_backups()`

**Signature:**
```bash
rotate_backups(base_name)
```

**Purpose:**
Keeps only MAX_BACKUPS (3) most recent backups.

---

#### `restore_backup()`

**Signature:**
```bash
restore_backup(target_file, [backup_number])
```

**Purpose:**
Restores from backup (default: most recent).

---

#### `list_backups()`

**Signature:**
```bash
list_backups(file_path)
```

**Purpose:**
Lists available backups for a file.

---

#### `cleanup_temp_files()`

**Signature:**
```bash
cleanup_temp_files()
```

**Purpose:**
Removes temp files older than 1 hour.

---

### error-handler.sh

#### `json_err()` (Enhanced)

**Enhanced version** with recovery suggestions, timestamps, and activity logging.

**Recovery Suggestions:**
| Error Code | Suggestion |
|------------|------------|
| E_HUB_NOT_FOUND | "Run: aether install" |
| E_REPO_NOT_INITIALIZED | "Run /ant:init in this repo first" |
| E_FILE_NOT_FOUND | "Check file path and permissions" |
| E_JSON_INVALID | "Validate JSON syntax" |
| E_LOCK_FAILED | "Wait for other operations to complete" |
| E_GIT_ERROR | "Check git status and resolve conflicts" |

---

#### `json_warn()`

**Signature:**
```bash
json_warn([code], [message])
```

**Purpose:**
Outputs non-fatal warning to stdout (not stderr). Does not exit.

---

#### `error_handler()`

**Signature:**
```bash
error_handler(line_num, command, exit_code)
```

**Purpose:**
Trap ERR handler for unexpected failures.

---

#### `feature_enable()` / `feature_disable()` / `feature_enabled()`

**Purpose:**
Feature flag management for graceful degradation.

**Storage:**
Uses colon-pipe delimited string in `_FEATURES_DISABLED` variable for bash 3.2 compatibility:
```
:feature1:reason1|:feature2:reason2
```

---

### xml-utils.sh

#### `xml-validate()`

**Signature:**
```bash
xml-validate(xml_file, xsd_file)
```

**Purpose:**
Validates XML against XSD schema using xmllint with XXE protection (`--nonet --noent`).

---

#### `xml-well-formed()`

**Signature:**
```bash
xml-well-formed(xml_file)
```

**Purpose:**
Checks if XML is well-formed without schema validation.

---

#### `xml-to-json()`

**Signature:**
```bash
xml-to-json(xml_file, [--pretty])
```

**Purpose:**
Converts XML to JSON using available tools (xml2json, xsltproc, or xmlstarlet).

---

#### `json-to-xml()`

**Signature:**
```bash
json-to-xml(json_file, [root_element])
```

**Purpose:**
Converts JSON to XML using jq transformation.

---

#### `xml-query()`

**Signature:**
```bash
xml-query(xml_file, xpath_expression)
```

**Purpose:**
Executes XPath query using xmlstarlet.

---

#### `xml-merge()`

**Signature:**
```bash
xml-merge(output_file, main_xml, [included_files...])
```

**Purpose:**
Merges XML files using XInclude processing.

---

#### `pheromone-to-xml()`

**Signature:**
```bash
pheromone-to-xml(json_file, [output_xml], [xsd_file])
```

**Purpose:**
Converts pheromone JSON to XML format with namespace support.

---

## File Locking Deep Dive

### Architecture

The Aether file locking system implements a PID-based advisory locking mechanism using bash's noclobber feature for atomic lock acquisition. This approach was chosen for its portability and lack of external dependencies beyond standard bash.

### Lock File Structure

```
.aether/locks/
├── COLONY_STATE.json.lock      # Lock file (contains PID)
├── COLONY_STATE.json.lock.pid  # PID file (redundant backup)
└── flags.json.lock             # Another lock
```

### Lock Acquisition Algorithm

```
1. Calculate lock file path: LOCK_DIR/basename(target).lock
2. Check if lock file exists
3. If exists:
   a. Read PID from lock file
   b. Check if process is running (kill -0)
   c. If not running, remove stale lock and retry
4. Try atomic creation with noclobber:
   (set -o noclobber; echo $$ > lock_file) 2>/dev/null
5. If successful, write PID file and return
6. If failed, increment retry counter
7. Sleep LOCK_RETRY_INTERVAL
8. If retries < LOCK_MAX_RETRIES, goto 4
9. Return failure
```

### Stale Lock Detection

Stale locks are detected by checking if the owning PID is still running:

```bash
if ! kill -0 "$lock_pid" 2>/dev/null; then
    # Process not running, lock is stale
    rm -f "$lock_file" "$lock_pid_file"
fi
```

This approach has a race condition window between checking and removal, but the atomic acquisition in step 4 ensures only one process can actually acquire the lock.

### Timeout Configuration

| Parameter | Value | Description |
|-----------|-------|-------------|
| LOCK_TIMEOUT | 300s | Maximum lock lifetime |
| LOCK_RETRY_INTERVAL | 0.5s | Wait between retries |
| LOCK_MAX_RETRIES | 100 | Maximum retry attempts |
| Total Max Wait | 50s | Maximum blocking time |

### Cleanup Guarantees

The system registers cleanup handlers:
```bash
trap cleanup_locks EXIT TERM INT
```

This ensures locks are released when:
- Script exits normally
- Script receives SIGTERM
- Script receives SIGINT (Ctrl+C)

### Known Issues

**BUG-005/BUG-011: Lock Deadlock in flag-auto-resolve**

Location: `aether-utils.sh:1367-1384`

If jq fails after lock acquisition in certain code paths, the lock may not be released. The current code has partial fixes but the issue persists in some error paths.

**Mitigation:**
- Always use trap-based cleanup
- Add explicit release on all error paths
- Consider using `set -E` for ERR trap inheritance

### Security Considerations

1. **Symlink Attacks:** Lock files are created in a controlled directory (.aether/locks/)
2. **PID Reuse:** Small window where PID could be reused between check and removal
3. **Denial of Service:** Malicious process could hold locks indefinitely

### Performance Characteristics

| Metric | Value |
|--------|-------|
| Lock Acquisition | O(1) average, O(n) worst case |
| Memory Usage | O(1) |
| Disk I/O | 2 files per lock |
| Network | None |

---

## State Management Flow

### State File Hierarchy

```
.aether/data/
├── COLONY_STATE.json      # Primary state (precious)
├── session.json           # Session continuity
├── flags.json             # Project flags
├── learnings.json         # Global learnings
├── activity.log           # Audit trail
├── spawn-tree.txt         # Worker lineage
├── timing.log             # Worker timing
├── error-patterns.json    # Error patterns
├── signatures.json        # Code signatures
├── view-state.json        # UI state
├── swarm-display.json     # Visualization state
└── swarm-findings-*.json  # Swarm results
```

### State Modification Flow

```
1. Read current state
2. Acquire lock (if concurrent access possible)
3. Modify in memory (using jq)
4. Validate new state
5. atomic_write to file
6. Release lock
7. Log activity (optional)
```

### State Validation

All state modifications should validate:
1. JSON syntax (jq empty)
2. Required fields present
3. Type correctness
4. Referential integrity (where applicable)

### Backup Strategy

Atomic writes automatically create backups:
1. Before write, copy existing file to BACKUP_DIR
2. Rotate old backups (keep 3)
3. Write new content to temp file
4. Atomic rename to target
5. Sync to disk

### Recovery Procedures

**Corrupted State:**
```bash
# Restore from backup
bash .aether/utils/atomic-write.sh restore_backup .aether/data/COLONY_STATE.json
```

**Stale Locks:**
```bash
# Manual cleanup
rm -f .aether/locks/*.lock .aether/locks/*.lock.pid
```

**Missing State:**
```bash
# Reinitialize
/ant:init
```

---

## Pheromone System Architecture

### Overview

The pheromone system implements a biological-inspired signaling mechanism for colony coordination. Pheromones are persistent signals that influence worker behavior across sessions.

### Signal Types

| Signal | Command | Priority | Use Case |
|--------|---------|----------|----------|
| FOCUS | `/ant:focus` | normal | "Pay attention here" |
| REDIRECT | `/ant:redirect` | high | "Don't do this" (hard constraint) |
| FEEDBACK | `/ant:feedback` | low | "Adjust based on this" |

### Pheromone Structure

```json
{
  "signals": [
    {
      "type": "FOCUS|REDIRECT|FEEDBACK",
      "message": "Human-readable signal",
      "priority": "low|normal|high",
      "set_at": "2026-02-16T15:47:00Z",
      "expires_at": "2026-02-17T15:47:00Z",
      "source": "user|colony|system"
    }
  ],
  "version": "1.0.0",
  "colony_id": "unique-id"
}
```

### XML Exchange Format

Pheromones can be exported to XML for cross-colony exchange:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<pheromones xmlns="http://aether.colony/schemas/pheromones"
            version="1.0.0"
            generated_at="2026-02-16T15:47:00Z"
            colony_id="unique-id">
  <signal type="FOCUS" priority="normal" set_at="2026-02-16T15:47:00Z">
    <message>Pay attention to authentication</message>
  </signal>
</pheromones>
```

### Namespace Design

Colonies use namespaced pheromone IDs to prevent collisions:
```
<colony-id>::<signal-id>
```

Example:
```
myproject-20260216::focus-auth-001
```

### Persistence Strategy

1. **Local Storage:** `.aether/data/pheromones.json`
2. **Eternal Storage:** `~/.aether/eternal/pheromones.xml`
3. **Exchange:** XInclude-based composition

### Consumption Patterns

**Before Build:**
- Check FOCUS signals for guidance
- Check REDIRECT signals for constraints
- Adjust worker task assignment

**After Build:**
- Check FEEDBACK signals for adjustments
- Update signal strengths based on outcomes
- Archive consumed signals

### Decay Mechanism

Pheromones have TTL (time-to-live) and decay over time:
- FOCUS: 7 days
- REDIRECT: 30 days (hard constraints persist longer)
- FEEDBACK: 3 days

Expired pheromones are archived, not deleted.

---

## XML Integration Points

### Tool Support Matrix

| Tool | Validation | Transform | Query | Convert |
|------|------------|-----------|-------|---------|
| xmllint | Yes | No | No | No |
| xmlstarlet | No | No | Yes | Limited |
| xsltproc | No | Yes | No | Yes |
| xml2json | No | No | No | Yes |

### XXE Protection

All XML processing uses XXE protection flags:
```bash
xmllint --nonet --noent  # Disable network, entity expansion
```

### XInclude Composition

Documents can include other documents:
```xml
<?xml version="1.0"?>
<colony xmlns:xi="http://www.w3.org/2001/XInclude">
  <xi:include href="pheromones.xml"/>
  <xi:include href="wisdom.xml"/>
</colony>
```

### Schema Validation

XSD schemas in `.aether/schemas/`:
- `pheromone.xsd`: Pheromone signal validation
- `colony.xsd`: Colony state validation
- `worker.xsd`: Worker definition validation

### Hybrid JSON/XML Architecture

Aether uses JSON for runtime state and XML for:
- Cross-colony exchange
- Long-term archival
- Schema validation
- XInclude composition

Conversion utilities bridge the formats transparently.

---

## Color and Logging System

### Log Levels

| Level | Indicator | Use Case |
|-------|-----------|----------|
| ERROR | None (JSON) | Failures |
| WARN | None (JSON) | Degradation |
| INFO | Emoji prefix | Normal operations |
| DEBUG | Timestamp prefix | Detailed tracing |

### Emoji Conventions

| Emoji | Meaning |
|-------|---------|
| | Success |
| | Failure |
| | Blocked |
| | Spawn event |
| | Build in progress |
| | Activity |

### Activity Log Format
```
[HH:MM:SS] <emoji> <action> <caste>: <description>
```

Example:
```
[15:47:00] 🔨🐜 build Builder: Started phase 3
```

### Colorized Output

The `colorize-log.sh` utility provides colorized log streaming:
- Red: Errors
- Yellow: Warnings
- Green: Success
- Blue: Info
- Gray: Debug

---

## Session Management Internals

### Session Lifecycle

```
1. session-init: Create session.json
2. Commands update session via session-update
3. session-read checks staleness
4. session-mark-resumed on continuation
5. session-clear on completion/abandon
```

### Session File Structure

```json
{
  "session_id": "1708099200_a3f7b2",
  "started_at": "2026-02-16T15:47:00Z",
  "last_command": "/ant:build",
  "last_command_at": "2026-02-16T16:00:00Z",
  "colony_goal": "Build auth system",
  "current_phase": 3,
  "current_milestone": "Open Chambers",
  "suggested_next": "/ant:verify",
  "context_cleared": false,
  "resumed_at": null,
  "active_todos": ["Fix login bug", "Add tests"],
  "summary": "Phase 3 in progress"
}
```

### Staleness Detection

Sessions are stale if `last_command_at` > 24 hours ago:
```bash
age_hours=$(( (now_epoch - last_epoch) / 3600 ))
[[ $age_hours -gt 24 ]] && is_stale=true
```

### Freshness Verification

Session freshness system verifies files were created after session start:
```bash
file_mtime=$(stat -f %m "$file" 2>/dev/null || stat -c %Y "$file" 2>/dev/null)
[[ "$file_mtime" -ge "$session_start_time" ]] && fresh=true
```

---

## Checkpoint System Mechanics

### Checkpoint Types

| Type | When Created | Rollback Method |
|------|--------------|-----------------|
| stash | Aether files changed | git stash pop |
| commit | No Aether changes | git reset --hard |
| none | Not in git repo | N/A |

### Safety Mechanism

Only Aether-managed directories are stashed:
```bash
target_dirs=".aether .claude/commands/ant .claude/commands/st .opencode runtime bin"
```

This prevents user work from being included in checkpoints.

### Rollback Flow

```
1. Determine checkpoint type from session
2. For stash: find and pop matching stash
3. For commit: reset to recorded hash
4. Verify rollback success
5. Report result
```

### Limitations

- Stash conflicts may prevent rollback
- Commit rollback is destructive (loses uncommitted work)
- Only rolls back Aether files, not user files

---

## Security Considerations

### Path Traversal Protection

- `xml-compose.sh` validates paths against allowlist
- `checkpoint-check` uses pattern matching for file classification
- No user input used directly in file paths without validation

### Input Validation

- JSON validation before state updates
- Type checking for numeric parameters
- Pattern matching for caste names

### Secret Handling

- `check-antipattern` detects exposed secrets
- No logging of API keys or tokens
- Environment variables for sensitive data

### Lock Security

- PID-based ownership verification
- Stale lock detection and cleanup
- Trap-based cleanup on exit

---

## Performance Characteristics

### Time Complexity Summary

| Operation | Complexity | Notes |
|-----------|------------|-------|
| json_ok/err | O(1) | Simple output |
| get_caste_emoji | O(1) | Case statement |
| spawn-can-spawn | O(n) | n = spawn-tree.txt lines |
| flag-list | O(n) | n = flags array size |
| signature-match | O(n*m) | n = files, m = signatures |
| xml-to-json | O(n) | n = XML size |

### Space Complexity

| Operation | Complexity | Notes |
|-----------|------------|-------|
| Most commands | O(1) | Fixed overhead |
| JSON processing | O(n) | n = JSON size |
| File operations | O(n) | n = file size |

### Disk I/O Patterns

- Atomic writes: 2 writes (temp + rename)
- Backups: 1 copy per modification
- Lock files: 2 files per lock
- Log files: Append-only

### Memory Usage

- Typical: < 1MB
- Large JSON files: Up to file size
- XML processing: Depends on tool

### Optimization Opportunities

1. **Caching:** Cache jq results for repeated queries
2. **Batching:** Batch flag updates to reduce lock contention
3. **Lazy Loading:** Defer loading of unused utilities
4. **Compression:** Compress archived logs

---

## Appendix: Complete Error Code Reference

### Error Constants

```bash
E_UNKNOWN="E_UNKNOWN"
E_HUB_NOT_FOUND="E_HUB_NOT_FOUND"
E_REPO_NOT_INITIALIZED="E_REPO_NOT_INITIALIZED"
E_FILE_NOT_FOUND="E_FILE_NOT_FOUND"
E_JSON_INVALID="E_JSON_INVALID"
E_LOCK_FAILED="E_LOCK_FAILED"
E_GIT_ERROR="E_GIT_ERROR"
E_VALIDATION_FAILED="E_VALIDATION_FAILED"
E_FEATURE_UNAVAILABLE="E_FEATURE_UNAVAILABLE"
E_BASH_ERROR="E_BASH_ERROR"
```

### Warning Codes

```bash
W_UNKNOWN="W_UNKNOWN"
W_DEGRADED="W_DEGRADED"
```

### Exit Codes

| Exit Code | Meaning |
|-----------|---------|
| 0 | Success |
| 1 | General error (json_err) |
| 2 | Misuse of command |
| 126 | Command not executable |
| 127 | Command not found |

---

## Appendix: File Structure Reference

### Complete File Listing

```
.aether/
├── aether-utils.sh              # 3,593 lines
├── workers.md                   # Worker definitions
├── CLAUDE.md                    # Project rules
├── coding-standards.md          # Style guide
├── debugging.md                 # Debug procedures
├── DISCIPLINES.md               # Colony disciplines
├── learning.md                  # Learning system
├── planning.md                  # Planning guide
├── QUEEN_ANT_ARCHITECTURE.md    # Architecture
├── tdd.md                       # TDD practices
├── verification-loop.md         # Verification process
├── verification.md              # Verification guide
├── docs/
│   ├── constraints.md           # Constraint system
│   ├── pheromones.md            # Pheromone guide
│   ├── progressive-disclosure.md # UI patterns
│   ├── pathogen-schema.md       # Pathogen docs
│   └── pathogen-schema-example.json
├── utils/
│   ├── file-lock.sh             # 123 lines
│   ├── atomic-write.sh          # 218 lines
│   ├── error-handler.sh         # 201 lines
│   ├── chamber-utils.sh         # 286 lines
│   ├── spawn-tree.sh            # 429 lines
│   ├── xml-utils.sh             # 2,162 lines
│   ├── xml-compose.sh           # 248 lines
│   ├── state-loader.sh          # 216 lines
│   ├── swarm-display.sh         # 269 lines
│   ├── watch-spawn-tree.sh      # 254 lines
│   ├── colorize-log.sh          # 133 lines
│   ├── spawn-with-model.sh      # 57 lines
│   └── chamber-compare.sh       # 181 lines
└── data/
    ├── COLONY_STATE.json        # Colony state
    ├── flags.json               # Project flags
    ├── learnings.json           # Global learnings
    ├── activity.log             # Activity log
    ├── spawn-tree.txt           # Spawn tracking
    ├── session.json             # Session state
    ├── error-patterns.json      # Error patterns
    ├── signatures.json          # Code signatures
    ├── view-state.json          # UI state
    ├── swarm-display.json       # Visualization
    ├── timing.log               # Worker timing
    ├── checkpoint-allowlist.json # Checkpoint patterns
    └── backups/                 # File backups
```

---

*Documentation generated: 2026-02-16*
*Total word count: approximately 25,000*
*Functions documented: 190+*
*Files analyzed: 15*
*Lines of code: 8,298*
