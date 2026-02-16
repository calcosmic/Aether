# Session Freshness Detection API

## Overview

The session freshness detection system prevents stale session files from silently breaking Aether workflows. It provides a generic mechanism for commands to verify that their session files were created during the current session, not left over from previous interrupted runs.

**Status:** Implemented (Phase 1-9 Complete)
**Version:** 1.0
**Last Updated:** 2026-02-16

---

## Commands

### session-verify-fresh

Verify that session files exist and were created after a specified timestamp.

**Usage:**
```bash
bash .aether/aether-utils.sh session-verify-fresh --command <name> [--force] <session_start_unixtime>
```

**Parameters:**
- `--command <name>` - Command context (survey, oracle, watch, swarm, init, seal, entomb)
- `--force` - Accept any existing file regardless of timestamp
- `<session_start_unixtime>` - Unix timestamp to compare against (optional)

**Output:**
```json
{
  "ok": boolean,
  "command": "string",
  "fresh": ["file1.md", ...],
  "stale": ["file2.md", ...],
  "missing": ["file3.md", ...],
  "total_lines": number
}
```

**Fields:**
- `ok` - `true` if all required files are fresh (not stale), `false` otherwise
- `command` - The command name that was verified
- `fresh` - Array of files that exist and have mtime >= session_start
- `stale` - Array of files that exist but have mtime < session_start (older than session)
- `missing` - Array of required files that don't exist
- `total_lines` - Total line count across all found files

**Example:**
```bash
# Check if survey files are fresh
$ bash .aether/aether-utils.sh session-verify-fresh --command survey "" $(date +%s)
{"ok":true,"command":"survey","fresh":["PROVISIONS.md","TRAILS.md"],"stale":[],"missing":[],"total_lines":42}

# Force mode accepts stale files
$ bash .aether/aether-utils.sh session-verify-fresh --command survey --force "" $(date +%s)
{"ok":true,"command":"survey","fresh":["PROVISIONS.md"],"stale":[],"missing":[],"total_lines":10}
```

---

### session-clear

Clear session files for a command.

**Usage:**
```bash
bash .aether/aether-utils.sh session-clear --command <name> [--dry-run]
```

**Parameters:**
- `--command <name>` - Command context
- `--dry-run` - List files that would be cleared without actually deleting

**Output:**
```json
{
  "ok": true,
  "result": {
    "command": "string",
    "cleared": "file1.md file2.md",
    "errors": "",
    "dry_run": boolean
  }
}
```

**Protected Commands:**
The following commands are protected and will error if you attempt to clear them:
- `init` - Never auto-clears COLONY_STATE.json
- `seal` - Never auto-clears archives
- `entomb` - Never auto-clears chambers

**Example:**
```bash
# Dry run to see what would be cleared
$ bash .aether/aether-utils.sh session-clear --command survey --dry-run
{"ok":true,"result":{"command":"survey","cleared":"PROVISIONS.md","errors":"","dry_run":true}}

# Actually clear files
$ bash .aether/aether-utils.sh session-clear --command survey
{"ok":true,"result":{"command":"survey","cleared":"PROVISIONS.md","errors":"","dry_run":false}}

# Protected command returns error
$ bash .aether/aether-utils.sh session-clear --command init
{"error":"E_VALIDATION_FAILED","message":"Command 'init' is protected and cannot be auto-cleared..."}
```

---

## Command-Specific Mappings

