# Coding Conventions

**Analysis Date:** 2026-04-01

This document covers conventions across three languages used in the Aether codebase: **Bash** (primary, ~5,500 lines in the dispatcher + 50 utility modules), **JavaScript/Node.js** (CLI and unit tests), and **Go** (early-stage port of colony types and storage).

---

## Bash Conventions

### File Structure

- **Dispatcher:** `.aether/aether-utils.sh` -- single entry point, sources all utility modules at startup, contains the main `case "$1"` router
- **Utility modules:** `.aether/utils/<module>.sh` -- each sourced by the dispatcher; provides one or more underscore-prefixed functions
- **Curation ants:** `.aether/utils/curation-ants/<ant>.sh` -- specialized utility modules, also sourced at startup
- **Naming:** `kebab-case.sh` (e.g., `trust-scoring.sh`, `atomic-write.sh`, `file-lock.sh`)

### Function Design

**Internal functions (sourced, not subcommand):**
- Prefixed with underscore: `_trust_calculate`, `_learning_promote`, `_event_publish`
- Called internally by the dispatcher or other utility functions
- Not exposed as subcommands

**Subcommand routing:**
- Functions are mapped to subcommands via a `case "$1"` block in `aether-utils.sh`
- Subcommand names use `kebab-case`: `trust-calculate`, `learning-observe`, `state-read`
- The hyphenated subcommand name maps to the underscore-prefixed function: `trust-calculate` calls `_trust_calculate`

**Function signatures:**
```bash
# Subcommand-style: positional args
_learning_promote() {
    [[ $# -ge 3 ]] || json_err "$E_VALIDATION_FAILED" "Usage: learning-promote <content> <source_project> <source_phase> [tags]"
    content="$1"
    source_project="$2"
    ...
}

# Flag-style: --flag <value> parsing
_trust_calculate() {
    local source_type=""
    local evidence_type=""
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --source) source_type="${2:-}"; shift 2 ;;
            --evidence) evidence_type="${2:-}"; shift 2 ;;
            *) json_err "$E_VALIDATION_FAILED" "Usage: ..." ;;
        esac
    done
}
```

### Variable Naming

- **Global constants:** `UPPER_SNAKE_CASE` with `_` prefix for "private": `_EVENT_BUS_DEFAULT_TTL`, `_FEATURES_DISABLED`
- **Local variables:** `lowercase` or `lower_snake_case` (inconsistent; both observed)
- **Function-local variables with prefix** to avoid collisions: `sr_state_file`, `srf_field`, `ep_topic`, `ep_payload` (first letter of function name as prefix)
- **Exported globals:** `DATA_DIR`, `COLONY_DATA_DIR`, `AETHER_ROOT`, `SCRIPT_DIR`, `LOCK_ACQUIRED`, `CURRENT_LOCK`

### JSON I/O Protocol

All subcommands output JSON to stdout. This is the central contract:

**Success:**
```bash
json_ok '{"key":"value"}'
# Output: {"ok":true,"result":{"key":"value"}}
```

**Error:**
```bash
json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found" '{"file":"COLONY_STATE.json"}'
# Output to stderr: {"ok":false,"error":{"code":"E_FILE_NOT_FOUND","message":"...","details":{...},"recovery":...,"timestamp":"..."}}
```

**Warning (non-fatal):**
```bash
json_warn "W_UNKNOWN" "message"
# Output to stdout: {"ok":true,"warning":{"code":"...","message":"...","timestamp":"..."}}
```

**The `json_ok` function definition** (line 117 of `aether-utils.sh`):
```bash
json_ok() { printf '{"ok":true,"result":%s}\n' "$1"; }
```

### Error Handling

**Error codes** defined as uppercase constants in `.aether/utils/error-handler.sh`:
```bash
E_UNKNOWN="E_UNKNOWN"
E_FILE_NOT_FOUND="E_FILE_NOT_FOUND"
E_JSON_INVALID="E_JSON_INVALID"
E_LOCK_FAILED="E_LOCK_FAILED"
E_VALIDATION_FAILED="E_VALIDATION_FAILED"
# ... 13 total codes
```

