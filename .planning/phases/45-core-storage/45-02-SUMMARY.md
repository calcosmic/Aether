---
phase: 45-core-storage
plan: 02
subsystem: storage
tags: [go, atomic-write, backup, path-resolution, jsonl]

# Dependency graph
requires:
  - phase: 45-core-storage (plan 01)
    provides: "Typed Go structs for colony data files"
provides:
  - "Storage package with Store type: AtomicWrite, SaveJSON, LoadJSON, AppendJSONL, ReadJSONL"
  - "Backup rotation: CreateBackup + RotateBackups (keep 3, matching shell MAX_BACKUPS)"
  - "Path resolution: ResolveAetherRoot (3-tier fallback) and ResolveDataDir (COLONY_DATA_DIR override)"
  - "JSONL malformed line logging (D-05 compliance)"
affects: [46-event-bus, 47-trust-scoring, 48-memory-pipeline, 49-graph-layer]

# Tech tracking
tech-stack:
  added: [crypto/rand for unique temp file naming]
  patterns: [success-flag defer for temp cleanup, filepath.Glob for backup rotation, mtime-based sort for rotation]

key-files:
  created:
    - pkg/storage/storage.go
    - pkg/storage/backup.go
    - pkg/storage/backup_test.go
    - pkg/storage/paths.go
    - pkg/storage/paths_test.go
    - pkg/storage/storage_malformed_test.go
  modified: []

key-decisions:
  - "Used crypto/rand hex suffix for temp file naming instead of PID-only for concurrent safety"
  - "Used fmt.Fprintf to stderr for malformed JSONL logging instead of log.Printf to keep output testable"
  - "Created full Store type (AtomicWrite, SaveJSON, LoadJSON, AppendJSONL, ReadJSONL) in addition to plan-specified backup/paths files since pkg/storage/storage.go did not exist"

patterns-established:
  - "Success-flag defer pattern: success=false, defer cleanup if !success, set success=true after rename"
  - "resolvePath helper: relative paths joined to basePath, absolute paths passed through"
  - "Backup naming: {path}.bak.{timestamp} matching shell atomic-write.sh pattern"

requirements-completed: [STOR-02, STOR-03]

# Metrics
duration: 12min
completed: 2026-04-01
---

# Phase 45 Plan 02: Storage Infrastructure Summary

**Go storage package with atomic writes, backup rotation (keep 3), AETHER_ROOT/COLONY_DATA_DIR path resolution, and JSONL malformed line logging**

## Performance

- **Duration:** 12 min
- **Started:** 2026-04-01T20:16:04Z
- **Completed:** 2026-04-01T20:28:17Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- Complete `pkg/storage` package with Store type providing atomic write, JSON, and JSONL operations
- Backup rotation matching shell `atomic-write.sh` behavior (keep 3 newest, delete oldest via mtime sort)
- Path resolution with AETHER_ROOT (3-tier: env var, git root, cwd) and COLONY_DATA_DIR override
- JSONL ReadJSONL logs malformed lines to stderr and skips them instead of erroring (D-05)
- AtomicWrite uses success-flag defer pattern eliminating redundant stat+remove after successful rename
- Temp file naming uses crypto/rand hex suffix for concurrent safety

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement backup rotation and path resolution** - `8cbdf6f` (feat)
2. **Task 2: Fix JSONL malformed line handling and AtomicWrite defer cleanup** - `2767bc3` (fix)

## Files Created/Modified
- `pkg/storage/storage.go` - Store type with AtomicWrite, SaveJSON, LoadJSON, AppendJSONL, ReadJSONL
- `pkg/storage/backup.go` - CreateBackup and RotateBackups (maxBackups=3, mtime-based rotation)
- `pkg/storage/backup_test.go` - Tests for backup rotation edge cases (0, 2, 3, 5 backups) and create backup
- `pkg/storage/paths.go` - ResolveAetherRoot (3-tier fallback) and ResolveDataDir (COLONY_DATA_DIR)
- `pkg/storage/paths_test.go` - Tests for env var, git fallback, and default path resolution
- `pkg/storage/storage_malformed_test.go` - Tests for malformed JSONL, blank lines, atomic write cleanup, concurrent writes, JSON validation

## Decisions Made
- Used crypto/rand hex suffix instead of PID-only for temp file naming to prevent collisions in concurrent writes
- Used fmt.Fprintf to stderr for malformed JSONL logging instead of log.Printf, keeping output testable without log capture
- Created full Store type with all methods since pkg/storage/storage.go did not exist (the plan's interface section assumed it existed)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Created base Store type in storage.go**
- **Found during:** Task 1 (Implement backup rotation and path resolution)
- **Issue:** Plan referenced `pkg/storage/storage.go` with Store type as existing interface, but the file did not exist in the worktree
- **Fix:** Created complete storage.go with Store type, NewStore, AtomicWrite, SaveJSON, LoadJSON, AppendJSONL, ReadJSONL, and resolvePath helper
- **Files modified:** pkg/storage/storage.go (created)
- **Verification:** All 18 storage tests pass, race detector clean
- **Committed in:** 8cbdf6f (Task 1 commit)

**2. [Rule 3 - Blocking] Fixed temp file naming for concurrent safety**
- **Found during:** Task 2 (JSONL malformed line handling)
- **Issue:** Temp file naming used PID-only which is identical across goroutines, causing concurrent write failures
- **Fix:** Added crypto/rand hex suffix to temp file names for per-call uniqueness
- **Files modified:** pkg/storage/storage.go
- **Verification:** TestConcurrentWrites_NoRace passes with -race flag
- **Committed in:** 2767bc3 (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (2 blocking issues)
**Impact on plan:** Both auto-fixes necessary for correctness and concurrent safety. No scope creep.

## Issues Encountered
- Backup test timestamps had second-level precision causing file overwrites; fixed by appending unique index suffix to test backup filenames

## Deferred Items
- pkg/colony tests fail with undefined strPtr -- pre-existing from plan 45-01 (helper function missing from colony_test.go). Out of scope for this plan.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Storage package fully operational with 18 passing tests and race detector clean
- Ready for event bus (46) to use AppendJSONL/ReadJSONL
- Ready for trust scoring (47) to use SaveJSON/LoadJSON
- Ready for all downstream phases to use ResolveDataDir for path resolution

---
*Phase: 45-core-storage*
*Completed: 2026-04-01*

## Self-Check: PASSED

All files verified present. All commits verified in git history.