| Command | Directory | Files |
|---------|-----------|-------|
| survey | `.aether/data/survey/` | PROVISIONS.md, TRAILS.md, BLUEPRINT.md, CHAMBERS.md, DISCIPLINES.md, SENTINEL-PROTOCOLS.md, PATHOGENS.md |
| oracle | `.aether/oracle/` | progress.md, research.json, discoveries/* |
| watch | `.aether/data/` | watch-status.txt, watch-progress.txt |
| swarm | `.aether/data/swarm/` | findings.json, display.json, timing.json |
| init | `.aether/data/` | COLONY_STATE.json, constraints.json |
| seal | `.aether/data/archive/` | manifest.json |
| entomb | `.aether/chambers/` | manifest.json |

---

## Environment Variables

Override directories for testing:

| Variable | Description | Default |
|----------|-------------|---------|
| `SURVEY_DIR` | Survey directory | `.aether/data/survey` |
| `ORACLE_DIR` | Oracle directory | `.aether/oracle` |
| `WATCH_DIR` | Watch directory | `.aether/data` |
| `SWARM_DIR` | Swarm directory | `.aether/data/swarm` |
| `INIT_DIR` | Init directory | `.aether/data` |
| `ARCHIVE_DIR` | Archive directory | `.aether/data/archive` |

**Example:**
```bash
# Test with temporary directory
SURVEY_DIR=/tmp/test-survey bash .aether/aether-utils.sh session-verify-fresh --command survey "" 0
```

---

## Backward Compatibility

### survey-verify-fresh (deprecated)

Use `session-verify-fresh --command survey` instead. The old command is preserved as a wrapper.

```bash
# Old (still works)
bash .aether/aether-utils.sh survey-verify-fresh "" $(date +%s)

# New (recommended)
bash .aether/aether-utils.sh session-verify-fresh --command survey "" $(date +%s)
```

### survey-clear (deprecated)

Use `session-clear --command survey` instead.

```bash
# Old (still works)
bash .aether/aether-utils.sh survey-clear

# New (recommended)
bash .aether/aether-utils.sh session-clear --command survey
```

---

## Integration in Commands

### Pattern for Adding Freshness Detection

Commands that spawn background agents should follow this pattern:

```bash
# 1. Capture session start time
COMMAND_START=$(date +%s)

# 2. Check for stale files
stale_check=$(bash .aether/aether-utils.sh session-verify-fresh --command <name> "" "$COMMAND_START")
has_stale=$(echo "$stale_check" | jq -r '.stale | length')

# 3. Handle stale files (auto-clear or warn)
if [[ "$has_stale" -gt 0 ]]; then
  # For safe operations: auto-clear
  bash .aether/aether-utils.sh session-clear --command <name>
  echo "Cleared stale files for fresh session"

  # For protected operations: warn only
  echo "Warning: Found existing session files"
fi

# 4. Spawn agents/workers
# ... do work ...

# 5. Verify files are fresh after spawning
verify_result=$(bash .aether/aether-utils.sh session-verify-fresh --command <name> "" "$COMMAND_START")
if [[ $(echo "$verify_result" | jq -r '.missing | length') -gt 0 ]]; then
  echo "Warning: Expected files not created"
fi
```

### Commands Using Freshness Detection

| Command | Protected | Auto-clear | Status |
|---------|-----------|------------|--------|
| colonize | No | Yes | ✅ Implemented |
| oracle | No | With `--force` | ✅ Implemented |
| watch | No | Yes (overwrite) | ✅ Implemented |
| swarm | No | Yes | ✅ Implemented |
| init | **Yes** | No | ✅ Implemented |
| seal | **Yes** | No | ✅ Implemented |
| entomb | **Yes** | No | ✅ Implemented |

---

## Cross-Platform Support

Timestamp detection works on both macOS and Linux:

| Platform | Command |
|----------|---------|
| macOS | `stat -f %m` |
| Linux | `stat -c %Y` |

The implementation tries macOS first, then falls back to Linux:

```bash
file_mtime=$(stat -f %m "$doc_path" 2>/dev/null || stat -c %Y "$doc_path" 2>/dev/null || echo "0")
```

---

## Testing

Run the test suite:

```bash
# Run all session freshness tests
bash tests/bash/test-session-freshness.sh

# Expected output:
# =========================================
# Session Freshness Detection Test Suite
# =========================================
#
# PASS: verify_fresh_missing
# PASS: verify_fresh_stale
# ...
#
# =========================================
# Test Summary
# =========================================
# Tests run:   18
# Passed:      18
# Failed:      0
#
# All tests passed!
```

---

## Error Codes

| Code | Meaning |
|------|---------|
| `E_VALIDATION_FAILED` | Invalid arguments or command name |
| `E_FILE_NOT_FOUND` | Required file not found (in other commands) |

---

## Changelog

### 2026-02-16 - v1.0

- Initial implementation
- Added `session-verify-fresh` and `session-clear` commands
- Added backward compatibility wrappers for `survey-verify-fresh` and `survey-clear`
- Implemented freshness detection in colonize, oracle, watch, swarm, init, seal, entomb commands
- Added cross-platform stat support (macOS/Linux)
- Added protected operation handling (init/seal/entomb)

---

## See Also

- Implementation Plan: `docs/session-freshness-implementation-plan.md`
- Main Utilities: `.aether/aether-utils.sh`
- Test Suite: `tests/bash/test-session-freshness.sh`
