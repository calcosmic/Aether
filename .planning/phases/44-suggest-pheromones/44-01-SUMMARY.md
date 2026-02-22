---
phase: 44-suggest-pheromones
plan: 01
type: execute
wave: 1
subsystem: pheromone-suggestions
tags: [analysis, suggestions, code-quality]
dependency_graph:
  requires: []
  provides: [suggest-analyze, suggest-record, suggest-check, suggest-clear]
  affects: [.aether/aether-utils.sh]
tech_stack:
  added: []
  patterns: [shell-scripting, json-processing, file-analysis]
key_files:
  created: []
  modified:
    - .aether/aether-utils.sh
key_decisions:
  - Use ERR trap disable/enable pattern to handle grep exit code 1 on no matches
  - Use jq for JSON manipulation and deduplication
  - Use sha256 hashes for suggestion deduplication
metrics:
  duration_minutes: 45
  tasks_completed: 2
  files_modified: 1
  lines_added: 282
  lines_removed: 1
---

# Phase 44 Plan 01: Code Analysis Engine Summary

## Overview

Created the code analysis engine that detects patterns worth signaling as pheromones. The `suggest-analyze` command provides the backend analysis capability that scans source files and identifies complexity hotspots, anti-patterns, and gaps.

## What Was Built

### 1. suggest-analyze Command

A new command in `aether-utils.sh` that analyzes the codebase and returns pheromone suggestions.

**Features:**
- Accepts `--source-dir` parameter (auto-detects src/, lib/, or current directory)
- Accepts `--max-suggestions` parameter (default: 5)
- Accepts `--dry-run` flag for testing
- Returns JSON with suggestions array and analysis metadata

**Heuristics Implemented:**

| Pattern | Detection | Suggestion Type | Priority |
|---------|-----------|-----------------|----------|
| Large files | >300 lines | FOCUS | 7 |
| TODO/FIXME/XXX | grep pattern | FEEDBACK | 4 |
| Debug artifacts | console.log, debugger | REDIRECT | 9 |
| Type safety gaps | : any, : unknown | FEEDBACK | 5 |
| High complexity | >20 functions | FOCUS | 6 |
| Test coverage gaps | No .test. file | FOCUS | 5 |

**Exclusions Respected:**
- node_modules/
- .aether/
- dist/
- build/
- .git/
- coverage/
- *.min.js

### 2. Session Tracking Commands

Three helper commands for session-based suggestion tracking:

- **suggest-record** `<hash> <type>`: Records a suggested pheromone hash to session.json
- **suggest-check** `<hash>`: Checks if a hash was already suggested this session
- **suggest-clear**: Clears the suggested_pheromones array from session.json

**Deduplication Strategy:**
1. Filter out suggestions already in pheromones.json (active signals)
2. Filter out suggestions already in session.json suggested_pheromones
3. Sort by priority (REDIRECT > FOCUS > FEEDBACK)
4. Limit to max-suggestions (default 5)

## Implementation Details

### JSON Output Format

```json
{
  "suggestions": [
    {
      "type": "FOCUS|REDIRECT|FEEDBACK",
      "content": "Human-readable suggestion",
      "file": "path/to/file",
      "reason": "Why this was suggested",
      "hash": "sha256-hash",
      "priority": 1-10
    }
  ],
  "analyzed_files": N,
  "patterns_found": N
}
```

### Key Technical Decisions

1. **ERR Trap Handling**: Disabled ERR trap during grep operations because grep returns exit code 1 when no matches are found, which triggers `set -e` to exit the script.

2. **Argument Parsing**: Fixed argument indexing - after the global `shift` at the script level, command arguments start at `$1`, not `$2`.

3. **Deduplication**: Used jq for JSON manipulation since bash 3.2 doesn't support associative arrays for hash tracking.

## Verification Results

All verification steps passed:

```bash
# Returns valid JSON with ok:true
bash .aether/aether-utils.sh suggest-analyze --dry-run

# Returns suggestions array
bash .aether/aether-utils.sh suggest-analyze --source-dir ./bin

# Session tracking works
bash .aether/aether-utils.sh suggest-record test-hash FOCUS
bash .aether/aether-utils.sh suggest-check test-hash  # returns true
bash .aether/aether-utils.sh suggest-clear
```

## Deviations from Plan

None. The implementation follows the plan exactly as written.

## Bug Fixes During Implementation

### Fix 1: ERR Trap Triggering on Grep
**Issue**: `set -e` with ERR trap caused script to exit when grep found no matches (exit code 1).
**Fix**: Disabled ERR trap at command start with `trap '' ERR`, re-enabled before exit.

### Fix 2: Argument Indexing
**Issue**: Commands used `${2:-}` and `${3:-}` but after global `shift`, arguments start at `$1`.
**Fix**: Changed to `${1:-}` and `${2:-}` for suggest-record and suggest-check.

### Fix 3: Shift on Empty Arguments
**Issue**: `shift` without arguments when `$#` is 0 returns exit code 1, triggering `set -e`.
**Fix**: Changed to `shift || true` in suggest-analyze argument parsing.

## Commits

- `524d429`: feat(44-01): add suggest-analyze command for pheromone suggestions

## Next Steps

The code analysis engine is ready. Next plans in Phase 44 should:
1. Integrate suggest-analyze into the build flow
2. Create UI for tick-to-approve suggestions
3. Wire suggestions into the pheromone system

## Self-Check: PASSED

- [x] suggest-analyze command exists and returns valid JSON
- [x] All 6 heuristics implemented
- [x] Exclusion patterns respected
- [x] Duplicate detection works
- [x] Priority scoring orders suggestions correctly
- [x] Session tracking functions work correctly
- [x] Commit created with proper message format