**Error handling patterns:**
1. **Validation guard:** `[[ $# -ge N ]] || json_err "$E_VALIDATION_FAILED" "Usage: ..."` at function start
2. **File existence check:** `[[ ! -f "$file" ]] && json_err "$E_FILE_NOT_FOUND" "..."`
3. **JSON validation:** `echo "$content" | jq empty 2>/dev/null || json_err "$E_JSON_INVALID" "..."`
4. **Trap ERR:** `trap 'if type error_handler &>/dev/null; then error_handler ${LINENO} "$BASH_COMMAND" $?; fi' ERR` in dispatcher
5. **Best-effort logging:** `2>/dev/null || true` for non-critical writes (activity.log, safety-stats.json)

**Feature flags** for graceful degradation:
```bash
feature_enabled "feature_name"  # returns 0 if enabled, 1 if disabled
feature_disable "feature_name" "reason"
```

### jq Usage Patterns

jq is the primary JSON manipulation tool throughout the codebase.

**Read field:**
```bash
jq -r '.field // empty' "$file"
```

**Update file (atomic via variable):**
```bash
updated=$(jq --arg key "$value" '.field = $key' "$file") || json_err "..."
atomic_write "$file" "$updated"
```

**Construct new JSON:**
```bash
jq -nc --arg id "$id" --arg ts "$ts" '{id:$id,timestamp:$ts}'
```

**Validate JSON:**
```bash
jq empty "$file" 2>/dev/null || json_err "$E_JSON_INVALID" "..."
```

**Read array length:**
```bash
jq '.learnings | length' "$file"
```

**Always use `||` after jq to catch parse failures.**

### Atomic Writes and Locking

**Atomic write pattern** (defined in `.aether/utils/atomic-write.sh`):
1. Write to temp file in `.aether/temp/`
2. Validate JSON if `.json` extension
3. Create backup of existing file
4. `mv` temp to target (atomic on same filesystem)
5. `sync` to disk

**File locking** (defined in `.aether/utils/file-lock.sh`):
- Uses `set -o noclobber` for atomic lock creation
- Lock files stored in `.aether/locks/`
- PID sidecar files for stale lock detection
- `acquire_lock` / `release_lock` for global locking
- `acquire_lock_at` / `release_lock_at` for explicit directory locking
- Trap on EXIT/TERM/INT/HUP for cleanup

**Locking is the caller's responsibility.** Functions like `atomic_write` do NOT acquire locks internally.

### Shebang and Strict Mode

All shell scripts use:
```bash
#!/usr/bin/env bash
set -euo pipefail
```

**Bash 3.2 compatibility** is maintained (macOS default). No associative arrays. No `mapfile`. No `readarray`.

### Comments

- Module header comment block at top of each `.sh` file listing provided functions
- `# --- Section headers ---` with em-dash separators
- `# ============================================================================` section dividers
- Inline comments for non-obvious logic
- `# SUPPRESS:OK` annotation for lint suppressions (shellcheck)
- `# Usage:` pattern in function doc comments

---

## JavaScript/Node.js Conventions

### File Structure

- **CLI entry points:** `bin/cli.js`, `bin/npx-entry.js`
- **Library modules:** `bin/lib/<module>.js` (e.g., `state-guard.js`, `errors.js`, `file-lock.js`)
- **Naming:** `kebab-case.js`

### Module Style

**CommonJS `require()`** -- no ES modules:
```javascript
const fs = require('fs');
const { EventTypes, createEvent } = require('./event-types');
```

### Class Design

**Error class hierarchy:**
```javascript
class AetherError extends Error {
  constructor(code, message, details = {}, recovery = null) {
    super(message);
    this.name = 'AetherError';
    this.code = code;
    this.details = details;
    this.recovery = recovery;
    this.timestamp = new Date().toISOString();
    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, AetherError);
    }
  }
  toJSON() { ... }
  toString() { ... }
}
```

**Error codes as object constants:**
```javascript
const ErrorCodes = {
  E_HUB_NOT_FOUND: 'E_HUB_NOT_FOUND',
  E_FILE_SYSTEM: 'E_FILE_SYSTEM',
  // ...
};
```

### JSDoc Comments

Used on classes and public methods:
```javascript
/**
 * @param {string} code - Error code from ErrorCodes
 * @param {string} message - Human-readable error message
 * @param {object} details - Additional error context
 * @returns {object} Structured error representation
 */
```

### Module Annotations

JSDoc module tags on files:
```javascript
/**
 * State Guard Module
 * @module bin/lib/state-guard
 */
```

---

## Go Conventions

### Package Structure

Standard Go layout under `github.com/aether-colony/aether`:

```
cmd/aether/main.go       -- CLI entry point (currently empty main)
internal/config/config.go -- Configuration (stub)
internal/testing/testing.go -- Test helpers (stub)
pkg/agent/agent.go       -- Worker pool (stub)
pkg/colony/colony.go     -- Core types + state machine
pkg/colony/state_machine.go -- State transitions
pkg/events/events.go     -- Event bus (stub)
pkg/graph/graph.go       -- Knowledge graph (stub)
pkg/llm/llm.go           -- LLM client (stub)
pkg/memory/memory.go     -- Wisdom pipeline (stub)
pkg/storage/storage.go   -- Atomic JSON file operations
```

**Pattern:** `pkg/` for reusable public packages, `internal/` for private application code.

### Naming

**Types:** PascalCase: `ColonyState`, `PhaseLearning`, `FlaggedPattern`
**Interfaces:** Not used yet (no interfaces in the codebase)
**Constants:** PascalCase for exported, camelCase for unexported:
```go
const (
    StateREADY     State = "READY"       // exported
    PhasePending   = "pending"           // unexported string constant
)
```
**Sentinel errors:** `ErrInvalidTransition` (exported, package-level `var`)
**Functions:** PascalCase for exported: `Transition()`, `AdvancePhase()`, `NewStore()`
**Methods:** PascalCase: `(s *Store) BasePath()`, `(s *Store) AtomicWrite()`
**Local variables:** camelCase: `lockFile`, `tmpPath`, `updated`
**JSON tags:** snake_case: `json:"colony_version"`, `json:"phase_learnings"`

### Nullable Fields

Go structs use pointer types for JSON fields that can be null:
```go
type ColonyState struct {
    Goal          *string    `json:"goal"`          // null in JSON -> nil in Go
    ColonyName    *string    `json:"colony_name"`
    InitializedAt *time.Time `json:"initialized_at"`
    Plan          Plan       `json:"plan"`          // struct -> never null
}
```

### Error Wrapping

Uses `fmt.Errorf` with `%w` verb for error chain wrapping:
```go
return fmt.Errorf("storage: read %q: %w", path, err)
return fmt.Errorf("%w: %s -> %s is not allowed", ErrInvalidTransition, current, target)
```

**Sentinel errors** checked with `errors.Is()`:
```go
if !errors.Is(err, ErrInvalidTransition) { ... }
```

### Doc Comments

Every exported type, constant, and function has a doc comment:
```go
// ColonyState is the top-level colony state matching COLONY_STATE.json.
type ColonyState struct { ... }

// Transition validates and returns the target state if the transition from
// current to target is legal.
func Transition(current, target State) error { ... }
```

### Package Comments

Every package has a doc comment on the first line:
```go
// Package colony defines the core data types for the Aether colony state system.
package colony
```

### Section Separators

Go files use `// ---------------------------------------------------------------------------` visual separators between logical sections (constants, types, functions).

### JSON Compatibility

Go types are designed for **exact round-trip compatibility** with the existing `COLONY_STATE.json` schema used by the bash implementation. JSON field names use `snake_case` to match the existing shell-produced JSON files.

---

## Cross-Language Conventions

### Error Code Consistency

All three languages share the same error code vocabulary:
- `E_FILE_NOT_FOUND`, `E_JSON_INVALID`, `E_LOCK_FAILED`, `E_VALIDATION_FAILED`, etc.
- Defined as string constants in each language

### JSON Output Format

The `{"ok": true/false, ...}` envelope is used consistently:
- **Bash:** `json_ok()` / `json_err()` produce this format
- **Node.js:** `AetherError.toJSON()` produces a compatible `error` object
- **Go:** Not yet implemented for output (types only)

### Timestamp Format

ISO 8601 UTC: `2026-02-15T16:00:00Z`

Generated with:
- Bash: `date -u +"%Y-%m-%dT%H:%M:%SZ"`
- Node.js: `new Date().toISOString()`
- Go: `time.Now().UTC()` formatted with `time.RFC3339`

---

## Code Style Tools

### Linting

- **Shell:** shellcheck (severity: error) via `npm run lint:shell`
- **JSON:** custom validation via `npm run lint:json` (parses all data JSON files)
- **Sync:** `npm run lint:sync` checks command/agent file synchronization

### Formatting

- **Bash:** No automated formatter configured; manual 2-space indentation observed
- **JavaScript:** No Prettier or ESLint configured
- **Go:** Standard `go fmt` (implied by standard Go conventions)

---

*Convention analysis: 2026-04-01*
