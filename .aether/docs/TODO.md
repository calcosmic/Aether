# Aether System TODOs

Extracted from Phase 0 self-reference cleanup analysis.

## Unfixed Bugs (Critical - Fix Immediately)

- **BUG-005**: Missing lock release in flag-auto-resolve (aether-utils.sh:1022) [HIGH PRIORITY]
- **BUG-011**: Missing error handling in flag-auto-resolve jq (aether-utils.sh:1022) [HIGH PRIORITY]
- **BUG-002**: Missing release_lock in flag-add error path (aether-utils.sh:814)

## Unfixed Bugs (Standard Priority)

- **BUG-003**: Race condition in backup creation (atomic-write.sh:75)
- **BUG-004**: Missing error code in flag-acknowledge (aether-utils.sh:930)
- **BUG-006**: No lock release on JSON validation failure (atomic-write.sh:66)
- **BUG-007**: 17+ instances of missing error codes (aether-utils.sh various lines)
- **BUG-008**: Missing error code in flag-add jq failure (aether-utils.sh:856)
- **BUG-009**: Missing error codes in file checks (aether-utils.sh:899,933)
- **BUG-010**: Missing error codes in context-update (aether-utils.sh:1758+)
- **BUG-012**: Missing error code in unknown command (aether-utils.sh:2947)

## Unfixed Issues

- **ISSUE-001**: Inconsistent error code usage across codebase
- **ISSUE-002**: Missing exec error handling in model-get/list
- **ISSUE-003**: Incomplete help command (missing newer commands)
- **ISSUE-004**: Template path hardcoded to runtime/ directory
- **ISSUE-005**: Potential infinite loop edge case in spawn-tree
- **ISSUE-006**: Fallback json_err incompatible with enhanced signature
- **ISSUE-007**: Feature detection race condition during sourcing

## Architecture Gaps

- **GAP-001**: No validation of COLONY_STATE.json schema version
- **GAP-002**: No cleanup for stale spawn-tree.txt entries
- **GAP-003**: No retry logic for failed worker spawns
- **GAP-004**: Missing queen-* documentation
- **GAP-005**: No validation of queen-read JSON output
- **GAP-006**: Missing queen-* command documentation
- **GAP-007**: No error code standards documentation
- **GAP-008**: Missing error path test coverage
- **GAP-009**: context-update has no file locking
- **GAP-010**: Missing error code standards documentation

---

*Generated from Oracle Research findings - 2026-02-15*
