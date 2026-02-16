---
phase: 05
plan: 02
name: spawn-tree-reconstruction
subsystem: state-management
tags: [bash, spawn-tree, state-restoration, json]

requires:
  - 05-01-state-loading-utility

provides:
  - spawn-tree-persistence
  - parent-child-relationships
  - spawn-depth-calculation
  - active-spawn-query

affects:
  - 05-03-context-restoration

tech-stack:
  added: []
  patterns:
    - Bash 3.2 compatible temporary file storage
    - Pipe-delimited log parsing
    - JSON output for all functions

key-files:
  created:
    - .aether/utils/spawn-tree.sh
    - tests/unit/spawn-tree.test.js
  modified:
    - .aether/aether-utils.sh

decisions:
  - Use temporary files instead of associative arrays for Bash 3.2 compatibility
  - Safety limit of 5 levels in depth calculation to prevent infinite loops
  - Default depth of 1 for unknown ants
  - JSON output format consistent with other aether-utils commands

metrics:
  duration: "1 hour"
  completed: 2026-02-14
---

# Phase 5 Plan 2: Spawn Tree Reconstruction Summary

## Overview

Implemented spawn tree persistence and reconstruction across sessions. The spawn tree tracks parent-child relationships between ants, enabling depth calculation and active spawn queries.

## What Was Built

### spawn-tree.sh Module

Created `.aether/utils/spawn-tree.sh` with the following functions:

1. **parse_spawn_tree** - Parses spawn-tree.txt into structured JSON
   - Reads pipe-delimited format (timestamp|parent|caste|child|task|status)
   - Handles completion events (timestamp|ant|status|summary)
   - Builds parent-child relationships
   - Outputs complete tree with metadata

2. **get_spawn_depth** - Calculates depth in the spawn tree
   - Queen returns depth 0
   - Unknown ants return depth 1 (default)
   - Traverses parent chain with safety limit of 5 levels

3. **get_active_spawns** - Lists currently active spawns
   - Filters out completed/failed/blocked spawns
   - Returns array with name, caste, parent, task, spawned_at

4. **get_spawn_children** - Gets direct children of a spawn
   - Returns JSON array of child names
   - Reads directly from spawn-tree.txt

5. **get_spawn_lineage** - Gets full ancestry from ant to Queen
   - Returns array from ant up to Queen (inclusive)
   - Example: ["Forge-34", "Queen"] for depth 1

6. **reconstruct_tree_json** - Full tree reconstruction
   - Wrapper around parse_spawn_tree
   - Includes metadata: total_count, active_count, completed_count

### aether-utils.sh Integration

Added three new subcommands:

- **spawn-tree-load** - Returns full tree as JSON
- **spawn-tree-active** - Returns only active spawns
- **spawn-tree-depth** - Returns depth for specific ant

All commands follow the same JSON output format as other aether-utils commands.

### Test Coverage

Created comprehensive test suite in `tests/unit/spawn-tree.test.js`:

1. spawn-tree-load returns valid tree JSON
2. spawn-tree-active returns only active spawns
3. spawn-tree-depth returns 0 for Queen
4. spawn-tree-depth returns correct depth for known spawn
5. spawn-tree-depth returns depth 1 for unknown ant
6. spawn-tree-load handles missing file gracefully
7. spawn-tree-depth handles deep chains correctly
8. spawn-tree-load includes parent-child relationships
9. spawn-tree-active returns empty array when no active spawns

All 9 tests pass.

## Design Decisions

### Bash 3.2 Compatibility

Used temporary files for data storage instead of associative arrays (which require Bash 4+). This ensures the module works on macOS and older systems.

### Safety Limits

Depth calculation stops after 5 parent traversals to prevent infinite loops in case of circular references (which shouldn't happen, but safety first).

### JSON Consistency

All functions output JSON matching the format used by other aether-utils commands: `{"ok": true, "result": ...}` for success, structured error objects for failures.

## Verification

- [x] spawn-tree.sh exists with all reconstruction functions
- [x] spawn-tree-load outputs valid JSON tree
- [x] spawn-tree-active lists only active spawns
- [x] spawn-tree-depth calculates correctly
- [x] Parent-child relationships preserved in reconstruction
- [x] Tests pass: npm test -- tests/unit/spawn-tree.test.js (9/9)

## Success Criteria

1. [x] Spawn tree can be fully reconstructed from spawn-tree.txt (STATE-03)
2. [x] Parent-child relationships are preserved across sessions
3. [x] Spawn depth calculation works correctly
4. [x] Active spawns can be queried
5. [x] All tests pass

## Deviations from Plan

None - plan executed exactly as written.

## Next Phase Readiness

This plan provides the foundation for:
- Context restoration (05-03) - can now query spawn tree for active work
- Pause/resume functionality - can identify what was in progress
- Colony state visualization - full tree structure available
