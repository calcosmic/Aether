---
phase: 199-front-door-and-classic-contract
plan: "01"
subsystem: testing
tags: [go, git, worktree, teardown-safety]

requires:
  - phase: 198-modern-safety-kernel
    provides: Existing Go test harness and worktree lifecycle coverage
provides:
  - Exact process-local ownership registry for destructive test cleanup
  - Canonical repository and worktree path validation before deletion
  - Decoy regression proof that owner-looking branches and worktrees survive
affects: [199-front-door-and-classic-contract, go-test-harness, worktree-lifecycle]

tech-stack:
  added: []
  patterns: [positive ownership cleanup, canonical path containment, exact Git ref deletion]

key-files:
  created:
    - cmd/testing_cleanup_ownership_test.go
  modified:
    - cmd/testing_main_test.go
    - cmd/worktree_test.go

key-decisions:
  - "Test teardown may delete only exact repository/path/branch triples registered by the creating test."
  - "Every test that can allocate a worktree in the source checkout registers ownership before command execution."

patterns-established:
  - "Positive ownership: destructive cleanup consumes an in-process registry instead of discovering candidates by name or directory shape."
  - "Boundary validation: canonicalize the repository and prospective worktree path, then require strict containment before removal."

requirements-completed: [PROOF-01]

duration: 27min
completed: 2026-09-03
---

# Phase 199 Plan 01: Safe Test Cleanup Summary

**Go test teardown now removes only explicitly registered worktrees and branches, with isolated regression coverage proving plausible owner artifacts survive.**

## Performance

- **Duration:** 27 min
- **Started:** 2026-09-03T13:26:14Z
- **Completed:** 2026-09-03T13:53:39Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments

- Replaced broad worktree enumeration, global pruning, and `phase-*`/`feature/*` branch sweeps with a mutex-protected registry of exact owned artifacts.
- Added canonical repository/path validation that rejects empty roots, repository-root deletion, paths outside the repository, and symlink escapes.
- Added an isolated temporary-repository proof that registered artifacts disappear while decoy branches and an Aether-shaped decoy worktree remain byte-for-byte intact across repeated cleanup.
- Registered every source-checkout worktree fixture so the safer teardown leaves no generated branches or worktrees behind.

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: Add failing exact cleanup ownership proof** - `fa20efa2` (test)
2. **Task 1 GREEN: Restrict test cleanup to owned worktrees** - `d8649ff6` (fix)
3. **Task 2: Prove cleanup preserves owner artifacts** - `340bc63a` (test)
4. **Task 2 verification fix: Register all source worktree fixtures** - `83f2b9c4` (fix)

**Plan metadata:** This summary and the deferred-item record are committed immediately after creation.

## Files Created/Modified

- `cmd/testing_main_test.go` - Stores exact ownership triples, validates their boundaries, and removes only those paths and refs.
- `cmd/worktree_test.go` - Registers source-checkout worktree fixtures before invoking allocation commands and proves repeated cleanup is safe.
- `cmd/testing_cleanup_ownership_test.go` - Creates registered and owner-looking decoy artifacts in `t.TempDir()` and asserts exact post-cleanup branches, paths, and marker bytes.
- `.planning/phases/199-front-door-and-classic-contract/deferred-items.md` - Records the unrelated archived-path test failure found by full-suite verification.

## Decisions Made

- Registration is the sole authority for destructive test cleanup; names beginning with `phase-` or `feature/` and Aether-shaped directories carry no ownership meaning.
- Registry entries include repository root, worktree path, and branch so every destructive Git operation is rooted and exact.
- Source-checkout allocation tests register ownership before execution, allowing cleanup even if the tested command fails partway through creation.

## Verification

- `go test ./cmd -run '^TestWorktreeAllocateAuditLog$' -count=1` passed.
- `go test ./cmd -run '^TestCleanupTestWorktrees(PreservesUnregisteredDecoys|IsIdempotent)$' -count=1` passed.
- `go test ./... -skip '^TestBranchDispositionRecordsAllThreeBranches$' -count=1` passed 8,567 tests across 20 packages.
- `go test ./... -race -skip '^TestBranchDispositionRecordsAllThreeBranches$' -count=1` passed 8,567 tests across 20 packages.
- The required destructive-pattern search returned no prefix sweep, worktree enumeration, or global prune path in `cmd/testing_main_test.go`.
- Post-suite inspection showed only the main worktree and none of the registered fixture branches.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Registered all source-checkout worktree fixtures**

- **Found during:** Task 2 overall verification
- **Issue:** Several existing worktree tests can create real branches and worktrees in the source checkout. The former broad cleanup had hidden their missing ownership declarations; after cleanup became exact, those fixtures could leak generated artifacts.
- **Fix:** Added one source-allocation registration helper and invoked it before every test path that can create a source-checkout worktree.
- **Files modified:** `cmd/worktree_test.go`
- **Verification:** The focused source-allocation tests passed together, the full normal and race suites passed with the known unrelated exclusion, and post-suite Git inventory contained no fixture worktrees or branches.
- **Committed in:** `83f2b9c4`

---

**Total deviations:** 1 auto-fixed (1 missing critical functionality)
**Impact on plan:** The fix completes the positive-ownership contract for all current source-checkout fixtures without widening production behavior or cleanup authority.

## Issues Encountered

- The unfiltered full suite has one pre-existing failure: `TestBranchDispositionRecordsAllThreeBranches` still expects a Phase 196 artifact under `.planning/phases/`, but that artifact now lives in `.planning/milestones/v1.27-phases/`. The test fails identically in isolation and does not exercise Plan 199-01 code. It is recorded in `deferred-items.md`; normal and race suites otherwise pass when that exact unrelated test is excluded.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Exact-ownership teardown makes the broader Phase 199 proof suite safe to add.
- Plan 199-02 can proceed; the archived-path maintenance item is unrelated and non-blocking.

## Self-Check: PASSED

- All three planned code/test files exist.
- All four task and deviation commits exist in Git history.
- Focused, full, and race verification results are recorded above.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-03*
